package connchecker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/internal/testing/livetest"
)

func TestConnection(t *testing.T) {
	_, err := New([]string{"faultyIP"}, nil, time.Second)
	require.Error(t, err, "New must reject a DNS entry that isn't an address")

	c, err := New([]string{"127.0.0.1"}, []string{"localhost"}, time.Second)
	require.NoError(t, err, "New must not error")
	assert.True(t, c.IsConnected(), "a resolvable loopback address and name should count as connected")
	c.Shutdown()

	t.Run("live", func(t *testing.T) {
		if livetest.Mock {
			t.Skip("queries public DNS; build with the mock_test_off tag to run it")
		}
		_, err := New(DefaultDNSList, nil, time.Second)
		require.NoError(t, err, "New must not error with the default DNS list")

		_, err = New(nil, []string{"faultyHost"}, time.Second)
		require.NoError(t, err, "New must not error for a host that doesn't resolve")

		c, err := New(nil, nil, 0)
		require.NoError(t, err, "New must not error with the defaults")
		if !c.IsConnected() {
			t.Log("Test - No internet connection found")
		} else {
			t.Log("Test - Internet connection found")
		}
		c.Shutdown()
	})
}
