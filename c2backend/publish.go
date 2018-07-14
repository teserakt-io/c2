package main

import (
	"log"

	e4 "teserakt/e4common"
)

func (s *C2) publish(payload []byte, topic string, qos byte) error {

	payloadstring := string(payload)

	log.Printf("published to topic %s", topic)

	if token := s.mqClient.Publish(topic, qos, false, payloadstring); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}

func (s *C2) sendCommandToClient(id, payload []byte) error {

	topic := e4.TopicForID(id)
	qos := byte(2)

	return s.publish(payload, topic, qos)
}
