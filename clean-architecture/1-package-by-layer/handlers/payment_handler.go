package handlers

import (
	"fmt"

	"github.com/gcarrenho/package-by-layer/services"
)

func HandlePayment() {
	payment := services.ProcessPayment("o1")
	fmt.Println("Payment processed:", payment)
}
