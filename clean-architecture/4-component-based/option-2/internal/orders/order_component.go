package orders

type OrderComponent interface {
	// ProcessPayment processes a payment for a given order ID and amount.
	GetOrderByID(orderID string) (string, error)
}
