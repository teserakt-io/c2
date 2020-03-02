package commands

import (
	"crypto/rand"
	reflect "reflect"
	"testing"

	"golang.org/x/crypto/ed25519"

	e4 "github.com/teserakt-io/e4go"
	e4crypto "github.com/teserakt-io/e4go/crypto"
)

func newTopicHash(t *testing.T) []byte {
	hash := make([]byte, e4crypto.HashLen)
	_, err := rand.Read(hash)
	if err != nil {
		t.Fatalf("Failed to generate topic hash: %v", err)
	}

	return hash
}

func newKey(t *testing.T) []byte {
	key := make([]byte, e4crypto.KeyLen)
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

	t.Run("CreateSetPubKeyCommand creates the expected command", func(t *testing.T) {
		expectedPubKey, _, err := ed25519.GenerateKey(nil)
		if err != nil {
			t.Fatalf("failed to generate pubkey: %v", err)
		}

		targetName := "targetClient"
		expectedTargetClientID := e4crypto.HashIDAlias(targetName)

		command, err := factory.CreateSetPubKeyCommand(expectedPubKey, targetName)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		assertCommandContains(t, command, e4.SetPubKey, append(expectedPubKey, expectedTargetClientID...))
	})

	t.Run("CreateSetPubKeyCommand handle errors", func(t *testing.T) {
		edPubKey, _, err := ed25519.GenerateKey(nil)
		if err != nil {
			t.Fatalf("failed to generate pubkey: %v", err)
		}

		testDataset := []struct {
			name       string
			pubKey     []byte
			targetName string
		}{
			{
				name:       "invalid pubkey",
				pubKey:     edPubKey[:len(edPubKey)-1],
				targetName: "valid",
			},
			{
				name:       "invalid target name",
				pubKey:     edPubKey,
				targetName: "",
			},
		}

		for _, testData := range testDataset {
			_, err := factory.CreateSetPubKeyCommand(testData.pubKey, testData.targetName)
			if err == nil {
				t.Errorf("CreateSetPubKeyCommand must fail with %s, got no error", testData.name)
			}
		}
	})

	t.Run("CreateRemovePubKeyCommand creates the expected command", func(t *testing.T) {
		targetName := "targetClient"
		expectedTargetClientID := e4crypto.HashIDAlias(targetName)

		command, err := factory.CreateRemovePubKeyCommand(targetName)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		assertCommandContains(t, command, e4.RemovePubKey, expectedTargetClientID)
	})

	t.Run("CreateRemovePubKeyCommand handle errors", func(t *testing.T) {
		_, err := factory.CreateRemovePubKeyCommand("")
		if err == nil {
			t.Errorf("Expected an error when calling CreateRemovePubKeyCommand with empty name")
		}
	})
}
