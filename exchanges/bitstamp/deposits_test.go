package bitstamp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestGetCryptoDepositAddress(t *testing.T) { //nolint:tparallel // top-level parallel is safe; the cases send their requests through the package exchange in order
	t.Parallel()
	_, err := e.GetCryptoDepositAddress(t.Context(), currency.EMPTYCODE, "")
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetCryptoDepositAddress should error on an empty currency")

	for _, tc := range []struct {
		name    string
		c       currency.Code
		network string
		exp     DepositAddressResponse
	}{
		{name: "bitcoin", c: currency.BTC, exp: DepositAddressResponse{Address: core.BitcoinDonationAddress}},
		{name: "ether on the ethereum network", c: currency.ETH, network: "ethereum", exp: DepositAddressResponse{Address: "0x6a56f5b80f04b4fd70d64d72e1396698635e5436"}},
		{name: "ripple with a destination tag", c: currency.XRP, exp: DepositAddressResponse{Address: "rvYAfWj5gh67oV6fW32ZzP3Aw4Eubs59B", DestinationTag: 89473951}},
	} {
		// Serial and in order, as each request goes through the package exchange
		t.Run(tc.name, func(t *testing.T) {
			skipIfLiveWithoutCredentials(t)
			addr, err := e.GetCryptoDepositAddress(t.Context(), tc.c, tc.network)
			require.NoErrorf(t, err, "GetCryptoDepositAddress must not error for %s", tc.c)
			if mockTests {
				assert.Equalf(t, tc.exp, *addr, "GetCryptoDepositAddress should return the address for %s", tc.c)
			}
		})
	}
}

func TestGetUnconfirmedBitcoinDeposits(t *testing.T) {
	t.Parallel()
	skipIfLiveWithoutCredentials(t)
	deposits, err := e.GetUnconfirmedBitcoinDeposits(t.Context())
	require.NoError(t, err, "GetUnconfirmedBitcoinDeposits must not error")
	if mockTests {
		require.Len(t, deposits, 1, "GetUnconfirmedBitcoinDeposits must return the deposits")
		assert.Equal(t, UnconfirmedDepositResponse{Amount: 0.001, Address: core.BitcoinDonationAddress, Confirmations: 1}, deposits[0], "deposit should be correct")
	}
}

func TestGetRippleIOUDepositAddress(t *testing.T) {
	t.Parallel()
	skipIfLiveWithoutCredentials(t)
	addr, err := e.GetRippleIOUDepositAddress(t.Context())
	require.NoError(t, err, "GetRippleIOUDepositAddress must not error")
	if mockTests {
		assert.Equal(t, &RippleIOUDepositAddressResponse{Address: "rvYAfWj5gh67oV6fW32ZzP3Aw4Eubs59B", DestinationTag: 89473951}, addr, "address should be correct")
	}
}
