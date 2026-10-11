//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package hyperliquid

import (
	"context"
	"log"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

var (
	mockTests = false
	// apiCredentials holds the account address (Key), an optional private key of the account or of one of its approved
	// API wallets (Secret) and an optional subaccount or vault (SubAccount); please supply your own for live testing
	apiCredentials = &accounts.Credentials{}
)

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Hyperliquid Setup error: %s", err)
	}
	if apiCredentials.Key != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(apiCredentials)
	}
	log.Printf(sharedtestvalues.LiveTesting, e.Name)
	os.Exit(m.Run())
}

// newTestExchange returns a live exchange named after the test, so the ticker and orderbook stores it writes do not
// collide with other tests'
func newTestExchange(tb testing.TB) *Exchange {
	tb.Helper()
	ex := new(Exchange)
	require.NoError(tb, testexch.Setup(ex), "Setup must not error")
	ex.Name = tb.Name()
	if apiCredentials.Key != "" {
		setTestCredentials(ex, apiCredentials)
	}
	return ex
}

// newUserSignedTestExchange returns the live exchange, whose nonces follow the clock
func newUserSignedTestExchange(*testing.T) *Exchange {
	return e
}

// venueWebsocketExchange connects, on first use, the websocket of the exchange every live test shares: Hyperliquid allows
// 10 connections and 30 new connections a minute per IP, and signing every live action with one exchange's nonce
// counter keeps parallel tests from reusing a nonce
var venueWebsocketExchange = sync.OnceValues(func() (*Exchange, error) {
	e.Websocket.SetSubscriptionsNotRequired()
	return e, e.Websocket.Enable(context.Background())
})

// newVenueWebsocketTestExchange returns the exchange connected to Hyperliquid's websocket, which every caller shares, so
// each must subscribe only to its own feeds and unsubscribe them; nothing records the requests sent live
func newVenueWebsocketTestExchange(t *testing.T) (*Exchange, *websocketVenueRequests) {
	t.Helper()
	ex, err := venueWebsocketExchange()
	require.NoError(t, err, "connecting to Hyperliquid's websocket must not error")
	return ex, nil
}

// venueWebsocketURL returns Hyperliquid's websocket URL
func venueWebsocketURL(*testing.T) string {
	return websocketURL
}
