package adapter

import (
	"context"

	"github.com/gcarrenho/hexagonal/good-example/internal/orders/core/ports"
	portPaymentSvc "github.com/gcarrenho/hexagonal/good-example/internal/payments/core/ports"
)

// Implement the contract that order expects
type OrderPaymentAdapter struct {
	paymentService portPaymentSvc.PaymentsService
}

func NewOrderPayment(paymentService portPaymentSvc.PaymentsService) *OrderPaymentAdapter {
	return &OrderPaymentAdapter{
		paymentService: paymentService,
	}
}

func (p *OrderPaymentAdapter) OrderPaymentStatus(ctx context.Context, orderID string) (ports.PaymentDTO, error) {
	return ports.PaymentDTO{}, nil
}
