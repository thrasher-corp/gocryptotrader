package cryptodotcom

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// advancedOrderIDResult is the result the docs give for creating, amending and cancelling an advanced order
const advancedOrderIDResult = `{"order_id":"5755600460443882762","client_oid":"c5f682ed-7108-4f1c-b755-972fcdca0f02"}`

// newAdvancedRefusalExchange returns an exchange for checking a method refuses a request before sending it, so the
// checks send nothing to Crypto.com or the recorded mock even when a check fails: its test server fails the test on any
// request, and its websocket is not connected
func newAdvancedRefusalExchange(t *testing.T) *Exchange {
	t.Helper()
	return newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Failf(t, "A refused request should not be sent", "%s %s", r.Method, r.URL.Path)
		http.Error(w, "a refused request was sent", http.StatusBadRequest)
	}))
}

// advancedParamsCase is a request to send and the params it should be sent with
type advancedParamsCase[T any] struct {
	name   string
	req    T
	params string
}

// testAdvancedOrderParams checks each case's request is sent with its params, by send against a test server answering
// method's requests with result
func testAdvancedOrderParams[T any](t *testing.T, method, result string, send func(*Exchange, T) error, cases []advancedParamsCase[T]) {
	t.Helper()
	ex, sent := newSignedRecorder(t, method, result)
	for i, tc := range cases {
		require.NoErrorf(t, send(ex, tc.req), "Sending %s must not error", tc.name)
		params := sent()
		require.Lenf(t, params, i+1, "Sending %s must send one request", tc.name)
		assert.JSONEqf(t, tc.params, params[i], "Sending %s should send its params", tc.name)
	}
}

// advancedTriggerOrder returns the docs' example trigger order repriced away from the market, as the live tests that may
// change the account place it: a sell STOP_LIMIT of 10 CRO_USD at 0.0279, placed once the mark price falls to 0.028,
// half CRO_USD's price on 2026-10-25
func advancedTriggerOrder() CreateAdvancedOrderRequest {
	return CreateAdvancedOrderRequest{
		InstrumentName: "CRO_USD",
		Side:           SideSell,
		OrderType:      OrderTypeStopLimit,
		Quantity:       10,
		Price:          0.0279,
		ReferencePrice: 0.028,
		ClientOrderID:  "c5f682ed-7108-4f1c-b755-972fcdca0f02",
	}
}

// triggerOrderRefusals are changes to a trigger order which Crypto.com refuses, and the errors CreateAdvancedOrder,
// CreateAdvancedOrderList and their websocket forms refuse them with before sending
var triggerOrderRefusals = []struct {
	name   string
	change func(*CreateAdvancedOrderRequest)
	err    error
}{
	{name: "an order without an instrument", change: func(r *CreateAdvancedOrderRequest) { r.InstrumentName = "" }, err: errInstrumentNameRequired},
	{name: "an order with a client order ID of 37 characters", change: func(r *CreateAdvancedOrderRequest) { r.ClientOrderID = strings.Repeat("c", 37) }, err: errClientOrderIDTooLong},
	{name: "an order with an unknown time in force", change: func(r *CreateAdvancedOrderRequest) { r.TimeInForce = "GOOD_TILL_DATE" }, err: errInvalidTimeInForce},
	{name: "a post-only fill-or-kill order", change: func(r *CreateAdvancedOrderRequest) {
		r.TimeInForce, r.ExecutionInstructions = TimeInForceFillOrKill, []string{ExecutionInstructionPostOnly}
	}, err: errInvalidTimeInForce},
	{name: "an order with both post-only instructions", change: func(r *CreateAdvancedOrderRequest) {
		r.ExecutionInstructions = []string{ExecutionInstructionPostOnly, ExecutionInstructionSmartPostOnly}
	}, err: errInvalidExecutionInstruction},
	{name: "an order with an unknown execution instruction", change: func(r *CreateAdvancedOrderRequest) { r.ExecutionInstructions = []string{"HIDDEN"} }, err: errInvalidExecutionInstruction},
	{name: "a self-trade prevention scope without an instruction", change: func(r *CreateAdvancedOrderRequest) {
		r.SelfTradePreventionScope = SelfTradePreventionScopeMasterOrSubAccount
	}, err: errInvalidSelfTradePreventionInstruction},
	{name: "a self-trade prevention ID that is not a number", change: func(r *CreateAdvancedOrderRequest) { r.SelfTradePreventionID = "first" }, err: errInvalidSelfTradePreventionID},
	{name: "an order without a side", change: func(r *CreateAdvancedOrderRequest) { r.Side = "" }, err: errInvalidSide},
	{name: "an order with an unknown side", change: func(r *CreateAdvancedOrderRequest) { r.Side = "HOLD" }, err: errInvalidSide},
	{name: "a LIMIT order", change: func(r *CreateAdvancedOrderRequest) { r.OrderType = OrderTypeLimit }, err: errInvalidOrderType},
	{name: "a MARKET order", change: func(r *CreateAdvancedOrderRequest) { r.OrderType = OrderTypeMarket }, err: errInvalidOrderType},
	{name: "an order without a type", change: func(r *CreateAdvancedOrderRequest) { r.OrderType = "" }, err: errInvalidOrderType},
	{name: "an order without a quantity", change: func(r *CreateAdvancedOrderRequest) { r.Quantity = 0 }, err: errInvalidQuantity},
	{name: "an order with a negative quantity", change: func(r *CreateAdvancedOrderRequest) { r.Quantity = -10 }, err: errInvalidQuantity},
	{name: "a STOP_LIMIT order without a price", change: func(r *CreateAdvancedOrderRequest) { r.Price = 0 }, err: errInvalidPrice},
	{name: "a TAKE_PROFIT_LIMIT order without a price", change: func(r *CreateAdvancedOrderRequest) { r.OrderType, r.Price = OrderTypeTakeProfitLimit, 0 }, err: errInvalidPrice},
	{name: "a STOP_LIMIT order without a reference price", change: func(r *CreateAdvancedOrderRequest) { r.ReferencePrice = 0 }, err: errInvalidReferencePrice},
	{name: "a STOP_LOSS order without a reference price", change: func(r *CreateAdvancedOrderRequest) { r.OrderType, r.ReferencePrice = OrderTypeStopLoss, 0 }, err: errInvalidReferencePrice},
	{name: "a TAKE_PROFIT order without a reference price", change: func(r *CreateAdvancedOrderRequest) { r.OrderType, r.ReferencePrice = OrderTypeTakeProfit, 0 }, err: errInvalidReferencePrice},
	{name: "a TAKE_PROFIT_LIMIT order without a reference price", change: func(r *CreateAdvancedOrderRequest) { r.OrderType, r.ReferencePrice = OrderTypeTakeProfitLimit, 0 }, err: errInvalidReferencePrice},
	{name: "an order with an unknown reference price type", change: func(r *CreateAdvancedOrderRequest) { r.ReferencePriceType = "MID_PRICE" }, err: errInvalidReferencePriceType},
	{name: "an order attached to a position without REDUCE_ONLY", change: func(r *CreateAdvancedOrderRequest) { r.AttachIsolationID = "0" }, err: errAttachIsolationReduceOnly},
	{
		name: "an order attached to a position with POST_ONLY rather than REDUCE_ONLY",
		change: func(r *CreateAdvancedOrderRequest) {
			r.AttachIsolationID, r.ExecutionInstructions = "0", []string{ExecutionInstructionPostOnly}
		},
		err: errAttachIsolationReduceOnly,
	},
	{
		name: "an order attached to both a position and an order",
		change: func(r *CreateAdvancedOrderRequest) {
			r.AttachIsolationID, r.ExecutionInstructions, r.AttachOrderID = "0", []string{ExecutionInstructionReduceOnly}, "5755600460443890101"
		},
		err: errAttachIsolationConflict,
	},
	{
		name: "an order attached to a position with an isolation ID",
		change: func(r *CreateAdvancedOrderRequest) {
			r.AttachIsolationID, r.ExecutionInstructions, r.IsolationID = "5755600460443891277", []string{ExecutionInstructionReduceOnly}, "5755600460443891277"
		},
		err: errAttachIsolationConflict,
	},
}

// advancedOrderLegChanges are changes to an order of an OCO, OTO or OTOCO which Crypto.com refuses, and the errors the
// strategies' create methods refuse them with before sending
var advancedOrderLegChanges = []struct {
	name   string
	change func(*AdvancedOrderLegRequest)
	err    error
}{
	{name: "an order without an instrument", change: func(o *AdvancedOrderLegRequest) { o.InstrumentName = "" }, err: errInstrumentNameRequired},
	{name: "an order with a client order ID of 37 characters", change: func(o *AdvancedOrderLegRequest) { o.ClientOrderID = strings.Repeat("c", 37) }, err: errClientOrderIDTooLong},
	{name: "a post-only fill-or-kill order", change: func(o *AdvancedOrderLegRequest) {
		o.TimeInForce, o.ExecutionInstructions = TimeInForceFillOrKill, []string{ExecutionInstructionPostOnly}
	}, err: errInvalidTimeInForce},
	{name: "an order with an unknown execution instruction", change: func(o *AdvancedOrderLegRequest) { o.ExecutionInstructions = []string{"HIDDEN"} }, err: errInvalidExecutionInstruction},
	{name: "a self-trade prevention ID that is not a number", change: func(o *AdvancedOrderLegRequest) { o.SelfTradePreventionID = "first" }, err: errInvalidSelfTradePreventionID},
	{name: "an order without a side", change: func(o *AdvancedOrderLegRequest) { o.Side = "" }, err: errInvalidSide},
	{name: "an order with an unknown type", change: func(o *AdvancedOrderLegRequest) { o.OrderType = "STOP" }, err: errInvalidOrderType},
	{name: "an order without a quantity", change: func(o *AdvancedOrderLegRequest) { o.Quantity = 0 }, err: errInvalidQuantity},
	{name: "a sell spending a notional", change: func(o *AdvancedOrderLegRequest) { o.Side, o.Quantity, o.Notional = SideSell, 0, 8000 }, err: errInvalidQuantity},
	{
		name: "a STOP_LIMIT buy spending a notional",
		change: func(o *AdvancedOrderLegRequest) {
			o.Side, o.OrderType, o.Price, o.Quantity, o.Notional = SideBuy, OrderTypeStopLimit, 81000, 0, 8000
		},
		err: errInvalidQuantity,
	},
	{name: "a LIMIT order without a price", change: func(o *AdvancedOrderLegRequest) { o.OrderType, o.Price = OrderTypeLimit, 0 }, err: errInvalidPrice},
	{name: "a STOP_LIMIT order without a price", change: func(o *AdvancedOrderLegRequest) { o.OrderType, o.Price = OrderTypeStopLimit, 0 }, err: errInvalidPrice},
	{name: "a TAKE_PROFIT_LIMIT order without a price", change: func(o *AdvancedOrderLegRequest) { o.OrderType, o.Price = OrderTypeTakeProfitLimit, 0 }, err: errInvalidPrice},
	{name: "a trigger order without a reference price", change: func(o *AdvancedOrderLegRequest) { o.ReferencePrice = 0 }, err: errInvalidReferencePrice},
	{name: "an order with an unknown reference price type", change: func(o *AdvancedOrderLegRequest) { o.ReferencePriceType = "MID_PRICE" }, err: errInvalidReferencePriceType},
}

