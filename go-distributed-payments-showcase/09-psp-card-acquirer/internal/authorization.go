// Package authorization implements card payment authorization for a PSP.
//
// All types, business logic, and interfaces for the authorization capability
// live in this package. There is no separate "domain", "ports", or "service"
// layer: those are architectural concepts, not Go packages.
//
// Sub-packages (visanet, mastercard, amex) exist only because each has distinct
// protocol dependencies (ISO 8583 field mappings, proprietary REST APIs).
// They are adapters: they translate authorization.CardTransaction ↔ wire format.
package authorization

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

// ── Types ─────────────────────────────────────────────────────────────────────

type CardBrand string

const (
	BrandVisa       CardBrand = "VISA"
	BrandMastercard CardBrand = "MASTERCARD"
	BrandAmex       CardBrand = "AMEX" // closed-loop: AmEx is issuer AND network
)

// BrandFromBIN derives the card brand from the BIN (first 6-8 digits of the PAN).
//
//	Visa:        4xxxxxx
//	Mastercard:  51-55xxxxxx | 2221-2720xxxxxx (2-series added 2017)
//	AmEx:        34xxxxxx | 37xxxxxx
func BrandFromBIN(bin string) CardBrand {
	if len(bin) < 2 {
		return BrandVisa
	}
	switch {
	case bin[0] == '4':
		return BrandVisa
	case bin[:2] >= "51" && bin[:2] <= "55":
		return BrandMastercard
	case len(bin) >= 4 && bin[:4] >= "2221" && bin[:4] <= "2720":
		return BrandMastercard
	case bin[:2] == "34" || bin[:2] == "37":
		return BrandAmex
	default:
		return BrandVisa
	}
}

type AuthorizationStatus string

const (
	StatusPending  AuthorizationStatus = "PENDING"
	StatusApproved AuthorizationStatus = "APPROVED"
	StatusDeclined AuthorizationStatus = "DECLINED"
	StatusReversed AuthorizationStatus = "REVERSED"
)

// CardTransaction is the bounded context's single aggregate type.
// Every operation in this package speaks CardTransaction, never wire protocol types.
type CardTransaction struct {
	ID             string
	IdempotencyKey string
	AmountCents    int64
	CurrencyCode   string // ISO 4217 numeric: "840"=USD "032"=ARS "978"=EUR
	CardBIN        string
	CardBrand      CardBrand
	CardLast4      string
	CardCountry    string // ISO 3166-1 alpha-2
	MerchantID     string
	TerminalID     string
	Status         AuthorizationStatus
	NetworkRef     string // issuer-assigned; required for reversals
	AuthCode       string // 6 chars, present only on approval
}

var (
	ErrDeclined           = errors.New("psp: card declined by issuer")
	ErrExpiredCard        = errors.New("psp: card expired")
	ErrInsufficientFunds  = errors.New("psp: insufficient funds")
	ErrNetworkUnavailable = errors.New("psp: card network unavailable — retry")
	ErrDuplicateSTAN      = errors.New("psp: duplicate STAN — already processed")
)

// ── Secondary ports ───────────────────────────────────────────────────────────
// Unexported: cmd/main.go passes adapter values structurally; can never
// inject them into the wrong place because the type names are inaccessible.

//go:generate go run go.uber.org/mock/mockgen -source authorization.go -destination mocks_test.go -package authorization

type network interface {
	Authorize(ctx context.Context, txn CardTransaction) (CardTransaction, error)
}

type idempotencyStore interface {
	Acquire(ctx context.Context, key string) (bool, error)
	Complete(ctx context.Context, key, authCode, networkRef string) error
	GetCompleted(ctx context.Context, key string) (authCode, networkRef string, found bool, err error)
	Release(ctx context.Context, key string) error
}

// ── Primary port ──────────────────────────────────────────────────────────────

// Authorizer is the exported contract. The handler and cmd/main.go depend on this.
type Authorizer interface {
	Authorize(ctx context.Context, txn CardTransaction) (CardTransaction, error)
}

// ── Service ───────────────────────────────────────────────────────────────────

