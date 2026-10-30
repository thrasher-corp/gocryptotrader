package htx

import (
	"fmt"
	"io"
	"maps"
	"math"
	"net/http"
	"net/http/httptest"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const emptySuccessResponse = `{"status":"ok","code":200,"msg":"","data":null}`

// httpTestResponse defines an additional route and its expected request count.
type httpTestResponse struct {
	method, path, body string
	calls              int32
}

func newHTTPTestExchange(t *testing.T, endpoint exchange.URL, method, path, response string, check func(*http.Request), additional ...httpTestResponse) *Exchange {
	t.Helper()
	counts := make([]atomic.Int32, len(additional))
	t.Cleanup(func() {
		for i, expected := range additional {
			assert.Equalf(t, expected.calls, counts[i].Load(), "%s %s request count should match", expected.method, expected.path)
		}
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		for i, expected := range additional {
			if r.URL.Path == expected.path {
				counts[i].Add(1)
				assert.Equal(t, expected.method, r.Method, "additional HTTP method should match")
				_, _ = w.Write([]byte(expected.body))
				return
			}
		}
		assert.Equal(t, method, r.Method, "HTTP method should match")
		assert.Equal(t, path, r.URL.Path, "endpoint path should match")
		if check != nil {
			check(r)
		}
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	// Production instances share one budget; mock requests must not consume another test's allowance.
	request.WithLimiter(newRateLimits())(h.Requester)
	h.API.AuthenticatedSupport = true
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	require.NoError(t, h.API.Endpoints.SetRunningURL(endpoint.String(), server.URL), "running endpoint must be set")
	return h
}

// setV5PositionModeEndpoint configures the position-mode response required by V5 order tests.
func setV5PositionModeEndpoint(t *testing.T, h *Exchange, mode string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v5/position/mode", r.URL.Path, "position mode endpoint should match")
		assert.Equal(t, http.MethodGet, r.Method, "position mode should be requested with GET")
		_, _ = w.Write([]byte(`{"code":200,"data":{"position_mode":"` + mode + `"}}`))
	}))
	t.Cleanup(server.Close)
	h.API.AuthenticatedSupport = true
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "position mode endpoint must be set")
}

func TestNewHTTPTestExchange(t *testing.T) {
	t.Parallel()
	var checked atomic.Bool
	h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/account/asset_mode", `{"code":200,"data":{"asset_mode":1}}`, func(r *http.Request) {
		checked.Store(true)
		assert.NotEmpty(t, r.URL.Query().Get("Signature"), "authenticated request should include a signature")
	}, httpTestResponse{method: http.MethodGet, path: "/v5/position/mode", body: `{"code":200,"data":{"position_mode":"single_side"}}`, calls: 1})
	mode, err := h.GetV5PositionMode(t.Context())
	require.NoError(t, err, "additional HTTP route must be available")
	assert.Equal(t, "single_side", mode.Mode.PositionMode, "additional route should return its own fixture")
	response, err := h.GetV5AssetMode(t.Context())
	require.NoError(t, err, "newHTTPTestExchange must provide a usable authenticated exchange")
	assert.True(t, checked.Load(), "request check should run")
	assert.Equal(t, uint64(1), response.Mode.AssetMode, "response should decode")
}

func TestSetDefaults(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	h.SetDefaults()
	assert.Equal(t, "HTX", h.Name, "exchange name should match")
	assert.False(t, h.Features.Supports.WebsocketCapabilities.FundingRateFetching, "unsupported notification transport should not advertise websocket funding")
	assert.True(t, h.Features.Supports.WebsocketCapabilities.SubmitOrder, "websocket order submission should be supported")
	assert.True(t, h.Features.Supports.WebsocketCapabilities.SubmitOrders, "websocket batch order submission should be supported")
	assert.True(t, h.Features.Supports.WebsocketCapabilities.CancelOrder, "websocket order cancellation should be supported")
	assert.True(t, h.Features.TradingRequirements.SpotMarketBuyQuotation, "spot market buys should require quote amount")
	assert.True(t, h.Features.TradingRequirements.SpotMarketSellBase, "spot market sells should require base amount")
	assert.Len(t, h.Features.Subscriptions, 36, "default subscriptions should cover public and private spot and derivatives channels")
	for _, sub := range h.Features.Subscriptions {
		if sub.Authenticated && sub.Asset != asset.Spot {
			assert.False(t, sub.Enabled, "private derivative subscriptions should default to disabled")
		}
	}
	type testCase struct {
		endpoint exchange.URL
		want     string
	}
	for _, tt := range []testCase{
		{endpoint: exchange.WebsocketFuturesPrivate, want: wsFuturesPrivateURL},
		{endpoint: exchange.WebsocketCoinMarginedPrivate, want: wsCoinMarginedPrivateURL},
		{endpoint: exchange.WebsocketUSDTMarginedPrivate, want: wsUSDTMarginedPrivateURL},
		{endpoint: exchange.WebsocketTrade, want: wsUSDTMarginedTradeURL},
	} {
		got, err := h.API.Endpoints.GetURL(tt.endpoint)
		require.NoError(t, err, "private derivative endpoint must be configured")
		assert.Equal(t, tt.want, got, "private derivative endpoint should match HTX documentation")
	}
}

func TestSetup(t *testing.T) {
	t.Parallel()
	t.Run("trade connection requires authenticated websocket support", func(t *testing.T) {
		t.Parallel()
		h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, wsFixture))
		_, err := h.Websocket.GetConnection(exchange.WebsocketTrade)
		require.Error(t, err, "trade connection must be skipped while authenticated endpoints are disabled")
	})
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	assert.NotNil(t, h.Websocket, "websocket manager should be configured")
	assert.True(t, h.SupportsAsset(asset.Futures), "delivery futures should remain enabled")
	assert.True(t, h.SupportsAsset(asset.CoinMarginedFutures), "coin-margined futures should remain enabled")
	assert.True(t, h.SupportsAsset(asset.USDTMarginedFutures), "USDT-margined futures should remain enabled")
	tradeSetup := &websocket.ConnectionSetup{
		URL:                      wsUSDTMarginedTradeURL,
		Connector:                h.wsConnect,
		Handler:                  h.wsHandleData,
		MessageFilter:            exchange.WebsocketTrade,
		SubscriptionsNotRequired: true,
	}
	require.NoError(t, h.Websocket.SetupNewConnection(tradeSetup), "trade websocket must be omitted when authenticated websocket support is disabled")

	authenticated := new(Exchange)
	authenticated.SetDefaults()
	require.NoError(t, authenticated.Setup(&config.Exchange{
		Name:                    "HTX",
		Enabled:                 true,
		WebsocketTrafficTimeout: time.Second,
		API: config.APIConfig{
			AuthenticatedWebsocketSupport: true,
		},
	}), "Setup must configure authenticated websocket support")
	authenticatedTradeSetup := *tradeSetup
	authenticatedTradeSetup.Connector = authenticated.wsConnect
	authenticatedTradeSetup.Handler = authenticated.wsHandleData
	err := authenticated.Websocket.SetupNewConnection(&authenticatedTradeSetup)
	require.Error(t, err, "trade websocket must already be configured when authenticated websocket support is enabled")
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Futures)
	require.NoError(t, err)
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	require.NoError(t, h.UpdateTradablePairs(t.Context()), "UpdateTradablePairs must not error")
}

func TestUpdateCurrencyStates(t *testing.T) {
	t.Parallel()
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v2/reference/currencies", `{
		"code":200,
		"data":[
			{"currency":"btc","instStatus":"normal","chains":[{"depositStatus":"allowed","withdrawStatus":"prohibited"},{"depositStatus":"prohibited","withdrawStatus":"allowed"},null]},
			{"currency":"bad","instStatus":"delisted","chains":[]}
		]
	}`, nil)
	require.NoError(t, h.UpdateCurrencyStates(t.Context(), asset.Spot), "UpdateCurrencyStates must not error")
	assert.NoError(t, h.CanTrade(currency.BTC, asset.Spot), "BTC trading should be enabled")
	assert.NoError(t, h.CanDeposit(currency.BTC, asset.Spot), "BTC deposits should be enabled when one chain allows deposits")
	assert.NoError(t, h.CanWithdraw(currency.BTC, asset.Spot), "BTC withdrawals should be enabled when one chain allows withdrawals")
	assert.Error(t, h.CanTrade(currency.NewCode("BAD"), asset.Spot), "delisted currency trading should be disabled")
	assert.Error(t, h.CanDeposit(currency.NewCode("BAD"), asset.Spot), "currency without chains should have deposits disabled")
	assert.Error(t, h.CanWithdraw(currency.NewCode("BAD"), asset.Spot), "currency without chains should have withdrawals disabled")
	for _, a := range []asset.Item{asset.Futures, asset.CoinMarginedFutures, asset.USDTMarginedFutures} {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, h.UpdateCurrencyStates(t.Context(), a), common.ErrNotYetImplemented, "derivatives must use default currency states")
		})
	}
	require.ErrorIs(t, h.UpdateCurrencyStates(t.Context(), asset.Binary), asset.ErrNotSupported, "unsupported assets must remain rejected")
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name           string
		asset          asset.Item
		pair           currency.Pair
		invalidPair    currency.Pair
		expectedErrMsg string
	}
	for _, tc := range []testCase{
		{name: "spot", asset: asset.Spot, pair: currency.NewPairWithDelimiter("BTC", "USDT", "_"), invalidPair: currency.NewPairWithDelimiter("INV", "ALID", "-"), expectedErrMsg: "invalid symbol"},
		{name: "coin margined futures", asset: asset.CoinMarginedFutures, pair: currency.NewPairWithDelimiter("BTC", "USD", "_"), invalidPair: currency.NewPairWithDelimiter("INV", "ALID", "_"), expectedErrMsg: "symbol data error"},
		{name: "delivery futures", asset: asset.Futures, pair: btccwPair},
		{name: "USDT margined futures", asset: asset.USDTMarginedFutures, pair: btcusdtPair},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "exchange setup must succeed")
			h.Name = t.Name()
			if tc.asset == asset.CoinMarginedFutures || tc.asset == asset.Futures {
				require.NoError(t, limits.Load([]limits.MinMaxLevel{{Key: key.NewExchangeAssetPair(h.Name, tc.asset, tc.pair), MultiplierDecimal: 100}}), "inverse multiplier must be loaded")
			}
			if tc.expectedErrMsg != "" {
				_, err := h.UpdateTicker(t.Context(), tc.invalidPair, tc.asset)
				assert.ErrorContains(t, err, tc.expectedErrMsg, "invalid pair should return the venue error")
			}
			_, err := h.UpdateTicker(t.Context(), tc.pair, tc.asset)
			require.NoError(t, err, "UpdateTicker must accept the asset's enabled pair")
		})
	}
	type tickerResponseCase struct {
		name                       string
		a                          asset.Item
		pair                       currency.Pair
		endpoint                   exchange.URL
		body                       string
		wantBase, wantQuote        float64
		wantBid, wantBidSize       float64
		wantLastUpdatedFromPayload bool
	}
	for _, tc := range []tickerResponseCase{
		{
			// trimmed from GET /market/detail/merged?symbol=btcusdt
			name: "spot records the quote turnover and the book", a: asset.Spot,
			pair: currency.NewBTCUSDT(), endpoint: exchange.RestSpot,
			body:     `{"status":"ok","ts":1788926189000,"tick":{"amount":6454.5661,"vol":506778563.47,"high":79000,"low":78000,"open":78500,"close":78616.33,"bid":[78615,1.5],"ask":[78617,2.5]}}`,
			wantBase: 6454.5661, wantQuote: 506778563.47, wantBid: 78615, wantBidSize: 1.5,
			wantLastUpdatedFromPayload: true,
		},
		{
			// trimmed from GET /swap-ex/market/detail/merged?contract_code=BTC-USD, where vol is
			// 65744 contracts of 100 USD against amount 83.81 BTC
			name: "coin margined records quote turnover", a: asset.CoinMarginedFutures,
			pair: currency.NewBTCUSD(), endpoint: exchange.RestFutures,
			body:     `{"status":"ok","ts":1788926189000,"tick":{"amount":"83.8102","vol":"65744","high":"79434.2","low":"77567.2","open":"79212.4","close":"78741.1","bid":[78740,3600],"ask":[78742,1200]}}`,
			wantBase: 83.8102, wantQuote: 6574400, wantBid: 78740, wantBidSize: 3600,
		},
		{
			// trimmed from GET /swap-ex/market/detail/merged?contract_code=OAS-USD, one of the
			// coin margined swaps with nothing resting on either side, which the batch endpoint
			// records with a zero book
			name: "coin margined records an empty book", a: asset.CoinMarginedFutures,
			pair: currency.NewPair(currency.NewCode("OAS"), currency.USD), endpoint: exchange.RestFutures,
			body: `{"ch":"market.OAS-USD.detail.merged","status":"ok","tick":{"amount":"0","ask":null,"bid":null,"close":"0.02","count":0,"high":"0.02","id":1789024695,"low":"0.02","open":"0.02","ts":1789024695555,"vol":"0"},"ts":1789024695555}`,
		},
		{
			// A side carrying only its price is recorded with a zero size, as on the futures paths
			name: "spot records a side carrying only its price", a: asset.Spot,
			pair: currency.NewPair(currency.ETH, currency.USDT), endpoint: exchange.RestSpot,
			body:     `{"status":"ok","ts":1788926189000,"tick":{"amount":1.5,"vol":3712.5,"high":2500,"low":2400,"open":2450,"close":2475,"bid":[2474.5],"ask":[2475.5,2]}}`,
			wantBase: 1.5, wantQuote: 3712.5, wantBid: 2474.5,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, err := fmt.Fprint(w, tc.body)
				assert.NoError(t, err, "writing the ticker response should not error")
			}))
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			ex.Name = t.Name()
			if tc.a == asset.CoinMarginedFutures || tc.a == asset.Futures {
				require.NoError(t, limits.Load([]limits.MinMaxLevel{{Key: key.NewExchangeAssetPair(ex.Name, tc.a, tc.pair), MultiplierDecimal: 100}}), "inverse multiplier must be loaded")
			}
			require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(tc.endpoint.String(), server.URL), "SetRunningURL must not error")

			got, err := ex.UpdateTicker(t.Context(), tc.pair, tc.a)
			require.NoError(t, err, "UpdateTicker must not error")
			assert.Equal(t, tc.wantBase, got.BaseVolume, "amount should be recorded as the base volume")
			assert.Equal(t, tc.wantQuote, got.QuoteVolume, "only an endpoint serving the quote currency should set a quote volume")
			assert.Equal(t, got.Close, got.Last, "Last should carry the latest price, as Close does")
			assert.Equal(t, tc.wantBid, got.Bid, "the resting bid should reach the store")
			assert.Equal(t, tc.wantBidSize, got.BidSize, "the resting bid size should reach the store")
			if tc.wantLastUpdatedFromPayload {
				assert.Equal(t, int64(1788926189000), got.LastUpdated.UnixMilli(), "the response timestamp should reach the store")
			}
		})
	}
	t.Run("USDT quote turnover", func(t *testing.T) {
		t.Parallel()
		h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/linear-swap-ex/market/detail/merged", `{"status":"ok","tick":{"amount":"10","trade_turnover":"747978922.8898","vol":"100","open":"1","close":"2","high":"3","low":"1","bid":[1,2],"ask":[2,3]}}`, nil)
		h.Name = t.Name()
		price, err := h.UpdateTicker(t.Context(), btcusdtPair, asset.USDTMarginedFutures)
		require.NoError(t, err, "single USDT ticker must decode")
		assert.Equal(t, 747978922.8898, price.QuoteVolume, "single ticker should retain quoted turnover")
	})
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name        string
		asset       asset.Item
		pair        currency.Pair
		expectedErr error
	}
	for _, tc := range []testCase{
		{name: "spot", asset: asset.Spot, pair: btcusdtPair},
		{name: "coin margined futures", asset: asset.CoinMarginedFutures, pair: btcusdPair},
		{name: "delivery futures", asset: asset.Futures, pair: btccwPair},
		{name: "USDT margined futures", asset: asset.USDTMarginedFutures, pair: btcusdtPair},
		{name: "unsupported asset", asset: asset.Binary, pair: btcusdtPair, expectedErr: asset.ErrNotSupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.UpdateOrderbook(t.Context(), tc.pair, tc.asset)
			require.ErrorIs(t, err, tc.expectedErr, "UpdateOrderbook must return the expected asset result")
		})
	}
}

func TestUpdateOrderbookWithLimit(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok","tick":{"ts":1604312615051,"bids":[[10,2],[9,3]],"asks":[[11,2],[12,3]]}}`))
	}))
	t.Cleanup(server.Close)

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.Name = t.Name()
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "spot endpoint must be set")
	book, err := h.UpdateOrderbookWithLimit(t.Context(), btcusdtPair, asset.Spot, 1)
	require.NoError(t, err, "UpdateOrderbookWithLimit must not error")
	assert.Len(t, book.Bids, 1, "bid depth should be capped")
	assert.Len(t, book.Asks, 1, "ask depth should be capped")
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	t.Run("missing spot pairs", func(t *testing.T) {
		t.Parallel()
		_, err := new(Exchange).GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide})
		require.ErrorIs(t, err, errCurrencyNotSupplied, "spot requests must supply pairs")
	})
	type testCase struct {
		item           asset.Item
		pair           currency.Pair
		path, contract string
	}
	for _, tc := range []testCase{
		{asset.Futures, btccwPair, "/api/v3/contract_hisorders", "BTC_CW"},
		{asset.CoinMarginedFutures, btcusdPair, "/swap-api/v3/swap_hisorders", "BTC-USD"},
	} {
		t.Run(tc.item.String()+" window overlap", func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, tc.path, `{"code":200,"data":[{"query_id":1,"order_id_str":"1","contract_code":"`+tc.contract+`","direction":"buy","order_price_type":6,"status":6,"create_date":1782867600000}]}`, func(r *http.Request) {
				var payload map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&payload), "history request must decode")
				calls.Add(1)
			})
			start := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
			orders, err := h.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: tc.item, Type: order.AnyType, Side: order.AnySide, Pairs: currency.Pairs{tc.pair}, StartTime: start, EndTime: start.Add(72 * time.Hour)})
			require.NoError(t, err, "history windows must decode")
			assert.Equal(t, int32(2), calls.Load(), "each 48-hour window should be requested")
			require.Len(t, orders, 1, "overlapping history rows must be returned once")
			if tc.item == asset.Futures {
				assert.Equal(t, time.UnixMilli(1782867600000), orders[0].Date, "delivery creation date should be retained")
			}
		})
	}
	type orderTypeCase struct {
		requestType string
		timeInForce order.TimeInForce
	}
	for _, tc := range []orderTypeCase{
		{requestType: "buy-limit-maker", timeInForce: order.PostOnly},
		{requestType: "buy-ioc", timeInForce: order.ImmediateOrCancel},
		{requestType: "buy-limit-fok", timeInForce: order.FillOrKill},
	} {
		t.Run(tc.requestType, func(t *testing.T) {
			t.Parallel()
			response := fmt.Sprintf(`{"status":"ok","data":[{"id":1,"symbol":"btcusdt","amount":"2","price":"100","type":%q,"state":"filled"}]}`, tc.requestType)
			h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v1/order/orders", response, nil)
			orders, err := h.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Pairs: currency.Pairs{btcusdtPair}})
			require.NoError(t, err, "spot orders must decode")
			require.Len(t, orders, 1, "special limit orders must survive side and type filtering")
			assert.Equal(t, order.Buy, orders[0].Side, "order side should match")
			assert.Equal(t, order.Limit, orders[0].Type, "order type should match")
			assert.Equal(t, tc.timeInForce, orders[0].TimeInForce, "time in force should match")
		})
	}

	t.Run("V5 pages", func(t *testing.T) {
		t.Parallel()
		const path, state = "/v5/trade/order/history", "filled"
		var mu sync.Mutex
		var cursors []string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !assert.Equal(t, path, r.URL.Path, "pager path should match") {
				return
			}
			from := r.URL.Query().Get("from")
			mu.Lock()
			cursors = append(cursors, from)
			calls := len(cursors)
			mu.Unlock()
			var high, low int
			switch {
			case calls > 2:
				http.Error(w, "unexpected extra page", http.StatusBadRequest)
				return
			case from == "":
				high, low = 300, 201
			case from == "201":
				high, low = 200, 196
			default:
				http.Error(w, "unexpected cursor "+from, http.StatusBadRequest)
				return
			}
			rows := make([]string, 0, high-low+1)
			for id := high; id >= low; id-- {
				rows = append(rows, fmt.Sprintf(`{"id":"%d","order_id":"%d","contract_code":"BTC-USDT","side":"buy","type":"limit","state":%q,"time_in_force":"gtc","margin_mode":"cross","price":"1","volume":"1","trade_volume":"1"}`, id, id, state))
			}
			_, _ = w.Write([]byte(`{"code":200,"data":[` + strings.Join(rows, ",") + `]}`))
		}))
		t.Cleanup(server.Close)
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "HTX setup must succeed")
		h.API.AuthenticatedSupport = true
		h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
		require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "mock endpoint must be set")
		req := &order.MultiOrderRequest{AssetType: asset.USDTMarginedFutures, Type: order.AnyType, Side: order.AnySide, Pairs: currency.Pairs{btcusdtPair}, MarginType: margin.Multi}
		got, err := h.GetOrderHistory(t.Context(), req)
		require.NoError(t, err, "paged orders must be fetched")
		assert.Len(t, got, 105, "every page should be returned")
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"", "201"}, cursors, "each page should start after the previous page's last id")
	})
	type optionalFieldCase struct {
		name, fields string
	}
	for _, tc := range []optionalFieldCase{
		{name: "documented response example", fields: `"field-amount":"1","field-cash-amount":"99.5","field-fees":"0.1"`},
		{name: "documented response table", fields: `"filled-amount":"1","filled-cash-amount":"99.5","filled-fees":"0.1"`},
	} {
		t.Run("spot "+tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v1/order/orders", `{"status":"ok","data":[{"id":1,"symbol":"btcusdt","amount":"2","price":"100","created-at":1604367611263,"finished-at":1604367700000,"type":"buy-limit","state":"partial-canceled",`+tc.fields+`}]}`, func(r *http.Request) {
				assert.Equal(t, "partial-canceled,filled,canceled", r.URL.Query().Get("states"), "history should request every final state")
			})
			orders, err := h.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide, Pairs: currency.Pairs{btcusdtPair}})
			require.NoError(t, err, "spot history must decode either documented field spelling")
			require.Len(t, orders, 1, "spot history must return one order")
			assert.Equal(t, 1.0, orders[0].ExecutedAmount, "executed amount should match")
			assert.Equal(t, 1.0, orders[0].RemainingAmount, "remaining amount should exclude executions")
			assert.Equal(t, 99.5, orders[0].Cost, "cost should match the executed quote amount")
			assert.Equal(t, 0.1, orders[0].Fee, "fee should match")
			assert.Equal(t, currency.USDT, orders[0].CostAsset, "spot turnover should use the quote currency")
			assert.Equal(t, time.UnixMilli(1604367700000), orders[0].CloseTime, "close time should use the completion timestamp")
		})
	}
	t.Run("validation", func(t *testing.T) {
		t.Parallel()
		getOrdersRequest := order.MultiOrderRequest{
			AssetType: asset.Spot,
			Type:      order.AnyType,
			Side:      order.AnySide,
		}
		_, err := e.GetOrderHistory(t.Context(), &getOrdersRequest)
		require.ErrorContains(t, err, "currency must be supplied", "GetOrderHistory must require a currency pair for spot")

		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "Setup must not error")
		type testCase struct {
			name        string
			pair        currency.Pair
			asset       asset.Item
			expectedErr error
		}
		testCases := []testCase{
			{name: "spot", pair: btcusdtPair, asset: asset.Spot, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
			{name: "coin margined futures", pair: btcusdPair, asset: asset.CoinMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
			{name: "usdt margined futures", pair: btcusdtPair, asset: asset.USDTMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
			{name: "futures", pair: btccwPair, asset: asset.Futures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
			{name: "unsupported asset", pair: btcusdtPair, asset: asset.Binary, expectedErr: asset.ErrNotSupported},
		}
		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := h.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
					Type:      order.AnyType,
					Pairs:     []currency.Pair{tt.pair},
					AssetType: tt.asset,
					Side:      order.AnySide,
					StartTime: time.Now().AddDate(0, 0, -1),
					EndTime:   time.Now(),
				})
				require.ErrorIs(t, err, tt.expectedErr, "GetOrderHistory must return the expected branch error")
			})
		}
	})
	type contractHistoryCase struct {
		name       string
		item       asset.Item
		pair       currency.Pair
		path       string
		contract   string
		extraEntry map[string]any
	}
	for _, tc := range []contractHistoryCase{
		{
			name:     "coin margined futures",
			item:     asset.CoinMarginedFutures,
			pair:     btcusdPair,
			path:     "/swap-api/v3/swap_hisorders",
			contract: "BTC-USD",
		},
		{
			name:     "delivery futures",
			item:     asset.Futures,
			pair:     btccwPair,
			path:     "/api/v3/contract_hisorders",
			contract: "BTC_CW",
			extraEntry: map[string]any{
				"create_date": time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tc.path, r.URL.Path, "history endpoint path should match")
				payload, err := io.ReadAll(r.Body)
				assert.NoError(t, err, "request body should be readable")
				var requestBody map[string]any
				assert.NoError(t, json.Unmarshal(payload, &requestBody), "request body should decode")
				assert.Equal(t, v3HistoryDirectionNext, requestBody["direct"], "history pagination should move forwards")
				cursor, _ := requestBody["from_id"].(float64)
				entries := make([]map[string]any, 0, 50)
				switch int64(cursor) {
				case 0:
					for queryID := int64(1); queryID <= 50; queryID++ {
						entry := map[string]any{
							"fee":              -0.1,
							"query_id":         queryID,
							"order_id":         queryID,
							"order_id_str":     strconv.FormatInt(queryID, 10),
							"contract_code":    tc.contract,
							"direction":        "buy",
							"order_price_type": 6,
							"status":           6,
						}
						maps.Copy(entry, tc.extraEntry)
						entries = append(entries, entry)
					}
				case 50:
					entry := map[string]any{
						"fee":              -0.1,
						"query_id":         int64(51),
						"order_id":         int64(51),
						"order_id_str":     "51",
						"contract_code":    tc.contract,
						"direction":        "buy",
						"order_price_type": 6,
						"status":           6,
					}
					maps.Copy(entry, tc.extraEntry)
					entries = append(entries, entry)
				}
				response, err := json.Marshal(map[string]any{"code": 200, "data": entries})
				assert.NoError(t, err, "response body should encode")
				_, _ = w.Write(response)
				calls.Add(1)
			}))
			t.Cleanup(server.Close)

			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must not error")
			h.API.AuthenticatedSupport = true
			h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "futures endpoint must be set")
			startTime := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
			orders, err := h.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
				Type:      order.AnyType,
				Pairs:     []currency.Pair{tc.pair},
				AssetType: tc.item,
				Side:      order.AnySide,
				StartTime: startTime,
				EndTime:   startTime.Add(24 * time.Hour),
			})
			require.NoError(t, err, "GetOrderHistory must not error")
			require.Len(t, orders, 51, "all cursor pages must be returned")
			for _, detail := range orders {
				assert.Equal(t, 0.1, detail.Fee, "history fees should use GCT charge signs")
			}
			assert.Equal(t, order.PostOnly, orders[0].TimeInForce, "numeric history type should retain post-only protection")
			assert.Equal(t, int64(2), calls.Load(), "pagination should stop after the short final page")
		})
	}
	t.Run("USDT margined futures", func(t *testing.T) {
		t.Parallel()
		var calls atomic.Int64
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v5/trade/order/history", r.URL.Path, "history endpoint path should match")
			assert.Equal(t, "next", r.URL.Query().Get("direct"), "history pagination should move forwards")
			start, err := strconv.ParseInt(r.URL.Query().Get("start_time"), 10, 64)
			if !assert.NoError(t, err, "history start time should be valid") {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			end, err := strconv.ParseInt(r.URL.Query().Get("end_time"), 10, 64)
			if !assert.NoError(t, err, "history end time should be valid") {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			assert.LessOrEqual(t, end-start, int64(48*time.Hour/time.Millisecond), "V5 history windows should not exceed 48 hours")
			calls.Add(1)
			if r.URL.Query().Get("margin_mode") == "cross" {
				_, _ = w.Write([]byte(`{"code":200,"data":[{"id":"1","contract_code":"BTC-USDT","order_id":"1","side":"buy","type":"limit","state":"filled","time_in_force":"rpi","margin_mode":"cross","price":"100","volume":"2","trade_volume":"1"}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"code":200,"data":[]}`))
		}))
		t.Cleanup(server.Close)
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "HTX setup must not error")
		h.API.AuthenticatedSupport = true
		h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
		require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT-margined endpoint must be set")
		startTime := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
		orders, err := h.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
			Type:      order.AnyType,
			Pairs:     currency.Pairs{btcusdtPair},
			AssetType: asset.USDTMarginedFutures,
			Side:      order.AnySide,
			StartTime: startTime,
			EndTime:   startTime.Add(49 * time.Hour),
		})
		require.NoError(t, err, "GetOrderHistory must not error")
		require.Len(t, orders, 1, "one historical order must be returned")
		assert.Equal(t, "1", orders[0].OrderID, "order ID should match")
		assert.Equal(t, int64(4), calls.Load(), "each history window should query cross and isolated margin modes")
	})
}

