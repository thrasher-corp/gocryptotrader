package lbank

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testutils "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
)

const (
	testAPIKey    = "key"
	testAPISecret = "secret"
)

type fakeRoundTripper struct {
	body string
	err  error
}

type wsConnectFixtureConnection struct {
	websocket.Connection
	dialErr error

	dialCalls atomic.Int32
	readCalls atomic.Int32
}

type manageSubsFixtureConnection struct {
	websocket.Connection
	sentMessages []any
	sendErr      error
}

type wsReadDataFixtureConnection struct {
	websocket.Connection
	messages [][]byte
	idx      int
}

// lbankWsMockHandler captures every raw message sent to the mock websocket
// server, letting the test assert on exactly what manageSubs put on the wire.
type lbankWsMockHandler struct {
	mu       sync.Mutex
	messages [][]byte
}

func (h *lbankWsMockHandler) handle(_ testing.TB, p []byte, _ *gws.Conn) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.messages = append(h.messages, p)
	return nil
}

func (h *lbankWsMockHandler) captured() [][]byte {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([][]byte, len(h.messages))
	copy(out, h.messages)
	return out
}

func (f *fakeRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(f.body)),
		Header:     make(http.Header),
	}, nil
}

func (f *manageSubsFixtureConnection) SendJSONMessage(_ context.Context, _ request.EndpointLimit, msg any) error {
	f.sentMessages = append(f.sentMessages, msg)
	return f.sendErr
}

func (f *wsConnectFixtureConnection) Dial(context.Context, *gws.Dialer, http.Header, url.Values) error {
	f.dialCalls.Add(1)
	return f.dialErr
}

func (f *wsConnectFixtureConnection) ReadMessage() websocket.Response {
	f.readCalls.Add(1)
	return websocket.Response{}
}

func (f *wsReadDataFixtureConnection) ReadMessage() websocket.Response {
	if f.idx >= len(f.messages) {
		return websocket.Response{}
	}
	msg := f.messages[f.idx]
	f.idx++
	return websocket.Response{Raw: msg}
}

func (f *wsReadDataFixtureConnection) GetURL() string {
	return "wss://fake-test-url"
}

func fillDataHandlerBuffer(t *testing.T, ex *Exchange) {
	t.Helper()
	for ex.Websocket.DataHandler.Send(t.Context(), "filler") == nil { //nolint:revive // intentionally empty
	}
}

func waitForWaitGroup(t *testing.T, wg *sync.WaitGroup, timeout time.Duration) {
	t.Helper()
	waitDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for websocket waitgroup within %s", timeout)
	}
}

func setupWithWebsocketEnabled(t *testing.T) *Exchange {
	t.Helper()

	cfg := &config.Config{}
	root, err := testutils.RootPathFromCWD()
	require.NoError(t, err, "RootPathFromCWD must not error")
	require.NoError(t, cfg.LoadConfig(filepath.Join(root, "testdata", "configtest.json"), true), "LoadConfig must not error")

	ex := new(Exchange)
	ex.SetDefaults()

	exchConf, err := cfg.GetExchangeConfig(ex.GetName())
	require.NoError(t, err, "GetExchangeConfig must not error")
	exchConf.Features.Enabled.Websocket = true

	b := ex.GetBase()
	b.Websocket = sharedtestvalues.NewTestWebsocket()

	require.NoError(t, ex.Setup(exchConf), "Setup must not error")

	b.Accounts = accounts.MustNewAccounts(b)

	return ex
}

//nolint:unparam // timeout is intentionally configurable for future test cases
func waitForCondition(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for condition")
}

func TestManageSubsOrderbookDepthZero(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()

	conn := &manageSubsFixtureConnection{}
	ex.Websocket.Conn = conn

	subs := subscription.List{
		{
			Enabled: true,
			Asset:   asset.Spot,
			Channel: subscription.OrderbookChannel,
			Pairs:   currency.Pairs{testPair},
			// Levels intentionally left unset (zero value) to simulate the bug scenario
		},
	}

	err := ex.manageSubs(t.Context(), subs, lbankWsSubscribe)
	require.NoError(t, err, "manageSubs must not error")
	require.Len(t, conn.sentMessages, 1, "manageSubs must send one message")

	req, ok := conn.sentMessages[0].(map[string]any)
	require.True(t, ok, "sent message must be a map[string]any")

	depth, ok := req["depth"].(string)
	require.True(t, ok, "depth field must be a string")
	assert.NotEqual(t, "0", depth, "depth should not be sent as 0 — orderbook subscriptions with 0 depth return no data")
}

