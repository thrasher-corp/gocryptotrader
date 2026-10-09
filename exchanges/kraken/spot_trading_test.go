package kraken

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

// tradingTestDeadline is 2026-10-09T00:10:30Z, given in another zone to prove deadlines are sent in UTC
var tradingTestDeadline = time.Date(2026, 10, 9, 10, 10, 30, 0, time.FixedZone("AEST", 10*60*60))

// tradingRequestBody checks a Spot REST private request is signed and returns its JSON body without the nonce, which
// varies. Handlers run on the server's goroutine, so it only asserts
func tradingRequestBody(t *testing.T, r *http.Request) string {
	t.Helper()
	assert.Equal(t, http.MethodPost, r.Method, "Method should be POST")
	var payload map[string]json.RawMessage
	body := checkSpotSignedRequest(t, r, func(body []byte) string {
		assert.NoError(t, json.Unmarshal(body, &payload), "Unmarshal should not error")
		return string(payload["nonce"])
	})
	assert.NotEmpty(t, body, "request should carry a body")
	assert.Contains(t, payload, "nonce", "body should carry the nonce")
	delete(payload, "nonce")
	stripped, err := json.Marshal(payload)
	assert.NoError(t, err, "Marshal should not error")
	return string(stripped)
}

func TestAddOrder(t *testing.T) {
	t.Parallel()
	_, err := e.AddOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AddOrder must reject a nil request")
	_, err = e.AddOrder(t.Context(), &AddOrderRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "AddOrder must reject an empty pair")
	valid := OrderParameters{OrderType: "limit", Side: "buy", Volume: 1, Price: OrderPrice{Value: 25000}}
	for _, tc := range []struct {
		name   string
		modify func(*OrderParameters)
		err    error
	}{
		{name: "an invalid side", modify: func(o *OrderParameters) { o.Side = "long" }, err: order.ErrSideIsInvalid},
		{name: "an invalid order type", modify: func(o *OrderParameters) { o.OrderType = "stop" }, err: order.ErrTypeIsInvalid},
		{name: "a negative volume", modify: func(o *OrderParameters) { o.Volume = -1 }, err: order.ErrAmountIsInvalid},
		{name: "a negative display volume", modify: func(o *OrderParameters) { o.DisplayVolume = -0.1 }, err: order.ErrAmountIsInvalid},
		{name: "a zero volume without leverage", modify: func(o *OrderParameters) { o.Volume = 0 }, err: order.ErrAmountIsInvalid},
		{name: "a user reference with a client order ID", modify: func(o *OrderParameters) { o.UserReference, o.ClientOrderID = 1, "arb-20240509-00010" }, err: errMultipleOrderIdentifiers},
		{name: "fees in both currencies", modify: func(o *OrderParameters) { o.FeeInBase, o.FeeInQuote = true, true }, err: errConflictingOrderFlags},
		{name: "an undocumented time in force", modify: func(o *OrderParameters) { o.TimeInForce = "GTX" }, err: order.ErrInvalidTimeInForce},
		{name: "a GTD order without an expiry", modify: func(o *OrderParameters) { o.TimeInForce = "GTD" }, err: order.ErrInvalidTimeInForce},
		{name: "a negative price", modify: func(o *OrderParameters) { o.Price = OrderPrice{Value: -1} }, err: errInvalidOrderPrice},
		{name: "an undocumented price offset", modify: func(o *OrderParameters) { o.Price = OrderPrice{Value: 1, Offset: "~"} }, err: errInvalidOrderPrice},
		{name: "an undocumented secondary price offset", modify: func(o *OrderParameters) { o.SecondaryPrice = OrderPrice{Value: 1, Offset: "+-"} }, err: errInvalidOrderPrice},
		{name: "an absolute trailing stop price", modify: func(o *OrderParameters) { o.OrderType = "trailing-stop" }, err: errInvalidOrderPrice},
		{
			name: "a trailing stop limit price offset by #",
			modify: func(o *OrderParameters) {
				o.OrderType, o.Price, o.SecondaryPrice = "trailing-stop-limit", OrderPrice{Value: 5, Offset: "+"}, OrderPrice{Value: 5, Offset: "#"}
			},
			err: errInvalidOrderPrice,
		},
		{name: "a start time with a start delay", modify: func(o *OrderParameters) { o.StartTime, o.StartDelay = time.Unix(1791590400, 0), time.Minute }, err: errInvalidOrderTime},
		{name: "a negative start delay", modify: func(o *OrderParameters) { o.StartDelay = -time.Second }, err: errInvalidOrderTime},
		{name: "a start delay under a second", modify: func(o *OrderParameters) { o.StartDelay = 999 * time.Millisecond }, err: errInvalidOrderTime},
		{name: "an expiry delay under 5 seconds", modify: func(o *OrderParameters) { o.ExpireDelay = 5*time.Second - time.Nanosecond }, err: errInvalidOrderTime},
		{name: "a market conditional close order", modify: func(o *OrderParameters) { o.Close = &ConditionalCloseOrder{OrderType: "market"} }, err: order.ErrTypeIsInvalid},
		{
			name: "an absolute trailing stop conditional close price",
			modify: func(o *OrderParameters) {
				o.Close = &ConditionalCloseOrder{OrderType: "trailing-stop", Price: OrderPrice{Value: 24000}}
			},
			err: errInvalidOrderPrice,
		},
	} {
		o := valid
		tc.modify(&o)
		_, err = e.AddOrder(t.Context(), &AddOrderRequest{Pair: spotTestPair, Order: o})
		require.ErrorIsf(t, err, tc.err, "AddOrder must reject %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *AddOrderRequest
		exp  *AddOrderResponse
	}{
		{
			name: "limit order with a conditional close order",
			req: &AddOrderRequest{
				Pair: spotTestPair,
				Order: OrderParameters{
					UserReference:       12345,
					OrderType:           "limit",
					Side:                "buy",
					Volume:              2.1234,
					Price:               OrderPrice{Value: 25000.1},
					TriggerSignal:       "index",
					Leverage:            2,
					SelfTradePrevention: "cancel-both",
					PostOnly:            true,
					FeeInQuote:          true,
					TimeInForce:         "GTD",
					StartTime:           time.Unix(1791590400, 0),
					ExpireTime:          time.Unix(1791676800, 0),
					Close:               &ConditionalCloseOrder{OrderType: "stop-loss-limit", Price: OrderPrice{Value: 22000}, SecondaryPrice: OrderPrice{Value: 21000}},
				},
				Deadline: tradingTestDeadline,
			},
			exp: &AddOrderResponse{
				Description: AddOrderDescription{
					Order: "buy 2.12340000 XBTUSD @ limit 25000.1 with 2:1 leverage",
					Close: "close position @ stop loss 22000.0 -> limit 21000.0",
				},
				TransactionIDs: []string{"OUF4EM-FRGI2-MQMWZD"},
			},
		},
		{
			name: "trailing stop limit order with relative prices and delays",
			req: &AddOrderRequest{
				Pair: spotTestPair,
				Order: OrderParameters{
					ClientOrderID:  "6d1b345e-2821-40e2-ad83-4ecb18a06876",
					OrderType:      "trailing-stop-limit",
					Side:           "sell",
					Volume:         0.5,
					Price:          OrderPrice{Value: 5, Offset: "+", Percent: true},
					SecondaryPrice: OrderPrice{Value: 100, Offset: "-"},
					TriggerSignal:  "last",
					Leverage:       3,
					ReduceOnly:     true,
					FeeInBase:      true,
					TimeInForce:    "GTD",
					StartDelay:     time.Minute,
					ExpireDelay:    time.Hour,
				},
			},
			exp: &AddOrderResponse{
				Description:    AddOrderDescription{Order: "sell 0.50000000 XBTUSD @ trailing stop +5.0000% -> limit -100.0 with 3:1 leverage"},
				TransactionIDs: []string{"OHQXPI-3JDGU-PBOZ3V"},
			},
		},
		{
			name: "iceberg order with a relative conditional close order",
			req: &AddOrderRequest{
				Pair: spotTestPair,
				Order: OrderParameters{
					OrderType:     "iceberg",
					Side:          "buy",
					Volume:        1.5,
					DisplayVolume: 0.1,
					Price:         OrderPrice{Value: 10, Offset: "#"},
					TimeInForce:   "GTC",
					Close:         &ConditionalCloseOrder{OrderType: "take-profit", Price: OrderPrice{Value: 2.5, Offset: "#", Percent: true}},
				},
			},
			exp: &AddOrderResponse{
				Description: AddOrderDescription{
					Order: "buy 1.50000000 XBTUSD @ iceberg #10.0",
					Close: "close position @ take profit #2.5000%",
				},
				TransactionIDs: []string{"OKTIUZ-6NBNQ-W3JOVA"},
			},
		},
		{
			name: "validated market order sized in the quote currency",
			req: &AddOrderRequest{
				Pair:        spotTestPair,
				AssetClass:  "tokenized_asset",
				Order:       OrderParameters{OrderType: "market", Side: "buy", Volume: 100, VolumeInQuote: true, TimeInForce: "FOK"},
				Validate:    true,
				BrokerIIBAN: "AA12 N84G WQRX 4ASV",
			},
			exp: &AddOrderResponse{Description: AddOrderDescription{Order: "buy 100.00000000 XBTUSD @ market"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.AddOrder(t.Context(), tc.req)
			require.NoError(t, err, "AddOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "AddOrder should decode every field")
				return
			}
			assert.NotEmpty(t, result.Description.Order, "AddOrder should describe the order")
		})
	}
}

func TestAmendOrder(t *testing.T) {
	t.Parallel()
	_, err := e.AmendOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AmendOrder must reject a nil request")
	_, err = e.AmendOrder(t.Context(), &AmendOrderRequest{OrderQuantity: 1})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "AmendOrder must reject a request without an order identifier")
	_, err = e.AmendOrder(t.Context(), &AmendOrderRequest{TransactionID: "OHYO67-6LP66-HMQ437", ClientOrderID: "arb-20240509-00010"})
	require.ErrorIs(t, err, errMultipleOrderIdentifiers, "AmendOrder must reject two order identifiers")
	_, err = e.AmendOrder(t.Context(), &AmendOrderRequest{TransactionID: "OHYO67-6LP66-HMQ437", DisplayQuantity: -1})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "AmendOrder must reject a negative quantity")
	_, err = e.AmendOrder(t.Context(), &AmendOrderRequest{TransactionID: "OHYO67-6LP66-HMQ437", LimitPrice: OrderPrice{Value: 50, Offset: "#"}})
	require.ErrorIs(t, err, errInvalidOrderPrice, "AmendOrder must reject a limit price offset by #")
	_, err = e.AmendOrder(t.Context(), &AmendOrderRequest{TransactionID: "OHYO67-6LP66-HMQ437", TriggerPrice: OrderPrice{Value: 50, Offset: "#"}})
	require.ErrorIs(t, err, errInvalidOrderPrice, "AmendOrder must reject a trigger price offset by #")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *AmendOrderRequest
		exp  *AmendOrderResponse
	}{
		{
			name: "quantity and limit price by transaction ID",
			req:  &AmendOrderRequest{TransactionID: "OHYO67-6LP66-HMQ437", OrderQuantity: 1.25, LimitPrice: OrderPrice{Value: 81000.5}, PostOnly: true, Deadline: tradingTestDeadline},
			exp:  &AmendOrderResponse{AmendID: "TEZA4R-DSDGT-IJBOJK"},
		},
		{
			name: "display quantity and relative trigger price by client order ID",
			req:  &AmendOrderRequest{ClientOrderID: "6d1b345e-2821-40e2-ad83-4ecb18a06876", DisplayQuantity: 0.1, TriggerPrice: OrderPrice{Value: 2.5, Offset: "-", Percent: true}, Pair: spotTestPair},
			exp:  &AmendOrderResponse{AmendID: "TUFTDH-GKJ6E-NOBVEX"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.AmendOrder(t.Context(), tc.req)
			require.NoError(t, err, "AmendOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "AmendOrder should decode every field")
				return
			}
			assert.NotEmpty(t, result.AmendID, "AmendOrder should return an amend ID")
		})
	}
}

