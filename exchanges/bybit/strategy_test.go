package bybit

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestCreateStrategyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreateStrategyOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateStrategyOrder must reject a nil request")
	pov := &StrategyPOVParametersRequest{Mode: "TradedVolume", ParticipationRate: 25.1}
	for _, tc := range []struct {
		name string
		req  *CreateStrategyOrderRequest
		err  error
	}{
		{name: "no category", req: &CreateStrategyOrderRequest{Symbol: "BTCUSDT", Side: "Buy", StrategyType: "chaseOrder", Size: 1}, err: errCategoryNotSet},
		{name: "no symbol", req: &CreateStrategyOrderRequest{Category: "UTA_USDT", Side: "Buy", StrategyType: "chaseOrder", Size: 1}, err: errSymbolMissing},
		{name: "no side", req: &CreateStrategyOrderRequest{Category: "UTA_USDT", Symbol: "BTCUSDT", StrategyType: "chaseOrder", Size: 1}, err: order.ErrSideIsInvalid},
		{name: "no strategy type", req: &CreateStrategyOrderRequest{Category: "UTA_USDT", Symbol: "BTCUSDT", Side: "Buy", Size: 1}, err: errStrategyTypeRequired},
		{name: "chase order without size or position value", req: &CreateStrategyOrderRequest{Category: "UTA_USDT", Symbol: "BTCUSDT", Side: "Buy", StrategyType: "chaseOrder"}, err: errStrategySizeRequired},
		{name: "iceberg without size or position value", req: &CreateStrategyOrderRequest{Category: "UTA_USDT", Symbol: "BTCUSDT", Side: "Buy", StrategyType: "iceberg"}, err: errStrategySizeRequired},
		{name: "TWAP without duration", req: &CreateStrategyOrderRequest{Category: "UTA_USDT", Symbol: "BTCUSDT", Side: "Buy", StrategyType: "twap", Size: 1}, err: errStrategyDurationRequired},
		{name: "POV without parameters", req: &CreateStrategyOrderRequest{Category: "UTA_USDT", Symbol: "BTCUSDT", Side: "Buy", StrategyType: "pov", Size: 1}, err: errStrategyPOVParametersRequired},
		{name: "POV without mode", req: &CreateStrategyOrderRequest{Category: "UTA_USDT", Symbol: "BTCUSDT", Side: "Buy", StrategyType: "pov", Size: 1, POVParameters: &StrategyPOVParametersRequest{ParticipationRate: 25}}, err: errStrategyPOVModeRequired},
		{name: "POV without participation rate", req: &CreateStrategyOrderRequest{Category: "UTA_USDT", Symbol: "BTCUSDT", Side: "Buy", StrategyType: "pov", Size: 1, POVParameters: &StrategyPOVParametersRequest{Mode: "TradedVolume"}}, err: errStrategyParticipationRateRequired},
		{name: "POV with size and position value", req: &CreateStrategyOrderRequest{Category: "UTA_USDT", Symbol: "BTCUSDT", Side: "Buy", StrategyType: "pov", Size: 1, PositionValue: 1000, POVParameters: pov}, err: errStrategySizeWithPositionValue},
		{name: "POV without stop condition", req: &CreateStrategyOrderRequest{Category: "UTA_USDT", Symbol: "BTCUSDT", Side: "Buy", StrategyType: "pov", POVParameters: pov}, err: errStrategyPOVStopConditionRequired},
	} {
		_, err = e.CreateStrategyOrder(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "CreateStrategyOrder must reject a request with %s", tc.name)
	}
	_, err = e.CreateStrategyOrder(t.Context(), &CreateStrategyOrderRequest{Category: "UTA_USDT", Symbol: "BTCUSDT", Side: "Buy", StrategyType: "pov", Interval: new(30 * time.Second), POVParameters: pov})
	require.ErrorIs(t, err, errStrategyPOVStopConditionRequired, "CreateStrategyOrder must not take a repeating interval as a POV stop condition")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *CreateStrategyOrderRequest
		exp  *CreateStrategyOrderResponse
	}{
		{
			name: "twap",
			req: &CreateStrategyOrderRequest{
				Category:       "UTA_USDT",
				Symbol:         "BTCUSDT",
				Side:           "Buy",
				StrategyType:   "twap",
				Size:           0.05,
				Duration:       26 * time.Minute,
				ReduceOnly:     true,
				PositionIndex:  1,
				Interval:       new(time.Minute),
				IsRandom:       true,
				TriggerPrice:   75000,
				MaxChasePrice:  83564.5,
				ChasePercentE4: 100,
			},
			exp: &CreateStrategyOrderResponse{StrategyID: "119b6211-2611-461b-be5e-5ac557099e82"},
		},
		{
			name: "chaseOrder",
			req: &CreateStrategyOrderRequest{
				Category:      "UTA_SPOT",
				Symbol:        "BTCUSDT",
				Side:          "Sell",
				StrategyType:  "chaseOrder",
				PositionValue: 5000,
				LeverageType:  1,
				TriggerPrice:  75000,
				MaxChasePrice: 70000,
				ChaseDistance: 0.5,
			},
			exp: &CreateStrategyOrderResponse{StrategyID: "2c2fb2a4-6b3c-4c55-9f2e-0e1b8a7d9c31"},
		},
		{
			name: "iceberg",
			req: &CreateStrategyOrderRequest{
				Category:         "UTA_USDT",
				Symbol:           "BTCUSDT",
				Side:             "Buy",
				StrategyType:     "iceberg",
				Size:             0.36,
				MaxChasePrice:    198000,
				ChaseDistance:    -1,
				SubSize:          0.02,
				OrderCount:       18,
				SubPositionValue: 1500,
				PostOnly:         new(uint64(0)),
				LimitPrice:       134500,
			},
			exp: &CreateStrategyOrderResponse{StrategyID: "cf7303ae-29c0-480a-8f3d-eaa9330054bc"},
		},
		{
			name: "pov",
			req: &CreateStrategyOrderRequest{
				Category:     "UTA_USDT",
				Symbol:       "BTCUSDT",
				Side:         "Buy",
				StrategyType: "pov",
				Size:         0.5,
				Duration:     time.Hour,
				Interval:     new(time.Duration(0)),
				POVParameters: &StrategyPOVParametersRequest{
					Mode:              "TradedVolume",
					ParticipationRate: 25.1,
					ReferenceWindow:   5 * time.Minute,
					DepthReference:    5,
				},
			},
			exp: &CreateStrategyOrderResponse{StrategyID: "6f1e2d3c-4b5a-4968-8776-655443322110"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.CreateStrategyOrder(t.Context(), tc.req)
			require.NoError(t, err, "CreateStrategyOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "CreateStrategyOrder should decode every field")
				return
			}
			assert.NotEmpty(t, result.StrategyID, "CreateStrategyOrder should return a strategy ID")
		})
	}
	if mockTests {
		result, err := e.CreateStrategyOrder(t.Context(), &CreateStrategyOrderRequest{Category: "UTA_USDT", Symbol: "BTCUSDT", Side: "Buy", StrategyType: "twap", Size: 0.0001, Duration: 5 * time.Minute})
		assert.ErrorIs(t, err, errStrategyOrderFailed, "CreateStrategyOrder should report a strategy that was not created")
		assert.Equal(t, &CreateStrategyOrderResponse{Result: "order quantity is below the minimum"}, result, "CreateStrategyOrder should return the failed result")
		result, err = e.CreateStrategyOrder(t.Context(), &CreateStrategyOrderRequest{Category: "UTA_USDC", Symbol: "BTCPERP", Side: "Buy", StrategyType: "chaseOrder", Size: 0.01})
		assert.ErrorIs(t, err, errAPIResponse, "CreateStrategyOrder should return the error reply to a chase order on USDC perpetuals")
		assert.Nil(t, result, "CreateStrategyOrder should return no strategy with an error reply")
	}
}

