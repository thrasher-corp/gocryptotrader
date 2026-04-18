package alert

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

const (
	inactive = uint32(iota)
	active
	alerting

	dataToActuatorDefaultBuffer = 1

	// PreAllocCommsDefaultBuffer is the default buffer size for comms
	PreAllocCommsDefaultBuffer = 5
)

var (
	// pool is a silent shared pool between all notice instances for alerting
	// external routines waiting on a state change.
	pool = sync.Pool{New: func() any { return make(chan bool) }}

	preAllocBufferSize = PreAllocCommsDefaultBuffer
	mu                 sync.RWMutex

	errInvalidBufferSize = errors.New("invalid buffer size cannot be equal or less than zero")
)

// SetPreAllocationCommsBuffer sets buffer size of the pre-allocated comms.
func SetPreAllocationCommsBuffer(size int) error {
	if size <= 0 {
		return fmt.Errorf("%w received %v", errInvalidBufferSize, size)
	}
	mu.Lock()
	preAllocBufferSize = size
	mu.Unlock()
	return nil
}

// SetDefaultPreAllocationCommsBuffer sets default buffer size of the
// pre-allocated comms.
func SetDefaultPreAllocationCommsBuffer() {
	mu.Lock()
	preAllocBufferSize = PreAllocCommsDefaultBuffer
	mu.Unlock()
}

// Notice defines fields required to alert sub-systems of a change of state so a
// routine can re-check in memory data
type Notice struct {
	// Channel to wait for an alert on.
	forAlert chan struct{}
	// Lets the updater functions know if there are any routines waiting for an
	// alert.
	sema uint32
	// After closing the forAlert channel this will notify when all the routines
	// that have waited, have completed their checks.
	wg sync.WaitGroup
	// Segregated lock only for waiting routines, so as this does not interfere
	// with the main calling lock, this acts as a rolling gate.
	mu sync.Mutex
	// toActuatorRoutine is communication between the alert call and the
	// actuator routine
	toActuatorRoutine chan struct{}
	// alerters are a pre allocated channel of communications pipes
	alerters chan chan struct{}
}

// Alert establishes a state change on the required struct.
func (n *Notice) Alert() {
	// CompareAndSwap is used to swap from 1 -> 2 so we don't keep actuating
	// the opposing compare and swap in method wait. This function can return
	// freely when an alert operation is in process.
	if !atomic.CompareAndSwapUint32(&n.sema, active, alerting) {
		// Return if no waiting routines or currently alerting.
		return
	}

	if n.toActuatorRoutine == nil {
		// Buffered communications channel in communication with actuate routine,
		// so as to not worry about slow receivers that will inhibit alert
		// returning.
		n.toActuatorRoutine = make(chan struct{}, dataToActuatorDefaultBuffer)
		// Spawn persistent routine that blocks only when required instead of
		// spawning a routine for every alert.
		go n.actuate()
	}
	// Buffered channel will alert actuate routine without waiting and return.
	n.toActuatorRoutine <- struct{}{}
}

// actuate locks in a different routine, as alerting is a second order priority
// compared to updating and releasing calling routine
func (n *Notice) actuate() {
	for range n.toActuatorRoutine {
		n.mu.Lock()
		// Closing; alerts many waiting routines.
		close(n.forAlert)
		// Wait for waiting routines to receive alert and return.
		n.wg.Wait()
		atomic.SwapUint32(&n.sema, inactive) // Swap back to neutral state.
		n.mu.Unlock()
	}
}

// generator routine pre-loads chan struct communicators that will be closed.
func (n *Notice) generator() {
	for {
		// This will block once filled appropriately.
		n.alerters <- make(chan struct{})
	}
}

// Wait pauses calling routine until change of state has been established via
// notice method Alert. Kick allows for cancellation of waiting or when the
// caller has been shut down, if this is not needed it can be set to nil. This
// returns a channel so strategies can cleanly wait on a select statement case.
// NOTE: Please see README.md for implementation example.
func (n *Notice) Wait(kick <-chan struct{}) chan bool {
	reply, ok := pool.Get().(chan bool)
	if !ok {
		reply = make(chan bool)
	}
	n.mu.Lock()
	if atomic.CompareAndSwapUint32(&n.sema, inactive, active) {
		if n.alerters == nil {
			mu.RLock()
			n.alerters = make(chan chan struct{}, preAllocBufferSize)
			mu.RUnlock()
			go n.generator()
		}
		n.forAlert = <-n.alerters
	}
	n.wg.Add(1)
	go n.hold(reply, kick)
	n.mu.Unlock()
	return reply
}

// hold waits on either channel in the event that the routine has
// finished/cancelled or an alert from an update has occurred. This routine
// has the potential to leak if receivers never read but this ensures sanity
// instead of closing and differentiation between alerting and kicking, also
// ensures chan bool item is clean before being put back into pool.
func (n *Notice) hold(ch chan bool, kick <-chan struct{}) {
	select {
	case <-n.forAlert: // Main waiting channel from alert
		n.wg.Done()
		ch <- false
	case <-kick: // This can be nil.
		n.wg.Done()
		ch <- true
	}
	pool.Put(ch)
}
