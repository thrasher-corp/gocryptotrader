package exchange

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

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
	e.SetDefaults()
	eName := e.GetName()
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

// FixtureError contains an error and the message that caused it
type FixtureError struct {
	Err error
	Msg []byte
}

// FixtureToDataHandler squirts the contents of a file to a reader function (probably e.wsHandleData) and asserts no errors are returned
func FixtureToDataHandler(tb testing.TB, fixturePath string, reader func([]byte) error) {
	tb.Helper()

	for _, e := range FixtureToDataHandlerWithErrors(tb, fixturePath, reader) {
		assert.NoErrorf(tb, e.Err, "Should not error handling message:\n%s", e.Msg)
	}
}

// FixtureToDataHandlerWithErrors squirts the contents of a file to a reader function (probably e.wsHandleData) and returns handler errors
// Any errors setting up the fixture will fail tests
func FixtureToDataHandlerWithErrors(tb testing.TB, fixturePath string, reader func([]byte) error) []FixtureError {
	tb.Helper()

	fixture, err := os.Open(fixturePath)
	require.NoError(tb, err, "Opening fixture '%s' must not error", fixturePath)
	defer func() {
		assert.NoError(tb, fixture.Close(), "Closing the fixture file should not error")
	}()

	errs := []FixtureError{}
	s := bufio.NewScanner(fixture)
	for s.Scan() {
		msg := s.Bytes()
		if err := reader(msg); err != nil {
			errs = append(errs, FixtureError{
				Err: err,
				Msg: msg,
			})
		}
	}
	assert.NoError(tb, s.Err(), "Fixture Scanner should not error")
	return errs
}

var (
	setupWsMutex sync.Mutex
	setupWsOnce  = make(map[exchange.IBotExchange]bool)
)

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

var (
	updatePairsMutex sync.Mutex
	updatePairsOnce  = make(map[string]*currency.PairsManager)
)

// UpdatePairsOnce ensures pairs are only updated once in parallel tests
// A clone of the cache of the updated pairs is used to populate duplicate requests
// Any pairs enabled after this is called will be lost on the next call
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
