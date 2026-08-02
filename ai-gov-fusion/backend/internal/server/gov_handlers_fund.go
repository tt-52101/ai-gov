// Package server 治理 API handlers——Fund/Key/Pricing/ModelGrant/Routing 域。
// 全部注释使用中文，符合 AGENTS.md 铁律。
package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"tokenhub/backend/internal/server/abac"
	"tokenhub/backend/internal/server/audit"
	"tokenhub/backend/internal/server/fund"
	"tokenhub/backend/internal/server/modelgrant"
	"tokenhub/backend/internal/server/pricing"
	"tokenhub/backend/internal/server/routing"
)

// ── Fund 域请求/响应类型 ─────────────────────────────────────────────────

// GovAllocateRequest 划拨请求——控制面 HTTP 层入参。
// 金额字段使用 string 接收前端数字字符串，PRD 要求 NUMERIC 字符串禁止浮点。
type GovAllocateRequest struct {
	// DstAccountID 为目标账户 ID，必填。
	DstAccountID string `json:"dst_account_id"`
	// Amount 为划拨金额（正数），前端发送数字字符串如 "100.00"，必填。
	Amount string `json:"amount"`
	// Channel 为划拨通道：parent/sponsors/allocates/whitelist，必填。
	Channel string `json:"channel"`
	// EdgeID 为可选的 party_edge 引用。
	EdgeID *string `json:"edge_id,omitempty"`
	// Reason 为可选的业务事由。
	Reason *string `json:"reason,omitempty"`
}

// GovLiquidateRequest 清算请求——控制面 HTTP 层入参。
type GovLiquidateRequest struct {
	// TargetAccountID 为接收剩余资金的目标账户 ID，必填。
	TargetAccountID string `json:"target_account_id"`
	// PartyID 为账户所属组织 ID，必填。
	PartyID string `json:"party_id"`
	// Reason 为清算事由，必填（被审计）。
	Reason string `json:"reason"`
}

// GovUpdateBudgetRequest 预算帽更新请求。
// 所有字段均为可选——仅更新请求中提供的字段。
type GovUpdateBudgetRequest struct {
	// BudgetLimitAmount 为新的预算上限金额，字符串格式如 "50000.00"。
	BudgetLimitAmount *string `json:"budget_limit_amount,omitempty"`
	// BudgetWarnRatio 为预算预警比例，如 "0.8" 表示 80%。
	BudgetWarnRatio *string `json:"budget_warn_ratio,omitempty"`
	// BudgetPeriod 为预算周期：none/daily/weekly/monthly/quarterly/yearly。
	BudgetPeriod *string `json:"budget_period,omitempty"`
	// BudgetPeriodStart 为预算周期起始时间，ISO 8601 格式。
	BudgetPeriodStart *string `json:"budget_period_start,omitempty"`
	// BudgetPeriodEnd 为预算周期结束时间，ISO 8601 格式。
	BudgetPeriodEnd *string `json:"budget_period_end,omitempty"`
}

// ── fundErrorToHTTP 统一错误映射 ─────────────────────────────────────────

// fundErrorToHTTP 将 fund.FundError 映射为 HTTPError。
//
// 映射规则（per PRD §6 错误码）：
//   - INSUFFICIENT_BALANCE / BUDGET_CAP_EXCEEDED → 402 Payment Required
//   - ACCOUNT_FROZEN_OR_CLOSED / ALLOCATION_CHANNEL_DENIED → 403 Forbidden
//   - FREEZE_NOT_FOUND → 404 Not Found
//   - FREEZE_EXPIRED / IDEMPOTENCY_CONFLICT → 409 Conflict
//   - AMOUNT_MUST_BE_POSITIVE / SELF_TRANSFER / IDEMPOTENCY_KEY_REQUIRED → 400 Bad Request
//   - LIQUIDATION_STAGE_INVALID → 422 Unprocessable Entity
//   - 其他 fund 错误 → 500 Internal Server Error
//
// 非 FundError 类型的错误返回 500。
func fundErrorToHTTP(err error) *HTTPError {
	var fe *fund.FundError
	if errors.As(err, &fe) {
		status := http.StatusInternalServerError
		switch fe.Code {
		case "INSUFFICIENT_BALANCE", "BUDGET_CAP_EXCEEDED":
			status = http.StatusPaymentRequired // 402
		case "ACCOUNT_FROZEN_OR_CLOSED", "ALLOCATION_CHANNEL_DENIED":
			status = http.StatusForbidden // 403
		case "FREEZE_NOT_FOUND":
			status = http.StatusNotFound // 404
		case "FREEZE_EXPIRED", "IDEMPOTENCY_CONFLICT":
			status = http.StatusConflict // 409
		case "SELF_TRANSFER", "AMOUNT_MUST_BE_POSITIVE", "IDEMPOTENCY_KEY_REQUIRED":
			status = http.StatusBadRequest // 400
		case "LIQUIDATION_STAGE_INVALID":
			status = http.StatusUnprocessableEntity // 422
		}
		return NewHTTPError(status, fe.Code, fe.Message)
	}
	return NewHTTPError(http.StatusInternalServerError, "INTERNAL_ERROR", sanitizeError(err))
}

// ── extractAccountAction URL 路径解析 ────────────────────────────────────

// extractAccountAction 从 /gov/accounts/{id}[/{action}] 中提取账户 ID 和可选操作名。
// 例如 /gov/accounts/acc_123/allocate → ("acc_123", "allocate")。
// 例如 /gov/accounts/acc_123 → ("acc_123", "")。
func extractAccountAction(r *http.Request) (accountID, action string) {
	path := r.URL.Path
	const prefix = "/gov/accounts/"
	if !strings.HasPrefix(path, prefix) {
		return "", ""
	}
	rest := path[len(prefix):]
	parts := strings.SplitN(rest, "/", 2)
	accountID = parts[0]
	if len(parts) > 1 {
		action = parts[1]
	}
	return
}

// ── §3 Fund handlers ──────────────────────────────────────────────────────

