package crypto

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/ed25519"

	"github.com/golang/mock/gomock"
	e4crypto "github.com/teserakt-io/e4go/crypto"

	"github.com/teserakt-io/c2/internal/commands"
)

func TestE4PubKey(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	c2PublicCurveKey, c2PrivateCurveKey, err := RandomCurve25519Keys()
	if err != nil {
		t.Fatalf("failed to generate curve key: %v", err)
	}

	tmpKeyDir := filepath.Join(os.TempDir(), "test-e4key")
	if err := os.Mkdir(tmpKeyDir, 0770); err != nil {
		t.Fatalf("failed to create tmp key directory %s: %v", tmpKeyDir, err)
	}
	defer os.RemoveAll(tmpKeyDir)

	keyFile, err := ioutil.TempFile(tmpKeyDir, "")
	if err != nil {
		t.Fatalf("failed to generate tmp file: %v", err)
	}

	n, err := keyFile.Write(c2PrivateCurveKey)
	if err != nil {
		t.Fatalf("failed to write key: %v", err)
	}
	if g, w := len(c2PrivateCurveKey), n; g != w {
		t.Fatalf("invalid key write, got %d bytes, want %d", g, w)
	}
	defer keyFile.Close()

	e4Key, err := NewE4PubKey(keyFile.Name())
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

	t.Run("IsPubKeyMode returns true", func(t *testing.T) {
		if !e4Key.IsPubKeyMode() {
			t.Fatal("IsPubKeyMode with an e4PubKey must return true")
		}
	})

	t.Run("Random keys generate new keys", func(t *testing.T) {
		privKey, pubKey, err := e4Key.RandomKey()
		if err != nil {
			t.Fatalf("Failed to generate random key: %v", err)
		}

		if bytes.Equal(privKey, pubKey) {
			t.Fatal("private and public key must not be equals")
		}

		privKey2, pubKey2, err := e4Key.RandomKey()
		if err != nil {
			t.Fatalf("failed to generate random key: %v", err)
		}

		if bytes.Equal(privKey, privKey2) {
			t.Fatal("successive private keys must not be equals")
		}
		if bytes.Equal(pubKey, pubKey2) {
			t.Fatal("successive public keys must not be equals")
		}
	})

	t.Run("NewC2KeyRotationTx backups the current c2 key, generates a new one and saves it", func(t *testing.T) {
		typedKey := e4Key.(*e4PubKey)

		c2KeyTx, err := e4Key.NewC2KeyRotationTx()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !bytes.Equal(typedKey.c2PrivKey, c2PrivateCurveKey) {
			t.Fatal("c2 private key have been modified")
		}
		if !bytes.Equal(typedKey.c2PubKey, c2PublicCurveKey) {
			t.Fatal("c2 public key have been modified")
		}
		if bytes.Equal(c2KeyTx.GetNewPublicKey(), c2PublicCurveKey) {
			t.Fatal("invalid returned public key")
		}

		currentC2Key, err := ioutil.ReadFile(keyFile.Name())
		if err != nil {
			t.Fatalf("failed to read C2 key: %v", err)
		}
		if !bytes.Equal(currentC2Key, typedKey.c2PrivKey) {
			t.Fatal("c2 key file must still contains the old c2 key")
		}

		expectedBackupFileName := fmt.Sprintf("%s.%s.old", keyFile.Name(), time.Now().Format("20060102150405"))
		oldKey, err := ioutil.ReadFile(expectedBackupFileName)
		if err != nil {
			t.Fatalf("failed to read old key: %v", err)
		}

		if !bytes.Equal(oldKey, c2PrivateCurveKey) {
			t.Fatalf("invalid old C2 key backup: got %v, want %v", oldKey, c2PrivateCurveKey)
		}

		// Rollbacking a C2KeyRotation tx restore the original key
		if err := c2KeyTx.Rollback(); err != nil {
			t.Fatalf("failed to rollback c2KeyTx: %v", err)
		}

		currentC2Key, err = ioutil.ReadFile(keyFile.Name())
		if err != nil {
			t.Fatalf("failed to read C2 key: %v", err)
		}
		if !bytes.Equal(currentC2Key, typedKey.c2PrivKey) {
			t.Fatalf("c2 key file must contains the current c2 key, got %v, want %v", currentC2Key, typedKey.c2PrivKey)
		}

		// Commiting a rollbacked tx has no effect
		if err := c2KeyTx.Commit(); err == nil {
			t.Fatalf("expected commit to return an error")
		}
		if !bytes.Equal(typedKey.c2PrivKey, c2PrivateCurveKey) {
			t.Fatal("c2 private key have been modified")
		}
		if !bytes.Equal(typedKey.c2PubKey, c2PublicCurveKey) {
			t.Fatal("c2 public key have been modified")
		}

		// Recreate a fresh new Tx
		c2KeyTx, err = e4Key.NewC2KeyRotationTx()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if err := c2KeyTx.Commit(); err != nil {
			t.Fatalf("failed to commit tx: %v", err)
		}

		currentC2Key, err = ioutil.ReadFile(keyFile.Name())
		if err != nil {
			t.Fatalf("failed to read C2 key: %v", err)
		}
		if bytes.Equal(currentC2Key, c2PrivateCurveKey) {
			t.Fatal("c2 key file must not contains the old c2 key anymore")
		}
		currentC2PubKey, err := curve25519.X25519(currentC2Key, curve25519.Basepoint)
		if err != nil {
			t.Fatalf("failed to X25519 current C2 key: %v", err)
		}
		if !bytes.Equal(currentC2PubKey, typedKey.c2PubKey) {
			t.Fatalf("invalid C2 public key: got %v, want %v", currentC2PubKey, typedKey.c2PubKey)
		}
		if !bytes.Equal(currentC2PubKey, c2KeyTx.GetNewPublicKey()) {
			t.Fatalf("invalid returned public key: got %v, want %v", c2KeyTx.GetNewPublicKey(), currentC2PubKey)
		}

		// Rerunning the key rotation within the same second must fail due to already existing backup file
		if _, err := e4Key.NewC2KeyRotationTx(); err == nil {
			t.Fatal("an error was expected  when rerunning the C2 key rotation with existing backup file")
		}
		failedRotationC2Key, err := ioutil.ReadFile(keyFile.Name())
		if err != nil {
			t.Fatalf("failed to read C2 key: %v", err)
		}
		if !bytes.Equal(failedRotationC2Key, currentC2Key) {
			t.Fatalf("c2 key must not have been modified by a failed rotation. got %v, want %v", failedRotationC2Key, currentC2Key)
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

	t.Run("IsPubKeyMode returns false", func(t *testing.T) {
		if e4Key.IsPubKeyMode() {
			t.Fatalf("IsPubKeyMode with an e4SymKey must return false")
		}
	})

	t.Run("Random keys generate new keys", func(t *testing.T) {
		clientKey, c2Key, err := e4Key.RandomKey()
		if err != nil {
			t.Fatalf("Failed to generate random key: %v", err)
		}

		if !bytes.Equal(clientKey, c2Key) {
			t.Fatalf("expected both keys to be equals")
		}

		clientKey2, c2Key2, err := e4Key.RandomKey()
		if err != nil {
			t.Fatalf("failed to generate random key: %v", err)
		}

		if bytes.Equal(clientKey, clientKey2) {
			t.Fatalf("successive client keys must not be equals")
		}
		if bytes.Equal(c2Key, c2Key2) {
			t.Fatalf("successive c2 keys must not be equals")
		}
	})
}
