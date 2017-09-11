package nonce

import (
	"strconv"
	"sync"
	"time"
)

// Nonce struct holds the nonce value
type Nonce struct {
	// Standard nonce
	n   int64
	mtx sync.Mutex
	// Hash table exclusive exchange specific nonce values
	boundedCall map[string]int64
	boundedMtx  sync.Mutex
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

// Value is a return type for GetValue
type Value int64

// GetValue returns a nonce value and can be set as a higher precision. Values
// stored in an exchange specific hash table using a single locked call.
func (n *Nonce) GetValue(exchName string, nanoPrecision bool) Value {
	n.boundedMtx.Lock()
	defer n.boundedMtx.Unlock()

	if n.boundedCall == nil {
		n.boundedCall = make(map[string]int64)
	}

	if n.boundedCall[exchName] == 0 {
		if nanoPrecision {
			n.boundedCall[exchName] = time.Now().UnixNano()
			return Value(n.boundedCall[exchName])
		}
		n.boundedCall[exchName] = time.Now().Unix()
		return Value(n.boundedCall[exchName])
	}
	n.boundedCall[exchName]++
	return Value(n.boundedCall[exchName])
}

// String is a Value method that changes format to a string
func (v Value) String() string {
	return strconv.FormatInt(int64(v), 10)
}
