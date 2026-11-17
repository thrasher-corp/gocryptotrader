package bingx

import (
	"context"
	"errors"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// spotDocumentedOrderSkip is why a live test skips a request for an order the documentation's example names
const spotDocumentedOrderSkip = "live accounts hold no order with the documented example's ID"

func TestPlaceSpotOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceSpotOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceSpotOrder must reject a nil request")
	_, err = e.PlaceSpotOrder(t.Context(), &PlaceSpotOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Type: OrderTypeMarket})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "PlaceSpotOrder must reject an order without an amount")
	_, err = new(Exchange).PlaceSpotOrder(t.Context(), &PlaceSpotOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Type: OrderTypeMarket, QuoteOrderQuantity: 100})
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "PlaceSpotOrder must return the symbol formatting error")

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex, requests := spotTestServer(t, `{"code":0,"msg":"","data":{"symbol":"BTC-USDT","orderId":1}}`)
		_, err := ex.PlaceSpotOrder(t.Context(), &PlaceSpotOrderRequest{
			Symbol:             currency.NewBTCUSDT(),
			Side:               SideBuy,
			Type:               OrderTypeTriggerMarket,
			StopPrice:          60000,
			QuoteOrderQuantity: 100,
			NewClientOrderID:   "gct_1",
			TimeInForce:        TimeInForceImmediateOrCancel,
		})
		require.NoError(t, err, "PlaceSpotOrder must not error")
		r := spotReceiveRequest(t, requests)
		assert.Equal(t, http.MethodPost, r.method, "method should be POST")
		assert.Equal(t, "/openApi/spot/v1/trade/order", r.path, "path should be the order endpoint")
		verifyQuerySignature(t, r.rawQuery)
		exp := url.Values{
			"symbol":           {"BTC-USDT"},
			"side":             {"BUY"},
			"type":             {"TRIGGER_MARKET"},
			"stopPrice":        {"60000"},
			"quoteOrderQty":    {"100"},
			"newClientOrderId": {"gct_1"},
			"timeInForce":      {"IOC"},
			"recvWindow":       {"5000"},
		}
		assert.Equal(t, exp, r.params, "PlaceSpotOrder should send every parameter set")
	})

	t.Run("reply without an order ID", func(t *testing.T) {
		t.Parallel()
		ex, _ := spotTestServer(t, `{"code":0,"msg":"","data":{"symbol":"BTC-USDT"}}`)
		_, err := ex.PlaceSpotOrder(t.Context(), &PlaceSpotOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideSell, Type: OrderTypeMarket, Quantity: 1})
		assert.ErrorIs(t, err, errSpotOrderIDMissing, "PlaceSpotOrder should reject a reply without an order ID")
	})

	t.Run("documented example", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
		}
		result, err := e.PlaceSpotOrder(t.Context(), &PlaceSpotOrderRequest{
			Symbol:      currency.NewPair(currency.NewCode("NEMS"), currency.USDT),
			Side:        SideSell,
			Type:        OrderTypeLimit,
			Quantity:    115,
			Price:       0.1557,
			TimeInForce: TimeInForceGoodTillCancel,
		})
		require.NoError(t, err, "PlaceSpotOrder must not error")
		if mockTests {
			exp := &SpotOrderResponse{
				Symbol:           "NEMS-USDT",
				OrderID:          1735963671248581000,
				TransactTime:     types.Time(time.UnixMilli(1702720966528)),
				Price:            0.1557,
				OriginalQuantity: 115,
				Status:           SpotOrderStatusPending,
				Type:             OrderTypeLimit,
				Side:             SideSell,
			}
			assert.Equal(t, exp, result, "PlaceSpotOrder should decode every field")
			return
		}
		assert.NotZero(t, result.OrderID, "OrderID should be set")
	})
}

func TestPlaceSpotBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceSpotBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceSpotBatchOrders must reject a nil request")
	_, err = e.PlaceSpotBatchOrders(t.Context(), &PlaceSpotBatchOrdersRequest{})
	require.ErrorIs(t, err, common.ErrEmptyParams, "PlaceSpotBatchOrders must reject a request without orders")
	valid := PlaceSpotOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Type: OrderTypeLimit, Quantity: 1, Price: 1}
	_, err = e.PlaceSpotBatchOrders(t.Context(), &PlaceSpotBatchOrdersRequest{Orders: []PlaceSpotOrderRequest{valid, valid, valid, valid, valid, valid}})
	require.ErrorIs(t, err, errSpotTooManyOrders, "PlaceSpotBatchOrders must reject more than five orders")
	_, err = e.PlaceSpotBatchOrders(t.Context(), &PlaceSpotBatchOrdersRequest{Orders: []PlaceSpotOrderRequest{valid, {Symbol: currency.NewBTCUSDT(), Type: OrderTypeLimit, Quantity: 1}}})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "PlaceSpotBatchOrders must reject an invalid order")
	require.ErrorContains(t, err, "order 1", "PlaceSpotBatchOrders must name the invalid order")

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex, requests := spotTestServer(t, `{"code":0,"msg":"","data":{"orders":[]}}`)
		_, err := ex.PlaceSpotBatchOrders(t.Context(), &PlaceSpotBatchOrdersRequest{
			Orders: []PlaceSpotOrderRequest{
				{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Type: OrderTypeMarket, QuoteOrderQuantity: 100, NewClientOrderID: "gct_1"},
				{Symbol: currency.NewPair(currency.ETH, currency.USDT), Side: SideSell, Type: OrderTypeTakeStopLimit, StopPrice: 2000, Quantity: 0.5, Price: 1990, TimeInForce: TimeInForcePostOnly},
			},
			Sync: true,
		})
		require.NoError(t, err, "PlaceSpotBatchOrders must not error")
		r := spotReceiveRequest(t, requests)
		assert.Equal(t, http.MethodPost, r.method, "method should be POST")
		assert.Equal(t, "/openApi/spot/v1/trade/batchOrders", r.path, "path should be the batch orders endpoint")
		verifyQuerySignature(t, r.rawQuery)
		exp := url.Values{
			"data":       {`[{"symbol":"BTC-USDT","side":"BUY","type":"MARKET","quoteOrderQty":100,"newClientOrderId":"gct_1"},{"symbol":"ETH-USDT","side":"SELL","type":"TAKE_STOP_LIMIT","stopPrice":2000,"quantity":0.5,"price":1990,"timeInForce":"PostOnly"}]`},
			"sync":       {"true"},
			"recvWindow": {"5000"},
		}
		assert.Equal(t, exp, r.params, "PlaceSpotBatchOrders should send the orders as a JSON array with bare numbers")
	})

	t.Run("orders JSON cannot hold", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(http.ResponseWriter, *http.Request) {
			assert.Fail(t, "a batch that cannot be encoded should not be sent")
		})
		unencodable := valid
		unencodable.Price = math.Inf(1)
		_, err := ex.PlaceSpotBatchOrders(t.Context(), &PlaceSpotBatchOrdersRequest{Orders: []PlaceSpotOrderRequest{valid, unencodable}})
		assert.ErrorContains(t, err, "unsupported value", "PlaceSpotBatchOrders should report a batch JSON cannot hold")
	})

	t.Run("request error", func(t *testing.T) {
		t.Parallel()
		ex, _ := spotTestServer(t, `{"code":100500,"msg":"busy"}`)
		result, err := ex.PlaceSpotBatchOrders(t.Context(), &PlaceSpotBatchOrdersRequest{Orders: []PlaceSpotOrderRequest{valid}})
		assert.ErrorIs(t, err, errAPIResponse, "PlaceSpotBatchOrders should return the request's error")
		assert.Nil(t, result, "PlaceSpotBatchOrders should return no orders for a failed request")
	})

	t.Run("documented example", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
		}
		result, err := e.PlaceSpotBatchOrders(t.Context(), &PlaceSpotBatchOrdersRequest{
			Orders: []PlaceSpotOrderRequest{
				{Symbol: currency.NewPair(currency.ETH, currency.USDT), Side: SideBuy, Type: OrderTypeLimit, Quantity: 7.663, Price: 12.479},
			},
		})
		require.NoError(t, err, "PlaceSpotBatchOrders must not error")
		if mockTests {
			exp := []SpotOrderResponse{
				{
					Symbol:           "ETH-USDT",
					OrderID:          1735967859282101200,
					TransactTime:     types.Time(time.UnixMilli(1702721965033)),
					Price:            12.479,
					OriginalQuantity: 7.663,
					Status:           SpotOrderStatusPending,
					Type:             OrderTypeLimit,
					Side:             SideBuy,
				},
			}
			assert.Equal(t, exp, result, "PlaceSpotBatchOrders should decode every field")
			return
		}
		assert.NotEmpty(t, result, "PlaceSpotBatchOrders should return the orders")
	})
}

func TestCancelSpotOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelSpotOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelSpotOrder must reject a nil request")
	_, err = e.CancelSpotOrder(t.Context(), &CancelSpotOrderRequest{OrderID: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelSpotOrder must reject an empty pair")
	_, err = e.CancelSpotOrder(t.Context(), &CancelSpotOrderRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelSpotOrder must reject a request without an order ID")
	_, err = new(Exchange).CancelSpotOrder(t.Context(), &CancelSpotOrderRequest{Symbol: currency.NewBTCUSDT(), OrderID: 1})
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "CancelSpotOrder must return the symbol formatting error")

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex, requests := spotTestServer(t, `{"code":0,"msg":"","data":{"symbol":"BTC-USDT","orderId":1}}`)
		_, err := ex.CancelSpotOrder(t.Context(), &CancelSpotOrderRequest{Symbol: currency.NewBTCUSDT(), ClientOrderID: "gct_1", CancelRestrictions: SpotOrderStatusPartiallyFilled})
		require.NoError(t, err, "CancelSpotOrder must not error")
		r := spotReceiveRequest(t, requests)
		assert.Equal(t, http.MethodPost, r.method, "method should be POST")
		assert.Equal(t, "/openApi/spot/v1/trade/cancel", r.path, "path should be the cancel endpoint")
		verifyQuerySignature(t, r.rawQuery)
		exp := url.Values{"symbol": {"BTC-USDT"}, "clientOrderID": {"gct_1"}, "cancelRestrictions": {"PARTIALLY_FILLED"}, "recvWindow": {"5000"}}
		assert.Equal(t, exp, r.params, "CancelSpotOrder should send every parameter set")
	})

	t.Run("order not found", func(t *testing.T) {
		t.Parallel()
		ex, _ := spotTestServer(t, `{"code":100404,"msg":"order not exist"}`)
		_, err := ex.CancelSpotOrder(t.Context(), &CancelSpotOrderRequest{Symbol: currency.NewBTCUSDT(), OrderID: 1})
		assert.ErrorIs(t, err, order.ErrOrderNotFound, "CancelSpotOrder should map code 100404 to ErrOrderNotFound")
	})

	t.Run("reply without an order ID", func(t *testing.T) {
		t.Parallel()
		ex, _ := spotTestServer(t, `{"code":0,"msg":"","data":{}}`)
		_, err := ex.CancelSpotOrder(t.Context(), &CancelSpotOrderRequest{Symbol: currency.NewBTCUSDT(), OrderID: 1})
		assert.ErrorIs(t, err, errSpotOrderIDMissing, "CancelSpotOrder should reject a reply without an order ID")
	})

	t.Run("documented example", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			t.Skip(spotDocumentedOrderSkip)
		}
		result, err := e.CancelSpotOrder(t.Context(), &CancelSpotOrderRequest{Symbol: currency.NewPair(currency.NewCode("NEMS"), currency.USDT), OrderID: 1735964079647111280})
		require.NoError(t, err, "CancelSpotOrder must not error")
		exp := &SpotOrderResponse{
			Symbol:           "NEMS-USDT",
			OrderID:          1735964079647111200,
			Price:            0.1532,
			StopPrice:        0.1532,
			OriginalQuantity: 126,
			Status:           SpotOrderStatusCancelled,
			Type:             OrderTypeLimit,
			Side:             SideSell,
		}
		assert.Equal(t, exp, result, "CancelSpotOrder should decode every field")
	})
}

func TestCancelSpotBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelSpotBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelSpotBatchOrders must reject a nil request")
	_, err = e.CancelSpotBatchOrders(t.Context(), &CancelSpotBatchOrdersRequest{OrderIDs: []uint64{1}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelSpotBatchOrders must reject an empty pair")
	_, err = e.CancelSpotBatchOrders(t.Context(), &CancelSpotBatchOrdersRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelSpotBatchOrders must reject a request without order IDs")
	_, err = new(Exchange).CancelSpotBatchOrders(t.Context(), &CancelSpotBatchOrdersRequest{Symbol: currency.NewBTCUSDT(), OrderIDs: []uint64{1}})
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "CancelSpotBatchOrders must return the symbol formatting error")

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex, requests := spotTestServer(t, `{"code":0,"msg":"","data":{"fails":[],"orders":[]}}`)
		_, err := ex.CancelSpotBatchOrders(t.Context(), &CancelSpotBatchOrdersRequest{Symbol: currency.NewBTCUSDT(), Process: true, ClientOrderIDs: []string{"gct_1", "gct_2"}})
		require.NoError(t, err, "CancelSpotBatchOrders must not error")
		r := spotReceiveRequest(t, requests)
		assert.Equal(t, http.MethodPost, r.method, "method should be POST")
		assert.Equal(t, "/openApi/spot/v1/trade/cancelOrders", r.path, "path should be the batch cancel endpoint")
		verifyQuerySignature(t, r.rawQuery)
		exp := url.Values{"symbol": {"BTC-USDT"}, "process": {"1"}, "clientOrderIDs": {"gct_1,gct_2"}, "recvWindow": {"5000"}}
		assert.Equal(t, exp, r.params, "CancelSpotBatchOrders should send every parameter set")
	})

	t.Run("order not found", func(t *testing.T) {
		t.Parallel()
		ex, _ := spotTestServer(t, `{"code":100404,"msg":"order not exist"}`)
		_, err := ex.CancelSpotBatchOrders(t.Context(), &CancelSpotBatchOrdersRequest{Symbol: currency.NewBTCUSDT(), OrderIDs: []uint64{1}})
		assert.ErrorIs(t, err, order.ErrOrderNotFound, "CancelSpotBatchOrders should map code 100404 to ErrOrderNotFound")
	})

	t.Run("documented example", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			t.Skip(spotDocumentedOrderSkip)
		}
		result, err := e.CancelSpotBatchOrders(t.Context(), &CancelSpotBatchOrdersRequest{
			Symbol:   currency.NewPair(currency.NewCode("GM"), currency.USDT),
			OrderIDs: []uint64{1735964997957275648, 1735965127519326208},
		})
		require.NoError(t, err, "CancelSpotBatchOrders must not error")
		exp := &CancelSpotBatchOrdersResponse{
			Fails: []SpotCancelFailure{
				{OrderID: 1874730962584076288, Error: "the  orderId:1874730962584076288 order status :CANCELED,can not delete, please verify and retry later"},
			},
			Orders: spotDocumentedCancelledOrders(),
		}
		assert.Equal(t, exp, result, "CancelSpotBatchOrders should decode every field")
	})
}

