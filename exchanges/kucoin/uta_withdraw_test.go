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

func TestGetUTAWithdrawalQuotas(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAWithdrawalQuotas(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTAWithdrawalQuotas must reject a nil request")
	_, err = e.GetUTAWithdrawalQuotas(t.Context(), &UTAWithdrawalQuotasRequest{WithdrawType: "ADDRESS"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetUTAWithdrawalQuotas must reject an empty currency")
	_, err = e.GetUTAWithdrawalQuotas(t.Context(), &UTAWithdrawalQuotasRequest{Currency: currency.USDT})
	require.ErrorIs(t, err, errWithdrawTypeRequired, "GetUTAWithdrawalQuotas must reject an empty withdraw type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		req *UTAWithdrawalQuotasRequest
		exp *UTAWithdrawalQuotasResponse
	}{
		{
			req: &UTAWithdrawalQuotasRequest{Currency: currency.USDT, WithdrawType: "ADDRESS"},
			exp: &UTAWithdrawalQuotasResponse{
				Currency:                 currency.USDT,
				Chain:                    "trx",
				ChainName:                "TRC20",
				IsWithdrawEnabled:        true,
				AvailableWithdrawAmount:  488940.50000108,
				RemainingQuotaAmount:     24487.5,
				LockedAmount:             1.5,
				WithdrawMinSize:          10,
				MinWithdrawFee:           1,
				Precision:                6,
				QuotaCurrency:            currency.USDT,
				LimitQuotaCurrencyAmount: 25000,
				UsedQuotaCurrencyAmount:  512.5,
				LimitBTCAmount:           0.48780487,
				UsedBTCAmount:            0.01,
			},
		},
		{
			req: &UTAWithdrawalQuotasRequest{Chain: "trx", Currency: currency.USDT, IsInner: true, WithdrawType: "UID"},
			exp: &UTAWithdrawalQuotasResponse{
				Currency:                 currency.USDT,
				Chain:                    "trx",
				ChainName:                "TRC20",
				Reason:                   "Withdrawals are suspended for wallet maintenance.",
				AvailableWithdrawAmount:  488940.50000108,
				RemainingQuotaAmount:     24487.5,
				LockedAmount:             1.5,
				WithdrawMinSize:          1,
				MinWithdrawFee:           1,
				MinInnerWithdrawFee:      0.5,
				Precision:                6,
				QuotaCurrency:            currency.USDT,
				LimitQuotaCurrencyAmount: 25000,
				UsedQuotaCurrencyAmount:  512.5,
				LimitBTCAmount:           0.48780487,
				UsedBTCAmount:            0.01,
			},
		},
	} {
		t.Run(tc.req.WithdrawType, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAWithdrawalQuotas(t.Context(), tc.req)
			require.NoError(t, err, "GetUTAWithdrawalQuotas must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTAWithdrawalQuotas should decode every field")
				return
			}
			assert.Equal(t, tc.req.Currency, result.Currency, "GetUTAWithdrawalQuotas should return the requested currency")
		})
	}
}

func TestGetUTAWithdrawalHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAWithdrawalHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTAWithdrawalHistory must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	cancelled := UTAWithdrawalRecord{
		ID:                   "6a71c7dfec51710007e9502c",
		Currency:             currency.USDT,
		Chain:                "aptos",
		Status:               "FAILURE",
		Address:              "0xb928d679ef4540b9d096114064ee1853f3bcac407650bdc520a32b635e1a0ebc",
		Amount:               3,
		Fee:                  0.5,
		FailureReason:        "withdraw.manual.cancel",
		FailureReasonMessage: "The withdrawal has been cancelled.",
		CreatedAt:            types.Time(time.UnixMilli(1785841631000)),
		UpdatedAt:            types.Time(time.UnixMilli(1785841631000)),
		Remark:               "v2-production-validation-cancel",
		SubStatus:            "CANCELLED",
	}
	for _, tc := range []struct {
		name string
		req  *UTAAssetHistoryRequest
		exp  *UTAWithdrawalHistoryResponse
	}{
		{
			name: "all",
			req:  &UTAAssetHistoryRequest{},
			exp: &UTAWithdrawalHistoryResponse{
				CurrentPage: 1,
				PageSize:    10,
				TotalNumber: 2,
				TotalPage:   1,
				Items: []UTAWithdrawalRecord{
					cancelled,
					{
						ID:                  "6a71c7dfec51710007e9502f",
						Currency:            currency.USDT,
						Chain:               "aptos",
						Status:              "SUCCESS",
						Address:             "0xb928d679ef4540b9d096114064ee1853f3bcac407650bdc520a32b635e1a0ebc",
						Memo:                "1234567",
						IsInner:             true,
						Amount:              3,
						Fee:                 0.5,
						WalletTransactionID: "0x52c4dd6a2a2bf77d5df3ecee8b61bd97cf771c0961e9dc92c643e6ae27485a30",
						CreatedAt:           types.Time(time.UnixMilli(1785841631000)),
						UpdatedAt:           types.Time(time.UnixMilli(1785841682000)),
						Remark:              "v2-production-validation-success",
					},
				},
			},
		},
		{
			name: "filtered",
			req: &UTAAssetHistoryRequest{
				Currency:    currency.USDT,
				ID:          "6a71c7dfec51710007e9502c",
				Status:      "FAILURE",
				StartAt:     time.UnixMilli(1785800000000),
				EndAt:       time.UnixMilli(1785900000000),
				CurrentPage: 1,
				PageSize:    50,
			},
			exp: &UTAWithdrawalHistoryResponse{CurrentPage: 1, PageSize: 50, TotalNumber: 1, TotalPage: 1, Items: []UTAWithdrawalRecord{cancelled}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAWithdrawalHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetUTAWithdrawalHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTAWithdrawalHistory should decode every field")
				return
			}
			assert.NotNil(t, result, "GetUTAWithdrawalHistory should return a result")
		})
	}
}

