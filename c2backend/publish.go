package main

import (
	"log"

	e4 "teserakt/e4common"
)

func (s *C2) sendToClient(id, payload []byte) error {

	topic := e4.TopicForID(id)
	payloadstring := string(payload)
	mqttQoS := byte(2)

	log.Printf("command sent to %s", topic)

	if token := s.mqClient.Publish(topic, mqttQoS, false, payloadstring); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}
