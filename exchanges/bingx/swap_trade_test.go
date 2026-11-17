package bingx

import (
	"io"
	"math"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const swapTradeEmptyDataReply = `{"code":0,"msg":"","data":{}}`

// swapTradeTakeProfit is the take profit of the documented order examples
var swapTradeTakeProfit = &SwapTakeProfitStopLossRequest{Type: OrderTypeTakeProfitMarket, StopPrice: 31968, Price: 31968, WorkingType: WorkingTypeMarkPrice}

// swapTradeMarketOrder returns the documented MARKET order example, which attaches a take profit
func swapTradeMarketOrder() *SwapOrderRequest {
	return &SwapOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy, PositionSide: PositionSideLong, Type: OrderTypeMarket, Quantity: 5, TakeProfit: swapTradeTakeProfit}
}

// swapTradeBusinessParams returns a signed request's parameters without those the transport adds
func swapTradeBusinessParams(r *http.Request) url.Values {
	params := r.URL.Query()
	params.Del("recvWindow")
	params.Del("timestamp")
	params.Del("signature")
	return params
}

// swapTradeSkipIfCannotManipulateOrders skips a test that would change live orders or positions without permission
func swapTradeSkipIfCannotManipulateOrders(t *testing.T) {
	t.Helper()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
	}
}

// swapTradeOrderExample returns the order the documented pending order status and order details examples share
func swapTradeOrderExample(symbol string, orderID uint64) SwapOrderResponse {
	return SwapOrderResponse{
		Symbol:           symbol,
		OrderID:          NumericID(orderID),
		Side:             SideSell,
		PositionSide:     PositionSideLong,
		Type:             OrderTypeLimit,
		OriginalQuantity: 1,
		Price:            2.171,
		Status:           "PENDING",
		Time:             types.Time(time.UnixMilli(1702732596168)),
		UpdateTime:       types.Time(time.UnixMilli(1702732596188)),
		ClientOrderID:    "l*****e",
		WorkingType:      WorkingTypeMarkPrice,
	}
}

func TestPlaceSwapTestOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceSwapTestOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceSwapTestOrder must reject a nil request")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.PlaceSwapTestOrder(t.Context(), swapTradeMarketOrder())
	require.NoError(t, err, "PlaceSwapTestOrder must not error")
	if mockTests {
		exp := &SwapNewOrderResponse{
			Symbol:       "BTC-USDT",
			OrderID:      1735950529123455000,
			Side:         SideBuy,
			PositionSide: PositionSideLong,
			Type:         OrderTypeMarket,
			WorkingType:  WorkingTypeMarkPrice,
		}
		assert.Equal(t, exp, result, "PlaceSwapTestOrder should decode the documented example")
	} else {
		assert.NotNil(t, result, "PlaceSwapTestOrder should return an order")
	}
}

func TestPlaceSwapOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceSwapOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceSwapOrder must reject a nil request")
	_, err = e.PlaceSwapOrder(t.Context(), &SwapOrderRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PlaceSwapOrder must validate the order")

	swapTradeSkipIfCannotManipulateOrders(t)
	for _, tc := range []struct {
		name string
		arg  *SwapOrderRequest
	}{
		{"market order with a take profit", swapTradeMarketOrder()},
		{"position stop market order", &SwapOrderRequest{
			Symbol: currency.NewBTCUSDT(), Type: OrderTypeStopMarket, Side: SideSell, PositionSide: PositionSideLong, ClosePosition: true,
			StopPrice: 50000, Quantity: 0.002, TimeInForce: TimeInForceGoodTillCancel,
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.PlaceSwapOrder(t.Context(), tc.arg)
			require.NoError(t, err, "PlaceSwapOrder must not error")
			if mockTests {
				exp := &SwapNewOrderResponse{
					Symbol:       "BTC-USDT",
					OrderID:      1735950529123455000,
					Side:         SideBuy,
					PositionSide: PositionSideLong,
					Type:         OrderTypeMarket,
					WorkingType:  WorkingTypeMarkPrice,
				}
				assert.Equal(t, exp, result, "PlaceSwapOrder should decode the documented example")
			}
		})
	}
}

