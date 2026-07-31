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

// Freeze 为即将进行的模型调用预留资金，将 amount 从 available_balance 移动到
// frozen_balance，并创建一条带 TTL（默认 15 分钟，可配置）的开放冻结记录。
//
// 预算上限检查（PRD S5.2）：若账户设置了 budget_limit_amount，
// 则 budget_consumed + estimated_sell 不得超过该上限。预算检查在冻结之前进行，
// 以确保账户能够承担预估成本。
//
// 冻结金额必须至少为预估售价（含安全系数，通常 1.2x）。
// 调用方负责根据定价预估计算适当的冻结金额。
//
// 返回包含 freeze_id 的 FreezeResult，供后续结算或续期使用。
// 若未结算，冻结将在 TTL 后自动过期。
//
// 副作用：减少 available_balance，增加 frozen_balance，
// 插入冻结记录和冻结账本条目。全部在同一事务内完成。
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
			return newAccountFrozenError(req.AccountID, "未找到")
		}
		if acct.Status != StatusActive {
			return newAccountFrozenError(req.AccountID, acct.Status)
		}

		// 预算上限检查。
		if err := s.freezeCheckBudget(ctx, acct, req.Amount, req.EstimatedSell, req.RequestID); err != nil {
			return err
		}

		r, err := s.freezeExecute(ctx, tx, req, acct, ttl)
		if err != nil {
			return err
		}
		result = r
		return nil
	})

	if err != nil {
		slog.ErrorContext(ctx, "冻结失败",
			"request_id", req.RequestID,
			"account_id", req.AccountID,
			"amount", req.Amount.String(),
			"error", err,
		)
		return nil, err
	}

	return result, nil
}

// freezeCheckBudget 检查账户预算上限。若预估消费超过预算上限则返回错误。
// 若超过预警比例则发出警告日志。
func (s *Service) freezeCheckBudget(ctx context.Context, acct *Account, amount, estimatedSell Decimal, requestID string) error {
	if acct.BudgetLimitAmount == nil {
		return nil
	}

	estimatedCost := estimatedSell.Decimal
	if estimatedCost.IsZero() {
		estimatedCost = amount.Decimal
	}
	newConsumed := acct.BudgetConsumedAmount.Decimal.Add(estimatedCost)
	if newConsumed.GreaterThan(acct.BudgetLimitAmount.Decimal) {
		return newBudgetCapExceededError(acct.ID,
			acct.BudgetConsumedAmount, *acct.BudgetLimitAmount, NewDecimal(estimatedCost.String()))
	}

	// 超过预警比例时发出警告。
	if acct.BudgetWarnRatio != nil {
		ratio := newConsumed.Div(acct.BudgetLimitAmount.Decimal)
		if ratio.GreaterThanOrEqual(acct.BudgetWarnRatio.Decimal) {
			slog.WarnContext(ctx, "预算预警比例超限",
				"account_id", acct.ID,
				"consumed", acct.BudgetConsumedAmount.String(),
				"limit", acct.BudgetLimitAmount.String(),
				"ratio", ratio.String(),
				"request_id", requestID,
			)
		}
	}
	return nil
}

// freezeExecute 在事务内执行核心冻结逻辑。
// 验证可用余额、计算新余额、更新账户、创建冻结记录和账本条目。
func (s *Service) freezeExecute(ctx context.Context, tx Tx, req FreezeRequest, acct *Account, ttl time.Duration) (*FreezeResult, error) {
	// 验证可用余额。
	if acct.AvailableBalance.Decimal.LessThan(req.Amount.Decimal) {
		return nil, newInsufficientBalanceError(req.AccountID, acct.AvailableBalance, req.Amount)
	}

	now := time.Now()
	expiresAt := now.Add(ttl)
	maxLifetimeAt := now.Add(maxFreezeLifetime)

	// 计算新余额。
	availableAfter := acct.AvailableBalance.Decimal.Sub(req.Amount.Decimal)
	frozenAfter := acct.FrozenBalance.Decimal.Add(req.Amount.Decimal)

	// 更新账户。
	if err := s.Store.UpdateAccountBalances(tx, ctx, req.AccountID, availableAfter, frozenAfter, acct.Version); err != nil {
		return nil, err
	}

	// 创建冻结记录。
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
		return nil, err
	}

	// 插入冻结账本条目。
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
		return nil, err
	}

	result := &FreezeResult{
		FreezeID:      freeze.ID,
		AccountID:     req.AccountID,
		Amount:        req.Amount,
		EstimatedSell: req.EstimatedSell,
		BalanceAfter:  NewDecimal(availableAfter.String()),
		FrozenAfter:   NewDecimal(frozenAfter.String()),
		ExpiresAt:     expiresAt,
		RequestID:     req.RequestID,
	}

	// 结构化日志。
	slog.InfoContext(ctx, "冻结获取成功",
		"request_id", req.RequestID,
		"account_id", req.AccountID,
		"freeze_id", freeze.ID,
		"amount", req.Amount.String(),
		"estimated_sell", req.EstimatedSell.String(),
		"balance_after", availableAfter.String(),
		"frozen_after", frozenAfter.String(),
		"expires_at", expiresAt,
	)

	return result, nil
}

