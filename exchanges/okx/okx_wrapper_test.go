package okx

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestMessageID(t *testing.T) {
	t.Parallel()
	id := new(Exchange).MessageID()
	require.Len(t, id, 32, "Must return the correct length of message id")
	u, err := uuid.FromString(id)
	require.NoError(t, err, "MessageID must return a valid UUID")
	require.Equal(t, uuid.V7, u.Version(), "MessageID must return a V7 uuid")
	require.Len(t, u.String(), 36, "UUID v7 string representation must be 36 characters long")
}

// 7696807	       153.1 ns/op	      48 B/op	       2 allocs/op
func BenchmarkMessageID(b *testing.B) {
	e := new(Exchange)
	for b.Loop() {
		_ = e.MessageID()
	}
}

func TestGetMarginRatesHistory(t *testing.T) {
	t.Parallel()
	currencies := []currency.Code{currency.USDT, currency.BTC, currency.ETH}
	if !mainPair.IsEmpty() && !mainPair.Base.IsEmpty() {
		currencies = append([]currency.Code{mainPair.Base}, currencies...)
	}
	testCases := []struct {
		name  string
		req   *margin.RateHistoryRequest
		errIs error
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
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetMarginRatesHistory(contextGenerate(), tc.req)
			require.ErrorIs(t, err, tc.errIs)
		})
	}
	t.Run("success", func(t *testing.T) {
		t.Parallel()
		for i := range currencies {
			resp, err := e.GetMarginRatesHistory(contextGenerate(), &margin.RateHistoryRequest{
				Asset:    asset.Margin,
				Currency: currencies[i],
			})
			require.NoError(t, err)
			require.NotNil(t, resp)
			if len(resp.Rates) > 0 {
				return
			}
		}
		t.Skip("OKX returned empty public borrow history for tested currencies")
	})
}

func TestGetMarginRates(t *testing.T) {
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
				Pairs: currency.Pairs{mainPair},
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

			rates, err := target.GetMarginRates(contextGenerate(), tc.req)
			if tc.errIs != nil {
				assert.ErrorIs(t, err, tc.errIs)
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
