package websocket

import (
	"errors"
	"fmt"
	"sync"
)

// ErrSignatureNotMatched is returned when a signature does not match a request
var ErrSignatureNotMatched = errors.New("websocket response to request signature not matched")

var (
	errSignatureCollision = errors.New("signature collision")
	errInvalidBufferSize  = errors.New("buffer size must be positive")
)

// NewMatch returns a new Match
func NewMatch() *Match {
	return &Match{m: make(map[any]*incoming)}
}

// Match is a distributed subtype that handles the matching of requests and
// responses in a timely manner, reducing the need to differentiate between
// connections. Stream systems fan in all incoming payloads to one routine for
// processing.
type Match struct {
	m  map[any]*incoming
	mu sync.Mutex
}

type incoming struct {
	expected int
	c        chan<- []byte
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
	ch.c <- data
	ch.expected--
	if ch.expected == 0 {
		close(ch.c)
		delete(m.m, signature)
	}
	return true
}

// RequireMatchWithData validates that incoming data matches a request's signature.
// If a match is found, the data is processed; otherwise, it returns an error.
func (m *Match) RequireMatchWithData(signature any, data []byte) error {
	if m.IncomingWithData(signature, data) {
		return nil
	}
	return fmt.Errorf("'%v' %w with data %v", signature, ErrSignatureNotMatched, string(data))
}

// Set the signature response channel for incoming data
func (m *Match) Set(signature any, bufSize int) (<-chan []byte, error) {
	if bufSize <= 0 {
		return nil, errInvalidBufferSize
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.m[signature]; ok {
		return nil, errSignatureCollision
	}
	ch := make(chan []byte, bufSize)
	m.m[signature] = &incoming{expected: bufSize, c: ch}
	return ch, nil
}

// RemoveSignature removes the signature response from map and closes the channel.
func (m *Match) RemoveSignature(signature any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ch, ok := m.m[signature]; ok {
		close(ch.c)
		delete(m.m, signature)
	}
}
