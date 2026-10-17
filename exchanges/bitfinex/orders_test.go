package bitfinex

import (
	"context"
	"math"
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
	// testOrderID and testClientOrderID identify the order the documented write examples act on
	testOrderID       uint64 = 1747566428
	testClientOrderID uint64 = 1678987199446
)

var (
	// testClientOrderIDDate is the UTC day testClientOrderID belongs to
	testClientOrderIDDate = time.Date(2023, time.March, 16, 0, 0, 0, 0, time.UTC)

	// testActiveOrder is the order the documented Retrieve Orders examples return
	testActiveOrder = Order{
		ID: 33950998275, ClientOrderID: 1573476747887, Symbol: "tETHUSD", CreatedAt: types.Time(time.UnixMilli(1573476748000)), UpdatedAt: types.Time(time.UnixMilli(1573476748000)),
		Amount: -0.5, OriginalAmount: -0.5, Type: OrderTypeLimit, Status: "ACTIVE", Price: 220, Routing: "BFX",
	}
	// testSubmittedOrder is the order the documented Submit Order and Cancel Orders (multiple) examples return
	testSubmittedOrder = Order{
		ID: testOrderID, ClientOrderID: testClientOrderID, Symbol: "tBTCUSD", CreatedAt: types.Time(time.UnixMilli(1678987199446)), UpdatedAt: types.Time(time.UnixMilli(1678987199448)),
		Amount: 0.1, OriginalAmount: 0.1, Type: ExchangeOrderTypePrefix + OrderTypeLimit, Status: "ACTIVE", Price: 10000, Routing: "API>BFX",
	}
	// testUpdatedOrder is the order the documented Update Order example returns
	testUpdatedOrder = Order{
		ID: testOrderID, ClientOrderID: testClientOrderID, Symbol: "tBTCUSD", CreatedAt: types.Time(time.UnixMilli(1678987199446)), UpdatedAt: types.Time(time.UnixMilli(1678988263843)),
		Amount: 0.25, OriginalAmount: 0.1, Type: ExchangeOrderTypePrefix + OrderTypeLimit, PreviousType: ExchangeOrderTypePrefix + OrderTypeLimit, Status: "ACTIVE", Price: 25000, Routing: "API>BFX",
	}
	// testCancelledOrder is the order the documented Cancel Order example returns
	testCancelledOrder = Order{
		ID: testOrderID, ClientOrderID: testClientOrderID, Symbol: "tBTCUSD", CreatedAt: types.Time(time.UnixMilli(1678987199446)), UpdatedAt: types.Time(time.UnixMilli(1678989283489)),
		Amount: 0.25, OriginalAmount: 0.1, Type: ExchangeOrderTypePrefix + OrderTypeLimit, PreviousType: ExchangeOrderTypePrefix + OrderTypeLimit, Status: "CANCELED", Price: 25000, Routing: "API>BFX",
	}
	// testHistoricOrder is the order the documented Orders History examples return
	testHistoricOrder = Order{
		ID: 33961681942, GroupID: 1227, ClientOrderID: 1337, Symbol: "tBTCUSD", CreatedAt: types.Time(time.UnixMilli(1573482478000)), UpdatedAt: types.Time(time.UnixMilli(1573485373000)),
		Amount: 0.001, OriginalAmount: 0.001, Type: ExchangeOrderTypePrefix + OrderTypeLimit, Status: "CANCELED", Price: 15, Routing: "API>BFX",
	}
	// testOTCOrder is the order the documented OTC Orders History example returns
	testOTCOrder = OTCOrder{
		ID: 123, Symbol: "tBTCUSD", CreatedAt: types.Time(time.UnixMilli(1669126974000)), UpdatedAt: types.Time(time.UnixMilli(1670596411000)), IsInitiator: true,
		InitiatorNickname: "john-doe", CounterpartyNickname: "jane-btc", Amount: 0.025, Price: 26325, Status: "REJECTED",
	}
	// testLedgerEntry is the entry the documented Ledgers example returns
	testLedgerEntry = LedgerEntry{
		ID: 2531822314, Currency: currency.USD, Wallet: "margin", Time: types.Time(time.UnixMilli(1573521810000)), Amount: 0.01644445, Description: "Settlement @ 185.79 on wallet margin",
	}
)

// liveTestOrderRequest returns an exchange limit buy on tBTCUSD priced at half the last price, so it rests on the book
// without filling
func liveTestOrderRequest(t *testing.T) *PlaceOrderRequest {
	t.Helper()
	tick, err := e.GetTicker(t.Context(), "tBTCUSD")
	require.NoError(t, err, "GetTicker must not error")
	// Bitfinex takes prices of five significant digits
	price := math.Round(tick.LastPrice/2000) * 1000
	require.Positive(t, price, "The live test order must have a price")
	return &PlaceOrderRequest{Type: ExchangeOrderTypePrefix + OrderTypeLimit, Symbol: "tBTCUSD", Amount: 0.0002, Price: price}
}

// cancelLiveTestOrderOnCleanup cancels an order the test placed once it ends, whether or not it reached its own
// cancellation; cancelling an order already cancelled fails harmlessly
func cancelLiveTestOrderOnCleanup(t *testing.T, id uint64) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = e.CancelExistingOrder(context.WithoutCancel(t.Context()), &CancelExistingOrderRequest{ID: id})
	})
}

// placeLiveTestOrder places liveTestOrderRequest's order, to be cancelled once the test ends
func placeLiveTestOrder(t *testing.T) *Order {
	t.Helper()
	resp, err := e.PlaceOrder(t.Context(), liveTestOrderRequest(t))
	require.NoError(t, err, "PlaceOrder must not error")
	require.Len(t, resp.Data, 1, "PlaceOrder must return the order placed")
	cancelLiveTestOrderOnCleanup(t, resp.Data[0].ID)
	return &resp.Data[0]
}

// assertBodyJSON asserts the JSON a request body marshals to
func assertBodyJSON(t *testing.T, exp string, body any) {
	t.Helper()
	encoded, err := json.Marshal(body)
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, exp, string(encoded), "The body should marshal to the request Bitfinex takes")
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOpenOrders must reject a nil request")
	_, err = e.GetOpenOrders(t.Context(), &OpenOrdersRequest{ClientOrderID: 1})
	assert.ErrorIs(t, err, errClientOrderIDDateUnset, "GetOpenOrders should refuse a client order ID without its date")

	skipLiveWithoutCredentials(t)
	req := &OpenOrdersRequest{}
	if mockTests {
		req = &OpenOrdersRequest{IDs: []uint64{33950998275}, ClientOrderID: 1573476747887, ClientOrderIDDate: time.Date(2019, time.November, 11, 0, 0, 0, 0, time.UTC)}
	}
	result, err := e.GetOpenOrders(t.Context(), req)
	require.NoError(t, err, "GetOpenOrders must not error")
	if mockTests {
		exp := []Order{testActiveOrder}
		assert.Equal(t, exp, result, "GetOpenOrders should decode every field")
	} else {
		for i := range result {
			assert.NotZero(t, result[i].ID, "GetOpenOrders should return each order's ID")
		}
	}

	_, err = newUnavailableServerExchange(t).GetOpenOrders(t.Context(), &OpenOrdersRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOpenOrders should return the request failure")
}

