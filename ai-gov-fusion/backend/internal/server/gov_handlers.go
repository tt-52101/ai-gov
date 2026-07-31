// Package server 实现治理 API HTTP handlers——全部控制面端点。
// 按 api-spec-v3.2.md 规范注册所有 /gov/* 路由，使用与 TokenHub 一致的
// net/http 标准库 + http.ServeMux + HandleFunc 模式。
//
// 组织方式：按域拆分为独立 handler 函数，每个函数：
//  1. 从 context 提取主体（ABAC 鉴权已完成）
//  2. 解析请求体
//  3. 调用对应的 Service 层方法
//  4. 返回 JSON 响应 + 错误码
//
// 文件拆分：
//   - gov_handlers.go: 核心路由注册、工具函数、类型定义、Party 域 handlers
//   - gov_handlers_fund.go: Fund、Key、Pricing、ModelGrant、Routing 域 handlers
//   - gov_handlers_abac.go: ABAC、UI Permission、Audit、Dashboard 域 handlers
package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tokenhub/backend/internal/server/abac"
	"tokenhub/backend/internal/server/audit"
	"tokenhub/backend/internal/server/fund"
	"tokenhub/backend/internal/server/modelgrant"
	"tokenhub/backend/internal/server/party"
	"tokenhub/backend/internal/server/ui_permission"

	"gorm.io/gorm"
)

// ── GovDependencies 治理 API 依赖聚合 ─────────────────────────────────────

// GovDependencies 聚合所有治理服务依赖。
// 每个字段对应一个领域服务，通过 RegisterGovHandlers 注入到路由中。
// 未注入的服务对应端点会返回 501（未实现）。
type GovDependencies struct {
	// FundService 资金治理服务：账本、划拨、冻结、清算、预算帽。
	FundService *fund.Service
	// PartyService 主体管理服务：组织/项目 CRUD、关系边、成员管理。
	PartyService *party.Service
	// PricingDB pricing 自由函数的数据库句柄。
	PricingDB *gorm.DB
	// ABACEngine ABAC 策略评估引擎——所有控制面端点的统一鉴权。
	ABACEngine *abac.Engine
	// ModelGrantChecker 模型授权检查器。
	ModelGrantChecker *modelgrant.Checker
	// UIPermProjector UI 权限投影器——将 ABAC 决策映射到前端菜单/路由/按钮。
	UIPermProjector *ui_permission.Projector
	// AuditRecorder 审计事件记录函数。若为 nil 则跳过审计。
	AuditRecorder func(ctx *GovRequestContext, event *audit.AuditEvent) error
	// RouteProfileDB 路由档案数据库句柄。
	RouteProfileDB *gorm.DB
	// DB 主数据库句柄——用于直接查询表。
	DB *gorm.DB

	// ── 数据面 Pipeline 依赖 ──────────────────────────────────────────────

	// Pipeline 14 步数据面管线编排器——注入后由 /v1/chat/completions 调用。
	// 若为 nil，回退到原有 startRoutedCall 路径。
	Pipeline *Pipeline
	// Integrator StartCall 事务插桩适配器——安全钩子/ModelGrant/定价/预算帽/冻结。
	// 由 DefaultIntegrator 或 NoopIntegrator 实现。
	Integrator StartCallIntegrator
}

// ── GovHandler 治理 API 处理器 ────────────────────────────────────────────

// GovHandler 治理 API HTTP handler 集合。
// 每个公开方法对应一个 HTTP handler，签名与 http.HandlerFunc 兼容。
type GovHandler struct {
	deps GovDependencies
}

// NewGovHandler 创建新的治理 API 处理器实例。
func NewGovHandler(deps GovDependencies) *GovHandler {
	return &GovHandler{deps: deps}
}

// ── GovRequestContext 请求上下文 ───────────────────────────────────────────

// GovRequestContext 治理 API 请求上下文——从 HTTP 请求中提取的公共信息。
type GovRequestContext struct {
	// RequestID 全链路唯一请求标识。
	RequestID string
	// SubjectType 主体类型（user / party）。
	SubjectType string
	// SubjectID 主体 ID。
	SubjectID string
	// UserName 操作者显示名。
	UserName string
	// ClientIP 客户端 IP。
	ClientIP string
	// UserAgent 客户端 User-Agent。
	UserAgent string
}

