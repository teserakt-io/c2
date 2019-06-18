package protocols

//go:generate mockgen -destination=mqtt_mocks.go -package protocols -self_package gitlab.com/teserakt/c2/internal/protocols gitlab.com/teserakt/c2/internal/protocols MQTTClient,MQTTMessage,MQTTToken

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
	"unicode/utf8"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-kit/kit/log"
	"go.opencensus.io/trace"

	"gitlab.com/teserakt/c2/internal/analytics"
	"gitlab.com/teserakt/c2/internal/config"
)

var (
	// ErrMQTTTimeout is returned when the response from the mqtt broker timeout
	ErrMQTTTimeout = errors.New("mqtt timed out")
)

// List of MQTT availabe QoS
var (
	QoSAtMostOnce  = byte(0)
	QosAtLeastOnce = byte(1)
	QoSExactlyOnce = byte(2)
)

// MQTTClient defines a minimal mqtt.Client needed to support E4 protocol
type MQTTClient interface {
	mqtt.Client
}

// MQTTMessage wrap around a mqtt.Message
type MQTTMessage interface {
	mqtt.Message
}

// MQTTToken wrap around a mqtt.Token
type MQTTToken interface {
	mqtt.Token
}

type mqttPubSubClient struct {
	mqtt              MQTTClient
	config            config.MQTTCfg
	logger            log.Logger
	monitor           analytics.MessageMonitor
	waitTimeout       time.Duration
	disconnectTimeout uint // idk why they used uint here instead of a time.Duration. They do convert internally tho.
}

var _ PubSubClient = &mqttPubSubClient{}

// NewMQTTPubSubClient creates and connect a new PubSubClient over MQTT
func NewMQTTPubSubClient(
	cfg config.MQTTCfg,
	logger log.Logger,
	monitor analytics.MessageMonitor,
) PubSubClient {
	// TODO: secure connection to broker
	mqOpts := mqtt.NewClientOptions()
	mqOpts.AddBroker(cfg.Broker)
	mqOpts.SetClientID(cfg.ID)
	mqOpts.SetPassword(cfg.Password)
	mqOpts.SetUsername(cfg.Username)

	mqtt := mqtt.NewClient(mqOpts)

	return &mqttPubSubClient{
		mqtt:              mqtt,
		config:            cfg,
		logger:            logger,
		monitor:           monitor,
		waitTimeout:       1 * time.Second,
		disconnectTimeout: 1000,
	}
}

func (c *mqttPubSubClient) Connect() error {
	c.logger.Log("msg", "mqtt parameters", "broker", c.config.Broker, "id", c.config.ID, "username", c.config.Username)
	token := c.mqtt.Connect()
	// WaitTimeout instead of Wait or this will block indefinitively the execution if the server is down
	if !token.WaitTimeout(c.waitTimeout) {
		c.logger.Log("msg", "connection failed", "error", ErrMQTTTimeout)
		return ErrMQTTTimeout
	}

	if token.Error() != nil {
		c.logger.Log("msg", "connection failed", "error", token.Error())
		return token.Error()
	}

	c.logger.Log("msg", "connected to broker")

	return nil
}

func (c *mqttPubSubClient) Disconnect() error {
	c.mqtt.Disconnect(c.disconnectTimeout)

	return nil
}

func (c *mqttPubSubClient) SubscribeToTopics(ctx context.Context, topics []string) error {
	ctx, span := trace.StartSpan(ctx, "mqtt.SubscribeToTopics")
	defer span.End()

	if !c.monitor.Enabled() {
		c.logger.Log("msg", "monitoring is not enabled, skipping topics subscription")
		return nil
	}

	if len(topics) == 0 {
		c.logger.Log("msg", "no topic provided, no subscribe request sent")
		return nil
	}

	// create map string->qos as needed by SubscribeMultiple
	filters := make(map[string]byte, len(topics))
	for _, topic := range topics {
		filters[topic] = byte(c.config.QoSSub)
	}

	token := c.mqtt.SubscribeMultiple(filters, func(mqttClient mqtt.Client, m mqtt.Message) {
		c.logMessage(ctx, m)
	})
	if !token.WaitTimeout(c.waitTimeout) {
		c.logger.Log("msg", "subscribe-multiple failed", "topics", len(topics), "error", ErrMQTTTimeout)

		return ErrMQTTTimeout
	}
	if token.Error() != nil {
		c.logger.Log("msg", "subscribe-multiple failed", "topics", len(topics), "error", token.Error())
		return token.Error()
	}
	c.logger.Log("msg", "subscribe-multiple succeeded", "topics", len(topics))

	return nil
}

