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

func TestCancelFuturesAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelFuturesAlgoOrder(t.Context(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelFuturesAlgoOrder must reject missing IDs")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name         string
		algoID       uint64
		clientAlgoID string
		exp          *CancelAlgoOrderResponse
	}{
		{name: "algoId", algoID: 1234, exp: &CancelAlgoOrderResponse{AlgoID: 1234, Success: true, Message: "OK"}},
		{name: "clientAlgoId", clientAlgoID: "65ce1630101a480b85915d7e11fd5078", exp: &CancelAlgoOrderResponse{AlgoID: 14511, Success: true, Message: "OK"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.CancelFuturesAlgoOrder(t.Context(), tc.algoID, tc.clientAlgoID)
			require.NoError(t, err, "CancelFuturesAlgoOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "CancelFuturesAlgoOrder should decode every field")
				return
			}
			assert.True(t, result.Success, "CancelFuturesAlgoOrder should succeed")
		})
	}
}

func TestGetFuturesCurrentAlgoOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesCurrentAlgoOpenOrders(t.Context())
	require.NoError(t, err, "GetFuturesCurrentAlgoOpenOrders must not error")
	if mockTests {
		exp := &FuturesAlgoOrdersResponse{
			Total: 2,
			Orders: []FuturesAlgoOrder{
				{
					AlgoID:           14517,
					Symbol:           "ETHUSDT",
					Side:             "SELL",
					PositionSide:     "SHORT",
					TotalQuantity:    5,
					ExecutedQuantity: 1,
					ExecutedAmount:   3229.44,
					AveragePrice:     3229.44,
					ClientAlgoID:     "d7096549481642f8a0bb69e9e2e31f2e",
					BookTime:         types.Time(time.UnixMilli(1649756817004)),
					AlgoStatus:       "WORKING",
					AlgoType:         "VP",
					Urgency:          "LOW",
				},
				{
					AlgoID:           14519,
					Symbol:           "BTCUSDT",
					Side:             "BUY",
					PositionSide:     "BOTH",
					TotalQuantity:    0.012,
					ExecutedQuantity: 0.004,
					ExecutedAmount:   312.4,
					AveragePrice:     78100,
					ClientAlgoID:     "65ce1630101a480b85915d7e11fd5078",
					BookTime:         types.Time(time.Unix(1744150000, 0)),
					EndTime:          types.Time(time.Unix(1744151000, 0)),
					AlgoStatus:       "WORKING",
					AlgoType:         "TWAP",
				},
			},
		}
		assert.Equal(t, exp, result, "GetFuturesCurrentAlgoOpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFuturesCurrentAlgoOpenOrders should return a result")
}

func TestGetFuturesHistoricalAlgoOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesHistoricalAlgoOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesHistoricalAlgoOrders must reject a nil request")
	startTime, endTime := getTime()
	arg := &AlgoHistoricalOrdersRequest{Symbol: currency.NewBTCUSDT(), Side: "BUY", StartTime: endTime, EndTime: startTime, Page: 10, PageSize: 100}
	_, err = e.GetFuturesHistoricalAlgoOrders(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesHistoricalAlgoOrders must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.StartTime, arg.EndTime = startTime, endTime
	result, err := e.GetFuturesHistoricalAlgoOrders(t.Context(), arg)
	require.NoError(t, err, "GetFuturesHistoricalAlgoOrders must not error")
	if mockTests {
		exp := &FuturesAlgoOrdersResponse{
			Total: 1,
			Orders: []FuturesAlgoOrder{
				{
					AlgoID:           14518,
					Symbol:           "BTCUSDT",
					Side:             "BUY",
					PositionSide:     "BOTH",
					TotalQuantity:    0.012,
					ExecutedQuantity: 0.005,
					ExecutedAmount:   390.5,
					AveragePrice:     78100,
					ClientAlgoID:     "acacab56b3c44bef9f6a8f8ebd2a8408",
					BookTime:         types.Time(time.Unix(1744150000, 0)),
					EndTime:          types.Time(time.UnixMilli(1744150068598)),
					AlgoStatus:       "CANCELLED",
					AlgoType:         "VP",
					Urgency:          "LOW",
				},
			},
		}
		assert.Equal(t, exp, result, "GetFuturesHistoricalAlgoOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFuturesHistoricalAlgoOrders should return a result")
}

func TestGetFuturesSubOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesSubOrders(t.Context(), 0, 0, 40)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetFuturesSubOrders must reject an empty algo ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesSubOrders(t.Context(), 1234, 0, 40)
	require.NoError(t, err, "GetFuturesSubOrders must not error")
	if mockTests {
		exp := &AlgoSubOrdersResponse{
			Total:            1,
			ExecutedQuantity: 1,
			ExecutedAmount:   3229.44,
			SubOrders: []AlgoSubOrder{
				{
					AlgoID:           1234,
					OrderID:          8389765519993909000,
					OrderStatus:      "FILLED",
					ExecutedQuantity: 1,
					ExecutedAmount:   3229.44,
					FeeAmount:        -1.61471999,
					FeeAsset:         currency.USDT,
					BookTime:         types.Time(time.UnixMilli(1649319001964)),
					AveragePrice:     3229.44,
					Side:             "SELL",
					Symbol:           "ETHUSDT",
					SubID:            1,
					TimeInForce:      "IMMEDIATE_OR_CANCEL",
					OriginalQuantity: 1,
				},
			},
		}
		assert.Equal(t, exp, result, "GetFuturesSubOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFuturesSubOrders should return a result")
}

func TestFuturesTWAPOrder(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesTWAPOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "FuturesTWAPOrder must reject a nil request")
	arg := &FuturesTWAPOrderRequest{PositionSide: "BOTH", Duration: 1000 * time.Second}
	_, err = e.FuturesTWAPOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "FuturesTWAPOrder must reject an empty symbol")
	arg.Symbol = currency.NewBTCUSDT()
	_, err = e.FuturesTWAPOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "FuturesTWAPOrder must reject an empty side")
	arg.Side = order.Sell.String()
	_, err = e.FuturesTWAPOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "FuturesTWAPOrder must reject an empty quantity")
	arg.Quantity, arg.Duration = 0.012, 0
	_, err = e.FuturesTWAPOrder(t.Context(), arg)
	require.ErrorIs(t, err, errDurationRequired, "FuturesTWAPOrder must reject an empty duration")
	for _, d := range []time.Duration{minTWAPDuration - time.Second, maxTWAPDuration + time.Second} {
		arg.Duration = d
		_, err = e.FuturesTWAPOrder(t.Context(), arg)
		require.ErrorIsf(t, err, errDurationRequired, "FuturesTWAPOrder must reject a duration of %s", d)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.Duration = 1000 * time.Second
	result, err := e.FuturesTWAPOrder(t.Context(), arg)
	require.NoError(t, err, "FuturesTWAPOrder must not error")
	if mockTests {
		exp := &AlgoOrderResponse{ClientAlgoID: "65ce1630101a480b85915d7e11fd5078", Success: true, Message: "OK"}
		assert.Equal(t, exp, result, "FuturesTWAPOrder should decode every field")
		return
	}
	assert.True(t, result.Success, "FuturesTWAPOrder should succeed")
}

