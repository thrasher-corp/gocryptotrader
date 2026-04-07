package kucoin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			require.NoError(t, e.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error")
			pairs, err := e.GetAvailablePairs(a)
			require.NoError(t, err, "GetPairs must not error")
			for _, p := range pairs {
				l, err := e.GetOrderExecutionLimits(a, p)
				require.NoError(t, err, "GetOrderExecutionLimits must not error")
				assert.Positive(t, l.AmountStepIncrementSize, "AmountStepIncrementSize should not be zero")
			}
		})
	}
	t.Run("unsupported asset", func(t *testing.T) {
		t.Parallel()
		require.ErrorIs(t, e.UpdateOrderExecutionLimits(t.Context(), asset.Binary), asset.ErrNotSupported)
	})
}

func TestGetMarginRatesHistory(t *testing.T) {
	t.Parallel()
	ccy := currency.USDT
	if !marginTradablePair.IsEmpty() && !marginTradablePair.Base.IsEmpty() {
		ccy = marginTradablePair.Base
	}
	testCases := []struct {
		name    string
		req     *margin.RateHistoryRequest
		errIs   error
		success bool
	}{
		{
			name:  "nil request",
			req:   nil,
			errIs: common.ErrNilPointer,
		},
		{
			name: "unsupported asset",
			req: &margin.RateHistoryRequest{
				Asset:    asset.Spot,
				Currency: currency.USDT,
			},
			errIs: asset.ErrNotSupported,
		},
		{
			name: "missing currency",
			req: &margin.RateHistoryRequest{
				Asset: asset.Margin,
			},
			errIs: currency.ErrCurrencyCodeEmpty,
		},
		{
			name: "success",
			req: &margin.RateHistoryRequest{
				Asset:    asset.Margin,
				Currency: ccy,
			},
			success: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := e.GetMarginRatesHistory(t.Context(), tc.req)
			if tc.success {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.NotEmpty(t, resp.Rates)
				return
			}
			require.ErrorIs(t, err, tc.errIs)
		})
	}
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
				Pairs: currency.Pairs{marginTradablePair},
			},
			expectSuccess: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
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

			rates, err := target.GetCurrentMarginRates(t.Context(), tc.req)
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
