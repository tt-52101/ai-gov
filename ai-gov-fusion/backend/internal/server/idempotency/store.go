package idempotency

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	"gorm.io/gorm"
)

// isDuplicateKeyError checks whether err is a duplicate-key / unique-constraint
// violation across both SQLite and PostgreSQL drivers.
//
// PostgreSQL: gorm.ErrDuplicatedKey (wraps pq error code 23505).
// SQLite:    the sqlite3 driver returns an error containing the string
//
//	"UNIQUE constraint failed".
//
// This helper avoids coupling the core Claim logic to a single driver's
// error type. IRON RULE 1: only this function and InsertRecord interact
// with the database error that signals a uniqueness conflict.
func isDuplicateKeyError(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	// SQLite driver (mattn/go-sqlite3) returns the SQLite error message
	// directly; GORM does not translate it into ErrDuplicatedKey on all
	// versions.
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return true
	}
	return false
}

// InsertRecord performs an INSERT into idempotency_records.
// It relies on the UNIQUE(scope, actor_id, idempotency_key) constraint
// to guarantee atomicity at the database level. If the constraint is
// violated, gorm.ErrDuplicatedKey is returned — the caller must handle
// that as either a replay or a conflict.
//
// 唯一的插入路径——禁止 SELECT 后 INSERT 模式。
// in the codebase, which would introduce a race condition (IRON RULE 1).
//
// The caller is responsible for setting the following fields before
// calling InsertRecord: ID, Scope, ActorID, IdempotencyKey, RequestHash,
// Status (should be StatusStarted), and ExpiresAt.
func InsertRecord(db *gorm.DB, rec *Record) error {
	result := db.Create(rec)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// GetRecord retrieves an idempotency record by its composite business key
// (scope, actor_id, idempotency_key). Returns gorm.ErrRecordNotFound if
// no record exists.
//
// This is safe to call after InsertRecord fails with a duplicate-key error,
// since the UNIQUE constraint guarantees that exactly one record exists
// for the given key at that point.
func GetRecord(db *gorm.DB, scope, actorID, key string) (*Record, error) {
	var rec Record
	result := db.Where("scope = ? AND actor_id = ? AND idempotency_key = ?",
		scope, actorID, key).First(&rec)
	if result.Error != nil {
		return nil, result.Error
	}
	return &rec, nil
}

// UpdateRecord updates an existing idempotency record in-place.
// Typically called by Complete or Fail to transition the record from
// StatusStarted to StatusSucceeded or StatusFailed.
//
// The caller must ensure that the record's ID is set and that the
// fields to update are populated on the struct. GORM will generate
// an UPDATE statement for the specific columns that have non-zero values.
func UpdateRecord(db *gorm.DB, rec *Record) error {
	result := db.Save(rec)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// CleanExpired deletes all idempotency records whose expires_at timestamp
// is before the current time. This is designed to be called periodically
// by a background goroutine (e.g., every 60 seconds) to enforce the
// retention window defined in PRD §8.7 (minimum 24 hours).
//
// Returns the number of deleted rows and any error that occurred.
//
// Uses a raw DELETE statement to avoid loading expired records into
// memory. The expires_at column has a B-tree index (see DDL), so the
// WHERE clause is efficient.
func CleanExpired(db *gorm.DB) (int64, error) {
	result := db.Exec("DELETE FROM idempotency_records WHERE expires_at < ?", time.Now())
	if result.Error != nil {
		slog.Error("idempotency_cleanup_failed",
			"error", result.Error.Error(),
		)
		return 0, result.Error
	}
	deleted := result.RowsAffected
	if deleted > 0 {
		slog.Info("idempotency_cleanup",
			"deleted_count", deleted,
		)
	}
	return deleted, nil
}
