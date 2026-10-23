package gateio

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
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
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testutils "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
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

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	require.NoError(t, testexch.MockHTTPInstance(e, ""), "MockHTTPInstance must not error")
	require.NoError(t, storeTestPairs(e), "storeTestPairs must not error")
	testexch.UpdatePairsOnce(t, e)
}

func TestGetAccountBalances(t *testing.T) {
	t.Parallel()
	ex := newIsolatedExchange(t)
	_, err := ex.UpdateAccountBalances(t.Context(), asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateAccountBalances must reject an unsupported asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
	}
	futuresBalance := accounts.CurrencyBalances{
		// Hold is what is not available, the float64 difference of the total and the available balance
		currency.USDT: {Currency: currency.USDT, Total: 9707.803567115145, Hold: 38.71218918100021, Free: 9669.091377934145},
	}
	for _, tc := range []struct {
		a   asset.Item
		exp accounts.CurrencyBalances
	}{
		{
			a: asset.Spot,
			exp: accounts.CurrencyBalances{
				currency.BTC:  {Currency: currency.BTC, Total: 969.3, Hold: 0.5, Free: 968.8},
				currency.USDT: {Currency: currency.USDT, Total: 2621, Hold: 120.75, Free: 2500.25},
			},
		},
		{
			a: asset.Margin,
			exp: accounts.CurrencyBalances{
				currency.BTC:  {Currency: currency.BTC, Total: 0.05, Free: 0.05, AvailableWithoutBorrow: 0.05},
				currency.USDT: {Currency: currency.USDT, Total: 1000, Free: 1000, AvailableWithoutBorrow: 1000},
			},
		},
		{
			// The interest owed counts as borrowed
			a: asset.CrossMargin,
			exp: accounts.CurrencyBalances{
				currency.BTC: {Currency: currency.BTC, Total: 0.6, Hold: 0.1, Free: 0.5, AvailableWithoutBorrow: 0.4799, Borrowed: 0.0201},
			},
		},
		{a: asset.USDTMarginedFutures, exp: futuresBalance},
		{
			a: asset.CoinMarginedFutures,
			exp: accounts.CurrencyBalances{
				currency.BTC: {Currency: currency.BTC, Total: 0.5, Hold: 0.04999999999999999, Free: 0.45},
			},
		},
		{a: asset.DeliveryFutures, exp: maps.Clone(futuresBalance)},
		{
			a: asset.Options,
			exp: accounts.CurrencyBalances{
				currency.USDT: {Currency: currency.USDT, Total: 1650.443022, Hold: 135.54148499999997, Free: 1514.901537},
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			started := time.Now()
			result, err := ex.UpdateAccountBalances(t.Context(), tc.a)
			require.NoError(t, err, "UpdateAccountBalances must not error")
			if !mockTests {
				assert.Len(t, result, 1, "UpdateAccountBalances should return the main account")
				return
			}
			require.Len(t, result, 1, "UpdateAccountBalances must return the main account")
			for c, b := range tc.exp {
				// Balances are stamped when they are saved
				assert.WithinRangef(t, result[0].Balances[c].UpdatedAt, started, time.Now(), "UpdateAccountBalances should stamp the %s balance", c)
				b.UpdatedAt = result[0].Balances[c].UpdatedAt
				tc.exp[c] = b
			}
			assert.Equal(t, accounts.SubAccounts{{AssetType: tc.a, Balances: tc.exp}}, result, "UpdateAccountBalances should return every balance")
		})
	}
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	ex := newIsolatedExchange(t)
	require.ErrorIs(t, ex.ValidateAPICredentials(t.Context(), asset.Binary), asset.ErrNotSupported, "ValidateAPICredentials must reject an unsupported asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
	}
	assert.NoError(t, ex.ValidateAPICredentials(t.Context(), asset.Spot), "ValidateAPICredentials should not error")
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCryptocurrencyFunds(t.Context(), nil)
	require.ErrorIs(t, err, withdraw.ErrRequestCannotBeNil)

	arg := &withdraw.Request{
		Description: "WITHDRAW IT ALL",
	}
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrExchangeNameNotSet)

	arg.Exchange = e.Name
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), arg)
	require.ErrorContains(t, err, withdraw.ErrStrAmountMustBeGreaterThanZero)

	arg.Amount = 1
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), arg)
	require.ErrorContains(t, err, withdraw.ErrStrNoCurrencySet)

	arg.Currency = currency.USD
	arg.Crypto = withdraw.CryptoRequest{
		Address: "bc1qk0jareu4jytc0cfrhr5wgshsq8282awp",
		Chain:   "",
	}
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), arg)
	require.ErrorContains(t, err, withdraw.ErrStrCurrencyNotCrypto)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		cryptocurrencyChains, err := e.GetAvailableTransferChains(t.Context(), currency.BTC)
		require.NoError(t, err, "GetAvailableTransferChains must not error")
		require.NotEmpty(t, cryptocurrencyChains, "GetAvailableTransferChains must return some chains")
		result, err := e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
			Exchange:    e.Name,
			Amount:      0.0001,
			Currency:    currency.BTC,
			Description: "WITHDRAW IT ALL",
			Type:        withdraw.Crypto,
			Crypto: withdraw.CryptoRequest{
				Address: core.BitcoinDonationAddress,
				Chain:   cryptocurrencyChains[0],
			},
		})
		require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error")
		assert.NotEmpty(t, result.ID, "WithdrawCryptocurrencyFunds should return the withdrawal ID")
		return
	}
	// The fixture only matches a request carrying the tag as Gate's memo and the client order ID
	result, err := e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange:      e.Name,
		Amount:        25,
		Currency:      currency.XRP,
		Description:   "WITHDRAW IT ALL",
		Type:          withdraw.Crypto,
		ClientOrderID: "gct-withdrawal-1",
		Crypto: withdraw.CryptoRequest{
			Address:    "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
			AddressTag: "100156837",
			Chain:      "XRP",
		},
	})
	require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error")
	// The record ID identifies the withdrawal, while the transaction hash is only known once it reaches the chain
	assert.Equal(t, &withdraw.ExchangeResponse{Name: e.Name, ID: "210496", Status: "DONE"}, result, "WithdrawCryptocurrencyFunds should return the withdrawal record")
}

func TestWithdrawFiatFunds(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFunds(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFunds should not be supported")
}

func TestWithdrawFiatFundsToInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFundsToInternationalBank should not be supported")
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	_, err := e.GetFeeByType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFeeByType must reject a nil fee builder")

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.SetCredentials(&accounts.Credentials{})
	feeBuilder := &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: currency.NewBTCUSDT(), PurchasePrice: 1000, Amount: 2}
	result, err := ex.GetFeeByType(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFeeByType must not error without credentials")
	assert.Equal(t, exchange.OfflineTradeFee, feeBuilder.FeeType, "GetFeeByType should estimate the trade fee offline without credentials")
	assert.Equal(t, 4.0, result, "GetFeeByType should return the worst case trade fee")

	result, err = ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyWithdrawalFee, Pair: currency.NewBTCUSDT()})
	require.NoError(t, err, "GetFeeByType must not error for a withdrawal fee")
	assert.Equal(t, 0.001, result, "GetFeeByType should return the BTC withdrawal fee")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	// The pair is given without Gate's delimiter, which the fixture only matches once it is formatted
	feeBuilder = &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: currency.NewBTCUSDT(), PurchasePrice: 1000, Amount: 2}
	result, err = e.GetFeeByType(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFeeByType must not error")
	assert.Equal(t, exchange.CryptocurrencyTradeFee, feeBuilder.FeeType, "GetFeeByType should fetch the trade fee with credentials")
	if mockTests {
		assert.Equal(t, 4.0, result, "GetFeeByType should return the account's taker fee")
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderInfo(t.Context(), "1234", currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOrderInfo must reject an empty pair")
	_, err = e.GetOrderInfo(t.Context(), "", currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOrderInfo must reject an empty order ID")
	_, err = e.GetOrderInfo(t.Context(), "1234", currency.NewBTCUSDT(), asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderInfo must reject an unsupported asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		a       asset.Item
		orderID string
		exp     *order.Detail
	}{
		{
			a:       asset.Spot,
			orderID: "1234",
			exp: &order.Detail{
				Exchange:             e.Name,
				OrderID:              "1234",
				ClientOrderID:        "t-abc123",
				Side:                 order.Buy,
				Type:                 order.Limit,
				Status:               order.Closed,
				Pair:                 getPair(t, asset.Spot),
				AssetType:            asset.Spot,
				Price:                65000,
				Amount:               0.001,
				ExecutedAmount:       0.001,
				AverageExecutedPrice: 63469.3,
				Cost:                 63.4693,
				CostAsset:            currency.USDT,
				Fee:                  0.00000022,
				FeeAsset:             currency.BTC,
				Date:                 time.UnixMilli(1744150000073),
				LastUpdated:          time.UnixMilli(1744150000074),
			},
		},
		{
			a:       asset.Margin,
			orderID: "1234",
			exp: &order.Detail{
				Exchange:             e.Name,
				OrderID:              "1234",
				ClientOrderID:        "t-margin-sell",
				Side:                 order.Sell,
				Type:                 order.Limit,
				Status:               order.Open,
				Pair:                 getPair(t, asset.Margin),
				AssetType:            asset.Margin,
				Price:                70000,
				Amount:               0.002,
				ExecutedAmount:       0.001,
				RemainingAmount:      0.001,
				AverageExecutedPrice: 70000,
				Cost:                 70,
				CostAsset:            currency.USDT,
				Fee:                  0.14,
				FeeAsset:             currency.USDT,
				Date:                 time.UnixMilli(1744150000173),
				LastUpdated:          time.UnixMilli(1744150100174),
			},
		},
		{
			a:       asset.CrossMargin,
			orderID: "1234",
			exp: &order.Detail{
				Exchange:        e.Name,
				OrderID:         "1234",
				Side:            order.Buy,
				Type:            order.Limit,
				Status:          order.Cancelled,
				Pair:            getPair(t, asset.CrossMargin),
				AssetType:       asset.CrossMargin,
				Price:           60000,
				Amount:          0.001,
				RemainingAmount: 0.001,
				CostAsset:       currency.USDT,
				FeeAsset:        currency.BTC,
				Date:            time.UnixMilli(1744150000273),
				LastUpdated:     time.UnixMilli(1744150300274),
			},
		},
		{
			// Gate finishes a partly filled IOC order as ioc, a status the order package has no name for
			a:       asset.USDTMarginedFutures,
			orderID: "15675402",
			exp: &order.Detail{
				Exchange:             e.Name,
				OrderID:              "15675402",
				ClientOrderID:        "t-usdt-market",
				Side:                 order.Short,
				Type:                 order.Market,
				Status:               order.Cancelled,
				TimeInForce:          order.ImmediateOrCancel,
				ReduceOnly:           true,
				Pair:                 getPair(t, asset.USDTMarginedFutures),
				AssetType:            asset.USDTMarginedFutures,
				SettlementCurrency:   currency.USDT,
				Amount:               10,
				ContractAmount:       10,
				ExecutedAmount:       6,
				RemainingAmount:      4,
				AverageExecutedPrice: 31500,
				Date:                 time.UnixMilli(1744150000123),
				LastUpdated:          time.UnixMilli(1744150000125),
			},
		},
		{
			a:       asset.CoinMarginedFutures,
			orderID: "15675394",
			exp: &order.Detail{
				Exchange:           e.Name,
				OrderID:            "15675394",
				ClientOrderID:      "t-my-custom-id",
				Side:               order.Long,
				Type:               order.Limit,
				Status:             order.Cancelled,
				TimeInForce:        order.GoodTillCancel,
				Pair:               getPair(t, asset.CoinMarginedFutures),
				AssetType:          asset.CoinMarginedFutures,
				SettlementCurrency: currency.BTC,
				Price:              3765,
				Amount:             6024,
				ContractAmount:     6024,
				RemainingAmount:    6024,
				Date:               time.Unix(1546569968, 0),
				LastUpdated:        time.UnixMilli(1546569970123),
			},
		},
		{
			a:       asset.DeliveryFutures,
			orderID: "123456",
			exp: &order.Detail{
				Exchange:           e.Name,
				OrderID:            "123456",
				ClientOrderID:      "t-my-custom-id",
				Side:               order.Long,
				Type:               order.Limit,
				Status:             order.Open,
				TimeInForce:        order.GoodTillCancel,
				Pair:               getPair(t, asset.DeliveryFutures),
				AssetType:          asset.DeliveryFutures,
				SettlementCurrency: currency.USDT,
				Price:              3765,
				Amount:             6024,
				ContractAmount:     6024,
				RemainingAmount:    6024,
				Date:               time.Unix(1546569968, 0),
				LastUpdated:        time.Unix(1546569968, 0),
			},
		},
		{
			// Gate sizes a sell negative
			a:       asset.Options,
			orderID: "1234",
			exp: &order.Detail{
				Exchange:        e.Name,
				OrderID:         "1234",
				Side:            order.Short,
				Type:            order.Limit,
				Status:          order.Open,
				TimeInForce:     order.GoodTillCancel,
				Pair:            getPair(t, asset.Options),
				AssetType:       asset.Options,
				Price:           100,
				Amount:          1,
				RemainingAmount: 1,
				Date:            time.Unix(1631763361, 0),
				LastUpdated:     time.Unix(1631763361, 0),
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			got, err := e.GetOrderInfo(t.Context(), tc.orderID, getPair(t, tc.a), tc.a)
			require.NoError(t, err, "GetOrderInfo must not error")
			if mockTests {
				assert.Equal(t, tc.exp, got, "GetOrderInfo should return the order")
				return
			}
			assert.Equal(t, tc.orderID, got.OrderID, "GetOrderInfo should return the requested order")
		})
	}
}

var mockedTickerTestCases = []struct {
	name         string
	asset        asset.Item
	pair         currency.Pair
	expectedPath string
}{
	{
		name:         "mocked USDT margined futures mark and index prices",
		asset:        asset.USDTMarginedFutures,
		pair:         currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter),
		expectedPath: "/api/v4/futures/usdt/tickers",
	},
	{
		name:         "mocked coin margined futures mark and index prices",
		asset:        asset.CoinMarginedFutures,
		pair:         currency.NewPairWithDelimiter("BTC", "USD", currency.UnderscoreDelimiter),
		expectedPath: "/api/v4/futures/btc/tickers",
	},
	{
		name:         "mocked delivery futures mark and index prices",
		asset:        asset.DeliveryFutures,
		pair:         currency.NewPairWithDelimiter("BTC", "USDT_20261225", currency.UnderscoreDelimiter),
		expectedPath: "/api/v4/delivery/usdt/tickers",
	},
	{
		name:         "mocked options mark and index prices",
		asset:        asset.Options,
		pair:         currency.NewPairWithDelimiter("BTC", "USDT-20261225-100000-C", currency.UnderscoreDelimiter),
		expectedPath: "/api/v4/options/tickers",
	},
}

// TestNewTestExchange isn't parallel, as it switches the shared exchange's sandbox setting
func TestNewTestExchange(t *testing.T) {
	useSandbox := e.Config.UseSandbox
	t.Cleanup(func() { e.Config.UseSandbox = useSandbox })
	for _, sandbox := range []bool{true, false} {
		e.Config.UseSandbox = sandbox
		ex := newTestExchange(t)
		assert.Equalf(t, sandbox, ex.Config.UseSandbox, "newTestExchange should follow the shared exchange's sandbox setting of %t", sandbox)
		assert.Equalf(t, sandbox, newIsolatedExchange(t).Config.UseSandbox, "newIsolatedExchange should follow the shared exchange's sandbox setting of %t", sandbox)
		for _, endpoint := range testnetEndpoints {
			exp := endpoint.production
			if sandbox {
				exp = endpoint.testnet
			}
			got, err := ex.API.Endpoints.GetURL(endpoint.kind)
			require.NoErrorf(t, err, "GetURL must not error for %s", endpoint.kind)
			assert.Equalf(t, exp, got, "newTestExchange should set %s for a sandbox setting of %t", endpoint.kind, sandbox)
		}
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), getPair(t, asset.Spot), asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateTicker must reject an unsupported asset")
	_, err = e.UpdateTicker(t.Context(), currency.NewPairWithDelimiter("BTC", "USDT-20260925-1-P", currency.UnderscoreDelimiter), asset.Options)
	require.ErrorIs(t, err, ticker.ErrTickerNotFound, "UpdateTicker must report an options contract its underlying's tickers do not list")
	_, err = newFixedResponseExchange(t, "/api/v4/futures/usdt/tickers", `[]`).UpdateTicker(t.Context(), currency.NewBTCUSDT(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, errNoTickerData, "UpdateTicker must reject an empty futures ticker list")

	t.Run("all supported assets", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			ex := newIsolatedExchange(t)
			btcusdtSpot := func() *ticker.Price {
				return &ticker.Price{Last: 67502.7, High: 67842.2, Low: 66610, Bid: 67502.6, Ask: 67502.7, BaseVolume: 4322.4404579762, QuoteVolume: 290264968.57010812722}
			}
			for _, tc := range []struct {
				a   asset.Item
				exp *ticker.Price
			}{
				{a: asset.Spot, exp: btcusdtSpot()},
				{a: asset.Margin, exp: btcusdtSpot()},
				{a: asset.CrossMargin, exp: btcusdtSpot()},
				{
					// The base volume is the contract count of volume_24h times the contract size, which beats the
					// truncated volume_24h_base
					a:   asset.USDTMarginedFutures,
					exp: &ticker.Price{Last: 67477.9, High: 67815.7, Low: 66580.1, BaseVolume: 32723.2988, QuoteVolume: 2196541426, MarkPrice: 67478, IndexPrice: 67503.92},
				},
				{
					a:   asset.CoinMarginedFutures,
					exp: &ticker.Price{Last: 67295.7, High: 67691.7, Low: 66490.4, BaseVolume: 14, QuoteVolume: 964612, MarkPrice: 67501.33, IndexPrice: 67499.05},
				},
				{
					a:   asset.DeliveryFutures,
					exp: &ticker.Price{Last: 67271.7, High: 68814.8, Low: 66355.1, BaseVolume: 7, QuoteVolume: 520353, MarkPrice: 67148.37, IndexPrice: 66878.24},
				},
				{
					a:   asset.Options,
					exp: &ticker.Price{Bid: 3408, BidSize: 455, Ask: 3752, AskSize: 2000, MarkPrice: 3597.4, IndexPrice: 67503.71},
				},
			} {
				t.Run(tc.a.String(), func(t *testing.T) {
					t.Parallel()
					started := time.Now()
					result, err := ex.UpdateTicker(t.Context(), getPair(t, tc.a), tc.a)
					require.NoError(t, err, "UpdateTicker must not error")
					// Gate's tickers carry no time, so they are stamped when stored
					assert.WithinRange(t, result.LastUpdated, started, time.Now(), "UpdateTicker should stamp the ticker when storing it")
					tc.exp.LastUpdated = result.LastUpdated
					tc.exp.Pair = getPair(t, tc.a)
					tc.exp.ExchangeName = ex.Name
					tc.exp.AssetType = tc.a
					assert.Equal(t, tc.exp, result, "UpdateTicker should return every field of the ticker")
				})
			}
			return
		}
		for _, a := range e.GetAssetTypes(false) {
			t.Run(a.String(), func(t *testing.T) {
				t.Parallel()
				pair := getPair(t, a)
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				testexch.UpdatePairsOnce(t, ex)
				// Isolate the cache so another test cannot supply the ticker being checked.
				ex.Name = t.Name()
				got, err := ex.UpdateTicker(t.Context(), pair, a)
				require.NoError(t, err, "UpdateTicker must not error")
				require.NotNil(t, got, "live ticker must not be nil")
				switch a {
				case asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.DeliveryFutures, asset.Options:
					if a == asset.Options {
						// Out-of-the-money options can have a zero mark price near expiry.
						assert.GreaterOrEqual(t, got.MarkPrice, 0.0, "live options mark price should be non-negative")
					} else {
						assert.Positive(t, got.MarkPrice, "live ticker mark price should be positive")
					}
					assert.Positive(t, got.IndexPrice, "live ticker index price should be positive")
				}
			})
		}
	})

	for _, tc := range mockedTickerTestCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			ex.Name = t.Name()
			if tc.asset == asset.Options {
				require.NoError(t, ex.UpdatePairs(currency.Pairs{tc.pair}, tc.asset, true), "mocked options pair must be enabled")
			}

			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method, "ticker request method should be GET")
				assert.Equal(t, tc.expectedPath, r.URL.Path, "ticker request path should match the asset endpoint")
				if tc.asset == asset.Options {
					assert.Equal(t, "BTC_USDT", r.URL.Query().Get("underlying"), "options underlying should match")
					_, err := fmt.Fprintf(w, `[{"name":%q,"last_price":"118.4","mark_price":"118.35","index_price":"100000.25"}]`, tc.pair.String())
					assert.NoError(t, err, "mocked response should be written")
					return
				}
				assert.Equal(t, tc.pair.String(), r.URL.Query().Get("contract"), "ticker request contract should match the pair")
				_, err := fmt.Fprintf(w, `[{"contract":%q,"last":"118.4","low_24h":"99.2","high_24h":"132.5","volume_24h_base":"5526","volume_24h_quote":"1665006","mark_price":"118.35","index_price":"118.36"}]`, tc.pair.String())
				assert.NoError(t, err, "mocked ticker response should be written")
			}))

			require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

			got, err := ex.UpdateTicker(t.Context(), tc.pair, tc.asset)
			require.NoError(t, err, "mocked ticker retrieval must succeed")
			assert.Equal(t, 118.35, got.MarkPrice, "ticker mark price should match the mocked response")
			if tc.asset == asset.Options {
				assert.Equal(t, 100000.25, got.IndexPrice, "options index price should match")
			} else {
				assert.Equal(t, 118.36, got.IndexPrice, "ticker index price should match the mocked response")
			}
		})
	}

	if mockTests {
		return // The volume checks below pick the busiest live pairs, which the recorded tickers do not cover
	}
	// presence alone cannot tell base from quote, so compare against the source fields. Both are
	// picked by volume rather than positionally, since InEpsilon errors outright on a zero
	// expected value and an untraded pair would decide the test on liquidity, not on the mapping
	spotTicks, err := e.GetTickers(t.Context(), currency.EMPTYPAIR, "")
	require.NoError(t, err, "GetTickers must not error")
	require.NotEmpty(t, spotTicks, "GetTickers must return tickers")
	// a pair quoting near parity reports both volumes alike and cannot tell a correct mapping from
	// a swapped one, so those are filtered out before picking rather than asserted against
	spotTicks = slices.DeleteFunc(spotTicks, func(tk *TickerResponse) bool {
		return tk.BaseVolume.Float64() == 0 || tk.BaseVolume.Float64() == tk.QuoteVolume.Float64()
	})
	require.NotEmpty(t, spotTicks, "at least one spot pair must report base and quote volumes that differ")
	busiestSpot := slices.MaxFunc(spotTicks, func(a, b *TickerResponse) int {
		return cmp.Compare(a.QuoteVolume.Float64(), b.QuoteVolume.Float64())
	})
	p := busiestSpot.CurrencyPair
	tick, err := e.UpdateTicker(t.Context(), p, asset.Spot)
	require.NoError(t, err, "UpdateTicker must not error")
	assert.InEpsilonf(t, busiestSpot.BaseVolume.Float64(), tick.BaseVolume, 0.05, "UpdateTicker should take the spot base volume for %s from base_volume", p)
	assert.InEpsilonf(t, busiestSpot.QuoteVolume.Float64(), tick.QuoteVolume, 0.05, "UpdateTicker should take the spot quote volume for %s from quote_volume", p)

	futuresTicks, err := e.GetFuturesTickers(t.Context(), currency.USDT, currency.EMPTYPAIR)
	require.NoError(t, err, "GetFuturesTickers must not error")
	require.NotEmpty(t, futuresTicks, "GetFuturesTickers must return tickers")
	// a multiplier of 1 reports the contract count and the base volume identically, so those
	// contracts are filtered out before picking the busiest rather than asserted against
	futuresTicks = slices.DeleteFunc(futuresTicks, func(tk *FuturesTickerResponse) bool {
		return tk.Volume24HourBase.Float64() == 0 || tk.Volume24HourBase.Float64() == tk.Volume24Hour.Float64()
	})
	require.NotEmpty(t, futuresTicks, "at least one futures contract must report a base volume that differs from its contract count")
	busiestFutures := slices.MaxFunc(futuresTicks, func(a, b *FuturesTickerResponse) int {
		return cmp.Compare(a.Volume24HourQuote.Float64(), b.Volume24HourQuote.Float64())
	})
	fp := busiestFutures.Contract
	futuresTick, err := e.UpdateTicker(t.Context(), fp, asset.USDTMarginedFutures)
	require.NoError(t, err, "UpdateTicker must not error")
	assert.InEpsilonf(t, busiestFutures.Volume24HourBase.Float64(), futuresTick.BaseVolume, 0.05,
		"UpdateTicker should report a base volume for %s of the right order, not the contract count in volume_24h", fp)
	assert.InEpsilonf(t, busiestFutures.Volume24HourQuote.Float64(), futuresTick.QuoteVolume, 0.05,
		"UpdateTicker should take the quote volume for %s from volume_24h_quote", fp)
}

