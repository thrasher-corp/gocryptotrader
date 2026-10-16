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

func TestGetC2CTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetC2CTradeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetC2CTradeHistory must reject a nil request")
	startTime, endTime := getTime()
	arg := &C2CTradeHistoryRequest{TradeType: order.Sell.String(), StartTime: endTime, EndTime: startTime, Page: 1, Rows: 100}
	_, err = e.GetC2CTradeHistory(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetC2CTradeHistory must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.StartTime, arg.EndTime = startTime, endTime
	result, err := e.GetC2CTradeHistory(t.Context(), arg)
	require.NoError(t, err, "GetC2CTradeHistory must not error")
	if mockTests {
		exp := &C2CTradeHistoryResponse{
			Code:    "000000",
			Message: "success",
			Data: []C2CTrade{
				{
					OrderNumber:         "20219644646554779648",
					AdvertisementNumber: "11218246497340923904",
					TradeType:           "SELL",
					Asset:               currency.USDT,
					Fiat:                currency.NewCode("CNY"),
					FiatSymbol:          "¥",
					Amount:              5000,
					TotalPrice:          36000,
					UnitPrice:           7.2,
					OrderStatus:         "COMPLETED",
					CreateTime:          types.Time(time.Unix(1744150000, 0)),
					Commission:          5,
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
		assert.Equal(t, exp, result, "GetC2CTradeHistory should decode every field")
		return
	}
	assert.True(t, result.Success, "GetC2CTradeHistory should succeed")
}
