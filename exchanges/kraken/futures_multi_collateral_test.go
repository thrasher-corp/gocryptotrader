package kraken

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestGetFuturesPNLCurrencyPreferences(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesPNLCurrencyPreferences(t.Context())
	require.NoError(t, err, "GetFuturesPNLCurrencyPreferences must not error")
	if mockTests {
		exp := &FuturesPNLCurrencyPreferencesResponse{
			Preferences: []FuturesPNLCurrencyPreference{
				{Symbol: "PF_XBTUSD", PNLCurrency: currency.BTC},
				{Symbol: "PF_ETHUSD", PNLCurrency: currency.USDT},
			},
			ServerTime: futuresTradingTestTime(8, 40, 0, 100),
		}
		assert.Equal(t, exp, result, "GetFuturesPNLCurrencyPreferences should decode every field")
		return
	}
	assert.NotZero(t, result.ServerTime, "GetFuturesPNLCurrencyPreferences should return the server time")
}

func TestSetFuturesPNLCurrencyPreference(t *testing.T) {
	t.Parallel()
	_, err := e.SetFuturesPNLCurrencyPreference(t.Context(), currency.EMPTYPAIR, currency.USDC)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "SetFuturesPNLCurrencyPreference must reject an empty symbol")
	_, err = e.SetFuturesPNLCurrencyPreference(t.Context(), futuresTestPair, currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SetFuturesPNLCurrencyPreference must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	// Sent upper case, as USDC
	result, err := e.SetFuturesPNLCurrencyPreference(t.Context(), futuresTestPair, currency.USDC.Lower())
	require.NoError(t, err, "SetFuturesPNLCurrencyPreference must not error")
	if mockTests {
		exp := &FuturesSetPreferenceResponse{ServerTime: futuresTradingTestTime(8, 41, 0, 200)}
		assert.Equal(t, exp, result, "SetFuturesPNLCurrencyPreference should decode every field")
		return
	}
	assert.NotZero(t, result.ServerTime, "SetFuturesPNLCurrencyPreference should return the server time")
}

func TestGetFuturesLeverageSettings(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesLeverageSettings(t.Context())
	require.NoError(t, err, "GetFuturesLeverageSettings must not error")
	if mockTests {
		exp := &FuturesLeverageSettingsResponse{
			LeveragePreferences: []FuturesLeverageSetting{
				{Symbol: "PF_XBTUSD", MaximumLeverage: 10},
				{Symbol: "PF_ETHUSD", MaximumLeverage: 25},
			},
			ServerTime: futuresTradingTestTime(8, 42, 0, 300),
		}
		assert.Equal(t, exp, result, "GetFuturesLeverageSettings should decode every field")
		return
	}
	assert.NotZero(t, result.ServerTime, "GetFuturesLeverageSettings should return the server time")
}

func TestSetFuturesLeverageSetting(t *testing.T) {
	t.Parallel()
	_, err := e.SetFuturesLeverageSetting(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetFuturesLeverageSetting must reject a nil request")
	_, err = e.SetFuturesLeverageSetting(t.Context(), &FuturesLeverageSettingRequest{MaximumLeverage: 10})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "SetFuturesLeverageSetting must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *FuturesLeverageSettingRequest
		exp  *FuturesSetPreferenceResponse
	}{
		{
			name: "isolated margin",
			req:  &FuturesLeverageSettingRequest{Symbol: futuresTestPair, MaximumLeverage: 10},
			exp:  &FuturesSetPreferenceResponse{ServerTime: futuresTradingTestTime(8, 43, 0, 400)},
		},
		{
			name: "cross margin",
			req:  &FuturesLeverageSettingRequest{Symbol: futuresTestPair},
			exp:  &FuturesSetPreferenceResponse{ServerTime: futuresTradingTestTime(8, 43, 30, 500)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.SetFuturesLeverageSetting(t.Context(), tc.req)
			require.NoError(t, err, "SetFuturesLeverageSetting must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "SetFuturesLeverageSetting should decode every field")
				return
			}
			assert.NotZero(t, result.ServerTime, "SetFuturesLeverageSetting should return the server time")
		})
	}
}
