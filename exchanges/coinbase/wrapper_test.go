package coinbase

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

var (
	// testDatedFuture is the dated US derivative the mock tests trade, whose recorded position the mock holds
	testDatedFuture = currency.NewPairWithDelimiter("BIT", "30OCT26-CDE", currency.DashDelimiter)
	// testWrapperWindowStart starts the window the mock history tests ask for, which their fixtures were recorded with
	testWrapperWindowStart = time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	// testWrapperRecentStart starts the window the mock gateway trade history test asks for, within the 24 hours the
	// gateway's trades by time reach back when the fixture was recorded
	testWrapperRecentStart = time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
)

// wrapperTestPairs returns each asset's available and enabled pairs in the wrapper tests: BTC-USD and ETH-USD on spot,
// a perpetual style and a dated US derivative, and a perpetual of each gateway asset. Live, the dated future is the
// front month's, which expires
func wrapperTestPairs(tb testing.TB) map[asset.Item]currency.Pairs {
	tb.Helper()
	dated := testDatedFuture
	if !mockTests {
		var err error
		dated, err = liveDatedFuture()
		require.NoError(tb, err, "liveDatedFuture must not error")
	}
	return map[asset.Item]currency.Pairs{
		asset.Spot:                {currency.NewBTCUSD(), testETHUSD},
		asset.Futures:             {testFuturesPair, dated},
		asset.USDCMarginedFutures: {testUSDCPerpetual},
		asset.CoinMarginedFutures: {testCoinPerpetual},
	}
}

// liveDatedFuture returns the first dated US derivative open to trading live
var liveDatedFuture = sync.OnceValues(func() (currency.Pair, error) {
	ex := new(Exchange)
	if err := setupTestExchange(ex); err != nil {
		return currency.EMPTYPAIR, err
	}
	products, err := ex.listProducts(context.Background(), asset.Futures)
	if err != nil {
		return currency.EMPTYPAIR, err
	}
	for i := range products {
		if tradableProduct(&products[i]) && !isPerpetualProduct(&products[i]) {
			return currency.NewPairDelimiter(products[i].ProductID, currency.DashDelimiter)
		}
	}
	return currency.EMPTYPAIR, errProductNotFound
})

// datedFuture returns the dated US derivative the wrapper tests trade
func datedFuture(tb testing.TB) currency.Pair {
	tb.Helper()
	return wrapperTestPairs(tb)[asset.Futures][1]
}

// newWrapperTestExchange returns an exchange named after the test with the wrapper test pairs available and enabled
func newWrapperTestExchange(t *testing.T) *Exchange {
	t.Helper()
	ex := newTestExchange(t)
	for a, pairs := range wrapperTestPairs(t) {
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, pairs, false), "StorePairs must not error for available %s pairs", a)
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, pairs, true), "StorePairs must not error for enabled %s pairs", a)
		require.NoErrorf(t, ex.CurrencyPairs.SetAssetEnabled(a, true), "SetAssetEnabled must not error for %s", a)
	}
	return ex
}

// wrapperWindow returns the window a wrapper history test asks for: the mock's recorded one, or the last stretch live
func wrapperWindow(d time.Duration) (start, end time.Time) {
	if mockTests {
		return testWrapperWindowStart, testWrapperWindowStart.Add(d)
	}
	end = time.Now().UTC().Truncate(time.Hour)
	return end.Add(-d), end
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	_, err := ex.FetchTradablePairs(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "FetchTradablePairs must refuse an asset Coinbase does not trade")
	for a, want := range map[asset.Item]currency.Pair{
		asset.Spot:                currency.NewBTCUSD(),
		asset.Futures:             testFuturesPair,
		asset.USDCMarginedFutures: testUSDCPerpetual,
		asset.CoinMarginedFutures: testCoinPerpetual,
	} {
		pairs, err := ex.FetchTradablePairs(t.Context(), a)
		require.NoErrorf(t, err, "FetchTradablePairs must not error for %s", a)
		assert.Truef(t, pairs.Contains(want, true), "FetchTradablePairs should return %s for %s", want, a)
		if !mockTests {
			continue
		}
		exp := map[asset.Item]currency.Pairs{
			asset.Spot:                {currency.NewBTCUSD(), currency.NewPair(currency.BTC, currency.USDC), testETHUSD, testETHUSDC},
			asset.Futures:             {testFuturesPair, testDatedFuture},
			asset.USDCMarginedFutures: {testUSDCPerpetual, testETHUSDCPerpetual},
			asset.CoinMarginedFutures: {testCoinPerpetual},
		}[a]
		assert.Truef(t, exp.Equal(pairs), "FetchTradablePairs should return the tradable %s pairs, got %s", a, pairs)
	}
	assert.True(t, ex.pairAliases.GetAlias(currency.NewBTCUSD()).Contains(currency.NewPair(currency.BTC, currency.USDC), true), "FetchTradablePairs should load BTC-USDC as an alias of BTC-USD")
	perpetual, err := ex.IsPerpetualFutureCurrency(asset.Futures, testFuturesPair)
	require.NoError(t, err, "IsPerpetualFutureCurrency must not error once product data has loaded")
	assert.True(t, perpetual, "FetchTradablePairs should load which US derivatives are perpetual style")
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	require.NoError(t, ex.UpdateTradablePairs(t.Context()), "UpdateTradablePairs must not error")
	for a, pairs := range wrapperTestPairs(t) {
		available, err := ex.GetAvailablePairs(a)
		require.NoErrorf(t, err, "GetAvailablePairs must not error for %s", a)
		assert.Truef(t, available.Contains(pairs[0], true), "UpdateTradablePairs should store %s as available for %s", pairs[0], a)
	}
}

func TestSetDefaultsGatewayAssetsOff(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	for a, enabled := range map[asset.Item]bool{asset.Spot: true, asset.Futures: true, asset.USDCMarginedFutures: false, asset.CoinMarginedFutures: false} {
		assert.Equalf(t, enabled, ex.CurrencyPairs.IsAssetEnabled(a) == nil, "SetDefaults should leave %s enabled %t", a, enabled)
	}
}

func TestUpdateTradablePairsGatewayOutage(t *testing.T) {
	t.Parallel()
	for name, gateway := range map[string]http.HandlerFunc{
		"down for maintenance": func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, `{"jsonrpc":"2.0","error":{"code":11051,"message":"system_maintenance"}}`, http.StatusServiceUnavailable)
		},
		"listing no instruments": func(w http.ResponseWriter, _ *http.Request) {
			writeJSONRPC(t, w, `{"jsonrpc":"2.0","result":[]}`)
		},
	} {
		testUpdateTradablePairsGatewayOutage(t, name, gateway)
	}
}

// testUpdateTradablePairsGatewayOutage updates the tradable pairs while the gateway answers with an outage
func testUpdateTradablePairsGatewayOutage(t *testing.T, name string, gateway http.HandlerFunc) {
	t.Helper()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/brokerage/market/products" {
			gateway(w, r)
			return
		}
		body := `{"products":[{"product_id":"BTC-USD","product_type":"SPOT","status":"online"}],"num_products":1,"pagination":{"has_next":false}}`
		if r.URL.Query().Get("product_type") == ProductTypeFuture {
			body = `{"products":[{"product_id":"BIT-30OCT26-CDE","product_type":"FUTURE","product_venue":"FCM","status":"online","future_product_details":{"contract_expiry_type":"EXPIRING"}}],"num_products":1,"pagination":{"has_next":false}}`
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "writing the response should not error")
	})
	for _, gatewayEnabled := range []bool{false, true} {
		ex := newTestServerExchange(t, handler)
		for _, a := range []asset.Item{asset.USDCMarginedFutures, asset.CoinMarginedFutures} {
			require.NoErrorf(t, ex.CurrencyPairs.SetAssetEnabled(a, gatewayEnabled), "SetAssetEnabled must not error for %s", a)
		}
		err := ex.UpdateTradablePairs(t.Context())
		if gatewayEnabled {
			assert.Errorf(t, err, "UpdateTradablePairs should report an enabled asset whose pairs fail to load, the gateway %s", name)
			continue
		}
		require.NoErrorf(t, err, "UpdateTradablePairs must not fail for a disabled asset whose pairs fail to load, the gateway %s", name)
		for a, p := range map[asset.Item]currency.Pair{asset.Spot: currency.NewBTCUSD(), asset.Futures: currency.NewPairWithDelimiter("BIT", "30OCT26-CDE", currency.DashDelimiter)} {
			available, err := ex.GetAvailablePairs(a)
			require.NoErrorf(t, err, "GetAvailablePairs must not error for %s", a)
			assert.Truef(t, available.Contains(p, true), "UpdateTradablePairs should store %s's pairs despite the gateway %s", a, name)
		}
	}
}

func TestLoadPairAliases(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	btcUSDC := currency.NewPair(currency.BTC, currency.USDC)
	require.NoError(t, ex.loadPairAliases([]ProductResponse{
		{ProductID: "BTC-USD", AliasTo: []string{"BTC-USDC"}},
		{ProductID: "BTC-USDC", Alias: "BTC-USD"},
		{ProductID: "ETH-USD"},
	}), "loadPairAliases must not error")
	assert.True(t, ex.pairAliases.GetAlias(currency.NewBTCUSD()).Equal(currency.Pairs{currency.NewBTCUSD(), btcUSDC}), "loadPairAliases should load a USD product with its own pair and its alias")
	assert.True(t, ex.pairAliases.GetAlias(btcUSDC).Equal(currency.Pairs{btcUSDC}), "loadPairAliases should load an alias as its own alias")
	assert.True(t, ex.pairAliases.GetAlias(testETHUSD).Equal(currency.Pairs{testETHUSD}), "loadPairAliases should load a product without aliases as its own alias")
	for _, p := range []ProductResponse{{ProductID: "BTCUSD"}, {ProductID: "BTC-USD", Alias: "BTCUSD"}, {ProductID: "BTC-USD", AliasTo: []string{"BTCUSDC"}}} {
		assert.Errorf(t, ex.loadPairAliases([]ProductResponse{p}), "loadPairAliases should error for %+v", p)
	}
}

func TestTradableProduct(t *testing.T) {
	t.Parallel()
	assert.True(t, tradableProduct(&ProductResponse{}), "tradableProduct should report a product open to trading")
	for _, p := range []ProductResponse{{TradingDisabled: true}, {IsDisabled: true}, {ViewOnly: true}} {
		assert.Falsef(t, tradableProduct(&p), "tradableProduct should not report %+v", p)
	}
	assert.False(t, isPerpetualProduct(&ProductResponse{}), "isPerpetualProduct should not report a product without futures details")
	assert.False(t, isPerpetualProduct(&ProductResponse{FutureProductDetails: &FutureProductDetails{}}), "isPerpetualProduct should not report a dated future")
	assert.True(t, isPerpetualProduct(&ProductResponse{FutureProductDetails: &FutureProductDetails{FundingInterval: Duration(time.Hour)}}), "isPerpetualProduct should report a future paying funding")
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	require.ErrorIs(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Options), asset.ErrNotSupported, "UpdateOrderExecutionLimits must refuse an asset Coinbase does not trade")
	for a, pairs := range wrapperTestPairs(t) {
		require.NoErrorf(t, ex.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error for %s", a)
		l, err := limits.GetOrderExecutionLimits(key.NewExchangeAssetPair(ex.Name, a, pairs[0]))
		require.NoErrorf(t, err, "GetOrderExecutionLimits must not error for %s %s", a, pairs[0])
		assert.Positivef(t, l.PriceStepIncrementSize, "UpdateOrderExecutionLimits should load a price increment for %s %s", a, pairs[0])
		assert.Positivef(t, l.AmountStepIncrementSize, "UpdateOrderExecutionLimits should load an amount increment for %s %s", a, pairs[0])
		assert.Positivef(t, l.MinimumBaseAmount, "UpdateOrderExecutionLimits should load a minimum amount for %s %s", a, pairs[0])
	}
	if !mockTests {
		return
	}
	for _, tc := range []struct {
		a   asset.Item
		p   currency.Pair
		exp limits.MinMaxLevel
	}{
		{asset.Spot, currency.NewBTCUSD(), limits.MinMaxLevel{
			PriceStepIncrementSize: 0.01, MinimumBaseAmount: 0.00000001, MaximumBaseAmount: 3400, MinimumQuoteAmount: 1,
			MaximumQuoteAmount: 150000000, AmountStepIncrementSize: 0.00000001, QuoteStepIncrementSize: 0.01,
		}},
		{asset.Futures, testFuturesPair, limits.MinMaxLevel{
			PriceStepIncrementSize: 5, MinimumBaseAmount: 1, MaximumBaseAmount: 5000, MaximumQuoteAmount: 100000000,
			AmountStepIncrementSize: 1, QuoteStepIncrementSize: 0.01,
		}},
		{asset.Futures, testDatedFuture, limits.MinMaxLevel{Expiry: time.Date(2026, 10, 30, 16, 0, 0, 0, time.UTC)}},
		{asset.USDCMarginedFutures, testUSDCPerpetual, limits.MinMaxLevel{PriceStepIncrementSize: 0.1, MinimumBaseAmount: 0.0001, AmountStepIncrementSize: 0.0001}},
		{asset.CoinMarginedFutures, testCoinPerpetual, limits.MinMaxLevel{PriceStepIncrementSize: 0.5, MinimumBaseAmount: 10, AmountStepIncrementSize: 10}},
	} {
		l, err := limits.GetOrderExecutionLimits(key.NewExchangeAssetPair(ex.Name, tc.a, tc.p))
		require.NoErrorf(t, err, "GetOrderExecutionLimits must not error for %s %s", tc.a, tc.p)
		if tc.exp.Expiry.IsZero() {
			tc.exp.Key = l.Key
			tc.exp.UpdatedAt = l.UpdatedAt
			assert.Equalf(t, tc.exp, l, "UpdateOrderExecutionLimits should load the %s %s limits Coinbase lists", tc.a, tc.p)
			continue
		}
		assert.Equalf(t, tc.exp.Expiry, l.Expiry, "UpdateOrderExecutionLimits should load %s's expiry", tc.p)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	require.ErrorIs(t, ex.UpdateTickers(t.Context(), asset.Options), asset.ErrNotSupported, "UpdateTickers must refuse an asset Coinbase does not trade")
	for a, pairs := range wrapperTestPairs(t) {
		require.NoErrorf(t, ex.UpdateTickers(t.Context(), a), "UpdateTickers must not error for %s", a)
		for _, p := range pairs {
			tick, err := ticker.GetTicker(ex.Name, p, a)
			require.NoErrorf(t, err, "GetTicker must not error for %s %s", a, p)
			assert.Positivef(t, tick.Last, "UpdateTickers should store a last price for %s %s", a, p)
		}
	}
	if mockTests {
		tick, err := ticker.GetTicker(ex.Name, currency.NewBTCUSD(), asset.Spot)
		require.NoError(t, err, "GetTicker must not error for BTC-USD")
		assert.Equal(t, &ticker.Price{
			ExchangeName:        ex.Name,
			Pair:                currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter),
			AssetType:           asset.Spot,
			Last:                82402.37,
			High:                82529.4,
			Low:                 80564.57,
			Bid:                 81331.92,
			BidSize:             0.40025047,
			Ask:                 81331.93,
			AskSize:             0.00006202,
			BaseVolume:          3018.40419018,
			QuoteVolume:         248723658.89,
			PercentChange24Hour: 0.95019084427705,
			LastUpdated:         tick.LastUpdated,
		}, tick, "UpdateTickers should store BTC-USD's day statistics as the product list reports them and its top of book")
		for _, tc := range []struct {
			a      asset.Item
			p      currency.Pair
			change float64
		}{{asset.USDCMarginedFutures, testUSDCPerpetual, 1.1176}, {asset.CoinMarginedFutures, testCoinPerpetual, 0.9958}} {
			tick, err := ticker.GetTicker(ex.Name, tc.p, tc.a)
			require.NoErrorf(t, err, "GetTicker must not error for %s %s", tc.a, tc.p)
			assert.Equalf(t, tc.change, tick.PercentChange24Hour, "UpdateTickers should store %s %s's 24 hour change as the gateway's book summary reports it", tc.a, tc.p)
		}
	}
	missing := newWrapperTestExchange(t)
	unlisted := currency.NewPair(currency.NewCode("UNLISTED"), currency.USD)
	require.NoError(t, missing.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{unlisted}, false), "StorePairs must not error for available pairs")
	require.NoError(t, missing.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{unlisted}, true), "StorePairs must not error for enabled pairs")
	assert.ErrorIs(t, missing.UpdateTickers(t.Context(), asset.Spot), errProductNotFound, "UpdateTickers should report an enabled pair Coinbase does not list")
}

func TestUpdateTickersTopOfBook(t *testing.T) {
	t.Parallel()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := `{"products":[{"product_id":"BTC-USD","price":"82000","volume_24h":"3000","approximate_quote_24h_volume":"246000000","high_24h":"83000","low_24h":"80000","price_percentage_change_24h":"1","best_bid_price":"","best_ask_price":""}],"num_products":1,"pagination":{"has_next":false}}`
		if r.URL.Path == "/api/v3/brokerage/market/product_book" {
			if r.URL.Query().Get("product_id") != "BTC-USD" || r.URL.Query().Get("limit") != "1" {
				http.Error(w, `{"error":"INVALID_ARGUMENT","message":"unexpected book request"}`, http.StatusBadRequest)
				return
			}
			body = `{"pricebook":{"product_id":"BTC-USD","bids":[{"price":"81999","size":"0.5"}],"asks":[{"price":"82001","size":"0.25"}],"time":"2026-10-20T08:00:00Z"},"last":"82000","mid_market":"82000","spread_bps":"0.24","spread_absolute":"2"}`
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "writing the response should not error")
	})
	for _, stored := range []bool{false, true} {
		ex := newTestServerExchange(t, handler)
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.NewBTCUSD()}, false), "StorePairs must not error for available pairs")
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.NewBTCUSD()}, true), "StorePairs must not error for enabled pairs")
		require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Spot, true), "SetAssetEnabled must not error")
		if stored {
			// The websocket's last ticker before its stream stalled, which made the sync manager fall back to REST
			require.NoError(t, ticker.ProcessTicker(&ticker.Price{ExchangeName: ex.Name, Pair: currency.NewBTCUSD(), AssetType: asset.Spot, Last: 60000, Bid: 59999, BidSize: 1, Ask: 60001, AskSize: 1, LastUpdated: time.Now().Add(-2 * time.Hour)}), "ProcessTicker must not error")
		}
		require.NoError(t, ex.UpdateTickers(t.Context(), asset.Spot), "UpdateTickers must not error")
		tick, err := ticker.GetTicker(ex.Name, currency.NewBTCUSD(), asset.Spot)
		require.NoError(t, err, "GetTicker must not error")
		assert.Equalf(t, 82000.0, tick.Last, "UpdateTickers should store the product list's last price with a stored ticker %v", stored)
		assert.Equalf(t, 81999.0, tick.Bid, "UpdateTickers should store the current best bid with a stored ticker %v", stored)
		assert.Equalf(t, 0.5, tick.BidSize, "UpdateTickers should store the current best bid size with a stored ticker %v", stored)
		assert.Equalf(t, 82001.0, tick.Ask, "UpdateTickers should store the current best ask with a stored ticker %v", stored)
		assert.Equalf(t, 0.25, tick.AskSize, "UpdateTickers should store the current best ask size with a stored ticker %v", stored)
	}
}

func TestListProductsPages(t *testing.T) {
	t.Parallel()
	product := func(id string) string {
		return `{"product_id":"` + id + `","price":"1","base_increment":"0.01","quote_increment":"0.01","base_min_size":"0.01","base_max_size":"100","quote_min_size":"1","quote_max_size":"1000","status":"online","product_type":"SPOT"}`
	}
	var cursors []string
	var mu sync.Mutex
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cursor := r.URL.Query().Get("cursor")
		mu.Lock()
		cursors = append(cursors, cursor)
		mu.Unlock()
		body := `{"products":[` + product("BTC-USD") + `],"num_products":2,"pagination":{"next_cursor":"page-2","has_next":true}}`
		if cursor == "page-2" {
			body = `{"products":[` + product("ETH-USD") + `],"num_products":2,"pagination":{"has_next":false}}`
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "writing the response should not error")
	}))
	products, err := ex.listProducts(t.Context(), asset.Spot)
	require.NoError(t, err, "listProducts must not error")
	ids := make([]string, len(products))
	for i := range products {
		ids[i] = products[i].ProductID
	}
	assert.Equal(t, []string{"BTC-USD", "ETH-USD"}, ids, "listProducts should list every page's products")
	assert.Equal(t, []string{"", "page-2"}, cursors, "listProducts should page by the next cursor")

	var cyclingRequests atomic.Int32
	cycling := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// A list paging its cycle forever is stopped, failing the test
		if cyclingRequests.Add(1) > 10 {
			assert.Fail(t, "listProducts should stop paging once a cursor repeats")
			http.Error(w, `{"error":"INTERNAL","message":"paged in a loop"}`, http.StatusInternalServerError)
			return
		}
		_, err := w.Write([]byte(`{"products":[` + product("BTC-USD") + `],"pagination":{"next_cursor":"same","has_next":true}}`))
		assert.NoError(t, err, "writing the response should not error")
	}))
	_, err = cycling.listProducts(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, errListPagingStalled, "listProducts should stop a list whose cursor repeats")
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.UpdateTicker(t.Context(), currency.NewBTCUSD(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateTicker must refuse an asset Coinbase does not trade")
	_, err = ex.UpdateTicker(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateTicker must refuse an empty pair")
	for a, pairs := range wrapperTestPairs(t) {
		tick, err := ex.UpdateTicker(t.Context(), pairs[0], a)
		require.NoErrorf(t, err, "UpdateTicker must not error for %s %s", a, pairs[0])
		assert.Positivef(t, tick.Last, "UpdateTicker should return a last price for %s %s", a, pairs[0])
		assert.Positivef(t, tick.Bid, "UpdateTicker should return a best bid for %s %s", a, pairs[0])
		assert.Positivef(t, tick.Ask, "UpdateTicker should return a best ask for %s %s", a, pairs[0])
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.UpdateOrderbook(t.Context(), currency.NewBTCUSD(), asset.Options)
	require.Error(t, err, "UpdateOrderbook must refuse an asset Coinbase does not trade")
	for a, pairs := range wrapperTestPairs(t) {
		book, err := ex.UpdateOrderbook(t.Context(), pairs[0], a)
		require.NoErrorf(t, err, "UpdateOrderbook must not error for %s %s", a, pairs[0])
		assert.NotEmptyf(t, book.Bids, "UpdateOrderbook should return bids for %s %s", a, pairs[0])
		assert.NotEmptyf(t, book.Asks, "UpdateOrderbook should return asks for %s %s", a, pairs[0])
	}
}

func TestAdvancedTradeBookLevels(t *testing.T) {
	t.Parallel()
	got := advancedTradeBookLevels([]BookLevel{{Price: 2, Size: 0.5}, {Price: 1, Size: 3}})
	assert.Equal(t, orderbook.Levels{{Price: 2, Amount: 0.5}, {Price: 1, Amount: 3}}, got, "advancedTradeBookLevels should convert levels in order")
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.UpdateAccountBalances(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateAccountBalances must refuse an asset Coinbase does not trade")
	if !mockTests {
		sharedTestCredentials(t, ex)
	}
	for _, a := range []asset.Item{asset.Spot, asset.Futures, asset.USDCMarginedFutures} {
		subAccounts, err := ex.UpdateAccountBalances(t.Context(), a)
		require.NoErrorf(t, err, "UpdateAccountBalances must not error for %s", a)
		require.NotEmptyf(t, subAccounts, "UpdateAccountBalances must return the %s sub-accounts", a)
		for _, s := range subAccounts {
			assert.Equalf(t, a, s.AssetType, "UpdateAccountBalances should return %s sub-accounts", a)
		}
	}
	if !mockTests {
		return
	}
	// The gateway's account backs both perpetual assets, which split its currencies between them, and its isolated margin
	// subaccounts are sub-accounts of their own, by their IDs
	perpetuals, err := ex.UpdateAccountBalances(t.Context(), asset.CoinMarginedFutures)
	require.NoError(t, err, "UpdateAccountBalances must not error for the inverse perpetuals")
	require.Len(t, perpetuals, 2, "UpdateAccountBalances must return the main account and the isolated margin subaccount")
	assert.Empty(t, perpetuals[0].ID, "UpdateAccountBalances should return the main account first, without an ID")
	assert.Equal(t, 302.60065765, perpetuals[0].Balances[currency.BTC].Total, "UpdateAccountBalances should return the main account's balances")
	assert.Equal(t, "123456", perpetuals[1].ID, "UpdateAccountBalances should name the isolated margin subaccount by its ID")
	usdcPerpetuals, err := ex.UpdateAccountBalances(t.Context(), asset.USDCMarginedFutures)
	require.NoError(t, err, "UpdateAccountBalances must not error for the USDC perpetuals")
	require.Len(t, usdcPerpetuals, 1, "UpdateAccountBalances must leave out an isolated margin subaccount holding none of the USDC perpetuals' currencies")
	_, ok := usdcPerpetuals[0].Balances[currency.BTC]
	assert.False(t, ok, "UpdateAccountBalances should count the gateway's bitcoin under the inverse perpetuals alone")
	isolatedETH := perpetuals[1].Balances[currency.ETH]
	assert.Equal(t, 100.0, isolatedETH.Total, "UpdateAccountBalances should return the isolated margin subaccount's balance")
	assert.Equal(t, 99.999597, isolatedETH.Free, "UpdateAccountBalances should return the isolated margin subaccount's withdrawable funds as free")
	assert.InDelta(t, 0.000403, isolatedETH.Hold, 1e-12, "UpdateAccountBalances should hold the isolated margin subaccount's funds that cannot be withdrawn")
	spot, err := ex.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error for spot")
	btc, ok := spot[0].Balances[currency.BTC]
	require.True(t, ok, "UpdateAccountBalances must return the spot BTC balance")
	assert.Equal(t, 0.5, btc.Total, "UpdateAccountBalances should total the available and held BTC")
	assert.Equal(t, 0.5, btc.Free, "UpdateAccountBalances should report the available BTC as free")
	_, ok = spot[0].Balances[currency.ETH]
	assert.False(t, ok, "UpdateAccountBalances should leave a US derivatives account out of spot")
	futuresAccounts, err := ex.UpdateAccountBalances(t.Context(), asset.Futures)
	require.NoError(t, err, "UpdateAccountBalances must not error for futures")
	eth, ok := futuresAccounts[0].Balances[currency.ETH]
	require.True(t, ok, "UpdateAccountBalances must return the US derivatives ETH balance")
	assert.Equal(t, 1.0, eth.Free, "UpdateAccountBalances should report the US derivatives ETH available")
}

// sharedTestCredentials skips a live test that needs credentials when none are set
func sharedTestCredentials(t *testing.T, ex *Exchange) {
	t.Helper()
	if !ex.AreCredentialsValid(t.Context()) {
		t.Skip("API credentials not set")
	}
}

func TestUpdateAccountBalancesClearsRecycledIsolatedSubaccount(t *testing.T) {
	t.Parallel()
	rpcResult := func(result string) string { return `{"jsonrpc":"2.0","id":1,"result":` + result + `}` }
	var calls atomic.Int32
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/private/get_account_summaries" {
			http.NotFound(w, r)
			return
		}
		body := rpcResult(`{"summaries":[{"currency":"USDC","balance":1100,"available_withdrawal_funds":1100}],"isolated_account_summaries":[]}`)
		if calls.Add(1) == 1 {
			body = rpcResult(`{"summaries":[{"currency":"USDC","balance":1000,"available_withdrawal_funds":1000}],"isolated_account_summaries":[{"id":123456,"summaries":[{"currency":"USDC","balance":100,"available_withdrawal_funds":0}]}]}`)
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "writing the response should not error")
	}))
	_, err := ex.UpdateAccountBalances(t.Context(), asset.USDCMarginedFutures)
	require.NoError(t, err, "UpdateAccountBalances must not error while the isolated margin subaccount is active")
	balances, err := ex.GetCachedCurrencyBalances(t.Context(), asset.USDCMarginedFutures)
	require.NoError(t, err, "GetCachedCurrencyBalances must not error")
	assert.Equal(t, 1100.0, balances[currency.USDC].Total, "the cached balances should add the isolated margin subaccount's collateral to the main account's")
	// The position closes: the gateway sweeps the subaccount's collateral back to the main account and stops listing it
	subAccounts, err := ex.UpdateAccountBalances(t.Context(), asset.USDCMarginedFutures)
	require.NoError(t, err, "UpdateAccountBalances must not error once the isolated margin subaccount is recycled")
	assert.Len(t, subAccounts, 1, "UpdateAccountBalances should return the main account alone")
	balances, err = ex.GetCachedCurrencyBalances(t.Context(), asset.USDCMarginedFutures)
	require.NoError(t, err, "GetCachedCurrencyBalances must not error")
	assert.Equal(t, 1100.0, balances[currency.USDC].Total, "the cached balances should not count a recycled subaccount's collateral again")
}

