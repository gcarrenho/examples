package ports

import "github.com/gcarrenho/hexagonal/good-example/internal/orders/core/model"

type OrdersRepository interface {
	GetOrderByID(orderID string) (model.Order, error)
}
