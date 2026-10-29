package kucoin

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

func TestPlaceUTAOrderRequestValidate(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, (*PlaceUTAOrderRequest)(nil).Validate(), common.ErrNilPointer, "Validate must reject a nil request")
	for _, tc := range []struct {
		name   string
		mutate func(*PlaceUTAOrderRequest)
		err    error
	}{
		{name: "valid", mutate: func(*PlaceUTAOrderRequest) {}},
		{name: "no trade type", mutate: func(r *PlaceUTAOrderRequest) { r.TradeType = "" }, err: errTradeTypeMissing},
		{name: "no symbol", mutate: func(r *PlaceUTAOrderRequest) { r.Symbol = "" }, err: currency.ErrSymbolStringEmpty},
		{name: "no side", mutate: func(r *PlaceUTAOrderRequest) { r.Side = "" }, err: order.ErrSideIsInvalid},
		{name: "no order type", mutate: func(r *PlaceUTAOrderRequest) { r.OrderType = "" }, err: order.ErrTypeIsInvalid},
		{name: "no size", mutate: func(r *PlaceUTAOrderRequest) { r.Size = 0 }, err: order.ErrAmountIsInvalid},
		{name: "negative size", mutate: func(r *PlaceUTAOrderRequest) { r.Size = -1 }, err: order.ErrAmountIsInvalid},
		{name: "no size unit", mutate: func(r *PlaceUTAOrderRequest) { r.SizeUnit = "" }, err: errSizeUnitRequired},
		{name: "futures without client order ID", mutate: func(r *PlaceUTAOrderRequest) { r.TradeType = "FUTURES" }, err: order.ErrClientOrderIDMustBeSet},
		{name: "margin without client order ID", mutate: func(r *PlaceUTAOrderRequest) { r.TradeType = "MARGIN" }, err: order.ErrClientOrderIDMustBeSet},
		{name: "futures with client order ID", mutate: func(r *PlaceUTAOrderRequest) {
			r.TradeType, r.ClientOrderID = "FUTURES", "12259e46-6da3-40e5-9944-782a9f6fa05d"
		}},
		{name: "trigger price without direction", mutate: func(r *PlaceUTAOrderRequest) { r.TriggerPrice = 66330 }, err: errTriggerDirectionAndPriceRequired},
		{name: "trigger direction without price", mutate: func(r *PlaceUTAOrderRequest) { r.TriggerDirection = "UP" }, err: errTriggerDirectionAndPriceRequired},
		{name: "conditional order", mutate: func(r *PlaceUTAOrderRequest) { r.TriggerDirection, r.TriggerPrice = "UP", 66330 }},
		{name: "conditional order with take profit", mutate: func(r *PlaceUTAOrderRequest) {
			r.TriggerDirection, r.TriggerPrice, r.TakeProfitTriggerPrice = "UP", 66330, 70000
		}, err: errConditionalOrderWithTPSL},
		{name: "conditional order with stop loss", mutate: func(r *PlaceUTAOrderRequest) {
			r.TriggerDirection, r.TriggerPrice, r.StopLossTriggerPrice = "UP", 66330, 60000
		}, err: errConditionalOrderWithTPSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := &PlaceUTAOrderRequest{TradeType: "SPOT", Symbol: "BTC-USDT", Side: "BUY", OrderType: "LIMIT", Size: 0.00001, SizeUnit: "BASECCY"}
			tc.mutate(r)
			assert.ErrorIs(t, r.Validate(), tc.err, "Validate should return the expected error")
		})
	}
}

func TestPlaceUTAOrderRequestPayload(t *testing.T) {
	t.Parallel()
	_, err := (*PlaceUTAOrderRequest)(nil).Payload()
	require.ErrorIs(t, err, common.ErrNilPointer, "Payload must reject a nil request")
	_, err = (&PlaceUTAOrderRequest{}).Payload()
	require.ErrorIs(t, err, errTradeTypeMissing, "Payload must validate the request")

	for _, tc := range []struct {
		name string
		req  *PlaceUTAOrderRequest
		exp  map[string]any
	}{
		{
			name: "conditional order",
			req: &PlaceUTAOrderRequest{
				TradeType: "SPOT", ClientOrderID: "273edcef-532d-46ba-9e0c-14a8da03ca9b", Symbol: "BTC-USDT", Side: "SELL",
				PositionSide: "BOTH", OrderType: "LIMIT", Size: 0.00001, SizeUnit: "BASECCY", Price: 66380, MarginMode: "CROSS",
				SelfTradePrevention: "CN", TimeInForce: "GTT", CancelAfter: 3600, PostOnly: true, Tags: "KuCoin API Service",
				TriggerDirection: "DOWN", TriggerPrice: 66340, TriggerPriceType: "TP",
			},
			exp: map[string]any{
				"tradeType": "SPOT", "clientOid": "273edcef-532d-46ba-9e0c-14a8da03ca9b", "symbol": "BTC-USDT", "side": "SELL",
				"positionSide": "BOTH", "orderType": "LIMIT", "size": "0.00001", "sizeUnit": "BASECCY", "price": "66380",
				"marginMode": "CROSS", "stp": "CN", "timeInForce": "GTT", "cancelAfter": uint64(3600), "postOnly": true,
				"tags": "KuCoin API Service", "triggerDirection": "DOWN", "triggerPrice": "66340", "triggerPriceType": "TP",
			},
		},
		{
			name: "take profit and stop loss order",
			req: &PlaceUTAOrderRequest{
				TradeType: "FUTURES", ClientOrderID: "07f70ced-d556-4120-b920-f5e74f72227a", Symbol: "XBTUSDTM", Side: "BUY",
				OrderType: "MARKET", Size: 1, SizeUnit: "UNIT", ReduceOnly: true, TakeProfitTriggerPrice: 64880,
				TakeProfitTriggerPriceType: "MP", StopLossTriggerPrice: 64800, StopLossTriggerPriceType: "IP", CloseOrder: true,
			},
			exp: map[string]any{
				"tradeType": "FUTURES", "clientOid": "07f70ced-d556-4120-b920-f5e74f72227a", "symbol": "XBTUSDTM", "side": "BUY",
				"orderType": "MARKET", "size": "1", "sizeUnit": "UNIT", "reduceOnly": true, "tpTriggerPrice": "64880",
				"tpTriggerPriceType": "MP", "slTriggerPrice": "64800", "slTriggerPriceType": "IP", "closeOrder": true,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body, err := tc.req.Payload()
			require.NoError(t, err, "Payload must not error")
			assert.Equal(t, tc.exp, body, "Payload should send every documented parameter that is set")
		})
	}
}

