package bitstamp

import (
	"bytes"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"slices"
	"strconv"
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
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
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

func TestHandleWebsocketOrderbook(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	assert.Error(t, ex.handleWSOrderbook(t.Context(), []byte(`{"bids":"wibble"}`), wsSpotPair, asset.Spot), "handleWSOrderbook should error on malformed data")

	for _, tc := range []struct {
		data string
		exp  *orderbook.Book
	}{
		{
			data: `{"timestamp":"1580336834","microtimestamp":"1580336834607546","bids":[["9328.28","0.05925332"],["9327.34","0.43120000"],["0","69"]],"asks":[["9337.10","0.03000000"],["9340.85","2.67820000"]]}`,
			exp: &orderbook.Book{
				Bids:              orderbook.Levels{{Price: 9328.28, Amount: 0.05925332}, {Price: 9327.34, Amount: 0.4312}},
				Asks:              orderbook.Levels{{Price: 9337.1, Amount: 0.03}, {Price: 9340.85, Amount: 2.6782}},
				Exchange:          ex.Name,
				Pair:              wsSpotPair,
				Asset:             asset.Spot,
				LastUpdated:       time.UnixMicro(1580336834607546),
				ValidateOrderbook: ex.ValidateOrderbook,
			},
		},
		{
			data: `{"timestamp":"1790318551","microtimestamp":"1790318551908216","bids":[["84010","0.01190"],["84006","0.33643"]],"asks":[["84018","0.33440"],["84021","1.07000"]]}`,
			exp: &orderbook.Book{
				Bids:              orderbook.Levels{{Price: 84010, Amount: 0.0119}, {Price: 84006, Amount: 0.33643}},
				Asks:              orderbook.Levels{{Price: 84018, Amount: 0.3344}, {Price: 84021, Amount: 1.07}},
				Exchange:          ex.Name,
				Pair:              wsPerpetualPair,
				Asset:             asset.PerpetualContract,
				LastUpdated:       time.UnixMicro(1790318551908216),
				ValidateOrderbook: ex.ValidateOrderbook,
			},
		},
	} {
		require.NoErrorf(t, ex.handleWSOrderbook(t.Context(), []byte(tc.data), tc.exp.Pair, tc.exp.Asset), "handleWSOrderbook must not error for %s", tc.exp.Asset)
		ob, err := ex.Websocket.Orderbook.GetOrderbook(tc.exp.Pair, tc.exp.Asset)
		require.NoErrorf(t, err, "GetOrderbook must not error for %s", tc.exp.Asset)
		assert.NotZerof(t, ob.InsertedAt, "InsertedAt should be set for %s", tc.exp.Asset)
		ob.InsertedAt = time.Time{}
		assert.Equalf(t, tc.exp, ob, "handleWSOrderbook should load the snapshot without zero priced bids for %s", tc.exp.Asset)
	}
}

func TestHandleWebsocketTrade(t *testing.T) {
	t.Parallel()
	data := []byte(`{"microtimestamp":"1580336751488517","amount":0.00598803,"buy_order_id":4621328909,"sell_order_id":4621329035,"amount_str":"0.00598803","price_str":"9334.73","timestamp":"1580336751","price":9334.73,"type":1,"id":104007706}`)
	ex := newWebsocketTestInstance(t)
	require.NoError(t, ex.handleWSTrade(data, wsSpotPair, asset.Spot), "handleWSTrade must not error with the trade feed disabled")
	assert.Empty(t, dataHandlerPayloads(ex), "trades should not be sent with the trade feed disabled")

	ex = newWebsocketTestInstance(t)
	ex.SetTradeFeedStatus(true)
	ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
	assert.Error(t, ex.handleWSTrade([]byte(`{"price_str":[]}`), wsSpotPair, asset.Spot), "handleWSTrade should error on malformed data")
	require.NoError(t, ex.handleWSTrade(data, wsSpotPair, asset.Spot), "handleWSTrade must not error")
	assert.Equal(t, []any{[]trade.Data{{
		TID:          "104007706",
		Exchange:     ex.Name,
		CurrencyPair: wsSpotPair,
		AssetType:    asset.Spot,
		Side:         order.Sell,
		Price:        9334.73,
		Amount:       0.00598803,
		Timestamp:    time.UnixMicro(1580336751488517),
	}}}, dataHandlerPayloads(ex), "trades should be sent to the data handler")
}

func TestHandleWebsocketFundingRate(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	assert.Error(t, ex.handleWSFundingRate(t.Context(), []byte(`{"funding_rate":[]}`), wsPerpetualPair, asset.PerpetualContract), "handleWSFundingRate should error on malformed data")
	data := []byte(`{"market":"btcusd-perp","mark_price":"83997.21486186","index_price":"83991.59300000002","funding_rate":"0.0001","timestamp":"1790318552","next_funding_time":"1790323200"}`)
	require.NoError(t, ex.handleWSFundingRate(t.Context(), data, wsPerpetualPair, asset.PerpetualContract), "handleWSFundingRate must not error")
	assert.Equal(t, []any{
		websocket.FundingData{
			Timestamp:    time.Unix(1790318552, 0),
			CurrencyPair: wsPerpetualPair,
			AssetType:    asset.PerpetualContract,
			Exchange:     ex.Name,
			Rate:         0.0001,
		},
		&WebsocketFundingRate{
			Pair:            wsPerpetualPair,
			AssetType:       asset.PerpetualContract,
			Market:          "btcusd-perp",
			MarkPrice:       83997.21486186,
			IndexPrice:      83991.59300000002,
			FundingRate:     0.0001,
			Timestamp:       types.Time(time.Unix(1790318552, 0)),
			NextFundingTime: types.Time(time.Unix(1790323200, 0)),
		},
	}, dataHandlerPayloads(ex), "funding rates and updates should be sent to the data handler")
}

func TestWebsocketOrderUpdate(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	testexch.FixtureToDataHandler(t, "testdata/wsMyOrders.json", ex.wsHandleData)
	payloads := dataHandlerPayloads(ex)
	require.Len(t, payloads, 8, "the data handler must receive an update for each order event")

	// The fixture was captured before order subtypes were published, so every order is decoded as a limit order
	partialMarketBuy, partialMarketSell := 0.00038667-0.00000001, 0.00038679-0.00000001
	for i, exp := range []struct {
		orderID, clientOrderID                    string
		status                                    order.Status
		side                                      order.Side
		price, amount, remainingAmount, execution float64
		date                                      time.Time
	}{
		{"1658864794234880", "test_market_buy", order.New, order.Buy, 999999999, 0, 0, 0, time.UnixMicro(1693831262313000)},
		{"1658864794234880", "test_market_buy", order.PartiallyFilled, order.Buy, 25862, 0.00038667, 0.00000001, partialMarketBuy, time.UnixMicro(1693831262313000)},
		{"1658864794234880", "test_market_buy", order.PartiallyCancelled, order.Buy, 25862, 0.00038667, 0.00000001, partialMarketBuy, time.UnixMicro(1693831262313000)},
		{"1658870500933632", "test_market_sell", order.New, order.Sell, 0, 0, 0, 0, time.UnixMicro(1693832655550000)},
		{"1658870500933632", "test_market_sell", order.PartiallyFilled, order.Sell, 25854, 0.00038679, 0.00000001, partialMarketSell, time.UnixMicro(1693832655550000)},
		{"1658870500933632", "test_market_sell", order.PartiallyCancelled, order.Sell, 25854, 0.00038679, 0.00000001, partialMarketSell, time.UnixMicro(1693832655550000)},
		{"1658869033291777", "test_limit_sell", order.New, order.Sell, 25845, 0.00038692, 0.00038692, 0, time.UnixMicro(1693832297239000)},
		{"1658869033291777", "test_limit_sell", order.Filled, order.Sell, 25845, 0.00038692, 0, 0.00038692, time.UnixMicro(1693832302664000)},
	} {
		d, ok := payloads[i].(*order.Detail)
		require.Truef(t, ok, "payload %d must be an order detail", i)
		assert.Equalf(t, &order.Detail{
			Exchange:        ex.Name,
			OrderID:         exp.orderID,
			ClientOrderID:   exp.clientOrderID,
			Type:            order.Limit,
			TimeInForce:     order.GoodTillCancel,
			Side:            exp.side,
			Status:          exp.status,
			AssetType:       asset.Spot,
			Pair:            wsSpotPair,
			Date:            exp.date,
			LastUpdated:     exp.date,
			AccountID:       "3141592",
			Price:           exp.price,
			Amount:          exp.amount,
			RemainingAmount: exp.remainingAmount,
			ExecutedAmount:  exp.execution,
		}, d, "order event %d should be correct", i)
	}
}

func TestProcessOrderEvent(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	err := ex.processOrderEvent(t.Context(), orderEventCreated, &OrderEventData{RemainingAmount: 1}, wsPerpetualPair, asset.PerpetualContract)
	assert.ErrorIs(t, err, errOrderIDMissing, "processOrderEvent should error without an order ID")
	err = ex.handleWSAccountOrder(t.Context(), nil, orderEventCreated, []byte(`{"id":"wibble"}`), wsPerpetualPair, asset.PerpetualContract)
	assert.Error(t, err, "handleWSAccountOrder should error on malformed data")

	for _, tc := range []struct {
		event, data string
	}{
		{event: orderEventChanged, data: `{"id":1500000001,"id_str":"1500000001","client_order_id":"my-order-001","amount":0.5,"amount_str":"0.50000000","amount_traded":"0","amount_at_create":"0.50000000","price":61200.00,"price_str":"61200.00","order_type":0,"order_subtype":0,"datetime":"1712131200","microtimestamp":"1712131200000000","trade_account_id":0,"is_liquidation":false}`},
		{event: orderEventReplaced, data: `{"id":1500000002,"id_str":"1500000002","orig_order_id":1500000001,"client_order_id":"my-order-002","amount":0.3,"amount_str":"0.30000000","amount_traded":"0.00000000","amount_at_create":"0.30000000","price":61500.00,"price_str":"61500.00","order_type":0,"order_subtype":8,"datetime":"1712131201","microtimestamp":"1712131201000000","trade_account_id":0,"is_liquidation":false}`},
		{event: orderEventStopActive, data: `{"id":1500000003,"id_str":"1500000003","amount":0.01,"amount_str":"0.01000","amount_traded":"0","amount_at_create":"0.01000","price":83000,"price_str":"83000","order_type":1,"order_subtype":22,"datetime":"1712131202","microtimestamp":"1712131202000000","trade_account_id":0,"is_liquidation":false,"reduce_only":true,"stop_price":"83500"}`},
		{event: orderEventStopInactive, data: `{"id":1500000004,"id_str":"1500000004","amount":0.01,"amount_str":"0.01000","amount_traded":"0","amount_at_create":"0.01000","price":0,"price_str":"0","order_type":1,"order_subtype":20,"datetime":"1712131203","microtimestamp":"1712131203000000","trade_account_id":0,"is_liquidation":false,"stop_price":"82000"}`},
		{event: orderEventDeleted, data: `{"client_order_id":"my-order-005","amount_str":"0.40000000","amount_at_create":"1.00000000","price_str":"61000","order_type":0,"microtimestamp":"1712131204000000","trade_account_id":42}`},
		{event: orderEventDeleted, data: `{"id":1500000006,"amount_str":"1.00000000","amount_at_create":"1.00000000","price_str":"61000","order_type":0,"microtimestamp":"1712131205000000"}`},
		{event: orderEventDeleted, data: `{"id":1500000007,"amount_str":"0","amount_at_create":"1.00000000","price_str":"61000","order_type":0,"microtimestamp":"1712131206000000"}`},
		{event: orderEventStopActive, data: `{"id":1500000009,"amount_str":"0.01000","amount_at_create":"0.01000","price_str":"0","order_type":1,"order_subtype":24,"microtimestamp":"1712131207000000","activation_price":"50000","trailing_delta":"50"}`},
		{event: "order_unknown", data: `{"id":1500000008}`},
	} {
		require.NoErrorf(t, ex.handleWSAccountOrder(t.Context(), nil, tc.event, []byte(tc.data), wsPerpetualPair, asset.PerpetualContract), "handleWSAccountOrder must not error for %s", tc.event)
	}
	assert.Equal(t, []any{
		&order.Detail{
			Exchange:        ex.Name,
			OrderID:         "1500000001",
			ClientOrderID:   "my-order-001",
			Type:            order.Limit,
			TimeInForce:     order.GoodTillCancel,
			Side:            order.Buy,
			Status:          order.Open,
			AssetType:       asset.PerpetualContract,
			Pair:            wsPerpetualPair,
			Date:            time.UnixMicro(1712131200000000),
			LastUpdated:     time.UnixMicro(1712131200000000),
			Price:           61200,
			Amount:          0.5,
			RemainingAmount: 0.5,
		},
		[]order.Detail{
			{
				Exchange:        ex.Name,
				OrderID:         "1500000002",
				ClientOrderID:   "my-order-002",
				Type:            order.Limit,
				TimeInForce:     order.GoodTillTime,
				Side:            order.Buy,
				Status:          order.New,
				AssetType:       asset.PerpetualContract,
				Pair:            wsPerpetualPair,
				Date:            time.UnixMicro(1712131201000000),
				LastUpdated:     time.UnixMicro(1712131201000000),
				Price:           61500,
				Amount:          0.3,
				RemainingAmount: 0.3,
			},
			{
				Exchange:    ex.Name,
				OrderID:     "1500000001",
				Status:      order.Cancelled,
				AssetType:   asset.PerpetualContract,
				Pair:        wsPerpetualPair,
				LastUpdated: time.UnixMicro(1712131201000000),
			},
		},
		&order.Detail{
			Exchange:        ex.Name,
			OrderID:         "1500000003",
			Type:            order.StopLimit,
			Side:            order.Sell,
			Status:          order.Active,
			AssetType:       asset.PerpetualContract,
			Pair:            wsPerpetualPair,
			Date:            time.UnixMicro(1712131202000000),
			LastUpdated:     time.UnixMicro(1712131202000000),
			Price:           83000,
			Amount:          0.01,
			RemainingAmount: 0.01,
			ReduceOnly:      true,
			TriggerPrice:    83500,
		},
		&order.Detail{
			Exchange:        ex.Name,
			OrderID:         "1500000004",
			Type:            order.StopMarket,
			Side:            order.Sell,
			Status:          order.Cancelled,
			AssetType:       asset.PerpetualContract,
			Pair:            wsPerpetualPair,
			Date:            time.UnixMicro(1712131203000000),
			LastUpdated:     time.UnixMicro(1712131203000000),
			Amount:          0.01,
			RemainingAmount: 0.01,
			TriggerPrice:    82000,
		},
		&order.Detail{
			Exchange:        ex.Name,
			ClientOrderID:   "my-order-005",
			AccountID:       "42",
			Type:            order.Limit,
			TimeInForce:     order.GoodTillCancel,
			Side:            order.Buy,
			Status:          order.PartiallyCancelled,
			AssetType:       asset.PerpetualContract,
			Pair:            wsPerpetualPair,
			Date:            time.UnixMicro(1712131204000000),
			LastUpdated:     time.UnixMicro(1712131204000000),
			Price:           61000,
			Amount:          1,
			RemainingAmount: 0.4,
			ExecutedAmount:  0.6,
		},
		&order.Detail{
			Exchange:        ex.Name,
			OrderID:         "1500000006",
			Type:            order.Limit,
			TimeInForce:     order.GoodTillCancel,
			Side:            order.Buy,
			Status:          order.Cancelled,
			AssetType:       asset.PerpetualContract,
			Pair:            wsPerpetualPair,
			Date:            time.UnixMicro(1712131205000000),
			LastUpdated:     time.UnixMicro(1712131205000000),
			Price:           61000,
			Amount:          1,
			RemainingAmount: 1,
		},
		&order.Detail{
			Exchange:       ex.Name,
			OrderID:        "1500000007",
			Type:           order.Limit,
			TimeInForce:    order.GoodTillCancel,
			Side:           order.Buy,
			Status:         order.Filled,
			AssetType:      asset.PerpetualContract,
			Pair:           wsPerpetualPair,
			Date:           time.UnixMicro(1712131206000000),
			LastUpdated:    time.UnixMicro(1712131206000000),
			Price:          61000,
			Amount:         1,
			ExecutedAmount: 1,
		},
		&order.Detail{
			Exchange:        ex.Name,
			OrderID:         "1500000009",
			Type:            order.TrailingStop,
			Side:            order.Sell,
			Status:          order.Active,
			AssetType:       asset.PerpetualContract,
			Pair:            wsPerpetualPair,
			Date:            time.UnixMicro(1712131207000000),
			LastUpdated:     time.UnixMicro(1712131207000000),
			Amount:          0.01,
			RemainingAmount: 0.01,
			TriggerPrice:    50000,
		},
		websocket.UnhandledMessageWarning{Message: fmt.Sprintf("%s%sorder_unknown %+v", ex.Name, websocket.UnhandledMessage, &OrderEventData{ID: 1500000008})},
	}, dataHandlerPayloads(ex), "order events should be sent to the data handler")

	// A replacement and the order it replaced take one payload, so the data handler accepts or refuses them together
	ex.Websocket.DataHandler = stream.NewRelay(1)
	require.NoError(t, ex.processOrderEvent(t.Context(), orderEventReplaced, &OrderEventData{ID: 2, OriginalOrderID: 1}, wsPerpetualPair, asset.PerpetualContract), "processOrderEvent must relay a replacement to a data handler with room for one payload")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "the data handler must receive one payload")
	replacement, ok := (<-ex.Websocket.DataHandler.C).Data.([]order.Detail)
	require.True(t, ok, "a replacement must be relayed as order details")
	assert.Len(t, replacement, 2, "the payload should hold the replacement and the order it replaced")
}

