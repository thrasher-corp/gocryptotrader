package kraken

import (
	"math"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

const (
	futuresTradingLimitOrderID    = "4f1c3a5e-7b2d-4e8f-9a6b-1c2d3e4f5a6b"
	futuresTradingStopOrderID     = "a3b4c5d6-e7f8-4a1b-8c2d-3e4f5a6b7c8d"
	futuresTradingTrailingOrderID = "b4c5d6e7-f8a9-4b2c-9d3e-4f5a6b7c8d9e"
	futuresTradingPostOrderID     = "c5d6e7f8-a9b0-4c3d-8e4f-5a6b7c8d9e0f"
	futuresTradingRestingOrderID  = "d6e7f8a9-b0c1-4d2e-9f3a-4b5c6d7e8f90"
	futuresTradingBatchLimitID    = "e7f8a9b0-c1d2-4e3f-8a4b-5c6d7e8f9a0b"
	futuresTradingBatchStopID     = "f8a9b0c1-d2e3-4f4a-9b5c-6d7e8f9a0b1c"
	futuresTradingBatchTrailingID = "09b0c1d2-e3f4-4a5b-8c6d-7e8f9a0b1c2d"
	futuresTradingTrailing3ID     = "1a2b3c4d-5e6f-4a7b-9c8d-0e1f2a3b4c5d"
)

// futuresTradingTestTime returns a time on 9 October 2026 UTC, the day the Derivatives REST fixtures are set on
func futuresTradingTestTime(hour, minute, second, millisecond int) time.Time {
	return time.Date(2026, 10, 9, hour, minute, second, millisecond*int(time.Millisecond), time.UTC)
}

func TestSendFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := e.SendFuturesOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SendFuturesOrder must reject a nil request")
	_, err = e.SendFuturesOrder(t.Context(), &FuturesSendOrderRequest{Symbol: futuresTestPair, Side: "sell", OrderType: "stp", Size: 0.001})
	require.ErrorIs(t, err, errFuturesStopPriceRequired, "SendFuturesOrder must validate the order")
	_, err = e.SendFuturesOrder(t.Context(), &FuturesSendOrderRequest{Symbol: futuresTestPair, Side: "sell", OrderType: "stp", Size: 0.001, StopPrice: 79000, LimitPriceOffsetValue: -0.5})
	require.ErrorIs(t, err, errFuturesLimitPriceOffsetUnitRequired, "SendFuturesOrder must reject a limit price offset without its unit")
	for _, offset := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		_, err = e.SendFuturesOrder(t.Context(), &FuturesSendOrderRequest{Symbol: futuresTestPair, Side: "sell", OrderType: "stp", Size: 0.001, StopPrice: 79000, LimitPriceOffsetValue: offset, LimitPriceOffsetUnit: "QUOTE_CURRENCY"})
		require.ErrorIsf(t, err, errInvalidOrderPrice, "SendFuturesOrder must reject a limit price offset of %v", offset)
	}
	_, err = e.SendFuturesOrder(t.Context(), &FuturesSendOrderRequest{Symbol: futuresTestPair, Side: "buy", OrderType: "lmt", Size: 0.001, LimitPrice: math.NaN()})
	require.ErrorIs(t, err, errInvalidOrderPrice, "SendFuturesOrder must reject a NaN limit price")
	_, err = new(Exchange).SendFuturesOrder(t.Context(), &FuturesSendOrderRequest{Symbol: futuresTestPair, Side: "buy", OrderType: "lmt", Size: 0.001, LimitPrice: 81500})
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "SendFuturesOrder must return the error formatting the symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *FuturesSendOrderRequest
		exp  *FuturesSendOrderResponse
	}{
		{
			name: "limit order executing in part",
			req: &FuturesSendOrderRequest{
				Symbol:        futuresTestPair,
				Side:          "buy",
				OrderType:     "lmt",
				Size:          0.0005,
				LimitPrice:    81500,
				ClientOrderID: "gct-limit-1",
				ReduceOnly:    true,
				Broker:        "AB12CD34EF56GH78",
				// Sent in UTC, as 08:15:31.123456Z
				ProcessBefore: time.Date(2026, 10, 9, 19, 15, 31, 123456000, time.FixedZone("AEDT", 11*60*60)),
				AlgorithmID:   "gct-algo-1",
			},
			exp: &FuturesSendOrderResponse{
				SendStatus: FuturesSendOrderStatus{
					OrderID:       futuresTradingLimitOrderID,
					ClientOrderID: "gct-limit-1",
					Status:        "placed",
					ReceivedTime:  futuresTradingTestTime(8, 15, 30, 412),
					OrderEvents: []FuturesOrderEvent{
						{
							Type:           "EXECUTION",
							ExecutionID:    "8d2e6f10-3c4b-4a59-b7e8-0f1a2b3c4d5e",
							ExecutionPrice: 81496,
							ExecutedAmount: 0.0002,
							OrderPriorExecution: &FuturesOrder{
								OrderID:        futuresTradingLimitOrderID,
								ClientOrderID:  "gct-limit-1",
								OrderType:      "lmt",
								Symbol:         "PF_XBTUSD",
								Side:           "buy",
								Quantity:       0.0004,
								LimitPrice:     81500,
								ReduceOnly:     true,
								PlacedTime:     futuresTradingTestTime(8, 15, 30, 412),
								LastUpdateTime: futuresTradingTestTime(8, 15, 30, 412),
								AlgorithmID:    "gct-algo-1",
							},
							TakerReducedQuantity: 0.0001,
						},
						{
							Type: "PLACE",
							Order: &FuturesOrder{
								OrderID:        futuresTradingLimitOrderID,
								ClientOrderID:  "gct-limit-1",
								OrderType:      "lmt",
								Symbol:         "PF_XBTUSD",
								Side:           "buy",
								Quantity:       0.0004,
								FilledQuantity: 0.0002,
								LimitPrice:     81500,
								ReduceOnly:     true,
								PlacedTime:     futuresTradingTestTime(8, 15, 30, 412),
								LastUpdateTime: futuresTradingTestTime(8, 15, 30, 413),
								AlgorithmID:    "gct-algo-1",
							},
							ReducedQuantity: 0.0001,
						},
					},
				},
				ServerTime: futuresTradingTestTime(8, 15, 30, 415),
			},
		},
		{
			name: "stop order with a relative limit price",
			req: &FuturesSendOrderRequest{
				Symbol:                futuresTestPair,
				Side:                  "sell",
				OrderType:             "stp",
				Size:                  0.001,
				StopPrice:             79000,
				ClientOrderID:         "gct-stop-1",
				TriggerSignal:         "mark",
				ReduceOnly:            true,
				LimitPriceOffsetValue: -0.5,
				LimitPriceOffsetUnit:  "PERCENT",
			},
			exp: &FuturesSendOrderResponse{
				SendStatus: FuturesSendOrderStatus{
					OrderID:       futuresTradingStopOrderID,
					ClientOrderID: "gct-stop-1",
					Status:        "placed",
					ReceivedTime:  futuresTradingTestTime(8, 16, 2, 118),
					OrderEvents: []FuturesOrderEvent{
						{
							Type: "PLACE",
							OrderTrigger: &FuturesOrderTrigger{
								OrderID:        futuresTradingStopOrderID,
								ClientOrderID:  "gct-stop-1",
								OrderType:      "lmt",
								Symbol:         "PF_XBTUSD",
								Side:           "sell",
								Quantity:       0.001,
								LimitPrice:     78605,
								TriggerPrice:   79000,
								TriggerSide:    "trigger_below",
								TriggerSignal:  "mark_price",
								ReduceOnly:     true,
								PlacedTime:     futuresTradingTestTime(8, 16, 2, 118),
								LastUpdateTime: futuresTradingTestTime(8, 16, 2, 119),
								StartTime:      futuresTradingTestTime(8, 16, 2, 120),
							},
						},
					},
				},
				ServerTime: futuresTradingTestTime(8, 16, 2, 121),
			},
		},
		{
			name: "trailing stop rejected",
			req: &FuturesSendOrderRequest{
				Symbol:                       futuresTestPair,
				Side:                         "sell",
				OrderType:                    "trailing_stop",
				Size:                         0.002,
				ClientOrderID:                "gct-trailing-1",
				TriggerSignal:                "last",
				TrailingStopMaximumDeviation: 1.5,
				TrailingStopDeviationUnit:    "PERCENT",
			},
			exp: &FuturesSendOrderResponse{
				SendStatus: FuturesSendOrderStatus{
					OrderID:       futuresTradingTrailingOrderID,
					ClientOrderID: "gct-trailing-1",
					Status:        "insufficientAvailableFunds",
					ReceivedTime:  futuresTradingTestTime(8, 17, 45, 250),
					OrderEvents: []FuturesOrderEvent{
						{
							Type:    "REJECT",
							OrderID: futuresTradingTrailingOrderID,
							OrderTrigger: &FuturesOrderTrigger{
								OrderID:        futuresTradingTrailingOrderID,
								ClientOrderID:  "gct-trailing-1",
								OrderType:      "lmt",
								Symbol:         "PF_XBTUSD",
								Side:           "sell",
								Quantity:       0.002,
								TriggerPrice:   80592,
								TriggerSide:    "trigger_below",
								TriggerSignal:  "last_price",
								PlacedTime:     futuresTradingTestTime(8, 17, 45, 250),
								LastUpdateTime: futuresTradingTestTime(8, 17, 45, 251),
							},
							Reason: "INSUFFICIENT_MARGIN",
						},
					},
				},
				ServerTime: futuresTradingTestTime(8, 17, 45, 252),
			},
		},
		{
			name: "post-only order that would execute",
			req:  &FuturesSendOrderRequest{Symbol: futuresTestPair, Side: "buy", OrderType: "post", Size: 0.0005, LimitPrice: 81900},
			exp: &FuturesSendOrderResponse{
				SendStatus: FuturesSendOrderStatus{
					OrderID:      futuresTradingPostOrderID,
					Status:       "postWouldExecute",
					ReceivedTime: futuresTradingTestTime(8, 18, 10, 4),
					OrderEvents: []FuturesOrderEvent{
						{
							Type:    "REJECT",
							OrderID: futuresTradingPostOrderID,
							Order: &FuturesOrder{
								OrderID:        futuresTradingPostOrderID,
								OrderType:      "post",
								Symbol:         "PF_XBTUSD",
								Side:           "buy",
								Quantity:       0.0005,
								LimitPrice:     81900,
								PlacedTime:     futuresTradingTestTime(8, 18, 10, 4),
								LastUpdateTime: futuresTradingTestTime(8, 18, 10, 5),
							},
							Reason: "POST_WOULD_EXECUTE",
						},
					},
				},
				ServerTime: futuresTradingTestTime(8, 18, 10, 6),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.SendFuturesOrder(t.Context(), tc.req)
			require.NoError(t, err, "SendFuturesOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "SendFuturesOrder should decode every field")
				return
			}
			assert.NotEmpty(t, result.SendStatus.Status, "SendFuturesOrder should return a status")
		})
	}
}

func TestEditFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := e.EditFuturesOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "EditFuturesOrder must reject a nil request")
	_, err = e.EditFuturesOrder(t.Context(), &FuturesEditOrderRequest{Size: 1})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "EditFuturesOrder must reject a request without an order ID")
	_, err = e.EditFuturesOrder(t.Context(), &FuturesEditOrderRequest{OrderID: futuresTradingTrailingOrderID, TrailingStopMaximumDeviation: 51, TrailingStopDeviationUnit: "PERCENT"})
	require.ErrorIs(t, err, errFuturesInvalidTrailingStopDeviation, "EditFuturesOrder must reject a percentage deviation above 50")
	for _, v := range []float64{-1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		_, err = e.EditFuturesOrder(t.Context(), &FuturesEditOrderRequest{OrderID: futuresTradingLimitOrderID, Size: v})
		require.ErrorIsf(t, err, order.ErrAmountIsInvalid, "EditFuturesOrder must reject a size of %v", v)
		_, err = e.EditFuturesOrder(t.Context(), &FuturesEditOrderRequest{OrderID: futuresTradingLimitOrderID, LimitPrice: v})
		require.ErrorIsf(t, err, errInvalidOrderPrice, "EditFuturesOrder must reject a limit price of %v", v)
		_, err = e.EditFuturesOrder(t.Context(), &FuturesEditOrderRequest{OrderID: futuresTradingStopOrderID, StopPrice: v})
		require.ErrorIsf(t, err, errInvalidOrderPrice, "EditFuturesOrder must reject a stop price of %v", v)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	orderBeforeEdit := &FuturesOrder{
		OrderID:        futuresTradingLimitOrderID,
		ClientOrderID:  "gct-limit-1",
		OrderType:      "lmt",
		Symbol:         "PF_XBTUSD",
		Side:           "buy",
		Quantity:       0.0004,
		FilledQuantity: 0.0002,
		LimitPrice:     81500,
		ReduceOnly:     true,
		PlacedTime:     futuresTradingTestTime(8, 15, 30, 412),
		LastUpdateTime: futuresTradingTestTime(8, 15, 30, 413),
		AlgorithmID:    "gct-algo-1",
	}
	orderAfterEdit := &FuturesOrder{
		OrderID:        futuresTradingLimitOrderID,
		ClientOrderID:  "gct-limit-1",
		OrderType:      "lmt",
		Symbol:         "PF_XBTUSD",
		Side:           "buy",
		Quantity:       0.0006,
		FilledQuantity: 0.0002,
		LimitPrice:     81520,
		ReduceOnly:     true,
		PlacedTime:     futuresTradingTestTime(8, 15, 30, 412),
		LastUpdateTime: futuresTradingTestTime(8, 19, 58, 730),
		AlgorithmID:    "gct-algo-1",
	}
	for _, tc := range []struct {
		name string
		req  *FuturesEditOrderRequest
		exp  *FuturesEditOrderResponse
	}{
		{
			name: "absolute resize by order ID",
			req: &FuturesEditOrderRequest{
				OrderID:       futuresTradingLimitOrderID,
				Size:          0.0006,
				LimitPrice:    81520,
				QuantityMode:  "ABSOLUTE",
				ProcessBefore: time.Date(2026, 10, 9, 8, 20, 0, 500000000, time.UTC),
				AlgorithmID:   "gct-algo-1",
			},
			exp: &FuturesEditOrderResponse{
				EditStatus: FuturesEditOrderStatus{
					OrderID:       futuresTradingLimitOrderID,
					ClientOrderID: "gct-limit-1",
					Status:        "edited",
					ReceivedTime:  futuresTradingTestTime(8, 19, 58, 729),
					OrderEvents: []FuturesOrderEvent{
						{Type: "EDIT", OldOrder: orderBeforeEdit, NewOrder: orderAfterEdit, ReducedQuantity: 0.0001},
						{
							Type:                 "EXECUTION",
							ExecutionID:          "9e3f7a21-4d5c-4b6a-8c9d-1a2b3c4d5e6f",
							ExecutionPrice:       81518,
							ExecutedAmount:       0.0001,
							OrderPriorEdit:       orderBeforeEdit,
							OrderPriorExecution:  orderAfterEdit,
							TakerReducedQuantity: 0.00005,
						},
					},
				},
				ServerTime: futuresTradingTestTime(8, 19, 58, 735),
			},
		},
		{
			name: "stop by client order ID once gone",
			req: &FuturesEditOrderRequest{
				ClientOrderID:                "gct-stop-1",
				StopPrice:                    78800,
				TrailingStopMaximumDeviation: 250,
				TrailingStopDeviationUnit:    "QUOTE_CURRENCY",
			},
			exp: &FuturesEditOrderResponse{
				EditStatus: FuturesEditOrderStatus{
					ClientOrderID: "gct-stop-1",
					Status:        "orderForEditNotFound",
					ReceivedTime:  futuresTradingTestTime(8, 20, 12, 54),
					OrderEvents:   []FuturesOrderEvent{},
				},
				ServerTime: futuresTradingTestTime(8, 20, 12, 57),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.EditFuturesOrder(t.Context(), tc.req)
			require.NoError(t, err, "EditFuturesOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "EditFuturesOrder should decode every field")
				return
			}
			assert.NotEmpty(t, result.EditStatus.Status, "EditFuturesOrder should return a status")
		})
	}
}

func TestCancelFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelFuturesOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelFuturesOrder must reject a nil request")
	_, err = e.CancelFuturesOrder(t.Context(), &FuturesCancelOrderRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelFuturesOrder must reject a request without an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *FuturesCancelOrderRequest
		exp  *FuturesCancelOrderResponse
	}{
		{
			name: "order by order ID",
			req: &FuturesCancelOrderRequest{
				OrderID:       futuresTradingLimitOrderID,
				ProcessBefore: time.Date(2026, 10, 9, 8, 21, 0, 0, time.UTC),
				AlgorithmID:   "gct-algo-1",
			},
			exp: &FuturesCancelOrderResponse{
				CancelStatus: FuturesCancelOrderStatus{
					OrderID:       futuresTradingLimitOrderID,
					ClientOrderID: "gct-limit-1",
					Status:        "cancelled",
					ReceivedTime:  futuresTradingTestTime(8, 20, 41, 377),
					OrderEvents: []FuturesOrderEvent{
						{
							Type:    "CANCEL",
							OrderID: futuresTradingLimitOrderID,
							Order: &FuturesOrder{
								OrderID:        futuresTradingLimitOrderID,
								ClientOrderID:  "gct-limit-1",
								OrderType:      "lmt",
								Symbol:         "PF_XBTUSD",
								Side:           "buy",
								Quantity:       0.0006,
								FilledQuantity: 0.0003,
								LimitPrice:     81520,
								ReduceOnly:     true,
								PlacedTime:     futuresTradingTestTime(8, 15, 30, 412),
								LastUpdateTime: futuresTradingTestTime(8, 19, 58, 730),
								AlgorithmID:    "gct-algo-1",
							},
						},
					},
				},
				ServerTime: futuresTradingTestTime(8, 20, 41, 380),
			},
		},
		{
			name: "trigger order by client order ID",
			req:  &FuturesCancelOrderRequest{ClientOrderID: "gct-stop-1"},
			exp: &FuturesCancelOrderResponse{
				CancelStatus: FuturesCancelOrderStatus{
					OrderID:       futuresTradingStopOrderID,
					ClientOrderID: "gct-stop-1",
					Status:        "cancelled",
					ReceivedTime:  futuresTradingTestTime(8, 20, 55, 910),
					OrderEvents: []FuturesOrderEvent{
						{
							Type:    "CANCEL",
							OrderID: futuresTradingStopOrderID,
							OrderTrigger: &FuturesOrderTrigger{
								OrderID:        futuresTradingStopOrderID,
								ClientOrderID:  "gct-stop-1",
								OrderType:      "lmt",
								Symbol:         "PF_XBTUSD",
								Side:           "sell",
								Quantity:       0.001,
								LimitPrice:     78605,
								TriggerPrice:   79000,
								TriggerSide:    "trigger_below",
								TriggerSignal:  "mark_price",
								ReduceOnly:     true,
								PlacedTime:     futuresTradingTestTime(8, 16, 2, 118),
								LastUpdateTime: futuresTradingTestTime(8, 16, 2, 119),
								StartTime:      futuresTradingTestTime(8, 16, 2, 120),
							},
						},
					},
				},
				ServerTime: futuresTradingTestTime(8, 20, 55, 913),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.CancelFuturesOrder(t.Context(), tc.req)
			require.NoError(t, err, "CancelFuturesOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "CancelFuturesOrder should decode every field")
				return
			}
			assert.NotEmpty(t, result.CancelStatus.Status, "CancelFuturesOrder should return a status")
		})
	}
}

func TestCancelAllFuturesOrders(t *testing.T) {
	t.Parallel()
	for _, pair := range partialTestPairs {
		_, err := e.CancelAllFuturesOrders(t.Context(), &FuturesCancelAllOrdersRequest{Symbol: pair})
		require.ErrorIsf(t, err, currency.ErrCurrencyCodeEmpty, "CancelAllFuturesOrders must reject %q, which lacks a base or a quote", pair)
	}
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *FuturesCancelAllOrdersRequest
		exp  *FuturesCancelAllOrdersResponse
	}{
		{
			name: "every order",
			exp: &FuturesCancelAllOrdersResponse{
				CancelStatus: FuturesCancelAllOrdersStatus{
					CancelOnly:   "all",
					Status:       "cancelled",
					ReceivedTime: futuresTradingTestTime(8, 22, 3, 551),
					CancelledOrders: []FuturesCancelledOrder{
						{OrderID: futuresTradingRestingOrderID, ClientOrderID: "gct-limit-2"},
						{OrderID: futuresTradingStopOrderID, ClientOrderID: "gct-stop-2"},
					},
					OrderEvents: []FuturesOrderEvent{
						{
							Type:    "CANCEL",
							OrderID: futuresTradingRestingOrderID,
							Order: &FuturesOrder{
								OrderID:        futuresTradingRestingOrderID,
								ClientOrderID:  "gct-limit-2",
								OrderType:      "lmt",
								Symbol:         "PF_XBTUSD",
								Side:           "sell",
								Quantity:       0.0008,
								FilledQuantity: 0.0001,
								LimitPrice:     82750,
								PlacedTime:     futuresTradingTestTime(7, 58, 12, 1),
								LastUpdateTime: futuresTradingTestTime(8, 1, 40, 270),
								AlgorithmID:    "gct-algo-2",
							},
						},
						{Type: "CANCEL", OrderID: futuresTradingStopOrderID},
					},
				},
				ServerTime: futuresTradingTestTime(8, 22, 3, 553),
			},
		},
		{
			name: "contract without orders",
			req:  &FuturesCancelAllOrdersRequest{Symbol: futuresTestPair, AlgorithmID: "gct-algo-1"},
			exp: &FuturesCancelAllOrdersResponse{
				CancelStatus: FuturesCancelAllOrdersStatus{
					CancelOnly:      "PF_XBTUSD",
					Status:          "noOrdersToCancel",
					ReceivedTime:    futuresTradingTestTime(8, 22, 30, 8),
					CancelledOrders: []FuturesCancelledOrder{},
					OrderEvents:     []FuturesOrderEvent{},
				},
				ServerTime: futuresTradingTestTime(8, 22, 30, 10),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.CancelAllFuturesOrders(t.Context(), tc.req)
			require.NoError(t, err, "CancelAllFuturesOrders must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "CancelAllFuturesOrders should decode every field")
				return
			}
			assert.NotEmpty(t, result.CancelStatus.Status, "CancelAllFuturesOrders should return a status")
		})
	}
}