func TestGetOpenOrdersBySymbol(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenOrdersBySymbol(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOpenOrdersBySymbol must reject a nil request")
	for _, symbol := range []string{"", "t", "fUSD", "BTCUSD"} {
		_, err = e.GetOpenOrdersBySymbol(t.Context(), &OpenOrdersBySymbolRequest{Symbol: symbol})
		assert.ErrorIsf(t, err, errTradingSymbolRequired, "GetOpenOrdersBySymbol should refuse symbol %q", symbol)
	}
	_, err = e.GetOpenOrdersBySymbol(t.Context(), &OpenOrdersBySymbolRequest{Symbol: "tBTCUSD", ClientOrderID: 1})
	assert.ErrorIs(t, err, errClientOrderIDDateUnset, "GetOpenOrdersBySymbol should refuse a client order ID without its date")

	skipLiveWithoutCredentials(t)
	symbol := "tBTCUSD"
	if mockTests {
		symbol = "tETHUSD"
	}
	result, err := e.GetOpenOrdersBySymbol(t.Context(), &OpenOrdersBySymbolRequest{Symbol: symbol})
	require.NoError(t, err, "GetOpenOrdersBySymbol must not error")
	if mockTests {
		exp := []Order{testActiveOrder}
		assert.Equal(t, exp, result, "GetOpenOrdersBySymbol should decode every field")
	} else {
		for i := range result {
			assert.Equal(t, symbol, result[i].Symbol, "GetOpenOrdersBySymbol should return only the pair's orders")
		}
	}

	_, err = newUnavailableServerExchange(t).GetOpenOrdersBySymbol(t.Context(), &OpenOrdersBySymbolRequest{Symbol: "tBTCUSD"})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOpenOrdersBySymbol should return the request failure")
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceOrder must reject a nil request")
	for _, orderType := range []string{"", "LIMITED", "limit", ExchangeOrderTypePrefix, ExchangeOrderTypePrefix + ExchangeOrderTypePrefix + OrderTypeLimit} {
		_, err = e.PlaceOrder(t.Context(), &PlaceOrderRequest{Type: orderType, Symbol: "tBTCUSD", Amount: 1, Price: 1})
		assert.ErrorIsf(t, err, errInvalidOrderType, "PlaceOrder should refuse order type %q", orderType)
	}
	_, err = e.PlaceOrder(t.Context(), &PlaceOrderRequest{Type: OrderTypeLimit, Symbol: "fUSD", Amount: 1, Price: 1})
	assert.ErrorIs(t, err, errTradingSymbolRequired, "PlaceOrder should refuse a funding currency symbol")
	_, err = e.PlaceOrder(t.Context(), &PlaceOrderRequest{Type: OrderTypeLimit, Symbol: "tBTCUSD", Price: 1})
	assert.ErrorIs(t, err, errInvalidAmount, "PlaceOrder should refuse a zero amount")
	_, err = e.PlaceOrder(t.Context(), &PlaceOrderRequest{Type: OrderTypeLimit, Symbol: "tBTCUSD", Amount: 1, Price: -1})
	assert.ErrorIs(t, err, errInvalidPrice, "PlaceOrder should refuse a negative price")

	skipLiveUnlessCanManipulate(t)
	if mockTests {
		result, err := e.PlaceOrder(t.Context(), &PlaceOrderRequest{Type: ExchangeOrderTypePrefix + OrderTypeLimit, Symbol: "tBTCUSD", Amount: 0.1, Price: 10000, ClientOrderID: testClientOrderID})
		require.NoError(t, err, "PlaceOrder must not error")
		exp := &Notification[[]Order]{
			Time:   types.Time(time.Unix(1678987199, 0)),
			Type:   "on-req",
			Data:   []Order{testSubmittedOrder},
			Status: NotificationStatusSuccess,
			Text:   "Submitting exchange limit buy order for 0.1 BTC.",
		}
		assert.Equal(t, exp, result, "PlaceOrder should decode every field")
	} else {
		placed := placeLiveTestOrder(t)
		assert.Equal(t, "ACTIVE", placed.Status, "PlaceOrder should place an active order")
	}

	req := &PlaceOrderRequest{Type: OrderTypeLimit, Symbol: "tBTCUSD", Amount: 1, Price: 1}
	_, err = newResponseServerExchange(t, `null`).PlaceOrder(t.Context(), req)
	assert.ErrorIs(t, err, common.ErrNoResponse, "PlaceOrder should error for a null response")
	_, err = newUnavailableServerExchange(t).PlaceOrder(t.Context(), req)
	assert.ErrorIs(t, err, request.ErrBadStatus, "PlaceOrder should return the request failure")
}

func TestUpdateOrder(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "UpdateOrder must reject a nil request")
	_, err = e.UpdateOrder(t.Context(), &UpdateOrderRequest{Price: 1})
	assert.ErrorIs(t, err, errOrderIDRequired, "UpdateOrder should refuse a request without an order ID")
	_, err = e.UpdateOrder(t.Context(), &UpdateOrderRequest{ID: 1, Price: -1})
	assert.ErrorIs(t, err, errInvalidPrice, "UpdateOrder should refuse a negative price")

	skipLiveUnlessCanManipulate(t)
	if mockTests {
		result, err := e.UpdateOrder(t.Context(), &UpdateOrderRequest{ID: testOrderID, Amount: 0.25, Price: 25000})
		require.NoError(t, err, "UpdateOrder must not error")
		exp := &Notification[Order]{
			Time:   types.Time(time.UnixMilli(1678988263842)),
			Type:   "ou-req",
			Data:   testUpdatedOrder,
			Status: NotificationStatusSuccess,
			Text:   "Submitting update to exchange limit buy order for 0.1 BTC.",
		}
		assert.Equal(t, exp, result, "UpdateOrder should decode every field")
	} else {
		placed := placeLiveTestOrder(t)
		result, err := e.UpdateOrder(t.Context(), &UpdateOrderRequest{ID: placed.ID, Price: placed.Price - 1000})
		require.NoError(t, err, "UpdateOrder must not error")
		assert.Equal(t, placed.ID, result.Data.ID, "UpdateOrder should return the order updated")
	}

	_, err = newResponseServerExchange(t, `null`).UpdateOrder(t.Context(), &UpdateOrderRequest{ID: 1})
	assert.ErrorIs(t, err, common.ErrNoResponse, "UpdateOrder should error for a null response")
	_, err = newUnavailableServerExchange(t).UpdateOrder(t.Context(), &UpdateOrderRequest{ID: 1})
	assert.ErrorIs(t, err, request.ErrBadStatus, "UpdateOrder should return the request failure")
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelExistingOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelExistingOrder must reject a nil request")
	_, err = e.CancelExistingOrder(t.Context(), &CancelExistingOrderRequest{ClientOrderIDDate: testClientOrderIDDate})
	assert.ErrorIs(t, err, errOrderIDRequired, "CancelExistingOrder should refuse a request without an order ID or client order ID")
	_, err = e.CancelExistingOrder(t.Context(), &CancelExistingOrderRequest{ClientOrderID: 1})
	assert.ErrorIs(t, err, errClientOrderIDDateUnset, "CancelExistingOrder should refuse a client order ID without its date")

	skipLiveUnlessCanManipulate(t)
	if mockTests {
		exp := &Notification[Order]{
			Time:   types.Time(time.UnixMilli(1678989283488)),
			Type:   "oc-req",
			Data:   testCancelledOrder,
			Status: NotificationStatusSuccess,
			Text:   "Submitted for cancellation; waiting for confirmation (ID: 1747566428).",
		}
		for _, req := range []*CancelExistingOrderRequest{{ID: testOrderID}, {ClientOrderID: testClientOrderID, ClientOrderIDDate: testClientOrderIDDate}} {
			result, err := e.CancelExistingOrder(t.Context(), req)
			require.NoErrorf(t, err, "CancelExistingOrder must not error for %+v", req)
			assert.Equalf(t, exp, result, "CancelExistingOrder should decode every field for %+v", req)
		}
	} else {
		placed := placeLiveTestOrder(t)
		result, err := e.CancelExistingOrder(t.Context(), &CancelExistingOrderRequest{ID: placed.ID})
		require.NoError(t, err, "CancelExistingOrder must not error")
		assert.Equal(t, placed.ID, result.Data.ID, "CancelExistingOrder should return the order cancelled")
	}

	_, err = newResponseServerExchange(t, `null`).CancelExistingOrder(t.Context(), &CancelExistingOrderRequest{ID: 1})
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelExistingOrder should error for a null response")
	_, err = newUnavailableServerExchange(t).CancelExistingOrder(t.Context(), &CancelExistingOrderRequest{ID: 1})
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelExistingOrder should return the request failure")
}

func TestCancelOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelOrders must reject a nil request")
	_, err = e.CancelOrders(t.Context(), &CancelOrdersRequest{})
	assert.ErrorIs(t, err, errOrdersToCancelRequired, "CancelOrders should refuse a request selecting no orders")
	_, err = e.CancelOrders(t.Context(), &CancelOrdersRequest{ClientOrderIDs: []DatedClientOrderID{{ID: 1, Date: testClientOrderIDDate}, {ID: 2}}})
	assert.ErrorIs(t, err, errClientOrderIDDateUnset, "CancelOrders should refuse a client order ID without its date")

	skipLiveUnlessCanManipulate(t)
	if mockTests {
		result, err := e.CancelOrders(t.Context(), &CancelOrdersRequest{IDs: []uint64{testOrderID}, ClientOrderIDs: []DatedClientOrderID{{ID: testClientOrderID, Date: testClientOrderIDDate}}})
		require.NoError(t, err, "CancelOrders must not error")
		exp := &Notification[[]Order]{
			Time:   types.Time(time.UnixMilli(1568711312683)),
			Type:   "oc_multi-req",
			Data:   []Order{testSubmittedOrder},
			Status: NotificationStatusSuccess,
			Text:   "Submitting 2 order cancellations.",
		}
		assert.Equal(t, exp, result, "CancelOrders should decode every field")
	} else {
		placed := placeLiveTestOrder(t)
		result, err := e.CancelOrders(t.Context(), &CancelOrdersRequest{IDs: []uint64{placed.ID}})
		require.NoError(t, err, "CancelOrders must not error")
		require.Len(t, result.Data, 1, "CancelOrders must return the order cancelled")
		assert.Equal(t, placed.ID, result.Data[0].ID, "CancelOrders should return the order cancelled")
	}

	_, err = newResponseServerExchange(t, `null`).CancelOrders(t.Context(), &CancelOrdersRequest{All: true})
	assert.ErrorIs(t, err, common.ErrNoResponse, "CancelOrders should error for a null response")
	_, err = newUnavailableServerExchange(t).CancelOrders(t.Context(), &CancelOrdersRequest{All: true})
	assert.ErrorIs(t, err, request.ErrBadStatus, "CancelOrders should return the request failure")
}

