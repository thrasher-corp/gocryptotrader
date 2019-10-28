package timedmutex

import (
	"sync"
	"time"
)

func NewTimedMutex(length time.Duration) *TimedMutex {
	return &TimedMutex{
		duration: length,
	}
}

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
// The timer will be nil if the timeout has been hit
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

func (t *TimedMutex) stopTimer() bool {
	t.timerLock.Lock()
	defer t.timerLock.Unlock()
	if !t.Timer.Stop() {
		select {
		case <-t.Timer.C:
		default:
		}
		return false
	}
	return true
}

func (t *TimedMutex) isTimerNil() bool {
	t.timerLock.RLock()
	defer t.timerLock.RUnlock()
	return t.Timer == nil
}

func (t *TimedMutex) setTimer() {
	t.timerLock.Lock()
	t.Timer = time.AfterFunc(t.duration, func() {
		t.mtx.Unlock()
	})
	t.timerLock.Unlock()
}
