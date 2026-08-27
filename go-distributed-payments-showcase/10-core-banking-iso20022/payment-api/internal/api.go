// Package api is the HTTP inbound adapter for the payment-api microservice.
// Receives POST /payments, validates, publishes PaymentOrderInitiated to Kafka, returns 202.
// No banking domain logic lives here — it's a thin translation layer.
package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	banking "github.com/examples/banking-core"
)

// publisher is the handler's private contract for sending events.
// Defined here by the consumer (api package), not by banking-core.
type publisher interface {
	Publish(ctx context.Context, event banking.PaymentOrderInitiated) error
}

// Handler is the HTTP inbound adapter.
type Handler struct {
	pub       publisher
	results   *ResultsStore
	pending   *PendingIndex
	callbacks *CallbackStore
}

func NewHandler(pub publisher, results *ResultsStore, pending *PendingIndex, callbacks *CallbackStore) *Handler {
	return &Handler{pub: pub, results: results, pending: pending, callbacks: callbacks}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// AUTH TODO: this is the client-facing edge (merchant/bank client calls this API
	// directly), so it needs strong caller authentication before any processing:
	//   - OAuth2 client_credentials or mTLS client certs (bank-to-bank integrations)
	//   - Validate the token/cert BEFORE decoding the body — reject early, don't waste CPU
	//   - Extract the caller's identity and store it in ctx for audit logging
	//
	// mux.Handle("POST /payments", requireAuth(http.HandlerFunc(h.initiate)))
	mux.HandleFunc("POST /payments", h.initiate)
	mux.HandleFunc("GET /payments/{uetr}", h.getStatus)
}

// requireAuth would validate a bearer token (JWT) or mTLS client cert and reject
// unauthenticated requests with 401 before h.initiate ever runs.
//
// func requireAuth(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		token := r.Header.Get("Authorization")
// 		claims, err := verifyJWT(token) // validate signature, expiry, issuer
// 		if err != nil {
// 			http.Error(w, "unauthorized", http.StatusUnauthorized)
// 			return
// 		}
// 		ctx := context.WithValue(r.Context(), callerIDKey, claims.Subject)
// 		next.ServeHTTP(w, r.WithContext(ctx))
// 	})
// }

// InitiateRequest — what the CALLER provides.
type InitiateRequest struct {
	UETR         string `json:"uetr"`
	EndToEndID   string `json:"end_to_end_id"`
	AmountCents  int64  `json:"amount_cents"`
	Currency     string `json:"currency"`
	DebtorIBAN   string `json:"debtor_iban"`
	DebtorBIC    string `json:"debtor_bic"`
	DebtorName   string `json:"debtor_name"`
	CreditorIBAN string `json:"creditor_iban"`
	CreditorBIC  string `json:"creditor_bic"`
	CreditorName string `json:"creditor_name"`
	Rail         string `json:"rail"` // "SEPA" | "SWIFT" | "FAKE" (test rail)
	// CallbackURL is optional. If set, payment-api POSTs the final result here
	// instead of requiring the client to poll GET /payments/{uetr}.
	CallbackURL string `json:"callback_url,omitempty"`
}

func (h *Handler) initiate(w http.ResponseWriter, r *http.Request) {
	var req InitiateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if req.UETR == "" || req.DebtorIBAN == "" || req.CreditorIBAN == "" {
		http.Error(w, "uetr, debtor_iban, creditor_iban required", http.StatusBadRequest)
		return
	}

	instrID := req.UETR
	if len(instrID) > 16 {
		instrID = instrID[:16]
	}

	event := banking.PaymentOrderInitiated{
		EventID:    req.UETR,
		OccurredAt: time.Now().UTC(),
		Order: banking.PaymentOrder{
			ID: req.UETR, UETR: req.UETR, InstrID: instrID, EndToEndID: req.EndToEndID,
			Amount:   banking.Money{AmountCents: req.AmountCents, Currency: req.Currency},
			Debtor:   banking.Party{Name: req.DebtorName, IBAN: req.DebtorIBAN, BIC: req.DebtorBIC},
			Creditor: banking.Party{Name: req.CreditorName, IBAN: req.CreditorIBAN, BIC: req.CreditorBIC},
			Rail:     banking.RailName(req.Rail),
		},
	}

	if err := h.pub.Publish(r.Context(), event); err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			http.Error(w, "upstream timeout", http.StatusGatewayTimeout)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	// Track for the timeout reaper and register the optional webhook — best-effort:
	// a failure here does not block the response, it only means GET /payments/{uetr}
	// remains the client's only path to the result (no reaper cleanup, no webhook).
	if err := h.pending.Add(r.Context(), req.UETR); err != nil {
		slog.Default().ErrorContext(r.Context(), "failed to track pending order", slog.String("uetr", req.UETR), slog.String("err", err.Error()))
	}
	if err := h.callbacks.Set(r.Context(), req.UETR, req.CallbackURL); err != nil {
		slog.Default().ErrorContext(r.Context(), "failed to store callback url", slog.String("uetr", req.UETR), slog.String("err", err.Error()))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"uetr":   req.UETR,
		"status": "PDNG",
	})
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// getStatus answers GET /payments/{uetr} — the mechanism the client uses to learn
// the final outcome of a payment that was accepted asynchronously via Kafka.
// Returns 202/PDNG while the worker hasn't published a result yet.
func (h *Handler) getStatus(w http.ResponseWriter, r *http.Request) {
	uetr := r.PathValue("uetr")
	result, found, err := h.results.Get(r.Context(), uetr)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !found {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{"uetr": uetr, "status": "PDNG"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"uetr": result.UETR, "status": string(result.Status), "reason": result.Reason,
	})
}
