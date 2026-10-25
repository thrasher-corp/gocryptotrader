package lbank

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

const testSubscribeKey = "4e9958623e6006bd7b13ff9f36c03b36132f0f8da37f70b14ff2c4eab1fe0c97"

// subscribeKeyServer counts a test server's subscribe key requests by path, answering each as LBank does
type subscribeKeyServer struct {
	mu       sync.Mutex
	requests map[string][]string
	failGet  bool
}

func (s *subscribeKeyServer) handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if !assert.NoError(t, r.ParseForm(), "the request body should parse as a form") {
			return
		}
		s.mu.Lock()
		if s.requests == nil {
			s.requests = make(map[string][]string)
		}
		s.requests[r.URL.Path] = append(s.requests[r.URL.Path], r.PostForm.Get("subscribeKey"))
		failGet := s.failGet
		s.mu.Unlock()
		switch {
		case r.URL.Path == "/v2/subscribe/get_key.do" && failGet:
			writeResponse(t, w, `{"result":"false","msg":"API Key permission denied","error_code":10022,"ts":1}`)
		case r.URL.Path == "/v2/subscribe/get_key.do":
			writeResponse(t, w, `{"result":true,"data":"`+testSubscribeKey+`","error_code":0,"ts":1}`)
		default:
			writeResponse(t, w, `{"result":"true"}`)
		}
	}
}

func (s *subscribeKeyServer) sent(path string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requests[path]...)
}

// waitForRenewers waits for an exchange's subscribe key renewers to stop, failing the test if they don't
func waitForRenewers(t *testing.T, ex *Exchange) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		ex.Websocket.Wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the subscribe key renewer must stop")
	}
}

func TestCreateSubscribeKey(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.CreateSubscribeKey(t.Context())
	require.NoError(t, err, "CreateSubscribeKey must not error")
	if mockTests {
		assert.Equal(t, testSubscribeKey, result, "CreateSubscribeKey should decode the key")
		return
	}
	assert.NotEmpty(t, result, "CreateSubscribeKey should return a key")
	assert.NoError(t, e.CloseSubscribeKey(t.Context(), result), "CloseSubscribeKey should close the key")
}

func TestRefreshSubscribeKey(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.RefreshSubscribeKey(t.Context(), ""), errSubscribeKeyRequired, "RefreshSubscribeKey must require a key")
	if !mockTests {
		t.Skip("RefreshSubscribeKey needs a key the account created")
	}
	assert.NoError(t, e.RefreshSubscribeKey(t.Context(), testSubscribeKey), "RefreshSubscribeKey should not error")
}

func TestCloseSubscribeKey(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.CloseSubscribeKey(t.Context(), ""), errSubscribeKeyRequired, "CloseSubscribeKey must require a key")
	if !mockTests {
		t.Skip("CloseSubscribeKey needs a key the account created")
	}
	assert.NoError(t, e.CloseSubscribeKey(t.Context(), testSubscribeKey), "CloseSubscribeKey should not error")
}

func TestSubscribeKeyStore(t *testing.T) {
	t.Parallel()
	var s subscribeKeyStore
	a, b := &recordingConnection{}, &recordingConnection{}
	assert.Empty(t, s.get(a), "get should return no key for a connection without one")
	s.set(a, "key-a")
	s.set(b, "key-b")
	assert.Equal(t, "key-a", s.get(a), "get should return the connection's own key")
	s.remove(a, "stale")
	assert.Equal(t, "key-a", s.get(a), "remove should keep a key the connection was given since")
	s.remove(a, "key-a")
	assert.Empty(t, s.get(a), "remove should forget the connection's key")
	assert.Equal(t, "key-b", s.get(b), "remove should leave other connections' keys")
}

