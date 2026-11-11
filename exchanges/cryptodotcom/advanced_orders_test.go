package cryptodotcom

import (
	"context"
	"net/http"
	"slices"
	"strconv"
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
	// The cases share ex and its recorder in order, each counting the requests before it, so each also checks that a
	// call keeps nothing from the last
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoErrorf(t, send(ex, tc.req), "Sending %s must not error", tc.name)
			params := sent()
			require.Lenf(t, params, i+1, "Sending %s must send one request", tc.name)
			assert.JSONEqf(t, tc.params, params[i], "Sending %s should send its params", tc.name)
		})
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

// placeAdvancedTestOrder places advancedTriggerOrder, with a client order ID of its own, for a live test to act on, and
// returns the order's IDs
func placeAdvancedTestOrder(t *testing.T) AdvancedOrderIDRequest {
	t.Helper()
	req := advancedTriggerOrder()
	req.ClientOrderID = newTradingTestClientOrderID()
	resp, err := e.CreateAdvancedOrder(t.Context(), &req)
	require.NoError(t, err, "CreateAdvancedOrder must place the trigger order the test acts on")
	return AdvancedOrderIDRequest{OrderID: resp.OrderID, ClientOrderID: req.ClientOrderID}
}

// cancelAdvancedOrderAfterTest cancels a trigger order a live test placed once the test ends
func cancelAdvancedOrderAfterTest(t *testing.T, orderID string) {
	t.Helper()
	t.Cleanup(func() {
		_, err := e.CancelAdvancedOrder(context.WithoutCancel(t.Context()), &AdvancedOrderIDRequest{OrderID: orderID})
		assert.NoErrorf(t, err, "CancelAdvancedOrder should cancel trigger order %s, which the test placed", orderID)
	})
}

// cancelAdvancedOrderIfTestFails cancels a trigger order a live test placed to cancel itself, should the test fail
// before it does
func cancelAdvancedOrderIfTestFails(t *testing.T, orderID string) {
	t.Helper()
	t.Cleanup(func() {
		if !t.Failed() {
			return
		}
		if _, err := e.CancelAdvancedOrder(context.WithoutCancel(t.Context()), &AdvancedOrderIDRequest{OrderID: orderID}); err != nil {
			t.Logf("CancelAdvancedOrder could not cancel trigger order %s, which the test placed: %v", orderID, err)
		}
	})
}

// contingencyTestOrders returns an OCO's, OTO's or OTOCO's orders, each with a client order ID of its own in live mode, so
// that strategies the tests place are not refused as duplicates; mock mode keeps the orders as recorded
func contingencyTestOrders(orders []AdvancedOrderLegRequest) []AdvancedOrderLegRequest {
	if mockTests {
		return orders
	}
	clientOrderID := newTradingTestClientOrderID()
	for i := range orders {
		orders[i].ClientOrderID = clientOrderID + "-" + strconv.Itoa(i)
	}
	return orders
}

