package binance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetCustomerToCustomerTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCustomerToCustomerTradeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCustomerToCustomerTradeHistory must reject a nil request")
	startTime, endTime := getTime()
	arg := &CustomerToCustomerTradeHistoryRequest{TradeType: order.Sell.String(), StartTime: endTime, EndTime: startTime, Page: 1, Rows: 100}
	_, err = e.GetCustomerToCustomerTradeHistory(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetCustomerToCustomerTradeHistory must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.StartTime, arg.EndTime = startTime, endTime
	result, err := e.GetCustomerToCustomerTradeHistory(t.Context(), arg)
	require.NoError(t, err, "GetCustomerToCustomerTradeHistory must not error")
	if mockTests {
		exp := &CustomerToCustomerTradeHistoryResponse{
			Code:    "000000",
			Message: "success",
			Data: []CustomerToCustomerTrade{
				{
					OrderNumber:         "20219644646554779648",
					AdvertisementNumber: "11218246497340923904",
					TradeType:           "SELL",
					Asset:               currency.USDT,
					Fiat:                currency.NewCode("CNY"),
					FiatSymbol:          "¥",
					AmountInCrypto:      5000,
					TotalPriceInFiat:    36000,
					UnitPriceInFiat:     7.2,
					OrderStatus:         "COMPLETED",
					CreateTime:          types.Time(time.Unix(1744150000, 0)),
					CommissionInCrypto:  5,
					CounterPartNickName: "ab***",
					PayMethodName:       "BANK",
					AdditionalKYCVerify: 2,
					TakerCommissionRate: 0.001,
					TakerCommission:     5,
					TakerAmount:         4995,
					AdvertisementRole:   "TAKER",
				},
			},
			Total:   1,
			Success: true,
		}
		assert.Equal(t, exp, result, "GetCustomerToCustomerTradeHistory should decode every field")
		return
	}
	assert.True(t, result.Success, "GetCustomerToCustomerTradeHistory should succeed")
}
