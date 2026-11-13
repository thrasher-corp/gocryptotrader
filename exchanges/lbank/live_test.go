//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package lbank

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

var (
	mockTests = false
	// apiCredentials holds an API key (Key) and its HmacSHA256 secret or RSA private key (Secret); please supply your
	// own to test the authenticated endpoints live
	apiCredentials = &accounts.Credentials{}
)

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := setupLiveExchange(e); err != nil {
		log.Fatalf("Lbank Setup error: %s", err)
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
// writes do not collide with other tests'. It shares the package exchange's requester, so the tests keep to LBank's
// rate limits together
func newTestExchange(tb testing.TB) *Exchange {
	tb.Helper()
	ex := new(Exchange)
	require.NoError(tb, setupLiveExchange(ex), "setupLiveExchange must not error")
	ex.Name = tb.Name()
	ex.Requester = e.Requester
	return ex
}

// newWebsocketTestExchange returns a live exchange named after the test whose websocket is connected to LBank, with its
// account connection when the credentials above are set, and subs subscribed, or none; nothing records the requests
// sent live, so the venue is nil
func newWebsocketTestExchange(t *testing.T, subs subscription.List) (*Exchange, *websocketVenue) {
	t.Helper()
	ex := newTestExchange(t)
	return ex, connectWebsocketTestExchange(t, ex, subs)
}

// websocketVenue stands in for the mock of LBank's websocket, which live tests do not have
type websocketVenue struct{}

func (*websocketVenue) received(string) []websocketSubscribeRequest { return nil }

// connectWebsocketTestExchange connects an exchange's websocket to LBank, with its account connection when the exchange
// has credentials, and subs subscribed, or none
func connectWebsocketTestExchange(t *testing.T, ex *Exchange, subs subscription.List) *websocketVenue {
	t.Helper()
	venue, err := dialWebsocketTestExchange(t, ex, subs)
	require.NoError(t, err, "Connect must not error")
	return venue
}

// dialWebsocketTestExchange connects an exchange's websocket as connectWebsocketTestExchange does, returning Connect's
// error. The websocket is disabled and shut down when the test ends
func dialWebsocketTestExchange(t *testing.T, ex *Exchange, subs subscription.List) (*websocketVenue, error) {
	t.Helper()
	ex.Features.Subscriptions = subs
	if len(subs) == 0 {
		ex.Websocket.SetSubscriptionsNotRequired()
	}
	ex.Websocket.SetCanUseAuthenticatedEndpoints(ex.API.AuthenticatedWebsocketSupport)
	startRelayRecorder(t, ex)
	err := ex.Websocket.Connect(t.Context())
	t.Cleanup(func() {
		if ex.Websocket.IsEnabled() {
			assert.NoError(t, ex.Websocket.Disable(), "Disable should not error")
		}
		if err := ex.Websocket.Shutdown(); err != nil {
			assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should only fail when the connection monitor has shut the connection down first")
		}
	})
	return nil, err
}
