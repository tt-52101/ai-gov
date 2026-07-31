package fund

import (
	"context"

	"github.com/shopspring/decimal"
)

// Tx represents a database transaction. Implementations typically wrap
// *gorm.DB. All mutating Store methods accept a Tx to ensure atomicity.
type Tx interface {
	// Commit persists the transaction. Returns an error if the underlying
	// database commit fails.
	Commit() error

	// Rollback aborts the transaction. Safe to call multiple times.
	Rollback() error
}

// Store defines the persistence contract for the fund domain.
// All monetary mutations MUST go through this interface — never call GORM
// directly from service logic (AGENTS.md governance rule).
//
// The implementation is expected to use GORM under the hood, but the interface
// isolates the service from ORM details, enabling clean testing with in-memory
// fakes and making it straightforward to swap databases.
type Store interface {
	// WithTx executes fn within a single database transaction.
	// If fn returns an error the transaction is rolled back; otherwise committed.
	WithTx(ctx context.Context, fn func(tx Tx) error) error

	// -----------------------------------------------------------------------
	// Account operations
	// -----------------------------------------------------------------------

	// GetAccount retrieves an account by its primary key.
	// Returns nil, nil when the account does not exist.
	GetAccount(ctx context.Context, id string) (*Account, error)

	// GetAccountForUpdate retrieves an account with a row-level lock (SELECT ... FOR UPDATE).
	// Must be called within a transaction. The returned account's Version field is
	// used for optimistic concurrency control.
	GetAccountForUpdate(tx Tx, ctx context.Context, id string) (*Account, error)

	// UpdateAccountBalances atomically updates available_balance and frozen_balance,
	// incrementing version. The version parameter is the previously-read version;
	// if it does not match the database row the update fails (optimistic lock).
	UpdateAccountBalances(tx Tx, ctx context.Context, id string, available, frozen decimal.Decimal, version int64) error

	// UpdateAccountStatus sets the account status. The version parameter enforces
	// optimistic locking.
	UpdateAccountStatus(tx Tx, ctx context.Context, id string, status string, version int64) error

	// UpdateAccountBudgetConsumed increments budget_consumed_amount by delta.
	UpdateAccountBudgetConsumed(tx Tx, ctx context.Context, id string, delta decimal.Decimal) error

	// -----------------------------------------------------------------------
	// Ledger (append-only)
	// -----------------------------------------------------------------------

	// InsertLedger appends a single ledger entry. Ledger rows are never updated
	// or deleted — this is an append-only log (F-CON-01).
	InsertLedger(tx Tx, ctx context.Context, entry *Ledger) error

	// -----------------------------------------------------------------------
	// Freeze operations
	// -----------------------------------------------------------------------

	// InsertFreeze creates a new freeze record. The freeze status must be "open".
	InsertFreeze(tx Tx, ctx context.Context, f *Freeze) error

	// GetFreeze retrieves a freeze by its primary key.
	// Returns nil, nil when the freeze does not exist.
	GetFreeze(ctx context.Context, freezeID string) (*Freeze, error)

	// UpdateFreezeStatus sets the freeze status and optionally updates
	// settle_amount, settle_cost, and settled_at for settlement.
	UpdateFreezeStatus(tx Tx, ctx context.Context, freezeID string, status string, settleAmount, settleCost *decimal.Decimal) error

	// RenewFreeze extends expires_at and increments renewal_count.
	// Returns the number of rows affected (0 means freeze not found or not open).
	RenewFreeze(tx Tx, ctx context.Context, freezeID string, newExpiresAt string) (int64, error)

	// ListExpiredFreezes returns open freezes whose expires_at is before now,
	// up to limit rows. Used by the TTL scanner background job.
	ListExpiredFreezes(ctx context.Context, limit int) ([]*Freeze, error)

	// -----------------------------------------------------------------------
	// Allocation
	// -----------------------------------------------------------------------

	// InsertAllocation creates an allocation record.
	InsertAllocation(tx Tx, ctx context.Context, a *Allocation) error

	// UpdateAllocationStatus sets the allocation status and completed_at timestamp.
	UpdateAllocationStatus(tx Tx, ctx context.Context, id string, status string) error

	// -----------------------------------------------------------------------
	// Liquidation
	// -----------------------------------------------------------------------

	// GetLiquidation retrieves the active liquidation for an account.
	// Returns nil, nil when no liquidation exists.
	GetLiquidation(ctx context.Context, accountID string) (*Liquidation, error)

	// InsertLiquidation creates a new liquidation record.
	InsertLiquidation(tx Tx, ctx context.Context, l *Liquidation) error

	// UpdateLiquidationStage advances the liquidation status and updates metadata.
	UpdateLiquidationStage(tx Tx, ctx context.Context, id string, stage string) error

	// -----------------------------------------------------------------------
	// Idempotency
	// -----------------------------------------------------------------------

	// CheckIdempotency looks up a previous result by idempotency key.
	// Returns the stored result and true if found, or nil, false if not.
	CheckIdempotency(ctx context.Context, key string) (*AllocateResult, bool, error)

	// StoreIdempotency persists an idempotency result for future lookups.
	// Must be called within the same transaction as the allocation it guards.
	StoreIdempotency(tx Tx, ctx context.Context, key string, result *AllocateResult) error
}
