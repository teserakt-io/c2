package commands

//go:generate mockgen -destination=factory_mocks.go -package commands -self_package github.com/teserakt-io/c2/internal/commands github.com/teserakt-io/c2/internal/commands Factory

import (
	"fmt"

	e4 "github.com/teserakt-io/e4go"
	e4crypto "github.com/teserakt-io/e4go/crypto"
)

// Factory defines interface able to create e4 Commands
type Factory interface {
	CreateRemoveTopicCommand(topichash []byte) (Command, error)
	CreateResetTopicsCommand() (Command, error)
	CreateSetIDKeyCommand(key []byte) (Command, error)
	CreateSetTopicKeyCommand(topicHash, key []byte) (Command, error)
}

type factory struct {
}

var _ Factory = (*factory)(nil)

// NewFactory creates a new Command factory
func NewFactory() Factory {
	return &factory{}
}

func (f *factory) CreateRemoveTopicCommand(topichash []byte) (Command, error) {
	if err := e4crypto.ValidateTopicHash(topichash); err != nil {
		return nil, fmt.Errorf("invalid topic hash for RemoveTopic: %v", err)
	}

	cmd := e4.RemoveTopic
	return e4Command(append([]byte{cmd.ToByte()}, topichash...)), nil
}

func (f *factory) CreateResetTopicsCommand() (Command, error) {
	cmd := e4.ResetTopics
	return e4Command([]byte{cmd.ToByte()}), nil
}

func (f *factory) CreateSetIDKeyCommand(key []byte) (Command, error) {
	if err := e4crypto.ValidateSymKey(key); err != nil {
		return nil, fmt.Errorf("invalid key for SetIdKey: %v", err)
	}

	cmd := e4.SetIDKey
	return e4Command(append([]byte{cmd.ToByte()}, key...)), nil
}

func (f *factory) CreateSetTopicKeyCommand(topicHash, key []byte) (Command, error) {
	if err := e4crypto.ValidateSymKey(key); err != nil {
		return nil, fmt.Errorf("invalid key for SetTopicKey: %v", err)
	}
	if err := e4crypto.ValidateTopicHash(topicHash); err != nil {
		return nil, fmt.Errorf("invalid topic hash for SetTopicKey: %v", err)
	}

	cmd := e4.SetTopicKey
	return e4Command(append(append([]byte{cmd.ToByte()}, key...), topicHash...)), nil
}
