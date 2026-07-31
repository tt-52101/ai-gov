package fund

import (
	"context"

	"github.com/shopspring/decimal"
)

// Tx 表示一个数据库事务。实现通常包装 *gorm.DB。
// 所有变更性 Store 方法都接受 Tx 以确保原子性。
type Tx interface {
	// Commit 提交事务。若底层数据库提交失败则返回错误。
	Commit() error

	// Rollback 中止事务。可安全地多次调用。
	Rollback() error
}

// Store 定义 fund 领域的持久化契约。
// 所有货币变更必须通过此接口——绝不可从服务逻辑中直接调用 GORM（AGENTS.md 治理规则）。
//
// 实现预期底层使用 GORM，但接口将服务与 ORM 细节隔离，使得可以使用内存假体进行清晰测试，
// 并且可以方便地切换数据库。
type Store interface {
	// WithTx 在单个数据库事务中执行 fn。
	// 若 fn 返回错误则回滚事务，否则提交。
	WithTx(ctx context.Context, fn func(tx Tx) error) error

	// -----------------------------------------------------------------------
	// 账户操作
	// -----------------------------------------------------------------------

	// GetAccount 按主键检索账户。
	// 当账户不存在时返回 nil, nil。
	GetAccount(ctx context.Context, id string) (*Account, error)

	// GetAccountForUpdate 以行级锁检索账户（SELECT ... FOR UPDATE）。
	// 必须在事务内调用。返回账户的 Version 字段用于乐观并发控制。
	GetAccountForUpdate(tx Tx, ctx context.Context, id string) (*Account, error)

	// UpdateAccountBalances 原子更新 available_balance 和 frozen_balance，递增 version。
	// version 参数是先前读取的版本；若与数据库行不匹配则更新失败（乐观锁）。
	UpdateAccountBalances(tx Tx, ctx context.Context, id string, available, frozen decimal.Decimal, version int64) error

	// UpdateAccountStatus 设置账户状态。version 参数强制执行乐观锁。
	UpdateAccountStatus(tx Tx, ctx context.Context, id string, status string, version int64) error

	// UpdateAccountBudgetConsumed 按 delta 递增 budget_consumed_amount。
	UpdateAccountBudgetConsumed(tx Tx, ctx context.Context, id string, delta decimal.Decimal) error

	// -----------------------------------------------------------------------
	// 账本（追加不可变）
	// -----------------------------------------------------------------------

	// InsertLedger 追加一条账本记录。账本行永不更新或删除——这是追加不可变日志（F-CON-01）。
	InsertLedger(tx Tx, ctx context.Context, entry *Ledger) error

	// -----------------------------------------------------------------------
	// 冻结操作
	// -----------------------------------------------------------------------

	// InsertFreeze 创建新的冻结记录。冻结状态必须为 "open"。
	InsertFreeze(tx Tx, ctx context.Context, f *Freeze) error

	// GetFreeze 按主键检索冻结记录。
	// 当冻结记录不存在时返回 nil, nil。
	GetFreeze(ctx context.Context, freezeID string) (*Freeze, error)

	// GetFreezeForUpdate 以行级锁检索冻结记录（SELECT ... FOR UPDATE）。
	// 必须在事务内调用。用于 Settle 操作以防止同一 freeze_id 的并发结算（RED-2 竞态漏洞修复）。
	GetFreezeForUpdate(tx Tx, ctx context.Context, freezeID string) (*Freeze, error)

	// UpdateFreezeStatus 设置冻结状态，并可选择更新 settle_amount、settle_cost 和 settled_at 用于结算。
	UpdateFreezeStatus(tx Tx, ctx context.Context, freezeID string, status string, settleAmount, settleCost *decimal.Decimal) error

	// RenewFreeze 延长 expires_at 并递增 renewal_count。
	// 返回受影响的行数（0 表示冻结未找到或非 open 状态）。
	RenewFreeze(tx Tx, ctx context.Context, freezeID string, newExpiresAt string) (int64, error)

	// ListExpiredFreezes 返回 expires_at 在当前时间之前且状态为 open 的冻结记录，最多 limit 行。
	// 由 TTL 扫描器后台任务使用。
	ListExpiredFreezes(ctx context.Context, limit int) ([]*Freeze, error)

	// -----------------------------------------------------------------------
	// 划拨
	// -----------------------------------------------------------------------

	// InsertAllocation 创建划拨记录。
	InsertAllocation(tx Tx, ctx context.Context, a *Allocation) error

	// UpdateAllocationStatus 设置划拨状态和 completed_at 时间戳。
	UpdateAllocationStatus(tx Tx, ctx context.Context, id string, status string) error

	// -----------------------------------------------------------------------
	// 清算
	// -----------------------------------------------------------------------

	// GetLiquidation 检索账户的活跃清算记录。
	// 当不存在清算记录时返回 nil, nil。
	GetLiquidation(ctx context.Context, accountID string) (*Liquidation, error)

	// InsertLiquidation 创建新的清算记录。
	InsertLiquidation(tx Tx, ctx context.Context, l *Liquidation) error

	// UpdateLiquidationStage 推进清算状态并更新元数据。
	UpdateLiquidationStage(tx Tx, ctx context.Context, id string, stage string) error

	// -----------------------------------------------------------------------
	// 幂等
	// -----------------------------------------------------------------------

	// CheckIdempotency 按幂等键查找先前的结果。
	// 若找到则返回存储的结果和 true，否则返回 nil, false。
	CheckIdempotency(ctx context.Context, key string) (*AllocateResult, bool, error)

	// StoreIdempotency 持久化幂等结果以供将来查找。
	// 必须在其保护的同一次划拨事务内调用。
	StoreIdempotency(tx Tx, ctx context.Context, key string, result *AllocateResult) error
}
