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

func TestGetUTATransferQuotas(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTATransferQuotas(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTATransferQuotas must reject a nil request")
	_, err = e.GetUTATransferQuotas(t.Context(), &UTATransferQuotasRequest{Currency: currency.USDT})
	require.ErrorIs(t, err, errAccountTypeMissing, "GetUTATransferQuotas must reject an empty account type")
	_, err = e.GetUTATransferQuotas(t.Context(), &UTATransferQuotasRequest{AccountType: "SPOT"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetUTATransferQuotas must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		req *UTATransferQuotasRequest
		exp *UTATransferQuotasResponse
	}{
		{
			req: &UTATransferQuotasRequest{AccountType: "SPOT", Currency: currency.USDT},
			exp: &UTATransferQuotasResponse{AccountType: "SPOT", Currency: currency.USDT, Transferable: 501},
		},
		{
			req: &UTATransferQuotasRequest{AccountType: "ISOLATED", Currency: currency.USDT, Symbol: "BTC-USDT"},
			exp: &UTATransferQuotasResponse{AccountType: "ISOLATED", Currency: currency.USDT, Transferable: 120.5},
		},
	} {
		t.Run(tc.req.AccountType, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTATransferQuotas(t.Context(), tc.req)
			require.NoError(t, err, "GetUTATransferQuotas must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTATransferQuotas should decode every field")
				return
			}
			assert.Equal(t, tc.req.AccountType, result.AccountType, "GetUTATransferQuotas should return the requested account type")
		})
	}
}

func TestUTAFlexTransfer(t *testing.T) {
	t.Parallel()
	valid := UTAFlexTransferRequest{
		ClientOrderID:   "d50a79c0-b94c-4c3d-81b7-c08114021025",
		TransferType:    "INTERNAL",
		Currency:        currency.USDT,
		Amount:          0.000001,
		FromAccountType: "FUNDING",
		FromAccountTag:  "DEFAULT",
		ToAccountType:   "SPOT",
		ToAccountTag:    "DEFAULT",
	}
	_, err := e.UTAFlexTransfer(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UTAFlexTransfer must reject a nil request")
	for _, tc := range []struct {
		name   string
		modify func(*UTAFlexTransferRequest)
		err    error
	}{
		{name: "client order ID", modify: func(r *UTAFlexTransferRequest) { r.ClientOrderID = "" }, err: order.ErrClientOrderIDMustBeSet},
		{name: "transfer type", modify: func(r *UTAFlexTransferRequest) { r.TransferType = "" }, err: errTransferTypeMissing},
		{name: "currency", modify: func(r *UTAFlexTransferRequest) { r.Currency = currency.EMPTYCODE }, err: currency.ErrCurrencyCodeEmpty},
		{name: "amount", modify: func(r *UTAFlexTransferRequest) { r.Amount = 0 }, err: limits.ErrAmountBelowMin},
		{name: "from account type", modify: func(r *UTAFlexTransferRequest) { r.FromAccountType = "" }, err: errAccountTypeMissing},
		{name: "from account tag", modify: func(r *UTAFlexTransferRequest) { r.FromAccountTag = "" }, err: errAccountTagRequired},
		{name: "to account type", modify: func(r *UTAFlexTransferRequest) { r.ToAccountType = "" }, err: errAccountTypeMissing},
		{name: "to account tag", modify: func(r *UTAFlexTransferRequest) { r.ToAccountTag = "" }, err: errAccountTagRequired},
	} {
		req := valid
		tc.modify(&req)
		_, err := e.UTAFlexTransfer(t.Context(), &req)
		require.ErrorIsf(t, err, tc.err, "UTAFlexTransfer must reject an empty %s", tc.name)
	}

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.UTAFlexTransfer(t.Context(), &valid)
		return err
	})
	assert.JSONEq(t, `{"clientOid":"d50a79c0-b94c-4c3d-81b7-c08114021025","transferType":"INTERNAL","currency":"USDT","amount":"0.000001","fromAccountType":"FUNDING","fromAccountTag":"DEFAULT","toAccountType":"SPOT","toAccountTag":"DEFAULT"}`, sent.body, "UTAFlexTransfer should send the amount as the string KuCoin documents")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *UTAFlexTransferRequest
		exp  *UTAFlexTransferResponse
	}{
		{
			name: "internal",
			req:  &valid,
			exp:  &UTAFlexTransferResponse{OrderID: "6a688592584f6c000797c0b9", ClientOrderID: "d50a79c0-b94c-4c3d-81b7-c08114021025"},
		},
		{
			name: "sub to sub",
			req: &UTAFlexTransferRequest{
				ClientOrderID:   "internal-20260811-001",
				TransferType:    "SUB_TO_SUB",
				Currency:        currency.USDT,
				Amount:          10,
				FromUID:         "100000003",
				FromAccountType: "UNIFIED",
				FromAccountTag:  "DEFAULT",
				ToUID:           "100000004",
				ToAccountType:   "ISOLATED",
				ToAccountTag:    "BTC-USDT",
			},
			exp: &UTAFlexTransferResponse{OrderID: "6a688592584f6c000797c0ba", ClientOrderID: "internal-20260811-001"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := tc.req
			if !mockTests {
				if req.TransferType != "INTERNAL" {
					t.Skip("A transfer between sub-accounts needs funded sub-accounts, which a live test cannot set up")
				}
				// KuCoin takes each client order ID once, so every live run needs its own
				live := *req
				live.ClientOrderID = e.MessageID()
				req = &live
			}
			result, err := e.UTAFlexTransfer(t.Context(), req)
			require.NoError(t, err, "UTAFlexTransfer must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "UTAFlexTransfer should decode every field")
				return
			}
			require.NotNil(t, result, "UTAFlexTransfer must return the transfer")
			assert.NotEmpty(t, result.OrderID, "UTAFlexTransfer should return the transfer order ID")
		})
	}
}
