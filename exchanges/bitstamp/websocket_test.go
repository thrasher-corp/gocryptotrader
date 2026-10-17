package bitstamp

import (
	"bytes"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	testutils "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetWebsocketToken(t *testing.T) {
	t.Parallel()
	skipIfLiveWithoutCredentials(t)
	token, err := e.GetWebsocketToken(t.Context())
	require.NoError(t, err, "GetWebsocketToken must not error")
	if mockTests {
		assert.Equal(t, &WebsocketTokenResponse{Token: "vRQqHEFoxp23seihJnbhPGt0lgBG5EFe", ValidSeconds: 60, UserID: 314159}, token, "GetWebsocketToken should decode the token")
		return
	}
	assert.Len(t, token.Token, 32, "Token should be 32 characters")
	assert.Positive(t, token.UserID, "UserID should be positive")
	assert.Positive(t, token.ValidSeconds, "ValidSeconds should be positive")
}

func TestShutdownDoesNotReconnect(t *testing.T) {
	t.Parallel()
	var upgrades atomic.Int64
	h := func(w http.ResponseWriter, r *http.Request) {
		if !gws.IsWebSocketUpgrade(r) {
			// Connecting makes no REST requests, so anything but the websocket upgrade is refused
			http.NotFound(w, r)
			return
		}
		upgrades.Add(1)
		mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
	}
	e := testexch.MockWsInstance[Exchange](t, h)
	require.True(t, e.Websocket.IsConnected(), "websocket must be connected")
	before := upgrades.Load()

	require.NoError(t, e.Shutdown(), "Shutdown must not error")
	assert.False(t, e.Websocket.IsConnected(), "websocket should be disconnected after Shutdown")
	assert.Never(t, func() bool {
		return upgrades.Load() != before || e.Websocket.IsConnected()
	}, 3*time.Second, 10*time.Millisecond, "a shut down exchange should not reconnect")
}

var (
	wsSpotPair      = currency.NewPairWithDelimiter("BTC", "USD", "/")
	wsPerpetualPair = currency.NewPairWithDelimiter("BTC", "USD-PERP", "/")
)

// newWebsocketTestInstance returns an exchange instance named after the test, so the orderbooks it stores do not
// collide with those of other tests
func newWebsocketTestInstance(t *testing.T) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	return ex
}

// newFeedTestInstance returns an exchange instance set up from the test configuration with the trade and fills feeds
// enabled, so the feeds are configured by Setup as they are for a running bot
func newFeedTestInstance(t *testing.T) *Exchange {
	t.Helper()
	root, err := testutils.RootPathFromCWD()
	require.NoError(t, err, "RootPathFromCWD must not error")
	cfg := &config.Config{}
	require.NoError(t, cfg.LoadConfig(filepath.Join(root, "testdata", "configtest.json"), true), "LoadConfig must not error")
	exchCfg, err := cfg.GetExchangeConfig("Bitstamp")
	require.NoError(t, err, "GetExchangeConfig must not error")
	exchCfg.Features.Enabled.TradeFeed = true
	exchCfg.Features.Enabled.FillsFeed = true
	ex := new(Exchange)
	ex.SetDefaults()
	ex.Websocket = sharedtestvalues.NewTestWebsocket()
	require.NoError(t, ex.Setup(exchCfg), "Setup must not error")
	ex.Accounts = accounts.MustNewAccounts(ex)
	ex.Name = t.Name()
	return ex
}

// dataHandlerPayloads closes the data handler and returns everything sent to it
func dataHandlerPayloads(ex *Exchange) []any {
	// The totals of deleted orders are relayed off the read loop, so they must be waited for before the relay is closed
	ex.orderTotalFetches.Wait()
	ex.Websocket.DataHandler.Close()
	var payloads []any
	for p := range ex.Websocket.DataHandler.C {
		payloads = append(payloads, p.Data)
	}
	return payloads
}

// newWebsocketServer returns the URL of a websocket server which sends msgs to each connection and then waits for the
// client to close it
func newWebsocketServer(t *testing.T, msgs ...string) string {
	t.Helper()
	var upgrader gws.Upgrader
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if !assert.NoError(t, err, "Upgrade should not error") {
			return
		}
		defer c.Close()
		for _, msg := range msgs {
			assert.NoError(t, c.WriteMessage(gws.TextMessage, []byte(msg)), "WriteMessage should not error")
		}
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}))
	t.Cleanup(s.Close)
	return "ws" + strings.TrimPrefix(s.URL, "http")
}

// receiveError returns the next error sent to the data handler
func receiveError(t *testing.T, ex *Exchange) error {
	t.Helper()
	select {
	case p := <-ex.Websocket.DataHandler.C:
		err, ok := p.Data.(error)
		require.Truef(t, ok, "the data handler must receive an error, not %T", p.Data)
		return err
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the data handler must receive an error")
		return nil
	}
}

// waitForReadLoop waits for wsReadData to return once its connection is closed, failing rather than hanging when it
// does not
func waitForReadLoop(t *testing.T, ex *Exchange) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		ex.Websocket.Wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "wsReadData must return once the connection is closed")
	}
}

func TestWebsocketConnect(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	ex.SetEnabled(false)
	assert.ErrorIs(t, ex.WebsocketConnect(), websocket.ErrWebsocketNotEnabled, "WebsocketConnect should error when the exchange is disabled")

	ex = newWebsocketTestInstance(t)
	require.NoError(t, ex.Websocket.SetAllConnectionURLs(newWebsocketServer(t, `{"event":"bts:heartbeat","channel":"","data":{"status":"success"}}`, `{"channel":"live_trades_btcusd"}`)), "SetAllConnectionURLs must not error")
	require.NoError(t, ex.WebsocketConnect(), "WebsocketConnect must not error")
	assert.ErrorIs(t, receiveError(t, ex), common.ErrParsingWSField, "WebsocketConnect should start reading messages")

	require.NoError(t, ex.Websocket.Conn.Shutdown(), "Shutdown must not error")
	close(ex.Websocket.ShutdownC) // Stops the heartbeat
	waitForReadLoop(t, ex)
}

