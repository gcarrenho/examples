> 🌐 [Español](README-es.md)

# go-distributed-payments-showcase

An educational monorepo demonstrating the **hard architectural problems** that arise when building distributed payment systems at scale. Each case study is self-contained, uses only the Go standard library, and is accompanied by concurrent tests that prove the solution works under real-world traffic conditions.

## Why Payments Are Architecturally Interesting

Payments are the domain where **correctness is non-negotiable**. A bug in a social media feed shows the wrong post; a bug in a payment system charges a customer twice, loses money, or both. This forces engineers to reason carefully about:

- **Exactly-once semantics** — the same charge request must never be processed twice, even if the network retries it 100 times.
- **Concurrent state mutation** — thousands of goroutines may try to update the same account balance simultaneously.
- **Message ordering** — a refund must never be processed before the original charge.
- **Memory efficiency** — payment processors handle millions of requests per second; every unnecessary allocation adds GC pressure and tail latency.

## Cases

| # | Case | Core Concept | Infrastructure |
|---|------|-------------|----------------|
| 01 | [idempotency-distributed](01-idempotency-distributed/) | Redis SETNX state machine (STARTED → COMPLETED) | Redis |
| 02 | [concurrency-occ](02-concurrency-occ/) | Optimistic (OCC) vs Pessimistic (PCC) Concurrency Control | PostgreSQL |
| 03 | [kafka-ordering](03-kafka-ordering/) | Partition-scoped ordering + Dead Letter Queue | Kafka |
| 04 | [memory-syncpool](04-memory-syncpool/) | `sync.Pool` vs heap allocation — 1 alloc/op | None |
| 05 | [sqs-ordering](05-sqs-ordering/) | SQS FIFO as a Kafka alternative — `MessageGroupId` ordering + redrive DLQ | AWS SQS |

## Running the Tests

```bash
# Start infrastructure (required for integration tests)
docker-compose up -d

# Run all unit tests (no infrastructure required)
go test ./... -race -count=1 -v

# Run benchmarks for Case 04
go test ./04-memory-syncpool/... -bench=. -benchmem -count=3

# Parallel benchmark across GOMAXPROCS levels
go test ./04-memory-syncpool/... -bench=Parallel -benchmem -cpu=1,2,4,8
```

## Project Structure

```
go-distributed-payments-showcase/
├── go.mod                         # Single module; no external dependencies
├── docker-compose.yml             # Postgres + Redis + Kafka for integration testing
├── 01-idempotency-distributed/    # SETNX two-phase state machine
│   ├── redis_idempotency.go       # Filter with StateStore interface
│   └── redis_idempotency_test.go  # 1000-goroutine exactly-once proof
├── 02-concurrency-occ/            # OCC version guard vs PCC row lock
│   ├── db_occ.go                  # Ledger with OCC/PCC strategies
│   └── db_occ_test.go             # 100-goroutine convergence + conflict counter
├── 03-kafka-ordering/             # Deterministic partition routing + DLQ
│   ├── producer.go                # SHA-256 partition assignment
│   ├── consumer.go                # Synchronous processing + DLQ routing
│   └── ordering_test.go           # Partition stability + consumer state machine
├── 04-memory-syncpool/            # GC pressure elimination
│   ├── parser.go                  # Pool-backed []byte canonical builder
│   └── parser_bench_test.go       # 1 alloc/op (Pool) vs 2 allocs/op (Direct)
└── 05-sqs-ordering/               # Kafka alternative: SQS FIFO
    ├── sqs_producer.go            # MessageGroupId=AccountID + EventID deduplication
    ├── sqs_consumer.go            # Poll loop + explicit DLQ write
    └── sqs_test.go                # Producer contracts + consumer state machine
```

## Crash Behaviour at a Glance

A pod can crash at any point. The table below shows what each infrastructure component
does automatically, and what the application code must add on top.

| Case | Infrastructure auto-recovery | Application responsibility |
|------|------------------------------|----------------------------|
| 01 — Idempotency | TTL expires → abandoned STARTED key may be recovered | Reuse the same idempotency key at the external processor |
| 02 — OCC | DB rolls back uncommitted write on connection close (zero partial state) | Nothing extra needed |
| 02 — PCC | `SELECT FOR UPDATE` lock released on connection close (seconds) | Tune `tcp_keepalives_idle` to detect dead connections fast |
| 03 — Kafka | Session timeout → rebalance → unconfirmed offset re-delivered | Handler **must** check idempotency filter (Case 01) before executing |
| 04 — sync.Pool | N/A — pool is scratch memory; crash discards it safely | Upstream retries the HTTP request |
| 05 — SQS FIFO | VisibilityTimeout expires → message reappears automatically | Handler **must** check idempotency filter (Case 01) before executing |

> **The common thread**: Cases 03 and 05 guarantee *at-least-once* delivery, not
> *exactly-once*. Exactly-once requires composing them with Case 01 and propagating
> the same idempotency key to the payment provider.

## Design Principles

Each case follows the same structure:

1. **Interface-first**: external dependencies (Redis, Postgres, Kafka) are abstracted behind narrow interfaces. Unit tests run against in-memory implementations; integration tests run against real infrastructure.
2. **No frameworks**: standard library only. The patterns are universal — adding a library would obscure the technique.
3. **Measurable correctness**: tests use `atomic.Int64` counters and `sync.WaitGroup` barriers to assert behaviour scientifically, not probabilistically.
