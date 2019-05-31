package services

import (
	"bytes"
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

	t.Run("NewClient encrypt key and save properly", func(t *testing.T) {
		client, clearKey := createTestClient(t, e4Key)

		mockDB.EXPECT().InsertClient(client.Name, client.E4ID, client.Key)

		if err := service.NewClient(client.Name, client.E4ID, clearKey); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("RemoveClientByID deletes the client", func(t *testing.T) {
		client, _ := createTestClient(t, e4Key)

		mockDB.EXPECT().DeleteClientByID(client.E4ID)
		if err := service.RemoveClientByID(client.E4ID); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
	t.Run("RemoveClientByName deletes the client", func(t *testing.T) {
		client, _ := createTestClient(t, e4Key)

		mockDB.EXPECT().DeleteClientByID(client.E4ID)
		if err := service.RemoveClientByName(client.Name); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("NewTopicClient links a client to a topic and notify it before updating DB", func(t *testing.T) {
		client, clearClientKey := createTestClient(t, e4Key)
		topicKey, clearTopicKey := createTestTopicKey(t, e4Key)

		mockCommand := commands.NewMockCommand(mockCtrl)
		commandPayload := []byte("command-payload")

		gomock.InOrder(
			mockDB.EXPECT().GetClientByName(client.Name).Return(client, nil),
			mockDB.EXPECT().GetTopicKey(topicKey.Topic).Return(topicKey, nil),

			mockCommandFactory.EXPECT().CreateSetTopicKeyCommand(topicKey.Hash(), clearTopicKey).Return(mockCommand, nil),
			mockCommand.EXPECT().Protect(clearClientKey).Return(commandPayload, nil),

			mockPubSubClient.EXPECT().Publish(commandPayload, client.Topic(), protocols.QoSExactlyOnce),

			mockDB.EXPECT().LinkClientTopic(client, topicKey),
		)

		if err := service.NewTopicClient(client.Name, client.E4ID, topicKey.Topic); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("RemoveTopicClient unlink client from topic and notify it before updating DB", func(t *testing.T) {
		client, clearIDKey := createTestClient(t, e4Key)
		topicKey, _ := createTestTopicKey(t, e4Key)

		mockCommand := commands.NewMockCommand(mockCtrl)
		commandPayload := []byte("command-payload")

		gomock.InOrder(
			mockDB.EXPECT().GetClientByID(client.E4ID).Return(client, nil),
			mockDB.EXPECT().GetTopicKey(topicKey.Topic).Return(topicKey, nil),

			mockCommandFactory.EXPECT().CreateRemoveTopicCommand(topicKey.Hash()).Return(mockCommand, nil),
			mockCommand.EXPECT().Protect(clearIDKey).Return(commandPayload, nil),

			mockPubSubClient.EXPECT().Publish(commandPayload, client.Topic(), protocols.QoSExactlyOnce),

			mockDB.EXPECT().UnlinkClientTopic(client, topicKey),
		)

		if err := service.RemoveTopicClientByID(client.E4ID, topicKey.Topic); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("ResetClient send a reset command to client", func(t *testing.T) {
		client, clearIDKey := createTestClient(t, e4Key)

		mockCommand := commands.NewMockCommand(mockCtrl)
		commandPayload := []byte("command-payload")

		gomock.InOrder(
			mockDB.EXPECT().GetClientByID(client.E4ID).Return(client, nil),

			mockCommandFactory.EXPECT().CreateResetTopicsCommand().Return(mockCommand, nil),
			mockCommand.EXPECT().Protect(clearIDKey).Return(commandPayload, nil),

			mockPubSubClient.EXPECT().Publish(commandPayload, client.Topic(), protocols.QoSExactlyOnce),
		)

		if err := service.ResetClientByID(client.E4ID); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("NewTopic creates a new topic and enable its monitoring", func(t *testing.T) {
		topic := "topic"

		gomock.InOrder(
			mockDB.EXPECT().InsertTopicKey(topic, gomock.Any()),
			mockPubSubClient.EXPECT().SubscribeToTopic(topic),
		)

		if err := service.NewTopic(topic); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("RemoveTopic cancel topic monitoring and removes it from DB", func(t *testing.T) {

		topic := "topic"

		gomock.InOrder(
			mockPubSubClient.EXPECT().UnsubscribeFromTopic(topic),
			mockDB.EXPECT().DeleteTopicKey(topic),
		)

		if err := service.RemoveTopic(topic); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("GetAllTopicIds returns all topics", func(t *testing.T) {
		t1, _ := createTestTopicKey(t, e4Key)
		t2, _ := createTestTopicKey(t, e4Key)
		t3, _ := createTestTopicKey(t, e4Key)

		topicKeys := []models.TopicKey{t1, t2, t3}
		expectedTopics := []string{t1.Topic, t2.Topic, t3.Topic}

		mockDB.EXPECT().GetAllTopics().Return(topicKeys, nil)

		topics, err := service.GetAllTopics()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(expectedTopics, topics) == false {
			t.Errorf("Expected topics to be %#v, got %#v", expectedTopics, topics)
		}
	})

	t.Run("SendMessage send the given message on the topic", func(t *testing.T) {
		topicKey, clearTopicKey := createTestTopicKey(t, e4Key)

		message := "message"
		expectedPayload, err := e4.Protect([]byte(message), clearTopicKey)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		gomock.InOrder(
			mockDB.EXPECT().GetTopicKey(topicKey.Topic).Return(topicKey, nil),
			mockPubSubClient.EXPECT().Publish(expectedPayload, topicKey.Topic, protocols.QoSAtMostOnce),
		)

		if err := service.SendMessage(topicKey.Topic, message); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("NewClientKey generates a new key, send it to the client and update the DB", func(t *testing.T) {
	})

	t.Run("GetAllClientHexIds returns all clients", func(t *testing.T) {
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

		ids, err := service.GetAllClientsAsHexIDs()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(expectedIds, ids) == false {
			t.Errorf("Expected ids to be %#v, got %#v", expectedIds, ids)
		}
	})

	t.Run("CountTopicsForID return topic count", func(t *testing.T) {
		client, _ := createTestClient(t, e4Key)

		expectedCount := 10

		mockDB.EXPECT().CountTopicsForClientByID(client.E4ID).Return(expectedCount, nil)

		count, err := service.CountTopicsForClientByID(client.E4ID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if count != expectedCount {
			t.Errorf("Expected count to be %d, got %d", expectedCount, count)
		}
	})

	t.Run("GetTopicsForID returns topics for a given ID", func(t *testing.T) {
		client, _ := createTestClient(t, e4Key)
		expectedOffset := 1
		expectedCount := 2

		t1, _ := createTestTopicKey(t, e4Key)
		t2, _ := createTestTopicKey(t, e4Key)
		t3, _ := createTestTopicKey(t, e4Key)

		topicKeys := []models.TopicKey{t1, t2, t3}
		expectedTopics := []string{t1.Topic, t2.Topic, t3.Topic}

		mockDB.EXPECT().GetTopicsForClientByID(client.E4ID, expectedOffset, expectedCount).Return(topicKeys, nil)

		topics, err := service.GetTopicsForClientByID(client.E4ID, expectedOffset, expectedCount)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(topics, expectedTopics) == false {
			t.Errorf("Expected topics to be %v, got %v", expectedTopics, topics)
		}
	})

	t.Run("CountIDsForTopic returns the IDs count for a given topic", func(t *testing.T) {
		topicKey, _ := createTestTopicKey(t, e4Key)

		expectedCount := 10

		mockDB.EXPECT().CountClientsForTopic(topicKey.Topic).Return(expectedCount, nil)

		count, err := service.CountClientsForTopic(topicKey.Topic)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if count != expectedCount {
			t.Errorf("Expected count to be %d, got %d", expectedCount, count)
		}
	})

	t.Run("GetIdsforTopic returns all clients for a given topic", func(t *testing.T) {
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

		ids, err := service.GetClientsByNameForTopic(topicKey.Topic, expectedOffset, expectedCount)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(ids, expectedNames) == false {
			t.Errorf("Expected ids to be %v, got %v", expectedNames, ids)
		}
	})
}
