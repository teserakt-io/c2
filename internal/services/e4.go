package services

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/go-kit/kit/log"

	"gitlab.com/teserakt/c2/internal/commands"
	"gitlab.com/teserakt/c2/internal/models"
	"gitlab.com/teserakt/c2/internal/protocols"
	e4 "gitlab.com/teserakt/e4common"
)

/* E4 API Naming

   The APIs are duplicated at several levels within the C2 for a good reason,
   so that we can separate out the various levels of logic. Naming however
   needs to be consistent, or as consistent as possible.

   E4 interface items should match the verbs in the specification as closely
   as possible. However, go functions cannot have overloaded type signatures.
   As such, we use the following syntax for all APIs:

   VERB (ALL) Thing SUFFIX where:

   SUFFIX takes these possible options and appears strictly in this order:

	- For = Specifies that we are retrieving something for some specific value,
	        GetTopicsForClient = Get Topics given CLIENT=client.
    - As = Specifies how we retrieve it
	- By  = If the thing we are retrieving has multiple query specifiers,
	        By specifies how.
	- Range = if the item in question has offset/count limits, this specifier
	          is included.
*/

// E4 describe the available methods on the E4 service
type E4 interface {

	// Client Only Manipulation
	NewClient(name string, id, key []byte) error
	NewClientKey(name string, id []byte) error
	RemoveClientByID(id []byte) error
	RemoveClientByName(name string) error
	ResetClientByID(id []byte) error
	ResetClientByName(name string) error
	GetAllClientsAsHexIDs() ([]string, error)
	GetAllClientsAsNames() ([]string, error)
	GetClientsAsHexIDsRange(offset, count int) ([]string, error)
	GetClientsAsNamesRange(offset, count int) ([]string, error)
	CountClients() (int, error)

	// Individual Topic Manipulaton
	NewTopic(topic string) error
	RemoveTopic(topic string) error
	GetTopicsRange(offset, count int) ([]string, error)
	GetAllTopics() ([]string, error)
	GetAllTopicsUnsafe() ([]string, error)
	CountTopics() (int, error)

	// Linking, removing topic-client mappings:
	NewTopicClient(name string, id []byte, topic string) error
	RemoveTopicClientByID(id []byte, topic string) error
	RemoveTopicClientByName(name string, topic string) error

	// > Counting topics per client, or clients per topic.
	CountTopicsForClientByID(id []byte) (int, error)
	CountTopicsForClientByName(name string) (int, error)
	CountClientsForTopic(topic string) (int, error)

	// > Retrieving clients per topic or topics per client
	GetTopicsForClientByID(id []byte, offset, count int) ([]string, error)
	GetTopicsForClientByName(name string, offset, count int) ([]string, error)
	GetClientsByNameForTopic(topic string, offset, count int) ([]string, error)
	GetClientsByIDForTopic(topic string, offset, count int) ([]string, error)

	// Communications
	SendMessage(topic, msg string) error
}

type e4impl struct {
	db             models.Database
	pubSubClient   protocols.PubSubClient
	commandFactory commands.Factory
	logger         log.Logger
	keyenckey      []byte
}

var _ E4 = &e4impl{}

// NewE4 creates a new E4 service
func NewE4(
	db models.Database,
	pubSubClient protocols.PubSubClient,
	commandFactory commands.Factory,
	logger log.Logger,
	keyenckey []byte,
) E4 {
	return &e4impl{
		db:             db,
		pubSubClient:   pubSubClient,
		commandFactory: commandFactory,
		logger:         logger,
		keyenckey:      keyenckey,
	}
}

func validateE4NameOrIDPair(name string, id []byte) ([]byte, error) {

	// The logic here is as follows:
	// 1. We can pass name AND/OR id
	// 2. If a name is passed and an ID, these should be consistent.
	// 3. If just a name is passed, derive the ID here.
	// 4. If a name is not passed, an empty string is acceptable
	//    (but all lookups must be by ID)
	//    This option will not be exposed to GRPC or HTTP APIs
	//    and is reserved for any future protocol.

	if len(name) != 0 {
		if len(id) != 0 {
			idtest := e4.HashIDAlias(name)
			if bytes.Equal(idtest, id) == false {
				return nil, fmt.Errorf("Inconsistent Name Alias and E4ID")
			}
			return id, nil
		}
		return e4.HashIDAlias(name), nil
	}

	if len(id) != e4.IDLen {
		return nil, fmt.Errorf("Incorrect ID Length")
	}
	return id, nil
}

func (s *e4impl) NewClient(name string, id, key []byte) error {
	logger := log.With(s.logger, "protocol", "e4", "command", "newClient")

	var newid []byte

	newid, err := validateE4NameOrIDPair(name, id)
	if err != nil {
		logger.Log("msg", "Inconsistent E4 ID/Alias, refusing insert")
		return err
	}

	protectedkey, err := e4.Encrypt(s.keyenckey, nil, key)
	if err != nil {
		logger.Log("msg", "failed to encrypt key", "error", err)
		return err
	}

	if err := s.db.InsertClient(name, newid, protectedkey); err != nil {
		logger.Log("msg", "insertClient failed", "error", err)
		return err
	}

	logger.Log("msg", "succeeded", "client", e4.PrettyID(newid))

	return nil
}