func TestAccountAsset(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		acct Account
		exp  asset.Item
	}{
		{Account{Platform: AccountPlatformConsumer}, asset.Spot},
		{Account{}, asset.Spot},
		{Account{Platform: AccountPlatformUnspecified}, asset.Spot},
		{Account{Platform: AccountPlatformCFMConsumer}, asset.Futures},
		{Account{Platform: AccountPlatformINTX}, asset.Empty},
		{Account{Platform: AccountPlatformConsumer, Type: AccountTypeVault}, asset.Empty},
	} {
		assert.Equalf(t, tc.exp, accountAsset(&tc.acct), "accountAsset should return %s for %+v", tc.exp, tc.acct)
	}
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	if !mockTests {
		sharedTestCredentials(t, ex)
	}
	assert.NoError(t, ex.ValidateAPICredentials(t.Context(), asset.Spot), "ValidateAPICredentials should not error")
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	if !mockTests {
		sharedTestCredentials(t, ex)
	}
	history, err := ex.GetAccountFundingHistory(t.Context())
	require.NoError(t, err, "GetAccountFundingHistory must not error")
	for i := range history {
		assert.Containsf(t, []string{depositTransferType, withdrawalTransferType}, history[i].TransferType, "GetAccountFundingHistory should return only deposits and withdrawals, got %s", history[i].TransferType)
	}
	if !mockTests {
		return
	}
	require.NotEmpty(t, history, "GetAccountFundingHistory must return the recorded send")
	assert.Equal(t, exchange.FundingHistory{
		ExchangeName:    ex.Name,
		Status:          "completed",
		TransferID:      testAppTransactionID,
		Timestamp:       history[0].Timestamp,
		Currency:        "BTC",
		Amount:          0.00100000,
		TransferType:    withdrawalTransferType,
		CryptoToAddress: history[0].CryptoToAddress,
		CryptoTxID:      history[0].CryptoTxID,
		CryptoChain:     history[0].CryptoChain,
		Fee:             history[0].Fee,
	}, history[0], "GetAccountFundingHistory should return the recorded send as a withdrawal")
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	if !mockTests {
		sharedTestCredentials(t, ex)
	}
	withdrawals, err := ex.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	require.NoError(t, err, "GetWithdrawalsHistory must not error")
	for i := range withdrawals {
		assert.Equal(t, withdrawalTransferType, withdrawals[i].TransferType, "GetWithdrawalsHistory should return only withdrawals")
		assert.Equal(t, "BTC", withdrawals[i].Currency, "GetWithdrawalsHistory should return only the currency's withdrawals")
	}
	if mockTests {
		assert.NotEmpty(t, withdrawals, "GetWithdrawalsHistory should return the recorded send")
		none, err := ex.GetWithdrawalsHistory(t.Context(), currency.ETH, asset.Spot)
		require.NoError(t, err, "GetWithdrawalsHistory must not error for a currency without accounts")
		assert.Empty(t, none, "GetWithdrawalsHistory should return nothing for a currency without accounts")
	}
}

func TestFundingTransferType(t *testing.T) {
	t.Parallel()
	for transactionType, exp := range map[string]string{
		AppTransactionTypeReceive:        depositTransferType,
		AppTransactionTypeFiatDeposit:    depositTransferType,
		AppTransactionTypeSend:           withdrawalTransferType,
		AppTransactionTypeFiatWithdrawal: withdrawalTransferType,
	} {
		got, ok := fundingTransferType(&AppTransactionResponse{Type: transactionType})
		require.Truef(t, ok, "fundingTransferType must report %s as a funding transfer", transactionType)
		assert.Equalf(t, exp, got, "fundingTransferType should return %s for %s", exp, transactionType)
	}
	for _, transactionType := range []string{AppTransactionTypeAdvancedTradeFill, AppTransactionTypeTransfer, AppTransactionTypeBuy, ""} {
		_, ok := fundingTransferType(&AppTransactionResponse{Type: transactionType})
		assert.Falsef(t, ok, "fundingTransferType should not report %q as a funding transfer", transactionType)
	}
}

func TestFundingHistory(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.Name = t.Name()
	created := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	got := ex.fundingHistory(&AppTransactionResponse{
		ID:          "tx",
		Status:      AppTransactionStatusCompleted,
		Amount:      AppMoney{Amount: -0.25, Currency: currency.ETH},
		Description: "sent",
		CreatedAt:   created,
		Network:     &AppTransactionNetwork{Hash: "0xhash", Name: "ethereum", TransactionFee: &AppMoney{Amount: 0.001, Currency: currency.ETH}},
		To:          &AppTransactionParty{Address: "0xto"},
		From:        &AppTransactionParty{Address: "0xfrom"},
	}, withdrawalTransferType)
	assert.Equal(t, exchange.FundingHistory{
		ExchangeName:      ex.Name,
		Status:            AppTransactionStatusCompleted,
		TransferID:        "tx",
		Description:       "sent",
		Timestamp:         created,
		Currency:          "ETH",
		Amount:            0.25,
		Fee:               0.001,
		TransferType:      withdrawalTransferType,
		CryptoToAddress:   "0xto",
		CryptoFromAddress: "0xfrom",
		CryptoTxID:        "0xhash",
		CryptoChain:       "ethereum",
	}, got, "fundingHistory should convert a send, its amount unsigned and its network's name as its chain")
	got = ex.fundingHistory(&AppTransactionResponse{Network: &AppTransactionNetwork{NetworkName: "bitcoin", Name: "ignored"}}, depositTransferType)
	assert.Equal(t, "bitcoin", got.CryptoChain, "fundingHistory should prefer the network name Coinbase documents")
}

func TestGetAccountFundingHistoryCreditedSend(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body string
		switch r.URL.Path {
		case "/v2/accounts":
			body = `{"pagination":{"limit":100,"order":"desc"},"data":[{"id":"btc-account","name":"BTC Wallet","primary":true,"type":"wallet","currency":{"code":"BTC","name":"Bitcoin"},"balance":{"amount":"0.4","currency":"BTC"}}]}`
		case "/v2/accounts/btc-account/transactions":
			// send was Coinbase's type for crypto received too before it was restricted to sends; a credit's amount is
			// positive
			body = `{"pagination":{"limit":100,"order":"desc"},"data":[` +
				`{"id":"credit","type":"send","status":"completed","amount":{"amount":"0.50000000","currency":"BTC"},"created_at":"2016-03-11T13:13:35-07:00","resource":"transaction","network":{"status":"confirmed","name":"bitcoin"},"from":{"resource":"bitcoin_network"}},` +
				`{"id":"debit","type":"send","status":"completed","amount":{"amount":"-0.10000000","currency":"BTC"},"created_at":"2016-03-12T13:13:35-07:00","resource":"transaction","network":{"status":"off_blockchain","name":"bitcoin"},"to":{"resource":"bitcoin_address","address":"1AUJ8z5RuHRTqD1eikyfUUetzGmdWLGkpT"}}]}`
		default:
			http.NotFound(w, r)
			return
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "writing the response should not error")
	}))
	history, err := ex.GetAccountFundingHistory(t.Context())
	require.NoError(t, err, "GetAccountFundingHistory must not error")
	require.Len(t, history, 2, "GetAccountFundingHistory must return both sends")
	assert.Equal(t, depositTransferType, history[0].TransferType, "GetAccountFundingHistory should report a send that credited the account as a deposit")
	assert.Equal(t, withdrawalTransferType, history[1].TransferType, "GetAccountFundingHistory should report a send that debited the account as a withdrawal")
	withdrawals, err := ex.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	require.NoError(t, err, "GetWithdrawalsHistory must not error")
	require.Len(t, withdrawals, 1, "GetWithdrawalsHistory must return the debit alone")
	assert.Equal(t, "debit", withdrawals[0].TransferID, "GetWithdrawalsHistory should return the send that debited the account")
}

func TestAppPages(t *testing.T) {
	t.Parallel()
	pages := map[string]*AppListResponse[AppAccountResponse]{
		"":  {Data: []AppAccountResponse{{ID: "a"}, {ID: "b"}}, Pagination: AppPagination{NextURI: "/v2/accounts?starting_after=b"}},
		"b": {Data: []AppAccountResponse{{ID: "c"}}, Pagination: AppPagination{NextURI: "/v2/accounts?starting_after=c", NextStartingAfter: "c"}},
		"c": {Data: []AppAccountResponse{{ID: "d"}}},
	}
	var requested []string
	list := func(page *AppPageRequest) (*AppListResponse[AppAccountResponse], error) {
		assert.Equal(t, uint64(appMaximumPageLimit), page.Limit, "appPages should ask for full pages")
		requested = append(requested, page.StartingAfter)
		require.LessOrEqual(t, len(requested), len(pages), "appPages must not ask for more pages than the list has")
		return pages[page.StartingAfter], nil
	}
	id := func(a *AppAccountResponse) string { return a.ID }
	var visited []string
	require.NoError(t, appPages(list, id, func(a *AppAccountResponse) bool {
		visited = append(visited, a.ID)
		return false
	}), "appPages must not error")
	assert.Equal(t, []string{"a", "b", "c", "d"}, visited, "appPages should visit every record of every page")
	assert.Equal(t, []string{"", "b", "c"}, requested, "appPages should page after the last record, or the cursor sent")

	requested, visited = nil, nil
	require.NoError(t, appPages(list, id, func(a *AppAccountResponse) bool {
		visited = append(visited, a.ID)
		return a.ID == "b"
	}), "appPages must not error when stopped")
	assert.Equal(t, []string{"a", "b"}, visited, "appPages should stop once visit has what it needs")
	assert.Equal(t, []string{""}, requested, "appPages should not request another page once stopped")

	var asked int
	err := appPages(func(page *AppPageRequest) (*AppListResponse[AppAccountResponse], error) {
		asked++
		require.LessOrEqual(t, asked, 3, "appPages must stop a list that does not advance")
		next := "x"
		if page.StartingAfter != "" {
			next = page.StartingAfter
		}
		return &AppListResponse[AppAccountResponse]{Data: []AppAccountResponse{{ID: "x"}}, Pagination: AppPagination{NextURI: "/next", NextStartingAfter: next}}, nil
	}, id, func(*AppAccountResponse) bool { return false })
	assert.ErrorIs(t, err, errListPagingStalled, "appPages should error when a page does not advance")
	asked = 0
	err = appPages(func(page *AppPageRequest) (*AppListResponse[AppAccountResponse], error) {
		asked++
		require.LessOrEqual(t, asked, 4, "appPages must stop a list whose pages cycle")
		next := "x"
		if page.StartingAfter == "x" {
			next = "y"
		}
		return &AppListResponse[AppAccountResponse]{Data: []AppAccountResponse{{ID: next}}, Pagination: AppPagination{NextURI: "/next", NextStartingAfter: next}}, nil
	}, id, func(*AppAccountResponse) bool { return false })
	assert.ErrorIs(t, err, errListPagingStalled, "appPages should error when its pages cycle")
	assert.ErrorIs(t, appPages(func(*AppPageRequest) (*AppListResponse[AppAccountResponse], error) { return nil, errTestJWT }, id, func(*AppAccountResponse) bool { return false }), errTestJWT, "appPages should return a list error")
}

func TestAppAccountID(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	id, err := ex.appAccountID(t.Context(), currency.BTC, "given")
	require.NoError(t, err, "appAccountID must not error for a given account")
	assert.Equal(t, "given", id, "appAccountID should return the account given")
	_, err = ex.appAccountID(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "appAccountID must refuse an empty currency without an account")
	if !mockTests {
		return
	}
	id, err = ex.appAccountID(t.Context(), currency.BTC, "")
	require.NoError(t, err, "appAccountID must not error for a currency with an account")
	assert.Equal(t, testAppAccountID, id, "appAccountID should return the currency's primary account")
	_, err = ex.appAccountID(t.Context(), currency.NewCode("NOWALLET"), "")
	assert.ErrorIs(t, err, errAccountNotFound, "appAccountID should report a currency without an account")
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.GetRecentTrades(t.Context(), currency.NewBTCUSD(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetRecentTrades must refuse an asset Coinbase does not trade")
	for a, pairs := range wrapperTestPairs(t) {
		trades, err := ex.GetRecentTrades(t.Context(), pairs[0], a)
		require.NoErrorf(t, err, "GetRecentTrades must not error for %s %s", a, pairs[0])
		require.NotEmptyf(t, trades, "GetRecentTrades must return %s %s trades", a, pairs[0])
		assert.Truef(t, slices.IsSortedFunc(trades, func(x, y trade.Data) int { return x.Timestamp.Compare(y.Timestamp) }), "GetRecentTrades should sort %s trades oldest first", a)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	start, end := wrapperWindow(time.Minute)
	_, err := ex.GetHistoricTrades(t.Context(), currency.NewBTCUSD(), asset.Spot, end, start)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetHistoricTrades must refuse a window ending before it starts")
	_, err = ex.GetHistoricTrades(t.Context(), currency.NewBTCUSD(), asset.Options, start, end)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricTrades must refuse an asset Coinbase does not trade")
	recentStart, recentEnd := start, end
	if mockTests {
		recentStart, recentEnd = testWrapperRecentStart, testWrapperRecentStart.Add(time.Minute)
	}
	for _, tc := range []struct {
		a          asset.Item
		p          currency.Pair
		start, end time.Time
	}{{asset.Spot, currency.NewBTCUSD(), start, end}, {asset.USDCMarginedFutures, testUSDCPerpetual, recentStart, recentEnd}} {
		start, end := tc.start, tc.end
		trades, err := ex.GetHistoricTrades(t.Context(), tc.p, tc.a, start, end)
		require.NoErrorf(t, err, "GetHistoricTrades must not error for %s %s", tc.a, tc.p)
		require.NotEmptyf(t, trades, "GetHistoricTrades must return %s %s trades", tc.a, tc.p)
		for i := range trades {
			assert.Falsef(t, trades[i].Timestamp.Before(start) || trades[i].Timestamp.After(end), "GetHistoricTrades should return only %s trades within the window, got %s", tc.a, trades[i].Timestamp)
		}
	}
}

func TestPageMarketTrades(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	tradeAt := func(id string, at time.Time) MarketTrade { return MarketTrade{TradeID: id, Time: at} }
	full := func(at time.Time, prefix string) []MarketTrade {
		page := make([]MarketTrade, maximumMarketTradesLimit)
		for i := range page {
			page[i] = tradeAt(fmt.Sprintf("%s%d", prefix, i), at.Add(-time.Duration(i)*time.Millisecond))
		}
		return page
	}
	oldest := end.Add(-10 * time.Minute)
	var ends []time.Time
	trades, err := pageMarketTrades(start, end, func(pageEnd time.Time) ([]MarketTrade, error) {
		ends = append(ends, pageEnd)
		if len(ends) == 1 {
			return full(oldest.Add(time.Second-time.Millisecond), "a"), nil
		}
		return []MarketTrade{tradeAt("a999", oldest), tradeAt("b", oldest.Add(-time.Minute))}, nil
	})
	require.NoError(t, err, "pageMarketTrades must not error")
	assert.Len(t, trades, maximumMarketTradesLimit+1, "pageMarketTrades should keep a trade two pages hold once")
	assert.Equal(t, []time.Time{end, oldest.Add(time.Second)}, ends, "pageMarketTrades should end each page a second after the last page's oldest trade")

	_, err = pageMarketTrades(start, end, func(pageEnd time.Time) ([]MarketTrade, error) { return full(pageEnd.Add(-time.Millisecond), "c"), nil })
	assert.ErrorIs(t, err, errTradesPageFull, "pageMarketTrades should report a page of trades all from its last second")

	reached, err := pageMarketTrades(start, end, func(time.Time) ([]MarketTrade, error) { return full(start.Add(500*time.Millisecond), "d"), nil })
	require.NoError(t, err, "pageMarketTrades must not error once a page reaches the start")
	assert.Len(t, reached, maximumMarketTradesLimit, "pageMarketTrades should stop once a page reaches the start")

	_, err = pageMarketTrades(start, end, func(time.Time) ([]MarketTrade, error) { return nil, errTestJWT })
	assert.ErrorIs(t, err, errTestJWT, "pageMarketTrades should return a page error")
}

func TestPageDerivativesTrades(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	at := func(d time.Duration) types.Time { return types.Time(start.Add(d)) }
	var starts []time.Time
	trades, err := pageDerivativesTrades(start, func(pageStart time.Time) (*DerivativesLastTradesResponse, error) {
		starts = append(starts, pageStart)
		if len(starts) == 1 {
			return &DerivativesLastTradesResponse{Trades: []DerivativesPublicTrade{{TradeID: "1", Timestamp: at(time.Second)}, {TradeID: "2", Timestamp: at(2 * time.Second)}}, HasMore: true}, nil
		}
		return &DerivativesLastTradesResponse{Trades: []DerivativesPublicTrade{{TradeID: "2", Timestamp: at(2 * time.Second)}, {TradeID: "3", Timestamp: at(3 * time.Second)}}}, nil
	})
	require.NoError(t, err, "pageDerivativesTrades must not error")
	assert.Equal(t, []string{"1", "2", "3"}, []string{trades[0].TradeID, trades[1].TradeID, trades[2].TradeID}, "pageDerivativesTrades should keep a trade two pages hold once")
	assert.Equal(t, []time.Time{start, start.Add(2 * time.Second)}, starts, "pageDerivativesTrades should start each page at the last page's last trade")

	_, err = pageDerivativesTrades(start, func(time.Time) (*DerivativesLastTradesResponse, error) {
		return &DerivativesLastTradesResponse{Trades: []DerivativesPublicTrade{{TradeID: "1", Timestamp: at(0)}}, HasMore: true}, nil
	})
	assert.ErrorIs(t, err, errTradesPageFull, "pageDerivativesTrades should report a page that does not advance")
	_, err = pageDerivativesTrades(start, func(time.Time) (*DerivativesLastTradesResponse, error) { return nil, errTestJWT })
	assert.ErrorIs(t, err, errTestJWT, "pageDerivativesTrades should return a page error")
}

func TestMarketTrades(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.Name = t.Name()
	at := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	trades, err := ex.marketTrades([]MarketTrade{{TradeID: "1", Price: 100, Size: 2, Time: at, Side: OrderSideBuy}}, currency.NewBTCUSD(), asset.Spot)
	require.NoError(t, err, "marketTrades must not error")
	assert.Equal(t, []trade.Data{{TID: "1", Exchange: ex.Name, CurrencyPair: currency.NewBTCUSD(), AssetType: asset.Spot, Side: order.Sell, Price: 100, Amount: 2, Timestamp: at}}, trades, "marketTrades should report a trade hitting a resting buy as its taker's sell")
	_, err = ex.marketTrades([]MarketTrade{{TradeID: "1", Side: "SIDEWAYS"}}, currency.NewBTCUSD(), asset.Spot)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "marketTrades should error for an unknown side")
	_, err = ex.derivativesTrades([]DerivativesPublicTrade{{TradeID: "1", Direction: "sideways"}}, testUSDCPerpetual, asset.USDCMarginedFutures)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "derivativesTrades should error for an unknown direction")
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.SubmitOrder(t.Context(), &order.Submit{})
	require.ErrorIs(t, err, common.ErrExchangeNameNotSet, "SubmitOrder must validate the order")
	_, err = ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: currency.NewBTCUSD(), AssetType: asset.Options, Side: order.Buy, Type: order.Market, Amount: 1})
	require.ErrorIs(t, err, asset.ErrNotSupported, "SubmitOrder must refuse an asset Coinbase does not trade")
	if !mockTests {
		sharedTestCredentials(t, ex)
		if !canManipulateRealOrders {
			t.Skip("canManipulateRealOrders is not set")
		}
	}
	resp, err := ex.SubmitOrder(t.Context(), &order.Submit{
		Exchange:      ex.Name,
		Pair:          currency.NewBTCUSD(),
		AssetType:     asset.Spot,
		Side:          order.Sell,
		Type:          order.Limit,
		Amount:        1,
		Price:         70000.44,
		ClientOrderID: "sandbox_success_order",
	})
	require.NoError(t, err, "SubmitOrder must not error for a spot order")
	assert.NotEmpty(t, resp.OrderID, "SubmitOrder should return the spot order's ID")
	assert.Equal(t, "sandbox_success_order", resp.ClientOrderID, "SubmitOrder should return the spot order's client order ID")

	resp, err = ex.SubmitOrder(t.Context(), &order.Submit{
		Exchange:      ex.Name,
		Pair:          testUSDCPerpetual,
		AssetType:     asset.USDCMarginedFutures,
		Side:          order.Buy,
		Type:          order.Limit,
		Amount:        0.001,
		Price:         50000,
		ClientOrderID: "gct-wrapper-order",
	})
	require.NoError(t, err, "SubmitOrder must not error for a gateway order")
	assert.NotEmpty(t, resp.OrderID, "SubmitOrder should return the gateway order's ID")
	assert.Equal(t, order.Open, resp.Status, "SubmitOrder should return the gateway order's status")
}

func TestSubmitOrderRetrieveFees(t *testing.T) {
	t.Parallel()
	const orderID = "b6f0a1c2-3d4e-4f50-8a6b-7c8d9e0f1a2b"
	var lookupFails atomic.Bool
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body string
		switch r.Method + " " + r.URL.Path {
		case http.MethodPost + " /api/v3/brokerage/orders":
			body = `{"success":true,"success_response":{"order_id":"` + orderID + `","product_id":"BTC-USD","side":"BUY","client_order_id":"gct-fees"}}`
		case http.MethodGet + " /api/v3/brokerage/orders/historical/" + orderID:
			if lookupFails.Load() {
				http.Error(w, `{"error":"NOT_FOUND","message":"order not found"}`, http.StatusNotFound)
				return
			}
			body = `{"order":{"order_id":"` + orderID + `","status":"FILLED","filled_size":"0.00012","total_fees":"0.06"}}`
		default:
			assert.Failf(t, "SubmitOrder should only place the order and look it up", "%s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "writing the response should not error")
	}))
	submit := &order.Submit{
		Exchange:      ex.Name,
		Pair:          currency.NewBTCUSD(),
		AssetType:     asset.Spot,
		Side:          order.Buy,
		Type:          order.Market,
		QuoteAmount:   10,
		ClientOrderID: "gct-fees",
		RetrieveFees:  true,
	}
	resp, err := ex.SubmitOrder(t.Context(), submit)
	require.NoError(t, err, "SubmitOrder must not error retrieving the order's fees")
	assert.Equal(t, 0.06, resp.Fee, "SubmitOrder should return the order's fees")
	size, ok := ex.orderSizes.get(orderID)
	require.True(t, ok, "SubmitOrder must record the order's size, which the user channel leaves out")
	assert.Equal(t, 10.0, size.quote, "SubmitOrder should record a quote sized order's size in the quote currency")
	assert.Equal(t, order.Filled, resp.Status, "SubmitOrder should return the state of the order it read")

	submit.RetrieveFees = false
	resp, err = ex.SubmitOrder(t.Context(), submit)
	require.NoError(t, err, "SubmitOrder must not error")
	assert.Equal(t, order.New, resp.Status, "SubmitOrder should report an accepted market order new, as Coinbase's response holds no fills")
	assert.Zero(t, resp.Fee, "SubmitOrder should not report fees it did not read")
	submit.RetrieveFees = true

	lookupFails.Store(true)
	resp, err = ex.SubmitOrder(t.Context(), submit)
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "SubmitOrder should report the failed fee lookup")
	require.NotNil(t, resp, "SubmitOrder must return the placed order when its fees cannot be read")
	assert.Equal(t, orderID, resp.OrderID, "SubmitOrder should return the placed order's ID when its fees cannot be read")
}

