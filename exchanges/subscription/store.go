package subscription

import (
	"maps"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common"
)

// Store is a container of subscription pointers
type Store struct {
	m  map[any]*Subscription
	mu sync.RWMutex
}

// NewStore creates a ready to use store and should always be used
func NewStore() *Store {
	return &Store{
		m: map[any]*Subscription{},
	}
}

// NewStoreFromList creates a Store from a List
func NewStoreFromList(l List) (*Store, error) {
	s := NewStore()
	for _, sub := range l {
		if err := s.add(sub); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// Add adds a subscription to the store
// Key can be already set; if omitted EnsureKeyed will be used
// Errors if it already exists
func (s *Store) Add(sub *Subscription) error {
	if s == nil || sub == nil {
		return common.ErrNilPointer
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.add(sub)
}

// Add adds a subscription to the store
// This method provides no locking protection
func (s *Store) add(sub *Subscription) error {
	if s.m == nil {
		s.m = map[any]*Subscription{}
	}
	key := sub.EnsureKeyed()
	if found := s.get(key); found != nil {
		return ErrDuplicate
	}
	s.m[key] = sub
	return nil
}

// Get returns a pointer to a subscription or nil if not found
// If the key passed in is a Subscription then its Key will be used; which may be a pointer to itself.
// If key implements MatchableKey then key.Match will be used; Note that *Subscription implements MatchableKey
func (s *Store) Get(key any) *Subscription {
	if s == nil || s.m == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.get(key)
}

// get returns a pointer to subscription or nil if not found
// If the key passed in is a Subscription then its Key will be used; which may be a pointer to itself.
// If key implements MatchableKey then key.Match will be used; Note that *Subscription implements MatchableKey
// This method provides no locking protection
func (s *Store) get(key any) *Subscription {
	if s.m == nil {
		return nil
	}
	switch v := key.(type) {
	case Subscription:
		key = v.EnsureKeyed()
	case *Subscription:
		key = v.EnsureKeyed()
	}

	switch v := key.(type) {
	case MatchableKey:
		return s.match(v)
	default:
		return s.m[v]
	}
}

// Remove removes a subscription from the store
// If the key passed in is a Subscription then its Key will be used; which may be a pointer to itself.
// If key implements MatchableKey then key.Match will be used; Note that *Subscription implements MatchableKey
func (s *Store) Remove(key any) error {
	if s == nil || key == nil {
		return common.ErrNilPointer
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if found := s.get(key); found != nil {
		delete(s.m, found.Key)
		return nil
	}

	return ErrNotFound
}

// List returns a slice of Subscriptions pointers
func (s *Store) List() List {
	if s == nil || s.m == nil {
		return List{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	subs := make(List, 0, len(s.m))
	for _, s := range s.m {
		subs = append(subs, s)
	}
	return subs
}

// Clear empties the subscription store
func (s *Store) Clear() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.m == nil {
		s.m = map[any]*Subscription{}
	}
	clear(s.m)
}

// match returns the first subscription which matches the Key's Asset, Channel and Pairs
// If the key provided has:
// 1) Empty pairs then only Subscriptions without pairs will be considered
// 2) >=1 pairs then Subscriptions which contain all the pairs will be considered
// This method provides no locking protection
func (s *Store) match(key MatchableKey) *Subscription {
	for anyKey, s := range s.m {
		if key.Match(anyKey) {
			return s
		}
	}
	return nil
}

// Diff returns a list of the added and missing subs from a new list
// The store Diff is invoked upon is read-lock protected
// The new store is assumed to be a new instance and enjoys no locking protection
func (s *Store) Diff(compare List) (added, removed List) {
	if s == nil {
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	removedMap := maps.Clone(s.m)
	for _, sub := range compare {
		if found := s.get(sub); found != nil {
			delete(removedMap, found.Key)
		} else {
			added = append(added, sub)
		}
	}

	for _, c := range removedMap {
		removed = append(removed, c)
	}

	return
}

// Len returns the number of subscriptions
func (s *Store) Len() int {
	if s == nil || s.m == nil {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.m)
}
