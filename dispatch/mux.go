package dispatch

import (
	"errors"
	"reflect"

	"github.com/gofrs/uuid"
)

// GetNewMux returns a new multiplexor to track subsystem updates
func GetNewMux() *Mux {
	return &Mux{d: dispatcher}
}

// Subscribe takes in a package defined signature element pointing to an ID set
// and returns the associated pipe
func (m *Mux) Subscribe(id uuid.UUID) (Pipe, error) {
	if id == (uuid.UUID{}) {
		return Pipe{}, errors.New("id not set")
	}

	ch, err := m.d.subscribe(id)
	if err != nil {
		return Pipe{}, err
	}

	return Pipe{C: ch, id: id, m: m}, nil
}

// Unsubscribe returns channel to the pool for the full signature set
func (m *Mux) Unsubscribe(id uuid.UUID, ch chan interface{}) error {
	return m.d.unsubscribe(id, ch)
}

// Publish takes in a persistent memory address and dispatches changes to
// required pipes. Data should be of *type.
func (m *Mux) Publish(ids []uuid.UUID, data interface{}) error {
	if data == nil {
		return errors.New("data payload is nil")
	}

	cpy := reflect.ValueOf(data).Elem().Interface()

	for i := range ids {
		// Create copy to not interfere with stored value
		err := m.d.publish(ids[i], &cpy)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetID gets a lovely new ID
func (m *Mux) GetID() (uuid.UUID, error) {
	return m.d.getNewID()
}

// Release returns the channel to the communications pool to be reused
func (p *Pipe) Release() error {
	return p.m.Unsubscribe(p.id, p.C)
}
