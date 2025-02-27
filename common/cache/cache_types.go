package cache

import (
	"container/list"
	"sync"
)

// LRUCache thread safe fixed size LRU cache
type LRUCache struct {
	lru *LRU
	m   sync.Mutex
}

// LRU non-thread safe fixed size LRU cache
type LRU struct {
	Cap   uint64
	l     *list.List
	items map[any]*list.Element
}

// item holds key/value for the cache
type item struct {
	key   any
	value any
}
