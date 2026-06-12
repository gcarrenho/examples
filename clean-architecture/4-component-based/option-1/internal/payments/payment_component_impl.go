// El componente debe exponerse solo por su punto de entrada, y ser responsable de crear sus propias dependencias internas.
package payments

import (
	"database/sql"
)

var _ PaymentComponent = (*PaymentComponentImpl)(nil)

type PaymentComponentImpl struct {
	paymentRepository paymentRepository
}

type Deps struct {
	DB *sql.DB
	//KafkaProd KafkaProducer // interfaz para el productor Kafka
}

// NewPaymentComponent creates a new instance of PaymentComponent.
func NewPaymentComponentImpl(deps Deps) *PaymentComponentImpl {
	repo := newPaymentRepositoryImpl(deps.DB)             // uso interno, no exportado
	return &PaymentComponentImpl{paymentRepository: repo} //controller: deps.Controller

}

// GetPaymentByID retrieves a payment by its ID.
func (c *PaymentComponentImpl) ProcessPayment(orderID string, amount float64) (string, error) {
	p, err := c.paymentRepository.GetPaymentByID(orderID)
	if err != nil {
		return "", err
	}
	return p.ID, nil
}