// spotDocumentedCancelledOrders returns the orders the documented examples of the spot batch and open order
// cancellations report
func spotDocumentedCancelledOrders() []SpotOrderResponse {
	return []SpotOrderResponse{
		{
			Symbol:           "GM-USDT",
			OrderID:          1735964997957275600,
			TransactTime:     types.Time(time.UnixMilli(1702721242701)),
			Price:            0.00000398,
			StopPrice:        0.00000398,
			OriginalQuantity: 8061558,
			Status:           SpotOrderStatusCancelled,
			Type:             OrderTypeLimit,
			Side:             SideSell,
			ClientOrderID:    "2most51702721242645506402",
		},
		{
			Symbol:           "GM-USDT",
			OrderID:          1735965127519326200,
			TransactTime:     types.Time(time.UnixMilli(1702721249787)),
			Price:            0.00000398,
			StopPrice:        0.00000398,
			OriginalQuantity: 5806281,
			Status:           SpotOrderStatusCancelled,
			Type:             OrderTypeLimit,
			Side:             SideSell,
			ClientOrderID:    "2most51702721249647382871",
		},
	}
}

func TestCancelSpotOpenOrders(t *testing.T) {
	t.Parallel()
	t.Run("every symbol request", func(t *testing.T) {
		t.Parallel()
		ex, requests := spotTestServer(t, `{"code":0,"msg":"","data":{"orders":[]}}`)
		_, err := ex.CancelSpotOpenOrders(t.Context(), currency.EMPTYPAIR)
		require.NoError(t, err, "CancelSpotOpenOrders must not error")
		r := spotReceiveRequest(t, requests)
		assert.Equal(t, http.MethodPost, r.method, "method should be POST")
		assert.Equal(t, "/openApi/spot/v1/trade/cancelOpenOrders", r.path, "path should be the cancel open orders endpoint")
		verifyQuerySignature(t, r.rawQuery)
		assert.Equal(t, url.Values{"recvWindow": {"5000"}}, r.params, "no symbol should be sent to cancel every open order")
	})

	t.Run("symbol formatting error", func(t *testing.T) {
		t.Parallel()
		_, err := new(Exchange).CancelSpotOpenOrders(t.Context(), currency.NewBTCUSDT())
		assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "CancelSpotOpenOrders should return the symbol formatting error")
	})

	t.Run("documented example", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
		}
		result, err := e.CancelSpotOpenOrders(t.Context(), currency.NewPair(currency.NewCode("GM"), currency.USDT))
		require.NoError(t, err, "CancelSpotOpenOrders must not error")
		if mockTests {
			assert.Equal(t, spotDocumentedCancelledOrders(), result, "CancelSpotOpenOrders should decode every field")
		}
	})
}

func TestCancelReplaceSpotOrder(t *testing.T) {
	t.Parallel()
	valid := CancelReplaceSpotOrderRequest{
		Symbol:            currency.NewBTCUSDT(),
		CancelOrderID:     1,
		CancelReplaceMode: CancelReplaceModeStopOnFailure,
		Side:              SideBuy,
		Type:              OrderTypeLimit,
		Quantity:          1,
		Price:             40000,
	}
	for _, tc := range []struct {
		name   string
		modify func(*CancelReplaceSpotOrderRequest)
		err    error
	}{
		{name: "empty pair", modify: func(r *CancelReplaceSpotOrderRequest) { r.Symbol = currency.EMPTYPAIR }, err: currency.ErrCurrencyPairEmpty},
		{name: "no order to cancel", modify: func(r *CancelReplaceSpotOrderRequest) { r.CancelOrderID = 0 }, err: order.ErrOrderIDNotSet},
		{name: "no mode", modify: func(r *CancelReplaceSpotOrderRequest) { r.CancelReplaceMode = "" }, err: errSpotCancelReplaceModeEmpty},
		{name: "no side", modify: func(r *CancelReplaceSpotOrderRequest) { r.Side = "" }, err: order.ErrSideIsInvalid},
		{name: "no type", modify: func(r *CancelReplaceSpotOrderRequest) { r.Type = "" }, err: order.ErrTypeIsInvalid},
		{name: "no amount", modify: func(r *CancelReplaceSpotOrderRequest) { r.Quantity = 0 }, err: order.ErrAmountIsInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := valid
			tc.modify(&req)
			_, err := e.CancelReplaceSpotOrder(t.Context(), &req)
			assert.ErrorIs(t, err, tc.err, "CancelReplaceSpotOrder should reject an incomplete request")
		})
	}

	t.Run("nil request", func(t *testing.T) {
		t.Parallel()
		_, err := e.CancelReplaceSpotOrder(t.Context(), nil)
		assert.ErrorIs(t, err, common.ErrNilPointer, "CancelReplaceSpotOrder should reject a nil request")
	})

	t.Run("symbol formatting error", func(t *testing.T) {
		t.Parallel()
		req := valid
		_, err := new(Exchange).CancelReplaceSpotOrder(t.Context(), &req)
		assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "CancelReplaceSpotOrder should return the symbol formatting error")
	})

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex, requests := spotTestServer(t, `{"code":0,"msg":"","data":{"cancelResult":{"code":0,"msg":"","result":true}}}`)
		_, err := ex.CancelReplaceSpotOrder(t.Context(), &CancelReplaceSpotOrderRequest{
			Symbol:              currency.NewBTCUSDT(),
			CancelClientOrderID: "gct_1",
			CancelRestrictions:  SpotOrderStatusNew,
			CancelReplaceMode:   CancelReplaceModeStopOnFailure,
			Side:                SideSell,
			Type:                OrderTypeTriggerLimit,
			StopPrice:           70000,
			QuoteOrderQuantity:  100,
			Price:               69000,
			NewClientOrderID:    "gct_2",
		})
		require.NoError(t, err, "CancelReplaceSpotOrder must not error")
		r := spotReceiveRequest(t, requests)
		assert.Equal(t, http.MethodPost, r.method, "method should be POST")
		assert.Equal(t, "/openApi/spot/v1/trade/order/cancelReplace", r.path, "path should be the cancel-replace endpoint")
		verifyQuerySignature(t, r.rawQuery)
		exp := url.Values{
			"symbol":              {"BTC-USDT"},
			"cancelClientOrderID": {"gct_1"},
			"cancelRestrictions":  {"NEW"},
			"cancelReplaceMode":   {"STOP_ON_FAILURE"},
			"side":                {"SELL"},
			"type":                {"TRIGGER_LIMIT"},
			"stopPrice":           {"70000"},
			"quoteOrderQty":       {"100"},
			"price":               {"69000"},
			"newClientOrderId":    {"gct_2"},
			"recvWindow":          {"5000"},
		}
		assert.Equal(t, exp, r.params, "CancelReplaceSpotOrder should send every parameter set")
	})

	t.Run("refused with an outcome", func(t *testing.T) {
		t.Parallel()
		ex, _ := spotTestServer(t, `{"code":100400,"msg":"cancel failed","data":{"cancelResult":{"code":100400,"msg":"order not exist","result":false},"openResult":{"code":0,"msg":"","result":false}}}`)
		req := valid
		result, err := ex.CancelReplaceSpotOrder(t.Context(), &req)
		require.ErrorIs(t, err, errAPIResponse, "CancelReplaceSpotOrder must return the refusal")
		require.NotNil(t, result, "CancelReplaceSpotOrder must return the outcome the refusal carries")
		exp := &CancelReplaceSpotOrderResponse{CancelResult: SpotCancelReplaceResult{Code: 100400, Message: "order not exist"}}
		assert.Equal(t, exp, result, "CancelReplaceSpotOrder should decode both halves' outcomes")
	})

	t.Run("order not found", func(t *testing.T) {
		t.Parallel()
		ex, _ := spotTestServer(t, `{"code":100404,"msg":"order not exist"}`)
		req := valid
		_, err := ex.CancelReplaceSpotOrder(t.Context(), &req)
		assert.ErrorIs(t, err, order.ErrOrderNotFound, "CancelReplaceSpotOrder should map code 100404 to ErrOrderNotFound")
	})

	t.Run("documented example", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			t.Skip(spotDocumentedOrderSkip)
		}
		result, err := e.CancelReplaceSpotOrder(t.Context(), &CancelReplaceSpotOrderRequest{
			Symbol:            currency.NewBTCUSDT(),
			CancelOrderID:     17543893539094511234,
			CancelReplaceMode: CancelReplaceModeAllowFailure,
			Side:              SideBuy,
			Type:              OrderTypeLimit,
			Quantity:          1,
			Price:             40000,
		})
		require.NoError(t, err, "CancelReplaceSpotOrder must not error")
		exp := &CancelReplaceSpotOrderResponse{
			CancelResult: SpotCancelReplaceResult{Code: 100400, Message: " order not exist"},
			OpenResult:   SpotCancelReplaceResult{Result: true},
			OrderOpenResponse: SpotOrderResponse{
				Symbol:           "BTC-USDT",
				OrderID:          1754389353909452800,
				TransactTime:     types.Time(time.UnixMilli(1707113991607)),
				Price:            40000,
				OriginalQuantity: 1,
				Status:           SpotOrderStatusPending,
				Type:             OrderTypeLimit,
				Side:             SideBuy,
			},
		}
		assert.Equal(t, exp, result, "CancelReplaceSpotOrder should decode both halves")
	})
}

func TestGetSpotOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotOrderDetails(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotOrderDetails must reject a nil request")
	_, err = e.GetSpotOrderDetails(t.Context(), &SpotOrderDetailsRequest{OrderID: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSpotOrderDetails must reject an empty pair")
	_, err = e.GetSpotOrderDetails(t.Context(), &SpotOrderDetailsRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSpotOrderDetails must reject a request without an order ID")
	_, err = new(Exchange).GetSpotOrderDetails(t.Context(), &SpotOrderDetailsRequest{Symbol: currency.NewBTCUSDT(), OrderID: 1})
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSpotOrderDetails must return the symbol formatting error")

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex, requests := spotTestServer(t, `{"code":0,"msg":"","data":{"symbol":"BTC-USDT","orderId":1}}`)
		_, err := ex.GetSpotOrderDetails(t.Context(), &SpotOrderDetailsRequest{Symbol: currency.NewBTCUSDT(), ClientOrderID: "gct_1"})
		require.NoError(t, err, "GetSpotOrderDetails must not error")
		r := spotReceiveRequest(t, requests)
		assert.Equal(t, http.MethodGet, r.method, "method should be GET")
		assert.Equal(t, "/openApi/spot/v1/trade/query", r.path, "path should be the order query endpoint")
		verifyQuerySignature(t, r.rawQuery)
		assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "clientOrderID": {"gct_1"}, "recvWindow": {"5000"}}, r.params, "GetSpotOrderDetails should send every parameter set")
	})

	t.Run("order not found", func(t *testing.T) {
		t.Parallel()
		ex, _ := spotTestServer(t, `{"code":100404,"msg":"order not exist"}`)
		_, err := ex.GetSpotOrderDetails(t.Context(), &SpotOrderDetailsRequest{Symbol: currency.NewBTCUSDT(), OrderID: 1})
		assert.ErrorIs(t, err, order.ErrOrderNotFound, "GetSpotOrderDetails should map code 100404 to ErrOrderNotFound")
	})

	t.Run("reply without an order ID", func(t *testing.T) {
		t.Parallel()
		ex, _ := spotTestServer(t, `{"code":0,"msg":"","data":{}}`)
		_, err := ex.GetSpotOrderDetails(t.Context(), &SpotOrderDetailsRequest{Symbol: currency.NewBTCUSDT(), OrderID: 1})
		assert.ErrorIs(t, err, errSpotOrderIDMissing, "GetSpotOrderDetails should reject a reply without an order ID")
	})

	t.Run("documented example", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			t.Skip(spotDocumentedOrderSkip)
		}
		result, err := e.GetSpotOrderDetails(t.Context(), &SpotOrderDetailsRequest{Symbol: currency.NewPair(currency.NewCode("BRG"), currency.USDT), OrderID: 1735965009395131234})
		require.NoError(t, err, "GetSpotOrderDetails must not error")
		exp := &SpotOrderDetailResponse{
			Symbol:           "BRG-USDT",
			OrderID:          1735965009395131100,
			Price:            0.0005027,
			OriginalQuantity: 4038,
			Status:           SpotOrderStatusPending,
			Type:             OrderTypeLimit,
			Side:             SideBuy,
			Time:             types.Time(time.UnixMilli(1702721285567)),
			UpdateTime:       types.Time(time.UnixMilli(1702721285567)),
			FeeAsset:         currency.NewCode("BRG"),
			AveragePrice:     0.0005028,
		}
		assert.Equal(t, exp, result, "GetSpotOrderDetails should decode every field")
	})
}

func TestGetSpotOpenOrders(t *testing.T) {
	t.Parallel()
	t.Run("every symbol request", func(t *testing.T) {
		t.Parallel()
		ex, requests := spotTestServer(t, `{"code":0,"msg":"","data":{"orders":[]}}`)
		_, err := ex.GetSpotOpenOrders(t.Context(), currency.EMPTYPAIR)
		require.NoError(t, err, "GetSpotOpenOrders must not error")
		r := spotReceiveRequest(t, requests)
		assert.Equal(t, http.MethodGet, r.method, "method should be GET")
		assert.Equal(t, "/openApi/spot/v1/trade/openOrders", r.path, "path should be the open orders endpoint")
		verifyQuerySignature(t, r.rawQuery)
		assert.Equal(t, url.Values{"recvWindow": {"5000"}}, r.params, "no symbol should be sent for every open order")
	})

	t.Run("symbol formatting error", func(t *testing.T) {
		t.Parallel()
		_, err := new(Exchange).GetSpotOpenOrders(t.Context(), currency.NewBTCUSDT())
		assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSpotOpenOrders should return the symbol formatting error")
	})

	t.Run("documented example", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetSpotOpenOrders(t.Context(), currency.NewPair(currency.BNB, currency.USDC))
		require.NoError(t, err, "GetSpotOpenOrders must not error")
		if mockTests {
			exp := []SpotOrderDetailResponse{
				{
					Symbol:           "BNB-USDC",
					OrderID:          1735930294290081300,
					Price:            255.27,
					OriginalQuantity: 0.16261,
					Status:           SpotOrderStatusPending,
					Type:             OrderTypeLimit,
					Side:             SideSell,
					Time:             types.Time(time.UnixMilli(1702713008841)),
					UpdateTime:       types.Time(time.UnixMilli(1702713008841)),
				},
			}
			assert.Equal(t, exp, result, "GetSpotOpenOrders should decode every field")
		}
	})
}

