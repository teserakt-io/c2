package services

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/golang/mock/gomock"

	"gitlab.com/teserakt/c2/internal/commands"
	"gitlab.com/teserakt/c2/internal/models"
	"gitlab.com/teserakt/c2/internal/protocols"
	e4 "gitlab.com/teserakt/e4common"
)

func encryptKey(t *testing.T, keyEncKey []byte, key []byte) []byte {
	protectedkey, err := e4.Encrypt(keyEncKey, nil, key)
	if err != nil {
		t.Fatalf("Failed to encrypt key %v: %v", key, err)
	}

	return protectedkey
}

func decryptKey(t *testing.T, keyEncKey []byte, encKey []byte) []byte {
	key, err := e4.Decrypt(keyEncKey, nil, encKey)
	if err != nil {
		t.Fatalf("Failed to decrypt key %v: %v", encKey, err)
	}

	return key
}

func newKey(t *testing.T) []byte {
	key := make([]byte, e4.KeyLen)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	return key
}

func createTestClient(t *testing.T, e4Key []byte) (models.Client, []byte) {
	clearKey := newKey(t)
	encryptedKey := encryptKey(t, e4Key, clearKey)

	randombytes := make([]byte, e4.IDLen)

	_, err := rand.Read(randombytes)
	if err != nil {
		t.Fatalf("Failed to generate random bytes: %v", err)
	}

	name := hex.EncodeToString(randombytes)
	id := e4.HashIDAlias(name)

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

	logger := log.NewNopLogger()

	e4Key := newKey(t)

	service := NewE4(mockDB, mockPubSubClient, mockCommandFactory, logger, e4Key)

	t.Run("Validation works successfully", func(t *testing.T) {

		names := []string{"test1", "testtest2", "e4test3", "test4", "test5"}

		// test names return the correct hashes:
		for _, name := range names {
			id, err := validateE4NameOrIDPair(name, nil)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if bytes.Equal(id, e4.HashIDAlias(name)) == false {
				t.Errorf("Did not return correctly hashed name")
			}
		}

		for _, name := range names {
			submittedid := e4.HashIDAlias(name)
			id, err := validateE4NameOrIDPair(name, submittedid)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if bytes.Equal(id, submittedid) == false {
				t.Errorf("Did not return correctly hashed name")
			}
		}

		for _, name := range names {
			submittedid := e4.HashIDAlias(name)
			submittedid[0] ^= 0x01
			_, err := validateE4NameOrIDPair(name, submittedid)
			if err == nil {
				t.Errorf("Expected an error, received a non-error result")
			}
			submittedid = e4.HashIDAlias(name)
			shorterid := submittedid[0 : e4.IDLen-2]
			_, err = validateE4NameOrIDPair(name, shorterid)
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

	t.Run("RemoveClientByID deletes the client", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		client, _ := createTestClient(t, e4Key)

		mockDB.EXPECT().DeleteClientByID(client.E4ID)
		if err := service.RemoveClientByID(ctx, client.E4ID); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
	t.Run("RemoveClientByName deletes the client", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		client, _ := createTestClient(t, e4Key)

		mockDB.EXPECT().DeleteClientByID(client.E4ID)
		if err := service.RemoveClientByName(ctx, client.Name); err != nil {
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

		gomock.InOrder(
			mockDB.EXPECT().GetClientByID(client.E4ID).Return(client, nil),
			mockDB.EXPECT().GetTopicKey(topicKey.Topic).Return(topicKey, nil),

			mockCommandFactory.EXPECT().CreateSetTopicKeyCommand(topicKey.Hash(), clearTopicKey).Return(mockCommand, nil),
			mockCommand.EXPECT().Protect(clearClientKey).Return(commandPayload, nil),

			mockPubSubClient.EXPECT().Publish(gomock.Any(), commandPayload, client.Topic(), protocols.QoSExactlyOnce),

			mockDB.EXPECT().LinkClientTopic(client, topicKey),
		)

		if err := service.NewTopicClient(ctx, client.Name, client.E4ID, topicKey.Topic); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("RemoveTopicClientByID unlink client from topic and notify it before updating DB", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		client, clearIDKey := createTestClient(t, e4Key)
		topicKey, _ := createTestTopicKey(t, e4Key)

		mockCommand := commands.NewMockCommand(mockCtrl)
		commandPayload := []byte("command-payload")

		gomock.InOrder(
			mockDB.EXPECT().GetClientByID(client.E4ID).Return(client, nil),
			mockDB.EXPECT().GetTopicKey(topicKey.Topic).Return(topicKey, nil),

			mockCommandFactory.EXPECT().CreateRemoveTopicCommand(topicKey.Hash()).Return(mockCommand, nil),
			mockCommand.EXPECT().Protect(clearIDKey).Return(commandPayload, nil),

			mockPubSubClient.EXPECT().Publish(gomock.Any(), commandPayload, client.Topic(), protocols.QoSExactlyOnce),

			mockDB.EXPECT().UnlinkClientTopic(client, topicKey),
		)

		if err := service.RemoveTopicClientByID(ctx, client.E4ID, topicKey.Topic); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("RemoveTopicClientByName removes the topic - client relation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		client, clearIDKey := createTestClient(t, e4Key)
		topicKey, _ := createTestTopicKey(t, e4Key)

		mockCommand := commands.NewMockCommand(mockCtrl)
		commandPayload := []byte("command-payload")

		gomock.InOrder(
			mockDB.EXPECT().GetClientByID(client.E4ID).Return(client, nil),
			mockDB.EXPECT().GetTopicKey(topicKey.Topic).Return(topicKey, nil),

			mockCommandFactory.EXPECT().CreateRemoveTopicCommand(topicKey.Hash()).Return(mockCommand, nil),
			mockCommand.EXPECT().Protect(clearIDKey).Return(commandPayload, nil),

			mockPubSubClient.EXPECT().Publish(gomock.Any(), commandPayload, client.Topic(), protocols.QoSExactlyOnce),

			mockDB.EXPECT().UnlinkClientTopic(client, topicKey),
		)

		if err := service.RemoveTopicClientByName(ctx, client.Name, topicKey.Topic); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("ResetClientByID send a reset command to client", func(t *testing.T) {
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

		if err := service.ResetClientByID(ctx, client.E4ID); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("ResetClientByName send a reset command to client", func(t *testing.T) {
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

		if err := service.ResetClientByName(ctx, client.Name); err != nil {
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
				if len(newKey) != e4.KeyLen {
					t.Errorf("Expected newKey to be %d bytes, got %d", e4.KeyLen, len(newKey))
				}

				var err error
				protectedNewKey, err = e4.Encrypt(e4Key, nil, newKey)
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

		if err := service.NewClientKey(ctx, client.Name, []byte{}); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("GetAllTopics returns all topics", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		t1, _ := createTestTopicKey(t, e4Key)
		t2, _ := createTestTopicKey(t, e4Key)
		t3, _ := createTestTopicKey(t, e4Key)

		topicKeys := []models.TopicKey{t1, t2, t3}
		expectedTopics := []string{t1.Topic, t2.Topic, t3.Topic}

		mockDB.EXPECT().GetAllTopics().Return(topicKeys, nil)

		topics, err := service.GetAllTopics(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(expectedTopics, topics) == false {
			t.Errorf("Expected topics to be %#v, got %#v", expectedTopics, topics)
		}
	})

	t.Run("GetAllTopics returns an empty slice when no results", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockDB.EXPECT().GetAllTopics().Return(nil, nil)
		topics, err := service.GetAllTopics(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if topics == nil {
			t.Errorf("Expected topics to be an empty slice, got nil")
		}
	})

	t.Run("GetAllTopicsUnsafe returns all topics", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		t1, _ := createTestTopicKey(t, e4Key)
		t2, _ := createTestTopicKey(t, e4Key)
		t3, _ := createTestTopicKey(t, e4Key)

		topicKeys := []models.TopicKey{t1, t2, t3}
		expectedTopics := []string{t1.Topic, t2.Topic, t3.Topic}

		mockDB.EXPECT().GetAllTopicsUnsafe().Return(topicKeys, nil)

		topics, err := service.GetAllTopicsUnsafe(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(expectedTopics, topics) == false {
			t.Errorf("Expected topics to be %#v, got %#v", expectedTopics, topics)
		}
	})

	t.Run("GetAllTopics returns an empty slice when no results", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockDB.EXPECT().GetAllTopicsUnsafe().Return(nil, nil)
		topics, err := service.GetAllTopicsUnsafe(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if topics == nil {
			t.Errorf("Expected topics to be an empty slice, got nil")
		}
	})

	t.Run("SendMessage send the given message on the topic", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		topicKey, clearTopicKey := createTestTopicKey(t, e4Key)

		message := "message"
		expectedPayload, err := e4.Protect([]byte(message), clearTopicKey)
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

	t.Run("GetAllClientsAsHexIDs returns all clients", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		i1, _ := createTestClient(t, e4Key)
		i2, _ := createTestClient(t, e4Key)
		i3, _ := createTestClient(t, e4Key)

		clients := []models.Client{i1, i2, i3}
		expectedIds := []string{
			hex.EncodeToString(i1.E4ID),
			hex.EncodeToString(i2.E4ID),
			hex.EncodeToString(i3.E4ID),
		}

		mockDB.EXPECT().GetAllClients().Return(clients, nil)

		ids, err := service.GetAllClientsAsHexIDs(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(expectedIds, ids) == false {
			t.Errorf("Expected ids to be %#v, got %#v", expectedIds, ids)
		}
	})

	t.Run("GetAllClientsAsHexIDs returns an empty slice when no results", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockDB.EXPECT().GetAllClients().Return(nil, nil)
		clients, err := service.GetAllClientsAsHexIDs(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if clients == nil {
			t.Errorf("Expected topics to be an empty slice, got nil")
		}
	})

	t.Run("CountTopicsForClientByID return topic count", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		client, _ := createTestClient(t, e4Key)

		expectedCount := 10

		mockDB.EXPECT().CountTopicsForClientByID(client.E4ID).Return(expectedCount, nil)

		count, err := service.CountTopicsForClientByID(ctx, client.E4ID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if count != expectedCount {
			t.Errorf("Expected count to be %d, got %d", expectedCount, count)
		}
	})

	t.Run("CountTopicsForClientByName return topic count", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		client, _ := createTestClient(t, e4Key)

		expectedCount := 10

		mockDB.EXPECT().CountTopicsForClientByID(client.E4ID).Return(expectedCount, nil)

		count, err := service.CountTopicsForClientByName(ctx, client.Name)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if count != expectedCount {
			t.Errorf("Expected count to be %d, got %d", expectedCount, count)
		}
	})

	t.Run("GetTopicsForClientByID returns topics for a given ID", func(t *testing.T) {
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

		topics, err := service.GetTopicsForClientByID(ctx, client.E4ID, expectedOffset, expectedCount)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(topics, expectedTopics) == false {
			t.Errorf("Expected topics to be %v, got %v", expectedTopics, topics)
		}
	})

	t.Run("GetTopicsForClientByID returns an empty slice when no results", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockDB.EXPECT().GetTopicsForClientByID(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
		topics, err := service.GetTopicsForClientByID(ctx, e4.HashIDAlias("client"), 1, 2)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if topics == nil {
			t.Errorf("Expected topics to be an empty slice, got nil")
		}
	})

	t.Run("GetTopicsForClientByName returns topics for a given client name", func(t *testing.T) {
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

		topics, err := service.GetTopicsForClientByName(ctx, client.Name, expectedOffset, expectedCount)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(topics, expectedTopics) == false {
			t.Errorf("Expected topics to be %v, got %v", expectedTopics, topics)
		}
	})

	t.Run("CountIDsForTopic returns the IDs count for a given topic", func(t *testing.T) {
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

	t.Run("GetClientsByNameForTopic returns all clients for a given topic", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		topicKey, _ := createTestTopicKey(t, e4Key)
		expectedOffset := 1
		expectedCount := 2

		i1, _ := createTestClient(t, e4Key)
		i2, _ := createTestClient(t, e4Key)
		i3, _ := createTestClient(t, e4Key)

		clients := []models.Client{i1, i2, i3}
		expectedNames := []string{
			i1.Name,
			i2.Name,
			i3.Name,
		}

		mockDB.EXPECT().GetClientsForTopic(topicKey.Topic, expectedOffset, expectedCount).Return(clients, nil)

		names, err := service.GetClientsByNameForTopic(ctx, topicKey.Topic, expectedOffset, expectedCount)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(names, expectedNames) == false {
			t.Errorf("Expected names to be %v, got %v", expectedNames, names)
		}
	})

	t.Run("GetClientsByNameForTopic returns an empty slice when no results", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockDB.EXPECT().GetClientsForTopic(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
		clients, err := service.GetClientsByNameForTopic(ctx, "topic", 1, 2)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if clients == nil {
			t.Errorf("Expected clients to be an empty slice, got nil")
		}
	})

	t.Run("GetClientsByIDForTopic returns client IDs as hex encoded string for a given topic", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		topicKey, _ := createTestTopicKey(t, e4Key)
		expectedOffset := 1
		expectedCount := 2

		i1, _ := createTestClient(t, e4Key)
		i2, _ := createTestClient(t, e4Key)
		i3, _ := createTestClient(t, e4Key)

		clients := []models.Client{i1, i2, i3}
		expectedIds := []string{
			hex.EncodeToString(i1.E4ID),
			hex.EncodeToString(i2.E4ID),
			hex.EncodeToString(i3.E4ID),
		}

		mockDB.EXPECT().GetClientsForTopic(topicKey.Topic, expectedOffset, expectedCount).Return(clients, nil)

		ids, err := service.GetClientsByIDForTopic(ctx, topicKey.Topic, expectedOffset, expectedCount)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(ids, expectedIds) == false {
			t.Errorf("Expected ids to be %v, got %v", expectedIds, ids)
		}
	})

	t.Run("GetClientsByIDForTopic returns an empty slice when no results", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockDB.EXPECT().GetClientsForTopic(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
		clients, err := service.GetClientsByIDForTopic(ctx, "topic", 1, 2)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if clients == nil {
			t.Errorf("Expected clients to be an empty slice, got nil")
		}
	})

	t.Run("GetAllClientsAsNames returns client names", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		i1, _ := createTestClient(t, e4Key)
		i2, _ := createTestClient(t, e4Key)
		i3, _ := createTestClient(t, e4Key)

		clients := []models.Client{i1, i2, i3}
		expectedNames := []string{
			i1.Name,
			i2.Name,
			i3.Name,
		}

		mockDB.EXPECT().GetAllClients().Return(clients, nil)

		names, err := service.GetAllClientsAsNames(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(names, expectedNames) == false {
			t.Errorf("Expected names to be %#v, got %#v", expectedNames, names)
		}
	})

	t.Run("GetClientsAsHexIDsRange returns an empty slice when no results", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockDB.EXPECT().GetAllClients().Return(nil, nil)
		clients, err := service.GetAllClientsAsNames(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if clients == nil {
			t.Errorf("Expected clients to be an empty slice, got nil")
		}
	})

	t.Run("GetClientsAsHexIDsRange returns clients IDs as hex strings from offset and count", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		i1, _ := createTestClient(t, e4Key)
		i2, _ := createTestClient(t, e4Key)
		i3, _ := createTestClient(t, e4Key)

		clients := []models.Client{i1, i2, i3}
		expectedIds := []string{
			hex.EncodeToString(i1.E4ID),
			hex.EncodeToString(i2.E4ID),
			hex.EncodeToString(i3.E4ID),
		}

		offset := 1
		count := 2

		mockDB.EXPECT().GetClientsRange(offset, count).Return(clients, nil)

		ids, err := service.GetClientsAsHexIDsRange(ctx, offset, count)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(ids, expectedIds) == false {
			t.Errorf("Expected ids to be %#v, got %#v", expectedIds, ids)
		}
	})

	t.Run("GetClientsAsHexIDsRange returns an empty slice when no results", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockDB.EXPECT().GetClientsRange(gomock.Any(), gomock.Any()).Return(nil, nil)
		clients, err := service.GetClientsAsHexIDsRange(ctx, 1, 2)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if clients == nil {
			t.Errorf("Expected clients to be an empty slice, got nil")
		}
	})

	t.Run("GetClientsAsNamesRange returns clients names from offset and count", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		i1, _ := createTestClient(t, e4Key)
		i2, _ := createTestClient(t, e4Key)
		i3, _ := createTestClient(t, e4Key)

		clients := []models.Client{i1, i2, i3}
		expectedNames := []string{
			i1.Name,
			i2.Name,
			i3.Name,
		}

		offset := 1
		count := 2

		mockDB.EXPECT().GetClientsRange(offset, count).Return(clients, nil)

		names, err := service.GetClientsAsNamesRange(ctx, offset, count)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(names, expectedNames) == false {
			t.Errorf("Expected names to be %#v, got %#v", expectedNames, names)
		}
	})

	t.Run("GetClientsAsNamesRange returns an empty slice when no results", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockDB.EXPECT().GetClientsRange(gomock.Any(), gomock.Any()).Return(nil, nil)
		clients, err := service.GetClientsAsNamesRange(ctx, 1, 2)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if clients == nil {
			t.Errorf("Expected clients to be an empty slice, got nil")
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
