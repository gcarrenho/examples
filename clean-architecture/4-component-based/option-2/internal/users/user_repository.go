package users

type repository interface {
	FindByID(userID string, items []string) error
}
