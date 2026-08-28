package api

import (
	"context"
	"log/slog"
	"time"

	banking "github.com/examples/banking-core"
)

// TimeoutReaper periodically scans for orders that never got a result within
// the business timeout and marks them as rejected — without it, a payment-worker
// crash or a lost Kafka message would leave the client polling GET /payments/{uetr}
// forever with no way to distinguish "still processing" from "never going to finish".
type TimeoutReaper struct {
	pending   *PendingIndex
	store     *ResultsStore
	callbacks *CallbackStore
	webhook   *WebhookNotifier
	timeout   time.Duration
	logger    *slog.Logger
}

func NewTimeoutReaper(pending *PendingIndex, store *ResultsStore, callbacks *CallbackStore, webhook *WebhookNotifier, timeout time.Duration, logger *slog.Logger) *TimeoutReaper {
	return &TimeoutReaper{pending: pending, store: store, callbacks: callbacks, webhook: webhook, timeout: timeout, logger: logger}
}

// Run blocks until ctx is cancelled, sweeping for stale orders every interval.
// Intended to run in its own goroutine alongside the HTTP server.
func (r *TimeoutReaper) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.sweep(ctx)
		}
	}
}

func (r *TimeoutReaper) sweep(ctx context.Context) {
	stale, err := r.pending.Stale(ctx, r.timeout)
	if err != nil {
		r.logger.ErrorContext(ctx, "reaper: stale scan failed", slog.String("err", err.Error()))
		return
	}
	for _, uetr := range stale {
		result := banking.PaymentOrderResult{
			UETR:       uetr,
			Status:     banking.StatusRejected,
			Reason:     "timeout: no result received within business SLA",
			OccurredAt: time.Now().UTC(),
		}
		if err := r.store.Save(ctx, result); err != nil {
			r.logger.ErrorContext(ctx, "reaper: save timeout result failed", slog.String("uetr", uetr), slog.String("err", err.Error()))
			continue
		}
		_ = r.pending.Remove(ctx, uetr)
		r.logger.WarnContext(ctx, "order timed out — marked RJCT", slog.String("uetr", uetr))

		if url, found, _ := r.callbacks.Get(ctx, uetr); found {
			go r.webhook.Notify(context.Background(), url, result)
		}
	}
}