func TestSubmitOrderClientOrderIDInUse(t *testing.T) {
	t.Parallel()
	const orderID = "e3b0c442-98fc-4c14-9afb-f4c8996fb924"
	limit := order.Submit{Side: order.Buy, Type: order.Limit, Amount: 1, Price: 90000}
	// A USDC alias trades on its USD product's book, which a reply could name in its place
	usdcLimit := order.Submit{Pair: currency.NewPair(currency.BTC, currency.USDC), Side: order.Buy, Type: order.Limit, Amount: 1, Price: 90000}
	quoteSized := order.Submit{Side: order.Buy, Type: order.Market, QuoteAmount: 100}
	stopLimit := order.Submit{Side: order.Buy, Type: order.StopLimit, Amount: 1, Price: 90000, TriggerPrice: 89000, StopDirection: order.StopUp}
	bracket := order.Submit{Side: order.Sell, Type: order.Bracket, Amount: 1, Price: 95000, TriggerPrice: 85000}
	for _, tc := range []struct {
		name                   string
		submit                 order.Submit
		product, side, replied string
		inUse                  bool
	}{
		{"another side", limit, "BTC-USD", "SELL", `null`, true},
		{"another product", limit, "ETH-USD", "BUY", `null`, true},
		{"the product's book", usdcLimit, "BTC-USD", "BUY", `null`, false},
		{"the alias's book", limit, "BTC-USDC", "BUY", `null`, false},
		{"another base size", limit, "BTC-USD", "BUY", `{"limit_limit_gtc":{"base_size":"2","limit_price":"90000"}}`, true},
		{"another limit price", limit, "BTC-USD", "BUY", `{"limit_limit_gtc":{"base_size":"1","limit_price":"80000"}}`, true},
		{"another quote size", quoteSized, "BTC-USD", "BUY", `{"market_market_ioc":{"quote_size":"50"}}`, true},
		{"another stop price", stopLimit, "BTC-USD", "BUY", `{"stop_limit_stop_limit_gtc":{"base_size":"1","limit_price":"90000","stop_price":"88000","stop_direction":"STOP_DIRECTION_STOP_UP"}}`, true},
		{"another trigger price", bracket, "BTC-USD", "SELL", `{"trigger_bracket_gtc":{"base_size":"1","limit_price":"95000","stop_trigger_price":"84000"}}`, true},
		{"the same order", limit, "BTC-USD", "BUY", `{"limit_limit_gtc":{"base_size":"1.0","limit_price":"90000.00","post_only":false,"rfq_disabled":false}}`, false},
		{"no configuration", limit, "BTC-USD", "BUY", `null`, false},
		{"a size the request leaves out", quoteSized, "BTC-USD", "BUY", `{"market_market_ioc":{"quote_size":"100","base_size":"0.001"}}`, false},
		{"no side", limit, "BTC-USD", "", `null`, false},
		{"no size or price", limit, "BTC-USD", "BUY", `{"limit_limit_gtc":{}}`, false},
		{"no quote size", quoteSized, "BTC-USD", "BUY", `{"market_market_ioc":{}}`, false},
		{"no stop price", stopLimit, "BTC-USD", "BUY", `{"stop_limit_stop_limit_gtc":{"base_size":"1","limit_price":"90000"}}`, false},
		{"no limit price", stopLimit, "BTC-USD", "BUY", `{"stop_limit_stop_limit_gtc":{"base_size":"1","stop_price":"89000"}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !assert.Equal(t, http.MethodPost+" /api/v3/brokerage/orders", r.Method+" "+r.URL.Path, "SubmitOrder should only place the order") {
					http.NotFound(w, r)
					return
				}
				_, err := w.Write([]byte(`{"success":true,"success_response":{"order_id":"` + orderID + `","product_id":"` + tc.product + `","side":"` + tc.side + `","client_order_id":"reused"},"order_configuration":` + tc.replied + `}`))
				assert.NoError(t, err, "writing the response should not error")
			}))
			// The record holds the order the client order ID already names
			ex.orderSizes.set(orderID, 3, 0)
			s := tc.submit
			s.Exchange, s.AssetType, s.ClientOrderID = ex.Name, asset.Spot, "reused"
			if s.Pair.IsEmpty() {
				s.Pair = currency.NewBTCUSD()
			}
			resp, err := ex.SubmitOrder(t.Context(), &s)
			size, ok := ex.orderSizes.get(orderID)
			require.True(t, ok, "the order's size must stay recorded")
			if tc.inUse {
				require.ErrorIs(t, err, errClientOrderIDInUse, "SubmitOrder must refuse a reply naming another order than the one asked for")
				assert.Nil(t, resp, "SubmitOrder should return no order for one it did not place")
				assert.Equal(t, 3.0, size.base, "SubmitOrder should keep the size of the order the reply names")
				return
			}
			require.NoError(t, err, "SubmitOrder must not error for a reply naming the order asked for")
			assert.Equal(t, orderID, resp.OrderID, "SubmitOrder should return the order's ID")
			assert.Equal(t, s.Amount, size.base, "SubmitOrder should record the order's base size")
			assert.Equal(t, s.QuoteAmount, size.quote, "SubmitOrder should record the order's quote size")
		})
	}
}

func TestSubmitOrderRetrieveFeesExecution(t *testing.T) {
	t.Parallel()
	limit := order.Submit{Type: order.Limit, Amount: 2, Price: 90000}
	for _, tc := range []struct {
		name                                                  string
		submit                                                order.Submit
		read                                                  string
		status                                                order.Status
		amount, remaining, average, cost, fee, price, trigger float64
	}{
		{
			name:   "partly filled",
			submit: limit,
			read:   `"order_type":"LIMIT","status":"OPEN","time_in_force":"GOOD_UNTIL_CANCELLED","order_configuration":{"limit_limit_gtc":{"base_size":"2","limit_price":"90000"}},"filled_size":"0.5","filled_value":"44950","average_filled_price":"89900","total_fees":"12"`,
			status: order.PartiallyFilled, amount: 2, remaining: 1.5, average: 89900, cost: 44950, fee: 12, price: 90000,
		},
		{
			name:   "resized by an edit before it was read",
			submit: limit,
			read:   `"order_type":"LIMIT","status":"OPEN","time_in_force":"GOOD_UNTIL_CANCELLED","order_configuration":{"limit_limit_gtc":{"base_size":"3","limit_price":"90000"}},"filled_size":"0.5","filled_value":"44950","average_filled_price":"89900","total_fees":"12"`,
			status: order.PartiallyFilled, amount: 3, remaining: 2.5, average: 89900, cost: 44950, fee: 12, price: 90000,
		},
		{
			name:   "repriced by an edit before it was read",
			submit: limit,
			read:   `"order_type":"LIMIT","status":"OPEN","time_in_force":"GOOD_UNTIL_CANCELLED","order_configuration":{"limit_limit_gtc":{"base_size":"2","limit_price":"91000"}},"filled_size":"0","total_fees":"0"`,
			status: order.Open, amount: 2, remaining: 2, price: 91000,
		},
		{
			name:   "unfilled",
			submit: limit,
			read:   `"order_type":"LIMIT","status":"OPEN","time_in_force":"GOOD_UNTIL_CANCELLED","order_configuration":{"limit_limit_gtc":{"base_size":"2","limit_price":"90000"}},"filled_size":"0","total_fees":"0"`,
			status: order.Open, amount: 2, remaining: 2, price: 90000,
		},
		{
			name:   "a filled market buy sized in the quote currency",
			submit: order.Submit{Type: order.Market, QuoteAmount: 100},
			read:   `"order_type":"MARKET","status":"FILLED","time_in_force":"IMMEDIATE_OR_CANCEL","order_configuration":{"market_market_ioc":{"quote_size":"100"}},"size_in_quote":true,"filled_size":"0.001","filled_value":"99.9","average_filled_price":"99900","total_fees":"0.1"`,
			status: order.Filled, amount: 0.001, average: 99900, cost: 99.9, fee: 0.1,
		},
		{
			name:   "a filled limit buy sized in the quote currency",
			submit: order.Submit{Type: order.Limit, QuoteAmount: 100, Price: 100000},
			read:   `"order_type":"LIMIT","status":"FILLED","time_in_force":"GOOD_UNTIL_CANCELLED","order_configuration":{"limit_limit_gtc":{"quote_size":"100","limit_price":"100000"}},"size_in_quote":true,"filled_size":"0.001","filled_value":"100","average_filled_price":"100000","total_fees":"0.1"`,
			status: order.Filled, amount: 0.001, average: 100000, cost: 100, fee: 0.1, price: 100000,
		},
		{
			name:   "a partly filled limit buy sized in the quote currency",
			submit: order.Submit{Type: order.Limit, QuoteAmount: 100, Price: 100000},
			read:   `"order_type":"LIMIT","status":"OPEN","time_in_force":"GOOD_UNTIL_CANCELLED","order_configuration":{"limit_limit_gtc":{"quote_size":"100","limit_price":"100000"}},"size_in_quote":true,"filled_size":"0.0005","filled_value":"49.95","average_filled_price":"99900","total_fees":"0.05"`,
			status: order.PartiallyFilled, average: 99900, cost: 49.95, fee: 0.05, price: 100000,
		},
		{
			name:   "a stop limit order whose stop an edit moved before it was read",
			submit: order.Submit{Type: order.StopLimit, Amount: 2, Price: 90000, TriggerPrice: 89000, StopDirection: order.StopDown},
			read:   `"order_type":"STOP_LIMIT","status":"OPEN","time_in_force":"GOOD_UNTIL_CANCELLED","order_configuration":{"stop_limit_stop_limit_gtc":{"base_size":"2","limit_price":"90000","stop_price":"88500","stop_direction":"STOP_DIRECTION_STOP_DOWN"}},"filled_size":"0","total_fees":"0"`,
			status: order.Open, amount: 2, remaining: 2, price: 90000, trigger: 88500,
		},
		{
			name:   "a stop limit order an edit repriced down and its stop moved up before it was read",
			submit: order.Submit{Type: order.StopLimit, Amount: 2, Price: 90000, TriggerPrice: 89000, StopDirection: order.StopDown},
			read:   `"order_type":"STOP_LIMIT","status":"OPEN","time_in_force":"GOOD_UNTIL_CANCELLED","order_configuration":{"stop_limit_stop_limit_gtc":{"base_size":"2","limit_price":"89500","stop_price":"89200","stop_direction":"STOP_DIRECTION_STOP_DOWN"}},"filled_size":"0","total_fees":"0"`,
			status: order.Open, amount: 2, remaining: 2, price: 89500, trigger: 89200,
		},
		{
			name:   "a bracket order an edit repriced up and its stop moved down before it was read",
			submit: order.Submit{Type: order.Bracket, Amount: 2, Price: 85000, TriggerPrice: 95000},
			read:   `"order_type":"BRACKET","status":"OPEN","time_in_force":"GOOD_UNTIL_CANCELLED","order_configuration":{"trigger_bracket_gtc":{"base_size":"2","limit_price":"86000","stop_trigger_price":"94500"}},"filled_size":"0","total_fees":"0"`,
			status: order.Open, amount: 2, remaining: 2, price: 86000, trigger: 94500,
		},
		{
			name:   "a bracket order an edit repriced down and its stop moved up before it was read",
			submit: order.Submit{Type: order.Bracket, Amount: 2, Price: 85000, TriggerPrice: 95000},
			read:   `"order_type":"BRACKET","status":"OPEN","time_in_force":"GOOD_UNTIL_CANCELLED","order_configuration":{"trigger_bracket_gtc":{"base_size":"2","limit_price":"84000","stop_trigger_price":"95500"}},"filled_size":"0","total_fees":"0"`,
			status: order.Open, amount: 2, remaining: 2, price: 84000, trigger: 95500,
		},
		{
			name:   "a filled market buy priced by the caller",
			submit: order.Submit{Type: order.Market, Amount: 0.001, Price: 99000},
			read:   `"order_type":"MARKET","status":"FILLED","time_in_force":"IMMEDIATE_OR_CANCEL","order_configuration":{"market_market_ioc":{"base_size":"0.001"}},"filled_size":"0.001","filled_value":"99.9","average_filled_price":"99900","total_fees":"0.1"`,
			status: order.Filled, amount: 0.001, average: 99900, cost: 99.9, fee: 0.1, price: 99000,
		},
		{
			name:   "a stop limit order read with a configuration this package does not decode",
			submit: order.Submit{Type: order.StopLimit, Amount: 2, Price: 90000, TriggerPrice: 89000, StopDirection: order.StopDown},
			read:   `"order_type":"STOP_LIMIT","status":"OPEN","time_in_force":"GOOD_UNTIL_CANCELLED","order_configuration":{},"filled_size":"0","total_fees":"0"`,
			status: order.Open, amount: 2, price: 90000, trigger: 89000,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			const orderID = "c7a1b2d3-4e5f-4a6b-9c8d-7e6f5a4b3c2d"
			ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body string
				switch r.Method + " " + r.URL.Path {
				case http.MethodPost + " /api/v3/brokerage/orders":
					body = `{"success":true,"success_response":{"order_id":"` + orderID + `","product_id":"BTC-USD","side":"BUY","client_order_id":"gct-read"}}`
				case http.MethodGet + " /api/v3/brokerage/orders/historical/" + orderID:
					body = `{"order":{"order_id":"` + orderID + `","product_id":"BTC-USD","product_type":"SPOT","side":"BUY",` + tc.read + `}}`
				default:
					assert.Failf(t, "SubmitOrder should only place the order and read it", "%s %s", r.Method, r.URL.Path)
					http.NotFound(w, r)
					return
				}
				_, err := w.Write([]byte(body))
				assert.NoError(t, err, "writing the response should not error")
			}))
			s := tc.submit
			s.Exchange, s.Pair, s.AssetType, s.Side, s.ClientOrderID, s.RetrieveFees = ex.Name, currency.NewBTCUSD(), asset.Spot, order.Buy, "gct-read", true
			resp, err := ex.SubmitOrder(t.Context(), &s)
			require.NoError(t, err, "SubmitOrder must not error reading the order")
			assert.Equal(t, tc.status, resp.Status, "SubmitOrder should return the status of the order it read")
			assert.Equal(t, tc.amount, resp.Amount, "SubmitOrder should return the size of the order it read")
			assert.Equal(t, tc.remaining, resp.RemainingAmount, "SubmitOrder should return what remains of the order it read")
			assert.Equal(t, tc.average, resp.AverageExecutedPrice, "SubmitOrder should return the average price the order it read filled at")
			assert.Equal(t, tc.cost, resp.Cost, "SubmitOrder should return what the order it read cost")
			assert.Equal(t, tc.fee, resp.Fee, "SubmitOrder should return the fees of the order it read")
			assert.Equal(t, tc.price, resp.Price, "SubmitOrder should return the price of the order it read")
			assert.Equal(t, tc.trigger, resp.TriggerPrice, "SubmitOrder should return the trigger price of the order it read")
		})
	}
}

func TestPlacedOrder(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := e.placedOrder(ctx, "b6f0a1c2-3d4e-4f50-8a6b-7c8d9e0f1a2b", time.Hour)
	assert.ErrorIs(t, err, context.Canceled, "placedOrder should stop waiting when its context ends")
}

func TestCreateOrderRequest(t *testing.T) {
	t.Parallel()
	req, err := createOrderRequest(&order.Submit{Side: order.Buy, Type: order.Market, QuoteAmount: 10, RFQDisabled: true, MarginType: margin.Isolated, Leverage: 2}, "BTC-USD")
	require.NoError(t, err, "createOrderRequest must not error")
	_, err = uuid.Parse(req.ClientOrderID)
	assert.NoError(t, err, "createOrderRequest should generate a client order ID when the order has none")
	assert.Equal(t, &CreateOrderRequest{
		ClientOrderID:      req.ClientOrderID,
		ProductID:          "BTC-USD",
		Side:               OrderSideBuy,
		OrderConfiguration: &MarketMarketIOCRequest{QuoteSize: 10},
		Leverage:           2,
		MarginType:         OrderMarginTypeIsolated,
		SORPreference:      SORPreferenceDisabled,
	}, req, "createOrderRequest should convert the order")
	req, err = createOrderRequest(&order.Submit{Side: order.Sell, Type: order.Limit, Amount: 1, Price: 2, ClientOrderID: "mine"}, "BTC-USD")
	require.NoError(t, err, "createOrderRequest must not error for a sell")
	assert.Equal(t, "mine", req.ClientOrderID, "createOrderRequest should keep the order's client order ID")
	assert.Equal(t, OrderSideSell, req.Side, "createOrderRequest should sell")
	_, err = createOrderRequest(&order.Submit{Side: order.UnknownSide, Type: order.Market}, "BTC-USD")
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "createOrderRequest should refuse an unknown side")
	_, err = createOrderRequest(&order.Submit{Side: order.Buy, Type: order.TrailingStop}, "BTC-USD")
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "createOrderRequest should refuse an unsupported order type")
	_, err = createOrderRequest(&order.Submit{Side: order.Buy, Type: order.Market, MarginType: margin.SpotIsolated}, "BTC-USD")
	assert.ErrorIs(t, err, margin.ErrMarginTypeUnsupported, "createOrderRequest should refuse an unsupported margin type")
	_, err = createOrderRequest(&order.Submit{Side: order.Sell, Type: order.Limit, Amount: 1, Price: 2, ReduceOnly: true}, "BIP-20DEC30-CDE")
	assert.ErrorIs(t, err, errReduceOnlyNotSupported, "createOrderRequest should refuse a reduce only order, which Advanced Trade orders cannot carry")
}

func TestSubmitOrderQuoteSizedStopAndBracket(t *testing.T) {
	t.Parallel()
	var creates atomic.Int32
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		creates.Add(1)
		_, err := w.Write([]byte(`{"success":true,"success_response":{"order_id":"o","product_id":"BTC-USD","side":"BUY","client_order_id":"c"}}`))
		assert.NoError(t, err, "writing the response should not error")
	}))
	for _, s := range []order.Submit{
		{Type: order.StopLimit, Side: order.Buy, QuoteAmount: 100, Price: 90000, TriggerPrice: 89000, StopDirection: order.StopUp},
		{Type: order.Bracket, Side: order.Sell, QuoteAmount: 100, Price: 90000, TriggerPrice: 80000},
	} {
		s.Exchange, s.Pair, s.AssetType = ex.Name, currency.NewBTCUSD(), asset.Spot
		_, err := ex.SubmitOrder(t.Context(), &s)
		assert.ErrorIsf(t, err, order.ErrAmountMustBeSet, "SubmitOrder should refuse a %s order sized in the quote currency", s.Type)
	}
	assert.Zero(t, creates.Load(), "SubmitOrder should not send an order without the size its configuration takes")
}

func TestValidateAdvancedTradeTimeInForce(t *testing.T) {
	t.Parallel()
	end := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	for _, s := range []order.Submit{
		{Type: order.Market},
		{Type: order.Market, TimeInForce: order.ImmediateOrCancel},
		{Type: order.Market, TimeInForce: order.FillOrKill},
		{Type: order.Limit},
		{Type: order.Limit, TimeInForce: order.GoodTillCancel | order.PostOnly},
		{Type: order.Limit, TimeInForce: order.PostOnly, EndTime: end},
		{Type: order.Limit, TimeInForce: order.GoodTillTime, EndTime: end},
		{Type: order.Limit, TimeInForce: order.GoodTillDay, EndTime: end},
		{Type: order.Limit, TimeInForce: order.ImmediateOrCancel},
		{Type: order.Limit, TimeInForce: order.FillOrKill},
		{Type: order.StopLimit, TimeInForce: order.GoodTillCancel},
		{Type: order.StopLimit, EndTime: end},
		{Type: order.Bracket, TimeInForce: order.GoodTillTime, EndTime: end},
	} {
		assert.NoErrorf(t, validateAdvancedTradeTimeInForce(&s), "validateAdvancedTradeTimeInForce should accept a %s %s order", s.TimeInForce, s.Type)
	}
	for _, tc := range []struct {
		s   order.Submit
		err error
	}{
		{order.Submit{Type: order.Limit, TimeInForce: order.GoodTillDay}, common.ErrDateUnset},
		{order.Submit{Type: order.Limit, TimeInForce: order.GoodTillTime}, common.ErrDateUnset},
		{order.Submit{Type: order.Market, TimeInForce: order.GoodTillCancel}, order.ErrUnsupportedTimeInForce},
		{order.Submit{Type: order.Market, TimeInForce: order.PostOnly}, order.ErrUnsupportedTimeInForce},
		{order.Submit{Type: order.Market, EndTime: end}, order.ErrUnsupportedTimeInForce},
		{order.Submit{Type: order.Limit, TimeInForce: order.ImmediateOrCancel | order.FillOrKill}, order.ErrUnsupportedTimeInForce},
		{order.Submit{Type: order.Limit, TimeInForce: order.FillOrKill | order.PostOnly}, order.ErrUnsupportedTimeInForce},
		{order.Submit{Type: order.Limit, TimeInForce: order.ImmediateOrCancel, EndTime: end}, order.ErrUnsupportedTimeInForce},
		{order.Submit{Type: order.Limit, TimeInForce: order.GoodTillCancel, EndTime: end}, order.ErrUnsupportedTimeInForce},
		{order.Submit{Type: order.Limit, TimeInForce: order.GoodTillCrossing}, order.ErrUnsupportedTimeInForce},
		{order.Submit{Type: order.Limit, TimeInForce: order.StopOrReduce}, order.ErrUnsupportedTimeInForce},
		{order.Submit{Type: order.StopLimit, TimeInForce: order.ImmediateOrCancel}, order.ErrUnsupportedTimeInForce},
		{order.Submit{Type: order.Bracket, TimeInForce: order.PostOnly}, order.ErrUnsupportedTimeInForce},
	} {
		assert.ErrorIsf(t, validateAdvancedTradeTimeInForce(&tc.s), tc.err, "validateAdvancedTradeTimeInForce should refuse a %s %s order", tc.s.TimeInForce, tc.s.Type)
	}
	_, err := orderConfigurationRequest(&order.Submit{Type: order.StopLimit, TimeInForce: order.ImmediateOrCancel, Amount: 1, Price: 2, TriggerPrice: 3})
	assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "orderConfigurationRequest should refuse a time in force rather than configure another")
}

func TestOrderConfigurationRequest(t *testing.T) {
	t.Parallel()
	end := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		s   order.Submit
		exp OrderConfigurationRequest
	}{
		{order.Submit{Type: order.Market, Amount: 1}, &MarketMarketIOCRequest{BaseSize: 1}},
		{order.Submit{Type: order.Market, Amount: 1, TimeInForce: order.FillOrKill}, &MarketMarketFOKRequest{BaseSize: 1}},
		{order.Submit{Type: order.Limit, Amount: 1, Price: 2}, &LimitLimitGTCRequest{BaseSize: 1, LimitPrice: 2}},
		{order.Submit{Type: order.Limit, Amount: 1, Price: 2, TimeInForce: order.GoodTillCancel | order.PostOnly}, &LimitLimitGTCRequest{BaseSize: 1, LimitPrice: 2, PostOnly: true}},
		{order.Submit{Type: order.Limit, Amount: 1, Price: 2, TimeInForce: order.FillOrKill}, &LimitLimitFOKRequest{BaseSize: 1, LimitPrice: 2}},
		{order.Submit{Type: order.Limit, QuoteAmount: 5, Price: 2, TimeInForce: order.ImmediateOrCancel}, &SORLimitIOCRequest{QuoteSize: 5, LimitPrice: 2}},
		{order.Submit{Type: order.Limit, Amount: 1, Price: 2, EndTime: end, TimeInForce: order.PostOnly}, &LimitLimitGTDRequest{BaseSize: 1, LimitPrice: 2, EndTime: end, PostOnly: true}},
		{order.Submit{Type: order.Limit, Amount: 1, Price: 2, EndTime: end, TimeInForce: order.GoodTillDay}, &LimitLimitGTDRequest{BaseSize: 1, LimitPrice: 2, EndTime: end}},
		{order.Submit{Type: order.StopLimit, Amount: 1, Price: 2, TriggerPrice: 3, StopDirection: order.StopUp}, &StopLimitStopLimitGTCRequest{BaseSize: 1, LimitPrice: 2, StopPrice: 3, StopDirection: StopDirectionStopUp}},
		{order.Submit{Type: order.StopLimit, Amount: 1, Price: 2, TriggerPrice: 3, EndTime: end}, &StopLimitStopLimitGTDRequest{BaseSize: 1, LimitPrice: 2, StopPrice: 3, EndTime: end, StopDirection: StopDirectionStopDown}},
		{order.Submit{Type: order.Bracket, Amount: 1, Price: 2, TriggerPrice: 3}, &TriggerBracketGTCRequest{BaseSize: 1, LimitPrice: 2, StopTriggerPrice: 3}},
		{order.Submit{Type: order.Bracket, Amount: 1, Price: 2, TriggerPrice: 3, TimeInForce: order.GoodTillTime, EndTime: end}, &TriggerBracketGTDRequest{BaseSize: 1, LimitPrice: 2, StopTriggerPrice: 3, EndTime: end}},
	} {
		got, err := orderConfigurationRequest(&tc.s)
		require.NoErrorf(t, err, "orderConfigurationRequest must not error for %s %s", tc.s.Type, tc.s.TimeInForce)
		assert.Equalf(t, tc.exp, got, "orderConfigurationRequest should convert a %s %s order", tc.s.Type, tc.s.TimeInForce)
	}
}

func TestOrderMarginType(t *testing.T) {
	t.Parallel()
	for marginType, exp := range map[margin.Type]string{margin.Unset: "", margin.Multi: OrderMarginTypeCross, margin.Isolated: OrderMarginTypeIsolated} {
		got, err := orderMarginType(marginType)
		require.NoErrorf(t, err, "orderMarginType must not error for %s", marginType)
		assert.Equalf(t, exp, got, "orderMarginType should convert %s", marginType)
	}
	_, err := orderMarginType(margin.NoMargin)
	assert.ErrorIs(t, err, margin.ErrMarginTypeUnsupported, "orderMarginType should refuse an unsupported margin type")
	for marginType, exp := range map[string]margin.Type{OrderMarginTypeCross: margin.Multi, OrderMarginTypeIsolated: margin.Isolated, OrderMarginTypeUnknown: margin.Unset} {
		assert.Equalf(t, exp, marginTypeFromOrder(marginType), "marginTypeFromOrder should convert %s", marginType)
	}
}

func TestDerivativesOrderRequest(t *testing.T) {
	t.Parallel()
	req, err := derivativesOrderRequest(&order.Submit{
		Side:             order.Sell,
		Type:             order.TrailingStop,
		Amount:           1,
		TriggerPrice:     100,
		TriggerPriceType: order.MarkPrice,
		TrackingMode:     order.Distance,
		TrackingValue:    5,
		ReduceOnly:       true,
		ClientOrderID:    "label",
	}, "BTC_USDC-PERPETUAL")
	require.NoError(t, err, "derivativesOrderRequest must not error")
	assert.Equal(t, &DerivativesOrderRequest{
		InstrumentName: "BTC_USDC-PERPETUAL",
		Amount:         1,
		Type:           DerivativesOrderTypeTrailingStop,
		Label:          "label",
		ReduceOnly:     true,
		Trigger:        DerivativesTriggerMarkPrice,
		TriggerPrice:   100,
		TriggerOffset:  5,
	}, req, "derivativesOrderRequest should convert a trailing stop")
	req, err = derivativesOrderRequest(&order.Submit{Side: order.Buy, Type: order.Limit, Amount: 1, Price: 2, TimeInForce: order.GoodTillCancel | order.PostOnly, MarginType: margin.Isolated, Leverage: 1}, "SOL_USDC-PERPETUAL")
	require.NoError(t, err, "derivativesOrderRequest must not error for a post only limit order")
	assert.Equal(t, &DerivativesOrderRequest{
		InstrumentName: "SOL_USDC-PERPETUAL",
		Amount:         1,
		Type:           DerivativesOrderTypeLimit,
		Price:          2,
		TimeInForce:    DerivativesTimeInForceGoodTilCancelled,
		PostOnly:       true,
		RejectPostOnly: true,
		Isolated:       true,
	}, req, "derivativesOrderRequest should reject a post only order that would take liquidity, and accept no extra leverage")
	for _, tc := range []struct {
		s   order.Submit
		err error
	}{
		{order.Submit{Side: order.Buy, Type: order.Limit, Leverage: 2}, order.ErrSubmitLeverageNotSupported},
		{order.Submit{Side: order.Buy, Type: order.Limit, Leverage: 0.5}, order.ErrSubmitLeverageNotSupported},
		{order.Submit{Side: order.UnknownSide, Type: order.Limit}, order.ErrSideIsInvalid},
		{order.Submit{Side: order.Buy, Type: order.Bracket}, order.ErrUnsupportedOrderType},
		{order.Submit{Side: order.Buy, Type: order.Limit, TimeInForce: order.GoodTillTime}, order.ErrUnsupportedTimeInForce},
		{order.Submit{Side: order.Buy, Type: order.Limit, MarginType: margin.NoMargin}, margin.ErrMarginTypeUnsupported},
		{order.Submit{Side: order.Buy, Type: order.StopMarket, TriggerPriceType: order.UnknownPriceType}, order.ErrUnknownPriceType},
		{order.Submit{Side: order.Buy, Type: order.TrailingStop, TrackingMode: order.Percentage, TrackingValue: 1}, errTrailingStopDistance},
		{order.Submit{Side: order.Buy, Type: order.Limit, EndTime: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)}, order.ErrUnsupportedTimeInForce},
	} {
		_, err := derivativesOrderRequest(&tc.s, "BTC_USDC-PERPETUAL")
		assert.ErrorIsf(t, err, tc.err, "derivativesOrderRequest should refuse %+v", tc.s)
	}
}

func TestDerivativesOrderParams(t *testing.T) {
	t.Parallel()
	for orderType, exp := range map[order.Type]string{
		order.Limit: DerivativesOrderTypeLimit, order.Market: DerivativesOrderTypeMarket, order.StopLimit: DerivativesOrderTypeStopLimit,
		order.StopMarket: DerivativesOrderTypeStopMarket, order.TakeProfit: DerivativesOrderTypeTakeLimit,
		order.TakeProfitMarket: DerivativesOrderTypeTakeMarket, order.TrailingStop: DerivativesOrderTypeTrailingStop,
	} {
		got, err := derivativesOrderTypeParam(orderType)
		require.NoErrorf(t, err, "derivativesOrderTypeParam must not error for %s", orderType)
		assert.Equalf(t, exp, got, "derivativesOrderTypeParam should convert %s", orderType)
	}
	for tif, exp := range map[order.TimeInForce]string{
		order.UnknownTIF: "", order.PostOnly: "", order.GoodTillCancel: DerivativesTimeInForceGoodTilCancelled, order.GoodTillDay: DerivativesTimeInForceGoodTilDay,
		order.FillOrKill: DerivativesTimeInForceFillOrKill, order.ImmediateOrCancel: DerivativesTimeInForceImmediateOrCancel,
	} {
		got, err := derivativesTimeInForceParam(tif)
		require.NoErrorf(t, err, "derivativesTimeInForceParam must not error for %s", tif)
		assert.Equalf(t, exp, got, "derivativesTimeInForceParam should convert %s", tif)
	}
	for priceType, exp := range map[order.PriceType]string{order.LastPrice: DerivativesTriggerLastPrice, order.IndexPrice: DerivativesTriggerIndexPrice, order.MarkPrice: DerivativesTriggerMarkPrice} {
		got, err := derivativesTriggerParam(priceType)
		require.NoErrorf(t, err, "derivativesTriggerParam must not error for %s", priceType)
		assert.Equalf(t, exp, got, "derivativesTriggerParam should convert %s", priceType)
	}
	for marginType, exp := range map[margin.Type]bool{margin.Unset: false, margin.Multi: false, margin.Isolated: true} {
		got, err := derivativesIsolated(marginType)
		require.NoErrorf(t, err, "derivativesIsolated must not error for %s", marginType)
		assert.Equalf(t, exp, got, "derivativesIsolated should convert %s", marginType)
	}
	type cancelType struct {
		category string
		exact    bool
	}
	for orderType, exp := range map[order.Type]cancelType{
		order.AnyType:          {DerivativesCancelOrderTypeAll, true},
		order.UnknownType:      {DerivativesCancelOrderTypeAll, true},
		order.Limit:            {DerivativesCancelOrderTypeLimit, true},
		order.Market:           {DerivativesCancelOrderTypeLimit, false},
		order.Stop:             {DerivativesCancelOrderTypeStop, true},
		order.StopLimit:        {DerivativesCancelOrderTypeStop, false},
		order.StopMarket:       {DerivativesCancelOrderTypeStop, false},
		order.TakeProfit:       {DerivativesCancelOrderTypeTake, false},
		order.TakeProfitMarket: {DerivativesCancelOrderTypeTake, false},
		order.TrailingStop:     {DerivativesCancelOrderTypeTrailingStop, true},
	} {
		category, exact, err := derivativesCancelType(orderType)
		require.NoErrorf(t, err, "derivativesCancelType must not error for %s", orderType)
		assert.Equalf(t, exp, cancelType{category, exact}, "derivativesCancelType should convert %s", orderType)
	}
	_, _, err := derivativesCancelType(order.Bracket)
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "derivativesCancelType should refuse an order type the gateway does not place")
	for _, tc := range []struct {
		cancel, orderType order.Type
		exp               bool
	}{
		{order.AnyType, order.Limit, true},
		{order.UnknownType, order.StopMarket, true},
		{order.Stop, order.StopLimit, true},
		{order.Stop, order.StopMarket, true},
		{order.Stop, order.TakeProfit, false},
		{order.StopLimit, order.StopLimit, true},
		{order.StopLimit, order.StopMarket, false},
		{order.TakeProfit, order.TakeProfitMarket, false},
	} {
		assert.Equalf(t, tc.exp, derivativesCancelTypeMatches(tc.cancel, tc.orderType), "derivativesCancelTypeMatches should match %s orders for %s", tc.orderType, tc.cancel)
	}
}

func TestDerivativesTradeHistory(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.Name = t.Name()
	at := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	got, err := ex.derivativesTradeHistory(&DerivativesUserTradeResponse{TradeID: "USDC-1", Direction: DerivativesDirectionSell, OrderType: DerivativesOrderTypeLimit, Price: 100, Amount: 2, Fee: 0.01, FeeCurrency: currency.USDC, Timestamp: types.Time(at), Liquidity: derivativesLiquidityMaker})
	require.NoError(t, err, "derivativesTradeHistory must not error")
	assert.Equal(t, order.TradeHistory{Price: 100, Amount: 2, Fee: 0.01, FeeAsset: "USDC", Exchange: ex.Name, TID: "USDC-1", Type: order.Limit, Side: order.Sell, Timestamp: at, IsMaker: true}, got, "derivativesTradeHistory should convert a trade")
	_, err = ex.derivativesTradeHistory(&DerivativesUserTradeResponse{Direction: "sideways"})
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "derivativesTradeHistory should error for an unknown direction")
	_, err = ex.derivativesTradeHistory(&DerivativesUserTradeResponse{Direction: DerivativesDirectionBuy, OrderType: "teleport"})
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "derivativesTradeHistory should error for an unknown order type")
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.ModifyOrder(t.Context(), &order.Modify{})
	require.ErrorIs(t, err, order.ErrPairIsEmpty, "ModifyOrder must validate the modification")
	_, err = ex.ModifyOrder(t.Context(), &order.Modify{Pair: currency.NewBTCUSD(), AssetType: asset.Spot})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "ModifyOrder must refuse an order without an ID or label")
	_, err = ex.ModifyOrder(t.Context(), &order.Modify{Pair: currency.NewBTCUSD(), AssetType: asset.Spot, ClientOrderID: "client"})
	require.ErrorIs(t, err, errOrderIDRequired, "ModifyOrder must refuse an Advanced Trade order without an ID")
	_, err = ex.ModifyOrder(t.Context(), &order.Modify{Pair: currency.NewBTCUSD(), AssetType: asset.Options, OrderID: "1"})
	require.ErrorIs(t, err, asset.ErrNotSupported, "ModifyOrder must refuse an asset Coinbase does not trade")
	if !mockTests {
		sharedTestCredentials(t, ex)
		if !canManipulateRealOrders {
			t.Skip("canManipulateRealOrders is not set")
		}
	}
	resp, err := ex.ModifyOrder(t.Context(), &order.Modify{OrderID: testSandboxOrderID, Pair: currency.NewBTCUSD(), AssetType: asset.Spot, Price: 68000.44, Amount: 1})
	require.NoError(t, err, "ModifyOrder must not error for a spot order")
	assert.Equal(t, 68000.44, resp.Price, "ModifyOrder should return the spot order's price")
	_, err = ex.ModifyOrder(t.Context(), &order.Modify{OrderID: testUnknownOrderID, Pair: currency.NewBTCUSD(), AssetType: asset.Spot, Price: 68000.44, Amount: 1})
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "ModifyOrder should report an order Coinbase does not know")

	resp, err = ex.ModifyOrder(t.Context(), &order.Modify{OrderID: testWrapperDerivativesOrderID, Pair: testUSDCPerpetual, AssetType: asset.USDCMarginedFutures, Price: 50100, Amount: 0.001})
	require.NoError(t, err, "ModifyOrder must not error for a gateway order")
	assert.Equal(t, testWrapperDerivativesOrderID, resp.OrderID, "ModifyOrder should return the gateway order's ID")
}

// testWrapperDerivativesOrderID is the gateway order the wrapper's mock order tests act on
const testWrapperDerivativesOrderID = "USDC-1000000001"

func TestModifyDerivativesOrder(t *testing.T) {
	t.Parallel()
	// The order is a post only, reduce only order of an isolated margin subaccount, which the gateway only finds there
	const isolatedOrder = `{"order_id":"USDC-1","order_state":"open","order_type":"limit","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"sell","price":90000,"amount":0.002,"post_only":true,"reject_post_only":true,"reduce_only":true,"label":"gct-edit","isolated":true}`
	const otherOrders = `{"order_id":"USDC-2","order_state":"open","order_type":"limit","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"buy","price":80000,"amount":0.001,"label":"gct-twice"},` +
		`{"order_id":"USDC-3","order_state":"open","order_type":"limit","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"buy","price":80001,"amount":0.001,"label":"gct-twice"},` +
		`{"order_id":"ETH-4","order_state":"open","order_type":"limit","time_in_force":"good_til_cancelled","instrument_name":"ETH_USDC-PERPETUAL","direction":"buy","price":3000,"amount":0.1,"label":"gct-edit"}`
	ex, sent := newDerivativesMethodServer(t, func(req receivedJSONRPC) string {
		switch req.Method {
		case "private/get_order_state":
			if !strings.Contains(string(req.Params), `"isolated":true`) {
				return `{"jsonrpc":"2.0","error":{"code":10004,"message":"order_not_found"}}`
			}
			return `{"jsonrpc":"2.0","result":` + isolatedOrder + `}`
		case "private/get_open_orders":
			// The gateway lists the isolated margin subaccounts' orders only when asked to
			if !strings.Contains(string(req.Params), `"include_isolated":true`) {
				return `{"jsonrpc":"2.0","result":[` + otherOrders + `]}`
			}
			return `{"jsonrpc":"2.0","result":[` + isolatedOrder + `,` + otherOrders + `]}`
		case "private/edit", "private/edit_by_label":
			return `{"jsonrpc":"2.0","result":{"order":` + isolatedOrder + `,"trades":[]}}`
		}
		assert.Failf(t, "ModifyOrder should only read and edit the order", "sent %s", req.Method)
		return `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"}}`
	})
	sentFor := func(method string) []string {
		var params []string
		for _, req := range sent() {
			if req.Method == method {
				params = append(params, string(req.Params))
			}
		}
		return params
	}

	resp, err := ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "USDC-1", Pair: testUSDCPerpetual, AssetType: asset.USDCMarginedFutures, Price: 91000})
	require.NoError(t, err, "ModifyOrder must not error editing an isolated margin order by its ID")
	assert.Equal(t, "USDC-1", resp.OrderID, "ModifyOrder should return the edited order")
	edits := sentFor("private/edit")
	require.Len(t, edits, 1, "ModifyOrder must edit the order once, in the scope holding it")
	assert.JSONEq(t, `{"order_id":"USDC-1","amount":0.002,"price":91000,"post_only":true,"reduce_only":true,"reject_post_only":true,"isolated":true}`, edits[0], "ModifyOrder should keep the order's amount and flags")

	_, err = ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "USDC-1", Pair: testUSDCPerpetual, AssetType: asset.USDCMarginedFutures, Amount: 0.003, TimeInForce: order.GoodTillCancel})
	require.NoError(t, err, "ModifyOrder must not error changing an order's time in force")
	edits = sentFor("private/edit")
	require.Len(t, edits, 1, "ModifyOrder must edit the order once")
	assert.JSONEq(t, `{"order_id":"USDC-1","amount":0.003,"price":90000,"post_only":false,"reduce_only":true,"isolated":true}`, edits[0], "ModifyOrder should clear post only for a time in force without it, and keep the price")

	_, err = ex.ModifyOrder(t.Context(), &order.Modify{ClientOrderID: "gct-edit", Pair: testUSDCPerpetual, AssetType: asset.USDCMarginedFutures, Price: 91000})
	require.NoError(t, err, "ModifyOrder must not error editing an isolated margin order by its label")
	edits = sentFor("private/edit_by_label")
	require.Len(t, edits, 1, "ModifyOrder must edit the order by its label once, in the scope holding it")
	assert.JSONEq(t, `{"label":"gct-edit","instrument_name":"BTC_USDC-PERPETUAL","amount":0.002,"price":91000,"post_only":true,"reduce_only":true,"reject_post_only":true,"isolated":true}`, edits[0], "ModifyOrder should edit the labelled order in its isolated margin subaccount")

	_, err = ex.ModifyOrder(t.Context(), &order.Modify{ClientOrderID: "gct-missing", Pair: testUSDCPerpetual, AssetType: asset.USDCMarginedFutures, Price: 91000})
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "ModifyOrder should report a label no open order of the pair holds")
	_, err = ex.ModifyOrder(t.Context(), &order.Modify{ClientOrderID: "gct-twice", Pair: testUSDCPerpetual, AssetType: asset.USDCMarginedFutures, Price: 91000})
	assert.ErrorIs(t, err, errOrderLabelNotUnique, "ModifyOrder should refuse a label two open orders of the pair hold")
	assert.Empty(t, sentFor("private/edit_by_label"), "ModifyOrder should not edit by a label it cannot resolve to one order")

	// An edit cannot change how long an order rests, only whether it is post only
	_, err = ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "USDC-1", Pair: testUSDCPerpetual, AssetType: asset.USDCMarginedFutures, Price: 91000, TimeInForce: order.ImmediateOrCancel})
	assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "ModifyOrder should refuse a time in force the gateway's edit cannot apply")
	assert.Empty(t, sentFor("private/edit"), "ModifyOrder should not edit an order whose time in force it cannot change")
}

