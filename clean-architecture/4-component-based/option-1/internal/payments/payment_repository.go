package payments

import "github.com/gcarrenho/component-based/option-1/internal/payments/model"

type paymentRepository interface {
	// GetPaymentByID retrieves a payment by its ID.
	GetPaymentByID(paymentID string) (model.Payment, error)
}
