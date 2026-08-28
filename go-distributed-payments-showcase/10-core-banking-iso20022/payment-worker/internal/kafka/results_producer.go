package kafka

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	banking "github.com/examples/banking-core"
)

// ResultsTopic carries PaymentOrderResult events consumed by payment-api.
const ResultsTopic = "banking.payment.orders.results"

// ResultsProducer publishes PaymentOrderResult events.
// Satisfies worker.resultPublisher via structural typing.
type ResultsProducer struct {
	producer sarama.SyncProducer
}

func NewResultsProducer(brokers []string) (*ResultsProducer, error) {
	cfg := sarama.NewConfig()
	cfg.Producer.Return.Successes = true
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	p, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return nil, fmt.Errorf("kafka results producer: %w", err)
	}
	return &ResultsProducer{producer: p}, nil
}

func (p *ResultsProducer) PublishResult(_ context.Context, result banking.PaymentOrderResult) error {
	data, err := result.Encode()
	if err != nil {
		return fmt.Errorf("kafka results producer: encode: %w", err)
	}
	msg := &sarama.ProducerMessage{
		Topic: ResultsTopic,
		Key:   sarama.StringEncoder(result.UETR),
		Value: sarama.ByteEncoder(data),
	}
	if _, _, err := p.producer.SendMessage(msg); err != nil {
		return fmt.Errorf("kafka results producer: send: %w", err)
	}
	return nil
}

func (p *ResultsProducer) Close() error { return p.producer.Close() }
