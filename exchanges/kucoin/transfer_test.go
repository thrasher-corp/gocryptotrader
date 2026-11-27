package kucoin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

func TestGetTransferQuotas(t *testing.T) {
	t.Parallel()
	_, err := e.GetTransferQuotas(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTransferQuotas must reject a nil request")
	_, err = e.GetTransferQuotas(t.Context(), &TransferQuotasRequest{AccountType: "MAIN"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetTransferQuotas must reject an empty currency")
	_, err = e.GetTransferQuotas(t.Context(), &TransferQuotasRequest{Currency: currency.USDT})
	require.ErrorIs(t, err, errAccountTypeMissing, "GetTransferQuotas must reject an empty account type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *TransferQuotasRequest
		exp  *TransferQuotasResponse
	}{
		{
			name: "funding account",
			req:  &TransferQuotasRequest{Currency: currency.USDT, AccountType: "MAIN"},
			exp: &TransferQuotasResponse{
				Currency:     currency.USDT,
				Balance:      12,
				Available:    10.5,
				Holds:        1.5,
				Transferable: 10.5,
			},
		},
		{
			name: "isolated margin account",
			req:  &TransferQuotasRequest{Currency: currency.BTC, AccountType: "ISOLATED", Tag: "BTC-USDT"},
			exp: &TransferQuotasResponse{
				Currency:     currency.BTC,
				Balance:      0.5,
				Available:    0.5,
				Transferable: 0.42,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetTransferQuotas(t.Context(), tc.req)
			require.NoError(t, err, "GetTransferQuotas must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetTransferQuotas should decode every field")
				return
			}
			require.NotNil(t, result, "GetTransferQuotas must return a result")
			assert.Truef(t, result.Currency.Equal(tc.req.Currency), "GetTransferQuotas should return the quota of %s", tc.req.Currency)
		})
	}
}

func TestFlexTransfer(t *testing.T) {
	t.Parallel()
	_, err := e.FlexTransfer(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "FlexTransfer must reject a nil request")
	arg := &FlexTransferRequest{ToAccountTag: "1234"}
	_, err = e.FlexTransfer(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "FlexTransfer must reject an empty client order ID")
	arg.ClientOrderID = "64ccc0f164781800010d8c09"
	_, err = e.FlexTransfer(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "FlexTransfer must reject an empty currency")
	arg.Currency = currency.BTC
	_, err = e.FlexTransfer(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "FlexTransfer must reject a zero amount")
	arg.Amount = 1
	_, err = e.FlexTransfer(t.Context(), arg)
	require.ErrorIs(t, err, errAccountTypeMissing, "FlexTransfer must reject an empty from account type")
	arg.FromAccountType = "MAIN"
	_, err = e.FlexTransfer(t.Context(), arg)
	require.ErrorIs(t, err, errTransferTypeMissing, "FlexTransfer must reject an empty transfer type")
	arg.Type = "INTERNAL"
	_, err = e.FlexTransfer(t.Context(), arg)
	require.ErrorIs(t, err, errAccountTypeMissing, "FlexTransfer must reject an empty to account type")

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.FlexTransfer(t.Context(), &FlexTransferRequest{ClientOrderID: "64ccc0f164781800010d8c09", Currency: currency.BTC, Amount: 1, FromAccountType: "TRADE", Type: "INTERNAL", ToAccountType: "CONTRACT"})
		return err
	})
	assert.JSONEq(t, `{"clientOid":"64ccc0f164781800010d8c09","currency":"BTC","amount":"1","fromAccountType":"TRADE","type":"INTERNAL","toAccountType":"CONTRACT"}`, sent.body, "FlexTransfer should send the amount as the string KuCoin documents")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *FlexTransferRequest
		exp  *FlexTransferResponse
	}{
		{
			name: "internal",
			req: &FlexTransferRequest{
				ClientOrderID:   "64ccc0f164781800010d8c09",
				Currency:        currency.BTC,
				Amount:          1,
				FromAccountType: "TRADE",
				Type:            "INTERNAL",
				ToAccountType:   "CONTRACT",
			},
			exp: &FlexTransferResponse{OrderID: "6705f7248c6954000733ecac"},
		},
		{
			name: "master to sub-account",
			req: &FlexTransferRequest{
				ClientOrderID:   "64ccc0f164781800010d8c0a",
				Currency:        currency.BTC,
				Amount:          1,
				FromAccountType: "TRADE",
				Type:            "PARENT_TO_SUB",
				ToUserID:        "62f5f5d4d72aaf000122707e",
				ToAccountType:   "CONTRACT",
			},
			exp: &FlexTransferResponse{OrderID: "6705f7248c6954000733ecad"},
		},
		{
			name: "sub-account to master",
			req: &FlexTransferRequest{
				ClientOrderID:   "64ccc0f164781800010d8c0b",
				Currency:        currency.USDT,
				Amount:          0.01,
				FromUserID:      "63743f07e0c5230001761d08",
				FromAccountType: "TRADE",
				Type:            "SUB_TO_PARENT",
				ToAccountType:   "TRADE",
			},
			exp: &FlexTransferResponse{OrderID: "6705f7248c6954000733ecae"},
		},
		{
			name: "isolated margin accounts",
			req: &FlexTransferRequest{
				ClientOrderID:   "64ccc0f164781800010d8c0c",
				Currency:        currency.USDT,
				Amount:          5,
				FromAccountType: "ISOLATED",
				FromAccountTag:  "BTC-USDT",
				Type:            "INTERNAL",
				ToAccountType:   "ISOLATED",
				ToAccountTag:    "ETH-USDT",
			},
			exp: &FlexTransferResponse{OrderID: "6705f7248c6954000733ecaf"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !mockTests && (tc.req.FromUserID != "" || tc.req.ToUserID != "") {
				t.Skip("the sub-account user IDs are the docs' examples, and a test must not move funds to an account it does not own")
			}
			result, err := e.FlexTransfer(t.Context(), tc.req)
			require.NoError(t, err, "FlexTransfer must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "FlexTransfer should decode every field")
				return
			}
			require.NotNil(t, result, "FlexTransfer must return a result")
			assert.NotEmpty(t, result.OrderID, "FlexTransfer should return a transfer order ID")
		})
	}
}