func TestFuturesBaseVolume(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		tick FuturesTickerResponse
		exp  float64
	}{
		{"base truncated a whole unit off the product", FuturesTickerResponse{Volume24HourBase: 52438, Volume24Hour: 524387795, QuantoMultiplier: 0.0001}, 52438.7795},
		{"base truncated the contract away entirely", FuturesTickerResponse{Volume24Hour: 5128, QuantoMultiplier: 0.0001}, 0.5128},
		{"decimal lots: volume_24h dropped a part contract, so base is the better bound", FuturesTickerResponse{Volume24HourBase: 8819700, Volume24Hour: 8819, QuantoMultiplier: 1000}, 8819700},
		{"no contract size, as coin margined and delivery report it", FuturesTickerResponse{Volume24HourBase: 30, Volume24Hour: 2352026}, 30},
		{"nothing traded", FuturesTickerResponse{}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.InDelta(t, tc.exp, futuresBaseVolume(&tc.tick), 1e-9, "futuresBaseVolume should return the 24h volume in the base currency")
		})
	}
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()

	err := e.SetLeverage(t.Context(), asset.Spot, getPair(t, asset.Spot), margin.Isolated, 1, order.UnknownSide)
	require.ErrorIs(t, err, asset.ErrNotSupported, "SetLeverage must reject spot")

	err = e.SetLeverage(t.Context(), asset.CoinMarginedFutures, getPair(t, asset.CoinMarginedFutures), margin.NoMargin, 1, order.UnknownSide)
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported, "SetLeverage must reject an unsupported margin type")

	err = e.SetLeverage(t.Context(), asset.CoinMarginedFutures, getPair(t, asset.CoinMarginedFutures), margin.Isolated, 0, order.UnknownSide)
	require.ErrorIs(t, err, errInvalidLeverage, "SetLeverage must reject a leverage of 0")

	err = e.SetLeverage(t.Context(), asset.CoinMarginedFutures, currency.EMPTYPAIR, margin.Isolated, 1, order.UnknownSide)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "SetLeverage must reject an empty pair")

	err = e.SetLeverage(t.Context(), asset.DeliveryFutures, getPair(t, asset.DeliveryFutures), margin.Isolated, 0, order.UnknownSide)
	require.ErrorIs(t, err, errInvalidLeverage, "SetLeverage must reject a delivery leverage of 0")

	err = e.SetLeverage(t.Context(), asset.DeliveryFutures, getPair(t, asset.DeliveryFutures), margin.Multi, 1, order.UnknownSide)
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported, "SetLeverage must reject cross margin for delivery")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		a        asset.Item
		mt       margin.Type
		leverage float64
	}{
		{a: asset.CoinMarginedFutures, mt: margin.Isolated, leverage: 1},
		{a: asset.USDTMarginedFutures, mt: margin.Multi, leverage: 5},
		{a: asset.DeliveryFutures, mt: margin.Isolated, leverage: 1},
	} {
		err = e.SetLeverage(t.Context(), tc.a, getPair(t, tc.a), tc.mt, tc.leverage, order.UnknownSide)
		assert.NoErrorf(t, err, "SetLeverage should not error for %s %s", tc.a, tc.mt)
	}
}

// TestSetLeverageRequests checks the leverage each margin type sends, which the mock fixtures cannot tell apart since
// they match a POST by its empty body
func TestSetLeverageRequests(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	require.NoError(t, storeTestPairs(ex), "storeTestPairs must not error")
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "test-api-key", Secret: "test-api-secret"})
	var mu sync.Mutex
	var gotPath string
	var gotQuery url.Values
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "leverage request method should be POST")
		mu.Lock()
		gotPath, gotQuery = r.URL.Path, r.URL.Query()
		mu.Unlock()
		_, err := fmt.Fprint(w, `{"contract":"BTC_USDT","leverage":"0"}`)
		assert.NoError(t, err, "writing the position should not error")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

	pairs := mockTestPairs()
	for _, tc := range []struct {
		a        asset.Item
		mt       margin.Type
		expPath  string
		expQuery url.Values
	}{
		// Isolated margin sets the leverage itself, and cross margin a zero leverage capped by the cross leverage limit
		{a: asset.USDTMarginedFutures, mt: margin.Isolated, expPath: "/api/v4/futures/usdt/positions/BTC_USDT/leverage", expQuery: url.Values{"leverage": {"5"}}},
		{a: asset.USDTMarginedFutures, mt: margin.Unset, expPath: "/api/v4/futures/usdt/positions/BTC_USDT/leverage", expQuery: url.Values{"leverage": {"5"}}},
		{a: asset.USDTMarginedFutures, mt: margin.Multi, expPath: "/api/v4/futures/usdt/positions/BTC_USDT/leverage", expQuery: url.Values{"leverage": {"0"}, "cross_leverage_limit": {"5"}}},
		{a: asset.CoinMarginedFutures, mt: margin.Multi, expPath: "/api/v4/futures/btc/positions/BTC_USD/leverage", expQuery: url.Values{"leverage": {"0"}, "cross_leverage_limit": {"5"}}},
		{a: asset.DeliveryFutures, mt: margin.Isolated, expPath: "/api/v4/delivery/usdt/positions/BTC_USDT_20260626/leverage", expQuery: url.Values{"leverage": {"5"}}},
	} {
		require.NoErrorf(t, ex.SetLeverage(t.Context(), tc.a, pairs[tc.a], tc.mt, 5, order.UnknownSide), "SetLeverage must not error for %s %s", tc.a, tc.mt)
		mu.Lock()
		assert.Equalf(t, tc.expPath, gotPath, "SetLeverage should set the %s %s leverage of the contract", tc.a, tc.mt)
		assert.Equalf(t, tc.expQuery, gotQuery, "SetLeverage should send the %s %s leverage", tc.a, tc.mt)
		mu.Unlock()
	}
}

// Wrapper test functions

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.FetchTradablePairs(t.Context(), a)
		require.NoErrorf(t, err, "FetchTradablePairs must not error for %s", a)
		require.NotEmptyf(t, pairs, "FetchTradablePairs must return some pairs for %s", a)

		if a == asset.USDTMarginedFutures || a == asset.CoinMarginedFutures {
			for _, p := range pairs {
				_, err := getSettlementCurrency(p, a)
				require.NoErrorf(t, err, "Fetched pair %s %s must not error on getSettlementCurrency", a, p)
			}
		}
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.UpdateTickers(t.Context(), asset.Binary), asset.ErrNotSupported, "UpdateTickers must reject an unsupported asset")

	t.Run("all supported assets", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			ex := newIsolatedExchange(t)
			ratsSpot := func() *ticker.Price {
				return &ticker.Price{Last: 0.00003885, High: 0.00004023, Low: 0.00003663, Bid: 0.00003874, Ask: 0.00003879, BaseVolume: 5504390622, QuoteVolume: 209618.0648051, Pair: currency.NewPairWithDelimiter("RATS", "USDT", currency.UnderscoreDelimiter)}
			}
			// Every ticker Gate returns is stored, enabled or not, so the first of each recorded list is checked
			for _, tc := range []struct {
				a   asset.Item
				exp *ticker.Price
			}{
				{a: asset.Spot, exp: ratsSpot()},
				{a: asset.Margin, exp: ratsSpot()},
				{a: asset.CrossMargin, exp: ratsSpot()},
				{
					a:   asset.USDTMarginedFutures,
					exp: &ticker.Price{Last: 84.8, High: 85.24, Low: 84.59, BaseVolume: 37.19, QuoteVolume: 3157, MarkPrice: 84.8, IndexPrice: 85.049, Pair: currency.NewPairWithDelimiter("EWJ", "USDT", currency.UnderscoreDelimiter)},
				},
				{
					a:   asset.CoinMarginedFutures,
					exp: &ticker.Price{Last: 67295.7, High: 67691.7, Low: 66490.4, BaseVolume: 14, QuoteVolume: 964612, MarkPrice: 67501.33, IndexPrice: 67499.05, Pair: getPair(t, asset.CoinMarginedFutures)},
				},
				{
					a:   asset.DeliveryFutures,
					exp: &ticker.Price{Last: 0.091, High: 0.0933, Low: 0.0893, BaseVolume: 103971, QuoteVolume: 9461, MarkPrice: 0.091, IndexPrice: 0.0911, Pair: currency.NewPairWithDelimiter("ARB", "USDT_20260410", currency.UnderscoreDelimiter)},
				},
				{
					a:   asset.Options,
					exp: &ticker.Price{Bid: 3408, BidSize: 455, Ask: 3752, AskSize: 2000, MarkPrice: 3597.4, IndexPrice: 67503.71, Pair: getPair(t, asset.Options)},
				},
			} {
				t.Run(tc.a.String(), func(t *testing.T) {
					t.Parallel()
					started := time.Now()
					require.NoError(t, ex.UpdateTickers(t.Context(), tc.a), "UpdateTickers must not error")
					result, err := ticker.GetTicker(ex.Name, tc.exp.Pair, tc.a)
					require.NoError(t, err, "UpdateTickers must store the ticker")
					assert.WithinRange(t, result.LastUpdated, started, time.Now(), "UpdateTickers should stamp the ticker when storing it")
					tc.exp.LastUpdated = result.LastUpdated
					tc.exp.ExchangeName = ex.Name
					tc.exp.AssetType = tc.a
					assert.Equal(t, tc.exp, result, "UpdateTickers should store every field of the ticker")
				})
			}
			return
		}
		for _, a := range e.GetAssetTypes(false) {
			t.Run(a.String(), func(t *testing.T) {
				t.Parallel()
				switch a {
				case asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.DeliveryFutures, asset.Options:
					pair := getPair(t, a)
					ex := new(Exchange)
					require.NoError(t, testexch.Setup(ex), "Setup must not error")
					if a == asset.Options {
						testexch.UpdatePairsOnce(t, ex)
					}
					// Isolate the cache so another test cannot supply the ticker being checked.
					ex.Name = t.Name()
					require.NoError(t, ex.UpdateTickers(t.Context(), a), "live UpdateTickers must not error")
					got, err := ticker.GetTicker(ex.Name, pair, a)
					require.NoError(t, err, "live UpdateTickers must cache the requested ticker")
					require.NotNil(t, got, "live ticker must not be nil")
					if a == asset.Options {
						// Out-of-the-money options can have a zero mark price near expiry.
						assert.GreaterOrEqual(t, got.MarkPrice, 0.0, "live options mark price should be non-negative")
					} else {
						assert.Positive(t, got.MarkPrice, "live ticker mark price should be positive")
					}
					assert.Positive(t, got.IndexPrice, "live ticker index price should be positive")
				default:
					assert.NoError(t, e.UpdateTickers(t.Context(), a), "UpdateTickers should not error")
				}
			})
		}
	})

	for _, tc := range mockedTickerTestCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			ex.Name = t.Name()
			if tc.asset == asset.Options {
				require.NoError(t, ex.UpdatePairs(currency.Pairs{tc.pair}, tc.asset, true), "mocked options pair must be enabled")
			}

			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method, "ticker request method should be GET")
				assert.Equal(t, tc.expectedPath, r.URL.Path, "ticker request path should match the asset endpoint")
				if tc.asset == asset.Options {
					assert.Equal(t, "BTC_USDT", r.URL.Query().Get("underlying"), "options underlying should match")
					_, err := fmt.Fprintf(w, `[{"name":%q,"last_price":"118.4","mark_price":"118.35","index_price":"100000.25"}]`, tc.pair.String())
					assert.NoError(t, err, "mocked response should be written")
					return
				}
				assert.Empty(t, r.URL.Query().Get("contract"), "bulk ticker request should not filter by contract")
				_, err := fmt.Fprintf(w, `[{"contract":%q,"last":"118.4","low_24h":"99.2","high_24h":"132.5","volume_24h":"745487577","volume_24h_quote":"1665006","mark_price":"118.35","index_price":"118.36"}]`, tc.pair.String())
				assert.NoError(t, err, "mocked ticker response should be written")
			}))

			require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

			require.NoError(t, ex.UpdateTickers(t.Context(), tc.asset), "mocked bulk ticker update must succeed")
			got, err := ticker.GetTicker(ex.Name, tc.pair, tc.asset)
			require.NoError(t, err, "mocked ticker retrieval must succeed")
			assert.Equal(t, 118.35, got.MarkPrice, "ticker mark price should match the mocked response")
			if tc.asset == asset.Options {
				assert.Equal(t, 100000.25, got.IndexPrice, "options index price should match")
			} else {
				assert.Equal(t, 118.36, got.IndexPrice, "ticker index price should match the mocked response")
			}
		})
	}

	if mockTests {
		return // The base volume check below needs live tickers from an unmocked instance
	}
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
	ex.Name = t.Name()
	require.NoError(t, ex.UpdateTickers(t.Context(), asset.USDTMarginedFutures), "UpdateTickers must not error")

	// a multiplier of 1 reports both fields identically, and truncation to a whole number can
	// cost a full unit, so pick a contract that differs and carries enough base
	tickers, err := ex.GetFuturesTickers(t.Context(), currency.USDT, currency.EMPTYPAIR)
	require.NoError(t, err, "GetFuturesTickers must not error")
	require.NotEmpty(t, tickers, "GetFuturesTickers must return tickers")
	var checked bool
	for i := range tickers {
		tk := tickers[i]
		if tk.Volume24HourBase.Float64() < 1000 || tk.Volume24HourBase.Float64() == tk.Volume24Hour.Float64() {
			continue
		}
		p := tk.Contract
		stored, err := ticker.GetTicker(ex.Name, p, asset.USDTMarginedFutures)
		if err != nil {
			continue
		}
		assert.InEpsilonf(t, tk.Volume24HourBase.Float64(), stored.BaseVolume, 0.05,
			"UpdateTickers should report the base volume for %s, not the contract count in volume_24h", p)
		checked = true
		break
	}
	require.True(t, checked, "must find a futures contract whose base volume and contract count differ")
}