// ── 路由注册 ──────────────────────────────────────────────────────────────

// RegisterGovHandlers 在 TokenHub 现有 mux 上注册全部 /gov/* 路由。
// 所有路由统一前缀 /gov，与控制台 API /api/admin 完全隔离。
// 用法：在 Server 初始化中调用 RegisterGovHandlers(s.mux, govDeps)。
func RegisterGovHandlers(mux *http.ServeMux, deps GovDependencies) {
	h := NewGovHandler(deps)

	// ── §2 Party（主体管理）────────────────────────────────────
	mux.HandleFunc("/v1/gov/parties", wrapGovHandler(h.handleParties))
	mux.HandleFunc("/v1/gov/parties/", wrapGovHandler(h.handlePartyItem))
	mux.HandleFunc("/v1/gov/party-edges", wrapGovHandler(h.handlePartyEdges))
	mux.HandleFunc("/v1/gov/party-edges/", wrapGovHandler(h.handlePartyEdgeItem))
	mux.HandleFunc("/v1/gov/party-members", wrapGovHandler(h.handlePartyMembers))
	mux.HandleFunc("/v1/gov/party-members/", wrapGovHandler(h.handlePartyMemberItem))

	// ── §3 Fund（资金治理）────────────────────────────────────
	mux.HandleFunc("/v1/gov/accounts", wrapGovHandler(h.handleAccounts))
	mux.HandleFunc("/v1/gov/accounts/", wrapGovHandler(h.handleAccountItem))
	mux.HandleFunc("/v1/gov/allocations", wrapGovHandler(h.handleAllocations))
	mux.HandleFunc("/v1/gov/allocations/", wrapGovHandler(h.handleAllocationItem))

	// ── §4 Key（密钥管理）─────────────────────────────────────
	mux.HandleFunc("/v1/gov/keys", wrapGovHandler(h.handleKeys))
	mux.HandleFunc("/v1/gov/keys/", wrapGovHandler(h.handleKeyItem))

	// ── §5 Pricing（双轨计价）─────────────────────────────────
	mux.HandleFunc("/v1/gov/model-prices", wrapGovHandler(h.handleModelPrices))
	mux.HandleFunc("/v1/gov/model-prices/", wrapGovHandler(h.handleModelPriceItem))

	// ── §6 Model Grant（模型授权）─────────────────────────────
	mux.HandleFunc("/v1/gov/model-grants", wrapGovHandler(h.handleModelGrants))
	mux.HandleFunc("/v1/gov/model-grants/", wrapGovHandler(h.handleModelGrantItem))

	// ── §7 Routing（路由调度）─────────────────────────────────
	mux.HandleFunc("/v1/gov/route-profiles", wrapGovHandler(h.handleRouteProfiles))
	mux.HandleFunc("/v1/gov/route-profiles/", wrapGovHandler(h.handleRouteProfileItem))
	mux.HandleFunc("/v1/gov/route-strategies", wrapGovHandler(h.handleRouteStrategies))
	mux.HandleFunc("/v1/gov/model-routes", wrapGovHandler(h.handleModelRoutes))
	mux.HandleFunc("/v1/gov/model-routes/", wrapGovHandler(h.handleModelRouteItem))

	// ── §8 ABAC（策略引擎）────────────────────────────────────
	mux.HandleFunc("/v1/gov/action-catalogs", wrapGovHandler(h.handleActionCatalogs))
	mux.HandleFunc("/v1/gov/roles", wrapGovHandler(h.handleRoles))
	mux.HandleFunc("/v1/gov/roles/", wrapGovHandler(h.handleRoleItem))
	mux.HandleFunc("/v1/gov/policies", wrapGovHandler(h.handlePolicies))
	mux.HandleFunc("/v1/gov/policies/", wrapGovHandler(h.handlePolicyItem))
	mux.HandleFunc("/v1/gov/subject-role-bindings", wrapGovHandler(h.handleSubjectRoleBindings))
	mux.HandleFunc("/v1/gov/subject-role-bindings/", wrapGovHandler(h.handleSubjectRoleBindingItem))
	mux.HandleFunc("/v1/gov/grants", wrapGovHandler(h.handleGrants))
	mux.HandleFunc("/v1/gov/grants/", wrapGovHandler(h.handleGrantItem))

	// ── §9 UI Permission（UI权限治理）─────────────────────────
	mux.HandleFunc("/v1/gov/ui-menus", wrapGovHandler(h.handleUIMenus))
	mux.HandleFunc("/v1/gov/ui-menus/", wrapGovHandler(h.handleUIMenuItem))
	mux.HandleFunc("/v1/gov/ui-routes", wrapGovHandler(h.handleUIRoutes))
	mux.HandleFunc("/v1/gov/ui-routes/", wrapGovHandler(h.handleUIRouteItem))
	mux.HandleFunc("/v1/gov/ui-action-bindings", wrapGovHandler(h.handleUIActionBindings))
	mux.HandleFunc("/v1/gov/ui-action-bindings/", wrapGovHandler(h.handleUIActionBindingItem))
	mux.HandleFunc("/v1/gov/ui-permissions/snapshot", wrapGovHandler(h.handleUIPermissionSnapshot))

		// ── §10 Audit（审计与对账）────────────────────────────────
		mux.HandleFunc("/v1/gov/audit-events", wrapGovHandler(h.handleAuditEvents))
		mux.HandleFunc("/v1/gov/audit-events/", wrapGovHandler(h.handleAuditEventItem))
		mux.HandleFunc("/v1/gov/request-logs", wrapGovHandler(h.handleRequestLogs))
		mux.HandleFunc("/v1/gov/request-logs/", wrapGovHandler(h.handleRequestLogTrace))
		mux.HandleFunc("/v1/gov/audit-chain-anchors", wrapGovHandler(h.handleAuditChainAnchors))
		mux.HandleFunc("/v1/gov/reconciliation-runs", wrapGovHandler(h.handleReconciliationRuns))
		mux.HandleFunc("/v1/gov/reconciliation-runs/", wrapGovHandler(h.handleReconciliationRunItem))

	// ── §11 Dashboard（仪表盘与报表）──────────────────────────
	mux.HandleFunc("/v1/gov/dashboard", wrapGovHandler(h.handleDashboard))
	mux.HandleFunc("/v1/gov/security-reports", wrapGovHandler(h.handleSecurityReports))
	mux.HandleFunc("/v1/gov/trace", wrapGovHandler(h.handleTrace))
}

