package binanceus

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// signedQuery returns a signed request's query without its timestamp and signature, which change every request, after
// checking they and the default receive window were sent
func signedQuery(t *testing.T, r *http.Request) url.Values {
	t.Helper()
	q := r.URL.Query()
	assert.NotEmpty(t, q.Get("timestamp"), "a signed request should send a timestamp")
	assert.NotEmpty(t, q.Get("signature"), "a signed request should send a signature")
	assert.Equal(t, "5000", q.Get("recvWindow"), "a signed request should send the default receive window")
	q.Del("timestamp")
	q.Del("signature")
	q.Del("recvWindow")
	return q
}

// signedRequestExchange returns an exchange whose signed requests must be method requests to path with exactly exp,
// answered with body
func signedRequestExchange(t *testing.T, method, path string, exp url.Values, body string) *Exchange {
	t.Helper()
	return newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, method, r.Method, "the request should use the documented method")
		assert.Equal(t, path, r.URL.Path, "the request should go to the documented path")
		assert.Equal(t, exp, signedQuery(t, r), "the request should send exactly the expected parameters")
		writeResponse(t, w, body)
	})
}

func TestGetCurrentOrderCountUsage(t *testing.T) {
	t.Parallel()
	skipIfCredentialsUnset(t)
	result, err := e.GetCurrentOrderCountUsage(t.Context())
	require.NoError(t, err, "GetCurrentOrderCountUsage must not error")
	if mockTests {
		exp := []RateLimitItemResponse{
			{RateLimitType: "ORDERS", Interval: "SECOND", IntervalNumber: 10, Limit: 100, Count: 2},
			{RateLimitType: "ORDERS", Interval: "DAY", IntervalNumber: 1, Limit: 200000, Count: 13},
		}
		assert.Equal(t, exp, result, "GetCurrentOrderCountUsage should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCurrentOrderCountUsage should return the order rate limits")
}

func TestNewOrderParams(t *testing.T) {
	t.Parallel()
	_, err := e.newOrderParams(nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "newOrderParams must reject a nil request")
	_, err = e.newOrderParams(&NewOrderRequest{Side: SideBuy, Type: OrderTypeMarket})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "newOrderParams must reject an empty pair")
	_, err = e.newOrderParams(&NewOrderRequest{Symbol: currency.NewBTCUSDT(), Type: OrderTypeMarket})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "newOrderParams must reject an empty side")
	_, err = e.newOrderParams(&NewOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "newOrderParams must reject an empty type")

	for _, tc := range []struct {
		name string
		req  *NewOrderRequest
		exp  url.Values
	}{
		{
			name: "every parameter",
			req: &NewOrderRequest{
				Symbol: currency.NewBTCUSDT(), Side: SideSell, Type: OrderTypeTakeProfitLimit, TimeInForce: TimeInForceFillOrKill,
				Quantity: 0.00001234, QuoteOrderQuantity: 25.5, Price: 80000.01, NewClientOrderID: "gct-1", StopPrice: 79999.99,
				TrailingDeltaInBasisPoints: 100, IcebergQuantity: 0.000005, NewOrderResponseType: NewOrderResponseTypeResult,
				SelfTradePreventionMode: SelfTradePreventionModeExpireBoth,
			},
			exp: url.Values{
				"symbol": {"BTCUSDT"}, "side": {"SELL"}, "type": {"TAKE_PROFIT_LIMIT"}, "timeInForce": {"FOK"},
				"quantity": {"0.00001234"}, "quoteOrderQty": {"25.5"}, "price": {"80000.01"}, "newClientOrderId": {"gct-1"},
				"stopPrice": {"79999.99"}, "trailingDelta": {"100"}, "icebergQty": {"0.000005"}, "newOrderRespType": {"RESULT"},
				"selfTradePreventionMode": {"EXPIRE_BOTH"},
			},
		},
		{
			name: "required parameters only",
			req:  &NewOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Type: OrderTypeMarket},
			exp:  url.Values{"symbol": {"BTCUSDT"}, "side": {"BUY"}, "type": {"MARKET"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := e.newOrderParams(tc.req)
			require.NoError(t, err, "newOrderParams must not error")
			assert.Equal(t, tc.exp, params, "newOrderParams should set exactly the request's parameters")
		})
	}

	for _, orderType := range []OrderType{OrderTypeLimit, OrderTypeStopLossLimit, OrderTypeTakeProfitLimit, OrderTypeLimitMaker} {
		t.Run(string(orderType)+" price", func(t *testing.T) {
			t.Parallel()
			params, err := e.newOrderParams(&NewOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Type: orderType, Price: 70000.5})
			require.NoError(t, err, "newOrderParams must not error")
			assert.Equal(t, "70000.5", params.Get("price"), "newOrderParams should send the price of every type that takes one")
		})
	}
}

func TestNewOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "NewOrder must reject a nil request")

	t.Run("posts the order's parameters", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodPost, "/api/v3/order", url.Values{"symbol": {"BTCUSDT"}, "side": {"BUY"}, "type": {"LIMIT"}, "timeInForce": {"GTC"}, "quantity": {"0.001"}, "price": {"70000"}}, `{"symbol":"BTCUSDT","orderId":1}`)
		result, err := ex.NewOrder(t.Context(), &NewOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Type: OrderTypeLimit, TimeInForce: TimeInForceGoodTillCancel, Quantity: 0.001, Price: 70000})
		require.NoError(t, err, "NewOrder must not error")
		assert.Equal(t, &NewOrderResponse{Symbol: "BTCUSDT", OrderID: 1}, result, "NewOrder should decode the response")
	})

	if !mockTests && !canManipulateRealOrders {
		t.Skip("canManipulateRealOrders is false; skipping a live order")
	}
	skipIfCredentialsUnset(t)
	result, err := e.NewOrder(t.Context(), &NewOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Type: OrderTypeMarket, QuoteOrderQuantity: 25, NewClientOrderID: "gct-market-1", NewOrderResponseType: NewOrderResponseTypeFull})
	require.NoError(t, err, "NewOrder must not error")
	if mockTests {
		exp := &NewOrderResponse{
			Symbol:                     "BTCUSDT",
			OrderID:                    28,
			OrderListID:                -1,
			ClientOrderID:              "gct-market-1",
			TransactTime:               types.Time(time.UnixMilli(1794464635319)),
			OriginalQuantity:           0.00033,
			ExecutedQuantity:           0.00033,
			OriginalQuoteOrderQuantity: 25,
			CumulativeQuoteQuantity:    24.3265199,
			Status:                     "FILLED",
			TimeInForce:                TimeInForceGoodTillCancel,
			Type:                       OrderTypeMarket,
			Side:                       SideBuy,
			WorkingTime:                1794464635319,
			SelfTradePreventionMode:    SelfTradePreventionModeExpireMaker,
			Fills: []OrderFill{
				{Price: 73716.55, Quantity: 0.0002, Commission: 0.0000002, CommissionAsset: currency.BTC, TradeID: 32183709},
				{Price: 73716.56, Quantity: 0.00013, Commission: 0.00000013, CommissionAsset: currency.BTC, TradeID: 32183710},
			},
		}
		assert.Equal(t, exp, result, "NewOrder should decode every field, the fills' trade IDs included")
	}
}

