package fund

import (
	"errors"
	"fmt"
)

// Sentinel errors for the fund domain. Each error wraps contextual information
// to aid diagnostics. Callers should use errors.Is to check error identity
// and errors.As to extract the structured FundError for logging or API responses.

// FundError is a structured domain error carrying an error code, a human-readable
// message, and the original sentinel error for identity checks.
type FundError struct {
	Code    string
	Message string
	Err     error
}

// Error implements the error interface.
func (e *FundError) Error() string {
	if e.Message != "" {
		return e.Code + ": " + e.Message
	}
	return e.Code
}

// Unwrap enables errors.Is / errors.As to match against the sentinel.
func (e *FundError) Unwrap() error {
	return e.Err
}

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

// ErrInsufficientBalance is returned when the available balance is less than
// the requested freeze or transfer amount.
var ErrInsufficientBalance = errors.New("insufficient available balance")

// ErrAccountFrozen is returned when an operation targets an account whose status
// is not active (e.g. liquidating, frozen, or closed).
var ErrAccountFrozen = errors.New("account is not active")

// ErrBudgetCapExceeded is returned when budget_consumed + estimated cost exceeds
// the configured budget_limit_amount.
var ErrBudgetCapExceeded = errors.New("budget cap exceeded")

// ErrFreezeExpired is returned when attempting to settle a freeze whose
// expires_at has passed.
var ErrFreezeExpired = errors.New("freeze has expired")

// ErrFreezeNotFound is returned when a freeze_id does not correspond to
// any existing freeze record.
var ErrFreezeNotFound = errors.New("freeze not found")

// ErrAllocationChannelDenied is returned when the requested transfer channel
// is not permitted for the given source/destination pair.
var ErrAllocationChannelDenied = errors.New("allocation channel denied")

// ErrIdempotencyConflict is returned when an idempotency key is reused with
// a different request body (same key, different hash).
var ErrIdempotencyConflict = errors.New("idempotency conflict: same key, different request")

// ErrLiquidationStageInvalid is returned when the requested liquidation
// stage transition is not valid from the current stage.
var ErrLiquidationStageInvalid = errors.New("invalid liquidation stage transition")

// ErrSelfTransfer is returned when an allocation source equals destination.
var ErrSelfTransfer = errors.New("cannot transfer to the same account")

// ErrAmountMustBePositive is returned when amount is zero or negative.
var ErrAmountMustBePositive = errors.New("amount must be positive")

// ---------------------------------------------------------------------------
// Constructor helpers — wrap a sentinel with contextual details.
// ---------------------------------------------------------------------------

// newInsufficientBalanceError constructs an ErrInsufficientBalance with a message.
func newInsufficientBalanceError(accountID string, available, requested Decimal) *FundError {
	return &FundError{
		Code:    "INSUFFICIENT_BALANCE",
		Message: fmt.Sprintf("account %s has %s available, requested %s", accountID, available.String(), requested.String()),
		Err:     ErrInsufficientBalance,
	}
}

// newAccountFrozenError constructs an ErrAccountFrozen with a message.
func newAccountFrozenError(accountID string, status string) *FundError {
	return &FundError{
		Code:    "ACCOUNT_FROZEN_OR_CLOSED",
		Message: fmt.Sprintf("account %s is %s", accountID, status),
		Err:    ErrAccountFrozen,
	}
}

// newBudgetCapExceededError constructs an ErrBudgetCapExceeded with a message.
func newBudgetCapExceededError(accountID string, consumed, limit, estimate Decimal) *FundError {
	return &FundError{
		Code:    "BUDGET_CAP_EXCEEDED",
		Message: fmt.Sprintf("account %s budget: consumed %s + estimate %s > limit %s", accountID, consumed.String(), estimate.String(), limit.String()),
		Err:     ErrBudgetCapExceeded,
	}
}

// newFreezeExpiredError constructs an ErrFreezeExpired with a message.
func newFreezeExpiredError(freezeID string) *FundError {
	return &FundError{
		Code:    "FREEZE_EXPIRED",
		Message: fmt.Sprintf("freeze %s has expired", freezeID),
		Err:     ErrFreezeExpired,
	}
}

// newFreezeNotFoundError constructs an ErrFreezeNotFound with a message.
func newFreezeNotFoundError(freezeID string) *FundError {
	return &FundError{
		Code:    "FREEZE_NOT_FOUND",
		Message: fmt.Sprintf("freeze %s not found", freezeID),
		Err:     ErrFreezeNotFound,
	}
}

// newAllocationChannelDeniedError constructs an ErrAllocationChannelDenied with a message.
func newAllocationChannelDeniedError(src, dst, channel string) *FundError {
	return &FundError{
		Code:    "ALLOCATION_CHANNEL_DENIED",
		Message: fmt.Sprintf("channel %s not permitted from %s to %s", channel, src, dst),
		Err:     ErrAllocationChannelDenied,
	}
}

// newIdempotencyConflictError constructs an ErrIdempotencyConflict with a message.
func newIdempotencyConflictError(key string) *FundError {
	return &FundError{
		Code:    "IDEMPOTENCY_CONFLICT",
		Message: fmt.Sprintf("idempotency key %s already used with different parameters", key),
		Err:     ErrIdempotencyConflict,
	}
}

// newLiquidationStageInvalidError constructs an ErrLiquidationStageInvalid with a message.
func newLiquidationStageInvalidError(accountID string, current, target string) *FundError {
	return &FundError{
		Code:    "LIQUIDATION_STAGE_INVALID",
		Message: fmt.Sprintf("account %s cannot transition from %s to %s", accountID, current, target),
		Err:     ErrLiquidationStageInvalid,
	}
}

// newSelfTransferError constructs an ErrSelfTransfer with a message.
func newSelfTransferError(accountID string) *FundError {
	return &FundError{
		Code:    "SELF_TRANSFER",
		Message: fmt.Sprintf("cannot transfer from account %s to itself", accountID),
		Err:     ErrSelfTransfer,
	}
}

// newAmountMustBePositiveError constructs an ErrAmountMustBePositive with a message.
func newAmountMustBePositiveError(amount Decimal) *FundError {
	return &FundError{
		Code:    "AMOUNT_MUST_BE_POSITIVE",
		Message: fmt.Sprintf("amount must be positive, got %s", amount.String()),
		Err:     ErrAmountMustBePositive,
	}
}
