package payment

// definido por orders, implementado por payment
type PaymentInitiator interface {
	InitPayment(orderID string, amount float64, method string) (paymentID string, err error)
	//IsPaymentMethodValid(method string) bool
	//GetPaymentStatus(orderID string) (status string, err error)
	//CancelPayment(paymentID string) error
}