func TestManageSubsMyOrdersChannelSendFailure(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.ws.subscribeKey = "fake-key"

	conn := &manageSubsFixtureConnection{sendErr: errors.New("send failed")}
	ex.Websocket.Conn = conn

	subs := subscription.List{
		{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true},
	}
	err := ex.manageSubs(t.Context(), subs, lbankWsSubscribe)
	assert.Error(t, err, "manageSubs should return an error when SendJSONMessage fails")
}

func TestManageSubsTickerChannelSendFailure(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()

	conn := &manageSubsFixtureConnection{sendErr: errors.New("send failed")}
	ex.Websocket.Conn = conn

	subs := subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{testPair}},
	}
	err := ex.manageSubs(t.Context(), subs, lbankWsSubscribe)
	assert.Error(t, err, "manageSubs should return an error when SendJSONMessage fails")
}

func TestManageSubsEmptyPairsSkipsBookkeeping(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()

	conn := &manageSubsFixtureConnection{}
	ex.Websocket.Conn = conn

	subs := subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{}},
	}
	err := ex.manageSubs(t.Context(), subs, lbankWsSubscribe)
	assert.NoError(t, err, "manageSubs should not error when a channel has no pairs")
	assert.Empty(t, conn.sentMessages, "no message should be sent when there are no pairs")
}

func TestManageSubsMyOrdersChannelUnsubscribeWithoutSubscribe(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.ws.subscribeKey = "fake-key"

	conn := &manageSubsFixtureConnection{}
	ex.Websocket.Conn = conn

	subs := subscription.List{
		{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true},
	}
	// Unsubscribing without ever having subscribed should surface a bookkeeping error
	err := ex.manageSubs(t.Context(), subs, lbankWsUnsubscribe)
	assert.Error(t, err, "manageSubs should surface an error when removing a subscription that was never added")
}

func TestWsHandleDataPingPong(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()

	conn := &manageSubsFixtureConnection{}
	ex.Websocket.Conn = conn

	err := ex.wsHandleData(t.Context(), []byte(`{"action":"ping","ping":"12345"}`))
	require.NoError(t, err, "wsHandleData must not error on ping")
	require.Len(t, conn.sentMessages, 1, "wsHandleData must send one pong reply")

	sent, ok := conn.sentMessages[0].(map[string]string)
	require.True(t, ok, "sent message must be a map[string]string")
	assert.Equal(t, lbankWsPong, sent[lbankWsAction], "reply action should be pong")
	assert.Equal(t, "12345", sent["pong"], "pong value should echo the ping value")
}

func TestWsHandleKbar(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	ex.Name = t.Name()

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "kbar",
		"pair": "btc_usdt",
		"kbar": {"o": "29000.0","h": "29500.0","l": "28800.0","c": "29200.0","v": "100.5","t": "2026-07-07T12:30:00.000","slot": "1min"},
		"SERVER": "V2",
		"TS": "2026-07-07T12:30:00.000"
	}`))
	require.NoError(t, err, "wsHandleData kbar must not error")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "wsHandleData must send one kbar update")
	exp := kline.Item{
		Exchange: ex.Name,
		Pair:     currency.NewPairWithDelimiter("btc", "usdt", "_"),
		Asset:    asset.Spot,
		Interval: kline.OneMin,
		Candles: []kline.Candle{{
			Time:   time.Date(2026, 7, 7, 12, 30, 0, 0, wsTimeLocation).UTC(),
			Open:   29000.0,
			High:   29500.0,
			Low:    28800.0,
			Close:  29200.0,
			Volume: 100.5,
		}},
	}
	assert.Equal(t, exp, (<-ex.Websocket.DataHandler.C).Data, "wsHandleData should send the expected kbar")

	t.Run("invalid interval", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{
			"type": "kbar",
			"pair": "btc_usdt",
			"kbar": {"o": "100","h": "110","l": "90","c": "105","v": "50","t": "2026-07-07T12:30:00.000","slot": "invalid"}
		}`))
		assert.Error(t, err, "invalid interval should return error")
	})

	t.Run("mismatched kbar type", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{"type":"kbar","pair":"btc_usdt","kbar":[1,2,3]}`))
		assert.Error(t, err, "mismatched kbar type should return error")
	})

	t.Run("malformed JSON", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{"type":"kbar","pair":"btc_usdt","kbar":!!!}`))
		assert.Error(t, err, "malformed JSON should return error")
	})
}

