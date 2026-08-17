package gateio

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()

	_, err := e.CancelAllOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil)

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{Pair: currency.EMPTYPAIR, AssetType: 1336})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{Pair: currency.NewBTCUSDT(), AssetType: 1336})
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{
		Pair:      currency.NewBTCUSDT(),
		AssetType: asset.Options,
		Side:      order.ClosePosition,
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{
		Pair:      currency.NewPair(currency.BTC, currency.EMPTYCODE),
		AssetType: asset.USDTMarginedFutures,
		Side:      order.Long,
	})
	require.ErrorIs(t, err, errInvalidSettlementQuote)

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{
		Pair:      currency.NewPair(currency.BTC, currency.EMPTYCODE),
		AssetType: asset.USDTMarginedFutures,
		Side:      order.Short,
	})
	require.ErrorIs(t, err, errInvalidSettlementQuote)

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{
		Pair:      currency.NewPair(currency.BTC, currency.EMPTYCODE),
		AssetType: asset.USDTMarginedFutures,
		Side:      order.AnySide,
	})
	require.ErrorIs(t, err, errInvalidSettlementQuote)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	for _, a := range e.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			r := &order.Cancel{
				OrderID:   "1",
				AccountID: "1",
				AssetType: a,
				Pair:      currency.EMPTYPAIR,
			}
			_, err := e.CancelAllOrders(t.Context(), r)
			assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

			r.Pair = getPair(t, a)
			_, err = e.CancelAllOrders(t.Context(), r)
			assert.NoError(t, err)
		})
	}
}

func TestOpenInterestFromStats(t *testing.T) {
	t.Parallel()

	_, err := openInterestFromStats(nil)
	require.ErrorIs(t, err, errNoValidResponseFromServer)

	openInterest, err := openInterestFromStats([]*ContractStat{
		{Time: types.Time(time.Unix(100, 0)), OpenInterest: types.Number(2)},
		{Time: types.Time(time.Unix(300, 0)), OpenInterest: types.Number(4)},
		{Time: types.Time(time.Unix(200, 0)), OpenInterest: types.Number(3)},
	})
	require.NoError(t, err)
	assert.Equal(t, 4.0, openInterest)
}

func TestUseOpenInterestStats(t *testing.T) {
	t.Parallel()

	assert.False(t, useOpenInterestStats(nil, asset.USDTMarginedFutures))
	assert.False(t, useOpenInterestStats([]key.PairAsset{{Asset: asset.CoinMarginedFutures}, {Asset: asset.CoinMarginedFutures}}, asset.CoinMarginedFutures))
	assert.False(t, useOpenInterestStats([]key.PairAsset{{Asset: asset.CoinMarginedFutures}}, asset.USDTMarginedFutures))
	assert.False(t, useOpenInterestStats([]key.PairAsset{{Asset: asset.DeliveryFutures}}, asset.DeliveryFutures))
	assert.True(t, useOpenInterestStats([]key.PairAsset{{Asset: asset.CoinMarginedFutures}}, asset.CoinMarginedFutures))
	assert.True(t, useOpenInterestStats([]key.PairAsset{{Asset: asset.USDTMarginedFutures}}, asset.USDTMarginedFutures))
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	t.Cleanup(server.Close)

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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	t.Cleanup(server.Close)

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
	got, err := uuid.FromString(id)
	require.NoError(t, err, "ID string must convert back to a UUID")
	require.Equal(t, uuid.V7, got.Version(), "message ID must be a UUID v7")
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
			assert.LessOrEqual(t, len(got.Asks), 5, "Asks count should not exceed limit, but may be empty especially for options")
			assert.LessOrEqual(t, len(got.Bids), 5, "Bids count should not exceed limit, but may be empty especially for options")
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

	setCrossMarginAccountBalances(&balances, &CrossMarginAccount{
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

	err := setIsolatedMarginAccountBalances(&accounts.CurrencyBalances{}, []MarginAccountItem{{}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	err = setIsolatedMarginAccountBalances(&accounts.CurrencyBalances{}, []MarginAccountItem{{
		Base: AccountBalanceInformation{Currency: currency.BTC},
	}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	err = setIsolatedMarginAccountBalances(&accounts.CurrencyBalances{}, []MarginAccountItem{{
		AccountType: "inactive",
	}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	balances := accounts.CurrencyBalances{}
	err = setIsolatedMarginAccountBalances(&balances, []MarginAccountItem{
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
