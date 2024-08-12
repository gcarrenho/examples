package main

import (
	"fmt"

	"github.com/gcarrenho/golang-kafka/internal/product"
)

const (
	KafkaServer  = "localhost:9092"
	KafkaTopic   = "orders-v1-topic"
	KafkaGroupId = "product-service"
)

func main() {
	productConsumer, err := product.NewkafKaProductConsumer(&product.Config{
		KafkaServer:  KafkaServer,
		KafkaTopic:   KafkaTopic,
		KafkaGroupId: KafkaGroupId,
	})
	if err != nil {
		fmt.Println("error NewKAfkaProductConsumer ", err)
	}

	productSvc := product.NewProductEventConsumer(productConsumer)

	product, err := productSvc.Consume()
	if err != nil {
		fmt.Println("error ", err)
	}

	fmt.Printf("Received Product: %+v\n", product)
}