// ── 通用包装器 ────────────────────────────────────────────────────────────

// wrapGovHandler 将 GovHandler 方法包装为 http.HandlerFunc。
// 统一设置 Content-Type、提取请求上下文、记录响应日志。
func wrapGovHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fn(w, r)
	}
}

// ── ABAC 鉴权辅助 ─────────────────────────────────────────────────────────

// requireGovAuth 从请求中提取治理请求上下文，并执行 ABAC 鉴权。
// 返回 GovRequestContext 和是否放行。
// 注意：此为列表/集合端点鉴权——Resource.ID 设为 URL 路径。
// 单品端点应使用 requireGovItemAuth 以正确传入资源 ID 和 PartyID。
func (h *GovHandler) requireGovAuth(w http.ResponseWriter, r *http.Request, action string) (*GovRequestContext, bool) {
	gctx := &GovRequestContext{
		RequestID: r.Header.Get("X-Request-ID"),
		ClientIP:  govClientIP(r),
		UserAgent: r.UserAgent(),
		SubjectType: "user",
	}
	if gctx.RequestID == "" {
		gctx.RequestID = NewID("gov")
	}
		// 从 Header 提取认证——Bearer Token 或 X-API-Key。
		if token := extractBearerToken(r); token != "" {
			if user, userID, ok := h.validateGovToken(token); ok {
				gctx.SubjectID = userID
				gctx.UserName = user
			}
		} else if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
			gctx.SubjectID = apiKey
		}
	if gctx.SubjectID == "" {
		writeError(w, r, NewHTTPError(http.StatusUnauthorized, "AUTH_INVALID_KEY", "认证凭证无效或缺失"))
		return nil, false
	}
	// ABAC 鉴权——若未配置引擎则跳过（开发模式）。
	if h.deps.ABACEngine != nil && action != "" {
		subject := abac.Subject{Type: gctx.SubjectType, ID: gctx.SubjectID}
		resource := abac.Resource{Type: "gov_api", ID: r.URL.Path}
		if err := h.deps.ABACEngine.Evaluate(r.Context(), subject, action, resource); err != nil {
			writeError(w, r, NewHTTPError(http.StatusForbidden, "AUTHZ_DENIED", "权限不足: "+sanitizeError(err)))
			return nil, false
		}
	}
	return gctx, true
}

