// El componente debe exponerse solo por su punto de entrada, y ser responsable de crear sus propias dependencias internas.
package payments

import "database/sql"

var _ PaymentComponent = (*PaymentComponetImpl)(nil)

type PaymentComponetImpl struct {
	paymentRepository paymentRepository
}

type Deps struct {
	DB *sql.DB
	//Logger Logger
}

// NewPaymentComponent creates a new instance of PaymentComponet.
func NewPaymentComponentImpl(deps Deps) *PaymentComponetImpl {
	repo := newPaymentRepositoryImpl(deps.DB) // uso interno, no exportado
	return &PaymentComponetImpl{paymentRepository: repo}
}

// GetPaymentByID retrieves a payment by its ID.
func (c *PaymentComponetImpl) ProcessPayment(orderID string, amount float64) (string, error) {
	p, err := c.paymentRepository.GetPaymentByID(orderID)
	if err != nil {
		return "", err
	}
	return p.ID, nil
}
