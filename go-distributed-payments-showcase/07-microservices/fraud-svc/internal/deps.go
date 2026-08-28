package fraud

import "context"

// ports.go centralises every outbound contract fraud-svc defines.
// Rule: interfaces are defined by the CONSUMER (fraud-svc), never imported from providers.
// When a new dependency is added, a new interface block goes here — fraud.go stays clean.

//go:generate go run go.uber.org/mock/mockgen -source deps.go -destination mocks_test.go -package fraud

// paymentGateway is what fraud-svc needs from payment-svc.
// orders-svc has charger{Charge only}; fraud-svc has {Charge+Reserve} — two independent
// consumer-defined subtypes of payment-svc's ChargeService.
type paymentGateway interface {
	Charge(ctx context.Context, key, accountID string, amountCents int64) error
	Reserve(ctx context.Context, key, accountID string, amountCents int64) (reservationID string, err error)
}

// kycProvider is what fraud-svc would need from a KYC (Know Your Customer) service.
// Uncomment and implement the HTTP client in kyc/client.go when the service exists.
//
// type kycProvider interface {
// 	IsVerified(ctx context.Context, accountID string) (bool, error)
// }

// riskEngine is what fraud-svc would need from a dedicated ML risk scoring service.
//
// type riskEngine interface {
// 	Score(ctx context.Context, req RiskRequest) (score float64, err error)
// }
