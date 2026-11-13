package cryptodotcom

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	tradingTestOrderID              = "6530219466236720401"
	tradingTestOrderClientOrderID   = "c5f682ed-7108-4f1c-b755-972fcdca0f02"
	tradingTestAccountID            = "52e7c00f-1324-5a6a-bf6e-de445bde21a5"
	tradingTestUpdateUserID         = "fd797356-55db-48c2-a44d-157aabf702e8"
	tradingTestIsolationID          = "6530219466236720402"
	tradingTestCancelAllInstrument  = "ETHUSD-PERP"
	tradingTestOrderHistoryStart    = 1610905028000081486
	tradingTestOrderHistoryEnd      = 1613570791058211357
	tradingTestTradesWindowStart    = 1619089031996081486
	tradingTestTradesWindowEnd      = 1619200052124211357
	tradingTestOrderIDResult        = `{"order_id":"6530219466236720401","client_oid":"c5f682ed-7108-4f1c-b755-972fcdca0f02"}`
	tradingTestOrderListEntryResult = `[{"code":0,"index":0,"order_id":"1","client_oid":"c"}]`
)

// tradingTestOrder returns the order the tests create: a post-only buy of BTCUSD-PERP's least quantity far below its
// market, so in live mode it rests until the test cancels it
func tradingTestOrder() *CreateOrderRequest {
	return &CreateOrderRequest{
		InstrumentName:        "BTCUSD-PERP",
		Side:                  SideBuy,
		OrderType:             OrderTypeLimit,
		Price:                 50000.5,
		Quantity:              0.0001,
		ClientOrderID:         tradingTestOrderClientOrderID,
		TimeInForce:           TimeInForceGoodTillCancel,
		ExecutionInstructions: []string{ExecutionInstructionPostOnly},
	}
}

// tradingTestMarketBuy turns an order into a market buy spending a notional rather than a quantity
func tradingTestMarketBuy(r *CreateOrderRequest) {
	r.OrderType, r.Price, r.Quantity, r.Notional = OrderTypeMarket, 0, 0, 5
}

// tradingTestOrderList returns the order list the tests create: sells of CRO_USD far above its market
func tradingTestOrderList() []CreateOrderRequest {
	return []CreateOrderRequest{
		{InstrumentName: "CRO_USD", Side: SideSell, OrderType: OrderTypeLimit, Quantity: 10, Price: 0.12, ClientOrderID: "api_leg1"},
		{InstrumentName: "CRO_USD", Side: SideSell, OrderType: OrderTypeLimit, Quantity: 20, Price: 0.122, ClientOrderID: "api_leg2"},
	}
}

// tradingTestOrderListParams is the params creating tradingTestOrderList
const tradingTestOrderListParams = `{"contingency_type":"LIST","order_list":[{"instrument_name":"CRO_USD","side":"SELL","type":"LIMIT","price":"0.12","quantity":"10","client_oid":"api_leg1"},{"instrument_name":"CRO_USD","side":"SELL","type":"LIMIT","price":"0.122","quantity":"20","client_oid":"api_leg2"}]}`

// tradingTestOrderParams is the params creating tradingTestOrder
const tradingTestOrderParams = `{"instrument_name":"BTCUSD-PERP","side":"BUY","type":"LIMIT","price":"50000.5","quantity":"0.0001","client_oid":"c5f682ed-7108-4f1c-b755-972fcdca0f02","time_in_force":"GOOD_TILL_CANCEL","exec_inst":["POST_ONLY"]}`

// createOrderRefusals are orders private/create-order refuses, each tradingTestOrder changed, with the error refusing
// it; the order lists and batches refuse them too. An order without an instrument is not among them, as a batch's
// orders inherit the batch's
var createOrderRefusals = []struct {
	name   string
	modify func(*CreateOrderRequest)
	err    error
}{
	{"a lower-case side", func(r *CreateOrderRequest) { r.Side = "buy" }, errInvalidSide},
	{"a trigger order", func(r *CreateOrderRequest) { r.OrderType = OrderTypeStopLimit }, errInvalidOrderType},
	{"an order without a type", func(r *CreateOrderRequest) { r.OrderType = "" }, errInvalidOrderType},
	{"a limit order without a price", func(r *CreateOrderRequest) { r.Price = 0 }, errInvalidPrice},
	{"a market order with a price", func(r *CreateOrderRequest) { r.OrderType = OrderTypeMarket }, errInvalidPrice},
	{"an order without a quantity", func(r *CreateOrderRequest) { r.Quantity = 0 }, errInvalidQuantity},
	{"a negative quantity", func(r *CreateOrderRequest) { r.Quantity = -0.0001 }, errInvalidQuantity},
	{"a limit order with a notional", func(r *CreateOrderRequest) { r.Quantity, r.Notional = 0, 5 }, errInvalidNotional},
	{"a market sell with a notional", func(r *CreateOrderRequest) { tradingTestMarketBuy(r); r.Side = SideSell }, errInvalidNotional},
	{"a negative notional", func(r *CreateOrderRequest) { tradingTestMarketBuy(r); r.Notional = -5 }, errInvalidNotional},
	{"a notional with a quantity", func(r *CreateOrderRequest) { tradingTestMarketBuy(r); r.Quantity = 0.0001 }, errInvalidNotional},
	{"a 37 character client order ID", func(r *CreateOrderRequest) { r.ClientOrderID = strings.Repeat("c", 37) }, errClientOrderIDTooLong},
	{"an abbreviated time in force", func(r *CreateOrderRequest) { r.TimeInForce = "GTC" }, errInvalidTimeInForce},
	{"the liquidation instruction, which is only reported", func(r *CreateOrderRequest) {
		r.ExecutionInstructions = []string{ExecutionInstructionLiquidation}
	}, errInvalidExecutionInstruction},
	{"both post-only instructions", func(r *CreateOrderRequest) {
		r.ExecutionInstructions = []string{ExecutionInstructionPostOnly, ExecutionInstructionSmartPostOnly}
	}, errInvalidExecutionInstruction},
	{"a post-only order not good till cancelled", func(r *CreateOrderRequest) { r.TimeInForce = TimeInForceImmediateOrCancel }, errInvalidTimeInForce},
	{"an unknown spot margin mode", func(r *CreateOrderRequest) { r.SpotMargin = "CROSS" }, errInvalidSpotMargin},
	{"an unknown self-trade prevention scope", func(r *CreateOrderRequest) { r.SelfTradePreventionScope = "X" }, errInvalidSelfTradePreventionScope},
	{"an unknown self-trade prevention instruction", func(r *CreateOrderRequest) {
		r.SelfTradePreventionScope, r.SelfTradePreventionInstruction = SelfTradePreventionScopeSubAccount, "X"
	}, errInvalidSelfTradePreventionInstruction},
	{"a self-trade prevention scope without an instruction", func(r *CreateOrderRequest) {
		r.SelfTradePreventionScope = SelfTradePreventionScopeMasterOrSubAccount
	}, errInvalidSelfTradePreventionInstruction},
	{"a self-trade prevention ID above 32767", func(r *CreateOrderRequest) {
		r.SelfTradePreventionScope, r.SelfTradePreventionInstruction, r.SelfTradePreventionID = SelfTradePreventionScopeSubAccount, SelfTradePreventionCancelMaker, 32768
	}, errInvalidSelfTradePreventionID},
	{"a self-trade prevention ID without a scope", func(r *CreateOrderRequest) { r.SelfTradePreventionID = 100 }, errInvalidSelfTradePreventionID},
	{"a negative isolated margin amount", func(r *CreateOrderRequest) { r.IsolatedMarginAmount = -50 }, errInvalidAmount},
	{"a negative leverage", func(r *CreateOrderRequest) { r.Leverage = -10 }, errInvalidLeverage},
}

// amendOrderRefusals are amend requests private/amend-order refuses, with the error refusing each
var amendOrderRefusals = []struct {
	name string
	req  *AmendOrderRequest
	err  error
}{
	{"a nil request", nil, common.ErrNilPointer},
	{"a request naming no order", &AmendOrderRequest{NewPrice: 50000, NewQuantity: 0.0002}, errOrderIDRequired},
	{"a request naming the order by its new client order ID only", &AmendOrderRequest{ClientOrderID: "c", NewPrice: 50000, NewQuantity: 0.0002}, errOrderIDRequired},
	{"a 37 character client order ID", &AmendOrderRequest{OrderID: tradingTestOrderID, ClientOrderID: strings.Repeat("c", 37), NewPrice: 50000, NewQuantity: 0.0002}, errClientOrderIDTooLong},
	{"a request without a new price", &AmendOrderRequest{OrderID: tradingTestOrderID, NewQuantity: 0.0002}, errInvalidPrice},
	{"a negative new price", &AmendOrderRequest{OrderID: tradingTestOrderID, NewPrice: -50000, NewQuantity: 0.0002}, errInvalidPrice},
	{"a request without a new quantity", &AmendOrderRequest{OrderID: tradingTestOrderID, NewPrice: 50000}, errInvalidQuantity},
	{"a negative new quantity", &AmendOrderRequest{OrderID: tradingTestOrderID, NewPrice: 50000, NewQuantity: -0.0002}, errInvalidQuantity},
}

// cancelOrderRefusals are cancel requests private/cancel-order refuses, with the error refusing each
var cancelOrderRefusals = []struct {
	name string
	req  *CancelOrderRequest
	err  error
}{
	{"a nil request", nil, common.ErrNilPointer},
	{"a request naming no order", &CancelOrderRequest{}, errOrderIDRequired},
}

// cancelAllOrdersRefusals are cancel-all requests private/cancel-all-orders refuses, with the error refusing each
var cancelAllOrdersRefusals = []struct {
	name string
	req  *CancelAllOrdersRequest
	err  error
}{
	{"a nil request", nil, common.ErrNilPointer},
	{"trigger orders, which the advanced order methods cancel", &CancelAllOrdersRequest{OrderType: "TRIGGER"}, errInvalidOrderType},
	{"an unknown type", &CancelAllOrdersRequest{OrderType: OrderTypeMarket}, errInvalidOrderType},
}

// closePositionRefusals are close requests private/close-position refuses, with the error refusing each
var closePositionRefusals = []struct {
	name string
	req  *ClosePositionRequest
	err  error
}{
	{"a nil request", nil, common.ErrNilPointer},
	{"a request without an instrument", &ClosePositionRequest{OrderType: OrderTypeMarket}, errInstrumentNameRequired},
	{"a request without a type", &ClosePositionRequest{InstrumentName: "BTCUSD-PERP"}, errInvalidOrderType},
	{"a trigger order", &ClosePositionRequest{InstrumentName: "BTCUSD-PERP", OrderType: OrderTypeStopLoss}, errInvalidOrderType},
	{"a limit order without a price", &ClosePositionRequest{InstrumentName: "BTCUSD-PERP", OrderType: OrderTypeLimit}, errInvalidPrice},
	{"a market order with a price", &ClosePositionRequest{InstrumentName: "BTCUSD-PERP", OrderType: OrderTypeMarket, Price: 30000}, errInvalidPrice},
	{"a negative quantity", &ClosePositionRequest{InstrumentName: "BTCUSD-PERP", OrderType: OrderTypeMarket, Quantity: -0.0001}, errInvalidQuantity},
}

// cancelOrderListRefusals are lists private/cancel-order-list refuses, with the error refusing each
var cancelOrderListRefusals = []struct {
	name   string
	orders []CancelOrderListEntryRequest
	err    error
}{
	{"an empty list", nil, errOrderListEmpty},
	{"a list of 11 orders", make([]CancelOrderListEntryRequest, 11), errOrderListTooLong},
	{"an order named by neither ID", []CancelOrderListEntryRequest{{OrderID: tradingTestOrderID}, {InstrumentName: "CRO_USD"}}, errOrderIDRequired},
}

// newTradingTestClientOrderID returns a client order ID for an order a live test places, unique so that orders the
// tests place at once are not refused as duplicates
func newTradingTestClientOrderID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}

// placeTradingTestOrder places tradingTestOrder, with a client order ID of its own, for a live test to act on, and
// returns its order ID
func placeTradingTestOrder(t *testing.T) string {
	t.Helper()
	req := tradingTestOrder()
	req.ClientOrderID = newTradingTestClientOrderID()
	resp, err := e.CreateOrder(t.Context(), req)
	require.NoError(t, err, "CreateOrder must place the order the test acts on")
	return resp.OrderID
}

// cancelTradingOrderAfterTest cancels an order a live test placed once the test ends
func cancelTradingOrderAfterTest(t *testing.T, orderID string) {
	t.Helper()
	t.Cleanup(func() {
		_, err := e.CancelExistingOrder(context.WithoutCancel(t.Context()), &CancelOrderRequest{OrderID: orderID})
		assert.NoErrorf(t, err, "CancelExistingOrder should cancel order %s, which the test placed", orderID)
	})
}

// cancelTradingOrderIfTestFails cancels an order a live test placed to cancel itself, should the test fail before it
// does
func cancelTradingOrderIfTestFails(t *testing.T, orderID string) {
	t.Helper()
	t.Cleanup(func() {
		if !t.Failed() {
			return
		}
		if _, err := e.CancelExistingOrder(context.WithoutCancel(t.Context()), &CancelOrderRequest{OrderID: orderID}); err != nil {
			t.Logf("CancelExistingOrder could not cancel order %s, which the test placed: %v", orderID, err)
		}
	})
}

// assertListOutcomes checks a live list answer holds one outcome for each of its n orders, by index, and that each order
// was placed or cancelled as the test asked
func assertListOutcomes(t *testing.T, result []OrderListEntryResponse, n int, method, action string) {
	t.Helper()
	require.Lenf(t, result, n, "%s must report one outcome for each order", method)
	indexes := make([]int64, n)
	for i := range result {
		indexes[i] = result[i].Index.Int64()
		assert.Zerof(t, result[i].Code.Int64(), "%s should %s order %d: %s", method, action, indexes[i], result[i].Message)
	}
	want := make([]int64, n)
	for i := range want {
		want[i] = int64(i)
	}
	assert.ElementsMatchf(t, want, indexes, "%s should report each order's outcome once", method)
}

// The codes Crypto.com's docs give for refusing a request for what the account does not hold: an order or list ID it
// does not know (NOT_FOUND, INVALID_ORDERID), a position (NO_POSITION, POSITION_NO_EXIST) and an isolation ID
// (INVALID_ISOLATION_ID)
var (
	unknownOrderCodes       = []int64{40401, 212}
	noPositionCodes         = []int64{201, 317}
	unknownIsolationIDCodes = []int64{613}
)

// assertRefusedFor checks Crypto.com refused a request with one of codes, for what the request names, which the account
// does not hold, rather than refusing the request itself, as it does for its credentials or a malformed request
func assertRefusedFor(t *testing.T, err error, codes []int64, name string) {
	t.Helper()
	var apiErr *APIError
	require.ErrorAsf(t, err, &apiErr, "%s must return Crypto.com's refusal", name)
	assert.Containsf(t, codes, apiErr.Code, "%s should be refused for what it names, not for the request: %s", name, apiErr)
}

// closeTestPositionRequest returns a request closing part of the account's cross margin position on BTCUSD-PERP with a
// LIMIT order priced away from the market, which rests until the test cancels it: a long position closes with a sell
// above the market and a short one with a buy below it, at the prices the other live tests use. It returns nil when the
// account holds no such position
func closeTestPositionRequest(t *testing.T) *ClosePositionRequest {
	t.Helper()
	positions, err := e.GetPositions(t.Context(), &PositionsRequest{InstrumentName: "BTCUSD-PERP"})
	require.NoError(t, err, "GetPositions must return the account's positions")
	i := slices.IndexFunc(positions, func(p PositionResponse) bool {
		return p.InstrumentName == "BTCUSD-PERP" && !isIsolated(p.IsolationID) && !p.Quantity.IsZero()
	})
	if i < 0 {
		return nil
	}
	price := types.Number(120000)
	if positions[i].Quantity.Float64() < 0 {
		price = tradingTestOrder().Price
	}
	return &ClosePositionRequest{InstrumentName: "BTCUSD-PERP", OrderType: OrderTypeLimit, Price: price, Quantity: 0.0001}
}