func (s *e4impl) RemoveClientByID(id []byte) error {
	logger := log.With(s.logger, "protocol", "e4", "command", "removeClient")

	err := s.db.DeleteClientByID(id)
	if err != nil {
		logger.Log("msg", "deleteClient failed", "error", err)
		return err
	}
	logger.Log("msg", "succeeded", "client", e4.PrettyID(id))
	return nil
}

func (s *e4impl) RemoveClientByName(name string) error {
	id := e4.HashIDAlias(name)
	return s.RemoveClientByID(id)
}

func (s *e4impl) NewTopicClient(name string, id []byte, topic string) error {
	logger := log.With(s.logger, "protocol", "e4", "command", "newTopicClient")

	if name != "" && len(id) == 0 {
		logger.Log("msg", "invalid ntc command received")
	}

	// A name or an ID can be passed to this function as well.
	// We will first attempt a lookup by name. If this fails,
	// we will pass the ID to the database first.
	// Callers
	var client models.Client
	var clientname string

	client, err := s.db.GetClientByName(name)
	if err != nil {
		// we failed to retrieve the client by name. Now we try ID:

		client, err = s.db.GetClientByID(id)
		if err != nil {
			logger.Log("msg", "failed to retrieve client", "error", err)
			return err
		}
		clientname = fmt.Sprintf("id = %s", e4.PrettyID(id))
	} else {
		clientname = fmt.Sprintf("name = %s", name)
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

	err = s.sendCommandToClient(command, client)
	if err != nil {
		logger.Log("msg", "sendCommandToClient failed", "error", err)
		return err
	}

	err = s.db.LinkClientTopic(client, topicKey)
	if err != nil {
		logger.Log("msg", "Database record of client-topic link failed", err)
		return err
	}

	logger.Log("msg", "succeeded", "client", clientname, "topic", topic, "topichash", topicKey.Hash())
	return nil
}

func (s *e4impl) RemoveTopicClientByID(id []byte, topic string) error {
	logger := log.With(s.logger, "protocol", "e4", "command", "removeTopicClient")

	client, err := s.db.GetClientByID(id)
	if err != nil {
		logger.Log("msg", "failed to retrieve client", "error", err)
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

	err = s.sendCommandToClient(command, client)
	if err != nil {
		logger.Log("msg", "sendCommandToClient failed", "error", err)
		return err
	}

	err = s.db.UnlinkClientTopic(client, topicKey)
	if err != nil {
		logger.Log("msg", "Cannot remove DB record of client-topic link", err)
		return err
	}

	logger.Log("msg", "succeeded", "topic", topic)

	return nil
}

func (s *e4impl) RemoveTopicClientByName(name string, topic string) error {
	id := e4.HashIDAlias(name)
	return s.RemoveTopicClientByID(id, topic)
}

func (s *e4impl) ResetClientByID(id []byte) error {
	logger := log.With(s.logger, "protocol", "e4", "command", "resetClient")

	client, err := s.db.GetClientByID(id)
	if err != nil {
		logger.Log("msg", "failed to retrieve client", "error", err)
		return err
	}

	command, err := s.commandFactory.CreateResetTopicsCommand()
	if err != nil {
		logger.Log("msg", "failed to create resetTopics command", "error", err)
		return err
	}

	err = s.sendCommandToClient(command, client)
	if err != nil {
		logger.Log("msg", "sendCommandToClient failed", "error", err)
		return err
	}

	logger.Log("msg", "succeeded", "client", e4.PrettyID(id))

	return nil
}

func (s *e4impl) ResetClientByName(name string) error {
	id := e4.HashIDAlias(name)
	return s.ResetClientByID(id)
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

	err = s.pubSubClient.SubscribeToTopic(topic) // Monitoring
	if err != nil {
		logger.Log("msg", "subscribeToTopic failed", "topic", topic, "error", err)
		return err
	}
	logger.Log("msg", "subscribeToTopic succeeded", "topic", topic)

	return nil
}

func (s *e4impl) RemoveTopic(topic string) error {
	logger := log.With(s.logger, "protocol", "e4", "command", "removeTopic")

	err := s.pubSubClient.UnsubscribeFromTopic(topic) // Monitoring
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
	err = s.pubSubClient.Publish(payload, topic, protocols.QoSAtMostOnce)
	if err != nil {
		logger.Log("msg", "publish failed", "error", err)
		return err
	}

	logger.Log("msg", "succeeded", "topic", topic)
	return nil
}

func (s *e4impl) NewClientKey(name string, id []byte) error {
	logger := log.With(s.logger, "protocol", "e4", "command", "newClientKey")

	newID, err := validateE4NameOrIDPair(name, id)
	if err != nil {
		logger.Log("msg", "Unable to validate name/id pair")
		return err
	}

	client, err := s.db.GetClientByID(newID)
	if err != nil {
		logger.Log("msg", "failed to retrieve client", "error", err)
		return err
	}

	newKey := e4.RandomKey()
	command, err := s.commandFactory.CreateSetIDKeyCommand(newKey)
	if err != nil {
		logger.Log("msg", "failed to create SetClient command", "error", err)
		return err
	}

	err = s.sendCommandToClient(command, client)
	if err != nil {
		logger.Log("msg", "sendCommandToClient failed", "error", err)
		return err
	}

	protectedkey, err := e4.Encrypt(s.keyenckey, nil, newKey)
	if err != nil {
		return err
	}

	err = s.db.InsertClient(name, newID, protectedkey)
	if err != nil {
		logger.Log("msg", "insertClient failed", "error", err)
		return err
	}
	logger.Log("msg", "succeeded", "id", e4.PrettyID(newID))

	return nil
}

func (s *e4impl) GetAllTopics() ([]string, error) {
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

// GetAllTopicsUnsafe returns *all* topics and should not be used
// from *ANY* API endpoint. This is for internal use only.
func (s *e4impl) GetAllTopicsUnsafe() ([]string, error) {
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

func (s *e4impl) GetAllClientsAsHexIDs() ([]string, error) {
	clients, err := s.db.GetAllClients()
	if err != nil {
		return nil, err
	}

	var hexids []string
	for _, client := range clients {
		hexids = append(hexids, hex.EncodeToString(client.E4ID))
	}

	return hexids, nil
}

func (s *e4impl) GetAllClientsAsNames() ([]string, error) {
	clients, err := s.db.GetAllClients()
	if err != nil {
		return nil, err
	}

	var names []string
	for _, client := range clients {
		names = append(names, client.Name)
	}

	return names, nil
}

func (s *e4impl) GetClientsAsHexIDsRange(offset, count int) ([]string, error) {
	clients, err := s.db.GetClientsRange(offset, count)
	if err != nil {
		return nil, err
	}

	var hexids []string
	for _, client := range clients {
		hexids = append(hexids, hex.EncodeToString(client.E4ID))
	}

	return hexids, nil
}

func (s *e4impl) GetClientsAsNamesRange(offset, count int) ([]string, error) {
	clients, err := s.db.GetClientsRange(offset, count)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, client := range clients {
		names = append(names, client.Name)
	}

	return names, nil
}

func (s *e4impl) GetTopicsRange(offset, count int) ([]string, error) {
	topics, err := s.db.GetTopicsRange(offset, count)
	if err != nil {
		return nil, err
	}

	var topicnames []string
	for _, topic := range topics {
		topicnames = append(topicnames, topic.Topic)
	}

	return topicnames, nil
}

func (s *e4impl) CountClients() (int, error) {
	return s.db.CountClients()
}

func (s *e4impl) CountTopics() (int, error) {
	return s.db.CountTopicKeys()
}

func (s *e4impl) CountTopicsForClientByID(id []byte) (int, error) {
	return s.db.CountTopicsForClientByID(id)
}

func (s *e4impl) CountTopicsForClientByName(name string) (int, error) {
	return s.db.CountTopicsForClientByName(name)
}

func (s *e4impl) GetTopicsForClientByID(id []byte, offset, count int) ([]string, error) {
	topicKeys, err := s.db.GetTopicsForClientByID(id, offset, count)
	if err != nil {
		return nil, err
	}

	var topics []string
	for _, topicKey := range topicKeys {
		topics = append(topics, topicKey.Topic)
	}

	return topics, nil
}

func (s *e4impl) GetTopicsForClientByName(name string, offset, count int) ([]string, error) {
	id := e4.HashIDAlias(name)
	return s.GetTopicsForClientByID(id, offset, count)
}

func (s *e4impl) CountClientsForTopic(topic string) (int, error) {
	return s.db.CountClientsForTopic(topic)
}

func (s *e4impl) GetClientsByNameForTopic(topic string, offset, count int) ([]string, error) {
	clients, err := s.db.GetClientsForTopic(topic, offset, count)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, client := range clients {
		names = append(names, client.Name)
	}

	return names, nil
}

func (s *e4impl) GetClientsByIDForTopic(topic string, offset, count int) ([]string, error) {
	clients, err := s.db.GetClientsForTopic(topic, offset, count)
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, client := range clients {
		ids = append(ids, hex.EncodeToString(client.E4ID))
	}

	return ids, nil
}

func (s *e4impl) sendCommandToClient(command commands.Command, client models.Client) error {
	clearKey, err := client.DecryptKey(s.keyenckey)
	if err != nil {
		return fmt.Errorf("failed to decrypt client: %v", err)
	}

	payload, err := command.Protect(clearKey)
	if err != nil {
		return fmt.Errorf("failed to protected command: %v", err)
	}

	return s.pubSubClient.Publish(payload, client.Topic(), protocols.QoSExactlyOnce)
}

// IsErrRecordNotFound indiquate whenever error is a RecordNotFound error
func IsErrRecordNotFound(err error) bool {
	return models.IsErrRecordNotFound(err)
}
