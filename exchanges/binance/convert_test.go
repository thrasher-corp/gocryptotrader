package binance

import (
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

func TestGetAllConvertPairs(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllConvertPairs(t.Context(), currency.EMPTYCODE, currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAllConvertPairs must reject missing assets")

	result, err := e.GetAllConvertPairs(t.Context(), currency.BTC, currency.EMPTYCODE)
	require.NoError(t, err, "GetAllConvertPairs must not error")
	if mockTests {
		exp := []ConvertPair{
			{FromAsset: currency.BTC, ToAsset: currency.USDT, FromAssetMinAmount: 0.00000012, FromAssetMaxAmount: 9e24, ToAssetMinAmount: 0.01, ToAssetMaxAmount: 9e24, FromIsBase: true},
			{FromAsset: currency.BTC, ToAsset: currency.NewCode("BAR"), FromAssetMinAmount: 0.00000012, FromAssetMaxAmount: 0.27, ToAssetMinAmount: 0.035, ToAssetMaxAmount: 86000},
		}
		assert.Equal(t, exp, result, "GetAllConvertPairs should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetAllConvertPairs should return pairs")
}

func TestGetOrderQuantityPrecisionPerAsset(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOrderQuantityPrecisionPerAsset(t.Context())
	require.NoError(t, err, "GetOrderQuantityPrecisionPerAsset must not error")
	if mockTests {
		exp := []OrderQuantityPrecision{{Asset: currency.BTC, Fraction: 8}, {Asset: currency.USDT, Fraction: 2}}
		assert.Equal(t, exp, result, "GetOrderQuantityPrecisionPerAsset should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetOrderQuantityPrecisionPerAsset should return assets")
}

func TestAcceptQuote(t *testing.T) {
	t.Parallel()
	_, err := e.AcceptQuote(t.Context(), "")
	require.ErrorIs(t, err, errQuoteIDRequired, "AcceptQuote must reject an empty quote ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AcceptQuote(t.Context(), "933256278426274426")
	require.NoError(t, err, "AcceptQuote must not error")
	if mockTests {
		exp := &AcceptQuoteResponse{
			OrderID:     "933256278426274426",
			CreateTime:  types.Time(time.UnixMilli(1623381330472)),
			OrderStatus: "PROCESS",
		}
		assert.Equal(t, exp, result, "AcceptQuote should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "AcceptQuote should return an order ID")
}

func TestCancelLimitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelLimitOrder(t.Context(), 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelLimitOrder must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelLimitOrder(t.Context(), 1603680255057330400)
	require.NoError(t, err, "CancelLimitOrder must not error")
	if mockTests {
		exp := &ConvertLimitOrderResponse{OrderID: 1603680255057330400, Status: "CANCELED"}
		assert.Equal(t, exp, result, "CancelLimitOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result.Status, "CancelLimitOrder should return a status")
}

func TestGetConvertTradeHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetConvertTradeHistory(t.Context(), time.Time{}, endTime, 10)
	require.ErrorIs(t, err, common.ErrDateUnset, "GetConvertTradeHistory must reject a missing start time")
	_, err = e.GetConvertTradeHistory(t.Context(), endTime, startTime, 10)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetConvertTradeHistory must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetConvertTradeHistory(t.Context(), startTime, endTime, 10)
	require.NoError(t, err, "GetConvertTradeHistory must not error")
	if mockTests {
		exp := &ConvertTradeHistoryResponse{
			List: []ConvertTrade{
				{
					QuoteID:      "f3b91c525b2644c7bc1e1cd31b6e1aa6",
					OrderID:      940708407462087200,
					OrderStatus:  "SUCCESS",
					FromAsset:    currency.USDT,
					FromAmount:   20,
					ToAsset:      currency.BNB,
					ToAmount:     0.06154036,
					Ratio:        0.00307702,
					InverseRatio: 324.99,
					CreateTime:   types.Time(time.UnixMilli(1744150000000)),
				},
			},
			StartTime: types.Time(time.UnixMilli(1744103854944)),
			EndTime:   types.Time(time.UnixMilli(1744190254944)),
			Limit:     10,
			MoreData:  true,
		}
		assert.Equal(t, exp, result, "GetConvertTradeHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetConvertTradeHistory should return a result")
}

func TestGetConvertOrderStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetConvertOrderStatus(t.Context(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetConvertOrderStatus must reject missing IDs")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetConvertOrderStatus(t.Context(), "933256278426274426", "")
	require.NoError(t, err, "GetConvertOrderStatus must not error")
	if mockTests {
		exp := &ConvertOrderStatusResponse{
			OrderID:      933256278426274426,
			OrderStatus:  "SUCCESS",
			FromAsset:    currency.BTC,
			FromAmount:   0.00054414,
			ToAsset:      currency.USDT,
			ToAmount:     20,
			Ratio:        36755,
			InverseRatio: 0.00002721,
			CreateTime:   types.Time(time.UnixMilli(1623381330472)),
		}
		assert.Equal(t, exp, result, "GetConvertOrderStatus should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderStatus, "GetConvertOrderStatus should return a status")
}

func TestPlaceLimitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceLimitOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceLimitOrder must reject a nil request")

	arg := &ConvertLimitOrderRequest{ExpiredType: "7_D"}
	_, err = e.PlaceLimitOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PlaceLimitOrder must reject an empty base asset")

	arg.BaseAsset = currency.BTC
	_, err = e.PlaceLimitOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "PlaceLimitOrder must reject an empty quote asset")

	arg.QuoteAsset = currency.ETH
	_, err = e.PlaceLimitOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "PlaceLimitOrder must reject a missing limit price")

	arg.LimitPrice = 0.0122
	_, err = e.PlaceLimitOrder(t.Context(), arg)
	require.ErrorIs(t, err, errExactlyOneAmountRequired, "PlaceLimitOrder must reject missing amounts")

	arg.BaseAmount, arg.QuoteAmount = 0.01, 0.000122
	_, err = e.PlaceLimitOrder(t.Context(), arg)
	require.ErrorIs(t, err, errExactlyOneAmountRequired, "PlaceLimitOrder must reject both amounts")

	arg.QuoteAmount = 0
	_, err = e.PlaceLimitOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "PlaceLimitOrder must reject an empty side")

	arg.Side = order.Sell.String()
	arg.ExpiredType = ""
	_, err = e.PlaceLimitOrder(t.Context(), arg)
	require.ErrorIs(t, err, errExpiredTypeRequired, "PlaceLimitOrder must reject an empty expiry type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.ExpiredType = "7_D"
	result, err := e.PlaceLimitOrder(t.Context(), arg)
	require.NoError(t, err, "PlaceLimitOrder must not error")
	if mockTests {
		exp := &ConvertLimitOrderResponse{OrderID: 1603680255057330400, Status: "PROCESS"}
		assert.Equal(t, exp, result, "PlaceLimitOrder should decode every field")
		return
	}
	assert.NotZero(t, result.OrderID, "PlaceLimitOrder should return an order ID")
}

func TestGetLimitOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLimitOpenOrders(t.Context())
	require.NoError(t, err, "GetLimitOpenOrders must not error")
	if mockTests {
		exp := &ConvertLimitOpenOrdersResponse{
			List: []ConvertLimitOrder{
				{
					QuoteID:          "18sdf87kh9df",
					OrderID:          1150901289839,
					OrderStatus:      "PROCESS",
					FromAsset:        currency.BNB,
					FromAmount:       10,
					ToAsset:          currency.USDT,
					ToAmount:         2317.89,
					Ratio:            231.789,
					InverseRatio:     0.00431427,
					CreateTime:       types.Time(time.UnixMilli(1614089498000)),
					ExpiredTimestamp: types.Time(time.UnixMilli(1614099498000)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetLimitOpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetLimitOpenOrders should return a result")
}

func TestSendQuoteRequest(t *testing.T) {
	t.Parallel()
	_, err := e.SendQuoteRequest(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SendQuoteRequest must reject a nil request")

	arg := &ConvertQuoteRequest{ToAsset: currency.USDT, FromAmount: 10, WalletType: "FUNDING", ValidTime: "1m"}
	_, err = e.SendQuoteRequest(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SendQuoteRequest must reject an empty from asset")

	arg.FromAsset, arg.ToAsset = currency.BTC, currency.EMPTYCODE
	_, err = e.SendQuoteRequest(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "SendQuoteRequest must reject an empty to asset")

	arg.ToAsset, arg.FromAmount = currency.USDT, 0
	_, err = e.SendQuoteRequest(t.Context(), arg)
	require.ErrorIs(t, err, errExactlyOneAmountRequired, "SendQuoteRequest must reject missing amounts")

	arg.FromAmount, arg.ToAmount = 10, 20
	_, err = e.SendQuoteRequest(t.Context(), arg)
	require.ErrorIs(t, err, errExactlyOneAmountRequired, "SendQuoteRequest must reject both amounts")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.ToAmount = 0
	result, err := e.SendQuoteRequest(t.Context(), arg)
	require.NoError(t, err, "SendQuoteRequest must not error")
	if mockTests {
		exp := &ConvertQuoteResponse{
			QuoteID:        "12415572564",
			Ratio:          38163.7,
			InverseRatio:   0.0000262,
			ValidTimestamp: types.Time(time.UnixMilli(1623319461670)),
			ToAmount:       381637,
			FromAmount:     10,
		}
		assert.Equal(t, exp, result, "SendQuoteRequest should decode every field")
		return
	}
	assert.Positive(t, result.Ratio.Float64(), "SendQuoteRequest should return a ratio")
}
