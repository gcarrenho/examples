package payment

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

// paymentService is the handler's private contract — defined by the handler, not the service.
type paymentService interface {
	Charge(ctx context.Context, key, accountID string, amountCents int64) error
	Refund(ctx context.Context, key, accountID string, amountCents int64) error
}

type chargeRequest struct {
	AccountID      string `json:"account_id"`
	AmountCents    int64  `json:"amount_cents"`
	IdempotencyKey string `json:"idempotency_key"`
}

// Handler is the HTTP inbound adapter for the payment-processor component.
type Handler struct{ svc paymentService }

func NewHandler(svc paymentService) *Handler { return &Handler{svc: svc} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req chargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.IdempotencyKey == "" || req.AccountID == "" {
		http.Error(w, "idempotency_key and account_id required", http.StatusBadRequest)
		return
	}

	var err error
	switch r.Method {
	case http.MethodPost:
		err = h.svc.Charge(r.Context(), req.IdempotencyKey, req.AccountID, req.AmountCents)
	case http.MethodDelete:
		err = h.svc.Refund(r.Context(), req.IdempotencyKey, req.AccountID, req.AmountCents)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err != nil {
		switch {
		case errors.Is(err, ErrAlreadyProcessed):
			w.Header().Set("Idempotent-Replayed", "true")
			w.WriteHeader(http.StatusOK)
		case errors.Is(err, ErrDuplicateRequest):
			http.Error(w, "request already in progress", http.StatusConflict)
		case errors.Is(err, ErrInsufficientFunds):
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		case errors.Is(err, ErrAccountNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusCreated)
}
