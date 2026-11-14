package binanceus

import (
	"context"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// routedTestExchange returns an exchange named after the test whose REST requests are answered by the handler of
// their method and path, failing the test for any other request
func routedTestExchange(t *testing.T, routes map[string]http.HandlerFunc) *Exchange {
	t.Helper()
	return newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		handler, ok := routes[r.Method+" "+r.URL.Path]
		if !assert.Truef(t, ok, "%s %s should not be requested", r.Method, r.URL.Path) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		handler(w, r)
	})
}

// reply returns a handler answering every request with body, after checking its query when exp is not nil
func reply(t *testing.T, exp url.Values, body string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if exp != nil {
			q := r.URL.Query()
			q.Del("timestamp")
			q.Del("signature")
			q.Del("recvWindow")
			assert.Equalf(t, exp, q, "%s %s should send the expected parameters", r.Method, r.URL.Path)
		}
		writeResponse(t, w, body)
	}
}

// replyError returns a handler answering every request with a Binance.US error
func replyError(t *testing.T, code int, msg string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		writeResponse(t, w, `{"code":`+strconv.Itoa(code)+`,"msg":"`+msg+`"}`)
	}
}

const testExchangeInfo = `{"timezone":"UTC","serverTime":1794466395184,"symbols":[` +
	`{"symbol":"BTCUSDT","status":"TRADING","baseAsset":"BTC","quoteAsset":"USDT","isSpotTradingAllowed":true,"filters":[` +
	`{"filterType":"PRICE_FILTER","minPrice":"0.01000000","maxPrice":"1000000.00000000","tickSize":"0.01000000"},` +
	`{"filterType":"PERCENT_PRICE_BY_SIDE","bidMultiplierUp":"5","bidMultiplierDown":"0.2","askMultiplierUp":"4","askMultiplierDown":"0.1","avgPriceMins":5},` +
	`{"filterType":"LOT_SIZE","minQty":"0.00001000","maxQty":"9000.00000000","stepSize":"0.00001000"},` +
	`{"filterType":"NOTIONAL","minNotional":"1.00000000","applyMinToMarket":true,"maxNotional":"9000000.00000000","applyMaxToMarket":false,"avgPriceMins":5},` +
	`{"filterType":"ICEBERG_PARTS","limit":10},` +
	`{"filterType":"MARKET_LOT_SIZE","minQty":"0.00000000","maxQty":"40.00000000","stepSize":"0.00000000"},` +
	`{"filterType":"MAX_NUM_ORDERS","maxNumOrders":200},` +
	`{"filterType":"MAX_NUM_ALGO_ORDERS","maxNumAlgoOrders":5}]},` +
	`{"symbol":"ETHUSDT","status":"BREAK","baseAsset":"ETH","quoteAsset":"USDT","isSpotTradingAllowed":true,"filters":[]},` +
	`{"symbol":"LTCUSDT","status":"TRADING","baseAsset":"LTC","quoteAsset":"USDT","isSpotTradingAllowed":false,"filters":[]}]}`

func TestSetDefaults(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	assert.Equal(t, "Binanceus", ex.Name, "SetDefaults should name the exchange")
	for endpoint, exp := range map[exchange.URL]string{
		exchange.RestSpot:       apiURL,
		exchange.WebsocketSpot:  websocketStreamsURL,
		exchange.WebsocketTrade: websocketAPIURL,
	} {
		got, err := ex.API.Endpoints.GetURL(endpoint)
		require.NoErrorf(t, err, "GetURL must not error for %s", endpoint)
		assert.Equalf(t, exp, got, "SetDefaults should set the %s endpoint", endpoint)
	}
}

func TestSetup(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	cfg, err := ex.GetStandardConfig()
	require.NoError(t, err, "GetStandardConfig must not error")
	cfg.Enabled = false
	require.NoError(t, ex.Setup(cfg), "Setup must not error for a disabled exchange")
	assert.False(t, ex.IsEnabled(), "Setup should leave a disabled exchange disabled")
	assert.Error(t, ex.Setup(nil), "Setup should error without a configuration")

	ex = new(Exchange)
	ex.SetDefaults()
	cfg, err = ex.GetStandardConfig()
	require.NoError(t, err, "GetStandardConfig must not error")
	cfg.API.AuthenticatedWebsocketSupport = true
	ex.wsAPI.set(newFakeConnection(), time.Second)
	require.NoError(t, ex.Setup(cfg), "Setup must not error")
	assert.NotNil(t, ex.orderbookSync, "Setup should set up the order book synchronisation")
	assert.Nil(t, ex.wsAPI.kept(), "Setup should forget a WebSocket API connection of an earlier setup")
}

func TestParseOrderID(t *testing.T) {
	t.Parallel()
	id, err := parseOrderID("")
	require.NoError(t, err, "parseOrderID must not error for an empty ID")
	assert.Zero(t, id, "an empty ID should be 0")
	id, err = parseOrderID("12345")
	require.NoError(t, err, "parseOrderID must not error")
	assert.Equal(t, uint64(12345), id, "parseOrderID should parse the ID")
	_, err = parseOrderID("x")
	assert.ErrorIs(t, err, errInvalidOrderID, "parseOrderID should reject an ID that is not a number")
}

func TestFormatOrderListID(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "list-5", formatOrderListID(5), "formatOrderListID should tag the order list ID")
}

func TestParseSpotOrderID(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderID string
		id      uint64
		isList  bool
		err     error
	}{
		{orderID: "", id: 0},
		{orderID: "7", id: 7},
		{orderID: "list-5", id: 5, isList: true},
		{orderID: "list-x", isList: true, err: errInvalidOrderID},
		{orderID: "x", err: errInvalidOrderID},
	} {
		id, isList, err := parseSpotOrderID(tc.orderID)
		require.ErrorIsf(t, err, tc.err, "parseSpotOrderID must return the expected error for %q", tc.orderID)
		assert.Equalf(t, tc.id, id, "parseSpotOrderID should return the ID of %q", tc.orderID)
		assert.Equalf(t, tc.isList, isList, "parseSpotOrderID should report whether %q is an order list's", tc.orderID)
	}
}

func TestRemainingAmount(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0.8, remainingAmount(1.1, 0.3), "remainingAmount should subtract exactly")
}

func TestAveragePrice(t *testing.T) {
	t.Parallel()
	assert.Zero(t, averagePrice(10, 0), "averagePrice should be zero without fills")
	assert.Equal(t, 2.5, averagePrice(10, 4), "averagePrice should divide the quote amount by the quantity")
}

func TestSpotOrderTypeString(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType order.Type
		tif       order.TimeInForce
		exp       OrderType
	}{
		{orderType: order.Limit, exp: OrderTypeLimit},
		{orderType: order.Limit, tif: order.PostOnly, exp: OrderTypeLimitMaker},
		{orderType: order.LimitMaker, exp: OrderTypeLimitMaker},
		{orderType: order.Market, exp: OrderTypeMarket},
		{orderType: order.StopMarket, exp: OrderTypeStopLoss},
		{orderType: order.StopLimit, exp: OrderTypeStopLossLimit},
		{orderType: order.TakeProfitMarket, exp: OrderTypeTakeProfit},
		{orderType: order.TakeProfitLimit, exp: OrderTypeTakeProfitLimit},
	} {
		got, err := spotOrderTypeString(tc.orderType, tc.tif)
		require.NoErrorf(t, err, "spotOrderTypeString must not error for %s %s", tc.orderType, tc.tif)
		assert.Equalf(t, tc.exp, got, "spotOrderTypeString should convert %s %s", tc.orderType, tc.tif)
	}
	_, err := spotOrderTypeString(order.TrailingStop, order.UnknownTIF)
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "spotOrderTypeString should reject an unsupported type")
}

func TestStringToSpotOrderType(t *testing.T) {
	t.Parallel()
	for orderType, exp := range map[OrderType]order.Type{
		OrderTypeLimit:           order.Limit,
		OrderTypeMarket:          order.Market,
		OrderTypeStopLoss:        order.StopMarket,
		OrderTypeStopLossLimit:   order.StopLimit,
		OrderTypeTakeProfit:      order.TakeProfitMarket,
		OrderTypeTakeProfitLimit: order.TakeProfitLimit,
		OrderTypeLimitMaker:      order.LimitMaker,
	} {
		got, err := stringToSpotOrderType(orderType)
		require.NoErrorf(t, err, "stringToSpotOrderType must not error for %s", orderType)
		assert.Equalf(t, exp, got, "stringToSpotOrderType should convert %s", orderType)
	}
	_, err := stringToSpotOrderType("UNKNOWN")
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "stringToSpotOrderType should reject an unknown type")
}

func TestSpotTimeInForce(t *testing.T) {
	t.Parallel()
	assert.Equal(t, order.PostOnly, spotTimeInForce(order.LimitMaker, order.GoodTillCancel), "a LIMIT_MAKER order should read as post only")
	assert.Equal(t, order.ImmediateOrCancel, spotTimeInForce(order.Limit, order.ImmediateOrCancel), "another type should keep its time in force")
}

func TestOrderListStatus(t *testing.T) {
	t.Parallel()
	for status, exp := range map[string]order.Status{"EXECUTING": order.Active, "ALL_DONE": order.Closed, "REJECT": order.Rejected} {
		got, err := orderListStatus(status)
		require.NoErrorf(t, err, "orderListStatus must not error for %s", status)
		assert.Equalf(t, exp, got, "orderListStatus should convert %s", status)
	}
	_, err := orderListStatus("UNKNOWN")
	assert.ErrorIs(t, err, errUnknownOrderStatus, "orderListStatus should reject an unknown status")
}

func TestOrderListDetail(t *testing.T) {
	t.Parallel()
	ex := &Exchange{Name: "test"}
	at := time.UnixMilli(1794466395184)
	d, err := ex.orderListDetail(5, "listClient", "EXECUTING", at, currency.NewBTCUSDT())
	require.NoError(t, err, "orderListDetail must not error")
	assert.Equal(t, order.Detail{
		Exchange:      "test",
		OrderID:       "list-5",
		ClientOrderID: "listClient",
		Type:          order.OCO,
		Status:        order.Active,
		AssetType:     asset.Spot,
		Date:          at,
		LastUpdated:   at,
		Pair:          currency.NewBTCUSDT(),
	}, d, "orderListDetail should convert the order list")
	_, err = ex.orderListDetail(5, "", "UNKNOWN", at, currency.NewBTCUSDT())
	assert.ErrorIs(t, err, errUnknownOrderStatus, "orderListDetail should return an unknown status")
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipUnlessLive(t, false)
		pairs, err := e.FetchTradablePairs(t.Context(), asset.Spot)
		require.NoError(t, err, "FetchTradablePairs must not error")
		assert.NotEmpty(t, pairs, "FetchTradablePairs should return the tradable pairs")
	})

	ex := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/exchangeInfo": reply(t, url.Values{}, testExchangeInfo)})
	pairs, err := ex.FetchTradablePairs(t.Context(), asset.Spot)
	require.NoError(t, err, "FetchTradablePairs must not error")
	require.Len(t, pairs, 1, "FetchTradablePairs must return only the symbols trading on the spot market")
	assert.Equal(t, "BTC-USDT", pairs[0].String(), "FetchTradablePairs should return pairs in the config format")
	_, err = ex.FetchTradablePairs(t.Context(), asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "FetchTradablePairs should reject an asset other than spot")
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipUnlessLive(t, false)
		require.NoError(t, e.UpdateTradablePairs(t.Context()), "UpdateTradablePairs must not error")
		pairs, err := e.GetAvailablePairs(asset.Spot)
		require.NoError(t, err, "GetAvailablePairs must not error")
		assert.NotEmpty(t, pairs, "UpdateTradablePairs should store the tradable pairs")
	})

	ex := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/exchangeInfo": reply(t, url.Values{}, testExchangeInfo)})
	require.NoError(t, ex.UpdateTradablePairs(t.Context()), "UpdateTradablePairs must not error")
	pairs, err := ex.GetAvailablePairs(asset.Spot)
	require.NoError(t, err, "GetAvailablePairs must not error")
	require.Len(t, pairs, 1, "UpdateTradablePairs must store the tradable pairs alone")
	assert.True(t, pairs.Contains(currency.NewBTCUSDT(), true), "UpdateTradablePairs should store the tradable pairs")
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipUnlessLive(t, false)
		require.NoError(t, e.UpdateTickers(t.Context(), asset.Spot), "UpdateTickers must not error")
		_, err := ticker.GetTicker(e.Name, spotTradablePair, asset.Spot)
		assert.NoError(t, err, "UpdateTickers should store the enabled pair's ticker")
	})

	body := `[{"symbol":"BTCUSDT","lastPrice":"70000","bidPrice":"69999","askPrice":"70001","volume":"10","closeTime":1794466395184},` +
		`{"symbol":"NOTAPAIR","lastPrice":"1","closeTime":1794466395184}]`
	ex := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/ticker/24hr": reply(t, url.Values{}, body)})
	require.NoError(t, ex.UpdateTickers(t.Context(), asset.Spot), "UpdateTickers must not error")
	tick, err := ticker.GetTicker(ex.Name, currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err, "GetTicker must not error")
	assert.Equal(t, 70000.0, tick.Last, "UpdateTickers should store the pair's ticker")
	assert.ErrorIs(t, ex.UpdateTickers(t.Context(), asset.Futures), asset.ErrNotSupported, "UpdateTickers should reject an asset other than spot")
}

func TestTickerPair(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	pair, ok, err := ex.tickerPair("BTCUSDT")
	require.NoError(t, err, "tickerPair must not error")
	assert.True(t, ok, "tickerPair should find an available pair")
	assert.True(t, pair.Equal(currency.NewBTCUSDT()), "tickerPair should return the stored pair")
	_, ok, err = ex.tickerPair("NOTAPAIR")
	require.NoError(t, err, "tickerPair must not error for an unknown symbol")
	assert.False(t, ok, "tickerPair should report an unknown symbol")
	_, _, err = ex.tickerPair("")
	assert.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "tickerPair should return an error other than an unknown symbol")
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipUnlessLive(t, false)
		tick, err := e.UpdateTicker(t.Context(), spotTradablePair, asset.Spot)
		require.NoError(t, err, "UpdateTicker must not error")
		assert.True(t, tick.Pair.Equal(spotTradablePair), "UpdateTicker should return the pair's ticker")
	})

	ex := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /api/v3/ticker/24hr": reply(t, url.Values{"symbol": {"BTCUSDT"}}, `{"symbol":"BTCUSDT","lastPrice":"70000","closeTime":1794466395184}`),
	})
	tick, err := ex.UpdateTicker(t.Context(), currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err, "UpdateTicker must not error")
	assert.Equal(t, 70000.0, tick.Last, "UpdateTicker should return the stored ticker")
	_, err = ex.UpdateTicker(t.Context(), currency.EMPTYPAIR, asset.Spot)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateTicker should reject an empty pair")
	_, err = ex.UpdateTicker(t.Context(), currency.NewBTCUSDT(), asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "UpdateTicker should reject an asset other than spot")

	empty := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/ticker/24hr": reply(t, nil, `[]`)})
	_, err = empty.UpdateTicker(t.Context(), currency.NewBTCUSDT(), asset.Spot)
	assert.ErrorIs(t, err, ticker.ErrTickerNotFound, "UpdateTicker should error when no ticker is returned")
}

