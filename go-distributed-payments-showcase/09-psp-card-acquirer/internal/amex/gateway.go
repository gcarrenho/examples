// Package amex is the outbound adapter for American Express (closed-loop network).
// AmEx uses their own REST API — not ISO 8583. This adapter accepts
// authorization.CardTransaction and translates to AmEx's proprietary format.
package amex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	authorization "github.com/examples/go-distributed-payments-showcase/09-psp-card-acquirer/internal"
)

type authorizeRequest struct {
	TransactionID string  `json:"transaction_id"`
	MerchantID    string  `json:"merchant_id"`
	Amount        amexAmt `json:"amount"`
	Card          amexCard `json:"card"`
}

type amexAmt struct {
	Total    string `json:"total"`    // decimal string: "50.00"
	Currency string `json:"currency"` // ISO 4217 alpha-3: "USD", "ARS"
}

type amexCard struct {
	NumberToken string `json:"number_token"`
}

type authorizeResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`  // "APPROVED" | "DECLINED" | "REFERRAL"
	AuthID        string `json:"auth_id"`
}

var currencyAlpha = map[string]string{
	"840": "USD", "032": "ARS", "986": "BRL", "826": "GBP", "978": "EUR",
}

type Gateway struct {
	endpoint string
	apiKey   string
	http     *http.Client
}

func New(endpoint, apiKey string, client *http.Client) *Gateway {
	return &Gateway{endpoint: endpoint, apiKey: apiKey, http: client}
}

// Authorize satisfies authorization.network via structural typing.
// Translates: authorization.CardTransaction → AmEx REST → authorization.CardTransaction
func (g *Gateway) Authorize(ctx context.Context, txn authorization.CardTransaction) (authorization.CardTransaction, error) {
	currency := currencyAlpha[txn.CurrencyCode]
	if currency == "" {
		currency = "USD"
	}

	amexReq := authorizeRequest{
		TransactionID: txn.IdempotencyKey,
		MerchantID:    txn.MerchantID,
		Amount: amexAmt{
			Total:    formatAmount(txn.AmountCents),
			Currency: currency,
		},
		Card: amexCard{NumberToken: txn.CardBIN + txn.CardLast4},
	}

	body, err := json.Marshal(amexReq)
	if err != nil {
		return authorization.CardTransaction{}, fmt.Errorf("amex: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.endpoint+"/payments/authorize", bytes.NewReader(body))
	if err != nil {
		return authorization.CardTransaction{}, fmt.Errorf("amex: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-AMEX-API-KEY", g.apiKey)

	resp, err := g.http.Do(req)
	if err != nil {
		return authorization.CardTransaction{}, fmt.Errorf("amex: %w", err)
	}
	defer resp.Body.Close()

	var amexResp authorizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&amexResp); err != nil {
		return authorization.CardTransaction{}, fmt.Errorf("amex: decode: %w", err)
	}

	if amexResp.Status != "APPROVED" {
		return authorization.CardTransaction{}, authorization.ErrDeclined
	}

	txn.Status = authorization.StatusApproved
	txn.AuthCode = amexResp.AuthID
	txn.NetworkRef = amexResp.TransactionID
	return txn, nil
}

func formatAmount(cents int64) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}
