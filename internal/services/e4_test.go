package services

import (
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

func createTestIDKey(t *testing.T, e4Key []byte) (models.IDKey, []byte) {
	clearIDKey := newKey(t)
	encryptedIDKey := encryptKey(t, e4Key, clearIDKey)

	id := make([]byte, e4.IDLen)
	_, err := rand.Read(id)
	if err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}

	idKey := models.IDKey{
		E4ID: id,
		Key:  encryptedIDKey,
	}

	return idKey, clearIDKey
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

	t.Run("NewClient encrypt key and save properly", func(t *testing.T) {
		idKey, clearKey := createTestIDKey(t, e4Key)

		mockDB.EXPECT().InsertIDKey(idKey.E4ID, idKey.Key)

		if err := service.NewClient(idKey.E4ID, clearKey); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("RemoveClient delete the client", func(t *testing.T) {
		idKey, _ := createTestIDKey(t, e4Key)

		mockDB.EXPECT().DeleteIDKey(idKey.E4ID)

		if err := service.RemoveClient(idKey.E4ID); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("NewTopicClient links a client to a topic and notify it before updating DB", func(t *testing.T) {
		idKey, clearIDKey := createTestIDKey(t, e4Key)
		topicKey, clearTopicKey := createTestTopicKey(t, e4Key)

		mockCommand := commands.NewMockCommand(mockCtrl)
		commandPayload := []byte("command-payload")

		gomock.InOrder(
			mockDB.EXPECT().GetIDKey(idKey.E4ID).Return(idKey, nil),
			mockDB.EXPECT().GetTopicKey(topicKey.Topic).Return(topicKey, nil),

			mockCommandFactory.EXPECT().CreateSetTopicKeyCommand(topicKey.Hash(), clearTopicKey).Return(mockCommand, nil),
			mockCommand.EXPECT().Protect(clearIDKey).Return(commandPayload, nil),

			mockPubSubClient.EXPECT().Publish(commandPayload, idKey.Topic(), protocols.QoSExactlyOnce),

			mockDB.EXPECT().LinkIDTopic(idKey, topicKey),
		)

		if err := service.NewTopicClient(idKey.E4ID, topicKey.Topic); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("RemoveTopicClient unlink client from topic and notify it before updating DB", func(t *testing.T) {
		idKey, clearIDKey := createTestIDKey(t, e4Key)
		topicKey, _ := createTestTopicKey(t, e4Key)

		mockCommand := commands.NewMockCommand(mockCtrl)
		commandPayload := []byte("command-payload")

		gomock.InOrder(
			mockDB.EXPECT().GetIDKey(idKey.E4ID).Return(idKey, nil),
			mockDB.EXPECT().GetTopicKey(topicKey.Topic).Return(topicKey, nil),

			mockCommandFactory.EXPECT().CreateRemoveTopicCommand(topicKey.Hash()).Return(mockCommand, nil),
			mockCommand.EXPECT().Protect(clearIDKey).Return(commandPayload, nil),

			mockPubSubClient.EXPECT().Publish(commandPayload, idKey.Topic(), protocols.QoSExactlyOnce),

			mockDB.EXPECT().UnlinkIDTopic(idKey, topicKey),
		)

		if err := service.RemoveTopicClient(idKey.E4ID, topicKey.Topic); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("ResetClient send a reset command to client", func(t *testing.T) {
		idKey, clearIDKey := createTestIDKey(t, e4Key)

		mockCommand := commands.NewMockCommand(mockCtrl)
		commandPayload := []byte("command-payload")

		gomock.InOrder(
			mockDB.EXPECT().GetIDKey(idKey.E4ID).Return(idKey, nil),

			mockCommandFactory.EXPECT().CreateResetTopicsCommand().Return(mockCommand, nil),
			mockCommand.EXPECT().Protect(clearIDKey).Return(commandPayload, nil),

			mockPubSubClient.EXPECT().Publish(commandPayload, idKey.Topic(), protocols.QoSExactlyOnce),
		)

		if err := service.ResetClient(idKey.E4ID); err != nil {
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

		topics, err := service.GetAllTopicIds()
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
		idKey, clearIDKey := createTestIDKey(t, e4Key)

		mockCommand := commands.NewMockCommand(mockCtrl)
		commandPayload := []byte("command-payload")

		var protectedNewKey []byte

		gomock.InOrder(
			mockDB.EXPECT().GetIDKey(idKey.E4ID).Return(idKey, nil),
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
			mockCommand.EXPECT().Protect(clearIDKey).Return(commandPayload, nil),
			mockPubSubClient.EXPECT().Publish(commandPayload, idKey.Topic(), protocols.QoSExactlyOnce),
			mockDB.EXPECT().InsertIDKey(idKey.E4ID, gomock.Any()).Do(func(id, key []byte) {
				if reflect.DeepEqual(key, protectedNewKey) == false {
					t.Errorf("Expected protected new key to be %#v, got %#v", protectedNewKey, key)
				}
			}),
		)

		if err := service.NewClientKey(idKey.E4ID); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("GetAllClientHexIds returns all clients", func(t *testing.T) {
		i1, _ := createTestIDKey(t, e4Key)
		i2, _ := createTestIDKey(t, e4Key)
		i3, _ := createTestIDKey(t, e4Key)

		idKeys := []models.IDKey{i1, i2, i3}
		expectedIds := []string{
			hex.EncodeToString(i1.E4ID),
			hex.EncodeToString(i2.E4ID),
			hex.EncodeToString(i3.E4ID),
		}

		mockDB.EXPECT().GetAllIDKeys().Return(idKeys, nil)

		ids, err := service.GetAllClientHexIds()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(expectedIds, ids) == false {
			t.Errorf("Expected ids to be %#v, got %#v", expectedIds, ids)
		}
	})

	t.Run("CountTopicsForID return topic count", func(t *testing.T) {
		idKey, _ := createTestIDKey(t, e4Key)

		expectedCount := 10

		mockDB.EXPECT().CountTopicsForID(idKey.E4ID).Return(expectedCount, nil)

		count, err := service.CountTopicsForID(idKey.E4ID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if count != expectedCount {
			t.Errorf("Expected count to be %d, got %d", expectedCount, count)
		}
	})

	t.Run("GetTopicsForID returns topics for a given ID", func(t *testing.T) {
		idKey, _ := createTestIDKey(t, e4Key)
		expectedOffset := 1
		expectedCount := 2

		t1, _ := createTestTopicKey(t, e4Key)
		t2, _ := createTestTopicKey(t, e4Key)
		t3, _ := createTestTopicKey(t, e4Key)

		topicKeys := []models.TopicKey{t1, t2, t3}
		expectedTopics := []string{t1.Topic, t2.Topic, t3.Topic}

		mockDB.EXPECT().GetTopicsForID(idKey.E4ID, expectedOffset, expectedCount).Return(topicKeys, nil)

		topics, err := service.GetTopicsForID(idKey.E4ID, expectedOffset, expectedCount)
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

		mockDB.EXPECT().CountIDsForTopic(topicKey.Topic).Return(expectedCount, nil)

		count, err := service.CountIDsForTopic(topicKey.Topic)
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

		i1, _ := createTestIDKey(t, e4Key)
		i2, _ := createTestIDKey(t, e4Key)
		i3, _ := createTestIDKey(t, e4Key)

		idKeys := []models.IDKey{i1, i2, i3}
		expectedIds := []string{
			hex.EncodeToString(i1.E4ID),
			hex.EncodeToString(i2.E4ID),
			hex.EncodeToString(i3.E4ID),
		}

		mockDB.EXPECT().GetIdsforTopic(topicKey.Topic, expectedOffset, expectedCount).Return(idKeys, nil)

		ids, err := service.GetIdsforTopic(topicKey.Topic, expectedOffset, expectedCount)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if reflect.DeepEqual(ids, expectedIds) == false {
			t.Errorf("Expected ids to be %v, got %v", expectedIds, ids)
		}
	})
}