func TestOrderMultiOp(t *testing.T) {
	t.Parallel()
	_, err := e.OrderMultiOp(t.Context(), nil)
	require.ErrorIs(t, err, errOrderOperationsRequired, "OrderMultiOp must refuse no operations")
	ops := make([]OrderOperation, maxOrderOperations+1)
	for i := range ops {
		ops[i] = &CancelExistingOrderRequest{ID: uint64(i) + 1}
	}
	_, err = e.OrderMultiOp(t.Context(), ops)
	assert.ErrorIs(t, err, errTooManyOrderOperations, "OrderMultiOp should refuse more operations than Bitfinex takes")
	for _, op := range []OrderOperation{nil, (*PlaceOrderRequest)(nil), (*UpdateOrderRequest)(nil), (*CancelExistingOrderRequest)(nil), (*CancelOrdersRequest)(nil)} {
		_, err = e.OrderMultiOp(t.Context(), []OrderOperation{&CancelExistingOrderRequest{ID: 1}, op})
		assert.ErrorIsf(t, err, common.ErrNilPointer, "OrderMultiOp should refuse a nil %T operation", op)
	}
	for _, tc := range []struct {
		op  OrderOperation
		err error
	}{
		{&PlaceOrderRequest{Type: OrderTypeLimit, Symbol: "tBTCUSD", Price: 1}, errInvalidAmount},
		{&UpdateOrderRequest{Price: 1}, errOrderIDRequired},
		{&CancelExistingOrderRequest{}, errOrderIDRequired},
		{&CancelOrdersRequest{}, errOrdersToCancelRequired},
	} {
		_, err = e.OrderMultiOp(t.Context(), []OrderOperation{tc.op})
		assert.ErrorIsf(t, err, tc.err, "OrderMultiOp should return the validation error of a %T operation", tc.op)
	}

	skipLiveUnlessCanManipulate(t)
	if mockTests {
		result, err := e.OrderMultiOp(t.Context(), []OrderOperation{
			&PlaceOrderRequest{Type: OrderTypeLimit, Symbol: "tBTCUSD", Amount: 0.001, Price: 15, ClientOrderID: 1569347312970, Flags: OrderFlagPostOnly},
			&PlaceOrderRequest{Type: OrderTypeLimit, Symbol: "tBTCUSD", Amount: 0.001, Price: 15, ClientOrderID: 1569347312977},
			&UpdateOrderRequest{ID: testOrderID, Amount: 0.25, Price: 25000},
			&CancelExistingOrderRequest{ID: testOrderID},
			&CancelOrdersRequest{IDs: []uint64{testOrderID}},
		})
		require.NoError(t, err, "OrderMultiOp must not error")
		exp := &Notification[[]Notification[OrderOperationOrders]]{
			Time: types.Time(time.UnixMilli(1569347312984)),
			Type: "ox_multi-req",
			Data: []Notification[OrderOperationOrders]{
				{
					Time: types.Time(time.UnixMilli(1569347312977)),
					Type: "on-req",
					Data: OrderOperationOrders{Orders: []Order{{
						ID: 31492970019, ClientOrderID: 1569347312970, Symbol: "tBTCUSD", CreatedAt: types.Time(time.UnixMilli(1569347312971)), UpdatedAt: types.Time(time.UnixMilli(1569347312971)),
						Amount: 0.001, OriginalAmount: 0.001, Type: OrderTypeLimit, Flags: OrderFlagPostOnly, Status: "ACTIVE", Price: 15, Routing: "API>BFX", Meta: json.RawMessage(`{"$F7":1}`),
					}}},
					Status: NotificationStatusSuccess,
					Text:   "Submitting 1 orders.",
				},
				{
					Time: types.Time(time.UnixMilli(1569347312984)),
					Type: "on-req",
					Data: OrderOperationOrders{Orders: []Order{{
						ID: 31492970020, ClientOrderID: 1569347312977, Symbol: "tBTCUSD", CreatedAt: types.Time(time.UnixMilli(1569347312978)), UpdatedAt: types.Time(time.UnixMilli(1569347312978)),
						Amount: 0.001, OriginalAmount: 0.001, Type: OrderTypeLimit, Status: "ACTIVE", Price: 15, Routing: "API>BFX",
					}}},
					Status: NotificationStatusSuccess,
					Text:   "Submitting 1 orders.",
				},
				{
					Time:   types.Time(time.UnixMilli(1678988263842)),
					Type:   "ou-req",
					Data:   OrderOperationOrders{Orders: []Order{testUpdatedOrder}},
					Status: NotificationStatusSuccess,
					Text:   "Submitting update to exchange limit buy order for 0.1 BTC.",
				},
				{
					Time:   types.Time(time.UnixMilli(1678989283488)),
					Type:   "oc-req",
					Data:   OrderOperationOrders{Orders: []Order{testCancelledOrder}},
					Status: NotificationStatusSuccess,
					Text:   "Submitted for cancellation; waiting for confirmation (ID: 1747566428).",
				},
				{
					Time:   types.Time(time.UnixMilli(1568711312683)),
					Type:   "oc_multi-req",
					Data:   OrderOperationOrders{Orders: []Order{testSubmittedOrder}},
					Status: NotificationStatusSuccess,
					Text:   "Submitting 2 order cancellations.",
				},
			},
			Status: NotificationStatusSuccess,
			Text:   "Submitting 5 order operations.",
		}
		assert.Equal(t, exp, result, "OrderMultiOp should decode every operation's notification")
	} else {
		submitted, err := e.OrderMultiOp(t.Context(), []OrderOperation{liveTestOrderRequest(t)})
		require.NoError(t, err, "OrderMultiOp must not error submitting an order")
		require.Len(t, submitted.Data, 1, "OrderMultiOp must return the submission's notification")
		require.NoError(t, submitted.Data[0].Err(), "OrderMultiOp's submission must succeed")
		require.Len(t, submitted.Data[0].Data.Orders, 1, "OrderMultiOp must return the order submitted")
		id := submitted.Data[0].Data.Orders[0].ID
		cancelLiveTestOrderOnCleanup(t, id)
		cancelled, err := e.OrderMultiOp(t.Context(), []OrderOperation{&CancelExistingOrderRequest{ID: id}})
		require.NoError(t, err, "OrderMultiOp must not error cancelling an order")
		require.Len(t, cancelled.Data, 1, "OrderMultiOp must return the cancellation's notification")
		assert.NoError(t, cancelled.Data[0].Err(), "OrderMultiOp's cancellation should succeed")
	}

	ops = []OrderOperation{&CancelExistingOrderRequest{ID: 1}}
	_, err = newResponseServerExchange(t, `null`).OrderMultiOp(t.Context(), ops)
	assert.ErrorIs(t, err, common.ErrNoResponse, "OrderMultiOp should error for a null response")
	_, err = newUnavailableServerExchange(t).OrderMultiOp(t.Context(), ops)
	assert.ErrorIs(t, err, request.ErrBadStatus, "OrderMultiOp should return the request failure")
}

func TestGetOrdersHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrdersHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOrdersHistory must reject a nil request")
	_, err = e.GetOrdersHistory(t.Context(), &OrdersHistoryRequest{Start: time.UnixMilli(1573486000000), End: time.UnixMilli(1573482000000)})
	assert.ErrorIs(t, err, common.ErrStartAfterEnd, "GetOrdersHistory should refuse a reversed window")

	skipLiveWithoutCredentials(t)
	req := &OrdersHistoryRequest{Start: time.Now().AddDate(0, 0, -14), End: time.Now()}
	if mockTests {
		req = &OrdersHistoryRequest{Start: time.UnixMilli(1573482000000), End: time.UnixMilli(1573486000000), Limit: 25, IDs: []uint64{33961681942}}
	}
	result, err := e.GetOrdersHistory(t.Context(), req)
	require.NoError(t, err, "GetOrdersHistory must not error")
	if mockTests {
		exp := []Order{testHistoricOrder}
		assert.Equal(t, exp, result, "GetOrdersHistory should decode every field")
	} else {
		for i := range result {
			assert.NotZero(t, result[i].ID, "GetOrdersHistory should return each order's ID")
		}
	}

	_, err = newUnavailableServerExchange(t).GetOrdersHistory(t.Context(), &OrdersHistoryRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrdersHistory should return the request failure")
}