// isolatedTestPosition returns one of the account's isolated positions, or nil when it holds none
func isolatedTestPosition(t *testing.T) *IsolatedPositionBalance {
	t.Helper()
	balances, err := e.GetUserBalance(t.Context())
	require.NoError(t, err, "GetUserBalance must return the account's isolated positions")
	return findIsolatedPosition(balances, func(p *IsolatedPositionBalance) bool { return isIsolated(p.IsolationID) })
}

func TestCreateOrder(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same method
	t.Parallel()
	_, err := e.CreateOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateOrder must refuse a nil request")
	noInstrument := tradingTestOrder()
	noInstrument.InstrumentName = ""
	_, err = e.CreateOrder(t.Context(), noInstrument)
	require.ErrorIs(t, err, errInstrumentNameRequired, "CreateOrder must refuse an order without an instrument")
	// The cases run in order on e, before the CreateOrder calls below, so each also checks that a refusal leaves
	// nothing behind
	for _, tc := range createOrderRefusals {
		t.Run(tc.name, func(t *testing.T) {
			req := tradingTestOrder()
			tc.modify(req)
			_, err := e.CreateOrder(t.Context(), req)
			assert.ErrorIsf(t, err, tc.err, "CreateOrder should refuse %s", tc.name)
		})
	}

	// Serial, so the cases still run before the CreateOrder calls below rather than after them
	for _, tc := range []struct {
		name   string
		req    *CreateOrderRequest
		params string
	}{
		{
			name: "every optional param",
			req: &CreateOrderRequest{
				InstrumentName:                 "BTCUSD-PERP",
				Side:                           SideBuy,
				OrderType:                      OrderTypeLimit,
				Price:                          50000.5,
				Quantity:                       0.0001,
				ClientOrderID:                  strings.Repeat("c", 36),
				TimeInForce:                    TimeInForceGoodTillCancel,
				ExecutionInstructions:          []string{ExecutionInstructionPostOnly, ExecutionInstructionReduceOnly, ExecutionInstructionIsolatedMargin, ExecutionInstructionMarginOrder},
				SpotMargin:                     SpotMarginMargin,
				FeeInstrumentName:              currency.NewCode("cro"),
				SelfTradePreventionID:          32767,
				SelfTradePreventionInstruction: SelfTradePreventionCancelBoth,
				SelfTradePreventionScope:       SelfTradePreventionScopeSubAccount,
				IsolatedMarginAmount:           50,
				IsolationID:                    tradingTestIsolationID,
				Leverage:                       10,
			},
			params: `{"instrument_name":"BTCUSD-PERP","side":"BUY","type":"LIMIT","price":"50000.5","quantity":"0.0001","client_oid":"cccccccccccccccccccccccccccccccccccc","time_in_force":"GOOD_TILL_CANCEL","exec_inst":["POST_ONLY","REDUCE_ONLY","ISOLATED_MARGIN","MARGIN_ORDER"],"spot_margin":"MARGIN","fee_instrument_name":"CRO","stp_id":"32767","stp_inst":"B","stp_scope":"S","isolated_margin_amount":"50","isolation_id":"6530219466236720402","leverage":"10"}`,
		},
		{
			name:   "a market buy spending a notional",
			req:    &CreateOrderRequest{InstrumentName: "BTC_USD", Side: SideBuy, OrderType: OrderTypeMarket, Notional: 100},
			params: `{"instrument_name":"BTC_USD","side":"BUY","type":"MARKET","notional":"100"}`,
		},
		{
			name:   "a market sell of only the required params",
			req:    &CreateOrderRequest{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeMarket, Quantity: 0.00001},
			params: `{"instrument_name":"BTC_USD","side":"SELL","type":"MARKET","quantity":"0.00001"}`,
		},
		{
			name:   "a limit order of only the required params, priced in plain notation",
			req:    &CreateOrderRequest{InstrumentName: "ONE_USDT", Side: SideBuy, OrderType: OrderTypeLimit, Price: 0.0000013, Quantity: 1000},
			params: `{"instrument_name":"ONE_USDT","side":"BUY","type":"LIMIT","price":"0.0000013","quantity":"1000"}`,
		},
		{
			name: "a post-only order resting by default and resetting self-trade prevention",
			req: &CreateOrderRequest{
				InstrumentName:           "BTCUSD-PERP",
				Side:                     SideBuy,
				OrderType:                OrderTypeLimit,
				Price:                    50000.5,
				Quantity:                 0.0001,
				ExecutionInstructions:    []string{ExecutionInstructionPostOnly},
				SelfTradePreventionScope: SelfTradePreventionScopeReset,
			},
			params: `{"instrument_name":"BTCUSD-PERP","side":"BUY","type":"LIMIT","price":"50000.5","quantity":"0.0001","exec_inst":["POST_ONLY"],"stp_scope":"D"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/create-order", tc.params, testResponse("private/create-order", tradingTestOrderIDResult))
			_, err := ex.CreateOrder(t.Context(), tc.req)
			assert.NoErrorf(t, err, "CreateOrder should not error for %s", tc.name)
		})
	}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/create-order","code":0,"result":null}`).CreateOrder(t.Context(), tradingTestOrder())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateOrder should error for a null result")
	_, err = newUnavailableServerExchange(t).CreateOrder(t.Context(), tradingTestOrder())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateOrder should return a request failure")

	skipLiveUnlessCanManipulate(t)
	req := tradingTestOrder()
	if !mockTests {
		req.ClientOrderID = newTradingTestClientOrderID()
	}
	result, err := e.CreateOrder(t.Context(), req)
	require.NoError(t, err, "CreateOrder must not error")
	if mockTests {
		exp := &OrderIDResponse{OrderID: tradingTestOrderID, ClientOrderID: tradingTestOrderClientOrderID}
		assert.Equal(t, exp, result, "CreateOrder should decode every field")
	} else {
		cancelTradingOrderAfterTest(t, result.OrderID)
		assert.Equal(t, req.ClientOrderID, result.ClientOrderID, "CreateOrder should return the order's client order ID")
	}
}

func TestAmendOrder(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same method
	t.Parallel()
	// The cases run in order on e, before the AmendOrder calls below, so each also checks that a refusal leaves nothing
	// behind
	for _, tc := range amendOrderRefusals {
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.AmendOrder(t.Context(), tc.req)
			assert.ErrorIsf(t, err, tc.err, "AmendOrder should refuse %s", tc.name)
		})
	}

	// Serial, so the cases still run before the AmendOrder calls below rather than after them
	for _, tc := range []struct {
		name   string
		req    *AmendOrderRequest
		params string
	}{
		{
			name:   "every optional param",
			req:    &AmendOrderRequest{OrderID: tradingTestOrderID, OriginalClientOrderID: tradingTestOrderClientOrderID, ClientOrderID: strings.Repeat("c", 36), NewPrice: 50000, NewQuantity: 0.0002},
			params: `{"order_id":"6530219466236720401","orig_client_oid":"c5f682ed-7108-4f1c-b755-972fcdca0f02","client_oid":"cccccccccccccccccccccccccccccccccccc","new_price":"50000","new_quantity":"0.0002"}`,
		},
		{
			name:   "an order named by its order ID",
			req:    &AmendOrderRequest{OrderID: tradingTestOrderID, NewPrice: 50000, NewQuantity: 0.0002},
			params: `{"order_id":"6530219466236720401","new_price":"50000","new_quantity":"0.0002"}`,
		},
		{
			name:   "an order named by its original client order ID",
			req:    &AmendOrderRequest{OriginalClientOrderID: tradingTestOrderClientOrderID, NewPrice: 50000, NewQuantity: 0.0002},
			params: `{"orig_client_oid":"c5f682ed-7108-4f1c-b755-972fcdca0f02","new_price":"50000","new_quantity":"0.0002"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/amend-order", tc.params, testResponse("private/amend-order", tradingTestOrderIDResult))
			_, err := ex.AmendOrder(t.Context(), tc.req)
			assert.NoErrorf(t, err, "AmendOrder should not error for %s", tc.name)
		})
	}
	req := &AmendOrderRequest{OrderID: tradingTestOrderID, NewPrice: 50000, NewQuantity: 0.0002}
	_, err := newResponseServerExchange(t, `{"id":1,"method":"private/amend-order","code":0,"result":null}`).AmendOrder(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "AmendOrder should error for a null result")
	_, err = newUnavailableServerExchange(t).AmendOrder(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "AmendOrder should return a request failure")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		req.OrderID = placeTradingTestOrder(t)
		// Crypto.com answers an amend with the order ID it was given, as its documented example shows
		cancelTradingOrderAfterTest(t, req.OrderID)
	}
	result, err := e.AmendOrder(t.Context(), req)
	require.NoError(t, err, "AmendOrder must not error")
	if mockTests {
		exp := &OrderIDResponse{OrderID: tradingTestOrderID, ClientOrderID: tradingTestOrderClientOrderID}
		assert.Equal(t, exp, result, "AmendOrder should decode every field")
	} else {
		assert.Equal(t, req.OrderID, result.OrderID, "AmendOrder should return the amended order's ID")
	}
}

func TestCancelExistingOrder(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same method
	t.Parallel()
	// The cases run in order on e, before the CancelExistingOrder calls below, so each also checks that a refusal
	// leaves nothing behind
	for _, tc := range cancelOrderRefusals {
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.CancelExistingOrder(t.Context(), tc.req)
			assert.ErrorIsf(t, err, tc.err, "CancelExistingOrder should refuse %s", tc.name)
		})
	}

	// Serial, so the cases still run before the CancelExistingOrder calls below rather than after them
	for _, tc := range []struct {
		name   string
		req    *CancelOrderRequest
		params string
	}{
		{
			name:   "both IDs",
			req:    &CancelOrderRequest{OrderID: tradingTestOrderID, ClientOrderID: tradingTestOrderClientOrderID},
			params: `{"order_id":"6530219466236720401","client_oid":"c5f682ed-7108-4f1c-b755-972fcdca0f02"}`,
		},
		{
			name:   "an order named by its order ID",
			req:    &CancelOrderRequest{OrderID: tradingTestOrderID},
			params: `{"order_id":"6530219466236720401"}`,
		},
		{
			name:   "an order named by its client order ID",
			req:    &CancelOrderRequest{ClientOrderID: tradingTestOrderClientOrderID},
			params: `{"client_oid":"c5f682ed-7108-4f1c-b755-972fcdca0f02"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/cancel-order", tc.params, testResponse("private/cancel-order", tradingTestOrderIDResult))
			_, err := ex.CancelExistingOrder(t.Context(), tc.req)
			assert.NoErrorf(t, err, "CancelExistingOrder should not error for %s", tc.name)
		})
	}
	req := &CancelOrderRequest{OrderID: tradingTestOrderID}
	_, err := newResponseServerExchange(t, `{"id":1,"method":"private/cancel-order","code":0,"result":null}`).CancelExistingOrder(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelExistingOrder should error for a null result")
	_, err = newUnavailableServerExchange(t).CancelExistingOrder(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelExistingOrder should return a request failure")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		req.OrderID = placeTradingTestOrder(t)
	}
	result, err := e.CancelExistingOrder(t.Context(), req)
	require.NoError(t, err, "CancelExistingOrder must not error")
	if mockTests {
		exp := &OrderIDResponse{OrderID: tradingTestOrderID, ClientOrderID: tradingTestOrderClientOrderID}
		assert.Equal(t, exp, result, "CancelExistingOrder should decode every field")
	} else {
		assert.Equal(t, req.OrderID, result.OrderID, "CancelExistingOrder should return the cancelled order's ID")
	}
}

func TestCancelAllExistingOrders(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same method
	t.Parallel()
	// The cases run in order on e, before the CancelAllExistingOrders calls below, so each also checks that a refusal
	// leaves nothing behind
	for _, tc := range cancelAllOrdersRefusals {
		t.Run(tc.name, func(t *testing.T) {
			assert.ErrorIsf(t, e.CancelAllExistingOrders(t.Context(), tc.req), tc.err, "CancelAllExistingOrders should refuse %s", tc.name)
		})
	}

	// Serial, so the cases still run before the CancelAllExistingOrders calls below rather than after them
	for _, tc := range []struct {
		name   string
		req    *CancelAllOrdersRequest
		params string
	}{
		{name: "every optional param", req: &CancelAllOrdersRequest{InstrumentName: "BTCUSD-PERP", OrderType: CancelOrderTypeAll}, params: `{"instrument_name":"BTCUSD-PERP","type":"ALL"}`},
		{name: "limit orders", req: &CancelAllOrdersRequest{OrderType: CancelOrderTypeLimit}, params: `{"type":"LIMIT"}`},
		{name: "no params", req: &CancelAllOrdersRequest{}, params: `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/cancel-all-orders", tc.params, testResponse("private/cancel-all-orders", ""))
			assert.NoErrorf(t, ex.CancelAllExistingOrders(t.Context(), tc.req), "CancelAllExistingOrders should not error for %s", tc.name)
		})
	}
	req := &CancelAllOrdersRequest{InstrumentName: tradingTestCancelAllInstrument}
	assert.ErrorIs(t, newUnavailableServerExchange(t).CancelAllExistingOrders(t.Context(), req), request.ErrBadStatus, "CancelAllExistingOrders should return a request failure")

	// An instrument the other tests place no orders on, so that cancelling its orders live leaves theirs
	skipLiveUnlessCanManipulate(t)
	require.NoError(t, e.CancelAllExistingOrders(t.Context(), req), "CancelAllExistingOrders must not error")
}

func TestClosePosition(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same method
	t.Parallel()
	// The cases run in order on e, before the ClosePosition calls below, so each also checks that a refusal leaves
	// nothing behind
	for _, tc := range closePositionRefusals {
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.ClosePosition(t.Context(), tc.req)
			assert.ErrorIsf(t, err, tc.err, "ClosePosition should refuse %s", tc.name)
		})
	}

	// Serial, so the cases still run before the ClosePosition calls below rather than after them
	for _, tc := range []struct {
		name   string
		req    *ClosePositionRequest
		params string
	}{
		{
			name:   "every optional param",
			req:    &ClosePositionRequest{InstrumentName: "BTCUSD-PERP", OrderType: OrderTypeLimit, Price: 30000, Quantity: 0.0001, IsolationID: tradingTestIsolationID},
			params: `{"instrument_name":"BTCUSD-PERP","type":"LIMIT","price":"30000","quantity":"0.0001","isolation_id":"6530219466236720402"}`,
		},
		{
			name:   "only the required params",
			req:    &ClosePositionRequest{InstrumentName: "BTCUSD-PERP", OrderType: OrderTypeMarket},
			params: `{"instrument_name":"BTCUSD-PERP","type":"MARKET"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/close-position", tc.params, testResponse("private/close-position", tradingTestOrderIDResult))
			_, err := ex.ClosePosition(t.Context(), tc.req)
			assert.NoErrorf(t, err, "ClosePosition should not error for %s", tc.name)
		})
	}
	req := &ClosePositionRequest{InstrumentName: "BTCUSD-PERP", OrderType: OrderTypeLimit, Price: 30000, Quantity: 0.0001}
	_, err := newResponseServerExchange(t, `{"id":1,"method":"private/close-position","code":0,"result":null}`).ClosePosition(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "ClosePosition should error for a null result")
	_, err = newUnavailableServerExchange(t).ClosePosition(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "ClosePosition should return a request failure")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		live := closeTestPositionRequest(t)
		if live == nil {
			// With no cross margin position to close, Crypto.com refusing the close for it shows it looked the position up
			_, err = e.ClosePosition(t.Context(), req)
			assertRefusedFor(t, err, noPositionCodes, "ClosePosition")
			return
		}
		req = live
	}
	result, err := e.ClosePosition(t.Context(), req)
	require.NoError(t, err, "ClosePosition must not error")
	if mockTests {
		exp := &OrderIDResponse{OrderID: "6530219466236720404", ClientOrderID: "1684d6e4-2c55-64e1-52c3-3aa9febc3a23"}
		assert.Equal(t, exp, result, "ClosePosition should decode every field")
	} else {
		require.NotEmpty(t, result.OrderID, "ClosePosition must return the closing order's ID")
		cancelTradingOrderAfterTest(t, result.OrderID)
	}
}

