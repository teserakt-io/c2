package commands

//go:generate mockgen -destination=factory_mocks.go -package commands -self_package gitlab.com/teserakt/c2/internal/commands gitlab.com/teserakt/c2/internal/commands Factory

import (
	"fmt"

	e4 "gitlab.com/teserakt/e4common"
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
	if err := e4.IsValidTopicHash(topichash); err != nil {
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
	if err := e4.IsValidKey(key); err != nil {
		return nil, fmt.Errorf("invalid key for SetIdKey: %v", err)
	}

	cmd := e4.SetIDKey
	return e4Command(append([]byte{cmd.ToByte()}, key...)), nil
}

func (f *factory) CreateSetTopicKeyCommand(topicHash, key []byte) (Command, error) {
	if err := e4.IsValidKey(key); err != nil {
		return nil, fmt.Errorf("invalid key for SetTopicKey: %v", err)
	}
	if err := e4.IsValidTopicHash(topicHash); err != nil {
		return nil, fmt.Errorf("invalid topic hash for SetTopicKey: %v", err)
	}

	cmd := e4.SetTopicKey
	return e4Command(append(append([]byte{cmd.ToByte()}, key...), topicHash...)), nil
}