func TestGetOrdersHistoryBySymbol(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrdersHistoryBySymbol(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOrdersHistoryBySymbol must reject a nil request")
	for _, symbol := range []string{"", "t", "fUSD", "BTCUSD"} {
		_, err = e.GetOrdersHistoryBySymbol(t.Context(), &OrdersHistoryBySymbolRequest{Symbol: symbol})
		assert.ErrorIsf(t, err, errTradingSymbolRequired, "GetOrdersHistoryBySymbol should refuse symbol %q", symbol)
	}
	_, err = e.GetOrdersHistoryBySymbol(t.Context(), &OrdersHistoryBySymbolRequest{Symbol: "tBTCUSD", Start: time.UnixMilli(1573486000000), End: time.UnixMilli(1573482000000)})
	assert.ErrorIs(t, err, common.ErrStartAfterEnd, "GetOrdersHistoryBySymbol should refuse a reversed window")

	skipLiveWithoutCredentials(t)
	req := &OrdersHistoryBySymbolRequest{Symbol: "tBTCUSD", Start: time.Now().AddDate(0, 0, -14), End: time.Now()}
	if mockTests {
		req = &OrdersHistoryBySymbolRequest{Symbol: "tBTCUSD", Limit: 25}
	}
	result, err := e.GetOrdersHistoryBySymbol(t.Context(), req)
	require.NoError(t, err, "GetOrdersHistoryBySymbol must not error")
	if mockTests {
		exp := []Order{testHistoricOrder}
		assert.Equal(t, exp, result, "GetOrdersHistoryBySymbol should decode every field")
	} else {
		for i := range result {
			assert.Equal(t, req.Symbol, result[i].Symbol, "GetOrdersHistoryBySymbol should return only the pair's orders")
		}
	}

	_, err = newUnavailableServerExchange(t).GetOrdersHistoryBySymbol(t.Context(), &OrdersHistoryBySymbolRequest{Symbol: "tBTCUSD"})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrdersHistoryBySymbol should return the request failure")
}

func TestGetOrderTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOrderTrades must reject a nil request")
	for _, symbol := range []string{"", "t", "fUSD", "BTCUSD"} {
		_, err = e.GetOrderTrades(t.Context(), &OrderTradesRequest{Symbol: symbol, OrderID: 1})
		assert.ErrorIsf(t, err, errTradingSymbolRequired, "GetOrderTrades should refuse symbol %q", symbol)
	}
	_, err = e.GetOrderTrades(t.Context(), &OrderTradesRequest{Symbol: "tBTCUSD"})
	assert.ErrorIs(t, err, errOrderIDRequired, "GetOrderTrades should refuse a request without an order ID")

	skipLiveWithoutCredentials(t)
	req := &OrderTradesRequest{Symbol: "tETHUSD", OrderID: 33963608932}
	if !mockTests {
		trades, err := e.GetAccountTrades(t.Context(), &AccountTradesRequest{Limit: 1})
		require.NoError(t, err, "GetAccountTrades must not error")
		if len(trades) == 0 {
			t.Skip("The account has no trades whose order's trades could be returned")
		}
		req = &OrderTradesRequest{Symbol: trades[0].Symbol, OrderID: trades[0].OrderID}
	}
	result, err := e.GetOrderTrades(t.Context(), req)
	require.NoError(t, err, "GetOrderTrades must not error")
	if mockTests {
		exp := []AccountTrade{{
			ID: 399251013, Symbol: "tETHUSD", ExecutedAt: types.Time(time.UnixMilli(1573485493000)), OrderID: 33963608932, ExecutedAmount: 0.26334268, ExecutedPrice: 187.37,
			OrderType: ExchangeOrderTypePrefix + OrderTypeMarket, Maker: -1, Fee: -0.09868591, FeeCurrency: currency.USD, ClientOrderID: 1234,
		}}
		assert.Equal(t, exp, result, "GetOrderTrades should decode every field")
	} else {
		for i := range result {
			assert.Equal(t, req.OrderID, result[i].OrderID, "GetOrderTrades should return only the order's trades")
		}
	}

	_, err = newUnavailableServerExchange(t).GetOrderTrades(t.Context(), &OrderTradesRequest{Symbol: "tBTCUSD", OrderID: 1})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrderTrades should return the request failure")
}

func TestGetAccountTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAccountTrades must reject a nil request")
	_, err = e.GetAccountTrades(t.Context(), &AccountTradesRequest{Start: time.UnixMilli(1574970000000), End: time.UnixMilli(1574960000000)})
	assert.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAccountTrades should refuse a reversed window")
	_, err = e.GetAccountTrades(t.Context(), &AccountTradesRequest{Sort: 2})
	assert.ErrorIs(t, err, errInvalidSort, "GetAccountTrades should refuse an invalid sort")

	skipLiveWithoutCredentials(t)
	req := &AccountTradesRequest{End: time.Now(), Limit: 25}
	if mockTests {
		req = &AccountTradesRequest{Start: time.UnixMilli(1574960000000), End: time.UnixMilli(1574970000000), Limit: 25, Sort: -1}
	}
	result, err := e.GetAccountTrades(t.Context(), req)
	require.NoError(t, err, "GetAccountTrades must not error")
	if mockTests {
		exp := []AccountTrade{{
			ID: 402088407, Symbol: "tETHUST", ExecutedAt: types.Time(time.UnixMilli(1574963975602)), OrderID: 34938060782, ExecutedAmount: -0.2, ExecutedPrice: 153.57,
			OrderType: OrderTypeMarket, Maker: -1, Fee: -0.061668, FeeCurrency: currency.USD, ClientOrderID: 1234,
		}}
		assert.Equal(t, exp, result, "GetAccountTrades should decode every field")
	} else {
		for i := range result {
			assert.NotZero(t, result[i].OrderID, "GetAccountTrades should return each trade's order")
		}
	}

	_, err = newUnavailableServerExchange(t).GetAccountTrades(t.Context(), &AccountTradesRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAccountTrades should return the request failure")
}

