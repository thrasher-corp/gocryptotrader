package cache

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
)

func TestCache(t *testing.T) {
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
	if convert.InterfaceToStringOrZeroValue(v) != "world" {
		t.Fatal("expected \"hello\" key to contain value \"world\"")
	}

	if r := lruCache.Remove("hello"); !r {
		t.Fatal("expected \"hello\" key to be removed from cache")
	}

	v = lruCache.Get("hello")
	if v != nil {
		t.Fatal("expected cache to not contain \"hello\" key")
	}
}

func TestContainsOrAdd(t *testing.T) {
	lruCache := New(5)

	if lruCache.ContainsOrAdd("hello", "world") {
		t.Fatal("expected ContainsOrAdd() to add new key when not found")
	}

	if !lruCache.ContainsOrAdd("hello", "world") {
		t.Fatal("expected ContainsOrAdd() to return true when key found")
	}
}

func TestClear(t *testing.T) {
	lruCache := New(5)
	for x := range 5 {
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

func TestAdd(t *testing.T) {
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
	if convert.InterfaceToIntOrZeroValue(v) != 2 {
		t.Fatal("expected \"2\" key to contain value \"2\"")
	}
	k, v := lruCache.getNewest()
	if convert.InterfaceToIntOrZeroValue(k) != 2 {
		t.Fatal("expected latest key to be 2")
	}
	if convert.InterfaceToIntOrZeroValue(v) != 2 {
		t.Fatal("expected latest value to be 2")
	}
	lruCache.Add(3, 3)
	k, _ = lruCache.getNewest()
	if convert.InterfaceToIntOrZeroValue(k) != 3 {
		t.Fatal("expected latest key to be 3")
	}
	k, _ = lruCache.getOldest()
	if convert.InterfaceToIntOrZeroValue(k) != 2 {
		t.Fatal("expected oldest key to be 2")
	}
	k, v = lruCache.getOldest()
	if convert.InterfaceToIntOrZeroValue(k) != 2 {
		t.Fatal("expected oldest key to be 2")
	}
	if convert.InterfaceToIntOrZeroValue(v) != 2 {
		t.Fatal("expected latest value to be 2")
	}
	lruCache.Add(2, 2)
	k, _ = lruCache.getNewest()
	if convert.InterfaceToIntOrZeroValue(k) != 2 {
		t.Fatal("expected latest key to be 2")
	}
	k, _ = lruCache.getOldest()
	if convert.InterfaceToIntOrZeroValue(k) != 3 {
		t.Fatal("expected oldest key to be 3")
	}
}

func TestRemove(t *testing.T) {
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

func TestGetNewest(t *testing.T) {
	lruCache := New(2)
	if k, _ := lruCache.getNewest(); k != nil {
		t.Fatal("expected GetNewest() on empty cache to return nil")
	}
}

func TestGetOldest(t *testing.T) {
	lruCache := New(2)
	if k, _ := lruCache.getOldest(); k != nil {
		t.Fatal("expected GetOldest() on empty cache to return nil")
	}
}
