package fund

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// Decimal wraps shopspring/decimal.Decimal for seamless GORM integration.
// It implements sql.Scanner and driver.Valuer so that monetary values
// are stored as NUMERIC in PostgreSQL and REAL in SQLite without losing
// precision.
type Decimal struct {
	decimal.Decimal
}

// NewDecimal creates a Decimal from a string representation.
// It panics on invalid input because monetary values must be explicitly validated
// at the boundary layer before entering the fund domain.
func NewDecimal(s string) Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(fmt.Sprintf("fund: invalid decimal %q: %v", s, err))
	}
	return Decimal{Decimal: d}
}

// DecPtr creates a Decimal from a float64 for convenience in tests and defaults.
func DecPtr(f float64) Decimal {
	return Decimal{Decimal: decimal.NewFromFloat(f)}
}

// Scan implements sql.Scanner. It reads a NUMERIC/REAL value from the database
// into a Decimal.
func (d *Decimal) Scan(value interface{}) error {
	if value == nil {
		d.Decimal = decimal.Zero
		return nil
	}
	switch v := value.(type) {
	case int64:
		d.Decimal = decimal.NewFromInt(v)
	case float64:
		d.Decimal = decimal.NewFromFloat(v)
	case []byte:
		dec, err := decimal.NewFromString(string(v))
		if err != nil {
			return err
		}
		d.Decimal = dec
	case string:
		dec, err := decimal.NewFromString(v)
		if err != nil {
			return err
		}
		d.Decimal = dec
	default:
		return fmt.Errorf("fund: cannot scan %T into Decimal", value)
	}
	return nil
}

// Value implements driver.Valuer. It serializes a Decimal for database storage.
func (d Decimal) Value() (driver.Value, error) {
	if d.Decimal.IsZero() {
		return "0", nil
	}
	return d.Decimal.String(), nil
}

// MarshalJSON serializes Decimal as a JSON number string for API responses.
func (d Decimal) MarshalJSON() ([]byte, error) {
	return []byte(d.Decimal.String()), nil
}

// UnmarshalJSON deserializes a JSON number string into Decimal.
func (d *Decimal) UnmarshalJSON(data []byte) error {
	str := string(data)
	// Strip quotes if present (JSON string quote)
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}
	dec, err := decimal.NewFromString(str)
	if err != nil {
		return err
	}
	d.Decimal = dec
	return nil
}

// ---------------------------------------------------------------------------
// Account status constants
// ---------------------------------------------------------------------------

const (
	StatusActive              = "active"
	StatusLiquidatingBlockNew = "liquidating_block_new"
	StatusLiquidatingDrain    = "liquidating_drain"
	StatusLiquidatingTransfer = "liquidating_transfer"
	StatusLiquidated          = "liquidated"
	StatusClosed              = "closed"
)

// ---------------------------------------------------------------------------
// Ledger direction constants
// ---------------------------------------------------------------------------

const (
	DirectionDebit      = "debit"
	DirectionCredit     = "credit"
	DirectionFreeze     = "freeze"
	DirectionUnfreeze   = "unfreeze"
	DirectionSettle     = "settle"
	DirectionAllocateIn  = "allocate_in"
	DirectionAllocateOut = "allocate_out"
)

// ---------------------------------------------------------------------------
// Freeze status constants
// ---------------------------------------------------------------------------

const (
	FreezeStatusOpen           = "open"
	FreezeStatusSettled        = "settled"
	FreezeStatusTimeoutReleased = "timeout_released"
	FreezeStatusCancelled      = "cancelled"
)

// ---------------------------------------------------------------------------
// Allocation channel constants
// ---------------------------------------------------------------------------

const (
	ChannelParent    = "parent"
	ChannelSponsors  = "sponsors"
	ChannelAllocates = "allocates"
	ChannelWhitelist = "whitelist"
)

// ---------------------------------------------------------------------------
// Allocation status constants
// ---------------------------------------------------------------------------

const (
	AllocationStatusPending   = "pending"
	AllocationStatusCompleted = "completed"
	AllocationStatusReverted  = "reverted"
)

