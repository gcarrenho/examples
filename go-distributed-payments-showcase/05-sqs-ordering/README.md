> 🌐 [Español](README-es.md)

# Case 05 — SQS FIFO as a Kafka Alternative for Ordered Payment Events

## When You Can't (or Don't Want to) Use Kafka

Kafka is excellent, but it carries real operational weight:

- Requires Zookeeper or KRaft for cluster coordination
- Needs a dedicated ops team (or a managed offering like Confluent / MSK)
- Over-engineered for teams processing < 50K messages/second
- Adds infrastructure cost that may not be justified early in a product

**AWS SQS FIFO** solves the ordering and exactly-once problem with fully managed
infrastructure, zero cluster management, and a simple HTTP API.

## How SQS FIFO Maps to Kafka Concepts

| Kafka concept | SQS FIFO equivalent | Notes |
|---------------|---------------------|-------|
| Topic | Queue URL | One queue per logical stream |
| Partition key | `MessageGroupId` | Same account_id → same group → ordered |
| Offset commit | `DeleteMessage` | Deleting = acknowledging success |
| Retention / replay | ❌ (max 14 days, no seek) | SQS does not support offset replay |
| Consumer group | Invisible via `VisibilityTimeout` | Only one consumer processes a group at a time |
| DLQ topic (manual write) | Redrive policy (infrastructure) | Automatic after `MaxReceiveCount` retries |
| Idempotency key | `MessageDeduplicationId` | Built-in 5-minute deduplication window |

## Ordering Guarantee

SQS FIFO delivers messages within a `MessageGroupId` in strict order and allows
only one in-flight message per group at a time:

```
Group "acc-1234": [CHARGE@0] → [REFUND@1] → [REVERSAL@2]
                       ↑ processed and deleted before @1 becomes visible
```

This is structurally identical to Kafka's per-partition guarantee.

## Failure Handling

```
Message received (VisibilityTimeout = 30s)
         │
         ▼
   ┌──────────┐
   │ Handler  │ → ResultOK    → DeleteMessage (removes from queue)
   │          │ → ResultRetry → do nothing; timeout expires; message reappears
   │          │ → ResultDLQ   → DeleteMessage + write to DLQ with error metadata
   └──────────┘
                                     ↑
          Or: let it expire MaxReceiveCount times → redrive policy auto-moves to DLQ
```

**Key difference from Kafka DLQ**: with Kafka you write to a DLQ topic in your code.
With SQS, the redrive policy is configured at the queue level in AWS — your code
can simply not delete the message and let it expire past `MaxReceiveCount`.
Explicit DLQ writes (shown in this case) add error metadata for observability.

## Kafka vs SQS FIFO — When to Choose

| Criteria | Kafka | SQS FIFO |
|----------|-------|----------|
| Throughput | Millions/sec | 3,000 msg/s per group; 300/s baseline |
| Replay / rewind | ✅ Any offset | ❌ Once deleted, gone |
| Infrastructure | Self-hosted or managed (MSK, Confluent) | Fully managed (AWS) |
| Ordering | Per partition | Per `MessageGroupId` |
| Exactly-once producer | ✅ (idempotent producer) | ✅ (`MessageDeduplicationId`) |
| Retention | Unlimited (disk) | Max 14 days |
| Cost model | Fixed cluster cost | Per-request ($0.50 / million) |
| Best for | High-throughput streams, analytics, replay | Moderate-volume task queues, AWS-native stacks |

## Running the Tests

```bash
go test ./05-sqs-ordering/... -race -v -count=1
```

## What If the Pod Crashes Mid-Operation?

**Scenario**: The consumer calls `Receive()` (SQS starts the `VisibilityTimeout`
clock), begins processing, and crashes before calling `DeleteMessage()`.

**What SQS does automatically** — no code required:
1. `VisibilityTimeout` expires (configurable, default 30 s).
2. The message **becomes visible in the queue again**.
3. Another consumer (or the recovered pod) receives and processes it.

```
Pod A: Receive [CHARGE] → execute charge → 💥 crash (DeleteMessage never called)

           VisibilityTimeout expires (e.g. 30 s) → message reappears

Pod B: Receive [CHARGE] → idempotency filter (Case 01) → ErrAlreadyCompleted → DeleteMessage ✓
```

This is SQS’s structural advantage over Kafka for crash recovery: there is no session
timeout, no rebalance, no partition reassignment. The visibility model handles
re-delivery automatically at the broker level.

**The risk: double processing** (same as Case 03). Apply the Case 01 idempotency
filter in every SQS handler before executing the charge.

**Tuning `VisibilityTimeout`**: set it to slightly longer than your maximum
processing time. Too short → false re-deliveries under normal load; too long →
slow recovery after a crash.