// utaLiveFuturesPrice returns a buy price for XBTUSDTM the given share of KuCoin's price limit below the best bid,
// rounded down to the tick size: inside the band KuCoin accepts orders in, but too far from the market to fill while a
// live test holds the order
func utaLiveFuturesPrice(t *testing.T, share float64) float64 {
	t.Helper()
	symbols, err := e.GetUTASymbol(t.Context(), "FUTURES", "XBTUSDTM")
	require.NoError(t, err, "GetUTASymbol must not error")
	require.NotNil(t, symbols, "GetUTASymbol must return a response")
	require.Len(t, symbols.List, 1, "GetUTASymbol must return the contract")
	tickers, err := e.GetUTATicker(t.Context(), "FUTURES", "XBTUSDTM")
	require.NoError(t, err, "GetUTATicker must not error")
	require.NotNil(t, tickers, "GetUTATicker must return a response")
	require.Len(t, tickers.List, 1, "GetUTATicker must return the contract's ticker")
	limit, tick := symbols.List[0].PriceLimitRatio.Float64(), symbols.List[0].TickSize.Decimal()
	if limit <= 0 || !tick.IsPositive() {
		t.Skip("XBTUSDTM has no price limit or tick size to place a resting order by")
	}
	price := tickers.List[0].BestBidPrice.Decimal().Mul(decimal.MustFromFloat(1 - limit*share)).Div(tick).Floor().Mul(tick)
	require.True(t, price.IsPositive(), "the resting price must be positive")
	return price.InexactFloat64()
}

// newUTALiveFuturesOrder returns a resting limit buy of one XBTUSDTM contract for a live order test, with a fresh
// client order ID
func newUTALiveFuturesOrder(t *testing.T) *PlaceUTAOrderRequest {
	t.Helper()
	return &PlaceUTAOrderRequest{
		TradeType:     "FUTURES",
		ClientOrderID: e.MessageID(),
		Symbol:        "XBTUSDTM",
		Side:          "BUY",
		OrderType:     "LIMIT",
		Size:          1,
		SizeUnit:      "UNIT",
		Price:         utaLiveFuturesPrice(t, 0.5),
		TimeInForce:   "GTC",
	}
}

// cancelUTALiveOrderOnFailure cancels a live test's order by its client order ID if the test fails, so that a test
// failing before it cancels its order leaves nothing resting
func cancelUTALiveOrderOnFailure(t *testing.T, r *PlaceUTAOrderRequest) {
	t.Helper()
	t.Cleanup(func() {
		if !t.Failed() {
			return
		}
		if _, err := e.CancelUTAOrder(context.WithoutCancel(t.Context()), &CancelUTAOrderRequest{TradeType: r.TradeType, Symbol: r.Symbol, ClientOrderID: r.ClientOrderID}); err != nil {
			t.Logf("Cancelling order %s after the test failed: %v", r.ClientOrderID, err)
		}
	})
}

// placeUTALiveOrder places a live test's order, which is cancelled if the test fails
func placeUTALiveOrder(t *testing.T, r *PlaceUTAOrderRequest) *UTAOrderResponse {
	t.Helper()
	cancelUTALiveOrderOnFailure(t, r)
	resp, err := e.PlaceUTAOrder(t.Context(), r)
	require.NoError(t, err, "PlaceUTAOrder must not error")
	require.NotNil(t, resp, "PlaceUTAOrder must return the order")
	require.NotEmpty(t, resp.OrderID, "PlaceUTAOrder must return the order ID")
	return resp
}

func TestPlaceUTAOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceUTAOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceUTAOrder must reject a nil request")
	_, err = e.PlaceUTAOrder(t.Context(), &PlaceUTAOrderRequest{})
	require.ErrorIs(t, err, errTradeTypeMissing, "PlaceUTAOrder must reject a request without a trade type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		req := newUTALiveFuturesOrder(t)
		placed := placeUTALiveOrder(t, req)
		_, err = e.CancelUTAOrder(t.Context(), &CancelUTAOrderRequest{TradeType: req.TradeType, Symbol: req.Symbol, OrderID: placed.OrderID})
		require.NoError(t, err, "CancelUTAOrder must cancel the test's order")
		return
	}
	result, err := e.PlaceUTAOrder(t.Context(), &PlaceUTAOrderRequest{
		TradeType: "SPOT", ClientOrderID: "273edcef-532d-46ba-9e0c-14a8da03ca9b", Symbol: "BTC-USDT", Side: "BUY", OrderType: "LIMIT",
		Size: 0.00001, SizeUnit: "BASECCY", Price: 66550, TimeInForce: "GTC", Tags: "KuCoin API Service",
	})
	require.NoError(t, err, "PlaceUTAOrder must not error")
	exp := &UTAOrderResponse{
		TradeType:     "SPOT",
		OrderID:       "385543068587233280",
		ClientOrderID: "273edcef-532d-46ba-9e0c-14a8da03ca9b",
		Timestamp:     types.Time(time.UnixMilli(1764733832617)),
	}
	assert.Equal(t, exp, result, "PlaceUTAOrder should decode every field")
}

