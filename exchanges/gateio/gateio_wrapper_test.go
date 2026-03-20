package gateio

import (
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
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

func TestMessageID(t *testing.T) {
	t.Parallel()
	id := e.MessageID()
	require.Len(t, id, 32, "message ID must be 32 characters long for usage as a request ID")
	got, err := uuid.FromString(id)
	require.NoError(t, err, "ID string must convert back to a UUID")
	require.Equal(t, uuid.V7, got.Version(), "message ID must be a UUID v7")
	require.Len(t, got.String(), 36, "UUID v7 string representation must be 36 characters long")
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

	availSpot, err := e.GetAvailablePairs(asset.Spot)
	require.NoError(t, err, "GetAvailablePairs must not error")
	require.NotEmpty(t, availSpot, "spot pairs must not be empty")

	enabledMargin, err := e.GetEnabledPairs(asset.Margin)
	require.NoError(t, err, "GetEnabledPairs must not error")

	marginPair := availMargin[0]
	marginPairNotInSpot := currency.EMPTYPAIR
	for _, candidate := range enabledMargin {
		if availMargin.Contains(candidate, true) {
			marginPair = candidate
			if !availSpot.Contains(candidate, true) {
				marginPairNotInSpot = candidate
				break
			}
		}
	}
	if marginPairNotInSpot.IsEmpty() {
		for _, candidate := range availMargin {
			if !availSpot.Contains(candidate, true) {
				marginPairNotInSpot = candidate
				break
			}
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

	testCases := []struct {
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
	}
	if !marginPairNotInSpot.IsEmpty() {
		testCases = append(testCases,
			struct {
				pair currency.Pair
				a    asset.Item
				err  error
			}{pair: marginPairNotInSpot, a: asset.Margin, err: errNoSpotInstrument},
			struct {
				pair currency.Pair
				a    asset.Item
				err  error
			}{pair: marginPairNotInSpot, a: asset.Binary, err: asset.ErrNotSupported},
		)
	}
	for _, tc := range testCases {
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

func TestGetCurrentMarginRates(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name          string
		req           *margin.CurrentRatesRequest
		errIs         error
		useLocal      bool
		disableAsset  bool
		clearEnabled  bool
		expectSuccess bool
		skipCreds     bool
	}{
		{
			name:  "nil request",
			req:   nil,
			errIs: common.ErrNilPointer,
		},
		{
			name: "unsupported asset",
			req: &margin.CurrentRatesRequest{
				Asset: asset.Spot,
			},
			errIs: asset.ErrNotSupported,
		},
		{
			name: "empty pair",
			req: &margin.CurrentRatesRequest{
				Asset: asset.Margin,
				Pairs: currency.Pairs{currency.EMPTYPAIR},
			},
			errIs: currency.ErrCurrencyPairEmpty,
		},
		{
			name: "empty pairs lookup error",
			req: &margin.CurrentRatesRequest{
				Asset: asset.Margin,
			},
			useLocal:     true,
			disableAsset: true,
			errIs:        asset.ErrNotEnabled,
		},
		{
			name: "empty pairs after lookup",
			req: &margin.CurrentRatesRequest{
				Asset: asset.Margin,
			},
			useLocal:     true,
			clearEnabled: true,
			errIs:        currency.ErrCurrencyPairsEmpty,
		},
		{
			name: "success",
			req: &margin.CurrentRatesRequest{
				Asset: asset.Margin,
				Pairs: currency.Pairs{currency.EMPTYPAIR},
			},
			expectSuccess: true,
			skipCreds:     true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.skipCreds {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			}
			target := e
			if tc.useLocal {
				local := new(Exchange)
				require.NoError(t, testexch.Setup(local))
				if tc.disableAsset {
					require.NoError(t, local.CurrencyPairs.SetAssetEnabled(asset.Margin, false))
				}
				if tc.clearEnabled {
					ps, err := local.CurrencyPairs.Get(asset.Margin)
					require.NoError(t, err)
					ps.AssetEnabled = true
					ps.Enabled = nil
					require.NoError(t, local.CurrencyPairs.Store(asset.Margin, ps))
				}
				target = local
			}

			req := tc.req
			if tc.expectSuccess && req != nil && len(req.Pairs) == 1 && req.Pairs[0].IsEmpty() {
				req = &margin.CurrentRatesRequest{
					Asset: asset.Margin,
					Pairs: currency.Pairs{getPair(t, asset.Margin)},
				}
			}

			rates, err := target.GetCurrentMarginRates(t.Context(), req)
			if tc.errIs != nil {
				require.ErrorIs(t, err, tc.errIs)
				return
			}
			require.NoError(t, err)
			if tc.expectSuccess {
				require.NotEmpty(t, rates)
				for i := range rates {
					assert.Equal(t, target.Name, rates[i].Exchange)
					assert.Equal(t, asset.Margin, rates[i].Asset)
					assert.NotNil(t, rates[i].CurrentRate)
					assert.False(t, rates[i].CurrentRate.Time.IsZero())
					assert.False(t, rates[i].TimeChecked.IsZero())
					assert.False(t,
						rates[i].CurrentRate.HourlyRate.IsZero() &&
							rates[i].CurrentRate.YearlyRate.IsZero() &&
							rates[i].CurrentRate.HourlyBorrowRate.IsZero() &&
							rates[i].CurrentRate.YearlyBorrowRate.IsZero(),
					)
				}
			}
		})
	}
}
