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

//go:generate mockgen -copyright_file ../../doc/COPYRIGHT_TEMPLATE.txt -destination=listener_mocks.go -package events -self_package github.com/teserakt-io/c2/internal/events github.com/teserakt-io/c2/internal/events Listener

import "errors"

var (
	// EventChanBufferSize is the maximum number of events to be retained on an listener event channel
	EventChanBufferSize = 100
)

// events errors
var (
	ErrListenerNotFound = errors.New("listener not found")
)

// Listener defines a type listening for events
type Listener interface {
	C() <-chan Event
	Close() error
	Send(Event)
}

type listener struct {
	c          chan Event
	dispatcher Dispatcher
}

var _ Listener = (*listener)(nil)

// NewListener creates a new Listener and register it on the dispatcher
// It holds an internal buffered channel for events, of size EventChanBufferSize.
func NewListener(dispatcher Dispatcher) Listener {
	lis := &listener{
		dispatcher: dispatcher,
		c:          make(chan Event, EventChanBufferSize),
	}

	// Safety check that the listener channel is buffered
	if cap(lis.c) == 0 {
		panic("listener channel must be buffered to avoid blocking")
	}

	dispatcher.AddListener(lis)

	return lis
}

// C returns a receive only channel of Events
func (e *listener) C() <-chan Event {
	return e.c
}

// Close removes the listener from its dispatcher
// and will not receive events anymore
func (e *listener) Close() error {
	return e.dispatcher.RemoveListener(e)
}

// Send will try to send the event on the listener channel (1st case)
// if the channel block, (like when its buffer is full, or the client isn't reading fast enough)
// we discard the oldest event from the channel and push the new one at the end (2nd case).
// this ensure we never block when sending event to the listener
func (e *listener) Send(evt Event) {
	select {
	case e.c <- evt:
	default:
		<-e.c
		e.c <- evt
	}
}
