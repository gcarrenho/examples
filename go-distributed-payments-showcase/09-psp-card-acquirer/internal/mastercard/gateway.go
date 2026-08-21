// Package mastercard is the outbound adapter for Mastercard Banknet.
// Same ISO 8583 framing as Visa NET with Mastercard-specific DE48 subfields.
package mastercard

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	authorization "github.com/examples/go-distributed-payments-showcase/09-psp-card-acquirer/internal"
	"github.com/examples/go-distributed-payments-showcase/09-psp-card-acquirer/internal/iso8583"
)

type wireRequest struct {
	MTI          string `json:"mti"`
	DE2          string `json:"de02"`
	DE3          string `json:"de03"`
	DE4          string `json:"de04"`
	DE7          string `json:"de07"`
	DE11         string `json:"de11"`
	DE37         string `json:"de37"`
	DE41         string `json:"de41"`
	DE42         string `json:"de42"`
	DE49         string `json:"de49"`
	DE48_SF43    string `json:"de48_sf43,omitempty"` // Wallet identifier (APPLE_PAY, GOOGLE_PAY)
	DE48_SF71    string `json:"de48_sf71,omitempty"` // Tokenization indicator
}

type wireResponse struct {
	MTI  string `json:"mti"`
	DE37 string `json:"de37"`
	DE38 string `json:"de38"`
	DE39 string `json:"de39"`
}

type Gateway struct {
	endpoint string
	http     *http.Client
}

func New(endpoint string, client *http.Client) *Gateway {
	return &Gateway{endpoint: endpoint, http: client}
}

func (g *Gateway) Authorize(ctx context.Context, txn authorization.CardTransaction) (authorization.CardTransaction, error) {
	now := time.Now().UTC()
	rrn := txn.IdempotencyKey
	if len(rrn) > 12 {
		rrn = rrn[:12]
	}
	stan := txn.ID
	if len(stan) > 6 {
		stan = stan[:6]
	}

	wire := wireRequest{
		MTI:  string(iso8583.AuthorizationRequest),
		DE2:  txn.CardBIN + txn.CardLast4,
		DE3:  string(iso8583.PurchaseDebit),
		DE4:  iso8583.FormatAmount(txn.AmountCents),
		DE7:  now.Format("0102150405"),
		DE11: stan,
		DE37: rrn,
		DE41: txn.TerminalID,
		DE42: txn.MerchantID,
		DE49: txn.CurrencyCode,
	}

	body, err := json.Marshal(wire)
	if err != nil {
		return authorization.CardTransaction{}, fmt.Errorf("mastercard: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.endpoint+"/authorize", bytes.NewReader(body))
	if err != nil {
		return authorization.CardTransaction{}, fmt.Errorf("mastercard: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.http.Do(req)
	if err != nil {
		return authorization.CardTransaction{}, fmt.Errorf("mastercard: %w", err)
	}
	defer resp.Body.Close()

	var wire0110 wireResponse
	if err := json.NewDecoder(resp.Body).Decode(&wire0110); err != nil {
		return authorization.CardTransaction{}, fmt.Errorf("mastercard: decode 0110: %w", err)
	}
	if wire0110.MTI != string(iso8583.AuthorizationResponse) {
		return authorization.CardTransaction{}, fmt.Errorf("mastercard: unexpected MTI %q", wire0110.MTI)
	}

	if err := mapResponseCode(iso8583.ResponseCode(wire0110.DE39)); err != nil {
		return authorization.CardTransaction{}, err
	}

	txn.Status = authorization.StatusApproved
	txn.AuthCode = wire0110.DE38
	txn.NetworkRef = wire0110.DE37
	return txn, nil
}

func mapResponseCode(rc iso8583.ResponseCode) error {
	switch rc {
	case iso8583.ResponseApproved:
		return nil
	case iso8583.ResponseInsufficientFunds:
		return authorization.ErrInsufficientFunds
	case iso8583.ResponseExpiredCard:
		return authorization.ErrExpiredCard
	case iso8583.ResponseIssuerUnavailable:
		return authorization.ErrNetworkUnavailable
	default:
		return authorization.ErrDeclined
	}
}
