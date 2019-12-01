package keys

import "github.com/teserakt-io/c2/internal/commands"

type e4SymKey struct {
}

var _ E4Key = (*e4SymKey)(nil)

func NewE4SymKey() E4Key {
	return &e4SymKey{}
}

func (k *e4SymKey) ProtectMessage(payload []byte, topicKey []byte) ([]byte, error) {
	return nil, nil
}

func (k *e4SymKey) UnprotectMessage(protected []byte, topicKey []byte) ([]byte, error) {
	return nil, nil
}

func (k *e4SymKey) ProtectCommand(cmd commands.Command, clientKey []byte) ([]byte, error) {
	return nil, nil
}

func (k *e4SymKey) UnprotectCommand(protected []byte) (commands.Command, error) {
	return nil, nil
}