func TestCancelAllFuturesOrdersAfter(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllFuturesOrdersAfter(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelAllFuturesOrdersAfter must reject a nil request")
	for _, timeout := range []time.Duration{-time.Second, 1500 * time.Millisecond, (1 << 32) * time.Second} {
		_, err = e.CancelAllFuturesOrdersAfter(t.Context(), &FuturesCancelAllOrdersAfterRequest{Timeout: timeout})
		require.ErrorIsf(t, err, errFuturesInvalidTimeout, "CancelAllFuturesOrdersAfter must reject a timeout of %s", timeout)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *FuturesCancelAllOrdersAfterRequest
		exp  *FuturesCancelAllOrdersAfterResponse
	}{
		{
			name: "armed",
			req:  &FuturesCancelAllOrdersAfterRequest{Timeout: time.Minute, AlgorithmID: "gct-algo-1"},
			exp: &FuturesCancelAllOrdersAfterResponse{
				Status:     FuturesDeadMansSwitchStatus{CurrentTime: futuresTradingTestTime(8, 23, 10, 250), TriggerTime: futuresTradingTestTime(8, 24, 10, 250)},
				ServerTime: futuresTradingTestTime(8, 23, 10, 251),
			},
		},
		{
			name: "deactivated",
			req:  &FuturesCancelAllOrdersAfterRequest{},
			exp: &FuturesCancelAllOrdersAfterResponse{
				Status:     FuturesDeadMansSwitchStatus{CurrentTime: futuresTradingTestTime(8, 23, 25, 700)},
				ServerTime: futuresTradingTestTime(8, 23, 25, 701),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.CancelAllFuturesOrdersAfter(t.Context(), tc.req)
			require.NoError(t, err, "CancelAllFuturesOrdersAfter must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "CancelAllFuturesOrdersAfter should decode every field")
				return
			}
			assert.NotZero(t, result.Status.CurrentTime, "CancelAllFuturesOrdersAfter should return the current time")
		})
	}
}

func TestFuturesDeadMansSwitchStatusUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var status FuturesDeadMansSwitchStatus
	require.NoError(t, status.UnmarshalJSON([]byte(`{"currentTime":"2026-10-09T08:23:10.250Z","triggerTime":"2026-10-09T08:24:10.250Z"}`)), "UnmarshalJSON must not error for an armed switch")
	assert.Equal(t, FuturesDeadMansSwitchStatus{CurrentTime: futuresTradingTestTime(8, 23, 10, 250), TriggerTime: futuresTradingTestTime(8, 24, 10, 250)}, status, "UnmarshalJSON should decode both times")
	require.NoError(t, status.UnmarshalJSON([]byte(`{"currentTime":"2026-10-09T08:23:25.700Z","triggerTime":"0"}`)), "UnmarshalJSON must not error for a deactivated switch")
	assert.Equal(t, FuturesDeadMansSwitchStatus{CurrentTime: futuresTradingTestTime(8, 23, 25, 700)}, status, "UnmarshalJSON should clear the trigger time of a deactivated switch")
	var parseErr *time.ParseError
	assert.ErrorAs(t, status.UnmarshalJSON([]byte(`{"currentTime":"2026-10-09T08:23:25.700Z","triggerTime":"soon"}`)), &parseErr, "UnmarshalJSON should reject a trigger time that is not RFC 3339")
	assert.Error(t, status.UnmarshalJSON([]byte(`[]`)), "UnmarshalJSON should reject a status that is not an object")
}

