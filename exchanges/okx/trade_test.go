package okx

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// orderActionReply is a successful order action row for the request pins
var orderActionReply = []map[string]string{{"ordId": "1", "sCode": "0", "sMsg": ""}}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &PlaceOrderRequest{
		ReduceOnly: true,
		AssetType:  asset.Margin,
	}
	_, err = e.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	_, err = e.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = order.Buy.Lower()
	_, err = e.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	arg.TradeMode = TradeModeCross
	_, err = e.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = order.Limit.String()
	_, err = e.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.AssetType = asset.Futures
	arg.PositionSide = "reverse"
	_, err = e.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.PositionSide = "long"
	_, err = e.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.Size = 1
	arg.TargetCurrency = "abcd"
	_, err = e.PlaceOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errCurrencyQuantityTypeRequired)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *PlaceOrderRequest
			body string
		}{
			{
				name: "required parameters only",
				arg:  &PlaceOrderRequest{InstrumentID: mainPair.String(), TradeMode: TradeModeCash, Side: order.Buy.Lower(), OrderType: orderMarket, Size: 100},
				body: `{"instId":"BTC-USDT","tdMode":"cash","side":"buy","ordType":"market","sz":"100"}`,
			},
			{
				name: "every optional parameter",
				arg: &PlaceOrderRequest{
					AssetType:               asset.Spot,
					InstrumentID:            mainPair.String(),
					TradeMode:               TradeModeCross,
					Currency:                currency.USDT,
					ClientOrderID:           "b15",
					OrderTag:                "tag1",
					Side:                    "BUY",
					PositionSide:            positionSideNet,
					OrderType:               "LIMIT",
					Size:                    2,
					Price:                   2.15,
					Outcome:                 "yes",
					PriceUSD:                3612.5,
					PriceVolatility:         0.55,
					ReduceOnly:              true,
					TargetCurrency:          targetCurrencyBase,
					BanAmend:                true,
					PriceAmendType:          "1",
					TradeQuoteCurrency:      currency.USDC,
					SlippagePercentage:      0.0123,
					SelfTradePreventionMode: "cancel_both",
					RPITakerAccess:          true,
					RPIPriceRound:           true,
					AttachAlgoOrders: []AttachAlgoOrderRequest{{
						AttachAlgoClientOrderID:    "attach1",
						TakeProfitTriggerPrice:     2.5,
						TakeProfitTriggerRatio:     0.3,
						TakeProfitOrderPrice:       -1,
						TakeProfitOrderKind:        "condition",
						StopLossTriggerPrice:       2,
						StopLossTriggerRatio:       0.1,
						StopLossOrderPrice:         -1,
						TakeProfitTriggerPriceType: "last",
						StopLossTriggerPriceType:   "mark",
						Size:                       1,
						AmendPriceOnTriggerType:    "1",
						CallbackRatio:              0.05,
						CallbackSpread:             0.1,
						ActivePrice:                2.2,
					}},
				},
				body: `{"instId":"BTC-USDT","tdMode":"cross","ccy":"USDT","clOrdId":"b15","tag":"tag1","side":"buy","posSide":"net","ordType":"limit","sz":"2","px":"2.15","outcome":"yes","pxUsd":"3612.5","pxVol":"0.55","reduceOnly":true,"tgtCcy":"base_ccy","banAmend":true,"pxAmendType":"1","tradeQuoteCcy":"USDC","slippagePct":"0.0123","stpMode":"cancel_both","rpiTakerAccess":true,"rpiPxRound":true,"attachAlgoOrds":[{"attachAlgoClOrdId":"attach1","tpTriggerPx":"2.5","tpTriggerRatio":"0.3","tpOrdPx":"-1","tpOrdKind":"condition","slTriggerPx":"2","slTriggerRatio":"0.1","slOrdPx":"-1","tpTriggerPxType":"last","slTriggerPxType":"mark","sz":"1","amendPxOnTriggerType":"1","callbackRatio":"0.05","callbackSpread":"0.1","activePx":"2.2"}]}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/trade/order", tc.body, orderActionReply)
				_, err := ex.PlaceOrder(t.Context(), tc.arg)
				require.NoError(t, err, "PlaceOrder must not error")
			})
		}
	})

	for _, tc := range []struct {
		name string
		arg  *PlaceOrderRequest
		exp  *OrderActionResponse
	}{
		{
			name: "margin order",
			arg: &PlaceOrderRequest{
				InstrumentID: "BTC-USDC",
				TradeMode:    TradeModeCross,
				Side:         order.Buy.String(),
				OrderType:    orderLimit,
				Size:         2.6,
				Price:        2.1,
				Currency:     currency.BTC,
				AssetType:    asset.Margin,
			},
			exp: &OrderActionResponse{OrderID: "312269865356374016", Timestamp: types.Time(time.UnixMilli(1695190491421))},
		},
		{
			name: "futures order with a position side",
			arg: &PlaceOrderRequest{
				InstrumentID: "BTC-USD-261225",
				TradeMode:    TradeModeCross,
				Side:         order.Buy.Lower(),
				PositionSide: positionSideLong,
				OrderType:    orderLimit,
				Size:         2.6,
				Price:        2.1,
				AssetType:    asset.Futures,
			},
			exp: &OrderActionResponse{OrderID: "312269865356374017", Timestamp: types.Time(time.UnixMilli(1695190491422))},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !mockTests {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
			}
			result, err := e.PlaceOrder(contextGenerate(), tc.arg)
			require.NoError(t, err, "PlaceOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "PlaceOrder should decode every field")
			} else {
				assert.NotEmpty(t, result.OrderID, "PlaceOrder should return the order ID")
			}
		})
	}
}

const placeMultipleOrderParamsJSON = `[{"instId":"BTC-USDT","tdMode":"cash","clOrdId":"b159","side":"buy","ordType":"limit","px":"2.15","sz":"2"},{"instId":"BTC-USDT","tdMode":"cash","clOrdId":"b15","side":"buy","ordType":"limit","px":"2.15","sz":"2"}]`

func TestPlaceMultipleOrders(t *testing.T) {
	t.Parallel()
	var params []PlaceOrderRequest
	require.NoError(t, json.Unmarshal([]byte(placeMultipleOrderParamsJSON), &params), "the batch parameters must decode")

	_, err := e.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequest{})
	require.ErrorIs(t, err, order.ErrSubmissionIsNil)

	arg := PlaceOrderRequest{
		ReduceOnly: true,
	}
	_, err = e.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequest{arg})
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	_, err = e.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequest{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "buy"
	_, err = e.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequest{arg})
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	arg.TradeMode = "cross"
	_, err = e.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequest{arg})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = orderLimit
	_, err = e.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequest{arg})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	arg.AssetType = asset.Futures
	arg.PositionSide = "reverse"
	_, err = e.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequest{arg})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.PositionSide = "long"
	_, err = e.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequest{arg})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	t.Run("limits the batch size", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}))
		args := make([]PlaceOrderRequest, 21)
		_, err := ex.PlaceMultipleOrders(t.Context(), args)
		assert.ErrorIs(t, err, errExceedLimit, "an oversized REST batch should be rejected before transmission")
	})

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/trade/batch-orders",
			`[{"instId":"BTC-USDT","tdMode":"cash","side":"buy","ordType":"market","sz":"100"},{"instId":"BTC-USDT-SWAP","tdMode":"cross","clOrdId":"b16","tag":"tag1","side":"sell","posSide":"short","ordType":"post_only","sz":"1","px":"95000","stpMode":"cancel_maker"}]`,
			[]map[string]string{{"ordId": "1", "sCode": "0"}, {"ordId": "2", "sCode": "0"}})
		_, err := ex.PlaceMultipleOrders(t.Context(), []PlaceOrderRequest{
			{InstrumentID: mainPair.String(), TradeMode: TradeModeCash, Side: order.Buy.Lower(), OrderType: orderMarket, Size: 100},
			{
				AssetType: asset.PerpetualSwap, InstrumentID: perpetualSwapPair.String(), TradeMode: TradeModeCross, ClientOrderID: "b16", OrderTag: "tag1",
				Side: order.Sell.Lower(), PositionSide: positionSideShort, OrderType: orderPostOnly, Size: 1, Price: 95000, SelfTradePreventionMode: "cancel_maker",
			},
		})
		require.NoError(t, err, "PlaceMultipleOrders must not error")
	})

	t.Run("charges each order to OKX's order budget", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			orders int
			exp    map[request.EndpointLimit]float64
		}{
			{"a lone order", 1, map[request.EndpointLimit]float64{placeOrderEPL: 1}},
			{"two orders", 2, map[request.EndpointLimit]float64{placeMultipleOrdersEPL: 2}},
			{"a full batch", maxOrdersPerBatch, map[request.EndpointLimit]float64{placeMultipleOrdersEPL: maxOrdersPerBatch}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, orderRowsHandler(t, "/trade/batch-orders", "ordId", tc.orders))
				pools := chargeToPools(t, ex)
				args := slices.Repeat([]PlaceOrderRequest{{InstrumentID: mainPair.String(), TradeMode: TradeModeCash, Side: order.Buy.Lower(), OrderType: orderMarket, Size: 100}}, tc.orders)
				_, err := ex.PlaceMultipleOrders(chargeContext(t), args)
				require.NoError(t, err, "PlaceMultipleOrders must not error")
				assert.Equal(t, tc.exp, takenTokens(pools), "PlaceMultipleOrders should take a token per order, a lone order's from the single order limit as OKX charges it")
			})
		}
	})

	t.Run("sends nothing its order budget cannot cover", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected request %s", r.URL.Path)
			http.NotFound(w, r)
		}))
		pools := chargeToPools(t, ex)
		require.True(t, pools[placeMultipleOrdersEPL].AllowN(time.Now(), orderBudgetPoolSize-1), "the pool must give out all but one of its tokens")
		_, err := ex.PlaceMultipleOrders(chargeContext(t), slices.Repeat([]PlaceOrderRequest{{InstrumentID: mainPair.String(), TradeMode: TradeModeCash, Side: order.Buy.Lower(), OrderType: orderMarket, Size: 100}}, 2))
		assert.ErrorIs(t, err, context.DeadlineExceeded, "PlaceMultipleOrders should fail when its budget cannot cover both orders in time")
		assert.Equal(t, 1.0, pools[placeMultipleOrdersEPL].Tokens(), "PlaceMultipleOrders should leave the budget it could not use")
	})

	t.Run("charges every attempt of a retried batch", func(t *testing.T) {
		t.Parallel()
		ex, pools, requests := newRetriedBatchExchange(t, "/trade/batch-orders", "ordId")
		_, err := ex.PlaceMultipleOrders(chargeContext(t), slices.Repeat([]PlaceOrderRequest{{InstrumentID: mainPair.String(), TradeMode: TradeModeCash, Side: order.Buy.Lower(), OrderType: orderMarket, Size: 100}}, 3))
		require.NoError(t, err, "PlaceMultipleOrders must not error")
		require.Equal(t, int32(2), requests.Load(), "the batch must be sent again after the venue refused it")
		assert.Equal(t, map[request.EndpointLimit]float64{placeMultipleOrdersEPL: 6}, takenTokens(pools), "PlaceMultipleOrders should take a token per order on every attempt")
	})

	t.Run("places every order", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		}
		result, err := e.PlaceMultipleOrders(contextGenerate(), params)
		require.NoError(t, err, "PlaceMultipleOrders must not error")
		if mockTests {
			exp := []OrderActionResponse{
				{OrderID: "12345689", ClientOrderID: "b159", Timestamp: types.Time(time.UnixMilli(1695190491421))},
				{OrderID: "12344", ClientOrderID: "b15", Timestamp: types.Time(time.UnixMilli(1695190491421))},
			}
			assert.Equal(t, exp, result, "PlaceMultipleOrders should decode every field")
		} else {
			assert.Len(t, result, len(params), "PlaceMultipleOrders should return a result for every order")
		}
	})

	t.Run("partial success reports the failed orders", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("the partially successful batch is a mocked reply")
		}
		result, err := e.PlaceMultipleOrders(contextGenerate(), []PlaceOrderRequest{
			{InstrumentID: mainPair.String(), TradeMode: TradeModeCash, ClientOrderID: "oktswap6", OrderTag: "batch1", Side: order.Buy.Lower(), OrderType: orderLimit, Price: 2.15, Size: 2},
			{InstrumentID: mainPair.String(), TradeMode: TradeModeCash, ClientOrderID: "oktswap7", OrderTag: "batch1", Side: order.Buy.Lower(), OrderType: orderLimit, Price: 2.15, Size: 2000},
		})
		require.ErrorIs(t, err, errPartialSuccess, "PlaceMultipleOrders must report the partial success")
		assert.ErrorContains(t, err, "51008", "PlaceMultipleOrders should report the failed order's status code")
		exp := []OrderActionResponse{
			{OrderID: "12345690", ClientOrderID: "oktswap6", Tag: "batch1", Timestamp: types.Time(time.UnixMilli(1695190491421))},
			{
				ClientOrderID: "oktswap7", Tag: "batch1", StatusCode: 51008, StatusMessage: "Order failed. Insufficient USDT balance in account",
				SubCode: "1000", Timestamp: types.Time(time.UnixMilli(1695190491421)),
			},
		}
		assert.Equal(t, exp, result, "PlaceMultipleOrders should decode every order's result")
	})

	t.Run("partial success returns every row beside the error", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/trade/batch-orders":
				_, _ = w.Write([]byte(`{"code":"2","msg":"Bulk operation partially succeeded.","data":[{"ordId":"1","sCode":"0","sMsg":""},{"ordId":"","sCode":"51008","sMsg":"Order failed. Insufficient USDT balance in account."}]}`))
			default:
				t.Errorf("unexpected request path %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		placed, err := ex.PlaceMultipleOrders(t.Context(), []PlaceOrderRequest{
			{InstrumentID: mainPair.String(), TradeMode: TradeModeCash, Side: order.Buy.Lower(), OrderType: orderLimit, Size: 1, Price: 1},
			{InstrumentID: mainPair.String(), TradeMode: TradeModeCash, Side: order.Buy.Lower(), OrderType: orderLimit, Size: 1, Price: 1},
		})
		require.ErrorIs(t, err, errPartialSuccess, "PlaceMultipleOrders must return the partial success")
		exp := []OrderActionResponse{
			{OrderID: "1"},
			{StatusCode: 51008, StatusMessage: "Order failed. Insufficient USDT balance in account."},
		}
		assert.Equal(t, exp, placed, "PlaceMultipleOrders should return every row beside the error")
	})
}

func TestCancelSingleOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelSingleOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.CancelSingleOrder(contextGenerate(), &CancelOrderRequest{})
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = e.CancelSingleOrder(contextGenerate(), &CancelOrderRequest{OrderID: "12321312312"})
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = e.CancelSingleOrder(contextGenerate(), &CancelOrderRequest{InstrumentID: mainPair.String()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *CancelOrderRequest
			body string
		}{
			{"order ID", &CancelOrderRequest{InstrumentID: mainPair.String(), OrderID: "2510789768709120"}, `{"instId":"BTC-USDT","ordId":"2510789768709120"}`},
			{"client order ID", &CancelOrderRequest{InstrumentID: mainPair.String(), ClientOrderID: "b15"}, `{"instId":"BTC-USDT","clOrdId":"b15"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/trade/cancel-order", tc.body, orderActionReply)
				_, err := ex.CancelSingleOrder(t.Context(), tc.arg)
				require.NoError(t, err, "CancelSingleOrder must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelSingleOrder(contextGenerate(), &CancelOrderRequest{InstrumentID: mainPair.String(), OrderID: "2510789768709120"})
	require.NoError(t, err, "CancelSingleOrder must not error")
	if mockTests {
		exp := &OrderActionResponse{OrderID: "2510789768709120", Timestamp: types.Time(time.UnixMilli(1695190491421))}
		assert.Equal(t, exp, result, "CancelSingleOrder should decode every field")
	} else {
		assert.NotEmpty(t, result.OrderID, "CancelSingleOrder should return the order ID")
	}
}

func TestCancelMultipleOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelMultipleOrders(contextGenerate(), []CancelOrderRequest{})
	require.ErrorIs(t, err, common.ErrEmptyParams)
	arg := CancelOrderRequest{}
	_, err = e.CancelMultipleOrders(contextGenerate(), []CancelOrderRequest{arg})
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	_, err = e.CancelMultipleOrders(contextGenerate(), []CancelOrderRequest{arg})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/trade/cancel-batch-orders",
			`[{"instId":"BTC-USDT","ordId":"2510789768709120"},{"instId":"BTC-USDT-SWAP","clOrdId":"b15"}]`,
			[]map[string]string{{"ordId": "2510789768709120", "sCode": "0"}, {"ordId": "2", "clOrdId": "b15", "sCode": "0"}})
		_, err := ex.CancelMultipleOrders(t.Context(), []CancelOrderRequest{
			{InstrumentID: mainPair.String(), OrderID: "2510789768709120"},
			{InstrumentID: perpetualSwapPair.String(), ClientOrderID: "b15"},
		})
		require.NoError(t, err, "CancelMultipleOrders must not error")
	})

	t.Run("charges each order to OKX's order budget", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			orders int
			exp    map[request.EndpointLimit]float64
		}{
			{"a lone order", 1, map[request.EndpointLimit]float64{cancelOrderEPL: 1}},
			{"two orders", 2, map[request.EndpointLimit]float64{cancelMultipleOrdersEPL: 2}},
			{"a full batch", maxOrdersPerBatch, map[request.EndpointLimit]float64{cancelMultipleOrdersEPL: maxOrdersPerBatch}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, orderRowsHandler(t, "/trade/cancel-batch-orders", "ordId", tc.orders))
				pools := chargeToPools(t, ex)
				args := make([]CancelOrderRequest, tc.orders)
				for i := range args {
					args[i] = CancelOrderRequest{InstrumentID: mainPair.String(), OrderID: strconv.Itoa(i + 1)}
				}
				_, err := ex.CancelMultipleOrders(chargeContext(t), args)
				require.NoError(t, err, "CancelMultipleOrders must not error")
				assert.Equal(t, tc.exp, takenTokens(pools), "CancelMultipleOrders should take a token per order, a lone order's from the single order limit as OKX charges it")
			})
		}
	})

	t.Run("sends nothing its order budget cannot cover", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected request %s", r.URL.Path)
			http.NotFound(w, r)
		}))
		pools := chargeToPools(t, ex)
		require.True(t, pools[cancelMultipleOrdersEPL].AllowN(time.Now(), orderBudgetPoolSize-1), "the pool must give out all but one of its tokens")
		_, err := ex.CancelMultipleOrders(chargeContext(t), []CancelOrderRequest{{InstrumentID: mainPair.String(), OrderID: "1"}, {InstrumentID: mainPair.String(), OrderID: "2"}})
		assert.ErrorIs(t, err, context.DeadlineExceeded, "CancelMultipleOrders should fail when its budget cannot cover both orders in time")
		assert.Equal(t, 1.0, pools[cancelMultipleOrdersEPL].Tokens(), "CancelMultipleOrders should leave the budget it could not use")
	})

	t.Run("charges every attempt of a retried batch", func(t *testing.T) {
		t.Parallel()
		ex, pools, requests := newRetriedBatchExchange(t, "/trade/cancel-batch-orders", "ordId")
		_, err := ex.CancelMultipleOrders(chargeContext(t), []CancelOrderRequest{{InstrumentID: mainPair.String(), OrderID: "1"}, {InstrumentID: mainPair.String(), OrderID: "2"}, {InstrumentID: mainPair.String(), OrderID: "3"}})
		require.NoError(t, err, "CancelMultipleOrders must not error")
		require.Equal(t, int32(2), requests.Load(), "the batch must be sent again after the venue refused it")
		assert.Equal(t, map[request.EndpointLimit]float64{cancelMultipleOrdersEPL: 6}, takenTokens(pools), "CancelMultipleOrders should take a token per order on every attempt")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelMultipleOrders(contextGenerate(), []CancelOrderRequest{{InstrumentID: mainPair.String(), OrderID: "2510789768709120"}})
	require.NoError(t, err, "CancelMultipleOrders must not error")
	if mockTests {
		exp := []*OrderActionResponse{{OrderID: "2510789768709120", Timestamp: types.Time(time.UnixMilli(1695190491421))}}
		assert.Equal(t, exp, result, "CancelMultipleOrders should decode every field")
	} else {
		assert.Len(t, result, 1, "CancelMultipleOrders should return a result for the order")
	}
}

func TestAmendOrderRequestValidate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  AmendOrderRequest
		err  error
	}{
		{"missing instrument ID", AmendOrderRequest{OrderID: "1", NewSize: 1}, errMissingInstrumentID},
		{"missing order IDs", AmendOrderRequest{InstrumentID: mainPair.String(), NewSize: 1}, order.ErrOrderIDNotSet},
		{"nothing to amend", AmendOrderRequest{InstrumentID: mainPair.String(), OrderID: "1"}, errInvalidNewSizeOrPriceInformation},
		{"new size", AmendOrderRequest{InstrumentID: mainPair.String(), OrderID: "1", NewSize: 1}, nil},
		{"new price", AmendOrderRequest{InstrumentID: mainPair.String(), ClientOrderID: "b15", NewPrice: 1}, nil},
		{"new option price in USD", AmendOrderRequest{InstrumentID: optionsPair.String(), OrderID: "1", NewPriceUSD: 1}, nil},
		{"new option implied volatility", AmendOrderRequest{InstrumentID: optionsPair.String(), OrderID: "1", NewPriceVolatility: 0.5}, nil},
		{"new option price in USD and implied volatility", AmendOrderRequest{InstrumentID: optionsPair.String(), OrderID: "1", NewPriceUSD: 1, NewPriceVolatility: 0.5}, errConflictingAmendPrices},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, tc.arg.validate(), tc.err, "validate should return the expected error")
		})
	}
}

