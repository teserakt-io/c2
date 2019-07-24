package protocols

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/Shopify/sarama"
	"github.com/go-kit/kit/log"
	"go.opencensus.io/trace"

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
	subscribedTopics map[string]chan bool
}

var _ PubSubClient = (*kafkaPubSubClient)(nil)

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

	c.consumer = nil
	c.producer = nil
	c.connected = false
	c.subscribedTopics = make(map[string]chan bool)

	return nil
}

func (c *kafkaPubSubClient) SubscribeToTopics(ctx context.Context, topics []string) error {
	ctx, span := trace.StartSpan(ctx, "kafka.SubscribeToTopics")
	defer span.End()

	for _, topic := range topics {
		if err := c.SubscribeToTopic(ctx, topic); err != nil {
			return err
		}
	}

	return nil
}

func (c *kafkaPubSubClient) SubscribeToTopic(ctx context.Context, rawTopic string) error {
	ctx, span := trace.StartSpan(ctx, "kafka.SubscribeToTopic")
	defer span.End()

	topic := filterTopicName(rawTopic)

	partitionConsumer, err := c.consumer.ConsumePartition(topic, 0, sarama.OffsetNewest)
	if err != nil {
		c.logger.Log("msg", "failed to subscribe to topic", "topic", rawTopic, "error", err)
		return err
	}

	stopChan := make(chan bool)
	c.subscribedTopics[rawTopic] = stopChan

	go c.watchForMessages(ctx, partitionConsumer, stopChan)

	c.logger.Log("msg", "successfully subscribed to topic", "topic", rawTopic)

	return nil
}

func (c *kafkaPubSubClient) UnsubscribeFromTopic(ctx context.Context, rawTopic string) error {
	ctx, span := trace.StartSpan(ctx, "kafka.UnsubscribeFromTopic")
	defer span.End()

	stopChan, exists := c.subscribedTopics[rawTopic]
	if !exists {
		c.logger.Log("msg", "cannot unsubscribe to a non subscribed topic", "topic", rawTopic)

		return nil
	}

	delete(c.subscribedTopics, rawTopic)

	close(stopChan)
	c.logger.Log("msg", "successfully unsubscribed from topic", "topic", rawTopic)

	return nil
}

func (c *kafkaPubSubClient) Publish(ctx context.Context, payload []byte, rawTopic string, qos byte) error {
	ctx, span := trace.StartSpan(ctx, "kafka.Publish")
	defer span.End()

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

func (c *kafkaPubSubClient) watchForMessages(ctx context.Context, partitionConsumer sarama.PartitionConsumer, stopChan <-chan bool) {
	for {
		select {
		case err := <-partitionConsumer.Errors():
			c.logger.Log("msg", "partitionConsumer error", "error", err)
		case msg := <-partitionConsumer.Messages():
			ctx, span := trace.StartSpan(ctx, "kafka.onMessage")

			c.logger.Log("msg", "received kafka message", "data", msg)
			loggedMsg := analytics.LoggedMessage{
				Duplicate:       false,
				Qos:             byte(0),
				Retained:        false,
				Topic:           msg.Topic,
				MessageID:       0,
				Payload:         msg.Value,
				IsUTF8:          utf8.Valid(msg.Value),
				IsJSON:          false,
				IsBase64:        false,
				LooksCompressed: false,
				LooksEncrypted:  false,
			}

			// try to determine type
			if !loggedMsg.IsUTF8 {
				if analytics.LooksCompressed(loggedMsg.Payload) {
					loggedMsg.LooksCompressed = true
				} else {
					loggedMsg.LooksEncrypted = analytics.LooksEncrypted(loggedMsg.Payload)
				}
			} else {
				var js map[string]interface{}
				if json.Unmarshal(loggedMsg.Payload, &js) == nil {
					loggedMsg.IsJSON = true
				} else {
					c.logger.Log(string(loggedMsg.Payload))
					if _, err := base64.StdEncoding.DecodeString(string(loggedMsg.Payload)); err == nil {
						loggedMsg.IsBase64 = true
					}
				}
			}

			c.monitor.OnMessage(ctx, loggedMsg)
			span.End()

		case <-stopChan:
			c.logger.Log("msg", "stopping watching for messages by stop channel")
			if err := partitionConsumer.Close(); err != nil {
				c.logger.Log("msg", "failed to stop partition consumer", "error", err)
				return
			}

			return
		case <-ctx.Done():
			c.logger.Log("msg", "stopping watching for messages by context", "error", ctx.Err())
			return
		}
	}
}

func filterTopicName(topic string) string {
	// Kafka have restricted charlist for topic names,
	// see https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/common/internals/Topic.java#L29
	return strings.Replace(topic, "/", "-", -1)
}
