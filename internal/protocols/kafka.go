package protocols

import (
	"strings"

	"github.com/Shopify/sarama"
	"github.com/go-kit/kit/log"

	"gitlab.com/teserakt/c2/internal/analytics"
	"gitlab.com/teserakt/c2/internal/config"
)

type kafkaPubSubClient struct {
	logger  log.Logger
	cfg     config.KafkaCfg
	monitor analytics.MessageMonitor

	consumer sarama.Consumer
	producer sarama.SyncProducer

	connected        bool
	stopChan         chan bool
	subscribedTopics map[string]chan bool
}

var _ PubSubClient = &kafkaPubSubClient{}

// NewKafkaPubSubClient creates a new PubSubClient backed by Kafka
func NewKafkaPubSubClient(cfg config.KafkaCfg, logger log.Logger, monitor analytics.MessageMonitor) PubSubClient {
	return &kafkaPubSubClient{
		logger:  logger,
		cfg:     cfg,
		monitor: monitor,

		subscribedTopics: make(map[string]chan bool),
	}
}

func (c *kafkaPubSubClient) Connect() error {
	if c.connected {
		return ErrAlreadyConnected
	}

	kafkaCfg := sarama.NewConfig()
	kafkaCfg.Producer.Return.Successes = true // Enable SyncProducer

	kafkaClient, err := sarama.NewClient(c.cfg.Brokers, kafkaCfg)
	if err != nil {
		c.logger.Log("msg", "kafka client failed to connect to broker(s)", "error", err)
		return err
	}

	consumer, err := sarama.NewConsumerFromClient(kafkaClient)
	if err != nil {
		c.logger.Log("msg", "failed to initialise kafka consumer", "error", err)
		return err
	}
	c.consumer = consumer

	producer, err := sarama.NewSyncProducerFromClient(kafkaClient)
	if err != nil {
		c.logger.Log("msg", "failed to initialise kafka producer", "error", err)
		return err
	}
	c.producer = producer

	c.stopChan = make(chan bool)
	c.connected = true

	return nil
}

func (c *kafkaPubSubClient) Disconnect() error {
	if !c.connected {
		return ErrNotConnected
	}

	if err := c.consumer.Close(); err != nil {
		c.logger.Log("msg", "failed to close kafka consumer", "error", err)
		return err
	}

	if err := c.producer.Close(); err != nil {
		c.logger.Log("msg", "failed to close kafka producer", "error", err)
		return err
	}

	for _, stopChan := range c.subscribedTopics {
		close(stopChan)
	}

	c.connected = false
	c.subscribedTopics = make(map[string]chan bool)

	return nil
}

func (c *kafkaPubSubClient) SubscribeToTopics(topics []string) error {
	for _, topic := range topics {
		if err := c.SubscribeToTopic(topic); err != nil {
			return err
		}
	}

	return nil
}

func (c *kafkaPubSubClient) SubscribeToTopic(rawTopic string) error {
	topic := filterTopicName(rawTopic)

	partitionConsumer, err := c.consumer.ConsumePartition(topic, 0, sarama.OffsetNewest)
	if err != nil {
		c.logger.Log("msg", "failed to subscribe to topic", "topic", topic, "error", err)
		return err
	}

	stopChan := make(chan bool)
	c.subscribedTopics[topic] = stopChan

	go c.onMessage(partitionConsumer, stopChan)

	c.logger.Log("msg", "successfully subscribed to topic", "topic", topic)

	return nil
}

func (c *kafkaPubSubClient) UnsubscribeFromTopic(rawTopic string) error {
	topic := filterTopicName(rawTopic)

	stopChan, exists := c.subscribedTopics[topic]
	if !exists {
		c.logger.Log("msg", "cannot unsubscribe to a non subscribed topic", "topic", topic)

		return nil
	}

	close(stopChan)
	c.logger.Log("msg", "successfully unsubscribed from topic", "topic", topic)

	return nil
}

func (c *kafkaPubSubClient) Publish(payload []byte, rawTopic string, qos byte) error {
	topic := filterTopicName(rawTopic)

	partition, offset, err := c.producer.SendMessage(&sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(payload),
	})
	if err != nil {
		c.logger.Log("msg", "failed to published message", "topic", topic, "partition", partition, "offset", offset)

		return err
	}

	c.logger.Log("msg", "successfully published message", "topic", topic, "partition", partition, "offset", offset)

	return nil
}

func (c *kafkaPubSubClient) onMessage(partitionConsumer sarama.PartitionConsumer, stopChan <-chan bool) {
	select {
	case err := <-partitionConsumer.Errors():
		c.logger.Log("msg", "partitionConsumer error", "error", err)
	case msg := <-partitionConsumer.Messages():
		c.logger.Log("msg", "received kafka message", "data", msg)
	case <-stopChan:
		if err := partitionConsumer.Close(); err != nil {
			c.logger.Log("msg", "failed to stop partition consumer", "error", err)
			return
		}

		return
	}
}

func filterTopicName(topic string) string {
	// Kafka have restricted charlist for topic names,
	// see https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/common/internals/Topic.java#L29
	return strings.Replace(topic, "/", "-", -1)
}