func TestNewOrderTest(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.NewOrderTest(t.Context(), nil), common.ErrNilPointer, "NewOrderTest must reject a nil request")
	skipIfCredentialsUnset(t)
	assert.NoError(t, e.NewOrderTest(t.Context(), &NewOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideSell, Type: OrderTypeLimit, TimeInForce: TimeInForceGoodTillCancel, Quantity: 0.001, Price: 80000}), "NewOrderTest should not error")
}

func TestQueryOrder(t *testing.T) {
	t.Parallel()
	_, err := e.QueryOrder(t.Context(), currency.EMPTYPAIR, "myOrder1", 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "QueryOrder must reject an empty pair")
	_, err = e.QueryOrder(t.Context(), currency.NewBTCUSDT(), "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "QueryOrder must reject a request without an ID")

	t.Run("sends either ID or both", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name     string
			clientID string
			orderID  uint64
			exp      url.Values
		}{
			{name: "order ID", orderID: 1, exp: url.Values{"symbol": {"BTCUSDT"}, "orderId": {"1"}}},
			{name: "client order ID", clientID: "myOrder1", exp: url.Values{"symbol": {"BTCUSDT"}, "origClientOrderId": {"myOrder1"}}},
			{name: "both", clientID: "myOrder1", orderID: 1, exp: url.Values{"symbol": {"BTCUSDT"}, "orderId": {"1"}, "origClientOrderId": {"myOrder1"}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/api/v3/order", tc.exp, `{}`)
				_, err := ex.QueryOrder(t.Context(), currency.NewBTCUSDT(), tc.clientID, tc.orderID)
				require.NoError(t, err, "QueryOrder must not error")
				_, err = ex.queryOrder(t.Context(), currency.NewBTCUSDT(), tc.clientID, tc.orderID, cancelLookupRate, false)
				require.NoError(t, err, "queryOrder must not error on the cancel pool")
			})
		}
	})

	skipIfCredentialsUnset(t)
	result, err := e.QueryOrder(t.Context(), currency.NewBTCUSDT(), "myOrder1", 1)
	if !mockTests {
		require.ErrorIs(t, err, order.ErrOrderNotFound, "QueryOrder must report an order the account does not have")
		return
	}
	require.NoError(t, err, "QueryOrder must not error")
	exp := &TradeOrderResponse{
		Symbol:                     "BTCUSDT",
		OrderID:                    1,
		OrderListID:                -1,
		ClientOrderID:              "myOrder1",
		Price:                      69900,
		OriginalQuantity:           0.001,
		ExecutedQuantity:           0.0004,
		CumulativeQuoteQuantity:    27.96,
		Status:                     "PARTIALLY_FILLED",
		TimeInForce:                TimeInForceGoodTillCancel,
		Type:                       OrderTypeStopLossLimit,
		Side:                       SideSell,
		Time:                       types.Time(time.UnixMilli(1794464000000)),
		UpdateTime:                 types.Time(time.UnixMilli(1794464635319)),
		IsWorking:                  true,
		WorkingTime:                1794464600000,
		SelfTradePreventionMode:    SelfTradePreventionModeExpireMaker,
		IcebergQuantity:            0.0002,
		PreventedMatchID:           3,
		PreventedQuantity:          0.0001,
		StopPrice:                  70000,
		TrailingDeltaInBasisPoints: 100,
		TrailingTime:               1794464500000,
	}
	assert.Equal(t, exp, result, "QueryOrder should decode every field")
}