func TestModifyDerivativesOrderPostOnlyMode(t *testing.T) {
	t.Parallel()
	// A post only order placed without reject_post_only, which the gateway reprices rather than rejects
	const repricing = `{"order_id":"USDC-1","order_state":"open","order_type":"limit","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"buy","price":80000,"amount":0.002,"post_only":true,"reject_post_only":false}`
	ex, sent := newDerivativesMethodServer(t, func(req receivedJSONRPC) string {
		switch req.Method {
		case "private/get_order_state":
			return `{"jsonrpc":"2.0","result":` + repricing + `}`
		case "private/edit":
			return `{"jsonrpc":"2.0","result":{"order":` + repricing + `,"trades":[]}}`
		}
		assert.Failf(t, "ModifyOrder should only read and edit the order", "sent %s", req.Method)
		return `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"}}`
	})
	// The order manager passes the order's stored time in force with every modification
	_, err := ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "USDC-1", Pair: testUSDCPerpetual, AssetType: asset.USDCMarginedFutures, Price: 80500, TimeInForce: order.GoodTillCancel | order.PostOnly})
	require.NoError(t, err, "ModifyOrder must not error")
	var edits []string
	for _, req := range sent() {
		if req.Method == "private/edit" {
			edits = append(edits, string(req.Params))
		}
	}
	require.Len(t, edits, 1, "ModifyOrder must edit the order once")
	assert.JSONEq(t, `{"order_id":"USDC-1","amount":0.002,"price":80500,"post_only":true}`, edits[0], "ModifyOrder should leave a repricing post only order repricing when its time in force keeps post only")
}

func TestModifyDerivativesTriggerOrder(t *testing.T) {
	t.Parallel()
	triggerOrders := map[string]string{
		"USDC-7": `{"order_id":"USDC-7","order_state":"untriggered","order_type":"stop_limit","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"sell","price":84900,"trigger":"mark_price","trigger_price":85000,"amount":0.001,"reduce_only":true}`,
		"USDC-8": `{"order_id":"USDC-8","order_state":"untriggered","order_type":"stop_market","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"sell","price":"market_price","trigger":"mark_price","trigger_price":84000,"amount":0.001}`,
		"USDC-9": `{"order_id":"USDC-9","order_state":"untriggered","order_type":"trailing_stop","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"sell","price":"market_price","trigger":"mark_price","trigger_price":83000,"trigger_offset":500,"amount":0.001}`,
	}
	ex, sent := newDerivativesMethodServer(t, func(req receivedJSONRPC) string {
		var params struct {
			OrderID string `json:"order_id"`
		}
		assert.NoError(t, json.Unmarshal(req.Params, &params), "the params should decode")
		switch req.Method {
		case "private/get_order_state":
			return `{"jsonrpc":"2.0","result":` + triggerOrders[params.OrderID] + `}`
		case "private/edit":
			return `{"jsonrpc":"2.0","result":{"order":` + triggerOrders[params.OrderID] + `,"trades":[]}}`
		}
		assert.Failf(t, "ModifyOrder should only read and edit the order", "sent %s", req.Method)
		return `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"}}`
	})
	for _, tc := range []struct {
		orderID string
		m       order.Modify
		exp     string
	}{
		{"USDC-7", order.Modify{Amount: 0.002}, `{"order_id":"USDC-7","amount":0.002,"price":84900,"post_only":false,"reduce_only":true,"trigger_price":85000}`},
		{"USDC-8", order.Modify{TriggerPrice: 83500}, `{"order_id":"USDC-8","amount":0.001,"post_only":false,"trigger_price":83500}`},
		// A trailing stop's trigger follows the market, so its offset is kept and its current trigger price not sent
		{"USDC-9", order.Modify{Amount: 0.002}, `{"order_id":"USDC-9","amount":0.002,"post_only":false,"trigger_offset":500}`},
	} {
		m := tc.m
		m.OrderID, m.Pair, m.AssetType = tc.orderID, testUSDCPerpetual, asset.USDCMarginedFutures
		_, err := ex.ModifyOrder(t.Context(), &m)
		require.NoErrorf(t, err, "ModifyOrder must not error for %s", tc.orderID)
		var edits []string
		for _, req := range sent() {
			if req.Method == "private/edit" {
				edits = append(edits, string(req.Params))
			}
		}
		require.Lenf(t, edits, 1, "ModifyOrder must edit %s once", tc.orderID)
		assert.JSONEqf(t, tc.exp, edits[0], "ModifyOrder should keep %s's trigger and flags", tc.orderID)
	}
}

// newDerivativesMethodServer returns an exchange whose gateway HTTP requests are each answered with the body answer
// returns for it, and a function returning the requests sent since it was last called
func newDerivativesMethodServer(t *testing.T, answer func(req receivedJSONRPC) string) (ex *Exchange, sent func() []receivedJSONRPC) {
	t.Helper()
	var mu sync.Mutex
	var requests []receivedJSONRPC
	ex = newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, ok := readJSONRPC(t, r)
		if !ok {
			return
		}
		mu.Lock()
		requests = append(requests, req)
		mu.Unlock()
		writeJSONRPC(t, w, answer(req))
	}))
	return ex, func() []receivedJSONRPC {
		mu.Lock()
		defer mu.Unlock()
		got := requests
		requests = nil
		return got
	}
}

// testWrapperDerivativesOpenOrders are open orders of the USDC perpetuals in the main account and two isolated margin
// subaccounts, and of an inverse perpetual
const testWrapperDerivativesOpenOrders = `[` +
	`{"order_id":"USDC-1","order_state":"open","order_type":"limit","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"buy","price":80000,"amount":0.001,"creation_timestamp":1792281600000,"last_update_timestamp":1792281600000},` +
	`{"order_id":"USDC-2","order_state":"open","order_type":"limit","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"sell","price":90000,"amount":0.001,"creation_timestamp":1792281600000,"last_update_timestamp":1792281600000},` +
	`{"order_id":"USDC-3","order_state":"untriggered","order_type":"stop_limit","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"buy","price":90100,"trigger":"mark_price","trigger_price":90000,"amount":0.001,"isolated":true,"user_id":123456,"main_uid":1000,"creation_timestamp":1792281600000,"last_update_timestamp":1792281600000},` +
	`{"order_id":"USDC-4","order_state":"untriggered","order_type":"stop_market","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"buy","price":"market_price","trigger":"mark_price","trigger_price":91000,"amount":0.001,"creation_timestamp":1792281600000,"last_update_timestamp":1792281600000},` +
	`{"order_id":"USDC-5","order_state":"open","order_type":"limit","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"buy","price":79000,"amount":0.001,"post_only":true,"reject_post_only":true,"creation_timestamp":1792281600000,"last_update_timestamp":1792281600000},` +
	`{"order_id":"USDC-6","order_state":"open","order_type":"limit","time_in_force":"good_til_cancelled","instrument_name":"SOL_USDC-PERPETUAL","direction":"buy","price":100,"amount":1,"isolated":true,"user_id":123457,"main_uid":1000,"creation_timestamp":1792281600000,"last_update_timestamp":1792281600000},` +
	`{"order_id":"7","order_state":"open","order_type":"limit","time_in_force":"good_til_cancelled","instrument_name":"BTC-PERPETUAL","direction":"buy","price":80000,"amount":10,"creation_timestamp":1792281600000,"last_update_timestamp":1792281600000}` +
	`]`

// orderIDs returns the IDs of orders
func orderIDs(orders []order.Detail) []string {
	ids := make([]string, len(orders))
	for i := range orders {
		ids[i] = orders[i].OrderID
	}
	return ids
}

func TestModifyOrderEquivalentTimeInForce(t *testing.T) {
	t.Parallel()
	limitOrder := func(id, tif, config string) string {
		return `{"order":{"order_id":"` + id + `","product_id":"BTC-USD","product_type":"SPOT","side":"BUY","status":"OPEN","order_type":"LIMIT","time_in_force":"` + tif + `","filled_size":"0","order_configuration":` + config + `}}`
	}
	var edits atomic.Int32
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body string
		switch r.Method + " " + r.URL.Path {
		case http.MethodGet + " /api/v3/brokerage/orders/historical/post-only-gtc":
			body = limitOrder("post-only-gtc", TimeInForceGoodUntilCancelled, `{"limit_limit_gtc":{"base_size":"1","limit_price":"80000","post_only":true}}`)
		case http.MethodGet + " /api/v3/brokerage/orders/historical/post-only-gtd":
			body = limitOrder("post-only-gtd", TimeInForceGoodUntilDateTime, `{"limit_limit_gtd":{"base_size":"1","limit_price":"80000","end_time":"2026-11-01T00:00:00Z","post_only":true}}`)
		case http.MethodGet + " /api/v3/brokerage/orders/historical/gtd":
			body = limitOrder("gtd", TimeInForceGoodUntilDateTime, `{"limit_limit_gtd":{"base_size":"1","limit_price":"80000","end_time":"2026-11-01T00:00:00Z"}}`)
		case http.MethodGet + " /api/v3/brokerage/orders/historical/gtc":
			body = limitOrder("gtc", TimeInForceGoodUntilCancelled, `{"limit_limit_gtc":{"base_size":"1","limit_price":"80000"}}`)
		case http.MethodPost + " /api/v3/brokerage/orders/edit":
			edits.Add(1)
			body = `{"success":true,"errors":[]}`
		default:
			http.NotFound(w, r)
			return
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "writing the response should not error")
	}))
	// The time in force each order was placed with, as the engine copies it from its stored order into the edit
	for _, tc := range []struct {
		orderID string
		tif     order.TimeInForce
	}{
		{"post-only-gtc", order.PostOnly},
		{"post-only-gtd", order.PostOnly},
		{"post-only-gtd", order.GoodTillDay | order.PostOnly},
		{"gtd", order.GoodTillDay},
		{"gtd", order.GoodTillTime},
		{"gtc", order.GoodTillCancel},
	} {
		before := edits.Load()
		_, err := ex.ModifyOrder(t.Context(), &order.Modify{OrderID: tc.orderID, Pair: currency.NewBTCUSD(), AssetType: asset.Spot, Price: 81000, Amount: 1, TimeInForce: tc.tif})
		require.NoErrorf(t, err, "ModifyOrder must not error for %s with the time in force it was placed with, %s", tc.orderID, tc.tif)
		assert.Equalf(t, before+1, edits.Load(), "ModifyOrder should edit %s", tc.orderID)
	}
	// A time in force the edit cannot apply is still refused before anything is sent
	for _, tc := range []struct {
		orderID string
		tif     order.TimeInForce
	}{
		{"gtc", order.ImmediateOrCancel},
		{"gtc", order.FillOrKill},
		{"gtc", order.PostOnly},
		{"gtc", order.GoodTillTime},
		{"gtd", order.GoodTillCancel},
		{"post-only-gtc", order.GoodTillCancel},
	} {
		before := edits.Load()
		_, err := ex.ModifyOrder(t.Context(), &order.Modify{OrderID: tc.orderID, Pair: currency.NewBTCUSD(), AssetType: asset.Spot, Price: 81000, Amount: 1, TimeInForce: tc.tif})
		assert.ErrorIsf(t, err, order.ErrUnsupportedTimeInForce, "ModifyOrder should refuse %s for %s", tc.tif, tc.orderID)
		assert.Equalf(t, before, edits.Load(), "ModifyOrder should not edit %s for %s", tc.orderID, tc.tif)
	}
}

func TestGetActiveOrdersDerivativesFilters(t *testing.T) {
	t.Parallel()
	ex, sent := newDerivativesMethodServer(t, func(receivedJSONRPC) string {
		return `{"jsonrpc":"2.0","result":` + testWrapperDerivativesOpenOrders + `}`
	})
	for _, tc := range []struct {
		name string
		req  order.MultiOrderRequest
		exp  []string
	}{
		// The isolated margin subaccounts' orders of a pair are among its open orders
		{"pair", order.MultiOrderRequest{Pairs: currency.Pairs{testUSDCPerpetual}, Side: order.AnySide}, []string{"USDC-1", "USDC-2", "USDC-3", "USDC-4", "USDC-5"}},
		{"isolated", order.MultiOrderRequest{MarginType: margin.Isolated, Side: order.AnySide}, []string{"USDC-3", "USDC-6"}},
		{"cross margin sell", order.MultiOrderRequest{MarginType: margin.Multi, Side: order.Sell}, []string{"USDC-2"}},
		{"post only", order.MultiOrderRequest{TimeInForce: order.PostOnly, Side: order.AnySide}, []string{"USDC-5"}},
	} {
		req := tc.req
		req.AssetType, req.Type = asset.USDCMarginedFutures, order.AnyType
		orders, err := ex.GetActiveOrders(t.Context(), &req)
		require.NoErrorf(t, err, "GetActiveOrders must not error for the %s orders", tc.name)
		assert.ElementsMatchf(t, tc.exp, orderIDs(orders), "GetActiveOrders should return the %s orders", tc.name)
		requests := sent()
		require.Lenf(t, requests, 1, "GetActiveOrders must list the open orders once for the %s orders", tc.name)
		assert.Equalf(t, "private/get_open_orders", requests[0].Method, "GetActiveOrders should list every open order for the %s orders", tc.name)
		assert.JSONEqf(t, `{"kind":"future","include_isolated":true}`, string(requests[0].Params), "GetActiveOrders should include the isolated margin subaccounts' orders for the %s orders", tc.name)
	}
	orders, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.USDCMarginedFutures, Pairs: currency.Pairs{testUSDCPerpetual}, Side: order.AnySide, Type: order.AnyType})
	require.NoError(t, err, "GetActiveOrders must not error")
	for _, o := range orders {
		exp := margin.Multi
		if o.OrderID == "USDC-3" {
			exp = margin.Isolated
		}
		assert.Equalf(t, exp, o.MarginType, "GetActiveOrders should report order %s's margin type", o.OrderID)
	}
}

func TestDerivativesOrderHistory(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	at := func(offset time.Duration) string { return strconv.FormatInt(start.Add(offset).UnixMilli(), 10) }
	historyOrder := func(id, instrument, state string, created, updated time.Duration, isolated bool) string {
		return `{"order_id":"` + id + `","order_state":"` + state + `","order_type":"limit","time_in_force":"good_til_cancelled","instrument_name":"` + instrument +
			`","direction":"buy","price":80000,"amount":0.001,"creation_timestamp":` + at(created) + `,"last_update_timestamp":` + at(updated) +
			`,"isolated":` + strconv.FormatBool(isolated) + `}`
	}
	page := func(orders ...string) string {
		return `{"jsonrpc":"2.0","result":{"orders":[` + strings.Join(orders, ",") + `],"continuation":"none"}}`
	}
	ex, sent := newDerivativesMethodServer(t, func(req receivedJSONRPC) string {
		historical := strings.Contains(string(req.Params), `"historical":true`)
		switch {
		case req.Method == "private/get_subaccounts":
			return `{"jsonrpc":"2.0","result":[{"id":1000,"type":"main","isolated":null},{"id":123456,"type":"subaccount","isolated":true,"isolated_margin_instrument":"BTC_USDC-PERPETUAL"},{"id":777,"type":"subaccount","isolated":null}]}`
		case req.Method == "private/get_order_history_by_instrument" && !historical:
			// The recent store holds the cancelled order's latest state
			return page(historyOrder("USDC-10", "BTC_USDC-PERPETUAL", "filled", 2*time.Hour, 3*time.Hour, false), historyOrder("USDC-11", "BTC_USDC-PERPETUAL", "cancelled", time.Hour, 90*time.Minute, false))
		case req.Method == "private/get_order_history_by_instrument":
			return page(historyOrder("USDC-11", "BTC_USDC-PERPETUAL", "open", time.Hour, 70*time.Minute, false), historyOrder("USDC-13", "BTC_USDC-PERPETUAL", "filled", 30*time.Minute, 40*time.Minute, false), historyOrder("USDC-12", "BTC_USDC-PERPETUAL", "filled", -time.Hour, -time.Hour, false))
		case req.Method == "private/get_order_history_by_currency" && strings.Contains(string(req.Params), `"subaccount_id":123456`) && !historical:
			return page(historyOrder("USDC-14", "BTC_USDC-PERPETUAL", "filled", 20*time.Minute, 25*time.Minute, true), historyOrder("USDC-15", "SOL_USDC-PERPETUAL", "filled", 10*time.Minute, 15*time.Minute, true))
		case req.Method == "private/get_order_history_by_currency" && strings.Contains(string(req.Params), `"subaccount_id":123456`):
			return page()
		}
		assert.Failf(t, "GetOrderHistory should read the main account's and isolated margin subaccounts' history", "sent %s %s", req.Method, req.Params)
		return `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"}}`
	})
	orders, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.USDCMarginedFutures, Pairs: currency.Pairs{testUSDCPerpetual}, Side: order.AnySide, Type: order.AnyType, StartTime: start, EndTime: start.Add(4 * time.Hour)})
	require.NoError(t, err, "GetOrderHistory must not error")
	assert.Equal(t, []string{"USDC-10", "USDC-11", "USDC-13", "USDC-14"}, orderIDs(orders), "GetOrderHistory should return both stores' and the isolated margin subaccount's orders of the pair, newest first")
	require.Len(t, orders, 4, "GetOrderHistory must return the four orders")
	assert.Equal(t, order.Cancelled, orders[1].Status, "GetOrderHistory should keep an order's latest state")
	assert.Equal(t, margin.Isolated, orders[3].MarginType, "GetOrderHistory should report the isolated margin subaccount's order isolated")
	requests := sent()
	methods := make([]string, len(requests))
	for i, req := range requests {
		methods[i] = req.Method + " " + string(req.Params)
	}
	assert.ElementsMatch(t, []string{
		`private/get_subaccounts {"include_isolated":true}`,
		`private/get_order_history_by_instrument {"instrument_name":"BTC_USDC-PERPETUAL","count":1000,"include_old":true,"include_unfilled":true,"with_continuation":true}`,
		`private/get_order_history_by_instrument {"instrument_name":"BTC_USDC-PERPETUAL","count":1000,"include_old":true,"include_unfilled":true,"with_continuation":true,"historical":true}`,
		`private/get_order_history_by_currency {"kind":"future","count":1000,"include_old":true,"include_unfilled":true,"with_continuation":true,"subaccount_id":123456,"currency":"USDC"}`,
		`private/get_order_history_by_currency {"kind":"future","count":1000,"include_old":true,"include_unfilled":true,"with_continuation":true,"historical":true,"subaccount_id":123456,"currency":"USDC"}`,
	}, methods, "GetOrderHistory should read the history of the main account and the isolated margin subaccount alone, from both stores")
}