func TestWebsocketConnectAccount(t *testing.T) {
	t.Parallel()
	server := &subscribeKeyServer{}
	ex := newTestServerExchange(t, server.handler(t))
	conn := &recordingConnection{connected: true}
	ctx, cancel := context.WithCancel(t.Context())
	require.NoError(t, ex.wsConnectAccount(ctx, conn), "wsConnectAccount must not error")
	assert.Equal(t, 1, conn.dialled, "wsConnectAccount should dial the connection")
	assert.Equal(t, testSubscribeKey, ex.subscribeKeys.get(conn), "wsConnectAccount should give the connection its key")
	assert.Equal(t, []string{""}, server.sent("/v2/subscribe/get_key.do"), "wsConnectAccount should create one key")

	cancel()
	waitForRenewers(t, ex)
	assert.Equal(t, []string{testSubscribeKey}, server.sent("/v2/subscribe/destroy_key.do"), "the renewer should close the key when the connection ends")
	assert.Empty(t, ex.subscribeKeys.get(conn), "the renewer should forget the closed key")
}

func TestWebsocketConnectAccountFailures(t *testing.T) {
	t.Parallel()
	refused := &subscribeKeyServer{failGet: true}
	ex := newTestServerExchange(t, refused.handler(t))
	conn := &recordingConnection{}
	err := ex.wsConnectAccount(t.Context(), conn)
	require.ErrorIs(t, err, request.ErrAuthRequestFailed, "wsConnectAccount must report a key it couldn't create")
	assert.Zero(t, conn.dialled, "wsConnectAccount should not dial without a key")

	server := &subscribeKeyServer{}
	ex = newTestServerExchange(t, server.handler(t))
	failing := &recordingConnection{dialErr: errSendFailed}
	require.ErrorIs(t, ex.wsConnectAccount(t.Context(), failing), errSendFailed, "wsConnectAccount must report a failed dial")
	assert.Equal(t, []string{testSubscribeKey}, server.sent("/v2/subscribe/destroy_key.do"), "wsConnectAccount should close the key of a connection it couldn't dial")
	assert.Empty(t, ex.subscribeKeys.get(failing), "wsConnectAccount should not keep the key of a connection it couldn't dial")
	waitForRenewers(t, ex)
}

func TestKeepSubscribeKeyAlive(t *testing.T) {
	t.Parallel()
	server := &subscribeKeyServer{}
	ex := newTestServerExchange(t, server.handler(t))
	conn := &recordingConnection{connected: true}
	ex.subscribeKeys.set(conn, testSubscribeKey)
	ex.Websocket.Wg.Add(1)
	go ex.keepSubscribeKeyAlive(t.Context(), conn, testSubscribeKey, 10*time.Millisecond)
	require.Eventually(t, func() bool {
		return len(server.sent("/v2/subscribe/refresh_key.do")) >= 2
	}, 5*time.Second, 5*time.Millisecond, "the renewer must refresh the key while the connection lives")

	conn.setConnected(false)
	waitForRenewers(t, ex)
	for _, key := range server.sent("/v2/subscribe/refresh_key.do") {
		assert.Equal(t, testSubscribeKey, key, "the renewer should refresh the connection's own key")
	}
	assert.Equal(t, []string{testSubscribeKey}, server.sent("/v2/subscribe/destroy_key.do"), "the renewer should close the key once the connection has gone")
	assert.Empty(t, ex.subscribeKeys.get(conn), "the renewer should forget the closed key")
}

func TestKeepSubscribeKeyAliveRefreshFailure(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/subscribe/refresh_key.do" {
			writeResponse(t, w, `{"result":"false","msg":"Not existing or expired subscribeKey","error_code":10003,"ts":1}`)
			return
		}
		writeResponse(t, w, `{"result":"true"}`)
	}))
	conn := &recordingConnection{connected: true}
	ctx, cancel := context.WithCancel(t.Context())
	ex.Websocket.Wg.Add(1)
	go ex.keepSubscribeKeyAlive(ctx, conn, testSubscribeKey, 10*time.Millisecond)
	var relayed []error
	require.Eventually(t, func() bool {
		for len(ex.Websocket.DataHandler.C) > 0 {
			if err, ok := (<-ex.Websocket.DataHandler.C).Data.(error); ok {
				relayed = append(relayed, err)
			}
		}
		return len(relayed) >= 2
	}, 5*time.Second, 5*time.Millisecond, "the renewer must keep retrying a refresh it couldn't make, relaying each failure")
	cancel()
	waitForRenewers(t, ex)
	assert.ErrorIs(t, relayed[0], errRequestFailed, "the renewer should relay LBank's refusal")
}

