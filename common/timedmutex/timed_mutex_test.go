package timedmutex

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
)

func BenchmarkTimedMutexTime(b *testing.B) {
	tm := NewTimedMutex(0)
	for b.Loop() {
		tm.LockForDuration()
	}
}

func BenchmarkTimedMutexTimeUnlockNotPrimed(b *testing.B) {
	tm := NewTimedMutex(0)
	for b.Loop() {
		tm.UnlockIfLocked()
	}
}

func BenchmarkTimedMutexTimeUnlockPrimed(b *testing.B) {
	tm := NewTimedMutex(0)
	tm.LockForDuration()
	for b.Loop() {
		tm.UnlockIfLocked()
	}
}

func BenchmarkTimedMutexTimeLinearInteraction(b *testing.B) {
	tm := NewTimedMutex(0)
	for b.Loop() {
		tm.LockForDuration()
		tm.UnlockIfLocked()
	}
}

func TestConsistencyOfPanicFreeUnlock(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(*testing.T) {
		duration := 20 * time.Microsecond
		tm := NewTimedMutex(duration)
		for i := 1; i <= 50; i++ {
			testUnlockTime := time.Duration(i) * time.Microsecond
			tm.LockForDuration()
			// sweeps either side of duration, so both the timer and the call get to unlock
			synctest.Sleep(testUnlockTime)
			tm.UnlockIfLocked()
		}
	})
}

func TestUnlockAfterTimeout(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
		tm := NewTimedMutex(time.Nanosecond)
		tm.LockForDuration()
		synctest.Sleep(time.Millisecond)
		assert.False(t, tm.UnlockIfLocked(), "UnlockIfLocked should report the timeout unlocked the mutex, not the call")
	})
}

func TestUnlockBeforeTimeout(t *testing.T) {
	t.Parallel()
	tm := NewTimedMutex(20 * time.Millisecond)
	tm.LockForDuration()
	wasUnlocked := tm.UnlockIfLocked()
	if !wasUnlocked {
		t.Error("Mutex should have been unlocked by command, not timeout")
	}
}

// TestUnlockAtSameTimeAsTimeout this test ensures
// that even if the timeout and the command occur at
// the same time, no panics occur. The result of the
// 'who' unlocking this doesn't matter, so long as
// the unlock occurs without this test panicking
func TestUnlockAtSameTimeAsTimeout(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(*testing.T) {
		duration := time.Millisecond
		tm := NewTimedMutex(duration)
		tm.LockForDuration()
		synctest.Sleep(duration)
		tm.UnlockIfLocked()
	})
}

func TestMultipleUnlocks(t *testing.T) {
	t.Parallel()
	tm := NewTimedMutex(10 * time.Second)
	tm.LockForDuration()
	wasUnlocked := tm.UnlockIfLocked()
	if !wasUnlocked {
		t.Error("Mutex should have been unlocked by command, not timeout")
	}
	wasUnlocked = tm.UnlockIfLocked()
	if wasUnlocked {
		t.Error("Mutex should have been already unlocked by command")
	}
	wasUnlocked = tm.UnlockIfLocked()
	if wasUnlocked {
		t.Error("Mutex should have been already unlocked by command")
	}
}

func TestJustWaitItOut(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(*testing.T) {
		tm := NewTimedMutex(1 * time.Millisecond)
		tm.LockForDuration()
		synctest.Sleep(2 * time.Millisecond)
	})
}
