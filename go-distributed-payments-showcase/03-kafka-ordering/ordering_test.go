package ordering

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
)

// --- Test doubles ---

type stubSender struct {
	mu       sync.Mutex
	messages map[int32][][]byte // partition → payloads in arrival order
}

func newStubSender() *stubSender {
	return &stubSender{messages: make(map[int32][][]byte)}
}

func (s *stubSender) Send(_ context.Context, _ string, partition int32, _, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages[partition] = append(s.messages[partition], value)
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

type stubDLQ struct {
	count atomic.Int64
}

func (d *stubDLQ) Write(_ context.Context, _ string, _, _ []byte) error {
	d.count.Add(1)
	return nil
}

type stubCommitter struct {
	count atomic.Int64
}

func (c *stubCommitter) Commit(_ context.Context, _ string, _ int32, _ int64) error {
	c.count.Add(1)
	return nil
}

type stubUnmarshaler struct {
	evt PaymentEvent
	err error
}

func (u *stubUnmarshaler) Unmarshal(_ []byte) (PaymentEvent, error) {
	return u.evt, u.err
}

// --- Partition routing ---

// TestPartitionFor_Determinism proves the same accountID always maps to the same
// partition — the ordering guarantee across all producer instances and restarts.
func TestPartitionFor_Determinism(t *testing.T) {
	t.Parallel()
	cases := []struct {
		accountID     string
		numPartitions int32
	}{
		{"acc-001", 16},
		{"acc-abc-xyz", 16},
		{"acc-9999", 16},
		{"acc-001", 1},    // single partition: must always return 0
		{"acc-001", 1024}, // large fleet
	}
	for _, tc := range cases {
		name := fmt.Sprintf("%s/n=%d", tc.accountID, tc.numPartitions)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p1 := PartitionFor(tc.accountID, tc.numPartitions)
			p2 := PartitionFor(tc.accountID, tc.numPartitions)
			if p1 != p2 {
				t.Errorf("non-deterministic: got %d then %d", p1, p2)
			}
			if p1 < 0 || p1 >= tc.numPartitions {
				t.Errorf("out of range: got %d, want [0, %d)", p1, tc.numPartitions)
			}
		})
	}
}

// TestPartitionFor_Distribution verifies SHA-256 spreads accounts uniformly.
func TestPartitionFor_Distribution(t *testing.T) {
	t.Parallel()
	const (
		numPartitions = 8
		numAccounts   = 800
	)
	counts := make([]int, numPartitions)
	for i := range numAccounts {
		p := PartitionFor(fmt.Sprintf("acc-%06d", i), numPartitions)
		counts[p]++
	}
	want := numAccounts / numPartitions
	for p, got := range counts {
		if got < want/2 || got > want*2 {
			t.Errorf("partition %d: %d messages (want ~%d); distribution is skewed", p, got, want)
		}
	}
}

// --- Consumer state machine ---

// TestConsumer_StateMachine drives all result paths through a table.
// Each subtest is isolated (own doubles) and runs in parallel.
func TestConsumer_StateMachine(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		result      ProcessResult
		wantCommits int64
		wantDLQ     int64
	}{
		{"OK commits offset", ResultOK, 1, 0},
		{"Retry skips commit for re-delivery", ResultRetry, 0, 0},
		{"DLQ writes then commits to unblock partition", ResultDLQ, 1, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			handler := &stubHandler{result: tc.result}
			dlq := &stubDLQ{}
			committer := &stubCommitter{}
			c := NewConsumer(handler, dlq, committer, &stubUnmarshaler{}, "dlq-payments", slog.Default())

			if err := c.Process(context.Background(), Message{Topic: "payments", Partition: 0, Offset: 42}); err != nil {
				t.Fatalf("Process: %v", err)
			}
			if got := committer.count.Load(); got != tc.wantCommits {
				t.Errorf("commits = %d, want %d", got, tc.wantCommits)
			}
			if got := dlq.count.Load(); got != tc.wantDLQ {
				t.Errorf("DLQ writes = %d, want %d", got, tc.wantDLQ)
			}
		})
	}
}

// --- Producer routing ---

// TestProducer_Routing proves all events for one account land on exactly one partition.
func TestProducer_Routing(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		accountID string
		events    int
	}{
		{"single event", "acc-001", 1},
		{"ten events same account", "acc-001", 10},
		{"another account", "acc-002", 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sender := newStubSender()
			producer := NewProducer(sender, "payments", 4)
			ctx := context.Background()

			for i := range tc.events {
				evt := PaymentEvent{EventID: fmt.Sprintf("evt-%d", i), AccountID: tc.accountID}
				if err := producer.Publish(ctx, evt, []byte(`{}`)); err != nil {
					t.Fatalf("Publish: %v", err)
				}
			}

			sender.mu.Lock()
			defer sender.mu.Unlock()

			nonEmpty, total := 0, 0
			for _, msgs := range sender.messages {
				if len(msgs) > 0 {
					nonEmpty++
				}
				total += len(msgs)
			}
			if nonEmpty != 1 {
				t.Errorf("messages spread across %d partitions; want 1 for %q", nonEmpty, tc.accountID)
			}
			if total != tc.events {
				t.Errorf("total messages = %d, want %d", total, tc.events)
			}
		})
	}
}
