package timedmutex

import (
	"sync"
	"time"
)

// TimedMutex is a blocking mutex which will unlock
// after a specified time
type TimedMutex struct {
	mtx      sync.Mutex
	timer    *time.Timer
	duration time.Duration
}

// NewTimedMutex creates a new timed mutex with a
// specified duration
func NewTimedMutex(length time.Duration) *TimedMutex { return &TimedMutex{duration: length} }

// LockForDuration will start a timer, lock the mutex,
// then allow the caller to continue
// After the duration, the mutex will be unlocked
func (t *TimedMutex) LockForDuration() {
	t.mtx.Lock()
	if t.timer == nil {
		t.timer = time.AfterFunc(t.duration, func() { t.mtx.Unlock() })
	} else {
		// Timer C channel is not used with AfterFunc, so no need to drain.
		t.timer.Reset(t.duration)
	}
}

// UnlockIfLocked will unlock the mutex if its currently locked
// Will return true if successfully unlocked
func (t *TimedMutex) UnlockIfLocked() bool {
	if t.timer == nil || !t.timer.Stop() {
		// Timer has already fired and the mutex has been unlocked.
		// Timer C channel is not used with AfterFunc, so no need to drain.
		return false
	}
	t.mtx.Unlock()
	return true
}