func TestGetAccountTradesBySymbol(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountTradesBySymbol(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAccountTradesBySymbol must reject a nil request")
	for _, symbol := range []string{"", "t", "fUSD", "BTCUSD"} {
		_, err = e.GetAccountTradesBySymbol(t.Context(), &AccountTradesBySymbolRequest{Symbol: symbol})
		assert.ErrorIsf(t, err, errTradingSymbolRequired, "GetAccountTradesBySymbol should refuse symbol %q", symbol)
	}
	_, err = e.GetAccountTradesBySymbol(t.Context(), &AccountTradesBySymbolRequest{Symbol: "tBTCUSD", Start: time.UnixMilli(1574970000000), End: time.UnixMilli(1574960000000)})
	assert.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAccountTradesBySymbol should refuse a reversed window")
	_, err = e.GetAccountTradesBySymbol(t.Context(), &AccountTradesBySymbolRequest{Symbol: "tBTCUSD", Sort: -2})
	assert.ErrorIs(t, err, errInvalidSort, "GetAccountTradesBySymbol should refuse an invalid sort")

	skipLiveWithoutCredentials(t)
	req := &AccountTradesBySymbolRequest{Symbol: "tBTCUSD", End: time.Now(), Limit: 25}
	if mockTests {
		req = &AccountTradesBySymbolRequest{Symbol: "tETHUST", Sort: 1}
	}
	result, err := e.GetAccountTradesBySymbol(t.Context(), req)
	require.NoError(t, err, "GetAccountTradesBySymbol must not error")
	if mockTests {
		exp := []AccountTrade{{
			ID: 402088407, Symbol: "tETHUST", ExecutedAt: types.Time(time.UnixMilli(1574963975602)), OrderID: 34938060782, ExecutedAmount: -0.2, ExecutedPrice: 153.57,
			OrderType: OrderTypeMarket, Maker: -1, Fee: -0.061668, FeeCurrency: currency.USD, ClientOrderID: 1234,
		}}
		assert.Equal(t, exp, result, "GetAccountTradesBySymbol should decode every field")
	} else {
		for i := range result {
			assert.Equal(t, req.Symbol, result[i].Symbol, "GetAccountTradesBySymbol should return only the pair's trades")
		}
	}

	_, err = newUnavailableServerExchange(t).GetAccountTradesBySymbol(t.Context(), &AccountTradesBySymbolRequest{Symbol: "tBTCUSD"})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetAccountTradesBySymbol should return the request failure")
}

func TestGetOTCOrdersHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOTCOrdersHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOTCOrdersHistory must reject a nil request")
	for _, symbol := range []string{"t", "fUSD", "BTCUSD"} {
		_, err = e.GetOTCOrdersHistory(t.Context(), &OTCOrdersHistoryRequest{Symbol: symbol})
		assert.ErrorIsf(t, err, errTradingSymbolRequired, "GetOTCOrdersHistory should refuse symbol %q", symbol)
	}
	_, err = e.GetOTCOrdersHistory(t.Context(), &OTCOrdersHistoryRequest{Start: time.UnixMilli(1670600000000), End: time.UnixMilli(1669000000000)})
	assert.ErrorIs(t, err, common.ErrStartAfterEnd, "GetOTCOrdersHistory should refuse a reversed window")

	skipLiveWithoutCredentials(t)
	reqs := []*OTCOrdersHistoryRequest{{}, {Symbol: "tBTCUSD"}}
	if mockTests {
		reqs = []*OTCOrdersHistoryRequest{{}, {Symbol: "tBTCUSD", Start: time.UnixMilli(1669000000000), End: time.UnixMilli(1670600000000), Limit: 25, IDs: []uint64{123}}}
	}
	for _, req := range reqs {
		result, err := e.GetOTCOrdersHistory(t.Context(), req)
		require.NoErrorf(t, err, "GetOTCOrdersHistory must not error for symbol %q", req.Symbol)
		if mockTests {
			exp := []OTCOrder{testOTCOrder}
			assert.Equalf(t, exp, result, "GetOTCOrdersHistory should decode every field for symbol %q", req.Symbol)
		} else {
			for i := range result {
				assert.NotZerof(t, result[i].ID, "GetOTCOrdersHistory should return each order's ID for symbol %q", req.Symbol)
			}
		}
	}

	_, err = newUnavailableServerExchange(t).GetOTCOrdersHistory(t.Context(), &OTCOrdersHistoryRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOTCOrdersHistory should return the request failure")
}

func TestGetLedgers(t *testing.T) {
	t.Parallel()
	_, err := e.GetLedgers(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLedgers must reject a nil request")
	_, err = e.GetLedgers(t.Context(), &LedgersRequest{Category: 6})
	assert.ErrorIs(t, err, errInvalidLedgerCategory, "GetLedgers should refuse an undocumented category")
	_, err = e.GetLedgers(t.Context(), &LedgersRequest{Start: time.UnixMilli(1573530000000), End: time.UnixMilli(1573500000000)})
	assert.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLedgers should refuse a reversed window")

	skipLiveWithoutCredentials(t)
	reqs := []*LedgersRequest{{}, {Currency: currency.USD, End: time.Now(), Limit: 25}}
	if mockTests {
		// The lower case currency is sent upper case
		reqs = []*LedgersRequest{{}, {Currency: currency.NewCode("usd"), Category: LedgerCategorySettlement, Start: time.UnixMilli(1573500000000), End: time.UnixMilli(1573530000000), Limit: 25, Wallet: "margin"}}
	}
	for _, req := range reqs {
		result, err := e.GetLedgers(t.Context(), req)
		require.NoErrorf(t, err, "GetLedgers must not error for currency %q", req.Currency)
		if mockTests {
			exp := []LedgerEntry{testLedgerEntry}
			assert.Equalf(t, exp, result, "GetLedgers should decode every field for currency %q", req.Currency)
		} else {
			for i := range result {
				assert.NotZerof(t, result[i].ID, "GetLedgers should return each entry's ID for currency %q", req.Currency)
			}
		}
	}

	_, err = newUnavailableServerExchange(t).GetLedgers(t.Context(), &LedgersRequest{})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLedgers should return the request failure")
}

func TestNewOpenOrdersBody(t *testing.T) {
	t.Parallel()
	body, err := newOpenOrdersBody([]uint64{1, 2}, 3, 4, time.Date(2023, time.March, 16, 23, 30, 0, 0, time.FixedZone("UTC-1", -60*60)))
	require.NoError(t, err, "newOpenOrdersBody must not error")
	assertBodyJSON(t, `{"id":[1,2],"gid":3,"cid":4,"cid_date":"2023-03-17"}`, body)
	body, err = newOpenOrdersBody(nil, 0, 0, time.Time{})
	require.NoError(t, err, "newOpenOrdersBody must not error without filters")
	assertBodyJSON(t, `{}`, body)
	_, err = newOpenOrdersBody(nil, 0, 4, time.Time{})
	assert.ErrorIs(t, err, errClientOrderIDDateUnset, "newOpenOrdersBody should refuse a client order ID without its date")
}

func TestPlaceOrderRequestBody(t *testing.T) {
	t.Parallel()
	_, err := (*PlaceOrderRequest)(nil).body()
	require.ErrorIs(t, err, common.ErrNilPointer, "body must reject a nil request")
	body, err := (&PlaceOrderRequest{
		Type:                OrderTypeStopLimit,
		Symbol:              "tBTCF0:USTF0",
		Amount:              -0.00000001,
		Price:               25000.5,
		Leverage:            20,
		TrailingPrice:       10,
		AuxiliaryLimitPrice: 24999,
		OCOStopPrice:        26000,
		GroupID:             7,
		ClientOrderID:       8,
		Flags:               OrderFlagHidden | OrderFlagOneCancelsOther,
		TimeInForce:         time.Date(2020, time.January, 15, 21, 45, 23, 0, time.FixedZone("AEDT", 11*60*60)),
		Meta:                &OrderMeta{AffiliateCode: "abc", MakeVisible: true, ProtectSelfMatch: true},
	}).body()
	require.NoError(t, err, "body must not error")
	assertBodyJSON(t, `{"type":"STOP LIMIT","symbol":"tBTCF0:USTF0","amount":"-0.00000001","price":"25000.5","lev":20,"price_trailing":"10","price_aux_limit":"24999","price_oco_stop":"26000","gid":7,"cid":8,"flags":16448,"tif":"2020-01-15 10:45:23","meta":{"aff_code":"abc","make_visible":1,"protect_selfmatch":1}}`, body)
	body, err = (&PlaceOrderRequest{Type: ExchangeOrderTypePrefix + OrderTypeMarket, Symbol: "tBTCUSD", Amount: 1}).body()
	require.NoError(t, err, "body must not error for a market order without a price")
	assertBodyJSON(t, `{"type":"EXCHANGE MARKET","symbol":"tBTCUSD","amount":"1","price":"0"}`, body)
}

func TestUpdateOrderRequestBody(t *testing.T) {
	t.Parallel()
	_, err := (*UpdateOrderRequest)(nil).body()
	require.ErrorIs(t, err, common.ErrNilPointer, "body must reject a nil request")
	body, err := (&UpdateOrderRequest{
		ID:                  1,
		Amount:              -0.5,
		Price:               100,
		Delta:               0.1,
		ClientOrderID:       2,
		ClientOrderIDDate:   testClientOrderIDDate,
		GroupID:             3,
		Flags:               new(uint64(0)),
		Leverage:            5,
		TrailingPrice:       1.5,
		AuxiliaryLimitPrice: 99,
		TimeInForce:         time.Date(2020, time.June, 30, 12, 0, 0, 0, time.UTC),
		Meta:                &OrderMeta{MakeVisible: true},
	}).body()
	require.NoError(t, err, "body must not error")
	assertBodyJSON(t, `{"id":1,"amount":"-0.5","price":"100","delta":"0.1","cid":2,"cid_date":"2023-03-16","gid":3,"flags":0,"lev":5,"price_trailing":"1.5","price_aux_limit":"99","tif":"2020-06-30 12:00:00","meta":{"make_visible":1}}`, body)
	body, err = (&UpdateOrderRequest{ID: 1}).body()
	require.NoError(t, err, "body must not error for an update of nothing")
	assertBodyJSON(t, `{"id":1}`, body)
}

func TestCancelExistingOrderRequestBody(t *testing.T) {
	t.Parallel()
	_, err := (*CancelExistingOrderRequest)(nil).body()
	require.ErrorIs(t, err, common.ErrNilPointer, "body must reject a nil request")
	body, err := (&CancelExistingOrderRequest{ID: 1}).body()
	require.NoError(t, err, "body must not error for an order ID")
	assertBodyJSON(t, `{"id":1}`, body)
	body, err = (&CancelExistingOrderRequest{ClientOrderID: 2, ClientOrderIDDate: time.Date(2023, time.March, 16, 23, 30, 0, 0, time.FixedZone("UTC-1", -60*60))}).body()
	require.NoError(t, err, "body must not error for a client order ID")
	assertBodyJSON(t, `{"cid":2,"cid_date":"2023-03-17"}`, body)
}

func TestCancelOrdersRequestBody(t *testing.T) {
	t.Parallel()
	_, err := (*CancelOrdersRequest)(nil).body()
	require.ErrorIs(t, err, common.ErrNilPointer, "body must reject a nil request")
	body, err := (&CancelOrdersRequest{
		IDs:            []uint64{1, 2},
		GroupIDs:       []uint64{3},
		ClientOrderIDs: []DatedClientOrderID{{ID: 4, Date: testClientOrderIDDate}, {ID: 5, Date: time.Date(2023, time.March, 16, 23, 30, 0, 0, time.FixedZone("UTC-1", -60*60))}},
		All:            true,
	}).body()
	require.NoError(t, err, "body must not error")
	assertBodyJSON(t, `{"id":[1,2],"gid":[3],"cid":[[4,"2023-03-16"],[5,"2023-03-17"]],"all":1}`, body)
	body, err = (&CancelOrdersRequest{All: true}).body()
	require.NoError(t, err, "body must not error cancelling every order")
	assertBodyJSON(t, `{"all":1}`, body)
}

func TestOrderOperation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		op           OrderOperation
		abbreviation string
		body         string
	}{
		{&PlaceOrderRequest{Type: OrderTypeMarket, Symbol: "tBTCUSD", Amount: 1}, "on", `{"type":"MARKET","symbol":"tBTCUSD","amount":"1","price":"0"}`},
		{&UpdateOrderRequest{ID: 1, Price: 2}, "ou", `{"id":1,"price":"2"}`},
		{&CancelExistingOrderRequest{ID: 1}, "oc", `{"id":1}`},
		{&CancelOrdersRequest{All: true}, "oc_multi", `{"all":1}`},
	} {
		abbreviation, body, err := tc.op.operation()
		require.NoErrorf(t, err, "operation must not error for %T", tc.op)
		assert.Equalf(t, tc.abbreviation, abbreviation, "operation should give the abbreviation of a %T", tc.op)
		assertBodyJSON(t, tc.body, body)
	}
	for _, op := range []OrderOperation{&PlaceOrderRequest{}, &UpdateOrderRequest{}, &CancelExistingOrderRequest{}, &CancelOrdersRequest{}} {
		_, _, err := op.operation()
		assert.Errorf(t, err, "operation should return the validation error of a %T", op)
	}
}

