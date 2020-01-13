package crypto

//go:generate mockgen -destination=keys_mocks.go -package crypto -self_package github.com/teserakt-io/c2/internal/crypto github.com/teserakt-io/c2/internal/crypto E4Key

import (
	"fmt"

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
}

type e4PubKey struct {
	c2PrivKey e4crypto.Curve25519PrivateKey
}

var _ E4Key = (*e4PubKey)(nil)

// NewE4PubKey creates a new E4 Public key
func NewE4PubKey(c2PrivKey e4crypto.Curve25519PrivateKey) (E4Key, error) {
	if err := e4crypto.ValidateCurve25519PrivKey(c2PrivKey); err != nil {
		return nil, err
	}

	return &e4PubKey{
		c2PrivKey: c2PrivKey,
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

func (k *e4PubKey) RandomKey() (clientKey, c2StoredKey []byte, err error) {
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, nil, err
	}

	return privKey, pubKey, nil
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
