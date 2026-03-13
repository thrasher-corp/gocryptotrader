package exchange

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bybit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

const multiConnectionFilter = "multi-connection-test"

type multiConnectionSetupExchange struct {
	sharedtestvalues.CustomEx
}

func (e *multiConnectionSetupExchange) GetBase() *exchange.Base {
	return &e.Base
}

func newMultiConnectionSetupExchange(tb testing.TB, websocketURL string) *multiConnectionSetupExchange {
	tb.Helper()

	e := &multiConnectionSetupExchange{}
	e.Base.Name = "MultiConnectionSetupExchange"
	e.Base.Features.Subscriptions = subscription.List{{Channel: "ticker"}}
	e.Base.Websocket = websocket.NewManager()

	err := e.Base.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig: &config.Exchange{
			Name: "MultiConnectionSetupExchange",
			Features: &config.FeaturesConfig{
				Enabled: config.FeaturesEnabledConfig{Websocket: true},
			},
			WebsocketTrafficTimeout: 5 * time.Second,
		},
		Features:                     &protocol.Features{},
		UseMultiConnectionManagement: true,
	})
	require.NoError(tb, err, "Setup must not error for the multi-connection manager")

	err = e.Base.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL: websocketURL,
		Connector: func(ctx context.Context, conn websocket.Connection) error {
			return conn.Dial(ctx, gws.DefaultDialer, nil, nil)
		},
		GenerateSubscriptions: func() (subscription.List, error) {
			return e.Base.Features.Subscriptions.Clone(), nil
		},
		Subscriber:    func(context.Context, websocket.Connection, subscription.List) error { return nil },
		Handler:       func(context.Context, websocket.Connection, []byte) error { return nil },
		MessageFilter: multiConnectionFilter,
	})
	require.NoError(tb, err, "SetupNewConnection must not error for the multi-connection manager")

	return e
}

// TestSetup exercises Setup
func TestSetup(t *testing.T) {
	b := new(binance.Exchange)
	require.NoError(t, Setup(b), "Setup must not error")
	assert.NotNil(t, b.Websocket, "Websocket should not be nil after Setup")

	e := new(sharedtestvalues.CustomEx)
	assert.ErrorIs(t, Setup(e), config.ErrExchangeNotFound, "Setup should error correctly on a missing exchange")
}

// TestMockHTTPInstance exercises MockHTTPInstance
func TestMockHTTPInstance(t *testing.T) {
	b := new(binance.Exchange)
	require.NoError(t, Setup(b), "Test exchange Setup must not error")
	require.NoError(t, MockHTTPInstance(b), "MockHTTPInstance with no optional path must not error")
	require.NoError(t, MockHTTPInstance(b, "api"), "MockHTTPInstance with optional path must not error")
}

// TestMockWsInstance exercises MockWsInstance
func TestMockWsInstance(t *testing.T) {
	b := MockWsInstance[binance.Exchange](t, mockws.CurryWsMockUpgrader(t, func(_ testing.TB, _ []byte, _ *gws.Conn) error { return nil }))
	require.NotNil(t, b, "MockWsInstance must not be nil")
}

func TestMockWsInstanceSupportsMultiConnectionManagement(t *testing.T) {
	b := MockWsInstance[bybit.Exchange](t, mockws.CurryWsMockUpgrader(t, func(_ testing.TB, _ []byte, _ *gws.Conn) error { return nil }))
	require.NotNil(t, b, "MockWsInstance must not be nil for multi-connection websocket exchanges")
	t.Cleanup(func() {
		if b.GetBase().Websocket.IsConnected() {
			assert.NoError(t, b.GetBase().Websocket.Shutdown(), "Websocket shutdown should not error for multi-connection websocket exchanges")
		}
	})
	assert.True(t, b.GetBase().Websocket.IsConnected(), "Websocket manager should be connected for multi-connection websocket exchanges")
}

func TestSetupWsSupportsMultiConnectionManagement(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
	}))
	t.Cleanup(server.Close)

	e := newMultiConnectionSetupExchange(t, "ws"+strings.TrimPrefix(server.URL, "http"))
	t.Cleanup(func() {
		if e.Base.Websocket.IsConnected() {
			assert.NoError(t, e.Base.Websocket.Shutdown(), "Websocket shutdown should not error after SetupWs")
		}
	})

	SetupWs(t, e)

	assert.Empty(t, e.Base.Features.Subscriptions, "Features.Subscriptions should be cleared by SetupWs")
	assert.True(t, e.Base.Websocket.IsConnected(), "Websocket manager should be connected after SetupWs")

	conn, err := e.Base.Websocket.GetConnection(multiConnectionFilter)
	require.NoError(t, err, "GetConnection must not error after SetupWs on a multi-connection manager")
	assert.NotNil(t, conn, "GetConnection should return a connection after SetupWs on a multi-connection manager")
	assert.Empty(t, conn.Subscriptions().List(), "Connection subscriptions should remain empty when subscriptions are not required")
}
