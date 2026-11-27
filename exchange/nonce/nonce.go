package nonce

import (
	"strconv"
	"sync"
	"time"
)

// UnixNano and Unix are default nonce setters
var (
	UnixNano Setter = func() int64 { return time.Now().UnixNano() }
	Unix     Setter = func() int64 { return time.Now().Unix() }
)

// Setter is a function that returns a nonce start value.
type Setter func() int64

// Nonce struct holds the nonce value
type Nonce struct {
	n int64
	m sync.Mutex
}

// GetAndIncrement returns the current nonce value and increments it. If value
// is 0, it will set the value to the current time.
func (n *Nonce) GetAndIncrement(set Setter) Value {
	n.m.Lock()
	defer n.m.Unlock()
	if n.n == 0 {
		n.n = set()
	}
	val := n.n
	n.n++
	return Value(val)
}

// Value is a return type for GetValue
type Value int64

// String is a Value method that changes format to a string
func (v Value) String() string {
	return strconv.FormatInt(int64(v), 10)
}
