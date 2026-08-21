package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

// authorizer is the handler's private view of the Router.
type authorizer interface {
	Authorize(ctx context.Context, req AuthRequest) (AuthResponse, error)
}

type Handler struct{ router authorizer }

func NewHandler(router authorizer) *Handler { return &Handler{router: router} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /acquire", h.authorize)
}

type authorizeRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	AmountCents    int64  `json:"amount_cents"`
	Currency       string `json:"currency"`
	CardBIN        string `json:"card_bin"`
	CardCountry    string `json:"card_country"`
	MerchantID     string `json:"merchant_id"`
}

type authorizeResponse struct {
	Approved   bool   `json:"approved"`
	AuthCode   string `json:"auth_code"`
	NetworkRef string `json:"network_ref"`
}

func (h *Handler) authorize(w http.ResponseWriter, r *http.Request) {
	var req authorizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	resp, err := h.router.Authorize(r.Context(), AuthRequest{
		IdempotencyKey: req.IdempotencyKey,
		AmountCents:    req.AmountCents,
		Currency:       req.Currency,
		CardBIN:        req.CardBIN,
		CardCountry:    req.CardCountry,
		MerchantID:     req.MerchantID,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrDeclined):
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		case errors.Is(err, ErrAcquirerUnavailable):
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(authorizeResponse{
		Approved:   resp.Approved,
		AuthCode:   resp.AuthCode,
		NetworkRef: resp.NetworkRef,
	})
}
