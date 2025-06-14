package orders

import "github.com/gcarrenho/component-based/option-2/internal/orders/model"

type OrderComponent interface {
	FindOrderByID(orderID string) (model.Order, error)
}
