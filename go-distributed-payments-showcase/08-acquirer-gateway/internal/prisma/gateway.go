// Package prisma is the outbound adapter for Prisma (Argentina's largest acquirer).
package prisma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	gateway "github.com/examples/go-distributed-payments-showcase/08-acquirer-gateway/internal"
)

type authorizeRequest struct {
	NumeroTransaccion string  `json:"nro_transaccion"`
	Monto             float64 `json:"monto"`
	CodigoMoneda      string  `json:"cod_moneda"`
	Comercio          string  `json:"id_comercio"`
}

type authorizeResponse struct {
	CodigoRespuesta  string `json:"cod_respuesta"`
	CodigoAprobacion string `json:"cod_aprobacion"`
	NumeroReferencia string `json:"nro_referencia"`
}

type HTTPGateway struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string, client *http.Client) *HTTPGateway {
	return &HTTPGateway{baseURL: baseURL, http: client}
}

func (g *HTTPGateway) Authorize(ctx context.Context, req gateway.AuthRequest) (gateway.AuthResponse, error) {
	prismaReq := authorizeRequest{
		NumeroTransaccion: req.IdempotencyKey,
		Monto:             float64(req.AmountCents) / 100,
		CodigoMoneda:      req.Currency,
		Comercio:          req.MerchantID,
	}
	body, err := json.Marshal(prismaReq)
	if err != nil {
		return gateway.AuthResponse{}, fmt.Errorf("prisma: marshal: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+"/autorizacion", bytes.NewReader(body))
	if err != nil {
		return gateway.AuthResponse{}, fmt.Errorf("prisma: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := g.http.Do(httpReq)
	if err != nil {
		return gateway.AuthResponse{}, fmt.Errorf("prisma: %w", err)
	}
	defer resp.Body.Close()

	var prismaResp authorizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&prismaResp); err != nil {
		return gateway.AuthResponse{}, fmt.Errorf("prisma: decode: %w", err)
	}
	if prismaResp.CodigoRespuesta != "00" {
		return gateway.AuthResponse{}, gateway.ErrDeclined
	}
	return gateway.AuthResponse{
		Approved:   true,
		AuthCode:   prismaResp.CodigoAprobacion,
		NetworkRef: prismaResp.NumeroReferencia,
	}, nil
}
