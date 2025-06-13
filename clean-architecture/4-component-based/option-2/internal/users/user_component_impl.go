package users

import "github.com/gcarrenho/component-based/option-2/internal/users/domain"

var _ UserComponent = (*UserComponentImpl)(nil)

type UserComponentImpl struct {
	repo repository
}

func NewService(repo repository) *UserComponentImpl {
	return &UserComponentImpl{repo: repo}
}

func (s *UserComponentImpl) CreateUser(userID string) (domain.User, error) {

	/*ok, err := s.userSvc.IsUserActive(userID)
	  if err != nil || !ok {
	      return ErrUserNotActive
	  }

	  order, err := domain.NewOrder(userID, items)
	  if err != nil {
	      return err
	  }*/

	//return s.repo.Save(order)
	return domain.User{}, nil
}
