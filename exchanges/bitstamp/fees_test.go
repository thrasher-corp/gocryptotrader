package bitstamp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestGetTradingFees(t *testing.T) {
	t.Parallel()
	skipIfLiveWithoutCredentials(t)
	fees, err := e.GetTradingFees(t.Context())
	require.NoError(t, err, "GetTradingFees must not error")
	require.NotEmpty(t, fees, "GetTradingFees must return fees")
	if mockTests {
		exp := []TradingFeeResponse{
			{Market: "btcusd", Fees: MakerTakerFees{Maker: 0.3, Taker: 0.4}},
			{Market: "ltcbtc", Fees: MakerTakerFees{Maker: 0.3, Taker: 0.4}},
		}
		assert.Equal(t, exp, fees, "GetTradingFees should decode the fees")
	}
	for _, f := range fees {
		assert.NotEmpty(t, f.Market, "Market should be set")
		assert.GreaterOrEqual(t, f.Fees.Taker.Float64(), f.Fees.Maker.Float64(), "Taker fee should not be below the maker fee")
	}

	_, err = e.GetTradingFee(t.Context(), currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetTradingFee should error on an empty pair")
	fee, err := e.GetTradingFee(t.Context(), spotPair)
	require.NoError(t, err, "GetTradingFee must not error")
	if mockTests {
		assert.Equal(t, &TradingFeeResponse{Market: "btcusd", Fees: MakerTakerFees{Maker: 0.3, Taker: 0.4}}, fee, "GetTradingFee should decode the fee")
		return
	}
	assert.Equal(t, "btcusd", fee.Market, "Market should be correct")
}

func TestGetWithdrawalFees(t *testing.T) {
	t.Parallel()
	skipIfLiveWithoutCredentials(t)
	fees, err := e.GetWithdrawalFees(t.Context())
	require.NoError(t, err, "GetWithdrawalFees must not error")
	require.NotEmpty(t, fees, "GetWithdrawalFees must return fees")
	if mockTests {
		// The API sends lower case currency codes
		exp := []WithdrawalFeeResponse{
			{Currency: currency.NewCode("btc"), Fee: 0.00015, Network: "bitcoin"},
			{Currency: currency.NewCode("eth"), Fee: 0.001, Network: "ethereum"},
		}
		assert.Equal(t, exp, fees, "GetWithdrawalFees should decode the fees")
	}

	_, err = e.GetWithdrawalFee(t.Context(), currency.EMPTYCODE, "")
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetWithdrawalFee should error on an empty currency")
	for _, network := range []string{"", "lightning"} {
		fee, err := e.GetWithdrawalFee(t.Context(), currency.BTC, network)
		require.NoErrorf(t, err, "GetWithdrawalFee must not error for network %q", network)
		assert.True(t, fee.Currency.Equal(currency.BTC), "Currency should be correct")
		if mockTests {
			exp := map[string]WithdrawalFeeResponse{
				"":          {Currency: currency.NewCode("btc"), Fee: 0.00015, Network: "bitcoin"},
				"lightning": {Currency: currency.NewCode("btc"), Fee: 0.00001, Network: "lightning"},
			}[network]
			assert.Equalf(t, exp, *fee, "GetWithdrawalFee should return the fee for network %q", network)
		}
	}
}
