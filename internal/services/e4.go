// Copyright 2020 Teserakt AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"golang.org/x/crypto/ed25519"

	e4crypto "github.com/teserakt-io/e4go/crypto"

	"github.com/teserakt-io/c2/internal/commands"
	"github.com/teserakt-io/c2/internal/config"
	"github.com/teserakt-io/c2/internal/crypto"
	"github.com/teserakt-io/c2/internal/events"
	"github.com/teserakt-io/c2/internal/models"
	"github.com/teserakt-io/c2/internal/protocols"
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

// NewTopicBatchSize is the maximum number of topic clients pulled from the database at a time
// when sending them the new topic key.
// Having it too low will create too many queries, and too high will create too slow queries.
var NewTopicBatchSize = 500

// NewC2KeyBatchSize is the maximum number of clients pulled from the database at a time
// when sending them the new C2 key.
var NewC2KeyBatchSize = 500

// GetLinkedClientsBatchSize is the maximum number of clients pulled from the database at a time
// when sending them the new client pubkey.
var GetLinkedClientsBatchSize = 500

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

	// Counting topics per client, or clients per topic.
	CountTopicsForClient(ctx context.Context, id []byte) (int, error)
	CountClientsForTopic(ctx context.Context, topic string) (int, error)

	// Retrieving clients per topic or topics per client
	GetTopicsRangeByClient(ctx context.Context, id []byte, offset, count int) ([]string, error)
	GetClientsRangeByTopic(ctx context.Context, topic string, offset, count int) ([]IDNamePair, error)

	// Clients linking / unlinking / counting linked
	LinkClient(ctx context.Context, sourceClientID, targetClientID []byte) error
	UnlinkClient(ctx context.Context, sourceClientID, targetClientID []byte) error
	CountLinkedClients(ctx context.Context, id []byte) (int, error)
	GetLinkedClients(ctx context.Context, id []byte, offset, count int) ([]IDNamePair, error)

	// SendClientPubKey send the sourceClientID public key to targetClientID via a SetPubKeyCmd.
	// Only when C2 is configured in pubkey mode, otherwise an error will be immediately returned.
	SendClientPubKey(ctx context.Context, sourceClientID, targetClientID []byte) error
	// RemoveClientPubKey removes the sourceClientID public key from targetClientID via a RemovePubKeyCmd.
	// Only when C2 is configured in pubkey mode, otherwise an error will be immediately returned.
	RemoveClientPubKey(ctx context.Context, sourceClientID, targetClientID []byte) error
	// ResetClientPubKeys removes all public keys stored on targetClientID via a ResetPubKeyCmd
	// Only when C2 is configured in pubkey mode, otherwise an error will be immediately returned.
	ResetClientPubKeys(ctx context.Context, targetClientID []byte) error
	// NewC2Key generates a random C2 key pair and the public key is sent to all clients via a SetC2KeyCmd.
	// The new key pair is then used by the C2 and replaces the previous one.
	// Only when C2 is configured in pubkey mode, otherwise an error will be immediately returned.
	NewC2Key(ctx context.Context) error
	// ProtectMessage protects the given data with the given topic's key.
	ProtectMessage(ctx context.Context, topic string, data []byte) ([]byte, error)
}

type e4impl struct {
	db              models.Database
	pubSubClient    protocols.PubSubClient
	commandFactory  commands.Factory
	e4Key           crypto.E4Key
	dbEncKey        []byte
	logger          log.FieldLogger
	eventDispatcher events.Dispatcher
	eventFactory    events.Factory
	cfg             config.CryptoCfg
}

var _ E4 = (*e4impl)(nil)

// NewE4 creates a new E4 service
func NewE4(
	db models.Database,
	pubSubClient protocols.PubSubClient,
	commandFactory commands.Factory,
	eventDispatcher events.Dispatcher,
	eventFactory events.Factory,
	e4Key crypto.E4Key,
	logger log.FieldLogger,
	dbEncKey []byte,
	cfg config.CryptoCfg,
) E4 {
	return &e4impl{
		db:              db,
		pubSubClient:    pubSubClient,
		commandFactory:  commandFactory,
		eventDispatcher: eventDispatcher,
		eventFactory:    eventFactory,
		e4Key:           e4Key,
		logger:          logger,
		dbEncKey:        dbEncKey,
		cfg:             cfg,
	}
}

