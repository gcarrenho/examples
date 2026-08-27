// Package fake provides deterministic payment rails for local and integration tests.
package fake

import (
	"context"
	"log/slog"
	"time"

	banking "github.com/examples/banking-core"
)

// Rail simulates a transient rail failure followed by a successful settlement.
// It exercises retry handling without calling an external bank.
type Rail struct {
	retryDelay time.Duration
	logger     *slog.Logger
}

func New(retryDelay time.Duration, logger *slog.Logger) *Rail {
	return &Rail{retryDelay: retryDelay, logger: logger}
}

func (r *Rail) Send(ctx context.Context, order banking.PaymentOrder) (banking.PaymentOrder, error) {
	for attempt := 1; attempt <= 2; attempt++ {
		if attempt == 1 {
			r.logger.WarnContext(ctx, "fake rail transient failure; retrying",
				slog.String("uetr", order.UETR), slog.Int("attempt", attempt))
			if err := wait(ctx, r.retryDelay); err != nil {
				return banking.PaymentOrder{}, err
			}
			continue
		}

		r.logger.InfoContext(ctx, "fake rail accepted payment",
			slog.String("uetr", order.UETR), slog.Int("attempt", attempt))
		order.Status = banking.StatusSettled
		return order, nil
	}
	return banking.PaymentOrder{}, banking.ErrRailUnavailable
}

func wait(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}