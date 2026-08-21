// Package kafka is the Kafka producer/consumer adapters for payment-api.
package kafka

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/IBM/sarama"
	banking "github.com/examples/banking-core"
)

const ResultsTopic = "banking.payment.orders.results"

// resultHandler is the consumer's private contract — defined here by the consumer.
type resultHandler interface {
	HandleResult(ctx context.Context, result banking.PaymentOrderResult)
}

// ResultsConsumer reads PaymentOrderResult events published by payment-worker.
type ResultsConsumer struct {
	handler   resultHandler
	consumer  sarama.Consumer
	partition sarama.PartitionConsumer
	logger    *slog.Logger
}

func NewResultsConsumer(brokers []string, handler resultHandler, logger *slog.Logger) (*ResultsConsumer, error) {
	c, err := sarama.NewConsumer(brokers, sarama.NewConfig())
	if err != nil {
		return nil, fmt.Errorf("kafka results consumer: %w", err)
	}
	pc, err := c.ConsumePartition(ResultsTopic, 0, sarama.OffsetNewest)
	if err != nil {
		return nil, fmt.Errorf("kafka results consumer partition: %w", err)
	}
	return &ResultsConsumer{handler: handler, consumer: c, partition: pc, logger: logger}, nil
}

// Run blocks until ctx is cancelled. Intended to run in its own goroutine
// alongside the HTTP server in cmd/main.go.
func (c *ResultsConsumer) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c.partition.Messages():
			if !ok {
				return
			}
			result, err := banking.DecodeResult(msg.Value)
			if err != nil {
				c.logger.ErrorContext(ctx, "decode result failed", slog.String("err", err.Error()))
				continue
			}
			c.handler.HandleResult(ctx, result)
		}
	}
}

func (c *ResultsConsumer) Close() error { return c.consumer.Close() }
