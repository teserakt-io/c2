package services

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"

	"github.com/go-kit/kit/log"
	"go.opencensus.io/trace"

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
	NewClient(ctx context.Context, name string, id, key []byte) error
	NewClientKey(ctx context.Context, name string, id []byte) error
	RemoveClientByID(ctx context.Context, id []byte) error
	RemoveClientByName(ctx context.Context, name string) error
	ResetClientByID(ctx context.Context, id []byte) error
	ResetClientByName(ctx context.Context, name string) error
	GetAllClientsAsHexIDs(ctx context.Context) ([]string, error)
	GetAllClientsAsNames(ctx context.Context) ([]string, error)
	GetClientsAsHexIDsRange(ctx context.Context, offset, count int) ([]string, error)
	GetClientsAsNamesRange(ctx context.Context, offset, count int) ([]string, error)
	CountClients(ctx context.Context) (int, error)

	// Individual Topic Manipulaton
	NewTopic(ctx context.Context, topic string) error
	RemoveTopic(ctx context.Context, topic string) error
	GetTopicsRange(ctx context.Context, offset, count int) ([]string, error)
	GetAllTopics(ctx context.Context) ([]string, error)
	GetAllTopicsUnsafe(ctx context.Context) ([]string, error)
	CountTopics(ctx context.Context) (int, error)

	// Linking, removing topic-client mappings:
	NewTopicClient(ctx context.Context, name string, id []byte, topic string) error
	RemoveTopicClientByID(ctx context.Context, id []byte, topic string) error
	RemoveTopicClientByName(ctx context.Context, name string, topic string) error

	// > Counting topics per client, or clients per topic.
	CountTopicsForClientByID(ctx context.Context, id []byte) (int, error)
	CountTopicsForClientByName(ctx context.Context, name string) (int, error)
	CountClientsForTopic(ctx context.Context, topic string) (int, error)

	// > Retrieving clients per topic or topics per client
	GetTopicsForClientByID(ctx context.Context, id []byte, offset, count int) ([]string, error)
	GetTopicsForClientByName(ctx context.Context, name string, offset, count int) ([]string, error)
	GetClientsByNameForTopic(ctx context.Context, topic string, offset, count int) ([]string, error)
	GetClientsByIDForTopic(ctx context.Context, topic string, offset, count int) ([]string, error)

	// Communications
	SendMessage(ctx context.Context, topic, msg string) error
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

func (s *e4impl) NewClient(ctx context.Context, name string, id, key []byte) error {
	ctx, span := trace.StartSpan(ctx, "e4.NewClient")
	defer span.End()

	logger := log.With(s.logger, "protocol", "e4", "command", "newClient")

	newID, err := validateE4NameOrIDPair(name, id)
	if err != nil {
		logger.Log("msg", "Inconsistent E4 ID/Alias, refusing insert")
		return err
	}

	protectedkey, err := e4.Encrypt(s.keyenckey, nil, key)
	if err != nil {
		logger.Log("msg", "failed to encrypt key", "error", err)
		return err
	}

	if err := s.db.InsertClient(name, newID, protectedkey); err != nil {
		logger.Log("msg", "insertClient failed", "error", err)
		return err
	}

	logger.Log("msg", "succeeded", "client", e4.PrettyID(newID))

	return nil
}

func (s *e4impl) RemoveClientByID(ctx context.Context, id []byte) error {
	ctx, span := trace.StartSpan(ctx, "e4.RemoveClientByID")
	defer span.End()

	logger := log.With(s.logger, "protocol", "e4", "command", "removeClient")

	err := s.db.DeleteClientByID(id)
	if err != nil {
		logger.Log("msg", "deleteClient failed", "error", err)
		return err
	}
	logger.Log("msg", "succeeded", "client", e4.PrettyID(id))
	return nil
}

func (s *e4impl) RemoveClientByName(ctx context.Context, name string) error {
	ctx, span := trace.StartSpan(ctx, "e4.RemoveClientByName")
	defer span.End()

	id := e4.HashIDAlias(name)
	return s.RemoveClientByID(ctx, id)
}