// requireGovItemAuth 对单品端点执行 ABAC 鉴权，传入完整的资源归属信息。
//
// 与 requireGovAuth 的区别：
//   - Resource.ID 设为资源实际主键（而非 URL 路径），使 ABAC 精确到具体资源
//   - 查询 DB 获取资源所属 PartyID，传入 Resource.PartyID，使 scope_party_id 角色绑定生效
//   - 覆盖 IDOR 归属校验——ABAC 策略可依赖 Resource.PartyID 判定跨组织越权
//
// resourceType 为资源类型标识（如 "account", "party", "model_grant"）。
// resourceID 为从 URL 提取的资源主键。
func (h *GovHandler) requireGovItemAuth(w http.ResponseWriter, r *http.Request, action, resourceType, resourceID string) (*GovRequestContext, bool) {
	gctx := &GovRequestContext{
		RequestID: r.Header.Get("X-Request-ID"),
		ClientIP:  govClientIP(r),
		UserAgent: r.UserAgent(),
		SubjectType: "user",
	}
	if gctx.RequestID == "" {
		gctx.RequestID = NewID("gov")
	}
		// 从 Header 提取认证——Bearer Token 或 X-API-Key。
		if token := extractBearerToken(r); token != "" {
			if user, userID, ok := h.validateGovToken(token); ok {
				gctx.SubjectID = userID
				gctx.UserName = user
			}
		} else if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
			gctx.SubjectID = apiKey
		}
	if gctx.SubjectID == "" {
		writeError(w, r, NewHTTPError(http.StatusUnauthorized, "AUTH_INVALID_KEY", "认证凭证无效或缺失"))
		return nil, false
	}
	// ABAC 鉴权——若未配置引擎则跳过（开发模式）。
	if h.deps.ABACEngine != nil && action != "" {
		// 查询资源所属 party_id，用于 scope_party_id 角色绑定 + IDOR 归属校验。
		partyID := lookupResourceParty(h.deps.DB, r.Context(), resourceType, resourceID)
		subject := abac.Subject{Type: gctx.SubjectType, ID: gctx.SubjectID}
		resource := abac.Resource{
			Type:    resourceType,
			ID:      resourceID,
			PartyID: partyID,
		}
		if err := h.deps.ABACEngine.Evaluate(r.Context(), subject, action, resource); err != nil {
			writeError(w, r, NewHTTPError(http.StatusForbidden, "AUTHZ_DENIED", "权限不足: "+sanitizeError(err)))
			return nil, false
		}
	}
	return gctx, true
}

