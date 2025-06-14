package users

import "github.com/gcarrenho/component-based/option-1/internal/users/model"

type userRepository interface {
	GetUserByID(userID string) (model.User, error)
}
