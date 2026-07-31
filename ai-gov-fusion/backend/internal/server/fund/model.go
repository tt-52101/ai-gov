package fund

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// Decimal 包装 shopspring/decimal.Decimal，实现与 GORM 的无缝集成。
// 它实现 sql.Scanner 和 driver.Valuer，使得货币值在 PostgreSQL 中以 NUMERIC 存储、
// 在 SQLite 中以 REAL 存储，同时不丢失精度。
type Decimal struct {
	decimal.Decimal
}

// NewDecimal 从字符串表示创建 Decimal。
// 若输入无效则 panic，因为货币值必须在进入 fund 领域之前于边界层显式校验。
func NewDecimal(s string) Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(fmt.Sprintf("fund: 无效 decimal %q: %v", s, err))
	}
	return Decimal{Decimal: d}
}

// DecPtr 从 float64 创建 Decimal，方便测试和默认值使用。
func DecPtr(f float64) Decimal {
	return Decimal{Decimal: decimal.NewFromFloat(f)}
}

// Scan 实现 sql.Scanner。将数据库中的 NUMERIC/REAL 值读取为 Decimal。
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
		return fmt.Errorf("fund: 无法将 %T 扫描为 Decimal", value)
	}
	return nil
}

// Value 实现 driver.Valuer。将 Decimal 序列化供数据库存储。
func (d Decimal) Value() (driver.Value, error) {
	if d.Decimal.IsZero() {
		return "0", nil
	}
	return d.Decimal.String(), nil
}

// MarshalJSON 将 Decimal 序列化为 JSON 数字字符串用于 API 响应。
func (d Decimal) MarshalJSON() ([]byte, error) {
	return []byte(d.Decimal.String()), nil
}