// TestUpdateTickersRequestsEachOptionsUnderlyingOnce checks the options tickers are requested once per underlying,
// since each request returns every contract of the underlying however many of them are enabled
func TestUpdateTickersRequestsEachOptionsUnderlyingOnce(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	put := currency.NewPairWithDelimiter("BTC", "USDT-20261225-100000-P", currency.UnderscoreDelimiter)
	call := currency.NewPairWithDelimiter("BTC", "USDT-20261225-100000-C", currency.UnderscoreDelimiter)
	require.NoError(t, ex.UpdatePairs(currency.Pairs{put, call}, asset.Options, true), "UpdatePairs must not error")

	var requests atomic.Int64
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		assert.Equal(t, "/api/v4/options/tickers", r.URL.Path, "request should reach the options tickers endpoint")
		assert.Equal(t, "BTC_USDT", r.URL.Query().Get("underlying"), "request should name the underlying")
		_, err := fmt.Fprintf(w, `[{"name":%q,"mark_price":"118.35","index_price":"100000.25"},{"name":%q,"mark_price":"7.5","index_price":"100000.25"}]`, put, call)
		assert.NoError(t, err, "writing the tickers should not error")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

	require.NoError(t, ex.UpdateTickers(t.Context(), asset.Options), "UpdateTickers must not error")
	assert.Equal(t, int64(1), requests.Load(), "UpdateTickers should request the tickers of an underlying once")
	for _, p := range []currency.Pair{put, call} {
		_, err := ticker.GetTicker(ex.Name, p, asset.Options)
		assert.NoErrorf(t, err, "UpdateTickers should store the %s ticker", p)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	ex := newIsolatedExchange(t)
	_, err := ex.UpdateOrderbook(t.Context(), currency.EMPTYPAIR, 1336)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateOrderbook must reject an empty pair")

	btcusdtSpot := func() *orderbook.Book {
		return &orderbook.Book{
			Bids:         orderbook.Levels{{Price: 67502.6, Amount: 1.705211}, {Price: 67502.4, Amount: 0.0005}, {Price: 67500.1, Amount: 0.013209}, {Price: 67500, Amount: 0.01}, {Price: 67499.9, Amount: 0.0005}},
			Asks:         orderbook.Levels{{Price: 67502.7, Amount: 0.586481}, {Price: 67506, Amount: 0.01}, {Price: 67507.4, Amount: 0.0005}, {Price: 67507.7, Amount: 0.014813}, {Price: 67507.9, Amount: 0.012953}},
			LastUpdated:  time.UnixMilli(1775423974345),
			LastPushed:   time.UnixMilli(1775423974347),
			LastUpdateID: 34890909828,
		}
	}
	for _, tc := range []struct {
		a   asset.Item
		exp *orderbook.Book
	}{
		{a: asset.Spot, exp: btcusdtSpot()},
		{a: asset.Margin, exp: btcusdtSpot()},
		{a: asset.CrossMargin, exp: btcusdtSpot()},
		{
			a: asset.USDTMarginedFutures,
			exp: &orderbook.Book{
				Bids:         orderbook.Levels{{Price: 67477.9, Amount: 19507}, {Price: 67477.5, Amount: 362}, {Price: 67477, Amount: 362}, {Price: 67476.9, Amount: 2}, {Price: 67476.5, Amount: 362}},
				Asks:         orderbook.Levels{{Price: 67478, Amount: 16139}, {Price: 67478.5, Amount: 362}, {Price: 67479, Amount: 362}, {Price: 67479.5, Amount: 362}, {Price: 67479.7, Amount: 222}},
				LastUpdated:  time.UnixMilli(1775423973949),
				LastPushed:   time.UnixMilli(1775423973950),
				LastUpdateID: 116388268171,
			},
		},
		{
			a: asset.CoinMarginedFutures,
			exp: &orderbook.Book{
				Bids:         orderbook.Levels{{Price: 1.17, Amount: 150}, {Price: 1.16, Amount: 203}},
				Asks:         orderbook.Levels{{Price: 1.52, Amount: 100}, {Price: 1.53, Amount: 40}},
				LastUpdated:  time.UnixMilli(1623898993121),
				LastPushed:   time.UnixMilli(1623898993123),
				LastUpdateID: 123456,
			},
		},
		{
			a: asset.DeliveryFutures,
			exp: &orderbook.Book{
				Bids:         orderbook.Levels{{Price: 67848.1, Amount: 1007}, {Price: 67847.2, Amount: 3}, {Price: 67837.9, Amount: 2706}, {Price: 67828.2, Amount: 3}, {Price: 67827.7, Amount: 2979}},
				Asks:         orderbook.Levels{{Price: 67863.8, Amount: 3}, {Price: 67868.6, Amount: 915}, {Price: 67876.1, Amount: 3}, {Price: 67878.8, Amount: 2714}, {Price: 67890, Amount: 3}},
				LastUpdated:  time.UnixMilli(1775423976249),
				LastPushed:   time.UnixMilli(1775423976701),
				LastUpdateID: 175093772,
			},
		},
		{
			a: asset.Options,
			exp: &orderbook.Book{
				Bids:         orderbook.Levels{{Price: 13588, Amount: 102}, {Price: 13487, Amount: 144}, {Price: 11639, Amount: 244}, {Price: 11611, Amount: 182}, {Price: 11584, Amount: 173}},
				Asks:         orderbook.Levels{{Price: 13676, Amount: 600}, {Price: 13817, Amount: 102}, {Price: 14577, Amount: 214}, {Price: 14605, Amount: 230}, {Price: 14632, Amount: 197}},
				LastUpdated:  time.UnixMilli(1774970470308),
				LastPushed:   time.UnixMilli(1774970474039),
				LastUpdateID: 2836984,
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			marketEx, pair := getMarketDataPair(t, ex, tc.a, hasTwoSidedDeliveryBook)
			started := time.Now()
			result, err := marketEx.UpdateOrderbook(t.Context(), pair, tc.a)
			require.NoErrorf(t, err, "UpdateOrderbook must not error for %s", pair)
			if !mockTests {
				if tc.a != asset.Options { // Options orderbooks can be empty
					assert.NotEmptyf(t, result.Bids, "UpdateOrderbook should return bids for %s", pair)
					assert.NotEmptyf(t, result.Asks, "UpdateOrderbook should return asks for %s", pair)
				}
				return
			}
			assert.WithinRange(t, result.InsertedAt, started, time.Now(), "UpdateOrderbook should stamp the book when storing it")
			tc.exp.InsertedAt = result.InsertedAt
			tc.exp.Exchange = ex.Name
			tc.exp.Pair = pair
			tc.exp.Asset = tc.a
			tc.exp.ValidateOrderbook = true
			tc.exp.RestSnapshot = true
			assert.Equal(t, tc.exp, result, "UpdateOrderbook should return every level of the book")
		})
	}
}

func TestUpdateOrderbookWithLimit(t *testing.T) {
	t.Parallel()
	ex := newIsolatedExchange(t)
	_, err := ex.UpdateOrderbookWithLimit(t.Context(), currency.EMPTYPAIR, asset.Spot, 20)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateOrderbookWithLimit must reject an empty pair")

	started := time.Now()
	result, err := ex.UpdateOrderbookWithLimit(t.Context(), getPair(t, asset.Spot), asset.Spot, 20)
	require.NoError(t, err, "UpdateOrderbookWithLimit must not error")
	if !mockTests {
		assert.LessOrEqual(t, len(result.Bids), 20, "UpdateOrderbookWithLimit should return no more bids than the limit")
		return
	}
	assert.WithinRange(t, result.InsertedAt, started, time.Now(), "UpdateOrderbookWithLimit should stamp the book when storing it")
	exp := &orderbook.Book{
		Bids:              orderbook.Levels{{Price: 67197, Amount: 0.442669}},
		Asks:              orderbook.Levels{{Price: 67197.1, Amount: 0.485259}},
		Exchange:          ex.Name,
		Pair:              getPair(t, asset.Spot),
		Asset:             asset.Spot,
		LastUpdated:       time.UnixMilli(1774968317359),
		LastPushed:        time.UnixMilli(1774968317360),
		InsertedAt:        result.InsertedAt,
		LastUpdateID:      27596272446,
		ValidateOrderbook: true,
		RestSnapshot:      true,
	}
	assert.Equal(t, exp, result, "UpdateOrderbookWithLimit should return the limited book")
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Empty)
	require.NoError(t, err, "GetWithdrawalsHistory must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetWithdrawalsHistory should return withdrawals")
		return
	}
	exp := []exchange.WithdrawalHistory{
		{
			Status:          "DONE",
			TransferID:      "w1879219868",
			Timestamp:       time.Unix(1744150000, 0),
			Currency:        "BTC",
			Amount:          0.5,
			Fee:             0.0001,
			TransferType:    "withdrawal",
			CryptoToAddress: "1HkxtBAMrA3tP5ENnYY2CZortjZvFDH5Cs",
			CryptoTxID:      "Internal transaction 260594131",
			CryptoChain:     "BTC",
		},
		{
			Status:          "CANCEL",
			TransferID:      "w1879219870",
			Description:     "Withdrawal address is not valid",
			Timestamp:       time.Unix(1744155000, 0),
			Currency:        "BTC",
			Amount:          0.25,
			Fee:             0.0002,
			TransferType:    "withdrawal",
			CryptoToAddress: "bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc",
			CryptoChain:     "BTC",
		},
	}
	assert.Equal(t, exp, result, "GetWithdrawalsHistory should return every withdrawal of the currency")
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountFundingHistory(t.Context())
	require.NoError(t, err, "GetAccountFundingHistory must not error")
	if !mockTests {
		assert.NotNil(t, result, "GetAccountFundingHistory should return records")
		return
	}
	// Deposits and withdrawals are merged in time order, a deposit first where both share a time
	exp := []exchange.FundingHistory{
		{
			ExchangeName:    e.Name,
			Status:          "BLOCKED",
			TransferID:      "d123456",
			Timestamp:       time.Unix(1744150000, 0),
			Currency:        "BTC",
			Amount:          10,
			TransferType:    "deposit",
			CryptoToAddress: "bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc",
			CryptoTxID:      "0xabcdef",
			CryptoChain:     "BTC",
		},
		{
			ExchangeName:    e.Name,
			Status:          "DONE",
			TransferID:      "w1879219868",
			Timestamp:       time.Unix(1744150000, 0),
			Currency:        "BTC",
			Amount:          0.5,
			Fee:             0.0001,
			TransferType:    "withdrawal",
			CryptoToAddress: "1HkxtBAMrA3tP5ENnYY2CZortjZvFDH5Cs",
			CryptoTxID:      "Internal transaction 260594131",
			CryptoChain:     "BTC",
		},
		{
			ExchangeName:    e.Name,
			Status:          "CANCEL",
			TransferID:      "w1879219870",
			Description:     "Withdrawal address is not valid",
			Timestamp:       time.Unix(1744155000, 0),
			Currency:        "BTC",
			Amount:          0.25,
			Fee:             0.0002,
			TransferType:    "withdrawal",
			CryptoToAddress: "bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc",
			CryptoChain:     "BTC",
		},
		{
			ExchangeName:    e.Name,
			Status:          "DONE",
			TransferID:      "210496",
			Timestamp:       time.Unix(1744160000, 0),
			Currency:        "BTC",
			Amount:          222.61,
			TransferType:    "deposit",
			CryptoToAddress: "1HkxtBAMrA3tP5ENnYY2CZortjZvFDH5Cs",
			CryptoTxID:      "128988928203223323290",
			CryptoChain:     "BTC",
		},
	}
	assert.Equal(t, exp, result, "GetAccountFundingHistory should return every deposit and withdrawal")
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTrades(t.Context(), getPair(t, asset.Spot), asset.Binary)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetRecentTrades must reject an unsupported asset")

	spotTrades := func() []trade.Data {
		return []trade.Data{
			{TID: "192741997", Side: order.Sell, Price: 67510, Amount: 0.002623, Timestamp: time.UnixMicro(1775423953817400)},
			{TID: "192741998", Side: order.Sell, Price: 67510, Amount: 0.059324, Timestamp: time.UnixMicro(1775423954285137)},
			{TID: "192741999", Side: order.Sell, Price: 67510, Amount: 0.001481, Timestamp: time.UnixMicro(1775423954285142)},
			{TID: "192742000", Side: order.Sell, Price: 67503.9, Amount: 0.000296, Timestamp: time.UnixMicro(1775423954292325)},
			{TID: "192742001", Side: order.Buy, Price: 67502.7, Amount: 0.100977, Timestamp: time.UnixMicro(1775423957197522)},
		}
	}
	for _, tc := range []struct {
		a   asset.Item
		exp []trade.Data
	}{
		{a: asset.Spot, exp: spotTrades()},
		{a: asset.Margin, exp: spotTrades()},
		{a: asset.CrossMargin, exp: spotTrades()},
		{
			// A negative size is a trade whose taker sold, and trades sharing a time keep Gate's order
			a: asset.USDTMarginedFutures,
			exp: []trade.Data{
				{TID: "724267225", Side: order.Buy, Price: 67478, Amount: 1467, Timestamp: time.UnixMilli(1775423969219)},
				{TID: "724267226", Side: order.Buy, Price: 67478, Amount: 105, Timestamp: time.UnixMilli(1775423969612)},
				{TID: "724267228", Side: order.Sell, Price: 67477.9, Amount: 19, Timestamp: time.UnixMilli(1775423970266)},
				{TID: "724267227", Side: order.Sell, Price: 67477.9, Amount: 1, Timestamp: time.UnixMilli(1775423970266)},
				{TID: "724267229", Side: order.Sell, Price: 67477.9, Amount: 4, Timestamp: time.UnixMilli(1775423970950)},
			},
		},
		{
			a: asset.CoinMarginedFutures,
			exp: []trade.Data{
				{TID: "49573344", Side: order.Buy, Price: 67438, Amount: 448, Timestamp: time.UnixMilli(1775423940134)},
				{TID: "49573345", Side: order.Sell, Price: 67296.7, Amount: 530, Timestamp: time.UnixMilli(1775423970135)},
			},
		},
		{
			a: asset.DeliveryFutures,
			exp: []trade.Data{
				{TID: "1017839", Side: order.Sell, Price: 67859.2, Amount: 10, Timestamp: time.UnixMilli(1775423940596)},
				{TID: "1017840", Side: order.Buy, Price: 67858.4, Amount: 1, Timestamp: time.UnixMilli(1775423948920)},
				{TID: "1017841", Side: order.Sell, Price: 67857.5, Amount: 5, Timestamp: time.UnixMilli(1775423954620)},
				{TID: "1017842", Side: order.Sell, Price: 67857.5, Amount: 3, Timestamp: time.UnixMilli(1775423959637)},
				{TID: "1017843", Side: order.Sell, Price: 67854, Amount: 1, Timestamp: time.UnixMilli(1775423963713)},
			},
		},
		{
			a: asset.Options,
			exp: []trade.Data{
				{TID: "1", Side: order.Sell, Price: 10460, Amount: 1, Timestamp: time.Unix(1773735690, 0)},
				{TID: "2", Side: order.Buy, Price: 12227, Amount: 1, Timestamp: time.Unix(1774074656, 0)},
				{TID: "3", Side: order.Sell, Price: 12988, Amount: 1, Timestamp: time.Unix(1775386206, 0)},
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			result, err := e.GetRecentTrades(t.Context(), getPair(t, tc.a), tc.a)
			require.NoError(t, err, "GetRecentTrades must not error")
			if !mockTests {
				for i := range result {
					assert.Positivef(t, result[i].Amount, "GetRecentTrades should return an unsigned amount for trade %d", i)
				}
				return
			}
			for i := range tc.exp {
				tc.exp[i].Exchange = e.Name
				tc.exp[i].CurrencyPair = getPair(t, tc.a)
				tc.exp[i].AssetType = tc.a
			}
			assert.Equal(t, tc.exp, result, "GetRecentTrades should return every trade")
		})
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricTrades(t.Context(), getPair(t, asset.Spot), asset.Spot, time.Now().Add(-time.Hour), time.Now())
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetHistoricTrades should not be supported")
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil, "SubmitOrder must reject a nil submission")
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		err    error
	}{
		{"unsupported asset", &order.Submit{AssetType: asset.Binary, Pair: currency.NewBTCUSDT(), Type: order.Limit, Side: order.Buy, Price: 1, Amount: 1}, asset.ErrNotSupported},
		{"futures good till time", &order.Submit{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), Type: order.Limit, Side: order.Buy, Price: 1, Amount: 1, TimeInForce: order.GoodTillTime}, order.ErrUnsupportedTimeInForce},
		{"USDT-margined futures settled in USD", &order.Submit{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSD(), Type: order.Limit, Side: order.Buy, Price: 1, Amount: 1}, errInvalidSettlementQuote},
		{"options fill or kill", &order.Submit{AssetType: asset.Options, Pair: currency.NewBTCUSDT(), Type: order.Limit, Side: order.Buy, Price: 1, Amount: 1, TimeInForce: order.FillOrKill}, order.ErrUnsupportedTimeInForce},
		{"options stop limit", &order.Submit{AssetType: asset.Options, Pair: currency.NewBTCUSDT(), Type: order.StopLimit, Side: order.Buy, Price: 1, TriggerPrice: 2, Amount: 1}, order.ErrUnsupportedOrderType},
		{"options good till cancel market", &order.Submit{AssetType: asset.Options, Pair: currency.NewBTCUSDT(), Type: order.Market, Side: order.Buy, Amount: 1, TimeInForce: order.GoodTillCancel}, order.ErrUnsupportedTimeInForce},
		{"futures stop limit", &order.Submit{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), Type: order.StopLimit, Side: order.Buy, Price: 1, TriggerPrice: 2, Amount: 1}, order.ErrUnsupportedOrderType},
		{"delivery stop limit", &order.Submit{AssetType: asset.DeliveryFutures, Pair: getPair(t, asset.DeliveryFutures), Type: order.StopLimit, Side: order.Buy, Price: 1, TriggerPrice: 2, Amount: 1}, order.ErrUnsupportedOrderType},
		{"margin market buy without quote amount", &order.Submit{AssetType: asset.Margin, Pair: currency.NewBTCUSDT(), Type: order.Market, Side: order.Buy, Amount: 1}, order.ErrAmountMustBeSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.submit.Exchange = e.Name
			_, err := e.SubmitOrder(t.Context(), tc.submit)
			assert.ErrorIs(t, err, tc.err, "SubmitOrder should reject the submission")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	// Prices are far from the market, so a live run leaves resting orders or, for IOC, unfilled orders
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		exp    *order.SubmitResponse
	}{
		{
			name:   "spot",
			submit: &order.Submit{AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Price: 10000, Amount: 0.001, ClientOrderID: "spot-limit"},
			exp: &order.SubmitResponse{
				Exchange:      e.Name,
				Type:          order.Limit,
				Side:          order.Buy,
				Pair:          getPair(t, asset.Spot),
				AssetType:     asset.Spot,
				Price:         10000,
				Amount:        0.001,
				ClientOrderID: "t-spot-limit",
				Date:          time.UnixMilli(1744150000073),
				LastUpdated:   time.UnixMilli(1744150000074),
				Status:        order.Open,
				OrderID:       "1852454425",
				FeeAsset:      currency.BTC,
			},
		},
		{
			name:   "margin",
			submit: &order.Submit{AssetType: asset.Margin, Type: order.Limit, Side: order.Sell, Price: 200000, Amount: 0.001, ClientOrderID: "margin-limit"},
			exp: &order.SubmitResponse{
				Exchange:      e.Name,
				Type:          order.Limit,
				Side:          order.Sell,
				Pair:          getPair(t, asset.Margin),
				AssetType:     asset.Margin,
				Price:         200000,
				Amount:        0.001,
				ClientOrderID: "t-margin-limit",
				Date:          time.UnixMilli(1744150000173),
				LastUpdated:   time.UnixMilli(1744150000174),
				Status:        order.Open,
				OrderID:       "1852454426",
				FeeAsset:      currency.USDT,
			},
		},
		{
			name:   "cross margin",
			submit: &order.Submit{AssetType: asset.CrossMargin, Type: order.Limit, Side: order.Buy, Price: 10000, Amount: 0.001, TimeInForce: order.PostOnly},
			exp: &order.SubmitResponse{
				Exchange:    e.Name,
				Type:        order.Limit,
				Side:        order.Buy,
				Pair:        getPair(t, asset.CrossMargin),
				AssetType:   asset.CrossMargin,
				TimeInForce: order.PostOnly,
				Price:       10000,
				Amount:      0.001,
				Date:        time.UnixMilli(1744150000273),
				LastUpdated: time.UnixMilli(1744150000274),
				Status:      order.Open,
				OrderID:     "1852454427",
				FeeAsset:    currency.BTC,
			},
		},
		{
			// Gate finishes an unfilled IOC order as ioc, a status the order package has no name for
			name:   "USDT-margined futures IOC finishing unfilled",
			submit: &order.Submit{AssetType: asset.USDTMarginedFutures, Type: order.Limit, Side: order.Buy, Price: 10000, Amount: 1, TimeInForce: order.ImmediateOrCancel, ClientOrderID: "usdt-ioc"},
			exp: &order.SubmitResponse{
				Exchange:        e.Name,
				Type:            order.Limit,
				Side:            order.Buy,
				Pair:            getPair(t, asset.USDTMarginedFutures),
				AssetType:       asset.USDTMarginedFutures,
				TimeInForce:     order.ImmediateOrCancel,
				Price:           10000,
				Amount:          1,
				RemainingAmount: 1,
				ClientOrderID:   "t-usdt-ioc",
				Date:            time.UnixMilli(1744150000123),
				LastUpdated:     time.UnixMilli(1744150000125),
				Status:          order.Cancelled,
				OrderID:         "15675401",
			},
		},
		{
			name:   "coin-margined futures",
			submit: &order.Submit{AssetType: asset.CoinMarginedFutures, Type: order.Limit, Side: order.Buy, Price: 3765, Amount: 6024, ClientOrderID: "my-custom-id"},
			exp: &order.SubmitResponse{
				Exchange:        e.Name,
				Type:            order.Limit,
				Side:            order.Buy,
				Pair:            getPair(t, asset.CoinMarginedFutures),
				AssetType:       asset.CoinMarginedFutures,
				Price:           3765,
				Amount:          6024,
				RemainingAmount: 6024,
				ClientOrderID:   "t-my-custom-id",
				Date:            time.Unix(1546569968, 0),
				LastUpdated:     time.UnixMilli(1546569970123),
				Status:          order.Cancelled,
				OrderID:         "15675394",
			},
		},
		{
			name:   "delivery futures",
			submit: &order.Submit{AssetType: asset.DeliveryFutures, Type: order.Limit, Side: order.Buy, Price: 3765, Amount: 6024, ClientOrderID: "my-custom-id"},
			exp: &order.SubmitResponse{
				Exchange:        e.Name,
				Type:            order.Limit,
				Side:            order.Buy,
				Pair:            getPair(t, asset.DeliveryFutures),
				AssetType:       asset.DeliveryFutures,
				Price:           3765,
				Amount:          6024,
				RemainingAmount: 6024,
				ClientOrderID:   "t-my-custom-id",
				Date:            time.Unix(1546569968, 0),
				LastUpdated:     time.Unix(1546569968, 0),
				Status:          order.Open,
				OrderID:         "15675394",
			},
		},
		{
			// Gate reports a finished options order's status as finished, which only its finish_as explains
			name:   "options IOC finishing unfilled",
			submit: &order.Submit{AssetType: asset.Options, Type: order.Limit, Side: order.Buy, Price: 1, Amount: 1, TimeInForce: order.ImmediateOrCancel, ClientOrderID: "options-ioc"},
			exp: &order.SubmitResponse{
				Exchange:        e.Name,
				Type:            order.Limit,
				Side:            order.Buy,
				Pair:            getPair(t, asset.Options),
				AssetType:       asset.Options,
				TimeInForce:     order.ImmediateOrCancel,
				Price:           1,
				Amount:          1,
				RemainingAmount: 1,
				ClientOrderID:   "t-options-ioc",
				Date:            time.Unix(1744150000, 0),
				LastUpdated:     time.Unix(1744150001, 0),
				Status:          order.Cancelled,
				OrderID:         "5",
			},
		},
		{
			// Gate sells options with a negative size
			name:   "options post only sell",
			submit: &order.Submit{AssetType: asset.Options, Type: order.Limit, Side: order.Sell, Price: 100, Amount: 1, TimeInForce: order.PostOnly, ClientOrderID: "options-sell"},
			exp: &order.SubmitResponse{
				Exchange:        e.Name,
				Type:            order.Limit,
				Side:            order.Sell,
				Pair:            getPair(t, asset.Options),
				AssetType:       asset.Options,
				TimeInForce:     order.PostOnly,
				Price:           100,
				Amount:          1,
				RemainingAmount: 1,
				ClientOrderID:   "t-options-sell",
				Date:            time.Unix(1744150000, 0),
				LastUpdated:     time.Unix(1744150000, 0),
				Status:          order.Open,
				OrderID:         "6",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.submit.Exchange = e.Name
			tc.submit.Pair = getPair(t, tc.submit.AssetType)
			got, err := e.SubmitOrder(t.Context(), tc.submit)
			require.NoError(t, err, "SubmitOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, got, "SubmitOrder should return the placed order")
				return
			}
			assert.NotEmpty(t, got.OrderID, "SubmitOrder should return the order ID")
		})
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyOrder(t.Context(), &order.Modify{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "ModifyOrder should report that it is not supported")
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.CancelOrder(t.Context(), nil), order.ErrCancelOrderIsNil, "CancelOrder must reject a nil cancellation")
	cancel := &order.Cancel{AssetType: asset.Spot, Pair: currency.NewBTCUSDT()}
	require.ErrorIs(t, e.CancelOrder(t.Context(), cancel), order.ErrOrderIDNotSet, "CancelOrder must reject a cancellation without an order ID")
	cancel.OrderID, cancel.Pair = "1234", currency.EMPTYPAIR
	require.ErrorIs(t, e.CancelOrder(t.Context(), cancel), currency.ErrCurrencyPairEmpty, "CancelOrder must reject a cancellation without a pair")
	cancel.Pair, cancel.AssetType = currency.NewBTCUSDT(), asset.Binary
	require.ErrorIs(t, e.CancelOrder(t.Context(), cancel), asset.ErrNotSupported, "CancelOrder must reject an unsupported asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for a, orderID := range map[asset.Item]string{
		asset.Spot:                "1234",
		asset.Margin:              "1234",
		asset.CrossMargin:         "1234",
		asset.USDTMarginedFutures: "15675402",
		asset.CoinMarginedFutures: "15675394",
		asset.DeliveryFutures:     "123456",
		asset.Options:             "1234",
	} {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			err := e.CancelOrder(t.Context(), &order.Cancel{OrderID: orderID, Pair: getPair(t, a), AssetType: a})
			assert.NoError(t, err, "CancelOrder should not error")
		})
	}
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelBatchOrders must reject an empty batch")
	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Binary, Pair: currency.NewBTCUSDT(), OrderID: "1"}})
	require.ErrorIs(t, err, asset.ErrNotSupported, "CancelBatchOrders must reject an unsupported asset")
	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{
		{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), OrderID: "1"},
		{AssetType: asset.DeliveryFutures, Pair: currency.NewBTCUSDT(), OrderID: "2"},
	})
	require.ErrorIs(t, err, errSingleAssetRequired, "CancelBatchOrders must reject a batch of several assets")
	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Spot, Pair: currency.NewBTCUSDT()}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelBatchOrders must reject an order without an ID")
	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Spot, OrderID: "1"}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelBatchOrders must reject an order without a pair")
	_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.USDTMarginedFutures, Pair: currency.NewBTCUSD(), OrderID: "1"}})
	require.ErrorIs(t, err, errInvalidSettlementQuote, "CancelBatchOrders must reject a pair the asset does not settle")

	for _, a := range []asset.Item{asset.DeliveryFutures, asset.Options} {
		_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: a, Pair: currency.NewBTCUSDT(), OrderID: "1"}})
		require.ErrorIsf(t, err, common.ErrFunctionNotSupported, "CancelBatchOrders must refuse %s orders, which Gate cannot cancel in a batch", a)
	}

	t.Run("one request", func(t *testing.T) {
		t.Parallel()
		// Gate cancels up to 20 orders in one request; a larger batch is refused before anything is sent
		var requests atomic.Int32
		ex := futuresLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			requests.Add(1)
			assert.Equal(t, "/api/v4/futures/usdt/batch_cancel_orders", r.URL.Path, "the orders should be cancelled by the batch endpoint")
			var ids []string
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&ids), "Decode should not error")
			results := make([]map[string]any, len(ids))
			for i, id := range ids {
				results[i] = map[string]any{"id": id, "succeeded": true}
			}
			assert.NoError(t, json.NewEncoder(w).Encode(results), "Encode should not error")
		})
		pair := currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
		cancels := make([]order.Cancel, 21)
		for i := range cancels {
			cancels[i] = order.Cancel{OrderID: strconv.Itoa(i + 1), Pair: pair, AssetType: asset.USDTMarginedFutures}
		}
		_, err := ex.CancelBatchOrders(t.Context(), cancels)
		require.ErrorIs(t, err, errTooManyOrders, "CancelBatchOrders must refuse more orders than one request takes")
		require.Zero(t, requests.Load(), "CancelBatchOrders must not send any request for a batch it refuses")

		got, err := ex.CancelBatchOrders(t.Context(), cancels[:20])
		require.NoError(t, err, "CancelBatchOrders must not error")
		assert.Equal(t, int32(1), requests.Load(), "CancelBatchOrders should cancel a full batch in one request")
		require.Len(t, got.Status, 20, "CancelBatchOrders must report every order's outcome")
		for _, c := range cancels[:20] {
			assert.Equalf(t, order.Cancelled.String(), got.Status[c.OrderID], "order %s should be reported cancelled", c.OrderID)
		}
	})

	t.Run("failed request", func(t *testing.T) {
		t.Parallel()
		ex := futuresLocalServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte(`{"label":"INVALID_PARAM_VALUE","message":"Invalid order id"}`))
			assert.NoError(t, err, "Write should not error")
		})
		pair := currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
		got, err := ex.CancelBatchOrders(t.Context(), []order.Cancel{
			{OrderID: "1", Pair: pair, AssetType: asset.USDTMarginedFutures},
			{OrderID: "2", Pair: pair, AssetType: asset.USDTMarginedFutures},
		})
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "CancelBatchOrders must return the failed request's error")
		require.NotNil(t, got, "CancelBatchOrders must report the outcomes alongside the error")
		for _, id := range []string{"1", "2"} {
			assert.Containsf(t, got.Status[id], "INVALID_PARAM_VALUE", "order %s should report the failed request's reason", id)
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	spotOutcomes := &order.CancelBatchResponse{Status: map[string]string{"1234567": order.Cancelled.String(), "123453": "ORDER_NOT_FOUND: Order not found"}}
	for _, tc := range []struct {
		a        asset.Item
		orderIDs []string
		exp      *order.CancelBatchResponse
	}{
		{a: asset.Spot, orderIDs: []string{"1234567", "123453"}, exp: spotOutcomes},
		{a: asset.Margin, orderIDs: []string{"1234567", "123453"}, exp: spotOutcomes},
		{a: asset.CrossMargin, orderIDs: []string{"1234567", "123453"}, exp: spotOutcomes},
		{a: asset.USDTMarginedFutures, orderIDs: []string{"1", "2"}, exp: &order.CancelBatchResponse{Status: map[string]string{"1": order.Cancelled.String(), "2": "ORDER_NOT_FOUND"}}},
		{a: asset.CoinMarginedFutures, orderIDs: []string{"15675394"}, exp: &order.CancelBatchResponse{Status: map[string]string{"15675394": order.Cancelled.String()}}},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			cancels := make([]order.Cancel, len(tc.orderIDs))
			for i, id := range tc.orderIDs {
				cancels[i] = order.Cancel{OrderID: id, Pair: getPair(t, tc.a), AssetType: tc.a}
			}
			got, err := e.CancelBatchOrders(t.Context(), cancels)
			require.NoError(t, err, "CancelBatchOrders must not error")
			if mockTests {
				assert.Equal(t, tc.exp, got, "CancelBatchOrders should report each order's own outcome")
				return
			}
			assert.Len(t, got.Status, len(tc.orderIDs), "CancelBatchOrders should report each order's outcome")
		})
	}
}

