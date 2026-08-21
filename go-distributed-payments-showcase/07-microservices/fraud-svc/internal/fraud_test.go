package fraud

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
)

// TestService_ProcessPayment_Approved verifies the two-phase Reserve→Charge flow.
// gomock.InOrder asserts Reserve is called BEFORE Charge — impossible to express with stubs.
func TestService_ProcessPayment_Approved(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	payments := NewMockpaymentGateway(ctrl)

	// $1,000 — below $5,000 fraud threshold → approved
	gomock.InOrder(
		payments.EXPECT().Reserve(ctx, "rsv:key-001", "acc-alice", int64(100_000)).Return("rsv-abc", nil),
		payments.EXPECT().Charge(ctx, "key-001", "acc-alice", int64(100_000)).Return(nil),
	)

	if err := New(payments).ProcessPayment(ctx, "key-001", "acc-alice", 100_000); err != nil {
		t.Errorf("ProcessPayment() = %v, want nil", err)
	}
}

// TestService_ProcessPayment_FraudBlocked verifies that Reserve IS called but Charge is NOT.
// This is the critical assertion: gomock fails the test if Charge is called unexpectedly,
// giving us fraud-blocking confidence without any explicit "assert not called".
func TestService_ProcessPayment_FraudBlocked(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	payments := NewMockpaymentGateway(ctrl)

	// $6,000 — above $5,000 threshold → fraud detected
	// Reserve IS called (we verify funds before scoring), Charge is NOT registered
	payments.EXPECT().Reserve(ctx, "rsv:key-002", "acc-alice", int64(600_000)).Return("rsv-xyz", nil)
	// No EXPECT for Charge — any unexpected Charge call will fail the test

	err := New(payments).ProcessPayment(ctx, "key-002", "acc-alice", 600_000)
	if !errors.Is(err, ErrFraudDetected) {
		t.Errorf("want ErrFraudDetected, got %v", err)
	}
}

// TestService_ProcessPayment_ReserveFails verifies that Charge is NOT called if Reserve fails.
func TestService_ProcessPayment_ReserveFails(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	payments := NewMockpaymentGateway(ctrl)
	errInsufficientFunds := errors.New("payment: insufficient funds")

	payments.EXPECT().Reserve(ctx, "rsv:key-003", "acc-broke", int64(50_000)).Return("", errInsufficientFunds)
	// Charge is not registered — gomock guarantees it won't be called

	err := New(payments).ProcessPayment(ctx, "key-003", "acc-broke", 50_000)
	if err == nil {
		t.Fatal("want error, got nil")
	}
}