func TestWebsocketTimeUnmarshal(t *testing.T) {
	t.Parallel()

	t.Run("valid timestamp", func(t *testing.T) {
		t.Parallel()
		var wt websocketTime
		err := json.Unmarshal([]byte(`"2026-09-06T13:29:00.000"`), &wt)
		require.NoError(t, err)
		assert.False(t, wt.Time().IsZero(), "time should not be zero")
	})

	t.Run("empty string", func(t *testing.T) {
		t.Parallel()
		var wt websocketTime
		err := json.Unmarshal([]byte(`""`), &wt)
		assert.NoError(t, err, "empty string should not error")
	})

	t.Run("invalid timestamp", func(t *testing.T) {
		t.Parallel()
		var wt websocketTime
		err := json.Unmarshal([]byte(`"not-a-time"`), &wt)
		assert.ErrorIs(t, err, errInvalidWebsocketTime, "should return errInvalidWebsocketTime")
	})

	t.Run("non-string JSON", func(t *testing.T) {
		t.Parallel()
		var wt websocketTime
		err := json.Unmarshal([]byte(`12345`), &wt)
		assert.Error(t, err, "number should return error")
	})
}

func TestWsHandleOrderUpdate(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	err := ex.wsHandleData(t.Context(), []byte(`{"orderUpdate":{"accAmt":"0.4","amount":"0.2","avgPrice":"99.5","orderAmt":"1","orderPrice":"100","orderStatus":1,"price":"99","remainAmt":"0.6","type":"buy","updateTime":1704067200000,"uuid":"test-order-uuid"},"pair":"btc_usdt","type":"orderUpdate","SERVER":"V2","TS":"2024-01-01T08:00:00.000"}`))
	require.NoError(t, err, "wsHandleData must not error")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "wsHandleData must send one order update")
	exp := &order.Detail{
		Exchange:             ex.Name,
		AssetType:            asset.Spot,
		Pair:                 currency.NewPairWithDelimiter("btc", "usdt", "_"),
		Price:                100,
		Amount:               1,
		ExecutedAmount:       0.4,
		RemainingAmount:      0.6,
		AverageExecutedPrice: 99.5,
		Side:                 order.Buy,
		OrderID:              "test-order-uuid",
		Status:               order.PartiallyFilled,
		LastUpdated:          time.UnixMilli(1704067200000),
	}
	assert.Equal(t, exp, (<-ex.Websocket.DataHandler.C).Data, "wsHandleData should send the expected order detail")

	t.Run("invalid order status", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{
			"type": "orderUpdate",
			"pair": "btc_usdt",
			"orderUpdate": {"orderAmt": "1.0","orderStatus": 99,"orderPrice": "100","type": "buy","updateTime": 1704067200000,"uuid": "test"}
		}`))
		assert.Error(t, err, "invalid order status should return error")
	})

	t.Run("mismatched order update type", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{"type":"orderUpdate","pair":"btc_usdt","orderUpdate":[1,2,3]}`))
		assert.Error(t, err, "mismatched orderUpdate type should return error")
	})

	t.Run("invalid order side", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "orderUpdate",
		"pair": "btc_usdt",
		"orderUpdate": {"orderAmt": "1.0","orderStatus": 2,"orderPrice": "100","type": "notaside","updateTime": 1704067200000,"uuid": "test"}
	}`))
		assert.ErrorIs(t, err, order.ErrSideIsInvalid, "invalid order side should return order.ErrSideIsInvalid")
	})

	t.Run("malformed JSON", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{"type":"orderUpdate","pair":"btc_usdt","orderUpdate":!!!}`))
		assert.Error(t, err, "malformed JSON should return error")
	})
}

