package timedmutex

import (
	"sync"
	"time"
)

// TimedMutex is a blocking mutex which will unlock
// after a specified time
type TimedMutex struct {
	mtx       sync.Mutex
	timerLock sync.RWMutex
	timer     *time.Timer
	duration  time.Duration
}