func TestGetV3HistoryWindows(t *testing.T) {
	t.Parallel()
	for _, span := range []time.Duration{48 * time.Hour, 48*time.Hour + time.Millisecond, 96*time.Hour + 2*time.Millisecond} {
		t.Run(span.String(), func(t *testing.T) {
			t.Parallel()
			start := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
			windows, err := getV3HistoryWindows(start, start.Add(span))
			require.NoError(t, err, "windows must succeed")
			for i, window := range windows {
				assert.True(t, window.start.Before(window.end), "each window should have a positive duration")
				if i > 0 {
					assert.Equal(t, windows[i-1].end, window.start, "windows should share boundaries")
				}
			}
			assert.Equal(t, start.Add(span), windows[len(windows)-1].end, "windows should cover the full requested range")
		})
	}
	t.Run("equal times", func(t *testing.T) {
		t.Parallel()
		now := time.Now()
		_, err := getV3HistoryWindows(now, now)
		require.ErrorIs(t, err, common.ErrStartEqualsEnd, "equal dates must be rejected")
	})
	startTime := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	endTime := startTime.Add(5 * 24 * time.Hour)
	windows, err := getV3HistoryWindows(startTime, endTime)
	require.NoError(t, err, "getV3HistoryWindows must not error")
	require.Len(t, windows, 3, "five days must be split into three windows")
	for x := range windows {
		assert.LessOrEqual(t, windows[x].end.Sub(windows[x].start), 48*time.Hour, "each window should be at most 48 hours")
	}
	assert.Equal(t, startTime, windows[0].start, "first window should preserve the requested start")
	assert.Equal(t, endTime, windows[len(windows)-1].end, "last window should preserve the requested end")
	assert.Equal(t, windows[0].end, windows[1].start, "adjacent ranges should be contiguous")

	windows, err = getV3HistoryWindows(time.Time{}, time.Time{})
	require.NoError(t, err, "getV3HistoryWindows must accept an unspecified interval")
	require.Len(t, windows, 1, "an unspecified interval must produce one request")
	assert.True(t, windows[0].start.IsZero(), "unspecified start should remain zero")
	assert.True(t, windows[0].end.IsZero(), "unspecified end should remain zero")

	_, err = getV3HistoryWindows(endTime, startTime)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "getV3HistoryWindows must reject reversed intervals")
	_, err = getV3HistoryWindows(startTime, startTime.Add(91*24*time.Hour))
	require.ErrorIs(t, err, errInvalidCreateDate, "getV3HistoryWindows must reject intervals over 90 days")
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name, response string
		wantErr        error
	}
	for _, tc := range []testCase{
		{name: "API rejection", response: `{"code":400,"message":"cancel unavailable"}`, wantErr: request.ErrAuthRequestFailed},
		{name: "null response", response: `null`, wantErr: errEmptyResult},
	} {
		t.Run("USDT fanout after "+tc.name, func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var requested []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method, "cancel-all should use POST")
				assert.Equal(t, "/v5/trade/cancel_all_orders", r.URL.Path, "cancel-all endpoint should match")
				var payload V5CancelAllOrdersRequest
				if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload), "cancel request should decode") {
					return
				}
				mu.Lock()
				requested = append(requested, payload.ContractCode)
				mu.Unlock()
				if payload.ContractCode == "BTC-USDT" {
					_, _ = w.Write([]byte(tc.response))
					return
				}
				_, _ = w.Write([]byte(`{"code":200,"data":[{"order_id":"8"},{"client_order_id":"77","code":200},{"order_id":"9","code":1061,"message":"not found"}]}`))
			}))
			t.Cleanup(server.Close)
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "exchange setup must succeed")
			h.API.AuthenticatedSupport = true
			h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "cancel endpoint must use the mock server")
			pairs := currency.Pairs{btcusdtPair, currency.NewPair(currency.ETH, currency.USDT), currency.NewPair(currency.LTC, currency.USDT)}
			require.NoError(t, h.SetPairs(pairs, asset.USDTMarginedFutures, false), "available pairs must include the disabled pair")
			require.NoError(t, h.SetPairs(pairs[:2], asset.USDTMarginedFutures, true), "only the first two pairs must be enabled")
			response, err := h.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.USDTMarginedFutures})
			require.ErrorIs(t, err, tc.wantErr, "cancel-all must retain the failed request")
			assert.Equal(t, map[string]string{"8": htxStatusSuccess, "77": htxStatusSuccess, "9": "not found"}, response.Status, "later cancellations should retain every acknowledgement")
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, []string{"BTC-USDT", "ETH-USDT"}, requested, "cancel-all should attempt every enabled pair after a failure")
		})
	}
	type cancellationRequestCase struct {
		name     string
		item     asset.Item
		pair     currency.Pair
		endpoint exchange.URL
		path     string
		response string
		want     map[string]string
	}
	for _, tc := range []cancellationRequestCase{
		{
			name: "coin margined", item: asset.CoinMarginedFutures, pair: btcusdPair, endpoint: exchange.RestFutures, path: "/swap-api/v1/swap_cancelall",
			response: `{"status":"ok","data":{"errors":[{"order_id":"9","err_code":1061,"err_msg":"not found"}],"successes":"8"},"ts":1700000000000}`,
			want:     map[string]string{"8": htxStatusSuccess, "9": "fail: not found"},
		},
		{
			name: "delivery", item: asset.Futures, pair: currency.NewPair(currency.BTC, currency.NewCode("261016")), endpoint: exchange.RestFutures, path: "/api/v1/contract_cancelall",
			response: `{"status":"ok","data":{"errors":[{"order_id":"9","err_code":1061,"err_msg":"not found"}],"successes":"8"},"ts":1700000000000}`,
			want:     map[string]string{"8": htxStatusSuccess, "9": "fail: not found"},
		},
		{
			name: "USDT margined", item: asset.USDTMarginedFutures, pair: btcusdtPair, endpoint: exchange.RestUSDTMargined, path: "/v5/trade/cancel_all_orders",
			response: `{"code":200,"data":[{"order_id":"8","code":200,"message":"Success"},{"order_id":"9","code":1061,"message":"not found"}]}`,
			want:     map[string]string{"8": htxStatusSuccess, "9": "not found"},
		},
	} {
		for _, allPairs := range []bool{false, true} {
			t.Run(tc.name+map[bool]string{false: " pair", true: " all pairs"}[allPairs], func(t *testing.T) {
				t.Parallel()
				var requested []string
				h := newHTTPTestExchange(t, tc.endpoint, http.MethodPost, tc.path, tc.response, func(r *http.Request) {
					var payload map[string]any
					if assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload), "cancel-all payload should decode") {
						code, ok := payload["contract_code"].(string)
						assert.True(t, ok, "cancel-all should explicitly restrict the contract")
						requested = append(requested, code)
					}
				})
				otherPair := currency.NewPair(currency.ETH, tc.pair.Quote)
				pairs := currency.Pairs{tc.pair, otherPair}
				require.NoError(t, h.SetPairs(pairs, tc.item, false), "both contracts must be available")
				require.NoError(t, h.SetPairs(pairs, tc.item, true), "both contracts must be enabled")
				cancel := &order.Cancel{Exchange: h.Name, AssetType: tc.item, Pair: tc.pair}
				if allPairs {
					cancel.Pair = currency.EMPTYPAIR
				}
				resp, err := h.CancelAllOrders(t.Context(), cancel)
				require.NoError(t, err, "cancel-all must not error")
				assert.Equal(t, tc.want, resp.Status, "cancel-all should report failed cancellations as failures")
				expectedCode, err := h.FormatSymbol(tc.pair, tc.item)
				require.NoError(t, err, "selected contract must format")
				expected := []string{expectedCode}
				if allPairs {
					otherCode, err := h.FormatSymbol(otherPair, tc.item)
					require.NoError(t, err, "other contract must format")
					expected = append(expected, otherCode)
				}
				assert.Equal(t, expected, requested, "explicit-pair cancellation should never cancel another enabled contract")
			})
		}
	}
	t.Run("validation", func(t *testing.T) {
		t.Parallel()
		_, err := e.CancelAllOrders(t.Context(), nil)
		require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelAllOrders must reject nil cancellations")

		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "Setup must not error")
		type testCase struct {
			name        string
			pair        currency.Pair
			asset       asset.Item
			expectedErr error
		}
		testCases := []testCase{
			{name: "spot", pair: btcusdtPair, asset: asset.Spot, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
			{name: "coin margined futures", pair: btcusdPair, asset: asset.CoinMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
			{name: "usdt margined futures", pair: btcusdtPair, asset: asset.USDTMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
			{name: "futures", pair: btccwPair, asset: asset.Futures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
			{name: "unsupported asset", pair: btcusdtPair, asset: asset.Binary, expectedErr: asset.ErrNotSupported},
		}
		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := h.CancelAllOrders(t.Context(), &order.Cancel{
					OrderID:   "1",
					Pair:      tt.pair,
					AssetType: tt.asset,
				})
				require.ErrorIs(t, err, tt.expectedErr, "CancelAllOrders must return the expected branch error")
			})
		}
	})
	t.Run("spot incomplete cancellation", func(t *testing.T) {
		t.Parallel()
		h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodPost, "/v1/order/orders/batchCancelOpenOrders", `{"status":"ok","data":{"success-count":100,"failed-count":0,"next-id":5454600}}`, nil)
		_, err := h.CancelAllOrders(t.Context(), &order.Cancel{AccountID: "1", Pair: btcusdtPair, AssetType: asset.Spot})
		assert.ErrorIs(t, err, errOrderCancellationFailed, "remaining open orders should prevent complete cancellation success")
		assert.ErrorContains(t, err, "5454600", "incomplete result should retain the next order identifier")
	})

	type cancellationResponseCase struct {
		name     string
		endpoint exchange.URL
		path     string
		cancel   order.Cancel
		response string
	}
	for _, tt := range []cancellationResponseCase{
		{
			name: "spot", endpoint: exchange.RestSpot, path: "/v1/order/orders/batchCancelOpenOrders",
			cancel:   order.Cancel{AccountID: "1", Pair: btcusdtPair, AssetType: asset.Spot},
			response: `{"status":"ok","data":{"success-count":1,"failed-count":0,"next-id":-1}}`,
		},
		{
			name: "coin margined futures", endpoint: exchange.RestFutures, path: "/swap-api/v1/swap_cancelall",
			cancel:   order.Cancel{Pair: btcusdPair, AssetType: asset.CoinMarginedFutures},
			response: `{"status":"ok","data":{"successes":"1","errors":[]}}`,
		},
		{
			name: "USDT margined futures", endpoint: exchange.RestUSDTMargined, path: "/v5/trade/cancel_all_orders",
			cancel:   order.Cancel{Pair: btcusdtPair, AssetType: asset.USDTMarginedFutures},
			response: `{"code":200,"data":[{"order_id":"1"}]}`,
		},
		{
			name: "delivery futures", endpoint: exchange.RestFutures, path: "/api/v1/contract_cancelall",
			cancel:   order.Cancel{Pair: btccwPair, AssetType: asset.Futures},
			response: `{"status":"ok","data":{"successes":"1","errors":[]}}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, tt.endpoint, http.MethodPost, tt.path, tt.response, nil)
			resp, err := h.CancelAllOrders(t.Context(), &tt.cancel)
			require.NoError(t, err, "CancelAllOrders must not error")
			if tt.cancel.AssetType != asset.Spot {
				assert.Equal(t, htxStatusSuccess, resp.Status["1"], "cancellation status should indicate success")
			}
		})
	}

	t.Run("spot fanout combines errors and attempts every pair", func(t *testing.T) {
		t.Parallel()
		var calls atomic.Int32
		h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodPost, "/v1/order/orders/batchCancelOpenOrders", `{"status":"ok","data":{"success-count":0,"failed-count":1}}`, func(*http.Request) {
			calls.Add(1)
		})
		pairs := currency.Pairs{btcusdtPair, currency.NewPair(currency.ETH, currency.USDT)}
		require.NoError(t, h.SetPairs(pairs, asset.Spot, false), "available spot pairs must be set")
		require.NoError(t, h.SetPairs(pairs, asset.Spot, true), "enabled spot pairs must be set")
		_, err := h.CancelAllOrders(t.Context(), &order.Cancel{AccountID: "1", AssetType: asset.Spot})
		require.ErrorIs(t, err, errOrderCancellationFailed, "CancelAllOrders must retain cancellation failures")
		assert.Equal(t, int32(2), calls.Load(), "CancelAllOrders should attempt every enabled pair")
	})
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	t.Run("USDT margined endpoint", func(t *testing.T) {
		t.Parallel()
		now := time.Now().UTC().Truncate(time.Minute)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/linear-swap-ex/market/history/kline", r.URL.Path, "USDT-margined kline path should match")
			_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":` + strconv.FormatInt(now.Add(-time.Minute).Unix(), 10) + `,"open":1,"high":2,"low":0.5,"close":1.5,"vol":10}]}`))
		}))
		t.Cleanup(server.Close)

		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "HTX setup must not error")
		require.NoError(t, h.SetPairs(currency.Pairs{btcusdtPair}, asset.USDTMarginedFutures, false), "available pair must be set")
		require.NoError(t, h.SetPairs(currency.Pairs{btcusdtPair}, asset.USDTMarginedFutures, true), "enabled pair must be set")
		require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT-margined endpoint must be set")

		got, err := h.GetHistoricCandles(t.Context(), btcusdtPair, asset.USDTMarginedFutures, kline.OneMin, now.Add(-2*time.Minute), now)
		require.NoError(t, err, "GetHistoricCandles must not error for USDT-margined futures")
		require.NotEmpty(t, got.Candles, "candles must be returned")
	})
	for _, a := range []asset.Item{asset.Futures, asset.CoinMarginedFutures, asset.USDTMarginedFutures} {
		for _, count := range []int{1999, 2000} {
			t.Run(a.String()+"/"+strconv.Itoa(count), func(t *testing.T) {
				t.Parallel()
				pair, path, endpoint := btcusdPair, "/swap-ex/market/history/kline", exchange.RestFutures
				switch a {
				case asset.Futures:
					pair, path = btccwPair, "/market/history/kline"
				case asset.USDTMarginedFutures:
					pair, path, endpoint = btcusdtPair, "/linear-swap-ex/market/history/kline", exchange.RestUSDTMargined
				}
				end := time.Now().UTC().Truncate(time.Minute).Add(-24 * time.Hour)
				start := end.Add(-time.Duration(count) * time.Minute)
				h := newHTTPTestExchange(t, endpoint, http.MethodGet, path, "", nil)
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					from, err := strconv.ParseInt(r.URL.Query().Get("from"), 10, 64)
					assert.NoError(t, err, "from should parse")
					to, err := strconv.ParseInt(r.URL.Query().Get("to"), 10, 64)
					assert.NoError(t, err, "to should parse")
					assert.Equal(t, start.Unix(), from, "start should remain inclusive")
					assert.Equal(t, end.Unix()-1, to, "exclusive end should convert to inclusive seconds")
					if to-from >= 2000*60 {
						_, _ = w.Write([]byte(`{"status":"error","err-msg":"history kline query with invalid from to"}`))
						return
					}
					data := make([]map[string]any, 0, count)
					for at := from; at <= to; at += 60 {
						data = append(data, map[string]any{"id": at, "open": 10, "close": 10, "high": 10, "low": 10, "vol": 2})
					}
					assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{"status": "ok", "data": data}), "candles should encode")
				}))
				t.Cleanup(server.Close)
				require.NoError(t, h.API.Endpoints.SetRunningURL(endpoint.String(), server.URL), "candle endpoint must be configured")
				require.NoError(t, h.SetPairs(currency.Pairs{pair}, a, false), "candle pair must be available")
				require.NoError(t, h.SetPairs(currency.Pairs{pair}, a, true), "candle pair must be enabled")
				candles, err := h.GetHistoricCandles(t.Context(), pair, a, kline.OneMin, start, end)
				require.NoError(t, err, "maximum candle request must succeed")
				require.Len(t, candles.Candles, count, "exclusive range must contain the requested candles")
				assert.Equal(t, end.Add(-time.Minute), candles.Candles[count-1].Time.UTC(), "last candle should start before exclusive end")
			})
		}
	}

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	updatePairsOnce(t, e)

	endTime := time.Now().Add(-time.Hour).Truncate(time.Hour)
	_, err := e.GetHistoricCandles(t.Context(), btcusdtPair, asset.Spot, kline.OneMin, endTime.Add(-time.Hour), endTime)
	require.NoError(t, err)

	_, err = e.GetHistoricCandles(t.Context(), btcusdtPair, asset.Spot, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	require.NoError(t, err)

	_, err = e.GetHistoricCandles(t.Context(), btcFutureDatedPair, asset.Futures, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	require.NoError(t, err)

	_, err = e.GetHistoricCandles(t.Context(), btcusdPair, asset.CoinMarginedFutures, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	require.NoError(t, err)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	t.Run("USDT margined endpoint", func(t *testing.T) {
		t.Parallel()
		now := time.Now().UTC().Truncate(time.Minute)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/linear-swap-ex/market/history/kline", r.URL.Path, "USDT-margined kline path should match")
			_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":` + strconv.FormatInt(now.Add(-time.Minute).Unix(), 10) + `,"open":1,"high":2,"low":0.5,"close":1.5,"vol":10}]}`))
		}))
		t.Cleanup(server.Close)

		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "HTX setup must not error")
		require.NoError(t, h.SetPairs(currency.Pairs{btcusdtPair}, asset.USDTMarginedFutures, false), "available pair must be set")
		require.NoError(t, h.SetPairs(currency.Pairs{btcusdtPair}, asset.USDTMarginedFutures, true), "enabled pair must be set")
		require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT-margined endpoint must be set")

		got, err := h.GetHistoricCandlesExtended(t.Context(), btcusdtPair, asset.USDTMarginedFutures, kline.OneMin, now.Add(-2*time.Minute), now)
		require.NoError(t, err, "GetHistoricCandlesExtended must not error for USDT-margined futures")
		require.NotEmpty(t, got.Candles, "extended candles must be returned")
	})
	type testCase struct {
		item     asset.Item
		pair     currency.Pair
		path     string
		endpoint exchange.URL
	}
	for _, tc := range []testCase{
		{asset.Futures, btccwPair, "/market/history/kline", exchange.RestFutures},
		{asset.CoinMarginedFutures, btcusdPair, "/swap-ex/market/history/kline", exchange.RestFutures},
		{asset.USDTMarginedFutures, btcusdtPair, "/linear-swap-ex/market/history/kline", exchange.RestUSDTMargined},
	} {
		t.Run(tc.item.String()+" full ranges", func(t *testing.T) {
			t.Parallel()
			start := time.Now().UTC().Add(-72 * time.Hour).Truncate(time.Minute)
			end := start.Add(36 * time.Hour)
			var calls atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tc.path, r.URL.Path, "candles should use the asset endpoint")
				from, err := strconv.ParseInt(r.URL.Query().Get("from"), 10, 64)
				assert.NoError(t, err, "from should parse")
				to, err := strconv.ParseInt(r.URL.Query().Get("to"), 10, 64)
				assert.NoError(t, err, "to should parse")
				assert.Equalf(t, int64(59), to%60, "exclusive range ends should become inclusive seconds")
				assert.Less(t, to-from, int64(2000*60), "inclusive requests should stay within the candle limit")
				candles := make([]map[string]any, 0, 2000)
				for at := from; at <= to; at += 60 {
					candles = append(candles, map[string]any{"id": at, "open": 10, "close": 10, "high": 10, "low": 10, "vol": 2})
				}
				payload, err := json.Marshal(map[string]any{"status": "ok", "data": candles})
				assert.NoError(t, err, "candles should encode")
				_, _ = w.Write(payload)
				calls.Add(1)
			}))
			t.Cleanup(server.Close)
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must succeed")
			require.NoError(t, h.API.Endpoints.SetRunningURL(tc.endpoint.String(), server.URL), "candle endpoint must be set")
			require.NoError(t, h.SetPairs(currency.Pairs{tc.pair}, tc.item, false), "candle pair must be available")
			require.NoError(t, h.SetPairs(currency.Pairs{tc.pair}, tc.item, true), "candle pair must be enabled")
			candles, err := h.GetHistoricCandlesExtended(t.Context(), tc.pair, tc.item, kline.OneMin, start, end)
			require.NoError(t, err, "extended candles must fetch multiple full ranges")
			assert.Greater(t, calls.Load(), int64(1), "extended request should need multiple calls")
			assert.Len(t, candles.Candles, 2160, "all requested candles should return once")
		})
	}

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	updatePairsOnce(t, e)

	endTime := time.Now().Add(-time.Hour).Truncate(time.Hour)
	_, err := e.GetHistoricCandlesExtended(t.Context(), btcusdtPair, asset.Spot, kline.OneMin, endTime.Add(-time.Hour), endTime)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)

	_, err = e.GetHistoricCandlesExtended(t.Context(), btcFutureDatedPair, asset.Futures, kline.OneDay, endTime.AddDate(0, 0, -7), endTime)
	require.NoError(t, err)

	// demonstrate that adjusting time doesn't wreck non-day intervals
	_, err = e.GetHistoricCandlesExtended(t.Context(), btcFutureDatedPair, asset.Futures, kline.OneHour, endTime.AddDate(0, 0, -1), endTime)
	require.NoError(t, err)

	_, err = e.GetHistoricCandlesExtended(t.Context(), btcusdPair, asset.CoinMarginedFutures, kline.OneDay, endTime.AddDate(0, 0, -7), time.Now())
	require.NoError(t, err)

	_, err = e.GetHistoricCandlesExtended(t.Context(), btcusdPair, asset.CoinMarginedFutures, kline.OneHour, endTime.AddDate(0, 0, -1), time.Now())
	require.NoError(t, err)
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	st, err := e.GetServerTime(t.Context(), asset.Spot)
	require.NoError(t, err)
	assert.NotEmpty(t, st, "GetServerTime should return a time")
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoWithSetupText + " & " + exchange.NoFiatWithdrawalsText
	withdrawPermissions := e.FormatWithdrawPermissions()
	assert.Equal(t, expectedResult, withdrawPermissions)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	t.Run("missing spot pairs", func(t *testing.T) {
		t.Parallel()
		_, err := new(Exchange).GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.AnyType, Side: order.AnySide})
		require.ErrorIs(t, err, errCurrencyNotSupplied, "spot requests must supply pairs")
	})
	type testCase struct {
		requestType string
		timeInForce order.TimeInForce
	}
	for _, tc := range []testCase{
		{requestType: "buy-limit-maker", timeInForce: order.PostOnly},
		{requestType: "buy-ioc", timeInForce: order.ImmediateOrCancel},
		{requestType: "buy-limit-fok", timeInForce: order.FillOrKill},
	} {
		t.Run(tc.requestType, func(t *testing.T) {
			t.Parallel()
			response := fmt.Sprintf(`{"status":"ok","data":[{"id":1,"symbol":"btcusdt","amount":"2","price":"100","type":%q,"state":"submitted"}]}`, tc.requestType)
			h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v1/order/openOrders", response, nil)
			orders, err := h.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Pairs: currency.Pairs{btcusdtPair}})
			require.NoError(t, err, "spot orders must decode")
			require.Len(t, orders, 1, "special limit orders must survive side and type filtering")
			assert.Equal(t, order.Buy, orders[0].Side, "order side should match")
			assert.Equal(t, order.Limit, orders[0].Type, "order type should match")
			assert.Equal(t, tc.timeInForce, orders[0].TimeInForce, "time in force should match")
		})
	}

	type marginScopeCase struct {
		name       string
		marginType margin.Type
		expected   []string
	}
	for _, tc := range []marginScopeCase{
		{name: "cross", marginType: margin.Multi, expected: []string{"cross"}},
		{name: "isolated", marginType: margin.Isolated, expected: []string{"isolated"}},
		{name: "all margin modes", marginType: margin.Unset, expected: []string{"cross", "isolated"}},
	} {
		t.Run("all contracts "+tc.name, func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var modes []string
			h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/trade/order/opens", `{"code":200,"data":[]}`, func(r *http.Request) {
				assert.NotContains(t, r.URL.Query(), "contract_code", "all-contract query should omit the pair filter")
				mu.Lock()
				modes = append(modes, r.URL.Query().Get("margin_mode"))
				mu.Unlock()
			})
			orders, err := h.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.USDTMarginedFutures, MarginType: tc.marginType, Side: order.AnySide, Type: order.AnyType})
			require.NoError(t, err, "all-contract lookup must support each accepted margin selection")
			assert.Empty(t, orders, "empty API pages should remain empty")
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, tc.expected, modes, "all-contract lookup should preserve the requested margin scope")
		})
	}

	for _, allPairs := range []bool{false, true} {
		t.Run(fmt.Sprintf("V5 pages all pairs %t", allPairs), func(t *testing.T) {
			t.Parallel()
			const path, state = "/v5/trade/order/opens", "new"
			var mu sync.Mutex
			var cursors []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !assert.Equal(t, path, r.URL.Path, "pager path should match") {
					return
				}
				if allPairs {
					assert.NotContains(t, r.URL.Query(), "contract_code", "all-contract query should omit contract_code")
				} else {
					assert.Equal(t, "BTC-USDT", r.URL.Query().Get("contract_code"), "paired query should restrict its contract")
				}
				from := r.URL.Query().Get("from")
				mu.Lock()
				cursors = append(cursors, from)
				calls := len(cursors)
				mu.Unlock()
				var high, low int
				switch {
				case calls > 2:
					http.Error(w, "unexpected extra page", http.StatusBadRequest)
					return
				case from == "":
					high, low = 300, 201
				case from == "201":
					high, low = 200, 196
				default:
					http.Error(w, "unexpected cursor "+from, http.StatusBadRequest)
					return
				}
				contract := "BTC-USDT"
				if allPairs && from != "" {
					contract = "ETH-USDT"
				}
				rows := make([]string, 0, high-low+1)
				for id := high; id >= low; id-- {
					rows = append(rows, fmt.Sprintf(`{"id":"%d","order_id":"%d","contract_code":%q,"side":"buy","type":"limit","state":%q,"time_in_force":"gtc","margin_mode":"cross","price":"1","volume":"1","trade_volume":"1"}`, id, id, contract, state))
				}
				_, _ = w.Write([]byte(`{"code":200,"data":[` + strings.Join(rows, ",") + `]}`))
			}))
			t.Cleanup(server.Close)
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must succeed")
			h.API.AuthenticatedSupport = true
			h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "mock endpoint must be set")
			req := &order.MultiOrderRequest{AssetType: asset.USDTMarginedFutures, Type: order.AnyType, Side: order.AnySide, Pairs: currency.Pairs{btcusdtPair}, MarginType: margin.Multi}
			if allPairs {
				req.Pairs = nil
			}
			got, err := h.GetActiveOrders(t.Context(), req)
			require.NoError(t, err, "paged orders must be fetched")
			require.Len(t, got, 105, "every page must be returned")
			if allPairs {
				assert.True(t, got[104].Pair.Equal(currency.NewPair(currency.ETH, currency.USDT)), "orders from other contracts should be retained")
			}
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, []string{"", "201"}, cursors, "each page should start after the previous page's last id")
		})
	}

	type contractRequestCase struct {
		item     asset.Item
		pair     currency.Pair
		path     string
		contract string
	}
	for _, tc := range []contractRequestCase{
		{asset.CoinMarginedFutures, btcusdPair, "/swap-api/v1/swap_openorders", "BTC-USD"},
		{asset.Futures, currency.NewPair(currency.BTC, currency.NewCode("261016")), "/api/v1/contract_openorders", "BTC261016"},
	} {
		t.Run(tc.item.String(), func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var pages []int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				type requestPayload struct {
					PageIndex int64 `json:"page_index"`
				}
				var body requestPayload
				if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "request should decode") {
					return
				}
				mu.Lock()
				pages = append(pages, body.PageIndex)
				calls := len(pages)
				mu.Unlock()
				if calls > 2 {
					http.Error(w, "unexpected extra page", http.StatusBadRequest)
					return
				}
				n := 50
				if body.PageIndex == 2 {
					n = 3
				}
				rows := make([]string, 0, n)
				for i := range n {
					rows = append(rows, `{"order_id_str":"`+strconv.FormatInt(body.PageIndex*1000+int64(i), 10)+`","contract_code":"`+tc.contract+`","direction":"buy","order_price_type":"limit","status":3,"volume":1}`)
				}
				_, _ = w.Write([]byte(`{"status":"ok","data":{"total_page":2,"orders":[` + strings.Join(rows, ",") + `]}}`))
			}))
			t.Cleanup(server.Close)
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must succeed")
			h.API.AuthenticatedSupport = true
			h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
			require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "mock endpoint must be set")
			got, err := h.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: tc.item, Type: order.AnyType, Side: order.AnySide, Pairs: currency.Pairs{tc.pair}})
			require.NoError(t, err, "paged open orders must be fetched")
			assert.Len(t, got, 53, "every page should be returned")
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, []int64{1, 2}, pages, "both pages should be requested once")
		})
	}
	t.Run("coin margined fractional frozen margin", func(t *testing.T) {
		t.Parallel()
		h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_openorders", `{"status":"ok","data":{"orders":[{"symbol":"THETA","contract_code":"THETA-USD","volume":1,"price":0.6,"order_price_type":"post_only","direction":"buy","offset":"open","lever_rate":20,"order_id_str":"771043577949732864","created_at":1603872714279,"trade_volume":0,"trade_turnover":0,"fee":0,"trade_avg_price":null,"margin_frozen":0.833333333333333333,"status":3,"fee_asset":"THETA"}],"total_page":1,"current_page":1,"total_size":1}}`, nil)
		orders, err := h.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
			AssetType: asset.CoinMarginedFutures,
			Type:      order.AnyType,
			Side:      order.AnySide,
			Pairs:     currency.Pairs{currency.NewPair(currency.THETA, currency.USD)},
		})
		require.NoError(t, err, "fractional frozen margin must not prevent decoding open orders")
		require.Len(t, orders, 1, "one active order must be returned")
		assert.Equal(t, "771043577949732864", orders[0].OrderID, "order ID should match")
	})
	t.Run("validation", func(t *testing.T) {
		t.Parallel()

		getOrdersRequest := order.MultiOrderRequest{
			AssetType: asset.Spot,
			Type:      order.AnyType,
			Side:      order.AnySide,
		}
		_, err := e.GetActiveOrders(t.Context(), &getOrdersRequest)
		require.ErrorContains(t, err, "currency must be supplied", "GetActiveOrders must require a currency pair for spot")

		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "Setup must not error")
		type testCase struct {
			name        string
			pair        currency.Pair
			asset       asset.Item
			expectedErr error
		}
		testCases := []testCase{
			{name: "spot", pair: btcusdtPair, asset: asset.Spot, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
			{name: "coin margined futures", pair: btcusdPair, asset: asset.CoinMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
			{name: "usdt margined futures", pair: btcusdtPair, asset: asset.USDTMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
			{name: "futures", pair: btccwPair, asset: asset.Futures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
			{name: "unsupported asset", pair: btcusdtPair, asset: asset.Binary, expectedErr: asset.ErrNotSupported},
		}
		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := h.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
					Type:      order.AnyType,
					Pairs:     []currency.Pair{tt.pair},
					AssetType: tt.asset,
					Side:      order.AnySide,
				})
				require.ErrorIs(t, err, tt.expectedErr, "GetActiveOrders must return the expected branch error")
			})
		}
	})
	type openOrderRequestCase struct {
		item           asset.Item
		pair           currency.Pair
		path, contract string
	}
	for _, tc := range []openOrderRequestCase{
		{asset.Futures, btccwPair, "/api/v1/contract_openorders", "BTC_CW"},
		{asset.CoinMarginedFutures, btcusdPair, "/swap-api/v1/swap_openorders", "BTC-USD"},
	} {
		for _, fee := range []float64{-0.1, 0, 0.02} {
			t.Run(fmt.Sprintf("%s fee %g", tc.item, fee), func(t *testing.T) {
				t.Parallel()
				response := fmt.Sprintf(`{"status":"ok","data":{"total_page":1,"orders":[{"order_id_str":"1","contract_code":%q,"direction":"sell","order_price_type":"ioc","status":4,"volume":4,"trade_volume":1,"price":100,"lever_rate":3,"fee":%g}]}}`, tc.contract, fee)
				h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, tc.path, response, nil)
				orders, err := h.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: tc.item, Type: order.AnyType, Side: order.AnySide, Pairs: currency.Pairs{tc.pair}})
				require.NoError(t, err, "active orders must decode")
				require.Len(t, orders, 1, "one active order must be returned")
				assert.Equal(t, -fee, orders[0].Fee, "charges should be positive and rebates negative")
				assert.Equal(t, 1.0, orders[0].ExecutedAmount, "executed amount should match")
				assert.Equal(t, 3.0, orders[0].RemainingAmount, "remaining amount should exclude fills")
				assert.Equal(t, 100.0, orders[0].Price, "limit price should match")
				assert.Equal(t, 3.0, orders[0].Leverage, "leverage should match")
				assert.Equal(t, order.ImmediateOrCancel, orders[0].TimeInForce, "IOC should be retained")
				assert.Equal(t, order.PartiallyFilled, orders[0].Status, "partial status should match")
				assert.Equal(t, order.Sell, orders[0].Side, "side should match")
			})
		}
	}

	t.Run("delivery contracts share base", func(t *testing.T) {
		t.Parallel()
		var calls atomic.Int32
		h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/api/v1/contract_openorders", `{"status":"ok","data":{"total_page":1,"orders":[{"order_id_str":"1","contract_code":"BTC261009","direction":"buy","order_price_type":"limit","status":3},{"order_id_str":"2","contract_code":"BTC261016","direction":"buy","order_price_type":"limit","status":3}]}}`, func(_ *http.Request) { calls.Add(1) })
		pairs := currency.Pairs{currency.NewPair(currency.BTC, currency.NewCode("261009")), currency.NewPair(currency.BTC, currency.NewCode("261016"))}
		orders, err := h.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Type: order.AnyType, Side: order.AnySide, Pairs: pairs})
		require.NoError(t, err, "delivery orders must decode")
		assert.Equal(t, int32(1), calls.Load(), "base-scoped orders should be fetched once")
		require.Len(t, orders, 2, "each requested contract order must occur once")
		assert.NotEqual(t, orders[0].OrderID, orders[1].OrderID, "separate orders should retain identity")
	})
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v1/order/openOrders", `{
		"status":"ok",
		"data":[{"id":1,"symbol":"btcusdt","account-id":2,"amount":"2","price":"100","type":"buy-limit","state":"submitted","filled-amount":"1","filled-fees":"0.1"}]
	}`, func(r *http.Request) {
		assert.Equal(t, "buy", r.URL.Query().Get("side"), "spot order-side filter should be forwarded")
	})
	getOrdersRequest := order.MultiOrderRequest{
		AssetType: asset.Spot,
		Type:      order.AnyType,
		Pairs:     []currency.Pair{currency.NewBTCUSDT()},
		Side:      order.Buy,
	}

	orders, err := h.GetActiveOrders(t.Context(), &getOrdersRequest)
	require.NoError(t, err, "GetActiveOrders must not error")
	require.Len(t, orders, 1, "one active order must be returned")
	assert.Equal(t, "1", orders[0].OrderID, "order ID should match")
	assert.Equal(t, 1.0, orders[0].ExecutedAmount, "spot executed amount should match")
	assert.Equal(t, 1.0, orders[0].RemainingAmount, "spot remaining amount should exclude fills")
	assert.Equal(t, 0.1, orders[0].Fee, "spot fee should match")

	var crossRequests, isolatedRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		assert.Equal(t, "/v5/trade/order/opens", r.URL.Path, "V5 open-order path should match")
		marginMode := r.URL.Query().Get("margin_mode")
		switch marginMode {
		case "cross":
			crossRequests.Add(1)
		case "isolated":
			isolatedRequests.Add(1)
		default:
			assert.Failf(t, "unexpected margin mode", "received %q", marginMode)
		}
		response, err := json.Marshal(map[string]any{
			"code": 200,
			"data": []map[string]string{{"id": "1", "contract_code": "BTC-USDT", "order_id": "1", "side": "buy", "type": "limit", "state": "submitted", "time_in_force": "rpi", "margin_mode": marginMode, "price": "100", "volume": "2", "trade_volume": "1"}},
		})
		assert.NoError(t, err, "response should encode")
		_, err = w.Write(response)
		assert.NoError(t, err, "response should write")
	}))
	t.Cleanup(server.Close)
	h = new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT endpoint must be set")
	orders, err = h.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
		AssetType: asset.USDTMarginedFutures,
		Type:      order.AnyType,
		Pairs:     currency.Pairs{btcusdtPair},
		Side:      order.AnySide,
	})
	require.NoError(t, err, "GetActiveOrders must query both margin modes")
	require.Len(t, orders, 2, "cross and isolated active orders must be returned")
	assert.Equal(t, int32(1), crossRequests.Load(), "cross mode should be queried once")
	assert.Equal(t, int32(1), isolatedRequests.Load(), "isolated mode should be queried once")
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	var requests atomic.Uint64
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v1/query/deposit-withdraw",
		`{"status":"ok","data":[{"id":1,"type":"deposit","currency":"btc","tx-hash":"tx-1","chain":"btc","amount":2,"address":"address-1","fee":0,"state":"safe","created-at":1612261330443},{"id":2,"type":"withdraw","currency":"usdt","tx-hash":"tx-2","chain":"trc20usdt","amount":3,"address":"address-2","fee":0.1,"state":"confirmed","error-msg":"withdraw rejected: address risk","created-at":1612261389250}]}`,
		func(r *http.Request) {
			assert.Equal(t, "next", r.URL.Query().Get("direct"), "query direction should request the newest records")
			assert.Equal(t, "500", r.URL.Query().Get("size"), "query size should use HTX's maximum page size")
			switch r.URL.Query().Get("type") {
			case "deposit", "withdraw":
			default:
				require.Failf(t, "unexpected funding history type", "type %q must be deposit or withdraw", r.URL.Query().Get("type"))
			}
			requests.Add(1)
		})
	h.Name = "HTX"
	history, err := h.GetAccountFundingHistory(t.Context())
	require.NoError(t, err, "GetAccountFundingHistory must not error")
	require.Len(t, history, 4, "funding history must include both endpoint responses")
	assert.Equal(t, uint64(2), requests.Load(), "funding history should make two requests")
	assert.Equal(t, exchange.FundingHistory{
		ExchangeName:    "HTX",
		Status:          "safe",
		TransferID:      "1",
		Timestamp:       time.UnixMilli(1612261330443),
		Currency:        "BTC",
		Amount:          2,
		TransferType:    "deposit",
		CryptoToAddress: "address-1",
		CryptoTxID:      "tx-1",
		CryptoChain:     "btc",
	}, history[0], "deposit history should be normalised")
	assert.Equal(t, "withdraw", history[1].TransferType, "withdrawal type should be normalised")
	assert.Equal(t, 0.1, history[1].Fee, "withdrawal fee should be retained")
	assert.Equal(t, "withdraw rejected: address risk", history[1].Description, "withdrawal rejection reason should decode")
}

func TestGetAccountID(t *testing.T) {
	t.Parallel()
	t.Run("empty accounts", func(t *testing.T) {
		t.Parallel()
		h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v1/account/accounts", `{"status":"ok","data":[]}`, nil)
		_, err := h.GetAccountID(t.Context())
		require.ErrorIs(t, err, errNoAccountReturned, "empty accounts must return the sentinel")
	})
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	_, err := h.GetAccountID(t.Context())
	require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "GetAccountID must require credentials")
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	t.Run("protect mode until submission completes", func(t *testing.T) {
		t.Parallel()
		started, release := make(chan struct{}), make(chan struct{})
		h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodPost, "/v5/trade/order", `{"code":200,"data":{"order_id":"1"}}`, func(*http.Request) { close(started); <-release }, httpTestResponse{method: http.MethodGet, path: "/v5/position/mode", body: `{"code":200,"data":{"position_mode":"single_side"}}`, calls: 1})
		var releaseOnce sync.Once
		t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
		submission := &order.Submit{Exchange: h.Name, Pair: btcusdtPair, AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Limit, Price: 100, Amount: 1}
		result := make(chan error, 1)
		go func() { _, err := h.SubmitOrder(t.Context(), submission); result <- err }()
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "submission must reach the mock transport")
		}
		_, err := h.SetV5PositionMode(t.Context(), "dual_side")
		require.ErrorIs(t, err, errPositionModeInUse, "mode changes must be excluded until the order acknowledgement arrives")
		releaseOnce.Do(func() { close(release) })
		require.NoError(t, <-result, "submission must complete after acknowledgement")
		assert.False(t, h.positionModeInUse.Load(), "completed submission should release the guard")
	})

	for _, leverage := range []float64{-1, 1.5, math.NaN(), math.Inf(1), math.Exp2(64)} {
		t.Run(fmt.Sprintf("invalid integral leverage %v", leverage), func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "HTX setup must succeed")
			_, err := h.SubmitOrder(t.Context(), &order.Submit{Exchange: h.Name, Pair: btccwPair, AssetType: asset.Futures, Side: order.Buy, Type: order.Limit, Price: 100, Amount: 1, Leverage: leverage})
			require.ErrorIs(t, err, errInvalidLeverage, "SubmitOrder must reject leverage that cannot be represented as a non-negative integer before I/O")
		})
	}

	type testCase struct {
		name          string
		clientOrderID string
		wantID        string
	}
	for _, tc := range []testCase{
		{name: "spot without client order ID"},
		{name: "spot client order ID equal to account ID", clientOrderID: "1", wantID: "1"},
		{name: "spot distinct client order ID", clientOrderID: "client-1", wantID: "client-1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodPost, "/v1/order/orders/place", `{"status":"ok","data":"1"}`, func(r *http.Request) {
				calls.Add(1)
				var payload map[string]any
				if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload), "spot order request should decode") {
					return
				}
				assert.Equal(t, "1", payload["account-id"], "account ID should be retained")
				if tc.wantID == "" {
					assert.NotContains(t, payload, "client-order-id", "absent client order IDs should be omitted")
				} else {
					assert.Equal(t, tc.wantID, payload["client-order-id"], "explicit client order IDs should be submitted")
				}
			})
			submission := &order.Submit{Exchange: h.Name, Pair: btcusdtPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Price: 5, Amount: 1, ClientID: "1", ClientOrderID: tc.clientOrderID}
			response, err := h.SubmitOrder(t.Context(), submission)
			require.NoError(t, err, "spot order submission must succeed")
			assert.Equal(t, "1", response.ClientID, "response should retain the account ID")
			assert.Equal(t, tc.wantID, response.ClientOrderID, "response should reflect the submitted client order ID")
			assert.Equal(t, tc.clientOrderID, submission.ClientOrderID, "submission should remain unchanged")
			assert.Equal(t, int32(1), calls.Load(), "submission should reach the endpoint once")
		})
	}
	t.Run("request bodies", func(t *testing.T) {
		t.Parallel()
		type marketCase struct {
			asset          asset.Item
			pair           currency.Pair
			endpoint       exchange.URL
			path, response string
			contract       string
			additional     []httpTestResponse
		}
		for _, market := range []marketCase{
			{asset.CoinMarginedFutures, btcusdPair, exchange.RestFutures, "/swap-api/v1/swap_order", `{"status":"ok","data":{"order_id_str":"1"}}`, "BTC-USD", nil},
			{asset.Futures, currency.NewPair(currency.BTC, currency.NewCode("261016")), exchange.RestFutures, "/api/v1/contract_order", `{"status":"ok","data":{"order_id_str":"1"}}`, "BTC261016", nil},
			{asset.USDTMarginedFutures, btcusdtPair, exchange.RestUSDTMargined, "/v5/trade/order", `{"code":200,"data":{"order_id":"1"}}`, "BTC-USDT", []httpTestResponse{{method: http.MethodGet, path: "/v5/position/mode", body: `{"code":200,"data":{"position_mode":"single_side"}}`, calls: 1}}},
		} {
			for _, side := range []order.Side{order.Buy, order.Sell} {
				for _, reduceOnly := range []bool{false, true} {
					t.Run(market.asset.String()+" "+side.String()+map[bool]string{false: " open", true: " reduce"}[reduceOnly], func(t *testing.T) {
						t.Parallel()
						h := newHTTPTestExchange(t, market.endpoint, http.MethodPost, market.path, market.response, func(r *http.Request) {
							var payload map[string]any
							if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload), "order request should decode") {
								return
							}
							assert.Equal(t, market.contract, payload["contract_code"], "contract code should identify the instrument")
							if market.asset == asset.USDTMarginedFutures {
								assert.Equal(t, map[order.Side]string{order.Buy: "buy", order.Sell: "sell"}[side], payload["side"], "side should be preserved")
								assert.Equal(t, "5", payload["price"], "limit price should be submitted")
								assert.Equal(t, "2", payload["volume"], "contract volume should come from the base amount")
								assert.Equal(t, "77", payload["client_order_id"], "client order ID should be submitted")
								if reduceOnly {
									assert.Equal(t, 1.0, payload["reduce_only"], "reduce-only orders should be flagged")
								} else {
									assert.NotContains(t, payload, "reduce_only", "opening orders should not be flagged reduce-only")
								}
								return
							}
							// HTX documents lowercase directions; comparing without case pins the side, not its spelling
							assert.Equal(t, map[order.Side]string{order.Buy: "buy", order.Sell: "sell"}[side], strings.ToLower(fmt.Sprint(payload["direction"])), "direction should be preserved")
							assert.Equal(t, map[bool]string{false: "open", true: "close"}[reduceOnly], payload["offset"], "offset should follow reduce-only")
							assert.Equal(t, 5.0, payload["price"], "limit price should be submitted")
							assert.Equal(t, 2.0, payload["volume"], "contract volume should come from the base amount")
							assert.Equal(t, 3.0, payload["lever_rate"], "leverage should be submitted")
							assert.Contains(t, []any{"77", 77.0}, payload["client_order_id"], "client order ID should be submitted")
						}, market.additional...)
						response, err := h.SubmitOrder(t.Context(), &order.Submit{
							Exchange: h.Name, Pair: market.pair, AssetType: market.asset, Side: side, Type: order.Limit,
							Amount: 2, QuoteAmount: 9, Price: 5, Leverage: 3, ClientOrderID: "77", ReduceOnly: reduceOnly,
						})
						require.NoError(t, err, "order submission must succeed")
						assert.Equal(t, "1", response.OrderID, "submitted order ID should match")
					})
				}
			}
		}
	})
	type marketCase struct {
		asset          asset.Item
		pair           currency.Pair
		endpoint       exchange.URL
		path, response string
	}
	for _, market := range []marketCase{
		{asset.Spot, btcusdtPair, exchange.RestSpot, "/v1/order/orders/place", `{"status":"ok","data":"1"}`},
		{asset.CoinMarginedFutures, btcusdPair, exchange.RestFutures, "/swap-api/v1/swap_order", `{"status":"ok","data":{"order_id_str":"1"}}`},
		{asset.Futures, btccwPair, exchange.RestFutures, "/api/v1/contract_order", `{"status":"ok","data":{"order_id_str":"1"}}`},
	} {
		for _, side := range []order.Side{order.Buy, order.Sell} {
			type testCase struct {
				name                         string
				orderType                    order.Type
				timeInForce                  order.TimeInForce
				spotSuffix, derivativeType   string
				expectedErr, expectedSpotErr error
			}
			for _, tc := range []testCase{
				{name: "limit default", orderType: order.Limit, spotSuffix: "-limit", derivativeType: "limit"},
				{name: "limit GTC", orderType: order.Limit, timeInForce: order.GoodTillCancel, spotSuffix: "-limit", derivativeType: "limit"},
				{name: "limit post only", orderType: order.Limit, timeInForce: order.PostOnly, spotSuffix: "-limit-maker", derivativeType: "post_only"},
				{name: "limit GTC post only", orderType: order.Limit, timeInForce: order.GoodTillCancel | order.PostOnly, spotSuffix: "-limit-maker", derivativeType: "post_only"},
				{name: "limit IOC", orderType: order.Limit, timeInForce: order.ImmediateOrCancel, spotSuffix: "-ioc", derivativeType: "ioc"},
				{name: "limit FOK", orderType: order.Limit, timeInForce: order.FillOrKill, spotSuffix: "-limit-fok", derivativeType: "fok"},
				{name: "market default", orderType: order.Market, spotSuffix: "-market", derivativeType: "optimal_20"},
				{name: "market GTC", orderType: order.Market, timeInForce: order.GoodTillCancel, spotSuffix: "-market", derivativeType: "optimal_20"},
				{name: "market IOC", orderType: order.Market, timeInForce: order.ImmediateOrCancel, derivativeType: "optimal_20_ioc", expectedSpotErr: order.ErrUnsupportedTimeInForce},
				{name: "market FOK", orderType: order.Market, timeInForce: order.FillOrKill, derivativeType: "optimal_20_fok", expectedSpotErr: order.ErrUnsupportedTimeInForce},
				{name: "market post only", orderType: order.Market, timeInForce: order.PostOnly, expectedErr: order.ErrUnsupportedTimeInForce},
				{name: "limit timed expiry", orderType: order.Limit, timeInForce: order.GoodTillTime, expectedErr: order.ErrUnsupportedTimeInForce},
				{name: "limit combined expiry", orderType: order.Limit, timeInForce: order.GoodTillCancel | order.GoodTillTime, expectedErr: order.ErrUnsupportedTimeInForce},
				{name: "stop", orderType: order.Stop, expectedErr: order.ErrUnsupportedOrderType},
				{name: "stop limit", orderType: order.StopLimit, expectedErr: order.ErrUnsupportedOrderType},
				{name: "take profit", orderType: order.TakeProfit, expectedErr: order.ErrUnsupportedOrderType},
				{name: "take profit market", orderType: order.TakeProfitMarket, expectedErr: order.ErrUnsupportedOrderType},
				{name: "trailing stop", orderType: order.TrailingStop, expectedErr: order.ErrUnsupportedOrderType},
				{name: "trailing stop limit", orderType: order.TrailingStopLimit, expectedErr: order.ErrUnsupportedOrderType},
			} {
				t.Run(fmt.Sprintf("instructions %s %s %s", market.asset, side, tc.name), func(t *testing.T) {
					t.Parallel()
					expectedErr := tc.expectedErr
					if market.asset == asset.Spot && tc.expectedSpotErr != nil {
						expectedErr = tc.expectedSpotErr
					}
					h := newHTTPTestExchange(t, market.endpoint, http.MethodPost, market.path, market.response, func(r *http.Request) {
						if expectedErr != nil {
							assert.Fail(t, "unsupported order instructions should not reach HTX")
						}
						var payload map[string]any
						require.NoError(t, json.NewDecoder(r.Body).Decode(&payload), "order request must decode")
						if market.asset == asset.Spot {
							assert.Equal(t, strings.ToLower(side.String())+tc.spotSuffix, payload["type"], "spot order instructions should be preserved")
							if tc.orderType == order.Limit {
								assert.Equal(t, "5", payload["price"], "spot limit price should be submitted")
							}
						} else {
							assert.Equal(t, tc.derivativeType, payload["order_price_type"], "derivative order instructions should be preserved")
						}
					})
					response, err := h.SubmitOrder(t.Context(), &order.Submit{
						Exchange: h.Name, Pair: market.pair, AssetType: market.asset, Side: side,
						Type: tc.orderType, TimeInForce: tc.timeInForce, Amount: 1, QuoteAmount: 5, Price: 5, Leverage: 5, ClientID: "1",
					})
					if expectedErr != nil {
						require.ErrorIs(t, err, expectedErr, "unsupported order instructions must be rejected")
						assert.Nil(t, response, "rejected orders should have no submission response")
						return
					}
					require.NoError(t, err, "supported order instructions must be submitted")
					assert.Equal(t, "1", response.OrderID, "submitted order ID should match")
				})
			}
		}
	}
	t.Run("validation", func(t *testing.T) {
		t.Parallel()

		_, err := e.SubmitOrder(t.Context(), nil)
		require.ErrorIs(t, err, order.ErrSubmissionIsNil, "SubmitOrder must reject nil submissions")

		_, err = e.SubmitOrder(t.Context(), &order.Submit{
			Exchange: e.Name, Pair: btcusdtPair, AssetType: asset.Spot,
			Side: order.Buy, Type: order.Market, Amount: 1, ClientID: "1",
		})
		require.ErrorIs(t, err, order.ErrAmountMustBeSet, "SubmitOrder must require quote amount for spot market buys")

		_, err = e.SubmitOrder(t.Context(), &order.Submit{
			Exchange:  e.Name,
			Pair:      btcusdtPair,
			AssetType: asset.Spot,
			Side:      order.Buy,
			Type:      order.Limit,
			Price:     1,
			Amount:    1,
			ClientID:  "invalid",
		})
		require.Error(t, err, "SubmitOrder must reject invalid spot account IDs")

		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "Setup must not error")
		type testCase struct {
			name        string
			pair        currency.Pair
			asset       asset.Item
			expectedErr error
		}
		testCases := []testCase{
			{name: "coin margined futures", pair: btcusdPair, asset: asset.CoinMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
			{name: "usdt margined futures", pair: btcusdtPair, asset: asset.USDTMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
			{name: "futures", pair: btccwPair, asset: asset.Futures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
			{name: "unsupported asset", pair: btcusdtPair, asset: asset.Binary, expectedErr: asset.ErrNotSupported},
		}
		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := h.SubmitOrder(t.Context(), &order.Submit{
					Exchange:  h.Name,
					Pair:      tt.pair,
					AssetType: tt.asset,
					Side:      order.Buy,
					Type:      order.Limit,
					Price:     1,
					Amount:    1,
					Leverage:  1,
				})
				require.ErrorIs(t, err, tt.expectedErr, "SubmitOrder must return the expected branch error")
			})
		}
	})
	type submissionResponseCase struct {
		name       string
		endpoint   exchange.URL
		path       string
		submission order.Submit
		response   string
		status     order.Status
		check      func(*testing.T, *http.Request)
		additional []httpTestResponse
	}
	for _, tt := range []submissionResponseCase{
		{
			name: "spot", endpoint: exchange.RestSpot, path: "/v1/order/orders/place",
			submission: order.Submit{Pair: btcusdtPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Price: 5, Amount: 1, ClientID: "1"},
			response:   `{"status":"ok","data":"1"}`,
			status:     order.New,
		},
		{
			name: "spot market buy quote amount", endpoint: exchange.RestSpot, path: "/v1/order/orders/place",
			submission: order.Submit{Pair: btcusdtPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Market, QuoteAmount: 25, ClientID: "1", ClientOrderID: "client-1"},
			response:   `{"status":"ok","data":"1"}`,
			status:     order.New,
			check: func(t *testing.T, r *http.Request) {
				t.Helper()
				var req map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&req), "spot market request must decode")
				assert.Equal(t, "25", req["amount"], "spot market buy should submit the quote amount")
				assert.Equal(t, "client-1", req["client-order-id"], "spot client order ID should be submitted")
			},
		},
		{
			name: "coin margined futures", endpoint: exchange.RestFutures, path: "/swap-api/v1/swap_order",
			submission: order.Submit{Pair: btcusdPair, AssetType: asset.CoinMarginedFutures, Side: order.Buy, Type: order.Limit, Price: 5, Amount: 1, Leverage: 5},
			response:   `{"status":"ok","data":{"order_id":1,"order_id_str":"1"}}`,
			status:     order.New,
		},
		{
			name: "USDT margined futures", endpoint: exchange.RestUSDTMargined, path: "/v5/trade/order",
			additional: []httpTestResponse{{method: http.MethodGet, path: "/v5/position/mode", body: `{"code":200,"data":{"position_mode":"single_side"}}`, calls: 1}},
			submission: order.Submit{Pair: btcusdtPair, AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Limit, Price: 5, Amount: 1, Leverage: 5, TimeInForce: order.GoodTillCancel | order.PostOnly},
			response:   `{"code":200,"data":{"order_id":"1"}}`,
			status:     order.New,
			check: func(t *testing.T, r *http.Request) {
				t.Helper()
				var req V5OrderRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&req), "USDT-margined order request must decode")
				assert.Equal(t, orderPriceTypePostOnly, req.Type, "post-only protection should be preserved")
			},
		},
		{
			name: "delivery futures", endpoint: exchange.RestFutures, path: "/api/v1/contract_order",
			submission: order.Submit{Pair: btccwPair, AssetType: asset.Futures, Side: order.Buy, Type: order.Limit, Price: 5, Amount: 1, Leverage: 5},
			response:   `{"status":"ok","data":{"order_id":1,"order_id_str":"1"}}`,
			status:     order.New,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var check func(*http.Request)
			if tt.check != nil {
				check = func(r *http.Request) { tt.check(t, r) }
			}
			h := newHTTPTestExchange(t, tt.endpoint, http.MethodPost, tt.path, tt.response, check, tt.additional...)
			tt.submission.Exchange = h.Name
			response, err := h.SubmitOrder(t.Context(), &tt.submission)
			require.NoError(t, err, "SubmitOrder must not error")
			assert.Equal(t, "1", response.OrderID, "order ID should match")
			assert.Equal(t, tt.status, response.Status, "response status should match")
		})
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()

	err := e.CancelOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelOrder must reject nil cancellations")

	err = e.CancelOrder(t.Context(), &order.Cancel{
		OrderID:   "invalid",
		Pair:      btcusdtPair,
		AssetType: asset.Spot,
	})
	require.Error(t, err, "CancelOrder must reject non-numeric spot order IDs")

	err = e.CancelOrder(t.Context(), &order.Cancel{
		OrderID:   "1",
		Pair:      btcusdtPair,
		AssetType: asset.Binary,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported, "CancelOrder must reject unsupported assets")

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	type testCase struct {
		name  string
		pair  currency.Pair
		asset asset.Item
	}
	testCases := []testCase{
		{name: "spot", pair: btcusdtPair, asset: asset.Spot},
		{name: "coin margined futures", pair: btcusdPair, asset: asset.CoinMarginedFutures},
		{name: "usdt margined futures", pair: btcusdtPair, asset: asset.USDTMarginedFutures},
		{name: "futures", pair: btccwPair, asset: asset.Futures},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := h.CancelOrder(t.Context(), &order.Cancel{
				OrderID:   "1",
				Pair:      tt.pair,
				AssetType: tt.asset,
			})
			require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "CancelOrder must require credentials")
		})
	}
	type cancellationRequestCase struct {
		name          string
		endpoint      exchange.URL
		path          string
		cancel        order.Cancel
		expectedField string
		expectedValue any
		response      string
		expectedErr   error
	}
	for _, tt := range []cancellationRequestCase{
		{
			name: "spot response", endpoint: exchange.RestSpot, path: "/v1/order/orders/42/submitcancel",
			cancel: order.Cancel{OrderID: "42", Pair: btcusdtPair, AssetType: asset.Spot},
		},
		{
			name: "spot client order ID", endpoint: exchange.RestSpot, path: "/v1/order/orders/batchcancel",
			cancel:        order.Cancel{ClientOrderID: "client-42", Pair: btcusdtPair, AssetType: asset.Spot},
			expectedField: "client-order-ids", expectedValue: []any{"client-42"},
			response: `{"status":"ok","data":{"success":["client-42"],"failed":[]}}`,
		},
		{
			name: "spot client order ID failure", endpoint: exchange.RestSpot, path: "/v1/order/orders/batchcancel",
			cancel:        order.Cancel{ClientOrderID: "client-42", Pair: btcusdtPair, AssetType: asset.Spot},
			expectedField: "client-order-ids", expectedValue: []any{"client-42"},
			response:    `{"status":"ok","data":{"success":[],"failed":[{"client-order-id":"client-42","err-code":"order-orderstate-error","err-msg":"Incorrect order state"}]}}`,
			expectedErr: errAPIResponse,
		},
		{
			name: "coin margined client order ID", endpoint: exchange.RestFutures, path: "/swap-api/v1/swap_cancel",
			cancel:        order.Cancel{ClientOrderID: "client-42", Pair: btcusdPair, AssetType: asset.CoinMarginedFutures},
			expectedField: "client_order_id", expectedValue: "client-42",
			response: `{"status":"ok","data":{"successes":"client-42","errors":[]}}`,
		},
		{
			name: "USDT margined client order ID", endpoint: exchange.RestUSDTMargined, path: "/v5/trade/cancel_order",
			cancel:        order.Cancel{ClientOrderID: "client-42", Pair: btcusdtPair, AssetType: asset.USDTMarginedFutures},
			expectedField: "client_order_id", expectedValue: "client-42",
		},
		{
			name: "delivery futures order ID", endpoint: exchange.RestFutures, path: "/api/v1/contract_cancel",
			cancel:        order.Cancel{OrderID: "42", Pair: btccwPair, AssetType: asset.Futures},
			expectedField: "order_id", expectedValue: "42",
			response: `{"status":"ok","data":{"successes":"42","errors":[]}}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			response := tt.response
			if response == "" {
				response = emptySuccessResponse
			}
			h := newHTTPTestExchange(t, tt.endpoint, http.MethodPost, tt.path, response, func(r *http.Request) {
				if tt.expectedField == "" {
					return
				}
				payload, err := io.ReadAll(r.Body)
				require.NoError(t, err, "request body must be readable")
				var body map[string]any
				require.NoError(t, json.Unmarshal(payload, &body), "request body must decode")
				assert.Equal(t, tt.expectedValue, body[tt.expectedField], "order identifier should match")
			})
			err := h.CancelOrder(t.Context(), &tt.cancel)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr, "CancelOrder must return the expected error")
				if tt.name == "spot client order ID failure" {
					assert.ErrorContains(t, err, "Incorrect order state", "CancelOrder should include the exchange reason")
				}
			} else {
				require.NoError(t, err, "CancelOrder must not error")
			}
		})
	}
	for _, a := range []asset.Item{asset.CoinMarginedFutures, asset.Futures} {
		type identifierCase struct{ name, orderID, clientID, acknowledgement string }
		for _, identifiers := range []identifierCase{
			{"order ID", "9", "", "9"}, {"client ID", "", "7", "7"}, {"order ID takes precedence", "9", "7", "9"},
		} {
			type testCase struct {
				name, data string
				want       error
			}
			for _, tc := range []testCase{
				{"success", `{"successes":"` + identifiers.acknowledgement + `","errors":[]}`, nil},
				{"rejected", `{"successes":"","errors":[{"order_id":"9","err_code":1061,"err_msg":"order cannot be cancelled"}]}`, errAPIResponse},
				{"partial rejection", `{"successes":"` + identifiers.acknowledgement + `","errors":[{"order_id":"9","err_code":1061,"err_msg":"order cannot be cancelled"}]}`, errAPIResponse},
				{"empty acknowledgement", `{}`, common.ErrInvalidResponse},
				{"unrelated acknowledgement", `{"successes":"19","errors":[]}`, common.ErrInvalidResponse},
			} {
				t.Run(a.String()+"/"+identifiers.name+"/"+tc.name, func(t *testing.T) {
					t.Parallel()
					path := "/swap-api/v1/swap_cancel"
					if a == asset.Futures {
						path = "/api/v1/contract_cancel"
					}
					h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, path, `{"status":"ok","data":`+tc.data+`}`, func(r *http.Request) {
						var body map[string]any
						if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "cancellation request should decode") {
							return
						}
						clientID := identifiers.clientID
						if identifiers.orderID != "" {
							clientID = ""
						}
						for field, value := range map[string]string{"order_id": identifiers.orderID, "client_order_id": clientID} {
							if value == "" {
								assert.NotContains(t, body, field, "unused identifier should be omitted")
							} else {
								assert.Equal(t, value, body[field], "request should preserve the selected identifier")
							}
						}
						if a == asset.Futures {
							assert.Equal(t, "BTC", body["symbol"], "delivery cancellation should select the base currency")
						} else {
							assert.Equal(t, "BTC-USD", body["contract_code"], "perpetual cancellation should select the contract")
						}
					})
					err := h.CancelOrder(t.Context(), &order.Cancel{AssetType: a, Pair: btcusdPair, OrderID: identifiers.orderID, ClientOrderID: identifiers.clientID, ClientID: "client-name"})
					assert.ErrorIs(t, err, tc.want, "cancellation should acknowledge the requested identifier and preserve rejections")
					if tc.want == errAPIResponse {
						assert.ErrorContains(t, err, "order cannot be cancelled", "rejection should retain the venue reason")
					}
				})
			}
		}
	}
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	t.Run("validation", func(t *testing.T) {
		t.Parallel()
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "Setup must not error")
		type testCase struct {
			name        string
			asset       asset.Item
			expectedErr error
		}
		testCases := []testCase{
			{name: "spot", asset: asset.Spot, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
			{name: "coin margined futures", asset: asset.CoinMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
			{name: "usdt margined futures", asset: asset.USDTMarginedFutures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
			{name: "futures", asset: asset.Futures, expectedErr: exchange.ErrAuthenticationSupportNotEnabled},
			{name: "unsupported asset", asset: asset.Binary, expectedErr: asset.ErrNotSupported},
		}
		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := h.UpdateAccountBalances(t.Context(), tt.asset)
				require.ErrorIs(t, err, tt.expectedErr, "UpdateAccountBalances must return the expected branch error")
			})
		}
	})
	type testCase struct {
		name, data        string
		total, free, hold float64
	}
	for _, tc := range []testCase{
		{"documented balance", `"equity":"113.54","isolated_equity":"0.5","available":"113.54","isolated_available":"30"`, 114.04, 113.54, 0.5},
		{"losing cross position", `"equity":"90","available":"100","profit_unreal":"-10","initial_margin":"20"`, 90, 70, 20},
		{"isolated collateral", `"equity":"100","isolated_equity":"20","available":"100","isolated_available":"15"`, 120, 100, 20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/account/balance", `{"code":200,"data":{"details":[{"currency":"USDT",`+tc.data+`}]}}`, nil)
			subAccounts, err := h.UpdateAccountBalances(t.Context(), asset.USDTMarginedFutures)
			require.NoError(t, err, "REST balance must decode")
			balance := subAccounts[0].Balances[currency.USDT]
			assert.InDelta(t, tc.total, balance.Total, 1e-10, "total should include isolated equity")
			assert.InDelta(t, tc.free, balance.Free, 1e-10, "free should represent cross margin available for trading")
			assert.InDelta(t, tc.hold, balance.Hold, 1e-10, "held margin should reconcile with equity")
			raw := []byte(`{"ts":1603878749908,"data":{"details":[{"currency":"USDT",` + tc.data + `}]}}`)
			require.NoError(t, h.wsHandleUSDTMarginedPrivateMessage(t.Context(), &subscription.Subscription{Asset: asset.USDTMarginedFutures, Channel: subscription.MyAccountChannel}, raw), "equivalent websocket balance must decode")
			require.Len(t, h.Websocket.DataHandler.C, 1, "account push must dispatch one message")
			message := <-h.Websocket.DataHandler.C
			changes, ok := message.Data.([]accounts.Change)
			require.True(t, ok, "account push must contain changes")
			require.Len(t, changes, 1, "account push must contain one currency")
			assert.Equal(t, balance.Total, changes[0].Balance.Total, "REST and websocket total should agree")
			assert.Equal(t, balance.Free, changes[0].Balance.Free, "REST and websocket free should agree")
			assert.Equal(t, balance.Hold, changes[0].Balance.Hold, "REST and websocket held margin should agree")
		})
	}
	t.Run("null V5 response", func(t *testing.T) {
		t.Parallel()
		h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/account/balance", `null`, nil)
		_, err := h.UpdateAccountBalances(t.Context(), asset.USDTMarginedFutures)
		require.ErrorIs(t, err, errEmptyResult, "null balances must return an error")
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/account/accounts":
			_, _ = w.Write([]byte(`{"status":"ok","data":[{"id":1,"type":"spot","state":"working"}]}`))
		case "/v1/account/accounts/1/balance":
			_, _ = w.Write([]byte(`{"status":"ok","data":{"id":1,"type":"spot","state":"working","list":[{"currency":"btc","type":"trade","balance":"2"},{"currency":"btc","type":"frozen","balance":"3"},{"currency":"btc","type":"loan","balance":"7"}]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	h.API.AuthenticatedSupport = true
	h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "spot endpoint must be set")
	subAccounts, err := h.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	require.Len(t, subAccounts, 1, "one spot subaccount must be returned")
	balance := subAccounts[0].Balances[currency.BTC]
	assert.Equal(t, 5.0, balance.Total, "total should include free and held balances")
	assert.Equal(t, 2.0, balance.Free, "free balance should include trade funds")
	assert.Equal(t, 3.0, balance.Hold, "held balance should include frozen funds")

	h = newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/account/balance",
		`{"code":200,"data":{"details":[{"currency":"USDT","equity":"10","isolated_equity":"3","available":"4","isolated_available":"2"}]}}`, nil)
	subAccounts, err = h.UpdateAccountBalances(t.Context(), asset.USDTMarginedFutures)
	require.NoError(t, err, "USDT-margined UpdateAccountBalances must not error")
	balance = subAccounts[0].Balances[currency.USDT]
	assert.Equal(t, 4.0, balance.Free, "free balance should exclude isolated collateral")
	assert.Equal(t, 13.0, balance.Total, "total should include isolated equity")
	assert.Equal(t, 9.0, balance.Hold, "held balance should include isolated collateral")
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyOrder(t.Context(), &order.Modify{AssetType: asset.Spot})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "ModifyOrder must return unsupported")
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()

	withdrawCryptoRequest := withdraw.Request{
		Exchange:    e.Name,
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	_, err := e.WithdrawCryptocurrencyFunds(t.Context(), &withdrawCryptoRequest)
	require.ErrorContains(t, err, withdraw.ErrStrAmountMustBeGreaterThanZero)
}

func TestWithdrawFiatFunds(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFunds(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestWithdrawFiatFundsToInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestAuthenticateWebsocket(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	err := h.AuthenticateWebsocket(t.Context())
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "AuthenticateWebsocket must defer to the connection setups")
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	err := h.ValidateAPICredentials(t.Context(), asset.Spot)
	require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "ValidateAPICredentials must report authentication errors")
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, currency, chain string }{
		{name: "missing currency", currency: "btc"}, {name: "missing chain", currency: "usdt", chain: "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v2/account/deposit/address", `{"code":200,"data":[{"currency":"usdt","address":"address","chain":"usdterc20"}]}`, nil)
			_, err := h.GetDepositAddress(t.Context(), currency.NewCode(tc.currency), "", tc.chain)
			require.ErrorIs(t, err, errDepositAddressNotFound, "unmatched deposit details must return the sentinel")
		})
	}
	t.Run("authentication required", func(t *testing.T) {
		t.Parallel()
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "Setup must not error")
		_, err := h.GetDepositAddress(t.Context(), currency.USDT, "", "uSdTeRc20")
		require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "GetDepositAddress must require credentials")
	})
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v2/account/deposit/address", `{
		"code":200,
		"data":[{"currency":"usdt","address":"address","addressTag":"","chain":"usdterc20"}]
	}`, nil)
	address, err := h.GetDepositAddress(t.Context(), currency.USDT, "", "uSdTeRc20")
	require.NoError(t, err, "GetDepositAddress must not error")
	assert.Equal(t, "address", address.Address, "deposit address should match")
}

func TestV5FreeMargin(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name string
		data V5CurrencyBalance
		want float64
	}
	for _, tc := range []testCase{
		{name: "documented balance", data: V5CurrencyBalance{Available: 113.54}, want: 113.54},
		{name: "losing cross position", data: V5CurrencyBalance{Available: 100, ProfitUnreal: -10, InitialMargin: 20}, want: 70},
		{name: "profitable cross position", data: V5CurrencyBalance{Available: 100, ProfitUnreal: 10, InitialMargin: 20}, want: 90},
		{name: "isolated funds remain held", data: V5CurrencyBalance{Available: 100, IsolatedAvailable: 30, IsolatedEquity: 0.5}, want: 100},
		{name: "empty account"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, v5FreeMargin(&tc.data), "free margin should follow the cross-margin contract")
		})
	}
}

func TestFormatV5OrderDetail(t *testing.T) {
	t.Parallel()
	for _, tif := range []string{"gtc", "ioc", "fok", "rpi", "RPI"} {
		for _, feeCurrency := range []string{"USDT", "HTX", "USDT,HTX"} {
			t.Run(tif+"/"+feeCurrency, func(t *testing.T) {
				t.Parallel()
				h := new(Exchange)
				h.SetDefaults()
				detail, err := h.formatV5OrderDetail(&V5Order{ContractCode: "BTC-USDT", Side: "buy", Type: "limit", State: "filled", TimeInForce: tif, MarginMode: "cross", Fee: -0.1143702, FeeCurrency: feeCurrency}, asset.USDTMarginedFutures)
				require.NoError(t, err, "documented order enums must decode")
				assert.Equal(t, currency.USDT, detail.FeeAsset, "aggregate fee should remain denominated in USDT")
				assert.Equal(t, 0.1143702, detail.Fee, "charged fee should be positive in GCT")
				if strings.EqualFold(tif, "rpi") {
					assert.Equal(t, order.PostOnly, detail.TimeInForce, "RPI should retain resting liquidity behaviour")
				}
			})
		}
	}
	h := new(Exchange)
	h.SetDefaults()
	_, err := h.formatV5OrderDetail(nil, asset.USDTMarginedFutures)
	require.ErrorIs(t, err, common.ErrNilPointer, "formatV5OrderDetail must reject nil data")
	detail, err := h.formatV5OrderDetail(&V5Order{
		ContractCode:      "BTC-USDT",
		OrderID:           "1",
		ClientOrderID:     "2",
		Side:              "buy",
		Type:              orderPriceTypePostOnly,
		State:             "filled",
		TimeInForce:       "gtc",
		MarginMode:        "cross",
		Price:             100,
		Volume:            2,
		TradeVolume:       1,
		TradeTurnover:     100,
		TradeAveragePrice: 100,
		Fee:               0.1,
		FeeCurrency:       "USDT",
		LeverageRate:      5,
		ReduceOnly:        true,
	}, asset.USDTMarginedFutures)
	require.NoError(t, err, "formatV5OrderDetail must not error")
	assert.Equal(t, "1", detail.OrderID, "order ID should match")
	assert.Equal(t, order.Buy, detail.Side, "side should match")
	assert.Equal(t, order.Limit, detail.Type, "type should match")
	assert.Equal(t, order.Filled, detail.Status, "status should match")
	assert.Equal(t, order.PostOnly, detail.TimeInForce, "post-only order should retain its time in force")
	assert.Equal(t, margin.Multi, detail.MarginType, "margin type should match")
	assert.Equal(t, 1.0, detail.RemainingAmount, "remaining amount should match")

	detail, err = h.formatV5OrderDetail(&V5Order{
		ContractCode: "BTC-USDT",
		Side:         "sell",
		Type:         "limit",
		State:        "partially_canceled",
		TimeInForce:  "gtc",
		MarginMode:   "isolated",
	}, asset.USDTMarginedFutures)
	require.NoError(t, err, "formatV5OrderDetail must accept documented partially-cancelled states")
	assert.Equal(t, order.PartiallyCancelled, detail.Status, "partially-cancelled state should be retained")
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	t.Run("V5 execution prices and cost", func(t *testing.T) {
		t.Parallel()
		// Volumes count contracts of 0.001 BTC or 0.01 ETH, so turnover differs from volume times average price.
		type testCase struct {
			name, data                                        string
			pair                                              currency.Pair
			side                                              order.Side
			status                                            order.Status
			price, amount, executed, remaining, cost, average float64
		}
		for _, tc := range []testCase{
			{name: "partially filled sell", data: `"contract_code":"BTC-USDT","side":"sell","state":"partially_filled","price":"100","volume":"4000","trade_volume":"3000","trade_turnover":"303","trade_avg_price":"101"`, pair: btcusdtPair, side: order.Sell, status: order.PartiallyFilled, price: 100, amount: 4000, executed: 3000, remaining: 1000, cost: 303, average: 101},
			{name: "filled sell", data: `"contract_code":"BTC-USDT","side":"sell","state":"filled","price":"100","volume":"3","trade_volume":"3","trade_turnover":"0.303","trade_avg_price":"101"`, pair: btcusdtPair, side: order.Sell, status: order.Filled, price: 100, amount: 3, executed: 3, cost: 0.303, average: 101},
			{name: "partially filled buy", data: `"contract_code":"BTC-USDT","side":"buy","state":"partially_filled","price":"100","volume":"4","trade_volume":"3","trade_turnover":"0.297","trade_avg_price":"99"`, pair: btcusdtPair, side: order.Buy, status: order.PartiallyFilled, price: 100, amount: 4, executed: 3, remaining: 1, cost: 0.297, average: 99},
			{name: "filled buy", data: `"contract_code":"BTC-USDT","side":"buy","state":"filled","price":"100","volume":"3","trade_volume":"3","trade_turnover":"0.297","trade_avg_price":"99"`, pair: btcusdtPair, side: order.Buy, status: order.Filled, price: 100, amount: 3, executed: 3, cost: 0.297, average: 99},
			{name: "ETH-USDT partially filled sell", data: `"contract_code":"ETH-USDT","side":"sell","state":"partially_filled","price":"3000","volume":"4","trade_volume":"3","trade_turnover":"90.03","trade_avg_price":"3001"`, pair: currency.NewPairWithDelimiter("ETH", "USDT", "-"), side: order.Sell, status: order.PartiallyFilled, price: 3000, amount: 4, executed: 3, remaining: 1, cost: 90.03, average: 3001},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/trade/order", `{"code":200,"data":{"order_id":"9","type":"limit","time_in_force":"gtc","margin_mode":"cross",`+tc.data+`}}`, nil)
				got, err := h.GetOrderInfo(t.Context(), "9", tc.pair, asset.USDTMarginedFutures)
				require.NoError(t, err, "V5 order must decode")
				exp := order.Detail{
					Exchange:             h.Name,
					OrderID:              "9",
					Pair:                 tc.pair,
					Type:                 order.Limit,
					Side:                 tc.side,
					TimeInForce:          order.GoodTillCancel,
					Status:               tc.status,
					Price:                tc.price,
					Amount:               tc.amount,
					ExecutedAmount:       tc.executed,
					RemainingAmount:      tc.remaining,
					Cost:                 tc.cost,
					CostAsset:            currency.USDT,
					AverageExecutedPrice: tc.average,
					FeeAsset:             currency.USDT,
					AssetType:            asset.USDTMarginedFutures,
					MarginType:           margin.Multi,
				}
				assert.Equal(t, exp, *got, "V5 order detail should retain HTX's turnover and execution prices")
			})
		}
	})
	t.Run("derivative mapping", func(t *testing.T) {
		t.Parallel()
		const fields = `"volume":4,"price":0.1,"order_price_type":"ioc","direction":"sell","offset":"close","lever_rate":10,"order_id_str":"9","client_order_id":55,"created_at":1604367611263,"trade_volume":3,"trade_turnover":30,"trade_avg_price":0.09,"fee":-0.02,"status":5,"fee_asset":"ADA","liquidation_type":"0","canceled_at":1604367700000`
		type testCase struct {
			name      string
			asset     asset.Item
			pair      currency.Pair
			path      string
			data      string
			costAsset currency.Code
		}
		for _, tc := range []testCase{
			{name: "delivery", asset: asset.Futures, pair: currency.NewPair(currency.ADA, currency.NewCode("201225")), path: "/api/v1/contract_order_info", data: `{"symbol":"ADA","contract_code":"ADA201225","contract_type":"quarter",` + fields + `}`, costAsset: currency.USD},
			{name: "coin margined", asset: asset.CoinMarginedFutures, pair: currency.NewPair(currency.ADA, currency.USD), path: "/swap-api/v1/swap_order_info", data: `{"symbol":"ADA","contract_code":"ADA-USD",` + fields + `}`, costAsset: currency.USD},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, tc.path, `{"status":"ok","data":[`+tc.data+`]}`, nil)
				detail, err := h.GetOrderInfo(t.Context(), "9", tc.pair, tc.asset)
				require.NoError(t, err, "derivative order information must decode")
				exp := order.Detail{
					Exchange:             h.Name,
					OrderID:              "9",
					ClientOrderID:        "55",
					Pair:                 tc.pair,
					Type:                 order.Limit,
					Side:                 order.Sell,
					TimeInForce:          order.ImmediateOrCancel,
					Date:                 time.UnixMilli(1604367611263),
					CloseTime:            time.UnixMilli(1604367700000),
					Status:               order.PartiallyCancelled,
					Price:                0.1,
					Amount:               4,
					ExecutedAmount:       3,
					RemainingAmount:      1,
					Cost:                 30,
					CostAsset:            tc.costAsset,
					AverageExecutedPrice: 0.09,
					Fee:                  0.02,
					FeeAsset:             currency.ADA,
					Leverage:             10,
					ReduceOnly:           true,
					AssetType:            tc.asset,
					MarginType:           margin.Isolated,
					LastUpdated:          time.UnixMilli(1604367700000),
				}
				assert.Equal(t, exp, *detail, "order detail should map every documented field")
			})
		}
	})
	type testCase struct {
		name      string
		asset     asset.Item
		pair      currency.Pair
		path      string
		data      string
		closeTime time.Time
		cost      float64
	}
	for _, tc := range []testCase{
		{
			name: "delivery filled", asset: asset.Futures,
			pair: currency.NewPair(currency.ADA, currency.NewCode("201225")), path: "/api/v1/contract_order_info",
			data: `{"symbol":"ADA","contract_code":"ADA201225","contract_type":"quarter","volume":1,"price":0.0933,"order_price_type":"post_only","direction":"sell","offset":"open","lever_rate":10,"order_id_str":"1","created_at":1604367611263,"trade_volume":1,"trade_turnover":10,"trade_avg_price":0.0933,"status":6,"fee_asset":"ADA","liquidation_type":LIQUIDATION_TYPE,"canceled_at":0}`,
			cost: 10,
		},
		{
			name: "delivery partially cancelled", asset: asset.Futures,
			pair: currency.NewPair(currency.ADA, currency.NewCode("201225")), path: "/api/v1/contract_order_info",
			data:      `{"symbol":"ADA","contract_code":"ADA201225","contract_type":"quarter","volume":2,"price":0.0933,"order_price_type":"post_only","direction":"sell","offset":"open","lever_rate":10,"order_id_str":"1","created_at":1604367611263,"trade_volume":1,"trade_turnover":10,"trade_avg_price":0.0933,"status":5,"fee_asset":"ADA","liquidation_type":LIQUIDATION_TYPE,"canceled_at":1604367700000}`,
			closeTime: time.UnixMilli(1604367700000), cost: 10,
		},
		{
			name: "coin margined cancelled", asset: asset.CoinMarginedFutures,
			pair: currency.NewPair(currency.THETA, currency.USD), path: "/swap-api/v1/swap_order_info",
			data:      `{"symbol":"THETA","contract_code":"THETA-USD","volume":1,"price":0.6,"order_price_type":"post_only","direction":"buy","offset":"open","lever_rate":20,"order_id_str":"1","created_at":1603872714279,"trade_volume":0,"trade_turnover":0,"trade_avg_price":null,"margin_frozen":0.8333333333333334,"status":7,"fee_asset":"THETA","liquidation_type":LIQUIDATION_TYPE,"canceled_at":1603872800000}`,
			closeTime: time.UnixMilli(1603872800000),
		},
	} {
		for _, liquidationType := range []string{`"0"`, `0`} {
			t.Run(tc.name+" liquidation type "+liquidationType, func(t *testing.T) {
				t.Parallel()
				response := `{"status":"ok","data":[` + strings.ReplaceAll(tc.data, "LIQUIDATION_TYPE", liquidationType) + `]}`
				h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, tc.path, response, nil)
				detail, err := h.GetOrderInfo(t.Context(), "1", tc.pair, tc.asset)
				require.NoError(t, err, "derivative order information must decode")
				assert.Equal(t, "1", detail.OrderID, "order ID should match")
				assert.Equal(t, tc.closeTime, detail.CloseTime, "close time should match the cancellation timestamp")
				assert.Equal(t, tc.cost, detail.Cost, "cost should match the executed turnover")
				assert.Equal(t, currency.USD, detail.CostAsset, "inverse contract turnover should be denominated in USD")
			})
		}
	}

	_, err := e.GetOrderInfo(t.Context(), "", btcusdtPair, asset.Spot)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOrderInfo must require an order ID")

	_, err = e.GetOrderInfo(t.Context(), "1", currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOrderInfo must reject empty pairs")

	_, err = e.GetOrderInfo(t.Context(), "1", btcusdtPair, asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderInfo must reject unsupported assets")

	_, err = e.GetOrderInfo(t.Context(), "invalid", btcusdtPair, asset.Spot)
	require.Error(t, err, "GetOrderInfo must reject non-numeric spot order IDs")

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	type validationCase struct {
		name  string
		pair  currency.Pair
		asset asset.Item
	}
	testCases := []validationCase{
		{name: "spot", pair: btcusdtPair, asset: asset.Spot},
		{name: "coin margined futures", pair: btcusdPair, asset: asset.CoinMarginedFutures},
		{name: "usdt margined futures", pair: btcusdtPair, asset: asset.USDTMarginedFutures},
		{name: "futures", pair: btccwPair, asset: asset.Futures},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := h.GetOrderInfo(t.Context(), "1", tt.pair, tt.asset)
			require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "GetOrderInfo must require credentials")
		})
	}
	type orderResponseCase struct {
		name               string
		endpoint           exchange.URL
		method             string
		path               string
		pair               currency.Pair
		asset              asset.Item
		response           string
		expectedMarginMode string
		expectedType       order.Type
		expectedTIF        order.TimeInForce
	}
	for _, tt := range []orderResponseCase{
		{
			name: "spot maker response", endpoint: exchange.RestSpot, method: http.MethodGet, path: "/v1/order/orders/1", pair: btcusdtPair, asset: asset.Spot,
			response:     `{"status":"ok","data":{"id":1,"symbol":"btcusdt","account-id":2,"amount":"2","price":"100","type":"buy-limit-maker","state":"filled","field-amount":"1","field-cash-amount":"100","field-fees":"0.1"}}`,
			expectedType: order.Limit, expectedTIF: order.PostOnly,
		},
		{
			name: "spot IOC response", endpoint: exchange.RestSpot, method: http.MethodGet, path: "/v1/order/orders/1", pair: btcusdtPair, asset: asset.Spot,
			response:     `{"status":"ok","data":{"id":1,"symbol":"btcusdt","amount":"2","price":"100","type":"buy-ioc","state":"filled","field-amount":"1","field-cash-amount":"100","field-fees":"0.1"}}`,
			expectedType: order.Limit, expectedTIF: order.ImmediateOrCancel,
		},
		{
			name: "spot stop limit response", endpoint: exchange.RestSpot, method: http.MethodGet, path: "/v1/order/orders/1", pair: btcusdtPair, asset: asset.Spot,
			response:     `{"status":"ok","data":{"id":1,"symbol":"btcusdt","account-id":2,"amount":"2","price":"100","type":"buy-stop-limit","state":"filled","filled-amount":"1","filled-cash-amount":"100","filled-fees":"0.1"}}`,
			expectedType: order.StopLimit,
		},
		{
			name: "spot limit FOK response", endpoint: exchange.RestSpot, method: http.MethodGet, path: "/v1/order/orders/1", pair: btcusdtPair, asset: asset.Spot,
			response:     `{"status":"ok","data":{"id":1,"symbol":"btcusdt","account-id":2,"amount":"2","price":"100","type":"sell-limit-fok","state":"filled","filled-amount":"1","filled-cash-amount":"100","filled-fees":"0.1"}}`,
			expectedType: order.Limit, expectedTIF: order.FillOrKill,
		},
		{
			name: "spot stop limit FOK response", endpoint: exchange.RestSpot, method: http.MethodGet, path: "/v1/order/orders/1", pair: btcusdtPair, asset: asset.Spot,
			response:     `{"status":"ok","data":{"id":1,"symbol":"btcusdt","account-id":2,"amount":"2","price":"100","type":"buy-stop-limit-fok","state":"filled","filled-amount":"1","filled-cash-amount":"100","filled-fees":"0.1"}}`,
			expectedType: order.StopLimit, expectedTIF: order.FillOrKill,
		},
		{
			name: "coin margined response", endpoint: exchange.RestFutures, method: http.MethodPost, path: "/swap-api/v1/swap_order_info", pair: btcusdPair, asset: asset.CoinMarginedFutures,
			response: `{"status":"ok","data":[{"contract_code":"BTC-USD","volume":2,"price":100,"order_price_type":"limit","direction":"buy","offset":"close","lever_rate":5,"order_id":1,"order_id_str":"1","client_order_id":2,"trade_volume":1,"trade_turnover":100,"fee":-0.1,"trade_avg_price":100,"status":6,"fee_asset":"BTC"}]}`,
		},
		{
			name: "USDT margined response", endpoint: exchange.RestUSDTMargined, method: http.MethodGet, path: "/v5/trade/order", pair: btcusdtPair, asset: asset.USDTMarginedFutures,
			response:           `{"code":200,"data":{"id":"1","contract_code":"BTC-USDT","order_id":"1","client_order_id":"2","side":"buy","type":"limit","state":"filled","time_in_force":"rpi","margin_mode":"isolated","price":"100","volume":"2","trade_volume":"1","trade_turnover":"100","trade_avg_price":"100","fee":"-0.1","fee_currency":"USDT","lever_rate":"5","reduce_only":true}}`,
			expectedMarginMode: "cross",
		},
		{
			name: "delivery futures response", endpoint: exchange.RestFutures, method: http.MethodPost, path: "/api/v1/contract_order_info", pair: btccwPair, asset: asset.Futures,
			response: `{"status":"ok","data":[{"contract_code":"BTC_CW","volume":2,"price":100,"order_price_type":"limit","direction":"buy","offset":"close","lever_rate":5,"order_id":1,"order_id_str":"1","client_order_id":2,"trade_volume":1,"trade_turnover":100,"fee":-0.1,"trade_avg_price":100,"status":6,"fee_asset":"BTC"}]}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var check func(*http.Request)
			if tt.expectedMarginMode != "" {
				check = func(r *http.Request) {
					assert.Equal(t, tt.expectedMarginMode, r.URL.Query().Get("margin_mode"), "GetOrderInfo should query a concrete margin mode")
				}
			}
			h := newHTTPTestExchange(t, tt.endpoint, tt.method, tt.path, tt.response, check)
			detail, err := h.GetOrderInfo(t.Context(), "1", tt.pair, tt.asset)
			require.NoError(t, err, "GetOrderInfo must not error")
			assert.Equal(t, "1", detail.OrderID, "order ID should match")
			assert.Equal(t, 2.0, detail.Amount, "amount should match")
			assert.Equal(t, 1.0, detail.ExecutedAmount, "executed amount should match")
			assert.Equal(t, 0.1, detail.Fee, "charged fee should be positive for every asset")
			assert.Equal(t, 100.0, detail.Cost, "executed cost should match")
			stored := &order.Detail{OrderID: detail.OrderID, Exchange: detail.Exchange, Pair: detail.Pair, AssetType: detail.AssetType}
			require.NoError(t, stored.UpdateOrderFromDetail(detail), "order store must accept the completed order")
			assert.Equal(t, detail.Fee, stored.Fee, "order store should record the charged fee")
			assert.Equal(t, detail.ExecutedAmount, stored.ExecutedAmount, "order store should record executed volume")
			assert.Equal(t, tt.asset, detail.AssetType, "asset type should match")
			if tt.expectedType != order.UnknownType {
				assert.Equal(t, tt.expectedType, detail.Type, "order type should match")
				assert.Equal(t, tt.expectedTIF, detail.TimeInForce, "time in force should match")
			}
		})
	}
	t.Run("USDT isolated fallback after cross error", func(t *testing.T) {
		t.Parallel()
		var calls atomic.Int64
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v5/trade/order", r.URL.Path, "order endpoint should match")
			calls.Add(1)
			if r.URL.Query().Get("margin_mode") == "cross" {
				_, _ = w.Write([]byte(`{"code":1048,"message":"order not found"}`))
				return
			}
			_, _ = w.Write([]byte(`{"code":200,"data":{"contract_code":"BTC-USDT","order_id":"1","side":"buy","type":"limit","state":"filled","time_in_force":"rpi","margin_mode":"isolated","price":"100","volume":"2","trade_volume":"1"}}`))
		}))
		t.Cleanup(server.Close)
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "HTX setup must not error")
		h.API.AuthenticatedSupport = true
		h.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
		require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "USDT-margined endpoint must be set")

		detail, err := h.GetOrderInfo(t.Context(), "1", btcusdtPair, asset.USDTMarginedFutures)
		require.NoError(t, err, "GetOrderInfo must try isolated margin after a cross-margin lookup error")
		assert.Equal(t, margin.Isolated, detail.MarginType, "isolated order should retain its margin type")
		assert.Equal(t, int64(2), calls.Load(), "both margin modes should be queried")
	})
}

