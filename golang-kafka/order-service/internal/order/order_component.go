package order

type OrderComponent interface {
	Produce(order Order) error
}