func TestEditOrder(t *testing.T) {
	t.Parallel()
	_, err := e.EditOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "EditOrder must reject a nil request")
	_, err = e.EditOrder(t.Context(), &EditOrderRequest{TransactionID: "OHYO67-6LP66-HMQ437"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "EditOrder must reject an empty pair")
	_, err = e.EditOrder(t.Context(), &EditOrderRequest{Pair: spotTestPair})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "EditOrder must reject a request without an order identifier")
	_, err = e.EditOrder(t.Context(), &EditOrderRequest{Pair: spotTestPair, TransactionID: "OHYO67-6LP66-HMQ437", OrderUserReference: 1234})
	require.ErrorIs(t, err, errMultipleOrderIdentifiers, "EditOrder must reject two order identifiers")
	_, err = e.EditOrder(t.Context(), &EditOrderRequest{Pair: spotTestPair, TransactionID: "OHYO67-6LP66-HMQ437", Volume: -1})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "EditOrder must reject a negative volume")
	_, err = e.EditOrder(t.Context(), &EditOrderRequest{Pair: spotTestPair, TransactionID: "OHYO67-6LP66-HMQ437", Price: OrderPrice{Value: -1}})
	require.ErrorIs(t, err, errInvalidOrderPrice, "EditOrder must reject a negative price")
	_, err = e.EditOrder(t.Context(), &EditOrderRequest{Pair: spotTestPair, TransactionID: "OHYO67-6LP66-HMQ437", SecondaryPrice: OrderPrice{Value: 1, Offset: "%"}})
	require.ErrorIs(t, err, errInvalidOrderPrice, "EditOrder must reject an undocumented secondary price offset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *EditOrderRequest
		exp  *EditOrderResponse
	}{
		{
			name: "replacement by transaction ID",
			req: &EditOrderRequest{
				TransactionID:    "OHYO67-6LP66-HMQ437",
				NewUserReference: 666,
				Pair:             spotTestPair,
				Volume:           0.0003,
				Price:            OrderPrice{Value: 19500},
				SecondaryPrice:   OrderPrice{Value: 32500},
				PostOnly:         true,
				Deadline:         tradingTestDeadline,
				CancelResponse:   true,
			},
			exp: &EditOrderResponse{
				Description:           EditOrderDescription{Order: "buy 0.00030000 XXBTZUSD @ limit 19500.0"},
				TransactionID:         "OFVXHJ-KPQ3B-VS7ELA",
				NewUserReference:      666,
				OldUserReference:      1234,
				OrdersCancelled:       1,
				OriginalTransactionID: "OHYO67-6LP66-HMQ437",
				Status:                "ok",
				Volume:                0.0003,
				Price:                 OrderPrice{Value: 19500},
				SecondaryPrice:        OrderPrice{Value: 32500},
			},
		},
		{
			name: "failed validation by user reference",
			req: &EditOrderRequest{
				OrderUserReference: 1234,
				Pair:               spotTestPair,
				AssetClass:         "tokenized_asset",
				DisplayVolume:      0.0001,
				Price:              OrderPrice{Value: 1.5, Offset: "+", Percent: true},
				Validate:           true,
			},
			exp: &EditOrderResponse{
				Description:           EditOrderDescription{Order: "buy 0.00020000 XXBTZUSD @ iceberg 19400.0"},
				OldUserReference:      1234,
				OriginalTransactionID: "OKTIUZ-6NBNQ-W3JOVA",
				Status:                "err",
				Volume:                0.0002,
				Price:                 OrderPrice{Value: 19400},
				ErrorMessage:          "EOrder:Insufficient funds",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.EditOrder(t.Context(), tc.req)
			require.NoError(t, err, "EditOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "EditOrder should decode every field")
				return
			}
			assert.NotEmpty(t, result.Status, "EditOrder should return a status")
		})
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelExistingOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelExistingOrder must reject a nil request")
	_, err = e.CancelExistingOrder(t.Context(), &CancelExistingOrderRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelExistingOrder must reject a request without an order identifier")
	_, err = e.CancelExistingOrder(t.Context(), &CancelExistingOrderRequest{TransactionID: "OYVGEW-VYV5B-UUEXSK", UserReference: 1680953421})
	require.ErrorIs(t, err, errMultipleOrderIdentifiers, "CancelExistingOrder must reject a transaction ID with another identifier")
	_, err = e.CancelExistingOrder(t.Context(), &CancelExistingOrderRequest{UserReference: 1680953421, ClientOrderID: "arb-20240509-00010"})
	require.ErrorIs(t, err, errMultipleOrderIdentifiers, "CancelExistingOrder must reject a user reference with a client order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *CancelExistingOrderRequest
		exp  *CancelOrderResponse
	}{
		{name: "transaction ID", req: &CancelExistingOrderRequest{TransactionID: "OYVGEW-VYV5B-UUEXSK"}, exp: &CancelOrderResponse{Count: 1, Pending: true}},
		{name: "user reference", req: &CancelExistingOrderRequest{UserReference: 1680953421}, exp: &CancelOrderResponse{Count: 3, Pending: true}},
		{name: "client order ID", req: &CancelExistingOrderRequest{ClientOrderID: "6d1b345e-2821-40e2-ad83-4ecb18a06876"}, exp: &CancelOrderResponse{Count: 2, Pending: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.CancelExistingOrder(t.Context(), tc.req)
			require.NoError(t, err, "CancelExistingOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "CancelExistingOrder should decode every field")
				return
			}
			assert.NotNil(t, result, "CancelExistingOrder should return a result")
		})
	}
}

func TestCancelAllOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelAllOpenOrders(t.Context())
	require.NoError(t, err, "CancelAllOpenOrders must not error")
	if mockTests {
		assert.Equal(t, &CancelOrderResponse{Count: 4, Pending: true}, result, "CancelAllOpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelAllOpenOrders should return a result")
}

func TestCancelAllOrdersAfter(t *testing.T) {
	t.Parallel()
	for _, timeout := range []time.Duration{-time.Second, time.Second - time.Nanosecond, 24 * time.Hour} {
		_, err := e.CancelAllOrdersAfter(t.Context(), timeout)
		require.ErrorIsf(t, err, errInvalidCancelTimeout, "CancelAllOrdersAfter must reject a timeout of %s", timeout)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	// Disabling the switch last leaves it off after a live run
	for _, tc := range []struct {
		name    string
		timeout time.Duration
		exp     *CancelAllOrdersAfterResponse
	}{
		{
			name:    "armed",
			timeout: time.Minute,
			exp:     &CancelAllOrdersAfterResponse{CurrentTime: time.Date(2026, 10, 9, 0, 10, 2, 0, time.UTC), TriggerTime: time.Date(2026, 10, 9, 0, 11, 2, 0, time.UTC)},
		},
		{
			name: "disabled",
			exp:  &CancelAllOrdersAfterResponse{CurrentTime: time.Date(2026, 10, 9, 0, 10, 5, 0, time.UTC)},
		},
	} {
		result, err := e.CancelAllOrdersAfter(t.Context(), tc.timeout)
		require.NoErrorf(t, err, "CancelAllOrdersAfter must not error when %s", tc.name)
		if mockTests {
			assert.Equalf(t, tc.exp, result, "CancelAllOrdersAfter should decode every field when %s", tc.name)
			continue
		}
		assert.NotZerof(t, result.CurrentTime, "CancelAllOrdersAfter should return the current time when %s", tc.name)
	}
}

func TestCancelAllOrdersAfterResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var resp CancelAllOrdersAfterResponse
	require.NoError(t, resp.UnmarshalJSON([]byte(`{"currentTime":"2026-10-09T00:10:02.5Z","triggerTime":"2026-10-09T00:11:02.5Z"}`)), "UnmarshalJSON must not error for an armed timer")
	assert.Equal(t, CancelAllOrdersAfterResponse{CurrentTime: time.Date(2026, 10, 9, 0, 10, 2, 5e8, time.UTC), TriggerTime: time.Date(2026, 10, 9, 0, 11, 2, 5e8, time.UTC)}, resp, "UnmarshalJSON should decode fractional seconds")
	require.NoError(t, resp.UnmarshalJSON([]byte(`{"currentTime":"2026-10-09T00:10:05Z","triggerTime":"0"}`)), "UnmarshalJSON must not error for a disabled timer")
	assert.Zero(t, resp.TriggerTime, "UnmarshalJSON should clear the trigger time of a disabled timer")
	assert.Error(t, resp.UnmarshalJSON([]byte(`{"currentTime":"2026-10-09T00:10:05Z","triggerTime":"soon"}`)), "UnmarshalJSON should reject a trigger time that is neither RFC3339 nor 0")
	assert.Error(t, resp.UnmarshalJSON([]byte(`[]`)), "UnmarshalJSON should reject an array")
}

func TestAddOrderBatch(t *testing.T) {
	t.Parallel()
	_, err := e.AddOrderBatch(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AddOrderBatch must reject a nil request")
	limitOrder := OrderParameters{OrderType: "limit", Side: "buy", Volume: 1, Price: OrderPrice{Value: 25000}}
	_, err = e.AddOrderBatch(t.Context(), &AddOrderBatchRequest{Orders: []OrderParameters{limitOrder, limitOrder}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "AddOrderBatch must reject an empty pair")
	for _, orders := range [][]OrderParameters{{limitOrder}, make([]OrderParameters, 16)} {
		_, err = e.AddOrderBatch(t.Context(), &AddOrderBatchRequest{Pair: spotTestPair, Orders: orders})
		require.ErrorIsf(t, err, errInvalidCount, "AddOrderBatch must reject %d orders", len(orders))
	}
	_, err = e.AddOrderBatch(t.Context(), &AddOrderBatchRequest{Pair: spotTestPair, Orders: []OrderParameters{limitOrder, {OrderType: "limit", Volume: 1}}})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "AddOrderBatch must reject an invalid order")
	assert.ErrorContains(t, err, "order 1", "AddOrderBatch should name the invalid order")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name     string
		req      *AddOrderBatchRequest
		expBody  string
		response string
		exp      *AddOrderBatchResponse
	}{
		{
			name: "placed",
			req: &AddOrderBatchRequest{
				Pair: spotTestPair,
				Orders: []OrderParameters{
					{
						UserReference:       1680953421,
						OrderType:           "limit",
						Side:                "buy",
						Volume:              0.803,
						Price:               OrderPrice{Value: 28300},
						TimeInForce:         "GTC",
						SelfTradePrevention: "cancel-oldest",
						Close:               &ConditionalCloseOrder{OrderType: "stop-loss-limit", Price: OrderPrice{Value: 27000}, SecondaryPrice: OrderPrice{Value: 26000}},
					},
					{
						ClientOrderID: "da8e4ad59b78481c93e589746b0cf91f",
						OrderType:     "limit",
						Side:          "sell",
						Volume:        0.105,
						Price:         OrderPrice{Value: 36000},
						PostOnly:      true,
						TimeInForce:   "GTD",
						StartTime:     time.Unix(1791590400, 0),
						ExpireDelay:   time.Hour,
					},
					{
						OrderType:      "stop-loss-limit",
						Side:           "sell",
						Volume:         2,
						Price:          OrderPrice{Value: 1, Offset: "-", Percent: true},
						SecondaryPrice: OrderPrice{Value: 79000},
						TriggerSignal:  "index",
						TimeInForce:    "IOC",
					},
				},
				Deadline:    tradingTestDeadline,
				BrokerIIBAN: "AA12 N84G WQRX 4ASV",
			},
			expBody: `{
				"orders": [
					{
						"userref": 1680953421,
						"ordertype": "limit",
						"type": "buy",
						"volume": "0.803",
						"price": "28300",
						"stptype": "cancel-oldest",
						"timeinforce": "GTC",
						"close": {"ordertype": "stop-loss-limit", "price": "27000", "price2": "26000"}
					},
					{
						"cl_ord_id": "da8e4ad59b78481c93e589746b0cf91f",
						"ordertype": "limit",
						"type": "sell",
						"volume": "0.105",
						"price": "36000",
						"oflags": "post",
						"timeinforce": "GTD",
						"starttm": "1791590400",
						"expiretm": "+3600"
					},
					{
						"ordertype": "stop-loss-limit",
						"type": "sell",
						"volume": "2",
						"price": "-1%",
						"price2": "79000",
						"trigger": "index",
						"timeinforce": "IOC"
					}
				],
				"pair": "XBTUSD",
				"deadline": "2026-10-09T00:10:30Z",
				"broker": "AA12 N84G WQRX 4ASV"
			}`,
			response: `{"error":[],"result":{"orders":[
				{"txid":"O5OR23-ADFAD-Y2G61C","descr":{"order":"buy 0.80300000 XBTUSD @ limit 28300.0","close":"close position @ stop loss 27000.0 -> limit 26000.0"}},
				{"txid":"OK8HFF-5J2PL-XLR17S","descr":{"order":"sell 0.10500000 XBTUSD @ limit 36000.0"}},
				{"error":"EOrder:Insufficient funds"}
			]}}`,
			exp: &AddOrderBatchResponse{Orders: []BatchOrderResult{
				{
					TransactionID: "O5OR23-ADFAD-Y2G61C",
					Description: AddOrderDescription{
						Order: "buy 0.80300000 XBTUSD @ limit 28300.0",
						Close: "close position @ stop loss 27000.0 -> limit 26000.0",
					},
				},
				{TransactionID: "OK8HFF-5J2PL-XLR17S", Description: AddOrderDescription{Order: "sell 0.10500000 XBTUSD @ limit 36000.0"}},
				{Error: "EOrder:Insufficient funds"},
			}},
		},
		{
			name: "validated",
			req: &AddOrderBatchRequest{
				Pair:       spotTestPair,
				AssetClass: "tokenized_asset",
				Orders: []OrderParameters{
					{OrderType: "market", Side: "buy", Volume: 0, Leverage: 2, ReduceOnly: true},
					{OrderType: "take-profit", Side: "sell", Volume: 0.25, Price: OrderPrice{Value: 90000}, FeeInBase: true, StartDelay: time.Minute},
					{OrderType: "iceberg", Side: "sell", Volume: 3, DisplayVolume: 0.2, Price: OrderPrice{Value: 95000}},
				},
				Validate: true,
			},
			expBody: `{
				"orders": [
					{"ordertype": "market", "type": "buy", "volume": "0", "leverage": "2", "reduce_only": true},
					{"ordertype": "take-profit", "type": "sell", "volume": "0.25", "price": "90000", "oflags": "fcib", "starttm": "+60"},
					{"ordertype": "iceberg", "type": "sell", "volume": "3", "displayvol": "0.2", "price": "95000"}
				],
				"pair": "XBTUSD",
				"asset_class": "tokenized_asset",
				"validate": true
			}`,
			response: `{"error":[],"result":{"orders":[
				{"descr":{"order":"buy 0.00000000 XBTUSD @ market with 2:1 leverage"}},
				{"descr":{"order":"sell 0.25000000 XBTUSD @ take profit 90000.0"}},
				{"descr":{"order":"sell 3.00000000 XBTUSD @ iceberg 95000.0"}}
			]}}`,
			exp: &AddOrderBatchResponse{Orders: []BatchOrderResult{
				{Description: AddOrderDescription{Order: "buy 0.00000000 XBTUSD @ market with 2:1 leverage"}},
				{Description: AddOrderDescription{Order: "sell 0.25000000 XBTUSD @ take profit 90000.0"}},
				{Description: AddOrderDescription{Order: "sell 3.00000000 XBTUSD @ iceberg 95000.0"}},
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := e
			if mockTests {
				// The VCR server cannot match a body nesting arrays and objects, so the batch is checked and served here
				ex = newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/0/private/AddOrderBatch", r.URL.Path, "AddOrderBatch should call its endpoint")
					assert.JSONEq(t, tc.expBody, tradingRequestBody(t, r), "AddOrderBatch should send each parameter as documented")
					_, _ = w.Write([]byte(tc.response))
				})
			}
			result, err := ex.AddOrderBatch(t.Context(), tc.req)
			require.NoError(t, err, "AddOrderBatch must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "AddOrderBatch should decode every field")
				return
			}
			assert.Len(t, result.Orders, len(tc.req.Orders), "AddOrderBatch should return a result for each order")
		})
	}
}