func (s *e4impl) NewTopicClient(ctx context.Context, name string, id []byte, topic string) error {
	ctx, span := trace.StartSpan(ctx, "e4.NewTopicClient")
	defer span.End()

	logger := log.With(s.logger, "protocol", "e4", "command", "newTopicClient")

	if name != "" && len(id) == 0 {
		logger.Log("msg", "invalid ntc command received")
	}

	newID, err := validateE4NameOrIDPair(name, id)
	if err != nil {
		logger.Log("msg", "Inconsistent E4 ID/Alias, refusing ntc")
		return err
	}

	client, err := s.db.GetClientByID(newID)
	if err != nil {
		logger.Log("msg", "failed to retrieve client", "id", newID, "error", err)
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

	err = s.sendCommandToClient(ctx, command, client)
	if err != nil {
		logger.Log("msg", "sendCommandToClient failed", "error", err)
		return err
	}

	err = s.db.LinkClientTopic(client, topicKey)
	if err != nil {
		logger.Log("msg", "Database record of client-topic link failed", err)
		return err
	}

	logger.Log(
		"msg", "succeeded",
		"clientID", client.E4ID, "clientName", client.Name,
		"topic", topic, "topichash", topicKey.Hash(),
	)

	return nil
}

func (s *e4impl) RemoveTopicClientByID(ctx context.Context, id []byte, topic string) error {
	ctx, span := trace.StartSpan(ctx, "e4.RemoveTopicClientByID")
	defer span.End()

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

	err = s.sendCommandToClient(ctx, command, client)
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

func (s *e4impl) RemoveTopicClientByName(ctx context.Context, name string, topic string) error {
	ctx, span := trace.StartSpan(ctx, "e4.RemoveTopicClientByName")
	defer span.End()

	id := e4.HashIDAlias(name)
	return s.RemoveTopicClientByID(ctx, id, topic)
}

func (s *e4impl) ResetClientByID(ctx context.Context, id []byte) error {
	ctx, span := trace.StartSpan(ctx, "e4.ResetClientByID")
	defer span.End()

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

	err = s.sendCommandToClient(ctx, command, client)
	if err != nil {
		logger.Log("msg", "sendCommandToClient failed", "error", err)
		return err
	}

	logger.Log("msg", "succeeded", "client", e4.PrettyID(id))

	return nil
}

func (s *e4impl) ResetClientByName(ctx context.Context, name string) error {
	ctx, span := trace.StartSpan(ctx, "e4.ResetClientByName")
	defer span.End()

	id := e4.HashIDAlias(name)
	return s.ResetClientByID(ctx, id)
}

func (s *e4impl) NewTopic(ctx context.Context, topic string) error {
	ctx, span := trace.StartSpan(ctx, "e4.NewTopic")
	defer span.End()

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

	err = s.pubSubClient.SubscribeToTopic(ctx, topic) // Monitoring
	if err != nil {
		logger.Log("msg", "subscribeToTopic failed", "topic", topic, "error", err)
		return err
	}
	logger.Log("msg", "subscribeToTopic succeeded", "topic", topic)

	return nil
}

func (s *e4impl) RemoveTopic(ctx context.Context, topic string) error {
	ctx, span := trace.StartSpan(ctx, "e4.RemoveTopic")
	defer span.End()

	logger := log.With(s.logger, "protocol", "e4", "command", "removeTopic")

	err := s.pubSubClient.UnsubscribeFromTopic(ctx, topic) // Monitoring
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

// SendMessage allows to publish an E4 protected message on the given topic
func (s *e4impl) SendMessage(ctx context.Context, topic, msg string) error {
	ctx, span := trace.StartSpan(ctx, "e4.SendMessage")
	defer span.End()

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
	err = s.pubSubClient.Publish(ctx, payload, topic, protocols.QoSAtMostOnce)
	if err != nil {
		logger.Log("msg", "publish failed", "error", err)
		return err
	}

	logger.Log("msg", "succeeded", "topic", topic)
	return nil
}

// NewClientKey will generate a new client key, send it to the client, and update the database.
func (s *e4impl) NewClientKey(ctx context.Context, name string, id []byte) error {
	ctx, span := trace.StartSpan(ctx, "e4.NewClientKey")
	defer span.End()

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

	err = s.sendCommandToClient(ctx, command, client)
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

// GetAllTopics will returns up to models.QueryLimit topics
func (s *e4impl) GetAllTopics(ctx context.Context) ([]string, error) {
	ctx, span := trace.StartSpan(ctx, "e4.GetAllTopics")
	defer span.End()

	topicKeys, err := s.db.GetAllTopics()
	if err != nil {
		return nil, err
	}

	topics := []string{}
	for _, topickey := range topicKeys {
		topics = append(topics, topickey.Topic)
	}
	return topics, nil
}

// GetAllTopicsUnsafe returns *all* topics and should not be used
// from *ANY* API endpoint. This is for internal use only.
func (s *e4impl) GetAllTopicsUnsafe(ctx context.Context) ([]string, error) {
	ctx, span := trace.StartSpan(ctx, "e4.GetAllTopicsUnsafe")
	defer span.End()

	topicKeys, err := s.db.GetAllTopicsUnsafe()
	if err != nil {
		return nil, err
	}

	topics := []string{}
	for _, topickey := range topicKeys {
		topics = append(topics, topickey.Topic)
	}
	return topics, nil
}

// GetAllClientsAsHexIDs returns up to models.QueryLimit client IDs, as hexadecimal string
func (s *e4impl) GetAllClientsAsHexIDs(ctx context.Context) ([]string, error) {
	ctx, span := trace.StartSpan(ctx, "e4.GetAllClientsAsHexIDs")
	defer span.End()

	clients, err := s.db.GetAllClients()
	if err != nil {
		return nil, err
	}

	hexids := []string{}
	for _, client := range clients {
		hexids = append(hexids, hex.EncodeToString(client.E4ID))
	}

	return hexids, nil
}

// GetAllClientsAsNames will retrieve up to models.QueryLimit client names.
func (s *e4impl) GetAllClientsAsNames(ctx context.Context) ([]string, error) {
	ctx, span := trace.StartSpan(ctx, "e4.GetAllClientsAsNames")
	defer span.End()

	clients, err := s.db.GetAllClients()
	if err != nil {
		return nil, err
	}

	names := []string{}
	for _, client := range clients {
		names = append(names, client.Name)
	}

	return names, nil
}

// GetClientsAsHexIDsRange allow to retrieve up to `count` clients IDs, as hex encoded string, starting from `offset`.
// The total count can be retrieved from CountClients()
func (s *e4impl) GetClientsAsHexIDsRange(ctx context.Context, offset, count int) ([]string, error) {
	ctx, span := trace.StartSpan(ctx, "e4.GetClientsAsHexIDsRange")
	defer span.End()

	clients, err := s.db.GetClientsRange(offset, count)
	if err != nil {
		return nil, err
	}

	hexids := []string{}
	for _, client := range clients {
		hexids = append(hexids, hex.EncodeToString(client.E4ID))
	}

	return hexids, nil
}

func (s *e4impl) GetClientsAsNamesRange(ctx context.Context, offset, count int) ([]string, error) {
	ctx, span := trace.StartSpan(ctx, "e4.GetClientsAsNamesRange")
	defer span.End()

	clients, err := s.db.GetClientsRange(offset, count)
	if err != nil {
		return nil, err
	}

	names := []string{}
	for _, client := range clients {
		names = append(names, client.Name)
	}

	return names, nil
}

func (s *e4impl) GetTopicsRange(ctx context.Context, offset, count int) ([]string, error) {
	ctx, span := trace.StartSpan(ctx, "e4.GetTopicsRange")
	defer span.End()

	topics, err := s.db.GetTopicsRange(offset, count)
	if err != nil {
		return nil, err
	}

	topicnames := []string{}
	for _, topic := range topics {
		topicnames = append(topicnames, topic.Topic)
	}

	return topicnames, nil
}

func (s *e4impl) CountClients(ctx context.Context) (int, error) {
	ctx, span := trace.StartSpan(ctx, "e4.CountClients")
	defer span.End()

	return s.db.CountClients()
}

func (s *e4impl) CountTopics(ctx context.Context) (int, error) {
	ctx, span := trace.StartSpan(ctx, "e4.CountTopics")
	defer span.End()

	return s.db.CountTopicKeys()
}

func (s *e4impl) CountTopicsForClientByID(ctx context.Context, id []byte) (int, error) {
	ctx, span := trace.StartSpan(ctx, "e4.CountTopicsForClientByID")
	defer span.End()

	return s.db.CountTopicsForClientByID(id)
}

func (s *e4impl) CountTopicsForClientByName(ctx context.Context, name string) (int, error) {
	ctx, span := trace.StartSpan(ctx, "e4.CountTopicsForClientByName")
	defer span.End()

	id := e4.HashIDAlias(name)
	return s.CountTopicsForClientByID(ctx, id)
}

func (s *e4impl) GetTopicsForClientByID(ctx context.Context, id []byte, offset, count int) ([]string, error) {
	ctx, span := trace.StartSpan(ctx, "e4.GetTopicsForClientByID")
	defer span.End()

	topicKeys, err := s.db.GetTopicsForClientByID(id, offset, count)
	if err != nil {
		return nil, err
	}

	topics := []string{}
	for _, topicKey := range topicKeys {
		topics = append(topics, topicKey.Topic)
	}

	return topics, nil
}

func (s *e4impl) GetTopicsForClientByName(ctx context.Context, name string, offset, count int) ([]string, error) {
	ctx, span := trace.StartSpan(ctx, "e4.GetTopicsForClientByName")
	defer span.End()

	id := e4.HashIDAlias(name)
	return s.GetTopicsForClientByID(ctx, id, offset, count)
}

func (s *e4impl) CountClientsForTopic(ctx context.Context, topic string) (int, error) {
	ctx, span := trace.StartSpan(ctx, "e4.CountClientsForTopic")
	defer span.End()

	return s.db.CountClientsForTopic(topic)
}

func (s *e4impl) GetClientsByNameForTopic(ctx context.Context, topic string, offset, count int) ([]string, error) {
	ctx, span := trace.StartSpan(ctx, "e4.GetClientsByNameForTopic")
	defer span.End()

	clients, err := s.db.GetClientsForTopic(topic, offset, count)
	if err != nil {
		return nil, err
	}

	names := []string{}

	for _, client := range clients {
		names = append(names, client.Name)
	}

	return names, nil
}

// GetClientsByIDForTopic returns a batch of client E4IDs which are subscribed to given topic
// The total count can be retrieved with CountClientsForTopic(), and offset / count must be provided
// to retrieve subset of client E4IDs
func (s *e4impl) GetClientsByIDForTopic(ctx context.Context, topic string, offset, count int) ([]string, error) {
	ctx, span := trace.StartSpan(ctx, "e4.GetClientsByIDForTopic")
	defer span.End()

	clients, err := s.db.GetClientsForTopic(topic, offset, count)
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for _, client := range clients {
		ids = append(ids, hex.EncodeToString(client.E4ID))
	}

	return ids, nil
}

func (s *e4impl) sendCommandToClient(ctx context.Context, command commands.Command, client models.Client) error {
	ctx, span := trace.StartSpan(ctx, "e4.sendCommandToClient")
	defer span.End()

	clearKey, err := client.DecryptKey(s.keyenckey)
	if err != nil {
		return fmt.Errorf("failed to decrypt client: %v", err)
	}

	payload, err := command.Protect(clearKey)
	if err != nil {
		return fmt.Errorf("failed to protected command: %v", err)
	}

	return s.pubSubClient.Publish(ctx, payload, client.Topic(), protocols.QoSExactlyOnce)
}

// IsErrRecordNotFound indiquate whenever error is a RecordNotFound error
func IsErrRecordNotFound(err error) bool {
	return models.IsErrRecordNotFound(err)
}
