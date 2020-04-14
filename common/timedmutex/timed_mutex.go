package timedmutex

import (
	"sync"
	"time"
)

// NewTimedMutex creates a new timed mutex with a
// specified duration
func NewTimedMutex(length time.Duration) *TimedMutex {
	return &TimedMutex{
		duration: length,
	}
}

// LockForDuration will start a timer, lock the mutex,
// then allow the caller to continue
// After the duration, the mutex will be unlocked
func (t *TimedMutex) LockForDuration() {
	var wg sync.WaitGroup
	wg.Add(1)
	go t.lockAndSetTimer(&wg)
	wg.Wait()
}

func (t *TimedMutex) lockAndSetTimer(wg *sync.WaitGroup) {
	t.mtx.Lock()
	t.setTimer()
	wg.Done()
}

// UnlockIfLocked will unlock the mutex if its currently locked
// Will return true if successfully unlocked
func (t *TimedMutex) UnlockIfLocked() bool {
	if t.isTimerNil() {
		return false
	}

	if !t.stopTimer() {
		return false
	}
	t.mtx.Unlock()
	return true
}

// stopTimer will return true if timer has been stopped by this command
// If the timer has expired, clear the channel
func (t *TimedMutex) stopTimer() bool {
	t.timerLock.Lock()
	defer t.timerLock.Unlock()
	if !t.timer.Stop() {
		select {
		case <-t.timer.C:
		default:
		}
		return false
	}
	return true
}

// isTimerNil safely read locks to detect nil
func (t *TimedMutex) isTimerNil() bool {
	t.timerLock.RLock()
	isNil := t.timer == nil
	t.timerLock.RUnlock()
	return isNil
}

// setTimer safely locks and sets a timer
// which will automatically execute a mutex unlock
// once timer expires
func (t *TimedMutex) setTimer() {
	t.timerLock.Lock()
	t.timer = time.AfterFunc(t.duration, func() {
		t.mtx.Unlock()
	})
	t.timerLock.Unlock()
}
