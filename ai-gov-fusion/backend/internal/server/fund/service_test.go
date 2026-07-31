package fund

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

// fakeStore is an in-memory implementation of Store for unit tests.
// It provides full transaction support (serialized via a mutex) and
// deterministic behaviour without a database.
type fakeStore struct {
	mu              sync.Mutex
	accounts        map[string]*Account
	ledgers         []*Ledger
	freezes         map[string]*Freeze
	allocations     map[string]*Allocation
	liquidations    map[string]*Liquidation
	idempotency     map[string]*AllocateResult
	idempotencyKeys map[string]bool
}

// fakeTx implements Tx for the fake store. Since the fake store is
// single-threaded and uses a mutex, the transaction is a no-op — all
// mutations apply directly to the in-memory maps under the mutex.
type fakeTx struct{}

func (fakeTx) Commit() error   { return nil }
func (fakeTx) Rollback() error { return nil }

func newFakeStore() *fakeStore {
	return &fakeStore{
		accounts:        make(map[string]*Account),
		freezes:         make(map[string]*Freeze),
		allocations:     make(map[string]*Allocation),
		liquidations:    make(map[string]*Liquidation),
		idempotency:     make(map[string]*AllocateResult),
		idempotencyKeys: make(map[string]bool),
	}
}

func (s *fakeStore) WithTx(ctx context.Context, fn func(Tx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fn(fakeTx{})
}

func (s *fakeStore) GetAccount(ctx context.Context, id string) (*Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	acct, ok := s.accounts[id]
	if !ok {
		return nil, nil
	}
	cp := *acct
	return &cp, nil
}

func (s *fakeStore) GetAccountForUpdate(tx Tx, ctx context.Context, id string) (*Account, error) {
	// Already holding mutex from WithTx.
	acct, ok := s.accounts[id]
	if !ok {
		return nil, nil
	}
	cp := *acct
	return &cp, nil
}

func (s *fakeStore) UpdateAccountBalances(tx Tx, ctx context.Context, id string, available, frozen decimal.Decimal, version int64) error {
	acct, ok := s.accounts[id]
	if !ok {
		return ErrAccountFrozen
	}
	if acct.Version != version {
		return &FundError{Code: "OPTIMISTIC_LOCK", Message: "version mismatch", Err: ErrAccountFrozen}
	}
	acct.AvailableBalance = NewDecimal(available.String())
	acct.FrozenBalance = NewDecimal(frozen.String())
	acct.Version++
	acct.UpdatedAt = time.Now()
	return nil
}

func (s *fakeStore) UpdateAccountStatus(tx Tx, ctx context.Context, id string, status string, version int64) error {
	acct, ok := s.accounts[id]
	if !ok {
		return ErrAccountFrozen
	}
	if acct.Version != version {
		return &FundError{Code: "OPTIMISTIC_LOCK", Message: "version mismatch", Err: ErrAccountFrozen}
	}
	acct.Status = status
	acct.Version++
	acct.UpdatedAt = time.Now()
	return nil
}

func (s *fakeStore) UpdateAccountBudgetConsumed(tx Tx, ctx context.Context, id string, delta decimal.Decimal) error {
	acct, ok := s.accounts[id]
	if !ok {
		return ErrAccountFrozen
	}
	acct.BudgetConsumedAmount = NewDecimal(acct.BudgetConsumedAmount.Decimal.Add(delta).String())
	return nil
}

func (s *fakeStore) InsertLedger(tx Tx, ctx context.Context, entry *Ledger) error {
	s.ledgers = append(s.ledgers, entry)
	return nil
}

func (s *fakeStore) InsertFreeze(tx Tx, ctx context.Context, f *Freeze) error {
	cp := *f
	s.freezes[f.ID] = &cp
	return nil
}

func (s *fakeStore) GetFreeze(ctx context.Context, freezeID string) (*Freeze, error) {
	f, ok := s.freezes[freezeID]
	if !ok {
		return nil, nil
	}
	cp := *f
	return &cp, nil
}

func (s *fakeStore) GetFreezeForUpdate(tx Tx, ctx context.Context, freezeID string) (*Freeze, error) {
	// 在测试环境中 fakeStore 已通过 mutex 串行化，行锁语义天然满足。
	f, ok := s.freezes[freezeID]
	if !ok {
		return nil, nil
	}
	cp := *f
	return &cp, nil
}

func (s *fakeStore) UpdateFreezeStatus(tx Tx, ctx context.Context, freezeID string, status string, settleAmount, settleCost *decimal.Decimal) error {
	f, ok := s.freezes[freezeID]
	if !ok {
		return ErrFreezeNotFound
	}
	f.Status = status
	if settleAmount != nil {
		d := Decimal{*settleAmount}
		f.SettleAmount = &d
	}
	if settleCost != nil {
		d := Decimal{*settleCost}
		f.SettleCost = &d
	}
	if status == FreezeStatusSettled {
		now := time.Now()
		f.SettledAt = &now
	}
	f.UpdatedAt = time.Now()
	return nil
}

func (s *fakeStore) RenewFreeze(tx Tx, ctx context.Context, freezeID string, newExpiresAt string) (int64, error) {
	f, ok := s.freezes[freezeID]
	if !ok || f.Status != FreezeStatusOpen {
		return 0, nil
	}
	exp, err := time.Parse(time.RFC3339Nano, newExpiresAt)
	if err != nil {
		return 0, err
	}
	f.ExpiresAt = exp
	f.RenewalCount++
	now := time.Now()
	f.LastRenewedAt = &now
	f.UpdatedAt = now
	return 1, nil
}

func (s *fakeStore) ListExpiredFreezes(ctx context.Context, limit int) ([]*Freeze, error) {
	var expired []*Freeze
	now := time.Now()
	for _, f := range s.freezes {
		if f.Status == FreezeStatusOpen && now.After(f.ExpiresAt) {
			cp := *f
			expired = append(expired, &cp)
			if len(expired) >= limit {
				break
			}
		}
	}
	return expired, nil
}

func (s *fakeStore) InsertAllocation(tx Tx, ctx context.Context, a *Allocation) error {
	cp := *a
	s.allocations[a.ID] = &cp
	return nil
}

func (s *fakeStore) UpdateAllocationStatus(tx Tx, ctx context.Context, id string, status string) error {
	a, ok := s.allocations[id]
	if !ok {
		return ErrAllocationChannelDenied
	}
	a.Status = status
	if status == AllocationStatusCompleted {
		now := time.Now()
		a.CompletedAt = &now
	}
	return nil
}

func (s *fakeStore) GetLiquidation(ctx context.Context, accountID string) (*Liquidation, error) {
	for _, l := range s.liquidations {
		if l.AccountID == accountID && l.Status != LiquidationStatusClosed {
			cp := *l
			return &cp, nil
		}
	}
	return nil, nil
}

func (s *fakeStore) InsertLiquidation(tx Tx, ctx context.Context, l *Liquidation) error {
	cp := *l
	s.liquidations[l.ID] = &cp
	return nil
}

func (s *fakeStore) UpdateLiquidationStage(tx Tx, ctx context.Context, id string, stage string) error {
	l, ok := s.liquidations[id]
	if !ok {
		return ErrLiquidationStageInvalid
	}
	l.Status = stage
	l.UpdatedAt = time.Now()
	if stage == LiquidationStatusClosed {
		now := time.Now()
		l.ClosedAt = &now
	}
	return nil
}

func (s *fakeStore) CheckIdempotency(ctx context.Context, key string) (*AllocateResult, bool, error) {
	result, ok := s.idempotency[key]
	if !ok {
		return nil, false, nil
	}
	return result, true, nil
}

func (s *fakeStore) StoreIdempotency(tx Tx, ctx context.Context, key string, result *AllocateResult) error {
	cp := *result
	s.idempotency[key] = &cp
	return nil
}

// fakeIdempotencyChecker implements IdempotencyChecker for tests.
// It shares the same idempotency storage maps with the fakeStore so that
// results stored via StoreIdempotency are visible to subsequent Retrieve calls.
type fakeIdempotencyChecker struct {
	mu       sync.Mutex
	keys     map[string]bool
	results  map[string]*AllocateResult
}

func newFakeIdempotencyChecker(keys map[string]bool, results map[string]*AllocateResult) *fakeIdempotencyChecker {
	return &fakeIdempotencyChecker{
		keys:    keys,
		results: results,
	}
}

func (c *fakeIdempotencyChecker) Claim(ctx context.Context, key string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
		if c.keys[key] {
			// Key already claimed — the caller should check Retrieve.
			return false, nil
		}
	c.keys[key] = true
	return true, nil
}

func (c *fakeIdempotencyChecker) Store(ctx context.Context, key string, result any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if r, ok := result.(*AllocateResult); ok {
		cp := *r
		c.results[key] = &cp
	}
	return nil
}

func (c *fakeIdempotencyChecker) Retrieve(ctx context.Context, key string, result any) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	stored, ok := c.results[key]
	if !ok {
		return false, nil
	}
	if r, ok := result.(*AllocateResult); ok {
		*r = *stored
	}
	return true, nil
}