func TestAmendOrderRequestValidateAmendment(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  AmendOrderRequest
		err  error
	}{
		{"missing order IDs", AmendOrderRequest{NewSize: 1}, order.ErrOrderIDNotSet},
		{"nothing to amend", AmendOrderRequest{OrderID: "1"}, errInvalidNewSizeOrPriceInformation},
		{"new size without an instrument", AmendOrderRequest{OrderID: "1", NewSize: 1}, nil},
		{"new price", AmendOrderRequest{ClientOrderID: "b15", NewPrice: 1}, nil},
		{"new option price in USD", AmendOrderRequest{OrderID: "1", NewPriceUSD: 1}, nil},
		{"new option implied volatility", AmendOrderRequest{OrderID: "1", NewPriceVolatility: 0.5}, nil},
		{"new size and a new option price in USD", AmendOrderRequest{OrderID: "1", NewSize: 2, NewPriceUSD: 1}, nil},
		{"new price and a new option price in USD", AmendOrderRequest{OrderID: "1", NewPrice: 1, NewPriceUSD: 1}, errConflictingAmendPrices},
		{"new price and a new option implied volatility", AmendOrderRequest{OrderID: "1", NewPrice: 1, NewPriceVolatility: 0.5}, errConflictingAmendPrices},
		{"new option price in USD and implied volatility", AmendOrderRequest{OrderID: "1", NewPriceUSD: 1, NewPriceVolatility: 0.5}, errConflictingAmendPrices},
		{"every new price", AmendOrderRequest{OrderID: "1", NewPrice: 1, NewPriceUSD: 1, NewPriceVolatility: 0.5}, errConflictingAmendPrices},
		{"a negative new price beside a new option price in USD", AmendOrderRequest{OrderID: "1", NewSize: 2, NewPrice: -1, NewPriceUSD: 1}, errConflictingAmendPrices},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, tc.arg.validateAmendment(), tc.err, "validateAmendment should return the expected error")
		})
	}
}