func TestSpotTickerPrice(t *testing.T) {
	t.Parallel()
	ex := &Exchange{Name: "test"}
	stats := &PriceChangeStatsResponse{
		PriceChangePercent: 1.5, WeightedAveragePrice: 69000, PreviousClosePrice: 68000, LastPrice: 70000, LastQuantity: 0.1,
		BidPrice: 69999, BidQuantity: 1, AskPrice: 70001, AskQuantity: 2, OpenPrice: 68500, HighPrice: 71000, LowPrice: 67000,
		Volume: 10, QuoteVolume: 700000, CloseTime: types.Time(time.UnixMilli(1794466395184)),
	}
	assert.Equal(t, ticker.Price{
		Last: 70000, LastSize: 0.1, VolumeWeightedAveragePrice: 69000, High: 71000, Low: 67000, Bid: 69999, BidSize: 1,
		Ask: 70001, AskSize: 2, BaseVolume: 10, QuoteVolume: 700000, Open: 68500, PercentChange24Hour: 1.5, Close: 68000,
		Pair: currency.NewBTCUSDT(), ExchangeName: "test", AssetType: asset.Spot, LastUpdated: time.UnixMilli(1794466395184),
	}, ex.spotTickerPrice(stats, currency.NewBTCUSDT()), "spotTickerPrice should convert every field")
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipUnlessLive(t, false)
		book, err := e.UpdateOrderbook(t.Context(), spotTradablePair, asset.Spot)
		require.NoError(t, err, "UpdateOrderbook must not error")
		assert.NotEmpty(t, book.Bids, "UpdateOrderbook should return the book's bids")
	})

	ex := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /api/v3/depth": reply(t, url.Values{"symbol": {"BTCUSDT"}, "limit": {"1000"}}, `{"lastUpdateId":42,"bids":[["69999","1"]],"asks":[["70001","2"]]}`),
	})
	book, err := ex.UpdateOrderbook(t.Context(), currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err, "UpdateOrderbook must not error")
	assert.Equal(t, int64(42), book.LastUpdateID, "UpdateOrderbook should store the snapshot's update ID")
	assert.Equal(t, orderbook.Levels{{Price: 69999, Amount: 1}}, book.Bids, "UpdateOrderbook should store the bids")
	_, err = ex.UpdateOrderbook(t.Context(), currency.EMPTYPAIR, asset.Spot)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateOrderbook should reject an empty pair")
	_, err = ex.UpdateOrderbook(t.Context(), currency.NewBTCUSDT(), asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "UpdateOrderbook should reject an asset other than spot")

	for _, tc := range []struct {
		name   string
		stream bool
		state  subscription.State
		kept   bool
	}{
		{name: "websocket book of a subscribed order book stream", stream: true, state: subscription.SubscribedState, kept: true},
		{name: "websocket book of an order book stream unsubscribing", stream: true, state: subscription.UnsubscribingState},
		{name: "websocket book without an order book stream"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := routedTestExchange(t, map[string]http.HandlerFunc{
				"GET /api/v3/depth": reply(t, url.Values{"symbol": {"BTCUSDT"}, "limit": {"1000"}}, orderBookReply(42)),
			})
			p := currency.NewBTCUSDT()
			if tc.stream {
				s := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{p}}
				require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(nil, s), "AddSuccessfulSubscriptions must not error")
				if tc.state != subscription.SubscribedState {
					require.NoError(t, s.SetState(tc.state), "SetState must not error")
				}
			}
			streamed := loadWebsocketBook(t, ex, p, 43)
			book, err := ex.UpdateOrderbook(t.Context(), p, asset.Spot)
			require.NoError(t, err, "UpdateOrderbook must not error")
			if tc.kept {
				assert.Equal(t, int64(43), book.LastUpdateID, "UpdateOrderbook should leave in place a book the websocket updated within websocketBookFreshness")
				assert.Equal(t, streamed.Bids, book.Bids, "UpdateOrderbook should return the websocket book's bids")
				assert.False(t, book.RestSnapshot, "UpdateOrderbook should return the websocket book")
				return
			}
			assert.Equal(t, int64(42), book.LastUpdateID, "UpdateOrderbook should replace a websocket book no subscribed order book stream maintains")
			assert.True(t, book.RestSnapshot, "UpdateOrderbook should return the REST snapshot")
		})
	}

	t.Run("reply slower than websocketBookFreshness", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			ex := newWebsocketTestExchange(t)
			p := currency.NewBTCUSDT()
			require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(nil, &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{p}}), "AddSuccessfulSubscriptions must not error")
			client := orderBookClient(42, func() {
				// Binance.US has read its book for the reply when the websocket loads a newer one
				loadWebsocketBook(t, ex, p, 43)
				time.Sleep(websocketBookFreshness + time.Second)
			})
			client.Timeout = time.Minute
			require.NoError(t, ex.SetHTTPClient(client), "SetHTTPClient must not error")
			book, err := ex.UpdateOrderbook(t.Context(), p, asset.Spot)
			require.NoError(t, err, "UpdateOrderbook must not error")
			assert.Equal(t, int64(43), book.LastUpdateID, "UpdateOrderbook should leave in place a book the websocket updated after the request was made, however long the reply took")
		})
	})

	for _, stop := range []string{"shutdown", "unsubscription"} {
		t.Run("book a snapshot fetch in flight loads after "+stop, func(t *testing.T) {
			t.Parallel()
			ex, conn := newSubscriptionTestExchange(t)
			p := currency.NewBTCUSDT()
			require.NoError(t, ex.SetHTTPClient(orderBookClient(12, nil)), "SetHTTPClient must not error")
			subs := subscription.List{{Channel: subscription.OrderbookChannel, QualifiedChannel: "btcusdt@depth@100ms", Asset: asset.Spot, Pairs: currency.Pairs{p}}}
			require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, subs...), "AddSuccessfulSubscriptions must not error")
			loadWebsocketBook(t, ex, p, 10)
			fetching, release := make(chan struct{}), make(chan struct{})
			releaseOnce := sync.OnceFunc(func() { close(release) })
			t.Cleanup(releaseOnce)
			ex.orderbookSync = ex.newOrderbookSync(func(ctx context.Context, _ currency.Pair, _ asset.Item) (*orderbook.Book, error) {
				close(fetching)
				select {
				case <-release:
					return &orderbook.Book{Exchange: ex.Name, Pair: p, Asset: asset.Spot, Bids: orderbook.Levels{{Price: 1, Amount: 1}}, Asks: orderbook.Levels{{Price: 2, Amount: 1}}, LastUpdated: time.Now(), LastUpdateID: 11}, nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}, 0, time.Minute)
			require.NoError(t, ex.processOrderbookUpdate(t.Context(), 11, &orderbook.Update{Pair: p, Asset: asset.Spot, UpdateID: 11, UpdateTime: time.Now(), AllowEmpty: true}), "processOrderbookUpdate must not error")
			select {
			case <-fetching:
			case <-time.After(5 * time.Second):
				require.FailNow(t, "the diff event must start a snapshot fetch")
			}
			if stop == "shutdown" {
				require.NoError(t, ex.Websocket.Shutdown(), "Shutdown must not error")
			} else {
				require.NoError(t, ex.manageSubs(t.Context(), conn, wsUnsubscribeMethod, subs), "manageSubs must not error")
			}
			releaseOnce()
			require.Eventually(t, func() bool {
				b, err := orderbook.Get(ex.Name, p, asset.Spot)
				return err == nil && b.LastUpdateID == 11
			}, 5*time.Second, time.Millisecond, "the fetched snapshot must load after the stream stopped")
			book, err := ex.UpdateOrderbook(t.Context(), p, asset.Spot)
			require.NoError(t, err, "UpdateOrderbook must not error")
			assert.Equal(t, int64(12), book.LastUpdateID, "UpdateOrderbook should replace a book no subscribed order book stream maintains")
		})
	}

	for _, stream := range []struct{ name, channel, frame string }{
		{name: "partial depth", channel: "btcusdt@depth5@100ms", frame: `{"stream":"btcusdt@depth5@100ms","data":{"lastUpdateId":11,"bids":[["1","1"]],"asks":[["2","1"]]}}`},
		{name: "diff depth", channel: "btcusdt@depth@100ms", frame: `{"stream":"btcusdt@depth@100ms","data":{"e":"depthUpdate","s":"BTCUSDT","E":1794466395184,"U":11,"u":11,"b":[["1","1"]],"a":[["2","1"]]}}`},
	} {
		t.Run("book a "+stream.name+" frame loads after an unanswered unsubscription", func(t *testing.T) {
			t.Parallel()
			ex := newWebsocketTestExchange(t)
			p := currency.NewBTCUSDT()
			handling, release := make(chan struct{}), make(chan struct{})
			handlingOnce, releaseOnce := sync.OnceFunc(func() { close(handling) }), sync.OnceFunc(func() { close(release) })
			t.Cleanup(releaseOnce)
			setup := ex.streamConnectionSetup(ex.Config)
			setup.ResponseMaxLimit = 100 * time.Millisecond
			setup.GenerateSubscriptions = func() (subscription.List, error) {
				return subscription.List{{Channel: "keepalive", QualifiedChannel: "btcusdt@trade", Asset: asset.Spot, Pairs: currency.Pairs{p}}}, nil
			}
			setup.Handler = func(ctx context.Context, conn websocket.Connection, data []byte) error {
				if strings.Contains(string(data), `"stream"`) {
					// The reader stalls on the stream's last frame, leaving the unsubscription's answer behind it unread
					handlingOnce()
					<-release
				}
				return ex.wsHandleData(ctx, conn, data)
			}
			useMockWebsocket(t, ex, setup, func(tb testing.TB, msg []byte, w *gws.Conn) error {
				tb.Helper()
				var req WebsocketPayloadRequest
				require.NoError(tb, json.Unmarshal(msg, &req), "request must decode")
				if req.Method == wsUnsubscribeMethod {
					if err := w.WriteMessage(gws.TextMessage, []byte(stream.frame)); err != nil {
						return err
					}
				}
				return streamsMock(tb, msg, w)
			})
			require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
			conn, err := ex.Websocket.GetConnection(asset.Spot)
			require.NoError(t, err, "GetConnection must not error")
			require.NoError(t, ex.SetHTTPClient(orderBookClient(12, nil)), "SetHTTPClient must not error")
			loadWebsocketBook(t, ex, p, 10)
			ex.orderbookSync = ex.newOrderbookSync(func(context.Context, currency.Pair, asset.Item) (*orderbook.Book, error) {
				return &orderbook.Book{Exchange: ex.Name, Pair: p, Asset: asset.Spot, Bids: orderbook.Levels{{Price: 1, Amount: 1}}, Asks: orderbook.Levels{{Price: 2, Amount: 1}}, LastUpdated: time.Now(), LastUpdateID: 11}, nil
			}, 0, time.Minute)
			subs := subscription.List{{Channel: subscription.OrderbookChannel, QualifiedChannel: stream.channel, Asset: asset.Spot, Pairs: currency.Pairs{p}}}
			require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, subs...), "AddSuccessfulSubscriptions must not error")
			require.ErrorIs(t, ex.manageSubs(t.Context(), conn, wsUnsubscribeMethod, subs), websocket.ErrSignatureTimeout, "manageSubs must time out behind the stalled reader")
			select {
			case <-handling:
			default:
				require.FailNow(t, "the reader must be handling the stream's last frame")
			}
			releaseOnce()
			require.Eventually(t, func() bool {
				b, err := orderbook.Get(ex.Name, p, asset.Spot)
				return err == nil && b.LastUpdateID == 11
			}, 5*time.Second, time.Millisecond, "the stalled frame must load its book after the unsubscription")
			book, err := ex.UpdateOrderbook(t.Context(), p, asset.Spot)
			require.NoError(t, err, "UpdateOrderbook must not error")
			assert.Equal(t, int64(12), book.LastUpdateID, "UpdateOrderbook should replace a book whose order book stream is unsubscribing")
		})
	}

	for _, tc := range []struct {
		stop          string
		releaseBefore bool
	}{
		{stop: "shutdown", releaseBefore: true},
		{stop: "shutdown"},
		{stop: "unsubscription", releaseBefore: true},
		{stop: "unsubscription"},
	} {
		name := "snapshot fetched across a " + tc.stop + " returning after the stream subscribes again"
		if tc.releaseBefore {
			name = "snapshot fetched across a " + tc.stop + " returning before the stream subscribes again"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ex, conn := newSubscriptionTestExchange(t)
			p := currency.NewBTCUSDT()
			fetching, release := make(chan struct{}), make(chan struct{})
			fetchingOnce, releaseOnce := sync.OnceFunc(func() { close(fetching) }), sync.OnceFunc(func() { close(release) })
			t.Cleanup(releaseOnce)
			var requests atomic.Int32
			require.NoError(t, ex.SetHTTPClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				if requests.Add(1) == 1 {
					// The diff depth synchronisation's snapshot, held until the stream has stopped
					fetchingOnce()
					<-release
					return orderBookResponse(11), nil
				}
				return orderBookResponse(12), nil
			})}), "SetHTTPClient must not error")
			streams := func() subscription.List {
				return subscription.List{{Channel: subscription.OrderbookChannel, QualifiedChannel: "btcusdt@depth@100ms", Asset: asset.Spot, Pairs: currency.Pairs{p}}}
			}
			stopped := streams()
			require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, stopped...), "AddSuccessfulSubscriptions must not error")
			loadWebsocketBook(t, ex, p, 10)
			fetched := make(chan error, 1)
			ex.orderbookSync = ex.newOrderbookSync(func(ctx context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
				book, err := ex.fetchOrderbookSnapshot(ctx, p, a)
				select {
				case fetched <- err:
				default:
				}
				return book, err
			}, 0, time.Minute)
			require.NoError(t, ex.processOrderbookUpdate(t.Context(), 11, &orderbook.Update{Pair: p, Asset: asset.Spot, UpdateID: 11, UpdateTime: time.Now(), AllowEmpty: true}), "processOrderbookUpdate must not error")
			select {
			case <-fetching:
			case <-time.After(5 * time.Second):
				require.FailNow(t, "the diff event must start a snapshot fetch")
			}
			if tc.stop == "shutdown" {
				require.NoError(t, ex.Websocket.Shutdown(), "Shutdown must not error")
			} else {
				require.NoError(t, ex.manageSubs(t.Context(), conn, wsUnsubscribeMethod, stopped), "manageSubs must unsubscribe the order book stream")
			}
			subscribeAgain := func() {
				if tc.stop == "shutdown" {
					require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
					var err error
					conn, err = ex.Websocket.GetConnection(asset.Spot)
					require.NoError(t, err, "GetConnection must not error")
				}
				require.NoError(t, ex.manageSubs(t.Context(), conn, wsSubscribeMethod, streams()), "manageSubs must subscribe the order book stream again")
			}
			if !tc.releaseBefore {
				subscribeAgain()
			}
			releaseOnce()
			select {
			case err := <-fetched:
				require.ErrorIs(t, err, errOrderbookStreamStopped, "fetchOrderbookSnapshot must discard the snapshot fetched across the stop")
			case <-time.After(5 * time.Second):
				require.FailNow(t, "the snapshot fetch must return")
			}
			if tc.releaseBefore {
				subscribeAgain()
			}
			book, err := ex.UpdateOrderbook(t.Context(), p, asset.Spot)
			require.NoError(t, err, "UpdateOrderbook must not error")
			assert.Equal(t, int64(12), book.LastUpdateID, "UpdateOrderbook should replace the book the stopped stream left")
		})
	}

	for _, tc := range []struct {
		name string
		age  time.Duration
		kept bool
	}{
		{name: "websocket book updated within websocketBookFreshness before the request", age: websocketBookFreshness - time.Second, kept: true},
		{name: "websocket book last updated longer than websocketBookFreshness before the request", age: websocketBookFreshness + time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				ex := newWebsocketTestExchange(t)
				p := currency.NewBTCUSDT()
				require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(nil, &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{p}}), "AddSuccessfulSubscriptions must not error")
				require.NoError(t, ex.SetHTTPClient(orderBookClient(42, nil)), "SetHTTPClient must not error")
				loadWebsocketBook(t, ex, p, 43)
				time.Sleep(tc.age)
				book, err := ex.UpdateOrderbook(t.Context(), p, asset.Spot)
				require.NoError(t, err, "UpdateOrderbook must not error")
				if tc.kept {
					assert.Equal(t, int64(43), book.LastUpdateID, "UpdateOrderbook should leave in place a book the websocket updated within websocketBookFreshness before the request")
					return
				}
				assert.Equal(t, int64(42), book.LastUpdateID, "UpdateOrderbook should replace a book the websocket last updated longer than websocketBookFreshness before the request")
			})
		})
	}
}

