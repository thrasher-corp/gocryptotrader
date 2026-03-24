package poloniex

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

func TestSetupWsSupportsMultiConnectionManagement(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
	}))
	t.Cleanup(server.Close)

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	err := e.Websocket.SetAllConnectionURLs(wsURL)
	require.NoError(t, err, "SetAllConnectionURLs must not error for Poloniex")

	testexch.SetupWs(t, e)
	t.Cleanup(func() {
		if e.Websocket.IsConnected() {
			assert.NoError(t, e.Websocket.Shutdown(), "Websocket shutdown should not error")
		}
	})

	assert.Empty(t, e.Features.Subscriptions, "Features.Subscriptions should be cleared by SetupWs")
	assert.True(t, e.Websocket.IsConnected(), "Websocket manager should be connected after SetupWs")

	for _, messageFilter := range []string{connSpotPublic, connSpotPrivate, connFuturesPublic, connFuturesPrivate} {
		conn, connErr := e.Websocket.GetConnection(messageFilter)
		require.NoErrorf(t, connErr, "GetConnection must not error for message filter %s", messageFilter)
		assert.Equalf(t, wsURL, conn.GetURL(), "Connection URL should be redirected for message filter %s", messageFilter)
		assert.Emptyf(t, conn.Subscriptions().List(), "Connection subscriptions should remain empty for message filter %s", messageFilter)
	}
}
