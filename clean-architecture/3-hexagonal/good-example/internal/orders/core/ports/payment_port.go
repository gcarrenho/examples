package ports

import "context"

// OrderPaymentChecker defines the interface for checking the payment status of an order.
// It is used to decouple the payment status checking logic from the order processing logic.
// This allows for flexibility in how payment status is checked, whether through an external service,
type PaymentDTO struct {
	Status string
}

type OrderPaymentChecker interface {
	OrderPaymentStatus(ctx context.Context, orderID string) (PaymentDTO, error)
}
