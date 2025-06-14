package handlers

import (
	"fmt"

	"github.com/gcarrenho/package-by-layer/services"
)

func HandleCreateOrder() {
	order := services.CreateOrder("o1", 100.0)
	fmt.Println("Order created:", order)
}
