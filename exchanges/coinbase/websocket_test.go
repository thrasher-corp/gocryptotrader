package coinbase

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
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
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	testETHUSD  = currency.NewPair(currency.ETH, currency.USD)
	testETHUSDC = currency.NewPair(currency.ETH, currency.USDC)
	// testFuturesPair is the US derivatives perpetual style future the websocket fixtures stream
	testFuturesPair = currency.NewPairWithDelimiter("BIP", "20DEC30-CDE", currency.DashDelimiter)
	// testUSDCPerpetual and testCoinPerpetual are a perpetual of each gateway asset
	testUSDCPerpetual = currency.NewPairWithDelimiter("BTC", "USDC-PERPETUAL", currency.UnderscoreDelimiter)
	testCoinPerpetual = currency.NewPairWithDelimiter("BTC", "PERPETUAL", currency.DashDelimiter)
)

// testWebsocketPairs are each asset's pairs in the websocket tests: the fixtures' spot pairs, ETH-USDC streaming as
// ETH-USD, and their future, and a perpetual of each gateway asset
var testWebsocketPairs = map[asset.Item]currency.Pairs{
	asset.Spot:                {currency.NewBTCUSD(), testETHUSD, testETHUSDC},
	asset.Futures:             {testFuturesPair},
	asset.USDCMarginedFutures: {testUSDCPerpetual},
	asset.CoinMarginedFutures: {testCoinPerpetual},
}

// newWebsocketTestExchange returns an exchange named after the test, with credentials, the trade feed on and the
// websocket test pairs available and enabled, so its handlers store and relay what the fixtures stream
func newWebsocketTestExchange(t *testing.T) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, setupTestExchange(ex), "setupTestExchange must not error")
	ex.Name = t.Name()
	setTestCredentials(ex, testCredentials(t))
	ex.Features.Enabled.TradeFeed = true
	for a, pairs := range testWebsocketPairs {
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, pairs, false), "StorePairs must not error for available %s pairs", a)
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, pairs, true), "StorePairs must not error for enabled %s pairs", a)
		require.NoErrorf(t, ex.CurrencyPairs.SetAssetEnabled(a, true), "SetAssetEnabled must not error for %s", a)
	}
	return ex
}

// websocketFixture returns a websocket fixture's messages, one a line
func websocketFixture(tb testing.TB, name string) [][]byte {
	tb.Helper()
	f, err := os.Open("testdata/" + name)
	require.NoError(tb, err, "the websocket fixture must open")
	defer f.Close()
	var messages [][]byte
	s := bufio.NewScanner(f)
	s.Buffer(nil, 1<<20)
	for s.Scan() {
		messages = append(messages, bytes.Clone(s.Bytes()))
	}
	require.NoError(tb, s.Err(), "the websocket fixture must read")
	return messages
}

// replayWebsocketFixture handles every message of a fixture on a connection, returning the errors handling them
func replayWebsocketFixture(t *testing.T, ex *Exchange, conn websocket.Connection, name string) []error {
	t.Helper()
	var errs []error
	for _, message := range websocketFixture(t, name) {
		if err := ex.wsHandleData(t.Context(), conn, message); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// relayed drains what the exchange has relayed so far, returning the payloads of one type and leaving out the order
// book depths every book change relays
func relayed[T any](ex *Exchange) []T {
	var got []T
	for {
		select {
		case p := <-ex.Websocket.DataHandler.C:
			if v, ok := p.Data.(T); ok {
				got = append(got, v)
			}
		default:
			return got
		}
	}
}

// mustParseTime parses an RFC 3339 time as the websocket messages send it
func mustParseTime(tb testing.TB, value string) time.Time {
	tb.Helper()
	parsed, err := time.Parse(time.RFC3339Nano, value)
	require.NoError(tb, err, "the time must parse")
	return parsed
}

func TestWebsocketHandleTickers(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn, err := ex.Websocket.CreateTestConnection(websocketMarketDataConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	errs := replayWebsocketFixture(t, ex, conn, "wsPublic.json")
	require.Len(t, errs, 1, "replaying the public fixture must error only for its refused request")
	assert.ErrorIs(t, errs[0], errWebsocketRequestRefused, "a refusal no request awaits should be reported")

	// A batched ticker keeps the best bid and ask of the pair's stored ticker
	exp := map[currency.Pair]ticker.Price{
		currency.NewBTCUSD(): {
			Last: 82376.79, High: 82454.5, Low: 80564.57, BaseVolume: 3073.57853188, PercentChange24Hour: 1.03951611004047,
			Bid: 82376.78, BidSize: 0.08497946, Ask: 82376.79, AskSize: 0.00030548,
			Pair: currency.NewBTCUSD(), AssetType: asset.Spot, LastUpdated: mustParseTime(t, "2026-10-19T01:25:16.611395438Z"),
		},
		testETHUSD: {
			Last: 2419.76, High: 2427.05, Low: 2325.48, BaseVolume: 50262.27255631, PercentChange24Hour: 0.53889205123837,
			Bid: 2420.02, BidSize: 1.22931395, Ask: 2420.1, AskSize: 0.001,
			Pair: testETHUSD, AssetType: asset.Spot, LastUpdated: mustParseTime(t, "2026-10-19T01:25:16.01220171Z"),
		},
		testFuturesPair: {
			Last: 82400, High: 82470, Low: 80620, BaseVolume: 135507, PercentChange24Hour: 1.02991662579696,
			Bid: 82405, BidSize: 100, Ask: 82415, AskSize: 5,
			Pair: testFuturesPair, AssetType: asset.Futures, LastUpdated: mustParseTime(t, "2026-10-19T01:25:19.2223531Z"),
		},
	}
	// ETH-USDC streams as ETH-USD
	eth := exp[testETHUSD]
	eth.Pair = testETHUSDC
	exp[testETHUSDC] = eth
	for p, want := range exp {
		want.ExchangeName = ex.Name
		got, err := ticker.GetTicker(ex.Name, p, want.AssetType)
		require.NoErrorf(t, err, "GetTicker must not error for %s", p)
		assert.Equalf(t, &want, got, "the stored %s ticker should be the last streamed", p)
	}
	tickers := relayed[[]ticker.Price](ex)
	assert.Len(t, tickers, 12, "each ticker message should relay its processed tickers")
}

func TestWebsocketHandleTickerBatchWithoutTicker(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	envelope := &WebsocketEnvelope{
		Channel:   ChannelTickerBatch,
		Timestamp: mustParseTime(t, "2026-10-19T01:25:15.786603825Z"),
		Events:    json.RawMessage(`[{"type":"snapshot","tickers":[{"type":"ticker","product_id":"BTC-USD","price":"82371.43","volume_24_h":"3073.57853188","low_24_h":"80564.57","high_24_h":"82454.5","low_52_w":"57717.55","high_52_w":"116410.06","price_percent_chg_24_h":"1.03294178484098"}]}]`),
	}
	require.NoError(t, ex.wsHandleTicker(t.Context(), envelope), "wsHandleTicker must not error")
	got, err := ticker.GetTicker(ex.Name, currency.NewBTCUSD(), asset.Spot)
	require.NoError(t, err, "GetTicker must not error")
	exp := &ticker.Price{
		Last: 82371.43, High: 82454.5, Low: 80564.57, BaseVolume: 3073.57853188, PercentChange24Hour: 1.03294178484098,
		Pair: currency.NewBTCUSD(), AssetType: asset.Spot, ExchangeName: ex.Name, LastUpdated: envelope.Timestamp,
	}
	assert.Equal(t, exp, got, "a batched ticker without a stored ticker should store no best bid or ask")

	envelope.Events = json.RawMessage(`[{"type":"update","tickers":[{"type":"ticker","product_id":"NOPE-USD","price":"1"}]}]`)
	assert.ErrorIs(t, ex.wsHandleTicker(t.Context(), envelope), errWebsocketUnknownProduct, "wsHandleTicker should error for an unknown product")
	assert.Empty(t, relayed[[]ticker.Price](ex)[1:], "wsHandleTicker should relay nothing when no ticker is processed")
	envelope.Events = json.RawMessage(`[{"tickers":{}}]`)
	assert.Error(t, ex.wsHandleTicker(t.Context(), envelope), "wsHandleTicker should error for malformed events")
}

func TestWebsocketHandleLevel2(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn, err := ex.Websocket.CreateTestConnection(websocketMarketDataConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	require.Len(t, replayWebsocketFixture(t, ex, conn, "wsPublic.json"), 1, "replaying the public fixture must error only for its refused request")

	type book struct {
		bids, asks  orderbook.Levels
		lastUpdated time.Time
	}
	eth := book{
		bids:        orderbook.Levels{{Price: 2420.09, Amount: 2.77133332}, {Price: 2420.08, Amount: 1}, {Price: 2420.07, Amount: 1.87040069}, {Price: 2420.06, Amount: 1}, {Price: 2420.01, Amount: 0.4298574}, {Price: 2419.55, Amount: 3.44595938}, {Price: 2419.54, Amount: 3.75436}},
		asks:        orderbook.Levels{{Price: 2420.1, Amount: 0.00122657}, {Price: 2420.17, Amount: 0.00000035}, {Price: 2420.37, Amount: 0.00000035}, {Price: 2420.39, Amount: 1.11060355}, {Price: 2420.4, Amount: 1.86030745}, {Price: 2420.67, Amount: 0.00000035}},
		lastUpdated: mustParseTime(t, "2026-10-19T01:25:16.89160381Z"),
	}
	exp := map[asset.Item]map[currency.Pair]book{
		asset.Spot: {
			currency.NewBTCUSD(): {
				bids:        orderbook.Levels{{Price: 82385, Amount: 0.0183}, {Price: 82384.24, Amount: 0.03}, {Price: 82384.22, Amount: 0.03}, {Price: 82383.89, Amount: 0.00017467}, {Price: 82380.41, Amount: 0.04349756}, {Price: 82367.01, Amount: 0.57223519}, {Price: 82367, Amount: 0.7}, {Price: 82363.33, Amount: 0.00121229}, {Price: 82363.02, Amount: 0.2837802}, {Price: 82360.84, Amount: 0.04906645}, {Price: 82140.53, Amount: 0.00583}, {Price: 81711.18, Amount: 0.00583}},
				asks:        orderbook.Levels{{Price: 82385.01, Amount: 0.01820742}, {Price: 82385.86, Amount: 0.03919461}, {Price: 82385.87, Amount: 0.03919461}, {Price: 82386.59, Amount: 0.024}, {Price: 82389.77, Amount: 0.07286023}, {Price: 82391.36, Amount: 0.05677741}, {Price: 82394.59, Amount: 0.0062491}, {Price: 82394.6, Amount: 0.02}, {Price: 82429.7, Amount: 0.25164455}, {Price: 83912.33, Amount: 0.00583}},
				lastUpdated: mustParseTime(t, "2026-10-19T01:25:16.786979223Z"),
			},
			testETHUSD:  eth,
			testETHUSDC: eth,
		},
		asset.Futures: {
			testFuturesPair: {
				bids:        orderbook.Levels{{Price: 82410, Amount: 1}, {Price: 82405, Amount: 130}, {Price: 82400, Amount: 217}, {Price: 82395, Amount: 210}, {Price: 82390, Amount: 421}, {Price: 82385, Amount: 375}},
				asks:        orderbook.Levels{{Price: 82415, Amount: 4}, {Price: 82420, Amount: 13}, {Price: 82425, Amount: 199}, {Price: 82430, Amount: 205}, {Price: 82435, Amount: 247}, {Price: 82445, Amount: 264}, {Price: 82465, Amount: 524}, {Price: 82470, Amount: 930}},
				lastUpdated: mustParseTime(t, "2026-10-19T01:25:17.017894336Z"),
			},
		},
	}
	for a, books := range exp {
		for p, want := range books {
			got, err := ex.Websocket.Orderbook.GetOrderbook(p, a)
			require.NoErrorf(t, err, "GetOrderbook must not error for %s %s", a, p)
			assert.Equalf(t, want.bids, got.Bids, "the %s %s book's bids should be its snapshot's with its updates applied", a, p)
			assert.Equalf(t, want.asks, got.Asks, "the %s %s book's asks should be its snapshot's with its updates applied", a, p)
			assert.Equalf(t, want.lastUpdated, got.LastUpdated, "the %s %s book should be updated as its last message was sent", a, p)
		}
	}
}

func TestWebsocketHandleLevel2Errors(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn, err := ex.Websocket.CreateTestConnection(websocketMarketDataConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	envelope := &WebsocketEnvelope{Channel: wsChannelLevel2Data, Timestamp: mustParseTime(t, "2026-10-19T01:25:16.765522905Z")}

	envelope.Events = json.RawMessage(`[{"type":"snapshot","product_id":"BTC-USD","updates":[{"side":"ask","price_level":"1","new_quantity":"1"}]}]`)
	assert.ErrorIs(t, ex.wsHandleLevel2(t.Context(), conn, envelope), errLevel2SideUnknown, "wsHandleLevel2 should error for an unknown side")
	envelope.Events = json.RawMessage(`[{"type":"replace","product_id":"BTC-USD","updates":[]}]`)
	assert.ErrorIs(t, ex.wsHandleLevel2(t.Context(), conn, envelope), errWebsocketEventTypeUnknown, "wsHandleLevel2 should error for an unknown event type")
	envelope.Events = json.RawMessage(`[{"type":"update","product_id":"NOPE-USD","updates":[]}]`)
	assert.ErrorIs(t, ex.wsHandleLevel2(t.Context(), conn, envelope), errWebsocketUnknownProduct, "wsHandleLevel2 should error for an unknown product")
	envelope.Events = json.RawMessage(`{}`)
	assert.Error(t, ex.wsHandleLevel2(t.Context(), conn, envelope), "wsHandleLevel2 should error for malformed events")

	// An update to a book that was never loaded is reported when its connection carries no subscription to resubscribe
	envelope.Events = json.RawMessage(`[{"type":"update","product_id":"BTC-USD","updates":[{"side":"bid","price_level":"1","new_quantity":"1"}]}]`)
	assert.ErrorIs(t, ex.wsHandleLevel2(t.Context(), conn, envelope), orderbook.ErrDepthNotFound, "wsHandleLevel2 should report an update to a book never loaded")

	// A subscription already resubscribing awaits its snapshot, so its updates are dropped
	s := &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: wsChannelLevel2, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSD().Format(productIDFormat)}}
	require.NoError(t, s.SetState(subscription.ResubscribingState), "SetState must not error")
	require.NoError(t, conn.Subscriptions().Add(s), "Add must not error")
	assert.NoError(t, ex.wsHandleLevel2(t.Context(), conn, envelope), "wsHandleLevel2 should drop an update to a book awaiting its new snapshot")
}

func TestWebsocketHandleCandles(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	envelope := &WebsocketEnvelope{
		Channel: wsChannelCandles,
		Events:  json.RawMessage(`[{"type":"snapshot","candles":[{"start":"1792330500","high":"81393.38","low":"81208.6","open":"81216.12","close":"81328.68","volume":"8.95974801","product_id":"BTC-USD"},{"start":"1792335600","high":"2381.42","low":"2373.95","open":"2373.95","close":"2379.95","volume":"121.9164883","product_id":"ETH-USD"},{"start":"1792330800","high":"81387.6","low":"81296.89","open":"81328.68","close":"81337.68","volume":"15.05090546","product_id":"BTC-USD"}]}]`),
	}
	require.NoError(t, ex.wsHandleCandles(t.Context(), envelope), "wsHandleCandles must not error")
	btc := []kline.Candle{
		{Time: time.Unix(1792330500, 0), Open: 81216.12, High: 81393.38, Low: 81208.6, Close: 81328.68, Volume: 8.95974801},
		{Time: time.Unix(1792330800, 0), Open: 81328.68, High: 81387.6, Low: 81296.89, Close: 81337.68, Volume: 15.05090546},
	}
	eth := []kline.Candle{{Time: time.Unix(1792335600, 0), Open: 2373.95, High: 2381.42, Low: 2373.95, Close: 2379.95, Volume: 121.9164883}}
	exp := [][]kline.Item{{
		{Exchange: ex.Name, Pair: currency.NewBTCUSD(), Asset: asset.Spot, Interval: kline.FiveMin, Candles: btc},
		{Exchange: ex.Name, Pair: testETHUSD, Asset: asset.Spot, Interval: kline.FiveMin, Candles: eth},
		{Exchange: ex.Name, Pair: testETHUSDC, Asset: asset.Spot, Interval: kline.FiveMin, Candles: eth},
	}}
	assert.Equal(t, exp, relayed[[]kline.Item](ex), "wsHandleCandles should relay each pair's candles, a USDC alias's from its USD product")

	envelope.Events = json.RawMessage(`[{"type":"update","candles":[{"start":"1792330500","product_id":"NOPE-USD"}]}]`)
	assert.ErrorIs(t, ex.wsHandleCandles(t.Context(), envelope), errWebsocketUnknownProduct, "wsHandleCandles should error for an unknown product")
	assert.Empty(t, relayed[[]kline.Item](ex), "wsHandleCandles should relay nothing without candles")
	envelope.Events = json.RawMessage(`{}`)
	assert.Error(t, ex.wsHandleCandles(t.Context(), envelope), "wsHandleCandles should error for malformed events")
}

func TestWebsocketHandleMarketTrades(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	envelope := &WebsocketEnvelope{
		Channel: wsChannelMarketTrades,
		Events:  json.RawMessage(`[{"type":"update","trades":[{"product_id":"BIP-20DEC30-CDE","trade_id":"131237897380","price":"82395","size":"3","time":"2026-10-19T01:25:22.278527Z","side":"BUY"},{"product_id":"ETH-USD","trade_id":"849228088","price":"2420.44","size":"0.00903827","time":"2026-10-19T01:25:17.706972Z","side":"SELL"}]}]`),
	}
	require.NoError(t, ex.wsHandleMarketTrades(t.Context(), envelope), "wsHandleMarketTrades must not error")
	ethTrade := trade.Data{TID: "849228088", Exchange: ex.Name, CurrencyPair: testETHUSD, AssetType: asset.Spot, Side: order.Buy, Price: 2420.44, Amount: 0.00903827, Timestamp: mustParseTime(t, "2026-10-19T01:25:17.706972Z")}
	ethAliasTrade := ethTrade
	ethAliasTrade.CurrencyPair = testETHUSDC
	exp := [][]trade.Data{{
		{TID: "131237897380", Exchange: ex.Name, CurrencyPair: testFuturesPair, AssetType: asset.Futures, Side: order.Sell, Price: 82395, Amount: 3, Timestamp: mustParseTime(t, "2026-10-19T01:25:22.278527Z")},
		ethTrade,
		ethAliasTrade,
	}}
	assert.Equal(t, exp, relayed[[]trade.Data](ex), "wsHandleMarketTrades should relay each trade with the taker's side, the opposite of the maker's Coinbase sends")

	envelope.Events = json.RawMessage(`[{"type":"update","trades":[{"product_id":"BTC-USD","trade_id":"1","side":"HOLD"},{"product_id":"NOPE-USD","trade_id":"2","side":"BUY"}]}]`)
	err := ex.wsHandleMarketTrades(t.Context(), envelope)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "wsHandleMarketTrades should error for an unknown side")
	assert.ErrorIs(t, err, errWebsocketUnknownProduct, "wsHandleMarketTrades should error for an unknown product")
	envelope.Events = json.RawMessage(`{}`)
	assert.Error(t, ex.wsHandleMarketTrades(t.Context(), envelope), "wsHandleMarketTrades should error for malformed events")

	ex.Features.Enabled.TradeFeed = false
	assert.NoError(t, ex.wsHandleMarketTrades(t.Context(), envelope), "wsHandleMarketTrades should ignore trades when neither the trade feed nor trade saving is on")
}

func TestTakerSide(t *testing.T) {
	t.Parallel()
	for maker, exp := range map[string]order.Side{"BUY": order.Sell, "SELL": order.Buy} {
		side, err := takerSide(maker)
		require.NoErrorf(t, err, "takerSide must not error for %s", maker)
		assert.Equalf(t, exp, side, "takerSide should return the opposite of the maker's %s", maker)
	}
	_, err := takerSide("ANY")
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "takerSide should error for a side with no opposite")
	_, err = takerSide("HOLD")
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "takerSide should error for an unknown side")
}

