package fund

import (
	"context"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
)

// ---------------------------------------------------------------------------
// RenewFreeze
// ---------------------------------------------------------------------------

// RenewFreeze extends the expiry of an open freeze for streaming calls.
//
// It does NOT increase the frozen amount. It extends expires_at by the
// default TTL and increments renewal_count. The total cumulative lifetime
// is capped at max_lifetime_at (2 hours from original freeze).
//
// If the freeze is not open or has already exceeded max_lifetime_at,
// RenewFreeze returns an error.
//
// Side effects: updates freeze.expires_at, freeze.renewal_count,
// and freeze.last_renewed_at.
func (s *Service) RenewFreeze(ctx context.Context, freezeID string) error {
	err := s.Store.WithTx(ctx, func(tx Tx) error {
		freeze, err := s.Store.GetFreeze(ctx, freezeID)
		if err != nil {
			return err
		}
		if freeze == nil {
			return newFreezeNotFoundError(freezeID)
		}
		if freeze.Status != FreezeStatusOpen {
			return &FundError{
				Code:    "FREEZE_NOT_OPEN",
				Message: "freeze " + freezeID + " is " + freeze.Status,
				Err:     ErrFreezeExpired,
			}
		}

		now := time.Now()
		if now.After(freeze.ExpiresAt) {
			return newFreezeExpiredError(freezeID)
		}

		newExpires := now.Add(defaultFreezeTTL)
		if freeze.MaxLifetimeAt != nil && newExpires.After(*freeze.MaxLifetimeAt) {
			slog.WarnContext(ctx, "freeze_renewal_capped",
				"freeze_id", freezeID,
				"account_id", freeze.AccountID,
				"requested_expires", newExpires,
				"max_lifetime", freeze.MaxLifetimeAt,
			)
			return &FundError{
				Code:    "FREEZE_MAX_LIFETIME",
				Message: "freeze " + freezeID + " has reached max lifetime",
				Err:     ErrFreezeExpired,
			}
		}

		newExpiresStr := newExpires.Format(time.RFC3339Nano)
		rowsAffected, err := s.Store.RenewFreeze(tx, ctx, freezeID, newExpiresStr)
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return newFreezeNotFoundError(freezeID)
		}

		slog.InfoContext(ctx, "freeze_renewed",
			"freeze_id", freezeID,
			"account_id", freeze.AccountID,
			"amount", freeze.Amount.String(),
			"new_expires_at", newExpires,
			"renewal_count", freeze.RenewalCount+1,
		)

		return nil
	})

	if err != nil {
		slog.ErrorContext(ctx, "freeze_renewal_failed",
			"freeze_id", freezeID,
			"error", err,
		)
		return err
	}

	return nil
}

// ---------------------------------------------------------------------------
// UnfreezeTimeout
// ---------------------------------------------------------------------------

// UnfreezeTimeout releases expired open freezes. This is called by the
// background TTL scanner worker (operations plane).
//
// It scans for open freezes past expires_at (up to limit rows), and for each:
//   - Returns the frozen amount to available_balance
//   - Decrements frozen_balance
//   - Inserts an unfreeze ledger entry
//   - Marks the freeze as timeout_released
//
// Returns the number of freezes released.
//
// Each freeze release runs in its own transaction to minimise lock contention.
// Individual failures are logged and skipped — the scanner continues processing
// remaining freezes.
func (s *Service) UnfreezeTimeout(ctx context.Context) (int, error) {
	freezes, err := s.Store.ListExpiredFreezes(ctx, 100)
	if err != nil {
		return 0, err
	}

	released := 0
	for _, freeze := range freezes {
		releaseErr := s.Store.WithTx(ctx, func(tx Tx) error {
			acct, err := s.Store.GetAccountForUpdate(tx, ctx, freeze.AccountID)
			if err != nil {
				return err
			}
			if acct == nil {
				return nil
			}

			amount := freeze.Amount.Decimal
			availableAfter := acct.AvailableBalance.Decimal.Add(amount)
			frozenAfter := acct.FrozenBalance.Decimal.Sub(amount)

			if err := s.Store.UpdateAccountBalances(tx, ctx, freeze.AccountID, availableAfter, frozenAfter, acct.Version); err != nil {
				return err
			}

			// Insert unfreeze ledger entry.
			now := time.Now()
			ledger := &Ledger{
				ID:           newUUID(),
				AccountID:    freeze.AccountID,
				Direction:    DirectionUnfreeze,
				Amount:       NewDecimal(amount.String()),
				BalanceAfter: NewDecimal(availableAfter.String()),
				FrozenAfter:  decPtr(frozenAfter),
				FreezeID:     stringPtr(freeze.ID),
				CreatedAt:    now,
			}
			if err := s.Store.InsertLedger(tx, ctx, ledger); err != nil {
				return err
			}

			var zero decimal.Decimal
			if err := s.Store.UpdateFreezeStatus(tx, ctx, freeze.ID, FreezeStatusTimeoutReleased, &zero, &zero); err != nil {
				return err
			}

			return nil
		})

		if releaseErr != nil {
			slog.ErrorContext(ctx, "unfreeze_timeout_failed",
				"freeze_id", freeze.ID,
				"account_id", freeze.AccountID,
				"amount", freeze.Amount.String(),
				"error", releaseErr,
			)
			continue
		}

		released++
		slog.InfoContext(ctx, "freeze_timeout_released",
			"freeze_id", freeze.ID,
			"account_id", freeze.AccountID,
			"amount", freeze.Amount.String(),
		)
	}

	return released, nil
}