func TestCancelOrderBatch(t *testing.T) {
	t.Parallel()
	_, err := e.CancelOrderBatch(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelOrderBatch must reject a nil request")
	_, err = e.CancelOrderBatch(t.Context(), &CancelOrderBatchRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelOrderBatch must reject a request without an order identifier")
	_, err = e.CancelOrderBatch(t.Context(), &CancelOrderBatchRequest{TransactionIDs: make([]string, 25), UserReferences: make([]int32, 25), ClientOrderIDs: []string{"arb-20240509-00010"}})
	require.ErrorIs(t, err, errInvalidCount, "CancelOrderBatch must reject more than 50 identifiers")
	_, err = e.CancelOrderBatch(t.Context(), &CancelOrderBatchRequest{TransactionIDs: []string{""}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelOrderBatch must reject an empty transaction ID")
	_, err = e.CancelOrderBatch(t.Context(), &CancelOrderBatchRequest{UserReferences: []int32{0}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelOrderBatch must reject a zero user reference")
	_, err = e.CancelOrderBatch(t.Context(), &CancelOrderBatchRequest{ClientOrderIDs: []string{""}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelOrderBatch must reject an empty client order ID")

	ex := e
	if mockTests {
		// The batch is checked here, where the body's JSON types are compared as well as its values
		ex = newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/0/private/CancelOrderBatch", r.URL.Path, "CancelOrderBatch should call its endpoint")
			assert.JSONEq(t, `{
				"orders": ["OP5V2Y-RYKVL-ET3V3B", "OP5V2Y-7YKVL-ET3V3B", 1680953421],
				"cl_ord_ids": ["6d1b345e-2821-40e2-ad83-4ecb18a06876"]
			}`, tradingRequestBody(t, r), "CancelOrderBatch should send each identifier bare, as Kraken's example does")
			_, _ = w.Write([]byte(`{"error":[],"result":{"count":4}}`))
		})
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := ex.CancelOrderBatch(t.Context(), &CancelOrderBatchRequest{
		TransactionIDs: []string{"OP5V2Y-RYKVL-ET3V3B", "OP5V2Y-7YKVL-ET3V3B"},
		UserReferences: []int32{1680953421},
		ClientOrderIDs: []string{"6d1b345e-2821-40e2-ad83-4ecb18a06876"},
	})
	require.NoError(t, err, "CancelOrderBatch must not error")
	if mockTests {
		assert.Equal(t, &CancelOrderBatchResponse{Count: 4}, result, "CancelOrderBatch should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelOrderBatch should return a result")
}

func TestGetWebsocketToken(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWebsocketToken(t.Context())
	require.NoError(t, err, "GetWebsocketToken must not error")
	if mockTests {
		exp := &WebsocketTokenResponse{Token: "1Dwc4lzSwNWOAwkMdqhssNNFhs1ed606d1WcF3XfEMw", ExpiresInSeconds: 900}
		assert.Equal(t, exp, result, "GetWebsocketToken should decode every field")
		return
	}
	assert.NotEmpty(t, result.Token, "GetWebsocketToken should return a token")
}

func TestOrderPriceString(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		price OrderPrice
		exp   string
	}{
		{price: OrderPrice{Value: 40000.5}, exp: "40000.5"},
		{price: OrderPrice{Offset: "+"}, exp: "+0"},
		{price: OrderPrice{Value: 5, Offset: "+", Percent: true}, exp: "+5%"},
		{price: OrderPrice{Value: 100, Offset: "-"}, exp: "-100"},
		{price: OrderPrice{Value: 2.5, Offset: "#", Percent: true}, exp: "#2.5%"},
	} {
		assert.Equalf(t, tc.exp, tc.price.String(), "String should format %+v as Kraken takes it", tc.price)
	}
}

func TestOrderPriceUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		data string
		exp  OrderPrice
	}{
		{data: `"19500.0"`, exp: OrderPrice{Value: 19500}},
		{data: `81720.1`, exp: OrderPrice{Value: 81720.1}},
		{data: `"+5.0000%"`, exp: OrderPrice{Value: 5, Offset: "+", Percent: true}},
		{data: `"-100"`, exp: OrderPrice{Value: 100, Offset: "-"}},
		{data: `"#2.5%"`, exp: OrderPrice{Value: 2.5, Offset: "#", Percent: true}},
		{data: `"+0"`, exp: OrderPrice{Offset: "+"}},
		{data: `""`},
		{data: `null`},
	} {
		price := OrderPrice{Value: 1, Offset: "#"}
		require.NoErrorf(t, price.UnmarshalJSON([]byte(tc.data)), "UnmarshalJSON must not error for %s", tc.data)
		assert.Equalf(t, tc.exp, price, "UnmarshalJSON should decode %s", tc.data)
	}
	for _, data := range []string{`"market"`, `"+%"`, `"5%%"`, `true`} {
		var price OrderPrice
		assert.ErrorIsf(t, price.UnmarshalJSON([]byte(data)), errInvalidOrderPrice, "UnmarshalJSON should reject %s", data)
	}
	var price OrderPrice
	assert.Error(t, price.UnmarshalJSON([]byte(`"19500.0`)), "UnmarshalJSON should reject an unterminated string")
}

func TestCheckOrderPrices(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                  string
		orderType             string
		price, secondaryPrice OrderPrice
		err                   error
	}{
		{name: "absolute price", orderType: "limit", price: OrderPrice{Value: 19500}},
		{name: "relative prices", orderType: "stop-loss-limit", price: OrderPrice{Value: 5, Offset: "+", Percent: true}, secondaryPrice: OrderPrice{Value: 100, Offset: "#"}},
		{name: "negative price", orderType: "limit", price: OrderPrice{Value: -1}, err: errInvalidOrderPrice},
		{name: "negative secondary price", orderType: "stop-loss-limit", price: OrderPrice{Value: 19500}, secondaryPrice: OrderPrice{Value: -1}, err: errInvalidOrderPrice},
		{name: "unknown offset", orderType: "limit", price: OrderPrice{Value: 1, Offset: "*"}, err: errInvalidOrderPrice},
		{name: "offset of two characters", orderType: "limit", price: OrderPrice{Value: 1, Offset: "+-"}, err: errInvalidOrderPrice},
		{name: "relative trailing stop", orderType: "trailing-stop", price: OrderPrice{Value: 50, Offset: "+"}},
		{name: "absolute trailing stop", orderType: "trailing-stop", price: OrderPrice{Value: 19500}, err: errInvalidOrderPrice},
		{name: "trailing stop limit", orderType: "trailing-stop-limit", price: OrderPrice{Value: 50, Offset: "+"}, secondaryPrice: OrderPrice{Value: 10, Offset: "-"}},
		{name: "trailing stop limit with an absolute limit price", orderType: "trailing-stop-limit", price: OrderPrice{Value: 50, Offset: "+"}, secondaryPrice: OrderPrice{Value: 19500}, err: errInvalidOrderPrice},
		{name: "trailing stop limit with a last price limit offset", orderType: "trailing-stop-limit", price: OrderPrice{Value: 50, Offset: "+"}, secondaryPrice: OrderPrice{Value: 10, Offset: "#"}, err: errInvalidOrderPrice},
		{name: "trailing stop limit with an absolute trigger price", orderType: "trailing-stop-limit", price: OrderPrice{Value: 19500}, secondaryPrice: OrderPrice{Value: 10, Offset: "-"}, err: errInvalidOrderPrice},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, checkOrderPrices(tc.orderType, tc.price, tc.secondaryPrice), tc.err, "checkOrderPrices should return the expected error")
		})
	}
}

func TestCheckOrderPrice(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		price   OrderPrice
		offsets string
		err     error
	}{
		{name: "absolute price", price: OrderPrice{Value: 19500}, offsets: "+-"},
		{name: "zero price", offsets: "+-"},
		{name: "offset taken", price: OrderPrice{Value: 5, Offset: "-", Percent: true}, offsets: "+-"},
		{name: "last price offset taken", price: OrderPrice{Value: 5, Offset: "#"}, offsets: "+-#"},
		{name: "last price offset not taken", price: OrderPrice{Value: 5, Offset: "#"}, offsets: "+-", err: errInvalidOrderPrice},
		{name: "offset of two characters", price: OrderPrice{Value: 5, Offset: "+-"}, offsets: "+-", err: errInvalidOrderPrice},
		{name: "negative price", price: OrderPrice{Value: -5}, offsets: "+-", err: errInvalidOrderPrice},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, checkOrderPrice(tc.price, tc.offsets), tc.err, "checkOrderPrice should return the expected error")
		})
	}
}

func TestOrderParametersParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		change func(*OrderParameters)
		err    error
	}{
		{name: "side", change: func(o *OrderParameters) { o.Side = "hold" }, err: order.ErrSideIsInvalid},
		{name: "order type", change: func(o *OrderParameters) { o.OrderType = "moon" }, err: order.ErrTypeIsInvalid},
		{name: "negative volume", change: func(o *OrderParameters) { o.Volume = -1 }, err: order.ErrAmountIsInvalid},
		{name: "negative display volume", change: func(o *OrderParameters) { o.DisplayVolume = -1 }, err: order.ErrAmountIsInvalid},
		{name: "zero volume", change: func(o *OrderParameters) { o.Volume = 0 }, err: order.ErrAmountIsInvalid},
		{name: "user reference and client order ID", change: func(o *OrderParameters) { o.UserReference, o.ClientOrderID = 1, "a" }, err: errMultipleOrderIdentifiers},
		{name: "fee in base and quote currency", change: func(o *OrderParameters) { o.FeeInBase, o.FeeInQuote = true, true }, err: errConflictingOrderFlags},
		{name: "time in force", change: func(o *OrderParameters) { o.TimeInForce = "DAY" }, err: order.ErrInvalidTimeInForce},
		{name: "GTD without an expiry", change: func(o *OrderParameters) { o.TimeInForce = timeInForceGTD }, err: order.ErrInvalidTimeInForce},
		{name: "negative price", change: func(o *OrderParameters) { o.Price = OrderPrice{Value: -1} }, err: errInvalidOrderPrice},
		{name: "start time and delay", change: func(o *OrderParameters) { o.StartTime, o.StartDelay = time.Unix(1791504000, 0), time.Minute }, err: errInvalidOrderTime},
		{name: "short expiry delay", change: func(o *OrderParameters) { o.ExpireDelay = 4 * time.Second }, err: errInvalidOrderTime},
		{name: "conditional close order type", change: func(o *OrderParameters) { o.Close = &ConditionalCloseOrder{OrderType: "market"} }, err: order.ErrTypeIsInvalid},
	} {
		o := &OrderParameters{OrderType: "limit", Side: "buy", Volume: 1, Price: OrderPrice{Value: 19500}}
		tc.change(o)
		_, _, err := o.params()
		assert.ErrorIsf(t, err, tc.err, "params should reject an order by its %s", tc.name)
	}

	body, closeOrder, err := (&OrderParameters{OrderType: "market", Side: "sell", Leverage: 3}).params()
	require.NoError(t, err, "params must not error for a margin order without a volume")
	assert.Equal(t, map[string]any{"ordertype": "market", "type": "sell", "volume": "0", "leverage": "3"}, body, "params should send a margin order without a volume")
	assert.Nil(t, closeOrder, "params should return no conditional close order without one")

	o := &OrderParameters{
		ClientOrderID:       "6d1b345e-2821-40e2-ad83-4ecb18a06876",
		OrderType:           "stop-loss-limit",
		Side:                "buy",
		Volume:              1.25,
		DisplayVolume:       0.5,
		Price:               OrderPrice{Value: 5, Offset: "+", Percent: true},
		SecondaryPrice:      OrderPrice{Value: 100, Offset: "#"},
		TriggerSignal:       "index",
		Leverage:            2,
		ReduceOnly:          true,
		SelfTradePrevention: "cancel-newest",
		PostOnly:            true,
		FeeInQuote:          true,
		VolumeInQuote:       true,
		TimeInForce:         timeInForceGTD,
		StartTime:           time.Unix(1791504000, 0),
		ExpireDelay:         30 * time.Second,
		Close:               &ConditionalCloseOrder{OrderType: "limit", Price: OrderPrice{Value: 90000}},
	}
	body, closeOrder, err = o.params()
	require.NoError(t, err, "params must not error")
	exp := map[string]any{
		"ordertype":   "stop-loss-limit",
		"type":        "buy",
		"volume":      "1.25",
		"cl_ord_id":   "6d1b345e-2821-40e2-ad83-4ecb18a06876",
		"displayvol":  "0.5",
		"price":       "+5%",
		"price2":      "#100",
		"trigger":     "index",
		"leverage":    "2",
		"reduce_only": true,
		"stptype":     "cancel-newest",
		"oflags":      "post,fciq,viqc",
		"timeinforce": timeInForceGTD,
		"starttm":     "1791504000",
		"expiretm":    "+30",
	}
	assert.Equal(t, exp, body, "params should send every field set")
	assert.Equal(t, map[string]any{"ordertype": "limit", "price": "90000"}, closeOrder, "params should return the conditional close order apart")
}

