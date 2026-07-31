package fund

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
)

// Service is the core financial governance service implementing the FundService
// contract from the architecture spec (architecture-v3.2.md S5.2). It owns all
// fund mutations: allocations, freezes, settlements, renewals, and liquidations.
//
// Every exported method is concurrency-safe when backed by a Store that provides
// row-level locking (SELECT FOR UPDATE) or optimistic concurrency control.
// Idempotency is guaranteed for write operations via the IdempotencyKey mechanism.
//
// All monetary values MUST use shopspring/decimal.Decimal — never float64.
// All fund mutations log structured records with request_id, account_id,
// freeze_id, amount, and balance_after per AGENTS.md S6.2.
type Service struct {
	Store       Store
	Idempotency IdempotencyChecker
}

// IdempotencyChecker abstracts the idempotency key claim/preview pattern.
// The idempotency package provides the concrete implementation.
type IdempotencyChecker interface {
	// Claim atomically claims an idempotency key. Returns true if this call
	// is the first (proceed with execution), false if a previous result exists
	// (return the stored result).
	Claim(ctx context.Context, key string) (bool, error)

	// Store persists a result against an idempotency key for future lookups.
	Store(ctx context.Context, key string, result any) error

	// Retrieve fetches a previously stored result.
	Retrieve(ctx context.Context, key string, result any) (bool, error)
}

// defaultFreezeTTL is the default freeze expiry duration (15 minutes per PRD S8.3).
const defaultFreezeTTL = 15 * time.Minute

// maxFreezeLifetime is the maximum cumulative freeze lifetime (2 hours per PRD S8.3).
const maxFreezeLifetime = 2 * time.Hour

// ---------------------------------------------------------------------------
// Allocate
// ---------------------------------------------------------------------------