// orderEventFrame returns a private orders channel event for an order, chained to the event before it
func orderEventFrame(eventID, preEventID string, orderID uint64) []byte {
	pre := "null"
	if preEventID != "" {
		pre = `"` + preEventID + `"`
	}
	return fmt.Appendf(nil, `{"event":"order_changed","channel":"private-my_orders_btcusd-123456","event_id":%q,"pre_event_id":%s,"order_source":"orderbook","data":{"id":%d,"amount_str":"0.5","amount_at_create":"1","price_str":"61200","order_type":0,"microtimestamp":"1712131200000000"}}`, eventID, pre, orderID)
}

func TestHandleWebsocketAccountOrderGapRecovery(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var requests []url.Values
	failRecovery := true
	ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/account_order_data/", r.URL.Path, "path should be the account order data")
		assert.NoError(t, r.ParseForm(), "ParseForm should not error")
		mu.Lock()
		defer mu.Unlock()
		if failRecovery {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		requests = append(requests, r.PostForm)
		// Events are returned from the since ID inclusive, at most 1000 at a time
		var events []string
		switch r.PostForm.Get("since_id") {
		case "event-2":
			for i := 2; i <= 1001; i++ {
				events = append(events, fmt.Sprintf(`{"event":"order_changed","event_id":"event-%d","order_source":"orderbook","data":{"id":%d,"amount_str":"0.5","amount_at_create":"1"}}`, i, i))
			}
		case "event-1001":
			events = append(events,
				`{"event":"order_changed","event_id":"event-1001","order_source":"orderbook","data":{"id":1001,"amount_str":"0.5","amount_at_create":"1"}}`,
				`{"event":"order_deleted","event_id":"event-1002","order_source":"orderbook","data":{"id":1002,"amount_str":"0","amount_at_create":"1"}}`)
		}
		_, err := w.Write([]byte("[" + strings.Join(events, ",") + "]"))
		assert.NoError(t, err, "Write should not error")
	})
	ex.Websocket.DataHandler = stream.NewRelay(2000)

	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 1)), "the first event must be processed")
	assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "the first event should be processed without recovery")
	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-2", "event-1", 2)), "a chained event must be processed")
	assert.Equal(t, []string{"2"}, relayedOrderIDs(t, ex), "a chained event should be processed without recovery")

	// A gap whose recovery fails holds the event back, and is retried from the same event with the next event
	err := ex.wsHandleData(t.Context(), orderEventFrame("event-1002", "event-1001", 1002))
	assert.ErrorIs(t, err, errOrderEventRecovery, "a failed recovery should be reported")
	assert.Empty(t, relayedOrderIDs(t, ex), "an event should not be relayed before the events missed ahead of it")

	mu.Lock()
	failRecovery = false
	mu.Unlock()
	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1003", "event-1002", 1003)), "an event after a gap must be processed")
	ids := relayedOrderIDs(t, ex)
	require.Len(t, ids, 1001, "every missed event must be processed once before the event")
	assert.Equal(t, "3", ids[0], "recovered events should start after the last processed event")
	assert.Equal(t, "1001", ids[998], "recovered events should be paged")
	assert.Equal(t, "1002", ids[999], "the event held back by the failed recovery should be recovered")
	assert.Equal(t, "1003", ids[1000], "the event should be processed after the recovered events")
	mu.Lock()
	assert.Equal(t, []url.Values{
		{"market": {"BTC/USD"}, "order_source": {"orderbook"}, "since_id": {"event-2"}, "until_id": {"event-1003"}},
		{"market": {"BTC/USD"}, "order_source": {"orderbook"}, "since_id": {"event-1001"}, "until_id": {"event-1003"}},
	}, requests, "recovery should resume from the last processed event and page from the last recovered event")
	mu.Unlock()

	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1004", "event-1003", 1004)), "a chained event must be processed")
	assert.Equal(t, []string{"1004"}, relayedOrderIDs(t, ex), "an event chained to a recovered gap should be processed without recovery")
}