// lookupResourceParty 查询指定资源类型的所属 party_id。
//
// 资源类型到 DB 表 + 列的映射：
//   - "party"       → parties 表，party_id = resourceID（自身即 party）
//   - "account"     → fund_accounts 表，party_id 列
//   - "key"         → api_keys 表，party_id 列
//   - "allocation"  → fund_allocations 表，party_id 列
//   - "model_grant" → model_grants 表，party_id 列
//   - "route_profile" → route_profiles 表，party_id 列
//   - "model_price" / "role" / "policy" 等系统级资源 → 返回空字符串
//
// 返回空字符串表示资源无 party 归属或不存在。ABAC 引擎将 PartyID 为空等同为
// "不做 scope 过滤"，对系统级资源这是预期行为。
func lookupResourceParty(db *gorm.DB, ctx context.Context, resourceType, resourceID string) string {
	if db == nil || resourceID == "" {
		return ""
	}

	// 各资源类型的 party_id 查询映射。
	type partyQuery struct {
		table    string
		idColumn string
		col      string // party_id 列名
	}
	var mapping *partyQuery
	switch resourceType {
	case "party":
		// Party 自身即 party——ID 同时也是 party_id。
		mapping = &partyQuery{table: "parties", idColumn: "id", col: "id"}
	case "account":
		mapping = &partyQuery{table: "fund_accounts", idColumn: "id", col: "party_id"}
	case "key":
		mapping = &partyQuery{table: "api_keys", idColumn: "id", col: "party_id"}
	case "allocation":
		mapping = &partyQuery{table: "fund_allocations", idColumn: "id", col: "party_id"}
	case "model_grant":
		mapping = &partyQuery{table: "model_grants", idColumn: "id", col: "party_id"}
	case "route_profile":
		mapping = &partyQuery{table: "route_profiles", idColumn: "id", col: "party_id"}
	case "model_price", "role", "policy", "subject_role_binding":
		// 系统级资源，无 party_id 归属。
		return ""
	default:
		// 未知资源类型，不做 scope 过滤。
		return ""
	}

	var partyID string
	err := db.WithContext(ctx).
		Table(mapping.table).
		Select(mapping.col).
		Where(mapping.idColumn+" = ?", resourceID).
		Limit(1).
		Scan(&partyID).Error
	if err != nil {
		slog.WarnContext(ctx, "查询资源 party_id 失败",
			"resource_type", resourceType,
			"resource_id", resourceID,
			"error", err,
		)
		return ""
	}
	return partyID
}

// ── 错误脱敏 ────────────────────────────────────────────────────────────────

// sanitizeError 脱敏错误信息，防止 account_id / freeze_id 等内部标识泄露到 HTTP 响应体。
//
// 策略：
//   - HTTPError 类型：保留其 Message（业务层显式设置的消息视为已脱敏）
//   - GORM / 数据库错误：返回通用"服务器内部错误"
//   - 其他 fmt.Errorf 包装的错误：返回通用"服务器内部错误"
//
// 调试时可将最后的 return 改为 err.Error() 以获得完整错误信息。
func sanitizeError(err error) string {
	if err == nil {
		return "未知错误"
	}
	// HTTPError 类型——Message 由业务层显式设置，视为已脱敏。
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.Message
	}
	// 非 HTTPError 类型——生产环境返回统一错误码，避免泄露内部细节。
	// 调试：return err.Error()
	return "服务器内部错误，请稍后重试"
}

// ── JSON 辅助 ─────────────────────────────────────────────────────────────

// readJSON 解析请求体为指定类型。解析失败时写入错误响应并返回 false。
func readJSON[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var body T
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_JSON", "请求体解析失败: "+sanitizeError(err)))
		return body, false
	}
	return body, true
}

// okJSON 写入 200 成功响应。
func okJSON(w http.ResponseWriter, data any) {
	if data == nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	writeJSON(w, http.StatusOK, data)
}

// createdJSON 写入 201 创建成功响应。
func createdJSON(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusCreated, data)
}

// ── URL 路径参数提取 ──────────────────────────────────────────────────────

// extractItemID 从带 trailing slash 的集合路由中提取资源 ID。
// 例如 URL /gov/parties/{id} → 返回 {id}；/gov/parties 或 /gov/parties/ → 返回 ""。
func extractItemID(r *http.Request, prefix string) string {
	path := r.URL.Path
	if path == prefix || path == prefix+"/" {
		return ""
	}
	if len(path) > len(prefix)+1 {
		return path[len(prefix)+1:]
	}
	return ""
}

// ── 认证辅助 ──────────────────────────────────────────────────────────────

// extractBearerToken 从 Authorization header 提取 Bearer Token。
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && strings.EqualFold(auth[:7], "Bearer ") {
		return auth[7:]
	}
	return ""
}