// advancedListRefusal is a list of orders which Crypto.com refuses, and the error the methods sending it refuse it with
// before sending
type advancedListRefusal[T any] struct {
	name   string
	orders []T
	err    error
}

// contingencyRefusal is an OCO, OTO or OTOCO which Crypto.com refuses, and the error its create methods refuse it with
type contingencyRefusal = advancedListRefusal[AdvancedOrderLegRequest]

// legRefusals returns a strategy's orders with its last order, a trigger order, changed in each way Crypto.com refuses
func legRefusals(orders func() []AdvancedOrderLegRequest) []contingencyRefusal {
	refusals := make([]contingencyRefusal, 0, len(advancedOrderLegChanges))
	for _, c := range advancedOrderLegChanges {
		o := orders()
		c.change(&o[len(o)-1])
		refusals = append(refusals, contingencyRefusal{name: "an OCO, OTO or OTOCO with " + c.name, orders: o, err: c.err})
	}
	return refusals
}

// ocoOrders returns the docs' example OCO resized and repriced away from the market, as the live tests that may change
// the account place it: a sell LIMIT of 0.0001 BTC_USD at 120000, and a sell STOP_LOSS of 0.0001 placed once the mark
// price falls to 40000, BTC_USD trading at 80700 on 2026-10-25
func ocoOrders() []AdvancedOrderLegRequest {
	return []AdvancedOrderLegRequest{
		{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeLimit, Quantity: 0.0001, Price: 120000},
		{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeStopLoss, Quantity: 0.0001, ReferencePrice: 40000},
	}
}

// ocoRefusals returns the OCOs CreateOCO and its websocket form refuse before sending
func ocoRefusals() []contingencyRefusal {
	limit, stopLoss := ocoOrders()[0], ocoOrders()[1]
	market := limit
	market.OrderType, market.Price = OrderTypeMarket, 0
	indexStopLoss, lastStopLoss := stopLoss, stopLoss
	indexStopLoss.ReferencePriceType, lastStopLoss.ReferencePriceType = ReferencePriceTypeIndexPrice, ReferencePriceTypeLastPrice
	return append(legRefusals(ocoOrders),
		contingencyRefusal{name: "an OCO without orders", err: errOrderListEmpty},
		contingencyRefusal{name: "an OCO of one order", orders: []AdvancedOrderLegRequest{limit}, err: errOrderListTooShort},
		contingencyRefusal{name: "an OCO of three orders", orders: []AdvancedOrderLegRequest{limit, stopLoss, stopLoss}, err: errOrderListTooLong},
		contingencyRefusal{name: "an OCO of two LIMIT orders", orders: []AdvancedOrderLegRequest{limit, limit}, err: errInvalidContingencyOrders},
		contingencyRefusal{name: "an OCO of two trigger orders", orders: []AdvancedOrderLegRequest{stopLoss, stopLoss}, err: errInvalidContingencyOrders},
		contingencyRefusal{name: "an OCO of a MARKET order and a trigger order", orders: []AdvancedOrderLegRequest{market, stopLoss}, err: errInvalidContingencyOrders},
		contingencyRefusal{name: "an OCO whose trigger order compares the index price", orders: []AdvancedOrderLegRequest{limit, indexStopLoss}, err: errInvalidReferencePriceType},
		contingencyRefusal{name: "an OCO whose trigger order compares the last price", orders: []AdvancedOrderLegRequest{limit, lastStopLoss}, err: errInvalidReferencePriceType},
	)
}

// otoOrders returns the docs' example OTO resized and repriced away from the market, as the live tests that may change
// the account place it: a buy LIMIT of 0.0001 BTC_USD at 40000, and a sell STOP_LOSS of 0.0001 placed once the mark
// price falls to 30000, BTC_USD trading at 80700 on 2026-10-25
func otoOrders() []AdvancedOrderLegRequest {
	return []AdvancedOrderLegRequest{
		{InstrumentName: "BTC_USD", Side: SideBuy, OrderType: OrderTypeLimit, Quantity: 0.0001, Price: 40000},
		{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeStopLoss, Quantity: 0.0001, ReferencePrice: 30000},
	}
}

// otoRefusals returns the OTOs CreateOTO and its websocket form refuse before sending
func otoRefusals() []contingencyRefusal {
	limit, stopLoss := otoOrders()[0], otoOrders()[1]
	market := limit
	market.OrderType, market.Price = OrderTypeMarket, 0
	return append(legRefusals(otoOrders),
		contingencyRefusal{name: "an OTO without orders", err: errOrderListEmpty},
		contingencyRefusal{name: "an OTO of one order", orders: []AdvancedOrderLegRequest{limit}, err: errOrderListTooShort},
		contingencyRefusal{name: "an OTO of three orders", orders: []AdvancedOrderLegRequest{limit, stopLoss, stopLoss}, err: errOrderListTooLong},
		contingencyRefusal{name: "an OTO of two LIMIT orders", orders: []AdvancedOrderLegRequest{limit, limit}, err: errInvalidContingencyOrders},
		contingencyRefusal{name: "an OTO of a LIMIT and a MARKET order", orders: []AdvancedOrderLegRequest{limit, market}, err: errInvalidContingencyOrders},
		contingencyRefusal{name: "an OTO of two trigger orders", orders: []AdvancedOrderLegRequest{stopLoss, stopLoss}, err: errInvalidContingencyOrders},
	)
}

// otocoOrders returns the docs' example OTOCO resized and repriced away from the market, as the live tests that may
// change the account place it: a buy LIMIT of 0.0001 BTC_USD at 40000, a sell STOP_LOSS of 0.0001 placed once the mark
// price falls to 30000, and a sell TAKE_PROFIT of 0.0001 placed once it rises to 120000, BTC_USD trading at 80700 on
// 2026-10-25
func otocoOrders() []AdvancedOrderLegRequest {
	return []AdvancedOrderLegRequest{
		{InstrumentName: "BTC_USD", Side: SideBuy, OrderType: OrderTypeLimit, Quantity: 0.0001, Price: 40000},
		{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeStopLoss, Quantity: 0.0001, ReferencePrice: 30000},
		{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeTakeProfit, Quantity: 0.0001, ReferencePrice: 120000},
	}
}

// otocoRefusals returns the OTOCOs CreateOTOCO and its websocket form refuse before sending
func otocoRefusals() []contingencyRefusal {
	limit, stopLoss, takeProfit := otocoOrders()[0], otocoOrders()[1], otocoOrders()[2]
	return append(legRefusals(otocoOrders),
		contingencyRefusal{name: "an OTOCO without orders", err: errOrderListEmpty},
		contingencyRefusal{name: "an OTOCO of two orders", orders: []AdvancedOrderLegRequest{limit, stopLoss}, err: errOrderListTooShort},
		contingencyRefusal{name: "an OTOCO of four orders", orders: []AdvancedOrderLegRequest{limit, stopLoss, takeProfit, takeProfit}, err: errOrderListTooLong},
		contingencyRefusal{name: "an OTOCO of two LIMIT orders", orders: []AdvancedOrderLegRequest{limit, limit, takeProfit}, err: errInvalidContingencyOrders},
		contingencyRefusal{name: "an OTOCO without a LIMIT or MARKET order", orders: []AdvancedOrderLegRequest{stopLoss, takeProfit, takeProfit}, err: errInvalidContingencyOrders},
		contingencyRefusal{name: "an OTOCO of two stop losses", orders: []AdvancedOrderLegRequest{limit, stopLoss, stopLoss}, err: errInvalidContingencyOrders},
		contingencyRefusal{name: "an OTOCO of two take profits", orders: []AdvancedOrderLegRequest{limit, takeProfit, takeProfit}, err: errInvalidContingencyOrders},
	)
}

// testCancelContingency tests a method cancelling an OCO, OTO or OTOCO by its list ID; the recorded mock cancels listID
func testCancelContingency(t *testing.T, name, method, listID string, cancel func(*Exchange, context.Context, string) error) {
	t.Helper()
	require.ErrorIsf(t, cancel(newAdvancedRefusalExchange(t), t.Context(), ""), errListIDRequired, "%s must refuse an empty list ID", name)
	testAdvancedOrderParams(t, method, "", func(ex *Exchange, id string) error {
		return cancel(ex, t.Context(), id)
	}, []advancedParamsCase[string]{
		{name: "a list ID", req: listID, params: `{"list_id":"` + listID + `"}`},
	})
	assert.ErrorIsf(t, cancel(newUnavailableServerExchange(t), t.Context(), listID), request.ErrBadStatus, "%s should error for an unavailable server", name)

	skipLiveUnlessCanManipulate(t)
	require.NoErrorf(t, cancel(e, t.Context(), listID), "%s must not error", name)
}