// ---------------------------------------------------------------------------
// Helper: create a test account with a balance.
// ---------------------------------------------------------------------------

func newTestAccount(id string, available, frozen float64) *Account {
	return &Account{
		ID:               id,
		PartyID:          "party-" + id,
		AvailableBalance: DecPtr(available),
		FrozenBalance:    DecPtr(frozen),
		Status:           StatusActive,
		Version:          1,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

// ---------------------------------------------------------------------------
// TestAllocate_Success
// ---------------------------------------------------------------------------

// TestAllocate_Success verifies a normal parent-to-child fund transfer:
// source debited, destination credited, allocation complete.
func TestAllocate_Success(t *testing.T) {
	store := newFakeStore()
	store.accounts["src"] = newTestAccount("src", 1000, 0)
	store.accounts["dst"] = newTestAccount("dst", 0, 0)
	idem := newFakeIdempotencyChecker(store.idempotencyKeys, store.idempotency)

	svc := &Service{Store: store, Idempotency: idem}

	req := AllocateRequest{
		SrcAccountID:   "src",
		DstAccountID:   "dst",
		Amount:         DecPtr(500),
		Channel:        ChannelParent,
		IdempotencyKey: "idem-001",
		OperatorID:     "op-1",
	}

	result, err := svc.Allocate(context.Background(), req)
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}

	if result.SrcBalanceAfter.Decimal.String() != "500" {
		t.Errorf("src balance after = %s, want 500", result.SrcBalanceAfter.Decimal.String())
	}
	if result.DstBalanceAfter.Decimal.String() != "500" {
		t.Errorf("dst balance after = %s, want 500", result.DstBalanceAfter.Decimal.String())
	}
	if result.Status != AllocationStatusCompleted {
		t.Errorf("allocation status = %s, want %s", result.Status, AllocationStatusCompleted)
	}

	// Verify ledger entries.
	if len(store.ledgers) != 2 {
		t.Fatalf("expected 2 ledger entries, got %d", len(store.ledgers))
	}
	if store.ledgers[0].Direction != DirectionAllocateOut {
		t.Errorf("first ledger direction = %s, want %s", store.ledgers[0].Direction, DirectionAllocateOut)
	}
	if store.ledgers[1].Direction != DirectionAllocateIn {
		t.Errorf("second ledger direction = %s, want %s", store.ledgers[1].Direction, DirectionAllocateIn)
	}
}

// ---------------------------------------------------------------------------
// TestAllocate_Conservation
// ---------------------------------------------------------------------------

// TestAllocate_Conservation verifies F-CON-02: the sum of source and destination
// balance changes is zero (total money preserved after allocation).
func TestAllocate_Conservation(t *testing.T) {
	store := newFakeStore()
	store.accounts["src"] = newTestAccount("src", 1000, 0)
	store.accounts["dst"] = newTestAccount("dst", 500, 0)

	svc := &Service{Store: store}

	totalBefore := store.accounts["src"].AvailableBalance.Decimal.Add(
		store.accounts["dst"].AvailableBalance.Decimal)

	req := AllocateRequest{
		SrcAccountID: "src",
		DstAccountID: "dst",
		Amount:       DecPtr(300),
		Channel:      ChannelParent,
		OperatorID:   "op-1",
		IdempotencyKey: "idem-conservation",
	}

	_, err := svc.Allocate(context.Background(), req)
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}

	totalAfter := store.accounts["src"].AvailableBalance.Decimal.Add(
		store.accounts["dst"].AvailableBalance.Decimal)

	if !totalBefore.Equal(totalAfter) {
		t.Errorf("conservation violation: total before = %s, total after = %s",
			totalBefore.String(), totalAfter.String())
	}
}

