package ports

import (
	"context"

	"github.com/gcarrenho/hexagonal/bad-example/internal/core/model"
)

type PaymentsRepository interface {
	GetPaymentByID(ctx context.Context, paymentID string) (model.Payment, error)
}
