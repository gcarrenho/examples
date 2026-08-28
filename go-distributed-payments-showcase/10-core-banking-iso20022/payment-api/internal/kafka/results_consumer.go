// Package kafka is the Kafka producer/consumer adapters for payment-api.
package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/IBM/sarama"
	banking "github.com/examples/banking-core"
)

const ResultsTopic = "banking.payment.orders.results"

// resultHandler is the consumer's private contract — defined here by the consumer.
type resultHandler interface {
	HandleResult(ctx context.Context, result banking.PaymentOrderResult)
}

// ResultsConsumer reads PaymentOrderResult events from all topic partitions.
type ResultsConsumer struct {
	handler    resultHandler
	consumer   sarama.Consumer
	partitions []sarama.PartitionConsumer
	logger     *slog.Logger
}

func NewResultsConsumer(brokers []string, handler resultHandler, logger *slog.Logger) (*ResultsConsumer, error) {
	c, err := sarama.NewConsumer(brokers, sarama.NewConfig())
	if err != nil {
		return nil, fmt.Errorf("kafka results consumer: %w", err)
	}
	partIDs, err := c.Partitions(ResultsTopic)
	if err != nil {
		return nil, fmt.Errorf("kafka results consumer: list partitions: %w", err)
	}
	pcs := make([]sarama.PartitionConsumer, 0, len(partIDs))
	for _, p := range partIDs {
		pc, err := c.ConsumePartition(ResultsTopic, p, sarama.OffsetNewest)
		if err != nil {
			return nil, fmt.Errorf("kafka results consumer: partition %d: %w", p, err)
		}
		pcs = append(pcs, pc)
	}
	return &ResultsConsumer{handler: handler, consumer: c, partitions: pcs, logger: logger}, nil
}

// Run blocks until ctx is cancelled. One goroutine per partition.
func (c *ResultsConsumer) Run(ctx context.Context) {
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
					result, err := banking.DecodeResult(msg.Value)
					if err != nil {
						c.logger.ErrorContext(ctx, "decode result failed", slog.String("err", err.Error()))
						continue
					}
					c.handler.HandleResult(ctx, result)
				}
			}
		}(pc)
	}
	wg.Wait()
}

func (c *ResultsConsumer) Close() error {
	for _, pc := range c.partitions {
		_ = pc.Close()
	}
	return c.consumer.Close()
}
