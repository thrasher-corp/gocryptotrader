package kucoin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetWithdrawalQuotas(t *testing.T) {
	t.Parallel()
	_, err := e.GetWithdrawalQuotas(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetWithdrawalQuotas must reject an empty currency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name  string
		ccy   currency.Code
		chain string
		exp   *WithdrawalQuotasResponse
	}{
		{
			name: "default chain",
			ccy:  currency.BTC,
			exp: &WithdrawalQuotasResponse{
				Currency:                 currency.BTC,
				LimitBTCAmount:           15.79590095,
				UsedBTCAmount:            0.25,
				QuotaCurrency:            currency.USDT,
				LimitQuotaCurrencyAmount: 999999,
				UsedQuotaCurrencyAmount:  16250.5,
				RemainAmount:             15.54590095,
				AvailableAmount:          0.75,
				WithdrawMinFee:           0.0005,
				WithdrawMinSize:          0.001,
				IsWithdrawEnabled:        true,
				Precision:                8,
				Chain:                    "BTC",
			},
		},
		{
			name:  "one chain",
			ccy:   currency.USDT,
			chain: "trx",
			exp: &WithdrawalQuotasResponse{
				Currency:                 currency.USDT,
				LimitBTCAmount:           15.79590095,
				QuotaCurrency:            currency.USDT,
				LimitQuotaCurrencyAmount: 999999,
				RemainAmount:             999999,
				WithdrawMinFee:           1,
				InnerWithdrawMinFee:      0.1,
				WithdrawMinSize:          10,
				Precision:                6,
				Chain:                    "TRC20",
				Reason:                   "Withdrawals are suspended for wallet maintenance",
				LockedAmount:             25.5,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetWithdrawalQuotas(t.Context(), tc.ccy, tc.chain)
			require.NoError(t, err, "GetWithdrawalQuotas must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetWithdrawalQuotas should decode every field")
				return
			}
			require.NotNil(t, result, "GetWithdrawalQuotas must return a result")
			assert.Truef(t, result.Currency.Equal(tc.ccy), "GetWithdrawalQuotas should return the quota of %s", tc.ccy)
		})
	}
}