// cancelContingencyAfterTest cancels an OCO, OTO or OTOCO a live test placed once the test ends
func cancelContingencyAfterTest(t *testing.T, listID string, cancel func(*Exchange, context.Context, string) error) {
	t.Helper()
	t.Cleanup(func() {
		assert.NoErrorf(t, cancel(e, context.WithoutCancel(t.Context()), listID), "Cancelling list %s, which the test placed, should not error", listID)
	})
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

// oneCancelsTheOtherRefusals returns the OCOs CreateOneCancelsTheOtherOrder and its websocket form refuse before
// sending
func oneCancelsTheOtherRefusals() []contingencyRefusal {
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

// oneTriggersTheOtherRefusals returns the OTOs CreateOneTriggersTheOtherOrder and its websocket form refuse before
// sending
func oneTriggersTheOtherRefusals() []contingencyRefusal {
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

// oneTriggersAOneCancelsTheOtherRefusals returns the OTOCOs CreateOneTriggersAOneCancelsTheOtherOrder and its websocket
// form refuse before sending
func oneTriggersAOneCancelsTheOtherRefusals() []contingencyRefusal {
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

// testCancelContingency checks a method cancelling an OCO, OTO or OTOCO by its list ID refuses an empty list ID, sends
// listID and returns a failed request's error
func testCancelContingency(t *testing.T, name, method, listID string, cancel func(*Exchange, context.Context, string) error) {
	t.Helper()
	require.ErrorIsf(t, cancel(newAdvancedRefusalExchange(t), t.Context(), ""), errListIDRequired, "%s must refuse an empty list ID", name)
	testAdvancedOrderParams(t, method, "", func(ex *Exchange, id string) error {
		return cancel(ex, t.Context(), id)
	}, []advancedParamsCase[string]{
		{name: "a list ID", req: listID, params: `{"list_id":"` + listID + `"}`},
	})
	assert.ErrorIsf(t, cancel(newUnavailableServerExchange(t), t.Context(), listID), request.ErrBadStatus, "%s should error for an unavailable server", name)
}

func TestCreateAdvancedOrder(t *testing.T) { //nolint:tparallel // the refusals share refuser in order
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t)
	_, err := refuser.CreateAdvancedOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateAdvancedOrder must refuse a nil request")
	// The cases share refuser in order, before the calls below, so each also checks that a refusal leaves nothing behind
	for _, tc := range triggerOrderRefusals {
		t.Run(tc.name, func(t *testing.T) {
			req := advancedTriggerOrder()
			tc.change(&req)
			_, err := refuser.CreateAdvancedOrder(t.Context(), &req)
			assert.ErrorIsf(t, err, tc.err, "CreateAdvancedOrder should refuse %s", tc.name)
		})
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
			name: "a STOP_LOSS order with a self-trade prevention ID",
			req: &CreateAdvancedOrderRequest{
				InstrumentName:                 "BTC_USD",
				Side:                           SideSell,
				OrderType:                      OrderTypeStopLoss,
				Quantity:                       0.1,
				ReferencePrice:                 45000,
				SelfTradePreventionID:          100,
				SelfTradePreventionInstruction: SelfTradePreventionCancelBoth,
				SelfTradePreventionScope:       SelfTradePreventionScopeMasterOrSubAccount,
			},
			params: `{"instrument_name":"BTC_USD","side":"SELL","type":"STOP_LOSS","quantity":"0.1","ref_price":"45000","stp_id":"100","stp_inst":"B","stp_scope":"M"}`,
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
				SelfTradePreventionID:          0,
				SelfTradePreventionInstruction: SelfTradePreventionCancelBoth,
				SelfTradePreventionScope:       SelfTradePreventionScopeMasterOrSubAccount,
				AttachOrderID:                  "5755600460443890101",
			},
			params: `{"instrument_name":"SHIB_USD","side":"BUY","type":"TAKE_PROFIT_LIMIT","quantity":"1000000","price":"0.00001234","ref_price":"0.00001235","ref_price_type":"LAST_PRICE","client_oid":"shib-take-profit","time_in_force":"GOOD_TILL_CANCEL","exec_inst":["POST_ONLY"],"fee_instrument_name":"CRO","stp_inst":"B","stp_scope":"M","attach_order_id":"5755600460443890101"}`,
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
	if !mockTests {
		req.ClientOrderID = newTradingTestClientOrderID()
	}
	result, err := e.CreateAdvancedOrder(t.Context(), &req)
	require.NoError(t, err, "CreateAdvancedOrder must not error")
	if mockTests {
		exp := &OrderIDResponse{OrderID: "5755600460443882762", ClientOrderID: "c5f682ed-7108-4f1c-b755-972fcdca0f02"}
		assert.Equal(t, exp, result, "CreateAdvancedOrder should decode every field")
	} else {
		require.NotEmpty(t, result.OrderID, "CreateAdvancedOrder must return the order ID")
		cancelAdvancedOrderAfterTest(t, result.OrderID)
		assert.Equal(t, req.ClientOrderID, result.ClientOrderID, "CreateAdvancedOrder should return the order's client order ID")
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
	{name: "an amendment with a negative new price", req: &AmendAdvancedOrderRequest{OrderID: "5755600460443882762", NewQuantity: 10, NewPrice: new(types.Number(-0.0278))}, err: errInvalidPrice},
	{name: "an amendment with a negative new reference price", req: &AmendAdvancedOrderRequest{OrderID: "5755600460443882762", NewQuantity: 10, NewPrice: new(types.Number(0.0278)), NewReferencePrice: -0.0279}, err: errInvalidReferencePrice},
}

func TestAmendAdvancedOrder(t *testing.T) { //nolint:tparallel // the refusals share refuser in order
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t)
	// The cases share refuser in order, before the calls below, so each also checks that a refusal leaves nothing behind
	for _, tc := range amendAdvancedOrderRefusals {
		t.Run(tc.name, func(t *testing.T) {
			_, err := refuser.AmendAdvancedOrder(t.Context(), tc.req)
			assert.ErrorIsf(t, err, tc.err, "AmendAdvancedOrder should refuse %s", tc.name)
		})
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
	if !mockTests {
		req.OrderID = placeAdvancedTestOrder(t).OrderID
		cancelAdvancedOrderAfterTest(t, req.OrderID)
	}
	result, err := e.AmendAdvancedOrder(t.Context(), req)
	require.NoError(t, err, "AmendAdvancedOrder must not error")
	if mockTests {
		exp := &OrderIDResponse{OrderID: "5755600460443882762", ClientOrderID: "c5f682ed-7108-4f1c-b755-972fcdca0f02"}
		assert.Equal(t, exp, result, "AmendAdvancedOrder should decode every field")
	} else {
		assert.Equal(t, req.OrderID, result.OrderID, "AmendAdvancedOrder should return the amended order's ID")
	}
}

func TestCreateOneCancelsTheOtherOrder(t *testing.T) { //nolint:tparallel // the refusals share refuser in order
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t)
	// The cases share refuser in order, before the calls below, so each also checks that a refusal leaves nothing behind
	for _, tc := range oneCancelsTheOtherRefusals() {
		t.Run(tc.name, func(t *testing.T) {
			_, err := refuser.CreateOneCancelsTheOtherOrder(t.Context(), tc.orders)
			assert.ErrorIsf(t, err, tc.err, "CreateOneCancelsTheOtherOrder should refuse %s", tc.name)
		})
	}

	testAdvancedOrderParams(t, "private/advanced/create-oco", `{"list_id":"6498090546073120100"}`, func(ex *Exchange, orders []AdvancedOrderLegRequest) error {
		_, err := ex.CreateOneCancelsTheOtherOrder(t.Context(), orders)
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
					SelfTradePreventionID:          7,
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

	_, err := newResponseServerExchange(t, `{"id":1,"method":"private/advanced/create-oco","code":0,"result":null}`).CreateOneCancelsTheOtherOrder(t.Context(), ocoOrders())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateOneCancelsTheOtherOrder should error for a null result")
	_, err = newUnavailableServerExchange(t).CreateOneCancelsTheOtherOrder(t.Context(), ocoOrders())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateOneCancelsTheOtherOrder should error for an unavailable server")

	skipLiveUnlessCanManipulate(t)
	result, err := e.CreateOneCancelsTheOtherOrder(t.Context(), contingencyTestOrders(ocoOrders()))
	require.NoError(t, err, "CreateOneCancelsTheOtherOrder must not error")
	if mockTests {
		exp := &AdvancedOrderListIDResponse{ListID: "6498090546073120100"}
		assert.Equal(t, exp, result, "CreateOneCancelsTheOtherOrder should decode every field")
	} else {
		require.NotEmpty(t, result.ListID, "CreateOneCancelsTheOtherOrder must return the list ID")
		cancelContingencyAfterTest(t, string(result.ListID), (*Exchange).CancelOneCancelsTheOtherOrder)
	}
}

func TestCancelOneCancelsTheOtherOrder(t *testing.T) {
	t.Parallel()
	testCancelContingency(t, "CancelOneCancelsTheOtherOrder", "private/advanced/cancel-oco", "6498090546073120100", (*Exchange).CancelOneCancelsTheOtherOrder)

	skipLiveUnlessCanManipulate(t)
	placed, err := e.CreateOneCancelsTheOtherOrder(t.Context(), contingencyTestOrders(ocoOrders()))
	require.NoError(t, err, "CreateOneCancelsTheOtherOrder must place the OCO the test cancels")
	require.NoError(t, e.CancelOneCancelsTheOtherOrder(t.Context(), string(placed.ListID)), "CancelOneCancelsTheOtherOrder must not error")
}

func TestCreateOneTriggersTheOtherOrder(t *testing.T) { //nolint:tparallel // the refusals share refuser in order
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t)
	// The cases share refuser in order, before the calls below, so each also checks that a refusal leaves nothing behind
	for _, tc := range oneTriggersTheOtherRefusals() {
		t.Run(tc.name, func(t *testing.T) {
			_, err := refuser.CreateOneTriggersTheOtherOrder(t.Context(), tc.orders)
			assert.ErrorIsf(t, err, tc.err, "CreateOneTriggersTheOtherOrder should refuse %s", tc.name)
		})
	}

	testAdvancedOrderParams(t, "private/advanced/create-oto", `{"list_id":"6498090546073120199"}`, func(ex *Exchange, orders []AdvancedOrderLegRequest) error {
		_, err := ex.CreateOneTriggersTheOtherOrder(t.Context(), orders)
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
					SelfTradePreventionID:          32767,
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

	_, err := newResponseServerExchange(t, `{"id":1,"method":"private/advanced/create-oto","code":0,"result":null}`).CreateOneTriggersTheOtherOrder(t.Context(), otoOrders())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateOneTriggersTheOtherOrder should error for a null result")
	// Crypto.com's docs give the list ID as a bare integer, one a float cannot hold
	bare, err := newResponseServerExchange(t, `{"id":1,"method":"private/advanced/create-oto","code":0,"result":{"list_id":6498090546073120199}}`).CreateOneTriggersTheOtherOrder(t.Context(), otoOrders())
	require.NoError(t, err, "CreateOneTriggersTheOtherOrder must not error for a bare list ID")
	assert.Equal(t, &AdvancedOrderListIDResponse{ListID: "6498090546073120199"}, bare, "CreateOneTriggersTheOtherOrder should keep a bare list ID digit for digit")
	_, err = newUnavailableServerExchange(t).CreateOneTriggersTheOtherOrder(t.Context(), otoOrders())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateOneTriggersTheOtherOrder should error for an unavailable server")

	skipLiveUnlessCanManipulate(t)
	result, err := e.CreateOneTriggersTheOtherOrder(t.Context(), contingencyTestOrders(otoOrders()))
	require.NoError(t, err, "CreateOneTriggersTheOtherOrder must not error")
	if mockTests {
		exp := &AdvancedOrderListIDResponse{ListID: "6498090546073120199"}
		assert.Equal(t, exp, result, "CreateOneTriggersTheOtherOrder should decode every field")
	} else {
		require.NotEmpty(t, result.ListID, "CreateOneTriggersTheOtherOrder must return the list ID")
		cancelContingencyAfterTest(t, string(result.ListID), (*Exchange).CancelOneTriggersTheOtherOrder)
	}
}

func TestCancelOneTriggersTheOtherOrder(t *testing.T) {
	t.Parallel()
	testCancelContingency(t, "CancelOneTriggersTheOtherOrder", "private/advanced/cancel-oto", "6498090546073120199", (*Exchange).CancelOneTriggersTheOtherOrder)

	skipLiveUnlessCanManipulate(t)
	placed, err := e.CreateOneTriggersTheOtherOrder(t.Context(), contingencyTestOrders(otoOrders()))
	require.NoError(t, err, "CreateOneTriggersTheOtherOrder must place the OTO the test cancels")
	require.NoError(t, e.CancelOneTriggersTheOtherOrder(t.Context(), string(placed.ListID)), "CancelOneTriggersTheOtherOrder must not error")
}

func TestCreateOneTriggersAOneCancelsTheOtherOrder(t *testing.T) { //nolint:tparallel // the refusals share refuser in order
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t)
	// The cases share refuser in order, before the calls below, so each also checks that a refusal leaves nothing behind
	for _, tc := range oneTriggersAOneCancelsTheOtherRefusals() {
		t.Run(tc.name, func(t *testing.T) {
			_, err := refuser.CreateOneTriggersAOneCancelsTheOtherOrder(t.Context(), tc.orders)
			assert.ErrorIsf(t, err, tc.err, "CreateOneTriggersAOneCancelsTheOtherOrder should refuse %s", tc.name)
		})
	}

	testAdvancedOrderParams(t, "private/advanced/create-otoco", `{"list_id":"6498090546073120243"}`, func(ex *Exchange, orders []AdvancedOrderLegRequest) error {
		_, err := ex.CreateOneTriggersAOneCancelsTheOtherOrder(t.Context(), orders)
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

	_, err := newResponseServerExchange(t, `{"id":1,"method":"private/advanced/create-otoco","code":0,"result":null}`).CreateOneTriggersAOneCancelsTheOtherOrder(t.Context(), otocoOrders())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateOneTriggersAOneCancelsTheOtherOrder should error for a null result")
	_, err = newUnavailableServerExchange(t).CreateOneTriggersAOneCancelsTheOtherOrder(t.Context(), otocoOrders())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateOneTriggersAOneCancelsTheOtherOrder should error for an unavailable server")

	skipLiveUnlessCanManipulate(t)
	result, err := e.CreateOneTriggersAOneCancelsTheOtherOrder(t.Context(), contingencyTestOrders(otocoOrders()))
	require.NoError(t, err, "CreateOneTriggersAOneCancelsTheOtherOrder must not error")
	if mockTests {
		exp := &AdvancedOrderListIDResponse{ListID: "6498090546073120243"}
		assert.Equal(t, exp, result, "CreateOneTriggersAOneCancelsTheOtherOrder should decode every field")
	} else {
		require.NotEmpty(t, result.ListID, "CreateOneTriggersAOneCancelsTheOtherOrder must return the list ID")
		cancelContingencyAfterTest(t, string(result.ListID), (*Exchange).CancelOneTriggersAOneCancelsTheOtherOrder)
	}
}

func TestCancelOneTriggersAOneCancelsTheOtherOrder(t *testing.T) {
	t.Parallel()
	testCancelContingency(t, "CancelOneTriggersAOneCancelsTheOtherOrder", "private/advanced/cancel-otoco", "6498090546073120243", (*Exchange).CancelOneTriggersAOneCancelsTheOtherOrder)

	skipLiveUnlessCanManipulate(t)
	placed, err := e.CreateOneTriggersAOneCancelsTheOtherOrder(t.Context(), contingencyTestOrders(otocoOrders()))
	require.NoError(t, err, "CreateOneTriggersAOneCancelsTheOtherOrder must place the OTOCO the test cancels")
	require.NoError(t, e.CancelOneTriggersAOneCancelsTheOtherOrder(t.Context(), string(placed.ListID)), "CancelOneTriggersAOneCancelsTheOtherOrder must not error")
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
	if !mockTests {
		req.OrderID = placeAdvancedTestOrder(t).OrderID
	}
	result, err := e.CancelAdvancedOrder(t.Context(), req)
	require.NoError(t, err, "CancelAdvancedOrder must not error")
	if mockTests {
		exp := &OrderIDResponse{OrderID: "5755600460443882762", ClientOrderID: "c5f682ed-7108-4f1c-b755-972fcdca0f02"}
		assert.Equal(t, exp, result, "CancelAdvancedOrder should decode every field")
	} else {
		assert.Equal(t, req.OrderID, result.OrderID, "CancelAdvancedOrder should return the cancelled order's ID")
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

func TestCancelAllAdvancedOrders(t *testing.T) { //nolint:tparallel // the refusals share refuser in order
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t)
	// The cases share refuser in order, before the calls below, so each also checks that a refusal leaves nothing behind
	for _, tc := range cancelAllAdvancedOrdersRefusals {
		t.Run(tc.name, func(t *testing.T) {
			assert.ErrorIsf(t, refuser.CancelAllAdvancedOrders(t.Context(), tc.req), tc.err, "CancelAllAdvancedOrders should refuse %s", tc.name)
		})
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
	if !mockTests {
		// An instrument the other tests place no advanced orders on, so that cancelling its orders live leaves the BTC_USD
		// OTOs and OTOCOs the cancel tests place and cancel
		req.InstrumentName = tradingTestCancelAllInstrument
	}
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
			// With no advanced order of the account's from the past day to look up, Crypto.com refusing the recorded order,
			// which is not the account's, shows it looked the order up
			_, err = e.GetAdvancedOrderDetail(t.Context(), req)
			assertRefusedFor(t, err, unknownOrderCodes, "GetAdvancedOrderDetail")
			return
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
			ContingencyType:       ContingencyTypeDerivativesAttach,
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

func TestGetAdvancedOrderHistory(t *testing.T) { //nolint:tparallel // the refusals share refuser in order
	t.Parallel()
	startTime, endTime := time.Unix(0, 1792713600000081486), time.Unix(0, 1792799991058211357)
	refuser := newAdvancedRefusalExchange(t)
	// The cases share refuser in order, before the calls below, so each also checks that a refusal leaves nothing behind
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
		t.Run(tc.name, func(t *testing.T) {
			_, err := refuser.GetAdvancedOrderHistory(t.Context(), tc.req)
			assert.ErrorIsf(t, err, tc.err, "GetAdvancedOrderHistory should refuse %s", tc.name)
		})
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
				ContingencyType:       ContingencyTypeDerivativesAttach,
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

// advancedOrderListTestOrders returns advancedOrderListOrders, each with a client order ID of its own in live mode, as a
// run placing the docs' client order IDs would have its next run refused as duplicates; mock mode keeps them, as recorded
func advancedOrderListTestOrders() []CreateAdvancedOrderRequest {
	orders := advancedOrderListOrders()
	if mockTests {
		return orders
	}
	clientOrderID := newTradingTestClientOrderID()
	for i := range orders {
		orders[i].ClientOrderID = clientOrderID + "-" + strconv.Itoa(i)
	}
	return orders
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

func TestCreateAdvancedOrderList(t *testing.T) { //nolint:tparallel // the refusals share refuser in order
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t)
	// The cases share refuser in order, before the calls below, so each also checks that a refusal leaves nothing behind
	for _, tc := range createAdvancedOrderListRefusals() {
		t.Run(tc.name, func(t *testing.T) {
			_, err := refuser.CreateAdvancedOrderList(t.Context(), tc.orders)
			assert.ErrorIsf(t, err, tc.err, "CreateAdvancedOrderList should refuse %s", tc.name)
		})
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
	orders = advancedOrderListTestOrders()
	result, err = e.CreateAdvancedOrderList(t.Context(), orders)
	require.NoError(t, err, "CreateAdvancedOrderList must not error")
	if mockTests {
		exp := []OrderListEntryResponse{
			{OrderID: "289991951579667814", ClientOrderID: "my_sl_order"},
			{Index: 1, OrderID: "289991951579667815", ClientOrderID: "my_tp_order"},
		}
		assert.Equal(t, exp, result, "CreateAdvancedOrderList should decode every field")
	} else {
		for i := range result {
			if result[i].Code == 0 {
				cancelAdvancedOrderAfterTest(t, result[i].OrderID)
			}
		}
		assertListOutcomes(t, result, len(orders), "CreateAdvancedOrderList", "place")
	}
}

func TestCancelAdvancedOrderList(t *testing.T) { //nolint:tparallel // the refusals share refuser in order
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t)
	// The cases share refuser in order, before the calls below, so each also checks that a refusal leaves nothing behind
	for _, tc := range cancelAdvancedOrderListRefusals() {
		t.Run(tc.name, func(t *testing.T) {
			_, err := refuser.CancelAdvancedOrderList(t.Context(), tc.orders)
			assert.ErrorIsf(t, err, tc.err, "CancelAdvancedOrderList should refuse %s", tc.name)
		})
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
	if !mockTests {
		first := placeAdvancedTestOrder(t)
		cancelAdvancedOrderIfTestFails(t, first.OrderID)
		second := placeAdvancedTestOrder(t)
		cancelAdvancedOrderIfTestFails(t, second.OrderID)
		orders = []AdvancedOrderIDRequest{{OrderID: first.OrderID}, {ClientOrderID: second.ClientOrderID}}
	}
	result, err = e.CancelAdvancedOrderList(t.Context(), orders)
	require.NoError(t, err, "CancelAdvancedOrderList must not error")
	require.Len(t, result, len(orders), "CancelAdvancedOrderList must return each cancel's outcome")
	if mockTests {
		exp := []OrderListEntryResponse{{}, {Index: 1}}
		assert.Equal(t, exp, result, "CancelAdvancedOrderList should decode every field")
		return
	}
	assertListOutcomes(t, result, len(orders), "CancelAdvancedOrderList", "cancel")
}

func TestGetAdvancedOrderListDetail(t *testing.T) { //nolint:tparallel // the refusals share refuser in order
	t.Parallel()
	refuser := newAdvancedRefusalExchange(t)
	// The cases share refuser in order, before the calls below, so each also checks that a refusal leaves nothing behind
	for _, tc := range []struct {
		name    string
		listIDs []string
	}{
		{name: "a nil list", listIDs: nil},
		{name: "an empty list", listIDs: []string{}},
		{name: "a list holding an empty ID", listIDs: []string{"4611686018427705308", ""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := refuser.GetAdvancedOrderListDetail(t.Context(), tc.listIDs)
			assert.ErrorIsf(t, err, errListIDRequired, "GetAdvancedOrderListDetail should refuse list IDs %q", tc.listIDs)
		})
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
			// With no list of the account's from the past day to look up, Crypto.com's answer for the recorded lists, which
			// are not the account's, shows it looked them up; its docs leave out whether it refuses unknown list IDs or
			// returns no orders for them, so either passes
			unknown, err := e.GetAdvancedOrderListDetail(t.Context(), listIDs)
			if err != nil {
				assertRefusedFor(t, err, unknownOrderCodes, "GetAdvancedOrderListDetail")
			} else {
				assert.Empty(t, unknown, "GetAdvancedOrderListDetail should return no orders for lists the account does not hold")
			}
			return
		}
		listIDs = []string{string(history[i].ListID)}
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
			assert.Equalf(t, listIDs[0], string(result[i].ListID), "GetAdvancedOrderListDetail should return order %d of the list asked for", i)
		}
	}
}

func TestCreateAdvancedOrderRequestValidate(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, (*CreateAdvancedOrderRequest)(nil).validate(), common.ErrNilPointer, "validate should refuse a nil request")
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, tc := range triggerOrderRefusals {
		t.Run(tc.name, func(t *testing.T) {
			req := advancedTriggerOrder()
			tc.change(&req)
			assert.ErrorIsf(t, req.validate(), tc.err, "validate should refuse %s", tc.name)
		})
	}
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, tc := range []struct {
		name   string
		change func(*CreateAdvancedOrderRequest)
	}{
		{"a self-trade prevention ID above 32767", func(r *CreateAdvancedOrderRequest) {
			r.SelfTradePreventionScope, r.SelfTradePreventionInstruction, r.SelfTradePreventionID = SelfTradePreventionScopeMasterOrSubAccount, SelfTradePreventionCancelBoth, 32768
		}},
		{"a self-trade prevention ID without a scope", func(r *CreateAdvancedOrderRequest) { r.SelfTradePreventionID = 100 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := advancedTriggerOrder()
			tc.change(&req)
			assert.ErrorIsf(t, req.validate(), errInvalidSelfTradePreventionID, "validate should refuse %s", tc.name)
		})
	}
	for _, tc := range []struct {
		name   string
		change func(*CreateAdvancedOrderRequest)
	}{
		{"the docs' trigger order", func(*CreateAdvancedOrderRequest) {}},
		{"an order attached to the cross margin position with REDUCE_ONLY", func(r *CreateAdvancedOrderRequest) {
			r.InstrumentName, r.AttachIsolationID, r.ExecutionInstructions = "BTCUSD-PERP", "0", []string{ExecutionInstructionReduceOnly}
		}},
		{"an order with a self-trade prevention ID", func(r *CreateAdvancedOrderRequest) {
			r.SelfTradePreventionScope, r.SelfTradePreventionInstruction, r.SelfTradePreventionID = SelfTradePreventionScopeMasterOrSubAccount, SelfTradePreventionCancelBoth, 100
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := advancedTriggerOrder()
			tc.change(&req)
			assert.NoErrorf(t, req.validate(), "validate should accept %s", tc.name)
		})
	}
}

func TestAdvancedOrderLegRequestValidate(t *testing.T) {
	t.Parallel()
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, tc := range advancedOrderLegChanges {
		t.Run(tc.name, func(t *testing.T) {
			leg := ocoOrders()[1]
			tc.change(&leg)
			assert.ErrorIsf(t, leg.validate(), tc.err, "validate should refuse %s", tc.name)
		})
	}
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, tc := range []struct {
		name   string
		change func(*AdvancedOrderLegRequest)
		err    error
	}{
		{"a MARKET buy spending neither a quantity nor a notional", func(o *AdvancedOrderLegRequest) {
			o.Side, o.OrderType, o.Quantity, o.ReferencePrice = SideBuy, OrderTypeMarket, 0, 0
		}, errInvalidQuantity},
		{"a self-trade prevention ID above 32767", func(o *AdvancedOrderLegRequest) {
			o.SelfTradePreventionScope, o.SelfTradePreventionInstruction, o.SelfTradePreventionID = SelfTradePreventionScopeSubAccount, SelfTradePreventionCancelMaker, 32768
		}, errInvalidSelfTradePreventionID},
		{"a self-trade prevention ID without a scope", func(o *AdvancedOrderLegRequest) { o.SelfTradePreventionID = 7 }, errInvalidSelfTradePreventionID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			leg := ocoOrders()[1]
			tc.change(&leg)
			assert.ErrorIsf(t, leg.validate(), tc.err, "validate should refuse %s", tc.name)
		})
	}
	for _, tc := range []struct {
		name   string
		change func(*AdvancedOrderLegRequest)
	}{
		{"an OCO's STOP_LOSS sell", func(*AdvancedOrderLegRequest) {}},
		{"a MARKET buy spending a notional", func(o *AdvancedOrderLegRequest) {
			o.Side, o.OrderType, o.Quantity, o.Notional, o.ReferencePrice = SideBuy, OrderTypeMarket, 0, 8000, 0
		}},
		{"a STOP_LOSS buy spending a notional", func(o *AdvancedOrderLegRequest) {
			o.Side, o.Quantity, o.Notional, o.ReferencePrice = SideBuy, 0, 8300, 83000
		}},
		{"a TAKE_PROFIT buy spending a notional", func(o *AdvancedOrderLegRequest) {
			o.Side, o.OrderType, o.Quantity, o.Notional, o.ReferencePrice = SideBuy, OrderTypeTakeProfit, 0, 7600, 76000
		}},
		{"an order with a self-trade prevention ID", func(o *AdvancedOrderLegRequest) {
			o.SelfTradePreventionScope, o.SelfTradePreventionInstruction, o.SelfTradePreventionID = SelfTradePreventionScopeSubAccount, SelfTradePreventionCancelMaker, 32767
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			leg := ocoOrders()[1]
			tc.change(&leg)
			assert.NoErrorf(t, leg.validate(), "validate should accept %s", tc.name)
		})
	}
}

func TestValidateAdvancedOrder(t *testing.T) {
	t.Parallel()
	// Each order holds the fields validateAdvancedOrder is given, as AdvancedOrderLegRequest.validate passes them
	for _, tc := range []struct {
		name  string
		order *AdvancedOrderLegRequest
		err   error
	}{
		{name: "a LIMIT order", order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideBuy, OrderType: OrderTypeLimit, Price: 78000}},
		{name: "a MARKET order", order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeMarket}},
		{name: "a STOP_LOSS order", order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeStopLoss, ReferencePrice: 40000}},
		{name: "a STOP_LIMIT order", order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeStopLimit, Price: 39900, ReferencePrice: 40000}},
		{name: "a TAKE_PROFIT order", order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeTakeProfit, ReferencePrice: 120000}},
		{
			name:  "a TAKE_PROFIT_LIMIT order",
			order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeTakeProfitLimit, Price: 120100, ReferencePrice: 120000},
		},
		{
			name:  "a trigger order comparing the mark price",
			order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeStopLoss, ReferencePrice: 40000, ReferencePriceType: ReferencePriceTypeMarkPrice},
		},
		{
			name:  "a trigger order comparing the index price",
			order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeStopLoss, ReferencePrice: 40000, ReferencePriceType: ReferencePriceTypeIndexPrice},
		},
		{
			name:  "a trigger order comparing the last price",
			order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeStopLoss, ReferencePrice: 40000, ReferencePriceType: ReferencePriceTypeLastPrice},
		},
		{name: "an order without an instrument", order: &AdvancedOrderLegRequest{Side: SideBuy, OrderType: OrderTypeLimit, Price: 78000}, err: errInstrumentNameRequired},
		{name: "an order without a side", order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", OrderType: OrderTypeLimit, Price: 78000}, err: errInvalidSide},
		{name: "a lower-case side", order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: "buy", OrderType: OrderTypeLimit, Price: 78000}, err: errInvalidSide},
		{name: "an order without a type", order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideBuy, Price: 78000}, err: errInvalidOrderType},
		{name: "an unknown type", order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideBuy, OrderType: "STOP", ReferencePrice: 40000}, err: errInvalidOrderType},
		{name: "a lower-case type", order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideBuy, OrderType: "limit", Price: 78000}, err: errInvalidOrderType},
		{name: "a LIMIT order without a price", order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideBuy, OrderType: OrderTypeLimit}, err: errInvalidPrice},
		{name: "a LIMIT order with a negative price", order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideBuy, OrderType: OrderTypeLimit, Price: -78000}, err: errInvalidPrice},
		{
			name:  "a STOP_LIMIT order without a price",
			order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeStopLimit, ReferencePrice: 40000},
			err:   errInvalidPrice,
		},
		{
			name:  "a TAKE_PROFIT_LIMIT order without a price",
			order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeTakeProfitLimit, ReferencePrice: 120000},
			err:   errInvalidPrice,
		},
		{name: "a STOP_LOSS order without a reference price", order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeStopLoss}, err: errInvalidReferencePrice},
		{
			name:  "a STOP_LIMIT order without a reference price",
			order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeStopLimit, Price: 39900},
			err:   errInvalidReferencePrice,
		},
		{name: "a TAKE_PROFIT order without a reference price", order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeTakeProfit}, err: errInvalidReferencePrice},
		{
			name:  "a TAKE_PROFIT_LIMIT order without a reference price",
			order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeTakeProfitLimit, Price: 120100},
			err:   errInvalidReferencePrice,
		},
		{
			name:  "a trigger order with a negative reference price",
			order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeStopLoss, ReferencePrice: -40000},
			err:   errInvalidReferencePrice,
		},
		{
			name:  "an unknown reference price type",
			order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeStopLoss, ReferencePrice: 40000, ReferencePriceType: "MID_PRICE"},
			err:   errInvalidReferencePriceType,
		},
		{
			name:  "a lower-case reference price type",
			order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeStopLoss, ReferencePrice: 40000, ReferencePriceType: "mark_price"},
			err:   errInvalidReferencePriceType,
		},
		{
			name:  "a LIMIT order with an unknown reference price type",
			order: &AdvancedOrderLegRequest{InstrumentName: "BTC_USD", Side: SideBuy, OrderType: OrderTypeLimit, Price: 78000, ReferencePriceType: "MID_PRICE"},
			err:   errInvalidReferencePriceType,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			o := tc.order
			err := validateAdvancedOrder(o.InstrumentName, o.Side, o.OrderType, o.Price, o.ReferencePrice, o.ReferencePriceType)
			assert.ErrorIsf(t, err, tc.err, "validateAdvancedOrder should return the expected error for %s", tc.name)
		})
	}
}