func (s *e4impl) NewClient(ctx context.Context, name string, id, key []byte) error {
	_, span := trace.StartSpan(ctx, "e4.NewClient")
	defer span.End()

	logger := s.logger.WithFields(log.Fields{
		"name": name,
		"id":   prettyID(id),
	})

	newID, err := ValidateE4NameOrIDPair(name, id)
	if err != nil {
		logger.WithError(err).Error("inconsistent E4 ID/Alias, refusing insert")
		return ErrValidation{fmt.Errorf("inconsistent E4 ID/Name: %v", err)}
	}

	if err := s.e4Key.ValidateKey(key); err != nil {
		return ErrValidation{fmt.Errorf("invalid key: %v", err)}
	}

	protectedkey, err := e4crypto.Encrypt(s.dbEncKey, nil, key)
	if err != nil {
		logger.WithError(err).Error("failed to encrypt key")
		return ErrInternal{}
	}

	if err := s.db.InsertClient(name, newID, protectedkey); err != nil {
		logger.WithError(err).Error("failed to insert client")
		return ErrInternal{}
	}

	logger.Info("succeeded")

	return nil
}

func (s *e4impl) RemoveClient(ctx context.Context, id []byte) error {
	_, span := trace.StartSpan(ctx, "e4.RemoveClient")
	defer span.End()

	logger := s.logger.WithField("id", prettyID(id))

	err := s.db.DeleteClientByID(id)
	if err != nil {
		logger.WithError(err).Error("failed to delete client")
		if models.IsErrRecordNotFound(err) {
			return ErrClientNotFound{}
		}
		return ErrInternal{}
	}

	logger.Info("succeeded")
	return nil
}

func (s *e4impl) NewTopicClient(ctx context.Context, id []byte, topic string) error {
	ctx, span := trace.StartSpan(ctx, "e4.NewTopicClient")
	defer span.End()

	logger := s.logger.WithFields(log.Fields{
		"id":    prettyID(id),
		"topic": topic,
	})

	client, err := s.db.GetClientByID(id)
	if err != nil {
		logger.WithError(err).Error("failed to retrieve client")
		if models.IsErrRecordNotFound(err) {
			return ErrClientNotFound{}
		}
		return ErrInternal{}
	}

	topicKey, err := s.db.GetTopicKey(topic)
	if err != nil {
		logger.WithError(err).Error("failed to retrieve topicKey")
		if models.IsErrRecordNotFound(err) {
			return ErrTopicNotFound{}
		}
		return ErrInternal{}
	}

	clearTopicKey, err := topicKey.DecryptKey(s.dbEncKey)
	if err != nil {
		logger.WithError(err).Error("failed to decrypt topicKey")
		return ErrInternal{}
	}

	command, err := s.commandFactory.CreateSetTopicKeyCommand(topicKey.Topic, clearTopicKey)
	if err != nil {
		logger.WithError(err).Error("failed to create setTopicKey command")
		return ErrInternal{}
	}

	err = s.sendCommandToClient(ctx, command, client)
	if err != nil {
		logger.WithError(err).Error("failed to send command to client")
		return ErrInternal{}
	}

	err = s.db.LinkClientTopic(client, topicKey)
	if err != nil {
		logger.WithError(err).Error("saving client-topic link failed")
		return ErrInternal{}
	}

	s.eventDispatcher.Dispatch(s.eventFactory.NewClientSubscribedEvent(client.Name, topic))

	logger.Info("succeeded")

	return nil
}

func (s *e4impl) RemoveTopicClient(ctx context.Context, id []byte, topic string) error {
	ctx, span := trace.StartSpan(ctx, "e4.RemoveTopicClient")
	defer span.End()

	logger := s.logger.WithFields(log.Fields{
		"id":    prettyID(id),
		"topic": topic,
	})

	client, err := s.db.GetClientByID(id)
	if err != nil {
		logger.WithError(err).Error("failed to retrieve client")
		if models.IsErrRecordNotFound(err) {
			return ErrClientNotFound{}
		}
		return ErrInternal{}
	}

	topicKey, err := s.db.GetTopicKey(topic)
	if err != nil {
		logger.WithError(err).Error("failed to retrieve topicKey")
		if models.IsErrRecordNotFound(err) {
			return ErrTopicNotFound{}
		}
		return ErrInternal{}
	}

	command, err := s.commandFactory.CreateRemoveTopicCommand(topicKey.Topic)
	if err != nil {
		logger.WithError(err).Error("failed to create removeTopic command")
		return ErrInternal{}
	}

	err = s.sendCommandToClient(ctx, command, client)
	if err != nil {
		logger.WithError(err).Error("sendCommandToClient failed")
		return ErrInternal{}
	}

	err = s.db.UnlinkClientTopic(client, topicKey)
	if err != nil {
		logger.WithError(err).Error("cannot remove DB record of client-topic link")
		return ErrInternal{}
	}

	s.eventDispatcher.Dispatch(s.eventFactory.NewClientUnsubscribedEvent(client.Name, topic))

	logger.Info("succeeded")

	return nil
}

