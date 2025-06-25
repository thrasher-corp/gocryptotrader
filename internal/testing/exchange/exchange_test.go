package exchange

import (
	"testing"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

// TestSetup exercises Setup
func TestSetup(t *testing.T) {
	b := new(binance.Exchange)
	require.NoError(t, Setup(b), "Setup must not error")
	assert.NotNil(t, b.Websocket, "Websocket should not be nil after Setup")

	e := new(sharedtestvalues.CustomEx)
	assert.ErrorIs(t, Setup(e), config.ErrExchangeNotFound, "Setup should error correctly on a missing exchange")
}

// TestMockHTTPInstance exercises MockHTTPInstance
func TestMockHTTPInstance(t *testing.T) {
	b := new(binance.Exchange)
	require.NoError(t, Setup(b), "Test exchange Setup must not error")
	require.NoError(t, MockHTTPInstance(b), "MockHTTPInstance with no optional path must not error")
	require.NoError(t, MockHTTPInstance(b, "api"), "MockHTTPInstance with optional path must not error")
}

// TestMockWsInstance exercises MockWsInstance
func TestMockWsInstance(t *testing.T) {
	b := MockWsInstance[binance.Exchange](t, mockws.CurryWsMockUpgrader(t, func(_ testing.TB, _ []byte, _ *gws.Conn) error { return nil }))
	require.NotNil(t, b, "MockWsInstance must not be nil")
}
