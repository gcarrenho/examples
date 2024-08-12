package main

import (
	"fmt"

	"github.com/gcarrenho/golang-kafka/internal/user"
)

const (
	KafkaServer  = "localhost:9092"
	KafkaTopic   = "orders-v1-topic"
	KafkaGroupId = "user-service"
)

func main() {
	userRepository, err := user.NewKafkaUserConsumer(&user.Config{
		KafkaServer:  KafkaServer,
		KafkaGroupId: KafkaGroupId,
		KafkaTopic:   KafkaTopic,
	})
	if err != nil {
		fmt.Println("error ", err)

	}

	userSvc := user.NewUserEventConsumer(userRepository)

	user, err := userSvc.Consume()
	if err != nil {
		fmt.Println("error ", err)

	}

	fmt.Printf("Received User: %+v\n", user)
}