// orderBookReply returns a REST order book reply whose lastUpdateId is id
func orderBookReply(id int64) string {
	return `{"lastUpdateId":` + strconv.FormatInt(id, 10) + `,"bids":[["69999","1"]],"asks":[["70001","2"]]}`
}

// orderBookResponse returns an HTTP response holding orderBookReply(id)
func orderBookResponse(id int64) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(orderBookReply(id)))}
}

// roundTripFunc answers HTTP requests in memory
type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip answers r
func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// orderBookClient returns an HTTP client answering every request with orderBookReply(id), after calling before when it
// is not nil
func orderBookClient(id int64, before func()) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		if before != nil {
			before()
		}
		return orderBookResponse(id), nil
	})}
}

// loadWebsocketBook loads a book of p whose update ID is id, as the websocket loads one, and returns it
func loadWebsocketBook(tb testing.TB, ex *Exchange, p currency.Pair, id int64) *orderbook.Book {
	tb.Helper()
	b := &orderbook.Book{Exchange: ex.Name, Pair: p, Asset: asset.Spot, Bids: orderbook.Levels{{Price: 70000, Amount: 3}}, Asks: orderbook.Levels{{Price: 70002, Amount: 4}}, LastUpdated: time.Now(), LastUpdateID: id}
	require.NoError(tb, ex.Websocket.Orderbook.LoadSnapshot(tb.Context(), b), "LoadSnapshot must load the websocket book")
	return b
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipUnlessLive(t, true)
		_, err := e.UpdateAccountBalances(t.Context(), asset.Spot)
		assert.NoError(t, err, "UpdateAccountBalances should not error")
	})

	body := `{"updateTime":1794466000000,"balances":[{"asset":"BTC","free":"1.5","locked":"0.5"},{"asset":"ETH","free":"2","locked":"0"}]}`
	ex := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/account": reply(t, url.Values{}, body)})
	creds := configuredCredentials(t, ex)
	// The stream reported ETH after the snapshot's account update time, so the snapshot's ETH predates the stored one
	stored := accounts.NewSubAccount(asset.Spot, "")
	stored.Balances.Set(currency.ETH, accounts.Balance{Total: 9, Free: 9})
	require.NoError(t, ex.Accounts.Save(t.Context(), accounts.SubAccounts{stored}, true), "Save must not error")
	ex.recordBalanceEvent(time.UnixMilli(1794466000001), currency.ETH)

	subAccounts, err := ex.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	require.Len(t, subAccounts, 1, "UpdateAccountBalances must return one sub-account")
	btc := subAccounts[0].Balances[currency.BTC]
	assert.Equal(t, [3]float64{2, 1.5, 0.5}, [3]float64{btc.Total, btc.Free, btc.Hold}, "UpdateAccountBalances should convert the snapshot's total, free and held balances")
	eth, err := ex.Accounts.GetBalance("", creds, asset.Spot, currency.ETH)
	require.NoError(t, err, "GetBalance must not error")
	assert.Equal(t, 9.0, eth.Total, "a currency streamed after the snapshot should keep its stored balance")
	assert.Equal(t, time.UnixMilli(1794466000000), ex.balanceSnapshotTime(), "UpdateAccountBalances should record the snapshot's account update time")

	stale := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/account": reply(t, url.Values{}, `{"updateTime":1794465000000,"balances":[{"asset":"BTC","free":"1","locked":"0"}]}`)})
	newer := accounts.NewSubAccount(asset.Spot, "")
	newer.Balances.Set(currency.BTC, accounts.Balance{Total: 2, Free: 2})
	require.NoError(t, stale.Accounts.Save(t.Context(), accounts.SubAccounts{newer}, true), "Save must not error")
	stale.recordBalanceSnapshot(time.UnixMilli(1794466000000))
	subAccounts, err = stale.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error for a snapshot older than the stored one")
	require.Len(t, subAccounts, 1, "UpdateAccountBalances must return the stored sub-account")
	assert.Equal(t, 2.0, subAccounts[0].Balances[currency.BTC].Total, "UpdateAccountBalances should return the stored balances in place of an older snapshot's")
	btcStored, err := stale.Accounts.GetBalance("", configuredCredentials(t, stale), asset.Spot, currency.BTC)
	require.NoError(t, err, "GetBalance must not error")
	assert.Equal(t, 2.0, btcStored.Total, "an older snapshot should not replace the stored balances")
	assert.Equal(t, time.UnixMilli(1794466000000), stale.balanceSnapshotTime(), "an older snapshot should leave the stored snapshot's time")
	_, err = ex.UpdateAccountBalances(t.Context(), asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "UpdateAccountBalances should reject an asset other than spot")
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipUnlessLive(t, false)
		trades, err := e.GetRecentTrades(t.Context(), spotTradablePair, asset.Spot)
		require.NoError(t, err, "GetRecentTrades must not error")
		assert.NotEmpty(t, trades, "GetRecentTrades should return the pair's trades")
	})

	body := `[{"id":2,"price":"70001","qty":"0.2","time":1794466395185,"isBuyerMaker":true},{"id":1,"price":"70000","qty":"0.1","time":1794466395184,"isBuyerMaker":false}]`
	ex := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/trades": reply(t, url.Values{"symbol": {"BTCUSDT"}, "limit": {"1000"}}, body)})
	trades, err := ex.GetRecentTrades(t.Context(), currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err, "GetRecentTrades must not error")
	require.Len(t, trades, 2, "GetRecentTrades must return every trade")
	assert.Equal(t, trade.Data{
		TID: "1", Exchange: ex.Name, CurrencyPair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Buy,
		Price: 70000, Amount: 0.1, Timestamp: time.UnixMilli(1794466395184),
	}, trades[0], "GetRecentTrades should convert the trades sorted by time")
	assert.Equal(t, order.Sell, trades[1].Side, "a trade whose buyer made the market should be a sell")
	_, err = ex.GetRecentTrades(t.Context(), currency.NewBTCUSDT(), asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetRecentTrades should reject an asset other than spot")
}

func TestTradeTakerSide(t *testing.T) {
	t.Parallel()
	assert.Equal(t, order.Sell, tradeTakerSide(true), "the taker should sell when the buyer made the market")
	assert.Equal(t, order.Buy, tradeTakerSide(false), "the taker should buy when the seller made the market")
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipUnlessLive(t, false)
		_, err := e.GetHistoricTrades(t.Context(), spotTradablePair, asset.Spot, time.Now().Add(-15*time.Minute), time.Now())
		assert.NoError(t, err, "GetHistoricTrades should not error")
	})

	start, end := time.UnixMilli(1794466000000), time.UnixMilli(1794466400000)
	var requests atomic.Int32
	ex := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/aggTrades": func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if requests.Add(1) == 1 {
			assert.Equal(t, url.Values{"symbol": {"BTCUSDT"}, "startTime": {"1794466000000"}, "endTime": {"1794466400000"}, "limit": {"1000"}}, q, "the first page should cover the window")
			writeResponse(t, w, `[{"a":1,"p":"70000","q":"0.1","T":1794466100000,"m":true},{"a":2,"p":"70001","q":"0.2","T":1794466200000,"m":false}]`)
			return
		}
		assert.Equal(t, url.Values{"symbol": {"BTCUSDT"}, "fromId": {"2"}, "limit": {"1000"}}, q, "a later page should start at the last trade")
		writeResponse(t, w, `[{"a":2,"p":"70001","q":"0.2","T":1794466200000,"m":false},{"a":3,"p":"70002","q":"0.3","T":1794466500000,"m":false}]`)
	}})
	trades, err := ex.GetHistoricTrades(t.Context(), currency.NewBTCUSDT(), asset.Spot, start, end)
	require.NoError(t, err, "GetHistoricTrades must not error")
	require.Len(t, trades, 2, "GetHistoricTrades must return the window's trades once each")
	assert.Equal(t, trade.Data{
		TID: "1", Exchange: ex.Name, CurrencyPair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Sell,
		Price: 70000, Amount: 0.1, Timestamp: time.UnixMilli(1794466100000),
	}, trades[0], "GetHistoricTrades should convert the aggregate trades")
	_, err = ex.GetHistoricTrades(t.Context(), currency.NewBTCUSDT(), asset.Spot, end, start)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd, "GetHistoricTrades should reject a reversed window")
	_, err = ex.GetHistoricTrades(t.Context(), currency.NewBTCUSDT(), asset.Futures, start, end)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricTrades should reject an asset other than spot")
}

func TestCanUseWebsocketAPIForOrders(t *testing.T) {
	t.Parallel()
	assert.False(t, newWebsocketTestExchange(t).canUseWebsocketAPIForOrders(), "orders should not go over a disconnected WebSocket API")
	ex := newWebsocketAPITestExchange(t, false)
	assert.False(t, ex.canUseWebsocketAPIForOrders(), "orders should not go over the WebSocket API without authenticated websocket use")
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	assert.True(t, ex.canUseWebsocketAPIForOrders(), "orders should go over a connected WebSocket API with authenticated websocket use")
	ex.websocketAPIKeyRejected.Store(true)
	assert.False(t, ex.canUseWebsocketAPIForOrders(), "orders should not go over the WebSocket API once it refused the configured account's key")
}

func TestOverWebsocketAPI(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, false)
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	assert.True(t, ex.overWebsocketAPI(true, false), "a request naming an order ID alone should go over the WebSocket API")
	assert.True(t, ex.overWebsocketAPI(false, true), "a request naming a client order ID alone should go over the WebSocket API")
	assert.False(t, ex.overWebsocketAPI(true, true), "a request naming both IDs should go over REST, which checks both")
}

func TestLogOrderIssue(t *testing.T) {
	t.Parallel()
	assert.NotPanics(t, func() { new(Exchange).logOrderIssue("1", errUnknownOrderStatus) }, "logOrderIssue should log the issue")
}

func TestOrderSideString(t *testing.T) {
	t.Parallel()
	for side, exp := range map[order.Side]Side{order.Buy: SideBuy, order.Bid: SideBuy, order.Sell: SideSell, order.Ask: SideSell} {
		got, err := orderSideString(side)
		require.NoErrorf(t, err, "orderSideString must not error for %s", side)
		assert.Equalf(t, exp, got, "orderSideString should convert %s", side)
	}
	_, err := orderSideString(order.UnknownSide)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "orderSideString should reject an unknown side")
}

func TestResolveTriggerOrderType(t *testing.T) {
	t.Parallel()
	assert.Equal(t, order.StopLimit, resolveTriggerOrderType(order.Stop, 1), "a priced stop should be a stop limit order")
	assert.Equal(t, order.StopMarket, resolveTriggerOrderType(order.Stop, 0), "an unpriced stop should be a stop market order")
	assert.Equal(t, order.TakeProfitLimit, resolveTriggerOrderType(order.TakeProfit, 1), "a priced take profit should be a take profit limit order")
	assert.Equal(t, order.TakeProfitMarket, resolveTriggerOrderType(order.TakeProfit, 0), "an unpriced take profit should be a take profit market order")
	assert.Equal(t, order.Limit, resolveTriggerOrderType(order.Limit, 1), "another type should be kept")
}

func TestSubmittedOrderType(t *testing.T) {
	t.Parallel()
	assert.Equal(t, order.LimitMaker, submittedOrderType(&order.Submit{Type: order.Limit, TimeInForce: order.PostOnly}), "a post only limit order should be placed as LIMIT_MAKER")
	assert.Equal(t, order.StopLimit, submittedOrderType(&order.Submit{Type: order.Stop, Price: 1}), "a bare stop should resolve to its placed form")
	assert.Equal(t, order.Market, submittedOrderType(&order.Submit{Type: order.Market}), "another type should be kept")
}

func TestTimeInForceString(t *testing.T) {
	t.Parallel()
	for tif, exp := range map[order.TimeInForce]TimeInForce{
		order.UnknownTIF: TimeInForceGoodTillCancel, order.GoodTillCancel: TimeInForceGoodTillCancel,
		order.ImmediateOrCancel: TimeInForceImmediateOrCancel, order.FillOrKill: TimeInForceFillOrKill,
	} {
		got, err := timeInForceString(tif)
		require.NoErrorf(t, err, "timeInForceString must not error for %s", tif)
		assert.Equalf(t, exp, got, "timeInForceString should convert %s", tif)
	}
	_, err := timeInForceString(order.GoodTillDay)
	assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "timeInForceString should reject an unsupported time in force")
}

func TestNewSpotOrderValues(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		s    order.Submit
		exp  *spotOrderValues
		err  error
	}{
		{name: "limit", s: order.Submit{Side: order.Buy, Type: order.Limit, Price: 1, Amount: 2}, exp: &spotOrderValues{side: SideBuy, orderType: OrderTypeLimit, timeInForce: TimeInForceGoodTillCancel, price: 1, quantity: 2}},
		{name: "limit maker", s: order.Submit{Side: order.Sell, Type: order.LimitMaker, Price: 1, Amount: 2}, exp: &spotOrderValues{side: SideSell, orderType: OrderTypeLimitMaker, price: 1, quantity: 2}},
		{name: "quote market", s: order.Submit{Side: order.Sell, Type: order.Market, QuoteAmount: 5}, exp: &spotOrderValues{side: SideSell, orderType: OrderTypeMarket, quoteOrderQuantity: 5}},
		{name: "stop limit", s: order.Submit{Side: order.Buy, Type: order.StopLimit, Price: 1, TriggerPrice: 2, Amount: 3, TimeInForce: order.FillOrKill}, exp: &spotOrderValues{side: SideBuy, orderType: OrderTypeStopLossLimit, timeInForce: TimeInForceFillOrKill, price: 1, stopPrice: 2, quantity: 3}},
		{name: "take profit", s: order.Submit{Side: order.Buy, Type: order.TakeProfitMarket, TriggerPrice: 2, Amount: 3}, exp: &spotOrderValues{side: SideBuy, orderType: OrderTypeTakeProfit, stopPrice: 2, quantity: 3}},
		{name: "no side", s: order.Submit{Type: order.Limit, Price: 1, Amount: 2}, err: order.ErrSideIsInvalid},
		{name: "unsupported type", s: order.Submit{Side: order.Buy, Type: order.TrailingStop, Amount: 2}, err: order.ErrUnsupportedOrderType},
		{name: "market time in force", s: order.Submit{Side: order.Buy, Type: order.Market, Amount: 2, TimeInForce: order.FillOrKill}, err: order.ErrUnsupportedTimeInForce},
		{name: "limit time in force", s: order.Submit{Side: order.Buy, Type: order.Limit, Price: 1, Amount: 2, TimeInForce: order.GoodTillDay}, err: order.ErrUnsupportedTimeInForce},
		{name: "unpriced limit", s: order.Submit{Side: order.Buy, Type: order.Limit, Amount: 2}, err: order.ErrPriceMustBeSetIfLimitOrder},
		{name: "untriggered stop", s: order.Submit{Side: order.Buy, Type: order.StopMarket, Amount: 2}, err: errTriggerPriceRequired},
		{name: "no amount", s: order.Submit{Side: order.Buy, Type: order.Limit, Price: 1}, err: order.ErrAmountIsInvalid},
	} {
		got, err := newSpotOrderValues(&tc.s)
		require.ErrorIsf(t, err, tc.err, "newSpotOrderValues must return the expected error for %s", tc.name)
		assert.Equalf(t, tc.exp, got, "newSpotOrderValues should work out the values of %s", tc.name)
	}
}

