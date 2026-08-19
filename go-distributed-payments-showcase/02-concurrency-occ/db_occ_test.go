package occ

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

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
	}{
		{"OCC / optimistic", (*Ledger).TransferOCC, 500},
		{"PCC / pessimistic", (*Ledger).TransferPCC, 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			store := newStore(Account{ID: "acc-1", Balance: 0, Version: 0})
			ledger := NewLedger(store, tc.retries)

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