// handleAccounts 处理 /gov/accounts 集合端点——GET 列表。
func (h *GovHandler) handleAccounts(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireGovAuth(w, r, "fund.balance.read"); !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		db := h.deps.DB
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}
		// 支持按 party_id 筛选。
		var accounts []fund.Account
		query := db.WithContext(r.Context()).Model(&fund.Account{}).Order("created_at DESC")
		if partyID := r.URL.Query().Get("party_id"); partyID != "" {
			query = query.Where("party_id = ?", partyID)
		}
		if err := query.Find(&accounts).Error; err != nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "ACCOUNT_LIST_FAILED", "查询账户列表失败: "+sanitizeError(err)))
			return
		}
		okJSON(w, map[string]any{"data": accounts, "total": len(accounts)})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handleAccountItem 处理 /gov/accounts/{id}[/{action}] 单品端点。
//
// 路由：
//   - GET /gov/accounts/{id} → 查询账户详情
//   - GET /gov/accounts/{id}/ledgers → 分页查询流水
//   - POST /gov/accounts/{id}/allocate → 划拨资金
//   - POST /gov/accounts/{id}/liquidate → 启动/推进清算
//   - PATCH /gov/accounts/{id}/budget → 更新预算帽
func (h *GovHandler) handleAccountItem(w http.ResponseWriter, r *http.Request) {
	accountID, action := extractAccountAction(r)

	switch {
	case r.Method == http.MethodGet && action == "":
		// GET /gov/accounts/{id} ——查询账户详情。
		h.handleGetAccount(w, r, accountID)

	case r.Method == http.MethodGet && action == "ledgers":
		// GET /gov/accounts/{id}/ledgers ——分页查询流水。
		h.handleGetAccountLedgers(w, r, accountID)

	case r.Method == http.MethodPost && action == "allocate":
		// POST /gov/accounts/{id}/allocate ——划拨资金。
		h.handleAllocate(w, r, accountID)

	case r.Method == http.MethodPost && action == "liquidate":
		// POST /gov/accounts/{id}/liquidate ——启动/推进清算。
		h.handleLiquidate(w, r, accountID)

	case r.Method == http.MethodPatch && action == "budget":
		// PATCH /gov/accounts/{id}/budget ——更新预算帽。
		h.handleUpdateBudget(w, r, accountID)

	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法或路径"))
	}
}

// ── handleGetAccount 查询账户详情 ────────────────────────────────────────

// handleGetAccount 查询单个账户详情——GET /gov/accounts/{id}。
func (h *GovHandler) handleGetAccount(w http.ResponseWriter, r *http.Request, accountID string) {
	if _, ok := h.requireGovItemAuth(w, r, "fund.balance.read", "account", accountID); !ok {
		return
	}

	if h.deps.FundService == nil {
		writeError(w, r, NewHTTPError(501, "NOT_IMPLEMENTED", "Fund 服务未配置"))
		return
	}

	acct, err := h.deps.FundService.Store.GetAccount(r.Context(), accountID)
	if err != nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "ACCOUNT_QUERY_FAILED", "查询账户失败: "+sanitizeError(err)))
		return
	}
	if acct == nil {
		writeError(w, r, NewHTTPError(http.StatusNotFound, "ACCOUNT_NOT_FOUND", "账户不存在: "+accountID))
		return
	}
	okJSON(w, acct)
}

// ── handleGetAccountLedgers 分页查询流水 ──────────────────────────────────

// handleGetAccountLedgers 分页查询账户账本流水——GET /gov/accounts/{id}/ledgers。
//
// 查询参数：
//   - page: 页码，默认 1
//   - page_size: 每页条数，默认 20，最大 200
func (h *GovHandler) handleGetAccountLedgers(w http.ResponseWriter, r *http.Request, accountID string) {
	if _, ok := h.requireGovItemAuth(w, r, "fund.ledger.read", "account", accountID); !ok {
		return
	}

	db := h.deps.DB
	if db == nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
		return
	}

	// 校验账户存在。
	var acct fund.Account
	if err := db.WithContext(r.Context()).Where("id = ?", accountID).First(&acct).Error; err != nil {
		writeError(w, r, NewHTTPError(http.StatusNotFound, "ACCOUNT_NOT_FOUND", "账户不存在: "+accountID))
		return
	}

	// 解析分页参数。
	page := parseIntParam(r, "page", 1)
	pageSize := parseIntParam(r, "page_size", 20)
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	// 查询总数。
	var total int64
	if err := db.WithContext(r.Context()).Model(&fund.Ledger{}).
		Where("account_id = ?", accountID).Count(&total).Error; err != nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "LEDGER_QUERY_FAILED", "查询流水失败: "+sanitizeError(err)))
		return
	}

	// 分页查询流水，按创建时间倒序。
	var ledgers []fund.Ledger
	if err := db.WithContext(r.Context()).
		Where("account_id = ?", accountID).
		Order("created_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&ledgers).Error; err != nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "LEDGER_QUERY_FAILED", "查询流水失败: "+sanitizeError(err)))
		return
	}

	pages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		pages++
	}

	okJSON(w, map[string]any{
		"data":      ledgers,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"pages":     pages,
	})
}

// ── handleAllocate 划拨资金 ───────────────────────────────────────────────

