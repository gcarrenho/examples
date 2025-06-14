package adapter

import (
	"github.com/gcarrenho/component-based/option-2/internal/payments/contracts/user"
	"github.com/gcarrenho/component-based/option-2/internal/users"
)

var _ user.PaymentUserService = (*PaymentAdapter)(nil)

// Implement that is needed by consumero of user in this case payment.
type PaymentAdapter struct {
	userService *users.UserComponentImpl
}

func NewOrderAdapter(userService *users.UserComponentImpl) *PaymentAdapter {
	return &PaymentAdapter{
		userService: userService,
	}
}

func (p *PaymentAdapter) IsUserActive(userID string) (bool, error) {
	/*user := a.userService.FindByID(userID)
	return user.Status == "active", nil*/
	return true, nil
}

func (a *PaymentAdapter) GetBillingInfo(userID string) (user.BillingInfo, error) {
	return user.BillingInfo{
		Name: "example",
	}, nil
}
