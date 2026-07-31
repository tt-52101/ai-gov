package idempotency

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupDB creates an in-memory SQLite database with the idempotency_records
// table pre-migrated. Each test gets a fresh database, guaranteeing test
// isolation (AGENTS.md §6.5).
func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory SQLite: %v", err)
	}
	if err := db.AutoMigrate(&Record{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

// newRecord creates a Record with sensible defaults for testing.
// The ID is derived from idempotencyKey to ensure uniqueness within a test.
func newRecord(id, scope, actorID, idempotencyKey, body string) *Record {
	hash := sha256.Sum256([]byte(body))
	return &Record{
		ID:             id,
		Scope:          scope,
		ActorID:        actorID,
		IdempotencyKey: idempotencyKey,
		RequestHash:    hex.EncodeToString(hash[:]),
		Status:         StatusStarted,
		ExpiresAt:      time.Now().Add(DefaultRetentionWindow),
	}
}

// testRecord is a convenience helper that creates and claims a record,
// returning the DB and the record. Fails the test if claiming fails.
func testClaim(t *testing.T, db *gorm.DB, rec *Record) {
	t.Helper()
	claimed, _, err := Claim(context.Background(), db, rec)
	if err != nil {
		t.Fatalf("Claim failed: %v", err)
	}
	if !claimed {
		t.Fatal("expected key to be newly claimed")
	}
}

// ──────────────────────────────────────────────────────────────────────
// Claim tests
// ──────────────────────────────────────────────────────────────────────

func TestClaim_FirstAttempt(t *testing.T) {
	db := setupDB(t)
	rec := newRecord("rec-1", "allocate", "actor-1", "550e8400-e29b-41d4-a716-446655440000", "payload-v1")

	claimed, existing, err := Claim(context.Background(), db, rec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !claimed {
		t.Fatal("expected key to be claimed on first attempt")
	}
	if existing != nil {
		t.Fatal("expected no existing record on first claim")
	}

	// Verify the record was actually persisted.
	stored, err := GetRecord(db, rec.Scope, rec.ActorID, rec.IdempotencyKey)
	if err != nil {
		t.Fatalf("failed to retrieve record: %v", err)
	}
	if stored.Status != StatusStarted {
		t.Errorf("expected status %q, got %q", StatusStarted, stored.Status)
	}
}

func TestClaim_SameFingerprintReplay(t *testing.T) {
	db := setupDB(t)
	payload := "allocate-100-to-project-alpha"
	rec := newRecord("rec-2", "allocate", "actor-2", "660e8400-e29b-41d4-a716-446655440001", payload)

	// First claim.
	testClaim(t, db, rec)

	// Complete the operation.
	if err := Complete(context.Background(), db, rec.ID, 200, []byte(`{"ok":true}`), "alloc-001"); err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	// Second attempt with the same key and same payload.
	rec2 := newRecord("rec-2b", "allocate", "actor-2", "660e8400-e29b-41d4-a716-446655440001", payload)
	claimed, existing, err := Claim(context.Background(), db, rec2)

	if err == nil {
		t.Fatal("expected replay error, got nil")
	}
	if !errors.Is(err, ErrIdempotencyReplay) {
		t.Errorf("expected ErrIdempotencyReplay, got %v", err)
	}
	if claimed {
		t.Fatal("expected claimed=false for replay")
	}
	if existing == nil {
		t.Fatal("expected existing record to be returned")
	}
	if existing.Status != StatusSucceeded {
		t.Errorf("expected StatusSucceeded, got %q", existing.Status)
	}
}

func TestClaim_DifferentFingerprintConflict(t *testing.T) {
	db := setupDB(t)
	rec1 := newRecord("rec-3", "allocate", "actor-3", "770e8400-e29b-41d4-a716-446655440002", "payload-v1")

	// First claim.
	testClaim(t, db, rec1)

	// Second attempt: same key, different payload.
	rec2 := newRecord("rec-3b", "allocate", "actor-3", "770e8400-e29b-41d4-a716-446655440002", "DIFFERENT-payload-v2")
	claimed, existing, err := Claim(context.Background(), db, rec2)

	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Errorf("expected ErrIdempotencyConflict, got %v", err)
	}
	if claimed {
		t.Fatal("expected claimed=false for conflict")
	}
	if existing != nil {
		t.Fatal("expected nil existing record for conflict")
	}
}

func TestClaim_ExpiredKeyReclaim(t *testing.T) {
	db := setupDB(t)

	// Create a record that expires in the past.
	rec := newRecord("rec-4", "liquidate", "actor-4", "880e8400-e29b-41d4-a716-446655440003", "payload-v1")
	rec.ExpiresAt = time.Now().Add(-1 * time.Hour) // already expired

	// First claim: should succeed even though it looks like a duplicate
	// because we set it up already-expired.
	// Actually, we need a real previously-inserted record that's expired.
	// Insert directly via DB.
	oldRec := newRecord("rec-4-old", "liquidate", "actor-4", "880e8400-e29b-41d4-a716-446655440003", "old-payload")
	oldRec.ExpiresAt = time.Now().Add(-1 * time.Hour)
	oldRec.Status = StatusSucceeded
	encoded := EncodeResponse(200, []byte(`{"old":true}`))
	oldRec.ResponseJSON = &encoded
	ref := "old-ref"
	oldRec.ResourceRef = &ref

	if err := InsertRecord(db, oldRec); err != nil {
		t.Fatalf("failed to insert old record: %v", err)
	}

	// Now try to claim with the same key but a different payload.
	// The old record is expired, so reclaim should succeed.
	newRec := newRecord("rec-4-new", "liquidate", "actor-4", "880e8400-e29b-41d4-a716-446655440003", "new-payload")
	claimed, existing, err := Claim(context.Background(), db, newRec)

	if err != nil {
		t.Fatalf("unexpected error during reclaim: %v", err)
	}
	if !claimed {
		t.Fatal("expected key to be reclaimed after expiry")
	}
	if existing != nil {
		t.Fatal("expected nil existing record after successful reclaim")
	}

	// Verify the new record is in the database.
	stored, err := GetRecord(db, newRec.Scope, newRec.ActorID, newRec.IdempotencyKey)
	if err != nil {
		t.Fatalf("failed to retrieve reclaimed record: %v", err)
	}
	if stored.ID != "rec-4-new" {
		t.Errorf("expected reclaimed record ID %q, got %q", "rec-4-new", stored.ID)
	}
	if stored.RequestHash != newRec.RequestHash {
		t.Errorf("expected hash %q, got %q", newRec.RequestHash, stored.RequestHash)
	}
}

func TestClaim_ConcurrentInsert(t *testing.T) {
	// Simulate concurrent INSERT attempts with the same key.
	// Only one goroutine should succeed; all others should get a replay,
	// conflict, or in-progress depending on timing.
	//
	// Uses a file-based SQLite database with WAL disabled to avoid
	// file-locking issues on Windows during cleanup.
	dir := t.TempDir()
	dbPath := dir + "/test.db"
	db, err := gorm.Open(sqlite.Open(dbPath+"?_journal_mode=DELETE"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open SQLite: %v", err)
	}
	if err := db.AutoMigrate(&Record{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Ensure the database connection is closed before cleanup.
	if s, err := db.DB(); err == nil {
		defer s.Close()
	}

	payload := "concurrent-payload"
	key := "990e8400-e29b-41d4-a716-446655440004"
	scope := "allocate"
	actor := "actor-concurrent"

	const numWorkers = 10
	var wg sync.WaitGroup
	results := make(chan bool, numWorkers)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			id := "rec-concurrent-" + string(rune('a'+idx))
			rec := newRecord(id, scope, actor, key, payload)
			claimed, _, err := Claim(context.Background(), db, rec)
			if err != nil {
				// Replay or in-progress is acceptable (same hash).
				if errors.Is(err, ErrIdempotencyReplay) || errors.Is(err, ErrIdempotencyInProgress) {
					results <- false
					return
				}
				// Other errors are unexpected.
				t.Errorf("unexpected error from worker %d: %v", idx, err)
				results <- false
				return
			}
			results <- claimed
		}(i)
	}

	wg.Wait()
	close(results)

	claimedCount := 0
	for claimed := range results {
		if claimed {
			claimedCount++
		}
	}

	if claimedCount != 1 {
		t.Errorf("expected exactly 1 goroutine to claim the key, got %d", claimedCount)
	}
}

func TestClaim_DifferentScopeSameKey(t *testing.T) {
	// Same actor + key, different scope: should be two independent records.
	db := setupDB(t)

	payload := "payload-multi-scope"
	rec1 := newRecord("rec-scope-1", "allocate", "actor-s", "aa0e8400-e29b-41d4-a716-446655440005", payload)
	rec2 := newRecord("rec-scope-2", "liquidate", "actor-s", "aa0e8400-e29b-41d4-a716-446655440005", payload)

	claimed1, _, err := Claim(context.Background(), db, rec1)
	if err != nil || !claimed1 {
		t.Fatalf("first claim failed: err=%v, claimed=%v", err, claimed1)
	}

	claimed2, _, err := Claim(context.Background(), db, rec2)
	if err != nil {
		t.Fatalf("second claim (different scope) failed: %v", err)
	}
	if !claimed2 {
		t.Fatal("expected second claim with different scope to succeed")
	}
}

// ──────────────────────────────────────────────────────────────────────
// Complete / Fail tests
// ──────────────────────────────────────────────────────────────────────

func TestComplete_SetsStatusAndResponse(t *testing.T) {
	db := setupDB(t)
	rec := newRecord("rec-comp-1", "allocate", "actor-c1", "bb0e8400-e29b-41d4-a716-446655440006", "payload-comp-1")
	testClaim(t, db, rec)

	code := 201
	body := []byte(`{"transaction_id":"txn-001","status":"completed"}`)
	ref := "txn-001"

	if err := Complete(context.Background(), db, rec.ID, code, body, ref); err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	// Verify the record was updated.
	stored, err := GetRecord(db, rec.Scope, rec.ActorID, rec.IdempotencyKey)
	if err != nil {
		t.Fatalf("failed to retrieve completed record: %v", err)
	}
	if stored.Status != StatusSucceeded {
		t.Errorf("expected StatusSucceeded, got %q", stored.Status)
	}
	if stored.ResourceRef == nil || *stored.ResourceRef != ref {
		t.Errorf("expected resource_ref %q, got %v", ref, stored.ResourceRef)
	}

	decodedCode, decodedBody := DecodeResponse(stored.ResponseJSON)
	if decodedCode != code {
		t.Errorf("expected response code %d, got %d", code, decodedCode)
	}
	if string(decodedBody) != string(body) {
		t.Errorf("expected body %q, got %q", string(body), string(decodedBody))
	}
}

func TestComplete_NonStartedRecordFails(t *testing.T) {
	db := setupDB(t)
	rec := newRecord("rec-comp-2", "allocate", "actor-c2", "cc0e8400-e29b-41d4-a716-446655440007", "payload-comp-2")
	testClaim(t, db, rec)

	// Complete once.
	if err := Complete(context.Background(), db, rec.ID, 200, []byte(`{}`), "ref-1"); err != nil {
		t.Fatalf("first Complete failed: %v", err)
	}

	// Complete again on the same record — should fail.
	err := Complete(context.Background(), db, rec.ID, 200, []byte(`{}`), "ref-2")
	if err == nil {
		t.Fatal("expected second Complete to fail on a non-Started record")
	}
}

func TestFail_SetsStatusFailed(t *testing.T) {
	db := setupDB(t)
	rec := newRecord("rec-fail-1", "allocate", "actor-f1", "dd0e8400-e29b-41d4-a716-446655440008", "payload-fail-1")
	testClaim(t, db, rec)

	if err := Fail(context.Background(), db, rec.ID, "balance rollback failed"); err != nil {
		t.Fatalf("Fail failed: %v", err)
	}

	stored, err := GetRecord(db, rec.Scope, rec.ActorID, rec.IdempotencyKey)
	if err != nil {
		t.Fatalf("failed to retrieve failed record: %v", err)
	}
	if stored.Status != StatusFailed {
		t.Errorf("expected StatusFailed, got %q", stored.Status)
	}
}

func TestFail_NonStartedRecordFails(t *testing.T) {
	db := setupDB(t)
	rec := newRecord("rec-fail-2", "allocate", "actor-f2", "ee0e8400-e29b-41d4-a716-446655440009", "payload-fail-2")
	testClaim(t, db, rec)

	// Fail once.
	if err := Fail(context.Background(), db, rec.ID, "first error"); err != nil {
		t.Fatalf("first Fail failed: %v", err)
	}

	// Fail again on the same record — should fail.
	err := Fail(context.Background(), db, rec.ID, "second error")
	if err == nil {
		t.Fatal("expected second Fail to be rejected on a non-Started record")
	}
}

func TestReplay_FailedRecordReturnsConflict(t *testing.T) {
	// After Fail, a replay with the same key+hash should return
	// ErrIdempotencyConflict, not ErrIdempotencyReplay.
	db := setupDB(t)
	payload := "payload-failed-replay"
	key := "ff0e8400-e29b-41d4-a716-446655440010"

	rec := newRecord("rec-fr-1", "allocate", "actor-fr", key, payload)
	testClaim(t, db, rec)

	if err := Fail(context.Background(), db, rec.ID, "operation failed"); err != nil {
		t.Fatalf("Fail failed: %v", err)
	}

	// Replay same key+hash.
	rec2 := newRecord("rec-fr-2", "allocate", "actor-fr", key, payload)
	claimed, existing, err := Claim(context.Background(), db, rec2)

	if err == nil {
		t.Fatal("expected error on replay of failed record")
	}
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Errorf("expected ErrIdempotencyConflict for failed record replay, got %v", err)
	}
	if claimed {
		t.Fatal("expected claimed=false")
	}
	if existing == nil {
		t.Fatal("expected existing record to be returned")
	}
	if existing.Status != StatusFailed {
		t.Errorf("expected StatusFailed, got %q", existing.Status)
	}
}

// ──────────────────────────────────────────────────────────────────────
// ValidateKey tests
// ──────────────────────────────────────────────────────────────────────

func TestValidateKey_ValidUUID(t *testing.T) {
	validKeys := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"6ba7b810-9dad-41d1-80b4-00c04fd430c8",
		"6ba7b811-9dad-41d1-80b4-00c04fd430c8",
		"00000000-0000-4000-8000-000000000000",
		"ffffffff-ffff-4fff-bfff-ffffffffffff",
	}
	for _, key := range validKeys {
		if err := ValidateKey(key); err != nil {
			t.Errorf("ValidateKey(%q): expected valid, got %v", key, err)
		}
	}
}

func TestValidateKey_InvalidUUID(t *testing.T) {
	invalidKeys := []string{
		"",                            // empty
		"not-a-uuid",                  // wrong format
		"550e8400-e29b-31d4-a716-446655440000", // version nibble != 4
		"550e8400-e29b-41d4-c716-446655440000", // variant nibble invalid
		"550e8400-e29b-41d4-a716-44665544000",  // too short
		"550e8400-e29b-41d4-a716-4466554400000", // too long
		"550e8400-e29b-41d4-a716-44665544000g",  // non-hex character
	}
	for _, key := range invalidKeys {
		if err := ValidateKey(key); err == nil {
			t.Errorf("ValidateKey(%q): expected error, got nil", key)
		}
	}
}

func TestValidateKey_TooLong(t *testing.T) {
	longKey := "550e8400-e29b-41d4-a716-446655440000"
	for len(longKey) <= MaxKeyLength {
		longKey += "-extra"
	}
	if err := ValidateKey(longKey); err == nil {
		t.Errorf("expected error for key of length %d (>%d)", len(longKey), MaxKeyLength)
	}
}

func TestValidateKey_Empty(t *testing.T) {
	if err := ValidateKey(""); err == nil {
		t.Fatal("expected error for empty key")
	}
}

// ──────────────────────────────────────────────────────────────────────
// Middleware tests
// ──────────────────────────────────────────────────────────────────────

func TestMiddleware_PassesThroughGET(t *testing.T) {
	// GET requests should pass through without validation.
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestMiddleware_InjectsValidKey(t *testing.T) {
	validKey := "550e8400-e29b-41d4-a716-446655440000"

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, ok := GetIdempotencyKey(r.Context())
		if !ok {
			t.Error("expected idempotency key in context")
		}
		if key != validKey {
			t.Errorf("expected key %q, got %q", validKey, key)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/allocate", nil)
	req.Header.Set(IdempotencyKeyHeader, validKey)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestMiddleware_RejectsInvalidKey(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for invalid key")
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/allocate", nil)
	req.Header.Set(IdempotencyKeyHeader, "not-a-valid-uuid")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid key, got %d", rec.Code)
	}
}

func TestMiddleware_NoKeyIsFine(t *testing.T) {
	// POST without Idempotency-Key should pass through.
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := GetIdempotencyKey(r.Context())
		if ok {
			t.Error("expected no idempotency key in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/allocate", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

// ──────────────────────────────────────────────────────────────────────
// Store tests
// ──────────────────────────────────────────────────────────────────────

func TestInsertRecord_UniqueConstraint(t *testing.T) {
	db := setupDB(t)
	rec := newRecord("rec-store-1", "allocate", "actor-store", "111e8400-e29b-41d4-a716-446655440011", "payload-store")

	if err := InsertRecord(db, rec); err != nil {
		t.Fatalf("first insert should succeed: %v", err)
	}

	// Insert the same key again with a different ID — should fail.
	rec2 := newRecord("rec-store-2", "allocate", "actor-store", "111e8400-e29b-41d4-a716-446655440011", "payload-store")
	err := InsertRecord(db, rec2)
	if err == nil {
		t.Fatal("expected unique constraint violation on second insert")
	}
	if !isDuplicateKeyError(err) {
		t.Errorf("expected duplicate key error, got %v", err)
	}
}

func TestGetRecord_NotFound(t *testing.T) {
	db := setupDB(t)
	_, err := GetRecord(db, "allocate", "actor-nonexistent", "00000000-0000-4000-8000-000000000000")
	if err == nil {
		t.Fatal("expected error for non-existent record")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}

func TestCleanExpired_RemovesExpiredRecords(t *testing.T) {
	db := setupDB(t)

	// Insert an expired record directly.
	expired := newRecord("rec-expired", "allocate", "actor-clean", "222e8400-e29b-41d4-a716-446655440012", "expired")
	expired.ExpiresAt = time.Now().Add(-1 * time.Hour)
	if err := InsertRecord(db, expired); err != nil {
		t.Fatalf("failed to insert expired record: %v", err)
	}

	// Insert a non-expired record.
	valid := newRecord("rec-valid", "allocate", "actor-clean", "333e8400-e29b-41d4-a716-446655440013", "valid")
	if err := InsertRecord(db, valid); err != nil {
		t.Fatalf("failed to insert valid record: %v", err)
	}

	deleted, err := CleanExpired(db)
	if err != nil {
		t.Fatalf("CleanExpired failed: %v", err)
	}
	if deleted < 1 {
		t.Errorf("expected at least 1 deleted record, got %d", deleted)
	}

	// Verify valid record still exists.
	_, err = GetRecord(db, valid.Scope, valid.ActorID, valid.IdempotencyKey)
	if err != nil {
		t.Errorf("valid record should still exist: %v", err)
	}
}

// ──────────────────────────────────────────────────────────────────────
// Model tests
// ──────────────────────────────────────────────────────────────────────

func TestEncodeDecodeResponse_Roundtrip(t *testing.T) {
	code := 201
	body := []byte(`{"status":"ok","id":"abc-123"}`)
	encoded := EncodeResponse(code, body)

	decodedCode, decodedBody := DecodeResponse(&encoded)
	if decodedCode != code {
		t.Errorf("expected code %d, got %d", code, decodedCode)
	}
	if string(decodedBody) != string(body) {
		t.Errorf("expected body %q, got %q", string(body), string(decodedBody))
	}
}

func TestDecodeResponse_Nil(t *testing.T) {
	code, body := DecodeResponse(nil)
	if code != 0 {
		t.Errorf("expected code 0 for nil, got %d", code)
	}
	if body != nil {
		t.Errorf("expected nil body for nil, got %v", body)
	}
}

func TestDecodeResponse_Empty(t *testing.T) {
	empty := ""
	code, body := DecodeResponse(&empty)
	if code != 0 {
		t.Errorf("expected code 0 for empty, got %d", code)
	}
	if body != nil {
		t.Errorf("expected nil body for empty, got %v", body)
	}
}

func TestRecord_TableName(t *testing.T) {
	rec := Record{}
	if rec.TableName() != "idempotency_records" {
		t.Errorf("expected table name 'idempotency_records', got %q", rec.TableName())
	}
}

// ──────────────────────────────────────────────────────────────────────
// Context helpers
// ──────────────────────────────────────────────────────────────────────

func TestGetIdempotencyKey_NotSet(t *testing.T) {
	ctx := context.Background()
	key, ok := GetIdempotencyKey(ctx)
	if ok {
		t.Error("expected ok=false when key is not set")
	}
	if key != "" {
		t.Errorf("expected empty key, got %q", key)
	}
}

func TestGetIdempotencyKey_Roundtrip(t *testing.T) {
	ctx := context.Background()
	original := "550e8400-e29b-41d4-a716-446655440000"
	ctx = WithIdempotencyKey(ctx, original)

	key, ok := GetIdempotencyKey(ctx)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if key != original {
		t.Errorf("expected %q, got %q", original, key)
	}
}
