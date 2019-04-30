package services

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/go-kit/kit/log"

	"gitlab.com/teserakt/c2/internal/models"
	"gitlab.com/teserakt/c2/internal/protocols"
	e4 "gitlab.com/teserakt/e4common"
)

// E4 describe the available methods on the E4 service
type E4 interface {
	NewClient(id, key []byte) error
	RemoveClient(id []byte) error
	NewTopicClient(id []byte, topic string) error
	RemoveTopicClient(id []byte, topic string) error
	ResetClient(id []byte) error
	NewTopic(topic string) error
	RemoveTopic(topic string) error
	GetTopicList() ([]string, error)
	SendMessage(topic, msg string) error
	GetAllTopicIds() ([]string, error)
	GetAllClientHexIds() ([]string, error)
	NewClientKey(id []byte) error
	CreateAndProtectForID(cmd e4.Command, topichash, key, id []byte) ([]byte, error)
	CountTopicsForID(id []byte) (int, error)
	CountIDsForTopic(topic string) (int, error)
	GetTopicsForID(id []byte, offset, count int) ([]string, error)
	GetIdsforTopic(topic string, offset, count int) ([]string, error)
}

type e4impl struct {
	db         models.Database
	mqttClient protocols.MQTTClient
	logger     log.Logger
	keyenckey  []byte
}

var _ E4 = &e4impl{}

func (s *e4impl) encryptKey(key []byte) ([]byte, error) {
	protectedkey, err := e4.Encrypt(s.keyenckey, nil, key)
	if err != nil {
		return nil, err
	}

	return protectedkey, nil
}

func (s *e4impl) decryptKey(enckey []byte) ([]byte, error) {
	key, err := e4.Decrypt(s.keyenckey, nil, enckey)
	if err != nil {
		return nil, err
	}

	return key, nil
}

// NewE4 creates a new E4 service
func NewE4(db models.Database, mqttClient protocols.MQTTClient, logger log.Logger, keyenckey []byte) E4 {
	return &e4impl{
		db:         db,
		mqttClient: mqttClient,
		logger:     logger,
		keyenckey:  keyenckey,
	}
}

func (s *e4impl) NewClient(id, key []byte) error {
	logger := log.With(s.logger, "protocol", "e4", "command", "newClient")

	protectedkey, err := s.encryptKey(key)
	if err != nil {
		return err
	}

	if err := s.db.InsertIDKey(id, protectedkey); err != nil {
		logger.Log("msg", "insertIDKey failed", "error", err)
		return err
	}

	logger.Log("msg", "succeeded", "client", e4.PrettyID(id))

	return nil
}

func (s *e4impl) RemoveClient(id []byte) error {
	logger := log.With(s.logger, "protocol", "e4", "command", "removeClient")

	err := s.db.DeleteIDKey(id)
	if err != nil {
		logger.Log("msg", "deleteIDKey failed", "error", err)
		return err
	}
	logger.Log("msg", "succeeded", "client", e4.PrettyID(id))
	return nil
}