func TestUTAWithdraw(t *testing.T) {
	t.Parallel()
	_, err := e.UTAWithdraw(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UTAWithdraw must reject a nil request")
	_, err = e.UTAWithdraw(t.Context(), &UTAWithdrawRequest{Amount: 3, ToAddress: "test@example.com", WithdrawType: "MAIL"})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UTAWithdraw must reject an empty currency")
	_, err = e.UTAWithdraw(t.Context(), &UTAWithdrawRequest{Currency: currency.USDT, ToAddress: "test@example.com", WithdrawType: "MAIL"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "UTAWithdraw must reject a zero amount")
	_, err = e.UTAWithdraw(t.Context(), &UTAWithdrawRequest{Currency: currency.USDT, Amount: 3, WithdrawType: "MAIL"})
	require.ErrorIs(t, err, errAddressRequired, "UTAWithdraw must reject an empty address")
	_, err = e.UTAWithdraw(t.Context(), &UTAWithdrawRequest{Currency: currency.USDT, Amount: 3, ToAddress: "test@example.com"})
	require.ErrorIs(t, err, errWithdrawTypeRequired, "UTAWithdraw must reject an empty withdraw type")

	if !mockTests {
		t.Skip("UTAWithdraw sends funds out of the account, so it runs against the mock only")
	}
	for _, tc := range []struct {
		name string
		req  *UTAWithdrawRequest
	}{
		{
			name: "internal",
			req:  &UTAWithdrawRequest{Currency: currency.USDT, Amount: 3, ToAddress: "test@example.com", IsInner: true, WithdrawType: "MAIL"},
		},
		{
			name: "on chain",
			req: &UTAWithdrawRequest{
				Currency:      currency.USDT,
				Chain:         "ton",
				Amount:        3,
				ToAddress:     "UQ***",
				Memo:          "2085123456",
				Remark:        "v2-production-validation-success",
				WithdrawType:  "ADDRESS",
				FeeDeductType: "INTERNAL",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.UTAWithdraw(t.Context(), tc.req)
			require.NoError(t, err, "UTAWithdraw must not error")
			assert.Equal(t, &UTAWithdrawResponse{WithdrawalID: "6a71c7dfec51710007e9502f"}, result, "UTAWithdraw should decode every field")
		})
	}
}

func TestCancelUTAWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := e.CancelUTAWithdrawal(t.Context(), "")
	require.ErrorIs(t, err, errWithdrawalIDRequired, "CancelUTAWithdrawal must reject an empty withdrawal ID")

	if !mockTests {
		t.Skip("CancelUTAWithdrawal needs a withdrawal still being processed, which a test cannot make without sending funds out of the account")
	}
	result, err := e.CancelUTAWithdrawal(t.Context(), "6a71c7dfec51710007e9502c")
	require.NoError(t, err, "CancelUTAWithdrawal must not error")
	assert.Equal(t, &CancelUTAWithdrawalResponse{WithdrawID: "6a71c7dfec51710007e9502c"}, result, "CancelUTAWithdrawal should decode every field")
}
