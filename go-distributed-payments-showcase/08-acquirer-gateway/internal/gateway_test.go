package gateway

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
)

// TestRouter_RoutesToCorrectGateway verifies that the correct Gateway.Authorize is called per country.
// The unregistered gateway's Authorize must never be called — gomock enforces this.
func TestRouter_RoutesToCorrectGateway(t *testing.T) {
	t.Parallel()

	cases := []struct {
		country     string
		wantNetwork string // which mock should receive the Authorize call
	}{
		{"AR", "prisma"},
		{"BR", "cielo"},
		{"DE", "fallback"}, // not in routes → fallback
		{"US", "fallback"},
	}

	for _, tc := range cases {
		t.Run(tc.country+"/"+tc.wantNetwork, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			ctx := context.Background()

			prisma   := NewMockGateway(ctrl)
			cielo    := NewMockGateway(ctrl)
			fallback := NewMockGateway(ctrl)

			req := AuthRequest{CardCountry: tc.country, AmountCents: 5_000, Currency: "USD"}
			wantResp := AuthResponse{Approved: true, AuthCode: tc.wantNetwork + "-AUTH"}

			// Only the expected gateway gets a call — all others must stay silent
			switch tc.wantNetwork {
			case "prisma":
				prisma.EXPECT().Authorize(ctx, req).Return(wantResp, nil)
			case "cielo":
				cielo.EXPECT().Authorize(ctx, req).Return(wantResp, nil)
			case "fallback":
				fallback.EXPECT().Authorize(ctx, req).Return(wantResp, nil)
			}

			router := NewRouter(fallback, map[string]Gateway{"AR": prisma, "BR": cielo})
			resp, err := router.Authorize(ctx, req)
			if err != nil {
				t.Fatalf("Authorize() = %v", err)
			}
			if resp.AuthCode != wantResp.AuthCode {
				t.Errorf("routed to wrong gateway: got authCode %q", resp.AuthCode)
			}
		})
	}
}

// TestCircuitBreaker_OpensAfterThreshold verifies the circuit opens and blocks subsequent calls.
func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	t.Parallel()
	const threshold = 3
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	underlying := NewMockGateway(ctrl)
	cb := WithCircuitBreaker(underlying, threshold)
	req := AuthRequest{CardCountry: "AR", AmountCents: 1_000}

	// Exactly `threshold` calls reach the underlying gateway, each returning an error
	underlying.EXPECT().Authorize(ctx, req).Return(AuthResponse{}, ErrDeclined).Times(threshold)

	for i := range threshold {
		if _, err := cb.Authorize(ctx, req); !errors.Is(err, ErrDeclined) {
			t.Fatalf("call %d: want ErrDeclined, got %v", i+1, err)
		}
	}

	// Circuit is now open — underlying must NOT be called again
	_, err := cb.Authorize(ctx, req)
	if !errors.Is(err, ErrAcquirerUnavailable) {
		t.Errorf("open circuit: want ErrAcquirerUnavailable, got %v", err)
	}
}

// TestCircuitBreaker_ResetsOnSuccess verifies consecutive successes keep the counter at zero.
func TestCircuitBreaker_ResetsOnSuccess(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	ctx := context.Background()
	req := AuthRequest{CardCountry: "AR"}

	underlying := NewMockGateway(ctrl)
	underlying.EXPECT().Authorize(ctx, req).Return(AuthResponse{Approved: true}, nil).Times(5)

	cb := WithCircuitBreaker(underlying, 3)
	for range 5 {
		if _, err := cb.Authorize(ctx, req); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}