func TestIsTriggerOrderType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType string
		exp       bool
	}{
		{OrderTypeStopLoss, true},
		{OrderTypeStopLimit, true},
		{OrderTypeTakeProfit, true},
		{OrderTypeTakeProfitLimit, true},
		{OrderTypeLimit, false},
		{OrderTypeMarket, false},
		{"", false},
		{"stop_loss", false},
		{CancelOrderTypeTrigger, false},
	} {
		t.Run(subtestName(tc.orderType), func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, isTriggerOrderType(tc.orderType), "isTriggerOrderType should report whether %q is a trigger order type", tc.orderType)
		})
	}
}

func TestAmendAdvancedOrderRequestParams(t *testing.T) {
	t.Parallel()
	for _, tc := range amendAdvancedOrderRefusals {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := tc.req.params()
			assert.ErrorIsf(t, err, tc.err, "params should refuse %s", tc.name)
			assert.Nilf(t, got, "params should return no params for %s", tc.name)
		})
	}
	for _, tc := range []struct {
		name string
		req  *AmendAdvancedOrderRequest
		exp  *amendAdvancedOrderParams
	}{
		{
			name: "a trigger market order named by its order ID, its zero price kept",
			req:  &AmendAdvancedOrderRequest{OrderID: "5755600460443882762", NewQuantity: 10, NewPrice: new(types.Number(0))},
			exp:  &amendAdvancedOrderParams{OrderID: "5755600460443882762", NewQuantity: 10, NewPrice: "0"},
		},
		{
			name: "a trigger limit order named by its client order ID",
			req:  &AmendAdvancedOrderRequest{ClientOrderID: "c5f682ed-7108-4f1c-b755-972fcdca0f02", NewQuantity: 10, NewPrice: new(types.Number(0.0279))},
			exp:  &amendAdvancedOrderParams{ClientOrderID: "c5f682ed-7108-4f1c-b755-972fcdca0f02", NewQuantity: 10, NewPrice: "0.0279"},
		},
		{
			name: "every field",
			req: &AmendAdvancedOrderRequest{
				OrderID:           "5755600460443882762",
				ClientOrderID:     "c5f682ed-7108-4f1c-b755-972fcdca0f02",
				NewQuantity:       12,
				NewPrice:          new(types.Number(0.0278)),
				NewReferencePrice: 0.0279,
			},
			exp: &amendAdvancedOrderParams{
				OrderID:           "5755600460443882762",
				ClientOrderID:     "c5f682ed-7108-4f1c-b755-972fcdca0f02",
				NewQuantity:       12,
				NewPrice:          "0.0278",
				NewReferencePrice: 0.0279,
			},
		},
		{
			name: "a price below a millionth, in plain notation",
			req:  &AmendAdvancedOrderRequest{OrderID: "5755600460443882762", NewQuantity: 1000000, NewPrice: new(types.Number(0.00000123))},
			exp:  &amendAdvancedOrderParams{OrderID: "5755600460443882762", NewQuantity: 1000000, NewPrice: "0.00000123"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := tc.req.params()
			require.NoErrorf(t, err, "params must not error for %s", tc.name)
			assert.Equalf(t, tc.exp, got, "params should return the params of %s", tc.name)
		})
	}
}

