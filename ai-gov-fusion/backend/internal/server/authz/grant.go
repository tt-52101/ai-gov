package authz

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ── Grant CRUD ──────────────────────────────────────────────────────────────

// CreateGrant 插入一条新授权记录。
// 调用方负责填充 Grant 结构体的必填字段（PrincipalType、PrincipalID、Axis、Action、Effect）。
// 若 Effect 为空则默认设为 allow。返回已赋 ID 的记录。
func CreateGrant(db *gorm.DB, g *Grant) error {
	if g == nil {
		return errors.New("authz: 不能创建 nil 授权记录")
	}
	if g.PrincipalType == "" {
		return errors.New("authz: principal_type 为必填项")
	}
	if g.PrincipalID == "" {
		return errors.New("authz: principal_id 为必填项")
	}
	if g.Axis == "" {
		return errors.New("authz: axis 为必填项")
	}
	if g.Action == "" {
		return errors.New("authz: action 为必填项")
	}
	if g.Effect == "" {
		g.Effect = EffectAllow
	}
	if g.Effect != EffectAllow && g.Effect != EffectDeny {
		return fmt.Errorf("authz: 无效的 effect %q，必须是 %q 或 %q", g.Effect, EffectAllow, EffectDeny)
	}
	return db.Create(g).Error
}

// GetGrant 按 ID 查询单条授权记录。
// 若记录不存在则返回 nil 与错误。
func GetGrant(db *gorm.DB, id string) (*Grant, error) {
	var g Grant
	if err := db.First(&g, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("authz: 授权记录 %s 未找到", id)
		}
		return nil, err
	}
	return &g, nil
}

// DeleteGrant 按 ID 删除一条授权记录。
// 若记录不存在则返回错误。
func DeleteGrant(db *gorm.DB, id string) error {
	result := db.Delete(&Grant{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("authz: 授权记录 %s 未找到", id)
	}
	return nil
}

// ListGrants 按主体与轴筛选授权记录。
// principalType 与 axis 为空字符串表示不做该维度筛选。
// 返回按创建时间升序排列的切片。
func ListGrants(db *gorm.DB, principalType, axis string) ([]*Grant, error) {
	var grants []*Grant
	q := db.Order("created_at ASC")
	if principalType != "" {
		q = q.Where("principal_type = ?", principalType)
	}
	if axis != "" {
		q = q.Where("axis = ?", axis)
	}
	if err := q.Find(&grants).Error; err != nil {
		return nil, err
	}
	return grants, nil
}

// ── 权限评估 ────────────────────────────────────────────────────────────────

// Evaluate 对 grants 表做主体、轴、操作的三元组匹配，判断是否允许。
// 规则：DENY 优先于 ALLOW。若同时存在匹配的 deny 与 allow，返回 deny。
// 若未匹配到任何规则，返回默认拒绝（A-CON-02）。
//
// 返回值：
//   - (true, nil)  表示允许
//   - (false, nil) 表示拒绝
//   - (false, err) 表示查询异常
func Evaluate(db *gorm.DB, principalType, principalID, axis, action string) (bool, error) {
	// 先查 DENY——任意一条匹配即立即拒绝。
	var denyCount int64
	if err := db.Model(&Grant{}).
		Where("principal_type = ? AND principal_id = ? AND axis = ? AND action = ? AND effect = ?",
			principalType, principalID, axis, action, EffectDeny).
		Count(&denyCount).Error; err != nil {
		return false, err
	}
	if denyCount > 0 {
		return false, nil
	}

	// 再查 ALLOW。
	var allowCount int64
	if err := db.Model(&Grant{}).
		Where("principal_type = ? AND principal_id = ? AND axis = ? AND action = ? AND effect = ?",
			principalType, principalID, axis, action, EffectAllow).
		Count(&allowCount).Error; err != nil {
		return false, err
	}
	return allowCount > 0, nil
}

// ── 迁移 ────────────────────────────────────────────────────────────────────

// Migrate 对 grants 表执行 GORM AutoMigrate。
// 由 store.go 在阶段 2 迁移中调用。
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&Grant{}); err != nil {
		return fmt.Errorf("authz 迁移失败: %w", err)
	}
	return nil
}