func TestCreateAdvancedOrder(t *testing.T) {
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t)
	_, err := refuser.CreateAdvancedOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateAdvancedOrder must refuse a nil request")
	for _, tc := range triggerOrderRefusals {
		req := advancedTriggerOrder()
		tc.change(&req)
		_, err := refuser.CreateAdvancedOrder(t.Context(), &req)
		assert.ErrorIsf(t, err, tc.err, "CreateAdvancedOrder should refuse %s", tc.name)
	}

	testAdvancedOrderParams(t, "private/advanced/create-order", advancedOrderIDResult, func(ex *Exchange, req *CreateAdvancedOrderRequest) error {
		_, err := ex.CreateAdvancedOrder(t.Context(), req)
		return err
	}, []advancedParamsCase[*CreateAdvancedOrderRequest]{
		{
			name:   "a STOP_LOSS order with only the required fields",
			req:    &CreateAdvancedOrderRequest{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeStopLoss, Quantity: 0.1, ReferencePrice: 45000},
			params: `{"instrument_name":"BTC_USD","side":"SELL","type":"STOP_LOSS","quantity":"0.1","ref_price":"45000"}`,
		},
		{
			name: "a TAKE_PROFIT_LIMIT order attached to an order, with every option",
			req: &CreateAdvancedOrderRequest{
				InstrumentName:                 "SHIB_USD",
				Side:                           SideBuy,
				OrderType:                      OrderTypeTakeProfitLimit,
				Quantity:                       1000000,
				Price:                          0.00001234,
				ReferencePrice:                 0.00001235,
				ReferencePriceType:             ReferencePriceTypeLastPrice,
				ClientOrderID:                  "shib-take-profit",
				TimeInForce:                    TimeInForceGoodTillCancel,
				ExecutionInstructions:          []string{ExecutionInstructionPostOnly},
				FeeInstrumentName:              currency.CRO,
				SelfTradePreventionID:          "0",
				SelfTradePreventionInstruction: SelfTradePreventionCancelBoth,
				SelfTradePreventionScope:       SelfTradePreventionScopeMasterOrSubAccount,
				AttachOrderID:                  "5755600460443890101",
			},
			params: `{"instrument_name":"SHIB_USD","side":"BUY","type":"TAKE_PROFIT_LIMIT","quantity":"1000000","price":"0.00001234","ref_price":"0.00001235","ref_price_type":"LAST_PRICE","client_oid":"shib-take-profit","time_in_force":"GOOD_TILL_CANCEL","exec_inst":["POST_ONLY"],"fee_instrument_name":"CRO","stp_id":"0","stp_inst":"B","stp_scope":"M","attach_order_id":"5755600460443890101"}`,
		},
		{
			name: "a STOP_LIMIT order attached to the cross margin position",
			req: &CreateAdvancedOrderRequest{
				InstrumentName:        "BTCUSD-PERP",
				Side:                  SideSell,
				OrderType:             OrderTypeStopLimit,
				Quantity:              0.01,
				Price:                 78400,
				ReferencePrice:        78500,
				ReferencePriceType:    ReferencePriceTypeMarkPrice,
				ExecutionInstructions: []string{ExecutionInstructionPostOnly, ExecutionInstructionReduceOnly},
				AttachIsolationID:     "0",
			},
			params: `{"instrument_name":"BTCUSD-PERP","side":"SELL","type":"STOP_LIMIT","quantity":"0.01","price":"78400","ref_price":"78500","ref_price_type":"MARK_PRICE","exec_inst":["POST_ONLY","REDUCE_ONLY"],"attach_isolation_id":"0"}`,
		},
		{
			name:   "a TAKE_PROFIT order on an isolated position",
			req:    &CreateAdvancedOrderRequest{InstrumentName: "BTCUSD-PERP", Side: SideBuy, OrderType: OrderTypeTakeProfit, Quantity: 0.01, ReferencePrice: 76000, ReferencePriceType: ReferencePriceTypeIndexPrice, IsolationID: "5755600460443891277"},
			params: `{"instrument_name":"BTCUSD-PERP","side":"BUY","type":"TAKE_PROFIT","quantity":"0.01","ref_price":"76000","ref_price_type":"INDEX_PRICE","isolation_id":"5755600460443891277"}`,
		},
	})

	req := advancedTriggerOrder()
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/advanced/create-order","code":0,"result":null}`).CreateAdvancedOrder(t.Context(), &req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateAdvancedOrder should error for a null result")
	_, err = newUnavailableServerExchange(t).CreateAdvancedOrder(t.Context(), &req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateAdvancedOrder should error for an unavailable server")

	skipLiveUnlessCanManipulate(t)
	result, err := e.CreateAdvancedOrder(t.Context(), &req)
	require.NoError(t, err, "CreateAdvancedOrder must not error")
	if mockTests {
		exp := &OrderIDResponse{OrderID: "5755600460443882762", ClientOrderID: "c5f682ed-7108-4f1c-b755-972fcdca0f02"}
		assert.Equal(t, exp, result, "CreateAdvancedOrder should decode every field")
	} else {
		assert.NotEmpty(t, result.OrderID, "CreateAdvancedOrder should return the order ID")
	}
}

// amendAdvancedOrderRefusals are the amendments AmendAdvancedOrder and its websocket form refuse before sending, and the
// errors they refuse them with
var amendAdvancedOrderRefusals = []struct {
	name string
	req  *AmendAdvancedOrderRequest
	err  error
}{
	{name: "a nil request", err: common.ErrNilPointer},
	{name: "an amendment without an order ID or client order ID", req: &AmendAdvancedOrderRequest{NewQuantity: 10, NewPrice: new(types.Number(0.0278))}, err: errOrderIDRequired},
	{name: "an amendment without a new quantity", req: &AmendAdvancedOrderRequest{OrderID: "5755600460443882762", NewPrice: new(types.Number(0.0278))}, err: errInvalidQuantity},
	{name: "an amendment with a negative new quantity", req: &AmendAdvancedOrderRequest{OrderID: "5755600460443882762", NewQuantity: -10, NewPrice: new(types.Number(0.0278))}, err: errInvalidQuantity},
	{name: "an amendment without a new price", req: &AmendAdvancedOrderRequest{OrderID: "5755600460443882762", NewQuantity: 10}, err: errNewPriceRequired},
}

func TestAmendAdvancedOrder(t *testing.T) {
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t)
	for _, tc := range amendAdvancedOrderRefusals {
		_, err := refuser.AmendAdvancedOrder(t.Context(), tc.req)
		assert.ErrorIsf(t, err, tc.err, "AmendAdvancedOrder should refuse %s", tc.name)
	}

	testAdvancedOrderParams(t, "private/advanced/amend-order", advancedOrderIDResult, func(ex *Exchange, req *AmendAdvancedOrderRequest) error {
		_, err := ex.AmendAdvancedOrder(t.Context(), req)
		return err
	}, []advancedParamsCase[*AmendAdvancedOrderRequest]{
		{
			name:   "an amendment to a trigger market order with only the required fields",
			req:    &AmendAdvancedOrderRequest{OrderID: "5755600460443882762", NewQuantity: 10, NewPrice: new(types.Number(0))},
			params: `{"order_id":"5755600460443882762","new_quantity":"10","new_price":"0"}`,
		},
		{
			name:   "an amendment by client order ID",
			req:    &AmendAdvancedOrderRequest{ClientOrderID: "c5f682ed-7108-4f1c-b755-972fcdca0f02", NewQuantity: 10, NewPrice: new(types.Number(0.0279))},
			params: `{"client_oid":"c5f682ed-7108-4f1c-b755-972fcdca0f02","new_quantity":"10","new_price":"0.0279"}`,
		},
		{
			name:   "an amendment with every field",
			req:    &AmendAdvancedOrderRequest{OrderID: "5755600460443882762", ClientOrderID: "c5f682ed-7108-4f1c-b755-972fcdca0f02", NewQuantity: 12, NewPrice: new(types.Number(0.0278)), NewReferencePrice: 0.0279},
			params: `{"order_id":"5755600460443882762","client_oid":"c5f682ed-7108-4f1c-b755-972fcdca0f02","new_quantity":"12","new_price":"0.0278","new_ref_price":"0.0279"}`,
		},
	})

	req := &AmendAdvancedOrderRequest{OrderID: "5755600460443882762", NewQuantity: 10, NewPrice: new(types.Number(0.0278)), NewReferencePrice: 0.0279}
	_, err := newResponseServerExchange(t, `{"id":1,"method":"private/advanced/amend-order","code":0,"result":null}`).AmendAdvancedOrder(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "AmendAdvancedOrder should error for a null result")
	_, err = newUnavailableServerExchange(t).AmendAdvancedOrder(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "AmendAdvancedOrder should error for an unavailable server")

	skipLiveUnlessCanManipulate(t)
	result, err := e.AmendAdvancedOrder(t.Context(), req)
	require.NoError(t, err, "AmendAdvancedOrder must not error")
	if mockTests {
		exp := &OrderIDResponse{OrderID: "5755600460443882762", ClientOrderID: "c5f682ed-7108-4f1c-b755-972fcdca0f02"}
		assert.Equal(t, exp, result, "AmendAdvancedOrder should decode every field")
	} else {
		assert.NotEmpty(t, result.OrderID, "AmendAdvancedOrder should return the order ID")
	}
}

func TestCreateOCO(t *testing.T) {
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t)
	for _, tc := range ocoRefusals() {
		_, err := refuser.CreateOCO(t.Context(), tc.orders)
		assert.ErrorIsf(t, err, tc.err, "CreateOCO should refuse %s", tc.name)
	}

	testAdvancedOrderParams(t, "private/advanced/create-oco", `{"list_id":"6498090546073120100"}`, func(ex *Exchange, orders []AdvancedOrderLegRequest) error {
		_, err := ex.CreateOCO(t.Context(), orders)
		return err
	}, []advancedParamsCase[[]AdvancedOrderLegRequest]{
		{
			name:   "an OCO with only the required fields",
			req:    ocoOrders(),
			params: `{"order_list":[{"instrument_name":"BTC_USD","side":"SELL","type":"LIMIT","quantity":"0.0001","price":"120000"},{"instrument_name":"BTC_USD","side":"SELL","type":"STOP_LOSS","quantity":"0.0001","ref_price":"40000"}]}`,
		},
		{
			name: "a margin OCO with every option a spot order takes",
			req: []AdvancedOrderLegRequest{
				{
					InstrumentName:                 "BTC_USD",
					Side:                           SideSell,
					OrderType:                      OrderTypeLimit,
					Quantity:                       0.1,
					Price:                          93000,
					ClientOrderID:                  "oco-take-profit",
					TimeInForce:                    TimeInForceGoodTillCancel,
					ExecutionInstructions:          []string{ExecutionInstructionPostOnly},
					FeeInstrumentName:              currency.CRO,
					SelfTradePreventionID:          "7",
					SelfTradePreventionInstruction: SelfTradePreventionCancelTaker,
					SelfTradePreventionScope:       SelfTradePreventionScopeSubAccount,
					SpotMargin:                     SpotMarginMargin,
				},
				{
					InstrumentName:     "BTC_USD",
					Side:               SideSell,
					OrderType:          OrderTypeStopLimit,
					Quantity:           0.1,
					Price:              79900,
					ReferencePrice:     80000,
					ReferencePriceType: ReferencePriceTypeMarkPrice,
					ClientOrderID:      "oco-stop-loss",
					SpotMargin:         SpotMarginMargin,
				},
			},
			params: `{"order_list":[{"instrument_name":"BTC_USD","side":"SELL","type":"LIMIT","quantity":"0.1","price":"93000","client_oid":"oco-take-profit","time_in_force":"GOOD_TILL_CANCEL","exec_inst":["POST_ONLY"],"fee_instrument_name":"CRO","stp_id":"7","stp_inst":"T","stp_scope":"S","spot_margin":"MARGIN"},{"instrument_name":"BTC_USD","side":"SELL","type":"STOP_LIMIT","quantity":"0.1","price":"79900","ref_price":"80000","ref_price_type":"MARK_PRICE","client_oid":"oco-stop-loss","spot_margin":"MARGIN"}]}`,
		},
		{
			name: "an OCO buying below the market or on a breakout, the STOP_LOSS buy spending a notional",
			req: []AdvancedOrderLegRequest{
				{InstrumentName: "BTC_USD", Side: SideBuy, OrderType: OrderTypeLimit, Quantity: 0.1, Price: 78000},
				{InstrumentName: "BTC_USD", Side: SideBuy, OrderType: OrderTypeStopLoss, Notional: 8300, ReferencePrice: 83000},
			},
			params: `{"order_list":[{"instrument_name":"BTC_USD","side":"BUY","type":"LIMIT","quantity":"0.1","price":"78000"},{"instrument_name":"BTC_USD","side":"BUY","type":"STOP_LOSS","notional":"8300","ref_price":"83000"}]}`,
		},
	})

	_, err := newResponseServerExchange(t, `{"id":1,"method":"private/advanced/create-oco","code":0,"result":null}`).CreateOCO(t.Context(), ocoOrders())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateOCO should error for a null result")
	_, err = newUnavailableServerExchange(t).CreateOCO(t.Context(), ocoOrders())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateOCO should error for an unavailable server")

	skipLiveUnlessCanManipulate(t)
	result, err := e.CreateOCO(t.Context(), ocoOrders())
	require.NoError(t, err, "CreateOCO must not error")
	if mockTests {
		exp := &AdvancedOrderListIDResponse{ListID: "6498090546073120100"}
		assert.Equal(t, exp, result, "CreateOCO should decode every field")
	} else {
		assert.NotEmpty(t, result.ListID, "CreateOCO should return the list ID")
	}
}

