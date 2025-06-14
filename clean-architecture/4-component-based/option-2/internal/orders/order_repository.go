package orders

import "github.com/gcarrenho/component-based/option-2/internal/orders/model"

type orderRepository interface {
	GetOrderByID(paymentID string) (model.Order, error)
}
