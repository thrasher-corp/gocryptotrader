package bybit

import (
	"context"
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceOrder must reject a nil request")
	_, err = e.PlaceOrder(t.Context(), &PlaceOrderRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "PlaceOrder must reject a request without a category")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *PlaceOrderRequest
		exp  *OrderResponse
	}{
		{
			name: "spot post only",
			req:  &PlaceOrderRequest{Category: cSpot, Symbol: "BTCUSDT", Side: "Buy", OrderType: "Limit", Quantity: 0.1, Price: 15600, TimeInForce: "PostOnly", OrderLinkID: "spot-test-01", OrderFilter: "Order"},
			exp:  &OrderResponse{OrderID: "1321003749386327552", OrderLinkID: "spot-test-01"},
		},
		{
			name: "linear with partial limit take profit and stop loss",
			req:  &PlaceOrderRequest{Category: cLinear, Symbol: "BTCUSDT", Side: "Buy", OrderType: "Limit", Quantity: 1, Price: 25000, TimeInForce: "GTC", OrderLinkID: "usdt-test-01", TakeProfit: 28000, StopLoss: 20000, TPSLMode: "Partial", TPOrderType: "Limit", SLOrderType: "Limit", TPLimitPrice: 27500, SLLimitPrice: 20500},
			exp:  &OrderResponse{OrderID: "9ac8a0c5-5fb4-4b47-a0bb-4a6a7c2d1bb1", OrderLinkID: "usdt-test-01"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.PlaceOrder(t.Context(), tc.req)
			require.NoError(t, err, "PlaceOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "PlaceOrder should decode every field")
				return
			}
			assert.NotEmpty(t, result.OrderID, "PlaceOrder should return the order ID")
		})
	}
}

func TestPlaceOrderRequestValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *PlaceOrderRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no category", req: &PlaceOrderRequest{}, err: errCategoryNotSet},
		{name: "no symbol", req: &PlaceOrderRequest{Category: cSpot}, err: errSymbolMissing},
		{name: "no side", req: &PlaceOrderRequest{Category: cSpot, Symbol: "BTCUSDT"}, err: order.ErrSideIsInvalid},
		{name: "no order type", req: &PlaceOrderRequest{Category: cSpot, Symbol: "BTCUSDT", Side: "Buy"}, err: order.ErrTypeIsInvalid},
		{name: "no quantity", req: &PlaceOrderRequest{Category: cSpot, Symbol: "BTCUSDT", Side: "Buy", OrderType: "Market"}, err: limits.ErrAmountBelowMin},
		{name: "negative quantity", req: &PlaceOrderRequest{Category: cSpot, Symbol: "BTCUSDT", Side: "Buy", OrderType: "Market", Quantity: -1}, err: limits.ErrAmountBelowMin},
		{name: "zero quantity reduce only", req: &PlaceOrderRequest{Category: cLinear, Symbol: "BTCUSDT", Side: "Sell", OrderType: "Market", ReduceOnly: true}, err: limits.ErrAmountBelowMin},
		{name: "zero quantity closes the position", req: &PlaceOrderRequest{Category: cLinear, Symbol: "BTCUSDT", Side: "Sell", OrderType: "Market", ReduceOnly: true, CloseOnTrigger: true}},
		{name: "valid", req: &PlaceOrderRequest{Category: cSpot, Symbol: "BTCUSDT", Side: "Buy", OrderType: "Market", Quantity: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, tc.req.Validate(), tc.err, "Validate should return the expected error")
		})
	}
}

func TestPlaceOrderRequestPayload(t *testing.T) {
	t.Parallel()
	_, err := (*PlaceOrderRequest)(nil).Payload()
	require.ErrorIs(t, err, common.ErrNilPointer, "Payload must reject a nil request")
	_, err = (&PlaceOrderRequest{}).Payload()
	require.ErrorIs(t, err, errCategoryNotSet, "Payload must validate the request")

	r := &PlaceOrderRequest{
		Category: cLinear, Symbol: "BTCUSDT", IsLeverage: true, Side: "Buy", OrderType: "Limit", Quantity: 0.01,
		MarketUnit: "baseCoin", RPITakerAccess: true, SlippageToleranceType: "Percent", SlippageTolerance: 0.5,
		Price: 28000, TriggerDirection: 1, OrderFilter: "Order", TriggerPrice: 27500, TriggerBy: "MarkPrice",
		OrderIV: 0.1, TimeInForce: "GTC", PositionIndex: 1, OrderLinkID: "usdt-test-03", TakeProfit: 35000,
		StopLoss: 27000, TPTriggerBy: "LastPrice", SLTriggerBy: "IndexPrice", ReduceOnly: true, CloseOnTrigger: true,
		SMPType: "CancelMaker", MMP: true, TPSLMode: "Partial", TPLimitPrice: 36000, SLLimitPrice: 26500,
		TPOrderType: "Limit", SLOrderType: "Limit", BBOSideType: "Queue", BBOLevel: 2,
	}
	exp := map[string]any{
		"category": "linear", "symbol": "BTCUSDT", "isLeverage": 1, "side": "Buy", "orderType": "Limit", "qty": "0.01",
		"marketUnit": "baseCoin", "rpiTakerAccess": true, "slippageToleranceType": "Percent", "slippageTolerance": "0.5",
		"price": "28000", "triggerDirection": uint64(1), "orderFilter": "Order", "triggerPrice": "27500",
		"triggerBy": "MarkPrice", "orderIv": "0.1", "timeInForce": "GTC", "positionIdx": uint64(1),
		"orderLinkId": "usdt-test-03", "takeProfit": "35000", "stopLoss": "27000", "tpTriggerBy": "LastPrice",
		"slTriggerBy": "IndexPrice", "reduceOnly": true, "closeOnTrigger": true, "smpType": "CancelMaker", "mmp": true,
		"tpslMode": "Partial", "tpLimitPrice": "36000", "slLimitPrice": "26500", "tpOrderType": "Limit",
		"slOrderType": "Limit", "bboSideType": "Queue", "bboLevel": "2",
	}
	body, err := r.Payload()
	require.NoError(t, err, "Payload must not error")
	assert.Equal(t, exp, body, "Payload should send every documented parameter")
}

func TestAmendOrder(t *testing.T) {
	t.Parallel()
	_, err := e.AmendOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AmendOrder must reject a nil request")
	_, err = e.AmendOrder(t.Context(), &AmendOrderRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "AmendOrder must reject a request without a category")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	cancel := 0.0
	result, err := e.AmendOrder(t.Context(), &AmendOrderRequest{Category: cLinear, Symbol: "ETHPERP", OrderLinkID: "linear-004", TriggerPrice: 1145, Quantity: 0.15, Price: 1050, TakeProfit: &cancel, StopLoss: &cancel})
	require.NoError(t, err, "AmendOrder must not error")
	if mockTests {
		exp := &OrderResponse{OrderID: "c6f055d9-7f21-4079-913d-e6523a9cfffa", OrderLinkID: "linear-004"}
		assert.Equal(t, exp, result, "AmendOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "AmendOrder should return the order ID")
}

func TestAmendOrderRequestValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *AmendOrderRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no category", req: &AmendOrderRequest{}, err: errCategoryNotSet},
		{name: "no symbol", req: &AmendOrderRequest{Category: cSpot}, err: errSymbolMissing},
		{name: "no order ID", req: &AmendOrderRequest{Category: cSpot, Symbol: "BTCUSDT"}, err: errEitherOrderIDOROrderLinkIDRequired},
		{name: "order ID", req: &AmendOrderRequest{Category: cSpot, Symbol: "BTCUSDT", OrderID: "1321003749386327552"}},
		{name: "order link ID", req: &AmendOrderRequest{Category: cSpot, Symbol: "BTCUSDT", OrderLinkID: "spot-test-01"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, tc.req.Validate(), tc.err, "Validate should return the expected error")
		})
	}
}