func TestWithdrawV3(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawV3(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WithdrawV3 must reject a nil request")
	arg := &WithdrawV3Request{}
	_, err = e.WithdrawV3(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "WithdrawV3 must reject an empty currency")
	arg.Currency = currency.USDT
	_, err = e.WithdrawV3(t.Context(), arg)
	require.ErrorIs(t, err, errAddressRequired, "WithdrawV3 must reject an empty destination")
	arg.ToAddress = "test@example.com"
	_, err = e.WithdrawV3(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "WithdrawV3 must reject a zero amount")
	arg.Amount = 1
	_, err = e.WithdrawV3(t.Context(), arg)
	require.ErrorIs(t, err, errWithdrawTypeRequired, "WithdrawV3 must reject an empty withdraw type")

	if !mockTests {
		t.Skip("a withdrawal sends funds to the docs' example destinations, which a test must not do")
	}
	for _, tc := range []struct {
		name string
		req  *WithdrawV3Request
		exp  *WithdrawalResponse
	}{
		{
			name: "internal",
			req:  &WithdrawV3Request{Currency: currency.USDT, Amount: 1, IsInner: true, ToAddress: "test@example.com", WithdrawType: "MAIL"},
			exp:  &WithdrawalResponse{WithdrawalID: "670deec84d64da0007d7c946"},
		},
		{
			name: "on chain",
			req: &WithdrawV3Request{
				Currency:      currency.USDT,
				Chain:         "ton",
				Amount:        3,
				Memo:          "2085123456",
				Remark:        "this is Remark",
				FeeDeductType: "INTERNAL",
				ToAddress:     "UQ***",
				WithdrawType:  "ADDRESS",
			},
			exp: &WithdrawalResponse{WithdrawalID: "670deec84d64da0007d7c947"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.WithdrawV3(t.Context(), tc.req)
			require.NoError(t, err, "WithdrawV3 must not error")
			assert.Equal(t, tc.exp, result, "WithdrawV3 should decode every field")
		})
	}
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	err := e.CancelWithdrawal(t.Context(), "")
	require.ErrorIs(t, err, errWithdrawalIDRequired, "CancelWithdrawal must reject an empty withdrawal ID")

	if !mockTests {
		t.Skip("the withdrawal ID is the docs' example, and a test cannot make a withdrawal to cancel without sending funds")
	}
	err = e.CancelWithdrawal(t.Context(), "5bffb63303aa675e8bbe18f9")
	assert.NoError(t, err, "CancelWithdrawal should not error")
}

func TestGetWithdrawalHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetWithdrawalHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetWithdrawalHistory must reject a nil request")
	_, err = e.GetWithdrawalHistory(t.Context(), &WithdrawalHistoryRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetWithdrawalHistory must reject an empty currency")
	_, err = e.GetWithdrawalHistory(t.Context(), &WithdrawalHistoryRequest{Currency: currency.USDT, StartAt: time.UnixMilli(1766000000000), EndAt: time.UnixMilli(1765900000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetWithdrawalHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *WithdrawalHistoryRequest
		exp  *WithdrawalHistoryResponse
	}{
		{
			name: "currency",
			req:  &WithdrawalHistoryRequest{Currency: currency.USDT},
			exp: &WithdrawalHistoryResponse{
				CurrentPage: 1,
				PageSize:    50,
				TotalNumber: 1,
				TotalPage:   1,
				Items: []Withdrawal{
					{
						ID:        "670deec84d64da0007d7c946",
						Currency:  currency.USDT,
						Status:    "SUCCESS",
						Address:   "test@example.com",
						IsInner:   true,
						Amount:    1,
						CreatedAt: types.Time(time.UnixMilli(1728555875000)),
						UpdatedAt: types.Time(time.UnixMilli(1728555875000)),
					},
				},
			},
		},
		{
			name: "every filter",
			req: &WithdrawalHistoryRequest{
				Currency:    currency.USDT,
				Status:      "FAILURE",
				StartAt:     time.UnixMilli(1765900000000),
				EndAt:       time.UnixMilli(1766000000000),
				CurrentPage: 1,
				PageSize:    10,
			},
			exp: &WithdrawalHistoryResponse{
				CurrentPage: 1,
				PageSize:    10,
				TotalNumber: 1,
				TotalPage:   1,
				Items: []Withdrawal{
					{
						ID:                   "670deec84d64da0007d7c947",
						Currency:             currency.USDT,
						Chain:                "ton",
						Status:               "FAILURE",
						Address:              "UQ***",
						Memo:                 "2085123456",
						Amount:               3,
						Fee:                  0.3,
						WalletTxID:           "5b4c3c6c9000d3c8292781a152d8cf603accb5a4f2f86d123f0ea45a393ae7fe",
						CreatedAt:            types.Time(time.UnixMilli(1765954390000)),
						UpdatedAt:            types.Time(time.UnixMilli(1765954990000)),
						Remark:               "this is Remark",
						FailureReason:        "WALLET_FAILURE",
						FailureReasonMessage: "The wallet failed to process the withdrawal",
						Arrears:              true,
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetWithdrawalHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetWithdrawalHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetWithdrawalHistory should decode every field")
				return
			}
			require.NotNil(t, result, "GetWithdrawalHistory must return a result")
			assert.NotZero(t, result.PageSize, "GetWithdrawalHistory should return a page size")
		})
	}
}

func TestGetWithdrawalHistoryByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetWithdrawalHistoryByID(t.Context(), "")
	require.ErrorIs(t, err, errWithdrawalIDRequired, "GetWithdrawalHistoryByID must reject an empty withdrawal ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		withdrawals, err := e.GetWithdrawalHistory(t.Context(), &WithdrawalHistoryRequest{Currency: currency.USDT})
		require.NoError(t, err, "GetWithdrawalHistory must not error")
		require.NotNil(t, withdrawals, "GetWithdrawalHistory must return a page")
		if len(withdrawals.Items) == 0 {
			t.Skip("the account has no USDT withdrawals to look up")
		}
		result, err := e.GetWithdrawalHistoryByID(t.Context(), withdrawals.Items[0].ID)
		require.NoError(t, err, "GetWithdrawalHistoryByID must not error")
		require.NotNil(t, result, "GetWithdrawalHistoryByID must return a result")
		assert.Equal(t, withdrawals.Items[0].ID, result.ID, "GetWithdrawalHistoryByID should return the requested withdrawal")
		return
	}
	for _, tc := range []struct {
		name string
		id   string
		exp  *WithdrawalHistoryByIDResponse
	}{
		{
			name: "internal",
			id:   "67e6515f7960ba0007b42025",
			exp: &WithdrawalHistoryByIDResponse{
				ID:             "67e6515f7960ba0007b42025",
				UID:            100000001,
				Currency:       currency.USDT,
				ChainID:        "trx",
				ChainName:      "TRC20",
				CurrencyName:   "USDT",
				Status:         "SUCCESS",
				Address:        "TKFRQXSDcY****GmLrjJggwX8",
				IsInner:        true,
				Amount:         3,
				AddressRemark:  "test",
				Remark:         "this is Remark",
				CreatedAt:      types.Time(time.UnixMilli(1743147359000)),
				CancelType:     "NON_CANCELABLE",
				ReturnStatus:   "NOT_RETURN",
				ReturnCurrency: currency.NewCode("KCS"),
			},
		},
		{
			name: "returned",
			id:   "67e6515f7960ba0007b42026",
			exp: &WithdrawalHistoryByIDResponse{
				ID:                   "67e6515f7960ba0007b42026",
				UID:                  100000001,
				Currency:             currency.USDT,
				ChainID:              "ton",
				ChainName:            "TON",
				CurrencyName:         "USDT",
				Status:               "FAILURE",
				FailureReason:        "WALLET_FAILURE",
				FailureReasonMessage: "The wallet failed to process the withdrawal",
				Address:              "UQ***",
				Memo:                 "2085123456",
				Amount:               3,
				Fee:                  0.3,
				WalletTxID:           "6c4c3c6c9000d3c8292781a152d8cf603accb5a4f2f86d123f0ea45a393ae7ff",
				AddressRemark:        "cold wallet",
				Remark:               "this is Remark",
				CreatedAt:            types.Time(time.UnixMilli(1743147359000)),
				CancelType:           "NON_CANCELABLE",
				Taxes:                []string{"0.03 USDT withholding tax"},
				TaxDescription:       "Withholding tax deducted under local regulations",
				TxID:                 "6c4c3c6c9000d3c8292781a152d8cf603accb5a4f2f86d123f0ea45a393ae7ff",
				ReturnStatus:         "SUCCESS",
				ReturnAmount:         3,
				ReturnCurrency:       currency.USDT,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetWithdrawalHistoryByID(t.Context(), tc.id)
			require.NoError(t, err, "GetWithdrawalHistoryByID must not error")
			assert.Equal(t, tc.exp, result, "GetWithdrawalHistoryByID should decode every field")
		})
	}
}
