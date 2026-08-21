package orders

import (
	"context"
	"encoding/json"
	"net/http"
)

type orderPlacer interface {
	PlaceOrder(ctx context.Context, accountID string, amountCents int64) (string, error)
}

type Handler struct{ svc orderPlacer }

func NewHandler(svc orderPlacer) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// AUTH TODO: orders-svc is the client-facing edge (end users / merchant frontends
	// call this directly) — needs end-user authentication before placing any order:
	//   - OAuth2/OIDC bearer JWT validated against the identity provider's JWKS
	//   - Extract the authenticated user/account ID from claims — never trust
	//     account_id from the request body alone, it must match the token's subject
	//
	// mux.Handle("POST /orders", requireAuth(http.HandlerFunc(h.placeOrder)))
	mux.HandleFunc("POST /orders", h.placeOrder)
}

type placeOrderRequest struct {
	AccountID   string `json:"account_id"`
	AmountCents int64  `json:"amount_cents"`
}

type placeOrderResponse struct {
	OrderID string `json:"order_id"`
}

func (h *Handler) placeOrder(w http.ResponseWriter, r *http.Request) {
	var req placeOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	orderID, err := h.svc.PlaceOrder(r.Context(), req.AccountID, req.AmountCents)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(placeOrderResponse{OrderID: orderID})
}
