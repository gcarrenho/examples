// Package ordering demonstrates partition-scoped ordering guarantees and
// Dead Letter Queue (DLQ) patterns for a Kafka-based payment event bus.
//
// The ordering problem
//
// Kafka guarantees message order within a partition, not across the topic.
// For payment events this creates a hard constraint: a REFUND must always
// be processed after its source CHARGE, or it touches account state that
// does not yet exist.
//
// Solution: route all events for the same account to the same partition
// using a deterministic hash of account_id. The same account always maps
// to the same partition on every producer instance and after every restart.
package ordering

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"
)

// EventType classifies a payment event.
type EventType string

const (
	EventCharge   EventType = "CHARGE"
	EventRefund   EventType = "REFUND"
	EventReversal EventType = "REVERSAL"
)

// PaymentEvent is the canonical Kafka message payload.
type PaymentEvent struct {
	EventID     string    `json:"event_id"`
	AccountID   string    `json:"account_id"`
	Type        EventType `json:"type"`
	AmountCents int64     `json:"amount_cents"`
	OccurredAt  time.Time `json:"occurred_at"`
}

// MessageSender is the minimal Kafka producer interface.
// In production this wraps sarama.SyncProducer or confluent-kafka-go.
type MessageSender interface {
	Send(ctx context.Context, topic string, partition int32, key, value []byte) error
}

// Producer routes PaymentEvents to the correct Kafka partition.
type Producer struct {
	sender     MessageSender
	topic      string
	partitions int32
}

// NewProducer creates a Producer for the given topic and partition count.
// partitions must match the actual topic configuration in Kafka.
func NewProducer(sender MessageSender, topic string, partitions int32) *Producer {
	return &Producer{sender: sender, topic: topic, partitions: partitions}
}

// Publish sends a PaymentEvent to the partition determined by its AccountID.
//
// Partition assignment is deterministic: the same AccountID always maps to the
// same partition, guaranteeing that all events for one account are consumed in
// arrival order by a single consumer instance.
func (p *Producer) Publish(ctx context.Context, evt PaymentEvent, payload []byte) error {
	partition := PartitionFor(evt.AccountID, p.partitions)
	if err := p.sender.Send(ctx, p.topic, partition, []byte(evt.AccountID), payload); err != nil {
		return fmt.Errorf("producer: account=%s partition=%d: %w", evt.AccountID, partition, err)
	}
	return nil
}

// PartitionFor maps accountID to a partition index deterministically.
//
// Algorithm: SHA-256(accountID) → first 4 bytes as uint32 → mod numPartitions.
// SHA-256 provides a uniform distribution across all account IDs, preventing
// hot partitions even when account ID prefixes cluster (e.g., "acc-000001").
func PartitionFor(accountID string, numPartitions int32) int32 {
	h := sha256.Sum256([]byte(accountID))
	v := binary.BigEndian.Uint32(h[:4])
	return int32(v % uint32(numPartitions))
}