// relayedOrderIDs returns the IDs of the orders relayed to the data handler since it was last read
func relayedOrderIDs(t *testing.T, ex *Exchange) []string {
	t.Helper()
	var ids []string
	for {
		select {
		case p := <-ex.Websocket.DataHandler.C:
			d, ok := p.Data.(*order.Detail)
			require.Truef(t, ok, "the data handler must only receive order details, not %T %v", p.Data, p.Data)
			ids = append(ids, d.OrderID)
		default:
			return ids
		}
	}
}

// fullDataHandler returns a data handler whose buffer is full, so it refuses everything sent to it
func fullDataHandler(t *testing.T) *stream.Relay {
	t.Helper()
	r := stream.NewRelay(1)
	require.NoError(t, r.Send(t.Context(), "filler"), "Send must not error")
	return r
}

// newOrderEventsTestExchange returns an exchange whose account order data requests return events event-N for order N,
// from the since ID inclusive to the until ID exclusive and at most 1000 at a time as the API does, with no order ID
// for the events listed in withoutOrderID, and a function returning the since ID of every request
// A request fails when fail returns true for its since ID, and like the API, a range which is empty is rejected
func newOrderEventsTestExchange(t *testing.T, fail func(sinceID string) bool, withoutOrderID ...int) (ex *Exchange, sinceIDs func() []string) {
	t.Helper()
	var mu sync.Mutex
	var received []string
	ex = newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/account_order_data/", r.URL.Path, "path should be the account order data")
		assert.NoError(t, r.ParseForm(), "ParseForm should not error")
		sinceID := r.PostForm.Get("since_id")
		mu.Lock()
		received = append(received, sinceID)
		mu.Unlock()
		since, err := strconv.Atoi(strings.TrimPrefix(sinceID, "event-"))
		assert.NoError(t, err, "since_id should name an event")
		until, err := strconv.Atoi(strings.TrimPrefix(r.PostForm.Get("until_id"), "event-"))
		assert.NoError(t, err, "until_id should name an event")
		if until <= since {
			w.WriteHeader(http.StatusBadRequest)
			_, err = w.Write([]byte(`{"status":"error","reason":"Invalid event ID range: until_id must be greater than since_id.","response_code":"400.085"}`))
			assert.NoError(t, err, "Write should not error")
			return
		}
		if fail != nil && fail(sinceID) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var events []string
		for i := since; i < until && len(events) < maxResultLimit; i++ {
			orderID := i
			if slices.Contains(withoutOrderID, i) {
				orderID = 0
			}
			events = append(events, fmt.Sprintf(`{"event":"order_changed","event_id":"event-%d","order_source":"orderbook","data":{"id":%d,"amount_str":"0.5","amount_at_create":"1"}}`, i, orderID))
		}
		_, err = w.Write([]byte("[" + strings.Join(events, ",") + "]"))
		assert.NoError(t, err, "Write should not error")
	})
	ex.Websocket.DataHandler = stream.NewRelay(100)
	return ex, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(received)
	}
}

