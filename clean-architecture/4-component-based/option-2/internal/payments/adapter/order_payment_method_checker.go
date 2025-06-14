package adapter

import (
	"github.com/gcarrenho/component-based/option-2/internal/orders/contracts/payment"
	"github.com/gcarrenho/component-based/option-2/internal/payments"
)

var _ payment.PaymentMethodChecker = (*PaymentMethodCheckerAdapter)(nil)

// implement that is needed by payment consumer in this case is order.
type PaymentMethodCheckerAdapter struct {
	paymentSvc *payments.PaymentComponent
}

func NewPaymentMethodCheckerAdapter(paymentSvc *payments.PaymentComponent) payment.PaymentMethodChecker {
	return &PaymentMethodCheckerAdapter{
		paymentSvc: paymentSvc,
	}
}

func (p *PaymentMethodCheckerAdapter) IsPaymentMethodValid(method string) bool {
	// p.paymentSvc.IsValidPaymentMehod()
	return true
}
