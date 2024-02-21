package timedmutex

import (
	"sync"
	"sync/atomic"
	"time"
)

// TimedMutex is a blocking mutex which will unlock after a specified time
type TimedMutex struct {
	primary   sync.Mutex
	secondary sync.Mutex
	timer     *time.Timer
	primed    atomic.Bool
	duration  time.Duration
}

// NewTimedMutex creates a new timed mutex with a specified duration
func NewTimedMutex(length time.Duration) *TimedMutex { return &TimedMutex{duration: length} }

// LockForDuration will start a timer, lock the mutex, then allow the caller to continue
// After the duration, the mutex will be unlocked
func (t *TimedMutex) LockForDuration() {
	t.primary.Lock()
	if t.primed.CompareAndSwap(false, true) {
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
	hasStopped := t.timer.Stop()
	t.secondary.Unlock()

	if !hasStopped {
		// Timer has already fired and the mutex has been unlocked.
		// Timer C channel is not used with AfterFunc, so no need to drain.
		return false
	}
	t.primary.Unlock()
	return true
}