func TestDerivativesOrderHistoryPaging(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	historyOrder := func(id string, created, updated time.Time) string {
		return `{"order_id":"` + id + `","order_state":"filled","order_type":"limit","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"buy","price":80000,"amount":0.001,"creation_timestamp":` +
			strconv.FormatInt(created.UnixMilli(), 10) + `,"last_update_timestamp":` + strconv.FormatInt(updated.UnixMilli(), 10) + `}`
	}
	var continuations []string
	var mu sync.Mutex
	ex, _ := newDerivativesMethodServer(t, func(req receivedJSONRPC) string {
		if req.Method == "private/get_subaccounts" {
			return `{"jsonrpc":"2.0","result":[{"id":1000,"type":"main","isolated":null}]}`
		}
		var params struct {
			Continuation string `json:"continuation"`
			Historical   bool   `json:"historical"`
		}
		assert.NoError(t, json.Unmarshal(req.Params, &params), "the history params should decode")
		if params.Historical {
			return `{"jsonrpc":"2.0","result":{"orders":[],"continuation":"none"}}`
		}
		mu.Lock()
		continuations = append(continuations, params.Continuation)
		mu.Unlock()
		switch params.Continuation {
		case "":
			// The page's last order was created before the start but updated after it, so earlier pages can still
			// hold orders created after it
			return `{"jsonrpc":"2.0","result":{"orders":[` + historyOrder("USDC-1", start.Add(time.Hour), start.Add(time.Hour)) + `,` + historyOrder("USDC-2", start.Add(-time.Hour), start.Add(2*time.Hour)) + `],"continuation":"page-2"}}`
		case "page-2":
			return `{"jsonrpc":"2.0","result":{"orders":[` + historyOrder("USDC-3", start.Add(time.Minute), start.Add(time.Minute)) + `,` + historyOrder("USDC-4", start.Add(-2*time.Hour), start.Add(-2*time.Hour)) + `],"continuation":"page-3"}}`
		}
		assert.Failf(t, "the history should stop paging once a page's last order was updated before the start", "continuation %q", params.Continuation)
		return `{"jsonrpc":"2.0","result":{"orders":[],"continuation":"none"}}`
	})
	orders, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.USDCMarginedFutures, Pairs: currency.Pairs{testUSDCPerpetual}, Side: order.AnySide, Type: order.AnyType, StartTime: start, EndTime: start.Add(4 * time.Hour)})
	require.NoError(t, err, "GetOrderHistory must not error")
	assert.Equal(t, []string{"USDC-1", "USDC-3"}, orderIDs(orders), "GetOrderHistory should return the orders created in the window from every page")
	assert.Equal(t, []string{"", "page-2"}, continuations, "GetOrderHistory should page on past an order created before the start but updated after it")

	stalled, _ := newDerivativesMethodServer(t, func(req receivedJSONRPC) string {
		if req.Method == "private/get_subaccounts" {
			return `{"jsonrpc":"2.0","result":[]}`
		}
		return `{"jsonrpc":"2.0","result":{"orders":[` + historyOrder("USDC-1", start.Add(time.Hour), start.Add(time.Hour)) + `],"continuation":"same"}}`
	})
	_, err = stalled.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.USDCMarginedFutures, Pairs: currency.Pairs{testUSDCPerpetual}, Side: order.AnySide, Type: order.AnyType})
	assert.ErrorIs(t, err, errListPagingStalled, "GetOrderHistory should stop a history whose continuation does not advance")

	var cyclingRequests atomic.Int32
	cycling, _ := newDerivativesMethodServer(t, func(req receivedJSONRPC) string {
		if req.Method == "private/get_subaccounts" {
			return `{"jsonrpc":"2.0","result":[]}`
		}
		// A history paging its cycle forever is stopped, failing the test
		if cyclingRequests.Add(1) > 10 {
			assert.Fail(t, "GetOrderHistory should stop paging once a continuation repeats")
			return `{"jsonrpc":"2.0","error":{"code":11000,"message":"paged in a loop"}}`
		}
		next := "page-a"
		if strings.Contains(string(req.Params), `"continuation":"page-a"`) {
			next = "page-b"
		}
		return `{"jsonrpc":"2.0","result":{"orders":[` + historyOrder("USDC-1", start.Add(time.Hour), start.Add(time.Hour)) + `],"continuation":"` + next + `"}}`
	})
	_, err = cycling.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.USDCMarginedFutures, Pairs: currency.Pairs{testUSDCPerpetual}, Side: order.AnySide, Type: order.AnyType})
	assert.ErrorIs(t, err, errListPagingStalled, "GetOrderHistory should stop a history whose continuations cycle")
}

func TestDerivativesOrderTrades(t *testing.T) {
	t.Parallel()
	userTrade := func(id string, at int64) string {
		return `{"trade_id":"` + id + `","trade_seq":` + strconv.FormatInt(at, 10) + `,"timestamp":` + strconv.FormatInt(1792281600000+at, 10) +
			`,"instrument_name":"BTC_USDC-PERPETUAL","order_id":"USDC-1","direction":"buy","order_type":"limit","price":80000,"amount":0.001,"fee":0.01,"fee_currency":"USDC","liquidity":"T"}`
	}
	ex, sent := newDerivativesMethodServer(t, func(req receivedJSONRPC) string {
		// The historical store holds an older trade the recent store no longer does, and one both hold
		if strings.Contains(string(req.Params), `"historical":true`) {
			return `{"jsonrpc":"2.0","result":[` + userTrade("USDC-T0", 1) + `,` + userTrade("USDC-T2", 3) + `]}`
		}
		return `{"jsonrpc":"2.0","result":[` + userTrade("USDC-T1", 2) + `,` + userTrade("USDC-T2", 3) + `]}`
	})
	trades, err := ex.derivativesOrderTrades(t.Context(), false, "USDC-1", true)
	require.NoError(t, err, "derivativesOrderTrades must not error")
	ids := make([]string, len(trades))
	for i := range trades {
		ids[i] = trades[i].TradeID
	}
	assert.Equal(t, []string{"USDC-T0", "USDC-T1", "USDC-T2"}, ids, "derivativesOrderTrades should merge both stores' trades once each, oldest first")
	requests := sent()
	params := make([]string, len(requests))
	for i, req := range requests {
		params[i] = string(req.Params)
	}
	assert.Equal(t, []string{
		`{"order_id":"USDC-1","sorting":"asc","isolated":true}`,
		`{"order_id":"USDC-1","sorting":"asc","historical":true,"isolated":true}`,
	}, params, "derivativesOrderTrades should read the recent and historical stores in the order's scope")
}

func TestCancelAllOrdersDerivativesFilters(t *testing.T) {
	t.Parallel()
	ex, sent := newDerivativesMethodServer(t, func(req receivedJSONRPC) string {
		switch req.Method {
		case "private/get_open_orders":
			return `{"jsonrpc":"2.0","result":` + testWrapperDerivativesOpenOrders + `}`
		case "private/cancel":
			var params struct {
				OrderID string `json:"order_id"`
			}
			assert.NoError(t, json.Unmarshal(req.Params, &params), "the cancel params should decode")
			return `{"jsonrpc":"2.0","result":{"order_id":"` + params.OrderID + `","order_state":"cancelled","order_type":"limit","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"buy","price":80000,"amount":0.001}}`
		case "private/cancel_all_by_instrument":
			return `{"jsonrpc":"2.0","result":[{"currency":"USDC","instrument_name":"BTC_USDC-PERPETUAL","type":"all","result":[{"order_id":"USDC-1","order_state":"cancelled","order_type":"limit","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"buy","price":80000,"amount":0.001}]}]}`
		}
		assert.Failf(t, "CancelAllOrders should list and cancel orders", "sent %s", req.Method)
		return `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"}}`
	})
	for _, tc := range []struct {
		name   string
		cancel order.Cancel
		exp    []string
	}{
		{"sell", order.Cancel{Side: order.Sell}, []string{`private/cancel {"order_id":"USDC-2"}`}},
		{"isolated", order.Cancel{MarginType: margin.Isolated}, []string{`private/cancel {"order_id":"USDC-3","isolated":true}`}},
		{"stop limit", order.Cancel{Type: order.StopLimit}, []string{`private/cancel {"order_id":"USDC-3","isolated":true}`}},
		{"stop buy", order.Cancel{Type: order.Stop, Side: order.Buy}, []string{`private/cancel {"order_id":"USDC-3","isolated":true}`, `private/cancel {"order_id":"USDC-4"}`}},
		{"post only", order.Cancel{TimeInForce: order.PostOnly}, []string{`private/cancel {"order_id":"USDC-5"}`}},
		{"take profit", order.Cancel{Type: order.TakeProfit}, nil},
	} {
		c := tc.cancel
		c.AssetType, c.Pair = asset.USDCMarginedFutures, testUSDCPerpetual
		resp, err := ex.CancelAllOrders(t.Context(), &c)
		require.NoErrorf(t, err, "CancelAllOrders must not error for the %s orders", tc.name)
		var cancels []string
		for _, req := range sent() {
			if req.Method == "private/cancel" {
				cancels = append(cancels, req.Method+" "+string(req.Params))
			}
		}
		assert.Equalf(t, tc.exp, cancels, "CancelAllOrders should cancel the %s orders alone, one at a time in their scopes", tc.name)
		assert.Lenf(t, resp.Status, len(tc.exp), "CancelAllOrders should report each %s order cancelled", tc.name)
	}
	for _, tc := range []struct {
		orderType order.Type
		exp       string
	}{
		{order.AnyType, `{"instrument_name":"BTC_USDC-PERPETUAL","type":"all","detailed":true,"include_isolated":true}`},
		{order.Stop, `{"instrument_name":"BTC_USDC-PERPETUAL","type":"stop","detailed":true,"include_isolated":true}`},
	} {
		resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.USDCMarginedFutures, Pair: testUSDCPerpetual, Type: tc.orderType})
		require.NoErrorf(t, err, "CancelAllOrders must not error for %s orders", tc.orderType)
		requests := sent()
		require.Lenf(t, requests, 1, "CancelAllOrders must cancel the instrument's %s orders together", tc.orderType)
		assert.Equalf(t, "private/cancel_all_by_instrument", requests[0].Method, "CancelAllOrders should cancel the instrument's %s orders together", tc.orderType)
		assert.JSONEqf(t, tc.exp, string(requests[0].Params), "CancelAllOrders should cancel the instrument's %s orders in both margin scopes", tc.orderType)
		assert.Equalf(t, map[string]string{"USDC-1": order.Cancelled.String()}, resp.Status, "CancelAllOrders should report the %s orders cancelled", tc.orderType)
	}
}

func TestCancelAllOrdersDerivativesMarketLimitRemainder(t *testing.T) {
	t.Parallel()
	// A market limit order's unfilled remainder rests as a limit order, which the gateway lists as the market order placed
	const open = `[{"order_id":"USDC-8","order_state":"open","order_type":"limit","original_order_type":"market_limit","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"buy","price":90000,"amount":0.002,"filled_amount":0.001,"creation_timestamp":1792281600000,"last_update_timestamp":1792281600001},` +
		`{"order_id":"USDC-9","order_state":"open","order_type":"limit","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"buy","price":80000,"amount":0.001,"creation_timestamp":1792281600000,"last_update_timestamp":1792281600000},` +
		`{"order_id":"USDC-10","order_state":"untriggered","order_type":"stop_limit","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"sell","price":83900,"trigger":"mark_price","trigger_price":84000,"amount":0.001,"creation_timestamp":1792281600000,"last_update_timestamp":1792281600000},` +
		`{"order_id":"USDC-11","order_state":"untriggered","order_type":"stop_market","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"sell","price":"market_price","trigger":"mark_price","trigger_price":84000,"amount":0.001,"creation_timestamp":1792281600000,"last_update_timestamp":1792281600000},` +
		`{"order_id":"USDC-12","order_state":"untriggered","order_type":"take_limit","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"sell","price":96100,"trigger":"mark_price","trigger_price":96000,"amount":0.001,"creation_timestamp":1792281600000,"last_update_timestamp":1792281600000},` +
		`{"order_id":"USDC-13","order_state":"untriggered","order_type":"take_market","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"sell","price":"market_price","trigger":"mark_price","trigger_price":96000,"amount":0.001,"creation_timestamp":1792281600000,"last_update_timestamp":1792281600000},` +
		`{"order_id":"USDC-14","order_state":"untriggered","order_type":"trailing_stop","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"sell","price":"market_price","trigger":"mark_price","trigger_price":83000,"trigger_offset":500,"amount":0.001,"creation_timestamp":1792281600000,"last_update_timestamp":1792281600000}]`
	ex, sent := newDerivativesMethodServer(t, func(req receivedJSONRPC) string {
		switch req.Method {
		case "private/get_open_orders":
			return `{"jsonrpc":"2.0","result":` + open + `}`
		case "private/cancel":
			return `{"jsonrpc":"2.0","result":{"order_id":"USDC-8","order_state":"cancelled","order_type":"limit","original_order_type":"market_limit","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"buy","price":90000,"amount":0.002,"filled_amount":0.001}}`
		}
		assert.Failf(t, "CancelAllOrders should list the open orders and cancel the market order alone", "sent %s", req.Method)
		return `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"}}`
	})
	orders, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.USDCMarginedFutures, Type: order.Market, Side: order.AnySide})
	require.NoError(t, err, "GetActiveOrders must not error")
	require.Len(t, orders, 1, "GetActiveOrders must list the market limit order's resting remainder as a market order")
	sent()
	resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.USDCMarginedFutures, Pair: testUSDCPerpetual, Type: orders[0].Type})
	require.NoError(t, err, "CancelAllOrders must not error for market orders")
	requests := sent()
	methods := make([]string, len(requests))
	for i, req := range requests {
		methods[i] = req.Method + " " + string(req.Params)
	}
	assert.Equal(t, []string{`private/get_open_orders {"kind":"future","include_isolated":true}`, `private/cancel {"order_id":"USDC-8"}`}, methods, "CancelAllOrders should cancel the resting market order alone, leaving the limit order")
	assert.Equal(t, map[string]string{"USDC-8": order.PartiallyFilledCancelled.String()}, resp.Status, "CancelAllOrders should report the market order's remainder cancelled")
}

func TestCancelAllOrdersDerivativesCancelRefused(t *testing.T) {
	t.Parallel()
	// An order can close between the list and its cancellation, which the gateway then refuses, and the orders after it
	// are still cancelled
	const open = `[{"order_id":"USDC-8","order_state":"open","order_type":"limit","original_order_type":"market_limit","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"buy","price":90000,"amount":0.002,"filled_amount":0.001,"creation_timestamp":1792281600000,"last_update_timestamp":1792281600001},` +
		`{"order_id":"USDC-15","order_state":"open","order_type":"limit","original_order_type":"market","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"sell","price":95000,"amount":0.002,"filled_amount":0.001,"creation_timestamp":1792281600000,"last_update_timestamp":1792281600001}]`
	ex, sent := newDerivativesMethodServer(t, func(req receivedJSONRPC) string {
		switch {
		case req.Method == "private/get_open_orders":
			return `{"jsonrpc":"2.0","result":` + open + `}`
		case req.Method == "private/cancel" && string(req.Params) == `{"order_id":"USDC-8"}`:
			return `{"jsonrpc":"2.0","error":{"code":11044,"message":"not_open_order"}}`
		case req.Method == "private/cancel" && string(req.Params) == `{"order_id":"USDC-15"}`:
			return `{"jsonrpc":"2.0","result":{"order_id":"USDC-15","order_state":"cancelled","order_type":"limit","original_order_type":"market","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"sell","price":95000,"amount":0.002,"filled_amount":0.001}}`
		}
		assert.Failf(t, "CancelAllOrders should list the open orders and cancel the market orders", "sent %s %s", req.Method, req.Params)
		return `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"}}`
	})
	resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.USDCMarginedFutures, Pair: testUSDCPerpetual, Type: order.Market})
	require.NoError(t, err, "CancelAllOrders must not error when the gateway refuses one order's cancellation")
	var cancels []string
	for _, req := range sent() {
		if req.Method == "private/cancel" {
			cancels = append(cancels, string(req.Params))
		}
	}
	assert.Equal(t, []string{`{"order_id":"USDC-8"}`, `{"order_id":"USDC-15"}`}, cancels, "CancelAllOrders should go on to cancel the orders after one the gateway refused")
	assert.Contains(t, resp.Status["USDC-8"], "not_open_order", "CancelAllOrders should report why the gateway refused an order's cancellation")
	assert.Equal(t, order.PartiallyFilledCancelled.String(), resp.Status["USDC-15"], "CancelAllOrders should report the order after it cancelled")
}

func TestCancelAllOrdersDerivativesDocumentedReports(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesMethodServer(t, func(req receivedJSONRPC) string {
		if !assert.Equal(t, "private/cancel_all_by_instrument", req.Method, "CancelAllOrders should cancel the instrument's orders together") {
			return `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"}}`
		}
		// The documented example's shape, a cancelled order in each report
		return `{"jsonrpc":"2.0","result":[{"order":{"order_id":"USDC-1","order_state":"cancelled","order_type":"limit","instrument_name":"BTC_USDC-PERPETUAL","direction":"buy","price":80000,"amount":0.001}},` +
			`{"order":{"order_id":"USDC-2","order_state":"cancelled","order_type":"limit","instrument_name":"BTC_USDC-PERPETUAL","direction":"sell","price":90000,"amount":0.001,"filled_amount":0.0005}}]}`
	})
	resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.USDCMarginedFutures, Pair: testUSDCPerpetual})
	require.NoError(t, err, "CancelAllOrders must not error")
	assert.Equal(t, map[string]string{"USDC-1": order.Cancelled.String(), "USDC-2": order.PartiallyFilledCancelled.String()}, resp.Status, "CancelAllOrders should report each order the documented reports hold")
}

func TestCancelAllOrdersByType(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var cancelled []string
	listed := func(id, productType, product, orderType string) string {
		return `{"order_id":"` + id + `","product_id":"` + product + `","product_type":"` + productType + `","side":"BUY","status":"OPEN","order_type":"` + orderType + `","time_in_force":"GOOD_UNTIL_CANCELLED","filled_size":"0","order_configuration":{"limit_limit_gtc":{"base_size":"1","limit_price":"80000"}}}`
	}
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body string
		switch r.URL.Path {
		case "/api/v3/brokerage/orders/historical/batch":
			body = `{"orders":[],"has_next":false,"cursor":""}`
			if r.URL.Query().Get("order_status") == OrderStatusOpen {
				if r.URL.Query().Get("product_type") == ProductTypeFuture {
					body = `{"orders":[` + listed("limit-f", ProductTypeFuture, "BIP-20DEC30-CDE", OrderTypeLimit) + `,` + listed("roll-open", ProductTypeFuture, "BIP-20DEC30-CDE", OrderTypeRollOpen) + `],"has_next":false,"cursor":""}`
				} else {
					body = `{"orders":[` + listed("limit-1", ProductTypeSpot, "BTC-USD", OrderTypeLimit) + `,` + listed("scaled-1", ProductTypeSpot, "BTC-USD", OrderTypeScaled) + `,` + listed("stop-limit-1", ProductTypeSpot, "BTC-USD", OrderTypeStopLimit) + `],"has_next":false,"cursor":""}`
				}
			}
		case "/api/v3/brokerage/orders/batch_cancel":
			var req CancelOrdersWireRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&req), "the request should decode") {
				return
			}
			mu.Lock()
			cancelled = append(cancelled, req.OrderIDs...)
			mu.Unlock()
			results := make([]string, len(req.OrderIDs))
			for i, id := range req.OrderIDs {
				results[i] = `{"success":true,"failure_reason":"UNKNOWN_CANCEL_FAILURE_REASON","order_id":"` + id + `"}`
			}
			body = `{"results":[` + strings.Join(results, ",") + `]}`
		default:
			http.NotFound(w, r)
			return
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "writing the response should not error")
	}))
	for _, a := range []asset.Item{asset.Spot, asset.Futures} {
		_, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: a, Type: order.Limit})
		require.NoErrorf(t, err, "CancelAllOrders must not error for %s", a)
	}
	mu.Lock()
	defer mu.Unlock()
	assert.ElementsMatch(t, []string{"limit-1", "limit-f"}, cancelled, "CancelAllOrders should cancel only the limit orders")
}

func TestAdvancedTradeProofTokenRequired(t *testing.T) {
	t.Parallel()
	// The EU's strong customer authentication answers a list with a proof token challenge in place of the orders
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body string
		switch r.URL.Path {
		case "/api/v3/brokerage/orders/historical/batch":
			body = `{"orders":[],"has_next":false,"cursor":"","proof_token_required":true}`
		case "/api/v3/brokerage/orders/historical/fills":
			body = `{"fills":[],"cursor":"","proof_token_required":true}`
		case "/api/v3/brokerage/orders/historical/o":
			body = `{"order":{"order_id":"o","product_id":"BTC-USD","product_type":"SPOT","side":"BUY","status":"FILLED","order_type":"MARKET","time_in_force":"IMMEDIATE_OR_CANCEL","filled_size":"1","order_configuration":{"market_market_ioc":{"base_size":"1"}}}}`
		default:
			assert.Failf(t, "the lists should only list orders and fills", "requested %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "writing the response should not error")
	}))
	_, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
	assert.ErrorIs(t, err, errProofTokenRequired, "GetActiveOrders should report a proof token challenge rather than no orders")
	_, err = ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
	assert.ErrorIs(t, err, errProofTokenRequired, "GetOrderHistory should report a proof token challenge rather than no orders")
	_, err = ex.GetOrderInfo(t.Context(), "o", currency.NewBTCUSD(), asset.Spot)
	assert.ErrorIs(t, err, errProofTokenRequired, "GetOrderInfo should report a proof token challenge rather than no fills")
}

func TestAdvancedTradeCursorCycles(t *testing.T) {
	t.Parallel()
	// Each list sends its cursors in a cycle: a, then b, then a again
	next := func(r *http.Request) string {
		if r.URL.Query().Get("cursor") == "a" {
			return "b"
		}
		return "a"
	}
	var requests atomic.Int32
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A list paging its cycle forever is stopped, failing the test
		if requests.Add(1) > 20 {
			assert.Fail(t, "the lists should stop paging once a cursor repeats")
			http.Error(w, `{"error":"INTERNAL","message":"paged in a loop"}`, http.StatusInternalServerError)
			return
		}
		var body string
		switch r.URL.Path {
		case "/api/v3/brokerage/accounts":
			body = `{"accounts":[],"has_next":true,"cursor":"` + next(r) + `"}`
		case "/api/v3/brokerage/orders/historical/batch":
			body = `{"orders":[],"has_next":true,"cursor":"` + next(r) + `"}`
		case "/api/v3/brokerage/orders/historical/fills":
			body = `{"fills":[{"entry_id":"e","trade_id":"t","order_id":"o","trade_time":"2026-10-18T00:00:00Z","trade_type":"FILL","price":"1","size":"1","commission":"0","product_id":"BTC-USD","liquidity_indicator":"TAKER","side":"BUY"}],"cursor":"` + next(r) + `"}`
		case "/api/v3/brokerage/orders/historical/o":
			body = `{"order":{"order_id":"o","product_id":"BTC-USD","product_type":"SPOT","side":"BUY","status":"FILLED","order_type":"MARKET","time_in_force":"IMMEDIATE_OR_CANCEL","filled_size":"1","order_configuration":{"market_market_ioc":{"base_size":"1"}}}}`
		default:
			assert.Failf(t, "the lists should only page their own endpoints", "requested %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "writing the response should not error")
	}))
	_, err := ex.UpdateAccountBalances(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, errListPagingStalled, "UpdateAccountBalances should stop an account list whose cursors cycle")
	_, err = ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
	assert.ErrorIs(t, err, errListPagingStalled, "GetActiveOrders should stop an order list whose cursors cycle")
	_, err = ex.GetOrderInfo(t.Context(), "o", currency.NewBTCUSD(), asset.Spot)
	assert.ErrorIs(t, err, errListPagingStalled, "GetOrderInfo should stop a fill list whose cursors cycle")
}

func TestGetOrderInfoReversedFill(t *testing.T) {
	t.Parallel()
	const orderID = "eeeeeeee-ffff-0000-1111-555555555555"
	fill := func(tradeID, tradeType, size string) string {
		return `{"entry_id":"` + tradeID + tradeType + `","trade_id":"` + tradeID + `","order_id":"` + orderID + `","trade_time":"2026-10-18T00:00:00Z","trade_type":"` + tradeType +
			`","price":"80000","size":"` + size + `","commission":"0","product_id":"BTC-USD","liquidity_indicator":"TAKER","side":"BUY"}`
	}
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body string
		switch r.URL.Path {
		case "/api/v3/brokerage/orders/historical/" + orderID:
			body = `{"order":{"order_id":"` + orderID + `","product_id":"BTC-USD","product_type":"SPOT","side":"BUY","status":"OPEN","order_type":"LIMIT","time_in_force":"GOOD_UNTIL_CANCELLED","filled_size":"0.25","order_configuration":{"limit_limit_gtc":{"base_size":"1","limit_price":"80000"}}}}`
		case "/api/v3/brokerage/orders/historical/fills":
			// The reversal is listed a page after the fill it undoes, and a correction beside its fill
			body = `{"fills":[` + fill("t1", FillTradeTypeFill, "0.5") + `,` + fill("t2", FillTradeTypeFill, "0.25") + `,` + fill("t2", FillTradeTypeCorrection, "0.25") + `],"cursor":"next"}`
			if r.URL.Query().Get("cursor") == "next" {
				body = `{"fills":[` + fill("t1", FillTradeTypeReversal, "0.5") + `],"cursor":""}`
			}
		default:
			assert.Failf(t, "GetOrderInfo should only read the order and its fills", "requested %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "writing the response should not error")
	}))
	d, err := ex.GetOrderInfo(t.Context(), orderID, currency.NewBTCUSD(), asset.Spot)
	require.NoError(t, err, "GetOrderInfo must not error")
	require.Len(t, d.Trades, 1, "GetOrderInfo must leave out a reversed fill and the adjusted fills")
	assert.Equal(t, "t2", d.Trades[0].TID, "GetOrderInfo should keep the fill no reversal undoes")
	assert.Equal(t, 0.25, d.Trades[0].Amount, "GetOrderInfo should report the fill's own size")
}

func TestModifyOrderAdvancedTradeTimeInForce(t *testing.T) {
	t.Parallel()
	var edits atomic.Int32
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body string
		switch r.Method + " " + r.URL.Path {
		case http.MethodGet + " /api/v3/brokerage/orders/historical/open-gtc":
			body = `{"order":{"order_id":"open-gtc","product_id":"BTC-USD","product_type":"SPOT","side":"BUY","status":"OPEN","order_type":"LIMIT","time_in_force":"GOOD_UNTIL_CANCELLED","filled_size":"0","order_configuration":{"limit_limit_gtc":{"base_size":"1","limit_price":"80000","post_only":true}}}}`
		case http.MethodPost + " /api/v3/brokerage/orders/edit":
			edits.Add(1)
			body = `{"success":true,"errors":[]}`
		default:
			assert.Failf(t, "ModifyOrder should only read and edit the order", "%s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "writing the response should not error")
	}))
	_, err := ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "open-gtc", Pair: currency.NewBTCUSD(), AssetType: asset.Spot, Price: 81000, Amount: 1, TimeInForce: order.ImmediateOrCancel})
	assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "ModifyOrder should refuse a time in force Advanced Trade's edit cannot apply")
	assert.Zero(t, edits.Load(), "ModifyOrder should not edit an order whose time in force it cannot change")
	resp, err := ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "open-gtc", Pair: currency.NewBTCUSD(), AssetType: asset.Spot, Price: 81000, Amount: 1, TimeInForce: order.GoodTillCancel | order.PostOnly})
	require.NoError(t, err, "ModifyOrder must not error for the order's own time in force")
	assert.Equal(t, order.GoodTillCancel|order.PostOnly, resp.TimeInForce, "ModifyOrder should report the order's time in force")
	resp, err = ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "open-gtc", Pair: currency.NewBTCUSD(), AssetType: asset.Spot, Price: 81000})
	require.NoError(t, err, "ModifyOrder must not error for a price change")
	assert.Equal(t, 1.0, resp.Amount, "ModifyOrder should keep the order's size")
	assert.Equal(t, order.GoodTillCancel|order.PostOnly, resp.TimeInForce, "ModifyOrder should report the time in force of the order it read")
	_, err = ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "open-gtc", Pair: currency.NewBTCUSD(), AssetType: asset.Spot, Price: 81000, Amount: 2})
	require.NoError(t, err, "ModifyOrder must not error for a size change")
	size, ok := ex.orderSizes.get("open-gtc")
	require.True(t, ok, "ModifyOrder must record the edited size, which the user channel leaves out")
	assert.Equal(t, 2.0, size.base, "ModifyOrder should record the order's new size")
	assert.Equal(t, int32(3), edits.Load(), "ModifyOrder should edit the order for each change it can apply")
}

