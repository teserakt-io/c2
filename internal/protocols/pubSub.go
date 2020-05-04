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

package protocols

import (
	"context"
	"errors"

	"github.com/teserakt-io/c2/internal/models"
)

//go:generate mockgen -copyright_file ../../doc/COPYRIGHT_TEMPLATE.txt -destination=pubSub_mocks.go -package protocols -self_package github.com/teserakt-io/c2/internal/protocols github.com/teserakt-io/c2/internal/protocols PubSubClient

var (
	// ErrAlreadyConnected is returned when trying to connect an already connected client
	ErrAlreadyConnected = errors.New("already connected")
	// ErrNotConnected is returned when trying to disconnect a not connected client
	ErrNotConnected = errors.New("not connected")
	// ErrInvalidTopic is returned when a topic contains invalid characters
	ErrInvalidTopic = errors.New("topic has an invalid format")
)

// PubSubClient defines a publish / subscribe client interface for the E4 service.
type PubSubClient interface {
	Connect() error
	Disconnect() error
	SubscribeToTopics(ctx context.Context, topics []string) error
	SubscribeToTopic(ctx context.Context, topic string) error
	UnsubscribeFromTopic(ctx context.Context, topic string) error
	Publish(ctx context.Context, payload []byte, client models.Client, qos byte) error
	// ValidateTopic allows to check a given topic against a specific implementation
	// and returns an error if the given topic is not acceptable on the current PubSubClient.
	ValidateTopic(topic string) error
}
