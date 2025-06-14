package services

import "github.com/gcarrenho/package-by-layer/models"

func CreateOrder(id string, amount float64) models.Order {
	return models.Order{ID: id, Amount: amount}
}