func TestOpenOrders(t *testing.T) {
	t.Parallel()
	t.Run("lists every symbol's orders without a symbol", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodGet, "/api/v3/openOrders", url.Values{}, `[]`)
		_, err := ex.OpenOrders(t.Context(), currency.EMPTYPAIR)
		require.NoError(t, err, "OpenOrders must not error")
		_, err = ex.openOrders(t.Context(), currency.EMPTYPAIR, cancelAllListingRate, false)
		require.NoError(t, err, "openOrders must not error on the cancel pool")
	})

	skipIfCredentialsUnset(t)
	result, err := e.OpenOrders(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "OpenOrders must not error")
	if mockTests {
		exp := []TradeOrderResponse{{
			Symbol:                  "BTCUSDT",
			OrderID:                 2,
			OrderListID:             -1,
			ClientOrderID:           "myOrder2",
			Price:                   70000,
			OriginalQuantity:        0.001,
			Status:                  "NEW",
			TimeInForce:             TimeInForceGoodTillCancel,
			Type:                    OrderTypeLimit,
			Side:                    SideBuy,
			Time:                    types.Time(time.UnixMilli(1794464000000)),
			UpdateTime:              types.Time(time.UnixMilli(1794464000000)),
			IsWorking:               true,
			WorkingTime:             1794464000000,
			SelfTradePreventionMode: SelfTradePreventionModeExpireMaker,
		}}
		assert.Equal(t, exp, result, "OpenOrders should decode every field")
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelExistingOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelExistingOrder must reject a nil request")
	_, err = e.CancelExistingOrder(t.Context(), &CancelOrderRequest{OrderID: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelExistingOrder must reject an empty pair")
	_, err = e.CancelExistingOrder(t.Context(), &CancelOrderRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelExistingOrder must reject a request without an ID")

	t.Run("sends both IDs when both are set", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodDelete, "/api/v3/order", url.Values{"symbol": {"BTCUSDT"}, "orderId": {"4"}, "origClientOrderId": {"myOrder1"}}, `{}`)
		_, err := ex.CancelExistingOrder(t.Context(), &CancelOrderRequest{Symbol: currency.NewBTCUSDT(), OrderID: 4, OriginalClientOrderID: "myOrder1"})
		require.NoError(t, err, "CancelExistingOrder must not error")
	})

	if !mockTests && !canManipulateRealOrders {
		t.Skip("canManipulateRealOrders is false; skipping a live cancellation")
	}
	skipIfCredentialsUnset(t)
	result, err := e.CancelExistingOrder(t.Context(), &CancelOrderRequest{Symbol: currency.NewBTCUSDT(), OrderID: 4, OriginalClientOrderID: "myOrder1", NewClientOrderID: "cancelMyOrder1", CancelRestrictions: CancelRestrictionsOnlyNew})
	require.NoError(t, err, "CancelExistingOrder must not error")
	if mockTests {
		exp := &CancelOrderResponse{
			Symbol:                  "BTCUSDT",
			OriginalClientOrderID:   "myOrder1",
			OrderID:                 4,
			OrderListID:             -1,
			ClientOrderID:           "cancelMyOrder1",
			TransactTime:            types.Time(time.UnixMilli(1794464635319)),
			Price:                   70000,
			OriginalQuantity:        0.001,
			Status:                  "CANCELED",
			TimeInForce:             TimeInForceGoodTillCancel,
			Type:                    OrderTypeLimit,
			Side:                    SideBuy,
			SelfTradePreventionMode: SelfTradePreventionModeExpireMaker,
		}
		assert.Equal(t, exp, result, "CancelExistingOrder should decode every field")
	}
}

func TestCancelAllOpenOrderOnSymbol(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOpenOrderOnSymbol(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelAllOpenOrderOnSymbol must reject an empty pair")

	if !mockTests && !canManipulateRealOrders {
		t.Skip("canManipulateRealOrders is false; skipping a live cancellation")
	}
	skipIfCredentialsUnset(t)
	result, err := e.CancelAllOpenOrderOnSymbol(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "CancelAllOpenOrderOnSymbol must not error")
	if !mockTests {
		return
	}
	cancelled := func(clientOrderID, origClientOrderID string, orderID uint64, orderListID int64, orderType OrderType, price, stopPrice, quantity float64, side Side, transactTime int64) CancelOrderResponse {
		return CancelOrderResponse{
			Symbol: "BTCUSDT", OriginalClientOrderID: origClientOrderID, OrderID: orderID, OrderListID: orderListID, ClientOrderID: clientOrderID,
			TransactTime: types.Time(time.UnixMilli(transactTime)), Price: types.Number(price), OriginalQuantity: types.Number(quantity),
			Status: "CANCELED", TimeInForce: TimeInForceGoodTillCancel, Type: orderType, Side: side,
			SelfTradePreventionMode: SelfTradePreventionModeExpireMaker, StopPrice: types.Number(stopPrice),
		}
	}
	listOrderReport := func(clientOrderID, origClientOrderID string, orderID uint64, orderType OrderType, price, stopPrice float64) OrderResponse {
		return OrderResponse{
			Symbol: "BTCUSDT", OriginalClientOrderID: origClientOrderID, OrderID: orderID, OrderListID: 1929, ClientOrderID: clientOrderID,
			TransactTime: types.Time(time.UnixMilli(1794464635320)), Price: types.Number(price), OriginalQuantity: 0.002, Status: "CANCELED",
			TimeInForce: TimeInForceGoodTillCancel, Type: orderType, Side: SideSell, SelfTradePreventionMode: SelfTradePreventionModeExpireMaker,
			StopPrice: types.Number(stopPrice),
		}
	}
	exp := []CancelledOpenOrderResponse{
		{CancelOrderResponse: cancelled("8epSIUMvNCknntXcSzWs7H", "KZJijIsAFf5BU5oxkWTAU3", 5, -1, OrderTypeLimit, 70000, 0, 0.001, SideBuy, 1794464635319)},
		{
			Symbol: "BTCUSDT", OrderListID: 1929,
			ContingencyType:   "OCO",
			ListStatusType:    "ALL_DONE",
			ListOrderStatus:   "ALL_DONE",
			ListClientOrderID: "2inzWQdDvZLHbbAmAozX2N",
			TransactionTime:   types.Time(time.UnixMilli(1794464635320)),
			Orders:            []OrderListOrder{{Symbol: "BTCUSDT", OrderID: 20, ClientOrderID: "CwOOIPHSmYywx6jZX77TdL"}, {Symbol: "BTCUSDT", OrderID: 21, ClientOrderID: "461cPg51vQjV3zIMOXNz39"}},
			OrderReports: []OrderResponse{
				listOrderReport("pXLV6Hz6mprAcVYpVMTGgx", "CwOOIPHSmYywx6jZX77TdL", 20, OrderTypeStopLossLimit, 69900, 70000),
				listOrderReport("pXLV6Hz6mprAcVYpVMTGgx", "461cPg51vQjV3zIMOXNz39", 21, OrderTypeLimitMaker, 80000, 0),
			},
		},
	}
	assert.Equal(t, exp, result, "CancelAllOpenOrderOnSymbol should decode cancelled orders and order lists")
}

func TestGetAccountTradeList(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountTradeList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAccountTradeList must reject a nil request")
	_, err = e.GetAccountTradeList(t.Context(), &AccountTradeListRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetAccountTradeList must reject an empty pair")
	start := time.UnixMilli(1794464000000)
	for _, tc := range []struct {
		name string
		req  *AccountTradeListRequest
	}{
		{name: "order ID with a start time", req: &AccountTradeListRequest{Symbol: currency.NewBTCUSDT(), OrderID: 1, StartTime: start}},
		{name: "from ID with an end time", req: &AccountTradeListRequest{Symbol: currency.NewBTCUSDT(), FromID: 1, EndTime: start}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetAccountTradeList(t.Context(), tc.req)
			require.ErrorIs(t, err, errInvalidParameterCombination, "GetAccountTradeList must reject orderId or fromId with a time")
		})
	}
	_, err = e.GetAccountTradeList(t.Context(), &AccountTradeListRequest{Symbol: currency.NewBTCUSDT(), StartTime: start, EndTime: start.Add(-time.Second)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAccountTradeList must reject a start after the end")

	t.Run("sends the set parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *AccountTradeListRequest
			exp  url.Values
		}{
			{name: "order and from ID", req: &AccountTradeListRequest{Symbol: currency.NewBTCUSDT(), OrderID: 100234, FromID: 28457, Limit: 1000}, exp: url.Values{"symbol": {"BTCUSDT"}, "orderId": {"100234"}, "fromId": {"28457"}, "limit": {"1000"}}},
			{name: "time range", req: &AccountTradeListRequest{Symbol: currency.NewBTCUSDT(), StartTime: start, EndTime: start.Add(time.Hour)}, exp: url.Values{"symbol": {"BTCUSDT"}, "startTime": {"1794464000000"}, "endTime": {"1794467600000"}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/api/v3/myTrades", tc.exp, `[]`)
				_, err := ex.GetAccountTradeList(t.Context(), tc.req)
				require.NoError(t, err, "GetAccountTradeList must not error")
			})
		}
	})

	skipIfCredentialsUnset(t)
	result, err := e.GetAccountTradeList(t.Context(), &AccountTradeListRequest{Symbol: currency.NewBTCUSDT(), OrderID: 100234})
	require.NoError(t, err, "GetAccountTradeList must not error")
	if mockTests {
		exp := []AccountTradeResponse{{
			Symbol:          "BTCUSDT",
			ID:              28457,
			OrderID:         100234,
			OrderListID:     -1,
			Price:           73716.55,
			Quantity:        0.0002,
			QuoteQuantity:   14.74331,
			Commission:      0.0000002,
			CommissionAsset: currency.BTC,
			Time:            types.Time(time.UnixMilli(1794464635319)),
			IsBuyer:         true,
			IsMaker:         true,
			IsBestMatch:     true,
		}}
		assert.Equal(t, exp, result, "GetAccountTradeList should decode every field")
	}
}

func TestCancelExistingOrderAndSendNewOrder(t *testing.T) {
	t.Parallel()
	valid := CancelReplaceOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Type: OrderTypeLimit, CancelReplaceMode: CancelReplaceModeStopOnFailure, CancelOrderID: 9}
	for _, tc := range []struct {
		name   string
		modify func(*CancelReplaceOrderRequest)
		err    error
	}{
		{name: "empty pair", modify: func(r *CancelReplaceOrderRequest) { r.Symbol = currency.EMPTYPAIR }, err: currency.ErrCurrencyPairEmpty},
		{name: "no side", modify: func(r *CancelReplaceOrderRequest) { r.Side = "" }, err: order.ErrSideIsInvalid},
		{name: "no type", modify: func(r *CancelReplaceOrderRequest) { r.Type = "" }, err: order.ErrTypeIsInvalid},
		{name: "no mode", modify: func(r *CancelReplaceOrderRequest) { r.CancelReplaceMode = "" }, err: errCancelReplaceModeRequired},
		{name: "no order to cancel", modify: func(r *CancelReplaceOrderRequest) { r.CancelOrderID = 0 }, err: order.ErrOrderIDNotSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := valid
			tc.modify(&req)
			_, err := e.CancelExistingOrderAndSendNewOrder(t.Context(), &req)
			require.ErrorIsf(t, err, tc.err, "CancelExistingOrderAndSendNewOrder must reject a request with %s", tc.name)
		})
	}
	_, err := e.CancelExistingOrderAndSendNewOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelExistingOrderAndSendNewOrder must reject a nil request")

	t.Run("sends every parameter", func(t *testing.T) {
		t.Parallel()
		exp := url.Values{
			"symbol": {"BTCUSDT"}, "side": {"SELL"}, "type": {"STOP_LOSS_LIMIT"}, "cancelReplaceMode": {"ALLOW_FAILURE"},
			"timeInForce": {"IOC"}, "quantity": {"0.00001234"}, "quoteOrderQty": {"12.5"}, "price": {"69000.01"},
			"cancelNewClientOrderId": {"cancel-1"}, "cancelOrigClientOrderId": {"orig-1"}, "cancelOrderId": {"9"},
			"newClientOrderId": {"new-1"}, "strategyId": {"7"}, "strategyType": {"1000000"}, "stopPrice": {"69500.5"},
			"trailingDelta": {"50"}, "icebergQty": {"0.000005"}, "newOrderRespType": {"ACK"},
			"selfTradePreventionMode": {"EXPIRE_TAKER"}, "cancelRestrictions": {"ONLY_PARTIALLY_FILLED"},
		}
		ex := signedRequestExchange(t, http.MethodPost, "/api/v3/order/cancelReplace", exp, `{"cancelResult":"SUCCESS","newOrderResult":"SUCCESS"}`)
		result, err := ex.CancelExistingOrderAndSendNewOrder(t.Context(), &CancelReplaceOrderRequest{
			Symbol: currency.NewBTCUSDT(), Side: SideSell, Type: OrderTypeStopLossLimit, CancelReplaceMode: CancelReplaceModeAllowFailure,
			TimeInForce: TimeInForceImmediateOrCancel, Quantity: 0.00001234, QuoteOrderQuantity: 12.5, Price: 69000.01,
			CancelNewClientOrderID: "cancel-1", CancelOriginalClientOrderID: "orig-1", CancelOrderID: 9, NewClientOrderID: "new-1",
			StrategyID: 7, StrategyType: 1000000, StopPrice: 69500.5, TrailingDeltaInBasisPoints: 50, IcebergQuantity: 0.000005,
			NewOrderResponseType: NewOrderResponseTypeAcknowledgement, SelfTradePreventionMode: SelfTradePreventionModeExpireTaker,
			CancelRestrictions: CancelRestrictionsOnlyPartiallyFilled,
		})
		require.NoError(t, err, "CancelExistingOrderAndSendNewOrder must not error")
		assert.Equal(t, &CancelAndReplaceResponse{CancelResult: "SUCCESS", NewOrderResult: "SUCCESS"}, result, "CancelExistingOrderAndSendNewOrder should decode the outcome")
	})

	t.Run("returns both halves' outcome with a partial failure", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusConflict)
			writeResponse(t, w, `{"code":-2021,"msg":"Order cancel-replace partially failed.","data":{"cancelResult":"FAILURE","newOrderResult":"SUCCESS","cancelResponse":{"code":-2011,"msg":"Unknown order sent."},"newOrderResponse":{"symbol":"BTCUSDT","orderId":11,"orderListId":-1,"clientOrderId":"new-2","status":"NEW"}}}`)
		})
		result, err := ex.CancelExistingOrderAndSendNewOrder(t.Context(), &valid)
		require.ErrorIs(t, err, errAPIResponse, "CancelExistingOrderAndSendNewOrder must return the API error")
		require.NotNil(t, result, "CancelExistingOrderAndSendNewOrder must return the outcome with the error")
		exp := &CancelAndReplaceResponse{
			CancelResult:     "FAILURE",
			NewOrderResult:   "SUCCESS",
			CancelResponse:   CancelReplaceCancelResponse{Code: -2011, Message: "Unknown order sent."},
			NewOrderResponse: CancelReplaceNewOrderResponse{NewOrderResponse: NewOrderResponse{Symbol: "BTCUSDT", OrderID: 11, OrderListID: -1, ClientOrderID: "new-2", Status: "NEW"}},
		}
		assert.Equal(t, exp, result, "CancelExistingOrderAndSendNewOrder should decode both halves from the error data")
	})

	if !mockTests && !canManipulateRealOrders {
		t.Skip("canManipulateRealOrders is false; skipping a live cancel-replace")
	}
	skipIfCredentialsUnset(t)
	result, err := e.CancelExistingOrderAndSendNewOrder(t.Context(), &CancelReplaceOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Type: OrderTypeLimit, CancelReplaceMode: CancelReplaceModeStopOnFailure, TimeInForce: TimeInForceGoodTillCancel, Quantity: 0.04, Price: 70000, CancelOrderID: 9})
	require.NoError(t, err, "CancelExistingOrderAndSendNewOrder must not error")
	if mockTests {
		exp := &CancelAndReplaceResponse{
			CancelResult:   "SUCCESS",
			NewOrderResult: "SUCCESS",
			CancelResponse: CancelReplaceCancelResponse{CancelOrderResponse: CancelOrderResponse{
				Symbol: "BTCUSDT", OriginalClientOrderID: "DnLo3vTAQcjha43lAZhZ0y", OrderID: 9, OrderListID: -1, ClientOrderID: "osxN3JXAtJvKvCqGeMWMVR",
				TransactTime: types.Time(time.UnixMilli(1794464635319)), Price: 69000, OriginalQuantity: 0.04, Status: "CANCELED",
				TimeInForce: TimeInForceGoodTillCancel, Type: OrderTypeLimit, Side: SideBuy, SelfTradePreventionMode: SelfTradePreventionModeExpireMaker,
			}},
			NewOrderResponse: CancelReplaceNewOrderResponse{NewOrderResponse: NewOrderResponse{
				Symbol: "BTCUSDT", OrderID: 10, OrderListID: -1, ClientOrderID: "wOceeeOzNORyLiQfw7jd8S", TransactTime: types.Time(time.UnixMilli(1794464635319)),
				Price: 70000, OriginalQuantity: 0.04, Status: "NEW", TimeInForce: TimeInForceGoodTillCancel, Type: OrderTypeLimit, Side: SideBuy,
				WorkingTime: 1794464635319, SelfTradePreventionMode: SelfTradePreventionModeExpireMaker, Fills: []OrderFill{},
			}},
		}
		assert.Equal(t, exp, result, "CancelExistingOrderAndSendNewOrder should decode every field")
	}
}

