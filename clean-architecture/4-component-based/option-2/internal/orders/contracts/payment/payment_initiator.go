package payment

// defined by orders, implemented by payment
type PaymentInitiator interface {
	InitPayment(orderID string, amount float64, method string) (paymentID string, err error)
}
