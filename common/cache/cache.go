package cache

// New returns a new thread-safe LRU cache with input capacity
func New(capacity uint64) *LRUCache {
	return &LRUCache{
		lru: NewLRUCache(capacity),
	}
}

// Add new entry to Cache return true if entry removed
func (l *LRUCache) Add(k, v interface{}) {
	l.m.Lock()
	l.lru.Add(k, v)
	l.m.Unlock()
}

// Get looks up a key's value from the cache.
func (l *LRUCache) Get(key interface{}) (value interface{}) {
	l.m.Lock()
	defer l.m.Unlock()
	return l.lru.Get(key)
}

// GetOldest looks up old key's value from the cache.
func (l *LRUCache) GetOldest() (key, value interface{}) {
	l.m.Lock()
	defer l.m.Unlock()
	return l.lru.GetOldest()
}

// Get looks up a key's value from the cache.
func (l *LRUCache) GetNewest() (key, value interface{}) {
	l.m.Lock()
	defer l.m.Unlock()
	return l.lru.GetNewest()
}

// Contain checks if cache contains key if not adds to cache
func (l *LRUCache) ContainsOrAdd(key, value interface{}) bool {
	l.m.Lock()
	defer l.m.Unlock()
	if l.lru.Contains(key) {
		return true
	}
	l.lru.Add(key, value)
	return false
}

// Contain checks if cache contains key
func (l *LRUCache) Contains(key interface{}) bool {
	l.m.Lock()
	defer l.m.Unlock()
	return l.lru.Contains(key)
}

func (l *LRUCache) Remove(key interface{}) bool {
	l.m.Lock()
	defer l.m.Unlock()
	return l.lru.Remove(key)
}

// Clear is used to clear the cache.
func (l *LRUCache) Clear() {
	l.m.Lock()
	l.lru.Clear()
	l.m.Unlock()
}

// Len returns the number of items in the cache.
func (l *LRUCache) Len() uint64 {
	l.m.Lock()
	defer l.m.Unlock()
	return l.lru.Len()
}
