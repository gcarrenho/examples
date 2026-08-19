> 🌐 [Español](README-es.md)

# Case 03 — Kafka Partition Ordering + Dead Letter Queue

## The Ordering Problem

Kafka guarantees **in-order delivery within a partition**, not across the topic.

For payment events, this creates a hard constraint: a REFUND for a transaction
must be processed *after* the original CHARGE, or the refund touches an account
state that does not yet exist.

If events land on different partitions:

```
Partition 0: [REFUND  ← processed by consumer-1 first]
Partition 1: [CHARGE  ← processed by consumer-2, arrives later]
```

Consumer-1 processes the REFUND against an account that has no charge yet — invalid state.

## Solution 1: Deterministic Partition Assignment

Route all events for the same account to the same partition using a deterministic hash:

```
partition = SHA-256(account_id)[0:4] mod numPartitions
```

SHA-256 provides a uniform distribution — no hot partitions even when account IDs share a common prefix (e.g., `acc-0000001`, `acc-0000002`, ...).

The same account always lands on the same partition **forever**, regardless of which
producer instance sends it or how many times the service restarts.

## Solution 2: Synchronous Consumer per Partition

Within a partition, the consumer processes one message at a time (no inner goroutines).
Cross-partition parallelism is achieved by running **one goroutine per assigned partition**:

```
Partition 0 goroutine: [msg@0] → [msg@1] → [msg@2] → ...   (sequential)
Partition 1 goroutine: [msg@0] → [msg@1] → [msg@2] → ...   (sequential)
Partition 2 goroutine: [msg@0] → [msg@1] → [msg@2] → ...   (sequential)
```

## The Dead Letter Queue (DLQ) Pattern

Some messages cannot be processed — malformed JSON, permanently unavailable
downstream service, unknown event type. Retrying forever blocks the partition:
**one poison pill stops an entire event stream**.

The DLQ routes unprocessable messages to a separate topic and commits the original
offset, allowing the partition to advance past the poison pill.

```
                ┌──────────┐
Normal message →│ Handler  │→ ResultOK    → commit offset
                │          │→ ResultRetry → do not commit (Kafka re-delivers)
                │          │→ ResultDLQ   → write to DLQ topic → commit offset
                └──────────┘
```

## Running the Tests

```bash
go test ./03-kafka-ordering/... -race -v -count=1
```

## What If the Pod Crashes Mid-Operation?

**Scenario**: The consumer processes a message and crashes **before committing the offset**.
Kafka has no record the message was ever processed.

**What Kafka does automatically**:
1. The session heartbeat stops → session timeout fires (default: ~30 s).
2. The consumer group coordinator triggers a **rebalance**.
3. The partition is reassigned to another live consumer instance.
4. The new consumer resumes from the last **committed** offset — the unprocessed
   message is **re-delivered**.

```
Pod A: receive [CHARGE@offset=7] → execute charge → 💥 crash (offset never committed)

           session timeout → rebalance → Pod B takes the partition

Pod B: receive [CHARGE@offset=7] → idempotency filter (Case 01) → ErrAlreadyCompleted → commit offset ✓
```

**The risk: double processing.** If the business logic ran before the crash, the
message is processed twice without protection — the account is debited twice.

**The solution: combine with Case 01 (idempotency filter).** Every payment handler
must check the idempotency filter using the event's idempotency key *before* executing
the charge. The second delivery hits `ErrAlreadyCompleted` and returns immediately.

**Tuning recovery speed**:
- `session.timeout.ms` — lower values detect crashes faster but risk false positives under GC pauses
- `heartbeat.interval.ms` — set to ~1/3 of `session.timeout.ms`
- `max.poll.interval.ms` — maximum time between polls; if exceeded, Kafka considers the consumer dead
