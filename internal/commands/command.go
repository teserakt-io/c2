package commands

import (
	"errors"

	e4 "github.com/teserakt-io/e4go"
)

//go:generate mockgen -destination=command_mocks.go -package commands -self_package github.com/teserakt-io/c2/internal/commands github.com/teserakt-io/c2/internal/commands Command

var (
	// ErrEmptyCommand is returned when trying to access content of an empty command
	ErrEmptyCommand = errors.New("empty command")
)

// Command defines an interface for a protectable Commands
type Command interface {
	Type() (e4.Command, error)
	Content() ([]byte, error)
	Bytes() []byte
}

type e4Command []byte

var _ Command = (e4Command)(nil)

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

func (c e4Command) Bytes() []byte {
	return c
}
