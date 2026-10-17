package kraken

import (
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// wsTradingTestToken is the websocket token the trading tests hold
const wsTradingTestToken = "G38a1tGFzqGiUCmnegBcm8d4nfP3tytiNQz6tkCBYXY"

// newWsTradingTestExchange returns an exchange whose private connection is served by a mock server, which checks each
// request against exp, the expected request without its req_id, and answers with replies, setting their req_id to the
// request's
func newWsTradingTestExchange(t *testing.T, exp string, replies ...string) *Exchange {
	t.Helper()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	ex.wsToken = wsTradingTestToken
	mock := func(tb testing.TB, msg []byte, w *gws.Conn) error {
		tb.Helper()
		reqID, err := jsonparser.GetInt(msg, "req_id")
		if err != nil {
			return err
		}
		assert.JSONEq(tb, exp, string(jsonparser.Delete(slices.Clone(msg), "req_id")), "request should be the expected one")
		for _, r := range replies {
			reply, err := jsonparser.Set([]byte(r), []byte(strconv.FormatInt(reqID, 10)), "req_id")
			if err != nil {
				return err
			}
			if err := w.WriteMessage(gws.TextMessage, reply); err != nil {
				return err
			}
		}
		return nil
	}
	ex.Features.Subscriptions = subscription.List{}
	useTestWebsocket(t, ex, newMockWebsocketURL(t, mock), wsPrivateConnection)
	ex.Websocket.SetSubscriptionsNotRequired()
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	return ex
}

// wsTestTime parses an RFC3339 time
func wsTestTime(tb testing.TB, s string) time.Time {
	tb.Helper()
	v, err := time.Parse(time.RFC3339Nano, s)
	require.NoErrorf(tb, err, "Parse must not error for %s", s)
	return v
}

func TestWsAddOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WsAddOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsAddOrder must reject a nil request")
	valid := WsOrder{OrderType: "limit", Side: "buy", Quantity: 1, LimitPrice: 26500}
	for _, tc := range []struct {
		name string
		req  *WsAddOrderRequest
		err  error
	}{
		{"pair", &WsAddOrderRequest{Order: valid}, currency.ErrCurrencyPairEmpty},
		{"pair without a quote", &WsAddOrderRequest{Pair: partialTestPairs[0], Order: valid}, currency.ErrCurrencyPairEmpty},
		{"pair without a base", &WsAddOrderRequest{Pair: partialTestPairs[1], Order: valid}, currency.ErrCurrencyPairEmpty},
		{"order type", &WsAddOrderRequest{Pair: spotTestPair, Order: WsOrder{Side: "buy", Quantity: 1}}, order.ErrTypeIsInvalid},
		{"side", &WsAddOrderRequest{Pair: spotTestPair, Order: WsOrder{OrderType: "market", Quantity: 1}}, order.ErrSideIsInvalid},
		{"quantity", &WsAddOrderRequest{Pair: spotTestPair, Order: WsOrder{OrderType: "market", Side: "buy"}}, order.ErrAmountIsInvalid},
		{"identifiers", &WsAddOrderRequest{Pair: spotTestPair, Order: WsOrder{OrderType: "market", Side: "buy", Quantity: 1, ClientOrderID: "a", UserReference: 1}}, errClientOrderIDWithUserReference},
		{"expiry", &WsAddOrderRequest{Pair: spotTestPair, Order: WsOrder{OrderType: "limit", Side: "buy", Quantity: 1, LimitPrice: 26500, TimeInForce: "gtd"}}, order.ErrInvalidTimeInForce},
	} {
		_, err := e.WsAddOrder(t.Context(), tc.req)
		assert.ErrorIsf(t, err, tc.err, "WsAddOrder should reject the %s", tc.name)
	}

	ex := newWsTradingTestExchange(t,
		`{"method":"add_order","params":{"order_type":"stop-loss-limit","side":"buy","order_qty":1.2,"limit_price":28400,"limit_price_type":"static","triggers":{"reference":"last","price":28410,"price_type":"static"},"time_in_force":"gtd","margin":true,"post_only":true,"reduce_only":true,"effective_time":"2026-10-09T10:00:00Z","expire_time":"2026-10-10T10:00:00Z","cl_ord_id":"2c6be801-1f53-4f79-a0bb-4ea1c95dfae9","conditional":{"order_type":"take-profit-limit","limit_price":30000,"limit_price_type":"static","trigger_price":29900,"trigger_price_type":"static"},"display_qty":0.5,"fee_preference":"quote","stp_type":"cancel_newest","cash_order_qty":34000,"sender_sub_id":"trader-7","symbol":"BTC/USD","deadline":"2026-10-09T09:59:59.5Z","validate":true,"token":"`+wsTradingTestToken+`"}}`,
		`{"method":"add_order","result":{"order_id":"AA5JGQ-SBMRC-SCJ7J7","cl_ord_id":"2c6be801-1f53-4f79-a0bb-4ea1c95dfae9","warnings":["Order validated but not placed"]},"success":true,"time_in":"2023-09-21T14:15:07.197274Z","time_out":"2023-09-21T14:15:07.205301Z"}`,
	)
	sydney := time.FixedZone("AEST", 10*60*60)
	result, err := ex.WsAddOrder(t.Context(), &WsAddOrderRequest{
		Pair: spotTestPair,
		Order: WsOrder{
			OrderType:           "stop-loss-limit",
			Side:                "buy",
			Quantity:            1.2,
			LimitPrice:          28400,
			LimitPriceType:      "static",
			Triggers:            &WsOrderTriggers{Reference: "last", Price: 28410, PriceType: "static"},
			TimeInForce:         "gtd",
			Margin:              true,
			PostOnly:            true,
			ReduceOnly:          true,
			EffectiveTime:       time.Date(2026, 10, 9, 20, 0, 0, 0, sydney),
			ExpireTime:          time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC),
			ClientOrderID:       "2c6be801-1f53-4f79-a0bb-4ea1c95dfae9",
			Conditional:         &WsConditionalOrder{OrderType: "take-profit-limit", LimitPrice: 30000, LimitPriceType: "static", TriggerPrice: 29900, TriggerPriceType: "static"},
			DisplayQuantity:     0.5,
			FeePreference:       "quote",
			SelfTradePrevention: "cancel_newest",
			CashOrderQuantity:   34000,
			SenderSubID:         "trader-7",
		},
		Deadline: time.Date(2026, 10, 9, 19, 59, 59, 500000000, sydney),
		Validate: true,
	})
	require.NoError(t, err, "WsAddOrder must not error")
	assert.Equal(t, &WsAddOrderResponse{
		OrderID:       "AA5JGQ-SBMRC-SCJ7J7",
		ClientOrderID: "2c6be801-1f53-4f79-a0bb-4ea1c95dfae9",
		Warnings:      []string{"Order validated but not placed"},
		TimeIn:        wsTestTime(t, "2023-09-21T14:15:07.197274Z"),
		TimeOut:       wsTestTime(t, "2023-09-21T14:15:07.205301Z"),
	}, result, "WsAddOrder should decode every field")

	ex = newWsTradingTestExchange(t,
		`{"method":"add_order","params":{"order_type":"limit","side":"buy","limit_price":26500.4,"order_userref":100054,"order_qty":1.2,"symbol":"BTC/USD","token":"`+wsTradingTestToken+`"}}`,
		`{"error":"EOrder:Insufficient funds","method":"add_order","success":false,"time_in":"2023-09-21T14:15:07.197274Z","time_out":"2023-09-21T14:15:07.205301Z"}`,
	)
	_, err = ex.WsAddOrder(t.Context(), &WsAddOrderRequest{Pair: spotTestPair, Order: WsOrder{OrderType: "limit", Side: "buy", Quantity: 1.2, LimitPrice: 26500.4, UserReference: 100054}})
	require.ErrorIs(t, err, errAPIResponse, "WsAddOrder must return a rejection as an API error")
	assert.ErrorContains(t, err, "EOrder:Insufficient funds", "the error should carry Kraken's reason")
}

func TestWsEditOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WsEditOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsEditOrder must reject a nil request")
	_, err = e.WsEditOrder(t.Context(), &WsEditOrderRequest{Pair: spotTestPair})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "WsEditOrder must reject a missing order ID")
	_, err = e.WsEditOrder(t.Context(), &WsEditOrderRequest{OrderID: "ORDERX-IDXXX-XXXXX1"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "WsEditOrder must reject a missing pair")
	for _, pair := range partialTestPairs {
		_, err = e.WsEditOrder(t.Context(), &WsEditOrderRequest{OrderID: "ORDERX-IDXXX-XXXXX1", Pair: pair})
		require.ErrorIsf(t, err, currency.ErrCurrencyPairEmpty, "WsEditOrder must reject %q, which lacks a base or a quote", pair)
	}

	ex := newWsTradingTestExchange(t,
		`{"method":"edit_order","params":{"order_id":"ORDERX-IDXXX-XXXXX1","symbol":"BTC/USD","order_qty":0.2123456789,"limit_price":26500.5,"display_qty":0.1,"fee_preference":"base","post_only":true,"reduce_only":true,"triggers":{"reference":"index","price":-2,"price_type":"pct"},"order_userref":7,"validate":true,"deadline":"2026-10-09T09:59:59Z","token":"`+wsTradingTestToken+`"}}`,
		`{"method":"edit_order","result":{"order_id":"ORDERX-IDXXX-XXXXX2","original_order_id":"ORDERX-IDXXX-XXXXX1","warnings":["Order validated but not placed"]},"success":true,"time_in":"2022-07-15T12:56:09.876488Z","time_out":"2022-07-15T12:56:09.923422Z"}`,
	)
	result, err := ex.WsEditOrder(t.Context(), &WsEditOrderRequest{
		OrderID:         "ORDERX-IDXXX-XXXXX1",
		Pair:            spotTestPair,
		Quantity:        0.2123456789,
		LimitPrice:      26500.5,
		DisplayQuantity: 0.1,
		FeePreference:   "base",
		PostOnly:        true,
		ReduceOnly:      true,
		Triggers:        &WsOrderTriggers{Reference: "index", Price: -2, PriceType: "pct"},
		UserReference:   7,
		Validate:        true,
		Deadline:        time.Date(2026, 10, 9, 9, 59, 59, 0, time.UTC),
	})
	require.NoError(t, err, "WsEditOrder must not error")
	assert.Equal(t, &WsEditOrderResponse{
		OrderID:         "ORDERX-IDXXX-XXXXX2",
		OriginalOrderID: "ORDERX-IDXXX-XXXXX1",
		Warnings:        []string{"Order validated but not placed"},
		TimeIn:          wsTestTime(t, "2022-07-15T12:56:09.876488Z"),
		TimeOut:         wsTestTime(t, "2022-07-15T12:56:09.923422Z"),
	}, result, "WsEditOrder should decode every field")
}

func TestWsAmendOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WsAmendOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsAmendOrder must reject a nil request")
	_, err = e.WsAmendOrder(t.Context(), &WsAmendOrderRequest{LimitPrice: 1})
	require.ErrorIs(t, err, errOrderIdentifierRequired, "WsAmendOrder must reject a request without an identifier")
	_, err = e.WsAmendOrder(t.Context(), &WsAmendOrderRequest{OrderID: "OAIYAU-LGI3M-PFM5VW", ClientOrderID: "a", LimitPrice: 1})
	require.ErrorIs(t, err, errOrderIdentifierRequired, "WsAmendOrder must reject a request with both identifiers")
	for _, pair := range partialTestPairs {
		_, err = e.WsAmendOrder(t.Context(), &WsAmendOrderRequest{OrderID: "OAIYAU-LGI3M-PFM5VW", LimitPrice: 1, Pair: pair})
		require.ErrorIsf(t, err, currency.ErrCurrencyCodeEmpty, "WsAmendOrder must reject %q, which lacks a base or a quote", pair)
	}

	ex := newWsTradingTestExchange(t,
		`{"method":"amend_order","params":{"order_id":"OAIYAU-LGI3M-PFM5VW","symbol":"BTC/USD","order_qty":1.2,"display_qty":0.2,"limit_price":61031.3,"limit_price_type":"static","post_only":true,"trigger_price":61000,"trigger_price_type":"static","deadline":"2024-07-21T09:53:59.05Z","token":"`+wsTradingTestToken+`"}}`,
		`{"method":"amend_order","result":{"amend_id":"TTW6PD-RC36L-ZZSWNU","order_id":"OAIYAU-LGI3M-PFM5VW","cl_ord_id":"2c6be801-1f53-4f79-a0bb-4ea1c95dfae9","warnings":["Display quantity rounded"]},"success":true,"time_in":"2024-07-26T13:39:04.922699Z","time_out":"2024-07-26T13:39:04.924912Z"}`,
	)
	result, err := ex.WsAmendOrder(t.Context(), &WsAmendOrderRequest{
		OrderID:          "OAIYAU-LGI3M-PFM5VW",
		Pair:             spotTestPair,
		Quantity:         1.2,
		DisplayQuantity:  0.2,
		LimitPrice:       61031.3,
		LimitPriceType:   "static",
		PostOnly:         true,
		TriggerPrice:     61000,
		TriggerPriceType: "static",
		Deadline:         time.Date(2024, 7, 21, 9, 53, 59, 50000000, time.UTC),
	})
	require.NoError(t, err, "WsAmendOrder must not error")
	assert.Equal(t, &WsAmendOrderResponse{
		AmendID:       "TTW6PD-RC36L-ZZSWNU",
		OrderID:       "OAIYAU-LGI3M-PFM5VW",
		ClientOrderID: "2c6be801-1f53-4f79-a0bb-4ea1c95dfae9",
		Warnings:      []string{"Display quantity rounded"},
		TimeIn:        wsTestTime(t, "2024-07-26T13:39:04.922699Z"),
		TimeOut:       wsTestTime(t, "2024-07-26T13:39:04.924912Z"),
	}, result, "WsAmendOrder should decode every field")

	ex = newWsTradingTestExchange(t,
		`{"method":"amend_order","params":{"cl_ord_id":"2c6be801-1f53-4f79-a0bb-4ea1c95dfae9","limit_price":490795,"order_qty":1.2,"token":"`+wsTradingTestToken+`"}}`,
		`{"method":"amend_order","result":{"amend_id":"TTW6PD-RC36L-ZZSWNU","cl_ord_id":"2c6be801-1f53-4f79-a0bb-4ea1c95dfae9"},"success":true,"time_in":"2024-07-26T13:39:04.922699Z","time_out":"2024-07-26T13:39:04.924912Z"}`,
	)
	_, err = ex.WsAmendOrder(t.Context(), &WsAmendOrderRequest{ClientOrderID: "2c6be801-1f53-4f79-a0bb-4ea1c95dfae9", LimitPrice: 490795, Quantity: 1.2})
	assert.NoError(t, err, "WsAmendOrder should amend by client order ID")
}

