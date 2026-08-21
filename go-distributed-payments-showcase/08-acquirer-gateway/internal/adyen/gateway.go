// Package adyen is the outbound adapter for Adyen (global acquirer).
package adyen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	gateway "github.com/examples/go-distributed-payments-showcase/08-acquirer-gateway/internal"
)

type paymentRequest struct {
	Reference       string `json:"reference"`
	Amount          amount `json:"amount"`
	MerchantAccount string `json:"merchantAccount"`
}

type amount struct {
	Value    int64  `json:"value"`
	Currency string `json:"currency"`
}

type paymentResponse struct {
	ResultCode   string `json:"resultCode"`
	PspReference string `json:"pspReference"`
	AuthCode     string `json:"authCode"`
}

type HTTPGateway struct {
	baseURL    string
	apiKey     string
	merchantID string
	http       *http.Client
}

func New(baseURL, apiKey, merchantID string, client *http.Client) *HTTPGateway {
	return &HTTPGateway{baseURL: baseURL, apiKey: apiKey, merchantID: merchantID, http: client}
}

func (g *HTTPGateway) Authorize(ctx context.Context, req gateway.AuthRequest) (gateway.AuthResponse, error) {
	adyenReq := paymentRequest{
		Reference:       req.IdempotencyKey,
		Amount:          amount{Value: req.AmountCents, Currency: req.Currency},
		MerchantAccount: g.merchantID,
	}
	body, err := json.Marshal(adyenReq)
	if err != nil {
		return gateway.AuthResponse{}, fmt.Errorf("adyen: marshal: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+"/v68/payments", bytes.NewReader(body))
	if err != nil {
		return gateway.AuthResponse{}, fmt.Errorf("adyen: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", g.apiKey)
	resp, err := g.http.Do(httpReq)
	if err != nil {
		return gateway.AuthResponse{}, fmt.Errorf("adyen: %w", err)
	}
	defer resp.Body.Close()

	var adyenResp paymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&adyenResp); err != nil {
		return gateway.AuthResponse{}, fmt.Errorf("adyen: decode: %w", err)
	}
	if adyenResp.ResultCode != "Authorised" {
		return gateway.AuthResponse{}, gateway.ErrDeclined
	}
	return gateway.AuthResponse{
		Approved:   true,
		AuthCode:   adyenResp.AuthCode,
		NetworkRef: adyenResp.PspReference,
	}, nil
}