func TestFormatExchangeKlineInterval(t *testing.T) {
	t.Parallel()
	type testCase struct {
		interval kline.Interval
		output   string
	}
	for _, tt := range []testCase{
		{kline.OneMin, "1min"},
		{kline.FourHour, "4hour"},
		{kline.OneDay, "1day"},
		{kline.OneWeek, "1week"},
		{kline.OneMonth, "1mon"},
		{kline.OneYear, "1year"},
		{kline.TwoWeek, ""},
	} {
		assert.Equalf(t, tt.output, e.FormatExchangeKlineInterval(tt.interval), "FormatExchangeKlineInterval should return correctly for %s", tt.output)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	type testCase struct {
		asset    asset.Item
		endpoint exchange.URL
		path     string
		pair     currency.Pair
	}
	for _, tc := range []testCase{
		{asset.CoinMarginedFutures, exchange.RestFutures, "/swap-ex/market/history/trade", btcusdPair},
		{asset.USDTMarginedFutures, exchange.RestUSDTMargined, "/linear-swap-ex/market/history/trade", btcusdtPair},
	} {
		t.Run(tc.asset.String()+" nested batches", func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, tc.endpoint, http.MethodGet, tc.path, `{"status":"ok","data":[{"id":1,"ts":1604312615051,"data":[{"id":123,"price":10,"amount":2,"direction":"buy","ts":1604312615051},{"id":124,"price":11,"amount":3,"direction":"sell","ts":1604312615052}]}]}`, nil)
			trades, err := h.GetRecentTrades(t.Context(), tc.pair, tc.asset)
			require.NoError(t, err, "recent trades must decode")
			require.Len(t, trades, 2, "nested executions must be flattened")
			assert.Equal(t, "123", trades[0].TID, "execution ID should be retained")
			assert.Equal(t, 10.0, trades[0].Price, "price should decode")
			assert.Equal(t, 2.0, trades[0].Amount, "amount should decode")
			assert.Equal(t, order.Buy, trades[0].Side, "buy side should decode")
			assert.Equal(t, order.Sell, trades[1].Side, "sell side should decode")
		})
	}
	_, err := e.GetRecentTrades(t.Context(), btcusdtPair, asset.Spot)
	require.NoError(t, err)
	_, err = e.GetRecentTrades(t.Context(), btccwPair, asset.Futures)
	require.NoError(t, err)
	_, err = e.GetRecentTrades(t.Context(), btcusdPair, asset.CoinMarginedFutures)
	require.NoError(t, err)
	_, err = e.GetRecentTrades(t.Context(), btcusdtPair, asset.USDTMarginedFutures)
	require.NoError(t, err)
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricTrades(t.Context(), btcusdtPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestGetV5PositionModeName(t *testing.T) {
	t.Parallel()
	t.Run("context credentials remain independent", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mode := "single_side"
			if r.URL.Query().Get("AccessKeyId") == "second-account" {
				mode = "dual_side"
			}
			_, _ = w.Write([]byte(`{"code":200,"data":{"position_mode":"` + mode + `"}}`))
		}))
		t.Cleanup(server.Close)
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "setup must succeed")
		h.API.AuthenticatedSupport = true
		require.NoError(t, h.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), server.URL), "mock endpoint must be configured")
		type testCase struct{ key, mode string }
		for _, tc := range []testCase{{"first-account", "single_side"}, {"second-account", "dual_side"}} {
			t.Run(tc.key, func(t *testing.T) {
				t.Parallel()
				ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: tc.key, Secret: "test-secret"})
				for range 2 {
					mode, err := h.getV5PositionModeName(ctx)
					require.NoError(t, err, "context account mode must be fetched")
					assert.Equal(t, tc.mode, mode, "mode should belong to the context account")
				}
			})
		}
	})

	type testCase struct {
		name        string
		response    string
		expected    string
		expectedErr error
		wantErr     bool
	}
	for _, tc := range []testCase{
		{name: "success", response: `{"code":200,"data":{"position_mode":"dual_side"}}`, expected: "dual_side"},
		{name: "empty response", response: `null`, expectedErr: errEmptyResult},
		{name: "empty mode", response: `{"code":200,"data":{}}`, expectedErr: errEmptyResult},
		{name: "invalid mode", response: `{"code":200,"data":{"position_mode":"invalid"}}`, expectedErr: errInvalidPositionMode},
		{name: "decode error", response: `{`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/position/mode", tc.response, nil)
			mode, err := h.getV5PositionModeName(t.Context())
			if tc.wantErr || tc.expectedErr != nil {
				if tc.expectedErr != nil {
					require.ErrorIs(t, err, tc.expectedErr, "getV5PositionModeName must return the expected error")
				} else {
					require.Error(t, err, "getV5PositionModeName must return an endpoint error")
				}
				return
			}
			require.NoError(t, err, "getV5PositionModeName must not error")
			assert.Equal(t, tc.expected, mode, "position mode should match")
		})
	}
}

