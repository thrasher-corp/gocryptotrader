package quickspy

import (
	"sync"
	"time"
)

func NewFocusData(focusType FocusType, isOnceOff, useWebsocket bool, restPollTime time.Duration) *FocusData {
	return &FocusData{
		Type:                  focusType,
		Enabled:               true,
		UseWebsocket:          useWebsocket,
		RESTPollTime:          restPollTime, // 10 seconds in nanoseconds
		m:                     new(sync.RWMutex),
		IsOnceOff:             isOnceOff,
		HasBeenSuccessfulChan: make(chan any),
		Stream:                make(chan any),
	}
}

// Init called to ensure that lame data is initialised
func (f *FocusData) Init() {
	f.Enabled = true
	f.m = new(sync.RWMutex)
	f.HasBeenSuccessfulChan = make(chan any)
	f.Stream = make(chan any)
	f.hasBeenSuccessful = false
}

func (f *FocusData) SetSuccessful() {
	f.m.Lock()
	defer f.m.Unlock()
	f.hasBeenSuccessful = true
	close(f.HasBeenSuccessfulChan)
}
