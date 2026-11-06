package kraken

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestGetFuturesHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesHistoricalFundingRates(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesHistoricalFundingRates must reject an empty pair")

	result, err := e.GetFuturesHistoricalFundingRates(t.Context(), futuresTestPair)
	require.NoError(t, err, "GetFuturesHistoricalFundingRates must not error")
	if mockTests {
		exp := &FuturesHistoricalFundingRatesResponse{
			Rates: []FuturesFundingRate{
				{Timestamp: time.Date(2026, 10, 8, 22, 0, 0, 0, time.UTC), FundingRate: 1.741186912535, RelativeFundingRate: 2.133005e-05},
				{Timestamp: time.Date(2026, 10, 8, 23, 0, 0, 0, time.UTC), FundingRate: 1.50748405495997, RelativeFundingRate: 1.8420433333333e-05},
				{Timestamp: time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC), FundingRate: 1.4840707878906, RelativeFundingRate: 1.8166270833333e-05},
			},
			ServerTime: time.Date(2026, 10, 9, 0, 34, 20, 240000000, time.UTC),
		}
		assert.Equal(t, exp, result, "GetFuturesHistoricalFundingRates should decode every field")
		return
	}
	assert.NotEmpty(t, result.Rates, "GetFuturesHistoricalFundingRates should return rates")
}