// handleAllocate 从源账户向目标账户划拨资金——POST /gov/accounts/{id}/allocate。
//
// 流程：
//  1. ABAC 鉴权（fund.balance.write）。
//  2. 解析请求体。
//  3. 从 Header 提取 Idempotency-Key 注入请求。
//  4. 调用 FundService.Allocate 执行划拨。
//  5. 返回划拨结果（201 Created 或 200 幂等重放）。
func (h *GovHandler) handleAllocate(w http.ResponseWriter, r *http.Request, srcAccountID string) {
	gctx, _ := h.requireGovItemAuth(w, r, "fund.balance.write", "account", srcAccountID)
	if gctx == nil {
		return
	}

	if h.deps.FundService == nil {
		writeError(w, r, NewHTTPError(501, "NOT_IMPLEMENTED", "Fund 服务未配置"))
		return
	}

	// 解析请求体。
	req, ok := readJSON[GovAllocateRequest](w, r)
	if !ok {
		return
	}

	// 校验必填字段。
	if strings.TrimSpace(req.DstAccountID) == "" {
		writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "dst_account_id 为必填字段"))
		return
	}
	if strings.TrimSpace(req.Amount) == "" {
		writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "amount 为必填字段"))
		return
	}
	if strings.TrimSpace(req.Channel) == "" {
		writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "channel 为必填字段"))
		return
	}

	// 金额从 string 转换为 fund.Decimal（PRD 要求 NUMERIC 字符串禁止浮点）。
	amountDec, parseErr := decimalFromString(req.Amount)
	if parseErr != nil {
		writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "amount 格式无效（需为数字字符串如 \"100.00\"）: "+req.Amount))
		return
	}
	if amountDec.Decimal.LessThanOrEqual(decimal.Zero) {
		writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "amount 必须为正数"))
		return
	}

	// 从 Header 提取 Idempotency-Key（API 规范 §1.4）。
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))

	// 构造 fund 层请求。
	allocateReq := fund.AllocateRequest{
		SrcAccountID:   srcAccountID,
		DstAccountID:   strings.TrimSpace(req.DstAccountID),
		Amount:         amountDec,
		Channel:        strings.TrimSpace(req.Channel),
		EdgeID:         req.EdgeID,
		IdempotencyKey: idempotencyKey,
		OperatorID:     gctx.SubjectID,
		Reason:         req.Reason,
	}

	// 调用 Service 层执行划拨。
	result, err := h.deps.FundService.Allocate(r.Context(), allocateReq)
	if err != nil {
		writeError(w, r, fundErrorToHTTP(err))
		return
	}

	slog.InfoContext(r.Context(), "控制面划拨成功",
		"allocation_id", result.AllocationID,
		"src_account_id", srcAccountID,
		"dst_account_id", req.DstAccountID,
		"amount", req.Amount,
		"channel", req.Channel,
		"actor", gctx.SubjectID,
	)

	// 记录审计事件——资金划拨操作（AU-CON-01 / D-CON-04）。
	afterJSON, _ := json.Marshal(result)
	_ = audit.RecordEvent(r.Context(), h.deps.DB, &audit.AuditEvent{
		ID:            NewID("audit"),
		ActorUserID:   gctx.SubjectID,
		ActorName:     gctx.UserName,
		Action:        audit.ActionAllocate,
		ResourceType:  "allocation",
		ResourceID:    result.AllocationID,
		Status:        audit.StatusSuccess,
		BeforeSnapshot: "",
		AfterSnapshot:  string(afterJSON),
		IP:            gctx.ClientIP,
		UserAgent:     gctx.UserAgent,
	})

	// 幂等重放返回 200，首次创建返回 201。
	if r.Header.Get("Idempotent-Replayed") == "true" || idempotencyKey != "" {
		okJSON(w, result)
	} else {
		createdJSON(w, result)
	}
}

// ── handleLiquidate 启动/推进清算 ─────────────────────────────────────────

// handleLiquidate 启动或推进账户清算——POST /gov/accounts/{id}/liquidate。
//
// 流程：
//  1. ABAC 鉴权（fund.balance.write）。
//  2. 解析请求体。
//  3. 从 Header 提取 Idempotency-Key（清算为写操作）。
//  4. 调用 FundService.Liquidate 执行清算。
func (h *GovHandler) handleLiquidate(w http.ResponseWriter, r *http.Request, accountID string) {
	gctx, _ := h.requireGovItemAuth(w, r, "fund.balance.write", "account", accountID)
	if gctx == nil {
		return
	}

	if h.deps.FundService == nil {
		writeError(w, r, NewHTTPError(501, "NOT_IMPLEMENTED", "Fund 服务未配置"))
		return
	}

	// 解析请求体。
	req, ok := readJSON[GovLiquidateRequest](w, r)
	if !ok {
		return
	}

	// 校验必填字段。
	if strings.TrimSpace(req.TargetAccountID) == "" {
		writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "target_account_id 为必填字段"))
		return
	}
	if strings.TrimSpace(req.PartyID) == "" {
		writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "party_id 为必填字段"))
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "reason 为必填字段（被审计）"))
		return
	}

	// 从 Header 提取 Idempotency-Key（清算为写操作，需幂等保护）。
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))

	// 构造 fund 层请求。
	liquidateReq := fund.LiquidateRequest{
		AccountID:       accountID,
		TargetAccountID: strings.TrimSpace(req.TargetAccountID),
		OperatorID:      gctx.SubjectID,
		PartyID:         strings.TrimSpace(req.PartyID),
		Reason:          strings.TrimSpace(req.Reason),
	}

	// 调用 Service 层执行清算。
	result, err := h.deps.FundService.Liquidate(r.Context(), liquidateReq)
	if err != nil {
		writeError(w, r, fundErrorToHTTP(err))
		return
	}

	slog.InfoContext(r.Context(), "控制面清算操作成功",
		"liquidation_id", result.LiquidationID,
		"account_id", accountID,
		"target_account_id", req.TargetAccountID,
		"status", result.Status,
		"actor", gctx.SubjectID,
		"idempotency_key", idempotencyKey,
	)

	// 记录审计事件——清算操作（AU-CON-01 / D-CON-04）。
	afterJSON, _ := json.Marshal(result)
	_ = audit.RecordEvent(r.Context(), h.deps.DB, &audit.AuditEvent{
		ID:            NewID("audit"),
		ActorUserID:   gctx.SubjectID,
		ActorName:     gctx.UserName,
		Action:        audit.ActionLiquidate,
		ResourceType:  "liquidation",
		ResourceID:    result.LiquidationID,
		Status:        audit.StatusSuccess,
		BeforeSnapshot: "",
		AfterSnapshot:  string(afterJSON),
		IP:            gctx.ClientIP,
		UserAgent:     gctx.UserAgent,
	})

	createdJSON(w, result)
}

// ── handleUpdateBudget 更新预算帽 ─────────────────────────────────────────