func TestVolumeParticipationNewOrder(t *testing.T) {
	t.Parallel()
	_, err := e.VolumeParticipationNewOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "VolumeParticipationNewOrder must reject a nil request")
	arg := &VolumeParticipationOrderRequest{PositionSide: "BOTH"}
	_, err = e.VolumeParticipationNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "VolumeParticipationNewOrder must reject an empty symbol")
	arg.Symbol = currency.NewBTCUSDT()
	_, err = e.VolumeParticipationNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "VolumeParticipationNewOrder must reject an empty side")
	arg.Side = order.Sell.String()
	_, err = e.VolumeParticipationNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "VolumeParticipationNewOrder must reject an empty quantity")
	arg.Quantity = 0.12
	_, err = e.VolumeParticipationNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, errPossibleValuesRequired, "VolumeParticipationNewOrder must reject an empty urgency")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.Urgency = "HIGH"
	result, err := e.VolumeParticipationNewOrder(t.Context(), arg)
	require.NoError(t, err, "VolumeParticipationNewOrder must not error")
	if mockTests {
		exp := &AlgoOrderResponse{ClientAlgoID: "00358ce6a268403398bd34eaa36dffe7", Success: true, Message: "OK"}
		assert.Equal(t, exp, result, "VolumeParticipationNewOrder should decode every field")
		return
	}
	assert.True(t, result.Success, "VolumeParticipationNewOrder should succeed")
}

func TestCancelSpotAlgoOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelSpotAlgoOrder(t.Context(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelSpotAlgoOrder must reject missing IDs")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name         string
		algoID       uint64
		clientAlgoID string
		exp          *CancelAlgoOrderResponse
	}{
		{name: "algoId", algoID: 1234, exp: &CancelAlgoOrderResponse{AlgoID: 1234, Success: true, Message: "OK"}},
		{name: "clientAlgoId", clientAlgoID: "65ce1630101a480b85915d7e11fd5078", exp: &CancelAlgoOrderResponse{AlgoID: 14511, Success: true, Message: "OK"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.CancelSpotAlgoOrder(t.Context(), tc.algoID, tc.clientAlgoID)
			require.NoError(t, err, "CancelSpotAlgoOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "CancelSpotAlgoOrder should decode every field")
				return
			}
			assert.True(t, result.Success, "CancelSpotAlgoOrder should succeed")
		})
	}
}

