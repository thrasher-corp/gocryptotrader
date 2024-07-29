package stream

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatch(t *testing.T) {
	t.Parallel()
	nm := NewMatch()
	require.False(t, nm.Incoming("wow"))

	// try to match with unset signature
	require.False(t, nm.Incoming("hello"))

	ch, err := nm.Set("hello")
	require.NoError(t, err)

	_, err = nm.Set("hello")
	require.ErrorIs(t, err, errSignatureCollision)

	// try and match with initial payload
	require.True(t, nm.Incoming("hello"))
	require.Nil(t, <-ch)

	// put in secondary payload with conflicting signature
	require.False(t, nm.Incoming("hello"))

	ch, err = nm.Set("hello")
	require.NoError(t, err)

	expected := []byte("payload")
	require.True(t, nm.IncomingWithData("hello", expected))

	require.Equal(t, expected, <-ch)

	_, err = nm.Set("purge me")
	require.NoError(t, err)
	nm.RemoveSignature("purge me")
	require.False(t, nm.IncomingWithData("purge me", expected))
}
