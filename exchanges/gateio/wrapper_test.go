package gateio

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testutils "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()

	_, err := e.CancelAllOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelAllOrders must reject a nil cancellation")

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{Pair: currency.EMPTYPAIR, AssetType: 1336})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelAllOrders must reject an empty pair")

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{Pair: currency.NewBTCUSDT(), AssetType: 1336})
	require.ErrorIs(t, err, asset.ErrNotSupported, "CancelAllOrders must reject an unsupported asset")

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{
		Pair:      currency.NewBTCUSDT(),
		AssetType: asset.Options,
		Side:      order.ClosePosition,
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CancelAllOrders must reject a side it cannot cancel by")

	for _, side := range []order.Side{order.Long, order.Short, order.AnySide} {
		_, err = e.CancelAllOrders(t.Context(), &order.Cancel{
			Pair:      currency.NewPair(currency.BTC, currency.EMPTYCODE),
			AssetType: asset.USDTMarginedFutures,
			Side:      side,
		})
		require.ErrorIsf(t, err, errInvalidSettlementQuote, "CancelAllOrders must reject a pair the asset does not settle for side %s", side)
	}

	for _, a := range e.GetAssetTypes(false) {
		_, err = e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: a})
		require.ErrorIsf(t, err, currency.ErrCurrencyPairEmpty, "CancelAllOrders must reject an empty pair for %s", a)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		a    asset.Item
		side order.Side
		exp  order.CancelAllResponse
	}{
		{a: asset.Spot, side: order.Buy, exp: order.CancelAllResponse{Status: map[string]string{"1852454420": "cancelled"}}},
		{a: asset.Margin, side: order.Buy, exp: order.CancelAllResponse{Status: map[string]string{"1852454430": "cancelled"}}},
		{a: asset.CrossMargin, side: order.Buy, exp: order.CancelAllResponse{Status: map[string]string{"1852454431": "cancelled"}}},
		{a: asset.USDTMarginedFutures, side: order.Sell, exp: order.CancelAllResponse{Status: map[string]string{"15675394": "cancelled"}}},
		{a: asset.CoinMarginedFutures, exp: order.CancelAllResponse{Status: map[string]string{"15675405": "cancelled"}}},
		{a: asset.DeliveryFutures, side: order.Short, exp: order.CancelAllResponse{Status: map[string]string{"123456": "cancelled"}}},
		{a: asset.Options, exp: order.CancelAllResponse{Status: map[string]string{"1236": "cancelled"}}},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			got, err := e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: tc.a, Pair: getPair(t, tc.a), Side: tc.side})
			require.NoError(t, err, "CancelAllOrders must not error")
			if mockTests {
				assert.Equal(t, tc.exp, got, "CancelAllOrders should report each cancelled order")
			}
		})
	}
}

// TestContractOpenInterest pins the open interest each contract kind reports, in its quote
// currency. Figures are a live BTC_USD and BTC_USDT response, so a regression restates a real
// contract rather than an invented one
func TestContractOpenInterest(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		contract openInterestContract
		exp      float64
	}{
		{
			// Gate reports quanto_multiplier "0" here, so multiplying by the index price gave
			// $515bn against the $13m the contract stats report for the same contract
			name:     "inverse futures contract is one unit of its quote currency",
			contract: &FuturesContractResponse{Type: "inverse", PositionSize: 6542499, IndexPrice: 78827.15},
			exp:      6542499,
		},
		{
			name:     "direct futures contract is scaled by its multiplier and index price",
			contract: &FuturesContractResponse{Type: "direct", PositionSize: 1000, QuantoMultiplier: 0.0001, IndexPrice: 50000},
			exp:      5000,
		},
		{
			// nothing to size the position with, rather than a figure invented from the index price
			name:     "direct futures contract reporting no multiplier sizes no position",
			contract: &FuturesContractResponse{Type: "direct", PositionSize: 1000, IndexPrice: 50000},
			exp:      0,
		},
		{
			name:     "delivery contract is scaled by its multiplier and index price",
			contract: &DeliveryContractResponse{QuantoMultiplier: 1, PositionSize: 45, IndexPrice: 6.7608},
			exp:      304.236,
		},
		{
			// Gate lists no inverse delivery contract today, so this pins the shape rather than a
			// live response
			name:     "inverse delivery contract is one unit of its quote currency",
			contract: &DeliveryContractResponse{Type: "inverse", PositionSize: 6542499, IndexPrice: 78827.15},
			exp:      6542499,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.InDelta(t, tc.exp, tc.contract.openInterest(), 1e-6, "openInterest should return the position sized in the quote currency")
		})
	}
}

// TestFuturesContractOpenInterestFromResponse pins contractTypeInverse and the type tag against a
// real Gate response, which the table above cannot do while it builds the contract in Go
func TestFuturesContractOpenInterestFromResponse(t *testing.T) {
	t.Parallel()

	// trimmed from GET /api/v4/futures/btc/contracts/BTC_USD
	const resp = `{"name":"BTC_USD","type":"inverse","quanto_multiplier":"0","index_price":"78827.15","position_size":6542499}`

	var c FuturesContractResponse
	require.NoError(t, json.Unmarshal([]byte(resp), &c), "Unmarshal must not error")
	require.Equal(t, contractTypeInverse, c.Type, "the contract must decode as the inverse type the constant names")
	assert.Equal(t, 6542499.0, c.openInterest(), "openInterest should report the position size for an inverse contract")
}

// TestGetOpenInterestIndependentOfRequestShape pins a contract reporting the same open interest
// however many pairs are asked for, contract_stats having summed both sides of the book
func TestGetOpenInterestIndependentOfRequestShape(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	// Trimmed from GET /api/v4/futures/usdt/contracts, which serves one object for a named contract
	// and an array for all of them
	const btcContract = `{"name":"BTC_USDT","type":"direct","quanto_multiplier":"0.0001","index_price":"50000","position_size":1000}`
	const ethContract = `{"name":"ETH_USDT","type":"direct","quanto_multiplier":"0.01","index_price":"2000","position_size":500}`

	var requests atomic.Int64
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		assert.NotContains(t, r.URL.Path, "contract_stats", "open interest should not reach the contract stats endpoint")
		body := "[" + btcContract + "," + ethContract + "]"
		if strings.HasSuffix(r.URL.Path, "/BTC_USDT") {
			body = btcContract
		}
		_, err := fmt.Fprint(w, body)
		assert.NoError(t, err, "writing the contracts response should not error")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

	btcPair := currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
	ethPair := currency.NewPairWithDelimiter("ETH", "USDT", currency.UnderscoreDelimiter)
	pairs := currency.Pairs{btcPair, ethPair}
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.USDTMarginedFutures, pairs, false), "StorePairs must not error for available pairs")
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.USDTMarginedFutures, pairs, true), "StorePairs must not error for enabled pairs")

	btc := key.PairAsset{Base: currency.BTC.Item, Quote: currency.USDT.Item, Asset: asset.USDTMarginedFutures}
	eth := key.PairAsset{Base: currency.ETH.Item, Quote: currency.USDT.Item, Asset: asset.USDTMarginedFutures}

	single, err := ex.GetOpenInterest(t.Context(), btc)
	require.NoError(t, err, "GetOpenInterest must not error for a single pair")
	require.Len(t, single, 1, "GetOpenInterest must report the one pair asked for")
	singleRequests := requests.Load()

	multi, err := ex.GetOpenInterest(t.Context(), btc, eth)
	require.NoError(t, err, "GetOpenInterest must not error for several pairs")
	require.Len(t, multi, 2, "GetOpenInterest must report both pairs asked for")

	assert.Equal(t, 5000.0, single[0].OpenInterest, "a single pair should report the position sized in its quote currency")
	for _, oi := range multi {
		if oi.Key.Base == currency.BTC.Item {
			assert.Equal(t, single[0].OpenInterest, oi.OpenInterest, "the same contract should report the same open interest however many pairs are asked for")
		}
	}
	assert.Equal(t, int64(1), singleRequests, "a single pair should cost one request rather than a second for contract stats")
}

func TestContractStatUnmarshalLastFundingRate(t *testing.T) {
	t.Parallel()

	var stats []ContractStatResponse
	require.NoError(t, json.Unmarshal([]byte(`[{"last_funding_rate":"0.00125"}]`), &stats))
	require.Len(t, stats, 1)
	assert.Equal(t, 0.00125, stats[0].LastFundingRate.Float64())
}

