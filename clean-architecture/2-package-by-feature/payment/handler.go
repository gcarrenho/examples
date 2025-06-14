package payment

import "fmt"

func HandlePayment() {
	payment := ProcessPayment("o1")
	fmt.Println("Payment processed:", payment)
}
