package main

import (
	"errors"

	e4 "teserakt/e4/common"
)

// CreateAndProtectForID creates a protected command for a given ID.
func (s *C2) CreateAndProtectForID(cmd e4.Command, topichash, key, id []byte) ([]byte, error) {

	// call CreateCommand
	command, err := CreateCommand(cmd, topichash, key)
	if err != nil {
		return nil, err
	}

	// get key of the given id
	idkey, err := s.getIDKey(id)
	if err != nil {
		return nil, err
	}

	// protect
	payload, err := e4.Protect(command, idkey)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

// CreateCommand assembles a command's arguments to create an encoded command.
func CreateCommand(cmd e4.Command, topichash, key []byte) ([]byte, error) {
	switch cmd {

	case e4.RemoveTopic:
		if !e4.IsValidTopicHash(topichash) {
			return nil, errors.New("invalid topic hash for RemoveTopic")
		}
		if key != nil {
			return nil, errors.New("unexpected key for RemoveTopic")
		}
		return append([]byte{cmd.ToByte()}, topichash...), nil

	case e4.ResetTopics:
		if topichash != nil || key != nil {
			return nil, errors.New("unexpected argument for ResetTopics")
		}
		return []byte{cmd.ToByte()}, nil

	case e4.SetIDKey:
		if !e4.IsValidKey(key) {
			return nil, errors.New("invalid key for SetIdKey")
		}
		if topichash != nil {
			return nil, errors.New("unexpected topichash for SetIdKey")
		}
		return append([]byte{cmd.ToByte()}, key...), nil

	case e4.SetTopicKey:
		if !e4.IsValidKey(key) {
			return nil, errors.New("invalid key for SetTopicKey")
		}
		if !e4.IsValidTopicHash(topichash) {
			return nil, errors.New("invalid topic hash for SetTopicKey")
		}
		return append(append([]byte{cmd.ToByte()}, key...), topichash...), nil
	}

	return nil, errors.New("invalid command")
}
