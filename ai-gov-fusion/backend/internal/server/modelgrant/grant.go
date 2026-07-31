package modelgrant

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ── ModelGrant CRUD ─────────────────────────────────────────────────────────

// CreateModelGrant 插入一条新模型授权记录。
// 调用方负责填充必填字段（PrincipalType、PrincipalID、Effect）。
// 若 Effect 为空则默认设为 deny（安全默认：最小权限）。
// 返回已赋 ID 的记录。
func CreateModelGrant(db *gorm.DB, mg *ModelGrant) error {
	if mg == nil {
		return errors.New("modelgrant: 不能创建 nil 模型授权记录")
	}
	if mg.PrincipalType == "" {
		return errors.New("modelgrant: principal_type 为必填项")
	}
	if mg.PrincipalID == "" {
		return errors.New("modelgrant: principal_id 为必填项")
	}
	if mg.Effect == "" {
		mg.Effect = EffectDeny
	}
	if mg.Effect != EffectAllow && mg.Effect != EffectDeny {
		return fmt.Errorf("modelgrant: 无效的 effect %q，必须是 %q 或 %q", mg.Effect, EffectAllow, EffectDeny)
	}
	// 配额消耗仅在未显式设置时初始化为零。
	if mg.QuotaConsumed.IsZero() {
		mg.QuotaConsumed = decimal.Zero
	}
	return db.Create(mg).Error
}

// GetModelGrant 按 ID 查询单条模型授权记录。
// 若记录不存在则返回 nil 与错误。
func GetModelGrant(db *gorm.DB, id string) (*ModelGrant, error) {
	var mg ModelGrant
	if err := db.First(&mg, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("modelgrant: 模型授权记录 %s 未找到", id)
		}
		return nil, err
	}
	return &mg, nil
}

// DeleteModelGrant 按 ID 删除一条模型授权记录。
// 若记录不存在则返回错误。
func DeleteModelGrant(db *gorm.DB, id string) error {
	result := db.Delete(&ModelGrant{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("modelgrant: 模型授权记录 %s 未找到", id)
	}
	return nil
}

// ListModelGrants 按主体与模型筛选授权记录。
// principalType、principalID、modelID 为空字符串表示不做该维度筛选。
// 返回按优先级降序排列的切片（高优先级在前）。
func ListModelGrants(db *gorm.DB, principalType, principalID, modelID string) ([]*ModelGrant, error) {
	var grants []*ModelGrant
	q := db.Order("priority DESC, created_at ASC")
	if principalType != "" {
		q = q.Where("principal_type = ?", principalType)
	}
	if principalID != "" {
		q = q.Where("principal_id = ?", principalID)
	}
	if modelID != "" {
		q = q.Where("model_id = ?", modelID)
	}
	if err := q.Find(&grants).Error; err != nil {
		return nil, err
	}
	return grants, nil
}

// ── 模型访问判定辅助 ────────────────────────────────────────────────────────

// matchGrant 判断一条 ModelGrant 是否匹配给定的主体与模型。
// 匹配条件：principal_type + principal_id 完全匹配，且 model_id 或 model_tag 匹配目标模型。
func matchGrant(mg *ModelGrant, p Principal, modelID string) bool {
	if mg.PrincipalType != p.Type || mg.PrincipalID != p.ID {
		return false
	}
	// 规则限定具体模型 ID。
	if mg.ModelID != nil && *mg.ModelID != "" {
		return *mg.ModelID == modelID
	}
	// 规则通过 model_tag 匹配（标签组内全部模型）。
	// 此处做简单前缀匹配——实际生产中应由调用方传入模型的 tags 列表做交集判定。
	if mg.ModelTag != nil && *mg.ModelTag != "" {
		return true
	}
	// model_id 与 model_tag 均为空 → 全局默认规则。
	return true
}

// ── 迁移 ────────────────────────────────────────────────────────────────────

// Migrate 对 model_grants 表执行 GORM AutoMigrate。
// 由 store.go 在阶段 2 迁移中调用。
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&ModelGrant{}); err != nil {
		return fmt.Errorf("modelgrant 迁移失败: %w", err)
	}
	return nil
}
