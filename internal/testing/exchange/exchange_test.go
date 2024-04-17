package exchange

import (
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

// TestTestInstance exercises TestInstance
func TestTestInstance(t *testing.T) {
	b := new(binance.Binance)
	require.NoError(t, TestInstance(b), "TestInstance must not error")
	assert.NotNil(t, b.Websocket, "TestInstance should set up a websocket")

	e := new(sharedtestvalues.CustomEx)
	require.ErrorIs(t, TestInstance(e), config.ErrExchangeNotFound, "TestInstance error correctly on a missing exchange")
}

// TestMockHTTPInstance exercises MockHTTPInstance
func TestMockHTTPInstance(t *testing.T) {
	b := new(binance.Binance)
	require.NoError(t, TestInstance(b), "TestInstance must not error")
	require.NoError(t, MockHTTPInstance(b), "MockHTTPInstance must not error")
}

// TestMockWsInstance exercises MockWsInstance
func TestMockWsInstance(t *testing.T) {
	b := MockWsInstance[binance.Binance](t, CurryWsMockUpgrader(t, func(_ []byte, _ *websocket.Conn) error { return nil }))
	require.NotNil(t, b, "MockWsInstance must not be nil")
}
