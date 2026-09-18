package cache

import (
	"testing"

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
	for x := range 5 {
		lruCache.Add(x, x)
	}
	require.Equal(t, uint64(5), lruCache.Len(), "Len must report every added entry")
	lruCache.Clear()
	assert.Zero(t, lruCache.Len(), "Len should be zero after Clear")
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