func TestGetActiveOrdersAdvancedTradeQueued(t *testing.T) {
	t.Parallel()
	advancedTradeOrder := func(id, status string) string {
		return `{"order_id":"` + id + `","product_id":"BTC-USD","product_type":"SPOT","side":"BUY","status":"` + status + `","order_type":"LIMIT","time_in_force":"GOOD_UNTIL_CANCELLED","filled_size":"0","order_configuration":{"limit_limit_gtc":{"base_size":"1","limit_price":"80000"}}}`
	}
	var statuses [][]string
	var mu sync.Mutex
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested := r.URL.Query()["order_status"]
		mu.Lock()
		statuses = append(statuses, requested)
		mu.Unlock()
		body := `{"orders":[` + advancedTradeOrder("open-1", OrderStatusCancelQueued) + `,` + advancedTradeOrder("cancel-queued-2", OrderStatusCancelQueued) + `],"has_next":false,"cursor":""}`
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "writing the response should not error")
	}))
	orders, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
	require.NoError(t, err, "GetActiveOrders must not error")
	assert.Equal(t, []string{"open-1", "cancel-queued-2"}, orderIDs(orders), "GetActiveOrders should return the open and the queued orders once each")
	require.Len(t, orders, 2, "GetActiveOrders must return both orders")
	assert.Equal(t, order.PendingCancel, orders[0].Status, "GetActiveOrders should report an open order whose cancellation is queued pending cancellation")
	assert.Equal(t, order.PendingCancel, orders[1].Status, "GetActiveOrders should report an order whose cancellation is queued pending cancellation")
	assert.Equal(t, [][]string{{OrderStatusOpen, OrderStatusPending, OrderStatusQueued, OrderStatusEditQueued, OrderStatusCancelQueued}}, statuses, "GetActiveOrders should list every live status in one request")
}

func TestGetActiveOrdersAdvancedTradeCombinedListRefused(t *testing.T) {
	t.Parallel()
	advancedTradeOrder := func(id, status string) string {
		return `{"order_id":"` + id + `","product_id":"BTC-USD","product_type":"SPOT","side":"BUY","status":"` + status + `","order_type":"LIMIT","time_in_force":"GOOD_UNTIL_CANCELLED","filled_size":"0","order_configuration":{"limit_limit_gtc":{"base_size":"1","limit_price":"80000"}}}`
	}
	liveStatuses := []string{OrderStatusOpen, OrderStatusPending, OrderStatusQueued, OrderStatusEditQueued, OrderStatusCancelQueued}
	for _, tc := range []struct {
		name     string
		refusal  string
		code     int
		fallback bool
	}{
		// Coinbase's earlier documentation refused OPEN listed with other statuses
		{"invalid argument", `{"error":"INVALID_ARGUMENT","error_details":"order_status OPEN cannot be combined with other statuses","message":"order_status OPEN cannot be combined with other statuses"}`, http.StatusBadRequest, true},
		{"another failure", `{"error":"UNAVAILABLE","message":"try again"}`, http.StatusServiceUnavailable, false},
	} {
		var mu sync.Mutex
		var statuses [][]string
		ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requested := r.URL.Query()["order_status"]
			mu.Lock()
			statuses = append(statuses, requested)
			mu.Unlock()
			var body string
			switch {
			case slices.Equal(requested, liveStatuses):
				http.Error(w, tc.refusal, tc.code)
				return
			case slices.Equal(requested, liveStatuses[1:]):
				body = `{"orders":[` + advancedTradeOrder("pending-1", OrderStatusPending) + `,` + advancedTradeOrder("cancel-queued-2", OrderStatusCancelQueued) + `],"has_next":false,"cursor":""}`
			case slices.Equal(requested, []string{OrderStatusOpen}):
				// The pending order once it was accepted between the lists, and an order open throughout
				body = `{"orders":[` + advancedTradeOrder("pending-1", OrderStatusOpen) + `,` + advancedTradeOrder("open-3", OrderStatusOpen) + `],"has_next":false,"cursor":""}`
			default:
				assert.Failf(t, "GetActiveOrders should only list live statuses", "listed %v with the %s", requested, tc.name)
				http.Error(w, `{"error":"INVALID_ARGUMENT","message":"unexpected statuses"}`, http.StatusBadRequest)
				return
			}
			_, err := w.Write([]byte(body))
			assert.NoError(t, err, "writing the response should not error")
		}))
		orders, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
		mu.Lock()
		requested := slices.Clone(statuses)
		mu.Unlock()
		if !tc.fallback {
			assert.ErrorIsf(t, err, request.ErrAuthRequestFailed, "GetActiveOrders should return the error with %s", tc.name)
			assert.Equalf(t, [][]string{liveStatuses}, requested, "GetActiveOrders should not list again after %s", tc.name)
			continue
		}
		require.NoErrorf(t, err, "GetActiveOrders must not error once the combined list is refused with %s", tc.name)
		assert.Equalf(t, []string{"pending-1", "cancel-queued-2", "open-3"}, orderIDs(orders), "GetActiveOrders should return the queued and the open orders once each with %s", tc.name)
		require.Lenf(t, orders, 3, "GetActiveOrders must return every order with %s", tc.name)
		assert.Equalf(t, order.Open, orders[0].Status, "GetActiveOrders should report the newer state of an order in both lists with %s", tc.name)
		assert.Equalf(t, [][]string{liveStatuses, liveStatuses[1:], {OrderStatusOpen}}, requested, "GetActiveOrders should list the open orders after the other live ones once the combined list is refused with %s", tc.name)
	}
}

func TestGetDepositAddressDefaultNetwork(t *testing.T) {
	t.Parallel()
	address := func(id, network string) string {
		return `{"id":"` + id + `","address":"` + id + `-address","network":"` + network + `","created_at":"2026-10-18T00:00:00Z","updated_at":"2026-10-18T00:00:00Z","resource":"address"}`
	}
	var created []string
	var mu sync.Mutex
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body string
		switch r.Method + " " + r.URL.Path {
		case http.MethodGet + " /currencies/USDC":
			body = `{"id":"USDC","name":"USD Coin","status":"online","default_network":"ethereum","supported_networks":[{"id":"ethereum","status":"online"},{"id":"solana","status":"online"}]}`
		case http.MethodGet + " /currencies/BTC":
			http.Error(w, `{"message":"NotFound"}`, http.StatusNotFound)
			return
		case http.MethodGet + " /v2/accounts/usdc-account/addresses":
			// The newest address is on another network than the default
			body = `{"pagination":{"limit":100,"order":"desc"},"data":[` + address("solana-1", "solana") + `,` + address("ethereum-1", "ethereum") + `]}`
		case http.MethodGet + " /v2/accounts/empty-account/addresses":
			body = `{"pagination":{"limit":100,"order":"desc"},"data":[]}`
		case http.MethodPost + " /v2/accounts/empty-account/addresses":
			req, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "reading the request should not error")
			mu.Lock()
			created = append(created, string(req))
			mu.Unlock()
			body = `{"data":` + address("ethereum-2", "ethereum") + `}`
		default:
			assert.Failf(t, "GetDepositAddress should read the currency and the account's addresses", "%s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "writing the response should not error")
	}))
	got, err := ex.GetDepositAddress(t.Context(), currency.USDC, "usdc-account", "")
	require.NoError(t, err, "GetDepositAddress must not error")
	assert.Equal(t, "ethereum", got.Chain, "GetDepositAddress should return an address on the currency's default network when none is given")
	got, err = ex.GetDepositAddress(t.Context(), currency.USDC, "usdc-account", "solana")
	require.NoError(t, err, "GetDepositAddress must not error for a network given")
	assert.Equal(t, "solana-1-address", got.Address, "GetDepositAddress should return an address on the network given")
	got, err = ex.GetDepositAddress(t.Context(), currency.USDC, "empty-account", "")
	require.NoError(t, err, "GetDepositAddress must not error when it creates an address")
	assert.Equal(t, "ethereum", got.Chain, "GetDepositAddress should create an address on the default network")
	assert.Equal(t, []string{`{"network":"ethereum"}`}, created, "GetDepositAddress should create the address on the currency's default network")
	_, err = ex.GetDepositAddress(t.Context(), currency.BTC, "usdc-account", "")
	assert.Error(t, err, "GetDepositAddress should report a currency whose default network it cannot read")
}

func TestGetOrderInfoClosedDerivativesOrder(t *testing.T) {
	t.Parallel()
	closedOrder := func(id string) string {
		return `{"order_id":"` + id + `","order_state":"filled","order_type":"limit","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"buy","price":80000,"average_price":80000,"amount":0.001,"filled_amount":0.001,"creation_timestamp":1792281600000,"last_update_timestamp":1792281660000}`
	}
	ex, sent := newDerivativesMethodServer(t, func(req receivedJSONRPC) string {
		params := string(req.Params)
		switch {
		case req.Method == "private/get_order_state":
			// The gateway reports an order's state until 30 minutes after it closes
			return `{"jsonrpc":"2.0","error":{"code":10004,"message":"order_not_found"}}`
		case req.Method == "private/get_subaccounts":
			return `{"jsonrpc":"2.0","result":[{"id":1000,"type":"main","isolated":null},{"id":123456,"type":"subaccount","isolated":true}]}`
		case req.Method == "private/get_order_history_by_instrument" && strings.Contains(params, `"continuation":"page-3"`):
			return `{"jsonrpc":"2.0","result":{"orders":[],"continuation":"none"}}`
		case req.Method == "private/get_order_history_by_instrument" && strings.Contains(params, `"historical":true`) && strings.Contains(params, `"continuation":"page-2"`):
			return `{"jsonrpc":"2.0","result":{"orders":[` + closedOrder("USDC-30") + `,` + closedOrder("USDC-31") + `],"continuation":"page-3"}}`
		case req.Method == "private/get_order_history_by_instrument" && strings.Contains(params, `"historical":true`):
			return `{"jsonrpc":"2.0","result":{"orders":[` + closedOrder("USDC-32") + `],"continuation":"page-2"}}`
		case req.Method == "private/get_order_history_by_instrument", req.Method == "private/get_order_history_by_currency":
			return `{"jsonrpc":"2.0","result":{"orders":[],"continuation":"none"}}`
		case req.Method == "private/get_user_trades_by_order":
			return `{"jsonrpc":"2.0","result":[]}`
		}
		assert.Failf(t, "GetOrderInfo should look the order up and read its history", "sent %s %s", req.Method, params)
		return `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"}}`
	})
	detail, err := ex.GetOrderInfo(t.Context(), "USDC-31", testUSDCPerpetual, asset.USDCMarginedFutures)
	require.NoError(t, err, "GetOrderInfo must find an order closed over 30 minutes ago in the history")
	assert.Equal(t, "USDC-31", detail.OrderID, "GetOrderInfo should return the order")
	assert.Equal(t, order.Filled, detail.Status, "GetOrderInfo should return the order's state from the history")
	var historyRequests []string
	for _, req := range sent() {
		if strings.HasPrefix(req.Method, "private/get_order_history") {
			historyRequests = append(historyRequests, req.Method+" "+string(req.Params))
		}
	}
	assert.NotContains(t, historyRequests, `private/get_order_history_by_instrument {"instrument_name":"BTC_USDC-PERPETUAL","count":1000,"include_old":true,"include_unfilled":true,"with_continuation":true,"continuation":"page-3","historical":true}`, "GetOrderInfo should stop paging the history once it finds the order")
	assert.NotContains(t, historyRequests, `private/get_order_history_by_currency {"kind":"future","count":1000,"include_old":true,"include_unfilled":true,"with_continuation":true,"historical":true,"subaccount_id":123456,"currency":"USDC"}`, "GetOrderInfo should stop reading histories once it finds the order")

	_, err = ex.GetOrderInfo(t.Context(), "USDC-99", testUSDCPerpetual, asset.USDCMarginedFutures)
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "GetOrderInfo should report an order neither the order state nor the history holds")
}

func TestSubmitOrderDerivativesAcceptedAmount(t *testing.T) {
	t.Parallel()
	// The gateway reduces a reduce only order to the position it closes
	ex, _ := newDerivativesMethodServer(t, func(receivedJSONRPC) string {
		return `{"jsonrpc":"2.0","result":{"order":{"order_id":"USDC-20","order_state":"open","order_type":"limit","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"sell","price":90000,"amount":0.5,"filled_amount":0,"reduce_only":true,"creation_timestamp":1792281600000,"last_update_timestamp":1792281600000},"trades":[]}}`
	})
	resp, err := ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: testUSDCPerpetual, AssetType: asset.USDCMarginedFutures, Side: order.Sell, Type: order.Limit, Amount: 1, Price: 90000, ReduceOnly: true})
	require.NoError(t, err, "SubmitOrder must not error")
	assert.Equal(t, 0.5, resp.Amount, "SubmitOrder should report the amount the gateway accepted")
	assert.Equal(t, 0.5, resp.RemainingAmount, "SubmitOrder should report the accepted amount remaining")
}

func TestCancelBatchOrdersDuplicateAdvancedTradeOrder(t *testing.T) {
	t.Parallel()
	const id = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	submitted := `{"success":true,"failure_reason":"UNKNOWN_CANCEL_FAILURE_REASON","order_id":"` + id + `"}`
	for _, tc := range []struct {
		name    string
		results []string
	}{
		// Coinbase answers each entry: the first cancels the order, which the repeat then cannot find
		{"first entry submitted", []string{submitted, `{"success":false,"failure_reason":"UNKNOWN_CANCEL_ORDER","order_id":"` + id + `"}`}},
		// The first is refused while the order is being edited, and the repeat cancels it
		{"first entry refused", []string{`{"success":false,"failure_reason":"ORDER_IS_BEING_REPLACED","order_id":"` + id + `"}`, submitted}},
	} {
		ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req CancelOrdersWireRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&req), "decoding the request should not error") {
				return
			}
			assert.Equalf(t, []string{id, id}, req.OrderIDs, "CancelBatchOrders should send each entry with the %s", tc.name)
			_, err := w.Write([]byte(`{"results":[` + strings.Join(tc.results, ",") + `]}`))
			assert.NoError(t, err, "writing the response should not error")
		}))
		c := order.Cancel{Exchange: ex.Name, OrderID: id, Pair: currency.NewBTCUSD(), AssetType: asset.Spot}
		resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{c, c})
		require.NoErrorf(t, err, "CancelBatchOrders must not error with the %s", tc.name)
		assert.Equalf(t, order.PendingCancel.String(), resp.Status[id], "CancelBatchOrders should report an order whose cancel Coinbase submitted pending cancellation, with the %s", tc.name)
	}
}

func TestCancelBatchOrdersValidatesRepeats(t *testing.T) {
	t.Parallel()
	var requests atomic.Int32
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, err := w.Write([]byte(`{"results":[{"success":true,"failure_reason":"UNKNOWN_CANCEL_FAILURE_REASON","order_id":"same"}]}`))
		assert.NoError(t, err, "writing the response should not error")
	}))
	_, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "same", AssetType: asset.Spot}, {OrderID: "same", AssetType: asset.Options}})
	assert.ErrorIs(t, err, asset.ErrNotSupported, "CancelBatchOrders should refuse a repeat of an order for an asset it does not trade")
	assert.Zero(t, requests.Load(), "CancelBatchOrders should send nothing when it refuses an entry")
}

func TestDerivativesOrderLookup(t *testing.T) {
	t.Parallel()
	notFound := &DerivativesError{Code: 10004, Message: derivativesOrderNotFound}
	notOwner := &DerivativesError{Code: 11050, Message: derivativesNotOwnerOfOrder}
	var scopes []bool
	got, err := derivativesOrderLookup(false, func(isolated bool) (string, error) {
		scopes = append(scopes, isolated)
		if !isolated {
			return "", notFound
		}
		return "isolated", nil
	})
	require.NoError(t, err, "derivativesOrderLookup must not error when the other scope holds the order")
	assert.Equal(t, "isolated", got, "derivativesOrderLookup should return the isolated scope's result")
	assert.Equal(t, []bool{false, true}, scopes, "derivativesOrderLookup should look in the main account, then the isolated subaccounts")

	scopes = nil
	_, err = derivativesOrderLookup(true, func(isolated bool) (string, error) {
		scopes = append(scopes, isolated)
		return "", fmt.Errorf("%w: %w", errTestJWT, notOwner)
	})
	require.ErrorIs(t, err, order.ErrOrderNotFound, "derivativesOrderLookup must report an order neither scope holds")
	assert.Equal(t, []bool{true, false}, scopes, "derivativesOrderLookup should look in the scope asked for first")

	scopes = nil
	_, err = derivativesOrderLookup(false, func(isolated bool) (string, error) {
		scopes = append(scopes, isolated)
		return "", errTestJWT
	})
	assert.ErrorIs(t, err, errTestJWT, "derivativesOrderLookup should return any other error")
	assert.Equal(t, []bool{false}, scopes, "derivativesOrderLookup should not look elsewhere after any other error")
	assert.False(t, isDerivativesOrderNotFound(&DerivativesError{Message: "other"}), "isDerivativesOrderNotFound should not report another gateway error")
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	require.ErrorIs(t, ex.CancelOrder(t.Context(), &order.Cancel{}), order.ErrOrderIDNotSet, "CancelOrder must refuse an order without an ID")
	require.ErrorIs(t, ex.CancelOrder(t.Context(), &order.Cancel{OrderID: "1", AssetType: asset.Options}), asset.ErrNotSupported, "CancelOrder must refuse an asset Coinbase does not trade")
	if !mockTests {
		sharedTestCredentials(t, ex)
		if !canManipulateRealOrders {
			t.Skip("canManipulateRealOrders is not set")
		}
	}
	require.NoError(t, ex.CancelOrder(t.Context(), &order.Cancel{OrderID: testSandboxOrderID, AssetType: asset.Spot}), "CancelOrder must not error for a spot order")
	assert.ErrorIs(t, ex.CancelOrder(t.Context(), &order.Cancel{OrderID: testUnknownOrderID, AssetType: asset.Spot}), order.ErrOrderNotFound, "CancelOrder should report an order Coinbase does not know")
	assert.NoError(t, ex.CancelOrder(t.Context(), &order.Cancel{OrderID: testWrapperDerivativesOrderID, AssetType: asset.USDCMarginedFutures}), "CancelOrder should not error for a gateway order")
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.CancelBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelBatchOrders must refuse no orders")
	_, err = ex.CancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "1", AssetType: asset.Options}})
	require.ErrorIs(t, err, asset.ErrNotSupported, "CancelBatchOrders must refuse an asset Coinbase does not trade")
	_, err = ex.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Spot}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelBatchOrders must refuse an order without an ID")
	if !mockTests {
		sharedTestCredentials(t, ex)
		if !canManipulateRealOrders {
			t.Skip("canManipulateRealOrders is not set")
		}
	}
	resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
		{OrderID: testSandboxOrderID, AssetType: asset.Spot},
		{OrderID: testWrapperDerivativesOrderID, AssetType: asset.USDCMarginedFutures},
	})
	require.NoError(t, err, "CancelBatchOrders must not error")
	assert.Equal(t, map[string]string{
		testSandboxOrderID:            order.PendingCancel.String(),
		testWrapperDerivativesOrderID: order.Cancelled.String(),
	}, resp.Status, "CancelBatchOrders should report the Advanced Trade order pending cancellation, and the gateway order cancelled")
}

func TestCancelBatchOrdersDuplicateDerivativesOrder(t *testing.T) {
	t.Parallel()
	const cancelledOrder = `{"jsonrpc":"2.0","result":{"order_id":"USDC-1","order_state":"cancelled","order_type":"limit","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"buy","price":80000,"amount":0.001}}`
	for _, tc := range []struct {
		name    string
		replies []string
	}{
		// The gateway refuses to cancel an order it has cancelled, which a repeat would be
		{"first cancel succeeding", []string{cancelledOrder}},
		{"first cancel failing", []string{`{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error"}}`, cancelledOrder}},
	} {
		var calls atomic.Int32
		ex, sent := newDerivativesMethodServer(t, func(req receivedJSONRPC) string {
			if req.Method != "private/cancel" {
				assert.Failf(t, "CancelBatchOrders should only cancel orders", "sent %s", req.Method)
				return `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"}}`
			}
			if n := int(calls.Add(1)); n <= len(tc.replies) {
				return tc.replies[n-1]
			}
			return `{"jsonrpc":"2.0","error":{"code":11044,"message":"not_open_order"}}`
		})
		resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
			{OrderID: "USDC-1", AssetType: asset.USDCMarginedFutures},
			{OrderID: "USDC-1", AssetType: asset.USDCMarginedFutures},
		})
		require.NoErrorf(t, err, "CancelBatchOrders must not error with the %s", tc.name)
		assert.Equalf(t, map[string]string{"USDC-1": order.Cancelled.String()}, resp.Status, "CancelBatchOrders should report an order named twice cancelled with the %s", tc.name)
		assert.Lenf(t, sent(), len(tc.replies), "CancelBatchOrders should send a repeat only after a failed cancel, with the %s", tc.name)
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.CancelAllOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelAllOrders must refuse a nil request")
	_, err = ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Options})
	require.ErrorIs(t, err, asset.ErrNotSupported, "CancelAllOrders must refuse an asset Coinbase does not trade")
	_, err = ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.USDCMarginedFutures, Pair: testUSDCPerpetual, Type: order.Bracket})
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType, "CancelAllOrders must refuse to cancel an order type the gateway does not place")
	if !mockTests {
		sharedTestCredentials(t, ex)
		if !canManipulateRealOrders {
			t.Skip("canManipulateRealOrders is not set")
		}
	}
	resp, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot})
	require.NoError(t, err, "CancelAllOrders must not error for spot")
	assert.NotEmpty(t, resp.Status, "CancelAllOrders should report the spot orders it cancelled")
	resp, err = ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.USDCMarginedFutures, Pair: testUSDCPerpetual})
	require.NoError(t, err, "CancelAllOrders must not error for a gateway instrument")
	assert.NotEmpty(t, resp.Status, "CancelAllOrders should report the gateway orders it cancelled")
}

func TestCancelBatchOrdersLaterBatchFails(t *testing.T) {
	t.Parallel()
	var batches atomic.Int32
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assert.Equal(t, "/api/v3/brokerage/orders/batch_cancel", r.URL.Path, "CancelBatchOrders should only send cancel requests") {
			http.NotFound(w, r)
			return
		}
		if batches.Add(1) > 1 {
			http.Error(w, `{"error":"UNAVAILABLE","message":"try again"}`, http.StatusServiceUnavailable)
			return
		}
		body, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err, "reading the request should not error") {
			return
		}
		var req CancelOrdersWireRequest
		if !assert.NoError(t, json.Unmarshal(body, &req), "the request should decode") {
			return
		}
		results := make([]string, len(req.OrderIDs))
		for i, id := range req.OrderIDs {
			results[i] = `{"success":true,"failure_reason":"UNKNOWN_CANCEL_FAILURE_REASON","order_id":"` + id + `"}`
		}
		_, err = w.Write([]byte(`{"results":[` + strings.Join(results, ",") + `]}`))
		assert.NoError(t, err, "writing the response should not error")
	}))
	cancels := make([]order.Cancel, cancelOrdersBatchSize+50)
	for i := range cancels {
		cancels[i] = order.Cancel{OrderID: "order-" + strconv.Itoa(i), AssetType: asset.Spot}
	}
	resp, err := ex.CancelBatchOrders(t.Context(), cancels)
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "CancelBatchOrders should report the batch Coinbase did not answer")
	require.NotNil(t, resp, "CancelBatchOrders must report the orders whose cancellations Coinbase submitted before the failure")
	assert.Len(t, resp.Status, cancelOrdersBatchSize, "CancelBatchOrders should report every order of the batch Coinbase answered")
	assert.Equal(t, order.PendingCancel.String(), resp.Status["order-0"], "CancelBatchOrders should report a submitted cancellation pending")
	assert.NotContains(t, resp.Status, "order-"+strconv.Itoa(cancelOrdersBatchSize), "CancelBatchOrders should not report an order of the failed batch")
}

func TestCancelBatchOrdersGatewayAfterAdvancedTradeFailure(t *testing.T) {
	t.Parallel()
	var gatewayCancels atomic.Int32
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v3/brokerage/orders/batch_cancel":
			http.Error(w, `{"error":"UNAVAILABLE","message":"try again"}`, http.StatusServiceUnavailable)
		case "/api/v2/private/cancel":
			gatewayCancels.Add(1)
			writeJSONRPC(t, w, `{"jsonrpc":"2.0","result":{"order_id":"USDC-1","order_state":"cancelled","order_type":"limit","time_in_force":"good_til_cancelled","instrument_name":"BTC_USDC-PERPETUAL","direction":"buy","price":80000,"amount":0.001}}`)
		default:
			assert.Failf(t, "CancelBatchOrders should only cancel orders", "requested %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	resp, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{{OrderID: "order-1", AssetType: asset.Spot}, {OrderID: "USDC-1", AssetType: asset.USDCMarginedFutures}})
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "CancelBatchOrders should report the Advanced Trade batch Coinbase did not answer")
	require.NotNil(t, resp, "CancelBatchOrders must report the gateway order it cancelled")
	assert.Equal(t, map[string]string{"USDC-1": order.Cancelled.String()}, resp.Status, "CancelBatchOrders should cancel the gateway's orders after an Advanced Trade batch fails")
	assert.EqualValues(t, 1, gatewayCancels.Load(), "CancelBatchOrders should send the gateway order's cancel once")
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.GetOrderInfo(t.Context(), "", currency.NewBTCUSD(), asset.Spot)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOrderInfo must refuse an empty order ID")
	_, err = ex.GetOrderInfo(t.Context(), "1", currency.NewBTCUSD(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderInfo must refuse an asset Coinbase does not trade")
	if !mockTests {
		sharedTestCredentials(t, ex)
		t.Skip("live order IDs are the account's own")
	}
	d, err := ex.GetOrderInfo(t.Context(), testSandboxBracketedOrderID, currency.NewBTCUSD(), asset.Spot)
	require.NoError(t, err, "GetOrderInfo must not error for a spot order")
	assert.Equal(t, testSandboxBracketedOrderID, d.OrderID, "GetOrderInfo should return the spot order")
	assert.Equal(t, asset.Spot, d.AssetType, "GetOrderInfo should return the spot order's asset")
	_, err = ex.GetOrderInfo(t.Context(), testSandboxBracketedOrderID, testETHUSD, asset.Spot)
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "GetOrderInfo should not return an order of another pair")

	d, err = ex.GetOrderInfo(t.Context(), testWrapperDerivativesOrderID, testUSDCPerpetual, asset.USDCMarginedFutures)
	require.NoError(t, err, "GetOrderInfo must not error for a gateway order")
	assert.Equal(t, testWrapperDerivativesOrderID, d.OrderID, "GetOrderInfo should return the gateway order")
	assert.NotEmpty(t, d.Trades, "GetOrderInfo should return the gateway order's trades")
}

func TestOrderConfigurationValues(t *testing.T) {
	t.Parallel()
	end := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		c   *OrderConfiguration
		exp orderConfiguration
	}{
		{nil, orderConfiguration{}},
		{&OrderConfiguration{}, orderConfiguration{}},
		{&OrderConfiguration{MarketMarketIOC: &MarketMarketIOC{BaseSize: 1, QuoteSize: 2}}, orderConfiguration{baseSize: 1, quoteSize: 2}},
		{&OrderConfiguration{MarketMarketFOK: &MarketMarketFOK{BaseSize: 1}}, orderConfiguration{baseSize: 1}},
		{&OrderConfiguration{SORLimitIOC: &SORLimitIOC{BaseSize: 1, LimitPrice: 3}}, orderConfiguration{baseSize: 1, limitPrice: 3}},
		{&OrderConfiguration{LimitLimitGTC: &LimitLimitGTC{BaseSize: 1, LimitPrice: 3, PostOnly: true}}, orderConfiguration{baseSize: 1, limitPrice: 3, postOnly: true}},
		{&OrderConfiguration{LimitLimitGTD: &LimitLimitGTD{QuoteSize: 2, LimitPrice: 3, EndTime: end, PostOnly: true}}, orderConfiguration{quoteSize: 2, limitPrice: 3, endTime: end, postOnly: true}},
		{&OrderConfiguration{LimitLimitFOK: &LimitLimitFOK{BaseSize: 1, LimitPrice: 3}}, orderConfiguration{baseSize: 1, limitPrice: 3}},
		{&OrderConfiguration{TWAPLimitGTD: &TWAPLimitGTD{BaseSize: 1, LimitPrice: 3, EndTime: end}}, orderConfiguration{baseSize: 1, limitPrice: 3, endTime: end}},
		{&OrderConfiguration{StopLimitStopLimitGTC: &StopLimitStopLimitGTC{BaseSize: 1, LimitPrice: 3, StopPrice: 4}}, orderConfiguration{baseSize: 1, limitPrice: 3, stopPrice: 4}},
		{&OrderConfiguration{StopLimitStopLimitGTD: &StopLimitStopLimitGTD{BaseSize: 1, LimitPrice: 3, StopPrice: 4, EndTime: end}}, orderConfiguration{baseSize: 1, limitPrice: 3, stopPrice: 4, endTime: end}},
		{&OrderConfiguration{TriggerBracketGTC: &TriggerBracketGTC{BaseSize: 1, LimitPrice: 3, StopTriggerPrice: 4}}, orderConfiguration{baseSize: 1, limitPrice: 3, stopPrice: 4}},
		{&OrderConfiguration{TriggerBracketGTD: &TriggerBracketGTD{BaseSize: 1, LimitPrice: 3, StopTriggerPrice: 4, EndTime: end}}, orderConfiguration{baseSize: 1, limitPrice: 3, stopPrice: 4, endTime: end}},
		{&OrderConfiguration{ScaledLimitGTC: &ScaledLimitGTC{BaseSize: 1, QuoteSize: 2}}, orderConfiguration{baseSize: 1, quoteSize: 2}},
	} {
		assert.Equalf(t, tc.exp, orderConfigurationValues(tc.c), "orderConfigurationValues should read %+v", tc.c)
	}
}

