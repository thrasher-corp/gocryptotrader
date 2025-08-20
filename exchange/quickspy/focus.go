package quickspy

import (
	"maps"
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

func NewFocusStore() *FocusStore {
	return &FocusStore{
		s: make(map[FocusType]*FocusData),
		m: new(sync.RWMutex),
	}
}

func (s *FocusStore) Upsert(key FocusType, data *FocusData) {
	s.m.Lock()
	defer s.m.Unlock()
	if data == nil {
		delete(s.s, key)
		return
	}
	data.Type = key
	if data.m == nil {
		data.m = new(sync.RWMutex)
	}
	s.s[key] = data
}

func (f *FocusData) SetSuccessful() {
	f.m.Lock()
	defer f.m.Unlock()
	f.hasBeenSuccessful = true
	close(f.HasBeenSuccessfulChan)
}

func (s *FocusStore) GetByKey(key FocusType) *FocusData {
	s.m.RLock()
	defer s.m.RUnlock()
	data, ok := s.s[key]
	if !ok {
		return nil
	}
	return data
}

// List returns a new list of store data.
// store data are pointers
func (s *FocusStore) List() []*FocusData {
	s.m.RLock()
	defer s.m.RUnlock()
	list := make([]*FocusData, len(s.s))
	for v := range maps.Values(s.s) {
		list = append(list, v)
	}
	return list
}
