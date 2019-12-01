package keys

import (
	"github.com/teserakt-io/c2/internal/commands"
	"golang.org/x/crypto/ed25519"
)

type e4PubKey struct {
	c2PrivKey ed25519.PrivateKey
}

var _ E4Key = (*e4PubKey)(nil)

func NewE4PubKey(c2PrivKey ed25519.PrivateKey) E4Key {
	return &e4PubKey{
		c2PrivKey: c2PrivKey,
	}
}

func (k *e4PubKey) ProtectMessage(payload []byte, topicKey []byte) ([]byte, error) {
	return nil, nil
}

func (k *e4PubKey) UnprotectMessage(protected []byte, topicKey []byte) ([]byte, error) {
	return nil, nil
}

func (k *e4PubKey) ProtectCommand(cmd commands.Command, clientKey []byte) ([]byte, error) {
	return nil, nil
}

func (k *e4PubKey) UnprotectCommand(protected []byte) (commands.Command, error) {
	return nil, nil
}
