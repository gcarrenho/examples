// implementa repository
package users

import (
	"database/sql"

	"github.com/gcarrenho/component-based/option-1/internal/users/model"
)

type userRepositoryImpl struct {
	DB *sql.DB
}

func newUserRepositoryImpl(db *sql.DB) *userRepositoryImpl {
	return &userRepositoryImpl{
		DB: db,
	}
}

func (r *userRepositoryImpl) GetUserByID(userID string) (model.User, error) {
	return model.User{}, nil // This is a placeholder implementation.
}