func (s *e4impl) ResetClient(ctx context.Context, id []byte) error {
	ctx, span := trace.StartSpan(ctx, "e4.ResetClient")
	defer span.End()

	logger := s.logger.WithField("id", prettyID(id))

	client, err := s.db.GetClientByID(id)
	if err != nil {
		logger.WithError(err).Error("failed to retrieve client")
		if models.IsErrRecordNotFound(err) {
			return ErrClientNotFound{}
		}
		return ErrInternal{}
	}

	command, err := s.commandFactory.CreateResetTopicsCommand()
	if err != nil {
		logger.WithError(err).Error("failed to create resetTopics command")
		return ErrInternal{}
	}

	err = s.sendCommandToClient(ctx, command, client)
	if err != nil {
		logger.WithError(err).Error("sendCommandToClient failed")
		return ErrInternal{}
	}

	logger.Info("succeeded")

	return nil
}

func (s *e4impl) NewTopic(ctx context.Context, topic string) error {
	ctx, span := trace.StartSpan(ctx, "e4.NewTopic")
	defer span.End()

	logger := s.logger.WithField("topic", topic)

	if err := s.pubSubClient.ValidateTopic(topic); err != nil {
		return err
	}

	key := e4crypto.RandomKey()

	protectedKey, err := e4crypto.Encrypt(s.dbEncKey[:], nil, key)
	if err != nil {
		logger.WithError(err).Error("failed to encrypt key")
		return ErrInternal{}
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		logger.WithError(err).Error("failed to begin transaction")
		return ErrInternal{}
	}

	if err := tx.InsertTopicKey(topic, protectedKey); err != nil {
		logger.WithError(err).Error("insertTopicKey failed")
		if err := tx.Rollback(); err != nil {
			logger.WithError(err).Error("failed to rollback transaction")
		}

		return ErrInternal{}
	}
	logger.Info("insertTopicKey succeeded")

	command, err := s.commandFactory.CreateSetTopicKeyCommand(topic, key)
	if err != nil {
		logger.WithError(err).Error("failed to create setTopicKey command")
		if err := tx.Rollback(); err != nil {
			logger.WithError(err).Error("failed to rollback transaction")
		}
		return ErrInternal{}
	}

	clientCount, err := tx.CountClientsForTopic(topic)
	if err != nil {
		logger.WithError(err).Error("failed to count topic clients")
		if err := tx.Rollback(); err != nil {
			logger.WithError(err).Error("failed to rollback transaction")
		}
		return ErrInternal{}
	}

	if err := tx.CommitTx(); err != nil {
		logger.WithError(err).Error("failed to commit transaction")
		return ErrInternal{}
	}

	s.sendCommandToTopicClients(ctx, command, topic, clientCount)

	err = s.pubSubClient.SubscribeToTopic(ctx, topic) // Monitoring
	if err != nil {
		logger.WithError(err).Error("subscribeToTopic failed")
		return ErrInternal{}
	}
	logger.Info("subscribeToTopic succeeded")

	return nil
}

// sendCommandToTopicClients will send the given command to all clientCount clients of given topic,
// by fetching NewTopicBatchSize from the DB at a time.
// If any error happen and prevent the key to be sent to the client, the error is just logged for now
// with all metadata to allow reforging and publishing the message.
func (s *e4impl) sendCommandToTopicClients(ctx context.Context, command commands.Command, topic string, clientCount int) {
	logger := s.logger.WithField("topic", topic)

	cmdType, err := command.Type()
	if err != nil {
		logger.WithError(err).Error("failed to get command type")

		return
	}

	logger = logger.WithField("command", cmdType)

	ctx, span := trace.StartSpan(ctx, "e4.sendCommandToTopicClients")
	defer span.End()

	for offset := 0; offset < clientCount; offset += NewTopicBatchSize {
		span.Annotate([]trace.Attribute{
			trace.Int64Attribute("offset", int64(offset)),
			trace.Int64Attribute("command", int64(cmdType)),
			trace.Int64Attribute("clientCount", int64(clientCount)),
			trace.Int64Attribute("batchSize", int64(NewTopicBatchSize)),
		}, "e4.sendCommandToTopicClients")

		clients, err := s.db.GetClientsForTopic(topic, offset, NewTopicBatchSize)
		if err != nil {
			logger.WithError(err).WithFields(log.Fields{
				"offset":    offset,
				"batchSize": NewTopicBatchSize,
			}).Error("failed to retrieve topic clients")

			continue
		}

		for _, client := range clients {
			err = s.sendCommandToClient(ctx, command, client)
			if err != nil {
				logger.WithError(err).WithField("client", client.Name).Error("sendCommandToClient failed")

				continue
			}
		}

		logger.WithFields(log.Fields{
			"offset":    offset,
			"batchSize": NewTopicBatchSize,
		}).Info("successfully sent command to clients")
	}
}

