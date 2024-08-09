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

	m, err := match.Set("hello", 2)
	require.NoError(t, err, "Set must not error")
	assert.Equal(t, "hello", m.sig)

	_, err = match.Set("hello", 1)
	assert.ErrorIs(t, err, errSigCollision, "Should error on signature collision")

	assert.True(t, match.IncomingWithData("hello", load), "Should match with matching message and signature")
	assert.True(t, match.IncomingWithData("hello", load), "Should match with matching message and signature")

	assert.Len(t, m.C, 2, "Channel should have 2 items")

	m.Cleanup()
}
