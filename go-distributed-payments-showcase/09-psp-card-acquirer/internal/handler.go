package authorization

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

// authorizer is the handler's private contract — the minimum it needs from the service.
// Defined here, by the consumer. The handler never imports the service package.
type authorizer interface {
	Authorize(ctx context.Context, txn CardTransaction) (CardTransaction, error)
}

// Handler is the HTTP inbound adapter for card authorization.
type Handler struct{ svc authorizer }

func NewHandler(svc authorizer) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /authorize", h.authorize)
}

// ── Request / Response ────────────────────────────────────────────────────────
// Structs live next to the handler that uses them — no separate request.go file.

// AuthorizeRequest is the inbound payload: what the CALLER provides.
// auth_code and network_ref are absent — only the network assigns them.
type AuthorizeRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	AmountCents    int64  `json:"amount_cents"`
	CurrencyCode   string `json:"currency_code"` // ISO 4217 numeric
	CardBIN        string `json:"card_bin"`        // first 6-8 digits; used for routing
	CardLast4      string `json:"card_last4"`
	CardCountry    string `json:"card_country"` // ISO 3166-1 alpha-2
	MerchantID     string `json:"merchant_id"`
	TerminalID     string `json:"terminal_id"`
}

// AuthorizeResponse is the outbound payload: what the SYSTEM computed.
type AuthorizeResponse struct {
	Status     string `json:"status"`
	AuthCode   string `json:"auth_code,omitempty"`
	NetworkRef string `json:"network_ref,omitempty"`
}

// ── Handler logic ─────────────────────────────────────────────────────────────

func (h *Handler) authorize(w http.ResponseWriter, r *http.Request) {
	var req AuthorizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.IdempotencyKey == "" || req.MerchantID == "" || req.CardBIN == "" {
		http.Error(w, "idempotency_key, merchant_id, and card_bin are required", http.StatusBadRequest)
		return
	}

	txn := CardTransaction{
		ID:             req.IdempotencyKey,
		IdempotencyKey: req.IdempotencyKey,
		AmountCents:    req.AmountCents,
		CurrencyCode:   req.CurrencyCode,
		CardBIN:        req.CardBIN,
		CardBrand:      BrandFromBIN(req.CardBIN), // derived; never trusted from caller
		CardLast4:      req.CardLast4,
		CardCountry:    req.CardCountry,
		MerchantID:     req.MerchantID,
		TerminalID:     req.TerminalID,
	}

	result, err := h.svc.Authorize(r.Context(), txn)
	if err != nil {
		h.writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(AuthorizeResponse{
		Status:     string(result.Status),
		AuthCode:   result.AuthCode,
		NetworkRef: result.NetworkRef,
	})
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrDeclined), errors.Is(err, ErrInsufficientFunds), errors.Is(err, ErrExpiredCard):
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(AuthorizeResponse{Status: "DECLINED"})
	case errors.Is(err, ErrNetworkUnavailable):
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	case errors.Is(err, ErrDuplicateSTAN):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
