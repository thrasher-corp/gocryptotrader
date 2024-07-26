package stream

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatch(t *testing.T) {
	t.Parallel()
	load := []byte("42")
	assert.False(t, new(Match).IncomingWithData("hello", load), "Should not match an uninitilized Match")

	match := NewMatch()
	assert.False(t, match.IncomingWithData("hello", load), "Should not match an empty signature")

	_, err := match.Set("hello", -0)
	require.ErrorIs(t, err, errBufferShouldBeGreaterThanZero, "Should error on buffer size less than 0")
	ch, err := match.Set("hello", 2)
	require.NoError(t, err, "Set must not error")
	assert.True(t, match.IncomingWithData("hello", []byte("hello")))
	assert.Equal(t, "hello", string(<-ch))

	_, err = match.Set("hello", 1)
	assert.ErrorIs(t, err, errSignatureCollision, "Should error on signature collision")

	assert.True(t, match.IncomingWithData("hello", load), "Should match with matching message and signature")
	assert.True(t, match.IncomingWithData("hello", load), "Should match with matching message and signature")

	assert.Len(t, ch, 2, "Channel should have 2 items")
}