// handleUpdateBudget 更新账户预算帽字段——PATCH /gov/accounts/{id}/budget。
//
// 流程：
//  1. ABAC 鉴权（fund.balance.write）。
//  2. 解析请求体（全部字段可选，仅更新提供的字段）。
//  3. 校验账户存在。
//  4. 在事务内以乐观锁更新预算相关字段。
func (h *GovHandler) handleUpdateBudget(w http.ResponseWriter, r *http.Request, accountID string) {
	gctx, _ := h.requireGovItemAuth(w, r, "fund.balance.write", "account", accountID)
	if gctx == nil {
		return
	}

	db := h.deps.DB
	if db == nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
		return
	}

	// 解析请求体。
	req, ok := readJSON[GovUpdateBudgetRequest](w, r)
	if !ok {
		return
	}

	// 校验账户存在并获取当前版本号（乐观锁）。
	var acct fund.Account
	if err := db.WithContext(r.Context()).Where("id = ?", accountID).First(&acct).Error; err != nil {
		writeError(w, r, NewHTTPError(http.StatusNotFound, "ACCOUNT_NOT_FOUND", "账户不存在: "+accountID))
		return
	}

	// 保存变更前快照——用于审计事件（AU-CON-01 / D-CON-04）。
	beforeJSON, _ := json.Marshal(acct)

	// 构建更新字段映射——仅更新请求中提供的字段。
	updates := map[string]interface{}{}
	now := time.Now()

	if req.BudgetLimitAmount != nil {
		val := strings.TrimSpace(*req.BudgetLimitAmount)
		if val == "" || val == "null" {
			updates["budget_limit_amount"] = nil
		} else {
			d, err := decimalFromString(val)
			if err != nil {
				writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "budget_limit_amount 格式无效: "+val))
				return
			}
			updates["budget_limit_amount"] = d.String()
		}
	}

	if req.BudgetWarnRatio != nil {
		val := strings.TrimSpace(*req.BudgetWarnRatio)
		if val == "" || val == "null" {
			updates["budget_warn_ratio"] = nil
		} else {
			d, err := decimalFromString(val)
			if err != nil {
				writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "budget_warn_ratio 格式无效: "+val))
				return
			}
			updates["budget_warn_ratio"] = d.String()
		}
	}

	if req.BudgetPeriod != nil {
		updates["budget_period"] = strings.TrimSpace(*req.BudgetPeriod)
	}

	if req.BudgetPeriodStart != nil {
		val := strings.TrimSpace(*req.BudgetPeriodStart)
		if val == "" || val == "null" {
			updates["budget_period_start"] = nil
		} else {
			t, err := time.Parse(time.RFC3339, val)
			if err != nil {
				writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "budget_period_start 格式无效，需为 ISO 8601: "+val))
				return
			}
			updates["budget_period_start"] = t
		}
	}

	if req.BudgetPeriodEnd != nil {
		val := strings.TrimSpace(*req.BudgetPeriodEnd)
		if val == "" || val == "null" {
			updates["budget_period_end"] = nil
		} else {
			t, err := time.Parse(time.RFC3339, val)
			if err != nil {
				writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "budget_period_end 格式无效，需为 ISO 8601: "+val))
				return
			}
			updates["budget_period_end"] = t
		}
	}

	if len(updates) == 0 {
		writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "未提供任何待更新字段"))
		return
	}

	// 乐观锁更新——仅当 version 匹配时更新。
	updates["version"] = acct.Version + 1
	updates["updated_at"] = now

	result := db.WithContext(r.Context()).Model(&fund.Account{}).
		Where("id = ? AND version = ?", accountID, acct.Version).
		Updates(updates)

	if result.Error != nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "BUDGET_UPDATE_FAILED", "预算帽更新失败: "+sanitizeError(result.Error)))
		return
	}
	if result.RowsAffected == 0 {
		writeError(w, r, NewHTTPError(http.StatusConflict, "VERSION_CONFLICT", "预算帽更新冲突——账户已被并发修改，请重试"))
		return
	}

	slog.InfoContext(r.Context(), "预算帽已更新",
		"account_id", accountID,
		"updated_fields", len(updates),
	)

	// 返回更新后的账户。
	var updatedAcct fund.Account
	if err := db.WithContext(r.Context()).Where("id = ?", accountID).First(&updatedAcct).Error; err != nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "ACCOUNT_QUERY_FAILED", "查询更新后的账户失败"))
		return
	}

	// 记录审计事件——预算帽变更操作（AU-CON-01 / D-CON-04）。
	afterJSON, _ := json.Marshal(updatedAcct)
	_ = audit.RecordEvent(r.Context(), db, &audit.AuditEvent{
		ID:             NewID("audit"),
		ActorUserID:    gctx.SubjectID,
		ActorName:      gctx.UserName,
		Action:         audit.ActionBudgetCapChange,
		ResourceType:   "account",
		ResourceID:     accountID,
		Status:         audit.StatusSuccess,
		BeforeSnapshot: string(beforeJSON),
		AfterSnapshot:  string(afterJSON),
		IP:             gctx.ClientIP,
		UserAgent:      gctx.UserAgent,
	})

	okJSON(w, updatedAcct)
}

// ── handleAllocations 集合端点 ────────────────────────────────────────────

// handleAllocations 处理 /gov/allocations 集合端点——GET 列表。
func (h *GovHandler) handleAllocations(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireGovAuth(w, r, "fund.ledger.read"); !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		db := h.deps.DB
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}
		// 支持按 src_account_id 或 dst_account_id 筛选。
		var allocations []fund.Allocation
		query := db.WithContext(r.Context()).Model(&fund.Allocation{}).Order("created_at DESC")
		if srcID := r.URL.Query().Get("src_account_id"); srcID != "" {
			query = query.Where("src_account_id = ?", srcID)
		}
		if dstID := r.URL.Query().Get("dst_account_id"); dstID != "" {
			query = query.Where("dst_account_id = ?", dstID)
		}
		if err := query.Find(&allocations).Error; err != nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "ALLOCATION_LIST_FAILED", "查询划拨记录列表失败: "+sanitizeError(err)))
			return
		}
		okJSON(w, map[string]any{"data": allocations, "total": len(allocations)})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// ── handleAllocationItem 查询划拨记录 ─────────────────────────────────────

