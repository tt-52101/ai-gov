package sqlstore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"tokenhub/backend/internal/server/fund"
)

// PgStore 使用 GORM 实现 fund.Store 接口。
// 它数据库无关——PostgreSQL 和 SQLite 均支持（包名反映主要生产目标）。
type PgStore struct {
	db *gorm.DB
}

// NewPgStore 基于 GORM 数据库连接创建新的 PgStore。
// 调用方负责在使用 Store 之前调用 AutoMigrate。
func NewPgStore(db *gorm.DB) *PgStore {
	return &PgStore{db: db}
}

// DB 返回底层 GORM 数据库句柄。仅用于 AutoMigrate 注册和诊断——
// 绝不可在服务代码中直接查询。
func (s *PgStore) DB() *gorm.DB {
	return s.db
}

// AutoMigrate 注册 fund 领域表用于 GORM 自动迁移。
// 由 store.go 中的主存储初始化调用。
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&fund.Account{},
		&fund.Ledger{},
		&fund.Freeze{},
		&fund.Allocation{},
		&fund.Liquidation{},
		&idempotencyRecord{},
	)
}

// pgTx 包装 *gorm.DB 事务以实现 fund.Tx。
type pgTx struct {
	tx *gorm.DB
}

func (t *pgTx) Commit() error {
	return t.tx.Commit().Error
}

func (t *pgTx) Rollback() error {
	return t.tx.Rollback().Error
}

// ---------------------------------------------------------------------------
// Store 接口实现
// ---------------------------------------------------------------------------

// WithTx 在单个数据库事务中执行 fn。
// 若 fn 返回错误则回滚事务，否则提交。
func (s *PgStore) WithTx(ctx context.Context, fn func(fund.Tx) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&pgTx{tx: tx})
	})
}

// ---------------------------------------------------------------------------
// 账户操作
// ---------------------------------------------------------------------------

// GetAccount 按主键检索账户。
func (s *PgStore) GetAccount(ctx context.Context, id string) (*fund.Account, error) {
	var acct fund.Account
	result := s.db.WithContext(ctx).Where("id = ?", id).First(&acct)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &acct, nil
}

// GetAccountForUpdate 以行级锁检索账户（SELECT ... FOR UPDATE）。
func (s *PgStore) GetAccountForUpdate(tx fund.Tx, ctx context.Context, id string) (*fund.Account, error) {
	gtx := tx.(*pgTx).tx
	var acct fund.Account
	result := gtx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).First(&acct)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &acct, nil
}

