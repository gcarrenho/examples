package user

var _ UserComponent = (*userEventConsumer)(nil)

type userEventConsumer struct {
	userConsumer userConsumer
}

func NewUserEventConsumer(userConsumer userConsumer) UserComponent {
	return &userEventConsumer{
		userConsumer: userConsumer,
	}
}

func (p userEventConsumer) Consume() (User, error) {
	orderMsg, err := p.userConsumer.ConsumeOrderEvent()
	if err != nil {
		return User{}, err
	}

	return User{ID: orderMsg.UserID}, nil
}
