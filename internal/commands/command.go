package commands

import (
	"errors"
)

//go:generate mockgen -destination=command_mocks.go -package commands -self_package github.com/teserakt-io/c2/internal/commands github.com/teserakt-io/c2/internal/commands Command

var (
	// ErrEmptyCommand is returned when trying to access content of an empty command
	ErrEmptyCommand = errors.New("empty command")
)

// Command defines an interface for a protectable Commands
type Command interface {
	Type() (byte, error)
	Content() ([]byte, error)
	Bytes() []byte
}

type e4Command []byte

var _ Command = (e4Command)(nil)

func (c e4Command) Type() (byte, error) {
	if len(c) <= 0 {
		return 0, ErrEmptyCommand
	}

	return c[0], nil
}

func (c e4Command) Content() ([]byte, error) {
	if len(c) <= 0 {
		return nil, ErrEmptyCommand
	}

	return c[1:], nil
}

func (c e4Command) Bytes() []byte {
	return c
}