func (s *e4impl) RemoveTopic(ctx context.Context, topic string) error {
	ctx, span := trace.StartSpan(ctx, "e4.RemoveTopic")
	defer span.End()

	logger := s.logger.WithField("topic", topic)

	err := s.pubSubClient.UnsubscribeFromTopic(ctx, topic) // Monitoring
	if err != nil {
		logger.WithError(err).Warn("UnsubscribeFromTopic failed")
	} else {
		logger.Debug("UnsubscribeFromTopic succeeded")
	}

	if err := s.db.DeleteTopicKey(topic); err != nil {
		logger.WithError(err).Error("deleteTopicKey failed")
		if models.IsErrRecordNotFound(err) {
			return ErrTopicNotFound{}
		}
		return ErrInternal{}
	}
	logger.Info("succeeded")

	return nil
}

// NewClientKey will generate a new client key, send it to the client, and update the database.
func (s *e4impl) NewClientKey(ctx context.Context, id []byte) error {
	ctx, span := trace.StartSpan(ctx, "e4.NewClientKey")
	defer span.End()

	logger := s.logger.WithField("id", prettyID(id))

	client, err := s.db.GetClientByID(id)
	if err != nil {
		logger.WithError(err).Error("failed to retrieve client")
		if models.IsErrRecordNotFound(err) {
			return ErrClientNotFound{}
		}
		return ErrInternal{}
	}

	clientKey, c2StoredKey, err := s.e4Key.RandomKey()
	if err != nil {
		logger.WithError(err).Error("failed to generate new key")
		return ErrInternal{}
	}

	command, err := s.commandFactory.CreateSetIDKeyCommand(clientKey)
	if err != nil {
		logger.WithError(err).Error("failed to create SetClient command")
		return ErrInternal{}
	}

	err = s.sendCommandToClient(ctx, command, client)
	if err != nil {
		logger.WithError(err).Error("failed to send command to client")
		return ErrInternal{}
	}

	protectedkey, err := e4crypto.Encrypt(s.dbEncKey, nil, c2StoredKey)
	if err != nil {
		logger.WithError(err).Error("encrypt failed")
		return ErrInternal{}
	}

	err = s.db.InsertClient(client.Name, id, protectedkey)
	if err != nil {
		logger.WithError(err).Error("insertClient failed")
		return ErrInternal{}
	}

	// PubKey mode requires to send the new public key to linked clients
	if s.cfg.NewClientKeySendPubkey && s.e4Key.IsPubKeyMode() {
		if err := s.sendNewPubKeyToLinkedClients(ctx, client, c2StoredKey); err != nil {
			logger.WithError(err).Error("failed to send new pubkey to linked clients")
			return ErrInternal{}
		}
	}

	logger.Info("succeeded")

	return nil
}

func (s *e4impl) sendNewPubKeyToLinkedClients(ctx context.Context, client models.Client, newPubKey ed25519.PublicKey) error {
	ctx, span := trace.StartSpan(ctx, "e4.sendNewPubKeyToLinkedClients")
	defer span.End()

	logger := s.logger.WithField("id", prettyID(client.E4ID))

	cmd, err := s.commandFactory.CreateSetPubKeyCommand(newPubKey, client.Name)
	if err != nil {
		return fmt.Errorf("failed to create SetPubKey command: %v", err)
	}

	linkedClientsCount, err := s.db.CountLinkedClients(client.E4ID)
	if err != nil {
		return fmt.Errorf("failed to get client count: %v", err)
	}

	for offset := 0; offset < linkedClientsCount; offset += GetLinkedClientsBatchSize {
		linkedClients, err := s.db.GetLinkedClientsForClientByID(client.E4ID, offset, GetLinkedClientsBatchSize)
		if err != nil {
			logger.WithError(err).Error("failed to retrieve linked clients")
			continue
		}

		for _, linkedClient := range linkedClients {
			if err := s.sendCommandToClient(ctx, cmd, linkedClient); err != nil {
				logger.WithField("linkedClient", prettyID(linkedClient.E4ID)).
					WithError(err).
					Error("failed to send new client public key to linked client")
				continue
			}
		}
	}

	return nil
}

