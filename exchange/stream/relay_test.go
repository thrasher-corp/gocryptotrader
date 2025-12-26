package stream

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRelay(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() { NewRelay(0) }, "buffer size should be greater than 0")
	r := NewRelay(5)
	require.NotNil(t, r)
	assert.Equal(t, 5, cap(r.comm))
}

func TestSend(t *testing.T) {
	t.Parallel()
	r := NewRelay(1)
	require.NotNil(t, r)
	assert.NoError(t, r.Send(t.Context(), "test"))
	assert.ErrorIs(t, r.Send(t.Context(), "overflow"), errChannelBufferFull)
}

func TestRead(t *testing.T) {
	t.Parallel()
	r := NewRelay(1)
	require.NotNil(t, r)
	require.Empty(t, r.C)
	assert.NoError(t, r.Send(t.Context(), "test"))
	require.Len(t, r.C, 1)
	assert.Equal(t, "test", (<-r.C).Data)
}

func TestClose(t *testing.T) {
	t.Parallel()
	r := NewRelay(1)
	require.NotNil(t, r)
	r.Close()
	_, ok := <-r.C
	assert.False(t, ok)
}
