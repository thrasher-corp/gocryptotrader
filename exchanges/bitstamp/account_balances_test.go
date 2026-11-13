package bitstamp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestGetAccountBalances(t *testing.T) {
	t.Parallel()
	skipIfLiveWithoutCredentials(t)
	balances, err := e.GetAccountBalances(t.Context())
	require.NoError(t, err, "GetAccountBalances must not error")
	// The API sends lower case currency codes
	btc := AccountBalanceResponse{Currency: currency.NewCode("btc"), Total: 1.5, Available: 1.25, Reserved: 0.25}
	if mockTests {
		exp := []AccountBalanceResponse{
			{Currency: currency.NewCode("usd"), Total: 100, Available: 90, Reserved: 10},
			btc,
		}
		assert.Equal(t, exp, balances, "GetAccountBalances should decode the balances")
	}

	_, err = e.GetAccountBalance(t.Context(), currency.EMPTYCODE)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAccountBalance should error on an empty currency")
	balance, err := e.GetAccountBalance(t.Context(), currency.BTC)
	require.NoError(t, err, "GetAccountBalance must not error")
	if mockTests {
		assert.Equal(t, &btc, balance, "GetAccountBalance should decode the balance")
		return
	}
	assert.True(t, balance.Currency.Equal(currency.BTC), "Currency should be correct")
}
