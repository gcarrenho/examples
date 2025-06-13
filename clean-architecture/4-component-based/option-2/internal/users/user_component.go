package users

import "github.com/gcarrenho/component-based/option-2/internal/users/domain"

type UserComponent interface {
	CreateUser(userID string) (domain.User, error)
}
