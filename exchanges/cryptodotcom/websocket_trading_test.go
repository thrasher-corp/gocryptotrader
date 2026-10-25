package cryptodotcom

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
)

// tradingTestBatch returns the batch the tests send: two cancels and two post-only buys of BTCUSD-PERP far below its
// market, which inherit the batch's instrument
func tradingTestBatch() *BatchOrderRequest {
	return &BatchOrderRequest{
		InstrumentName: "BTCUSD-PERP",
		Cancels:        []CancelOrderRequest{{OrderID: tradingTestOrderID}, {ClientOrderID: "my-client-order-id"}},
		Creates: []CreateOrderRequest{
			{
				Side:                  SideBuy,
				OrderType:             OrderTypeLimit,
				Price:                 41555,
				Quantity:              0.01,
				ExecutionInstructions: []string{ExecutionInstructionPostOnly, ExecutionInstructionReduceOnly},
				TimeInForce:           TimeInForceGoodTillCancel,
				ClientOrderID:         "optional-client-id",
			},
			{Side: SideBuy, OrderType: OrderTypeLimit, Price: 41000, Quantity: 0.0001, ExecutionInstructions: []string{ExecutionInstructionPostOnly}},
		},
	}
}

func TestWebsocketAPICreateOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAPI().CreateOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateOrder must refuse a nil request")
	noInstrument := tradingTestOrder()
	noInstrument.InstrumentName = ""
	_, err = e.WebsocketAPI().CreateOrder(t.Context(), noInstrument)
	require.ErrorIs(t, err, errInstrumentNameRequired, "CreateOrder must refuse an order without an instrument")
	for _, tc := range createOrderRefusals {
		req := tradingTestOrder()
		tc.modify(req)
		_, err := e.WebsocketAPI().CreateOrder(t.Context(), req)
		assert.ErrorIsf(t, err, tc.err, "CreateOrder should refuse %s", tc.name)
	}
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	req := tradingTestOrder()
	if !mockTests {
		req.ClientOrderID = newTradingTestClientOrderID()
	}
	result, err := ex.WebsocketAPI().CreateOrder(t.Context(), req)
	require.NoError(t, err, "CreateOrder must not error")
	if mockTests {
		exp := &OrderIDResponse{OrderID: tradingTestOrderID, ClientOrderID: tradingTestOrderClientOrderID}
		assert.Equal(t, exp, result, "CreateOrder should decode every field")
		requests := venue.received("private/create-order")
		require.Len(t, requests, 1, "CreateOrder must send one request")
		assert.JSONEq(t, tradingTestOrderParams, string(requests[0].Params), "CreateOrder should send the order")

		venue.respond("private/create-order", `null`)
		_, err = ex.WebsocketAPI().CreateOrder(t.Context(), req)
		assert.ErrorIs(t, err, common.ErrNoResponse, "CreateOrder should error for a null result")
	} else {
		cancelTradingOrderAfterTest(t, result.OrderID)
		assert.Equal(t, req.ClientOrderID, result.ClientOrderID, "CreateOrder should return the order's client order ID")
	}
}

func TestWebsocketAPIAmendOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range amendOrderRefusals {
		_, err := e.WebsocketAPI().AmendOrder(t.Context(), tc.req)
		assert.ErrorIsf(t, err, tc.err, "AmendOrder should refuse %s", tc.name)
	}
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	req := &AmendOrderRequest{OrderID: tradingTestOrderID, NewPrice: 50000, NewQuantity: 0.0002}
	if !mockTests {
		req.OrderID = placeTradingTestOrder(t)
		// Crypto.com answers an amend with the order ID it was given, as its documented example shows
		cancelTradingOrderAfterTest(t, req.OrderID)
	}
	result, err := ex.WebsocketAPI().AmendOrder(t.Context(), req)
	require.NoError(t, err, "AmendOrder must not error")
	if mockTests {
		exp := &OrderIDResponse{OrderID: tradingTestOrderID, ClientOrderID: tradingTestOrderClientOrderID}
		assert.Equal(t, exp, result, "AmendOrder should decode every field")
		requests := venue.received("private/amend-order")
		require.Len(t, requests, 1, "AmendOrder must send one request")
		assert.JSONEq(t, `{"order_id":"6530219466236720401","new_price":"50000","new_quantity":"0.0002"}`, string(requests[0].Params), "AmendOrder should send the amend")

		venue.respond("private/amend-order", `null`)
		_, err = ex.WebsocketAPI().AmendOrder(t.Context(), req)
		assert.ErrorIs(t, err, common.ErrNoResponse, "AmendOrder should error for a null result")
	} else {
		assert.Equal(t, req.OrderID, result.OrderID, "AmendOrder should return the amended order's ID")
	}
}

func TestWebsocketAPICancelExistingOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range cancelOrderRefusals {
		_, err := e.WebsocketAPI().CancelExistingOrder(t.Context(), tc.req)
		assert.ErrorIsf(t, err, tc.err, "CancelExistingOrder should refuse %s", tc.name)
	}
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	req := &CancelOrderRequest{OrderID: tradingTestOrderID}
	if !mockTests {
		req.OrderID = placeTradingTestOrder(t)
	}
	result, err := ex.WebsocketAPI().CancelExistingOrder(t.Context(), req)
	require.NoError(t, err, "CancelExistingOrder must not error")
	if mockTests {
		exp := &OrderIDResponse{OrderID: tradingTestOrderID, ClientOrderID: tradingTestOrderClientOrderID}
		assert.Equal(t, exp, result, "CancelExistingOrder should decode every field")
		requests := venue.received("private/cancel-order")
		require.Len(t, requests, 1, "CancelExistingOrder must send one request")
		assert.JSONEq(t, `{"order_id":"6530219466236720401"}`, string(requests[0].Params), "CancelExistingOrder should send the order to cancel")

		venue.respond("private/cancel-order", `null`)
		_, err = ex.WebsocketAPI().CancelExistingOrder(t.Context(), req)
		assert.ErrorIs(t, err, common.ErrNoResponse, "CancelExistingOrder should error for a null result")
	} else {
		assert.Equal(t, req.OrderID, result.OrderID, "CancelExistingOrder should return the cancelled order's ID")
	}
}

func TestWebsocketAPICancelAllExistingOrders(t *testing.T) {
	t.Parallel()
	for _, tc := range cancelAllOrdersRefusals {
		assert.ErrorIsf(t, e.WebsocketAPI().CancelAllExistingOrders(t.Context(), tc.req), tc.err, "CancelAllExistingOrders should refuse %s", tc.name)
	}
	// An instrument the other tests place no orders on, so that cancelling its orders live leaves theirs
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	req := &CancelAllOrdersRequest{InstrumentName: tradingTestCancelAllInstrument}
	require.NoError(t, ex.WebsocketAPI().CancelAllExistingOrders(t.Context(), req), "CancelAllExistingOrders must not error")
	if mockTests {
		requests := venue.received("private/cancel-all-orders")
		require.Len(t, requests, 1, "CancelAllExistingOrders must send one request")
		assert.JSONEq(t, `{"instrument_name":"ETHUSD-PERP"}`, string(requests[0].Params), "CancelAllExistingOrders should send the instrument")

		venue.fail("private/cancel-all-orders", &APIError{Code: 40004, Message: "MISSING_OR_INVALID_ARGUMENT"})
		var apiErr *APIError
		require.ErrorAs(t, ex.WebsocketAPI().CancelAllExistingOrders(t.Context(), req), &apiErr, "CancelAllExistingOrders must return the error Crypto.com reports")
		assert.Equal(t, &APIError{Code: 40004, Message: "MISSING_OR_INVALID_ARGUMENT"}, apiErr, "CancelAllExistingOrders should decode the reported error")
	}
}

func TestWebsocketAPIClosePosition(t *testing.T) {
	t.Parallel()
	for _, tc := range closePositionRefusals {
		_, err := e.WebsocketAPI().ClosePosition(t.Context(), tc.req)
		assert.ErrorIsf(t, err, tc.err, "ClosePosition should refuse %s", tc.name)
	}
	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		t.Skip("ClosePosition needs an open position, which the live tests do not open")
	}

	ex, venue := newWebsocketTestExchange(t)
	req := &ClosePositionRequest{InstrumentName: "BTCUSD-PERP", OrderType: OrderTypeLimit, Price: 30000, Quantity: 0.0001}
	result, err := ex.WebsocketAPI().ClosePosition(t.Context(), req)
	require.NoError(t, err, "ClosePosition must not error")
	exp := &OrderIDResponse{OrderID: "6530219466236720404", ClientOrderID: "1684d6e4-2c55-64e1-52c3-3aa9febc3a23"}
	assert.Equal(t, exp, result, "ClosePosition should decode every field")
	requests := venue.received("private/close-position")
	require.Len(t, requests, 1, "ClosePosition must send one request")
	assert.JSONEq(t, `{"instrument_name":"BTCUSD-PERP","type":"LIMIT","price":"30000","quantity":"0.0001"}`, string(requests[0].Params), "ClosePosition should send the close")

	venue.respond("private/close-position", `null`)
	_, err = ex.WebsocketAPI().ClosePosition(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "ClosePosition should error for a null result")
}

func TestWebsocketAPICreateOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAPI().CreateOrderList(t.Context(), nil)
	require.ErrorIs(t, err, errOrderListEmpty, "CreateOrderList must refuse an empty list")
	_, err = e.WebsocketAPI().CreateOrderList(t.Context(), make([]CreateOrderRequest, 11))
	require.ErrorIs(t, err, errOrderListTooLong, "CreateOrderList must refuse a list of 11 orders")
	_, err = e.WebsocketAPI().CreateOrderList(t.Context(), []CreateOrderRequest{*tradingTestOrder(), {Side: SideBuy, OrderType: OrderTypeMarket, Quantity: 0.0001}})
	require.ErrorIs(t, err, errInstrumentNameRequired, "CreateOrderList must refuse an order without an instrument")
	for _, tc := range createOrderRefusals {
		orders := []CreateOrderRequest{*tradingTestOrder(), *tradingTestOrder()}
		tc.modify(&orders[1])
		_, err := e.WebsocketAPI().CreateOrderList(t.Context(), orders)
		assert.ErrorIsf(t, err, tc.err, "CreateOrderList should refuse %s", tc.name)
		assert.ErrorContainsf(t, err, "order 1:", "CreateOrderList should name the index of %s", tc.name)
	}
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	orders := tradingTestOrderList()
	if !mockTests {
		for i := range orders {
			orders[i].ClientOrderID = newTradingTestClientOrderID() + "-" + strconv.Itoa(i)
		}
	}
	result, err := ex.WebsocketAPI().CreateOrderList(t.Context(), orders)
	require.NoError(t, err, "CreateOrderList must not error")
	if mockTests {
		exp := []OrderListEntryResponse{
			{Index: 0, Code: 0, ClientOrderID: "api_leg1", OrderID: "5755600460443882762"},
			{Index: 1, Code: 0, ClientOrderID: "api_leg2", OrderID: "5755600460443882763"},
		}
		assert.Equal(t, exp, result, "CreateOrderList should decode every field")
		requests := venue.received("private/create-order-list")
		require.Len(t, requests, 1, "CreateOrderList must send one request")
		assert.JSONEq(t, tradingTestOrderListParams, string(requests[0].Params), "CreateOrderList should send the list")

		venue.respond("private/create-order-list", `null`)
		_, err = ex.WebsocketAPI().CreateOrderList(t.Context(), orders)
		assert.ErrorIs(t, err, common.ErrNoResponse, "CreateOrderList should error for a null result")
	} else {
		for i := range result {
			if result[i].Code == 0 {
				cancelTradingOrderAfterTest(t, result[i].OrderID)
			}
		}
		assert.Equal(t, len(orders), len(result), "CreateOrderList should report each order's outcome")
	}
}