func TestWebsocketHandleStatus(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	envelope := &WebsocketEnvelope{
		Channel: ChannelStatus,
		Events:  json.RawMessage(`[{"type":"snapshot","products":[{"product_type":"SPOT","id":"BTC-USD","base_currency":"BTC","quote_currency":"USD","base_increment":"0.00000001","quote_increment":"0.01","display_name":"BTC/USD","status":"online","status_message":"","min_market_funds":"1"}]},{"type":"update","products":[]}]`),
	}
	require.NoError(t, ex.wsHandleStatus(t.Context(), envelope), "wsHandleStatus must not error")
	exp := [][]WebsocketProductStatus{{{
		ProductType: ProductTypeSpot, ID: "BTC-USD", BaseCurrency: currency.BTC, QuoteCurrency: currency.USD, BaseIncrement: 0.00000001,
		QuoteIncrement: 0.01, DisplayName: "BTC/USD", Status: "online", MinimumMarketFunds: 1,
	}}}
	assert.Equal(t, exp, relayed[[]WebsocketProductStatus](ex), "wsHandleStatus should relay each event's products, and nothing for an event without any")
	envelope.Events = json.RawMessage(`{}`)
	assert.Error(t, ex.wsHandleStatus(t.Context(), envelope), "wsHandleStatus should error for malformed events")
}

func TestWebsocketHandleAccount(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	// The user channel reports no order's size, so the orders' sizes are recorded as placing them records them
	ex.orderSizes.set("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", 0.001, 0)
	ex.orderSizes.set("bbbbbbbb-cccc-dddd-eeee-ffffffffffff", 1, 0)
	ex.orderSizes.set("cccccccc-dddd-eeee-ffff-000000000000", 2, 0)
	conn, err := ex.Websocket.CreateTestConnection(websocketUserConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	require.Empty(t, replayWebsocketFixture(t, ex, conn, "wsAccount.json"), "replaying the account fixture must not error")
	_, ok := ex.orderSizes.get("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	assert.False(t, ok, "the user channel should forget a closed order's size")

	snapshot := mustParseTime(t, "2026-10-19T01:30:00.2Z")
	exp := [][]order.Detail{
		{
			{
				Exchange: ex.Name, OrderID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", ClientOrderID: "11111111-2222-3333-4444-555555555555",
				AccountID: "ffffffff-0000-1111-2222-333333333333", Pair: currency.NewBTCUSD(), AssetType: asset.Spot, Side: order.Buy,
				Type: order.Limit, Status: order.Open, TimeInForce: order.GoodTillCancel | order.PostOnly, Price: 80000, Amount: 0.001,
				RemainingAmount: 0.001, Date: mustParseTime(t, "2026-10-19T01:20:00.123456Z"), LastUpdated: snapshot,
			},
			{
				Exchange: ex.Name, OrderID: "bbbbbbbb-cccc-dddd-eeee-ffffffffffff", ClientOrderID: "66666666-7777-8888-9999-000000000000",
				AccountID: "ffffffff-0000-1111-2222-333333333333", Pair: testETHUSDC, AssetType: asset.Spot, Side: order.Sell,
				Type: order.Limit, Status: order.PartiallyFilled, TimeInForce: order.GoodTillTime, Price: 2420.1, AverageExecutedPrice: 2420.1,
				Amount: 1, RemainingAmount: 0.6, ExecutedAmount: 0.4, Cost: 968.04, Fee: 2.904, Date: mustParseTime(t, "2026-10-19T01:21:00.5Z"),
				LastUpdated: snapshot,
			},
			{
				Exchange: ex.Name, OrderID: "cccccccc-dddd-eeee-ffff-000000000000", ClientOrderID: "12121212-3434-5656-7878-909090909090",
				AccountID: "ffffffff-0000-1111-2222-333333333333", Pair: testFuturesPair, AssetType: asset.Futures, Side: order.Sell,
				Type: order.StopLimit, Status: order.Open, TimeInForce: order.GoodTillCancel, Price: 83000, TriggerPrice: 82900, Amount: 2,
				RemainingAmount: 2, Date: mustParseTime(t, "2026-10-19T01:22:00Z"), LastUpdated: snapshot,
			},
		},
		{
			{
				Exchange: ex.Name, OrderID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", ClientOrderID: "11111111-2222-3333-4444-555555555555",
				AccountID: "ffffffff-0000-1111-2222-333333333333", Pair: currency.NewBTCUSD(), AssetType: asset.Spot, Side: order.Buy,
				Type: order.Limit, Status: order.Filled, TimeInForce: order.GoodTillCancel | order.PostOnly, Price: 80000, Amount: 0.001,
				AverageExecutedPrice: 80000, ExecutedAmount: 0.001, Cost: 80, Fee: 0.48, Date: mustParseTime(t, "2026-10-19T01:20:00.123456Z"),
				CloseTime: mustParseTime(t, "2026-10-19T01:30:04.9Z"), LastUpdated: mustParseTime(t, "2026-10-19T01:30:05Z"),
			},
		},
	}
	positions := []*WebsocketUserPositions{{
		PerpetualFuturesPositions: []WebsocketPerpetualFuturesPosition{},
		ExpiringFuturesPositions: []WebsocketExpiringFuturesPosition{{
			ProductID: "BIP-20DEC30-CDE", Side: "Long", NumberOfContracts: 3, RealisedPNL: 12.5, UnrealisedPNL: -4.25, EntryPrice: 82390,
		}},
		PredictionMarketPositions: json.RawMessage(`[]`),
	}}
	balances := []*WebsocketFuturesBalanceSummary{{
		FuturesBuyingPower: 1500.25, TotalUSDBalance: 2500.5, CBIUSDBalance: 1000.25, CFMUSDBalance: 1500.25, TotalOpenOrdersHoldAmount: 80.48,
		UnrealisedPNL: -4.25, DailyRealisedPNL: 12.5, InitialMargin: 1200, AvailableMargin: 1300.25, LiquidationThreshold: 600,
		LiquidationBufferAmount: 700.25, LiquidationBufferPercentage: 116.7,
		IntradayMarginWindowMeasure: WebsocketMarginWindowMeasure{
			MarginWindowType: "FCM_MARGIN_WINDOW_TYPE_INTRADAY", MarginLevel: "MARGIN_LEVEL_TYPE_BASE", InitialMargin: 1200, MaintenanceMargin: 600,
			LiquidationBufferPercentage: 116.7, TotalHold: 80.48, FuturesBuyingPower: 1500.25,
		},
		OvernightMarginWindowMeasure: WebsocketMarginWindowMeasure{
			MarginWindowType: "FCM_MARGIN_WINDOW_TYPE_OVERNIGHT", MarginLevel: "MARGIN_LEVEL_TYPE_WARNING", InitialMargin: 2400, MaintenanceMargin: 1200,
			LiquidationBufferPercentage: 8.35, TotalHold: 80.48, FuturesBuyingPower: 300.25,
		},
	}}
	var gotOrders [][]order.Detail
	var gotPositions []*WebsocketUserPositions
	var gotBalances []*WebsocketFuturesBalanceSummary
	for {
		var p any
		select {
		case payload := <-ex.Websocket.DataHandler.C:
			p = payload.Data
		default:
		}
		if p == nil {
			break
		}
		switch v := p.(type) {
		case []order.Detail:
			gotOrders = append(gotOrders, v)
		case *WebsocketUserPositions:
			gotPositions = append(gotPositions, v)
		case *WebsocketFuturesBalanceSummary:
			gotBalances = append(gotBalances, v)
		}
	}
	assert.Equal(t, exp, gotOrders, "the user channel's orders should be relayed as order details")
	assert.Equal(t, positions, gotPositions, "the user channel's positions should be relayed when it has any")
	assert.Equal(t, balances, gotBalances, "the futures balance summary should be relayed")
}

func TestWebsocketLookUpOrderSizes(t *testing.T) {
	t.Parallel()
	const orderID = "dddddddd-eeee-ffff-0000-111111111111"
	var lookups atomic.Int32
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v3/brokerage/orders/historical/batch", r.URL.Path, "the user channel should look orders up in the order list")
		assert.Equal(t, []string{orderID}, r.URL.Query()["order_ids"], "the user channel should look up the orders it reported without their sizes")
		lookups.Add(1)
		_, err := w.Write([]byte(`{"orders":[{"order_id":"` + orderID + `","product_id":"BTC-USD","product_type":"SPOT","side":"SELL","status":"OPEN","order_type":"LIMIT","time_in_force":"GOOD_UNTIL_CANCELLED","filled_size":"0","order_configuration":{"limit_limit_gtc":{"base_size":"0.5","limit_price":"90000","post_only":false}}}],"has_next":false,"cursor":""}`))
		assert.NoError(t, err, "writing the response should not error")
	}))
	update := func(status, leaves string) *WebsocketEnvelope {
		return &WebsocketEnvelope{
			Channel:   wsChannelUser,
			Timestamp: time.Date(2026, 10, 19, 1, 30, 0, 0, time.UTC),
			Events: json.RawMessage(`[{"type":"update","orders":[{"order_id":"` + orderID + `","order_side":"SELL","order_type":"LIMIT","status":"` + status +
				`","product_id":"BTC-USD","product_type":"SPOT","time_in_force":"GOOD_UNTIL_CANCELLED","limit_price":"90000","cumulative_quantity":"0","leaves_quantity":"` + leaves + `"}]}]`),
		}
	}
	// An order placed elsewhere arrives without its size, which is looked up and relayed
	require.NoError(t, ex.wsHandleUser(t.Context(), update(OrderStatusOpen, "0.5")), "wsHandleUser must not error")
	var details [][]order.Detail
	require.Eventually(t, func() bool {
		details = append(details, relayed[[]order.Detail](ex)...)
		return len(details) == 2
	}, 5*time.Second, 10*time.Millisecond, "the user channel must relay the order, then the order with its size")
	assert.Zero(t, details[0][0].Amount, "the user channel should relay the order as reported, without a size")
	assert.Equal(t, 0.5, details[1][0].Amount, "the user channel should relay the order again with the size it looked up")
	assert.True(t, details[1][0].IsActive(), "the order with its size should be active")

	require.NoError(t, ex.wsHandleUser(t.Context(), update(OrderStatusOpen, "0.5")), "wsHandleUser must not error for the order's next update")
	got := relayed[[]order.Detail](ex)
	require.Len(t, got, 1, "the user channel must relay the order's next update")
	assert.Equal(t, 0.5, got[0][0].Amount, "the user channel should give the order's next update its recorded size")
	assert.Equal(t, 0.5, got[0][0].RemainingAmount, "the user channel should report a base sized order's remaining amount")
	require.NoError(t, ex.wsHandleUser(t.Context(), update(OrderStatusCancelled, "0")), "wsHandleUser must not error for the order's cancellation")
	_, ok := ex.orderSizes.get(orderID)
	assert.False(t, ok, "the user channel should forget a closed order's size")
	assert.Equal(t, int32(1), lookups.Load(), "the user channel should look an order up once")
}