func TestGetOpenOrders(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same method
	t.Parallel()
	_, err := e.GetOpenOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOpenOrders must refuse a nil request")

	// Serial, so the cases still run before the GetOpenOrders calls below rather than after them
	for _, tc := range []struct {
		name   string
		req    *OpenOrdersRequest
		params string
	}{
		{name: "an instrument", req: &OpenOrdersRequest{InstrumentName: "BTCUSD-PERP"}, params: `{"instrument_name":"BTCUSD-PERP"}`},
		{name: "every instrument", req: &OpenOrdersRequest{}, params: `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/get-open-orders", tc.params, testResponse("private/get-open-orders", `{"data":[]}`))
			_, err := ex.GetOpenOrders(t.Context(), tc.req)
			assert.NoErrorf(t, err, "GetOpenOrders should not error for %s", tc.name)
		})
	}
	req := &OpenOrdersRequest{InstrumentName: "BTCUSD-PERP"}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/get-open-orders","code":0,"result":null}`).GetOpenOrders(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOpenOrders should error for a null result")
	_, err = newUnavailableServerExchange(t).GetOpenOrders(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOpenOrders should return a request failure")

	skipLiveWithoutCredentials(t)
	result, err := e.GetOpenOrders(t.Context(), req)
	require.NoError(t, err, "GetOpenOrders must not error")
	if mockTests {
		exp := []OrderResponse{
			{
				AccountID:             tradingTestAccountID,
				OrderID:               tradingTestOrderID,
				AveragePrice:          preciseNumber("0.0"),
				CumulativeQuantity:    preciseNumber("0.0000"),
				CumulativeValue:       preciseNumber("0.000000"),
				CumulativeFee:         preciseNumber("0.000000"),
				ClientOrderID:         tradingTestOrderClientOrderID,
				OrderType:             OrderTypeLimit,
				TimeInForce:           TimeInForceGoodTillCancel,
				Side:                  SideBuy,
				ExecutionInstructions: []string{ExecutionInstructionPostOnly},
				Quantity:              preciseNumber("0.0001"),
				LimitPrice:            preciseNumber("50000.5"),
				OrderValue:            preciseNumber("5.000050"),
				MakerFeeRate:          0.00025,
				TakerFeeRate:          0.0004,
				Status:                OrderStatusActive,
				UpdateUserID:          tradingTestUpdateUserID,
				OrderDate:             "2026-10-25",
				InstrumentName:        "BTCUSD-PERP",
				FeeInstrumentName:     currency.USD,
				CreateTime:            types.Time(time.UnixMilli(1792890506012)),
				CreateTimeNano:        types.Time(time.Unix(0, 1792890506012345678)),
				UpdateTime:            types.Time(time.UnixMilli(1792890506012)),
			},
			{
				AccountID:             tradingTestAccountID,
				OrderID:               tradingTestIsolationID,
				AveragePrice:          preciseNumber("0.0"),
				CumulativeQuantity:    preciseNumber("0.0000"),
				CumulativeValue:       preciseNumber("0.000000"),
				CumulativeFee:         preciseNumber("0.000000"),
				ClientOrderID:         "1792890500001",
				OrderType:             OrderTypeLimit,
				TimeInForce:           TimeInForceGoodTillCancel,
				Side:                  SideBuy,
				ExecutionInstructions: []string{ExecutionInstructionIsolatedMargin},
				Quantity:              preciseNumber("0.0100"),
				LimitPrice:            preciseNumber("50000.0"),
				OrderValue:            preciseNumber("500.000000"),
				MakerFeeRate:          0.00025,
				TakerFeeRate:          0.0004,
				Status:                OrderStatusActive,
				UpdateUserID:          tradingTestUpdateUserID,
				OrderDate:             "2026-10-25",
				InstrumentName:        "BTCUSD-PERP",
				FeeInstrumentName:     currency.USD,
				CreateTime:            types.Time(time.UnixMilli(1792890500001)),
				CreateTimeNano:        types.Time(time.Unix(0, 1792890500001123457)),
				UpdateTime:            types.Time(time.UnixMilli(1792890500001)),
				IsolationID:           tradingTestIsolationID,
				IsolationType:         IsolationTypeIsolatedMargin,
				Leverage:              10,
				IsolatedMarginAmount:  preciseNumber("50"),
			},
		}
		assert.Equal(t, exp, result, "GetOpenOrders should decode every field")
	} else {
		for i := range result {
			assert.Equalf(t, req.InstrumentName, result[i].InstrumentName, "GetOpenOrders should only return orders of %s", req.InstrumentName)
		}
	}
}

func TestGetOrderDetail(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same method
	t.Parallel()
	_, err := e.GetOrderDetail(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOrderDetail must refuse a nil request")
	_, err = e.GetOrderDetail(t.Context(), &OrderDetailRequest{})
	require.ErrorIs(t, err, errOrderIDRequired, "GetOrderDetail must refuse a request naming no order")

	// Serial, so the cases still run before the GetOrderDetail calls below rather than after them
	for _, tc := range []struct {
		name   string
		req    *OrderDetailRequest
		params string
	}{
		{
			name:   "both IDs",
			req:    &OrderDetailRequest{OrderID: tradingTestOrderID, ClientOrderID: tradingTestOrderClientOrderID},
			params: `{"order_id":"6530219466236720401","client_oid":"c5f682ed-7108-4f1c-b755-972fcdca0f02"}`,
		},
		{name: "an order named by its order ID", req: &OrderDetailRequest{OrderID: tradingTestOrderID}, params: `{"order_id":"6530219466236720401"}`},
		{
			name:   "an order named by its client order ID",
			req:    &OrderDetailRequest{ClientOrderID: tradingTestOrderClientOrderID},
			params: `{"client_oid":"c5f682ed-7108-4f1c-b755-972fcdca0f02"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/get-order-detail", tc.params, testResponse("private/get-order-detail", tradingTestOrderIDResult))
			_, err := ex.GetOrderDetail(t.Context(), tc.req)
			assert.NoErrorf(t, err, "GetOrderDetail should not error for %s", tc.name)
		})
	}
	req := &OrderDetailRequest{OrderID: "6530219466236720403"}
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/get-order-detail","code":0,"result":null}`).GetOrderDetail(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetOrderDetail should error for a null result")
	_, err = newUnavailableServerExchange(t).GetOrderDetail(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrderDetail should return a request failure")

	skipLiveWithoutCredentials(t)
	if !mockTests {
		orders, err := e.GetUserOrderHistory(t.Context(), &UserOrderHistoryRequest{Limit: 1})
		require.NoError(t, err, "GetUserOrderHistory must not error")
		if len(orders) == 0 {
			// With no order of the account's to look up, Crypto.com refusing the recorded order, which is not the
			// account's, shows it looked the order up
			_, err = e.GetOrderDetail(t.Context(), req)
			assertRefusedFor(t, err, unknownOrderCodes, "GetOrderDetail")
			return
		}
		req.OrderID = orders[0].OrderID
	}
	result, err := e.GetOrderDetail(t.Context(), req)
	require.NoError(t, err, "GetOrderDetail must not error")
	if mockTests {
		exp := &OrderResponse{
			AccountID:             tradingTestAccountID,
			OrderID:               "6530219466236720403",
			AveragePrice:          preciseNumber("0.0"),
			CumulativeQuantity:    preciseNumber("0.0000"),
			CumulativeValue:       preciseNumber("0.000000"),
			CumulativeFee:         preciseNumber("0.000000"),
			ClientOrderID:         "1613571154900",
			OrderType:             OrderTypeLimit,
			TimeInForce:           TimeInForceGoodTillCancel,
			Side:                  SideBuy,
			ExecutionInstructions: []string{},
			Quantity:              preciseNumber("0.0100"),
			LimitPrice:            preciseNumber("50000.0"),
			OrderValue:            preciseNumber("500.000000"),
			MakerFeeRate:          0.00025,
			TakerFeeRate:          0.0004,
			Status:                OrderStatusCanceled,
			UpdateUserID:          tradingTestUpdateUserID,
			OrderDate:             "2021-02-17",
			InstrumentName:        "BTCUSD-PERP",
			FeeInstrumentName:     currency.USD,
			CreateTime:            types.Time(time.UnixMilli(1613575617173)),
			CreateTimeNano:        types.Time(time.Unix(0, 1613575617173123456)),
			UpdateTime:            types.Time(time.UnixMilli(1613575617209)),
			Reason:                "43012",
		}
		assert.Equal(t, exp, result, "GetOrderDetail should decode every field")
	} else {
		assert.Equal(t, req.OrderID, result.OrderID, "GetOrderDetail should return the order asked for")
	}
}

func TestCreateOrderList(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same method
	t.Parallel()
	_, err := e.CreateOrderList(t.Context(), nil)
	require.ErrorIs(t, err, errOrderListEmpty, "CreateOrderList must refuse an empty list")
	_, err = e.CreateOrderList(t.Context(), make([]CreateOrderRequest, 11))
	require.ErrorIs(t, err, errOrderListTooLong, "CreateOrderList must refuse a list of 11 orders")
	_, err = e.CreateOrderList(t.Context(), []CreateOrderRequest{*tradingTestOrder(), {Side: SideBuy, OrderType: OrderTypeMarket, Quantity: 0.0001}})
	require.ErrorIs(t, err, errInstrumentNameRequired, "CreateOrderList must refuse an order without an instrument")
	// The cases run in order on e, before the CreateOrderList calls below, so each also checks that a refusal leaves
	// nothing behind
	for _, tc := range createOrderRefusals {
		t.Run(tc.name, func(t *testing.T) {
			orders := []CreateOrderRequest{*tradingTestOrder(), *tradingTestOrder()}
			tc.modify(&orders[1])
			_, err := e.CreateOrderList(t.Context(), orders)
			assert.ErrorIsf(t, err, tc.err, "CreateOrderList should refuse %s", tc.name)
			assert.ErrorContainsf(t, err, "order 1:", "CreateOrderList should name the index of %s", tc.name)
		})
	}

	ex := newSignedParamsServer(t, "private/create-order-list", tradingTestOrderListParams, testResponse("private/create-order-list", tradingTestOrderListEntryResult))
	_, err = ex.CreateOrderList(t.Context(), tradingTestOrderList())
	assert.NoError(t, err, "CreateOrderList should not error")
	tenOrders := make([]CreateOrderRequest, 10)
	for i := range tenOrders {
		tenOrders[i] = *tradingTestOrder()
	}
	_, err = newResponseServerExchange(t, testResponse("private/create-order-list", tradingTestOrderListEntryResult)).CreateOrderList(t.Context(), tenOrders)
	assert.NoError(t, err, "CreateOrderList should send a list of 10 orders")
	// Crypto.com's live answer, as ccxt recorded it: codes and indexes as numbers, and an order refused in its entry
	result, err := newResponseServerExchange(t, `{"id":1698068111133,"method":"private/create-order-list","code":0,"result":[{"code":0,"index":0,"client_oid":"1698068111133_0","order_id":"6142909896519488206"},{"code":306,"index":1,"client_oid":"1698068111133_1","message":"INSUFFICIENT_AVAILABLE_BALANCE","order_id":"6142909896519488207"}]}`).CreateOrderList(t.Context(), tradingTestOrderList())
	require.NoError(t, err, "CreateOrderList must not error when Crypto.com refuses an order of the list")
	exp := []OrderListEntryResponse{
		{Index: 0, Code: 0, ClientOrderID: "1698068111133_0", OrderID: "6142909896519488206"},
		{Index: 1, Code: 306, ClientOrderID: "1698068111133_1", OrderID: "6142909896519488207", Message: "INSUFFICIENT_AVAILABLE_BALANCE"},
	}
	assert.Equal(t, exp, result, "CreateOrderList should report the refused order in its entry")
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/create-order-list","code":0,"result":null}`).CreateOrderList(t.Context(), tradingTestOrderList())
	assert.ErrorIs(t, err, common.ErrNoResponse, "CreateOrderList should error for a null result")
	_, err = newUnavailableServerExchange(t).CreateOrderList(t.Context(), tradingTestOrderList())
	assert.ErrorIs(t, err, request.ErrBadStatus, "CreateOrderList should return a request failure")

	skipLiveUnlessCanManipulate(t)
	orders := tradingTestOrderList()
	if !mockTests {
		for i := range orders {
			orders[i].ClientOrderID = newTradingTestClientOrderID() + "-" + strconv.Itoa(i)
		}
	}
	result, err = e.CreateOrderList(t.Context(), orders)
	require.NoError(t, err, "CreateOrderList must not error")
	if mockTests {
		exp := []OrderListEntryResponse{
			{Index: 0, Code: 0, ClientOrderID: "api_leg1", OrderID: "5755600460443882762"},
			{Index: 1, Code: 0, ClientOrderID: "api_leg2", OrderID: "5755600460443882763"},
		}
		assert.Equal(t, exp, result, "CreateOrderList should decode every field")
	} else {
		for i := range result {
			if result[i].Code == 0 {
				cancelTradingOrderAfterTest(t, result[i].OrderID)
			}
		}
		assertListOutcomes(t, result, len(orders), "CreateOrderList", "place")
	}
}

func TestCancelOrderList(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same method
	t.Parallel()
	// The cases run in order on e, before the CancelOrderList calls below, so each also checks that a refusal leaves
	// nothing behind
	for _, tc := range cancelOrderListRefusals {
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.CancelOrderList(t.Context(), tc.orders)
			assert.ErrorIsf(t, err, tc.err, "CancelOrderList should refuse %s", tc.name)
		})
	}
	_, err := e.CancelOrderList(t.Context(), []CancelOrderListEntryRequest{{OrderID: tradingTestOrderID}, {InstrumentName: "CRO_USD"}})
	assert.ErrorContains(t, err, "order 1:", "CancelOrderList should name the index of the order named by neither ID")

	// Serial, so the cases still run before the CancelOrderList calls below rather than after them
	for _, tc := range []struct {
		name   string
		orders []CancelOrderListEntryRequest
		params string
	}{
		{
			name:   "orders named by client order ID",
			orders: []CancelOrderListEntryRequest{{InstrumentName: "CRO_USD", ClientOrderID: "api_leg1"}, {InstrumentName: "CRO_USD", ClientOrderID: "api_leg2"}},
			params: `{"contingency_type":"LIST","order_list":[{"instrument_name":"CRO_USD","client_oid":"api_leg1"},{"instrument_name":"CRO_USD","client_oid":"api_leg2"}]}`,
		},
		{
			name:   "an order with every optional param and one with none",
			orders: []CancelOrderListEntryRequest{{InstrumentName: "CRO_USD", OrderID: "5755600460443882762", ClientOrderID: "api_leg1"}, {OrderID: "5755600460443882763"}},
			params: `{"contingency_type":"LIST","order_list":[{"instrument_name":"CRO_USD","order_id":"5755600460443882762","client_oid":"api_leg1"},{"order_id":"5755600460443882763"}]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/cancel-order-list", tc.params, testResponse("private/cancel-order-list", `[{"code":"0","index":"0"}]`))
			_, err := ex.CancelOrderList(t.Context(), tc.orders)
			assert.NoErrorf(t, err, "CancelOrderList should not error for %s", tc.name)
		})
	}
	orders := []CancelOrderListEntryRequest{{InstrumentName: "CRO_USD", ClientOrderID: "api_leg1"}, {InstrumentName: "CRO_USD", ClientOrderID: "api_leg2"}}
	tenOrders := make([]CancelOrderListEntryRequest, 10)
	for i := range tenOrders {
		tenOrders[i] = CancelOrderListEntryRequest{OrderID: strconv.Itoa(i + 1)}
	}
	_, err = newResponseServerExchange(t, testResponse("private/cancel-order-list", `[{"code":"0","index":"0"}]`)).CancelOrderList(t.Context(), tenOrders)
	assert.NoError(t, err, "CancelOrderList should send a list of 10 orders")
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/cancel-order-list","code":0,"result":null}`).CancelOrderList(t.Context(), orders)
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelOrderList should error for a null result")
	_, err = newUnavailableServerExchange(t).CancelOrderList(t.Context(), orders)
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelOrderList should return a request failure")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		first := placeTradingTestOrder(t)
		cancelTradingOrderIfTestFails(t, first)
		second := placeTradingTestOrder(t)
		cancelTradingOrderIfTestFails(t, second)
		orders = []CancelOrderListEntryRequest{{InstrumentName: "BTCUSD-PERP", OrderID: first}, {InstrumentName: "BTCUSD-PERP", OrderID: second}}
	}
	result, err := e.CancelOrderList(t.Context(), orders)
	require.NoError(t, err, "CancelOrderList must not error")
	if mockTests {
		exp := []OrderListEntryResponse{{Index: 0, Code: 0}, {Index: 1, Code: 0}}
		assert.Equal(t, exp, result, "CancelOrderList should decode every field")
	} else {
		assertListOutcomes(t, result, len(orders), "CancelOrderList", "cancel")
	}
}

func TestChangeAccountLeverage(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same method
	t.Parallel()
	// The cases run in order on e, before the ChangeAccountLeverage calls below, so each also checks that a refusal
	// leaves nothing behind
	for _, tc := range []struct {
		name string
		req  *ChangeAccountLeverageRequest
		err  error
	}{
		{"a nil request", nil, common.ErrNilPointer},
		{"a request without an account", &ChangeAccountLeverageRequest{Leverage: 10}, errAccountIDRequired},
		{"no leverage", &ChangeAccountLeverageRequest{AccountID: tradingTestAccountID}, errInvalidLeverage},
		{"a leverage above 100", &ChangeAccountLeverageRequest{AccountID: tradingTestAccountID, Leverage: 101}, errInvalidLeverage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.ErrorIsf(t, e.ChangeAccountLeverage(t.Context(), tc.req), tc.err, "ChangeAccountLeverage should refuse %s", tc.name)
		})
	}

	// Serial, so the cases still run before the ChangeAccountLeverage calls below rather than after them
	for _, leverage := range []uint64{1, 100} {
		t.Run(strconv.FormatUint(leverage, 10), func(t *testing.T) {
			params := `{"account_id":"52e7c00f-1324-5a6a-bf6e-de445bde21a5","leverage":` + strconv.FormatUint(leverage, 10) + `}`
			ex := newSignedParamsServer(t, "private/change-account-leverage", params, testResponse("private/change-account-leverage", ""))
			assert.NoErrorf(t, ex.ChangeAccountLeverage(t.Context(), &ChangeAccountLeverageRequest{AccountID: tradingTestAccountID, Leverage: leverage}), "ChangeAccountLeverage should not error for a leverage of %d", leverage)
		})
	}
	req := &ChangeAccountLeverageRequest{AccountID: tradingTestAccountID, Leverage: 10}
	assert.ErrorIs(t, newUnavailableServerExchange(t).ChangeAccountLeverage(t.Context(), req), request.ErrBadStatus, "ChangeAccountLeverage should return a request failure")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		accounts, err := e.GetAccounts(t.Context(), &AccountsRequest{})
		require.NoError(t, err, "GetAccounts must return the account whose leverage the test sets")
		settings, err := e.GetAccountSettings(t.Context())
		require.NoError(t, err, "GetAccountSettings must not error")
		require.NotEmpty(t, settings, "GetAccountSettings must return the account's settings")
		leverage, err := strconv.ParseUint(settings[0].Leverage.String(), 10, 64)
		require.NoError(t, err, "the account's leverage must be a whole number")
		// Setting the leverage the account has leaves it as it was, which a change and its restore could not promise while
		// TestChangeAccountSettings reads and sets the same leverage
		req = &ChangeAccountLeverageRequest{AccountID: accounts.ThisAccount.UUID, Leverage: leverage}
	}
	require.NoError(t, e.ChangeAccountLeverage(t.Context(), req), "ChangeAccountLeverage must not error")
}

func TestChangeAccountSettings(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same method
	t.Parallel()
	// The cases run in order on e, before the ChangeAccountSettings calls below, so each also checks that a refusal
	// leaves nothing behind
	for _, tc := range []struct {
		name string
		req  *ChangeAccountSettingsRequest
		err  error
	}{
		{"a nil request", nil, common.ErrNilPointer},
		{"a request changing nothing", &ChangeAccountSettingsRequest{}, errAccountSettingsRequired},
		{"a leverage above 100", &ChangeAccountSettingsRequest{Leverage: 101}, errInvalidLeverage},
		{"an unknown self-trade prevention scope", &ChangeAccountSettingsRequest{SelfTradePreventionScope: "X", SelfTradePreventionInstruction: SelfTradePreventionCancelMaker}, errInvalidSelfTradePreventionScope},
		{"an unknown self-trade prevention instruction", &ChangeAccountSettingsRequest{SelfTradePreventionScope: SelfTradePreventionScopeSubAccount, SelfTradePreventionInstruction: "X"}, errInvalidSelfTradePreventionInstruction},
		{"a self-trade prevention scope without an instruction", &ChangeAccountSettingsRequest{SelfTradePreventionScope: SelfTradePreventionScopeSubAccount}, errInvalidSelfTradePreventionInstruction},
		{"a self-trade prevention ID above 32767", &ChangeAccountSettingsRequest{SelfTradePreventionScope: SelfTradePreventionScopeSubAccount, SelfTradePreventionInstruction: SelfTradePreventionCancelMaker, SelfTradePreventionID: 32768}, errInvalidSelfTradePreventionID},
		{"a self-trade prevention ID without a scope", &ChangeAccountSettingsRequest{SelfTradePreventionInstruction: SelfTradePreventionCancelMaker, SelfTradePreventionID: 100}, errInvalidSelfTradePreventionID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.ErrorIsf(t, e.ChangeAccountSettings(t.Context(), tc.req), tc.err, "ChangeAccountSettings should refuse %s", tc.name)
		})
	}

	// Serial, so the cases still run before the ChangeAccountSettings calls below rather than after them
	for _, tc := range []struct {
		name   string
		req    *ChangeAccountSettingsRequest
		params string
	}{
		{
			name:   "every optional param",
			req:    &ChangeAccountSettingsRequest{SelfTradePreventionScope: SelfTradePreventionScopeMasterOrSubAccount, SelfTradePreventionInstruction: SelfTradePreventionCancelTaker, SelfTradePreventionID: 32767, Leverage: 100},
			params: `{"stp_scope":"M","stp_inst":"T","stp_id":"32767","leverage":100}`,
		},
		{name: "only the leverage", req: &ChangeAccountSettingsRequest{Leverage: 20}, params: `{"leverage":20}`},
		{name: "a reset of self-trade prevention", req: &ChangeAccountSettingsRequest{SelfTradePreventionScope: SelfTradePreventionScopeReset}, params: `{"stp_scope":"D"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/change-account-settings", tc.params, testResponse("private/change-account-settings", ""))
			assert.NoErrorf(t, ex.ChangeAccountSettings(t.Context(), tc.req), "ChangeAccountSettings should not error for %s", tc.name)
		})
	}
	req := &ChangeAccountSettingsRequest{SelfTradePreventionScope: SelfTradePreventionScopeSubAccount, SelfTradePreventionID: 100, SelfTradePreventionInstruction: SelfTradePreventionCancelMaker}
	assert.ErrorIs(t, newUnavailableServerExchange(t).ChangeAccountSettings(t.Context(), req), request.ErrBadStatus, "ChangeAccountSettings should return a request failure")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		// Setting the leverage the account has leaves its settings as they were
		settings, err := e.GetAccountSettings(t.Context())
		require.NoError(t, err, "GetAccountSettings must not error")
		require.NotEmpty(t, settings, "GetAccountSettings must return the account's settings")
		leverage, err := strconv.ParseUint(settings[0].Leverage.String(), 10, 64)
		require.NoError(t, err, "the account's leverage must be a whole number")
		req = &ChangeAccountSettingsRequest{Leverage: leverage}
	}
	require.NoError(t, e.ChangeAccountSettings(t.Context(), req), "ChangeAccountSettings must not error")
}

func TestGetAccountSettings(t *testing.T) {
	t.Parallel()
	ex := newSignedParamsServer(t, "private/get-account-settings", `{}`, testResponse("private/get-account-settings", `[]`))
	_, err := ex.GetAccountSettings(t.Context())
	assert.NoError(t, err, "GetAccountSettings should not error")
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/get-account-settings","code":0,"result":null}`).GetAccountSettings(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetAccountSettings should error for a null result")
	_, err = newUnavailableServerExchange(t).GetAccountSettings(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAccountSettings should return a request failure")

	skipLiveWithoutCredentials(t)
	result, err := e.GetAccountSettings(t.Context())
	require.NoError(t, err, "GetAccountSettings must not error")
	if mockTests {
		exp := []AccountSettingsResponse{{Leverage: 20, SelfTradePreventionID: 100, SelfTradePreventionScope: SelfTradePreventionScopeSubAccount, SelfTradePreventionInstruction: SelfTradePreventionCancelMaker}}
		assert.Equal(t, exp, result, "GetAccountSettings should decode every field")
	} else {
		require.NotEmpty(t, result, "GetAccountSettings must return the account's settings")
		assert.Positive(t, result[0].Leverage.Float64(), "GetAccountSettings should return the account's maximum leverage")
	}
}

func TestCreateIsolatedMarginTransfer(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same method
	t.Parallel()
	// The cases run in order on e, before the CreateIsolatedMarginTransfer calls below, so each also checks that a
	// refusal leaves nothing behind
	for _, tc := range []struct {
		name string
		req  *CreateIsolatedMarginTransferRequest
		err  error
	}{
		{"a nil request", nil, common.ErrNilPointer},
		{"a request without an isolated position", &CreateIsolatedMarginTransferRequest{Direction: IsolatedMarginTransferDirectionCredit, Amount: 50}, errIsolationIDRequired},
		{"a request without a direction", &CreateIsolatedMarginTransferRequest{IsolationID: tradingTestIsolationID, Amount: 50}, errInvalidIsolatedMarginTransferDirection},
		{"a lower-case direction", &CreateIsolatedMarginTransferRequest{IsolationID: tradingTestIsolationID, Direction: "credit", Amount: 50}, errInvalidIsolatedMarginTransferDirection},
		{"a request without an amount", &CreateIsolatedMarginTransferRequest{IsolationID: tradingTestIsolationID, Direction: IsolatedMarginTransferDirectionCredit}, errInvalidAmount},
		{"a negative amount", &CreateIsolatedMarginTransferRequest{IsolationID: tradingTestIsolationID, Direction: IsolatedMarginTransferDirectionDebit, Amount: -50}, errInvalidAmount},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.ErrorIsf(t, e.CreateIsolatedMarginTransfer(t.Context(), tc.req), tc.err, "CreateIsolatedMarginTransfer should refuse %s", tc.name)
		})
	}

	// Serial, so the cases still run before the CreateIsolatedMarginTransfer calls below rather than after them
	for _, tc := range []struct {
		req    *CreateIsolatedMarginTransferRequest
		params string
	}{
		{&CreateIsolatedMarginTransferRequest{IsolationID: tradingTestIsolationID, Direction: IsolatedMarginTransferDirectionCredit, Amount: 50}, `{"isolation_id":"6530219466236720402","direction":"CREDIT","amount":"50"}`},
		{&CreateIsolatedMarginTransferRequest{IsolationID: tradingTestIsolationID, Direction: IsolatedMarginTransferDirectionDebit, Amount: 25.5}, `{"isolation_id":"6530219466236720402","direction":"DEBIT","amount":"25.5"}`},
	} {
		t.Run(tc.req.Direction, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/create-isolated-margin-transfer", tc.params, testResponse("private/create-isolated-margin-transfer", ""))
			assert.NoErrorf(t, ex.CreateIsolatedMarginTransfer(t.Context(), tc.req), "CreateIsolatedMarginTransfer should not error for a %s transfer", tc.req.Direction)
		})
	}
	req := &CreateIsolatedMarginTransferRequest{IsolationID: tradingTestIsolationID, Direction: IsolatedMarginTransferDirectionCredit, Amount: 50}
	assert.ErrorIs(t, newUnavailableServerExchange(t).CreateIsolatedMarginTransfer(t.Context(), req), request.ErrBadStatus, "CreateIsolatedMarginTransfer should return a request failure")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		position := isolatedTestPosition(t)
		if position == nil {
			// With no isolated position, Crypto.com refusing the recorded isolation ID, which is not the account's, shows it
			// looked the position up
			assertRefusedFor(t, e.CreateIsolatedMarginTransfer(t.Context(), req), unknownIsolationIDCodes, "CreateIsolatedMarginTransfer")
			return
		}
		req = &CreateIsolatedMarginTransferRequest{IsolationID: position.IsolationID, Direction: IsolatedMarginTransferDirectionCredit, Amount: 1}
	}
	require.NoError(t, e.CreateIsolatedMarginTransfer(t.Context(), req), "CreateIsolatedMarginTransfer must not error")
	if !mockTests {
		debit := &CreateIsolatedMarginTransferRequest{IsolationID: req.IsolationID, Direction: IsolatedMarginTransferDirectionDebit, Amount: req.Amount}
		t.Cleanup(func() {
			assert.NoError(t, e.CreateIsolatedMarginTransfer(context.WithoutCancel(t.Context()), debit), "CreateIsolatedMarginTransfer should debit the margin the test credited")
		})
	}
}

func TestChangeIsolatedMarginLeverage(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same method
	t.Parallel()
	// The cases run in order on e, before the ChangeIsolatedMarginLeverage calls below, so each also checks that a
	// refusal leaves nothing behind
	for _, tc := range []struct {
		name string
		req  *ChangeIsolatedMarginLeverageRequest
		err  error
	}{
		{"a nil request", nil, common.ErrNilPointer},
		{"a request without an isolated position", &ChangeIsolatedMarginLeverageRequest{Leverage: 10}, errIsolationIDRequired},
		{"a request without a leverage", &ChangeIsolatedMarginLeverageRequest{IsolationID: tradingTestIsolationID}, errInvalidLeverage},
		{"a negative leverage", &ChangeIsolatedMarginLeverageRequest{IsolationID: tradingTestIsolationID, Leverage: -10}, errInvalidLeverage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.ErrorIsf(t, e.ChangeIsolatedMarginLeverage(t.Context(), tc.req), tc.err, "ChangeIsolatedMarginLeverage should refuse %s", tc.name)
		})
	}

	// Serial, so the cases still run before the ChangeIsolatedMarginLeverage calls below rather than after them
	for _, tc := range []struct {
		req    *ChangeIsolatedMarginLeverageRequest
		params string
	}{
		{&ChangeIsolatedMarginLeverageRequest{IsolationID: tradingTestIsolationID, Leverage: 10}, `{"isolation_id":"6530219466236720402","leverage":"10"}`},
		{&ChangeIsolatedMarginLeverageRequest{IsolationID: tradingTestIsolationID, Leverage: 12.5}, `{"isolation_id":"6530219466236720402","leverage":"12.5"}`},
	} {
		t.Run(tc.req.Leverage.String(), func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/change-isolated-margin-leverage", tc.params, testResponse("private/change-isolated-margin-leverage", ""))
			assert.NoErrorf(t, ex.ChangeIsolatedMarginLeverage(t.Context(), tc.req), "ChangeIsolatedMarginLeverage should not error for a leverage of %v", tc.req.Leverage)
		})
	}
	req := &ChangeIsolatedMarginLeverageRequest{IsolationID: tradingTestIsolationID, Leverage: 10}
	assert.ErrorIs(t, newUnavailableServerExchange(t).ChangeIsolatedMarginLeverage(t.Context(), req), request.ErrBadStatus, "ChangeIsolatedMarginLeverage should return a request failure")

	skipLiveUnlessCanManipulate(t)
	if !mockTests {
		position := isolatedTestPosition(t)
		if position == nil {
			// With no isolated position, Crypto.com refusing the recorded isolation ID, which is not the account's, shows it
			// looked the position up
			assertRefusedFor(t, e.ChangeIsolatedMarginLeverage(t.Context(), req), unknownIsolationIDCodes, "ChangeIsolatedMarginLeverage")
			return
		}
		// Setting the maximum leverage the position has leaves it as it was
		req = &ChangeIsolatedMarginLeverageRequest{IsolationID: position.IsolationID, Leverage: position.Leverage}
	}
	require.NoError(t, e.ChangeIsolatedMarginLeverage(t.Context(), req), "ChangeIsolatedMarginLeverage must not error")
}