func TestCancelOCO(t *testing.T) {
	t.Parallel()
	testCancelContingency(t, "CancelOCO", "private/advanced/cancel-oco", "6498090546073120100", (*Exchange).CancelOCO)
}

func TestCreateOTO(t *testing.T) {
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t)
	for _, tc := range otoRefusals() {
		_, err := refuser.CreateOTO(t.Context(), tc.orders)
		assert.ErrorIsf(t, err, tc.err, "CreateOTO should refuse %s", tc.name)
	}

	testAdvancedOrderParams(t, "private/advanced/create-oto", `{"list_id":"6498090546073120199"}`, func(ex *Exchange, orders []AdvancedOrderLegRequest) error {
		_, err := ex.CreateOTO(t.Context(), orders)
		return err
	}, []advancedParamsCase[[]AdvancedOrderLegRequest]{
		{
			name:   "an OTO with only the required fields",
			req:    otoOrders(),
			params: `{"order_list":[{"instrument_name":"BTC_USD","side":"BUY","type":"LIMIT","quantity":"0.0001","price":"40000"},{"instrument_name":"BTC_USD","side":"SELL","type":"STOP_LOSS","quantity":"0.0001","ref_price":"30000"}]}`,
		},
		{
			name: "an OTO of a MARKET buy spending a notional and a TAKE_PROFIT_LIMIT comparing the last price",
			req: []AdvancedOrderLegRequest{
				{InstrumentName: "BTC_USD", Side: SideBuy, OrderType: OrderTypeMarket, Notional: 8000},
				{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeTakeProfitLimit, Quantity: 0.1, Price: 84000, ReferencePrice: 84100, ReferencePriceType: ReferencePriceTypeLastPrice},
			},
			params: `{"order_list":[{"instrument_name":"BTC_USD","side":"BUY","type":"MARKET","notional":"8000"},{"instrument_name":"BTC_USD","side":"SELL","type":"TAKE_PROFIT_LIMIT","quantity":"0.1","price":"84000","ref_price":"84100","ref_price_type":"LAST_PRICE"}]}`,
		},
		{
			name: "a short OTO whose TAKE_PROFIT buy spends a notional",
			req: []AdvancedOrderLegRequest{
				{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeLimit, Quantity: 0.1, Price: 83000},
				{InstrumentName: "BTC_USD", Side: SideBuy, OrderType: OrderTypeTakeProfit, Notional: 7600, ReferencePrice: 76000},
			},
			params: `{"order_list":[{"instrument_name":"BTC_USD","side":"SELL","type":"LIMIT","quantity":"0.1","price":"83000"},{"instrument_name":"BTC_USD","side":"BUY","type":"TAKE_PROFIT","notional":"7600","ref_price":"76000"}]}`,
		},
		{
			name: "a DERIV_ATTACH OTO on an isolated position, with every option",
			req: []AdvancedOrderLegRequest{
				{
					InstrumentName:                 "BTCUSD-PERP",
					Side:                           SideBuy,
					OrderType:                      OrderTypeLimit,
					Quantity:                       0.01,
					Price:                          80000,
					ClientOrderID:                  "perp-entry",
					TimeInForce:                    TimeInForceGoodTillCancel,
					ExecutionInstructions:          []string{ExecutionInstructionIsolatedMargin, ExecutionInstructionPostOnly},
					FeeInstrumentName:              currency.USD,
					SelfTradePreventionID:          "32767",
					SelfTradePreventionInstruction: SelfTradePreventionCancelMaker,
					SelfTradePreventionScope:       SelfTradePreventionScopeMasterOrSubAccount,
					IsolationID:                    "5755600460443891277",
					Leverage:                       10,
					IsolatedMarginAmount:           80,
					AttachOrderID:                  "5755600460443891201",
				},
				{InstrumentName: "BTCUSD-PERP", Side: SideSell, OrderType: OrderTypeStopLoss, Quantity: 0.01, ReferencePrice: 76000, ReferencePriceType: ReferencePriceTypeIndexPrice, ClientOrderID: "perp-stop-loss"},
			},
			params: `{"order_list":[{"instrument_name":"BTCUSD-PERP","side":"BUY","type":"LIMIT","quantity":"0.01","price":"80000","client_oid":"perp-entry","time_in_force":"GOOD_TILL_CANCEL","exec_inst":["ISOLATED_MARGIN","POST_ONLY"],"fee_instrument_name":"USD","stp_id":"32767","stp_inst":"M","stp_scope":"M","isolation_id":"5755600460443891277","leverage":"10","isolated_margin_amount":"80","attach_order_id":"5755600460443891201"},{"instrument_name":"BTCUSD-PERP","side":"SELL","type":"STOP_LOSS","quantity":"0.01","ref_price":"76000","ref_price_type":"INDEX_PRICE","client_oid":"perp-stop-loss"}]}`,
		},
	})

	_, err := newResponseServerExchange(t, `{"id":1,"method":"private/advanced/create-oto","code":0,"result":null}`).CreateOTO(t.Context(), otoOrders())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateOTO should error for a null result")
	_, err = newUnavailableServerExchange(t).CreateOTO(t.Context(), otoOrders())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateOTO should error for an unavailable server")

	skipLiveUnlessCanManipulate(t)
	result, err := e.CreateOTO(t.Context(), otoOrders())
	require.NoError(t, err, "CreateOTO must not error")
	if mockTests {
		exp := &AdvancedOrderListIDResponse{ListID: "6498090546073120199"}
		assert.Equal(t, exp, result, "CreateOTO should decode every field")
	} else {
		assert.NotEmpty(t, result.ListID, "CreateOTO should return the list ID")
	}
}

func TestCancelOTO(t *testing.T) {
	t.Parallel()
	testCancelContingency(t, "CancelOTO", "private/advanced/cancel-oto", "6498090546073120199", (*Exchange).CancelOTO)
}

func TestCreateOTOCO(t *testing.T) {
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t)
	for _, tc := range otocoRefusals() {
		_, err := refuser.CreateOTOCO(t.Context(), tc.orders)
		assert.ErrorIsf(t, err, tc.err, "CreateOTOCO should refuse %s", tc.name)
	}

	testAdvancedOrderParams(t, "private/advanced/create-otoco", `{"list_id":"6498090546073120243"}`, func(ex *Exchange, orders []AdvancedOrderLegRequest) error {
		_, err := ex.CreateOTOCO(t.Context(), orders)
		return err
	}, []advancedParamsCase[[]AdvancedOrderLegRequest]{
		{
			name:   "an OTOCO with only the required fields",
			req:    otocoOrders(),
			params: `{"order_list":[{"instrument_name":"BTC_USD","side":"BUY","type":"LIMIT","quantity":"0.0001","price":"40000"},{"instrument_name":"BTC_USD","side":"SELL","type":"STOP_LOSS","quantity":"0.0001","ref_price":"30000"},{"instrument_name":"BTC_USD","side":"SELL","type":"TAKE_PROFIT","quantity":"0.0001","ref_price":"120000"}]}`,
		},
		{
			name: "an OTOCO of a MARKET buy, a STOP_LIMIT and a TAKE_PROFIT_LIMIT, with client order IDs and reference price types",
			req: []AdvancedOrderLegRequest{
				{InstrumentName: "BTC_USD", Side: SideBuy, OrderType: OrderTypeMarket, Quantity: 0.1, ClientOrderID: "entry"},
				{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeStopLimit, Quantity: 0.1, Price: 79900, ReferencePrice: 80000, ReferencePriceType: ReferencePriceTypeIndexPrice, ClientOrderID: "stop-loss"},
				{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeTakeProfitLimit, Quantity: 0.1, Price: 108000, ReferencePrice: 107900, ReferencePriceType: ReferencePriceTypeLastPrice, ClientOrderID: "take-profit"},
			},
			params: `{"order_list":[{"instrument_name":"BTC_USD","side":"BUY","type":"MARKET","quantity":"0.1","client_oid":"entry"},{"instrument_name":"BTC_USD","side":"SELL","type":"STOP_LIMIT","quantity":"0.1","price":"79900","ref_price":"80000","ref_price_type":"INDEX_PRICE","client_oid":"stop-loss"},{"instrument_name":"BTC_USD","side":"SELL","type":"TAKE_PROFIT_LIMIT","quantity":"0.1","price":"108000","ref_price":"107900","ref_price_type":"LAST_PRICE","client_oid":"take-profit"}]}`,
		},
	})

	_, err := newResponseServerExchange(t, `{"id":1,"method":"private/advanced/create-otoco","code":0,"result":null}`).CreateOTOCO(t.Context(), otocoOrders())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateOTOCO should error for a null result")
	_, err = newUnavailableServerExchange(t).CreateOTOCO(t.Context(), otocoOrders())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateOTOCO should error for an unavailable server")

	skipLiveUnlessCanManipulate(t)
	result, err := e.CreateOTOCO(t.Context(), otocoOrders())
	require.NoError(t, err, "CreateOTOCO must not error")
	if mockTests {
		exp := &AdvancedOrderListIDResponse{ListID: "6498090546073120243"}
		assert.Equal(t, exp, result, "CreateOTOCO should decode every field")
	} else {
		assert.NotEmpty(t, result.ListID, "CreateOTOCO should return the list ID")
	}
}