func TestRecordCancelOutcomes(t *testing.T) {
	t.Parallel()
	requested := []order.Cancel{{OrderID: "1"}, {OrderID: "t-custom"}, {OrderID: "3"}, {OrderID: "4"}}
	resp := &order.CancelBatchResponse{Status: map[string]string{}}
	err := recordCancelOutcomes(resp, requested, []cancelOutcome{
		{orderID: "1", succeeded: true},
		// Gate reports the order ID of an order cancelled by its custom text
		{orderID: "2", text: "t-custom", succeeded: true},
		{orderID: "3", reason: "ORDER_NOT_FOUND: Order not found"},
	}, nil)
	require.ErrorIs(t, err, errCancelOutcomeMissing, "recordCancelOutcomes must report an order Gate returned no outcome for")
	exp := map[string]string{
		"1":        order.Cancelled.String(),
		"t-custom": order.Cancelled.String(),
		"3":        "ORDER_NOT_FOUND: Order not found",
		"4":        errCancelOutcomeMissing.Error(),
	}
	assert.Equal(t, exp, resp.Status, "recordCancelOutcomes should report each requested order's outcome")

	resp = &order.CancelBatchResponse{Status: map[string]string{}}
	err = recordCancelOutcomes(resp, requested[:2], nil, common.ErrNoResponse)
	require.ErrorIs(t, err, common.ErrNoResponse, "recordCancelOutcomes must return a failed request's error")
	assert.Equal(t, map[string]string{"1": common.ErrNoResponse.Error(), "t-custom": common.ErrNoResponse.Error()}, resp.Status, "recordCancelOutcomes should report a failed request against every order it carried")
}

func TestJoinNonEmpty(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "ORDER_NOT_FOUND: Order not found", joinNonEmpty("ORDER_NOT_FOUND", "Order not found"), "joinNonEmpty should join a label and a message")
	assert.Equal(t, "ORDER_NOT_FOUND", joinNonEmpty("ORDER_NOT_FOUND", ""), "joinNonEmpty should return a label without a message")
	assert.Equal(t, "Order not found", joinNonEmpty("", "Order not found"), "joinNonEmpty should return a message without a label")
	assert.Empty(t, joinNonEmpty("", ""), "joinNonEmpty should return nothing for nothing")
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddress(t.Context(), currency.EMPTYCODE, "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDepositAddress must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetDepositAddress(t.Context(), currency.BTC, "", "GCTNOTACHAIN")
	require.ErrorIs(t, err, deposit.ErrAddressNotFound, "GetDepositAddress must reject a chain without an address")
	if !mockTests {
		chains, err := e.GetAvailableTransferChains(t.Context(), currency.BTC)
		require.NoError(t, err, "GetAvailableTransferChains must not error")
		for _, chain := range chains {
			result, err := e.GetDepositAddress(t.Context(), currency.BTC, "", chain)
			require.NoErrorf(t, err, "GetDepositAddress must not error for chain %s", chain)
			assert.NotEmptyf(t, result.Address, "GetDepositAddress should return an address for chain %s", chain)
		}
		return
	}
	// The fixture records an XLM address Gate failed to obtain
	_, err = e.GetDepositAddress(t.Context(), currency.BTC, "", "XLM")
	require.ErrorIs(t, err, deposit.ErrAddressNotFound, "GetDepositAddress must reject a chain whose address Gate failed to obtain")
	for _, tc := range []struct {
		ccy   currency.Code
		chain string
		exp   *deposit.Address
	}{
		{ccy: currency.BTC, exp: &deposit.Address{Address: "1HkxtBAMrA3tP5ENnYY2CZortjZvFDH5Cs"}},
		{ccy: currency.BTC, chain: "BSC", exp: &deposit.Address{Address: "0x1234567890abcdef1234567890abcdef12345678", Chain: "BSC"}},
		// The tag is the memo a deposit must carry, not the name of its kind
		{ccy: currency.XRP, chain: "XRP", exp: &deposit.Address{Address: "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", Tag: "100156837", Chain: "XRP"}},
	} {
		result, err := e.GetDepositAddress(t.Context(), tc.ccy, "", tc.chain)
		require.NoErrorf(t, err, "GetDepositAddress must not error for %s chain %q", tc.ccy, tc.chain)
		assert.Equalf(t, tc.exp, result, "GetDepositAddress should return the %s address of chain %q", tc.ccy, tc.chain)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetActiveOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetActiveOrders must reject a nil request")
	_, err = e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Binary, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetActiveOrders must reject an unsupported asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		a   asset.Item
		exp order.FilteredOrders
	}{
		{
			a: asset.Spot,
			exp: order.FilteredOrders{{
				Exchange:             e.Name,
				OrderID:              "12332324",
				ClientOrderID:        "t-123456",
				Side:                 order.Buy,
				Type:                 order.Limit,
				Status:               order.Open,
				Pair:                 getPair(t, asset.Spot),
				AssetType:            asset.Spot,
				Price:                5.00032,
				Amount:               1,
				ExecutedAmount:       0.5,
				RemainingAmount:      0.5,
				AverageExecutedPrice: 5.00032,
				Cost:                 2.50016,
				CostAsset:            currency.USDT,
				Fee:                  0.005,
				FeeAsset:             currency.BTC,
				Date:                 time.UnixMilli(1744150000123),
				LastUpdated:          time.UnixMilli(1744150100123),
			}},
		},
		// Gate lists every account's open orders together, and the only one recorded is a spot account order
		{a: asset.Margin, exp: order.FilteredOrders{}},
		{a: asset.CrossMargin, exp: order.FilteredOrders{}},
		{
			// An order of another contract is left out
			a: asset.USDTMarginedFutures,
			exp: order.FilteredOrders{{
				Exchange:             e.Name,
				OrderID:              "15675403",
				ClientOrderID:        "t-usdt-open",
				Side:                 order.Short,
				Type:                 order.Limit,
				Status:               order.Open,
				TimeInForce:          order.GoodTillCancel,
				ReduceOnly:           true,
				Pair:                 getPair(t, asset.USDTMarginedFutures),
				AssetType:            asset.USDTMarginedFutures,
				SettlementCurrency:   currency.USDT,
				Price:                31503.3,
				Amount:               10,
				ContractAmount:       10,
				ExecutedAmount:       6,
				RemainingAmount:      4,
				AverageExecutedPrice: 31510,
				Date:                 time.UnixMilli(1744150000500),
				LastUpdated:          time.UnixMilli(1744150100250),
			}},
		},
		{
			a: asset.CoinMarginedFutures,
			exp: order.FilteredOrders{{
				Exchange:           e.Name,
				OrderID:            "15675404",
				Side:               order.Long,
				Type:               order.Limit,
				Status:             order.Open,
				TimeInForce:        order.GoodTillCancel,
				Pair:               getPair(t, asset.CoinMarginedFutures),
				AssetType:          asset.CoinMarginedFutures,
				SettlementCurrency: currency.BTC,
				Price:              50000,
				Amount:             100,
				ContractAmount:     100,
				RemainingAmount:    100,
				Date:               time.Unix(1744150000, 0),
				LastUpdated:        time.Unix(1744150000, 0),
			}},
		},
		{
			a: asset.DeliveryFutures,
			exp: order.FilteredOrders{{
				Exchange:           e.Name,
				OrderID:            "15675394",
				ClientOrderID:      "t-my-custom-id",
				Side:               order.Long,
				Type:               order.Limit,
				Status:             order.Open,
				TimeInForce:        order.GoodTillCancel,
				Pair:               getPair(t, asset.DeliveryFutures),
				AssetType:          asset.DeliveryFutures,
				SettlementCurrency: currency.USDT,
				Price:              3765,
				Amount:             6024,
				ContractAmount:     6024,
				RemainingAmount:    6024,
				Date:               time.Unix(1546569968, 0),
				LastUpdated:        time.Unix(1546569968, 0),
			}},
		},
		{
			// Gate sizes a sell negative
			a: asset.Options,
			exp: order.FilteredOrders{{
				Exchange:        e.Name,
				OrderID:         "2",
				Side:            order.Short,
				Type:            order.Limit,
				Status:          order.Open,
				TimeInForce:     order.GoodTillCancel,
				Pair:            getPair(t, asset.Options),
				AssetType:       asset.Options,
				Price:           100,
				Amount:          1,
				RemainingAmount: 1,
				Date:            time.Unix(1631763361, 0),
				LastUpdated:     time.Unix(1631763361, 0),
			}},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			got, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{
				Pairs:     currency.Pairs{getPair(t, tc.a)},
				Type:      order.AnyType,
				Side:      order.AnySide,
				AssetType: tc.a,
			})
			require.NoError(t, err, "GetActiveOrders must not error")
			if mockTests {
				assert.Equal(t, tc.exp, got, "GetActiveOrders should return every open order of the pair")
				return
			}
			for i := range got {
				assert.Equal(t, tc.a, got[i].AssetType, "GetActiveOrders should return orders of the requested asset")
			}
		})
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetOrderHistory must reject a nil request")
	_, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Binary, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderHistory must reject an unsupported asset")
	_, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Options, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "GetOrderHistory must require options pairs")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	// spotFill is the spot fill the spot, margin and cross margin requests return, as Gate's trade history covers every
	// account
	spotFill := func(a asset.Item) order.FilteredOrders {
		return order.FilteredOrders{{
			Exchange:             e.Name,
			OrderID:              "987345678235",
			Side:                 order.Buy,
			Pair:                 getPair(t, a),
			AssetType:            a,
			Price:                65000,
			Amount:               0.5,
			ExecutedAmount:       0.5,
			AverageExecutedPrice: 65000,
			Cost:                 32500,
			CostAsset:            currency.USDT,
			Fee:                  0.0005,
			FeeAsset:             currency.BTC,
			Date:                 time.UnixMicro(1744150000123456),
			LastUpdated:          time.UnixMicro(1744150000123456),
			Trades: []order.TradeHistory{{
				TID:       "1232893233",
				Price:     65000,
				Amount:    0.5,
				Fee:       0.0005,
				FeeAsset:  "BTC",
				Side:      order.Buy,
				IsMaker:   true,
				Timestamp: time.UnixMicro(1744150000123456),
				Exchange:  e.Name,
			}},
		}}
	}
	// futuresFill is a futures fill, which Gate sizes negative for a sell, in contracts and identified by its own fill
	// ID besides the ID of the order it filled
	futuresFill := func(a asset.Item, orderID, fillID string, side order.Side, price, amount, fee float64, isMaker bool, date time.Time) order.FilteredOrders {
		settle, err := getSettlementCurrency(currency.EMPTYPAIR, a)
		require.NoErrorf(t, err, "getSettlementCurrency must not error for %s", a)
		return order.FilteredOrders{{
			Exchange:    e.Name,
			OrderID:     orderID,
			Side:        side,
			Pair:        getPair(t, a),
			AssetType:   a,
			Price:       price,
			Amount:      amount,
			CostAsset:   settle,
			Date:        date,
			LastUpdated: date,
			Trades: []order.TradeHistory{{
				TID:       fillID,
				Price:     price,
				Amount:    amount,
				Fee:       fee,
				Side:      side,
				IsMaker:   isMaker,
				Timestamp: date,
				Exchange:  e.Name,
			}},
		}}
	}
	usdtFill := futuresFill(asset.USDTMarginedFutures, "21893289840", "121234235", order.Short, 31500.5, 30, 0.0473, true, time.UnixMilli(1744150000123))
	coinFill := futuresFill(asset.CoinMarginedFutures, "21893289839", "121234231", order.Long, 100.123, 100, 0.01, false, time.UnixMilli(1514764800123))
	deliveryFill := futuresFill(asset.DeliveryFutures, "21893289839", "121234231", order.Short, 100.123, 100, 0.01, true, time.UnixMilli(1514764800123))
	optionsFill := futuresFill(asset.Options, "6", "7", order.Short, 100, 1, 0, true, time.Unix(1631763397, 0))
	for _, tc := range []struct {
		a         asset.Item
		withPairs bool
		exp       order.FilteredOrders
	}{
		{a: asset.Spot, withPairs: true, exp: spotFill(asset.Spot)},
		{a: asset.Spot, exp: spotFill(asset.Spot)},
		{a: asset.Margin, withPairs: true, exp: spotFill(asset.Margin)},
		{a: asset.Margin, exp: spotFill(asset.Margin)},
		{a: asset.CrossMargin, withPairs: true, exp: spotFill(asset.CrossMargin)},
		{a: asset.CrossMargin, exp: spotFill(asset.CrossMargin)},
		{a: asset.USDTMarginedFutures, withPairs: true, exp: usdtFill},
		{a: asset.USDTMarginedFutures, exp: usdtFill},
		{a: asset.CoinMarginedFutures, withPairs: true, exp: coinFill},
		{a: asset.CoinMarginedFutures, exp: coinFill},
		{a: asset.DeliveryFutures, withPairs: true, exp: deliveryFill},
		{a: asset.DeliveryFutures, exp: deliveryFill},
		{a: asset.Options, withPairs: true, exp: optionsFill},
	} {
		name := tc.a.String() + "/without_pairs"
		if tc.withPairs {
			name = tc.a.String() + "/with_pairs"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			req := &order.MultiOrderRequest{Type: order.AnyType, Side: order.AnySide, AssetType: tc.a}
			if tc.withPairs {
				req.Pairs = currency.Pairs{getPair(t, tc.a)}
			}
			got, err := e.GetOrderHistory(t.Context(), req)
			require.NoError(t, err, "GetOrderHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, got, "GetOrderHistory should return every fill")
				return
			}
			for i := range got {
				assert.Equal(t, tc.a, got[i].AssetType, "GetOrderHistory should return fills of the requested asset")
				assert.True(t, got[i].Pair.IsPopulated(), "GetOrderHistory should populate each fill's pair")
			}
		})
	}
}