func TestGetSupportedFlashSwapCurrencyPairsResponse(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "request method should be GET")
		assert.Equal(t, "/api/v4/flash_swap/currency_pairs", r.URL.Path, "request path should be flash swap currency pairs")
		assert.Equal(t, "BTC", r.URL.Query().Get("currency"), "currency query should match")
		assert.Equal(t, "10", r.URL.Query().Get("limit"), "limit query should match")
		assert.Equal(t, "2", r.URL.Query().Get("page"), "page query should match")
		_, err := fmt.Fprint(w, `[{"currency_pair":"BTC_USDT","sell_currency":"BTC","buy_currency":"USDT","sell_min_amount":"0.001","sell_max_amount":"1","buy_min_amount":"1","buy_max_amount":"100000"}]`)
		assert.NoError(t, err, "writing flash swap currency pairs should not error")
	}))

	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

	pairs, err := ex.GetSupportedFlashSwapCurrencyPairs(t.Context(), currency.BTC, 10, 2)
	require.NoError(t, err, "GetSupportedFlashSwapCurrencyPairs must not error")
	exp := []*FlashSwapCurrencyPairResponse{{
		CurrencyPair:  currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter),
		SellCurrency:  currency.BTC,
		BuyCurrency:   currency.USDT,
		SellMinAmount: 0.001,
		SellMaxAmount: 1,
		BuyMinAmount:  1,
		BuyMaxAmount:  100000,
	}}
	assert.Equal(t, exp, pairs, "GetSupportedFlashSwapCurrencyPairs should decode every field")
}

func TestGetCrossMarginMinimums(t *testing.T) {
	t.Parallel()

	minimums, err := e.getCrossMarginMinimums(t.Context())
	require.NoError(t, err, "getCrossMarginMinimums must not error")
	require.NotEmpty(t, minimums, "getCrossMarginMinimums must return loanable currencies")
	for ccy, minimum := range minimums {
		assert.Falsef(t, ccy.IsEmpty(), "currency should not be empty for minimum %f", minimum)
		assert.Positivef(t, minimum, "minimum should be positive for %s", ccy)
	}
}

func TestUpdateOrderExecutionLimitsUsesProductBorrowMinimums(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = "GateIOProductBorrowMinimums"

	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "request method should be GET")
		switch r.URL.Path {
		case "/api/v4/spot/currency_pairs":
			_, err := fmt.Fprint(w, `[{"id":"BTC_USDT","base":"BTC","quote":"USDT","min_base_amount":"0.001","min_quote_amount":"1","amount_precision":3,"precision":2,"trade_status":"tradable"}]`)
			assert.NoError(t, err, "writing spot currency pairs should not error")
		case "/api/v4/margin/uni/currency_pairs":
			_, err := fmt.Fprint(w, `[{"currency_pair":"BTC_USDT","base_min_borrow_amount":"0.01","quote_min_borrow_amount":"2","status":"enabled","delisted_time":0}]`)
			assert.NoError(t, err, "writing isolated margin lending markets should not error")
		case "/api/v4/margin/cross/currencies":
			_, err := fmt.Fprint(w, `[{"name":"BTC","min_borrow_amount":"0.03","loanable":true,"status":1},{"name":"USDT","min_borrow_amount":"4","loanable":true,"status":1}]`)
			assert.NoError(t, err, "writing cross-margin currencies should not error")
		default:
			http.NotFound(w, r)
		}
	}))

	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

	pair := currency.NewBTCUSDT()
	require.NoError(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Margin), "UpdateOrderExecutionLimits must not error for margin")
	isolatedLimits, err := ex.GetOrderExecutionLimits(asset.Margin, pair)
	require.NoError(t, err, "GetOrderExecutionLimits must not error for margin")
	assert.Equal(t, 0.01, isolatedLimits.MinimumBorrowAmountBase, "margin base borrow minimum should use the isolated pair value")
	assert.Equal(t, 2.0, isolatedLimits.MinimumBorrowAmountQuote, "margin quote borrow minimum should use the isolated pair value")

	require.NoError(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.CrossMargin), "UpdateOrderExecutionLimits must not error for cross margin")
	crossLimits, err := ex.GetOrderExecutionLimits(asset.CrossMargin, pair)
	require.NoError(t, err, "GetOrderExecutionLimits must not error for cross margin")
	assert.Equal(t, 0.03, crossLimits.MinimumBorrowAmountBase, "cross-margin base borrow minimum should use the currency value")
	assert.Equal(t, 4.0, crossLimits.MinimumBorrowAmountQuote, "cross-margin quote borrow minimum should use the currency value")
}

func TestFetchTradablePairsUsesMarginProductSources(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = "GateIOTradableMarginPairs"

	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "request method should be GET")
		switch r.URL.Path {
		case "/api/v4/spot/currency_pairs":
			_, err := fmt.Fprint(w, `[{"id":"BTC_USDT","base":"BTC","quote":"USDT","trade_status":"tradable"},{"id":"ETH_USDT","base":"ETH","quote":"USDT","trade_status":"tradable"},{"id":"DOGE_USDT","base":"DOGE","quote":"USDT","trade_status":"untradable"}]`)
			assert.NoError(t, err, "writing spot currency pairs should not error")
		case "/api/v4/margin/uni/currency_pairs":
			_, err := fmt.Fprint(w, `[{"currency_pair":"BTC_USDT","base_min_borrow_amount":"0.01","status":"enabled","delisted_time":0},{"currency_pair":"ETH_USDT","base_min_borrow_amount":"0.02","status":"enabled","delisted_time":0},{"currency_pair":"DOGE_USDT","base_min_borrow_amount":"1","status":"disabled","delisted_time":0},{"currency_pair":"SOL_USDT","base_min_borrow_amount":"0.1","status":"enabled","delisted_time":1700000000},{"currency_pair":"XRP_USDT","base_min_borrow_amount":"1","status":"enabled","delisted_time":4102444800}]`)
			assert.NoError(t, err, "writing isolated margin lending markets should not error")
		case "/api/v4/margin/cross/currencies":
			_, err := fmt.Fprint(w, `[{"name":"BTC","min_borrow_amount":"0.03","loanable":true,"status":1},{"name":"USDT","min_borrow_amount":"4","loanable":true,"status":1},{"name":"ETH","min_borrow_amount":"0.05","loanable":true,"status":0},{"name":"DOGE","min_borrow_amount":"1","loanable":true,"status":1}]`)
			assert.NoError(t, err, "writing cross-margin currencies should not error")
		default:
			http.NotFound(w, r)
		}
	}))

	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

	marginPairs, err := ex.FetchTradablePairs(t.Context(), asset.Margin)
	require.NoError(t, err, "FetchTradablePairs must not error for margin")
	require.Len(t, marginPairs, 3, "margin must return enabled pairs that are not yet delisted")
	assert.True(t, marginPairs[0].Equal(currency.NewBTCUSDT()), "margin should include the first isolated-margin pair")
	assert.True(t, marginPairs[1].Equal(currency.NewPair(currency.ETH, currency.USDT)), "margin should include the second isolated-margin pair")
	assert.True(t, marginPairs[2].Equal(currency.NewPair(currency.XRP, currency.USDT)), "margin should include a pair with a future delisting time")

	crossPairs, err := ex.FetchTradablePairs(t.Context(), asset.CrossMargin)
	require.NoError(t, err, "FetchTradablePairs must not error for cross margin")
	require.Len(t, crossPairs, 1, "cross margin must return one enabled tradable pair")
	assert.True(t, crossPairs[0].Equal(currency.NewBTCUSDT()), "cross margin should use enabled currencies and tradable spot pairs")
}

