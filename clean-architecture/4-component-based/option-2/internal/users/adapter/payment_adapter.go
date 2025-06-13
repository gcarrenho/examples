package adapter

import (
	"github.com/gcarrenho/component-based/option-2/internal/payments/contracts/user"
	"github.com/gcarrenho/component-based/option-2/internal/users"
)

var _ user.PaymentUserService = (*PaymentAdapter)(nil)

// implementa lo que necesita el consumidor de user en este caso es payment
type PaymentAdapter struct {
	userService *users.UserComponentImpl
}

func NewOrderAdapter(userService *users.UserComponentImpl) *PaymentAdapter {
	return &PaymentAdapter{
		userService: userService,
	}
}

func (a *PaymentAdapter) IsUserActive(userID string) (user.UserDTO, error) {
	/*user := a.userService.FindByID(userID)
	return user.Status == "active", nil*/
	return user.UserDTO{}, nil
}
