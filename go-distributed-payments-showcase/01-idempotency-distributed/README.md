> 🌐 [Español](README-es.md)

# Case 01 — Distributed Idempotency Filter

## The Problem

A mobile banking app retries a failed payment request 3 times due to a network timeout.
The payment processor receives 3 identical requests. Without a guard, the customer is charged 3 times.

## The Solution: SETNX State Machine

```
Client sends request with idempotency_key: "txn-abc123"

Request 1 → SETNX("txn-abc123", "STARTED") → key inserted → process payment → SET("txn-abc123", "COMPLETED")
Request 2 → SETNX(...) loses → state STARTED → return ErrDuplicate or wait and query
Request 3 → same as Request 2; once finished, it receives ErrAlreadyCompleted
```

`SETNX` ("SET if Not eXists") is **atomic** — it checks and writes in a single operation
with no window for a race condition between two concurrent readers.

## State Machine

```
             ┌─────────────┐
             │   (no key)  │ ◄── Release() on transient failure (enables retry)
             └──────┬──────┘
                    │ Acquire() — SETNX wins
                    ▼
             ┌─────────────┐
             │   STARTED   │ ◄── concurrent Acquire() → ErrDuplicate
             └──────┬──────┘
                    │ Complete()
                    ▼
             ┌─────────────┐
             │  COMPLETED  │ ◄── future Acquire() → ErrAlreadyCompleted
             └─────────────┘     (expires after TTL)
```

## Key Go Primitives

| Primitive | Role |
|-----------|------|
| `sync.Mutex` | Simulates Redis SETNX atomicity in the in-memory test store |
| `sync/atomic.Int64` | Lock-free winner counter in the 1000-goroutine test |
| `chan struct{}` (closed) | Synchronised goroutine release for maximum contention |

## Running the Tests

```bash
go test ./01-idempotency-distributed/... -race -v -count=1
```

With `-race`, Go's data race detector validates that the filter itself has no races
even when 1000 goroutines compete simultaneously.

## What If the Pod Crashes Mid-Operation?

This is the most critical failure scenario for the idempotency filter.

**Scenario**: Pod A calls `Acquire()` — the key is set to STARTED — then crashes
before calling `Complete()` or `Release()`.

```
Pod A: Acquire() → STARTED → 💥 crash
                                    │
                                    │ TTL expires
                                    ▼
Pod B: Acquire() → STARTED (fresh) → Complete() → COMPLETED ✓
```

**Without TTL**: the key stays in STARTED forever. No other pod can process the
request. The customer is permanently locked out of retrying their payment.

**With TTL** (already set in `NewFilter`): the key expires after the configured
duration and another pod can attempt to recover the operation. See `TestFilter_PodCrashRecoveryViaTTL`.

But TTL **does not prove that the payment was not executed**. If Pod A charged the
external processor and crashed before saving `COMPLETED`, Pod B could charge again
when the TTL expires. Therefore the downstream operation must also receive
`txn-abc123` as an idempotency key, or Pod B must query the processor before retrying:

```
Request 2/3 while STARTED → ErrDuplicate; wait with backoff or query status
       │
       ├── COMPLETED → return the stored response, without charging again
       └── STARTED + expired lease → query provider; recover only after that
```

The correct strategy is **idempotency at every boundary**: Redis prevents two pods
from working simultaneously, and the payment processor prevents a retry from charging twice.

**Sizing the TTL**:

| TTL too short | TTL too long |
|---------------|--------------|
| Two pods process the same payment concurrently → double charge | Customer waits a long time before the retry is accepted → poor UX |

Rule of thumb: `TTL = 2–3× maximum expected processing duration`.
For a payment that takes ~2 s, use a TTL of 5–10 s.