func TestWsHandleOrderUpdateCompoundTypes(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()

	types := []string{"buy", "sell", "buy_market", "sell_market", "buy_maker", "sell_maker", "buy_ioc", "sell_ioc", "buy_fok", "sell_fok"}
	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			t.Parallel()
			err := ex.wsHandleData(t.Context(), fmt.Appendf(nil, `{
				"type": "orderUpdate",
				"pair": "btc_usdt",
				"orderUpdate": {"orderAmt": "5","orderStatus": 0,"orderPrice": "0.009834","type": %q,"updateTime": 1705676718532,"uuid": "test"}
			}`, typ))
			assert.NoError(t, err, "type %s should parse successfully", typ)
		})
	}
}

func TestLbankOrderStatusToOrderStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    int64
		expected order.Status
		wantErr  bool
	}{
		{-1, order.Cancelled, false},
		{0, order.New, false},
		{1, order.PartiallyFilled, false},
		{2, order.Filled, false},
		{3, order.PartiallyCancelled, false},
		{4, order.PendingCancel, false},
		{99, order.UnknownStatus, true},
	}
	for _, tt := range tests {
		status, err := lbankOrderStatusToOrderStatus(tt.input)
		if tt.wantErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, status)
		}
	}
}

func TestKlineIntervalFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected kline.Interval
		wantErr  bool
	}{
		{lbankWsKline1Min, kline.OneMin, false},
		{lbankWsKline5Min, kline.FiveMin, false},
		{lbankWsKline15Min, kline.FifteenMin, false},
		{lbankWsKline30Min, kline.ThirtyMin, false},
		{lbankWsKline1Hr, kline.OneHour, false},
		{lbankWsKline4Hr, kline.FourHour, false},
		{lbankWsKlineDay, kline.OneDay, false},
		{lbankWsKlineWeek, kline.OneWeek, false},
		{lbankWsKlineMonth, kline.OneMonth, false},
		{lbankWsKlineYear, kline.OneYear, false},
		{"invalid", 0, true},
	}
	for _, tt := range tests {
		interval, err := klineIntervalFromString(tt.input)
		if tt.wantErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, interval)
		}
	}
}

func TestWsHandleDataServerError(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	err := ex.wsHandleData(t.Context(), []byte(`{
		"SERVER": "V2",
		"message": "Missing parameter ['pair']",
		"status": "error",
		"TS": "2021-07-26T19:48:03.270"
	}`))
	assert.Error(t, err, "server error message should return error")
	assert.Contains(t, err.Error(), "Missing parameter")
}

func TestWsHandleData(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	t.Run("unknown type returns no error", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{"type":"unknown","pair":"eth_usdt"}`))
		assert.NoError(t, err)
	})

	t.Run("missing type returns no error", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{"pair":"eth_usdt"}`))
		assert.NoError(t, err)
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`not json`))
		assert.Error(t, err)
	})
}