func TestHandleWebsocketAccountOrderRefusedEvents(t *testing.T) {
	t.Parallel()
	t.Run("later event", func(t *testing.T) {
		t.Parallel()
		ex, sinceIDs := newOrderEventsTestExchange(t, nil)
		require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 1)), "the first event must be processed")
		ex.Websocket.DataHandler = fullDataHandler(t)
		assert.Error(t, ex.wsHandleData(t.Context(), orderEventFrame("event-2", "event-1", 2)), "a refused event should error")
		ex.Websocket.DataHandler = stream.NewRelay(100)
		require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-3", "event-2", 3)), "the next event must be processed")
		assert.Equal(t, []string{"2", "3"}, relayedOrderIDs(t, ex), "the refused event should be recovered before the next event")
		assert.Equal(t, []string{"event-2"}, sinceIDs(), "recovery should start with the refused event")
	})
	for name, preEventID := range map[string]string{"first event": "event-0", "first event after a server restart": ""} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ex, sinceIDs := newOrderEventsTestExchange(t, nil)
			ex.Websocket.DataHandler = fullDataHandler(t)
			assert.Error(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", preEventID, 1)), "a refused event should error")
			ex.Websocket.DataHandler = stream.NewRelay(100)
			require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-2", "event-1", 2)), "the next event must be processed")
			assert.Equal(t, []string{"1", "2"}, relayedOrderIDs(t, ex), "the refused event should be recovered before the next event")
			assert.Equal(t, []string{"event-1"}, sinceIDs(), "recovery should start with the refused event")
		})
	}
	t.Run("event after recovery", func(t *testing.T) {
		t.Parallel()
		ex, sinceIDs := newOrderEventsTestExchange(t, nil)
		require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 1)), "the first event must be processed")
		assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "the first event should be relayed")
		// The data handler takes the two recovered events and refuses the event they were recovered for
		ex.Websocket.DataHandler = stream.NewRelay(2)
		assert.Error(t, ex.wsHandleData(t.Context(), orderEventFrame("event-4", "event-3", 4)), "a refused event should error")
		assert.Equal(t, []string{"2", "3"}, relayedOrderIDs(t, ex), "the missed events should be relayed")
		ex.Websocket.DataHandler = stream.NewRelay(100)
		require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-5", "event-4", 5)), "the next event must be processed")
		assert.Equal(t, []string{"4", "5"}, relayedOrderIDs(t, ex), "only the refused event should be recovered before the next event")
		assert.Equal(t, []string{"event-1", "event-4"}, sinceIDs(), "recovery should resume with the refused event")
	})
	t.Run("recovered event", func(t *testing.T) {
		t.Parallel()
		ex, sinceIDs := newOrderEventsTestExchange(t, nil)
		require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 1)), "the first event must be processed")
		assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "the first event should be relayed")
		// The data handler takes one recovered event and refuses the next
		ex.Websocket.DataHandler = stream.NewRelay(1)
		err := ex.wsHandleData(t.Context(), orderEventFrame("event-4", "event-3", 4))
		assert.ErrorIs(t, err, errOrderEventRecovery, "a refused recovered event should stop recovery")
		assert.Equal(t, []string{"2"}, relayedOrderIDs(t, ex), "the event should be held back with the events after the refused event")
		ex.Websocket.DataHandler = stream.NewRelay(100)
		require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-5", "event-4", 5)), "the next event must be processed")
		assert.Equal(t, []string{"3", "4", "5"}, relayedOrderIDs(t, ex), "recovery should resume with the refused event")
		assert.Equal(t, []string{"event-1", "event-3"}, sinceIDs(), "recovery should resume with the refused event")
	})
}