func TestCheckUntimedSpotTimeInForce(t *testing.T) {
	t.Parallel()
	assert.NoError(t, checkUntimedSpotTimeInForce(OrderTypeLimit, order.FillOrKill), "a timed type should take any time in force")
	assert.NoError(t, checkUntimedSpotTimeInForce(OrderTypeLimitMaker, order.PostOnly), "a LIMIT_MAKER order should take post only")
	assert.NoError(t, checkUntimedSpotTimeInForce(OrderTypeMarket, order.ImmediateOrCancel), "a MARKET order should take IOC")
	assert.ErrorIs(t, checkUntimedSpotTimeInForce(OrderTypeStopLoss, order.ImmediateOrCancel), order.ErrUnsupportedTimeInForce, "a STOP_LOSS order should refuse IOC")
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipUnlessLive(t, true, canManipulateRealOrders)
		require.NoError(t, e.UpdateOrderExecutionLimits(t.Context(), asset.Spot), "UpdateOrderExecutionLimits must not error")
		l, err := e.GetOrderExecutionLimits(asset.Spot, spotTradablePair)
		require.NoError(t, err, "GetOrderExecutionLimits must not error")
		tick, err := e.UpdateTicker(t.Context(), spotTradablePair, asset.Spot)
		require.NoError(t, err, "UpdateTicker must not error")
		// A post only buy at half the last price rests on the book until it is cancelled
		price := l.FloorPriceToStepIncrement(tick.Last / 2)
		require.Positive(t, price, "the order's price must be positive")
		amount := l.FloorAmountToStepIncrement(max(l.MinimumBaseAmount, l.MinNotional/price) * 2)
		resp, err := e.SubmitOrder(t.Context(), &order.Submit{
			Exchange:    e.Name,
			Pair:        spotTradablePair,
			AssetType:   asset.Spot,
			Side:        order.Buy,
			Type:        order.Limit,
			TimeInForce: order.PostOnly,
			Price:       price,
			Amount:      amount,
		})
		require.NoError(t, err, "SubmitOrder must not error")
		t.Cleanup(func() {
			ctx := context.WithoutCancel(t.Context())
			require.NoError(t, e.CancelOrder(ctx, &order.Cancel{OrderID: resp.OrderID, Pair: spotTradablePair, AssetType: asset.Spot}), "CancelOrder must cancel the live order")
			detail, err := e.GetOrderInfo(ctx, resp.OrderID, spotTradablePair, asset.Spot)
			require.NoError(t, err, "GetOrderInfo must not error")
			assert.Equal(t, order.Cancelled, detail.Status, "the live order should read back cancelled")
		})
		assert.Equal(t, order.New, resp.Status, "SubmitOrder should report the resting order as new")
	})

	placed := `{"symbol":"BTCUSDT","orderId":28,"clientOrderId":"gct-1","transactTime":1794466395184,"price":"70000","origQty":"0.5","executedQty":"0","cummulativeQuoteQty":"0","status":"NEW","timeInForce":"GTC","type":"LIMIT_MAKER","side":"BUY"}`
	ex := routedTestExchange(t, map[string]http.HandlerFunc{"POST /api/v3/order": reply(t, url.Values{
		"symbol": {"BTCUSDT"}, "side": {"BUY"}, "type": {"LIMIT_MAKER"}, "price": {"70000"}, "quantity": {"0.5"},
		"newClientOrderId": {"gct-1"}, "newOrderRespType": {"FULL"},
	}, placed)})
	resp, err := ex.SubmitOrder(t.Context(), &order.Submit{
		Exchange: ex.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Buy, Type: order.Limit,
		TimeInForce: order.PostOnly, Price: 70000, Amount: 0.5, ClientOrderID: "gct-1",
	})
	require.NoError(t, err, "SubmitOrder must not error")
	assert.Equal(t, "28", resp.OrderID, "SubmitOrder should return the order ID")
	assert.Equal(t, ex.Name, resp.Exchange, "SubmitOrder should report the exchange")
	assert.True(t, resp.Pair.Equal(currency.NewBTCUSDT()), "SubmitOrder should report the pair")
	assert.Equal(t, asset.Spot, resp.AssetType, "SubmitOrder should report the asset")
	assert.Equal(t, order.Buy, resp.Side, "SubmitOrder should report the side")
	assert.Equal(t, order.LimitMaker, resp.Type, "SubmitOrder should report the type placed")
	assert.Equal(t, order.PostOnly, resp.TimeInForce, "SubmitOrder should report a LIMIT_MAKER order as post only")
	assert.Equal(t, order.New, resp.Status, "SubmitOrder should report the order's status")

	_, err = ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.Futures, Side: order.Buy, Type: order.Market, Amount: 1})
	assert.ErrorIs(t, err, asset.ErrNotSupported, "SubmitOrder should reject an asset other than spot")
	_, err = ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Buy, Type: order.Market, Amount: 1, Leverage: 2})
	assert.ErrorIs(t, err, order.ErrSubmitLeverageNotSupported, "SubmitOrder should reject leverage")
	_, err = ex.SubmitOrder(t.Context(), nil)
	assert.ErrorIs(t, err, order.ErrSubmissionIsNil, "SubmitOrder should reject a nil submission")
}

func TestSubmitSpotOrder(t *testing.T) {
	t.Parallel()
	ex := routedTestExchange(t, map[string]http.HandlerFunc{"POST /api/v3/order": reply(t, url.Values{
		"symbol": {"BTCUSDT"}, "side": {"SELL"}, "type": {"MARKET"}, "quoteOrderQty": {"25"}, "newOrderRespType": {"FULL"},
	}, `{"orderId":29,"transactTime":1794466395184,"origQty":"0","executedQty":"0.0003","cummulativeQuoteQty":"25","status":"FILLED"}`)})
	resp, err := ex.submitSpotOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Sell, Type: order.Market, QuoteAmount: 25})
	require.NoError(t, err, "submitSpotOrder must not error")
	assert.Equal(t, order.Filled, resp.Status, "a quote sized market order should keep Binance.US's status")
	_, err = ex.submitSpotOrder(t.Context(), &order.Submit{Side: order.Sell, Type: order.Market})
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid, "submitSpotOrder should reject an order without an amount")
}

func TestSpotSubmitResponse(t *testing.T) {
	t.Parallel()
	ex := &Exchange{Name: "test"}
	s := &order.Submit{Exchange: "test", Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Price: 70000, Amount: 1}
	resp, err := ex.spotSubmitResponse(s, &NewOrderResponse{
		OrderID: 30, ClientOrderID: "c", TransactTime: types.Time(time.UnixMilli(1794466395184)), OriginalQuantity: 1,
		ExecutedQuantity: 0.4, CumulativeQuoteQuantity: 28000, Status: "EXPIRED",
		Fills: []OrderFill{
			{Price: 70000, Quantity: 0.3, Commission: 0.0003, CommissionAsset: currency.BTC, TradeID: 77},
			{Price: 70000, Quantity: 0.1, Commission: 0.0001, CommissionAsset: currency.BTC, TradeID: 78},
		},
	})
	require.NoError(t, err, "spotSubmitResponse must not error")
	assert.Equal(t, order.New, resp.Status, "an order finished after a partial execution should be reported New for the order manager to settle")
	assert.Equal(t, 0.6, resp.RemainingAmount, "spotSubmitResponse should report the remaining amount")
	assert.Equal(t, 28000.0, resp.Cost, "spotSubmitResponse should report the cost")
	assert.Equal(t, 70000.0, resp.AverageExecutedPrice, "spotSubmitResponse should report the average price")
	exp := []order.TradeHistory{
		{Price: 70000, Amount: 0.3, Fee: 0.0003, Exchange: "test", TID: "77", Type: order.Limit, Side: order.Buy, Timestamp: time.UnixMilli(1794466395184), FeeAsset: "BTC"},
		{Price: 70000, Amount: 0.1, Fee: 0.0001, Exchange: "test", TID: "78", Type: order.Limit, Side: order.Buy, Timestamp: time.UnixMilli(1794466395184), FeeAsset: "BTC"},
	}
	assert.Equal(t, exp, resp.Trades, "spotSubmitResponse should report each fill with its trade ID")
	d := &order.Detail{Exchange: "test", OrderID: resp.OrderID, Trades: resp.Trades}
	cp := d.Copy()
	require.NoError(t, d.UpdateOrderFromDetail(&cp), "UpdateOrderFromDetail must not error")
	assert.Equal(t, exp, d.Trades, "fills should stay distinct when the order is merged with a copy of itself")

	resp, err = ex.spotSubmitResponse(s, &NewOrderResponse{OrderID: 31, Status: "UNKNOWN"})
	require.NoError(t, err, "spotSubmitResponse must not error for an unknown status")
	assert.Equal(t, order.UnknownStatus, resp.Status, "an unknown status should be left unset")
}

func TestNewOneCancelsTheOtherOrderRequest(t *testing.T) {
	t.Parallel()
	s := order.Submit{Pair: currency.NewBTCUSDT(), Side: order.Sell, Type: order.OCO, Amount: 0.002, Price: 80000, TriggerPrice: 70000, ClientOrderID: "gct-oco-1"}
	req, err := newOneCancelsTheOtherOrderRequest(&s)
	require.NoError(t, err, "newOneCancelsTheOtherOrderRequest must not error")
	assert.Equal(t, &OneCancelsTheOtherOrderRequest{Symbol: currency.NewBTCUSDT(), ListClientOrderID: "gct-oco-1", Side: SideSell, Quantity: 0.002, Price: 80000, StopPrice: 70000}, req, "newOneCancelsTheOtherOrderRequest should work out a stop loss list")

	withLimit := s
	withLimit.RiskManagementModes.StopLoss.LimitPrice = 69900
	withLimit.TimeInForce = order.PostOnly | order.ImmediateOrCancel
	req, err = newOneCancelsTheOtherOrderRequest(&withLimit)
	require.NoError(t, err, "newOneCancelsTheOtherOrderRequest must not error with a stop limit price")
	assert.Equal(t, 69900.0, req.StopLimitPrice, "a stop limit price should make the stop order a stop limit order")
	assert.Equal(t, TimeInForceImmediateOrCancel, req.StopLimitTimeInForce, "the time in force should be the stop limit order's")

	for name, modify := range map[string]func(*order.Submit){
		"no side":          func(s *order.Submit) { s.Side = order.UnknownSide },
		"no price":         func(s *order.Submit) { s.Price = 0 },
		"no trigger price": func(s *order.Submit) { s.TriggerPrice = 0 },
		"time in force":    func(s *order.Submit) { s.TimeInForce = order.FillOrKill },
	} {
		invalid := s
		modify(&invalid)
		_, err := newOneCancelsTheOtherOrderRequest(&invalid)
		assert.Errorf(t, err, "newOneCancelsTheOtherOrderRequest should reject a list with %s", name)
	}
}

func TestSubmitOneCancelsTheOtherOrder(t *testing.T) {
	t.Parallel()
	ex := routedTestExchange(t, map[string]http.HandlerFunc{"POST /api/v3/order/oco": reply(t, url.Values{
		"symbol": {"BTCUSDT"}, "side": {"SELL"}, "quantity": {"0.002"}, "price": {"80000"}, "stopPrice": {"70000"}, "listClientOrderId": {"gct-oco-1"},
	}, `{"orderListId":9,"contingencyType":"OCO","listOrderStatus":"EXECUTING","listClientOrderId":"gct-oco-1","transactionTime":1794466395184,"symbol":"BTCUSDT"}`)})
	resp, err := ex.SubmitOrder(t.Context(), &order.Submit{
		Exchange: ex.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Sell, Type: order.OCO,
		Amount: 0.002, Price: 80000, TriggerPrice: 70000, ClientOrderID: "gct-oco-1",
	})
	require.NoError(t, err, "SubmitOrder must not error for an OCO order list")
	assert.Equal(t, "list-9", resp.OrderID, "an OCO order list should be identified by its tagged order list ID")
	assert.Equal(t, order.Active, resp.Status, "an executing order list should be active")
	_, err = ex.submitOneCancelsTheOtherOrder(t.Context(), &order.Submit{Side: order.Sell})
	assert.ErrorIs(t, err, order.ErrPriceMustBeSetIfLimitOrder, "submitOneCancelsTheOtherOrder should reject an unpriced list")

	unreadable := routedTestExchange(t, map[string]http.HandlerFunc{"POST /api/v3/order/oco": reply(t, nil, `{"orderListId":9,"contingencyType":"OCO","listOrderStatus":"UNKNOWN","symbol":"BTCUSDT"}`)})
	resp, err = unreadable.SubmitOrder(t.Context(), &order.Submit{
		Exchange: unreadable.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Sell, Type: order.OCO,
		Amount: 0.002, Price: 80000, TriggerPrice: 70000,
	})
	require.NoError(t, err, "SubmitOrder must not report a placed order list as failed for a status it cannot read")
	assert.Equal(t, "list-9", resp.OrderID, "SubmitOrder should return the placed order list")
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).ModifyOrder(t.Context(), &order.Modify{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "ModifyOrder should not be supported")
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipUnlessLive(t, true)
		err := e.CancelOrder(t.Context(), &order.Cancel{OrderID: "1", Pair: spotTradablePair, AssetType: asset.Spot})
		assert.ErrorIs(t, err, order.ErrOrderNotFound, "CancelOrder should report an order the account does not have as not found")
	})

	listBody := `{"orderListId":9,"contingencyType":"OCO","listOrderStatus":"ALL_DONE","listClientOrderId":"L","symbol":"BTCUSDT"}`
	ex := routedTestExchange(t, map[string]http.HandlerFunc{
		"DELETE /api/v3/orderList": reply(t, url.Values{"symbol": {"BTCUSDT"}, "orderListId": {"9"}}, listBody),
		"GET /api/v3/orderList":    replyError(t, -2011, "Order list does not exist."),
		"DELETE /api/v3/order":     reply(t, url.Values{"symbol": {"BTCUSDT"}, "orderId": {"5"}}, `{"orderId":5,"status":"CANCELED"}`),
	})
	cancel := order.Cancel{Exchange: ex.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.Spot}
	listCancel := cancel
	listCancel.OrderID = "list-9"
	require.NoError(t, ex.CancelOrder(t.Context(), &listCancel), "CancelOrder must cancel an order list by its tagged ID")
	untyped := cancel
	untyped.OrderID = "5"
	require.NoError(t, ex.CancelOrder(t.Context(), &untyped), "CancelOrder must cancel an untyped request's order when no order list has its ID")

	noID := cancel
	assert.ErrorIs(t, ex.CancelOrder(t.Context(), &noID), order.ErrOrderIDNotSet, "CancelOrder should reject a request without an ID")
	futures := cancel
	futures.AssetType = asset.Futures
	futures.OrderID = "5"
	assert.ErrorIs(t, ex.CancelOrder(t.Context(), &futures), asset.ErrNotSupported, "CancelOrder should reject an asset other than spot")
	invalid := cancel
	invalid.OrderID = "x"
	assert.ErrorIs(t, ex.CancelOrder(t.Context(), &invalid), errInvalidOrderID, "CancelOrder should reject an invalid order ID")

	zero := routedTestExchange(t, map[string]http.HandlerFunc{
		"DELETE /api/v3/orderList": reply(t, url.Values{"symbol": {"BTCUSDT"}, "orderListId": {"0"}}, `{"orderListId":0,"contingencyType":"OCO","listOrderStatus":"ALL_DONE","symbol":"BTCUSDT"}`),
	})
	listZero := cancel
	listZero.OrderID = formatOrderListID(0)
	require.NoError(t, zero.CancelOrder(t.Context(), &listZero), "CancelOrder must cancel the order list SubmitOrder returns as list-0")

	gone := routedTestExchange(t, map[string]http.HandlerFunc{"DELETE /api/v3/order": replyError(t, -2011, "Unknown order sent.")})
	typed := cancel
	typed.OrderID, typed.Type = "5", order.Limit
	assert.ErrorIs(t, gone.CancelOrder(t.Context(), &typed), order.ErrOrderNotFound, "CancelOrder should report an unknown order as not found")

	ocoTyped := cancel
	ocoTyped.OrderID, ocoTyped.Type = "9", order.OCO
	require.NoError(t, ex.CancelOrder(t.Context(), &ocoTyped), "CancelOrder must cancel an OCO order list by its untagged ID")

	listOnly := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /api/v3/orderList":    reply(t, url.Values{"orderListId": {"9"}}, listBody),
		"GET /api/v3/order":        replyError(t, -2013, "Order does not exist."),
		"DELETE /api/v3/orderList": reply(t, url.Values{"symbol": {"BTCUSDT"}, "orderListId": {"9"}}, listBody),
	})
	untypedList := cancel
	untypedList.OrderID = "9"
	require.NoError(t, listOnly.CancelOrder(t.Context(), &untypedList), "CancelOrder must cancel the order list an untyped request's ID names")

	listCancelFails := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /api/v3/orderList":    reply(t, nil, listBody),
		"GET /api/v3/order":        replyError(t, -2013, "Order does not exist."),
		"DELETE /api/v3/orderList": replyError(t, -1000, "An unknown error occurred while processing the request."),
	})
	err := listCancelFails.CancelOrder(t.Context(), &untypedList)
	assert.ErrorIs(t, err, errAPIResponse, "CancelOrder should return the order list cancellation's error")
	assert.ErrorContains(t, err, "error cancelling order list", "CancelOrder should say the order list was not cancelled")

	lookupFails := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/orderList": replyError(t, -1000, "An unknown error occurred while processing the request.")})
	assert.ErrorIs(t, lookupFails.CancelOrder(t.Context(), &untypedList), errAPIResponse, "CancelOrder should return the order list lookup's error")
}

