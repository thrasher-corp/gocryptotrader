package bitstamp

import (
	"math"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange"
	"github.com/thrasher-corp/gocryptotrader/exchange/asset"
	"github.com/thrasher-corp/gocryptotrader/exchange/order"
	"github.com/thrasher-corp/gocryptotrader/exchange/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchange/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchange/trade"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestHandleWebsocketOrderbook(t *testing.T) { //nolint:tparallel // top-level parallel is safe; the cases load their snapshots into the test's own exchange in turn
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	assert.Error(t, ex.handleWSOrderbook(t.Context(), []byte(`{"bids":"wibble"}`), wsSpotPair, asset.Spot), "handleWSOrderbook should error on malformed data")

	// Serial, as the snapshots are loaded into the same exchange
	for _, tc := range []struct {
		name string
		data string
		exp  *orderbook.Book
	}{
		{
			name: "spot",
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
			name: "perpetual contract",
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
		t.Run(tc.name, func(t *testing.T) {
			require.NoErrorf(t, ex.handleWSOrderbook(t.Context(), []byte(tc.data), tc.exp.Pair, tc.exp.Asset), "handleWSOrderbook must not error for %s", tc.exp.Asset)
			ob, err := ex.Websocket.Orderbook.GetOrderbook(tc.exp.Pair, tc.exp.Asset)
			require.NoErrorf(t, err, "GetOrderbook must not error for %s", tc.exp.Asset)
			assert.NotZerof(t, ob.InsertedAt, "InsertedAt should be set for %s", tc.exp.Asset)
			ob.InsertedAt = time.Time{}
			assert.Equalf(t, tc.exp, ob, "handleWSOrderbook should load the snapshot without zero priced bids for %s", tc.exp.Asset)
		})
	}
}

func TestHandleWebsocketTrade(t *testing.T) {
	t.Parallel()
	data := []byte(`{"microtimestamp":"1580336751488517","amount":0.00598803,"buy_order_id":4621328909,"sell_order_id":4621329035,"amount_str":"0.00598803","price_str":"9334.73","timestamp":"1580336751","price":9334.73,"type":1,"id":104007706}`)
	ex := newWebsocketTestInstance(t)
	require.NoError(t, ex.handleWSTrade(data, wsSpotPair, asset.Spot), "handleWSTrade must not error with the trade feed disabled")
	assert.Empty(t, dataHandlerPayloads(t, ex), "trades should not be sent with the trade feed disabled")

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
	}}}, dataHandlerPayloads(t, ex), "trades should be sent to the data handler")
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
	}, dataHandlerPayloads(t, ex), "funding rates and updates should be sent to the data handler")
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

// newOrderbookSeedTestInstance returns a websocket test instance whose REST requests are served by handler
func newOrderbookSeedTestInstance(t *testing.T, handler http.HandlerFunc) *Exchange {
	t.Helper()
	s := httptest.NewTestServer(t, handler)
	s.Start()
	ex := newWebsocketTestInstance(t)
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), s.URL), "SetRunningURL must not error")
	return ex
}