func TestValidateContingencyOrders(t *testing.T) {
	t.Parallel()
	limit, stopLoss := ocoOrders()[0], ocoOrders()[1]
	market := limit
	market.OrderType, market.Price = OrderTypeMarket, 0
	stopLimit := stopLoss
	stopLimit.OrderType, stopLimit.Price = OrderTypeStopLimit, 39900
	takeProfit := stopLoss
	takeProfit.OrderType, takeProfit.ReferencePrice = OrderTypeTakeProfit, 130000
	takeProfitLimit := takeProfit
	takeProfitLimit.OrderType, takeProfitLimit.Price = OrderTypeTakeProfitLimit, 130100
	noInstrument := stopLoss
	noInstrument.InstrumentName = ""
	for _, tc := range []struct {
		name            string
		orders          []AdvancedOrderLegRequest
		contingencyType string
		count           int
		exp             *contingencyOrderCounts
		err             error
		contains        string
	}{
		{name: "no orders", contingencyType: ContingencyTypeOneCancelsTheOther, count: 2, err: errOrderListEmpty},
		{
			name:            "an OCO of one order",
			orders:          []AdvancedOrderLegRequest{limit},
			contingencyType: ContingencyTypeOneCancelsTheOther,
			count:           2,
			err:             errOrderListTooShort,
			contains:        "an OCO takes 2 orders, not 1",
		},
		{
			name:            "an OTOCO of two orders",
			orders:          []AdvancedOrderLegRequest{limit, stopLoss},
			contingencyType: ContingencyTypeOneTriggersAOneCancelsTheOther,
			count:           3,
			err:             errOrderListTooShort,
			contains:        "an OTOCO takes 3 orders, not 2",
		},
		{
			name:            "an OTO of three orders",
			orders:          []AdvancedOrderLegRequest{limit, stopLoss, takeProfit},
			contingencyType: ContingencyTypeOneTriggersTheOther,
			count:           2,
			err:             errOrderListTooLong,
			contains:        "an OTO takes 2 orders, not 3",
		},
		{
			name:            "an OTOCO of four orders",
			orders:          []AdvancedOrderLegRequest{limit, stopLoss, takeProfit, takeProfit},
			contingencyType: ContingencyTypeOneTriggersAOneCancelsTheOther,
			count:           3,
			err:             errOrderListTooLong,
			contains:        "an OTOCO takes 3 orders, not 4",
		},
		{
			name:            "an invalid first order",
			orders:          []AdvancedOrderLegRequest{noInstrument, limit},
			contingencyType: ContingencyTypeOneCancelsTheOther,
			count:           2,
			err:             errInstrumentNameRequired,
			contains:        "order 0:",
		},
		{
			name:            "an invalid last order",
			orders:          []AdvancedOrderLegRequest{limit, stopLoss, noInstrument},
			contingencyType: ContingencyTypeOneTriggersAOneCancelsTheOther,
			count:           3,
			err:             errInstrumentNameRequired,
			contains:        "order 2:",
		},
		{
			name:            "a LIMIT order and a STOP_LOSS",
			orders:          []AdvancedOrderLegRequest{limit, stopLoss},
			contingencyType: ContingencyTypeOneCancelsTheOther,
			count:           2,
			exp:             &contingencyOrderCounts{limit: 1, stopLoss: 1},
		},
		{
			name:            "a MARKET order and a STOP_LIMIT, counted as a stop loss",
			orders:          []AdvancedOrderLegRequest{market, stopLimit},
			contingencyType: ContingencyTypeOneTriggersTheOther,
			count:           2,
			exp:             &contingencyOrderCounts{market: 1, stopLoss: 1},
		},
		{
			name:            "a LIMIT order, a STOP_LOSS and a TAKE_PROFIT",
			orders:          []AdvancedOrderLegRequest{limit, stopLoss, takeProfit},
			contingencyType: ContingencyTypeOneTriggersAOneCancelsTheOther,
			count:           3,
			exp:             &contingencyOrderCounts{limit: 1, stopLoss: 1, takeProfit: 1},
		},
		{
			name:            "a TAKE_PROFIT_LIMIT, counted as a take profit, before a MARKET order",
			orders:          []AdvancedOrderLegRequest{takeProfitLimit, market},
			contingencyType: ContingencyTypeOneTriggersTheOther,
			count:           2,
			exp:             &contingencyOrderCounts{market: 1, takeProfit: 1},
		},
		{
			name:            "two LIMIT orders, counted for the strategy to judge",
			orders:          []AdvancedOrderLegRequest{limit, limit},
			contingencyType: ContingencyTypeOneCancelsTheOther,
			count:           2,
			exp:             &contingencyOrderCounts{limit: 2},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := validateContingencyOrders(tc.orders, tc.contingencyType, tc.count)
			assert.ErrorIsf(t, err, tc.err, "validateContingencyOrders should return the expected error for %s", tc.name)
			if tc.contains != "" {
				assert.ErrorContainsf(t, err, tc.contains, "validateContingencyOrders should say why it refuses %s", tc.name)
			}
			assert.Equalf(t, tc.exp, got, "validateContingencyOrders should count the orders of %s", tc.name)
		})
	}
}

