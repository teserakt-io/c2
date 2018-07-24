package main

import (
	"github.com/go-kit/kit/log"

	e4 "teserakt/e4go/pkg/e4common"
)

func (s *C2) newClient(id, key []byte) error {

	logger := log.With(s.logger, "protocol", "e4", "command", "newClient")

	err := s.insertIDKey(id, key)
	if err != nil {
		logger.Log("msg", "insertIDKey failed", "error", err)
		return err
	}
	logger.Log("msg", "succeeded", "client", e4.PrettyID(id))
	return nil
}

func (s *C2) removeClient(id []byte) error {

	logger := log.With(s.logger, "protocol", "e4", "command", "removeClient")

	err := s.deleteIDKey(id)
	if err != nil {
		logger.Log("msg", "deleteIDKey failed", "error", err)
		return err
	}
	logger.Log("msg", "succeeded", "client", e4.PrettyID(id))
	return nil
}

func (s *C2) newTopicClient(id []byte, topic string) error {

	logger := log.With(s.logger, "protocol", "e4", "command", "newTopicClient")

	key, err := s.getTopicKey(topic)
	if err != nil {
		logger.Log("msg", "getTopicKey failed", "error", err)
		return err
	}

	topichash := e4.HashTopic(topic)

	payload, err := s.CreateAndProtectForID(e4.SetTopicKey, topichash, key, id)
	if err != nil {
		logger.Log("msg", "CreateAndProtectForID failed", "error", err)
		return err
	}
	err = s.sendCommandToClient(id, payload)
	if err != nil {
		logger.Log("msg", "sendCommandToClient failed", "error", err)
		return err
	}

	logger.Log("msg", "succeeded", "client", e4.PrettyID(id), "topic", topic, "topichash", topichash)
	return nil
}

func (s *C2) removeTopicClient(id []byte, topic string) error {

	logger := log.With(s.logger, "protocol", "e4", "command", "removeTopicClient")

	topichash := e4.HashTopic(topic)

	payload, err := s.CreateAndProtectForID(e4.RemoveTopic, topichash, nil, id)
	if err != nil {
		logger.Log("msg", "CreateAndProtectForID failed", "error", err)
		return err
	}
	err = s.sendCommandToClient(id, payload)
	if err != nil {
		logger.Log("msg", "sendCommandToClient failed", "error", err)
		return err
	}
	logger.Log("msg", "succeeded", "topic", topic)

	return nil
}

func (s *C2) resetClient(id []byte) error {

	logger := log.With(s.logger, "protocol", "e4", "command", "resetClient")

	payload, err := s.CreateAndProtectForID(e4.ResetTopics, nil, nil, id)
	if err != nil {
		logger.Log("msg", "CreateAndProtectForID failed", "error", err)
		return err
	}
	err = s.sendCommandToClient(id, payload)
	if err != nil {
		logger.Log("msg", "sendCommandToClient failed", "error", err)
		return err
	}
	logger.Log("msg", "succeeded", "client", e4.PrettyID(id))

	return nil
}

func (s *C2) newTopic(topic string) error {

	logger := log.With(s.logger, "protocol", "e4", "command", "newTopic")

	key := e4.RandomKey()

	err := s.insertTopicKey(topic, key)
	if err != nil {
		logger.Log("msg", "insertTopicKey failed", "error", err)
		return err
	}
	logger.Log("msg", "succeeded", "topic", topic)

	return nil
}

func (s *C2) removeTopic(topic string) error {

	logger := log.With(s.logger, "protocol", "e4", "command", "removeTopic")

	err := s.deleteTopicKey(topic)
	if err != nil {
		logger.Log("msg", "deleteTopicKey failed", "error", err)
		return err
	}
	logger.Log("msg", "succeeded", "topic", topic)

	return nil
}

func (s *C2) sendMessage(topic, msg string) error {

	logger := log.With(s.logger, "protocol", "e4", "command", "sendMessage")

	topickey, err := s.getTopicKey(topic)
	if err != nil {
		logger.Log("msg", "getTopicKey failed", "error", err)
		return err
	}
	payload, err := e4.Protect([]byte(msg), topickey)
	if err != nil {
		logger.Log("msg", "Protect failed", "error", err)
		return err
	}
	err = s.publish(payload, topic, byte(0))
	if err != nil {
		logger.Log("msg", "publish failed", "error", err)
		return err
	}

	logger.Log("msg", "succeeded", "topic", topic)
	return nil
}

func (s *C2) newClientKey(id []byte) error {

	logger := log.With(s.logger, "protocol", "e4", "command", "newClientKey")

	key := e4.RandomKey()

	// first send to the client, and only update locally afterwards
	payload, err := s.CreateAndProtectForID(e4.SetIDKey, nil, key, id)
	if err != nil {
		logger.Log("msg", "CreateAndProtectForID failed", "error", err)
		return err
	}
	err = s.sendCommandToClient(id, payload)
	if err != nil {
		logger.Log("msg", "sendCommandToClient failed", "error", err)
		return err
	}

	err = s.insertIDKey(id, key)
	if err != nil {
		logger.Log("msg", "insertIDKey failed", "error", err)
		return err
	}
	logger.Log("msg", "succeeded", "id", e4.PrettyID(id))

	return nil
}