func TestHandleWebsocketAccountOrderWithoutOrderID(t *testing.T) {
	t.Parallel()
	ex, sinceIDs := newOrderEventsTestExchange(t, nil, 3)
	assert.ErrorIs(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 0)), errOrderIDMissing, "an event without an order ID should error")
	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-2", "event-1", 2)), "an event chained to an event without an order ID must be processed")
	assert.Empty(t, sinceIDs(), "an event without an order ID should not be recovered, as its recovered copy has no order ID either")
	assert.Equal(t, []string{"2"}, relayedOrderIDs(t, ex), "the chained event should be relayed")

	// Recovered events without an order ID are reported and skipped, so recovery does not stop at them
	err := ex.wsHandleData(t.Context(), orderEventFrame("event-5", "event-4", 5))
	assert.ErrorIs(t, err, errOrderIDMissing, "a recovered event without an order ID should be reported")
	assert.NotErrorIs(t, err, errOrderEventRecovery, "a recovered event without an order ID should not stop recovery")
	assert.Equal(t, []string{"4", "5"}, relayedOrderIDs(t, ex), "the events after an event without an order ID should be relayed")
	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-6", "event-5", 6)), "a chained event must be processed")
	assert.Equal(t, []string{"6"}, relayedOrderIDs(t, ex), "a chained event should be relayed without recovery")
	assert.Equal(t, []string{"event-2"}, sinceIDs(), "recovery should only be requested for the gap")
}

func TestHandleWebsocketAccountOrderRepeatedEvent(t *testing.T) {
	t.Parallel()
	t.Run("relayed event", func(t *testing.T) {
		t.Parallel()
		ex, sinceIDs := newOrderEventsTestExchange(t, nil)
		require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 1)), "the first event must be processed")
		require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 1)), "a repeated event must be processed")
		assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "a repeated event should not be relayed again")
		require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-2", "event-1", 2)), "the next event must be processed")
		assert.Equal(t, []string{"2"}, relayedOrderIDs(t, ex), "the next event should be relayed")
		assert.Empty(t, sinceIDs(), "a repeated event should not be recovered")
	})
	t.Run("refused event", func(t *testing.T) {
		t.Parallel()
		ex, sinceIDs := newOrderEventsTestExchange(t, nil)
		ex.Websocket.DataHandler = fullDataHandler(t)
		assert.Error(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 1)), "a refused event should error")
		ex.Websocket.DataHandler = stream.NewRelay(100)
		require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 1)), "a repeated refused event must be processed")
		assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "a repeated refused event should be relayed")
		require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-2", "event-1", 2)), "the next event must be processed")
		assert.Equal(t, []string{"2"}, relayedOrderIDs(t, ex), "the next event should be relayed")
		assert.Empty(t, sinceIDs(), "a repeated event should not be recovered")
	})
}

func TestHandleWebsocketAccountOrderRecoveryPageFailure(t *testing.T) {
	t.Parallel()
	var failed atomic.Bool
	ex, sinceIDs := newOrderEventsTestExchange(t, func(sinceID string) bool {
		return sinceID == "event-1000" && failed.CompareAndSwap(false, true)
	})
	ex.Websocket.DataHandler = stream.NewRelay(2000)
	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 1)), "the first event must be processed")
	assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "the first event should be relayed")

	err := ex.wsHandleData(t.Context(), orderEventFrame("event-1003", "event-1002", 1003))
	assert.ErrorIs(t, err, errOrderEventRecovery, "a page which cannot be fetched should be reported")
	ids := relayedOrderIDs(t, ex)
	require.Len(t, ids, 999, "the events of the first page must be relayed")
	assert.Equal(t, "2", ids[0], "recovered events should start after the last relayed event")
	assert.Equal(t, "1000", ids[998], "recovered events should end with the first page")

	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1004", "event-1003", 1004)), "the next event must be processed")
	assert.Equal(t, []string{"1001", "1002", "1003", "1004"}, relayedOrderIDs(t, ex), "recovery should resume after the first page without relaying it again")
	assert.Equal(t, []string{"event-1", "event-1000", "event-1000"}, sinceIDs(), "recovery should resume from the last relayed event")
}

func TestHandleWebsocketAccountOrderRecoveryGiveUp(t *testing.T) {
	t.Parallel()
	frame := func(id uint64) []byte {
		return orderEventFrame("event-"+strconv.FormatUint(id, 10), "event-"+strconv.FormatUint(id-1, 10), id)
	}
	t.Run("recovery keeps failing", func(t *testing.T) {
		t.Parallel()
		ex, sinceIDs := newOrderEventsTestExchange(t, func(string) bool { return true })
		require.NoError(t, ex.wsHandleData(t.Context(), frame(1)), "the first event must be processed")
		assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "the first event should be relayed")
		for _, id := range []uint64{4, 5} {
			err := ex.wsHandleData(t.Context(), frame(id))
			assert.ErrorIsf(t, err, errOrderEventRecovery, "the failed recovery before event %d should be reported", id)
			assert.NotErrorIsf(t, err, errOrderEventsSkipped, "the failed recovery before event %d should not skip the missed events", id)
			assert.Emptyf(t, relayedOrderIDs(t, ex), "event %d should be held back while recovery fails", id)
		}
		err := ex.wsHandleData(t.Context(), frame(6))
		assert.ErrorIs(t, err, errOrderEventRecovery, "the third failed recovery should be reported")
		assert.ErrorIs(t, err, errOrderEventsSkipped, "the third failed recovery in a row should skip the missed events")
		assert.Equal(t, []string{"6"}, relayedOrderIDs(t, ex), "the event should be relayed once the missed events are skipped")
		require.NoError(t, ex.wsHandleData(t.Context(), frame(7)), "an event chained to the skipped events must be processed")
		assert.Equal(t, []string{"7"}, relayedOrderIDs(t, ex), "an event chained to the skipped events should be relayed without recovery")
		assert.Equal(t, []string{"event-1", "event-1", "event-1"}, sinceIDs(), "recovery should be retried from the last relayed event until the missed events are skipped")
	})
	t.Run("recovery succeeds between failures", func(t *testing.T) {
		t.Parallel()
		var failing atomic.Bool
		failing.Store(true)
		ex, _ := newOrderEventsTestExchange(t, func(string) bool { return failing.Load() })
		require.NoError(t, ex.wsHandleData(t.Context(), frame(1)), "the first event must be processed")
		for _, id := range []uint64{4, 5} {
			assert.ErrorIsf(t, ex.wsHandleData(t.Context(), frame(id)), errOrderEventRecovery, "the failed recovery before event %d should be reported", id)
		}
		failing.Store(false)
		require.NoError(t, ex.wsHandleData(t.Context(), frame(6)), "an event after a recovered gap must be processed")
		assert.Equal(t, []string{"1", "2", "3", "4", "5", "6"}, relayedOrderIDs(t, ex), "the missed events should be recovered before the event")
		failing.Store(true)
		err := ex.wsHandleData(t.Context(), frame(9))
		assert.ErrorIs(t, err, errOrderEventRecovery, "a failed recovery should be reported")
		assert.NotErrorIs(t, err, errOrderEventsSkipped, "a failed recovery after a successful one should not skip the missed events")
		assert.Empty(t, relayedOrderIDs(t, ex), "the event should be held back while recovery fails")
	})
	t.Run("no permission", func(t *testing.T) {
		t.Parallel()
		ex := newResponseTestExchange(t, http.StatusForbidden, `{"status":"error","reason":"No permission found","code":"API0003"}`)
		ex.Websocket.DataHandler = stream.NewRelay(100)
		require.NoError(t, ex.wsHandleData(t.Context(), frame(1)), "the first event must be processed")
		err := ex.wsHandleData(t.Context(), frame(4))
		assert.ErrorIs(t, err, errNoPermission, "the permission error should be reported")
		assert.ErrorIs(t, err, errOrderEventsSkipped, "a recovery the API key lacks permission for should skip the missed events at once")
		assert.Equal(t, []string{"1", "4"}, relayedOrderIDs(t, ex), "the event should be relayed without the missed events")
	})
}

