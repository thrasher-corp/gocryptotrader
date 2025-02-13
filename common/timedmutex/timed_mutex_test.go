package timedmutex

import (
	"testing"
	"time"
)

// 1000000	        1074 ns/op	     136 B/op	       4 allocs/op (prev)
// 2423571	       503.9 ns/op	       0 B/op	       0 allocs/op (current)
func BenchmarkTimedMutexTime(b *testing.B) {
	tm := NewTimedMutex(0)
	for b.Loop() {
		tm.LockForDuration()
	}
}

// 352309195	         3.194 ns/op	       0 B/op	       0 allocs/op (prev)
// 927051118	         1.298 ns/op	       0 B/op	       0 allocs/op
func BenchmarkTimedMutexTimeUnlockNotPrimed(b *testing.B) {
	tm := NewTimedMutex(0)
	for b.Loop() {
		tm.UnlockIfLocked()
	}
}

// 95322825				15.51 ns/op	       0 B/op	       0 allocs/op (prev)
// 239158972			4.621 ns/op	       0 B/op	       0 allocs/op
func BenchmarkTimedMutexTimeUnlockPrimed(b *testing.B) {
	tm := NewTimedMutex(0)
	tm.LockForDuration()
	for b.Loop() {
		tm.UnlockIfLocked()
	}
}

// 1000000	         1193 ns/op	     136 B/op	       4 allocs/op (prev)
// 38592405	        36.12 ns/op	       0 B/op	       0 allocs/op
func BenchmarkTimedMutexTimeLinearInteraction(b *testing.B) {
	tm := NewTimedMutex(0)
	for b.Loop() {
		tm.LockForDuration()
		tm.UnlockIfLocked()
	}
}

func TestConsistencyOfPanicFreeUnlock(t *testing.T) {
	t.Parallel()
	duration := 20 * time.Microsecond
	tm := NewTimedMutex(duration)
	for i := 1; i <= 50; i++ {
		testUnlockTime := time.Duration(i) * time.Microsecond
		tm.LockForDuration()
		time.Sleep(testUnlockTime)
		tm.UnlockIfLocked()
	}
}

func TestUnlockAfterTimeout(t *testing.T) {
	t.Parallel()
	tm := NewTimedMutex(time.Nanosecond)
	tm.LockForDuration()
	time.Sleep(time.Millisecond * 200)
	wasUnlocked := tm.UnlockIfLocked()
	if wasUnlocked {
		t.Error("Mutex should have been unlocked by timeout, not command")
	}
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
	duration := time.Millisecond
	tm := NewTimedMutex(duration)
	tm.LockForDuration()
	time.Sleep(duration)
	tm.UnlockIfLocked()
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
	tm := NewTimedMutex(1 * time.Millisecond)
	tm.LockForDuration()
	time.Sleep(2 * time.Millisecond)
}
