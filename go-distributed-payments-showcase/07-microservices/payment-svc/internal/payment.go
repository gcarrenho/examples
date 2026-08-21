// Package payment implements the payment service for the payment-svc microservice.
//
// When this file grows beyond ~300 lines the answer is NOT new sub-packages —
// it's more files in the same package. charge.go and reserve.go demonstrate this:
// they are separate files but share the same package, struct, and helpers.
package payment

import (
	"context"
	"errors"
)

// ── Domain errors ─────────────────────────────────────────────────────────────

var (
	ErrDuplicateRequest    = errors.New("payment: request already in-flight")
	ErrAlreadyProcessed    = errors.New("payment: request already completed")
	ErrInsufficientFunds   = errors.New("payment: insufficient funds")
	ErrAccountNotFound     = errors.New("payment: account not found")
	ErrVersionConflict     = errors.New("payment: concurrent modification — retry")
	ErrRetryExhausted      = errors.New("payment: too many concurrent modifications")
	ErrReservationNotFound = errors.New("payment: reservation not found")
)

// ── Private secondary ports ───────────────────────────────────────────────────
// Unexported: cmd/main.go passes implementations structurally without naming these types.

//go:generate go run go.uber.org/mock/mockgen -source payment.go -destination mocks_test.go -package payment
type idempotencyStore interface {
	Acquire(ctx context.Context, key string) (acquired bool, err error)
	Complete(ctx context.Context, key string) error
	IsCompleted(ctx context.Context, key string) (bool, error)
	Release(ctx context.Context, key string) error
}

type accountRepository interface {
	GetAccount(ctx context.Context, accountID string) (balanceCents, version int64, err error)
	UpdateAccount(ctx context.Context, accountID string, newBalanceCents, expectedVersion int64) error
}

type reservationStore interface {
	Create(ctx context.Context, id, accountID string, amountCents int64) error
	Get(ctx context.Context, id string) (accountID string, amountCents int64, err error)
}

// ── Primary port ──────────────────────────────────────────────────────────────

// ChargeService is the exported contract — what the HTTP handler calls.
// Each consumer (orders-svc, fraud-svc, billing-svc...) uses only the subset it needs,
// defining its own private interface via structural typing.
type ChargeService interface {
	Charge(ctx context.Context, idempotencyKey, accountID string, amountCents int64) error
	Refund(ctx context.Context, idempotencyKey, accountID string, amountCents int64) error
	// Reserve holds funds without debiting. Used for two-phase payments (fraud check, etc).
	Reserve(ctx context.Context, idempotencyKey, accountID string, amountCents int64) (reservationID string, err error)
}

// ── Service struct and constructor ────────────────────────────────────────────
// Methods live in charge.go and reserve.go — same package, different files.

var _ ChargeService = (*Service)(nil)

type Service struct {
	idempotency  idempotencyStore
	accounts     accountRepository
	reservations reservationStore
}

func New(idempotency idempotencyStore, accounts accountRepository, reservations reservationStore) *Service {
	return &Service{idempotency: idempotency, accounts: accounts, reservations: reservations}
}