func TestSetStrategyTimeWindow(t *testing.T) {
	t.Parallel()
	begin, end := time.Unix(1773700000, 0), time.Unix(1773800000, 0)
	for _, tc := range []struct {
		name       string
		begin, end time.Time
		exp        url.Values
		err        error
	}{
		{name: "no window", exp: url.Values{}},
		{name: "begin only", begin: begin, exp: url.Values{"beginTimeE0": {"1773700000"}}},
		{name: "end only", end: end, exp: url.Values{"endTimeE0": {"1773800000"}}},
		{name: "both bounds", begin: begin, end: end, exp: url.Values{"beginTimeE0": {"1773700000"}, "endTimeE0": {"1773800000"}}},
		{name: "begin after end", begin: end, end: begin, exp: url.Values{}, err: common.ErrStartAfterEnd},
		{name: "begin equals end", begin: begin, end: begin, exp: url.Values{}, err: common.ErrStartEqualsEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params := url.Values{}
			err := setStrategyTimeWindow(params, tc.begin, tc.end)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "setStrategyTimeWindow should reject the window")
			} else {
				assert.NoError(t, err, "setStrategyTimeWindow should accept the window")
			}
			assert.Equal(t, tc.exp, params, "setStrategyTimeWindow should set each given bound in whole seconds, and nothing for a rejected window")
		})
	}
	params := url.Values{}
	require.NoError(t, setStrategyTimeWindow(params, time.UnixMilli(1773700000999), time.Time{}), "setStrategyTimeWindow must not error")
	assert.Equal(t, url.Values{"beginTimeE0": {"1773700000"}}, params, "setStrategyTimeWindow should drop the fraction of a second")
}

