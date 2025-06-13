package services

import (
	"github.com/gcarrenho/hexagonal/bad-example/internal/core/model"
	"github.com/gcarrenho/hexagonal/bad-example/internal/core/ports"
)

var _ ports.OrdersService = (*OrdersService)(nil)

type OrdersService struct {
	orderRepo ports.OrdersRepository
}

func NewOrdersService(orderRepo ports.OrdersRepository) ports.OrdersService {
	return &OrdersService{
		orderRepo: orderRepo,
	}
}

func (o *OrdersService) FindOrderByID(orderID string) (model.Order, error) {
	return o.orderRepo.GetOrderByID(orderID)
}
