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

func TestTransferParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *TransferRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &TransferRequest{}, err: order.ErrAmountIsInvalid},
		{req: &TransferRequest{Amount: 1}, err: currency.ErrCurrencyCodeEmpty},
	} {
		_, err := transferParams(tc.req)
		assert.ErrorIs(t, err, tc.err, "transferParams should return the correct error")
	}

	params, err := transferParams(&TransferRequest{Amount: 1.5, Currency: currency.NewCode("usd")})
	require.NoError(t, err, "transferParams must not error")
	assert.Equal(t, url.Values{"amount": {"1.5"}, "currency": {"USD"}}, params, "transferParams should set the amount and upper case currency")

	params, err = transferParams(&TransferRequest{Amount: 1.5, Currency: currency.USD, SubAccount: 990129})
	require.NoError(t, err, "transferParams must not error with a sub account")
	assert.Equal(t, url.Values{"amount": {"1.5"}, "currency": {"USD"}, "subAccount": {"990129"}}, params, "transferParams should set the sub account")
}

func TestTransferToMainAccount(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *TransferRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &TransferRequest{}, err: order.ErrAmountIsInvalid},
		{req: &TransferRequest{Amount: 1}, err: currency.ErrCurrencyCodeEmpty},
	} {
		assert.ErrorIs(t, e.TransferToMainAccount(t.Context(), tc.req), tc.err, "TransferToMainAccount should return the correct error")
	}

	skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
	err := e.TransferToMainAccount(t.Context(), &TransferRequest{Amount: 10000, Currency: currency.BTC, SubAccount: 1234567})
	assert.ErrorIs(t, err, errAPIResponse, "TransferToMainAccount should error for an unknown sub account")
	assert.ErrorContains(t, err, `Sub account with identifier "1234567" does not exist.`, "error should contain the reason")
	if mockTests {
		assert.NoError(t, e.TransferToMainAccount(t.Context(), &TransferRequest{Amount: 10, Currency: currency.BTC, SubAccount: 990129}), "TransferToMainAccount should not error")
	}
}

func TestTransferFromMainAccount(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, e.TransferFromMainAccount(t.Context(), &TransferRequest{Amount: 10, Currency: currency.BTC}), errSubAccountRequired, "TransferFromMainAccount should error without a sub account")

	if !mockTests {
		t.Skip("TransferFromMainAccount is not tested live to avoid transferring funds")
	}
	assert.NoError(t, e.TransferFromMainAccount(t.Context(), &TransferRequest{Amount: 10, Currency: currency.BTC, SubAccount: 990129}), "TransferFromMainAccount should not error")
}