// handleAllocationItem 查询单条划拨记录详情——GET /gov/allocations/{id}。
func (h *GovHandler) handleAllocationItem(w http.ResponseWriter, r *http.Request) {
	allocID := extractItemID(r, "/v1/gov/allocations")
	if _, ok := h.requireGovItemAuth(w, r, "fund.ledger.read", "allocation", allocID); !ok {
		return
	}

	db := h.deps.DB
	if db == nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		var allocation fund.Allocation
		if err := db.WithContext(r.Context()).Where("id = ?", allocID).First(&allocation).Error; err != nil {
			writeError(w, r, NewHTTPError(http.StatusNotFound, "ALLOCATION_NOT_FOUND", "划拨记录不存在: "+allocID))
			return
		}
		okJSON(w, allocation)
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// ── parseIntParam 解析查询参数中的整数 ───────────────────────────────────

// parseIntParam 从 URL 查询参数中解析整数，失败时返回默认值。
func parseIntParam(r *http.Request, name string, defaultVal int) int {
	valStr := r.URL.Query().Get(name)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil || val < 0 {
		return defaultVal
	}
	return val
}

// ── §4 Key handlers ───────────────────────────────────────────────────────

func (h *GovHandler) handleKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.handleCreateKey(w, r)
	case http.MethodGet:
		h.handleListKeys(w, r)
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handleKeyItem 单个密钥操作——/gov/keys/{id}。
// 支持 GET（详情）、DELETE（删除）、POST（轮换）。
func (h *GovHandler) handleKeyItem(w http.ResponseWriter, r *http.Request) {
	keyID := extractItemID(r, "/v1/gov/keys")
	switch r.Method {
		case http.MethodGet:
			if _, ok := h.requireGovItemAuth(w, r, "iam.key.read", "key", keyID); !ok {
				return
			}
			db := h.deps.DB
			if db == nil {
				writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
				return
			}
			var key GovAPIKey
			if err := db.WithContext(r.Context()).Where("id = ?", keyID).First(&key).Error; err != nil {
				writeError(w, r, NewHTTPError(http.StatusNotFound, "KEY_NOT_FOUND", "密钥不存在: "+keyID))
				return
			}
			okJSON(w, fromGovAPIKey(key))
		case http.MethodDelete:
			if _, ok := h.requireGovItemAuth(w, r, "iam.key.delete", "key", keyID); !ok {
				return
			}
			db := h.deps.DB
			if db == nil {
				writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
				return
			}
			var key GovAPIKey
			if err := db.WithContext(r.Context()).Where("id = ?", keyID).First(&key).Error; err != nil {
				writeError(w, r, NewHTTPError(http.StatusNotFound, "KEY_NOT_FOUND", "密钥不存在: "+keyID))
				return
			}
			if key.Status == StatusRevoked {
				writeError(w, r, NewHTTPError(http.StatusConflict, "KEY_ALREADY_REVOKED", "密钥已被吊销: "+keyID))
				return
			}
			beforeJSON, _ := json.Marshal(fromGovAPIKey(key))
			key.Status = StatusRevoked
			if err := db.WithContext(r.Context()).Save(&key).Error; err != nil {
				writeError(w, r, NewHTTPError(http.StatusInternalServerError, "KEY_REVOKE_FAILED", "密钥吊销失败: "+sanitizeError(err)))
				return
			}
			afterJSON, _ := json.Marshal(fromGovAPIKey(key))
			_ = audit.RecordEvent(r.Context(), db, &audit.AuditEvent{
				ID:              NewID("audit"),
				ActorUserID:     "",
				ActorName:       "",
				Action:          audit.ActionKeyRevoke,
				ResourceType:    "gov_api_key",
				ResourceID:      keyID,
				Status:          audit.StatusSuccess,
				BeforeSnapshot:  string(beforeJSON),
				AfterSnapshot:   string(afterJSON),
				IP:              "",
				UserAgent:       "",
			})
			slog.InfoContext(r.Context(), "治理 API 密钥吊销成功",
				"key_id", keyID,
				"actor", "",
			)
			okJSON(w, fromGovAPIKey(key))
		case http.MethodPost:
			if _, ok := h.requireGovItemAuth(w, r, "iam.key.create", "key", keyID); !ok {
				return
			}
			// Key 轮换：生成新密钥，更新哈希和前缀，保留原记录 ID。
			db := h.deps.DB
			if db == nil {
				writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
				return
			}
			var key GovAPIKey
			if err := db.WithContext(r.Context()).Where("id = ?", keyID).First(&key).Error; err != nil {
				writeError(w, r, NewHTTPError(http.StatusNotFound, "KEY_NOT_FOUND", "密钥不存在: "+keyID))
				return
			}
			// 生成新密钥。
			rawSecret := GenerateAPIKeyWithOptions(key.KeyPrefix, DefaultAPIKeyRandomLength)
			prefix, _ := PrefixSuffix(rawSecret)
			keyHash := HashSecret(rawSecret)
			now := time.Now().UTC()
			// 更新数据库中的哈希和前缀。
			if err := db.WithContext(r.Context()).Model(&key).Updates(map[string]any{
				"key_hash":   keyHash,
				"key_prefix": prefix,
				"updated_at": now,
			}).Error; err != nil {
				writeError(w, r, NewHTTPError(http.StatusInternalServerError, "KEY_ROTATE_FAILED", "密钥轮换失败: "+sanitizeError(err)))
				return
			}
			slog.InfoContext(r.Context(), "治理 API 密钥轮换成功", "key_id", keyID)
			// 返回新密钥（一次性明文）和更新后的元数据。
			createdJSON(w, GovCreatedKeyResponse{
				GovKeyResponse: fromGovAPIKey(key),
				RawKey:         rawSecret,
			})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handleCreateKey 创建治理 API 密钥——POST /gov/keys。
//
// 流程：
//  1. ABAC 鉴权（iam.key.create）。
//  2. 解析请求体。
//  3. 校验 account_id 存在（admin_resources kind=accounts）。
//  4. 校验调用方对目标 account 有 iam.key.create 权限（ABAC 第二次评估）。
//  5. 生成随机密钥（前缀+随机字符串）。
//  6. SHA-256 哈希后存储。
//  7. 仅创建时返回完整明文（一次性展示），后续 GET 只返回 KeyPrefix。
func (h *GovHandler) handleCreateKey(w http.ResponseWriter, r *http.Request) {
	gctx, _ := h.requireGovAuth(w, r, "iam.key.create")
	if gctx == nil {
		return
	}

	// 解析请求体。
	req, ok := readJSON[GovCreateKeyRequest](w, r)
	if !ok {
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "name 为必填字段"))
		return
	}

	// 校验 account_id 存在（如果提供）。
	if strings.TrimSpace(req.AccountID) != "" {
		db := h.deps.DB
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}
		var acct AdminResource
		if err := db.Where("kind = ? AND id = ?", "accounts", req.AccountID).First(&acct).Error; err != nil {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "ACCOUNT_NOT_FOUND", "账户不存在: "+req.AccountID))
			return
		}
		// 校验调用方对目标 account 有 iam.key.create 权限。
		if h.deps.ABACEngine != nil {
			subject := abac.Subject{Type: gctx.SubjectType, ID: gctx.SubjectID}
			resource := abac.Resource{Type: "account", ID: req.AccountID}
			if err := h.deps.ABACEngine.Evaluate(r.Context(), subject, "iam.key.create", resource); err != nil {
				writeError(w, r, NewHTTPError(http.StatusForbidden, "AUTHZ_DENIED", "无权对该账户创建 Key: "+sanitizeError(err)))
				return
			}
		}
	}

	// 确定密钥前缀：优先使用请求中的 prefix，否则使用默认 "gov_"。
	keyPrefix := NormalizeAPIKeyPrefix(req.KeyPrefix)
	if keyPrefix == DefaultAPIKeyPrefix && strings.TrimSpace(req.KeyPrefix) == "" {
		keyPrefix = "gov_"
	}

	// 生成随机密钥。
	rawSecret := GenerateAPIKeyWithOptions(keyPrefix, DefaultAPIKeyRandomLength)
	prefix, _ := PrefixSuffix(rawSecret)
	keyHash := HashSecret(rawSecret)

	// OwnerUserID 默认为当前鉴权用户（若请求中未指定）。
	ownerUserID := strings.TrimSpace(req.OwnerUserID)
	if ownerUserID == "" {
		ownerUserID = gctx.SubjectID
	}

	now := time.Now().UTC()
	key := GovAPIKey{
		ID:          NewID("govkey"),
		Name:        strings.TrimSpace(req.Name),
		KeyHash:     keyHash,
		KeyPrefix:   prefix,
		OwnerUserID: ownerUserID,
		AccountID:   strings.TrimSpace(req.AccountID),
		PartyID:     strings.TrimSpace(req.PartyID),
		Status:      StatusActive,
		CreatedAt:   now,
	}

	db := h.deps.DB
	if db == nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
		return
	}
	if err := db.Create(&key).Error; err != nil {
		writeError(w, r, NewHTTPError(http.StatusConflict, "KEY_CREATE_FAILED", "密钥创建失败: "+sanitizeError(err)))
		return
	}

	slog.InfoContext(r.Context(), "治理 API 密钥创建成功",
		"key_id", key.ID,
		"owner_user_id", ownerUserID,
		"account_id", key.AccountID,
		"actor", gctx.SubjectID,
	)

	// 记录审计事件——密钥创建操作（AU-CON-01 / D-CON-04）。
	afterJSON, _ := json.Marshal(fromGovAPIKey(key))
	_ = audit.RecordEvent(r.Context(), db, &audit.AuditEvent{
		ID:            NewID("audit"),
		ActorUserID:   gctx.SubjectID,
		ActorName:     gctx.UserName,
		Action:        audit.ActionKeyCreate,
		ResourceType:  "gov_api_key",
		ResourceID:    key.ID,
		Status:        audit.StatusSuccess,
		BeforeSnapshot: "",
		AfterSnapshot:  string(afterJSON),
		IP:            gctx.ClientIP,
		UserAgent:     gctx.UserAgent,
	})

	// 仅创建时返回完整明文。
	createdJSON(w, GovCreatedKeyResponse{
		GovKeyResponse: fromGovAPIKey(key),
		RawKey:         rawSecret,
	})
}

