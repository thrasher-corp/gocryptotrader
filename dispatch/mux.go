package dispatch

import (
	"errors"
	"reflect"
	"sync"

	"github.com/gofrs/uuid"
)

// Mux defines a new multiplexor for the dispatch system, these a generated
// per subsystem
type Mux struct {
	// Reference to the main running dispatch service
	c *Communications
	sync.RWMutex
}

// GetNewMux returns a new multiplexor to track subsystem updates
func GetNewMux() *Mux {
	if comms == nil {
		panic("communications not initialised while getting new mux, not ideal")
	}
	return &Mux{c: comms}
}

// Subscribe takes in a package defined signature element pointing to an ID set
// and returns the associated pipe
func (m *Mux) Subscribe(id uuid.UUID) (Pipe, error) {
	if id == (uuid.UUID{}) {
		return Pipe{}, errors.New("id not set")
	}

	ch, err := m.c.subscribe(id)
	if err != nil {
		return Pipe{}, err
	}

	return Pipe{C: ch, id: id, m: m}, nil
}

// Unsubscribe returns channel to the pool for the full signature set
func (m *Mux) Unsubscribe(id uuid.UUID, ch chan interface{}) error {
	return m.c.unsubscribe(id, ch)
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
		err := m.c.publish(ids[i], &cpy)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetID gets a lovely new ID
func (m *Mux) GetID() (uuid.UUID, error) {
	return m.c.getNewID()
}

// Pipe defines an outbound object to the desired routine
type Pipe struct {
	// Channel to get all our lovely informations
	C chan interface{}
	// ID to tracked system
	id uuid.UUID
	// Reference to multiplexor
	m *Mux
}

// Release returns the channel to the communications pool to be reused
func (p *Pipe) Release() error {
	return p.m.Unsubscribe(p.id, p.C)
}