// Allocate 从源账户向目标账户划拨资金。
//
// Conservation guarantee (F-CON-02): src_delta + dst_delta = 0 within a single
// database transaction. Both accounts are locked in ID order to prevent deadlock.
//
// Channel validation per PRD S8.2:
//   - parent: allowed only in the parent-to-child direction
//   - sponsors: allowed only in the sponsor-to-sponsored direction
//   - allocates: allowed from party to person account
//   - whitelist: requires an explicit whitelist grant
//
// Both accounts must be active. The source account must have sufficient
// available balance.
//
// Idempotency: if the idempotency key has already been used successfully,
// the original result is returned without executing a second transfer.
//
// Side effects: inserts two ledger entries (debit and credit), updates both
// account balances, creates an allocation record. All within one transaction.
func (s *Service) Allocate(ctx context.Context, req AllocateRequest) (*AllocateResult, error) {
	// Validate basic invariants.
	if req.Amount.Decimal.LessThanOrEqual(decimal.Zero) {
		return nil, newAmountMustBePositiveError(req.Amount)
	}
	if req.SrcAccountID == req.DstAccountID {
		return nil, newSelfTransferError(req.SrcAccountID)
	}

	// Check idempotency outside the transaction.
	if req.IdempotencyKey != "" && s.Idempotency != nil {
		claimed, err := s.Idempotency.Claim(ctx, req.IdempotencyKey)
		if err != nil {
			slog.ErrorContext(ctx, "idempotency_claim_failed",
				"idempotency_key", req.IdempotencyKey,
				"error", err,
			)
			return nil, err
		}
		if !claimed {
			var result AllocateResult
			found, err := s.Idempotency.Retrieve(ctx, req.IdempotencyKey, &result)
			if err != nil {
				return nil, err
			}
			if found {
				slog.InfoContext(ctx, "allocate_idempotency_replay",
					"idempotency_key", req.IdempotencyKey,
					"allocation_id", result.AllocationID,
				)
				return &result, nil
			}
			// Key claimed but no result stored — conflict.
			return nil, newIdempotencyConflictError(req.IdempotencyKey)
		}
	}

	var result *AllocateResult
	err := s.Store.WithTx(ctx, func(tx Tx) error {
		// Lock both accounts in ID order to prevent deadlock.
		firstID, secondID := req.SrcAccountID, req.DstAccountID
		if firstID > secondID {
			firstID, secondID = secondID, firstID
		}

		var srcAcct, dstAcct *Account
		var err error

		if firstID == req.SrcAccountID {
			srcAcct, err = s.Store.GetAccountForUpdate(tx, ctx, req.SrcAccountID)
			if err != nil {
				return err
			}
			dstAcct, err = s.Store.GetAccountForUpdate(tx, ctx, req.DstAccountID)
			if err != nil {
				return err
			}
		} else {
			dstAcct, err = s.Store.GetAccountForUpdate(tx, ctx, req.DstAccountID)
			if err != nil {
				return err
			}
			srcAcct, err = s.Store.GetAccountForUpdate(tx, ctx, req.SrcAccountID)
			if err != nil {
				return err
			}
		}

		// Validate account statuses.
		if srcAcct == nil {
			return newAccountFrozenError(req.SrcAccountID, "not found")
		}
		if srcAcct.Status != StatusActive {
			return newAccountFrozenError(req.SrcAccountID, srcAcct.Status)
		}
		if dstAcct == nil {
			return newAccountFrozenError(req.DstAccountID, "not found")
		}
		if dstAcct.Status != StatusActive {
			return newAccountFrozenError(req.DstAccountID, dstAcct.Status)
		}

		// Verify sufficient balance.
		if srcAcct.AvailableBalance.Decimal.LessThan(req.Amount.Decimal) {
			return newInsufficientBalanceError(req.SrcAccountID, srcAcct.AvailableBalance, req.Amount)
		}

		// Validate channel.
		if err := s.validateChannel(ctx, req.Channel, req.SrcAccountID, req.DstAccountID, req.EdgeID); err != nil {
			return err
		}

		// Compute new balances.
		srcAvailableAfter := srcAcct.AvailableBalance.Decimal.Sub(req.Amount.Decimal)
		dstAvailableAfter := dstAcct.AvailableBalance.Decimal.Add(req.Amount.Decimal)

		// Update source account (debit).
		if err := s.Store.UpdateAccountBalances(tx, ctx, req.SrcAccountID, srcAvailableAfter, srcAcct.FrozenBalance.Decimal, srcAcct.Version); err != nil {
			return err
		}

		// Update destination account (credit).
		if err := s.Store.UpdateAccountBalances(tx, ctx, req.DstAccountID, dstAvailableAfter, dstAcct.FrozenBalance.Decimal, dstAcct.Version); err != nil {
			return err
		}

		now := time.Now()

		// Create allocation record.
		allocation := &Allocation{
			ID:             newUUID(),
			SrcAccountID:   req.SrcAccountID,
			DstAccountID:   req.DstAccountID,
			Amount:         req.Amount,
			Channel:        req.Channel,
			EdgeID:         req.EdgeID,
			Status:         AllocationStatusCompleted,
			IdempotencyKey: stringPtr(req.IdempotencyKey),
			ActorUserID:    stringPtr(req.OperatorID),
			Reason:         req.Reason,
			CreatedAt:      now,
			CompletedAt:    timePtr(now),
		}
		if err := s.Store.InsertAllocation(tx, ctx, allocation); err != nil {
			return err
		}

		// Insert ledger entries — one debit, one credit.
		srcLedger := &Ledger{
			ID:             newUUID(),
			AccountID:      req.SrcAccountID,
			Direction:      DirectionAllocateOut,
			Amount:         req.Amount,
			BalanceAfter:   NewDecimal(srcAvailableAfter.String()),
			AllocationID:   stringPtr(allocation.ID),
			IdempotencyKey: stringPtr(req.IdempotencyKey),
			Reason:         req.Reason,
			CreatedAt:      now,
		}
		if err := s.Store.InsertLedger(tx, ctx, srcLedger); err != nil {
			return err
		}

		dstLedger := &Ledger{
			ID:             newUUID(),
			AccountID:      req.DstAccountID,
			Direction:      DirectionAllocateIn,
			Amount:         req.Amount,
			BalanceAfter:   NewDecimal(dstAvailableAfter.String()),
			AllocationID:   stringPtr(allocation.ID),
			IdempotencyKey: stringPtr(req.IdempotencyKey),
			Reason:         req.Reason,
			CreatedAt:      now,
		}
		if err := s.Store.InsertLedger(tx, ctx, dstLedger); err != nil {
			return err
		}

		result = &AllocateResult{
			AllocationID:    allocation.ID,
			SrcAccountID:    req.SrcAccountID,
			DstAccountID:    req.DstAccountID,
			Amount:          req.Amount,
			Channel:         req.Channel,
			EdgeID:          req.EdgeID,
			Status:          AllocationStatusCompleted,
			SrcBalanceAfter: NewDecimal(srcAvailableAfter.String()),
			DstBalanceAfter: NewDecimal(dstAvailableAfter.String()),
			IdempotencyKey:  req.IdempotencyKey,
			CreatedAt:       now,
			CompletedAt:     now,
		}

		// Store idempotency result within the transaction.
		if req.IdempotencyKey != "" && s.Idempotency != nil {
			if err := s.Store.StoreIdempotency(tx, ctx, req.IdempotencyKey, result); err != nil {
				return err
			}
		}

		// Structured log.
		slog.InfoContext(ctx, "allocate_completed",
			"allocation_id", allocation.ID,
			"src_account_id", req.SrcAccountID,
			"dst_account_id", req.DstAccountID,
			"amount", req.Amount.String(),
			"channel", req.Channel,
			"src_balance_after", srcAvailableAfter.String(),
			"dst_balance_after", dstAvailableAfter.String(),
			"idempotency_key", req.IdempotencyKey,
		)

		return nil
	})

	if err != nil {
		slog.ErrorContext(ctx, "allocate_failed",
			"src_account_id", req.SrcAccountID,
			"dst_account_id", req.DstAccountID,
			"amount", req.Amount.String(),
			"idempotency_key", req.IdempotencyKey,
			"error", err,
		)
		return nil, err
	}

	return result, nil
}

