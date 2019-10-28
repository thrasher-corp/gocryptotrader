package timedmutex

import (
	"sync"
	"time"
)

type TimedMutex struct {
	mtx       sync.Mutex
	timerLock sync.RWMutex
	Timer     *time.Timer
	duration  time.Duration
}
