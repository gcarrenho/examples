package payment

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

type paymentRequest struct {
IdempotencyKey string `json:"idempotency_key"`
AccountID      string `json:"account_id"`
AmountCents    int64  `json:"amount_cents"`
}

// Handler is the inbound HTTP adapter for payment-svc.
type Handler struct{ svc ChargeService }

func NewHandler(svc ChargeService) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// AUTH TODO: payment-svc is only called by other internal services (orders-svc,
	// fraud-svc), never directly by an end user — this needs service-to-service auth,
	// not end-user auth:
	//   - mTLS between services (mesh-enforced, e.g. Istio/Linkerd) is the strongest option
	//   - Alternatively, a short-lived service JWT (client_credentials grant) per caller
	//   - Reject any request without a valid service identity before touching ChargeService
	//
	// mux.Handle("POST /payments/charge",  requireServiceAuth(http.HandlerFunc(h.charge)))
	mux.HandleFunc("POST /payments/charge",  h.charge)
	mux.HandleFunc("POST /payments/refund",  h.refund)
	mux.HandleFunc("POST /payments/reserve", h.reserve)
}

// requireServiceAuth would validate the calling service's mTLS cert or service JWT
// and confirm it's an allow-listed internal caller (orders-svc, fraud-svc) before
// any handler runs. Rejects with 401/403 — never reaches ChargeService.
//
// func requireServiceAuth(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		callerCert := r.TLS.PeerCertificates[0] // populated by mTLS termination
// 		if !isAllowedCaller(callerCert.Subject.CommonName) {
// 			http.Error(w, "forbidden", http.StatusForbidden)
// 			return
// 		}
// 		next.ServeHTTP(w, r)
// 	})
// }

type operationFn func(ctx context.Context, key, accountID string, amountCents int64) error

func (h *Handler) charge(w http.ResponseWriter, r *http.Request) { h.handle(w, r, h.svc.Charge) }
func (h *Handler) refund(w http.ResponseWriter, r *http.Request) { h.handle(w, r, h.svc.Refund) }

func (h *Handler) handle(w http.ResponseWriter, r *http.Request, fn operationFn) {
var req paymentRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
http.Error(w, "invalid body", http.StatusBadRequest)
return
}
if err := fn(r.Context(), req.IdempotencyKey, req.AccountID, req.AmountCents); err != nil {
h.writeError(w, err)
return
}
w.WriteHeader(http.StatusCreated)
}

func (h *Handler) reserve(w http.ResponseWriter, r *http.Request) {
var req paymentRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
http.Error(w, "invalid body", http.StatusBadRequest)
return
}
reservationID, err := h.svc.Reserve(r.Context(), req.IdempotencyKey, req.AccountID, req.AmountCents)
if err != nil {
h.writeError(w, err)
return
}
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusCreated)
_ = json.NewEncoder(w).Encode(map[string]string{"reservation_id": reservationID})
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
switch {
case errors.Is(err, ErrAlreadyProcessed):
w.WriteHeader(http.StatusOK)
case errors.Is(err, ErrDuplicateRequest):
http.Error(w, err.Error(), http.StatusConflict)
case errors.Is(err, ErrInsufficientFunds):
http.Error(w, err.Error(), http.StatusUnprocessableEntity)
default:
http.Error(w, "internal server error", http.StatusInternalServerError)
}
}
