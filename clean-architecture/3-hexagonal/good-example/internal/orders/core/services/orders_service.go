package services

import (
	"github.com/gcarrenho/hexagonal/good-example/internal/orders/core/model"
	"github.com/gcarrenho/hexagonal/good-example/internal/orders/core/ports"
)

var _ ports.OrdersService = (*OrdersService)(nil)

type OrdersService struct {
	orderRepo ports.OrdersRepository
}

func NewOrdersService(orderRepo ports.OrdersRepository) *OrdersService {
	return &OrdersService{
		orderRepo: orderRepo,
	}
}

func (o *OrdersService) FindOrderByID(orderID string) (model.Order, error) {
	// Here we would typically call the OrderRepo to fetch the order by ID.
	// For this example, we will return a dummy order.
	return o.orderRepo.GetOrderByID(orderID)
}