func TestWsHandleTicker(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "tick",
		"pair": "eth_usdt",
		"tick": {"high": "2149.67","low": "2010.26","latest": "2124.36","vol": "51774.0345"},
		"SERVER": "V2",
		"TS": "2024-01-01T00:00:00.000"
	}`))
	require.NoError(t, err, "wsHandleData ticker must not error")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "wsHandleData must send one ticker update")
	exp := &ticker.Price{
		ExchangeName: ex.Name,
		Pair:         currency.NewPairWithDelimiter("eth", "usdt", "_"),
		AssetType:    asset.Spot,
		High:         2149.67,
		Low:          2010.26,
		Last:         2124.36,
		Volume:       51774.0345,
		LastUpdated:  time.Date(2024, 1, 1, 0, 0, 0, 0, wsTimeLocation).UTC(),
	}
	assert.Equal(t, exp, (<-ex.Websocket.DataHandler.C).Data, "wsHandleData should send the expected ticker")

	t.Run("malformed JSON", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{"type":"tick","pair":"btc_usdt","tick":!!!}`))
		assert.Error(t, err, "malformed JSON should return error")
	})

	t.Run("mismatched tick type", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{"type":"tick","pair":"btc_usdt","tick":[1,2,3]}`))
		assert.Error(t, err, "mismatched tick type should return error")
	})
}

func TestWsHandleAssetUpdate(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.Accounts = accounts.MustNewAccounts(ex.GetBase())
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: testAPIKey, Secret: testAPISecret})

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "assetUpdate",
		"data": {
			"asset": "114548.31881315",
			"assetCode": "usdt",
			"free": "97430.6739041",
			"freeze": "17117.64490905",
			"time": 1627300043270,
			"type": "ORDER_CREATE"
		},
		"SERVER": "V2",
		"TS": "2021-07-26T19:48:03.270"
	}`))
	require.NoError(t, err, "wsHandleData assetUpdate must not error")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "wsHandleData must send one asset update")
	exp := accounts.Change{
		AssetType: asset.Spot,
		Balance: accounts.Balance{
			Currency: currency.NewCode("usdt"),
			Total:    114548.31881315,
			Free:     97430.6739041,
			Hold:     17117.64490905,
		},
	}
	assert.Equal(t, exp, (<-ex.Websocket.DataHandler.C).Data, "wsHandleData should send the expected asset update")

	t.Run("mismatched data type", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{"type":"assetUpdate","data":[1,2,3]}`))
		assert.Error(t, err, "mismatched data type should return error")
	})
}

func TestWsReadData(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()

	conn := &wsReadDataFixtureConnection{
		messages: [][]byte{
			[]byte(`{"type":"unknown","pair":"eth_usdt"}`), // valid message, unhandled type
			[]byte(`not json`), // invalid JSON, error forwarded
		},
	}
	ex.Websocket.Conn = conn

	ex.Websocket.Wg.Add(1)
	ex.wsReadData(t.Context())

	require.Len(t, ex.Websocket.DataHandler.C, 2, "wsReadData must forward two messages to DataHandler")

	first := <-ex.Websocket.DataHandler.C
	_, ok := first.Data.(websocket.UnhandledMessageWarning)
	assert.True(t, ok, "first message should be an UnhandledMessageWarning")

	second := <-ex.Websocket.DataHandler.C
	sentErr, ok := second.Data.(error)
	require.True(t, ok, "second message must be an error")
	assert.Error(t, sentErr, "second message's error should be non-nil")
}

func TestWsHandleTradesFeedDisabled(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	// Neither trade feed nor save-trade-data enabled — should be a no-op
	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "trade",
		"pair": "eth_usdt",
		"trade": {"volume": "0.5","price": "2100.0","direction": "buy","TS": "2026-07-07T12:30:00.000"}
	}`))
	assert.NoError(t, err, "wsHandleData should not error when both trade feed and save trade data are disabled")
}

func TestWsHandleTradesFeedOnly(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.SetTradeFeedStatus(true)
	// SaveTradeData intentionally left disabled

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "trade",
		"pair": "eth_usdt",
		"trade": {"volume": "0.5","price": "2100.0","direction": "buy","TS": "2026-07-07T12:30:00.000"}
	}`))
	require.NoError(t, err, "wsHandleData must not error")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "trade must be sent via DataHandler when trade feed is enabled")
	<-ex.Websocket.DataHandler.C
}

func TestWsHandleTradesDataHandlerSendFailure(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.SetTradeFeedStatus(true)

	// Fill the DataHandler's buffer completely without draining it,
	// so the next Send call hits the full-buffer error path.
	fillDataHandlerBuffer(t, ex)

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "trade",
		"pair": "eth_usdt",
		"trade": {"volume": "0.5","price": "2100.0","direction": "buy","TS": "2026-07-07T12:30:00.000"}
	}`))
	assert.Error(t, err, "wsHandleData should return an error when DataHandler's buffer is full")
	assert.Contains(t, err.Error(), "buffer is full", "error should indicate the channel buffer is full")
}