func TestGetCurrentSpotAlgoOpenOrder(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCurrentSpotAlgoOpenOrder(t.Context())
	require.NoError(t, err, "GetCurrentSpotAlgoOpenOrder must not error")
	if mockTests {
		exp := &SpotAlgoOrdersResponse{
			Total: 1,
			Orders: []SpotAlgoOrder{
				{
					AlgoID:           14517,
					Symbol:           "BTCUSDT",
					Side:             "SELL",
					TotalQuantity:    0.012,
					ExecutedQuantity: 0.004,
					ExecutedAmount:   312.4,
					AveragePrice:     78100,
					ClientAlgoID:     "d7096549481642f8a0bb69e9e2e31f2e",
					BookTime:         types.Time(time.Unix(1744150000, 0)),
					EndTime:          types.Time(time.Unix(1744236400, 0)),
					AlgoStatus:       "WORKING",
					AlgoType:         "TWAP",
					Urgency:          "LOW",
				},
			},
		}
		assert.Equal(t, exp, result, "GetCurrentSpotAlgoOpenOrder should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCurrentSpotAlgoOpenOrder should return a result")
}

func TestGetSpotHistoricalAlgoOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotHistoricalAlgoOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotHistoricalAlgoOrders must reject a nil request")
	startTime, endTime := getTime()
	arg := &AlgoHistoricalOrdersRequest{Symbol: currency.NewBTCUSDT(), Side: "BUY", StartTime: endTime, EndTime: startTime, Page: 10, PageSize: 100}
	_, err = e.GetSpotHistoricalAlgoOrders(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSpotHistoricalAlgoOrders must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	arg.StartTime, arg.EndTime = startTime, endTime
	result, err := e.GetSpotHistoricalAlgoOrders(t.Context(), arg)
	require.NoError(t, err, "GetSpotHistoricalAlgoOrders must not error")
	if mockTests {
		exp := &SpotAlgoOrdersResponse{
			Total: 1,
			Orders: []SpotAlgoOrder{
				{
					AlgoID:           14518,
					Symbol:           "BTCUSDT",
					Side:             "BUY",
					TotalQuantity:    0.012,
					ExecutedQuantity: 0.005,
					ExecutedAmount:   390.5,
					AveragePrice:     78100,
					ClientAlgoID:     "acacab56b3c44bef9f6a8f8ebd2a8408",
					BookTime:         types.Time(time.Unix(1744150000, 0)),
					EndTime:          types.Time(time.UnixMilli(1744150068598)),
					AlgoStatus:       "CANCELLED",
					AlgoType:         "TWAP",
					Urgency:          "LOW",
				},
			},
		}
		assert.Equal(t, exp, result, "GetSpotHistoricalAlgoOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotHistoricalAlgoOrders should return a result")
}

func TestGetSpotSubOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotSubOrders(t.Context(), 0, 1, 40)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSpotSubOrders must reject an empty algo ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpotSubOrders(t.Context(), 1234, 1, 40)
	require.NoError(t, err, "GetSpotSubOrders must not error")
	if mockTests {
		exp := &AlgoSubOrdersResponse{
			Total:            1,
			ExecutedQuantity: 1,
			ExecutedAmount:   3229.44,
			SubOrders: []AlgoSubOrder{
				{
					AlgoID:           1234,
					OrderID:          8389765519993909000,
					OrderStatus:      "FILLED",
					ExecutedQuantity: 1,
					ExecutedAmount:   3229.44,
					FeeAmount:        -1.61471999,
					FeeAsset:         currency.USDT,
					BookTime:         types.Time(time.UnixMilli(1649319001964)),
					AveragePrice:     3229.44,
					Side:             "SELL",
					Symbol:           "ETHUSDT",
					SubID:            1,
					TimeInForce:      "IMMEDIATE_OR_CANCEL",
					OriginalQuantity: 1,
				},
			},
		}
		assert.Equal(t, exp, result, "GetSpotSubOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetSpotSubOrders should return a result")
}

func TestSpotTWAPNewOrder(t *testing.T) {
	t.Parallel()
	_, err := e.SpotTWAPNewOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SpotTWAPNewOrder must reject a nil request")
	arg := &SpotTWAPOrderRequest{Duration: 24 * time.Hour}
	_, err = e.SpotTWAPNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "SpotTWAPNewOrder must reject an empty symbol")
	arg.Symbol = currency.NewBTCUSDT()
	_, err = e.SpotTWAPNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "SpotTWAPNewOrder must reject an empty side")
	arg.Side = order.Sell.String()
	_, err = e.SpotTWAPNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "SpotTWAPNewOrder must reject an empty quantity")
	arg.Quantity, arg.Duration = 0.012, 0
	_, err = e.SpotTWAPNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, errDurationRequired, "SpotTWAPNewOrder must reject an empty duration")
	for _, d := range []time.Duration{minTWAPDuration - time.Second, maxTWAPDuration + time.Second} {
		arg.Duration = d
		_, err = e.SpotTWAPNewOrder(t.Context(), arg)
		require.ErrorIsf(t, err, errDurationRequired, "SpotTWAPNewOrder must reject a duration of %s", d)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	arg.Duration = 24 * time.Hour
	result, err := e.SpotTWAPNewOrder(t.Context(), arg)
	require.NoError(t, err, "SpotTWAPNewOrder must not error")
	if mockTests {
		exp := &AlgoOrderResponse{ClientAlgoID: "65ce1630101a480b85915d7e11fd5078", Success: true, Message: "OK"}
		assert.Equal(t, exp, result, "SpotTWAPNewOrder should decode every field")
		return
	}
	assert.True(t, result.Success, "SpotTWAPNewOrder should succeed")
}
