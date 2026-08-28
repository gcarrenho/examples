// Package payment is the outbound HTTP client adapter that calls payment-svc.
// It satisfies the orders.charger interface via structural typing —
// orders-svc imports zero code from payment-svc.
package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrInsufficientFunds = errors.New("payment client: insufficient funds")
	ErrDuplicateRequest  = errors.New("payment client: duplicate request in-flight")
	ErrAlreadyProcessed  = errors.New("payment client: already processed")
)

type HTTPClient struct {
	baseURL string
	http    *http.Client
}

func NewHTTPClient(baseURL string, client *http.Client) *HTTPClient {
	return &HTTPClient{baseURL: baseURL, http: client}
}

type chargeRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	AccountID      string `json:"account_id"`
	AmountCents    int64  `json:"amount_cents"`
}

func (c *HTTPClient) Charge(ctx context.Context, key, accountID string, amountCents int64) error {
	body, err := json.Marshal(chargeRequest{IdempotencyKey: key, AccountID: accountID, AmountCents: amountCents})
	if err != nil {
		return fmt.Errorf("payment client: marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/payments/charge", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("payment client: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("payment client: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusCreated, http.StatusOK:
		return nil
	case http.StatusConflict:
		return ErrDuplicateRequest
	case http.StatusUnprocessableEntity:
		return ErrInsufficientFunds
	default:
		return fmt.Errorf("payment client: unexpected status %d", resp.StatusCode)
	}
}
