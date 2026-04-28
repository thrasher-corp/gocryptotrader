package gateio

import (
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
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

	openInterest, err := openInterestFromStats([]ContractStat{
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

func TestOldestTime(t *testing.T) {
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

			got := oldestTime(tc.times...)
			assert.Equal(t, tc.expect, got, "oldest time should match expected value")
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

	testexch.UpdatePairsOnce(t, e)

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
			assert.NotZero(t, got.LastUpdated, "Last updated timestamp should be set")
			assert.NotZero(t, got.LastUpdateID, "Last update ID should be set")
			assert.NotZero(t, got.LastPushed, "Last pushed timestamp should be set")
			assert.LessOrEqual(t, got.LastUpdated, got.LastPushed, "Last updated timestamp should be before last pushed timestamp")
		})
	}
}

func TestFetchOrderbookNoSpotInstrument(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	ex.SetDefaults()
	ex.Name = t.Name()

	require.NoError(t, ex.Base.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.NewBTCUSDT()}, false))

	fakePair := currency.NewPair(currency.NewCode("ZZFAKE"), currency.USDT)
	_, err := ex.fetchOrderbook(t.Context(), fakePair, asset.Margin, 1)
	require.ErrorIs(t, err, errNoSpotInstrument)
}
