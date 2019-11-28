package services

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	log "github.com/sirupsen/logrus"
	e4crypto "github.com/teserakt-io/e4go/crypto"

	"github.com/teserakt-io/c2/internal/commands"
	"github.com/teserakt-io/c2/internal/events"
	"github.com/teserakt-io/c2/internal/models"
	"github.com/teserakt-io/c2/internal/protocols"
)

func encryptKey(t *testing.T, keyEncKey []byte, key []byte) []byte {
	protectedkey, err := e4crypto.Encrypt(keyEncKey, nil, key)
	if err != nil {
		t.Fatalf("Failed to encrypt key %v: %v", key, err)
	}

	return protectedkey
}

func newKey(t *testing.T) []byte {
	key := make([]byte, e4crypto.KeyLen)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	return key
}

func createTestClient(t *testing.T, e4Key []byte) (models.Client, []byte) {
	clearKey := newKey(t)
	encryptedKey := encryptKey(t, e4Key, clearKey)

	randombytes := make([]byte, e4crypto.IDLen)

	_, err := rand.Read(randombytes)
	if err != nil {
		t.Fatalf("Failed to generate random bytes: %v", err)
	}

	name := hex.EncodeToString(randombytes)
	id := e4crypto.HashIDAlias(name)

	client := models.Client{
		Name: name,
		E4ID: id,
		Key:  encryptedKey,
	}

	return client, clearKey
}

func createTestTopicKey(t *testing.T, e4Key []byte) (models.TopicKey, []byte) {
	clearTopicKey := newKey(t)
	encryptedTopicKey := encryptKey(t, e4Key, clearTopicKey)

	topicKey := models.TopicKey{
		Topic: fmt.Sprintf("topic-%d", rand.Int()),
		Key:   encryptedTopicKey,
	}

	return topicKey, clearTopicKey
}