func TestOrderDetail(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	created := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	filled := created.Add(time.Minute)
	o := &Order{
		OrderID:            "order",
		ProductID:          "BTC-USD",
		ProductType:        ProductTypeSpot,
		OrderConfiguration: &OrderConfiguration{LimitLimitGTC: &LimitLimitGTC{BaseSize: 2, LimitPrice: 100, PostOnly: true}},
		Side:               OrderSideBuy,
		ClientOrderID:      "client",
		Status:             OrderStatusFilled,
		TimeInForce:        TimeInForceGoodUntilCancelled,
		CreatedTime:        created,
		FilledSize:         2,
		AverageFilledPrice: 99,
		FilledValue:        198,
		TotalFees:          1.2,
		OrderType:          OrderTypeLimit,
		Settled:            true,
		LastFillTime:       filled,
		LastUpdateTime:     filled,
		Leverage:           1,
		MarginType:         OrderMarginTypeUnknown,
		RetailPortfolioID:  "portfolio",
	}
	d, err := ex.orderDetail(o, ex.orderSizes.stamp())
	require.NoError(t, err, "orderDetail must not error")
	assert.Equal(t, &order.Detail{
		Exchange:             ex.Name,
		OrderID:              "order",
		ClientOrderID:        "client",
		AccountID:            "portfolio",
		Pair:                 currency.NewBTCUSD(),
		AssetType:            asset.Spot,
		Side:                 order.Buy,
		Type:                 order.Limit,
		Status:               order.Filled,
		TimeInForce:          order.GoodTillCancel | order.PostOnly,
		Price:                100,
		Amount:               2,
		ExecutedAmount:       2,
		AverageExecutedPrice: 99,
		Cost:                 198,
		Fee:                  1.2,
		Leverage:             1,
		Date:                 created,
		CloseTime:            filled,
		LastUpdated:          filled,
	}, d, "orderDetail should convert a filled limit order")
	// The user channel reports no order's size, so a live order's is recorded from the list, and a closed one's forgotten
	live := *o
	live.Status, live.Settled, live.FilledSize = OrderStatusOpen, false, 0
	_, err = ex.orderDetail(&live, ex.orderSizes.stamp())
	require.NoError(t, err, "orderDetail must not error for an open order")
	size, ok := ex.orderSizes.get("order")
	require.True(t, ok, "orderDetail must record a live order's size")
	assert.Equal(t, 2.0, size.base, "orderDetail should record a live order's base size")
	// A size recorded since the list was asked for, by an edit, is newer than the list's
	listed := ex.orderSizes.stamp()
	ex.orderSizes.set("order", 3, 0)
	_, err = ex.orderDetail(&live, listed)
	require.NoError(t, err, "orderDetail must not error for an order listed before its edit")
	size, _ = ex.orderSizes.get("order")
	assert.Equal(t, 3.0, size.base, "orderDetail should keep a size recorded after the list was asked for")
	_, err = ex.orderDetail(o, ex.orderSizes.stamp())
	require.NoError(t, err, "orderDetail must not error")
	_, ok = ex.orderSizes.get("order")
	assert.False(t, ok, "orderDetail should forget a closed order's size")
	assert.Equal(t, 2.0, ex.orderSizes.closedSize("order"), "orderDetail should keep the base size the list gave a closed order the record held")
	// Coinbase reports no time an order was cancelled, and an expired order closed at its end time
	cancelled := *o
	cancelled.Status, cancelled.Settled, cancelled.FilledSize = OrderStatusCancelled, false, 0
	d, err = ex.orderDetail(&cancelled, ex.orderSizes.stamp())
	require.NoError(t, err, "orderDetail must not error for a cancelled order")
	assert.Zero(t, d.CloseTime, "orderDetail should leave a cancelled order's close time unknown")
	partlyFilled := *o
	partlyFilled.Status, partlyFilled.FilledSize = OrderStatusCancelled, 1
	d, err = ex.orderDetail(&partlyFilled, ex.orderSizes.stamp())
	require.NoError(t, err, "orderDetail must not error for a partly filled cancelled order")
	assert.Zero(t, d.CloseTime, "orderDetail should not close a cancelled order at its last fill, which came before its cancellation")
	end := created.Add(time.Hour)
	expired := cancelled
	expired.Status = OrderStatusExpired
	expired.OrderConfiguration = &OrderConfiguration{LimitLimitGTD: &LimitLimitGTD{BaseSize: 2, LimitPrice: 100, EndTime: end}}
	d, err = ex.orderDetail(&expired, ex.orderSizes.stamp())
	require.NoError(t, err, "orderDetail must not error for an expired order")
	assert.Equal(t, end, d.CloseTime, "orderDetail should close an expired order at its end time")
	quoteSized := *o
	quoteSized.OrderConfiguration = &OrderConfiguration{MarketMarketIOC: &MarketMarketIOC{QuoteSize: 100}}
	quoteSized.SizeInQuote = true
	d, err = ex.orderDetail(&quoteSized, ex.orderSizes.stamp())
	require.NoError(t, err, "orderDetail must not error for an order sized in the quote currency")
	assert.Zero(t, d.RemainingAmount, "orderDetail should report no remaining base amount for an order sized in the quote currency")
	assert.Equal(t, 2.0, d.Amount, "orderDetail should take the base amount a filled order sized in the quote currency alone executed as its size")
	quoteMarket := quoteSized
	quoteMarket.OrderType, quoteMarket.TimeInForce, quoteMarket.FilledSize = OrderTypeMarket, TimeInForceImmediateOrCancel, 0.001
	d, err = ex.orderDetail(&quoteMarket, ex.orderSizes.stamp())
	require.NoError(t, err, "orderDetail must not error for a market order sized in the quote currency")
	assert.Equal(t, 0.001, d.Amount, "orderDetail should take the base amount a filled market order sized in the quote currency executed as its size")
	quoteLimit := quoteSized
	quoteLimit.OrderConfiguration, quoteLimit.FilledSize, quoteLimit.FilledValue, quoteLimit.AverageFilledPrice = &OrderConfiguration{LimitLimitGTC: &LimitLimitGTC{QuoteSize: 100, LimitPrice: 100000}}, 0.001, 99, 99000
	d, err = ex.orderDetail(&quoteLimit, ex.orderSizes.stamp())
	require.NoError(t, err, "orderDetail must not error for a limit order sized in the quote currency")
	assert.Equal(t, 0.001, d.Amount, "orderDetail should take the base amount a filled limit order sized in the quote currency executed as its size")
	undecoded := *o
	undecoded.OrderConfiguration = &OrderConfiguration{}
	d, err = ex.orderDetail(&undecoded, ex.orderSizes.stamp())
	require.NoError(t, err, "orderDetail must not error for an order whose configuration it does not decode")
	assert.Equal(t, 2.0, d.Amount, "orderDetail should take the base amount a filled order whose configuration it does not decode executed as its size")
	quoteCancelled := quoteSized
	quoteCancelled.Status = OrderStatusCancelled
	d, err = ex.orderDetail(&quoteCancelled, ex.orderSizes.stamp())
	require.NoError(t, err, "orderDetail must not error for a cancelled order sized in the quote currency")
	assert.Zero(t, d.Amount, "orderDetail should report no base size for a cancelled order sized in the quote currency")
	dust := *o
	dust.FilledSize = 1.99999999
	d, err = ex.orderDetail(&dust, ex.orderSizes.stamp())
	require.NoError(t, err, "orderDetail must not error for a filled order leaving dust")
	assert.Equal(t, 2.0, d.Amount, "orderDetail should keep the base size of a filled order leaving dust")
	for _, broken := range []Order{
		{ProductType: ProductTypeEquity},
		{ProductType: ProductTypeSpot, ProductID: "BTCUSD!", Side: OrderSideBuy},
		{ProductType: ProductTypeSpot, ProductID: "BTC-USD", Side: "SIDEWAYS"},
		{ProductType: ProductTypeSpot, ProductID: "BTC-USD", Side: OrderSideBuy, OrderType: "TELEPORT"},
		{ProductType: ProductTypeSpot, ProductID: "BTC-USD", Side: OrderSideBuy, OrderType: OrderTypeLimit, Status: "SLEEPING"},
		{ProductType: ProductTypeSpot, ProductID: "BTC-USD", Side: OrderSideBuy, OrderType: OrderTypeLimit, Status: OrderStatusOpen, TimeInForce: "GOOD_UNTIL_LUNCH"},
	} {
		_, err := ex.orderDetail(&broken, ex.orderSizes.stamp())
		assert.Errorf(t, err, "orderDetail should error for %+v", broken)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Options, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetActiveOrders must refuse an asset Coinbase does not trade")
	if !mockTests {
		sharedTestCredentials(t, ex)
	}
	for _, a := range []asset.Item{asset.Spot, asset.USDCMarginedFutures} {
		orders, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: a, Side: order.AnySide, Type: order.AnyType})
		require.NoErrorf(t, err, "GetActiveOrders must not error for %s", a)
		for i := range orders {
			assert.Equalf(t, a, orders[i].AssetType, "GetActiveOrders should return only %s orders", a)
		}
		if mockTests {
			assert.NotEmptyf(t, orders, "GetActiveOrders should return the recorded %s orders", a)
		}
	}
}

func TestGetOrderInfoEditTimestamp(t *testing.T) {
	t.Parallel()
	const orderID = "eeeeeeee-ffff-0000-1111-444444444444"
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := `{"fills":[],"cursor":""}`
		if r.URL.Path == "/api/v3/brokerage/orders/historical/"+orderID {
			// Created and last filled on 1 October, and edited on 2 October
			body = `{"order":{"order_id":"` + orderID + `","product_id":"BTC-USD","product_type":"SPOT","side":"BUY","status":"OPEN","order_type":"LIMIT","time_in_force":"GOOD_UNTIL_CANCELLED","filled_size":"0","created_time":"2026-10-01T00:00:00Z","last_update_time":"2026-10-01T00:00:00Z","edit_history":[{"price":"100","size":"2","replace_accept_timestamp":"2026-10-02T00:00:00Z"}],"order_configuration":{"limit_limit_gtc":{"base_size":"2","limit_price":"100"}}}}`
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "writing the response should not error")
	}))
	d, err := ex.GetOrderInfo(t.Context(), orderID, currency.NewBTCUSD(), asset.Spot)
	require.NoError(t, err, "GetOrderInfo must not error")
	assert.Equal(t, time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC), d.LastUpdated, "GetOrderInfo should report an order's latest accepted edit as its last update")
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Options, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderHistory must refuse an asset Coinbase does not trade")
	if !mockTests {
		sharedTestCredentials(t, ex)
	}
	for _, tc := range []struct {
		a asset.Item
		p currency.Pairs
	}{{asset.Spot, nil}, {asset.USDCMarginedFutures, currency.Pairs{testUSDCPerpetual}}} {
		orders, err := ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: tc.a, Pairs: tc.p, Side: order.AnySide, Type: order.AnyType})
		require.NoErrorf(t, err, "GetOrderHistory must not error for %s", tc.a)
		for i := range orders {
			assert.Equalf(t, tc.a, orders[i].AssetType, "GetOrderHistory should return only %s orders", tc.a)
		}
		if !mockTests || tc.a != asset.USDCMarginedFutures {
			continue
		}
		// The mock's isolated margin subaccount holds an order of the pair and one of another
		i := slices.IndexFunc(orders, func(d order.Detail) bool { return d.OrderID == "USDC-1000000002" })
		require.NotEqual(t, -1, i, "GetOrderHistory must return the isolated margin subaccount's order of the pair")
		assert.Equal(t, margin.Isolated, orders[i].MarginType, "GetOrderHistory should report the isolated margin subaccount's order isolated")
		assert.False(t, slices.ContainsFunc(orders, func(d order.Detail) bool { return d.OrderID == "USDC-1000000003" }), "GetOrderHistory should leave out the isolated margin subaccount's order of another pair")
	}
}

func TestFilterOrdersGoodTillDay(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	placed := []order.Detail{
		{Exchange: ex.Name, OrderID: "gtd", Pair: currency.NewBTCUSD(), AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Status: order.Open, TimeInForce: order.GoodTillTime},
		{Exchange: ex.Name, OrderID: "gtc", Pair: currency.NewBTCUSD(), AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Status: order.Open, TimeInForce: order.GoodTillCancel},
	}
	got := ex.filterOrders(&order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType, TimeInForce: order.GoodTillDay}, placed)
	require.Len(t, got, 1, "filterOrders must match the order a good till day submission placed")
	assert.Equal(t, "gtd", got[0].OrderID, "filterOrders should match a good till day request to the good till time order Coinbase reports")
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	if !mockTests {
		sharedTestCredentials(t, ex)
	}
	address, err := ex.GetDepositAddress(t.Context(), currency.BTC, "", "")
	require.NoError(t, err, "GetDepositAddress must not error")
	assert.NotEmpty(t, address.Address, "GetDepositAddress should return an address")
	if mockTests {
		created, err := ex.GetDepositAddress(t.Context(), currency.BTC, testAppAccountID, "ethereum")
		require.NoError(t, err, "GetDepositAddress must not error when it creates an address")
		assert.Equal(t, "ethereum", created.Chain, "GetDepositAddress should create an address on the network asked for")
	}
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{})
	require.Error(t, err, "WithdrawCryptocurrencyFunds must validate the request")
	if !mockTests {
		sharedTestCredentials(t, ex)
		if !canManipulateRealOrders {
			t.Skip("canManipulateRealOrders is not set")
		}
	}
	resp, err := ex.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange:         ex.Name,
		Currency:         currency.BTC,
		Amount:           0.1,
		Type:             withdraw.Crypto,
		WalletID:         testAppAccountID,
		IdempotencyToken: "df087dce-92a8-45cf-b112-60aad22c0976",
		Crypto:           withdraw.CryptoRequest{Address: "1AUJ8z5RuHRTqD1eikyfUUetzGmdWLGkpT", Chain: "bitcoin"},
		Travel:           withdraw.TravelRule{IsSelf: true, BeneficiaryWalletType: AppTravelRuleWalletTypeExchange},
	})
	require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error")
	assert.NotEmpty(t, resp.ID, "WithdrawCryptocurrencyFunds should return the send's ID")
}

func TestWithdrawalTravelRule(t *testing.T) {
	t.Parallel()
	assert.Nil(t, withdrawalTravelRule(&withdraw.TravelRule{}), "withdrawalTravelRule should send no details for a withdrawal without any")
	assert.Equal(t, &AppTravelRuleData{IsSelf: AppTravelRuleIsSelfTrue}, withdrawalTravelRule(&withdraw.TravelRule{IsSelf: true}), "withdrawalTravelRule should send a withdrawal to the account itself")
	got := withdrawalTravelRule(&withdraw.TravelRule{
		BeneficiaryWalletType:           AppTravelRuleWalletTypeSelfHosted,
		BeneficiaryName:                 "Jane Citizen",
		BeneficiaryAddress:              withdraw.TravelAddress{Address1: "1 Test Street", City: "Sydney", State: "NSW", Country: "AU", PostalCode: "2000"},
		BeneficiaryFinancialInstitution: "vasp",
		TransferPurpose:                 "rent",
	})
	assert.Equal(t, &AppTravelRuleData{
		BeneficiaryWalletType:           AppTravelRuleWalletTypeSelfHosted,
		IsSelf:                          AppTravelRuleIsSelfFalse,
		BeneficiaryName:                 "Jane Citizen",
		BeneficiaryAddress:              &AppTravelRuleAddress{Address1: "1 Test Street", City: "Sydney", State: "NSW", Country: "AU", PostalCode: "2000"},
		BeneficiaryFinancialInstitution: "vasp",
		TransferPurpose:                 "rent",
	}, got, "withdrawalTravelRule should send a beneficiary's details")
}

func TestWithdrawFiatFunds(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.WithdrawFiatFunds(t.Context(), &withdraw.Request{})
	require.Error(t, err, "WithdrawFiatFunds must validate the request")
	req := &withdraw.Request{
		Exchange: ex.Name,
		Currency: currency.USD,
		Amount:   10,
		Type:     withdraw.Fiat,
		WalletID: testAppAccountID,
		Fiat: withdraw.FiatRequest{Bank: banking.Account{
			Enabled: true, AccountName: "Satoshi Nakamoto", AccountNumber: "12345", BankName: "Unknown bank", BankAddress: "33 Liberty Street",
			BankPostalCity: "New York", BankCountry: "United States", SWIFTCode: "FRNYUS33", SupportedCurrencies: "USD", SupportedExchanges: ex.Name,
		}},
	}
	if !mockTests {
		sharedTestCredentials(t, ex)
		if !canManipulateRealOrders {
			t.Skip("canManipulateRealOrders is not set")
		}
	}
	_, err = ex.WithdrawFiatFunds(t.Context(), req)
	require.ErrorIs(t, err, errPaymentMethodNotFound, "WithdrawFiatFunds must refuse a bank without a payment method")
	if !mockTests {
		return
	}
	req.Fiat.Bank.BankName = "ALLY BANK ******1234"
	resp, err := ex.WithdrawFiatFundsToInternationalBank(t.Context(), req)
	require.NoError(t, err, "WithdrawFiatFundsToInternationalBank must not error")
	assert.NotEmpty(t, resp.ID, "WithdrawFiatFundsToInternationalBank should return the withdrawal's ID")
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.GetFeeByType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFeeByType must refuse a nil fee builder")
	_, err = ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyWithdrawalFee})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetFeeByType must refuse a fee type it does not estimate")
	for _, tc := range []struct {
		f   exchange.FeeBuilder
		exp float64
	}{
		{exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, Pair: currency.NewBTCUSD(), PurchasePrice: 100, Amount: 2}, 2.4},
		{exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, Pair: currency.NewBTCUSD(), PurchasePrice: 100, Amount: 2, IsMaker: true}, 1.2},
		{exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, Pair: currency.NewPair(currency.USDT, currency.USD), PurchasePrice: 1, Amount: 1000}, 0.045},
		{exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, Pair: currency.NewPair(currency.USDT, currency.USD), PurchasePrice: 1, Amount: 1000, IsMaker: true}, 0},
	} {
		fee, err := ex.GetFeeByType(t.Context(), &tc.f)
		require.NoErrorf(t, err, "GetFeeByType must not error for %+v", tc.f)
		assert.InDeltaf(t, tc.exp, fee, 1e-12, "GetFeeByType should estimate %+v", tc.f)
	}
	if !mockTests {
		sharedTestCredentials(t, ex)
	}
	fee, err := ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: currency.NewBTCUSD(), PurchasePrice: 100, Amount: 2})
	require.NoError(t, err, "GetFeeByType must not error at the account's fee tier")
	assert.Positive(t, fee, "GetFeeByType should estimate a fee at the account's fee tier")

	// US derivatives are charged by the contract, and the gateway's perpetuals at their own tier
	for _, p := range []currency.Pair{testUSDCPerpetual, testCoinPerpetual, datedFuture(t)} {
		_, err = ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, Pair: p, PurchasePrice: 100, Amount: 2})
		assert.ErrorIsf(t, err, common.ErrFunctionNotSupported, "GetFeeByType should refuse to estimate %s's fees as spot's", p)
	}
	// Before pairs load, the symbols name the derivatives
	unloaded := new(Exchange)
	unloaded.SetDefaults()
	for _, p := range []currency.Pair{testUSDCPerpetual, testCoinPerpetual, testDatedFuture} {
		_, err = unloaded.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, Pair: p, PurchasePrice: 100, Amount: 2})
		assert.ErrorIsf(t, err, common.ErrFunctionNotSupported, "GetFeeByType should refuse to estimate %s's fees as spot's before pairs load", p)
	}
	fee, err = unloaded.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, Pair: currency.NewBTCUSD(), PurchasePrice: 100, Amount: 2})
	require.NoError(t, err, "GetFeeByType must not error for a spot pair before pairs load")
	assert.InDelta(t, 2.4, fee, 1e-12, "GetFeeByType should estimate a spot pair's fee before pairs load")
	// Without valid credentials the lowest tier is estimated, leaving the request as it was
	offline := newWrapperTestExchange(t)
	offline.SkipAuthCheck = false
	offline.SetCredentials(nil)
	f := &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: currency.NewBTCUSD(), PurchasePrice: 100, Amount: 2}
	fee, err = offline.GetFeeByType(t.Context(), f)
	require.NoError(t, err, "GetFeeByType must not error without credentials")
	assert.InDelta(t, 2.4, fee, 1e-12, "GetFeeByType should estimate the lowest tier's fee without credentials")
	assert.Equal(t, exchange.CryptocurrencyTradeFee, f.FeeType, "GetFeeByType should leave the request's fee type as it was")
}

func TestIsStablePair(t *testing.T) {
	t.Parallel()
	assert.True(t, isStablePair(currency.NewPair(currency.USDT, currency.USDC)), "isStablePair should report USDT-USDC")
	assert.False(t, isStablePair(currency.NewBTCUSD()), "isStablePair should not report BTC-USD")
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.GetAvailableTransferChains(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAvailableTransferChains must refuse an empty currency")
	chains, err := ex.GetAvailableTransferChains(t.Context(), currency.BTC)
	require.NoError(t, err, "GetAvailableTransferChains must not error")
	assert.Contains(t, chains, "bitcoin", "GetAvailableTransferChains should return bitcoin for BTC")
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	start, end := wrapperWindow(4 * time.Hour)
	for _, tc := range []struct {
		a        asset.Item
		p        currency.Pair
		interval kline.Interval
	}{
		{asset.Spot, currency.NewBTCUSD(), kline.OneHour},
		{asset.Futures, testFuturesPair, kline.OneHour},
		{asset.USDCMarginedFutures, testUSDCPerpetual, kline.OneHour},
		{asset.CoinMarginedFutures, testCoinPerpetual, kline.FourHour},
	} {
		candles, err := ex.GetHistoricCandles(t.Context(), tc.p, tc.a, tc.interval, start, end)
		require.NoErrorf(t, err, "GetHistoricCandles must not error for %s %s", tc.a, tc.p)
		require.NotEmptyf(t, candles.Candles, "GetHistoricCandles must return %s %s candles", tc.a, tc.p)
		assert.Equalf(t, tc.interval, candles.Interval, "GetHistoricCandles should return %s candles", tc.interval)
		for i := range candles.Candles {
			assert.Truef(t, candles.Candles[i].Time.Equal(candles.Candles[i].Time.Truncate(tc.interval.Duration())), "GetHistoricCandles should align %s %s candles to UTC, got %s", tc.a, tc.interval, candles.Candles[i].Time)
		}
		if mockTests && tc.a == asset.CoinMarginedFutures {
			require.Len(t, candles.Candles, 1, "GetHistoricCandles must return one 4 hour candle for the window")
			c := candles.Candles[0]
			assert.Equal(t, start, c.Time, "GetHistoricCandles should start the 4 hour candle at the window's start")
			assert.Equal(t, []float64{81410.5, 81983.5, 81314.5, 81859.5}, []float64{c.Open, c.High, c.Low, c.Close}, "GetHistoricCandles should build the 4 hour candle from the 2 hour candles")
			assert.InDelta(t, 1661.64346131, c.Volume, 1e-9, "GetHistoricCandles should sum the 2 hour candles' volumes")
		}
	}
}

func TestGetHistoricCandlesFullWindow(t *testing.T) {
	t.Parallel()
	// Coinbase includes both ends of a window and keeps its latest 350 candles, newest first
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hour := int64(time.Hour / time.Second)
		start, err := strconv.ParseInt(r.URL.Query().Get("start"), 10, 64)
		if !assert.NoError(t, err, "the candles request should hold a start") {
			return
		}
		end, err := strconv.ParseInt(r.URL.Query().Get("end"), 10, 64)
		if !assert.NoError(t, err, "the candles request should hold an end") {
			return
		}
		last := end - end%hour
		first := max((start+hour-1)/hour*hour, last-(maximumCandles-1)*hour)
		var candles []string
		for at := last; at >= first; at -= hour {
			candles = append(candles, `{"start":"`+strconv.FormatInt(at, 10)+`","low":"1","high":"3","open":"2","close":"2","volume":"1"}`)
		}
		_, err = w.Write([]byte(`{"candles":[` + strings.Join(candles, ",") + `]}`))
		assert.NoError(t, err, "writing the candles should not error")
	}))
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	candles, err := ex.GetHistoricCandles(t.Context(), currency.NewBTCUSD(), asset.Spot, kline.OneHour, start, start.Add(maximumCandles*time.Hour))
	require.NoError(t, err, "GetHistoricCandles must not error for a full window")
	require.Len(t, candles.Candles, maximumCandles, "GetHistoricCandles must return a full window's candles")
	assert.Equal(t, start, candles.Candles[0].Time, "GetHistoricCandles should return the candle at the window's start")
	extended, err := ex.GetHistoricCandlesExtended(t.Context(), currency.NewBTCUSD(), asset.Spot, kline.OneHour, start, start.Add(2*maximumCandles*time.Hour))
	require.NoError(t, err, "GetHistoricCandlesExtended must not error for two full windows")
	require.Len(t, extended.Candles, 2*maximumCandles, "GetHistoricCandlesExtended must return both windows' candles")
	for i := range extended.Candles {
		assert.Equalf(t, start.Add(time.Duration(i)*time.Hour), extended.Candles[i].Time, "GetHistoricCandlesExtended should return candle %d of an unbroken series", i)
		assert.Equalf(t, 2.0, extended.Candles[i].Open, "GetHistoricCandlesExtended should return Coinbase's candle %d rather than padding", i)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	start, end := wrapperWindow(4 * time.Hour)
	for _, tc := range []struct {
		a asset.Item
		p currency.Pair
	}{{asset.Spot, currency.NewBTCUSD()}, {asset.USDCMarginedFutures, testUSDCPerpetual}} {
		candles, err := ex.GetHistoricCandlesExtended(t.Context(), tc.p, tc.a, kline.OneHour, start, end)
		require.NoErrorf(t, err, "GetHistoricCandlesExtended must not error for %s %s", tc.a, tc.p)
		assert.NotEmptyf(t, candles.Candles, "GetHistoricCandlesExtended should return %s %s candles", tc.a, tc.p)
	}
}

func TestGetHistoricCandlesExtendedDerivativesBoundary(t *testing.T) {
	t.Parallel()
	// The gateway's 2 hour candles include the one at the window's end, each of volume 1
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start, err := strconv.ParseInt(r.URL.Query().Get("start_timestamp"), 10, 64)
		if !assert.NoError(t, err, "the candles request should hold a start") {
			return
		}
		end, err := strconv.ParseInt(r.URL.Query().Get("end_timestamp"), 10, 64)
		if !assert.NoError(t, err, "the candles request should hold an end") {
			return
		}
		twoHours := (2 * time.Hour).Milliseconds()
		var ticks, values []string
		for at := (start + twoHours - 1) / twoHours * twoHours; at <= end; at += twoHours {
			ticks = append(ticks, strconv.FormatInt(at, 10))
			values = append(values, "1")
		}
		list := strings.Join(values, ",")
		_, err = w.Write([]byte(`{"jsonrpc":"2.0","result":{"status":"ok","ticks":[` + strings.Join(ticks, ",") + `],"open":[` + list + `],"high":[` + list + `],"low":[` + list + `],"close":[` + list + `],"volume":[` + list + `],"cost":[` + list + `]}}`))
		assert.NoError(t, err, "writing the candles should not error")
	}))
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles, err := ex.GetHistoricCandlesExtended(t.Context(), testCoinPerpetual, asset.CoinMarginedFutures, kline.SixHour, start, start.Add((maximumCandles+1)*6*time.Hour))
	require.NoError(t, err, "GetHistoricCandlesExtended must not error")
	require.Len(t, candles.Candles, maximumCandles+1, "GetHistoricCandlesExtended must return every 6 hour candle across both requests")
	for i := range candles.Candles {
		assert.Equalf(t, 3.0, candles.Candles[i].Volume, "the 6 hour candle at %s should be built from three 2 hour candles", candles.Candles[i].Time)
	}
}

