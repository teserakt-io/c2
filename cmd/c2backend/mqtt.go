package main

import (
	"encoding/json"
	"fmt"

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

	// create map string->qos as needed by SubscribeMultiple
	filters := make(map[string]byte)
	for i := 0; i < len(topics); i += 1 {
		filters[topics[i]] = byte(s.mqttContext.qosSub)
	}

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

	type Message struct {
		Duplicate bool
		Qos       byte
		Retained  bool
		Topic     string
		MessageID uint16
		Payload   []byte
	}

	msg := &Message{
		Duplicate: m.Duplicate(),
		Qos:       m.Qos(),
		Retained:  m.Retained(),
		Topic:     m.Topic(),
		MessageID: m.MessageID(),
		Payload:   m.Payload(),
	}

	b, err := json.Marshal(msg)

	if err == nil {
		fmt.Println(string(b))
	}
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