func TestGetRequestedOpenInterestPair(t *testing.T) {
	t.Parallel()

	pair := getPair(t, asset.DeliveryFutures)
	requested, err := getRequestedOpenInterestPair(e, []key.PairAsset{{
		Base:  pair.Base.Item,
		Quote: pair.Quote.Item,
		Asset: asset.DeliveryFutures,
	}}, asset.DeliveryFutures)
	require.NoError(t, err)
	assert.Equal(t, pair, requested)

	requested, err = getRequestedOpenInterestPair(e, []key.PairAsset{{
		Base:  pair.Base.Item,
		Quote: pair.Quote.Item,
		Asset: asset.DeliveryFutures,
	}}, asset.CoinMarginedFutures)
	require.NoError(t, err)
	assert.Equal(t, currency.EMPTYPAIR, requested)

	requested, err = getRequestedOpenInterestPair(e, []key.PairAsset{{Asset: asset.DeliveryFutures}, {Asset: asset.DeliveryFutures}}, asset.DeliveryFutures)
	require.NoError(t, err)
	assert.Equal(t, currency.EMPTYPAIR, requested)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	for _, a := range []asset.Item{asset.CoinMarginedFutures, asset.USDTMarginedFutures} {
		is, err := e.IsPerpetualFutureCurrency(a, currency.EMPTYPAIR)
		require.NoErrorf(t, err, "IsPerpetualFutureCurrency must not error for %s", a)
		assert.Truef(t, is, "%s should be a perpetual future currency", a)
	}
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.CrossMargin, asset.DeliveryFutures, asset.Options} {
		is, err := e.IsPerpetualFutureCurrency(a, currency.EMPTYPAIR)
		require.NoErrorf(t, err, "IsPerpetualFutureCurrency must not error for %s", a)
		assert.Falsef(t, is, "%s should not be a perpetual future currency", a)
	}
}

func TestMessageID(t *testing.T) {
	t.Parallel()
	id := e.MessageID()
	require.Len(t, id, 32, "message ID must be 32 characters long for usage as a request ID")
	got, err := uuid.Parse(id)
	require.NoError(t, err, "ID string must convert back to a UUID")
	require.Equal(t, byte(7), got[6]>>4, "message ID must be a UUID v7") // RFC 9562 version nibble
	require.Len(t, got.String(), 36, "UUID v7 string representation must be 36 characters long")
}

func TestPriceDivisor(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		asset  asset.Item
		pair   currency.Pair
		expect float64
		errIs  error
	}{
		{
			name:   "standard pair uses divisor 1",
			asset:  asset.Spot,
			pair:   currency.NewBTCUSDT(),
			expect: 1,
		},
		{
			name:   "special futures pair uses scaled divisor",
			asset:  asset.USDTMarginedFutures,
			pair:   currency.NewPair(divisorCurrency, currency.USDT),
			expect: 1e6,
		},
		{
			name:   "special delivery pair uses scaled divisor",
			asset:  asset.DeliveryFutures,
			pair:   currency.NewPair(divisorCurrency, currency.USDT),
			expect: 1e6,
		},
		{
			name:  "special non futures pair returns unsupported error",
			asset: asset.Spot,
			pair:  currency.NewPair(divisorCurrency, currency.USDT),
			errIs: currency.ErrCurrencyNotSupported,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := priceDivisor(tc.asset, tc.pair)
			if tc.errIs != nil {
				require.ErrorIs(t, err, tc.errIs)
				return
			}

			require.NoError(t, err, "priceDivisor must not error")
			assert.Equal(t, tc.expect, got, "price divisor should match expected value")
		})
	}
}

func TestEarliestTime(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	pastOldest := time.Unix(1_700_000_000, 0).UTC()
	pastNewer := pastOldest.Add(2 * time.Hour)
	future := now.Add(24 * time.Hour)

	for _, tc := range []struct {
		name   string
		times  []time.Time
		expect time.Time
	}{
		{
			name:   "no times returns zero",
			expect: time.Time{},
		},
		{
			name:   "zero and future times are ignored",
			times:  []time.Time{{}, future},
			expect: time.Time{},
		},
		{
			name:   "time equal to now is ignored",
			times:  []time.Time{now},
			expect: time.Time{},
		},
		{
			name:   "single past time is returned",
			times:  []time.Time{pastNewer},
			expect: pastNewer,
		},
		{
			name:   "oldest past time is returned",
			times:  []time.Time{future, pastNewer, {}, pastOldest},
			expect: pastOldest,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := earliestTime(now, tc.times...)
			assert.Equal(t, tc.expect, got, "earliest time should match expected value")
		})
	}
}

// 7610378	       143.3 ns/op	      48 B/op	       2 allocs/op
func BenchmarkMessageID(b *testing.B) {
	for b.Loop() {
		_ = e.MessageID()
	}
}

func TestFetchOrderbook(t *testing.T) {
	t.Parallel()

	ensureTestPairs(t)

	availMargin, err := e.GetAvailablePairs(asset.Margin)
	require.NoError(t, err, "GetAvailablePairs must not error")
	require.NotEmpty(t, availMargin, "margin pairs must not be empty")

	enabledMargin, err := e.GetEnabledPairs(asset.Margin)
	require.NoError(t, err, "GetEnabledPairs must not error")

	marginPair := availMargin[0]
	for _, candidate := range enabledMargin {
		if availMargin.Contains(candidate, true) {
			marginPair = candidate
			break
		}
	}

	availOptions, err := e.GetAvailablePairs(asset.Options)
	require.NoError(t, err, "GetAvailablePairs must not error")
	require.NotEmpty(t, availOptions, "options pairs must not be empty")

	enabledOptions, err := e.GetEnabledPairs(asset.Options)
	require.NoError(t, err, "GetEnabledPairs must not error")

	optionsPair := availOptions[0]
	for _, candidate := range enabledOptions {
		if availOptions.Contains(candidate, true) {
			optionsPair = candidate
			break
		}
	}

	availDelivery, err := e.GetAvailablePairs(asset.DeliveryFutures)
	require.NoError(t, err, "GetAvailablePairs must not error")

	deliveryPair, err := availDelivery.GetRandomPair()
	require.NoError(t, err, "GetRandomPair must not error")

	for _, tc := range []struct {
		pair currency.Pair
		a    asset.Item
		err  error
	}{
		{pair: currency.EMPTYPAIR, a: asset.Spot, err: currency.ErrCurrencyPairEmpty},
		{pair: marginPair, a: asset.Binary, err: asset.ErrNotSupported},
		{pair: currency.NewBTCUSDT(), a: asset.Spot},
		{pair: marginPair, a: asset.Margin},
		{pair: currency.NewBTCUSDT(), a: asset.USDTMarginedFutures},
		{pair: deliveryPair, a: asset.DeliveryFutures},
		{pair: optionsPair, a: asset.Options},
	} {
		t.Run(fmt.Sprintf("%s-%s: expected err:%v", tc.pair, tc.a, tc.err), func(t *testing.T) {
			t.Parallel()
			got, err := e.fetchOrderbook(t.Context(), tc.pair, tc.a, 1)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, e.Name, got.Exchange, "Exchange name should be correct")
			assert.True(t, tc.pair.Equal(got.Pair), "Pair should be correct")
			assert.Equal(t, tc.a, got.Asset, "Asset should be correct")
			assert.LessOrEqual(t, len(got.Asks), 1, "Asks count should not exceed limit, but may be empty especially for options")
			assert.LessOrEqual(t, len(got.Bids), 1, "Bids count should not exceed limit, but may be empty especially for options")
			if tc.a == asset.Options && len(got.Asks) == 0 && len(got.Bids) == 0 {
				t.Skip("GateIO may return an empty options order book without timestamp metadata")
			}
			assert.NotZero(t, got.LastUpdated, "Last updated timestamp should be set")
			assert.NotZero(t, got.LastUpdateID, "Last update ID should be set")
			assert.NotZero(t, got.LastPushed, "Last pushed timestamp should be set")
			assert.LessOrEqual(t, got.LastUpdated, got.LastPushed, "Last updated timestamp should be before last pushed timestamp")
		})
	}
}

func TestFetchOrderbookNoSpotInstrument(t *testing.T) {
	t.Parallel()

	fakePair := currency.NewPair(currency.NewCode("ZZFAKE"), currency.USDT)
	_, err := e.fetchOrderbook(t.Context(), fakePair, asset.Margin, 1)
	require.ErrorIs(t, err, errNoSpotInstrument)
}

func TestSetCrossMarginAccountBalances(t *testing.T) {
	t.Parallel()

	balances := accounts.CurrencyBalances{}
	setCrossMarginAccountBalances(&balances, nil)

	setCrossMarginAccountBalances(&balances, &CrossMarginAccountResponse{
		Balances: map[string]CrossMarginCurrencyBalance{
			"BTC": {
				Available: types.Number(2),
				Freeze:    types.Number(0.5),
				Borrowed:  types.Number(0.25),
				Interest:  types.Number(0.05),
			},
		},
	})

	got := balances[currency.BTC]
	assert.InDelta(t, 2.5, got.Total, 0.00000001, "total should include available and frozen balances")
	assert.InDelta(t, 0.5, got.Hold, 0.00000001, "hold should match frozen balance")
	assert.InDelta(t, 2, got.Free, 0.00000001, "free should match available balance")
	assert.InDelta(t, 0.3, got.Borrowed, 0.00000001, "borrowed should include principal and interest")
	assert.InDelta(t, 1.7, got.AvailableWithoutBorrow, 0.00000001, "available without borrow should subtract borrowed principal and interest")
}