func TestIsUntypedRequest(t *testing.T) {
	t.Parallel()
	assert.True(t, isUntypedRequest(order.UnknownType), "an unknown type should be untyped")
	assert.True(t, isUntypedRequest(order.AnyType), "any type should be untyped")
	assert.False(t, isUntypedRequest(order.Limit), "a limit type should be typed")
}

func TestIsUnknownOrder(t *testing.T) {
	t.Parallel()
	assert.True(t, isUnknownOrder(order.ErrOrderNotFound), "order.ErrOrderNotFound should be an unknown order")
	assert.True(t, isUnknownOrder(&APIError{Code: -2011, Message: "Unknown order sent."}), "-2011 Unknown order should be an unknown order")
	assert.True(t, isUnknownOrder(&APIError{Code: -2011, Message: "Order list does not exist."}), "-2011 for an order list should be an unknown order")
	assert.False(t, isUnknownOrder(&APIError{Code: -2011, Message: "Something else"}), "another -2011 message should not be an unknown order")
	assert.False(t, isUnknownOrder(errAPIResponse), "another error should not be an unknown order")
}

func TestCancelError(t *testing.T) {
	t.Parallel()
	assert.NoError(t, cancelError(nil), "cancelError should keep a nil error")
	assert.ErrorIs(t, cancelError(&APIError{Code: -2011, Message: "Unknown order sent."}), order.ErrOrderNotFound, "cancelError should wrap an unknown order as not found")
	assert.Equal(t, errAPIResponse, cancelError(errAPIResponse), "cancelError should keep another error")
}

func TestUntaggedOrderList(t *testing.T) {
	t.Parallel()
	listBody := `{"orderListId":5,"contingencyType":"OCO","listOrderStatus":"EXECUTING","listClientOrderId":"L","symbol":"BTCUSDT"}`
	ex := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /api/v3/orderList": reply(t, url.Values{"orderListId": {"5"}}, listBody),
		"GET /api/v3/order":     replyError(t, -2013, "Order does not exist."),
	})
	found, err := ex.untaggedOrderList(t.Context(), currency.NewBTCUSDT(), 5, "")
	require.NoError(t, err, "untaggedOrderList must not error")
	assert.Equal(t, &foundOrderList{orderListID: 5}, found, "untaggedOrderList should find the order list with the ID")

	ambiguous := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /api/v3/orderList": reply(t, nil, listBody),
		"GET /api/v3/order":     reply(t, nil, `{"orderId":5,"symbol":"BTCUSDT","status":"NEW"}`),
	})
	_, err = ambiguous.untaggedOrderList(t.Context(), currency.NewBTCUSDT(), 5, "")
	assert.ErrorIs(t, err, errAmbiguousOrderListID, "untaggedOrderList should reject an ID naming both an order and an order list")

	none := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/orderList": replyError(t, -2011, "Order list does not exist.")})
	found, err = none.untaggedOrderList(t.Context(), currency.NewBTCUSDT(), 5, "")
	require.NoError(t, err, "untaggedOrderList must not error without an order list")
	assert.Nil(t, found, "untaggedOrderList should find nothing without an order list")

	byClientID := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /api/v3/orderList": reply(t, url.Values{"origClientOrderId": {"L"}}, listBody),
		"GET /api/v3/order":     reply(t, url.Values{"symbol": {"BTCUSDT"}, "origClientOrderId": {"L"}}, `{"orderId":7,"symbol":"BTCUSDT","status":"NEW"}`),
	})
	_, err = byClientID.untaggedOrderList(t.Context(), currency.NewBTCUSDT(), 0, "L")
	assert.ErrorIs(t, err, errAmbiguousOrderListID, "untaggedOrderList should reject a client order ID naming both an order and an order list")
	assert.ErrorContains(t, err, "name the order by its order ID", "untaggedOrderList should name the order ID as what tells a client order ID's order apart")

	orderLookupFails := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /api/v3/orderList": reply(t, nil, listBody),
		"GET /api/v3/order":     replyError(t, -1000, "An unknown error occurred while processing the request."),
	})
	_, err = orderLookupFails.untaggedOrderList(t.Context(), currency.NewBTCUSDT(), 5, "")
	assert.ErrorIs(t, err, errAPIResponse, "untaggedOrderList should return the order lookup's error")
	assert.ErrorContains(t, err, "error fetching order", "untaggedOrderList should say the order lookup failed")

	listLookupFails := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/orderList": replyError(t, -1000, "An unknown error occurred while processing the request.")})
	_, err = listLookupFails.untaggedOrderList(t.Context(), currency.NewBTCUSDT(), 5, "")
	assert.ErrorIs(t, err, errAPIResponse, "untaggedOrderList should return the order list lookup's error")
}

func TestUntaggedID(t *testing.T) {
	t.Parallel()
	assert.Nil(t, untaggedID(0), "an empty order ID should name no order list")
	assert.Equal(t, new(uint64(5)), untaggedID(5), "an untagged order ID should name the order list with that ID")
}

func TestIsStaleSpotSnapshot(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	ex.recordBalanceSnapshot(time.UnixMilli(2000))
	assert.True(t, ex.isStaleSpotSnapshot(t.Context(), time.UnixMilli(1000)), "a snapshot older than the stored one should be stale")
	assert.False(t, ex.isStaleSpotSnapshot(t.Context(), time.UnixMilli(2000)), "a snapshot as new as the stored one should not be stale")
	assert.False(t, ex.isStaleSpotSnapshot(t.Context(), time.Time{}), "a snapshot without a time should not be stale")
	other := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "other-api-key", Secret: "other-api-secret"})
	assert.False(t, ex.isStaleSpotSnapshot(other, time.UnixMilli(1000)), "another account's snapshot should not be ordered")
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).CancelBatchOrders(t.Context(), []order.Cancel{{}})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "CancelBatchOrders should not be supported")
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipUnlessLive(t, true, canManipulateRealOrders)
		_, err := e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: spotTradablePair})
		assert.NoError(t, err, "CancelAllOrders should not error")
	})

	ex := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /api/v3/openOrders": reply(t, url.Values{}, `[{"symbol":"BTCUSDT","orderId":1},{"symbol":"BTCUSDT","orderId":2},{"symbol":"ETHUSDT","orderId":3}]`),
		"DELETE /api/v3/openOrders": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("symbol") == "ETHUSDT" {
				w.WriteHeader(http.StatusBadRequest)
				writeResponse(t, w, `{"code":-2011,"msg":"Unknown order sent."}`)
				return
			}
			writeResponse(t, w, `[{"orderId":1,"status":"CANCELED"},{"orderListId":7,"contingencyType":"OCO","orderReports":[{"orderId":4,"status":"CANCELED"},{"orderId":5,"status":"EXPIRED"}]}]`)
		},
	})
	resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{Exchange: ex.Name, AssetType: asset.Spot})
	require.NoError(t, err, "CancelAllOrders must not error when a pair has no orders left")
	assert.Equal(t, map[string]string{"1": "CANCELED", "4": "CANCELED", "5": "EXPIRED"}, resp.Status, "CancelAllOrders should report each cancelled order's status")
	_, err = ex.CancelAllOrders(t.Context(), &order.Cancel{Exchange: ex.Name, AssetType: asset.Futures})
	assert.ErrorIs(t, err, asset.ErrNotSupported, "CancelAllOrders should reject an asset other than spot")

	failing := routedTestExchange(t, map[string]http.HandlerFunc{"DELETE /api/v3/openOrders": replyError(t, -1021, "Timestamp for this request is outside of the recvWindow.")})
	_, err = failing.CancelAllOrders(t.Context(), &order.Cancel{Exchange: failing.Name, AssetType: asset.Spot, Pair: currency.NewBTCUSDT()})
	assert.ErrorIs(t, err, errAPIResponse, "CancelAllOrders should return a pair's failed cancellation")
}

func TestSymbolPairs(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	pairs, err := ex.symbolPairs([]string{"BTCUSDT", "XRPUSDT"})
	require.NoError(t, err, "symbolPairs must not error")
	require.Len(t, pairs, 2, "symbolPairs must return a pair per symbol")
	assert.True(t, pairs[0].Equal(currency.NewBTCUSDT()), "symbolPairs should return the available pair of a symbol")
	formatted, err := ex.FormatSymbol(pairs[1], asset.Spot)
	require.NoError(t, err, "FormatSymbol must not error")
	assert.Equal(t, "XRPUSDT", formatted, "symbolPairs should return a pair formatting back to an unavailable symbol")
}

func TestCancelAllPairError(t *testing.T) {
	t.Parallel()
	assert.NoError(t, cancelAllPairError(currency.NewBTCUSDT(), &APIError{Code: -2011, Message: "Unknown order sent."}), "a pair without orders left to cancel should not error")
	assert.ErrorIs(t, cancelAllPairError(currency.NewBTCUSDT(), errAPIResponse), errAPIResponse, "another error should be returned")
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipUnlessLive(t, true)
		_, err := e.GetOrderInfo(t.Context(), "1", spotTradablePair, asset.Spot)
		assert.ErrorIs(t, err, order.ErrOrderNotFound, "GetOrderInfo should report an order the account does not have as not found")
	})

	orderBody := `{"symbol":"BTCUSDT","orderId":5,"clientOrderId":"c","price":"70000","origQty":"1","executedQty":"0.25","cummulativeQuoteQty":"17500","status":"PARTIALLY_FILLED","timeInForce":"GTC","type":"LIMIT","side":"BUY","time":1794466395184,"updateTime":1794466395185}`
	listBody := `{"orderListId":9,"contingencyType":"OCO","listOrderStatus":"EXECUTING","listClientOrderId":"L","transactionTime":1794466395184,"symbol":"BTCUSDT"}`
	ex := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /api/v3/order": reply(t, url.Values{"symbol": {"BTCUSDT"}, "orderId": {"5"}}, orderBody),
		"GET /api/v3/orderList": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("orderListId") == "9" {
				writeResponse(t, w, listBody)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			writeResponse(t, w, `{"code":-2011,"msg":"Order list does not exist."}`)
		},
	})
	d, err := ex.GetOrderInfo(t.Context(), "5", currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err, "GetOrderInfo must not error")
	assert.Equal(t, 0.75, d.RemainingAmount, "GetOrderInfo should report the remaining amount")
	assert.Equal(t, order.PartiallyFilled, d.Status, "GetOrderInfo should convert the status")
	d, err = ex.GetOrderInfo(t.Context(), "list-9", currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err, "GetOrderInfo must not error for an order list")
	assert.Equal(t, order.OCO, d.Type, "GetOrderInfo should return an order list as an OCO order")

	zero := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /api/v3/orderList": reply(t, url.Values{"orderListId": {"0"}}, `{"orderListId":0,"contingencyType":"OCO","listOrderStatus":"EXECUTING","symbol":"BTCUSDT"}`),
	})
	d, err = zero.GetOrderInfo(t.Context(), formatOrderListID(0), currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err, "GetOrderInfo must look up the order list SubmitOrder returns as list-0")
	assert.Equal(t, "list-0", d.OrderID, "GetOrderInfo should return the order list by its tagged ID")

	_, err = ex.GetOrderInfo(t.Context(), "", currency.NewBTCUSDT(), asset.Spot)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOrderInfo should reject an empty ID")
	_, err = ex.GetOrderInfo(t.Context(), "5", currency.EMPTYPAIR, asset.Spot)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOrderInfo should reject an empty pair")
	_, err = ex.GetOrderInfo(t.Context(), "5", currency.NewBTCUSDT(), asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderInfo should reject an asset other than spot")

	ambiguous := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /api/v3/order":     reply(t, nil, orderBody),
		"GET /api/v3/orderList": reply(t, nil, `{"orderListId":5,"contingencyType":"OCO","listOrderStatus":"EXECUTING","symbol":"BTCUSDT"}`),
	})
	_, err = ambiguous.GetOrderInfo(t.Context(), "5", currency.NewBTCUSDT(), asset.Spot)
	assert.ErrorIs(t, err, errAmbiguousOrderListID, "GetOrderInfo should reject an ID naming both an order and an order list")

	listOnly := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /api/v3/order":     replyError(t, -2013, "Order does not exist."),
		"GET /api/v3/orderList": reply(t, url.Values{"orderListId": {"9"}}, listBody),
	})
	d, err = listOnly.GetOrderInfo(t.Context(), "9", currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err, "GetOrderInfo must not error for an untagged order list ID")
	assert.Equal(t, "list-9", d.OrderID, "GetOrderInfo should return the order list an untagged ID names")

	neither := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /api/v3/order":     replyError(t, -2013, "Order does not exist."),
		"GET /api/v3/orderList": replyError(t, -2011, "Order list does not exist."),
	})
	_, err = neither.GetOrderInfo(t.Context(), "9", currency.NewBTCUSDT(), asset.Spot)
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "GetOrderInfo should report an ID naming neither an order nor an order list as not found")

	orderFails := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/order": replyError(t, -1000, "An unknown error occurred while processing the request.")})
	_, err = orderFails.GetOrderInfo(t.Context(), "5", currency.NewBTCUSDT(), asset.Spot)
	assert.ErrorIs(t, err, errAPIResponse, "GetOrderInfo should return the order lookup's error")

	listFails := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /api/v3/order":     replyError(t, -2013, "Order does not exist."),
		"GET /api/v3/orderList": replyError(t, -1000, "An unknown error occurred while processing the request."),
	})
	_, err = listFails.GetOrderInfo(t.Context(), "5", currency.NewBTCUSDT(), asset.Spot)
	assert.ErrorIs(t, err, errAPIResponse, "GetOrderInfo should return the order list lookup's error")

	unconvertible := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /api/v3/order":     reply(t, nil, strings.Replace(orderBody, `"status":"PARTIALLY_FILLED"`, `"status":"UNKNOWN"`, 1)),
		"GET /api/v3/orderList": replyError(t, -2011, "Order list does not exist."),
	})
	_, err = unconvertible.GetOrderInfo(t.Context(), "5", currency.NewBTCUSDT(), asset.Spot)
	assert.ErrorIs(t, err, errUnknownOrderStatus, "GetOrderInfo should return an order it cannot convert as an error")
}

