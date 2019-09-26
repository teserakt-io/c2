package events

//go:generate mockgen -destination=events_mocks.go -package events -self_package github.com/teserakt-io/c2/internal/events github.com/teserakt-io/c2/internal/events Factory

import "time"

// EventType defines a custom type for describing events
type EventType int

// List of event types
const (
	Undefined EventType = iota
	ClientSubscribed
	ClientUnsubscribed
)

// Event defines an interface for a generic
// system event
type Event struct {
	Type      EventType
	Source    string
	Target    string
	Timestamp time.Time
}

// Factory allows to create events
type Factory interface {
	NewClientSubscribedEvent(source string, target string) Event
	NewClientUnsubscribedEvent(source string, target string) Event
}

type factory struct {
}

var _ Factory = (*factory)(nil)

// NewFactory creates a new event factory
func NewFactory() Factory {
	return &factory{}
}

func (f *factory) NewClientSubscribedEvent(source string, target string) Event {
	return Event{
		Type:      ClientSubscribed,
		Source:    source,
		Target:    target,
		Timestamp: time.Now(),
	}
}

func (f *factory) NewClientUnsubscribedEvent(source string, target string) Event {
	return Event{
		Type:      ClientUnsubscribed,
		Source:    source,
		Target:    target,
		Timestamp: time.Now(),
	}
}