func TestSetIsolatedMarginAccountBalances(t *testing.T) {
	t.Parallel()

	err := setIsolatedMarginAccountBalances(&accounts.CurrencyBalances{}, []MarginAccountItemResponse{{}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	err = setIsolatedMarginAccountBalances(&accounts.CurrencyBalances{}, []MarginAccountItemResponse{{
		Base: AccountBalanceInformation{Currency: currency.BTC},
	}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	err = setIsolatedMarginAccountBalances(&accounts.CurrencyBalances{}, []MarginAccountItemResponse{{
		AccountType: "inactive",
	}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	balances := accounts.CurrencyBalances{}
	err = setIsolatedMarginAccountBalances(&balances, []MarginAccountItemResponse{
		{
			Base: AccountBalanceInformation{
				Currency:     currency.BTC,
				Available:    types.Number(1),
				LockedAmount: types.Number(0.2),
				Borrowed:     types.Number(0.25),
			},
			Quote: AccountBalanceInformation{
				Currency:     currency.USDT,
				Available:    types.Number(10),
				LockedAmount: types.Number(2),
				Borrowed:     types.Number(2),
			},
		},
		{
			Base: AccountBalanceInformation{
				Currency:     currency.BTC,
				Available:    types.Number(3),
				LockedAmount: types.Number(0.4),
				Borrowed:     types.Number(0.5),
			},
			Quote: AccountBalanceInformation{
				Currency:     currency.ETH,
				Available:    types.Number(5),
				LockedAmount: types.Number(0.6),
			},
		},
		{
			Base: AccountBalanceInformation{
				Currency:     currency.ETH,
				Available:    types.Number(7),
				LockedAmount: types.Number(0.8),
				Borrowed:     types.Number(1),
			},
			Quote: AccountBalanceInformation{
				Currency:     currency.USDT,
				Available:    types.Number(20),
				LockedAmount: types.Number(4),
			},
		},
	})
	require.NoError(t, err, "setIsolatedMarginAccountBalances must add valid isolated margin balances")

	btc := balances[currency.BTC]
	assert.InDelta(t, 4.6, btc.Total, 0.00000001, "BTC total should include all isolated margin markets")
	assert.InDelta(t, 0.6, btc.Hold, 0.00000001, "BTC hold should include all isolated margin markets")
	assert.InDelta(t, 4, btc.Free, 0.00000001, "BTC free should include all isolated margin markets")
	assert.InDelta(t, 0.75, btc.Borrowed, 0.00000001, "BTC borrowed should include principal from all isolated margin markets")
	assert.InDelta(t, 3.25, btc.AvailableWithoutBorrow, 0.00000001, "BTC available without borrow should subtract borrowed principal")

	usdt := balances[currency.USDT]
	assert.InDelta(t, 36, usdt.Total, 0.00000001, "USDT total should include all isolated margin markets")
	assert.InDelta(t, 6, usdt.Hold, 0.00000001, "USDT hold should include all isolated margin markets")
	assert.InDelta(t, 30, usdt.Free, 0.00000001, "USDT free should include all isolated margin markets")
	assert.InDelta(t, 2, usdt.Borrowed, 0.00000001, "USDT borrowed should include principal from all isolated margin markets")
	assert.InDelta(t, 28, usdt.AvailableWithoutBorrow, 0.00000001, "USDT available without borrow should subtract borrowed principal")

	eth := balances[currency.ETH]
	assert.InDelta(t, 13.4, eth.Total, 0.00000001, "ETH total should include base and quote isolated margin entries")
	assert.InDelta(t, 1.4, eth.Hold, 0.00000001, "ETH hold should include base and quote isolated margin entries")
	assert.InDelta(t, 12, eth.Free, 0.00000001, "ETH free should include base and quote isolated margin entries")
	assert.InDelta(t, 1, eth.Borrowed, 0.00000001, "ETH borrowed should include principal from all isolated margin markets")
	assert.InDelta(t, 11, eth.AvailableWithoutBorrow, 0.00000001, "ETH available without borrow should subtract borrowed principal")
}

func TestAddIsolatedMarginAccountBalanceWithNegativeAvailable(t *testing.T) {
	t.Parallel()
	balances := accounts.CurrencyBalances{}
	err := addIsolatedMarginAccountBalance(&balances, AccountBalanceInformation{
		Currency:  currency.LRC,
		Available: types.Number(-0.01462404),
		Borrowed:  types.Number(4.85),
	})
	require.NoError(t, err, "addIsolatedMarginAccountBalance must add a valid isolated margin balance")

	lrc := balances[currency.LRC]
	assert.InDelta(t, -0.01462404, lrc.Total, 0.00000001, "total should preserve the exchange-reported negative available balance")
	assert.InDelta(t, -0.01462404, lrc.Free, 0.00000001, "free should preserve the exchange-reported negative available balance")
	assert.InDelta(t, 4.85, lrc.Borrowed, 0.00000001, "borrowed should include the outstanding principal")
	assert.InDelta(t, -4.86462404, lrc.AvailableWithoutBorrow, 0.00000001, "available without borrow should account for the outstanding principal")
}

// TestSetupUpgradesLegacyRESTEndpoints loads a config saved before the REST endpoints dropped their version prefix
// through the config loader, and checks the old defaults no longer override the endpoints GateIO now prefixes itself
func TestSetupUpgradesLegacyRESTEndpoints(t *testing.T) {
	t.Parallel()

	root, err := testutils.RootPathFromCWD()
	require.NoError(t, err, "RootPathFromCWD must not error")
	raw, err := os.ReadFile(filepath.Join(root, "testdata", "configtest.json"))
	require.NoError(t, err, "ReadFile must not error")
	var cfgMap map[string]any
	require.NoError(t, json.Unmarshal(raw, &cfgMap), "Unmarshal must not error")
	cfgMap["version"] = 17
	exchanges, ok := cfgMap["exchanges"].([]any)
	require.True(t, ok, "config must hold an exchange list")
	var found bool
	for _, x := range exchanges {
		exch, ok := x.(map[string]any)
		require.True(t, ok, "exchange config must be an object")
		if exch["name"] != "GateIO" {
			continue
		}
		api, ok := exch["api"].(map[string]any)
		require.True(t, ok, "GateIO api config must be an object")
		api["urlEndpoints"] = map[string]any{
			"RestSpotURL":              "https://api.gateio.ws/api/v4/",
			"RestFuturesURL":           "https://fx-api.gateio.ws/api/v4/",
			"RestSpotSupplementaryURL": "https://fx-api-testnet.gateio.ws",
			"WebsocketSpotURL":         "wss://api.gateio.ws/ws/v4/",
		}
		found = true
	}
	require.True(t, found, "config must hold a GateIO exchange")
	legacy, err := json.Marshal(cfgMap)
	require.NoError(t, err, "Marshal must not error")
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, legacy, 0o600), "WriteFile must not error")

	cfg := &config.Config{}
	require.NoError(t, cfg.LoadConfig(path, true), "LoadConfig must not error")
	exchCfg, err := cfg.GetExchangeConfig("GateIO")
	require.NoError(t, err, "GetExchangeConfig must not error")
	ex := new(Exchange)
	ex.SetDefaults()
	ex.Websocket = sharedtestvalues.NewTestWebsocket()
	require.NoError(t, ex.Setup(exchCfg), "Setup must not error")
	for ep, exp := range map[exchange.URL]string{
		exchange.RestSpot:              gateioTradeURL,
		exchange.RestFutures:           gateioFuturesLiveTradingAlternative,
		exchange.RestSpotSupplementary: gateioFuturesTestnetTrading,
	} {
		got, err := ex.API.Endpoints.GetURL(ep)
		require.NoErrorf(t, err, "GetURL must not error for %s", ep)
		assert.Equalf(t, exp, got, "%s should use the default endpoint after the upgrade", ep)
	}
}

func TestSpotOrderAsset(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		account string
		exp     asset.Item
		err     error
	}{
		{account: "", exp: asset.Spot},
		{account: "unified", exp: asset.Spot},
		{account: "spot", exp: asset.Spot},
		{account: "margin", exp: asset.Margin},
		{account: "cross_margin", exp: asset.CrossMargin},
		{account: "options", exp: asset.Options},
		{account: "bogus", err: asset.ErrNotSupported},
	} {
		a, err := spotOrderAsset(tc.account)
		require.ErrorIsf(t, err, tc.err, "spotOrderAsset must return the expected error for %q", tc.account)
		assert.Equalf(t, tc.exp, a, "spotOrderAsset should return the expected asset for %q", tc.account)
	}
}

func TestGetActiveOrdersUnifiedAccount(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	ex.SkipAuthCheck = true
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v4/spot/open_orders", r.URL.Path, "request path should be the open orders endpoint")
		assert.Empty(t, r.URL.Query().Get("account"), "request should list every account's open orders")
		labels := []string{"spot", "unified", "margin", "cross_margin", "", "bogus"}
		orders := make([]string, 0, len(labels))
		for id, account := range labels {
			orders = append(orders, fmt.Sprintf(`{"id":"%d","text":"t-%d","create_time_ms":1744150200123,"update_time_ms":1744150300123,"currency_pair":"BTC_USDT","status":"open","type":"limit","account":%q,"side":"sell","amount":"2","price":"6","left":"2","fee_currency":"USDT"}`, id, id, account))
		}
		_, err := fmt.Fprintf(w, `[{"currency_pair":"BTC_USDT","total":%d,"orders":[%s]}]`, len(orders), strings.Join(orders, ","))
		assert.NoError(t, err, "writing open orders should not error")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

	for a, exp := range map[asset.Item][]string{
		asset.Spot:        {"0", "1", "4"},
		asset.Margin:      {"2"},
		asset.CrossMargin: {"3"},
	} {
		result, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: a, Side: order.AnySide, Type: order.AnyType})
		require.NoErrorf(t, err, "GetActiveOrders must not error for %s", a)
		ids := make([]string, len(result))
		for i := range result {
			ids[i] = result[i].OrderID
			assert.Equalf(t, a, result[i].AssetType, "GetActiveOrders should report order %s as %s", result[i].OrderID, a)
		}
		assert.Equalf(t, exp, ids, "GetActiveOrders should report the %s orders, counting unified and unlabelled orders as spot", a)
	}
}

func TestSetDefaultsMarginWebsocketSupport(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	for _, a := range []asset.Item{asset.Margin, asset.CrossMargin} {
		assert.Falsef(t, ex.IsAssetWebsocketSupported(a), "SetDefaults should leave %s balances to REST", a)
	}
	assert.True(t, ex.IsAssetWebsocketSupported(asset.Spot), "SetDefaults should keep spot websocket support")

	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := ex.generateSubscriptionsSpot(t.Context())
	require.NoError(t, err, "generateSubscriptionsSpot must not error")
	require.NotEmpty(t, subs, "generateSubscriptionsSpot must generate subscriptions")
	for _, s := range subs {
		assert.NotContainsf(t, []string{marginBalancesChannel, crossMarginBalanceChannel}, s.Channel, "generateSubscriptionsSpot should not subscribe to %s", s.Channel)
	}
}

func TestFetchTradablePairsPerpetualContractTypes(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Each settlement lists a contract of the other margin type, which the asset must leave out
		var err error
		switch r.URL.Path {
		case "/api/v4/futures/usdt/contracts":
			_, err = fmt.Fprint(w, `[{"name":"BTC_USDT","type":"direct"},{"name":"ETH_USD","type":"inverse"}]`)
		case "/api/v4/futures/btc/contracts":
			_, err = fmt.Fprint(w, `[{"name":"BTC_USD","type":"inverse"},{"name":"ETH_USDT","type":"direct"}]`)
		default:
			http.NotFound(w, r)
		}
		assert.NoError(t, err, "writing contracts should not error")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

	for a, exp := range map[asset.Item]string{
		asset.USDTMarginedFutures: "BTC_USDT",
		asset.CoinMarginedFutures: "BTC_USD",
	} {
		pairs, err := ex.FetchTradablePairs(t.Context(), a)
		require.NoErrorf(t, err, "FetchTradablePairs must not error for %s", a)
		require.Lenf(t, pairs, 1, "FetchTradablePairs must return only %s's own contract type", a)
		assert.Equalf(t, exp, pairs[0].String(), "FetchTradablePairs should return the %s contract", a)
	}
}

func TestGetSpotOrderRequest(t *testing.T) {
	t.Parallel()
	pair := currency.NewBTCUSDT()
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		amount types.Number
		price  types.Number
		tif    string
		err    error
	}{
		{name: "spot market buy", submit: &order.Submit{AssetType: asset.Spot, Type: order.Market, Side: order.Buy, Amount: 0.001, QuoteAmount: 50}, amount: 50, tif: iocTIF},
		{name: "margin market buy", submit: &order.Submit{AssetType: asset.Margin, Type: order.Market, Side: order.Buy, Amount: 0.001, QuoteAmount: 50}, amount: 50, tif: iocTIF},
		{name: "cross margin market buy", submit: &order.Submit{AssetType: asset.CrossMargin, Type: order.Market, Side: order.Bid, QuoteAmount: 60}, amount: 60, tif: iocTIF},
		{name: "margin market buy without quote amount", submit: &order.Submit{AssetType: asset.Margin, Type: order.Market, Side: order.Buy, Amount: 0.001}, err: order.ErrAmountMustBeSet},
		{name: "margin market sell", submit: &order.Submit{AssetType: asset.Margin, Type: order.Market, Side: order.Sell, Amount: 0.001, QuoteAmount: 50}, amount: 0.001, tif: iocTIF},
		{name: "cross margin limit buy", submit: &order.Submit{AssetType: asset.CrossMargin, Type: order.Limit, Side: order.Buy, Price: 10000, Amount: 0.001, QuoteAmount: 50}, amount: 0.001, price: 10000, tif: gtcTIF},
		{name: "invalid side", submit: &order.Submit{AssetType: asset.Margin, Type: order.Market, Side: order.UnknownSide, QuoteAmount: 50}, err: order.ErrSideIsInvalid},
		{name: "unsupported time in force", submit: &order.Submit{AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Price: 1, Amount: 1, TimeInForce: order.GoodTillTime}, err: order.ErrUnsupportedTimeInForce},
		{name: "spot market sell with a price", submit: &order.Submit{AssetType: asset.Spot, Type: order.Market, Side: order.Sell, Price: 10000, Amount: 0.001}, amount: 0.001, tif: iocTIF},
		{name: "margin fill or kill market sell", submit: &order.Submit{AssetType: asset.Margin, Type: order.Market, Side: order.Sell, Price: 10000, Amount: 0.001, TimeInForce: order.FillOrKill}, amount: 0.001, tif: fokTIF},
		{name: "spot post only limit sell", submit: &order.Submit{AssetType: asset.Spot, Type: order.Limit, Side: order.Sell, Price: 10000, Amount: 0.001, TimeInForce: order.PostOnly}, amount: 0.001, price: 10000, tif: pocTIF},
		{name: "spot good till cancel market sell", submit: &order.Submit{AssetType: asset.Spot, Type: order.Market, Side: order.Sell, Price: 10000, Amount: 0.001, TimeInForce: order.GoodTillCancel}, err: order.ErrUnsupportedTimeInForce},
		{name: "spot post only market sell", submit: &order.Submit{AssetType: asset.Spot, Type: order.Market, Side: order.Sell, Amount: 0.001, TimeInForce: order.PostOnly}, err: order.ErrUnsupportedTimeInForce},
		{name: "spot stop limit sell", submit: &order.Submit{AssetType: asset.Spot, Type: order.StopLimit, Side: order.Sell, Price: 10000, TriggerPrice: 9900, Amount: 0.001}, err: order.ErrUnsupportedOrderType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.submit.Pair = pair
			req, err := e.getSpotOrderRequest(tc.submit)
			require.ErrorIs(t, err, tc.err, "getSpotOrderRequest must return the expected error")
			if tc.err != nil {
				return
			}
			assert.Equal(t, tc.amount, req.Amount, "getSpotOrderRequest should send the amount in the currency Gate sizes it by")
			assert.Equal(t, tc.submit.AssetType, req.Account, "getSpotOrderRequest should send the account")
			assert.Equal(t, tc.price, req.Price, "getSpotOrderRequest should send a price only for a limit order")
			assert.Equal(t, tc.tif, req.TimeInForce, "getSpotOrderRequest should send a time in force Gate accepts for the order type")
		})
	}
}

