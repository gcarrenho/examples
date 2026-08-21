package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	banking "github.com/examples/banking-core"
)

// WebhookNotifier delivers the final payment result to a client-provided callback URL.
// Best-effort with a small bounded retry — the client's GET /payments/{uetr} remains
// the source of truth if the webhook delivery ultimately fails.
type WebhookNotifier struct {
	http   *http.Client
	logger *slog.Logger
}

func NewWebhookNotifier(client *http.Client, logger *slog.Logger) *WebhookNotifier {
	return &WebhookNotifier{http: client, logger: logger}
}

const (
	webhookMaxAttempts = 3
	webhookTimeout     = 5 * time.Second
)

// Notify POSTs the result as JSON to callbackURL. Intended to be called in its own
// goroutine — never blocks Kafka consumption or the HTTP request that triggered it.
func (n *WebhookNotifier) Notify(ctx context.Context, callbackURL string, result banking.PaymentOrderResult) {
	body, err := json.Marshal(map[string]string{
		"uetr": result.UETR, "status": string(result.Status), "reason": result.Reason,
	})
	if err != nil {
		n.logger.ErrorContext(ctx, "webhook: marshal failed", slog.String("uetr", result.UETR), slog.String("err", err.Error()))
		return
	}

	for attempt := 1; attempt <= webhookMaxAttempts; attempt++ {
		reqCtx, cancel := context.WithTimeout(ctx, webhookTimeout)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, callbackURL, bytes.NewReader(body))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			resp, doErr := n.http.Do(req)
			if doErr == nil {
				resp.Body.Close()
				cancel()
				if resp.StatusCode < 300 {
					return // delivered
				}
			}
		}
		cancel()
		if attempt < webhookMaxAttempts {
			time.Sleep(time.Duration(attempt) * time.Second) // linear backoff, no need for jitter at this low volume
		}
	}
	n.logger.WarnContext(ctx, "webhook: all delivery attempts failed",
		slog.String("uetr", result.UETR), slog.String("callback_url", callbackURL))
}
