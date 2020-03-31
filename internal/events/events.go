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

package events

//go:generate mockgen -copyright_file ../../doc/COPYRIGHT_TEMPLATE.txt -destination=events_mocks.go -package events -self_package github.com/teserakt-io/c2/internal/events github.com/teserakt-io/c2/internal/events Factory

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