// validateGovToken 验证治理 API Bearer Token——完整实现。
// 校验链：
//  1. 对传入 token 做 SHA-256 哈希。
//  2. 在 gov_api_keys 表中查询匹配记录。
//  3. 校验密钥状态为 active、未过期。
//  4. 校验所有者用户（owner_user_id）未被禁用——禁人即禁Key。
//
// 返回 (userName, userID, ok)：ok 为 true 时校验通过，注入 Subject 到 context。
func (h *GovHandler) validateGovToken(token string) (string, string, bool) {
	if token == "" {
		return "", "", false
	}

	// 1. SHA-256 哈希。
	sum := sha256.Sum256([]byte(token))
	keyHash := hex.EncodeToString(sum[:])

	// 2. 查询 gov_api_keys 表。
	var key GovAPIKey
	db := h.deps.DB
	if db == nil {
		slog.Warn("validateGovToken: DB 未配置，无法校验 Token")
		return "", "", false
	}
	if err := db.Where("key_hash = ?", keyHash).First(&key).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Error("validateGovToken: 查询 gov_api_keys 失败", "error", err)
		}
		return "", "", false
	}

	// 3. 校验密钥状态。
	if key.Status != "" && key.Status != StatusActive {
		return "", "", false
	}

	// 校验密钥未过期。
	if key.ExpiresAt != nil && time.Now().UTC().After(*key.ExpiresAt) {
		return "", "", false
	}

	// 4. 禁人即禁Key——校验所有者用户状态。
	var user AdminUser
	if key.OwnerUserID != "" {
		if err := db.Where("id = ?", key.OwnerUserID).First(&user).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				slog.Error("validateGovToken: 查询 admin_users 失败", "error", err, "owner_user_id", key.OwnerUserID)
			}
			return "", "", false
		}
		// 用户被禁用（非 active 状态）→ Key 立即失效。
		if user.Status != "" && user.Status != StatusActive {
			slog.Warn("validateGovToken: 用户已被禁用，拒绝 Key 使用",
				"key_id", key.ID,
				"owner_user_id", key.OwnerUserID,
				"user_status", user.Status,
			)
			return "", "", false
		}
	}

	// 5. 校验通过——更新 LastUsedAt。
	now := time.Now().UTC()
	_ = db.Model(&key).Update("last_used_at", now).Error

	// 返回所有者的用户名和用户 ID，作为 ABAC 鉴权的 Subject。
	// ABAC 策略绑定到用户，而非密钥，因此 SubjectID 必须为 owner_user_id。
	userName := user.Name
	if userName == "" {
		userName = key.OwnerUserID
	}
	return userName, key.OwnerUserID, true
}

// govClientIP 从 HTTP 请求提取客户端 IP。
func govClientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		return host[:idx]
	}
	return host
}

// ── §2 Party handlers ─────────────────────────────────────────────────────