func TestE4(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockDB := models.NewMockDatabase(mockCtrl)
	mockPubSubClient := protocols.NewMockPubSubClient(mockCtrl)
	mockCommandFactory := commands.NewMockFactory(mockCtrl)
	mockEventFactory := events.NewMockFactory(mockCtrl)
	mockEventDispatcher := events.NewMockDispatcher(mockCtrl)

	logger := log.New()
	logger.SetOutput(ioutil.Discard)

	e4Key := newKey(t)

	service := NewE4(mockDB, mockPubSubClient, mockCommandFactory, mockEventDispatcher, mockEventFactory, logger, e4Key)

	t.Run("Validation works successfully", func(t *testing.T) {
		names := []string{"test1", "testtest2", "e4test3", "test4", "test5"}

		// test names return the correct hashes:
		for _, name := range names {
			id, err := ValidateE4NameOrIDPair(name, nil)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if bytes.Equal(id, e4crypto.HashIDAlias(name)) == false {
				t.Errorf("Did not return correctly hashed name")
			}
		}

		for _, name := range names {
			submittedID := e4crypto.HashIDAlias(name)
			id, err := ValidateE4NameOrIDPair(name, submittedID)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if bytes.Equal(id, submittedID) == false {
				t.Errorf("Did not return correctly hashed name")
			}
		}

		for _, name := range names {
			submittedID := e4crypto.HashIDAlias(name)
			submittedID[0] ^= 0x01
			_, err := ValidateE4NameOrIDPair(name, submittedID)
			if err == nil {
				t.Errorf("Expected an error, received a non-error result")
			}
			submittedID = e4crypto.HashIDAlias(name)
			shorterID := submittedID[0 : e4crypto.IDLen-2]
			_, err = ValidateE4NameOrIDPair(name, shorterID)
			if err == nil {
				t.Errorf("Expected an error, received a non-error result")
			}
		}

	})

	t.Run("NewClient encrypt key and save properly with name only", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		client, clearKey := createTestClient(t, e4Key)

		mockDB.EXPECT().InsertClient(client.Name, client.E4ID, client.Key).Times(2)

		if err := service.NewClient(ctx, client.Name, nil, clearKey); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if err := service.NewClient(ctx, client.Name, client.E4ID, clearKey); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("RemoveClient deletes the client", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		client, _ := createTestClient(t, e4Key)

		mockDB.EXPECT().DeleteClientByID(client.E4ID)
		if err := service.RemoveClient(ctx, client.E4ID); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("NewTopicClient links a client to a topic and notify it before updating DB", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		client, clearClientKey := createTestClient(t, e4Key)
		topicKey, clearTopicKey := createTestTopicKey(t, e4Key)

		mockCommand := commands.NewMockCommand(mockCtrl)
		commandPayload := []byte("command-payload")

		expectedEvt := events.Event{
			Type:      events.ClientSubscribed,
			Source:    client.Name,
			Target:    topicKey.Topic,
			Timestamp: time.Now(),
		}

		gomock.InOrder(
			mockDB.EXPECT().GetClientByID(client.E4ID).Return(client, nil),
			mockDB.EXPECT().GetTopicKey(topicKey.Topic).Return(topicKey, nil),

			mockCommandFactory.EXPECT().CreateSetTopicKeyCommand(topicKey.Hash(), clearTopicKey).Return(mockCommand, nil),
			mockCommand.EXPECT().Protect(clearClientKey).Return(commandPayload, nil),

			mockPubSubClient.EXPECT().Publish(gomock.Any(), commandPayload, client.Topic(), protocols.QoSExactlyOnce),

			mockDB.EXPECT().LinkClientTopic(client, topicKey),

			mockEventFactory.EXPECT().NewClientSubscribedEvent(client.Name, topicKey.Topic).Return(expectedEvt),
			mockEventDispatcher.EXPECT().Dispatch(expectedEvt),
		)

		if err := service.NewTopicClient(ctx, client.E4ID, topicKey.Topic); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("RemoveTopicClient unlink client from topic and notify it before updating DB", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		client, clearIDKey := createTestClient(t, e4Key)
		topicKey, _ := createTestTopicKey(t, e4Key)

		mockCommand := commands.NewMockCommand(mockCtrl)
		commandPayload := []byte("command-payload")

		expectedEvt := events.Event{
			Type:      events.ClientSubscribed,
			Source:    client.Name,
			Target:    topicKey.Topic,
			Timestamp: time.Now(),
		}

		gomock.InOrder(
			mockDB.EXPECT().GetClientByID(client.E4ID).Return(client, nil),
			mockDB.EXPECT().GetTopicKey(topicKey.Topic).Return(topicKey, nil),

			mockCommandFactory.EXPECT().CreateRemoveTopicCommand(topicKey.Hash()).Return(mockCommand, nil),
			mockCommand.EXPECT().Protect(clearIDKey).Return(commandPayload, nil),

			mockPubSubClient.EXPECT().Publish(gomock.Any(), commandPayload, client.Topic(), protocols.QoSExactlyOnce),

			mockDB.EXPECT().UnlinkClientTopic(client, topicKey),

			mockEventFactory.EXPECT().NewClientUnsubscribedEvent(client.Name, topicKey.Topic).Return(expectedEvt),
			mockEventDispatcher.EXPECT().Dispatch(expectedEvt),
		)

		if err := service.RemoveTopicClient(ctx, client.E4ID, topicKey.Topic); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("ResetClient send a reset command to client", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		client, clearIDKey := createTestClient(t, e4Key)

		mockCommand := commands.NewMockCommand(mockCtrl)
		commandPayload := []byte("command-payload")

		gomock.InOrder(
			mockDB.EXPECT().GetClientByID(client.E4ID).Return(client, nil),

			mockCommandFactory.EXPECT().CreateResetTopicsCommand().Return(mockCommand, nil),
			mockCommand.EXPECT().Protect(clearIDKey).Return(commandPayload, nil),

			mockPubSubClient.EXPECT().Publish(gomock.Any(), commandPayload, client.Topic(), protocols.QoSExactlyOnce),
		)

		if err := service.ResetClient(ctx, client.E4ID); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("NewTopic creates a new topic and enable its monitoring", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		topic := "topic"

		gomock.InOrder(
			mockDB.EXPECT().InsertTopicKey(topic, gomock.Any()),
			mockPubSubClient.EXPECT().SubscribeToTopic(gomock.Any(), topic),
		)

		if err := service.NewTopic(ctx, topic); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("RemoveTopic cancel topic monitoring and removes it from DB", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		topic := "topic"

		gomock.InOrder(
			mockPubSubClient.EXPECT().UnsubscribeFromTopic(gomock.Any(), topic),
			mockDB.EXPECT().DeleteTopicKey(topic),
		)

		if err := service.RemoveTopic(ctx, topic); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("NewClientKey generates a new key, send it to the client and update the DB", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		client, clearClientKey := createTestClient(t, e4Key)

		mockCommand := commands.NewMockCommand(mockCtrl)
		commandPayload := []byte("command-payload")

		var protectedNewKey []byte

		gomock.InOrder(
			mockDB.EXPECT().GetClientByID(client.E4ID).Return(client, nil),
			mockCommandFactory.EXPECT().CreateSetIDKeyCommand(gomock.Any()).Do(func(newKey []byte) {
				if len(newKey) != e4crypto.KeyLen {
					t.Errorf("Expected newKey to be %d bytes, got %d", e4crypto.KeyLen, len(newKey))
				}

				var err error
				protectedNewKey, err = e4crypto.Encrypt(e4Key, nil, newKey)
				if err != nil {
					t.Fatalf("Expected no error, got %v", err)
				}

			}).Return(mockCommand, nil),
			mockCommand.EXPECT().Protect(clearClientKey).Return(commandPayload, nil),
			mockPubSubClient.EXPECT().Publish(gomock.Any(), commandPayload, client.Topic(), protocols.QoSExactlyOnce),
			mockDB.EXPECT().InsertClient(client.Name, client.E4ID, gomock.Any()).Do(func(name string, id, key []byte) {
				if reflect.DeepEqual(key, protectedNewKey) == false {
					t.Errorf("Expected protected new key to be %#v, got %#v", protectedNewKey, key)
				}
			}),
		)

		if err := service.NewClientKey(ctx, client.E4ID); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("SendMessage send the given message on the topic", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		topicKey, clearTopicKey := createTestTopicKey(t, e4Key)

		message := "message"
		expectedPayload, err := e4crypto.ProtectSymKey([]byte(message), clearTopicKey)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		gomock.InOrder(
			mockDB.EXPECT().GetTopicKey(topicKey.Topic).Return(topicKey, nil),
			mockPubSubClient.EXPECT().Publish(gomock.Any(), expectedPayload, topicKey.Topic, protocols.QoSAtMostOnce),
		)

		if err := service.SendMessage(ctx, topicKey.Topic, message); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("CountTopicsForClient return topic count", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		client, _ := createTestClient(t, e4Key)

		expectedCount := 10

		mockDB.EXPECT().CountTopicsForClientByID(client.E4ID).Return(expectedCount, nil)

		count, err := service.CountTopicsForClient(ctx, client.E4ID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if count != expectedCount {
			t.Errorf("Expected count to be %d, got %d", expectedCount, count)
		}
	})

	t.Run("GetTopicsForClient returns topics for a given ID", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		client, _ := createTestClient(t, e4Key)
		expectedOffset := 1
		expectedCount := 2

		t1, _ := createTestTopicKey(t, e4Key)
		t2, _ := createTestTopicKey(t, e4Key)
		t3, _ := createTestTopicKey(t, e4Key)

		topicKeys := []models.TopicKey{t1, t2, t3}
		expectedTopics := []string{t1.Topic, t2.Topic, t3.Topic}

		mockDB.EXPECT().GetTopicsForClientByID(client.E4ID, expectedOffset, expectedCount).Return(topicKeys, nil)

		topics, err := service.GetTopicsRangeByClient(ctx, client.E4ID, expectedOffset, expectedCount)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(topics, expectedTopics) == false {
			t.Errorf("Expected topics to be %v, got %v", expectedTopics, topics)
		}
	})

	t.Run("GetTopicsForClient returns an empty slice when no results", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockDB.EXPECT().GetTopicsForClientByID(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
		topics, err := service.GetTopicsRangeByClient(ctx, e4crypto.HashIDAlias("client"), 1, 2)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if topics == nil {
			t.Errorf("Expected topics to be an empty slice, got nil")
		}
	})

	t.Run("CountClientsForTopic returns the IDs count for a given topic", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		topicKey, _ := createTestTopicKey(t, e4Key)

		expectedCount := 10

		mockDB.EXPECT().CountClientsForTopic(topicKey.Topic).Return(expectedCount, nil)

		count, err := service.CountClientsForTopic(ctx, topicKey.Topic)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if count != expectedCount {
			t.Errorf("Expected count to be %d, got %d", expectedCount, count)
		}
	})

	t.Run("GetClientsByTopic returns all clients for a given topic", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		topicKey, _ := createTestTopicKey(t, e4Key)
		expectedOffset := 1
		expectedCount := 2

		i1, _ := createTestClient(t, e4Key)
		i2, _ := createTestClient(t, e4Key)
		i3, _ := createTestClient(t, e4Key)

		clients := []models.Client{i1, i2, i3}
		expectedIDNamePairs := []IDNamePair{
			IDNamePair{Name: i1.Name, ID: i1.E4ID},
			IDNamePair{Name: i2.Name, ID: i2.E4ID},
			IDNamePair{Name: i3.Name, ID: i3.E4ID},
		}

		mockDB.EXPECT().GetClientsForTopic(topicKey.Topic, expectedOffset, expectedCount).Return(clients, nil)

		idNamePairs, err := service.GetClientsRangeByTopic(ctx, topicKey.Topic, expectedOffset, expectedCount)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(idNamePairs, expectedIDNamePairs) == false {
			t.Errorf("Expected idNamePairs to be %v, got %v", expectedIDNamePairs, idNamePairs)
		}
	})

	t.Run("GetClientsByTopic returns an empty slice when no results", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockDB.EXPECT().GetClientsForTopic(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
		idNamePairs, err := service.GetClientsRangeByTopic(ctx, "topic", 1, 2)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if idNamePairs == nil {
			t.Errorf("Expected idNamePairs to be an empty slice, got nil")
		}
	})

	t.Run("GetClientsRange returns client ID and Name pairs rom offset and count", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		i1, _ := createTestClient(t, e4Key)
		i2, _ := createTestClient(t, e4Key)
		i3, _ := createTestClient(t, e4Key)

		clients := []models.Client{i1, i2, i3}
		expectedPäirs := []IDNamePair{
			IDNamePair{Name: i1.Name, ID: i1.E4ID},
			IDNamePair{Name: i2.Name, ID: i2.E4ID},
			IDNamePair{Name: i3.Name, ID: i3.E4ID},
		}

		offset := 1
		count := 2

		mockDB.EXPECT().GetClientsRange(offset, count).Return(clients, nil)

		idNamePairs, err := service.GetClientsRange(ctx, offset, count)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(idNamePairs, expectedPäirs) == false {
			t.Errorf("Expected idNamePairs to be %#v, got %#v", expectedPäirs, idNamePairs)
		}
	})

	t.Run("GetClientsRange returns an empty slice when no results", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockDB.EXPECT().GetClientsRange(gomock.Any(), gomock.Any()).Return(nil, nil)
		idNamePairs, err := service.GetClientsRange(ctx, 1, 2)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if idNamePairs == nil {
			t.Errorf("Expected idNamePairs to be an empty slice, got nil")
		}
	})

	t.Run("GetTopicsRange returns topics from offset and count", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		t1, _ := createTestTopicKey(t, e4Key)
		t2, _ := createTestTopicKey(t, e4Key)
		t3, _ := createTestTopicKey(t, e4Key)

		topicKeys := []models.TopicKey{t1, t2, t3}
		expectedTopics := []string{
			t1.Topic,
			t2.Topic,
			t3.Topic,
		}

		offset := 1
		count := 2

		mockDB.EXPECT().GetTopicsRange(offset, count).Return(topicKeys, nil)

		topics, err := service.GetTopicsRange(ctx, offset, count)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(topics, expectedTopics) == false {
			t.Errorf("Expected topics to be %#v, got %#v", expectedTopics, topics)
		}
	})

	t.Run("GetTopicsRange returns an empty slice when no results", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockDB.EXPECT().GetTopicsRange(gomock.Any(), gomock.Any()).Return(nil, nil)
		topics, err := service.GetTopicsRange(ctx, 1, 2)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if topics == nil {
			t.Errorf("Expected topics to be an empty slice, got nil")
		}
	})

	t.Run("CountClients returns client count", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		expectedCount := 42
		mockDB.EXPECT().CountClients().Return(expectedCount, nil)

		c, err := service.CountClients(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if c != expectedCount {
			t.Errorf("Expected count to be %d, got %d", expectedCount, c)
		}
	})

	t.Run("CountTopics returns client count", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		expectedCount := 42
		mockDB.EXPECT().CountTopicKeys().Return(expectedCount, nil)

		c, err := service.CountTopics(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if c != expectedCount {
			t.Errorf("Expected count to be %d, got %d", expectedCount, c)
		}
	})

}
