package cache

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCache(t *testing.T) {
	t.Parallel()
	lruCache := New(5)
	lruCache.Add("hello", "world")
	require.True(t, lruCache.Contains("hello"), "Contains must return true for a stored key")

	assert.Equal(t, "world", lruCache.Get("hello"), "Get should return the stored value")
	assert.True(t, lruCache.Remove("hello"), "Remove should return true for a stored key")
	assert.Nil(t, lruCache.Get("hello"), "Get should return nil after the key is removed")
}

func TestContainsOrAdd(t *testing.T) {
	t.Parallel()
	lruCache := New(5)
	assert.False(t, lruCache.ContainsOrAdd("hello", "world"), "ContainsOrAdd should return false and add the key when not found")
	assert.True(t, lruCache.ContainsOrAdd("hello", "world"), "ContainsOrAdd should return true when the key is found")
}

func TestClear(t *testing.T) {
	t.Parallel()
	lruCache := New(5)
	lruCache.Add("a", 1)
	lruCache.Add("b", 2)
	require.Equal(t, uint64(2), lruCache.Len(), "Len must report every added entry")

	lruCache.Clear()
	assert.Zero(t, lruCache.Len(), "Len should be zero after Clear")
	assert.False(t, lruCache.Contains("a"), "Contains should return false for a cleared key")
	assert.False(t, lruCache.Contains("b"), "Contains should return false for a cleared key")
	assert.Nil(t, lruCache.Get("a"), "Get should return nil for a cleared key")

	lruCache.Add("c", 3)
	assert.Equal(t, 3, lruCache.Get("c"), "Get should return a value added after Clear")
}

func TestClearFullCache(t *testing.T) {
	t.Parallel()
	lruCache := New(5)
	for x := range 5 {
		lruCache.Add(x, x)
	}
	require.Equal(t, uint64(5), lruCache.Len(), "Len must report every added entry")

	lruCache.Clear()
	assert.Zero(t, lruCache.Len(), "Len should be zero after Clear")
	for x := range 5 {
		assert.Falsef(t, lruCache.Contains(x), "Contains should return false for cleared key %d", x)
	}

	for x := range 5 {
		lruCache.Add(x, x)
	}
	for x := range 5 {
		assert.Equalf(t, x, lruCache.Get(x), "Get should return the value re-added after Clear for key %d", x)
	}
}

func TestAdd(t *testing.T) {
	t.Parallel()
	lruCache := New(2)
	lruCache.Add(1, 1)
	lruCache.Add(2, 2)
	require.Equal(t, uint64(2), lruCache.Len(), "Len must match the number of added entries")

	lruCache.Add(3, 3)
	require.Equal(t, uint64(2), lruCache.Len(), "Len must not exceed capacity")

	assert.Nil(t, lruCache.Get(1), "Get should evict the oldest key")
	assert.Equal(t, 2, lruCache.Get(2), "Get should return the retained value")

	k, v := lruCache.getNewest()
	assert.Equal(t, 2, k, "getNewest should return the most recently used key")
	assert.Equal(t, 2, v, "getNewest should return the most recently used value")

	lruCache.Add(3, 3)
	k, _ = lruCache.getNewest()
	assert.Equal(t, 3, k, "getNewest should return the freshly added key")

	k, v = lruCache.getOldest()
	assert.Equal(t, 2, k, "getOldest should return the least recently used key")
	assert.Equal(t, 2, v, "getOldest should return the least recently used value")

	lruCache.Add(2, 2)
	k, _ = lruCache.getNewest()
	assert.Equal(t, 2, k, "getNewest should return the re-added key")
	k, _ = lruCache.getOldest()
	assert.Equal(t, 3, k, "getOldest should return the displaced key")
}

func TestRemove(t *testing.T) {
	t.Parallel()
	lruCache := New(2)
	lruCache.Add(1, 1)
	assert.True(t, lruCache.Remove(1), "Remove should return true for a valid key")
	assert.False(t, lruCache.Remove(2), "Remove should return false for an invalid key")
}

func TestGetNewest(t *testing.T) {
	t.Parallel()
	k, _ := New(2).getNewest()
	assert.Nil(t, k, "getNewest should return nil on an empty cache")
}

func TestGetOldest(t *testing.T) {
	t.Parallel()
	k, _ := New(2).getOldest()
	assert.Nil(t, k, "getOldest should return nil on an empty cache")
}