func TestSeedOrderbook(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("The orderbook is loaded from the live API")
		}
		ex := newLiveTestInstance(t)
		require.NoError(t, ex.seedOrderbook(t.Context(), &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{wsSpotPair}}), "seedOrderbook must not error")
		ob, err := orderbook.Get(ex.Name, wsSpotPair, asset.Spot)
		require.NoError(t, err, "the orderbook must be loaded")
		assert.False(t, ob.PriceDuplication, "the orderbook should not repeat prices")
		assert.NotEmpty(t, ob.Bids, "the orderbook should hold bids")
		assert.NotEmpty(t, ob.Asks, "the orderbook should hold asks")
	})
	for _, tc := range []struct {
		name             string
		sub              *subscription.Subscription
		requests         []string
		loaded           currency.Pairs
		unloaded         currency.Pairs
		priceDuplication bool
		errs             []error
	}{
		{
			name:     "orderbook",
			sub:      &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{wsSpotPair}},
			requests: []string{"/v2/order_book/btcusd/?group=1"},
			loaded:   currency.Pairs{wsSpotPair},
		},
		{
			name:             "detail orderbook",
			sub:              &subscription.Subscription{Channel: channelDetailOrderbook, Asset: asset.PerpetualContract, Pairs: currency.Pairs{wsPerpetualPair}},
			requests:         []string{"/v2/order_book/btcusd-perp/?group=2"},
			loaded:           currency.Pairs{wsPerpetualPair},
			priceDuplication: true,
		},
		{
			name:     "a market which cannot be loaded",
			sub:      &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPairWithDelimiter("BTC", "EUR", "/"), wsSpotPair}},
			requests: []string{"/v2/order_book/btceur/?group=1", "/v2/order_book/btcusd/?group=1"},
			loaded:   currency.Pairs{wsSpotPair},
			unloaded: currency.Pairs{currency.NewPairWithDelimiter("BTC", "EUR", "/")},
			errs:     []error{errOrderbookSeed, errAPIResponse},
		},
		{
			name: "no pairs",
			sub:  &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var requests []string
			ex := newOrderbookSeedTestInstance(t, func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				requests = append(requests, r.URL.Path+"?"+r.URL.RawQuery)
				mu.Unlock()
				var body string
				switch r.URL.Path {
				case "/v2/order_book/btcusd/":
					body = `{"timestamp":"1791848800","microtimestamp":"1791848800102345","bids":[["100","1"],["99","2"]],"asks":[["101","3"]]}`
				case "/v2/order_book/btcusd-perp/":
					body = `{"timestamp":"1791848800","microtimestamp":"1791848800102346","bids":[["100","1","11"],["100","2","12"]],"asks":[["101","3","13"]]}`
				default:
					http.Error(w, `{"status":"error","reason":"Unavailable"}`, http.StatusInternalServerError)
					return
				}
				_, err := w.Write([]byte(body))
				assert.NoError(t, err, "Write should not error")
			})
			err := ex.seedOrderbook(t.Context(), tc.sub)
			if len(tc.errs) == 0 {
				require.NoError(t, err, "seedOrderbook must not error")
			}
			for _, exp := range tc.errs {
				assert.ErrorIs(t, err, exp, "seedOrderbook should return the expected error")
			}
			mu.Lock()
			assert.Equal(t, tc.requests, requests, "seedOrderbook should request the orderbook of each market with the channel's grouping")
			mu.Unlock()
			for _, p := range tc.loaded {
				ob, err := orderbook.Get(ex.Name, p, tc.sub.Asset)
				if assert.NoErrorf(t, err, "the %s orderbook should be loaded", p) {
					assert.Equalf(t, tc.priceDuplication, ob.PriceDuplication, "the %s orderbook should repeat prices only for the detail orderbook channel", p)
				}
			}
			for _, p := range tc.unloaded {
				_, err := orderbook.Get(ex.Name, p, tc.sub.Asset)
				assert.ErrorIsf(t, err, orderbook.ErrOrderbookNotFound, "the %s orderbook should not be loaded", p)
			}
		})
	}
}

