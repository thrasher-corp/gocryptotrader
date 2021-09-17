package alert

import (
	"sync"
	"sync/atomic"
)

// Notice defines fields required to alert sub-systems of a change of state to
// re-check depth list
type Notice struct {
	// Channel to wait for an alert on.
	forAlert chan struct{}
	// Lets the updater functions know if there are any routines waiting for an
	// alert.
	sema uint32
	// After closing the forAlert channel this will notify when all the routines
	// that have waited, have either checked the orderbook depth or finished.
	wg sync.WaitGroup
	// Segregated lock only for waiting routines, so as this does not interfere
	// with the main depth lock, acts as a rolling gate.
	m sync.Mutex
}

// Alert establishes a state change on the orderbook depth.
func (n *Notice) Alert() {
	// CompareAndSwap is used to swap from 1 -> 2 so we don't keep actuating
	// the opposing compare and swap in method wait. This function can return
	// freely when an alert operation is in process.
	if !atomic.CompareAndSwapUint32(&n.sema, 1, 2) {
		// Return if no waiting routines or currently alerting.
		return
	}
	go n.actuate()
}

// Actuate lock in a different routine, as alerting is a second order priority
// compared to updating and releasing calling routine.
func (n *Notice) actuate() {
	n.m.Lock()
	// Closing; alerts many waiting routines.
	close(n.forAlert)
	// Wait for waiting routines to receive alert and return.
	n.wg.Wait()
	atomic.SwapUint32(&n.sema, 0) // Swap back to neutral state.
	n.m.Unlock()
}

// Wait pauses calling routine until depth change has been established via depth
// method alert. Kick allows for cancellation of waiting or when the caller
// has been shut down, if this is not needed it can be set to nil. This
// returns a channel so strategies can cleanly wait on a select statement case.
func (n *Notice) Wait(kick <-chan struct{}) <-chan bool {
	reply := make(chan bool)
	n.m.Lock()
	n.wg.Add(1)
	if atomic.CompareAndSwapUint32(&n.sema, 0, 1) {
		n.forAlert = make(chan struct{})
	}
	go n.hold(reply, kick)
	n.m.Unlock()
	return reply
}

// hold waits on either channel in the event that the routine has finished or an
// alert from a depth update has occurred.
func (n *Notice) hold(ch chan<- bool, kick <-chan struct{}) {
	select {
	// In a select statement, if by chance there is no receiver or its late,
	// we can still close and return, limiting dead-lock potential.
	case <-n.forAlert: // Main waiting channel from alert
		select {
		case ch <- false:
		default:
		}
	case <-kick: // This can be nil.
		select {
		case ch <- true:
		default:
		}
	}
	n.wg.Done()
	close(ch)
}