func TestWsHandleTradesSaveOnly(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.SetSaveTradeDataStatus(true)
	// TradeFeed intentionally left disabled

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "trade",
		"pair": "eth_usdt",
		"trade": {"volume": "0.5","price": "2100.0","direction": "buy","TS": "2026-07-07T12:30:00.000"}
	}`))
	require.NoError(t, err, "wsHandleData must not error")
	assert.Empty(t, ex.Websocket.DataHandler.C, "trade should not be sent via DataHandler when trade feed is disabled")
}

func TestWsHandleTrades(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.SetSaveTradeDataStatus(true)
	ex.SetTradeFeedStatus(true)

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "trade",
		"pair": "eth_usdt",
		"trade": {"volume": "0.5","price": "2100.0","direction": "buy","TS": "2026-07-07T12:30:00.000"},
		"SERVER": "V2",
		"TS": "2021-07-26T19:48:03.270"
	}`))
	require.NoError(t, err, "wsHandleData trades must not error")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "wsHandleData must send one trade update")
	exp := trade.Data{
		Exchange:     ex.Name,
		AssetType:    asset.Spot,
		CurrencyPair: currency.NewPairWithDelimiter("eth", "usdt", "_"),
		Price:        2100.0,
		Amount:       0.5,
		Timestamp:    time.Date(2026, 7, 7, 12, 30, 0, 0, wsTimeLocation).UTC(),
		Side:         order.Buy,
	}
	assert.Equal(t, exp, (<-ex.Websocket.DataHandler.C).Data, "wsHandleData should send the expected trade")

	t.Run("invalid direction", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "trade",
		"pair": "btc_usdt",
		"trade": {"volume": "0.5","price": "100","direction": "invalid","TS": "2026-07-07T12:30:00.000"}
	}`))
		assert.ErrorIs(t, err, order.ErrSideIsInvalid, "invalid direction should return order.ErrSideIsInvalid")
	})

	t.Run("compound direction buy_market", func(t *testing.T) {
		t.Parallel()
		compoundEx := new(Exchange)
		require.NoError(t, testexch.Setup(compoundEx), "Setup must not error")
		compoundEx.SetTradeFeedStatus(true)
		err := compoundEx.wsHandleData(t.Context(), []byte(`{
		"type": "trade",
		"pair": "sui_usdt",
		"trade": {"volume": "19.0","price": "0.7054","direction": "buy_market","TS": "2026-09-14T09:12:08.322"}
	}`))
		assert.NoError(t, err, "compound direction buy_market should parse as BUY")
	})

	t.Run("mismatched trade type", func(t *testing.T) {
		t.Parallel()
		mismatchEx := new(Exchange)
		require.NoError(t, testexch.Setup(mismatchEx), "Setup must not error")
		mismatchEx.SetTradeFeedStatus(true)
		err := mismatchEx.wsHandleData(t.Context(), []byte(`{"type":"trade","pair":"btc_usdt","trade":[1,2,3]}`))
		assert.Error(t, err, "mismatched trade type should return error")
	})

	t.Run("malformed JSON", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{"type":"trade","pair":"btc_usdt","trade":!!!}`))
		assert.Error(t, err, "malformed JSON should return error")
	})
}

