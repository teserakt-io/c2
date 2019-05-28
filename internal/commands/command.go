package commands

import (
	"errors"
	e4 "gitlab.com/teserakt/e4common"
)

//go:generate mockgen -destination=command_mocks.go -package commands -self_package gitlab.com/teserakt/c2/internal/commands gitlab.com/teserakt/c2/internal/commands Command

var (
	// ErrEmptyCommand is returned when trying to access content of an empty command
	ErrEmptyCommand = errors.New("empty command")
)

// Command defines an interface for a protectable Commands
type Command interface {
	Protect(key []byte) ([]byte, error)
	Type() (e4.Command, error)
	Content() ([]byte, error)
}

type e4Command []byte

var _ Command = e4Command{}

func (c e4Command) Type() (e4.Command, error) {
	if len(c) <= 0 {
		return 0, ErrEmptyCommand
	}

	return e4.Command(c[0]), nil
}

func (c e4Command) Content() ([]byte, error) {
	if len(c) <= 0 {
		return nil, ErrEmptyCommand
	}

	return c[1:], nil
}

// Protect returns an encrypoted command payload with the given key
func (c e4Command) Protect(key []byte) ([]byte, error) {
	return e4.Protect(c, key)
}