// userOrderEnvelope returns a user channel message reporting an order
func userOrderEnvelope(orderID, orderType, status, executed, remaining string) *WebsocketEnvelope {
	return &WebsocketEnvelope{
		Channel:   wsChannelUser,
		Timestamp: time.Date(2026, 10, 19, 1, 30, 0, 0, time.UTC),
		Events: json.RawMessage(`[{"type":"update","orders":[{"order_id":"` + orderID + `","order_side":"SELL","order_type":"` + orderType + `","status":"` + status +
			`","product_id":"BTC-USD","product_type":"SPOT","time_in_force":"GOOD_UNTIL_CANCELLED","limit_price":"90000","cumulative_quantity":"` + executed + `","leaves_quantity":"` + remaining + `"}]}]`),
	}
}

// listedOrder returns an order list holding a limit order of a base size
func listedOrder(orderID, size string) string {
	return `{"orders":[{"order_id":"` + orderID + `","product_id":"BTC-USD","product_type":"SPOT","side":"SELL","status":"OPEN","order_type":"LIMIT","time_in_force":"GOOD_UNTIL_CANCELLED","filled_size":"0","size_in_quote":false,"order_configuration":{"limit_limit_gtc":{"base_size":"` + size + `","limit_price":"90000"}}}],"has_next":false,"cursor":""}`
}

// awaitRelayedOrders returns the order updates relayed, waiting until there are n
func awaitRelayedOrders(t *testing.T, ex *Exchange, n int) [][]order.Detail {
	t.Helper()
	var got [][]order.Detail
	require.Eventuallyf(t, func() bool {
		got = append(got, relayed[[]order.Detail](ex)...)
		return len(got) >= n
	}, 30*time.Second, time.Millisecond, "%d order updates must be relayed", n)
	return got
}

func TestWebsocketHandleUserEditedSize(t *testing.T) {
	t.Parallel()
	const orderID = "dddddddd-eeee-ffff-0000-333333333333"
	for _, tc := range []struct {
		name                         string
		base, quote                  float64
		orderType                    string
		status, executed, remaining  string
		listed                       string
		amount, quoteAmount, settled float64
	}{
		// An edit made elsewhere changes the size the order's quantities give, which a lookup settles
		{name: "edited elsewhere", base: 1, orderType: OrderTypeLimit, status: OrderStatusOpen, executed: "0", remaining: "2", listed: "2", amount: 1, settled: 2},
		{name: "edited elsewhere and partly filled", base: 1, orderType: OrderTypeLimit, status: OrderStatusOpen, executed: "0.5", remaining: "1.5", listed: "2", amount: 1, settled: 2},
		{name: "shrunk elsewhere", base: 2, orderType: OrderTypeStopLimit, status: OrderStatusOpen, executed: "0.25", remaining: "0.75", listed: "1", amount: 2, settled: 1},
		{name: "edited to a size whose quantities' floats round", base: 1, orderType: OrderTypeLimit, status: OrderStatusOpen, executed: "0.1", remaining: "0.2", listed: "0.3", amount: 1, settled: 0.3},
		// A report older than an edit's response gives the size before the edit, which the lookup does not take
		{name: "reported before an edit recorded", base: 2, orderType: OrderTypeLimit, status: OrderStatusOpen, executed: "0", remaining: "1", listed: "2", amount: 2, settled: 2},
		// The quantities' floats sum to 0.30000000000000004, not the size Coinbase holds
		{name: "unedited", base: 0.3, orderType: OrderTypeLimit, status: OrderStatusOpen, executed: "0.1", remaining: "0.2", amount: 0.3},
		// A pending order's remaining quantity can predate its size
		{name: "pending", base: 1, orderType: OrderTypeLimit, status: OrderStatusPending, executed: "0", remaining: "2", amount: 1},
		// An order sized in the quote currency reports what remains of its quote size
		{name: "sized in quote", quote: 100, orderType: OrderTypeLimit, status: OrderStatusOpen, executed: "0.0001", remaining: "20", quoteAmount: 100},
		{name: "sized in quote with a base size", base: 1, quote: 100, orderType: OrderTypeLimit, status: OrderStatusOpen, executed: "0.1", remaining: "90", amount: 1, quoteAmount: 100},
		// An edit cannot resize a TWAP order, whose quantities need not give its size
		{name: "not resizable", base: 1, orderType: OrderTypeTWAP, status: OrderStatusOpen, executed: "0", remaining: "2", amount: 1},
		{name: "nothing remaining", base: 1, orderType: OrderTypeLimit, status: OrderStatusOpen, executed: "1", remaining: "0", amount: 1},
		// A filled order that executed more than its size was enlarged by an edit elsewhere, and one that executed less
		// left dust unfilled
		{name: "filled after an edit elsewhere", base: 1, orderType: OrderTypeLimit, status: OrderStatusFilled, executed: "2", remaining: "0", amount: 2},
		{name: "filled leaving dust", base: 1, orderType: OrderTypeLimit, status: OrderStatusFilled, executed: "0.999999", remaining: "0", amount: 1},
	} {
		var lists atomic.Int32
		ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !assert.Equalf(t, "/api/v3/brokerage/orders/historical/batch", r.URL.Path, "the user channel should only look orders up for the order %s", tc.name) {
				http.NotFound(w, r)
				return
			}
			lists.Add(1)
			_, err := w.Write([]byte(listedOrder(orderID, tc.listed)))
			assert.NoError(t, err, "writing the response should not error")
		}))
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.NewBTCUSD()}, false), "StorePairs must not error for the order %s", tc.name)
		ex.orderSizes.set(orderID, tc.base, tc.quote)
		require.NoErrorf(t, ex.wsHandleUser(t.Context(), userOrderEnvelope(orderID, tc.orderType, tc.status, tc.executed, tc.remaining)), "wsHandleUser must not error for the order %s", tc.name)
		relays := 1
		if tc.listed != "" {
			relays = 2
		}
		got := awaitRelayedOrders(t, ex, relays)
		require.Lenf(t, got[0], 1, "the user channel must relay the order %s", tc.name)
		assert.Equalf(t, tc.amount, got[0][0].Amount, "the user channel should relay the order %s with its recorded size", tc.name)
		assert.Equalf(t, tc.quoteAmount, got[0][0].QuoteAmount, "the user channel should relay the order %s with its quote size", tc.name)
		if tc.listed == "" {
			assert.Lenf(t, got, 1, "the user channel should relay the order %s once", tc.name)
			assert.Zerof(t, lists.Load(), "the user channel should not look up the order %s", tc.name)
			if tc.status != OrderStatusFilled {
				size, ok := ex.orderSizes.get(orderID)
				require.Truef(t, ok, "the user channel must keep the size of the order %s", tc.name)
				assert.Equalf(t, tc.base, size.base, "the user channel should keep the size of the order %s", tc.name)
				assert.Equalf(t, tc.quote, size.quote, "the user channel should keep the quote size of the order %s", tc.name)
			}
			continue
		}
		assert.EqualValuesf(t, 1, lists.Load(), "the user channel should look up the order %s once", tc.name)
		require.Lenf(t, got, 2, "the user channel must relay the order %s again with the size looked up", tc.name)
		require.Lenf(t, got[1], 1, "the lookup must relay the order %s", tc.name)
		assert.Equalf(t, tc.settled, got[1][0].Amount, "the lookup should relay the order %s with its listed size", tc.name)
		size, ok := ex.orderSizes.get(orderID)
		require.Truef(t, ok, "the lookup must record the size of the order %s", tc.name)
		assert.Equalf(t, tc.settled, size.base, "the lookup should record the size of the order %s", tc.name)
		assert.Falsef(t, size.disputed, "the lookup should settle the dispute over the size of the order %s", tc.name)
	}
}

func TestEditOrderReportedBeforeResponse(t *testing.T) {
	t.Parallel()
	const orderID = "dddddddd-eeee-ffff-0000-444444444444"
	var ex *Exchange
	ex = newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body string
		switch r.URL.Path {
		case "/api/v3/brokerage/orders/edit":
			// The user channel reports the edit before Coinbase answers the request
			// The reader's context outlives the request, as the connection's does
			assert.NoError(t, ex.wsHandleUser(t.Context(), userOrderEnvelope(orderID, OrderTypeLimit, OrderStatusOpen, "0", "2")), "wsHandleUser should not error for the edit's report")
			body = `{"success":true,"errors":[]}`
		case "/api/v3/brokerage/orders/historical/batch":
			body = listedOrder(orderID, "2")
		default:
			assert.Failf(t, "EditOrder should edit the order and the user channel look it up", "requested %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "writing the response should not error")
	}))
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.NewBTCUSD()}, false), "StorePairs must not error")
	ex.orderSizes.set(orderID, 1, 0)
	_, err := ex.EditOrder(t.Context(), &EditOrderRequest{OrderID: orderID, Price: 90000, Size: 2})
	require.NoError(t, err, "EditOrder must not error")
	// The edit's report is relayed with the size recorded before it, and the lookup it starts relays the edited size
	got := awaitRelayedOrders(t, ex, 2)
	require.Len(t, got, 2, "the user channel must relay the edit's report and the lookup's")
	assert.Equal(t, 1.0, got[0][0].Amount, "the user channel should relay the edit's report with the size it had recorded")
	last := got[1]
	require.Len(t, last, 1, "the user channel must relay the edited order")
	assert.Equal(t, 2.0, last[0].Amount, "the user channel should relay an edit reported before its response with the edited size")
	assert.Equal(t, 2.0, last[0].RemainingAmount, "the user channel should relay what remains of the edited order")
	size, ok := ex.orderSizes.get(orderID)
	require.True(t, ok, "the edited order's size must be recorded")
	assert.Equal(t, 2.0, size.base, "the edited order's size should be recorded")
}

func TestWebsocketLookUpOrderSizesEditedMeanwhile(t *testing.T) {
	t.Parallel()
	const orderID = "dddddddd-eeee-ffff-0000-555555555555"
	for _, tc := range []struct {
		orderType string
		lists     int32
		settled   float64
	}{
		{orderType: OrderTypeLimit, lists: 2, settled: 2},
		// An edit cannot resize a TWAP order, whose quantities need not give its size, so its list is taken
		{orderType: OrderTypeTWAP, lists: 1, settled: 1},
	} {
		listed, release := make(chan struct{}), make(chan struct{})
		var lists atomic.Int32
		ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			size := "2"
			if lists.Add(1) == 1 {
				close(listed)
				select {
				case <-release:
				case <-r.Context().Done():
					return
				}
				// The first list read the order before its edit
				size = "1"
			}
			_, err := w.Write([]byte(listedOrder(orderID, size)))
			assert.NoError(t, err, "writing the response should not error")
		}))
		require.NoErrorf(t, ex.wsHandleUser(t.Context(), userOrderEnvelope(orderID, tc.orderType, OrderStatusOpen, "0.25", "0.75")), "wsHandleUser must not error for the %s order", tc.orderType)
		<-listed
		// The order is edited elsewhere to 2 while its size is being looked up, then its cancellation is queued
		require.NoErrorf(t, ex.wsHandleUser(t.Context(), userOrderEnvelope(orderID, tc.orderType, OrderStatusOpen, "0.25", "1.75")), "wsHandleUser must not error for the %s order's edit report", tc.orderType)
		require.NoErrorf(t, ex.wsHandleUser(t.Context(), userOrderEnvelope(orderID, tc.orderType, OrderStatusCancelQueued, "0.25", "1.75")), "wsHandleUser must not error for the %s order's queued cancellation report", tc.orderType)
		close(release)
		got := awaitRelayedOrders(t, ex, 4)
		assert.Equalf(t, tc.lists, lists.Load(), "the %s order should be listed as often as its reports dispute the list", tc.orderType)
		require.Lenf(t, got, 4, "the user channel must relay the %s order's three reports, then the lookup's", tc.orderType)
		require.Lenf(t, got[3], 1, "the lookup must relay the %s order", tc.orderType)
		assert.Equalf(t, tc.settled, got[3][0].Amount, "the lookup should relay the %s order's settled size", tc.orderType)
		assert.Equalf(t, tc.settled-0.25, got[3][0].RemainingAmount, "the lookup should relay what remains of the %s order", tc.orderType)
		assert.Equalf(t, order.PendingCancel, got[3][0].Status, "the lookup should relay the channel's latest report of the %s order", tc.orderType)
		size, ok := ex.orderSizes.get(orderID)
		require.Truef(t, ok, "the lookup must record the %s order's size", tc.orderType)
		assert.Equalf(t, tc.settled, size.base, "the lookup should record the %s order's settled size", tc.orderType)
	}
}

func TestWebsocketLookUpOrderSizesOlderList(t *testing.T) {
	t.Parallel()
	const orderID = "dddddddd-eeee-ffff-0000-666666666666"
	listing, looking := make(chan struct{}), make(chan struct{})
	releaseList, releaseLookup := make(chan struct{}), make(chan struct{})
	var lookups atomic.Int32
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		size, wait := "1", releaseList
		switch r.URL.Query().Has("order_ids") {
		case true:
			if lookups.Add(1) != 1 {
				size, wait = "3", nil
				break
			}
			size, wait = "2", releaseLookup
			close(looking)
		default:
			close(listing)
		}
		if wait != nil {
			select {
			case <-wait:
			case <-r.Context().Done():
				return
			}
		}
		_, err := w.Write([]byte(listedOrder(orderID, size)))
		assert.NoError(t, err, "writing the response should not error")
	}))
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.NewBTCUSD()}, false), "StorePairs must not error")
	ex.orderSizes.set(orderID, 1, 0)
	// An unrelated list of the live orders begins before the order is edited elsewhere, and reads it at its old size
	listed := make(chan error, 1)
	go func() {
		_, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
		listed <- err
	}()
	<-listing
	require.NoError(t, ex.wsHandleUser(t.Context(), userOrderEnvelope(orderID, OrderTypeLimit, OrderStatusOpen, "0", "2")), "wsHandleUser must not error for the first edit's report")
	<-looking
	require.NoError(t, ex.wsHandleUser(t.Context(), userOrderEnvelope(orderID, OrderTypeLimit, OrderStatusOpen, "0", "3")), "wsHandleUser must not error for the second edit's report")
	// The older list records the old size while the lookup is under way, and the lookup lists the first edit's
	close(releaseList)
	require.NoError(t, <-listed, "GetActiveOrders must not error")
	close(releaseLookup)
	got := awaitRelayedOrders(t, ex, 3)
	assert.EqualValues(t, 2, lookups.Load(), "the order should be looked up again despite the older list")
	require.Len(t, got, 3, "the user channel must relay the two reports, then the second lookup's")
	require.Len(t, got[2], 1, "the second lookup must relay the order")
	assert.Equal(t, 3.0, got[2][0].Amount, "the second lookup should relay the latest edit's size")
	size, ok := ex.orderSizes.get(orderID)
	require.True(t, ok, "the lookup must record the order's size")
	assert.Equal(t, 3.0, size.base, "the lookup should record the latest edit's size")
	assert.False(t, size.disputed, "the lookup should settle the dispute")
}