func TestSpotOrder(t *testing.T) {
	t.Parallel()
	ex := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/order": replyError(t, -2013, "Order does not exist.")})
	_, err := ex.spotOrder(t.Context(), currency.NewBTCUSDT(), 5, "", queryOrderRate)
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "spotOrder should report an unknown order as not found")
}

func TestSpotOrderList(t *testing.T) {
	t.Parallel()
	for name, body := range map[string]string{
		"another symbol":           `{"orderListId":5,"contingencyType":"OCO","listClientOrderId":"L","symbol":"ETHUSDT"}`,
		"another contingency type": `{"orderListId":5,"contingencyType":"OTO","listClientOrderId":"L","symbol":"BTCUSDT"}`,
		"another client ID":        `{"orderListId":5,"contingencyType":"OCO","listClientOrderId":"other","symbol":"BTCUSDT"}`,
	} {
		ex := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/orderList": reply(t, nil, body)})
		_, err := ex.spotOrderList(t.Context(), new(uint64(5)), "L", currency.NewBTCUSDT(), queryOrderListRate)
		assert.ErrorIsf(t, err, order.ErrOrderNotFound, "spotOrderList should not find an order list of %s", name)
	}
	ex := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/orderList": reply(t, url.Values{"orderListId": {"5"}, "origClientOrderId": {"L"}}, `{"orderListId":5,"contingencyType":"OCO","listClientOrderId":"L","symbol":"BTCUSDT"}`)})
	list, err := ex.spotOrderList(t.Context(), new(uint64(5)), "L", currency.NewBTCUSDT(), queryOrderListRate)
	require.NoError(t, err, "spotOrderList must not error")
	assert.Equal(t, uint64(5), list.OrderListID, "spotOrderList should return the order list")
}

func TestOrderListInfo(t *testing.T) {
	t.Parallel()
	ex := &Exchange{Name: "test"}
	d, err := ex.orderListInfo(&OneCancelsTheOtherOrderResponse{OrderListID: 5, ListOrderStatus: "ALL_DONE"}, currency.NewBTCUSDT())
	require.NoError(t, err, "orderListInfo must not error")
	assert.Equal(t, order.Closed, d.Status, "orderListInfo should convert the order list")
	_, err = ex.orderListInfo(&OneCancelsTheOtherOrderResponse{OrderListID: 5, ListOrderStatus: "UNKNOWN"}, currency.NewBTCUSDT())
	assert.ErrorIs(t, err, errUnknownOrderStatus, "orderListInfo should return a conversion error")
}

func TestOrderPair(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	pair, err := ex.orderPair(currency.NewPair(currency.ETH, currency.USDT), "BTCUSDT")
	require.NoError(t, err, "orderPair must not error")
	assert.True(t, pair.Equal(currency.NewPair(currency.ETH, currency.USDT)), "orderPair should keep the requested pair")
	pair, err = ex.orderPair(currency.EMPTYPAIR, "BTCUSDT")
	require.NoError(t, err, "orderPair must not error for a symbol")
	assert.True(t, pair.Equal(currency.NewBTCUSDT()), "orderPair should match the symbol")
}

func TestAppendListedOrder(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	orders := ex.appendListedOrder(nil, "1", currency.EMPTYPAIR, "NOTAPAIR", func(currency.Pair) (order.Detail, error) {
		return order.Detail{OrderID: "1"}, nil
	})
	assert.Empty(t, orders, "an order of an unknown symbol should be left out")
	orders = ex.appendListedOrder(nil, "2", currency.EMPTYPAIR, "BTCUSDT", func(currency.Pair) (order.Detail, error) {
		return order.Detail{OrderID: "2"}, errUnknownOrderStatus
	})
	assert.Equal(t, []order.Detail{{OrderID: "2"}}, orders, "an order with a value that cannot be converted should be kept")
}

func TestSpotOrderVars(t *testing.T) {
	t.Parallel()
	vars, err := spotOrderVars(SideSell, "FILLED", OrderTypeStopLossLimit, TimeInForceFillOrKill)
	require.NoError(t, err, "spotOrderVars must not error")
	assert.Equal(t, orderVars{side: order.Sell, status: order.Filled, orderType: order.StopLimit, timeInForce: order.FillOrKill}, vars, "spotOrderVars should convert every value")
	vars, err = spotOrderVars("", "", "", "")
	require.NoError(t, err, "spotOrderVars must not error for empty values")
	assert.Equal(t, orderVars{}, vars, "spotOrderVars should leave empty values unset")
	_, err = spotOrderVars("X", "X", "X", "X")
	assert.ErrorIs(t, err, errUnknownOrderStatus, "spotOrderVars should join the conversion errors")
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "spotOrderVars should join the conversion errors")
}

func TestSetFilledCost(t *testing.T) {
	t.Parallel()
	d := &order.Detail{Pair: currency.NewBTCUSDT()}
	setFilledCost(d, -1, 1)
	assert.Zero(t, d.Cost, "a negative quote amount, unavailable for some historical orders, should be left out")
	setFilledCost(d, 30, 0.5)
	assert.Equal(t, 30.0, d.Cost, "setFilledCost should set the cost")
	assert.Equal(t, currency.USDT, d.CostAsset, "setFilledCost should set the cost asset")
	assert.Equal(t, 60.0, d.AverageExecutedPrice, "setFilledCost should set the average price")
}

func TestSpotOrderDetail(t *testing.T) {
	t.Parallel()
	ex := &Exchange{Name: "test"}
	o := &TradeOrderResponse{
		OrderID: 5, ClientOrderID: "c", Price: 70000, OriginalQuantity: 1, ExecutedQuantity: 0.25, CumulativeQuoteQuantity: 17500,
		OriginalQuoteOrderQuantity: 0, Status: "PARTIALLY_FILLED", TimeInForce: TimeInForceGoodTillCancel, Type: OrderTypeLimitMaker,
		Side: SideBuy, Time: types.Time(time.UnixMilli(1794466395184)), UpdateTime: types.Time(time.UnixMilli(1794466395185)),
		StopPrice: 0,
	}
	d, err := ex.spotOrderDetail(o, currency.NewBTCUSDT())
	require.NoError(t, err, "spotOrderDetail must not error")
	assert.Equal(t, order.Detail{
		TimeInForce: order.PostOnly, Price: 70000, Amount: 1, ExecutedAmount: 0.25, RemainingAmount: 0.75, Cost: 17500,
		CostAsset: currency.USDT, AverageExecutedPrice: 70000, Exchange: "test", OrderID: "5", ClientOrderID: "c",
		Type: order.LimitMaker, Side: order.Buy, Status: order.PartiallyFilled, AssetType: asset.Spot,
		Date: time.UnixMilli(1794466395184), LastUpdated: time.UnixMilli(1794466395185), Pair: currency.NewBTCUSDT(),
	}, d, "spotOrderDetail should convert every field, the remaining amount included")
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipUnlessLive(t, true)
		_, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{Type: order.AnyType, Side: order.AnySide, AssetType: asset.Spot, Pairs: currency.Pairs{spotTradablePair}})
		assert.NoError(t, err, "GetActiveOrders should not error")
	})

	ex := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /api/v3/openOrders":    reply(t, url.Values{"symbol": {"BTCUSDT"}}, `[{"symbol":"BTCUSDT","orderId":5,"origQty":"1","executedQty":"0.4","status":"PARTIALLY_FILLED","type":"LIMIT","side":"BUY","timeInForce":"GTC"}]`),
		"GET /api/v3/openOrderList": reply(t, url.Values{}, `[{"orderListId":9,"contingencyType":"OCO","listOrderStatus":"EXECUTING","symbol":"BTCUSDT"},{"orderListId":10,"contingencyType":"OTO","symbol":"BTCUSDT"}]`),
	})
	orders, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType, Pairs: currency.Pairs{currency.NewBTCUSDT()}})
	require.NoError(t, err, "GetActiveOrders must not error")
	require.Len(t, orders, 1, "GetActiveOrders must return the open order")
	assert.Equal(t, 0.6, orders[0].RemainingAmount, "GetActiveOrders should report the remaining amount")

	lists, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.OCO})
	require.NoError(t, err, "GetActiveOrders must not error for OCO order lists")
	require.Len(t, lists, 1, "GetActiveOrders must return the OCO order lists alone")
	assert.Equal(t, "list-9", lists[0].OrderID, "GetActiveOrders should identify an order list by its tagged ID")

	_, err = ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Side: order.AnySide, Type: order.AnyType})
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetActiveOrders should reject an asset other than spot")

	every := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /api/v3/openOrders": reply(t, url.Values{}, `[{"symbol":"BTCUSDT","orderId":5,"origQty":"1","executedQty":"0","status":"NEW","type":"LIMIT","side":"BUY","timeInForce":"GTC"},{"symbol":"ETHUSDT","orderId":6,"origQty":"2","executedQty":"0","status":"NEW","type":"LIMIT","side":"SELL","timeInForce":"GTC"}]`),
	})
	orders, err = every.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
	require.NoError(t, err, "GetActiveOrders must not error without pairs")
	require.Len(t, orders, 2, "GetActiveOrders must return every symbol's open orders without pairs")
	assert.True(t, orders[1].Pair.Equal(currency.NewPair(currency.ETH, currency.USDT)), "GetActiveOrders should give each order its symbol's pair")
}

func TestAppendOrderLists(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	orders := ex.appendOrderLists(nil, []OneCancelsTheOtherOrderResponse{
		{OrderListID: 1, ContingencyType: "OCO", ListOrderStatus: "EXECUTING", Symbol: "BTCUSDT"},
		{OrderListID: 2, ContingencyType: "OTO", ListOrderStatus: "EXECUTING", Symbol: "BTCUSDT"},
	})
	require.Len(t, orders, 1, "appendOrderLists must add the OCO order lists alone")
	assert.Equal(t, "list-1", orders[0].OrderID, "appendOrderLists should add an order list by its tagged ID")
}

func TestFilterOrders(t *testing.T) {
	t.Parallel()
	ex := &Exchange{Name: "test"}
	orders := []order.Detail{
		{Exchange: "test", OrderID: "1", Type: order.StopLimit, Side: order.Buy, AssetType: asset.Spot},
		{Exchange: "test", OrderID: "2", Type: order.Limit, Side: order.Buy, AssetType: asset.Spot},
	}
	filtered := ex.filterOrders(&order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.Stop}, orders)
	require.Len(t, filtered, 1, "a bare stop filter must keep the stop orders")
	assert.Equal(t, "1", filtered[0].OrderID, "a bare stop filter should keep the stop limit order")
	assert.Len(t, ex.filterOrders(&order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType}, orders), 2, "any type should keep every order")
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipUnlessLive(t, true)
		_, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{Type: order.AnyType, Side: order.AnySide, AssetType: asset.Spot, Pairs: currency.Pairs{spotTradablePair}, StartTime: time.Now().Add(-24 * time.Hour), EndTime: time.Now()})
		assert.NoError(t, err, "GetOrderHistory should not error")
	})

	start, end := time.UnixMilli(1794380000000), time.UnixMilli(1794466000000)
	ex := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /api/v3/allOrders": reply(t, url.Values{"symbol": {"BTCUSDT"}, "startTime": {"1794380000000"}, "endTime": {"1794466000000"}, "limit": {"1000"}},
			`[{"symbol":"BTCUSDT","orderId":5,"origQty":"1","executedQty":"1","cummulativeQuoteQty":"70000","status":"FILLED","type":"MARKET","side":"SELL","time":1794400000000}]`),
		"GET /api/v3/allOrderList": reply(t, url.Values{"startTime": {"1794380000000"}, "endTime": {"1794466000000"}, "limit": {"1000"}},
			`[{"orderListId":9,"contingencyType":"OCO","listOrderStatus":"ALL_DONE","symbol":"BTCUSDT","transactionTime":1794400000000}]`),
	})
	req := order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType, Pairs: currency.Pairs{currency.NewBTCUSDT()}, StartTime: start, EndTime: end}
	orders, err := ex.GetOrderHistory(t.Context(), &req)
	require.NoError(t, err, "GetOrderHistory must not error")
	require.Len(t, orders, 1, "GetOrderHistory must return the window's orders")
	assert.Equal(t, order.Filled, orders[0].Status, "GetOrderHistory should convert the orders")

	lists := req
	lists.Type = order.OCO
	orders, err = ex.GetOrderHistory(t.Context(), &lists)
	require.NoError(t, err, "GetOrderHistory must not error for OCO order lists")
	require.Len(t, orders, 1, "GetOrderHistory must return the window's order lists")
	assert.Equal(t, "list-9", orders[0].OrderID, "GetOrderHistory should identify an order list by its tagged ID")

	listFrom := req
	listFrom.FromOrderID = "list-9"
	_, err = ex.GetOrderHistory(t.Context(), &listFrom)
	assert.ErrorIs(t, err, errOrderListIDRequiresOCOType, "GetOrderHistory should reject an order list ID without the OCO type")
	noPairs := req
	noPairs.Pairs = nil
	_, err = ex.GetOrderHistory(t.Context(), &noPairs)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "GetOrderHistory should reject a request without pairs")
	futures := req
	futures.AssetType = asset.Futures
	_, err = ex.GetOrderHistory(t.Context(), &futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderHistory should reject an asset other than spot")

	// Replies listing their most recent first, as the order list history does
	reversed := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /api/v3/allOrders":    reply(t, nil, `[{"symbol":"BTCUSDT","orderId":6,"origQty":"1","executedQty":"1","status":"FILLED","type":"MARKET","side":"SELL","time":1794400000001},{"symbol":"BTCUSDT","orderId":5,"origQty":"1","executedQty":"1","status":"FILLED","type":"MARKET","side":"SELL","time":1794400000000}]`),
		"GET /api/v3/allOrderList": reply(t, nil, `[{"orderListId":10,"contingencyType":"OCO","listOrderStatus":"ALL_DONE","symbol":"BTCUSDT","transactionTime":1794400000001},{"orderListId":9,"contingencyType":"OCO","listOrderStatus":"ALL_DONE","symbol":"BTCUSDT","transactionTime":1794400000000}]`),
	})
	orders, err = reversed.GetOrderHistory(t.Context(), &req)
	require.NoError(t, err, "GetOrderHistory must not error")
	require.Len(t, orders, 2, "GetOrderHistory must return the window's orders")
	assert.Equal(t, []string{"5", "6"}, []string{orders[0].OrderID, orders[1].OrderID}, "GetOrderHistory should put the orders in time order")
	orders, err = reversed.GetOrderHistory(t.Context(), &lists)
	require.NoError(t, err, "GetOrderHistory must not error for OCO order lists")
	require.Len(t, orders, 2, "GetOrderHistory must return the window's order lists")
	assert.Equal(t, []string{"list-9", "list-10"}, []string{orders[0].OrderID, orders[1].OrderID}, "GetOrderHistory should put the order lists in time order")
}