func TestAmendOrderRequestPayload(t *testing.T) {
	t.Parallel()
	_, err := (*AmendOrderRequest)(nil).Payload()
	require.ErrorIs(t, err, common.ErrNilPointer, "Payload must reject a nil request")
	_, err = (&AmendOrderRequest{}).Payload()
	require.ErrorIs(t, err, errCategoryNotSet, "Payload must validate the request")

	takeProfit, stopLoss := 2600.5, 0.0
	r := &AmendOrderRequest{
		Category: cLinear, Symbol: "ETHUSDT", OrderID: "fd4300ae-7847-404e-b947-b46980a4d140", OrderLinkID: "test-000005",
		OrderIV: 0.3, TriggerPrice: 2500, Quantity: 0.2, Price: 2450, TPSLMode: "Partial", TakeProfit: &takeProfit,
		StopLoss: &stopLoss, TPTriggerBy: "MarkPrice", SLTriggerBy: "IndexPrice", TriggerBy: "LastPrice",
		TPLimitPrice: 2590, SLLimitPrice: 1450,
	}
	exp := map[string]any{
		"category": "linear", "symbol": "ETHUSDT", "orderId": "fd4300ae-7847-404e-b947-b46980a4d140",
		"orderLinkId": "test-000005", "orderIv": "0.3", "triggerPrice": "2500", "qty": "0.2", "price": "2450",
		"tpslMode": "Partial", "takeProfit": "2600.5", "stopLoss": "0", "tpTriggerBy": "MarkPrice",
		"slTriggerBy": "IndexPrice", "triggerBy": "LastPrice", "tpLimitPrice": "2590", "slLimitPrice": "1450",
	}
	body, err := r.Payload()
	require.NoError(t, err, "Payload must not error")
	assert.Equal(t, exp, body, "Payload should send every documented parameter, and a zero stop loss to cancel it")
}

func TestCancelTradeOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelTradeOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelTradeOrder must reject a nil request")
	_, err = e.CancelTradeOrder(t.Context(), &CancelOrderRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "CancelTradeOrder must reject a request without a category")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	ensureTradablePairs(t)
	for _, tc := range []struct {
		name string
		req  *CancelOrderRequest
		exp  *OrderResponse
	}{
		{
			name: "option by order ID",
			req:  &CancelOrderRequest{Category: cOption, Symbol: optionsTradablePair.String(), OrderID: "c6f055d9-7f21-4079-913d-e6523a9cfffa"},
			exp:  &OrderResponse{OrderID: "c6f055d9-7f21-4079-913d-e6523a9cfffa", OrderLinkID: "option-test-001"},
		},
		{
			name: "spot TP/SL order by order link ID",
			req:  &CancelOrderRequest{Category: cSpot, Symbol: "BTCUSDT", OrderLinkID: "spot-test-02", OrderFilter: "tpslOrder"},
			exp:  &OrderResponse{OrderID: "1321003749386327553", OrderLinkID: "spot-test-02"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.CancelTradeOrder(t.Context(), tc.req)
			require.NoError(t, err, "CancelTradeOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "CancelTradeOrder should decode every field")
				return
			}
			assert.NotEmpty(t, result.OrderID, "CancelTradeOrder should return the order ID")
		})
	}
}

func TestCancelOrderRequestValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *CancelOrderRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no category", req: &CancelOrderRequest{}, err: errCategoryNotSet},
		{name: "no symbol", req: &CancelOrderRequest{Category: cSpot}, err: errSymbolMissing},
		{name: "no order ID", req: &CancelOrderRequest{Category: cSpot, Symbol: "BTCUSDT"}, err: errEitherOrderIDOROrderLinkIDRequired},
		{name: "order ID", req: &CancelOrderRequest{Category: cSpot, Symbol: "BTCUSDT", OrderID: "1321003749386327552"}},
		{name: "order link ID", req: &CancelOrderRequest{Category: cSpot, Symbol: "BTCUSDT", OrderLinkID: "spot-test-01"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, tc.req.Validate(), tc.err, "Validate should return the expected error")
		})
	}
}