func TestFormatV5OrderRequest(t *testing.T) {
	t.Parallel()
	for _, expiry := range []order.TimeInForce{order.GoodTillTime, order.GoodTillDay, order.GoodTillCrossing, order.StopOrReduce} {
		for _, base := range []order.TimeInForce{order.GoodTillCancel, order.PostOnly} {
			t.Run("unsupported "+(base|expiry).String(), func(t *testing.T) {
				t.Parallel()
				h := new(Exchange)
				require.NoError(t, testexch.Setup(h), "HTX setup must succeed")
				_, err := h.formatV5OrderRequest(&order.Submit{Pair: btcusdtPair, Side: order.Buy, Type: order.Limit, TimeInForce: base | expiry, Price: 100, Amount: 1}, "single_side")
				require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "unsupported expiry flags must not be dropped")
			})
		}
	}
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "HTX setup must not error")
	_, err := h.formatV5OrderRequest(nil, "dual_side")
	require.ErrorIs(t, err, order.ErrSubmissionIsNil, "formatV5OrderRequest must reject nil submissions")

	for _, side := range []order.Side{order.Buy, order.Sell} {
		for _, reduce := range []bool{false, true} {
			for _, mode := range []string{"single_side", "dual_side"} {
				t.Run(fmt.Sprintf("%s reduce %t %s", side, reduce, mode), func(t *testing.T) {
					t.Parallel()
					orderRequest, err := h.formatV5OrderRequest(&order.Submit{Pair: btcusdtPair, AssetType: asset.USDTMarginedFutures, Side: side, Type: order.Market, Amount: 1, ReduceOnly: reduce}, mode)
					require.NoError(t, err, "order formatting must succeed")
					expected := "both"
					if mode == "dual_side" {
						expected = "long"
						if (side == order.Buy && reduce) || (side == order.Sell && !reduce) {
							expected = "short"
						}
					}
					assert.Equal(t, expected, orderRequest.PositionSide, "position side should identify the opened or closed leg")
				})
			}
		}
	}

	type testCase struct {
		name             string
		submit           order.Submit
		expectedType     string
		expectedTIF      string
		expectedMode     string
		expectedPosition string
	}
	for _, tc := range []testCase{
		{
			name: "isolated post only",
			submit: order.Submit{
				Pair:          btcusdtPair,
				AssetType:     asset.USDTMarginedFutures,
				Side:          order.Buy,
				Type:          order.Limit,
				Price:         100,
				Amount:        2,
				MarginType:    margin.Isolated,
				TimeInForce:   order.GoodTillCancel | order.PostOnly,
				ReduceOnly:    true,
				ClientOrderID: "client-id",
			},
			expectedType:     orderPriceTypePostOnly,
			expectedTIF:      "gtc",
			expectedMode:     "isolated",
			expectedPosition: "short",
		},
		{
			name: "cross market FOK",
			submit: order.Submit{
				Pair:        btcusdtPair,
				AssetType:   asset.USDTMarginedFutures,
				Side:        order.Sell,
				Type:        order.Market,
				Amount:      2,
				MarginType:  margin.Multi,
				TimeInForce: order.FillOrKill,
			},
			expectedType:     "market",
			expectedTIF:      "fok",
			expectedMode:     "cross",
			expectedPosition: "short",
		},
		{
			name: "default GTC",
			submit: order.Submit{
				Pair:      btcusdtPair,
				AssetType: asset.USDTMarginedFutures,
				Side:      order.Buy,
				Type:      order.Limit,
				Price:     100,
				Amount:    2,
			},
			expectedType:     "limit",
			expectedTIF:      "gtc",
			expectedMode:     "cross",
			expectedPosition: "long",
		},
		{
			name: "IOC",
			submit: order.Submit{
				Pair:        btcusdtPair,
				AssetType:   asset.USDTMarginedFutures,
				Side:        order.Buy,
				Type:        order.Limit,
				Price:       100,
				Amount:      2,
				TimeInForce: order.ImmediateOrCancel,
			},
			expectedType:     "limit",
			expectedTIF:      "ioc",
			expectedMode:     "cross",
			expectedPosition: "long",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := h.formatV5OrderRequest(&tc.submit, "dual_side")
			require.NoError(t, err, "formatV5OrderRequest must not error")
			assert.Equal(t, "BTC-USDT", got.ContractCode, "contract code should be formatted")
			assert.Equal(t, tc.expectedType, got.Type, "order type should match")
			assert.Equal(t, tc.expectedTIF, got.TimeInForce, "time in force should match")
			assert.Equal(t, tc.expectedMode, got.MarginMode, "margin mode should match")
			assert.Equal(t, tc.expectedPosition, got.PositionSide, "position side should match")
		})
	}

	type validationCase struct {
		name        string
		submit      order.Submit
		expectedErr error
	}
	for _, tc := range []validationCase{
		{name: "empty pair", submit: order.Submit{Side: order.Buy, Type: order.Limit}, expectedErr: currency.ErrCurrencyPairEmpty},
		{name: "market post only", submit: order.Submit{Pair: btcusdtPair, Side: order.Buy, Type: order.Market, TimeInForce: order.PostOnly}, expectedErr: order.ErrUnsupportedTimeInForce},
		{name: "market GTC post only", submit: order.Submit{Pair: btcusdtPair, Side: order.Buy, Type: order.Market, TimeInForce: order.GoodTillCancel | order.PostOnly}, expectedErr: order.ErrUnsupportedTimeInForce},
		{name: "unsupported margin", submit: order.Submit{Pair: btcusdtPair, Side: order.Buy, Type: order.Limit, MarginType: margin.NoMargin}, expectedErr: margin.ErrMarginTypeUnsupported},
		{name: "invalid side", submit: order.Submit{Pair: btcusdtPair, Type: order.Limit}, expectedErr: order.ErrSideIsInvalid},
		{name: "unsupported type", submit: order.Submit{Pair: btcusdtPair, Side: order.Buy, Type: order.Stop}, expectedErr: order.ErrUnsupportedOrderType},
		{name: "unsupported time in force", submit: order.Submit{Pair: btcusdtPair, Side: order.Buy, Type: order.Limit, TimeInForce: order.TimeInForce(1 << 15)}, expectedErr: order.ErrUnsupportedTimeInForce},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := h.formatV5OrderRequest(&tc.submit, "dual_side")
			require.ErrorIs(t, err, tc.expectedErr, "formatV5OrderRequest must return the expected validation error")
		})
	}
	t.Run("unconfigured pair format", func(t *testing.T) {
		t.Parallel()
		_, err := new(Exchange).formatV5OrderRequest(&order.Submit{
			Pair: btcusdtPair,
			Side: order.Buy,
			Type: order.Limit,
		}, "dual_side")
		require.Error(t, err, "formatV5OrderRequest must return pair-format errors")
	})

	t.Run("single-side mode", func(t *testing.T) {
		t.Parallel()
		got, err := h.formatV5OrderRequest(&order.Submit{
			Pair: btcusdtPair,
			Side: order.Buy,
			Type: order.Market,
		}, "single_side")
		require.NoError(t, err, "formatV5OrderRequest must support single-side mode")
		assert.Equal(t, "both", got.PositionSide, "single-side mode should target both positions")
	})
}

