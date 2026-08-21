package banking

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
)

func sampleOrder() PaymentOrder {
	return PaymentOrder{
		ID: "ORD001", UETR: "550e8400-e29b-41d4-a716-446655440000",
		InstrID: "INSTR001", EndToEndID: "E2E001",
		Amount:   Money{AmountCents: 150_000, Currency: "EUR"},
		Debtor:   Party{Name: "Alice", IBAN: "DE89370400440532013000", BIC: "DEUTDEDB"},
		Creditor: Party{Name: "Bob", IBAN: "FR7630004000031234567890143", BIC: "BNPAFRPP"},
		Rail:     RailSEPA,
	}
}

func TestEngine_Initiate_SEPASettled(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	sepa, swift, ledger := NewMockRail(ctrl), NewMockRail(ctrl), NewMockLedger(ctrl)
	order := sampleOrder()

	// Exact sequence: GetBalance → Debit → SEPA.Send → Settle. swift.Send must never be called.
	gomock.InOrder(
		ledger.EXPECT().GetBalance(ctx, order.Debtor.IBAN).Return(int64(1_000_000), int64(0), nil),
		ledger.EXPECT().Debit(ctx, order.Debtor.IBAN, order.Amount.AmountCents, int64(0)).Return(nil),
		sepa.EXPECT().Send(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, o PaymentOrder) (PaymentOrder, error) {
			o.Status = StatusSettled
			return o, nil
		}),
		ledger.EXPECT().Settle(ctx, order.ID).Return(nil),
	)

	result, err := New(sepa, swift, ledger).Initiate(ctx, order)
	if err != nil || result.Status != StatusSettled {
		t.Errorf("Initiate() = status=%q err=%v, want ACSC/nil", result.Status, err)
	}
	if result.SettledAt == nil {
		t.Error("expected SettledAt to be set")
	}
}

func TestEngine_Initiate_Rejected(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	sepa, swift, ledger := NewMockRail(ctrl), NewMockRail(ctrl), NewMockLedger(ctrl)
	order := sampleOrder()

	gomock.InOrder(
		ledger.EXPECT().GetBalance(ctx, order.Debtor.IBAN).Return(int64(1_000_000), int64(0), nil),
		ledger.EXPECT().Debit(ctx, order.Debtor.IBAN, order.Amount.AmountCents, int64(0)).Return(nil),
		sepa.EXPECT().Send(ctx, gomock.Any()).Return(PaymentOrder{}, ErrRejectedByBank),
		ledger.EXPECT().Reverse(ctx, order.ID).Return(nil), // funds released; Settle must NOT be called
	)

	if _, err := New(sepa, swift, ledger).Initiate(ctx, order); !errors.Is(err, ErrRejectedByBank) {
		t.Errorf("want ErrRejectedByBank, got %v", err)
	}
}

func TestEngine_Initiate_InsufficientBalance(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	sepa, swift, ledger := NewMockRail(ctrl), NewMockRail(ctrl), NewMockLedger(ctrl)
	order := sampleOrder()
	order.Amount.AmountCents = 999_999_999

	// Only GetBalance is called — Debit and rail.Send must never be reached
	ledger.EXPECT().GetBalance(ctx, order.Debtor.IBAN).Return(int64(100), int64(0), nil)

	if _, err := New(sepa, swift, ledger).Initiate(ctx, order); !errors.Is(err, ErrInsufficientBalance) {
		t.Errorf("want ErrInsufficientBalance, got %v", err)
	}
}

func TestEngine_Initiate_RailError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	sepa, swift, ledger := NewMockRail(ctrl), NewMockRail(ctrl), NewMockLedger(ctrl)
	order := sampleOrder()

	gomock.InOrder(
		ledger.EXPECT().GetBalance(ctx, order.Debtor.IBAN).Return(int64(1_000_000), int64(0), nil),
		ledger.EXPECT().Debit(ctx, order.Debtor.IBAN, order.Amount.AmountCents, int64(0)).Return(nil),
		sepa.EXPECT().Send(ctx, gomock.Any()).Return(PaymentOrder{}, ErrRailUnavailable),
		ledger.EXPECT().Reverse(ctx, order.ID).Return(nil),
	)

	if _, err := New(sepa, swift, ledger).Initiate(ctx, order); !errors.Is(err, ErrRailUnavailable) {
		t.Errorf("want ErrRailUnavailable, got %v", err)
	}
}

