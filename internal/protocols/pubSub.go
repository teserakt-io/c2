package protocols

import (
	"context"
	"errors"
)

//go:generate mockgen -destination=pubSub_mocks.go -package protocols -self_package github.com/teserakt-io/c2/internal/protocols github.com/teserakt-io/c2/internal/protocols PubSubClient

var (
	// ErrAlreadyConnected is returned when trying to connect an already connected client
	ErrAlreadyConnected = errors.New("already connected")
	// ErrNotConnected is returned when trying to disconnect a not connected client
	ErrNotConnected = errors.New("not connected")
)

// PubSubClient defines a publish / subscribe client interface for the E4 service.
type PubSubClient interface {
	Connect() error
	Disconnect() error
	SubscribeToTopics(ctx context.Context, topics []string) error
	SubscribeToTopic(ctx context.Context, topic string) error
	UnsubscribeFromTopic(ctx context.Context, topic string) error
	Publish(ctx context.Context, payload []byte, topic string, qos byte) error
}
