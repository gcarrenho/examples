package services

import (
	"context"

	"github.com/gcarrenho/hexagonal/good-example/internal/payments/core/model"
	"github.com/gcarrenho/hexagonal/good-example/internal/payments/core/ports"
)

var _ ports.PaymentsService = (*PaymentsService)(nil)

type PaymentsService struct {
	paymentRepo ports.PaymentsRepository
}

func NewPaymentsService(paymentRepo ports.PaymentsRepository) *PaymentsService {
	return &PaymentsService{
		paymentRepo: paymentRepo,
	}
}

func (p *PaymentsService) FindPaymentByID(ctx context.Context, ID string) (model.Payment, error) {
	return p.paymentRepo.GetPaymentByID(ctx, ID)
}
