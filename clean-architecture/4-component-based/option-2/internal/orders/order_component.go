package orders

type OrderComponent interface {
	// ProcessPayment processes a payment for a given order ID and amount.
	FindOrderByID(orderID string) (string, error)
}