func TestCancelOrderRequestPayload(t *testing.T) {
	t.Parallel()
	_, err := (*CancelOrderRequest)(nil).Payload()
	require.ErrorIs(t, err, common.ErrNilPointer, "Payload must reject a nil request")
	_, err = (&CancelOrderRequest{}).Payload()
	require.ErrorIs(t, err, errCategoryNotSet, "Payload must validate the request")

	r := &CancelOrderRequest{Category: cSpot, Symbol: "BTCUSDT", OrderID: "1321003749386327553", OrderLinkID: "spot-test-02", OrderFilter: "tpslOrder"}
	exp := map[string]any{"category": "spot", "symbol": "BTCUSDT", "orderId": "1321003749386327553", "orderLinkId": "spot-test-02", "orderFilter": "tpslOrder"}
	body, err := r.Payload()
	require.NoError(t, err, "Payload must not error")
	assert.Equal(t, exp, body, "Payload should send every documented parameter")
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOpenOrders must reject a nil request")
	_, err = e.GetOpenOrders(t.Context(), &OpenOrdersRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetOpenOrders must reject a request without a category")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *OpenOrdersRequest
		// mockOnly marks requests with a placeholder cursor that Bybit could reject live
		mockOnly bool
		exp      *OpenOrdersResponse
	}{
		{
			name: "spot open orders",
			req:  &OpenOrdersRequest{Category: cSpot},
			exp: &OpenOrdersResponse{
				Category:       "spot",
				NextPageCursor: "page_args1321003749386327553",
				List: []OpenOrder{
					{
						OrderID:                    "1321003749386327552",
						OrderLinkID:                "spot-test-01",
						Symbol:                     "BTCUSDT",
						Price:                      15600,
						Quantity:                   0.1,
						Side:                       "Buy",
						IsLeverage:                 "1",
						OrderStatus:                "PartiallyFilled",
						CancelType:                 "UNKNOWN",
						RejectReason:               "EC_NoError",
						AveragePrice:               15600,
						LeavesQuantity:             0.06,
						LeavesValue:                936,
						CumulativeExecutedQuantity: 0.04,
						CumulativeExecutedValue:    624,
						CumulativeExecutedFee:      0.00004,
						TimeInForce:                "GTC",
						OrderType:                  "Limit",
						BasePrice:                  15610.5,
						SMPType:                    "None",
						CreatedTime:                types.Time(time.UnixMilli(1684738540559)),
						UpdatedTime:                types.Time(time.UnixMilli(1684738540561)),
						CumulativeFeeDetail: map[currency.Code]types.Number{
							currency.BTC: 0.00004,
						},
					},
					{
						OrderID:             "1321003749386327553",
						OrderLinkID:         "spot-test-02",
						Symbol:              "BTCUSDT",
						Quantity:            0.05,
						Side:                "Sell",
						IsLeverage:          "0",
						OrderStatus:         "Untriggered",
						CancelType:          "UNKNOWN",
						RejectReason:        "EC_NoError",
						LeavesQuantity:      0.05,
						LeavesValue:         825,
						TimeInForce:         "GTC",
						OrderType:           "Limit",
						StopOrderType:       "OcoOrder",
						TakeProfit:          16400,
						StopLoss:            15000,
						OCOTriggerBy:        "OcoTriggerByUnknown",
						TPLimitPrice:        16500,
						SLLimitPrice:        14900,
						BasePrice:           15610.5,
						SMPType:             "None",
						CreatedTime:         types.Time(time.Unix(1684738600, 0)),
						UpdatedTime:         types.Time(time.Unix(1684738600, 0)),
						CumulativeFeeDetail: map[currency.Code]types.Number{},
					},
				},
			},
		},
		{
			name: "spot closed order by order link ID",
			req:  &OpenOrdersRequest{Category: cSpot, OrderLinkID: "spot-test-04", OpenOnly: 1, OrderFilter: "Order", Limit: 1},
			exp: &OpenOrdersResponse{
				Category: "spot",
				List: []OpenOrder{
					{
						OrderID:                    "1321003749386327560",
						OrderLinkID:                "spot-test-04",
						Symbol:                     "BTCUSDT",
						Quantity:                   0.002,
						Side:                       "Buy",
						IsLeverage:                 "0",
						OrderStatus:                "Filled",
						CancelType:                 "UNKNOWN",
						RejectReason:               "EC_NoError",
						AveragePrice:               15620.5,
						CumulativeExecutedQuantity: 0.002,
						CumulativeExecutedValue:    31.241,
						CumulativeExecutedFee:      0.000002,
						TimeInForce:                "IOC",
						OrderType:                  "Market",
						MarketUnit:                 "baseCoin",
						BasePrice:                  15619,
						SMPType:                    "None",
						CreatedTime:                types.Time(time.Unix(1684738700, 0)),
						UpdatedTime:                types.Time(time.UnixMilli(1684738700003)),
						CumulativeFeeDetail: map[currency.Code]types.Number{
							currency.BTC: 0.000002,
						},
					},
				},
			},
		},
		{
			name:     "linear by symbol, base coin and settle coin",
			req:      &OpenOrdersRequest{Category: cLinear, Symbol: "ETHUSDT", BaseCoin: currency.ETH, SettleCoin: currency.USDT, Limit: 2, Cursor: "page2"},
			mockOnly: true,
			exp: &OpenOrdersResponse{
				Category:       "linear",
				NextPageCursor: "page3",
				List: []OpenOrder{
					{
						OrderID:             "fd4300ae-7847-404e-b947-b46980a4d140",
						OrderLinkID:         "test-000005",
						ParentOrderLinkID:   "test-000004",
						Symbol:              "ETHUSDT",
						Price:               2450,
						Quantity:            0.1,
						Side:                "Sell",
						PositionIndex:       1,
						OrderStatus:         "Untriggered",
						CreateType:          "CreateByPartialTakeProfit",
						CancelType:          "UNKNOWN",
						RejectReason:        "EC_NoError",
						LeavesQuantity:      0.1,
						LeavesValue:         245,
						TimeInForce:         "GTC",
						OrderType:           "Limit",
						StopOrderType:       "PartialTakeProfit",
						TriggerPrice:        2500,
						TPSLMode:            "Partial",
						TriggerDirection:    1,
						TriggerBy:           "LastPrice",
						LastPriceOnCreated:  1600.05,
						ReduceOnly:          true,
						CloseOnTrigger:      true,
						SMPType:             "CancelMaker",
						SMPGroup:            12,
						SMPOrderID:          "5ab0f1d4-b25c-4d2f-a0f7-6c4f0e8d1e2a",
						CreatedTime:         types.Time(time.UnixMilli(1684738540600)),
						UpdatedTime:         types.Time(time.UnixMilli(1684738540600)),
						CumulativeFeeDetail: map[currency.Code]types.Number{},
					},
					{
						OrderID:                    "5bf6aa3e-0c07-4a1f-a3a3-1cb5e3bf8f30",
						OrderLinkID:                "test-000004",
						Symbol:                     "ETHUSDT",
						Price:                      1600,
						Quantity:                   0.2,
						Side:                       "Buy",
						PositionIndex:              1,
						OrderStatus:                "PartiallyFilled",
						CreateType:                 "CreateByUser",
						CancelType:                 "UNKNOWN",
						RejectReason:               "EC_NoError",
						AveragePrice:               1600,
						LeavesQuantity:             0.1,
						LeavesValue:                160,
						CumulativeExecutedQuantity: 0.1,
						CumulativeExecutedValue:    160,
						CumulativeExecutedFee:      0.0352,
						TimeInForce:                "GTC",
						OrderType:                  "Limit",
						StopOrderType:              "UNKNOWN",
						TakeProfit:                 2500,
						StopLoss:                   1500,
						TPSLMode:                   "Partial",
						TPLimitPrice:               2450,
						SLLimitPrice:               1450,
						TPTriggerBy:                "LastPrice",
						SLTriggerBy:                "MarkPrice",
						TriggerBy:                  "UNKNOWN",
						LastPriceOnCreated:         1600.05,
						SMPType:                    "None",
						CreatedTime:                types.Time(time.UnixMilli(1684738540559)),
						UpdatedTime:                types.Time(time.UnixMilli(1684738540600)),
						CumulativeFeeDetail: map[currency.Code]types.Number{
							currency.USDT: 0.0352,
						},
					},
				},
			},
		},
		{
			name: "option by order ID",
			req:  &OpenOrdersRequest{Category: cOption, BaseCoin: currency.BTC, OrderID: "8d3a3cbd-14ac-4f3e-a72b-3a5b1a4d1f9a"},
			exp: &OpenOrdersResponse{
				Category: "option",
				List: []OpenOrder{
					{
						OrderID:                    "8d3a3cbd-14ac-4f3e-a72b-3a5b1a4d1f9a",
						OrderLinkID:                "option-test-001",
						BlockTradeID:               "1a82e5801af74b67b7ad71ba00a7391a",
						Symbol:                     "BTC-26NOV24-92000-C",
						Price:                      3500,
						Quantity:                   0.5,
						Side:                       "Buy",
						OrderStatus:                "PartiallyFilled",
						CreateType:                 "CreateByUser",
						CancelType:                 "UNKNOWN",
						RejectReason:               "EC_NoError",
						AveragePrice:               3500,
						LeavesQuantity:             0.2,
						LeavesValue:                700,
						CumulativeExecutedQuantity: 0.3,
						CumulativeExecutedValue:    1050,
						CumulativeExecutedFee:      0.6,
						TimeInForce:                "GTC",
						OrderType:                  "Limit",
						StopOrderType:              "UNKNOWN",
						OrderIV:                    0.48,
						LastPriceOnCreated:         3480,
						PlaceType:                  "iv",
						SMPType:                    "None",
						CreatedTime:                types.Time(time.Unix(1731900000, 0)),
						UpdatedTime:                types.Time(time.Unix(1731900005, 0)),
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.mockOnly && !mockTests {
				t.Skip("request carries a placeholder cursor")
			}
			result, err := e.GetOpenOrders(t.Context(), tc.req)
			require.NoError(t, err, "GetOpenOrders must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetOpenOrders should decode every field")
				return
			}
			assert.Equal(t, tc.req.Category, result.Category, "GetOpenOrders should return the requested category")
		})
	}
}

func TestCancelAllTradeOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllTradeOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelAllTradeOrders must reject a nil request")
	_, err = e.CancelAllTradeOrders(t.Context(), &CancelAllOrdersRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "CancelAllTradeOrders must reject a request without a category")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *CancelAllOrdersRequest
		exp  *CancelAllOrdersResponse
	}{
		{
			name: "every option order",
			req:  &CancelAllOrdersRequest{Category: cOption},
			exp: &CancelAllOrdersResponse{
				List:    []OrderResponse{{OrderID: "8d3a3cbd-14ac-4f3e-a72b-3a5b1a4d1f9a", OrderLinkID: "option-test-001"}},
				Success: true,
			},
		},
		{
			name: "linear conditional orders",
			req:  &CancelAllOrdersRequest{Category: cLinear, Symbol: "BTCUSDT", BaseCoin: currency.BTC, SettleCoin: currency.USDT, OrderFilter: "StopOrder", StopOrderType: "Stop"},
			exp: &CancelAllOrdersResponse{
				List: []OrderResponse{
					{
						OrderID:     "1616024329462743808",
						OrderLinkID: "1616024329462743809",
					},
					{
						OrderID:     "1616024287544869632",
						OrderLinkID: "1616024287544869633",
					},
				},
				Success: true,
			},
		},
		{
			name: "spot TP/SL orders",
			req:  &CancelAllOrdersRequest{Category: cSpot, OrderFilter: "tpslOrder"},
			exp: &CancelAllOrdersResponse{
				List:    []OrderResponse{{OrderID: "1321003749386327553", OrderLinkID: "spot-test-02"}},
				Success: true,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.CancelAllTradeOrders(t.Context(), tc.req)
			require.NoError(t, err, "CancelAllTradeOrders must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "CancelAllTradeOrders should decode every field")
				return
			}
			assert.True(t, result.Success.Bool(), "CancelAllTradeOrders should succeed")
		})
	}
}

func TestGetTradeOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradeOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTradeOrderHistory must reject a nil request")
	_, err = e.GetTradeOrderHistory(t.Context(), &OrderHistoryRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetTradeOrderHistory must reject a request without a category")
	_, err = e.GetTradeOrderHistory(t.Context(), &OrderHistoryRequest{Category: cSpot, StartTime: time.UnixMilli(1788739200000), EndTime: time.UnixMilli(1788220800000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetTradeOrderHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *OrderHistoryRequest
		// mockOnly marks requests with a placeholder cursor that Bybit could reject live
		mockOnly bool
		exp      *OrderHistoryResponse
	}{
		{
			name: "spot conditional orders in a window",
			req:  &OrderHistoryRequest{Category: cSpot, Symbol: "BTCUSDT", BaseCoin: currency.BTC, OrderFilter: "StopOrder", StartTime: time.UnixMilli(1788220800000), EndTime: time.UnixMilli(1788739200000), Limit: 50},
			exp: &OrderHistoryResponse{
				Category:       "spot",
				NextPageCursor: "page_token1321003749386327571",
				List: []HistoricalOrder{
					{
						OrderID:                    "1321003749386327570",
						OrderLinkID:                "spot-test-05",
						Symbol:                     "BTCUSDT",
						Quantity:                   0.004,
						Side:                       "Buy",
						IsLeverage:                 "1",
						OrderStatus:                "Filled",
						CancelType:                 "UNKNOWN",
						RejectReason:               "EC_NoError",
						AveragePrice:               15640,
						CumulativeExecutedQuantity: 0.004,
						CumulativeExecutedValue:    62.56,
						CumulativeExecutedFee:      0.000004,
						TimeInForce:                "IOC",
						OrderType:                  "Market",
						StopOrderType:              "Stop",
						MarketUnit:                 "baseCoin",
						TriggerPrice:               15630,
						TriggerDirection:           1,
						BasePrice:                  15500,
						SMPType:                    "None",
						CreatedTime:                types.Time(time.Unix(1788300000, 0)),
						UpdatedTime:                types.Time(time.UnixMilli(1788300000002)),
						CumulativeFeeDetail: map[currency.Code]types.Number{
							currency.BTC: 0.000004,
						},
						SlippageToleranceType: "Percent",
						SlippageTolerance:     0.5,
						ExtraFees:             "[{\"feeCoin\":\"USDT\",\"feeType\":\"GST\",\"subFeeType\":\"IND_GST\",\"feeRate\":\"0.0000675\",\"fee\":\"0.0042\"}]",
						RPITakerAccess:        true,
						RPIMatchedQuantity:    0.001,
					},
					{
						OrderID:               "1321003749386327571",
						OrderLinkID:           "spot-test-06",
						Symbol:                "BTCUSDT",
						Price:                 16000,
						Quantity:              0.01,
						Side:                  "Sell",
						IsLeverage:            "0",
						OrderStatus:           "Deactivated",
						CancelType:            "UNKNOWN",
						RejectReason:          "EC_NoError",
						LeavesQuantity:        0.01,
						LeavesValue:           160,
						TimeInForce:           "GTC",
						OrderType:             "Limit",
						StopOrderType:         "Stop",
						TriggerPrice:          15900,
						TriggerDirection:      2,
						BasePrice:             15610.5,
						SMPType:               "None",
						CreatedTime:           types.Time(time.Unix(1788300100, 0)),
						UpdatedTime:           types.Time(time.Unix(1788390100, 0)),
						CumulativeFeeDetail:   map[currency.Code]types.Number{},
						SlippageToleranceType: "UNKNOWN",
					},
				},
			},
		},
		{
			name: "spot OCO order by order link ID",
			req:  &OrderHistoryRequest{Category: cSpot, OrderFilter: "OcoOrder", OrderLinkID: "spot-test-02"},
			exp: &OrderHistoryResponse{
				Category: "spot",
				List: []HistoricalOrder{
					{
						OrderID:                    "1321003749386327553",
						OrderLinkID:                "spot-test-02",
						Symbol:                     "BTCUSDT",
						Quantity:                   0.05,
						Side:                       "Sell",
						IsLeverage:                 "0",
						OrderStatus:                "Filled",
						CancelType:                 "UNKNOWN",
						RejectReason:               "EC_NoError",
						AveragePrice:               16500,
						CumulativeExecutedQuantity: 0.05,
						CumulativeExecutedValue:    825,
						CumulativeExecutedFee:      0.825,
						TimeInForce:                "GTC",
						OrderType:                  "Limit",
						StopOrderType:              "OcoOrder",
						TakeProfit:                 16400,
						StopLoss:                   15000,
						OCOTriggerBy:               "OcoTriggerByTp",
						TPLimitPrice:               16500,
						SLLimitPrice:               14900,
						BasePrice:                  15610.5,
						SMPType:                    "None",
						CreatedTime:                types.Time(time.Unix(1684738600, 0)),
						UpdatedTime:                types.Time(time.Unix(1684740000, 0)),
						CumulativeFeeDetail: map[currency.Code]types.Number{
							currency.USDT: 0.825,
						},
						SlippageToleranceType: "UNKNOWN",
					},
				},
			},
		},
		{
			name:     "linear cancelled orders by settle coin",
			req:      &OrderHistoryRequest{Category: cLinear, SettleCoin: currency.USDT, OrderStatus: "Cancelled", Limit: 2, Cursor: "page2"},
			mockOnly: true,
			exp: &OrderHistoryResponse{
				Category:       "linear",
				NextPageCursor: "page3",
				List: []HistoricalOrder{
					{
						OrderID:                    "5bf6aa3e-0c07-4a1f-a3a3-1cb5e3bf8f30",
						OrderLinkID:                "test-000004",
						Symbol:                     "ETHUSDT",
						Price:                      1600,
						Quantity:                   0.2,
						Side:                       "Buy",
						PositionIndex:              1,
						OrderStatus:                "Cancelled",
						CreateType:                 "CreateByUser",
						CancelType:                 "CancelByUser",
						RejectReason:               "EC_NoError",
						AveragePrice:               1600,
						CumulativeExecutedQuantity: 0.1,
						CumulativeExecutedValue:    160,
						CumulativeExecutedFee:      0.0352,
						TimeInForce:                "GTC",
						OrderType:                  "Limit",
						StopOrderType:              "UNKNOWN",
						TakeProfit:                 2500,
						StopLoss:                   1500,
						TPSLMode:                   "Partial",
						TPLimitPrice:               2450,
						SLLimitPrice:               1450,
						TPTriggerBy:                "LastPrice",
						SLTriggerBy:                "MarkPrice",
						TriggerBy:                  "UNKNOWN",
						LastPriceOnCreated:         1600.05,
						SMPType:                    "None",
						CreatedTime:                types.Time(time.UnixMilli(1684738540559)),
						UpdatedTime:                types.Time(time.Unix(1684738600, 0)),
						CumulativeFeeDetail: map[currency.Code]types.Number{
							currency.USDT: 0.0352,
						},
						SlippageToleranceType: "TickSize",
						SlippageTolerance:     5,
					},
					{
						OrderID:               "fd4300ae-7847-404e-b947-b46980a4d140",
						OrderLinkID:           "test-000005",
						ParentOrderLinkID:     "test-000004",
						Symbol:                "ETHUSDT",
						Price:                 2450,
						Quantity:              0.1,
						Side:                  "Sell",
						PositionIndex:         1,
						OrderStatus:           "Cancelled",
						CreateType:            "CreateByPartialTakeProfit",
						CancelType:            "CancelByTpSlTsClear",
						RejectReason:          "EC_NoError",
						TimeInForce:           "GTC",
						OrderType:             "Limit",
						StopOrderType:         "PartialTakeProfit",
						TriggerPrice:          2500,
						TPSLMode:              "Partial",
						TriggerDirection:      1,
						TriggerBy:             "LastPrice",
						LastPriceOnCreated:    1600.05,
						ReduceOnly:            true,
						CloseOnTrigger:        true,
						SMPType:               "CancelMaker",
						SMPGroup:              12,
						SMPOrderID:            "5ab0f1d4-b25c-4d2f-a0f7-6c4f0e8d1e2a",
						CreatedTime:           types.Time(time.UnixMilli(1684738540600)),
						UpdatedTime:           types.Time(time.UnixMilli(1684738600001)),
						CumulativeFeeDetail:   map[currency.Code]types.Number{},
						SlippageToleranceType: "UNKNOWN",
					},
				},
			},
		},
		{
			name: "option order by order ID",
			req:  &OrderHistoryRequest{Category: cOption, BaseCoin: currency.BTC, OrderID: "8d3a3cbd-14ac-4f3e-a72b-3a5b1a4d1f9a", OrderLinkID: "option-test-001", OrderFilter: "Order"},
			exp: &OrderHistoryResponse{
				Category: "option",
				List: []HistoricalOrder{
					{
						OrderID:                    "8d3a3cbd-14ac-4f3e-a72b-3a5b1a4d1f9a",
						OrderLinkID:                "option-test-001",
						BlockTradeID:               "1a82e5801af74b67b7ad71ba00a7391a",
						Symbol:                     "BTC-26NOV24-92000-C",
						Price:                      3500,
						Quantity:                   0.5,
						Side:                       "Buy",
						OrderStatus:                "PartiallyFilled",
						CreateType:                 "CreateByUser",
						CancelType:                 "UNKNOWN",
						RejectReason:               "EC_NoError",
						AveragePrice:               3500,
						LeavesQuantity:             0.2,
						LeavesValue:                700,
						CumulativeExecutedQuantity: 0.3,
						CumulativeExecutedValue:    1050,
						CumulativeExecutedFee:      0.6,
						TimeInForce:                "GTC",
						OrderType:                  "Limit",
						StopOrderType:              "UNKNOWN",
						OrderIV:                    0.48,
						LastPriceOnCreated:         3480,
						PlaceType:                  "iv",
						SMPType:                    "None",
						CreatedTime:                types.Time(time.Unix(1731900000, 0)),
						UpdatedTime:                types.Time(time.Unix(1731900005, 0)),
						SlippageToleranceType:      "UNKNOWN",
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.mockOnly && !mockTests {
				t.Skip("request carries a placeholder cursor")
			}
			result, err := e.GetTradeOrderHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetTradeOrderHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetTradeOrderHistory should decode every field")
				return
			}
			assert.Equal(t, tc.req.Category, result.Category, "GetTradeOrderHistory should return the requested category")
		})
	}
}

func TestGetExecution(t *testing.T) {
	t.Parallel()
	_, err := e.GetExecution(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetExecution must reject a nil request")
	_, err = e.GetExecution(t.Context(), &ExecutionRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetExecution must reject a request without a category")
	_, err = e.GetExecution(t.Context(), &ExecutionRequest{Category: cSpot, StartTime: time.UnixMilli(1788307200000), EndTime: time.UnixMilli(1788220800000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetExecution must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *ExecutionRequest
		// mockOnly marks requests with a placeholder cursor that Bybit could reject live
		mockOnly bool
		exp      *ExecutionResponse
	}{
		{
			name: "spot trades",
			req:  &ExecutionRequest{Category: cSpot, ExecutionType: "Trade"},
			exp: &ExecutionResponse{
				Category:       "spot",
				NextPageCursor: "132766%3A2%2C132766%3A2",
				List: []Execution{
					{
						Symbol:            "BTCUSDT",
						OrderID:           "1321003749386327552",
						OrderLinkID:       "spot-test-01",
						Side:              "Buy",
						OrderQuantity:     0.1,
						OrderType:         "Market",
						ExecutionFee:      0.0001,
						ExecutionID:       "2100000000041295316",
						ExecutionPrice:    15600.25,
						ExecutionQuantity: 0.1,
						ExecutionType:     "Trade",
						ExecutionValue:    1560.025,
						ExecutionTime:     types.Time(time.UnixMilli(1684738540560)),
						FeeCurrency:       currency.BTC,
						FeeRate:           0.001,
						Sequence:          4688002127,
						ExtraFees:         "[{\"feeCoin\":\"USDT\",\"feeType\":\"GST\",\"subFeeType\":\"IND_GST\",\"feeRate\":\"0.0000675\",\"fee\":\"0.1053\"}]",
					},
				},
			},
		},
		{
			name:     "linear fills of an order in a window",
			req:      &ExecutionRequest{Category: cLinear, Symbol: "ETHUSDT", OrderLinkID: "test-000004", BaseCoin: currency.ETH, SettleCoin: currency.USDT, StartTime: time.UnixMilli(1788220800000), EndTime: time.UnixMilli(1788307200000), Limit: 1, Cursor: "page2"},
			mockOnly: true,
			exp: &ExecutionResponse{
				Category:       "linear",
				NextPageCursor: "page3",
				List: []Execution{
					{
						Symbol:            "ETHUSDT",
						OrderID:           "5bf6aa3e-0c07-4a1f-a3a3-1cb5e3bf8f30",
						OrderLinkID:       "test-000004",
						Side:              "Sell",
						OrderPrice:        1236.9,
						OrderQuantity:     0.1,
						CreateType:        "CreateByUser",
						OrderType:         "Market",
						StopOrderType:     "UNKNOWN",
						ExecutionFee:      0.071409,
						ExecutionFeeV2:    0.0185,
						ExecutionID:       "e0cbe81d-0f18-5866-9415-cf319b5dab3b",
						ExecutionPrice:    1190.15,
						ExecutionQuantity: 0.1,
						ExecutionType:     "FutureSpread",
						ExecutionValue:    119.015,
						ExecutionTime:     types.Time(time.UnixMilli(1788282722429)),
						FeeCurrency:       currency.USDT,
						IsMaker:           true,
						FeeRate:           0.0006,
						MarkPrice:         1183.54,
						ClosedSize:        0.1,
						Sequence:          4688002128,
					},
				},
			},
		},
		{
			name: "option fills of an order",
			req:  &ExecutionRequest{Category: cOption, OrderID: "8d3a3cbd-14ac-4f3e-a72b-3a5b1a4d1f9a"},
			exp: &ExecutionResponse{
				Category: "option",
				List: []Execution{
					{
						Symbol:            "BTC-26NOV24-92000-C",
						OrderID:           "8d3a3cbd-14ac-4f3e-a72b-3a5b1a4d1f9a",
						OrderLinkID:       "option-test-001",
						Side:              "Buy",
						OrderPrice:        3500,
						OrderQuantity:     0.5,
						LeavesQuantity:    0.2,
						CreateType:        "CreateByUser",
						OrderType:         "Limit",
						StopOrderType:     "UNKNOWN",
						ExecutionFee:      0.6,
						ExecutionID:       "7e3f2a01-3c55-58b2-9f8a-4bd2b6d90a17",
						ExecutionPrice:    3500,
						ExecutionQuantity: 0.3,
						ExecutionType:     "BlockTrade",
						ExecutionValue:    1050,
						ExecutionTime:     types.Time(time.Unix(1731900003, 0)),
						FeeCurrency:       currency.USDT,
						FeeRate:           0.0003,
						TradeIV:           0.4812,
						MarkIV:            0.4798,
						MarkPrice:         3495.2,
						IndexPrice:        91840.55,
						UnderlyingPrice:   91850.12,
						BlockTradeID:      "1a82e5801af74b67b7ad71ba00a7391a",
						Sequence:          4688002129,
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.mockOnly && !mockTests {
				t.Skip("request carries a placeholder cursor")
			}
			result, err := e.GetExecution(t.Context(), tc.req)
			require.NoError(t, err, "GetExecution must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetExecution should decode every field")
				return
			}
			assert.Equal(t, tc.req.Category, result.Category, "GetExecution should return the requested category")
		})
	}
}

// testRateLimitedInstance returns an exchange that throttles with rate limits of its own, so a test can make a request
// wait on a budget no other test draws on
func testRateLimitedInstance(tb testing.TB) *Exchange {
	tb.Helper()
	ex := new(Exchange)
	var err error
	ex.Requester, err = request.New(tb.Name(), &http.Client{}, request.WithLimiter(rateLimitDefinitions()))
	require.NoError(tb, err, "request.New must not error")
	return ex
}

func TestPlaceBatchOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceBatchOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceBatchOrder must reject a nil request")
	_, err = e.PlaceBatchOrder(t.Context(), &BatchPlaceOrderRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "PlaceBatchOrder must reject a request without a category")
	synctest.Test(t, func(t *testing.T) {
		// The bubble's clock stands still, so the third order has to wait for the budget the second took
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		result, err := testRateLimitedInstance(t).PlaceBatchOrder(ctx, &BatchPlaceOrderRequest{Category: cSpot, Request: []BatchPlaceOrderItem{
			{Symbol: "BTCUSDT", Side: "Buy", OrderType: "Limit", Quantity: 0.01, Price: 60000},
			{Symbol: "ETHUSDT", Side: "Buy", OrderType: "Limit", Quantity: 0.1, Price: 2500},
			{Symbol: "SOLUSDT", Side: "Buy", OrderType: "Limit", Quantity: 1, Price: 150},
		}})
		assert.ErrorIs(t, err, context.Canceled, "PlaceBatchOrder should give up a batch cancelled while its orders wait for their rate limit")
		assert.Nil(t, result, "PlaceBatchOrder should return no result for a batch it did not send")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	if mockTests {
		_, err = e.PlaceBatchOrder(t.Context(), &BatchPlaceOrderRequest{Category: cSpot, Request: []BatchPlaceOrderItem{
			{Symbol: "BTCUSDT", IsLeverage: true, Side: "Buy", OrderType: "Limit", Quantity: 0.05, Price: 30000, TimeInForce: "GTC", OrderLinkID: "spot-btc-03"},
		}})
		require.ErrorIs(t, err, errBatchOutcomeMismatch, "PlaceBatchOrder must reject a reply whose outcomes do not match its orders")
		_, err = e.PlaceBatchOrder(t.Context(), &BatchPlaceOrderRequest{Category: cLinear, Request: []BatchPlaceOrderItem{
			{Symbol: "BTCUSDT", Side: "Buy", OrderType: "Limit", Quantity: 0.01, Price: 60000, OrderLinkID: "linear-btc-04"},
		}})
		require.ErrorIs(t, err, errAPIResponse, "PlaceBatchOrder must return an error reply")
	}
	ensureTradablePairs(t)
	result, err := e.PlaceBatchOrder(t.Context(), &BatchPlaceOrderRequest{Category: cOption, Request: []BatchPlaceOrderItem{
		{Symbol: optionsTradablePair.String(), OrderType: "Limit", Side: "Buy", Quantity: 1, OrderIV: 6, TimeInForce: "GTC", OrderLinkID: "option-test-001"},
		{Symbol: optionsTradablePair.String(), OrderType: "Limit", Side: "Sell", Quantity: 2, Price: 700, TimeInForce: "GTC", OrderLinkID: "option-test-002"},
	}})
	require.NoError(t, err, "PlaceBatchOrder must not error")
	if mockTests {
		exp := &BatchPlaceOrderResponse{
			List: []BatchPlacedOrder{
				{
					Category:    "option",
					Symbol:      "BTC-26NOV24-92000-C",
					OrderID:     "1666800494330512128",
					OrderLinkID: "option-test-001",
					CreateAt:    types.Time(time.UnixMilli(1713434102752)),
					Message:     "OK",
				},
				{
					Code:    110007,
					Message: "Insufficient available balance",
				},
			},
		}
		assert.Equal(t, exp, result, "PlaceBatchOrder should decode every field and each order's outcome")
		return
	}
	assert.Len(t, result.List, 2, "PlaceBatchOrder should return an outcome for each order")
}

func TestBatchPlaceOrderRequestPayload(t *testing.T) {
	t.Parallel()
	_, err := (*BatchPlaceOrderRequest)(nil).Payload()
	require.ErrorIs(t, err, common.ErrNilPointer, "Payload must reject a nil request")
	_, err = (&BatchPlaceOrderRequest{}).Payload()
	require.ErrorIs(t, err, errCategoryNotSet, "Payload must reject a request without a category")
	_, err = (&BatchPlaceOrderRequest{Category: cLinear}).Payload()
	require.ErrorIs(t, err, errNoOrderPassed, "Payload must reject a request without orders")
	_, err = (&BatchPlaceOrderRequest{Category: cLinear, Request: []BatchPlaceOrderItem{{}}}).Payload()
	require.ErrorIs(t, err, errSymbolMissing, "Payload must validate each order")

	r := &BatchPlaceOrderRequest{Category: cLinear, Request: []BatchPlaceOrderItem{
		{Symbol: "BTCUSDT", Side: "Buy", OrderType: "Market", Quantity: 0.01},
		{Symbol: "ETHUSDT", Side: "Sell", OrderType: "Limit", Quantity: 0.1, Price: 2500},
	}}
	exp := map[string]any{"category": "linear", "request": []map[string]any{
		{"symbol": "BTCUSDT", "side": "Buy", "orderType": "Market", "qty": "0.01"},
		{"symbol": "ETHUSDT", "side": "Sell", "orderType": "Limit", "qty": "0.1", "price": "2500"},
	}}
	body, err := r.Payload()
	require.NoError(t, err, "Payload must not error")
	assert.Equal(t, exp, body, "Payload should send the category and every order")
}

func TestBatchPlaceOrderItemPayload(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		item *BatchPlaceOrderItem
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no symbol", item: &BatchPlaceOrderItem{}, err: errSymbolMissing},
		{name: "no side", item: &BatchPlaceOrderItem{Symbol: "BTCUSDT"}, err: order.ErrSideIsInvalid},
		{name: "no order type", item: &BatchPlaceOrderItem{Symbol: "BTCUSDT", Side: "Buy"}, err: order.ErrTypeIsInvalid},
		{name: "no quantity", item: &BatchPlaceOrderItem{Symbol: "BTCUSDT", Side: "Buy", OrderType: "Limit"}, err: limits.ErrAmountBelowMin},
		{name: "negative quantity", item: &BatchPlaceOrderItem{Symbol: "BTCUSDT", Side: "Buy", OrderType: "Limit", Quantity: -1}, err: limits.ErrAmountBelowMin},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := tc.item.Payload()
			assert.ErrorIs(t, err, tc.err, "Payload should return the expected error")
		})
	}

	item := &BatchPlaceOrderItem{
		Symbol: "BTCUSDT", IsLeverage: true, Side: "Sell", OrderType: "Market", ReduceOnly: true, CloseOnTrigger: true,
		MarketUnit: "baseCoin", Price: 28000, TriggerDirection: 2, OrderFilter: "StopOrder", TriggerPrice: 27900,
		TriggerBy: "LastPrice", OrderIV: 0.25, TimeInForce: "IOC", PositionIndex: 2, OrderLinkID: "batch-close-01",
		TakeProfit: 25000, StopLoss: 29000, TPTriggerBy: "MarkPrice", SLTriggerBy: "IndexPrice", SMPType: "CancelTaker",
		MMP: true, TPSLMode: "Partial", TPLimitPrice: 25100, SLLimitPrice: 29100, TPOrderType: "Limit",
		SLOrderType: "Limit",
	}
	exp := map[string]any{
		"symbol": "BTCUSDT", "isLeverage": 1, "side": "Sell", "orderType": "Market", "qty": "0", "reduceOnly": true,
		"closeOnTrigger": true, "marketUnit": "baseCoin", "price": "28000", "triggerDirection": uint64(2),
		"orderFilter": "StopOrder", "triggerPrice": "27900", "triggerBy": "LastPrice", "orderIv": "0.25",
		"timeInForce": "IOC", "positionIdx": uint64(2), "orderLinkId": "batch-close-01", "takeProfit": "25000",
		"stopLoss": "29000", "tpTriggerBy": "MarkPrice", "slTriggerBy": "IndexPrice", "smpType": "CancelTaker",
		"mmp": true, "tpslMode": "Partial", "tpLimitPrice": "25100", "slLimitPrice": "29100", "tpOrderType": "Limit",
		"slOrderType": "Limit",
	}
	body, err := item.Payload()
	require.NoError(t, err, "Payload must not error")
	assert.Equal(t, exp, body, "Payload should send every documented parameter, and a zero quantity to close the position")
}

func TestBatchAmendOrder(t *testing.T) {
	t.Parallel()
	_, err := e.BatchAmendOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "BatchAmendOrder must reject a nil request")
	_, err = e.BatchAmendOrder(t.Context(), &BatchAmendOrderRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "BatchAmendOrder must reject a request without a category")
	synctest.Test(t, func(t *testing.T) {
		// The bubble's clock stands still, so the third order has to wait for the budget the second took
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		result, err := testRateLimitedInstance(t).BatchAmendOrder(ctx, &BatchAmendOrderRequest{Category: cLinear, Request: []BatchAmendOrderItem{
			{Symbol: "BTCUSDT", OrderID: "c0ffee00-0000-4000-8000-000000000011", Price: 61000},
			{Symbol: "ETHUSDT", OrderID: "c0ffee00-0000-4000-8000-000000000012", Price: 2600},
			{Symbol: "SOLUSDT", OrderID: "c0ffee00-0000-4000-8000-000000000013", Price: 160},
		}})
		assert.ErrorIs(t, err, context.Canceled, "BatchAmendOrder should give up a batch cancelled while its orders wait for their rate limit")
		assert.Nil(t, result, "BatchAmendOrder should return no result for a batch it did not send")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	if mockTests {
		for _, tc := range []struct {
			name    string
			orderID string
			err     error
		}{
			{name: "API error", orderID: "c0ffee00-0000-4000-8000-000000000001", err: order.ErrOrderNotFound},
			{name: "null result", orderID: "c0ffee00-0000-4000-8000-000000000002", err: common.ErrNoResponse},
		} {
			_, err = e.BatchAmendOrder(t.Context(), &BatchAmendOrderRequest{Category: cLinear, Request: []BatchAmendOrderItem{{Symbol: "BTCUSDT", OrderID: tc.orderID, Price: 61000}}})
			require.ErrorIsf(t, err, tc.err, "BatchAmendOrder must return the expected error for a reply with an %s", tc.name)
		}
		_, err = e.BatchAmendOrder(t.Context(), &BatchAmendOrderRequest{Category: cLinear, Request: []BatchAmendOrderItem{{Symbol: "BTCUSDT", OrderID: "c0ffee00-0000-4000-8000-000000000003", Price: 61000}}})
		require.Error(t, err, "BatchAmendOrder must reject a reply whose retExtInfo is not an object, since its outcomes are unknown")
	}
	result, err := e.BatchAmendOrder(t.Context(), &BatchAmendOrderRequest{Category: cOption, Request: []BatchAmendOrderItem{
		{Symbol: "ETH-30DEC22-500-C", OrderIV: 6.8, OrderID: "b551f227-7059-4fb5-a6a6-699c04dbd2f2"},
		{Symbol: "ETH-30DEC22-700-C", Price: 650, OrderID: "fa6a595f-1a57-483f-b9d3-30e9c8235a52"},
	}})
	require.NoError(t, err, "BatchAmendOrder must not error")
	if mockTests {
		exp := &BatchOrderResponse{
			List: []BatchOrderResult{
				{
					Category:    "option",
					Symbol:      "ETH-30DEC22-500-C",
					OrderID:     "b551f227-7059-4fb5-a6a6-699c04dbd2f2",
					OrderLinkID: "option-amend-001",
					Message:     "OK",
				},
				{
					Category: "option",
					Symbol:   "ETH-30DEC22-700-C",
					OrderID:  "fa6a595f-1a57-483f-b9d3-30e9c8235a52",
					Code:     110001,
					Message:  "order not exists or too late to replace",
				},
			},
		}
		assert.Equal(t, exp, result, "BatchAmendOrder should decode every field and each order's outcome")
		return
	}
	assert.Len(t, result.List, 2, "BatchAmendOrder should return an outcome for each order")
}

func TestBatchAmendOrderRequestPayload(t *testing.T) {
	t.Parallel()
	_, err := (*BatchAmendOrderRequest)(nil).Payload()
	require.ErrorIs(t, err, common.ErrNilPointer, "Payload must reject a nil request")
	_, err = (&BatchAmendOrderRequest{}).Payload()
	require.ErrorIs(t, err, errCategoryNotSet, "Payload must reject a request without a category")
	_, err = (&BatchAmendOrderRequest{Category: cOption}).Payload()
	require.ErrorIs(t, err, errNoOrderPassed, "Payload must reject a request without orders")
	_, err = (&BatchAmendOrderRequest{Category: cOption, Request: []BatchAmendOrderItem{{}}}).Payload()
	require.ErrorIs(t, err, errSymbolMissing, "Payload must validate each order")

	r := &BatchAmendOrderRequest{Category: cOption, Request: []BatchAmendOrderItem{
		{Symbol: "ETH-30DEC22-500-C", OrderIV: 6.8, OrderID: "b551f227-7059-4fb5-a6a6-699c04dbd2f2"},
	}}
	exp := map[string]any{"category": "option", "request": []map[string]any{
		{"symbol": "ETH-30DEC22-500-C", "orderIv": "6.8", "orderId": "b551f227-7059-4fb5-a6a6-699c04dbd2f2"},
	}}
	body, err := r.Payload()
	require.NoError(t, err, "Payload must not error")
	assert.Equal(t, exp, body, "Payload should send the category and every order")
}

func TestBatchAmendOrderItemPayload(t *testing.T) {
	t.Parallel()
	_, err := (*BatchAmendOrderItem)(nil).Payload()
	require.ErrorIs(t, err, common.ErrNilPointer, "Payload must reject a nil order")
	_, err = (&BatchAmendOrderItem{}).Payload()
	require.ErrorIs(t, err, errSymbolMissing, "Payload must reject an order without a symbol")
	_, err = (&BatchAmendOrderItem{Symbol: "BTCUSDT"}).Payload()
	require.ErrorIs(t, err, errEitherOrderIDOROrderLinkIDRequired, "Payload must reject an order without an ID")

	takeProfit, stopLoss := 0.0, 1450.0
	item := &BatchAmendOrderItem{
		Symbol: "ETHUSDT", OrderID: "5bf6aa3e-0c07-4a1f-a3a3-1cb5e3bf8f30", OrderLinkID: "test-000004", OrderIV: 0.4,
		TriggerPrice: 1590, Quantity: 0.3, Price: 1595, TPSLMode: "Full", TakeProfit: &takeProfit, StopLoss: &stopLoss,
		TPTriggerBy: "LastPrice", SLTriggerBy: "MarkPrice", TriggerBy: "IndexPrice", TPLimitPrice: 2440,
		SLLimitPrice: 1440,
	}
	exp := map[string]any{
		"symbol": "ETHUSDT", "orderId": "5bf6aa3e-0c07-4a1f-a3a3-1cb5e3bf8f30", "orderLinkId": "test-000004",
		"orderIv": "0.4", "triggerPrice": "1590", "qty": "0.3", "price": "1595", "tpslMode": "Full", "takeProfit": "0",
		"stopLoss": "1450", "tpTriggerBy": "LastPrice", "slTriggerBy": "MarkPrice", "triggerBy": "IndexPrice",
		"tpLimitPrice": "2440", "slLimitPrice": "1440",
	}
	body, err := item.Payload()
	require.NoError(t, err, "Payload must not error")
	assert.Equal(t, exp, body, "Payload should send every documented parameter, and a zero take profit to cancel it")
}

func TestCancelBatchOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelBatchOrder must reject a nil request")
	_, err = e.CancelBatchOrder(t.Context(), &BatchCancelOrderRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "CancelBatchOrder must reject a request without a category")
	synctest.Test(t, func(t *testing.T) {
		// The bubble's clock stands still, so the third order has to wait for the budget the second took
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		result, err := testRateLimitedInstance(t).CancelBatchOrder(ctx, &BatchCancelOrderRequest{Category: cSpot, Request: []BatchCancelOrderItem{
			{Symbol: "BTCUSDT", OrderID: "1666800494330512128"},
			{Symbol: "ETHUSDT", OrderID: "1666800494330512130"},
			{Symbol: "SOLUSDT", OrderID: "1666800494330512131"},
		}})
		assert.ErrorIs(t, err, context.Canceled, "CancelBatchOrder should give up a batch cancelled while its orders wait for their rate limit")
		assert.Nil(t, result, "CancelBatchOrder should return no result for a batch it did not send")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	if mockTests {
		_, err = e.CancelBatchOrder(t.Context(), &BatchCancelOrderRequest{Category: cOption, Request: []BatchCancelOrderItem{
			{Symbol: "BTC-26NOV24-92000-C", OrderID: "b551f227-7059-4fb5-a6a6-699c04dbd2f2"},
		}})
		require.ErrorIs(t, err, errBatchOutcomeMismatch, "CancelBatchOrder must reject a reply whose outcomes do not match its orders")
		_, err = e.CancelBatchOrder(t.Context(), &BatchCancelOrderRequest{Category: cLinear, Request: []BatchCancelOrderItem{
			{Symbol: "BTCUSDT", OrderLinkID: "linear-btc-04"},
		}})
		require.ErrorIs(t, err, errAPIResponse, "CancelBatchOrder must return an error reply")
	}
	result, err := e.CancelBatchOrder(t.Context(), &BatchCancelOrderRequest{Category: cSpot, Request: []BatchCancelOrderItem{
		{Symbol: "BTCUSDT", OrderID: "1666800494330512128"},
		{Symbol: "ATOMUSDT", OrderLinkID: "1666800494330512129"},
	}})
	require.NoError(t, err, "CancelBatchOrder must not error")
	if mockTests {
		exp := &BatchOrderResponse{
			List: []BatchOrderResult{
				{
					Category:    "spot",
					Symbol:      "BTCUSDT",
					OrderID:     "1666800494330512128",
					OrderLinkID: "spot-btc-03",
					Message:     "OK",
				},
				{
					Category:    "spot",
					Symbol:      "ATOMUSDT",
					OrderLinkID: "1666800494330512129",
					Code:        170213,
					Message:     "Order does not exist.",
				},
			},
		}
		assert.Equal(t, exp, result, "CancelBatchOrder should decode every field and each order's outcome")
		return
	}
	assert.Len(t, result.List, 2, "CancelBatchOrder should return an outcome for each order")
}

func TestBatchCancelOrderRequestPayload(t *testing.T) {
	t.Parallel()
	_, err := (*BatchCancelOrderRequest)(nil).Payload()
	require.ErrorIs(t, err, common.ErrNilPointer, "Payload must reject a nil request")
	_, err = (&BatchCancelOrderRequest{}).Payload()
	require.ErrorIs(t, err, errCategoryNotSet, "Payload must reject a request without a category")
	_, err = (&BatchCancelOrderRequest{Category: cSpot}).Payload()
	require.ErrorIs(t, err, errNoOrderPassed, "Payload must reject a request without orders")
	_, err = (&BatchCancelOrderRequest{Category: cSpot, Request: []BatchCancelOrderItem{{}}}).Payload()
	require.ErrorIs(t, err, errSymbolMissing, "Payload must validate each order")

	r := &BatchCancelOrderRequest{Category: cSpot, Request: []BatchCancelOrderItem{
		{Symbol: "BTCUSDT", OrderID: "1666800494330512128"},
		{Symbol: "ATOMUSDT", OrderLinkID: "1666800494330512129"},
	}}
	exp := map[string]any{"category": "spot", "request": []map[string]any{
		{"symbol": "BTCUSDT", "orderId": "1666800494330512128"},
		{"symbol": "ATOMUSDT", "orderLinkId": "1666800494330512129"},
	}}
	body, err := r.Payload()
	require.NoError(t, err, "Payload must not error")
	assert.Equal(t, exp, body, "Payload should send the category and every order")
}

func TestBatchCancelOrderItemPayload(t *testing.T) {
	t.Parallel()
	_, err := (*BatchCancelOrderItem)(nil).Payload()
	require.ErrorIs(t, err, common.ErrNilPointer, "Payload must reject a nil order")
	_, err = (&BatchCancelOrderItem{}).Payload()
	require.ErrorIs(t, err, errSymbolMissing, "Payload must reject an order without a symbol")
	_, err = (&BatchCancelOrderItem{Symbol: "BTCUSDT"}).Payload()
	require.ErrorIs(t, err, errEitherOrderIDOROrderLinkIDRequired, "Payload must reject an order without an ID")

	body, err := (&BatchCancelOrderItem{Symbol: "BTCUSDT", OrderID: "1666800494330512128", OrderLinkID: "spot-btc-03"}).Payload()
	require.NoError(t, err, "Payload must not error")
	exp := map[string]any{"symbol": "BTCUSDT", "orderId": "1666800494330512128", "orderLinkId": "spot-btc-03"}
	assert.Equal(t, exp, body, "Payload should send every documented parameter")
}

func TestSendBatchOrderRequest(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("sendBatchOrderRequest's failure cases need crafted replies, and its success case changes orders")
	}
	var result BatchOrderResponse
	outcomes, err := e.sendBatchOrderRequest(t.Context(), "/v5/order/amend-batch", map[string]any{"category": cOption, "request": []map[string]any{
		{"symbol": "ETH-30DEC22-500-C", "orderIv": "6.8", "orderId": "b551f227-7059-4fb5-a6a6-699c04dbd2f2"},
		{"symbol": "ETH-30DEC22-700-C", "price": "650", "orderId": "fa6a595f-1a57-483f-b9d3-30e9c8235a52"},
	}}, amendOptionBatchOrderEPL, &result)
	require.NoError(t, err, "sendBatchOrderRequest must not error")
	expResult := BatchOrderResponse{List: []BatchOrderResult{
		{Category: "option", Symbol: "ETH-30DEC22-500-C", OrderID: "b551f227-7059-4fb5-a6a6-699c04dbd2f2", OrderLinkID: "option-amend-001"},
		{Category: "option", Symbol: "ETH-30DEC22-700-C", OrderID: "fa6a595f-1a57-483f-b9d3-30e9c8235a52"},
	}}
	assert.Equal(t, expResult, result, "sendBatchOrderRequest should decode the result into the given value")
	expOutcomes := []BatchOrderOutcome{{Message: "OK"}, {Code: 110001, Message: "order not exists or too late to replace"}}
	assert.Equal(t, expOutcomes, outcomes, "sendBatchOrderRequest should return each order's outcome from retExtInfo in request order")

	for _, tc := range []struct {
		name    string
		orderID string
		err     error
		errText string
	}{
		{name: "error reply", orderID: "c0ffee00-0000-4000-8000-000000000001", err: order.ErrOrderNotFound},
		{name: "null result", orderID: "c0ffee00-0000-4000-8000-000000000002", err: common.ErrNoResponse},
		{name: "retExtInfo that is not an object", orderID: "c0ffee00-0000-4000-8000-000000000003", errText: "decoding batch order outcomes"},
		{name: "result whose list is not an array", orderID: "c0ffee00-0000-4000-8000-000000000004", errText: "decoding batch order result"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var result BatchOrderResponse
			outcomes, err := e.sendBatchOrderRequest(t.Context(), "/v5/order/amend-batch", map[string]any{"category": cLinear, "request": []map[string]any{
				{"symbol": "BTCUSDT", "orderId": tc.orderID, "price": "61000"},
			}}, amendBatchOrderEPL, &result)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "sendBatchOrderRequest should return the expected error")
			} else {
				assert.ErrorContains(t, err, tc.errText, "sendBatchOrderRequest should say which part of the reply did not decode")
			}
			assert.Nil(t, outcomes, "sendBatchOrderRequest should return no outcomes for a failed request")
		})
	}
}

func TestApplyBatchOrderOutcomes(t *testing.T) {
	t.Parallel()
	orders := func() *BatchOrderResponse {
		return &BatchOrderResponse{List: []BatchOrderResult{
			{Category: "spot", Symbol: "BTCUSDT", OrderID: "1666800494330512128", OrderLinkID: "spot-btc-03"},
			{Category: "spot", Symbol: "ATOMUSDT", OrderLinkID: "1666800494330512129"},
		}}
	}
	for _, tc := range []struct {
		name     string
		outcomes []BatchOrderOutcome
	}{
		{name: "no outcomes"},
		{name: "fewer outcomes than orders", outcomes: []BatchOrderOutcome{{Message: "OK"}}},
		{name: "more outcomes than orders", outcomes: []BatchOrderOutcome{{Message: "OK"}, {Code: 170213, Message: "Order does not exist."}, {Message: "OK"}}},
	} {
		result, err := applyBatchOrderOutcomes(orders(), tc.outcomes)
		assert.ErrorIsf(t, err, errBatchOutcomeMismatch, "applyBatchOrderOutcomes should reject %s", tc.name)
		assert.Nilf(t, result, "applyBatchOrderOutcomes should return no result with %s", tc.name)
	}

	resp := orders()
	result, err := applyBatchOrderOutcomes(resp, []BatchOrderOutcome{{Message: "OK"}, {Code: 170213, Message: "Order does not exist."}})
	require.NoError(t, err, "applyBatchOrderOutcomes must not error")
	exp := &BatchOrderResponse{List: []BatchOrderResult{
		{Category: "spot", Symbol: "BTCUSDT", OrderID: "1666800494330512128", OrderLinkID: "spot-btc-03", Message: "OK"},
		{Category: "spot", Symbol: "ATOMUSDT", OrderLinkID: "1666800494330512129", Code: 170213, Message: "Order does not exist."},
	}}
	assert.Equal(t, exp, result, "applyBatchOrderOutcomes should give each order the outcome at its position, so a rejected order carries its error code")
	assert.Same(t, resp, result, "applyBatchOrderOutcomes should set the outcomes on the result it was given")
}

func TestGetBorrowQuota(t *testing.T) {
	t.Parallel()
	_, err := e.GetBorrowQuota(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetBorrowQuota must reject a nil request")
	_, err = e.GetBorrowQuota(t.Context(), &BorrowQuotaRequest{Symbol: "BTCUSDT", Side: "Buy"})
	require.ErrorIs(t, err, errCategoryNotSet, "GetBorrowQuota must reject a request without a category")
	_, err = e.GetBorrowQuota(t.Context(), &BorrowQuotaRequest{Category: cSpot, Side: "Buy"})
	require.ErrorIs(t, err, errSymbolMissing, "GetBorrowQuota must reject a request without a symbol")
	_, err = e.GetBorrowQuota(t.Context(), &BorrowQuotaRequest{Category: cSpot, Symbol: "BTCUSDT"})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "GetBorrowQuota must reject a request without a side")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetBorrowQuota(t.Context(), &BorrowQuotaRequest{Category: cSpot, Symbol: "BTCUSDT", Side: "Buy"})
	require.NoError(t, err, "GetBorrowQuota must not error")
	if mockTests {
		exp := &BorrowQuotaResponse{
			Symbol:               "BTCUSDT",
			Side:                 "Buy",
			MaxTradeQuantity:     6.6065,
			MaxTradeAmount:       218014.01330797,
			SpotMaxTradeQuantity: 0.2728,
			SpotMaxTradeAmount:   9004.75628594,
			BorrowCoin:           currency.USDT,
		}
		assert.Equal(t, exp, result, "GetBorrowQuota should decode every field")
		return
	}
	assert.Equal(t, "BTCUSDT", result.Symbol, "GetBorrowQuota should return the requested symbol")
}

func TestSetDisconnectCancelAll(t *testing.T) {
	t.Parallel()
	err := e.SetDisconnectCancelAll(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetDisconnectCancelAll must reject a nil request")
	err = e.SetDisconnectCancelAll(t.Context(), &SetDisconnectCancelAllRequest{TimeWindow: time.Millisecond * 500})
	require.ErrorIs(t, err, errDisconnectTimeWindowNotSet, "SetDisconnectCancelAll must reject a window under a second")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, req := range []*SetDisconnectCancelAllRequest{
		{TimeWindow: 300 * time.Second},
		{Product: "DERIVATIVES", TimeWindow: 40 * time.Second},
	} {
		assert.NoErrorf(t, e.SetDisconnectCancelAll(t.Context(), req), "SetDisconnectCancelAll should not error for product %q", req.Product)
	}
}

func TestPreCheckOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PreCheckOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PreCheckOrder must reject a nil request")
	_, err = e.PreCheckOrder(t.Context(), &PlaceOrderRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "PreCheckOrder must reject a request without a category")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.PreCheckOrder(t.Context(), &PlaceOrderRequest{Category: cLinear, Symbol: "BTCUSDT", Side: "Buy", OrderType: "Limit", Quantity: 0.01, Price: 28000, TimeInForce: "GTC", OrderLinkID: "test129"})
	require.NoError(t, err, "PreCheckOrder must not error")
	if mockTests {
		exp := &PreCheckOrderResponse{
			OrderID:                     "24920bdb-4019-4e37-ad1c-876e3a855ac3",
			OrderLinkID:                 "test129",
			PreInitialMarginRateE4:      30,
			PreMaintenanceMarginRateE4:  21,
			PostInitialMarginRateE4:     357,
			PostMaintenanceMarginRateE4: 294,
		}
		assert.Equal(t, exp, result, "PreCheckOrder should decode every field")
		return
	}
	assert.Positive(t, result.PostInitialMarginRateE4.Float64(), "PreCheckOrder should return the margin rate after the order")
}
