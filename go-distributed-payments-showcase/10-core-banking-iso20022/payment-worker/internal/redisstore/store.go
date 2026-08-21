// Package redisstore is the Redis-backed idempotency store for payment-worker.
package redisstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultTTL = 48 * time.Hour

type Store struct {
	rdb *redis.Client
	ttl time.Duration
}

func New(rdb *redis.Client, ttl time.Duration) *Store {
	if ttl == 0 {
		ttl = defaultTTL
	}
	return &Store{rdb: rdb, ttl: ttl}
}

func (s *Store) Acquire(ctx context.Context, key string) (bool, error) {
	ok, err := s.rdb.SetNX(ctx, key, "STARTED", s.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("redis acquire: %w", err)
	}
	return ok, nil
}

func (s *Store) Complete(ctx context.Context, key string) error {
	return s.rdb.Set(ctx, key, "COMPLETED", s.ttl).Err()
}

func (s *Store) IsCompleted(ctx context.Context, key string) (bool, error) {
	val, err := s.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("redis get: %w", err)
	}
	return val == "COMPLETED", nil
}

func (s *Store) Release(ctx context.Context, key string) error {
	return s.rdb.Del(ctx, key).Err()
}
