package main

import (
// mqtt stuff
)

func TopicForId(id []byte) string {

	return idTopicPrefix + string(id)
}

func (s *C2) sendToClient(id, payload []byte) error {

	// concert []byte to string  as string(b)
	topic := TopicForId(id)
	payloadstring := string(payload)

	if token := s.mqClient.Publish(topic, mqttQoS, false, payloadstring); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}