// TestEngine_Initiate_OCCRetry proves the debit loop re-reads the balance and
// succeeds on the second attempt after a version conflict — the OCC pattern from Case 02.
func TestEngine_Initiate_OCCRetry(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	sepa, swift, ledger := NewMockRail(ctrl), NewMockRail(ctrl), NewMockLedger(ctrl)
	order := sampleOrder()

	gomock.InOrder(
		ledger.EXPECT().GetBalance(ctx, order.Debtor.IBAN).Return(int64(1_000_000), int64(0), nil),
		ledger.EXPECT().Debit(ctx, order.Debtor.IBAN, order.Amount.AmountCents, int64(0)).Return(ErrVersionConflict),
		ledger.EXPECT().GetBalance(ctx, order.Debtor.IBAN).Return(int64(1_000_000), int64(1), nil),
		ledger.EXPECT().Debit(ctx, order.Debtor.IBAN, order.Amount.AmountCents, int64(1)).Return(nil),
		sepa.EXPECT().Send(ctx, gomock.Any()).Return(order, nil),
	)

	if _, err := New(sepa, swift, ledger).Initiate(ctx, order); err != nil {
		t.Errorf("Initiate() = %v, want nil after OCC retry", err)
	}
}

// TestEngine_Initiate_OCCRetryExhausted verifies the loop gives up after maxOCCRetries.
func TestEngine_Initiate_OCCRetryExhausted(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	sepa, swift, ledger := NewMockRail(ctrl), NewMockRail(ctrl), NewMockLedger(ctrl)
	order := sampleOrder()

	// GetBalance + Debit called exactly maxOCCRetries times, always conflicting
	ledger.EXPECT().GetBalance(ctx, order.Debtor.IBAN).Return(int64(1_000_000), int64(0), nil).Times(maxOCCRetries)
	ledger.EXPECT().Debit(ctx, order.Debtor.IBAN, order.Amount.AmountCents, int64(0)).Return(ErrVersionConflict).Times(maxOCCRetries)

	if _, err := New(sepa, swift, ledger).Initiate(ctx, order); !errors.Is(err, ErrRetryExhausted) {
		t.Errorf("want ErrRetryExhausted, got %v", err)
	}
}

// TestEngine_WithBackoff_JitteredBackoff verifies the Engine sleeps using the configured
// backoff function on each OCC conflict, and that JitteredBackoff never exceeds attempt*base.
func TestEngine_WithBackoff_JitteredBackoff(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	sepa, swift, ledger := NewMockRail(ctrl), NewMockRail(ctrl), NewMockLedger(ctrl)
	order := sampleOrder()

	gomock.InOrder(
		ledger.EXPECT().GetBalance(ctx, order.Debtor.IBAN).Return(int64(1_000_000), int64(0), nil),
		ledger.EXPECT().Debit(ctx, order.Debtor.IBAN, order.Amount.AmountCents, int64(0)).Return(ErrVersionConflict),
		ledger.EXPECT().GetBalance(ctx, order.Debtor.IBAN).Return(int64(1_000_000), int64(1), nil),
		ledger.EXPECT().Debit(ctx, order.Debtor.IBAN, order.Amount.AmountCents, int64(1)).Return(nil),
		sepa.EXPECT().Send(ctx, gomock.Any()).Return(order, nil),
	)

	engine := New(sepa, swift, ledger, WithBackoff(JitteredBackoff(5*time.Millisecond)))
	start := time.Now()
	if _, err := engine.Initiate(ctx, order); err != nil {
		t.Fatalf("Initiate() = %v, want nil", err)
	}
	// One retry with base=5ms → sleep is in [0, 5ms). Assert we didn't block excessively.
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Errorf("unexpectedly long backoff sleep: %v", elapsed)
	}
}

func TestJitteredBackoff_NeverExceedsBound(t *testing.T) {
	t.Parallel()
	backoff := JitteredBackoff(10 * time.Millisecond)
	for attempt := 1; attempt <= 5; attempt++ {
		for range 20 {
			d := backoff(attempt)
			max := time.Duration(attempt) * 10 * time.Millisecond
			if d < 0 || d >= max {
				t.Errorf("attempt=%d: backoff=%v out of bounds [0, %v)", attempt, d, max)
			}
		}
	}
}

// TestEngine_DebitWithOCC_ContextCancelled verifies the retry loop aborts immediately
// when the context is already cancelled, rather than sleeping through the backoff.
func TestEngine_DebitWithOCC_ContextCancelled(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel

	sepa, swift, ledger := NewMockRail(ctrl), NewMockRail(ctrl), NewMockLedger(ctrl)
	order := sampleOrder()

	ledger.EXPECT().GetBalance(ctx, order.Debtor.IBAN).Return(int64(1_000_000), int64(0), nil)
	ledger.EXPECT().Debit(ctx, order.Debtor.IBAN, order.Amount.AmountCents, int64(0)).Return(ErrVersionConflict)
	// No further GetBalance/Debit calls — ctx.Done() short-circuits the retry loop

	engine := New(sepa, swift, ledger, WithBackoff(JitteredBackoff(time.Hour))) // huge backoff to prove we never sleep
	if _, err := engine.Initiate(ctx, order); !errors.Is(err, context.Canceled) {
		t.Errorf("want context.Canceled, got %v", err)
	}
}
