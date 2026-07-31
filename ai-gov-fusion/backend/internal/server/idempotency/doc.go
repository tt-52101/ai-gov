// Package idempotency provides idempotency-key based deduplication
// for mutation operations. Guarantees at-most-once semantics for
// balance mutations, allocation requests, and administrative actions.
package idempotency
