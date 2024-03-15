package timedmutex

import (
	"sync"
	"sync/atomic"
	"time"
)

// TimedMutex is a blocking mutex which will unlock after a specified time
type TimedMutex struct {
	// primary mutex is the main lock that will be unlocked after the duration
	primary sync.Mutex
	// secondary mutex is used to protect the timer
	secondary sync.Mutex
	timer     *time.Timer
	// primed is used to determine if the timer has been started this is
	// slightly more performant than checking the timer directly and interacting
	// with a RW mutex.
	primed   atomic.Bool
	duration time.Duration
}

// NewTimedMutex creates a new timed mutex with a specified duration
func NewTimedMutex(length time.Duration) *TimedMutex {
	return &TimedMutex{duration: length}
}

// LockForDuration will start a timer, lock the mutex, then allow the caller to continue
// After the duration, the mutex will be unlocked
func (t *TimedMutex) LockForDuration() {
	t.primary.Lock()
	if !t.primed.Swap(true) {
		t.secondary.Lock()
		t.timer = time.AfterFunc(t.duration, func() { t.primary.Unlock() })
		t.secondary.Unlock()
	} else {
		// Timer C channel is not used with AfterFunc, so no need to drain.
		t.secondary.Lock()
		t.timer.Reset(t.duration)
		t.secondary.Unlock()
	}
}

// UnlockIfLocked will unlock the mutex if its currently locked Will return true
// if successfully unlocked
func (t *TimedMutex) UnlockIfLocked() bool {
	if !t.primed.Load() {
		return false
	}

	t.secondary.Lock()
	wasStoppedByCall := t.timer.Stop()
	t.secondary.Unlock()

	if !wasStoppedByCall {
		// Timer has already fired and the mutex has been unlocked.
		// Timer C channel is not used with AfterFunc, so no need to drain.
		return false
	}
	t.primary.Unlock()
	return true
}
