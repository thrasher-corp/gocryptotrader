package lbank

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestGetCurrencyPairs(t *testing.T) {
	t.Parallel()
	result, err := e.GetCurrencyPairs(t.Context())
	require.NoError(t, err, "GetCurrencyPairs must not error")
	if mockTests {
		exp := []string{"btc_usdt", "eth_usdt", "btc3l_usdt"}
		assert.Equal(t, exp, result, "GetCurrencyPairs should decode every symbol")
		return
	}
	assert.Contains(t, result, "btc_usdt", "GetCurrencyPairs should list btc_usdt")
}

func TestGetPairInfo(t *testing.T) { //nolint:tparallel // Cases share the package exchange and run in order
	t.Parallel()
	ctx := t.Context()
	btcUSDT := PairInfoResponse{Symbol: "btc_usdt", QuantityDecimals: 5, PriceDecimals: 2, MinimumQuantity: 0.00001, MinimumOrderAmount: 1}
	t.Run("one pair", func(t *testing.T) {
		result, err := e.GetPairInfo(ctx, spotTestPair)
		require.NoError(t, err, "GetPairInfo must not error")
		if mockTests {
			assert.Equal(t, []PairInfoResponse{btcUSDT}, result, "GetPairInfo should decode every field")
		} else {
			require.Len(t, result, 1, "GetPairInfo must return the pair asked for")
			assert.Equal(t, "btc_usdt", result[0].Symbol, "GetPairInfo should return the pair asked for")
		}
	})
	t.Run("every pair", func(t *testing.T) {
		all, err := e.GetPairInfo(ctx, currency.EMPTYPAIR)
		require.NoError(t, err, "GetPairInfo must not error for every pair")
		if mockTests {
			exp := []PairInfoResponse{
				btcUSDT,
				{Symbol: "eth_usdt", QuantityDecimals: 4, PriceDecimals: 2, MinimumQuantity: 0.0001, MinimumOrderAmount: 1},
				{Symbol: "btc3l_usdt", QuantityDecimals: 2, PriceDecimals: 6, MinimumQuantity: 0.01, MinimumOrderAmount: 1},
			}
			assert.Equal(t, exp, all, "GetPairInfo should decode every pair")
			return
		}
		assert.Greater(t, len(all), 1, "GetPairInfo should return every pair")
	})
}

func TestGetAssetConfigs(t *testing.T) { //nolint:tparallel // Cases share the package exchange and run in order
	t.Parallel()
	t.Run("disabled chain with a null fee and contract", func(t *testing.T) {
		if !mockTests {
			t.Skip("LBank's USDT chains change over time; the mock holds a disabled chain recorded live")
		}
		result, err := e.GetAssetConfigs(t.Context(), currency.USDT)
		require.NoError(t, err, "GetAssetConfigs must not error")
		require.Len(t, result, 3, "GetAssetConfigs must return every recorded chain")
		usdt := currency.NewCode("usdt")
		exp := AssetConfigResponse{
			AssetCode: usdt, ChainName: "omni",
			AssetFee: AssetFee{Type: 1, FeeCode: usdt, WithdrawalDecimals: 4, TransferDecimals: 4},
		}
		assert.Equal(t, exp, result[2], "GetAssetConfigs should decode a null fee and contract as zero and empty")
	})
	ctx := t.Context()
	t.Run("no coin", func(t *testing.T) {
		_, err := e.GetAssetConfigs(ctx, currency.EMPTYCODE)
		require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAssetConfigs must require a coin")
	})
	t.Run("BTC", func(t *testing.T) {
		result, err := e.GetAssetConfigs(ctx, currency.BTC)
		require.NoError(t, err, "GetAssetConfigs must not error")
		if mockTests {
			exp := []AssetConfigResponse{
				{
					AssetCode: currency.NewCode("btc"), ChainName: "btc", CanWithdraw: true, CanTransfer: true, CanDeposit: true,
					AssetFee: AssetFee{
						Type: 1, FeeCode: currency.NewCode("btc"), WithdrawalDecimals: 6, MinimumWithdrawal: 0.00005, FeeAmount: 0.00005,
						TransferDecimals: 4, MinimumTransferAmount: 0.0001, MinimumDepositAmount: 0.00001,
					},
				},
				{
					AssetCode: currency.NewCode("btc"), ChainName: "trc20", ContractInfo: "TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9",
					AssetFee: AssetFee{
						Type: 1, FeeCode: currency.NewCode("btc"), WithdrawalDecimals: 4, MinimumWithdrawal: 0.001, FeeAmount: 0.0001,
						TransferDecimals: 4, MinimumTransferAmount: 0.0001, MinimumDepositAmount: 0.0001,
					},
				},
			}
			assert.Equal(t, exp, result, "GetAssetConfigs should decode every field")
			return
		}
		require.NotEmpty(t, result, "GetAssetConfigs must return a chain")
		assert.True(t, result[0].AssetCode.Equal(currency.BTC), "GetAssetConfigs should return the coin asked for")
	})
}

func TestGetTimestamp(t *testing.T) {
	t.Parallel()
	result, err := e.GetTimestamp(t.Context())
	require.NoError(t, err, "GetTimestamp must not error")
	if mockTests {
		exp := time.UnixMilli(1792890784108)
		assert.Equal(t, exp, result, "GetTimestamp should decode the server time")
		return
	}
	assert.WithinDuration(t, time.Now(), result, time.Minute, "GetTimestamp should return the current time")
}
