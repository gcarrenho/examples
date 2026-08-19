> 🌐 [Español](README-es.md)

# Case 04 — sync.Pool vs Heap Allocation

## The Problem: GC Pressure at Scale

Building a canonical payment string (for idempotency keys and HMAC signing) on
every inbound request allocates a temporary `[]byte` scratch buffer:

```
Call 1: alloc []byte → build string → string(b) → GC collects []byte
Call 2: alloc []byte → build string → string(b) → GC collects []byte
...
```

At 1 million requests/second the GC reclaims millions of short-lived buffers per
second, raising stop-the-world GC pause frequency and inflating tail latency.

## The Solution: sync.Pool

`sync.Pool` maintains a per-P (per-OS-thread) cache of reusable objects.

- **`Get()`** returns a recycled buffer from the local P's pool — **no allocation**.
- **`Put()`** returns the buffer for reuse by the next caller on the same P.
- Cross-P theft is automatic; the pool scales linearly with `GOMAXPROCS`.

```go
var pool = sync.Pool{New: func() any { b := make([]byte, 0, 256); return &b }}

bp := pool.Get().(*[]byte)
b := (*bp)[:0]      // ← reset length, preserve capacity — always do this
defer func() {
    *bp = b         // write back in case append grew the slice
    pool.Put(bp)
}()
```

## Critical Invariant: Always Reset Before Use

The pool may return a buffer populated by a previous caller. Forgetting `b = b[:0]`
is a **silent data-corruption bug** — it will not panic, it will prepend stale bytes
that are nearly impossible to catch in production.

## Expected Benchmark Results

```
BenchmarkCanonical_WithPool-8          20_000_000     58 ns/op    48 B/op    1 allocs/op
BenchmarkCanonical_Direct-8             8_000_000    145 ns/op   160 B/op    5 allocs/op
BenchmarkCanonical_Parallel_WithPool-8 60_000_000     20 ns/op    48 B/op    1 allocs/op
```

The remaining `1 alloc/op` in `WithPool` is the `string(b)` copy — the caller owns
that string, so it cannot be pooled. The scratch `[]byte` is entirely eliminated.

## Running the Benchmarks

```bash
# Basic benchmark with allocation stats
go test ./04-memory-syncpool/... -bench=. -benchmem -count=3

# Observe Pool behaviour across GOMAXPROCS values (should scale linearly)
go test ./04-memory-syncpool/... -bench=Parallel -benchmem -cpu=1,2,4,8
```

## What If the Pod Crashes Mid-Operation?

`sync.Pool` is **purely in-process scratch memory** — it holds no business state,
no money, no account balances. A pod crash discards the pool along with the entire
process, and that is perfectly safe.

If a pod crashes while building a canonical payment string, the HTTP request fails
at the transport layer. The client retries. The next pod starts with a cold pool
(all buffers freshly allocated from the heap, same as startup), which is the normal
initial state.

The idempotency filter from Case 01 ensures the retried request is not processed twice.

**Impact of a crash on sync.Pool: none.** The only concern is the brief latency spike
after a restart while the pool warms up (first N calls allocate fresh buffers before
the pool has recycled objects to offer).