func TestOneCancelsTheOtherParams(t *testing.T) {
	t.Parallel()
	for _, tc := range oneCancelsTheOtherRefusals() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := oneCancelsTheOtherParams(tc.orders)
			assert.ErrorIsf(t, err, tc.err, "oneCancelsTheOtherParams should refuse %s", tc.name)
			assert.Nilf(t, got, "oneCancelsTheOtherParams should return no params for %s", tc.name)
		})
	}
	limit, stopLoss := ocoOrders()[0], ocoOrders()[1]
	markPrice := stopLoss
	markPrice.ReferencePriceType = ReferencePriceTypeMarkPrice
	stopLimit := stopLoss
	stopLimit.OrderType, stopLimit.Price = OrderTypeStopLimit, 39900
	takeProfit := stopLoss
	takeProfit.OrderType, takeProfit.ReferencePrice = OrderTypeTakeProfit, 130000
	takeProfitLimit := takeProfit
	takeProfitLimit.OrderType, takeProfitLimit.Price = OrderTypeTakeProfitLimit, 130100
	for _, tc := range []struct {
		name   string
		orders []AdvancedOrderLegRequest
	}{
		{"the docs' OCO", ocoOrders()},
		{"a trigger order comparing the mark price", []AdvancedOrderLegRequest{limit, markPrice}},
		{"a STOP_LIMIT trigger order", []AdvancedOrderLegRequest{limit, stopLimit}},
		{"a TAKE_PROFIT trigger order", []AdvancedOrderLegRequest{limit, takeProfit}},
		{"a TAKE_PROFIT_LIMIT trigger order", []AdvancedOrderLegRequest{limit, takeProfitLimit}},
		{"the trigger order first", []AdvancedOrderLegRequest{stopLoss, limit}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := oneCancelsTheOtherParams(tc.orders)
			require.NoErrorf(t, err, "oneCancelsTheOtherParams must not error for %s", tc.name)
			assert.Equalf(t, &advancedOrderListParams[AdvancedOrderLegRequest]{OrderList: tc.orders}, got, "oneCancelsTheOtherParams should return the params of %s", tc.name)
		})
	}
}