func TestOrderStatusLiveness(t *testing.T) {
	t.Parallel()
	for status, exp := range map[order.Status][2]bool{
		order.Open:          {true, false},
		order.Pending:       {true, false},
		order.PendingCancel: {true, false},
		order.Filled:        {false, true},
		order.Cancelled:     {false, true},
		order.Rejected:      {false, true},
		order.UnknownStatus: {false, false},
	} {
		assert.Equalf(t, exp[0], isLiveOrderStatus(status), "isLiveOrderStatus should report whether a %s order is live", status)
		assert.Equalf(t, exp[1], isClosedOrderStatus(status), "isClosedOrderStatus should report whether a %s order is closed", status)
	}
}

func TestAdvancedTradeOrderSizes(t *testing.T) {
	t.Parallel()
	open := func(id string) order.Detail { return order.Detail{OrderID: id, Status: order.Open} }
	var sizes advancedTradeOrderSizes
	claimed, started := sizes.claim([]order.Detail{open("a"), open("b")})
	assert.Equal(t, []string{"a", "b"}, claimed, "claim should claim orders neither recorded nor being looked up")
	claimed, _ = sizes.claim([]order.Detail{open("a")})
	assert.Empty(t, claimed, "claim should leave an order being looked up to its lookup")
	sizes.release([]string{"a", "b"}, started)
	sizes.set("a", 1, 0)
	claimed, started = sizes.claim([]order.Detail{open("a"), open("b")})
	assert.Equal(t, []string{"b"}, claimed, "claim should leave out a recorded order")
	sizes.release([]string{"b"}, started)
	// A report disputing a recorded size has the order looked up
	sizes.dispute("a")
	claimed, started = sizes.claim([]order.Detail{open("a")})
	assert.Equal(t, []string{"a"}, claimed, "claim should claim an order whose recorded size is disputed")
	// A lookup begun since is left by the release of an earlier one
	sizes.release([]string{"a"}, started-1)
	claimed, _ = sizes.claim([]order.Detail{open("a")})
	assert.Empty(t, claimed, "release should leave a lookup begun after the one it ends")
	sizes.release([]string{"a"}, started)
	// A list begun before a report disputed a size leaves the dispute standing
	sizes.set("listed", 1, 0)
	listed := sizes.stamp()
	sizes.dispute("listed")
	sizes.setListed("listed", 1, 0, listed)
	claimed, started = sizes.claim([]order.Detail{open("listed")})
	assert.Equal(t, []string{"listed"}, claimed, "claim should claim an order a list recorded after a report disputed its size")
	sizes.release(claimed, started)
	sizes.remove("listed")
	sizes.dispute("unrecorded")
	_, ok := sizes.get("unrecorded")
	assert.False(t, ok, "dispute should record nothing for an order without a size")
	size, ok := sizes.get("a")
	require.True(t, ok, "get must return a recorded order's size")
	assert.Equal(t, 1.0, size.base, "get should return the recorded size")
	sizes.remove("a")
	_, ok = sizes.get("a")
	assert.False(t, ok, "remove should forget an order's size")
	for i := range orderSizesPruneThreshold {
		sizes.sizes[strconv.Itoa(i)] = advancedTradeOrderSize{base: 1, recorded: time.Now().Add(-2 * orderSizeLifetime)}
	}
	sizes.set("fresh", 1, 0)
	assert.Len(t, sizes.sizes, 1, "set should forget sizes recorded long ago once the record is large")
}

func TestAdvancedTradeOrderSizeLookups(t *testing.T) {
	t.Parallel()
	var sizes advancedTradeOrderSizes
	claim := func(id string) []string {
		claimed, _ := sizes.claim([]order.Detail{{OrderID: id, Status: order.Open}})
		return claimed
	}
	// The channel's latest report of the order is relayed with the size listed, not the list's older state
	require.Equal(t, []string{"partly"}, claim("partly"), "claim must claim the order")
	sizes.observe(&order.Detail{OrderID: "partly", Status: order.PartiallyFilled, ExecutedAmount: 0.2}, 0)
	d, outcome := sizes.complete("partly", 0.5, 0)
	require.Equal(t, lookupRelay, outcome, "complete must relay a live order")
	assert.Equal(t, order.Detail{OrderID: "partly", Status: order.PartiallyFilled, Amount: 0.5, ExecutedAmount: 0.2, RemainingAmount: 0.3}, d, "complete should give the latest report the size listed")
	size, ok := sizes.get("partly")
	require.True(t, ok, "complete must record the size listed")
	assert.Equal(t, 0.5, size.base, "complete should record the size listed")

	// An order the channel reports closed while it is looked up is not relayed again, nor its size kept
	claimed, started := sizes.claim([]order.Detail{{OrderID: "cancelled", Status: order.Open}})
	require.Equal(t, []string{"cancelled"}, claimed, "claim must claim the order")
	sizes.observe(&order.Detail{OrderID: "cancelled", Status: order.Cancelled}, 0)
	_, outcome = sizes.complete("cancelled", 0.5, 0)
	assert.Equal(t, lookupDrop, outcome, "complete should not relay an order the channel reported closed")
	sizes.release([]string{"cancelled"}, started)
	_, ok = sizes.get("cancelled")
	assert.False(t, ok, "complete should not record a closed order's size")

	// A size an edit records while the order is looked up is newer than the list's
	require.Equal(t, []string{"edited"}, claim("edited"), "claim must claim the order")
	sizes.set("edited", 2, 0)
	d, outcome = sizes.complete("edited", 1, 0)
	require.Equal(t, lookupRelay, outcome, "complete must relay a live order")
	assert.Equal(t, 2.0, d.Amount, "complete should give the order the size its edit recorded")
	size, _ = sizes.get("edited")
	assert.Equal(t, 2.0, size.base, "complete should keep the size its edit recorded")
	_, outcome = sizes.complete("unclaimed", 1, 0)
	assert.Equal(t, lookupDrop, outcome, "complete should relay nothing for an order not being looked up")

	// An open report since the lookup began that gives another size than the list's has the order looked up again
	require.Equal(t, []string{"moved"}, claim("moved"), "claim must claim the order")
	sizes.observe(&order.Detail{OrderID: "moved", Status: order.Open}, 2)
	d, outcome = sizes.complete("moved", 1, 0)
	assert.Equal(t, lookupAgain, outcome, "complete should look an order up again when a report since gave another size")
	assert.Equal(t, "moved", d.OrderID, "complete should return the latest report to look up again with")
	_, ok = sizes.get("moved")
	assert.False(t, ok, "complete should record neither size it cannot choose between")
	// A list begun before the lookup that records the order meanwhile leaves it to be looked up again
	relisted := sizes.stamp()
	require.Equal(t, []string{"relisted"}, claim("relisted"), "claim must claim the order")
	sizes.observe(&order.Detail{OrderID: "relisted", Status: order.Open}, 3)
	sizes.setListed("relisted", 1, 0, relisted)
	_, outcome = sizes.complete("relisted", 2, 0)
	require.Equal(t, lookupAgain, outcome, "complete must look an order up again when a report since gave another size")
	assert.Equal(t, []string{"relisted"}, claim("relisted"), "claim should claim the repeat of an order a list recorded meanwhile")
	// An order listed with a quote size as well gives no base size to compare
	require.Equal(t, []string{"mixed"}, claim("mixed"), "claim must claim the order")
	sizes.observe(&order.Detail{OrderID: "mixed", Status: order.Open}, 90.1)
	d, outcome = sizes.complete("mixed", 1, 100)
	require.Equal(t, lookupRelay, outcome, "complete must relay an order listed with a quote size")
	assert.Equal(t, 1.0, d.Amount, "complete should relay the listed base size")
	assert.Equal(t, 100.0, d.QuoteAmount, "complete should relay the listed quote size")
	// A quote sized order's report gives no base size to compare
	require.Equal(t, []string{"quote"}, claim("quote"), "claim must claim the order")
	sizes.observe(&order.Detail{OrderID: "quote", Status: order.Open}, 20.0001)
	d, outcome = sizes.complete("quote", 0, 100)
	require.Equal(t, lookupRelay, outcome, "complete must relay a quote sized order")
	assert.Zero(t, d.Amount, "complete should not take a quote sized order's report as a base size")
	assert.Equal(t, 100.0, d.QuoteAmount, "complete should relay the listed quote size")
}

func TestWebsocketLookUpOrderSizesAfterClose(t *testing.T) {
	t.Parallel()
	const orderID = "eeeeeeee-ffff-0000-1111-222222222222"
	listed, release := make(chan struct{}), make(chan struct{})
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(listed)
		<-release
		_, err := w.Write([]byte(`{"orders":[{"order_id":"` + orderID + `","product_id":"BTC-USD","product_type":"SPOT","side":"SELL","status":"OPEN","order_type":"LIMIT","time_in_force":"GOOD_UNTIL_CANCELLED","filled_size":"0","order_configuration":{"limit_limit_gtc":{"base_size":"0.5","limit_price":"90000"}}}],"has_next":false,"cursor":""}`))
		assert.NoError(t, err, "writing the response should not error")
	}))
	update := func(status string) *WebsocketEnvelope {
		return &WebsocketEnvelope{
			Channel:   wsChannelUser,
			Timestamp: time.Date(2026, 10, 19, 1, 30, 0, 0, time.UTC),
			Events: json.RawMessage(`[{"type":"update","orders":[{"order_id":"` + orderID + `","order_side":"SELL","order_type":"LIMIT","status":"` + status +
				`","product_id":"BTC-USD","product_type":"SPOT","time_in_force":"GOOD_UNTIL_CANCELLED","limit_price":"90000","cumulative_quantity":"0","leaves_quantity":"0.5"}]}]`),
		}
	}
	require.NoError(t, ex.wsHandleUser(t.Context(), update(OrderStatusOpen)), "wsHandleUser must not error")
	<-listed
	// The order is cancelled while its size is being looked up
	require.NoError(t, ex.wsHandleUser(t.Context(), update(OrderStatusCancelled)), "wsHandleUser must not error for the cancellation")
	close(release)
	require.Eventually(t, func() bool {
		ex.orderSizes.m.Lock()
		defer ex.orderSizes.m.Unlock()
		return len(ex.orderSizes.pending) == 0
	}, 5*time.Second, 10*time.Millisecond, "the lookup must finish")
	var statuses []order.Status
	for _, details := range relayed[[]order.Detail](ex) {
		for i := range details {
			statuses = append(statuses, details[i].Status)
		}
	}
	assert.Equal(t, []order.Status{order.Open, order.Cancelled}, statuses, "the lookup should not relay an order the channel reported cancelled meanwhile")
	_, ok := ex.orderSizes.get(orderID)
	assert.False(t, ok, "the lookup should not record a closed order's size")
}

// assertWaitsForLock asserts an operation waits while a lock is held and finishes once it is released
func assertWaitsForLock(t *testing.T, mu *sync.Mutex, operation func(), name string) {
	t.Helper()
	mu.Lock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		operation()
	}()
	finished := func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}
	assert.Neverf(t, finished, 50*time.Millisecond, 5*time.Millisecond, "%s should wait for the lock", name)
	mu.Unlock()
	assert.Eventuallyf(t, finished, 5*time.Second, 5*time.Millisecond, "%s should finish once the lock is released", name)
}

func TestWebsocketUserRelaysInOrder(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	// A sized order, so the channel's report needs no lookup
	ex.orderSizes.set("ffffffff-0000-1111-2222-333333333333", 1, 0)
	envelope := &WebsocketEnvelope{
		Channel:   wsChannelUser,
		Timestamp: time.Date(2026, 10, 19, 1, 30, 0, 0, time.UTC),
		Events: json.RawMessage(`[{"type":"update","orders":[{"order_id":"ffffffff-0000-1111-2222-333333333333","order_side":"BUY","order_type":"LIMIT","status":"OPEN",` +
			`"product_id":"BTC-USD","product_type":"SPOT","time_in_force":"GOOD_UNTIL_CANCELLED","limit_price":"80000","cumulative_quantity":"0","leaves_quantity":"1"}]}]`),
	}
	// The channel's relays and a lookup's are held in order, so a lookup's report cannot follow a newer one
	assertWaitsForLock(t, &ex.orderSizes.relayMu, func() {
		assert.NoError(t, ex.wsHandleUser(t.Context(), envelope), "wsHandleUser should not error")
	}, "wsHandleUser")
	assertWaitsForLock(t, &ex.orderSizes.relayMu, func() {
		ex.relayLookedUpOrders(t.Context(), nil)
	}, "relayLookedUpOrders")
}

func TestProcessMergedTickersHoldsTheStore(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	envelope := &WebsocketEnvelope{
		Channel:   wsChannelTicker,
		Timestamp: time.Now(),
		Events:    json.RawMessage(`[{"type":"update","tickers":[{"type":"ticker","product_id":"BTC-USD","price":"82410","volume_24_h":"100","best_bid":"82409","best_bid_quantity":"1","best_ask":"82411","best_ask_quantity":"2"}]}]`),
	}
	// A merge between another's read of the stored ticker and its store would restore what that one replaced
	assertWaitsForLock(t, &ex.tickerMergeMu, func() {
		assert.NoError(t, ex.wsHandleTicker(t.Context(), envelope), "wsHandleTicker should not error")
	}, "wsHandleTicker")
}

func TestWebsocketTickerKeepsStoredQuoteVolume(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	// A REST ticker stored the quote volume the websocket's ticker leaves out
	_, err := ex.processMergedTickers([]ticker.Price{{ExchangeName: ex.Name, Pair: currency.NewBTCUSD(), AssetType: asset.Spot, Last: 82400, QuoteVolume: 12345, LastUpdated: time.Now()}}, false, false)
	require.NoError(t, err, "processMergedTickers must not error for a REST ticker")
	envelope := &WebsocketEnvelope{
		Channel:   wsChannelTicker,
		Timestamp: time.Now(),
		Events:    json.RawMessage(`[{"type":"update","tickers":[{"type":"ticker","product_id":"BTC-USD","price":"82410","volume_24_h":"100","low_24_h":"80000","high_24_h":"83000","price_percent_chg_24_h":"1","best_bid":"82409","best_bid_quantity":"1","best_ask":"82411","best_ask_quantity":"2"}]}]`),
	}
	require.NoError(t, ex.wsHandleTicker(t.Context(), envelope), "wsHandleTicker must not error")
	tick, err := ticker.GetTicker(ex.Name, currency.NewBTCUSD(), asset.Spot)
	require.NoError(t, err, "GetTicker must not error")
	assert.Equal(t, 82410.0, tick.Last, "wsHandleTicker should store the websocket's last price")
	assert.Equal(t, 82409.0, tick.Bid, "wsHandleTicker should store the websocket's best bid")
	assert.Equal(t, 12345.0, tick.QuoteVolume, "wsHandleTicker should keep the stored quote volume the websocket's ticker leaves out")
}

