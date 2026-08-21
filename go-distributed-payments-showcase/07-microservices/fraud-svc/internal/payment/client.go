// Package payment is the outbound HTTP client adapter that calls payment-svc.
// It satisfies fraud-svc's paymentGateway interface (Charge + Reserve) structurally —
// no Go code is shared with payment-svc; the contract is the HTTP API.
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
)

type chargeRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	AccountID      string `json:"account_id"`
	AmountCents    int64  `json:"amount_cents"`
}

type reserveResponse struct {
	ReservationID string `json:"reservation_id"`
}

// Client satisfies fraud-svc's paymentGateway interface {Charge + Reserve}.
// orders-svc has its own client satisfying {Charge only} — completely independent.
type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string, client *http.Client) *Client {
	return &Client{baseURL: baseURL, http: client}
}

// Charge calls POST /payments/charge.
func (c *Client) Charge(ctx context.Context, key, accountID string, amountCents int64) error {
	return c.post(ctx, "/payments/charge", chargeRequest{key, accountID, amountCents}, nil)
}

// Reserve calls POST /payments/reserve and returns the reservation ID.
func (c *Client) Reserve(ctx context.Context, key, accountID string, amountCents int64) (string, error) {
	var resp reserveResponse
	if err := c.post(ctx, "/payments/reserve", chargeRequest{key, accountID, amountCents}, &resp); err != nil {
		return "", err
	}
	return resp.ReservationID, nil
}

func (c *Client) post(ctx context.Context, path string, body any, out any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("payment client: marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(encoded))
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
		if out != nil {
			return json.NewDecoder(resp.Body).Decode(out)
		}
		return nil
	case http.StatusConflict:
		return ErrDuplicateRequest
	case http.StatusUnprocessableEntity:
		return ErrInsufficientFunds
	default:
		return fmt.Errorf("payment client: unexpected status %d", resp.StatusCode)
	}
}
