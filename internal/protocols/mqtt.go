package protocols

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-kit/kit/log"
	"github.com/olivere/elastic"

	"gitlab.com/teserakt/c2/internal/config"
)

// MQTTClient ...
type MQTTClient interface {
	SubscribeToTopics(topics []string) error
	SubscribeToTopic(topic string) error
	UnsubscribeFromTopic(topic string) error
	Publish(payload []byte, topic string, qos byte) error
}

type mqttClient struct {
	mqtt     mqtt.Client
	config   config.MQTTCfg
	logger   log.Logger
	esClient *elastic.Client
}

type loggedMessage struct {
	Duplicate       bool   `json:"duplicate"`
	Qos             byte   `json:"qos"`
	Retained        bool   `json:"retained"`
	Topic           string `json:"topic"`
	MessageID       uint16 `json:"messageid"`
	Payload         []byte `json:"payload"`
	LooksEncrypted  bool   `json:"looksencrypted"`
	LooksCompressed bool   `json:"lookscompressed"`
	IsBase64        bool   `json:"isbase64"`
	IsUTF8          bool   `json:"isutf8"`
	IsJSON          bool   `json:"isjson"`
}

// NewMQTTClient creates and connect a new MQTT client
func NewMQTTClient(scfg config.MQTTCfg, logger log.Logger, esClient *elastic.Client) (MQTTClient, error) {
	// TODO: secure connection to broker
	logger.Log("addr", scfg.Broker)

	mqOpts := mqtt.NewClientOptions()
	mqOpts.AddBroker(scfg.Broker)
	mqOpts.SetClientID(scfg.ID)
	mqOpts.SetPassword(scfg.Password)
	mqOpts.SetUsername(scfg.Username)

	mqtt := mqtt.NewClient(mqOpts)

	logger.Log("msg", "mqtt parameters", "broker", scfg.Broker, "id", scfg.ID, "username", scfg.Username)
	if token := mqtt.Connect(); token.Wait() && token.Error() != nil {
		logger.Log("msg", "connection failed", "error", token.Error())
		return nil, token.Error()
	}

	logger.Log("msg", "connected to broker")

	return &mqttClient{
		mqtt:     mqtt,
		config:   scfg,
		logger:   logger,
		esClient: esClient,
	}, nil
}

func (c *mqttClient) SubscribeToTopics(topics []string) error {
	if len(topics) == 0 {
		c.logger.Log("msg", "no topic found in the db, no subscribe request sent")
		return nil
	}

	// create map string->qos as needed by SubscribeMultiple
	filters := make(map[string]byte, len(topics))
	for _, topic := range topics {
		filters[topic] = byte(c.config.QoSSub)
	}

	fmt.Println(filters)

	if token := c.mqtt.SubscribeMultiple(filters, func(mqttClient mqtt.Client, m mqtt.Message) {
		c.onMessage(m)
	}); token.Wait() && token.Error() != nil {
		c.logger.Log("msg", "subscribe-multiple failed", "topics", len(topics), "error", token.Error())
		return token.Error()
	}
	c.logger.Log("msg", "subscribe-multiple succeeded", "topics", len(topics))

	return nil
}

func (c *mqttClient) SubscribeToTopic(topic string) error {
	// Only index message if monitoring enabled, i.e. if esClient is defined
	if c.esClient == nil {
		return nil
	}

	logger := log.With(c.logger, "protocol", "mqtt")

	qos := byte(c.config.QoSSub)

	if token := c.mqtt.Subscribe(topic, qos, func(mqttClient mqtt.Client, message mqtt.Message) {
		c.onMessage(message)
	}); token.Wait() && token.Error() != nil {
		logger.Log("msg", "subscribe failed", "topic", topic, "error", token.Error())

		return token.Error()
	}
	logger.Log("msg", "subscribe succeeded", "topic", topic)

	return nil
}

func (c *mqttClient) UnsubscribeFromTopic(topic string) error {
	// Only index message if monitoring enabled, i.e. if esClient is defined
	if c.esClient == nil {
		return nil
	}

	logger := log.With(c.logger, "protocol", "mqtt")

	if token := c.mqtt.Unsubscribe(topic); token.Wait() && token.Error() != nil {
		logger.Log("msg", "unsubscribe failed", "topic", topic, "error", token.Error())
		return token.Error()
	}
	logger.Log("msg", "unsubscribe succeeded", "topic", topic)

	return nil
}

func (c *mqttClient) Publish(payload []byte, topic string, qos byte) error {
	logger := log.With(c.logger, "protocol", "mqtt")

	payloadStr := string(payload)

	if token := c.mqtt.Publish(topic, qos, true, payloadStr); token.Wait() && token.Error() != nil {
		logger.Log("msg", "publish failed", "topic", topic, "error", token.Error())
		return token.Error()
	}
	logger.Log("msg", "publish succeeded", "topic", topic)

	return nil
}

func (c *mqttClient) onMessage(m mqtt.Message) {
	msg := &loggedMessage{
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
		if looksCompressed(m.Payload()) {
			msg.LooksCompressed = true
		} else {
			msg.LooksEncrypted = looksEncrypted(m.Payload())
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

	b, _ := json.Marshal(msg)
	ctx := context.Background()

	c.esClient.Index().Index("messages").Type("message").BodyString(string(b)).Do(ctx)
}

func looksEncrypted(data []byte) bool {
	// efficient, lazy heuristic, FN/FP-prone
	// will fail if e.g. ciphertext is prepended with non-random nonce
	if len(data) < 16 {
		// make the assumption that <16-byte data won't be encrypted
		return false
	}
	counter := make(map[int]int)
	for i := range data[:16] {
		counter[int(data[i])]++
	}
	if len(counter) < 10 {
		return false
	}
	// if encrypted, fails with low prob

	return true
}

func looksCompressed(data []byte) bool {
	// application/zip
	if bytes.Equal(data[:4], []byte("\x50\x4b\x03\x04")) {
		return true
	}

	// application/x-gzip
	if bytes.Equal(data[:3], []byte("\x1F\x8B\x08")) {
		return true
	}

	// application/x-rar-compressed
	if bytes.Equal(data[:7], []byte("\x52\x61\x72\x20\x1A\x07\x00")) {
		return true
	}

	// zlib no/low compression
	if bytes.Equal(data[:2], []byte("\x78\x01")) {
		return true
	}

	// zlib default compression
	if bytes.Equal(data[:2], []byte("\x78\x9c")) {
		return true
	}

	// zlib best compression
	if bytes.Equal(data[:2], []byte("\x78\xda")) {
		return true
	}

	return false
}
