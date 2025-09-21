package quickdata

import (
	"maps"
	"slices"
	"sync"
)

// FocusStore is a store for FocusData instances identified by FocusType
// Use methods to interact with the store in a thread-safe manner
type FocusStore struct {
	s map[FocusType]*FocusData
	m sync.RWMutex
}

// NewFocusStore creates a ready to use FocusStore
func NewFocusStore() *FocusStore {
	return &FocusStore{
		s: make(map[FocusType]*FocusData),
	}
}

// Upsert adds or updates FocusData in the store.
// If data is nil, the method does nothing. Use Remove to delete entries.
func (s *FocusStore) Upsert(key FocusType, data *FocusData) {
	if data == nil {
		return
	}
	s.m.Lock()
	defer s.m.Unlock()
	data.focusType = key
	if data.FailureTolerance == 0 {
		data.Init()
	}
	s.s[key] = data
}

// Remove deletes FocusData from the store by its FocusType key.
// It does nothing with the data itself, like closing channels or cleaning up resources.
func (s *FocusStore) Remove(key FocusType) {
	s.m.Lock()
	defer s.m.Unlock()
	delete(s.s, key)
}

// GetByFocusType returns FocusData if exists
func (s *FocusStore) GetByFocusType(key FocusType) *FocusData {
	s.m.RLock()
	defer s.m.RUnlock()
	return s.s[key]
}

// List returns a slice of FocusData pointers to iterate over
// Note: order of the slice is not guaranteed
// Note: is unsafe
func (s *FocusStore) List() []*FocusData {
	s.m.RLock()
	defer s.m.RUnlock()
	return slices.Collect(maps.Values(s.s))
}

// DisableWebsocketFocuses sets all FocusData in the store to not use websockets
func (s *FocusStore) DisableWebsocketFocuses() {
	s.m.Lock()
	defer s.m.Unlock()
	for k := range s.s {
		s.s[k].m.Lock()
		s.s[k].useWebsocket = false
		s.s[k].m.Unlock()
	}
}
