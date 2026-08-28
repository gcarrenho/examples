package fraud

import (
	"encoding/json"
	"errors"
	"net/http"
)

type processRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	AccountID      string `json:"account_id"`
	AmountCents    int64  `json:"amount_cents"`
}

// Handler is the inbound HTTP adapter for fraud-svc.
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// AUTH TODO: fraud-svc is called by other internal services only — same
	// service-to-service auth requirement as payment-svc (mTLS or service JWT).
	// A fraud-scoring endpoint is a common attack target if left unauthenticated:
	// anyone able to reach it could probe the fraud model or bypass Charge entirely.
	//
	// mux.Handle("POST /fraud/process", requireServiceAuth(http.HandlerFunc(h.process)))
	mux.HandleFunc("POST /fraud/process", h.process)
}

func (h *Handler) process(w http.ResponseWriter, r *http.Request) {
	var req processRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := h.svc.ProcessPayment(r.Context(), req.IdempotencyKey, req.AccountID, req.AmountCents); err != nil {
		switch {
		case errors.Is(err, ErrFraudDetected):
			http.Error(w, err.Error(), http.StatusForbidden)
		default:
			http.Error(w, err.Error(), http.StatusBadGateway)
		}
		return
	}
	w.WriteHeader(http.StatusCreated)
}
