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

package commands

import (
	"crypto/rand"
	reflect "reflect"
	"testing"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/ed25519"

	e4 "github.com/teserakt-io/e4go"
	e4crypto "github.com/teserakt-io/e4go/crypto"
)

func newKey(t *testing.T) []byte {
	key := make([]byte, e4crypto.KeyLen)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	return key
}

func assertCommandContains(t *testing.T, command Command, expectedType byte, expectedContent []byte) {
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
		topic := "testTopic"
		command, err := factory.CreateRemoveTopicCommand(topic)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expectedTopicHash := e4crypto.HashTopic(topic)
		assertCommandContains(t, command, e4.RemoveTopic, expectedTopicHash)
	})

	t.Run("CreateRemoveTopicCommand return an error with invalid topic", func(t *testing.T) {
		_, err := factory.CreateRemoveTopicCommand("")
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
		topic := "testTopic"

		expectedKey := newKey(t)
		command, err := factory.CreateSetTopicKeyCommand(topic, expectedKey)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expectedTopicHash := e4crypto.HashTopic(topic)
		assertCommandContains(t, command, e4.SetTopicKey, append(expectedKey, expectedTopicHash...))
	})

	t.Run("CreateSetTopicKeyCommand with invalid topicHash or key returns errors", func(t *testing.T) {
		validKey := newKey(t)
		validTopic := "testTopic"

		invalidTopic := ""
		invalidKey := []byte("invalid-key")

		testDataSet := []struct {
			topic string
			key   []byte
		}{
			{
				topic: validTopic,
				key:   invalidKey,
			},
			{
				topic: invalidTopic,
				key:   validKey,
			},
			{
				topic: invalidTopic,
				key:   invalidKey,
			},
		}

		for _, testData := range testDataSet {
			_, err := factory.CreateSetTopicKeyCommand(testData.topic, testData.key)
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

	t.Run("CreateResetPubKeysCommand creates the expected command", func(t *testing.T) {
		command, err := factory.CreateResetPubKeysCommand()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		assertCommandContains(t, command, e4.ResetPubKeys, []byte{})
	})

	t.Run("CreateSetC2KeyCommand creates the expected command", func(t *testing.T) {
		pubKey, err := curve25519.X25519(e4crypto.RandomKey(), curve25519.Basepoint)
		if err != nil {
			t.Errorf("failed to generate curve key: %v", err)
		}

		command, err := factory.CreateSetC2KeyCommand(pubKey)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		assertCommandContains(t, command, e4.SetC2Key, pubKey)
	})

	t.Run("CreateSetC2KeyCommand handle errors", func(t *testing.T) {
		pubKey, err := curve25519.X25519(e4crypto.RandomKey(), curve25519.Basepoint)
		if err != nil {
			t.Errorf("failed to generate curve key: %v", err)
		}

		_, err = factory.CreateSetC2KeyCommand(pubKey[:len(pubKey)-1])
		if err == nil {
			t.Errorf("Expected an error when calling CreateRemovePubKeyCommand with empty name")
		}
	})
}