func TestWebsocketAPICancelOrderList(t *testing.T) {
	t.Parallel()
	for _, tc := range cancelOrderListRefusals {
		_, err := e.WebsocketAPI().CancelOrderList(t.Context(), tc.orders)
		assert.ErrorIsf(t, err, tc.err, "CancelOrderList should refuse %s", tc.name)
	}
	_, err := e.WebsocketAPI().CancelOrderList(t.Context(), []CancelOrderListEntryRequest{{OrderID: tradingTestOrderID}, {InstrumentName: "CRO_USD"}})
	assert.ErrorContains(t, err, "order 1:", "CancelOrderList should name the index of the order named by neither ID")
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	orders := []CancelOrderListEntryRequest{{InstrumentName: "CRO_USD", ClientOrderID: "api_leg1"}, {InstrumentName: "CRO_USD", ClientOrderID: "api_leg2"}}
	if !mockTests {
		orders = []CancelOrderListEntryRequest{
			{InstrumentName: "BTCUSD-PERP", OrderID: placeTradingTestOrder(t)},
			{InstrumentName: "BTCUSD-PERP", OrderID: placeTradingTestOrder(t)},
		}
	}
	result, err := ex.WebsocketAPI().CancelOrderList(t.Context(), orders)
	require.NoError(t, err, "CancelOrderList must not error")
	if mockTests {
		exp := []OrderListEntryResponse{{Index: 0, Code: 0}, {Index: 1, Code: 0}}
		assert.Equal(t, exp, result, "CancelOrderList should decode every field")
		requests := venue.received("private/cancel-order-list")
		require.Len(t, requests, 1, "CancelOrderList must send one request")
		assert.JSONEq(t, `{"contingency_type":"LIST","order_list":[{"instrument_name":"CRO_USD","client_oid":"api_leg1"},{"instrument_name":"CRO_USD","client_oid":"api_leg2"}]}`, string(requests[0].Params), "CancelOrderList should send the list")

		venue.respond("private/cancel-order-list", `null`)
		_, err = ex.WebsocketAPI().CancelOrderList(t.Context(), orders)
		assert.ErrorIs(t, err, common.ErrNoResponse, "CancelOrderList should error for a null result")
	} else {
		for i := range result {
			assert.Zerof(t, result[i].Code, "CancelOrderList should cancel order %v", result[i].Index)
		}
	}
}