func TestSpotOrderType(t *testing.T) {
	t.Parallel()
	for in, exp := range map[string]order.Type{
		"limit":               order.Limit,
		"market":              order.Market,
		"limit_repay":         order.Limit,
		"limit_borrow":        order.Limit,
		"limit_borrow_repay":  order.Limit,
		"market_repay":        order.Market,
		"market_borrow":       order.Market,
		"market_borrow_repay": order.Market,
	} {
		got, err := spotOrderType(in)
		require.NoErrorf(t, err, "spotOrderType must not error for %q", in)
		assert.Equalf(t, exp, got, "spotOrderType should read %q as a %s order", in, exp)
	}
	_, err := spotOrderType("iceberg")
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "spotOrderType should reject an unknown type")
}

func TestOrderPriceAndTIF(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		orderType order.Type
		tif       order.TimeInForce
		price     float64
		exchTIF   string
		err       error
	}{
		{name: "limit", orderType: order.Limit, price: 3000, exchTIF: gtcTIF},
		{name: "post only limit", orderType: order.Limit, tif: order.PostOnly, price: 3000, exchTIF: pocTIF},
		{name: "market", orderType: order.Market, exchTIF: iocTIF},
		{name: "immediate or cancel market", orderType: order.Market, tif: order.ImmediateOrCancel, exchTIF: iocTIF},
		{name: "fill or kill market", orderType: order.Market, tif: order.FillOrKill, exchTIF: fokTIF},
		{name: "good till cancel market", orderType: order.Market, tif: order.GoodTillCancel, err: order.ErrUnsupportedTimeInForce},
		{name: "post only market", orderType: order.Market, tif: order.PostOnly, err: order.ErrUnsupportedTimeInForce},
		{name: "good till time limit", orderType: order.Limit, tif: order.GoodTillTime, err: order.ErrUnsupportedTimeInForce},
		{name: "stop limit", orderType: order.StopLimit, err: order.ErrUnsupportedOrderType},
		{name: "stop market", orderType: order.StopMarket, err: order.ErrUnsupportedOrderType},
		{name: "take profit", orderType: order.TakeProfit, err: order.ErrUnsupportedOrderType},
		{name: "trailing stop", orderType: order.TrailingStop, err: order.ErrUnsupportedOrderType},
	} {
		price, tif, err := orderPriceAndTIF(&order.Submit{Type: tc.orderType, TimeInForce: tc.tif, Price: 3000, TriggerPrice: 2900})
		require.ErrorIsf(t, err, tc.err, "orderPriceAndTIF must return the expected error for a %s order", tc.name)
		assert.Equalf(t, tc.price, price, "orderPriceAndTIF should return the price to send for a %s order", tc.name)
		assert.Equalf(t, tc.exchTIF, tif, "orderPriceAndTIF should return the time in force to send for a %s order", tc.name)
	}
}