// ---------------------------------------------------------------------------
// TestAllocate_Idempotency
// ---------------------------------------------------------------------------

// TestAllocate_Idempotency verifies that a second call with the same
// idempotency key returns the original result without double-charging.
func TestAllocate_Idempotency(t *testing.T) {
	store := newFakeStore()
	store.accounts["src"] = newTestAccount("src", 1000, 0)
	store.accounts["dst"] = newTestAccount("dst", 0, 0)
	idem := newFakeIdempotencyChecker(store.idempotencyKeys, store.idempotency)

	svc := &Service{Store: store, Idempotency: idem}

	req := AllocateRequest{
		SrcAccountID:   "src",
		DstAccountID:   "dst",
		Amount:         DecPtr(100),
		Channel:        ChannelParent,
		IdempotencyKey: "idem-dup",
		OperatorID:     "op-1",
	}

	// First call.
	result1, err := svc.Allocate(context.Background(), req)
	if err != nil {
		t.Fatalf("first Allocate failed: %v", err)
	}

	// Second call with same key.
	result2, err := svc.Allocate(context.Background(), req)
	if err != nil {
		t.Fatalf("second Allocate failed: %v", err)
	}

	if result1.AllocationID != result2.AllocationID {
		t.Errorf("idempotency failed: allocation IDs differ: %s vs %s",
			result1.AllocationID, result2.AllocationID)
	}

	// Verify balances were only changed once.
	src := store.accounts["src"]
	if src.AvailableBalance.Decimal.String() != "900" {
		t.Errorf("src balance = %s, want 900 (only debited once)", src.AvailableBalance.Decimal.String())
	}

	if len(store.ledgers) != 2 {
		t.Errorf("expected exactly 2 ledger entries, got %d", len(store.ledgers))
	}
}

