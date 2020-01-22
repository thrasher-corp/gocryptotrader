package cache

import "container/list"

// NewLRUCache returns a new non-thread-safe LRU cache with input capacity
func NewLRUCache(capacity uint64) *LRU {
	return &LRU{
		Cap:   capacity,
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

// Get returns keys value from cache if found
func (l *LRU) Get(key interface{}) interface{} {
	if i, f := l.Items[key]; f {
		l.List.MoveToFront(i)

		return i.Value.(*item).value
	}
	return nil
}

// GetOldest returns the oldest entry
func (l *LRU) GetOldest() (key, value interface{}) {
	x := l.List.Back()
	if x != nil {
		return x.Value.(*item).key, x.Value.(*item).value
	}
	return
}

// GetNewest returns the newest entry
func (l *LRU) GetNewest() (key, value interface{}) {
	x := l.List.Front()
	if x != nil {
		return x.Value.(*item).key, x.Value.(*item).value
	}
	return
}

// Contains check if key is in cache this does not update LRU
func (l *LRU) Contains(key interface{}) (f bool) {
	_, f = l.Items[key]
	return
}

// Remove removes key from the cache, if the key was removed.
func (l *LRU) Remove(key interface{}) bool {
	if i, f := l.Items[key]; f {
		l.removeElement(i)
		return true
	}
	return false
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
	delete(l.Items, e.Value.(*item).key)
}
