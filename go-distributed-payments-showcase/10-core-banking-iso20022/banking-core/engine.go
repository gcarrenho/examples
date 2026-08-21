// Package banking is the core banking domain library.
// Zero external dependencies — pure Go, no infrastructure, no frameworks.
// Services import this module and bring their own infrastructure adapters.
package banking

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"
)

// ── Domain types ──────────────────────────────────────────────────────────────

type PaymentOrder struct {
	ID         string
	UETR       string        // Unique End-to-End Transaction Reference (UUID, SWIFT gpi mandatory)
	InstrID    string        // max 16 chars — assigned by sending bank
	EndToEndID string        // passed through unchanged by all intermediaries
	Status     PaymentStatus
	Amount     Money
	Debtor     Party
	Creditor   Party
	Rail       RailName
	CreatedAt  time.Time
	SettledAt  *time.Time
}

type PaymentStatus string

const (
	StatusInitiated PaymentStatus = "INIT"
	StatusPending   PaymentStatus = "PDNG"
	StatusSettled   PaymentStatus = "ACSC" // AcceptedSettlementCompleted
	StatusRejected  PaymentStatus = "RJCT"
)

type Money struct {
	AmountCents int64
	Currency    string // ISO 4217 alpha-3: "EUR", "GBP", "USD"
}

type Party struct {
	Name string
	IBAN string
	BIC  string
}

// RailName identifies the interbank payment rail.
type RailName string

const (
	RailSEPA  RailName = "SEPA"
	RailSWIFT RailName = "SWIFT"
)

var (
	ErrInsufficientBalance = errors.New("banking: insufficient balance for debit")
	ErrRailUnavailable     = errors.New("banking: payment rail unavailable — retry later")
	ErrRejectedByBank      = errors.New("banking: payment rejected by receiving bank")
	ErrVersionConflict     = errors.New("banking: concurrent modification — retry")
	ErrRetryExhausted      = errors.New("banking: too many concurrent modifications")
)

// ── Extension ports (exported — infrastructure adapters implement these) ───────

// Rail is the outbound port each payment rail adapter must satisfy.
// payment-worker provides sepa.Rail and swift.Rail implementations.
//
//go:generate go run go.uber.org/mock/mockgen -source engine.go -destination mocks_test.go -package banking
type Rail interface {
	Send(ctx context.Context, order PaymentOrder) (PaymentOrder, error)
}

// Ledger is the outbound port for double-entry account management.
// payment-worker provides the concrete implementation (memory or Postgres).
type Ledger interface {
	GetBalance(ctx context.Context, iban string) (balanceCents, version int64, err error)
	// Debit applies OCC: returns ErrVersionConflict if version is stale.
	Debit(ctx context.Context, iban string, amountCents, expectedVersion int64) error
	Settle(ctx context.Context, orderID string) error
	Reverse(ctx context.Context, orderID string) error
}

// ── Primary port ──────────────────────────────────────────────────────────────

type PaymentEngine interface {
	Initiate(ctx context.Context, order PaymentOrder) (PaymentOrder, error)
}

// ── Engine ────────────────────────────────────────────────────────────────────

var _ PaymentEngine = (*Engine)(nil)

type Engine struct {
	rails   map[RailName]Rail
	ledger  Ledger
	backoff func(attempt int) time.Duration
}

func New(sepaRail, swiftRail Rail, ledger Ledger, opts ...func(*Engine)) *Engine {
	e := &Engine{
		rails:  map[RailName]Rail{RailSEPA: sepaRail, RailSWIFT: swiftRail},
		ledger: ledger,
	}
	for _, o := range opts {
		o(e)
	}
	return e
}

func WithBackoff(fn func(attempt int) time.Duration) func(*Engine) {
	return func(e *Engine) { e.backoff = fn }
}

// JitteredBackoff spreads OCC retries across time — prevents thundering herd
// when multiple workers debit the same account concurrently.
func JitteredBackoff(base time.Duration) func(int) time.Duration {
	return func(attempt int) time.Duration {
		max := int64(time.Duration(attempt) * base)
		if max <= 0 {
			return 0
		}
		return time.Duration(rand.Int64N(max))
	}
}

const maxOCCRetries = 5

func (e *Engine) Initiate(ctx context.Context, order PaymentOrder) (PaymentOrder, error) {
	order.Status = StatusInitiated
	order.CreatedAt = time.Now().UTC()

	if err := e.debitWithOCC(ctx, order.Debtor.IBAN, order.Amount.AmountCents); err != nil {
		return PaymentOrder{}, fmt.Errorf("ledger debit: %w", err)
	}

	rail, ok := e.rails[order.Rail]
	if !ok {
		_ = e.ledger.Reverse(ctx, order.ID)
		return PaymentOrder{}, fmt.Errorf("engine: unknown rail %q", order.Rail)
	}

	sent, err := rail.Send(ctx, order)
	if err != nil {
		_ = e.ledger.Reverse(ctx, order.ID)
		return PaymentOrder{}, fmt.Errorf("rail send: %w", err)
	}

	switch sent.Status {
	case StatusSettled:
		_ = e.ledger.Settle(ctx, order.ID)
		now := time.Now().UTC()
		sent.SettledAt = &now
	case StatusRejected:
		_ = e.ledger.Reverse(ctx, order.ID)
	}
	return sent, nil
}

func (e *Engine) debitWithOCC(ctx context.Context, iban string, amountCents int64) error {
	for attempt := range maxOCCRetries {
		balance, version, err := e.ledger.GetBalance(ctx, iban)
		if err != nil {
			return err
		}
		if balance < amountCents {
			return ErrInsufficientBalance
		}
		err = e.ledger.Debit(ctx, iban, amountCents, version)
		if errors.Is(err, ErrVersionConflict) {
			select {
			case <-ctx.Done():
				return fmt.Errorf("debit: %w after %d attempt(s)", ctx.Err(), attempt+1)
			default:
			}
			if e.backoff != nil {
				time.Sleep(e.backoff(attempt + 1))
			}
			continue
		}
		return err
	}
	return ErrRetryExhausted
}
