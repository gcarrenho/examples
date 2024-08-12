package order

var _ OrderComponent = (*OrderEventProducer)(nil)

type OrderEventProducer struct {
	orderProducer orderProducer
}

func NewOrderEventProducer(producer orderProducer) OrderComponent {
	return &OrderEventProducer{
		orderProducer: producer,
	}
}

func (o OrderEventProducer) Produce(order Order) error {
	return o.orderProducer.ProduceOrderAsyncEvent(order)
}