func TestGetStrategyList(t *testing.T) {
	t.Parallel()
	_, err := e.GetStrategyList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetStrategyList must reject a nil request")
	_, err = e.GetStrategyList(t.Context(), &GetStrategyListRequest{BeginTime: time.Unix(1773800000, 0), EndTime: time.Unix(1773700000, 0)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetStrategyList must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetStrategyList(t.Context(), &GetStrategyListRequest{StrategyID: "119b6211-2611-461b-be5e-5ac557099e82", Status: 3, StrategyType: "twap"})
	require.NoError(t, err, "GetStrategyList must not error")
	if mockTests {
		require.Len(t, result.List, 1, "GetStrategyList must return the strategy")
		assert.Equal(t, "119b6211-2611-461b-be5e-5ac557099e82", result.List[0].StrategyID, "GetStrategyList should filter by strategy ID")
	}
	result, err = e.GetStrategyList(t.Context(), &GetStrategyListRequest{
		Symbol:    "BTCUSDT",
		Category:  "UTA_USDT",
		BeginTime: time.Unix(1773700000, 0),
		EndTime:   time.Unix(1773800000, 0),
		PageSize:  2,
		Cursor:    "eyJPZmZzZXQiOjIsIlBhZ2VTaXplIjoyfQ==",
	})
	require.NoError(t, err, "GetStrategyList must not error")
	if mockTests {
		exp := &StrategyListResponse{
			List: []Strategy{
				{
					StrategyID:           "119b6211-2611-461b-be5e-5ac557099e82",
					Category:             "UTA_USDT",
					Symbol:               "BTCUSDT",
					Side:                 "Buy",
					Size:                 0.05,
					StrategyType:         "twap",
					Status:               3,
					ExecutedSize:         0.046,
					ExecutedAveragePrice: 76695.48,
					ExecutedStartTime:    types.Time(time.UnixMilli(1773711467045)),
					ExecutedEndTime:      types.Time(time.UnixMilli(1773713075628)),
					CreatedTime:          types.Time(time.UnixMilli(1773711467045)),
					UpdatedTime:          types.Time(time.UnixMilli(1773713075628)),
					ReduceOnly:           true,
					TriggerPrice:         75000,
					IsTriggered:          true,
					LeverageType:         1,
					TerminateType:        2,
					TerminateRemark:      "RunningStop",
					TriggerCount:         26,
					TradingCount:         24,
					RealizedPNL:          1.25,
					StrategyName:         "accumulate btc",
					StrategyPrefer:       "quickExecution",
					Duration:             1560,
					ExecutedDuration:     1608,
					IsRandom:             true,
					Interval:             60,
					LimitPrice:           77000,
					ChasePercentE4:       100,
					ChaseDistance:        0.5,
					MaxChasePrice:        83564.5,
					ChaseOrderPrice:      76690.1,
					ChasePrice:           "Bid1",
					PostOnly:             1,
					IsRebalance:          true,
					OrderType:            "UNKNOWN",
					OrderPriceOffset:     0.1,
					StrategySL:           70000,
					StrategyTP:           90000,
					ArbitrageOrders:      []json.RawMessage{json.RawMessage(`{"orderId":"d67e97b1-7a6e-4a89-b9d7-098756e8b86f"}`)},
					PositionValue:        3834.77,
					FilledPositionValue:  3527.99,
				},
				{
					StrategyID:           "6f1e2d3c-4b5a-4968-8776-655443322110",
					Category:             "UTA_USDT",
					Symbol:               "BTCUSDT",
					Side:                 "Buy",
					Size:                 0.5,
					StrategyType:         "pov",
					Status:               2,
					ExecutedSize:         0.12,
					ExecutedAveragePrice: 76702.1,
					ExecutedStartTime:    types.Time(time.Unix(1773720000, 0)),
					CreatedTime:          types.Time(time.Unix(1773719999, 0)),
					UpdatedTime:          types.Time(time.Unix(1773721200, 0)),
					TriggerCount:         40,
					TradingCount:         40,
					StrategyPrefer:       "quickExecution",
					Duration:             3600,
					ExecutedDuration:     1200,
					Interval:             30,
					OrderType:            "UNKNOWN",
					ArbitrageOrders:      []json.RawMessage{},
					Mode:                 "TradedVolume",
					ParticipationRate:    25.1,
					ReferenceWindow:      300,
					DepthReference:       5,
				},
			},
			NextCursor:     "eyJPZmZzZXQiOjQsIlBhZ2VTaXplIjoyfQ==",
			PreviousCursor: "eyJPZmZzZXQiOjAsIlBhZ2VTaXplIjoyfQ==",
		}
		assert.Equal(t, exp, result, "GetStrategyList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetStrategyList should return a result")
}

func TestGetStrategyOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetStrategyOrderList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetStrategyOrderList must reject a nil request")
	_, err = e.GetStrategyOrderList(t.Context(), &GetStrategyOrderListRequest{})
	require.ErrorIs(t, err, errStrategyIDRequired, "GetStrategyOrderList must reject a request without a strategy ID")
	_, err = e.GetStrategyOrderList(t.Context(), &GetStrategyOrderListRequest{StrategyID: "119b6211-2611-461b-be5e-5ac557099e82", BeginTime: time.Unix(1773800000, 0), EndTime: time.Unix(1773700000, 0)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetStrategyOrderList must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetStrategyOrderList(t.Context(), &GetStrategyOrderListRequest{
		StrategyID:   "119b6211-2611-461b-be5e-5ac557099e82",
		Status:       5,
		Symbol:       "BTCUSDT",
		StrategyType: "twap",
		BeginTime:    time.Unix(1773700000, 0),
		EndTime:      time.Unix(1773800000, 0),
		PageSize:     2,
		Cursor:       "eyJPZmZzZXQiOjIsIlBhZ2VTaXplIjoyfQ==",
	})
	require.NoError(t, err, "GetStrategyOrderList must not error")
	if mockTests {
		exp := &StrategyOrderListResponse{
			List: []StrategyOrder{
				{
					StrategyID:    "119b6211-2611-461b-be5e-5ac557099e82",
					OrderID:       "d67e97b1-7a6e-4a89-b9d7-098756e8b86f",
					Symbol:        "BTCUSDT",
					Side:          "Buy",
					Size:          0.004,
					Status:        "3",
					ParentOrderID: "276a053f-fedf-4834-bbab-d5fc3bc0dc94",
					CreatedTime:   types.Time(time.UnixMilli(1773713015709)),
					UpdatedTime:   types.Time(time.UnixMilli(1773713020768)),
					Code:          30208,
					Message:       "Failed to submit order(s). The order price is higher than the maximum buying price.",
					Category:      "UTA_USDT",
					OrderType:     "UNKNOWN",
				},
				{
					StrategyID:          "119b6211-2611-461b-be5e-5ac557099e82",
					OrderID:             "8a3c1e52-0d4b-4f7e-9a61-3b2c5d7e9f10",
					Symbol:              "BTCUSDT",
					Side:                "Buy",
					Size:                0.004,
					Price:               76695.5,
					Status:              "5",
					ExecutedSize:        0.004,
					DealTime:            types.Time(time.UnixMilli(1773713080125)),
					CreatedTime:         types.Time(time.UnixMilli(1773713075628)),
					UpdatedTime:         types.Time(time.UnixMilli(1773713080125)),
					Category:            "UTA_USDT",
					PositionIndex:       1,
					OrderType:           "UNKNOWN",
					LeverageType:        1,
					TPPrice:             90000,
					SLPrice:             70000,
					OrderPriceOffset:    0.1,
					PositionValue:       306.78,
					FilledPositionValue: 306.78,
				},
			},
			NextCursor:     "eyJPZmZzZXQiOjQsIlBhZ2VTaXplIjoyfQ==",
			PreviousCursor: "eyJPZmZzZXQiOjAsIlBhZ2VTaXplIjoyfQ==",
		}
		assert.Equal(t, exp, result, "GetStrategyOrderList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetStrategyOrderList should return a result")
}

func TestStopStrategy(t *testing.T) {
	t.Parallel()
	_, err := e.StopStrategy(t.Context(), "")
	require.ErrorIs(t, err, errStrategyIDRequired, "StopStrategy must reject an empty strategy ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.StopStrategy(t.Context(), "119b6211-2611-461b-be5e-5ac557099e82")
	require.NoError(t, err, "StopStrategy must not error")
	if mockTests {
		assert.Equal(t, &StopStrategyResponse{StrategyID: "119b6211-2611-461b-be5e-5ac557099e82"}, result, "StopStrategy should decode every field")
		return
	}
	assert.NotEmpty(t, result.StrategyID, "StopStrategy should return the stopped strategy's ID")
}
