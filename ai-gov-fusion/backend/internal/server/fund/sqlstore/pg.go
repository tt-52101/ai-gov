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

// PgStore implements the fund.Store interface using GORM.
// It is database-agnostic — both PostgreSQL and SQLite are supported
// (the package name reflects the primary production target).
type PgStore struct {
	db *gorm.DB
}

// NewPgStore creates a new PgStore backed by a GORM database connection.
// The caller is responsible for calling AutoMigrate before using the store.
func NewPgStore(db *gorm.DB) *PgStore {
	return &PgStore{db: db}
}

// DB returns the underlying GORM database handle. Use only for AutoMigrate
// registration and diagnostics — never for direct queries in service code.
func (s *PgStore) DB() *gorm.DB {
	return s.db
}

// AutoMigrate registers fund domain tables for GORM auto-migration.
// Called from the main store initialisation in store.go.
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

// pgTx wraps a *gorm.DB transaction to implement fund.Tx.
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
// Store interface implementation
// ---------------------------------------------------------------------------

// WithTx executes fn within a single database transaction.
// If fn returns an error, the transaction is rolled back. Otherwise committed.
func (s *PgStore) WithTx(ctx context.Context, fn func(fund.Tx) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&pgTx{tx: tx})
	})
}

// ---------------------------------------------------------------------------
// Account operations
// ---------------------------------------------------------------------------

// GetAccount retrieves an account by its primary key.
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

// GetAccountForUpdate retrieves an account with a row-level lock (SELECT ... FOR UPDATE).
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

// UpdateAccountBalances atomically updates available_balance and frozen_balance
// with optimistic locking via the version field.
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
		return fmt.Errorf("fund: optimistic lock failure on account %s (version %d)", id, version)
	}
	return nil
}

// UpdateAccountStatus sets the account status with optimistic locking.
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
		return fmt.Errorf("fund: optimistic lock failure on account %s (version %d)", id, version)
	}
	return nil
}

// UpdateAccountBudgetConsumed increments budget_consumed_amount by delta.
func (s *PgStore) UpdateAccountBudgetConsumed(tx fund.Tx, ctx context.Context, id string, delta decimal.Decimal) error {
	gtx := tx.(*pgTx).tx
	result := gtx.WithContext(ctx).Model(&fund.Account{}).
		Where("id = ?", id).
		Update("budget_consumed_amount", gorm.Expr("budget_consumed_amount + ?", delta.String()))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("fund: account %s not found", id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Ledger (append-only)
// ---------------------------------------------------------------------------

// InsertLedger appends a single ledger entry.
func (s *PgStore) InsertLedger(tx fund.Tx, ctx context.Context, entry *fund.Ledger) error {
	gtx := tx.(*pgTx).tx
	return gtx.WithContext(ctx).Create(entry).Error
}

// ---------------------------------------------------------------------------
// Freeze operations
// ---------------------------------------------------------------------------

// InsertFreeze creates a new freeze record.
func (s *PgStore) InsertFreeze(tx fund.Tx, ctx context.Context, f *fund.Freeze) error {
	gtx := tx.(*pgTx).tx
	return gtx.WithContext(ctx).Create(f).Error
}

// GetFreeze retrieves a freeze by its primary key.
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

// UpdateFreezeStatus sets the freeze status and optionally updates settle fields.
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
		return fmt.Errorf("fund: freeze %s not found", freezeID)
	}
	return nil
}

// RenewFreeze extends expires_at and increments renewal_count.
func (s *PgStore) RenewFreeze(tx fund.Tx, ctx context.Context, freezeID string, newExpiresAt string) (int64, error) {
	gtx := tx.(*pgTx).tx
	expiresAt, err := time.Parse(time.RFC3339Nano, newExpiresAt)
	if err != nil {
		return 0, fmt.Errorf("fund: invalid expires_at format: %w", err)
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

// ListExpiredFreezes returns open freezes past their expiry, up to limit rows.
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
// Allocation
// ---------------------------------------------------------------------------

// InsertAllocation creates an allocation record.
func (s *PgStore) InsertAllocation(tx fund.Tx, ctx context.Context, a *fund.Allocation) error {
	gtx := tx.(*pgTx).tx
	return gtx.WithContext(ctx).Create(a).Error
}

// UpdateAllocationStatus sets the allocation status and completed_at timestamp.
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
		return fmt.Errorf("fund: allocation %s not found", id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Liquidation
// ---------------------------------------------------------------------------

// GetLiquidation retrieves the active liquidation for an account.
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

// InsertLiquidation creates a new liquidation record.
func (s *PgStore) InsertLiquidation(tx fund.Tx, ctx context.Context, l *fund.Liquidation) error {
	gtx := tx.(*pgTx).tx
	return gtx.WithContext(ctx).Create(l).Error
}

// UpdateLiquidationStage advances the liquidation status.
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
		return fmt.Errorf("fund: liquidation %s not found", id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Idempotency
// ---------------------------------------------------------------------------

// idempotencyRecord is a private table for storing allocation idempotency records.
type idempotencyRecord struct {
	ID             string    `gorm:"primaryKey"`
	IdempotencyKey string    `gorm:"uniqueIndex:idx_fund_idem_key"`
	SrcAccountID   string    `gorm:"not null"`
	DstAccountID   string    `gorm:"not null"`
	Amount         string    `gorm:"not null"`
	Channel        string    `gorm:"not null"`
	EdgeID         string
	Status         string    `gorm:"not null;default:completed"`
	AllocationID   string    `gorm:"not null"`
	ResultJSON     string    `gorm:"type:text;not null"`
	CreatedAt      time.Time
}

// TableName overrides the default table name.
func (idempotencyRecord) TableName() string { return "fund_idempotency" }

// CheckIdempotency looks up a previous allocation result by idempotency key.
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
		return nil, false, fmt.Errorf("fund: idempotency record corruption: %w", err)
	}
	return &allocateResult, true, nil
}

// StoreIdempotency persists an idempotency result.
func (s *PgStore) StoreIdempotency(tx fund.Tx, ctx context.Context, key string, result *fund.AllocateResult) error {
	gtx := tx.(*pgTx).tx
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("fund: marshal idempotency result: %w", err)
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
