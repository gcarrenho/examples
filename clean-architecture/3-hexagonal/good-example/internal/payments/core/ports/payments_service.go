package ports

import (
	"context"

	"github.com/gcarrenho/hexagonal/good-example/internal/payments/core/model"
)

type PaymentsService interface {
	FindPaymentByID(ctx context.Context, paymentID string) (model.Payment, error)
}
