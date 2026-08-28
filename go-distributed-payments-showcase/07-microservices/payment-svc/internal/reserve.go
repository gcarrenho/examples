package payment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// Reserve holds amountCents for accountID without debiting.
// Returns a reservationID that identifies the hold.
//
// Use case: fraud-svc calls Reserve first to confirm funds exist, runs its fraud
// scoring, then calls Charge only if the payment is approved. The reservation
// prevents another request from using the same funds in the interim.
func (s *Service) Reserve(ctx context.Context, key, accountID string, amountCents int64) (string, error) {
	if err := s.acquireKey(ctx, key); err != nil {
		return "", err
	}

	// Verify funds are available without debiting (read-only check).
	balance, _, err := s.accounts.GetAccount(ctx, accountID)
	if err != nil {
		_ = s.idempotency.Release(ctx, key)
		return "", fmt.Errorf("reserve: get account: %w", err)
	}
	if balance < amountCents {
		_ = s.idempotency.Release(ctx, key)
		return "", ErrInsufficientFunds
	}

	reservationID := newReservationID()
	if err := s.reservations.Create(ctx, reservationID, accountID, amountCents); err != nil {
		_ = s.idempotency.Release(ctx, key)
		return "", fmt.Errorf("reserve: create: %w", err)
	}

	_ = s.idempotency.Complete(ctx, key)
	return reservationID, nil
}

func newReservationID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "rsv-" + hex.EncodeToString(b)
}
