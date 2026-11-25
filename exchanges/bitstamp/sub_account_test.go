package bitstamp

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// transferRequestErrors are the transfer requests refused before they are sent
var transferRequestErrors = []struct {
	name string
	req  *TransferRequest
	err  error
}{
	{name: "nil request", req: nil, err: common.ErrNilPointer},
	{name: "without an amount", req: &TransferRequest{}, err: order.ErrAmountIsInvalid},
	{name: "without a currency", req: &TransferRequest{Amount: 1}, err: currency.ErrCurrencyCodeEmpty},
}

func TestTransferParams(t *testing.T) {
	t.Parallel()
	for _, tc := range transferRequestErrors {
		// Serial and in order, as the valid requests below go to the same function
		t.Run(tc.name, func(t *testing.T) {
			_, err := transferParams(tc.req)
			assert.ErrorIs(t, err, tc.err, "transferParams should return the correct error")
		})
	}
	t.Run("main account", func(t *testing.T) {
		t.Parallel()
		params, err := transferParams(&TransferRequest{Amount: 1.5, Currency: currency.NewCode("usd")})
		require.NoError(t, err, "transferParams must not error")
		assert.Equal(t, url.Values{"amount": {"1.5"}, "currency": {"USD"}}, params, "transferParams should set the amount and upper case currency")
	})
	t.Run("sub account", func(t *testing.T) {
		t.Parallel()
		params, err := transferParams(&TransferRequest{Amount: 1.5, Currency: currency.USD, SubAccount: 990129})
		require.NoError(t, err, "transferParams must not error with a sub account")
		assert.Equal(t, url.Values{"amount": {"1.5"}, "currency": {"USD"}, "subAccount": {"990129"}}, params, "transferParams should set the sub account")
	})
}

func TestTransferToMainAccount(t *testing.T) {
	t.Parallel()
	for _, tc := range transferRequestErrors {
		// Serial and in order, as the requests below go through the same exchange
		t.Run(tc.name, func(t *testing.T) {
			assert.ErrorIs(t, e.TransferToMainAccount(t.Context(), tc.req), tc.err, "TransferToMainAccount should return the correct error")
		})
	}
	t.Run("unknown sub account", func(t *testing.T) {
		t.Parallel()
		skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
		err := e.TransferToMainAccount(t.Context(), &TransferRequest{Amount: 10000, Currency: currency.BTC, SubAccount: 1234567})
		assert.ErrorIs(t, err, errAPIResponse, "TransferToMainAccount should error for an unknown sub account")
		assert.ErrorContains(t, err, `Sub account with identifier "1234567" does not exist.`, "error should contain the reason")
		if mockTests {
			assert.NoError(t, e.TransferToMainAccount(t.Context(), &TransferRequest{Amount: 10, Currency: currency.BTC, SubAccount: 990129}), "TransferToMainAccount should not error")
		}
	})
}

func TestTransferFromMainAccount(t *testing.T) {
	t.Parallel()
	for _, tc := range transferRequestErrors {
		// Serial and in order, as the requests below go through the same exchange
		t.Run(tc.name, func(t *testing.T) {
			assert.ErrorIs(t, e.TransferFromMainAccount(t.Context(), tc.req), tc.err, "TransferFromMainAccount should return the correct error")
		})
	}
	t.Run("without a sub account", func(t *testing.T) {
		assert.ErrorIs(t, e.TransferFromMainAccount(t.Context(), &TransferRequest{Amount: 10, Currency: currency.BTC}), errSubAccountRequired, "TransferFromMainAccount should error without a sub account")
	})
	t.Run("mocked", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("The mocked transfer is not sent live, to avoid transferring funds")
		}
		assert.NoError(t, e.TransferFromMainAccount(t.Context(), &TransferRequest{Amount: 10, Currency: currency.BTC, SubAccount: 990129}), "TransferFromMainAccount should not error")
	})
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("The request is built for the live API to refuse")
		}
		skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
		// More bitcoin than will ever exist, so no account holds it
		requireRefusedOnMerits(t, e.TransferFromMainAccount(t.Context(), &TransferRequest{Amount: 21000001, Currency: currency.BTC, SubAccount: 1234567}), invalidChoice)
	})
}
