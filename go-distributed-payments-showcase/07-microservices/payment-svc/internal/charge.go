package payment

import (
	"context"
	"errors"
	"fmt"
)

const maxOCCRetries = 5

// Charge debits amountCents from accountID under idempotency key.
func (s *Service) Charge(ctx context.Context, key, accountID string, amountCents int64) error {
	if err := s.acquireKey(ctx, key); err != nil {
		return err
	}
	if err := s.applyDelta(ctx, accountID, -amountCents); err != nil {
		_ = s.idempotency.Release(ctx, key)
		return err
	}
	return s.idempotency.Complete(ctx, key)
}

// Refund credits amountCents back to accountID under idempotency key.
func (s *Service) Refund(ctx context.Context, key, accountID string, amountCents int64) error {
	if err := s.acquireKey(ctx, key); err != nil {
		return err
	}
	if err := s.applyDelta(ctx, accountID, +amountCents); err != nil {
		_ = s.idempotency.Release(ctx, key)
		return err
	}
	return s.idempotency.Complete(ctx, key)
}

// acquireKey is shared by Charge, Refund, and Reserve — lives here, used by reserve.go too.
func (s *Service) acquireKey(ctx context.Context, key string) error {
	acquired, err := s.idempotency.Acquire(ctx, key)
	if err != nil {
		return fmt.Errorf("idempotency: %w", err)
	}
	if acquired {
		return nil
	}
	done, _ := s.idempotency.IsCompleted(ctx, key)
	if done {
		return ErrAlreadyProcessed
	}
	return ErrDuplicateRequest
}

// applyDelta is shared by Charge and Refund.
func (s *Service) applyDelta(ctx context.Context, accountID string, delta int64) error {
	for range maxOCCRetries {
		balance, version, err := s.accounts.GetAccount(ctx, accountID)
		if err != nil {
			return err
		}
		newBalance := balance + delta
		if newBalance < 0 {
			return ErrInsufficientFunds
		}
		err = s.accounts.UpdateAccount(ctx, accountID, newBalance, version)
		if errors.Is(err, ErrVersionConflict) {
			continue
		}
		return err
	}
	return ErrRetryExhausted
}