var _ Authorizer = (*Service)(nil)

// Service orchestrates idempotency filtering and network routing.
type Service struct {
	network     network
	idempotency idempotencyStore
}

func NewService(network network, idempotency idempotencyStore) *Service {
	return &Service{network: network, idempotency: idempotency}
}

func (s *Service) Authorize(ctx context.Context, txn CardTransaction) (CardTransaction, error) {
	acquired, err := s.idempotency.Acquire(ctx, txn.IdempotencyKey)
	if err != nil {
		return CardTransaction{}, fmt.Errorf("idempotency: %w", err)
	}
	if !acquired {
		authCode, networkRef, _, _ := s.idempotency.GetCompleted(ctx, txn.IdempotencyKey)
		txn.Status, txn.AuthCode, txn.NetworkRef = StatusApproved, authCode, networkRef
		return txn, nil
	}

	result, err := s.network.Authorize(ctx, txn)
	if err != nil {
		_ = s.idempotency.Release(ctx, txn.IdempotencyKey)
		return CardTransaction{}, err
	}

	_ = s.idempotency.Complete(ctx, txn.IdempotencyKey, result.AuthCode, result.NetworkRef)
	return result, nil
}

// ── NetworkRouter ─────────────────────────────────────────────────────────────

// NetworkRouter routes to the correct card network based on card brand.
type NetworkRouter struct {
	byBrand  map[CardBrand]network
	fallback network
}

func NewNetworkRouter(fallback network) *NetworkRouter {
	return &NetworkRouter{byBrand: make(map[CardBrand]network), fallback: fallback}
}

func (r *NetworkRouter) Register(brand CardBrand, n network) *NetworkRouter {
	r.byBrand[brand] = n
	return r
}

func (r *NetworkRouter) Authorize(ctx context.Context, txn CardTransaction) (CardTransaction, error) {
	gw, ok := r.byBrand[txn.CardBrand]
	if !ok {
		gw = r.fallback
	}
	resp, err := gw.Authorize(ctx, txn)
	if err != nil {
		return CardTransaction{}, fmt.Errorf("network[%s]: %w", txn.CardBrand, err)
	}
	return resp, nil
}

// ── CircuitBreaker ────────────────────────────────────────────────────────────

// WithCircuitBreaker wraps a network adapter. Opens after `threshold` consecutive
// failures; returns ErrNetworkUnavailable without hitting the network while open.
func WithCircuitBreaker(n network, threshold int64) network {
	return &circuitBreaker{n: n, threshold: threshold}
}

type circuitBreaker struct {
	n                 network
	threshold         int64
	consecutiveErrors atomic.Int64
	open              atomic.Bool
}

func (cb *circuitBreaker) Authorize(ctx context.Context, txn CardTransaction) (CardTransaction, error) {
	if cb.open.Load() {
		return CardTransaction{}, ErrNetworkUnavailable
	}
	resp, err := cb.n.Authorize(ctx, txn)
	if err != nil {
		if cb.consecutiveErrors.Add(1) >= cb.threshold {
			cb.open.Store(true)
		}
	} else {
		cb.consecutiveErrors.Store(0)
	}
	return resp, err
}

// ── In-memory idempotency store (dev / tests) ─────────────────────────────────

type memoryIdempotency struct {
	mu   sync.Mutex
	data map[string]struct{ authCode, networkRef string }
}

func NewMemoryIdempotency() idempotencyStore {
	return &memoryIdempotency{data: make(map[string]struct{ authCode, networkRef string })}
}

func (m *memoryIdempotency) Acquire(_ context.Context, key string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[key]; ok {
		return false, nil
	}
	m.data[key] = struct{ authCode, networkRef string }{}
	return true, nil
}

func (m *memoryIdempotency) Complete(_ context.Context, key, authCode, networkRef string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = struct{ authCode, networkRef string }{authCode, networkRef}
	return nil
}

func (m *memoryIdempotency) GetCompleted(_ context.Context, key string) (string, string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[key]
	return v.authCode, v.networkRef, ok, nil
}

func (m *memoryIdempotency) Release(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}
