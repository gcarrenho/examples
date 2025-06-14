package order

// defined by payment, implemented by orders
type OrderUpdater interface {
	MarkOrderAsPaid(orderID string, paymentID string) error
}