func (s *e4impl) NewTopicClient(id []byte, topic string) error {
	logger := log.With(s.logger, "protocol", "e4", "command", "newTopicClient")

	topicKey, err := s.db.GetTopicKey(topic)
	if err != nil {
		logger.Log("msg", "getTopicKey failed", "error", err)
		return err
	}

	key, err := s.decryptKey(topicKey.Key)
	if err != nil {
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

	err = s.db.LinkIDTopic(id, topic)
	if err != nil {
		logger.Log("msg", "Database record of client-topic link failed", err)
		return err
	}

	logger.Log("msg", "succeeded", "client", e4.PrettyID(id), "topic", topic, "topichash", topichash)
	return nil
}

func (s *e4impl) RemoveTopicClient(id []byte, topic string) error {
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
	err = s.db.UnlinkIDTopic(id, topic)
	if err != nil {
		logger.Log("msg", "Cannot remove DB record of client-topic link", err)
		return err
	}

	logger.Log("msg", "succeeded", "topic", topic)

	return nil
}

func (s *e4impl) ResetClient(id []byte) error {
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

func (s *e4impl) NewTopic(topic string) error {
	logger := log.With(s.logger, "protocol", "e4", "command", "newTopic")

	key := e4.RandomKey()

	protectedKey, err := e4.Encrypt(s.keyenckey[:], nil, key)
	if err != nil {
		return err
	}

	if err := s.db.InsertTopicKey(topic, protectedKey); err != nil {
		logger.Log("msg", "insertTopicKey failed", "error", err)
		return err
	}
	logger.Log("msg", "insertTopicKey succeeded", "topic", topic)

	err = s.mqttClient.SubscribeToTopic(topic)
	if err != nil {
		logger.Log("msg", "subscribeToTopic failed", "topic", topic, "error", err)
		return err
	}
	logger.Log("msg", "subscribeToTopic succeeded", "topic", topic)

	return nil
}

func (s *e4impl) RemoveTopic(topic string) error {
	logger := log.With(s.logger, "protocol", "e4", "command", "removeTopic")

	err := s.mqttClient.UnsubscribeFromTopic(topic)
	if err != nil {
		logger.Log("msg", "unsubscribeFromTopic failed", "error", err)
	} else {
		logger.Log("msg", "unsubscribeFromTopic succeeded")
	}

	if err := s.db.DeleteTopicKey(topic); err != nil {
		logger.Log("msg", "deleteTopicKey failed", "error", err)
		return err
	}
	logger.Log("msg", "succeeded", "topic", topic)

	return nil
}

func (s *e4impl) GetTopicList() ([]string, error) {
	topicKeys, err := s.db.GetAllTopics()
	if err != nil {
		return nil, err
	}

	var topics []string
	for _, topicKey := range topicKeys {
		topics = append(topics, topicKey.Topic)
	}

	return topics, nil
}

func (s *e4impl) SendMessage(topic, msg string) error {
	logger := log.With(s.logger, "protocol", "e4", "command", "sendMessage")

	topickey, err := s.db.GetTopicKey(topic)
	if err != nil {
		logger.Log("msg", "getTopicKey failed", "error", err)
		return err
	}

	payload, err := e4.Protect([]byte(msg), topickey.Key)
	if err != nil {
		logger.Log("msg", "Protect failed", "error", err)
		return err
	}
	err = s.mqttClient.Publish(payload, topic, byte(0))
	if err != nil {
		logger.Log("msg", "publish failed", "error", err)
		return err
	}

	logger.Log("msg", "succeeded", "topic", topic)
	return nil
}

func (s *e4impl) NewClientKey(id []byte) error {
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

	protectedkey, err := e4.Encrypt(s.keyenckey[:], nil, key)
	if err != nil {
		return err
	}

	err = s.db.InsertIDKey(id, protectedkey)
	if err != nil {
		logger.Log("msg", "insertIDKey failed", "error", err)
		return err
	}
	logger.Log("msg", "succeeded", "id", e4.PrettyID(id))

	return nil
}

// CreateAndProtectForID creates a protected command for a given ID.
func (s *e4impl) CreateAndProtectForID(cmd e4.Command, topichash, key, id []byte) ([]byte, error) {

	command, err := createCommand(cmd, topichash, key)
	if err != nil {
		return nil, err
	}

	// get key of the given id
	idkey, err := s.db.GetIDKey(id)
	if err != nil {
		return nil, err
	}

	clearkey, err := e4.Decrypt(s.keyenckey, nil, idkey.Key)
	if err != nil {
		return nil, err
	}

	// protect
	payload, err := e4.Protect(command, clearkey)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func (s *e4impl) GetAllTopicIds() ([]string, error) {
	topicKeys, err := s.db.GetAllTopics()
	if err != nil {
		return nil, err
	}

	var topics []string
	for _, topickey := range topicKeys {
		topics = append(topics, topickey.Topic)
	}

	return topics, nil
}

func (s *e4impl) GetAllClientHexIds() ([]string, error) {
	idkeys, err := s.db.GetAllIDKeys()
	if err != nil {
		return nil, err
	}

	var hexids []string
	for _, idkey := range idkeys {
		hexids = append(hexids, hex.EncodeToString(idkey.E4ID[0:]))
	}

	return hexids, nil
}

func (s *e4impl) CountTopicsForID(id []byte) (int, error) {
	return s.db.CountTopicsForID(id)
}

func (s *e4impl) GetTopicsForID(id []byte, offset, count int) ([]string, error) {
	topicKeys, err := s.db.GetTopicsForID(id, offset, count)
	if err != nil {
		return nil, err
	}

	var topics []string
	for _, topicKey := range topicKeys {
		topics = append(topics, topicKey.Topic)
	}

	return topics, nil
}

func (s *e4impl) CountIDsForTopic(topic string) (int, error) {
	return s.db.CountIDsForTopic(topic)
}

func (s *e4impl) GetIdsforTopic(topic string, offset, count int) ([]string, error) {
	idKeys, err := s.db.GetIdsforTopic(topic, offset, count)
	if err != nil {
		return nil, err
	}

	var hexids []string
	for _, idkey := range idKeys {
		hexids = append(hexids, hex.EncodeToString(idkey.E4ID))
	}

	return hexids, nil
}

func createCommand(cmd e4.Command, topichash, key []byte) ([]byte, error) {
	switch cmd {

	case e4.RemoveTopic:
		if err := e4.IsValidTopicHash(topichash); err != nil {
			return nil, fmt.Errorf("invalid topic hash for RemoveTopic: %s", err)
		}
		if key != nil {
			return nil, errors.New("unexpected key for RemoveTopic")
		}
		return append([]byte{cmd.ToByte()}, topichash...), nil

	case e4.ResetTopics:
		if topichash != nil || key != nil {
			return nil, errors.New("unexpected argument for ResetTopics")
		}
		return []byte{cmd.ToByte()}, nil

	case e4.SetIDKey:
		if err := e4.IsValidKey(key); err != nil {
			return nil, fmt.Errorf("invalid key for SetIdKey: %s", err)
		}
		if topichash != nil {
			return nil, errors.New("unexpected topichash for SetIdKey")
		}
		return append([]byte{cmd.ToByte()}, key...), nil

	case e4.SetTopicKey:
		if err := e4.IsValidKey(key); err != nil {
			return nil, fmt.Errorf("invalid key for SetTopicKey: %s", err)
		}
		if err := e4.IsValidTopicHash(topichash); err != nil {
			return nil, fmt.Errorf("invalid topic hash for SetTopicKey: %s", err)
		}
		return append(append([]byte{cmd.ToByte()}, key...), topichash...), nil
	}

	return nil, errors.New("invalid command")
}

func (s *e4impl) sendCommandToClient(id, payload []byte) error {
	topic := e4.TopicForID(id)
	qos := byte(2)

	return s.mqttClient.Publish(payload, topic, qos)
}