// ---------------------------------------------------------------------------
// TestAllocate_ChannelDenied
// ---------------------------------------------------------------------------

// TestAllocate_ChannelDenied verifies that an allocation outside the allowed
// channel types is rejected.
func TestAllocate_ChannelDenied(t *testing.T) {
	store := newFakeStore()
	store.accounts["src"] = newTestAccount("src", 1000, 0)
	store.accounts["dst"] = newTestAccount("dst", 0, 0)

	svc := &Service{Store: store}

	req := AllocateRequest{
		SrcAccountID:   "src",
		DstAccountID:   "dst",
		Amount:         DecPtr(100),
		Channel:        "owns", // owns does not permit fund transfers.
		OperatorID:     "op-1",
		IdempotencyKey: "idem-channel-denied",
	}

	_, err := svc.Allocate(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for denied channel, got nil")
	}
	if !errorsIs(err, ErrAllocationChannelDenied) {
		t.Errorf("error chain does not contain ErrAllocationChannelDenied: %v", err)
	}

	// Balances should be unchanged.
	src := store.accounts["src"]
	if src.AvailableBalance.Decimal.String() != "1000" {
		t.Errorf("src balance changed: expected 1000, got %s", src.AvailableBalance.Decimal.String())
	}
}

// ---------------------------------------------------------------------------
// TestAllocate_IdempotencyKeyRequired
// ---------------------------------------------------------------------------