// ---------------------------------------------------------------------------
// Liquidate
// ---------------------------------------------------------------------------

// Liquidate initiates or advances the account liquidation state machine per
// PRD S8.4.
//
// State transitions:
//
//	active (no existing liquidation) -> blocking (LiquidationStatusBlocking)
//	  Rejects new calls and freezes immediately.
//	blocking -> draining (LiquidationStatusDraining)
//	  Waits for existing freezes to expire/settle.
//	draining -> refunding (LiquidationStatusRefunding)
//	  Transfers remaining balance to target account.
//	refunding -> closing (LiquidationStatusClosing)
//	  Moves account to liquidating transfer stage.
//	closing -> closed (LiquidationStatusClosed)
//	  Terminal state; account is closed.
//
// If no liquidation exists for the account, this starts the process (active -> blocking).
// If a liquidation is already in progress, it advances one step.
//
// Side effects: updates account status and liquidation_stage, creates/updates
// liquidation record. Transitions to refunding also transfer remaining balance.
func (s *Service) Liquidate(ctx context.Context, req LiquidateRequest) (*LiquidateResult, error) {
	var result *LiquidateResult
	err := s.Store.WithTx(ctx, func(tx Tx) error {
		// Lock account.
		acct, err := s.Store.GetAccountForUpdate(tx, ctx, req.AccountID)
		if err != nil {
			return err
		}
		if acct == nil {
			return newAccountFrozenError(req.AccountID, "not found")
		}

		// Check for existing liquidation.
		existing, err := s.Store.GetLiquidation(ctx, req.AccountID)
		if err != nil {
			return err
		}

		now := time.Now()

		if existing == nil {
			// Start new liquidation: active -> blocking.
			if acct.Status != StatusActive {
				return newAccountFrozenError(req.AccountID, acct.Status)
			}

			if req.TargetAccountID == req.AccountID {
				return newSelfTransferError(req.AccountID)
			}

			// Verify target account is active.
			targetAcct, err := s.Store.GetAccountForUpdate(tx, ctx, req.TargetAccountID)
			if err != nil {
				return err
			}
			if targetAcct == nil {
				return newAccountFrozenError(req.TargetAccountID, "not found")
			}
			if targetAcct.Status != StatusActive {
				return newAccountFrozenError(req.TargetAccountID, targetAcct.Status)
			}

			// Update account status to liquidating.
			if err := s.Store.UpdateAccountStatus(tx, ctx, req.AccountID, StatusLiquidatingBlockNew, acct.Version); err != nil {
				return err
			}

			liq := &Liquidation{
				ID:              newUUID(),
				PartyID:         req.PartyID,
				AccountID:       req.AccountID,
				TargetAccountID: stringPtr(req.TargetAccountID),
				Status:          LiquidationStatusBlocking,
				InitiatedBy:     req.OperatorID,
				InitiatedAt:     now,
			}
			if err := s.Store.InsertLiquidation(tx, ctx, liq); err != nil {
				return err
			}

			result = &LiquidateResult{
				LiquidationID:   liq.ID,
				AccountID:       req.AccountID,
				PartyID:         req.PartyID,
				TargetAccountID: req.TargetAccountID,
				Status:          LiquidationStatusBlocking,
				InitiatedBy:     req.OperatorID,
				Reason:          req.Reason,
				InitiatedAt:     now,
			}

			slog.InfoContext(ctx, "liquidation_started",
				"liquidation_id", liq.ID,
				"account_id", req.AccountID,
				"party_id", req.PartyID,
				"target_account_id", req.TargetAccountID,
				"status", LiquidationStatusBlocking,
			)
			return nil
		}

		// Advance existing liquidation.
		nextStage, err := advanceLiquidationStage(existing.Status)
		if err != nil {
			return err
		}

		switch nextStage {
		case LiquidationStatusDraining:
			// blocking -> draining: just update status.
			if err := s.Store.UpdateAccountStatus(tx, ctx, req.AccountID, StatusLiquidatingDrain, acct.Version); err != nil {
				return err
			}
		case LiquidationStatusRefunding:
			// draining -> refunding: transfer remaining balance.
			targetID := ""
			if existing.TargetAccountID != nil {
				targetID = *existing.TargetAccountID
			}

			targetAcct, err := s.Store.GetAccountForUpdate(tx, ctx, targetID)
			if err != nil {
				return err
			}
			if targetAcct == nil {
				return newAccountFrozenError(targetID, "not found")
			}
			if targetAcct.Status != StatusActive {
				return newAccountFrozenError(targetID, targetAcct.Status)
			}

			remainingBalance := acct.AvailableBalance.Decimal
			if remainingBalance.GreaterThan(decimal.Zero) {
				// Transfer remaining balance to target.
				targetAvailableAfter := targetAcct.AvailableBalance.Decimal.Add(remainingBalance)
				srcAvailableAfter := decimal.Zero

				if err := s.Store.UpdateAccountBalances(tx, ctx, req.AccountID, srcAvailableAfter, acct.FrozenBalance.Decimal, acct.Version); err != nil {
					return err
				}
				acct.Version++ // Version was incremented by UpdateAccountBalances.
				if err := s.Store.UpdateAccountBalances(tx, ctx, targetID, targetAvailableAfter, targetAcct.FrozenBalance.Decimal, targetAcct.Version); err != nil {
					return err
				}

				// Insert ledger entries for the transfer.
				srcLedger := &Ledger{
					ID:           newUUID(),
					AccountID:    req.AccountID,
					Direction:    DirectionAllocateOut,
					Amount:       NewDecimal(remainingBalance.String()),
					BalanceAfter: NewDecimal(srcAvailableAfter.String()),
					Reason:       stringPtr("liquidation transfer to " + targetID),
					CreatedAt:    now,
				}
				if err := s.Store.InsertLedger(tx, ctx, srcLedger); err != nil {
					return err
				}

				dstLedger := &Ledger{
					ID:           newUUID(),
					AccountID:    targetID,
					Direction:    DirectionAllocateIn,
					Amount:       NewDecimal(remainingBalance.String()),
					BalanceAfter: NewDecimal(targetAvailableAfter.String()),
					Reason:       stringPtr("liquidation transfer from " + req.AccountID),
					CreatedAt:    now,
				}
				if err := s.Store.InsertLedger(tx, ctx, dstLedger); err != nil {
					return err
				}

				slog.InfoContext(ctx, "liquidation_balance_transferred",
					"liquidation_id", existing.ID,
					"account_id", req.AccountID,
					"target_account_id", targetID,
					"amount", remainingBalance.String(),
				)
			}

			if err := s.Store.UpdateAccountStatus(tx, ctx, req.AccountID, StatusLiquidatingTransfer, acct.Version); err != nil {
				return err
			}
		case LiquidationStatusClosing:
			// refunding -> closing: account becomes liquidating with transfer stage.
			if err := s.Store.UpdateAccountStatus(tx, ctx, req.AccountID, StatusLiquidatingTransfer, acct.Version); err != nil {
				return err
			}
		case LiquidationStatusClosed:
			// closing -> closed: terminal state, account closed.
			if err := s.Store.UpdateAccountStatus(tx, ctx, req.AccountID, StatusClosed, acct.Version); err != nil {
				return err
			}
		default:
			return newLiquidationStageInvalidError(req.AccountID, existing.Status, nextStage)
		}

		if err := s.Store.UpdateLiquidationStage(tx, ctx, existing.ID, nextStage); err != nil {
			return err
		}

		result = &LiquidateResult{
			LiquidationID:   existing.ID,
			AccountID:       req.AccountID,
			PartyID:         req.PartyID,
			TargetAccountID: stringValue(existing.TargetAccountID),
			Status:          nextStage,
			InitiatedBy:     req.OperatorID,
			Reason:          req.Reason,
			InitiatedAt:     now,
		}

		slog.InfoContext(ctx, "liquidation_advanced",
			"liquidation_id", existing.ID,
			"account_id", req.AccountID,
			"from_stage", existing.Status,
			"to_stage", nextStage,
		)

		return nil
	})

	if err != nil {
		slog.ErrorContext(ctx, "liquidation_failed",
			"account_id", req.AccountID,
			"error", err,
		)
		return nil, err
	}

	return result, nil
}

// advanceLiquidationStage returns the next valid stage in the liquidation
// state machine. Returns an error if the current stage is terminal or unknown.
func advanceLiquidationStage(current string) (string, error) {
	transitions := map[string]string{
		LiquidationStatusBlocking:  LiquidationStatusDraining,
		LiquidationStatusDraining:  LiquidationStatusRefunding,
		LiquidationStatusRefunding: LiquidationStatusClosing,
		LiquidationStatusClosing:    LiquidationStatusClosed,
	}
	next, ok := transitions[current]
	if !ok {
		return "", &FundError{
			Code:    "LIQUIDATION_STAGE_INVALID",
			Message: "cannot advance from terminal stage: " + current,
			Err:     ErrLiquidationStageInvalid,
		}
	}
	return next, nil
}
