// Package idempotency provides idempotency-key based deduplication
// for mutation operations. Guarantees at-most-once semantics for
// balance mutations, allocation requests, and administrative actions.
package idempotency

import (
	"encoding/json"
	"time"
)

// 状态常量定义幂等记录的生命周期。状态机：started → succeeded | failed。一旦到达终态，状态永不改变。
const (
	// StatusStarted marks a newly claimed idempotency key whose operation
	// has not yet completed. The caller must eventually call Complete or
	// Fail to transition out of this state.
	StatusStarted = "started"

	// StatusSucceeded marks a completed idempotency record whose response
	// is stored in ResponseJSON. Replays of the same key+hash return the
	// stored response without re-executing the operation.
	StatusSucceeded = "succeeded"

	// StatusFailed marks an idempotency record whose operation failed.
	// A failed record prevents the caller from retrying with the same key.
	StatusFailed = "failed"
)

// Record represents a row in the idempotency_records table.
// It stores the state of an idempotency-key-guarded write operation.
//
// Fields mapped from DDL (schema/ai-gov-fusion-v3.2.sql table 29):
//
//	id              TEXT PRIMARY KEY          — UUID v4
//	scope           TEXT NOT NULL             — allocate / liquidate / compensate
//	actor_id        TEXT NOT NULL             — user or system actor
//	idempotency_key TEXT NOT NULL             — client-supplied UUID v4, ≤255 chars
//	request_hash    TEXT NOT NULL             — SHA-256 of the request body
//	status          TEXT NOT NULL DEFAULT 'started'
//	response_json   JSONB                     — first successful response snapshot
//	resource_ref    TEXT                      — e.g. transaction_id
//	expires_at      TIMESTAMPTZ NOT NULL      — ≥24h window (PRD §8.7)
//
// UNIQUE(scope, actor_id, idempotency_key) — enforced at DB level by
// uniqueIndex:idx_idempotency_scope_actor_key on fields Scope, ActorID,
// and IdempotencyKey. GORM's AutoMigrate creates this composite index.
type Record struct {
	// ID is the primary key, generated as a UUID v4 by the caller before
	// calling Claim. Stored as TEXT per the DDL.
	ID string `json:"id" gorm:"primaryKey"`

	// Scope identifies the operation domain (e.g. "allocate", "liquidate").
	// Part of the unique constraint: (scope, actor_id, idempotency_key).
	Scope string `json:"scope" gorm:"uniqueIndex:idx_idempotency_scope_actor_key,priority:1;not null"`

	// ActorID identifies the user or system actor performing the operation.
	// Part of the unique constraint: (scope, actor_id, idempotency_key).
	ActorID string `json:"actor_id" gorm:"uniqueIndex:idx_idempotency_scope_actor_key,priority:2;not null;column:actor_id"`

	// IdempotencyKey is the client-supplied UUID v4 key (max 255 chars).
	// Part of the unique constraint: (scope, actor_id, idempotency_key).
	IdempotencyKey string `json:"idempotency_key" gorm:"uniqueIndex:idx_idempotency_scope_actor_key,priority:3;not null;column:idempotency_key"`

	// RequestHash is the SHA-256 hex digest of the request payload.
	// Used to detect replay (same hash) vs. conflict (different hash).
	RequestHash string `json:"request_hash" gorm:"not null;column:request_hash"`

	// Status tracks the operation lifecycle: started → succeeded | failed.
	// See StatusStarted, StatusSucceeded, StatusFailed constants.
	Status string `json:"status" gorm:"not null;default:started"`

	// ResponseJSON stores the first successful response as a JSONB snapshot.
	// On replay, this value is returned without re-executing the operation.
	// Nil when the operation has not yet completed.
	ResponseJSON *string `json:"response_json,omitempty" gorm:"type:jsonb;column:response_json"`

	// ResourceRef stores a reference to the created resource, such as a
	// transaction ID or allocation ID. Nil when the operation has not
	// yet produced a resource.
	ResourceRef *string `json:"resource_ref,omitempty" gorm:"column:resource_ref"`

	// ExpiresAt is the time after which the record may be cleaned up by
	// the background CleanExpired job. The retention window is at least
	// 24 hours per PRD §8.7.
	ExpiresAt time.Time `json:"expires_at" gorm:"not null;index;column:expires_at"`

	// CreatedAt is set automatically on insert. Implements GORM's
	// auto-create timestamp convention.
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`

	// UpdatedAt is set automatically on insert and update. Implements
	// GORM's auto-update timestamp convention.
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at"`
}

// TableName overrides the default GORM table name to match the DDL.
func (Record) TableName() string {
	return "idempotency_records"
}

// ResponsePayload is the structured content stored in ResponseJSON.
// It captures the HTTP status code and response body of a completed
// idempotent operation, enabling exact replay.
type ResponsePayload struct {
	// Code is the HTTP status code returned by the original operation.
	Code int `json:"code"`

	// Body is the response body returned by the original operation.
	Body json.RawMessage `json:"body"`
}

// EncodeResponse serializes a response code and body into a JSON string
// suitable for storage in Record.ResponseJSON.
func EncodeResponse(code int, body []byte) string {
	payload, _ := json.Marshal(ResponsePayload{Code: code, Body: body})
	return string(payload)
}

// DecodeResponse extracts the HTTP status code and body from a stored
// ResponseJSON value. Returns zero values if the stored JSON is invalid
// or empty.
func DecodeResponse(raw *string) (code int, body []byte) {
	if raw == nil || *raw == "" {
		return 0, nil
	}
	var p ResponsePayload
	if err := json.Unmarshal([]byte(*raw), &p); err != nil {
		return 0, nil
	}
	return p.Code, p.Body
}