func TestPagedOrderHistory(t *testing.T) {
	t.Parallel()
	type record struct {
		id uint64
		at time.Time
	}
	timeOf := func(r *record) time.Time { return r.at }
	idOf := func(r *record) uint64 { return r.id }

	var windows [][2]time.Time
	fetch := func(records []record) func(start, end time.Time) ([]record, error) {
		return func(start, end time.Time) ([]record, error) {
			windows = append(windows, [2]time.Time{start, end})
			var page []record
			for _, r := range records {
				if (start.IsZero() || !r.at.Before(start)) && (end.IsZero() || !r.at.After(end)) && len(page) < 2 {
					page = append(page, r)
				}
			}
			return page, nil
		}
	}
	base := time.UnixMilli(1794380000000)
	records := []record{{1, base}, {2, base.Add(time.Hour)}, {3, base.Add(2 * time.Hour)}, {4, base.Add(25 * time.Hour)}}

	got, err := pagedOrderHistory(base, base.Add(30*time.Hour), 0, 24*time.Hour, 2, fetch(records), timeOf, idOf)
	require.NoError(t, err, "pagedOrderHistory must not error")
	assert.Equal(t, records, got, "pagedOrderHistory should collect every record over spans and full pages once each")
	assert.Greater(t, len(windows), 2, "pagedOrderHistory should page within spans and across them")

	windows = nil
	got, err = pagedOrderHistory(base.Add(time.Hour), base.Add(2*time.Hour), 1, 24*time.Hour, 2, func(start, end time.Time) ([]record, error) {
		assert.True(t, start.IsZero() && end.IsZero(), "a request from an order ID should send no window")
		return slices.Clone(records), nil
	}, timeOf, idOf)
	require.NoError(t, err, "pagedOrderHistory must not error from an order ID")
	assert.Equal(t, []record{records[1], records[2]}, got, "a request from an order ID should apply the window to the records returned")

	_, err = pagedOrderHistory(base.Add(time.Hour), base, 1, 24*time.Hour, 2, fetch(records), timeOf, idOf)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd, "pagedOrderHistory should reject a reversed window")

	// The order list history lists its most recent order lists first
	windows = nil
	descending := fetch(records)
	got, err = pagedOrderHistory(base, base.Add(30*time.Hour), 0, 24*time.Hour, 2, func(start, end time.Time) ([]record, error) {
		page, err := descending(start, end)
		slices.Reverse(page)
		return page, err
	}, timeOf, idOf)
	require.NoError(t, err, "pagedOrderHistory must not error for pages in descending order")
	assert.Equal(t, records, got, "pagedOrderHistory should collect every record of pages in descending order once each, in time order")
	assert.LessOrEqual(t, len(windows), 8, "pagedOrderHistory should page descending records as few times as ascending ones")

	calls := 0
	_, err = pagedOrderHistory(time.Time{}, time.Time{}, 0, 24*time.Hour, 2, func(time.Time, time.Time) ([]record, error) {
		calls++
		return nil, nil
	}, timeOf, idOf)
	require.NoError(t, err, "pagedOrderHistory must not error without a window")
	assert.Equal(t, 1, calls, "a request without a window should be one request")

	windows = nil
	within := base.Add(500 * time.Microsecond)
	_, err = pagedOrderHistory(base, within, 0, 24*time.Hour, 2, fetch(records), timeOf, idOf)
	require.NoError(t, err, "pagedOrderHistory must not error for a window within one millisecond")
	assert.Equal(t, [][2]time.Time{{base, within}}, windows, "a window within one millisecond should be one request for the window")

	// newest answers as a venue returning a window's most recent records would
	newest := func(records []record) func(start, end time.Time) ([]record, error) {
		return func(start, end time.Time) ([]record, error) {
			var page []record
			for i := len(records) - 1; i >= 0 && len(page) < 2; i-- {
				if r := records[i]; !r.at.Before(start) && !r.at.After(end) {
					page = append(page, r)
				}
			}
			return page, nil
		}
	}
	hourly := make([]record, 7)
	for i := range hourly {
		hourly[i] = record{uint64(i + 1), base.Add(time.Duration(i) * time.Hour)}
	}
	got, err = pagedOrderHistory(base, base.Add(30*time.Hour), 0, 24*time.Hour, 2, newest(hourly), timeOf, idOf)
	require.NoError(t, err, "pagedOrderHistory must not error for a venue returning a window's most recent records")
	assert.Equal(t, hourly, got, "pagedOrderHistory should page back through a span from its full first page's earliest record")

	crowded := []record{{1, base}, {2, base.Add(5 * time.Hour)}, {3, base.Add(5 * time.Hour)}, {4, base.Add(5 * time.Hour)}}
	got, err = pagedOrderHistory(base, base.Add(24*time.Hour), 0, 24*time.Hour, 2, newest(crowded), timeOf, idOf)
	require.NoError(t, err, "pagedOrderHistory must not error for a millisecond holding more records than a page")
	assert.Equal(t, []record{crowded[0], crowded[3], crowded[2]}, got, "pagedOrderHistory should page past a millisecond holding more records than a page, forward and back")

	edge := []record{{1, base}, {2, base.Add(2 * time.Hour)}, {3, base.Add(2 * time.Hour)}}
	windows = nil
	got, err = pagedOrderHistory(base, base.Add(2*time.Hour), 0, 24*time.Hour, 2, fetch(edge), timeOf, idOf)
	require.NoError(t, err, "pagedOrderHistory must not error for records in a span's last millisecond")
	assert.Equal(t, edge, got, "pagedOrderHistory should page the span's last millisecond from the millisecond before it")
	assert.Contains(t, windows, [2]time.Time{base.Add(2*time.Hour - time.Millisecond), base.Add(2 * time.Hour)}, "the span's last millisecond should be paged from the millisecond before it")
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	ex.SetCredentials(&accounts.Credentials{})
	ex.SkipAuthCheck = false
	feeBuilder := &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: currency.NewBTCUSDT(), PurchasePrice: 100, Amount: 2}
	fee, err := ex.GetFeeByType(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFeeByType must not error")
	assert.Equal(t, exchange.OfflineTradeFee, feeBuilder.FeeType, "GetFeeByType should estimate trade fees offline without credentials")
	assert.InDelta(t, 0.2, fee, 1e-12, "GetFeeByType should return the offline estimate")
	_, err = ex.GetFeeByType(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "GetFeeByType should reject a nil fee builder")
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	ex := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /sapi/v1/asset/query/trading-fee": reply(t, url.Values{"symbol": {"BTCUSDT"}}, `[{"symbol":"BTCUSDT","makerCommission":"0.004","takerCommission":"0.006"}]`),
	})
	fee, err := ex.GetFee(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: currency.NewBTCUSDT(), PurchasePrice: 100, Amount: 2})
	require.NoError(t, err, "GetFee must not error")
	assert.InDelta(t, 1.2, fee, 1e-12, "GetFee should charge the taker rate, a fraction of the trade's value")
	fee, err = ex.GetFee(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: currency.NewBTCUSDT(), PurchasePrice: 100, Amount: 2, IsMaker: true})
	require.NoError(t, err, "GetFee must not error for a maker")
	assert.InDelta(t, 0.8, fee, 1e-12, "GetFee should charge a maker the maker rate")
	fee, err = ex.GetFee(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyDepositFee})
	require.NoError(t, err, "GetFee must not error for a deposit")
	assert.Zero(t, fee, "deposits should cost nothing")
	_, err = ex.GetFee(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "GetFee should reject a nil fee builder")

	t.Run("through the shared exchange", func(t *testing.T) {
		t.Parallel()
		skipIfCredentialsUnset(t)
		fee, err := e.GetFee(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: currency.NewBTCUSDT(), PurchasePrice: 100, Amount: 2})
		require.NoError(t, err, "GetFee must not error")
		assert.GreaterOrEqual(t, fee, 0.0, "GetFee should not return a negative fee")
	})
}

func TestGetOfflineTradeFee(t *testing.T) {
	t.Parallel()
	assert.InDelta(t, 0.2, getOfflineTradeFee(100, 2), 1e-12, "getOfflineTradeFee should charge the worst case rate")
}

func TestTradeFeeRate(t *testing.T) {
	t.Parallel()
	ex := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /sapi/v1/asset/query/trading-fee": reply(t, nil, `[{"symbol":"ETHUSDT","makerCommission":"0.004","takerCommission":"0.006"}]`),
	})
	_, err := ex.tradeFeeRate(t.Context(), currency.NewBTCUSDT(), false)
	assert.ErrorIs(t, err, errTradeFeeNotFound, "tradeFeeRate should error when the pair's fee is not returned")
	_, err = ex.tradeFeeRate(t.Context(), currency.EMPTYPAIR, false)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "tradeFeeRate should reject an empty pair")
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	ex := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/account": reply(t, nil, `{"balances":[]}`)})
	assert.NoError(t, ex.ValidateAPICredentials(t.Context(), asset.Spot), "ValidateAPICredentials should accept credentials that fetch balances")
	assert.ErrorIs(t, ex.ValidateAPICredentials(t.Context(), asset.Futures), asset.ErrNotSupported, "ValidateAPICredentials should reject an asset other than spot")
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipUnlessLive(t, false)
		end := time.Now().Truncate(time.Hour)
		candles, err := e.GetHistoricCandles(t.Context(), spotTradablePair, asset.Spot, kline.OneHour, end.Add(-24*time.Hour), end)
		require.NoError(t, err, "GetHistoricCandles must not error")
		assert.NotEmpty(t, candles.Candles, "GetHistoricCandles should return the pair's candles")
	})

	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	ex := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/klines": func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "1h", r.URL.Query().Get("interval"), "the candles should be requested by the interval's name")
		writeResponse(t, w, `[[1793491200000,"1","2","0.5","1.5","10",1793494799999,"15",3,"5","7","0"],[1793494800000,"1.5","2.5","1","2","20",1793498399999,"40",4,"10","20","0"]]`)
	}})
	candles, err := ex.GetHistoricCandles(t.Context(), currency.NewBTCUSDT(), asset.Spot, kline.OneHour, start, start.Add(2*time.Hour))
	require.NoError(t, err, "GetHistoricCandles must not error")
	require.Len(t, candles.Candles, 2, "GetHistoricCandles must return the candles")
	assert.Equal(t, kline.Candle{Time: start, Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 10}, candles.Candles[0], "GetHistoricCandles should convert the candles")
	_, err = ex.GetHistoricCandles(t.Context(), currency.NewBTCUSDT(), asset.Futures, kline.OneHour, start, start.Add(time.Hour))
	assert.Error(t, err, "GetHistoricCandles should reject an asset other than spot")
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipUnlessLive(t, false)
		end := time.Now().Truncate(time.Hour)
		candles, err := e.GetHistoricCandlesExtended(t.Context(), spotTradablePair, asset.Spot, kline.OneHour, end.Add(-60*24*time.Hour), end)
		require.NoError(t, err, "GetHistoricCandlesExtended must not error")
		assert.NotEmpty(t, candles.Candles, "GetHistoricCandlesExtended should return the pair's candles")
	})

	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	ex := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/klines": reply(t, nil, `[[1793491200000,"1","2","0.5","1.5","10",1793494799999,"15",3,"5","7","0"],[1793494800000,"1.5","2.5","1","2","20",1793498399999,"40",4,"10","20","0"]]`)})
	candles, err := ex.GetHistoricCandlesExtended(t.Context(), currency.NewBTCUSDT(), asset.Spot, kline.OneHour, start, start.Add(2*time.Hour))
	require.NoError(t, err, "GetHistoricCandlesExtended must not error")
	assert.Len(t, candles.Candles, 2, "GetHistoricCandlesExtended should return the candles")
}

func TestKlineCandles(t *testing.T) {
	t.Parallel()
	ex := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/klines": replyError(t, -1121, "Invalid symbol.")})
	_, err := ex.klineCandles(t.Context(), currency.NewBTCUSDT(), kline.OneHour, time.Time{}, time.Time{}, 1)
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "klineCandles should return a failed request")
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipUnlessLive(t, false)
		require.NoError(t, e.UpdateOrderExecutionLimits(t.Context(), asset.Spot), "UpdateOrderExecutionLimits must not error")
		_, err := e.GetOrderExecutionLimits(asset.Spot, spotTradablePair)
		assert.NoError(t, err, "UpdateOrderExecutionLimits should store the enabled pair's limits")
	})

	ex := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/exchangeInfo": reply(t, url.Values{}, testExchangeInfo)})
	require.NoError(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Spot), "UpdateOrderExecutionLimits must not error")
	l, err := limits.GetOrderExecutionLimits(key.NewExchangeAssetPair(ex.Name, asset.Spot, currency.NewBTCUSDT()))
	require.NoError(t, err, "GetOrderExecutionLimits must not error")
	assert.Equal(t, 0.01, l.PriceStepIncrementSize, "UpdateOrderExecutionLimits should load the price filter")
	assert.ErrorIs(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Futures), asset.ErrNotSupported, "UpdateOrderExecutionLimits should reject an asset other than spot")
}

func TestNarrowPriceBand(t *testing.T) {
	t.Parallel()
	var mml limits.MinMaxLevel
	narrowPriceBand(&mml, 5, 0.2, 5)
	assert.Equal(t, limits.MinMaxLevel{MultiplierUp: 5, MultiplierDown: 0.2, AveragePriceMinutes: 5}, mml, "narrowPriceBand should take the first price band")
	narrowPriceBand(&mml, 1.2, 0.8, 5)
	assert.Equal(t, limits.MinMaxLevel{MultiplierUp: 1.2, MultiplierDown: 0.8, AveragePriceMinutes: 5}, mml, "narrowPriceBand should narrow to a narrower price band")
	narrowPriceBand(&mml, 4, 0.1, 5)
	assert.Equal(t, limits.MinMaxLevel{MultiplierUp: 1.2, MultiplierDown: 0.8, AveragePriceMinutes: 5}, mml, "narrowPriceBand should keep a band narrower than the next")
}

