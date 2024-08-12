package product

type productConsumer interface {
	ConsumeOrderEvent() (OrderMsg, error)
}
