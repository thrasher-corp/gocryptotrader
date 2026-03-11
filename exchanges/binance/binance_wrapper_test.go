package binance

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestGetMarginRatesHistory(t *testing.T) {
	t.Parallel()
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
				Asset:     asset.Margin,
				Currency:  currency.USDT,
				StartDate: time.Now().Add(-30 * 24 * time.Hour),
				EndDate:   time.Now(),
			},
			success: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.success {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			}
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
		name           string
		req            *margin.CurrentRatesRequest
		errIs          error
		useEnabledPair bool
		skipCreds      bool
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
			name:           "success",
			req:            &margin.CurrentRatesRequest{Asset: asset.Margin},
			useEnabledPair: true,
			skipCreds:      true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.skipCreds {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			}
			req := tc.req
			if tc.useEnabledPair {
				pairs, err := e.GetEnabledPairs(asset.Margin)
				require.NoError(t, err)
				require.NotEmpty(t, pairs)
				req = &margin.CurrentRatesRequest{
					Asset: asset.Margin,
					Pairs: currency.Pairs{pairs[0]},
				}
			}
			rates, err := e.GetCurrentMarginRates(t.Context(), req)
			if tc.errIs != nil {
				require.ErrorIs(t, err, tc.errIs)
				return
			}
			if err != nil && (strings.Contains(err.Error(), "REQUEST_FORBIDDEN") || strings.Contains(err.Error(), "not authorized")) {
				t.Skipf("credentials do not have access to current margin rates endpoint: %v", err)
			}
			require.NoError(t, err)
			require.NotEmpty(t, rates)
			for i := range rates {
				require.NotNil(t, rates[i].CurrentRate)
				assert.False(t, rates[i].CurrentRate.Time.IsZero())
				assert.False(t, rates[i].TimeChecked.IsZero())
				assert.False(t,
					rates[i].CurrentRate.HourlyRate.IsZero() &&
						rates[i].CurrentRate.YearlyRate.IsZero() &&
						rates[i].CurrentRate.HourlyBorrowRate.IsZero() &&
						rates[i].CurrentRate.YearlyBorrowRate.IsZero(),
				)
			}
		})
	}
}
