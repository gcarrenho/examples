package services

import (
	"context"

	"github.com/gcarrenho/hexagonal/bad-example/internal/core/model"
	"github.com/gcarrenho/hexagonal/bad-example/internal/core/ports"
)

var _ ports.PaymentsService = (*PaymentsService)(nil)

type PaymentsService struct {
	paymentRepo ports.PaymentsRepository
}

func NewPaymentsService(paymentRepo ports.PaymentsRepository) ports.PaymentsService {
	return &PaymentsService{
		paymentRepo: paymentRepo,
	}
}

func (p *PaymentsService) FindPaymentByID(ctx context.Context, ID string) (model.Payment, error) {
	return p.paymentRepo.GetPaymentByID(ctx, ID)
}

func (p *PaymentsService) OrderPaymentStatus(ctx context.Context, orderID string) (model.Payment, error) {
	// This method is a bad design because it creates a dependency on the payment service.
	// It should not be here, as it couples the payment service to the order service.
	// Instead, the order service should define what it wants for payment.
	// This is an example of structure coupling.
	return model.Payment{}, nil
}
