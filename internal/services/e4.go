package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	"github.com/go-kit/kit/log"
	"go.opencensus.io/trace"

	"gitlab.com/teserakt/c2/internal/commands"
	"gitlab.com/teserakt/c2/internal/events"
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
	- By  = If the thing we are retrieving has multiple query specifiers,
	        By specifies how.
	- Range = if the item in question has offset/count limits, this specifier
	          is included.
*/

// IDNamePair stores an E4 client ID and names, omitting the key
type IDNamePair struct {
	ID   []byte
	Name string
}

// E4 describe the available methods on the E4 service
type E4 interface {
	// Client Only Manipulation
	NewClient(ctx context.Context, name string, id, key []byte) error
	NewClientKey(ctx context.Context, id []byte) error
	RemoveClient(ctx context.Context, id []byte) error
	ResetClient(ctx context.Context, id []byte) error
	GetClientsRange(ctx context.Context, offset, count int) ([]IDNamePair, error)
	CountClients(ctx context.Context) (int, error)

	// Individual Topic Manipulaton
	NewTopic(ctx context.Context, topic string) error
	RemoveTopic(ctx context.Context, topic string) error
	GetTopicsRange(ctx context.Context, offset, count int) ([]string, error)
	CountTopics(ctx context.Context) (int, error)

	// Linking, removing topic-client mappings:
	NewTopicClient(ctx context.Context, id []byte, topic string) error
	RemoveTopicClient(ctx context.Context, id []byte, topic string) error

	// > Counting topics per client, or clients per topic.
	CountTopicsForClient(ctx context.Context, id []byte) (int, error)
	CountClientsForTopic(ctx context.Context, topic string) (int, error)

	// > Retrieving clients per topic or topics per client
	GetTopicsRangeByClient(ctx context.Context, id []byte, offset, count int) ([]string, error)
	GetClientsRangeByTopic(ctx context.Context, topic string, offset, count int) ([]IDNamePair, error)

	// Communications
	SendMessage(ctx context.Context, topic, msg string) error
}

type e4impl struct {
	db              models.Database
	pubSubClient    protocols.PubSubClient
	commandFactory  commands.Factory
	logger          log.Logger
	keyenckey       []byte
	eventDispatcher events.Dispatcher
	eventFactory    events.Factory
}

var _ E4 = (*e4impl)(nil)

// NewE4 creates a new E4 service
func NewE4(
	db models.Database,
	pubSubClient protocols.PubSubClient,
	commandFactory commands.Factory,
	eventDispatcher events.Dispatcher,
	eventFactory events.Factory,
	logger log.Logger,
	keyenckey []byte,
) E4 {
	return &e4impl{
		db:              db,
		pubSubClient:    pubSubClient,
		commandFactory:  commandFactory,
		eventDispatcher: eventDispatcher,
		eventFactory:    eventFactory,
		logger:          logger,
		keyenckey:       keyenckey,
	}
}

func (s *e4impl) NewClient(ctx context.Context, name string, id, key []byte) error {
	ctx, span := trace.StartSpan(ctx, "e4.NewClient")
	defer span.End()

	logger := log.With(s.logger, "protocol", "e4", "command", "newClient", "name", name, "id", prettyID(id))
	newID, err := ValidateE4NameOrIDPair(name, id)
	if err != nil {
		logger.Log("msg", "Inconsistent E4 ID/Alias, refusing insert", "error", err)
		return ErrValidation{fmt.Errorf("inconsistent E4 ID/Name: %v", err)}
	}

	if err := e4.IsValidKey(key); err != nil {
		return ErrValidation{fmt.Errorf("invalid key: %v", err)}
	}

	protectedkey, err := e4.Encrypt(s.keyenckey, nil, key)
	if err != nil {
		logger.Log("msg", "failed to encrypt key", "error", err)
		return ErrInternal{}
	}

	if err := s.db.InsertClient(name, newID, protectedkey); err != nil {
		logger.Log("msg", "insertClient failed", "error", err)
		return ErrInternal{}
	}

	logger.Log("msg", "succeeded")

	return nil
}

