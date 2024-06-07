package exchange

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mock"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testutils "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
)

// Setup takes an empty exchange instance and loads config for it from testdata/configtest and connects a NewTestWebsocket
func Setup(e exchange.IBotExchange) error {
	cfg := &config.Config{}

	root, err := testutils.RootPathFromCWD()
	if err != nil {
		return err
	}

	err = cfg.LoadConfig(filepath.Join(root, "testdata", "configtest.json"), true)
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

// WsMockFunc is a websocket handler to be called with each websocket message
type WsMockFunc func([]byte, *websocket.Conn) error

// MockWsInstance creates a new Exchange instance with a mock websocket instance and HTTP server
// It accepts an exchange package type argument and a http.HandlerFunc
// See CurryWsMockUpgrader for a convenient way to curry t and a ws mock function
// It is expected to be run from any WS tests which need a specific response function
// No default subscriptions will be run since they disrupt unit tests
func MockWsInstance[T any, PT interface {
	*T
	exchange.IBotExchange
}](tb testing.TB, h http.HandlerFunc) *T {
	tb.Helper()

	e := PT(new(T))
	require.NoError(tb, Setup(e), "Test exchange Setup must not error")

	s := httptest.NewServer(h)

	b := e.GetBase()
	b.SkipAuthCheck = true
	b.API.AuthenticatedWebsocketSupport = true
	err := b.API.Endpoints.SetRunning("RestSpotURL", s.URL)
	require.NoError(tb, err, "Endpoints.SetRunning should not error for RestSpotURL")
	for _, auth := range []bool{true, false} {
		err = b.Websocket.SetWebsocketURL("ws"+strings.TrimPrefix(s.URL, "http"), auth, true)
		require.NoErrorf(tb, err, "SetWebsocketURL should not error for auth: %v", auth)
	}

	// For testing we never want to use the default subscriptions; Tests of GenerateSubscriptions should be exercising it directly
	b.Features.Subscriptions = subscription.List{}
	// Exchanges which don't support subscription conf; Can be removed when all exchanges support sub conf
	b.Websocket.GenerateSubs = func() (subscription.List, error) { return subscription.List{}, nil }

	err = b.Websocket.Connect()
	require.NoError(tb, err, "Connect should not error")

	return e
}

// CurryWsMockUpgrader curries a WsMockUpgrader with a testing.TB and a mock func
// bridging the gap between information known before the Server is created and during a request
func CurryWsMockUpgrader(tb testing.TB, wsHandler WsMockFunc) http.HandlerFunc {
	tb.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		WsMockUpgrader(tb, w, r, wsHandler)
	}
}

// WsMockUpgrader handles upgrading an initial HTTP request to WS, and then runs a for loop calling the mock func on each input
func WsMockUpgrader(tb testing.TB, w http.ResponseWriter, r *http.Request, wsHandler WsMockFunc) {
	tb.Helper()
	c, err := upgrader.Upgrade(w, r, nil)
	require.NoError(tb, err, "Upgrade connection should not error")
	defer c.Close()
	for {
		_, p, err := c.ReadMessage()
		if websocket.IsUnexpectedCloseError(err) {
			return
		}
		require.NoError(tb, err, "ReadMessage should not error")

		err = wsHandler(p, c)
		assert.NoError(tb, err, "WS Mock Function should not error")
	}
}

// FixtureToDataHandler squirts the contents of a file to a reader function (probably e.wsHandleData)
func FixtureToDataHandler(tb testing.TB, fixturePath string, reader func([]byte) error) {
	tb.Helper()

	fixture, err := os.Open(fixturePath)
	assert.NoError(tb, err, "Opening fixture '%s' should not error", fixturePath)
	defer func() {
		assert.NoError(tb, fixture.Close(), "Closing the fixture file should not error")
	}()

	s := bufio.NewScanner(fixture)
	for s.Scan() {
		msg := s.Bytes()
		err := reader(msg)
		assert.NoErrorf(tb, err, "Fixture message should not error:\n%s", msg)
	}
	assert.NoError(tb, s.Err(), "Fixture Scanner should not error")
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
	w, err := b.GetWebsocket()
	if err != nil || !b.Websocket.IsEnabled() {
		tb.Skip("Websocket not enabled")
	}
	if w.IsConnected() {
		return
	}

	// For testing we never want to use the default subscriptions; Tests of GenerateSubscriptions should be exercising it directly
	b.Features.Subscriptions = subscription.List{}
	// Exchanges which don't support subscription conf; Can be removed when all exchanges support sub conf
	w.GenerateSubs = func() (subscription.List, error) { return subscription.List{}, nil }

	err = w.Connect()
	require.NoError(tb, err, "WsConnect should not error")

	setupWsOnce[e] = true
}

var updatePairsMutex sync.Mutex
var updatePairsOnce = make(map[string]*currency.PairsManager)

// UpdatePairsOnce ensures pairs are only updated once in parallel tests
// A clone of the cache of the updated pairs is used to populate duplicate requests
func UpdatePairsOnce(tb testing.TB, e exchange.IBotExchange) {
	tb.Helper()

	updatePairsMutex.Lock()
	defer updatePairsMutex.Unlock()

	b := e.GetBase()
	if c, ok := updatePairsOnce[e.GetName()]; ok {
		b.CurrencyPairs.Load(c)
		return
	}

	err := e.UpdateTradablePairs(context.Background(), true)
	require.NoError(tb, err, "UpdateTradablePairs must not error")

	cache := new(currency.PairsManager)
	cache.Load(&b.CurrencyPairs)
	updatePairsOnce[e.GetName()] = cache
}