func TestFuturesBatchOrder(t *testing.T) {
	t.Parallel()
	_, err := e.FuturesBatchOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "FuturesBatchOrder must reject a nil request")
	_, err = e.FuturesBatchOrder(t.Context(), &FuturesBatchOrderRequest{})
	require.ErrorIs(t, err, common.ErrEmptyParams, "FuturesBatchOrder must reject a request without instructions")
	_, err = e.FuturesBatchOrder(t.Context(), &FuturesBatchOrderRequest{Instructions: make([]FuturesBatchInstruction, 501)})
	require.ErrorIs(t, err, errFuturesTooManyEntries, "FuturesBatchOrder must reject more than 500 instructions")
	for _, tc := range []struct {
		name        string
		instruction FuturesBatchInstruction
		err         error
	}{
		{name: "no kind", err: errFuturesInvalidBatchInstruction},
		{
			name:        "two kinds",
			instruction: FuturesBatchInstruction{Edit: &FuturesBatchEditInstruction{OrderID: futuresTradingBatchLimitID}, Cancel: &FuturesBatchCancelInstruction{OrderID: futuresTradingBatchLimitID}},
			err:         errFuturesInvalidBatchInstruction,
		},
		{name: "send without an order tag", instruction: FuturesBatchInstruction{Send: &FuturesBatchSendInstruction{}}, err: errFuturesOrderTagRequired},
		{name: "send without a symbol", instruction: FuturesBatchInstruction{Send: &FuturesBatchSendInstruction{OrderTag: "1"}}, err: currency.ErrCurrencyPairEmpty},
		{name: "edit without an order ID", instruction: FuturesBatchInstruction{Edit: &FuturesBatchEditInstruction{Size: 1}}, err: order.ErrOrderIDNotSet},
		{
			name:        "edit with a percentage deviation below 0.1",
			instruction: FuturesBatchInstruction{Edit: &FuturesBatchEditInstruction{OrderID: futuresTradingBatchTrailingID, TrailingStopMaximumDeviation: 0.05, TrailingStopDeviationUnit: "PERCENT"}},
			err:         errFuturesInvalidTrailingStopDeviation,
		},
		{name: "cancel without an order ID", instruction: FuturesBatchInstruction{Cancel: &FuturesBatchCancelInstruction{}}, err: order.ErrOrderIDNotSet},
	} {
		_, err = e.FuturesBatchOrder(t.Context(), &FuturesBatchOrderRequest{Instructions: []FuturesBatchInstruction{tc.instruction}})
		require.ErrorIsf(t, err, tc.err, "FuturesBatchOrder must reject an invalid instruction: %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	batchLimitBeforeEdit := &FuturesOrder{
		OrderID:        futuresTradingBatchLimitID,
		ClientOrderID:  "gct-batch-1",
		OrderType:      "lmt",
		Symbol:         "PF_XBTUSD",
		Side:           "buy",
		Quantity:       0.0003,
		FilledQuantity: 0.0001,
		LimitPrice:     81400,
		ReduceOnly:     true,
		PlacedTime:     futuresTradingTestTime(8, 24, 59, 120),
		LastUpdateTime: futuresTradingTestTime(8, 24, 59, 121),
		AlgorithmID:    "gct-algo-1",
	}
	for _, tc := range []struct {
		name string
		req  *FuturesBatchOrderRequest
		exp  *FuturesBatchOrderResponse
	}{
		{
			name: "sends",
			req: &FuturesBatchOrderRequest{
				Instructions: []FuturesBatchInstruction{
					{Send: &FuturesBatchSendInstruction{OrderTag: "1", Symbol: futuresTestPair, Side: "buy", OrderType: "lmt", Size: 0.0005, LimitPrice: 81400, ClientOrderID: "gct-batch-1", ReduceOnly: true}},
					{Send: &FuturesBatchSendInstruction{OrderTag: "2", Symbol: futuresTestPair, Side: "sell", OrderType: "stp", Size: 0.0005, LimitPrice: 78900, StopPrice: 79000, TriggerSignal: "mark"}},
					{Send: &FuturesBatchSendInstruction{
						OrderTag:                     "3",
						Symbol:                       futuresTestPair,
						Side:                         "sell",
						OrderType:                    "trailing_stop",
						Size:                         0.0005,
						TriggerSignal:                "index",
						TrailingStopMaximumDeviation: 300,
						TrailingStopDeviationUnit:    "QUOTE_CURRENCY",
					}},
				},
				Broker:        "AB12CD34EF56GH78",
				ProcessBefore: time.Date(2026, 10, 9, 8, 25, 0, 250000000, time.UTC),
				AlgorithmID:   "gct-algo-1",
			},
			exp: &FuturesBatchOrderResponse{
				BatchStatus: []FuturesBatchInstructionResult{
					{
						OrderTag:      "1",
						OrderID:       futuresTradingBatchLimitID,
						ClientOrderID: "gct-batch-1",
						Status:        "placed",
						ReceivedTime:  futuresTradingTestTime(8, 24, 59, 120),
						OrderEvents:   []FuturesOrderEvent{{Type: "PLACE", Order: batchLimitBeforeEdit, ReducedQuantity: 0.0002}},
					},
					{
						OrderTag:     "2",
						OrderID:      futuresTradingBatchStopID,
						Status:       "placed",
						ReceivedTime: futuresTradingTestTime(8, 24, 59, 122),
						OrderEvents: []FuturesOrderEvent{
							{
								Type: "PLACE",
								OrderTrigger: &FuturesOrderTrigger{
									OrderID:        futuresTradingBatchStopID,
									OrderType:      "lmt",
									Symbol:         "PF_XBTUSD",
									Side:           "sell",
									Quantity:       0.0005,
									LimitPrice:     78900,
									TriggerPrice:   79000,
									TriggerSide:    "trigger_below",
									TriggerSignal:  "mark_price",
									PlacedTime:     futuresTradingTestTime(8, 24, 59, 122),
									LastUpdateTime: futuresTradingTestTime(8, 24, 59, 123),
								},
							},
						},
					},
					{
						OrderTag:     "3",
						OrderID:      futuresTradingBatchTrailingID,
						Status:       "insufficientAvailableFunds",
						ReceivedTime: futuresTradingTestTime(8, 24, 59, 124),
						OrderEvents: []FuturesOrderEvent{
							{
								Type:    "REJECT",
								OrderID: futuresTradingBatchTrailingID,
								OrderTrigger: &FuturesOrderTrigger{
									OrderID:        futuresTradingBatchTrailingID,
									OrderType:      "lmt",
									Symbol:         "PF_XBTUSD",
									Side:           "sell",
									Quantity:       0.0005,
									TriggerPrice:   81108,
									TriggerSide:    "trigger_below",
									TriggerSignal:  "spot_price",
									PlacedTime:     futuresTradingTestTime(8, 24, 59, 124),
									LastUpdateTime: futuresTradingTestTime(8, 24, 59, 125),
								},
								Reason: "INSUFFICIENT_MARGIN",
							},
						},
					},
				},
				ServerTime: futuresTradingTestTime(8, 24, 59, 130),
			},
		},
		{
			name: "edits",
			req: &FuturesBatchOrderRequest{
				Instructions: []FuturesBatchInstruction{
					{Edit: &FuturesBatchEditInstruction{OrderID: futuresTradingBatchLimitID, ClientOrderID: "gct-batch-1", Size: 0.0007, LimitPrice: 81450, QuantityMode: "ABSOLUTE"}},
					{Edit: &FuturesBatchEditInstruction{OrderID: futuresTradingBatchStopID, LimitPrice: 79000, StopPrice: 79100}},
					{Edit: &FuturesBatchEditInstruction{ClientOrderID: "gct-trailing-3", TrailingStopMaximumDeviation: 0.8, TrailingStopDeviationUnit: "PERCENT"}},
				},
			},
			exp: &FuturesBatchOrderResponse{
				BatchStatus: []FuturesBatchInstructionResult{
					{
						OrderID:       futuresTradingBatchLimitID,
						ClientOrderID: "gct-batch-1",
						Status:        "edited",
						ReceivedTime:  futuresTradingTestTime(8, 26, 14, 981),
						OrderEvents: []FuturesOrderEvent{
							{
								Type:     "EDIT",
								OldOrder: batchLimitBeforeEdit,
								NewOrder: &FuturesOrder{
									OrderID:        futuresTradingBatchLimitID,
									ClientOrderID:  "gct-batch-1",
									OrderType:      "lmt",
									Symbol:         "PF_XBTUSD",
									Side:           "buy",
									Quantity:       0.0006,
									FilledQuantity: 0.0001,
									LimitPrice:     81450,
									ReduceOnly:     true,
									PlacedTime:     futuresTradingTestTime(8, 24, 59, 120),
									LastUpdateTime: futuresTradingTestTime(8, 26, 14, 982),
									AlgorithmID:    "gct-algo-1",
								},
								ReducedQuantity: 0.0001,
							},
						},
					},
					{
						OrderID:       futuresTradingBatchStopID,
						ClientOrderID: "gct-stop-3",
						Status:        "edited",
						ReceivedTime:  futuresTradingTestTime(8, 26, 15, 3),
						OrderEvents: []FuturesOrderEvent{
							{
								Type: "EDIT",
								OldOrder: &FuturesOrder{
									OrderID:        futuresTradingBatchStopID,
									ClientOrderID:  "gct-stop-3",
									OrderType:      "stp",
									Symbol:         "PF_XBTUSD",
									Side:           "sell",
									Quantity:       0.0005,
									FilledQuantity: 0.0002,
									LimitPrice:     78900,
									PlacedTime:     futuresTradingTestTime(8, 24, 59, 122),
									LastUpdateTime: futuresTradingTestTime(8, 24, 59, 123),
									AlgorithmID:    "gct-algo-3",
								},
								NewOrder: &FuturesOrder{
									OrderID:        futuresTradingBatchStopID,
									ClientOrderID:  "gct-stop-3",
									OrderType:      "stp",
									Symbol:         "PF_XBTUSD",
									Side:           "sell",
									Quantity:       0.0005,
									FilledQuantity: 0.0002,
									LimitPrice:     79000,
									PlacedTime:     futuresTradingTestTime(8, 24, 59, 122),
									LastUpdateTime: futuresTradingTestTime(8, 26, 15, 4),
									AlgorithmID:    "gct-algo-3",
								},
							},
						},
					},
					{
						OrderID:       futuresTradingTrailing3ID,
						ClientOrderID: "gct-trailing-3",
						Status:        "invalidPrice",
						ReceivedTime:  futuresTradingTestTime(8, 26, 15, 19),
						OrderEvents:   []FuturesOrderEvent{},
					},
				},
				ServerTime: futuresTradingTestTime(8, 26, 15, 25),
			},
		},
		{
			name: "cancels",
			req: &FuturesBatchOrderRequest{
				Instructions: []FuturesBatchInstruction{
					{Cancel: &FuturesBatchCancelInstruction{OrderID: futuresTradingBatchLimitID}},
					{Cancel: &FuturesBatchCancelInstruction{ClientOrderID: "gct-trailing-3"}},
				},
			},
			exp: &FuturesBatchOrderResponse{
				BatchStatus: []FuturesBatchInstructionResult{
					{
						OrderID:       futuresTradingBatchLimitID,
						ClientOrderID: "gct-batch-1",
						Status:        "cancelled",
						ReceivedTime:  futuresTradingTestTime(8, 26, 40, 510),
						OrderEvents: []FuturesOrderEvent{
							{
								Type:    "CANCEL",
								OrderID: futuresTradingBatchLimitID,
								Order: &FuturesOrder{
									OrderID:        futuresTradingBatchLimitID,
									ClientOrderID:  "gct-batch-1",
									OrderType:      "lmt",
									Symbol:         "PF_XBTUSD",
									Side:           "buy",
									Quantity:       0.0006,
									FilledQuantity: 0.0003,
									LimitPrice:     81450,
									ReduceOnly:     true,
									PlacedTime:     futuresTradingTestTime(8, 24, 59, 120),
									LastUpdateTime: futuresTradingTestTime(8, 26, 14, 982),
									AlgorithmID:    "gct-algo-1",
								},
							},
						},
					},
					{
						OrderID:       futuresTradingTrailing3ID,
						ClientOrderID: "gct-trailing-3",
						Status:        "cancelled",
						ReceivedTime:  futuresTradingTestTime(8, 26, 40, 512),
						OrderEvents: []FuturesOrderEvent{
							{
								Type:    "CANCEL",
								OrderID: futuresTradingTrailing3ID,
								OrderTrigger: &FuturesOrderTrigger{
									OrderID:        futuresTradingTrailing3ID,
									ClientOrderID:  "gct-trailing-3",
									OrderType:      "lmt",
									Symbol:         "PF_XBTUSD",
									Side:           "sell",
									Quantity:       0.0005,
									LimitPrice:     80400,
									TriggerPrice:   80650,
									TriggerSide:    "trigger_below",
									TriggerSignal:  "last_price",
									ReduceOnly:     true,
									PlacedTime:     futuresTradingTestTime(8, 12, 31, 448),
									LastUpdateTime: futuresTradingTestTime(8, 26, 39, 903),
									StartTime:      futuresTradingTestTime(8, 12, 31, 500),
								},
							},
						},
					},
				},
				ServerTime: futuresTradingTestTime(8, 26, 40, 515),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.FuturesBatchOrder(t.Context(), tc.req)
			require.NoError(t, err, "FuturesBatchOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "FuturesBatchOrder should decode every field")
				return
			}
			assert.Len(t, result.BatchStatus, len(tc.req.Instructions), "FuturesBatchOrder should return a result for each instruction")
		})
	}
}

func TestGetFuturesOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesOpenOrders(t.Context())
	require.NoError(t, err, "GetFuturesOpenOrders must not error")
	if mockTests {
		exp := &FuturesOpenOrdersResponse{
			OpenOrders: []FuturesOpenOrder{
				{
					OrderID:        futuresTradingBatchLimitID,
					ClientOrderID:  "gct-batch-1",
					Symbol:         "PF_XBTUSD",
					Side:           "buy",
					OrderType:      "lmt",
					Status:         "partiallyFilled",
					LimitPrice:     81450,
					FilledSize:     0.0003,
					UnfilledSize:   0.0004,
					ReduceOnly:     true,
					ReceivedTime:   futuresTradingTestTime(8, 24, 59, 120),
					LastUpdateTime: futuresTradingTestTime(8, 26, 14, 982),
					AlgorithmID:    "gct-algo-1",
				},
				{
					OrderID:        futuresTradingBatchStopID,
					ClientOrderID:  "gct-stop-3",
					Symbol:         "PF_XBTUSD",
					Side:           "sell",
					OrderType:      "stp",
					Status:         "untouched",
					LimitPrice:     79000,
					StopPrice:      79100,
					FilledSize:     0.0002,
					UnfilledSize:   0.0003,
					TriggerSignal:  "mark",
					ReceivedTime:   futuresTradingTestTime(8, 24, 59, 122),
					LastUpdateTime: futuresTradingTestTime(8, 26, 15, 4),
					AlgorithmID:    "gct-algo-3",
				},
			},
			ServerTime: futuresTradingTestTime(8, 27, 0, 512),
		}
		assert.Equal(t, exp, result, "GetFuturesOpenOrders should decode every field")
		return
	}
	assert.NotZero(t, result.ServerTime, "GetFuturesOpenOrders should return the server time")
}

func TestGetFuturesOrdersStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrdersStatus(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesOrdersStatus must reject a nil request")
	_, err = e.GetFuturesOrdersStatus(t.Context(), &FuturesOrdersStatusRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetFuturesOrdersStatus must reject a request without IDs")
	_, err = e.GetFuturesOrdersStatus(t.Context(), &FuturesOrdersStatusRequest{OrderIDs: []string{futuresTradingBatchLimitID}, ClientOrderIDs: []string{""}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetFuturesOrdersStatus must reject an empty ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesOrdersStatus(t.Context(), &FuturesOrdersStatusRequest{
		OrderIDs:       []string{futuresTradingBatchLimitID, futuresTradingBatchStopID},
		ClientOrderIDs: []string{"gct-trailing-3"},
	})
	require.NoError(t, err, "GetFuturesOrdersStatus must not error")
	if mockTests {
		exp := &FuturesOrdersStatusResponse{
			Orders: []FuturesOrderStatusDetails{
				{
					Order: FuturesCachedOrder{
						Type:           "ORDER",
						OrderID:        futuresTradingBatchLimitID,
						ClientOrderID:  "gct-batch-1",
						Symbol:         "PF_XBTUSD",
						Side:           "buy",
						Quantity:       0.0007,
						FilledQuantity: 0.0003,
						LimitPrice:     81450,
						ReduceOnly:     true,
						PlacedTime:     futuresTradingTestTime(8, 24, 59, 120),
						LastUpdateTime: futuresTradingTestTime(8, 26, 14, 982),
						AlgorithmID:    "gct-algo-1",
					},
					Status:       "ENTERED_BOOK",
					UpdateReason: "PARTIAL_FILL",
				},
				{
					Order: FuturesCachedOrder{
						Type:           "TRIGGER_ORDER",
						OrderID:        futuresTradingTrailing3ID,
						ClientOrderID:  "gct-trailing-3",
						Symbol:         "PF_XBTUSD",
						Side:           "sell",
						Quantity:       0.0005,
						FilledQuantity: 0.0001,
						LimitPrice:     80400,
						ReduceOnly:     true,
						PlacedTime:     futuresTradingTestTime(8, 12, 31, 448),
						LastUpdateTime: futuresTradingTestTime(8, 27, 41, 302),
						AlgorithmID:    "gct-algo-4",
						PriceTriggerOptions: &FuturesPriceTriggerOptions{
							TriggerPrice:        80650,
							TriggerSide:         "TRIGGER_BELOW",
							TriggerSignal:       "LAST_PRICE",
							TrailingStopOptions: &FuturesTrailingStopOptions{MaximumDeviation: 0.8, Unit: "PERCENT"},
						},
						TriggerTime: futuresTradingTestTime(8, 27, 41, 300),
					},
					Status:       "TRIGGER_ACTIVATION_FAILURE",
					UpdateReason: "STOP_ORDER_TRIGGERED",
					Error:        "INSUFFICIENT_MARGIN",
				},
			},
			ServerTime: futuresTradingTestTime(8, 28, 0, 100),
		}
		assert.Equal(t, exp, result, "GetFuturesOrdersStatus should decode every field")
		return
	}
	assert.NotZero(t, result.ServerTime, "GetFuturesOrdersStatus should return the server time")
}

func TestFuturesValidateNewOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		symbol        currency.Pair
		side          string
		orderType     string
		size          float64
		limitPrice    float64
		stopPrice     float64
		deviation     float64
		deviationUnit string
		clientOrderID string
		err           error
	}{
		{name: "empty symbol", err: currency.ErrCurrencyPairEmpty},
		{name: "symbol without a quote", symbol: partialTestPairs[0], err: currency.ErrCurrencyPairEmpty},
		{name: "symbol without a base", symbol: partialTestPairs[1], err: currency.ErrCurrencyPairEmpty},
		{name: "empty side", symbol: futuresTestPair, err: order.ErrSideIsInvalid},
		{name: "empty order type", symbol: futuresTestPair, side: "buy", err: order.ErrTypeIsInvalid},
		{name: "zero size", symbol: futuresTestPair, side: "buy", orderType: "lmt", err: order.ErrAmountIsInvalid},
		{name: "negative size", symbol: futuresTestPair, side: "buy", orderType: "lmt", size: -1, err: order.ErrAmountIsInvalid},
		{name: "NaN size", symbol: futuresTestPair, side: "buy", orderType: "lmt", size: math.NaN(), err: order.ErrAmountIsInvalid},
		{name: "infinite size", symbol: futuresTestPair, side: "buy", orderType: "lmt", size: math.Inf(1), err: order.ErrAmountIsInvalid},
		{name: "negative infinite size", symbol: futuresTestPair, side: "buy", orderType: "lmt", size: math.Inf(-1), err: order.ErrAmountIsInvalid},
		{name: "negative limit price", symbol: futuresTestPair, side: "buy", orderType: "lmt", size: 1, limitPrice: -1, err: errInvalidOrderPrice},
		{name: "NaN limit price", symbol: futuresTestPair, side: "buy", orderType: "lmt", size: 1, limitPrice: math.NaN(), err: errInvalidOrderPrice},
		{name: "infinite limit price", symbol: futuresTestPair, side: "buy", orderType: "lmt", size: 1, limitPrice: math.Inf(1), err: errInvalidOrderPrice},
		{name: "negative infinite limit price", symbol: futuresTestPair, side: "buy", orderType: "lmt", size: 1, limitPrice: math.Inf(-1), err: errInvalidOrderPrice},
		{name: "negative stop price", symbol: futuresTestPair, side: "sell", orderType: "stp", size: 1, stopPrice: -1, err: errInvalidOrderPrice},
		{name: "NaN stop price", symbol: futuresTestPair, side: "sell", orderType: "stp", size: 1, stopPrice: math.NaN(), err: errInvalidOrderPrice},
		{name: "infinite stop price", symbol: futuresTestPair, side: "sell", orderType: "stp", size: 1, stopPrice: math.Inf(1), err: errInvalidOrderPrice},
		{name: "negative infinite stop price", symbol: futuresTestPair, side: "sell", orderType: "stp", size: 1, stopPrice: math.Inf(-1), err: errInvalidOrderPrice},
		{name: "stop without a stop price", symbol: futuresTestPair, side: "sell", orderType: "stp", size: 1, err: errFuturesStopPriceRequired},
		{name: "take profit without a stop price", symbol: futuresTestPair, side: "sell", orderType: "take_profit", size: 1, err: errFuturesStopPriceRequired},
		{name: "trailing stop without a deviation", symbol: futuresTestPair, side: "sell", orderType: "trailing_stop", size: 1, deviationUnit: "PERCENT", err: errFuturesTrailingStopDeviationRequired},
		{name: "trailing stop without a deviation unit", symbol: futuresTestPair, side: "sell", orderType: "trailing_stop", size: 1, deviation: 1, err: errFuturesTrailingStopDeviationRequired},
		{name: "percentage deviation above 50", symbol: futuresTestPair, side: "sell", orderType: "trailing_stop", size: 1, deviation: 50.5, deviationUnit: "PERCENT", err: errFuturesInvalidTrailingStopDeviation},
		{name: "NaN percentage deviation", symbol: futuresTestPair, side: "sell", orderType: "trailing_stop", size: 1, deviation: math.NaN(), deviationUnit: "PERCENT", err: errFuturesInvalidTrailingStopDeviation},
		{name: "negative quote currency deviation", symbol: futuresTestPair, side: "sell", orderType: "trailing_stop", size: 1, deviation: -100, deviationUnit: "QUOTE_CURRENCY", err: errFuturesInvalidTrailingStopDeviation},
		{name: "client order ID above 100 characters", symbol: futuresTestPair, side: "buy", orderType: "lmt", size: 1, clientOrderID: strings.Repeat("a", 101), err: errFuturesClientOrderIDTooLong},
		{name: "client order ID of 100 multibyte characters", symbol: futuresTestPair, side: "buy", orderType: "lmt", size: 1, clientOrderID: strings.Repeat("é", 100)},
		{name: "stop with a stop price", symbol: futuresTestPair, side: "sell", orderType: "stp", size: 1, stopPrice: 79000},
		{name: "stop limit", symbol: futuresTestPair, side: "sell", orderType: "stp", size: 1, limitPrice: 78900, stopPrice: 79000},
		{name: "trailing stop by quote currency beyond 50", symbol: futuresTestPair, side: "sell", orderType: "trailing_stop", size: 1, deviation: 300, deviationUnit: "QUOTE_CURRENCY"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := futuresValidateNewOrder(tc.symbol, tc.side, tc.orderType, tc.size, tc.limitPrice, tc.stopPrice, tc.deviation, tc.deviationUnit, tc.clientOrderID)
			assert.ErrorIs(t, err, tc.err, "futuresValidateNewOrder should return the expected error")
		})
	}
}

