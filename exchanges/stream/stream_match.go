package stream

import (
	"errors"
	"sync"
)

var errSignatureCollision = errors.New("signature collision")

// NewMatch returns a new Match
func NewMatch() *Match {
	return &Match{m: make(map[any]chan<- []byte)}
}

// Match is a distributed subtype that handles the matching of requests and
// responses in a timely manner, reducing the need to differentiate between
// connections. Stream systems fan in all incoming payloads to one routine for
// processing.
type Match struct {
	m  map[any]chan<- []byte
	mu sync.Mutex
}

// Incoming matches with request, disregarding the returned payload
func (m *Match) Incoming(signature any) bool {
	return m.IncomingWithData(signature, nil)
}

// IncomingWithData matches with requests and takes in the returned payload, to
// be processed outside of a stream processing routine and returns true if a handler was found
func (m *Match) IncomingWithData(signature any, data []byte) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch, ok := m.m[signature]
	if !ok {
		return false
	}
	ch <- data
	close(ch)
	delete(m.m, signature)
	return true

}

// Set the signature response channel for incoming data
func (m *Match) Set(signature any) (<-chan []byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.m[signature]; ok {
		return nil, errSignatureCollision
	}
	ch := make(chan []byte, 1) // This is buffered so we don't need to wait for receiver.
	m.m[signature] = ch
	return ch, nil
}

// Timeout the signature response channel
func (m *Match) Timeout(signature any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ch, ok := m.m[signature]; ok {
		close(ch)
		delete(m.m, signature)
	}
}