func TestOneTriggersTheOtherParams(t *testing.T) {
	t.Parallel()
	for _, tc := range oneTriggersTheOtherRefusals() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := oneTriggersTheOtherParams(tc.orders)
			assert.ErrorIsf(t, err, tc.err, "oneTriggersTheOtherParams should refuse %s", tc.name)
			assert.Nilf(t, got, "oneTriggersTheOtherParams should return no params for %s", tc.name)
		})
	}
	limit, stopLoss := otoOrders()[0], otoOrders()[1]
	indexPrice := stopLoss
	indexPrice.ReferencePriceType = ReferencePriceTypeIndexPrice
	stopLimit := stopLoss
	stopLimit.OrderType, stopLimit.Price = OrderTypeStopLimit, 29900
	for _, tc := range []struct {
		name   string
		orders []AdvancedOrderLegRequest
	}{
		{"the docs' OTO", otoOrders()},
		{
			"a MARKET buy spending a notional and a TAKE_PROFIT_LIMIT comparing the last price",
			[]AdvancedOrderLegRequest{
				{InstrumentName: "BTC_USD", Side: SideBuy, OrderType: OrderTypeMarket, Notional: 8000},
				{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeTakeProfitLimit, Quantity: 0.1, Price: 84000, ReferencePrice: 84100, ReferencePriceType: ReferencePriceTypeLastPrice},
			},
		},
		{"a trigger order comparing the index price, which only an OCO refuses", []AdvancedOrderLegRequest{limit, indexPrice}},
		{"a STOP_LIMIT trigger order", []AdvancedOrderLegRequest{limit, stopLimit}},
		{"the trigger order first", []AdvancedOrderLegRequest{stopLoss, limit}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := oneTriggersTheOtherParams(tc.orders)
			require.NoErrorf(t, err, "oneTriggersTheOtherParams must not error for %s", tc.name)
			assert.Equalf(t, &advancedOrderListParams[AdvancedOrderLegRequest]{OrderList: tc.orders}, got, "oneTriggersTheOtherParams should return the params of %s", tc.name)
		})
	}
}

