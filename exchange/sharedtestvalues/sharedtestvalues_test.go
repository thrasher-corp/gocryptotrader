package sharedtestvalues

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestWebsocket(t *testing.T) {
	t.Parallel()
	w := NewTestWebsocket()
	require.NotNil(t, w, "NewTestWebsocket must return a manager")
	require.NotNil(t, w.DataHandler, "NewTestWebsocket must give the manager a DataHandler")
	assert.Equal(t, WebsocketChannelOverrideCapacity, cap(w.DataHandler.C), "NewTestWebsocket should give the DataHandler the test capacity")
}

// BenchmarkNewTestWebsocket guards against the manager allocating a default DataHandler that NewTestWebsocket discards,
// which every test exchange set up through testexch.Setup pays for
func BenchmarkNewTestWebsocket(b *testing.B) {
	for b.Loop() {
		_ = NewTestWebsocket()
	}
}
