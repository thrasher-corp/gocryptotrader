//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package cryptodotcom

import (
	"context"
	"log"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

var (
	mockTests = false
	// apiCredentials holds an API key and secret; please supply your own for live testing
	apiCredentials = &accounts.Credentials{}
)

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := setupLiveExchange(e); err != nil {
		log.Fatalf("Cryptodotcom Setup error: %s", err)
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
	return nil
}

// newTestExchange returns a live exchange named after the test, so the ticker, orderbook and account stores it writes
// do not collide with other tests'
func newTestExchange(tb testing.TB) *Exchange {
	tb.Helper()
	ex := new(Exchange)
	require.NoError(tb, setupLiveExchange(ex), "setupLiveExchange must not error")
	ex.Name = tb.Name()
	return ex
}

// websocketTestExchange connects, on first use, the websocket of the exchange every live test shares, so the tests do
// not open a connection each
var websocketTestExchange = sync.OnceValues(func() (*Exchange, error) {
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	return e, e.Websocket.Connect(context.Background())
})

// newWebsocketTestExchange returns the exchange connected to Crypto.com's websocket, which every caller shares; it
// skips the test without credentials, as the trade connection only serves authenticated requests
func newWebsocketTestExchange(t *testing.T) (*Exchange, *websocketVenue) {
	t.Helper()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	ex, err := websocketTestExchange()
	require.NoError(t, err, "connecting to Crypto.com's websocket must not error")
	return ex, nil
}

// websocketVenueRequest stands in for a request the mock websocket received
type websocketVenueRequest struct {
	ID     int64
	Method string
	Params json.RawMessage
}

// websocketVenue stands in for the mock of Crypto.com's websocket, which the live tests do not use; it records nothing
type websocketVenue struct {
	url string
}

// newWebsocketVenue skips the test, as the live tests use Crypto.com's websocket rather than a mock of it
func newWebsocketVenue(t *testing.T) *websocketVenue {
	t.Helper()
	t.Skip("the live tests use Crypto.com's websocket")
	return nil
}

func (*websocketVenue) closing() {}

func (*websocketVenue) respond(string, string) {}

func (*websocketVenue) fail(string, *APIError) {}

func (*websocketVenue) received(string) []websocketVenueRequest { return nil }

func (*websocketVenue) rejectSubscriptions(string, *APIError) {}

func (*websocketVenue) answerSubscriptions(string, ...[]json.RawMessage) {}

func (*websocketVenue) beforeAnswering(string, func()) {}

// newMarketDataVenue stands in for the mock of Crypto.com's market data websocket, which the live tests do not use
func newMarketDataVenue(*testing.T) *websocketVenue {
	return nil
}

// newMarketDataTestExchange returns a live exchange named after the test, with its trade feed enabled and prepared by
// each of prepare, whose market data connections are subscribed to subs on Crypto.com's market data websocket, and the
// error connecting; the connections are not authenticated
func newMarketDataTestExchange(t *testing.T, _ *websocketVenue, subs subscription.List, prepare ...func(*Exchange)) (*Exchange, error) {
	t.Helper()
	ex := newTestExchange(t)
	enableTradeFeed(ex)
	for _, p := range prepare {
		p(ex)
	}
	ex.Features.Subscriptions = subs
	ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
	err := ex.Websocket.Connect(t.Context())
	t.Cleanup(func() {
		// Disabling stops the connection monitor, which would otherwise redial Crypto.com after the test
		assert.NoError(t, ex.Websocket.Disable(), "Disable should not error")
		if err := ex.Websocket.Shutdown(); err != nil {
			assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should only fail when the connection monitor has shut the connection down first")
		}
	})
	return ex, err
}

// newUserDataVenue skips the test, as the user channels push only what an account does, which the live tests cannot
// arrange
func newUserDataVenue(t *testing.T) *websocketVenue {
	t.Helper()
	t.Skip("the user channels are tested against the mock websocket")
	return nil
}

// newUserDataTestExchange skips the test, as the user channels are tested against the mock websocket
func newUserDataTestExchange(t *testing.T, _ *websocketVenue, _ subscription.List, _ ...func(*Exchange)) (*Exchange, error) {
	t.Helper()
	t.Skip("the user channels are tested against the mock websocket")
	return nil, nil
}
