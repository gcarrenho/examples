package idempotency

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// storeEntry holds a value and an optional expiry (zero = no expiry).
type storeEntry struct {
	value  string
	expiry time.Time
}

func expiryFrom(ttl time.Duration) time.Time {
	if ttl <= 0 {
		return time.Time{}
	}
	return time.Now().Add(ttl)
}

// inMemoryStore is a thread-safe in-memory StateStore for unit tests.
// Its SetNX faithfully replicates Redis SETNX semantics: the existence check
// and the write happen under a single mutex lock, making the operation truly
// atomic — no goroutine can observe the key in a half-written state.
// TTL expiry is fully implemented so TestFilter_PodCrashRecoveryViaTTL can run
// in milliseconds without a real Redis instance.
type inMemoryStore struct {
	mu   sync.Mutex
	data map[string]storeEntry
}

func newInMemoryStore() *inMemoryStore {
	return &inMemoryStore{data: make(map[string]storeEntry)}
}

func (s *inMemoryStore) SetNX(_ context.Context, key, value string, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.data[key]; ok && (e.expiry.IsZero() || time.Now().Before(e.expiry)) {
		return false, nil
	}
	s.data[key] = storeEntry{value: value, expiry: expiryFrom(ttl)}
	return true, nil
}

func (s *inMemoryStore) Get(_ context.Context, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data[key]
	if !ok || (!e.expiry.IsZero() && time.Now().After(e.expiry)) {
		return "", errors.New("key not found or expired")
	}
	return e.value, nil
}

func (s *inMemoryStore) Set(_ context.Context, key, value string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = storeEntry{value: value, expiry: expiryFrom(ttl)}
	return nil
}

func (s *inMemoryStore) Del(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

// TestFilter_Acquire covers every state-machine transition via a table.
// Each subtest is isolated (own filter + store) and runs in parallel.
func TestFilter_Acquire(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	const key = "payment:txn"

	cases := []struct {
		name  string
		setup func(*Filter)
		want  error
	}{
		{
			name:  "fresh key succeeds",
			setup: func(*Filter) {},
			want:  nil,
		},
		{
			name:  "STARTED blocks concurrent callers with ErrDuplicate",
			setup: func(f *Filter) { _ = f.Acquire(ctx, key) },
			want:  ErrDuplicate,
		},
		{
			name: "COMPLETED rejects re-processing with ErrAlreadyCompleted",
			setup: func(f *Filter) {
				_ = f.Acquire(ctx, key)
				_ = f.Complete(ctx, key)
			},
			want: ErrAlreadyCompleted,
		},
		{
			name: "Release after STARTED unblocks retry",
			setup: func(f *Filter) {
				_ = f.Acquire(ctx, key)
				_ = f.Release(ctx, key)
			},
			want: nil,
		},
		{
			name: "Release after COMPLETED reopens the key",
			setup: func(f *Filter) {
				_ = f.Acquire(ctx, key)
				_ = f.Complete(ctx, key)
				_ = f.Release(ctx, key)
			},
			want: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			filter := NewFilter(newInMemoryStore(), time.Hour)
			tc.setup(filter)
			err := filter.Acquire(ctx, key)
			if !errors.Is(err, tc.want) {
				t.Errorf("Acquire() = %v, want %v", err, tc.want)
			}
		})
	}
}

// TestFilter_ExactlyOnce is the core scientific proof:
// 1000 goroutines race on the same key simultaneously and exactly one wins.
func TestFilter_ExactlyOnce(t *testing.T) {
	t.Parallel()
	const goroutines = 1000

	filter := NewFilter(newInMemoryStore(), time.Hour)
	ctx := context.Background()

	var (
		winners atomic.Int64
		wg      sync.WaitGroup
		start   = make(chan struct{})
	)

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if err := filter.Acquire(ctx, "payment:txn-abc123"); err == nil {
				winners.Add(1)
			}
		}()
	}

	close(start)
	wg.Wait()

	if got := winners.Load(); got != 1 {
		t.Errorf("exactly 1 winner expected across %d goroutines; got %d", goroutines, got)
	}
}

// TestFilter_PodCrashRecoveryViaTTL proves that a STARTED key left by a crashed
// pod expires and allows a subsequent pod to retry.
// TTL sizing rule: 2–3× the maximum expected processing duration.
func TestFilter_PodCrashRecoveryViaTTL(t *testing.T) {
	t.Parallel()
	const ttl = 50 * time.Millisecond

	filter := NewFilter(newInMemoryStore(), ttl)
	ctx := context.Background()
	key := "payment:txn-pod-crash"

	if err := filter.Acquire(ctx, key); err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	// ~~~ pod crashes here; Complete/Release never called ~~~

	if err := filter.Acquire(ctx, key); !errors.Is(err, ErrDuplicate) {
		t.Errorf("pre-expiry: want ErrDuplicate, got %v", err)
	}

	time.Sleep(ttl + 20*time.Millisecond)

	if err := filter.Acquire(ctx, key); err != nil {
		t.Errorf("post-expiry: retry must succeed, got %v", err)
	}
}
