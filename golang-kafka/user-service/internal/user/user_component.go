package user

type UserComponent interface {
	Consume() (User, error)
}