func TestHandleWebsocketAccountOrderStaleEvent(t *testing.T) {
	t.Parallel()
	var failing atomic.Bool
	ex, sinceIDs := newOrderEventsTestExchange(t, func(string) bool { return failing.Load() })
	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 1)), "the first event must be processed")
	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-2", "event-1", 2)), "a chained event must be processed")
	assert.Equal(t, []string{"1", "2"}, relayedOrderIDs(t, ex), "the events should be relayed")
	for range maxOrderEventRecoveryFailures {
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 1)), errStaleOrderEvent, "an event before the last one handled should be dropped")
	}
	assert.Empty(t, relayedOrderIDs(t, ex), "an event before the last one handled should not be relayed")

	// Dropped events are not failed recoveries, so a later gap is still held back until recovery has failed three times
	failing.Store(true)
	for _, id := range []uint64{5, 6} {
		err := ex.wsHandleData(t.Context(), orderEventFrame("event-"+strconv.FormatUint(id, 10), "event-"+strconv.FormatUint(id-1, 10), id))
		assert.ErrorIsf(t, err, errOrderEventRecovery, "the failed recovery before event %d should be reported", id)
		assert.NotErrorIsf(t, err, errOrderEventsSkipped, "the failed recovery before event %d should not skip the missed events", id)
	}
	assert.Empty(t, relayedOrderIDs(t, ex), "events should be held back while recovery fails")
	failing.Store(false)
	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-7", "event-6", 7)), "an event after a recovered gap must be processed")
	assert.Equal(t, []string{"3", "4", "5", "6", "7"}, relayedOrderIDs(t, ex), "the missed events should be recovered before the event")
	assert.Equal(t, []string{"event-2", "event-2", "event-2", "event-2", "event-2", "event-2"}, sinceIDs(), "recovery should be requested from the last relayed event")
}

func TestRecoverOrderEvents(t *testing.T) {
	t.Parallel()
	ex, sinceIDs := newOrderEventsTestExchange(t, nil, 3)
	checkpoint, err := ex.recoverOrderEvents(t.Context(), wsSpotPair, asset.Spot, "orderbook", orderEventCheckpoint{eventID: "event-1"}, "event-5")
	assert.ErrorIs(t, err, errOrderIDMissing, "recoverOrderEvents should report an event without an order ID")
	assert.NotErrorIs(t, err, errOrderEventRecovery, "recoverOrderEvents should not stop at an event without an order ID")
	assert.Equal(t, orderEventCheckpoint{eventID: "event-4"}, checkpoint, "recoverOrderEvents should return the last event handled")
	assert.Equal(t, []string{"2", "4"}, relayedOrderIDs(t, ex), "recoverOrderEvents should relay the events after the checkpoint")

	checkpoint, err = ex.recoverOrderEvents(t.Context(), wsSpotPair, asset.Spot, "orderbook", orderEventCheckpoint{eventID: "event-4", pending: true}, "event-6")
	require.NoError(t, err, "recoverOrderEvents must not error")
	assert.Equal(t, orderEventCheckpoint{eventID: "event-5"}, checkpoint, "recoverOrderEvents should return the last event handled")
	assert.Equal(t, []string{"4", "5"}, relayedOrderIDs(t, ex), "recoverOrderEvents should relay a pending checkpoint's event")

	ex.Websocket.DataHandler = stream.NewRelay(1)
	checkpoint, err = ex.recoverOrderEvents(t.Context(), wsSpotPair, asset.Spot, "orderbook", orderEventCheckpoint{eventID: "event-5"}, "event-9")
	assert.ErrorIs(t, err, errOrderEventRecovery, "recoverOrderEvents should error when the data handler refuses an event")
	assert.Equal(t, orderEventCheckpoint{eventID: "event-7", pending: true}, checkpoint, "recoverOrderEvents should return the refused event as pending")
	assert.Equal(t, []string{"6"}, relayedOrderIDs(t, ex), "recoverOrderEvents should stop at the refused event")
	assert.Equal(t, []string{"event-1", "event-4", "event-5"}, sinceIDs(), "recoverOrderEvents should request events from the checkpoint")

	ex = newTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	checkpoint, err = ex.recoverOrderEvents(t.Context(), wsSpotPair, asset.Spot, "orderbook", orderEventCheckpoint{eventID: "event-1"}, "event-3")
	assert.ErrorIs(t, err, errOrderEventRecovery, "recoverOrderEvents should error when the events cannot be fetched")
	assert.Equal(t, orderEventCheckpoint{eventID: "event-1"}, checkpoint, "recoverOrderEvents should keep the checkpoint when the events cannot be fetched")
}

func TestSetOrderEventCheckpoint(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	ex.setOrderEventCheckpoint("BTC/USD|spot|orderbook", orderEventCheckpoint{eventID: "event-1", pending: true})
	ex.setOrderEventCheckpoint("BTC/USD|spot|stop_order", orderEventCheckpoint{eventID: "event-2"})
	ex.setOrderEventCheckpoint("BTC/USD|spot|orderbook", orderEventCheckpoint{eventID: "event-3"})
	assert.Equal(t, map[string]orderEventCheckpoint{
		"BTC/USD|spot|orderbook":  {eventID: "event-3"},
		"BTC/USD|spot|stop_order": {eventID: "event-2"},
	}, ex.orderEventCheckpoints, "setOrderEventCheckpoint should store the last checkpoint of each key")
}