func TestFuturesValidateTrailingStopDeviation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		deviation float64
		unit      string
		err       error
	}{
		{deviation: 0, unit: "PERCENT"},
		{deviation: 0.09, unit: "PERCENT", err: errFuturesInvalidTrailingStopDeviation},
		{deviation: 0.1, unit: "PERCENT"},
		{deviation: 50, unit: "PERCENT"},
		{deviation: 50.01, unit: "PERCENT", err: errFuturesInvalidTrailingStopDeviation},
		{deviation: -1, unit: "PERCENT", err: errFuturesInvalidTrailingStopDeviation},
		{deviation: math.NaN(), unit: "PERCENT", err: errFuturesInvalidTrailingStopDeviation},
		{deviation: math.Inf(1), unit: "PERCENT", err: errFuturesInvalidTrailingStopDeviation},
		{deviation: math.Inf(-1), unit: "PERCENT", err: errFuturesInvalidTrailingStopDeviation},
		{deviation: 0, unit: "QUOTE_CURRENCY"},
		{deviation: 0.05, unit: "QUOTE_CURRENCY"},
		{deviation: 100, unit: "QUOTE_CURRENCY"},
		{deviation: -100, unit: "QUOTE_CURRENCY", err: errFuturesInvalidTrailingStopDeviation},
		{deviation: math.NaN(), unit: "QUOTE_CURRENCY", err: errFuturesInvalidTrailingStopDeviation},
		{deviation: math.Inf(1), unit: "QUOTE_CURRENCY", err: errFuturesInvalidTrailingStopDeviation},
		{deviation: math.Inf(-1), unit: "QUOTE_CURRENCY", err: errFuturesInvalidTrailingStopDeviation},
	} {
		err := futuresValidateTrailingStopDeviation(tc.deviation, tc.unit)
		assert.ErrorIsf(t, err, tc.err, "futuresValidateTrailingStopDeviation should return the expected error for %v %s", tc.deviation, tc.unit)
	}
}

func TestFuturesValidateEdit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                        string
		size, limitPrice, stopPrice float64
		deviation                   float64
		deviationUnit               string
		err                         error
	}{
		{name: "nothing changed"},
		{name: "every value changed", size: 2, limitPrice: 82100, stopPrice: 82000, deviation: 0.5, deviationUnit: "PERCENT"},
		{name: "negative size", size: -1, err: order.ErrAmountIsInvalid},
		{name: "NaN size", size: math.NaN(), err: order.ErrAmountIsInvalid},
		{name: "infinite size", size: math.Inf(1), err: order.ErrAmountIsInvalid},
		{name: "negative infinite size", size: math.Inf(-1), err: order.ErrAmountIsInvalid},
		{name: "NaN limit price", limitPrice: math.NaN(), err: errInvalidOrderPrice},
		{name: "negative stop price", stopPrice: -1, err: errInvalidOrderPrice},
		{name: "deviation out of range", deviation: 51, deviationUnit: "PERCENT", err: errFuturesInvalidTrailingStopDeviation},
	} {
		err := futuresValidateEdit(tc.size, tc.limitPrice, tc.stopPrice, tc.deviation, tc.deviationUnit)
		assert.ErrorIsf(t, err, tc.err, "futuresValidateEdit should return the expected error for %s", tc.name)
	}
}

func TestFuturesValidatePrices(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		limitPrice, stopPrice float64
		err                   error
	}{
		{},
		{limitPrice: 81500, stopPrice: 81600},
		{limitPrice: -1, err: errInvalidOrderPrice},
		{limitPrice: math.NaN(), err: errInvalidOrderPrice},
		{limitPrice: math.Inf(1), err: errInvalidOrderPrice},
		{limitPrice: math.Inf(-1), err: errInvalidOrderPrice},
		{stopPrice: -1, err: errInvalidOrderPrice},
		{stopPrice: math.NaN(), err: errInvalidOrderPrice},
		{stopPrice: math.Inf(1), err: errInvalidOrderPrice},
		{stopPrice: math.Inf(-1), err: errInvalidOrderPrice},
	} {
		err := futuresValidatePrices(tc.limitPrice, tc.stopPrice)
		assert.ErrorIsf(t, err, tc.err, "futuresValidatePrices should return the expected error for limit price %v and stop price %v", tc.limitPrice, tc.stopPrice)
	}
}