// handleParties 处理 /gov/parties 集合端点。
//
// 路由：
//   - POST /gov/parties → 创建组织或项目主体（ABAC: iam.party.create）
//   - GET /gov/parties → 列表查询，支持 ?type=org|project 筛选（ABAC: data.party.read）
func (h *GovHandler) handleParties(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		gctx, _ := h.requireGovAuth(w, r, "iam.party.create")
		if gctx == nil {
			return
		}
		if h.deps.PartyService == nil {
			writeError(w, r, NewHTTPError(501, "NOT_IMPLEMENTED", "Party 服务未配置"))
			return
		}
		req, ok := readJSON[party.CreatePartyRequest](w, r)
		if !ok {
			return
		}
		p, err := h.deps.PartyService.CreateParty(r.Context(), req)
		if err != nil {
			writeError(w, r, NewHTTPError(400, "CREATE_FAILED", sanitizeError(err)))
			return
		}
		slog.InfoContext(r.Context(), "创建Party成功", "party_id", p.ID, "name", p.Name)
		createdJSON(w, p)

	case http.MethodGet:
		gctx, _ := h.requireGovAuth(w, r, "data.party.read")
		if gctx == nil {
			return
		}
		if h.deps.PartyService == nil {
			writeError(w, r, NewHTTPError(501, "NOT_IMPLEMENTED", "Party 服务未配置"))
			return
		}

		// 按 ?type=org|project 筛选，空字符串返回全部。
		partyType := strings.TrimSpace(r.URL.Query().Get("type"))
		parties, err := h.deps.PartyService.GetParties(r.Context(), partyType)
		if err != nil {
			writeError(w, r, NewHTTPError(500, "PARTY_LIST_FAILED", "查询Party列表失败: "+sanitizeError(err)))
			return
		}

		slog.InfoContext(r.Context(), "查询Party列表",
			"type_filter", partyType,
			"count", len(parties),
			"actor", gctx.SubjectID,
		)

		okJSON(w, map[string]any{
			"data":  parties,
			"total": len(parties),
		})

	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handlePartyItem 处理 /gov/parties/{id} 单品端点。
//
// 路由：
//   - GET /gov/parties/{id} → 查询单品详情（ABAC: data.party.read）
//   - PATCH /gov/parties/{id} → 更新状态（ABAC: iam.party.write）
func (h *GovHandler) handlePartyItem(w http.ResponseWriter, r *http.Request) {
	partyIDStr := extractItemID(r, "/gov/parties")
	if partyIDStr == "" {
		writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "缺少 party_id"))
		return
	}
	partyID, err := strconv.ParseInt(partyIDStr, 10, 64)
	if err != nil {
		writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "party_id 格式无效: "+partyIDStr))
		return
	}

	switch r.Method {
	case http.MethodGet:
		gctx, _ := h.requireGovItemAuth(w, r, "data.party.read", "party", partyIDStr)
		if gctx == nil {
			return
		}
		if h.deps.PartyService == nil {
			writeError(w, r, NewHTTPError(501, "NOT_IMPLEMENTED", "Party 服务未配置"))
			return
		}

		// 通过 party 包级函数直接查询。
		p, err := party.GetParty(h.deps.PartyService.DB, partyID)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusNotFound, "PARTY_NOT_FOUND", "Party 不存在: "+partyIDStr))
			return
		}

		slog.InfoContext(r.Context(), "查询Party详情",
			"party_id", partyID,
			"actor", gctx.SubjectID,
		)
		okJSON(w, p)

	case http.MethodPatch:
		gctx, _ := h.requireGovItemAuth(w, r, "iam.party.write", "party", partyIDStr)
		if gctx == nil {
			return
		}
		if h.deps.PartyService == nil {
			writeError(w, r, NewHTTPError(501, "NOT_IMPLEMENTED", "Party 服务未配置"))
			return
		}

		// 解析状态更新请求。
		type partyStatusUpdate struct {
			Status string `json:"status"`
		}
		req, ok := readJSON[partyStatusUpdate](w, r)
		if !ok {
			return
		}
		if strings.TrimSpace(req.Status) == "" {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "status 为必填字段"))
			return
		}

		if err := party.UpdatePartyStatus(h.deps.PartyService.DB, partyID, strings.TrimSpace(req.Status)); err != nil {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "UPDATE_FAILED", sanitizeError(err)))
			return
		}

		slog.InfoContext(r.Context(), "更新Party状态",
			"party_id", partyID,
			"new_status", req.Status,
			"actor", gctx.SubjectID,
		)

		// 返回更新后的 Party。
		p, err := party.GetParty(h.deps.PartyService.DB, partyID)
		if err != nil {
			writeError(w, r, NewHTTPError(500, "PARTY_QUERY_FAILED", "查询更新后的Party失败"))
			return
		}
		okJSON(w, p)

	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handlePartyEdges 处理 /gov/party-edges 集合端点。
//
// 路由：
//   - POST /gov/party-edges → 创建关系边（ABAC: iam.party.write）
//   - GET /gov/party-edges → 列表查询（待实现）
func (h *GovHandler) handlePartyEdges(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		gctx, _ := h.requireGovAuth(w, r, "iam.party.write")
		if gctx == nil {
			return
		}
		if h.deps.PartyService == nil {
			writeError(w, r, NewHTTPError(501, "NOT_IMPLEMENTED", "Party 服务未配置"))
			return
		}
		req, ok := readJSON[party.CreateEdgeRequest](w, r)
		if !ok {
			return
		}
		edge, err := h.deps.PartyService.CreateEdge(r.Context(), req)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "CREATE_EDGE_FAILED", sanitizeError(err)))
			return
		}
		slog.InfoContext(r.Context(), "创建PartyEdge成功",
			"edge_id", edge.ID,
			"src_party_id", edge.SrcPartyID,
			"dst_party_id", edge.DstPartyID,
			"edge_type", edge.EdgeType,
			"actor", gctx.SubjectID,
		)
		createdJSON(w, edge)

	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "iam.party.write")
		okJSON(w, map[string]string{"message": "PartyEdge 列表——待实现"})

	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handlePartyEdgeItem 处理 /gov/party-edges/{id} 单品端点。