func TestAmendUTAOrderRequestValidate(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, (*AmendUTAOrderRequest)(nil).Validate(), common.ErrNilPointer, "Validate must reject a nil request")
	for _, tc := range []struct {
		name string
		req  *AmendUTAOrderRequest
		err  error
	}{
		{name: "no symbol", req: &AmendUTAOrderRequest{OrderID: "461379758287044608"}, err: currency.ErrSymbolStringEmpty},
		{name: "no order IDs", req: &AmendUTAOrderRequest{Symbol: "XBTUSDTM"}, err: order.ErrOrderIDNotSet},
		{name: "order ID", req: &AmendUTAOrderRequest{Symbol: "XBTUSDTM", OrderID: "461379758287044608"}},
		{name: "client order ID", req: &AmendUTAOrderRequest{Symbol: "XBTUSDTM", ClientOrderID: "12259e46-6da3-40e5-9944-782a9f6fa05d"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, tc.req.Validate(), tc.err, "Validate should return the expected error")
		})
	}
}

func TestAmendUTAOrderRequestPayload(t *testing.T) {
	t.Parallel()
	_, err := (*AmendUTAOrderRequest)(nil).Payload()
	require.ErrorIs(t, err, common.ErrNilPointer, "Payload must reject a nil request")
	_, err = (&AmendUTAOrderRequest{}).Payload()
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "Payload must validate the request")

	body, err := (&AmendUTAOrderRequest{
		Symbol: "DOGSUSDTM", NewSize: 9, OrderID: "454538143408951297", ClientOrderID: "11", NewPrice: 0.0000433, SizeUnit: "UNIT",
		CancelOnFail: true, StopLossTriggerPrice: 0.00003, TakeProfitTriggerPrice: 0.000045, StopLossTriggerPriceType: "TP",
		TakeProfitTriggerPriceType: "MP",
	}).Payload()
	require.NoError(t, err, "Payload must not error")
	exp := map[string]any{
		"symbol": "DOGSUSDTM", "newSize": 9.0, "orderId": "454538143408951297", "clientOid": "11", "newPrice": 0.0000433,
		"sizeUnit": "UNIT", "cxlOnFail": true, "slTriggerPrice": 0.00003, "tpTriggerPrice": 0.000045, "slTriggerPriceType": "TP",
		"tpTriggerPriceType": "MP",
	}
	assert.Equal(t, exp, body, "Payload should send every documented parameter that is set, the decimals as bare numbers")
}

func TestAmendUTAOrder(t *testing.T) {
	t.Parallel()
	_, err := e.AmendUTAOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AmendUTAOrder must reject a nil request")
	_, err = e.AmendUTAOrder(t.Context(), &AmendUTAOrderRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "AmendUTAOrder must reject a request without a symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		req := newUTALiveFuturesOrder(t)
		placed := placeUTALiveOrder(t, req)
		result, err := e.AmendUTAOrder(t.Context(), &AmendUTAOrderRequest{Symbol: req.Symbol, OrderID: placed.OrderID, NewPrice: utaLiveFuturesPrice(t, 0.75)})
		require.NoError(t, err, "AmendUTAOrder must not error")
		require.NotNil(t, result, "AmendUTAOrder must return the order")
		assert.NotEmpty(t, result.OrderID, "AmendUTAOrder should return the order ID")
		// The client order ID finds the order whether or not the amendment gave it a new order ID
		_, err = e.CancelUTAOrder(t.Context(), &CancelUTAOrderRequest{TradeType: req.TradeType, Symbol: req.Symbol, ClientOrderID: req.ClientOrderID})
		require.NoError(t, err, "CancelUTAOrder must cancel the test's order")
		return
	}
	result, err := e.AmendUTAOrder(t.Context(), &AmendUTAOrderRequest{Symbol: "XBTUSDTM", OrderID: "461379758287044608", NewPrice: 50000})
	require.NoError(t, err, "AmendUTAOrder must not error")
	exp := &AmendUTAOrderResponse{
		OrderID:       "461379758287044608",
		ClientOrderID: "12259e46-6da3-40e5-9944-782a9f6fa05d",
	}
	assert.Equal(t, exp, result, "AmendUTAOrder should decode every field")
}

func TestCancelUTAOrderRequestValidate(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, (*CancelUTAOrderRequest)(nil).Validate(), common.ErrNilPointer, "Validate must reject a nil request")
	for _, tc := range []struct {
		name string
		req  *CancelUTAOrderRequest
		err  error
	}{
		{name: "no trade type", req: &CancelUTAOrderRequest{Symbol: "BTC-USDT", OrderID: "385577101236916224"}, err: errTradeTypeMissing},
		{name: "no symbol", req: &CancelUTAOrderRequest{TradeType: "SPOT", OrderID: "385577101236916224"}, err: currency.ErrSymbolStringEmpty},
		{name: "no order IDs", req: &CancelUTAOrderRequest{TradeType: "SPOT", Symbol: "BTC-USDT"}, err: order.ErrOrderIDNotSet},
		{name: "order ID", req: &CancelUTAOrderRequest{TradeType: "SPOT", Symbol: "BTC-USDT", OrderID: "385577101236916224"}},
		{name: "client order ID", req: &CancelUTAOrderRequest{TradeType: "SPOT", Symbol: "BTC-USDT", ClientOrderID: "451345g234rwegrwe24"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, tc.req.Validate(), tc.err, "Validate should return the expected error")
		})
	}
}

