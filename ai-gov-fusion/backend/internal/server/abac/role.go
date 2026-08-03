package abac

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// ── 预定义错误 ──────────────────────────────────────────────────────────

// ErrSystemRole 表示尝试删除系统内置角色。
var ErrSystemRole = errors.New("系统内置角色不可删除")

// ── 角色 CRUD ───────────────────────────────────────────────────────────

// CreateRole 创建新的角色定义。
// 角色编码必须唯一；is_system=true 的角色不可删除。
// 返回创建后的角色记录（ID 已填充）。
func CreateRole(ctx context.Context, db *gorm.DB, r *SysRole) error {
	if r == nil {
		return errors.New("abac: 不能创建 nil 角色")
	}
	if r.RoleCode == "" {
		return errors.New("abac: 角色编码不能为空")
	}
	if r.RoleName == "" {
		return errors.New("abac: 角色名称不能为空")
	}
	if r.ID == "" {
		r.ID = NewID()
	}

	if err := db.WithContext(ctx).Create(r).Error; err != nil {
		slog.ErrorContext(ctx, "创建角色失败",
			"role_code", r.RoleCode,
			"error", err,
		)
		return fmt.Errorf("abac: 创建角色失败: %w", err)
	}

	slog.InfoContext(ctx, "创建角色成功",
		"role_id", r.ID,
		"role_code", r.RoleCode,
		"role_name", r.RoleName,
		"is_system", r.IsSystem,
	)
	return nil
}

// GetRole 按 ID 查询角色。
func GetRole(ctx context.Context, db *gorm.DB, id string) (*SysRole, error) {
	var r SysRole
	if err := db.WithContext(ctx).First(&r, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("abac: 角色 %s 不存在", id)
		}
		return nil, fmt.Errorf("abac: 查询角色失败: %w", err)
	}
	return &r, nil
}

// GetRoleByCode 按角色编码查询角色。
func GetRoleByCode(ctx context.Context, db *gorm.DB, code string) (*SysRole, error) {
	var r SysRole
	if err := db.WithContext(ctx).First(&r, "role_code = ?", code).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("abac: 角色编码 %s 不存在", code)
		}
		return nil, fmt.Errorf("abac: 查询角色失败: %w", err)
	}
	return &r, nil
}

// UpdateRole 更新角色的可变字段（名称、描述）。
// 系统角色（is_system=true）仅允许更新名称和描述，不允许修改编码。
func UpdateRole(ctx context.Context, db *gorm.DB, r *SysRole) error {
	if r == nil || r.ID == "" {
		return errors.New("abac: 角色 ID 不能为空")
	}

	updates := map[string]any{
		"role_name":   r.RoleName,
		"description": r.Description,
	}

	if err := db.WithContext(ctx).Model(&SysRole{}).Where("id = ?", r.ID).Updates(updates).Error; err != nil {
		slog.ErrorContext(ctx, "更新角色失败", "role_id", r.ID, "error", err)
		return fmt.Errorf("abac: 更新角色失败: %w", err)
	}

	slog.InfoContext(ctx, "更新角色成功", "role_id", r.ID)
	return nil
}

// DeleteRole 删除角色。
// 系统角色（is_system=true）禁止删除。
// 删除角色时会级联删除关联的权限映射和主体绑定（数据库 ON DELETE CASCADE）。
func DeleteRole(ctx context.Context, db *gorm.DB, id string) error {
	existing, err := GetRole(ctx, db, id)
	if err != nil {
		return err
	}
	if existing.IsSystem {
		return fmt.Errorf("abac: %w: %s", ErrSystemRole, existing.RoleCode)
	}

	if err := db.WithContext(ctx).Delete(&SysRole{}, "id = ?", id).Error; err != nil {
		slog.ErrorContext(ctx, "删除角色失败", "role_id", id, "error", err)
		return fmt.Errorf("abac: 删除角色失败: %w", err)
	}

	slog.InfoContext(ctx, "删除角色成功", "role_id", id, "role_code", existing.RoleCode)
	return nil
}

// ListRoles 列出所有角色。
func ListRoles(ctx context.Context, db *gorm.DB) ([]SysRole, error) {
	var roles []SysRole
	if err := db.WithContext(ctx).Order("created_at ASC").Find(&roles).Error; err != nil {
		return nil, fmt.Errorf("abac: 查询角色列表失败: %w", err)
	}
	return roles, nil
}