func TestGetFuturesOrderRequest(t *testing.T) {
	t.Parallel()
	_, err := getFuturesOrderRequest(&order.Submit{Type: order.Limit, Side: order.UnknownSide, Price: 3000, Amount: 1})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "getFuturesOrderRequest must reject an invalid side")
	_, err = getFuturesOrderRequest(&order.Submit{Type: order.StopLimit, Side: order.Buy, Price: 3000, TriggerPrice: 2900, Amount: 1})
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType, "getFuturesOrderRequest must not send a stop limit as a plain limit order")

	req, err := getFuturesOrderRequest(&order.Submit{Pair: currency.NewBTCUSDT(), Type: order.Market, Side: order.Sell, Price: 3000, Amount: 2, ClientOrderID: "t-market"})
	require.NoError(t, err, "getFuturesOrderRequest must not error for a market order")
	exp := &FuturesOrderCreateRequest{Contract: currency.NewBTCUSDT(), Size: -2, TimeInForce: iocTIF, Text: "t-market"}
	assert.Equal(t, exp, req, "getFuturesOrderRequest should send a market order at price 0 with IOC")
}

func TestGetDeliveryOrderRequest(t *testing.T) {
	t.Parallel()
	_, err := getDeliveryOrderRequest(&order.Submit{Type: order.Limit, Side: order.UnknownSide, Price: 3000, Amount: 1})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "getDeliveryOrderRequest must reject an invalid side")
	_, err = getDeliveryOrderRequest(&order.Submit{Type: order.Limit, Side: order.Buy, Price: 3000, Amount: 1, TimeInForce: order.GoodTillTime})
	require.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "getDeliveryOrderRequest must reject an unsupported time in force")
	_, err = getDeliveryOrderRequest(&order.Submit{Type: order.StopLimit, Side: order.Buy, Price: 3000, TriggerPrice: 2900, Amount: 1})
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType, "getDeliveryOrderRequest must not send a stop limit as a plain limit order")

	req, err := getDeliveryOrderRequest(&order.Submit{Pair: currency.NewBTCUSDT(), Type: order.Limit, Side: order.Buy, Price: 3000, Amount: 2, ReduceOnly: true})
	require.NoError(t, err, "getDeliveryOrderRequest must not error for a limit order")
	exp := &DeliveryOrderCreateRequest{Contract: currency.NewBTCUSDT(), Size: 2, Price: 3000, ReduceOnly: true, TimeInForce: gtcTIF}
	assert.Equal(t, exp, req, "getDeliveryOrderRequest should send a limit order at its price with GTC")
}

func TestSetupWebsocketConnections(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	urls, err := ex.Websocket.GetConfiguredWebsocketURLs()
	require.NoError(t, err, "GetConfiguredWebsocketURLs must not error")
	// Gate lists no BTC-settled delivery contracts, so there is no BTC delivery connection
	assert.ElementsMatch(t, []string{gateioWebsocketEndpoint, usdtFuturesWebsocketURL, btcFuturesWebsocketURL, deliveryRealUSDTTradingURL, optionsWebsocketURL}, urls, "Setup should connect spot, both perpetual settlements, USDT delivery and options")
	deliveryURL, err := ex.API.Endpoints.GetURL(exchange.WebsocketFutures)
	require.NoError(t, err, "GetURL must not error")
	assert.Equal(t, deliveryRealUSDTTradingURL, deliveryURL, "SetDefaults should default the futures websocket endpoint to USDT delivery")
}

func TestDeriveSpotWebsocketOrderResponseAccount(t *testing.T) {
	t.Parallel()
	for account, exp := range map[string]asset.Item{
		"spot":         asset.Spot,
		"unified":      asset.Spot,
		"":             asset.Spot,
		"margin":       asset.Margin,
		"cross_margin": asset.CrossMargin,
	} {
		var resp *WebsocketOrderResponse
		require.NoErrorf(t, json.Unmarshal([]byte(`{"left":"0","amount":"0.0001","finish_as":"filled","time_in_force":"ioc","currency_pair":"BTC_USDT","type":"market","account":"`+account+`","side":"sell","text":"t-1","id":"766075454481","update_time_ms":1735720637188,"create_time_ms":1735720637188}`), &resp), "Unmarshal must decode account %q", account)
		got, err := e.deriveSpotWebsocketOrderResponse(resp)
		require.NoErrorf(t, err, "deriveSpotWebsocketOrderResponse must not error for account %q", account)
		assert.Equalf(t, exp, got.AssetType, "deriveSpotWebsocketOrderResponse should report account %q as %s", account, exp)
	}
	_, err := e.deriveSpotWebsocketOrderResponse(&WebsocketOrderResponse{Account: "bogus", Side: "sell", Type: "market", TimeInForce: "ioc", FinishAs: "filled"})
	assert.ErrorIs(t, err, asset.ErrNotSupported, "deriveSpotWebsocketOrderResponse should reject an unknown account")
	_, err = e.deriveSpotWebsocketOrderResponse(&WebsocketOrderResponse{Side: "bogus", Type: "market", TimeInForce: "ioc", FinishAs: "filled"})
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "deriveSpotWebsocketOrderResponse should reject an unknown side")
	_, err = e.deriveSpotWebsocketOrderResponse(&WebsocketOrderResponse{Side: "sell", Type: "market", TimeInForce: "ioc", FinishAs: "bogus"})
	assert.ErrorContains(t, err, "unrecognised order status", "deriveSpotWebsocketOrderResponse should reject an unknown finish")
	_, err = e.deriveSpotWebsocketOrderResponse(&WebsocketOrderResponse{Side: "sell", Type: "iceberg", TimeInForce: "ioc", FinishAs: "filled"})
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "deriveSpotWebsocketOrderResponse should reject an unknown order type")
	_, err = e.deriveSpotWebsocketOrderResponse(&WebsocketOrderResponse{Side: "sell", Type: "market", TimeInForce: "bogus", FinishAs: "filled"})
	assert.ErrorIs(t, err, order.ErrInvalidTimeInForce, "deriveSpotWebsocketOrderResponse should reject an unknown time in force")
}

