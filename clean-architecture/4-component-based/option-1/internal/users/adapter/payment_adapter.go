package adapter

import (
	user "github.com/gcarrenho/component-based/option-1/internal/contracts/user"
	"github.com/gcarrenho/component-based/option-1/internal/users"
)

var _ user.PaymentUserService = (*PaymentAdapter)(nil)

// implementa lo que necesita el consumidor de user en este caso es payment
type PaymentAdapter struct {
	userService *users.UserComponentImpl
}

func NewOrderAdapter(userService *users.UserComponentImpl) user.PaymentUserService {
	return &PaymentAdapter{
		userService: userService,
	}
}

func (p *PaymentAdapter) IsUserActive(userID string) (user.UserDTO, error) {
	userFinding, err := p.userService.FindUserByID(userID)
	if err != nil {
		return user.UserDTO{}, err
	}

	return user.UserDTO{
		ID:   userFinding.ID,
		Name: userFinding.Name,
	}, nil
}
