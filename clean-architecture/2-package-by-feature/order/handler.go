package order

import "fmt"

func HandleCreateOrder() {
	order := CreateOrder("o1", 100.0)
	fmt.Println("Order created:", order)
}
