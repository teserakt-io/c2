package commands

import (
	"crypto/rand"
	reflect "reflect"
	"testing"

	e4 "gitlab.com/teserakt/e4common"
)

func newTopicHash(t *testing.T) []byte {
	hash := make([]byte, e4.HashLen)
	_, err := rand.Read(hash)
	if err != nil {
		t.Fatalf("Failed to generate topic hash: %v", err)
	}

	return hash
}

func newKey(t *testing.T) []byte {
	key := make([]byte, e4.KeyLen)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	return key
}

func assertCommandContains(t *testing.T, command Command, expectedType e4.Command, expectedContent []byte) {
	commandType, err := command.Type()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if commandType != expectedType {
		t.Errorf("Expected command type to be %v, got %v", expectedType, commandType)
	}

	commandContent, err := command.Content()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if reflect.DeepEqual(commandContent, expectedContent) == false {
		t.Errorf("Expected command content to be %v, got %v", expectedContent, commandContent)
	}
}

func TestFactory(t *testing.T) {
	factory := NewFactory()

	t.Run("CreateRemoveTopicCommand returns the expected command", func(t *testing.T) {
		expectedTopicHash := newTopicHash(t)
		command, err := factory.CreateRemoveTopicCommand(expectedTopicHash)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		assertCommandContains(t, command, e4.RemoveTopic, expectedTopicHash)
	})

	t.Run("CreateRemoveTopicCommand return an error with invalid topic", func(t *testing.T) {
		invalidHash := []byte("invalid")
		_, err := factory.CreateRemoveTopicCommand(invalidHash)
		if err == nil {
			t.Errorf("Expected an error, got nil")
		}
	})

	t.Run("CreateResetTopicsCommand returns the expected command", func(t *testing.T) {
		command, err := factory.CreateResetTopicsCommand()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		assertCommandContains(t, command, e4.ResetTopics, []byte{})
	})

	t.Run("CreateSetIDKeyCommand creates expected command", func(t *testing.T) {
		expectedKey := newKey(t)

		command, err := factory.CreateSetIDKeyCommand(expectedKey)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		assertCommandContains(t, command, e4.SetIDKey, expectedKey)
	})

	t.Run("CreateSetIDKeyCommand with invalid key returns an error", func(t *testing.T) {
		invalidKey := []byte("invalid")
		_, err := factory.CreateSetIDKeyCommand(invalidKey)
		if err == nil {
			t.Errorf("Expected an error, got nil")
		}
	})

	t.Run("CreateSetTopicKeyCommand creates the expected command", func(t *testing.T) {
		expectedTopicHash := newTopicHash(t)
		expectedKey := newKey(t)

		command, err := factory.CreateSetTopicKeyCommand(expectedTopicHash, expectedKey)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		assertCommandContains(t, command, e4.SetTopicKey, append(expectedKey, expectedTopicHash...))
	})

	t.Run("CreateSetTopicKeyCommand with invalid topicHash or key returns errors", func(t *testing.T) {
		validKey := newKey(t)
		validTopicHash := newTopicHash(t)

		invalidTopicHash := []byte("invalid-topic")
		invalidKey := []byte("invalid-key")

		testDataSet := [][][]byte{
			[][]byte{validTopicHash, invalidKey},
			[][]byte{invalidTopicHash, validKey},
			[][]byte{invalidTopicHash, invalidKey},
		}

		for _, testData := range testDataSet {
			_, err := factory.CreateSetTopicKeyCommand(testData[0], testData[1])
			if err == nil {
				t.Errorf("Expected an error, got nil")
			}
		}
	})
}