func TestCancelReplaceOutcome(t *testing.T) {
	t.Parallel()
	resp := &CancelAndReplaceResponse{CancelResult: "SUCCESS"}
	result, err := cancelReplaceOutcome(resp, nil)
	require.NoError(t, err, "cancelReplaceOutcome must not error without an error")
	assert.Same(t, resp, result, "cancelReplaceOutcome should return the response without an error")

	otherErr := errInvalidParameterCombination
	result, err = cancelReplaceOutcome(resp, otherErr)
	assert.ErrorIs(t, err, otherErr, "cancelReplaceOutcome should return another error as it is")
	assert.Same(t, resp, result, "cancelReplaceOutcome should return the response with another error")

	noData := &APIError{Code: -1013, Message: "Filter failure"}
	result, err = cancelReplaceOutcome[CancelAndReplaceResponse](nil, noData)
	assert.ErrorIs(t, err, errAPIResponse, "cancelReplaceOutcome should return an API error without data as it is")
	assert.Nil(t, result, "cancelReplaceOutcome should return no outcome for an API error without data")

	badData := &APIError{Code: -2021, Data: json.RawMessage(`"x"`)}
	result, err = cancelReplaceOutcome[CancelAndReplaceResponse](nil, badData)
	assert.ErrorIs(t, err, errAPIResponse, "cancelReplaceOutcome should keep the API error when its data cannot be decoded")
	assert.ErrorContains(t, err, "error decoding cancel-replace outcome", "cancelReplaceOutcome should report the decoding failure")
	assert.Nil(t, result, "cancelReplaceOutcome should return no outcome for undecodable data")
}

