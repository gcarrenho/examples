package users

import "github.com/gcarrenho/component-based/option-2/internal/users/model"

type userRepository interface {
	GetUserByID(userID string) (model.User, error)
}