func TestSwapTradeNewOrder(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "method should be POST")
		assert.Equal(t, "/openApi/swap/v2/trade/order", r.URL.Path, "path should be the path given")
		exp := url.Values{
			"symbol":          {"ETH-USDT"},
			"type":            {"MARKET"},
			"side":            {"SELL"},
			"positionSide":    {"BOTH"},
			"reduceOnly":      {"true"},
			"price":           {"12.5"},
			"quantity":        {"0.002"},
			"quoteOrderQty":   {"100"},
			"stopPrice":       {"2900"},
			"priceRate":       {"0.1"},
			"workingType":     {"CONTRACT_PRICE"},
			"stopLoss":        {`{"type":"STOP","stopPrice":2800,"price":2790,"quantity":0.002,"stopGuaranteed":"cutfee"}`},
			"takeProfit":      {`{"type":"TAKE_PROFIT_MARKET","stopPrice":31968,"price":31968,"workingType":"MARK_PRICE"}`},
			"clientOrderId":   {"swap-order-1"},
			"timeInForce":     {"PostOnly"},
			"activationPrice": {"3000"},
			"stopGuaranteed":  {"true"},
			"positionId":      {"1769649551460794368"},
		}
		assert.Equal(t, exp, swapTradeBusinessParams(r), "every order field should be sent")
		verifyQuerySignature(t, r.URL.RawQuery)
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"order":{"symbol":"ETH-USDT","orderId":"1735950529123455001","side":"SELL","positionSide":"BOTH","type":"MARKET","clientOrderId":"swap-order-1","workingType":"CONTRACT_PRICE","reduceOnly":"true","stopGuaranteed":"true","status":"NEW","avgPrice":"2950.5","executedQty":"0.001"}}}`)
	})
	result, err := ex.swapTradeNewOrder(t.Context(), "/openApi/swap/v2/trade/order", swapPlaceOrderRate, &SwapOrderRequest{
		Symbol:             currency.NewPair(currency.ETH, currency.USDT),
		Type:               OrderTypeMarket,
		Side:               SideSell,
		PositionSide:       PositionSideBoth,
		ReduceOnly:         true,
		Price:              12.5,
		Quantity:           0.002,
		QuoteOrderQuantity: 100,
		StopPrice:          2900,
		PriceRate:          0.1,
		WorkingType:        WorkingTypeContractPrice,
		StopLoss:           &SwapTakeProfitStopLossRequest{Type: OrderTypeStop, StopPrice: 2800, Price: 2790, Quantity: 0.002, StopGuaranteed: StopGuaranteedCutFee},
		TakeProfit:         swapTradeTakeProfit,
		ClientOrderID:      "swap-order-1",
		TimeInForce:        TimeInForcePostOnly,
		ActivationPrice:    3000,
		StopGuaranteed:     StopGuaranteedTrue,
		PositionID:         1769649551460794368,
	})
	require.NoError(t, err, "swapTradeNewOrder must not error")
	exp := &SwapNewOrderResponse{
		Symbol:           "ETH-USDT",
		OrderID:          1735950529123455001,
		Side:             SideSell,
		PositionSide:     PositionSideBoth,
		Type:             OrderTypeMarket,
		ClientOrderID:    "swap-order-1",
		WorkingType:      WorkingTypeContractPrice,
		ReduceOnly:       true,
		StopGuaranteed:   StopGuaranteedTrue,
		Status:           "NEW",
		AveragePrice:     2950.5,
		ExecutedQuantity: 0.001,
	}
	assert.Equal(t, exp, result, "swapTradeNewOrder should return the order")

	_, err = ex.swapTradeNewOrder(t.Context(), "/openApi/swap/v2/trade/order", swapPlaceOrderRate, &SwapOrderRequest{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "swapTradeNewOrder should validate the order before sending it")

	ex = newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, swapTradeEmptyDataReply)
	})
	_, err = ex.swapTradeNewOrder(t.Context(), "/openApi/swap/v2/trade/order", swapPlaceOrderRate, swapTradeMarketOrder())
	assert.ErrorIs(t, err, common.ErrNoResponse, "swapTradeNewOrder should error when the reply holds no order")
}

func TestSwapTradeOrder(t *testing.T) {
	t.Parallel()
	_, err := e.swapTradeOrder(nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "swapTradeOrder must reject a nil request")
	arg := &SwapOrderRequest{}
	_, err = e.swapTradeOrder(arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "swapTradeOrder must reject an empty symbol")
	arg.Symbol = currency.NewBTCUSDT()
	_, err = e.swapTradeOrder(arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "swapTradeOrder must reject an empty order type")
	arg.Type = OrderTypeStopMarket
	_, err = e.swapTradeOrder(arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "swapTradeOrder must reject an empty side")
	arg.Side = SideSell
	arg.ReduceOnly = true
	arg.ClosePosition = true
	_, err = e.swapTradeOrder(arg)
	require.ErrorIs(t, err, errSwapClosePositionWithReduceOnly, "swapTradeOrder must reject close position with reduce only")
	arg.ClosePosition = false
	for _, side := range []PositionSide{PositionSideLong, PositionSideShort} {
		arg.PositionSide = side
		_, err = e.swapTradeOrder(arg)
		require.ErrorIsf(t, err, errSwapReduceOnlyInHedgeMode, "swapTradeOrder must reject reduce only with position side %s", side)
	}
	arg.PositionSide = PositionSideBoth
	arg.StopLoss = &SwapTakeProfitStopLossRequest{Type: OrderTypeStop, StopPrice: math.NaN()}
	_, err = e.swapTradeOrder(arg)
	require.ErrorContains(t, err, "stop loss", "swapTradeOrder must report a stop loss it cannot encode")
	arg.StopLoss = nil
	arg.TakeProfit = &SwapTakeProfitStopLossRequest{Type: OrderTypeTakeProfit, StopPrice: math.Inf(1)}
	_, err = e.swapTradeOrder(arg)
	require.ErrorContains(t, err, "take profit", "swapTradeOrder must report a take profit it cannot encode")
	arg.TakeProfit = nil
	_, err = new(Exchange).swapTradeOrder(arg)
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "swapTradeOrder must return the symbol formatting error")

	item, err := e.swapTradeOrder(arg)
	require.NoError(t, err, "swapTradeOrder must not error for a one-way reduce only order")
	assert.Equal(t, &SwapBatchOrderRequest{Symbol: "BTC-USDT", Type: OrderTypeStopMarket, Side: SideSell, PositionSide: PositionSideBoth, ReduceOnly: "true"}, item, "swapTradeOrder should send reduce only as a string")

	arg.ReduceOnly = false
	arg.ClosePosition = true
	arg.PositionSide = PositionSideLong
	item, err = e.swapTradeOrder(arg)
	require.NoError(t, err, "swapTradeOrder must not error for a hedge mode close position order")
	assert.Equal(t, &SwapBatchOrderRequest{Symbol: "BTC-USDT", Type: OrderTypeStopMarket, Side: SideSell, PositionSide: PositionSideLong, ClosePosition: "true"}, item, "swapTradeOrder should send close position as a string")

	t.Run("client order ID", func(t *testing.T) {
		t.Parallel()
		_, err := e.swapTradeOrder(&SwapOrderRequest{Symbol: currency.NewBTCUSDT(), Type: OrderTypeStopMarket, Side: SideSell, ClientOrderID: "c1"})
		assert.ErrorIs(t, err, errSwapClientOrderIDUnsupportedType, "swapTradeOrder should refuse a client order ID on a STOP_MARKET order")
		item, err := e.swapTradeOrder(&SwapOrderRequest{Symbol: currency.NewBTCUSDT(), Type: OrderTypeLimit, Side: SideSell, ClientOrderID: "c1"})
		require.NoError(t, err, "swapTradeOrder must not error for a LIMIT order with a client order ID")
		assert.Equal(t, "c1", item.ClientOrderID, "swapTradeOrder should send a LIMIT order's client order ID")
	})
}

func TestSwapTradeCheckClientOrderID(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		orderType     OrderType
		clientOrderID string
		err           error
	}{
		{name: "LIMIT order", orderType: OrderTypeLimit, clientOrderID: "c1"},
		{name: "MARKET order", orderType: OrderTypeMarket, clientOrderID: "c1"},
		{name: "STOP order without one", orderType: OrderTypeStop},
		{name: "STOP order", orderType: OrderTypeStop, clientOrderID: "c1", err: errSwapClientOrderIDUnsupportedType},
		{name: "TRAILING_STOP_MARKET order", orderType: OrderTypeTrailingStopMarket, clientOrderID: "c1", err: errSwapClientOrderIDUnsupportedType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := swapTradeCheckClientOrderID(tc.orderType, tc.clientOrderID)
			assert.ErrorIs(t, err, tc.err, "swapTradeCheckClientOrderID should accept a client order ID only on a LIMIT or MARKET order")
		})
	}
}

func TestSwapTradeTakeProfitStopLoss(t *testing.T) {
	t.Parallel()
	encoded, err := swapTradeTakeProfitStopLoss(nil)
	require.NoError(t, err, "swapTradeTakeProfitStopLoss must not error without an order")
	assert.Empty(t, encoded, "swapTradeTakeProfitStopLoss should return nothing without an order")

	encoded, err = swapTradeTakeProfitStopLoss(swapTradeTakeProfit)
	require.NoError(t, err, "swapTradeTakeProfitStopLoss must not error")
	assert.JSONEq(t, `{"type":"TAKE_PROFIT_MARKET","stopPrice":31968,"price":31968,"workingType":"MARK_PRICE"}`, encoded, "swapTradeTakeProfitStopLoss should encode the documented example's fields")

	encoded, err = swapTradeTakeProfitStopLoss(&SwapTakeProfitStopLossRequest{Type: OrderTypeStop, StopPrice: 37000, Price: 37000, WorkingType: WorkingTypeContractPrice, Quantity: 1, StopGuaranteed: StopGuaranteedTrue})
	require.NoError(t, err, "swapTradeTakeProfitStopLoss must not error")
	assert.JSONEq(t, `{"type":"STOP","stopPrice":37000,"price":37000,"workingType":"CONTRACT_PRICE","quantity":1,"stopGuaranteed":"true"}`, encoded, "swapTradeTakeProfitStopLoss should encode every field")

	_, err = swapTradeTakeProfitStopLoss(&SwapTakeProfitStopLossRequest{StopPrice: math.NaN()})
	assert.Error(t, err, "swapTradeTakeProfitStopLoss should error for a value JSON cannot hold")
}

func TestSwapTradeOrderValues(t *testing.T) {
	t.Parallel()
	assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "type": {"MARKET"}, "side": {"BUY"}}, swapTradeOrderValues(&SwapBatchOrderRequest{Symbol: "BTC-USDT", Type: OrderTypeMarket, Side: SideBuy}), "swapTradeOrderValues should leave unset fields out")

	item := &SwapBatchOrderRequest{
		Symbol: "BTC-USDT", Type: OrderTypeTriggerLimit, Side: SideBuy, PositionSide: PositionSideShort, ReduceOnly: "true", Price: 0.00000001,
		Quantity: 1e-7, QuoteOrderQuantity: 25.5, StopPrice: 70000, PriceRate: 0.05, WorkingType: WorkingTypeIndexPrice, StopLoss: `{"type":"STOP"}`,
		TakeProfit: `{"type":"TAKE_PROFIT"}`, ClientOrderID: "id", TimeInForce: TimeInForceFillOrKill, ClosePosition: "true", ActivationPrice: 69000,
		StopGuaranteed: StopGuaranteedFalse, PositionID: 1,
	}
	exp := url.Values{
		"symbol": {"BTC-USDT"}, "type": {"TRIGGER_LIMIT"}, "side": {"BUY"}, "positionSide": {"SHORT"}, "reduceOnly": {"true"}, "price": {"0.00000001"},
		"quantity": {"0.0000001"}, "quoteOrderQty": {"25.5"}, "stopPrice": {"70000"}, "priceRate": {"0.05"}, "workingType": {"INDEX_PRICE"},
		"stopLoss": {`{"type":"STOP"}`}, "takeProfit": {`{"type":"TAKE_PROFIT"}`}, "clientOrderId": {"id"}, "timeInForce": {"FOK"},
		"closePosition": {"true"}, "activationPrice": {"69000"}, "stopGuaranteed": {"false"}, "positionId": {"1"},
	}
	assert.Equal(t, exp, swapTradeOrderValues(item), "swapTradeOrderValues should send every field, numbers without exponents")
}

func TestPlaceSwapBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceSwapBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "PlaceSwapBatchOrders must reject an empty batch")
	_, err = e.PlaceSwapBatchOrders(t.Context(), make([]SwapOrderRequest, swapTradeBatchOrdersLimit+1))
	require.ErrorIs(t, err, errSwapTooManyItems, "PlaceSwapBatchOrders must reject more than five orders")
	_, err = e.PlaceSwapBatchOrders(t.Context(), []SwapOrderRequest{*swapTradeMarketOrder(), {}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PlaceSwapBatchOrders must validate every order")
	require.ErrorContains(t, err, "order 1", "PlaceSwapBatchOrders must name the order it rejects")
	_, err = e.PlaceSwapBatchOrders(t.Context(), []SwapOrderRequest{{Symbol: currency.NewBTCUSDT(), Type: OrderTypeLimit, Side: SideBuy, Price: math.Inf(-1)}})
	require.ErrorContains(t, err, "error encoding batch orders", "PlaceSwapBatchOrders must report a batch it cannot encode")
	_, err = e.PlaceSwapBatchOrders(t.Context(), []SwapOrderRequest{*swapTradeMarketOrder(), {Symbol: currency.NewBTCUSDT(), Type: OrderTypeStop, Side: SideSell, ClientOrderID: "a"}})
	require.ErrorIs(t, err, errSwapClientOrderIDUnsupportedType, "PlaceSwapBatchOrders must refuse a client order ID on a STOP order")

	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "method should be POST")
		exp := `[{"symbol":"BTC-USDT","type":"STOP","side":"SELL","positionSide":"LONG","price":80000,"quantity":0.002,"quoteOrderQty":160,` +
			`"stopPrice":79000,"priceRate":0.5,"workingType":"MARK_PRICE","stopLoss":"{\"type\":\"STOP\",\"stopPrice\":78000}",` +
			`"takeProfit":"{\"type\":\"TAKE_PROFIT_MARKET\",\"stopPrice\":31968,\"price\":31968,\"workingType\":\"MARK_PRICE\"}",` +
			`"timeInForce":"GTC","closePosition":"true","activationPrice":79500,"stopGuaranteed":"cutfee","positionId":12},` +
			`{"symbol":"ETH-USDT","type":"MARKET","side":"BUY","positionSide":"BOTH","reduceOnly":"true","quantity":1,"clientOrderId":"a"}]`
		assert.Equal(t, url.Values{"batchOrders": {exp}}, swapTradeBusinessParams(r), "every order field should be sent with its JSON type")
		verifyQuerySignature(t, r.URL.RawQuery)
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"orders":[{"symbol":"BTC-USDT","orderId":"7","reduceOnly":"false","stopGuaranteed":"cutfee","status":"NEW"}]}}`)
	})
	result, err := ex.PlaceSwapBatchOrders(t.Context(), []SwapOrderRequest{
		{
			Symbol: currency.NewBTCUSDT(), Type: OrderTypeStop, Side: SideSell, PositionSide: PositionSideLong, Price: 80000, Quantity: 0.002, QuoteOrderQuantity: 160,
			StopPrice: 79000, PriceRate: 0.5, WorkingType: WorkingTypeMarkPrice, StopLoss: &SwapTakeProfitStopLossRequest{Type: OrderTypeStop, StopPrice: 78000},
			TakeProfit: swapTradeTakeProfit, TimeInForce: TimeInForceGoodTillCancel, ClosePosition: true, ActivationPrice: 79500,
			StopGuaranteed: StopGuaranteedCutFee, PositionID: 12,
		},
		{Symbol: currency.NewPair(currency.ETH, currency.USDT), Type: OrderTypeMarket, Side: SideBuy, PositionSide: PositionSideBoth, ReduceOnly: true, Quantity: 1, ClientOrderID: "a"},
	})
	require.NoError(t, err, "PlaceSwapBatchOrders must not error")
	assert.Equal(t, []SwapNewOrderResponse{{Symbol: "BTC-USDT", OrderID: 7, StopGuaranteed: StopGuaranteedCutFee, Status: "NEW"}}, result, "PlaceSwapBatchOrders should return the orders BingX accepted")

	failing := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"code":100500,"msg":"busy"}`)
	})
	result, err = failing.PlaceSwapBatchOrders(t.Context(), []SwapOrderRequest{*swapTradeMarketOrder()})
	require.ErrorIs(t, err, errAPIResponse, "PlaceSwapBatchOrders must return the request's error")
	assert.Nil(t, result, "PlaceSwapBatchOrders should return no orders for a failed request")

	swapTradeSkipIfCannotManipulateOrders(t)
	result, err = e.PlaceSwapBatchOrders(t.Context(), []SwapOrderRequest{
		{Symbol: currency.NewPair(currency.ETH, currency.USDT), Type: OrderTypeMarket, Side: SideBuy, PositionSide: PositionSideLong, Quantity: 1},
		{Symbol: currency.NewBTCUSDT(), Type: OrderTypeMarket, Side: SideBuy, PositionSide: PositionSideLong, Quantity: 0.001},
	})
	require.NoError(t, err, "PlaceSwapBatchOrders must not error")
	if mockTests {
		exp := []SwapNewOrderResponse{
			{Symbol: "ID-USDT", OrderID: 1736010300483712300, Side: SideSell, PositionSide: PositionSideLong, Type: OrderTypeMarket},
		}
		assert.Equal(t, exp, result, "PlaceSwapBatchOrders should decode the documented example")
	}
}

func TestSwapTradeCheckListLength(t *testing.T) {
	t.Parallel()
	assert.NoError(t, swapTradeCheckListLength("orders", 0, 5), "swapTradeCheckListLength should accept an empty list")
	assert.NoError(t, swapTradeCheckListLength("orders", 5, 5), "swapTradeCheckListLength should accept a full list")
	err := swapTradeCheckListLength("orders", 6, 5)
	assert.ErrorIs(t, err, errSwapTooManyItems, "swapTradeCheckListLength should reject a list over its maximum")
	assert.ErrorContains(t, err, "6 orders, at most 5", "swapTradeCheckListLength should name the list and its limit")
}

func TestModifySwapOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifySwapOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ModifySwapOrder must reject a nil request")
	arg := &ModifySwapOrderRequest{}
	_, err = e.ModifySwapOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "ModifySwapOrder must reject a missing order ID")
	arg.OrderID = 1989246372640980993
	_, err = e.ModifySwapOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "ModifySwapOrder must reject an empty quantity")
	arg.Quantity = 0.00009
	_, err = e.ModifySwapOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "ModifySwapOrder must reject an empty symbol")
	arg.Symbol = currency.NewBTCUSDT()

	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, url.Values{"clientOrderId": {"swap-order-1"}, "quantity": {"1.5"}, "symbol": {"BTC-USDT"}}, swapTradeBusinessParams(r), "a client order ID should identify the order")
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"orderId":"1","clientOrderId":"swap-order-1","symbol":"BTC-USDT","quantity":1.5}}`)
	})
	result, err := ex.ModifySwapOrder(t.Context(), &ModifySwapOrderRequest{Symbol: currency.NewBTCUSDT(), ClientOrderID: "swap-order-1", Quantity: 1.5})
	require.NoError(t, err, "ModifySwapOrder must not error")
	assert.Equal(t, &ModifySwapOrderResponse{OrderID: 1, ClientOrderID: "swap-order-1", Symbol: "BTC-USDT", Quantity: 1.5}, result, "ModifySwapOrder should return the modified order")

	swapTradeSkipIfCannotManipulateOrders(t)
	result, err = e.ModifySwapOrder(t.Context(), arg)
	require.NoError(t, err, "ModifySwapOrder must not error")
	if mockTests {
		exp := &ModifySwapOrderResponse{OrderID: 1989246372640981000, Symbol: "BTC-USDT", Quantity: 0.00009}
		assert.Equal(t, exp, result, "ModifySwapOrder should decode the documented example")
	}
}

func TestSwapTradeSymbolParams(t *testing.T) {
	t.Parallel()
	_, err := e.swapTradeSymbolParams(currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "swapTradeSymbolParams must reject an empty symbol")
	params, err := e.swapTradeSymbolParams(currency.NewPair(currency.BTC, currency.USDC))
	require.NoError(t, err, "swapTradeSymbolParams must not error")
	assert.Equal(t, url.Values{"symbol": {"BTC-USDC"}}, params, "swapTradeSymbolParams should format the symbol for the USDT-M API")
	_, err = new(Exchange).swapTradeSymbolParams(currency.NewBTCUSDT())
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "swapTradeSymbolParams should return the symbol formatting error")
}

func TestSwapTradeSetSymbol(t *testing.T) {
	t.Parallel()
	params := url.Values{}
	require.NoError(t, e.swapTradeSetSymbol(params, currency.EMPTYPAIR), "swapTradeSetSymbol must not error without a symbol")
	assert.Empty(t, params, "swapTradeSetSymbol should leave the symbol out when none is given")
	require.NoError(t, e.swapTradeSetSymbol(params, currency.NewPairWithDelimiter("btc", "usdt", "_")), "swapTradeSetSymbol must not error")
	assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}}, params, "swapTradeSetSymbol should format the symbol for the USDT-M API")
	unformatted := url.Values{}
	assert.ErrorIs(t, new(Exchange).swapTradeSetSymbol(unformatted, currency.NewBTCUSDT()), currency.ErrPairManagerNotInitialised, "swapTradeSetSymbol should return the symbol formatting error")
	assert.Empty(t, unformatted, "swapTradeSetSymbol should set no symbol it cannot format")
}

