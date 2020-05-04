// Copyright 2020 Teserakt AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

//go:generate mockgen -copyright_file ../../doc/COPYRIGHT_TEMPLATE.txt  -destination=factory_mocks.go -package commands -self_package github.com/teserakt-io/c2/internal/commands github.com/teserakt-io/c2/internal/commands Factory

import (
	"golang.org/x/crypto/ed25519"

	e4 "github.com/teserakt-io/e4go"
	e4crypto "github.com/teserakt-io/e4go/crypto"
)

// Factory defines interface able to create e4 Commands
type Factory interface {
	CreateRemoveTopicCommand(topic string) (Command, error)
	CreateResetTopicsCommand() (Command, error)
	CreateSetIDKeyCommand(key []byte) (Command, error)
	CreateSetTopicKeyCommand(topic string, key []byte) (Command, error)
	CreateSetPubKeyCommand(publicKey ed25519.PublicKey, clientName string) (Command, error)
	CreateRemovePubKeyCommand(clientName string) (Command, error)
	CreateResetPubKeysCommand() (Command, error)
	CreateSetC2KeyCommand(c2PublicKey e4crypto.Curve25519PublicKey) (Command, error)
}

type factory struct {
}

var _ Factory = (*factory)(nil)

// NewFactory creates a new Command factory
func NewFactory() Factory {
	return &factory{}
}

func (f *factory) CreateRemoveTopicCommand(topic string) (Command, error) {
	cmd, err := e4.CmdRemoveTopic(topic)
	if err != nil {
		return nil, err
	}
	return e4Command(cmd), err
}

func (f *factory) CreateResetTopicsCommand() (Command, error) {
	cmd, err := e4.CmdResetTopics()
	if err != nil {
		return nil, err
	}
	return e4Command(cmd), nil
}

func (f *factory) CreateSetIDKeyCommand(key []byte) (Command, error) {
	cmd, err := e4.CmdSetIDKey(key)
	if err != nil {
		return nil, err
	}
	return e4Command(cmd), nil
}

func (f *factory) CreateSetTopicKeyCommand(topic string, key []byte) (Command, error) {
	cmd, err := e4.CmdSetTopicKey(key, topic)
	if err != nil {
		return nil, err
	}
	return e4Command(cmd), nil
}

func (f *factory) CreateSetPubKeyCommand(publicKey ed25519.PublicKey, clientName string) (Command, error) {
	cmd, err := e4.CmdSetPubKey(publicKey, clientName)
	if err != nil {
		return nil, err
	}
	return e4Command(cmd), nil
}

func (f *factory) CreateRemovePubKeyCommand(clientName string) (Command, error) {
	cmd, err := e4.CmdRemovePubKey(clientName)
	if err != nil {
		return nil, err
	}
	return e4Command(cmd), nil
}

func (f *factory) CreateResetPubKeysCommand() (Command, error) {
	cmd, err := e4.CmdResetPubKeys()
	if err != nil {
		return nil, err
	}
	return e4Command(cmd), nil
}

func (f *factory) CreateSetC2KeyCommand(c2PublicKey e4crypto.Curve25519PublicKey) (Command, error) {
	cmd, err := e4.CmdSetC2Key(c2PublicKey)
	if err != nil {
		return nil, err
	}
	return e4Command(cmd), nil
}
