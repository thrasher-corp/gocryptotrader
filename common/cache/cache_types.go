package cache

import (
	"container/list"
	"sync"
)

// LRUCache thread safe fixed size LRU cache
type LRUCache struct {
	lru *LRU
	m   sync.RWMutex
}

// LRU non-thread safe fixed size LRU cache
type LRU struct {
	Cap   uint64
	List  *list.List
	Items map[interface{}]*list.Element
}

// item is
type item struct {
	key   interface{}
	value interface{}
}
