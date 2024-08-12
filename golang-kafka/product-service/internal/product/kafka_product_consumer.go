package product

import (
	"encoding/json"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

var _ productConsumer = (*kafkaProductConsumer)(nil)

type Config struct {
	KafkaServer  string
	KafkaTopic   string
	KafkaGroupId string
}

type kafkaProductConsumer struct {
	kafkaConsumer *kafka.Consumer
	*Config
}

func NewkafKaProductConsumer(config *Config) (productConsumer, error) {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": config.KafkaServer,
		"group.id":          config.KafkaGroupId, // define the Kafka Group Id into user-service. Because if the consumer group is the same with product service, one of the services wouldn’t receive the message because one partition in the topic can only be listened to by one consumer in one group
		"auto.offset.reset": kafka.OffsetEnd,     // If there are mesasge previous it wont consume it
	})
	if err != nil {
		fmt.Println("error ", err)
	}

	return &kafkaProductConsumer{
		kafkaConsumer: consumer,
		Config:        config,
	}, nil
}

func (k *kafkaProductConsumer) ConsumeOrderEvent() (OrderMsg, error) {
	defer k.kafkaConsumer.Close()

	err := k.kafkaConsumer.SubscribeTopics([]string{k.Config.KafkaTopic}, nil)
	if err != nil {
		fmt.Println("error ", err)

	}
	fmt.Println(k.Config.KafkaTopic)
	fmt.Println(k.Config.KafkaServer)

	for {
		msg, err := k.kafkaConsumer.ReadMessage(-1) // The parameter -1 specifies that the method should block the process and wait for a message indefinitely until a message is received.

		if err == nil {
			var order OrderMsg
			err := json.Unmarshal(msg.Value, &order)
			if err != nil {
				fmt.Printf("error decoding message: %v\n", err)
				continue
			}

			return order, nil
		} else {
			fmt.Printf("error: %v\n", err)
		}
	}
}