func TestSeedMarketOrderbook(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("The orderbook is loaded from the live API")
		}
		ex := newLiveTestInstance(t)
		require.NoError(t, ex.seedMarketOrderbook(t.Context(), "btcusd-perp", OrderbookUngroupedWithOrderIDs, true), "seedMarketOrderbook must not error")
		ob, err := orderbook.Get(ex.Name, wsPerpetualPair, asset.PerpetualContract)
		require.NoError(t, err, "the orderbook must be loaded")
		assert.True(t, ob.PriceDuplication, "the orderbook of orders should repeat prices")
		require.NotEmpty(t, ob.Bids, "the orderbook must hold bids")
		require.NotEmpty(t, ob.Asks, "the orderbook must hold asks")
		assert.Positive(t, ob.Bids[0].ID, "the orderbook should hold each bid's order ID")
		assert.Positive(t, ob.Asks[0].ID, "the orderbook should hold each ask's order ID")
	})
	for _, tc := range []struct {
		name     string
		market   string
		grouping OrderbookGrouping
		detail   bool
		request  string
		status   int
		response string
		exp      *orderbook.Book
		err      error
	}{
		{
			name:     "price levels",
			market:   "btcusd",
			grouping: OrderbookGroupedByPrice,
			request:  "/v2/order_book/btcusd/?group=1",
			response: `{"timestamp":"1791848800","microtimestamp":"1791848800102345","bids":[["100","1"],["99","2"],["0","1"]],"asks":[["101","3"]]}`,
			exp: &orderbook.Book{
				Bids:        orderbook.Levels{{Price: 100, Amount: 1}, {Price: 99, Amount: 2}},
				Asks:        orderbook.Levels{{Price: 101, Amount: 3}},
				Pair:        wsSpotPair,
				Asset:       asset.Spot,
				LastUpdated: time.UnixMicro(1791848800102345),
			},
		},
		{
			name:     "orders",
			market:   "btcusd-perp",
			grouping: OrderbookUngroupedWithOrderIDs,
			detail:   true,
			request:  "/v2/order_book/btcusd-perp/?group=2",
			response: `{"timestamp":"1791848800","microtimestamp":"1791848800102346","bids":[["100","1","11"],["100","2","12"],["0","1","14"]],"asks":[["101","3","13"]]}`,
			exp: &orderbook.Book{
				Bids:             orderbook.Levels{{Price: 100, Amount: 1, ID: 11}, {Price: 100, Amount: 2, ID: 12}},
				Asks:             orderbook.Levels{{Price: 101, Amount: 3, ID: 13}},
				Pair:             wsPerpetualPair,
				Asset:            asset.PerpetualContract,
				LastUpdated:      time.UnixMicro(1791848800102346),
				PriceDuplication: true,
			},
		},
		{name: "unknown market", market: "wibble", grouping: OrderbookGroupedByPrice, err: errParsingWSPair},
		{name: "unavailable orderbook", market: "btcusd", grouping: OrderbookGroupedByPrice, request: "/v2/order_book/btcusd/?group=1", status: http.StatusInternalServerError, response: `{"status":"error","reason":"Unavailable"}`, err: errAPIResponse},
		{name: "bid order ID beyond a level ID", market: "btcusd-perp", grouping: OrderbookUngroupedWithOrderIDs, detail: true, request: "/v2/order_book/btcusd-perp/?group=2", response: `{"bids":[["1","1","9223372036854775808"]],"asks":[]}`, err: errOrderIDOverflow},
		{name: "ask order ID beyond a level ID", market: "btcusd-perp", grouping: OrderbookUngroupedWithOrderIDs, detail: true, request: "/v2/order_book/btcusd-perp/?group=2", response: `{"bids":[],"asks":[["1","1","9223372036854775808"]]}`, err: errOrderIDOverflow},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newOrderbookSeedTestInstance(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tc.request, r.URL.Path+"?"+r.URL.RawQuery, "the request should be for the market's orderbook with the grouping")
				if tc.status != 0 {
					w.WriteHeader(tc.status)
				}
				_, err := w.Write([]byte(tc.response))
				assert.NoError(t, err, "Write should not error")
			})
			err := ex.seedMarketOrderbook(t.Context(), tc.market, tc.grouping, tc.detail)
			require.ErrorIs(t, err, tc.err, "seedMarketOrderbook must return the expected error")
			if tc.err != nil {
				return
			}
			ob, err := ex.Websocket.Orderbook.GetOrderbook(tc.exp.Pair, tc.exp.Asset)
			require.NoError(t, err, "GetOrderbook must not error")
			assert.NotZero(t, ob.InsertedAt, "InsertedAt should be set")
			ob.InsertedAt = time.Time{}
			exp := *tc.exp
			exp.Exchange, exp.ValidateOrderbook = ex.Name, ex.ValidateOrderbook
			assert.Equal(t, &exp, ob, "seedMarketOrderbook should load the REST orderbook without zero priced bids")
		})
	}
}

func TestIsOrderbookSubscription(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		channel string
		exp     bool
	}{
		{name: "orderbook", channel: subscription.OrderbookChannel, exp: true},
		{name: "detail orderbook", channel: channelDetailOrderbook, exp: true},
		{name: "all trades", channel: subscription.AllTradesChannel},
		{name: "my orders", channel: subscription.MyOrdersChannel},
		{name: "unsupported channel", channel: "wibble"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, isOrderbookSubscription(&subscription.Subscription{Channel: tc.channel}), "isOrderbookSubscription should report whether the channel loads an orderbook")
		})
	}
}

func TestOrderLevels(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		levels []OrderbookLevel
		exp    orderbook.Levels
		err    error
	}{
		{name: "no levels", exp: orderbook.Levels{}},
		{
			name:   "orders at one price",
			levels: []OrderbookLevel{{Price: 81730.34, Amount: 0.01, OrderID: 2060344614850561}, {Price: 81730.34, Amount: 0.02, OrderID: 2060344614850562}},
			exp:    orderbook.Levels{{Price: 81730.34, Amount: 0.01, ID: 2060344614850561}, {Price: 81730.34, Amount: 0.02, ID: 2060344614850562}},
		},
		{name: "largest level ID", levels: []OrderbookLevel{{Price: 1, Amount: 1, OrderID: math.MaxInt64}}, exp: orderbook.Levels{{Price: 1, Amount: 1, ID: math.MaxInt64}}},
		{name: "order ID beyond the largest level ID", levels: []OrderbookLevel{{Price: 1, Amount: 1, OrderID: 1}, {Price: 1, Amount: 1, OrderID: math.MaxInt64 + 1}}, err: errOrderIDOverflow},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			levels, err := orderLevels(tc.levels)
			require.ErrorIs(t, err, tc.err, "orderLevels must return the expected error")
			assert.Equal(t, tc.exp, levels, "orderLevels should return a level holding each order")
		})
	}
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
	}}, dataHandlerPayloads(t, ex), "announcements should be sent to the data handler")
}
