package payment

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
)

// TestService_Charge_HappyPath verifies the EXACT call sequence using uber/mock.
// A hand-written stub can verify "was this called" but NOT "in this exact order with these exact args".
// gomock fails the test if any expected call is skipped OR any unexpected call is made.
func TestService_Charge_HappyPath(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	idem := NewMockidempotencyStore(ctrl)
	repo := NewMockaccountRepository(ctrl)
	rsv  := NewMockreservationStore(ctrl)

	gomock.InOrder(
		idem.EXPECT().Acquire(ctx, "txn-001").Return(true, nil),
		repo.EXPECT().GetAccount(ctx, "acc-alice").Return(int64(10_000), int64(0), nil),
		repo.EXPECT().UpdateAccount(ctx, "acc-alice", int64(9_500), int64(0)).Return(nil),
		idem.EXPECT().Complete(ctx, "txn-001").Return(nil),
	)

	if err := New(idem, repo, rsv).Charge(ctx, "txn-001", "acc-alice", 500); err != nil {
		t.Errorf("Charge() = %v, want nil", err)
	}
}

// TestService_Charge_OCCRetry verifies the OCC retry loop.
// gomock.Times(2) asserts GetAccount is called exactly twice (read → conflict → re-read → success).
func TestService_Charge_OCCRetry(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	idem := NewMockidempotencyStore(ctrl)
	repo := NewMockaccountRepository(ctrl)
	rsv  := NewMockreservationStore(ctrl)

	idem.EXPECT().Acquire(ctx, "txn-retry").Return(true, nil)
	repo.EXPECT().GetAccount(ctx, "acc-1").Return(int64(5_000), int64(0), nil).Times(2)
	// First UpdateAccount returns ErrVersionConflict (another writer committed first)
	gomock.InOrder(
		repo.EXPECT().UpdateAccount(ctx, "acc-1", int64(4_000), int64(0)).Return(ErrVersionConflict),
		repo.EXPECT().UpdateAccount(ctx, "acc-1", int64(4_000), int64(0)).Return(nil),
	)
	idem.EXPECT().Complete(ctx, "txn-retry").Return(nil)

	if err := New(idem, repo, rsv).Charge(ctx, "txn-retry", "acc-1", 1_000); err != nil {
		t.Errorf("Charge() = %v, want nil", err)
	}
}

// TestService_Charge_InsufficientFunds verifies that UpdateAccount is NEVER called on overdraft.
// With a hand-written stub this is hard to assert; with gomock it's implicit — any unexpected
// call to UpdateAccount fails the test automatically.
func TestService_Charge_InsufficientFunds(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	idem := NewMockidempotencyStore(ctrl)
	repo := NewMockaccountRepository(ctrl)
	rsv  := NewMockreservationStore(ctrl)

	idem.EXPECT().Acquire(ctx, "txn-nsf").Return(true, nil)
	repo.EXPECT().GetAccount(ctx, "acc-broke").Return(int64(100), int64(0), nil) // only $1
	// UpdateAccount must NOT be called — gomock will fail the test if it is
	idem.EXPECT().Release(ctx, "txn-nsf").Return(nil)

	err := New(idem, repo, rsv).Charge(ctx, "txn-nsf", "acc-broke", 5_000)
	if !errors.Is(err, ErrInsufficientFunds) {
		t.Errorf("want ErrInsufficientFunds, got %v", err)
	}
}

// TestService_Charge_IdempotentReplay verifies that the network is NOT hit on duplicate requests.
// GetAccount and UpdateAccount must never be called — gomock enforces this without explicit assertions.
func TestService_Charge_IdempotentReplay(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	idem := NewMockidempotencyStore(ctrl)
	repo := NewMockaccountRepository(ctrl)
	rsv  := NewMockreservationStore(ctrl)

	// Idempotency key is already COMPLETED — return false (key exists)
	idem.EXPECT().Acquire(ctx, "txn-dup").Return(false, nil)
	idem.EXPECT().IsCompleted(ctx, "txn-dup").Return(true, nil)
	// repo.EXPECT() — no calls registered; any call would fail the test

	if err := New(idem, repo, rsv).Charge(ctx, "txn-dup", "acc-1", 500); !errors.Is(err, ErrAlreadyProcessed) {
		t.Errorf("want ErrAlreadyProcessed, got %v", err)
	}
}

// TestService_Reserve_HappyPath verifies the reserve flow creates the reservation.
func TestService_Reserve_HappyPath(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	idem := NewMockidempotencyStore(ctrl)
	repo := NewMockaccountRepository(ctrl)
	rsv  := NewMockreservationStore(ctrl)

	idem.EXPECT().Acquire(ctx, "rsv-001").Return(true, nil)
	repo.EXPECT().GetAccount(ctx, "acc-alice").Return(int64(10_000), int64(0), nil)
	rsv.EXPECT().Create(ctx, gomock.Any(), "acc-alice", int64(3_000)).Return(nil)
	idem.EXPECT().Complete(ctx, "rsv-001").Return(nil)

	reservationID, err := New(idem, repo, rsv).Reserve(ctx, "rsv-001", "acc-alice", 3_000)
	if err != nil {
		t.Fatalf("Reserve() = %v, want nil", err)
	}
	if reservationID == "" {
		t.Error("expected non-empty reservationID")
	}
}
