package api

import (
	"context"
	"log/slog"

	banking "github.com/examples/banking-core"
)

// ResultProcessor satisfies kafka.resultHandler — the single place that reacts to a
// PaymentOrderResult arriving from payment-worker: persist it, stop tracking it as
// pending, and fire the client's webhook if one was registered.
type ResultProcessor struct {
	store     *ResultsStore
	pending   *PendingIndex
	callbacks *CallbackStore
	webhook   *WebhookNotifier
	logger    *slog.Logger
}

func NewResultProcessor(store *ResultsStore, pending *PendingIndex, callbacks *CallbackStore, webhook *WebhookNotifier, logger *slog.Logger) *ResultProcessor {
	return &ResultProcessor{store: store, pending: pending, callbacks: callbacks, webhook: webhook, logger: logger}
}

func (p *ResultProcessor) HandleResult(ctx context.Context, result banking.PaymentOrderResult) {
	if err := p.store.Save(ctx, result); err != nil {
		p.logger.ErrorContext(ctx, "failed to save result", slog.String("uetr", result.UETR), slog.String("err", err.Error()))
	}
	if err := p.pending.Remove(ctx, result.UETR); err != nil {
		p.logger.ErrorContext(ctx, "failed to remove from pending index", slog.String("uetr", result.UETR), slog.String("err", err.Error()))
	}
	p.notifyWebhook(ctx, result)
}

func (p *ResultProcessor) notifyWebhook(ctx context.Context, result banking.PaymentOrderResult) {
	url, found, err := p.callbacks.Get(ctx, result.UETR)
	if err != nil {
		p.logger.ErrorContext(ctx, "failed to read callback url", slog.String("uetr", result.UETR), slog.String("err", err.Error()))
		return
	}
	if !found {
		return // client did not register a webhook — GET /payments/{uetr} is the fallback
	}
	// Detached from ctx (which is tied to the Kafka consumer loop) so cancelling the
	// consumer doesn't abort an in-flight webhook delivery; Notify has its own timeout.
	go p.webhook.Notify(context.Background(), url, result)
}
