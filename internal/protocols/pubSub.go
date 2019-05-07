package protocols

import "errors"

//go:generate mockgen -destination=pubSub_mocks.go -package protocols -self_package gitlab.com/teserakt/c2/internal/protocols gitlab.com/teserakt/c2/internal/protocols PubSubClient

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
	SubscribeToTopics(topics []string) error
	SubscribeToTopic(topic string) error
	UnsubscribeFromTopic(topic string) error
	Publish(payload []byte, topic string, qos byte) error
}