func TestCancelOTOCO(t *testing.T) {
	t.Parallel()
	testCancelContingency(t, "CancelOTOCO", "private/advanced/cancel-otoco", "6498090546073120243", (*Exchange).CancelOTOCO)
}

func TestCancelAdvancedOrder(t *testing.T) {
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t)
	_, err := refuser.CancelAdvancedOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelAdvancedOrder must refuse a nil request")
	_, err = refuser.CancelAdvancedOrder(t.Context(), &AdvancedOrderIDRequest{})
	require.ErrorIs(t, err, errOrderIDRequired, "CancelAdvancedOrder must refuse a request without an order ID or client order ID")

	testAdvancedOrderParams(t, "private/advanced/cancel-order", advancedOrderIDResult, func(ex *Exchange, req *AdvancedOrderIDRequest) error {
		_, err := ex.CancelAdvancedOrder(t.Context(), req)
		return err
	}, []advancedParamsCase[*AdvancedOrderIDRequest]{
		{name: "an order ID", req: &AdvancedOrderIDRequest{OrderID: "5755600460443882762"}, params: `{"order_id":"5755600460443882762"}`},
		{name: "a client order ID", req: &AdvancedOrderIDRequest{ClientOrderID: "c5f682ed-7108-4f1c-b755-972fcdca0f02"}, params: `{"client_oid":"c5f682ed-7108-4f1c-b755-972fcdca0f02"}`},
		{
			name:   "an order ID and a client order ID",
			req:    &AdvancedOrderIDRequest{OrderID: "5755600460443882762", ClientOrderID: "c5f682ed-7108-4f1c-b755-972fcdca0f02"},
			params: `{"order_id":"5755600460443882762","client_oid":"c5f682ed-7108-4f1c-b755-972fcdca0f02"}`,
		},
	})

	req := &AdvancedOrderIDRequest{OrderID: "5755600460443882762"}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/advanced/cancel-order","code":0,"result":null}`).CancelAdvancedOrder(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelAdvancedOrder should error for a null result")
	_, err = newUnavailableServerExchange(t).CancelAdvancedOrder(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelAdvancedOrder should error for an unavailable server")

	skipLiveUnlessCanManipulate(t)
	result, err := e.CancelAdvancedOrder(t.Context(), req)
	require.NoError(t, err, "CancelAdvancedOrder must not error")
	if mockTests {
		exp := &OrderIDResponse{OrderID: "5755600460443882762", ClientOrderID: "c5f682ed-7108-4f1c-b755-972fcdca0f02"}
		assert.Equal(t, exp, result, "CancelAdvancedOrder should decode every field")
	} else {
		assert.NotEmpty(t, result.OrderID, "CancelAdvancedOrder should return the order ID")
	}
}

// cancelAllAdvancedOrdersRefusals are the cancel-all requests CancelAllAdvancedOrders and its websocket form refuse
// before sending, and the errors they refuse them with
var cancelAllAdvancedOrdersRefusals = []struct {
	name string
	req  *CancelAllAdvancedOrdersRequest
	err  error
}{
	{name: "a nil request", err: common.ErrNilPointer},
	{name: "an unknown order type", req: &CancelAllAdvancedOrdersRequest{OrderType: "BOTH"}, err: errInvalidCancelOrderType},
	{name: "an unknown position type", req: &CancelAllAdvancedOrdersRequest{InstrumentName: "BTCUSD-PERP", PositionType: "PORTFOLIO_MARGIN"}, err: errInvalidPositionType},
	{name: "a position type without an instrument", req: &CancelAllAdvancedOrdersRequest{PositionType: PositionTypeCrossMargin}, err: errInstrumentNameRequired},
	{name: "a position type with the LIMIT order type", req: &CancelAllAdvancedOrdersRequest{InstrumentName: "BTCUSD-PERP", OrderType: CancelOrderTypeLimit, PositionType: PositionTypeIsolatedMargin}, err: errPositionTypeCancelOrderType},
	{name: "a position type with the TRIGGER order type", req: &CancelAllAdvancedOrdersRequest{InstrumentName: "BTCUSD-PERP", OrderType: CancelOrderTypeTrigger, PositionType: PositionTypeCrossMargin}, err: errPositionTypeCancelOrderType},
}

func TestCancelAllAdvancedOrders(t *testing.T) {
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t)
	for _, tc := range cancelAllAdvancedOrdersRefusals {
		assert.ErrorIsf(t, refuser.CancelAllAdvancedOrders(t.Context(), tc.req), tc.err, "CancelAllAdvancedOrders should refuse %s", tc.name)
	}

	testAdvancedOrderParams(t, "private/advanced/cancel-all-orders", "", func(ex *Exchange, req *CancelAllAdvancedOrdersRequest) error {
		return ex.CancelAllAdvancedOrders(t.Context(), req)
	}, []advancedParamsCase[*CancelAllAdvancedOrdersRequest]{
		{name: "a cancel of every instrument's advanced orders", req: &CancelAllAdvancedOrdersRequest{}, params: `{}`},
		{name: "a cancel of an instrument's trigger orders", req: &CancelAllAdvancedOrdersRequest{InstrumentName: "BTC_USD", OrderType: CancelOrderTypeTrigger}, params: `{"instrument_name":"BTC_USD","type":"TRIGGER"}`},
		{name: "a cancel of a cross margin position's orders", req: &CancelAllAdvancedOrdersRequest{InstrumentName: "BTCUSD-PERP", PositionType: PositionTypeCrossMargin}, params: `{"instrument_name":"BTCUSD-PERP","position_type":"CROSS_MARGIN"}`},
		{
			name:   "a cancel of every order of an isolated position, with every field",
			req:    &CancelAllAdvancedOrdersRequest{InstrumentName: "BTCUSD-PERP", OrderType: CancelOrderTypeAll, PositionType: PositionTypeIsolatedMargin},
			params: `{"instrument_name":"BTCUSD-PERP","type":"ALL","position_type":"ISOLATED_MARGIN"}`,
		},
	})

	req := &CancelAllAdvancedOrdersRequest{InstrumentName: "BTC_USD"}
	assert.ErrorIs(t, newUnavailableServerExchange(t).CancelAllAdvancedOrders(t.Context(), req), request.ErrBadStatus, "CancelAllAdvancedOrders should error for an unavailable server")

	skipLiveUnlessCanManipulate(t)
	require.NoError(t, e.CancelAllAdvancedOrders(t.Context(), req), "CancelAllAdvancedOrders must not error")
}

func TestGetAdvancedOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := newAdvancedRefusalExchange(t).GetAdvancedOpenOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAdvancedOpenOrders must refuse a nil request")

	testAdvancedOrderParams(t, "private/advanced/get-open-orders", `{"data":[]}`, func(ex *Exchange, req *GetAdvancedOpenOrdersRequest) error {
		_, err := ex.GetAdvancedOpenOrders(t.Context(), req)
		return err
	}, []advancedParamsCase[*GetAdvancedOpenOrdersRequest]{
		{name: "a request for every instrument's open orders", req: &GetAdvancedOpenOrdersRequest{}, params: `{}`},
		{name: "a request for an instrument's open orders", req: &GetAdvancedOpenOrdersRequest{InstrumentName: "BTC_USD"}, params: `{"instrument_name":"BTC_USD"}`},
	})

	req := &GetAdvancedOpenOrdersRequest{InstrumentName: "BTC_USD"}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/advanced/get-open-orders","code":0,"result":null}`).GetAdvancedOpenOrders(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetAdvancedOpenOrders should error for a null result")
	_, err = newUnavailableServerExchange(t).GetAdvancedOpenOrders(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAdvancedOpenOrders should error for an unavailable server")

	skipLiveWithoutCredentials(t)
	result, err := e.GetAdvancedOpenOrders(t.Context(), req)
	require.NoError(t, err, "GetAdvancedOpenOrders must not error")
	if mockTests {
		exp := []OrderResponse{
			{
				AccountID:             "5207a638-b79a-4c13-9d41-323be7556c17",
				OrderID:               "5755600460443890101",
				AveragePrice:          preciseNumber("0.0"),
				CumulativeQuantity:    preciseNumber("0.0000"),
				CumulativeValue:       preciseNumber("0.000000"),
				CumulativeFee:         preciseNumber("0.000000"),
				ClientOrderID:         "1792836351900",
				OrderType:             OrderTypeLimit,
				TimeInForce:           TimeInForceGoodTillCancel,
				Side:                  SideBuy,
				ExecutionInstructions: []string{},
				Quantity:              preciseNumber("0.0100"),
				LimitPrice:            preciseNumber("78000.0"),
				OrderValue:            preciseNumber("780.000000"),
				MakerFeeRate:          0.00025,
				TakerFeeRate:          0.0004,
				Status:                OrderStatusActive,
				UpdateUserID:          "5207a638-b79a-4c13-9d41-323be7556c17",
				OrderDate:             "2026-10-24",
				InstrumentName:        "BTC_USD",
				FeeInstrumentName:     currency.USD,
				CreateTime:            types.Time(time.UnixMilli(1792836351902)),
				CreateTimeNano:        types.Time(time.Unix(0, 1792836351902118356)),
				UpdateTime:            types.Time(time.UnixMilli(1792836351902)),
				ListID:                "6498090546073120187",
				ContingencyType:       ContingencyTypeSpotAttach,
				LegID:                 "1",
				ExchangeOrderID:       "5755600460443890177",
			},
			{
				AccountID:             "5207a638-b79a-4c13-9d41-323be7556c17",
				OrderID:               "5755600460443890102",
				AveragePrice:          preciseNumber("0.0"),
				CumulativeQuantity:    preciseNumber("0.0000"),
				CumulativeValue:       preciseNumber("0.000000"),
				CumulativeFee:         preciseNumber("0.000000"),
				ClientOrderID:         "1792836351901",
				OrderType:             OrderTypeStopLoss,
				TimeInForce:           TimeInForceGoodTillCancel,
				Side:                  SideSell,
				ExecutionInstructions: []string{},
				Quantity:              preciseNumber("0.0100"),
				MakerFeeRate:          0.00025,
				TakerFeeRate:          0.0004,
				Status:                OrderStatusActive,
				UpdateUserID:          "5207a638-b79a-4c13-9d41-323be7556c17",
				OrderDate:             "2026-10-24",
				InstrumentName:        "BTC_USD",
				FeeInstrumentName:     currency.USD,
				CreateTime:            types.Time(time.UnixMilli(1792836351902)),
				CreateTimeNano:        types.Time(time.Unix(0, 1792836351902118356)),
				UpdateTime:            types.Time(time.UnixMilli(1792836351902)),
				ListID:                "6498090546073120187",
				ContingencyType:       ContingencyTypeSpotAttach,
				LegID:                 "2",
				ReferencePrice:        preciseNumber("74000.00"),
				ReferencePriceType:    ReferencePriceTypeMarkPrice,
			},
			{
				AccountID:             "5207a638-b79a-4c13-9d41-323be7556c17",
				OrderID:               "5755600460443890103",
				AveragePrice:          preciseNumber("0.0"),
				CumulativeQuantity:    preciseNumber("0.0000"),
				CumulativeValue:       preciseNumber("0.000000"),
				CumulativeFee:         preciseNumber("0.000000"),
				ClientOrderID:         "1792836351902",
				OrderType:             OrderTypeTakeProfit,
				TimeInForce:           TimeInForceGoodTillCancel,
				Side:                  SideSell,
				ExecutionInstructions: []string{},
				Quantity:              preciseNumber("0.0100"),
				MakerFeeRate:          0.00025,
				TakerFeeRate:          0.0004,
				Status:                OrderStatusActive,
				UpdateUserID:          "5207a638-b79a-4c13-9d41-323be7556c17",
				OrderDate:             "2026-10-24",
				InstrumentName:        "BTC_USD",
				FeeInstrumentName:     currency.USD,
				CreateTime:            types.Time(time.UnixMilli(1792836351902)),
				CreateTimeNano:        types.Time(time.Unix(0, 1792836351902118356)),
				UpdateTime:            types.Time(time.UnixMilli(1792836351902)),
				ListID:                "6498090546073120187",
				ContingencyType:       ContingencyTypeSpotAttach,
				LegID:                 "3",
				ReferencePrice:        preciseNumber("82000.00"),
				ReferencePriceType:    ReferencePriceTypeMarkPrice,
			},
		}
		assert.Equal(t, exp, result, "GetAdvancedOpenOrders should decode every field")
	} else {
		for i := range result {
			assert.NotEmptyf(t, result[i].OrderID, "GetAdvancedOpenOrders should return order %d's ID", i)
		}
	}
}