func TestGetOrderHistoryRequestImmutability(t *testing.T) {
	t.Parallel()
	if !mockTests {
		testexch.UpdatePairsOnce(t, e)
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	enabledPairs := getPairs(t, asset.Spot)
	enabledPairs = enabledPairs[:min(2, len(enabledPairs))]

	type testCase struct {
		name    string
		request order.MultiOrderRequest
	}

	testCases := []testCase{
		{
			name: "nil_pairs",
			request: order.MultiOrderRequest{
				Type:      order.AnyType,
				Side:      order.Buy,
				AssetType: asset.Spot,
			},
		},
		{
			name: "provided_pairs",
			request: order.MultiOrderRequest{
				Type:      order.AnyType,
				Side:      order.Buy,
				Pairs:     slices.Clone(enabledPairs),
				AssetType: asset.Spot,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			expectedPairs := slices.Clone(tc.request.Pairs)
			_, err := e.GetOrderHistory(t.Context(), &tc.request)
			require.NoError(t, err)
			assert.Equal(t, expectedPairs, tc.request.Pairs)
		})
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetHistoricCandles(t.Context(), getPair(t, asset.Options), asset.Options, kline.OneDay, startTime, endTime)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricCandles must reject options")

	spotCandles := func() []kline.Candle {
		return []kline.Candle{
			{Time: time.Unix(1744070400, 0), Open: 79167.3, High: 80866.2, Low: 76249.4, Close: 76324.7, Volume: 3446.19726},
			{Time: time.Unix(1744156800, 0), Open: 76325.7, High: 83585, Low: 74633.5, Close: 82613.5, Volume: 9707.25837},
		}
	}
	for _, tc := range []struct {
		a   asset.Item
		exp []kline.Candle
	}{
		{a: asset.Spot, exp: spotCandles()},
		{a: asset.Margin, exp: spotCandles()},
		{a: asset.CrossMargin, exp: spotCandles()},
		{
			// Futures volumes count contracts
			a: asset.USDTMarginedFutures,
			exp: []kline.Candle{
				{Time: time.Unix(1744070400, 0), Open: 79145.8, High: 80829.6, Low: 76209.7, Close: 76303.1, Volume: 367739297},
				{Time: time.Unix(1744156800, 0), Open: 76303, High: 83598, Low: 74594.3, Close: 82586, Volume: 566153127},
			},
		},
		{
			a: asset.CoinMarginedFutures,
			exp: []kline.Candle{
				{Time: time.Unix(1744070400, 0), Open: 79173.1, High: 80689.2, Low: 76088, Close: 76095.5, Volume: 8333969},
				{Time: time.Unix(1744156800, 0), Open: 76183.3, High: 83450, Low: 74531, Close: 82394.2, Volume: 12502932},
			},
		},
		{
			a: asset.DeliveryFutures,
			exp: []kline.Candle{
				{Time: time.Unix(1744070400, 0), Open: 80020.4, High: 81726.2, Low: 77203.3, Close: 77242, Volume: 66260},
				{Time: time.Unix(1744156800, 0), Open: 77266.7, High: 84575.4, Low: 75565.3, Close: 83650.2, Volume: 65609},
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			marketEx, pair := getMarketDataPair(t, e, tc.a, hasDeliveryCandles(kline.OneDay, startTime, endTime))
			result, err := marketEx.GetHistoricCandles(t.Context(), pair, tc.a, kline.OneDay, startTime, endTime)
			require.NoErrorf(t, err, "GetHistoricCandles must not error for %s", pair)
			if !mockTests {
				assert.NotEmptyf(t, result.Candles, "GetHistoricCandles should return candles for %s", pair)
				return
			}
			// The fixtures only match a request spanning whole days, and the candle opening at its end, which Gate also
			// returns, is left out
			exp := &kline.Item{Exchange: e.Name, Pair: getPair(t, tc.a), Asset: tc.a, Interval: kline.OneDay, Candles: tc.exp}
			assert.Equal(t, exp, result, "GetHistoricCandles should return every candle of the window")
		})
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetHistoricCandlesExtended(t.Context(), getPair(t, asset.Options), asset.Options, kline.FourHour, startTime, endTime)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricCandlesExtended must reject options")

	// The fixtures return no candles after midnight, so the last two periods of the window are padded
	padding := []kline.Candle{{Time: time.Unix(1744171200, 0).UTC()}, {Time: time.Unix(1744185600, 0).UTC()}}
	spotCandles := func() []kline.Candle {
		return append([]kline.Candle{
			{Time: time.Unix(1744099200, 0), Open: 79494.3, High: 80065.6, Low: 78830.1, Close: 79981.3, Volume: 367.39632},
			{Time: time.Unix(1744113600, 0), Open: 79987, High: 80371.6, Low: 78185.5, Close: 78497.9, Volume: 788.95289},
			{Time: time.Unix(1744128000, 0), Open: 78497.8, High: 78524.3, Low: 76342.4, Close: 76857.5, Volume: 1039.09048},
			{Time: time.Unix(1744142400, 0), Open: 76857.8, High: 77285.1, Low: 76249.4, Close: 76324.7, Volume: 448.57618},
			{Time: time.Unix(1744156800, 0), Open: 76325.7, High: 76870.7, Low: 74633.5, Close: 75089.6, Volume: 1221.24348},
		}, padding...)
	}
	for _, tc := range []struct {
		a   asset.Item
		exp []kline.Candle
	}{
		{a: asset.Spot, exp: spotCandles()},
		{a: asset.Margin, exp: spotCandles()},
		{a: asset.CrossMargin, exp: spotCandles()},
		{
			a: asset.USDTMarginedFutures,
			exp: append([]kline.Candle{
				{Time: time.Unix(1744099200, 0), Open: 79456.8, High: 80032.3, Low: 78800, Close: 79948.1, Volume: 40040576},
				{Time: time.Unix(1744113600, 0), Open: 79943.7, High: 80342, Low: 78126, Close: 78463.6, Volume: 101353903},
				{Time: time.Unix(1744128000, 0), Open: 78461.2, High: 78493.8, Low: 76302.1, Close: 76824, Volume: 94027904},
				{Time: time.Unix(1744142400, 0), Open: 76828.3, High: 77260.6, Low: 76209.7, Close: 76303.1, Volume: 38795519},
				{Time: time.Unix(1744156800, 0), Open: 76303, High: 76842, Low: 74594.3, Close: 75048.5, Volume: 94167913},
			}, padding...),
		},
		{
			a: asset.CoinMarginedFutures,
			exp: append([]kline.Candle{
				{Time: time.Unix(1744099200, 0), Open: 79262.8, High: 79927.5, Low: 78780, Close: 79861.2, Volume: 759064},
				{Time: time.Unix(1744113600, 0), Open: 79927.5, High: 80249.7, Low: 78126.4, Close: 78460.7, Volume: 1733264},
				{Time: time.Unix(1744128000, 0), Open: 78450, High: 78450, Low: 76247, Close: 76646.4, Volume: 3369527},
				{Time: time.Unix(1744142400, 0), Open: 76742.5, High: 77088.2, Low: 76088, Close: 76095.5, Volume: 1173958},
				{Time: time.Unix(1744156800, 0), Open: 76183.3, High: 76867.5, Low: 74531, Close: 74976.3, Volume: 2822737},
			}, padding...),
		},
		{
			a: asset.DeliveryFutures,
			exp: append([]kline.Candle{
				{Time: time.Unix(1744099200, 0), Open: 80323.9, High: 80872.8, Low: 79664.1, Close: 80824.4, Volume: 10500},
				{Time: time.Unix(1744113600, 0), Open: 80870.9, High: 81285.1, Low: 79096.2, Close: 79406.8, Volume: 11067},
				{Time: time.Unix(1744128000, 0), Open: 79365.6, High: 79415.8, Low: 77285, Close: 77798.5, Volume: 11525},
				{Time: time.Unix(1744142400, 0), Open: 77795.1, High: 78235.5, Low: 77203.3, Close: 77242, Volume: 10375},
				{Time: time.Unix(1744156800, 0), Open: 77266.7, High: 77798.8, Low: 75565.3, Close: 75987.8, Volume: 11106},
			}, padding...),
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			marketEx, pair := getMarketDataPair(t, e, tc.a, hasDeliveryCandles(kline.FourHour, startTime, endTime))
			result, err := marketEx.GetHistoricCandlesExtended(t.Context(), pair, tc.a, kline.FourHour, startTime, endTime)
			require.NoErrorf(t, err, "GetHistoricCandlesExtended must not error for %s", pair)
			if !mockTests {
				assert.NotEmptyf(t, result.Candles, "GetHistoricCandlesExtended should return candles for %s", pair)
				return
			}
			// Spot volumes are the base amounts, as GetHistoricCandles reports them
			exp := &kline.Item{Exchange: e.Name, Pair: getPair(t, tc.a), Asset: tc.a, Interval: kline.FourHour, Candles: tc.exp}
			assert.Equal(t, exp, result, "GetHistoricCandlesExtended should return every candle of the window")
		})
	}

	t.Run("window beyond one request", func(t *testing.T) {
		t.Parallel()
		ex := newIsolatedExchange(t)
		// A limit of four candles a request splits the seven of the window into two requests
		ex.Features.Enabled.Kline.GlobalResultLimit = 4
		result, err := ex.GetHistoricCandlesExtended(t.Context(), getPair(t, asset.Spot), asset.Spot, kline.FourHour, startTime, endTime)
		require.NoError(t, err, "GetHistoricCandlesExtended must not error")
		if !mockTests {
			assert.NotEmpty(t, result.Candles, "GetHistoricCandlesExtended should return candles")
			return
		}
		exp := &kline.Item{Exchange: ex.Name, Pair: getPair(t, asset.Spot), Asset: asset.Spot, Interval: kline.FourHour, Candles: spotCandles()}
		assert.Equal(t, exp, result, "GetHistoricCandlesExtended should merge the candles of every request")
	})
}

// TestGetHistoricCandlesBuildsUnservedIntervals checks an interval Gate does not serve is built from the candles of a
// shorter one it does, requested over whole periods of the interval asked for
func TestGetHistoricCandlesBuildsUnservedIntervals(t *testing.T) {
	t.Parallel()
	type hourlyCandle struct {
		time                                time.Time
		open, high, low, closePrice, volume float64
	}
	// Gate also returns the candle opening at 12:00, the end of the window
	hourly := []hourlyCandle{
		{time.Unix(1744092000, 0), 100, 110, 95, 105, 1},
		{time.Unix(1744095600, 0), 105, 120, 100, 115, 2},
		{time.Unix(1744099200, 0), 115, 118, 90, 92, 3},
		{time.Unix(1744102800, 0), 92, 99, 91, 98, 4},
		{time.Unix(1744106400, 0), 98, 130, 97, 125, 5},
		{time.Unix(1744110000, 0), 125, 126, 110, 111, 6},
		{time.Unix(1744113600, 0), 111, 112, 108, 109, 7},
	}
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	require.NoError(t, storeTestPairs(ex), "storeTestPairs must not error")
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		// The window opening at 08:00 widens to the three hour period it falls in, which opens at 06:00
		assert.Equal(t, "1h", query.Get("interval"), "request should ask for the interval the candles are built from")
		assert.Equal(t, "1744092000", query.Get("from"), "request should start at the period the window starts in")
		assert.Equal(t, "1744113600", query.Get("to"), "request should end at the end of the window")
		candles := make([]string, len(hourly))
		for i, c := range hourly {
			if r.URL.Path == "/api/v4/spot/candlesticks" {
				candles[i] = fmt.Sprintf(`["%d","1","%v","%v","%v","%v","%v","true"]`, c.time.Unix(), c.closePrice, c.high, c.low, c.open, c.volume)
				continue
			}
			candles[i] = fmt.Sprintf(`{"t":%d,"o":"%v","h":"%v","l":"%v","c":"%v","v":%v}`, c.time.Unix(), c.open, c.high, c.low, c.closePrice, c.volume)
		}
		_, err := fmt.Fprint(w, "["+strings.Join(candles, ",")+"]")
		assert.NoError(t, err, "writing the candles should not error")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")

	start, end := time.Unix(1744099200, 0), time.Unix(1744113600, 0)
	for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.DeliveryFutures} {
		pair := mockTestPairs()[a]
		exp := &kline.Item{
			Exchange: ex.Name,
			Pair:     pair,
			Asset:    a,
			Interval: kline.ThreeHour,
			Candles: []kline.Candle{
				{Time: time.Unix(1744092000, 0), Open: 100, High: 120, Low: 90, Close: 92, Volume: 6},
				{Time: time.Unix(1744102800, 0), Open: 92, High: 130, Low: 91, Close: 111, Volume: 15},
			},
		}
		result, err := ex.GetHistoricCandles(t.Context(), pair, a, kline.ThreeHour, start, end)
		require.NoErrorf(t, err, "GetHistoricCandles must not error for %s", a)
		assert.Equalf(t, exp, result, "GetHistoricCandles should build the %s candles from hourly ones", a)
		result, err = ex.GetHistoricCandlesExtended(t.Context(), pair, a, kline.ThreeHour, start, end)
		require.NoErrorf(t, err, "GetHistoricCandlesExtended must not error for %s", a)
		assert.Equalf(t, exp, result, "GetHistoricCandlesExtended should build the %s candles from hourly ones", a)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	ex := newIsolatedExchange(t)
	require.ErrorIs(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Binary), asset.ErrNotSupported, "UpdateOrderExecutionLimits must reject an unsupported asset")

	if mockTests {
		for _, tc := range []struct {
			a   asset.Item
			exp limits.MinMaxLevel
		}{
			{
				// The earlier of the buy and sell start times is the listing time
				a: asset.Spot,
				exp: limits.MinMaxLevel{
					PriceStepIncrementSize:  0.00001,
					MultiplierUp:            0.4,
					MultiplierDown:          0.4,
					MinimumBaseAmount:       0.01,
					MinimumQuoteAmount:      3,
					MaximumQuoteAmount:      5000000,
					AmountStepIncrementSize: 0.01,
					MarketMaxQty:            7228044.81,
					Listed:                  time.Unix(1608264000, 0),
				},
			},
			{
				// Margin limits are the spot limits with the market's own borrow minimums
				a: asset.Margin,
				exp: limits.MinMaxLevel{
					PriceStepIncrementSize:   0.00001,
					MinimumBaseAmount:        0.01,
					MinimumQuoteAmount:       3,
					AmountStepIncrementSize:  0.01,
					MinimumBorrowAmountBase:  0.00001,
					MinimumBorrowAmountQuote: 1,
				},
			},
			{
				// Cross margin borrow minimums are those of each currency
				a: asset.CrossMargin,
				exp: limits.MinMaxLevel{
					PriceStepIncrementSize:   0.00001,
					MinimumBaseAmount:        0.01,
					MinimumQuoteAmount:       3,
					AmountStepIncrementSize:  0.01,
					MinimumBorrowAmountBase:  0.01,
					MinimumBorrowAmountQuote: 3,
				},
			},
			{
				// Futures amounts count whole contracts
				a: asset.USDTMarginedFutures,
				exp: limits.MinMaxLevel{
					PriceStepIncrementSize:  0.01,
					PriceDivisor:            1,
					MultiplierDecimal:       0.01,
					MinimumBaseAmount:       1,
					MaximumBaseAmount:       32000,
					AmountStepIncrementSize: 1,
					Listed:                  time.Unix(1769416200, 0),
				},
			},
			{
				a: asset.CoinMarginedFutures,
				exp: limits.MinMaxLevel{
					PriceStepIncrementSize:  0.1,
					PriceDivisor:            1,
					MinimumBaseAmount:       1,
					MaximumBaseAmount:       600000,
					AmountStepIncrementSize: 1,
					Listed:                  time.Unix(1545235200, 0),
				},
			},
			{
				a: asset.DeliveryFutures,
				exp: limits.MinMaxLevel{
					PriceStepIncrementSize:  0.00001,
					PriceDivisor:            1,
					MultiplierDecimal:       10,
					MinimumBaseAmount:       1,
					MaximumBaseAmount:       1000000,
					AmountStepIncrementSize: 1,
					Expiry:                  time.Unix(1775808000, 0),
				},
			},
			{
				a: asset.Options,
				exp: limits.MinMaxLevel{
					PriceStepIncrementSize:  1,
					PriceDivisor:            1,
					MultiplierDecimal:       0.01,
					MinimumBaseAmount:       1,
					MaximumBaseAmount:       10000,
					AmountStepIncrementSize: 1,
				},
			},
		} {
			t.Run(tc.a.String(), func(t *testing.T) {
				t.Parallel()
				started := time.Now()
				require.NoError(t, ex.UpdateOrderExecutionLimits(t.Context(), tc.a), "UpdateOrderExecutionLimits must not error")
				result, err := ex.GetOrderExecutionLimits(tc.a, getPair(t, tc.a))
				require.NoError(t, err, "GetOrderExecutionLimits must not error")
				assert.WithinRange(t, result.UpdatedAt, started, time.Now(), "UpdateOrderExecutionLimits should stamp the limits when loading them")
				tc.exp.UpdatedAt = result.UpdatedAt
				tc.exp.Key = key.NewExchangeAssetPair(ex.Name, tc.a, getPair(t, tc.a))
				assert.Equal(t, tc.exp, result, "UpdateOrderExecutionLimits should load the pair's limits")
			})
		}
		return
	}
	for _, a := range ex.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			require.NoError(t, ex.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error")
			pairs, err := ex.GetAvailablePairs(a)
			require.NoError(t, err, "GetAvailablePairs must not error")
			for _, p := range pairs {
				l, err := ex.GetOrderExecutionLimits(a, p)
				require.NoErrorf(t, err, "GetOrderExecutionLimits must not error for %s", p)
				require.NotNilf(t, l, "GetOrderExecutionLimits must return limits for %s", p)
				assert.Equalf(t, a, l.Key.Asset, "asset should equal for %s", p)
				assert.Truef(t, p.Equal(l.Key.Pair()), "pair should equal for %s", p)
				switch a {
				case asset.Options:
					assert.Positivef(t, l.MinimumBaseAmount, "MinimumBaseAmount should be positive for %s", p)
					assert.Positivef(t, l.MaximumBaseAmount, "MaximumBaseAmount should be positive for %s", p)
					assert.Positivef(t, l.PriceStepIncrementSize, "PriceStepIncrementSize should be positive for %s", p)
					assert.Positivef(t, l.AmountStepIncrementSize, "AmountStepIncrementSize should be positive for %s", p)
					assert.Positivef(t, l.MultiplierDecimal, "MultiplierDecimal should be positive for %s", p)
				case asset.USDTMarginedFutures:
					assert.Positivef(t, l.MultiplierDecimal, "MultiplierDecimal should be positive for %s", p)
					assert.NotZerof(t, l.Listed, "Listed should be populated for %s", p)
					fallthrough
				case asset.CoinMarginedFutures:
					if !l.Delisted.IsZero() {
						assert.Truef(t, l.Delisted.After(l.Delisting), "Delisted should be after Delisting for %s", p)
					}
					assert.Positivef(t, l.AmountStepIncrementSize, "AmountStepIncrementSize should be positive for %s", p)
				case asset.Spot:
					assert.Positivef(t, l.MinimumQuoteAmount, "MinimumQuoteAmount should be positive for %s", p)
					assert.Positivef(t, l.PriceStepIncrementSize, "PriceStepIncrementSize should be positive for %s", p)
					assert.Positivef(t, l.MinimumBaseAmount, "MinimumBaseAmount should be positive for %s", p)
					assert.Positivef(t, l.AmountStepIncrementSize, "AmountStepIncrementSize should be positive for %s", p)
				case asset.Margin, asset.CrossMargin:
					assert.Positivef(t, l.MinimumQuoteAmount, "MinimumQuoteAmount should be positive for %s", p)
					assert.Positivef(t, l.MinimumBaseAmount, "MinimumBaseAmount should be positive for %s", p)
					assert.Positivef(t, l.PriceStepIncrementSize, "PriceStepIncrementSize should be positive for %s", p)
					invalidPrice := l.PriceStepIncrementSize / 2
					assert.ErrorIsf(t, l.Validate(invalidPrice, l.MinimumBaseAmount, order.Limit), limits.ErrPriceExceedsStep, "Validate should reject an invalid price tick for %s", p)
					if l.AmountStepIncrementSize != 0 {
						assert.Positivef(t, l.AmountStepIncrementSize, "AmountStepIncrementSize should be positive for %s when set", p)
					}
					assert.Positivef(t, l.MinimumBorrowAmountBase, "MinimumBorrowAmountBase should be positive for %s", p)
					assert.Positivef(t, l.MinimumBorrowAmountQuote, "MinimumBorrowAmountQuote should be positive for %s", p)
				case asset.DeliveryFutures:
					assert.NotZerof(t, l.Expiry, "Expiry should be populated for %s", p)
					assert.Positivef(t, l.MinimumBaseAmount, "MinimumBaseAmount should be positive for %s", p)
					assert.Positivef(t, l.AmountStepIncrementSize, "AmountStepIncrementSize should be positive for %s", p)
					assert.Positivef(t, l.MultiplierDecimal, "MultiplierDecimal should be positive for %s", p)
				}
			}
		})
	}
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetFuturesContractDetails must reject a spot asset")

	_, err = e.GetFuturesContractDetails(t.Context(), asset.PerpetualContract)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetFuturesContractDetails must reject an unsupported futures asset")

	for _, tc := range []struct {
		a     asset.Item
		count int
		exp   futures.Contract
	}{
		{
			// The latest rate is the one applying since the previous funding time
			a:     asset.USDTMarginedFutures,
			count: 6,
			exp: futures.Contract{
				Exchange:           e.Name,
				Name:               getPair(t, asset.USDTMarginedFutures),
				Underlying:         getPair(t, asset.USDTMarginedFutures),
				Asset:              asset.USDTMarginedFutures,
				IsActive:           true,
				Type:               futures.Perpetual,
				SettlementType:     futures.Linear,
				SettlementCurrency: currency.USDT,
				Multiplier:         0.01,
				MaxLeverage:        25,
				LatestRate:         fundingrate.Rate{Time: time.Unix(1775404800, 0), Rate: decimal.MustFromFloat(0)},
			},
		},
		{
			a:     asset.CoinMarginedFutures,
			count: 1,
			exp: futures.Contract{
				Exchange:           e.Name,
				Name:               getPair(t, asset.CoinMarginedFutures),
				Underlying:         getPair(t, asset.CoinMarginedFutures),
				Asset:              asset.CoinMarginedFutures,
				IsActive:           true,
				Type:               futures.Perpetual,
				SettlementType:     futures.Inverse,
				SettlementCurrency: currency.BTC,
				MaxLeverage:        100,
				LatestRate:         fundingrate.Rate{Time: time.Unix(1775404800, 0), Rate: decimal.MustFromFloat(0.0001)},
			},
		},
		{
			// A bi-weekly contract is taken to start two weeks before it expires
			a:     asset.DeliveryFutures,
			count: 6,
			exp: futures.Contract{
				Exchange:           e.Name,
				Name:               getPair(t, asset.DeliveryFutures),
				Underlying:         currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter),
				Asset:              asset.DeliveryFutures,
				StartDate:          time.Unix(1774598400, 0),
				EndDate:            time.Unix(1775808000, 0),
				IsActive:           true,
				Type:               futures.Fortnightly,
				SettlementType:     futures.Linear,
				SettlementCurrency: currency.USDT,
				Multiplier:         10,
				MaxLeverage:        50,
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFuturesContractDetails(t.Context(), tc.a)
			require.NoError(t, err, "GetFuturesContractDetails must not error")
			if !mockTests {
				assert.NotEmpty(t, result, "GetFuturesContractDetails should return contracts")
				return
			}
			require.Len(t, result, tc.count, "GetFuturesContractDetails must return every recorded contract")
			assert.Equal(t, tc.exp, result[0], "GetFuturesContractDetails should return every field of the contract")
		})
	}
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLatestFundingRates must reject a nil request")
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.DeliveryFutures})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetLatestFundingRates must reject an asset without funding rates")

	usdtPair := currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
	coinPair := currency.NewPairWithDelimiter("BTC", "USD", currency.UnderscoreDelimiter)
	// Each contract's rate applies from one funding interval before its next funding time
	rate := func(a asset.Item, p currency.Pair, r float64) fundingrate.LatestRateResponse {
		return fundingrate.LatestRateResponse{
			Exchange:       e.Name,
			Asset:          a,
			Pair:           p,
			LatestRate:     fundingrate.Rate{Time: time.Unix(1775404800, 0), Rate: decimal.MustFromFloat(r)},
			TimeOfNextRate: time.Unix(1775433600, 0),
		}
	}
	predicted := rate(asset.USDTMarginedFutures, usdtPair, 0.000083)
	predicted.PredictedUpcomingRate = fundingrate.Rate{Time: time.Unix(1775433600, 0), Rate: decimal.MustFromFloat(0.000083)}
	for _, tc := range []struct {
		name string
		req  *fundingrate.LatestRateRequest
		exp  []fundingrate.LatestRateResponse
	}{
		{
			name: "USDT-M pair with the predicted rate",
			req:  &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures, Pair: currency.NewBTCUSDT(), IncludePredictedRate: true},
			exp:  []fundingrate.LatestRateResponse{predicted},
		},
		{
			name: "COIN-M pair",
			req:  &fundingrate.LatestRateRequest{Asset: asset.CoinMarginedFutures, Pair: currency.NewBTCUSD()},
			exp:  []fundingrate.LatestRateResponse{rate(asset.CoinMarginedFutures, coinPair, 0.0001)},
		},
		{
			// Contracts that are not enabled are left out
			name: "COIN-M enabled pairs",
			req:  &fundingrate.LatestRateRequest{Asset: asset.CoinMarginedFutures},
			exp:  []fundingrate.LatestRateResponse{rate(asset.CoinMarginedFutures, coinPair, 0.0001)},
		},
		{
			// The recorded contract list holds a rate of 0 for the pair
			name: "USDT-M enabled pairs",
			req:  &fundingrate.LatestRateRequest{Asset: asset.USDTMarginedFutures},
			exp:  []fundingrate.LatestRateResponse{rate(asset.USDTMarginedFutures, usdtPair, 0)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			started := time.Now()
			result, err := e.GetLatestFundingRates(t.Context(), tc.req)
			require.NoError(t, err, "GetLatestFundingRates must not error")
			if !mockTests {
				assert.NotEmpty(t, result, "GetLatestFundingRates should return rates")
				return
			}
			require.Len(t, result, len(tc.exp), "GetLatestFundingRates must return a rate for every contract")
			for i := range result {
				assert.WithinRange(t, result[i].TimeChecked, started, time.Now(), "GetLatestFundingRates should stamp the time each rate is checked")
				tc.exp[i].TimeChecked = result[i].TimeChecked
			}
			assert.Equal(t, tc.exp, result, "GetLatestFundingRates should return every field of the rates")
		})
	}
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricalFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetHistoricalFundingRates must reject a nil request")

	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricalFundingRates must reject an unset asset")

	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.CoinMarginedFutures})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetHistoricalFundingRates must reject an empty pair")

	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Futures})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricalFundingRates must reject an asset without funding rates")

	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset: asset.USDTMarginedFutures,
		Pair:  getPair(t, asset.USDTMarginedFutures),
	})
	require.ErrorIs(t, err, fundingrate.ErrPaymentCurrencyCannotBeEmpty, "GetHistoricalFundingRates must reject an empty payment currency")

	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.USDTMarginedFutures,
		Pair:            getPair(t, asset.USDTMarginedFutures),
		PaymentCurrency: currency.USDT,
		IncludePayments: true,
	})
	require.ErrorIs(t, err, common.ErrNotYetImplemented, "GetHistoricalFundingRates must reject a request for payments")

	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:                asset.USDTMarginedFutures,
		Pair:                 getPair(t, asset.USDTMarginedFutures),
		PaymentCurrency:      currency.USDT,
		IncludePredictedRate: true,
	})
	require.ErrorIs(t, err, common.ErrNotYetImplemented, "GetHistoricalFundingRates must reject a request for the predicted rate")

	startTime, endTime := getTime()
	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.USDTMarginedFutures,
		Pair:            getPair(t, asset.USDTMarginedFutures),
		PaymentCurrency: currency.USDT,
		StartDate:       endTime,
		EndDate:         startTime,
	})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetHistoricalFundingRates must reject a start after the end")

	_, err = e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.USDTMarginedFutures,
		Pair:            getPair(t, asset.USDTMarginedFutures),
		PaymentCurrency: currency.USDT,
		StartDate:       time.Now().Add(-time.Hour * 8008),
		EndDate:         time.Now(),
	})
	require.ErrorIs(t, err, fundingrate.ErrFundingRateOutsideLimits, "GetHistoricalFundingRates must reject a start before the rates Gate returns")

	result, err := e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.USDTMarginedFutures,
		Pair:            getPair(t, asset.USDTMarginedFutures),
		PaymentCurrency: currency.USDT,
	})
	require.NoError(t, err, "GetHistoricalFundingRates must not error")
	if !mockTests {
		assert.NotEmpty(t, result.FundingRates, "GetHistoricalFundingRates should return rates")
		return
	}
	// Gate lists the latest rate first
	exp := &fundingrate.HistoricalRates{
		Exchange:   e.Name,
		Asset:      asset.USDTMarginedFutures,
		Pair:       getPair(t, asset.USDTMarginedFutures),
		StartDate:  time.Unix(1774954800, 0),
		EndDate:    time.Unix(1774969200, 0),
		LatestRate: fundingrate.Rate{Time: time.Unix(1774969200, 0), Rate: decimal.MustFromFloat(-0.000449)},
		FundingRates: []fundingrate.Rate{
			{Time: time.Unix(1774969200, 0), Rate: decimal.MustFromFloat(-0.000449)},
			{Time: time.Unix(1774965600, 0), Rate: decimal.MustFromFloat(0.000012)},
			{Time: time.Unix(1774962000, 0), Rate: decimal.MustFromFloat(-0.000236)},
			{Time: time.Unix(1774958401, 0), Rate: decimal.MustFromFloat(0.000012)},
			{Time: time.Unix(1774954800, 0), Rate: decimal.MustFromFloat(-0.000002)},
		},
		PaymentCurrency: currency.USDT,
	}
	assert.Equal(t, exp, result, "GetHistoricalFundingRates should return every rate")
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.getOpenInterestContracts(t.Context(), asset.Options, currency.EMPTYPAIR)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "options should remain unsupported by futures open interest endpoints")
	_, err = e.GetOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.NewCode("GOLDFISH").Item,
		Quote: currency.USDT.Item,
		Asset: asset.USDTMarginedFutures,
	})
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "GetOpenInterest should error correctly")

	if mockTests {
		pairKey := func(a asset.Item) key.PairAsset {
			p := getPair(t, a)
			return key.PairAsset{Base: p.Base.Item, Quote: p.Quote.Item, Asset: a}
		}
		openInterest := func(a asset.Item, amount float64) futures.OpenInterest {
			return futures.OpenInterest{Key: key.NewExchangeAssetPair(e.Name, a, getPair(t, a)), OpenInterest: amount}
		}
		// A pair asked for on its own is read from its contract and several from the contract list, which the fixtures
		// record at different times. Open interest is sized in the quote currency, which an inverse contract's
		// position already is
		for _, tc := range []struct {
			name string
			keys []key.PairAsset
			exp  []futures.OpenInterest
		}{
			{
				name: "USDT-M pair",
				keys: []key.PairAsset{pairKey(asset.USDTMarginedFutures)},
				exp:  []futures.OpenInterest{openInterest(asset.USDTMarginedFutures, 1949859099.6860802)},
			},
			{
				name: "COIN-M pair",
				keys: []key.PairAsset{pairKey(asset.CoinMarginedFutures)},
				exp:  []futures.OpenInterest{openInterest(asset.CoinMarginedFutures, 8701256)},
			},
			{
				name: "delivery pair",
				keys: []key.PairAsset{pairKey(asset.DeliveryFutures)},
				exp:  []futures.OpenInterest{openInterest(asset.DeliveryFutures, 8485.242744000001)},
			},
			{
				name: "perpetual pairs",
				keys: []key.PairAsset{pairKey(asset.CoinMarginedFutures), pairKey(asset.USDTMarginedFutures)},
				exp:  []futures.OpenInterest{openInterest(asset.CoinMarginedFutures, 8701256), openInterest(asset.USDTMarginedFutures, 118106.768)},
			},
			{
				name: "enabled pairs",
				exp:  []futures.OpenInterest{openInterest(asset.DeliveryFutures, 95.8187), openInterest(asset.CoinMarginedFutures, 8701256), openInterest(asset.USDTMarginedFutures, 118106.768)},
			},
		} {
			result, err := e.GetOpenInterest(t.Context(), tc.keys...)
			require.NoErrorf(t, err, "GetOpenInterest must not error for %s", tc.name)
			assert.Equalf(t, tc.exp, result, "GetOpenInterest should return the open interest of %s", tc.name)
		}
		return
	}
	var resp []futures.OpenInterest
	validPairs := make(map[asset.Item]key.PairAsset, 3)
	for _, a := range []asset.Item{asset.CoinMarginedFutures, asset.USDTMarginedFutures, asset.DeliveryFutures} {
		pairs := getPairs(t, a)
		if a == asset.DeliveryFutures {
			// The enabled delivery contract is picked at random and may have no open positions
			pairs, err = e.GetAvailablePairs(a)
			require.NoErrorf(t, err, "GetAvailablePairs must not error for %s", a)
		}
		pair, response := getPairWithOpenInterest(t, a, pairs)
		validPairs[a] = pair
		resp = response
		require.Lenf(t, resp, 1, "GetOpenInterest must return 1 item for %s asset", a)
		assert.Positivef(t, resp[0].OpenInterest, "GetOpenInterest should return positive open interest for %s asset", a)
	}

	coinPair := validPairs[asset.CoinMarginedFutures]
	usdtPair := validPairs[asset.USDTMarginedFutures]
	resp, err = e.GetOpenInterest(
		t.Context(),
		coinPair,
		usdtPair,
	)
	assert.NoError(t, err, "GetOpenInterest should not error for multiple explicit perpetual pairs")
	require.Len(t, resp, 2, "GetOpenInterest returns exactly the requested perpetual pairs")

	expected := map[asset.Item]currency.Pair{
		asset.CoinMarginedFutures: coinPair.Pair(),
		asset.USDTMarginedFutures: usdtPair.Pair(),
	}
	found := make(map[asset.Item]bool, len(expected))
	for _, oi := range resp {
		expPair, ok := expected[oi.Key.Asset]
		require.Truef(t, ok, "unexpected asset in OpenInterest response: %v", oi.Key.Asset)
		assert.Truef(t, expPair.Equal(oi.Key.Pair()), "OpenInterest pair mismatch for asset %v", oi.Key.Asset)
		assert.Positivef(t, oi.OpenInterest, "OpenInterest should return positive open interest for asset %v", oi.Key.Asset)
		found[oi.Key.Asset] = true
	}
	require.Len(t, found, len(expected), "OpenInterest response missing expected assets")

	resp, err = e.GetOpenInterest(t.Context())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp, "GetOpenInterest should return some items")
}

