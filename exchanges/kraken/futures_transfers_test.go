package kraken

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestFuturesWalletTransfer(t *testing.T) {
	t.Parallel()
	err := e.FuturesWalletTransfer(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "FuturesWalletTransfer must reject a nil request")
	err = e.FuturesWalletTransfer(t.Context(), &FuturesWalletTransferRequest{FromWallet: "fi_ethusd", Currency: currency.ETH, Amount: 0.1})
	require.ErrorIs(t, err, errFuturesWalletEmpty, "FuturesWalletTransfer must reject an empty wallet")
	err = e.FuturesWalletTransfer(t.Context(), &FuturesWalletTransferRequest{FromWallet: "fi_ethusd", ToWallet: "flex", Amount: 0.1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "FuturesWalletTransfer must reject an empty currency")
	err = e.FuturesWalletTransfer(t.Context(), &FuturesWalletTransferRequest{FromWallet: "fi_ethusd", ToWallet: "flex", Currency: currency.ETH})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "FuturesWalletTransfer must reject an amount that is not positive")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.FuturesWalletTransfer(t.Context(), &FuturesWalletTransferRequest{FromWallet: "fi_ethusd", ToWallet: "flex", Currency: currency.ETH, Amount: 0.1})
	assert.NoError(t, err, "FuturesWalletTransfer should not error")
}

func TestFuturesSubaccountTransfer(t *testing.T) {
	t.Parallel()
	err := e.FuturesSubaccountTransfer(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "FuturesSubaccountTransfer must reject a nil request")
	req := &FuturesSubaccountTransferRequest{
		FromUser:   futuresTestMasterAccountUID,
		FromWallet: "flex",
		ToWallet:   "cash",
		Currency:   currency.USD,
		Amount:     250.5,
	}
	err = e.FuturesSubaccountTransfer(t.Context(), req)
	require.ErrorIs(t, err, errFuturesUserEmpty, "FuturesSubaccountTransfer must reject an empty recipient")
	err = e.FuturesSubaccountTransfer(t.Context(), &FuturesSubaccountTransferRequest{ToUser: futuresTestSubaccountUID})
	require.ErrorIs(t, err, errFuturesUserEmpty, "FuturesSubaccountTransfer must reject an empty sender")
	err = e.FuturesSubaccountTransfer(t.Context(), &FuturesSubaccountTransferRequest{FromUser: futuresTestMasterAccountUID, ToUser: futuresTestSubaccountUID})
	require.ErrorIs(t, err, errFuturesWalletEmpty, "FuturesSubaccountTransfer must reject an empty wallet")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	req.FromUser, req.ToUser = futuresTestAccountUIDs(t)
	assert.NoError(t, e.FuturesSubaccountTransfer(t.Context(), req), "FuturesSubaccountTransfer should not error")
}

func TestFuturesWithdrawToSpotWallet(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesWithdrawToSpotWallet(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "FuturesWithdrawToSpotWallet must reject a nil request")
	_, err = e.FuturesWithdrawToSpotWallet(t.Context(), &FuturesWithdrawToSpotWalletRequest{Amount: 0.1})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "FuturesWithdrawToSpotWallet must reject an empty currency")
	_, err = e.FuturesWithdrawToSpotWallet(t.Context(), &FuturesWithdrawToSpotWalletRequest{Currency: currency.ETH, Amount: -0.1})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "FuturesWithdrawToSpotWallet must reject an amount that is not positive")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *FuturesWithdrawToSpotWalletRequest
		exp  *FuturesWithdrawToSpotWalletResponse
	}{
		{
			name: "source wallet",
			req:  &FuturesWithdrawToSpotWalletRequest{Currency: currency.ETH, Amount: 0.1, SourceWallet: "flex"},
			exp:  &FuturesWithdrawToSpotWalletResponse{UID: "9053db5f-0d5e-48dd-b606-a5c92576b706", ServerTime: time.Date(2022, 6, 28, 14, 48, 58, 711000000, time.UTC)},
		},
		{
			name: "cash wallet",
			req:  &FuturesWithdrawToSpotWalletRequest{Currency: currency.XBT, Amount: 0.25},
			exp:  &FuturesWithdrawToSpotWalletResponse{UID: "c4a9e0d2-6b1f-4f7e-9d3a-81e5b2f07c64", ServerTime: time.Date(2026, 10, 9, 0, 44, 52, 318000000, time.UTC)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.FuturesWithdrawToSpotWallet(t.Context(), tc.req)
			require.NoError(t, err, "FuturesWithdrawToSpotWallet must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "FuturesWithdrawToSpotWallet should decode every field")
				return
			}
			assert.NotEmpty(t, result.UID, "FuturesWithdrawToSpotWallet should return the withdrawal's reference")
		})
	}
}

func TestFuturesTransferParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		from, to string
		code     currency.Code
		amount   float64
		err      error
	}{
		{name: "no source wallet", to: "flex", code: currency.USD, amount: 1, err: errFuturesWalletEmpty},
		{name: "no destination wallet", from: "cash", code: currency.USD, amount: 1, err: errFuturesWalletEmpty},
		{name: "no currency", from: "cash", to: "flex", amount: 1, err: currency.ErrCurrencyCodeEmpty},
		{name: "no amount", from: "cash", to: "flex", code: currency.USD, err: limits.ErrAmountBelowMin},
		{name: "negative amount", from: "cash", to: "flex", code: currency.USD, amount: -1, err: limits.ErrAmountBelowMin},
	} {
		_, err := futuresTransferParams(tc.from, tc.to, tc.code, tc.amount)
		assert.ErrorIsf(t, err, tc.err, "futuresTransferParams should reject a transfer with %s", tc.name)
	}
	params, err := futuresTransferParams("cash", "flex", currency.USD, 0.0125)
	require.NoError(t, err, "futuresTransferParams must not error")
	assert.Equal(t, url.Values{"fromAccount": {"cash"}, "toAccount": {"flex"}, "unit": {"usd"}, "amount": {"0.0125"}}, params, "futuresTransferParams should set the wallets, lower case unit and amount")
}