// ---------------------------------------------------------------------------
// Liquidation status constants (maps to DDL: blocking/draining/refunding/closing/closed)
// ---------------------------------------------------------------------------

const (
	LiquidationStatusBlocking  = "blocking"
	LiquidationStatusDraining  = "draining"
	LiquidationStatusRefunding = "refunding"
	LiquidationStatusClosing   = "closing"
	LiquidationStatusClosed    = "closed"
)

// ---------------------------------------------------------------------------
// Data models
// ---------------------------------------------------------------------------

// Account represents a party's financial account with balance, budget cap,
// and liquidation metadata. It uses optimistic locking via the Version field.
// All monetary fields use Decimal for arbitrary precision.
type Account struct {
	ID                   string    `json:"id" gorm:"primaryKey"`
	PartyID              string    `json:"party_id" gorm:"uniqueIndex;not null"`
	AvailableBalance     Decimal   `json:"available_balance" gorm:"type:numeric(18,6);not null;default:0"`
	FrozenBalance        Decimal   `json:"frozen_balance" gorm:"type:numeric(18,6);not null;default:0"`
	Status               string    `json:"status" gorm:"not null;default:active"`
	BudgetLimitAmount    *Decimal  `json:"budget_limit_amount,omitempty" gorm:"type:numeric(18,6)"`
	BudgetWarnRatio      *Decimal  `json:"budget_warn_ratio,omitempty" gorm:"type:numeric(5,4)"`
	BudgetPeriod         string    `json:"budget_period" gorm:"default:none"`
	BudgetPeriodStart    *time.Time `json:"budget_period_start,omitempty"`
	BudgetPeriodEnd      *time.Time `json:"budget_period_end,omitempty"`
	BudgetConsumedAmount Decimal   `json:"budget_consumed_amount" gorm:"type:numeric(18,6);not null;default:0"`
	BudgetVersion        int64     `json:"budget_version" gorm:"not null;default:0"`
	LiquidationStage     *string   `json:"liquidation_stage,omitempty"`
	LiquidationTargetID  *string   `json:"liquidation_target_id,omitempty"`
	LiquidationStartedAt *time.Time `json:"liquidation_started_at,omitempty"`
	Version              int64     `json:"version" gorm:"not null;default:0"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// TableName overrides the default table name for GORM.
func (Account) TableName() string { return "accounts" }

// Ledger is an append-only transaction record. It captures every balance mutation
// with full context: direction, amount, post-mutation balance snapshot, and
// associated business identifiers. Rows are never updated or deleted.
type Ledger struct {
	ID             string    `json:"id" gorm:"primaryKey"`
	AccountID      string    `json:"account_id" gorm:"index:idx_ledgers_account;not null"`
	Direction      string    `json:"direction" gorm:"not null"`
	Amount         Decimal   `json:"amount" gorm:"type:numeric(18,6);not null"`
	BalanceAfter   Decimal   `json:"balance_after" gorm:"type:numeric(18,6);not null"`
	FrozenAfter    *Decimal  `json:"frozen_after,omitempty" gorm:"type:numeric(18,6)"`
	CostAmount     *Decimal  `json:"cost_amount,omitempty" gorm:"type:numeric(18,6)"`
	SellAmount     *Decimal  `json:"sell_amount,omitempty" gorm:"type:numeric(18,6)"`
	FreezeID       *string   `json:"freeze_id,omitempty" gorm:"index:idx_ledgers_freeze"`
	RequestID      *string   `json:"request_id,omitempty" gorm:"index:idx_ledgers_request"`
	AllocationID   *string   `json:"allocation_id,omitempty"`
	UserID         *string   `json:"user_id,omitempty"`
	APIKeyID       *string   `json:"api_key_id,omitempty"`
	IdempotencyKey *string   `json:"idempotency_key,omitempty" gorm:"index:idx_ledgers_idem"`
	Reason         *string   `json:"reason,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// TableName overrides the default table name for GORM.
func (Ledger) TableName() string { return "ledgers" }

// Freeze represents a fund reservation for an in-flight model call.
// The frozen amount is deducted from available balance and held until
// settlement or timeout. Streaming calls can extend expires_at via RenewFreeze.
type Freeze struct {
	ID             string     `json:"id" gorm:"primaryKey"`
	AccountID      string     `json:"account_id" gorm:"index:idx_freezes_account;not null"`
	RequestID      *string    `json:"request_id,omitempty" gorm:"index:idx_freezes_request"`
	APIKeyID       *string    `json:"api_key_id,omitempty"`
	UserID         string     `json:"user_id" gorm:"index:idx_freezes_user;not null"`
	Amount         Decimal    `json:"amount" gorm:"type:numeric(18,6);not null"`
	EstimatedSell  Decimal    `json:"estimated_sell" gorm:"type:numeric(18,6);not null"`
	Status         string     `json:"status" gorm:"not null;default:active"`
	ExpiresAt      time.Time  `json:"expires_at" gorm:"index:idx_freezes_expiry;not null"`
	MaxLifetimeAt  *time.Time `json:"max_lifetime_at,omitempty"`
	RenewalCount   int        `json:"renewal_count" gorm:"not null;default:0"`
	LastRenewedAt  *time.Time `json:"last_renewed_at,omitempty"`
	SettledAt      *time.Time `json:"settled_at,omitempty"`
	SettleAmount   *Decimal   `json:"settle_amount,omitempty" gorm:"type:numeric(18,6)"`
	SettleCost     *Decimal   `json:"settle_cost,omitempty" gorm:"type:numeric(18,6)"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// TableName overrides the default table name for GORM.
func (Freeze) TableName() string { return "freezes" }

// Allocation records a fund transfer between two accounts, along a permitted
// channel (parent, sponsors, allocates, or whitelist).
type Allocation struct {
	ID             string    `json:"id" gorm:"primaryKey"`
	SrcAccountID   string    `json:"src_account_id" gorm:"index:idx_allocations_src;not null"`
	DstAccountID   string    `json:"dst_account_id" gorm:"index:idx_allocations_dst;not null"`
	Amount         Decimal   `json:"amount" gorm:"type:numeric(18,6);not null"`
	Channel        string    `json:"channel" gorm:"not null"`
	EdgeID         *string   `json:"edge_id,omitempty"`
	Status         string    `json:"status" gorm:"not null;default:pending"`
	IdempotencyKey *string   `json:"idempotency_key,omitempty" gorm:"index:idx_allocations_idem"`
	ActorUserID    *string   `json:"actor_user_id,omitempty"`
	Reason         *string   `json:"reason,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// TableName overrides the default table name for GORM.
func (Allocation) TableName() string { return "allocations" }

// Liquidation tracks the account closure state machine. Once initiated,
// new calls and freezes are blocked; existing freezes drain; remaining
// balance transfers to a target account.
type Liquidation struct {
	ID               string     `json:"id" gorm:"primaryKey"`
	PartyID          string     `json:"party_id" gorm:"index:idx_liquidations_party;not null"`
	AccountID        string     `json:"account_id" gorm:"index:idx_liquidations_account;not null"`
	TargetAccountID  *string    `json:"target_account_id,omitempty"`
	Status           string     `json:"status" gorm:"index:idx_liquidations_status;not null;default:blocking"`
	InitiatedBy      string     `json:"initiated_by" gorm:"not null"`
	InitiatedAt      time.Time  `json:"initiated_at"`
	ClosedAt         *time.Time `json:"closed_at,omitempty"`
	Metadata         *string    `json:"metadata,omitempty" gorm:"type:jsonb"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// TableName overrides the default table name for GORM.
func (Liquidation) TableName() string { return "liquidations" }

// ---------------------------------------------------------------------------
// Request / Result structs
// ---------------------------------------------------------------------------

// AllocateRequest carries the parameters for a fund transfer between accounts.
type AllocateRequest struct {
	// SrcAccountID identifies the source account from which funds are debited.
	SrcAccountID string

	// DstAccountID identifies the destination account to which funds are credited.
	DstAccountID string

	// Amount is the transfer amount. Must be strictly positive.
	Amount Decimal

	// EdgeID optionally references a party_edge authorising this channel.
	EdgeID *string

	// Channel is the transfer channel: parent, sponsors, allocates, or whitelist.
	Channel string

	// IdempotencyKey guarantees at-most-once execution.
	IdempotencyKey string

	// OperatorID identifies the user or service initiating the allocation.
	OperatorID string

	// Reason is an optional business justification.
	Reason *string
}

// AllocateResult holds the outcome of a successful allocation.
type AllocateResult struct {
	AllocationID    string    `json:"allocation_id"`
	SrcAccountID    string    `json:"src_account_id"`
	DstAccountID    string    `json:"dst_account_id"`
	Amount          Decimal   `json:"amount"`
	Channel         string    `json:"channel"`
	EdgeID          *string   `json:"edge_id,omitempty"`
	Status          string    `json:"status"`
	SrcBalanceAfter Decimal   `json:"src_balance_after"`
	DstBalanceAfter Decimal   `json:"dst_balance_after"`
	IdempotencyKey  string    `json:"idempotency_key"`
	CreatedAt       time.Time `json:"created_at"`
	CompletedAt     time.Time `json:"completed_at"`
}

// FreezeRequest carries parameters for reserving funds before a model call.
type FreezeRequest struct {
	// AccountID identifies the account whose balance will be frozen.
	AccountID string

	// Amount is the number of funds to freeze. It should cover the estimated
	// maximum sell amount across all eligible route candidates.
	Amount Decimal

	// EstimatedSell is the pre-computed maximum sell estimate across candidates.
	EstimatedSell Decimal

	// RequestID links this freeze to a specific API call request.
	RequestID string

	// UserID identifies the end user making the call.
	UserID string

	// APIKeyID identifies the gateway key used for the call.
	APIKeyID *string

	// TTL is the freeze duration before automatic expiry. Default 15 minutes.
	TTL time.Duration
}

// FreezeResult holds the outcome of a successful freeze.
type FreezeResult struct {
	FreezeID      string    `json:"freeze_id"`
	AccountID     string    `json:"account_id"`
	Amount        Decimal   `json:"amount"`
	EstimatedSell Decimal   `json:"estimated_sell"`
	BalanceAfter  Decimal   `json:"balance_after"`
	FrozenAfter   Decimal   `json:"frozen_after"`
	ExpiresAt     time.Time `json:"expires_at"`
	RequestID     string    `json:"request_id"`
}

// SettleRequest carries parameters for finalizing a freeze after usage is known.
type SettleRequest struct {
	// FreezeID identifies the freeze to settle.
	FreezeID string

	// ActualSell is the final computed sell amount based on actual usage.
	ActualSell Decimal

	// ActualCost is the final computed cost amount based on actual usage.
	ActualCost Decimal

	// RequestID links this settlement to the originating call.
	RequestID string
}

// SettleResult holds the outcome of a successful settlement.
type SettleResult struct {
	FreezeID       string  `json:"freeze_id"`
	ActualSell     Decimal `json:"actual_sell"`
	ActualCost     Decimal `json:"actual_cost"`
	ReleasedAmount Decimal `json:"released_amount"`  // frozen - actual_sell (refund)
	BalanceAfter   Decimal `json:"balance_after"`
	FrozenAfter    Decimal `json:"frozen_after"`
}

// LiquidateRequest carries parameters for initiating or advancing an account liquidation.
type LiquidateRequest struct {
	// AccountID identifies the account to liquidate.
	AccountID string

	// TargetAccountID is the account that receives remaining funds after drain.
	TargetAccountID string

	// OperatorID identifies the user initiating the liquidation.
	OperatorID string

	// PartyID identifies the party owning the account.
	PartyID string

	// Reason is a mandatory business justification (audited).
	Reason string
}

// LiquidateResult holds the outcome of a liquidation step.
type LiquidateResult struct {
	LiquidationID   string    `json:"liquidation_id"`
	AccountID       string    `json:"account_id"`
	PartyID         string    `json:"party_id"`
	TargetAccountID string    `json:"target_account_id"`
	Status          string    `json:"status"`
	InitiatedBy     string    `json:"initiated_by"`
	Reason          string    `json:"reason"`
	InitiatedAt     time.Time `json:"initiated_at"`
}
