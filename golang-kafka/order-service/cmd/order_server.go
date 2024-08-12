package main

import (
	"fmt"

	"github.com/gcarrenho/golang-kafka/internal/order"
	"github.com/google/uuid"
)

const (
	KafkaServer = "localhost:9092"
	KafkaTopic  = "orders-v1-topic"
)

func main() {
	orderProducer, err := order.NewSaramaKafkaOrderProducer(&order.Config{
		KafkaServer: KafkaServer,
		KafkaTopic:  KafkaTopic,
		BrokerList:  []string{KafkaServer},
	})
	if err != nil {
		fmt.Println("error ", err)
	}

	orderSvc := order.NewOrderEventProducer(orderProducer)

	order := order.Order{
		ID:        uuid.New().String(),
		ProductID: uuid.New().String(),
		UserID:    uuid.New().String(),
		Amount:    456000,
	}

	err = orderSvc.Produce(order)
	if err != nil {
		fmt.Println("error ", err)

	}

	fmt.Printf("Message sent %+v\n", order)
}
