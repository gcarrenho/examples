package ports

import (
	"context"

	"github.com/gcarrenho/hexagonal/bad-example/internal/core/model"
)

type PaymentsService interface {
	FindPaymentByID(ctx context.Context, paymentID string) (model.Payment, error)

	// Orders need to know if a payment is successful or not.
	// So payments define a method to check the status of a payment for an order.
	// But this is not the responsibility of the payment service.
	// This is a bad design because it creates a dependency on the payment service.
	// Instead, the order service must define what it wants for payment.
	// What happens if the payment service changes its implementation?
	// What happens if the model.Payment changes? This would affect orders.
	// This is that we call structure coupling.
	OrderPaymentStatus(ctx context.Context, orderID string) (model.Payment, error)
}
