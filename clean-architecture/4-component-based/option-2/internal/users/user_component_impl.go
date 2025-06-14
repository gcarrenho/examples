package users

import (
	"database/sql"

	"github.com/gcarrenho/component-based/option-2/internal/users/model"
)

var _ UserComponent = (*UserComponentImpl)(nil)

type Deps struct {
	DB *sql.DB
}

type UserComponentImpl struct {
	repo userRepository
}

func NewUserComponentImpl(deps Deps) UserComponent {
	repo := newUserRepositoryImpl(deps.DB)
	return &UserComponentImpl{repo: repo}
}

func (s *UserComponentImpl) CreateUser(userID string) (model.User, error) {

	/*ok, err := s.userSvc.IsUserActive(userID)
	  if err != nil || !ok {
	      return ErrUserNotActive
	  }

	  order, err := domain.NewOrder(userID, items)
	  if err != nil {
	      return err
	  }*/

	//return s.repo.Save(order)
	return model.User{}, nil
}

func (s *UserComponentImpl) FindUserByID(userID string) (model.User, error) {
	return s.repo.GetUserByID(userID)
}