func TestWsHandleOrderbook(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	ex.Name = t.Name()

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "depth",
		"pair": "btc_usdt",
		"depth": {
			"asks": [
				["2125.0", "1.5"],
				["2126.0", "2.0"]
			],
			"bids": [
				["2124.0", "1.0"],
				["2123.0", "3.0"]
			]
		},
		"SERVER": "V2",
		"TS": "2024-01-01T00:00:00.000"
	}`))
	require.NoError(t, err, "wsHandleData orderbook must not error")

	p, err := currency.NewPairFromString("btc_usdt")
	require.NoError(t, err)
	ob, err := orderbook.Get(ex.Name, p, asset.Spot)
	require.NoError(t, err, "orderbook.Get must not error")
	assert.Len(t, ob.Asks, 2, "orderbook should have 2 asks")
	assert.Len(t, ob.Bids, 2, "orderbook should have 2 bids")

	t.Run("mismatched depth type", func(t *testing.T) {
		t.Parallel()
		err := ex.wsHandleData(t.Context(), []byte(`{"type":"depth","pair":"btc_usdt","depth":[1,2,3]}`))
		assert.Error(t, err, "mismatched depth type should return error")
	})
}

func TestManageSubsUnsupportedChannel(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()

	conn := &manageSubsFixtureConnection{}
	ex.Websocket.Conn = conn

	subs := subscription.List{
		{Enabled: true, Channel: "not-a-real-channel"},
	}
	err := ex.manageSubs(t.Context(), subs, lbankWsSubscribe)
	assert.Error(t, err, "manageSubs should error on an unsupported channel")
	assert.Empty(t, conn.sentMessages, "no message should be sent for an unsupported channel")
}

func TestManageSubsMyOrdersChannel(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.ws.subscribeKey = "fake-key"

	conn := &manageSubsFixtureConnection{}
	ex.Websocket.Conn = conn

	subs := subscription.List{
		{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true},
	}
	err := ex.manageSubs(t.Context(), subs, lbankWsSubscribe)
	require.NoError(t, err, "manageSubs must not error")
	require.Len(t, conn.sentMessages, 1, "manageSubs must send one message")

	req, ok := conn.sentMessages[0].(map[string]any)
	require.True(t, ok, "sent message must be a map[string]any")
	assert.Equal(t, lbankWsSubscribe, req[lbankWsAction])
	assert.Equal(t, lbankWsOrderUpdate, req["subscribe"])
	assert.Equal(t, "fake-key", req["subscribeKey"])
	assert.Equal(t, "all", req["pair"])
}

func TestManageSubsMyAccountChannel(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.ws.subscribeKey = "fake-key"

	conn := &manageSubsFixtureConnection{}
	ex.Websocket.Conn = conn

	subs := subscription.List{
		{Enabled: true, Channel: subscription.MyAccountChannel, Authenticated: true},
	}
	err := ex.manageSubs(t.Context(), subs, lbankWsSubscribe)
	require.NoError(t, err, "manageSubs must not error")
	require.Len(t, conn.sentMessages, 1, "manageSubs must send one message")

	req, ok := conn.sentMessages[0].(map[string]any)
	require.True(t, ok, "sent message must be a map[string]any")
	assert.Equal(t, lbankWsSubscribe, req[lbankWsAction])
	assert.Equal(t, lbankWsAssetUpdate, req["subscribe"])
	assert.Equal(t, "fake-key", req["subscribeKey"])
}

func TestManageSubsCandlesChannel(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()

	conn := &manageSubsFixtureConnection{}
	ex.Websocket.Conn = conn

	subs := subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin, Pairs: currency.Pairs{testPair}},
	}
	err := ex.manageSubs(t.Context(), subs, lbankWsSubscribe)
	require.NoError(t, err, "manageSubs must not error")
	require.Len(t, conn.sentMessages, 1, "manageSubs must send one message")

	req, ok := conn.sentMessages[0].(map[string]any)
	require.True(t, ok, "sent message must be a map[string]any")
	assert.Equal(t, lbankWsKbar, req["subscribe"])
	assert.Equal(t, lbankWsKline1Min, req["kbar"])
}

func TestManageSubsCandlesChannelUnsupportedInterval(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()

	conn := &manageSubsFixtureConnection{}
	ex.Websocket.Conn = conn

	subs := subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.ThreeMin, Pairs: currency.Pairs{testPair}},
	}
	err := ex.manageSubs(t.Context(), subs, lbankWsSubscribe)
	assert.Error(t, err, "manageSubs should error on an unsupported kline interval")
	assert.Empty(t, conn.sentMessages, "no message should be sent for an unsupported interval")
}

func TestManageSubsTickerChannelDefault(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()

	conn := &manageSubsFixtureConnection{}
	ex.Websocket.Conn = conn

	subs := subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{testPair}},
	}
	err := ex.manageSubs(t.Context(), subs, lbankWsSubscribe)
	require.NoError(t, err, "manageSubs must not error")
	require.Len(t, conn.sentMessages, 1, "manageSubs must send one message")

	req, ok := conn.sentMessages[0].(map[string]any)
	require.True(t, ok, "sent message must be a map[string]any")
	assert.Equal(t, lbankWsTicker, req["subscribe"])
	assert.Equal(t, testPair.Lower().String(), req["pair"])
}

func TestWsHandleOrderUpdateDataHandlerSendFailure(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()

	fillDataHandlerBuffer(t, ex)

	err := ex.wsHandleData(t.Context(), []byte(`{"orderUpdate":{"accAmt":"0.4","avgPrice":"99.5","orderAmt":"1","orderPrice":"100","orderStatus":1,"remainAmt":"0.6","type":"buy","updateTime":1704067200000,"uuid":"test-order-uuid"},"pair":"btc_usdt","type":"orderUpdate"}`))
	assert.Error(t, err, "wsHandleData should return an error when DataHandler's buffer is full")
}

func TestWsHandleKbarDataHandlerSendFailure(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()

	fillDataHandlerBuffer(t, ex)

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "kbar",
		"pair": "btc_usdt",
		"kbar": {"o": "29000.0","h": "29500.0","l": "28800.0","c": "29200.0","v": "100.5","t": "2026-07-07T12:30:00.000","slot": "1min"}
	}`))
	assert.Error(t, err, "wsHandleData should return an error when DataHandler's buffer is full")
}

