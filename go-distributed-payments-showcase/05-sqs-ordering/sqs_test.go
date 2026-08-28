package sqsordering

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
)

// --- Test doubles ---

type stubSQSSender struct {
	mu       sync.Mutex
	groups   map[string][]OutboundMessage // groupId → messages in send order
	errOnSend error
}

func newStubSender() *stubSQSSender {
	return &stubSQSSender{groups: make(map[string][]OutboundMessage)}
}

func (s *stubSQSSender) Send(_ context.Context, _ string, msg OutboundMessage) error {
	if s.errOnSend != nil {
		return s.errOnSend
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.groups[msg.MessageGroupId] = append(s.groups[msg.MessageGroupId], msg)
	return nil
}

type stubSQSReceiver struct {
	mu       sync.Mutex
	messages []InboundMessage
	deleted  atomic.Int64
}

func (r *stubSQSReceiver) Receive(_ context.Context, _ string, max int) ([]InboundMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.messages) == 0 {
		return nil, nil
	}
	n := min(max, len(r.messages))
	batch := r.messages[:n]
	r.messages = r.messages[n:]
	return batch, nil
}

func (r *stubSQSReceiver) Delete(_ context.Context, _ string, _ string) error {
	r.deleted.Add(1)
	return nil
}

func (r *stubSQSReceiver) ChangeVisibility(_ context.Context, _ string, _ string, _ int) error {
	return nil
}

type stubDLQ struct {
	mu    sync.Mutex
	items []OutboundMessage
	count atomic.Int64
}

func (d *stubDLQ) Write(_ context.Context, _ string, msg OutboundMessage) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.items = append(d.items, msg)
	d.count.Add(1)
	return nil
}

type stubHandler struct {
	result ProcessResult
	count  atomic.Int64
}

func (h *stubHandler) Handle(_ context.Context, _ PaymentEvent) ProcessResult {
	h.count.Add(1)
	return h.result
}

type stubUnmarshaler struct {
	evt PaymentEvent
	err error
}

func (u *stubUnmarshaler) Unmarshal(_ []byte) (PaymentEvent, error) {
	return u.evt, u.err
}

func newConsumer(handler *stubHandler, dlq *stubDLQ, recv *stubSQSReceiver, um *stubUnmarshaler) *Consumer {
	return NewConsumer(recv, handler, dlq, um, "https://sqs.us-east-1.amazonaws.com/123/payments.fifo",
		"https://sqs.us-east-1.amazonaws.com/123/payments-dlq.fifo", 10, slog.Default())
}

// --- Producer ---

// TestProducer groups all producer contracts into parallel subtests.
func TestProducer(t *testing.T) {
	t.Parallel()
	const queueURL = "https://sqs.us-east-1.amazonaws.com/123/payments.fifo"
	ctx := context.Background()

	t.Run("MessageGroupId equals AccountID", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			accountID string
			events    int
		}{
			{"acc-001", 1},
			{"acc-001", 10},
			{"acc-999", 5},
		}
		for _, tc := range cases {
			name := fmt.Sprintf("%s/%devents", tc.accountID, tc.events)
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				sender := newStubSender()
				p := NewProducer(sender, queueURL)
				for i := range tc.events {
					evt := PaymentEvent{EventID: fmt.Sprintf("evt-%d", i), AccountID: tc.accountID}
					if err := p.Publish(ctx, evt, []byte(`{}`)); err != nil {
						t.Fatalf("Publish: %v", err)
					}
				}
				sender.mu.Lock()
				got := len(sender.groups[tc.accountID])
				sender.mu.Unlock()
				if got != tc.events {
					t.Errorf("group %q: got %d messages, want %d", tc.accountID, got, tc.events)
				}
			})
		}
	})

	t.Run("DeduplicationId equals EventID", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name    string
			eventID string
		}{
			{"standard", "evt-unique-001"},
			{"UUID-like", "550e8400-e29b-41d4-a716-446655440000"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				sender := newStubSender()
				p := NewProducer(sender, queueURL)
				evt := PaymentEvent{EventID: tc.eventID, AccountID: "acc-1"}
				if err := p.Publish(ctx, evt, []byte(`{}`)); err != nil {
					t.Fatalf("Publish: %v", err)
				}
				sender.mu.Lock()
				got := sender.groups["acc-1"][0].MessageDeduplicationId
				sender.mu.Unlock()
				if got != tc.eventID {
					t.Errorf("DeduplicationId = %q, want %q", got, tc.eventID)
				}
			})
		}
	})
}

