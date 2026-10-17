package kraken

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestCreateSubaccount(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSubaccount(t.Context(), "", "abc123@example.com")
	require.ErrorIs(t, err, errSubaccountUsernameRequired, "CreateSubaccount must reject an empty username")
	_, err = e.CreateSubaccount(t.Context(), "abc123", "")
	require.ErrorIs(t, err, errSubaccountEmailRequired, "CreateSubaccount must reject an empty email address")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	created, err := e.CreateSubaccount(t.Context(), "abc123", "abc123@example.com")
	require.NoError(t, err, "CreateSubaccount must not error")
	assert.True(t, created, "CreateSubaccount should report the subaccount created")
}

func TestAccountTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.AccountTransfer(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AccountTransfer must reject a nil request")
	_, err = e.AccountTransfer(t.Context(), &SubaccountTransferRequest{Amount: 2.54, FromAccountID: "ABCD 1234 EFGH 5678", ToAccountID: "IJKL 0987 MNOP 6543"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "AccountTransfer must reject an empty asset")
	_, err = e.AccountTransfer(t.Context(), &SubaccountTransferRequest{Asset: currency.XBT, FromAccountID: "ABCD 1234 EFGH 5678", ToAccountID: "IJKL 0987 MNOP 6543"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "AccountTransfer must reject a zero amount")
	_, err = e.AccountTransfer(t.Context(), &SubaccountTransferRequest{Asset: currency.XBT, Amount: 2.54, ToAccountID: "IJKL 0987 MNOP 6543"})
	require.ErrorIs(t, err, errSubaccountAccountIDRequired, "AccountTransfer must reject an empty source account")
	_, err = e.AccountTransfer(t.Context(), &SubaccountTransferRequest{Asset: currency.XBT, Amount: 2.54, FromAccountID: "ABCD 1234 EFGH 5678"})
	require.ErrorIs(t, err, errSubaccountAccountIDRequired, "AccountTransfer must reject an empty destination account")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AccountTransfer(t.Context(), &SubaccountTransferRequest{
		Asset:         currency.XBT,
		AssetClass:    "currency",
		Amount:        2.54,
		FromAccountID: "ABCD 1234 EFGH 5678",
		ToAccountID:   "IJKL 0987 MNOP 6543",
	})
	require.NoError(t, err, "AccountTransfer must not error")
	if mockTests {
		assert.Equal(t, &SubaccountTransferResponse{TransferID: "TOH3AS2-LPCWR8-JDQGEU", Status: "complete"}, result, "AccountTransfer should decode every field")
		return
	}
	assert.NotEmpty(t, result.TransferID, "AccountTransfer should return a transfer ID")
}
