package events

//go:generate mockgen -destination=dispatcher_mocks.go -package events -self_package github.com/teserakt-io/c2/internal/events github.com/teserakt-io/c2/internal/events Dispatcher

import (
	"fmt"
	"sync"

	"github.com/go-kit/kit/log"
)

// Dispatcher defines a component able to dispatch an event to
// all subscribed listeners
type Dispatcher interface {
	AddListener(Listener)
	RemoveListener(Listener) error
	Listeners() []Listener
	Dispatch(Event)
}

type dispatcher struct {
	logger    log.Logger
	listeners []Listener
	lock      sync.RWMutex
}

var _ Dispatcher = (*dispatcher)(nil)

// NewDispatcher returns a new instance of an event dispatcher
func NewDispatcher(logger log.Logger) Dispatcher {
	return &dispatcher{
		logger: logger,
	}
}

// AddListener register the given listener on the dispatcher, making it ready to receive events
func (d *dispatcher) AddListener(lis Listener) {
	d.lock.Lock()
	d.listeners = append(d.listeners, lis)
	d.lock.Unlock()

	d.logger.Log("msg", "registered new listener on event dispatcher", "listener", fmt.Sprintf("%p", lis))
}

// Listeners returns the list of registered listeners on the dispatcher
func (d *dispatcher) Listeners() []Listener {
	return d.listeners
}

// RemoveListener will remove given listener from the dispatcher listeners.
// or return ErrListenerNotFound when the listener is not registered on this
// dispatcher.
func (d *dispatcher) RemoveListener(l Listener) error {
	for i, lis := range d.listeners {
		if lis == l {
			d.lock.Lock()
			d.listeners = append(d.listeners[:i], d.listeners[i+1:]...)
			d.lock.Unlock()

			d.logger.Log("msg", "removed listener from event dispatcher", "listener", fmt.Sprintf("%p", l))
			return nil
		}
	}

	return ErrListenerNotFound
}

// Dispatch will fan out the provided event to every registered listerners
func (d *dispatcher) Dispatch(evt Event) {
	d.lock.RLock()
	for _, lis := range d.listeners {
		lis.Send(evt)
	}
	d.logger.Log("msg", "dispatched event to listeners", "count", len(d.listeners))
	d.lock.RUnlock()
}