func TestWsCancelOrders(t *testing.T) {
	t.Parallel()
	_, err := e.WsCancelOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsCancelOrders must reject a nil request")
	_, err = e.WsCancelOrders(t.Context(), &WsCancelOrdersRequest{})
	require.ErrorIs(t, err, errOrderIdentifierRequired, "WsCancelOrders must reject a request without identifiers")
	_, err = e.WsCancelOrders(t.Context(), &WsCancelOrdersRequest{OrderIDs: []string{"a"}, ClientOrderIDs: []string{"b"}})
	require.ErrorIs(t, err, errOrderIdentifierRequired, "WsCancelOrders must reject a request with two kinds of identifier")

	ex := newWsTradingTestExchange(t,
		`{"method":"cancel_order","params":{"order_id":["OM5CRX-N2HAL-GFGWE9","OLUMT4-UTEGU-ZYM7E9"],"token":"`+wsTradingTestToken+`"}}`,
		`{"error":"EOrder:Unknown order","method":"cancel_order","success":false,"time_in":"2023-09-21T14:36:57.428972Z","time_out":"2023-09-21T14:36:57.437952Z"}`,
		`{"method":"cancel_order","result":{"order_id":"OLUMT4-UTEGU-ZYM7E9","cl_ord_id":"6d1b345e-2821-40e2-ad83-4ecb18a06876","warnings":["Order already pending cancel"]},"success":true,"time_in":"2023-09-21T14:36:57.428972Z","time_out":"2023-09-21T14:36:57.437952Z"}`,
	)
	result, err := ex.WsCancelOrders(t.Context(), &WsCancelOrdersRequest{OrderIDs: []string{"OM5CRX-N2HAL-GFGWE9", "OLUMT4-UTEGU-ZYM7E9"}})
	require.NoError(t, err, "WsCancelOrders must not error")
	require.Len(t, result, 2, "WsCancelOrders must return a result for each order")
	assert.ErrorIs(t, result[0].Error, order.ErrOrderNotFound, "an order Kraken does not know should be reported as not found")
	result[0].Error = nil
	inTime, outTime := wsTestTime(t, "2023-09-21T14:36:57.428972Z"), wsTestTime(t, "2023-09-21T14:36:57.437952Z")
	assert.Equal(t, []WsCancelOrderResponse{
		{TimeIn: inTime, TimeOut: outTime},
		{OrderID: "OLUMT4-UTEGU-ZYM7E9", ClientOrderID: "6d1b345e-2821-40e2-ad83-4ecb18a06876", Warnings: []string{"Order already pending cancel"}, TimeIn: inTime, TimeOut: outTime},
	}, result, "WsCancelOrders should decode every field")

	for _, tc := range []struct {
		name string
		req  *WsCancelOrdersRequest
		exp  string
	}{
		{"client order IDs", &WsCancelOrdersRequest{ClientOrderIDs: []string{"6d1b345e-2821-40e2-ad83-4ecb18a06876"}}, `{"cl_ord_id":["6d1b345e-2821-40e2-ad83-4ecb18a06876"]`},
	} {
		ex := newWsTradingTestExchange(t,
			`{"method":"cancel_order","params":`+tc.exp+`,"token":"`+wsTradingTestToken+`"}}`,
			`{"method":"cancel_order","result":{"order_id":"OLUMT4-UTEGU-ZYM7E9"},"success":true}`,
		)
		_, err := ex.WsCancelOrders(t.Context(), tc.req)
		assert.NoErrorf(t, err, "WsCancelOrders should cancel by %s", tc.name)
	}
}

