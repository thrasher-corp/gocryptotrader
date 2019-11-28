package nonce

import (
	"strconv"
	"sync"
)

// Nonce struct holds the nonce value
type Nonce struct {
	n int64
	m sync.Mutex
}

// Inc increments the nonce value
func (n *Nonce) Inc() {
	n.m.Lock()
	n.n++
	n.m.Unlock()
}

// Get retrives the nonce value
func (n *Nonce) Get() Value {
	n.m.Lock()
	defer n.m.Unlock()
	return Value(n.n)
}

// GetInc increments and returns the value of the nonce
func (n *Nonce) GetInc() Value {
	n.Inc()
	return n.Get()
}

// Set sets the nonce value
func (n *Nonce) Set(val int64) {
	n.m.Lock()
	n.n = val
	n.m.Unlock()
}

// String returns a string version of the nonce
func (n *Nonce) String() string {
	return n.Get().String()
}

// Value is a return type for GetValue
type Value int64

// String is a Value method that changes format to a string
func (v Value) String() string {
	return strconv.FormatInt(int64(v), 10)
}
