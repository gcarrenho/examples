package payment

func ProcessPayment(orderID string) Payment {
	return Payment{ID: "p1", OrderID: orderID, Status: "Processed"}
}
