// Package occ demonstrates two strategies for safe concurrent balance updates.
//
// Optimistic Concurrency Control (OCC) — "be optimistic: let everyone try"
//
//	Read:     fetch account row, including a monotonic version counter.
//	Modify:   compute new balance in memory.
//	Write:    UPDATE WHERE version = expectedVersion.
//	Conflict: if UPDATE affected 0 rows, another writer committed first → retry.
//
// Pessimistic Concurrency Control (PCC) — "be pessimistic: lock first"
//
//	Lock:          SELECT ... FOR UPDATE acquires a row-level exclusive lock.
//	Modify+Write:  safe to read-then-write; no other writer can enter the row.
//	Unlock:        released automatically on COMMIT or ROLLBACK.
//
// When to prefer OCC:
//   - Low contention: most operations succeed on the first attempt.
//   - Read-heavy workloads where long lock waits would stall readers.
//
// When to prefer PCC:
//   - High contention: many concurrent writers to the same row.
//   - Operations where retry is expensive or has observable side effects.
package occ

import (
	"context"
	"errors"
	"fmt"
)

// ErrVersionConflict is returned by UpdateOCC when another writer committed
// a newer version between our read and write. The caller should retry.
var ErrVersionConflict = errors.New("occ: version conflict")

// Account is a payment account with an optimistic version counter.
type Account struct {
	ID      string
	Balance int64 // in cents; must never go negative
	Version int64 // incremented on every committed write
}

// AccountStore is the minimal persistence interface for both strategies.
// In production this wraps pgx transaction methods.
type AccountStore interface {
	GetByID(ctx context.Context, id string) (*Account, error)
	// UpdateOCC writes newBalance only when the stored version equals expectedVersion.
	// Returns ErrVersionConflict on mismatch — another writer beat us.
	UpdateOCC(ctx context.Context, id string, newBalance, expectedVersion int64) error
	// UpdatePCC applies delta inside a serialised lock (SELECT FOR UPDATE equivalent).
	// The store or database is responsible for acquiring the row-level lock.
	UpdatePCC(ctx context.Context, id string, delta int64) error
}

// Ledger applies financial operations using the configured concurrency strategy.
type Ledger struct {
	store      AccountStore
	maxRetries int
}

// NewLedger creates a Ledger. maxRetries bounds the OCC retry loop.
func NewLedger(store AccountStore, maxRetries int) *Ledger {
	return &Ledger{store: store, maxRetries: maxRetries}
}

// TransferOCC applies delta to an account balance using Optimistic Concurrency Control.
//
// The read-modify-write cycle is retried on version conflicts. Under low contention
// one iteration suffices; under high contention retry overhead grows with goroutine count.
func (l *Ledger) TransferOCC(ctx context.Context, accountID string, delta int64) error {
	for attempt := range l.maxRetries {
		acc, err := l.store.GetByID(ctx, accountID)
		if err != nil {
			return fmt.Errorf("occ read [attempt %d]: %w", attempt+1, err)
		}

		newBalance := acc.Balance + delta
		if newBalance < 0 {
			// Domain error: do NOT retry, fail immediately.
			return fmt.Errorf("occ: insufficient funds (balance=%d cents, delta=%d)", acc.Balance, delta)
		}

		err = l.store.UpdateOCC(ctx, accountID, newBalance, acc.Version)
		if errors.Is(err, ErrVersionConflict) {
			continue // lost the race — re-read the latest state and retry
		}
		return err // nil on success, or a hard infrastructure error
	}
	return fmt.Errorf("occ: %s exceeded %d retries", accountID, l.maxRetries)
}

// TransferPCC applies delta to an account balance using Pessimistic Concurrency Control.
//
// The store serialises access via a row-level lock; no retry logic is necessary.
// Writers block until the lock is available rather than failing fast.
func (l *Ledger) TransferPCC(ctx context.Context, accountID string, delta int64) error {
	if err := l.store.UpdatePCC(ctx, accountID, delta); err != nil {
		return fmt.Errorf("pcc: %w", err)
	}
	return nil
}