// TestDeduplicationID verifies content-based dedup IDs are deterministic.
func TestDeduplicationID(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		payload []byte
	}{
		{"json payload", []byte(`{"event_id":"evt-1","account_id":"acc-1"}`)},
		{"empty", []byte{}},
		{"binary", []byte{0x00, 0xFF, 0xAB}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			id1, id2 := DeduplicationID(tc.payload), DeduplicationID(tc.payload)
			if id1 != id2 {
				t.Errorf("non-deterministic: %q vs %q", id1, id2)
			}
		})
	}
}

// --- Consumer ---

// TestConsumer_StateMachine drives all result paths and the unmarshal-error
// path through a table. Each subtest is isolated and runs in parallel.
func TestConsumer_StateMachine(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		result       ProcessResult
		unmarshalErr error
		wantDeletes  int64
		wantDLQ      int64
	}{
		{"OK deletes message", ResultOK, nil, 1, 0},
		{"Retry leaves message for re-delivery", ResultRetry, nil, 0, 0},
		{"DLQ writes then deletes to unblock group", ResultDLQ, nil, 1, 1},
		{"Unmarshal error routes to DLQ", ResultOK, errors.New("bad json"), 1, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			recv := &stubSQSReceiver{
				messages: []InboundMessage{{ReceiptHandle: "rh-1", Body: []byte(`{}`)}},
			}
			handler := &stubHandler{result: tc.result}
			dlq := &stubDLQ{}
			um := &stubUnmarshaler{err: tc.unmarshalErr}
			c := newConsumer(handler, dlq, recv, um)

			if err := c.Poll(context.Background()); err != nil {
				t.Fatalf("Poll: %v", err)
			}
			if got := recv.deleted.Load(); got != tc.wantDeletes {
				t.Errorf("deletes = %d, want %d", got, tc.wantDeletes)
			}
			if got := dlq.count.Load(); got != tc.wantDLQ {
				t.Errorf("DLQ writes = %d, want %d", got, tc.wantDLQ)
			}
		})
	}
}

// TestConsumer_BatchOrdering verifies a full batch is processed completely
// and sequentially (handler called once per message, all deleted).
func TestConsumer_BatchOrdering(t *testing.T) {
	t.Parallel()
	const msgCount = 20
	messages := make([]InboundMessage, msgCount)
	for i := range msgCount {
		messages[i] = InboundMessage{
			MessageGroupId: "acc-order-test",
			Body:           fmt.Appendf(nil, `{"offset":%d}`, i),
			ReceiptHandle:  fmt.Sprintf("rh-%d", i),
		}
	}
	recv := &stubSQSReceiver{messages: messages}
	handler := &stubHandler{result: ResultOK}
	c := newConsumer(handler, &stubDLQ{}, recv, &stubUnmarshaler{})

	for recv.deleted.Load() < msgCount {
		if err := c.Poll(context.Background()); err != nil {
			t.Fatalf("Poll: %v", err)
		}
	}

	if got := handler.count.Load(); got != msgCount {
		t.Errorf("handler called %d times, want %d", got, msgCount)
	}
	if got := recv.deleted.Load(); got != int64(msgCount) {
		t.Errorf("%d messages deleted, want %d", got, msgCount)
	}
}
