package bitstamp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestCreateInstantConvertAddress(t *testing.T) {
	t.Parallel()
	t.Run("without a currency", func(t *testing.T) {
		_, err := e.CreateInstantConvertAddress(t.Context(), currency.EMPTYCODE, "")
		assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateInstantConvertAddress should error on an empty currency")
	})
	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("The mocked address is not created live, as the API would sell deposits to it")
		}
		resp, err := e.CreateInstantConvertAddress(t.Context(), currency.USD, "BECH32")
		require.NoError(t, err, "CreateInstantConvertAddress must not error")
		assert.Equal(t, &InstantConvertAddressResponse{Address: "3MDvHUAg41uJx1511gDotsf4ccKuc9frz1"}, resp, "CreateInstantConvertAddress should return the address")
	})
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("The request is built for the live API to refuse")
		}
		sharedtestvalues.SkipLiveTestIfCredentialsUnset(t, e, mockTests, canManipulateRealOrders)
		_, err := e.CreateInstantConvertAddress(t.Context(), unlistedCurrency, "BECH32")
		requireRefusedOnMerits(t, err, invalidAddressFormat)
	})
}

func TestGetInstantConvertAddressInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipLiveTestIfCredentialsUnset(t, e, mockTests)
	info, err := e.GetInstantConvertAddressInfo(t.Context(), "")
	require.NoError(t, err, "GetInstantConvertAddressInfo must not error")
	if mockTests {
		exp := []InstantConvertAddressInfoResponse{{
			Address:      "3MDvHUAg41uJx1511gDotsf4ccKuc9frz1",
			CurrencyPair: currency.NewPairWithDelimiter("BTC", "USD", "/"),
			Transactions: []InstantConvertTransaction{{OrderID: 1, Count: 1, Trades: []InstantConvertTrade{{ExchangeRate: 84000, BTCAmount: 0.01}}}},
		}}
		assert.Equal(t, exp, info, "GetInstantConvertAddressInfo should return the address transactions")
	}
}
