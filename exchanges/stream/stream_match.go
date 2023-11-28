package stream

import (
	"errors"
	"sync"
)

// NewMatch returns a new Match
func NewMatch() *Match {
	return &Match{
		m: make(map[interface{}]chan []byte),
	}
}

// Match is a distributed subtype that handles the matching of requests and
// responses in a timely manner, reducing the need to differentiate between
// connections. Stream systems fan in all incoming payloads to one routine for
// processing.
type Match struct {
	m  map[interface{}]chan []byte
	mu sync.Mutex
}

// Matcher defines a payload matching return mechanism
type Matcher struct {
	C   chan []byte
	sig interface{}
	m   *Match
}

// Incoming matches with request, disregarding the returned payload
func (m *Match) Incoming(signature interface{}) bool {
	return m.IncomingWithData(signature, nil)
}

// IncomingWithData matches with requests and takes in the returned payload, to
// be processed outside of a stream processing routine and returns true if a handler was found
func (m *Match) IncomingWithData(signature interface{}, data []byte) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch, ok := m.m[signature]
	if ok {
		select {
		case ch <- data:
		default:
			// this shouldn't occur but if it does continue to process as normal
			return false
		}
		return true
	}
	return false
}

// Set the signature response channel for incoming data
func (m *Match) Set(signature interface{}) (Matcher, error) {
	var ch chan []byte
	m.mu.Lock()
	if _, ok := m.m[signature]; ok {
		m.mu.Unlock()
		return Matcher{}, errors.New("signature collision")
	}
	// This is buffered so we don't need to wait for receiver.
	ch = make(chan []byte, 1)
	m.m[signature] = ch
	m.mu.Unlock()

	return Matcher{
		C:   ch,
		sig: signature,
		m:   m,
	}, nil
}

// Cleanup closes underlying channel and deletes signature from map
func (m *Matcher) Cleanup() {
	m.m.mu.Lock()
	close(m.C)
	delete(m.m.m, m.sig)
	m.m.mu.Unlock()
}
