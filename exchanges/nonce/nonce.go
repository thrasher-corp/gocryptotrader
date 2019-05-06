package nonce

import (
	"strconv"
	"sync/atomic"
)

// Nonce struct holds the nonce value
type Nonce struct {
	n int64
}

// Inc increments the nonce value
func (n *Nonce) Inc() {
	atomic.AddInt64(&n.n, 1)
}

// Get retrives the nonce value
func (n *Nonce) Get() Value {
	return Value(atomic.LoadInt64(&n.n))
}

// GetInc increments and returns the value of the nonce
func (n *Nonce) GetInc() Value {
	n.Inc()
	return n.Get()
}

// Set sets the nonce value
func (n *Nonce) Set(val int64) {
	atomic.StoreInt64(&n.n, val)
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
