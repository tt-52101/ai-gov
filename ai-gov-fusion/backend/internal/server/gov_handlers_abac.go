// Package server 治理 API handlers——ABAC/UI Permission/Audit/Dashboard 域。
// 全部注释使用中文，符合 AGENTS.md 铁律。
package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tokenhub/backend/internal/server/abac"
	"tokenhub/backend/internal/server/audit"
	"tokenhub/backend/internal/server/ui_permission"

	"gorm.io/gorm"
)

// ── GovGrant 直接四轴授权记录 ────────────────────────────────────────────

// GovGrant 直接四轴授权——用于一次性"用户 X 对资源 Y 做操作 Z"的快速授权。
// 存储于 grants 表，独立于 ABAC 策略引擎。
type GovGrant struct {
	ID            string    `json:"id" gorm:"type:text;primaryKey"`
	PrincipalType string    `json:"principal_type" gorm:"type:text;not null;index:idx_grants_principal"`
	PrincipalID   string    `json:"principal_id" gorm:"type:text;not null;index:idx_grants_principal"`
	Axis          string    `json:"axis" gorm:"type:text;not null;index:idx_grants_axis"`
	Action        string    `json:"action" gorm:"type:text;not null"`
	ResourceType  string    `json:"resource_type" gorm:"type:text;not null"`
	ResourceID    string    `json:"resource_id" gorm:"type:text;not null"`
	Effect        string    `json:"effect" gorm:"type:text;not null;default:'allow'"`
	Conditions    string    `json:"conditions,omitempty" gorm:"type:text"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 覆盖 GORM 默认表名。
func (GovGrant) TableName() string { return "grants" }

// ── 请求/响应类型 ────────────────────────────────────────────────────────

// GovRoleRequest 角色创建/更新请求体。
type GovRoleRequest struct {
	RoleCode    string   `json:"role_code"`
	RoleName    string   `json:"role_name"`
	Description string   `json:"description,omitempty"`
	IsSystem    bool     `json:"is_system"`
	Permissions []string `json:"permissions,omitempty"` // 操作编码列表（action_code）
}

// GovPolicyRequest 策略创建/更新请求体。
type GovPolicyRequest struct {
	PolicyCode     string         `json:"policy_code"`
	PolicyName     string         `json:"policy_name"`
	Description    string         `json:"description,omitempty"`
	Effect         string         `json:"effect"`
	Priority       int            `json:"priority"`
	IsSystem       bool           `json:"is_system"`
	ConditionsJSON map[string]any `json:"conditions_json"`
}

// GovSubjectRoleBindingRequest 主体角色绑定请求体。
type GovSubjectRoleBindingRequest struct {
	SubjectType  string     `json:"subject_type"`
	SubjectID    string     `json:"subject_id"`
	RoleID       string     `json:"role_id"`
	ScopePartyID *string    `json:"scope_party_id,omitempty"`
	ValidFrom    *time.Time `json:"valid_from,omitempty"`
	ValidUntil   *time.Time `json:"valid_until,omitempty"`
}

// GovGrantRequest 直接授权请求体。
type GovGrantRequest struct {
	PrincipalType string         `json:"principal_type"`
	PrincipalID   string         `json:"principal_id"`
	Axis          string         `json:"axis"`
	Action        string         `json:"action"`
	ResourceType  string         `json:"resource_type"`
	ResourceID    string         `json:"resource_id"`
	Effect        string         `json:"effect"`
	Conditions    map[string]any `json:"conditions,omitempty"`
}

// GovPolicyEvaluateRequest 策略模拟评估请求体。
type GovPolicyEvaluateRequest struct {
	SubjectUserID string `json:"subject_user_id"`
	ResourceType  string `json:"resource_type"`
	ResourceID    string `json:"resource_id"`
	Action        string `json:"action"`
}

// ── §8 ABAC handlers ──────────────────────────────────────────────────────

// handleActionCatalogs 操作目录列表——GET /gov/action-catalogs。
// 支持 ?axis=data|fund|iam|routing 按治理轴筛选。
func (h *GovHandler) handleActionCatalogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
		return
	}
	_, _ = h.requireGovAuth(w, r, "iam.role.read")

	db := h.deps.DB
	if db == nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
		return
	}

	axis := r.URL.Query().Get("axis")
	actions, err := abac.ListActions(r.Context(), db, axis)
	if err != nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "LIST_FAILED", sanitizeError(err)))
		return
	}
	okJSON(w, map[string]any{"data": actions, "total": len(actions)})
}

// handleRoles 角色列表/创建——POST/GET /gov/roles。
func (h *GovHandler) handleRoles(w http.ResponseWriter, r *http.Request) {
	db := h.deps.DB
	switch r.Method {
	case http.MethodPost:
		gctx, _ := h.requireGovAuth(w, r, "iam.role.write")
		if gctx == nil {
			return
		}
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		req, ok := readJSON[GovRoleRequest](w, r)
		if !ok {
			return
		}
		if strings.TrimSpace(req.RoleCode) == "" {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "role_code 为必填字段"))
			return
		}
		if strings.TrimSpace(req.RoleName) == "" {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "role_name 为必填字段"))
			return
		}

		role := &abac.SysRole{
			RoleCode:    strings.TrimSpace(req.RoleCode),
			RoleName:    strings.TrimSpace(req.RoleName),
			Description: strings.TrimSpace(req.Description),
			IsSystem:    req.IsSystem,
		}
		if err := abac.CreateRole(r.Context(), db, role); err != nil {
			writeError(w, r, NewHTTPError(http.StatusConflict, "CREATE_FAILED", sanitizeError(err)))
			return
		}

		// 处理权限授予：将 action_code 列表解析为 action ID 并批量授权。
		if len(req.Permissions) > 0 {
			actionIDs := resolveActionCodes(db, req.Permissions)
			if len(actionIDs) > 0 {
				if err := abac.GrantPermission(r.Context(), db, role.ID, actionIDs); err != nil {
					writeError(w, r, NewHTTPError(http.StatusInternalServerError, "GRANT_PERM_FAILED", sanitizeError(err)))
					return
				}
			}
		}

		slog.InfoContext(r.Context(), "Role 创建成功", "role_id", role.ID, "role_code", role.RoleCode, "actor", gctx.SubjectID)
		createdJSON(w, role)
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "iam.role.read")
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		roles, err := abac.ListRoles(r.Context(), db)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "LIST_FAILED", sanitizeError(err)))
			return
		}
		okJSON(w, map[string]any{"data": roles, "total": len(roles)})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handleRoleItem 单个角色操作——GET/PUT/DELETE /gov/roles/{id}。
func (h *GovHandler) handleRoleItem(w http.ResponseWriter, r *http.Request) {
	roleID := extractItemID(r, "/gov/roles")
	db := h.deps.DB
	switch r.Method {
	case http.MethodGet:
		_, _ = h.requireGovItemAuth(w, r, "iam.role.read", "role", roleID)
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		role, err := abac.GetRole(r.Context(), db, roleID)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusNotFound, "NOT_FOUND", sanitizeError(err)))
			return
		}
		// 附加权限列表。
		permissions, _ := abac.GetRolePermissions(r.Context(), db, roleID)
		resp := map[string]any{
			"id":          role.ID,
			"role_code":   role.RoleCode,
			"role_name":   role.RoleName,
			"description": role.Description,
			"is_system":   role.IsSystem,
			"permissions": permissions,
			"created_at":  role.CreatedAt,
			"updated_at":  role.UpdatedAt,
		}
		okJSON(w, resp)
	case http.MethodPut:
		gctx, _ := h.requireGovItemAuth(w, r, "iam.role.write", "role", roleID)
		if gctx == nil {
			return
		}
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		req, ok := readJSON[GovRoleRequest](w, r)
		if !ok {
			return
		}

		role := &abac.SysRole{
			ID:          roleID,
			RoleName:    strings.TrimSpace(req.RoleName),
			Description: strings.TrimSpace(req.Description),
		}
		if err := abac.UpdateRole(r.Context(), db, role); err != nil {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "UPDATE_FAILED", sanitizeError(err)))
			return
		}

		// 更新权限列表：先撤销旧权限，再授予新权限。
		if req.Permissions != nil {
			db.WithContext(r.Context()).Where("role_id = ?", roleID).Delete(&abac.SysRolePermission{})
			if len(req.Permissions) > 0 {
				actionIDs := resolveActionCodes(db, req.Permissions)
				if len(actionIDs) > 0 {
					if err := abac.GrantPermission(r.Context(), db, roleID, actionIDs); err != nil {
						writeError(w, r, NewHTTPError(http.StatusInternalServerError, "GRANT_PERM_FAILED", sanitizeError(err)))
						return
					}
				}
			}
		}

		slog.InfoContext(r.Context(), "Role 更新成功", "role_id", roleID, "actor", gctx.SubjectID)

		// 返回更新后的角色。
		updated, _ := abac.GetRole(r.Context(), db, roleID)
		if updated != nil {
			permissions, _ := abac.GetRolePermissions(r.Context(), db, roleID)
			okJSON(w, map[string]any{
				"id":          updated.ID,
				"role_code":   updated.RoleCode,
				"role_name":   updated.RoleName,
				"description": updated.Description,
				"is_system":   updated.IsSystem,
				"permissions": permissions,
				"created_at":  updated.CreatedAt,
				"updated_at":  updated.UpdatedAt,
			})
		} else {
			okJSON(w, map[string]string{"id": roleID, "message": "角色已更新"})
		}
	case http.MethodDelete:
		_, _ = h.requireGovItemAuth(w, r, "iam.role.write", "role", roleID)
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		if err := abac.DeleteRole(r.Context(), db, roleID); err != nil {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "DELETE_FAILED", sanitizeError(err)))
			return
		}
		okJSON(w, map[string]any{"deleted": true, "id": roleID})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handlePolicies 策略列表/创建——POST/GET /gov/policies。
func (h *GovHandler) handlePolicies(w http.ResponseWriter, r *http.Request) {
	db := h.deps.DB
	switch r.Method {
	case http.MethodPost:
		gctx, _ := h.requireGovAuth(w, r, "iam.policy.write")
		if gctx == nil {
			return
		}
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		req, ok := readJSON[GovPolicyRequest](w, r)
		if !ok {
			return
		}
		if strings.TrimSpace(req.PolicyCode) == "" {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "policy_code 为必填字段"))
			return
		}
		if strings.TrimSpace(req.PolicyName) == "" {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "policy_name 为必填字段"))
			return
		}

		// 序列化 conditions_json。
		condJSON := "{}"
		if req.ConditionsJSON != nil {
			b, _ := json.Marshal(req.ConditionsJSON)
			condJSON = string(b)
		}

		policy := &abac.SysAccessPolicy{
			PolicyCode:     strings.TrimSpace(req.PolicyCode),
			PolicyName:     strings.TrimSpace(req.PolicyName),
			Description:    strings.TrimSpace(req.Description),
			Effect:         req.Effect,
			Priority:       req.Priority,
			IsSystem:       req.IsSystem,
			ConditionsJSON: condJSON,
		}
		if err := abac.CreatePolicy(r.Context(), db, policy); err != nil {
			writeError(w, r, NewHTTPError(http.StatusConflict, "CREATE_FAILED", sanitizeError(err)))
			return
		}

		slog.InfoContext(r.Context(), "Policy 创建成功", "policy_id", policy.ID, "policy_code", policy.PolicyCode, "actor", gctx.SubjectID)
		createdJSON(w, policy)
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "iam.policy.read")
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		effect := r.URL.Query().Get("effect")
		policies, err := abac.ListPolicies(r.Context(), db, effect)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "LIST_FAILED", sanitizeError(err)))
			return
		}
		okJSON(w, map[string]any{"data": policies, "total": len(policies)})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handlePolicyItem 单个策略操作——GET/PUT/DELETE/POST(评估) /gov/policies/{id}。
// POST 用于 /gov/policies/{id}/evaluate 策略模拟评估。
func (h *GovHandler) handlePolicyItem(w http.ResponseWriter, r *http.Request) {
	policyID := extractItemID(r, "/gov/policies")
	// 处理 /gov/policies/{id}/evaluate 子路径。
	isEvaluate := strings.HasSuffix(policyID, "/evaluate")
	if isEvaluate {
		policyID = strings.TrimSuffix(policyID, "/evaluate")
	}

	db := h.deps.DB
	switch r.Method {
	case http.MethodGet:
		_, _ = h.requireGovItemAuth(w, r, "iam.policy.read", "policy", policyID)
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		policy, err := abac.GetPolicy(r.Context(), db, policyID)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusNotFound, "NOT_FOUND", sanitizeError(err)))
			return
		}
		okJSON(w, policy)
	case http.MethodPut:
		gctx, _ := h.requireGovItemAuth(w, r, "iam.policy.write", "policy", policyID)
		if gctx == nil {
			return
		}
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		req, ok := readJSON[GovPolicyRequest](w, r)
		if !ok {
			return
		}

		condJSON := "{}"
		if req.ConditionsJSON != nil {
			b, _ := json.Marshal(req.ConditionsJSON)
			condJSON = string(b)
		}

		policy := &abac.SysAccessPolicy{
			ID:             policyID,
			PolicyCode:     strings.TrimSpace(req.PolicyCode),
			PolicyName:     strings.TrimSpace(req.PolicyName),
			Description:    strings.TrimSpace(req.Description),
			Effect:         req.Effect,
			Priority:       req.Priority,
			ConditionsJSON: condJSON,
		}
		if err := abac.UpdatePolicy(r.Context(), db, policy); err != nil {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "UPDATE_FAILED", sanitizeError(err)))
			return
		}

		slog.InfoContext(r.Context(), "Policy 更新成功", "policy_id", policyID, "actor", gctx.SubjectID)
		updated, _ := abac.GetPolicy(r.Context(), db, policyID)
		if updated != nil {
			okJSON(w, updated)
		} else {
			okJSON(w, map[string]string{"id": policyID, "message": "策略已更新"})
		}
	case http.MethodDelete:
		_, _ = h.requireGovItemAuth(w, r, "iam.policy.write", "policy", policyID)
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		if err := abac.DeletePolicy(r.Context(), db, policyID); err != nil {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "DELETE_FAILED", sanitizeError(err)))
			return
		}
		okJSON(w, map[string]any{"deleted": true, "id": policyID})
	case http.MethodPost:
		// POST 用于策略模拟评估——/gov/policies/{id}/evaluate。
		// 若路径不含 /evaluate 后缀则视为不支持的操作。
		if !isEvaluate {
			writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "策略 POST 仅支持 /evaluate 子路径"))
			return
		}
		_, _ = h.requireGovItemAuth(w, r, "iam.policy.read", "policy", policyID)
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		req, ok := readJSON[GovPolicyEvaluateRequest](w, r)
		if !ok {
			return
		}

		subject := abac.Subject{Type: abac.SubjectTypeUser, ID: req.SubjectUserID}
		resource := abac.Resource{Type: req.ResourceType, ID: req.ResourceID}
		result, err := abac.EvaluatePolicy(r.Context(), db, subject, req.Action, resource)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "EVAL_FAILED", sanitizeError(err)))
			return
		}

		okJSON(w, map[string]any{
			"result":       result,
			"evaluated_at": time.Now().UTC().Format(time.RFC3339),
		})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handleSubjectRoleBindings 主体角色绑定列表/创建——POST/GET /gov/subject-role-bindings。
func (h *GovHandler) handleSubjectRoleBindings(w http.ResponseWriter, r *http.Request) {
	db := h.deps.DB
	switch r.Method {
	case http.MethodPost:
		gctx, _ := h.requireGovAuth(w, r, "iam.role.write")
		if gctx == nil {
			return
		}
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		req, ok := readJSON[GovSubjectRoleBindingRequest](w, r)
		if !ok {
			return
		}
		if strings.TrimSpace(req.SubjectType) == "" {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "subject_type 为必填字段"))
			return
		}
		if strings.TrimSpace(req.SubjectID) == "" {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "subject_id 为必填字段"))
			return
		}
		if strings.TrimSpace(req.RoleID) == "" {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "role_id 为必填字段"))
			return
		}

		if err := abac.AssignRole(r.Context(), db,
			strings.TrimSpace(req.SubjectType),
			strings.TrimSpace(req.SubjectID),
			strings.TrimSpace(req.RoleID),
			req.ScopePartyID,
			req.ValidFrom,
			req.ValidUntil,
		); err != nil {
			writeError(w, r, NewHTTPError(http.StatusConflict, "BIND_FAILED", sanitizeError(err)))
			return
		}

		slog.InfoContext(r.Context(), "SubjectRoleBinding 创建成功",
			"subject_type", req.SubjectType,
			"subject_id", req.SubjectID,
			"role_id", req.RoleID,
			"actor", gctx.SubjectID,
		)

		// 查询刚创建的绑定以返回完整信息。
		bindings, _ := abac.GetSubjectRoles(r.Context(), db, req.SubjectType, req.SubjectID)
		okJSON(w, map[string]any{"data": bindings, "message": "角色绑定成功"})
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "iam.role.read")
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		subjectType := r.URL.Query().Get("subject_type")
		subjectID := r.URL.Query().Get("subject_id")

		bindings, err := abac.GetSubjectRoles(r.Context(), db, subjectType, subjectID)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "LIST_FAILED", sanitizeError(err)))
			return
		}
		okJSON(w, map[string]any{"data": bindings, "total": len(bindings)})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handleSubjectRoleBindingItem 单个绑定删除——DELETE /gov/subject-role-bindings/{id}。
func (h *GovHandler) handleSubjectRoleBindingItem(w http.ResponseWriter, r *http.Request) {
	bindingID := extractItemID(r, "/gov/subject-role-bindings")
	switch r.Method {
	case http.MethodDelete:
		_, _ = h.requireGovItemAuth(w, r, "iam.role.write", "subject_role_binding", bindingID)
		db := h.deps.DB
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		if err := abac.RevokeRole(r.Context(), db, bindingID); err != nil {
			writeError(w, r, NewHTTPError(http.StatusNotFound, "REVOKE_FAILED", sanitizeError(err)))
			return
		}
		okJSON(w, map[string]any{"deleted": true, "id": bindingID})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handleGrants 直接授权列表/创建——POST/GET /gov/grants。
// GET 支持 ?axis=data|fund|iam|routing 按治理轴筛选。
func (h *GovHandler) handleGrants(w http.ResponseWriter, r *http.Request) {
	db := h.deps.DB
	switch r.Method {
	case http.MethodPost:
		gctx, _ := h.requireGovAuth(w, r, "iam.policy.write")
		if gctx == nil {
			return
		}
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		req, ok := readJSON[GovGrantRequest](w, r)
		if !ok {
			return
		}
		if strings.TrimSpace(req.PrincipalType) == "" {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "principal_type 为必填字段"))
			return
		}
		if strings.TrimSpace(req.PrincipalID) == "" {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_PARAM", "principal_id 为必填字段"))
			return
		}

		// 序列化 conditions。
		condStr := "{}"
		if req.Conditions != nil {
			b, _ := json.Marshal(req.Conditions)
			condStr = string(b)
		}

		grant := GovGrant{
			ID:            NewID("grant"),
			PrincipalType: strings.TrimSpace(req.PrincipalType),
			PrincipalID:   strings.TrimSpace(req.PrincipalID),
			Axis:          strings.TrimSpace(req.Axis),
			Action:        strings.TrimSpace(req.Action),
			ResourceType:  strings.TrimSpace(req.ResourceType),
			ResourceID:    strings.TrimSpace(req.ResourceID),
			Effect:        strings.TrimSpace(req.Effect),
			Conditions:    condStr,
			CreatedAt:     time.Now().UTC(),
		}
		if grant.Effect == "" {
			grant.Effect = "allow"
		}

		if err := db.WithContext(r.Context()).Create(&grant).Error; err != nil {
			writeError(w, r, NewHTTPError(http.StatusConflict, "CREATE_FAILED", sanitizeError(err)))
			return
		}

		slog.InfoContext(r.Context(), "Grant 创建成功", "grant_id", grant.ID, "axis", grant.Axis, "actor", gctx.SubjectID)
		createdJSON(w, grant)
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "iam.policy.read")
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		// 支持按 axis、principal_type、principal_id 筛选。
		query := db.WithContext(r.Context()).Model(&GovGrant{})
		if axis := r.URL.Query().Get("axis"); axis != "" {
			query = query.Where("axis = ?", axis)
		}
		if pt := r.URL.Query().Get("principal_type"); pt != "" {
			query = query.Where("principal_type = ?", pt)
		}
		if pid := r.URL.Query().Get("principal_id"); pid != "" {
			query = query.Where("principal_id = ?", pid)
		}

		var grants []GovGrant
		if err := query.Order("created_at DESC").Find(&grants).Error; err != nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "LIST_FAILED", sanitizeError(err)))
			return
		}
		okJSON(w, map[string]any{"data": grants, "total": len(grants)})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handleGrantItem 单个授权删除——DELETE /gov/grants/{id}。
func (h *GovHandler) handleGrantItem(w http.ResponseWriter, r *http.Request) {
	grantID := extractItemID(r, "/gov/grants")
	switch r.Method {
	case http.MethodDelete:
		_, _ = h.requireGovItemAuth(w, r, "iam.policy.write", "grant", grantID)
		db := h.deps.DB
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		result := db.WithContext(r.Context()).Delete(&GovGrant{}, "id = ?", grantID)
		if result.Error != nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DELETE_FAILED", sanitizeError(result.Error)))
			return
		}
		if result.RowsAffected == 0 {
			writeError(w, r, NewHTTPError(http.StatusNotFound, "NOT_FOUND", "授权记录不存在: "+grantID))
			return
		}
		okJSON(w, map[string]any{"deleted": true, "id": grantID})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// ── §9 UI Permission handlers ─────────────────────────────────────────────

// handleUIMenus UI 菜单列表/创建——POST/GET /gov/ui-menus。
func (h *GovHandler) handleUIMenus(w http.ResponseWriter, r *http.Request) {
	db := h.deps.DB
	switch r.Method {
	case http.MethodPost:
		_, _ = h.requireGovAuth(w, r, "iam.ui.write")
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		req, ok := readJSON[ui_permission.CreateMenuRequest](w, r)
		if !ok {
			return
		}

		menu, err := ui_permission.CreateMenu(db, req)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusConflict, "CREATE_FAILED", sanitizeError(err)))
			return
		}
		createdJSON(w, menu)
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "iam.ui.read")
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		menus, err := ui_permission.ListMenus(db)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "LIST_FAILED", sanitizeError(err)))
			return
		}
		okJSON(w, map[string]any{"data": menus, "total": len(menus)})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handleUIMenuItem 单个菜单操作——GET/PUT/DELETE /gov/ui-menus/{id}。
func (h *GovHandler) handleUIMenuItem(w http.ResponseWriter, r *http.Request) {
	menuIDStr := extractItemID(r, "/gov/ui-menus")
	menuID, err := strconv.ParseInt(menuIDStr, 10, 64)
	if err != nil {
		writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_ID", "无效的菜单 ID: "+menuIDStr))
		return
	}

	db := h.deps.DB
	switch r.Method {
	case http.MethodGet:
		_, _ = h.requireGovItemAuth(w, r, "iam.ui.read", "ui_menu", menuIDStr)
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		menu, err := ui_permission.GetMenu(db, menuID)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusNotFound, "NOT_FOUND", sanitizeError(err)))
			return
		}
		okJSON(w, menu)
	case http.MethodPut:
		_, _ = h.requireGovItemAuth(w, r, "iam.ui.write", "ui_menu", menuIDStr)
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		req, ok := readJSON[ui_permission.CreateMenuRequest](w, r)
		if !ok {
			return
		}

		menu, err := ui_permission.UpdateMenu(db, menuID, req)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "UPDATE_FAILED", sanitizeError(err)))
			return
		}
		okJSON(w, menu)
	case http.MethodDelete:
		_, _ = h.requireGovItemAuth(w, r, "iam.ui.write", "ui_menu", menuIDStr)
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		if err := ui_permission.DeleteMenu(db, menuID); err != nil {
			writeError(w, r, NewHTTPError(http.StatusNotFound, "DELETE_FAILED", sanitizeError(err)))
			return
		}
		okJSON(w, map[string]any{"deleted": true, "id": menuID})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handleUIRoutes UI 路由列表/创建——POST/GET /gov/ui-routes。
func (h *GovHandler) handleUIRoutes(w http.ResponseWriter, r *http.Request) {
	db := h.deps.DB
	switch r.Method {
	case http.MethodPost:
		_, _ = h.requireGovAuth(w, r, "iam.ui.write")
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		req, ok := readJSON[ui_permission.CreateRouteRequest](w, r)
		if !ok {
			return
		}

		route, err := ui_permission.CreateRoute(db, req)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusConflict, "CREATE_FAILED", sanitizeError(err)))
			return
		}
		createdJSON(w, route)
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "iam.ui.read")
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		routes, err := ui_permission.ListRoutes(db)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "LIST_FAILED", sanitizeError(err)))
			return
		}
		okJSON(w, map[string]any{"data": routes, "total": len(routes)})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handleUIRouteItem 单个路由操作——GET/PUT/DELETE /gov/ui-routes/{id}。
func (h *GovHandler) handleUIRouteItem(w http.ResponseWriter, r *http.Request) {
	routeIDStr := extractItemID(r, "/gov/ui-routes")
	routeID, err := strconv.ParseInt(routeIDStr, 10, 64)
	if err != nil {
		writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_ID", "无效的路由 ID: "+routeIDStr))
		return
	}

	db := h.deps.DB
	switch r.Method {
	case http.MethodGet:
		_, _ = h.requireGovItemAuth(w, r, "iam.ui.read", "ui_route", routeIDStr)
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		route, err := ui_permission.GetRoute(db, routeID)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusNotFound, "NOT_FOUND", sanitizeError(err)))
			return
		}
		okJSON(w, route)
	case http.MethodPut:
		_, _ = h.requireGovItemAuth(w, r, "iam.ui.write", "ui_route", routeIDStr)
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		req, ok := readJSON[ui_permission.CreateRouteRequest](w, r)
		if !ok {
			return
		}

		route, err := ui_permission.UpdateRoute(db, routeID, req)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "UPDATE_FAILED", sanitizeError(err)))
			return
		}
		okJSON(w, route)
	case http.MethodDelete:
		_, _ = h.requireGovItemAuth(w, r, "iam.ui.write", "ui_route", routeIDStr)
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		if err := ui_permission.DeleteRoute(db, routeID); err != nil {
			writeError(w, r, NewHTTPError(http.StatusNotFound, "DELETE_FAILED", sanitizeError(err)))
			return
		}
		okJSON(w, map[string]any{"deleted": true, "id": routeID})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handleUIActionBindings UI 按钮绑定列表/创建——POST/GET /gov/ui-action-bindings。
func (h *GovHandler) handleUIActionBindings(w http.ResponseWriter, r *http.Request) {
	db := h.deps.DB
	switch r.Method {
	case http.MethodPost:
		_, _ = h.requireGovAuth(w, r, "iam.ui.write")
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		req, ok := readJSON[ui_permission.CreateActionBindingRequest](w, r)
		if !ok {
			return
		}

		binding, err := ui_permission.CreateActionBinding(db, req)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusConflict, "CREATE_FAILED", sanitizeError(err)))
			return
		}
		createdJSON(w, binding)
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "iam.ui.read")
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		bindings, err := ui_permission.ListAllActionBindings(db)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "LIST_FAILED", sanitizeError(err)))
			return
		}
		okJSON(w, map[string]any{"data": bindings, "total": len(bindings)})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handleUIActionBindingItem 单个按钮绑定操作——GET/PUT/DELETE /gov/ui-action-bindings/{id}。
func (h *GovHandler) handleUIActionBindingItem(w http.ResponseWriter, r *http.Request) {
	bindingIDStr := extractItemID(r, "/gov/ui-action-bindings")
	bindingID, err := strconv.ParseInt(bindingIDStr, 10, 64)
	if err != nil {
		writeError(w, r, NewHTTPError(http.StatusBadRequest, "INVALID_ID", "无效的按钮绑定 ID: "+bindingIDStr))
		return
	}

	db := h.deps.DB
	switch r.Method {
	case http.MethodGet:
		_, _ = h.requireGovItemAuth(w, r, "iam.ui.read", "ui_action_binding", bindingIDStr)
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		binding, err := ui_permission.GetActionBinding(db, bindingID)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusNotFound, "NOT_FOUND", sanitizeError(err)))
			return
		}
		okJSON(w, binding)
	case http.MethodPut:
		_, _ = h.requireGovItemAuth(w, r, "iam.ui.write", "ui_action_binding", bindingIDStr)
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		req, ok := readJSON[ui_permission.CreateActionBindingRequest](w, r)
		if !ok {
			return
		}

		binding, err := ui_permission.UpdateActionBinding(db, bindingID, req)
		if err != nil {
			writeError(w, r, NewHTTPError(http.StatusBadRequest, "UPDATE_FAILED", sanitizeError(err)))
			return
		}
		okJSON(w, binding)
	case http.MethodDelete:
		_, _ = h.requireGovItemAuth(w, r, "iam.ui.write", "ui_action_binding", bindingIDStr)
		if db == nil {
			writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
			return
		}

		if err := ui_permission.DeleteActionBinding(db, bindingID); err != nil {
			writeError(w, r, NewHTTPError(http.StatusNotFound, "DELETE_FAILED", sanitizeError(err)))
			return
		}
		okJSON(w, map[string]any{"deleted": true, "id": bindingID})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

// handleUIPermissionSnapshot UI 权限快照——GET /gov/ui-permissions/snapshot。
// 返回当前主体可访问的菜单树和路由列表。
func (h *GovHandler) handleUIPermissionSnapshot(w http.ResponseWriter, r *http.Request) {
	gctx, _ := h.requireGovAuth(w, r, "")
	if gctx == nil {
		return
	}

	if h.deps.UIPermProjector == nil {
		writeError(w, r, NewHTTPError(501, "NOT_IMPLEMENTED", "UI 权限投影器未配置"))
		return
	}

	subject := ui_permission.Subject{Type: gctx.SubjectType, ID: gctx.SubjectID}

	// 投影菜单树。
	menus, err := h.deps.UIPermProjector.ProjectMenus(r.Context(), subject)
	if err != nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "PROJECT_FAILED", sanitizeError(err)))
		return
	}

	// 投影可访问路由。
	routes, err := h.deps.UIPermProjector.ProjectRoutes(r.Context(), subject)
	if err != nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "PROJECT_FAILED", sanitizeError(err)))
		return
	}

	okJSON(w, map[string]any{
		"menus":  menus,
		"routes": routes,
	})
}

// ── §10 Audit handlers ────────────────────────────────────────────────────

// handleAuditEvents 审计事件检索——GET /gov/audit-events。
// 支持 ?actor/action/resource_type/start_time/end_time 查询参数。
func (h *GovHandler) handleAuditEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
		return
	}
	_, _ = h.requireGovAuth(w, r, "data.audit.read")

	db := h.deps.DB
	if db == nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
		return
	}

	// 解析查询参数，构建审计过滤条件。
	filter := audit.AuditFilter{
		Offset: 0,
		Limit:  200,
	}

	if actor := r.URL.Query().Get("actor"); actor != "" {
		filter.ActorUserID = actor
	}
	if action := r.URL.Query().Get("action"); action != "" {
		filter.Action = action
	}
	if resourceType := r.URL.Query().Get("resource_type"); resourceType != "" {
		filter.ResourceType = resourceType
	}
	if resourceID := r.URL.Query().Get("resource_id"); resourceID != "" {
		filter.ResourceID = resourceID
	}

	// 解析时间范围参数（ISO 8601 / RFC 3339 格式）。
	if startStr := r.URL.Query().Get("start_time"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			filter.StartTime = &t
		}
	}
	if endStr := r.URL.Query().Get("end_time"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			filter.EndTime = &t
		}
	}

	events, total, err := audit.SearchEvents(r.Context(), db, filter)
	if err != nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "SEARCH_FAILED", sanitizeError(err)))
		return
	}

	okJSON(w, map[string]any{
		"data":  events,
		"total": total,
	})
}

// handleAuditEventItem 单条审计事件详情——GET /gov/audit-events/{id}。
func (h *GovHandler) handleAuditEventItem(w http.ResponseWriter, r *http.Request) {
	eventID := extractItemID(r, "/gov/audit-events")
	if r.Method != http.MethodGet {
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
		return
	}
	_, _ = h.requireGovItemAuth(w, r, "data.audit.read", "audit_event", eventID)

	db := h.deps.DB
	if db == nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "DB_UNAVAILABLE", "数据库未配置"))
		return
	}

	event, err := audit.GetEvent(r.Context(), db, eventID)
	if err != nil {
		writeError(w, r, NewHTTPError(http.StatusInternalServerError, "GET_FAILED", sanitizeError(err)))
		return
	}
	if event == nil {
		writeError(w, r, NewHTTPError(http.StatusNotFound, "NOT_FOUND", "审计事件不存在: "+eventID))
		return
	}

	okJSON(w, event)
}

// handleRequestLogs 请求日志列表——GET /gov/request-logs（占位）。
func (h *GovHandler) handleRequestLogs(w http.ResponseWriter, r *http.Request) {
	_, _ = h.requireGovAuth(w, r, "data.usage.read")
	okJSON(w, map[string]string{"message": "RequestLog 列表——待实现"})
}

// handleRequestLogTrace 请求日志追踪——GET /gov/request-logs/{id}（占位）。
func (h *GovHandler) handleRequestLogTrace(w http.ResponseWriter, r *http.Request) {
	requestID := extractItemID(r, "/gov/request-logs")
	_, _ = h.requireGovItemAuth(w, r, "data.usage.read", "request_log", requestID)
	okJSON(w, map[string]string{"request_id": requestID, "message": "RequestLog 追踪——待实现"})
}

// handleAuditChainAnchors 审计哈希链锚点列表——GET /gov/audit-chain-anchors（占位）。
func (h *GovHandler) handleAuditChainAnchors(w http.ResponseWriter, r *http.Request) {
	_, _ = h.requireGovAuth(w, r, "data.audit.read")
	okJSON(w, map[string]string{"message": "AuditChainAnchor 列表——待实现"})
}

// ── §10.5 Reconciliation handlers（对账——阶段 D 实现）─────────────────────

func (h *GovHandler) handleReconciliationRuns(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		_, _ = h.requireGovAuth(w, r, "data.audit.write")
		// 阶段 D 实现：调用 ReconciliationService.StartRun(ctx, periodStart, periodEnd)。
		// 请求体应包含 period_start 和 period_end（ISO 8601 格式）。
		// 创建成功后记录审计事件——before_snapshot 为空, after_snapshot 为新建的 ReconciliationRun JSON。
		okJSON(w, map[string]string{"message": "对账运行创建——阶段 D 实现"})
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "data.audit.read")
		// 阶段 D 实现：调用 reconciliation.ListRuns(ctx, db, limit, offset)。
		okJSON(w, map[string]string{"message": "对账运行列表——阶段 D 实现"})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
}

func (h *GovHandler) handleReconciliationRunItem(w http.ResponseWriter, r *http.Request) {
	runID := extractItemID(r, "/gov/reconciliation-runs")
	switch r.Method {
	case http.MethodGet:
		_, _ = h.requireGovAuth(w, r, "data.audit.read")
		// 阶段 D 实现：调用 reconciliation.GetRun(ctx, db, runID)。
		okJSON(w, map[string]string{"run_id": runID, "message": "对账运行详情——阶段 D 实现"})
	default:
		writeError(w, r, NewHTTPError(405, "METHOD_NOT_ALLOWED", "不支持的 HTTP 方法"))
	}
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

// ── 辅助函数 ──────────────────────────────────────────────────────────────

// resolveActionCodes 将操作编码（action_code）列表解析为操作记录 ID 列表。
// 用于角色权限创建时将请求中的 action_code 转换为 sys_action_catalogs 中的记录 ID。
func resolveActionCodes(db *gorm.DB, codes []string) []string {
	if len(codes) == 0 || db == nil {
		return nil
	}
	var catalogs []abac.SysActionCatalog
	if err := db.Where("action_code IN ?", codes).Find(&catalogs).Error; err != nil {
		return nil
	}
	ids := make([]string, 0, len(catalogs))
	for _, c := range catalogs {
		ids = append(ids, c.ID)
	}
	return ids
}
