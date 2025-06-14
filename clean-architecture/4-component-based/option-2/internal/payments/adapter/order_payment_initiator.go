package adapter

import (
	"github.com/gcarrenho/component-based/option-2/internal/orders/contracts/payment"
	"github.com/gcarrenho/component-based/option-2/internal/payments"
)

var _ payment.PaymentInitiator = (*PaymentAdapter)(nil)

// implement that is needed by payment consumer in this case is order.
type PaymentAdapter struct {
	paymentSvc *payments.PaymentComponent
}

func NewOrderAdapter(paymentSvc *payments.PaymentComponent) payment.PaymentInitiator {
	return &PaymentAdapter{
		paymentSvc: paymentSvc,
	}
}

func (p *PaymentAdapter) InitPayment(orderID string, amount float64, method string) (paymentID string, err error) {
	// p.paymentSvc.InitPayment()
	return "payment-123", nil
}
