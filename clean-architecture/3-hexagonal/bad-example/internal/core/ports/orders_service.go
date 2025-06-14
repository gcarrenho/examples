package ports

import "github.com/gcarrenho/hexagonal/bad-example/internal/core/model"

type OrdersService interface {
	FindOrderByID(orderID string) (model.Order, error)
}