func (s *e4impl) GetClientsRange(ctx context.Context, offset, count int) ([]IDNamePair, error) {
	_, span := trace.StartSpan(ctx, "e4.GetClientsRange")
	defer span.End()

	logger := s.logger.WithFields(log.Fields{
		"offset": offset,
		"count":  count,
	})

	clients, err := s.db.GetClientsRange(offset, count)
	if err != nil {
		logger.WithError(err).Error("failed to retrieve clients")
		return nil, ErrInternal{}
	}

	idNamePairs := make([]IDNamePair, 0, len(clients))
	for _, client := range clients {
		idNamePairs = append(idNamePairs, IDNamePair{ID: client.E4ID, Name: client.Name})
	}

	logger.WithField("total", len(idNamePairs)).Info("succeeded")

	return idNamePairs, nil
}

func (s *e4impl) GetTopicsRange(ctx context.Context, offset, count int) ([]string, error) {
	_, span := trace.StartSpan(ctx, "e4.GetTopicsRange")
	defer span.End()

	logger := s.logger.WithFields(log.Fields{
		"offset": offset,
		"count":  count,
	})

	topics, err := s.db.GetTopicsRange(offset, count)
	if err != nil {
		logger.WithError(err).Error("failed to retrieve topics")
		return nil, ErrInternal{}
	}

	topicNames := []string{}
	for _, topic := range topics {
		topicNames = append(topicNames, topic.Topic)
	}

	logger.WithField("total", len(topicNames)).Info("succeeded")

	return topicNames, nil
}

func (s *e4impl) CountClients(ctx context.Context) (int, error) {
	_, span := trace.StartSpan(ctx, "e4.CountClients")
	defer span.End()

	count, err := s.db.CountClients()
	if err != nil {
		s.logger.WithError(err).Error("failed to count clients")
		return 0, ErrInternal{}
	}

	s.logger.WithField("total", count).Info("succeeded")

	return count, nil
}

func (s *e4impl) CountTopics(ctx context.Context) (int, error) {
	_, span := trace.StartSpan(ctx, "e4.CountTopics")
	defer span.End()

	count, err := s.db.CountTopicKeys()
	if err != nil {
		s.logger.WithError(err).Error("failed to count clients")
		return 0, ErrInternal{}
	}

	s.logger.WithField("total", count).Info("succeeded")

	return count, nil
}

func (s *e4impl) CountTopicsForClient(ctx context.Context, id []byte) (int, error) {
	_, span := trace.StartSpan(ctx, "e4.CountTopicsForClient")
	defer span.End()

	logger := s.logger.WithField("id", prettyID(id))

	count, err := s.db.CountTopicsForClientByID(id)
	if err != nil {
		logger.WithError(err).Error("failed to count topics for client")
		if models.IsErrRecordNotFound(err) {
			return 0, ErrClientNotFound{}
		}
		return 0, ErrInternal{}
	}

	logger.WithField("total", count).Info("succeeded")

	return count, nil
}

func (s *e4impl) GetTopicsRangeByClient(ctx context.Context, id []byte, offset, count int) ([]string, error) {
	_, span := trace.StartSpan(ctx, "e4.GetTopicsRangeByClient")
	defer span.End()

	logger := s.logger.WithFields(log.Fields{
		"id":     prettyID(id),
		"offset": offset,
		"count":  count,
	})

	topicKeys, err := s.db.GetTopicsForClientByID(id, offset, count)
	if err != nil {
		logger.WithError(err).Error("failed to get topics for client")
		if models.IsErrRecordNotFound(err) {
			return nil, ErrClientNotFound{}
		}
		return nil, ErrInternal{}
	}

	topics := []string{}
	for _, topicKey := range topicKeys {
		topics = append(topics, topicKey.Topic)
	}

	logger.WithField("total", len(topics)).Info("succeeded")

	return topics, nil
}

func (s *e4impl) CountClientsForTopic(ctx context.Context, topic string) (int, error) {
	_, span := trace.StartSpan(ctx, "e4.CountClientsForTopic")
	defer span.End()

	logger := s.logger.WithField("topic", topic)

	count, err := s.db.CountClientsForTopic(topic)
	if err != nil {
		logger.WithError(err).Error("failed to count clients for topic")
		if models.IsErrRecordNotFound(err) {
			return 0, ErrTopicNotFound{}
		}
		return 0, ErrInternal{}
	}

	logger.WithField("total", count).Info("succeeded")

	return count, nil
}