func TestOneTriggersAOneCancelsTheOtherParams(t *testing.T) {
	t.Parallel()
	limit, stopLoss, takeProfit := otocoOrders()[0], otocoOrders()[1], otocoOrders()[2]
	market := limit
	market.OrderType, market.Price = OrderTypeMarket, 0
	stopLimit := stopLoss
	stopLimit.OrderType, stopLimit.Price = OrderTypeStopLimit, 29900
	takeProfitLimit := takeProfit
	takeProfitLimit.OrderType, takeProfitLimit.Price = OrderTypeTakeProfitLimit, 120100
	refusals := append(
		oneTriggersAOneCancelsTheOtherRefusals(),
		contingencyRefusal{name: "an OTOCO of a LIMIT order, a MARKET order and a stop loss", orders: []AdvancedOrderLegRequest{limit, market, stopLoss}, err: errInvalidContingencyOrders},
		contingencyRefusal{name: "an OTOCO of a STOP_LOSS and a STOP_LIMIT", orders: []AdvancedOrderLegRequest{limit, stopLoss, stopLimit}, err: errInvalidContingencyOrders},
		contingencyRefusal{name: "an OTOCO of a TAKE_PROFIT and a TAKE_PROFIT_LIMIT", orders: []AdvancedOrderLegRequest{limit, takeProfit, takeProfitLimit}, err: errInvalidContingencyOrders},
	)
	for _, tc := range refusals {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := oneTriggersAOneCancelsTheOtherParams(tc.orders)
			assert.ErrorIsf(t, err, tc.err, "oneTriggersAOneCancelsTheOtherParams should refuse %s", tc.name)
			assert.Nilf(t, got, "oneTriggersAOneCancelsTheOtherParams should return no params for %s", tc.name)
		})
	}
	lastPrice := takeProfitLimit
	lastPrice.ReferencePriceType = ReferencePriceTypeLastPrice
	for _, tc := range []struct {
		name   string
		orders []AdvancedOrderLegRequest
	}{
		{"the docs' OTOCO", otocoOrders()},
		{"a MARKET order, a STOP_LIMIT and a TAKE_PROFIT_LIMIT comparing the last price", []AdvancedOrderLegRequest{market, stopLimit, lastPrice}},
		{"the trigger orders first", []AdvancedOrderLegRequest{takeProfit, stopLoss, limit}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := oneTriggersAOneCancelsTheOtherParams(tc.orders)
			require.NoErrorf(t, err, "oneTriggersAOneCancelsTheOtherParams must not error for %s", tc.name)
			assert.Equalf(t, &advancedOrderListParams[AdvancedOrderLegRequest]{OrderList: tc.orders}, got, "oneTriggersAOneCancelsTheOtherParams should return the params of %s", tc.name)
		})
	}
}

func TestNewAdvancedListIDParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, listID string
		exp          *advancedListIDParams
		err          error
	}{
		{name: "an empty list ID", err: errListIDRequired},
		{name: "a list ID", listID: "6498090546073120100", exp: &advancedListIDParams{ListID: "6498090546073120100"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := newAdvancedListIDParams(tc.listID)
			assert.ErrorIsf(t, err, tc.err, "newAdvancedListIDParams should return the expected error for %s", tc.name)
			assert.Equalf(t, tc.exp, got, "newAdvancedListIDParams should return the params of %s", tc.name)
		})
	}
}

func TestAdvancedOrderIDRequestValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *AdvancedOrderIDRequest
		err  error
	}{
		{"a nil request", nil, common.ErrNilPointer},
		{"a request without an order ID or client order ID", &AdvancedOrderIDRequest{}, errOrderIDRequired},
		{"an order ID", &AdvancedOrderIDRequest{OrderID: "5755600460443882762"}, nil},
		{"a client order ID", &AdvancedOrderIDRequest{ClientOrderID: "c5f682ed-7108-4f1c-b755-972fcdca0f02"}, nil},
		{"an order ID and a client order ID", &AdvancedOrderIDRequest{OrderID: "5755600460443882762", ClientOrderID: "c5f682ed-7108-4f1c-b755-972fcdca0f02"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, tc.req.validate(), tc.err, "validate should return the expected error for %s", tc.name)
		})
	}
}

func TestCancelAllAdvancedOrdersRequestValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range cancelAllAdvancedOrdersRefusals {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, tc.req.validate(), tc.err, "validate should refuse %s", tc.name)
		})
	}
	for _, tc := range []struct {
		name string
		req  *CancelAllAdvancedOrdersRequest
		err  error
	}{
		{"a lower-case order type", &CancelAllAdvancedOrdersRequest{OrderType: "all"}, errInvalidCancelOrderType},
		{"a lower-case position type", &CancelAllAdvancedOrdersRequest{InstrumentName: "BTCUSD-PERP", PositionType: "cross_margin"}, errInvalidPositionType},
		{"a cancel of every instrument's advanced orders", &CancelAllAdvancedOrdersRequest{}, nil},
		{"a cancel of an instrument's advanced orders", &CancelAllAdvancedOrdersRequest{InstrumentName: "BTC_USD"}, nil},
		{"a cancel of the limit orders of advanced orders", &CancelAllAdvancedOrdersRequest{InstrumentName: "BTC_USD", OrderType: CancelOrderTypeLimit}, nil},
		{"a cancel of every instrument's trigger orders", &CancelAllAdvancedOrdersRequest{OrderType: CancelOrderTypeTrigger}, nil},
		{"a cancel of every order", &CancelAllAdvancedOrdersRequest{OrderType: CancelOrderTypeAll}, nil},
		{"a cancel of a cross margin position's orders", &CancelAllAdvancedOrdersRequest{InstrumentName: "BTCUSD-PERP", PositionType: PositionTypeCrossMargin}, nil},
		{
			"a cancel of every order of an isolated position",
			&CancelAllAdvancedOrdersRequest{InstrumentName: "BTCUSD-PERP", OrderType: CancelOrderTypeAll, PositionType: PositionTypeIsolatedMargin},
			nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, tc.req.validate(), tc.err, "validate should return the expected error for %s", tc.name)
		})
	}
}

