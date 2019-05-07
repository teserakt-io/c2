package services

import (
	"encoding/hex"
	"fmt"

	"github.com/go-kit/kit/log"

	"gitlab.com/teserakt/c2/internal/commands"
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
	SendMessage(topic, msg string) error
	GetAllTopicIds() ([]string, error)
	GetAllClientHexIds() ([]string, error)
	NewClientKey(id []byte) error
	CountTopicsForID(id []byte) (int, error)
	CountIDsForTopic(topic string) (int, error)
	GetTopicsForID(id []byte, offset, count int) ([]string, error)
	GetIdsforTopic(topic string, offset, count int) ([]string, error)
}

type e4impl struct {
	db             models.Database
	mqttClient     protocols.PubSubClient
	commandFactory commands.Factory
	logger         log.Logger
	keyenckey      []byte
}

var _ E4 = &e4impl{}

// NewE4 creates a new E4 service
func NewE4(
	db models.Database,
	mqttClient protocols.PubSubClient,
	commandFactory commands.Factory,
	logger log.Logger,
	keyenckey []byte,
) E4 {
	return &e4impl{
		db:             db,
		mqttClient:     mqttClient,
		commandFactory: commandFactory,
		logger:         logger,
		keyenckey:      keyenckey,
	}
}

func (s *e4impl) NewClient(id, key []byte) error {
	logger := log.With(s.logger, "protocol", "e4", "command", "newClient")

	protectedkey, err := e4.Encrypt(s.keyenckey, nil, key)
	if err != nil {
		logger.Log("msg", "failed to encrypt key", "error", err)
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

	idKey, err := s.db.GetIDKey(id)
	if err != nil {
		logger.Log("msg", "failed to retrieve idKey", "error", err)
		return err
	}

	topicKey, err := s.db.GetTopicKey(topic)
	if err != nil {
		logger.Log("msg", "failed to retrieve topicKey", "error", err)
		return err
	}

	clearTopicKey, err := topicKey.DecryptKey(s.keyenckey)
	if err != nil {
		logger.Log("msg", "failed to decrypt topicKey", "error", err)
		return err
	}

	command, err := s.commandFactory.CreateSetTopicKeyCommand(topicKey.Hash(), clearTopicKey)
	if err != nil {
		logger.Log("msg", "failed to create setTopicKey command", "error", err)
		return err
	}

	err = s.sendCommandToClient(command, idKey)
	if err != nil {
		logger.Log("msg", "sendCommandToClient failed", "error", err)
		return err
	}

	err = s.db.LinkIDTopic(idKey, topicKey)
	if err != nil {
		logger.Log("msg", "Database record of client-topic link failed", err)
		return err
	}

	logger.Log("msg", "succeeded", "client", e4.PrettyID(id), "topic", topic, "topichash", topicKey.Hash())
	return nil
}

func (s *e4impl) RemoveTopicClient(id []byte, topic string) error {
	logger := log.With(s.logger, "protocol", "e4", "command", "removeTopicClient")

	idKey, err := s.db.GetIDKey(id)
	if err != nil {
		logger.Log("msg", "failed to retrieve idKey", "error", err)
		return err
	}

	topicKey, err := s.db.GetTopicKey(topic)
	if err != nil {
		logger.Log("msg", "failed to retrieve topicKey", "error", err)
		return err
	}

	command, err := s.commandFactory.CreateRemoveTopicCommand(topicKey.Hash())
	if err != nil {
		logger.Log("msg", "failed to create removeTopic command", "error", err)
		return err
	}

	err = s.sendCommandToClient(command, idKey)
	if err != nil {
		logger.Log("msg", "sendCommandToClient failed", "error", err)
		return err
	}

	err = s.db.UnlinkIDTopic(idKey, topicKey)
	if err != nil {
		logger.Log("msg", "Cannot remove DB record of client-topic link", err)
		return err
	}

	logger.Log("msg", "succeeded", "topic", topic)

	return nil
}

func (s *e4impl) ResetClient(id []byte) error {
	logger := log.With(s.logger, "protocol", "e4", "command", "resetClient")

	idKey, err := s.db.GetIDKey(id)
	if err != nil {
		logger.Log("msg", "failed to retrieve idKey", "error", err)
		return err
	}

	command, err := s.commandFactory.CreateResetTopicsCommand()
	if err != nil {
		logger.Log("msg", "failed to create resetTopics command", "error", err)
		return err
	}

	err = s.sendCommandToClient(command, idKey)
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

	err = s.mqttClient.SubscribeToTopic(topic) // Monitoring
	if err != nil {
		logger.Log("msg", "subscribeToTopic failed", "topic", topic, "error", err)
		return err
	}
	logger.Log("msg", "subscribeToTopic succeeded", "topic", topic)

	return nil
}

func (s *e4impl) RemoveTopic(topic string) error {
	logger := log.With(s.logger, "protocol", "e4", "command", "removeTopic")

	err := s.mqttClient.UnsubscribeFromTopic(topic) // Monitoring
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

func (s *e4impl) SendMessage(topic, msg string) error {
	logger := log.With(s.logger, "protocol", "e4", "command", "sendMessage")

	topicKey, err := s.db.GetTopicKey(topic)
	if err != nil {
		logger.Log("msg", "failed to retrieve topicKey", "error", err)
		return err
	}

	clearTopicKey, err := topicKey.DecryptKey(s.keyenckey)
	if err != nil {
		logger.Log("msg", "failed to decrypt topicKey", "error", err)
		return err
	}

	payload, err := e4.Protect([]byte(msg), clearTopicKey)
	if err != nil {
		logger.Log("msg", "Protect failed", "error", err)
		return err
	}
	err = s.mqttClient.Publish(payload, topic, protocols.QoSAtMostOnce)
	if err != nil {
		logger.Log("msg", "publish failed", "error", err)
		return err
	}

	logger.Log("msg", "succeeded", "topic", topic)
	return nil
}

func (s *e4impl) NewClientKey(id []byte) error {
	logger := log.With(s.logger, "protocol", "e4", "command", "newClientKey")

	idKey, err := s.db.GetIDKey(id)
	if err != nil {
		logger.Log("msg", "failed to retrieve idKey", "error", err)
		return err
	}

	newKey := e4.RandomKey()
	command, err := s.commandFactory.CreateSetIDKeyCommand(newKey)
	if err != nil {
		logger.Log("msg", "failed to create SetIDKey command", "error", err)
		return err
	}

	err = s.sendCommandToClient(command, idKey)
	if err != nil {
		logger.Log("msg", "sendCommandToClient failed", "error", err)
		return err
	}

	protectedkey, err := e4.Encrypt(s.keyenckey, nil, newKey)
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
		hexids = append(hexids, hex.EncodeToString(idkey.E4ID))
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

func (s *e4impl) sendCommandToClient(command commands.Command, idKey models.IDKey) error {
	clearIDKey, err := idKey.DecryptKey(s.keyenckey)
	if err != nil {
		return fmt.Errorf("failed to decrypt idKey: %v", err)
	}

	payload, err := command.Protect(clearIDKey)
	if err != nil {
		return fmt.Errorf("failed to protected command: %v", err)
	}

	return s.mqttClient.Publish(payload, idKey.Topic(), protocols.QoSExactlyOnce)
}