func TestGetPreventedMatches(t *testing.T) {
	t.Parallel()
	_, err := e.GetPreventedMatches(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPreventedMatches must reject a nil request")
	_, err = e.GetPreventedMatches(t.Context(), &PreventedMatchesRequest{OrderID: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPreventedMatches must reject an empty pair")
	_, err = e.GetPreventedMatches(t.Context(), &PreventedMatchesRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetPreventedMatches must reject a request without an ID")

	t.Run("sends the set parameters", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodGet, "/api/v3/myPreventedMatches", url.Values{"symbol": {"BTCUSDT"}, "orderId": {"5"}, "fromPreventedMatchId": {"2"}, "limit": {"1000"}}, `[]`)
		_, err := ex.GetPreventedMatches(t.Context(), &PreventedMatchesRequest{Symbol: currency.NewBTCUSDT(), OrderID: 5, FromPreventedMatchID: 2, Limit: 1000})
		require.NoError(t, err, "GetPreventedMatches must not error")
	})

	t.Run("decodes a prevented match of an account in no trade group", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodGet, "/api/v3/myPreventedMatches", url.Values{"symbol": {"BTCUSDT"}, "preventedMatchId": {"8"}}, `[{"symbol":"BTCUSDT","preventedMatchId":8,"takerOrderId":12,"makerOrderId":10,"tradeGroupId":-1,"selfTradePreventionMode":"EXPIRE_BOTH","price":"50.00000000","takerPreventedQuantity":"1.00000000","makerPreventedQuantity":"10.00000000","transactTime":1663190634060}]`)
		result, err := ex.GetPreventedMatches(t.Context(), &PreventedMatchesRequest{Symbol: currency.NewBTCUSDT(), PreventedMatchID: 8})
		require.NoError(t, err, "GetPreventedMatches must not error for an account in no trade group")
		require.Len(t, result, 1, "GetPreventedMatches must return the prevented match")
		assert.Equal(t, int64(-1), result[0].TradeGroupID, "GetPreventedMatches should decode the trade group ID of an account in no trade group")
	})

	t.Run("decodes a prevented match of an account in a trade group", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodGet, "/api/v3/myPreventedMatches", url.Values{"symbol": {"BTCUSDT"}, "preventedMatchId": {"9"}}, `[{"symbol":"BTCUSDT","preventedMatchId":9,"takerOrderId":14,"makerOrderId":13,"tradeGroupId":2,"selfTradePreventionMode":"EXPIRE_MAKER","price":"50.00000000","makerPreventedQuantity":"10.00000000","transactTime":1663190634061}]`)
		result, err := ex.GetPreventedMatches(t.Context(), &PreventedMatchesRequest{Symbol: currency.NewBTCUSDT(), PreventedMatchID: 9})
		require.NoError(t, err, "GetPreventedMatches must not error for an account in a trade group")
		require.Len(t, result, 1, "GetPreventedMatches must return the prevented match")
		assert.Equal(t, int64(2), result[0].TradeGroupID, "GetPreventedMatches should decode the trade group ID of an account in a trade group")
	})

	skipIfCredentialsUnset(t)
	result, err := e.GetPreventedMatches(t.Context(), &PreventedMatchesRequest{Symbol: currency.NewBTCUSDT(), PreventedMatchID: 1})
	require.NoError(t, err, "GetPreventedMatches must not error")
	if mockTests {
		exp := []PreventedMatchResponse{{
			Symbol:                  "BTCUSDT",
			PreventedMatchID:        1,
			TakerOrderID:            5,
			MakerOrderID:            3,
			TradeGroupID:            1,
			SelfTradePreventionMode: SelfTradePreventionModeExpireMaker,
			Price:                   73716.55,
			MakerPreventedQuantity:  0.0013,
			TransactTime:            types.Time(time.UnixMilli(1794464635319)),
		}}
		assert.Equal(t, exp, result, "GetPreventedMatches should decode every field")
	}
}

func TestAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.AllOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AllOrders must reject a nil request")
	_, err = e.AllOrders(t.Context(), &AllOrdersRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "AllOrders must reject an empty pair")
	start := time.UnixMilli(1794464000000)
	_, err = e.AllOrders(t.Context(), &AllOrdersRequest{Symbol: currency.NewBTCUSDT(), StartTime: start, EndTime: start.Add(-time.Second)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "AllOrders must reject a start after the end")

	t.Run("sends the set parameters", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodGet, "/api/v3/allOrders", url.Values{"symbol": {"BTCUSDT"}, "orderId": {"28"}, "startTime": {"1794464000000"}, "endTime": {"1794467600000"}, "limit": {"1000"}}, `[]`)
		_, err := ex.AllOrders(t.Context(), &AllOrdersRequest{Symbol: currency.NewBTCUSDT(), OrderID: 28, StartTime: start, EndTime: start.Add(time.Hour), Limit: 1000})
		require.NoError(t, err, "AllOrders must not error")
	})

	skipIfCredentialsUnset(t)
	result, err := e.AllOrders(t.Context(), &AllOrdersRequest{Symbol: currency.NewBTCUSDT()})
	require.NoError(t, err, "AllOrders must not error")
	if mockTests {
		exp := []TradeOrderResponse{{
			Symbol:                     "BTCUSDT",
			OrderID:                    28,
			OrderListID:                -1,
			ClientOrderID:              "gct-market-1",
			OriginalQuantity:           0.00033,
			ExecutedQuantity:           0.00033,
			CumulativeQuoteQuantity:    24.3265199,
			Status:                     "FILLED",
			TimeInForce:                TimeInForceGoodTillCancel,
			Type:                       OrderTypeMarket,
			Side:                       SideBuy,
			Time:                       types.Time(time.UnixMilli(1794464635319)),
			UpdateTime:                 types.Time(time.UnixMilli(1794464635319)),
			IsWorking:                  true,
			OriginalQuoteOrderQuantity: 25,
			WorkingTime:                1794464635319,
			SelfTradePreventionMode:    SelfTradePreventionModeExpireMaker,
		}}
		assert.Equal(t, exp, result, "AllOrders should decode every field")
	}
}

func TestNewOneCancelsTheOtherOrder(t *testing.T) {
	t.Parallel()
	valid := OneCancelsTheOtherOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideSell, Quantity: 0.002, Price: 80000, StopPrice: 70000}
	for _, tc := range []struct {
		name   string
		modify func(*OneCancelsTheOtherOrderRequest)
		err    error
	}{
		{name: "an empty pair", modify: func(r *OneCancelsTheOtherOrderRequest) { r.Symbol = currency.EMPTYPAIR }, err: currency.ErrCurrencyPairEmpty},
		{name: "no side", modify: func(r *OneCancelsTheOtherOrderRequest) { r.Side = "" }, err: order.ErrSideIsInvalid},
		{name: "no quantity", modify: func(r *OneCancelsTheOtherOrderRequest) { r.Quantity = 0 }, err: order.ErrAmountIsInvalid},
		{name: "no price", modify: func(r *OneCancelsTheOtherOrderRequest) { r.Price = 0 }, err: order.ErrPriceMustBeSetIfLimitOrder},
		{name: "no stop price", modify: func(r *OneCancelsTheOtherOrderRequest) { r.StopPrice = 0 }, err: errStopPriceRequired},
		{name: "a stop limit price without its time in force", modify: func(r *OneCancelsTheOtherOrderRequest) { r.StopLimitPrice = 69900 }, err: errStopLimitTimeInForceRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := valid
			tc.modify(&req)
			_, err := e.NewOneCancelsTheOtherOrder(t.Context(), &req)
			require.ErrorIsf(t, err, tc.err, "NewOneCancelsTheOtherOrder must reject %s", tc.name)
		})
	}
	_, err := e.NewOneCancelsTheOtherOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "NewOneCancelsTheOtherOrder must reject a nil request")

	t.Run("sends every parameter at full precision", func(t *testing.T) {
		t.Parallel()
		exp := url.Values{
			"symbol": {"BTCUSDT"}, "side": {"BUY"}, "quantity": {"0.00001234"}, "price": {"60000.01"}, "stopPrice": {"80000.12"},
			"listClientOrderId": {"list-1"}, "limitClientOrderId": {"limit-1"}, "limitIcebergQty": {"0.000006"}, "trailingDelta": {"25"},
			"stopClientOrderId": {"stop-1"}, "stopLimitPrice": {"80100.5"}, "stopIcebergQty": {"0.000004"},
			"stopLimitTimeInForce": {"FOK"}, "newOrderRespType": {"FULL"}, "selfTradePreventionMode": {"EXPIRE_MAKER"},
		}
		ex := signedRequestExchange(t, http.MethodPost, "/api/v3/order/oco", exp, `{}`)
		_, err := ex.NewOneCancelsTheOtherOrder(t.Context(), &OneCancelsTheOtherOrderRequest{
			Symbol: currency.NewBTCUSDT(), ListClientOrderID: "list-1", Side: SideBuy, Quantity: 0.00001234, LimitClientOrderID: "limit-1",
			Price: 60000.01, LimitIcebergQuantity: 0.000006, TrailingDeltaInBasisPoints: 25, StopClientOrderID: "stop-1", StopPrice: 80000.12,
			StopLimitPrice: 80100.5, StopIcebergQuantity: 0.000004, StopLimitTimeInForce: TimeInForceFillOrKill,
			NewOrderResponseType: NewOrderResponseTypeFull, SelfTradePreventionMode: SelfTradePreventionModeExpireMaker,
		})
		require.NoError(t, err, "NewOneCancelsTheOtherOrder must not error")
	})

	if !mockTests && !canManipulateRealOrders {
		t.Skip("canManipulateRealOrders is false; skipping a live order list")
	}
	skipIfCredentialsUnset(t)
	result, err := e.NewOneCancelsTheOtherOrder(t.Context(), &OneCancelsTheOtherOrderRequest{Symbol: currency.NewBTCUSDT(), ListClientOrderID: "gct-oco-1", Side: SideSell, Quantity: 0.002, Price: 80000, StopPrice: 70000, StopLimitPrice: 69900, StopLimitTimeInForce: TimeInForceGoodTillCancel})
	require.NoError(t, err, "NewOneCancelsTheOtherOrder must not error")
	if mockTests {
		report := func(orderID uint64, clientOrderID string, orderType OrderType, price, stopPrice float64, workingTime int64) OrderResponse {
			return OrderResponse{
				Symbol: "BTCUSDT", OrderID: orderID, OrderListID: 1929, ClientOrderID: clientOrderID, TransactTime: types.Time(time.UnixMilli(1794464635319)),
				Price: types.Number(price), OriginalQuantity: 0.002, Status: "NEW", TimeInForce: TimeInForceGoodTillCancel, Type: orderType, Side: SideSell,
				WorkingTime: workingTime, SelfTradePreventionMode: SelfTradePreventionModeExpireMaker, StopPrice: types.Number(stopPrice),
			}
		}
		exp := &OneCancelsTheOtherOrderResponse{
			OrderListID:       1929,
			ContingencyType:   "OCO",
			ListStatusType:    "EXEC_STARTED",
			ListOrderStatus:   "EXECUTING",
			ListClientOrderID: "gct-oco-1",
			TransactionTime:   types.Time(time.UnixMilli(1794464635319)),
			Symbol:            "BTCUSDT",
			Orders:            []OrderListOrder{{Symbol: "BTCUSDT", OrderID: 20, ClientOrderID: "CwOOIPHSmYywx6jZX77TdL"}, {Symbol: "BTCUSDT", OrderID: 21, ClientOrderID: "461cPg51vQjV3zIMOXNz39"}},
			OrderReports: []OrderResponse{
				report(20, "CwOOIPHSmYywx6jZX77TdL", OrderTypeStopLossLimit, 69900, 70000, -1),
				report(21, "461cPg51vQjV3zIMOXNz39", OrderTypeLimitMaker, 80000, 0, 1794464635319),
			},
		}
		assert.Equal(t, exp, result, "NewOneCancelsTheOtherOrder should decode every field")
	}
}

func TestGetOneCancelsTheOtherOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetOneCancelsTheOtherOrders(t.Context(), &QueryOrderListRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOneCancelsTheOtherOrders must reject a request without an ID")
	_, err = e.GetOneCancelsTheOtherOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOneCancelsTheOtherOrders must reject a nil request")

	t.Run("sends an order list ID of 0", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodGet, "/api/v3/orderList", url.Values{"orderListId": {"0"}}, `{}`)
		_, err := ex.GetOneCancelsTheOtherOrders(t.Context(), &QueryOrderListRequest{OrderListID: new(uint64(0))})
		require.NoError(t, err, "GetOneCancelsTheOtherOrders must not error")
	})

	t.Run("sends the list client order ID as origClientOrderId", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodGet, "/api/v3/orderList", url.Values{"origClientOrderId": {"h2USkA5YQpaXHPIrkd96xE"}}, `{}`)
		_, err := ex.GetOneCancelsTheOtherOrders(t.Context(), &QueryOrderListRequest{OriginalClientOrderID: "h2USkA5YQpaXHPIrkd96xE"})
		require.NoError(t, err, "GetOneCancelsTheOtherOrders must not error")
		_, err = ex.getOneCancelsTheOtherOrders(t.Context(), &QueryOrderListRequest{OriginalClientOrderID: "h2USkA5YQpaXHPIrkd96xE"}, cancelLookupRate, false)
		require.NoError(t, err, "getOneCancelsTheOtherOrders must not error on the cancel pool")
	})

	skipIfCredentialsUnset(t)
	result, err := e.GetOneCancelsTheOtherOrders(t.Context(), &QueryOrderListRequest{OrderListID: new(uint64(27))})
	if !mockTests {
		require.ErrorIs(t, err, errAPIResponse, "GetOneCancelsTheOtherOrders must report an order list the account does not have")
		return
	}
	require.NoError(t, err, "GetOneCancelsTheOtherOrders must not error")
	exp := &OneCancelsTheOtherOrderResponse{
		OrderListID:       27,
		ContingencyType:   "OCO",
		ListStatusType:    "EXEC_STARTED",
		ListOrderStatus:   "EXECUTING",
		ListClientOrderID: "h2USkA5YQpaXHPIrkd96xE",
		TransactionTime:   types.Time(time.UnixMilli(1794464635319)),
		Symbol:            "BTCUSDT",
		Orders:            []OrderListOrder{{Symbol: "BTCUSDT", OrderID: 4, ClientOrderID: "qD1gy3kc3Gx0rihm9Y3xwS"}, {Symbol: "BTCUSDT", OrderID: 5, ClientOrderID: "ARzZ9I00CPM8i3NhmU9Ega"}},
	}
	assert.Equal(t, exp, result, "GetOneCancelsTheOtherOrders should decode every field")
}

