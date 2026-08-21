// Package worker processes PaymentOrderInitiated events from Kafka.
// Applies exactly-once semantics: Redis idempotency (Case 01) + OCC ledger (Case 02).
package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	banking "github.com/examples/banking-core"
)

// idempotencyStore prevents double processing when Kafka re-delivers a message.
// Redis satisfies this; in-memory for tests.
type idempotencyStore interface {
	Acquire(ctx context.Context, key string) (bool, error)
	Complete(ctx context.Context, key string) error
	IsCompleted(ctx context.Context, key string) (bool, error)
	Release(ctx context.Context, key string) error
}

// resultPublisher reports the final outcome so payment-api can answer GET /payments/{uetr}.
type resultPublisher interface {
	PublishResult(ctx context.Context, result banking.PaymentOrderResult) error
}

// Worker handles one PaymentOrderInitiated event with exactly-once semantics.
type Worker struct {
	engine      banking.PaymentEngine
	idempotency idempotencyStore
	results     resultPublisher
	logger      *slog.Logger
}

func New(engine banking.PaymentEngine, idempotency idempotencyStore, results resultPublisher, logger *slog.Logger) *Worker {
	return &Worker{engine: engine, idempotency: idempotency, results: results, logger: logger}
}

// ProcessEvent implements the event handler contract expected by kafka.Consumer.
//
//  1. Redis SETNX idempotency gate — skip if already STARTED or COMPLETED
//  2. banking.Engine.Initiate — OCC debit + SEPA/SWIFT send
//  3. Mark COMPLETED and publish the final result — future re-deliveries are no-ops
//
// Business errors (insufficient funds, bank rejection) are terminal: the key is marked
// COMPLETED, not released, so a Kafka redelivery never retries a payment that can never
// succeed. Only infrastructure errors (rail down, Redis down) release the key for retry.
func (w *Worker) ProcessEvent(ctx context.Context, event banking.PaymentOrderInitiated) error {
	key := "payment-order:" + event.Order.UETR

	acquired, err := w.idempotency.Acquire(ctx, key)
	if err != nil {
		return fmt.Errorf("worker: idempotency acquire: %w", err)
	}
	if !acquired {
		done, _ := w.idempotency.IsCompleted(ctx, key)
		if done {
			w.logger.InfoContext(ctx, "skipping already-completed order", slog.String("uetr", event.Order.UETR))
			return nil
		}
		w.logger.WarnContext(ctx, "order already in-flight", slog.String("uetr", event.Order.UETR))
		return nil
	}

	result, err := w.engine.Initiate(ctx, event.Order)
	switch {
	case errors.Is(err, banking.ErrInsufficientBalance), errors.Is(err, banking.ErrRejectedByBank):
		// Business error: terminal, will never succeed on retry — mark COMPLETED
		// so Kafka redelivery is a no-op instead of retrying forever.
		_ = w.idempotency.Complete(ctx, key)
		w.logger.WarnContext(ctx, "payment rejected — business error",
			slog.String("uetr", event.Order.UETR), slog.String("reason", err.Error()))
		w.publishResult(ctx, event.Order.UETR, banking.StatusRejected, err.Error())
		return nil

	case err != nil:
		// Infrastructure error: transient, release the key so redelivery can retry.
		_ = w.idempotency.Release(ctx, key)
		return fmt.Errorf("worker: engine: %w", err)
	}

	if err := w.idempotency.Complete(ctx, key); err != nil {
		w.logger.ErrorContext(ctx, "failed to mark order completed",
			slog.String("uetr", event.Order.UETR), slog.String("err", err.Error()))
	}
	w.publishResult(ctx, result.UETR, result.Status, "")
	return nil
}

// publishResult best-effort reports the outcome. A failure here does not affect
// idempotency or the ledger — it only means the client's GET /payments/{uetr}
// will be stale until the next successful publish or a manual reconciliation.
func (w *Worker) publishResult(ctx context.Context, uetr string, status banking.PaymentStatus, reason string) {
	err := w.results.PublishResult(ctx, banking.PaymentOrderResult{
		UETR: uetr, Status: status, Reason: reason, OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		w.logger.ErrorContext(ctx, "failed to publish result", slog.String("uetr", uetr), slog.String("err", err.Error()))
	}
}