func TestCloseAllSwapPositions(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, swapTradeBusinessParams(r), "no symbol should be sent to close every position")
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"success":["1"],"failed":[{"positionId":"2","errCode":3,"errorMsg":"m"}]}}`)
	})
	result, err := ex.CloseAllSwapPositions(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "CloseAllSwapPositions must not error")
	exp := &CloseAllSwapPositionsResponse{Success: []NumericID{1}, Failed: []json.RawMessage{json.RawMessage(`{"positionId":"2","errCode":3,"errorMsg":"m"}`)}}
	assert.Equal(t, exp, result, "CloseAllSwapPositions should keep each failure as BingX sent it")
	_, err = new(Exchange).CloseAllSwapPositions(t.Context(), currency.NewBTCUSDT())
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "CloseAllSwapPositions must return the symbol formatting error")

	swapTradeSkipIfCannotManipulateOrders(t)
	result, err = e.CloseAllSwapPositions(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "CloseAllSwapPositions must not error")
	if mockTests {
		assert.Equal(t, &CloseAllSwapPositionsResponse{Success: []NumericID{1736008778921491200}}, result, "CloseAllSwapPositions should decode the documented example")
	}
}

func TestCancelSwapOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelSwapOrder(t.Context(), currency.EMPTYPAIR, 1, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelSwapOrder must reject an empty symbol")
	_, err = e.CancelSwapOrder(t.Context(), currency.NewBTCUSDT(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelSwapOrder must reject a missing order ID")

	swapTradeSkipIfCannotManipulateOrders(t)
	result, err := e.CancelSwapOrder(t.Context(), currency.NewPair(currency.RNDR, currency.USDT), 1736011869418901234, "")
	require.NoError(t, err, "CancelSwapOrder must not error")
	if mockTests {
		exp := &SwapOrderResponse{
			Symbol:           "RNDR-USDT",
			OrderID:          1736011869418901200,
			Side:             SideBuy,
			PositionSide:     PositionSideLong,
			Type:             OrderTypeLimit,
			OriginalQuantity: 3,
			Price:            4.5081,
			Status:           "CANCELLED",
			Time:             types.Time(time.UnixMilli(1702732457867)),
			UpdateTime:       types.Time(time.UnixMilli(1702732457888)),
			ClientOrderID:    "lo******7",
		}
		assert.Equal(t, exp, result, "CancelSwapOrder should decode the documented example")
	}
}

func TestSwapTradeOrderIDParams(t *testing.T) {
	t.Parallel()
	_, err := e.swapTradeOrderIDParams(currency.EMPTYPAIR, 1, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "swapTradeOrderIDParams must reject an empty symbol")
	_, err = e.swapTradeOrderIDParams(currency.NewBTCUSDT(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "swapTradeOrderIDParams must reject a missing order ID")
	for _, tc := range []struct {
		orderID       uint64
		clientOrderID string
		exp           url.Values
	}{
		{1736011869418901234, "", url.Values{"symbol": {"BTC-USDT"}, "orderId": {"1736011869418901234"}}},
		{0, "abc", url.Values{"symbol": {"BTC-USDT"}, "clientOrderId": {"abc"}}},
		{1, "abc", url.Values{"symbol": {"BTC-USDT"}, "orderId": {"1"}, "clientOrderId": {"abc"}}},
	} {
		params, err := e.swapTradeOrderIDParams(currency.NewBTCUSDT(), tc.orderID, tc.clientOrderID)
		require.NoErrorf(t, err, "swapTradeOrderIDParams must not error for order ID %d and client order ID %q", tc.orderID, tc.clientOrderID)
		assert.Equalf(t, tc.exp, params, "swapTradeOrderIDParams should send the IDs given for order ID %d and client order ID %q", tc.orderID, tc.clientOrderID)
	}
}

func TestSwapTradeOrderReply(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method, "method should be the method given")
		assert.Equal(t, "/openApi/swap/v2/trade/order", r.URL.Path, "path should be the path given")
		assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}}, swapTradeBusinessParams(r), "the parameters given should be sent")
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"order":{"symbol":"BTC-USDT","orderId":7}}}`)
	})
	result, err := ex.swapTradeOrderReply(t.Context(), http.MethodDelete, "/openApi/swap/v2/trade/order", url.Values{"symbol": {"BTC-USDT"}}, swapCancelOrderRate)
	require.NoError(t, err, "swapTradeOrderReply must not error")
	assert.Equal(t, &SwapOrderResponse{Symbol: "BTC-USDT", OrderID: 7}, result, "swapTradeOrderReply should return the order")

	ex = newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, swapTradeEmptyDataReply)
	})
	_, err = ex.swapTradeOrderReply(t.Context(), http.MethodGet, "/openApi/swap/v2/trade/order", nil, swapOrderDetailsRate)
	assert.ErrorIs(t, err, common.ErrNoResponse, "swapTradeOrderReply should error when the reply holds no order")

	ex = newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"code":109421,"msg":"order not exist"}`)
	})
	_, err = ex.swapTradeOrderReply(t.Context(), http.MethodGet, "/openApi/swap/v2/trade/order", nil, swapOrderDetailsRate)
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "swapTradeOrderReply should return the API error")
}

func TestCancelSwapBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelSwapBatchOrders(t.Context(), currency.EMPTYPAIR, []uint64{1}, nil)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelSwapBatchOrders must reject an empty symbol")
	_, err = e.CancelSwapBatchOrders(t.Context(), currency.NewBTCUSDT(), nil, nil)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelSwapBatchOrders must reject empty ID lists")
	_, err = e.CancelSwapBatchOrders(t.Context(), currency.NewBTCUSDT(), make([]uint64, swapTradeBatchCancelLimit+1), nil)
	require.ErrorIs(t, err, errSwapTooManyItems, "CancelSwapBatchOrders must reject more than ten order IDs")
	_, err = e.CancelSwapBatchOrders(t.Context(), currency.NewBTCUSDT(), nil, make([]string, swapTradeBatchCancelLimit+1))
	require.ErrorIs(t, err, errSwapTooManyItems, "CancelSwapBatchOrders must reject more than ten client order IDs")

	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method, "method should be DELETE")
		exp := url.Values{"symbol": {"BTC-USDT"}, "orderIdList": {"[1735924831603391122,1]"}, "clientOrderIdList": {`["abc1234567","abc2345678"]`}}
		assert.Equal(t, exp, swapTradeBusinessParams(r), "both ID lists should be sent as JSON arrays")
		verifyQuerySignature(t, r.URL.RawQuery)
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"success":[],"failed":[{"orderId":"1","clientOrderId":"abc1234567","errorCode":109421,"errorMessage":"order not exist"}]}}`)
	})
	result, err := ex.CancelSwapBatchOrders(t.Context(), currency.NewBTCUSDT(), []uint64{1735924831603391122, 1}, []string{"abc1234567", "abc2345678"})
	require.NoError(t, err, "CancelSwapBatchOrders must not error")
	exp := &SwapCancelOrdersResponse{Success: []SwapOrderResponse{}, Failed: []SwapCancelOrderFailureResponse{{OrderID: 1, ClientOrderID: "abc1234567", ErrorCode: 109421, ErrorMessage: "order not exist"}}}
	assert.Equal(t, exp, result, "CancelSwapBatchOrders should report the orders it could not cancel")

	swapTradeSkipIfCannotManipulateOrders(t)
	result, err = e.CancelSwapBatchOrders(t.Context(), currency.NewBTCUSDT(), []uint64{1735924831603391122, 1735924833239172233}, nil)
	require.NoError(t, err, "CancelSwapBatchOrders must not error")
	if mockTests {
		exp := &SwapCancelOrdersResponse{Success: []SwapOrderResponse{
			{
				Symbol:           "BTC-USDT",
				OrderID:          1735924831603391200,
				Side:             SideBuy,
				PositionSide:     PositionSideLong,
				Type:             OrderTypeLimit,
				OriginalQuantity: 0.0032,
				Price:            41682.9,
				Status:           "CANCELLED",
				Time:             types.Time(time.UnixMilli(1702711706435)),
				UpdateTime:       types.Time(time.UnixMilli(1702711706453)),
			},
			{
				Symbol:           "BTC-USDT",
				OrderID:          1735924833239172400,
				Side:             SideBuy,
				PositionSide:     PositionSideLong,
				Type:             OrderTypeLimit,
				OriginalQuantity: 0.0033,
				Price:            41182.9,
				Status:           "CANCELLED",
				Time:             types.Time(time.UnixMilli(1702711706825)),
				UpdateTime:       types.Time(time.UnixMilli(1702711706838)),
			},
		}}
		assert.Equal(t, exp, result, "CancelSwapBatchOrders should decode the documented example")
	}
}

