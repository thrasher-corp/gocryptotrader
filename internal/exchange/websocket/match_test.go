package websocket

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatch(t *testing.T) {
	t.Parallel()
	load := []byte("42")
	assert.False(t, new(Match).IncomingWithData("hello", load), "Should not match an uninitialised Match")

	match := NewMatch()
	assert.False(t, match.IncomingWithData("hello", load), "Should not match an empty signature")

	_, err := match.Set("hello", 0)
	require.ErrorIs(t, err, errInvalidBufferSize, "Must error on zero buffer size")
	_, err = match.Set("hello", -1)
	require.ErrorIs(t, err, errInvalidBufferSize, "Must error on negative buffer size")
	ch, err := match.Set("hello", 2)
	require.NoError(t, err, "Set must not error")
	assert.True(t, match.IncomingWithData("hello", []byte("hello")))
	assert.Equal(t, "hello", string(<-ch))

	_, err = match.Set("hello", 2)
	assert.ErrorIs(t, err, errSignatureCollision, "Should error on signature collision")

	assert.True(t, match.IncomingWithData("hello", load), "Should match with matching message and signature")
	assert.False(t, match.IncomingWithData("hello", load), "Should not match with matching message and signature")

	assert.Len(t, ch, 1, "Channel should have 1 items, 1 was already read above")
}

func TestRemoveSignature(t *testing.T) {
	t.Parallel()
	match := NewMatch()
	ch, err := match.Set("masterblaster", 1)
	select {
	case <-ch:
		t.Fatal("Should not be able to read from an empty channel")
	default:
	}
	require.NoError(t, err)
	match.RemoveSignature("masterblaster")
	select {
	case garbage := <-ch:
		require.Empty(t, garbage)
	default:
		t.Fatal("Should be able to read from a closed channel")
	}
}

func TestRequireMatchWithData(t *testing.T) {
	t.Parallel()
	match := NewMatch()
	err := match.RequireMatchWithData("hello", []byte("world"))
	require.ErrorIs(t, err, ErrSignatureNotMatched, "Must error on unmatched signature")
	assert.Contains(t, err.Error(), "world", "Should contain the data in the error message")
	assert.Contains(t, err.Error(), "hello", "Should contain the signature in the error message")

	ch, err := match.Set("hello", 1)
	require.NoError(t, err, "Set must not error")
	err = match.RequireMatchWithData("hello", []byte("world"))
	require.NoError(t, err, "Must not error on matched signature")
	assert.Equal(t, "world", string(<-ch))
}
