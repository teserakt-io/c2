package protocols

//go:generate mockgen -destination=mqtt_mocks.go -package protocols -self_package github.com/teserakt-io/c2/internal/protocols github.com/teserakt-io/c2/internal/protocols MQTTClient,MQTTMessage,MQTTToken

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"regexp"
	"time"
	"unicode/utf8"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"

	"github.com/teserakt-io/c2/internal/analytics"
	"github.com/teserakt-io/c2/internal/config"
)

var (
	// ErrMQTTTimeout is returned when the response from the mqtt broker timeout
	ErrMQTTTimeout = errors.New("mqtt timed out")
)

// List of MQTT availabe QoS
const (
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
	logger            log.FieldLogger
	monitor           analytics.MessageMonitor
	waitTimeout       time.Duration
	disconnectTimeout uint // idk why they used uint here instead of a time.Duration. They do convert internally tho.
}

var _ PubSubClient = (*mqttPubSubClient)(nil)

// NewMQTTPubSubClient creates and connect a new PubSubClient over MQTT
func NewMQTTPubSubClient(
	cfg config.MQTTCfg,
	logger log.FieldLogger,
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
	c.logger.WithFields(log.Fields{
		"broker":   c.config.Broker,
		"id":       c.config.ID,
		"username": c.config.Username,
	}).Debug("mqtt parameters")

	token := c.mqtt.Connect()
	// WaitTimeout instead of Wait or this will block indefinitely the execution if the server is down
	if !token.WaitTimeout(c.waitTimeout) {
		c.logger.WithError(ErrMQTTTimeout).Error("connection timeout")
		return ErrMQTTTimeout
	}

	if token.Error() != nil {
		c.logger.WithError(token.Error()).Error("connection failed")
		return token.Error()
	}

	c.logger.Info("connected to broker")

	return nil
}

func (c *mqttPubSubClient) Disconnect() error {
	c.mqtt.Disconnect(c.disconnectTimeout)

	return nil
}

func (c *mqttPubSubClient) SubscribeToTopics(ctx context.Context, topics []string) error {
	_, span := trace.StartSpan(ctx, "mqtt.SubscribeToTopics")
	defer span.End()

	logger := c.logger.WithField("topicCount", len(topics))

	if !c.monitor.Enabled() {
		logger.Warn("monitoring is not enabled, skipping topics subscription")
		return nil
	}

	if len(topics) == 0 {
		logger.Warn("no topic provided, no subscribe request sent")
		return nil
	}

	for _, topic := range topics {
		if err := c.ValidateTopic(topic); err != nil {
			return err
		}
	}

	// create map string->qos as needed by SubscribeMultiple
	filters := make(map[string]byte, len(topics))
	for _, topic := range topics {
		filters[topic] = byte(c.config.QoSSub)
	}

	token := c.mqtt.SubscribeMultiple(filters, func(mqttClient mqtt.Client, m mqtt.Message) {
		// Can't reuse global context, as it get canceled before request is sent
		c.logMessage(context.Background(), m)
	})
	if !token.WaitTimeout(c.waitTimeout) {
		logger.WithError(ErrMQTTTimeout).Error("subscribe-multiple timeout")
		return ErrMQTTTimeout
	}
	if token.Error() != nil {
		logger.WithError(token.Error()).Error("subscribe-multiple failed")
		return token.Error()
	}
	logger.Info("subscribe-multiple succeeded")

	return nil
}

func (c *mqttPubSubClient) SubscribeToTopic(ctx context.Context, topic string) error {
	_, span := trace.StartSpan(ctx, "mqtt.SubscribeToTopic")
	defer span.End()

	logger := c.logger.WithField("topic", topic)

	if err := c.ValidateTopic(topic); err != nil {
		return err
	}

	// Only index message if monitoring enabled, i.e. if esClient is defined
	if !c.monitor.Enabled() {
		logger.Warn("monitoring is not enabled, skipping topic subscription")
		return nil
	}

	token := c.mqtt.Subscribe(topic, byte(c.config.QoSSub), func(mqttClient mqtt.Client, message mqtt.Message) {
		// Can't reuse global context, as it get canceled before request is sent
		c.logMessage(context.Background(), message)
	})
	if !token.WaitTimeout(c.waitTimeout) {
		logger.WithError(ErrMQTTTimeout).Error("subscribe timeout")
		return ErrMQTTTimeout
	}
	if token.Error() != nil {
		logger.WithError(token.Error()).Error("subscribe failed")
		return token.Error()
	}
	logger.Info("subscribe succeeded")

	return nil
}

func (c *mqttPubSubClient) UnsubscribeFromTopic(ctx context.Context, topic string) error {
	_, span := trace.StartSpan(ctx, "mqtt.UnsubscribeFromTopic")
	defer span.End()

	logger := c.logger.WithField("topic", topic)

	if err := c.ValidateTopic(topic); err != nil {
		return err
	}

	// Only index message if monitoring enabled, i.e. if esClient is defined
	if !c.monitor.Enabled() {
		logger.Warn("monitoring is not enabled, skipping topic unsubscription")
		return nil
	}

	token := c.mqtt.Unsubscribe(topic)
	if !token.WaitTimeout(c.waitTimeout) {
		logger.WithError(ErrMQTTTimeout).Error("unsubscribe timeout")

		return ErrMQTTTimeout
	}
	if token.Error() != nil {
		logger.WithError(token.Error()).Error("unsubscribe failed")
		return token.Error()
	}
	logger.Info("unsubscribe succeeded")

	return nil
}

func (c *mqttPubSubClient) Publish(ctx context.Context, payload []byte, topic string, qos byte) error {
	_, span := trace.StartSpan(ctx, "mqtt.Publish")
	defer span.End()

	logger := c.logger.WithFields(log.Fields{
		"topic": topic,
		"qos":   qos,
	})

	if err := c.ValidateTopic(topic); err != nil {
		return err
	}

	payloadStr := string(payload)

	token := c.mqtt.Publish(topic, qos, true, payloadStr)
	if !token.WaitTimeout(c.waitTimeout) {
		logger.WithError(ErrMQTTTimeout).Error("publish timeout")
		return ErrMQTTTimeout
	}
	if token.Error() != nil {
		logger.WithError(token.Error()).Error("publish failed")
		return token.Error()
	}
	logger.Info("publish succeeded")

	return nil
}

func (c *mqttPubSubClient) ValidateTopic(topic string) error {
	matched, err := regexp.MatchString(`^\$SYS|[+#]`, topic)
	if err != nil {
		return err
	}

	if matched {
		return ErrInvalidTopic
	}

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