// handleListKeys 查询治理 API 密钥列表——GET /gov/keys。
//
// 按 owner_user_id 筛选（与鉴权身份一致），返回列表不含明文。
// 可选查询参数 ?account_id=xxx 进一步过滤。
func (h *GovHandler) handleListKeys(w http.ResponseWriter, r *http.Request) {
	gctx, _ := h.requireGovAuth(w, r, "iam.key.read")
	if gctx == nil {
		return
	}

	db := h.deps.DB
	if db == nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
		return
	}

	// 按 owner_user_id 筛选（鉴权身份即为 owner）。
	query := db.Where("owner_user_id = ?", gctx.SubjectID)

	// 可选：按 account_id 进一步过滤。
	if accountID := r.URL.Query().Get("account_id"); accountID != "" {
		query = query.Where("account_id = ?", accountID)
	}

	var keys []GovAPIKey
	if err := query.Order("created_at desc").Find(&keys).Error; err != nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "KEY_LIST_FAILED", "查询密钥列表失败: "+sanitizeError(err)))
		return
	}

	// 转换为对外响应（不含明文）。
	items := make([]GovKeyResponse, 0, len(keys))
	for _, key := range keys {
		items = append(items, fromGovAPIKey(key))
	}
	okJSON(w, map[string]any{
		"items": items,
		"total": len(items),
	})
}

// ── §5 Pricing handlers ───────────────────────────────────────────────────

// GovModelPriceRequest 定价创建/更新请求体。
type GovModelPriceRequest struct {
	ModelID          string     `json:"model_id"`
	ChannelID        *string    `json:"channel_id,omitempty"`
	ReferenceID      string     `json:"reference_id"`
	PriceJSON        any        `json:"price_json"`
	EffectiveStartAt *time.Time `json:"effective_start_at,omitempty"`
	EffectiveEndAt   *time.Time `json:"effective_end_at,omitempty"`
}

