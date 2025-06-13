package users

import "github.com/gcarrenho/component-based/option-1/internal/users/model"

type UserComponent interface {
	CreateUser(userID string) (model.User, error)
	FindUserByID(userID string) (model.User, error)
}