// UnmarshalJSON 将 JSON 数字字符串反序列化为 Decimal。
func (d *Decimal) UnmarshalJSON(data []byte) error {
	str := string(data)
	// 若存在引号则去除（JSON 字符串引号）
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
// 账户状态常量
// ---------------------------------------------------------------------------

const (
	StatusActive              = "active"                // 活跃
	StatusLiquidatingBlockNew = "liquidating_block_new" // 清算中-阻止新操作
	StatusLiquidatingDrain    = "liquidating_drain"     // 清算中-排空
	StatusLiquidatingTransfer = "liquidating_transfer"  // 清算中-划转
	StatusLiquidated          = "liquidated"            // 已清算
	StatusClosed              = "closed"                // 已关闭
)

// ---------------------------------------------------------------------------
// 账本方向常量
// ---------------------------------------------------------------------------

const (
	DirectionDebit       = "debit"        // 借方
	DirectionCredit      = "credit"       // 贷方
	DirectionFreeze      = "freeze"       // 冻结
	DirectionUnfreeze    = "unfreeze"     // 解冻
	DirectionSettle      = "settle"       // 结算
	DirectionAllocateIn  = "allocate_in"  // 划拨入
	DirectionAllocateOut = "allocate_out" // 划拨出
)

// ---------------------------------------------------------------------------
// 冻结状态常量
// ---------------------------------------------------------------------------

const (
	FreezeStatusOpen            = "open"             // 打开
	FreezeStatusSettled         = "settled"          // 已结算
	FreezeStatusTimeoutReleased = "timeout_released" // 超时释放
	FreezeStatusCancelled       = "cancelled"        // 已取消
)

// ---------------------------------------------------------------------------
// 划拨通道常量
// ---------------------------------------------------------------------------

const (
	ChannelParent    = "parent"    // 上级划拨
	ChannelSponsors  = "sponsors"  // 出资方划拨
	ChannelAllocates = "allocates" // 分配划拨
	ChannelWhitelist = "whitelist" // 白名单划拨
)

// ---------------------------------------------------------------------------
// 划拨状态常量
// ---------------------------------------------------------------------------

const (
	AllocationStatusPending   = "pending"   // 待处理
	AllocationStatusCompleted = "completed" // 已完成
	AllocationStatusReverted  = "reverted"  // 已回退
)

// ---------------------------------------------------------------------------
// 清算状态常量（对应 DDL：blocking/draining/refunding/closing/closed）
// ---------------------------------------------------------------------------

const (
	LiquidationStatusBlocking  = "blocking"  // 阻止中
	LiquidationStatusDraining  = "draining"  // 排空中
	LiquidationStatusRefunding = "refunding" // 退款中
	LiquidationStatusClosing   = "closing"   // 关闭中
	LiquidationStatusClosed    = "closed"    // 已关闭
)

// ---------------------------------------------------------------------------
// 数据模型
// ---------------------------------------------------------------------------

// Account 表示一个组织的财务账户，包含余额、预算上限和清算元数据。
// 使用 Version 字段实现乐观锁。所有货币字段使用 Decimal 以保证任意精度。
type Account struct {
	ID                   string     `json:"id" gorm:"primaryKey"`
	PartyID              string     `json:"party_id" gorm:"uniqueIndex;not null"`
	AvailableBalance     Decimal    `json:"available_balance" gorm:"type:numeric(18,6);not null;default:0"`
	FrozenBalance        Decimal    `json:"frozen_balance" gorm:"type:numeric(18,6);not null;default:0"`
	Status               string     `json:"status" gorm:"not null;default:active"`
	BudgetLimitAmount    *Decimal   `json:"budget_limit_amount,omitempty" gorm:"type:numeric(18,6)"`
	BudgetWarnRatio      *Decimal   `json:"budget_warn_ratio,omitempty" gorm:"type:numeric(5,4)"`
	BudgetPeriod         string     `json:"budget_period" gorm:"default:none"`
	BudgetPeriodStart    *time.Time `json:"budget_period_start,omitempty"`
	BudgetPeriodEnd      *time.Time `json:"budget_period_end,omitempty"`
	BudgetConsumedAmount Decimal    `json:"budget_consumed_amount" gorm:"type:numeric(18,6);not null;default:0"`
	BudgetVersion        int64      `json:"budget_version" gorm:"not null;default:0"`
	LiquidationStage     *string    `json:"liquidation_stage,omitempty"`
	LiquidationTargetID  *string    `json:"liquidation_target_id,omitempty"`
	LiquidationStartedAt *time.Time `json:"liquidation_started_at,omitempty"`
	Version              int64      `json:"version" gorm:"not null;default:0"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// TableName 覆盖 GORM 默认表名。
func (Account) TableName() string { return "accounts" }

// Ledger 是一条追加不可变的事务记录。它记录每一次余额变更，包含完整的上下文：
// 方向、金额、变更后余额快照和关联的业务标识。行记录永不更新或删除。
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

// TableName 覆盖 GORM 默认表名。
func (Ledger) TableName() string { return "ledgers" }

// Freeze 表示一次进行中的模型调用的资金预留。
// 冻结金额从可用余额中扣除，保留至结算或超时。流式调用可通过 RenewFreeze 延长 expires_at。
type Freeze struct {
	ID            string     `json:"id" gorm:"primaryKey"`
	AccountID     string     `json:"account_id" gorm:"index:idx_freezes_account;not null"`
	RequestID     *string    `json:"request_id,omitempty" gorm:"index:idx_freezes_request"`
	APIKeyID      *string    `json:"api_key_id,omitempty"`
	UserID        string     `json:"user_id" gorm:"index:idx_freezes_user;not null"`
	Amount        Decimal    `json:"amount" gorm:"type:numeric(18,6);not null"`
	EstimatedSell Decimal    `json:"estimated_sell" gorm:"type:numeric(18,6);not null"`
	Status        string     `json:"status" gorm:"not null;default:active"`
	ExpiresAt     time.Time  `json:"expires_at" gorm:"index:idx_freezes_expiry;not null"`
	MaxLifetimeAt *time.Time `json:"max_lifetime_at,omitempty"`
	RenewalCount  int        `json:"renewal_count" gorm:"not null;default:0"`
	LastRenewedAt *time.Time `json:"last_renewed_at,omitempty"`
	SettledAt     *time.Time `json:"settled_at,omitempty"`
	SettleAmount  *Decimal   `json:"settle_amount,omitempty" gorm:"type:numeric(18,6)"`
	SettleCost    *Decimal   `json:"settle_cost,omitempty" gorm:"type:numeric(18,6)"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// TableName 覆盖 GORM 默认表名。
func (Freeze) TableName() string { return "freezes" }

// Allocation 记录两个账户之间沿允许通道（parent/sponsors/allocates/whitelist）的资金划拨。
type Allocation struct {
	ID             string     `json:"id" gorm:"primaryKey"`
	SrcAccountID   string     `json:"src_account_id" gorm:"index:idx_allocations_src;not null"`
	DstAccountID   string     `json:"dst_account_id" gorm:"index:idx_allocations_dst;not null"`
	Amount         Decimal    `json:"amount" gorm:"type:numeric(18,6);not null"`
	Channel        string     `json:"channel" gorm:"not null"`
	EdgeID         *string    `json:"edge_id,omitempty"`
	Status         string     `json:"status" gorm:"not null;default:pending"`
	IdempotencyKey *string    `json:"idempotency_key,omitempty" gorm:"index:idx_allocations_idem"`
	ActorUserID    *string    `json:"actor_user_id,omitempty"`
	Reason         *string    `json:"reason,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// TableName 覆盖 GORM 默认表名。
func (Allocation) TableName() string { return "allocations" }

// Liquidation 追踪账户关闭状态机。一旦启动，新调用和冻结会被阻止；
// 既有冻结排空；剩余余额转移到目标账户。
type Liquidation struct {
	ID              string     `json:"id" gorm:"primaryKey"`
	PartyID         string     `json:"party_id" gorm:"index:idx_liquidations_party;not null"`
	AccountID       string     `json:"account_id" gorm:"index:idx_liquidations_account;not null"`
	TargetAccountID *string    `json:"target_account_id,omitempty"`
	Status          string     `json:"status" gorm:"index:idx_liquidations_status;not null;default:blocking"`
	InitiatedBy     string     `json:"initiated_by" gorm:"not null"`
	InitiatedAt     time.Time  `json:"initiated_at"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`
	Metadata        *string    `json:"metadata,omitempty" gorm:"type:jsonb"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// TableName 覆盖 GORM 默认表名。
func (Liquidation) TableName() string { return "liquidations" }

// ---------------------------------------------------------------------------
// 请求/结果结构体
// ---------------------------------------------------------------------------

// AllocateRequest 承载账户间资金划拨的参数。
type AllocateRequest struct {
	// SrcAccountID 标识资金扣减的源账户。
	SrcAccountID string

	// DstAccountID 标识资金贷记的目标账户。
	DstAccountID string

	// Amount 是划拨金额，必须严格为正。
	Amount Decimal

	// EdgeID 可选地引用一条授权此通道的 party_edge。
	EdgeID *string

	// Channel 是划拨通道：parent、sponsors、allocates 或 whitelist。
	Channel string

	// IdempotencyKey 保证最多执行一次。
	IdempotencyKey string

	// OperatorID 标识发起划拨的用户或服务。
	OperatorID string

	// Reason 是可选的业务事由。
	Reason *string
}

// AllocateResult 承载成功划拨的结果。
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

// FreezeRequest 承载模型调用前预留资金的参数。
type FreezeRequest struct {
	// AccountID 标识余额将被冻结的账户。
	AccountID string

	// Amount 是需要冻结的资金数量。应覆盖所有候选路由的最大预估售价。
	Amount Decimal

	// EstimatedSell 是所有候选路由的最大预估售价预计算值。
	EstimatedSell Decimal

	// RequestID 将此冻结关联到特定的 API 调用请求。
	RequestID string

	// UserID 标识发起调用的终端用户。
	UserID string

	// APIKeyID 标识调用所用的网关密钥。
	APIKeyID *string

	// TTL 是冻结持续时间，超时自动过期。默认 15 分钟。
	TTL time.Duration
}

// FreezeResult 承载成功冻结的结果。
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

// SettleRequest 承载在已知实际用量后完成冻结的参数。
type SettleRequest struct {
	// FreezeID 标识待结算的冻结。
	FreezeID string

	// ActualSell 是基于实际用量的最终售价。
	ActualSell Decimal

	// ActualCost 是基于实际用量的最终成本。
	ActualCost Decimal

	// RequestID 将此结算关联到原始调用。
	RequestID string
}

// SettleResult 承载成功结算的结果。
type SettleResult struct {
	FreezeID       string  `json:"freeze_id"`
	ActualSell     Decimal `json:"actual_sell"`
	ActualCost     Decimal `json:"actual_cost"`
	ReleasedAmount Decimal `json:"released_amount"` // 冻结金额 - 实际售价（退款）
	BalanceAfter   Decimal `json:"balance_after"`
	FrozenAfter    Decimal `json:"frozen_after"`
}

// LiquidateRequest 承载启动或推进账户清算的参数。
type LiquidateRequest struct {
	// AccountID 标识待清算的账户。
	AccountID string

	// TargetAccountID 是排空后接收剩余资金的账户。
	TargetAccountID string

	// OperatorID 标识启动清算的用户。
	OperatorID string

	// PartyID 标识拥有该账户的组织。
	PartyID string

	// Reason 是必填的业务事由（被审计）。
	Reason string
}

// LiquidateResult 承载清算步骤的结果。
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
