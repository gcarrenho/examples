package ports

import (
	"context"

	"github.com/gcarrenho/hexagonal/good-example/internal/payments/core/model"
)

type PaymentsRepository interface {
	GetPaymentByID(ctx context.Context, paymentID string) (model.Payment, error)
}