func TestWebsocketAccountSubscribe(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	subs := subscription.List{
		{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true},
		{Enabled: true, Channel: subscription.MyAccountChannel, Authenticated: true},
	}
	ex, venue := newWebsocketTestExchange(t, subs)
	stored := ex.Websocket.GetSubscriptions()
	require.Len(t, stored, 2, "both account subscriptions must be stored")
	for _, s := range stored {
		assert.Equalf(t, subscription.SubscribedState, s.State(), "subscription %s should be subscribed", s)
	}
	conn, err := ex.Websocket.GetConnection(wsAccountConnection)
	require.NoError(t, err, "GetConnection must find the account connection")
	require.NotEmpty(t, ex.subscribeKeys.get(conn), "the account connection must hold a subscribe key")
	if !mockTests {
		// LBank refuses an unknown or expired subscribe key with an error frame shortly after the subscription
		assert.Never(t, func() bool { return len(recordedErrors(t, ex)) > 0 }, 5*time.Second, 100*time.Millisecond, "LBank should accept the connection's subscribe key")
		return
	}
	exp := []websocketSubscribeRequest{
		{Action: "subscribe", Subscribe: "orderUpdate", Pair: "all", SubscribeKey: testSubscribeKey},
		{Action: "subscribe", Subscribe: "assetUpdate", SubscribeKey: testSubscribeKey},
	}
	// LBank confirms no subscription, so the mock may still be reading the last requests
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.ElementsMatch(c, exp, venue.received("subscribe"), "each account subscription should carry the key LBank created for the connection")
	}, 30*time.Second, 50*time.Millisecond, "the mock should receive each account subscription")
	// The mock follows the balance subscription with the recorded balance update, which is stored
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		balance, err := ex.Accounts.GetBalance("", creds, asset.Spot, currency.USDT)
		if assert.NoError(c, err, "GetBalance should find the streamed balance") {
			assert.InDelta(c, 114548.31881315, balance.Total, 1e-8, "the streamed balance should be stored")
		}
	}, 30*time.Second, 50*time.Millisecond, "the account channels must deliver updates")
	assert.Empty(t, recordedErrors(t, ex), "the account updates should be handled without error")
}

func TestRefreshDelay(t *testing.T) {
	t.Parallel()
	assert.Equal(t, subscribeKeyRefreshInterval, refreshDelay(nil, subscribeKeyRefreshInterval), "refreshDelay should wait the refresh interval after a renewal")
	assert.Equal(t, subscribeKeyRetryInterval, refreshDelay(errRequestFailed, subscribeKeyRefreshInterval), "refreshDelay should retry sooner after a failed renewal")
	assert.Equal(t, 10*time.Millisecond, refreshDelay(errRequestFailed, 10*time.Millisecond), "refreshDelay should never wait longer than the refresh interval")
}