func TestWsCancelOrderParamsTakeNoUserReferences(t *testing.T) {
	t.Parallel()
	for field := range reflect.TypeFor[wsCancelOrderParams]().Fields() {
		tag := field.Tag.Get("json")
		assert.Falsef(t, strings.HasPrefix(tag, "order_userref"), "wsCancelOrderParams should not send %s, as a user reference's replies cannot be counted", tag)
	}
}

func TestWsCancelAllOrders(t *testing.T) {
	t.Parallel()
	ex := newWsTradingTestExchange(t,
		`{"method":"cancel_all","params":{"token":"`+wsTradingTestToken+`"}}`,
		`{"method":"cancel_all","result":{"count":1,"warnings":["Untriggered orders cancelled"]},"success":true,"time_in":"2023-09-26T13:09:48.463201Z","time_out":"2023-09-26T13:09:48.471419Z"}`,
	)
	result, err := ex.WsCancelAllOrders(t.Context())
	require.NoError(t, err, "WsCancelAllOrders must not error")
	assert.Equal(t, &WsCancelAllOrdersResponse{
		Count:    1,
		Warnings: []string{"Untriggered orders cancelled"},
		TimeIn:   wsTestTime(t, "2023-09-26T13:09:48.463201Z"),
		TimeOut:  wsTestTime(t, "2023-09-26T13:09:48.471419Z"),
	}, result, "WsCancelAllOrders should decode every field")
}

func TestWsCancelAllOrdersAfter(t *testing.T) {
	t.Parallel()
	for _, timeout := range []time.Duration{-time.Second, time.Millisecond, 24 * time.Hour} {
		_, err := e.WsCancelAllOrdersAfter(t.Context(), timeout)
		assert.ErrorIsf(t, err, errInvalidTimeout, "WsCancelAllOrdersAfter should reject a timeout of %s", timeout)
	}
	ex := newWsTradingTestExchange(t,
		`{"method":"cancel_all_orders_after","params":{"timeout":100,"token":"`+wsTradingTestToken+`"}}`,
		`{"method":"cancel_all_orders_after","result":{"currentTime":"2023-09-21T15:49:29Z","triggerTime":"2023-09-21T15:51:09Z"},"success":true,"time_in":"2023-09-21T15:49:28.627900Z","time_out":"2023-09-21T15:49:28.649057Z"}`,
	)
	result, err := ex.WsCancelAllOrdersAfter(t.Context(), 100*time.Second+time.Millisecond)
	require.NoError(t, err, "WsCancelAllOrdersAfter must not error")
	assert.Equal(t, &WsCancelAllOrdersAfterResponse{
		CurrentTime: wsTestTime(t, "2023-09-21T15:49:29Z"),
		TriggerTime: wsTestTime(t, "2023-09-21T15:51:09Z"),
		TimeIn:      wsTestTime(t, "2023-09-21T15:49:28.6279Z"),
		TimeOut:     wsTestTime(t, "2023-09-21T15:49:28.649057Z"),
	}, result, "WsCancelAllOrdersAfter should decode every field")

	ex = newWsTradingTestExchange(t,
		`{"method":"cancel_all_orders_after","params":{"timeout":0,"token":"`+wsTradingTestToken+`"}}`,
		`{"method":"cancel_all_orders_after","result":{"currentTime":"2023-09-21T15:49:29Z","triggerTime":"2023-09-21T15:49:29Z"},"success":true}`,
	)
	_, err = ex.WsCancelAllOrdersAfter(t.Context(), 0)
	assert.NoError(t, err, "WsCancelAllOrdersAfter should send a zero timeout to disable the switch")
}

