package nonce

import (
	"strconv"
	"sync"
)

// Nonce struct holds the nonce value
type Nonce struct {
	n   int64
	mtx sync.Mutex
}

// Inc increments the nonce value
func (n *Nonce) Inc() {
	n.mtx.Lock()
	n.n++
	n.mtx.Unlock()
}

// Get retrives the nonce value
func (n *Nonce) Get() int64 {
	n.mtx.Lock()
	defer n.mtx.Unlock()
	return n.n
}

// GetInc increments and returns the value of the nonce
func (n *Nonce) GetInc() int64 {
	n.mtx.Lock()
	defer n.mtx.Unlock()
	n.n++
	return n.n
}

// Set sets the nonce value
func (n *Nonce) Set(val int64) {
	n.mtx.Lock()
	n.n = val
	n.mtx.Unlock()
}

// Returns a string version of the nonce
func (n *Nonce) String() string {
	n.mtx.Lock()
	result := strconv.FormatInt(n.n, 10)
	n.mtx.Unlock()
	return result
}
