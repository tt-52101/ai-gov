package fund

import (
	"errors"
	"fmt"
)

// 本文件定义 fund 领域的所有哨兵错误和构造函数。
// 每个错误携带上下文信息以辅助诊断。调用方应使用 errors.Is 检查错误标识，
// 使用 errors.As 提取结构化 FundError 用于日志记录或 API 响应。

// FundError 是一个结构化领域错误，携带错误码、人类可读消息和原始哨兵错误以供标识检查。
type FundError struct {
	Code    string
	Message string
	Err     error
}

// Error 实现 error 接口。
func (e *FundError) Error() string {
	if e.Message != "" {
		return e.Code + ": " + e.Message
	}
	return e.Code
}

// Unwrap 支持 errors.Is / errors.As 匹配哨兵错误。
func (e *FundError) Unwrap() error {
	return e.Err
}

// ---------------------------------------------------------------------------
// 哨兵错误
// ---------------------------------------------------------------------------

// ErrInsufficientBalance 在可用余额低于请求的冻结或划拨金额时返回。
var ErrInsufficientBalance = errors.New("可用余额不足")

// ErrAccountFrozen 在操作目标账户状态非活跃时返回（如清算中、已冻结、已关闭）。
var ErrAccountFrozen = errors.New("账户非活跃状态")

// ErrBudgetCapExceeded 在 budget_consumed + 预估成本超出预算上限时返回。
var ErrBudgetCapExceeded = errors.New("预算上限已超出")

// ErrFreezeExpired 在尝试结算一个已过期的冻结时返回。
var ErrFreezeExpired = errors.New("冻结已过期")

// ErrFreezeNotFound 在 freeze_id 不存在时返回。
var ErrFreezeNotFound = errors.New("冻结记录未找到")

// ErrAllocationChannelDenied 在请求的划拨通道不允许在给定源/目标账户之间操作时返回。
var ErrAllocationChannelDenied = errors.New("划拨通道被拒绝")

// ErrIdempotencyConflict 在同一个幂等键被用于不同请求体时返回（相同键，不同哈希）。
var ErrIdempotencyConflict = errors.New("幂等冲突：相同键，不同请求")

// ErrLiquidationStageInvalid 在当前清算阶段不允许请求的阶段转换时返回。
var ErrLiquidationStageInvalid = errors.New("无效的清算阶段转换")

// ErrSelfTransfer 在划拨的源账户与目标账户相同时返回。
var ErrSelfTransfer = errors.New("不能向同一账户划拨")

// ErrAmountMustBePositive 在金额为零或负数时返回。
var ErrAmountMustBePositive = errors.New("金额必须为正数")

// ErrIdempotencyKeyRequired 在调用方未提供 IdempotencyKey 时返回。
// 所有划拨操作必须提供幂等键以保证资金安全——无例外。
var ErrIdempotencyKeyRequired = errors.New("所有资金操作必须提供幂等键")

// ---------------------------------------------------------------------------
// 构造函数——将哨兵错误包装为带上下文信息的 FundError。
// ---------------------------------------------------------------------------

// newInsufficientBalanceError 构造携带有账户和金额上下文的余额不足错误。
func newInsufficientBalanceError(accountID string, available, requested Decimal) *FundError {
	return &FundError{
		Code:    "INSUFFICIENT_BALANCE",
		Message: fmt.Sprintf("账户 %s 可用余额 %s，请求金额 %s", accountID, available.String(), requested.String()),
		Err:     ErrInsufficientBalance,
	}
}

// newAccountFrozenError 构造携带账户和状态上下文的账户冻结错误。
func newAccountFrozenError(accountID string, status string) *FundError {
	return &FundError{
		Code:    "ACCOUNT_FROZEN_OR_CLOSED",
		Message: fmt.Sprintf("账户 %s 状态为 %s", accountID, status),
		Err:     ErrAccountFrozen,
	}
}

// newBudgetCapExceededError 构造携带预算消耗和限制上下文的预算超限错误。
func newBudgetCapExceededError(accountID string, consumed, limit, estimate Decimal) *FundError {
	return &FundError{
		Code:    "BUDGET_CAP_EXCEEDED",
		Message: fmt.Sprintf("账户 %s 预算：已消耗 %s + 预估 %s > 上限 %s", accountID, consumed.String(), estimate.String(), limit.String()),
		Err:     ErrBudgetCapExceeded,
	}
}

// newFreezeExpiredError 构造携带 freeze_id 上下文的冻结过期错误。
func newFreezeExpiredError(freezeID string) *FundError {
	return &FundError{
		Code:    "FREEZE_EXPIRED",
		Message: fmt.Sprintf("冻结 %s 已过期", freezeID),
		Err:     ErrFreezeExpired,
	}
}

// newFreezeNotFoundError 构造携带 freeze_id 上下文的冻结未找到错误。
func newFreezeNotFoundError(freezeID string) *FundError {
	return &FundError{
		Code:    "FREEZE_NOT_FOUND",
		Message: fmt.Sprintf("冻结 %s 未找到", freezeID),
		Err:     ErrFreezeNotFound,
	}
}

// newAllocationChannelDeniedError 构造携带源、目标和通道信息的通道拒绝错误。
func newAllocationChannelDeniedError(src, dst, channel string) *FundError {
	return &FundError{
		Code:    "ALLOCATION_CHANNEL_DENIED",
		Message: fmt.Sprintf("通道 %s 不允许从 %s 到 %s 的划拨", channel, src, dst),
		Err:     ErrAllocationChannelDenied,
	}
}

// newIdempotencyConflictError 构造携带幂等键上下文的冲突错误。
func newIdempotencyConflictError(key string) *FundError {
	return &FundError{
		Code:    "IDEMPOTENCY_CONFLICT",
		Message: fmt.Sprintf("幂等键 %s 已被不同参数使用", key),
		Err:     ErrIdempotencyConflict,
	}
}

// newLiquidationStageInvalidError 构造携带账户和阶段转换上下文的清算阶段错误。
func newLiquidationStageInvalidError(accountID string, current, target string) *FundError {
	return &FundError{
		Code:    "LIQUIDATION_STAGE_INVALID",
		Message: fmt.Sprintf("账户 %s 不能从 %s 转换到 %s", accountID, current, target),
		Err:     ErrLiquidationStageInvalid,
	}
}

// newSelfTransferError 构造携带账户上下文的自我划拨错误。
func newSelfTransferError(accountID string) *FundError {
	return &FundError{
		Code:    "SELF_TRANSFER",
		Message: fmt.Sprintf("不能从账户 %s 划拨到自身", accountID),
		Err:     ErrSelfTransfer,
	}
}

// newAmountMustBePositiveError 构造金额非正数错误。
func newAmountMustBePositiveError(amount Decimal) *FundError {
	return &FundError{
		Code:    "AMOUNT_MUST_BE_POSITIVE",
		Message: fmt.Sprintf("金额必须为正数，实际值为 %s", amount.String()),
		Err:     ErrAmountMustBePositive,
	}
}

// newIdempotencyKeyRequiredError 构造 IdempotencyKey 缺失错误。
// 所有划拨操作必须提供幂等键——无例外（RED-2）。
func newIdempotencyKeyRequiredError() *FundError {
	return &FundError{
		Code:    "IDEMPOTENCY_KEY_REQUIRED",
		Message: "所有划拨操作必须提供 IdempotencyKey 以保证幂等安全",
		Err:     ErrIdempotencyKeyRequired,
	}
}
