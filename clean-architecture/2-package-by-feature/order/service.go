package order

func CreateOrder(id string, amount float64) Order {
	return Order{ID: id, Amount: amount}
}
