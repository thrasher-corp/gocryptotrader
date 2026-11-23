package okx

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetSavingBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSavingBalance(contextGenerate(), currency.BTC)
	require.NoError(t, err, "GetSavingBalance must not error")
	if mockTests {
		exp := []SavingBalanceResponse{
			{
				Currency:         currency.BTC,
				Amount:           11.0010737453457821,
				Earnings:         0.0010737388791526,
				Rate:             0.01,
				LoanAmount:       11.0010630707982819,
				PendingAmount:    0.0000106745475002,
				RedemptionAmount: 1,
			},
		}
		assert.Equal(t, exp, result, "GetSavingBalance should decode every field")
	} else {
		assert.NotNil(t, result, "GetSavingBalance should return balances")
	}
}

func TestSavingsPurchaseOrRedemption(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *SavingsPurchaseRedemptionRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty request", arg: &SavingsPurchaseRedemptionRequest{}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing currency", arg: &SavingsPurchaseRedemptionRequest{Rate: 1}, err: currency.ErrCurrencyCodeEmpty},
		{name: "missing amount", arg: &SavingsPurchaseRedemptionRequest{Currency: currency.BTC, Rate: 1}, err: limits.ErrAmountBelowMin},
		{name: "missing side", arg: &SavingsPurchaseRedemptionRequest{Currency: currency.BTC, Amount: 123.4, Rate: 1}, err: order.ErrSideIsInvalid},
		{name: "rate below the minimum", arg: &SavingsPurchaseRedemptionRequest{Currency: currency.BTC, Amount: 123.4, Side: "purchase", Rate: 0.001}, err: errSavingsRateOutOfRange},
		{name: "rate above the maximum", arg: &SavingsPurchaseRedemptionRequest{Currency: currency.BTC, Amount: 123.4, Side: "purchase", Rate: 3.66}, err: errSavingsRateOutOfRange},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.SavingsPurchaseOrRedemption(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "SavingsPurchaseOrRedemption should reject an invalid request")
		})
	}
	for _, tc := range []struct {
		name string
		arg  *SavingsPurchaseRedemptionRequest
		body string
	}{
		{name: "every optional parameter", arg: &SavingsPurchaseRedemptionRequest{Currency: currency.BTC, Amount: 1, Side: "purchase", Rate: 0.01}, body: `{"ccy":"BTC","amt":"1","side":"purchase","rate":"0.01"}`},
		{name: "required parameters only", arg: &SavingsPurchaseRedemptionRequest{Currency: currency.BTC, Amount: 1, Side: "redempt"}, body: `{"ccy":"BTC","amt":"1","side":"redempt"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newRequestPinExchange(t, "/finance/savings/purchase-redempt", tc.body, []map[string]string{{"ccy": "BTC", "amt": "1", "side": tc.arg.Side}})
			_, err := ex.SavingsPurchaseOrRedemption(t.Context(), tc.arg)
			require.NoError(t, err, "SavingsPurchaseOrRedemption must not error")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		arg  *SavingsPurchaseRedemptionRequest
		exp  *SavingsPurchaseRedemptionResponse
	}{
		{
			name: "purchase",
			arg:  &SavingsPurchaseRedemptionRequest{Currency: currency.BTC, Amount: 123.4, Side: "purchase", Rate: 1},
			exp:  &SavingsPurchaseRedemptionResponse{Currency: currency.BTC, Amount: 123.4, Side: "purchase", Rate: 1},
		},
		{
			name: "redemption",
			arg:  &SavingsPurchaseRedemptionRequest{Currency: currency.BTC, Amount: 123.4, Side: "redempt", Rate: 1},
			exp:  &SavingsPurchaseRedemptionResponse{Currency: currency.BTC, Amount: 123.4, Side: "redempt"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.SavingsPurchaseOrRedemption(contextGenerate(), tc.arg)
			require.NoError(t, err, "SavingsPurchaseOrRedemption must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "SavingsPurchaseOrRedemption should decode every field")
			} else {
				assert.Equal(t, tc.arg.Side, result.Side, "SavingsPurchaseOrRedemption should return the requested side")
			}
		})
	}
}

func TestGetLendingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetLendingHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLendingHistory must reject a nil request")
	_, err = e.GetLendingHistory(t.Context(), &LendingHistoryRequest{After: time.UnixMilli(1597026383084), Before: time.UnixMilli(1597026383086)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLendingHistory must reject a window that starts after it ends")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLendingHistory(contextGenerate(), &LendingHistoryRequest{Currency: currency.USDT, Limit: 1})
	require.NoError(t, err, "GetLendingHistory must not error")
	if mockTests {
		exp := []LendingHistoryResponse{
			{Currency: currency.USDT, Amount: 0.01, Earnings: 0.001, Rate: 0.01, Timestamp: types.Time(time.UnixMilli(1597026383085))},
		}
		assert.Equal(t, exp, result, "GetLendingHistory should decode every field")
	} else {
		assert.NotNil(t, result, "GetLendingHistory should return lending history")
	}
}

func TestGetPublicBorrowInfo(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		ccy  currency.Code
		exp  []PublicBorrowInfoResponse
	}{
		{
			name: "every currency",
			ccy:  currency.EMPTYCODE,
			exp: []PublicBorrowInfoResponse{
				{Currency: currency.BTC, AverageRate: 0.005, PreviousRate: 0.005, EstimatedRate: 0.005},
				{Currency: currency.ETH, AverageRate: 0.01, PreviousRate: 0.01, EstimatedRate: 0.01},
				{Currency: currency.USDT, AverageRate: 0.038, PreviousRate: 0.038, EstimatedRate: 0.035},
			},
		},
		{
			name: "USDT",
			ccy:  currency.USDT,
			exp: []PublicBorrowInfoResponse{
				{Currency: currency.USDT, AverageRate: 0.038, PreviousRate: 0.038, EstimatedRate: 0.035},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetPublicBorrowInfo(contextGenerate(), tc.ccy)
			require.NoError(t, err, "GetPublicBorrowInfo must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetPublicBorrowInfo should decode every field")
				return
			}
			require.NotEmpty(t, result, "GetPublicBorrowInfo must return borrowing rates")
			assert.False(t, result[0].Currency.IsEmpty(), "GetPublicBorrowInfo should return a currency")
			if !tc.ccy.IsEmpty() {
				assert.Len(t, result, 1, "GetPublicBorrowInfo should return only the requested currency")
				assert.True(t, result[0].Currency.Equal(tc.ccy), "GetPublicBorrowInfo should return the requested currency")
			}
		})
	}
}

func TestGetPublicBorrowHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetPublicBorrowHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPublicBorrowHistory must reject a nil request")
	_, err = e.GetPublicBorrowHistory(t.Context(), &PublicBorrowHistoryRequest{After: time.UnixMilli(1597026383084), Before: time.UnixMilli(1597026383086)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPublicBorrowHistory must reject a window that starts after it ends")

	result, err := e.GetPublicBorrowHistory(contextGenerate(), &PublicBorrowHistoryRequest{Currency: currency.USDT, Limit: 1})
	require.NoError(t, err, "GetPublicBorrowHistory must not error")
	if mockTests {
		exp := []PublicBorrowHistoryResponse{
			{Currency: currency.USDT, Rate: 0.038, LendingRate: 0.0245, Timestamp: types.Time(time.UnixMilli(1795402800000))},
		}
		assert.Equal(t, exp, result, "GetPublicBorrowHistory should decode every field")
	} else {
		require.Len(t, result, 1, "GetPublicBorrowHistory must return the one requested record")
		assert.True(t, result[0].Currency.Equal(currency.USDT), "GetPublicBorrowHistory should return the requested currency")
		assert.False(t, result[0].Timestamp.Time().IsZero(), "GetPublicBorrowHistory should return a timestamp")
	}
}

func TestSavingsHistoryValues(t *testing.T) {
	t.Parallel()
	_, err := savingsHistoryValues(currency.EMPTYCODE, time.UnixMilli(1597026383084), time.UnixMilli(1597026383086), 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "savingsHistoryValues must reject a window that starts after it ends")
	for _, tc := range []struct {
		name          string
		ccy           currency.Code
		after, before time.Time
		limit         uint64
		exp           url.Values
	}{
		{name: "no filter", exp: url.Values{}},
		{name: "single instant", after: time.UnixMilli(1597026383085), before: time.UnixMilli(1597026383085), exp: url.Values{"after": {"1597026383085"}, "before": {"1597026383085"}}},
		{name: "after only", after: time.UnixMilli(1597026383086), exp: url.Values{"after": {"1597026383086"}}},
		{
			name:   "every filter",
			ccy:    currency.USDT,
			after:  time.UnixMilli(1597026383086),
			before: time.UnixMilli(1597026383084),
			limit:  100,
			exp:    url.Values{"ccy": {"USDT"}, "after": {"1597026383086"}, "before": {"1597026383084"}, "limit": {"100"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := savingsHistoryValues(tc.ccy, tc.after, tc.before, tc.limit)
			require.NoError(t, err, "savingsHistoryValues must not error")
			assert.Equal(t, tc.exp, params, "savingsHistoryValues should encode every set filter")
		})
	}
}