func TestIsOrderEventRetryable(t *testing.T) {
	t.Parallel()
	assert.False(t, isOrderEventRetryable(nil), "a relayed event should not be retried")
	missingOrderID := fmt.Errorf("%w: order_changed", errOrderIDMissing)
	assert.False(t, isOrderEventRetryable(missingOrderID), "an event without an order ID should not be retried")
	assert.True(t, isOrderEventRetryable(fullDataHandler(t).Send(t.Context(), nil)), "an event the data handler refused should be retried")
}

func TestHandleWebsocketAccountOrderRestart(t *testing.T) {
	t.Parallel()
	var recovered []url.Values
	var mu sync.Mutex
	ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, r.ParseForm(), "ParseForm should not error")
		mu.Lock()
		recovered = append(recovered, r.PostForm)
		mu.Unlock()
		_, err := w.Write([]byte(`[]`))
		assert.NoError(t, err, "Write should not error")
	})
	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "", 1)), "an event after a restart must be processed")
	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-2", "event-1", 2)), "a chained event must be processed")
	mu.Lock()
	assert.Empty(t, recovered, "events should not be recovered without a gap or a known previous event")
	mu.Unlock()
	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-3", "", 3)), "an event after a server restart must be processed")
	mu.Lock()
	assert.Equal(t, []url.Values{{"market": {"BTC/USD"}, "order_source": {"orderbook"}, "since_id": {"event-2"}, "until_id": {"event-3"}}}, recovered, "a server restart should recover events from the last processed event")
	mu.Unlock()
}

func TestOrderTypeFromSubtypeID(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		subtype uint8
		typ     order.Type
		tif     order.TimeInForce
	}{
		{0, order.Limit, order.GoodTillCancel},
		{1, order.Market, order.UnknownTIF},
		{2, order.Market, order.UnknownTIF},
		{3, order.Limit, order.GoodTillDay},
		{4, order.Limit, order.ImmediateOrCancel},
		{5, order.Limit, order.PostOnly},
		{6, order.Limit, order.FillOrKill},
		{7, order.Market, order.UnknownTIF},
		{8, order.Limit, order.GoodTillTime},
		{20, order.StopMarket, order.UnknownTIF},
		{21, order.TakeProfitMarket, order.UnknownTIF},
		{22, order.StopLimit, order.UnknownTIF},
		{23, order.TakeProfit | order.Limit, order.UnknownTIF},
		{24, order.TrailingStop, order.UnknownTIF},
		{25, order.TrailingStop, order.UnknownTIF},
		{26, order.TrailingStopLimit, order.UnknownTIF},
		{27, order.TrailingStopLimit, order.UnknownTIF},
		{99, order.UnknownType, order.UnknownTIF},
	} {
		typ, tif := orderTypeFromSubtypeID(tc.subtype)
		assert.Equalf(t, tc.typ, typ, "orderTypeFromSubtypeID should return the order type for %d", tc.subtype)
		assert.Equalf(t, tc.tif, tif, "orderTypeFromSubtypeID should return the time in force for %d", tc.subtype)
	}
}

func TestHandleWebsocketMyTrade(t *testing.T) {
	t.Parallel()
	data := []byte(`{"id":296050733,"id_str":"296050733","trade_uti":"BSTP-20240403-296050733","order_id":1500000001,"client_order_id":"my-order-001","amount":0.1,"price":61200.00,"fee":"0.01","side":"buy","microtimestamp":"1712131200500000","trade_account_id":0,"position_id":null,"is_liquidation":null,"trade_type":"ORDERBOOK"}`)
	expOrder := &order.Detail{
		Exchange:      "",
		OrderID:       "1500000001",
		ClientOrderID: "my-order-001",
		AssetType:     asset.PerpetualContract,
		Pair:          wsPerpetualPair,
		Side:          order.Buy,
		LastUpdated:   time.UnixMicro(1712131200500000),
		Trades: []order.TradeHistory{{
			TID:       "296050733",
			Price:     61200,
			Amount:    0.1,
			Fee:       0.01,
			Side:      order.Buy,
			Timestamp: time.UnixMicro(1712131200500000),
			FeeAsset:  "USD",
		}},
	}

	ex := newWebsocketTestInstance(t)
	expOrder.Exchange, expOrder.Trades[0].Exchange = ex.Name, ex.Name
	require.NoError(t, ex.handleWSMyTrade(t.Context(), data, wsPerpetualPair, asset.PerpetualContract), "handleWSMyTrade must not error with the fills feed disabled")
	assert.Equal(t, []any{expOrder}, dataHandlerPayloads(ex), "executions should be sent as order updates with the fills feed disabled")

	ex = newFeedTestInstance(t)
	expOrder.Exchange, expOrder.Trades[0].Exchange = ex.Name, ex.Name
	assert.Error(t, ex.handleWSMyTrade(t.Context(), []byte(`{"price":[]}`), wsPerpetualPair, asset.PerpetualContract), "handleWSMyTrade should error on malformed data")
	err := ex.handleWSMyTrade(t.Context(), []byte(`{"id":1,"side":"wibble"}`), wsPerpetualPair, asset.PerpetualContract)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "handleWSMyTrade should error on an invalid side")
	require.NoError(t, ex.handleWSMyTrade(t.Context(), data, wsPerpetualPair, asset.PerpetualContract), "handleWSMyTrade must not error")
	require.NoError(t, ex.handleWSMyTrade(t.Context(), []byte(`{"client_order_id":"my-order-002","amount":0.2,"price":61000,"side":"sell","microtimestamp":"1712131201000000"}`), wsSpotPair, asset.Spot), "handleWSMyTrade must not error without IDs")
	assert.Equal(t, []any{
		[]fill.Data{{
			Timestamp:     time.UnixMicro(1712131200500000),
			Exchange:      ex.Name,
			AssetType:     asset.PerpetualContract,
			CurrencyPair:  wsPerpetualPair,
			Side:          order.Buy,
			OrderID:       "1500000001",
			ClientOrderID: "my-order-001",
			TradeID:       "296050733",
			Price:         61200,
			Amount:        0.1,
		}},
		expOrder,
		[]fill.Data{{
			Timestamp:     time.UnixMicro(1712131201000000),
			Exchange:      ex.Name,
			AssetType:     asset.Spot,
			CurrencyPair:  wsSpotPair,
			Side:          order.Sell,
			ClientOrderID: "my-order-002",
			Price:         61000,
			Amount:        0.2,
		}},
	}, dataHandlerPayloads(ex), "fills and order updates should be sent to the data handler, and executions without IDs only as fills")
}

