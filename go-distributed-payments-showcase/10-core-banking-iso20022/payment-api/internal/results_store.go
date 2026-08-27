package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	banking "github.com/examples/banking-core"
)

// ResultsStore persists the latest PaymentOrderResult per UETR in Redis so
// GET /payments/{uetr} answers correctly regardless of which payment-api
// replica handles the request — an in-memory map only works for a single replica.
type ResultsStore struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewResultsStore(rdb *redis.Client, ttl time.Duration) *ResultsStore {
	return &ResultsStore{rdb: rdb, ttl: ttl}
}

func resultKey(uetr string) string { return "payment-result:" + uetr }

// Save persists a result. Used directly by ResultProcessor and TimeoutReaper.
func (s *ResultsStore) Save(ctx context.Context, result banking.PaymentOrderResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("results store: marshal: %w", err)
	}
	return s.rdb.Set(ctx, resultKey(result.UETR), data, s.ttl).Err()
}

// Get returns the result for uetr, or found=false if the worker hasn't published yet.
func (s *ResultsStore) Get(ctx context.Context, uetr string) (result banking.PaymentOrderResult, found bool, err error) {
	data, err := s.rdb.Get(ctx, resultKey(uetr)).Bytes()
	if errors.Is(err, redis.Nil) {
		return banking.PaymentOrderResult{}, false, nil
	}
	if err != nil {
		return banking.PaymentOrderResult{}, false, fmt.Errorf("results store: get: %w", err)
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return banking.PaymentOrderResult{}, false, fmt.Errorf("results store: unmarshal: %w", err)
	}
	return result, true, nil
}
