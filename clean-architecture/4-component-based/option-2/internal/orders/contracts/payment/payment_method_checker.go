package payment

type PaymentMethodChecker interface {
	IsPaymentMethodValid(method string) bool
}
