package kucoin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestGetBasicFee(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		currencyType uint64
		exp          *BasicFeeResponse
	}{
		{0, &BasicFeeResponse{TakerFeeRate: 0.001, MakerFeeRate: 0.001}},
		{1, &BasicFeeResponse{TakerFeeRate: 0.001, MakerFeeRate: 0.0008}},
	} {
		result, err := e.GetBasicFee(t.Context(), tc.currencyType)
		require.NoErrorf(t, err, "GetBasicFee must not error for currency type %d", tc.currencyType)
		if mockTests {
			assert.Equalf(t, tc.exp, result, "GetBasicFee should decode every field for currency type %d", tc.currencyType)
			continue
		}
		require.NotNilf(t, result, "GetBasicFee must return a result for currency type %d", tc.currencyType)
		assert.Positivef(t, result.TakerFeeRate.Float64(), "GetBasicFee should return a taker fee rate for currency type %d", tc.currencyType)
	}
}

func TestGetTradingPairActualFees(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradingPairActualFees(t.Context(), nil)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetTradingPairActualFees must reject no symbols")
	_, err = e.GetTradingPairActualFees(t.Context(), []string{"BTC-USDT", ""})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetTradingPairActualFees must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name    string
		symbols []string
		exp     []TradingPairActualFeeResponse
	}{
		{
			name:    "one symbol",
			symbols: []string{"BTC-USDT"},
			exp:     []TradingPairActualFeeResponse{{Symbol: "BTC-USDT", TakerFeeRate: 0.001, MakerFeeRate: 0.0008}},
		},
		{
			name:    "two symbols",
			symbols: []string{"BTC-USDT", "ETH-USDT"},
			exp: []TradingPairActualFeeResponse{
				{Symbol: "BTC-USDT", TakerFeeRate: 0.001, MakerFeeRate: 0.0008, SellTaxRate: 0.002, BuyTaxRate: 0.001},
				{Symbol: "ETH-USDT", TakerFeeRate: 0.001, MakerFeeRate: 0.001},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetTradingPairActualFees(t.Context(), tc.symbols)
			require.NoError(t, err, "GetTradingPairActualFees must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetTradingPairActualFees should decode every field")
				return
			}
			assert.Len(t, result, len(tc.symbols), "GetTradingPairActualFees should return each symbol's fees")
		})
	}
}

func TestGetFuturesTradingPairsActualFees(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesTradingPairsActualFees(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetFuturesTradingPairsActualFees must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesTradingPairsActualFees(t.Context(), "XBTUSDTM")
	require.NoError(t, err, "GetFuturesTradingPairsActualFees must not error")
	if mockTests {
		exp := &FuturesTradingPairsActualFeesResponse{Symbol: "XBTUSDTM", TakerFeeRate: 0.0006, MakerFeeRate: 0.0002}
		assert.Equal(t, exp, result, "GetFuturesTradingPairsActualFees should decode every field")
		return
	}
	require.NotNil(t, result, "GetFuturesTradingPairsActualFees must return a result")
	assert.Equal(t, "XBTUSDTM", result.Symbol, "GetFuturesTradingPairsActualFees should return the requested symbol")
}