// TestAllocate_IdempotencyKeyRequired 验证空 IdempotencyKey 被拒绝（RED-2 安全修复）。
// 所有划拨操作必须提供幂等键——无例外。
func TestAllocate_IdempotencyKeyRequired(t *testing.T) {
	store := newFakeStore()
	store.accounts["src"] = newTestAccount("src", 1000, 0)
	store.accounts["dst"] = newTestAccount("dst", 0, 0)

	svc := &Service{Store: store}

	req := AllocateRequest{
		SrcAccountID: "src",
		DstAccountID: "dst",
		Amount:       DecPtr(100),
		Channel:      ChannelParent,
		OperatorID:   "op-1",
		// IdempotencyKey 故意留空——应被拒绝。
	}

	_, err := svc.Allocate(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing idempotency key, got nil")
	}
	if !errorsIs(err, ErrIdempotencyKeyRequired) {
		t.Errorf("error chain does not contain ErrIdempotencyKeyRequired: %v", err)
	}

	// 余额不应发生变化。
	src := store.accounts["src"]
	if src.AvailableBalance.Decimal.String() != "1000" {
		t.Errorf("src balance changed: expected 1000, got %s", src.AvailableBalance.Decimal.String())
	}
}

// errorsIs is a local helper for tests that checks if the error chain contains
// the target sentinel.
func errorsIs(err, target error) bool {
	if err == nil {
		return false
	}
	for {
		if err == target {
			return true
		}
		// Check FundError wrapping.
		if fe, ok := err.(*FundError); ok {
			err = fe.Err
			continue
		}
		// Try standard unwrapping.
		type unwrapper interface{ Unwrap() error }
		if uw, ok := err.(unwrapper); ok {
			err = uw.Unwrap()
			if err == nil {
				return false
			}
			continue
		}
		return false
	}
}

// ---------------------------------------------------------------------------
// TestFreeze_Success
// ---------------------------------------------------------------------------

// TestFreeze_Success verifies normal freeze: sufficient balance, no budget cap,
// freeze created with correct status and ledger entry.
func TestFreeze_Success(t *testing.T) {
	store := newFakeStore()
	store.accounts["acct"] = newTestAccount("acct", 1000, 0)

	svc := &Service{Store: store}

	req := FreezeRequest{
		AccountID:     "acct",
		Amount:        DecPtr(200),
		EstimatedSell: DecPtr(200),
		RequestID:     "req-1",
		UserID:        "user-1",
	}

	result, err := svc.Freeze(context.Background(), req)
	if err != nil {
		t.Fatalf("Freeze failed: %v", err)
	}

	if result.Amount.Decimal.String() != "200" {
		t.Errorf("frozen amount = %s, want 200", result.Amount.Decimal.String())
	}
	if result.BalanceAfter.Decimal.String() != "800" {
		t.Errorf("balance after = %s, want 800", result.BalanceAfter.Decimal.String())
	}
	if result.FrozenAfter.Decimal.String() != "200" {
		t.Errorf("frozen after = %s, want 200", result.FrozenAfter.Decimal.String())
	}

	// Verify the account state.
	acct := store.accounts["acct"]
	if acct.AvailableBalance.Decimal.String() != "800" {
		t.Errorf("account available = %s, want 800", acct.AvailableBalance.Decimal.String())
	}
	if acct.FrozenBalance.Decimal.String() != "200" {
		t.Errorf("account frozen = %s, want 200", acct.FrozenBalance.Decimal.String())
	}

	// Verify freeze record exists and is open.
	if len(store.freezes) != 1 {
		t.Fatalf("expected 1 freeze, got %d", len(store.freezes))
	}
	for _, f := range store.freezes {
		if f.Status != FreezeStatusOpen {
			t.Errorf("freeze status = %s, want %s", f.Status, FreezeStatusOpen)
		}
		if f.AccountID != "acct" {
			t.Errorf("freeze account = %s, want acct", f.AccountID)
		}
	}
}