func TestWebsocketSubmitOrder(t *testing.T) {
	t.Parallel()
	t.Run("protect mode until submission completes", func(t *testing.T) {
		t.Parallel()
		started, release := make(chan struct{}), make(chan struct{})
		h := newV5TradeWebsocketTestExchange(t, func(tb testing.TB, message []byte, conn *gws.Conn) error {
			tb.Helper()
			var req V5WebsocketTradeRequest
			if err := json.Unmarshal(message, &req); err != nil {
				return err
			}
			if req.Operation == "place_order" {
				close(started)
				<-release
			}
			return wsFixture(tb, message, conn)
		})
		setV5PositionModeEndpoint(t, h, "single_side")
		var releaseOnce sync.Once
		t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
		submission := &order.Submit{Exchange: h.Name, Pair: btcusdtPair, AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Limit, Price: 100, Amount: 1}
		result := make(chan error, 1)
		go func() { _, err := h.WebsocketSubmitOrder(t.Context(), submission); result <- err }()
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "submission must reach the mock transport")
		}
		_, err := h.SetV5PositionMode(t.Context(), "dual_side")
		require.ErrorIs(t, err, errPositionModeInUse, "mode changes must be excluded until the order acknowledgement arrives")
		releaseOnce.Do(func() { close(release) })
		require.NoError(t, <-result, "submission must complete after acknowledgement")
		assert.False(t, h.positionModeInUse.Load(), "completed submission should release the guard")
	})

	t.Run("authentication required", func(t *testing.T) {
		t.Parallel()
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "exchange setup must succeed")
		_, err := h.WebsocketSubmitOrder(t.Context(), &order.Submit{Exchange: h.Name, Pair: btcusdtPair, AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Limit, Price: 100, Amount: 1})
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "WebsocketSubmitOrder must require websocket authentication")
	})
	h := newV5TradeWebsocketTestExchange(t, wsFixture)
	setV5PositionModeEndpoint(t, h, "single_side")
	submission := &order.Submit{
		Exchange: h.Name, Pair: btcusdtPair, AssetType: asset.USDTMarginedFutures,
		Side: order.Buy, Type: order.Limit, Price: 100, Amount: 1,
	}
	resp, err := h.WebsocketSubmitOrder(t.Context(), submission)
	require.NoError(t, err, "WebsocketSubmitOrder must not error")
	assert.Equal(t, "1", resp.OrderID, "order ID should match")
	assert.Equal(t, "2", resp.ClientOrderID, "client order ID should match")
	assert.Equal(t, order.New, resp.Status, "order status should be new")

	submission.AssetType = asset.Spot
	_, err = h.WebsocketSubmitOrder(t.Context(), submission)
	require.ErrorIs(t, err, asset.ErrNotSupported, "WebsocketSubmitOrder must reject unsupported assets")
}

