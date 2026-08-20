package occ

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// conflictStore always returns ErrVersionConflict from UpdateOCC.
// Used to test retry limits and context cancellation without real contention.
type conflictStore struct{ acc Account }

func (s *conflictStore) GetByID(_ context.Context, _ string) (*Account, error) {
	a := s.acc
	return &a, nil
}
func (s *conflictStore) UpdateOCC(_ context.Context, _ string, _, _ int64) error {
	return ErrVersionConflict
}
func (s *conflictStore) UpdatePCC(_ context.Context, _ string, _ int64) error { return nil }

// inMemoryStore simulates a PostgreSQL accounts table.
//
// Its mutex IS the row-level lock for PCC; the version field IS the OCC guard.
// Concurrency behaviour is structurally identical to a real database —
// the implementation details (b-tree vs hash map) are irrelevant to the test.
type inMemoryStore struct {
	mu            sync.Mutex
	acc           Account
	conflictCount atomic.Int64
}

func newStore(initial Account) *inMemoryStore {
	return &inMemoryStore{acc: initial}
}

func (s *inMemoryStore) GetByID(_ context.Context, id string) (*Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.acc.ID != id {
		return nil, errors.New("account not found")
	}
	snapshot := s.acc // return a copy so the caller can't mutate shared state
	return &snapshot, nil
}

// UpdateOCC simulates "UPDATE accounts SET ... WHERE version = expectedVersion".
// The mutex ensures the check-and-write is atomic, just as a database transaction does.
func (s *inMemoryStore) UpdateOCC(_ context.Context, id string, newBalance, expectedVersion int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.acc.ID != id {
		return errors.New("account not found")
	}
	if s.acc.Version != expectedVersion {
		s.conflictCount.Add(1)
		return ErrVersionConflict
	}
	s.acc.Balance = newBalance
	s.acc.Version++
	return nil
}

// UpdatePCC simulates "SELECT FOR UPDATE + UPDATE" inside a single transaction.
// Holding the mutex for the full read-modify-write is equivalent to a row lock.
func (s *inMemoryStore) UpdatePCC(_ context.Context, id string, delta int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.acc.ID != id {
		return errors.New("account not found")
	}
	s.acc.Balance += delta
	s.acc.Version++
	return nil
}

// TestLedger_ConvergesUnderContention verifies that both OCC and PCC correctly
// serialise 100 concurrent credits onto the same account.
// Each strategy runs as a parallel subtest with its own isolated store.
func TestLedger_ConvergesUnderContention(t *testing.T) {
	t.Parallel()
	const (
		goroutines  = 100
		creditCents = 100
	)

	cases := []struct {
		name     string
		transfer func(*Ledger, context.Context, string, int64) error
		retries  int
		opts     []func(*Ledger)
	}{
		{name: "OCC / optimistic", transfer: (*Ledger).TransferOCC, retries: 500},
		{
			// Same logic as OCC; jitter spreads retries — expect fewer conflicts in the log.
			name:     "OCC / optimistic + jitter backoff",
			transfer: (*Ledger).TransferOCC,
			retries:  500,
			opts:     []func(*Ledger){WithBackoff(JitteredBackoff(5 * time.Microsecond))},
		},
		{name: "PCC / pessimistic", transfer: (*Ledger).TransferPCC, retries: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			store := newStore(Account{ID: "acc-1", Balance: 0, Version: 0})
			ledger := NewLedger(store, tc.retries, tc.opts...)

			var wg sync.WaitGroup
			for range goroutines {
				wg.Add(1)
				go func() {
					defer wg.Done()
					if err := tc.transfer(ledger, context.Background(), "acc-1", creditCents); err != nil {
						t.Errorf("%s: %v", tc.name, err)
					}
				}()
			}
			wg.Wait()

			want := int64(goroutines * creditCents)
			if got := store.acc.Balance; got != want {
				t.Errorf("balance = %d, want %d", got, want)
			}
			t.Logf("%s: %d writes committed, %d OCC conflicts",
				tc.name, store.acc.Version, store.conflictCount.Load())
		})
	}
}

// TestOCC_ContextCancelledDuringRetry verifies that a cancelled context
// aborts the retry loop immediately rather than burning through maxRetries.
func TestOCC_ContextCancelledDuringRetry(t *testing.T) {
	t.Parallel()
	store := &conflictStore{acc: Account{ID: "acc-ctx", Balance: 1000, Version: 0}}
	ledger := NewLedger(store, 10_000) // large budget; context should terminate it first

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel: context is already done on the first retry check

	err := ledger.TransferOCC(ctx, "acc-ctx", -100)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("want context.Canceled, got %v", err)
	}
}

// TestOCC_DomainErrorIsNotRetried asserts that business violations (overdraft)
// fail immediately without consuming any of the OCC retry budget.
func TestOCC_DomainErrorIsNotRetried(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		balance int64
		delta   int64
	}{
		{"exact overdraft", 100, -101},
		{"debit from empty account", 0, -1},
		{"large debit", 1000, -1001},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			store := newStore(Account{ID: "acc", Balance: tc.balance, Version: 0})
			ledger := NewLedger(store, 10)

			if err := ledger.TransferOCC(context.Background(), "acc", tc.delta); err == nil {
				t.Fatal("want overdraft error, got nil")
			}
			if got := store.conflictCount.Load(); got != 0 {
				t.Errorf("domain error triggered %d OCC retries; want 0", got)
			}
		})
	}
}