func (s *e4impl) GetClientsRangeByTopic(ctx context.Context, topic string, offset, count int) ([]IDNamePair, error) {
	_, span := trace.StartSpan(ctx, "e4.GetClientsRangeByTopic")
	defer span.End()

	logger := s.logger.WithFields(log.Fields{
		"topic":  topic,
		"offset": offset,
		"count":  count,
	})

	clients, err := s.db.GetClientsForTopic(topic, offset, count)
	if err != nil {
		logger.WithError(err).Error("failed to get clients for topic")
		if models.IsErrRecordNotFound(err) {
			return nil, ErrTopicNotFound{}
		}
		return nil, ErrInternal{}
	}

	idNamePairs := make([]IDNamePair, 0, len(clients))
	for _, client := range clients {
		idNamePairs = append(idNamePairs, IDNamePair{ID: client.E4ID, Name: client.Name})
	}

	logger.WithField("total", len(idNamePairs)).Info("succeeded")

	return idNamePairs, nil
}

func (s *e4impl) LinkClient(ctx context.Context, sourceClientID, targetClientID []byte) error {
	_, span := trace.StartSpan(ctx, "e4.LinkClient")
	defer span.End()

	logger := s.logger.WithFields(log.Fields{
		"sourceClientID": prettyID(sourceClientID),
		"targetClientID": prettyID(targetClientID),
	})

	sourceClient, err := s.db.GetClientByID(sourceClientID)
	if err != nil {
		logger.WithError(err).Error("failed to retrieve sourceClient")
		if models.IsErrRecordNotFound(err) {
			return ErrClientNotFound{}
		}
		return ErrInternal{}
	}
	targetClient, err := s.db.GetClientByID(targetClientID)
	if err != nil {
		logger.WithError(err).Error("failed to retrieve targetClient")
		if models.IsErrRecordNotFound(err) {
			return ErrClientNotFound{}
		}
		return ErrInternal{}
	}

	if err := s.db.LinkClient(sourceClient, targetClient); err != nil {
		logger.WithError(err).Error("failed to link clients")
		return ErrInternal{}
	}

	logger.Info("succeeded")

	return nil
}
func (s *e4impl) UnlinkClient(ctx context.Context, sourceClientID, targetClientID []byte) error {
	_, span := trace.StartSpan(ctx, "e4.UnlinkClient")
	defer span.End()

	logger := s.logger.WithFields(log.Fields{
		"sourceClient": prettyID(sourceClientID),
		"targetClient": prettyID(targetClientID),
	})

	sourceClient, err := s.db.GetClientByID(sourceClientID)
	if err != nil {
		logger.WithError(err).Error("failed to retrieve sourceClient")
		if models.IsErrRecordNotFound(err) {
			return ErrClientNotFound{}
		}
		return ErrInternal{}
	}
	targetClient, err := s.db.GetClientByID(targetClientID)
	if err != nil {
		logger.WithError(err).Error("failed to retrieve targetClient")
		if models.IsErrRecordNotFound(err) {
			return ErrClientNotFound{}
		}
		return ErrInternal{}
	}

	if err := s.db.UnlinkClient(sourceClient, targetClient); err != nil {
		logger.WithError(err).Error("failed to unlink clients")
		return ErrInternal{}
	}

	logger.Info("succeeded")

	return nil
}
func (s *e4impl) CountLinkedClients(ctx context.Context, id []byte) (int, error) {
	_, span := trace.StartSpan(ctx, "e4.UnlinkClient")
	defer span.End()

	logger := s.logger.WithFields(log.Fields{
		"client": prettyID(id),
	})

	count, err := s.db.CountLinkedClients(id)
	if err != nil {
		logger.WithError(err).Error("failed to count linked clients")
		if models.IsErrRecordNotFound(err) {
			return 0, ErrClientNotFound{}
		}
		return 0, ErrInternal{}
	}

	logger.WithField("total", count).Info("succeeded")

	return count, nil
}
func (s *e4impl) GetLinkedClients(ctx context.Context, id []byte, offset, count int) ([]IDNamePair, error) {
	_, span := trace.StartSpan(ctx, "e4.GetClientsRangeByTopic")
	defer span.End()

	logger := s.logger.WithFields(log.Fields{
		"client": prettyID(id),
		"offset": offset,
		"count":  count,
	})

	clients, err := s.db.GetLinkedClientsForClientByID(id, offset, count)
	if err != nil {
		logger.WithError(err).Error("failed to get linked clients")
		if models.IsErrRecordNotFound(err) {
			return nil, ErrClientNotFound{}
		}
		return nil, ErrInternal{}
	}

	idNamePairs := make([]IDNamePair, 0, len(clients))
	for _, client := range clients {
		idNamePairs = append(idNamePairs, IDNamePair{ID: client.E4ID, Name: client.Name})
	}

	logger.WithField("total", len(idNamePairs)).Info("succeeded")

	return idNamePairs, nil
}