func TestOrderMetaBody(t *testing.T) {
	t.Parallel()
	assert.Nil(t, (*OrderMeta)(nil).body(), "body should send no meta object without meta data")
	assertBodyJSON(t, `{}`, (&OrderMeta{}).body())
	assertBodyJSON(t, `{"aff_code":"abc","make_visible":1,"protect_selfmatch":1}`, (&OrderMeta{AffiliateCode: "abc", MakeVisible: true, ProtectSelfMatch: true}).body())
}

func TestOrderOperationOrdersUnmarshalJSON(t *testing.T) {
	t.Parallel()
	testTupleDecode(t, `[1,null,2,"tBTCUSD"]`, OrderOperationOrders{Orders: []Order{{ID: 1, ClientOrderID: 2, Symbol: "tBTCUSD"}}})
	testTupleDecode(t, ` [ [1,null,2,"tBTCUSD"], [3] ] `, OrderOperationOrders{Orders: []Order{{ID: 1, ClientOrderID: 2, Symbol: "tBTCUSD"}, {ID: 3}}})
	testTupleDecode(t, `[]`, OrderOperationOrders{Orders: []Order{}})
	// An operation's notification, whose data the tuple decoder hands to OrderOperationOrders
	testTupleDecode(t, `[1678989283488,"oc-req",null,null,[1,null,2,"tBTCUSD"],null,"SUCCESS","Submitted"]`,
		Notification[OrderOperationOrders]{Time: types.Time(time.UnixMilli(1678989283488)), Type: "oc-req", Data: OrderOperationOrders{Orders: []Order{{ID: 1, ClientOrderID: 2, Symbol: "tBTCUSD"}}}, Status: NotificationStatusSuccess, Text: "Submitted"})

	reused := OrderOperationOrders{Orders: []Order{{ID: 9, Symbol: "tETHUSD", Price: 9}}}
	require.NoError(t, reused.UnmarshalJSON([]byte(`[[1,null,2,null]]`)), "UnmarshalJSON must not error into decoded orders")
	assert.Equal(t, OrderOperationOrders{Orders: []Order{{ID: 1, ClientOrderID: 2}}}, reused, "UnmarshalJSON should not keep an earlier decode's fields where the new orders are null")

	var orders OrderOperationOrders
	assert.ErrorIs(t, orders.UnmarshalJSON([]byte(`[1,"x"]`)), errTupleElementType, "UnmarshalJSON should refuse a malformed order")
	assert.ErrorIs(t, orders.UnmarshalJSON([]byte(`[["x"]]`)), errTupleElementType, "UnmarshalJSON should refuse a malformed order in a list")
	assert.Error(t, orders.UnmarshalJSON([]byte(`{}`)), "UnmarshalJSON should refuse an object")
}

