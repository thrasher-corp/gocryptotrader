package kraken

import (
	"math"
	"net/url"
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
			AccountWidePNLCurrency: currency.USD,
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
	for _, pair := range partialTestPairs {
		_, err := e.SetFuturesPNLCurrencyPreference(t.Context(), pair, currency.USDC)
		require.ErrorIsf(t, err, currency.ErrCurrencyCodeEmpty, "SetFuturesPNLCurrencyPreference must reject %q, which lacks a base or a quote", pair)
	}
	_, err := e.SetFuturesPNLCurrencyPreference(t.Context(), futuresTestPair, currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SetFuturesPNLCurrencyPreference must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name        string
		symbol      currency.Pair
		pnlCurrency currency.Code
		exp         *FuturesSetPreferenceResponse
	}{
		// Sent upper case, as USDC
		{name: "contract", symbol: futuresTestPair, pnlCurrency: currency.USDC.Lower(), exp: &FuturesSetPreferenceResponse{ServerTime: futuresTradingTestTime(8, 41, 0, 200)}},
		{name: "account-wide", pnlCurrency: currency.USD, exp: &FuturesSetPreferenceResponse{ServerTime: futuresTradingTestTime(8, 41, 15, 210)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.SetFuturesPNLCurrencyPreference(t.Context(), tc.symbol, tc.pnlCurrency)
			require.NoError(t, err, "SetFuturesPNLCurrencyPreference must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "SetFuturesPNLCurrencyPreference should decode every field")
				return
			}
			assert.NotZero(t, result.ServerTime, "SetFuturesPNLCurrencyPreference should return the server time")
		})
	}
}

func TestDeleteFuturesPNLCurrencyPreference(t *testing.T) {
	t.Parallel()
	for _, pair := range partialTestPairs {
		_, err := e.DeleteFuturesPNLCurrencyPreference(t.Context(), pair)
		require.ErrorIsf(t, err, currency.ErrCurrencyCodeEmpty, "DeleteFuturesPNLCurrencyPreference must reject %q, which lacks a base or a quote", pair)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name   string
		symbol currency.Pair
		exp    *FuturesSetPreferenceResponse
	}{
		{name: "contract", symbol: futuresTestPair, exp: &FuturesSetPreferenceResponse{ServerTime: futuresTradingTestTime(8, 41, 30, 220)}},
		{name: "account-wide", exp: &FuturesSetPreferenceResponse{ServerTime: futuresTradingTestTime(8, 41, 45, 230)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.DeleteFuturesPNLCurrencyPreference(t.Context(), tc.symbol)
			require.NoError(t, err, "DeleteFuturesPNLCurrencyPreference must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "DeleteFuturesPNLCurrencyPreference should decode every field")
				return
			}
			assert.NotZero(t, result.ServerTime, "DeleteFuturesPNLCurrencyPreference should return the server time")
		})
	}
}

func TestFuturesPNLPreferenceParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		symbol currency.Pair
		exp    url.Values
	}{
		{name: "account-wide", exp: url.Values{}},
		{name: "contract", symbol: futuresTestPair, exp: url.Values{"symbol": {"PF_XBTUSD"}}},
	} {
		params, err := e.futuresPNLPreferenceParams(tc.symbol)
		require.NoErrorf(t, err, "futuresPNLPreferenceParams must not error for the %s preference", tc.name)
		assert.Equalf(t, tc.exp, params, "futuresPNLPreferenceParams should return the %s preference's parameters", tc.name)
	}
	for _, pair := range partialTestPairs {
		_, err := e.futuresPNLPreferenceParams(pair)
		assert.ErrorIsf(t, err, currency.ErrCurrencyCodeEmpty, "futuresPNLPreferenceParams should reject %q, which lacks a base or a quote", pair)
	}
	_, err := new(Exchange).futuresPNLPreferenceParams(futuresTestPair)
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "futuresPNLPreferenceParams should return the error formatting the symbol")
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
	for _, maximumLeverage := range []float64{-0.01, math.NaN(), math.Inf(1), math.Inf(-1)} {
		_, err = e.SetFuturesLeverageSetting(t.Context(), &FuturesLeverageSettingRequest{Symbol: futuresTestPair, MaximumLeverage: maximumLeverage})
		require.ErrorIsf(t, err, errFuturesInvalidLimit, "SetFuturesLeverageSetting must reject a maximum leverage of %v", maximumLeverage)
	}

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
