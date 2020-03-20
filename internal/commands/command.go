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

import (
	"errors"
)

//go:generate mockgen -copyright_file ../../doc/COPYRIGHT_TEMPLATE.txt -destination=command_mocks.go -package commands -self_package github.com/teserakt-io/c2/internal/commands github.com/teserakt-io/c2/internal/commands Command

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
