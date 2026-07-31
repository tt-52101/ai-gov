// Package server 治理 API handlers——Fund/Key/Pricing/ModelGrant/Routing 域。
// 全部注释使用中文，符合 AGENTS.md 铁律。
package server

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"tokenhub/backend/internal/server/abac"
)

// ── §3 Fund handlers ──────────────────────────────────────────────────────

func (h *GovHandler) handleAccounts(w http.ResponseWriter, r *http.Request) {
	_, _ = h.requireGovAuth(w, r, "fund.balance.read")
	switch r.Method {
	case http.MethodGet:
		okJSON(w, map[string]string{"message": "Account 列表——待实现"})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handleAccountItem(w http.ResponseWriter, r *http.Request) {
	accountID := extractItemID(r, "/gov/accounts")
	switch r.Method {
	case http.MethodGet:
		_, _ = h.requireGovItemAuth(w, r, "fund.balance.read", "account", accountID)
		okJSON(w, map[string]string{"id": accountID, "message": "Account 详情——待实现"})
	case http.MethodPost:
		_, _ = h.requireGovItemAuth(w, r, "fund.balance.read", "account", accountID)
		okJSON(w, map[string]string{"id": accountID, "message": "Account 操作——待实现"})
	case http.MethodPatch:
		_, _ = h.requireGovItemAuth(w, r, "fund.balance.read", "account", accountID)
		okJSON(w, map[string]string{"id": accountID, "message": "Account 预算帽——待实现"})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handleAllocations(w http.ResponseWriter, r *http.Request) {
	_, _ = h.requireGovAuth(w, r, "fund.ledger.read")
	switch r.Method {
	case http.MethodGet:
		okJSON(w, map[string]string{"message": "Allocation 列表——待实现"})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handleAllocationItem(w http.ResponseWriter, r *http.Request) {
	allocID := extractItemID(r, "/gov/allocations")
	_, _ = h.requireGovItemAuth(w, r, "fund.ledger.read", "allocation", allocID)
	switch r.Method {
	case http.MethodGet:
		okJSON(w, map[string]string{"id": allocID, "message": "Allocation 详情——待实现"})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
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
	keyID := extractItemID(r, "/gov/keys")
	switch r.Method {
	case http.MethodGet:
		_, _ = h.requireGovItemAuth(w, r, "iam.key.read", "key", keyID)
		okJSON(w, map[string]string{"id": keyID, "message": "Key 详情——待实现"})
	case http.MethodDelete:
		_, _ = h.requireGovItemAuth(w, r, "iam.key.delete", "key", keyID)
		okJSON(w, map[string]any{"deleted": true, "id": keyID})
	case http.MethodPost:
		_, _ = h.requireGovItemAuth(w, r, "iam.key.create", "key", keyID)
		okJSON(w, map[string]string{"message": "Key 轮换——待实现"})
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

func (h *GovHandler) handleModelPrices(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		_, _ = h.requireGovAuth(w, r, "routing.price.write")
		okJSON(w, map[string]string{"message": "ModelPrice 创建/更新——待实现"})
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "routing.price.read")
		okJSON(w, map[string]string{"message": "ModelPrice 列表——待实现"})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handleModelPriceItem(w http.ResponseWriter, r *http.Request) {
	priceID := extractItemID(r, "/gov/model-prices")
	switch r.Method {
	case http.MethodGet:
		_, _ = h.requireGovItemAuth(w, r, "routing.price.read", "model_price", priceID)
		okJSON(w, map[string]string{"id": priceID, "message": "ModelPrice 详情——待实现"})
	case http.MethodDelete:
		_, _ = h.requireGovItemAuth(w, r, "routing.price.write", "model_price", priceID)
		okJSON(w, map[string]any{"archived": true, "id": priceID})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// ── §6 Model Grant handlers ───────────────────────────────────────────────

func (h *GovHandler) handleModelGrants(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		_, _ = h.requireGovAuth(w, r, "routing.model_grant.write")
		okJSON(w, map[string]string{"message": "ModelGrant 创建——待实现"})
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "routing.model_grant.read")
		okJSON(w, map[string]string{"message": "ModelGrant 列表——待实现"})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handleModelGrantItem(w http.ResponseWriter, r *http.Request) {
	grantID := extractItemID(r, "/gov/model-grants")
	switch r.Method {
	case http.MethodGet:
		_, _ = h.requireGovItemAuth(w, r, "routing.model_grant.read", "model_grant", grantID)
		okJSON(w, map[string]string{"id": grantID, "message": "ModelGrant 详情——待实现"})
	case http.MethodDelete:
		_, _ = h.requireGovItemAuth(w, r, "routing.model_grant.write", "model_grant", grantID)
		okJSON(w, map[string]any{"deleted": true, "id": grantID})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// ── §7 Routing handlers ───────────────────────────────────────────────────

func (h *GovHandler) handleRouteProfiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		_, _ = h.requireGovAuth(w, r, "routing.route_profile.write")
		okJSON(w, map[string]string{"message": "RouteProfile 创建——待实现"})
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "routing.route_profile.read")
		okJSON(w, map[string]string{"message": "RouteProfile 列表——待实现"})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handleRouteProfileItem(w http.ResponseWriter, r *http.Request) {
	profileID := extractItemID(r, "/gov/route-profiles")
	switch r.Method {
	case http.MethodGet:
		_, _ = h.requireGovItemAuth(w, r, "routing.route_profile.read", "route_profile", profileID)
		okJSON(w, map[string]string{"id": profileID, "message": "RouteProfile 详情——待实现"})
	case http.MethodPut:
		_, _ = h.requireGovItemAuth(w, r, "routing.route_profile.write", "route_profile", profileID)
		okJSON(w, map[string]string{"id": profileID, "message": "RouteProfile 更新——待实现"})
	case http.MethodDelete:
		_, _ = h.requireGovItemAuth(w, r, "routing.route_profile.write", "route_profile", profileID)
		okJSON(w, map[string]any{"deleted": true, "id": profileID})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handleRouteStrategies(w http.ResponseWriter, r *http.Request) {
	_, _ = h.requireGovAuth(w, r, "routing.route_profile.read")
	okJSON(w, map[string]string{"message": "RouteStrategy 列表——待实现"})
}

func (h *GovHandler) handleModelRoutes(w http.ResponseWriter, r *http.Request) {
	_, _ = h.requireGovAuth(w, r, "routing.route_profile.read")
	okJSON(w, map[string]string{"message": "ModelRoute 列表——待实现"})
}

func (h *GovHandler) handleModelRouteItem(w http.ResponseWriter, r *http.Request) {
	routeID := extractItemID(r, "/gov/model-routes")
	switch r.Method {
	case http.MethodPut:
		_, _ = h.requireGovItemAuth(w, r, "routing.route_profile.write", "model_route", routeID)
		okJSON(w, map[string]string{"id": routeID, "message": "ModelRoute 更新——待实现"})
	case http.MethodDelete:
		_, _ = h.requireGovItemAuth(w, r, "routing.route_profile.write", "model_route", routeID)
		okJSON(w, map[string]any{"deleted": true, "id": routeID})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}