func TestConditionalCloseOrderParams(t *testing.T) {
	t.Parallel()
	_, err := (&ConditionalCloseOrder{OrderType: "market"}).params()
	assert.ErrorIs(t, err, order.ErrTypeIsInvalid, "params should reject an order type a conditional close order cannot take")
	_, err = (&ConditionalCloseOrder{OrderType: "trailing-stop", Price: OrderPrice{Value: 19000}}).params()
	assert.ErrorIs(t, err, errInvalidOrderPrice, "params should reject a trailing stop at an absolute price")
	params, err := (&ConditionalCloseOrder{OrderType: "limit"}).params()
	require.NoError(t, err, "params must not error without prices")
	assert.Equal(t, map[string]any{"ordertype": "limit"}, params, "params should send only the order type without prices")
	params, err = (&ConditionalCloseOrder{OrderType: "stop-loss-limit", Price: OrderPrice{Value: 19000}, SecondaryPrice: OrderPrice{Value: 18900}}).params()
	require.NoError(t, err, "params must not error")
	assert.Equal(t, map[string]any{"ordertype": "stop-loss-limit", "price": "19000", "price2": "18900"}, params, "params should send both prices")
}

func TestFormatOrderTime(t *testing.T) {
	t.Parallel()
	at := time.Unix(1791504000, 0)
	for _, tc := range []struct {
		name           string
		at             time.Time
		delay, minimum time.Duration
		exp            string
		err            error
	}{
		{name: "neither a time nor a delay", minimum: time.Second},
		{name: "a time", at: at, minimum: time.Second, exp: "1791504000"},
		{name: "a time and a delay", at: at, delay: time.Minute, minimum: time.Second, err: errInvalidOrderTime},
		{name: "a negative delay", delay: -time.Second, minimum: time.Second, err: errInvalidOrderTime},
		{name: "a delay under the minimum in whole seconds", delay: 4900 * time.Millisecond, minimum: 5 * time.Second, err: errInvalidOrderTime},
		{name: "a fractional delay", delay: 5900 * time.Millisecond, minimum: 5 * time.Second, exp: "+5"},
		{name: "a delay", delay: 90 * time.Second, minimum: time.Second, exp: "+90"},
	} {
		got, err := formatOrderTime(tc.at, tc.delay, tc.minimum)
		require.ErrorIsf(t, err, tc.err, "formatOrderTime must return the expected error for %s", tc.name)
		assert.Equalf(t, tc.exp, got, "formatOrderTime should format %s", tc.name)
	}
}

func TestOrderPriceAbsolute(t *testing.T) {
	t.Parallel()
	assert.InDelta(t, 19500.0, OrderPrice{Value: 19500}.absolute(), 0, "absolute should return an absolute price")
	assert.Zero(t, OrderPrice{Value: 50, Offset: "+"}.absolute(), "absolute should return zero for an offset")
	assert.Zero(t, OrderPrice{Value: 2.5, Percent: true}.absolute(), "absolute should return zero for a percentage")
}
