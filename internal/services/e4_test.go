package services

import (
	"fmt"
	"math/rand"
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

func newKey() []byte {
	key := make([]byte, e4.KeyLen)
	rand.Read(key)

	return key
}

func createTestIDKey(t *testing.T, e4Key []byte) (models.IDKey, []byte) {
	clearIDKey := newKey()
	encryptedIDKey := encryptKey(t, e4Key, clearIDKey)

	id := make([]byte, e4.IDLen)
	rand.Read(id)

	idKey := models.IDKey{
		E4ID: id,
		Key:  encryptedIDKey,
	}

	return idKey, clearIDKey
}

func createTestTopicKey(t *testing.T, e4Key []byte) (models.TopicKey, []byte) {
	clearTopicKey := newKey()
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
	mockMQTTClient := protocols.NewMockMQTTClient(mockCtrl)
	mockCommandFactory := commands.NewMockFactory(mockCtrl)

	logger := log.NewNopLogger()

	e4Key := newKey()

	service := NewE4(mockDB, mockMQTTClient, mockCommandFactory, logger, e4Key)

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

			mockMQTTClient.EXPECT().Publish(commandPayload, idKey.Topic(), protocols.QoSExactlyOnce),

			mockDB.EXPECT().LinkIDTopic(idKey, topicKey),
		)

		if err := service.NewTopicClient(idKey.E4ID, topicKey.Topic); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}
