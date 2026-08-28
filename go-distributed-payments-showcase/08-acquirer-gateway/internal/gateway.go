// Package gateway implements the acquirer routing service.
// AuthRequest, AuthResponse, errors, Gateway interface, Router, and CircuitBreaker
// all live here — one package, one bounded context, no artificial layers.
package gateway

import (
	"context"
	"errors"
	"sync/atomic"
)

// ── Domain types ──────────────────────────────────────────────────────────────

// AuthRequest is the canonical authorization request.
// All acquirer adapters translate from this type — never the reverse.
type AuthRequest struct {
	IdempotencyKey string
	AmountCents    int64
	Currency       string // ISO 4217: "ARS", "BRL", "GBP", "EUR"
	CardBIN        string
	CardCountry    string // ISO 3166-1 alpha-2
	MerchantID     string
}

// AuthResponse is the canonical authorization result.
type AuthResponse struct {
	Approved   bool
	AuthCode   string
	NetworkRef string
}

var (
	ErrDeclined            = errors.New("acquirer: card declined")
	ErrAcquirerUnavailable = errors.New("acquirer: circuit open — too many recent failures")
	ErrTimeout             = errors.New("acquirer: authorization timed out")
)

// ── Gateway interface (exported — intentional extension point) ────────────────

// Gateway is the secondary port each acquirer adapter must satisfy.
// Exported because cmd/main.go composes adapters into the Router.
//
//go:generate go run go.uber.org/mock/mockgen -source gateway.go -destination mocks_test.go -package gateway
type Gateway interface {
	Authorize(ctx context.Context, req AuthRequest) (AuthResponse, error)
}

// ── Router ────────────────────────────────────────────────────────────────────

// Router selects the acquirer by card country using the Strategy pattern.
type Router struct {
	routes   map[string]Gateway
	fallback Gateway
}

func NewRouter(fallback Gateway, routes map[string]Gateway) *Router {
	return &Router{routes: routes, fallback: fallback}
}

func (r *Router) Authorize(ctx context.Context, req AuthRequest) (AuthResponse, error) {
	return r.selectGateway(req.CardCountry).Authorize(ctx, req)
}

func (r *Router) selectGateway(country string) Gateway {
	if gw, ok := r.routes[country]; ok {
		return gw
	}
	return r.fallback
}

// ── CircuitBreaker ────────────────────────────────────────────────────────────

// WithCircuitBreaker wraps a Gateway. Each acquirer gets its own breaker —
// a Prisma outage does not affect Adyen processing.
func WithCircuitBreaker(gw Gateway, threshold int64) Gateway {
	return &circuitBreaker{gw: gw, threshold: threshold}
}

type circuitBreaker struct {
	gw                Gateway
	threshold         int64
	consecutiveErrors atomic.Int64
	open              atomic.Bool
}

func (cb *circuitBreaker) Authorize(ctx context.Context, req AuthRequest) (AuthResponse, error) {
	if cb.open.Load() {
		return AuthResponse{}, ErrAcquirerUnavailable
	}
	resp, err := cb.gw.Authorize(ctx, req)
	if err != nil {
		if cb.consecutiveErrors.Add(1) >= cb.threshold {
			cb.open.Store(true)
		}
	} else {
		cb.consecutiveErrors.Store(0)
	}
	return resp, err
}
