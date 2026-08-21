package paymentprocessor

import (
	"context"
	"net/http"
	"time"

	payment "github.com/examples/go-distributed-payments-showcase/06-payment-processor/internal"
	"github.com/examples/go-distributed-payments-showcase/06-payment-processor/internal/memory"
)

var (
	ErrDuplicateRequest  = payment.ErrDuplicateRequest
	ErrAlreadyProcessed  = payment.ErrAlreadyProcessed
	ErrInsufficientFunds = payment.ErrInsufficientFunds
	ErrAccountNotFound   = payment.ErrAccountNotFound
)

type IdempotencyStore interface {
	SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Del(ctx context.Context, key string) error
}

type AccountRepository interface {
	GetAccount(ctx context.Context, accountID string) (balanceCents, version int64, err error)
	UpdateAccount(ctx context.Context, accountID string, newBalanceCents, expectedVersion int64) error
}

type EventPublisher interface {
	Publish(ctx context.Context, eventType, accountID, paymentKey string, amountCents int64) error
}

type Service interface {
	Charge(ctx context.Context, idempotencyKey, accountID string, amountCents int64) error
	Refund(ctx context.Context, idempotencyKey, accountID string, amountCents int64) error
}

func New(store IdempotencyStore, repo AccountRepository, pub EventPublisher) Service {
	return payment.New(store, repo, pub)
}

func NewInMemory() Service {
	return payment.New(memory.NewIdempotencyStore(), memory.NewAccountRepo(), memory.NewPublisher())
}

func NewHTTPHandler(svc Service) http.Handler {
	return payment.NewHandler(svc)
}

//
// This package follows the component pattern: only the root package is importable
// from outside. Internal business logic, repositories and adapters live under
// internal/ and are invisible to the rest of the module.
//
// # Consuming this component
//
// Other bounded contexts must NOT import [Service] directly. Instead they define
// the smallest interface they need and receive a *paymentprocessor.Service at
// wire-up time — Go's structural typing guarantees compatibility without coupling:
//
//	// In the "orders" bounded context:
//	type paymentCharge interface {
//	    Charge(ctx context.Context, key, accountID string, amountCents int64) error
//	}
//	func NewOrdersService(charger paymentCharge) *OrdersService { ... }
//
//	// At main.go wire-up time:
//	payment := paymentprocessor.NewInMemory()
//	orders  := ordersvc.NewOrdersService(payment) // ← structural typing, no import of Service