func TestGetSpotOrderHistory(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *SpotOrderHistoryRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "page size above 100", req: &SpotOrderHistoryRequest{PageSize: 101}, err: errSpotPageLimitExceeded},
		{name: "default page size past the window", req: &SpotOrderHistoryRequest{PageIndex: 101}, err: errSpotPageLimitExceeded},
		{name: "page past the window", req: &SpotOrderHistoryRequest{PageIndex: 201, PageSize: 50}, err: errSpotPageLimitExceeded},
		{name: "start after end", req: &SpotOrderHistoryRequest{StartTime: time.UnixMilli(2000), EndTime: time.UnixMilli(1000)}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetSpotOrderHistory(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetSpotOrderHistory should reject the request")
		})
	}

	t.Run("symbol formatting error", func(t *testing.T) {
		t.Parallel()
		_, err := new(Exchange).GetSpotOrderHistory(t.Context(), &SpotOrderHistoryRequest{Symbol: currency.NewBTCUSDT()})
		assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSpotOrderHistory should return the symbol formatting error")
	})

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex, requests := spotTestServer(t, `{"code":0,"msg":"","data":{"orders":[]}}`)
		_, err := ex.GetSpotOrderHistory(t.Context(), &SpotOrderHistoryRequest{
			OrderID:   1735966927102231300,
			PageIndex: 200,
			PageSize:  50,
			Status:    SpotOrderStatusFilled,
			Type:      OrderTypeMarket,
		})
		require.NoError(t, err, "GetSpotOrderHistory must not error at the end of the window")
		r := spotReceiveRequest(t, requests)
		assert.Equal(t, http.MethodGet, r.method, "method should be GET")
		assert.Equal(t, "/openApi/spot/v1/trade/historyOrders", r.path, "path should be the order history endpoint")
		verifyQuerySignature(t, r.rawQuery)
		exp := url.Values{
			"orderId":    {"1735966927102231300"},
			"pageIndex":  {"200"},
			"pageSize":   {"50"},
			"status":     {"FILLED"},
			"type":       {"MARKET"},
			"recvWindow": {"5000"},
		}
		assert.Equal(t, exp, r.params, "GetSpotOrderHistory should send every parameter set")
	})

	t.Run("documented example", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetSpotOrderHistory(t.Context(), &SpotOrderHistoryRequest{
			Symbol:    currency.NewPair(currency.NewCode("SWCH"), currency.USDT),
			StartTime: time.UnixMilli(1702720925417),
			EndTime:   time.UnixMilli(1702721825418),
			PageIndex: 1,
			PageSize:  100,
		})
		require.NoError(t, err, "GetSpotOrderHistory must not error")
		if mockTests {
			exp := []SpotOrderDetailResponse{
				{
					Symbol:                   "SWCH-USDT",
					OrderID:                  1735966927102231300,
					Price:                    0.8548,
					OriginalQuantity:         5.9,
					ExecutedQuantity:         1.1,
					CummulativeQuoteQuantity: 0.9402800000000001,
					Status:                   SpotOrderStatusCancelled,
					Type:                     OrderTypeLimit,
					Side:                     SideSell,
					Time:                     types.Time(time.UnixMilli(1702721743000)),
					UpdateTime:               types.Time(time.UnixMilli(1702721743000)),
					AveragePrice:             0.8881,
				},
			}
			assert.Equal(t, exp, result, "GetSpotOrderHistory should decode every field")
		}
	})
}

func TestGetSpotTransactionDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotTransactionDetails(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotTransactionDetails must reject a nil request")
	_, err = e.GetSpotTransactionDetails(t.Context(), &SpotTransactionDetailsRequest{StartTime: time.UnixMilli(2000), EndTime: time.UnixMilli(1000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSpotTransactionDetails must reject a start after the end")
	_, err = new(Exchange).GetSpotTransactionDetails(t.Context(), &SpotTransactionDetailsRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSpotTransactionDetails must return the symbol formatting error")

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex, requests := spotTestServer(t, `{"code":0,"msg":"","data":{"fills":[]}}`)
		_, err := ex.GetSpotTransactionDetails(t.Context(), &SpotTransactionDetailsRequest{
			Symbol:    currency.NewBTCUSDT(),
			StartTime: time.UnixMilli(1704960000000),
			EndTime:   time.UnixMilli(1704970000000),
			FromID:    36767057,
			Limit:     100,
		})
		require.NoError(t, err, "GetSpotTransactionDetails must not error")
		r := spotReceiveRequest(t, requests)
		assert.Equal(t, http.MethodGet, r.method, "method should be GET")
		assert.Equal(t, "/openApi/spot/v1/trade/myTrades", r.path, "path should be the fills endpoint")
		verifyQuerySignature(t, r.rawQuery)
		exp := url.Values{
			"symbol":     {"BTC-USDT"},
			"startTime":  {"1704960000000"},
			"endTime":    {"1704970000000"},
			"fromId":     {"36767057"},
			"limit":      {"100"},
			"recvWindow": {"5000"},
		}
		assert.Equal(t, exp, r.params, "GetSpotTransactionDetails should send every parameter set")
	})

	t.Run("request error", func(t *testing.T) {
		t.Parallel()
		ex, _ := spotTestServer(t, `{"code":100500,"msg":"busy"}`)
		result, err := ex.GetSpotTransactionDetails(t.Context(), &SpotTransactionDetailsRequest{Symbol: currency.NewBTCUSDT()})
		assert.ErrorIs(t, err, errAPIResponse, "GetSpotTransactionDetails should return the request's error")
		assert.Nil(t, result, "GetSpotTransactionDetails should return no fills for a failed request")
	})

	t.Run("documented example", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		result, err := e.GetSpotTransactionDetails(t.Context(), &SpotTransactionDetailsRequest{Symbol: currency.NewBTCUSDT(), OrderID: 1745362930595004400, Limit: 10})
		require.NoError(t, err, "GetSpotTransactionDetails must not error")
		if mockTests {
			exp := []SpotFillResponse{
				{
					Symbol:          "BTC-USDT",
					ID:              36767057,
					OrderID:         1745362930595004400,
					Price:           46820.155,
					Quantity:        0.1430254,
					QuoteQuantity:   6696.471396937,
					Commission:      -0.000046483255,
					CommissionAsset: currency.BTC,
					Time:            types.Time(time.UnixMilli(1704961925000)),
					IsBuyer:         true,
				},
				{
					Symbol:          "BTC-USDT",
					ID:              36767058,
					OrderID:         1745362930595004400,
					Price:           46820.155,
					Quantity:        0.0003844,
					QuoteQuantity:   17.997667582000002,
					Commission:      -1.2493e-7,
					CommissionAsset: currency.BTC,
					Time:            types.Time(time.UnixMilli(1704961925000)),
					IsBuyer:         true,
				},
			}
			assert.Equal(t, exp, result, "GetSpotTransactionDetails should decode every field")
		}
	})
}

func TestGetSpotCommissionRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotCommissionRate(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSpotCommissionRate must reject an empty pair")
	_, err = new(Exchange).GetSpotCommissionRate(t.Context(), currency.NewBTCUSDT())
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetSpotCommissionRate must return the symbol formatting error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSpotCommissionRate(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "GetSpotCommissionRate must not error")
	if mockTests {
		assert.Equal(t, &SpotCommissionRateResponse{TakerCommissionRate: 0.000325, MakerCommissionRate: 0.0001}, result, "GetSpotCommissionRate should decode every field")
	}
}

func TestCancelAllSpotOrdersAfter(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllSpotOrdersAfter(t.Context(), "", 10*time.Second)
	require.ErrorIs(t, err, errSpotCancelAllAfterTypeEmpty, "CancelAllSpotOrdersAfter must reject an empty type")
	for _, timeout := range []time.Duration{0, -time.Second, 1500 * time.Millisecond} {
		_, err = e.CancelAllSpotOrdersAfter(t.Context(), CancelAllAfterTypeActivate, timeout)
		require.ErrorIsf(t, err, errSpotInvalidTimeout, "CancelAllSpotOrdersAfter must reject a timeout of %s", timeout)
	}

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex, requests := spotTestServer(t, `{"code":0,"msg":"","data":{"triggerTime":0,"status":"CLOSED","note":""}}`)
		_, err := ex.CancelAllSpotOrdersAfter(t.Context(), CancelAllAfterTypeClose, 2*time.Minute)
		require.NoError(t, err, "CancelAllSpotOrdersAfter must not error")
		r := spotReceiveRequest(t, requests)
		assert.Equal(t, http.MethodPost, r.method, "method should be POST")
		assert.Equal(t, "/openApi/spot/v1/trade/cancelAllAfter", r.path, "path should be the cancel all after endpoint")
		verifyQuerySignature(t, r.rawQuery)
		assert.Equal(t, url.Values{"type": {"CLOSE"}, "timeOut": {"120"}, "recvWindow": {"5000"}}, r.params, "CancelAllSpotOrdersAfter should send the timeout in seconds")
	})

	t.Run("documented example", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
			t.Cleanup(func() {
				// The countdown would otherwise cancel every open spot order once it ends
				_, err := e.CancelAllSpotOrdersAfter(context.Background(), CancelAllAfterTypeClose, 10*time.Second)
				assert.NoError(t, err, "CancelAllSpotOrdersAfter should stop the countdown")
			})
		}
		result, err := e.CancelAllSpotOrdersAfter(t.Context(), CancelAllAfterTypeActivate, 10*time.Second)
		require.NoError(t, err, "CancelAllSpotOrdersAfter must not error")
		if mockTests {
			exp := &CancelAllSpotOrdersAfterResponse{
				TriggerTime: types.Time(time.Unix(1710389137, 0)),
				Status:      CancelAllAfterStatusActivated,
				Note:        "All your spot pending orders will be closed automatically at 2024-03-14 04:05:37 UTC(+0),before that you can cancel the timer, or extend triggerTime time by this request",
			}
			assert.Equal(t, exp, result, "CancelAllSpotOrdersAfter should decode every field")
			return
		}
		assert.Equal(t, CancelAllAfterStatusActivated, result.Status, "Status should be ACTIVATED")
	})
}

