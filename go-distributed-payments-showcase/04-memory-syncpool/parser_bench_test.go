package syncpool

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

var sampleMsg = &PaymentMessage{
	TransactionID: "txn-abc123def456",
	AccountID:     "acc-9876543210",
	AmountCents:   9999,
	Currency:      "USD",
}

var samplePayload = mustMarshal(sampleMsg)

func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// BenchmarkCanonical_WithPool measures canonical string building with a pooled
// []byte scratch buffer.
//
// Expected: 1 alloc/op — only the string(b) copy at the end; the []byte is recycled.
//
// Run: go test -bench=BenchmarkCanonical_WithPool -benchmem
func BenchmarkCanonical_WithPool(b *testing.B) {
	p := NewParser()
	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		s := p.Canonical(sampleMsg)
		_ = s
	}
}

// BenchmarkCanonical_Direct measures the baseline: string concatenation with +.
//
// Expected: 5 allocs/op — one per intermediate string produced by each + operator.
//
// Run: go test -bench=BenchmarkCanonical_Direct -benchmem
func BenchmarkCanonical_Direct(b *testing.B) {
	p := NewParser()
	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		s := p.CanonicalDirect(sampleMsg)
		_ = s
	}
}

// BenchmarkCanonical_Parallel_WithPool validates Pool under goroutine contention.
// Each P has its own local sub-pool, so there is no cross-goroutine lock contention.
// Throughput should scale linearly with CPU count.
//
// Run: go test -bench=BenchmarkCanonical_Parallel_WithPool -benchmem -cpu=1,2,4,8
func BenchmarkCanonical_Parallel_WithPool(b *testing.B) {
	p := NewParser()
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s := p.Canonical(sampleMsg)
			_ = s
		}
	})
}

// TestCanonical verifies pooled and direct canonicalisation produce identical
// output across a range of input shapes. A missing b[:0] reset would corrupt
// the pooled result for every call after the first.
func TestCanonical(t *testing.T) {
	t.Parallel()
	p := NewParser()
	cases := []struct {
		name string
		msg  *PaymentMessage
	}{
		{
			name: "standard USD payment",
			msg:  &PaymentMessage{TransactionID: "txn-001", AccountID: "acc-001", AmountCents: 9999, Currency: "USD"},
		},
		{
			name: "zero amount (auth hold)",
			msg:  &PaymentMessage{TransactionID: "txn-002", AccountID: "acc-002", AmountCents: 0, Currency: "USD"},
		},
		{
			name: "negative amount (refund)",
			msg:  &PaymentMessage{TransactionID: "txn-003", AccountID: "acc-003", AmountCents: -500, Currency: "EUR"},
		},
		{
			name: "non-USD currency",
			msg:  &PaymentMessage{TransactionID: "txn-004", AccountID: "acc-004", AmountCents: 100, Currency: "GBP"},
		},
		{
			name: "long IDs stress test",
			msg: &PaymentMessage{
				TransactionID: "txn-" + strings.Repeat("a", 60),
				AccountID:     "acc-" + strings.Repeat("b", 28),
				AmountCents:   1,
				Currency:      "USD",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			pooled := p.Canonical(tc.msg)
			direct := p.CanonicalDirect(tc.msg)
			if pooled != direct {
				t.Errorf("mismatch:\n  pooled: %q\n  direct: %q", pooled, direct)
			}
		})
	}
}

// TestCanonical_ConcurrentIsolation validates the pool does not leak state
// between 200 concurrent goroutines using different messages simultaneously.
func TestCanonical_ConcurrentIsolation(t *testing.T) {
	t.Parallel()
	p := NewParser()
	msgs := []*PaymentMessage{
		{TransactionID: "txn-a", AccountID: "acc-a", AmountCents: 111, Currency: "USD"},
		{TransactionID: "txn-b", AccountID: "acc-b", AmountCents: 222, Currency: "EUR"},
		{TransactionID: "txn-c", AccountID: "acc-c", AmountCents: 333, Currency: "GBP"},
	}
	var wg sync.WaitGroup
	for range 200 {
		for _, msg := range msgs {
			wg.Add(1)
			go func(m *PaymentMessage) {
				defer wg.Done()
				if got, want := p.Canonical(m), p.CanonicalDirect(m); got != want {
					t.Errorf("concurrent mismatch for %q:\n  got:  %q\n  want: %q", m.TransactionID, got, want)
				}
			}(msg)
		}
	}
	wg.Wait()
}
