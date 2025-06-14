package orders

import (
	"database/sql"

	"github.com/gcarrenho/component-based/option-2/internal/orders/model"
)

var _ OrderComponent = (*OrderComponentImpl)(nil)

type OrderComponentImpl struct {
	orderRepo orderRepository // uso interno, no exportado
}

type Deps struct {
	DB *sql.DB
	//Mailer MailService // interfaz que envía emails
}

func NewOrderComponentImpl(deps Deps) OrderComponent {
	repo := newOrderRepositoryImpl(deps.DB)
	return &OrderComponentImpl{orderRepo: repo}

}

// GetPaymentByID retrieves a payment by its ID.
func (o *OrderComponentImpl) FindOrderByID(orderID string) (model.Order, error) {
	return o.orderRepo.GetOrderByID(orderID)
}
