// Package idempotency implements a distributed idempotency filter backed by
// an atomic key-value store (e.g., Redis).
//
// Problem
//
// A payment gateway retries on network failure. Without a guard, the same
// charge request reaches the processor N times — one per retry.
//
// Solution: a two-phase state machine per idempotency key:
//
//	STARTED  → COMPLETED  (happy path)
//	STARTED  → deleted    (transient failure — caller may retry)
//
// The SETNX ("set if not exists") operation is the atomic gate. Across an
// entire distributed fleet, exactly one instance wins the race to set the key.
// Every other instance sees the key already present and backs off immediately.
package idempotency

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// State is the lifecycle stage of a payment operation.
type State string

const (
	StateStarted   State = "STARTED"
	StateCompleted State = "COMPLETED"
)

var (
	// ErrDuplicate is returned while the key is in STARTED state —
	// another process is actively handling the same request.
	ErrDuplicate = errors.New("idempotency: duplicate request in-flight")
	// ErrAlreadyCompleted is returned when the operation succeeded in a prior
	// attempt. The caller should return the previously computed response.
	ErrAlreadyCompleted = errors.New("idempotency: operation already completed")
)

// StateStore is the minimal atomic store interface required by Filter.
// Keeping it narrow makes the production Redis adapter and the unit-test
// in-memory adapter equally simple to implement and verify.
type StateStore interface {
	// SetNX sets key=value with TTL only when key does not yet exist.
	// Returns (true, nil) when the key was inserted (this caller wins the race).
	// Returns (false, nil) when the key already existed (this caller loses).
	SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Del(ctx context.Context, key string) error
}

// Filter enforces exactly-once processing for idempotent payment operations.
type Filter struct {
	store StateStore
	ttl   time.Duration
}

// NewFilter creates a Filter that retains completed keys for ttl duration.
// A 24-hour TTL is typical for payment deduplication windows.
func NewFilter(store StateStore, ttl time.Duration) *Filter {
	return &Filter{store: store, ttl: ttl}
}

// Acquire atomically claims the idempotency key for the caller.
//
// Exactly one caller wins per key. Concurrent callers in STARTED state receive
// ErrDuplicate. Callers arriving after COMPLETED receive ErrAlreadyCompleted.
//
// On success the caller MUST call Complete (happy path) or Release (error path).
func (f *Filter) Acquire(ctx context.Context, key string) error {
	won, err := f.store.SetNX(ctx, key, string(StateStarted), f.ttl)
	if err != nil {
		return fmt.Errorf("idempotency: acquire: %w", err)
	}
	if won {
		return nil
	}

	current, err := f.store.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("idempotency: read state: %w", err)
	}
	if State(current) == StateCompleted {
		return ErrAlreadyCompleted
	}
	return ErrDuplicate
}

// Complete transitions the key to COMPLETED, signalling successful processing.
// The key is retained for ttl so future duplicate requests can be detected.
func (f *Filter) Complete(ctx context.Context, key string) error {
	return f.store.Set(ctx, key, string(StateCompleted), f.ttl)
}

// Release deletes the key, making the operation retryable.
// Call this when processing fails transiently so the client may retry safely.
func (f *Filter) Release(ctx context.Context, key string) error {
	return f.store.Del(ctx, key)
}
