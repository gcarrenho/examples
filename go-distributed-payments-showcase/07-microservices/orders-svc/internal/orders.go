// Package orders implements the order service for orders-svc.
package orders

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// ── Domain ────────────────────────────────────────────────────────────────────

type Order struct {
	ID          string
	AccountID   string
	AmountCents int64
}

// ── Private port — consumer defines the contract ──────────────────────────────

// charger is what orders-svc needs from the payment capability.
// Defined here (the consumer), never imported from payment-svc.
// The HTTP client in payment/ satisfies this via structural typing.
type charger interface {
	Charge(ctx context.Context, idempotencyKey, accountID string, amountCents int64) error
}

// ── Service ───────────────────────────────────────────────────────────────────

type OrderService struct{ payments charger }

func New(payments charger) *OrderService { return &OrderService{payments: payments} }

func (s *OrderService) PlaceOrder(ctx context.Context, accountID string, amountCents int64) (string, error) {
	orderID := newID()
	if err := s.payments.Charge(ctx, "order:"+orderID, accountID, amountCents); err != nil {
		return "", fmt.Errorf("place order: %w", err)
	}
	return orderID, nil
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