func TestFetchTradablePairsErrors(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported, "FetchTradablePairs must reject an unsupported asset")

	failing := newRoutedExchange(t, nil)
	for _, a := range failing.GetAssetTypes(false) {
		_, err := failing.FetchTradablePairs(t.Context(), a)
		assert.ErrorContainsf(t, err, "SERVER_ERROR", "FetchTradablePairs should return a failed request for %s", a)
	}

	ex := newRoutedExchange(t, map[string]string{
		"/api/v4/spot/currency_pairs":     `[{"id":"BTC_USDT","base":"BTC","quote":"USDT","trade_status":"untradable"},{"id":"ETH_USDT","base":"ETH","quote":"USDT","trade_status":"tradable"}]`,
		"/api/v4/margin/cross/currencies": `[{"name":"BTC","min_borrow_amount":"1","loanable":true,"status":1}]`,
		"/api/v4/delivery/usdt/contracts": `[{"name":"BTC_USDT_20261225","in_delisting":true},{"name":""},{"name":"ETH_USDT_20261225"}]`,
		"/api/v4/options/underlyings":     `[{"name":"BTC_USDT"}]`,
	})
	pairs, err := ex.FetchTradablePairs(t.Context(), asset.Spot)
	require.NoError(t, err, "FetchTradablePairs must not error for spot")
	assert.Equal(t, currency.Pairs{currency.NewPair(currency.ETH, currency.USDT)}, pairs, "FetchTradablePairs should leave out untradable spot pairs")
	pairs, err = ex.FetchTradablePairs(t.Context(), asset.DeliveryFutures)
	require.NoError(t, err, "FetchTradablePairs must not error for delivery")
	assert.Equal(t, currency.Pairs{currency.NewPairWithDelimiter("ETH", "USDT_20261225", currency.UnderscoreDelimiter)}, pairs, "FetchTradablePairs should leave out delisting and unnamed delivery contracts")
	_, err = ex.FetchTradablePairs(t.Context(), asset.Options)
	assert.ErrorContains(t, err, "SERVER_ERROR", "FetchTradablePairs should return a failed contracts request")

	ex = newRoutedExchange(t, map[string]string{
		"/api/v4/margin/cross/currencies": `[{"name":"BTC","min_borrow_amount":"1","loanable":true,"status":1}]`,
		"/api/v4/options/underlyings":     `[{"name":"BTC_USDT"}]`,
		"/api/v4/options/contracts":       `[{"name":"_"}]`,
		"/api/v4/delivery/usdt/contracts": `[{"name":"_"}]`,
	})
	_, err = ex.FetchTradablePairs(t.Context(), asset.CrossMargin)
	assert.ErrorContains(t, err, "SERVER_ERROR", "FetchTradablePairs should return a failed spot pairs request for cross margin")
	_, err = ex.FetchTradablePairs(t.Context(), asset.Options)
	assert.ErrorIs(t, err, currency.ErrCreatingPair, "FetchTradablePairs should reject an invalid options contract")
	_, err = ex.FetchTradablePairs(t.Context(), asset.DeliveryFutures)
	assert.ErrorIs(t, err, currency.ErrCreatingPair, "FetchTradablePairs should reject an invalid delivery contract")
}

func TestGetActiveOrdersErrors(t *testing.T) {
	t.Parallel()
	failing := newRoutedExchange(t, nil)
	for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.DeliveryFutures, asset.Options} {
		_, err := failing.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: a, Side: order.AnySide, Type: order.AnyType})
		assert.ErrorContainsf(t, err, "SERVER_ERROR", "GetActiveOrders should return a failed request for %s", a)
	}

	ex := newRoutedExchange(t, map[string]string{
		"/api/v4/spot/open_orders":    `[{"currency_pair":"BTC_USDT","total":2,"orders":[{"id":"1","status":"open","account":"spot","side":"bogus","type":"limit"}]}]`,
		"/api/v4/futures/usdt/orders": `[{"id":3,"contract":"BTC_USDT","status":"open","finish_as":"","size":1,"tif":"bogus"}]`,
		"/api/v4/options/orders":      `[{"id":4,"contract":"_","status":"open","size":1,"tif":"gtc"}]`,
	})
	_, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "GetActiveOrders should reject an open spot order it cannot read")

	// A closed order the listing still carries is skipped before it is read
	closedOrders := newFixedResponseExchange(t, "/api/v4/spot/open_orders", `[{"currency_pair":"BTC_USDT","total":2,"orders":[{"id":"1","status":"closed","account":"spot","side":"bogus","type":"limit"},{"id":"2","status":"open","account":"spot","side":"buy","type":"limit","amount":"1","price":"2"}]}]`)
	result, err := closedOrders.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
	require.NoError(t, err, "GetActiveOrders must skip a closed order")
	require.Len(t, result, 1, "GetActiveOrders must report only the open order")
	assert.Equal(t, "2", result[0].OrderID, "GetActiveOrders should report the open order")
	_, err = ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.USDTMarginedFutures, Side: order.AnySide, Type: order.AnyType})
	assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "GetActiveOrders should reject a futures order it cannot read")
	_, err = ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Options, Side: order.AnySide, Type: order.AnyType})
	assert.ErrorIs(t, err, currency.ErrCreatingPair, "GetActiveOrders should reject an options order it cannot read")
}

func TestSubmitOrderErrors(t *testing.T) {
	t.Parallel()
	failing := newRoutedExchange(t, nil)
	for _, s := range []*order.Submit{
		{AssetType: asset.Spot, Pair: currency.NewBTCUSDT(), Type: order.Limit, Side: order.Buy, Price: 1, Amount: 1},
		{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), Type: order.Limit, Side: order.Buy, Price: 1, Amount: 1},
		{AssetType: asset.DeliveryFutures, Pair: getPair(t, asset.DeliveryFutures), Type: order.Limit, Side: order.Buy, Price: 1, Amount: 1},
		{AssetType: asset.Options, Pair: getPair(t, asset.Options), Type: order.Limit, Side: order.Buy, Price: 1, Amount: 1},
	} {
		s.Exchange = failing.Name
		_, err := failing.SubmitOrder(t.Context(), s)
		assert.ErrorContainsf(t, err, "SERVER_ERROR", "SubmitOrder should return a failed request for %s", s.AssetType)
	}

	// A side that is valid for submission but neither long nor short
	for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.DeliveryFutures, asset.Options} {
		_, err := failing.SubmitOrder(t.Context(), &order.Submit{Exchange: failing.Name, AssetType: a, Pair: getPair(t, a), Type: order.Limit, Side: order.Buy | order.Sell, Price: 1, Amount: 1})
		assert.ErrorIsf(t, err, order.ErrSideIsInvalid, "SubmitOrder should reject a side that is neither long nor short for %s", a)
	}
	_, err := failing.SubmitOrder(t.Context(), &order.Submit{Exchange: failing.Name, AssetType: asset.Options, Pair: getPair(t, asset.Options), Type: order.Limit, Side: order.Buy, Price: 1, Amount: 1, TimeInForce: order.GoodTillTime})
	assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "SubmitOrder should reject an options time in force Gate does not support")

	submit := func(ex *Exchange, a asset.Item) error {
		_, err := ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, AssetType: a, Pair: getPair(t, a), Type: order.Limit, Side: order.Buy, Price: 1, Amount: 1})
		return err
	}
	assert.ErrorIs(t, submit(newFixedResponseExchange(t, "/api/v4/spot/orders", `{"id":""}`), asset.Spot), order.ErrOrderIDNotSet, "SubmitOrder should reject a spot order without an ID")
	assert.ErrorIs(t, submit(newFixedResponseExchange(t, "/api/v4/spot/orders", `{"id":"1","side":"bogus"}`), asset.Spot), order.ErrSideIsInvalid, "SubmitOrder should reject a spot order with an unknown side")
	assert.ErrorContains(t, submit(newFixedResponseExchange(t, "/api/v4/spot/orders", `{"id":"1","side":"buy","status":"bogus"}`), asset.Spot), "unrecognised order status", "SubmitOrder should reject a spot order with an unknown status")
	assert.ErrorContains(t, submit(newFixedResponseExchange(t, "/api/v4/options/orders", `{"id":5,"status":"finished","finish_as":"bogus"}`), asset.Options), "unrecognised order status", "SubmitOrder should reject an options order with an unknown finish")
}