func TestWsBatchAddOrders(t *testing.T) {
	t.Parallel()
	_, err := e.WsBatchAddOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsBatchAddOrders must reject a nil request")
	valid := WsOrder{OrderType: "limit", Side: "buy", Quantity: 1, LimitPrice: 1010.1}
	for _, n := range []int{1, 16} {
		_, err = e.WsBatchAddOrders(t.Context(), &WsBatchAddOrdersRequest{Pair: spotTestPair, Orders: slices.Repeat([]WsOrder{valid}, n)})
		assert.ErrorIsf(t, err, errBatchSize, "WsBatchAddOrders should reject %d orders", n)
	}
	_, err = e.WsBatchAddOrders(t.Context(), &WsBatchAddOrdersRequest{Orders: []WsOrder{valid, valid}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "WsBatchAddOrders must reject a missing pair")
	for _, pair := range partialTestPairs {
		_, err = e.WsBatchAddOrders(t.Context(), &WsBatchAddOrdersRequest{Pair: pair, Orders: []WsOrder{valid, valid}})
		require.ErrorIsf(t, err, currency.ErrCurrencyPairEmpty, "WsBatchAddOrders must reject %q, which lacks a base or a quote", pair)
	}
	_, err = e.WsBatchAddOrders(t.Context(), &WsBatchAddOrdersRequest{Pair: spotTestPair, Orders: []WsOrder{valid, {OrderType: "limit", Side: "sell"}}})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "WsBatchAddOrders must reject an invalid order")

	ex := newWsTradingTestExchange(t,
		`{"method":"batch_add","params":{"deadline":"2022-06-13T08:09:10.123456Z","orders":[{"limit_price":1010.1,"order_qty":0.123456789,"order_type":"limit","order_userref":1,"side":"buy"},{"limit_price":2020.2,"order_qty":0.987654321,"order_type":"limit","order_userref":2,"side":"sell","stp_type":"cancel_both"}],"symbol":"BTC/USD","token":"`+wsTradingTestToken+`"}}`,
		`{"method":"batch_add","result":[{"order_id":"ORDERX-IDXXX-XXXXX1","order_userref":1},{"order_id":"ORDERX-IDXXX-XXXXX2","order_userref":2,"cl_ord_id":"","warnings":["Self trade prevention applied"]}],"success":true,"time_in":"2022-06-13T08:09:10.123456Z","time_out":"2022-06-13T08:09:10.7890123Z"}`,
	)
	result, err := ex.WsBatchAddOrders(t.Context(), &WsBatchAddOrdersRequest{
		Pair: spotTestPair,
		Orders: []WsOrder{
			{OrderType: "limit", Side: "buy", Quantity: 0.123456789, LimitPrice: 1010.1, UserReference: 1},
			{OrderType: "limit", Side: "sell", Quantity: 0.987654321, LimitPrice: 2020.2, UserReference: 2, SelfTradePrevention: "cancel_both"},
		},
		Deadline: time.Date(2022, 6, 13, 8, 9, 10, 123456000, time.UTC),
	})
	require.NoError(t, err, "WsBatchAddOrders must not error")
	inTime, outTime := wsTestTime(t, "2022-06-13T08:09:10.123456Z"), wsTestTime(t, "2022-06-13T08:09:10.7890123Z")
	assert.Equal(t, []WsAddOrderResponse{
		{OrderID: "ORDERX-IDXXX-XXXXX1", OrderUserReference: 1, TimeIn: inTime, TimeOut: outTime},
		{OrderID: "ORDERX-IDXXX-XXXXX2", OrderUserReference: 2, Warnings: []string{"Self trade prevention applied"}, TimeIn: inTime, TimeOut: outTime},
	}, result, "WsBatchAddOrders should decode every order's result")
}

