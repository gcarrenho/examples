package authorization

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestAuthorizerService_Approve(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	net  := NewMocknetwork(ctrl)
	idem := NewMockidempotencyStore(ctrl)

	txn := CardTransaction{
		ID: "txn123456", IdempotencyKey: "merchant-order-001",
		AmountCents: 5_000, CurrencyCode: "032",
		CardBIN: "411111", CardBrand: BrandVisa,
		MerchantID: "MERCHANT000001234", TerminalID: "TERM0001",
	}
	approvedTxn := txn
	approvedTxn.Status = StatusApproved
	approvedTxn.AuthCode = "AUTH01"
	approvedTxn.NetworkRef = "RRN000000001"

	// gomock.InOrder enforces the exact sequence: Acquire → Authorize → Complete
	gomock.InOrder(
		idem.EXPECT().Acquire(ctx, "merchant-order-001").Return(true, nil),
		net.EXPECT().Authorize(ctx, txn).Return(approvedTxn, nil),
		idem.EXPECT().Complete(ctx, "merchant-order-001", "AUTH01", "RRN000000001").Return(nil),
	)

	result, err := NewService(net, idem).Authorize(ctx, txn)
	if err != nil {
		t.Fatalf("Authorize() = %v, want nil", err)
	}
	if result.AuthCode != "AUTH01" {
		t.Errorf("AuthCode = %q, want AUTH01", result.AuthCode)
	}
}

// TestAuthorizerService_NetworkError verifies Release is called on network failure.
// With stubs you'd add a boolean flag; with gomock it's a first-class expectation.
func TestAuthorizerService_NetworkError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	net  := NewMocknetwork(ctrl)
	idem := NewMockidempotencyStore(ctrl)
	txn  := CardTransaction{ID: "t1", IdempotencyKey: "key-1", CardBrand: BrandVisa}

	gomock.InOrder(
		idem.EXPECT().Acquire(ctx, "key-1").Return(true, nil),
		net.EXPECT().Authorize(ctx, txn).Return(CardTransaction{}, ErrNetworkUnavailable),
		idem.EXPECT().Release(ctx, "key-1").Return(nil),
	)

	if _, err := NewService(net, idem).Authorize(ctx, txn); !errors.Is(err, ErrNetworkUnavailable) {
		t.Errorf("want ErrNetworkUnavailable, got %v", err)
	}
}

func TestAuthorizerService_IdempotentReplay(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	net  := NewMocknetwork(ctrl)
	idem := NewMockidempotencyStore(ctrl)
	txn  := CardTransaction{ID: "t2", IdempotencyKey: "dup-key", CardBrand: BrandVisa}

	idem.EXPECT().Acquire(ctx, "dup-key").Return(false, nil)
	idem.EXPECT().GetCompleted(ctx, "dup-key").Return("CACHED", "RRN-CACHED", true, nil)

	result, err := NewService(net, idem).Authorize(ctx, txn)
	if err != nil || result.AuthCode != "CACHED" {
		t.Errorf("replay: err=%v authCode=%q, want nil/CACHED", err, result.AuthCode)
	}
}

// TestNetworkRouter_BrandRouting verifies only the correct network is called per BIN.
func TestNetworkRouter_BrandRouting(t *testing.T) {
	t.Parallel()
	cases := []struct {
		bin         string
		wantNetwork string
	}{
		{"411111", "visa"},
		{"512345", "mc"},
		{"222100", "mc"},
		{"378282", "fallback"}, // AmEx not registered
	}
	for _, tc := range cases {
		t.Run(tc.bin, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			ctx := context.Background()

			visa, mc, fallback := NewMocknetwork(ctrl), NewMocknetwork(ctrl), NewMocknetwork(ctrl)
			txn := CardTransaction{CardBIN: tc.bin, CardBrand: BrandFromBIN(tc.bin)}
			want := txn
			want.AuthCode = tc.wantNetwork

			switch tc.wantNetwork {
			case "visa":
				visa.EXPECT().Authorize(ctx, txn).Return(want, nil)
			case "mc":
				mc.EXPECT().Authorize(ctx, txn).Return(want, nil)
			case "fallback":
				fallback.EXPECT().Authorize(ctx, txn).Return(want, nil)
			}

			router := NewNetworkRouter(fallback).Register(BrandVisa, visa).Register(BrandMastercard, mc)
			resp, err := router.Authorize(ctx, txn)
			if err != nil || resp.AuthCode != tc.wantNetwork {
				t.Errorf("bin=%s: authCode=%q err=%v", tc.bin, resp.AuthCode, err)
			}
		})
	}
}

// TestFilter_ExactlyOnce_Concurrent keeps the concurrency proof using real memory store —
// mocks serialize calls and cannot model concurrent atomicity correctly.
func TestFilter_ExactlyOnce_Concurrent(t *testing.T) {
	t.Parallel()
	const goroutines = 1000

	idem := NewMemoryIdempotency()
	svc  := NewService(&countingNetwork{}, idem)
	ctx  := context.Background()
	txn  := CardTransaction{ID: "txn1234567890", IdempotencyKey: "txn-concurrent"}

	var winners atomic.Int64
	var wg sync.WaitGroup
	start := make(chan struct{})

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if _, err := svc.Authorize(ctx, txn); err == nil {
				winners.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if got := winners.Load(); got != 1 {
		t.Errorf("exactly 1 winner expected; got %d", got)
	}
}

type countingNetwork struct{ calls atomic.Int64 }

func (n *countingNetwork) Authorize(_ context.Context, txn CardTransaction) (CardTransaction, error) {
	n.calls.Add(1)
	txn.Status, txn.AuthCode, txn.NetworkRef = StatusApproved, "AUTH-OK", "RRN-OK"
	return txn, nil
}
