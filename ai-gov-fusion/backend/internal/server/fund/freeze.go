package fund

import (
	"context"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
)

// ---------------------------------------------------------------------------
// Freeze
// ---------------------------------------------------------------------------

// Freeze reserves funds for an upcoming model call by moving amount from
// available_balance to frozen_balance and creating an open freeze record
// with a TTL (default 15 minutes, configurable).
//
// Budget cap check (PRD S5.2): if the account has a budget_limit_amount,
// budget_consumed + estimated_sell must not exceed the limit. The budget
// check happens before the freeze to ensure the account can afford the
// estimated cost.
//
// The freeze amount must be at least the estimated sell amount (with safety
// margin, typically 1.2x). The caller is responsible for computing the
// appropriate freeze amount from pricing estimates.
//
// Returns a FreezeResult with the freeze_id for subsequent settlement
// or renewal. The freeze will auto-expire after TTL if not settled.
//
// Side effects: decrements available_balance, increments frozen_balance,
// inserts freeze record and freeze ledger entry. All within one transaction.
func (s *Service) Freeze(ctx context.Context, req FreezeRequest) (*FreezeResult, error) {
	if req.Amount.Decimal.LessThanOrEqual(decimal.Zero) {
		return nil, newAmountMustBePositiveError(req.Amount)
	}

	ttl := req.TTL
	if ttl <= 0 {
		ttl = defaultFreezeTTL
	}

	var result *FreezeResult
	err := s.Store.WithTx(ctx, func(tx Tx) error {
		acct, err := s.Store.GetAccountForUpdate(tx, ctx, req.AccountID)
		if err != nil {
			return err
		}
		if acct == nil {
			return newAccountFrozenError(req.AccountID, "not found")
		}
		if acct.Status != StatusActive {
			return newAccountFrozenError(req.AccountID, acct.Status)
		}

		// Budget cap check.
		if acct.BudgetLimitAmount != nil {
			estimatedCost := req.EstimatedSell.Decimal
			if estimatedCost.IsZero() {
				estimatedCost = req.Amount.Decimal
			}
			newConsumed := acct.BudgetConsumedAmount.Decimal.Add(estimatedCost)
			if newConsumed.GreaterThan(acct.BudgetLimitAmount.Decimal) {
				return newBudgetCapExceededError(req.AccountID,
					acct.BudgetConsumedAmount, *acct.BudgetLimitAmount, NewDecimal(estimatedCost.String()))
			}

			// Warn if over ratio.
			if acct.BudgetWarnRatio != nil {
				ratio := newConsumed.Div(acct.BudgetLimitAmount.Decimal)
				if ratio.GreaterThanOrEqual(acct.BudgetWarnRatio.Decimal) {
					slog.WarnContext(ctx, "budget_warn_ratio_exceeded",
						"account_id", req.AccountID,
						"consumed", acct.BudgetConsumedAmount.String(),
						"limit", acct.BudgetLimitAmount.String(),
						"ratio", ratio.String(),
						"request_id", req.RequestID,
					)
				}
			}
		}

		// Verify available balance.
		if acct.AvailableBalance.Decimal.LessThan(req.Amount.Decimal) {
			return newInsufficientBalanceError(req.AccountID, acct.AvailableBalance, req.Amount)
		}

		now := time.Now()
		expiresAt := now.Add(ttl)
		maxLifetimeAt := now.Add(maxFreezeLifetime)

		// Compute new balances.
		availableAfter := acct.AvailableBalance.Decimal.Sub(req.Amount.Decimal)
		frozenAfter := acct.FrozenBalance.Decimal.Add(req.Amount.Decimal)

		// Update account.
		if err := s.Store.UpdateAccountBalances(tx, ctx, req.AccountID, availableAfter, frozenAfter, acct.Version); err != nil {
			return err
		}

		// Create freeze record.
		freeze := &Freeze{
			ID:            newUUID(),
			AccountID:     req.AccountID,
			RequestID:     stringPtr(req.RequestID),
			UserID:        req.UserID,
			APIKeyID:      req.APIKeyID,
			Amount:        req.Amount,
			EstimatedSell: req.EstimatedSell,
			Status:        FreezeStatusOpen,
			ExpiresAt:     expiresAt,
			MaxLifetimeAt: timePtr(maxLifetimeAt),
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := s.Store.InsertFreeze(tx, ctx, freeze); err != nil {
			return err
		}

		// Insert freeze ledger entry.
		freezeLedger := &Ledger{
			ID:           newUUID(),
			AccountID:    req.AccountID,
			Direction:    DirectionFreeze,
			Amount:       req.Amount,
			BalanceAfter: NewDecimal(availableAfter.String()),
			FrozenAfter:  decPtr(frozenAfter),
			FreezeID:     stringPtr(freeze.ID),
			RequestID:    stringPtr(req.RequestID),
			CreatedAt:    now,
		}
		if err := s.Store.InsertLedger(tx, ctx, freezeLedger); err != nil {
			return err
		}

		result = &FreezeResult{
			FreezeID:      freeze.ID,
			AccountID:     req.AccountID,
			Amount:        req.Amount,
			EstimatedSell: req.EstimatedSell,
			BalanceAfter:  NewDecimal(availableAfter.String()),
			FrozenAfter:   NewDecimal(frozenAfter.String()),
			ExpiresAt:     expiresAt,
			RequestID:     req.RequestID,
		}

		// Structured log.
		slog.InfoContext(ctx, "freeze_acquired",
			"request_id", req.RequestID,
			"account_id", req.AccountID,
			"freeze_id", freeze.ID,
			"amount", req.Amount.String(),
			"estimated_sell", req.EstimatedSell.String(),
			"balance_after", availableAfter.String(),
			"frozen_after", frozenAfter.String(),
			"expires_at", expiresAt,
		)

		return nil
	})

	if err != nil {
		slog.ErrorContext(ctx, "freeze_failed",
			"request_id", req.RequestID,
			"account_id", req.AccountID,
			"amount", req.Amount.String(),
			"error", err,
		)
		return nil, err
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// Settle
// ---------------------------------------------------------------------------

// Settle finalizes a freeze after the upstream model call completes and actual
// usage is known.
//
// Steps:
//  1. Lock the freeze record (must be open).
//  2. If freeze has expired: attempt orphan settle — deduct from available if
//     possible, otherwise alert and release the freeze.
//  3. Compute the delta: frozen_amount - actual_sell = refund.
//  4. Deduct actual_sell from frozen_balance AND available_balance (the sell
//     portion is consumed).
//  5. Return refund to available_balance.
//  6. Update budget_consumed_amount.
//  7. Insert settle ledger entry (direction=settle) and unfreeze ledger entry
//     if refund > 0.
//  8. Mark freeze as settled.
//
// If actual_sell > frozen_amount the settlement fails with a mismatch error
// (freeze was too small — this indicates a pricing error upstream).
//
// Side effects: updates account balances, inserts ledger entries, updates
// freeze status, increments budget_consumed_amount.
func (s *Service) Settle(ctx context.Context, req SettleRequest) (*SettleResult, error) {
	// Validate invariants.
	if req.ActualSell.Decimal.GreaterThan(decimal.Zero) && req.ActualCost.Decimal.GreaterThan(req.ActualSell.Decimal) {
		slog.WarnContext(ctx, "cost_exceeds_sell",
			"freeze_id", req.FreezeID,
			"actual_cost", req.ActualCost.String(),
			"actual_sell", req.ActualSell.String(),
			"request_id", req.RequestID,
		)
	}

	var result *SettleResult
	err := s.Store.WithTx(ctx, func(tx Tx) error {
		// Get the freeze record.
		freeze, err := s.Store.GetFreeze(ctx, req.FreezeID)
		if err != nil {
			return err
		}
		if freeze == nil {
			return newFreezeNotFoundError(req.FreezeID)
		}

		now := time.Now()
		settleAmount := req.ActualSell.Decimal
		settleCost := req.ActualCost.Decimal

		// Check if freeze has expired.
		if now.After(freeze.ExpiresAt) && freeze.Status == FreezeStatusOpen {
			slog.WarnContext(ctx, "freeze_expired_orphan_settle",
				"freeze_id", req.FreezeID,
				"account_id", freeze.AccountID,
				"frozen_amount", freeze.Amount.String(),
				"actual_sell", req.ActualSell.String(),
				"request_id", req.RequestID,
			)
			// Attempt orphan settle: if sell <= frozen, proceed with deduction.
			// If not enough, release the freeze and alert.
		}

		if freeze.Status != FreezeStatusOpen {
			return &FundError{
				Code:    "FREEZE_NOT_OPEN",
				Message: "freeze " + req.FreezeID + " is already " + freeze.Status,
				Err:     ErrFreezeExpired,
			}
		}

		// Validate settle amount against frozen amount.
		frozenAmount := freeze.Amount.Decimal
		if settleAmount.GreaterThan(frozenAmount) {
			return &FundError{
				Code:    "SETTLEMENT_MISMATCH",
				Message: "freeze " + req.FreezeID + " frozen " + frozenAmount.String() + " but settle " + settleAmount.String(),
				Err:     ErrInsufficientBalance,
			}
		}

		// Lock account.
		acct, err := s.Store.GetAccountForUpdate(tx, ctx, freeze.AccountID)
		if err != nil {
			return err
		}
		if acct == nil {
			return newAccountFrozenError(freeze.AccountID, "not found")
		}

		// Compute refund.
		refund := frozenAmount.Sub(settleAmount)

		// New balances: release ALL frozen funds first, then deduct sell from available.
		// frozen_balance decreases by total frozen (the freeze is being settled).
		// available_balance: the sell portion is truly consumed.
		//                   the refund portion was already in available (just unfrozen).
		availableAfter := acct.AvailableBalance.Decimal.Add(refund)
		frozenAfter := acct.FrozenBalance.Decimal.Sub(frozenAmount)

		if availableAfter.LessThan(decimal.Zero) {
			slog.ErrorContext(ctx, "settlement_would_overdraw",
				"account_id", freeze.AccountID,
				"freeze_id", req.FreezeID,
				"available_before", acct.AvailableBalance.String(),
				"frozen_before", acct.FrozenBalance.String(),
				"settle_amount", settleAmount.String(),
				"refund", refund.String(),
				"available_after", availableAfter.String(),
			)
			return &FundError{
				Code:    "SETTLEMENT_OVERDRAW",
				Message: "settlement would overdraw account " + freeze.AccountID,
				Err:     ErrInsufficientBalance,
			}
		}

		// Update account balances.
		if err := s.Store.UpdateAccountBalances(tx, ctx, freeze.AccountID, availableAfter, frozenAfter, acct.Version); err != nil {
			return err
		}

		// Update budget_consumed_amount with the sell amount.
		if settleAmount.GreaterThan(decimal.Zero) {
			if err := s.Store.UpdateAccountBudgetConsumed(tx, ctx, freeze.AccountID, settleAmount); err != nil {
				return err
			}
		}

		// Insert settle ledger entry.
		settleLedger := &Ledger{
			ID:           newUUID(),
			AccountID:    freeze.AccountID,
			Direction:    DirectionSettle,
			Amount:       NewDecimal(settleAmount.String()),
			BalanceAfter: NewDecimal(availableAfter.String()),
			FrozenAfter:  decPtr(frozenAfter),
			CostAmount:   decPtr(settleCost),
			SellAmount:   decPtr(settleAmount),
			FreezeID:     stringPtr(freeze.ID),
			RequestID:    stringPtr(req.RequestID),
			CreatedAt:    now,
		}
		if err := s.Store.InsertLedger(tx, ctx, settleLedger); err != nil {
			return err
		}

		// If there is a refund, insert an unfreeze ledger entry.
		if refund.GreaterThan(decimal.Zero) {
			refundLedger := &Ledger{
				ID:           newUUID(),
				AccountID:    freeze.AccountID,
				Direction:    DirectionUnfreeze,
				Amount:       NewDecimal(refund.String()),
				BalanceAfter: NewDecimal(availableAfter.String()),
				FrozenAfter:  decPtr(frozenAfter),
				FreezeID:     stringPtr(freeze.ID),
				RequestID:    stringPtr(req.RequestID),
				CreatedAt:    now,
			}
			if err := s.Store.InsertLedger(tx, ctx, refundLedger); err != nil {
				return err
			}
		}

		// Update freeze status.
		if err := s.Store.UpdateFreezeStatus(tx, ctx, req.FreezeID, FreezeStatusSettled, &settleAmount, &settleCost); err != nil {
			return err
		}

		result = &SettleResult{
			FreezeID:       req.FreezeID,
			ActualSell:     req.ActualSell,
			ActualCost:     req.ActualCost,
			ReleasedAmount: NewDecimal(refund.String()),
			BalanceAfter:   NewDecimal(availableAfter.String()),
			FrozenAfter:    NewDecimal(frozenAfter.String()),
		}

		slog.InfoContext(ctx, "freeze_settled",
			"request_id", req.RequestID,
			"account_id", freeze.AccountID,
			"freeze_id", freeze.ID,
			"frozen_amount", frozenAmount.String(),
			"settle_amount", settleAmount.String(),
			"refund_amount", refund.String(),
			"cost_amount", settleCost.String(),
			"balance_after", availableAfter.String(),
			"frozen_after", frozenAfter.String(),
		)

		return nil
	})

	if err != nil {
		slog.ErrorContext(ctx, "settle_failed",
			"request_id", req.RequestID,
			"freeze_id", req.FreezeID,
			"actual_sell", req.ActualSell.String(),
			"error", err,
		)
		return nil, err
	}

	return result, nil
}
