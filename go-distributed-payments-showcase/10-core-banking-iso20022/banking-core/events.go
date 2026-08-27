package banking

import (
	"context"
	"encoding/json"
	"time"
)

// PaymentOrderInitiated is the domain event published to Kafka when a payment is accepted.
// This is the contract between payment-api (producer) and payment-worker (consumer).
// Both services define this struct independently — the JSON wire format is the contract,
// not the Go type. This mirrors how orders-svc and payment-svc share no Go code.
type PaymentOrderInitiated struct {
	EventID    string // UETR reused as event idempotency key
	OccurredAt time.Time
	Order      PaymentOrder
}

func (e PaymentOrderInitiated) Encode() ([]byte, error) {
	return json.Marshal(e)
}

func DecodeEvent(data []byte) (PaymentOrderInitiated, error) {
	var e PaymentOrderInitiated
	return e, json.Unmarshal(data, &e)
}

// EventPublisher is the outbound port for the payment-api service.
// Kafka producer satisfies this; in-memory publisher for tests.
type EventPublisher interface {
	Publish(ctx context.Context, event PaymentOrderInitiated) error
}

// PaymentOrderResult is the domain event published by payment-worker once an order
// reaches a final state (ACSC/RJCT) or fails with a business error. payment-api
// consumes this to answer GET /payments/{uetr} — without it the client that got
// a 202 Accepted would have no way to ever learn the outcome of their payment.
type PaymentOrderResult struct {
	UETR       string
	Status     PaymentStatus // StatusSettled | StatusRejected
	Reason     string        // populated on rejection (e.g. "insufficient balance")
	OccurredAt time.Time
}

func (e PaymentOrderResult) Encode() ([]byte, error) {
	return json.Marshal(e)
}

func DecodeResult(data []byte) (PaymentOrderResult, error) {
	var e PaymentOrderResult
	return e, json.Unmarshal(data, &e)
}

// ResultPublisher is the outbound port payment-worker uses to report final outcomes.
type ResultPublisher interface {
	PublishResult(ctx context.Context, result PaymentOrderResult) error
}
