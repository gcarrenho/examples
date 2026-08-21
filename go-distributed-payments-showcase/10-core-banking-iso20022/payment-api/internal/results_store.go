package api

import (
	"context"
	"sync"

	banking "github.com/examples/banking-core"
)

// ResultsStore caches the latest PaymentOrderResult per UETR so GET /payments/{uetr}
// can answer without querying payment-worker or the ledger directly.
//
// Production note: an in-memory map only works for a single payment-api replica.
// With multiple replicas behind a load balancer, back this with Redis or Postgres
// so any replica can answer any UETR regardless of which one placed the order.
type ResultsStore struct {
	mu   sync.RWMutex
	data map[string]banking.PaymentOrderResult
}

func NewResultsStore() *ResultsStore {
	return &ResultsStore{data: make(map[string]banking.PaymentOrderResult)}
}

// HandleResult satisfies kafka.resultHandler — called for every consumed PaymentOrderResult.
func (s *ResultsStore) HandleResult(_ context.Context, result banking.PaymentOrderResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[result.UETR] = result
}

func (s *ResultsStore) Get(uetr string) (banking.PaymentOrderResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.data[uetr]
	return r, ok
}
