// Package kafka is the Kafka consumer adapter for payment-worker.
package kafka

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/IBM/sarama"
	banking "github.com/examples/banking-core"
)

const Topic = "banking.payment.orders.initiated"

// eventHandler is the consumer's private contract — defined here by the consumer.
// worker.Worker satisfies this via structural typing.
type eventHandler interface {
	ProcessEvent(ctx context.Context, event banking.PaymentOrderInitiated) error
}

// Consumer reads events from Kafka and drives eventHandler.
type Consumer struct {
	handler   eventHandler
	consumer  sarama.Consumer
	partition sarama.PartitionConsumer
	logger    *slog.Logger
}

func NewConsumer(brokers []string, handler eventHandler, logger *slog.Logger) (*Consumer, error) {
	c, err := sarama.NewConsumer(brokers, sarama.NewConfig())
	if err != nil {
		return nil, fmt.Errorf("kafka consumer: %w", err)
	}
	pc, err := c.ConsumePartition(Topic, 0, sarama.OffsetNewest)
	if err != nil {
		return nil, fmt.Errorf("kafka consumer partition: %w", err)
	}
	return &Consumer{handler: handler, consumer: c, partition: pc, logger: logger}, nil
}

// Run blocks until ctx is cancelled, processing one message at a time per partition.
func (c *Consumer) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c.partition.Messages():
			if !ok {
				return
			}
			event, err := banking.DecodeEvent(msg.Value)
			if err != nil {
				c.logger.ErrorContext(ctx, "decode event failed", slog.String("err", err.Error()))
				continue
			}
			if err := c.handler.ProcessEvent(ctx, event); err != nil {
				c.logger.ErrorContext(ctx, "process event failed",
					slog.String("uetr", event.Order.UETR), slog.String("err", err.Error()))
			}
		}
	}
}

func (c *Consumer) Close() error { return c.consumer.Close() }
