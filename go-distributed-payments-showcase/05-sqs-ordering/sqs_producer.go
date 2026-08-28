// Package sqsordering demonstrates AWS SQS FIFO as an alternative to Kafka
// for partition-scoped ordering of payment events.
//
// The ordering mechanism
//
// SQS FIFO uses MessageGroupId as the ordering unit. All messages sharing the
// same MessageGroupId are delivered in the order they were sent, and at most
// one message per group is in-flight at any time — structurally identical to
// Kafka's per-partition guarantee.
//
// Built-in idempotency
//
// MessageDeduplicationId provides a 5-minute exactly-once window at the
// broker level: if a producer sends two messages with the same ID within
// 5 minutes, the second is silently dropped. This complements (but does not
// replace) the application-level idempotency filter in Case 01.
package sqsordering

import (
	"context"
	"crypto/sha256"
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

// PaymentEvent is the canonical payload sent to SQS.
type PaymentEvent struct {
	EventID     string    `json:"event_id"`
	AccountID   string    `json:"account_id"`
	Type        EventType `json:"type"`
	AmountCents int64     `json:"amount_cents"`
	OccurredAt  time.Time `json:"occurred_at"`
}

// OutboundMessage is the SQS-enriched envelope around a PaymentEvent payload.
type OutboundMessage struct {
	// MessageGroupId routes the message to the correct ordering group.
	// Set to AccountID so all events for one account are strictly ordered.
	MessageGroupId string
	// MessageDeduplicationId is SQS's built-in idempotency key.
	// Using EventID ensures the producer can safely retry sends.
	MessageDeduplicationId string
	Body                   []byte
}

// SQSSender is the minimal SQS producer interface.
// In production this wraps aws-sdk-go-v2's sqs.SendMessage.
type SQSSender interface {
	Send(ctx context.Context, queueURL string, msg OutboundMessage) error
}

// Producer routes PaymentEvents to an SQS FIFO queue.
type Producer struct {
	sender   SQSSender
	queueURL string
}

// NewProducer creates a Producer for the given SQS FIFO queue URL.
func NewProducer(sender SQSSender, queueURL string) *Producer {
	return &Producer{sender: sender, queueURL: queueURL}
}

// Publish sends a PaymentEvent to SQS FIFO.
//
// MessageGroupId = AccountID  → ordering guarantee per account.
// MessageDeduplicationId = EventID → producer-side exactly-once within 5 minutes.
//
// Unlike the Kafka producer, no partition calculation is needed: SQS manages
// group routing internally. The trade-off is less control over throughput
// sharding (all events for one account are serialised regardless of volume).
func (p *Producer) Publish(ctx context.Context, evt PaymentEvent, payload []byte) error {
	if err := p.sender.Send(ctx, p.queueURL, OutboundMessage{
		MessageGroupId:         evt.AccountID,
		MessageDeduplicationId: evt.EventID,
		Body:                   payload,
	}); err != nil {
		return fmt.Errorf("sqs producer: account=%s event=%s: %w", evt.AccountID, evt.EventID, err)
	}
	return nil
}

// DeduplicationID generates a content-based deduplication ID from the payload.
// Use this when EventID is not available (e.g., forwarding third-party webhooks).
// SQS accepts both explicit IDs and content-based hashing via queue configuration.
func DeduplicationID(payload []byte) string {
	h := sha256.Sum256(payload)
	return fmt.Sprintf("%x", h[:16]) // 32-char hex, well within SQS's 128-char limit
}
