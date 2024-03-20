package nonce

import (
	"strconv"
	"sync"
	"time"
)

// Type is a type of nonce start value
type Type uint8

const (
	// Seconds is a type of nonce start value time.Now().Unix()
	Seconds Type = iota
	// Nanoseconds is a type of nonce start value time.Now().UnixNano()
	Nanoseconds
)

// Nonce struct holds the nonce value
type Nonce struct {
	n int64
	m sync.Mutex
}

// GetAndIncrement returns the current nonce value and increments it. If value
// is 0, it will set the value to the current time.
func (n *Nonce) GetAndIncrement(nonceType Type) Value {
	n.m.Lock()
	defer n.m.Unlock()
	if n.n == 0 {
		switch nonceType {
		case Nanoseconds:
			n.n = time.Now().UnixNano()
		case Seconds:
			n.n = time.Now().Unix()
		}
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