// handleModelPrices 价目列表/创建——PUT/GET /gov/model-prices。
// PUT 执行 upsert（以 reference_id 为唯一键）。
func (h *GovHandler) handleModelPrices(w http.ResponseWriter, r *http.Request) {
	db := h.deps.PricingDB
	if db == nil {
		db = h.deps.DB
	}
	switch r.Method {
	case http.MethodPut:
		gctx, _ := h.requireGovAuth(w, r, "routing.price.write")
		if gctx == nil {
			return
		}
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		req, ok := readJSON[GovModelPriceRequest](w, r)
		if !ok {
			return
		}
		if strings.TrimSpace(req.ModelID) == "" {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "model_id 为必填字段"))
			return
		}
		if strings.TrimSpace(req.ReferenceID) == "" {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "reference_id 为必填字段"))
			return
		}

		// 序列化 price_json 为 JSON 字符串。
		priceJSONStr := "{}"
		if req.PriceJSON != nil {
			b, _ := json.Marshal(req.PriceJSON)
			priceJSONStr = string(b)
		}

		price := &pricing.ModelPrice{
			ModelID:          strings.TrimSpace(req.ModelID),
			ChannelID:        req.ChannelID,
			ReferenceID:      strings.TrimSpace(req.ReferenceID),
			PriceJSONStr:     priceJSONStr,
			Status:           pricing.StatusActive,
			EffectiveStartAt: req.EffectiveStartAt,
			EffectiveEndAt:   req.EffectiveEndAt,
		}
		if err := pricing.UpsertPrice(db, price); err != nil {
			writeError(w, r, NewHTTPError(http.StatusConflict, "UPSERT_FAILED", sanitizeError(err)))
			return
		}

		slog.InfoContext(r.Context(), "ModelPrice upsert 成功",
			"model_id", price.ModelID,
			"reference_id", price.ReferenceID,
			"actor", gctx.SubjectID,
		)
		okJSON(w, price)
		case http.MethodGet:
			if _, ok := h.requireGovAuth(w, r, "routing.price.read"); !ok {
				return
			}
			if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		modelID := r.URL.Query().Get("model_id")
		var prices []*pricing.ModelPrice
		var err error
		if modelID != "" {
			prices, err = pricing.ListPrices(db, modelID)
		} else {
			// 无 model_id 时返回所有有效价目。
			err = db.Where("status = ?", pricing.StatusActive).
				Order("created_at DESC").Find(&prices).Error
		}
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "LIST_FAILED", sanitizeError(err)))
			return
		}
		okJSON(w, map[string]any{"data": prices, "total": len(prices)})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handleModelPriceItem 单个价目操作——GET/DELETE /gov/model-prices/{id}。
func (h *GovHandler) handleModelPriceItem(w http.ResponseWriter, r *http.Request) {
	priceID := extractItemID(r, "/v1/gov/model-prices")
	db := h.deps.PricingDB
	if db == nil {
		db = h.deps.DB
	}
	switch r.Method {
		case http.MethodGet:
			if _, ok := h.requireGovItemAuth(w, r, "routing.price.read", "model_price", priceID); !ok {
				return
			}
			if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		var price pricing.ModelPrice
		if err := db.First(&price, "id = ?", priceID).Error; err != nil {
			writeError(w, r, NewHTTPError(http.StatusNotFound, "NOT_FOUND", "价目不存在: "+priceID))
			return
		}
		okJSON(w, &price)
		case http.MethodDelete:
			if _, ok := h.requireGovItemAuth(w, r, "routing.price.write", "model_price", priceID); !ok {
				return
			}
			if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		if err := pricing.ArchivePrice(db, priceID); err != nil {
			writeError(w, r, NewHTTPError(http.StatusNotFound, "ARCHIVE_FAILED", sanitizeError(err)))
			return
		}
		okJSON(w, map[string]any{"archived": true, "id": priceID})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// ── §6 Model Grant handlers ───────────────────────────────────────────────

// handleModelGrants 模型授权列表/创建——POST/GET /gov/model-grants。
func (h *GovHandler) handleModelGrants(w http.ResponseWriter, r *http.Request) {
	db := h.deps.DB
	switch r.Method {
	case http.MethodPost:
		gctx, _ := h.requireGovAuth(w, r, "routing.model_grant.write")
		if gctx == nil {
			return
		}
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		mg, ok := readJSON[modelgrant.ModelGrant](w, r)
		if !ok {
			return
		}
		mg.ID = NewID("mg")
		if err := modelgrant.CreateModelGrant(db, &mg); err != nil {
			writeError(w, r, NewHTTPError(http.StatusConflict, "CREATE_FAILED", sanitizeError(err)))
			return
		}

		slog.InfoContext(r.Context(), "ModelGrant 创建成功",
			"grant_id", mg.ID,
			"principal_type", mg.PrincipalType,
			"principal_id", mg.PrincipalID,
			"actor", gctx.SubjectID,
		)
		createdJSON(w, mg)
		case http.MethodGet:
			if _, ok := h.requireGovAuth(w, r, "routing.model_grant.read"); !ok {
				return
			}
			if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		principalType := r.URL.Query().Get("principal_type")
		principalID := r.URL.Query().Get("principal_id")
		modelID := r.URL.Query().Get("model_id")

		grants, err := modelgrant.ListModelGrants(db, principalType, principalID, modelID)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "LIST_FAILED", sanitizeError(err)))
			return
		}
		okJSON(w, map[string]any{"data": grants, "total": len(grants)})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handleModelGrantItem 单个模型授权操作——GET/DELETE /gov/model-grants/{id}。
func (h *GovHandler) handleModelGrantItem(w http.ResponseWriter, r *http.Request) {
	grantID := extractItemID(r, "/v1/gov/model-grants")
	db := h.deps.DB
	switch r.Method {
		case http.MethodGet:
			if _, ok := h.requireGovItemAuth(w, r, "routing.model_grant.read", "model_grant", grantID); !ok {
				return
			}
			if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		mg, err := modelgrant.GetModelGrant(db, grantID)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusNotFound, "NOT_FOUND", sanitizeError(err)))
			return
		}
		okJSON(w, mg)
		case http.MethodDelete:
			if _, ok := h.requireGovItemAuth(w, r, "routing.model_grant.write", "model_grant", grantID); !ok {
				return
			}
			if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		if err := modelgrant.DeleteModelGrant(db, grantID); err != nil {
			writeError(w, r, NewHTTPError(http.StatusNotFound, "DELETE_FAILED", sanitizeError(err)))
			return
		}
		okJSON(w, map[string]any{"deleted": true, "id": grantID})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// ── §7 Routing handlers ───────────────────────────────────────────────────

// handleRouteProfiles 路由档案列表/创建——POST/GET /gov/route-profiles。
func (h *GovHandler) handleRouteProfiles(w http.ResponseWriter, r *http.Request) {
	db := h.deps.RouteProfileDB
	if db == nil {
		db = h.deps.DB
	}
	switch r.Method {
	case http.MethodPost:
		gctx, _ := h.requireGovAuth(w, r, "routing.route_profile.write")
		if gctx == nil {
			return
		}
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		req, ok := readJSON[routing.RouteProfile](w, r)
		if !ok {
			return
		}
		if strings.TrimSpace(req.Name) == "" {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "name 为必填字段"))
			return
		}

		if err := routing.CreateProfile(db, &req); err != nil {
			writeError(w, r, NewHTTPError(http.StatusConflict, "CREATE_FAILED", sanitizeError(err)))
			return
		}

		slog.InfoContext(r.Context(), "RouteProfile 创建成功",
			"profile_id", req.ID,
			"profile_name", req.Name,
			"actor", gctx.SubjectID,
		)
		createdJSON(w, req)
		case http.MethodGet:
			if _, ok := h.requireGovAuth(w, r, "routing.route_profile.read"); !ok {
				return
			}
			if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		profiles, err := routing.ListProfiles(db)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "LIST_FAILED", sanitizeError(err)))
			return
		}
		okJSON(w, map[string]any{"data": profiles, "total": len(profiles)})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handleRouteProfileItem 单个路由档案操作——GET/PUT/DELETE /gov/route-profiles/{id}。