func TestCandleGranularity(t *testing.T) {
	t.Parallel()
	for interval, exp := range map[kline.Interval]string{
		kline.OneMin: GranularityOneMinute, kline.FiveMin: GranularityFiveMinute, kline.FifteenMin: GranularityFifteenMinute,
		kline.ThirtyMin: GranularityThirtyMinute, kline.OneHour: GranularityOneHour, kline.TwoHour: GranularityTwoHour,
		kline.FourHour: GranularityFourHour, kline.SixHour: GranularitySixHour, kline.OneDay: GranularityOneDay,
	} {
		got, err := candleGranularity(interval)
		require.NoErrorf(t, err, "candleGranularity must not error for %s", interval)
		assert.Equalf(t, exp, got, "candleGranularity should convert %s", interval)
	}
	_, err := candleGranularity(kline.ThreeMin)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "candleGranularity should refuse an interval Advanced Trade does not serve")
	assert.True(t, derivativesCandleAligned(kline.TwoHour), "derivativesCandleAligned should report the gateway's 2 hour candles aligned to UTC")
	for _, interval := range []kline.Interval{kline.FourHour, kline.SixHour, kline.OneDay} {
		assert.Falsef(t, derivativesCandleAligned(interval), "derivativesCandleAligned should not report the gateway's %s candles aligned to UTC", interval)
	}
}

func TestAggregateCandles(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 10, 18, 2, 0, 0, 0, time.UTC)
	candles := []kline.Candle{
		{Time: start, Open: 1, High: 3, Low: 1, Close: 2, Volume: 1},
		{Time: start.Add(2 * time.Hour), Open: 2, High: 5, Low: 0.5, Close: 4, Volume: 2},
		{Time: start.Add(4 * time.Hour), Open: 4, High: 4, Low: 3, Close: 3, Volume: 3},
	}
	assert.Equal(t, []kline.Candle{
		{Time: start.Add(-2 * time.Hour), Open: 1, High: 5, Low: 0.5, Close: 4, Volume: 3},
		{Time: start.Add(4 * time.Hour), Open: 4, High: 4, Low: 3, Close: 3, Volume: 3},
	}, aggregateCandles(candles, kline.SixHour), "aggregateCandles should merge candles into UTC aligned candles of the longer interval")
	assert.Empty(t, aggregateCandles(nil, kline.OneDay), "aggregateCandles should return nothing for no candles")
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	for _, a := range []asset.Item{asset.Spot, asset.USDCMarginedFutures} {
		st, err := ex.GetServerTime(t.Context(), a)
		require.NoErrorf(t, err, "GetServerTime must not error for %s", a)
		assert.Falsef(t, st.IsZero(), "GetServerTime should return a time for %s", a)
	}
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetFuturesContractDetails must refuse a spot asset")
	_, err = ex.GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetFuturesContractDetails must refuse a futures asset Coinbase does not trade")
	for a, pairs := range wrapperTestPairs(t) {
		if !a.IsFutures() {
			continue
		}
		contracts, err := ex.GetFuturesContractDetails(t.Context(), a)
		require.NoErrorf(t, err, "GetFuturesContractDetails must not error for %s", a)
		i := slices.IndexFunc(contracts, func(c futures.Contract) bool { return c.Name.Equal(pairs[0]) })
		require.NotEqualf(t, -1, i, "GetFuturesContractDetails must return %s", pairs[0])
		assert.Equalf(t, futures.Perpetual, contracts[i].Type, "GetFuturesContractDetails should return %s as perpetual", pairs[0])
		assert.Positivef(t, contracts[i].Multiplier, "GetFuturesContractDetails should return %s's contract size", pairs[0])
		exp := map[asset.Item]futures.ContractSettlementType{asset.Futures: futures.Linear, asset.USDCMarginedFutures: futures.Linear, asset.CoinMarginedFutures: futures.Inverse}[a]
		assert.Equalf(t, exp, contracts[i].SettlementType, "GetFuturesContractDetails should return how %s settles", pairs[0])
	}
	if !mockTests {
		return
	}
	contracts, err := ex.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.NoError(t, err, "GetFuturesContractDetails must not error for futures")
	require.Len(t, contracts, 3, "GetFuturesContractDetails must return every listed US derivative, view only ones among them")
	assert.Equal(t, futures.Contract{
		Exchange:           ex.Name,
		Name:               testFuturesPair,
		Underlying:         currency.NewBTCUSD(),
		Asset:              asset.Futures,
		StartDate:          time.Date(2025, 7, 18, 13, 0, 0, 0, time.UTC),
		IsActive:           true,
		Status:             "FCM_TRADING_SESSION_STATE_OPEN",
		Type:               futures.Perpetual,
		SettlementType:     futures.Linear,
		SettlementCurrency: currency.USD,
		MarginCurrency:     currency.USD,
		Multiplier:         0.01,
		LatestRate:         fundingrate.Rate{Time: time.Date(2026, 10, 19, 2, 0, 0, 0, time.UTC), Rate: decimal.MustFromFloat(0.000012)},
	}, contracts[slices.IndexFunc(contracts, func(c futures.Contract) bool { return c.Name.Equal(testFuturesPair) })], "GetFuturesContractDetails should convert a perpetual style US derivative")
	dated := contracts[slices.IndexFunc(contracts, func(c futures.Contract) bool { return c.Name.Equal(testDatedFuture) })]
	assert.Equal(t, futures.Monthly, dated.Type, "GetFuturesContractDetails should return a dated US derivative as monthly")
	assert.Equal(t, time.Date(2026, 10, 30, 16, 0, 0, 0, time.UTC), dated.EndDate, "GetFuturesContractDetails should return a dated US derivative's expiry")
	viewOnly := contracts[slices.IndexFunc(contracts, func(c futures.Contract) bool { return c.Name.Quote.String() == "27NOV26-CDE" })]
	assert.False(t, viewOnly.IsActive, "GetFuturesContractDetails should return a view only US derivative as inactive")

	contracts, err = ex.GetFuturesContractDetails(t.Context(), asset.CoinMarginedFutures)
	require.NoError(t, err, "GetFuturesContractDetails must not error for coin margined futures")
	assert.Equal(t, []futures.Contract{{
		Exchange:           ex.Name,
		Name:               testCoinPerpetual,
		Underlying:         currency.NewBTCUSD(),
		Asset:              asset.CoinMarginedFutures,
		StartDate:          time.UnixMilli(1534242287000),
		IsActive:           true,
		Status:             "open",
		Type:               futures.Perpetual,
		SettlementType:     futures.Inverse,
		SettlementCurrency: currency.BTC,
		MarginCurrency:     currency.BTC,
		Multiplier:         10,
		MaxLeverage:        50,
	}}, contracts, "GetFuturesContractDetails should convert the gateway's inverse perpetual")
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.GetLatestFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLatestFundingRates must refuse a nil request")
	_, err = ex.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Futures, IncludePredictedRate: true})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetLatestFundingRates must refuse to predict rates")
	_, err = ex.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Spot})
	require.ErrorIs(t, err, futures.ErrNotPerpetualFuture, "GetLatestFundingRates must refuse spot")
	_, err = ex.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Futures, Pair: datedFuture(t)})
	require.ErrorIs(t, err, futures.ErrNotPerpetualFuture, "GetLatestFundingRates must refuse a dated future")
	rates, err := ex.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Futures, Pair: testFuturesPair})
	require.NoError(t, err, "GetLatestFundingRates must not error for a perpetual style future")
	require.Len(t, rates, 1, "GetLatestFundingRates must return the future's rate")
	if mockTests {
		lastFunding := time.Date(2026, 10, 19, 2, 0, 0, 0, time.UTC)
		assert.Equal(t, fundingrate.LatestRateResponse{
			Exchange:       ex.Name,
			Asset:          asset.Futures,
			Pair:           testFuturesPair,
			LatestRate:     fundingrate.Rate{Time: lastFunding, Rate: decimal.MustFromFloat(0.000012)},
			TimeOfNextRate: lastFunding.Add(time.Hour),
			TimeChecked:    rates[0].TimeChecked,
		}, rates[0], "GetLatestFundingRates should return the future's hourly funding rate")
	}
	assert.False(t, rates[0].LatestRate.Time.IsZero(), "GetLatestFundingRates should return when the future last paid funding")
	assert.True(t, rates[0].TimeOfNextRate.After(rates[0].LatestRate.Time), "GetLatestFundingRates should return when the future next pays funding")
	rates, err = ex.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Futures})
	require.NoError(t, err, "GetLatestFundingRates must not error for every enabled future")
	require.Len(t, rates, 1, "GetLatestFundingRates must return the enabled perpetual style future's rate alone")
	assert.True(t, rates[0].Pair.Equal(testFuturesPair), "GetLatestFundingRates should return the enabled perpetual style future")

	gateway := ex
	if mockTests {
		var venue *derivativesVenue
		gateway, venue = newDerivativesWebsocketTestExchange(t)
		for a, pairs := range wrapperTestPairs(t) {
			require.NoErrorf(t, gateway.CurrencyPairs.StorePairs(a, pairs, false), "StorePairs must not error for available %s pairs", a)
			require.NoErrorf(t, gateway.CurrencyPairs.StorePairs(a, pairs, true), "StorePairs must not error for enabled %s pairs", a)
		}
		// The window ends now, so the mock answers whatever window is asked for
		venue.override("public/get_funding_rate_history", []byte(`{"result":[{"timestamp":1792368000000,"index_price":82000,"prev_index_price":81900,"interest_8h":0.0001,"interest_1h":0.0000125}]}`))
	}
	rates, err = gateway.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.USDCMarginedFutures})
	require.NoError(t, err, "GetLatestFundingRates must not error for a gateway perpetual")
	require.Len(t, rates, 1, "GetLatestFundingRates must return the enabled gateway perpetual's rate")
	assert.Equal(t, rates[0].LatestRate.Time.Add(derivativesFundingInterval), rates[0].TimeOfNextRate, "GetLatestFundingRates should return when the perpetual next records funding")
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	start, end := wrapperWindow(4 * time.Hour)
	for _, tc := range []struct {
		r   *fundingrate.HistoricalRatesRequest
		err error
	}{
		{nil, common.ErrNilPointer},
		{&fundingrate.HistoricalRatesRequest{Asset: asset.Futures}, common.ErrFunctionNotSupported},
		{&fundingrate.HistoricalRatesRequest{Asset: asset.Spot}, futures.ErrNotPerpetualFuture},
		{&fundingrate.HistoricalRatesRequest{Asset: asset.USDCMarginedFutures, IncludePayments: true}, common.ErrFunctionNotSupported},
		{&fundingrate.HistoricalRatesRequest{Asset: asset.USDCMarginedFutures, IncludePredictedRate: true}, common.ErrFunctionNotSupported},
		{&fundingrate.HistoricalRatesRequest{Asset: asset.USDCMarginedFutures, Pair: testUSDCPerpetual, StartDate: end, EndDate: start}, common.ErrStartAfterEnd},
	} {
		_, err := ex.GetHistoricalFundingRates(t.Context(), tc.r)
		assert.ErrorIsf(t, err, tc.err, "GetHistoricalFundingRates should refuse %+v", tc.r)
	}
	rates, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.USDCMarginedFutures, Pair: testUSDCPerpetual, StartDate: start, EndDate: end})
	require.NoError(t, err, "GetHistoricalFundingRates must not error")
	require.NotEmpty(t, rates.FundingRates, "GetHistoricalFundingRates must return the window's hourly rates")
	assert.Equal(t, rates.FundingRates[len(rates.FundingRates)-1], rates.LatestRate, "GetHistoricalFundingRates should return the window's latest rate")
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	for a, exp := range map[asset.Item]bool{asset.Spot: false, asset.USDCMarginedFutures: true, asset.CoinMarginedFutures: true} {
		perpetual, err := ex.IsPerpetualFutureCurrency(a, currency.NewBTCUSD())
		require.NoErrorf(t, err, "IsPerpetualFutureCurrency must not error for %s", a)
		assert.Equalf(t, exp, perpetual, "IsPerpetualFutureCurrency should report whether %s trades perpetuals", a)
	}
	_, err := ex.IsPerpetualFutureCurrency(asset.Futures, testFuturesPair)
	require.ErrorIs(t, err, errPerpetualFuturesNotLoaded, "IsPerpetualFutureCurrency must refuse a future until product data loads")
	ex.loadPerpetualFutures([]ProductResponse{{ProductID: "BIP-20DEC30-CDE", FutureProductDetails: &FutureProductDetails{FundingInterval: Duration(time.Hour)}}, {ProductID: "BIT-30OCT26-CDE", FutureProductDetails: &FutureProductDetails{}}})
	for p, exp := range map[currency.Pair]bool{testFuturesPair: true, testDatedFuture: false} {
		perpetual, err := ex.IsPerpetualFutureCurrency(asset.Futures, p)
		require.NoErrorf(t, err, "IsPerpetualFutureCurrency must not error for %s", p)
		assert.Equalf(t, exp, perpetual, "IsPerpetualFutureCurrency should report whether %s pays funding", p)
	}
	_, err = ex.IsPerpetualFutureCurrency(asset.Futures, currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "IsPerpetualFutureCurrency should refuse an empty future")
}

func TestGetCollateralCurrencyForContract(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	for _, tc := range []struct {
		a   asset.Item
		p   currency.Pair
		exp currency.Code
	}{
		{asset.Futures, testFuturesPair, currency.USD},
		{asset.USDCMarginedFutures, testUSDCPerpetual, currency.USDC},
		{asset.CoinMarginedFutures, testCoinPerpetual, currency.BTC},
	} {
		c, a, err := ex.GetCollateralCurrencyForContract(tc.a, tc.p)
		require.NoErrorf(t, err, "GetCollateralCurrencyForContract must not error for %s %s", tc.a, tc.p)
		assert.Truef(t, tc.exp.Equal(c), "GetCollateralCurrencyForContract should return %s for %s %s, got %s", tc.exp, tc.a, tc.p, c)
		assert.Equalf(t, tc.a, a, "GetCollateralCurrencyForContract should return %s's own asset", tc.a)
	}
	_, _, err := ex.GetCollateralCurrencyForContract(asset.Spot, currency.NewBTCUSD())
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetCollateralCurrencyForContract should refuse spot")
	_, _, err = ex.GetCollateralCurrencyForContract(asset.USDCMarginedFutures, currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCollateralCurrencyForContract should refuse an empty perpetual")
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.GetOpenInterest(t.Context(), key.PairAsset{Base: currency.BTC.Item, Quote: currency.USD.Item, Asset: asset.Spot})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOpenInterest must refuse spot")
	keys := []key.PairAsset{
		{Base: testFuturesPair.Base.Item, Quote: testFuturesPair.Quote.Item, Asset: asset.Futures},
		{Base: testUSDCPerpetual.Base.Item, Quote: testUSDCPerpetual.Quote.Item, Asset: asset.USDCMarginedFutures},
		{Base: testCoinPerpetual.Base.Item, Quote: testCoinPerpetual.Quote.Item, Asset: asset.CoinMarginedFutures},
	}
	interest, err := ex.GetOpenInterest(t.Context(), keys...)
	require.NoError(t, err, "GetOpenInterest must not error for the pairs given")
	require.Len(t, interest, len(keys), "GetOpenInterest must return each pair given")
	for i := range interest {
		assert.Equalf(t, keys[i].Asset, interest[i].Key.Asset, "GetOpenInterest should return the pairs in the order given")
		assert.Positivef(t, interest[i].OpenInterest, "GetOpenInterest should return %s's open interest", interest[i].Key.Pair())
	}
	all, err := ex.GetOpenInterest(t.Context())
	require.NoError(t, err, "GetOpenInterest must not error for every pair")
	assert.Greater(t, len(all), len(keys), "GetOpenInterest should return every futures pair the venues list")
	_, err = ex.GetOpenInterest(t.Context(), key.PairAsset{Base: currency.NewCode("UNLISTED").Item, Quote: currency.NewCode("PERPETUAL").Item, Asset: asset.CoinMarginedFutures})
	assert.ErrorIs(t, err, errInstrumentNotFound, "GetOpenInterest should report a pair the venues do not list")
}

func TestLeverage(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	require.ErrorIs(t, ex.SetLeverage(t.Context(), asset.Futures, testFuturesPair, margin.Multi, 2, order.UnknownSide), common.ErrFunctionNotSupported, "SetLeverage must refuse US derivatives")
	_, err := ex.GetLeverage(t.Context(), asset.Futures, testFuturesPair, margin.Multi, order.UnknownSide)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetLeverage must refuse US derivatives")
	require.ErrorIs(t, ex.SetLeverage(t.Context(), asset.Spot, currency.NewBTCUSD(), margin.Multi, 2, order.UnknownSide), asset.ErrNotSupported, "SetLeverage must refuse spot")
	require.ErrorIs(t, ex.SetLeverage(t.Context(), asset.USDCMarginedFutures, testUSDCPerpetual, margin.NoMargin, 2, order.UnknownSide), margin.ErrMarginTypeUnsupported, "SetLeverage must refuse an unsupported margin type")
	if !mockTests {
		sharedTestCredentials(t, ex)
	}
	leverage, err := ex.GetLeverage(t.Context(), asset.CoinMarginedFutures, testCoinPerpetual, margin.Multi, order.UnknownSide)
	require.NoError(t, err, "GetLeverage must not error")
	assert.Positive(t, leverage, "GetLeverage should return the perpetual's leverage")
	if !mockTests && !canManipulateRealOrders {
		t.Skip("canManipulateRealOrders is not set")
	}
	assert.NoError(t, ex.SetLeverage(t.Context(), asset.USDCMarginedFutures, testSOLUSDCPerpetual, margin.Isolated, 50, order.UnknownSide), "SetLeverage should not error")
}

func TestGetFuturesPositionSummary(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.GetFuturesPositionSummary(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesPositionSummary must refuse a nil request")
	_, err = ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Spot})
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetFuturesPositionSummary must refuse spot")
	if !mockTests {
		sharedTestCredentials(t, ex)
		t.Skip("live positions are the account's own")
	}
	summary, err := ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Futures, Pair: testDatedFuture})
	require.NoError(t, err, "GetFuturesPositionSummary must not error for a US derivative")
	assert.False(t, summary.CurrentSize.IsZero(), "GetFuturesPositionSummary should return the US derivative position's size")
	assert.False(t, summary.ContractMultiplier.IsZero(), "GetFuturesPositionSummary should return the US derivative's contract size")
	summary, err = ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.CoinMarginedFutures, Pair: testCoinPerpetual})
	require.NoError(t, err, "GetFuturesPositionSummary must not error for a gateway perpetual")
	assert.Equal(t, futures.Inverse, summary.ContractSettlementType, "GetFuturesPositionSummary should return how the gateway perpetual settles")
}

func TestGetFuturesPositionSummaryGatewayContracts(t *testing.T) {
	t.Parallel()
	rpcResult := func(result string) string { return `{"jsonrpc":"2.0","id":1,"result":` + result + `}` }
	for _, tc := range []struct {
		a          asset.Item
		pair       currency.Pair
		instrument string
		position   string
		contracts  string
	}{
		{
			// The isolated positions guide's position: 10 SOL bought at 100, so its size is 1000 USDC
			a:          asset.USDCMarginedFutures,
			pair:       testSOLUSDCPerpetual,
			instrument: `{"instrument_name":"SOL_USDC-PERPETUAL","kind":"future","instrument_type":"linear","settlement_period":"perpetual","base_currency":"SOL","quote_currency":"USDC","counter_currency":"USDC","settlement_currency":"USDC","contract_size":0.001,"min_trade_amount":0.001,"tick_size":0.001,"is_active":true}`,
			position:   `{"instrument_name":"SOL_USDC-PERPETUAL","kind":"future","user_id":1000,"direction":"buy","size":1000,"size_currency":10,"average_price":100,"mark_price":100}`,
			contracts:  "10000",
		},
		{
			a:          asset.CoinMarginedFutures,
			pair:       testCoinPerpetual,
			instrument: `{"instrument_name":"BTC-PERPETUAL","kind":"future","instrument_type":"reversed","settlement_period":"perpetual","base_currency":"BTC","quote_currency":"USD","counter_currency":"USD","settlement_currency":"BTC","contract_size":10,"min_trade_amount":10,"tick_size":0.5,"is_active":true}`,
			position:   `{"instrument_name":"BTC-PERPETUAL","kind":"future","user_id":1000,"direction":"sell","size":-1000,"size_currency":-0.01,"average_price":100000,"mark_price":100000}`,
			contracts:  "-100",
		},
	} {
		ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body string
			switch r.URL.Path {
			case "/api/v2/public/get_instrument":
				body = rpcResult(tc.instrument)
			case "/api/v2/private/get_position":
				body = rpcResult(tc.position)
			default:
				http.NotFound(w, r)
				return
			}
			_, err := w.Write([]byte(body))
			assert.NoError(t, err, "writing the response should not error")
		}))
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(tc.a, currency.Pairs{tc.pair}, false), "StorePairs must not error for %s", tc.a)
		summary, err := ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: tc.a, Pair: tc.pair})
		require.NoErrorf(t, err, "GetFuturesPositionSummary must not error for %s", tc.pair)
		assert.Equalf(t, tc.contracts, summary.ContractSize.String(), "GetFuturesPositionSummary should return %s's size in contracts", tc.pair)
	}
}

func TestGetFuturesPositionOrders(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.GetFuturesPositionOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesPositionOrders must refuse a nil request")
	_, err = ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.Spot})
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetFuturesPositionOrders must refuse spot")
	_, err = ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.USDCMarginedFutures})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "GetFuturesPositionOrders must refuse no pairs")
	if !mockTests {
		sharedTestCredentials(t, ex)
	}
	start, end := wrapperWindow(24 * time.Hour)
	resp, err := ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.USDCMarginedFutures, Pairs: currency.Pairs{testUSDCPerpetual}, StartDate: start, EndDate: end})
	require.NoError(t, err, "GetFuturesPositionOrders must not error")
	require.Len(t, resp, 1, "GetFuturesPositionOrders must return the pair's orders")
	assert.Equal(t, futures.Linear, resp[0].ContractSettlementType, "GetFuturesPositionOrders should return how the perpetual settles")
}

func TestGetFuturesPositionSummaryGatewayUnrealisedProfit(t *testing.T) {
	t.Parallel()
	rpcResult := func(result string) string { return `{"jsonrpc":"2.0","id":1,"result":` + result + `}` }
	// The spec's private/get_positions example: total_profit_loss covers the position since it opened, while
	// realized_profit_loss and realized_funding are the session's
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body string
		switch r.URL.Path {
		case "/api/v2/public/get_instrument":
			body = rpcResult(`{"instrument_name":"BTC-PERPETUAL","kind":"future","instrument_type":"reversed","settlement_period":"perpetual","base_currency":"BTC","quote_currency":"USD","counter_currency":"USD","settlement_currency":"BTC","contract_size":10,"min_trade_amount":10,"tick_size":0.5,"is_active":true}`)
		case "/api/v2/private/get_position":
			body = rpcResult(`{"average_price":7440.18,"delta":0.006687487,"direction":"buy","estimated_liquidation_price":null,"floating_profit_loss":0,"index_price":7466.79,"initial_margin":0.000197283,"instrument_name":"BTC-PERPETUAL","interest_value":1.7362511643080387,"kind":"future","leverage":34,"maintenance_margin":0.000143783,"mark_price":7476.65,"open_orders_margin":0.000197288,"realized_funding":-1e-08,"realized_profit_loss":-9e-09,"settlement_price":7476.65,"size":50,"size_currency":0.006687487,"total_profit_loss":3.2781e-05,"user_id":1234}`)
		default:
			http.NotFound(w, r)
			return
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "writing the response should not error")
	}))
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.CoinMarginedFutures, currency.Pairs{testCoinPerpetual}, false), "StorePairs must not error")
	summary, err := ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.CoinMarginedFutures, Pair: testCoinPerpetual})
	require.NoError(t, err, "GetFuturesPositionSummary must not error")
	assert.Equal(t, "0.000032781", summary.UnrealisedPNL.String(), "GetFuturesPositionSummary should return the open position's profit as unrealised")
	assert.Equal(t, "-0.000000009", summary.RealisedPNL.String(), "GetFuturesPositionSummary should return the session's realised profit")
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	_, err := ex.GetCurrencyTradeURL(t.Context(), asset.Spot, currency.EMPTYPAIR)
	require.Error(t, err, "GetCurrencyTradeURL must refuse an empty pair")
	// Coinbase's web app links its trade pages by these paths, and lists no inverse perpetual
	for a, exp := range map[asset.Item]string{
		asset.Spot:                tradeBaseURL + "spot/BTC-USD",
		asset.Futures:             tradeBaseURL + "futures/BIP-20DEC30-CDE",
		asset.USDCMarginedFutures: tradeBaseURL + "perpetuals/BTC_USDC-PERPETUAL",
	} {
		pairs := wrapperTestPairs(t)[a]
		u, err := ex.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoErrorf(t, err, "GetCurrencyTradeURL must not error for %s", a)
		assert.Equalf(t, exp, u, "GetCurrencyTradeURL should return the %s trade page", a)
	}
	_, err = ex.GetCurrencyTradeURL(t.Context(), asset.CoinMarginedFutures, wrapperTestPairs(t)[asset.CoinMarginedFutures][0])
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetCurrencyTradeURL should report no trade page for an inverse perpetual")
	_, err = ex.GetCurrencyTradeURL(t.Context(), asset.Options, currency.NewBTCUSD())
	assert.Error(t, err, "GetCurrencyTradeURL should refuse an asset Coinbase does not trade")
}

func TestDerivativesOverWebsocket(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	assert.False(t, ex.derivativesOverWebsocket(t.Context(), false), "derivativesOverWebsocket should not use a websocket that is not connected")
	if !mockTests {
		t.Skip("the live tests share one websocket connection")
	}
	connected, venue := newDerivativesWebsocketTestExchange(t)
	assert.True(t, connected.derivativesOverWebsocket(t.Context(), false), "derivativesOverWebsocket should send public requests over a connected websocket")
	assert.True(t, connected.derivativesOverWebsocket(t.Context(), true), "derivativesOverWebsocket should send private requests over an authenticated websocket")
	// A session acts for the account of the API key it authenticated with alone
	other := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "organizations/other/apiKeys/other", Secret: "other"})
	assert.False(t, connected.derivativesOverWebsocket(other, true), "derivativesOverWebsocket should send private requests of another API key over HTTP")
	assert.True(t, connected.derivativesOverWebsocket(other, false), "derivativesOverWebsocket should send public requests over the websocket whatever the credentials")
	var positions []DerivativesPositionResponse
	err := connected.WebsocketAPI().SendRequest(other, derivativesNonMatchingEPL, "private/get_positions", nil, &positions)
	assert.ErrorIs(t, err, errDerivativesWebsocketOtherAccount, "SendRequest should refuse a private method of another API key over the websocket")
	assert.Empty(t, venue.requestsFor("private/get_positions"), "SendRequest should not send another API key's private method over the websocket")
	conn, err := connected.Websocket.GetConnection(derivativesPrivateConnection)
	require.NoError(t, err, "GetConnection must not error for the private gateway connection")
	connected.derivativesSessions.set(conn, "")
	assert.False(t, connected.derivativesOverWebsocket(t.Context(), true), "derivativesOverWebsocket should not send private requests over a connection that is not authenticated")
	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	assert.False(t, unauthenticated.derivativesOverWebsocket(t.Context(), true), "derivativesOverWebsocket should not send private requests when the websocket may not authenticate")
}

func TestWrapperOrdersThroughWebsocket(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("the mock records the requests sent over the websocket")
	}
	ex, venue := newDerivativesWebsocketTestExchange(t)
	for a, pairs := range wrapperTestPairs(t) {
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, pairs, false), "StorePairs must not error for available %s pairs", a)
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, pairs, true), "StorePairs must not error for enabled %s pairs", a)
	}
	_, err := ex.SubmitOrder(t.Context(), &order.Submit{
		Exchange:      ex.Name,
		Pair:          testUSDCPerpetual,
		AssetType:     asset.USDCMarginedFutures,
		Side:          order.Buy,
		Type:          order.Limit,
		Amount:        0.001,
		Price:         50000,
		ClientOrderID: "gct-wrapper-order",
	})
	require.NoError(t, err, "SubmitOrder must not error over the websocket")
	assert.Len(t, venue.requestsFor(derivativesBuyMethod), 1, "SubmitOrder should place the gateway order over the websocket")
	require.NoError(t, ex.CancelOrder(t.Context(), &order.Cancel{OrderID: testWrapperDerivativesOrderID, AssetType: asset.USDCMarginedFutures}), "CancelOrder must not error over the websocket")
	assert.Len(t, venue.requestsFor("private/cancel"), 1, "CancelOrder should cancel the gateway order over the websocket")
}

func TestAuthenticateWebsocket(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	assert.Error(t, ex.AuthenticateWebsocket(t.Context()), "AuthenticateWebsocket should error without a gateway connection")
	if !mockTests {
		t.Skip("the live tests share one websocket connection")
	}
	connected, venue := newDerivativesWebsocketTestExchange(t)
	require.NoError(t, connected.AuthenticateWebsocket(t.Context()), "AuthenticateWebsocket must not error")
	assert.Len(t, venue.requestsFor(derivativesAuthMethod), 2, "AuthenticateWebsocket should authenticate the gateway connection again")
}
