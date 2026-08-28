> 🌐 [Español](README-es.md)

# Case 02 — Optimistic vs Pessimistic Concurrency Control

## The Problem: Lost Update

Two goroutines debit the same account simultaneously:

```
Goroutine A reads:  balance=$100, version=5
Goroutine B reads:  balance=$100, version=5
Goroutine A writes: balance=$40,  version=6   (debited $60)
Goroutine B writes: balance=$40,  version=6   (debited $60, overwrites A's write)

Result: balance=$40 — but two $60 debits were applied. The account lost $60.
```

This is the **Lost Update** anomaly. Both writers read stale data and the second
writer overwrites the first without knowing a commit happened.

## Optimistic Concurrency Control (OCC)

Optimism: "collisions are rare; let everyone try and detect conflicts on write."

```sql
-- 1. Read with version
SELECT balance, version FROM accounts WHERE id = $1;

-- 2. Write with version guard
UPDATE accounts
   SET balance = $new_balance, version = version + 1
 WHERE id = $1 AND version = $expected_version;

-- If 0 rows were updated → version mismatch → another writer committed → retry
```

**Best for**: low-to-medium contention. Reads are non-blocking; conflicts are rare.

## Pessimistic Concurrency Control (PCC)

Pessimism: "collisions are expensive; acquire an exclusive lock before reading."

```sql
BEGIN;
-- Row-level lock: no other transaction can update this row until COMMIT
SELECT balance FROM accounts WHERE id = $1 FOR UPDATE;
UPDATE accounts SET balance = balance + $delta WHERE id = $1;
COMMIT;
```

**Best for**: high contention. Writers queue rather than spin-and-retry.

## OCC vs PCC Trade-off

| Dimension | OCC | PCC |
|-----------|-----|-----|
| Concurrency model | Fail-fast + retry | Block and wait |
| Throughput (low contention) | ✅ Higher | ❌ Lock overhead |
| Throughput (high contention) | ❌ Many retries | ✅ Predictable |
| Deadlock risk | None | Possible |
| Implementation complexity | Retry loop | Transaction management |

## Why PostgreSQL for OCC/PCC?

PostgreSQL provides the best native building blocks for both strategies in a single engine.

| Database | OCC | PCC | Notes |
|----------|-----|-----|-------|
| **PostgreSQL** | ✅ `UPDATE … WHERE version=$v` | ✅ `SELECT FOR UPDATE` | MVCC: readers never block writers. Row-level locking is precise. `UPDATE … RETURNING` gives the new state atomically. |
| **MySQL / MariaDB** | ✅ version column works | ⚠️ gap locks | `SELECT FOR UPDATE` on a range can acquire gap locks between rows, causing unexpected deadlocks under PCC with range queries. |
| **MongoDB** | ✅ `findOneAndUpdate` + `$where` | ⚠️ document-level only | Multi-document transactions added in v4.0 but carry session overhead. OCC is idiomatic; PCC requires manual two-phase locking. |
| **CockroachDB** | ✅ serializable by default | ✅ `SELECT FOR UPDATE` | Distributed Postgres-compatible, but every lock/conflict round-trip crosses the network — higher p99 latency than single-node Postgres. |
| **DynamoDB** | ✅ `ConditionExpression: version=:v` | ❌ no row locking | Conditional writes are OCC-compatible but there is no `SELECT FOR UPDATE` equivalent. PCC must be emulated with separate lock items (complex, error-prone). |
| **Redis** | ⚠️ `WATCH / MULTI / EXEC` | ❌ no row locking | Optimistic transactions via WATCH exist but are limited to simple key ops. Not designed for relational financial data. Use Redis for idempotency keys (Case 01), not balance management. |
| **SQLite** | ✅ yes | ✅ yes | Single-writer model; the database itself is the lock. Unsuitable for distributed payment systems but fine for local dev/testing. |

**Why Postgres wins for payments specifically:**

1. **MVCC** — Multi-Version Concurrency Control means `SELECT` never blocks `UPDATE`, so read-heavy OCC workloads don't starve writers.
2. **Serializable Snapshot Isolation (SSI)** — Postgres can automatically detect and abort serialization anomalies without any `version` column at all (set `isolation level serializable`).
3. **`RETURNING` clause** — `UPDATE accounts SET balance=$1, version=version+1 WHERE id=$2 AND version=$3 RETURNING balance` reads the committed new state in the same round-trip, eliminating a second `SELECT`.
4. **Advisory locks** — `pg_advisory_xact_lock(hashtext(account_id))` provides named application-level locks for PCC without touching the row at all.
5. **Partial indexes** — `CREATE INDEX ON payments (account_id) WHERE status = 'PENDING'` makes the version guard query O(log n) on the active subset of rows.

## Running the Tests

```bash
go test ./02-concurrency-occ/... -race -v -count=1
```

Watch the `OCC conflicts (retries)` log line. Under 100 goroutines, OCC typically
produces 400–900 conflicts for 100 successful writes — each conflict is wasted work.
PCC produces zero conflicts because the mutex serialises all writers.

## What If the Pod Crashes Mid-Operation?

**OCC**: Pod reads the account (balance + version), then crashes before `UpdateOCC` runs.
No state was written — the database row is completely untouched. The next request
reads fresh data and proceeds normally. OCC is naturally crash-safe because the write
is a single atomic `UPDATE` statement; there is no intermediate partial state.

**PCC**: Pod calls `SELECT FOR UPDATE` (acquires the row lock), then crashes before `COMMIT`.
PostgreSQL detects the broken TCP connection and **automatically rolls back the transaction
and releases the row lock**. No application code is needed for recovery.

```
Pod A: SELECT FOR UPDATE → 💥 crash
           │
           │ TCP connection closed → Postgres rolls back + releases lock (seconds)
           ▼
Pod B: SELECT FOR UPDATE → UPDATE → COMMIT ✓
```

**Key takeaway**: OCC leaves zero database state on crash; recovery is instant.
PCC holds a lock until the database detects the dead connection. Tune
`tcp_keepalives_idle` on the pgx connection pool to detect crashes in seconds
rather than the OS default of several minutes.
