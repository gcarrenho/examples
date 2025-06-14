package ports

import "github.com/gcarrenho/hexagonal/good-example/internal/orders/core/model"

type OrdersService interface {
	FindOrderByID(orderID string) (model.Order, error)
}
