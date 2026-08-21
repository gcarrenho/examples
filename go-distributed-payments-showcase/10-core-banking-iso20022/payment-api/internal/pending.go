package api

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const pendingIndexKey = "payment-pending" // Redis ZSET: member=uetr, score=request unix timestamp

// PendingIndex tracks in-flight orders so TimeoutReaper can find ones that
// never got a result within the business timeout window.
type PendingIndex struct{ rdb *redis.Client }

func NewPendingIndex(rdb *redis.Client) *PendingIndex { return &PendingIndex{rdb: rdb} }

// Add records uetr as pending at the current time. Called when payment-api
// publishes the PaymentOrderInitiated event, before returning 202 to the client.
func (p *PendingIndex) Add(ctx context.Context, uetr string) error {
	return p.rdb.ZAdd(ctx, pendingIndexKey, redis.Z{Score: float64(time.Now().Unix()), Member: uetr}).Err()
}

// Remove clears uetr from the pending index — called once a result (real or
// synthetic timeout) has been recorded, so the reaper never revisits it.
func (p *PendingIndex) Remove(ctx context.Context, uetr string) error {
	return p.rdb.ZRem(ctx, pendingIndexKey, uetr).Err()
}

// Stale returns UETRs added more than olderThan ago — candidates for the reaper
// to mark as timed out.
func (p *PendingIndex) Stale(ctx context.Context, olderThan time.Duration) ([]string, error) {
	maxScore := strconv.FormatInt(time.Now().Add(-olderThan).Unix(), 10)
	uetrs, err := p.rdb.ZRangeByScore(ctx, pendingIndexKey, &redis.ZRangeBy{Min: "-inf", Max: maxScore}).Result()
	if err != nil {
		return nil, fmt.Errorf("pending index: stale scan: %w", err)
	}
	return uetrs, nil
}

const callbackKeyPrefix = "payment-callback:"

// CallbackStore holds the optional webhook URL a client provides at request time.
// Kept separate from the Kafka event contract — delivery mechanism is an API concern,
// not part of the banking domain event.
type CallbackStore struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewCallbackStore(rdb *redis.Client, ttl time.Duration) *CallbackStore {
	return &CallbackStore{rdb: rdb, ttl: ttl}
}

func (c *CallbackStore) Set(ctx context.Context, uetr, url string) error {
	if url == "" {
		return nil
	}
	return c.rdb.Set(ctx, callbackKeyPrefix+uetr, url, c.ttl).Err()
}

// Get returns the callback URL for uetr, or found=false if the client didn't provide one.
func (c *CallbackStore) Get(ctx context.Context, uetr string) (url string, found bool, err error) {
	url, err = c.rdb.Get(ctx, callbackKeyPrefix+uetr).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("callback store: get: %w", err)
	}
	return url, true, nil
}