// ---------------------------------------------------------------------------
// TestFreeze_InsufficientBalance
// ---------------------------------------------------------------------------

// TestFreeze_InsufficientBalance verifies that attempting to freeze more than
// the available balance returns ErrInsufficientBalance.
func TestFreeze_InsufficientBalance(t *testing.T) {
	store := newFakeStore()
	store.accounts["acct"] = newTestAccount("acct", 100, 0)

	svc := &Service{Store: store}

	req := FreezeRequest{
		AccountID:     "acct",
		Amount:        DecPtr(500),
		EstimatedSell: DecPtr(500),
		RequestID:     "req-1",
		UserID:        "user-1",
	}

	_, err := svc.Freeze(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for insufficient balance, got nil")
	}
	if !errorsIs(err, ErrInsufficientBalance) {
		t.Errorf("error chain does not contain ErrInsufficientBalance: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestFreeze_BudgetCapExceeded
// ---------------------------------------------------------------------------

// TestFreeze_BudgetCapExceeded verifies that when budget_consumed + new estimate
// exceeds budget_limit_amount, the freeze is rejected.
func TestFreeze_BudgetCapExceeded(t *testing.T) {
	store := newFakeStore()
	acct := newTestAccount("acct", 500, 0)
	limit := DecPtr(400)
	acct.BudgetLimitAmount = &limit
	acct.BudgetConsumedAmount = DecPtr(350)
	store.accounts["acct"] = acct

	svc := &Service{Store: store}

	req := FreezeRequest{
		AccountID:     "acct",
		Amount:        DecPtr(100),
		EstimatedSell: DecPtr(100), // 350 + 100 = 450 > 400
		RequestID:     "req-1",
		UserID:        "user-1",
	}

	_, err := svc.Freeze(context.Background(), req)
	if err == nil {
		t.Fatal("expected budget cap error, got nil")
	}
	if !errorsIs(err, ErrBudgetCapExceeded) {
		t.Errorf("error chain does not contain ErrBudgetCapExceeded: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestSettle_Normal
// ---------------------------------------------------------------------------

// TestSettle_Normal verifies settlement when actual_sell equals frozen amount.
func TestSettle_Normal(t *testing.T) {
	store := newFakeStore()
	store.accounts["acct"] = newTestAccount("acct", 200, 500)

	// Insert an existing freeze.
	freeze := &Freeze{
		ID:            "fr-1",
		AccountID:     "acct",
		Amount:        DecPtr(100),
		EstimatedSell: DecPtr(100),
		Status:        FreezeStatusOpen,
		ExpiresAt:     time.Now().Add(1 * time.Hour),
		UserID:        "user-1",
	}
	store.freezes[freeze.ID] = freeze

	svc := &Service{Store: store}

	req := SettleRequest{
		FreezeID:   "fr-1",
		ActualSell: DecPtr(100),
		ActualCost: DecPtr(70),
		RequestID:  "req-1",
	}

	result, err := svc.Settle(context.Background(), req)
	if err != nil {
		t.Fatalf("Settle failed: %v", err)
	}

	if result.ActualSell.Decimal.String() != "100" {
		t.Errorf("actual sell = %s, want 100", result.ActualSell.Decimal.String())
	}
	// Refund should be 0 since sell == frozen.
	if !result.ReleasedAmount.Decimal.IsZero() {
		t.Errorf("released amount = %s, want 0", result.ReleasedAmount.Decimal.String())
	}

	// Account: available was 200; frozen 500 - 100 = 400.
	// After settle: sell=100, refund=0, available=200+0=200 (sell deducted was from frozen)
	acct := store.accounts["acct"]
	if acct.FrozenBalance.Decimal.String() != "400" {
		t.Errorf("account frozen after = %s, want 400", acct.FrozenBalance.Decimal.String())
	}

	// Verify freeze status.
	f := store.freezes["fr-1"]
	if f.Status != FreezeStatusSettled {
		t.Errorf("freeze status = %s, want %s", f.Status, FreezeStatusSettled)
	}
}

// ---------------------------------------------------------------------------
// TestSettle_Refund
// ---------------------------------------------------------------------------

// TestSettle_Refund verifies settlement when actual_sell is less than frozen,
// resulting in a refund of the difference to available balance.
func TestSettle_Refund(t *testing.T) {
	store := newFakeStore()
	store.accounts["acct"] = newTestAccount("acct", 100, 500)

	freeze := &Freeze{
		ID:            "fr-2",
		AccountID:     "acct",
		Amount:        DecPtr(100),
		EstimatedSell: DecPtr(100),
		Status:        FreezeStatusOpen,
		ExpiresAt:     time.Now().Add(1 * time.Hour),
		UserID:        "user-1",
	}
	store.freezes[freeze.ID] = freeze

	svc := &Service{Store: store}

	req := SettleRequest{
		FreezeID:   "fr-2",
		ActualSell: DecPtr(30),
		ActualCost: DecPtr(20),
		RequestID:  "req-1",
	}

	result, err := svc.Settle(context.Background(), req)
	if err != nil {
		t.Fatalf("Settle failed: %v", err)
	}

	// Refund should be 100 - 30 = 70.
	if result.ReleasedAmount.Decimal.String() != "70" {
		t.Errorf("released amount = %s, want 70", result.ReleasedAmount.Decimal.String())
	}

	// Available: 100 + 70 (refund) = 170.
	acct := store.accounts["acct"]
	if acct.AvailableBalance.Decimal.String() != "170" {
		t.Errorf("account available after = %s, want 170", acct.AvailableBalance.Decimal.String())
	}
	// Frozen: 500 - 100 = 400.
	if acct.FrozenBalance.Decimal.String() != "400" {
		t.Errorf("account frozen after = %s, want 400", acct.FrozenBalance.Decimal.String())
	}
}

// ---------------------------------------------------------------------------
// TestFreeze_TimeoutRelease
// ---------------------------------------------------------------------------

// TestFreeze_TimeoutRelease verifies that expired freezes are released by
// the TTL scanner, returning funds to available balance.
func TestFreeze_TimeoutRelease(t *testing.T) {
	store := newFakeStore()
	store.accounts["acct"] = newTestAccount("acct", 200, 500)

	// Insert an expired freeze.
	expiredFreeze := &Freeze{
		ID:            "fr-exp",
		AccountID:     "acct",
		Amount:        DecPtr(50),
		EstimatedSell: DecPtr(50),
		Status:        FreezeStatusOpen,
		ExpiresAt:     time.Now().Add(-1 * time.Hour), // expired 1 hour ago.
		UserID:        "user-1",
	}
	store.freezes[expiredFreeze.ID] = expiredFreeze

	svc := &Service{Store: store}

	count, err := svc.UnfreezeTimeout(context.Background())
	if err != nil {
		t.Fatalf("UnfreezeTimeout failed: %v", err)
	}
	if count != 1 {
		t.Errorf("released count = %d, want 1", count)
	}

	// Verify the freeze is now timeout_released.
	f := store.freezes["fr-exp"]
	if f.Status != FreezeStatusTimeoutReleased {
		t.Errorf("freeze status = %s, want %s", f.Status, FreezeStatusTimeoutReleased)
	}

	// Available: 200 + 50 = 250; Frozen: 500 - 50 = 450.
	acct := store.accounts["acct"]
	if acct.AvailableBalance.Decimal.String() != "250" {
		t.Errorf("available after release = %s, want 250", acct.AvailableBalance.Decimal.String())
	}
	if acct.FrozenBalance.Decimal.String() != "450" {
		t.Errorf("frozen after release = %s, want 450", acct.FrozenBalance.Decimal.String())
	}
}

// ---------------------------------------------------------------------------
// TestLiquidate_StateMachine
// ---------------------------------------------------------------------------

// TestLiquidate_StateMachine verifies the full liquidation lifecycle:
// active -> blocking -> draining -> refunding -> closing -> closed.
func TestLiquidate_StateMachine(t *testing.T) {
	store := newFakeStore()
	store.accounts["src"] = newTestAccount("src", 500, 0)
	store.accounts["dst"] = newTestAccount("dst", 0, 0)

	svc := &Service{Store: store}

	// Step 1: active -> blocking.
	req := LiquidateRequest{
		AccountID:       "src",
		TargetAccountID: "dst",
		OperatorID:      "op-1",
		PartyID:         "party-src",
		Reason:          "project closure",
	}
	result, err := svc.Liquidate(context.Background(), req)
	if err != nil {
		t.Fatalf("Liquidate (start) failed: %v", err)
	}
	if result.Status != LiquidationStatusBlocking {
		t.Errorf("step 1 status = %s, want %s", result.Status, LiquidationStatusBlocking)
	}
	if store.accounts["src"].Status != StatusLiquidatingBlockNew {
		t.Errorf("account status = %s, want %s", store.accounts["src"].Status, StatusLiquidatingBlockNew)
	}

	// Step 2: blocking -> draining.
	result, err = svc.Liquidate(context.Background(), req)
	if err != nil {
		t.Fatalf("Liquidate (draining) failed: %v", err)
	}
	if result.Status != LiquidationStatusDraining {
		t.Errorf("step 2 status = %s, want %s", result.Status, LiquidationStatusDraining)
	}

	// Step 3: draining -> refunding (balance transferred).
	result, err = svc.Liquidate(context.Background(), req)
	if err != nil {
		t.Fatalf("Liquidate (refunding) failed: %v", err)
	}
	if result.Status != LiquidationStatusRefunding {
		t.Errorf("step 3 status = %s, want %s", result.Status, LiquidationStatusRefunding)
	}

	// Verify balance was transferred.
	srcAcct := store.accounts["src"]
	dstAcct := store.accounts["dst"]
	if srcAcct.AvailableBalance.Decimal.String() != "0" {
		t.Errorf("src available = %s, want 0", srcAcct.AvailableBalance.Decimal.String())
	}
	if dstAcct.AvailableBalance.Decimal.String() != "500" {
		t.Errorf("dst available = %s, want 500", dstAcct.AvailableBalance.Decimal.String())
	}

	// Step 4: refunding -> closing.
	result, err = svc.Liquidate(context.Background(), req)
	if err != nil {
		t.Fatalf("Liquidate (closing) failed: %v", err)
	}
	if result.Status != LiquidationStatusClosing {
		t.Errorf("step 4 status = %s, want %s", result.Status, LiquidationStatusClosing)
	}
	if store.accounts["src"].Status != StatusLiquidatingTransfer {
		t.Errorf("account status = %s, want %s", store.accounts["src"].Status, StatusLiquidatingTransfer)
	}

	// Step 5: closing -> closed (terminal).
	result, err = svc.Liquidate(context.Background(), req)
	if err != nil {
		t.Fatalf("Liquidate (closed) failed: %v", err)
	}
	if result.Status != LiquidationStatusClosed {
		t.Errorf("step 5 status = %s, want %s", result.Status, LiquidationStatusClosed)
	}
	if store.accounts["src"].Status != StatusClosed {
		t.Errorf("account status = %s, want %s", store.accounts["src"].Status, StatusClosed)
	}
}