func TestFetchExchangeLimits(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipUnlessLive(t, false)
		l, err := e.FetchExchangeLimits(t.Context(), asset.Spot)
		require.NoError(t, err, "FetchExchangeLimits must not error")
		assert.NotEmpty(t, l, "FetchExchangeLimits should return the spot symbols' limits")
	})

	ex := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/exchangeInfo": reply(t, url.Values{}, testExchangeInfo)})
	l, err := ex.FetchExchangeLimits(t.Context(), asset.Spot)
	require.NoError(t, err, "FetchExchangeLimits must not error")
	require.Len(t, l, 2, "FetchExchangeLimits must return the limits of the symbols trading on the spot market")
	assert.Equal(t, limits.MinMaxLevel{
		Key:                     key.NewExchangeAssetPair(ex.Name, asset.Spot, currency.NewBTCUSDT()),
		MinPrice:                0.01,
		MaxPrice:                1000000,
		PriceStepIncrementSize:  0.01,
		MultiplierUp:            5,
		MultiplierDown:          0.1,
		AveragePriceMinutes:     5,
		MinimumBaseAmount:       0.00001,
		MaximumBaseAmount:       9000,
		AmountStepIncrementSize: 0.00001,
		MinNotional:             1,
		MaxIcebergParts:         10,
		MarketMaxQty:            40,
		MaxTotalOrders:          200,
		MaxAlgoOrders:           5,
	}, l[0], "FetchExchangeLimits should convert every filter")
	_, err = ex.FetchExchangeLimits(t.Context(), asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "FetchExchangeLimits should reject an asset other than spot")

	// Binance.US's TUSDUSDT, as exchangeInfo returned it on 2026-11-11, has both price band filters, which differ
	tusdusdt := `{"symbols":[{"symbol":"TUSDUSDT","status":"BREAK","baseAsset":"TUSD","baseAssetPrecision":8,"quoteAsset":"USDT","quotePrecision":8,"quoteAssetPrecision":8,"baseCommissionPrecision":8,"quoteCommissionPrecision":8,"orderTypes":["LIMIT","LIMIT_MAKER","MARKET","STOP_LOSS_LIMIT","TAKE_PROFIT_LIMIT"],"icebergAllowed":true,"ocoAllowed":true,"otoAllowed":false,"opoAllowed":false,"quoteOrderQtyMarketAllowed":true,"allowTrailingStop":true,"cancelReplaceAllowed":true,"amendAllowed":false,"pegInstructionsAllowed":false,"isSpotTradingAllowed":true,"isMarginTradingAllowed":false,"filters":[{"filterType":"PRICE_FILTER","minPrice":"0.00010000","maxPrice":"1000.00000000","tickSize":"0.00010000"},{"filterType":"PERCENT_PRICE","multiplierUp":"1.2","multiplierDown":"0.8","avgPriceMins":5},{"filterType":"LOT_SIZE","minQty":"1.00000000","maxQty":"900000.00000000","stepSize":"1.00000000"},{"filterType":"MIN_NOTIONAL","minNotional":"1.00000000","applyToMarket":true,"avgPriceMins":5},{"filterType":"ICEBERG_PARTS","limit":10},{"filterType":"MARKET_LOT_SIZE","minQty":"0.00000000","maxQty":"718.00000000","stepSize":"0.00000000"},{"filterType":"TRAILING_DELTA","minTrailingAboveDelta":10,"maxTrailingAboveDelta":2000,"minTrailingBelowDelta":10,"maxTrailingBelowDelta":2000},{"filterType":"PERCENT_PRICE_BY_SIDE","bidMultiplierUp":"1.05","bidMultiplierDown":"0.2","askMultiplierUp":"5","askMultiplierDown":"0.95","avgPriceMins":5},{"filterType":"MAX_NUM_ORDERS","maxNumOrders":200},{"filterType":"MAX_NUM_ALGO_ORDERS","maxNumAlgoOrders":5}],"permissions":[],"permissionSets":[["SPOT"]],"defaultSelfTradePreventionMode":"EXPIRE_MAKER","allowedSelfTradePreventionModes":["EXPIRE_TAKER","EXPIRE_MAKER","EXPIRE_BOTH"]}]}`
	both := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/exchangeInfo": reply(t, url.Values{}, tusdusdt)})
	l, err = both.FetchExchangeLimits(t.Context(), asset.Spot)
	require.NoError(t, err, "FetchExchangeLimits must not error")
	require.Len(t, l, 1, "FetchExchangeLimits must return the symbol's limits")
	assert.Equal(t, 1.2, l[0].MultiplierUp, "FetchExchangeLimits should take the lower of two price bands' upper multipliers")
	assert.Equal(t, 0.8, l[0].MultiplierDown, "FetchExchangeLimits should take the higher of two price bands' lower multipliers")
	assert.Equal(t, 1.0, l[0].MinNotional, "FetchExchangeLimits should take MIN_NOTIONAL's minimum notional")
}

func TestBoundedInt64(t *testing.T) {
	t.Parallel()
	assert.Equal(t, int64(5), boundedInt64(5), "boundedInt64 should keep a count that fits")
	assert.Equal(t, int64(math.MaxInt64), boundedInt64(math.MaxUint64), "boundedInt64 should cap a count that does not fit")
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	serverTime, err := e.GetServerTime(t.Context(), asset.Spot)
	require.NoError(t, err, "GetServerTime must not error")
	assert.NotZero(t, serverTime, "GetServerTime should return the server time")
	_, err = e.GetServerTime(t.Context(), asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetServerTime should reject an asset other than spot")
	failing := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/time": replyError(t, -1000, "An unknown error occurred while processing the request.")})
	_, err = failing.GetServerTime(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, errAPIResponse, "GetServerTime should return the request's error")
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).GetFuturesContractDetails(t.Context(), asset.Futures)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetFuturesContractDetails should not be supported")
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).GetLatestFundingRates(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetLatestFundingRates should not be supported")
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	u, err := ex.GetCurrencyTradeURL(t.Context(), asset.Spot, currency.NewBTCUSDT())
	require.NoError(t, err, "GetCurrencyTradeURL must not error")
	assert.Equal(t, tradeBaseURL+"BTCUSDT", u, "GetCurrencyTradeURL should return the pair's trade page")
	_, err = ex.GetCurrencyTradeURL(t.Context(), asset.Spot, currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCurrencyTradeURL should reject an empty pair")
	_, err = ex.GetCurrencyTradeURL(t.Context(), asset.Futures, currency.NewBTCUSDT())
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetCurrencyTradeURL should reject an asset other than spot")
}

// testCoinInfo lists a coin with a default network before BTC, as a fee taken from the first default network is wrong
const testCoinInfo = `[{"coin":"ETH","networkList":[{"network":"ETH","coin":"ETH","isDefault":true,"depositEnable":true,"withdrawFee":"0.004"}]},` +
	`{"coin":"BTC","networkList":[{"network":"BNB","coin":"BTC","isDefault":false,"depositEnable":false,"withdrawFee":"0.0001"},` +
	`{"network":"BTC","coin":"BTC","isDefault":true,"depositEnable":true,"withdrawFee":"0.0005"}]},` +
	`{"coin":"XRP","networkList":[{"network":"XRP","coin":"XRP","isDefault":false,"depositEnable":true}]}]`

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).GetAccountFundingHistory(t.Context())
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetAccountFundingHistory should not be supported")
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipUnlessLive(t, true)
		_, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
		assert.NoError(t, err, "GetWithdrawalsHistory should not error")
	})

	var full strings.Builder
	full.WriteByte('[')
	for i := range fundingHistoryPageLimit {
		if i > 0 {
			full.WriteByte(',')
		}
		full.WriteString(`{"id":"w` + strconv.Itoa(i) + `","amount":"0.1","transactionFee":"0.0005","coin":"BTC","status":6,"address":"addr","applyTime":"2026-11-12 08:00:00","network":"BTC"}`)
	}
	full.WriteByte(']')
	ex := routedTestExchange(t, map[string]http.HandlerFunc{"GET /sapi/v1/capital/withdraw/history": func(w http.ResponseWriter, r *http.Request) {
		q := signedQuery(t, r)
		assert.Equal(t, "BTC", q.Get("coin"), "each page should be requested for the coin")
		assert.Equal(t, "1000", q.Get("limit"), "each page should be requested in full")
		if q.Get("offset") == "1000" {
			writeResponse(t, w, `[{"id":"last","amount":"1","transactionFee":"0.0005","coin":"BTC","status":9,"address":"addr","applyTime":"2026-11-12 09:00:00","network":"BTC"}]`)
			return
		}
		assert.Empty(t, q.Get("offset"), "the first page should start at the first record")
		writeResponse(t, w, full.String())
	}})
	history, err := ex.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	require.NoError(t, err, "GetWithdrawalsHistory must not error")
	require.Len(t, history, fundingHistoryPageLimit+1, "GetWithdrawalsHistory must collect every page")
	assert.Equal(t, exchange.WithdrawalHistory{
		Status: "Completed", TransferID: "w0", Timestamp: time.Date(2026, 11, 12, 8, 0, 0, 0, time.UTC), Currency: "BTC",
		Amount: 0.1, Fee: 0.0005, TransferType: "withdrawal", CryptoToAddress: "addr", CryptoChain: "BTC",
	}, history[0], "GetWithdrawalsHistory should convert the withdrawals, which carry no transaction ID")
	assert.Equal(t, "9", history[fundingHistoryPageLimit].Status, "an undocumented status should be reported by its code")

	_, err = ex.GetWithdrawalsHistory(t.Context(), currency.EMPTYCODE, asset.Spot)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetWithdrawalsHistory should require a coin")
	_, err = ex.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetWithdrawalsHistory should reject an asset other than spot")
}

func TestWithdrawalStatusString(t *testing.T) {
	t.Parallel()
	for status, exp := range map[uint64]string{0: "Email sent", 1: "Cancelled", 2: "Awaiting approval", 3: "Rejected", 4: "Processing", 5: "Failure", 6: "Completed", 9: "9"} {
		assert.Equalf(t, exp, withdrawalStatusString(status), "withdrawalStatusString should name status %d", status)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipUnlessLive(t, true)
		_, err := e.GetDepositAddress(t.Context(), currency.BTC, "", "")
		assert.NoError(t, err, "GetDepositAddress should not error")
	})

	ex := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /sapi/v1/capital/deposit/address": reply(t, url.Values{"coin": {"XRP"}, "network": {"XRP"}}, `{"coin":"XRP","address":"rAddress","tag":"100345"}`),
	})
	addr, err := ex.GetDepositAddress(t.Context(), currency.XRP, "", "XRP")
	require.NoError(t, err, "GetDepositAddress must not error")
	assert.Equal(t, &deposit.Address{Address: "rAddress", Tag: "100345", Chain: "XRP"}, addr, "GetDepositAddress should return the address and its tag")
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipUnlessLive(t, true, canManipulateRealOrders)
		_, err := e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{Exchange: e.Name, Currency: currency.BTC, Amount: 0.001, Type: withdraw.Crypto, Crypto: withdraw.CryptoRequest{Address: core.BitcoinDonationAddress}})
		assert.NoError(t, err, "WithdrawCryptocurrencyFunds should not error")
	})

	var withdrawals atomic.Int32
	ex := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /sapi/v1/capital/config/getall": reply(t, nil, testCoinInfo),
		"POST /sapi/v1/capital/withdraw/apply": func(w http.ResponseWriter, r *http.Request) {
			withdrawals.Add(1)
			q := signedQuery(t, r)
			assert.Equal(t, "BTC", q.Get("network"), "a withdrawal without a chain should go on the coin's default network")
			assert.Equal(t, "0.01", q.Get("amount"), "the amount should be sent in full")
			writeResponse(t, w, `{"id":"withdrawal-1"}`)
		},
	})
	req := &withdraw.Request{Exchange: ex.Name, Currency: currency.BTC, Amount: 0.01, Type: withdraw.Crypto, Crypto: withdraw.CryptoRequest{Address: core.BitcoinDonationAddress}}
	resp, err := ex.WithdrawCryptocurrencyFunds(t.Context(), req)
	require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error")
	assert.Equal(t, "withdrawal-1", resp.ID, "WithdrawCryptocurrencyFunds should return the withdrawal ID")
	assert.Equal(t, int32(1), withdrawals.Load(), "WithdrawCryptocurrencyFunds should withdraw once")
	_, err = ex.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{})
	assert.Error(t, err, "WithdrawCryptocurrencyFunds should reject an invalid request")
}

func TestWithdrawFiatFunds(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).WithdrawFiatFunds(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFunds should not be supported")
}

func TestWithdrawFiatFundsToInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).WithdrawFiatFundsToInternationalBank(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFundsToInternationalBank should not be supported")
}

func TestCoinInfo(t *testing.T) {
	t.Parallel()
	ex := routedTestExchange(t, map[string]http.HandlerFunc{"GET /sapi/v1/capital/config/getall": reply(t, url.Values{}, testCoinInfo)})
	coin, err := ex.coinInfo(t.Context(), currency.XRP)
	require.NoError(t, err, "coinInfo must not error")
	assert.True(t, coin.Coin.Equal(currency.XRP), "coinInfo should return the coin's information")
	_, err = ex.coinInfo(t.Context(), currency.LTC)
	assert.ErrorIs(t, err, currency.ErrCurrencyNotFound, "coinInfo should error for a coin Binance.US does not list")
	_, err = ex.coinInfo(t.Context(), currency.EMPTYCODE)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "coinInfo should reject an empty coin")
}

func TestDefaultNetwork(t *testing.T) {
	t.Parallel()
	ex := routedTestExchange(t, map[string]http.HandlerFunc{"GET /sapi/v1/capital/config/getall": reply(t, nil, testCoinInfo)})
	network, err := ex.defaultNetwork(t.Context(), currency.BTC)
	require.NoError(t, err, "defaultNetwork must not error")
	assert.Equal(t, "BTC", network.Network, "defaultNetwork should return the coin's default network")
	_, err = ex.defaultNetwork(t.Context(), currency.XRP)
	assert.ErrorIs(t, err, errDefaultNetworkNotFound, "defaultNetwork should error for a coin without a default network")
}

func TestCryptocurrencyWithdrawalFee(t *testing.T) {
	t.Parallel()
	ex := routedTestExchange(t, map[string]http.HandlerFunc{"GET /sapi/v1/capital/config/getall": reply(t, nil, testCoinInfo)})
	fee, err := ex.cryptocurrencyWithdrawalFee(t.Context(), currency.BTC)
	require.NoError(t, err, "cryptocurrencyWithdrawalFee must not error")
	assert.Equal(t, 0.0005, fee, "cryptocurrencyWithdrawalFee should return the coin's default network fee, not another coin's")
	fee, err = ex.GetFee(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyWithdrawalFee, Pair: currency.NewBTCUSDT()})
	require.NoError(t, err, "GetFee must not error for a withdrawal")
	assert.Equal(t, 0.0005, fee, "GetFee should return the base coin's withdrawal fee")
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipUnlessLive(t, true)
		chains, err := e.GetAvailableTransferChains(t.Context(), currency.BTC)
		require.NoError(t, err, "GetAvailableTransferChains must not error")
		assert.NotEmpty(t, chains, "GetAvailableTransferChains should return the coin's networks")
	})

	ex := routedTestExchange(t, map[string]http.HandlerFunc{"GET /sapi/v1/capital/config/getall": reply(t, nil, testCoinInfo)})
	chains, err := ex.GetAvailableTransferChains(t.Context(), currency.BTC)
	require.NoError(t, err, "GetAvailableTransferChains must not error")
	assert.Equal(t, []string{"BTC"}, chains, "GetAvailableTransferChains should return the networks deposits are enabled on")
	_, err = ex.GetAvailableTransferChains(t.Context(), currency.EMPTYCODE)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAvailableTransferChains should reject an empty coin")
}

func TestSetupWithPreviousConfig(t *testing.T) {
	t.Parallel()
	// The configuration a previous version saved, from upstream's test configuration, holds neither the WebSocket API
	// endpoint nor subscriptions, as that version had no WebSocket API connection and ignored configured subscriptions
	raw, err := os.ReadFile("testdata/previous_config.json")
	require.NoError(t, err, "ReadFile must not error")
	for _, authenticated := range []bool{false, true} {
		var exch config.Exchange
		require.NoError(t, json.Unmarshal(raw, &exch), "the previous configuration must decode")
		exch.API.AuthenticatedWebsocketSupport = authenticated
		ex := new(Exchange)
		ex.SetDefaults()
		require.NoErrorf(t, ex.Setup(&exch), "Setup must not error with authenticated websocket support %t", authenticated)
		u, err := ex.API.Endpoints.GetURL(exchange.WebsocketTrade)
		require.NoError(t, err, "GetURL must not error")
		assert.Equal(t, websocketAPIURL, u, "a previous configuration should take the WebSocket API's default endpoint")
		assert.Equal(t, websocketAPIURL, exch.API.Endpoints[exchange.WebsocketTrade.String()], "Setup should save the WebSocket API endpoint to the configuration")
		channels := make([]string, len(exch.Features.Subscriptions))
		for i, s := range exch.Features.Subscriptions {
			channels[i] = s.Channel
		}
		assert.Equal(t, []string{subscription.TickerChannel, subscription.AllTradesChannel, subscription.CandlesChannel, subscription.OrderbookChannel}, channels,
			"a previous configuration should take the default subscriptions, the streams the previous version subscribed")
	}
}
