// Copyright 2020 Teserakt AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package protocols

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"regexp"
	"unicode/utf8"

	"github.com/Shopify/sarama"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"

	"github.com/teserakt-io/c2/internal/analytics"
	"github.com/teserakt-io/c2/internal/config"
	"github.com/teserakt-io/c2/internal/models"
	e4 "github.com/teserakt-io/e4go"
)

type kafkaPubSubClient struct {
	logger  log.FieldLogger
	cfg     config.KafkaCfg
	monitor analytics.MessageMonitor

	consumer sarama.Consumer
	producer sarama.SyncProducer

	connected        bool
	subscribedTopics map[string]chan bool
}

var _ PubSubClient = (*kafkaPubSubClient)(nil)

// NewKafkaPubSubClient creates a new PubSubClient backed by Kafka
func NewKafkaPubSubClient(cfg config.KafkaCfg, logger log.FieldLogger, monitor analytics.MessageMonitor) PubSubClient {
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
		c.logger.WithError(err).Error("kafka client failed to connect to broker(s)")
		return err
	}

	consumer, err := sarama.NewConsumerFromClient(kafkaClient)
	if err != nil {
		c.logger.WithError(err).Error("failed to initialize kafka consumer")
		return err
	}
	c.consumer = consumer

	producer, err := sarama.NewSyncProducerFromClient(kafkaClient)
	if err != nil {
		c.logger.WithError(err).Error("failed to initialize kafka producer")
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
		c.logger.WithError(err).Error("failed to close kafka consumer")
		return err
	}

	if err := c.producer.Close(); err != nil {
		c.logger.WithError(err).Error("failed to close kafka producer")
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
		if err := c.ValidateTopic(topic); err != nil {
			return err
		}

		if err := c.SubscribeToTopic(ctx, topic); err != nil {
			return err
		}
	}

	return nil
}

func (c *kafkaPubSubClient) SubscribeToTopic(ctx context.Context, topic string) error {
	ctx, span := trace.StartSpan(ctx, "kafka.SubscribeToTopic")
	defer span.End()

	logger := c.logger.WithField("topic", topic)

	if err := c.ValidateTopic(topic); err != nil {
		return err
	}

	partitionConsumer, err := c.consumer.ConsumePartition(topic, 0, sarama.OffsetNewest)
	if err != nil {
		logger.WithError(err).Error("failed to subscribe to topic")
		return err
	}

	stopChan := make(chan bool)
	c.subscribedTopics[topic] = stopChan

	go c.watchForMessages(ctx, partitionConsumer, stopChan)

	logger.Info("successfully subscribed to topic")

	return nil
}

func (c *kafkaPubSubClient) UnsubscribeFromTopic(ctx context.Context, topic string) error {
	_, span := trace.StartSpan(ctx, "kafka.UnsubscribeFromTopic")
	defer span.End()

	logger := c.logger.WithField("topic", topic)

	if err := c.ValidateTopic(topic); err != nil {
		return err
	}

	stopChan, exists := c.subscribedTopics[topic]
	if !exists {
		logger.Warn("cannot unsubscribe to a non subscribed topic")

		return nil
	}

	delete(c.subscribedTopics, topic)

	close(stopChan)
	logger.Info("successfully unsubscribed from topic")

	return nil
}

func (c *kafkaPubSubClient) Publish(ctx context.Context, payload []byte, client models.Client, qos byte) error {
	_, span := trace.StartSpan(ctx, "kafka.Publish")
	defer span.End()

	topic := e4.TopicForID(client.E4ID)

	logger := c.logger.WithField("topic", topic)

	if err := c.ValidateTopic(topic); err != nil {
		return err
	}

	partition, offset, err := c.producer.SendMessage(&sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(payload),
	})
	if err != nil {
		logger.WithFields(log.Fields{"partition": partition, "offset": offset}).WithError(err).Error("failed to publish message")
		return err
	}

	logger.WithFields(log.Fields{"partition": partition, "offset": offset}).Info("successfully published message")

	return nil
}

func (c *kafkaPubSubClient) ValidateTopic(topic string) error {
	// Kafka have restricted charlist for topic names,
	// see https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/common/internals/Topic.java#L29
	matched, err := regexp.MatchString(`^[a-zA-Z0-9\._-]+$`, topic)
	if err != nil {
		return err
	}

	if !matched {
		return ErrInvalidTopic
	}

	return nil
}

func (c *kafkaPubSubClient) watchForMessages(ctx context.Context, partitionConsumer sarama.PartitionConsumer, stopChan <-chan bool) {
	for {
		select {
		case err := <-partitionConsumer.Errors():
			c.logger.WithError(err).Error("partitionConsumer error")
		case msg := <-partitionConsumer.Messages():
			ctx, span := trace.StartSpan(ctx, "kafka.onMessage")

			c.logger.WithField("data", msg).Debug("received kafka message")
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
					if _, err := base64.StdEncoding.DecodeString(string(loggedMsg.Payload)); err == nil {
						loggedMsg.IsBase64 = true
					}
				}
			}

			c.monitor.OnMessage(ctx, loggedMsg)
			span.End()

		case <-stopChan:
			c.logger.Info("stopping watching for messages by stop channel")
			if err := partitionConsumer.Close(); err != nil {
				c.logger.WithError(err).Error("failed to stop partition consumer")
				return
			}

			return
		case <-ctx.Done():
			c.logger.WithError(ctx.Err()).Warn("stopping watching for messages by context")
			return
		}
	}
}
