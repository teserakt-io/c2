package main

import (
	"github.com/go-kit/kit/log"

	e4 "teserakt/e4common"
)

func (s *C2) publish(payload []byte, topic string, qos byte) error {

	payloadstring := string(payload)

	logger := log.With(s.logger, "protocol", "mqtt")

	if token := s.mqttClient.Publish(topic, qos, false, payloadstring); token.Wait() && token.Error() != nil {
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