func (s *e4impl) SendClientPubKey(ctx context.Context, sourceClientID, targetClientID []byte) error {
	ctx, span := trace.StartSpan(ctx, "e4.SendClientPubKey")
	defer span.End()

	logger := s.logger.WithFields(log.Fields{
		"sourceClientID": sourceClientID,
		"targetClientID": targetClientID,
	})

	if !s.e4Key.IsPubKeyMode() {
		logger.WithError(errors.New("e4Key is not a publicKey type")).Error("failed to send public key")
		return ErrInvalidCryptoMode{}
	}

	sourceClient, err := s.db.GetClientByID(sourceClientID)
	if err != nil {
		logger.WithError(err).Error("failed to get source client")
		if models.IsErrRecordNotFound(err) {
			return ErrClientNotFound{}
		}
		return ErrInternal{}
	}

	targetClient, err := s.db.GetClientByID(targetClientID)
	if err != nil {
		logger.WithError(err).Error("failed to get target client")
		if models.IsErrRecordNotFound(err) {
			return ErrClientNotFound{}
		}
		return ErrInternal{}
	}

	clearPubKey, err := sourceClient.DecryptKey(s.dbEncKey)
	if err != nil {
		logger.WithError(err).Error("failed to decrypt source client key")
		return ErrInternal{}
	}

	cmd, err := s.commandFactory.CreateSetPubKeyCommand(clearPubKey, sourceClient.Name)
	if err != nil {
		logger.WithError(err).Error("failed to create SetPubKey command")
		return ErrInternal{}
	}

	if err := s.sendCommandToClient(ctx, cmd, targetClient); err != nil {
		logger.WithError(err).Error("failed to send SetPubKey command to target client")
		return ErrInternal{}
	}

	logger.Info("success sending SetPubKey command")

	return nil
}

func (s *e4impl) RemoveClientPubKey(ctx context.Context, sourceClientID, targetClientID []byte) error {
	ctx, span := trace.StartSpan(ctx, "e4.RemoveClientPubKey")
	defer span.End()

	logger := s.logger.WithFields(log.Fields{
		"sourceClientID": sourceClientID,
		"targetClientID": targetClientID,
	})

	if !s.e4Key.IsPubKeyMode() {
		logger.WithError(errors.New("e4Key is not a publicKey type")).Error("failed to remove public key")
		return ErrInvalidCryptoMode{}
	}

	sourceClient, err := s.db.GetClientByID(sourceClientID)
	if err != nil {
		logger.WithError(err).Error("failed to get source client")
		if models.IsErrRecordNotFound(err) {
			return ErrClientNotFound{}
		}
		return ErrInternal{}
	}

	targetClient, err := s.db.GetClientByID(targetClientID)
	if err != nil {
		logger.WithError(err).Error("failed to get target client")
		if models.IsErrRecordNotFound(err) {
			return ErrClientNotFound{}
		}
		return ErrInternal{}
	}

	cmd, err := s.commandFactory.CreateRemovePubKeyCommand(sourceClient.Name)
	if err != nil {
		logger.WithError(err).Error("failed to create RemovePubKey command")
		return ErrInternal{}
	}

	if err := s.sendCommandToClient(ctx, cmd, targetClient); err != nil {
		logger.WithError(err).Error("failed to send RemovePubKey command to target client")
		return ErrInternal{}
	}

	logger.Info("success sending RemovePubKey command")

	return nil
}

func (s *e4impl) ResetClientPubKeys(ctx context.Context, targetClientID []byte) error {
	ctx, span := trace.StartSpan(ctx, "e4.ResetClientPubKeys")
	defer span.End()

	logger := s.logger.WithFields(log.Fields{
		"targetClientID": targetClientID,
	})

	if !s.e4Key.IsPubKeyMode() {
		logger.WithError(errors.New("e4Key is not a publicKey type")).Error("failed to remove public key")
		return ErrInvalidCryptoMode{}
	}

	targetClient, err := s.db.GetClientByID(targetClientID)
	if err != nil {
		logger.WithError(err).Error("failed to get target client")
		if models.IsErrRecordNotFound(err) {
			return ErrClientNotFound{}
		}
		return ErrInternal{}
	}

	cmd, err := s.commandFactory.CreateResetPubKeysCommand()
	if err != nil {
		logger.WithError(err).Error("failed to create ResetPubKeys command")
		return ErrInternal{}
	}

	if err := s.sendCommandToClient(ctx, cmd, targetClient); err != nil {
		logger.WithError(err).Error("failed to send ResetPubKeys command to target client")
		return ErrInternal{}
	}

	logger.Info("success sending ResetPubKeys command")

	return nil
}