func TestWebsocketAccountHandlers(t *testing.T) {
	t.Parallel()
	ex := newChannelTestExchange(t)
	conn := &recordingConnection{}
	testexch.FixtureToDataHandler(t, "testdata/wsAccount.json", func(ctx context.Context, b []byte) error {
		return ex.wsHandleData(ctx, conn, b)
	})
	pair, err := ex.MatchSymbolWithAvailablePairs("btc_usdt", asset.Spot, true)
	require.NoError(t, err, "MatchSymbolWithAvailablePairs must find btc_usdt")

	payloads := relayedPayloads(ex)
	require.Len(t, payloads, 4, "the handlers must relay both orders, the fill and the balance")
	balances, ok := payloads[3].(accounts.SubAccounts)
	require.True(t, ok, "the asset handler must relay the balance")
	require.Len(t, balances, 1, "the asset handler must relay one sub-account")
	// LBank sends its coins in lower case, which the codes it relays keep
	coin := currency.NewCode("usdt")
	usdt, ok := balances[0].Balances[coin]
	require.True(t, ok, "the asset handler must relay the coin's balance")
	require.NotZero(t, usdt.UpdatedAt, "the asset handler must stamp the balance")
	usdt.UpdatedAt = time.Time{}
	balances[0].Balances[coin] = usdt

	exp := []any{
		&order.Detail{
			Exchange: ex.Name, OrderID: "d0db191d-1a2b-4418-9c3d-fbb1aa2c2ea9", ClientOrderID: "gct-sell", Pair: pair, AssetType: asset.Spot,
			Side: order.Sell, Status: order.PartiallyFilled, Price: 80000, Amount: 0.005, ExecutedAmount: 0.003, RemainingAmount: 0.002,
			AverageExecutedPrice: 80000.5, LastUpdated: time.UnixMilli(1792888874775),
		},
		[]fill.Data{{
			ID: "da88f354d5e04c4ea12128aa5bdcb301", TradeID: "da88f354d5e04c4ea12128aa5bdcb301", Exchange: ex.Name, AssetType: asset.Spot,
			CurrencyPair: pair, Side: order.Sell, OrderID: "d0db191d-1a2b-4418-9c3d-fbb1aa2c2ea9", ClientOrderID: "gct-sell",
			Price: 80000.75, Amount: 0.002, Timestamp: time.UnixMilli(1792888874775),
		}},
		&order.Detail{
			Exchange: ex.Name, OrderID: "8241275e-8666-4ab8-a84d-2e87f82e12ff", Pair: pair, AssetType: asset.Spot,
			Side: order.Buy, Status: order.Active, Price: 50000, Amount: 0.001, LastUpdated: time.UnixMilli(1792888874999),
		},
		accounts.SubAccounts{{
			AssetType: asset.Spot,
			Balances: accounts.CurrencyBalances{
				coin: {Currency: coin, Total: 114548.31881315, Free: 97430.6739041, Hold: 17117.64490905},
			},
		}},
	}
	assert.Equal(t, exp, payloads, "the account handlers should relay the orders, the fill and the balance")

	stored, err := ex.Accounts.CurrencyBalances(nil, asset.Spot)
	require.NoError(t, err, "CurrencyBalances must not error")
	assert.Equal(t, 114548.31881315, stored[currency.USDT].Total, "the asset handler should store the balance it relays")
}

func TestWebsocketAccountHandlerErrors(t *testing.T) {
	t.Parallel()
	ex := newChannelTestExchange(t)
	conn := &recordingConnection{}
	for _, tc := range []struct {
		name  string
		frame string
		err   error
	}{
		{name: "unknown pair", frame: `{"orderUpdate":{"symbol":"nope_usdt","type":"buy","orderStatus":0},"type":"orderUpdate"}`, err: currency.ErrPairNotFound},
		{name: "unknown side", frame: `{"orderUpdate":{"symbol":"btc_usdt","type":"hold","orderStatus":0},"type":"orderUpdate"}`, err: order.ErrSideIsInvalid},
		{name: "unknown status", frame: `{"orderUpdate":{"symbol":"btc_usdt","type":"buy","orderStatus":9},"type":"orderUpdate"}`, err: order.ErrUnsupportedStatusType},
		{name: "balance without a coin", frame: `{"data":{"asset":"1","free":"1","freeze":"0"},"type":"assetUpdate"}`, err: currency.ErrCurrencyCodeEmpty},
	} {
		err := ex.wsHandleData(t.Context(), conn, []byte(tc.frame))
		assert.ErrorIsf(t, err, tc.err, "wsHandleData should reject an update with an %s", tc.name)
	}
	assert.Empty(t, relayedPayloads(ex), "wsHandleData should relay nothing it rejected")

	ex.Websocket.Fills.Setup(false, ex.Websocket.DataHandler)
	frame := `{"orderUpdate":{"accAmt":"1","amount":"1","symbol":"btc_usdt","type":"buy","orderStatus":2,"price":"1","txUuid":"t1","uuid":"o1","updateTime":1792888874775},"type":"orderUpdate"}`
	require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(frame)), "wsHandleData must relay an order's trade without the fills feed")
	payloads := relayedPayloads(ex)
	require.Len(t, payloads, 1, "wsHandleData must relay only the order without the fills feed")
	assert.IsType(t, &order.Detail{}, payloads[0], "wsHandleData should relay the order")
}
