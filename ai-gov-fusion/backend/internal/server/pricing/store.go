package pricing

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ── GORM 持久化操作 ──

// Migrate 执行 model_prices 表的自动建表/迁移。
// 遵循 TokenHub 的 AutoMigrate 模式，由 store.go 在启动阶段统一调用。
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&ModelPrice{}); err != nil {
		return fmt.Errorf("pricing migrate: %w", err)
	}
	return nil
}

// UpsertPrice 插入或更新价目记录。
//
// 以 reference_id 为唯一键执行 INSERT ON CONFLICT UPDATE。
// 若 reference_id 已存在，则更新 price_json、status、effective_* 等字段。
// 返回的 price 实例会填充 ID 和 timestamps。
//
// 并发安全: 依赖数据库的 unique 约束保证 reference_id 唯一性。
func UpsertPrice(db *gorm.DB, price *ModelPrice) error {
	if price == nil {
		return errors.New("pricing: cannot upsert nil price")
	}
	if price.ReferenceID == "" {
		return errors.New("pricing: reference_id is required for upsert")
	}

	result := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "reference_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"model_id", "channel_id", "price_json", "status", "effective_start_at", "effective_end_at", "updated_at"}),
	}).Create(price)

	if result.Error != nil {
		return fmt.Errorf("pricing: upsert price %q: %w", price.ReferenceID, result.Error)
	}
	return nil
}

// GetPrice 按 model_id + channel_id + reference_id 查找一条有效价目。
//
// 返回 status='active' 的唯一条目；若多条匹配（理论上不应发生），取第一条。
// channelID 可为空字符串，表示查找未绑定渠道的默认价目。
func GetPrice(db *gorm.DB, modelID string, channelID string, referenceID string) (*ModelPrice, error) {
	var price ModelPrice
	query := db.Where("model_id = ? AND reference_id = ? AND status = ?", modelID, referenceID, StatusActive)

	if channelID == "" {
		query = query.Where("channel_id IS NULL")
	} else {
		query = query.Where("channel_id = ?", channelID)
	}

	if err := query.First(&price).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("pricing: price not found for model=%q channel=%q ref=%q", modelID, channelID, referenceID)
		}
		return nil, fmt.Errorf("pricing: get price: %w", err)
	}
	return &price, nil
}

// ListPrices 列出指定模型的所有有效价目。
//
// 按 created_at 降序排列，最新的靠前。
// 仅返回 status='active' 的条目。
func ListPrices(db *gorm.DB, modelID string) ([]*ModelPrice, error) {
	var prices []*ModelPrice
	if err := db.Where("model_id = ? AND status = ?", modelID, StatusActive).
		Order("created_at DESC").
		Find(&prices).Error; err != nil {
		return nil, fmt.Errorf("pricing: list prices for model %q: %w", modelID, err)
	}
	return prices, nil
}

// ArchivePrice 软删除价目，将 status 置为 'archived'。
//
// 不会物理删除数据库行；历史结算记录仍可追溯原始价目。
// 若 price 已被归档则此操作为幂等 (不返回错误)。
func ArchivePrice(db *gorm.DB, id string) error {
	result := db.Model(&ModelPrice{}).Where("id = ?", id).Update("status", StatusArchived)
	if result.Error != nil {
		return fmt.Errorf("pricing: archive price %q: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("pricing: price %q not found for archive", id)
	}
	return nil
}