func (s *e4impl) NewC2Key(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "e4.SetC2Key")
	defer span.End()

	logger := s.logger

	if !s.e4Key.IsPubKeyMode() {
		logger.WithError(errors.New("e4Key is not a publicKey type")).Error("failed to remove public key")
		return ErrInvalidCryptoMode{}
	}

	tx, err := s.e4Key.NewC2KeyRotationTx()
	if err != nil {
		logger.WithError(err).Error("failed to backup and rotate new C2 key")
		return ErrInternal{}
	}

	cmd, err := s.commandFactory.CreateSetC2KeyCommand(tx.GetNewPublicKey())
	if err != nil {
		logger.WithError(err).Error("failed to create SetC2Key command")
		if err := tx.Rollback(); err != nil {
			logger.WithError(err).Error("failed to rollback newC2Key transaction")
		}
		return ErrInternal{}
	}

	// If anything fail while dispatching the new key to clients, we just
	// skip and logs either for the full batch on DB error, or for the client on broker error.
	offset := 0
	for {
		logger := logger.WithFields(log.Fields{
			"offset": offset,
			"count":  NewC2KeyBatchSize,
		})

		clients, err := s.db.GetClientsRange(offset, NewC2KeyBatchSize)
		if err != nil {
			logger.WithError(err).Error("failed to fetch client batch from database")
			continue
		}

		for _, client := range clients {
			if err := s.sendCommandToClient(ctx, cmd, client); err != nil {
				logger.WithError(err).WithField("client", client.E4ID).Error("failed to send SetC2Key command to client")
				continue
			}

		}

		if len(clients) < NewC2KeyBatchSize {
			break
		}

		offset += NewC2KeyBatchSize
	}

	if err := tx.Commit(); err != nil {
		logger.WithError(err).Error("failed to commit C2 key transaction")
		if err := tx.Rollback(); err != nil {
			logger.WithError(err).Error("failed to rollback C2 key transaction")
		}
		return ErrInternal{}
	}

	logger.Info("success setting new C2 key")

	return nil
}

func (s *e4impl) ProtectMessage(ctx context.Context, topic string, data []byte) ([]byte, error) {
	_, span := trace.StartSpan(ctx, "e4.ProtectMessage")
	defer span.End()

	logger := s.logger.WithFields(log.Fields{
		"topic":  topic,
		"msgLen": len(data),
	})

	topicKey, err := s.db.GetTopicKey(topic)
	if err != nil {
		logger.WithError(err).Error("failed to get topic")
		if models.IsErrRecordNotFound(err) {
			return nil, ErrTopicNotFound{}
		}
		return nil, ErrInternal{}
	}

	clearTopicKey, err := topicKey.DecryptKey(s.dbEncKey)
	if err != nil {
		logger.WithError(err).Error("failed to decrypt topic key")
		return nil, ErrInternal{}
	}

	protected, err := e4crypto.ProtectSymKey(data, clearTopicKey)
	if err != nil {
		logger.WithError(err).Error("failed to protect message")
		return nil, ErrInternal{}
	}

	logger.Info("success protecting message")

	return protected, nil
}

func (s *e4impl) sendCommandToClient(ctx context.Context, command commands.Command, client models.Client) error {
	ctx, span := trace.StartSpan(ctx, "e4.sendCommandToClient")
	defer span.End()

	clearKey, err := client.DecryptKey(s.dbEncKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt client: %v", err)
	}

	payload, err := s.e4Key.ProtectCommand(command, clearKey)
	if err != nil {
		return fmt.Errorf("failed to protect command: %v", err)
	}

	return s.pubSubClient.Publish(ctx, payload, client, protocols.QoSExactlyOnce)
}

// IsErrRecordNotFound indicate whenever error is a RecordNotFound error
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
			idTest := e4crypto.HashIDAlias(name)
			if !bytes.Equal(idTest, id) {
				return nil, fmt.Errorf("inconsistent Name Alias and E4ID")
			}
			return id, nil
		}
		return e4crypto.HashIDAlias(name), nil
	}

	if len(id) != e4crypto.IDLen {
		return nil, fmt.Errorf("incorrect ID Length, expected %d bytes, got %d", e4crypto.IDLen, len(id))
	}
	return id, nil
}

func prettyID(id []byte) string {
	return base64.StdEncoding.EncodeToString(id)
}