func TestGetClientOrderIDFromText(t *testing.T) {
	t.Parallel()
	assert.Empty(t, getClientOrderIDFromText("api"), "should not return anything")
	assert.Equal(t, "t-123", getClientOrderIDFromText("t-123"), "should return t-123")
}

func TestFormatClientOrderID(t *testing.T) {
	t.Parallel()
	assert.Empty(t, formatClientOrderID(""), "should not return anything")
	assert.Equal(t, "t-123", formatClientOrderID("t-123"), "should return t-123")
	assert.Equal(t, "t-456", formatClientOrderID("456"), "should return t-456")
}

func TestGetSideAndAmountFromSize(t *testing.T) {
	t.Parallel()
	side, amount, remaining := getSideAndAmountFromSize(1, 1)
	assert.Equal(t, order.Long, side, "should be a buy order")
	assert.Equal(t, 1.0, amount, "should be 1.0")
	assert.Equal(t, 1.0, remaining, "should be 1.0")

	side, amount, remaining = getSideAndAmountFromSize(-1, -1)
	assert.Equal(t, order.Short, side, "should be a sell order")
	assert.Equal(t, 1.0, amount, "should be 1.0")
	assert.Equal(t, 1.0, remaining, "should be 1.0")
}

func TestTradeSideAndAmount(t *testing.T) {
	t.Parallel()
	for size, exp := range map[float64]struct {
		side   order.Side
		amount float64
	}{
		4:  {side: order.Buy, amount: 4},
		-4: {side: order.Sell, amount: 4},
	} {
		side, amount := tradeSideAndAmount(size)
		assert.Equalf(t, exp.side, side, "tradeSideAndAmount should return the taker side of size %v", size)
		assert.Equalf(t, exp.amount, amount, "tradeSideAndAmount should return the unsigned amount of size %v", size)
	}
}

func TestGetFutureOrderSize(t *testing.T) {
	t.Parallel()
	_, err := getFutureOrderSize(&order.Submit{Side: order.CouldNotCloseShort, Amount: 1})
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)

	ret, err := getFutureOrderSize(&order.Submit{Side: order.Buy, Amount: 1})
	require.NoError(t, err)
	assert.Equal(t, 1.0, ret)

	ret, err = getFutureOrderSize(&order.Submit{Side: order.Sell, Amount: 1})
	require.NoError(t, err)
	assert.Equal(t, -1.0, ret)
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	if !mockTests {
		testexch.UpdatePairsOnce(t, e)
	}
	for a, page := range map[asset.Item]string{
		asset.Spot:                "trade/",
		asset.Margin:              "trade/",
		asset.CrossMargin:         "trade/",
		asset.USDTMarginedFutures: "futures/USDT/",
		asset.CoinMarginedFutures: "futures/BTC/",
		asset.DeliveryFutures:     "futures-delivery/USDT/",
	} {
		p := getPair(t, a)
		resp, err := e.GetCurrencyTradeURL(t.Context(), a, p)
		require.NoErrorf(t, err, "GetCurrencyTradeURL must not error for %s", a)
		p.Delimiter = currency.UnderscoreDelimiter
		assert.Equalf(t, "https://www.gate.io/"+page+p.Upper().String(), resp, "GetCurrencyTradeURL should return the %s trade page", a)
	}
	_, err := e.GetCurrencyTradeURL(t.Context(), asset.Options, getPair(t, asset.Options))
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetCurrencyTradeURL should reject options")
	_, err = e.GetCurrencyTradeURL(t.Context(), asset.Binary, getPair(t, asset.Spot))
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetCurrencyTradeURL should reject an asset the exchange does not trade")

	ex := newIsolatedExchange(t)
	ethusd := currency.NewPairWithDelimiter("ETH", "USD", currency.UnderscoreDelimiter)
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.CoinMarginedFutures, currency.Pairs{ethusd}, false), "StorePairs must add the contract")
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.CoinMarginedFutures, currency.Pairs{ethusd}, true), "StorePairs must enable the contract")
	_, err = ex.GetCurrencyTradeURL(t.Context(), asset.CoinMarginedFutures, ethusd)
	assert.ErrorIs(t, err, errInvalidSettlementBase, "GetCurrencyTradeURL should reject a coin-margined contract without a settlement")
}

func TestGetSettlementCurrency(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		a   asset.Item
		p   currency.Pair
		exp currency.Code
		err error
	}{
		{asset.Futures, currency.EMPTYPAIR, currency.EMPTYCODE, asset.ErrNotSupported},
		{asset.DeliveryFutures, currency.EMPTYPAIR, currency.USDT, nil},
		{asset.DeliveryFutures, getPair(t, asset.DeliveryFutures), currency.USDT, nil},
		{asset.Options, currency.EMPTYPAIR, currency.USDT, nil},
		{asset.Options, currency.NewBTCUSDT(), currency.USDT, nil},
		{asset.USDTMarginedFutures, currency.EMPTYPAIR, currency.USDT, nil},
		{asset.USDTMarginedFutures, getPair(t, asset.USDTMarginedFutures), currency.USDT, nil},
		{asset.USDTMarginedFutures, getPair(t, asset.DeliveryFutures), currency.EMPTYCODE, errInvalidSettlementQuote},

		{asset.CoinMarginedFutures, currency.EMPTYPAIR, currency.BTC, nil},
		{asset.CoinMarginedFutures, currency.Pair{Base: currency.BTC, Quote: currency.USD}, currency.BTC, nil},
		{asset.CoinMarginedFutures, currency.Pair{Base: currency.ETH, Quote: currency.USD}, currency.EMPTYCODE, errInvalidSettlementBase},
	} {
		t.Run(tt.a.String()+"_"+tt.p.String()+"_"+tt.exp.String(), func(t *testing.T) {
			t.Parallel()
			c, err := getSettlementCurrency(tt.p, tt.a)
			if tt.err == nil {
				require.NoErrorf(t, err, "getSettlementCurrency must not error for %s %s", tt.a, tt.p)
			} else {
				assert.ErrorIsf(t, err, tt.err, "getSettlementCurrency should return correct error for %s %s", tt.a, tt.p)
			}
			assert.Equalf(t, tt.exp, c, "getSettlementCurrency should return correct settlement currency for %s %s", tt.a, tt.p)
		})
	}
}

func TestDeriveSpotWebsocketOrderResponse(t *testing.T) {
	t.Parallel()
	var resp *WebsocketOrderResponse
	require.NoError(t, json.Unmarshal([]byte(`{"left":"0","update_time":"1735720637","amount":"0.0001","create_time":"1735720637","price":"0","finish_as":"filled","time_in_force":"ioc","currency_pair":"BTC_USDT","type":"market","account":"spot","side":"sell","amend_text":"-","text":"t-1735720637181634009","status":"closed","iceberg":"0","avg_deal_price":"93503.3","filled_total":"9.35033","id":"766075454481","fill_price":"9.35033","stp_id":"123456","update_time_ms":1735720637188,"create_time_ms":1735720637188}`), &resp), "unmarshal must not error")

	got, err := e.deriveSpotWebsocketOrderResponse(resp)
	require.NoError(t, err)
	assert.Equal(t, &order.SubmitResponse{
		Exchange:             e.Name,
		OrderID:              "766075454481",
		AssetType:            asset.Spot,
		Pair:                 currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
		ClientOrderID:        "t-1735720637181634009",
		Date:                 time.UnixMilli(1735720637188),
		LastUpdated:          time.UnixMilli(1735720637188),
		Amount:               0.0001,
		AverageExecutedPrice: 93503.3,
		Type:                 order.Market,
		Side:                 order.Sell,
		Status:               order.Filled,
		TimeInForce:          order.ImmediateOrCancel,
		Cost:                 9.35033,
		Purchased:            9.35033,
	}, got)
}

