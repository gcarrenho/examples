package order

import (
	"encoding/json"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

var _ orderProducer = (*kafkaOrderProducer)(nil)

type Config struct {
	KafkaServer string
	KafkaTopic  string
	BrokerList  []string
}

type kafkaOrderProducer struct {
	kafkaConsumer *kafka.Producer
	*Config
}

func NewKafkaOrderProducer(config *Config) (orderProducer, error) {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": config.KafkaServer,
	})
	if err != nil {
		fmt.Println("error ", err)
	}
	return &kafkaOrderProducer{
		kafkaConsumer: producer,
		Config:        config,
	}, nil
}

func (k *kafkaOrderProducer) ProduceOrderSyncEvent(order Order) error {

	defer k.kafkaConsumer.Close()
	value, err := json.Marshal(order)
	if err != nil {
		fmt.Println("error ", err)
	}

	err = k.kafkaConsumer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &k.Config.KafkaTopic, Partition: kafka.PartitionAny},
		Value:          value,
	}, nil)

	if err != nil {
		fmt.Println("error ", err)
	}

	return nil
}

func (k *kafkaOrderProducer) ProduceOrderAsyncEvent(order Order) error {

	return nil
}
