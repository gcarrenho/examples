package sqsordering

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// ProcessResult instructs the consumer how to handle a received message.
type ProcessResult int

const (
	ResultOK    ProcessResult = iota // delete the message; processing succeeded
	ResultRetry                      // do not delete; VisibilityTimeout expires → re-delivery
	ResultDLQ                        // unrecoverable; delete + explicit write to DLQ queue
)

// InboundMessage is a message received from SQS with its acknowledgement handle.
type InboundMessage struct {
	MessageGroupId string // the AccountID that ordered this group
	Body           []byte
	// ReceiptHandle is an opaque token returned by SQS on receive.
	// Required for DeleteMessage and ChangeMessageVisibility calls.
	ReceiptHandle string
	// ApproximateReceiveCount is the number of times SQS has delivered this message.
	// When this exceeds the queue's MaxReceiveCount, the redrive policy fires.
	ApproximateReceiveCount int
}

// SQSReceiver is the minimal SQS consumer interface.
// In production this wraps aws-sdk-go-v2's sqs.ReceiveMessage / DeleteMessage.
type SQSReceiver interface {
	// Receive long-polls the queue and returns up to maxMessages messages.
	// SQS FIFO delivers at most one message per MessageGroupId per call.
	Receive(ctx context.Context, queueURL string, maxMessages int) ([]InboundMessage, error)
	// Delete removes a successfully processed message from the queue.
	Delete(ctx context.Context, queueURL string, receiptHandle string) error
	// ChangeVisibility resets the visibility timeout, making the message
	// re-deliverable sooner (useful for fast retry) or later (back-pressure).
	ChangeVisibility(ctx context.Context, queueURL string, receiptHandle string, seconds int) error
}

// DLQWriter routes unprocessable messages to a Dead Letter Queue.
// Unlike Kafka (where you write to a DLQ topic in code), SQS can handle DLQ
// automatically via redrive policy. Explicit writes add error metadata for
// observability and structured alerting.
type DLQWriter interface {
	Write(ctx context.Context, queueURL string, msg OutboundMessage) error
}

// Handler processes a single PaymentEvent decoded from an InboundMessage.
type Handler interface {
	Handle(ctx context.Context, evt PaymentEvent) ProcessResult
}

// Unmarshaler decodes raw SQS bytes into a PaymentEvent.
type Unmarshaler interface {
	Unmarshal(data []byte) (PaymentEvent, error)
}

// Consumer polls an SQS FIFO queue and processes messages synchronously.
//
// SQS FIFO delivers at most one message per MessageGroupId at a time:
// the next message in the group becomes visible only after the current one
// is deleted (acknowledged) or its VisibilityTimeout expires.
//
// This is the ordering guarantee — no goroutine-per-group logic is needed
// in the consumer code. SQS enforces single-active-consumer per group at
// the broker level, unlike Kafka where you manage partition assignment.
type Consumer struct {
	receiver    SQSReceiver
	handler     Handler
	dlq         DLQWriter
	unmarshaler Unmarshaler
	queueURL    string
	dlqURL      string
	maxMessages int
	logger      *slog.Logger
}

// NewConsumer wires together all Consumer dependencies.
// maxMessages controls the SQS batch size per Receive call (max 10).
func NewConsumer(
	receiver SQSReceiver,
	handler Handler,
	dlq DLQWriter,
	unmarshaler Unmarshaler,
	queueURL, dlqURL string,
	maxMessages int,
	logger *slog.Logger,
) *Consumer {
	if maxMessages < 1 || maxMessages > 10 {
		maxMessages = 10
	}
	return &Consumer{
		receiver:    receiver,
		handler:     handler,
		dlq:         dlq,
		unmarshaler: unmarshaler,
		queueURL:    queueURL,
		dlqURL:      dlqURL,
		maxMessages: maxMessages,
		logger:      logger,
	}
}

// Poll performs one long-poll cycle: receives a batch of messages and processes
// each one synchronously before moving to the next.
//
// SQS FIFO guarantees that within a MessageGroupId all messages are delivered
// in order and only one is in-flight at a time. The consumer honours this by
// processing messages sequentially within each Poll call.
//
// Call Poll in a loop (with context cancellation for graceful shutdown):
//
//	for {
//	    if err := consumer.Poll(ctx); err != nil { ... }
//	}
func (c *Consumer) Poll(ctx context.Context) error {
	messages, err := c.receiver.Receive(ctx, c.queueURL, c.maxMessages)
	if err != nil {
		return fmt.Errorf("sqs consumer: receive: %w", err)
	}
	for _, msg := range messages {
		if err := c.process(ctx, msg); err != nil {
			c.logger.ErrorContext(ctx, "process error",
				slog.String("receipt", msg.ReceiptHandle),
				slog.String("error", err.Error()),
			)
		}
	}
	return nil
}

func (c *Consumer) process(ctx context.Context, msg InboundMessage) error {
	evt, err := c.unmarshaler.Unmarshal(msg.Body)
	if err != nil {
		return c.sendToDLQ(ctx, msg, fmt.Errorf("unmarshal: %w", err))
	}

	switch c.handler.Handle(ctx, evt) {
	case ResultOK:
		// DeleteMessage advances the group to the next message.
		return c.receiver.Delete(ctx, c.queueURL, msg.ReceiptHandle)

	case ResultRetry:
		// Do not delete; VisibilityTimeout expires and SQS re-delivers.
		// Optionally shorten the timeout for a faster retry.
		c.logger.WarnContext(ctx, "transient failure; message will be re-delivered",
			slog.String("event_id", evt.EventID),
			slog.Int("receive_count", msg.ApproximateReceiveCount),
		)
		return nil

	case ResultDLQ:
		return c.sendToDLQ(ctx, msg, errors.New("permanent processing failure"))

	default:
		return fmt.Errorf("consumer: unexpected result for receipt %s", msg.ReceiptHandle)
	}
}

func (c *Consumer) sendToDLQ(ctx context.Context, msg InboundMessage, reason error) error {
	c.logger.ErrorContext(ctx, "routing to DLQ",
		slog.String("group", msg.MessageGroupId),
		slog.Int("receive_count", msg.ApproximateReceiveCount),
		slog.String("reason", reason.Error()),
	)
	// Write to DLQ with the original body so consumers can inspect failures.
	dlqMsg := OutboundMessage{
		MessageGroupId:         msg.MessageGroupId,
		MessageDeduplicationId: DeduplicationID(msg.Body),
		Body:                   msg.Body,
	}
	if err := c.dlq.Write(ctx, c.dlqURL, dlqMsg); err != nil {
		return fmt.Errorf("consumer: DLQ write: %w", err)
	}
	// Delete from source so the group advances past the poison pill.
	return c.receiver.Delete(ctx, c.queueURL, msg.ReceiptHandle)
}
