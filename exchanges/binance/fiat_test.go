package binance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetFiatDepositAndWithdrawalHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatDepositAndWithdrawalHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFiatDepositAndWithdrawalHistory must reject a nil request")
	startTime, endTime := getTime()
	arg := &FiatHistoryRequest{TransactionType: 5, BeginTime: endTime, EndTime: startTime, Page: 10, Rows: 50}
	_, err = e.GetFiatDepositAndWithdrawalHistory(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidTransactionType, "GetFiatDepositAndWithdrawalHistory must reject an unknown transaction type")
	arg.TransactionType = 1
	_, err = e.GetFiatDepositAndWithdrawalHistory(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFiatDepositAndWithdrawalHistory must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.BeginTime, arg.EndTime = startTime, endTime
	result, err := e.GetFiatDepositAndWithdrawalHistory(t.Context(), arg)
	require.NoError(t, err, "GetFiatDepositAndWithdrawalHistory must not error")
	if mockTests {
		exp := &FiatOrdersResponse{
			Code:    "000000",
			Message: "success",
			Data: []FiatOrder{
				{
					OrderNumber:     "7d76d611-0568-4f43-afb6-24cac7767365",
					FiatCurrency:    currency.NewCode("BRL"),
					IndicatedAmount: 10,
					Amount:          9.5,
					TotalFee:        0.5,
					Method:          "BankAccount",
					Status:          "Successful",
					CreateTime:      types.Time(time.Unix(1744150000, 0)),
					UpdateTime:      types.Time(time.Unix(1744150900, 0)),
				},
			},
			Total:   1,
			Success: true,
		}
		assert.Equal(t, exp, result, "GetFiatDepositAndWithdrawalHistory should decode every field")
		return
	}
	assert.True(t, result.Success, "GetFiatDepositAndWithdrawalHistory should succeed")
}

func TestGetFiatPaymentHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFiatPaymentHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFiatPaymentHistory must reject a nil request")
	startTime, endTime := getTime()
	arg := &FiatHistoryRequest{TransactionType: 2, BeginTime: endTime, EndTime: startTime, Rows: 50}
	_, err = e.GetFiatPaymentHistory(t.Context(), arg)
	require.ErrorIs(t, err, errInvalidTransactionType, "GetFiatPaymentHistory must reject an unknown transaction type")
	arg.TransactionType = 0
	_, err = e.GetFiatPaymentHistory(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFiatPaymentHistory must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.BeginTime, arg.EndTime = startTime, endTime
	result, err := e.GetFiatPaymentHistory(t.Context(), arg)
	require.NoError(t, err, "GetFiatPaymentHistory must not error")
	if mockTests {
		exp := &FiatPaymentsResponse{
			Code:    "000000",
			Message: "success",
			Data: []FiatPayment{
				{
					OrderNumber:    "353fca443f06466db0c4dc89f94f027a",
					SourceAmount:   20,
					FiatCurrency:   currency.EUR,
					ObtainAmount:   0.0002,
					CryptoCurrency: currency.BTC,
					TotalFee:       0.2,
					Price:          99000,
					Status:         "Completed",
					PaymentMethod:  "Credit Card",
					CreateTime:     types.Time(time.Unix(1744150000, 0)),
					UpdateTime:     types.Time(time.Unix(1744150060, 0)),
				},
			},
			Total:   1,
			Success: true,
		}
		assert.Equal(t, exp, result, "GetFiatPaymentHistory should decode every field")
		return
	}
	assert.True(t, result.Success, "GetFiatPaymentHistory should succeed")
}
