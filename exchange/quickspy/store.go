package quickspy

import (
	"sync"
)

func NewFocusStore() *FocusStore {
	return &FocusStore{
		s: make(map[FocusType]*FocusData),
		m: new(sync.RWMutex),
	}
}

// Upsert adds or updates FocusData in the store.
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

// GetByFocusType returns FocusData if exists
func (s *FocusStore) GetByFocusType(key FocusType) *FocusData {
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
	list := make([]*FocusData, 0, len(s.s))
	for _, v := range s.s {
		list = append(list, v)
	}
	return list
}

func (s *FocusStore) DisableWebsocketFocuses() {
	s.m.Lock()
	defer s.m.Unlock()
	for k := range s.s {
		s.s[k].m.Lock()
		s.s[k].UseWebsocket = false
		s.s[k].m.Unlock()
	}
}