func TestProcessMergedTickersSourceAge(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	now := time.Now()
	price := func(at time.Time, bid, ask, quoteVolume float64) []ticker.Price {
		return []ticker.Price{{ExchangeName: ex.Name, Pair: currency.NewBTCUSD(), AssetType: asset.Spot, Last: 82000, Bid: bid, Ask: ask, QuoteVolume: quoteVolume, LastUpdated: at}}
	}
	stored := func() *ticker.Price {
		tick, err := ticker.GetTicker(ex.Name, currency.NewBTCUSD(), asset.Spot)
		require.NoError(t, err, "GetTicker must not error")
		return tick
	}
	// A REST ticker sends the quote volume and the websocket's tickers leave it out
	_, err := ex.processMergedTickers(price(now.Add(-30*time.Second), 81999, 82001, 123), false, false)
	require.NoError(t, err, "processMergedTickers must not error for a REST ticker")
	_, err = ex.processMergedTickers(price(now.Add(-20*time.Second), 81999, 82001, 0), false, true)
	require.NoError(t, err, "processMergedTickers must not error for a websocket ticker")
	assert.Equal(t, 123.0, stored().QuoteVolume, "a websocket ticker should keep a quote volume sent within the merge window")
	_, err = ex.processMergedTickers(price(now.Add(45*time.Second), 81999, 82001, 0), false, true)
	require.NoError(t, err, "processMergedTickers must not error for a websocket ticker")
	assert.Zero(t, stored().QuoteVolume, "a websocket ticker should not carry on a quote volume last sent outside the merge window")

	// The ticker channel sends the best bid and ask and the batched channel leaves them out
	_, err = ex.processMergedTickers(price(now, 81998, 82002, 0), true, true)
	require.NoError(t, err, "processMergedTickers must not error for a ticker carrying its book")
	_, err = ex.processMergedTickers(price(now.Add(5*time.Second), 0, 0, 0), true, true)
	require.NoError(t, err, "processMergedTickers must not error for a batched ticker")
	assert.Equal(t, 81998.0, stored().Bid, "a batched ticker should keep a best bid sent within the merge window")
	_, err = ex.processMergedTickers(price(now.Add(90*time.Second), 0, 0, 0), true, true)
	require.NoError(t, err, "processMergedTickers must not error for a batched ticker")
	assert.Zero(t, stored().Bid, "a batched ticker should not carry on a best bid last sent outside the merge window")
	assert.Zero(t, stored().Ask, "a batched ticker should not carry on a best ask last sent outside the merge window")
}

func TestProcessMergedTickersRefusedTicker(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	now := time.Now()
	price := ticker.Price{ExchangeName: ex.Name, Pair: currency.NewBTCUSD(), AssetType: asset.Spot, LastUpdated: now.Add(-2 * time.Minute), Last: 10, Bid: 9, Ask: 11, QuoteVolume: 100}
	_, err := ex.processMergedTickers([]ticker.Price{price}, false, false)
	require.NoError(t, err, "processMergedTickers must store the older ticker")
	// The store refuses a crossed book, so its source has sent nothing since the older ticker
	price.LastUpdated, price.Bid, price.Ask = now, 20, 10
	_, err = ex.processMergedTickers([]ticker.Price{price}, false, false)
	require.Error(t, err, "processMergedTickers must report the crossed book the store refuses")
	price.LastUpdated, price.Bid, price.Ask, price.QuoteVolume = now.Add(time.Second), 0, 0, 0
	got, err := ex.processMergedTickers([]ticker.Price{price}, true, true)
	require.NoError(t, err, "processMergedTickers must store the batched ticker")
	require.Len(t, got, 1, "processMergedTickers must return the ticker it stored")
	assert.Zero(t, got[0].Bid, "a refused ticker should not make a best bid sent long ago mergeable")
	assert.Zero(t, got[0].Ask, "a refused ticker should not make a best ask sent long ago mergeable")
	assert.Zero(t, got[0].QuoteVolume, "a refused ticker should not make a quote volume sent long ago mergeable")
}

func TestWebsocketOrderDetailErrors(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	envelope := &WebsocketEnvelope{Timestamp: time.Now()}
	valid := WebsocketUserOrder{OrderID: "1", ProductID: "BTC-USD", ProductType: ProductTypeSpot, OrderSide: "BUY", OrderType: "LIMIT", Status: "OPEN", TimeInForce: "GOOD_UNTIL_CANCELLED"}
	for name, tc := range map[string]struct {
		change func(*WebsocketUserOrder)
		err    error
	}{
		"product type":  {func(o *WebsocketUserOrder) { o.ProductType = "UNKNOWN_PRODUCT_TYPE" }, errUnknownProductType},
		"product":       {func(o *WebsocketUserOrder) { o.ProductID = "X" }, currency.ErrCreatingPair},
		"side":          {func(o *WebsocketUserOrder) { o.OrderSide = "HOLD" }, order.ErrSideIsInvalid},
		"type":          {func(o *WebsocketUserOrder) { o.OrderType = "ICEBERG" }, order.ErrUnrecognisedOrderType},
		"status":        {func(o *WebsocketUserOrder) { o.Status = "ARCHIVED" }, order.ErrUnsupportedStatusType},
		"time in force": {func(o *WebsocketUserOrder) { o.TimeInForce = "GOOD_UNTIL_LUNCH" }, errUnknownTimeInForce},
	} {
		o := valid
		tc.change(&o)
		_, err := ex.websocketOrderDetail(&o, envelope)
		assert.ErrorIsf(t, err, tc.err, "websocketOrderDetail should error for an unknown %s", name)
	}
	unmatched := valid
	unmatched.ProductID = "SOL-EUR"
	detail, err := ex.websocketOrderDetail(&unmatched, envelope)
	require.NoError(t, err, "websocketOrderDetail must not error for a product outside the available pairs")
	assert.Equal(t, currency.NewPairWithDelimiter("SOL", "EUR", currency.DashDelimiter), detail.Pair, "websocketOrderDetail should name a pair outside the available pairs by its product ID")

	envelope.Events = json.RawMessage(`[{"type":"update","orders":[{"order_id":"1","product_type":"OPTION"}]}]`)
	assert.ErrorIs(t, ex.wsHandleUser(t.Context(), envelope), errUnknownProductType, "wsHandleUser should report an order it cannot convert")
	envelope.Events = json.RawMessage(`{}`)
	assert.Error(t, ex.wsHandleUser(t.Context(), envelope), "wsHandleUser should error for malformed events")
	assert.Error(t, ex.wsHandleFuturesBalanceSummary(t.Context(), envelope), "wsHandleFuturesBalanceSummary should error for malformed events")
}

func TestProductTypeAsset(t *testing.T) {
	t.Parallel()
	for productType, exp := range map[string]asset.Item{ProductTypeSpot: asset.Spot, ProductTypeFuture: asset.Futures} {
		a, err := productTypeAsset(productType)
		require.NoErrorf(t, err, "productTypeAsset must not error for %s", productType)
		assert.Equalf(t, exp, a, "productTypeAsset should return the asset of %s", productType)
	}
	_, err := productTypeAsset(ProductTypeEquity)
	assert.ErrorIs(t, err, errUnknownProductType, "productTypeAsset should error for a product type without an asset")
}

func TestWebsocketHandleUserLatestReportInMessage(t *testing.T) {
	t.Parallel()
	const orderID = "dddddddd-eeee-ffff-0000-222222222222"
	report := func(status, leaves, filled string) string {
		return `{"order_id":"` + orderID + `","order_side":"SELL","order_type":"LIMIT","status":"` + status + `","product_id":"BTC-USD","product_type":"SPOT","time_in_force":"GOOD_UNTIL_CANCELLED","limit_price":"90000","cumulative_quantity":"` + filled + `","leaves_quantity":"` + leaves + `"}`
	}
	message := func(first, second string) *WebsocketEnvelope {
		return &WebsocketEnvelope{Channel: wsChannelUser, Timestamp: time.Now(), Events: json.RawMessage(`[{"type":"update","orders":[` + first + `,` + second + `]}]`)}
	}
	listed := func(t *testing.T) *Exchange {
		t.Helper()
		ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write([]byte(`{"orders":[{"order_id":"` + orderID + `","product_id":"BTC-USD","product_type":"SPOT","side":"SELL","status":"OPEN","order_type":"LIMIT","time_in_force":"GOOD_UNTIL_CANCELLED","filled_size":"0","order_configuration":{"limit_limit_gtc":{"base_size":"1","limit_price":"90000"}}}],"has_next":false,"cursor":""}`))
			assert.NoError(t, err, "writing the response should not error")
		}))
		return ex
	}
	lookedUp := func(t *testing.T, ex *Exchange) {
		t.Helper()
		require.Eventually(t, func() bool {
			ex.orderSizes.m.Lock()
			defer ex.orderSizes.m.Unlock()
			return len(ex.orderSizes.pending) == 0
		}, 5*time.Second, time.Millisecond, "the lookup must end")
	}

	ex := listed(t)
	require.NoError(t, ex.wsHandleUser(t.Context(), message(report(OrderStatusOpen, "1", "0"), report(OrderStatusCancelled, "0", "0"))), "wsHandleUser must not error")
	lookedUp(t, ex)
	var statuses []order.Status
	for _, details := range relayed[[]order.Detail](ex) {
		for i := range details {
			statuses = append(statuses, details[i].Status)
		}
	}
	assert.Equal(t, []order.Status{order.Open, order.Cancelled}, statuses, "an order a message closes should not be relayed open again")
	_, ok := ex.orderSizes.get(orderID)
	assert.False(t, ok, "an order a message closes should not have its size recorded")

	ex = listed(t)
	require.NoError(t, ex.wsHandleUser(t.Context(), message(report(OrderStatusOpen, "1", "0"), report(OrderStatusOpen, "0.5", "0.5"))), "wsHandleUser must not error")
	lookedUp(t, ex)
	got := relayed[[]order.Detail](ex)
	require.Len(t, got, 2, "the message and the lookup must each relay once")
	require.Len(t, got[1], 1, "the lookup must relay the order")
	assert.Equal(t, 0.5, got[1][0].ExecutedAmount, "the lookup should relay the message's latest fill")
	assert.Equal(t, 1.0, got[1][0].Amount, "the lookup should relay the size listed")
}

func TestStandardOrderType(t *testing.T) {
	t.Parallel()
	for orderType, exp := range map[string]order.Type{
		"LIMIT": order.Limit, "MARKET": order.Market, "STOP_LIMIT": order.StopLimit, "STOP": order.Stop, "BRACKET": order.Bracket, "TWAP": order.TWAP,
		"LIQUIDATION": order.Liquidation, "TAKE_PROFIT_STOP_LOSS": order.OCO, "": order.UnknownType, "UNKNOWN_ORDER_TYPE": order.UnknownType,
		"ROLL_OPEN": order.UnknownType, "ROLL_CLOSE": order.UnknownType, "SCALED": order.UnknownType,
	} {
		got, err := standardOrderType(orderType)
		require.NoErrorf(t, err, "standardOrderType must not error for %s", orderType)
		assert.Equalf(t, exp, got, "standardOrderType should convert %s", orderType)
	}
	_, err := standardOrderType("Limit")
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "standardOrderType should error for an unknown order type")
}

func TestStandardOrderStatus(t *testing.T) {
	t.Parallel()
	for status, exp := range map[string]order.Status{
		"PENDING": order.Pending, "QUEUED": order.Pending, "OPEN": order.Open, "EDIT_QUEUED": order.Open, "FILLED": order.Filled,
		"CANCEL_QUEUED": order.PendingCancel, "CANCELLED": order.Cancelled, "EXPIRED": order.Expired, "FAILED": order.Rejected,
		"UNKNOWN_ORDER_STATUS": order.UnknownStatus,
	} {
		got, err := standardOrderStatus(status, 0)
		require.NoErrorf(t, err, "standardOrderStatus must not error for %s", status)
		assert.Equalf(t, exp, got, "standardOrderStatus should convert %s", status)
	}
	for status, exp := range map[string]order.Status{"OPEN": order.PartiallyFilled, "EDIT_QUEUED": order.PartiallyFilled, "CANCELLED": order.PartiallyFilledCancelled} {
		got, err := standardOrderStatus(status, 0.1)
		require.NoErrorf(t, err, "standardOrderStatus must not error for a %s order that has started filling", status)
		assert.Equalf(t, exp, got, "standardOrderStatus should convert a %s order that has started filling", status)
	}
	_, err := standardOrderStatus("SLEEPING", 0)
	assert.ErrorIs(t, err, order.ErrUnsupportedStatusType, "standardOrderStatus should error for an unknown status")
}

func TestStandardTimeInForce(t *testing.T) {
	t.Parallel()
	for timeInForce, exp := range map[string]order.TimeInForce{
		"GOOD_UNTIL_CANCELLED": order.GoodTillCancel, "GOOD_UNTIL_DATE_TIME": order.GoodTillTime, "IMMEDIATE_OR_CANCEL": order.ImmediateOrCancel,
		"FILL_OR_KILL": order.FillOrKill, "": order.UnknownTIF, "UNKNOWN_TIME_IN_FORCE": order.UnknownTIF,
	} {
		got, err := standardTimeInForce(timeInForce, false)
		require.NoErrorf(t, err, "standardTimeInForce must not error for %q", timeInForce)
		assert.Equalf(t, exp, got, "standardTimeInForce should convert %q", timeInForce)
	}
	got, err := standardTimeInForce("GOOD_UNTIL_CANCELLED", true)
	require.NoError(t, err, "standardTimeInForce must not error for a post only order")
	assert.Equal(t, order.GoodTillCancel|order.PostOnly, got, "standardTimeInForce should add post only")
	_, err = standardTimeInForce("GOOD_UNTIL_LUNCH", false)
	assert.ErrorIs(t, err, errUnknownTimeInForce, "standardTimeInForce should error for an unknown time in force")
}