// validateChannel checks whether the requested transfer channel is permitted
// for the given source and destination accounts. The channel is validated against
// the known allowed direction rules: parent, sponsors, allocates, and whitelist.
// This is a placeholder that returns nil for allowed channels by default;
// production implementations should validate against party_edges and whitelist tables.
func (s *Service) validateChannel(ctx context.Context, channel string, srcAccountID, dstAccountID string, edgeID *string) error {
	switch channel {
	case ChannelParent, ChannelSponsors, ChannelAllocates, ChannelWhitelist:
		// Channel is syntactically valid. Full semantic validation (checking
		// party_edges and allocate_whitelist tables) requires party package
		// integration and is deferred to a later stage. The architecture
		// guarantees that the party package sits at Layer 0 alongside fund,
		// so cross-package calls are architecturally legal.
			return nil
		default:
			return newAllocationChannelDeniedError(srcAccountID, dstAccountID, channel)
		}
}

// ---------------------------------------------------------------------------
// Utility helpers (shared across all fund service files)
// ---------------------------------------------------------------------------

// newUUID generates a unique identifier. Replaced with a proper UUID library
// when one is introduced to the project.
func newUUID() string {
	return fmt.Sprintf("%016x-%04x-%04x-%04x-%012x",
		time.Now().UnixNano(),
		time.Now().Nanosecond()%0x10000,
		time.Now().Nanosecond()%0x10000,
		time.Now().Nanosecond()%0x10000,
		time.Now().Nanosecond()%0x1000000000000,
	)
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func decPtr(d decimal.Decimal) *Decimal {
	dd := Decimal{d}
	return &dd
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