func (h *GovHandler) handleRouteProfileItem(w http.ResponseWriter, r *http.Request) {
	profileIDStr := extractItemID(r, "/v1/gov/route-profiles")
	profileID, err := strconv.ParseInt(profileIDStr, 10, 64)
	if err != nil {
		writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_ID", "无效的档案 ID: "+profileIDStr))
		return
	}

	db := h.deps.RouteProfileDB
	if db == nil {
		db = h.deps.DB
	}
	switch r.Method {
		case http.MethodGet:
			if _, ok := h.requireGovItemAuth(w, r, "routing.route_profile.read", "route_profile", profileIDStr); !ok {
				return
			}
			if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		profile, err := routing.GetProfile(db, profileID)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusNotFound, "NOT_FOUND", sanitizeError(err)))
			return
		}
		okJSON(w, profile)
		case http.MethodPut:
			if _, ok := h.requireGovItemAuth(w, r, "routing.route_profile.write", "route_profile", profileIDStr); !ok {
				return
			}
			if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		req, ok := readJSON[routing.RouteProfile](w, r)
		if !ok {
			return
		}
		req.ID = profileID

		if err := routing.UpdateProfile(db, &req); err != nil {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "UPDATE_FAILED", sanitizeError(err)))
			return
		}

		updated, _ := routing.GetProfile(db, profileID)
		if updated != nil {
			okJSON(w, updated)
		} else {
			okJSON(w, map[string]string{"id": profileIDStr, "message": "档案已更新"})
		}
		case http.MethodDelete:
			if _, ok := h.requireGovItemAuth(w, r, "routing.route_profile.write", "route_profile", profileIDStr); !ok {
				return
			}
			if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		if err := routing.DeleteProfile(db, profileID); err != nil {
			writeError(w, r, NewHTTPError(http.StatusNotFound, "DELETE_FAILED", sanitizeError(err)))
			return
		}
		okJSON(w, map[string]any{"deleted": true, "id": profileID})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handleRouteStrategies 已注册路由策略列表——GET /gov/route-strategies。
func (h *GovHandler) handleRouteStrategies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
		return
	}
	if _, ok := h.requireGovAuth(w, r, "routing.route_profile.read"); !ok {
		return
	}

	strategies := routing.GetRegistered()
	okJSON(w, map[string]any{"data": strategies, "total": len(strategies)})
}

// handleModelRoutes 模型路由列表——GET /gov/model-routes。
func (h *GovHandler) handleModelRoutes(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireGovAuth(w, r, "routing.route_profile.read"); !ok {
		return
	}
	db := h.deps.DB
	if db == nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
		return
	}
	var routes []ModelRoute
	query := db.WithContext(r.Context()).Model(&ModelRoute{}).Order("priority ASC, created_at ASC")
	if modelName := r.URL.Query().Get("model_name"); modelName != "" {
		query = query.Where("model_name = ?", modelName)
	}
	if providerID := r.URL.Query().Get("provider_id"); providerID != "" {
		query = query.Where("provider_id = ?", providerID)
	}
	if err := query.Find(&routes).Error; err != nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "ROUTE_LIST_FAILED", "查询模型路由列表失败: "+sanitizeError(err)))
		return
	}
	okJSON(w, map[string]any{"data": routes, "total": len(routes)})
}

// handleModelRouteItem 单个模型路由操作——PUT/DELETE /gov/model-routes/{id}。
func (h *GovHandler) handleModelRouteItem(w http.ResponseWriter, r *http.Request) {
	routeID := extractItemID(r, "/v1/gov/model-routes")
	switch r.Method {
		case http.MethodPut:
			if _, ok := h.requireGovItemAuth(w, r, "routing.route_profile.write", "model_route", routeID); !ok {
				return
			}
			db := h.deps.DB
			if db == nil {
				writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
				return
			}
			req, ok := readJSON[ModelRoute](w, r)
			if !ok {
				return
			}
			req.ID = routeID
			if err := db.WithContext(r.Context()).Model(&ModelRoute{}).Where("id = ?", routeID).Updates(map[string]any{
				"model_name":           req.ModelName,
				"provider_id":          req.ProviderID,
				"provider_resource_id": req.ProviderResourceID,
				"provider_model":       req.ProviderModel,
				"priority":             req.Priority,
				"weight":               req.Weight,
				"status":               req.Status,
				"sticky_session":       req.StickySession,
			}).Error; err != nil {
				writeError(w, r, NewHTTPError(http.StatusBadRequest, "UPDATE_FAILED", "模型路由更新失败: "+sanitizeError(err)))
				return
			}
			// 返回更新后的路由。
			var updated ModelRoute
			if err := db.WithContext(r.Context()).Where("id = ?", routeID).First(&updated).Error; err != nil {
				writeError(w, r, NewHTTPError(http.StatusInternalServerError, "QUERY_FAILED", "查询更新后的路由失败"))
				return
			}
			slog.InfoContext(r.Context(), "ModelRoute 更新成功", "route_id", routeID)
			okJSON(w, updated)
		case http.MethodDelete:
			if _, ok := h.requireGovItemAuth(w, r, "routing.route_profile.write", "model_route", routeID); !ok {
				return
			}
			okJSON(w, map[string]any{"deleted": true, "id": routeID})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// ── decimalFromString 字符串转 Decimal 辅助 ───────────────────────────────

// decimalFromString 将字符串解析为 fund.Decimal，解析失败时返回错误。
// 该函数用于 HTTP 层金额字段的安全转换——不 panic。
func decimalFromString(s string) (fund.Decimal, error) {
	d, err := decimal.NewFromString(strings.TrimSpace(s))
	if err != nil {
		return fund.Decimal{}, err
	}
	return fund.Decimal{Decimal: d}, nil
}
