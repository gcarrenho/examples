package order

import (
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
)

var _ orderProducer = (*saramaKafkaOrderProducer)(nil)

type saramaKafkaOrderProducer struct {
	producerAsync sarama.AsyncProducer
	producerSync  sarama.SyncProducer
	*Config
}

func newProducerAsync(brokerList []string) sarama.AsyncProducer {
	// For the access log, we are looking for AP semantics, with high throughput.
	// By creating batches of compressed messages, we reduce network I/O at a cost of more latency.
	config := sarama.NewConfig()
	/*tlsConfig := createTlsConfiguration()
	if tlsConfig != nil {
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = tlsConfig
	}*/
	config.Producer.RequiredAcks = sarama.WaitForLocal       // Only wait for the leader to ack
	config.Producer.Compression = sarama.CompressionSnappy   // Compress messages
	config.Producer.Flush.Frequency = 500 * time.Millisecond // Flush batches every 500ms
	config.Producer.Return.Successes = true

	producer, err := sarama.NewAsyncProducer(brokerList, config)
	if err != nil {
		log.Fatalln("Failed to start Sarama producer:", err)
	}

	// We will just log to STDOUT if we're not able to produce messages.
	// Note: messages will only be returned here after all retry attempts are exhausted.
	go func() {
		for err := range producer.Errors() {
			log.Println("Failed to write access log entry:", err)
		}
	}()

	return producer
}

func newProducerSync(brokerList []string) sarama.SyncProducer {
	// For the data collector, we are looking for strong consistency semantics.
	// Because we don't change the flush settings, sarama will try to produce messages
	// as fast as possible to keep latency low.
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll // Wait for all in-sync replicas to ack the message
	config.Producer.Retry.Max = 10                   // Retry up to 10 times to produce the message
	config.Producer.Return.Successes = true
	/*tlsConfig := createTlsConfiguration()
	if tlsConfig != nil {
		config.Net.TLS.Config = tlsConfig
		config.Net.TLS.Enable = true
	}*/

	// On the broker side, you may want to change the following settings to get
	// stronger consistency guarantees:
	// - For your broker, set `unclean.leader.election.enable` to false
	// - For the topic, you could increase `min.insync.replicas`.

	producer, err := sarama.NewSyncProducer(brokerList, config)
	if err != nil {
		log.Fatalln("Failed to start Sarama producer:", err)
	}

	return producer
}

func NewSaramaKafkaOrderProducer(config *Config) (*saramaKafkaOrderProducer, error) {

	return &saramaKafkaOrderProducer{
		producerAsync: newProducerAsync(config.BrokerList),
		producerSync:  newProducerSync(config.BrokerList),
		Config:        config,
	}, nil
}

func (s *saramaKafkaOrderProducer) ProduceOrderAsyncEvent(order Order) error {
	s.producerAsync.Input() <- &sarama.ProducerMessage{
		Topic: s.Config.KafkaTopic,
		Key:   sarama.StringEncoder(order.ID),
		Value: &order,
	}

	select {
	case success := <-s.producerAsync.Successes():
		fmt.Println("Message produced:", success.Offset, s.Config.KafkaTopic)
	case err := <-s.producerAsync.Errors():
		fmt.Println("Failed to produce message:", err)
	}

	return nil

}

func (s *saramaKafkaOrderProducer) ProduceOrderSyncEvent(order Order) error {
	msg := &sarama.ProducerMessage{
		Topic: s.Config.KafkaTopic,
		Value: &order,
	}

	partition, offset, err := s.producerSync.SendMessage(msg)
	if err != nil {
		log.Printf("Failed to send message: %v", err)
		return err
	}

	fmt.Printf("Message sent successfully to partition %d, offset %d\n", partition, offset)
	return nil
}
