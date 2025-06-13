package users

import "github.com/gcarrenho/component-based/option-2/internal/users/model"

type repository interface {
	GetByID(userID string) (model.User, error)
}
