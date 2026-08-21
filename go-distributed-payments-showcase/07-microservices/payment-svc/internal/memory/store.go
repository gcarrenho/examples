// Package memory provides in-memory implementations of payment-svc secondary ports.
package memory

import (
	"context"
	"sync"

	payment "github.com/examples/go-distributed-payments-showcase/07-microservices/payment-svc/internal"
)

// ── IdempotencyStore ──────────────────────────────────────────────────────────

type IdempotencyStore struct {
	mu   sync.Mutex
	data map[string]string
}

func NewIdempotencyStore() *IdempotencyStore {
	return &IdempotencyStore{data: make(map[string]string)}
}

func (s *IdempotencyStore) Acquire(_ context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[key]; ok {
		return false, nil
	}
	s.data[key] = "STARTED"
	return true, nil
}

func (s *IdempotencyStore) Complete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = "COMPLETED"
	return nil
}

func (s *IdempotencyStore) IsCompleted(_ context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data[key] == "COMPLETED", nil
}

func (s *IdempotencyStore) Release(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

// ── AccountRepo ───────────────────────────────────────────────────────────────

type acct struct{ balance, version int64 }

type AccountRepo struct {
	mu   sync.Mutex
	data map[string]*acct
}

func NewAccountRepo(seed map[string]int64) *AccountRepo {
	data := make(map[string]*acct, len(seed))
	for id, bal := range seed {
		data[id] = &acct{balance: bal}
	}
	return &AccountRepo{data: data}
}

func (r *AccountRepo) GetAccount(_ context.Context, id string) (int64, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.data[id]
	if !ok {
		return 0, 0, payment.ErrAccountNotFound
	}
	return a.balance, a.version, nil
}

func (r *AccountRepo) UpdateAccount(_ context.Context, id string, newBal, expVer int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.data[id]
	if !ok {
		return payment.ErrAccountNotFound
	}
	if a.version != expVer {
		return payment.ErrVersionConflict
	}
	a.balance = newBal
	a.version++
	return nil
}

// ── ReservationStore ──────────────────────────────────────────────────────────

type res struct {
	accountID   string
	amountCents int64
}

type ReservationStore struct {
	mu   sync.Mutex
	data map[string]*res
}

func NewReservationStore() *ReservationStore {
	return &ReservationStore{data: make(map[string]*res)}
}

func (s *ReservationStore) Create(_ context.Context, id, accountID string, amountCents int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[id] = &res{accountID: accountID, amountCents: amountCents}
	return nil
}

func (s *ReservationStore) Get(_ context.Context, id string) (string, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.data[id]
	if !ok {
		return "", 0, payment.ErrReservationNotFound
	}
	return r.accountID, r.amountCents, nil
}