func TestSetupConfiguredWebsocketConnections(t *testing.T) {
	t.Parallel()
	// Each connection reads its URL from the configured endpoints and dials for its own asset, whatever the URL
	for _, tc := range []struct {
		a    asset.Item
		path string
	}{
		{a: asset.USDTMarginedFutures, path: "/custom/usdt"},
		{a: asset.CoinMarginedFutures, path: "/custom/btc"},
		{a: asset.DeliveryFutures, path: "/custom/delivery"},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			var dialled []string
			var sizeDecimal []string
			var mu sync.Mutex
			// A websocket dial needs a real listener, which NewTestServer does not provide
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				dialled = append(dialled, r.URL.Path)
				sizeDecimal = append(sizeDecimal, r.Header.Get(sizeDecimalHeader))
				mu.Unlock()
				w.WriteHeader(http.StatusServiceUnavailable)
			}))
			t.Cleanup(server.Close)
			wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

			cfg := &config.Config{}
			root, err := testutils.RootPathFromCWD()
			require.NoError(t, err, "RootPathFromCWD must not error")
			require.NoError(t, cfg.LoadConfig(filepath.Join(root, "testdata", "configtest.json"), true), "LoadConfig must not error")
			exchCfg, err := cfg.GetExchangeConfig("GateIO")
			require.NoError(t, err, "GetExchangeConfig must not error")
			exchCfg.Features.Enabled.Websocket = true
			exchCfg.API.AuthenticatedWebsocketSupport = false
			exchCfg.API.Endpoints = map[string]string{
				"WebsocketSpotURL":         wsURL + "/custom/spot",
				"WebsocketUSDTMarginedURL": wsURL + "/custom/usdt",
				"WebsocketCoinMarginedURL": wsURL + "/custom/btc",
				"WebsocketFuturesURL":      wsURL + "/custom/delivery",
				"WebsocketOptionsURL":      wsURL + "/custom/options",
			}
			for a, ps := range exchCfg.CurrencyPairs.Pairs {
				ps.AssetEnabled = a == tc.a
			}
			ex := new(Exchange)
			ex.SetDefaults()
			ex.Websocket = sharedtestvalues.NewTestWebsocket()
			require.NoError(t, ex.Setup(exchCfg), "Setup must not error")

			require.Error(t, ex.Websocket.Connect(t.Context()), "Connect must fail against a server refusing the upgrade")
			mu.Lock()
			defer mu.Unlock()
			require.NotEmpty(t, dialled, "Connect must dial the enabled asset's connection")
			assert.Equal(t, tc.path, dialled[0], "Connect should dial the configured URL of the enabled asset's connection")
			if tc.a != asset.DeliveryFutures {
				assert.Equal(t, "1", sizeDecimal[0], "Connect should request decimal sizes on perpetual connections")
			}
		})
	}
}

func TestSetup(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	assert.Error(t, ex.Setup(nil), "Setup should reject a nil config")

	cfg := &config.Config{}
	root, err := testutils.RootPathFromCWD()
	require.NoError(t, err, "RootPathFromCWD must not error")
	require.NoError(t, cfg.LoadConfig(filepath.Join(root, "testdata", "configtest.json"), true), "LoadConfig must not error")
	exchCfg, err := cfg.GetExchangeConfig("GateIO")
	require.NoError(t, err, "GetExchangeConfig must not error")
	exchCfg.Enabled = false
	ex = new(Exchange)
	ex.SetDefaults()
	require.NoError(t, ex.Setup(exchCfg), "Setup must not error for a disabled exchange")
	assert.False(t, ex.IsEnabled(), "Setup should leave a disabled exchange disabled")
}

func TestSetupSandbox(t *testing.T) {
	t.Parallel()
	testnet := map[exchange.URL]string{
		exchange.RestSpot:              "https://api-testnet.gateapi.io",
		exchange.RestFutures:           "https://api-testnet.gateapi.io",
		exchange.WebsocketSpot:         "wss://ws-testnet.gate.com/v4/ws/spot",
		exchange.WebsocketUSDTMargined: "wss://ws-testnet.gate.com/v4/ws/futures/usdt",
		exchange.WebsocketCoinMargined: "wss://fx-ws-testnet.gateio.ws/v4/ws/btc",
		exchange.WebsocketFutures:      "wss://fx-ws-testnet.gateio.ws/v4/ws/delivery/usdt",
		exchange.WebsocketOptions:      "wss://ws-testnet.gate.com/v4/ws/options",
		// Neither has a testnet form, so useSandbox leaves both as they are
		exchange.RestSpotSupplementary: "https://fx-api-testnet.gateio.ws",
		exchange.EdgeCase1:             "https://www.gate.com/apiw/v2/",
	}
	ex := new(Exchange)
	require.NoError(t, testexch.SetupWithConfig(ex, func(c *config.Exchange) { c.UseSandbox = true }), "SetupWithConfig must not error")
	for kind, exp := range testnet {
		got, err := ex.API.Endpoints.GetURL(kind)
		require.NoErrorf(t, err, "GetURL must not error for %s", kind)
		assert.Equalf(t, exp, got, "Setup should move %s to Gate's testnet when useSandbox is set", kind)
	}
	urls, err := ex.Websocket.GetConfiguredWebsocketURLs()
	require.NoError(t, err, "GetConfiguredWebsocketURLs must not error")
	assert.ElementsMatch(t, []string{
		testnet[exchange.WebsocketSpot], testnet[exchange.WebsocketUSDTMargined], testnet[exchange.WebsocketCoinMargined],
		testnet[exchange.WebsocketFutures], testnet[exchange.WebsocketOptions],
	}, urls, "Setup should connect every websocket to Gate's testnet when useSandbox is set")

	// An endpoint the config sets keeps its value, so a proxy or a hand-set testnet URL still works
	const proxy = "https://gate-proxy.example.com"
	ex = new(Exchange)
	require.NoError(t, testexch.SetupWithConfig(ex, func(c *config.Exchange) {
		c.UseSandbox = true
		c.API.Endpoints = map[string]string{exchange.RestSpot.String(): proxy}
	}), "SetupWithConfig must not error")
	got, err := ex.API.Endpoints.GetURL(exchange.RestSpot)
	require.NoError(t, err, "GetURL must not error")
	assert.Equal(t, proxy, got, "Setup should keep a configured endpoint when useSandbox is set")
	got, err = ex.API.Endpoints.GetURL(exchange.RestFutures)
	require.NoError(t, err, "GetURL must not error")
	assert.Equal(t, testnet[exchange.RestFutures], got, "Setup should still move the default endpoints when another is configured")

	// A configured endpoint that differs from the default only by a trailing slash is still the default
	ex = new(Exchange)
	require.NoError(t, testexch.SetupWithConfig(ex, func(c *config.Exchange) {
		c.UseSandbox = true
		c.API.Endpoints = map[string]string{exchange.RestSpot.String(): gateioTradeURL + "/", exchange.WebsocketSpot.String(): "wss://api.gateio.ws/ws/v4"}
	}), "SetupWithConfig must not error")
	for _, kind := range []exchange.URL{exchange.RestSpot, exchange.WebsocketSpot} {
		got, err = ex.API.Endpoints.GetURL(kind)
		require.NoErrorf(t, err, "GetURL must not error for %s", kind)
		assert.Equalf(t, testnet[kind], got, "Setup should move %s to the testnet when it differs from the default only by a trailing slash", kind)
	}

	ex = new(Exchange)
	require.NoError(t, testexch.SetupWithConfig(ex, func(c *config.Exchange) { c.UseSandbox = false }), "SetupWithConfig must not error")
	got, err = ex.API.Endpoints.GetURL(exchange.RestSpot)
	require.NoError(t, err, "GetURL must not error")
	assert.Equal(t, gateioTradeURL, got, "Setup should keep live trading endpoints when useSandbox is not set")
}