func TestGetAdvancedOrderDetail(t *testing.T) {
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t)
	_, err := refuser.GetAdvancedOrderDetail(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAdvancedOrderDetail must refuse a nil request")
	_, err = refuser.GetAdvancedOrderDetail(t.Context(), &AdvancedOrderIDRequest{})
	require.ErrorIs(t, err, errOrderIDRequired, "GetAdvancedOrderDetail must refuse a request without an order ID or client order ID")

	testAdvancedOrderParams(t, "private/advanced/get-order-detail", `{"order_id":"5755600460443891201"}`, func(ex *Exchange, req *AdvancedOrderIDRequest) error {
		_, err := ex.GetAdvancedOrderDetail(t.Context(), req)
		return err
	}, []advancedParamsCase[*AdvancedOrderIDRequest]{
		{name: "an order ID", req: &AdvancedOrderIDRequest{OrderID: "5755600460443891201"}, params: `{"order_id":"5755600460443891201"}`},
		{name: "a client order ID", req: &AdvancedOrderIDRequest{ClientOrderID: "1792830013640"}, params: `{"client_oid":"1792830013640"}`},
		{name: "an order ID and a client order ID", req: &AdvancedOrderIDRequest{OrderID: "5755600460443891201", ClientOrderID: "1792830013640"}, params: `{"order_id":"5755600460443891201","client_oid":"1792830013640"}`},
	})

	req := &AdvancedOrderIDRequest{OrderID: "5755600460443891201"}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/advanced/get-order-detail","code":0,"result":null}`).GetAdvancedOrderDetail(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetAdvancedOrderDetail should error for a null result")
	_, err = newUnavailableServerExchange(t).GetAdvancedOrderDetail(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAdvancedOrderDetail should error for an unavailable server")

	skipLiveWithoutCredentials(t)
	if !mockTests {
		history, err := e.GetAdvancedOrderHistory(t.Context(), &GetAdvancedOrderHistoryRequest{})
		require.NoError(t, err, "GetAdvancedOrderHistory must not error")
		if len(history) == 0 {
			t.Skip("the account has no advanced order in the past day to look up")
		}
		req.OrderID = history[0].OrderID
	}
	result, err := e.GetAdvancedOrderDetail(t.Context(), req)
	require.NoError(t, err, "GetAdvancedOrderDetail must not error")
	if mockTests {
		exp := &OrderResponse{
			AccountID:             "5207a638-b79a-4c13-9d41-323be7556c17",
			OrderID:               "5755600460443891201",
			ClientOrderID:         "1792830013640",
			OrderType:             OrderTypeLimit,
			TimeInForce:           TimeInForceGoodTillCancel,
			Side:                  SideBuy,
			ExecutionInstructions: []string{ExecutionInstructionIsolatedMargin},
			Quantity:              preciseNumber("0.0100"),
			LimitPrice:            preciseNumber("80000.0"),
			OrderValue:            preciseNumber("800.000000"),
			MakerFeeRate:          0.00025,
			TakerFeeRate:          0.0004,
			AveragePrice:          preciseNumber("80000.0"),
			CumulativeQuantity:    preciseNumber("0.0040"),
			CumulativeValue:       preciseNumber("320.000000"),
			CumulativeFee:         preciseNumber("0.080000"),
			Status:                OrderStatusActive,
			UpdateUserID:          "5207a638-b79a-4c13-9d41-323be7556c17",
			OrderDate:             "2026-10-24",
			InstrumentName:        "BTCUSD-PERP",
			FeeInstrumentName:     currency.USD,
			CreateTime:            types.Time(time.UnixMilli(1792830013645)),
			CreateTimeNano:        types.Time(time.Unix(0, 1792830013645371920)),
			UpdateTime:            types.Time(time.UnixMilli(1792836390118)),
			IsolationID:           "5755600460443891277",
			IsolationType:         IsolationTypeIsolatedMargin,
			ListID:                "6498090546073120314",
			ContingencyType:       ContingencyTypeDerivAttach,
			LegID:                 "1",
			ExchangeOrderID:       "5755600460443891277",
			Leverage:              10,
			IsolatedMarginAmount:  preciseNumber("80"),
		}
		assert.Equal(t, exp, result, "GetAdvancedOrderDetail should decode every field")
	} else {
		assert.Equal(t, req.OrderID, result.OrderID, "GetAdvancedOrderDetail should return the order asked for")
	}
}

