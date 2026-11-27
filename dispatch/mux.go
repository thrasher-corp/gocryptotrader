package dispatch

import (
	"context"
	"errors"
	"reflect"
	"uuid"

	"github.com/thrasher-corp/gocryptotrader/common"
)

var (
	errIDNotSet = errors.New("id not set")
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
	if err := common.NilGuard(m); err != nil {
		return Pipe{}, err
	}

	if id == uuid.Nil() {
		return Pipe{}, errIDNotSet
	}

	ch, err := m.d.subscribe(id)
	if err != nil {
		return Pipe{}, err
	}

	return Pipe{c: ch, id: id, m: m}, nil
}

// Unsubscribe returns channel to the pool for the full signature set
func (m *Mux) Unsubscribe(id uuid.UUID, ch chan any) error {
	if err := common.NilGuard(m); err != nil {
		return err
	}
	return m.d.unsubscribe(id, ch)
}

// Publish takes in a persistent memory address and dispatches changes to
// required pipes.
func (m *Mux) Publish(data any, ids ...uuid.UUID) error {
	if err := common.NilGuard(m, data); err != nil {
		return err
	}

	if len(ids) == 0 {
		return errNoIDs
	}
	if m.d.subscriberCount.Load() == 0 {
		return nil
	}

	for i := range ids {
		if err := m.d.publish(ids[i], data); err != nil {
			return err
		}
	}
	return nil
}

// GetID a new unique ID to track routing information in the dispatch system
func (m *Mux) GetID() (uuid.UUID, error) {
	if err := common.NilGuard(m); err != nil {
		return uuid.UUID{}, err
	}
	return m.d.getNewID(uuid.NewV4)
}

// Release returns the channel to the communications pool to be reused
func (p *Pipe) Release() error {
	return p.m.Unsubscribe(p.id, p.c)
}

// Channel returns the Pipe's channel
func (p *Pipe) Channel() <-chan any {
	return p.c
}

// Wait blocks until a value arrives, the Pipe's channel is closed or ctx is done. It returns the value as a T,
// ErrPipeClosed for a closed channel, as the dispatcher closes its pipes when it stops, or the context's error; when
// more than one is ready it may return any of them. A value that cannot be asserted to a T returns an error wrapping
// common.ErrTypeAssertFailure, and a nil Pipe or the zero Pipe, which has no channel, returns an error at once
func (p *Pipe) Wait[T any](ctx context.Context) (T, error) {
	var zero T
	if err := common.NilGuard(p); err != nil {
		return zero, err
	}
	if p.c == nil {
		return zero, errChannelIsNil
	}
	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	case data, ok := <-p.c:
		if !ok {
			return zero, ErrPipeClosed
		}
		v, ok := data.(T)
		if !ok {
			return zero, common.GetTypeAssertError(reflect.TypeFor[T]().String(), data)
		}
		return v, nil
	}
}