func (s *e4impl) RemoveClient(ctx context.Context, id []byte) error {
	ctx, span := trace.StartSpan(ctx, "e4.RemoveClient")
	defer span.End()

	logger := log.With(s.logger, "protocol", "e4", "command", "removeClient", "id", prettyID(id))

	err := s.db.DeleteClientByID(id)
	if err != nil {
		logger.Log("msg", "deleteClient failed", "error", err)
		if models.IsErrRecordNotFound(err) {
			return ErrClientNotFound{}
		}
		return ErrInternal{}
	}

	logger.Log("msg", "succeeded")
	return nil
}

func (s *e4impl) NewTopicClient(ctx context.Context, id []byte, topic string) error {
	ctx, span := trace.StartSpan(ctx, "e4.NewTopicClient")
	defer span.End()

	logger := log.With(s.logger, "protocol", "e4", "command", "newTopicClient", "id", prettyID(id), "topic", topic)

	client, err := s.db.GetClientByID(id)
	if err != nil {
		logger.Log("msg", "failed to retrieve client", "error", err)
		if models.IsErrRecordNotFound(err) {
			return ErrClientNotFound{}
		}
		return ErrInternal{}
	}

	topicKey, err := s.db.GetTopicKey(topic)
	if err != nil {
		logger.Log("msg", "failed to retrieve topicKey", "error", err)
		if models.IsErrRecordNotFound(err) {
			return ErrTopicNotFound{}
		}
		return ErrInternal{}
	}

	clearTopicKey, err := topicKey.DecryptKey(s.keyenckey)
	if err != nil {
		logger.Log("msg", "failed to decrypt topicKey", "error", err)
		return ErrInternal{}
	}

	command, err := s.commandFactory.CreateSetTopicKeyCommand(topicKey.Hash(), clearTopicKey)
	if err != nil {
		logger.Log("msg", "failed to create setTopicKey command", "error", err)
		return ErrInternal{}
	}

	err = s.sendCommandToClient(ctx, command, client)
	if err != nil {
		logger.Log("msg", "sendCommandToClient failed", "error", err)
		return ErrInternal{}
	}

	err = s.db.LinkClientTopic(client, topicKey)
	if err != nil {
		logger.Log("msg", "Database record of client-topic link failed", err)
		return ErrInternal{}
	}

	s.eventDispatcher.Dispatch(s.eventFactory.NewClientSubscribedEvent(client.Name, topic))

	logger.Log(
		"msg", "succeeded",
		"clientName", client.Name,
		"topichash", topicKey.Hash(),
	)

	return nil
}

func (s *e4impl) RemoveTopicClient(ctx context.Context, id []byte, topic string) error {
	ctx, span := trace.StartSpan(ctx, "e4.RemoveTopicClient")
	defer span.End()

	logger := log.With(s.logger, "protocol", "e4", "command", "removeTopicClient", "id", prettyID(id), "topic", topic)

	client, err := s.db.GetClientByID(id)
	if err != nil {
		logger.Log("msg", "failed to retrieve client", "error", err)
		if models.IsErrRecordNotFound(err) {
			return ErrClientNotFound{}
		}
		return ErrInternal{}
	}

	topicKey, err := s.db.GetTopicKey(topic)
	if err != nil {
		logger.Log("msg", "failed to retrieve topicKey", "error", err)
		if models.IsErrRecordNotFound(err) {
			return ErrTopicNotFound{}
		}
		return ErrInternal{}
	}

	command, err := s.commandFactory.CreateRemoveTopicCommand(topicKey.Hash())
	if err != nil {
		logger.Log("msg", "failed to create removeTopic command", "error", err)
		return ErrInternal{}
	}

	err = s.sendCommandToClient(ctx, command, client)
	if err != nil {
		logger.Log("msg", "sendCommandToClient failed", "error", err)
		return ErrInternal{}
	}

	err = s.db.UnlinkClientTopic(client, topicKey)
	if err != nil {
		logger.Log("msg", "cannot remove DB record of client-topic link", "error", err)
		return ErrInternal{}
	}

	s.eventDispatcher.Dispatch(s.eventFactory.NewClientUnsubscribedEvent(client.Name, topic))

	logger.Log("msg", "succeeded")

	return nil
}