func TestGetFeeRate(t *testing.T) {
	t.Parallel()
	ex := newSignedParamsServer(t, "private/get-fee-rate", `{}`, testResponse("private/get-fee-rate", `{"spot_tier":"1"}`))
	_, err := ex.GetFeeRate(t.Context())
	assert.NoError(t, err, "GetFeeRate should not error")
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/get-fee-rate","code":0,"result":null}`).GetFeeRate(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetFeeRate should error for a null result")
	_, err = newUnavailableServerExchange(t).GetFeeRate(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFeeRate should return a request failure")

	t.Run("bare fee tiers", func(t *testing.T) {
		t.Parallel()
		ex := newResponseServerExchange(t, testResponse("private/get-fee-rate", `{"spot_tier":1,"deriv_tier":2,"effective_spot_maker_rate_bps":"5"}`))
		got, err := ex.GetFeeRate(t.Context())
		require.NoError(t, err, "GetFeeRate must accept bare tiers")
		exp := &FeeRateResponse{SpotTier: 1, DerivativesTier: 2, EffectiveSpotMakerRateBasisPoints: 5}
		assert.Equal(t, exp, got, "GetFeeRate should decode bare tiers")
	})

	skipLiveWithoutCredentials(t)
	result, err := e.GetFeeRate(t.Context())
	require.NoError(t, err, "GetFeeRate must not error")
	if mockTests {
		exp := &FeeRateResponse{
			SpotTier:                                 3,
			DerivativesTier:                          3,
			EffectiveSpotMakerRateBasisPoints:        6.5,
			EffectiveSpotTakerRateBasisPoints:        6.9,
			EffectiveDerivativesMakerRateBasisPoints: 1.1,
			EffectiveDerivativesTakerRateBasisPoints: 3,
		}
		assert.Equal(t, exp, result, "GetFeeRate should decode every field")
	} else {
		assert.Positive(t, result.EffectiveSpotTakerRateBasisPoints.Float64(), "GetFeeRate should return the spot taker rate")
	}
}

func TestGetInstrumentFeeRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetInstrumentFeeRate(t.Context(), "")
	require.ErrorIs(t, err, errInstrumentNameRequired, "GetInstrumentFeeRate must refuse an empty instrument")

	ex := newSignedParamsServer(t, "private/get-instrument-fee-rate", `{"instrument_name":"BTC_USD"}`, testResponse("private/get-instrument-fee-rate", `{"instrument_name":"BTC_USD"}`))
	_, err = ex.GetInstrumentFeeRate(t.Context(), "BTC_USD")
	assert.NoError(t, err, "GetInstrumentFeeRate should not error")
	_, err = newResponseServerExchange(t, `{"id":1,"method":"private/get-instrument-fee-rate","code":0,"result":null}`).GetInstrumentFeeRate(t.Context(), "BTC_USD")
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetInstrumentFeeRate should error for a null result")
	_, err = newUnavailableServerExchange(t).GetInstrumentFeeRate(t.Context(), "BTC_USD")
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetInstrumentFeeRate should return a request failure")

	skipLiveWithoutCredentials(t)
	result, err := e.GetInstrumentFeeRate(t.Context(), "BTC_USD")
	require.NoError(t, err, "GetInstrumentFeeRate must not error")
	if mockTests {
		exp := &InstrumentFeeRateResponse{InstrumentName: "BTC_USD", EffectiveMakerRateBasisPoints: 6.5, EffectiveTakerRateBasisPoints: 6.9}
		assert.Equal(t, exp, result, "GetInstrumentFeeRate should decode every field")
	} else {
		assert.Equal(t, "BTC_USD", result.InstrumentName, "GetInstrumentFeeRate should return the instrument's rates")
	}
}