func TestGetAllOneCancelsTheOtherOrders(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1794464000000)
	_, err := e.GetAllOneCancelsTheOtherOrders(t.Context(), &AllOrderListsRequest{FromID: 1, StartTime: start})
	require.ErrorIs(t, err, errInvalidParameterCombination, "GetAllOneCancelsTheOtherOrders must reject fromId with a time")
	_, err = e.GetAllOneCancelsTheOtherOrders(t.Context(), &AllOrderListsRequest{StartTime: start, EndTime: start.Add(-time.Second)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAllOneCancelsTheOtherOrders must reject a start after the end")

	t.Run("sends the set parameters", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			req  *AllOrderListsRequest
			exp  url.Values
		}{
			{name: "end time alone", req: &AllOrderListsRequest{EndTime: start}, exp: url.Values{"endTime": {"1794464000000"}}},
			{name: "from ID and limit", req: &AllOrderListsRequest{FromID: 28, Limit: 1000}, exp: url.Values{"fromId": {"28"}, "limit": {"1000"}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := signedRequestExchange(t, http.MethodGet, "/api/v3/allOrderList", tc.exp, `[]`)
				_, err := ex.GetAllOneCancelsTheOtherOrders(t.Context(), tc.req)
				require.NoError(t, err, "GetAllOneCancelsTheOtherOrders must not error")
			})
		}
	})

	skipIfCredentialsUnset(t)
	result, err := e.GetAllOneCancelsTheOtherOrders(t.Context(), nil)
	require.NoError(t, err, "GetAllOneCancelsTheOtherOrders must not error")
	if mockTests {
		exp := []OneCancelsTheOtherOrderResponse{{
			OrderListID:       29,
			ContingencyType:   "OCO",
			ListStatusType:    "EXEC_STARTED",
			ListOrderStatus:   "EXECUTING",
			ListClientOrderID: "amEEAXryFzFwYF1FeRpUoZ",
			TransactionTime:   types.Time(time.UnixMilli(1794464635319)),
			Symbol:            "BTCUSDT",
			Orders:            []OrderListOrder{{Symbol: "BTCUSDT", OrderID: 4, ClientOrderID: "oD7aesZqjEGlZrbtRpy5zB"}, {Symbol: "BTCUSDT", OrderID: 5, ClientOrderID: "Jr1h6xirOxgeJOUuYQS7V3"}},
		}}
		assert.Equal(t, exp, result, "GetAllOneCancelsTheOtherOrders should decode every field")
	}
}

func TestGetOpenOneCancelsTheOtherList(t *testing.T) {
	t.Parallel()
	skipIfCredentialsUnset(t)
	result, err := e.GetOpenOneCancelsTheOtherList(t.Context())
	require.NoError(t, err, "GetOpenOneCancelsTheOtherList must not error")
	if mockTests {
		exp := []OneCancelsTheOtherOrderResponse{{
			OrderListID:       31,
			ContingencyType:   "OCO",
			ListStatusType:    "EXEC_STARTED",
			ListOrderStatus:   "EXECUTING",
			ListClientOrderID: "wuB13fmulKj3YjdqWEcsnp",
			TransactionTime:   types.Time(time.UnixMilli(1794464635319)),
			Symbol:            "BTCUSDT",
			Orders:            []OrderListOrder{{Symbol: "BTCUSDT", OrderID: 4, ClientOrderID: "r3EH2N76dHfLoSZWIUw1bT"}, {Symbol: "BTCUSDT", OrderID: 5, ClientOrderID: "Cv1SnyPD3qhqpbjpYEHbd2"}},
		}}
		assert.Equal(t, exp, result, "GetOpenOneCancelsTheOtherList should decode every field")
	}
}

func TestCancelOneCancelsTheOtherOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelOneCancelsTheOtherOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelOneCancelsTheOtherOrder must reject a nil request")
	_, err = e.CancelOneCancelsTheOtherOrder(t.Context(), &CancelOrderListRequest{OrderListID: new(uint64(1))})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelOneCancelsTheOtherOrder must reject an empty pair")
	_, err = e.CancelOneCancelsTheOtherOrder(t.Context(), &CancelOrderListRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelOneCancelsTheOtherOrder must reject a request without an ID")

	t.Run("deletes the list by its IDs and symbol", func(t *testing.T) {
		t.Parallel()
		ex := signedRequestExchange(t, http.MethodDelete, "/api/v3/orderList", url.Values{"symbol": {"BTCUSDT"}, "orderListId": {"1929"}, "listClientOrderId": {"list-1"}, "newClientOrderId": {"cancel-1"}}, `{}`)
		_, err := ex.CancelOneCancelsTheOtherOrder(t.Context(), &CancelOrderListRequest{Symbol: currency.NewBTCUSDT(), OrderListID: new(uint64(1929)), ListClientOrderID: "list-1", NewClientOrderID: "cancel-1"})
		require.NoError(t, err, "CancelOneCancelsTheOtherOrder must not error")
	})

	if !mockTests && !canManipulateRealOrders {
		t.Skip("canManipulateRealOrders is false; skipping a live cancellation")
	}
	skipIfCredentialsUnset(t)
	result, err := e.CancelOneCancelsTheOtherOrder(t.Context(), &CancelOrderListRequest{Symbol: currency.NewBTCUSDT(), ListClientOrderID: "C3wyj4WVEktd7u9aVBRXcN"})
	require.NoError(t, err, "CancelOneCancelsTheOtherOrder must not error")
	if mockTests {
		report := func(orderID uint64, origClientOrderID string, orderType OrderType, price, stopPrice float64) OrderResponse {
			return OrderResponse{
				Symbol: "BTCUSDT", OriginalClientOrderID: origClientOrderID, OrderID: orderID, OrderListID: 1929, ClientOrderID: "unfWT8ig8i0uj6lPuYLez6",
				TransactTime: types.Time(time.UnixMilli(1794464635319)), Price: types.Number(price), OriginalQuantity: 0.002, Status: "CANCELED",
				TimeInForce: TimeInForceGoodTillCancel, Type: orderType, Side: SideSell, SelfTradePreventionMode: SelfTradePreventionModeExpireMaker,
				StopPrice: types.Number(stopPrice),
			}
		}
		exp := &OneCancelsTheOtherOrderResponse{
			OrderListID:       1929,
			ContingencyType:   "OCO",
			ListStatusType:    "ALL_DONE",
			ListOrderStatus:   "ALL_DONE",
			ListClientOrderID: "C3wyj4WVEktd7u9aVBRXcN",
			TransactionTime:   types.Time(time.UnixMilli(1794464635319)),
			Symbol:            "BTCUSDT",
			Orders:            []OrderListOrder{{Symbol: "BTCUSDT", OrderID: 20, ClientOrderID: "pO9ufTiFGg3nw2fOdgeOXa"}, {Symbol: "BTCUSDT", OrderID: 21, ClientOrderID: "TXOvglzXuaubXAaENpaRCB"}},
			OrderReports: []OrderResponse{
				report(20, "pO9ufTiFGg3nw2fOdgeOXa", OrderTypeStopLossLimit, 69900, 70000),
				report(21, "TXOvglzXuaubXAaENpaRCB", OrderTypeLimitMaker, 80000, 0),
			},
		}
		assert.Equal(t, exp, result, "CancelOneCancelsTheOtherOrder should decode every field")
	}
}