func TestWsBatchCancelOrders(t *testing.T) {
	t.Parallel()
	_, err := e.WsBatchCancelOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsBatchCancelOrders must reject a nil request")
	for _, n := range []int{1, 51} {
		_, err = e.WsBatchCancelOrders(t.Context(), &WsBatchCancelOrdersRequest{Orders: slices.Repeat([]string{"1"}, n)})
		assert.ErrorIsf(t, err, errBatchSize, "WsBatchCancelOrders should reject %d orders", n)
	}
	ex := newWsTradingTestExchange(t,
		`{"method":"batch_cancel","params":{"orders":["1","2","ORDERX-IDXXX-XXXXX3"],"cl_ord_id":["6d1b345e-2821-40e2-ad83-4ecb18a06876"],"token":"`+wsTradingTestToken+`"}}`,
		`{"method":"batch_cancel","result":{"count":4,"warnings":["Order already pending cancel"]},"success":true,"time_in":"2022-06-13T08:09:10.123456Z","time_out":"2022-06-13T08:09:10.7890123Z"}`,
	)
	result, err := ex.WsBatchCancelOrders(t.Context(), &WsBatchCancelOrdersRequest{
		Orders:         []string{"1", "2", "ORDERX-IDXXX-XXXXX3"},
		ClientOrderIDs: []string{"6d1b345e-2821-40e2-ad83-4ecb18a06876"},
	})
	require.NoError(t, err, "WsBatchCancelOrders must not error")
	assert.Equal(t, &WsBatchCancelOrdersResponse{
		Count:    4,
		Warnings: []string{"Order already pending cancel"},
		TimeIn:   wsTestTime(t, "2022-06-13T08:09:10.123456Z"),
		TimeOut:  wsTestTime(t, "2022-06-13T08:09:10.7890123Z"),
	}, result, "WsBatchCancelOrders should decode every field")
}

func TestWsPrivateRequestNotConnected(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	_, err := ex.WsCancelAllOrders(t.Context())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "a private request should fail without a private connection")
}

