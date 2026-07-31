// Package server 治理 API handlers——Fund/Key/Pricing/ModelGrant/Routing 域。
// 全部注释使用中文，符合 AGENTS.md 铁律。
package server

import (
	"net/http"
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
	_ = accountID
	_, _ = h.requireGovAuth(w, r, "fund.balance.read")
	switch r.Method {
	case http.MethodGet:
		okJSON(w, map[string]string{"id": accountID, "message": "Account 详情——待实现"})
	case http.MethodPost:
		okJSON(w, map[string]string{"id": accountID, "message": "Account 操作——待实现"})
	case http.MethodPatch:
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
	_, _ = h.requireGovAuth(w, r, "fund.ledger.read")
	allocID := extractItemID(r, "/gov/allocations")
	_ = allocID
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
		_, _ = h.requireGovAuth(w, r, "iam.key.create")
		okJSON(w, map[string]string{"message": "Key 创建——待实现"})
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "iam.key.read")
		okJSON(w, map[string]string{"message": "Key 列表——待实现"})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handleKeyItem(w http.ResponseWriter, r *http.Request) {
	keyID := extractItemID(r, "/gov/keys")
	_ = keyID
	switch r.Method {
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "iam.key.read")
		okJSON(w, map[string]string{"id": keyID, "message": "Key 详情——待实现"})
	case http.MethodDelete:
		_, _ = h.requireGovAuth(w, r, "iam.key.delete")
		okJSON(w, map[string]any{"deleted": true, "id": keyID})
	case http.MethodPost:
		_, _ = h.requireGovAuth(w, r, "iam.key.create")
		okJSON(w, map[string]string{"message": "Key 轮换——待实现"})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
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
	_ = priceID
	switch r.Method {
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "routing.price.read")
		okJSON(w, map[string]string{"id": priceID, "message": "ModelPrice 详情——待实现"})
	case http.MethodDelete:
		_, _ = h.requireGovAuth(w, r, "routing.price.write")
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
	_ = grantID
	switch r.Method {
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "routing.model_grant.read")
		okJSON(w, map[string]string{"id": grantID, "message": "ModelGrant 详情——待实现"})
	case http.MethodDelete:
		_, _ = h.requireGovAuth(w, r, "routing.model_grant.write")
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
	_ = profileID
	switch r.Method {
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "routing.route_profile.read")
		okJSON(w, map[string]string{"id": profileID, "message": "RouteProfile 详情——待实现"})
	case http.MethodPut:
		_, _ = h.requireGovAuth(w, r, "routing.route_profile.write")
		okJSON(w, map[string]string{"id": profileID, "message": "RouteProfile 更新——待实现"})
	case http.MethodDelete:
		_, _ = h.requireGovAuth(w, r, "routing.route_profile.write")
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
	_ = routeID
	switch r.Method {
	case http.MethodPut:
		_, _ = h.requireGovAuth(w, r, "routing.route_profile.write")
		okJSON(w, map[string]string{"id": routeID, "message": "ModelRoute 更新——待实现"})
	case http.MethodDelete:
		_, _ = h.requireGovAuth(w, r, "routing.route_profile.write")
		okJSON(w, map[string]any{"deleted": true, "id": routeID})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}