func TestGetUserOrderHistory(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same method
	t.Parallel()
	// The cases run in order on e, before the GetUserOrderHistory calls below, so each also checks that a refusal
	// leaves nothing behind
	for _, tc := range []struct {
		name string
		req  *UserOrderHistoryRequest
		err  error
	}{
		{"a nil request", nil, common.ErrNilPointer},
		{"a window starting after it ends", &UserOrderHistoryRequest{StartTime: time.Unix(0, tradingTestOrderHistoryEnd), EndTime: time.Unix(0, tradingTestOrderHistoryStart)}, common.ErrStartAfterEnd},
		{"a limit above 100", &UserOrderHistoryRequest{Limit: 101}, errInvalidCount},
		{"a start before the Unix epoch", &UserOrderHistoryRequest{StartTime: time.Unix(-1, 0)}, errTimeBeforeUnixEpoch},
		{"an end before the Unix epoch", &UserOrderHistoryRequest{EndTime: time.Unix(-1, 0)}, errTimeBeforeUnixEpoch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.GetUserOrderHistory(t.Context(), tc.req)
			assert.ErrorIsf(t, err, tc.err, "GetUserOrderHistory should refuse %s", tc.name)
		})
	}

	// Serial, so the cases still run before the GetUserOrderHistory calls below rather than after them
	for _, tc := range []struct {
		name   string
		req    *UserOrderHistoryRequest
		params string
	}{
		{
			name:   "every optional param",
			req:    &UserOrderHistoryRequest{InstrumentName: "BTCUSD-PERP", StartTime: time.Unix(0, tradingTestOrderHistoryStart), EndTime: time.Unix(0, tradingTestOrderHistoryEnd), Limit: 100, IsolationID: tradingTestIsolationID},
			params: `{"instrument_name":"BTCUSD-PERP","start_time":"1610905028000081486","end_time":"1613570791058211357","limit":100,"isolation_id":"6530219466236720402"}`,
		},
		{name: "no params", req: &UserOrderHistoryRequest{}, params: `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/get-order-history", tc.params, testResponse("private/get-order-history", `{"data":[]}`))
			_, err := ex.GetUserOrderHistory(t.Context(), tc.req)
			assert.NoErrorf(t, err, "GetUserOrderHistory should not error for %s", tc.name)
		})
	}
	req := &UserOrderHistoryRequest{InstrumentName: "BTCUSD-PERP", StartTime: time.Unix(0, tradingTestOrderHistoryStart), EndTime: time.Unix(0, tradingTestOrderHistoryEnd), Limit: 20}
	_, err := newResponseServerExchange(t, `{"id":1,"method":"private/get-order-history","code":0,"result":null}`).GetUserOrderHistory(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetUserOrderHistory should error for a null result")
	_, err = newUnavailableServerExchange(t).GetUserOrderHistory(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserOrderHistory should return a request failure")

	skipLiveWithoutCredentials(t)
	if !mockTests {
		// The recorded window is older than the six months of history Crypto.com keeps
		req = &UserOrderHistoryRequest{Limit: 20}
	}
	result, err := e.GetUserOrderHistory(t.Context(), req)
	require.NoError(t, err, "GetUserOrderHistory must not error")
	if mockTests {
		exp := []OrderResponse{
			{
				AccountID:             tradingTestAccountID,
				OrderID:               "6530219466236720405",
				AveragePrice:          preciseNumber("0.0"),
				CumulativeQuantity:    preciseNumber("0.0000"),
				CumulativeValue:       preciseNumber("0.000000"),
				CumulativeFee:         preciseNumber("0.000000"),
				ClientOrderID:         "1610905028001",
				OrderType:             OrderTypeLimit,
				TimeInForce:           TimeInForceGoodTillCancel,
				Side:                  SideBuy,
				ExecutionInstructions: []string{},
				Quantity:              preciseNumber("0.0001"),
				LimitPrice:            preciseNumber("51000.0"),
				OrderValue:            preciseNumber("5.100000"),
				MakerFeeRate:          0.00025,
				TakerFeeRate:          0.0004,
				Status:                OrderStatusCanceled,
				UpdateUserID:          tradingTestUpdateUserID,
				OrderDate:             "2021-01-17",
				InstrumentName:        "BTCUSD-PERP",
				FeeInstrumentName:     currency.USD,
				CreateTime:            types.Time(time.UnixMilli(1610905028001)),
				CreateTimeNano:        types.Time(time.Unix(0, 1610905028001234567)),
				UpdateTime:            types.Time(time.UnixMilli(1613570700000)),
			},
			{
				AccountID:             tradingTestAccountID,
				OrderID:               "6530219466236720406",
				ClientOrderID:         "1613570791057",
				OrderType:             OrderTypeLimit,
				TimeInForce:           TimeInForceGoodTillCancel,
				Side:                  SideBuy,
				ExecutionInstructions: []string{},
				Quantity:              preciseNumber("0.0500"),
				LimitPrice:            preciseNumber("51283.0"),
				OrderValue:            preciseNumber("2564.150000"),
				MakerFeeRate:          0.00025,
				TakerFeeRate:          0.0004,
				AveragePrice:          preciseNumber("51278.5"),
				CumulativeQuantity:    preciseNumber("0.0500"),
				CumulativeValue:       preciseNumber("2563.925000"),
				CumulativeFee:         preciseNumber("1.025570"),
				Status:                OrderStatusFilled,
				UpdateUserID:          tradingTestUpdateUserID,
				OrderDate:             "2021-02-17",
				InstrumentName:        "BTCUSD-PERP",
				FeeInstrumentName:     currency.USD,
				CreateTime:            types.Time(time.UnixMilli(1613570791057)),
				CreateTimeNano:        types.Time(time.Unix(0, 1613570791057123456)),
				UpdateTime:            types.Time(time.UnixMilli(1613570791058)),
			},
		}
		assert.Equal(t, exp, result, "GetUserOrderHistory should decode every field")
	} else {
		for i := range result {
			assert.NotEmptyf(t, result[i].OrderID, "GetUserOrderHistory should return order %d's ID", i)
		}
	}
}

func TestGetUserTrades(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same method
	t.Parallel()
	// The cases run in order on e, before the GetUserTrades calls below, so each also checks that a refusal leaves
	// nothing behind
	for _, tc := range []struct {
		name string
		req  *UserTradesRequest
		err  error
	}{
		{"a nil request", nil, common.ErrNilPointer},
		{"a window starting after it ends", &UserTradesRequest{StartTime: time.Unix(0, tradingTestTradesWindowEnd), EndTime: time.Unix(0, tradingTestTradesWindowStart)}, common.ErrStartAfterEnd},
		{"a limit above 100", &UserTradesRequest{Limit: 101}, errInvalidCount},
		{"a start before the Unix epoch", &UserTradesRequest{StartTime: time.Unix(-1, 0)}, errTimeBeforeUnixEpoch},
		{"an end before the Unix epoch", &UserTradesRequest{EndTime: time.Unix(-1, 0)}, errTimeBeforeUnixEpoch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.GetUserTrades(t.Context(), tc.req)
			assert.ErrorIsf(t, err, tc.err, "GetUserTrades should refuse %s", tc.name)
		})
	}

	// Serial, so the cases still run before the GetUserTrades calls below rather than after them
	for _, tc := range []struct {
		name   string
		req    *UserTradesRequest
		params string
	}{
		{
			name:   "every optional param",
			req:    &UserTradesRequest{InstrumentName: "BTCUSD-PERP", StartTime: time.Unix(0, tradingTestTradesWindowStart), EndTime: time.Unix(0, tradingTestTradesWindowEnd), Limit: 100, IsolationID: tradingTestIsolationID},
			params: `{"instrument_name":"BTCUSD-PERP","start_time":"1619089031996081486","end_time":"1619200052124211357","limit":100,"isolation_id":"6530219466236720402"}`,
		},
		{name: "no params", req: &UserTradesRequest{}, params: `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/get-trades", tc.params, testResponse("private/get-trades", `{"data":[]}`))
			_, err := ex.GetUserTrades(t.Context(), tc.req)
			assert.NoErrorf(t, err, "GetUserTrades should not error for %s", tc.name)
		})
	}
	// Serial too, so the cases run before the GetUserTrades calls below
	for _, tc := range []struct {
		name  string
		trade string
	}{
		{"bare match counts", `{"match_count":2,"match_index":1}`},
		{"quoted match counts", `{"match_count":"2","match_index":"1"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex := newResponseServerExchange(t, testResponse("private/get-trades", `{"data":[`+tc.trade+`]}`))
			got, err := ex.GetUserTrades(t.Context(), &UserTradesRequest{})
			require.NoError(t, err, "GetUserTrades must accept the match counts")
			assert.Equal(t, []UserTradeResponse{{MatchCount: 2, MatchIndex: 1}}, got, "GetUserTrades should decode the match counts")
		})
	}
	req := &UserTradesRequest{InstrumentName: "BTCUSD-PERP", StartTime: time.Unix(0, tradingTestTradesWindowStart), EndTime: time.Unix(0, tradingTestTradesWindowEnd), Limit: 20}
	_, err := newResponseServerExchange(t, `{"id":1,"method":"private/get-trades","code":0,"result":null}`).GetUserTrades(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetUserTrades should error for a null result")
	_, err = newUnavailableServerExchange(t).GetUserTrades(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetUserTrades should return a request failure")

	skipLiveWithoutCredentials(t)
	if !mockTests {
		// The recorded window is older than the six months of history Crypto.com keeps
		req = &UserTradesRequest{Limit: 20}
	}
	result, err := e.GetUserTrades(t.Context(), req)
	require.NoError(t, err, "GetUserTrades must not error")
	if mockTests {
		exp := []UserTradeResponse{
			{
				AccountID:           tradingTestAccountID,
				EventDate:           "2021-04-22",
				JournalType:         JournalTypeTrading,
				TradedQuantity:      preciseNumber("0.0500"),
				TradedPrice:         preciseNumber("51278.5"),
				Fees:                preciseNumber("-0.140981"),
				FeeCredits:          preciseNumber("-0.500000"),
				OrderID:             "6530219466236720407",
				TradeID:             "1619100000000827635",
				TradeMatchID:        "4611686018427388001",
				ClientOrderID:       "7665b001-2753-4d17-b266-61ecb755922d",
				TakerSide:           TakerSideMaker,
				Side:                SideBuy,
				InstrumentName:      "BTCUSD-PERP",
				FeeInstrumentName:   currency.USD,
				CreateTime:          types.Time(time.UnixMilli(1619100000000)),
				CreateTimeNano:      types.Time(time.Unix(0, 1619100000000827635)),
				TransactionTimeNano: types.Time(time.Unix(0, 1619100000000827635)),
				MatchCount:          1,
			},
		}
		assert.Equal(t, exp, result, "GetUserTrades should decode every field")
	} else {
		for i := range result {
			assert.NotEmptyf(t, result[i].TradeID, "GetUserTrades should return trade %d's ID", i)
		}
	}
}

func TestGetTransactions(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same method
	t.Parallel()
	// The cases run in order on e, before the GetTransactions calls below, so each also checks that a refusal leaves
	// nothing behind
	for _, tc := range []struct {
		name string
		req  *TransactionsRequest
		err  error
	}{
		{"a nil request", nil, common.ErrNilPointer},
		{"a window starting after it ends", &TransactionsRequest{StartTime: time.Unix(0, tradingTestTradesWindowEnd), EndTime: time.Unix(0, tradingTestTradesWindowStart)}, common.ErrStartAfterEnd},
		{"a limit above 100", &TransactionsRequest{Limit: 101}, errInvalidCount},
		{"a start before the Unix epoch", &TransactionsRequest{StartTime: time.Unix(-1, 0)}, errTimeBeforeUnixEpoch},
		{"an end before the Unix epoch", &TransactionsRequest{EndTime: time.Unix(-1, 0)}, errTimeBeforeUnixEpoch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.GetTransactions(t.Context(), tc.req)
			assert.ErrorIsf(t, err, tc.err, "GetTransactions should refuse %s", tc.name)
		})
	}

	// Serial, so the cases still run before the GetTransactions calls below rather than after them
	for _, tc := range []struct {
		name   string
		req    *TransactionsRequest
		params string
	}{
		{
			name:   "every optional param",
			req:    &TransactionsRequest{InstrumentName: "BTCUSD-PERP", JournalType: JournalTypeSessionSettle, StartTime: time.Unix(0, tradingTestTradesWindowStart), EndTime: time.Unix(0, tradingTestTradesWindowEnd), Limit: 100, IsolationID: tradingTestIsolationID},
			params: `{"instrument_name":"BTCUSD-PERP","journal_type":"SESSION_SETTLE","start_time":"1619089031996081486","end_time":"1619200052124211357","limit":100,"isolation_id":"6530219466236720402"}`,
		},
		{name: "no params", req: &TransactionsRequest{}, params: `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex := newSignedParamsServer(t, "private/get-transactions", tc.params, testResponse("private/get-transactions", `{"data":[]}`))
			_, err := ex.GetTransactions(t.Context(), tc.req)
			assert.NoErrorf(t, err, "GetTransactions should not error for %s", tc.name)
		})
	}
	req := &TransactionsRequest{InstrumentName: "BTCUSD-PERP", StartTime: time.Unix(0, tradingTestTradesWindowStart), EndTime: time.Unix(0, tradingTestTradesWindowEnd), Limit: 20}
	_, err := newResponseServerExchange(t, `{"id":1,"method":"private/get-transactions","code":0,"result":null}`).GetTransactions(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetTransactions should error for a null result")
	_, err = newUnavailableServerExchange(t).GetTransactions(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTransactions should return a request failure")

	skipLiveWithoutCredentials(t)
	if !mockTests {
		// The recorded window is older than the six months of history Crypto.com keeps
		req = &TransactionsRequest{Limit: 20}
	}
	result, err := e.GetTransactions(t.Context(), req)
	require.NoError(t, err, "GetTransactions must not error")
	if mockTests {
		exp := []TransactionResponse{
			{
				AccountID:           tradingTestAccountID,
				EventDate:           "2021-04-23",
				JournalType:         JournalTypeTrading,
				JournalID:           "4611686018427575078",
				TransactionQuantity: preciseNumber("-0.0005"),
				TransactionCost:     preciseNumber("-24.500000"),
				RealisedPNL:         preciseNumber("-0.006125"),
				OrderID:             "6530219466236720408",
				TradeID:             "1619170752166234567",
				TradeMatchID:        "4611686018427388625",
				EventTime:           types.Time(time.UnixMilli(1619170752166)),
				EventTimeNano:       types.Time(time.Unix(0, 1619170752166234567)),
				ClientOrderID:       "6ac2421d-5078-4ef6-a9d5-9680602ce123",
				TakerSide:           TakerSideMaker,
				Side:                SideSell,
				InstrumentName:      "BTCUSD-PERP",
			},
			{
				AccountID:           tradingTestAccountID,
				EventDate:           "2021-04-23",
				JournalType:         JournalTypeSessionSettle,
				JournalID:           "4611686018427574960",
				TransactionQuantity: preciseNumber("0"),
				TransactionCost:     preciseNumber("0.000000"),
				RealisedPNL:         preciseNumber("-0.007800"),
				TradeMatchID:        "0",
				EventTime:           types.Time(time.UnixMilli(1619168400002)),
				EventTimeNano:       types.Time(time.Unix(0, 1619168400002124563)),
				InstrumentName:      "BTCUSD-PERP",
				IsolationID:         tradingTestIsolationID,
				IsolationType:       IsolationTypeIsolatedMargin,
			},
			{
				AccountID:           tradingTestAccountID,
				EventDate:           "2021-04-23",
				JournalType:         JournalTypeSessionSettle,
				JournalID:           "4611686018427574959",
				TransactionQuantity: preciseNumber("0"),
				TransactionCost:     preciseNumber("0.000000"),
				RealisedPNL:         preciseNumber("-0.007800"),
				TradeMatchID:        "0",
				EventTime:           types.Time(time.UnixMilli(1619168400001)),
				EventTimeNano:       types.Time(time.Unix(0, 1619168400001124563)),
				InstrumentName:      "BTCUSD-PERP",
			},
		}
		assert.Equal(t, exp, result, "GetTransactions should decode every field")
	} else {
		for i := range result {
			assert.NotEmptyf(t, result[i].JournalType, "GetTransactions should return transaction %d's journal type", i)
		}
	}
}

func TestNewCreateOrderParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *CreateOrderRequest
		err  error
	}{
		{"a nil request", nil, common.ErrNilPointer},
		{"an order without an instrument", &CreateOrderRequest{Side: SideBuy, OrderType: OrderTypeMarket, Quantity: 0.0001}, errInstrumentNameRequired},
		{"an order without a side", &CreateOrderRequest{InstrumentName: "BTC_USD", OrderType: OrderTypeMarket, Quantity: 0.0001}, errInvalidSide},
		{"a lower-case order type", &CreateOrderRequest{InstrumentName: "BTC_USD", Side: SideBuy, OrderType: "market", Quantity: 0.0001}, errInvalidOrderType},
		{"a limit order with a negative price", &CreateOrderRequest{InstrumentName: "BTC_USD", Side: SideBuy, OrderType: OrderTypeLimit, Price: -1, Quantity: 0.0001}, errInvalidPrice},
		{"a market order with a negative price", &CreateOrderRequest{InstrumentName: "BTC_USD", Side: SideBuy, OrderType: OrderTypeMarket, Price: -1, Quantity: 0.0001}, errInvalidPrice},
		{"a market buy spending neither a quantity nor a notional", &CreateOrderRequest{InstrumentName: "BTC_USD", Side: SideBuy, OrderType: OrderTypeMarket}, errInvalidQuantity},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := newCreateOrderParams(tc.req)
			assert.ErrorIsf(t, err, tc.err, "newCreateOrderParams should refuse %s", tc.name)
			assert.Nilf(t, got, "newCreateOrderParams should return no params for %s", tc.name)
		})
	}
	for _, tc := range createOrderRefusals {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := tradingTestOrder()
			tc.modify(req)
			got, err := newCreateOrderParams(req)
			assert.ErrorIsf(t, err, tc.err, "newCreateOrderParams should refuse %s", tc.name)
			assert.Nilf(t, got, "newCreateOrderParams should return no params for %s", tc.name)
		})
	}
	for _, tc := range []struct {
		name string
		req  *CreateOrderRequest
		exp  *createOrderParams
	}{
		{
			name: "every optional param, its fee currency upper-cased",
			req: &CreateOrderRequest{
				InstrumentName:                 "BTCUSD-PERP",
				Side:                           SideBuy,
				OrderType:                      OrderTypeLimit,
				Price:                          50000.5,
				Quantity:                       0.0001,
				ClientOrderID:                  strings.Repeat("c", 36),
				TimeInForce:                    TimeInForceGoodTillCancel,
				ExecutionInstructions:          []string{ExecutionInstructionPostOnly, ExecutionInstructionReduceOnly, ExecutionInstructionIsolatedMargin, ExecutionInstructionMarginOrder},
				SpotMargin:                     SpotMarginMargin,
				FeeInstrumentName:              currency.NewCode("cro"),
				SelfTradePreventionID:          32767,
				SelfTradePreventionInstruction: SelfTradePreventionCancelBoth,
				SelfTradePreventionScope:       SelfTradePreventionScopeSubAccount,
				IsolatedMarginAmount:           50,
				IsolationID:                    tradingTestIsolationID,
				Leverage:                       10,
			},
			exp: &createOrderParams{
				InstrumentName:                 "BTCUSD-PERP",
				Side:                           SideBuy,
				OrderType:                      OrderTypeLimit,
				Price:                          50000.5,
				Quantity:                       0.0001,
				ClientOrderID:                  strings.Repeat("c", 36),
				TimeInForce:                    TimeInForceGoodTillCancel,
				ExecutionInstructions:          []string{ExecutionInstructionPostOnly, ExecutionInstructionReduceOnly, ExecutionInstructionIsolatedMargin, ExecutionInstructionMarginOrder},
				SpotMargin:                     SpotMarginMargin,
				FeeInstrumentName:              "CRO",
				SelfTradePreventionID:          32767,
				SelfTradePreventionInstruction: SelfTradePreventionCancelBoth,
				SelfTradePreventionScope:       SelfTradePreventionScopeSubAccount,
				IsolatedMarginAmount:           50,
				IsolationID:                    tradingTestIsolationID,
				Leverage:                       10,
			},
		},
		{
			name: "a market buy spending a notional",
			req:  &CreateOrderRequest{InstrumentName: "BTC_USD", Side: SideBuy, OrderType: OrderTypeMarket, Notional: 100},
			exp:  &createOrderParams{InstrumentName: "BTC_USD", Side: SideBuy, OrderType: OrderTypeMarket, Notional: 100},
		},
		{
			name: "a market sell of a quantity on spot",
			req:  &CreateOrderRequest{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeMarket, Quantity: 0.00001, SpotMargin: SpotMarginSpot},
			exp:  &createOrderParams{InstrumentName: "BTC_USD", Side: SideSell, OrderType: OrderTypeMarket, Quantity: 0.00001, SpotMargin: SpotMarginSpot},
		},
		{
			name: "a limit order of only the required params",
			req:  &CreateOrderRequest{InstrumentName: "ONE_USDT", Side: SideBuy, OrderType: OrderTypeLimit, Price: 0.0000013, Quantity: 1000},
			exp:  &createOrderParams{InstrumentName: "ONE_USDT", Side: SideBuy, OrderType: OrderTypeLimit, Price: 0.0000013, Quantity: 1000},
		},
		{
			name: "a post-only order resting by default and resetting self-trade prevention",
			req:  &CreateOrderRequest{InstrumentName: "BTCUSD-PERP", Side: SideBuy, OrderType: OrderTypeLimit, Price: 50000.5, Quantity: 0.0001, ExecutionInstructions: []string{ExecutionInstructionPostOnly}, SelfTradePreventionScope: SelfTradePreventionScopeReset},
			exp:  &createOrderParams{InstrumentName: "BTCUSD-PERP", Side: SideBuy, OrderType: OrderTypeLimit, Price: 50000.5, Quantity: 0.0001, ExecutionInstructions: []string{ExecutionInstructionPostOnly}, SelfTradePreventionScope: SelfTradePreventionScopeReset},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := newCreateOrderParams(tc.req)
			require.NoErrorf(t, err, "newCreateOrderParams must not error for %s", tc.name)
			assert.Equalf(t, tc.exp, got, "newCreateOrderParams should return the params of %s", tc.name)
		})
	}
}

func TestValidateOrderInstructions(t *testing.T) {
	t.Parallel()
	// Each order holds the instructions validateOrderInstructions is given, as the order methods pass them
	for _, tc := range []struct {
		name  string
		order *CreateOrderRequest
		err   error
	}{
		{name: "no instructions", order: &CreateOrderRequest{}},
		{
			name: "every instruction",
			order: &CreateOrderRequest{
				ClientOrderID:                  strings.Repeat("c", 36),
				TimeInForce:                    TimeInForceGoodTillCancel,
				ExecutionInstructions:          []string{ExecutionInstructionPostOnly, ExecutionInstructionReduceOnly},
				SelfTradePreventionScope:       SelfTradePreventionScopeSubAccount,
				SelfTradePreventionInstruction: SelfTradePreventionCancelMaker,
				SelfTradePreventionID:          32767,
			},
		},
		{name: "an order immediate or cancel", order: &CreateOrderRequest{TimeInForce: TimeInForceImmediateOrCancel}},
		{name: "an order fill or kill", order: &CreateOrderRequest{TimeInForce: TimeInForceFillOrKill}},
		{name: "a client order ID of 37 characters", order: &CreateOrderRequest{ClientOrderID: strings.Repeat("c", 37)}, err: errClientOrderIDTooLong},
		{name: "an abbreviated time in force", order: &CreateOrderRequest{TimeInForce: "GTC"}, err: errInvalidTimeInForce},
		{name: "a lower-case time in force", order: &CreateOrderRequest{TimeInForce: "good_till_cancel"}, err: errInvalidTimeInForce},
		{name: "an unknown execution instruction", order: &CreateOrderRequest{ExecutionInstructions: []string{"HIDDEN"}}, err: errInvalidExecutionInstruction},
		{
			name:  "a post-only order fill or kill",
			order: &CreateOrderRequest{TimeInForce: TimeInForceFillOrKill, ExecutionInstructions: []string{ExecutionInstructionPostOnly}},
			err:   errInvalidTimeInForce,
		},
		{
			name:  "an unknown self-trade prevention scope",
			order: &CreateOrderRequest{SelfTradePreventionScope: "X", SelfTradePreventionInstruction: SelfTradePreventionCancelMaker},
			err:   errInvalidSelfTradePreventionScope,
		},
		{name: "a self-trade prevention ID without a scope", order: &CreateOrderRequest{SelfTradePreventionID: 1}, err: errInvalidSelfTradePreventionID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			o := tc.order
			err := validateOrderInstructions(o.ClientOrderID, o.TimeInForce, o.ExecutionInstructions, o.SelfTradePreventionScope, o.SelfTradePreventionInstruction, o.SelfTradePreventionID)
			assert.ErrorIsf(t, err, tc.err, "validateOrderInstructions should return the expected error for %s", tc.name)
		})
	}
}

