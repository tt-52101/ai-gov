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

// RenewFreeze 为流式调用延长开放冻结的过期时间。
//
// 它不增加冻结金额。而是将 expires_at 延长默认 TTL 并递增 renewal_count。
// 累计总生命周期上限为 max_lifetime_at（自原始冻结起 2 小时）。
//
// 若冻结非 open 状态或已超过 max_lifetime_at，RenewFreeze 返回错误。
//
// 副作用：更新 freeze.expires_at、freeze.renewal_count 和 freeze.last_renewed_at。
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
				Message: "冻结 " + freezeID + " 状态为 " + freeze.Status,
				Err:     ErrFreezeExpired,
			}
		}

		now := time.Now()
		if now.After(freeze.ExpiresAt) {
			return newFreezeExpiredError(freezeID)
		}

		newExpires := now.Add(defaultFreezeTTL)
		if freeze.MaxLifetimeAt != nil && newExpires.After(*freeze.MaxLifetimeAt) {
			slog.WarnContext(ctx, "冻结续期已达上限",
				"freeze_id", freezeID,
				"account_id", freeze.AccountID,
				"requested_expires", newExpires,
				"max_lifetime", freeze.MaxLifetimeAt,
			)
			return &FundError{
				Code:    "FREEZE_MAX_LIFETIME",
				Message: "冻结 " + freezeID + " 已达最大生命周期",
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

		slog.InfoContext(ctx, "冻结已续期",
			"freeze_id", freezeID,
			"account_id", freeze.AccountID,
			"amount", freeze.Amount.String(),
			"new_expires_at", newExpires,
			"renewal_count", freeze.RenewalCount+1,
		)

		return nil
	})

	if err != nil {
		slog.ErrorContext(ctx, "冻结续期失败",
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

// UnfreezeTimeout 释放已过期的开放冻结。由后台 TTL 扫描器工作进程（运维平面）调用。
//
// 它扫描 expires_at 已过的开放冻结（最多 limit 行），对每条记录：
//   - 将冻结金额归还 available_balance
//   - 减少 frozen_balance
//   - 插入解冻账本条目
//   - 将冻结标记为 timeout_released
//
// 返回已释放的冻结数量。
//
// 每条冻结释放在其独立事务中运行以最小化锁争用。
// 个别失败被记录并跳过——扫描器继续处理剩余冻结。
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

			// 插入解冻账本条目。
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
			slog.ErrorContext(ctx, "超时解冻失败",
				"freeze_id", freeze.ID,
				"account_id", freeze.AccountID,
				"amount", freeze.Amount.String(),
				"error", releaseErr,
			)
			continue
		}

		released++
		slog.InfoContext(ctx, "冻结超时已释放",
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

// Liquidate 按 PRD S8.4 启动或推进账户清算状态机。
//
// 状态转换（PRD 4 阶段）：
//
//	active（无现存清算） -> blocking（LiquidationStatusBlocking）
//	  立即拒绝新调用和冻结。
//	blocking -> draining（LiquidationStatusDraining）
//	  等待既有冻结过期/结算。
//	draining -> transfer（LiquidationStatusTransfer）
//	  将剩余余额转移到目标账户，完毕。
//	transfer -> liquidated（LiquidationStatusLiquidated）
//	  终态；账户已清算关闭。
//
// 若账户无现存清算，则启动流程（active -> blocking）。
// 若清算已在进行中，则推进一个阶段。
//
// 副作用：更新账户状态和 liquidation_stage，创建/更新清算记录。
// 转换到 transfer 时也会转移剩余余额。
func (s *Service) Liquidate(ctx context.Context, req LiquidateRequest) (*LiquidateResult, error) {
	// 校验基本参数。
	if err := s.liquidateValidateReq(req); err != nil {
		return nil, err
	}

	var result *LiquidateResult
	err := s.Store.WithTx(ctx, func(tx Tx) error {
		// 锁定账户。
		acct, err := s.Store.GetAccountForUpdate(tx, ctx, req.AccountID)
		if err != nil {
			return err
		}
		if acct == nil {
			return newAccountFrozenError(req.AccountID, "未找到")
		}

		// 检查是否存在清算记录。
		existing, err := s.Store.GetLiquidation(ctx, req.AccountID)
		if err != nil {
			return err
		}

		now := time.Now()

		if existing == nil {
			// 启动新清算：active -> blocking。
			result, err = s.liquidateStartNew(ctx, tx, req, acct, now)
		} else {
			// 推进既有清算。
			result, err = s.liquidateAdvance(ctx, tx, req, acct, existing, now)
		}
		return err
	})

	if err != nil {
		slog.ErrorContext(ctx, "清算失败",
			"account_id", req.AccountID,
			"error", err,
		)
		return nil, err
	}

	return result, nil
}

// liquidateValidateReq 校验清算请求基本参数。
func (s *Service) liquidateValidateReq(req LiquidateRequest) error {
	if req.AccountID == req.TargetAccountID {
		return newSelfTransferError(req.AccountID)
	}
	return nil
}

// liquidateStartNew 启动新清算流程（active -> blocking）。
// 验证账户状态、目标账户，更新账户状态并创建清算记录。
func (s *Service) liquidateStartNew(ctx context.Context, tx Tx, req LiquidateRequest, acct *Account, now time.Time) (*LiquidateResult, error) {
	if acct.Status != StatusActive {
		return nil, newAccountFrozenError(req.AccountID, acct.Status)
	}

	// 验证目标账户处于活跃状态。
	targetAcct, err := s.Store.GetAccountForUpdate(tx, ctx, req.TargetAccountID)
	if err != nil {
		return nil, err
	}
	if targetAcct == nil {
		return nil, newAccountFrozenError(req.TargetAccountID, "未找到")
	}
	if targetAcct.Status != StatusActive {
		return nil, newAccountFrozenError(req.TargetAccountID, targetAcct.Status)
	}

	// 更新账户状态为清算中-阻止新操作。
	if err := s.Store.UpdateAccountStatus(tx, ctx, req.AccountID, StatusLiquidatingBlockNew, acct.Version); err != nil {
		return nil, err
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
		return nil, err
	}

	slog.InfoContext(ctx, "清算已启动",
		"liquidation_id", liq.ID,
		"account_id", req.AccountID,
		"party_id", req.PartyID,
		"target_account_id", req.TargetAccountID,
		"status", LiquidationStatusBlocking,
	)

	return &LiquidateResult{
		LiquidationID:   liq.ID,
		AccountID:       req.AccountID,
		PartyID:         req.PartyID,
		TargetAccountID: req.TargetAccountID,
		Status:          LiquidationStatusBlocking,
		InitiatedBy:     req.OperatorID,
		Reason:          req.Reason,
		InitiatedAt:     now,
	}, nil
}

// liquidateAdvance 推进既有清算至下一阶段。
// 验证阶段转换后执行相应阶段的操作。
func (s *Service) liquidateAdvance(ctx context.Context, tx Tx, req LiquidateRequest, acct *Account, existing *Liquidation, now time.Time) (*LiquidateResult, error) {
	nextStage, err := advanceLiquidationStage(existing.Status)
	if err != nil {
		return nil, err
	}

	if err := s.liquidateTransitionStage(ctx, tx, req, acct, existing, nextStage, now); err != nil {
		return nil, err
	}

	if err := s.Store.UpdateLiquidationStage(tx, ctx, existing.ID, nextStage); err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "清算已推进",
		"liquidation_id", existing.ID,
		"account_id", req.AccountID,
		"from_stage", existing.Status,
		"to_stage", nextStage,
	)

	return &LiquidateResult{
		LiquidationID:   existing.ID,
		AccountID:       req.AccountID,
		PartyID:         req.PartyID,
		TargetAccountID: stringValue(existing.TargetAccountID),
		Status:          nextStage,
		InitiatedBy:     req.OperatorID,
		Reason:          req.Reason,
		InitiatedAt:     now,
	}, nil
}

// liquidateTransitionStage 根据下一阶段执行相应的状态转换操作。
// PRD 4 阶段：blocking → draining → transfer → liquidated
func (s *Service) liquidateTransitionStage(ctx context.Context, tx Tx, req LiquidateRequest, acct *Account, existing *Liquidation, nextStage string, now time.Time) error {
	switch nextStage {
	case LiquidationStatusDraining:
		// blocking -> draining：仅更新账户状态。
		return s.Store.UpdateAccountStatus(tx, ctx, req.AccountID, StatusLiquidatingDrain, acct.Version)

	case LiquidationStatusTransfer:
		// draining -> transfer：转移剩余余额后关闭账户。
		targetID := ""
		if existing.TargetAccountID != nil {
			targetID = *existing.TargetAccountID
		}

		targetAcct, err := s.Store.GetAccountForUpdate(tx, ctx, targetID)
		if err != nil {
			return err
		}
		if targetAcct == nil {
			return newAccountFrozenError(targetID, "未找到")
		}
		if targetAcct.Status != StatusActive {
			return newAccountFrozenError(targetID, targetAcct.Status)
		}

		remainingBalance := acct.AvailableBalance.Decimal
		if remainingBalance.GreaterThan(decimal.Zero) {
			// 将剩余余额转移到目标账户。
			targetAvailableAfter := targetAcct.AvailableBalance.Decimal.Add(remainingBalance)
			srcAvailableAfter := decimal.Zero

			if err := s.Store.UpdateAccountBalances(tx, ctx, req.AccountID, srcAvailableAfter, acct.FrozenBalance.Decimal, acct.Version); err != nil {
				return err
			}
			acct.Version++ // UpdateAccountBalances 已递增 Version。
			if err := s.Store.UpdateAccountBalances(tx, ctx, targetID, targetAvailableAfter, targetAcct.FrozenBalance.Decimal, targetAcct.Version); err != nil {
				return err
			}

			// 为划转插入账本条目。
			srcLedger := &Ledger{
				ID:           newUUID(),
				AccountID:    req.AccountID,
				Direction:    DirectionAllocateOut,
				Amount:       NewDecimal(remainingBalance.String()),
				BalanceAfter: NewDecimal(srcAvailableAfter.String()),
				Reason:       stringPtr("清算划转到 " + targetID),
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
				Reason:       stringPtr("清算划转来自 " + req.AccountID),
				CreatedAt:    now,
			}
			if err := s.Store.InsertLedger(tx, ctx, dstLedger); err != nil {
				return err
			}

			slog.InfoContext(ctx, "清算余额已划转",
				"liquidation_id", existing.ID,
				"account_id", req.AccountID,
				"target_account_id", targetID,
				"amount", remainingBalance.String(),
			)
		}

		// 设置账户状态为清算划转中。
		return s.Store.UpdateAccountStatus(tx, ctx, req.AccountID, StatusLiquidatingTransfer, acct.Version)

	case LiquidationStatusLiquidated:
		// transfer -> liquidated：终态，账户关闭。
		return s.Store.UpdateAccountStatus(tx, ctx, req.AccountID, StatusClosed, acct.Version)

	default:
		return newLiquidationStageInvalidError(req.AccountID, existing.Status, nextStage)
	}
}

// advanceLiquidationStage 返回清算状态机中的下一个合法阶段。
// PRD 4 阶段：blocking → draining → transfer → liquidated
// 若当前阶段为终态或未知则返回错误。
func advanceLiquidationStage(current string) (string, error) {
	transitions := map[string]string{
		LiquidationStatusBlocking:   LiquidationStatusDraining,
		LiquidationStatusDraining:   LiquidationStatusTransfer,
		LiquidationStatusTransfer:   LiquidationStatusLiquidated,
	}
	next, ok := transitions[current]
	if !ok {
		return "", &FundError{
			Code:    "LIQUIDATION_STAGE_INVALID",
			Message: "无法从终态阶段推进: " + current,
			Err:     ErrLiquidationStageInvalid,
		}
	}
	return next, nil
}
