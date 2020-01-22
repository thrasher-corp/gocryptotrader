package cache

import (
	"fmt"
	"testing"
)

func TestCache(t *testing.T) {
	lruCache := New(5)
	lruCache.Add("hello", "world")
	v, f := lruCache.Get("hello")
	if f {
		fmt.Println(v)
	}
}
