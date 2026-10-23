package bitstamp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestCreateInstantConvertAddress(t *testing.T) {
	t.Parallel()
	_, err := e.CreateInstantConvertAddress(t.Context(), currency.EMPTYCODE, "")
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "CreateInstantConvertAddress should error on an empty currency")

	if !mockTests {
		t.Skip("CreateInstantConvertAddress is not tested live to avoid creating addresses which sell deposits")
	}
	resp, err := e.CreateInstantConvertAddress(t.Context(), currency.USD, "BECH32")
	require.NoError(t, err, "CreateInstantConvertAddress must not error")
	assert.Equal(t, &InstantConvertAddressResponse{Address: "3MDvHUAg41uJx1511gDotsf4ccKuc9frz1"}, resp, "CreateInstantConvertAddress should return the address")
}

func TestGetInstantConvertAddressInfo(t *testing.T) {
	t.Parallel()
	skipIfLiveWithoutCredentials(t)
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