func TestWsReadDataSendFailureLogsError(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()

	fillDataHandlerBuffer(t, ex)

	conn := &wsReadDataFixtureConnection{
		messages: [][]byte{[]byte(`not json`)},
	}
	ex.Websocket.Conn = conn

	ex.Websocket.Wg.Add(1)
	ex.wsReadData(t.Context())
	// No assertion beyond "did not panic" — the error is only logged, not observable via DataHandler
	// since the buffer is intentionally full for this test.
}

func TestGetWebsocketSubscribeKeyEmptyData(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: testAPIKey, Secret: testAPISecret})

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "GenerateKey must not error")
	ex.privateKey = key

	require.NoError(t, ex.SetHTTPClient(&http.Client{
		Transport: &fakeRoundTripper{body: `{"result":"true","data":"","error_code":0}`},
	}), "SetHTTPClient must not error")

	_, err = ex.GetWebsocketSubscribeKey(t.Context())
	assert.ErrorIs(t, err, errEmptySubscribeKey, "empty data field should return errEmptySubscribeKey")
}

func TestWsHandleAssetUpdateSavesAccountBalance(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.Accounts = accounts.MustNewAccounts(ex.GetBase())
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: testAPIKey, Secret: testAPISecret})

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "assetUpdate",
		"data": {
			"asset": "114548.31881315",
			"assetCode": "usdt",
			"free": "97430.6739041",
			"freeze": "17117.64490905",
			"time": 1627300043270,
			"type": "ORDER_CREATE"
		},
		"SERVER": "V2",
		"TS": "2021-07-26T19:48:03.270"
	}`))
	require.NoError(t, err, "wsHandleData assetUpdate must not error")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "wsHandleData must send one asset update")
	<-ex.Websocket.DataHandler.C

	// Confirm the balance was actually persisted, not just forwarded to DataHandler
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")

	bal, err := ex.Accounts.GetBalance("", creds, asset.Spot, currency.NewCode("usdt"))
	require.NoError(t, err, "GetBalance must not error")
	assert.Equal(t, 114548.31881315, bal.Total, "saved balance Total should match the assetUpdate")
	assert.Equal(t, 97430.6739041, bal.Free, "saved balance Free should match the assetUpdate")
	assert.Equal(t, 17117.64490905, bal.Hold, "saved balance Hold should match the assetUpdate")
}

func TestWsHandleAssetUpdateSaveFailure(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	// ex.Accounts intentionally left nil — Save should fail via common.NilGuard

	err := ex.wsHandleData(t.Context(), []byte(`{
		"type": "assetUpdate",
		"data": {
			"asset": "100",
			"assetCode": "usdt",
			"free": "100",
			"freeze": "0",
			"time": 1627300043270,
			"type": "ORDER_CREATE"
		},
		"SERVER": "V2",
		"TS": "2021-07-26T19:48:03.270"
	}`))
	assert.Error(t, err, "wsHandleData should return an error when Accounts.Save fails")
}
