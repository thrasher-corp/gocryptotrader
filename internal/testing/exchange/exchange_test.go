package exchange

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/buffer"
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

func TestSetupVerbose(t *testing.T) {
	b := new(binance.Exchange)
	require.NoError(t, Setup(b, true), "Setup must not error when verbose is overridden")
	assert.True(t, b.IsVerbose(), "Setup should honour the verbose override")
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

func TestSetupWs(t *testing.T) {
	t.Parallel()
	e := new(binance.Exchange)
	require.NoError(t, Setup(e), "Test exchange Setup must not error")

	e.Websocket = websocket.NewManager()
	err := e.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig: e.Config,
		DefaultURL:     "wss://connect-me.com",
		RunningURL:     "wss://connect-me.com",
		Connector:      func() error { return nil },
		Subscriber:     func(subscription.List) error { return nil },
		Unsubscriber:   func(subscription.List) error { return nil },
		GenerateSubscriptions: func() (subscription.List, error) {
			return subscription.List{}, nil
		},
		Features: &e.Features.Supports.WebsocketCapabilities,
		OrderbookBufferConfig: buffer.Config{
			SortBuffer:            true,
			SortBufferByUpdateIDs: true,
		},
	})
	require.NoError(t, err)

	err = e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		ResponseCheckTimeout: e.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     e.WebsocketResponseMaxLimit,
	})
	require.NoError(t, err)
	require.NoError(t, MockHTTPInstance(e), "MockHTTPInstance with no optional path must not error")
	require.NoError(t, MockHTTPInstance(e, "api"), "MockHTTPInstance with optional path must not error")

	SetupWs(t, e)

	err = e.Websocket.DataHandler.Send(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	err = e.Websocket.DataHandler.Send(t.Context(), 1336)
	require.NoError(t, err)
	err = e.Websocket.DataHandler.Send(t.Context(), "intercepted")
	require.NoError(t, err)
	err = e.Websocket.DataHandler.Send(t.Context(), []byte(`{"stream":"btcusdt@ticker","data":{"e":"24hrTicker","E":1580254809477,"s":"ETHBTC","p":"420.97000000","P":"4.720","w":"9058.27981278","x":"8917.98000000","c":"9338.96000000","Q":"0.17246300","b":"9338.03000000","B":"0.18234600","a":"9339.70000000","A":"0.14097600","o":"8917.99000000","h":"9373.19000000","l":"8862.40000000","v":"72229.53692000","q":"654275356.16896672","O":1580168409456,"C":1580254809456,"F":235294268,"L":235894703,"n":600436}}`))
	require.NoError(t, err)

	close(e.Websocket.ShutdownC)
	e.Websocket.DataHandler.Close()
	e.Websocket.Wg.Wait()
}

func TestStreamDataConsumer(t *testing.T) {
	t.Parallel()
	wm := &websocket.Manager{
		ShutdownC:   make(chan struct{}),
		DataHandler: stream.NewRelay(1),
		Wg:          sync.WaitGroup{},
	}
	wm.Wg.Add(1)
	go streamDataConsumer(wm)

	err := wm.DataHandler.Send(context.Background(), 1234)
	require.NoError(t, err)
	err = wm.DataHandler.Send(context.Background(), "intercepted")
	require.NoError(t, err)

	close(wm.ShutdownC)
	wm.DataHandler.Close()
	wm.Wg.Wait()
}

func TestMockWsInstanceVerbose(t *testing.T) {
	b := MockWsInstance[binance.Exchange](t, mockws.CurryWsMockUpgrader(t, func(_ testing.TB, _ []byte, _ *gws.Conn) error { return nil }), true)
	require.NotNil(t, b, "MockWsInstance must not be nil when verbose is overridden")
	assert.True(t, b.IsVerbose(), "MockWsInstance should honour the verbose override")
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