func TestPlaceSpotOCOOrder(t *testing.T) {
	t.Parallel()
	valid := PlaceSpotOCOOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Quantity: 0.001, LimitPrice: 48000, OrderPrice: 88000, TriggerPrice: 87000}
	for _, tc := range []struct {
		name   string
		modify func(*PlaceSpotOCOOrderRequest)
		err    error
	}{
		{name: "empty pair", modify: func(r *PlaceSpotOCOOrderRequest) { r.Symbol = currency.EMPTYPAIR }, err: currency.ErrCurrencyPairEmpty},
		{name: "no side", modify: func(r *PlaceSpotOCOOrderRequest) { r.Side = "" }, err: order.ErrSideIsInvalid},
		{name: "no quantity", modify: func(r *PlaceSpotOCOOrderRequest) { r.Quantity = 0 }, err: order.ErrAmountIsInvalid},
		{name: "no limit price", modify: func(r *PlaceSpotOCOOrderRequest) { r.LimitPrice = 0 }, err: errSpotPriceRequired},
		{name: "no order price", modify: func(r *PlaceSpotOCOOrderRequest) { r.OrderPrice = 0 }, err: errSpotPriceRequired},
		{name: "no trigger price", modify: func(r *PlaceSpotOCOOrderRequest) { r.TriggerPrice = 0 }, err: errSpotPriceRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := valid
			tc.modify(&req)
			_, err := e.PlaceSpotOCOOrder(t.Context(), &req)
			assert.ErrorIs(t, err, tc.err, "PlaceSpotOCOOrder should reject an incomplete request")
		})
	}

	t.Run("nil request", func(t *testing.T) {
		t.Parallel()
		_, err := e.PlaceSpotOCOOrder(t.Context(), nil)
		assert.ErrorIs(t, err, common.ErrNilPointer, "PlaceSpotOCOOrder should reject a nil request")
	})

	t.Run("symbol formatting error", func(t *testing.T) {
		t.Parallel()
		req := valid
		_, err := new(Exchange).PlaceSpotOCOOrder(t.Context(), &req)
		assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "PlaceSpotOCOOrder should return the symbol formatting error")
	})

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex, requests := spotTestServer(t, `{"code":0,"msg":"","data":[]}`)
		req := valid
		_, err := ex.PlaceSpotOCOOrder(t.Context(), &req)
		require.NoError(t, err, "PlaceSpotOCOOrder must not error")
		r := spotReceiveRequest(t, requests)
		assert.Equal(t, http.MethodPost, r.method, "method should be POST")
		assert.Equal(t, "/openApi/spot/v1/oco/order", r.path, "path should be the OCO order endpoint")
		verifyQuerySignature(t, r.rawQuery)
		exp := url.Values{
			"symbol":       {"BTC-USDT"},
			"side":         {"BUY"},
			"quantity":     {"0.001"},
			"limitPrice":   {"48000"},
			"orderPrice":   {"88000"},
			"triggerPrice": {"87000"},
			"recvWindow":   {"5000"},
		}
		assert.Equal(t, exp, r.params, "PlaceSpotOCOOrder should leave unset client IDs out")
	})

	t.Run("documented example", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)
		}
		req := valid
		req.ListClientOrderID, req.AboveClientOrderID, req.BelowClientOrderID = "12345610030", "12345610031", "12345610031"
		result, err := e.PlaceSpotOCOOrder(t.Context(), &req)
		require.NoError(t, err, "PlaceSpotOCOOrder must not error")
		if mockTests {
			exp := []SpotOCOOrderResponse{
				{
					OrderID:       1827963624962916352,
					ClientOrderID: "12345610034",
					Symbol:        "BTC-USDT",
					OrderType:     OCOOrderTypeLimit,
					Side:          SideBuy,
					Price:         48000,
					Quantity:      0.001,
					OrderListID:   "12345610033",
				},
				{
					OrderID:       1827963624962916353,
					ClientOrderID: "12345610035",
					Symbol:        "BTC-USDT",
					OrderType:     OCOOrderTypeStopLimit,
					Side:          SideBuy,
					TriggerPrice:  87000,
					Price:         88000,
					Quantity:      0.001,
					OrderListID:   "12345610033",
				},
			}
			assert.Equal(t, exp, result, "PlaceSpotOCOOrder should decode every field")
			return
		}
		assert.Len(t, result, 2, "PlaceSpotOCOOrder should return the list's two orders")
	})
}

func TestCancelSpotOCOOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.CancelSpotOCOOrderList(t.Context(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelSpotOCOOrderList must reject a request without an order ID")

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex, requests := spotTestServer(t, `{"code":0,"msg":"","data":{"orderId":"1","clientOrderId":""}}`)
		_, err := ex.CancelSpotOCOOrderList(t.Context(), 1827980248763858944, "")
		require.NoError(t, err, "CancelSpotOCOOrderList must not error")
		r := spotReceiveRequest(t, requests)
		assert.Equal(t, http.MethodPost, r.method, "method should be POST")
		assert.Equal(t, "/openApi/spot/v1/oco/cancel", r.path, "path should be the OCO cancel endpoint")
		verifyQuerySignature(t, r.rawQuery)
		assert.Equal(t, url.Values{"orderId": {"1827980248763858944"}, "recvWindow": {"5000"}}, r.params, "CancelSpotOCOOrderList should send the order ID")
	})

	t.Run("order not found", func(t *testing.T) {
		t.Parallel()
		ex, _ := spotTestServer(t, `{"code":100404,"msg":"order not exist"}`)
		_, err := ex.CancelSpotOCOOrderList(t.Context(), 1, "")
		assert.ErrorIs(t, err, order.ErrOrderNotFound, "CancelSpotOCOOrderList should map code 100404 to ErrOrderNotFound")
	})

	t.Run("documented example", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			t.Skip(spotDocumentedOrderSkip)
		}
		result, err := e.CancelSpotOCOOrderList(t.Context(), 1735964079647111280, "123456789")
		require.NoError(t, err, "CancelSpotOCOOrderList must not error")
		assert.Equal(t, &CancelSpotOCOOrderListResponse{OrderID: 1827980248763858944, ClientOrderID: "123456789"}, result, "CancelSpotOCOOrderList should decode every field")
	})
}

func TestGetSpotOCOOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotOCOOrderList(t.Context(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetSpotOCOOrderList must reject a request without an order ID")

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex, requests := spotTestServer(t, `{"code":0,"msg":"","data":[]}`)
		_, err := ex.GetSpotOCOOrderList(t.Context(), 0, "12345610027")
		require.NoError(t, err, "GetSpotOCOOrderList must not error")
		r := spotReceiveRequest(t, requests)
		assert.Equal(t, http.MethodGet, r.method, "method should be GET")
		assert.Equal(t, "/openApi/spot/v1/oco/orderList", r.path, "path should be the OCO order list endpoint")
		verifyQuerySignature(t, r.rawQuery)
		assert.Equal(t, url.Values{"clientOrderId": {"12345610027"}, "recvWindow": {"5000"}}, r.params, "GetSpotOCOOrderList should send the client order ID")
	})

	t.Run("order not found", func(t *testing.T) {
		t.Parallel()
		ex, _ := spotTestServer(t, `{"code":100404,"msg":"order not exist"}`)
		_, err := ex.GetSpotOCOOrderList(t.Context(), 1, "")
		assert.ErrorIs(t, err, order.ErrOrderNotFound, "GetSpotOCOOrderList should map code 100404 to ErrOrderNotFound")
	})

	t.Run("documented example", func(t *testing.T) {
		t.Parallel()
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if !mockTests {
			t.Skip(spotDocumentedOrderSkip)
		}
		result, err := e.GetSpotOCOOrderList(t.Context(), 1827968197060460545, "12345610027")
		require.NoError(t, err, "GetSpotOCOOrderList must not error")
		exp := []SpotOCOOrderResponse{
			{
				TransactionTime: types.Time(time.UnixMilli(1724656554890)),
				OrderID:         1827968197060460545,
				Symbol:          "BTC-USDT",
				OrderType:       OCOOrderTypeStopLimit,
				Side:            SideBuy,
				TriggerPrice:    87000,
				Price:           88000,
				Quantity:        0.001,
				OrderListID:     "1827968196914479104",
			},
		}
		assert.Equal(t, exp, result, "GetSpotOCOOrderList should decode every field")
	})
}

func TestGetSpotOpenOCOOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotOpenOCOOrders(t.Context(), 0, 100)
	require.ErrorIs(t, err, errSpotPageIndexRequired, "GetSpotOpenOCOOrders must reject a request without a page index")
	_, err = e.GetSpotOpenOCOOrders(t.Context(), 1, 0)
	require.ErrorIs(t, err, errSpotPageSizeRequired, "GetSpotOpenOCOOrders must reject a request without a page size")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSpotOpenOCOOrders(t.Context(), 1, 100)
	require.NoError(t, err, "GetSpotOpenOCOOrders must not error")
	if mockTests {
		exp := []SpotOCOOrderResponse{
			{
				TransactionTime: types.Time(time.UnixMilli(1724656554890)),
				OrderID:         1827968197060460545,
				Symbol:          "BTC-USDT",
				OrderType:       OCOOrderTypeStopLimit,
				Side:            SideBuy,
				TriggerPrice:    87000,
				Price:           88000,
				Quantity:        0.001,
				OrderListID:     "1827968196914479104",
				Status:          SpotOrderStatusNew,
			},
			{
				TransactionTime: types.Time(time.UnixMilli(1724656554890)),
				OrderID:         1827968197060460544,
				Symbol:          "BTC-USDT",
				OrderType:       OCOOrderTypeLimit,
				Side:            SideBuy,
				Price:           48000,
				Quantity:        0.001,
				OrderListID:     "1827968196914479104",
				Status:          SpotOrderStatusPending,
			},
		}
		assert.Equal(t, exp, result, "GetSpotOpenOCOOrders should decode every field")
	}
}

func TestGetSpotOCOOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotOCOOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotOCOOrderHistory must reject a nil request")
	_, err = e.GetSpotOCOOrderHistory(t.Context(), &SpotOCOOrderHistoryRequest{PageSize: 100})
	require.ErrorIs(t, err, errSpotPageIndexRequired, "GetSpotOCOOrderHistory must reject a request without a page index")
	_, err = e.GetSpotOCOOrderHistory(t.Context(), &SpotOCOOrderHistoryRequest{PageIndex: 1, PageSize: 100, StartTime: time.UnixMilli(2000), EndTime: time.UnixMilli(1000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSpotOCOOrderHistory must reject a start after the end")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err := e.GetSpotOCOOrderHistory(t.Context(), &SpotOCOOrderHistoryRequest{
		PageIndex: 1,
		PageSize:  100,
		StartTime: time.UnixMilli(1724256000000),
		EndTime:   time.UnixMilli(1724342400000),
	})
	require.NoError(t, err, "GetSpotOCOOrderHistory must not error")
	if mockTests {
		exp := []SpotOCOOrderResponse{
			{
				TransactionTime: types.Time(time.UnixMilli(1724297880000)),
				OrderID:         1826458150027395073,
				Symbol:          "BTC-USDT",
				OrderType:       OCOOrderTypeStopLimit,
				Side:            SideBuy,
				TriggerPrice:    64000,
				Price:           65000,
				Quantity:        0.001,
				OrderListID:     "1826458148019142656",
				Status:          SpotOrderStatusFailed,
			},
			{
				TransactionTime: types.Time(time.UnixMilli(1724297880000)),
				OrderID:         1826458150027395072,
				Symbol:          "BTC-USDT",
				OrderType:       OCOOrderTypeLimit,
				Side:            SideBuy,
				Price:           48000,
				Quantity:        0.001,
				OrderListID:     "1826458148019142656",
				Status:          SpotOrderStatusCancelled,
			},
		}
		assert.Equal(t, exp, result, "GetSpotOCOOrderHistory should decode every field")
	}
}

func TestSpotValidateOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *PlaceSpotOrderRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "empty pair", req: &PlaceSpotOrderRequest{Side: SideBuy, Type: OrderTypeMarket, Quantity: 1}, err: currency.ErrCurrencyPairEmpty},
		{name: "empty side", req: &PlaceSpotOrderRequest{Symbol: currency.NewBTCUSDT(), Type: OrderTypeMarket, Quantity: 1}, err: order.ErrSideIsInvalid},
		{name: "empty type", req: &PlaceSpotOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Quantity: 1}, err: order.ErrTypeIsInvalid},
		{name: "no amount", req: &PlaceSpotOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Type: OrderTypeMarket, Quantity: -1}, err: order.ErrAmountIsInvalid},
		{name: "quantity", req: &PlaceSpotOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideSell, Type: OrderTypeMarket, Quantity: 1}},
		{name: "quote order quantity", req: &PlaceSpotOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Type: OrderTypeMarket, QuoteOrderQuantity: 100}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := spotValidateOrder(tc.req)
			if tc.err == nil {
				assert.NoError(t, err, "spotValidateOrder should accept a complete order")
				return
			}
			assert.ErrorIs(t, err, tc.err, "spotValidateOrder should reject an incomplete order")
		})
	}
}

func TestSpotBatchOrderItem(t *testing.T) {
	t.Parallel()
	req := &PlaceSpotOrderRequest{
		Symbol:             currency.NewBTCUSDT(),
		Side:               SideBuy,
		Type:               OrderTypeTakeStopLimit,
		StopPrice:          70000,
		Quantity:           0.1,
		QuoteOrderQuantity: 7000,
		Price:              69000,
		NewClientOrderID:   "gct_1",
		TimeInForce:        TimeInForceFillOrKill,
	}
	item, err := e.spotBatchOrderItem(req)
	require.NoError(t, err, "spotBatchOrderItem must not error")
	exp := SpotBatchOrderItemRequest{
		Symbol:             "BTC-USDT",
		Side:               SideBuy,
		Type:               OrderTypeTakeStopLimit,
		StopPrice:          70000,
		Quantity:           0.1,
		QuoteOrderQuantity: 7000,
		Price:              69000,
		NewClientOrderID:   "gct_1",
		TimeInForce:        TimeInForceFillOrKill,
	}
	assert.Equal(t, exp, item, "spotBatchOrderItem should copy every field with the symbol formatted")

	_, err = e.spotBatchOrderItem(&PlaceSpotOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Quantity: 1})
	assert.ErrorIs(t, err, order.ErrTypeIsInvalid, "spotBatchOrderItem should validate the order")

	_, err = new(Exchange).spotBatchOrderItem(req)
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "spotBatchOrderItem should return the format error")
}

func TestSpotOCOOrderParams(t *testing.T) {
	t.Parallel()
	_, err := spotOCOOrderParams(0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "spotOCOOrderParams must reject neither ID")
	for _, tc := range []struct {
		orderID       uint64
		clientOrderID string
		exp           url.Values
	}{
		{orderID: 1827968197060460545, exp: url.Values{"orderId": {"1827968197060460545"}}},
		{clientOrderID: "12345610027", exp: url.Values{"clientOrderId": {"12345610027"}}},
		{orderID: 1, clientOrderID: "2", exp: url.Values{"orderId": {"1"}, "clientOrderId": {"2"}}},
	} {
		params, err := spotOCOOrderParams(tc.orderID, tc.clientOrderID)
		require.NoError(t, err, "spotOCOOrderParams must not error")
		assert.Equal(t, tc.exp, params, "spotOCOOrderParams should set the IDs given")
	}
}

func TestSpotPageParams(t *testing.T) {
	t.Parallel()
	_, err := spotPageParams(0, 100)
	require.ErrorIs(t, err, errSpotPageIndexRequired, "spotPageParams must require a page index")
	_, err = spotPageParams(1, 0)
	require.ErrorIs(t, err, errSpotPageSizeRequired, "spotPageParams must require a page size")
	params, err := spotPageParams(2, 50)
	require.NoError(t, err, "spotPageParams must not error")
	assert.Equal(t, url.Values{"pageIndex": {"2"}, "pageSize": {"50"}}, params, "spotPageParams should set both page parameters")
}

func TestSpotOrderNotFoundError(t *testing.T) {
	t.Parallel()
	assert.NoError(t, spotOrderNotFoundError(nil), "spotOrderNotFoundError should leave no error alone")

	other := errors.New("transport failure")
	assert.Equal(t, other, spotOrderNotFoundError(other), "spotOrderNotFoundError should leave other errors alone")

	notFound := &APIError{Code: 100404, Message: "order not exist"}
	err := spotOrderNotFoundError(notFound)
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "spotOrderNotFoundError should map code 100404 to ErrOrderNotFound")
	assert.ErrorIs(t, err, errAPIResponse, "spotOrderNotFoundError should keep the API error")
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "the API error must stay reachable")
	assert.Equal(t, notFound, apiErr, "the API error should be the original")

	err = spotOrderNotFoundError(&APIError{Code: 100400, Message: "bad request"})
	assert.NotErrorIs(t, err, order.ErrOrderNotFound, "spotOrderNotFoundError should leave other codes unmapped")
}

func TestSpotCancelReplaceOutcome(t *testing.T) {
	t.Parallel()
	resp := &CancelReplaceSpotOrderResponse{OpenResult: SpotCancelReplaceResult{Result: true}}
	result, err := spotCancelReplaceOutcome(resp, nil)
	require.NoError(t, err, "spotCancelReplaceOutcome must not add an error")
	assert.Same(t, resp, result, "spotCancelReplaceOutcome should return a successful response unchanged")

	other := errors.New("transport failure")
	result, err = spotCancelReplaceOutcome(nil, other)
	assert.ErrorIs(t, err, other, "spotCancelReplaceOutcome should return a non-API error")
	assert.Nil(t, result, "spotCancelReplaceOutcome should return no outcome without one")

	result, err = spotCancelReplaceOutcome(nil, &APIError{Code: 100400})
	assert.ErrorIs(t, err, errAPIResponse, "spotCancelReplaceOutcome should return an API error without data")
	assert.Nil(t, result, "spotCancelReplaceOutcome should return no outcome without data")

	result, err = spotCancelReplaceOutcome(nil, &APIError{Code: 100400, Data: json.RawMessage(`{"cancelResult":{"code":100404,"msg":"order not exist","result":false},"openResult":{"code":0,"msg":"","result":false}}`)})
	assert.ErrorIs(t, err, errAPIResponse, "spotCancelReplaceOutcome should return the API error")
	assert.Equal(t, &CancelReplaceSpotOrderResponse{CancelResult: SpotCancelReplaceResult{Code: 100404, Message: "order not exist"}}, result, "spotCancelReplaceOutcome should decode the outcome the error carries")

	result, err = spotCancelReplaceOutcome(nil, &APIError{Code: 100400, Data: json.RawMessage(`["unexpected"]`)})
	assert.ErrorIs(t, err, errAPIResponse, "spotCancelReplaceOutcome should keep the API error")
	assert.ErrorContains(t, err, "error decoding cancel-replace outcome", "spotCancelReplaceOutcome should report the decoding failure")
	assert.Nil(t, result, "spotCancelReplaceOutcome should return no outcome it cannot decode")
}