func TestDeriveSpotWebsocketOrderResponses(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		orders   [][]byte
		error    error
		expected []*order.SubmitResponse
	}{
		{
			name:   "no response",
			orders: [][]byte{},
			error:  common.ErrNoResponse,
		},
		{
			name: "assortment of spot orders",
			orders: [][]byte{
				[]byte(`{"left":"0","update_time":"1735720637","amount":"0.0001","create_time":"1735720637","price":"0","finish_as":"filled","time_in_force":"ioc","currency_pair":"BTC_USDT","type":"market","account":"spot","side":"sell","amend_text":"-","text":"t-1735720637181634009","status":"closed","iceberg":"0","avg_deal_price":"93503.3","filled_total":"9.35033","id":"766075454481","fill_price":"9.35033","update_time_ms":1735720637188,"create_time_ms":1735720637188}`),
				[]byte(`{"left":"0.000008","update_time":"1735720637","amount":"9.99152","create_time":"1735720637","price":"0","finish_as":"filled","time_in_force":"ioc","currency_pair":"HNS_USDT","type":"market","account":"spot","side":"buy","amend_text":"-","text":"t-1735720637126962151","status":"closed","iceberg":"0","avg_deal_price":"0.01224","filled_total":"9.991512","id":"766075454188","fill_price":"9.991512","update_time_ms":1735720637142,"create_time_ms":1735720637142}`),
				[]byte(`{"left":"0","update_time":"1735778597","amount":"200","create_time":"1735778597","price":"0.03673","finish_as":"filled","time_in_force":"fok","currency_pair":"REX_USDT","type":"limit","account":"spot","side":"buy","amend_text":"-","text":"t-1364","status":"closed","iceberg":"0","avg_deal_price":"0.03673","filled_total":"7.346","id":"766488882062","fill_price":"7.346","update_time_ms":1735778597363,"create_time_ms":1735778597363}`),
				[]byte(`{"left":"0.0003","update_time":"1735780321","amount":"0.0003","create_time":"1735780321","price":"20000","finish_as":"open","time_in_force":"poc","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"buy","amend_text":"-","text":"t-1735780321603944400","status":"open","iceberg":"0","filled_total":"0","id":"766504537761","fill_price":"0","update_time_ms":1735780321729,"create_time_ms":1735780321729}`),
				[]byte(`{"left":"1","update_time":"1735784755","amount":"1","create_time":"1735784755","price":"100","finish_as":"open","time_in_force":"gtc","currency_pair":"GT_USDT","type":"limit","account":"spot","side":"sell","amend_text":"-","text":"t-1735784754905434100","status":"open","iceberg":"0","filled_total":"0","id":"766536556747","fill_price":"0","update_time_ms":1735784755068,"create_time_ms":1735784755068}`),
			},
			expected: []*order.SubmitResponse{
				{
					Exchange:             e.Name,
					OrderID:              "766075454481",
					AssetType:            asset.Spot,
					Pair:                 currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-1735720637181634009",
					Date:                 time.UnixMilli(1735720637188),
					LastUpdated:          time.UnixMilli(1735720637188),
					Amount:               0.0001,
					AverageExecutedPrice: 93503.3,
					Type:                 order.Market,
					Side:                 order.Sell,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
					Cost:                 9.35033,
					Purchased:            9.35033,
				},
				{
					Exchange:             e.Name,
					OrderID:              "766075454188",
					AssetType:            asset.Spot,
					Pair:                 currency.NewPair(currency.HNS, currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-1735720637126962151",
					Date:                 time.UnixMilli(1735720637142),
					LastUpdated:          time.UnixMilli(1735720637142),
					RemainingAmount:      0.000008,
					Amount:               9.99152,
					AverageExecutedPrice: 0.01224,
					Type:                 order.Market,
					Side:                 order.Buy,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
					Cost:                 9.991512,
					Purchased:            816.3,
				},
				{
					Exchange:             e.Name,
					OrderID:              "766488882062",
					AssetType:            asset.Spot,
					Pair:                 currency.NewPair(currency.NewCode("REX"), currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-1364",
					Date:                 time.UnixMilli(1735778597363),
					LastUpdated:          time.UnixMilli(1735778597363),
					Amount:               200,
					Price:                0.03673,
					AverageExecutedPrice: 0.03673,
					Type:                 order.Limit,
					Side:                 order.Buy,
					Status:               order.Filled,
					TimeInForce:          order.FillOrKill,
					Cost:                 7.346,
					Purchased:            200,
				},
				{
					Exchange:        e.Name,
					OrderID:         "766504537761",
					AssetType:       asset.Spot,
					Pair:            currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:   "t-1735780321603944400",
					Date:            time.UnixMilli(1735780321729),
					LastUpdated:     time.UnixMilli(1735780321729),
					RemainingAmount: 0.0003,
					Amount:          0.0003,
					Price:           20000,
					Type:            order.Limit,
					Side:            order.Buy,
					Status:          order.Open,
					TimeInForce:     order.PostOnly,
				},
				{
					Exchange:        e.Name,
					OrderID:         "766536556747",
					AssetType:       asset.Spot,
					Pair:            currency.NewPair(currency.NewCode("GT"), currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:   "t-1735784754905434100",
					Date:            time.UnixMilli(1735784755068),
					LastUpdated:     time.UnixMilli(1735784755068),
					RemainingAmount: 1,
					Amount:          1,
					Price:           100,
					Type:            order.Limit,
					Side:            order.Sell,
					Status:          order.Open,
					TimeInForce:     order.GoodTillCancel,
				},
			},
		},
		{
			// Gate's documented market buy reply, which gives the BTC bought as filled_amount, and a sell
			name: "orders with their filled amounts",
			orders: [][]byte{
				[]byte(`{"id":"12332324","text":"t-123456","amend_text":"-","create_time":"1734081653","update_time":"1734081653","create_time_ms":1734081653247,"update_time_ms":1734081653249,"status":"closed","currency_pair":"BTC_USDT","type":"market","account":"spot","side":"buy","amount":"70.0","price":"0.0","time_in_force":"ioc","iceberg":"0.0","left":"0.143511","filled_amount":"0.00299","fill_price":"69.856489","filled_total":"69.856489","avg_deal_price":"23363.3742475","fee":"0.0000028704","fee_currency":"BTC","finish_as":"filled"}`),
				[]byte(`{"id":"12332325","text":"t-123457","create_time_ms":1734081653250,"update_time_ms":1734081653251,"status":"open","currency_pair":"BTC_USDT","type":"limit","account":"spot","side":"sell","amount":"1","price":"60000","time_in_force":"gtc","left":"0.4","filled_amount":"0.6","filled_total":"36000","avg_deal_price":"60000","fee":"36","fee_currency":"USDT","finish_as":"open"}`),
			},
			expected: []*order.SubmitResponse{
				{
					Exchange:             e.Name,
					OrderID:              "12332324",
					AssetType:            asset.Spot,
					Pair:                 currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-123456",
					Date:                 time.UnixMilli(1734081653247),
					LastUpdated:          time.UnixMilli(1734081653249),
					RemainingAmount:      0.143511,
					Amount:               70,
					AverageExecutedPrice: 23363.3742475,
					Type:                 order.Market,
					Side:                 order.Buy,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
					Fee:                  0.0000028704,
					FeeAsset:             currency.BTC,
					Cost:                 69.856489,
					Purchased:            0.00299,
				},
				{
					Exchange:             e.Name,
					OrderID:              "12332325",
					AssetType:            asset.Spot,
					Pair:                 currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-123457",
					Date:                 time.UnixMilli(1734081653250),
					LastUpdated:          time.UnixMilli(1734081653251),
					RemainingAmount:      0.4,
					Amount:               1,
					Price:                60000,
					AverageExecutedPrice: 60000,
					Type:                 order.Limit,
					Side:                 order.Sell,
					Status:               order.Open,
					TimeInForce:          order.GoodTillCancel,
					Fee:                  36,
					FeeAsset:             currency.USDT,
					Cost:                 36000,
					Purchased:            36000, // A sell's purchase is in the quote currency, so filled_amount does not apply
				},
			},
		},
		{
			name: "batch of spot orders with error at end",
			// This is specifically testing the return responses of WebsocketSpotSubmitOrders
			// AverageDealPrice is not returned when using this endpoint so purchased and cost fields cannot be set.
			orders: [][]byte{
				[]byte(`{"account":"spot","status":"closed","side":"buy","amount":"9.98","id":"775453816782","create_time":"1736980695","update_time":"1736980695","text":"t-740","left":"0.047239","currency_pair":"BTC_USDT","type":"market","finish_as":"filled","price":"0","time_in_force":"fok","iceberg":"0","filled_total":"9.932761","fill_price":"9.932761","create_time_ms":1736980695949,"update_time_ms":1736980695949,"succeeded":true}`),
				[]byte(`{"account":"spot","status":"closed","side":"buy","amount":"0.00289718","id":"775453816824","create_time":"1736980695","update_time":"1736980695","text":"t-741","left":"0.00000000962","currency_pair":"BTC_USDT","type":"market","finish_as":"filled","price":"0","time_in_force":"fok","iceberg":"0","filled_total":"0.00289717038","fill_price":"0.00289717038","create_time_ms":1736980695956,"update_time_ms":1736980695956,"succeeded":true}`),
				[]byte(`{"text":"t-742","label":"BALANCE_NOT_ENOUGH","message":"Not enough balance"}`),
			},
			expected: []*order.SubmitResponse{
				{
					Exchange:        e.Name,
					OrderID:         "775453816782",
					AssetType:       asset.Spot,
					Pair:            getPair(t, asset.Spot).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:   "t-740",
					Date:            time.UnixMilli(1736980695949),
					LastUpdated:     time.UnixMilli(1736980695949),
					Amount:          9.98,
					RemainingAmount: 0.047239,
					Type:            order.Market,
					Side:            order.Buy,
					Status:          order.Filled,
					TimeInForce:     order.FillOrKill,
				},
				{
					Exchange:        e.Name,
					OrderID:         "775453816824",
					AssetType:       asset.Spot,
					Pair:            getPair(t, asset.Spot).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:   "t-741",
					Date:            time.UnixMilli(1736980695956),
					LastUpdated:     time.UnixMilli(1736980695956),
					RemainingAmount: 0.00000000962,
					Amount:          0.00289718,
					Type:            order.Market,
					Side:            order.Buy,
					Status:          order.Filled,
					TimeInForce:     order.FillOrKill,
				},
				{
					Exchange:        e.Name,
					ClientOrderID:   "t-742",
					SubmissionError: order.ErrUnableToPlaceOrder,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orders := bytes.Join(tc.orders, []byte(","))
			orders = append([]byte("["), append(orders, []byte("]")...)...)

			var resp []*WebsocketOrderResponse
			require.NoError(t, json.Unmarshal(orders, &resp), "unmarshal must not error")

			got, err := e.deriveSpotWebsocketOrderResponses(resp)
			require.ErrorIs(t, err, tc.error)

			require.Len(t, got, len(tc.expected))
			for i := range got {
				if tc.expected[i].SubmissionError != nil {
					assert.ErrorIs(t, got[i].SubmissionError, tc.expected[i].SubmissionError)
					assert.Equal(t, tc.expected[i].Exchange, got[i].Exchange)
					assert.Equal(t, tc.expected[i].ClientOrderID, got[i].ClientOrderID)
					continue
				}
				assert.Equal(t, tc.expected[i], got[i])
			}
		})
	}
}

func TestDeriveFuturesWebsocketOrderResponse(t *testing.T) {
	t.Parallel()

	var resp *WebsocketFuturesOrderResponse
	require.NoError(t, json.Unmarshal([]byte(`{"text":"t-1337","price":"0","biz_info":"-","tif":"ioc","amend_text":"-","status":"finished","contract":"CWIF_USDT","stp_act":"-","stp_id":"123456","finish_as":"filled","fill_price":"0.0000002625","id":596729318437,"create_time":1735787107.449,"size":2,"finish_time":1735787107.45,"update_time":1735787107.45,"left":0,"user":12870774,"is_reduce_only":true}`), &resp), "unmarshal must not error")

	got, err := e.deriveFuturesWebsocketOrderResponse(resp, asset.USDTMarginedFutures)
	require.NoError(t, err)
	assert.Equal(t, &order.SubmitResponse{
		Exchange:             e.Name,
		OrderID:              "596729318437",
		AssetType:            asset.USDTMarginedFutures,
		Pair:                 currency.NewPair(currency.NewCode("CWIF"), currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
		ClientOrderID:        "t-1337",
		Date:                 time.UnixMilli(1735787107449),
		LastUpdated:          time.UnixMilli(1735787107450),
		Amount:               2,
		AverageExecutedPrice: 0.0000002625,
		Type:                 order.Market,
		Side:                 order.Long,
		Status:               order.Filled,
		TimeInForce:          order.ImmediateOrCancel,
		ReduceOnly:           true,
	}, got)

	// Pairing a USDT contract with a coin-margined caller proves the asset comes from the caller, not the contract.
	got, err = e.deriveFuturesWebsocketOrderResponse(resp, asset.CoinMarginedFutures)
	require.NoError(t, err, "deriveFuturesWebsocketOrderResponse must not error")
	require.NotNil(t, got, "response must not be nil")
	assert.Equal(t, asset.CoinMarginedFutures, got.AssetType, "AssetType should follow the caller")
}

func TestDeriveFuturesWebsocketOrderResponses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		orders   [][]byte
		error    error
		expected []*order.SubmitResponse
	}{
		{
			name:   "no response",
			orders: [][]byte{},
			error:  common.ErrNoResponse,
		},
		{
			name: "assortment of futures orders",
			orders: [][]byte{
				[]byte(`{"text":"t-1337","price":"0","biz_info":"-","tif":"ioc","amend_text":"-","status":"finished","contract":"CWIF_USDT","stp_act":"-","finish_as":"filled","fill_price":"0.0000002625","id":596729318437,"create_time":1735787107.449,"size":2,"finish_time":1735787107.45,"update_time":1735787107.45,"left":0,"user":12870774,"is_reduce_only":true}`),
				[]byte(`{"text":"t-1336","price":"0","biz_info":"-","tif":"ioc","amend_text":"-","status":"finished","contract":"REX_USDT","stp_act":"-","finish_as":"filled","fill_price":"0.03654","id":596662040388,"create_time":1735778597.374,"size":-2,"finish_time":1735778597.374,"update_time":1735778597.374,"left":0,"user":12870774}`),
				[]byte(`{"text":"apiv4-ws","price":"40000","biz_info":"-","tif":"gtc","amend_text":"-","status":"open","contract":"BTC_USDT","stp_act":"-","fill_price":"0","id":596746193678,"create_time":1735789790.476,"size":1,"update_time":1735789790.476,"left":1,"user":2365748}`),
				[]byte(`{"text":"apiv4-ws","price":"200000","biz_info":"-","tif":"gtc","amend_text":"-","status":"open","contract":"BTC_USDT","stp_act":"-","fill_price":"0","id":596748780649,"create_time":1735790222.185,"size":-1,"update_time":1735790222.185,"left":-1,"user":2365748}`),
				[]byte(`{"text":"apiv4-ws","price":"0","biz_info":"-","tif":"ioc","amend_text":"-","status":"finished","contract":"BTC_USDT","stp_act":"-","finish_as":"filled","fill_price":"98172.9","id":36028797827161124,"create_time":1740108860.761,"size":1,"finish_time":1740108860.761,"update_time":1740108860.761,"left":0,"user":2365748}`),
				[]byte(`{"text":"apiv4-ws","price":"0","biz_info":"-","tif":"ioc","amend_text":"-","status":"finished","contract":"BTC_USDT","stp_act":"-","finish_as":"filled","fill_price":"98113.1","id":36028797827225781,"create_time":1740109172.06,"size":-1,"finish_time":1740109172.06,"update_time":1740109172.06,"left":0,"user":2365748,"is_reduce_only":true}`),
			},
			expected: []*order.SubmitResponse{
				{
					Exchange:             e.Name,
					OrderID:              "596729318437",
					AssetType:            asset.USDTMarginedFutures,
					Pair:                 currency.NewPair(currency.NewCode("CWIF"), currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-1337",
					Date:                 time.UnixMilli(1735787107449),
					LastUpdated:          time.UnixMilli(1735787107450),
					Amount:               2,
					AverageExecutedPrice: 0.0000002625,
					Type:                 order.Market,
					Side:                 order.Long,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
					ReduceOnly:           true,
				},
				{
					Exchange:             e.Name,
					OrderID:              "596662040388",
					AssetType:            asset.USDTMarginedFutures,
					Pair:                 currency.NewPair(currency.NewCode("REX"), currency.USDT).Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:        "t-1336",
					Date:                 time.UnixMilli(1735778597374),
					LastUpdated:          time.UnixMilli(1735778597374),
					Amount:               2,
					AverageExecutedPrice: 0.03654,
					Type:                 order.Market,
					Side:                 order.Short,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
				},
				{
					Exchange:        e.Name,
					OrderID:         "596746193678",
					AssetType:       asset.USDTMarginedFutures,
					Pair:            currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					Date:            time.UnixMilli(1735789790476),
					LastUpdated:     time.UnixMilli(1735789790476),
					RemainingAmount: 1,
					Amount:          1,
					Price:           40000,
					Type:            order.Limit,
					Side:            order.Long,
					Status:          order.Open,
					TimeInForce:     order.GoodTillCancel,
				},
				{
					Exchange:        e.Name,
					OrderID:         "596748780649",
					AssetType:       asset.USDTMarginedFutures,
					Pair:            currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					Date:            time.UnixMilli(1735790222185),
					LastUpdated:     time.UnixMilli(1735790222185),
					RemainingAmount: 1,
					Amount:          1,
					Price:           200000,
					Type:            order.Limit,
					Side:            order.Short,
					Status:          order.Open,
					TimeInForce:     order.GoodTillCancel,
				},
				{
					Exchange:             e.Name,
					OrderID:              "36028797827161124",
					AssetType:            asset.USDTMarginedFutures,
					Pair:                 currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					Date:                 time.UnixMilli(1740108860761),
					LastUpdated:          time.UnixMilli(1740108860761),
					Amount:               1,
					AverageExecutedPrice: 98172.9,
					Type:                 order.Market,
					Side:                 order.Long,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
				},
				{
					Exchange:             e.Name,
					OrderID:              "36028797827225781",
					AssetType:            asset.USDTMarginedFutures,
					Pair:                 currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					Date:                 time.UnixMilli(1740109172060),
					LastUpdated:          time.UnixMilli(1740109172060),
					Amount:               1,
					AverageExecutedPrice: 98113.1,
					Type:                 order.Market,
					Side:                 order.Short,
					Status:               order.Filled,
					TimeInForce:          order.ImmediateOrCancel,
					ReduceOnly:           true,
				},
			},
		},
		{
			// Gate rejects an order of a batch on its own, reporting why in its place
			name: "batch of futures orders with a rejection",
			orders: [][]byte{
				[]byte(`{"succeeded":true,"id":74046545,"user":6790020,"create_time":1681196536.592,"status":"open","contract":"BTC_USDT","size":"10","price":"31403.2","tif":"gtc","left":"10","fill_price":"0","text":"t-my-custom-id","tkfr":"0.0003","mkfr":"0"}`),
				[]byte(`{"succeeded":false,"label":"INSUFFICIENT_AVAILABLE","detail":"insufficient available balance"}`),
			},
			expected: []*order.SubmitResponse{
				{
					Exchange:        e.Name,
					OrderID:         "74046545",
					AssetType:       asset.USDTMarginedFutures,
					Pair:            currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: "_"}),
					ClientOrderID:   "t-my-custom-id",
					Date:            time.UnixMilli(1681196536592),
					LastUpdated:     time.UnixMilli(1681196536592),
					RemainingAmount: 10,
					Amount:          10,
					Price:           31403.2,
					Type:            order.Limit,
					Side:            order.Long,
					Status:          order.Open,
					TimeInForce:     order.GoodTillCancel,
				},
				{
					Exchange:        e.Name,
					AssetType:       asset.USDTMarginedFutures,
					SubmissionError: fmt.Errorf("%w reason label:%q detail:%q", order.ErrUnableToPlaceOrder, "INSUFFICIENT_AVAILABLE", "insufficient available balance"),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orders := bytes.Join(tc.orders, []byte(","))
			orders = append([]byte("["), append(orders, []byte("]")...)...)

			var resp []*WebsocketFuturesOrderResponse
			require.NoError(t, json.Unmarshal(orders, &resp), "unmarshal must not error")

			got, err := e.deriveFuturesWebsocketOrderResponses(resp, asset.USDTMarginedFutures)
			require.ErrorIs(t, err, tc.error)

			require.Len(t, got, len(tc.expected))
			for i := range got {
				assert.Equal(t, tc.expected[i], got[i])
			}
		})
	}
}

var (
	pairMap    = map[asset.Item]currency.Pairs{}
	pairsGuard sync.RWMutex
)

func getPairWithOpenInterest(t *testing.T, a asset.Item, pairs currency.Pairs) (key.PairAsset, []futures.OpenInterest) {
	t.Helper()
	var lastErr error
	var receivedResponse bool
	for _, pair := range pairs {
		pairAsset := key.PairAsset{Base: pair.Base.Item, Quote: pair.Quote.Item, Asset: a}
		response, err := e.GetOpenInterest(t.Context(), pairAsset)
		if err != nil {
			lastErr = err
			continue
		}
		receivedResponse = true
		if len(response) == 1 && response[0].OpenInterest > 0 {
			return pairAsset, response
		}
	}
	switch {
	case !receivedResponse:
		t.Fatalf("GetOpenInterest must find a live pair for %s asset: every request failed, last error: %v", a, lastErr)
	case lastErr != nil:
		t.Fatalf("GetOpenInterest must find a live pair with positive open interest for %s asset: no pair returned positive open interest and some requests failed, last error: %v", a, lastErr)
	default:
		t.Fatalf("GetOpenInterest must find a live pair with positive open interest for %s asset: no request errored, but no pair returned positive open interest", a)
	}
	return key.PairAsset{}, nil
}

// getMarketDataPair returns the instance and pair a market data test reads. Live delivery tests use the most active
// contract with the data the test asserts on, rather than the enabled one, because most of Gate's delivery contracts can
// go hours without a trade, often with one side of the book empty, and the enabled contract is picked at random once the
// configured one expires. That contract gets an instance of its own because candle requests need it enabled, and
// enabling it on e would not last: testexch.UpdatePairsOnce reloads e's cached pairs. Mock tests read the recorded pair
func getMarketDataPair(t *testing.T, ex *Exchange, a asset.Item, hasData deliveryDataCheck) (*Exchange, currency.Pair) {
	t.Helper()
	if mockTests || a != asset.DeliveryFutures {
		return ex, getPair(t, a)
	}
	pair := getActiveDeliveryPair(t, hasData)
	deliveryEx := new(Exchange)
	require.NoError(t, testexch.Setup(deliveryEx), "Setup must not error")
	deliveryEx.Name = t.Name()
	require.NoError(t, deliveryEx.CurrencyPairs.StorePairs(a, currency.Pairs{pair}, false), "StorePairs must not error for available pairs")
	require.NoError(t, deliveryEx.CurrencyPairs.StorePairs(a, currency.Pairs{pair}, true), "StorePairs must not error for enabled pairs")
	return deliveryEx, pair
}

// deliveryDataCheck returns why a delivery contract lacks the data a test asserts on, or nil if it has it. Each test
// checks only its own need, because a contract with a two-sided book can go hours without a trade while another trades
// with one side of its book empty
type deliveryDataCheck func(context.Context, currency.Pair) error

func hasTwoSidedDeliveryBook(ctx context.Context, p currency.Pair) error {
	book, err := e.GetDeliveryOrderbook(ctx, currency.USDT, "", p, 1, false)
	switch {
	case err != nil:
		return err
	case book == nil || len(book.Bids) == 0:
		return errors.New("no bids")
	case len(book.Asks) == 0:
		return errors.New("no asks")
	}
	return nil
}

// hasDeliveryCandles checks with the test's own interval and window, because Gate returns no candles unless a trade
// falls in a candle the window overlaps, and a contract's 24-hour volume does not show when it traded
func hasDeliveryCandles(interval kline.Interval, start, end time.Time) deliveryDataCheck {
	return func(ctx context.Context, p currency.Pair) error {
		candles, err := e.GetDeliveryFuturesCandlesticks(ctx, currency.USDT, p, start, end, 0, interval)
		if err == nil && len(candles) == 0 {
			return errors.New("no candles")
		}
		return err
	}
}

// getActiveDeliveryPair returns the contract with the most 24-hour volume that passes hasData. It selects afresh on
// each call, as each test checks for different data and a contract's book and recent trades can change between tests
func getActiveDeliveryPair(tb testing.TB, hasData deliveryDataCheck) currency.Pair {
	tb.Helper()
	tickers, err := e.GetDeliveryFutureTickers(tb.Context(), currency.USDT, currency.EMPTYPAIR)
	require.NoError(tb, err, "GetDeliveryFutureTickers must not error")
	slices.SortFunc(tickers, func(a, b *FuturesTickerResponse) int {
		return cmp.Compare(b.Volume24HourQuote.Float64(), a.Volume24HourQuote.Float64())
	})
	// A contract can expire mid-selection, so a failed request moves on to the next one
	var errs error
	for i := range tickers {
		if err := hasData(tb.Context(), tickers[i].Contract); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%s: %w", tickers[i].Contract, err))
			continue
		}
		return tickers[i].Contract
	}
	require.FailNowf(tb, "must find a delivery contract with the data this test asserts on", "%d contracts checked: %v", len(tickers), errs)
	return currency.EMPTYPAIR
}

func getPairs(tb testing.TB, a asset.Item) currency.Pairs {
	tb.Helper()
	pairsGuard.RLock()
	p, ok := pairMap[a]
	pairsGuard.RUnlock()
	if ok {
		return p
	}
	pairsGuard.Lock()
	defer pairsGuard.Unlock()
	p, ok = pairMap[a] // Protect Race if we blocked on Lock and another RW populated
	if ok {
		return p
	}

	if !mockTests {
		testexch.UpdatePairsOnce(tb, e)
	}
	enabledPairs, err := e.GetEnabledPairs(a)
	assert.NoErrorf(tb, err, "%s GetEnabledPairs should not error", a)
	if !assert.NotEmptyf(tb, enabledPairs, "%s GetEnabledPairs should not be empty", a) {
		tb.Fatalf("No pair available for asset %s", a)
		return nil
	}
	pairMap[a] = enabledPairs

	return enabledPairs
}

func TestGetTypeFromTimeInForce(t *testing.T) {
	t.Parallel()
	typeResp := getTypeFromTimeInForce("gtc", 0)
	assert.Equal(t, order.Limit, typeResp)

	typeResp = getTypeFromTimeInForce("ioc", 0)
	assert.Equal(t, order.Market, typeResp, "should be market order")

	typeResp = getTypeFromTimeInForce("poc", 123)
	assert.Equal(t, order.Limit, typeResp, "should be limit order")

	typeResp = getTypeFromTimeInForce("fok", 0)
	assert.Equal(t, order.Market, typeResp, "should be market order")
}

// TestOrderStatusFromFinishAs covers every finish_as value Gate documents for spot, futures, delivery and options orders
func TestOrderStatusFromFinishAs(t *testing.T) {
	t.Parallel()
	for finishAs, exp := range map[string]order.Status{
		"":                        order.Open,
		"open":                    order.Open,
		"filled":                  order.Filled,
		"cancelled":               order.Cancelled,
		"liquidated":              order.Liquidated,
		"liquidate_cancelled":     order.Cancelled,
		"auto_deleveraged":        order.AutoDeleverage,
		"auto_deleveraging":       order.AutoDeleverage,
		"position_closed":         order.Closed,
		"position_close":          order.Closed,
		"unified_check_failed":    order.Cancelled,
		"-":                       order.UnknownStatus,
		"stp":                     order.STP,
		"ioc":                     order.Cancelled,
		"poc":                     order.Cancelled,
		"fok":                     order.Cancelled,
		"small":                   order.Cancelled,
		"depth_not_enough":        order.Cancelled,
		"trader_not_enough":       order.Cancelled,
		"price_protect_cancelled": order.Cancelled,
		"reduce_only":             order.Cancelled,
		"reduce_out":              order.Cancelled,
		"mmp_cancelled":           order.Cancelled,
		"unknown":                 order.UnknownStatus,
	} {
		got, err := orderStatusFromFinishAs(finishAs)
		require.NoErrorf(t, err, "orderStatusFromFinishAs must not error for %q", finishAs)
		assert.Equalf(t, exp, got, "orderStatusFromFinishAs should return the status for %q", finishAs)
	}
	_, err := orderStatusFromFinishAs("stopped")
	assert.ErrorContains(t, err, "STOPPED", "orderStatusFromFinishAs should reject an undocumented value")
}

func TestFuturesOrderStatus(t *testing.T) {
	t.Parallel()
	got, err := futuresOrderStatus(statusOpen, "filled")
	require.NoError(t, err, "futuresOrderStatus must not error for an open order")
	assert.Equal(t, order.Open, got, "futuresOrderStatus should report an open order as open")

	got, err = futuresOrderStatus(statusFinished, "ioc")
	require.NoError(t, err, "futuresOrderStatus must not error for a finished order")
	assert.Equal(t, order.Cancelled, got, "futuresOrderStatus should report how a finished order finished")
}

func TestOrderLastUpdated(t *testing.T) {
	t.Parallel()
	updated, finished, created := types.Time(time.Unix(3, 0)), types.Time(time.Unix(2, 0)), types.Time(time.Unix(1, 0))
	assert.Equal(t, time.Unix(3, 0), orderLastUpdated(updated, finished, created), "orderLastUpdated should prefer the update time")
	assert.Equal(t, time.Unix(2, 0), orderLastUpdated(types.Time{}, finished, created), "orderLastUpdated should fall back to the finish time")
	assert.Equal(t, time.Unix(1, 0), orderLastUpdated(types.Time{}, types.Time{}, created), "orderLastUpdated should fall back to the creation time")
	assert.Zero(t, orderLastUpdated(types.Time{}, types.Time{}, types.Time{}), "orderLastUpdated should return a zero time without any time")
}

func TestSpotOrderDetail(t *testing.T) {
	t.Parallel()
	created, updated := time.UnixMilli(1744103854944), time.UnixMilli(1744103855944)
	pair := currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
	newOrder := func() *SpotOrderResponse {
		return &SpotOrderResponse{
			OrderID:          "1852454420",
			Text:             "t-1852454420",
			Side:             "buy",
			Type:             "limit",
			Status:           "open",
			Price:            65000,
			Amount:           0.75,
			RemainingAmount:  0.25,
			AverageFillPrice: 64990,
			FilledTotal:      32495,
			FeeDeducted:      0.0005,
			FeeCurrency:      currency.BTC,
			CreateTime:       types.Time(created),
			UpdateTime:       types.Time(updated),
		}
	}
	got, err := e.spotOrderDetail(newOrder(), pair, asset.Margin)
	require.NoError(t, err, "spotOrderDetail must not error")
	exp := &order.Detail{
		Exchange:             e.Name,
		OrderID:              "1852454420",
		ClientOrderID:        "t-1852454420",
		Side:                 order.Buy,
		Type:                 order.Limit,
		Status:               order.Open,
		Pair:                 pair,
		AssetType:            asset.Margin,
		Price:                65000,
		Amount:               0.75,
		ExecutedAmount:       0.5,
		RemainingAmount:      0.25,
		AverageExecutedPrice: 64990,
		Cost:                 32495,
		CostAsset:            currency.USDT,
		Fee:                  0.0005,
		FeeAsset:             currency.BTC,
		Date:                 created,
		LastUpdated:          updated,
	}
	assert.Equal(t, exp, got, "spotOrderDetail should convert every field")

	unified := newOrder()
	unified.TradeQuote = currency.USDC
	got, err = e.spotOrderDetail(unified, pair, asset.Margin)
	require.NoError(t, err, "spotOrderDetail must not error for a unified account order")
	exp.CostAsset = currency.USDC
	assert.Equal(t, exp, got, "spotOrderDetail should report the cost in the quote currency the order traded in")

	badSide := newOrder()
	badSide.Side = "sideways"
	_, err = e.spotOrderDetail(badSide, pair, asset.Spot)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "spotOrderDetail should reject an unknown side")

	badType := newOrder()
	badType.Type = "pegged"
	_, err = e.spotOrderDetail(badType, pair, asset.Spot)
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "spotOrderDetail should reject an unknown type")

	badStatus := newOrder()
	badStatus.Status = "stopped"
	_, err = e.spotOrderDetail(badStatus, pair, asset.Spot)
	assert.ErrorContains(t, err, "STOPPED", "spotOrderDetail should reject an unknown status")
}

