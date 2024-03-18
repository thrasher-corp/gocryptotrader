package exchange

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mock"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// TestInstance takes an empty exchange instance and loads config for it from testdata/configtest and connects a NewTestWebsocket
func TestInstance(e exchange.IBotExchange) error {
	cfg := &config.Config{}
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		return fmt.Errorf("LoadConfig() error: %w", err)
	}
	parts := strings.Split(fmt.Sprintf("%T", e), ".")
	if len(parts) != 2 {
		return errors.New("unexpected parts splitting exchange type name")
	}
	eName := parts[1]
	exchConf, err := cfg.GetExchangeConfig(eName)
	if err != nil {
		return fmt.Errorf("GetExchangeConfig(`%s`) error: %w", eName, err)
	}
	e.SetDefaults()
	b := e.GetBase()
	b.Websocket = sharedtestvalues.NewTestWebsocket()
	err = e.Setup(exchConf)
	if err != nil {
		return fmt.Errorf("Setup() error: %w", err)
	}
	return nil
}

// httpMockFile is a consistent path under each exchange to find the mock server definitions
const httpMockFile = "testdata/http.json"

// MockHTTPInstance takes an existing Exchange instance and attaches it to a new http server
// It is expected to be run once,  since http requests do not often tangle with each other
func MockHTTPInstance(e exchange.IBotExchange) error {
	serverDetails, newClient, err := mock.NewVCRServer(httpMockFile)
	if err != nil {
		return fmt.Errorf("mock server error %s", err)
	}
	b := e.GetBase()
	b.SkipAuthCheck = true
	err = b.SetHTTPClient(newClient)
	if err != nil {
		return fmt.Errorf("mock server error %s", err)
	}
	endpointMap := b.API.Endpoints.GetURLMap()
	for k := range endpointMap {
		err = b.API.Endpoints.SetRunning(k, serverDetails)
		if err != nil {
			return fmt.Errorf("mock server error %s", err)
		}
	}
	log.Printf(sharedtestvalues.MockTesting, e.GetName())

	return nil
}

var upgrader = websocket.Upgrader{}

type wsMockFunc func(msg []byte, w *websocket.Conn) error

// MockWSInstance creates a new Exchange instance with a mock WS instance and HTTP server
// It accepts an exchange package type argument and a mock WS function
// It is expected to be run from any WS tests which need a specific response function
func MockWSInstance[T any, PT interface {
	*T
	exchange.IBotExchange
}](tb testing.TB, m wsMockFunc) *T {
	tb.Helper()

	e := PT(new(T))
	require.NoError(tb, TestInstance(e), "TestInstance setup should not error")

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { wsMockWrapper(tb, w, r, m) }))

	b := e.GetBase()
	b.SkipAuthCheck = true
	err := b.API.Endpoints.SetRunning("RestSpotURL", s.URL)
	require.NoError(tb, err, "Endpoints.SetRunning should not error for RestSpotURL")
	for _, auth := range []bool{true, false} {
		err = b.Websocket.SetWebsocketURL("ws"+strings.TrimPrefix(s.URL, "http"), auth, true)
		require.NoErrorf(tb, err, "SetWebsocketURL should not error for auth: %v", auth)
	}

	b.Features.Subscriptions = []*subscription.Subscription{}
	err = b.Websocket.Connect()
	require.NoError(tb, err, "Connect should not error")

	return e
}

// wsMockWrapper handles upgrading an initial HTTP request to WS, and then runs a for loop calling the mock func on each input
func wsMockWrapper(tb testing.TB, w http.ResponseWriter, r *http.Request, m wsMockFunc) {
	tb.Helper()
	// TODO: This needs to move once this branch includes #1358, probably to use a new mock HTTP instance for kraken
	if strings.Contains(r.URL.Path, "GetWebSocketsToken") {
		_, err := w.Write([]byte(`{"result":{"token":"mockAuth"}}`))
		require.NoError(tb, err, "Write should not error")
		return
	}
	c, err := upgrader.Upgrade(w, r, nil)
	require.NoError(tb, err, "Upgrade connection should not error")
	defer c.Close()
	for {
		_, p, err := c.ReadMessage()
		require.NoError(tb, err, "ReadMessage should not error")

		err = m(p, c)
		assert.NoError(tb, err, "WS Mock Function should not error")
	}
}

var setupWsMutex sync.Mutex
var setupWsOnce = make(map[exchange.IBotExchange]bool)

// SetupWs is a helper function to connect both auth and normal websockets
// It will skip the test if websockets are not enabled
// It's up to the test to skip if it requires creds, though
func SetupWs(tb testing.TB, e exchange.IBotExchange) {
	tb.Helper()

	setupWsMutex.Lock()
	defer setupWsMutex.Unlock()

	if setupWsOnce[e] {
		return
	}

	b := e.GetBase()
	if !b.Websocket.IsEnabled() {
		tb.Skip("Websocket not enabled")
	}
	if b.Websocket.IsConnected() {
		return
	}
	err := b.Websocket.Connect()
	require.NoError(tb, err, "WsConnect should not error")

	setupWsOnce[e] = true
}

var updatePairsMutex sync.Mutex
var updatePairsOnce = make(map[exchange.IBotExchange]bool)

// UpdatePairsOnce ensures pairs are only updated once in parallel tests
func UpdatePairsOnce(tb testing.TB, e exchange.IBotExchange) {
	tb.Helper()

	updatePairsMutex.Lock()
	defer updatePairsMutex.Unlock()

	if updatePairsOnce[e] {
		return
	}

	err := e.UpdateTradablePairs(context.Background(), true)
	require.NoError(tb, err, "UpdateTradablePairs must not error")

	updatePairsOnce[e] = true
}
