package okx

import "time"

// positionModeCacheEntry keeps one account's lookup and mode changes coordinated.
type positionModeCacheEntry struct {
	mode       string
	expires    time.Time
	ready      chan struct{}
	lookup     *positionModeLookup
	changed    chan struct{}
	changing   int
	generation uint64
}

// positionModeLookup retains a flight's result for its existing waiters only.
type positionModeLookup struct {
	mode       string
	err        error
	generation uint64
}
