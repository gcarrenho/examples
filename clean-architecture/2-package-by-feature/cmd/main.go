package main

import (
	"fmt"

	"github.com/gcarrenho/package-by-feature/order"
	"github.com/gcarrenho/package-by-feature/payment"
)

func main() {
	fmt.Println("Running server...")
	order.HandleCreateOrder()
	payment.HandlePayment()
}