func TestWebsocketHandleData(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn, err := ex.Websocket.CreateTestConnection(websocketMarketDataConnection)
	require.NoError(t, err, "CreateTestConnection must not error")

	assert.Error(t, ex.wsHandleData(t.Context(), conn, []byte(`[`)), "wsHandleData should error for malformed data")
	refusal := []byte(`{"type":"error","message":"rate limit exceeded"}`)
	assert.ErrorIs(t, ex.wsHandleData(t.Context(), conn, refusal), errWebsocketRequestRefused, "wsHandleData should report a refusal no request awaits")
	responses, err := conn.MatchReturnResponses(t.Context(), wsChannelSubscriptions, 1)
	require.NoError(t, err, "MatchReturnResponses must not error")
	assert.NoError(t, ex.wsHandleData(t.Context(), conn, refusal), "wsHandleData should pass a refusal to the request awaiting it")
	assert.Equal(t, refusal, (<-responses).Responses[0], "the refusal should answer the request awaiting it")

	assert.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"channel":"subscriptions","sequence_num":0,"events":[{"subscriptions":{}}]}`)), "wsHandleData should accept a confirmation no request awaits")
	assert.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"channel":"heartbeats","sequence_num":1,"events":[]}`)), "wsHandleData should accept a heartbeat")
	require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"channel":"nonsense","sequence_num":2,"events":[]}`)), "wsHandleData must relay an unknown channel's message")
	assert.Len(t, relayed[websocket.UnhandledMessageWarning](ex), 1, "wsHandleData should relay an unknown channel's message as unhandled")

	err = ex.wsHandleData(t.Context(), conn, []byte(`{"channel":"heartbeats","sequence_num":4,"events":[]}`))
	assert.ErrorIs(t, err, errWebsocketSequenceGap, "wsHandleData should report a message following a gap")
	err = ex.wsHandleData(t.Context(), conn, []byte(`{"channel":"ticker","sequence_num":3,"events":[{"type":"update","tickers":[{"product_id":"BTC-USD","price":"1"}]}]}`))
	assert.ErrorIs(t, err, errWebsocketStaleMessage, "wsHandleData should drop a message older than the last")
	_, err = ticker.GetTicker(ex.Name, currency.NewBTCUSD(), asset.Spot)
	assert.Error(t, err, "a stale message's ticker should not be stored")
}

func TestWebsocketConnectionStateSequence(t *testing.T) {
	t.Parallel()
	var state websocketConnectionState
	for _, tc := range []struct {
		sequence   uint64
		gap, stale bool
	}{
		{sequence: 5},
		{sequence: 6},
		{sequence: 8, gap: true},
		{sequence: 8, stale: true},
		{sequence: 7, stale: true},
		{sequence: 9},
	} {
		gap, stale := state.sequence(tc.sequence)
		assert.Equalf(t, tc.gap, gap, "sequence should report a gap before %d correctly", tc.sequence)
		assert.Equalf(t, tc.stale, stale, "sequence should report %d stale correctly", tc.sequence)
	}
}

func TestWebsocketConnections(t *testing.T) {
	t.Parallel()
	var connections websocketConnections
	open, closed := &stateConnection{connected: true}, &stateConnection{}
	state := connections.get(open)
	assert.Same(t, state, connections.get(open), "get should return a connection's state each time")
	connections.get(closed)
	connections.pruneClosed()
	assert.Same(t, state, connections.get(open), "pruneClosed should keep an open connection's state")
	assert.Len(t, connections.states, 1, "pruneClosed should forget a closed connection's state")
}

// stateConnection is a connection reporting whether it is connected
type stateConnection struct {
	websocket.Connection
	connected bool
}

// IsConnected reports whether the connection is connected
func (c *stateConnection) IsConnected() bool { return c.connected }

func TestWebsocketStreamProduct(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	usdc := func(base currency.Code) currency.Pair {
		return currency.NewPairWithDelimiter(base.String(), currency.USDC.String(), currency.DashDelimiter)
	}
	// The documented rule names the aliases until product data loads them
	assert.Equal(t, "ETH-USD", ex.websocketStreamProduct(asset.Spot, usdc(currency.ETH)), "a USDC pair should stream as its USD pair")
	for _, base := range []currency.Code{currency.USDT, currency.EURC} {
		assert.Equalf(t, base.String()+"-USDC", ex.websocketStreamProduct(asset.Spot, usdc(base)), "%s-USDC should stream as itself", base)
	}
	assert.Equal(t, "BIP-20DEC30-CDE", ex.websocketStreamProduct(asset.Futures, testFuturesPair), "a future should stream as itself")

	tgbp := currency.NewCode("TGBP")
	ex.pairAliases.Load(map[currency.Pair]currency.Pairs{
		currency.NewPairWithDelimiter("ETH", "USD", currency.DashDelimiter): {usdc(currency.ETH)},
		usdc(tgbp): {usdc(tgbp)},
	})
	assert.Equal(t, "ETH-USD", ex.websocketStreamProduct(asset.Spot, usdc(currency.ETH)), "a USDC alias should stream as the product data's USD pair")
	assert.Equal(t, "TGBP-USDC", ex.websocketStreamProduct(asset.Spot, usdc(tgbp)), "a USDC pair the product data does not alias should stream as itself")
}

func TestWebsocketPairs(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	a, pairs, err := ex.websocketPairs("ETH-USD")
	require.NoError(t, err, "websocketPairs must not error")
	assert.Equal(t, asset.Spot, a, "websocketPairs should return a spot product's asset")
	assert.Equal(t, currency.Pairs{testETHUSD, testETHUSDC}, pairs, "websocketPairs should return the product's pair and its enabled USDC alias")
	a, pairs, err = ex.websocketPairs("BIP-20DEC30-CDE")
	require.NoError(t, err, "websocketPairs must not error for a future")
	assert.Equal(t, asset.Futures, a, "websocketPairs should return a future's asset")
	assert.Equal(t, currency.Pairs{testFuturesPair}, pairs, "websocketPairs should return a future's pair")
	_, _, err = ex.websocketPairs("NOPE-USD")
	assert.ErrorIs(t, err, errWebsocketUnknownProduct, "websocketPairs should error for an unknown product")

	ex.pairAliases.Load(map[currency.Pair]currency.Pairs{currency.NewBTCUSD(): {currency.NewBTCUSD(), currency.NewPair(currency.BTC, currency.USDC)}})
	_, pairs, err = ex.websocketPairs("BTC-USD")
	require.NoError(t, err, "websocketPairs must not error with aliases loaded")
	assert.Equal(t, currency.Pairs{currency.NewBTCUSD()}, pairs, "websocketPairs should leave out an alias that is not enabled")
}

func TestLevel2Levels(t *testing.T) {
	t.Parallel()
	bids, asks, err := level2Levels([]WebsocketLevel2Update{
		{Side: wsLevel2SideBid, PriceLevel: 2, NewQuantity: 3},
		{Side: wsLevel2SideOffer, PriceLevel: 4, NewQuantity: 0},
	})
	require.NoError(t, err, "level2Levels must not error")
	assert.Equal(t, orderbook.Levels{{Price: 2, Amount: 3}}, bids, "level2Levels should return the bids")
	assert.Equal(t, orderbook.Levels{{Price: 4}}, asks, "level2Levels should return the offers, a zero quantity removing a level")
	_, _, err = level2Levels([]WebsocketLevel2Update{{Side: "ask"}})
	assert.ErrorIs(t, err, errLevel2SideUnknown, "level2Levels should error for an unknown side")
}

func TestWebsocketInvalidateBooks(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	for _, p := range []currency.Pair{testETHUSD, testETHUSDC} {
		require.NoErrorf(t, ex.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{
			Exchange: ex.Name, Pair: p, Asset: asset.Spot, Bids: orderbook.Levels{{Price: 1, Amount: 1}}, Asks: orderbook.Levels{{Price: 2, Amount: 1}}, LastUpdated: time.Now(),
		}), "LoadSnapshot must not error for %s", p)
	}
	ex.wsInvalidateBooks(&subscription.Subscription{Asset: asset.Spot, Pairs: currency.Pairs{testETHUSDC}})
	for _, p := range []currency.Pair{testETHUSD, testETHUSDC} {
		_, err := ex.Websocket.Orderbook.GetOrderbook(p, asset.Spot)
		assert.ErrorIsf(t, err, orderbook.ErrOrderbookInvalid, "wsInvalidateBooks should invalidate the %s book its subscription streams", p)
	}
	ex.wsInvalidateBooks(&subscription.Subscription{Asset: asset.Spot})
	ex.wsInvalidateBooks(&subscription.Subscription{Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPair(currency.XRP, currency.USD)}})
}

// testGenerateSubscription returns the subscription a generator expands from a configured one, for comparison
func testGenerateSubscription(channel, qualified string, a asset.Item, pairs currency.Pairs, authenticated bool) *subscription.Subscription {
	return &subscription.Subscription{Enabled: true, Channel: channel, QualifiedChannel: qualified, Asset: a, Pairs: pairs, Authenticated: authenticated}
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	ex.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: subscription.HeartbeatChannel},
		{Enabled: true, Asset: asset.All, Channel: subscription.TickerChannel},
		{Enabled: true, Asset: asset.All, Channel: subscription.OrderbookChannel},
		{Enabled: true, Asset: asset.All, Channel: ChannelStatus},
		{Enabled: true, Asset: asset.All, Channel: subscription.MyAccountChannel, Authenticated: true},
		{Enabled: true, Asset: asset.Futures, Channel: ChannelFuturesBalanceSummary, Authenticated: true},
		{Enabled: true, Asset: asset.All, Channel: subscription.MyOrdersChannel, Authenticated: true},
		{Enabled: true, Channel: ChannelDerivativesPlatformState},
		{Enabled: true, Asset: asset.USDCMarginedFutures, Channel: ChannelDerivativesUserPortfolio, Authenticated: true, Params: map[string]any{"currency": "usdc"}},
	}
	spot := func(p currency.Pair) currency.Pairs {
		return currency.Pairs{p.Format(currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter})}
	}
	futures := currency.Pairs{testFuturesPair}
	usdcPerpetual := currency.Pairs{testUSDCPerpetual}
	coinPerpetual := currency.Pairs{testCoinPerpetual}
	exp := map[string]subscription.List{
		websocketMarketDataConnection: {
			// ETH-USDC streams as ETH-USD, which carries both
			testGenerateSubscription(subscription.TickerChannel, wsChannelTicker, asset.Spot, spot(currency.NewBTCUSD()), false),
			testGenerateSubscription(subscription.TickerChannel, wsChannelTicker, asset.Spot, spot(testETHUSD), false),
			testGenerateSubscription(subscription.TickerChannel, wsChannelTicker, asset.Futures, futures, false),
			testGenerateSubscription(subscription.OrderbookChannel, wsChannelLevel2, asset.Spot, spot(currency.NewBTCUSD()), false),
			testGenerateSubscription(subscription.OrderbookChannel, wsChannelLevel2, asset.Spot, spot(testETHUSD), false),
			testGenerateSubscription(subscription.OrderbookChannel, wsChannelLevel2, asset.Futures, futures, false),
			testGenerateSubscription(ChannelStatus, ChannelStatus, asset.Spot, spot(currency.NewBTCUSD()), false),
			testGenerateSubscription(ChannelStatus, ChannelStatus, asset.Spot, spot(testETHUSD), false),
			testGenerateSubscription(ChannelStatus, ChannelStatus, asset.Futures, futures, false),
		},
		websocketUserConnection: {
			testGenerateSubscription(subscription.MyAccountChannel, wsChannelUser, asset.Empty, nil, true),
			testGenerateSubscription(ChannelFuturesBalanceSummary, ChannelFuturesBalanceSummary, asset.Empty, nil, true),
		},
		derivativesStreamsConnection: {
			testGenerateSubscription(subscription.TickerChannel, "ticker.BTC_USDC-PERPETUAL.100ms", asset.USDCMarginedFutures, usdcPerpetual, false),
			testGenerateSubscription(subscription.TickerChannel, "ticker.BTC-PERPETUAL.100ms", asset.CoinMarginedFutures, coinPerpetual, false),
			testGenerateSubscription(subscription.OrderbookChannel, "book.BTC_USDC-PERPETUAL.100ms", asset.USDCMarginedFutures, usdcPerpetual, false),
			testGenerateSubscription(subscription.OrderbookChannel, "book.BTC-PERPETUAL.100ms", asset.CoinMarginedFutures, coinPerpetual, false),
			testGenerateSubscription(ChannelDerivativesPlatformState, ChannelDerivativesPlatformState, asset.Empty, nil, false),
		},
		derivativesPrivateConnection: {
			testGenerateSubscription(subscription.MyOrdersChannel, "user.orders.BTC_USDC-PERPETUAL.100ms", asset.USDCMarginedFutures, usdcPerpetual, true),
			testGenerateSubscription(subscription.MyOrdersChannel, "user.orders.BTC-PERPETUAL.100ms", asset.CoinMarginedFutures, coinPerpetual, true),
			testGenerateSubscription(ChannelDerivativesUserPortfolio, "user.portfolio.USDC", asset.Empty, nil, true),
		},
	}
	for filter, generate := range map[string]func() (subscription.List, error){
		websocketMarketDataConnection: ex.generateMarketDataSubscriptions,
		websocketUserConnection:       ex.generateUserSubscriptions,
		derivativesStreamsConnection:  ex.generateDerivativesStreamsSubscriptions,
		derivativesPrivateConnection:  ex.generateDerivativesPrivateSubscriptions,
	} {
		subs, err := generate()
		require.NoErrorf(t, err, "generating the %s connection's subscriptions must not error", filter)
		for _, s := range subs {
			if s.Channel == ChannelDerivativesUserPortfolio {
				// The parameters qualify the channel, and are compared through it
				s.Params = nil
			}
		}
		assert.ElementsMatchf(t, exp[filter], subs, "the %s connection should carry its channels' subscriptions", filter)
	}

	ex.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.All, Channel: subscription.CandlesChannel, Interval: kline.OneHour}}
	_, err := ex.generateMarketDataSubscriptions()
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "generating subscriptions should error for Advanced Trade candles other than five minutes")
}

func TestSubscriptionTemplateFunctions(t *testing.T) {
	t.Parallel()
	tickerSub := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.All}
	user := &subscription.Subscription{Channel: subscription.MyAccountChannel, Asset: asset.All, Authenticated: true}
	platform := &subscription.Subscription{Channel: ChannelDerivativesPlatformState, Asset: asset.All}
	for _, tc := range []struct {
		s   *subscription.Subscription
		a   asset.Item
		exp bool
	}{
		{tickerSub, asset.Spot, true},
		{tickerSub, asset.USDCMarginedFutures, true},
		{tickerSub, asset.Empty, false},
		{user, asset.Spot, false},
		{user, asset.CoinMarginedFutures, false},
		{platform, asset.CoinMarginedFutures, false},
	} {
		assert.Equalf(t, tc.exp, subscriptionPerPair(tc.s, tc.a), "subscriptionPerPair should report %s on %s correctly", tc.s.Channel, tc.a)
	}

	qualified, err := qualifyPairChannel(&subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.FiveMin}, asset.Spot, currency.NewBTCUSD())
	require.NoError(t, err, "qualifyPairChannel must not error for five minute candles")
	assert.Equal(t, wsChannelCandles, qualified, "qualifyPairChannel should name the Advanced Trade channel")
	_, err = qualifyPairChannel(&subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.OneMin}, asset.Spot, currency.NewBTCUSD())
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "qualifyPairChannel should error for candles other than five minutes")
	qualified, err = qualifyPairChannel(&subscription.Subscription{Channel: subscription.TickerChannel}, asset.USDCMarginedFutures, testUSDCPerpetual)
	require.NoError(t, err, "qualifyPairChannel must not error for a gateway ticker")
	assert.Equal(t, "ticker.BTC_USDC-PERPETUAL.100ms", qualified, "qualifyPairChannel should name the gateway channel for the instrument")

	for _, tc := range []struct {
		s   *subscription.Subscription
		a   asset.Item
		exp string
		err error
	}{
		{s: &subscription.Subscription{Channel: subscription.HeartbeatChannel}, a: asset.Empty},
		{s: &subscription.Subscription{Channel: subscription.MyAccountChannel, Authenticated: true}, a: asset.Empty, exp: wsChannelUser},
		{s: &subscription.Subscription{Channel: ChannelStatus}, a: asset.Empty, err: errChannelNeedsPairs},
		{s: &subscription.Subscription{Channel: ChannelDerivativesPlatformState}, a: asset.Empty, exp: ChannelDerivativesPlatformState},
		{s: &subscription.Subscription{Channel: "nonsense"}, a: asset.Empty, err: subscription.ErrNotSupported},
		{s: &subscription.Subscription{Channel: ChannelStatus, Asset: asset.All}, a: asset.USDCMarginedFutures},
		{s: &subscription.Subscription{Channel: ChannelStatus, Asset: asset.USDCMarginedFutures}, a: asset.USDCMarginedFutures, err: subscription.ErrNotSupported},
		{s: &subscription.Subscription{Channel: ChannelDerivativesUserPortfolio, Asset: asset.CoinMarginedFutures, Authenticated: true}, a: asset.CoinMarginedFutures, exp: "user.portfolio.any"},
		{s: &subscription.Subscription{Channel: ChannelDerivativesBook, Asset: asset.All}, a: asset.Spot},
		{s: &subscription.Subscription{Channel: ChannelDerivativesBook, Asset: asset.Spot}, a: asset.Spot, err: subscription.ErrNotSupported},
		{s: &subscription.Subscription{Channel: subscription.MyAccountChannel, Asset: asset.Futures, Authenticated: true}, a: asset.Futures, exp: wsChannelUser},
	} {
		qualified, err := qualifyChannel(tc.s, tc.a)
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "qualifyChannel should error for %s on %s", tc.s.Channel, tc.a)
			continue
		}
		require.NoErrorf(t, err, "qualifyChannel must not error for %s on %s", tc.s.Channel, tc.a)
		assert.Equalf(t, tc.exp, qualified, "qualifyChannel should qualify %s on %s", tc.s.Channel, tc.a)
	}

	tmpl, err := new(Exchange).GetSubscriptionTemplate(nil)
	require.NoError(t, err, "GetSubscriptionTemplate must not error")
	assert.NotNil(t, tmpl, "GetSubscriptionTemplate should return a template")
}

func TestIsDerivativesAsset(t *testing.T) {
	t.Parallel()
	for a, exp := range map[asset.Item]bool{asset.USDCMarginedFutures: true, asset.CoinMarginedFutures: true, asset.Spot: false, asset.Futures: false, asset.Empty: false} {
		assert.Equalf(t, exp, isDerivativesAsset(a), "isDerivativesAsset should report %s correctly", a)
	}
}

func TestGroupByChannel(t *testing.T) {
	t.Parallel()
	btcTicker := &subscription.Subscription{QualifiedChannel: wsChannelTicker, Pairs: currency.Pairs{currency.NewBTCUSD()}}
	book := &subscription.Subscription{QualifiedChannel: wsChannelLevel2}
	ethTicker := &subscription.Subscription{QualifiedChannel: wsChannelTicker, Pairs: currency.Pairs{testETHUSD}}
	assert.Equal(t, []subscription.List{{btcTicker, ethTicker}, {book}}, groupByChannel(subscription.List{btcTicker, book, ethTicker}), "groupByChannel should group subscriptions by channel in order")
	assert.Empty(t, groupByChannel(nil), "groupByChannel should return nothing for no subscriptions")
}

// advancedTradeVenue mocks Advanced Trade's websocket: it confirms subscribe and unsubscribe requests with the
// connection's subscriptions, streams a USDC alias under its USD product as Coinbase does, refuses an unknown channel
// or a product channel's request without products, and records the requests it receives
type advancedTradeVenue struct {
	mu       sync.Mutex
	requests []WebsocketWireRequest
	// refuse holds channels the venue refuses, with its refusal
	refuse map[string]string
	// ignore holds products the venue leaves out of its subscriptions, as it does an unknown product
	ignore map[string]bool
	// skip is how many sequence numbers the venue skips before its next message, as if it had sent messages the
	// client dropped
	skip uint64
	// delayOnce holds channels whose next confirmation the venue sends late, by the time held, as it does under load
	delayOnce map[string]time.Duration
}

// skipSequence has the venue skip sequence numbers before its next message
func (v *advancedTradeVenue) skipSequence(n uint64) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.skip += n
}

// requestsFor returns the requests the mock received for a channel
func (v *advancedTradeVenue) requestsFor(channel string) []WebsocketWireRequest {
	v.mu.Lock()
	defer v.mu.Unlock()
	return slices.DeleteFunc(slices.Clone(v.requests), func(r WebsocketWireRequest) bool { return r.Channel != channel })
}

// allRequests returns every request the mock received, in order
func (v *advancedTradeVenue) allRequests() []WebsocketWireRequest {
	v.mu.Lock()
	defer v.mu.Unlock()
	return slices.Clone(v.requests)
}

// upgrader returns the handler of a connection to the mock, which keeps each connection's subscriptions
func (v *advancedTradeVenue) upgrader() func(testing.TB, []byte, *gws.Conn) error {
	var mu sync.Mutex
	state := make(map[*gws.Conn]map[string][]string)
	sequence := make(map[*gws.Conn]uint64)
	return func(_ testing.TB, msg []byte, conn *gws.Conn) error {
		var req WebsocketWireRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			return err
		}
		v.mu.Lock()
		v.requests = append(v.requests, req)
		refusal, refused := v.refuse[req.Channel]
		skip := v.skip
		v.skip = 0
		delay := v.delayOnce[req.Channel]
		delete(v.delayOnce, req.Channel)
		v.mu.Unlock()
		// A late confirmation holds up the connection's later requests too, as the venue reads them in turn
		time.Sleep(delay)
		mu.Lock()
		defer mu.Unlock()
		subs, ok := state[conn]
		if !ok {
			subs = make(map[string][]string)
			state[conn] = subs
		}
		known := slices.Contains([]string{wsChannelHeartbeats, wsChannelTicker, ChannelTickerBatch, wsChannelCandles, wsChannelMarketTrades, wsChannelLevel2, ChannelStatus, wsChannelUser, ChannelFuturesBalanceSummary}, req.Channel)
		product := req.Channel != wsChannelHeartbeats && req.Channel != wsChannelUser && req.Channel != ChannelFuturesBalanceSummary
		if refused || !known || (product && len(req.ProductIDs) == 0 && req.Type == wsRequestSubscribe) {
			if !refused {
				refusal = "authentication failure"
			}
			return conn.WriteJSON(map[string]string{"type": wsMessageError, "message": refusal})
		}
		for _, id := range req.ProductIDs {
			if base, ok := strings.CutSuffix(id, "-USDC"); ok && req.Channel != wsChannelUser && base != "USDT" && base != "EURC" {
				id = base + "-USD"
			}
			v.mu.Lock()
			ignored := v.ignore[id]
			v.mu.Unlock()
			switch {
			case ignored:
			case req.Type == wsRequestSubscribe && !slices.Contains(subs[req.Channel], id):
				subs[req.Channel] = append(subs[req.Channel], id)
			case req.Type == wsRequestUnsubscribe:
				subs[req.Channel] = slices.DeleteFunc(subs[req.Channel], func(s string) bool { return s == id })
			}
		}
		if !product {
			if req.Type == wsRequestSubscribe {
				subs[req.Channel] = []string{req.Channel}
			} else {
				delete(subs, req.Channel)
			}
		}
		for channel, products := range subs {
			if len(products) == 0 {
				delete(subs, channel)
			}
		}
		events, err := json.Marshal([]WebsocketSubscriptionsEvent{{Subscriptions: subs}})
		if err != nil {
			return err
		}
		sequence[conn] += skip
		frame, err := json.Marshal(&WebsocketEnvelope{Channel: wsChannelSubscriptions, Sequence: sequence[conn], Events: events})
		sequence[conn]++
		if err != nil {
			return err
		}
		return conn.WriteMessage(gws.TextMessage, frame)
	}
}

// connectAdvancedTradeTestExchange connects a websocket test exchange's Advanced Trade connections to a new mock venue,
// with the subscriptions given; the gateway assets have no pairs, so its connections carry nothing
func connectAdvancedTradeTestExchange(t *testing.T, venue *advancedTradeVenue, subs subscription.List) (*Exchange, error) {
	t.Helper()
	server := httptest.NewTestServer(t, mockws.CurryWsMockUpgrader(t, venue.upgrader()))
	// The exchange dials with its own dialer, which reaches a listener but not the in-memory network
	server.Start()
	ex := newWebsocketTestExchange(t)
	for _, a := range []asset.Item{asset.USDCMarginedFutures, asset.CoinMarginedFutures} {
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, nil, true), "StorePairs must not error for %s", a)
	}
	require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(server.URL, "http")), "SetAllConnectionURLs must not error")
	ex.Features.Subscriptions = subs
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	err := ex.Websocket.Connect(t.Context())
	t.Cleanup(func() { stopWebsocketTestExchange(t, ex) })
	return ex, err
}

func TestWebsocketSubscribe(t *testing.T) {
	t.Parallel()
	venue := new(advancedTradeVenue)
	ex, err := connectAdvancedTradeTestExchange(t, venue, subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
		{Enabled: true, Asset: asset.All, Channel: subscription.OrderbookChannel},
		{Enabled: true, Asset: asset.All, Channel: subscription.MyAccountChannel, Authenticated: true},
	})
	require.NoError(t, err, "Connect must subscribe to the mock venue")
	assert.Len(t, ex.Websocket.GetSubscriptions(), 6, "every subscription should be recorded")

	requests := venue.allRequests()
	require.NotEmpty(t, requests, "the venue must receive requests")
	assert.Len(t, venue.requestsFor(wsChannelHeartbeats), 2, "each connection should subscribe to its heartbeats")
	for _, r := range venue.requestsFor(wsChannelHeartbeats) {
		assert.Equal(t, wsRequestSubscribe, r.Type, "heartbeats should be subscribed")
	}
	tickers := venue.requestsFor(wsChannelTicker)
	require.Len(t, tickers, 1, "the ticker channel must be subscribed in one request")
	assert.ElementsMatch(t, []string{"BTC-USD", "ETH-USD"}, tickers[0].ProductIDs, "the ticker request should name each product once, the alias streaming as its USD product")
	assert.Empty(t, tickers[0].JWT, "a market data request should carry no JWT")
	books := venue.requestsFor(wsChannelLevel2)
	require.Len(t, books, 1, "the level2 channel must be subscribed in one request")
	assert.ElementsMatch(t, []string{"BTC-USD", "ETH-USD", "BIP-20DEC30-CDE"}, books[0].ProductIDs, "the level2 request should name every asset's products")
	users := venue.requestsFor(wsChannelUser)
	require.Len(t, users, 1, "the user channel must be subscribed once")
	assert.Empty(t, users[0].ProductIDs, "the user channel should be subscribed for all products")
	verifyTestJWT(t, users[0].JWT)

	// Each connection's first request is its heartbeats
	var marketDataFirst, userFirst bool
	for i, r := range requests {
		if r.Channel == wsChannelHeartbeats && i == 0 {
			marketDataFirst = r.JWT == ""
			userFirst = r.JWT != ""
		}
	}
	assert.True(t, marketDataFirst || userFirst, "a connection should subscribe to its heartbeats before its channels")

	var bookSubs subscription.List
	for _, s := range ex.Websocket.GetSubscriptions() {
		if s.QualifiedChannel == wsChannelLevel2 && s.Asset == asset.Futures {
			bookSubs = append(bookSubs, s)
		}
	}
	require.Len(t, bookSubs, 1, "the future's book must be subscribed")
	conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
	require.NoError(t, err, "GetConnection must not error")
	require.NoError(t, ex.Websocket.UnsubscribeChannels(t.Context(), conn, bookSubs), "UnsubscribeChannels must not error")
	unsubscribe := venue.requestsFor(wsChannelLevel2)
	require.Len(t, unsubscribe, 2, "the level2 channel must receive the unsubscribe request")
	assert.Equal(t, WebsocketWireRequest{Type: wsRequestUnsubscribe, Channel: wsChannelLevel2, ProductIDs: []string{"BIP-20DEC30-CDE"}}, unsubscribe[1], "the unsubscribe request should name the product")
	assert.Len(t, ex.Websocket.GetSubscriptions(), 5, "the unsubscribed subscription should be removed")
}

func TestWebsocketRequestTimeoutReconnects(t *testing.T) {
	t.Parallel()
	// The level2 confirmation arrives once the request has timed out, and would be taken for the next request's
	venue := &advancedTradeVenue{delayOnce: map[string]time.Duration{wsChannelLevel2: 3 * time.Second}}
	server := httptest.NewTestServer(t, mockws.CurryWsMockUpgrader(t, venue.upgrader()))
	// The exchange dials with its own dialer, which reaches a listener but not the in-memory network
	server.Start()
	ex := newWebsocketTestExchange(t)
	raw, err := testExchangeConfig()
	require.NoError(t, err, "testExchangeConfig must not error")
	var exchCfg config.Exchange
	require.NoError(t, json.Unmarshal(raw, &exchCfg), "the test configuration must decode")
	exchCfg.WebsocketResponseMaxLimit = time.Second
	ex.Websocket = sharedtestvalues.NewTestWebsocket()
	require.NoError(t, ex.setupWebsocket(&exchCfg), "setupWebsocket must not error")
	require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(server.URL, "http")), "SetAllConnectionURLs must not error")
	for _, a := range []asset.Item{asset.Futures, asset.USDCMarginedFutures, asset.CoinMarginedFutures} {
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, nil, true), "StorePairs must not error for %s", a)
	}
	ex.Features.Subscriptions = subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
	}
	err = ex.Websocket.Connect(t.Context())
	t.Cleanup(func() { stopWebsocketTestExchange(t, ex) })
	assert.ErrorIs(t, err, websocket.ErrSignatureTimeout, "Connect should report the request that timed out")
	// The connection is closed, then dialled and subscribed again
	require.Eventually(t, func() bool {
		return len(venue.requestsFor(wsChannelHeartbeats)) >= 2 && len(ex.Websocket.GetSubscriptions()) == 6
	}, 30*time.Second, 10*time.Millisecond, "the websocket must reconnect and subscribe afresh")
	for _, channel := range []string{wsChannelTicker, wsChannelLevel2, wsChannelMarketTrades} {
		var held int
		for _, s := range ex.Websocket.GetSubscriptions() {
			if s.QualifiedChannel == channel {
				held++
			}
		}
		// The USDC alias streams as its USD product, so two products are subscribed
		assert.Equalf(t, 2, held, "the websocket should hold the %s subscriptions it streams once it reconnects", channel)
	}
}

func TestWebsocketRequestContextEndReconnects(t *testing.T) {
	t.Parallel()
	venue := new(advancedTradeVenue)
	ex, err := connectAdvancedTradeTestExchange(t, venue, subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel}})
	require.NoError(t, err, "Connect must subscribe to the mock venue")
	conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
	require.NoError(t, err, "GetConnection must not error")
	// The venue confirms the level2 request after its caller stopped waiting
	venue.mu.Lock()
	venue.delayOnce = map[string]time.Duration{wsChannelLevel2: 600 * time.Millisecond}
	venue.mu.Unlock()
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	_, err = ex.wsSendRequest(ctx, conn, wsRequestSubscribe, wsChannelLevel2, []string{"BTC-USD"}, false)
	require.ErrorIs(t, err, context.DeadlineExceeded, "wsSendRequest must return its context's error")
	// The connection is closed, then dialled and subscribed again, rather than reused for the next request
	require.Eventually(t, func() bool {
		return len(venue.requestsFor(wsChannelHeartbeats)) >= 2 && len(ex.Websocket.GetSubscriptions()) == 2
	}, 30*time.Second, 10*time.Millisecond, "the websocket must reconnect and subscribe afresh")
}

func TestWebsocketRequestCancelledBeforeSend(t *testing.T) {
	t.Parallel()
	venue := new(advancedTradeVenue)
	ex, err := connectAdvancedTradeTestExchange(t, venue, subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel}})
	require.NoError(t, err, "Connect must subscribe to the mock venue")
	conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
	require.NoError(t, err, "GetConnection must not error")
	counting := &closeCountingConnection{Connection: conn}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = ex.wsSendRequest(ctx, counting, wsRequestSubscribe, wsChannelLevel2, []string{"BTC-USD"}, false)
	require.ErrorIs(t, err, context.Canceled, "wsSendRequest must return its context's error")
	// A request never sent has no reply to arrive late, so the connection stays up
	assert.Empty(t, venue.requestsFor(wsChannelLevel2), "wsSendRequest should not send a request its caller stopped waiting for")
	assert.Zero(t, counting.closes.Load(), "wsSendRequest should not close the connection for a request it did not send")
}

// closeCountingConnection counts the close frames sent on a connection
type closeCountingConnection struct {
	websocket.Connection
	closes atomic.Int32
}

// SendRawMessage counts a close frame, then sends the message
func (c *closeCountingConnection) SendRawMessage(ctx context.Context, epl request.EndpointLimit, messageType int, message []byte) error {
	if messageType == gws.CloseMessage {
		c.closes.Add(1)
	}
	return c.Connection.SendRawMessage(ctx, epl, messageType, message)
}

func TestWebsocketSubscribeErrors(t *testing.T) {
	t.Parallel()
	venue := &advancedTradeVenue{refuse: map[string]string{ChannelTickerBatch: "rate limit exceeded"}, ignore: map[string]bool{"ETH-USD": true}}
	ex, err := connectAdvancedTradeTestExchange(t, venue, subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: ChannelTickerBatch},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
	})
	assert.ErrorIs(t, err, errWebsocketRequestRefused, "Connect should report a refused request")
	assert.ErrorIs(t, err, errWebsocketSubscribeUnconfirmed, "Connect should report a product the confirmation leaves out")
	subs := ex.Websocket.GetSubscriptions()
	require.Len(t, subs, 1, "only the confirmed subscription must be recorded")
	assert.Equal(t, currency.Pairs{currency.NewBTCUSD().Format(productIDFormat)}, subs[0].Pairs, "the confirmed product's subscription should be recorded")
}

func TestWebsocketResynchronise(t *testing.T) {
	t.Parallel()
	venue := new(advancedTradeVenue)
	ex, err := connectAdvancedTradeTestExchange(t, venue, subscription.List{
		{Enabled: true, Asset: asset.Futures, Channel: subscription.OrderbookChannel},
		{Enabled: true, Asset: asset.Futures, Channel: subscription.TickerChannel},
	})
	require.NoError(t, err, "Connect must subscribe to the mock venue")
	conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
	require.NoError(t, err, "GetConnection must not error")
	require.NoError(t, ex.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{
		Exchange: ex.Name, Pair: testFuturesPair, Asset: asset.Futures, Bids: orderbook.Levels{{Price: 1, Amount: 1}}, Asks: orderbook.Levels{{Price: 2, Amount: 1}}, LastUpdated: time.Now(),
	}), "LoadSnapshot must not error")

	// The venue has confirmed 3 requests, numbered from 0, so its next message is 3; one that is 4 follows a message
	// the connection dropped, after which the venue goes on from 5
	venue.skipSequence(2)
	err = ex.wsHandleData(t.Context(), conn, []byte(`{"channel":"heartbeats","sequence_num":4,"events":[]}`))
	require.ErrorIs(t, err, errWebsocketSequenceGap, "a dropped message must be reported")
	_, err = ex.Websocket.Orderbook.GetOrderbook(testFuturesPair, asset.Futures)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "the connection's books should be invalidated until resubscribing reloads them")
	require.Eventually(t, func() bool { return len(venue.requestsFor(wsChannelLevel2)) == 3 }, 5*time.Second, time.Millisecond, "the books must be unsubscribed and subscribed again")
	books := venue.requestsFor(wsChannelLevel2)
	assert.Equal(t, []string{wsRequestSubscribe, wsRequestUnsubscribe, wsRequestSubscribe}, []string{books[0].Type, books[1].Type, books[2].Type}, "the books should be resubscribed")
	assert.Len(t, venue.requestsFor(wsChannelTicker), 1, "a channel without state should not be resubscribed")
}

func TestWebsocketManageChannelUnknown(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn, err := ex.Websocket.CreateTestConnection(websocketMarketDataConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	err = ex.wsManageChannel(t.Context(), conn, wsRequestSubscribe, subscription.List{{Channel: "nonsense", QualifiedChannel: "nonsense"}})
	assert.ErrorIs(t, err, errWebsocketChannelUnknown, "wsManageChannel should error for a channel Advanced Trade does not stream")
}

func TestConfirmationLists(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	confirmation := &WebsocketSubscriptionsEvent{Subscriptions: map[string][]string{wsChannelTicker: {"ETH-USD", "TGBP-USDC"}, wsChannelUser: {wsChannelUser}}}
	for _, tc := range []struct {
		s   *subscription.Subscription
		exp bool
	}{
		{&subscription.Subscription{Asset: asset.Spot, Pairs: currency.Pairs{testETHUSDC.Format(currency.PairFormat{Uppercase: true, Delimiter: "-"})}}, true},
		{&subscription.Subscription{Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPairWithDelimiter("TGBP", "USDC", "-")}}, true},
		{&subscription.Subscription{Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPairWithDelimiter("BTC", "USD", "-")}}, false},
	} {
		assert.Equalf(t, tc.exp, ex.confirmationLists(confirmation, wsChannelTicker, tc.s), "confirmationLists should report %s correctly", tc.s.Pairs)
	}
	assert.True(t, ex.confirmationLists(confirmation, wsChannelUser, &subscription.Subscription{}), "confirmationLists should report an account channel by name")
	assert.False(t, ex.confirmationLists(confirmation, ChannelFuturesBalanceSummary, &subscription.Subscription{}), "confirmationLists should report an account channel left out")
}

func TestWebsocketSendRequestErrors(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	ex.SetCredentials(nil)
	conn, err := ex.Websocket.CreateTestConnection(websocketUserConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	_, err = ex.wsSendRequest(t.Context(), conn, wsRequestSubscribe, wsChannelUser, nil, true)
	assert.Error(t, err, "wsSendRequest should error without credentials for an authenticated request")
	_, err = ex.wsSendRequest(t.Context(), conn, wsRequestSubscribe, wsChannelHeartbeats, nil, false)
	assert.Error(t, err, "wsSendRequest should error on a connection that is not connected")
}

func TestWebsocketChannelOf(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		s       *subscription.Subscription
		ok      bool
		account bool
	}{
		{&subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: wsChannelTicker, Asset: asset.Spot}, true, false},
		{&subscription.Subscription{Channel: subscription.MyAccountChannel, QualifiedChannel: wsChannelUser}, true, true},
		{&subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: "ticker.BTC-PERPETUAL.100ms", Asset: asset.CoinMarginedFutures}, false, false},
		{&subscription.Subscription{Channel: ChannelDerivativesPlatformState, QualifiedChannel: ChannelDerivativesPlatformState}, false, false},
	} {
		ch, ok := advancedTradeChannelOf(tc.s)
		assert.Equalf(t, tc.ok, ok, "advancedTradeChannelOf should report %s correctly", tc.s.QualifiedChannel)
		assert.Equalf(t, tc.account, ch.account, "advancedTradeChannelOf should report whether %s is an account channel", tc.s.QualifiedChannel)
	}
}

// errorConnection is a connection whose requests fail
type errorConnection struct {
	websocket.Connection
	err error
}

// SendMessageReturnResponse fails
func (c *errorConnection) SendMessageReturnResponse(context.Context, request.EndpointLimit, any, any) ([]byte, error) {
	return nil, c.err
}

func TestWebsocketSendRequestResponses(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	failure := errors.New("send failure")
	_, err := ex.wsSendRequest(t.Context(), &errorConnection{err: failure}, wsRequestSubscribe, wsChannelHeartbeats, nil, false)
	assert.ErrorIs(t, err, failure, "wsSendRequest should return the send failure")
	for response, exp := range map[string]error{
		`[`: nil,
		`{"channel":"subscriptions","events":{}}`: nil,
		`{"channel":"subscriptions","events":[]}`: common.ErrNoResponse,
		`null`:                              common.ErrNoResponse,
		`{"type":"error","message":"nope"}`: errWebsocketRequestRefused,
	} {
		_, err := ex.wsSendRequest(t.Context(), &responseConnection{response: []byte(response)}, wsRequestSubscribe, wsChannelHeartbeats, nil, false)
		if exp == nil {
			assert.Errorf(t, err, "wsSendRequest should error for %s", response)
		} else {
			assert.ErrorIsf(t, err, exp, "wsSendRequest should error for %s", response)
		}
	}
}

// responseConnection is a connection answering every request with one response
type responseConnection struct {
	websocket.Connection
	response []byte
}

// SendMessageReturnResponse answers with the connection's response
func (c *responseConnection) SendMessageReturnResponse(context.Context, request.EndpointLimit, any, any) ([]byte, error) {
	return c.response, nil
}

func TestTypesBooleanPostOnly(t *testing.T) {
	t.Parallel()
	var o WebsocketUserOrder
	require.NoError(t, json.Unmarshal([]byte(`{"post_only":"true"}`), &o), "a post only flag must decode")
	assert.Equal(t, types.Boolean(true), o.PostOnly, "a post only flag sent as a string should decode")
}

func TestSetupFromVersion17Config(t *testing.T) {
	t.Parallel()
	raw, err := testExchangeConfig()
	require.NoError(t, err, "testExchangeConfig must not error")
	// A configuration the previous version saved: its default subscriptions, and spot and futures pairs alone, in the
	// global format
	saved, err := jsonparser.Set(raw, []byte(`[{"enabled":true,"channel":"heartbeat"},{"enabled":true,"channel":"status","asset":"all"},`+
		`{"enabled":true,"channel":"ticker","asset":"spot"},{"enabled":true,"channel":"candles","asset":"spot"},`+
		`{"enabled":true,"channel":"allTrades","asset":"spot"},{"enabled":true,"channel":"orderbook","asset":"spot"},`+
		`{"enabled":true,"channel":"myAccount","asset":"all","authenticated":true},{"enabled":true,"channel":"ticker_batch","asset":"spot"}]`), "features", "subscriptions")
	require.NoError(t, err, "jsonparser.Set must not error for the subscriptions")
	saved, err = jsonparser.Set(saved, []byte(`{"bypassConfigFormatUpgrades":false,"requestFormat":{"uppercase":true,"delimiter":"-"},`+
		`"configFormat":{"uppercase":true,"delimiter":"-"},"useGlobalFormat":true,"pairs":{`+
		`"futures":{"assetEnabled":true,"enabled":"BIP-20DEC30-CDE","available":"BIP-20DEC30-CDE,BIT-30OCT26-CDE"},`+
		`"spot":{"assetEnabled":true,"enabled":"BTC-USD","available":"BTC-USD,ETH-USD"}}}`), "currencyPairs")
	require.NoError(t, err, "jsonparser.Set must not error for the pairs")
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"name":"test","version":17,"encryptConfig":-1,"exchanges":[`+string(saved)+`]}`), 0o600), "WriteFile must save the version 17 config")

	var cfg config.Config
	require.NoError(t, cfg.ReadConfigFromFile(path, true), "ReadConfigFromFile must upgrade the version 17 config")
	exchCfg, err := cfg.GetExchangeConfig("Coinbase")
	require.NoError(t, err, "GetExchangeConfig must not error")
	testsubs.EqualLists(t, defaultSubscriptions.Clone(), exchCfg.Features.Subscriptions.Clone())

	ex := new(Exchange)
	ex.SetDefaults()
	ex.Websocket = sharedtestvalues.NewTestWebsocket()
	require.NoError(t, ex.Setup(exchCfg), "Setup must not error for the upgraded configuration")
	ex.Name = t.Name()
	for a, exp := range map[asset.Item]string{asset.Spot: "-", asset.Futures: "-", asset.USDCMarginedFutures: "_", asset.CoinMarginedFutures: "-"} {
		f, err := ex.GetPairFormat(a, true)
		require.NoErrorf(t, err, "GetPairFormat must not error for %s", a)
		assert.Equalf(t, exp, f.Delimiter, "Setup should give %s's requests their own delimiter in place of the saved global format", a)
	}
	enabled, err := ex.GetEnabledPairs(asset.Futures)
	require.NoError(t, err, "GetEnabledPairs must not error for futures")
	assert.True(t, enabled.Contains(testFuturesPair, true), "Setup should keep the saved futures pairs")
	for a, pairs := range map[asset.Item]currency.Pairs{asset.USDCMarginedFutures: {testUSDCPerpetual}, asset.CoinMarginedFutures: {testCoinPerpetual}} {
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, pairs, false), "StorePairs must not error for available %s pairs", a)
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, pairs, true), "StorePairs must not error for enabled %s pairs", a)
	}
	marketData, err := ex.generateMarketDataSubscriptions()
	require.NoError(t, err, "generateMarketDataSubscriptions must not error for the upgraded configuration")
	streams, err := ex.generateDerivativesStreamsSubscriptions()
	require.NoError(t, err, "generateDerivativesStreamsSubscriptions must not error for the upgraded configuration")
	subs := slices.Concat(marketData, streams)
	for _, a := range []asset.Item{asset.Spot, asset.Futures, asset.USDCMarginedFutures, asset.CoinMarginedFutures} {
		for _, channel := range []string{subscription.TickerChannel, subscription.CandlesChannel, subscription.AllTradesChannel, subscription.OrderbookChannel} {
			assert.Truef(t, slices.ContainsFunc(subs, func(s *subscription.Subscription) bool { return s.Channel == channel && s.Asset == a }), "the upgraded configuration should subscribe %s's %s channel", a, channel)
		}
	}
}
