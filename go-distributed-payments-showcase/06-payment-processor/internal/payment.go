// Package payment is the implementation package for the paymentprocessor component.
// Domain types, service logic, and private ports all live here — no separate layers.
// The root package (paymentprocessor) exposes the public API.
package payment

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ── Domain types ──────────────────────────────────────────────────────────────

type PaymentState string

const (
	PaymentStarted   PaymentState = "STARTED"
	PaymentCompleted PaymentState = "COMPLETED"
)

type EventType string

const (
	EventCharge EventType = "CHARGE"
	EventRefund EventType = "REFUND"
)

var (
	ErrDuplicateRequest  = errors.New("payment: duplicate request in-flight")
	ErrAlreadyProcessed  = errors.New("payment: request already completed")
	ErrInsufficientFunds = errors.New("payment: insufficient funds")
	ErrAccountNotFound   = errors.New("payment: account not found")
	ErrVersionConflict   = errors.New("payment: OCC version conflict")
)

// ── Private ports ─────────────────────────────────────────────────────────────

type idempotencyStore interface {
	SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Del(ctx context.Context, key string) error
}

type accountRepository interface {
	GetAccount(ctx context.Context, accountID string) (balanceCents, version int64, err error)
	UpdateAccount(ctx context.Context, accountID string, newBalanceCents, expectedVersion int64) error
}

type eventPublisher interface {
	Publish(ctx context.Context, eventType, accountID, paymentKey string, amountCents int64) error
}

// ── Service ───────────────────────────────────────────────────────────────────

const (
	maxOCCRetries  = 10
	idempotencyTTL = 24 * time.Hour
)

// PaymentService orchestrates idempotency (Case 01), OCC balance update (Case 02),
// and event publishing (Case 03) in a single flow.
type PaymentService struct {
	idempotency idempotencyStore
	accounts    accountRepository
	publisher   eventPublisher
}

func New(store idempotencyStore, repo accountRepository, pub eventPublisher) *PaymentService {
	return &PaymentService{idempotency: store, accounts: repo, publisher: pub}
}

func (s *PaymentService) Charge(ctx context.Context, key, accountID string, amountCents int64) error {
	return s.process(ctx, key, accountID, amountCents, string(EventCharge), func(b int64) int64 { return b - amountCents })
}

func (s *PaymentService) Refund(ctx context.Context, key, accountID string, amountCents int64) error {
	return s.process(ctx, key, accountID, amountCents, string(EventRefund), func(b int64) int64 { return b + amountCents })
}

func (s *PaymentService) process(ctx context.Context, key, accountID string, amountCents int64, eventType string, apply func(int64) int64) error {
	won, err := s.idempotency.SetNX(ctx, key, string(PaymentStarted), idempotencyTTL)
	if err != nil {
		return fmt.Errorf("idempotency gate: %w", err)
	}
	if !won {
		state, err := s.idempotency.Get(ctx, key)
		if err != nil {
			return fmt.Errorf("idempotency read: %w", err)
		}
		if PaymentState(state) == PaymentCompleted {
			return ErrAlreadyProcessed
		}
		return ErrDuplicateRequest
	}

	if err := s.updateBalance(ctx, accountID, amountCents, apply); err != nil {
		_ = s.idempotency.Del(ctx, key)
		return err
	}

	_ = s.publisher.Publish(ctx, eventType, accountID, key, amountCents)
	_ = s.idempotency.Set(ctx, key, string(PaymentCompleted), idempotencyTTL)
	return nil
}

func (s *PaymentService) updateBalance(ctx context.Context, accountID string, amountCents int64, apply func(int64) int64) error {
	for range maxOCCRetries {
		balance, version, err := s.accounts.GetAccount(ctx, accountID)
		if err != nil {
			return fmt.Errorf("get account: %w", err)
		}
		newBalance := apply(balance)
		if newBalance < 0 {
			return fmt.Errorf("%w (balance=%d)", ErrInsufficientFunds, balance)
		}
		err = s.accounts.UpdateAccount(ctx, accountID, newBalance, version)
		if errors.Is(err, ErrVersionConflict) {
			continue
		}
		return err
	}
	return fmt.Errorf("OCC: exceeded %d retries for account %s", maxOCCRetries, accountID)
}
