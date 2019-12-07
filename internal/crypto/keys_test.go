package crypto

import (
	"bytes"
	"testing"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/ed25519"

	"github.com/golang/mock/gomock"
	e4crypto "github.com/teserakt-io/e4go/crypto"

	"github.com/teserakt-io/c2/internal/commands"
)

func TestE4PubKey(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	c2PublicCurveKey, c2PrivateCurveKey, err := e4crypto.RandomCurve25519Keys()
	if err != nil {
		t.Fatalf("failed to generate curve key: %v", err)
	}

	e4Key, err := NewE4PubKey(c2PrivateCurveKey)
	if err != nil {
		t.Fatalf("failed to create e4 pub key: %v", err)
	}

	commandBytes := []byte("command to protect")
	mockCommand := commands.NewMockCommand(mockCtrl)
	mockCommand.EXPECT().Bytes().AnyTimes().Return(commandBytes)

	t.Run("ProtectCommand returns a protected command", func(t *testing.T) {
		clientPubKey, clientPrivateKey, err := ed25519.GenerateKey(nil)
		if err != nil {
			t.Fatalf("failed to generate ed25519 key: %v", err)
		}

		protectedCommand, err := e4Key.ProtectCommand(mockCommand, clientPubKey)
		if err != nil {
			t.Fatalf("failed to protect command: %v", err)
		}

		clientCurvePrivateKey := e4crypto.PrivateEd25519KeyToCurve25519(clientPrivateKey)
		shared, err := curve25519.X25519(clientCurvePrivateKey, c2PublicCurveKey)
		if err != nil {
			t.Fatalf("curve25519 X25519 failed: %v", err)
		}
		key := e4crypto.Sha3Sum256(shared[:])[:e4crypto.KeyLen]

		unprotectedCommand, err := e4crypto.UnprotectSymKey(protectedCommand, key)
		if err != nil {
			t.Fatalf("failed to unprotect command: %v", err)
		}

		if !bytes.Equal(unprotectedCommand, commandBytes) {
			t.Fatalf("invalid unprotected command, got %v, want %v", unprotectedCommand, commandBytes)
		}
	})

	t.Run("ValidateKey returns errors with invalid keys", func(t *testing.T) {
		invalidKeys := [][]byte{
			[]byte{},
			[]byte{0, 1, 2, 3, 4},
			bytes.Repeat([]byte{0}, ed25519.PublicKeySize),
		}

		for _, invalidKey := range invalidKeys {
			if err := e4Key.ValidateKey(invalidKey); err == nil {
				t.Fatalf("expected key %v to be invalid", invalidKey)
			}
		}
	})

	t.Run("ValidateKey returns no errors with valid keys", func(t *testing.T) {
		pubKey, _, err := ed25519.GenerateKey(nil)
		if err != nil {
			t.Fatalf("failed to generate ed25519 key: %v", err)
		}

		if err := e4Key.ValidateKey(pubKey); err != nil {
			t.Fatalf("got error: %v, expected key to be valid", err)
		}
	})
}

func TestE4SymKey(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	e4Key := NewE4SymKey()

	commandBytes := []byte("command to protect")
	mockCommand := commands.NewMockCommand(mockCtrl)
	mockCommand.EXPECT().Bytes().AnyTimes().Return(commandBytes)

	t.Run("ProtectCommand returns a protected command", func(t *testing.T) {
		clientKey := e4crypto.RandomKey()
		protectedCommand, err := e4Key.ProtectCommand(mockCommand, clientKey)
		if err != nil {
			t.Fatalf("failed to protect command: %v", err)
		}

		unprotectedCommand, err := e4crypto.UnprotectSymKey(protectedCommand, clientKey)
		if err != nil {
			t.Fatalf("failed to unprotect command")
		}

		if !bytes.Equal(unprotectedCommand, commandBytes) {
			t.Fatalf("invalid unprotected command, got %v, want %v", unprotectedCommand, commandBytes)
		}
	})

	t.Run("ValidateKey returns errors with invalid keys", func(t *testing.T) {
		invalidKeys := [][]byte{
			[]byte{},
			[]byte{0, 1, 2, 3, 4},
			bytes.Repeat([]byte{0}, e4crypto.KeyLen),
		}

		for _, invalidKey := range invalidKeys {
			if err := e4Key.ValidateKey(invalidKey); err == nil {
				t.Fatalf("expected key %v to be invalid", invalidKey)
			}
		}
	})

	t.Run("ValidateKey returns no errors with valid keys", func(t *testing.T) {
		key := e4crypto.RandomKey()
		if err := e4Key.ValidateKey(key); err != nil {
			t.Fatalf("got error: %v, expected key to be valid", err)
		}
	})
}