//
// 路由：
//   - DELETE /gov/party-edges/{id} → 删除关系边（ABAC: iam.party.write）
func (h *GovHandler) handlePartyEdgeItem(w http.ResponseWriter, r *http.Request) {
	edgeIDStr := extractItemID(r, "/gov/party-edges")
	if edgeIDStr == "" {
		writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "缺少 edge_id"))
		return
	}
	edgeID, err := strconv.ParseInt(edgeIDStr, 10, 64)
	if err != nil {
		writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "edge_id 格式无效: "+edgeIDStr))
		return
	}

	_, _ = h.requireGovItemAuth(w, r, "iam.party.write", "party_edge", edgeIDStr)

	switch r.Method {
	case http.MethodDelete:
		if h.deps.PartyService == nil {
			writeError(w, r, NewHTTPError(501, "NOT_IMPLEMENTED", "Party 服务未配置"))
			return
		}
		if err := h.deps.PartyService.DeleteEdge(r.Context(), edgeID); err != nil {
			writeError(w, r, NewHTTPError(http.StatusNotFound, "DELETE_EDGE_FAILED", sanitizeError(err)))
			return
		}
		slog.InfoContext(r.Context(), "删除PartyEdge成功", "edge_id", edgeID)
		okJSON(w, map[string]any{"deleted": true, "id": edgeID})

	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handlePartyMembers 处理 /gov/party-members 集合端点。
//
// 路由：
//   - POST /gov/party-members → 添加成员到 Party（ABAC: iam.member.write）
//   - GET /gov/party-members → 成员列表（待实现）
func (h *GovHandler) handlePartyMembers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		gctx, _ := h.requireGovAuth(w, r, "iam.member.write")
		if gctx == nil {
			return
		}
		if h.deps.PartyService == nil {
			writeError(w, r, NewHTTPError(501, "NOT_IMPLEMENTED", "Party 服务未配置"))
			return
		}
		req, ok := readJSON[party.AddMemberRequest](w, r)
		if !ok {
			return
		}
		member, err := h.deps.PartyService.AddMember(r.Context(), req)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "ADD_MEMBER_FAILED", sanitizeError(err)))
			return
		}
		slog.InfoContext(r.Context(), "添加PartyMember成功",
			"member_id", member.ID,
			"party_id", member.PartyID,
			"user_id", member.UserID,
			"role", member.Role,
			"actor", gctx.SubjectID,
		)
		createdJSON(w, member)

	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "data.member.read")
		okJSON(w, map[string]string{"message": "PartyMember 列表——待实现"})

	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handlePartyMemberItem 处理 /gov/party-members/{id} 单品端点。
//
// 路由：
//   - DELETE /gov/party-members/{id} → 移除成员（ABAC: iam.member.delete）
func (h *GovHandler) handlePartyMemberItem(w http.ResponseWriter, r *http.Request) {
	memberIDStr := extractItemID(r, "/gov/party-members")
	if memberIDStr == "" {
		writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "缺少 member_id"))
		return
	}
	memberID, err := strconv.ParseInt(memberIDStr, 10, 64)
	if err != nil {
		writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "member_id 格式无效: "+memberIDStr))
		return
	}

	_, _ = h.requireGovItemAuth(w, r, "iam.member.delete", "party_member", memberIDStr)

	switch r.Method {
	case http.MethodDelete:
		if h.deps.PartyService == nil {
			writeError(w, r, NewHTTPError(501, "NOT_IMPLEMENTED", "Party 服务未配置"))
			return
		}
		if err := h.deps.PartyService.RemoveMember(r.Context(), memberID); err != nil {
			writeError(w, r, NewHTTPError(http.StatusNotFound, "REMOVE_MEMBER_FAILED", sanitizeError(err)))
			return
		}
		slog.InfoContext(r.Context(), "移除PartyMember成功", "member_id", memberID)
		okJSON(w, map[string]any{"deleted": true, "id": memberID})

	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}
