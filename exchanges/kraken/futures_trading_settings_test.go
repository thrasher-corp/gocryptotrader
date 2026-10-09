package kraken

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestGetFuturesSelfTradeStrategy(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesSelfTradeStrategy(t.Context())
	require.NoError(t, err, "GetFuturesSelfTradeStrategy must not error")
	if mockTests {
		exp := &FuturesSelfTradeStrategyResponse{
			Strategy:   "CANCEL_MAKER_CHILD",
			ServerTime: time.Date(2026, 10, 9, 0, 41, 25, 831000000, time.UTC),
		}
		assert.Equal(t, exp, result, "GetFuturesSelfTradeStrategy should decode every field")
		return
	}
	assert.NotEmpty(t, result.Strategy, "GetFuturesSelfTradeStrategy should return a strategy")
}

func TestUpdateFuturesSelfTradeStrategy(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateFuturesSelfTradeStrategy(t.Context(), "")
	require.ErrorIs(t, err, errFuturesSelfTradeStrategyEmpty, "UpdateFuturesSelfTradeStrategy must reject an empty strategy")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UpdateFuturesSelfTradeStrategy(t.Context(), "CANCEL_MAKER_SELF")
	require.NoError(t, err, "UpdateFuturesSelfTradeStrategy must not error")
	if mockTests {
		exp := &FuturesSelfTradeStrategyResponse{
			Strategy:   "CANCEL_MAKER_SELF",
			ServerTime: time.Date(2026, 10, 9, 0, 41, 26, 402000000, time.UTC),
		}
		assert.Equal(t, exp, result, "UpdateFuturesSelfTradeStrategy should decode every field")
		return
	}
	assert.Equal(t, "CANCEL_MAKER_SELF", result.Strategy, "UpdateFuturesSelfTradeStrategy should return the strategy set")
}

func TestGetFuturesOffBookMaxLeverageCap(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesOffBookMaxLeverageCap(t.Context())
	require.NoError(t, err, "GetFuturesOffBookMaxLeverageCap must not error")
	if mockTests {
		exp := &FuturesOffBookMaxLeverageResponse{
			MaxLeverage: new(10.0),
			ServerTime:  time.Date(2022, 6, 28, 15, 1, 12, 762000000, time.UTC),
		}
		assert.Equal(t, exp, result, "GetFuturesOffBookMaxLeverageCap should decode every field")
		return
	}
	assert.NotZero(t, result.ServerTime, "GetFuturesOffBookMaxLeverageCap should return the server time")
}

func TestSetFuturesOffBookMaxLeverageCap(t *testing.T) {
	t.Parallel()
	for _, maxLeverage := range []float64{-0.01, 100.01, 5.125} {
		_, err := e.SetFuturesOffBookMaxLeverageCap(t.Context(), maxLeverage)
		require.ErrorIsf(t, err, errFuturesInvalidMaxLeverage, "SetFuturesOffBookMaxLeverageCap must reject %v", maxLeverage)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name        string
		maxLeverage float64
		exp         *FuturesOffBookMaxLeverageResponse
	}{
		{
			name:        "cap",
			maxLeverage: 5.25,
			exp:         &FuturesOffBookMaxLeverageResponse{MaxLeverage: new(5.25), ServerTime: time.Date(2026, 10, 9, 0, 42, 3, 118000000, time.UTC)},
		},
		{
			name: "opt out",
			exp:  &FuturesOffBookMaxLeverageResponse{MaxLeverage: new(0.0), ServerTime: time.Date(2026, 10, 9, 0, 42, 4, 590000000, time.UTC)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.SetFuturesOffBookMaxLeverageCap(t.Context(), tc.maxLeverage)
			require.NoError(t, err, "SetFuturesOffBookMaxLeverageCap must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "SetFuturesOffBookMaxLeverageCap should decode every field")
				return
			}
			assert.Equal(t, tc.exp.MaxLeverage, result.MaxLeverage, "SetFuturesOffBookMaxLeverageCap should return the cap set")
		})
	}
}

func TestClearFuturesOffBookMaxLeverageCap(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.ClearFuturesOffBookMaxLeverageCap(t.Context()), "ClearFuturesOffBookMaxLeverageCap should not error")
}
