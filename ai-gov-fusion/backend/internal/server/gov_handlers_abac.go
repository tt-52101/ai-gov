// Package server 治理 API handlers——ABAC/UI Permission/Audit/Dashboard 域。
// 全部注释使用中文，符合 AGENTS.md 铁律。
package server

import (
	"net/http"
)

// ── §8 ABAC handlers ──────────────────────────────────────────────────────

func (h *GovHandler) handleActionCatalogs(w http.ResponseWriter, r *http.Request) {
	_, _ = h.requireGovAuth(w, r, "iam.role.read")
	okJSON(w, map[string]string{"message": "ActionCatalog 列表——待实现"})
}

func (h *GovHandler) handleRoles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		_, _ = h.requireGovAuth(w, r, "iam.role.write")
		okJSON(w, map[string]string{"message": "Role 创建——待实现"})
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "iam.role.read")
		okJSON(w, map[string]string{"message": "Role 列表——待实现"})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handleRoleItem(w http.ResponseWriter, r *http.Request) {
	roleID := extractItemID(r, "/gov/roles")
	_ = roleID
	switch r.Method {
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "iam.role.read")
		okJSON(w, map[string]string{"id": roleID, "message": "Role 详情——待实现"})
	case http.MethodPut:
		_, _ = h.requireGovAuth(w, r, "iam.role.write")
		okJSON(w, map[string]string{"id": roleID, "message": "Role 更新——待实现"})
	case http.MethodDelete:
		_, _ = h.requireGovAuth(w, r, "iam.role.write")
		okJSON(w, map[string]any{"deleted": true, "id": roleID})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handlePolicies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		_, _ = h.requireGovAuth(w, r, "iam.policy.write")
		okJSON(w, map[string]string{"message": "Policy 创建——待实现"})
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "iam.policy.read")
		okJSON(w, map[string]string{"message": "Policy 列表——待实现"})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handlePolicyItem(w http.ResponseWriter, r *http.Request) {
	policyID := extractItemID(r, "/gov/policies")
	switch r.Method {
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "iam.policy.read")
		okJSON(w, map[string]string{"id": policyID, "message": "Policy 详情——待实现"})
	case http.MethodPut:
		_, _ = h.requireGovAuth(w, r, "iam.policy.write")
		okJSON(w, map[string]string{"id": policyID, "message": "Policy 更新——待实现"})
	case http.MethodDelete:
		_, _ = h.requireGovAuth(w, r, "iam.policy.write")
		okJSON(w, map[string]any{"deleted": true, "id": policyID})
	case http.MethodPost:
		_, _ = h.requireGovAuth(w, r, "iam.policy.read")
		okJSON(w, map[string]string{"message": "Policy 评估——待实现"})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handleSubjectRoleBindings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		_, _ = h.requireGovAuth(w, r, "iam.role.write")
		okJSON(w, map[string]string{"message": "SubjectRoleBinding 创建——待实现"})
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "iam.role.read")
		okJSON(w, map[string]string{"message": "SubjectRoleBinding 列表——待实现"})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handleSubjectRoleBindingItem(w http.ResponseWriter, r *http.Request) {
	bindingID := extractItemID(r, "/gov/subject-role-bindings")
	switch r.Method {
	case http.MethodDelete:
		_, _ = h.requireGovAuth(w, r, "iam.role.write")
		okJSON(w, map[string]any{"deleted": true, "id": bindingID})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handleGrants(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		_, _ = h.requireGovAuth(w, r, "iam.policy.write")
		okJSON(w, map[string]string{"message": "Grant 创建——待实现"})
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "iam.policy.read")
		okJSON(w, map[string]string{"message": "Grant 列表——待实现"})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handleGrantItem(w http.ResponseWriter, r *http.Request) {
	grantID := extractItemID(r, "/gov/grants")
	switch r.Method {
	case http.MethodDelete:
		_, _ = h.requireGovAuth(w, r, "iam.policy.write")
		okJSON(w, map[string]any{"deleted": true, "id": grantID})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// ── §9 UI Permission handlers ─────────────────────────────────────────────

func (h *GovHandler) handleUIMenus(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		_, _ = h.requireGovAuth(w, r, "iam.ui.write")
		okJSON(w, map[string]string{"message": "UIMenu 创建——待实现"})
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "iam.ui.read")
		okJSON(w, map[string]string{"message": "UIMenu 列表——待实现"})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handleUIMenuItem(w http.ResponseWriter, r *http.Request) {
	menuID := extractItemID(r, "/gov/ui-menus")
	switch r.Method {
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "iam.ui.read")
		okJSON(w, map[string]string{"id": menuID, "message": "UIMenu 详情——待实现"})
	case http.MethodPut:
		_, _ = h.requireGovAuth(w, r, "iam.ui.write")
		okJSON(w, map[string]string{"id": menuID, "message": "UIMenu 更新——待实现"})
	case http.MethodDelete:
		_, _ = h.requireGovAuth(w, r, "iam.ui.write")
		okJSON(w, map[string]any{"deleted": true, "id": menuID})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handleUIRoutes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		_, _ = h.requireGovAuth(w, r, "iam.ui.write")
		okJSON(w, map[string]string{"message": "UIRoute 创建——待实现"})
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "iam.ui.read")
		okJSON(w, map[string]string{"message": "UIRoute 列表——待实现"})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handleUIRouteItem(w http.ResponseWriter, r *http.Request) {
	routeID := extractItemID(r, "/gov/ui-routes")
	switch r.Method {
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "iam.ui.read")
		okJSON(w, map[string]string{"id": routeID, "message": "UIRoute 详情——待实现"})
	case http.MethodPut:
		_, _ = h.requireGovAuth(w, r, "iam.ui.write")
		okJSON(w, map[string]string{"id": routeID, "message": "UIRoute 更新——待实现"})
	case http.MethodDelete:
		_, _ = h.requireGovAuth(w, r, "iam.ui.write")
		okJSON(w, map[string]any{"deleted": true, "id": routeID})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handleUIActionBindings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		_, _ = h.requireGovAuth(w, r, "iam.ui.write")
		okJSON(w, map[string]string{"message": "UIActionBinding 创建——待实现"})
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "iam.ui.read")
		okJSON(w, map[string]string{"message": "UIActionBinding 列表——待实现"})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handleUIActionBindingItem(w http.ResponseWriter, r *http.Request) {
	bindingID := extractItemID(r, "/gov/ui-action-bindings")
	switch r.Method {
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "iam.ui.read")
		okJSON(w, map[string]string{"id": bindingID, "message": "UIActionBinding 详情——待实现"})
	case http.MethodPut:
		_, _ = h.requireGovAuth(w, r, "iam.ui.write")
		okJSON(w, map[string]string{"id": bindingID, "message": "UIActionBinding 更新——待实现"})
	case http.MethodDelete:
		_, _ = h.requireGovAuth(w, r, "iam.ui.write")
		okJSON(w, map[string]any{"deleted": true, "id": bindingID})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handleUIPermissionSnapshot(w http.ResponseWriter, r *http.Request) {
	_, _ = h.requireGovAuth(w, r, "")
	okJSON(w, map[string]string{"message": "UI 权限快照——待实现"})
}

// ── §10 Audit handlers ────────────────────────────────────────────────────

func (h *GovHandler) handleAuditEvents(w http.ResponseWriter, r *http.Request) {
	_, _ = h.requireGovAuth(w, r, "data.audit.read")
	okJSON(w, map[string]string{"message": "AuditEvent 列表——待实现"})
}

func (h *GovHandler) handleAuditEventItem(w http.ResponseWriter, r *http.Request) {
	_, _ = h.requireGovAuth(w, r, "data.audit.read")
	eventID := extractItemID(r, "/gov/audit-events")
	okJSON(w, map[string]string{"id": eventID, "message": "AuditEvent 详情——待实现"})
}

func (h *GovHandler) handleRequestLogs(w http.ResponseWriter, r *http.Request) {
	_, _ = h.requireGovAuth(w, r, "data.usage.read")
	okJSON(w, map[string]string{"message": "RequestLog 列表——待实现"})
}

func (h *GovHandler) handleRequestLogTrace(w http.ResponseWriter, r *http.Request) {
	_, _ = h.requireGovAuth(w, r, "data.usage.read")
	requestID := extractItemID(r, "/gov/request-logs")
	okJSON(w, map[string]string{"request_id": requestID, "message": "RequestLog 追踪——待实现"})
}

func (h *GovHandler) handleAuditChainAnchors(w http.ResponseWriter, r *http.Request) {
	_, _ = h.requireGovAuth(w, r, "data.audit.read")
	okJSON(w, map[string]string{"message": "AuditChainAnchor 列表——待实现"})
}

// ── §11 Dashboard handlers ────────────────────────────────────────────────

func (h *GovHandler) handleDashboard(w http.ResponseWriter, r *http.Request) {
	_, _ = h.requireGovAuth(w, r, "data.report.read")
	okJSON(w, map[string]string{"message": "仪表盘——待实现"})
}

func (h *GovHandler) handleSecurityReports(w http.ResponseWriter, r *http.Request) {
	_, _ = h.requireGovAuth(w, r, "data.report.read")
	okJSON(w, map[string]string{"message": "安全报表——待实现"})
}

func (h *GovHandler) handleTrace(w http.ResponseWriter, r *http.Request) {
	_, _ = h.requireGovAuth(w, r, "data.usage.read")
	okJSON(w, map[string]string{"message": "调用追踪——待实现"})
}
