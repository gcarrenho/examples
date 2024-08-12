package order

type orderProducer interface {
	ProduceOrderSyncEvent(order Order) error
	ProduceOrderAsyncEvent(order Order) error
}