func TestGetAdvancedOrderHistory(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.Unix(0, 1792713600000081486), time.Unix(0, 1792799991058211357)
	refuser := newAdvancedRefusalExchange(t)
	for _, tc := range []struct {
		name string
		req  *GetAdvancedOrderHistoryRequest
		err  error
	}{
		{name: "a nil request", err: common.ErrNilPointer},
		{name: "a start time after the end time", req: &GetAdvancedOrderHistoryRequest{StartTime: endTime, EndTime: startTime}, err: common.ErrStartAfterEnd},
		{name: "a limit above 100", req: &GetAdvancedOrderHistoryRequest{Limit: 101}, err: errInvalidCount},
		{name: "a start time before 1970", req: &GetAdvancedOrderHistoryRequest{StartTime: time.Unix(0, -1)}, err: errTimeBeforeUnixEpoch},
		{name: "an end time before 1970", req: &GetAdvancedOrderHistoryRequest{EndTime: time.Unix(0, -1)}, err: errTimeBeforeUnixEpoch},
	} {
		_, err := refuser.GetAdvancedOrderHistory(t.Context(), tc.req)
		assert.ErrorIsf(t, err, tc.err, "GetAdvancedOrderHistory should refuse %s", tc.name)
	}

	testAdvancedOrderParams(t, "private/advanced/get-order-history", `{"data":[]}`, func(ex *Exchange, req *GetAdvancedOrderHistoryRequest) error {
		_, err := ex.GetAdvancedOrderHistory(t.Context(), req)
		return err
	}, []advancedParamsCase[*GetAdvancedOrderHistoryRequest]{
		{name: "a request with Crypto.com's defaults", req: &GetAdvancedOrderHistoryRequest{}, params: `{}`},
		{
			name:   "a request with every field",
			req:    &GetAdvancedOrderHistoryRequest{InstrumentName: "BTCUSD-PERP", StartTime: startTime, EndTime: endTime, Limit: 100},
			params: `{"instrument_name":"BTCUSD-PERP","start_time":"1792713600000081486","end_time":"1792799991058211357","limit":"100"}`,
		},
	})

	req := &GetAdvancedOrderHistoryRequest{StartTime: startTime, EndTime: endTime, Limit: 20}
	_, err := newResponseServerExchange(t, `{"id":1,"method":"private/advanced/get-order-history","code":0,"result":null}`).GetAdvancedOrderHistory(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetAdvancedOrderHistory should error for a null result")
	_, err = newUnavailableServerExchange(t).GetAdvancedOrderHistory(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAdvancedOrderHistory should error for an unavailable server")

	skipLiveWithoutCredentials(t)
	result, err := e.GetAdvancedOrderHistory(t.Context(), req)
	require.NoError(t, err, "GetAdvancedOrderHistory must not error")
	if mockTests {
		exp := []OrderResponse{
			{
				AccountID:             "5207a638-b79a-4c13-9d41-323be7556c17",
				OrderID:               "5755600460443880501",
				ClientOrderID:         "1792746942215",
				OrderType:             OrderTypeLimit,
				TimeInForce:           TimeInForceGoodTillCancel,
				Side:                  SideBuy,
				ExecutionInstructions: []string{},
				Quantity:              preciseNumber("0.0100"),
				LimitPrice:            preciseNumber("81500.0"),
				OrderValue:            preciseNumber("815.000000"),
				MakerFeeRate:          0.00025,
				TakerFeeRate:          0.0004,
				AveragePrice:          preciseNumber("81500.0"),
				CumulativeQuantity:    preciseNumber("0.0100"),
				CumulativeValue:       preciseNumber("815.000000"),
				CumulativeFee:         preciseNumber("0.203750"),
				Status:                OrderStatusFilled,
				UpdateUserID:          "5207a638-b79a-4c13-9d41-323be7556c17",
				OrderDate:             "2026-10-23",
				InstrumentName:        "BTC_USD",
				FeeInstrumentName:     currency.USD,
				CreateTime:            types.Time(time.UnixMilli(1792746942217)),
				CreateTimeNano:        types.Time(time.Unix(0, 1792746942217123456)),
				UpdateTime:            types.Time(time.UnixMilli(1792753328904)),
				ListID:                "6498090546073120152",
				ContingencyType:       ContingencyTypeSpotAttach,
				LegID:                 "1",
				ExchangeOrderID:       "5755600460443880577",
			},
			{
				AccountID:             "5207a638-b79a-4c13-9d41-323be7556c17",
				OrderID:               "5755600460443880502",
				ClientOrderID:         "1792746942216",
				OrderType:             OrderTypeStopLoss,
				TimeInForce:           TimeInForceGoodTillCancel,
				Side:                  SideSell,
				ExecutionInstructions: []string{},
				Quantity:              preciseNumber("0.0100"),
				MakerFeeRate:          0.00025,
				TakerFeeRate:          0.0004,
				AveragePrice:          preciseNumber("78990.0"),
				CumulativeQuantity:    preciseNumber("0.0100"),
				CumulativeValue:       preciseNumber("789.900000"),
				CumulativeFee:         preciseNumber("0.315960"),
				Status:                OrderStatusFilled,
				UpdateUserID:          "5207a638-b79a-4c13-9d41-323be7556c17",
				OrderDate:             "2026-10-23",
				InstrumentName:        "BTC_USD",
				FeeInstrumentName:     currency.USD,
				CreateTime:            types.Time(time.UnixMilli(1792746942217)),
				CreateTimeNano:        types.Time(time.Unix(0, 1792746942217123456)),
				UpdateTime:            types.Time(time.UnixMilli(1792766851560)),
				ListID:                "6498090546073120152",
				ContingencyType:       ContingencyTypeSpotAttach,
				LegID:                 "2",
				ReferencePrice:        preciseNumber("79000.00"),
				ReferencePriceType:    ReferencePriceTypeMarkPrice,
				ExchangeOrderID:       "5755600460443880578",
			},
			{
				AccountID:             "5207a638-b79a-4c13-9d41-323be7556c17",
				OrderID:               "5755600460443880503",
				AveragePrice:          preciseNumber("0.0"),
				CumulativeQuantity:    preciseNumber("0.0000"),
				CumulativeValue:       preciseNumber("0.000000"),
				CumulativeFee:         preciseNumber("0.000000"),
				ClientOrderID:         "1792746942217",
				OrderType:             OrderTypeTakeProfit,
				TimeInForce:           TimeInForceGoodTillCancel,
				Side:                  SideSell,
				ExecutionInstructions: []string{},
				Quantity:              preciseNumber("0.0100"),
				MakerFeeRate:          0.00025,
				TakerFeeRate:          0.0004,
				Status:                OrderStatusCanceled,
				UpdateUserID:          "5207a638-b79a-4c13-9d41-323be7556c17",
				OrderDate:             "2026-10-23",
				InstrumentName:        "BTC_USD",
				FeeInstrumentName:     currency.USD,
				CreateTime:            types.Time(time.UnixMilli(1792746942217)),
				CreateTimeNano:        types.Time(time.Unix(0, 1792746942217123456)),
				UpdateTime:            types.Time(time.UnixMilli(1792766851560)),
				ListID:                "6498090546073120152",
				ContingencyType:       ContingencyTypeSpotAttach,
				LegID:                 "3",
				ReferencePrice:        preciseNumber("84000.00"),
				ReferencePriceType:    ReferencePriceTypeMarkPrice,
			},
			{
				AccountID:             "5207a638-b79a-4c13-9d41-323be7556c17",
				OrderID:               "5755600460443881909",
				AveragePrice:          preciseNumber("0.0"),
				CumulativeQuantity:    preciseNumber("0.0000"),
				CumulativeValue:       preciseNumber("0.000000"),
				CumulativeFee:         preciseNumber("0.000000"),
				ClientOrderID:         "1792787465330",
				OrderType:             OrderTypeStopLoss,
				TimeInForce:           TimeInForceGoodTillCancel,
				Side:                  SideSell,
				ExecutionInstructions: []string{ExecutionInstructionReduceOnly},
				Quantity:              preciseNumber("0.0100"),
				MakerFeeRate:          0.00025,
				TakerFeeRate:          0.0004,
				Status:                OrderStatusRejected,
				UpdateUserID:          "5207a638-b79a-4c13-9d41-323be7556c17",
				OrderDate:             "2026-10-23",
				InstrumentName:        "BTCUSD-PERP",
				FeeInstrumentName:     currency.USD,
				CreateTime:            types.Time(time.UnixMilli(1792787465338)),
				CreateTimeNano:        types.Time(time.Unix(0, 1792787465338604217)),
				UpdateTime:            types.Time(time.UnixMilli(1792787465338)),
				Reason:                "1110",
				ContingencyType:       ContingencyTypeDerivAttach,
				ReferencePrice:        preciseNumber("78500.0"),
				ReferencePriceType:    ReferencePriceTypeLastPrice,
				AttachIsolationID:     "0",
				RejectReason:          "REDUCE_ONLY_REJECTED",
			},
		}
		assert.Equal(t, exp, result, "GetAdvancedOrderHistory should decode every field")
	} else {
		for i := range result {
			assert.NotEmptyf(t, result[i].OrderID, "GetAdvancedOrderHistory should return order %d's ID", i)
		}
	}
}

// advancedOrderListElements returns a JSON array of count copies of element, the params an order list of count like
// orders sends
func advancedOrderListElements(element string, count int) string {
	return "[" + strings.Join(slices.Repeat([]string{element}, count), ",") + "]"
}

// advancedOrderListOrders returns the docs' example list of trigger orders resized and repriced away from the market, as
// the live tests that may change the account place it: a sell STOP_LOSS and a sell TAKE_PROFIT of 0.0001 BTC_USD, placed
// once the mark price falls to 40000 and once it rises to 120000, BTC_USD trading at 80700 on 2026-10-25
func advancedOrderListOrders() []CreateAdvancedOrderRequest {
	return []CreateAdvancedOrderRequest{
		{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeStopLoss, Quantity: 0.0001, ReferencePrice: 40000, ClientOrderID: "my_sl_order"},
		{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeTakeProfit, Quantity: 0.0001, ReferencePrice: 120000, ClientOrderID: "my_tp_order"},
	}
}

// createAdvancedOrderListRefusals returns the lists of trigger orders CreateAdvancedOrderList and its websocket form
// refuse before sending
func createAdvancedOrderListRefusals() []advancedListRefusal[CreateAdvancedOrderRequest] {
	order := advancedTriggerOrder()
	refusals := make([]advancedListRefusal[CreateAdvancedOrderRequest], 0, len(triggerOrderRefusals)+2)
	refusals = append(refusals,
		advancedListRefusal[CreateAdvancedOrderRequest]{name: "an empty list", err: errOrderListEmpty},
		advancedListRefusal[CreateAdvancedOrderRequest]{name: "a list of 11 orders", orders: slices.Repeat([]CreateAdvancedOrderRequest{order}, maxAdvancedOrderListLength+1), err: errOrderListTooLong},
	)
	for _, tc := range triggerOrderRefusals {
		refused := advancedTriggerOrder()
		tc.change(&refused)
		refusals = append(refusals, advancedListRefusal[CreateAdvancedOrderRequest]{name: "a list holding " + tc.name, orders: []CreateAdvancedOrderRequest{order, refused}, err: tc.err})
	}
	return refusals
}

// cancelAdvancedOrderListRefusals returns the lists of orders to cancel CancelAdvancedOrderList and its websocket form
// refuse before sending
func cancelAdvancedOrderListRefusals() []advancedListRefusal[AdvancedOrderIDRequest] {
	return []advancedListRefusal[AdvancedOrderIDRequest]{
		{name: "an empty list", err: errOrderListEmpty},
		{name: "a list of 11 orders", orders: slices.Repeat([]AdvancedOrderIDRequest{{OrderID: "289991951579667814"}}, maxAdvancedOrderListLength+1), err: errOrderListTooLong},
		{name: "a list holding an order without an order ID or client order ID", orders: []AdvancedOrderIDRequest{{OrderID: "289991951579667814"}, {}}, err: errOrderIDRequired},
	}
}

func TestCreateAdvancedOrderList(t *testing.T) {
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t)
	for _, tc := range createAdvancedOrderListRefusals() {
		_, err := refuser.CreateAdvancedOrderList(t.Context(), tc.orders)
		assert.ErrorIsf(t, err, tc.err, "CreateAdvancedOrderList should refuse %s", tc.name)
	}
	stopLoss := advancedOrderListOrders()[0]

	const stopLossParams = `{"instrument_name":"BTC_USD","side":"SELL","type":"STOP_LOSS","quantity":"0.0001","ref_price":"40000","client_oid":"my_sl_order"}`
	testAdvancedOrderParams(t, "private/advanced/create-order-list", `[{"index":0,"code":0,"order_id":"289991951579667814","client_oid":"my_sl_order"}]`, func(ex *Exchange, orders []CreateAdvancedOrderRequest) error {
		_, err := ex.CreateAdvancedOrderList(t.Context(), orders)
		return err
	}, []advancedParamsCase[[]CreateAdvancedOrderRequest]{
		{name: "a list of one order", req: []CreateAdvancedOrderRequest{stopLoss}, params: `{"order_list":[` + stopLossParams + `]}`},
		{
			name:   "a list of the docs' two orders",
			req:    advancedOrderListOrders(),
			params: `{"order_list":[` + stopLossParams + `,{"instrument_name":"BTC_USD","side":"SELL","type":"TAKE_PROFIT","quantity":"0.0001","ref_price":"120000","client_oid":"my_tp_order"}]}`,
		},
		{
			name:   "a list of 10 orders",
			req:    slices.Repeat([]CreateAdvancedOrderRequest{stopLoss}, maxAdvancedOrderListLength),
			params: `{"order_list":` + advancedOrderListElements(stopLossParams, maxAdvancedOrderListLength) + `}`,
		},
	})

	orders := advancedOrderListOrders()
	result, err := newResponseServerExchange(t, `{"id":1,"method":"private/advanced/create-order-list","code":0,"result":[{"index":0,"code":0,"order_id":"289991951579667814","client_oid":"my_sl_order"},{"index":1,"code":229,"message":"INVALID_REF_PRICE"}]}`).CreateAdvancedOrderList(t.Context(), orders)
	require.NoError(t, err, "CreateAdvancedOrderList must not error for a list with a refused order")
	expRefused := []OrderListEntryResponse{
		{OrderID: "289991951579667814", ClientOrderID: "my_sl_order"},
		{Index: 1, Code: 229, Message: "INVALID_REF_PRICE"},
	}
	assert.Equal(t, expRefused, result, "CreateAdvancedOrderList should decode a refused order's outcome")
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/advanced/create-order-list","code":0,"result":null}`).CreateAdvancedOrderList(t.Context(), orders)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateAdvancedOrderList should error for a null result")
	_, err = newUnavailableServerExchange(t).CreateAdvancedOrderList(t.Context(), orders)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateAdvancedOrderList should error for an unavailable server")

	skipLiveUnlessCanManipulate(t)
	result, err = e.CreateAdvancedOrderList(t.Context(), orders)
	require.NoError(t, err, "CreateAdvancedOrderList must not error")
	if mockTests {
		exp := []OrderListEntryResponse{
			{OrderID: "289991951579667814", ClientOrderID: "my_sl_order"},
			{Index: 1, OrderID: "289991951579667815", ClientOrderID: "my_tp_order"},
		}
		assert.Equal(t, exp, result, "CreateAdvancedOrderList should decode every field")
	} else {
		assert.Len(t, result, len(orders), "CreateAdvancedOrderList should return each order's outcome")
	}
}