func (c *mqttPubSubClient) SubscribeToTopic(ctx context.Context, topic string) error {
	ctx, span := trace.StartSpan(ctx, "mqtt.SubscribeToTopic")
	defer span.End()

	// Only index message if monitoring enabled, i.e. if esClient is defined
	if !c.monitor.Enabled() {
		c.logger.Log("msg", "monitoring is not enabled, skipping topic subscription")
		return nil
	}

	logger := log.With(c.logger, "protocol", "mqtt")

	token := c.mqtt.Subscribe(topic, byte(c.config.QoSSub), func(mqttClient mqtt.Client, message mqtt.Message) {
		c.logMessage(ctx, message)
	})
	if !token.WaitTimeout(c.waitTimeout) {
		logger.Log("msg", "subscribe failed", "topic", topic, "error", ErrMQTTTimeout)

		return ErrMQTTTimeout
	}
	if token.Error() != nil {
		logger.Log("msg", "subscribe failed", "topic", topic, "error", token.Error())

		return token.Error()
	}
	logger.Log("msg", "subscribe succeeded", "topic", topic)

	return nil
}

func (c *mqttPubSubClient) UnsubscribeFromTopic(ctx context.Context, topic string) error {
	ctx, span := trace.StartSpan(ctx, "mqtt.UnsubscribeFromTopic")
	defer span.End()

	// Only index message if monitoring enabled, i.e. if esClient is defined
	if !c.monitor.Enabled() {
		return nil
	}

	logger := log.With(c.logger, "protocol", "mqtt")

	token := c.mqtt.Unsubscribe(topic)
	if !token.WaitTimeout(c.waitTimeout) {
		logger.Log("msg", "unsubscribe failed", "topic", topic, "error", ErrMQTTTimeout)

		return ErrMQTTTimeout
	}
	if token.Error() != nil {
		logger.Log("msg", "unsubscribe failed", "topic", topic, "error", token.Error())
		return token.Error()
	}
	logger.Log("msg", "unsubscribe succeeded", "topic", topic)

	return nil
}

func (c *mqttPubSubClient) Publish(ctx context.Context, payload []byte, topic string, qos byte) error {
	ctx, span := trace.StartSpan(ctx, "mqtt.Publish")
	defer span.End()

	logger := log.With(c.logger, "protocol", "mqtt")

	payloadStr := string(payload)

	token := c.mqtt.Publish(topic, qos, true, payloadStr)
	if !token.WaitTimeout(c.waitTimeout) {
		logger.Log("msg", "publish failed", "topic", topic, "error", ErrMQTTTimeout)

		return ErrMQTTTimeout
	}
	if token.Error() != nil {
		logger.Log("msg", "publish failed", "topic", topic, "error", token.Error())
		return token.Error()
	}
	logger.Log("msg", "publish succeeded", "topic", topic)

	return nil
}

func (c *mqttPubSubClient) logMessage(ctx context.Context, m MQTTMessage) {
	ctx, span := trace.StartSpan(ctx, "mqtt.onMessage")
	defer span.End()

	msg := analytics.LoggedMessage{
		Timestamp:       time.Now(),
		Duplicate:       m.Duplicate(),
		Qos:             m.Qos(),
		Retained:        m.Retained(),
		Topic:           m.Topic(),
		MessageID:       m.MessageID(),
		Payload:         m.Payload(),
		IsUTF8:          utf8.Valid(m.Payload()),
		IsJSON:          false,
		IsBase64:        false,
		LooksCompressed: false,
		LooksEncrypted:  false,
	}

	// try to determine type
	if !msg.IsUTF8 {
		if analytics.LooksCompressed(m.Payload()) {
			msg.LooksCompressed = true
		} else {
			msg.LooksEncrypted = analytics.LooksEncrypted(m.Payload())
		}
	} else {
		var js map[string]interface{}
		if json.Unmarshal(m.Payload(), &js) == nil {
			msg.IsJSON = true
		} else {
			if _, err := base64.StdEncoding.DecodeString(string(m.Payload())); err == nil {
				msg.IsBase64 = true
			}
		}
	}

	c.monitor.OnMessage(ctx, msg)
}
