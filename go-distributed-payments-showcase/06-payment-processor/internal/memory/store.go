// Package memory provides in-memory implementations of the payment-processor outbound ports.
package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	payment "github.com/examples/go-distributed-payments-showcase/06-payment-processor/internal"
)

// ── IdempotencyStore ──────────────────────────────────────────────────────────

type idemEntry struct {
	value  string
	expiry time.Time
}

type IdempotencyStore struct {
	mu   sync.Mutex
	data map[string]idemEntry
}

func NewIdempotencyStore() *IdempotencyStore {
	return &IdempotencyStore{data: make(map[string]idemEntry)}
}

func (s *IdempotencyStore) SetNX(_ context.Context, key, value string, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.data[key]; ok && (e.expiry.IsZero() || time.Now().Before(e.expiry)) {
		return false, nil
	}
	s.data[key] = idemEntry{value: value, expiry: expiry(ttl)}
	return true, nil
}

func (s *IdempotencyStore) Get(_ context.Context, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data[key]
	if !ok || (!e.expiry.IsZero() && time.Now().After(e.expiry)) {
		return "", errors.New("key not found")
	}
	return e.value, nil
}

func (s *IdempotencyStore) Set(_ context.Context, key, value string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = idemEntry{value: value, expiry: expiry(ttl)}
	return nil
}

func (s *IdempotencyStore) Del(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

// ── AccountRepo ───────────────────────────────────────────────────────────────

type acct struct{ balance, version int64 }

type AccountRepo struct {
	mu             sync.Mutex
	accounts       map[string]acct
	initialBalance int64
}

func NewAccountRepo() *AccountRepo {
	return &AccountRepo{accounts: make(map[string]acct), initialBalance: 1_000_000}
}

func (r *AccountRepo) GetAccount(_ context.Context, id string) (int64, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.accounts[id]
	if !ok {
		a = acct{balance: r.initialBalance}
		r.accounts[id] = a
	}
	return a.balance, a.version, nil
}

func (r *AccountRepo) UpdateAccount(_ context.Context, id string, newBal, expectedVer int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.accounts[id]
	if !ok {
		return payment.ErrAccountNotFound
	}
	if a.version != expectedVer {
		return payment.ErrVersionConflict
	}
	r.accounts[id] = acct{balance: newBal, version: a.version + 1}
	return nil
}

// ── Publisher ─────────────────────────────────────────────────────────────────

type Publisher struct {
	mu    sync.Mutex
	count int
}

func NewPublisher() *Publisher { return &Publisher{} }

func (p *Publisher) Publish(_ context.Context, _, _, _ string, _ int64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.count++
	return nil
}

func expiry(ttl time.Duration) time.Time {
	if ttl <= 0 {
		return time.Time{}
	}
	return time.Now().Add(ttl)
}
