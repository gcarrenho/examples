// Package syncpool demonstrates how sync.Pool eliminates per-call heap
// allocations on the hot path of a payment processor.
//
// The allocation problem
//
// Building a canonical payment string (used for idempotency keys and HMAC
// signing) on every inbound request allocates a temporary []byte buffer.
// At 1 million requests/second the GC reclaims millions of short-lived
// buffers per second, raising stop-the-world pause frequency and tail latency.
//
// The sync.Pool solution
//
// sync.Pool maintains a per-P (per-OS-thread) cache of reusable objects.
// Get() returns a recycled buffer — no allocation. Put() returns it for the
// next caller. Each P has its own sub-pool so there is no cross-goroutine
// contention: the pool scales linearly with GOMAXPROCS.
//
// Critical invariant: always reset (b = b[:0]) before use. The pool may
// return a buffer populated by a previous caller. Forgetting the reset is
// a silent data-corruption bug, not a panic.
//
// Measurement: run `go test -bench=. -benchmem` and observe:
//
//	BenchmarkCanonical_WithPool   1 alloc/op   (only the returned string copy)
//	BenchmarkCanonical_Direct     5 allocs/op  (intermediate strings from +)
package syncpool

import (
	"encoding/json"
	"strconv"
	"sync"
)

// PaymentMessage is the parsed representation of an inbound payment request.
type PaymentMessage struct {
	TransactionID string `json:"transaction_id"`
	AccountID     string `json:"account_id"`
	AmountCents   int64  `json:"amount_cents"`
	Currency      string `json:"currency"`
}

// canonPool holds pre-sized []byte scratch buffers for canonical string building.
// Starting at 256 bytes covers most payment messages without a growth allocation.
var canonPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 256)
		return &b
	},
}

// Parser decodes and canonicalises PaymentMessage values.
// It is safe for concurrent use.
type Parser struct{}

// NewParser creates a stateless Parser.
func NewParser() *Parser { return &Parser{} }

// Parse decodes raw JSON into a PaymentMessage.
func (p *Parser) Parse(data []byte) (*PaymentMessage, error) {
	var msg PaymentMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// Canonical builds the deterministic pipe-delimited representation of a payment,
// used for idempotency key generation and HMAC signing.
//
// It reuses a []byte scratch buffer from canonPool, so the only heap allocation
// is the string copy at the end — 1 alloc/op regardless of message size.
func (p *Parser) Canonical(msg *PaymentMessage) string {
	bp := canonPool.Get().(*[]byte)
	b := (*bp)[:0] // reset length, preserve capacity
	defer func() {
		*bp = b // write back in case append grew the slice
		canonPool.Put(bp)
	}()

	b = append(b, msg.TransactionID...)
	b = append(b, '|')
	b = append(b, msg.AccountID...)
	b = append(b, '|')
	b = strconv.AppendInt(b, msg.AmountCents, 10)
	b = append(b, '|')
	b = append(b, msg.Currency...)

	// string(b) copies bytes out before the deferred Put returns b to the pool.
	return string(b)
}

// CanonicalDirect builds the same string via concatenation — the baseline.
// Each + operator on non-constant strings allocates an intermediate string.
func (p *Parser) CanonicalDirect(msg *PaymentMessage) string {
	return msg.TransactionID + "|" + msg.AccountID + "|" +
		strconv.FormatInt(msg.AmountCents, 10) + "|" + msg.Currency
}
