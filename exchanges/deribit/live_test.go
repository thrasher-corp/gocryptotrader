//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package deribit

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// useTestnet sends every live test to Deribit's testnet, whose credentials are separate from production's
const useTestnet = false

var (
	mockTests = false
	// apiCredentials holds the client ID (Key) and client secret (Secret) of an API key; please supply your own for live
	// testing
	apiCredentials = &accounts.Credentials{}
)

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := setupLiveExchange(e); err != nil {
		log.Fatalf("Deribit Setup error: %s", err)
	}
	log.Printf(sharedtestvalues.LiveTesting, e.Name)
	os.Exit(m.Run())
}

// setupLiveExchange sets an exchange up for live testing, with the credentials above when they are set
func setupLiveExchange(ex *Exchange) error {
	if err := setupTestExchange(ex); err != nil {
		return err
	}
	if apiCredentials.Key != "" {
		setTestCredentials(ex, apiCredentials)
	}
	if !useTestnet {
		return nil
	}
	for k, v := range ex.API.Endpoints.GetURLMap() {
		if err := ex.API.Endpoints.SetRunningURL(k, strings.Replace(v, apiURL, testnetAPIURL, 1)); err != nil {
			return err
		}
	}
	return ex.Websocket.SetAllConnectionURLs(testnetWebsocketURL)
}

// newTestExchange returns a live exchange named after the test, so the ticker and orderbook stores it writes do not
// collide with other tests'
func newTestExchange(tb testing.TB) *Exchange {
	tb.Helper()
	ex := new(Exchange)
	require.NoError(tb, setupLiveExchange(ex), "setupLiveExchange must not error")
	ex.Name = tb.Name()
	return ex
}

// websocketTestExchange connects, on first use, the websocket of the exchange every live test shares, authenticated
// when the credentials above are set, so the tests do not exhaust Deribit's connection limits
var websocketTestExchange = sync.OnceValues(func() (*Exchange, error) {
	e.Websocket.SetSubscriptionsNotRequired()
	e.Websocket.SetCanUseAuthenticatedEndpoints(apiCredentials.Key != "")
	return e, e.Websocket.Connect(context.Background())
})

// newWebsocketTestExchange returns the exchange connected to Deribit's websocket, which every caller shares, so each
// must subscribe only to its own channels and unsubscribe them; nothing records the requests sent live
func newWebsocketTestExchange(t *testing.T) (*Exchange, *websocketVenue) {
	t.Helper()
	ex, err := websocketTestExchange()
	require.NoError(t, err, "connecting to Deribit's websocket must not error")
	return ex, nil
}

// newUnauthenticatedWebsocketTestExchange skips the test, as the live tests share one websocket, whose connections log in
// when they connect with credentials set
func newUnauthenticatedWebsocketTestExchange(t *testing.T) (*Exchange, *websocketVenue) {
	t.Helper()
	t.Skip("the live tests share one websocket, whose connections log in when they connect")
	return nil, nil
}

// connectWebsocketTestExchange skips the test, as the live tests share one websocket, connected once without
// subscriptions
func connectWebsocketTestExchange(t *testing.T, _ bool, _ subscription.List, _ ...func(*config.Exchange)) (*Exchange, *websocketVenue) {
	t.Helper()
	t.Skip("the live tests share one websocket, connected once without subscriptions")
	return nil, nil
}

// websocketVenue stands in for the mock of Deribit's websocket, which the live tests do not use, as they share Deribit's
// websocket; it records nothing
type websocketVenue struct{}

func (*websocketVenue) delayNext(string, time.Duration) {}

func (*websocketVenue) holdNext(*testing.T, string) func() { return func() {} }

func (*websocketVenue) notifyBeforeNext(string, []byte) {}

func (*websocketVenue) refuse(string, int) {}

func (*websocketVenue) retryNext(string, int) {}

func (*websocketVenue) answerNext(string, string) {}

func (*websocketVenue) push(*testing.T, int, []byte) {}

func (*websocketVenue) drop(*testing.T, int) {}

func (*websocketVenue) received(string) []websocketVenueRequest { return nil }

func (*websocketVenue) receivedOn(int, string) []websocketVenueRequest { return nil }

func (*websocketVenue) connectionCount() int { return 0 }

// startWebsocketVenue skips the test, as the live tests share Deribit's websocket rather than a mock of it
func startWebsocketVenue(t *testing.T) (string, *websocketVenue) {
	t.Helper()
	t.Skip("the live tests share Deribit's websocket")
	return "", nil
}
