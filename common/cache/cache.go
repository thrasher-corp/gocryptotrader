package cache

// New returns a new thread-safe LRU cache with input capacity
func New(cap uint64) *LRUCache {
	return &LRUCache{
		lru: NewLRUCache(cap),
	}
}

// Add new entry to Cache return true if entry removed
func (l *LRUCache) Add(k, v interface{}) {
	l.m.Lock()
	l.lru.Add(k, v)
	l.m.Unlock()
}

// Get looks up a key's value from the cache.
func (l *LRUCache) Get(key interface{}) (value interface{}, ok bool) {
	l.m.Lock()
	defer l.m.Unlock()
	return l.lru.Get(key)
}

// Len returns the number of items in the cache.
func (l *LRUCache) Len() uint64 {
	l.m.Lock()
	defer l.m.Unlock()
	return l.lru.Len()
}

// Clear is used to clear the cache.
func (l *LRUCache) Clear() {
	l.m.Lock()
	l.lru.Clear()
	l.m.Unlock()
}
