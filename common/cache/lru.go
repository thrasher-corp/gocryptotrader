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
		items: make(map[interface{}]*list.Element),
	}
}

// Add adds a value to the cache
func (l *LRU) Add(key, value interface{}) {
	if f, o := l.items[key]; o {
		l.l.MoveToFront(f)
		f.Value.(*item).value = value
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
func (l *LRU) Get(key interface{}) interface{} {
	if i, f := l.items[key]; f {
		l.l.MoveToFront(i)
		return i.Value.(*item).value
	}
	return nil
}

// GetOldest returns the oldest entry
func (l *LRU) getOldest() (key, value interface{}) {
	x := l.l.Back()
	if x != nil {
		return x.Value.(*item).key, x.Value.(*item).value
	}
	return
}

// GetNewest returns the newest entry
func (l *LRU) getNewest() (key, value interface{}) {
	x := l.l.Front()
	if x != nil {
		return x.Value.(*item).key, x.Value.(*item).value
	}
	return
}

// Contains check if key is in cache this does not update LRU
func (l *LRU) Contains(key interface{}) (f bool) {
	_, f = l.items[key]
	return
}

// Remove removes key from the cache, if the key was removed.
func (l *LRU) Remove(key interface{}) bool {
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
	return uint64(l.l.Len())
}

// removeOldest removes the oldest item from the cache.
func (l *LRU) removeOldestEntry() {
	i := l.l.Back()
	if i != nil {
		l.removeElement(i)
	}
}

// removeElement element from the cache
func (l *LRU) removeElement(e *list.Element) {
	l.l.Remove(e)
	delete(l.items, e.Value.(*item).key)
}