func TestValidateCreateExecutionInstructions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		instructions []string
		timeInForce  string
		err          error
	}{
		{name: "no instructions"},
		{name: "every instruction but SMART_POST_ONLY", instructions: []string{ExecutionInstructionPostOnly, ExecutionInstructionReduceOnly, ExecutionInstructionIsolatedMargin, ExecutionInstructionMarginOrder}},
		{name: "SMART_POST_ONLY", instructions: []string{ExecutionInstructionSmartPostOnly, ExecutionInstructionReduceOnly}},
		{name: "a post-only order resting by default", instructions: []string{ExecutionInstructionPostOnly}},
		{name: "a post-only order good till cancelled", instructions: []string{ExecutionInstructionPostOnly}, timeInForce: TimeInForceGoodTillCancel},
		{name: "an order immediate or cancel", instructions: []string{ExecutionInstructionReduceOnly}, timeInForce: TimeInForceImmediateOrCancel},
		{name: "a post-only order immediate or cancel", instructions: []string{ExecutionInstructionPostOnly}, timeInForce: TimeInForceImmediateOrCancel, err: errInvalidTimeInForce},
		{name: "a post-only order fill or kill", instructions: []string{ExecutionInstructionPostOnly}, timeInForce: TimeInForceFillOrKill, err: errInvalidTimeInForce},
		{name: "both post-only instructions", instructions: []string{ExecutionInstructionPostOnly, ExecutionInstructionSmartPostOnly}, err: errInvalidExecutionInstruction},
		{name: "both post-only instructions, SMART_POST_ONLY first", instructions: []string{ExecutionInstructionSmartPostOnly, ExecutionInstructionPostOnly}, err: errInvalidExecutionInstruction},
		{name: "the liquidation instruction, which is only reported", instructions: []string{ExecutionInstructionLiquidation}, err: errInvalidExecutionInstruction},
		{name: "a lower-case instruction", instructions: []string{"post_only"}, err: errInvalidExecutionInstruction},
		{name: "an empty instruction", instructions: []string{""}, err: errInvalidExecutionInstruction},
		{name: "an unknown instruction after a known one", instructions: []string{ExecutionInstructionReduceOnly, "HIDDEN"}, err: errInvalidExecutionInstruction},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateCreateExecutionInstructions(tc.instructions, tc.timeInForce)
			assert.ErrorIsf(t, err, tc.err, "validateCreateExecutionInstructions should return the expected error for %s", tc.name)
		})
	}
}

func TestValidateSelfTradePrevention(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, scope, instruction string
		id                       uint64
		err                      error
	}{
		{name: "nothing set"},
		{name: "a scope and an instruction", scope: SelfTradePreventionScopeMasterOrSubAccount, instruction: SelfTradePreventionCancelMaker},
		{name: "the largest ID", scope: SelfTradePreventionScopeSubAccount, instruction: SelfTradePreventionCancelTaker, id: 32767},
		{name: "a reset", scope: SelfTradePreventionScopeReset},
		{name: "a reset with an ID", scope: SelfTradePreventionScopeReset, id: 100},
		{name: "an instruction alone", instruction: SelfTradePreventionCancelBoth},
		{name: "a lower-case scope", scope: "m", instruction: SelfTradePreventionCancelMaker, err: errInvalidSelfTradePreventionScope},
		{name: "a lower-case instruction", scope: SelfTradePreventionScopeSubAccount, instruction: "t", err: errInvalidSelfTradePreventionInstruction},
		{name: "an unknown instruction", instruction: "X", err: errInvalidSelfTradePreventionInstruction},
		{name: "the master or sub-account scope without an instruction", scope: SelfTradePreventionScopeMasterOrSubAccount, err: errInvalidSelfTradePreventionInstruction},
		{name: "the sub-account scope without an instruction", scope: SelfTradePreventionScopeSubAccount, err: errInvalidSelfTradePreventionInstruction},
		{name: "an ID above 32767", scope: SelfTradePreventionScopeSubAccount, instruction: SelfTradePreventionCancelBoth, id: 32768, err: errInvalidSelfTradePreventionID},
		{name: "an ID alone", id: 1, err: errInvalidSelfTradePreventionID},
		{name: "an ID with an instruction but no scope", instruction: SelfTradePreventionCancelMaker, id: 5, err: errInvalidSelfTradePreventionID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateSelfTradePrevention(tc.scope, tc.instruction, tc.id)
			assert.ErrorIsf(t, err, tc.err, "validateSelfTradePrevention should return the expected error for scope %q, instruction %q and ID %d", tc.scope, tc.instruction, tc.id)
		})
	}
}

func TestAmendOrderRequestValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range amendOrderRefusals {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, tc.req.validate(), tc.err, "validate should refuse %s", tc.name)
		})
	}
	for _, tc := range []struct {
		name string
		req  *AmendOrderRequest
	}{
		{"an order named by its order ID", &AmendOrderRequest{OrderID: tradingTestOrderID, NewPrice: 50000, NewQuantity: 0.0002}},
		{"an order named by its original client order ID", &AmendOrderRequest{OriginalClientOrderID: tradingTestOrderClientOrderID, NewPrice: 50000, NewQuantity: 0.0002}},
		{
			name: "every field, with a client order ID of 36 characters",
			req:  &AmendOrderRequest{OrderID: tradingTestOrderID, OriginalClientOrderID: tradingTestOrderClientOrderID, ClientOrderID: strings.Repeat("c", 36), NewPrice: 50000, NewQuantity: 0.0002},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.NoErrorf(t, tc.req.validate(), "validate should accept %s", tc.name)
		})
	}
}

func TestCancelOrderRequestValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range cancelOrderRefusals {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, tc.req.validate(), tc.err, "validate should refuse %s", tc.name)
		})
	}
	for _, tc := range []struct {
		name string
		req  *CancelOrderRequest
	}{
		{"an order named by its order ID", &CancelOrderRequest{OrderID: tradingTestOrderID}},
		{"an order named by its client order ID", &CancelOrderRequest{ClientOrderID: tradingTestOrderClientOrderID}},
		{"both IDs", &CancelOrderRequest{OrderID: tradingTestOrderID, ClientOrderID: tradingTestOrderClientOrderID}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.NoErrorf(t, tc.req.validate(), "validate should accept %s", tc.name)
		})
	}
}

func TestCancelAllOrdersRequestValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range cancelAllOrdersRefusals {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, tc.req.validate(), tc.err, "validate should refuse %s", tc.name)
		})
	}
	for _, tc := range []struct {
		name string
		req  *CancelAllOrdersRequest
		err  error
	}{
		{"a lower-case type", &CancelAllOrdersRequest{OrderType: "all"}, errInvalidOrderType},
		{"no params", &CancelAllOrdersRequest{}, nil},
		{"an instrument's orders", &CancelAllOrdersRequest{InstrumentName: "BTCUSD-PERP"}, nil},
		{"limit orders", &CancelAllOrdersRequest{OrderType: CancelOrderTypeLimit}, nil},
		{"every order of an instrument", &CancelAllOrdersRequest{InstrumentName: "BTCUSD-PERP", OrderType: CancelOrderTypeAll}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, tc.req.validate(), tc.err, "validate should return the expected error for %s", tc.name)
		})
	}
}

func TestClosePositionRequestValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range closePositionRefusals {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, tc.req.validate(), tc.err, "validate should refuse %s", tc.name)
		})
	}
	for _, tc := range []struct {
		name string
		req  *ClosePositionRequest
		err  error
	}{
		{"a limit order with a negative price", &ClosePositionRequest{InstrumentName: "BTCUSD-PERP", OrderType: OrderTypeLimit, Price: -30000}, errInvalidPrice},
		{"a market order with a negative price", &ClosePositionRequest{InstrumentName: "BTCUSD-PERP", OrderType: OrderTypeMarket, Price: -30000}, errInvalidPrice},
		{"a market order closing the whole position", &ClosePositionRequest{InstrumentName: "BTCUSD-PERP", OrderType: OrderTypeMarket}, nil},
		{"a market order closing part of the position", &ClosePositionRequest{InstrumentName: "BTCUSD-PERP", OrderType: OrderTypeMarket, Quantity: 0.0001}, nil},
		{
			name: "a limit order closing part of an isolated position",
			req:  &ClosePositionRequest{InstrumentName: "BTCUSD-PERP", OrderType: OrderTypeLimit, Price: 30000, Quantity: 0.0001, IsolationID: tradingTestIsolationID},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, tc.req.validate(), tc.err, "validate should return the expected error for %s", tc.name)
		})
	}
}

func TestNewCreateOrderListParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		orders   []CreateOrderRequest
		err      error
		contains string
	}{
		{name: "an empty list", err: errOrderListEmpty},
		{name: "a list of 11 orders", orders: slices.Repeat([]CreateOrderRequest{*tradingTestOrder()}, maxOrderListLength+1), err: errOrderListTooLong},
		{
			name:     "a list starting with an order without an instrument",
			orders:   []CreateOrderRequest{{Side: SideBuy, OrderType: OrderTypeMarket, Quantity: 0.0001}, *tradingTestOrder()},
			err:      errInstrumentNameRequired,
			contains: "order 0:",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := newCreateOrderListParams(tc.orders)
			assert.ErrorIsf(t, err, tc.err, "newCreateOrderListParams should refuse %s", tc.name)
			if tc.contains != "" {
				assert.ErrorContainsf(t, err, tc.contains, "newCreateOrderListParams should name the index of %s", tc.name)
			}
			assert.Nilf(t, got, "newCreateOrderListParams should return no params for %s", tc.name)
		})
	}
	for _, tc := range createOrderRefusals {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			orders := []CreateOrderRequest{*tradingTestOrder(), *tradingTestOrder()}
			tc.modify(&orders[1])
			got, err := newCreateOrderListParams(orders)
			assert.ErrorIsf(t, err, tc.err, "newCreateOrderListParams should refuse %s", tc.name)
			assert.ErrorContainsf(t, err, "order 1:", "newCreateOrderListParams should name the index of %s", tc.name)
			assert.Nilf(t, got, "newCreateOrderListParams should return no params for %s", tc.name)
		})
	}
	testOrderParams := createOrderParams{
		InstrumentName:        "BTCUSD-PERP",
		Side:                  SideBuy,
		OrderType:             OrderTypeLimit,
		Price:                 50000.5,
		Quantity:              0.0001,
		ClientOrderID:         tradingTestOrderClientOrderID,
		TimeInForce:           TimeInForceGoodTillCancel,
		ExecutionInstructions: []string{ExecutionInstructionPostOnly},
	}
	for _, tc := range []struct {
		name   string
		orders []CreateOrderRequest
		exp    *createOrderListParams
	}{
		{
			name:   "a list of one order",
			orders: []CreateOrderRequest{*tradingTestOrder()},
			exp:    &createOrderListParams{ContingencyType: ContingencyTypeList, OrderList: []createOrderParams{testOrderParams}},
		},
		{
			name:   "the docs' list of two orders",
			orders: tradingTestOrderList(),
			exp: &createOrderListParams{
				ContingencyType: ContingencyTypeList,
				OrderList: []createOrderParams{
					{InstrumentName: "CRO_USD", Side: SideSell, OrderType: OrderTypeLimit, Price: 0.12, Quantity: 10, ClientOrderID: "api_leg1"},
					{InstrumentName: "CRO_USD", Side: SideSell, OrderType: OrderTypeLimit, Price: 0.122, Quantity: 20, ClientOrderID: "api_leg2"},
				},
			},
		},
		{
			name:   "a list of 10 orders",
			orders: slices.Repeat([]CreateOrderRequest{*tradingTestOrder()}, maxOrderListLength),
			exp:    &createOrderListParams{ContingencyType: ContingencyTypeList, OrderList: slices.Repeat([]createOrderParams{testOrderParams}, maxOrderListLength)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := newCreateOrderListParams(tc.orders)
			require.NoErrorf(t, err, "newCreateOrderListParams must not error for %s", tc.name)
			assert.Equalf(t, tc.exp, got, "newCreateOrderListParams should return the params of %s", tc.name)
		})
	}
}

func TestNewCancelOrderListParams(t *testing.T) {
	t.Parallel()
	for _, tc := range cancelOrderListRefusals {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := newCancelOrderListParams(tc.orders)
			assert.ErrorIsf(t, err, tc.err, "newCancelOrderListParams should refuse %s", tc.name)
			assert.Nilf(t, got, "newCancelOrderListParams should return no params for %s", tc.name)
		})
	}
	t.Run("the index of an order named by neither ID", func(t *testing.T) {
		t.Parallel()
		_, err := newCancelOrderListParams([]CancelOrderListEntryRequest{{InstrumentName: "CRO_USD"}, {OrderID: tradingTestOrderID}})
		assert.ErrorContains(t, err, "order 0:", "newCancelOrderListParams should name the index of the order named by neither ID")
	})
	for _, tc := range []struct {
		name   string
		orders []CancelOrderListEntryRequest
	}{
		{"an order named by its client order ID", []CancelOrderListEntryRequest{{InstrumentName: "CRO_USD", ClientOrderID: "api_leg1"}}},
		{
			"an order with every field and one named by its order ID only",
			[]CancelOrderListEntryRequest{{InstrumentName: "CRO_USD", OrderID: "5755600460443882762", ClientOrderID: "api_leg1"}, {OrderID: "5755600460443882763"}},
		},
		{"a list of 10 orders", slices.Repeat([]CancelOrderListEntryRequest{{OrderID: tradingTestOrderID}}, maxOrderListLength)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := newCancelOrderListParams(tc.orders)
			require.NoErrorf(t, err, "newCancelOrderListParams must not error for %s", tc.name)
			assert.Equalf(t, &cancelOrderListParams{ContingencyType: ContingencyTypeList, OrderList: tc.orders}, got, "newCancelOrderListParams should return the params of %s", tc.name)
		})
	}
}

func TestCheckOrderListLength(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		length int
		err    error
	}{
		{"no orders", 0, errOrderListEmpty},
		{"one order", 1, nil},
		{"as many orders as the limit", maxOrderListLength, nil},
		{"an order more than the limit", maxOrderListLength + 1, errOrderListTooLong},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// The limits are passed as the constants the order list methods pass, as the function's nolint:unparam holds
			// only while every call passes 10
			assert.ErrorIsf(t, checkOrderListLength(tc.length, maxOrderListLength), tc.err, "checkOrderListLength should return the expected error for %d orders", tc.length)
			assert.ErrorIsf(t, checkOrderListLength(tc.length, maxAdvancedOrderListLength), tc.err, "checkOrderListLength should return the expected error for %d advanced orders", tc.length)
		})
	}
}

