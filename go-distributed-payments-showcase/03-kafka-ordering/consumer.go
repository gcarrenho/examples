package ordering

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// ProcessResult instructs the consumer how to handle a message after processing.
type ProcessResult int

const (
	ResultOK    ProcessResult = iota // commit offset; processing succeeded
	ResultRetry                      // do not commit; Kafka will re-deliver this offset
	ResultDLQ                        // unrecoverable; route to Dead Letter Queue and commit
)

// Handler processes a single PaymentEvent.
type Handler interface {
	Handle(ctx context.Context, evt PaymentEvent) ProcessResult
}

// DLQWriter routes unprocessable messages to a Dead Letter Queue topic.
type DLQWriter interface {
	Write(ctx context.Context, topic string, key, value []byte) error
}

// MessageCommitter acknowledges a processed message to the Kafka broker.
type MessageCommitter interface {
	Commit(ctx context.Context, topic string, partition int32, offset int64) error
}

// Unmarshaler decodes raw Kafka bytes into a PaymentEvent.
type Unmarshaler interface {
	Unmarshal(data []byte) (PaymentEvent, error)
}

// Message is a raw Kafka record with its routing metadata.
type Message struct {
	Topic     string
	Partition int32
	Offset    int64
	Key       []byte
	Value     []byte
}

// Consumer processes messages synchronously within a Kafka partition.
//
// Synchronous processing is the ordering guarantee: by handling one message at
// a time within a partition (no inner goroutines), a REFUND event at offset 5
// is never applied before the CHARGE at offset 4.
//
// Cross-partition parallelism is the caller's responsibility. The canonical
// pattern is one Consumer goroutine per assigned partition.
type Consumer struct {
	handler     Handler
	dlq         DLQWriter
	committer   MessageCommitter
	unmarshaler Unmarshaler
	dlqTopic    string
	logger      *slog.Logger
}

// NewConsumer wires together all Consumer dependencies.
func NewConsumer(
	handler Handler,
	dlq DLQWriter,
	committer MessageCommitter,
	unmarshaler Unmarshaler,
	dlqTopic string,
	logger *slog.Logger,
) *Consumer {
	return &Consumer{
		handler:     handler,
		dlq:         dlq,
		committer:   committer,
		unmarshaler: unmarshaler,
		dlqTopic:    dlqTopic,
		logger:      logger,
	}
}

// Process handles one Kafka message synchronously.
//
// Offset commitment only occurs on ResultOK or after a successful DLQ write,
// ensuring no message is silently dropped and no offset advances past an
// unprocessed record.
func (c *Consumer) Process(ctx context.Context, msg Message) error {
	evt, err := c.unmarshaler.Unmarshal(msg.Value)
	if err != nil {
		return c.sendToDLQ(ctx, msg, fmt.Errorf("unmarshal: %w", err))
	}

	switch c.handler.Handle(ctx, evt) {
	case ResultOK:
		return c.committer.Commit(ctx, msg.Topic, msg.Partition, msg.Offset)
	case ResultRetry:
		c.logger.WarnContext(ctx, "transient failure; offset will be re-delivered",
			slog.String("event_id", evt.EventID),
			slog.Int64("offset", msg.Offset),
		)
		return nil
	case ResultDLQ:
		return c.sendToDLQ(ctx, msg, errors.New("permanent processing failure"))
	default:
		return fmt.Errorf("consumer: unexpected result for offset %d", msg.Offset)
	}
}

func (c *Consumer) sendToDLQ(ctx context.Context, msg Message, reason error) error {
	c.logger.ErrorContext(ctx, "routing to DLQ",
		slog.String("topic", msg.Topic),
		slog.Int("partition", int(msg.Partition)),
		slog.Int64("offset", msg.Offset),
		slog.String("reason", reason.Error()),
	)
	if err := c.dlq.Write(ctx, c.dlqTopic, msg.Key, msg.Value); err != nil {
		return fmt.Errorf("consumer: DLQ write: %w", err)
	}
	// Commit the original offset so the partition advances past the poison pill.
	return c.committer.Commit(ctx, msg.Topic, msg.Partition, msg.Offset)
}
