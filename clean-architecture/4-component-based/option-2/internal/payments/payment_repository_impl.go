package payments

import (
	"database/sql"

	"github.com/gcarrenho/component-based/option-2/internal/payments/domain"
)

var _ paymentRepository = (*paymentRepositoryImpl)(nil)

type paymentRepositoryImpl struct {
	//DB
	DB *sql.DB
}

func newPaymentRepositoryImpl(db *sql.DB) *paymentRepositoryImpl {
	return &paymentRepositoryImpl{
		DB: db,
	}
}

func (r *paymentRepositoryImpl) GetPaymentByID(paymentID string) (domain.Payment, error) {
	return domain.Payment{}, nil // This is a placeholder implementation.
}