// ── 角色权限映射 ────────────────────────────────────────────────────────

// GrantPermission 为角色授予指定操作的权限。
// actionIDs 为操作记录 ID 列表（来自 sys_action_catalogs）。
// 对每个 (role_id, action_id) 组合执行幂等插入——已存在则跳过。
func GrantPermission(ctx context.Context, db *gorm.DB, roleID string, actionIDs []string) error {
	if roleID == "" {
		return errors.New("abac: role_id 不能为空")
	}
	if len(actionIDs) == 0 {
		return errors.New("abac: action_ids 不能为空")
	}

	for _, actionID := range actionIDs {
		rp := &SysRolePermission{
			ID:       NewID(),
			RoleID:   roleID,
			ActionID: actionID,
		}
		// 忽略唯一约束冲突（已存在的权限）。
		if err := db.WithContext(ctx).Create(rp).Error; err != nil {
			// 检查是否为唯一约束冲突，若是则跳过。
			if !isUniqueViolation(err) {
				slog.ErrorContext(ctx, "授予权限失败",
					"role_id", roleID,
					"action_id", actionID,
					"error", err,
				)
				return fmt.Errorf("abac: 授予权限失败: %w", err)
			}
		}
	}

	slog.InfoContext(ctx, "授予权限成功",
		"role_id", roleID,
		"action_count", len(actionIDs),
	)
	return nil
}

// RevokePermission 撤销角色的指定操作权限。
// actionIDs 为操作记录 ID 列表。
func RevokePermission(ctx context.Context, db *gorm.DB, roleID string, actionIDs []string) error {
	if roleID == "" || len(actionIDs) == 0 {
		return errors.New("abac: role_id 和 action_ids 均不能为空")
	}

	result := db.WithContext(ctx).
		Where("role_id = ? AND action_id IN ?", roleID, actionIDs).
		Delete(&SysRolePermission{})
	if result.Error != nil {
		return fmt.Errorf("abac: 撤销权限失败: %w", result.Error)
	}

	slog.InfoContext(ctx, "撤销权限成功",
		"role_id", roleID,
		"action_count", len(actionIDs),
		"deleted_rows", result.RowsAffected,
	)
	return nil
}

// ListRolePermissions 列出角色的所有权限映射（含操作编码和名称）。
type RolePermissionRow struct {
	PermissionID string `json:"permission_id"`
	ActionID     string `json:"action_id"`
	ActionCode   string `json:"action_code"`
	ActionName   string `json:"action_name"`
	Axis         string `json:"axis"`
}

// GetRolePermissions 返回角色的所有权限，含关联的操作目录信息。
func GetRolePermissions(ctx context.Context, db *gorm.DB, roleID string) ([]RolePermissionRow, error) {
	var rows []RolePermissionRow
	if err := db.WithContext(ctx).
		Table("sys_role_permissions rp").
		Select("rp.id AS permission_id, ac.id AS action_id, ac.action_code, ac.action_name, ac.axis").
		Joins("JOIN sys_action_catalogs ac ON ac.id = rp.action_id").
		Where("rp.role_id = ?", roleID).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("abac: 查询角色权限失败: %w", err)
	}
	return rows, nil
}

// ── 主体角色绑定 ──────────────────────────────────────────────────────

// AssignRole 将角色分配给主体，创建 subject-role 绑定。
// scopePartyID 为 NULL 表示全局角色；指定后角色仅在对应 Party 及子级生效。
// validFrom 为 NULL 表示立即生效；validUntil 为 NULL 表示永久有效。
func AssignRole(ctx context.Context, db *gorm.DB, subjectType, subjectID, roleID string, scopePartyID *string, validFrom, validUntil *time.Time) error {
	if subjectType == "" || subjectID == "" || roleID == "" {
		return errors.New("abac: subject_type、subject_id、role_id 均不能为空")
	}

	binding := &SysSubjectRoleBinding{
		ID:           NewID(),
		SubjectType:  subjectType,
		SubjectID:    subjectID,
		RoleID:       roleID,
		ScopePartyID: scopePartyID,
		ValidFrom:    validFrom,
		ValidUntil:   validUntil,
	}

	if err := db.WithContext(ctx).Create(binding).Error; err != nil {
		slog.ErrorContext(ctx, "分配角色失败",
			"subject_type", subjectType,
			"subject_id", subjectID,
			"role_id", roleID,
			"error", err,
		)
		return fmt.Errorf("abac: 分配角色失败: %w", err)
	}

	slog.InfoContext(ctx, "分配角色成功",
		"binding_id", binding.ID,
		"subject_type", subjectType,
		"subject_id", subjectID,
		"role_id", roleID,
	)
	return nil
}

