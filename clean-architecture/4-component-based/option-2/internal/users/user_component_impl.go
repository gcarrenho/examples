package users

import domain "github.com/gcarrenho/component-based/option-2/internal/users/model"

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

func (s *UserComponentImpl) FindUserByID(userID string) (domain.User, error) {
	// Simulate fetching user from repository
	user, err := s.repo.GetByID(userID)
	if err != nil {
		return domain.User{}, err
	}

	return user, nil
}
