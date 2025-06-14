package payments

type PaymentComponent interface {
	// ProcessPayment processes a payment for a given order ID and amount.
	ProcessPayment(orderID string, amount float64) (string, error)
}