func TestAmendOrder(t *testing.T) {
	t.Parallel()
	_, err := e.AmendOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &AmendOrderRequest{}
	_, err = e.AmendOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	_, err = e.AmendOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.OrderID = "1234"
	_, err = e.AmendOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *AmendOrderRequest
			body string
		}{
			{
				name: "required parameters only",
				arg:  &AmendOrderRequest{InstrumentID: mainPair.String(), OrderID: "2510789768709120", NewSize: 2},
				body: `{"instId":"BTC-USDT","ordId":"2510789768709120","newSz":"2"}`,
			},
			{
				name: "every optional parameter",
				arg: &AmendOrderRequest{
					InstrumentID:    optionsPair.String() + "-261127-90000-C",
					CancelOnFail:    true,
					OrderID:         "2510789768709120",
					ClientOrderID:   "b15",
					ClientRequestID: "b12344",
					NewSize:         2,
					NewPriceUSD:     3612.5,
					PriceAmendType:  "1",
					RPITakerAccess:  true,
					RPIPriceRound:   true,
					AttachAlgoOrders: []AmendAttachedAlgoOrderRequest{{
						AttachAlgoID:                  "1753184812254216194",
						AttachAlgoClientOrderID:       "attach1",
						NewTakeProfitTriggerPrice:     0.05,
						NewTakeProfitTriggerRatio:     0.3,
						NewTakeProfitOrderPrice:       -1,
						NewTakeProfitOrderKind:        "limit",
						NewStopLossTriggerPrice:       0.03,
						NewStopLossTriggerRatio:       0.1,
						NewStopLossOrderPrice:         -1,
						NewTakeProfitTriggerPriceType: "last",
						NewStopLossTriggerPriceType:   "index",
						NewSize:                       1,
						AmendPriceOnTriggerType:       "1",
						NewCallbackRatio:              0.05,
						NewCallbackSpread:             0.002,
						NewActivePrice:                0.045,
					}},
				},
				body: `{"instId":"BTC-USD-261127-90000-C","cxlOnFail":true,"ordId":"2510789768709120","clOrdId":"b15","reqId":"b12344","newSz":"2","newPxUsd":"3612.5","pxAmendType":"1","rpiTakerAccess":true,"rpiPxRound":true,"attachAlgoOrds":[{"attachAlgoId":"1753184812254216194","attachAlgoClOrdId":"attach1","newTpTriggerPx":"0.05","newTpTriggerRatio":"0.3","newTpOrdPx":"-1","newTpOrdKind":"limit","newSlTriggerPx":"0.03","newSlTriggerRatio":"0.1","newSlOrdPx":"-1","newTpTriggerPxType":"last","newSlTriggerPxType":"index","sz":"1","amendPxOnTriggerType":"1","newCallbackRatio":"0.05","newCallbackSpread":"0.002","newActivePx":"0.045"}]}`,
			},
			{
				name: "a new price",
				arg:  &AmendOrderRequest{InstrumentID: optionsPair.String() + "-261127-90000-C", OrderID: "2510789768709120", NewPrice: 0.0425},
				body: `{"instId":"BTC-USD-261127-90000-C","ordId":"2510789768709120","newPx":"0.0425"}`,
			},
			{
				name: "a new option implied volatility",
				arg:  &AmendOrderRequest{InstrumentID: optionsPair.String() + "-261127-90000-C", OrderID: "2510789768709120", NewPriceVolatility: 0.55},
				body: `{"instId":"BTC-USD-261127-90000-C","ordId":"2510789768709120","newPxVol":"0.55"}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/trade/amend-order", tc.body, orderActionReply)
				_, err := ex.AmendOrder(t.Context(), tc.arg)
				require.NoError(t, err, "AmendOrder must not error")
			})
		}
	})

	t.Run("refuses more than one new price", func(t *testing.T) {
		t.Parallel()
		var requests atomic.Int32
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests.Add(1)
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}))
		_, err := ex.AmendOrder(t.Context(), &AmendOrderRequest{InstrumentID: optionsPair.String() + "-261127-90000-C", OrderID: "1001", NewPriceUSD: 200, NewPriceVolatility: 0.7})
		require.ErrorIs(t, err, errConflictingAmendPrices, "AmendOrder must refuse an option repriced both in USD and as implied volatility")
		assert.Zero(t, requests.Load(), "no amendment should be sent")
	})

	for _, tc := range []struct {
		name string
		arg  *AmendOrderRequest
		exp  *OrderActionResponse
	}{
		{
			name: "new price",
			arg:  &AmendOrderRequest{InstrumentID: mainPair.String(), OrderID: "2510789768709120", NewPrice: 1233324.332},
			exp:  &OrderActionResponse{OrderID: "2510789768709120", Timestamp: types.Time(time.UnixMilli(1695190491421))},
		},
		{
			name: "client request ID",
			arg:  &AmendOrderRequest{InstrumentID: mainPair.String(), ClientOrderID: "oktswap6", ClientRequestID: "b12344", NewSize: 2},
			exp: &OrderActionResponse{
				OrderID: "2510789768709121", ClientOrderID: "oktswap6", RequestID: "b12344", Timestamp: types.Time(time.UnixMilli(1695190491421)),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !mockTests {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
			}
			result, err := e.AmendOrder(contextGenerate(), tc.arg)
			require.NoError(t, err, "AmendOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "AmendOrder should decode every field")
			} else {
				assert.NotEmpty(t, result.OrderID, "AmendOrder should return the order ID")
			}
		})
	}
}

func TestAmendMultipleOrders(t *testing.T) {
	t.Parallel()
	_, err := e.AmendMultipleOrders(contextGenerate(), []AmendOrderRequest{})
	require.ErrorIs(t, err, common.ErrEmptyParams)

	arg := AmendOrderRequest{
		NewPriceUSD: 1233,
	}
	_, err = e.AmendMultipleOrders(contextGenerate(), []AmendOrderRequest{arg})
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	_, err = e.AmendMultipleOrders(contextGenerate(), []AmendOrderRequest{arg})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	arg.ClientOrderID = "123212"
	arg.NewPriceUSD = 0
	_, err = e.AmendMultipleOrders(contextGenerate(), []AmendOrderRequest{arg})
	require.ErrorIs(t, err, errInvalidNewSizeOrPriceInformation)

	_, err = e.AmendMultipleOrders(contextGenerate(), make([]AmendOrderRequest, 21))
	require.ErrorIs(t, err, errExceedLimit)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/trade/amend-batch-orders",
			`[{"instId":"BTC-USDT","ordId":"2510789768709120","newSz":"2"},{"instId":"BTC-USD-261127-90000-C","clOrdId":"b15","reqId":"b12344","cxlOnFail":true,"newPxUsd":"3612.5"}]`,
			[]map[string]string{{"ordId": "2510789768709120", "sCode": "0"}, {"ordId": "2", "clOrdId": "b15", "reqId": "b12344", "sCode": "0"}})
		_, err := ex.AmendMultipleOrders(t.Context(), []AmendOrderRequest{
			{InstrumentID: mainPair.String(), OrderID: "2510789768709120", NewSize: 2},
			{InstrumentID: optionsPair.String() + "-261127-90000-C", ClientOrderID: "b15", ClientRequestID: "b12344", CancelOnFail: true, NewPriceUSD: 3612.5},
		})
		require.NoError(t, err, "AmendMultipleOrders must not error")
	})

	t.Run("refuses more than one new price", func(t *testing.T) {
		t.Parallel()
		var requests atomic.Int32
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests.Add(1)
			t.Errorf("unexpected request path %s", r.URL.Path)
			http.NotFound(w, r)
		}))
		_, err := ex.AmendMultipleOrders(t.Context(), []AmendOrderRequest{
			{InstrumentID: mainPair.String(), OrderID: "1", NewSize: 2},
			{InstrumentID: optionsPair.String() + "-261127-90000-C", OrderID: "2", NewPrice: 0.0425, NewPriceUSD: 3612.5},
		})
		require.ErrorIs(t, err, errConflictingAmendPrices, "AmendMultipleOrders must refuse an option repriced both by its price and in USD")
		assert.Zero(t, requests.Load(), "no amendment should be sent")
	})

	t.Run("charges each order to OKX's order budget", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			orders int
			exp    map[request.EndpointLimit]float64
		}{
			{"a lone order", 1, map[request.EndpointLimit]float64{amendOrderEPL: 1}},
			{"two orders", 2, map[request.EndpointLimit]float64{amendMultipleOrdersEPL: 2}},
			{"a full batch", maxOrdersPerBatch, map[request.EndpointLimit]float64{amendMultipleOrdersEPL: maxOrdersPerBatch}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newMockExchange(t, orderRowsHandler(t, "/trade/amend-batch-orders", "ordId", tc.orders))
				pools := chargeToPools(t, ex)
				args := make([]AmendOrderRequest, tc.orders)
				for i := range args {
					args[i] = AmendOrderRequest{InstrumentID: mainPair.String(), OrderID: strconv.Itoa(i + 1), NewSize: 2}
				}
				_, err := ex.AmendMultipleOrders(chargeContext(t), args)
				require.NoError(t, err, "AmendMultipleOrders must not error")
				assert.Equal(t, tc.exp, takenTokens(pools), "AmendMultipleOrders should take a token per order, a lone order's from the single order limit as OKX charges it")
			})
		}
	})

	t.Run("sends nothing its order budget cannot cover", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected request %s", r.URL.Path)
			http.NotFound(w, r)
		}))
		pools := chargeToPools(t, ex)
		require.True(t, pools[amendMultipleOrdersEPL].AllowN(time.Now(), orderBudgetPoolSize-1), "the pool must give out all but one of its tokens")
		_, err := ex.AmendMultipleOrders(chargeContext(t), []AmendOrderRequest{{InstrumentID: mainPair.String(), OrderID: "1", NewSize: 2}, {InstrumentID: mainPair.String(), OrderID: "2", NewSize: 2}})
		assert.ErrorIs(t, err, context.DeadlineExceeded, "AmendMultipleOrders should fail when its budget cannot cover both orders in time")
		assert.Equal(t, 1.0, pools[amendMultipleOrdersEPL].Tokens(), "AmendMultipleOrders should leave the budget it could not use")
	})

	t.Run("charges every attempt of a retried batch", func(t *testing.T) {
		t.Parallel()
		ex, pools, requests := newRetriedBatchExchange(t, "/trade/amend-batch-orders", "ordId")
		_, err := ex.AmendMultipleOrders(chargeContext(t), []AmendOrderRequest{{InstrumentID: mainPair.String(), OrderID: "1", NewSize: 2}, {InstrumentID: mainPair.String(), OrderID: "2", NewSize: 2}, {InstrumentID: mainPair.String(), OrderID: "3", NewSize: 2}})
		require.NoError(t, err, "AmendMultipleOrders must not error")
		require.Equal(t, int32(2), requests.Load(), "the batch must be sent again after the venue refused it")
		assert.Equal(t, map[request.EndpointLimit]float64{amendMultipleOrdersEPL: 6}, takenTokens(pools), "AmendMultipleOrders should take a token per order on every attempt")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AmendMultipleOrders(contextGenerate(), []AmendOrderRequest{{
		InstrumentID: mainPair.String(),
		OrderID:      "2510789768709120",
		NewPrice:     1233324.332,
	}})
	require.NoError(t, err, "AmendMultipleOrders must not error")
	if mockTests {
		exp := []OrderActionResponse{{OrderID: "2510789768709120", Timestamp: types.Time(time.UnixMilli(1695190491421))}}
		assert.Equal(t, exp, result, "AmendMultipleOrders should decode every field")
	} else {
		assert.Len(t, result, 1, "AmendMultipleOrders should return a result for the order")
	}
}

func TestClosePositions(t *testing.T) {
	t.Parallel()
	_, err := e.ClosePositions(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.ClosePositions(contextGenerate(), &ClosePositionsRequest{})
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = e.ClosePositions(contextGenerate(), &ClosePositionsRequest{MarginMode: "cross"})
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = e.ClosePositions(contextGenerate(), &ClosePositionsRequest{InstrumentID: mainPair.String(), MarginMode: "abc"})
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported)
	_, err = e.ClosePositions(contextGenerate(), &ClosePositionsRequest{InstrumentID: mainPair.String()})
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported)
	_, err = e.ClosePositions(contextGenerate(), &ClosePositionsRequest{InstrumentID: mainPair.String(), MarginMode: "cross", PositionSide: "invalid"})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	t.Run("position side is sent in lower case", func(t *testing.T) {
		t.Parallel()
		for in, want := range map[string]string{"LONG": positionSideLong, "Short": positionSideShort} {
			t.Run(in, func(t *testing.T) {
				t.Parallel()
				var mu sync.Mutex
				var body []byte
				ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/trade/close-position" {
						t.Errorf("unexpected request path %s", r.URL.Path)
						http.NotFound(w, r)
						return
					}
					b, err := io.ReadAll(r.Body)
					assert.NoError(t, err, "reading the close position body should not error")
					mu.Lock()
					body = b
					mu.Unlock()
					writeOKXData(t, w, []map[string]string{{"instId": perpetualSwapPair.String(), "posSide": want, "clOrdId": "", "tag": ""}})
				}))
				_, err := ex.ClosePositions(t.Context(), &ClosePositionsRequest{InstrumentID: perpetualSwapPair.String(), MarginMode: TradeModeCross, PositionSide: in})
				require.NoErrorf(t, err, "ClosePositions must accept position side %q", in)
				mu.Lock()
				defer mu.Unlock()
				var sent map[string]any
				require.NoError(t, json.Unmarshal(body, &sent), "the close position body must decode")
				assert.Equal(t, want, sent["posSide"], "the position side should be sent in lower case")
			})
		}
	})

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *ClosePositionsRequest
			body string
		}{
			{
				name: "required parameters only",
				arg:  &ClosePositionsRequest{InstrumentID: perpetualSwapPair.String(), MarginMode: TradeModeIsolated},
				body: `{"instId":"BTC-USDT-SWAP","mgnMode":"isolated"}`,
			},
			{
				name: "every optional parameter",
				arg: &ClosePositionsRequest{
					InstrumentID: mainPair.String(), PositionSide: positionSideNet, MarginMode: TradeModeCross, Currency: currency.USDT,
					AutomaticallyCanceled: true, ClientOrderID: "close1", Tag: "closetag",
				},
				body: `{"instId":"BTC-USDT","posSide":"net","mgnMode":"cross","ccy":"USDT","autoCxl":true,"clOrdId":"close1","tag":"closetag"}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/trade/close-position", tc.body, []map[string]string{{"instId": tc.arg.InstrumentID}})
				_, err := ex.ClosePositions(t.Context(), tc.arg)
				require.NoError(t, err, "ClosePositions must not error")
			})
		}
	})

	for _, tc := range []struct {
		name string
		arg  *ClosePositionsRequest
		exp  *ClosePositionResponse
	}{
		{
			name: "margin position",
			arg:  &ClosePositionsRequest{InstrumentID: mainPair.String(), MarginMode: TradeModeCross, Currency: currency.BTC},
			exp:  &ClosePositionResponse{InstrumentID: mainPair.String(), PositionSide: positionSideNet},
		},
		{
			name: "perpetual swap position with client order ID and tag",
			arg: &ClosePositionsRequest{
				InstrumentID: perpetualSwapPair.String(), PositionSide: positionSideLong, MarginMode: TradeModeCross,
				AutomaticallyCanceled: true, ClientOrderID: "close1", Tag: "closetag",
			},
			exp: &ClosePositionResponse{InstrumentID: perpetualSwapPair.String(), PositionSide: positionSideLong, ClientOrderID: "close1", Tag: "closetag"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !mockTests {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
			}
			result, err := e.ClosePositions(contextGenerate(), tc.arg)
			require.NoError(t, err, "ClosePositions must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "ClosePositions should decode every field")
			} else {
				assert.Equal(t, tc.arg.InstrumentID, result.InstrumentID, "ClosePositions should return the instrument ID")
			}
		})
	}
}

func TestGetOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderDetail(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetOrderDetail(contextGenerate(), &OrderDetailRequest{})
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = e.GetOrderDetail(contextGenerate(), &OrderDetailRequest{OrderID: "1234"})
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = e.GetOrderDetail(contextGenerate(), &OrderDetailRequest{InstrumentID: mainPair.String()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOrderDetail(contextGenerate(), &OrderDetailRequest{InstrumentID: "SUI-USDT", OrderID: "1974857619964870656"})
	require.NoError(t, err, "GetOrderDetail must not error")
	if mockTests {
		exp := &OrderDetailResponse{
			InstrumentType:          instTypeSpot,
			InstrumentID:            "SUI-USDT",
			TargetCurrency:          targetCurrencyQuote,
			OrderID:                 "1974857619964870656",
			ClientOrderID:           "b15",
			Tag:                     "spot1",
			Size:                    100,
			OrderType:               orderMarket,
			Side:                    order.Buy,
			PositionSide:            positionSideNet,
			TradeMode:               TradeModeCash,
			AccumulatedFillSize:     26.1,
			FillPrice:               3.83,
			TradeID:                 "744876980",
			FillSize:                26.1,
			FillTime:                types.Time(time.UnixMilli(1708587373361)),
			AveragePrice:            3.83,
			State:                   "filled",
			SelfTradePreventionMode: "cancel_maker",
			AttachedAlgoOrders:      []AttachedAlgoOrder{},
			FeeCurrency:             currency.NewCode("SUI"),
			TransactionFee:          -0.0261,
			RebateCurrency:          currency.USDT,
			Category:                "normal",
			UpdateTime:              types.Time(time.UnixMilli(1708587373362)),
			CreationTime:            types.Time(time.UnixMilli(1708587373361)),
			TradeQuoteCurrency:      currency.USDT,
		}
		assert.Equal(t, exp, result, "GetOrderDetail should decode every field")
	} else {
		assert.Equal(t, "1974857619964870656", result.OrderID, "GetOrderDetail should return the requested order")
	}
}

func TestGetOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderList(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOrderList(contextGenerate(), &OrderListRequest{Limit: 1})
	require.NoError(t, err, "GetOrderList must not error")
	if mockTests {
		exp := []OrderDetailResponse{{
			InstrumentType:          instTypeOption,
			InstrumentID:            "BTC-USD-261127-90000-C",
			OrderID:                 "1752588852617379840",
			ClientOrderID:           "opt1",
			Price:                   0.0425,
			PriceUSD:                3612.5,
			PriceVolatility:         0.55,
			PriceType:               "pxVol",
			Size:                    1,
			OrderType:               orderLimit,
			Side:                    order.Buy,
			PositionSide:            positionSideNet,
			TradeMode:               TradeModeCross,
			AccumulatedFillSize:     0.5,
			FillPrice:               0.0425,
			TradeID:                 "126",
			FillSize:                0.5,
			FillTime:                types.Time(time.UnixMilli(1724733618102)),
			AveragePrice:            0.0425,
			State:                   "partially_filled",
			SelfTradePreventionMode: "cancel_maker",
			AttachedAlgoOrders:      []AttachedAlgoOrder{},
			FeeCurrency:             currency.BTC,
			TransactionFee:          -0.00015,
			RebateCurrency:          currency.BTC,
			Category:                "normal",
			UpdateTime:              types.Time(time.UnixMilli(1724733618102)),
			CreationTime:            types.Time(time.UnixMilli(1724733617998)),
		}}
		assert.Equal(t, exp, result, "GetOrderList should decode every field")
	} else {
		assert.LessOrEqual(t, len(result), 1, "GetOrderList should honour the limit")
	}
}

func TestGet7DayOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.Get7DayOrderHistory(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.Get7DayOrderHistory(contextGenerate(), &OrderHistoryRequest{})
	require.ErrorIs(t, err, errInvalidInstrumentType)
	_, err = e.Get7DayOrderHistory(contextGenerate(), &OrderHistoryRequest{Category: "abc"})
	require.ErrorIs(t, err, errInvalidInstrumentType)
	_, err = e.Get7DayOrderHistory(contextGenerate(), &OrderHistoryRequest{InstrumentType: instTypeMargin, Begin: time.UnixMilli(1708587373362), End: time.UnixMilli(1708587373361)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	t.Run("request query", func(t *testing.T) {
		t.Parallel()
		ex := newQueryPinExchange(t, "/trade/orders-history", url.Values{"instType": {"SPOT"}, "ordType": {"limit"}}, []map[string]string{{"ordId": "680800019749904384"}})
		result, err := ex.Get7DayOrderHistory(t.Context(), &OrderHistoryRequest{InstrumentType: "spot", OrderType: "LIMIT"})
		require.NoError(t, err, "Get7DayOrderHistory must not error")
		assert.Equal(t, []OrderDetailResponse{{OrderID: "680800019749904384"}}, result, "Get7DayOrderHistory should decode the listed orders")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.Get7DayOrderHistory(contextGenerate(), &OrderHistoryRequest{InstrumentType: instTypeMargin})
	require.NoError(t, err, "Get7DayOrderHistory must not error")
	if mockTests {
		exp := []OrderDetailResponse{{
			InstrumentType:             instTypeMargin,
			InstrumentID:               mainPair.String(),
			Currency:                   currency.USDT,
			OrderID:                    "680800019749904384",
			ClientOrderID:              "margin7d",
			Price:                      51858,
			Size:                       0.00192834,
			ProfitAndLoss:              0.75,
			OrderType:                  orderLimit,
			Side:                       order.Sell,
			PositionSide:               positionSideNet,
			TradeMode:                  TradeModeCross,
			AccumulatedFillSize:        0.00192834,
			FillPrice:                  51858,
			TradeID:                    "744876980",
			FillSize:                   0.00192834,
			FillTime:                   types.Time(time.UnixMilli(1708587373361)),
			AveragePrice:               51858,
			State:                      "filled",
			SelfTradePreventionMode:    "cancel_maker",
			Leverage:                   3,
			AttachAlgoClientOrderID:    "attach7d",
			TakeProfitTriggerPrice:     60000,
			TakeProfitTriggerPriceType: "last",
			TakeProfitOrderPrice:       -1,
			StopLossTriggerPrice:       48000,
			StopLossTriggerPriceType:   "mark",
			StopLossOrderPrice:         -1,
			AttachedAlgoOrders: []AttachedAlgoOrder{
				{
					AttachAlgoID:               "1753184812254216194",
					AttachAlgoClientOrderID:    "attach7d",
					TakeProfitTriggerPrice:     60000,
					TakeProfitTriggerPriceType: "last",
					TakeProfitOrderPrice:       -1,
					TakeProfitOrderKind:        "condition",
					StopLossTriggerPrice:       48000,
					StopLossTriggerPriceType:   "mark",
					StopLossOrderPrice:         -1,
					Size:                       0.001,
					AmendPriceOnTriggerType:    "1",
					FailCode:                   "51020",
					FailReason:                 "Your order should meet or exceed the minimum order amount.",
				},
				{
					AttachAlgoID:            "1753184812254216195",
					AttachAlgoClientOrderID: "trail7d",
					TakeProfitTriggerRatio:  0.3,
					StopLossTriggerRatio:    0.1,
					AmendPriceOnTriggerType: "0",
					CallbackRatio:           0.05,
					CallbackSpread:          100,
					ActivePrice:             52000,
				},
			},
			LinkedAlgoOrder:    LinkedAlgoOrder{AlgoID: "1753184812254216196"},
			FeeCurrency:        currency.BTC,
			TransactionFee:     -0.00000192834,
			RebateCurrency:     currency.USDT,
			Source:             "13",
			Category:           "normal",
			ReduceOnly:         true,
			IsTakeProfitLimit:  true,
			AlgoClientOrderID:  "algo7d",
			AlgoID:             "1753184812254216193",
			UpdateTime:         types.Time(time.UnixMilli(1708587373362)),
			CreationTime:       types.Time(time.UnixMilli(1708587373361)),
			TradeQuoteCurrency: currency.USDT,
		}}
		assert.Equal(t, exp, result, "Get7DayOrderHistory should decode every field")
	} else {
		assert.NotNil(t, result, "Get7DayOrderHistory should return a list")
	}
}

func TestGet3MonthOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.Get3MonthOrderHistory(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.Get3MonthOrderHistory(contextGenerate(), &OrderHistoryRequest{})
	require.ErrorIs(t, err, errInvalidInstrumentType)
	_, err = e.Get3MonthOrderHistory(contextGenerate(), &OrderHistoryRequest{InstrumentType: instTypeMargin, Begin: time.UnixMilli(1708587373362), End: time.UnixMilli(1708587373361)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	t.Run("request query", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name  string
			arg   *OrderHistoryRequest
			query url.Values
		}{
			{name: "instrument type sent upper case", arg: &OrderHistoryRequest{InstrumentType: "margin"}, query: url.Values{"instType": {"MARGIN"}}},
			{
				name: "every parameter",
				arg: &OrderHistoryRequest{
					InstrumentType: instTypeSwap, InstrumentFamily: "BTC-USDT", InstrumentID: perpetualSwapPair.String(), OrderType: "POST_ONLY", State: "canceled",
					Category: "TWAP", After: "680800019749904385", Before: "680800019749904380", Begin: time.UnixMilli(1708587373361), End: time.UnixMilli(1708587373362), Limit: 100,
				},
				query: url.Values{
					"instType": {"SWAP"}, "instFamily": {"BTC-USDT"}, "instId": {"BTC-USDT-SWAP"}, "ordType": {"post_only"}, "state": {"canceled"}, "category": {"twap"},
					"after": {"680800019749904385"}, "before": {"680800019749904380"}, "begin": {"1708587373361"}, "end": {"1708587373362"}, "limit": {"100"},
				},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newQueryPinExchange(t, "/trade/orders-history-archive", tc.query, []map[string]string{{"ordId": "680800019749904385"}})
				result, err := ex.Get3MonthOrderHistory(t.Context(), tc.arg)
				require.NoError(t, err, "Get3MonthOrderHistory must not error")
				assert.Equal(t, []OrderDetailResponse{{OrderID: "680800019749904385"}}, result, "Get3MonthOrderHistory should decode the listed orders")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.Get3MonthOrderHistory(contextGenerate(), &OrderHistoryRequest{InstrumentType: instTypeMargin})
	require.NoError(t, err, "Get3MonthOrderHistory must not error")
	if mockTests {
		exp := []OrderDetailResponse{{
			InstrumentType:          instTypeMargin,
			InstrumentID:            mainPair.String(),
			Currency:                currency.BTC,
			OrderID:                 "680800019749904385",
			Tag:                     "archive",
			Price:                   52000,
			Size:                    0.01,
			OrderType:               orderPostOnly,
			Side:                    order.Buy,
			PositionSide:            positionSideNet,
			TradeMode:               TradeModeIsolated,
			State:                   "canceled",
			SelfTradePreventionID:   "1",
			SelfTradePreventionMode: "cancel_both",
			Leverage:                5,
			AttachedAlgoOrders:      []AttachedAlgoOrder{},
			FeeCurrency:             currency.BTC,
			RebateCurrency:          currency.USDT,
			RebateAmount:            0.0001,
			Category:                "partial_liquidation",
			CancelSource:            "20",
			CancelSourceReason:      "Cancel all after triggered",
			QuickMarginType:         "manual",
			UpdateTime:              types.Time(time.UnixMilli(1708587373362)),
			CreationTime:            types.Time(time.UnixMilli(1708587373361)),
			TradeQuoteCurrency:      currency.USDT,
			Outcome:                 "yes",
		}}
		assert.Equal(t, exp, result, "Get3MonthOrderHistory should decode every field")
	} else {
		assert.NotNil(t, result, "Get3MonthOrderHistory should return a list")
	}
}

func TestGetTransactionDetailsLast3Days(t *testing.T) {
	t.Parallel()
	_, err := e.GetTransactionDetailsLast3Days(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetTransactionDetailsLast3Days(contextGenerate(), &TransactionDetailRequest{Begin: time.UnixMilli(1708587373362), End: time.UnixMilli(1708587373361)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	t.Run("forwards the order and transaction type filters", func(t *testing.T) {
		t.Parallel()
		var mutex sync.Mutex
		var query url.Values
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method, "the fills request should be a GET")
			assert.Equal(t, "/trade/fills", r.URL.Path, "GetTransactionDetailsLast3Days should call the 3-day fills endpoint")
			mutex.Lock()
			query = r.URL.Query()
			mutex.Unlock()
			writeOKXData(t, w, []any{})
		}))
		const orderID = "312269865356374016"
		_, err := ex.GetTransactionDetailsLast3Days(t.Context(), &TransactionDetailRequest{
			InstrumentType: instTypeSpot,
			OrderID:        orderID,
			SubType:        "2",
			Limit:          3,
		})
		require.NoError(t, err, "GetTransactionDetailsLast3Days must not error")
		mutex.Lock()
		defer mutex.Unlock()
		require.NotNil(t, query, "a request must have been sent")
		assert.Equal(t, orderID, query.Get("ordId"), "GetTransactionDetailsLast3Days should forward the order ID filter as ordId")
		assert.Equal(t, "2", query.Get("subType"), "GetTransactionDetailsLast3Days should forward the transaction type filter as subType")
		assert.Equal(t, instTypeSpot, query.Get("instType"), "GetTransactionDetailsLast3Days should forward the instrument type")
		assert.Equal(t, "3", query.Get("limit"), "GetTransactionDetailsLast3Days should forward the limit")
	})

	t.Run("sends every filter", func(t *testing.T) {
		t.Parallel()
		var mutex sync.Mutex
		var query url.Values
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mutex.Lock()
			query = r.URL.Query()
			mutex.Unlock()
			writeOKXData(t, w, []any{})
		}))
		_, err := ex.GetTransactionDetailsLast3Days(t.Context(), &TransactionDetailRequest{
			InstrumentType:   "swap",
			InstrumentFamily: "BTC-USDT",
			InstrumentID:     perpetualSwapPair.String(),
			OrderID:          "680800019749904384",
			SubType:          "1",
			After:            "680800019754098688",
			Before:           "680800019754098600",
			Begin:            time.UnixMilli(1708587373000),
			End:              time.UnixMilli(1708587374000),
			Limit:            100,
		})
		require.NoError(t, err, "GetTransactionDetailsLast3Days must not error")
		mutex.Lock()
		defer mutex.Unlock()
		exp := url.Values{
			"instType": {instTypeSwap}, "instFamily": {"BTC-USDT"}, "instId": {"BTC-USDT-SWAP"}, "ordId": {"680800019749904384"},
			"subType": {"1"}, "after": {"680800019754098688"}, "before": {"680800019754098600"}, "begin": {"1708587373000"},
			"end": {"1708587374000"}, "limit": {"100"},
		}
		assert.Equal(t, exp, query, "GetTransactionDetailsLast3Days should send every filter")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTransactionDetailsLast3Days(contextGenerate(), &TransactionDetailRequest{InstrumentType: instTypeMargin, Limit: 1})
	require.NoError(t, err, "GetTransactionDetailsLast3Days must not error")
	if mockTests {
		exp := []TransactionDetailResponse{{
			InstrumentType:     instTypeMargin,
			InstrumentID:       mainPair.String(),
			TradeID:            "744876980",
			OrderID:            "680800019749904384",
			BillID:             "680800019754098688",
			TransactionType:    "1",
			FillPrice:          51858,
			FillSize:           0.00192834,
			FillIndexPrice:     51861.2,
			Side:               order.Buy,
			PositionSide:       positionSideNet,
			ExecutionType:      "T",
			FeeCurrency:        currency.BTC,
			Fee:                -0.00000192834,
			Timestamp:          types.Time(time.UnixMilli(1708587373362)),
			FillTime:           types.Time(time.UnixMilli(1708587373361)),
			FeeRate:            -0.001,
			TradeQuoteCurrency: currency.USDT,
		}}
		assert.Equal(t, exp, result, "GetTransactionDetailsLast3Days should decode every field")
	} else {
		assert.LessOrEqual(t, len(result), 1, "GetTransactionDetailsLast3Days should honour the limit")
	}
}

func TestGetTransactionDetailsLast3Months(t *testing.T) {
	t.Parallel()
	_, err := e.GetTransactionDetailsLast3Months(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetTransactionDetailsLast3Months(contextGenerate(), &TransactionDetailRequest{})
	require.ErrorIs(t, err, errInvalidInstrumentType)
	_, err = e.GetTransactionDetailsLast3Months(contextGenerate(), &TransactionDetailRequest{Limit: 10})
	require.ErrorIs(t, err, errInvalidInstrumentType)
	_, err = e.GetTransactionDetailsLast3Months(contextGenerate(), &TransactionDetailRequest{InstrumentType: instTypeMargin, Begin: time.UnixMilli(1708587373362), End: time.UnixMilli(1708587373361)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	for _, tc := range []struct {
		name string
		arg  *TransactionDetailRequest
		exp  []TransactionDetailResponse
	}{
		{
			name: "margin fills",
			arg:  &TransactionDetailRequest{InstrumentType: instTypeMargin},
			exp: []TransactionDetailResponse{{
				InstrumentType:     instTypeMargin,
				InstrumentID:       mainPair.String(),
				TradeID:            "744876981",
				OrderID:            "680800019749904384",
				BillID:             "680800019754098689",
				TransactionType:    "1",
				FillPrice:          51858,
				FillSize:           0.00192834,
				FillIndexPrice:     51861.2,
				Side:               order.Buy,
				PositionSide:       positionSideNet,
				ExecutionType:      "T",
				FeeCurrency:        currency.BTC,
				Fee:                -0.00000192834,
				Timestamp:          types.Time(time.UnixMilli(1708587373362)),
				FillTime:           types.Time(time.UnixMilli(1708587373361)),
				FeeRate:            -0.001,
				TradeQuoteCurrency: currency.USDT,
			}},
		},
		{
			name: "option fills of an order",
			arg: &TransactionDetailRequest{
				InstrumentType: instTypeOption, InstrumentFamily: "BTC-USD", OrderID: "1752588852617379840", SubType: "4", Limit: 1,
			},
			exp: []TransactionDetailResponse{{
				InstrumentType:      instTypeOption,
				InstrumentID:        "BTC-USD-261127-90000-C",
				TradeID:             "126",
				OrderID:             "1752588852617379840",
				ClientOrderID:       "opt1",
				BillID:              "1752588852617379841",
				TransactionType:     "4",
				Tag:                 "options",
				FillPrice:           0.0425,
				FillSize:            1,
				FillIndexPrice:      90050.1,
				FillProfitAndLoss:   0.0012,
				FillPriceVolatility: 0.55,
				FillPriceUSD:        3612.5,
				FillMarkVolatility:  0.56,
				FillForwardPrice:    90100.5,
				FillMarkPrice:       0.043,
				Side:                order.Sell,
				PositionSide:        positionSideShort,
				ExecutionType:       "M",
				FeeCurrency:         currency.BTC,
				Fee:                 -0.0003,
				Timestamp:           types.Time(time.UnixMilli(1724733618103)),
				FillTime:            types.Time(time.UnixMilli(1724733618102)),
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !mockTests {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
			}
			result, err := e.GetTransactionDetailsLast3Months(contextGenerate(), tc.arg)
			require.NoError(t, err, "GetTransactionDetailsLast3Months must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetTransactionDetailsLast3Months should decode every field")
			} else {
				assert.NotNil(t, result, "GetTransactionDetailsLast3Months should return a list")
			}
		})
	}
}

func TestGetTransactionDetails(t *testing.T) {
	t.Parallel()
	_, err := e.getTransactionDetails(t.Context(), &TransactionDetailRequest{Begin: time.UnixMilli(1708587373362), End: time.UnixMilli(1708587373361)}, "trade/fills-history", getTransactionDetail3MonthsEPL)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "getTransactionDetails must reject a window that begins after it ends")

	begin, end := time.UnixMilli(1708587373361), time.UnixMilli(1708587373362)
	reply := []map[string]string{{"instType": "SPOT", "instId": "BTC-USDT", "tradeId": "744876980", "ordId": "680800019749904384", "fillPx": "51858", "fillSz": "0.00192834", "side": "sell", "ts": "1708587373362"}}
	exp := []TransactionDetailResponse{{
		InstrumentType: instTypeSpot, InstrumentID: mainPair.String(), TradeID: "744876980", OrderID: "680800019749904384", FillPrice: 51858, FillSize: 0.00192834,
		Side: order.Sell, Timestamp: types.Time(time.UnixMilli(1708587373362)),
	}}
	for _, tc := range []struct {
		name  string
		arg   *TransactionDetailRequest
		query url.Values
	}{
		{name: "no filter", arg: &TransactionDetailRequest{}, query: url.Values{}},
		{name: "instrument type sent upper case", arg: &TransactionDetailRequest{InstrumentType: "spot"}, query: url.Values{"instType": {"SPOT"}}},
		{name: "instrument family", arg: &TransactionDetailRequest{InstrumentFamily: "BTC-USD"}, query: url.Values{"instFamily": {"BTC-USD"}}},
		{name: "instrument ID", arg: &TransactionDetailRequest{InstrumentID: mainPair.String()}, query: url.Values{"instId": {"BTC-USDT"}}},
		{name: "order ID", arg: &TransactionDetailRequest{OrderID: "680800019749904384"}, query: url.Values{"ordId": {"680800019749904384"}}},
		{name: "transaction type", arg: &TransactionDetailRequest{SubType: "2"}, query: url.Values{"subType": {"2"}}},
		{name: "after", arg: &TransactionDetailRequest{After: "744876980"}, query: url.Values{"after": {"744876980"}}},
		{name: "before", arg: &TransactionDetailRequest{Before: "744876979"}, query: url.Values{"before": {"744876979"}}},
		{name: "begin", arg: &TransactionDetailRequest{Begin: begin}, query: url.Values{"begin": {"1708587373361"}}},
		{name: "end", arg: &TransactionDetailRequest{End: end}, query: url.Values{"end": {"1708587373362"}}},
		{name: "equal begin and end", arg: &TransactionDetailRequest{Begin: begin, End: begin}, query: url.Values{"begin": {"1708587373361"}, "end": {"1708587373361"}}},
		{name: "limit", arg: &TransactionDetailRequest{Limit: 100}, query: url.Values{"limit": {"100"}}},
		{
			name: "every filter",
			arg: &TransactionDetailRequest{
				InstrumentType: instTypeSpot, InstrumentFamily: "BTC-USD", InstrumentID: mainPair.String(), OrderID: "680800019749904384", SubType: "2",
				After: "744876980", Before: "744876979", Begin: begin, End: end, Limit: 100,
			},
			query: url.Values{
				"instType": {"SPOT"}, "instFamily": {"BTC-USD"}, "instId": {"BTC-USDT"}, "ordId": {"680800019749904384"}, "subType": {"2"},
				"after": {"744876980"}, "before": {"744876979"}, "begin": {"1708587373361"}, "end": {"1708587373362"}, "limit": {"100"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newQueryPinExchange(t, "/trade/fills-history", tc.query, reply)
			result, err := ex.getTransactionDetails(t.Context(), tc.arg, "trade/fills-history", getTransactionDetail3MonthsEPL)
			require.NoError(t, err, "getTransactionDetails must not error")
			assert.Equal(t, exp, result, "getTransactionDetails should decode the fills from the caller's route")
		})
	}
}

func TestGetEasyConvertCurrencyList(t *testing.T) {
	t.Parallel()
	_, err := e.GetEasyConvertCurrencyList(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetEasyConvertCurrencyList(contextGenerate(), &EasyConvertCurrencyListRequest{Source: "1"})
	require.NoError(t, err, "GetEasyConvertCurrencyList must not error")
	if mockTests {
		exp := &EasyConvertCurrencyListResponse{
			FromData: []EasyConvertFromCurrency{
				{FromCurrency: currency.ADA, FromAmount: 6.580712708344864},
				{FromCurrency: currency.USDC, FromAmount: 2.9970000013055097},
			},
			ToCurrencies: []currency.Code{currency.USDT, currency.BTC, currency.ETH, currency.OKB},
		}
		assert.Equal(t, exp, result, "GetEasyConvertCurrencyList should decode every field")
	} else {
		assert.NotNil(t, result, "GetEasyConvertCurrencyList should return the currency lists")
	}
}

func TestPlaceEasyConvert(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceEasyConvert(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.PlaceEasyConvert(contextGenerate(), &PlaceEasyConvertRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.PlaceEasyConvert(contextGenerate(), &PlaceEasyConvertRequest{FromCurrencies: []currency.Code{currency.BTC}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.PlaceEasyConvert(contextGenerate(), &PlaceEasyConvertRequest{FromCurrencies: []currency.Code{currency.EMPTYCODE}, ToCurrency: currency.USDT})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.PlaceEasyConvert(contextGenerate(), &PlaceEasyConvertRequest{
		FromCurrencies: []currency.Code{currency.ADA, currency.USDC, currency.DOGE, currency.LTC, currency.XRP, currency.SOL},
		ToCurrency:     currency.USDT,
	})
	require.ErrorIs(t, err, errExceedLimit)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *PlaceEasyConvertRequest
			body string
		}{
			{"required parameters only", &PlaceEasyConvertRequest{FromCurrencies: []currency.Code{currency.ADA, currency.USDC}, ToCurrency: currency.OKB}, `{"fromCcy":["ADA","USDC"],"toCcy":"OKB"}`},
			{"every optional parameter", &PlaceEasyConvertRequest{FromCurrencies: []currency.Code{currency.ADA}, ToCurrency: currency.OKB, Source: "2"}, `{"fromCcy":["ADA"],"toCcy":"OKB","source":"2"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/trade/easy-convert", tc.body, []map[string]string{{"status": "running"}})
				_, err := ex.PlaceEasyConvert(t.Context(), tc.arg)
				require.NoError(t, err, "PlaceEasyConvert must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PlaceEasyConvert(contextGenerate(), &PlaceEasyConvertRequest{FromCurrencies: []currency.Code{currency.BTC}, ToCurrency: currency.USDT})
	require.NoError(t, err, "PlaceEasyConvert must not error")
	if mockTests {
		exp := []EasyConvertResponse{{
			Status: "running", FromCurrency: currency.BTC, ToCurrency: currency.USDT, FillFromSize: 0.0001, FillToSize: 8.3,
			UpdateTime: types.Time(time.UnixMilli(1661419684687)),
		}}
		assert.Equal(t, exp, result, "PlaceEasyConvert should decode every field")
	} else {
		assert.NotEmpty(t, result, "PlaceEasyConvert should return the conversion")
	}
}

func TestGetEasyConvertHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetEasyConvertHistory(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetEasyConvertHistory(contextGenerate(), &EasyConvertHistoryRequest{After: time.UnixMilli(1661313307979), Before: time.UnixMilli(1661313307980)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetEasyConvertHistory(contextGenerate(), &EasyConvertHistoryRequest{Limit: 1})
	require.NoError(t, err, "GetEasyConvertHistory must not error")
	if mockTests {
		exp := []EasyConvertResponse{{
			Status: "filled", FromCurrency: currency.OKB, ToCurrency: currency.ADA, FillFromSize: 0.1761712511667539,
			FillToSize: 6.73422059, Account: "18", UpdateTime: types.Time(time.UnixMilli(1661313307979)),
		}}
		assert.Equal(t, exp, result, "GetEasyConvertHistory should decode every field")
	} else {
		assert.LessOrEqual(t, len(result), 1, "GetEasyConvertHistory should honour the limit")
	}
}

func TestTimePageParams(t *testing.T) {
	t.Parallel()
	_, err := timePageParams(time.UnixMilli(1661313307980), time.UnixMilli(1661313307979), 0)
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	for _, tc := range []struct {
		name          string
		before, after time.Time
		limit         uint64
		exp           url.Values
	}{
		{"no parameters", time.Time{}, time.Time{}, 0, url.Values{}},
		{"milliseconds", time.UnixMilli(1661313307979), time.UnixMilli(1661313307990), 5, url.Values{"before": {"1661313307979"}, "after": {"1661313307990"}, "limit": {"5"}}},
		{"equal bounds", time.UnixMilli(1661313307979), time.UnixMilli(1661313307979), 0, url.Values{"before": {"1661313307979"}, "after": {"1661313307979"}}},
		{"after only", time.Time{}, time.UnixMilli(1661313307990), 0, url.Values{"after": {"1661313307990"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := timePageParams(tc.before, tc.after, tc.limit)
			require.NoError(t, err, "timePageParams must not error")
			assert.Equal(t, tc.exp, params, "timePageParams should send the cursors in Unix milliseconds")
		})
	}
}

func TestGetOneClickRepayCurrencyList(t *testing.T) {
	t.Parallel()
	_, err := e.GetOneClickRepayCurrencyList(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOneClickRepayCurrencyList(contextGenerate(), &OneClickRepayCurrencyListRequest{DebtType: "cross"})
	require.NoError(t, err, "GetOneClickRepayCurrencyList must not error")
	if mockTests {
		exp := []OneClickRepayCurrencyListResponse{{
			Debts: []OneClickRepayCurrencyListDebt{
				{Currency: currency.LTC, Amount: 29.653478},
				{Currency: currency.USDT, Amount: 237803.6828295906051002},
			},
			DebtType: "cross",
			Repays:   []OneClickRepayCurrencyListRepay{{Currency: currency.ETH, Amount: 0.4978335419825104}},
		}}
		assert.Equal(t, exp, result, "GetOneClickRepayCurrencyList should decode every field")
	} else {
		assert.NotNil(t, result, "GetOneClickRepayCurrencyList should return a list")
	}
}

func TestTradeOneClickRepay(t *testing.T) {
	t.Parallel()
	_, err := e.TradeOneClickRepay(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.TradeOneClickRepay(contextGenerate(), &TradeOneClickRepayRequest{DebtCurrencies: []currency.Code{}, RepayCurrency: currency.USDT})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.TradeOneClickRepay(contextGenerate(), &TradeOneClickRepayRequest{DebtCurrencies: []currency.Code{currency.BTC}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.TradeOneClickRepay(contextGenerate(), &TradeOneClickRepayRequest{DebtCurrencies: []currency.Code{currency.EMPTYCODE}, RepayCurrency: currency.USDT})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.TradeOneClickRepay(contextGenerate(), &TradeOneClickRepayRequest{
		DebtCurrencies: []currency.Code{currency.ETH, currency.BTC, currency.LTC, currency.DOGE, currency.XRP, currency.SOL},
		RepayCurrency:  currency.USDT,
	})
	require.ErrorIs(t, err, errExceedLimit)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/trade/one-click-repay", `{"debtCcy":["ETH","BTC"],"repayCcy":"USDT"}`, []map[string]string{{"status": "filled"}})
		_, err := ex.TradeOneClickRepay(t.Context(), &TradeOneClickRepayRequest{DebtCurrencies: []currency.Code{currency.ETH, currency.BTC}, RepayCurrency: currency.USDT})
		require.NoError(t, err, "TradeOneClickRepay must not error")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.TradeOneClickRepay(contextGenerate(), &TradeOneClickRepayRequest{
		DebtCurrencies: []currency.Code{currency.BTC},
		RepayCurrency:  currency.USDT,
	})
	require.NoError(t, err, "TradeOneClickRepay must not error")
	if mockTests {
		exp := []OneClickRepayResponse{{
			Status: "filled", DebtCurrency: currency.BTC, RepayCurrency: currency.USDT, FillDebtSize: 0.01023052, FillRepaySize: 30,
			UpdateTime: types.Time(time.UnixMilli(1646188520338)),
		}}
		assert.Equal(t, exp, result, "TradeOneClickRepay should decode every field")
	} else {
		assert.NotEmpty(t, result, "TradeOneClickRepay should return the repayment")
	}
}

func TestGetOneClickRepayHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOneClickRepayHistory(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetOneClickRepayHistory(contextGenerate(), &OneClickRepayHistoryRequest{After: time.UnixMilli(1661256148746), Before: time.UnixMilli(1661256148747)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOneClickRepayHistory(contextGenerate(), &OneClickRepayHistoryRequest{Limit: 1})
	require.NoError(t, err, "GetOneClickRepayHistory must not error")
	if mockTests {
		exp := []OneClickRepayResponse{{
			Status: "filled", DebtCurrency: currency.USDC, RepayCurrency: currency.ETH, FillDebtSize: 6950.48654479,
			FillRepaySize: 4.306797599509493, UpdateTime: types.Time(time.UnixMilli(1661256148746)),
		}}
		assert.Equal(t, exp, result, "GetOneClickRepayHistory should decode every field")
	} else {
		assert.LessOrEqual(t, len(result), 1, "GetOneClickRepayHistory should honour the limit")
	}
}

func TestGetOneClickRepayCurrencyListV2(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOneClickRepayCurrencyListV2(contextGenerate())
	require.NoError(t, err, "GetOneClickRepayCurrencyListV2 must not error")
	if mockTests {
		exp := &OneClickRepayCurrencyListV2Response{
			Debts: []OneClickRepayCurrencyListDebt{{Currency: currency.USDC, Amount: 100}},
			Repays: []OneClickRepayCurrencyListRepay{
				{Currency: currency.BTC, Amount: 1.000022977},
				{Currency: currency.USDT, Amount: 4998.0002397},
			},
		}
		assert.Equal(t, exp, result, "GetOneClickRepayCurrencyListV2 should decode every field")
	} else {
		assert.NotNil(t, result, "GetOneClickRepayCurrencyListV2 should return the currency lists")
	}
}

func TestTradeOneClickRepayV2(t *testing.T) {
	t.Parallel()
	_, err := e.TradeOneClickRepayV2(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.TradeOneClickRepayV2(contextGenerate(), &TradeOneClickRepayV2Request{RepayCurrencies: []currency.Code{currency.USDC}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.TradeOneClickRepayV2(contextGenerate(), &TradeOneClickRepayV2Request{DebtCurrency: currency.USDC})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.TradeOneClickRepayV2(contextGenerate(), &TradeOneClickRepayV2Request{DebtCurrency: currency.USDC, RepayCurrencies: []currency.Code{currency.EMPTYCODE}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		ex := newRequestPinExchange(t, "/trade/one-click-repay-v2", `{"debtCcy":"USDC","repayCcyList":["USDC","BTC"]}`, []map[string]any{{"debtCcy": "USDC"}})
		_, err := ex.TradeOneClickRepayV2(t.Context(), &TradeOneClickRepayV2Request{DebtCurrency: currency.USDC, RepayCurrencies: []currency.Code{currency.USDC, currency.BTC}})
		require.NoError(t, err, "TradeOneClickRepayV2 must not error")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.TradeOneClickRepayV2(contextGenerate(), &TradeOneClickRepayV2Request{DebtCurrency: currency.USDC, RepayCurrencies: []currency.Code{currency.USDC, currency.BTC}})
	require.NoError(t, err, "TradeOneClickRepayV2 must not error")
	if mockTests {
		exp := &TradeOneClickRepayV2Response{
			DebtCurrency:    currency.USDC,
			RepayCurrencies: []currency.Code{currency.USDC, currency.BTC},
			Timestamp:       types.Time(time.UnixMilli(1742192217514)),
		}
		assert.Equal(t, exp, result, "TradeOneClickRepayV2 should decode every field")
	} else {
		assert.Equal(t, currency.USDC, result.DebtCurrency, "TradeOneClickRepayV2 should return the debt currency")
	}
}

func TestGetOneClickRepayHistoryV2(t *testing.T) {
	t.Parallel()
	_, err := e.GetOneClickRepayHistoryV2(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.GetOneClickRepayHistoryV2(contextGenerate(), &OneClickRepayHistoryRequest{After: time.UnixMilli(1742194481852), Before: time.UnixMilli(1742194481853)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd)

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOneClickRepayHistoryV2(contextGenerate(), &OneClickRepayHistoryRequest{})
	require.NoError(t, err, "GetOneClickRepayHistoryV2 must not error")
	if mockTests {
		exp := []OneClickRepayHistoryV2Response{
			{
				DebtCurrency:    currency.USDC,
				RepayCurrencies: []currency.Code{currency.USDC, currency.BTC},
				FillDebtSize:    9.079631989,
				Status:          "filled",
				Orders: []OneClickRepayOrder{
					{
						OrderID: "2338478342062235648", InstrumentID: "USDC-USDT", OrderType: orderIOC, Side: order.Buy, Price: 1.0049,
						Size: 9.0886514537313433, FillPrice: 1, FillSize: 9.088651, State: "filled", CreationTime: types.Time(time.UnixMilli(1742194485439)),
					},
					{
						OrderID: "2338478237607288832", InstrumentID: mainPair.String(), OrderType: orderIOC, Side: order.Sell, Price: 82856.7,
						Size: 0.000109696512171, FillPrice: 83271.9, FillSize: 0.00010969, State: "filled", CreationTime: types.Time(time.UnixMilli(1742194482326)),
					},
				},
				Timestamp: types.Time(time.UnixMilli(1742194481852)),
			},
			{
				DebtCurrency:    currency.USDC,
				RepayCurrencies: []currency.Code{currency.USDC, currency.BTC},
				FillDebtSize:    100,
				Status:          "filled",
				Orders:          []OneClickRepayOrder{},
				Timestamp:       types.Time(time.UnixMilli(1742192217511)),
			},
		}
		assert.Equal(t, exp, result, "GetOneClickRepayHistoryV2 should decode every field")
	} else {
		assert.NotNil(t, result, "GetOneClickRepayHistoryV2 should return a list")
	}
}

func TestCancelAllMMPOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllMMPOrders(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.CancelAllMMPOrders(contextGenerate(), &MassCancelOrderRequest{InstrumentFamily: "BTC-USD", LockInterval: 2000})
	require.ErrorIs(t, err, errInvalidInstrumentType)
	_, err = e.CancelAllMMPOrders(contextGenerate(), &MassCancelOrderRequest{InstrumentType: instTypeOption, LockInterval: 2000})
	require.ErrorIs(t, err, errInstrumentFamilyRequired)
	_, err = e.CancelAllMMPOrders(contextGenerate(), &MassCancelOrderRequest{InstrumentType: instTypeOption, InstrumentFamily: "BTC-USD", LockInterval: 10001})
	require.ErrorIs(t, err, errInvalidLockInterval)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *MassCancelOrderRequest
			body string
		}{
			{"required parameters only", &MassCancelOrderRequest{InstrumentType: instTypeOption, InstrumentFamily: "BTC-USD"}, `{"instType":"OPTION","instFamily":"BTC-USD"}`},
			{"every optional parameter", &MassCancelOrderRequest{InstrumentType: instTypeOption, InstrumentFamily: "BTC-USD", LockInterval: 10000}, `{"instType":"OPTION","instFamily":"BTC-USD","lockInterval":"10000"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/trade/mass-cancel", tc.body, []map[string]bool{{"result": true}})
				_, err := ex.CancelAllMMPOrders(t.Context(), tc.arg)
				require.NoError(t, err, "CancelAllMMPOrders must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelAllMMPOrders(contextGenerate(), &MassCancelOrderRequest{InstrumentType: instTypeOption, InstrumentFamily: "BTC-USD", LockInterval: 2000})
	require.NoError(t, err, "CancelAllMMPOrders must not error")
	if mockTests {
		assert.Equal(t, &MassCancelOrderResponse{Result: true}, result, "CancelAllMMPOrders should decode every field")
	} else {
		assert.NotNil(t, result, "CancelAllMMPOrders should return the result")
	}
}

func TestCancelAllDelayed(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllDelayed(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)
	_, err = e.CancelAllDelayed(contextGenerate(), &CancelAllAfterRequest{TimeOut: 2})
	require.ErrorIs(t, err, errCountdownTimeoutRequired)
	_, err = e.CancelAllDelayed(contextGenerate(), &CancelAllAfterRequest{TimeOut: 121})
	require.ErrorIs(t, err, errCountdownTimeoutRequired)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *CancelAllAfterRequest
			body string
		}{
			{"disabling", &CancelAllAfterRequest{}, `{"timeOut":"0"}`},
			{"every optional parameter", &CancelAllAfterRequest{TimeOut: 10, Tag: "caa1"}, `{"timeOut":"10","tag":"caa1"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/trade/cancel-all-after", tc.body, []map[string]string{{"triggerTime": "0"}})
				_, err := ex.CancelAllDelayed(t.Context(), tc.arg)
				require.NoError(t, err, "CancelAllDelayed must not error")
			})
		}
	})

	for _, tc := range []struct {
		name string
		arg  *CancelAllAfterRequest
		exp  *CancelAllAfterResponse
	}{
		{
			name: "countdown",
			arg:  &CancelAllAfterRequest{TimeOut: 60},
			exp:  &CancelAllAfterResponse{TriggerTime: types.Time(time.Unix(1587971460, 0)), Timestamp: types.Time(time.Unix(1587971400, 0))},
		},
		{
			name: "tagged countdown",
			arg:  &CancelAllAfterRequest{TimeOut: 60, Tag: "caa1"},
			exp:  &CancelAllAfterResponse{TriggerTime: types.Time(time.Unix(1587971470, 0)), Tag: "caa1", Timestamp: types.Time(time.Unix(1587971410, 0))},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !mockTests {
				sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
			}
			result, err := e.CancelAllDelayed(contextGenerate(), tc.arg)
			require.NoError(t, err, "CancelAllDelayed must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "CancelAllDelayed should decode every field")
			} else {
				assert.NotZero(t, result.TriggerTime, "CancelAllDelayed should return the trigger time")
			}
		})
	}
}

func TestGetTradeAccountRateLimit(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTradeAccountRateLimit(contextGenerate())
	require.NoError(t, err, "GetTradeAccountRateLimit must not error")
	if mockTests {
		exp := &AccountRateLimitResponse{
			FillRatio:            0.1234,
			MainFillRatio:        0.1235,
			AccountRateLimit:     2000,
			NextAccountRateLimit: 2001,
			Timestamp:            types.Time(time.UnixMilli(1597026383085)),
		}
		assert.Equal(t, exp, result, "GetTradeAccountRateLimit should decode every field")
	} else {
		assert.Positive(t, result.AccountRateLimit.Float64(), "GetTradeAccountRateLimit should return the rate limit")
	}
}

func TestPreCheckOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PreCheckOrder(contextGenerate(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer)

	arg := &OrderPreCheckRequest{
		ClientOrderID: "b15",
	}
	_, err = e.PreCheckOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errMissingInstrumentID)

	arg.InstrumentID = mainPair.String()
	_, err = e.PreCheckOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, errInvalidTradeModeValue)

	arg.TradeMode = "cash"
	_, err = e.PreCheckOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	arg.Side = "buy"
	_, err = e.PreCheckOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	arg.OrderType = "limit"
	_, err = e.PreCheckOrder(contextGenerate(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin)

	t.Run("request body", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			arg  *OrderPreCheckRequest
			body string
		}{
			{
				name: "required parameters only",
				arg:  &OrderPreCheckRequest{InstrumentID: mainPair.String(), TradeMode: TradeModeCash, Side: order.Buy.Lower(), OrderType: orderMarket, Size: 100},
				body: `{"instId":"BTC-USDT","tdMode":"cash","side":"buy","ordType":"market","sz":"100"}`,
			},
			{
				name: "every optional parameter",
				arg: &OrderPreCheckRequest{
					InstrumentID: perpetualSwapPair.String(), TradeMode: TradeModeCross, ClientOrderID: "b15", Side: order.Sell.Lower(),
					PositionSide: positionSideNet, OrderType: orderLimit, Size: 2, Price: 95000, Outcome: "no", ReduceOnly: true,
					TargetCurrency: targetCurrencyBase,
					AttachAlgoOrders: []AttachAlgoOrderRequest{{
						AttachAlgoClientOrderID: "attach1", TakeProfitTriggerPrice: 90000, TakeProfitOrderPrice: -1, TakeProfitOrderKind: "condition",
						StopLossTriggerPrice: 99000, StopLossOrderPrice: -1, TakeProfitTriggerPriceType: "mark", StopLossTriggerPriceType: "last", Size: 1,
					}},
				},
				body: `{"instId":"BTC-USDT-SWAP","tdMode":"cross","clOrdId":"b15","side":"sell","posSide":"net","ordType":"limit","sz":"2","px":"95000","outcome":"no","reduceOnly":true,"tgtCcy":"base_ccy","attachAlgoOrds":[{"attachAlgoClOrdId":"attach1","tpTriggerPx":"90000","tpOrdPx":"-1","tpOrdKind":"condition","slTriggerPx":"99000","slOrdPx":"-1","tpTriggerPxType":"mark","slTriggerPxType":"last","sz":"1"}]}`,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newRequestPinExchange(t, "/trade/order-precheck", tc.body, []map[string]string{{"adjEq": "1"}})
				_, err := ex.PreCheckOrder(t.Context(), tc.arg)
				require.NoError(t, err, "PreCheckOrder must not error")
			})
		}
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.PreCheckOrder(contextGenerate(), &OrderPreCheckRequest{
		InstrumentID:  mainPair.String(),
		TradeMode:     "cash",
		ClientOrderID: "b15",
		Side:          order.Buy.Lower(),
		OrderType:     "limit",
		Price:         2.15,
		Size:          2,
	})
	require.NoError(t, err, "PreCheckOrder must not error")
	if mockTests {
		exp := &OrderPreCheckResponse{
			AdjustedEquity:                     41.94347460746277,
			AdjustedEquityChange:               -226.05616481626,
			InitialMarginRequirement:           0.25,
			InitialMarginRequirementChange:     57.74709688430927,
			MaintenanceMarginRequirement:       0.12,
			MaintenanceMarginRequirementChange: 0.02,
			MarginRatio:                        1.5,
			MarginRatioChange:                  -0.5,
			AvailableBalance:                   0.5,
			AvailableBalanceChange:             -0.1,
			LiquidationPrice:                   6764.8556232031115,
			LiquidationPriceDifference:         -57693.044376796888536773622035980224609375,
			LiquidationPriceDifferenceRatio:    -0.8950500152315991,
			PositiveAssetBalance:               0.4,
			PositiveAssetBalanceChange:         0.1,
			Liabilities:                        0.3,
			LiabilitiesChange:                  0.2,
			LiabilitiesChangeCurrency:          currency.USDT,
			PositiveAssetUnitType:              "1",
		}
		assert.Equal(t, exp, result, "PreCheckOrder should decode every field")
	} else {
		assert.NotNil(t, result, "PreCheckOrder should return the account information")
	}
}

func TestPlaceOrderRequestValidate(t *testing.T) {
	t.Parallel()
	var p *PlaceOrderRequest
	require.ErrorIs(t, p.Validate(), common.ErrNilPointer)
	p = &PlaceOrderRequest{}
	require.ErrorIs(t, p.Validate(), errMissingInstrumentID)
	p.InstrumentID = mainPair.String()
	require.ErrorIs(t, p.Validate(), order.ErrSideIsInvalid)
	p.Side = order.Buy.String()
	require.ErrorIs(t, p.Validate(), errInvalidTradeModeValue)
	p.TradeMode = TradeModeIsolated
	p.AssetType = asset.Futures
	p.PositionSide = "reverse"
	require.ErrorIs(t, p.Validate(), order.ErrSideIsInvalid)
	p.PositionSide = "long"
	require.ErrorIs(t, p.Validate(), order.ErrTypeIsInvalid)
	p.OrderType = order.Market.String()
	require.ErrorIs(t, p.Validate(), limits.ErrAmountBelowMin)
	p.Size = 1
	p.TargetCurrency = "moo cows"
	require.ErrorIs(t, p.Validate(), errCurrencyQuantityTypeRequired)
	p.TargetCurrency = "base_ccy"
	require.NoError(t, p.Validate())

	// The account's position mode decides the position side of futures and
	// perpetual swap orders, so empty and net pass alongside long and short.
	for _, a := range []asset.Item{asset.Futures, asset.PerpetualSwap} {
		p.AssetType = a
		for _, posSide := range []string{positionSideLong, positionSideShort, positionSideNet, ""} {
			p.PositionSide = posSide
			require.NoErrorf(t, p.Validate(), "Validate must accept position side %q for %s", posSide, a)
		}
		p.PositionSide = "reverse"
		require.ErrorIsf(t, p.Validate(), order.ErrSideIsInvalid, "Validate must reject an unknown position side for %s", a)
	}

	t.Run("requires a side for contracts", func(t *testing.T) {
		t.Parallel()
		for _, assetType := range []asset.Item{asset.Futures, asset.PerpetualSwap, asset.Options} {
			arg := &PlaceOrderRequest{
				AssetType:    assetType,
				InstrumentID: mainPair.String(),
				TradeMode:    TradeModeCross,
				OrderType:    orderLimit,
				Size:         1,
				PositionSide: "long",
			}
			err := arg.Validate()
			assert.ErrorIsf(t, err, order.ErrSideIsInvalid, "Validate should reject a %s order without side", assetType)
			arg.Side = "buy"
			require.NoErrorf(t, arg.Validate(), "Validate must accept a %s order with side", assetType)
		}
	})

	t.Run("accepts trade modes and order types beyond the known values", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct{ tradeMode, orderType string }{
			{"spot_isolated", orderLimit},
			{TradeModeCash, orderRPI},
			{TradeModeCash, orderOptionFOK},
		} {
			arg := &PlaceOrderRequest{InstrumentID: mainPair.String(), Side: "buy", TradeMode: tc.tradeMode, OrderType: tc.orderType, Size: 1}
			assert.NoErrorf(t, arg.Validate(), "Validate should accept trade mode %q with order type %q", tc.tradeMode, tc.orderType)
		}
	})
}

func TestPlaceOrderRequestValidateOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  PlaceOrderRequest
		err  error
	}{
		{"no side", PlaceOrderRequest{TradeMode: TradeModeCash, OrderType: orderLimit, Size: 1}, order.ErrSideIsInvalid},
		{"no trade mode", PlaceOrderRequest{Side: "buy", OrderType: orderLimit, Size: 1}, errInvalidTradeModeValue},
		{"no size", PlaceOrderRequest{Side: "buy", TradeMode: TradeModeCash, OrderType: orderLimit}, limits.ErrAmountBelowMin},
		{"complete without an instrument", PlaceOrderRequest{Side: "buy", TradeMode: TradeModeCash, OrderType: orderLimit, Size: 1}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, tc.arg.validateOrder(), tc.err, "validateOrder should return the expected error")
		})
	}
}

// TestDeprecatedUlyReplacedByInstFamily pins the uly to instFamily migration
// across the affected endpoints: OKX removed uly from their documented
// parameters in favour of instFamily, so every migrated case asserts
// instFamily carries the filter and uly stays absent. Where the filter is an
// underlying rather than a family — an underlying such as BTC-USD spans
// several families (BTC-USD and BTC-USD_UM) — OKX still honours the
// undocumented uly, so that case asserts the filter keeps travelling as uly
// and instFamily stays absent.
func TestDeprecatedUlyReplacedByInstFamily(t *testing.T) {
	// Serial: every case reads the last request the shared mock server recorded
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	var mu sync.Mutex
	var gotPath string
	var gotQuery url.Values
	var gotBody []byte

	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		gotBody, _ = io.ReadAll(r.Body)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		data := `{"code":"0","msg":"","data":[]}`
		if r.URL.Path == "/public/insurance-fund" || r.URL.Path == "/rfq/maker-instrument-settings" {
			// Single-pointer responses need a one-item array
			data = `{"code":"0","msg":"","data":[{}]}`
		}
		_, _ = w.Write([]byte(data))
	}))

	b := e.GetBase()
	b.SkipAuthCheck = true
	require.NoError(t, e.SetHTTPClient(srv.Client()), "SetHTTPClient must not error")
	for k := range b.API.Endpoints.GetURLMap() {
		require.NoErrorf(t, b.API.Endpoints.SetRunningURL(k, srv.URL+"/"), "Setup must point endpoint %s at the mock server", k)
	}

	for _, tc := range []struct {
		name  string
		call  func() error
		path  string
		param string
		value string
		body  bool
	}{
		{
			name: "tickers",
			call: func() error {
				_, err := e.GetTickers(t.Context(), &TickersRequest{InstrumentType: instTypeSwap, InstrumentFamily: "BTC-USDT"})
				return err
			},
			path:  "/market/tickers",
			param: "instFamily",
			value: "BTC-USDT",
		},
		{
			name: "block tickers",
			call: func() error {
				_, err := e.GetBlockTickers(t.Context(), &GetBlockTickersRequest{InstrumentType: instTypeSwap, InstrumentFamily: "BTC-USDT"})
				return err
			},
			path:  "/market/block-tickers",
			param: "instFamily",
			value: "BTC-USDT",
		},
		{
			name: "mark price",
			call: func() error {
				_, err := e.GetMarkPrice(t.Context(), &MarkPriceRequest{InstrumentType: instTypeSwap, InstrumentFamily: "BTC-USDT"})
				return err
			},
			path:  "/public/mark-price",
			param: "instFamily",
			value: "BTC-USDT",
		},
		{
			// An underlying such as BTC-USD spans several families (BTC-USD
			// and BTC-USD_UM), so querying by underlying still needs the
			// undocumented uly, which OKX continues to honour; pinned here so
			// the migration does not sweep it up.
			name: "public instruments keeps uly",
			call: func() error {
				_, err := e.GetInstruments(t.Context(), &InstrumentsRequest{
					InstrumentType: instTypeSwap,
					Underlying:     "BTC-USDT",
				})
				return err
			},
			path:  "/public/instruments",
			param: "uly",
			value: "BTC-USDT",
		},
		{
			name: "pending order list",
			call: func() error {
				_, err := e.GetOrderList(t.Context(), &OrderListRequest{
					InstrumentType:   instTypeSwap,
					InstrumentFamily: "BTC-USDT",
				})
				return err
			},
			path:  "/trade/orders-pending",
			param: "instFamily",
			value: "BTC-USDT",
		},
		{
			name: "7 day order history",
			call: func() error {
				_, err := e.Get7DayOrderHistory(t.Context(), &OrderHistoryRequest{
					InstrumentType:   instTypeSwap,
					InstrumentFamily: "BTC-USDT",
				})
				return err
			},
			path:  "/trade/orders-history",
			param: "instFamily",
			value: "BTC-USDT",
		},
		{
			name: "3 month order history",
			call: func() error {
				_, err := e.Get3MonthOrderHistory(t.Context(), &OrderHistoryRequest{
					InstrumentType:   instTypeSwap,
					InstrumentFamily: "BTC-USDT",
				})
				return err
			},
			path:  "/trade/orders-history-archive",
			param: "instFamily",
			value: "BTC-USDT",
		},
		{
			name: "transaction details last 3 days",
			call: func() error {
				_, err := e.GetTransactionDetailsLast3Days(t.Context(), &TransactionDetailRequest{
					InstrumentType:   instTypeSwap,
					InstrumentFamily: "BTC-USDT",
				})
				return err
			},
			path:  "/trade/fills",
			param: "instFamily",
			value: "BTC-USDT",
		},
		{
			name: "transaction details last 3 months",
			call: func() error {
				_, err := e.GetTransactionDetailsLast3Months(t.Context(), &TransactionDetailRequest{
					InstrumentType:   instTypeSwap,
					InstrumentFamily: "BTC-USDT",
				})
				return err
			},
			path:  "/trade/fills-history",
			param: "instFamily",
			value: "BTC-USDT",
		},
		{
			name: "account instruments",
			call: func() error {
				_, err := e.GetAccountInstruments(t.Context(), &AccountInstrumentsRequest{InstrumentType: instTypeSwap, InstrumentFamily: "BTC-USDT"})
				return err
			},
			path:  "/account/instruments",
			param: "instFamily",
			value: "BTC-USDT",
		},
		{
			name: "portfolio margin position tiers",
			call: func() error {
				_, err := e.GetPMPositionLimitation(t.Context(), &PMPositionLimitationRequest{InstrumentType: instTypeSwap, InstrumentFamilies: []string{"BTC-USDT"}})
				return err
			},
			path:  "/account/position-tiers",
			param: "instFamily",
			value: "BTC-USDT",
		},
		{
			name: "trade fee",
			call: func() error {
				_, err := e.GetTradeFee(t.Context(), &TradeFeeRequest{InstrumentType: instTypeSwap, InstrumentFamily: "BTC-USDT"})
				return err
			},
			path:  "/account/trade-fee",
			param: "instFamily",
			value: "BTC-USDT",
		},
		{
			name: "open interest swap family",
			call: func() error {
				_, err := e.GetOpenInterestData(t.Context(), &OpenInterestRequest{InstrumentType: instTypeSwap, InstrumentFamily: "BTC-USDT"})
				return err
			},
			path:  "/public/open-interest",
			param: "instFamily",
			value: "BTC-USDT",
		},
		{
			// uly takes an underlying, which spans several families (BTC-USD
			// spans BTC-USD and BTC-USD_UM), so an option family goes out as
			// instFamily, which OKX documents as required for OPTION.
			name: "open interest option family",
			call: func() error {
				_, err := e.GetOpenInterestData(t.Context(), &OpenInterestRequest{InstrumentType: instTypeOption, InstrumentFamily: "BTC-USD_UM"})
				return err
			},
			path:  "/public/open-interest",
			param: "instFamily",
			value: "BTC-USD_UM",
		},
		{
			name: "option market data",
			call: func() error {
				_, err := e.GetOptionMarketData(t.Context(), &OptionMarketDataRequest{InstrumentFamily: "BTC-USD"})
				return err
			},
			path:  "/public/opt-summary",
			param: "instFamily",
			value: "BTC-USD",
		},
		{
			name: "delivery history",
			call: func() error {
				_, err := e.GetDeliveryHistory(t.Context(), &DeliveryHistoryRequest{InstrumentType: instTypeFutures, InstrumentFamily: "BTC-USD"})
				return err
			},
			path:  "/public/delivery-exercise-history",
			param: "instFamily",
			value: "BTC-USD",
		},
		{
			name: "position tiers",
			call: func() error {
				_, err := e.GetPositionTiers(t.Context(), &PositionTiersRequest{InstrumentType: instTypeSwap, TradeMode: TradeModeCross, InstrumentFamilies: []string{"BTC-USDT"}, InstrumentIDs: []string{"BTC-USDT-SWAP"}})
				return err
			},
			path:  "/public/position-tiers",
			param: "instFamily",
			value: "BTC-USDT",
		},
		{
			name: "insurance fund",
			call: func() error {
				_, err := e.GetInsuranceFundInformation(t.Context(), &InsuranceFundRequest{
					InstrumentType:   instTypeSwap,
					InstrumentFamily: "BTC-USDT",
				})
				return err
			},
			path:  "/public/insurance-fund",
			param: "instFamily",
			value: "BTC-USDT",
		},
		{
			name: "set quote products",
			call: func() error {
				_, err := e.SetQuoteProducts(t.Context(), []SetQuoteProductRequest{{
					InstrumentType: instTypeSwap,
					Data:           []MakerInstrumentSetting{{InstrumentFamily: "BTC-USDT"}},
				}})
				return err
			},
			path:  "/rfq/maker-instrument-settings",
			param: "instFamily",
			value: "BTC-USDT",
			body:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, tc.call(), "request must not error")
			mu.Lock()
			path, query, body := gotPath, gotQuery, gotBody
			mu.Unlock()
			assert.Equal(t, tc.path, path, "the documented endpoint should be requested")
			if tc.body {
				assert.Contains(t, string(body), `"`+tc.param+`":"`+tc.value+`"`, "the filter should travel under the expected parameter in the posted body")
				assert.NotContains(t, string(body), `"uly"`, "only one of uly and instFamily should be sent")
				return
			}
			assert.Equal(t, tc.value, query.Get(tc.param), "the filter should travel under the expected parameter")
			other := "uly"
			if tc.param == "uly" {
				other = "instFamily"
			}
			assert.NotContains(t, query, other, "only one of uly and instFamily should be sent")
		})
	}
}