func TestWebsocketReadData(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	require.NoError(t, ex.Websocket.SetAllConnectionURLs(newWebsocketServer(t, `{"channel":"live_trades_btcusd"}`)), "SetAllConnectionURLs must not error")
	require.NoError(t, ex.Websocket.Conn.Dial(t.Context(), &gws.Dialer{}, http.Header{}, nil), "Dial must not error")
	ex.Websocket.Wg.Add(1)
	go ex.wsReadData(t.Context())

	assert.ErrorIs(t, receiveError(t, ex), common.ErrParsingWSField, "wsReadData should send handling errors to the data handler")
	require.NoError(t, ex.Websocket.Conn.Shutdown(), "Shutdown must not error")
	waitForReadLoop(t, ex)
}

func TestWebsocketHandleDataErrors(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	for _, tc := range []struct {
		name string
		msg  string
		err  error
	}{
		{name: "missing event", msg: `{"channel":"live_trades_btcusd","data":{}}`, err: common.ErrParsingWSField},
		{name: "missing channel", msg: `{"event":"trade","data":{}}`, err: common.ErrParsingWSField},
		{name: "malformed channel", msg: `{"event":"trade","channel":"livetrades","data":{}}`, err: errMalformedChannel},
		{name: "unknown channel market", msg: `{"event":"trade","channel":"live_trades","data":{}}`, err: errParsingWSPair},
		{name: "missing data", msg: `{"event":"trade","channel":"live_trades_btcusd"}`, err: common.ErrParsingWSField},
		{name: "unknown market", msg: `{"event":"trade","channel":"live_trades_btcwibble","data":{}}`, err: errParsingWSPair},
		{name: "unrequested subscription", msg: `{"event":"bts:subscription_succeeded","channel":"live_trades_btcusd","data":{}}`, err: websocket.ErrSignatureNotMatched},
		{name: "server error", msg: `{"event":"bts:error","channel":"","data":{"code":4009,"message":"Connection is unauthorized."}}`, err: errWebsocketError},
		{name: "malformed liquidation", msg: `{"event":"liquidation_start","channel":"private-my_liquidations-314159","data":{"initial_margin_ratio":[]}}`},
		{name: "malformed token settlement", msg: `{"event":"token_ready","channel":"private-my_token_settlements-314159","data":{"order_id":"wibble"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ex.wsHandleData(t.Context(), []byte(tc.msg))
			if tc.err == nil {
				assert.Error(t, err, "wsHandleData should error")
				return
			}
			assert.ErrorIs(t, err, tc.err, "wsHandleData should return the correct error")
		})
	}
}

func TestWebsocketHandleDataIgnoredEvents(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	for _, msg := range []string{
		`{"event":"bts:heartbeat","channel":"","data":{"status":"success"}}`,
	} {
		assert.NoErrorf(t, ex.wsHandleData(t.Context(), []byte(msg)), "wsHandleData should not error for %s", msg)
	}
	assert.Empty(t, dataHandlerPayloads(ex), "heartbeats should not be sent to the data handler")
}

func TestWebsocketHandleDataUnhandled(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	for _, msg := range []string{
		`{"event":"data","channel":"diff_order_book_btcusd","data":{}}`,
		`{"event":"settlement","channel":"my_settlements","data":{}}`,
		`{"event":"order_created","channel":"live_orders_btcusd-perp","data":{"id":2054076732301440,"order_type":0,"order_subtype":5,"amount_str":"0.00134","price_str":"75609"}}`,
	} {
		assert.NoErrorf(t, ex.wsHandleData(t.Context(), []byte(msg)), "wsHandleData should not error for %s", msg)
	}
	payloads := dataHandlerPayloads(ex)
	require.Len(t, payloads, 3, "each unhandled message must be sent to the data handler")
	for _, p := range payloads {
		assert.IsType(t, websocket.UnhandledMessageWarning{}, p, "unhandled messages should be sent as warnings")
	}
}

func TestWebsocketHandleDataRouting(t *testing.T) {
	t.Parallel()
	ex := newFeedTestInstance(t)
	for _, msg := range []string{
		`{"data":{"timestamp":"1790318551","microtimestamp":"1790318551908216","bids":[["84010","0.01190"]],"asks":[["84018","0.33440"]]},"channel":"order_book_btcusd-perp","event":"data"}`,
		`{"data":{"id":502711286,"amount_str":"0.01000","price_str":"84010","type":0,"microtimestamp":"1790318552115000"},"channel":"live_trades_btcusd-perp","event":"trade"}`,
		`{"data":{"market":"btcusd-perp","funding_rate":"0.0001","timestamp":"1790318552","next_funding_time":"1790323200"},"channel":"funding_rate_btcusd-perp","event":"funding_rate_saved"}`,
		`{"event":"order_created","channel":"private-my_orders_btcusd-123456","data":{"id":1500000001,"amount_str":"0.5","amount_at_create":"0.5","price_str":"61200","order_type":0,"microtimestamp":"1712131200000000"}}`,
		`{"event":"trade","channel":"private-my_trades_btcusd-perp-123456","data":{"id":296050733,"order_id":1500000001,"amount":0.1,"price":61200.00,"side":"buy","microtimestamp":"1712131200500000"}}`,
		`{"event":"settlement","channel":"private-my_settlements-123456","data":{"id":"1234567890","tnx_id":"123123","ts":"1712131200","ccy":"USD","as":"-1.5"}}`,
		`{"event":"liquidation_start","channel":"private-my_liquidations-123456","data":{"position_id":"1234567890","trade_account_id":0,"microtimestamp":"1712131200000000","alert_type":"liquidation_start","margin_mode":"ISOLATED","initial_margin_ratio":"1.2","maintenance_margin_ratio":"0.9"}}`,
		`{"event":"token_ready","channel":"private-my_token_settlements-123456","data":{"action":"credit","order_id":2046687597465600,"order_id_str":"2046687597465600","instrument":"TSLA/USD","token_security_currency":"TSLA","tokens_quantity":"1.00000000","datetime":"2026-08-25T10:35:09+00:00"}}`,
		`{"event":"published","channel":"announcements","data":{"announcement_id":12345,"announcement_type":"corporate_action","status":"published"}}`,
		`{"data":{"timestamp":"1791848800","microtimestamp":"1791848800102345","bids":[["81730.34","0.01000000","2060344614850561"],["81730.34","0.02000000","2060344614850562"]],"asks":[["81758.04","0.03000000","2060344614850560"]]},"channel":"detail_order_book_btcusd","event":"data"}`,
		`{"event":"self_trade","channel":"private-live_trades_btcusd-123456","data":{"buy_order_id":1500000010,"sell_order_id":1500000011,"amount":0.05,"amount_str":"0.05000000","price":61200.00,"price_str":"61200.00","type":0,"timestamp":"1712131205","microtimestamp":"1712131205000000","sellers_trade_account_id":0,"buyers_trade_account_id":0}}`,
		`{"event":"trade","channel":"private-live_order_trades_btcusd-123456","data":{"id":296050734,"id_str":"296050734","order_id":1500000001,"client_order_id":"my-order-001","amount":0.1,"amount_str":"0.10000000","price":61200.00,"price_str":"61200.00","fee":"0.01","side":"buy","microtimestamp":"1712131200500000","trade_account_id":0,"order_remaining_amount":"0.40000000"}}`,
	} {
		require.NoErrorf(t, ex.wsHandleData(t.Context(), []byte(msg)), "wsHandleData must not error for %s", msg)
	}

	ob, err := ex.Websocket.Orderbook.GetOrderbook(wsPerpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "orderbooks must be stored for their channel's market")
	assert.Equal(t, orderbook.Levels{{Price: 84010, Amount: 0.0119}}, ob.Bids, "Bids should be correct")

	payloads := dataHandlerPayloads(ex)
	require.Len(t, payloads, 15, "every channel must send its payloads to the data handler")
	assert.IsType(t, &orderbook.Depth{}, payloads[0], "orderbooks should be sent to the data handler")
	trades, ok := payloads[1].([]trade.Data)
	require.True(t, ok, "trades must be sent to the data handler")
	assert.Equal(t, wsPerpetualPair, trades[0].CurrencyPair, "trades should be for their channel's market")
	assert.Equal(t, asset.PerpetualContract, trades[0].AssetType, "trades should be for their channel's asset")
	rate, ok := payloads[2].(websocket.FundingData)
	require.True(t, ok, "funding rates must be sent to the data handler")
	assert.Equal(t, wsPerpetualPair, rate.CurrencyPair, "funding rates should be for their channel's market")
	assert.IsType(t, &WebsocketFundingRate{}, payloads[3], "funding updates should be sent to the data handler")
	d, ok := payloads[4].(*order.Detail)
	require.True(t, ok, "orders must be sent to the data handler")
	assert.Equal(t, wsSpotPair, d.Pair, "orders should be for their channel's market")
	assert.Equal(t, asset.Spot, d.AssetType, "orders should be for their channel's asset")
	fills, ok := payloads[5].([]fill.Data)
	require.True(t, ok, "fills must be sent to the data handler")
	assert.Equal(t, wsPerpetualPair, fills[0].CurrencyPair, "fills should be for their channel's market")
	assert.Equal(t, asset.PerpetualContract, fills[0].AssetType, "fills should be for their channel's asset")
	d, ok = payloads[6].(*order.Detail)
	require.True(t, ok, "executions must be sent to the data handler as order updates")
	require.Len(t, d.Trades, 1, "executions must update their order's trades")
	assert.IsType(t, &WebsocketSettlement{}, payloads[7], "settlements should be sent to the data handler")
	assert.Equal(t, &WebsocketLiquidationAlert{
		PositionID:             "1234567890",
		Microtimestamp:         types.Time(time.UnixMicro(1712131200000000)),
		AlertType:              "liquidation_start",
		MarginMode:             MarginModeIsolated,
		InitialMarginRatio:     1.2,
		MaintenanceMarginRatio: 0.9,
	}, payloads[8], "liquidation alerts should be sent to the data handler")
	assert.Equal(t, &WebsocketTokenSettlement{
		Action:                "credit",
		OrderID:               2046687597465600,
		Instrument:            "TSLA/USD",
		TokenSecurityCurrency: "TSLA",
		TokensQuantity:        1,
		DateTime:              parseRFC3339(t, "2026-08-25T10:35:09+00:00"),
	}, payloads[9], "token settlements should be sent to the data handler")
	assert.IsType(t, &WebsocketAnnouncement{}, payloads[10], "announcements should be sent to the data handler")
	assert.IsType(t, &orderbook.Depth{}, payloads[11], "detail orderbooks should be sent to the data handler")
	assert.IsType(t, &WebsocketSelfTrade{}, payloads[12], "self trades should be sent to the data handler")
	fills, ok = payloads[13].([]fill.Data)
	require.True(t, ok, "order trades must be sent to the data handler as fills")
	assert.Equal(t, "296050734", fills[0].TradeID, "order trade fills should be correct")
	d, ok = payloads[14].(*order.Detail)
	require.True(t, ok, "order trades must be sent to the data handler as order updates")
	assert.Equal(t, 0.4, d.RemainingAmount, "order trades should update the remaining amount")
}

func TestWebsocketRequestReconnect(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	err := ex.wsHandleData(t.Context(), []byte(`{"event":"bts:request_reconnect","channel":"","data":null}`))
	assert.NoError(t, err, "wsHandleData should not error on a reconnection request")
}

func TestHandleWebsocketSubscription(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	for event, signature := range map[string]string{"bts:subscription_succeeded": "sub:live_trades_btcusd", "bts:unsubscription_succeeded": "unsub:live_trades_btcusd"} {
		msg := []byte(`{"event":"` + event + `","channel":"live_trades_btcusd","data":{}}`)
		confirmation, err := ex.Websocket.Match.Set(signature, 1)
		require.NoErrorf(t, err, "Match.Set must not error for %s", signature)
		require.NoErrorf(t, ex.handleWSSubscription(event, msg), "handleWSSubscription must not error for %s", event)
		require.Lenf(t, confirmation, 1, "handleWSSubscription must deliver the %s event", event)
		assert.Equalf(t, msg, <-confirmation, "handleWSSubscription should deliver the %s event", event)
		assert.ErrorIsf(t, ex.handleWSSubscription(event, msg), websocket.ErrSignatureNotMatched, "handleWSSubscription should error on an unrequested %s event", event)
	}
	err := ex.handleWSSubscription("bts:subscription_succeeded", []byte(`{"event":"bts:subscription_succeeded","data":{}}`))
	assert.ErrorIs(t, err, common.ErrParsingWSField, "handleWSSubscription should error without a channel")
}

func TestHandleWebsocketError(t *testing.T) {
	t.Parallel()
	err := handleWSError([]byte(`{"event":"bts:error","channel":"","data":{"code":4009,"message":"Connection is unauthorized."}}`))
	assert.ErrorIs(t, err, errWebsocketError, "handleWSError should return a websocket error")
	assert.ErrorContains(t, err, "4009: Connection is unauthorized.", "handleWSError should include the code and message")

	err = handleWSError([]byte(`{"event":"bts:error","channel":"","data":{"code":null,"message":"Bad subscription string."}}`))
	assert.ErrorIs(t, err, errWebsocketError, "handleWSError should return a websocket error without a code")
	assert.EqualError(t, err, "websocket error: Bad subscription string.", "handleWSError should only include the message without a code")

	err = handleWSError([]byte(`{"event":"bts:error","channel":""}`))
	assert.ErrorIs(t, err, common.ErrParsingWSField, "handleWSError should error without data")

	err = handleWSError([]byte(`{"event":"bts:error","channel":"","data":{"code":"wibble"}}`))
	assert.Error(t, err, "handleWSError should error on malformed data")
}

func TestShippedConfigsMatchDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	root, err := testutils.RootPathFromCWD()
	require.NoError(t, err, "RootPathFromCWD must not error")
	expected, err := json.Marshal(defaultSubscriptions)
	require.NoError(t, err, "Marshal must not error for the default subscriptions")
	for _, name := range []string{"config_example.json", filepath.Join("testdata", "configtest.json")} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(filepath.Join(root, name))
			require.NoError(t, err, "ReadFile must not error")
			var cfg config.Config
			require.NoError(t, json.Unmarshal(data, &cfg), "Unmarshal must not error for the shipped config")
			exchCfg, err := cfg.GetExchangeConfig("Bitstamp")
			require.NoError(t, err, "GetExchangeConfig must not error")
			require.NotNil(t, exchCfg.Features, "Bitstamp features must be configured")
			got, err := json.Marshal(exchCfg.Features.Subscriptions)
			require.NoError(t, err, "Marshal must not error for the shipped subscriptions")
			assert.JSONEq(t, string(expected), string(got), "shipped subscriptions should match defaultSubscriptions")
		})
	}
}

func TestNormaliseSubscriptions(t *testing.T) {
	t.Parallel()
	settlements := &subscription.Subscription{Channel: channelMySettlements, Asset: asset.All, Pairs: currency.Pairs{wsSpotPair}, Authenticated: true}
	trades := &subscription.Subscription{Channel: subscription.MyTradesChannel, Asset: asset.Spot}
	public := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.All}
	expanded := &subscription.Subscription{Channel: channelMySettlements, Asset: asset.Spot, QualifiedChannel: "my_settlements"}
	normalised := normaliseSubscriptions(subscription.List{settlements, trades, public, expanded})
	require.Len(t, normalised, 4, "normaliseSubscriptions must return every subscription")
	assert.Equal(t, asset.Empty, normalised[0].Asset, "account channels should not be subscribed per asset")
	assert.Empty(t, normalised[0].Pairs, "account channels should not be subscribed per pair")
	assert.Equal(t, asset.All, settlements.Asset, "normaliseSubscriptions should not modify the original subscription")
	assert.True(t, normalised[1].Authenticated, "private channels should be authenticated")
	assert.False(t, trades.Authenticated, "normaliseSubscriptions should not modify the original private subscription")
	assert.Same(t, public, normalised[2], "public market channels should not change")
	assert.Same(t, expanded, normalised[3], "expanded subscriptions should not change")

	ex := newWebsocketTestInstance(t)
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	ex.Features.Subscriptions = subscription.List{settlements}
	subs, err := ex.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	require.Len(t, subs, 1, "an account channel configured for every asset must only be subscribed once")
	assert.Equal(t, channelMySettlements, subs[0].QualifiedChannel, "the account channel should be subscribed")
}

func TestSplitChannel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		channel, name, market string
		private               bool
		err                   error
	}{
		{channel: "live_trades_btcusd", name: "live_trades", market: "btcusd"},
		{channel: "order_book_btcusd-perp", name: "order_book", market: "btcusd-perp"},
		{channel: "funding_rate_btcusd-perp", name: "funding_rate", market: "btcusd-perp"},
		{channel: "announcements", name: "announcements"},
		{channel: "private-my_orders_btcusd-314159", name: "my_orders", market: "btcusd", private: true},
		{channel: "private-my_trades_btcusd-perp-314159", name: "my_trades", market: "btcusd-perp", private: true},
		{channel: "private-my_settlements-314159", name: "my_settlements", private: true},
		{channel: "private-my_token_settlements-314159", name: "my_token_settlements", private: true},
		{channel: "", err: errMalformedChannel},
		{channel: "livetrades", err: errMalformedChannel},
		{channel: "live_trades_", err: errMalformedChannel},
		{channel: "_btcusd", err: errMalformedChannel},
		{channel: "private-my_orders_btcusd", err: errMalformedChannel},
		{channel: "private-my_orders_btcusd-perp", err: errMalformedChannel},
		{channel: "private--314159", err: errMalformedChannel},
	} {
		name, market, private, err := splitChannel(tc.channel)
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "splitChannel should error for %q", tc.channel)
			continue
		}
		require.NoErrorf(t, err, "splitChannel must not error for %q", tc.channel)
		assert.Equalf(t, tc.name, name, "name should be correct for %q", tc.channel)
		assert.Equalf(t, tc.market, market, "market should be correct for %q", tc.channel)
		assert.Equalf(t, tc.private, private, "private should be correct for %q", tc.channel)
	}
}

func TestMarketPair(t *testing.T) {
	t.Parallel()
	for market, exp := range map[string]struct {
		pair currency.Pair
		a    asset.Item
	}{
		"btcusd":      {wsSpotPair, asset.Spot},
		"btcusd-perp": {wsPerpetualPair, asset.PerpetualContract},
	} {
		pair, a, err := e.marketPair(market)
		require.NoErrorf(t, err, "marketPair must not error for %s", market)
		assert.Equalf(t, exp.pair, pair, "marketPair should return the pair for %s", market)
		assert.Equalf(t, exp.a, a, "marketPair should return the asset for %s", market)
	}
	_, _, err := e.marketPair("wibble-perp")
	assert.ErrorIs(t, err, errParsingWSPair, "marketPair should error on an unknown market")
}

func TestChannelName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, channelOrderbook, channelName(&subscription.Subscription{Channel: subscription.OrderbookChannel}), "channelName should convert global channel names")
	assert.Equal(t, channelFundingRate, channelName(&subscription.Subscription{Channel: channelFundingRate}), "channelName should return exchange channel names")
	assert.PanicsWithError(t, "subscription channel not supported: wibble", func() { channelName(&subscription.Subscription{Channel: "wibble"}) }, "channelName should panic on an unsupported channel")
}

func TestIsAccountChannel(t *testing.T) {
	t.Parallel()
	assert.True(t, isAccountChannel(&subscription.Subscription{Channel: channelMySettlements}), "isAccountChannel should be true for account channels")
	assert.False(t, isAccountChannel(&subscription.Subscription{Channel: subscription.MyOrdersChannel}), "isAccountChannel should be false for market channels")
	assert.False(t, isAccountChannel(&subscription.Subscription{Channel: "wibble"}), "isAccountChannel should be false for unsupported channels")
}

func TestIsAccountChannelName(t *testing.T) {
	t.Parallel()
	for name, exp := range map[string]bool{
		channelAnnouncements:      true,
		channelMySettlements:      true,
		channelMyLiquidations:     true,
		channelMyTokenSettlements: true,
		channelMyOrders:           false,
		channelOrderbook:          false,
		"":                        false,
	} {
		assert.Equalf(t, exp, isAccountChannelName(name), "isAccountChannelName should be correct for %q", name)
	}
}

func TestGetSubscriptionTemplate(t *testing.T) {
	t.Parallel()
	tpl, err := e.GetSubscriptionTemplate(nil)
	require.NoError(t, err, "GetSubscriptionTemplate must not error")
	require.NotNil(t, tpl, "GetSubscriptionTemplate must return a template")
	for _, tc := range []struct {
		s          *subscription.Subscription
		assetPairs map[asset.Item]currency.Pairs
		exp        string
	}{
		{
			s: &subscription.Subscription{Channel: subscription.AllTradesChannel},
			assetPairs: map[asset.Item]currency.Pairs{
				asset.Spot:              {currency.NewPairWithDelimiter("btc", "usd", ""), currency.NewPairWithDelimiter("eth", "usd", "")},
				asset.PerpetualContract: {currency.NewPairWithDelimiter("btc", "usd-perp", "")},
			},
			exp: "live_trades_btcusd|live_trades_ethusd|#live_trades_btcusd-perp|#",
		},
		{
			s:          &subscription.Subscription{Channel: channelAnnouncements},
			assetPairs: map[asset.Item]currency.Pairs{asset.Empty: nil},
			exp:        "announcements#",
		},
		{
			// The pairs a subscription names reach the template as they were configured
			s: &subscription.Subscription{Channel: subscription.OrderbookChannel},
			assetPairs: map[asset.Item]currency.Pairs{
				asset.Spot:              {currency.NewPairWithDelimiter("BTC", "USD", "/")},
				asset.PerpetualContract: {currency.NewPairWithDelimiter("BTC", "USD-PERP", "/")},
			},
			exp: "order_book_btcusd|#order_book_btcusd-perp|#",
		},
	} {
		var buf bytes.Buffer
		require.NoErrorf(t, tpl.Execute(&buf, map[string]any{"S": tc.s, "AssetPairs": tc.assetPairs, "PairSeparator": "|", "AssetSeparator": "#"}), "Execute must not error for %s", tc.s.Channel)
		assert.Equalf(t, tc.exp, strings.Join(strings.Fields(buf.String()), ""), "the template should render a channel per pair and a record per asset for %s", tc.s.Channel)
	}
}

func TestExpandNamedPairs(t *testing.T) {
	t.Parallel()
	subs, err := subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Pairs: currency.Pairs{currency.NewPairWithDelimiter("BTC", "USD", "/")}},
		{Enabled: true, Asset: asset.PerpetualContract, Channel: channelFundingRate, Pairs: currency.Pairs{currency.NewPairWithDelimiter("BTC", "USD-PERP", "/")}},
	}.ExpandTemplates(newWebsocketTestInstance(t))
	require.NoError(t, err, "ExpandTemplates must not error")
	channels := make([]string, len(subs))
	for i := range subs {
		channels[i] = subs[i].QualifiedChannel
	}
	assert.Equal(t, []string{"order_book_btcusd", "funding_rate_btcusd-perp"}, channels, "subscriptions naming their pairs should subscribe to the pairs' market symbols")
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	ex.Features.Subscriptions = append(ex.Features.Subscriptions,
		&subscription.Subscription{Enabled: true, Channel: channelAnnouncements},
		&subscription.Subscription{Enabled: true, Channel: channelMySettlements, Authenticated: true},
	)
	subs, err := ex.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")

	exp := subscription.List{}
	for _, s := range ex.Features.Subscriptions {
		if isAccountChannel(s) {
			sub := s.Clone()
			sub.QualifiedChannel = channelName(s)
			exp = append(exp, sub)
			continue
		}
		for _, a := range []asset.Item{asset.Spot, asset.PerpetualContract} {
			if s.Asset != asset.All && s.Asset != a {
				continue
			}
			pairs, err := ex.GetEnabledPairs(a)
			require.NoErrorf(t, err, "GetEnabledPairs must not error for %s", a)
			for _, p := range pairs.Format(currency.PairFormat{}) {
				sub := s.Clone()
				sub.Asset = a
				sub.Pairs = currency.Pairs{p}
				sub.QualifiedChannel = channelName(s) + "_" + p.String()
				exp = append(exp, sub)
			}
		}
	}
	testsubs.EqualLists(t, exp, subs)

	ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
	subs, err = ex.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error without authentication")
	assert.Empty(t, subs.Private(), "generateSubscriptions should not return authenticated subscriptions without authentication")
}

// mockWsSubscriptions records the channels subscribed to by a mock websocket server
type mockWsSubscriptions struct {
	mu       sync.Mutex
	channels map[string]string
}

func (m *mockWsSubscriptions) handle(tb testing.TB, msg []byte, c *gws.Conn) error {
	tb.Helper()
	var req WebsocketEventRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return err
	}
	event := "bts:subscription_succeeded"
	m.mu.Lock()
	if req.Event == "bts:unsubscribe" {
		event = "bts:unsubscription_succeeded"
		delete(m.channels, req.Data.Channel)
	} else {
		m.channels[req.Data.Channel] = req.Data.Auth
	}
	m.mu.Unlock()
	resp, err := json.Marshal(map[string]any{"event": event, "channel": req.Data.Channel, "data": struct{}{}})
	if err != nil {
		return err
	}
	return c.WriteMessage(gws.TextMessage, resp)
}

func (m *mockWsSubscriptions) subscribed() map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return maps.Clone(m.channels)
}

// newMockWsInstance returns an exchange connected to a mock websocket server which confirms every subscription
// The server's REST endpoint returns websocket tokens when tokenAvailable is set
func newMockWsInstance(t *testing.T, tokenAvailable bool) (*Exchange, *mockWsSubscriptions) {
	t.Helper()
	m := &mockWsSubscriptions{channels: make(map[string]string)}
	wsHandler := mockws.CurryWsMockUpgrader(t, m.handle)
	ex := testexch.MockWsInstance[Exchange](t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			assert.Equal(t, "/v2/websockets_token/", r.URL.Path, "websocket token path should be correct")
			if !tokenAvailable {
				w.WriteHeader(http.StatusForbidden)
				_, err := w.Write([]byte(`{"status":"error","reason":"Invalid signature","code":"API0005"}`))
				assert.NoError(t, err, "Write should not error")
				return
			}
			_, err := w.Write([]byte(`{"token":"Tr6kJCxPiqCcTVWRmNaNBR1EUGPUTjlA","valid_sec":60,"user_id":314159}`))
			assert.NoError(t, err, "Write should not error")
			return
		}
		wsHandler(w, r)
	})
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	return ex, m
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	if !mockTests {
		testSubscribeLive(t)
		return
	}
	ex, m := newMockWsInstance(t, true)
	subs, err := subscription.List{
		{Enabled: true, Asset: asset.All, Channel: subscription.AllTradesChannel},
		{Enabled: true, Asset: asset.PerpetualContract, Channel: channelFundingRate},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.MyOrdersChannel, Authenticated: true},
		{Enabled: true, Asset: asset.PerpetualContract, Channel: subscription.MyTradesChannel, Authenticated: true},
		{Enabled: true, Channel: channelAnnouncements},
		{Enabled: true, Channel: channelMySettlements, Authenticated: true},
	}.ExpandTemplates(ex)
	require.NoError(t, err, "ExpandTemplates must not error")

	require.NoError(t, ex.Subscribe(subs), "Subscribe must not error")
	for _, s := range subs {
		assert.Equalf(t, subscription.SubscribedState, s.State(), "subscription %s should be subscribed", s)
	}
	const token = "Tr6kJCxPiqCcTVWRmNaNBR1EUGPUTjlA"
	assert.Equal(t, map[string]string{
		"live_trades_btcusd":                   "",
		"live_trades_btceur":                   "",
		"live_trades_eurusd":                   "",
		"live_trades_xrpusd":                   "",
		"live_trades_xrpeur":                   "",
		"live_trades_btcusd-perp":              "",
		"funding_rate_btcusd-perp":             "",
		"private-my_orders_btcusd-314159":      token,
		"private-my_orders_btceur-314159":      token,
		"private-my_orders_eurusd-314159":      token,
		"private-my_orders_xrpusd-314159":      token,
		"private-my_orders_xrpeur-314159":      token,
		"private-my_trades_btcusd-perp-314159": token,
		"announcements":                        "",
		"private-my_settlements-314159":        token,
	}, m.subscribed(), "the server should receive every channel, with private channels suffixed with the user ID and authorised with the token")

	require.NoError(t, ex.Unsubscribe(subs), "Unsubscribe must not error")
	for _, s := range subs {
		assert.Equalf(t, subscription.UnsubscribedState, s.State(), "subscription %s should be unsubscribed", s)
	}
	assert.Empty(t, m.subscribed(), "the server should receive an unsubscription for every channel")
}

// testSubscribeLive subscribes to every public channel on Bitstamp's websocket server, and to the private channels when
// credentials are set, and checks that order books stream into the store
func testSubscribeLive(t *testing.T) {
	t.Helper()
	ex := newWebsocketTestInstance(t)
	templates := subscription.List{
		{Enabled: true, Asset: asset.All, Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds},
		{Enabled: true, Asset: asset.All, Channel: subscription.AllTradesChannel},
		{Enabled: true, Asset: asset.PerpetualContract, Channel: channelFundingRate},
		{Enabled: true, Channel: channelAnnouncements},
	}
	if sharedtestvalues.AreAPICredentialsSet(e) {
		creds, err := e.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		ex.API.AuthenticatedSupport = true
		ex.API.AuthenticatedWebsocketSupport = true
		ex.SetCredentials(creds)
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		templates = append(templates,
			&subscription.Subscription{Enabled: true, Asset: asset.All, Channel: subscription.MyOrdersChannel, Authenticated: true},
			&subscription.Subscription{Enabled: true, Asset: asset.All, Channel: subscription.MyTradesChannel, Authenticated: true},
			&subscription.Subscription{Enabled: true, Channel: channelMySettlements, Authenticated: true},
			&subscription.Subscription{Enabled: true, Channel: channelMyLiquidations, Authenticated: true},
			&subscription.Subscription{Enabled: true, Channel: channelMyTokenSettlements, Authenticated: true},
		)
	}
	ex.Features.Subscriptions = subscription.List{}
	ex.Websocket.SetSubscriptionsNotRequired()

	// Order books stream continuously, so the data handler is drained to stop it filling up
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ex.Websocket.DataHandler.C:
			case <-done:
				return
			}
		}
	}()
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	t.Cleanup(func() {
		assert.NoError(t, ex.Websocket.Disable(), "Disable should not error")
		if err := ex.Websocket.Shutdown(); err != nil {
			assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should only error when the connection is already closed")
		}
		close(done)
	})

	subs, err := templates.ExpandTemplates(ex)
	require.NoError(t, err, "ExpandTemplates must not error")
	require.NoError(t, ex.Subscribe(subs), "Subscribe must not error")
	for _, s := range subs {
		assert.Equalf(t, subscription.SubscribedState, s.State(), "subscription %s should be subscribed", s)
	}
	for a, p := range map[asset.Item]currency.Pair{asset.Spot: spotPair, asset.PerpetualContract: perpetualPair} {
		assert.Eventuallyf(t, func() bool {
			_, err := orderbook.Get(ex.Name, p, a)
			return err == nil
		}, 30*time.Second, 100*time.Millisecond, "the %s order book for %s should be streamed", a, p)
	}

	require.NoError(t, ex.Unsubscribe(subs), "Unsubscribe must not error")
	for _, s := range subs {
		assert.Equalf(t, subscription.UnsubscribedState, s.State(), "subscription %s should be unsubscribed", s)
	}
}

func TestManageSubsWithCreds(t *testing.T) {
	t.Parallel()
	ex, m := newMockWsInstance(t, false)
	subs, err := subscription.List{
		{Enabled: true, Asset: asset.PerpetualContract, Channel: channelFundingRate},
		{Enabled: true, Asset: asset.PerpetualContract, Channel: subscription.MyOrdersChannel, Authenticated: true},
	}.ExpandTemplates(ex)
	require.NoError(t, err, "ExpandTemplates must not error")

	err = ex.manageSubsWithCreds(t.Context(), subs, "sub")
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "manageSubsWithCreds should error when a websocket token cannot be fetched")
	assert.ErrorIs(t, err, errAPIResponse, "manageSubsWithCreds should return the websocket token error")
	assert.Equal(t, map[string]string{"funding_rate_btcusd-perp": ""}, m.subscribed(), "public channels should still be subscribed")
	assert.Equal(t, subscription.SubscribedState, subs[0].State(), "the public subscription should be subscribed")
	assert.NotEqual(t, subscription.SubscribedState, subs[1].State(), "the private subscription should not be subscribed")

	ex, m = newMockWsInstance(t, true)
	subs, err = subscription.List{{Enabled: true, Asset: asset.PerpetualContract, Channel: subscription.MyOrdersChannel, Authenticated: true}}.ExpandTemplates(ex)
	require.NoError(t, err, "ExpandTemplates must not error")
	require.NoError(t, ex.manageSubsWithCreds(t.Context(), subs, "sub"), "manageSubsWithCreds must not error")
	assert.Equal(t, map[string]string{"private-my_orders_btcusd-perp-314159": "Tr6kJCxPiqCcTVWRmNaNBR1EUGPUTjlA"}, m.subscribed(), "private channels should use the fetched token")
	require.NoError(t, ex.manageSubsWithCreds(t.Context(), subs, "unsub"), "manageSubsWithCreds must not error when unsubscribing")
	assert.Empty(t, m.subscribed(), "private channels should be unsubscribed")

	// Private channels need not be marked as authenticated, whether or not their templates have been expanded
	for _, expand := range []bool{false, true} {
		ex, m = newMockWsInstance(t, true)
		unmarked := subscription.List{{Enabled: true, Asset: asset.PerpetualContract, Channel: subscription.MyOrdersChannel}}
		if expand {
			unmarked, err = unmarked.ExpandTemplates(ex)
			require.NoError(t, err, "ExpandTemplates must not error")
		}
		require.NoErrorf(t, ex.manageSubsWithCreds(t.Context(), unmarked, "sub"), "manageSubsWithCreds must not error on an unmarked private channel, expanded: %t", expand)
		assert.Equalf(t, map[string]string{"private-my_orders_btcusd-perp-314159": "Tr6kJCxPiqCcTVWRmNaNBR1EUGPUTjlA"}, m.subscribed(), "an unmarked private channel should use the fetched token, expanded: %t", expand)
		assert.Falsef(t, unmarked[0].Authenticated, "manageSubsWithCreds should not modify the subscription, expanded: %t", expand)
		if expand {
			// An expanded subscription is stored as it is, so the websocket manager tracks the caller's subscription
			assert.Equal(t, subscription.SubscribedState, unmarked[0].State(), "an expanded subscription should be subscribed")
		}
		require.NoErrorf(t, ex.manageSubsWithCreds(t.Context(), unmarked, "unsub"), "manageSubsWithCreds must not error when unsubscribing an unmarked private channel, expanded: %t", expand)
		assert.Emptyf(t, m.subscribed(), "an unmarked private channel should be unsubscribed, expanded: %t", expand)
		if expand {
			assert.Equal(t, subscription.UnsubscribedState, unmarked[0].State(), "an expanded subscription should be unsubscribed")
		}
	}
}

func TestUnsubscribeWithoutToken(t *testing.T) {
	t.Parallel()
	ex, m := newMockWsInstance(t, true)
	subs, err := subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.MyOrdersChannel, Authenticated: true}}.ExpandTemplates(ex)
	require.NoError(t, err, "ExpandTemplates must not error")
	require.NoError(t, ex.Subscribe(subs), "Subscribe must not error")

	// The token endpoint now refuses requests, so unsubscribing must use the user ID kept from subscribing
	refused := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, err := w.Write([]byte(`{"status":"error","reason":"Invalid signature","code":"API0005"}`))
		assert.NoError(t, err, "Write should not error")
	}))
	t.Cleanup(refused.Close)
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), refused.URL), "SetRunningURL must not error")
	require.NoError(t, ex.Unsubscribe(subs), "Unsubscribe must not need a new token")
	assert.Empty(t, m.subscribed(), "private channels should be unsubscribed with the user ID of the last token")
}

func TestIsPrivateSubscription(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		sub *subscription.Subscription
		exp bool
	}{
		{sub: &subscription.Subscription{Channel: subscription.OrderbookChannel}, exp: false},
		{sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Authenticated: true}, exp: true},
		{sub: &subscription.Subscription{Channel: subscription.MyOrdersChannel}, exp: true},
		{sub: &subscription.Subscription{Channel: channelMySettlements, QualifiedChannel: "my_settlements"}, exp: true},
	} {
		assert.Equalf(t, tc.exp, isPrivateSubscription(tc.sub), "isPrivateSubscription should return %t for %s, authenticated: %t", tc.exp, tc.sub.Channel, tc.sub.Authenticated)
	}
}

func TestManageSubs(t *testing.T) {
	t.Parallel()
	ex, m := newMockWsInstance(t, false)
	subs, err := subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.MyTradesChannel, Authenticated: true},
		{Enabled: true, Channel: channelMyLiquidations, Authenticated: true},
	}.ExpandTemplates(ex)
	require.NoError(t, err, "ExpandTemplates must not error")

	err = ex.manageSubs(t.Context(), subs, "sub", nil)
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "manageSubs should error on private channels without a token")
	assert.Empty(t, m.subscribed(), "private channels should not be subscribed without a token")

	creds := &WebsocketTokenResponse{Token: "token", UserID: 1}
	require.NoError(t, ex.manageSubs(t.Context(), subs, "sub", creds), "manageSubs must not error")
	for _, s := range subs {
		assert.Equalf(t, subscription.SubscribedState, s.State(), "subscription %s should be subscribed", s)
	}
	assert.Equal(t, map[string]string{
		"private-my_trades_btcusd-1": "token",
		"private-my_trades_btceur-1": "token",
		"private-my_trades_eurusd-1": "token",
		"private-my_trades_xrpusd-1": "token",
		"private-my_trades_xrpeur-1": "token",
		"private-my_liquidations-1":  "token",
	}, m.subscribed(), "private channels should be suffixed with the user ID and authorised with the token")

	require.NoError(t, ex.manageSubs(t.Context(), subs, "unsub", creds), "manageSubs must not error when unsubscribing")
	for _, s := range subs {
		assert.Equalf(t, subscription.UnsubscribedState, s.State(), "subscription %s should be unsubscribed", s)
	}
	assert.Empty(t, m.subscribed(), "every channel should be unsubscribed")
	assert.ErrorIs(t, ex.manageSubs(t.Context(), subs, "unsub", creds), subscription.ErrNotFound, "manageSubs should error when unsubscribing channels which are not subscribed")
}