func TestNewTradingHistoryParams(t *testing.T) {
	t.Parallel()
	start, end := time.Unix(0, tradingTestTradesWindowStart), time.Unix(0, tradingTestTradesWindowEnd)
	// Each request holds the query newTradingHistoryParams is given, as GetTransactions passes it
	for _, tc := range []struct {
		name string
		req  *TransactionsRequest
		exp  *tradingHistoryParams
		err  error
	}{
		{name: "a window starting after it ends", req: &TransactionsRequest{StartTime: end, EndTime: start}, err: common.ErrStartAfterEnd},
		{name: "a window starting as it ends", req: &TransactionsRequest{StartTime: start, EndTime: start}, err: common.ErrStartEqualsEnd},
		{name: "a window starting in the future", req: &TransactionsRequest{StartTime: time.Now().Add(time.Hour), EndTime: time.Now().Add(2 * time.Hour)}, err: common.ErrStartAfterTimeNow},
		{name: "a limit above 100", req: &TransactionsRequest{Limit: maxTradingHistoryLimit + 1}, err: errInvalidCount},
		{name: "a start before the Unix epoch", req: &TransactionsRequest{StartTime: time.Unix(-1, 0)}, err: errTimeBeforeUnixEpoch},
		{name: "an end before the Unix epoch", req: &TransactionsRequest{EndTime: time.Unix(-1, 0)}, err: errTimeBeforeUnixEpoch},
		{name: "a window before the Unix epoch", req: &TransactionsRequest{StartTime: time.Unix(-2, 0), EndTime: time.Unix(-1, 0)}, err: errTimeBeforeUnixEpoch},
		{name: "no params", req: &TransactionsRequest{}, exp: &tradingHistoryParams{}},
		{
			name: "every param",
			req:  &TransactionsRequest{InstrumentName: "BTCUSD-PERP", JournalType: JournalTypeSessionSettle, StartTime: start, EndTime: end, Limit: maxTradingHistoryLimit, IsolationID: tradingTestIsolationID},
			exp: &tradingHistoryParams{
				InstrumentName: "BTCUSD-PERP",
				JournalType:    JournalTypeSessionSettle,
				StartTime:      tradingTestTradesWindowStart,
				EndTime:        tradingTestTradesWindowEnd,
				Limit:          maxTradingHistoryLimit,
				IsolationID:    tradingTestIsolationID,
			},
		},
		{name: "a start alone", req: &TransactionsRequest{StartTime: start}, exp: &tradingHistoryParams{StartTime: tradingTestTradesWindowStart}},
		{name: "an end alone", req: &TransactionsRequest{EndTime: end}, exp: &tradingHistoryParams{EndTime: tradingTestTradesWindowEnd}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := tc.req
			got, err := newTradingHistoryParams(r.InstrumentName, r.JournalType, r.StartTime, r.EndTime, r.Limit, r.IsolationID)
			assert.ErrorIsf(t, err, tc.err, "newTradingHistoryParams should return the expected error for %s", tc.name)
			assert.Equalf(t, tc.exp, got, "newTradingHistoryParams should return the params of %s", tc.name)
		})
	}
}

func TestNewBatchOrderParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		req      *BatchOrderRequest
		err      error
		contains string
	}{
		{name: "a nil request", err: common.ErrNilPointer},
		{name: "a batch without an instrument", req: &BatchOrderRequest{Cancels: []CancelOrderRequest{{OrderID: tradingTestOrderID}}}, err: errInstrumentNameRequired},
		{name: "an empty batch", req: &BatchOrderRequest{InstrumentName: "BTCUSD-PERP"}, err: errOrderListEmpty},
		{
			name: "a batch of 11 creates",
			req:  &BatchOrderRequest{InstrumentName: "BTCUSD-PERP", Creates: slices.Repeat([]CreateOrderRequest{*tradingTestOrder()}, maxBatchCreates+1)},
			err:  errOrderListTooLong,
		},
		{
			name:     "a cancel naming no order",
			req:      &BatchOrderRequest{InstrumentName: "BTCUSD-PERP", Cancels: []CancelOrderRequest{{OrderID: tradingTestOrderID}, {}}},
			err:      errOrderIDRequired,
			contains: "cancel 1:",
		},
		{
			name:     "a create of another instrument",
			req:      &BatchOrderRequest{InstrumentName: "ETHUSD-PERP", Creates: []CreateOrderRequest{*tradingTestOrder()}},
			err:      errBatchInstrumentMismatch,
			contains: "create 0:",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := newBatchOrderParams(tc.req)
			assert.ErrorIsf(t, err, tc.err, "newBatchOrderParams should refuse %s", tc.name)
			if tc.contains != "" {
				assert.ErrorContainsf(t, err, tc.contains, "newBatchOrderParams should name the index of %s", tc.name)
			}
			assert.Nilf(t, got, "newBatchOrderParams should return no params for %s", tc.name)
		})
	}
	for _, tc := range createOrderRefusals {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := tradingTestBatch()
			tc.modify(&req.Creates[1])
			got, err := newBatchOrderParams(req)
			assert.ErrorIsf(t, err, tc.err, "newBatchOrderParams should refuse %s", tc.name)
			assert.ErrorContainsf(t, err, "create 1:", "newBatchOrderParams should name the index of %s", tc.name)
			assert.Nilf(t, got, "newBatchOrderParams should return no params for %s", tc.name)
		})
	}
	tenCreates := func() *BatchOrderRequest {
		return &BatchOrderRequest{InstrumentName: "BTCUSD-PERP", Creates: slices.Repeat([]CreateOrderRequest{{Side: SideBuy, OrderType: OrderTypeLimit, Price: 41000, Quantity: 0.0001}}, maxBatchCreates)}
	}
	for _, tc := range []struct {
		name string
		req  func() *BatchOrderRequest
		exp  *batchOrderParams
	}{
		{
			name: "the docs' batch, its creates inheriting its instrument",
			req:  tradingTestBatch,
			exp: &batchOrderParams{
				InstrumentName: "BTCUSD-PERP",
				Cancels:        []CancelOrderRequest{{OrderID: tradingTestOrderID}, {ClientOrderID: "my-client-order-id"}},
				Creates: []createOrderParams{
					{
						Side:                  SideBuy,
						OrderType:             OrderTypeLimit,
						Price:                 41555,
						Quantity:              0.01,
						ClientOrderID:         "optional-client-id",
						TimeInForce:           TimeInForceGoodTillCancel,
						ExecutionInstructions: []string{ExecutionInstructionPostOnly, ExecutionInstructionReduceOnly},
					},
					{Side: SideBuy, OrderType: OrderTypeLimit, Price: 41000, Quantity: 0.0001, ExecutionInstructions: []string{ExecutionInstructionPostOnly}},
				},
			},
		},
		{
			name: "every optional param, with a create naming the batch's instrument",
			req: func() *BatchOrderRequest {
				return &BatchOrderRequest{
					InstrumentName:                        "BTCUSD-PERP",
					Cancels:                               []CancelOrderRequest{{OrderID: tradingTestOrderID, ClientOrderID: tradingTestOrderClientOrderID}},
					Creates:                               []CreateOrderRequest{{InstrumentName: "BTCUSD-PERP", Side: SideSell, OrderType: OrderTypeMarket, Quantity: 0.01}},
					SelfTradePreventionCancelType:         SelfTradePreventionCancelMaker,
					SelfTradePreventionAccountEnforcement: true,
					SelfTradePreventionGroup:              "1",
				}
			},
			exp: &batchOrderParams{
				InstrumentName:                        "BTCUSD-PERP",
				Cancels:                               []CancelOrderRequest{{OrderID: tradingTestOrderID, ClientOrderID: tradingTestOrderClientOrderID}},
				Creates:                               []createOrderParams{{Side: SideSell, OrderType: OrderTypeMarket, Quantity: 0.01}},
				SelfTradePreventionCancelType:         SelfTradePreventionCancelMaker,
				SelfTradePreventionAccountEnforcement: true,
				SelfTradePreventionGroup:              "1",
			},
		},
		{
			name: "only cancels",
			req: func() *BatchOrderRequest {
				return &BatchOrderRequest{InstrumentName: "BTCUSD-PERP", Cancels: []CancelOrderRequest{{ClientOrderID: "my-client-order-id"}}}
			},
			exp: &batchOrderParams{InstrumentName: "BTCUSD-PERP", Cancels: []CancelOrderRequest{{ClientOrderID: "my-client-order-id"}}},
		},
		{
			name: "only 10 creates",
			req:  tenCreates,
			exp: &batchOrderParams{
				InstrumentName: "BTCUSD-PERP",
				Creates:        slices.Repeat([]createOrderParams{{Side: SideBuy, OrderType: OrderTypeLimit, Price: 41000, Quantity: 0.0001}}, maxBatchCreates),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := tc.req()
			got, err := newBatchOrderParams(req)
			require.NoErrorf(t, err, "newBatchOrderParams must not error for %s", tc.name)
			assert.Equalf(t, tc.exp, got, "newBatchOrderParams should return the params of %s", tc.name)
			assert.Equalf(t, tc.req(), req, "newBatchOrderParams should leave the request of %s as it was", tc.name)
		})
	}
}

func TestBareIntegerFields(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, data string
		into, exp  any
	}{
		{"a bare order reason", `{"reason":43012}`, new(OrderResponse), &OrderResponse{Reason: "43012"}},
		{"a quoted order reason", `{"reason":"43012"}`, new(OrderResponse), &OrderResponse{Reason: "43012"}},
		{"a bare self-trade prevention ID", `{"stp_id":100}`, new(AccountSettingsResponse), &AccountSettingsResponse{SelfTradePreventionID: 100}},
		{"a quoted self-trade prevention ID", `{"stp_id":"100"}`, new(AccountSettingsResponse), &AccountSettingsResponse{SelfTradePreventionID: 100}},
		{"a bare isolation ID", `{"isolation_id":19848526}`, new(TransactionResponse), &TransactionResponse{IsolationID: "19848526"}},
		{"a quoted isolation ID", `{"isolation_id":"19848526"}`, new(TransactionResponse), &TransactionResponse{IsolationID: "19848526"}},
		{"a bare staked token convert ID", `{"convert_id":1}`, new(ConvertStakedTokenResponse), &ConvertStakedTokenResponse{ConvertID: "1"}},
		{"a quoted staked token convert ID", `{"convert_id":"1"}`, new(ConvertStakedTokenResponse), &ConvertStakedTokenResponse{ConvertID: "1"}},
		{"a bare convert request ID", `{"convert_id":1}`, new(StakingConvertRequestResponse), &StakingConvertRequestResponse{ConvertID: "1"}},
		{"a quoted convert request ID", `{"convert_id":"1"}`, new(StakingConvertRequestResponse), &StakingConvertRequestResponse{ConvertID: "1"}},
		{"bare fiat history paging", `{"page":2,"page_size":10,"total":30}`, new(FiatTransactionHistoryResponse), &FiatTransactionHistoryResponse{Page: 2, PageSize: 10, Total: 30}},
		{"quoted fiat history paging", `{"page":"2","page_size":"10","total":"30"}`, new(FiatTransactionHistoryResponse), &FiatTransactionHistoryResponse{Page: 2, PageSize: 10, Total: 30}},
		{"a bare fiat transaction account ID", `{"account_id":6871391474021396480}`, new(FiatTransaction), &FiatTransaction{AccountID: "6871391474021396480"}},
		{"a quoted fiat transaction account ID", `{"account_id":"6871391474021396480"}`, new(FiatTransaction), &FiatTransaction{AccountID: "6871391474021396480"}},
		{"a bare confirmation count", `{"confirmation_required":12}`, new(CurrencyNetwork), &CurrencyNetwork{ConfirmationsRequired: 12}},
		{"a quoted confirmation count", `{"confirmation_required":"12"}`, new(CurrencyNetwork), &CurrencyNetwork{ConfirmationsRequired: 12}},
		{"a bare stake's pre-stake charge rate", `{"pre_stake_charge_rate_in_bps":50}`, new(StakeResponse), &StakeResponse{PreStakeChargeRateBasisPoints: 50}},
		{"a quoted stake's pre-stake charge rate", `{"pre_stake_charge_rate_in_bps":"50"}`, new(StakeResponse), &StakeResponse{PreStakeChargeRateBasisPoints: 50}},
		{"a bare staking instrument's pre-stake charge rate", `{"pre_stake_charge_rate_in_bps":50}`, new(StakingInstrumentResponse), &StakingInstrumentResponse{PreStakeChargeRateBasisPoints: 50}},
		{"a quoted staking instrument's pre-stake charge rate", `{"pre_stake_charge_rate_in_bps":"50"}`, new(StakingInstrumentResponse), &StakingInstrumentResponse{PreStakeChargeRateBasisPoints: 50}},
		{"bare trade match counts", `{"match_count":2,"match_index":1}`, new(UserTradeResponse), &UserTradeResponse{MatchCount: 2, MatchIndex: 1}},
		{"quoted trade match counts", `{"match_count":"2","match_index":"1"}`, new(UserTradeResponse), &UserTradeResponse{MatchCount: 2, MatchIndex: 1}},
		{"bare fee tiers", `{"spot_tier":1,"deriv_tier":2}`, new(FeeRateResponse), &FeeRateResponse{SpotTier: 1, DerivativesTier: 2}},
		{"quoted fee tiers", `{"spot_tier":"1","deriv_tier":"2"}`, new(FeeRateResponse), &FeeRateResponse{SpotTier: 1, DerivativesTier: 2}},
		{"a bare executed order count", `{"executed_order":5}`, new(TradingBotExecutionSummary), &TradingBotExecutionSummary{ExecutedOrders: 5}},
		{"a quoted executed order count", `{"executed_order":"5"}`, new(TradingBotExecutionSummary), &TradingBotExecutionSummary{ExecutedOrders: 5}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.NoErrorf(t, json.Unmarshal([]byte(tc.data), tc.into), "Unmarshal must not error for %s into %T", tc.data, tc.into)
			assert.Equalf(t, tc.exp, tc.into, "Unmarshal should decode %s into %T", tc.data, tc.into)
		})
	}
}

func TestNumericIDUnmarshalJSON(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same function
	t.Parallel()
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, tc := range []struct {
		name, data string
		exp        NumericID
	}{
		{"a quoted identifier", `"6498090546073120152"`, "6498090546073120152"},
		{"a bare identifier", `6498090546073120152`, "6498090546073120152"},
		// 2^53 + 1, which a float cannot hold
		{"beyond a float's precision", `9007199254740993`, "9007199254740993"},
		{"a single digit", `1`, "1"},
		{"an empty string", `""`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var direct NumericID
			require.NoErrorf(t, direct.UnmarshalJSON([]byte(tc.data)), "UnmarshalJSON must not error for %s", tc.data)
			assert.Equalf(t, tc.exp, direct, "UnmarshalJSON should decode %s digit for digit", tc.data)
			var n NumericID
			require.NoErrorf(t, json.Unmarshal([]byte(tc.data), &n), "Unmarshal must not error for %s", tc.data)
			assert.Equalf(t, tc.exp, n, "Unmarshal should decode %s digit for digit", tc.data)
		})
	}
	unchanged := NumericID("7")
	require.NoError(t, unchanged.UnmarshalJSON([]byte(`null`)), "UnmarshalJSON must not error for null")
	assert.Equal(t, NumericID("7"), unchanged, "UnmarshalJSON should leave an identifier unchanged for null")
	n := NumericID("7")
	require.NoError(t, json.Unmarshal([]byte(`null`), &n), "Unmarshal must not error for null")
	assert.Equal(t, NumericID("7"), n, "Unmarshal should leave an identifier unchanged for null")
	// The cases decode into n in order, before the check below that reuses it
	for _, tc := range []struct{ name, data string }{
		{"a decimal", `1.5`},
		{"an exponent", `1e3`},
		{"a boolean", `true`},
		{"an object", `{}`},
		{"an array", `[]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			refused := NumericID("7")
			assert.ErrorIsf(t, refused.UnmarshalJSON([]byte(tc.data)), errInvalidInteger, "UnmarshalJSON should refuse %s", tc.data)
			assert.Equalf(t, NumericID("7"), refused, "UnmarshalJSON should leave an identifier unchanged refusing %s", tc.data)
			assert.ErrorIsf(t, json.Unmarshal([]byte(tc.data), &n), errInvalidInteger, "Unmarshal should refuse %s", tc.data)
		})
	}
	assert.ErrorIs(t, n.UnmarshalJSON([]byte(`"6498`)), errInvalidInteger, "UnmarshalJSON should refuse a string it cannot decode")
}