func TestCancelAllSwapOpenOrders(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method, "method should be DELETE")
		assert.Empty(t, swapTradeBusinessParams(r), "no symbol or type should be sent to cancel every order")
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"success":[],"failed":null}}`)
	})
	result, err := ex.CancelAllSwapOpenOrders(t.Context(), currency.EMPTYPAIR, "")
	require.NoError(t, err, "CancelAllSwapOpenOrders must not error")
	assert.Equal(t, &SwapCancelOrdersResponse{Success: []SwapOrderResponse{}}, result, "CancelAllSwapOpenOrders should decode an empty reply")
	_, err = new(Exchange).CancelAllSwapOpenOrders(t.Context(), currency.NewBTCUSDT(), OrderTypeLimit)
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "CancelAllSwapOpenOrders must return the symbol formatting error")

	swapTradeSkipIfCannotManipulateOrders(t)
	result, err = e.CancelAllSwapOpenOrders(t.Context(), currency.NewPair(currency.ATOM, currency.USDT), OrderTypeLimit)
	require.NoError(t, err, "CancelAllSwapOpenOrders must not error")
	if mockTests {
		exp := &SwapCancelOrdersResponse{
			Success: []SwapOrderResponse{
				{
					Symbol:           "ATOM-USDT",
					OrderID:          1736013373487123500,
					Side:             SideSell,
					PositionSide:     PositionSideShort,
					Type:             OrderTypeLimit,
					OriginalQuantity: 3,
					Price:            13.044,
					Status:           "CANCELLED",
					Time:             types.Time(time.UnixMilli(1702732816465)),
					UpdateTime:       types.Time(time.UnixMilli(1702732816488)),
				},
				{
					Symbol:           "ATOM-USDT",
					OrderID:          1736013373487123500,
					Side:             SideBuy,
					PositionSide:     PositionSideShort,
					Type:             OrderTypeLimit,
					OriginalQuantity: 3,
					Price:            11.292,
					Status:           "CANCELLED",
					Time:             types.Time(time.UnixMilli(1702732816820)),
					UpdateTime:       types.Time(time.UnixMilli(1702732816839)),
				},
			},
			Failed: []SwapCancelOrderFailureResponse{
				{OrderID: 111111, ClientOrderID: "111111", ErrorCode: 80012, ErrorMessage: "cancel order failed"},
				{OrderID: 222222, ClientOrderID: "222222", ErrorCode: 80012, ErrorMessage: "cancel order failed"},
			},
		}
		assert.Equal(t, exp, result, "CancelAllSwapOpenOrders should decode the documented example")
	}
}

func TestGetSwapOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).GetSwapOpenOrders(t.Context(), currency.NewBTCUSDT(), OrderTypeLimit)
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSwapOpenOrders must return the symbol formatting error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSwapOpenOrders(t.Context(), currency.NewBTCUSDT(), OrderTypeLimit)
	require.NoError(t, err, "GetSwapOpenOrders must not error")
	if mockTests {
		exp := []SwapOrderResponse{
			{
				Symbol:           "BTC-USDT",
				OrderID:          1733405587011123500,
				Side:             SideSell,
				PositionSide:     PositionSideLong,
				Type:             OrderTypeLimit,
				OriginalQuantity: 0.003,
				Price:            44459.6,
				Status:           "PENDING",
				Time:             types.Time(time.UnixMilli(1702256915574)),
				UpdateTime:       types.Time(time.UnixMilli(1702256915610)),
				WorkingType:      WorkingTypeMarkPrice,
			},
			{
				Symbol:           "BTC-USDT",
				OrderID:          1733405587011123500,
				Side:             SideSell,
				PositionSide:     PositionSideLong,
				Type:             OrderTypeLimit,
				OriginalQuantity: 0.003,
				Price:            44454.6,
				Status:           "PENDING",
				Time:             types.Time(time.UnixMilli(1702111071719)),
				UpdateTime:       types.Time(time.UnixMilli(1702111071735)),
				WorkingType:      WorkingTypeMarkPrice,
			},
		}
		assert.Equal(t, exp, result, "GetSwapOpenOrders should decode the documented example")
	}
}

func TestSwapTradeOrderList(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "method should be GET")
		assert.Equal(t, "/openApi/swap/v2/trade/openOrders", r.URL.Path, "path should be the path given")
		assert.Equal(t, url.Values{"type": {"LIMIT"}}, swapTradeBusinessParams(r), "the parameters given should be sent")
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"orders":[{"symbol":"BTC-USDT","orderId":"7","priceRate":"0.1","actPrice":"30000","closePosition":"true","reduceOnly":"false","postOnly":true,"trailingStopRate":0.1,"trailingStopDistance":25.5}]}}`)
	})
	result, err := ex.swapTradeOrderList(t.Context(), "/openApi/swap/v2/trade/openOrders", url.Values{"type": {"LIMIT"}}, swapOpenOrdersRate)
	require.NoError(t, err, "swapTradeOrderList must not error")
	exp := []SwapOrderResponse{{
		Symbol: "BTC-USDT", OrderID: 7, PriceRate: 0.1, ActivationPrice: 30000, ClosePosition: true, PostOnly: true, TrailingStopRate: 0.1, TrailingStopDistance: 25.5,
	}}
	assert.Equal(t, exp, result, "swapTradeOrderList should return the orders")

	ex = newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"code":100001,"msg":"signature verification failed"}`)
	})
	_, err = ex.swapTradeOrderList(t.Context(), "/openApi/swap/v2/trade/openOrders", nil, swapOpenOrdersRate)
	assert.ErrorIs(t, err, errAPIResponse, "swapTradeOrderList should return the API error")
}

func TestGetSwapPendingOrderStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapPendingOrderStatus(t.Context(), currency.EMPTYPAIR, 1, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSwapPendingOrderStatus must reject an empty symbol")
	_, err = e.GetSwapPendingOrderStatus(t.Context(), currency.NewBTCUSDT(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSwapPendingOrderStatus must reject a missing order ID")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSwapPendingOrderStatus(t.Context(), currency.NewPair(currency.OP, currency.USDT), 1736012449498123456, "")
	require.NoError(t, err, "GetSwapPendingOrderStatus must not error")
	if mockTests {
		exp := swapTradeOrderExample("OP-USDT", 1736012449498123500)
		assert.Equal(t, &exp, result, "GetSwapPendingOrderStatus should decode the documented example")
	}
}

func TestGetSwapOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapOrderDetails(t.Context(), currency.EMPTYPAIR, 1, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSwapOrderDetails must reject an empty symbol")
	_, err = e.GetSwapOrderDetails(t.Context(), currency.NewBTCUSDT(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSwapOrderDetails must reject a missing order ID")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSwapOrderDetails(t.Context(), currency.NewPair(currency.OP, currency.USDT), 1736012449498123456, "")
	require.NoError(t, err, "GetSwapOrderDetails must not error")
	if mockTests {
		exp := swapTradeOrderExample("OP-USDT", 1736012449498123500)
		exp.StopGuaranteed = StopGuaranteedFalse
		exp.TriggerOrderID = 1736012449498123500
		assert.Equal(t, &exp, result, "GetSwapOrderDetails should decode the documented example")
	}
}

func TestGetSwapMarginType(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapMarginType(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSwapMarginType must reject an empty symbol")
	ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, swapTradeEmptyDataReply)
	})
	_, err = ex.GetSwapMarginType(t.Context(), currency.NewBTCUSDT())
	require.ErrorIs(t, err, common.ErrNoResponse, "GetSwapMarginType must error for a reply without a margin type")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSwapMarginType(t.Context(), currency.NewPair(currency.WOO, currency.USDT))
	require.NoError(t, err, "GetSwapMarginType must not error")
	if mockTests {
		assert.Equal(t, &SwapMarginTypeResponse{MarginType: MarginTypeCrossed}, result, "GetSwapMarginType should decode the documented example")
	} else {
		assert.NotEmpty(t, result.MarginType, "GetSwapMarginType should return a margin type")
	}
}

func TestChangeSwapMarginType(t *testing.T) {
	t.Parallel()
	err := e.ChangeSwapMarginType(t.Context(), currency.EMPTYPAIR, MarginTypeCrossed)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "ChangeSwapMarginType must reject an empty symbol")
	err = e.ChangeSwapMarginType(t.Context(), currency.NewBTCUSDT(), "")
	require.ErrorIs(t, err, errSwapMarginTypeRequired, "ChangeSwapMarginType must reject an empty margin type")

	swapTradeSkipIfCannotManipulateOrders(t)
	require.NoError(t, e.ChangeSwapMarginType(t.Context(), currency.NewPair(currency.MINA, currency.USDT), MarginTypeCrossed), "ChangeSwapMarginType must not error")
}

func TestGetSwapLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapLeverage(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSwapLeverage must reject an empty symbol")
	ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, swapTradeEmptyDataReply)
	})
	_, err = ex.GetSwapLeverage(t.Context(), currency.NewBTCUSDT())
	require.ErrorIs(t, err, common.ErrNoResponse, "GetSwapLeverage must error for a reply without leverage")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSwapLeverage(t.Context(), currency.NewPair(currency.BCH, currency.USDT))
	require.NoError(t, err, "GetSwapLeverage must not error")
	if mockTests {
		assert.Equal(t, &SwapLeverageResponse{LongLeverage: 50, ShortLeverage: 50, MaxLongLeverage: 75, MaxShortLeverage: 75}, result, "GetSwapLeverage should decode the documented example")
	} else {
		assert.NotZero(t, result.MaxLongLeverage, "GetSwapLeverage should return the maximum leverage")
	}
}

func TestSetSwapLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.SetSwapLeverage(t.Context(), currency.EMPTYPAIR, PositionSideShort, 8)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "SetSwapLeverage must reject an empty symbol")
	_, err = e.SetSwapLeverage(t.Context(), currency.NewBTCUSDT(), "", 8)
	require.ErrorIs(t, err, errPositionSideRequired, "SetSwapLeverage must reject an empty side")
	_, err = e.SetSwapLeverage(t.Context(), currency.NewBTCUSDT(), PositionSideShort, 0)
	require.ErrorIs(t, err, errSwapLeverageRequired, "SetSwapLeverage must reject a zero leverage")

	swapTradeSkipIfCannotManipulateOrders(t)
	result, err := e.SetSwapLeverage(t.Context(), currency.NewPair(currency.ETH, currency.USDT), PositionSideShort, 8)
	require.NoError(t, err, "SetSwapLeverage must not error")
	if mockTests {
		assert.Equal(t, &SetSwapLeverageResponse{Leverage: 8, Symbol: "ETH-USDT"}, result, "SetSwapLeverage should decode the documented example")
	}
}

func TestGetSwapForceOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapForceOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSwapForceOrders must reject a nil request")
	_, err = e.GetSwapForceOrders(t.Context(), &SwapForceOrdersRequest{StartTime: time.UnixMilli(2), EndTime: time.UnixMilli(1)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSwapForceOrders must reject a reversed time range")
	_, err = new(Exchange).GetSwapForceOrders(t.Context(), &SwapForceOrdersRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSwapForceOrders must return the symbol formatting error")

	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		exp := url.Values{"symbol": {"BTC-USDC"}, "currency": {"USDC"}, "autoCloseType": {"ADL"}, "startTime": {"1"}, "endTime": {"2"}, "limit": {"100"}}
		assert.Equal(t, exp, swapTradeBusinessParams(r), "every filter should be sent")
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"orders":[]}}`)
	})
	result, err := ex.GetSwapForceOrders(t.Context(), &SwapForceOrdersRequest{
		Symbol: currency.NewPair(currency.BTC, currency.USDC), Currency: currency.NewCode("usdc"), AutoCloseType: AutoCloseTypeAutoDeleveraging,
		StartTime: time.UnixMilli(1), EndTime: time.UnixMilli(2), Limit: 100,
	})
	require.NoError(t, err, "GetSwapForceOrders must not error")
	assert.Empty(t, result, "GetSwapForceOrders should return no orders")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err = e.GetSwapForceOrders(t.Context(), &SwapForceOrdersRequest{Symbol: currency.NewPair(currency.ATOM, currency.USDT), StartTime: time.Unix(1696291200, 0)})
	if mockTests {
		// The documented example's order ID, 172264854643022330000, overflows the int64 the documentation gives it
		assert.ErrorIs(t, err, errInvalidNumericID, "GetSwapForceOrders should reject the documented example's order ID")
		return
	}
	require.NoError(t, err, "GetSwapForceOrders must not error")
	assert.NotNil(t, result, "GetSwapForceOrders should return a list")
}

func TestGetSwapOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSwapOrderHistory must reject a nil request")
	_, err = e.GetSwapOrderHistory(t.Context(), &SwapOrderHistoryRequest{StartTime: time.UnixMilli(2), EndTime: time.UnixMilli(1)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSwapOrderHistory must reject a reversed time range")
	_, err = new(Exchange).GetSwapOrderHistory(t.Context(), &SwapOrderHistoryRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSwapOrderHistory must return the symbol formatting error")

	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, url.Values{"orderId": {"5"}}, swapTradeBusinessParams(r), "only the filters given should be sent")
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"orders":[]}}`)
	})
	_, err = ex.GetSwapOrderHistory(t.Context(), &SwapOrderHistoryRequest{OrderID: 5})
	require.NoError(t, err, "GetSwapOrderHistory must not error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSwapOrderHistory(t.Context(), &SwapOrderHistoryRequest{
		Symbol: currency.NewPair(currency.BTC, currency.USDC), Currency: currency.USDC, StartTime: time.UnixMilli(1785503625603), EndTime: time.UnixMilli(1786108425603), Limit: 100,
	})
	require.NoError(t, err, "GetSwapOrderHistory must not error")
	if mockTests {
		exp := []SwapOrderResponse{{
			Symbol:           "BTC-USDC",
			OrderID:          2085388523196895200,
			Side:             SideBuy,
			PositionSide:     PositionSideLong,
			Type:             OrderTypeMarket,
			OriginalQuantity: 0.0001,
			ExecutedQuantity: 0.0001,
			AveragePrice:     75586.05,
			CumulativeQuote:  8,
			Commission:       -0.003779,
			Status:           "FILLED",
			Time:             types.Time(time.UnixMilli(1786030343000)),
			UpdateTime:       types.Time(time.UnixMilli(1786030343000)),
			Leverage:         "20X",
			PositionID:       2085388523545022500,
			WorkingType:      WorkingTypeMarkPrice,
			StopGuaranteed:   StopGuaranteedFalse,
		}}
		assert.Equal(t, exp, result, "GetSwapOrderHistory should decode the documented example")
	}
}

func TestModifySwapIsolatedPositionMargin(t *testing.T) {
	t.Parallel()
	_, err := e.ModifySwapIsolatedPositionMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ModifySwapIsolatedPositionMargin must reject a nil request")
	arg := &ModifySwapIsolatedPositionMarginRequest{}
	_, err = e.ModifySwapIsolatedPositionMargin(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "ModifySwapIsolatedPositionMargin must reject an empty symbol")
	arg.Symbol = currency.NewBTCUSDT()
	_, err = e.ModifySwapIsolatedPositionMargin(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "ModifySwapIsolatedPositionMargin must reject an empty amount")
	arg.Amount = 3
	_, err = e.ModifySwapIsolatedPositionMargin(t.Context(), arg)
	require.ErrorIs(t, err, errSwapMarginAdjustmentTypeRequired, "ModifySwapIsolatedPositionMargin must reject an empty type")
	arg.Type = SwapMarginAdjustmentTypeIncrease
	arg.PositionSide = PositionSideLong

	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		exp := url.Values{"symbol": {"BTC-USDT"}, "amount": {"0.5"}, "type": {"2"}, "positionId": {"1847596444958068736"}}
		assert.Equal(t, exp, swapTradeBusinessParams(r), "a position ID should identify the position")
		_, _ = io.WriteString(w, `{"code":0,"msg":"","amount":0.5,"type":2,"positionId":"1847596444958068736"}`)
	})
	result, err := ex.ModifySwapIsolatedPositionMargin(t.Context(), &ModifySwapIsolatedPositionMarginRequest{Symbol: currency.NewBTCUSDT(), Amount: 0.5, Type: SwapMarginAdjustmentTypeDecrease, PositionID: 1847596444958068736})
	require.NoError(t, err, "ModifySwapIsolatedPositionMargin must not error")
	assert.Equal(t, &ModifySwapIsolatedPositionMarginResponse{Amount: 0.5, Type: SwapMarginAdjustmentTypeDecrease, PositionID: 1847596444958068736}, result, "ModifySwapIsolatedPositionMargin should decode the fields beside the code")

	swapTradeSkipIfCannotManipulateOrders(t)
	result, err = e.ModifySwapIsolatedPositionMargin(t.Context(), arg)
	require.NoError(t, err, "ModifySwapIsolatedPositionMargin must not error")
	if mockTests {
		assert.Equal(t, &ModifySwapIsolatedPositionMarginResponse{Amount: 3, Type: SwapMarginAdjustmentTypeIncrease}, result, "ModifySwapIsolatedPositionMargin should decode the documented example")
	}
}

func TestGetSwapHistoricalTransactionOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapHistoricalTransactionOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSwapHistoricalTransactionOrders must reject a nil request")
	arg := &SwapHistoricalTransactionOrdersRequest{}
	_, err = e.GetSwapHistoricalTransactionOrders(t.Context(), arg)
	require.ErrorIs(t, err, errSwapTradingUnitRequired, "GetSwapHistoricalTransactionOrders must reject an empty trading unit")
	arg.TradingUnit = TradingUnitCoin
	arg.StartTime = time.UnixMilli(1785503627542)
	_, err = e.GetSwapHistoricalTransactionOrders(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrDateUnset, "GetSwapHistoricalTransactionOrders must reject a missing end time")
	arg.EndTime = time.UnixMilli(1)
	_, err = e.GetSwapHistoricalTransactionOrders(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSwapHistoricalTransactionOrders must reject a reversed time range")
	arg.EndTime = time.UnixMilli(1786108427542)
	arg.Symbol = currency.NewPair(currency.BTC, currency.USDC)
	arg.Currency = currency.USDC
	_, err = new(Exchange).GetSwapHistoricalTransactionOrders(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSwapHistoricalTransactionOrders must return the symbol formatting error")

	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		exp := url.Values{"tradingUnit": {"CONT"}, "startTs": {"1"}, "endTs": {"2"}, "orderId": {"2085388523196895233"}}
		assert.Equal(t, exp, swapTradeBusinessParams(r), "every filter given should be sent")
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"fill_orders":[{"workingType":"MARK_PRICE","side":"SELL","type":"LIMIT","positionSide":"SHORT","onlyOnePosition":true}]}}`)
	})
	result, err := ex.GetSwapHistoricalTransactionOrders(t.Context(), &SwapHistoricalTransactionOrdersRequest{TradingUnit: TradingUnitContract, StartTime: time.UnixMilli(1), EndTime: time.UnixMilli(2), OrderID: 2085388523196895233})
	require.NoError(t, err, "GetSwapHistoricalTransactionOrders must not error")
	assert.Equal(t, []SwapHistoricalTransactionOrderResponse{{WorkingType: WorkingTypeMarkPrice, Side: SideSell, Type: OrderTypeLimit, PositionSide: PositionSideShort, OnlyOnePosition: true}}, result, "GetSwapHistoricalTransactionOrders should decode the documented fields its example leaves out")

	failing := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"code":100500,"msg":"busy"}`)
	})
	result, err = failing.GetSwapHistoricalTransactionOrders(t.Context(), arg)
	require.ErrorIs(t, err, errAPIResponse, "GetSwapHistoricalTransactionOrders must return the request's error")
	assert.Nil(t, result, "GetSwapHistoricalTransactionOrders should return no fills for a failed request")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err = e.GetSwapHistoricalTransactionOrders(t.Context(), arg)
	require.NoError(t, err, "GetSwapHistoricalTransactionOrders must not error")
	if mockTests {
		require.Len(t, result, 1, "GetSwapHistoricalTransactionOrders must return the documented fill")
		// filledTime carries BingX's +08:00 offset, which a time.Time comparison sees as a different location
		result[0].FilledTime = result[0].FilledTime.UTC()
		exp := []SwapHistoricalTransactionOrderResponse{{
			FilledTimeUTC: time.Date(2026, 8, 6, 15, 32, 23, 0, time.UTC),
			Symbol:        "BTC-USDC",
			Volume:        0.0001,
			Price:         75586.05,
			Amount:        7.5586,
			Commission:    -0.003779,
			Currency:      currency.USDC,
			OrderID:       2085388523196895233,
			FilledTime:    time.Date(2026, 8, 6, 15, 32, 23, 0, time.UTC),
		}}
		assert.Equal(t, exp, result, "GetSwapHistoricalTransactionOrders should decode the documented example")
	}
}

func TestSetSwapPositionMode(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, url.Values{"dualSidePosition": {"false"}}, swapTradeBusinessParams(r), "one-way mode should be sent as false")
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"dualSidePosition":"false"}}`)
	})
	result, err := ex.SetSwapPositionMode(t.Context(), false)
	require.NoError(t, err, "SetSwapPositionMode must not error")
	assert.Equal(t, &SwapPositionModeResponse{}, result, "SetSwapPositionMode should decode one-way mode")

	t.Run("cached mode forgotten", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v1/positionSide/dual":  {wrapperHedgeModeReply, wrapperOneWayModeReply, wrapperOneWayModeReply},
			"POST /openApi/swap/v1/positionSide/dual": {`{"code":0,"data":{"dualSidePosition":"false"}}`, `{"code":100500,"msg":"busy"}`},
		})
		hedgeMode, err := ex.swapHedgeMode(t.Context())
		require.NoError(t, err, "swapHedgeMode must not error")
		require.True(t, hedgeMode, "swapHedgeMode must cache hedge mode")
		account, err := ex.productAccount(t.Context(), exchange.RestUSDTMargined)
		require.NoError(t, err, "productAccount must not error")
		_, _, inFlight := ex.positionModes.get(account)
		_, err = ex.SetSwapPositionMode(t.Context(), false)
		require.NoError(t, err, "SetSwapPositionMode must not error")
		ex.positionModes.set(account, inFlight, true)
		_, cached, _ := ex.positionModes.get(account)
		assert.False(t, cached, "SetSwapPositionMode should drop the mode of a lookup in flight across the change")
		hedgeMode, err = ex.swapHedgeMode(t.Context())
		require.NoError(t, err, "swapHedgeMode must not error")
		assert.False(t, hedgeMode, "SetSwapPositionMode should make the next lookup fetch the new mode")

		_, err = ex.SetSwapPositionMode(t.Context(), true)
		assert.ErrorIs(t, err, errAPIResponse, "SetSwapPositionMode should return the request's error")
		_, err = ex.swapHedgeMode(t.Context())
		require.NoError(t, err, "swapHedgeMode must not error")
		assert.Len(t, requests(), 5, "SetSwapPositionMode should forget the cached mode whatever the outcome, as a failed change may have happened")
	})

	swapTradeSkipIfCannotManipulateOrders(t)
	result, err = e.SetSwapPositionMode(t.Context(), true)
	require.NoError(t, err, "SetSwapPositionMode must not error")
	if mockTests {
		assert.Equal(t, &SwapPositionModeResponse{DualSidePosition: true}, result, "SetSwapPositionMode should decode the documented example")
	}
}

func TestGetSwapPositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSwapPositionMode(t.Context())
	require.NoError(t, err, "GetSwapPositionMode must not error")
	if mockTests {
		assert.Equal(t, &SwapPositionModeResponse{DualSidePosition: true}, result, "GetSwapPositionMode should decode the documented example")
	} else {
		assert.NotNil(t, result, "GetSwapPositionMode should return the position mode")
	}
}

func TestCancelReplaceSwapOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelReplaceSwapOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelReplaceSwapOrder must reject a nil request")

	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "method should be POST")
		exp := url.Values{
			"cancelReplaceMode": {"ALLOW_FAILURE"}, "cancelClientOrderId": {"old"}, "symbol": {"BTC-USDT"}, "type": {"LIMIT"}, "side": {"SELL"},
			"positionSide": {"SHORT"}, "price": {"81000"}, "quantity": {"0.01"}, "clientOrderId": {"new"},
		}
		assert.Equal(t, exp, swapTradeBusinessParams(r), "the cancellation and the new order should be sent")
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"cancelResult":"false","cancelMsg":"order not exist","cancelResponse":{"cancelOrderId":0},"replaceResult":"true","replaceMsg":"","newOrderResponse":{"orderId":"5","symbol":"BTC-USDT","clientOrderId":"new","reduceOnly":"false"}}}`)
	})
	result, err := ex.CancelReplaceSwapOrder(t.Context(), &CancelReplaceSwapOrderRequest{
		Symbol: currency.NewBTCUSDT(), Type: OrderTypeLimit, Side: SideSell, PositionSide: PositionSideShort, Price: 81000, Quantity: 0.01, ClientOrderID: "new",
		CancelReplaceMode: CancelReplaceModeAllowFailure, CancelClientOrderID: "old",
	})
	require.NoError(t, err, "CancelReplaceSwapOrder must not error")
	exp := &SwapCancelReplaceOrderResponse{
		CancelMessage:    "order not exist",
		ReplaceResult:    true,
		NewOrderResponse: SwapCancelReplaceNewOrderResponse{OrderID: 5, Symbol: "BTC-USDT", ClientOrderID: "new"},
	}
	assert.Equal(t, exp, result, "CancelReplaceSwapOrder should report a failed cancellation beside the new order")

	swapTradeSkipIfCannotManipulateOrders(t)
	result, err = e.CancelReplaceSwapOrder(t.Context(), &CancelReplaceSwapOrderRequest{
		SwapOrderRequest:    *swapTradeMarketOrder(),
		CancelReplaceMode:   CancelReplaceModeStopOnFailure,
		CancelClientOrderID: "abc123test",
		CancelOrderID:       123456789,
		CancelRestrictions:  CancelRestrictionsOnlyNew,
	})
	require.NoError(t, err, "CancelReplaceSwapOrder must not error")
	if mockTests {
		exp := &SwapCancelReplaceOrderResponse{
			CancelResult: true,
			CancelResponse: SwapCancelReplaceCancelResponse{
				CancelOrderID: 123456789,
				SwapOrderResponse: SwapOrderResponse{
					Symbol:           "BTC-USDT",
					OrderID:          123456789,
					Side:             SideBuy,
					PositionSide:     PositionSideLong,
					Type:             OrderTypeLimit,
					OriginalQuantity: 1,
					Price:            38000,
					Status:           "PENDING",
					Time:             types.Time(time.UnixMilli(1706858471000)),
					UpdateTime:       types.Time(time.UnixMilli(1706858471000)),
					Leverage:         "15X",
					WorkingType:      WorkingTypeMarkPrice,
				},
			},
			ReplaceResult: true,
			NewOrderResponse: SwapCancelReplaceNewOrderResponse{
				OrderID:      987654321,
				Symbol:       "BTC-USDT",
				PositionSide: PositionSideLong,
				Side:         SideBuy,
				Type:         OrderTypeLimit,
				Price:        38000,
				Quantity:     1,
				WorkingType:  WorkingTypeMarkPrice,
				TimeInForce:  TimeInForceGoodTillCancel,
				StopLoss:     `{"type": "STOP", "stopPrice": 37000, "price": 37000}`,
				TakeProfit:   `{"type": "TAKE_PROFIT", "stopPrice": 45000, "price": 45000}`,
			},
		}
		assert.Equal(t, exp, result, "CancelReplaceSwapOrder should decode the documented example")
	}
}

func TestSwapTradeCancelReplaceOrder(t *testing.T) {
	t.Parallel()
	_, err := e.swapTradeCancelReplaceOrder(nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "swapTradeCancelReplaceOrder must reject a nil request")
	arg := &CancelReplaceSwapOrderRequest{}
	_, err = e.swapTradeCancelReplaceOrder(arg)
	require.ErrorIs(t, err, errSwapCancelReplaceModeRequired, "swapTradeCancelReplaceOrder must reject an empty cancel replace mode")
	arg.CancelReplaceMode = CancelReplaceModeStopOnFailure
	_, err = e.swapTradeCancelReplaceOrder(arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "swapTradeCancelReplaceOrder must reject a missing order to cancel")
	arg.CancelOrderID = 1753337028434464768
	_, err = e.swapTradeCancelReplaceOrder(arg)
	require.ErrorIs(t, err, errPositionSideRequired, "swapTradeCancelReplaceOrder must reject an empty position side")
	arg.PositionSide = PositionSideLong
	_, err = e.swapTradeCancelReplaceOrder(arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "swapTradeCancelReplaceOrder must validate the new order")
	arg.Symbol = currency.NewBTCUSDT()
	arg.Type = OrderTypeLimit
	arg.Side = SideBuy
	arg.CancelClientOrderID = "abc"
	arg.CancelRestrictions = CancelRestrictionsOnlyPartiallyFilled
	item, err := e.swapTradeCancelReplaceOrder(arg)
	require.NoError(t, err, "swapTradeCancelReplaceOrder must not error")
	exp := &SwapBatchCancelReplaceOrderRequest{
		CancelReplaceMode:   CancelReplaceModeStopOnFailure,
		CancelOrderID:       1753337028434464768,
		CancelClientOrderID: "abc",
		CancelRestrictions:  CancelRestrictionsOnlyPartiallyFilled,
		Symbol:              "BTC-USDT",
		Type:                OrderTypeLimit,
		Side:                SideBuy,
		PositionSide:        PositionSideLong,
	}
	assert.Equal(t, exp, item, "swapTradeCancelReplaceOrder should hold the cancellation and the new order")
}

func TestCancelReplaceSwapBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelReplaceSwapBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "CancelReplaceSwapBatchOrders must reject an empty batch")
	_, err = e.CancelReplaceSwapBatchOrders(t.Context(), []CancelReplaceSwapOrderRequest{{}})
	require.ErrorIs(t, err, errSwapCancelReplaceModeRequired, "CancelReplaceSwapBatchOrders must validate every cancel-replace")
	require.ErrorContains(t, err, "order 0", "CancelReplaceSwapBatchOrders must name the cancel-replace it rejects")
	_, err = e.CancelReplaceSwapBatchOrders(t.Context(), []CancelReplaceSwapOrderRequest{{
		Symbol: currency.NewBTCUSDT(), Type: OrderTypeLimit, Side: SideBuy, PositionSide: PositionSideLong, Quantity: math.NaN(),
		CancelReplaceMode: CancelReplaceModeAllowFailure, CancelOrderID: 1,
	}})
	require.ErrorContains(t, err, "error encoding batch orders", "CancelReplaceSwapBatchOrders must report a batch it cannot encode")

	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		exp := `[{"cancelReplaceMode":"STOP_ON_FAILURE","cancelOrderId":1,"cancelClientOrderId":"old","cancelRestrictions":"ONLY_PENDING",` +
			`"symbol":"BTC-USDT","type":"LIMIT","side":"SELL","positionSide":"SHORT","price":81000,"quantity":0.01}]`
		assert.Equal(t, url.Values{"batchOrders": {exp}}, swapTradeBusinessParams(r), "each cancel-replace should be sent as one JSON object")
		verifyQuerySignature(t, r.URL.RawQuery)
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":[]}`)
	})
	result, err := ex.CancelReplaceSwapBatchOrders(t.Context(), []CancelReplaceSwapOrderRequest{{
		Symbol: currency.NewBTCUSDT(), Type: OrderTypeLimit, Side: SideSell, PositionSide: PositionSideShort, Price: 81000, Quantity: 0.01,
		CancelReplaceMode: CancelReplaceModeStopOnFailure, CancelOrderID: 1, CancelClientOrderID: "old", CancelRestrictions: CancelRestrictionsOnlyPending,
	}})
	require.NoError(t, err, "CancelReplaceSwapBatchOrders must not error")
	assert.Empty(t, result, "CancelReplaceSwapBatchOrders should decode an empty reply")

	swapTradeSkipIfCannotManipulateOrders(t)
	result, err = e.CancelReplaceSwapBatchOrders(t.Context(), []CancelReplaceSwapOrderRequest{{
		Symbol: currency.NewBTCUSDT(), Type: OrderTypeLimit, Side: SideBuy, PositionSide: PositionSideLong, Price: 38000, Quantity: 1,
		TakeProfit:        &SwapTakeProfitStopLossRequest{Type: OrderTypeTakeProfit, Quantity: 1, StopPrice: 45000, Price: 45000},
		StopLoss:          &SwapTakeProfitStopLossRequest{Type: OrderTypeStop, Quantity: 1, StopPrice: 37000, Price: 37000},
		CancelReplaceMode: CancelReplaceModeAllowFailure,
		CancelOrderID:     1753337028434464768,
	}})
	require.NoError(t, err, "CancelReplaceSwapBatchOrders must not error")
	if mockTests {
		exp := []SwapCancelReplaceOrderResponse{{
			CancelResult: true,
			CancelResponse: SwapCancelReplaceCancelResponse{
				CancelOrderID: 1753337028434464800,
				SwapOrderResponse: SwapOrderResponse{
					Symbol:           "BTC-USDT",
					OrderID:          1753337028434464800,
					Side:             SideBuy,
					PositionSide:     PositionSideLong,
					Type:             OrderTypeLimit,
					OriginalQuantity: 10,
					Price:            38000,
					Status:           "PENDING",
					Time:             types.Time(time.UnixMilli(1706863098000)),
					UpdateTime:       types.Time(time.UnixMilli(1706863097000)),
					Leverage:         "15X",
					WorkingType:      WorkingTypeMarkPrice,
				},
			},
			ReplaceResult: true,
			NewOrderResponse: SwapCancelReplaceNewOrderResponse{
				OrderID:      1753337098747777000,
				Symbol:       "BTC-USDT",
				PositionSide: PositionSideLong,
				Side:         SideBuy,
				Type:         OrderTypeLimit,
				Price:        38000,
				Quantity:     1,
				WorkingType:  WorkingTypeMarkPrice,
				TimeInForce:  TimeInForceGoodTillCancel,
				StopLoss:     `{"type": "STOP", "quantity": 1, "stopPrice": 37000, "price": 37000}`,
				TakeProfit:   `{"type": "TAKE_PROFIT", "quantity": 1, "stopPrice": 45000, "price": 45000}`,
			},
		}}
		assert.Equal(t, exp, result, "CancelReplaceSwapBatchOrders should decode the documented example")
	}
}

