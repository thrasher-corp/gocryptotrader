//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package coinbase

import (
	"context"
	"log"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

var (
	mockTests = false
	// apiCredentials holds a CDP API key's name (Key) and private key (Secret); please supply your own to test the
	// authenticated endpoints live
	apiCredentials = &accounts.Credentials{}
)

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := setupLiveExchange(e); err != nil {
		log.Fatalf("Coinbase Setup error: %s", err)
	}
	log.Printf(sharedtestvalues.LiveTesting, e.Name)
	os.Exit(m.Run())
}

// setupLiveExchange sets an exchange up for live testing, with the credentials above when they are set
func setupLiveExchange(ex *Exchange) error {
	if err := setupTestExchange(ex); err != nil {
		return err
	}
	if apiCredentials.Key != "" && apiCredentials.Secret != "" {
		setTestCredentials(ex, apiCredentials)
	}
	return nil
}

// newTestExchange returns a live exchange named after the test, so the ticker, order book and account stores it
// writes do not collide with other tests'. It shares the package exchange's requester, so the rate limits hold across
// the tests
func newTestExchange(tb testing.TB) *Exchange {
	tb.Helper()
	ex := new(Exchange)
	require.NoError(tb, setupLiveExchange(ex), "setupLiveExchange must not error")
	ex.Name = tb.Name()
	ex.Requester = e.Requester
	return ex
}

// derivativesWebsocketTestExchange connects, on first use, the gateway websocket of the exchange every live test shares,
// authenticated when the credentials above are set, so the tests do not exhaust the gateway's connection limits
var derivativesWebsocketTestExchange = sync.OnceValues(func() (*Exchange, error) {
	if err := setupDerivativesWebsocket(e); err != nil {
		return nil, err
	}
	e.Websocket.SetSubscriptionsNotRequired()
	e.Websocket.SetCanUseAuthenticatedEndpoints(apiCredentials.Key != "")
	return e, e.Websocket.Connect(context.Background())
})

// derivativesVenue stands in for the mock of the gateway's websocket, which live tests do not have
type derivativesVenue struct{}

// derivativesVenueRequest stands in for a request the mock records
type derivativesVenueRequest struct {
	Method string
	Params []byte
}

// requestsFor returns nothing, as nothing records the requests sent live
func (*derivativesVenue) requestsFor(string) []derivativesVenueRequest { return nil }

// override does nothing, as nothing answers the requests sent live but the gateway
func (*derivativesVenue) override(string, []byte) {}

// newDerivativesWebsocketTestExchange returns the exchange connected to the gateway's websocket, which every caller
// shares, so each must subscribe only to its own channels and unsubscribe them; nothing records the requests sent
// live, so the venue is nil
func newDerivativesWebsocketTestExchange(t *testing.T) (*Exchange, *derivativesVenue) {
	t.Helper()
	ex, err := derivativesWebsocketTestExchange()
	require.NoError(t, err, "connecting to the gateway's websocket must not error")
	return ex, nil
}

// newUnauthenticatedDerivativesWebsocketTestExchange skips the test, as the live tests share one connection, which
// authenticates when it connects with credentials set
func newUnauthenticatedDerivativesWebsocketTestExchange(t *testing.T) (*Exchange, *derivativesVenue) {
	t.Helper()
	t.Skip("the live tests share one connection, which authenticates when it connects")
	return nil, nil
}
