// Package fraud implements the fraud detection service.
//
// fraud-svc defines its OWN contracts for every external capability it needs.
// Those contracts live in ports.go — separate from business logic — so fraud.go
// stays focused on orchestration without being cluttered by interface declarations.
package fraud

import (
	"context"
	"errors"
	"fmt"
)

var ErrFraudDetected = errors.New("fraud: transaction blocked — suspicious activity")

// Service evaluates payment requests and charges only when fraud scoring approves.
type Service struct{ payments paymentGateway }

func New(payments paymentGateway) *Service { return &Service{payments: payments} }

// ProcessPayment runs a two-phase flow: Reserve → fraud score → Charge if approved.
//
// Why Reserve first?
//   - Confirms funds exist before spending time on fraud scoring.
//   - In production, the Reserve holds the funds so they can't be spent elsewhere
//     during the fraud check window (which may take milliseconds to seconds).
func (s *Service) ProcessPayment(ctx context.Context, key, accountID string, amountCents int64) error {
	// Phase 1: Hold funds (doesn't debit — just confirms availability).
	_, err := s.payments.Reserve(ctx, "rsv:"+key, accountID, amountCents)
	if err != nil {
		return fmt.Errorf("fraud: reserve: %w", err)
	}

	// Phase 2: Fraud scoring (simplified — production would call an ML model).
	if result := score(accountID, amountCents); result == declined {
		// In production: release the reservation so funds are freed.
		return ErrFraudDetected
	}

	// Phase 3: Execute the charge only after approval.
	return s.payments.Charge(ctx, key, accountID, amountCents)
}

type verdict int

const (
	approved verdict = iota
	declined
)

// score is a simplified fraud rule. Replace with an ML model or rules engine.
func score(_ string, amountCents int64) verdict {
	if amountCents > 500_000 { // flag transactions above $5,000
		return declined
	}
	return approved
}