func TestHandleWebsocketDetailOrderbook(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	assert.Error(t, ex.handleWSDetailOrderbook(t.Context(), []byte(`{"bids":"wibble"}`), wsSpotPair, asset.Spot), "handleWSDetailOrderbook should error on malformed data")
	err := ex.handleWSDetailOrderbook(t.Context(), []byte(`{"bids":[["1","1","9223372036854775808"]]}`), wsSpotPair, asset.Spot)
	assert.ErrorIs(t, err, errOrderIDOverflow, "handleWSDetailOrderbook should error on an order ID beyond an orderbook level ID")
	err = ex.handleWSDetailOrderbook(t.Context(), []byte(`{"asks":[["1","1","9223372036854775808"]]}`), wsSpotPair, asset.Spot)
	assert.ErrorIs(t, err, errOrderIDOverflow, "handleWSDetailOrderbook should error on an ask order ID beyond an orderbook level ID")

	data := `{"timestamp":"1791848800","microtimestamp":"1791848800102345","bids":[["81730.34","0.01000000","2060344614850561"],["81730.34","0.02000000","2060344614850562"],["0","1","1"]],"asks":[["81758.04","0.03000000","2060344614850560"]]}`
	require.NoError(t, ex.handleWSDetailOrderbook(t.Context(), []byte(data), wsSpotPair, asset.Spot), "handleWSDetailOrderbook must not error")
	ob, err := ex.Websocket.Orderbook.GetOrderbook(wsSpotPair, asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.NotZero(t, ob.InsertedAt, "InsertedAt should be set")
	ob.InsertedAt = time.Time{}
	// Each level holds an order, so prices repeat and the zero priced bid is removed
	exp := &orderbook.Book{
		Bids:              orderbook.Levels{{Price: 81730.34, Amount: 0.01, ID: 2060344614850561}, {Price: 81730.34, Amount: 0.02, ID: 2060344614850562}},
		Asks:              orderbook.Levels{{Price: 81758.04, Amount: 0.03, ID: 2060344614850560}},
		Exchange:          ex.Name,
		Pair:              wsSpotPair,
		Asset:             asset.Spot,
		LastUpdated:       time.UnixMicro(1791848800102345),
		PriceDuplication:  true,
		ValidateOrderbook: ex.ValidateOrderbook,
	}
	assert.Equal(t, exp, ob, "handleWSDetailOrderbook should load the snapshot of orders")
}

func TestHandleWebsocketSelfTrade(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	assert.Error(t, ex.handleWSSelfTrade(t.Context(), []byte(`{"amount_str":[]}`), wsSpotPair, asset.Spot), "handleWSSelfTrade should error on malformed data")
	data := []byte(`{"buy_order_id":1500000010,"sell_order_id":1500000011,"amount":0.05,"amount_str":"0.05000000","price":61200.00,"price_str":"61200.00","type":0,"timestamp":"1712131205","microtimestamp":"1712131205000000","sellers_trade_account_id":7,"buyers_trade_account_id":0}`)
	require.NoError(t, ex.handleWSSelfTrade(t.Context(), data, wsSpotPair, asset.Spot), "handleWSSelfTrade must not error")
	assert.Equal(t, []any{&WebsocketSelfTrade{
		Pair:                  wsSpotPair,
		AssetType:             asset.Spot,
		BuyOrderID:            1500000010,
		SellOrderID:           1500000011,
		Amount:                0.05,
		Price:                 61200,
		Side:                  OrderSide(order.Buy),
		Timestamp:             types.Time(time.Unix(1712131205, 0)),
		Microtimestamp:        types.Time(time.UnixMicro(1712131205000000)),
		SellersTradeAccountID: 7,
	}}, dataHandlerPayloads(ex), "self trades should be sent to the data handler")
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

func TestHandleWebsocketSettlement(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	assert.Error(t, ex.handleWSSettlement(t.Context(), "settlement", []byte(`{"as":[]}`)), "handleWSSettlement should error on malformed data")
	for event, data := range map[string]string{
		"settlement":             `{"id":"1234567890","tnx_id":"123123","ts":"1712131200","price":"61200","ccy":"USD","as":"-1.5","ap":"-1","atf":"-0.25","alf":"0","afu":"-0.25","asl":"0","sp":"61000","typ":"C"}`,
		"insurance_fund_premium": `{"tnx_id":"123124","ts":"1712131200","ccy":"USD","amount":"-0.5"}`,
	} {
		require.NoErrorf(t, ex.handleWSSettlement(t.Context(), event, []byte(data)), "handleWSSettlement must not error for %s", event)
	}
	assert.ElementsMatch(t, []any{
		&WebsocketSettlement{
			Event:                 "settlement",
			PositionID:            "1234567890",
			TransactionID:         "123123",
			Timestamp:             types.Time(time.Unix(1712131200, 0)),
			Price:                 61200,
			Currency:              "USD",
			AmountSettled:         -1.5,
			AmountFromPrice:       -1,
			AmountFromTradingFees: -0.25,
			AmountFromFunding:     -0.25,
			StrikePrice:           61000,
			SettlementType:        "C",
		},
		&WebsocketSettlement{
			Event:         "insurance_fund_premium",
			TransactionID: "123124",
			Timestamp:     types.Time(time.Unix(1712131200, 0)),
			Currency:      "USD",
			Amount:        -0.5,
		},
	}, dataHandlerPayloads(ex), "settlements should be sent to the data handler")
}

func TestHandleWebsocketAnnouncement(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	assert.Error(t, ex.handleWSAnnouncement(t.Context(), "published", []byte(`{"announcement_id":"wibble"}`)), "handleWSAnnouncement should error on malformed data")
	data := []byte(`{"announcement_id":12345,"announcement_type":"trading_halt","status":"in_progress","published_at":"2026-08-25T10:00:00Z","updated_at":"2026-08-25T10:00:00Z","product_type":"TOKENIZED_SECURITY","product":["TSLA/USD"],"trading_halt_details":{"reason":"circuit_breaker","start_time":"2026-08-25T10:00:00Z","end_time":null}}`)
	require.NoError(t, ex.handleWSAnnouncement(t.Context(), "in_progress", data), "handleWSAnnouncement must not error")
	haltStart := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	assert.Equal(t, []any{&WebsocketAnnouncement{
		Event:              "in_progress",
		AnnouncementID:     12345,
		AnnouncementType:   "trading_halt",
		Status:             "in_progress",
		PublishedAt:        haltStart,
		UpdatedAt:          haltStart,
		ProductType:        "TOKENIZED_SECURITY",
		Product:            []string{"TSLA/USD"},
		TradingHaltDetails: &TradingHaltDetails{Reason: "circuit_breaker", StartTime: haltStart},
	}}, dataHandlerPayloads(ex), "announcements should be sent to the data handler")
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
	} {
		var buf bytes.Buffer
		require.NoErrorf(t, tpl.Execute(&buf, map[string]any{"S": tc.s, "AssetPairs": tc.assetPairs, "PairSeparator": "|", "AssetSeparator": "#"}), "Execute must not error for %s", tc.s.Channel)
		assert.Equalf(t, tc.exp, strings.Join(strings.Fields(buf.String()), ""), "the template should render a channel per pair and a record per asset for %s", tc.s.Channel)
	}
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
