package user

type userConsumer interface {
	ConsumeOrderEvent() (OrderMsg, error)
}