func TestFuturesAlgorithmIDHeader(t *testing.T) {
	t.Parallel()
	var (
		mu         sync.Mutex
		algorithms = make(map[string][]string)
	)
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		algorithms[path.Base(r.URL.Path)] = append(algorithms[path.Base(r.URL.Path)], r.Header.Get("algoId"))
		mu.Unlock()
		_, _ = w.Write([]byte(`{"result":"success","serverTime":"2026-10-09T08:15:30.415Z"}`))
	})
	_, err := ex.SendFuturesOrder(t.Context(), &FuturesSendOrderRequest{Symbol: futuresTestPair, Side: "buy", OrderType: "lmt", Size: 0.0005, LimitPrice: 81500, AlgorithmID: "gct-send"})
	require.NoError(t, err, "SendFuturesOrder must not error")
	_, err = ex.EditFuturesOrder(t.Context(), &FuturesEditOrderRequest{OrderID: futuresTradingLimitOrderID, Size: 0.0006, AlgorithmID: "gct-edit"})
	require.NoError(t, err, "EditFuturesOrder must not error")
	_, err = ex.CancelFuturesOrder(t.Context(), &FuturesCancelOrderRequest{OrderID: futuresTradingLimitOrderID, AlgorithmID: "gct-cancel"})
	require.NoError(t, err, "CancelFuturesOrder must not error")
	_, err = ex.CancelAllFuturesOrders(t.Context(), &FuturesCancelAllOrdersRequest{AlgorithmID: "gct-cancel-all"})
	require.NoError(t, err, "CancelAllFuturesOrders must not error")
	_, err = ex.CancelAllFuturesOrders(t.Context(), nil)
	require.NoError(t, err, "CancelAllFuturesOrders must not error without a request")
	_, err = ex.CancelAllFuturesOrdersAfter(t.Context(), &FuturesCancelAllOrdersAfterRequest{Timeout: time.Minute, AlgorithmID: "gct-dead-mans-switch"})
	require.NoError(t, err, "CancelAllFuturesOrdersAfter must not error")
	_, err = ex.FuturesBatchOrder(t.Context(), &FuturesBatchOrderRequest{Instructions: []FuturesBatchInstruction{{Cancel: &FuturesBatchCancelInstruction{OrderID: futuresTradingLimitOrderID}}}, AlgorithmID: "gct-batch"})
	require.NoError(t, err, "FuturesBatchOrder must not error")
	mu.Lock()
	defer mu.Unlock()
	exp := map[string][]string{
		"sendorder":            {"gct-send"},
		"editorder":            {"gct-edit"},
		"cancelorder":          {"gct-cancel"},
		"cancelallorders":      {"gct-cancel-all", ""},
		"cancelallordersafter": {"gct-dead-mans-switch"},
		"batchorder":           {"gct-batch"},
	}
	assert.Equal(t, exp, algorithms, "Each order management endpoint should send its algorithm ID in the algoId header, and none without one")
}

func TestFuturesBatchInstruction(t *testing.T) {
	t.Parallel()
	send := &FuturesBatchSendInstruction{OrderTag: "1", Symbol: futuresTestPair, Side: "buy", OrderType: "lmt", Size: 1, LimitPrice: 82000, ClientOrderID: "my-order", ReduceOnly: true}
	for _, tc := range []struct {
		name        string
		instruction *FuturesBatchInstruction
		exp         futuresBatchInstructionJSON
		err         error
	}{
		{name: "no instruction", instruction: &FuturesBatchInstruction{}, err: errFuturesInvalidBatchInstruction},
		{name: "send and cancel", instruction: &FuturesBatchInstruction{Send: send, Cancel: &FuturesBatchCancelInstruction{OrderID: "1"}}, err: errFuturesInvalidBatchInstruction},
		{name: "send without a tag", instruction: &FuturesBatchInstruction{Send: &FuturesBatchSendInstruction{Symbol: futuresTestPair, Side: "buy", OrderType: "lmt", Size: 1}}, err: errFuturesOrderTagRequired},
		{name: "send without a size", instruction: &FuturesBatchInstruction{Send: &FuturesBatchSendInstruction{OrderTag: "1", Symbol: futuresTestPair, Side: "buy", OrderType: "lmt"}}, err: order.ErrAmountIsInvalid},
		{name: "send with a NaN limit price", instruction: &FuturesBatchInstruction{Send: &FuturesBatchSendInstruction{OrderTag: "1", Symbol: futuresTestPair, Side: "buy", OrderType: "lmt", Size: 1, LimitPrice: math.NaN()}}, err: errInvalidOrderPrice},
		{
			name:        "send",
			instruction: &FuturesBatchInstruction{Send: send},
			exp:         futuresBatchInstructionJSON{Order: "send", OrderTag: "1", OrderType: "lmt", Symbol: "PF_XBTUSD", Side: "buy", Size: 1, LimitPrice: 82000, ClientOrderID: "my-order", ReduceOnly: true},
		},
		{name: "edit without an order ID", instruction: &FuturesBatchInstruction{Edit: &FuturesBatchEditInstruction{Size: 2}}, err: order.ErrOrderIDNotSet},
		{name: "edit with a negative size", instruction: &FuturesBatchInstruction{Edit: &FuturesBatchEditInstruction{OrderID: "1", Size: -2}}, err: order.ErrAmountIsInvalid},
		{name: "edit with an infinite limit price", instruction: &FuturesBatchInstruction{Edit: &FuturesBatchEditInstruction{OrderID: "1", LimitPrice: math.Inf(1)}}, err: errInvalidOrderPrice},
		{
			name:        "edit with a deviation out of range",
			instruction: &FuturesBatchInstruction{Edit: &FuturesBatchEditInstruction{OrderID: "1", TrailingStopMaximumDeviation: 60, TrailingStopDeviationUnit: "PERCENT"}},
			err:         errFuturesInvalidTrailingStopDeviation,
		},
		{
			name:        "edit",
			instruction: &FuturesBatchInstruction{Edit: &FuturesBatchEditInstruction{ClientOrderID: "my-order", Size: 2, LimitPrice: 82100, QuantityMode: "ABSOLUTE"}},
			exp:         futuresBatchInstructionJSON{Order: "edit", ClientOrderID: "my-order", Size: 2, LimitPrice: 82100, QuantityMode: "ABSOLUTE"},
		},
		{name: "cancel without an order ID", instruction: &FuturesBatchInstruction{Cancel: &FuturesBatchCancelInstruction{}}, err: order.ErrOrderIDNotSet},
		{
			name:        "cancel",
			instruction: &FuturesBatchInstruction{Cancel: &FuturesBatchCancelInstruction{OrderID: "e35d61dd-8a30-4d5f-a574-b5593ef0c050"}},
			exp:         futuresBatchInstructionJSON{Order: "cancel", OrderID: "e35d61dd-8a30-4d5f-a574-b5593ef0c050"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.futuresBatchInstruction(tc.instruction)
			require.ErrorIs(t, err, tc.err, "futuresBatchInstruction must return the expected error")
			assert.Equal(t, tc.exp, got, "futuresBatchInstruction should convert the instruction")
		})
	}
	_, err := new(Exchange).futuresBatchInstruction(&FuturesBatchInstruction{Send: send})
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "futuresBatchInstruction should return the error formatting a send's symbol")
}

func TestFuturesSetProcessBefore(t *testing.T) {
	t.Parallel()
	params := url.Values{}
	futuresSetProcessBefore(params, time.Time{})
	assert.Empty(t, params, "futuresSetProcessBefore should set nothing for an unset time")
	futuresSetProcessBefore(params, time.Date(2026, 10, 9, 18, 23, 10, 250e6, time.FixedZone("AEST", 10*60*60)))
	assert.Equal(t, url.Values{"processBefore": {"2026-10-09T08:23:10.250000Z"}}, params, "futuresSetProcessBefore should set the time in UTC to the microsecond")
}

func TestFuturesAlgorithmHeader(t *testing.T) {
	t.Parallel()
	assert.Nil(t, futuresAlgorithmHeader(""), "futuresAlgorithmHeader should return no header without an algorithm ID")
	assert.Equal(t, map[string]string{"algoId": "twap"}, futuresAlgorithmHeader("twap"), "futuresAlgorithmHeader should return the algoId header")
}
