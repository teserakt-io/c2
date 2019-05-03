package commands

import (
	e4 "gitlab.com/teserakt/e4common"
)

//go:generate mockgen -destination=command_mocks.go -package commands -self_package gitlab.com/teserakt/c2/internal/commands gitlab.com/teserakt/c2/internal/commands Command

// Command defines an interface for a protectable Commands
type Command interface {
	Protect(key []byte) ([]byte, error)
}

type e4Command []byte

var _ Command = e4Command{}

// Protect returns an encrypoted command payload with the given key
func (c e4Command) Protect(key []byte) ([]byte, error) {
	payload, err := e4.Protect(c, key)
	if err != nil {
		return nil, err
	}

	return payload, nil
}
