package visanet_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	authorization "github.com/examples/go-distributed-payments-showcase/09-psp-card-acquirer/internal"
	"github.com/examples/go-distributed-payments-showcase/09-psp-card-acquirer/internal/visanet"
)

func TestGateway_Authorize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		de39     string
		de38     string
		wantErr  error
		wantCode string
	}{
		{"DE39=00 → approved", "00", "AUTH01", nil, "AUTH01"},
		{"DE39=51 → insufficient funds", "51", "", authorization.ErrInsufficientFunds, ""},
		{"DE39=91 → network unavailable", "91", "", authorization.ErrNetworkUnavailable, ""},
		{"DE39=54 → expired card", "54", "", authorization.ErrExpiredCard, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/authorize" || r.Method != http.MethodPost {
					t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]string{
					"mti":  "0110",
					"de37": "RRN000000001",
					"de38": tc.de38,
					"de39": tc.de39,
				})
			}))
			defer srv.Close()

			gw := visanet.New(srv.URL, srv.Client())
			txn := authorization.CardTransaction{
				ID: "txn001234", IdempotencyKey: "merchant-order-001",
				AmountCents: 5000, CurrencyCode: "032",
				CardBIN: "411111", CardLast4: "1111",
				MerchantID: "MERCHANT000001234", TerminalID: "TERM0001",
			}

			result, err := gw.Authorize(context.Background(), txn)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("error = %v, want %v", err, tc.wantErr)
			}
			if tc.wantCode != "" && result.AuthCode != tc.wantCode {
				t.Errorf("AuthCode = %q, want %q", result.AuthCode, tc.wantCode)
			}
			if tc.wantErr == nil && result.Status != authorization.StatusApproved {
				t.Errorf("Status = %q, want APPROVED", result.Status)
			}
		})
	}
}
