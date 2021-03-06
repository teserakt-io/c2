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

package crypto

//go:generate mockgen -copyright_file ../../doc/COPYRIGHT_TEMPLATE.txt -destination=keys_mocks.go -package crypto -self_package github.com/teserakt-io/c2/internal/crypto github.com/teserakt-io/c2/internal/crypto E4Key

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"golang.org/x/crypto/ed25519"

	"golang.org/x/crypto/curve25519"

	e4crypto "github.com/teserakt-io/e4go/crypto"

	"github.com/teserakt-io/c2/internal/commands"
)

// E4Key defines an interface to protect client commands
type E4Key interface {
	// ProtectCommand encrypt the given command using the key material private key
	// and returns the protected command, or an error
	ProtectCommand(cmd commands.Command, clientKey []byte) ([]byte, error)
	// ValidateKey will return an error if given key does not match the expected key type by the E4Key implementation
	ValidateKey(key []byte) error
	// RandomKey generates a new random key, and returns distinct variables for the key to be sent to the client
	// and the one to be stored.
	// If the key has a public part, the clientKey will contains the private part and the c2StoredKey the public part
	// If the key is a symmetric one, both clientKey and c2StoredKey will be equals.
	RandomKey() (clientKey, c2StoredKey []byte, err error)
	// IsPubKeyMode returns true when the E4Key support pubkey mode, or false otherwise
	IsPubKeyMode() bool
	// NewC2KeyRotationTx creates a transaction to update the E4Key with a new C2 curve25519 key pair.
	// On creation, it will backup the current C2 key file, and generate a new key pair.
	// On commit, it will write the new key in the key file, and activate the new key, that will start being used immediately.
	// On rollback, it will restore the original key file from the backup file, and delete the backup.
	// On error creating the transaction, the current key is not modified.
	// It will fail if the given E4Key is not in pubKey mode.
	NewC2KeyRotationTx() (C2KeyRotationTx, error)
}

type e4PubKey struct {
	c2PrivKey e4crypto.Curve25519PrivateKey
	c2PubKey  e4crypto.Curve25519PublicKey
	keyPath   string
}

var _ E4Key = (*e4PubKey)(nil)

// NewE4PubKey creates a new E4 Public key, reading the private curve25519 key from the given path.
func NewE4PubKey(keyPath string) (E4Key, error) {
	keyFile, err := os.Open(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %v", keyPath, err)
	}
	defer keyFile.Close()

	keyBytes, err := ioutil.ReadAll(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read e4key from %s: %v", keyPath, err)
	}

	if err := e4crypto.ValidateCurve25519PrivKey(keyBytes); err != nil {
		return nil, err
	}

	pubKey, err := curve25519.X25519(keyBytes, curve25519.Basepoint)
	if err != nil {
		return nil, err
	}

	return &e4PubKey{
		c2PrivKey: keyBytes,
		c2PubKey:  pubKey,
		keyPath:   keyPath,
	}, nil
}

func (k *e4PubKey) ProtectCommand(cmd commands.Command, clientKey []byte) ([]byte, error) {
	if err := k.ValidateKey(clientKey); err != nil {
		return nil, fmt.Errorf("invalid ed25519 client public key: %v", err)
	}

	shared, err := curve25519.X25519(k.c2PrivKey, e4crypto.PublicEd25519KeyToCurve25519(clientKey))
	if err != nil {
		return nil, fmt.Errorf("curve25519 X25519 failed: %v", err)
	}

	return e4crypto.ProtectSymKey(cmd.Bytes(), e4crypto.Sha3Sum256(shared))
}

func (k *e4PubKey) ValidateKey(key []byte) error {
	return e4crypto.ValidateEd25519PubKey(key)
}

func (k *e4PubKey) RandomKey() ([]byte, []byte, error) {
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, nil, err
	}

	return privKey, pubKey, nil
}

func (k *e4PubKey) NewC2KeyRotationTx() (C2KeyRotationTx, error) {
	return newC2KeyRotationTx(k)
}

// backupCurrentC2Key writes the current C2 key into a backup file named after the current
// key file, with a <YYYYMMDDHHmmSS>.old suffix appended. The current key file is left untouched.
// An error is returned when the backup file already exists (meaning it can only be invoked once per seconds)
func (k *e4PubKey) backupCurrentC2Key() (string, error) {
	backupPath := fmt.Sprintf("%s.%s.old", k.keyPath, time.Now().Format("20060102150405"))
	backupFile, err := os.OpenFile(backupPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %v", backupPath, err)
	}
	defer backupFile.Close()

	n, err := backupFile.Write(k.c2PrivKey)
	if err != nil {
		return "", fmt.Errorf("failed to write backup key: %v", err)
	}
	if n != len(k.c2PrivKey) {
		return "", fmt.Errorf("invalid write, want %d bytes, got %d", len(k.c2PrivKey), n)
	}

	return backupPath, nil
}

func (k *e4PubKey) overwriteC2PrivateKey(newC2PrivateKey e4crypto.Curve25519PrivateKey) error {
	keyFile, err := os.OpenFile(k.keyPath, os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %v", k.keyPath, err)
	}
	defer keyFile.Close()

	n, err := keyFile.Write(newC2PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to write new C2 key: %v", err)
	}
	if g, w := len(k.c2PrivKey), n; g != w {
		return fmt.Errorf("invalid write, got %d bytes, want %d", g, w)
	}

	return nil
}

func (k *e4PubKey) IsPubKeyMode() bool {
	return true
}

type e4SymKey struct {
}

var _ E4Key = (*e4SymKey)(nil)

// NewE4SymKey creates a new E4Key able to protect message and commands using a symmetric key
func NewE4SymKey() E4Key {
	return &e4SymKey{}
}

func (k *e4SymKey) ProtectCommand(cmd commands.Command, clientKey []byte) ([]byte, error) {
	if err := k.ValidateKey(clientKey); err != nil {
		return nil, err
	}

	return e4crypto.ProtectSymKey(cmd.Bytes(), clientKey)
}

func (k *e4SymKey) ValidateKey(key []byte) error {
	if err := e4crypto.ValidateSymKey(key); err != nil {
		return fmt.Errorf("invalid symmetric key: %v", err)
	}

	return nil
}

func (k *e4SymKey) RandomKey() (clientKey, c2StoredKey []byte, err error) {
	key := e4crypto.RandomKey()
	return key, key, nil
}

func (k *e4SymKey) IsPubKeyMode() bool {
	return false
}

func (k *e4SymKey) NewC2KeyRotationTx() (C2KeyRotationTx, error) {
	return nil, errors.New("not available in symkey mode")
}

// RandomCurve25519Keys creates a new random Curve25519 key pair
func RandomCurve25519Keys() (e4crypto.Curve25519PublicKey, e4crypto.Curve25519PrivateKey, error) {
	privKey := e4crypto.RandomKey()
	pubKey, err := curve25519.X25519(privKey, curve25519.Basepoint)
	if err != nil {
		return nil, nil, err
	}

	return pubKey, privKey, nil
}
