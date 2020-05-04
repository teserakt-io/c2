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

package services

import (
	"fmt"
)

// ErrClientNotFound is a database error when a requested record cannot be found
// A message containing the missing entity type will be returned to the user
type ErrClientNotFound struct {
}

func (e ErrClientNotFound) Error() string {
	return "could not find client record in database"
}

// ErrTopicNotFound is a database error when a requested record cannot be found
// A message containing the missing entity type will be returned to the user
type ErrTopicNotFound struct {
}

func (e ErrTopicNotFound) Error() string {
	return "could not find topic record in database"
}

// ErrInternal describe an internal error
// A generic error message will be returned to the user
type ErrInternal struct {
}

func (e ErrInternal) Error() string {
	return "a internal error occured. check application logs for details"
}

// ErrValidation describes an error when validating service's input parameters
// its Err will get returned to the user
type ErrValidation struct {
	Err error
}

func (e ErrValidation) Error() string {
	return fmt.Sprintf("validation error: %v", e.Err)
}

// ErrInvalidCryptoMode is returned when trying to execute a method retricted by the current mode
type ErrInvalidCryptoMode struct {
	Err error
}

func (e ErrInvalidCryptoMode) Error() string {
	return fmt.Sprintf("invalid crypto mode: %v", e.Err)
}