func (s *e4impl) ResetClient(ctx context.Context, id []byte) error {
	ctx, span := trace.StartSpan(ctx, "e4.ResetClient")
	defer span.End()

	logger := log.With(s.logger, "protocol", "e4", "command", "resetClient", "id", prettyID(id))

	client, err := s.db.GetClientByID(id)
	if err != nil {
		logger.Log("msg", "failed to retrieve client", "error", err)
		if models.IsErrRecordNotFound(err) {
			return ErrClientNotFound{}
		}
		return ErrInternal{}
	}

	command, err := s.commandFactory.CreateResetTopicsCommand()
	if err != nil {
		logger.Log("msg", "failed to create resetTopics command", "error", err)
		return ErrInternal{}
	}

	err = s.sendCommandToClient(ctx, command, client)
	if err != nil {
		logger.Log("msg", "sendCommandToClient failed", "error", err)
		return ErrInternal{}
	}

	logger.Log("msg", "succeeded")

	return nil
}

func (s *e4impl) NewTopic(ctx context.Context, topic string) error {
	ctx, span := trace.StartSpan(ctx, "e4.NewTopic")
	defer span.End()

	logger := log.With(s.logger, "protocol", "e4", "command", "newTopic", "topic", topic)

	key := e4.RandomKey()

	protectedKey, err := e4.Encrypt(s.keyenckey[:], nil, key)
	if err != nil {
		logger.Log("msg", "failed to encrypt key", "error", err)
		return ErrInternal{}
	}

	if err := s.db.InsertTopicKey(topic, protectedKey); err != nil {
		logger.Log("msg", "insertTopicKey failed", "error", err)
		return ErrInternal{}
	}
	logger.Log("msg", "insertTopicKey succeeded")

	err = s.pubSubClient.SubscribeToTopic(ctx, topic) // Monitoring
	if err != nil {
		logger.Log("msg", "subscribeToTopic failed", "error", err)
		return ErrInternal{}
	}
	logger.Log("msg", "subscribeToTopic succeeded")

	return nil
}

func (s *e4impl) RemoveTopic(ctx context.Context, topic string) error {
	ctx, span := trace.StartSpan(ctx, "e4.RemoveTopic")
	defer span.End()

	logger := log.With(s.logger, "protocol", "e4", "command", "removeTopic", "topic", topic)

	err := s.pubSubClient.UnsubscribeFromTopic(ctx, topic) // Monitoring
	if err != nil {
		logger.Log("msg", "unsubscribeFromTopic failed", "error", err)
	} else {
		logger.Log("msg", "unsubscribeFromTopic succeeded")
	}

	if err := s.db.DeleteTopicKey(topic); err != nil {
		logger.Log("msg", "deleteTopicKey failed", "error", err)
		if models.IsErrRecordNotFound(err) {
			return ErrTopicNotFound{}
		}
		return ErrInternal{}
	}
	logger.Log("msg", "succeeded")

	return nil
}

// SendMessage allows to publish an E4 protected message on the given topic
func (s *e4impl) SendMessage(ctx context.Context, topic, msg string) error {
	ctx, span := trace.StartSpan(ctx, "e4.SendMessage")
	defer span.End()

	logger := log.With(s.logger, "protocol", "e4", "command", "sendMessage", "topic", topic)

	topicKey, err := s.db.GetTopicKey(topic)
	if err != nil {
		logger.Log("msg", "failed to retrieve topicKey", "error", err)
		if models.IsErrRecordNotFound(err) {
			return ErrTopicNotFound{}
		}
		return ErrInternal{}
	}

	clearTopicKey, err := topicKey.DecryptKey(s.keyenckey)
	if err != nil {
		logger.Log("msg", "failed to decrypt topicKey", "error", err)
		return ErrInternal{}
	}

	payload, err := e4.Protect([]byte(msg), clearTopicKey)
	if err != nil {
		logger.Log("msg", "Protect failed", "error", err)
		return ErrInternal{}
	}
	err = s.pubSubClient.Publish(ctx, payload, topic, protocols.QoSAtMostOnce)
	if err != nil {
		logger.Log("msg", "publish failed", "error", err)
		return ErrInternal{}
	}

	logger.Log("msg", "succeeded")
	return nil
}

