package order

// definido por payment, implementado por orders
type OrderUpdater interface {
	MarkOrderAsPaid(orderID string, paymentID string) error
}