func TestAdd_MutexReleaseOnPanic(t *testing.T) {
	t.Parallel()
	lruCache := New(2)
	require.Panics(t, func() { lruCache.Add([]int{1}, "val") }, "Add must panic on an unhashable key")
	require.True(t, lruCache.m.TryLock(), "Add must release the mutex when it panics")
	lruCache.m.Unlock()
	lruCache.Add("key", "val")
	assert.Equal(t, "val", lruCache.Get("key"), "Get should return the value added after the panic")
}

func TestSetCapacity(t *testing.T) {
	t.Parallel()
	t.Run("preserve recent entries when shrinking and growing", func(t *testing.T) {
		t.Parallel()
		c := New(3)
		c.Add("a", 1)
		c.Add("b", 2)
		c.Add("c", 3)
		require.Equal(t, 1, c.Get("a"), "Get must promote the oldest entry")
		c.SetCapacity(2)
		assert.False(t, c.Contains("b"), "shrinking should evict the least recently used entry")
		assert.Equal(t, uint64(2), c.Len(), "shrinking should honour the new capacity")
		c.SetCapacity(4)
		c.Add("d", 4)
		c.Add("e", 5)
		assert.Equal(t, 1, c.Get("a"), "growing should retain previous entries")
		assert.Equal(t, uint64(4), c.Len(), "growing should admit more entries")
		c.SetCapacity(0)
		assert.Zero(t, c.Len(), "zero capacity should empty the cache")
		c.Add("f", 6)
		assert.Nil(t, c.Get("f"), "zero capacity should not retain new entries")
		c.SetCapacity(1)
		c.Add("g", 7)
		assert.Equal(t, 7, c.Get("g"), "growing from zero should restore caching")
	})
	t.Run("shrink several entries to a smaller positive capacity", func(t *testing.T) {
		t.Parallel()
		c := New(5)
		for i, k := range []string{"a", "b", "c", "d", "e"} {
			c.Add(k, i)
		}
		require.Equal(t, 0, c.Get("a"), "Get must promote the oldest entry")
		c.SetCapacity(2)
		require.Equal(t, uint64(2), c.Len(), "shrinking must evict entries down to the new capacity")
		for _, k := range []string{"b", "c", "d"} {
			assert.Falsef(t, c.Contains(k), "shrinking should evict least recently used entry %s", k)
		}
		c.Add("f", 5)
		assert.False(t, c.Contains("e"), "the next Add should evict the least recently used survivor")
		assert.True(t, c.Contains("a"), "the next Add should retain the most recently used survivor")
		assert.True(t, c.Contains("f"), "the next Add should retain the new entry")
	})
	t.Run("resize an empty cache", func(t *testing.T) {
		t.Parallel()
		c := New(3)
		c.SetCapacity(2)
		require.Zero(t, c.Len(), "resizing an empty cache must leave it empty")
		c.Add("a", 1)
		c.Add("b", 2)
		require.Equal(t, uint64(2), c.Len(), "Len must report entries up to the new capacity")
		c.Add("c", 3)
		assert.False(t, c.Contains("a"), "the next Add should evict the oldest entry at the new capacity")
		assert.True(t, c.Contains("b"), "the next Add should retain the newer entry")
		assert.True(t, c.Contains("c"), "the next Add should retain the new entry")
		assert.Equal(t, uint64(2), c.Len(), "Len should not exceed the new capacity")
	})
	t.Run("concurrent reads writes and capacity changes", func(t *testing.T) {
		t.Parallel()
		c := New(3)
		var workers sync.WaitGroup
		for worker := range 4 {
			workers.Go(func() {
				for i := range 100 {
					c.Add(worker, i)
					c.Get(worker)
					c.SetCapacity(uint64(i % 4))
				}
			})
		}
		// Wait with a deadline so a leaked lock fails this test instead of hanging the package. The repeated resize
		// covers an unchanged capacity on every run, and the length is sent back so t is not used after a timeout.
		finalLen := make(chan uint64, 1)
		go func() {
			workers.Wait()
			c.SetCapacity(1)
			c.SetCapacity(1)
			finalLen <- c.Len()
		}()
		select {
		case n := <-finalLen:
			assert.LessOrEqual(t, n, uint64(1), "final resize should honour the capacity")
		case <-time.After(5 * time.Second):
			require.FailNow(t, "workers and the final resize must release the cache lock within 5 seconds")
		}
	})
}
