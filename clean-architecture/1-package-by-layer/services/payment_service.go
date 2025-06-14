package services

import "github.com/gcarrenho/package-by-layer/models"

func ProcessPayment(orderID string) models.Payment {
	return models.Payment{ID: "p1", OrderID: orderID, Status: "Processed"}
}