func TestCancelUTAOrderRequestPayload(t *testing.T) {
	t.Parallel()
	_, err := (*CancelUTAOrderRequest)(nil).Payload()
	require.ErrorIs(t, err, common.ErrNilPointer, "Payload must reject a nil request")
	_, err = (&CancelUTAOrderRequest{}).Payload()
	require.ErrorIs(t, err, errTradeTypeMissing, "Payload must validate the request")

	body, err := (&CancelUTAOrderRequest{TradeType: "FUTURES", Symbol: "XBTUSDTM", OrderID: "420781761635647489", ClientOrderID: "313e793a-a88b-4311-a590-a3b3bac00d2f-8df"}).Payload()
	require.NoError(t, err, "Payload must not error")
	exp := map[string]any{"tradeType": "FUTURES", "symbol": "XBTUSDTM", "orderId": "420781761635647489", "clientOid": "313e793a-a88b-4311-a590-a3b3bac00d2f-8df"}
	assert.Equal(t, exp, body, "Payload should send every documented parameter that is set")
}

func TestCancelUTAOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelUTAOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelUTAOrder must reject a nil request")
	_, err = e.CancelUTAOrder(t.Context(), &CancelUTAOrderRequest{})
	require.ErrorIs(t, err, errTradeTypeMissing, "CancelUTAOrder must reject a request without a trade type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		req := newUTALiveFuturesOrder(t)
		placed := placeUTALiveOrder(t, req)
		result, err := e.CancelUTAOrder(t.Context(), &CancelUTAOrderRequest{TradeType: req.TradeType, Symbol: req.Symbol, OrderID: placed.OrderID})
		require.NoError(t, err, "CancelUTAOrder must not error")
		require.NotNil(t, result, "CancelUTAOrder must return the order")
		assert.Equal(t, placed.OrderID, result.OrderID, "CancelUTAOrder should return the cancelled order's ID")
		return
	}
	result, err := e.CancelUTAOrder(t.Context(), &CancelUTAOrderRequest{TradeType: "SPOT", Symbol: "BTC-USDT", OrderID: "385577101236916224"})
	require.NoError(t, err, "CancelUTAOrder must not error")
	exp := &UTAOrderResponse{
		TradeType:     "SPOT",
		OrderID:       "385577101236916224",
		ClientOrderID: "451345g234rwegrwe24",
		Timestamp:     types.Time(time.UnixMilli(1764741957435)),
	}
	assert.Equal(t, exp, result, "CancelUTAOrder should decode every field")
}

func TestBatchCancelUTAOrdersByID(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *BatchCancelUTAOrdersByIDRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no trade type", req: &BatchCancelUTAOrdersByIDRequest{CancelOrderList: []BatchCancelUTAOrderItem{{Symbol: "BTC-USDT", OrderID: "403662035080048640"}}}, err: errTradeTypeMissing},
		{name: "no orders", req: &BatchCancelUTAOrdersByIDRequest{TradeType: "SPOT"}, err: errCancelOrderListEmpty},
		{name: "order without symbol", req: &BatchCancelUTAOrdersByIDRequest{TradeType: "SPOT", CancelOrderList: []BatchCancelUTAOrderItem{{OrderID: "403662035080048640"}}}, err: currency.ErrSymbolStringEmpty},
		{name: "order without IDs", req: &BatchCancelUTAOrdersByIDRequest{TradeType: "SPOT", CancelOrderList: []BatchCancelUTAOrderItem{{Symbol: "BTC-USDT"}}}, err: order.ErrOrderIDNotSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.BatchCancelUTAOrdersByID(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "BatchCancelUTAOrdersByID should reject the request")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		byOrderID, byClientOrderID := newUTALiveFuturesOrder(t), newUTALiveFuturesOrder(t)
		placed := placeUTALiveOrder(t, byOrderID)
		placeUTALiveOrder(t, byClientOrderID)
		result, err := e.BatchCancelUTAOrdersByID(t.Context(), &BatchCancelUTAOrdersByIDRequest{
			TradeType: "FUTURES",
			CancelOrderList: []BatchCancelUTAOrderItem{
				{Symbol: byOrderID.Symbol, OrderID: placed.OrderID},
				{Symbol: byClientOrderID.Symbol, ClientOrderID: byClientOrderID.ClientOrderID},
			},
		})
		require.NoError(t, err, "BatchCancelUTAOrdersByID must not error")
		require.NotNil(t, result, "BatchCancelUTAOrdersByID must return the outcomes")
		require.Len(t, result.Items, 2, "BatchCancelUTAOrdersByID must return the outcome of each order")
		for i := range result.Items {
			assert.Equalf(t, "200000", result.Items[i].Code, "BatchCancelUTAOrdersByID should cancel order %d: %s", i, result.Items[i].Message)
		}
		return
	}
	result, err := e.BatchCancelUTAOrdersByID(t.Context(), &BatchCancelUTAOrdersByIDRequest{
		TradeType: "SPOT",
		CancelOrderList: []BatchCancelUTAOrderItem{
			{Symbol: "BTC-USDT", OrderID: "403662035080048640"},
			{Symbol: "BTC-USDT", ClientOrderID: "a3fd4087f9854a68a0c285ebda2D03f5"},
			{Symbol: "BTC-USDT", OrderID: "403661901244530688"},
		},
	})
	require.NoError(t, err, "BatchCancelUTAOrdersByID must not error")
	exp := &BatchCancelUTAOrdersByIDResponse{
		TradeType: "SPOT",
		Items: []BatchCancelUTAOrderResult{
			{
				Code:      "200000",
				Message:   "success",
				OrderID:   "403662035080048640",
				Timestamp: types.Time(time.UnixMilli(1769053737742)),
			},
			{
				Code:          "200000",
				Message:       "success",
				OrderID:       "403661955870617600",
				ClientOrderID: "a3fd4087f9854a68a0c285ebda2D03f5",
				Timestamp:     types.Time(time.UnixMilli(1769053737743)),
			},
			{
				Code:      "110179",
				Message:   "Order not found.",
				OrderID:   "403661901244530688",
				Timestamp: types.Time(time.UnixMilli(1769053737744)),
			},
		},
	}
	assert.Equal(t, exp, result, "BatchCancelUTAOrdersByID should decode every field")
}