func TestWsPrivateRequestTokenError(t *testing.T) {
	t.Parallel()
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{"error":["EAPI:Invalid key"]}`))
		assert.NoError(t, err, "Write should not error")
	})
	seedTestAssetNames(ex)
	mock := func(tb testing.TB, msg []byte, _ *gws.Conn) error {
		tb.Helper()
		assert.Failf(tb, "no request should be sent without a token", "%s", msg)
		return nil
	}
	ex.Features.Subscriptions = subscription.List{}
	useTestWebsocket(t, ex, newMockWebsocketURL(t, mock), wsPrivateConnection)
	ex.Websocket.SetSubscriptionsNotRequired()
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	_, err := ex.WsCancelAllOrders(t.Context())
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "a private request should fail when its token cannot be fetched")
}

func TestWsOrderParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		order WsOrder
		err   error
	}{
		{name: "order type", order: WsOrder{Side: "buy", Quantity: 1}, err: order.ErrTypeIsInvalid},
		{name: "side", order: WsOrder{OrderType: "limit", Quantity: 1}, err: order.ErrSideIsInvalid},
		{name: "quantity", order: WsOrder{OrderType: "limit", Side: "buy"}, err: order.ErrAmountIsInvalid},
		{name: "client order ID beside a user reference", order: WsOrder{OrderType: "limit", Side: "buy", Quantity: 1, ClientOrderID: "a", UserReference: 1}, err: errClientOrderIDWithUserReference},
		{name: "GTD without an expiry", order: WsOrder{OrderType: "limit", Side: "buy", Quantity: 1, TimeInForce: "gtd"}, err: order.ErrInvalidTimeInForce},
	} {
		_, err := tc.order.params()
		assert.ErrorIsf(t, err, tc.err, "params should reject an order by its %s", tc.name)
	}
	aest := time.FixedZone("AEST", 10*60*60)
	triggers := &WsOrderTriggers{Reference: "last", Price: 28410, PriceType: "static"}
	o := WsOrder{
		OrderType:           "stop-loss-limit",
		Side:                "buy",
		Quantity:            1.2,
		LimitPrice:          28400,
		LimitPriceType:      "static",
		Triggers:            triggers,
		TimeInForce:         "gtd",
		Margin:              true,
		PostOnly:            true,
		ReduceOnly:          true,
		EffectiveTime:       time.Date(2026, 10, 9, 20, 0, 0, 0, aest),
		ExpireTime:          time.Date(2026, 10, 10, 20, 0, 0, 0, aest),
		ClientOrderID:       "2c6be801-1f53-4f79-a0bb-4ea1c95dfae9",
		DisplayQuantity:     0.5,
		FeePreference:       "quote",
		SelfTradePrevention: "cancel_newest",
		CashOrderQuantity:   34000,
		SenderSubID:         "trader-7",
	}
	params, err := o.params()
	require.NoError(t, err, "params must not error")
	exp := &wsOrderParams{
		OrderType:           "stop-loss-limit",
		Side:                "buy",
		OrderQuantity:       1.2,
		LimitPrice:          28400,
		LimitPriceType:      "static",
		Triggers:            triggers,
		TimeInForce:         "gtd",
		Margin:              true,
		PostOnly:            true,
		ReduceOnly:          true,
		EffectiveTime:       "2026-10-09T10:00:00Z",
		ExpireTime:          "2026-10-10T10:00:00Z",
		ClientOrderID:       "2c6be801-1f53-4f79-a0bb-4ea1c95dfae9",
		DisplayQuantity:     0.5,
		FeePreference:       "quote",
		SelfTradePrevention: "cancel_newest",
		CashOrderQuantity:   34000,
		SenderSubID:         "trader-7",
	}
	assert.Equal(t, exp, params, "params should send every field set, with times in UTC")
}

func TestWsPrivateRequest(t *testing.T) {
	t.Parallel()
	cancelAll := `{"method":"cancel_all","params":{"token":"` + wsTradingTestToken + `"}}`
	ex := newWsTradingTestExchange(t, cancelAll, `{"method":"cancel_all","result":{"count":1},"success":true,"time_in":"2023-09-26T13:09:48.463201Z","time_out":"2023-09-26T13:09:48.471419Z"}`)
	resp, err := ex.wsPrivateRequest(t.Context(), "cancel_all", &wsTokenParams{})
	require.NoError(t, err, "wsPrivateRequest must not error")
	require.NotNil(t, resp, "wsPrivateRequest must return the reply")
	assert.True(t, resp.Success, "wsPrivateRequest should return a successful reply")
	assert.JSONEq(t, `{"count":1}`, string(resp.Result), "wsPrivateRequest should return the reply's result")

	ex = newWsTradingTestExchange(t, cancelAll, `{"error":"EGeneral:Internal error","method":"cancel_all","success":false,"time_in":"2023-09-26T13:09:48.463201Z","time_out":"2023-09-26T13:09:48.471419Z"}`)
	_, err = ex.wsPrivateRequest(t.Context(), "cancel_all", &wsTokenParams{})
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "wsPrivateRequest must return a failed reply's error")
	assert.Equal(t, []string{"EGeneral:Internal error"}, apiErr.Errors, "wsPrivateRequest should return the error Kraken replies with")
}

func TestWsPrivateRequests(t *testing.T) {
	t.Parallel()
	_, err := newTestExchange(t).wsPrivateRequests(t.Context(), "cancel_all", &wsTokenParams{}, 1)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "wsPrivateRequests should reject a request without a private connection")
	ex := newWsTradingTestExchange(t,
		`{"method":"cancel_all","params":{"token":"`+wsTradingTestToken+`"}}`,
		`{"method":"cancel_all","result":{"count":1},"success":true,"time_in":"2023-09-26T13:09:48.463201Z","time_out":"2023-09-26T13:09:48.471419Z"}`,
		`{"error":"EGeneral:Internal error","method":"cancel_all","success":false,"time_in":"2023-09-26T13:09:48.463201Z","time_out":"2023-09-26T13:09:48.471419Z"}`,
	)
	params := &wsTokenParams{}
	resps, err := ex.wsPrivateRequests(t.Context(), "cancel_all", params, 2)
	require.NoError(t, err, "wsPrivateRequests must not error")
	assert.Equal(t, wsTradingTestToken, params.Token, "wsPrivateRequests should add the token to the request")
	require.Len(t, resps, 2, "wsPrivateRequests must return the replies expected")
	assert.True(t, resps[0].Success, "wsPrivateRequests should decode a successful reply")
	assert.Equal(t, "EGeneral:Internal error", resps[1].Error, "wsPrivateRequests should decode a failed reply without erroring")
}

func TestWsTokenParamsSetToken(t *testing.T) {
	t.Parallel()
	p := &wsTokenParams{}
	p.setToken(wsTradingTestToken)
	assert.Equal(t, wsTradingTestToken, p.Token, "setToken should set the token")
}
