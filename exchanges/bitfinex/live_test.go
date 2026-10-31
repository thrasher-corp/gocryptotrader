//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package bitfinex

import (
	"log"
	"os"
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
	// apiCredentials holds an API key (Key) and secret (Secret); please supply your own to test the authenticated
	// endpoints live
	apiCredentials = &accounts.Credentials{}
)

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := setupLiveExchange(e); err != nil {
		log.Fatalf("Bitfinex Setup error: %s", err)
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
// writes do not collide with other tests'. It shares the package exchange's requester, as Bitfinex limits each
// endpoint per IP, and blocks an IP over a limit for a minute, and each key's nonces must keep increasing
func newTestExchange(tb testing.TB) *Exchange {
	tb.Helper()
	ex := new(Exchange)
	require.NoError(tb, setupLiveExchange(ex), "setupLiveExchange must not error")
	ex.Name = tb.Name()
	ex.Requester = e.Requester
	return ex
}

// newWebsocketTestExchange returns a live exchange named after the test whose websocket is connected to Bitfinex,
// authenticated when the credentials above are set, with subs subscribed, or none; nothing records the requests sent
// live, so the venue is nil
func newWebsocketTestExchange(t *testing.T, subs subscription.List, opts ...websocketVenueOption) (*Exchange, *websocketVenue) {
	t.Helper()
	ex := newTestExchange(t)
	return ex, connectWebsocketTestExchange(t, ex, subs, opts...)
}

// websocketVenue stands in for the mock of Bitfinex's websocket, which live tests do not have
type websocketVenue struct{}

func (*websocketVenue) events(string) []map[string]any  { return nil }
func (*websocketVenue) inputs(string) []json.RawMessage { return nil }

// websocketVenueOption sets the mock of Bitfinex's websocket up, which live tests do not have
type websocketVenueOption func(*websocketVenue)

// withBadChecksum does nothing live, where checksums cannot be made to fail
func withBadChecksum(string) websocketVenueOption { return func(*websocketVenue) {} }

// withRefusedAuthNonces does nothing live, where Bitfinex cannot be made to refuse a nonce
func withRefusedAuthNonces(int) websocketVenueOption { return func(*websocketVenue) {} }

// withRefusedResubscriptions does nothing live, where Bitfinex cannot be made to refuse a subscription
func withRefusedResubscriptions(string, int) websocketVenueOption { return func(*websocketVenue) {} }

// connectWebsocketTestExchange connects an exchange's websocket to Bitfinex, authenticated when the exchange has
// credentials, with subs subscribed, or none. The websocket is disabled and shut down when the test ends
func connectWebsocketTestExchange(t *testing.T, ex *Exchange, subs subscription.List, _ ...websocketVenueOption) *websocketVenue {
	t.Helper()
	ex.Features.Subscriptions = subs
	if len(subs) == 0 {
		ex.Websocket.SetSubscriptionsNotRequired()
	}
	ex.Websocket.SetCanUseAuthenticatedEndpoints(ex.API.AuthenticatedWebsocketSupport)
	startRelayRecorder(t, ex)
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	t.Cleanup(func() {
		if ex.Websocket.IsEnabled() {
			assert.NoError(t, ex.Websocket.Disable(), "Disable should not error")
		}
		if err := ex.Websocket.Shutdown(); err != nil {
			assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should only fail when the connection monitor has shut the connection down first")
		}
	})
	return nil
}