func TestCancelAllSwapOrdersAfter(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllSwapOrdersAfter(t.Context(), "", 10*time.Second)
	require.ErrorIs(t, err, errSwapCancelAllAfterTypeRequired, "CancelAllSwapOrdersAfter must reject an empty request type")

	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, url.Values{"type": {"CLOSE"}, "timeOut": {"120"}}, swapTradeBusinessParams(r), "the timeout should be sent in whole seconds")
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"triggerTime":0,"status":"CLOSED","note":""}}`)
	})
	result, err := ex.CancelAllSwapOrdersAfter(t.Context(), CancelAllAfterTypeClose, 120*time.Second+time.Millisecond)
	require.NoError(t, err, "CancelAllSwapOrdersAfter must not error")
	assert.Equal(t, &SwapCancelAllAfterResponse{Status: "CLOSED"}, result, "CancelAllSwapOrdersAfter should decode a closed countdown")

	swapTradeSkipIfCannotManipulateOrders(t)
	result, err = e.CancelAllSwapOrdersAfter(t.Context(), CancelAllAfterTypeActivate, 10*time.Second)
	require.NoError(t, err, "CancelAllSwapOrdersAfter must not error")
	if !mockTests {
		_, err = e.CancelAllSwapOrdersAfter(t.Context(), CancelAllAfterTypeClose, 10*time.Second)
		require.NoError(t, err, "CancelAllSwapOrdersAfter must stop the countdown before it cancels the account's orders")
		return
	}
	exp := &SwapCancelAllAfterResponse{
		TriggerTime: types.Time(time.Unix(1710389137, 0)),
		Status:      "ACTIVATED",
		Note:        "All your spot pending orders will be closed automatically at 2024-03-14 04:05:37 UTC(+0),before that you can cancel the timer, or extend triggerTime time by this request",
	}
	assert.Equal(t, exp, result, "CancelAllSwapOrdersAfter should decode the documented example")
}

func TestCloseSwapPosition(t *testing.T) {
	t.Parallel()
	_, err := e.CloseSwapPosition(t.Context(), 0)
	require.ErrorIs(t, err, errSwapPositionIDRequired, "CloseSwapPosition must reject an empty position ID")

	swapTradeSkipIfCannotManipulateOrders(t)
	result, err := e.CloseSwapPosition(t.Context(), 1769649551460794368)
	require.NoError(t, err, "CloseSwapPosition must not error")
	if mockTests {
		exp := &CloseSwapPositionResponse{
			OrderID:          1769649628749234200,
			PositionID:       1769649551460794368,
			Symbol:           "BTC-USDT",
			Side:             "Ask",
			Type:             "Market",
			PositionSide:     PositionSideBoth,
			OriginalQuantity: 1,
		}
		assert.Equal(t, exp, result, "CloseSwapPosition should decode the documented example")
	}
}

func TestGetSwapAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapAllOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSwapAllOrders must reject a nil request")
	_, err = e.GetSwapAllOrders(t.Context(), &SwapAllOrdersRequest{StartTime: time.UnixMilli(2), EndTime: time.UnixMilli(1)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSwapAllOrders must reject a reversed time range")
	_, err = new(Exchange).GetSwapAllOrders(t.Context(), &SwapAllOrdersRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSwapAllOrders must return the symbol formatting error")

	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, url.Values{"orderId": {"7"}}, swapTradeBusinessParams(r), "only the filters given should be sent")
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"orders":[]}}`)
	})
	_, err = ex.GetSwapAllOrders(t.Context(), &SwapAllOrdersRequest{OrderID: 7})
	require.NoError(t, err, "GetSwapAllOrders must not error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSwapAllOrders(t.Context(), &SwapAllOrdersRequest{
		Symbol: currency.NewPair(currency.NewCode("PYTH"), currency.USDT), StartTime: time.UnixMilli(1702688795000), EndTime: time.UnixMilli(1702731995000), Limit: 500,
	})
	require.NoError(t, err, "GetSwapAllOrders must not error")
	if mockTests {
		exp := []SwapOrderResponse{{
			Symbol:           "PYTH-USDT",
			OrderID:          1736007506620112100,
			Side:             SideSell,
			PositionSide:     PositionSideShort,
			Type:             OrderTypeLimit,
			OriginalQuantity: 33,
			Price:            0.3916,
			ExecutedQuantity: 33,
			AveragePrice:     0.3916,
			CumulativeQuote:  13,
			Commission:       -0.002585,
			Status:           "FILLED",
			Time:             types.Time(time.UnixMilli(1702731418000)),
			UpdateTime:       types.Time(time.UnixMilli(1702731470000)),
			Leverage:         "15X",
			TakeProfit:       SwapTakeProfitStopLossResponse{Type: OrderTypeTakeProfit},
			StopLoss:         SwapTakeProfitStopLossResponse{Type: OrderTypeStop},
			WorkingType:      WorkingTypeMarkPrice,
			StopGuaranteed:   StopGuaranteedFalse,
			TriggerOrderID:   1736012449498123500,
			IsTWAP:           true,
			MainOrderID:      "21312431241234",
		}}
		assert.Equal(t, exp, result, "GetSwapAllOrders should decode the documented example")
	}
}

func TestGetSwapMaintenanceMarginRatio(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapMaintenanceMarginRatio(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSwapMaintenanceMarginRatio must reject an empty symbol")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSwapMaintenanceMarginRatio(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "GetSwapMaintenanceMarginRatio must not error")
	if mockTests {
		exp := []SwapMaintenanceMarginRatioResponse{
			{Tier: "Tier 1", Symbol: "BTC-USDT", MaxPositionValue: 150000, MaintenanceMarginRatio: 0.0038},
			{Tier: "Tier 2", Symbol: "BTC-USDT", MinPositionValue: 150000, MaxPositionValue: 900000, MaintenanceMarginRatio: 0.004, MaintenanceAmount: 30},
		}
		assert.Equal(t, exp, result, "GetSwapMaintenanceMarginRatio should decode the documented example")
	} else {
		assert.NotEmpty(t, result, "GetSwapMaintenanceMarginRatio should return tiers")
	}
}

func TestGetSwapHistoricalTransactionDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapHistoricalTransactionDetails(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSwapHistoricalTransactionDetails must reject a nil request")
	_, err = e.GetSwapHistoricalTransactionDetails(t.Context(), &SwapHistoricalTransactionDetailsRequest{StartTime: time.UnixMilli(2), EndTime: time.UnixMilli(1)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSwapHistoricalTransactionDetails must reject a reversed time range")
	_, err = new(Exchange).GetSwapHistoricalTransactionDetails(t.Context(), &SwapHistoricalTransactionDetailsRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSwapHistoricalTransactionDetails must return the symbol formatting error")

	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, url.Values{"endTs": {"2"}, "orderId": {"9"}}, swapTradeBusinessParams(r), "an end time alone should be sent")
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"fill_history_orders":[],"total":0}}`)
	})
	result, err := ex.GetSwapHistoricalTransactionDetails(t.Context(), &SwapHistoricalTransactionDetailsRequest{EndTime: time.UnixMilli(2), OrderID: 9})
	require.NoError(t, err, "GetSwapHistoricalTransactionDetails must not error")
	assert.Equal(t, &SwapHistoricalTransactionDetailsResponse{FillHistoryOrders: []SwapHistoricalTransactionDetailResponse{}}, result, "GetSwapHistoricalTransactionDetails should decode an empty page")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err = e.GetSwapHistoricalTransactionDetails(t.Context(), &SwapHistoricalTransactionDetailsRequest{
		Symbol: currency.NewPair(currency.BTC, currency.USDC), Currency: currency.USDC, PageIndex: 1, PageSize: 50, StartTime: time.UnixMilli(1785503622606), EndTime: time.UnixMilli(1786108422606),
	})
	require.NoError(t, err, "GetSwapHistoricalTransactionDetails must not error")
	if mockTests {
		require.Len(t, result.FillHistoryOrders, 1, "GetSwapHistoricalTransactionDetails must return the documented fill")
		// filledTime carries BingX's +08:00 offset, which a time.Time comparison sees as a different location
		result.FillHistoryOrders[0].FilledTime = result.FillHistoryOrders[0].FilledTime.UTC()
		exp := &SwapHistoricalTransactionDetailsResponse{
			FillHistoryOrders: []SwapHistoricalTransactionDetailResponse{{
				Symbol:          "BTC-USDC",
				Quantity:        0.0001,
				QuoteQuantity:   7.5586,
				Role:            "taker",
				Commission:      -0.003779,
				CommissionAsset: currency.USDC,
				Price:           75586.05,
				OrderID:         2085388523196895233,
				TradeID:         "1110873278",
				FilledTime:      time.Date(2026, 8, 6, 15, 32, 23, 0, time.UTC),
				Side:            SideBuy,
				PositionSide:    PositionSideLong,
			}},
			Total: 17,
		}
		assert.Equal(t, exp, result, "GetSwapHistoricalTransactionDetails should decode the documented example")
	}
}

func TestGetSwapPositionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapPositionHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSwapPositionHistory must reject a nil request")
	arg := &SwapPositionHistoryRequest{}
	_, err = e.GetSwapPositionHistory(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSwapPositionHistory must reject an empty symbol")
	arg.Symbol = currency.NewPair(currency.BTC, currency.USDC)
	arg.EndTime = time.UnixMilli(1703001599000)
	_, err = e.GetSwapPositionHistory(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrDateUnset, "GetSwapPositionHistory must reject a missing start time")
	arg.StartTime = time.UnixMilli(1703001599001)
	_, err = e.GetSwapPositionHistory(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSwapPositionHistory must reject a reversed time range")
	arg.StartTime = time.UnixMilli(1700409600000)
	arg.Currency = currency.USDC
	arg.PageSize = 20

	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		exp := url.Values{"symbol": {"BTC-USDT"}, "startTime": {"1"}, "endTime": {"2"}, "positionId": {"3"}, "pageIndex": {"4"}}
		assert.Equal(t, exp, swapTradeBusinessParams(r), "every filter given should be sent")
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":[]}`)
	})
	_, err = ex.GetSwapPositionHistory(t.Context(), &SwapPositionHistoryRequest{Symbol: currency.NewBTCUSDT(), StartTime: time.UnixMilli(1), EndTime: time.UnixMilli(2), PositionID: 3, PageIndex: 4})
	require.NoError(t, err, "GetSwapPositionHistory must not error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSwapPositionHistory(t.Context(), arg)
	require.NoError(t, err, "GetSwapPositionHistory must not error")
	if mockTests {
		exp := []SwapPositionHistoryResponse{{
			PositionID:          "180870089***590996",
			Symbol:              "BTC-USDC",
			PositionSide:        PositionSideLong,
			OpenTime:            types.Time(time.UnixMilli(1720062873000)),
			UpdateTime:          types.Time(time.UnixMilli(1720062878000)),
			AveragePrice:        58942.31,
			AverageClosePrice:   58930,
			RealizedProfit:      -0.04,
			NetProfit:           -0.16,
			PositionAmount:      33,
			ClosePositionAmount: 33,
			Leverage:            20,
			CloseAllPositions:   true,
			PositionCommission:  -0.11669358690000001,
			TotalFunding:        0.00000000000000001388,
		}}
		assert.Equal(t, exp, result, "GetSwapPositionHistory should decode the documented example")
	}
}

func TestGetSwapIsolatedMarginChangeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapIsolatedMarginChangeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSwapIsolatedMarginChangeHistory must reject a nil request")
	arg := &SwapIsolatedMarginChangeHistoryRequest{}
	_, err = e.GetSwapIsolatedMarginChangeHistory(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSwapIsolatedMarginChangeHistory must reject an empty symbol")
	arg.Symbol = currency.NewPair(currency.BNB, currency.USDT)
	_, err = e.GetSwapIsolatedMarginChangeHistory(t.Context(), arg)
	require.ErrorIs(t, err, errSwapPositionIDRequired, "GetSwapIsolatedMarginChangeHistory must reject an empty position ID")
	arg.PositionID = 1847596444958068736
	arg.StartTime = time.UnixMilli(1728722649000)
	_, err = e.GetSwapIsolatedMarginChangeHistory(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrDateUnset, "GetSwapIsolatedMarginChangeHistory must reject a missing end time")
	arg.EndTime = time.UnixMilli(1)
	_, err = e.GetSwapIsolatedMarginChangeHistory(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSwapIsolatedMarginChangeHistory must reject a reversed time range")
	arg.EndTime = time.UnixMilli(1729336359406)
	arg.PageIndex = 1
	arg.PageSize = 2

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSwapIsolatedMarginChangeHistory(t.Context(), arg)
	require.NoError(t, err, "GetSwapIsolatedMarginChangeHistory must not error")
	if mockTests {
		exp := &SwapIsolatedMarginChangeHistoryResponse{
			Records: []SwapIsolatedMarginChangeResponse{{
				Symbol:            "BTC-USDT",
				PositionID:        1847596444958068736,
				ChangeReason:      "OpenPosition",
				MarginChange:      7586.46841066,
				MarginAfterChange: 7586.46841066,
				Time:              types.Time(time.UnixMilli(1729336294000)),
			}},
			Total: 4,
		}
		assert.Equal(t, exp, result, "GetSwapIsolatedMarginChangeHistory should decode the documented example")
	}
}

func TestApplySwapVST(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, swapTradeBusinessParams(r), "no adjustment should be sent when none is given")
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"adjustType":"1","amount":"1"}}`)
	})
	result, err := ex.ApplySwapVST(t.Context(), "", 0)
	require.NoError(t, err, "ApplySwapVST must not error")
	assert.Equal(t, &ApplySwapVSTResponse{AdjustType: VSTAdjustTypeDecrease, Amount: 1}, result, "ApplySwapVST should decode the reply")

	t.Run("adjustment", func(t *testing.T) {
		t.Parallel()
		for _, adjustType := range []VSTAdjustType{VSTAdjustTypeIncrease, VSTAdjustTypeDecrease} {
			ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method, "method should be POST")
				assert.Equal(t, "/openApi/swap/v2/trade/getVst", r.URL.Path, "path should be the Apply VST path")
				assert.Equalf(t, url.Values{"adjustType": {string(adjustType)}, "amount": {"1000"}}, swapTradeBusinessParams(r), "adjustment %s and its amount should be sent", adjustType)
				_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"adjustType":"`+string(adjustType)+`","amount":"1000"}}`)
			})
			got, err := ex.ApplySwapVST(t.Context(), adjustType, 1000)
			require.NoErrorf(t, err, "ApplySwapVST must not error for adjustment %s", adjustType)
			assert.Equalf(t, &ApplySwapVSTResponse{AdjustType: adjustType, Amount: 1000}, got, "ApplySwapVST should decode adjustment %s", adjustType)
		}
	})

	if !mockTests {
		ex := newWrapperTestExchange(t)
		wrapperSkipIfCannotManipulateOrders(t, ex)
		require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), "https://open-api-vst.bingx.com"), "SetRunningURL must not error")
		for _, adjustType := range []VSTAdjustType{VSTAdjustTypeIncrease, VSTAdjustTypeDecrease} {
			result, err = ex.ApplySwapVST(t.Context(), adjustType, 1000)
			require.NoErrorf(t, err, "ApplySwapVST must not error for adjustment %s", adjustType)
			assert.Equalf(t, &ApplySwapVSTResponse{AdjustType: adjustType, Amount: 1000}, result, "ApplySwapVST should make adjustment %s on the demo trading host", adjustType)
		}
		return
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err = e.ApplySwapVST(t.Context(), VSTAdjustTypeIncrease, 500000)
	require.NoError(t, err, "ApplySwapVST must not error")
	assert.Equal(t, &ApplySwapVSTResponse{AdjustType: VSTAdjustTypeIncrease, Amount: 500000}, result, "ApplySwapVST should decode the documented example")
}

func TestPlaceSwapTWAPOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceSwapTWAPOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceSwapTWAPOrder must reject a nil request")
	arg := &SwapTWAPOrderRequest{}
	_, err = e.PlaceSwapTWAPOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "PlaceSwapTWAPOrder must reject an empty symbol")
	arg.Symbol = currency.NewPair(currency.BNB, currency.USDT)
	_, err = e.PlaceSwapTWAPOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "PlaceSwapTWAPOrder must reject an empty side")
	arg.Side = SideBuy
	_, err = e.PlaceSwapTWAPOrder(t.Context(), arg)
	require.ErrorIs(t, err, errPositionSideRequired, "PlaceSwapTWAPOrder must reject an empty position side")
	arg.PositionSide = PositionSideLong
	_, err = e.PlaceSwapTWAPOrder(t.Context(), arg)
	require.ErrorIs(t, err, errSwapTWAPPriceTypeRequired, "PlaceSwapTWAPOrder must reject an empty price type")
	arg.PriceType = TWAPPriceTypeConstant
	_, err = e.PlaceSwapTWAPOrder(t.Context(), arg)
	require.ErrorIs(t, err, errSwapTWAPPriceVarianceRequired, "PlaceSwapTWAPOrder must reject an empty price variance")
	arg.PriceVariance = 2000
	_, err = e.PlaceSwapTWAPOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "PlaceSwapTWAPOrder must reject an empty trigger price")
	arg.TriggerPrice = 68000
	arg.Interval = time.Millisecond
	_, err = e.PlaceSwapTWAPOrder(t.Context(), arg)
	require.ErrorIs(t, err, errSwapTWAPIntervalRequired, "PlaceSwapTWAPOrder must reject an interval under a second")
	arg.Interval = 8 * time.Second
	_, err = e.PlaceSwapTWAPOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "PlaceSwapTWAPOrder must reject an empty amount per order")
	require.ErrorContains(t, err, "amount per order", "PlaceSwapTWAPOrder must name the empty amount")
	arg.AmountPerOrder = 0.111
	_, err = e.PlaceSwapTWAPOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "PlaceSwapTWAPOrder must reject an empty total amount")
	require.ErrorContains(t, err, "total amount", "PlaceSwapTWAPOrder must name the empty amount")
	arg.TotalAmount = 0.511

	ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"mainOrderId":"12312435134"}}`)
	})
	result, err := ex.PlaceSwapTWAPOrder(t.Context(), arg)
	require.NoError(t, err, "PlaceSwapTWAPOrder must not error")
	assert.Equal(t, &PlaceSwapTWAPOrderResponse{MainOrderID: "12312435134"}, result, "PlaceSwapTWAPOrder should decode the documented field")

	swapTradeSkipIfCannotManipulateOrders(t)
	result, err = e.PlaceSwapTWAPOrder(t.Context(), arg)
	require.NoError(t, err, "PlaceSwapTWAPOrder must not error")
	if mockTests {
		// The documented example repeats Isolated Margin Change History's reply, which holds no mainOrderId
		assert.Equal(t, &PlaceSwapTWAPOrderResponse{}, result, "PlaceSwapTWAPOrder should decode the documented example")
	}
}

func TestGetSwapTWAPEntrustedOrders(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).GetSwapTWAPEntrustedOrders(t.Context(), currency.NewPair(currency.BNB, currency.USDT))
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSwapTWAPEntrustedOrders must return the symbol formatting error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSwapTWAPEntrustedOrders(t.Context(), currency.NewPair(currency.BNB, currency.USDT))
	require.NoError(t, err, "GetSwapTWAPEntrustedOrders must not error")
	if mockTests {
		assert.Equal(t, &SwapTWAPOrdersResponse{List: []SwapTWAPOrderResponse{swapTradeTWAPOrderExample()}, Total: 1}, result, "GetSwapTWAPEntrustedOrders should decode the documented example")
	}
}

// swapTradeTWAPOrderExample returns the TWAP order of the documented TWAP examples
func swapTradeTWAPOrderExample() SwapTWAPOrderResponse {
	return SwapTWAPOrderResponse{
		Symbol:           "BNB-USDT",
		Side:             SideBuy,
		PositionSide:     PositionSideLong,
		PriceType:        TWAPPriceTypeConstant,
		PriceVariance:    2000,
		TriggerPrice:     68000,
		Interval:         8,
		AmountPerOrder:   0.111,
		TotalAmount:      0.511,
		OrderStatus:      "Running",
		ExecutedQuantity: 0.1,
		Duration:         800,
		MaxDuration:      9000,
		CreatedTime:      types.Time(time.UnixMilli(1702731661854)),
		UpdateTime:       types.Time(time.UnixMilli(1702731661854)),
	}
}

func TestGetSwapTWAPHistoricalOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapTWAPHistoricalOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSwapTWAPHistoricalOrders must reject a nil request")
	arg := &SwapTWAPHistoricalOrdersRequest{EndTime: time.UnixMilli(1702738661854)}
	_, err = e.GetSwapTWAPHistoricalOrders(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrDateUnset, "GetSwapTWAPHistoricalOrders must reject a missing start time")
	arg.StartTime = time.UnixMilli(1702738661855)
	_, err = e.GetSwapTWAPHistoricalOrders(t.Context(), arg)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSwapTWAPHistoricalOrders must reject a reversed time range")
	arg.StartTime = time.UnixMilli(1702731661854)
	arg.Symbol = currency.NewPair(currency.BNB, currency.USDT)
	arg.PageIndex = 1
	arg.PageSize = 100
	_, err = new(Exchange).GetSwapTWAPHistoricalOrders(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSwapTWAPHistoricalOrders must return the symbol formatting error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSwapTWAPHistoricalOrders(t.Context(), arg)
	require.NoError(t, err, "GetSwapTWAPHistoricalOrders must not error")
	if mockTests {
		assert.Equal(t, &SwapTWAPOrdersResponse{List: []SwapTWAPOrderResponse{swapTradeTWAPOrderExample()}, Total: 1}, result, "GetSwapTWAPHistoricalOrders should decode the documented example")
	}
}

func TestGetSwapTWAPOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetSwapTWAPOrderDetails(t.Context(), "")
	require.ErrorIs(t, err, errSwapMainOrderIDRequired, "GetSwapTWAPOrderDetails must reject an empty main order ID")
	ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, swapTradeEmptyDataReply)
	})
	_, err = ex.GetSwapTWAPOrderDetails(t.Context(), "12312435134")
	require.ErrorIs(t, err, common.ErrNoResponse, "GetSwapTWAPOrderDetails must error for a reply without an order")
	failing := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"code":100500,"msg":"busy"}`)
	})
	_, err = failing.GetSwapTWAPOrderDetails(t.Context(), "12312435134")
	require.ErrorIs(t, err, errAPIResponse, "GetSwapTWAPOrderDetails must return the request's error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSwapTWAPOrderDetails(t.Context(), "12312435134")
	require.NoError(t, err, "GetSwapTWAPOrderDetails must not error")
	if mockTests {
		exp := swapTradeTWAPOrderExample()
		assert.Equal(t, &exp, result, "GetSwapTWAPOrderDetails should decode the documented example")
	}
}

func TestCancelSwapTWAPOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelSwapTWAPOrder(t.Context(), "")
	require.ErrorIs(t, err, errSwapMainOrderIDRequired, "CancelSwapTWAPOrder must reject an empty main order ID")

	swapTradeSkipIfCannotManipulateOrders(t)
	result, err := e.CancelSwapTWAPOrder(t.Context(), "12312435134")
	require.NoError(t, err, "CancelSwapTWAPOrder must not error")
	if mockTests {
		exp := swapTradeTWAPOrderExample()
		assert.Equal(t, &exp, result, "CancelSwapTWAPOrder should decode the documented example")
	}
}

func TestSwitchSwapMultiAssetsMode(t *testing.T) {
	t.Parallel()
	_, err := e.SwitchSwapMultiAssetsMode(t.Context(), "")
	require.ErrorIs(t, err, errSwapAssetModeRequired, "SwitchSwapMultiAssetsMode must reject an empty asset mode")

	swapTradeSkipIfCannotManipulateOrders(t)
	result, err := e.SwitchSwapMultiAssetsMode(t.Context(), AssetModeMultiAssets)
	require.NoError(t, err, "SwitchSwapMultiAssetsMode must not error")
	if mockTests {
		assert.Equal(t, &SwapMultiAssetsModeResponse{AssetMode: AssetModeMultiAssets}, result, "SwitchSwapMultiAssetsMode should decode the documented example")
	}
}

func TestGetSwapMultiAssetsMode(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, swapTradeEmptyDataReply)
	})
	_, err := ex.GetSwapMultiAssetsMode(t.Context())
	require.ErrorIs(t, err, common.ErrNoResponse, "GetSwapMultiAssetsMode must error for a reply without an asset mode")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSwapMultiAssetsMode(t.Context())
	require.NoError(t, err, "GetSwapMultiAssetsMode must not error")
	if mockTests {
		assert.Equal(t, &SwapMultiAssetsModeResponse{AssetMode: AssetModeMultiAssets}, result, "GetSwapMultiAssetsMode should decode the documented example")
	} else {
		assert.NotEmpty(t, result.AssetMode, "GetSwapMultiAssetsMode should return an asset mode")
	}
}

func TestGetSwapMultiAssetsRules(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSwapMultiAssetsRules(t.Context())
	require.NoError(t, err, "GetSwapMultiAssetsRules must not error")
	if mockTests {
		exp := []SwapMultiAssetsRuleResponse{
			{MarginAssets: currency.USDT, LTV: "99.99%", CollateralValueRatio: "99.99%", IndexPrice: 1.00069818},
			{MarginAssets: currency.USDC, LTV: "99.99%", CollateralValueRatio: "99.99%", IndexPrice: 0.9998772499},
			{MarginAssets: currency.TRX, LTV: "85.00%", CollateralValueRatio: "95.00%", MaxTransfer: 300000, IndexPrice: 0.1778813333},
			{MarginAssets: currency.DOGE, LTV: "95.00%", CollateralValueRatio: "95.00%", MaxTransfer: 300000, IndexPrice: 0.3891840533},
			{MarginAssets: currency.BNB, LTV: "90.00%", CollateralValueRatio: "95.00%", MaxTransfer: 160, IndexPrice: 622.95847921},
			{MarginAssets: currency.DOT, LTV: "90.00%", CollateralValueRatio: "95.00%", MaxTransfer: 12000, IndexPrice: 5.0431331628},
			{MarginAssets: currency.BTC, LTV: "95.00%", CollateralValueRatio: "95.00%", MaxTransfer: 10, IndexPrice: 90230.3058075903},
			{MarginAssets: currency.ETH, LTV: "95.00%", CollateralValueRatio: "95.00%", MaxTransfer: 100, IndexPrice: 3214.427968282},
		}
		assert.Equal(t, exp, result, "GetSwapMultiAssetsRules should decode the documented example")
	} else {
		assert.NotEmpty(t, result, "GetSwapMultiAssetsRules should return the margin assets")
	}
}

func TestGetSwapMultiAssetsMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSwapMultiAssetsMargin(t.Context())
	require.NoError(t, err, "GetSwapMultiAssetsMargin must not error")
	if mockTests {
		exp := []SwapMultiAssetsMarginResponse{
			{Currency: currency.USDC},
			{Currency: currency.USDT, TotalAmount: 6.8988632, AvailableTransfer: 6.8988632, LatestMortgageAmount: 6.8988632},
			{Currency: currency.BTC, TotalAmount: 6.8988632, AvailableTransfer: 6.8988632, LatestMortgageAmount: 6.8988632},
			{Currency: currency.ETH, TotalAmount: 6.8988632, AvailableTransfer: 6.8988632, LatestMortgageAmount: 6.8988632},
		}
		assert.Equal(t, exp, result, "GetSwapMultiAssetsMargin should decode the documented example")
	}
}

func TestReverseSwapPosition(t *testing.T) {
	t.Parallel()
	_, err := e.ReverseSwapPosition(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ReverseSwapPosition must reject a nil request")
	arg := &ReverseSwapPositionRequest{}
	_, err = e.ReverseSwapPosition(t.Context(), arg)
	require.ErrorIs(t, err, errSwapReverseTypeRequired, "ReverseSwapPosition must reject an empty reverse type")
	arg.Type = ReverseTypeReverse
	_, err = e.ReverseSwapPosition(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "ReverseSwapPosition must reject an empty symbol")
	arg.Symbol = currency.NewBTCUSDT()
	arg.PositionID = 1858503955839975424

	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		exp := url.Values{"type": {"TriggerReverse"}, "symbol": {"BTC-USDT"}, "triggerPrice": {"95000.5"}, "workingType": {"MARK_PRICE"}}
		assert.Equal(t, exp, swapTradeBusinessParams(r), "a planned reversal should send its trigger")
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"type":"TriggerReverse","symbol":"BTC-USDT","realisedProfit":"1.5","liquidationPrice":95000.5}}`)
	})
	result, err := ex.ReverseSwapPosition(t.Context(), &ReverseSwapPositionRequest{Type: ReverseTypeTriggerReverse, Symbol: currency.NewBTCUSDT(), TriggerPrice: 95000.5, WorkingType: WorkingTypeMarkPrice})
	require.NoError(t, err, "ReverseSwapPosition must not error")
	assert.Equal(t, &ReverseSwapPositionResponse{Type: ReverseTypeTriggerReverse, Symbol: "BTC-USDT", RealisedProfit: 1.5, LiquidationPrice: 95000.5}, result, "ReverseSwapPosition should decode the documented spelling and a bare liquidation price")

	swapTradeSkipIfCannotManipulateOrders(t)
	result, err = e.ReverseSwapPosition(t.Context(), arg)
	require.NoError(t, err, "ReverseSwapPosition must not error")
	if mockTests {
		exp := &ReverseSwapPositionResponse{
			Type:             ReverseTypeReverse,
			PositionID:       1858503955839975424,
			NewPositionID:    1858504148832485376,
			Symbol:           "BTC-USDT",
			PositionSide:     "Long",
			PositionAmount:   10000,
			AvailableAmount:  10000,
			UnrealizedProfit: 4.8768,
			RealizedProfit:   -26.9683,
			InitialMargin:    8989.4302,
			Margin:           8994.307,
			LiquidationPrice: 98138.33,
			AveragePrice:     89894.3,
			Leverage:         "10X",
			PositionValue:    89899.18,
			MarkPrice:        89899.1784855,
			RiskRate:         100,
			PNLRatio:         0.05,
			UpdateTime:       types.Time(time.UnixMilli(1731936893082)),
		}
		assert.Equal(t, exp, result, "ReverseSwapPosition should decode the documented example")
	}
}

func TestSetSwapAutoAddMargin(t *testing.T) {
	t.Parallel()
	_, err := e.SetSwapAutoAddMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetSwapAutoAddMargin must reject a nil request")
	arg := &SwapAutoAddMarginRequest{}
	_, err = e.SetSwapAutoAddMargin(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "SetSwapAutoAddMargin must reject an empty symbol")
	arg.Symbol = currency.NewBTCUSDT()
	_, err = e.SetSwapAutoAddMargin(t.Context(), arg)
	require.ErrorIs(t, err, errSwapPositionIDRequired, "SetSwapAutoAddMargin must reject an empty position ID")
	arg.PositionID = 1868671302923976704

	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		exp := url.Values{"symbol": {"BTC-USDT"}, "positionId": {"1868671302923976704"}, "functionSwitch": {"false"}}
		assert.Equal(t, exp, swapTradeBusinessParams(r), "turning the feature off should send false without an amount")
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"symbol":"BTC-USDT","positionId":1868671302923976704,"functionSwitch":"false","amount":"0"}}`)
	})
	result, err := ex.SetSwapAutoAddMargin(t.Context(), arg)
	require.NoError(t, err, "SetSwapAutoAddMargin must not error")
	assert.Equal(t, &SwapAutoAddMarginResponse{Symbol: "BTC-USDT", PositionID: 1868671302923976704}, result, "SetSwapAutoAddMargin should decode the documented fields")
	arg.FunctionSwitch = true
	arg.Amount = 130

	swapTradeSkipIfCannotManipulateOrders(t)
	result, err = e.SetSwapAutoAddMargin(t.Context(), arg)
	require.NoError(t, err, "SetSwapAutoAddMargin must not error")
	if mockTests {
		// The documented example repeats Modify Isolated Position Margin's reply, of which only amount is a field here
		assert.Equal(t, &SwapAutoAddMarginResponse{Amount: 3}, result, "SetSwapAutoAddMargin should decode the documented example")
	}
}

func TestStopGuaranteedUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		raw string
		exp StopGuaranteed
	}{
		{`true`, StopGuaranteedTrue},
		{`false`, StopGuaranteedFalse},
		{`"true"`, StopGuaranteedTrue},
		{`"false"`, StopGuaranteedFalse},
		{`"cutfee"`, StopGuaranteedCutFee},
		{`""`, ""},
	} {
		var s StopGuaranteed
		require.NoErrorf(t, json.Unmarshal([]byte(tc.raw), &s), "Unmarshal must not error for %s", tc.raw)
		assert.Equalf(t, tc.exp, s, "Unmarshal should decode %s", tc.raw)
		direct := StopGuaranteed("unset")
		require.NoErrorf(t, direct.UnmarshalJSON([]byte(tc.raw)), "UnmarshalJSON must not error for %s", tc.raw)
		assert.Equalf(t, tc.exp, direct, "UnmarshalJSON should decode %s", tc.raw)
	}
	s := StopGuaranteedCutFee
	require.NoError(t, json.Unmarshal([]byte(`null`), &s), "Unmarshal must not error for null")
	assert.Equal(t, StopGuaranteedCutFee, s, "Unmarshal should leave the value unchanged for null")
	require.NoError(t, s.UnmarshalJSON([]byte(`null`)), "UnmarshalJSON must not error for null")
	assert.Equal(t, StopGuaranteedCutFee, s, "UnmarshalJSON should leave the value unchanged for null")
	assert.ErrorIs(t, json.Unmarshal([]byte(`1`), &s), errInvalidStopGuaranteed, "Unmarshal should reject a number")
	assert.ErrorIs(t, s.UnmarshalJSON([]byte(`1`)), errInvalidStopGuaranteed, "UnmarshalJSON should reject a number")
}
