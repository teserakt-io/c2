package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-kit/kit/log"
	e4 "gitlab.com/teserakt/e4common"
)

// MQTTContext ...
type MQTTContext struct {
	client mqtt.Client
	qosSub int
	qosPub int
}

type startMQTTClientConfig struct {
	addr     string
	id       string
	password string
	username string
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

func (s *C2) createMQTTClient(scfg *startMQTTClientConfig) error {

	// TODO: secure connection to broker
	logger := log.With(s.logger, "protocol", "mqtt")
	logger.Log("addr", scfg.addr)
	mqOpts := mqtt.NewClientOptions()
	mqOpts.AddBroker(scfg.addr)
	mqOpts.SetClientID(scfg.id)
	mqOpts.SetPassword(scfg.password)
	mqOpts.SetUsername(scfg.username)
	mqttClient := mqtt.NewClient(mqOpts)
	logger.Log("msg", "mqtt parameters", "broker", scfg.addr, "id", scfg.id, "username", scfg.username)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		logger.Log("msg", "connection failed", "error", token.Error())
		return token.Error()
	}
	logger.Log("msg", "connected to broker")
	// instantiate C2
	s.mqttContext.client = mqttClient
	return nil
}

func (s *C2) subscribeToDBTopics() error {

	topics, err := s.dbGetTopicsList()

	if err != nil {
		s.logger.Log("msg", "failed to get topic list from db", "error", err)
		return err
	}

	logger := log.With(s.logger, "protocol", "mqtt")

	if len(topics) == 0 {
		logger.Log("msg", "no topic found in the db, no subscribe request sent")
		return nil
	}

	// create map string->qos as needed by SubscribeMultiple
	filters := make(map[string]byte)
	for i := 0; i < len(topics); i += 1 {
		filters[topics[i]] = byte(s.mqttContext.qosSub)
	}

	fmt.Println(filters)

	if token := s.mqttContext.client.SubscribeMultiple(filters, callbackSub); token.Wait() && token.Error() != nil {
		logger.Log("msg", "subscribe-multiple failed", "topics", len(topics), "error", token.Error())
		return token.Error()
	}
	logger.Log("msg", "subscribe-multiple succeeded", "topics", len(topics))

	return nil
}

func (s *C2) subscribeToTopic(topic string) error {

	logger := log.With(s.logger, "protocol", "mqtt")

	qos := byte(s.mqttContext.qosSub)

	if token := s.mqttContext.client.Subscribe(topic, qos, callbackSub); token.Wait() && token.Error() != nil {
		logger.Log("msg", "subscribe failed", "topic", topic, "error", token.Error())
		return token.Error()
	}
	logger.Log("msg", "subscribe succeeded", "topic", topic)

	return nil
}

func (s *C2) unsubscribeFromTopic(topic string) error {

	logger := log.With(s.logger, "protocol", "mqtt")

	if token := s.mqttContext.client.Unsubscribe(topic); token.Wait() && token.Error() != nil {
		logger.Log("msg", "unsubscribe failed", "topic", topic, "error", token.Error())
		return token.Error()
	}
	logger.Log("msg", "unsubscribe succeeded", "topic", topic)

	return nil
}

func callbackSub(c mqtt.Client, m mqtt.Message) {

	// Only index message if monitoring enabled, i.e. if esClient is defined
	if esClient == nil {
		return
	}

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

	esClient.Index().Index("messages").Type("message").BodyString(string(b)).Do(ctx)
}

func (s *C2) publish(payload []byte, topic string, qos byte) error {

	logger := log.With(s.logger, "protocol", "mqtt")

	payloadstring := string(payload)

	if token := s.mqttContext.client.Publish(topic, qos, true, payloadstring); token.Wait() && token.Error() != nil {
		logger.Log("msg", "publish failed", "topic", topic, "error", token.Error())
		return token.Error()
	}
	logger.Log("msg", "publish succeeded", "topic", topic)

	return nil
}

func (s *C2) sendCommandToClient(id, payload []byte) error {

	topic := e4.TopicForID(id)
	qos := byte(2)

	return s.publish(payload, topic, qos)
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
