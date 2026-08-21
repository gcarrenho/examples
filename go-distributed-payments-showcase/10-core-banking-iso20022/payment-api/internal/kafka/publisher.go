// Package kafka is the Kafka producer adapter for payment-api.
package kafka

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	banking "github.com/examples/banking-core"
)

const Topic = "banking.payment.orders.initiated"

// Producer publishes PaymentOrderInitiated events to Kafka.
// Satisfies api.publisher via structural typing.
type Producer struct {
	producer sarama.SyncProducer
	topic    string
}

func NewProducer(brokers []string) (*Producer, error) {
	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Idempotent = true // exactly-once producer
	cfg.Net.MaxOpenRequests = 1    // required when Idempotent=true

	p, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return nil, fmt.Errorf("kafka producer: %w", err)
	}
	return &Producer{producer: p, topic: Topic}, nil
}

// Publish routes all events for the same account to the same partition
// (Debtor IBAN as key) → ordering guarantee per account (Case 03 pattern).
func (p *Producer) Publish(_ context.Context, event banking.PaymentOrderInitiated) error {
	data, err := event.Encode()
	if err != nil {
		return fmt.Errorf("kafka producer: encode: %w", err)
	}
	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(event.Order.Debtor.IBAN),
		Value: sarama.ByteEncoder(data),
	}
	if _, _, err := p.producer.SendMessage(msg); err != nil {
		return fmt.Errorf("kafka producer: send: %w", err)
	}
	return nil
}

func (p *Producer) Close() error { return p.producer.Close() }