func TestCancelAdvancedOrderList(t *testing.T) {
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t)
	for _, tc := range cancelAdvancedOrderListRefusals() {
		_, err := refuser.CancelAdvancedOrderList(t.Context(), tc.orders)
		assert.ErrorIsf(t, err, tc.err, "CancelAdvancedOrderList should refuse %s", tc.name)
	}

	testAdvancedOrderParams(t, "private/advanced/cancel-order-list", `[{"index":0,"code":0}]`, func(ex *Exchange, orders []AdvancedOrderIDRequest) error {
		_, err := ex.CancelAdvancedOrderList(t.Context(), orders)
		return err
	}, []advancedParamsCase[[]AdvancedOrderIDRequest]{
		{name: "a list of one order ID", req: []AdvancedOrderIDRequest{{OrderID: "289991951579667814"}}, params: `{"order_list":[{"order_id":"289991951579667814"}]}`},
		{
			name:   "a list of an order ID and client order ID, and a client order ID",
			req:    []AdvancedOrderIDRequest{{OrderID: "289991951579667814", ClientOrderID: "my_sl_order"}, {ClientOrderID: "my_tp_order"}},
			params: `{"order_list":[{"order_id":"289991951579667814","client_oid":"my_sl_order"},{"client_oid":"my_tp_order"}]}`,
		},
		{
			name:   "a list of 10 orders",
			req:    slices.Repeat([]AdvancedOrderIDRequest{{ClientOrderID: "my_tp_order"}}, maxAdvancedOrderListLength),
			params: `{"order_list":` + advancedOrderListElements(`{"client_oid":"my_tp_order"}`, maxAdvancedOrderListLength) + `}`,
		},
	})

	orders := []AdvancedOrderIDRequest{{OrderID: "289991951579667814"}, {ClientOrderID: "my_tp_order"}}
	result, err := newResponseServerExchange(t, `{"id":1,"method":"private/advanced/cancel-order-list","code":0,"result":[{"index":0,"code":0},{"index":1,"code":316,"message":"NO_ACTIVE_ORDER"}]}`).CancelAdvancedOrderList(t.Context(), orders)
	require.NoError(t, err, "CancelAdvancedOrderList must not error for a list with a refused cancel")
	expRefused := []OrderListEntryResponse{{}, {Index: 1, Code: 316, Message: "NO_ACTIVE_ORDER"}}
	assert.Equal(t, expRefused, result, "CancelAdvancedOrderList should decode a refused cancel's outcome")
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/advanced/cancel-order-list","code":0,"result":null}`).CancelAdvancedOrderList(t.Context(), orders)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelAdvancedOrderList should error for a null result")
	_, err = newUnavailableServerExchange(t).CancelAdvancedOrderList(t.Context(), orders)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelAdvancedOrderList should error for an unavailable server")

	skipLiveUnlessCanManipulate(t)
	result, err = e.CancelAdvancedOrderList(t.Context(), orders)
	require.NoError(t, err, "CancelAdvancedOrderList must not error")
	if mockTests {
		exp := []OrderListEntryResponse{{}, {Index: 1}}
		assert.Equal(t, exp, result, "CancelAdvancedOrderList should decode every field")
	} else {
		assert.Len(t, result, len(orders), "CancelAdvancedOrderList should return each cancel's outcome")
	}
}

func TestGetAdvancedOrderListDetail(t *testing.T) {
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t)
	for _, listIDs := range [][]string{nil, {}, {"4611686018427705308", ""}} {
		_, err := refuser.GetAdvancedOrderListDetail(t.Context(), listIDs)
		assert.ErrorIsf(t, err, errListIDRequired, "GetAdvancedOrderListDetail should refuse list IDs %q", listIDs)
	}

	testAdvancedOrderParams(t, "private/advanced/get-order-list-detail", `{"data":[]}`, func(ex *Exchange, listIDs []string) error {
		_, err := ex.GetAdvancedOrderListDetail(t.Context(), listIDs)
		return err
	}, []advancedParamsCase[[]string]{
		{name: "a list ID", req: []string{"4611686018427705309"}, params: `{"list_id_list":["4611686018427705309"]}`},
		{name: "two list IDs", req: []string{"4611686018427705308", "4611686018427705309"}, params: `{"list_id_list":["4611686018427705308","4611686018427705309"]}`},
	})

	listIDs := []string{"4611686018427705308", "4611686018427705309"}
	_, err := newResponseServerExchange(t, `{"id":1,"method":"private/advanced/get-order-list-detail","code":0,"result":null}`).GetAdvancedOrderListDetail(t.Context(), listIDs)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetAdvancedOrderListDetail should error for a null result")
	_, err = newUnavailableServerExchange(t).GetAdvancedOrderListDetail(t.Context(), listIDs)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAdvancedOrderListDetail should error for an unavailable server")

	skipLiveWithoutCredentials(t)
	if !mockTests {
		history, err := e.GetAdvancedOrderHistory(t.Context(), &GetAdvancedOrderHistoryRequest{})
		require.NoError(t, err, "GetAdvancedOrderHistory must not error")
		i := slices.IndexFunc(history, func(o OrderResponse) bool { return o.ListID != "" })
		if i < 0 {
			t.Skip("the account has no OCO, SPOT_ATTACH or DERIV_ATTACH in the past day to look up")
		}
		listIDs = []string{history[i].ListID}
	}
	result, err := e.GetAdvancedOrderListDetail(t.Context(), listIDs)
	require.NoError(t, err, "GetAdvancedOrderListDetail must not error")
	if mockTests {
		exp := []OrderResponse{
			{
				AccountID:             "5207a638-b79a-4c13-9d41-323be7556c17",
				OrderID:               "289987352276367396",
				ClientOrderID:         "289987352276367396",
				OrderType:             OrderTypeLimit,
				TimeInForce:           TimeInForceGoodTillCancel,
				Side:                  SideBuy,
				ExecutionInstructions: []string{},
				Quantity:              preciseNumber("0.01000"),
				LimitPrice:            preciseNumber("100000.00"),
				OrderValue:            preciseNumber("1000.0"),
				AveragePrice:          preciseNumber("100000.00"),
				CumulativeQuantity:    preciseNumber("0.01000"),
				CumulativeValue:       preciseNumber("1000.0000000"),
				CumulativeFee:         preciseNumber("-2.5"),
				Status:                OrderStatusFilled,
				UpdateUserID:          "5207a638-b79a-4c13-9d41-323be7556c17",
				OrderDate:             "2025-09-09",
				InstrumentName:        "BTC_USD",
				FeeInstrumentName:     currency.USD,
				CreateTime:            types.Time(time.UnixMilli(1757409908295)),
				CreateTimeNano:        types.Time(time.Unix(0, 1757409908295832533)),
				UpdateTime:            types.Time(time.UnixMilli(1757430498603)),
				ListID:                "4611686018427705309",
				ContingencyType:       ContingencyTypeSpotAttach,
				LegID:                 "1",
				AttachOrderID:         "5755600460443882762",
			},
			{
				AccountID:             "5207a638-b79a-4c13-9d41-323be7556c17",
				OrderID:               "289987352276367397",
				LimitPrice:            preciseNumber("0.00"),
				AveragePrice:          preciseNumber("0.00"),
				CumulativeQuantity:    preciseNumber("0.00000"),
				CumulativeValue:       preciseNumber("0.0000000"),
				CumulativeFee:         preciseNumber("0"),
				ClientOrderID:         "289987352276367397",
				OrderType:             OrderTypeStopLoss,
				TimeInForce:           TimeInForceGoodTillCancel,
				Side:                  SideSell,
				ExecutionInstructions: []string{},
				Quantity:              preciseNumber("0.01000"),
				OrderValue:            preciseNumber("900.0"),
				Status:                OrderStatusPending,
				UpdateUserID:          "5207a638-b79a-4c13-9d41-323be7556c17",
				OrderDate:             "2025-09-09",
				InstrumentName:        "BTC_USD",
				FeeInstrumentName:     currency.USD,
				CreateTime:            types.Time(time.UnixMilli(1757409908295)),
				CreateTimeNano:        types.Time(time.Unix(0, 1757409908295832533)),
				UpdateTime:            types.Time(time.UnixMilli(1757409908295)),
				ListID:                "4611686018427705309",
				ContingencyType:       ContingencyTypeSpotAttach,
				LegID:                 "2",
				ReferencePrice:        preciseNumber("90000.00"),
				ReferencePriceType:    ReferencePriceTypeMarkPrice,
				AttachOrderID:         "5755600460443882762",
			},
		}
		assert.Equal(t, exp, result, "GetAdvancedOrderListDetail should decode every field")
	} else {
		for i := range result {
			assert.Equalf(t, listIDs[0], result[i].ListID, "GetAdvancedOrderListDetail should return order %d of the list asked for", i)
		}
	}
}