func TestGetAdvancedOrderHistoryRequestParams(t *testing.T) {
	t.Parallel()
	startTime, endTime := time.Unix(0, 1792713600000081486), time.Unix(0, 1792799991058211357)
	for _, tc := range []struct {
		name string
		req  *GetAdvancedOrderHistoryRequest
		exp  *advancedOrderHistoryParams
		err  error
	}{
		{name: "a nil request", err: common.ErrNilPointer},
		{name: "a start time after the end time", req: &GetAdvancedOrderHistoryRequest{StartTime: endTime, EndTime: startTime}, err: common.ErrStartAfterEnd},
		{name: "a start time at the end time", req: &GetAdvancedOrderHistoryRequest{StartTime: startTime, EndTime: startTime}, err: common.ErrStartEqualsEnd},
		{
			name: "a window starting in the future",
			req:  &GetAdvancedOrderHistoryRequest{StartTime: time.Now().Add(time.Hour), EndTime: time.Now().Add(2 * time.Hour)},
			err:  common.ErrStartAfterTimeNow,
		},
		{name: "a limit above 100", req: &GetAdvancedOrderHistoryRequest{Limit: maxAdvancedOrderHistoryLimit + 1}, err: errInvalidCount},
		{name: "a start time before 1970", req: &GetAdvancedOrderHistoryRequest{StartTime: time.Unix(0, -1)}, err: errTimeBeforeUnixEpoch},
		{name: "an end time before 1970", req: &GetAdvancedOrderHistoryRequest{EndTime: time.Unix(0, -1)}, err: errTimeBeforeUnixEpoch},
		{name: "a window before 1970", req: &GetAdvancedOrderHistoryRequest{StartTime: time.Unix(0, -2), EndTime: time.Unix(0, -1)}, err: errTimeBeforeUnixEpoch},
		{name: "a request with Crypto.com's defaults", req: &GetAdvancedOrderHistoryRequest{}, exp: &advancedOrderHistoryParams{}},
		{
			name: "a request with every field",
			req:  &GetAdvancedOrderHistoryRequest{InstrumentName: "BTCUSD-PERP", StartTime: startTime, EndTime: endTime, Limit: maxAdvancedOrderHistoryLimit},
			exp:  &advancedOrderHistoryParams{InstrumentName: "BTCUSD-PERP", StartTime: 1792713600000081486, EndTime: 1792799991058211357, Limit: maxAdvancedOrderHistoryLimit},
		},
		{name: "a start time alone", req: &GetAdvancedOrderHistoryRequest{StartTime: startTime}, exp: &advancedOrderHistoryParams{StartTime: 1792713600000081486}},
		{name: "an end time alone", req: &GetAdvancedOrderHistoryRequest{EndTime: endTime}, exp: &advancedOrderHistoryParams{EndTime: 1792799991058211357}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := tc.req.params()
			assert.ErrorIsf(t, err, tc.err, "params should return the expected error for %s", tc.name)
			assert.Equalf(t, tc.exp, got, "params should return the params of %s", tc.name)
		})
	}
}

func TestCreateAdvancedOrderListParams(t *testing.T) {
	t.Parallel()
	order := advancedTriggerOrder()
	for _, tc := range []struct {
		name   string
		orders []CreateAdvancedOrderRequest
		err    error
	}{
		{name: "an empty list", err: errOrderListEmpty},
		{name: "a list of 11 orders", orders: slices.Repeat([]CreateAdvancedOrderRequest{order}, maxAdvancedOrderListLength+1), err: errOrderListTooLong},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := createAdvancedOrderListParams(tc.orders)
			assert.ErrorIsf(t, err, tc.err, "createAdvancedOrderListParams should refuse %s", tc.name)
			assert.Nilf(t, got, "createAdvancedOrderListParams should return no params for %s", tc.name)
		})
	}
	for _, tc := range triggerOrderRefusals {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			refused := advancedTriggerOrder()
			tc.change(&refused)
			got, err := createAdvancedOrderListParams([]CreateAdvancedOrderRequest{advancedTriggerOrder(), refused})
			assert.ErrorIsf(t, err, tc.err, "createAdvancedOrderListParams should refuse a list holding %s", tc.name)
			assert.ErrorContainsf(t, err, "order 1:", "createAdvancedOrderListParams should name the index of %s", tc.name)
			assert.Nilf(t, got, "createAdvancedOrderListParams should return no params for a list holding %s", tc.name)
		})
	}
	stopLoss := advancedOrderListOrders()[0]
	for _, tc := range []struct {
		name   string
		orders []CreateAdvancedOrderRequest
	}{
		{"a list of one order", []CreateAdvancedOrderRequest{stopLoss}},
		{"the docs' list of two orders", advancedOrderListOrders()},
		{"a list of 10 orders", slices.Repeat([]CreateAdvancedOrderRequest{stopLoss}, maxAdvancedOrderListLength)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := createAdvancedOrderListParams(tc.orders)
			require.NoErrorf(t, err, "createAdvancedOrderListParams must not error for %s", tc.name)
			assert.Equalf(t, &advancedOrderListParams[CreateAdvancedOrderRequest]{OrderList: tc.orders}, got, "createAdvancedOrderListParams should return the params of %s", tc.name)
		})
	}
}

func TestCancelAdvancedOrderListParams(t *testing.T) {
	t.Parallel()
	for _, tc := range cancelAdvancedOrderListRefusals() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := cancelAdvancedOrderListParams(tc.orders)
			assert.ErrorIsf(t, err, tc.err, "cancelAdvancedOrderListParams should refuse %s", tc.name)
			assert.Nilf(t, got, "cancelAdvancedOrderListParams should return no params for %s", tc.name)
		})
	}
	t.Run("the index of an order without an order ID or client order ID", func(t *testing.T) {
		t.Parallel()
		_, err := cancelAdvancedOrderListParams([]AdvancedOrderIDRequest{{}, {OrderID: "289991951579667814"}})
		assert.ErrorContains(t, err, "order 0:", "cancelAdvancedOrderListParams should name the index of the order without an order ID or client order ID")
	})
	for _, tc := range []struct {
		name   string
		orders []AdvancedOrderIDRequest
	}{
		{"a list of one order ID", []AdvancedOrderIDRequest{{OrderID: "289991951579667814"}}},
		{
			"a list of an order ID and client order ID, and a client order ID",
			[]AdvancedOrderIDRequest{{OrderID: "289991951579667814", ClientOrderID: "my_sl_order"}, {ClientOrderID: "my_tp_order"}},
		},
		{"a list of 10 orders", slices.Repeat([]AdvancedOrderIDRequest{{ClientOrderID: "my_tp_order"}}, maxAdvancedOrderListLength)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := cancelAdvancedOrderListParams(tc.orders)
			require.NoErrorf(t, err, "cancelAdvancedOrderListParams must not error for %s", tc.name)
			assert.Equalf(t, &advancedOrderListParams[AdvancedOrderIDRequest]{OrderList: tc.orders}, got, "cancelAdvancedOrderListParams should return the params of %s", tc.name)
		})
	}
}
