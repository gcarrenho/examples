package ports

import "github.com/gcarrenho/hexagonal/bad-example/internal/core/model"

type OrdersRepository interface {
	GetOrderByID(orderID string) (model.Order, error)
}
