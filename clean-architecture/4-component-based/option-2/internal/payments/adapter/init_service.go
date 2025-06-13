package adapter

import (
	"github.com/gcarrenho/component-based/option-2/internal/orders/contracts/payment"
	"github.com/gcarrenho/component-based/option-2/internal/users"
)

var _ payment.PaymentInitiator = (*PaymentAdapter)(nil)

// implementa lo que necesita el consumidor de user en este caso es payment
type PaymentAdapter struct {
	userService *users.UserComponentImpl
}

func NewOrderAdapter(userService *users.UserComponentImpl) *PaymentAdapter {
	return &PaymentAdapter{
		userService: userService,
	}
}

func (a *PaymentAdapter) InitPayment(orderID string, amount float64, method string) (paymentID string, err error) {
	/*user := a.userService.FindByID(userID)
	return user.Status == "active", nil*/
	return "true", nil
}