// NewClientKey will generate a new client key, send it to the client, and update the database.
func (s *e4impl) NewClientKey(ctx context.Context, id []byte) error {
	ctx, span := trace.StartSpan(ctx, "e4.NewClientKey")
	defer span.End()

	logger := log.With(s.logger, "protocol", "e4", "command", "newClientKey", "id", prettyID(id))

	client, err := s.db.GetClientByID(id)
	if err != nil {
		logger.Log("msg", "failed to retrieve client", "error", err)
		if models.IsErrRecordNotFound(err) {
			return ErrClientNotFound{}
		}
		return ErrInternal{}
	}

	newKey := e4.RandomKey()
	command, err := s.commandFactory.CreateSetIDKeyCommand(newKey)
	if err != nil {
		logger.Log("msg", "failed to create SetClient command", "error", err)
		return ErrInternal{}
	}

	err = s.sendCommandToClient(ctx, command, client)
	if err != nil {
		logger.Log("msg", "sendCommandToClient failed", "error", err)
		return ErrInternal{}
	}

	protectedkey, err := e4.Encrypt(s.keyenckey, nil, newKey)
	if err != nil {
		return ErrInternal{}
	}

	err = s.db.InsertClient(client.Name, id, protectedkey)
	if err != nil {
		logger.Log("msg", "insertClient failed", "error", err)
		return ErrInternal{}
	}
	logger.Log("msg", "succeeded")

	return nil
}

func (s *e4impl) GetClientsRange(ctx context.Context, offset, count int) ([]IDNamePair, error) {
	ctx, span := trace.StartSpan(ctx, "e4.GetClientsRange")
	defer span.End()

	logger := log.With(s.logger, "protocol", "e4", "command", "getClientsRange", "offset", offset, "count", count)

	clients, err := s.db.GetClientsRange(offset, count)
	if err != nil {
		logger.Log("msg", "failed to retrieve clients", "error", err)
		return nil, ErrInternal{}
	}

	idNamePairs := make([]IDNamePair, 0, len(clients))
	for _, client := range clients {
		idNamePairs = append(idNamePairs, IDNamePair{ID: client.E4ID, Name: client.Name})
	}

	logger.Log("msg", "succeeded", "count", len(idNamePairs))

	return idNamePairs, nil
}

func (s *e4impl) GetTopicsRange(ctx context.Context, offset, count int) ([]string, error) {
	ctx, span := trace.StartSpan(ctx, "e4.GetTopicsRange")
	defer span.End()

	logger := log.With(s.logger, "protocol", "e4", "command", "getTopicsRange", "offset", offset, "count", count)

	topics, err := s.db.GetTopicsRange(offset, count)
	if err != nil {
		logger.Log("msg", "failed to retrieve topics", "error", err)
		return nil, ErrInternal{}
	}

	topicNames := []string{}
	for _, topic := range topics {
		topicNames = append(topicNames, topic.Topic)
	}

	logger.Log("msg", "succeeded", "count", len(topicNames))

	return topicNames, nil
}

func (s *e4impl) CountClients(ctx context.Context) (int, error) {
	ctx, span := trace.StartSpan(ctx, "e4.CountClients")
	defer span.End()

	logger := log.With(s.logger, "protocol", "e4", "command", "countClients")

	count, err := s.db.CountClients()
	if err != nil {
		logger.Log("msg", "failed to count clients", "error", err)
		return 0, ErrInternal{}
	}

	logger.Log("msg", "succeeded", "count", count)

	return count, nil
}

func (s *e4impl) CountTopics(ctx context.Context) (int, error) {
	ctx, span := trace.StartSpan(ctx, "e4.CountTopics")
	defer span.End()

	logger := log.With(s.logger, "protocol", "e4", "command", "countTopics")

	count, err := s.db.CountTopicKeys()
	if err != nil {
		logger.Log("msg", "failed to count clients", "error", err)
		return 0, ErrInternal{}
	}

	logger.Log("msg", "succeeded", "count", count)

	return count, nil
}

