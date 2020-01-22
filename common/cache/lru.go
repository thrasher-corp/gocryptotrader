package cache

import "container/list"

// NewLRUCache returns a new non-thread-safe LRU cache with input capacity
func NewLRUCache(cap uint64) *LRU {
	return &LRU{
		Cap:   cap,
		List:  list.New(),
		Items: make(map[interface{}]*list.Element),
	}
}

// Add adds a value to the cache
func (l *LRU) Add(key, value interface{}) {
	if f, o := l.Items[key]; o {
		l.List.MoveToFront(f)
		f.Value.(*item).value = value
		return
	}

	newItem := &item{key, value}
	itemList := l.List.PushFront(newItem)
	l.Items[key] = itemList

	if l.Len() > l.Cap {
		l.removeOldestEntry()
	}
}

// Remove removes key from the cache, if the key was removed.
func (l *LRU) Remove(key interface{}) bool {
	if i, f := l.Items[key]; f {
		l.removeElement(i)
		return true
	}
	return false
}

// Get returns key's value from cache if found
func (l *LRU) Get(key interface{}) (interface{}, bool) {
	if i, f := l.Items[key]; f {
		l.List.MoveToFront(i)

		if i.Value.(*item) == nil {
			return nil, false
		}

		return i.Value.(*item).value, true
	}
	return nil, false
}

// Clear is used to completely clear the cache.
func (l *LRU) Clear() {
	for x := range l.Items {
		delete(l.Items, l.Items[x])
	}
	l.List.Init()
}

// Len returns length of List
func (l *LRU) Len() uint64 {
	return uint64(l.List.Len())
}

// removeOldest removes the oldest item from the cache.
func (l *LRU) removeOldestEntry() {
	i := l.List.Back()
	if i != nil {
		l.removeElement(i)
	}
}

// removeElement element from the cache
func (l *LRU) removeElement(e *list.Element) {
	l.List.Remove(e)
	delete(l.Items, e.Value.(*item))
}