func TestFuturesOrderDetail(t *testing.T) {
	t.Parallel()
	created, updated, finished := time.Unix(1744103854, 0), time.Unix(1744103855, 0), time.Unix(1744103856, 0)
	contract := currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
	format := currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	for _, tc := range []struct {
		name   string
		a      asset.Item
		settle currency.Code
		order  *FuturesOrderResponse
		exp    *order.Detail
	}{
		{
			name:   "cancelled short limit",
			a:      asset.USDTMarginedFutures,
			settle: currency.USDT,
			order: &FuturesOrderResponse{
				ID:              15675394,
				Contract:        contract,
				CreateTime:      types.Time(created),
				FinishTime:      types.Time(finished),
				Size:            -100,
				RemainingAmount: -25,
				Price:           65000,
				FillPrice:       64990,
				TimeInForce:     gtcTIF,
				IsReduceOnly:    true,
				Text:            "t-15675394",
				Status:          statusFinished,
				FinishAs:        "cancelled",
			},
			exp: &order.Detail{
				Exchange:             e.Name,
				OrderID:              "15675394",
				ClientOrderID:        "t-15675394",
				Side:                 order.Short,
				Type:                 order.Limit,
				Status:               order.Cancelled,
				TimeInForce:          order.GoodTillCancel,
				ReduceOnly:           true,
				Pair:                 currency.NewPairWithDelimiter("BTC", "USDT", currency.DashDelimiter),
				AssetType:            asset.USDTMarginedFutures,
				SettlementCurrency:   currency.USDT,
				Price:                65000,
				Amount:               100,
				ContractAmount:       100,
				ExecutedAmount:       75,
				RemainingAmount:      25,
				AverageExecutedPrice: 64990,
				Date:                 created,
				LastUpdated:          finished,
			},
		},
		{
			name:   "open long market",
			a:      asset.DeliveryFutures,
			settle: currency.USDT,
			order: &FuturesOrderResponse{
				ID:              15675395,
				Contract:        contract,
				CreateTime:      types.Time(created),
				UpdateTime:      types.Time(updated),
				Size:            10,
				RemainingAmount: 10,
				TimeInForce:     iocTIF,
				Text:            "web",
				Status:          statusOpen,
			},
			exp: &order.Detail{
				Exchange:           e.Name,
				OrderID:            "15675395",
				Side:               order.Long,
				Type:               order.Market,
				Status:             order.Open,
				TimeInForce:        order.ImmediateOrCancel,
				Pair:               currency.NewPairWithDelimiter("BTC", "USDT", currency.DashDelimiter),
				AssetType:          asset.DeliveryFutures,
				SettlementCurrency: currency.USDT,
				Amount:             10,
				ContractAmount:     10,
				RemainingAmount:    10,
				Date:               created,
				LastUpdated:        updated,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.futuresOrderDetail(tc.order, tc.a, tc.settle, format)
			require.NoError(t, err, "futuresOrderDetail must not error")
			assert.Equal(t, tc.exp, got, "futuresOrderDetail should convert every field")
		})
	}

	_, err := e.futuresOrderDetail(&FuturesOrderResponse{Status: statusFinished, FinishAs: "stopped"}, asset.USDTMarginedFutures, currency.USDT, format)
	assert.ErrorContains(t, err, "STOPPED", "futuresOrderDetail should reject an unknown finish_as")

	_, err = e.futuresOrderDetail(&FuturesOrderResponse{Status: statusOpen, TimeInForce: "gtd"}, asset.USDTMarginedFutures, currency.USDT, format)
	assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "futuresOrderDetail should reject an unknown time in force")
}

func TestOptionsOrderDetail(t *testing.T) {
	t.Parallel()
	created, finished := time.Unix(1744103854, 0), time.Unix(1744103856, 0)
	got, err := e.optionsOrderDetail(&OptionOrderResponse{
		OptionOrderID: 2187,
		Contract:      "BTC_USDT-20260925-75000-P",
		CreateTime:    types.Time(created),
		FinishTime:    types.Time(finished),
		Status:        statusFinished,
		FinishAs:      "filled",
		Size:          -2,
		Price:         1500,
		FillPrice:     1490,
		TimeInForce:   iocTIF,
		IsReduceOnly:  true,
		Text:          "t-2187",
	})
	require.NoError(t, err, "optionsOrderDetail must not error")
	exp := &order.Detail{
		Exchange:             e.Name,
		OrderID:              "2187",
		ClientOrderID:        "t-2187",
		Side:                 order.Short,
		Type:                 order.Limit,
		Status:               order.Filled,
		TimeInForce:          order.ImmediateOrCancel,
		ReduceOnly:           true,
		Pair:                 currency.NewPairWithDelimiter("BTC", "USDT-20260925-75000-P", currency.UnderscoreDelimiter),
		AssetType:            asset.Options,
		Price:                1500,
		Amount:               2,
		ExecutedAmount:       2,
		AverageExecutedPrice: 1490,
		Date:                 created,
		LastUpdated:          finished,
	}
	assert.Equal(t, exp, got, "optionsOrderDetail should convert every field")

	_, err = e.optionsOrderDetail(&OptionOrderResponse{Status: statusOpen})
	assert.ErrorIs(t, err, currency.ErrCreatingPair, "optionsOrderDetail should reject a missing contract")

	_, err = e.optionsOrderDetail(&OptionOrderResponse{Contract: "BTC_USDT-20260925-75000-P", Status: statusFinished, FinishAs: "stopped"})
	assert.ErrorContains(t, err, "STOPPED", "optionsOrderDetail should reject an unknown finish_as")

	_, err = e.optionsOrderDetail(&OptionOrderResponse{Contract: "BTC_USDT-20260925-75000-P", Status: statusOpen, TimeInForce: "gtd"})
	assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "optionsOrderDetail should reject an unknown time in force")
}

func TestDeriveFuturesSubmitResponse(t *testing.T) {
	t.Parallel()
	created, updated := time.Unix(1744103854, 0), time.Unix(1744103855, 0)
	pair := currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
	s := &order.Submit{
		Exchange:      e.Name,
		Type:          order.Limit,
		Side:          order.Short,
		Pair:          pair,
		AssetType:     asset.USDTMarginedFutures,
		TimeInForce:   order.GoodTillCancel,
		Price:         65000,
		Amount:        100,
		ClientOrderID: "t-15675394",
	}
	placed := &FuturesOrderResponse{
		ID:              15675394,
		Contract:        pair,
		CreateTime:      types.Time(created),
		UpdateTime:      types.Time(updated),
		Size:            -100,
		RemainingAmount: -40,
		Price:           65000,
		FillPrice:       64995,
		TimeInForce:     gtcTIF,
		Text:            "t-15675394",
		Status:          statusOpen,
	}
	got, err := deriveFuturesSubmitResponse(s, placed)
	require.NoError(t, err, "deriveFuturesSubmitResponse must not error")
	exp := &order.SubmitResponse{
		Exchange:             e.Name,
		Type:                 order.Limit,
		Side:                 order.Short,
		Pair:                 pair,
		AssetType:            asset.USDTMarginedFutures,
		TimeInForce:          order.GoodTillCancel,
		Price:                65000,
		Amount:               100,
		RemainingAmount:      40,
		AverageExecutedPrice: 64995,
		ClientOrderID:        "t-15675394",
		OrderID:              "15675394",
		Status:               order.Open,
		Date:                 created,
		LastUpdated:          updated,
	}
	assert.Equal(t, exp, got, "deriveFuturesSubmitResponse should report the placed order")

	_, err = deriveFuturesSubmitResponse(s, &FuturesOrderResponse{ID: 15675394, Status: statusFinished, FinishAs: "stopped"})
	assert.ErrorContains(t, err, "STOPPED", "deriveFuturesSubmitResponse should reject an unknown finish_as")
}

func TestToExchangeTIF(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		tif      order.TimeInForce
		price    float64
		expected string
		err      error
	}{
		{price: 0, expected: iocTIF}, // market orders default to IOC
		{price: 0, tif: order.FillOrKill, expected: fokTIF},
		{price: 420, expected: gtcTIF}, // limit orders default to GTC
		{price: 420, tif: order.GoodTillCancel, expected: gtcTIF},
		{price: 420, tif: order.ImmediateOrCancel, expected: iocTIF},
		{price: 420, tif: order.PostOnly, expected: pocTIF},
		{price: 420, tif: order.FillOrKill, expected: fokTIF},
		{tif: order.GoodTillTime, err: order.ErrUnsupportedTimeInForce},
	} {
		t.Run(fmt.Sprintf("TIF:%q Price:'%v'", tc.tif, tc.price), func(t *testing.T) {
			t.Parallel()
			got, err := toExchangeTIF(tc.tif, tc.price)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestWebsocketSubmitOrders(t *testing.T) {
	t.Parallel()

	_, err := e.WebsocketSubmitOrders(t.Context(), nil)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sub := &order.Submit{
		Exchange:    e.Name,
		AssetType:   asset.Spot,
		Side:        order.Buy,
		Type:        order.Market,
		QuoteAmount: 10,
	}
	_, err = e.WebsocketSubmitOrders(t.Context(), []*order.Submit{sub})
	require.ErrorIs(t, err, order.ErrPairIsEmpty)

	sub.Pair = currency.NewBTCUSD()
	cpy := *sub
	cpy.AssetType = asset.Futures
	_, err = e.WebsocketSubmitOrders(t.Context(), []*order.Submit{sub, &cpy})
	require.ErrorIs(t, err, errSingleAssetRequired)

	cpy.AssetType = asset.Spread
	sub.AssetType = asset.Spread
	_, err = e.WebsocketSubmitOrders(t.Context(), []*order.Submit{sub, &cpy})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	sub.AssetType = asset.USDTMarginedFutures
	cpy.AssetType = asset.USDTMarginedFutures
	_, err = e.WebsocketSubmitOrders(t.Context(), []*order.Submit{sub, &cpy})
	require.ErrorIs(t, err, errInvalidOrderSize)

	// Prices are far from the market, so a live run leaves resting orders
	for _, tc := range []struct {
		a      asset.Item
		orders []*order.Submit
		exp    []*order.SubmitResponse
	}{
		{
			// Gate rejects an order of a batch on its own, reporting why in its place
			a: asset.Spot,
			orders: []*order.Submit{
				{Type: order.Limit, Side: order.Buy, Price: 10000, Amount: 0.001, ClientOrderID: "ws-spot-buy"},
				{Type: order.Limit, Side: order.Sell, Price: 200000, Amount: 0.001, ClientOrderID: "ws-spot-sell"},
			},
			exp: []*order.SubmitResponse{
				{
					Exchange:        "GateIO",
					OrderID:         "1700664332",
					AssetType:       asset.Spot,
					Pair:            BTCUSDT,
					ClientOrderID:   "t-ws-spot-buy",
					Date:            time.UnixMilli(1681986204832),
					LastUpdated:     time.UnixMilli(1681986204832),
					RemainingAmount: 0.001,
					Amount:          0.001,
					Price:           10000,
					Type:            order.Limit,
					Side:            order.Buy,
					FeeAsset:        currency.BTC,
					TimeInForce:     order.GoodTillCancel,
					Status:          order.Open,
				},
				{
					Exchange:        "GateIO",
					ClientOrderID:   "t-ws-spot-sell",
					SubmissionError: fmt.Errorf("%w reason label:%q message:%q", order.ErrUnableToPlaceOrder, "BALANCE_NOT_ENOUGH", "Not enough balance"),
				},
			},
		},
		{
			a: asset.USDTMarginedFutures,
			orders: []*order.Submit{
				{Type: order.Limit, Side: order.Buy, Price: 10000, Amount: 1, ClientOrderID: "ws-usdt-buy"},
				{Type: order.Limit, Side: order.Sell, Price: 200000, Amount: 1, ClientOrderID: "ws-usdt-sell"},
			},
			exp: []*order.SubmitResponse{
				{
					Exchange:        "GateIO",
					OrderID:         "74046545",
					AssetType:       asset.USDTMarginedFutures,
					Pair:            BTCUSDT,
					ClientOrderID:   "t-ws-usdt-buy",
					Date:            time.UnixMilli(1681196536592),
					LastUpdated:     time.UnixMilli(1681196536592),
					RemainingAmount: 1,
					Amount:          1,
					Price:           10000,
					Type:            order.Limit,
					Side:            order.Long,
					Status:          order.Open,
					TimeInForce:     order.GoodTillCancel,
				},
				{
					Exchange:        "GateIO",
					AssetType:       asset.USDTMarginedFutures,
					SubmissionError: fmt.Errorf("%w reason label:%q detail:%q", order.ErrUnableToPlaceOrder, "INSUFFICIENT_AVAILABLE", "insufficient available balance"),
				},
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			ex := newWsAPITestExchange(t, tc.a, canManipulateRealOrders)
			for _, o := range tc.orders {
				o.Exchange, o.AssetType, o.Pair = ex.Name, tc.a, getPair(t, tc.a)
			}
			got, err := ex.WebsocketSubmitOrders(t.Context(), tc.orders)
			require.NoError(t, err, "WebsocketSubmitOrders must not error")
			if mockTests {
				assert.Equal(t, tc.exp, got, "WebsocketSubmitOrders should return each order's outcome")
				return
			}
			assert.Len(t, got, len(tc.orders), "WebsocketSubmitOrders should return each order's outcome")
		})
	}
}

func TestWebsocketSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketSubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil, "WebsocketSubmitOrder must reject a nil submission")
	_, err = e.WebsocketSubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, AssetType: asset.DeliveryFutures, Pair: currency.NewBTCUSDT(), Type: order.Limit, Side: order.Buy, Price: 1, Amount: 1})
	require.ErrorIs(t, err, asset.ErrNotSupported, "WebsocketSubmitOrder must reject an asset Gate cannot trade over its WebSocket API")

	// Prices are far from the market, so a live run leaves resting orders
	for _, tc := range []struct {
		a      asset.Item
		submit *order.Submit
		exp    *order.SubmitResponse
	}{
		{
			a:      asset.Spot,
			submit: &order.Submit{Type: order.Limit, Side: order.Buy, Price: 10000, Amount: 0.001, ClientOrderID: "ws-spot"},
			exp: &order.SubmitResponse{
				Exchange:        "GateIO",
				OrderID:         "1700664333",
				AssetType:       asset.Spot,
				Pair:            BTCUSDT,
				ClientOrderID:   "t-ws-spot",
				Date:            time.UnixMilli(1681986204832),
				LastUpdated:     time.UnixMilli(1681986204832),
				RemainingAmount: 0.001,
				Amount:          0.001,
				Price:           10000,
				Type:            order.Limit,
				Side:            order.Buy,
				FeeAsset:        currency.BTC,
				TimeInForce:     order.GoodTillCancel,
				Status:          order.Open,
			},
		},
		{
			// Margin orders are placed through the spot connection
			a:      asset.Margin,
			submit: &order.Submit{Type: order.Limit, Side: order.Sell, Price: 200000, Amount: 0.001, ClientOrderID: "ws-margin"},
			exp: &order.SubmitResponse{
				Exchange:        "GateIO",
				OrderID:         "1700664334",
				AssetType:       asset.Margin,
				Pair:            BTCUSDT,
				ClientOrderID:   "t-ws-margin",
				Date:            time.UnixMilli(1681986204832),
				LastUpdated:     time.UnixMilli(1681986204832),
				RemainingAmount: 0.001,
				Amount:          0.001,
				Price:           200000,
				Type:            order.Limit,
				Side:            order.Sell,
				FeeAsset:        currency.USDT,
				TimeInForce:     order.GoodTillCancel,
				Status:          order.Open,
			},
		},
		{
			// Gate finishes an unfilled IOC order as ioc, and its futures order results carry no update time
			a:      asset.USDTMarginedFutures,
			submit: &order.Submit{Type: order.Limit, Side: order.Sell, Price: 200000, Amount: 10, TimeInForce: order.ImmediateOrCancel, ClientOrderID: "ws-usdt"},
			exp: &order.SubmitResponse{
				Exchange:        "GateIO",
				OrderID:         "74046514",
				AssetType:       asset.USDTMarginedFutures,
				Pair:            BTCUSDT,
				ClientOrderID:   "t-ws-usdt",
				Date:            time.UnixMilli(1681195484462),
				LastUpdated:     time.UnixMilli(1681195484462),
				RemainingAmount: 10,
				Amount:          10,
				Price:           200000,
				Type:            order.Limit,
				Side:            order.Short,
				Status:          order.Cancelled,
				TimeInForce:     order.ImmediateOrCancel,
			},
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			connAsset := tc.a
			if connAsset == asset.Margin {
				connAsset = asset.Spot
			}
			ex := newWsAPITestExchange(t, connAsset, canManipulateRealOrders)
			tc.submit.Exchange, tc.submit.AssetType, tc.submit.Pair = ex.Name, tc.a, getPair(t, tc.a)
			got, err := ex.WebsocketSubmitOrder(t.Context(), tc.submit)
			require.NoError(t, err, "WebsocketSubmitOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, got, "WebsocketSubmitOrder should return the placed order")
				return
			}
			assert.NotEmpty(t, got.OrderID, "WebsocketSubmitOrder should return the order ID")
		})
	}
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	_, err := e.GetAvailableTransferChains(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAvailableTransferChains must reject an empty currency")

	for _, tc := range []struct {
		ccy currency.Code
		exp []string
	}{
		// Chains Gate has disabled are left out
		{ccy: currency.USDT, exp: []string{"ETH", "APT", "ARBEVM"}},
		{ccy: currency.ETH, exp: []string{"ETH", "ABS", "ARBEVM", "ARBNOVA", "BASEEVM"}},
	} {
		result, err := e.GetAvailableTransferChains(t.Context(), tc.ccy)
		require.NoErrorf(t, err, "GetAvailableTransferChains must not error for %s", tc.ccy)
		if mockTests {
			assert.Equalf(t, tc.exp, result, "GetAvailableTransferChains should return every enabled %s chain", tc.ccy)
			continue
		}
		assert.NotEmptyf(t, result, "GetAvailableTransferChains should return %s chains", tc.ccy)
	}
}