// TestOrdersRecordsEveryPosition decodes each orders record from an array with a distinct value at every tagged
// position, so a field tagged with the wrong position or type cannot pass on a null or zero element
func TestOrdersRecordsEveryPosition(t *testing.T) {
	t.Parallel()
	testTupleDecode(t, `[1000,"s1",1700000000002,1700000000003,null,1,"s6","s7",null,9.5,10.5,null,"s12",1700000000013]`,
		OTCOrder{ID: 1000, Symbol: "s1", CreatedAt: types.Time(time.UnixMilli(1700000000002)), UpdatedAt: types.Time(time.UnixMilli(1700000000003)), IsInitiator: true, InitiatorNickname: "s6", CounterpartyNickname: "s7", Amount: 9.5, Price: 10.5, Status: "s12", TimeInForce: types.Time(time.UnixMilli(1700000000013))})
	testTupleDecode(t, `[1000,"UST","s2",1700000000003,null,5.5,6.5,null,"s8"]`,
		LedgerEntry{ID: 1000, Currency: currency.UST, Wallet: "s2", Time: types.Time(time.UnixMilli(1700000000003)), Amount: 5.5, Balance: 6.5, Description: "s8"})
}

func TestIsOrderType(t *testing.T) {
	t.Parallel()
	for _, orderType := range []string{OrderTypeLimit, OrderTypeMarket, OrderTypeStop, OrderTypeStopLimit, OrderTypeTrailingStop, OrderTypeFillOrKill, OrderTypeImmediateOrCancel} {
		assert.Truef(t, isOrderType(orderType), "isOrderType should accept %q", orderType)
		assert.Truef(t, isOrderType(ExchangeOrderTypePrefix+orderType), "isOrderType should accept %q", ExchangeOrderTypePrefix+orderType)
	}
	for _, orderType := range []string{"", "limit", "LIMITED", "EXCHANGE", ExchangeOrderTypePrefix, "EXCHANGELIMIT", ExchangeOrderTypePrefix + ExchangeOrderTypePrefix + OrderTypeLimit} {
		assert.Falsef(t, isOrderType(orderType), "isOrderType should refuse %q", orderType)
	}
}

func TestIsLedgerCategory(t *testing.T) {
	t.Parallel()
	for _, category := range []uint64{
		LedgerCategoryExchange, LedgerCategoryPositionChange, LedgerCategoryPositionClaim, LedgerCategoryPositionTransfer,
		LedgerCategoryPositionSwap, LedgerCategoryPositionFundingCost, LedgerCategoryMarginPayment, LedgerCategoryDerivativesFunding,
		LedgerCategorySettlement, LedgerCategoryTransfer, LedgerCategoryDeposit, LedgerCategoryWithdrawal,
		LedgerCategoryCancelledWithdrawal, LedgerCategoryTradingFee, LedgerCategoryTradingRebate, LedgerCategoryHiddenOrderFee,
		LedgerCategoryOTCTradeFee, LedgerCategorySwapFee, LedgerCategoryClaimingFee, LedgerCategoryUsedMarginFundingCharge,
		LedgerCategoryUnusedMarginFundingFee, LedgerCategoryEarnedFee, LedgerCategoryETHFXLoyaltyFee, LedgerCategoryDepositFee,
		LedgerCategoryWithdrawalFee, LedgerCategoryWithdrawalExpressFee, LedgerCategoryMinerFee, LedgerCategoryStakingPayment,
		LedgerCategoryAdjustment, LedgerCategoryExpense, LedgerCategoryCurrencyConversionFee, LedgerCategoryMonthlyProfitPayment,
		LedgerCategoryLosses,
	} {
		assert.Truef(t, isLedgerCategory(category), "isLedgerCategory should accept %d", category)
	}
	for _, category := range []uint64{0, 1, 6, 30, 999, math.MaxUint64} {
		assert.Falsef(t, isLedgerCategory(category), "isLedgerCategory should refuse %d", category)
	}
}

func TestOrderDecimalParam(t *testing.T) {
	t.Parallel()
	for v, want := range map[float64]string{0: "", 0.1: "0.1", -2.5: "-2.5", 1e-8: "0.00000001", 1e21: "1000000000000000000000", 25000: "25000"} {
		assert.Equalf(t, want, orderDecimalParam(v), "orderDecimalParam should format %v", v)
	}
}

func TestOrderTimeParam(t *testing.T) {
	t.Parallel()
	assert.Empty(t, orderTimeParam(time.Time{}, tifLayout), "orderTimeParam should leave the zero time unsent")
	local := time.Date(2020, time.January, 15, 21, 45, 23, 999, time.FixedZone("AEDT", 11*60*60))
	assert.Equal(t, "2020-01-15 10:45:23", orderTimeParam(local, tifLayout), "orderTimeParam should format a time in force in UTC")
	assert.Equal(t, "2020-01-15", orderTimeParam(local, cidDateLayout), "orderTimeParam should format a client order ID date in UTC")
}
