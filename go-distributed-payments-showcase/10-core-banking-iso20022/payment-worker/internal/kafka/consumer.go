// Package kafka is the Kafka consumer adapter for payment-worker.
package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/IBM/sarama"
	banking "github.com/examples/banking-core"
)

const Topic = "banking.payment.orders.initiated"

// eventHandler is the consumer's private contract — defined here by the consumer.
// worker.Worker satisfies this via structural typing.
type eventHandler interface {
	ProcessEvent(ctx context.Context, event banking.PaymentOrderInitiated) error
}

// Consumer reads events from all topic partitions and drives eventHandler.
type Consumer struct {
	handler    eventHandler
	consumer   sarama.Consumer
	partitions []sarama.PartitionConsumer
	logger     *slog.Logger
}

func NewConsumer(brokers []string, handler eventHandler, logger *slog.Logger) (*Consumer, error) {
	c, err := sarama.NewConsumer(brokers, sarama.NewConfig())
	if err != nil {
		return nil, fmt.Errorf("kafka consumer: %w", err)
	}
	partIDs, err := c.Partitions(Topic)
	if err != nil {
		return nil, fmt.Errorf("kafka consumer: list partitions: %w", err)
	}
	pcs := make([]sarama.PartitionConsumer, 0, len(partIDs))
	for _, p := range partIDs {
		pc, err := c.ConsumePartition(Topic, p, sarama.OffsetNewest)
		if err != nil {
			return nil, fmt.Errorf("kafka consumer: partition %d: %w", p, err)
		}
		pcs = append(pcs, pc)
	}
	return &Consumer{handler: handler, consumer: c, partitions: pcs, logger: logger}, nil
}

// Run blocks until ctx is cancelled. One goroutine per partition, all drain concurrently.
func (c *Consumer) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, pc := range c.partitions {
		wg.Add(1)
		go func(pc sarama.PartitionConsumer) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case msg, ok := <-pc.Messages():
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
		}(pc)
	}
	wg.Wait()
}

func (c *Consumer) Close() error {
	for _, pc := range c.partitions {
		_ = pc.Close()
	}
	return c.consumer.Close()
}
