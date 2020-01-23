package cache

import (
	"testing"
)

func TestNewLRUCache(t *testing.T) {
	lruCache := New(5)
	lruCache.Add("hello", "world")
	c := lruCache.Contains("hello")
	if !c {
		t.Fatal("expected cache to contain \"hello\" key")
	}

	v := lruCache.Get("hello")
	if v == nil {
		t.Fatal("expected cache to contain \"hello\" key")
	}
	if v.(string) != "world" {
		t.Fatal("expected \"hello\" key to contain value \"world\"")
	}

	r := lruCache.Remove("hello")
	if !r {
		t.Fatal("expected \"hello\" key to be removed from cache")
	}

	v = lruCache.Get("hello")
	if v != nil {
		t.Fatal("expected cache to not contain \"hello\" key")
	}
}

func TestNewLRUCache_ContainsOrAdd(t *testing.T) {
	lruCache := New(5)

	f := lruCache.ContainsOrAdd("hello", "world")
	if f {
		t.Fatal("expected ContainsOrAdd() to add new key when not found")
	}

	f = lruCache.ContainsOrAdd("hello", "world")
	if !f {
		t.Fatal("expected ContainsOrAdd() to return true when key found")
	}
}

func TestNewLRUCache_Clear(t *testing.T) {
	lruCache := New(5)
	for x := 0; x < 5; x++ {
		lruCache.Add(x, x)
	}
	if lruCache.Len() != 5 {
		t.Fatal("expected cache to have 5 entries")
	}
	lruCache.Clear()
	if lruCache.Len() != 0 {
		t.Fatal("expected cache to have 0 entries")
	}
}

func TestLRUCache_Add(t *testing.T) {
	lruCache := New(2)
	lruCache.Add(1, 1)
	lruCache.Add(2, 2)
	if lruCache.Len() != 2 {
		t.Fatal("expected cache to have 2 entries")
	}
	lruCache.Add(3, 3)
	if lruCache.Len() != 2 {
		t.Fatal("expected cache to have 2 entries")
	}

	v := lruCache.Get(1)
	if v != nil {
		t.Fatal("expected cache to no longer contain \"1\" key")
	}
	v = lruCache.Get(2)
	if v == nil {
		t.Fatal("expected cache to contain \"2\" key")
	}
	if v.(int) != 2 {
		t.Fatal("expected \"2\" key to contain value \"2\"")
	}
	lruCache.Add(3, 3)
	k, _ := lruCache.GetNewest()
	if k.(int) != 3 {
		t.Fatal("expected latest key to be 3")
	}
	k, _ = lruCache.GetOldest()
	if k.(int) != 2 {
		t.Fatal("expected oldest key to be 2")
	}
	lruCache.Add(2, 2)
	k, _ = lruCache.GetNewest()
	if k.(int) != 2 {
		t.Fatal("expected latest key to be 2")
	}
	k, _ = lruCache.GetOldest()
	if k.(int) != 3 {
		t.Fatal("expected oldest key to be 3")
	}
}

func TestLRUCache_Remove(t *testing.T) {
	lruCache := New(2)
	lruCache.Add(1, 1)

	v := lruCache.Remove(1)
	if !v {
		t.Fatal("expected remove on valid key to return true")
	}
	v = lruCache.Remove(2)
	if v {
		t.Fatal("expected remove on invalid key to return false")
	}
}

func TestLRUCache_GetNewest(t *testing.T) {
	lruCache := New(2)
	k, _ := lruCache.GetNewest()
	if k != nil {
		t.Fatal("expected GetNewest() on empty cache to return nil")
	}
}

func TestLRUCache_GetOldest(t *testing.T) {
	lruCache := New(2)
	k, _ := lruCache.GetOldest()
	if k != nil {
		t.Fatal("expected GetOldest() on empty cache to return nil")
	}
}