// ---------------------------------------------------------------------------
// Settle
// ---------------------------------------------------------------------------

// Settle 在上游模型调用完成且实际用量已知后完成冻结。
//
// 步骤：
//  1. 锁定冻结记录（必须处于 open 状态）。
//  2. 若冻结已过期：尝试孤儿结算——如可能则从可用余额中扣除，否则告警并释放冻结。
//  3. 计算差额：frozen_amount - actual_sell = refund。
//  4. 从 frozen_balance 和 available_balance 中扣除 actual_sell（售出部分被消费）。
//  5. 将退款归还 available_balance。
//  6. 更新 budget_consumed_amount。
//  7. 插入结算账本条目（direction=settle），若 refund > 0 则插入解冻账本条目。
//  8. 将冻结标记为已结算。
//
// 若 actual_sell > frozen_amount，结算失败并返回不匹配错误
//（冻结金额过小——表明上游定价错误）。
//
// 副作用：更新账户余额、插入账本条目、更新冻结状态、递增 budget_consumed_amount。
func (s *Service) Settle(ctx context.Context, req SettleRequest) (*SettleResult, error) {
	// 验证不变量。
	if req.ActualSell.Decimal.GreaterThan(decimal.Zero) && req.ActualCost.Decimal.GreaterThan(req.ActualSell.Decimal) {
		slog.WarnContext(ctx, "成本超过售价",
			"freeze_id", req.FreezeID,
			"actual_cost", req.ActualCost.String(),
			"actual_sell", req.ActualSell.String(),
			"request_id", req.RequestID,
		)
	}

	var result *SettleResult
	err := s.Store.WithTx(ctx, func(tx Tx) error {
		// 使用行锁获取 freeze 记录——防止并发 Settle 同一 freeze_id（RED-2 竞态修复）。
		freeze, err := s.Store.GetFreezeForUpdate(tx, ctx, req.FreezeID)
		if err != nil {
			return err
		}
		if freeze == nil {
			return newFreezeNotFoundError(req.FreezeID)
		}

		now := time.Now()
		settleAmount := req.ActualSell.Decimal
		settleCost := req.ActualCost.Decimal

		// 检查冻结是否已过期。
		if now.After(freeze.ExpiresAt) && freeze.Status == FreezeStatusOpen {
			slog.WarnContext(ctx, "冻结已过期孤儿结算",
				"freeze_id", req.FreezeID,
				"account_id", freeze.AccountID,
				"frozen_amount", freeze.Amount.String(),
				"actual_sell", req.ActualSell.String(),
				"request_id", req.RequestID,
			)
			// 尝试孤儿结算：若 sell <= frozen，继续扣减。
			// 若不足，释放冻结并告警。
		}

		if freeze.Status != FreezeStatusOpen {
			return &FundError{
				Code:    "FREEZE_NOT_OPEN",
				Message: "冻结 " + req.FreezeID + " 状态为 " + freeze.Status,
				Err:     ErrFreezeExpired,
			}
		}

		// 验证结算金额与冻结金额的匹配。
		frozenAmount := freeze.Amount.Decimal
		if settleAmount.GreaterThan(frozenAmount) {
			return &FundError{
				Code:    "SETTLEMENT_MISMATCH",
				Message: "冻结 " + req.FreezeID + " 冻结金额 " + frozenAmount.String() + " 但结算金额 " + settleAmount.String(),
				Err:     ErrInsufficientBalance,
			}
		}

		// 锁定账户。
		acct, err := s.Store.GetAccountForUpdate(tx, ctx, freeze.AccountID)
		if err != nil {
			return err
		}
		if acct == nil {
			return newAccountFrozenError(freeze.AccountID, "未找到")
		}

		// 计算退款和新的余额快照。
		refund, availableAfter, frozenAfter, err := s.settleCalculateRefund(freeze, settleAmount, acct)
		if err != nil {
			return err
		}

		r, err := s.settleExecute(ctx, tx, req, freeze, acct, settleAmount, settleCost, refund, availableAfter, frozenAfter, now)
		if err != nil {
			return err
		}
		result = r
		return nil
	})

	if err != nil {
		slog.ErrorContext(ctx, "结算失败",
			"request_id", req.RequestID,
			"freeze_id", req.FreezeID,
			"actual_sell", req.ActualSell.String(),
			"error", err,
		)
		return nil, err
	}

	return result, nil
}