func TestWebsocketSubmitOrders(t *testing.T) {
	t.Parallel()
	t.Run("protect mode until submission completes", func(t *testing.T) {
		t.Parallel()
		started, release := make(chan struct{}), make(chan struct{})
		h := newV5TradeWebsocketTestExchange(t, func(tb testing.TB, message []byte, conn *gws.Conn) error {
			tb.Helper()
			var req V5WebsocketTradeRequest
			if err := json.Unmarshal(message, &req); err != nil {
				return err
			}
			if req.Operation == "place_batch_orders" {
				close(started)
				<-release
			}
			return wsFixture(tb, message, conn)
		})
		setV5PositionModeEndpoint(t, h, "single_side")
		var releaseOnce sync.Once
		t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
		submission := &order.Submit{Exchange: h.Name, Pair: btcusdtPair, AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Limit, Price: 100, Amount: 1}
		result := make(chan error, 1)
		go func() { _, err := h.WebsocketSubmitOrders(t.Context(), []*order.Submit{submission}); result <- err }()
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "submission must reach the mock transport")
		}
		_, err := h.SetV5PositionMode(t.Context(), "dual_side")
		require.ErrorIs(t, err, errPositionModeInUse, "mode changes must be excluded until the order acknowledgement arrives")
		releaseOnce.Do(func() { close(release) })
		require.NoError(t, <-result, "submission must complete after acknowledgement")
		assert.False(t, h.positionModeInUse.Load(), "completed submission should release the guard")
	})

	for _, tc := range []struct{ name, data string }{
		{name: "short acknowledgement", data: `[{"code":0,"order_id":"1","client_order_id":"first"}]`},
		{name: "long acknowledgement", data: `[{"code":0,"order_id":"1"},{"code":0,"order_id":"2"},{"code":0,"order_id":"3"}]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newV5TradeWebsocketTestExchange(t, func(tb testing.TB, message []byte, conn *gws.Conn) error {
				tb.Helper()
				var req V5WebsocketTradeRequest
				if err := json.Unmarshal(message, &req); err != nil {
					return err
				}
				if req.Operation != "place_batch_orders" {
					return wsFixture(tb, message, conn)
				}
				return conn.WriteMessage(gws.TextMessage, []byte(`{"code":200,"cid":"`+req.CID+`","data":`+tc.data+`}`))
			})
			setV5PositionModeEndpoint(t, h, "dual_side")
			orders := make([]*order.Submit, 2)
			for i := range orders {
				orders[i] = &order.Submit{Exchange: h.Name, Pair: btcusdtPair, AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Limit, Price: 100, Amount: 1}
			}
			var err error
			require.NotPanics(t, func() { _, err = h.WebsocketSubmitOrders(t.Context(), orders) }, "mismatched acknowledgements must not panic")
			require.ErrorIs(t, err, errUnexpectedBatchResponseCount, "mismatched acknowledgements must return the sentinel")
		})
	}
	t.Run("authentication required", func(t *testing.T) {
		t.Parallel()
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "exchange setup must succeed")
		_, err := h.WebsocketSubmitOrders(t.Context(), []*order.Submit{{Exchange: h.Name, Pair: btcusdtPair, AssetType: asset.USDTMarginedFutures, Side: order.Buy, Type: order.Limit, Price: 100, Amount: 1}})
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "WebsocketSubmitOrders must require websocket authentication")
	})
	t.Run("mocked mixed batch results", func(t *testing.T) {
		t.Parallel()
		h := newV5TradeWebsocketTestExchange(t, func(tb testing.TB, message []byte, conn *gws.Conn) error {
			tb.Helper()
			var req V5WebsocketTradeRequest
			if err := json.Unmarshal(message, &req); err != nil {
				return err
			}
			if req.Operation != "place_batch_orders" {
				return wsFixture(tb, message, conn)
			}
			return conn.WriteMessage(gws.TextMessage, []byte(`{"code":200,"cid":"`+req.CID+`","data":[{"code":0,"order_id":"1","client_order_id":"first"},{"code":400,"message":"insufficient margin","order_id":"","client_order_id":"rejected"},{"code":200,"order_id":"3","client_order_id":"last"}]}`))
		})
		setV5PositionModeEndpoint(t, h, "single_side")
		orders := make([]*order.Submit, 3)
		for i := range orders {
			orders[i] = &order.Submit{
				Exchange: h.Name, Pair: btcusdtPair, AssetType: asset.USDTMarginedFutures,
				Side: order.Buy, Type: order.Limit, Price: 100, Amount: 1,
			}
		}
		responses, err := h.WebsocketSubmitOrders(t.Context(), orders)
		require.NoError(t, err, "a rejected member must not fail the entire batch")
		require.Len(t, responses, 3, "each submitted order must retain its response")
		type expectedOrder struct{ orderID, clientOrderID string }
		for i, expected := range []expectedOrder{{"1", "first"}, {"", "rejected"}, {"3", "last"}} {
			require.NotNil(t, responses[i], "each response must be populated")
			assert.Equal(t, expected.orderID, responses[i].OrderID, "order IDs should retain batch order")
			assert.Equal(t, expected.clientOrderID, responses[i].ClientOrderID, "client order IDs should retain batch order")
			assert.Equal(t, h.Name, responses[i].Exchange, "each response should identify the exchange")
			if i == 1 {
				assert.ErrorIs(t, responses[i].SubmissionError, errAPIResponse, "rejection should identify an HTX API response error")
				assert.ErrorContains(t, responses[i].SubmissionError, "insufficient margin", "rejection should include the exchange reason")
				assert.EqualError(t, responses[i].SubmissionError, "400: HTX API response error: insufficient margin", "rejection should include its code and reason")
				assert.NotEqual(t, order.New, responses[i].Status, "rejected order should not be marked as accepted")
				continue
			}
			assert.NoError(t, responses[i].SubmissionError, "successful siblings should not have submission errors")
			assert.Equal(t, order.New, responses[i].Status, "successful siblings should retain their accepted status")
		}
	})
	h := newV5TradeWebsocketTestExchange(t, wsFixture)
	setV5PositionModeEndpoint(t, h, "single_side")
	_, err := h.WebsocketSubmitOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "WebsocketSubmitOrders must reject empty submissions")
	responses, err := h.WebsocketSubmitOrders(t.Context(), []*order.Submit{{
		Exchange: h.Name, Pair: btcusdtPair, AssetType: asset.USDTMarginedFutures,
		Side: order.Buy, Type: order.Limit, Price: 100, Amount: 1,
	}})
	require.NoError(t, err, "WebsocketSubmitOrders must not error")
	require.Len(t, responses, 1, "one response must be returned")
	assert.Equal(t, "1", responses[0].OrderID, "order ID should match")
}

func TestWebsocketCancelOrder(t *testing.T) {
	t.Parallel()
	t.Run("authentication required", func(t *testing.T) {
		t.Parallel()
		h := new(Exchange)
		require.NoError(t, testexch.Setup(h), "exchange setup must succeed")
		err := h.WebsocketCancelOrder(t.Context(), &order.Cancel{Pair: btcusdtPair, AssetType: asset.USDTMarginedFutures, OrderID: "1"})
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "WebsocketCancelOrder must require websocket authentication")
	})
	h := newV5TradeWebsocketTestExchange(t, wsFixture)
	require.ErrorIs(t, h.WebsocketCancelOrder(t.Context(), nil), order.ErrCancelOrderIsNil, "WebsocketCancelOrder must reject nil cancellations")
	require.NoError(t, h.WebsocketCancelOrder(t.Context(), &order.Cancel{
		Pair: btcusdtPair, AssetType: asset.USDTMarginedFutures, ClientOrderID: "2",
	}), "WebsocketCancelOrder must accept a client order ID")
	require.ErrorIs(t, h.WebsocketCancelOrder(t.Context(), &order.Cancel{
		Pair: btcusdtPair, AssetType: asset.Spot, OrderID: "1",
	}), asset.ErrNotSupported, "WebsocketCancelOrder must reject unsupported assets")
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	t.Run("empty currencies", func(t *testing.T) {
		t.Parallel()
		h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v2/reference/currencies", `{"code":200,"data":[]}`, nil)
		_, err := h.GetAvailableTransferChains(t.Context(), currency.USDT)
		require.ErrorIs(t, err, errNoTransferChains, "missing currency metadata must return the sentinel")
	})
	c, err := e.GetAvailableTransferChains(t.Context(), currency.USDT)
	require.NoError(t, err)
	require.Greater(t, len(c), 2, "Must get more than 2 chains")
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	t.Run("unsupported asset", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Futures)
		require.ErrorIs(t, err, asset.ErrNotSupported, "GetWithdrawalsHistory must reject unsupported assets")
	})
	h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodGet, "/v1/query/deposit-withdraw", `{
		"status":"ok",
		"data":[{"type":"withdraw","currency":"btc","tx-hash":"tx","chain":"btc","amount":1,"address":"address","fee":0.1,"state":"confirmed","created-at":1772779491561}]
	}`, nil)
	history, err := h.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	require.NoError(t, err, "GetWithdrawalsHistory must not error")
	require.Len(t, history, 1, "one withdrawal must be returned")
	assert.Equal(t, "tx", history[0].TransferID, "transfer ID should match")
	assert.Equal(t, "confirmed", history[0].Status, "status should match")
}

func TestCompatibleVars(t *testing.T) {
	t.Parallel()

	for _, priceType := range []string{"optimal_20", "optimal_20_ioc", "optimal_20_fok", "lightning", "optimal_5", "optimal_10", "opponent_ioc", "opponent_fok"} {
		t.Run(priceType, func(t *testing.T) {
			t.Parallel()
			result, err := compatibleVars("buy", priceType, 3)
			require.NoError(t, err, "documented order type must be accepted")
			assert.Equal(t, order.Market, result.OrderType, "price matching should map to market")
			if strings.HasSuffix(priceType, "_ioc") {
				assert.Equal(t, order.ImmediateOrCancel, result.TimeInForce, "IOC should be retained")
			}
			if strings.HasSuffix(priceType, "_fok") {
				assert.Equal(t, order.FillOrKill, result.TimeInForce, "FOK should be retained")
			}
		})
	}

	type testCase struct {
		name           string
		side           string
		orderPriceType string
		status         uint64
		expectedSide   order.Side
		expectedType   order.Type
		expectedStatus order.Status
		expectedTIF    order.TimeInForce
		wantErr        bool
	}
	testCases := []testCase{
		{name: "buy limit active", side: "buy", orderPriceType: "limit", status: 3, expectedSide: order.Buy, expectedType: order.Limit, expectedStatus: order.Active},
		{name: "buy trigger active", side: "buy", orderPriceType: "trigger", status: 3, expectedSide: order.Buy, expectedType: order.Trigger, expectedStatus: order.Active},
		{name: "sell market filled", side: "sell", orderPriceType: "opponent", status: 6, expectedSide: order.Sell, expectedType: order.Market, expectedStatus: order.Filled},
		{name: "post only cancelled", side: "buy", orderPriceType: "post_only", status: 7, expectedSide: order.Buy, expectedType: order.Limit, expectedStatus: order.Cancelled, expectedTIF: order.PostOnly},
		{name: "invalid side", side: "hold", orderPriceType: "limit", status: 3, wantErr: true},
		{name: "invalid order price type", side: "buy", orderPriceType: "stop", status: 3, wantErr: true},
		{name: "invalid status", side: "buy", orderPriceType: "limit", status: 99, wantErr: true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp, err := compatibleVars(tt.side, tt.orderPriceType, tt.status)
			if tt.wantErr {
				require.Error(t, err, "compatibleVars must reject invalid input")
				return
			}
			require.NoError(t, err, "compatibleVars must not error")
			assert.Equal(t, tt.expectedSide, resp.Side, "side should match")
			assert.Equal(t, tt.expectedType, resp.OrderType, "order type should match")
			assert.Equal(t, tt.expectedStatus, resp.Status, "status should match")
			assert.Equal(t, tt.expectedTIF, resp.TimeInForce, "time in force should match")
		})
	}
}

func TestSetOrderSideStatusAndType(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name           string
		state          string
		requestType    string
		expectedSide   order.Side
		expectedType   order.Type
		expectedStatus order.Status
		expectedTIF    order.TimeInForce
	}
	testCases := []testCase{
		{
			name:           "buy market",
			state:          "filled",
			requestType:    string(SpotOrderTypeBuyMarket),
			expectedSide:   order.Buy,
			expectedType:   order.Market,
			expectedStatus: order.Filled,
		},
		{
			name:           "sell market",
			state:          "partial-filled",
			requestType:    string(SpotOrderTypeSellMarket),
			expectedSide:   order.Sell,
			expectedType:   order.Market,
			expectedStatus: order.PartiallyFilled,
		},
		{
			name:           "buy limit",
			state:          "submitted",
			requestType:    string(SpotOrderTypeBuyLimit),
			expectedSide:   order.Buy,
			expectedType:   order.Limit,
			expectedStatus: order.New,
		},
		{
			name:           "sell limit",
			state:          "canceled",
			requestType:    string(SpotOrderTypeSellLimit),
			expectedSide:   order.Sell,
			expectedType:   order.Limit,
			expectedStatus: order.Cancelled,
		},
		{name: "buy limit-maker", state: "submitted", requestType: "buy-limit-maker", expectedSide: order.Buy, expectedType: order.Limit, expectedStatus: order.New, expectedTIF: order.PostOnly},
		{name: "buy ioc", state: "submitted", requestType: "buy-ioc", expectedSide: order.Buy, expectedType: order.Limit, expectedStatus: order.New, expectedTIF: order.ImmediateOrCancel},
		{name: "buy limit-fok", state: "submitted", requestType: "buy-limit-fok", expectedSide: order.Buy, expectedType: order.Limit, expectedStatus: order.New, expectedTIF: order.FillOrKill},
		{name: "buy stop-limit", state: "submitted", requestType: "buy-stop-limit", expectedSide: order.Buy, expectedType: order.StopLimit, expectedStatus: order.New, expectedTIF: order.UnknownTIF},
		{name: "buy stop-limit-fok", state: "submitted", requestType: "buy-stop-limit-fok", expectedSide: order.Buy, expectedType: order.StopLimit, expectedStatus: order.New, expectedTIF: order.FillOrKill},
		{name: "sell limit-maker", state: "submitted", requestType: "sell-limit-maker", expectedSide: order.Sell, expectedType: order.Limit, expectedStatus: order.New, expectedTIF: order.PostOnly},
		{name: "sell ioc", state: "submitted", requestType: "sell-ioc", expectedSide: order.Sell, expectedType: order.Limit, expectedStatus: order.New, expectedTIF: order.ImmediateOrCancel},
		{name: "sell limit-fok", state: "submitted", requestType: "sell-limit-fok", expectedSide: order.Sell, expectedType: order.Limit, expectedStatus: order.New, expectedTIF: order.FillOrKill},
		{name: "sell stop-limit", state: "submitted", requestType: "sell-stop-limit", expectedSide: order.Sell, expectedType: order.StopLimit, expectedStatus: order.New, expectedTIF: order.UnknownTIF},
		{name: "sell stop-limit-fok", state: "submitted", requestType: "sell-stop-limit-fok", expectedSide: order.Sell, expectedType: order.StopLimit, expectedStatus: order.New, expectedTIF: order.FillOrKill},
		{name: "unknown", state: "invalid", requestType: "invalid", expectedSide: order.UnknownSide, expectedType: order.UnknownType, expectedStatus: order.UnknownStatus},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			detail := &order.Detail{Exchange: e.Name}
			setOrderSideStatusAndType(tt.state, tt.requestType, detail)
			assert.Equal(t, tt.expectedSide, detail.Side, "side should match")
			assert.Equal(t, tt.expectedType, detail.Type, "order type should match")
			assert.Equal(t, tt.expectedStatus, detail.Status, "status should match")
			assert.Equal(t, tt.expectedTIF, detail.TimeInForce, "time in force should match")
		})
	}
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset)
	resp, err := e.GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)

	_, err = e.GetFuturesContractDetails(t.Context(), asset.CoinMarginedFutures)
	require.NoError(t, err)
	_, err = e.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.NoError(t, err)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	t.Run("legacy missing prediction", func(t *testing.T) {
		t.Parallel()
		upcoming := time.Now().UTC().Add(time.Hour).Truncate(time.Millisecond)
		h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodGet, "/swap-api/v1/swap_funding_rate",
			fmt.Sprintf(`{"status":"ok","data":{"contract_code":"BTC-USD","funding_rate":"0.001","funding_time":"%d","next_funding_time":null,"estimated_rate":null}}`, upcoming.UnixMilli()), nil)
		rates, err := h.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.CoinMarginedFutures, Pair: btcusdPair, IncludePredictedRate: true})
		require.NoError(t, err, "legacy funding without a prediction must succeed")
		require.Len(t, rates, 1, "one legacy funding rate must return")
		assert.Equal(t, upcoming, rates[0].TimeOfNextRate.UTC(), "current future settlement should be the next funding time")
		assert.Equal(t, upcoming.Add(-8*time.Hour), rates[0].LatestRate.Time.UTC(), "legacy funding should retain its configured interval")
		assert.True(t, rates[0].PredictedUpcomingRate.Time.IsZero(), "an unavailable prediction should remain absent")
	})
	t.Run("null V5 response", func(t *testing.T) {
		t.Parallel()
		h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/market/funding_rate", `null`, nil)
		_, err := h.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures, Pair: btcusdtPair})
		require.ErrorIs(t, err, errEmptyResult, "null rates must return an error")
	})
	for _, interval := range []time.Duration{time.Hour, 4 * time.Hour, 8 * time.Hour} {
		t.Run(interval.String(), func(t *testing.T) {
			t.Parallel()
			upcoming := time.Now().UTC().Add(time.Hour).Truncate(time.Millisecond)
			next := upcoming.Add(interval)
			h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/market/funding_rate", fmt.Sprintf(`{"code":200,"data":[{"contract_code":"BTC-USDT","funding_rate":"0.001","funding_time":"%d","next_funding_time":"%d"}]}`, upcoming.UnixMilli(), next.UnixMilli()), nil)
			rates, err := h.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures, Pair: btcusdtPair})
			require.NoError(t, err, "per-contract rates must succeed")
			require.Len(t, rates, 1, "one funding rate must return")
			assert.Equal(t, upcoming, rates[0].TimeOfNextRate.UTC(), "upcoming rate time should remain future")
			assert.Equal(t, upcoming.Add(-interval), rates[0].LatestRate.Time.UTC(), "latest rate time should use the contract interval")
		})
	}
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test Instance Setup must not fail")
	updatePairsOnce(t, e)
	err := e.CurrencyPairs.EnablePair(asset.USDTMarginedFutures, currency.NewBTCUSDT())
	if err != nil {
		require.ErrorIs(t, err, currency.ErrPairAlreadyEnabled)
	}

	resp, err := e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 currency.NewBTCUSDT(),
		IncludePredictedRate: true,
	})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "USDT-margined predicted funding rates must be rejected")
	assert.Empty(t, resp, "unsupported predicted funding rates should not be returned")

	resp, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.CoinMarginedFutures,
		Pair:                 currency.NewBTCUSD(),
		IncludePredictedRate: true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)

	err = e.CurrencyPairs.EnablePair(asset.CoinMarginedFutures, currency.NewBTCUSD())
	require.ErrorIs(t, err, currency.ErrPairAlreadyEnabled)

	resp, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{
		Asset:                asset.CoinMarginedFutures,
		IncludePredictedRate: true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	is, err := e.IsPerpetualFutureCurrency(asset.Binary, currency.NewBTCUSDT())
	require.NoError(t, err)
	assert.False(t, is)

	is, err = e.IsPerpetualFutureCurrency(asset.CoinMarginedFutures, currency.NewBTCUSDT())
	require.NoError(t, err)
	assert.True(t, is)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name     string
		item     asset.Item
		pair     currency.Pair
		endpoint exchange.URL
		path     string
		response string
	}
	for _, tc := range []testCase{
		{
			name: "spot", item: asset.Spot, pair: btcusdtPair, endpoint: exchange.RestSpot, path: "/market/tickers",
			response: `{"status":"ok","data":[{"symbol":"btcusdt","open":1,"high":3,"low":1,"close":2,"amount":4,"vol":8,"bid":1.9,"bidSize":2,"ask":2.1,"askSize":3}]}`,
		},
		{
			name: "coin margined futures", item: asset.CoinMarginedFutures, pair: btcusdPair, endpoint: exchange.RestFutures, path: "/v2/swap-ex/market/detail/batch_merged",
			response: `{"status":"ok","ticks":[{"contract_code":"BTC-USD","open":"1","high":"3","low":"1","close":"2","amount":"4","vol":"8","bid":[1.9,2],"ask":[2.1,3]}]}`,
		},
		{
			name: "USDT margined futures", item: asset.USDTMarginedFutures, pair: btcusdtPair, endpoint: exchange.RestFutures, path: "/v2/linear-swap-ex/market/detail/batch_merged",
			response: `{"status":"ok","ticks":[{"contract_code":"BTC-USDT","open":"1","high":"3","low":"1","close":"2","amount":"4","vol":"8","bid":[1.9,2],"ask":[2.1,3]}]}`,
		},
		{
			name: "delivery futures", item: asset.Futures, pair: btccwPair, endpoint: exchange.RestFutures, path: "/v2/market/detail/batch_merged",
			response: `{"status":"ok","ticks":[{"contract_code":"BTC_CW","open":"1","high":"3","low":"1","close":"2","amount":"4","vol":"8","bid":[1.9,2],"ask":[2.1,3]}]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, tc.endpoint, http.MethodGet, tc.path, tc.response, nil)
			h.Name = t.Name()
			require.NoError(t, h.SetPairs(currency.Pairs{tc.pair}, tc.item, false), "available pair must be set")
			require.NoError(t, h.SetPairs(currency.Pairs{tc.pair}, tc.item, true), "enabled pair must be set")
			require.NoError(t, h.UpdateTickers(t.Context(), tc.item), "UpdateTickers must not error")
			price, err := ticker.GetTicker(h.Name, tc.pair, tc.item)
			require.NoError(t, err, "updated ticker must be cached")
			assert.Equal(t, price.Close, price.Last, "Last should carry the closing price")
		})
	}

	t.Run("delivery futures by alias", func(t *testing.T) {
		t.Parallel()
		h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodGet, "/v2/market/detail/batch_merged", `{"status":"ok","ticks":[{"symbol":"BTC_CW","open":"1","high":"3","low":"1","close":"2","amount":"4","vol":"800","number_of":"8","bid":[1.9,2],"ask":[2.1,3]}]}`, nil)
		h.Name = t.Name()
		dated := currency.NewPair(currency.BTC, currency.NewCode("261016"))
		require.NoError(t, h.SetPairs(currency.Pairs{dated}, asset.Futures, false), "dated pair must be available")
		require.NoError(t, h.SetPairs(currency.Pairs{dated}, asset.Futures, true), "dated pair must be enabled")
		h.futureContractCodes = map[string]currency.Code{"CW": dated.Quote}
		require.NoError(t, h.UpdateTickers(t.Context(), asset.Futures), "alias ticker must update")
		price, err := ticker.GetTicker(h.Name, dated, asset.Futures)
		require.NoError(t, err, "alias must resolve to its dated pair")
		assert.Equal(t, 2.0, price.Last, "alias ticker should retain its closing price")
		assert.Equal(t, 800.0, price.QuoteVolume, "batch vol should retain USD turnover")
	})

	type tickerResponseCase struct {
		name     string
		item     asset.Item
		pair     currency.Pair
		path     string
		response string
	}
	for _, tc := range []tickerResponseCase{
		{
			name: "coin margined empty bid", item: asset.CoinMarginedFutures, pair: btcusdPair, path: "/v2/swap-ex/market/detail/batch_merged",
			response: `{"status":"ok","ticks":[{"contract_code":"BTC-USD","bid":[],"ask":[2.1,3]}]}`,
		},
		{
			name: "USDT margined empty ask", item: asset.USDTMarginedFutures, pair: btcusdtPair, path: "/v2/linear-swap-ex/market/detail/batch_merged",
			response: `{"status":"ok","ticks":[{"contract_code":"BTC-USDT","bid":[1.9,2],"ask":[]}]}`,
		},
		{
			name: "delivery futures empty bid", item: asset.Futures, pair: btccwPair, path: "/v2/market/detail/batch_merged",
			response: `{"status":"ok","ticks":[{"contract_code":"BTC_CW","bid":[],"ask":[2.1,3]}]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodGet, tc.path, tc.response, nil)
			require.NoError(t, h.SetPairs(currency.Pairs{tc.pair}, tc.item, false), "available pair must be set")
			otherPair := currency.NewPair(currency.ETH, tc.pair.Quote)
			require.NoError(t, h.SetPairs(currency.Pairs{tc.pair, otherPair}, tc.item, false), "available pairs must be set")
			require.NoError(t, h.SetPairs(currency.Pairs{otherPair}, tc.item, true), "another pair must be enabled")
			require.NoError(t, h.UpdateTickers(t.Context(), tc.item), "malformed disabled ticker must be ignored")
			require.NoError(t, h.SetPairs(currency.Pairs{tc.pair}, tc.item, true), "enabled pair must be set")
			err := h.UpdateTickers(t.Context(), tc.item)
			if strings.Contains(tc.name, "ask") {
				require.ErrorIs(t, err, errInvalidAskData, "UpdateTickers must reject an empty ask")
			} else {
				require.ErrorIs(t, err, errInvalidBidData, "UpdateTickers must reject an empty bid")
			}
		})
	}
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	for _, all := range []bool{false, true} {
		t.Run(fmt.Sprintf("null V5 all=%v", all), func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodGet, "/v5/market/open_interest", `null`, nil)
			for _, a := range h.GetAssetTypes(true) {
				if a != asset.USDTMarginedFutures {
					require.NoError(t, h.CurrencyPairs.SetAssetEnabled(a, false), "other assets must be disabled for null V5 coverage")
				}
			}
			var keys []key.PairAsset
			if !all {
				keys = append(keys, key.PairAsset{Base: currency.BTC.Item, Quote: currency.USDT.Item, Asset: asset.USDTMarginedFutures})
			}
			_, err := h.GetOpenInterest(t.Context(), keys...)
			require.ErrorIs(t, err, errEmptyResult, "null open interest must return an error")
		})
	}
	updatePairsOnce(t, e)

	resp, err := e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.BTC.Item,
		Quote: currency.USDT.Item,
		Asset: asset.USDTMarginedFutures,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.BTC.Item,
		Quote: currency.USD.Item,
		Asset: asset.CoinMarginedFutures,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  btccwPair.Base.Item,
		Quote: btccwPair.Quote.Item,
		Asset: asset.Futures,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = e.GetOpenInterest(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	updatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		require.NotEmptyf(t, pairs, "no pairs for %s", a)
		resp, err := e.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	updatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			require.NoError(t, e.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error")
			pairs, err := e.CurrencyPairs.GetPairs(a, false)
			require.NoError(t, err, "GetPairs must not error")
			require.NotEmpty(t, pairs, "GetPairs must return pairs")
			for _, p := range pairs {
				l, err := e.GetOrderExecutionLimits(a, p)
				require.NoError(t, err, "GetOrderExecutionLimits must not error")
				assert.Positive(t, l.PriceStepIncrementSize, "PriceStepIncrementSize should be positive")
				assert.Positive(t, l.MinimumBaseAmount, "MinimumBaseAmount should be positive")
				assert.Positive(t, l.AmountStepIncrementSize, "AmountStepIncrementSize should be positive")
			}
		})
	}
	t.Run("unsupported asset", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, e.UpdateOrderExecutionLimits(t.Context(), asset.Binary), asset.ErrNotSupported)
	})
}

func TestBootstrap(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test Instance Setup must not fail")

	c, err := e.Bootstrap(t.Context())
	require.NoError(t, err)
	assert.True(t, c, "Bootstrap should return true to continue")

	e.futureContractCodes = nil
	e.Features.Enabled.AutoPairUpdates = false
	_, err = e.Bootstrap(t.Context())
	require.NoError(t, err)
	require.NotNil(t, e.futureContractCodes)
}

var (
	updatePairsMutex         sync.Mutex
	futureContractCodesCache map[string]currency.Code
)

// updatePairsOnce updates the pairs once, and ensures a future dated contract is enabled

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	type marketCase struct {
		assetType asset.Item
		endpoint  exchange.URL
		path      string
		pair      currency.Pair
		limit     int
		response  string
	}
	markets := []marketCase{
		{assetType: asset.Spot, endpoint: exchange.RestSpot, path: "/v1/order/orders/batchcancel", pair: btcusdtPair, limit: 50, response: `{"status":"ok","data":{"success":[],"failed":[]}}`},
		{assetType: asset.Futures, endpoint: exchange.RestFutures, path: "/api/v1/contract_cancel", pair: btccwPair, limit: 10, response: `{"status":"ok","data":{"successes":"","errors":[]}}`},
		{assetType: asset.CoinMarginedFutures, endpoint: exchange.RestFutures, path: "/swap-api/v1/swap_cancel", pair: btcusdPair, limit: 10, response: `{"status":"ok","data":{"successes":"","errors":[]}}`},
		{assetType: asset.USDTMarginedFutures, endpoint: exchange.RestUSDTMargined, path: "/v5/trade/cancel_batch_orders", pair: btcusdtPair, limit: 10, response: `{"code":200,"data":[]}`},
	}
	for _, market := range markets {
		for _, clientIDs := range []bool{false, true} {
			type testCase struct {
				name    string
				count   int
				mixed   bool
				wantErr error
			}
			for _, tc := range []testCase{
				{name: "one", count: 1},
				{name: "maximum", count: market.limit},
				{name: "over maximum", count: market.limit + 1, wantErr: errCancelOrderLimitExceeded},
				{name: "mixed identifier kinds", count: 2, mixed: true, wantErr: errMixedOrderIdentifiers},
			} {
				t.Run(fmt.Sprintf("%s/clientIDs=%t/%s", market.assetType, clientIDs, tc.name), func(t *testing.T) {
					t.Parallel()
					cancels := make([]order.Cancel, tc.count)
					ids := make([]string, tc.count)
					for i := range cancels {
						ids[i] = strconv.Itoa(i + 1)
						cancels[i] = order.Cancel{AssetType: market.assetType, Pair: market.pair, OrderID: ids[i]}
						if clientIDs {
							// An individual order's client ID takes precedence over its exchange ID.
							cancels[i].OrderID = "ignored"
							cancels[i].ClientOrderID = ids[i]
						}
					}
					if tc.mixed {
						if clientIDs {
							cancels[1].ClientOrderID = ""
							cancels[1].OrderID = ids[1]
						} else {
							cancels[1].ClientOrderID = ids[1]
						}
					}
					var calls atomic.Int32
					h := newHTTPTestExchange(t, market.endpoint, http.MethodPost, market.path, market.response, func(r *http.Request) {
						calls.Add(1)
						var body map[string]any
						require.NoError(t, json.NewDecoder(r.Body).Decode(&body), "batch request must decode")
						field, omitted := "order_id", "client_order_id"
						if market.assetType == asset.Spot {
							field, omitted = "order-ids", "client-order-ids"
						}
						if clientIDs {
							field, omitted = omitted, field
						}
						assert.NotContains(t, body, omitted, "batch should contain one identifier kind")
						var sent []string
						if market.assetType == asset.Futures || market.assetType == asset.CoinMarginedFutures {
							joined, ok := body[field].(string)
							require.True(t, ok, "legacy identifiers must decode as a string")
							sent = strings.Split(joined, ",")
						} else {
							values, ok := body[field].([]any)
							require.True(t, ok, "batch identifiers must decode as an array")
							for _, value := range values {
								identifier, ok := value.(string)
								require.True(t, ok, "batch identifiers must be strings")
								sent = append(sent, identifier)
							}
						}
						assert.Equal(t, ids, sent, "one request should contain every selected identifier")
					})
					response, err := h.CancelBatchOrders(t.Context(), cancels)
					if tc.wantErr != nil {
						require.ErrorIs(t, err, tc.wantErr, "invalid batches must return the validation error")
						assert.Zero(t, calls.Load(), "invalid batches should not send a request")
						return
					}
					require.NoError(t, err, "supported batches must succeed")
					require.NotNil(t, response, "successful batches must return statuses")
					assert.Equal(t, int32(1), calls.Load(), "a batch should send exactly one request")
				})
			}
		}
		t.Run(market.assetType.String()+"/API error", func(t *testing.T) {
			t.Parallel()
			body := `{"status":"error","err_code":1000,"err_msg":"batch refused"}`
			switch market.assetType {
			case asset.Spot:
				body = `{"status":"error","err-code":"system-busy","err-msg":"batch refused"}`
			case asset.USDTMarginedFutures:
				body = `{"code":1000,"message":"batch refused"}`
			}
			var calls atomic.Int32
			h := newHTTPTestExchange(t, market.endpoint, http.MethodPost, market.path, body, func(*http.Request) { calls.Add(1) })
			_, err := h.CancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "1", AssetType: market.assetType, Pair: market.pair}})
			require.ErrorContains(t, err, "batch refused", "API errors must reach the caller")
			assert.Equal(t, int32(1), calls.Load(), "a failed batch should send one request")
		})
		if market.assetType == asset.Spot || market.assetType == asset.USDTMarginedFutures {
			t.Run(market.assetType.String()+"/null result", func(t *testing.T) {
				t.Parallel()
				var calls atomic.Int32
				h := newHTTPTestExchange(t, market.endpoint, http.MethodPost, market.path, `null`, func(*http.Request) { calls.Add(1) })
				_, err := h.CancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "1", AssetType: market.assetType, Pair: market.pair}})
				require.ErrorIs(t, err, errEmptyResult, "null responses must return an empty-result error")
				assert.Equal(t, int32(1), calls.Load(), "an empty-result batch should send one request")
			})
		}
	}
	type testCase struct {
		name    string
		orders  []order.Cancel
		wantErr error
	}
	for _, tc := range []testCase{
		{name: "empty", wantErr: order.ErrCancelOrderIsNil},
		{name: "missing identifier", orders: []order.Cancel{{}}, wantErr: order.ErrOrderIDNotSet},
		{name: "mixed assets", orders: []order.Cancel{{OrderID: "1", AssetType: asset.Spot}, {OrderID: "2", AssetType: asset.Futures}}, wantErr: errBatchAssetMismatch},
		{name: "mixed derivative pairs", orders: []order.Cancel{{OrderID: "1", AssetType: asset.Futures, Pair: btccwPair}, {OrderID: "2", AssetType: asset.Futures, Pair: ethusdPair}}, wantErr: errBatchPairMismatch},
		{name: "empty derivative pair", orders: []order.Cancel{{OrderID: "1", AssetType: asset.CoinMarginedFutures}}, wantErr: currency.ErrCurrencyPairEmpty},
		{name: "unsupported asset", orders: []order.Cancel{{OrderID: "1", AssetType: asset.Margin}}, wantErr: asset.ErrNotSupported},
	} {
		t.Run("validation/"+tc.name, func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodPost, "/unused", emptySuccessResponse, func(*http.Request) { calls.Add(1) })
			_, err := h.CancelBatchOrders(t.Context(), tc.orders)
			require.ErrorIs(t, err, tc.wantErr, "invalid requests must return their validation error")
			assert.Zero(t, calls.Load(), "validation failures should not send a request")
		})
	}
	type cancellationFieldsCase struct {
		name string
		data string
		want map[string]string
	}
	for _, tc := range []cancellationFieldsCase{
		{name: "client failures", data: `{"success":[],"failed":[{"client-order-id":"client-1","err-msg":"first rejected"},{"client-order-id":"client-2","err-msg":"second rejected"}]}`, want: map[string]string{"client-1": "first rejected", "client-2": "second rejected"}},
		{name: "partial result", data: `{"success":["client-1"],"failed":[{"client-order-id":"client-2","err-msg":"second rejected"}]}`, want: map[string]string{"client-1": htxStatusSuccess, "client-2": "second rejected"}},
		{name: "exchange ID alias", data: `{"success":["client-1"],"failed":[{"order-id":"42","client-order-id":"client-2","err-msg":"second rejected"}]}`, want: map[string]string{"client-1": htxStatusSuccess, "42": "second rejected", "client-2": "second rejected"}},
		{name: "preserve existing alias status", data: `{"success":["client-2"],"failed":[{"order-id":"42","client-order-id":"client-2","err-msg":"second rejected"}]}`, want: map[string]string{"42": "second rejected", "client-2": htxStatusSuccess}},
		{name: "unrequested alias", data: `{"success":["client-1"],"failed":[{"order-id":"42","client-order-id":"other","err-msg":"second rejected"}]}`, want: map[string]string{"42": "second rejected", "client-1": htxStatusSuccess}},
	} {
		t.Run("spot statuses/"+tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, exchange.RestSpot, http.MethodPost, "/v1/order/orders/batchcancel", `{"status":"ok","data":`+tc.data+`}`, nil)
			response, err := h.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Spot, ClientOrderID: "client-1"}, {AssetType: asset.Spot, ClientOrderID: "client-2"}})
			require.NoError(t, err, "per-order result envelopes must decode")
			require.NotNil(t, response, "per-order statuses must be returned")
			assert.Equal(t, tc.want, response.Status, "statuses should retain individual failures and requested aliases")
		})
	}
	for _, clientIDs := range []bool{false, true} {
		t.Run(fmt.Sprintf("V5 statuses/clientIDs=%t", clientIDs), func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, exchange.RestUSDTMargined, http.MethodPost, "/v5/trade/cancel_batch_orders", `{"code":200,"data":[{"client_order_id":"100","order_id":"900","code":200,"message":"Success"},{"client_order_id":"101","order_id":"901","code":1061,"message":"not found"}]}`, nil)
			cancels := []order.Cancel{{OrderID: "900", AssetType: asset.USDTMarginedFutures, Pair: btcusdtPair}, {OrderID: "901", AssetType: asset.USDTMarginedFutures, Pair: btcusdtPair}}
			want := map[string]string{"900": htxStatusSuccess, "901": "not found"}
			if clientIDs {
				cancels[0].ClientOrderID, cancels[1].ClientOrderID = "100", "101"
				want["100"], want["101"] = htxStatusSuccess, "not found"
			}
			response, err := h.CancelBatchOrders(t.Context(), cancels)
			require.NoError(t, err, "V5 per-order results must decode")
			require.NotNil(t, response, "V5 statuses must be returned")
			assert.Equal(t, want, response.Status, "V5 statuses should retain failures and requested aliases")
		})
	}
	for _, market := range markets[1:3] {
		t.Run(market.assetType.String()+"/per-order rejection", func(t *testing.T) {
			t.Parallel()
			h := newHTTPTestExchange(t, market.endpoint, http.MethodPost, market.path, `{"status":"ok","data":{"successes":"1","errors":[{"order_id":"2","err_msg":"not found"}]}}`, nil)
			response, err := h.CancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "1", AssetType: market.assetType, Pair: market.pair}, {OrderID: "2", AssetType: market.assetType, Pair: market.pair}})
			require.NoError(t, err, "legacy per-order results must decode")
			require.NotNil(t, response, "legacy statuses must be returned")
			assert.Equal(t, map[string]string{"1": htxStatusSuccess, "2": "not found"}, response.Status, "legacy statuses should retain individual rejections")
		})
	}
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	feeBuilder := setFeeBuilder()

	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "Setup must not error")
	_, err := h.GetFeeByType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFeeByType must reject nil fee builder")

	_, err = h.GetFeeByType(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFeeByType must not error")
	assert.Equal(t, exchange.OfflineTradeFee, feeBuilder.FeeType, "fee type should fall back when credentials are not valid")
}

func TestFirstNonZero(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name   string
		values []types.Number
		want   float64
	}
	for _, tc := range []testCase{
		{name: "empty"},
		{name: "zero", values: []types.Number{0, 0}},
		{name: "detail preferred", values: []types.Number{2, 3}, want: 2},
		{name: "open order fallback", values: []types.Number{0, 3}, want: 3},
		{name: "negative", values: []types.Number{0, -0.1}, want: -0.1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, firstNonZero(tc.values...), "first populated value should be retained")
		})
	}
}

// TestBookLevel covers a tick whose book side is short or absent. HTX serves null for the side
// when nothing rests there, so neither element can be indexed unconditionally
func TestBookLevel(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name                string
		level               []float64
		wantPrice, wantSize float64
	}
	for _, tc := range []testCase{
		{name: "price and size", level: []float64{78553, 12}, wantPrice: 78553, wantSize: 12},
		{name: "price only", level: []float64{78553}, wantPrice: 78553},
		{name: "empty side", level: []float64{}},
		{name: "absent side"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			price, size := bookLevel(tc.level)
			assert.Equal(t, tc.wantPrice, price, "bookLevel should return the resting price")
			assert.Equal(t, tc.wantSize, size, "bookLevel should return the resting size")
		})
	}
}

func TestInverseContractQuoteVolume(t *testing.T) {
	t.Parallel()
	t.Run("metadata request fails", func(t *testing.T) {
		t.Parallel()
		h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodGet, "/swap-api/v1/swap_contract_info", `{"status":"error","err-msg":"unavailable"}`, nil)
		h.Name = t.Name()
		_, err := h.inverseContractQuoteVolume(t.Context(), btcusdPair, asset.CoinMarginedFutures, 1)
		require.ErrorIs(t, err, errAPIResponse, "metadata errors must propagate")
	})
	t.Run("metadata lacks requested pair", func(t *testing.T) {
		t.Parallel()
		h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodGet, "/swap-api/v1/swap_contract_info", `{"status":"ok","data":[{"contract_code":"BTC-USD","contract_status":1,"contract_size":100}]}`, nil)
		h.Name = t.Name()
		_, err := h.inverseContractQuoteVolume(t.Context(), currency.NewPair(currency.ETH, currency.USD), asset.CoinMarginedFutures, 1)
		require.ErrorIs(t, err, limits.ErrOrderLimitNotFound, "missing requested contracts must preserve the lookup error")
	})

	for _, tc := range []struct {
		name                        string
		contracts, multiplier, want float64
		wantErr                     error
	}{
		{name: "BTC face value", contracts: 123, multiplier: 100, want: 12300},
		{name: "smaller face value", contracts: 123, multiplier: 10, want: 1230},
		{name: "zero volume", multiplier: 100},
		{name: "invalid multiplier", contracts: 1, wantErr: common.ErrInvalidResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			h.Name = t.Name()
			require.NoError(t, limits.Load([]limits.MinMaxLevel{{Key: key.NewExchangeAssetPair(h.Name, asset.CoinMarginedFutures, btcusdPair), MultiplierDecimal: tc.multiplier}}), "multiplier fixture must load")
			got, err := h.inverseContractQuoteVolume(t.Context(), btcusdPair, asset.CoinMarginedFutures, tc.contracts)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr, "invalid multipliers must fail")
				return
			}
			require.NoError(t, err, "valid volume must convert")
			assert.Equal(t, tc.want, got, "quote turnover should use the contract face value")
		})
	}
	t.Run("load missing metadata", func(t *testing.T) {
		t.Parallel()
		h := newHTTPTestExchange(t, exchange.RestFutures, http.MethodGet, "/swap-api/v1/swap_contract_info", `{"status":"ok","data":[{"contract_code":"BTC-USD","contract_status":1,"contract_size":100}]}`, nil)
		h.Name = t.Name()
		got, err := h.inverseContractQuoteVolume(t.Context(), btcusdPair, asset.CoinMarginedFutures, 2)
		require.NoError(t, err, "missing multiplier must load from contract metadata")
		assert.Equal(t, 200.0, got, "loaded face value should convert contracts")
	})
}
