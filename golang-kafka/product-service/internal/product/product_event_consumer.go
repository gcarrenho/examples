package product

var _ ProductComponent = (*productEventConsumer)(nil)

type productEventConsumer struct {
	consumer productConsumer
}

func NewProductEventConsumer(consumer productConsumer) ProductComponent {
	return &productEventConsumer{
		consumer: consumer,
	}
}

func (p productEventConsumer) Consume() (Product, error) {
	orderMsg, err := p.consumer.ConsumeOrderEvent()
	if err != nil {
		return Product{}, err
	}

	return Product{ID: orderMsg.ProductID}, nil
}