// UpdateAccountBalances 通过 version 字段的乐观锁原子更新 available_balance 和 frozen_balance。
func (s *PgStore) UpdateAccountBalances(tx fund.Tx, ctx context.Context, id string, available, frozen decimal.Decimal, version int64) error {
	gtx := tx.(*pgTx).tx
	result := gtx.WithContext(ctx).Model(&fund.Account{}).
		Where("id = ? AND version = ?", id, version).
		Updates(map[string]interface{}{
			"available_balance": available.String(),
			"frozen_balance":    frozen.String(),
			"version":           version + 1,
			"updated_at":        time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("fund: 账户 %s 乐观锁失败 (version %d)", id, version)
	}
	return nil
}

// UpdateAccountStatus 以乐观锁设置账户状态。
func (s *PgStore) UpdateAccountStatus(tx fund.Tx, ctx context.Context, id string, status string, version int64) error {
	gtx := tx.(*pgTx).tx
	result := gtx.WithContext(ctx).Model(&fund.Account{}).
		Where("id = ? AND version = ?", id, version).
		Updates(map[string]interface{}{
			"status":     status,
			"version":    version + 1,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("fund: 账户 %s 乐观锁失败 (version %d)", id, version)
	}
	return nil
}

// UpdateAccountBudgetConsumed 按 delta 递增 budget_consumed_amount。
func (s *PgStore) UpdateAccountBudgetConsumed(tx fund.Tx, ctx context.Context, id string, delta decimal.Decimal) error {
	gtx := tx.(*pgTx).tx
	result := gtx.WithContext(ctx).Model(&fund.Account{}).
		Where("id = ?", id).
		Update("budget_consumed_amount", gorm.Expr("budget_consumed_amount + ?", delta.String()))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("fund: 账户 %s 未找到", id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// 账本（追加不可变）
// ---------------------------------------------------------------------------

// InsertLedger 追加一条账本记录。
func (s *PgStore) InsertLedger(tx fund.Tx, ctx context.Context, entry *fund.Ledger) error {
	gtx := tx.(*pgTx).tx
	return gtx.WithContext(ctx).Create(entry).Error
}

// ---------------------------------------------------------------------------
// 冻结操作
// ---------------------------------------------------------------------------

// InsertFreeze 创建新的冻结记录。
func (s *PgStore) InsertFreeze(tx fund.Tx, ctx context.Context, f *fund.Freeze) error {
	gtx := tx.(*pgTx).tx
	return gtx.WithContext(ctx).Create(f).Error
}

// GetFreeze 按主键检索冻结记录。
func (s *PgStore) GetFreeze(ctx context.Context, freezeID string) (*fund.Freeze, error) {
	var f fund.Freeze
	result := s.db.WithContext(ctx).Where("id = ?", freezeID).First(&f)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &f, nil
}

// GetFreezeForUpdate 以行级锁检索冻结记录（SELECT ... FOR UPDATE）。
// 用于 Settle 操作——防止并发结算同一 freeze_id 产生竞态窗口（RED-2 安全修复）。
func (s *PgStore) GetFreezeForUpdate(tx fund.Tx, ctx context.Context, freezeID string) (*fund.Freeze, error) {
	gtx := tx.(*pgTx).tx
	var f fund.Freeze
	result := gtx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", freezeID).First(&f)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &f, nil
}

// UpdateFreezeStatus 设置冻结状态，并可选择更新结算相关字段。
func (s *PgStore) UpdateFreezeStatus(tx fund.Tx, ctx context.Context, freezeID string, status string, settleAmount, settleCost *decimal.Decimal) error {
	gtx := tx.(*pgTx).tx
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if settleAmount != nil {
		updates["settle_amount"] = settleAmount.String()
	}
	if settleCost != nil {
		updates["settle_cost"] = settleCost.String()
	}
	if status == fund.FreezeStatusSettled {
		updates["settled_at"] = time.Now()
	}
	result := gtx.WithContext(ctx).Model(&fund.Freeze{}).
		Where("id = ?", freezeID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("fund: 冻结 %s 未找到", freezeID)
	}
	return nil
}

// RenewFreeze 延长 expires_at 并递增 renewal_count。
func (s *PgStore) RenewFreeze(tx fund.Tx, ctx context.Context, freezeID string, newExpiresAt string) (int64, error) {
	gtx := tx.(*pgTx).tx
	expiresAt, err := time.Parse(time.RFC3339Nano, newExpiresAt)
	if err != nil {
		return 0, fmt.Errorf("fund: 无效的 expires_at 格式: %w", err)
	}
	now := time.Now()
	result := gtx.WithContext(ctx).Model(&fund.Freeze{}).
		Where("id = ? AND status = ?", freezeID, fund.FreezeStatusOpen).
		Updates(map[string]interface{}{
			"expires_at":      expiresAt,
			"renewal_count":   gorm.Expr("renewal_count + 1"),
			"last_renewed_at": now,
			"updated_at":      now,
		})
	return result.RowsAffected, result.Error
}

// ListExpiredFreezes 返回过期且状态为 open 的冻结记录，最多 limit 行。
func (s *PgStore) ListExpiredFreezes(ctx context.Context, limit int) ([]*fund.Freeze, error) {
	var freezes []*fund.Freeze
	result := s.db.WithContext(ctx).
		Where("status = ? AND expires_at < ?", fund.FreezeStatusOpen, time.Now()).
		Order("expires_at ASC").
		Limit(limit).
		Find(&freezes)
	if result.Error != nil {
		return nil, result.Error
	}
	return freezes, nil
}

// ---------------------------------------------------------------------------
// 划拨
// ---------------------------------------------------------------------------

// InsertAllocation 创建划拨记录。
func (s *PgStore) InsertAllocation(tx fund.Tx, ctx context.Context, a *fund.Allocation) error {
	gtx := tx.(*pgTx).tx
	return gtx.WithContext(ctx).Create(a).Error
}

// UpdateAllocationStatus 设置划拨状态和 completed_at 时间戳。
func (s *PgStore) UpdateAllocationStatus(tx fund.Tx, ctx context.Context, id string, status string) error {
	gtx := tx.(*pgTx).tx
	updates := map[string]interface{}{
		"status": status,
	}
	if status == fund.AllocationStatusCompleted {
		updates["completed_at"] = time.Now()
	}
	result := gtx.WithContext(ctx).Model(&fund.Allocation{}).
		Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("fund: 划拨 %s 未找到", id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// 清算
// ---------------------------------------------------------------------------

// GetLiquidation 检索账户的活跃清算记录。
func (s *PgStore) GetLiquidation(ctx context.Context, accountID string) (*fund.Liquidation, error) {
	var liq fund.Liquidation
	result := s.db.WithContext(ctx).
		Where("account_id = ?", accountID).
		Where("status NOT IN ?", []string{fund.LiquidationStatusClosed}).
		Order("created_at DESC").
		First(&liq)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &liq, nil
}

// InsertLiquidation 创建新的清算记录。
func (s *PgStore) InsertLiquidation(tx fund.Tx, ctx context.Context, l *fund.Liquidation) error {
	gtx := tx.(*pgTx).tx
	return gtx.WithContext(ctx).Create(l).Error
}

// UpdateLiquidationStage 推进清算状态。
func (s *PgStore) UpdateLiquidationStage(tx fund.Tx, ctx context.Context, id string, stage string) error {
	gtx := tx.(*pgTx).tx
	updates := map[string]interface{}{
		"status":     stage,
		"updated_at": time.Now(),
	}
	if stage == fund.LiquidationStatusClosed {
		updates["closed_at"] = time.Now()
	}
	result := gtx.WithContext(ctx).Model(&fund.Liquidation{}).
		Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("fund: 清算 %s 未找到", id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// 幂等
// ---------------------------------------------------------------------------

// idempotencyRecord 是存储划拨幂等记录的私有表。
type idempotencyRecord struct {
	ID             string `gorm:"primaryKey"`
	IdempotencyKey string `gorm:"uniqueIndex:idx_fund_idem_key"`
	SrcAccountID   string `gorm:"not null"`
	DstAccountID   string `gorm:"not null"`
	Amount         string `gorm:"not null"`
	Channel        string `gorm:"not null"`
	EdgeID         string
	Status         string    `gorm:"not null;default:completed"`
	AllocationID   string    `gorm:"not null"`
	ResultJSON     string    `gorm:"type:text;not null"`
	CreatedAt      time.Time
}

// TableName 覆盖默认表名。
func (idempotencyRecord) TableName() string { return "fund_idempotency" }

// CheckIdempotency 按幂等键查找先前的划拨结果。
func (s *PgStore) CheckIdempotency(ctx context.Context, key string) (*fund.AllocateResult, bool, error) {
	var rec idempotencyRecord
	result := s.db.WithContext(ctx).Where("idempotency_key = ?", key).First(&rec)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, false, nil
		}
		return nil, false, result.Error
	}
	var allocateResult fund.AllocateResult
	if err := json.Unmarshal([]byte(rec.ResultJSON), &allocateResult); err != nil {
		return nil, false, fmt.Errorf("fund: 幂等记录损坏: %w", err)
	}
	return &allocateResult, true, nil
}

// StoreIdempotency 持久化幂等结果。
func (s *PgStore) StoreIdempotency(tx fund.Tx, ctx context.Context, key string, result *fund.AllocateResult) error {
	gtx := tx.(*pgTx).tx
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("fund: 序列化幂等结果失败: %w", err)
	}
	rec := idempotencyRecord{
		ID:             result.AllocationID,
		IdempotencyKey: key,
		SrcAccountID:   result.SrcAccountID,
		DstAccountID:   result.DstAccountID,
		Amount:         result.Amount.String(),
		Channel:        result.Channel,
		Status:         result.Status,
		AllocationID:   result.AllocationID,
		ResultJSON:     string(resultJSON),
		CreatedAt:      result.CreatedAt,
	}
	if result.EdgeID != nil {
		rec.EdgeID = *result.EdgeID
	}
	return gtx.WithContext(ctx).Create(&rec).Error
}
