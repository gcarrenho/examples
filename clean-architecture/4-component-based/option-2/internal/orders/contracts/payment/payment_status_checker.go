package payment

type PaymentStatusChecker interface {
	IsPaymentMethodValid(method string) bool
}