// settleCalculateRefund 计算退款金额并返回结算后的余额快照。
// 若结算会导致透支则返回错误。
func (s *Service) settleCalculateRefund(freeze *Freeze, settleAmount decimal.Decimal, acct *Account) (refund, availableAfter, frozenAfter decimal.Decimal, err error) {
	frozenAmount := freeze.Amount.Decimal

	// 计算退款：冻结金额 - 实际售价。
	refund = frozenAmount.Sub(settleAmount)

	// 新余额：先释放所有冻结资金，再从可用余额中扣除售出部分。
	// frozen_balance 减少总冻结金额（该冻结正在被结算）。
	// available_balance：售出部分被真正消费，退款部分原本就在可用余额中（仅解冻）。
	availableAfter = acct.AvailableBalance.Decimal.Add(refund)
	frozenAfter = acct.FrozenBalance.Decimal.Sub(frozenAmount)

	if availableAfter.LessThan(decimal.Zero) {
		return decimal.Zero, decimal.Zero, decimal.Zero, &FundError{
			Code:    "SETTLEMENT_OVERDRAW",
			Message: "结算将导致账户 " + freeze.AccountID + " 透支",
			Err:     ErrInsufficientBalance,
		}
	}

	return refund, availableAfter, frozenAfter, nil
}

// settleExecute 在事务内执行核心结算操作：
// 更新余额、预算消耗、插入账本条目、更新冻结状态。
func (s *Service) settleExecute(ctx context.Context, tx Tx, req SettleRequest, freeze *Freeze, acct *Account,
	settleAmount, settleCost, refund, availableAfter, frozenAfter decimal.Decimal, now time.Time) (*SettleResult, error) {

	// 更新账户余额。
	if err := s.Store.UpdateAccountBalances(tx, ctx, freeze.AccountID, availableAfter, frozenAfter, acct.Version); err != nil {
		return nil, err
	}

	// 以售出金额更新 budget_consumed_amount。
	if settleAmount.GreaterThan(decimal.Zero) {
		if err := s.Store.UpdateAccountBudgetConsumed(tx, ctx, freeze.AccountID, settleAmount); err != nil {
			return nil, err
		}
	}

	// 插入结算账本条目。
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
		return nil, err
	}

	// 若有退款，插入解冻账本条目。
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
			return nil, err
		}
	}

	// 更新冻结状态。
	if err := s.Store.UpdateFreezeStatus(tx, ctx, req.FreezeID, FreezeStatusSettled, &settleAmount, &settleCost); err != nil {
		return nil, err
	}

	result := &SettleResult{
		FreezeID:       req.FreezeID,
		ActualSell:     req.ActualSell,
		ActualCost:     req.ActualCost,
		ReleasedAmount: NewDecimal(refund.String()),
		BalanceAfter:   NewDecimal(availableAfter.String()),
		FrozenAfter:    NewDecimal(frozenAfter.String()),
	}

	slog.InfoContext(ctx, "冻结已结算",
		"request_id", req.RequestID,
		"account_id", freeze.AccountID,
		"freeze_id", freeze.ID,
		"frozen_amount", freeze.Amount.Decimal.String(),
		"settle_amount", settleAmount.String(),
		"refund_amount", refund.String(),
		"cost_amount", settleCost.String(),
		"balance_after", availableAfter.String(),
		"frozen_after", frozenAfter.String(),
	)

	return result, nil
}