func (s *e4impl) CountTopicsForClient(ctx context.Context, id []byte) (int, error) {
	ctx, span := trace.StartSpan(ctx, "e4.CountTopicsForClient")
	defer span.End()

	logger := log.With(s.logger, "protocol", "e4", "command", "countTopicsForClient", "id", prettyID(id))

	count, err := s.db.CountTopicsForClientByID(id)
	if err != nil {
		logger.Log("msg", "failed to count topics for client", "error", err)
		if models.IsErrRecordNotFound(err) {
			return 0, ErrClientNotFound{}
		}
		return 0, ErrInternal{}
	}

	logger.Log("msg", "succeeded", "count", count)

	return count, nil
}

func (s *e4impl) GetTopicsRangeByClient(ctx context.Context, id []byte, offset, count int) ([]string, error) {
	ctx, span := trace.StartSpan(ctx, "e4.GetTopicsRangeByClient")
	defer span.End()

	logger := log.With(s.logger, "protocol", "e4", "command", "getTopicsRangeByClient", "id", prettyID(id))

	topicKeys, err := s.db.GetTopicsForClientByID(id, offset, count)
	if err != nil {
		logger.Log("msg", "failed to get topics for client", "error", err)
		if models.IsErrRecordNotFound(err) {
			return nil, ErrClientNotFound{}
		}
		return nil, ErrInternal{}
	}

	topics := []string{}
	for _, topicKey := range topicKeys {
		topics = append(topics, topicKey.Topic)
	}

	logger.Log("msg", "succeeded", "topicCount", len(topics))

	return topics, nil
}

func (s *e4impl) CountClientsForTopic(ctx context.Context, topic string) (int, error) {
	ctx, span := trace.StartSpan(ctx, "e4.CountClientsForTopic")
	defer span.End()

	logger := log.With(s.logger, "protocol", "e4", "command", "countClientsForTopic", "topic", topic)

	count, err := s.db.CountClientsForTopic(topic)
	if err != nil {
		logger.Log("msg", "failed to count clients for topic", "error", err)
		if models.IsErrRecordNotFound(err) {
			return 0, ErrTopicNotFound{}
		}
		return 0, ErrInternal{}
	}

	logger.Log("msg", "succeeded", "count", count)

	return count, nil
}

func (s *e4impl) GetClientsRangeByTopic(ctx context.Context, topic string, offset, count int) ([]IDNamePair, error) {
	ctx, span := trace.StartSpan(ctx, "e4.GetClientsRangeByTopic")
	defer span.End()

	logger := log.With(s.logger, "protocol", "e4", "command", "getClientsRangeByTopic", "topic", topic)

	clients, err := s.db.GetClientsForTopic(topic, offset, count)
	if err != nil {
		logger.Log("msg", "failed to get clients for topic", "error", err)
		if models.IsErrRecordNotFound(err) {
			return nil, ErrTopicNotFound{}
		}
		return nil, ErrInternal{}
	}

	idNamePairs := make([]IDNamePair, 0, len(clients))
	for _, client := range clients {
		idNamePairs = append(idNamePairs, IDNamePair{ID: client.E4ID, Name: client.Name})
	}

	logger.Log("msg", "succeeded", "clientCount", len(idNamePairs))

	return idNamePairs, nil
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
		return fmt.Errorf("failed to protect command: %v", err)
	}

	return s.pubSubClient.Publish(ctx, payload, client.Topic(), protocols.QoSExactlyOnce)
}

// IsErrRecordNotFound indiquate whenever error is a RecordNotFound error
func IsErrRecordNotFound(err error) bool {
	return models.IsErrRecordNotFound(err)
}

// ValidateE4NameOrIDPair will check the following logic:
// 1. We can pass name AND/OR id
// 2. If a name is passed and an ID, these should be consistent.
// 3. If just a name is passed, derive the ID here.
// 4. If a name is not passed, an empty string is acceptable
//    (but all lookups must be by ID)
//    This option will not be exposed to GRPC or HTTP APIs
//    and is reserved for any future protocol.
func ValidateE4NameOrIDPair(name string, id []byte) ([]byte, error) {
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
		return nil, fmt.Errorf("Incorrect ID Length, expected %d bytes, got %d", e4.IDLen, len(id))
	}
	return id, nil
}

func prettyID(id []byte) string {
	return base64.StdEncoding.EncodeToString(id)
}
