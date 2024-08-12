package user

import (
	"encoding/json"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

var _ userConsumer = (*kafkaUserConsumer)(nil)

type Config struct {
	KafkaServer  string
	KafkaTopic   string
	KafkaGroupId string
}

type kafkaUserConsumer struct {
	kafkaConsumer *kafka.Consumer
	*Config
}

func NewKafkaUserConsumer(config *Config) (userConsumer, error) {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": config.KafkaServer,
		"group.id":          config.KafkaGroupId,
		"auto.offset.reset": "earliest", //  If there are mesasge previous it will consume it
	})
	if err != nil {
		fmt.Println("error ", err)
	}
	return &kafkaUserConsumer{
		kafkaConsumer: consumer,
		Config:        config,
	}, nil
}

func (k *kafkaUserConsumer) ConsumeOrderEvent() (OrderMsg, error) {

	defer k.kafkaConsumer.Close()

	k.kafkaConsumer.SubscribeTopics([]string{k.Config.KafkaTopic}, nil)

	for {
		msg, err := k.kafkaConsumer.ReadMessage(-1)
		if err == nil {
			var order OrderMsg
			err := json.Unmarshal(msg.Value, &order)
			if err != nil {
				fmt.Printf("error decoding message: %v\n", err)
				continue
			}

			//fmt.Printf("Received Order: %+v\n", order)
			return order, nil
		} else {
			fmt.Printf("error: %v\n", err)
		}
	}
}
