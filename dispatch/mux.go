package dispatch

import (
	"errors"
	"sync/atomic"

	"github.com/gofrs/uuid"
)

var (
	errMuxIsNil = errors.New("mux is nil")
	errIDNotSet = errors.New("id not set")
	errNoData   = errors.New("data payload is nil")
	errNoIDs    = errors.New("no IDs to publish data to")
)

// GetNewMux returns a new multiplexer to track subsystem updates, if nil
// dispatcher provided it will default to the global Dispatcher.
func GetNewMux(d *Dispatcher) *Mux {
	if d == nil {
		d = dispatcher
	}
	return &Mux{d: d}
}

// Subscribe takes in a package defined signature element pointing to an ID set
// and returns the associated pipe
func (m *Mux) Subscribe(id uuid.UUID) (Pipe, error) {
	if m == nil {
		return Pipe{}, errMuxIsNil
	}

	if id.IsNil() {
		return Pipe{}, errIDNotSet
	}

	ch, err := m.d.subscribe(id)
	if err != nil {
		return Pipe{}, err
	}

	return Pipe{c: ch, id: id, m: m}, nil
}

// Unsubscribe returns channel to the pool for the full signature set
func (m *Mux) Unsubscribe(id uuid.UUID, ch chan interface{}) error {
	if m == nil {
		return errMuxIsNil
	}
	return m.d.unsubscribe(id, ch)
}

// Publish takes in a persistent memory address and dispatches changes to
// required pipes.
func (m *Mux) Publish(data interface{}, ids ...uuid.UUID) error {
	if m == nil {
		return errMuxIsNil
	}

	if data == nil {
		return errNoData
	}

	if len(ids) == 0 {
		return errNoIDs
	}
	if atomic.LoadInt32(&m.d.subscriberCount) == 0 {
		return nil
	}

	for i := range ids {
		err := m.d.publish(ids[i], data)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetID a new unique ID to track routing information in the dispatch system
func (m *Mux) GetID() (uuid.UUID, error) {
	if m == nil {
		return uuid.UUID{}, errMuxIsNil
	}
	return m.d.getNewID(uuid.NewV4)
}

// Release returns the channel to the communications pool to be reused
func (p *Pipe) Release() error {
	return p.m.Unsubscribe(p.id, p.c)
}

// Channel returns the Pipe's channel
func (p *Pipe) Channel() <-chan interface{} {
	return p.c
}