func TestBatchCancelUTAOrdersBySymbol(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *BatchCancelUTAOrdersBySymbolRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no trade type", req: &BatchCancelUTAOrdersBySymbolRequest{Symbol: "BTC-USDT", OrderFilter: "NORMAL"}, err: errTradeTypeMissing},
		{name: "no symbol", req: &BatchCancelUTAOrdersBySymbolRequest{TradeType: "MARGIN", OrderFilter: "NORMAL"}, err: currency.ErrSymbolStringEmpty},
		{name: "no order filter", req: &BatchCancelUTAOrdersBySymbolRequest{TradeType: "MARGIN", Symbol: "BTC-USDT"}, err: errOrderFilterRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.BatchCancelUTAOrdersBySymbol(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "BatchCancelUTAOrdersBySymbol should reject the request")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.BatchCancelUTAOrdersBySymbol(t.Context(), &BatchCancelUTAOrdersBySymbolRequest{TradeType: "MARGIN", Symbol: "BTC-USDT", MarginMode: "CROSS", OrderFilter: "NORMAL"})
	require.NoError(t, err, "BatchCancelUTAOrdersBySymbol must not error")
	if mockTests {
		exp := &BatchCancelUTAOrdersBySymbolResponse{
			TradeType: "MARGIN",
			Timestamp: types.Time(time.UnixMilli(1787888179737)),
			Items: []UTACancelledOrder{
				{
					OrderID: "482659377044606976",
				},
				{
					OrderID: "482659377044606977",
				},
			},
		}
		assert.Equal(t, exp, result, "BatchCancelUTAOrdersBySymbol should decode every field")
		return
	}
	assert.Equal(t, "MARGIN", result.TradeType, "BatchCancelUTAOrdersBySymbol should return the trade type")
}

func TestGetUTAOrderDetails(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *UTAOrderDetailsRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no trade type", req: &UTAOrderDetailsRequest{Symbol: "BTC-USDT", OrderID: "402668191476129792"}, err: errTradeTypeMissing},
		{name: "no symbol", req: &UTAOrderDetailsRequest{TradeType: "SPOT", OrderID: "402668191476129792"}, err: currency.ErrSymbolStringEmpty},
		{name: "no order IDs", req: &UTAOrderDetailsRequest{TradeType: "SPOT", Symbol: "BTC-USDT"}, err: order.ErrOrderIDNotSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetUTAOrderDetails(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetUTAOrderDetails should reject the request")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		for _, tradeType := range []string{"SPOT", "MARGIN", "FUTURES"} {
			history, err := e.GetUTAOrderHistory(t.Context(), &UTAOrderHistoryRequest{TradeType: tradeType})
			require.NoErrorf(t, err, "GetUTAOrderHistory must not error for %s", tradeType)
			require.NotNilf(t, history, "GetUTAOrderHistory must return a page for %s", tradeType)
			if len(history.Items) == 0 {
				continue
			}
			o := history.Items[0]
			result, err := e.GetUTAOrderDetails(t.Context(), &UTAOrderDetailsRequest{TradeType: tradeType, Symbol: o.Symbol, OrderID: o.OrderID})
			require.NoError(t, err, "GetUTAOrderDetails must not error")
			require.NotNil(t, result, "GetUTAOrderDetails must return the order")
			assert.Equal(t, o.OrderID, result.OrderID, "GetUTAOrderDetails should return the order asked for")
			return
		}
		t.Skip("GetUTAOrderDetails needs an order in the account's recent order history")
	}
	for _, tc := range []struct {
		name string
		req  *UTAOrderDetailsRequest
		exp  *UTAOrderDetailsResponse
	}{
		{
			name: "futures order by client order ID",
			req:  &UTAOrderDetailsRequest{TradeType: "FUTURES", Symbol: "XBTUSDTM", ClientOrderID: "test20938451234982234374024"},
			exp: &UTAOrderDetailsResponse{
				TradeType:                  "FUTURES",
				OrderID:                    "402665260420558848",
				ClientOrderID:              "test20938451234982234374024",
				Status:                     6,
				FilledSize:                 1,
				AveragePrice:               93026.2,
				Fee:                        0.01860524,
				FeeCurrency:                currency.USDT,
				Tax:                        0.00001861,
				TradeID:                    "1907914684480",
				Symbol:                     "XBTUSDTM",
				Side:                       "SELL",
				PositionSide:               "BOTH",
				OrderType:                  "LIMIT",
				Size:                       2,
				SizeUnit:                   "UNIT",
				Price:                      93026.2,
				ReduceOnly:                 true,
				MarginMode:                 "ISOLATED",
				SelfTradePrevention:        "CN",
				TimeInForce:                "GTC",
				CancelReason:               "USER",
				CancelSize:                 1,
				TakeProfitTriggerPrice:     95000,
				TakeProfitTriggerPriceType: "MP",
				StopLossTriggerPrice:       91000,
				StopLossTriggerPriceType:   "TP",
				PostOnly:                   true,
				Tags:                       "KuCoin API Service",
				OrderTime:                  types.Time(time.Unix(0, 1768816081243361300)),
				UpdatedTime:                types.Time(time.UnixMilli(1768816081248)),
				BrokerTag:                  "KuCoinPartner",
				TakeProfitOrderPrice:       94990,
				StopLossOrderPrice:         90990,
			},
		},
		{
			name: "spot order by order ID",
			req:  &UTAOrderDetailsRequest{TradeType: "SPOT", Symbol: "BTC-USDT", OrderID: "402668191476129792"},
			exp: &UTAOrderDetailsResponse{
				TradeType:           "SPOT",
				OrderID:             "402668191476129792",
				Status:              3,
				FilledSize:          0.0001,
				AveragePrice:        93100,
				Fee:                 0.00931,
				FeeCurrency:         currency.USDT,
				TradeID:             "1907915128064",
				Symbol:              "BTC-USDT",
				Side:                "BUY",
				OrderType:           "LIMIT",
				Size:                0.0001,
				SizeUnit:            "BASECCY",
				Price:               93100,
				SelfTradePrevention: "DC",
				TimeInForce:         "GTT",
				CancelAfter:         86400,
				TriggerDirection:    "UP",
				TriggerPrice:        93050,
				TriggerPriceType:    "TP",
				TriggerOrderID:      "402667890145705984",
				OrderTime:           types.Time(time.UnixMilli(1768816780412)),
				UpdatedTime:         types.Time(time.UnixMilli(1768816780419)),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAOrderDetails(t.Context(), tc.req)
			require.NoError(t, err, "GetUTAOrderDetails must not error")
			assert.Equal(t, tc.exp, result, "GetUTAOrderDetails should decode every field")
		})
	}
}

func TestUTABooleanUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for in, exp := range map[string]UTABoolean{
		`true`:  true,
		`false`: false,
		`""`:    false,
		`null`:  false,
	} {
		b := !exp
		require.NoErrorf(t, json.Unmarshal([]byte(in), &b), "UnmarshalJSON must not error for %s", in)
		assert.Equalf(t, exp, b, "UnmarshalJSON should decode %s", in)
	}
	var b UTABoolean
	for _, in := range []string{`"true"`, `1`, `"yes"`} {
		assert.ErrorIsf(t, json.Unmarshal([]byte(in), &b), common.ErrMalformedData, "UnmarshalJSON should reject %s", in)
	}
}

func TestGetUTAOpenOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAOpenOrderList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTAOpenOrderList must reject a nil request")
	_, err = e.GetUTAOpenOrderList(t.Context(), &UTAOpenOrderListRequest{Symbol: "XBTUSDTM"})
	require.ErrorIs(t, err, errTradeTypeMissing, "GetUTAOpenOrderList must reject a request without a trade type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *UTAOpenOrderListRequest
		exp  *UTAOpenOrderListResponse
	}{
		{
			name: "futures",
			req:  &UTAOpenOrderListRequest{TradeType: "FUTURES", Symbol: "XBTUSDTM", OrderFilter: "NORMAL", PageNumber: 1, PageSize: 50},
			exp: &UTAOpenOrderListResponse{
				PageNumber:  1,
				PageSize:    50,
				TotalNumber: 2,
				TotalPage:   1,
				TradeType:   "FUTURES",
				Items: []UTAOrder{
					{
						OrderID:                    "449471627734831104",
						ClientOrderID:              "12259e46-6da3-40e5-9944-782a9f6fa05d",
						Status:                     4,
						FilledSize:                 1,
						AveragePrice:               97500,
						Fee:                        0.0195,
						FeeCurrency:                currency.USDT,
						Tax:                        0.0000195,
						TradeID:                    "1936081221632",
						Symbol:                     "XBTUSDTM",
						Side:                       "SELL",
						PositionSide:               "BOTH",
						OrderType:                  "LIMIT",
						Size:                       2,
						SizeUnit:                   "UNIT",
						Price:                      97500,
						ReduceOnly:                 true,
						MarginMode:                 "ISOLATED",
						SelfTradePrevention:        "CN",
						TimeInForce:                "GTC",
						TakeProfitTriggerPrice:     99000,
						TakeProfitTriggerPriceType: "MP",
						StopLossTriggerPrice:       94000,
						StopLossTriggerPriceType:   "TP",
						PostOnly:                   true,
						Tags:                       "KuCoin API Service",
						OrderTime:                  types.Time(time.UnixMicro(1779975588831036)),
						UpdatedTime:                types.Time(time.Unix(0, 1779975588877691600)),
						BrokerTag:                  "KuCoinPartner",
					},
					{
						OrderID:          "449471901635465216",
						Status:           2,
						FeeCurrency:      currency.USDT,
						Symbol:           "XBTUSDTM",
						Side:             "BUY",
						PositionSide:     "BOTH",
						OrderType:        "LIMIT",
						Size:             1,
						SizeUnit:         "UNIT",
						Price:            95200,
						MarginMode:       "CROSS",
						TimeInForce:      "GTC",
						TriggerDirection: "DOWN",
						TriggerPrice:     95300,
						TriggerPriceType: "IP",
						TriggerOrderID:   "449471702553624576",
						OrderTime:        types.Time(time.UnixMicro(1779975653992121)),
						UpdatedTime:      types.Time(time.UnixMicro(1779975653992121)),
					},
				},
			},
		},
		{
			name: "spot",
			req:  &UTAOpenOrderListRequest{TradeType: "SPOT", Symbol: "BTC-USDT"},
			exp: &UTAOpenOrderListResponse{
				PageNumber:  1,
				PageSize:    50,
				TotalNumber: 1,
				TotalPage:   1,
				TradeType:   "SPOT",
				Items: []UTAOrder{
					{
						OrderID:             "449472210319597568",
						ClientOrderID:       "5c52e11203aa677f33e493fb",
						Status:              4,
						FilledSize:          0.0002,
						AveragePrice:        96010,
						Fee:                 0.019202,
						FeeCurrency:         currency.USDT,
						TradeID:             "1936081774848",
						Symbol:              "BTC-USDT",
						Side:                "BUY",
						OrderType:           "LIMIT",
						Size:                0.001,
						SizeUnit:            "BASECCY",
						Price:               96010,
						SelfTradePrevention: "DC",
						TimeInForce:         "GTT",
						CancelReason:        "STP",
						CancelAfter:         3600,
						CancelSize:          0.0003,
						Tags:                "KuCoin API Service",
						OrderTime:           types.Time(time.UnixMicro(1779975727591210)),
						UpdatedTime:         types.Time(time.UnixMicro(1779975801113305)),
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAOpenOrderList(t.Context(), tc.req)
			require.NoError(t, err, "GetUTAOpenOrderList must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTAOpenOrderList should decode every field")
				return
			}
			assert.Equal(t, tc.req.TradeType, result.TradeType, "GetUTAOpenOrderList should return the requested trade type")
		})
	}
}

func TestGetUTAOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTAOrderHistory must reject a nil request")
	_, err = e.GetUTAOrderHistory(t.Context(), &UTAOrderHistoryRequest{Symbol: "XBTUSDTM"})
	require.ErrorIs(t, err, errTradeTypeMissing, "GetUTAOrderHistory must reject a request without a trade type")
	_, err = e.GetUTAOrderHistory(t.Context(), &UTAOrderHistoryRequest{TradeType: "FUTURES", StartAt: time.UnixMilli(1768820400000), EndAt: time.UnixMilli(1768813200000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUTAOrderHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *UTAOrderHistoryRequest
		exp  *UTAOrderHistoryResponse
	}{
		{
			name: "futures",
			req: &UTAOrderHistoryRequest{
				TradeType: "FUTURES", Symbol: "XBTUSDTM", Side: "BUY", OrderFilter: "NORMAL", StartAt: time.UnixMilli(1768813200000),
				EndAt: time.UnixMilli(1768820400000), LastID: 130610750, PageSize: 50,
			},
			exp: &UTAOrderHistoryResponse{
				LastID:    130610708,
				TradeType: "FUTURES",
				Items: []UTAOrder{
					{
						OrderID:                    "402666817186754560",
						ClientOrderID:              "1d70265a-b488-4342-9e8e-b4d150e3532b",
						Status:                     5,
						FeeCurrency:                currency.USDT,
						Symbol:                     "XBTUSDTM",
						Side:                       "BUY",
						PositionSide:               "BOTH",
						OrderType:                  "LIMIT",
						Size:                       10,
						SizeUnit:                   "UNIT",
						Price:                      90000,
						MarginMode:                 "CROSS",
						SelfTradePrevention:        "CO",
						TimeInForce:                "GTC",
						CancelReason:               "USER",
						CancelSize:                 10,
						TakeProfitTriggerPrice:     98000,
						TakeProfitTriggerPriceType: "MP",
						StopLossTriggerPrice:       88000,
						StopLossTriggerPriceType:   "TP",
						PostOnly:                   true,
						Tags:                       "KuCoin API Service",
						OrderTime:                  types.Time(time.UnixMicro(1768816452405841)),
						UpdatedTime:                types.Time(time.UnixMilli(1768816452408)),
						BrokerTag:                  "KuCoinPartner",
					},
					{
						OrderID:          "402666910342643712",
						Status:           3,
						FilledSize:       1,
						AveragePrice:     93512.4,
						Fee:              0.05610744,
						FeeCurrency:      currency.USDT,
						Tax:              0.00005611,
						TradeID:          "1907915004928",
						Symbol:           "XBTUSDTM",
						Side:             "BUY",
						PositionSide:     "BOTH",
						OrderType:        "MARKET",
						Size:             1,
						SizeUnit:         "UNIT",
						ReduceOnly:       true,
						MarginMode:       "ISOLATED",
						TimeInForce:      "GTC",
						TriggerDirection: "UP",
						TriggerPrice:     93500,
						TriggerPriceType: "MP",
						TriggerOrderID:   "402666700345581568",
						OrderTime:        types.Time(time.UnixMilli(1768816474618)),
						UpdatedTime:      types.Time(time.UnixMilli(1768816474623)),
					},
				},
			},
		},
		{
			name: "spot",
			req:  &UTAOrderHistoryRequest{TradeType: "SPOT"},
			exp: &UTAOrderHistoryResponse{
				LastID:    130610001,
				TradeType: "SPOT",
				Items: []UTAOrder{
					{
						OrderID:             "402660114562121728",
						ClientOrderID:       "e6e3ecd4-35d2-4b82-974b-0db138e0756f",
						Status:              6,
						FilledSize:          0.0004,
						AveragePrice:        93900,
						Fee:                 0.03756,
						FeeCurrency:         currency.USDT,
						TradeID:             "1907911020544",
						Symbol:              "BTC-USDT",
						Side:                "SELL",
						OrderType:           "LIMIT",
						Size:                0.001,
						SizeUnit:            "BASECCY",
						Price:               93900,
						SelfTradePrevention: "DC",
						TimeInForce:         "GTT",
						CancelReason:        "ORDER_EXPIRED",
						CancelAfter:         3600,
						CancelSize:          0.0006,
						OrderTime:           types.Time(time.UnixMilli(1768814854617)),
						UpdatedTime:         types.Time(time.UnixMilli(1768818454620)),
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAOrderHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetUTAOrderHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTAOrderHistory should decode every field")
				return
			}
			assert.Equal(t, tc.req.TradeType, result.TradeType, "GetUTAOrderHistory should return the requested trade type")
		})
	}
}

func TestGetUTATradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTATradeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTATradeHistory must reject a nil request")
	_, err = e.GetUTATradeHistory(t.Context(), &UTATradeHistoryRequest{Symbol: "XBTUSDTM"})
	require.ErrorIs(t, err, errTradeTypeMissing, "GetUTATradeHistory must reject a request without a trade type")
	_, err = e.GetUTATradeHistory(t.Context(), &UTATradeHistoryRequest{TradeType: "FUTURES", StartAt: time.UnixMilli(1788883200000), EndAt: time.UnixMilli(1788796800000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUTATradeHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetUTATradeHistory(t.Context(), &UTATradeHistoryRequest{
		TradeType: "FUTURES", Symbol: "XBTUSDTM", OrderID: "486825585260236800", Side: "SELL", StartAt: time.UnixMilli(1788796800000),
		EndAt: time.UnixMilli(1788883200000), LastID: 10000000138744600, PageSize: 50, FillType: "NORMAL",
	})
	require.NoError(t, err, "GetUTATradeHistory must not error")
	if mockTests {
		exp := &UTATradeHistoryResponse{
			LastID:    10000000138744534,
			TradeType: "FUTURES",
			Items: []UTAFill{
				{
					TradeType:     "FUTURES",
					Symbol:        "XBTUSDTM",
					OrderID:       "486825585260236800",
					ClientOrderID: "8c0c0f02-0c53-4e75-92f6-4bdfacb34496",
					OrderType:     "MARKET",
					Side:          "SELL",
					PositionSide:  "BOTH",
					FillType:      "NORMAL",
					TradeID:       "1943106280799",
					Size:          1,
					Value:         78.4306,
					Price:         78430.6,
					ExecutionTime: types.Time(time.UnixMilli(1788881466353)),
					Fee:           0.04705836,
					FeeCurrency:   currency.USDT,
					LiquidityRole: "TAKER",
					MarginMode:    "ISOLATED",
					Tax:           0.00004706,
					FillPNL:       0.0185,
				},
				{
					TradeType:     "FUTURES",
					Symbol:        "XBTUSDTM",
					OrderID:       "486825585260236800",
					ClientOrderID: "8c0c0f02-0c53-4e75-92f6-4bdfacb34496",
					OrderType:     "MARKET",
					Side:          "SELL",
					PositionSide:  "BOTH",
					FillType:      "NORMAL",
					TradeID:       "1943106280800",
					Size:          2,
					Value:         156.861,
					Price:         78430.5,
					ExecutionTime: types.Time(time.UnixMilli(1788881466353)),
					Fee:           0.0941166,
					FeeCurrency:   currency.USDT,
					LiquidityRole: "TAKER",
					MarginMode:    "ISOLATED",
					Tax:           0.00009412,
					FillPNL:       0.0368,
				},
			},
		}
		assert.Equal(t, exp, result, "GetUTATradeHistory should decode every field")
		return
	}
	assert.Equal(t, "FUTURES", result.TradeType, "GetUTATradeHistory should return the requested trade type")
}

func TestGetUTAMaxOrderQuantity(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *UTAMaxOrderQuantityRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no symbol", req: &UTAMaxOrderQuantityRequest{TradeType: "FUTURES"}, err: currency.ErrSymbolStringEmpty},
		{name: "no trade type", req: &UTAMaxOrderQuantityRequest{Symbol: "XBTUSDTM"}, err: errTradeTypeMissing},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetUTAMaxOrderQuantity(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetUTAMaxOrderQuantity should reject the request")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *UTAMaxOrderQuantityRequest
		exp  *UTAMaxOrderQuantityResponse
	}{
		{
			name: "futures at a price",
			req:  &UTAMaxOrderQuantityRequest{Symbol: "XBTUSDTM", TradeType: "FUTURES", Price: 78000},
			exp: &UTAMaxOrderQuantityResponse{
				Symbol:      "XBTUSDTM",
				MaxBuySize:  1160,
				MaxSellSize: 1150,
			},
		},
		{
			name: "spot at the market price",
			req:  &UTAMaxOrderQuantityRequest{Symbol: "BTC-USDT", TradeType: "SPOT"},
			exp: &UTAMaxOrderQuantityResponse{
				Symbol:      "BTC-USDT",
				MaxBuySize:  959.16648010714299972,
				MaxSellSize: 0.00000000528511207163,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAMaxOrderQuantity(t.Context(), tc.req)
			require.NoError(t, err, "GetUTAMaxOrderQuantity must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTAMaxOrderQuantity should decode every field")
				return
			}
			assert.Equal(t, tc.req.Symbol, result.Symbol, "GetUTAMaxOrderQuantity should return the requested symbol")
		})
	}
}
