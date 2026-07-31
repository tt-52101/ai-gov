package idempotency

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// ErrIdempotencyConflict is returned by Claim when a write request arrives
// with an idempotency key that already exists but carries a different
// request fingerprint (RequestHash). This is a hard rejection: the client
// must not retry with the same key for a different payload.
//
// HTTP mapping: 409 Conflict, error code: IDEMPOTENCY_CONFLICT (PRD §6.1).
var ErrIdempotencyConflict = errors.New("idempotency conflict: different request payload for same key")

// ErrIdempotencyReplay is returned by Claim when a write request arrives
// with the same idempotency key and the same request fingerprint as a
// previously completed operation. The caller should return the existing
// result (accessible via the *Record parameter) rather than executing
// the operation again.
//
// HTTP mapping: 200 OK, error code: IDEMPOTENCY_REPLAY (PRD §6.1).
var ErrIdempotencyReplay = errors.New("idempotency replay: duplicate request, use original result")

// ErrIdempotencyInProgress is returned when a key is claimed but the
// operation has not yet completed or failed. The caller should retry
// after a short backoff.
var ErrIdempotencyInProgress = errors.New("idempotency key in progress: operation not yet completed")

// errReclaimRace is an internal sentinel used in reclaim to signal that
// another goroutine won the reclaim race (old record was already deleted
// by a concurrent reclaim).
var errReclaimRace = errors.New("reclaim race: record already reclaimed by concurrent caller")

// DefaultRetentionWindow is the minimum lifetime for an idempotency record.
// After this period, the record may be cleaned up by CleanExpired.
// Set to 24 hours per PRD §8.7.
const DefaultRetentionWindow = 24 * time.Hour

// Claim atomically reserves an idempotency key for a fund write operation.
//
// Strategy (INSERT-first, IRON RULE 1):
//  1. INSERT the record into idempotency_records.
//  2. If INSERT succeeds: the key is claimed. Returns (true, nil, nil).
//  3. If UNIQUE constraint violation (gorm.ErrDuplicatedKey):
//     a. Fetch the existing record.
//     b. If the existing record is expired: attempt atomic DELETE+INSERT
//     reclaim in a transaction. If reclaim succeeds, returns (true, nil, nil).
//     c. If the existing record has the same RequestHash:
//     - If status is StatusSucceeded: returns (false, existing, ErrIdempotencyReplay).
//     The caller should return the stored response.
//     - If status is StatusStarted: returns (false, nil, ErrIdempotencyInProgress).
//     - If status is StatusFailed: returns (false, existing, ErrIdempotencyConflict).
//     d. If the existing record has a different RequestHash:
//     returns (false, nil, ErrIdempotencyConflict).
//
// NEVER use SELECT-then-INSERT: that path has a TOCTOU race between
// checking for existence and inserting. The INSERT-first approach lets
// the database UNIQUE constraint enforce atomicity (PRD §8.7, Stripe semantics).
//
// The caller is responsible for setting ID, Scope, ActorID, IdempotencyKey,
// RequestHash, Status (should be StatusStarted), and ExpiresAt on rec
// before calling Claim. ExpiresAt should be at least DefaultRetentionWindow
// in the future.
//
// After a successful claim (claimed=true), the caller must eventually call
// either Complete or Fail to release the key from the StatusStarted state.
func Claim(ctx context.Context, db *gorm.DB, rec *Record) (claimed bool, existing *Record, err error) {
	// Step 1: INSERT — the database UNIQUE constraint is the atomicity mechanism.
	if err := InsertRecord(db, rec); err == nil {
		// Insert succeeded: key is claimed, operation may proceed.
		slog.InfoContext(ctx, "idempotency_key_claimed",
			"idempotency_key", rec.IdempotencyKey,
			"scope", rec.Scope,
			"actor_id", rec.ActorID,
			"expires_at", rec.ExpiresAt,
		)
		return true, nil, nil
	} else if !isDuplicateKeyError(err) {
		// A non-duplicate error (e.g., connection failure) is fatal.
		return false, nil, fmt.Errorf("idempotency insert: %w", err)
	}

	// Step 2: UNIQUE constraint violated — a record already exists.
	// Fetch the existing record to determine replay vs. conflict.
	existing, err = GetRecord(db, rec.Scope, rec.ActorID, rec.IdempotencyKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Should not happen: the UNIQUE constraint guarantees a row exists,
			// but if it was deleted between our INSERT and SELECT, retry once.
			slog.WarnContext(ctx, "idempotency_key_vanished",
				"idempotency_key", rec.IdempotencyKey,
				"scope", rec.Scope,
			)
			return Claim(ctx, db, rec)
		}
		return false, nil, fmt.Errorf("idempotency lookup: %w", err)
	}

	// Step 3: Check if the existing record has expired.
	// An expired record can be atomically reclaimed.
	if time.Now().After(existing.ExpiresAt) {
		claimed, err := reclaim(ctx, db, existing, rec)
		if err != nil {
			return false, nil, err
		}
		if claimed {
			return true, nil, nil
		}
		// Reclaim failed (race): re-fetch and fall through to hash comparison.
		existing, err = GetRecord(db, rec.Scope, rec.ActorID, rec.IdempotencyKey)
		if err != nil {
			return false, nil, fmt.Errorf("idempotency re-lookup after reclaim race: %w", err)
		}
	}

	// Step 4: Same fingerprint → replay.
	if existing.RequestHash == rec.RequestHash {
		return handleReplay(ctx, existing)
	}

	// Step 5: Different fingerprint → conflict.
	slog.WarnContext(ctx, "idempotency_key_conflict",
		"idempotency_key", rec.IdempotencyKey,
		"scope", rec.Scope,
		"actor_id", rec.ActorID,
		"existing_hash", existing.RequestHash,
		"new_hash", rec.RequestHash,
	)
	return false, nil, ErrIdempotencyConflict
}