// RevokeRole 撤销主体的角色绑定（按绑定 ID）。
func RevokeRole(ctx context.Context, db *gorm.DB, bindingID string) error {
	result := db.WithContext(ctx).Delete(&SysSubjectRoleBinding{}, "id = ?", bindingID)
	if result.Error != nil {
		return fmt.Errorf("abac: 撤销角色失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("abac: 角色绑定 %s 不存在", bindingID)
	}

	slog.InfoContext(ctx, "撤销角色成功", "binding_id", bindingID)
	return nil
}

// ListSubjectRoles 列出指定主体的所有角色绑定（含角色信息）。
type SubjectRoleRow struct {
	BindingID    string     `json:"binding_id"`
	RoleID       string     `json:"role_id"`
	RoleCode     string     `json:"role_code"`
	RoleName     string     `json:"role_name"`
	ScopePartyID *string    `json:"scope_party_id,omitempty"`
	ValidFrom    *time.Time `json:"valid_from,omitempty"`
	ValidUntil   *time.Time `json:"valid_until,omitempty"`
}

// GetSubjectRoles 返回主体的所有角色绑定，含关联的角色信息。
func GetSubjectRoles(ctx context.Context, db *gorm.DB, subjectType, subjectID string) ([]SubjectRoleRow, error) {
	var rows []SubjectRoleRow
	if err := db.WithContext(ctx).
		Table("sys_subject_role_bindings srb").
		Select("srb.id AS binding_id, srb.role_id, r.role_code, r.role_name, srb.scope_party_id, srb.valid_from, srb.valid_until").
		Joins("JOIN sys_roles r ON r.id = srb.role_id").
		Where("srb.subject_type = ? AND srb.subject_id = ?", subjectType, subjectID).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("abac: 查询主体角色失败: %w", err)
	}
	return rows, nil
}

// ── 操作目录管理 ──────────────────────────────────────────────────────

// CreateAction 注册新的原子操作。
func CreateAction(ctx context.Context, db *gorm.DB, a *SysActionCatalog) error {
	if a == nil {
		return errors.New("abac: 不能创建 nil 操作")
	}
	if a.ActionCode == "" || a.ActionName == "" || a.Axis == "" {
		return errors.New("abac: action_code、action_name、axis 均不能为空")
	}
	if a.ID == "" {
		a.ID = NewID()
	}

	if err := db.WithContext(ctx).Create(a).Error; err != nil {
		return fmt.Errorf("abac: 创建操作失败: %w", err)
	}

	slog.InfoContext(ctx, "注册操作成功",
		"action_id", a.ID,
		"action_code", a.ActionCode,
		"axis", a.Axis,
	)
	return nil
}

// ListActions 列出所有已注册的原子操作，可按轴筛选。
func ListActions(ctx context.Context, db *gorm.DB, axis string) ([]SysActionCatalog, error) {
	var actions []SysActionCatalog
	q := db.WithContext(ctx).Order("axis, action_code")
	if axis != "" {
		q = q.Where("axis = ?", axis)
	}
	if err := q.Find(&actions).Error; err != nil {
		return nil, fmt.Errorf("abac: 查询操作列表失败: %w", err)
	}
	return actions, nil
}

// ── 辅助函数 ──────────────────────────────────────────────────────────

// isUniqueViolation 检查 GORM 错误是否为唯一约束冲突。
// SQLite 和 PostgreSQL 的唯一约束错误信息不同，通过字符串匹配检测。
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "UNIQUE constraint failed") ||
		contains(msg, "duplicate key") ||
		contains(msg, "duplicated key") ||
		contains(msg, "Duplicate entry")
}

// contains 检查 s 是否包含 substr（简单子串匹配）。
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstr(s, substr)
}

// findSubstr 在 s 中查找 substr。
func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
