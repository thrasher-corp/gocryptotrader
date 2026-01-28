/*
	LRU Cache package

	Based off information obtained from:

	https://girai.dev/blog/lru-cache-implementation-in-go/
	https://en.wikipedia.org/wiki/Cache_replacement_policies#Least_recently_used_(LRU)
*/

package cache

import "container/list"

// NewLRUCache returns a new non-concurrent-safe LRU cache with input capacity
func NewLRUCache(capacity uint64) *LRU {
	return &LRU{
		Cap:   capacity,
		l:     list.New(),
		items: make(map[any]*list.Element),
	}
}

// Add adds a value to the cache
func (l *LRU) Add(key, value any) {
	if f, o := l.items[key]; o {
		l.l.MoveToFront(f)
		if v, ok := f.Value.(*item); ok {
			v.value = value
		}
		return
	}

	newItem := &item{key, value}
	itemList := l.l.PushFront(newItem)
	l.items[key] = itemList
	if l.Len() > l.Cap {
		l.removeOldestEntry()
	}
}

// Get returns keys value from cache if found
func (l *LRU) Get(key any) any {
	if i, f := l.items[key]; f {
		l.l.MoveToFront(i)
		if v, ok := i.Value.(*item); ok {
			return v.value
		}
	}
	return nil
}

// getOldest returns the oldest entry
func (l *LRU) getOldest() (key, value any) {
	if x := l.l.Back(); x != nil {
		if v, ok := x.Value.(*item); ok {
			return v.key, v.value
		}
	}
	return
}

// getNewest returns the newest entry
func (l *LRU) getNewest() (key, value any) {
	if x := l.l.Front(); x != nil {
		if v, ok := x.Value.(*item); ok {
			return v.key, v.value
		}
	}
	return
}

// Contains check if key is in cache this does not update LRU
func (l *LRU) Contains(key any) (f bool) {
	_, f = l.items[key]
	return
}

// Remove removes key from the cache, if the key was removed.
func (l *LRU) Remove(key any) bool {
	if i, f := l.items[key]; f {
		l.removeElement(i)
		return true
	}
	return false
}

// Clear is used to completely clear the cache.
func (l *LRU) Clear() {
	for x := range l.items {
		delete(l.items, l.items[x])
	}
	l.l.Init()
}

// Len returns length of l
func (l *LRU) Len() uint64 {
	return uint64(l.l.Len()) //nolint:gosec // False positive as uint64 (2^64-1) can support both 2^31-1 on 32bit systems and 2^63-1 on 64bit systems
}

// removeOldestEntry removes the oldest item from the cache.
func (l *LRU) removeOldestEntry() {
	if i := l.l.Back(); i != nil {
		l.removeElement(i)
	}
}

// removeElement element from the cache
func (l *LRU) removeElement(e *list.Element) {
	l.l.Remove(e)
	if v, ok := e.Value.(*item); ok {
		delete(l.items, v.key)
	}
}