// handleReplay determines the outcome when the same request fingerprint
// is seen for an existing idempotency key.
func handleReplay(ctx context.Context, existing *Record) (bool, *Record, error) {
	switch existing.Status {
	case StatusSucceeded:
		slog.InfoContext(ctx, "idempotency_replay_success",
			"idempotency_key", existing.IdempotencyKey,
			"scope", existing.Scope,
			"resource_ref", ptrVal(existing.ResourceRef),
		)
		return false, existing, ErrIdempotencyReplay

	case StatusFailed:
		slog.WarnContext(ctx, "idempotency_replay_failed",
			"idempotency_key", existing.IdempotencyKey,
			"scope", existing.Scope,
		)
		return false, existing, ErrIdempotencyConflict

	case StatusStarted:
		slog.WarnContext(ctx, "idempotency_key_in_progress",
			"idempotency_key", existing.IdempotencyKey,
			"scope", existing.Scope,
		)
		return false, nil, ErrIdempotencyInProgress

	default:
		return false, nil, fmt.Errorf("idempotency: unknown status %q", existing.Status)
	}
}

// reclaim atomically deletes an expired record and inserts a new one
// in a single database transaction. This prevents race conditions
// where two concurrent goroutines both see an expired record and both
// try to reclaim it: only one INSERT will succeed.
func reclaim(ctx context.Context, db *gorm.DB, old *Record, new *Record) (bool, error) {
	err := db.Transaction(func(tx *gorm.DB) error {
		// Delete only if the record still has the same ID and is still expired.
		// This is a guard against a concurrent reclaim having already replaced
		// this record with a new one that has a different ID.
		delRes := tx.Where("id = ? AND expires_at < ?", old.ID, time.Now()).
			Delete(&Record{})
		if delRes.Error != nil {
			return fmt.Errorf("reclaim delete: %w", delRes.Error)
		}
		if delRes.RowsAffected == 0 {
			// Another goroutine already reclaimed this key.
			return errReclaimRace
		}

		// Insert the new record.
		if err := tx.Create(new).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		if errors.Is(err, errReclaimRace) || isDuplicateKeyError(err) {
			// Concurrent reclaim lost the race.
			slog.InfoContext(ctx, "idempotency_reclaim_race_lost",
				"idempotency_key", new.IdempotencyKey,
				"scope", new.Scope,
			)
			return false, nil
		}
		return false, fmt.Errorf("idempotency reclaim transaction: %w", err)
	}

	slog.InfoContext(ctx, "idempotency_key_reclaimed",
		"idempotency_key", new.IdempotencyKey,
		"scope", new.Scope,
		"actor_id", new.ActorID,
		"old_record_id", old.ID,
	)
	return true, nil
}

// Complete marks an idempotency record as StatusSucceeded and stores the
// response payload for future replays.
//
// Parameters:
//   - db: the GORM database handle (can be a transaction from the caller).
//   - id: the primary key of the record (Record.ID, a UUID v4 string).
//   - code: the HTTP status code of the successful response.
//   - body: the response body bytes.
//   - ref: a reference to the created resource (e.g., transaction ID).
//
// Complete loads the record by ID, verifies it is in StatusStarted state,
// then updates it to StatusSucceeded with the response and reference.
//
// Returns an error if the record is not found or is not in StatusStarted state.
func Complete(ctx context.Context, db *gorm.DB, id string, code int, body []byte, ref string) error {
	var rec Record
	if err := db.First(&rec, "id = ?", id).Error; err != nil {
		return fmt.Errorf("idempotency complete: record %s: %w", id, err)
	}

	if rec.Status != StatusStarted {
		return fmt.Errorf("idempotency complete: record %s has status %q, expected %q",
			id, rec.Status, StatusStarted)
	}

	encoded := EncodeResponse(code, body)
	rec.Status = StatusSucceeded
	rec.ResponseJSON = &encoded
	rec.ResourceRef = &ref

	if err := UpdateRecord(db, &rec); err != nil {
		return fmt.Errorf("idempotency complete: save record %s: %w", id, err)
	}

	slog.InfoContext(ctx, "idempotency_operation_completed",
		"idempotency_key", rec.IdempotencyKey,
		"scope", rec.Scope,
		"actor_id", rec.ActorID,
		"response_code", code,
		"resource_ref", ref,
	)
	return nil
}

// Fail marks an idempotency record as StatusFailed. This prevents the
// same key from being retried with a different payload.
//
// After Fail is called, subsequent Claim calls with the same key+fingerprint
// will return ErrIdempotencyConflict (not ErrIdempotencyReplay), since a
// failed operation should not be silently replayed.
//
// Parameters:
//   - db: the GORM database handle.
//   - id: the primary key of the record (Record.ID).
//   - errMsg: a human-readable error message describing the failure.
//
// Returns an error if the record is not found or is not in StatusStarted state.
func Fail(ctx context.Context, db *gorm.DB, id string, errMsg string) error {
	var rec Record
	if err := db.First(&rec, "id = ?", id).Error; err != nil {
		return fmt.Errorf("idempotency fail: record %s: %w", id, err)
	}

	if rec.Status != StatusStarted {
		return fmt.Errorf("idempotency fail: record %s has status %q, expected %q",
			id, rec.Status, StatusStarted)
	}

	rec.Status = StatusFailed
	// Store the error message in ResponseJSON so it's available for diagnostics.
	encoded := fmt.Sprintf(`{"error":%q}`, errMsg)
	rec.ResponseJSON = &encoded

	if err := UpdateRecord(db, &rec); err != nil {
		return fmt.Errorf("idempotency fail: save record %s: %w", id, err)
	}

	slog.ErrorContext(ctx, "idempotency_operation_failed",
		"idempotency_key", rec.IdempotencyKey,
		"scope", rec.Scope,
		"actor_id", rec.ActorID,
		"error", errMsg,
	)
	return nil
}

// ptrVal returns the dereferenced string, or "<nil>" for nil pointers.
// Used only for structured logging.
func ptrVal(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