func TestWebsocketAPISendBatchOrderRequest(t *testing.T) {
	t.Parallel()
	elevenCreates := make([]CreateOrderRequest, 11)
	for i := range elevenCreates {
		elevenCreates[i] = *tradingTestOrder()
	}
	for _, tc := range []struct {
		name string
		req  *BatchOrderRequest
		err  error
	}{
		{"a nil request", nil, common.ErrNilPointer},
		{"a batch without an instrument", &BatchOrderRequest{Cancels: []CancelOrderRequest{{OrderID: tradingTestOrderID}}}, errInstrumentNameRequired},
		{"an empty batch", &BatchOrderRequest{InstrumentName: "BTCUSD-PERP"}, errOrderListEmpty},
		{"a batch of 11 creates", &BatchOrderRequest{InstrumentName: "BTCUSD-PERP", Creates: elevenCreates}, errOrderListTooLong},
		{"a cancel naming no order", &BatchOrderRequest{InstrumentName: "BTCUSD-PERP", Cancels: []CancelOrderRequest{{OrderID: tradingTestOrderID}, {}}}, errOrderIDRequired},
		{"a create of another instrument", &BatchOrderRequest{InstrumentName: "ETHUSD-PERP", Creates: []CreateOrderRequest{*tradingTestOrder()}}, errBatchInstrumentMismatch},
	} {
		_, err := e.WebsocketAPI().SendBatchOrderRequest(t.Context(), tc.req)
		assert.ErrorIsf(t, err, tc.err, "SendBatchOrderRequest should refuse %s", tc.name)
	}
	_, err := e.WebsocketAPI().SendBatchOrderRequest(t.Context(), &BatchOrderRequest{InstrumentName: "BTCUSD-PERP", Cancels: []CancelOrderRequest{{OrderID: tradingTestOrderID}, {}}})
	assert.ErrorContains(t, err, "cancel 1:", "SendBatchOrderRequest should name the index of the cancel naming no order")
	for _, tc := range createOrderRefusals {
		req := tradingTestBatch()
		tc.modify(&req.Creates[1])
		_, err := e.WebsocketAPI().SendBatchOrderRequest(t.Context(), req)
		assert.ErrorIsf(t, err, tc.err, "SendBatchOrderRequest should refuse %s", tc.name)
		assert.ErrorContainsf(t, err, "create 1:", "SendBatchOrderRequest should name the index of %s", tc.name)
	}
	skipLiveUnlessCanManipulate(t)

	ex, venue := newWebsocketTestExchange(t)
	req := tradingTestBatch()
	result, err := ex.WebsocketAPI().SendBatchOrderRequest(t.Context(), req)
	require.NoError(t, err, "SendBatchOrderRequest must not error")
	if !mockTests {
		for i := range result.Creates {
			if result.Creates[i].Code == 0 {
				cancelTradingOrderAfterTest(t, result.Creates[i].OrderID)
			}
		}
		assert.Equal(t, len(req.Creates), len(result.Creates), "SendBatchOrderRequest should report each create's outcome")
		return
	}
	exp := &BatchOrderResponse{
		Cancels: []OrderListEntryResponse{{Code: 0, OrderID: tradingTestOrderID}, {Code: 0, ClientOrderID: "my-client-order-id"}},
		Creates: []OrderListEntryResponse{{Code: 0, OrderID: "6530219466236720409", ClientOrderID: "optional-client-id"}, {Code: 0, OrderID: "6530219466236720410", ClientOrderID: "1792890506012"}},
	}
	assert.Equal(t, exp, result, "SendBatchOrderRequest should decode every field")

	tenCreates := make([]CreateOrderRequest, 10)
	tenCreatesParams := make([]string, 10)
	for i := range tenCreates {
		tenCreates[i] = CreateOrderRequest{Side: SideBuy, OrderType: OrderTypeLimit, Price: 41000, Quantity: 0.0001}
		tenCreatesParams[i] = `{"side":"BUY","type":"LIMIT","price":"41000","quantity":"0.0001"}`
	}
	cases := []struct {
		name   string
		req    *BatchOrderRequest
		params string
	}{
		{
			name: "every optional param, with a create naming the batch's instrument",
			req: &BatchOrderRequest{
				InstrumentName:                        "BTCUSD-PERP",
				Cancels:                               []CancelOrderRequest{{OrderID: tradingTestOrderID, ClientOrderID: tradingTestOrderClientOrderID}},
				Creates:                               []CreateOrderRequest{{InstrumentName: "BTCUSD-PERP", Side: SideSell, OrderType: OrderTypeMarket, Quantity: 0.01}},
				SelfTradePreventionCancelType:         SelfTradePreventionCancelMaker,
				SelfTradePreventionAccountEnforcement: true,
				SelfTradePreventionGroup:              "1",
			},
			params: `{"instrument_name":"BTCUSD-PERP","cancels":[{"order_id":"6530219466236720401","client_oid":"c5f682ed-7108-4f1c-b755-972fcdca0f02"}],"creates":[{"side":"SELL","type":"MARKET","quantity":"0.01"}],"stp_cancel_type":"M","stp_account_enforce_flag":true,"stp_group":"1"}`,
		},
		{
			name:   "only cancels",
			req:    &BatchOrderRequest{InstrumentName: "BTCUSD-PERP", Cancels: []CancelOrderRequest{{ClientOrderID: "my-client-order-id"}}},
			params: `{"instrument_name":"BTCUSD-PERP","cancels":[{"client_oid":"my-client-order-id"}]}`,
		},
		{
			name:   "only 10 creates",
			req:    &BatchOrderRequest{InstrumentName: "BTCUSD-PERP", Creates: tenCreates},
			params: `{"instrument_name":"BTCUSD-PERP","creates":[` + strings.Join(tenCreatesParams, ",") + `]}`,
		},
	}
	for _, tc := range cases {
		_, err := ex.WebsocketAPI().SendBatchOrderRequest(t.Context(), tc.req)
		assert.NoErrorf(t, err, "SendBatchOrderRequest should not error for %s", tc.name)
	}
	requests := venue.received("private/batch-order-request")
	require.Len(t, requests, len(cases)+1, "SendBatchOrderRequest must send one request each call")
	assert.JSONEq(t, `{"instrument_name":"BTCUSD-PERP","cancels":[{"order_id":"6530219466236720401"},{"client_oid":"my-client-order-id"}],"creates":[{"side":"BUY","type":"LIMIT","price":"41555","quantity":"0.01","client_oid":"optional-client-id","time_in_force":"GOOD_TILL_CANCEL","exec_inst":["POST_ONLY","REDUCE_ONLY"]},{"side":"BUY","type":"LIMIT","price":"41000","quantity":"0.0001","exec_inst":["POST_ONLY"]}]}`, string(requests[0].Params), "SendBatchOrderRequest should send the batch, its creates inheriting its instrument")
	for i, tc := range cases {
		assert.JSONEqf(t, tc.params, string(requests[i+1].Params), "SendBatchOrderRequest should send the params of %s", tc.name)
	}
	assert.Equal(t, tradingTestBatch(), req, "SendBatchOrderRequest should leave the request's creates as they were")

	venue.respond("private/batch-order-request", `null`)
	_, err = ex.WebsocketAPI().SendBatchOrderRequest(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "SendBatchOrderRequest should error for a null result")
}
