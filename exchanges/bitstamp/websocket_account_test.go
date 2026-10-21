package bitstamp

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestWebsocketOrderUpdate(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	testexch.FixtureToDataHandler(t, "testdata/wsMyOrders.json", ex.wsHandleData)
	payloads := dataHandlerPayloads(t, ex)
	require.Len(t, payloads, 8, "the data handler must receive an update for each order event")

	// The fixture was captured before order subtypes were published, so every order is decoded as a limit order
	partialMarketBuy, partialMarketSell := 0.00038666, 0.00038678
	for i, exp := range []struct {
		orderID, clientOrderID                    string
		status                                    order.Status
		side                                      order.Side
		price, amount, remainingAmount, execution float64
		date                                      time.Time
	}{
		{"1658864794234880", "test_market_buy", order.New, order.Buy, 999999999, 0, 0, 0, time.UnixMicro(1693831262313000)},
		{"1658864794234880", "test_market_buy", order.PartiallyFilled, order.Buy, 25862, 0.00038667, 0.00000001, partialMarketBuy, time.UnixMicro(1693831262313000)},
		{"1658864794234880", "test_market_buy", order.PartiallyCancelled, order.Buy, 25862, 0.00038667, 0.00000001, partialMarketBuy, time.UnixMicro(1693831262313000)},
		{"1658870500933632", "test_market_sell", order.New, order.Sell, 0, 0, 0, 0, time.UnixMicro(1693832655550000)},
		{"1658870500933632", "test_market_sell", order.PartiallyFilled, order.Sell, 25854, 0.00038679, 0.00000001, partialMarketSell, time.UnixMicro(1693832655550000)},
		{"1658870500933632", "test_market_sell", order.PartiallyCancelled, order.Sell, 25854, 0.00038679, 0.00000001, partialMarketSell, time.UnixMicro(1693832655550000)},
		{"1658869033291777", "test_limit_sell", order.New, order.Sell, 25845, 0.00038692, 0.00038692, 0, time.UnixMicro(1693832297239000)},
		{"1658869033291777", "test_limit_sell", order.Filled, order.Sell, 25845, 0.00038692, 0, 0.00038692, time.UnixMicro(1693832302664000)},
	} {
		d, ok := payloads[i].(*order.Detail)
		require.Truef(t, ok, "payload %d must be an order detail", i)
		assert.Equalf(t, &order.Detail{
			Exchange:        ex.Name,
			OrderID:         exp.orderID,
			ClientOrderID:   exp.clientOrderID,
			Type:            order.Limit,
			TimeInForce:     order.GoodTillCancel,
			Side:            exp.side,
			Status:          exp.status,
			AssetType:       asset.Spot,
			Pair:            wsSpotPair,
			Date:            exp.date,
			LastUpdated:     exp.date,
			AccountID:       "3141592",
			Price:           exp.price,
			Amount:          exp.amount,
			RemainingAmount: exp.remainingAmount,
			ExecutedAmount:  exp.execution,
		}, d, "order event %d should be correct", i)
	}
}

func TestProcessOrderEvent(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	err := ex.processOrderEvent(t.Context(), orderEventCreated, &OrderEventData{RemainingAmount: 1}, wsPerpetualPair, asset.PerpetualContract)
	assert.ErrorIs(t, err, errOrderIDMissing, "processOrderEvent should error without an order ID")
	err = ex.handleWSAccountOrder(t.Context(), nil, orderEventCreated, []byte(`{"id":"wibble"}`), wsPerpetualPair, asset.PerpetualContract)
	assert.Error(t, err, "handleWSAccountOrder should error on malformed data")

	for _, tc := range []struct {
		event, data string
	}{
		{event: orderEventChanged, data: `{"id":1500000001,"id_str":"1500000001","client_order_id":"my-order-001","amount":0.5,"amount_str":"0.50000000","amount_traded":"0","amount_at_create":"0.50000000","price":61200.00,"price_str":"61200.00","order_type":0,"order_subtype":0,"datetime":"1712131200","microtimestamp":"1712131200000000","trade_account_id":0,"is_liquidation":false}`},
		{event: orderEventReplaced, data: `{"id":1500000002,"id_str":"1500000002","orig_order_id":1500000001,"client_order_id":"my-order-002","amount":0.3,"amount_str":"0.30000000","amount_traded":"0.00000000","amount_at_create":"0.30000000","price":61500.00,"price_str":"61500.00","order_type":0,"order_subtype":8,"datetime":"1712131201","microtimestamp":"1712131201000000","trade_account_id":0,"is_liquidation":false}`},
		{event: orderEventStopActive, data: `{"id":1500000003,"id_str":"1500000003","amount":0.01,"amount_str":"0.01000","amount_traded":"0","amount_at_create":"0.01000","price":83000,"price_str":"83000","order_type":1,"order_subtype":22,"datetime":"1712131202","microtimestamp":"1712131202000000","trade_account_id":0,"is_liquidation":false,"reduce_only":true,"stop_price":"83500"}`},
		{event: orderEventStopInactive, data: `{"id":1500000004,"id_str":"1500000004","amount":0.01,"amount_str":"0.01000","amount_traded":"0","amount_at_create":"0.01000","price":0,"price_str":"0","order_type":1,"order_subtype":20,"datetime":"1712131203","microtimestamp":"1712131203000000","trade_account_id":0,"is_liquidation":false,"stop_price":"82000"}`},
		{event: orderEventDeleted, data: `{"client_order_id":"my-order-005","amount_str":"0.40000000","amount_at_create":"1.00000000","price_str":"61000","order_type":0,"microtimestamp":"1712131204000000","trade_account_id":42}`},
		{event: orderEventDeleted, data: `{"id":1500000006,"amount_str":"1.00000000","amount_at_create":"1.00000000","price_str":"61000","order_type":0,"microtimestamp":"1712131205000000"}`},
		{event: orderEventDeleted, data: `{"id":1500000007,"amount_str":"0","amount_at_create":"1.00000000","price_str":"61000","order_type":0,"microtimestamp":"1712131206000000"}`},
		{event: orderEventStopActive, data: `{"id":1500000009,"amount_str":"0.01000","amount_at_create":"0.01000","price_str":"0","order_type":1,"order_subtype":24,"microtimestamp":"1712131207000000","activation_price":"50000","trailing_delta":"50"}`},
		{event: "order_unknown", data: `{"id":1500000008}`},
	} {
		require.NoErrorf(t, ex.handleWSAccountOrder(t.Context(), nil, tc.event, []byte(tc.data), wsPerpetualPair, asset.PerpetualContract), "handleWSAccountOrder must not error for %s", tc.event)
	}
	assert.Equal(t, []any{
		&order.Detail{
			Exchange:        ex.Name,
			OrderID:         "1500000001",
			ClientOrderID:   "my-order-001",
			Type:            order.Limit,
			TimeInForce:     order.GoodTillCancel,
			Side:            order.Buy,
			Status:          order.Open,
			AssetType:       asset.PerpetualContract,
			Pair:            wsPerpetualPair,
			Date:            time.UnixMicro(1712131200000000),
			LastUpdated:     time.UnixMicro(1712131200000000),
			Price:           61200,
			Amount:          0.5,
			RemainingAmount: 0.5,
		},
		[]order.Detail{
			{
				Exchange:        ex.Name,
				OrderID:         "1500000002",
				ClientOrderID:   "my-order-002",
				Type:            order.Limit,
				TimeInForce:     order.GoodTillTime,
				Side:            order.Buy,
				Status:          order.New,
				AssetType:       asset.PerpetualContract,
				Pair:            wsPerpetualPair,
				Date:            time.UnixMicro(1712131201000000),
				LastUpdated:     time.UnixMicro(1712131201000000),
				Price:           61500,
				Amount:          0.3,
				RemainingAmount: 0.3,
			},
			{
				Exchange:    ex.Name,
				OrderID:     "1500000001",
				Status:      order.Cancelled,
				AssetType:   asset.PerpetualContract,
				Pair:        wsPerpetualPair,
				LastUpdated: time.UnixMicro(1712131201000000),
			},
		},
		&order.Detail{
			Exchange:        ex.Name,
			OrderID:         "1500000003",
			Type:            order.StopLimit,
			Side:            order.Sell,
			Status:          order.Active,
			AssetType:       asset.PerpetualContract,
			Pair:            wsPerpetualPair,
			Date:            time.UnixMicro(1712131202000000),
			LastUpdated:     time.UnixMicro(1712131202000000),
			Price:           83000,
			Amount:          0.01,
			RemainingAmount: 0.01,
			ReduceOnly:      true,
			TriggerPrice:    83500,
		},
		&order.Detail{
			Exchange:        ex.Name,
			OrderID:         "1500000004",
			Type:            order.StopMarket,
			Side:            order.Sell,
			Status:          order.Cancelled,
			AssetType:       asset.PerpetualContract,
			Pair:            wsPerpetualPair,
			Date:            time.UnixMicro(1712131203000000),
			LastUpdated:     time.UnixMicro(1712131203000000),
			Amount:          0.01,
			RemainingAmount: 0.01,
			TriggerPrice:    82000,
		},
		&order.Detail{
			Exchange:        ex.Name,
			ClientOrderID:   "my-order-005",
			AccountID:       "42",
			Type:            order.Limit,
			TimeInForce:     order.GoodTillCancel,
			Side:            order.Buy,
			Status:          order.PartiallyCancelled,
			AssetType:       asset.PerpetualContract,
			Pair:            wsPerpetualPair,
			Date:            time.UnixMicro(1712131204000000),
			LastUpdated:     time.UnixMicro(1712131204000000),
			Price:           61000,
			Amount:          1,
			RemainingAmount: 0.4,
			ExecutedAmount:  0.6,
		},
		&order.Detail{
			Exchange:        ex.Name,
			OrderID:         "1500000006",
			Type:            order.Limit,
			TimeInForce:     order.GoodTillCancel,
			Side:            order.Buy,
			Status:          order.Cancelled,
			AssetType:       asset.PerpetualContract,
			Pair:            wsPerpetualPair,
			Date:            time.UnixMicro(1712131205000000),
			LastUpdated:     time.UnixMicro(1712131205000000),
			Price:           61000,
			Amount:          1,
			RemainingAmount: 1,
		},
		&order.Detail{
			Exchange:       ex.Name,
			OrderID:        "1500000007",
			Type:           order.Limit,
			TimeInForce:    order.GoodTillCancel,
			Side:           order.Buy,
			Status:         order.Filled,
			AssetType:      asset.PerpetualContract,
			Pair:           wsPerpetualPair,
			Date:           time.UnixMicro(1712131206000000),
			LastUpdated:    time.UnixMicro(1712131206000000),
			Price:          61000,
			Amount:         1,
			ExecutedAmount: 1,
		},
		&order.Detail{
			Exchange:        ex.Name,
			OrderID:         "1500000009",
			Type:            order.TrailingStop,
			Side:            order.Sell,
			Status:          order.Active,
			AssetType:       asset.PerpetualContract,
			Pair:            wsPerpetualPair,
			Date:            time.UnixMicro(1712131207000000),
			LastUpdated:     time.UnixMicro(1712131207000000),
			Amount:          0.01,
			RemainingAmount: 0.01,
			TriggerPrice:    50000,
		},
		websocket.UnhandledMessageWarning{Message: fmt.Sprintf("%s%sorder_unknown %+v", ex.Name, websocket.UnhandledMessage, &OrderEventData{ID: 1500000008})},
	}, dataHandlerPayloads(t, ex), "order events should be sent to the data handler")

	// A replacement and the order it replaced take one payload, so the data handler accepts or refuses them together
	ex.Websocket.DataHandler = stream.NewRelay(1)
	require.NoError(t, ex.processOrderEvent(t.Context(), orderEventReplaced, &OrderEventData{ID: 2, OriginalOrderID: 1}, wsPerpetualPair, asset.PerpetualContract), "processOrderEvent must relay a replacement to a data handler with room for one payload")
	require.Len(t, ex.Websocket.DataHandler.C, 1, "the data handler must receive one payload")
	replacement, ok := (<-ex.Websocket.DataHandler.C).Data.([]order.Detail)
	require.True(t, ok, "a replacement must be relayed as order details")
	assert.Len(t, replacement, 2, "the payload should hold the replacement and the order it replaced")
}

// orderEventsKey is the market and order source of the events orderEventFrame returns
var orderEventsKey = wsSpotPair.String() + "|" + asset.Spot.String() + "|orderbook"

// orderEventFrame returns a private orders channel event for an order, chained to the event before it
func orderEventFrame(eventID, preEventID string, orderID uint64) []byte {
	pre := "null"
	if preEventID != "" {
		pre = `"` + preEventID + `"`
	}
	return fmt.Appendf(nil, `{"event":"order_changed","channel":"private-my_orders_btcusd-123456","event_id":%q,"pre_event_id":%s,"order_source":"orderbook","data":{"id":%d,"amount_str":"0.5","amount_at_create":"1","price_str":"61200","order_type":0,"microtimestamp":"1712131200000000"}}`, eventID, pre, orderID)
}

func TestHandleWebsocketAccountOrderGapRecovery(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var requests []url.Values
	failRecovery := true
	var totalsFetched []url.Values
	ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, r.ParseForm(), "ParseForm should not error")
		mu.Lock()
		defer mu.Unlock()
		if r.URL.Path == "/api/v2/order_status/" {
			// The recovered deletion of an order which executed fetches its totals, which this test leaves unanswered
			totalsFetched = append(totalsFetched, r.PostForm)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		assert.Equal(t, "/api/v2/account_order_data/", r.URL.Path, "path should be the account order data")
		if failRecovery {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		requests = append(requests, r.PostForm)
		// Events are returned from the since ID inclusive, at most 1000 at a time
		var events []string
		switch r.PostForm.Get("since_id") {
		case "event-2":
			for i := 2; i <= 1001; i++ {
				events = append(events, fmt.Sprintf(`{"event":"order_changed","event_id":"event-%d","order_source":"orderbook","data":{"id":%d,"amount_str":"0.5","amount_at_create":"1"}}`, i, i))
			}
		case "event-1001":
			events = append(events,
				`{"event":"order_changed","event_id":"event-1001","order_source":"orderbook","data":{"id":1001,"amount_str":"0.5","amount_at_create":"1"}}`,
				`{"event":"order_deleted","event_id":"event-1002","order_source":"orderbook","data":{"id":1002,"amount_str":"0","amount_at_create":"1"}}`)
		}
		_, err := w.Write([]byte("[" + strings.Join(events, ",") + "]"))
		assert.NoError(t, err, "Write should not error")
	})
	ex.Websocket.DataHandler = stream.NewRelay(2000)

	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 1)), "the first event must be processed")
	assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "the first event should be processed without recovery")
	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-2", "event-1", 2)), "a chained event must be processed")
	assert.Equal(t, []string{"2"}, relayedOrderIDs(t, ex), "a chained event should be processed without recovery")

	// A gap whose recovery fails holds the event back, and is retried from the same event with the next event
	err := ex.wsHandleData(t.Context(), orderEventFrame("event-1002", "event-1001", 1002))
	assert.ErrorIs(t, err, errOrderEventRecovery, "a failed recovery should be reported")
	assert.Empty(t, relayedOrderIDs(t, ex), "an event should not be relayed before the events missed ahead of it")

	mu.Lock()
	failRecovery = false
	mu.Unlock()
	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1003", "event-1002", 1003)), "an event after a gap must be processed")
	ids := relayedOrderIDs(t, ex)
	require.Len(t, ids, 1001, "every missed event must be processed once before the event")
	assert.Equal(t, "3", ids[0], "recovered events should start after the last processed event")
	assert.Equal(t, "1001", ids[998], "recovered events should be paged")
	assert.Equal(t, "1002", ids[999], "the event held back by the failed recovery should be recovered")
	assert.Equal(t, "1003", ids[1000], "the event should be processed after the recovered events")
	mu.Lock()
	assert.Equal(t, []url.Values{
		{"market": {"BTC/USD"}, "order_source": {"orderbook"}, "since_id": {"event-2"}, "until_id": {"event-1003"}},
		{"market": {"BTC/USD"}, "order_source": {"orderbook"}, "since_id": {"event-1001"}, "until_id": {"event-1003"}},
	}, requests, "recovery should resume from the last processed event and page from the last recovered event")
	mu.Unlock()

	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1004", "event-1003", 1004)), "a chained event must be processed")
	assert.Equal(t, []string{"1004"}, relayedOrderIDs(t, ex), "an event chained to a recovered gap should be processed without recovery")
	waitForOrderTotalFetches(t, ex)
	mu.Lock()
	assert.Equal(t, []url.Values{{"id": {"1002"}}}, totalsFetched, "the totals of the recovered order which executed should be fetched")
	mu.Unlock()
}

// relayedOrderIDs returns the IDs of the orders relayed to the data handler since it was last read
func relayedOrderIDs(t *testing.T, ex *Exchange) []string {
	t.Helper()
	var ids []string
	for {
		select {
		case p := <-ex.Websocket.DataHandler.C:
			d, ok := p.Data.(*order.Detail)
			require.Truef(t, ok, "the data handler must only receive order details, not %T %v", p.Data, p.Data)
			ids = append(ids, d.OrderID)
		default:
			return ids
		}
	}
}

// fullDataHandler returns a data handler whose buffer is full, so it refuses everything sent to it
func fullDataHandler(t *testing.T) *stream.Relay {
	t.Helper()
	r := stream.NewRelay(1)
	require.NoError(t, r.Send(t.Context(), "filler"), "Send must not error")
	return r
}

// newOrderEventsTestExchange returns an exchange whose account order data requests return events event-N for order N,
// from the since ID inclusive to the until ID exclusive and at most 1000 at a time as the API does, with no order ID
// for the events listed in withoutOrderID, and a function returning the since ID of every request
// A request fails when fail returns true for its since ID, and like the API, a range which is empty is rejected
func newOrderEventsTestExchange(t *testing.T, fail func(sinceID string) bool, withoutOrderID ...int) (ex *Exchange, sinceIDs func() []string) {
	t.Helper()
	var mu sync.Mutex
	var received []string
	ex = newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/account_order_data/", r.URL.Path, "path should be the account order data")
		assert.NoError(t, r.ParseForm(), "ParseForm should not error")
		sinceID := r.PostForm.Get("since_id")
		mu.Lock()
		received = append(received, sinceID)
		mu.Unlock()
		since, err := strconv.Atoi(strings.TrimPrefix(sinceID, "event-"))
		assert.NoError(t, err, "since_id should name an event")
		until, err := strconv.Atoi(strings.TrimPrefix(r.PostForm.Get("until_id"), "event-"))
		assert.NoError(t, err, "until_id should name an event")
		if until <= since {
			w.WriteHeader(http.StatusBadRequest)
			_, err = w.Write([]byte(`{"status":"error","reason":"Invalid event ID range: until_id must be greater than since_id.","response_code":"400.085"}`))
			assert.NoError(t, err, "Write should not error")
			return
		}
		if fail != nil && fail(sinceID) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var events []string
		for i := since; i < until && len(events) < maxResultLimit; i++ {
			orderID := i
			if slices.Contains(withoutOrderID, i) {
				orderID = 0
			}
			events = append(events, fmt.Sprintf(`{"event":"order_changed","event_id":"event-%d","order_source":"orderbook","data":{"id":%d,"amount_str":"0.5","amount_at_create":"1"}}`, i, orderID))
		}
		_, err = w.Write([]byte("[" + strings.Join(events, ",") + "]"))
		assert.NoError(t, err, "Write should not error")
	})
	ex.Websocket.DataHandler = stream.NewRelay(100)
	return ex, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(received)
	}
}

func TestHandleWebsocketAccountOrderRefusedEvents(t *testing.T) {
	t.Parallel()
	t.Run("later event", func(t *testing.T) {
		t.Parallel()
		ex, sinceIDs := newOrderEventsTestExchange(t, nil)
		require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 1)), "the first event must be processed")
		ex.Websocket.DataHandler = fullDataHandler(t)
		assert.Error(t, ex.wsHandleData(t.Context(), orderEventFrame("event-2", "event-1", 2)), "a refused event should error")
		ex.Websocket.DataHandler = stream.NewRelay(100)
		require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-3", "event-2", 3)), "the next event must be processed")
		assert.Equal(t, []string{"2", "3"}, relayedOrderIDs(t, ex), "the refused event should be recovered before the next event")
		assert.Equal(t, []string{"event-2"}, sinceIDs(), "recovery should start with the refused event")
	})
	for name, preEventID := range map[string]string{"first event": "event-0", "first event after a server restart": ""} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ex, sinceIDs := newOrderEventsTestExchange(t, nil)
			ex.Websocket.DataHandler = fullDataHandler(t)
			assert.Error(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", preEventID, 1)), "a refused event should error")
			ex.Websocket.DataHandler = stream.NewRelay(100)
			require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-2", "event-1", 2)), "the next event must be processed")
			assert.Equal(t, []string{"1", "2"}, relayedOrderIDs(t, ex), "the refused event should be recovered before the next event")
			assert.Equal(t, []string{"event-1"}, sinceIDs(), "recovery should start with the refused event")
		})
	}
	t.Run("event after recovery", func(t *testing.T) {
		t.Parallel()
		ex, sinceIDs := newOrderEventsTestExchange(t, nil)
		require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 1)), "the first event must be processed")
		assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "the first event should be relayed")
		// The data handler takes the two recovered events and refuses the event they were recovered for
		ex.Websocket.DataHandler = stream.NewRelay(2)
		assert.Error(t, ex.wsHandleData(t.Context(), orderEventFrame("event-4", "event-3", 4)), "a refused event should error")
		assert.Equal(t, []string{"2", "3"}, relayedOrderIDs(t, ex), "the missed events should be relayed")
		ex.Websocket.DataHandler = stream.NewRelay(100)
		require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-5", "event-4", 5)), "the next event must be processed")
		assert.Equal(t, []string{"4", "5"}, relayedOrderIDs(t, ex), "only the refused event should be recovered before the next event")
		assert.Equal(t, []string{"event-1", "event-4"}, sinceIDs(), "recovery should resume with the refused event")
	})
	t.Run("recovered event", func(t *testing.T) {
		t.Parallel()
		ex, sinceIDs := newOrderEventsTestExchange(t, nil)
		require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 1)), "the first event must be processed")
		assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "the first event should be relayed")
		// The data handler takes one recovered event and refuses the next
		ex.Websocket.DataHandler = stream.NewRelay(1)
		err := ex.wsHandleData(t.Context(), orderEventFrame("event-4", "event-3", 4))
		assert.ErrorIs(t, err, errOrderEventRecovery, "a refused recovered event should stop recovery")
		assert.Equal(t, []string{"2"}, relayedOrderIDs(t, ex), "the event should be held back with the events after the refused event")
		ex.Websocket.DataHandler = stream.NewRelay(100)
		require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-5", "event-4", 5)), "the next event must be processed")
		assert.Equal(t, []string{"3", "4", "5"}, relayedOrderIDs(t, ex), "recovery should resume with the refused event")
		assert.Equal(t, []string{"event-1", "event-3"}, sinceIDs(), "recovery should resume with the refused event")
	})
}

func TestHandleWebsocketAccountOrderWithoutOrderID(t *testing.T) {
	t.Parallel()
	ex, sinceIDs := newOrderEventsTestExchange(t, nil, 3)
	assert.ErrorIs(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 0)), errOrderIDMissing, "an event without an order ID should error")
	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-2", "event-1", 2)), "an event chained to an event without an order ID must be processed")
	assert.Empty(t, sinceIDs(), "an event without an order ID should not be recovered, as its recovered copy has no order ID either")
	assert.Equal(t, []string{"2"}, relayedOrderIDs(t, ex), "the chained event should be relayed")

	// Recovered events without an order ID are reported and skipped, so recovery does not stop at them
	err := ex.wsHandleData(t.Context(), orderEventFrame("event-5", "event-4", 5))
	assert.ErrorIs(t, err, errOrderIDMissing, "a recovered event without an order ID should be reported")
	assert.NotErrorIs(t, err, errOrderEventRecovery, "a recovered event without an order ID should not stop recovery")
	assert.Equal(t, []string{"4", "5"}, relayedOrderIDs(t, ex), "the events after an event without an order ID should be relayed")
	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-6", "event-5", 6)), "a chained event must be processed")
	assert.Equal(t, []string{"6"}, relayedOrderIDs(t, ex), "a chained event should be relayed without recovery")
	assert.Equal(t, []string{"event-2"}, sinceIDs(), "recovery should only be requested for the gap")
}

func TestHandleWebsocketAccountOrderRepeatedEvent(t *testing.T) {
	t.Parallel()
	t.Run("relayed event", func(t *testing.T) {
		t.Parallel()
		ex, sinceIDs := newOrderEventsTestExchange(t, nil)
		require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 1)), "the first event must be processed")
		require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 1)), "a repeated event must be processed")
		assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "a repeated event should not be relayed again")
		require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-2", "event-1", 2)), "the next event must be processed")
		assert.Equal(t, []string{"2"}, relayedOrderIDs(t, ex), "the next event should be relayed")
		assert.Empty(t, sinceIDs(), "a repeated event should not be recovered")
	})
	t.Run("refused event", func(t *testing.T) {
		t.Parallel()
		ex, sinceIDs := newOrderEventsTestExchange(t, nil)
		ex.Websocket.DataHandler = fullDataHandler(t)
		assert.Error(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 1)), "a refused event should error")
		ex.Websocket.DataHandler = stream.NewRelay(100)
		require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 1)), "a repeated refused event must be processed")
		assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "a repeated refused event should be relayed")
		require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-2", "event-1", 2)), "the next event must be processed")
		assert.Equal(t, []string{"2"}, relayedOrderIDs(t, ex), "the next event should be relayed")
		assert.Empty(t, sinceIDs(), "a repeated event should not be recovered")
	})
}

func TestHandleWebsocketAccountOrderRecoveryPageFailure(t *testing.T) {
	t.Parallel()
	var failed atomic.Bool
	ex, sinceIDs := newOrderEventsTestExchange(t, func(sinceID string) bool {
		return sinceID == "event-1000" && failed.CompareAndSwap(false, true)
	})
	ex.Websocket.DataHandler = stream.NewRelay(2000)
	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 1)), "the first event must be processed")
	assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "the first event should be relayed")

	err := ex.wsHandleData(t.Context(), orderEventFrame("event-1003", "event-1002", 1003))
	assert.ErrorIs(t, err, errOrderEventRecovery, "a page which cannot be fetched should be reported")
	ids := relayedOrderIDs(t, ex)
	require.Len(t, ids, 999, "the events of the first page must be relayed")
	assert.Equal(t, "2", ids[0], "recovered events should start after the last relayed event")
	assert.Equal(t, "1000", ids[998], "recovered events should end with the first page")

	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1004", "event-1003", 1004)), "the next event must be processed")
	assert.Equal(t, []string{"1001", "1002", "1003", "1004"}, relayedOrderIDs(t, ex), "recovery should resume after the first page without relaying it again")
	assert.Equal(t, []string{"event-1", "event-1000", "event-1000"}, sinceIDs(), "recovery should resume from the last relayed event")
}

func TestHandleWebsocketAccountOrderRecoveryGiveUp(t *testing.T) {
	t.Parallel()
	frame := func(id uint64) []byte {
		return orderEventFrame("event-"+strconv.FormatUint(id, 10), "event-"+strconv.FormatUint(id-1, 10), id)
	}
	t.Run("recovery keeps failing", func(t *testing.T) {
		t.Parallel()
		ex, sinceIDs := newOrderEventsTestExchange(t, func(string) bool { return true })
		require.NoError(t, ex.wsHandleData(t.Context(), frame(1)), "the first event must be processed")
		assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "the first event should be relayed")
		for _, id := range []uint64{4, 5} {
			err := ex.wsHandleData(t.Context(), frame(id))
			assert.ErrorIsf(t, err, errOrderEventRecovery, "the failed recovery before event %d should be reported", id)
			assert.NotErrorIsf(t, err, errOrderEventsSkipped, "the failed recovery before event %d should not skip the missed events", id)
			assert.Emptyf(t, relayedOrderIDs(t, ex), "event %d should be held back while recovery fails", id)
		}
		err := ex.wsHandleData(t.Context(), frame(6))
		assert.ErrorIs(t, err, errOrderEventRecovery, "the third failed recovery should be reported")
		assert.ErrorIs(t, err, errOrderEventsSkipped, "the third failed recovery in a row should skip the missed events")
		assert.Equal(t, []string{"4", "5", "6"}, relayedOrderIDs(t, ex), "the events held back should be relayed before the event once the missed events are skipped")
		require.NoError(t, ex.wsHandleData(t.Context(), frame(7)), "an event chained to the skipped events must be processed")
		assert.Equal(t, []string{"7"}, relayedOrderIDs(t, ex), "an event chained to the skipped events should be relayed without recovery")
		assert.Equal(t, []string{"event-1", "event-1", "event-1"}, sinceIDs(), "recovery should be retried from the last relayed event until the missed events are skipped")
	})
	t.Run("recovery succeeds between failures", func(t *testing.T) {
		t.Parallel()
		var failing atomic.Bool
		failing.Store(true)
		ex, _ := newOrderEventsTestExchange(t, func(string) bool { return failing.Load() })
		require.NoError(t, ex.wsHandleData(t.Context(), frame(1)), "the first event must be processed")
		for _, id := range []uint64{4, 5} {
			assert.ErrorIsf(t, ex.wsHandleData(t.Context(), frame(id)), errOrderEventRecovery, "the failed recovery before event %d should be reported", id)
		}
		failing.Store(false)
		require.NoError(t, ex.wsHandleData(t.Context(), frame(6)), "an event after a recovered gap must be processed")
		assert.Equal(t, []string{"1", "2", "3", "4", "5", "6"}, relayedOrderIDs(t, ex), "the missed events should be recovered before the event")
		failing.Store(true)
		err := ex.wsHandleData(t.Context(), frame(9))
		assert.ErrorIs(t, err, errOrderEventRecovery, "a failed recovery should be reported")
		assert.NotErrorIs(t, err, errOrderEventsSkipped, "a failed recovery after a successful one should not skip the missed events")
		assert.Empty(t, relayedOrderIDs(t, ex), "the event should be held back while recovery fails")
	})
	t.Run("no permission", func(t *testing.T) {
		t.Parallel()
		ex := newResponseTestExchange(t, http.StatusForbidden, `{"status":"error","reason":"No permission found","code":"API0003"}`)
		ex.Websocket.DataHandler = stream.NewRelay(100)
		require.NoError(t, ex.wsHandleData(t.Context(), frame(1)), "the first event must be processed")
		err := ex.wsHandleData(t.Context(), frame(4))
		assert.ErrorIs(t, err, errNoPermission, "the permission error should be reported")
		assert.ErrorIs(t, err, errOrderEventsSkipped, "a recovery the API key lacks permission for should skip the missed events at once")
		assert.Equal(t, []string{"1", "4"}, relayedOrderIDs(t, ex), "the event should be relayed without the missed events")
	})
}

func TestHandleWebsocketAccountOrderStaleEvent(t *testing.T) {
	t.Parallel()
	var failing atomic.Bool
	ex, sinceIDs := newOrderEventsTestExchange(t, func(string) bool { return failing.Load() })
	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 1)), "the first event must be processed")
	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-2", "event-1", 2)), "a chained event must be processed")
	assert.Equal(t, []string{"1", "2"}, relayedOrderIDs(t, ex), "the events should be relayed")
	for range maxOrderEventRecoveryFailures {
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "event-0", 1)), errStaleOrderEvent, "an event before the last one handled should be dropped")
	}
	assert.Empty(t, relayedOrderIDs(t, ex), "an event before the last one handled should not be relayed")

	// Dropped events are not failed recoveries, so a later gap is still held back until recovery has failed three times
	failing.Store(true)
	for _, id := range []uint64{5, 6} {
		err := ex.wsHandleData(t.Context(), orderEventFrame("event-"+strconv.FormatUint(id, 10), "event-"+strconv.FormatUint(id-1, 10), id))
		assert.ErrorIsf(t, err, errOrderEventRecovery, "the failed recovery before event %d should be reported", id)
		assert.NotErrorIsf(t, err, errOrderEventsSkipped, "the failed recovery before event %d should not skip the missed events", id)
	}
	assert.Empty(t, relayedOrderIDs(t, ex), "events should be held back while recovery fails")
	failing.Store(false)
	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-7", "event-6", 7)), "an event after a recovered gap must be processed")
	assert.Equal(t, []string{"3", "4", "5", "6", "7"}, relayedOrderIDs(t, ex), "the missed events should be recovered before the event")
	assert.Equal(t, []string{"event-2", "event-2", "event-2", "event-2", "event-2", "event-2"}, sinceIDs(), "recovery should be requested from the last relayed event")
}

func TestRecoverOrderEvents(t *testing.T) {
	t.Parallel()
	ex, sinceIDs := newOrderEventsTestExchange(t, nil, 3)
	checkpoint, err := ex.recoverOrderEvents(t.Context(), orderEventsKey, wsSpotPair, asset.Spot, "orderbook", orderEventCheckpoint{eventID: "event-1"}, "event-5")
	assert.ErrorIs(t, err, errOrderIDMissing, "recoverOrderEvents should report an event without an order ID")
	assert.NotErrorIs(t, err, errOrderEventRecovery, "recoverOrderEvents should not stop at an event without an order ID")
	assert.Equal(t, orderEventCheckpoint{eventID: "event-4"}, checkpoint, "recoverOrderEvents should return the last event handled")
	assert.Equal(t, []string{"2", "4"}, relayedOrderIDs(t, ex), "recoverOrderEvents should relay the events after the checkpoint")

	checkpoint, err = ex.recoverOrderEvents(t.Context(), orderEventsKey, wsSpotPair, asset.Spot, "orderbook", orderEventCheckpoint{eventID: "event-4", pending: true}, "event-6")
	require.NoError(t, err, "recoverOrderEvents must not error")
	assert.Equal(t, orderEventCheckpoint{eventID: "event-5"}, checkpoint, "recoverOrderEvents should return the last event handled")
	assert.Equal(t, []string{"4", "5"}, relayedOrderIDs(t, ex), "recoverOrderEvents should relay a pending checkpoint's event")

	ex.Websocket.DataHandler = stream.NewRelay(1)
	checkpoint, err = ex.recoverOrderEvents(t.Context(), orderEventsKey, wsSpotPair, asset.Spot, "orderbook", orderEventCheckpoint{eventID: "event-5"}, "event-9")
	assert.ErrorIs(t, err, errOrderEventRecovery, "recoverOrderEvents should error when the data handler refuses an event")
	assert.Equal(t, orderEventCheckpoint{eventID: "event-7", pending: true}, checkpoint, "recoverOrderEvents should return the refused event as pending")
	assert.Equal(t, []string{"6"}, relayedOrderIDs(t, ex), "recoverOrderEvents should stop at the refused event")
	assert.Equal(t, []string{"event-1", "event-4", "event-5"}, sinceIDs(), "recoverOrderEvents should request events from the checkpoint")

	ex = newTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	checkpoint, err = ex.recoverOrderEvents(t.Context(), orderEventsKey, wsSpotPair, asset.Spot, "orderbook", orderEventCheckpoint{eventID: "event-1"}, "event-3")
	assert.ErrorIs(t, err, errOrderEventRecovery, "recoverOrderEvents should error when the events cannot be fetched")
	assert.Equal(t, orderEventCheckpoint{eventID: "event-1"}, checkpoint, "recoverOrderEvents should keep the checkpoint when the events cannot be fetched")
}

func TestSetOrderEventCheckpoint(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	ex.setOrderEventCheckpoint("BTC/USD|spot|orderbook", orderEventCheckpoint{eventID: "event-1", pending: true})
	ex.setOrderEventCheckpoint("BTC/USD|spot|stop_order", orderEventCheckpoint{eventID: "event-2"})
	ex.setOrderEventCheckpoint("BTC/USD|spot|orderbook", orderEventCheckpoint{eventID: "event-3"})
	assert.Equal(t, map[string]orderEventCheckpoint{
		"BTC/USD|spot|orderbook":  {eventID: "event-3"},
		"BTC/USD|spot|stop_order": {eventID: "event-2"},
	}, ex.orderEventCheckpoints, "setOrderEventCheckpoint should store the last checkpoint of each key")
}

// heartbeatReply is the server's reply to the heartbeats the ping handler sends
var heartbeatReply = []byte(`{"event":"bts:heartbeat","channel":"","data":{"status":"success"}}`)

func TestHandleWebsocketAccountOrderKeptEvents(t *testing.T) {
	t.Parallel()
	frame := func(id uint64) []byte {
		return orderEventFrame("event-"+strconv.FormatUint(id, 10), "event-"+strconv.FormatUint(id-1, 10), id)
	}
	t.Run("refused event", func(t *testing.T) {
		t.Parallel()
		ex, _ := newOrderEventsTestExchange(t, func(string) bool { return true })
		ex.Websocket.DataHandler = fullDataHandler(t)
		assert.Error(t, ex.wsHandleData(t.Context(), frame(1)), "a refused event should error")
		ex.Websocket.DataHandler = stream.NewRelay(100)
		for _, id := range []uint64{2, 3} {
			assert.ErrorIsf(t, ex.wsHandleData(t.Context(), frame(id)), errOrderEventRecovery, "the failed recovery before event %d should be reported", id)
		}
		assert.Empty(t, relayedOrderIDs(t, ex), "events should be held back while recovery fails")
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), frame(4)), errOrderEventsSkipped, "the third failed recovery in a row should skip the missed events")
		assert.Equal(t, []string{"1", "2", "3", "4"}, relayedOrderIDs(t, ex), "the refused event and the events held back should be relayed before the event")
	})
	t.Run("refused recovered event", func(t *testing.T) {
		t.Parallel()
		var failing atomic.Bool
		ex, _ := newOrderEventsTestExchange(t, func(string) bool { return failing.Load() })
		require.NoError(t, ex.wsHandleData(t.Context(), frame(1)), "the first event must be processed")
		assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "the first event should be relayed")
		failing.Store(true)
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), frame(4)), errOrderEventRecovery, "a failed recovery should be reported")
		failing.Store(false)
		// The data handler takes one recovered event and refuses the next
		ex.Websocket.DataHandler = stream.NewRelay(1)
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), frame(5)), errOrderEventRecovery, "a refused recovered event should stop recovery")
		assert.Equal(t, []string{"2"}, relayedOrderIDs(t, ex), "the recovered events before the refused event should be relayed")
		ex.Websocket.DataHandler = stream.NewRelay(100)
		failing.Store(true)
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), frame(6)), errOrderEventsSkipped, "the third failed recovery in a row should skip the missed events")
		assert.Equal(t, []string{"3", "4", "5", "6"}, relayedOrderIDs(t, ex), "the refused recovered event should be relayed ahead of the events held back behind it")
	})
	t.Run("events fetched after a refused event", func(t *testing.T) {
		t.Parallel()
		var failing atomic.Bool
		ex, _ := newOrderEventsTestExchange(t, func(string) bool { return failing.Load() })
		require.NoError(t, ex.wsHandleData(t.Context(), frame(1)), "the first event must be processed")
		assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "the first event should be relayed")
		// Event 4 is only ever fetched, after the refused event 3
		ex.Websocket.DataHandler = stream.NewRelay(1)
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), frame(5)), errOrderEventRecovery, "a refused recovered event should stop recovery")
		assert.Equal(t, []string{"2"}, relayedOrderIDs(t, ex), "the recovered events before the refused event should be relayed")
		ex.Websocket.DataHandler = stream.NewRelay(100)
		failing.Store(true)
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), frame(6)), errOrderEventRecovery, "a failed recovery should be reported")
		assert.Empty(t, relayedOrderIDs(t, ex), "events should be held back while recovery fails")
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), frame(7)), errOrderEventsSkipped, "the third failed recovery in a row should skip the missed events")
		assert.Equal(t, []string{"3", "4", "5", "6", "7"}, relayedOrderIDs(t, ex), "the refused event and the events fetched after it should be relayed before the events held back")
		require.NoError(t, ex.wsHandleData(t.Context(), frame(8)), "an event chained to the relayed events must be processed")
		assert.Equal(t, []string{"8"}, relayedOrderIDs(t, ex), "an event chained to the relayed events should be relayed without recovery")
	})
	t.Run("live copy of a fetched event", func(t *testing.T) {
		t.Parallel()
		var failing atomic.Bool
		ex, _ := newOrderEventsTestExchange(t, func(string) bool { return failing.Load() })
		require.NoError(t, ex.wsHandleData(t.Context(), frame(1)), "the first event must be processed")
		assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "the first event should be relayed")
		ex.Websocket.DataHandler = stream.NewRelay(1)
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), frame(5)), errOrderEventRecovery, "a refused recovered event should stop recovery")
		assert.Equal(t, []string{"2"}, relayedOrderIDs(t, ex), "the recovered events before the refused event should be relayed")
		ex.Websocket.DataHandler = stream.NewRelay(100)
		failing.Store(true)
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), orderEventFrame("event-4", "event-3", 4)), errOrderEventRecovery, "a failed recovery should be reported")
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), frame(6)), errOrderEventsSkipped, "the third failed recovery in a row should skip the missed events")
		assert.Equal(t, []string{"3", "4", "5", "6"}, relayedOrderIDs(t, ex), "an event received both fetched and live should be relayed once")
	})
	t.Run("refused while giving up", func(t *testing.T) {
		t.Parallel()
		ex, _ := newOrderEventsTestExchange(t, func(string) bool { return true })
		require.NoError(t, ex.wsHandleData(t.Context(), frame(1)), "the first event must be processed")
		assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "the first event should be relayed")
		for _, id := range []uint64{4, 5} {
			assert.ErrorIsf(t, ex.wsHandleData(t.Context(), frame(id)), errOrderEventRecovery, "the failed recovery before event %d should be reported", id)
		}
		// The data handler takes the first event held back and refuses the next
		ex.Websocket.DataHandler = stream.NewRelay(1)
		err := ex.wsHandleData(t.Context(), frame(6))
		assert.ErrorIs(t, err, errOrderEventsSkipped, "the third failed recovery in a row should skip the missed events")
		assert.ErrorIs(t, err, errOrderEventRecovery, "the refused event should be reported")
		assert.Equal(t, []string{"4"}, relayedOrderIDs(t, ex), "the events held back should be relayed until the data handler refuses one")
		ex.Websocket.DataHandler = stream.NewRelay(100)
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), frame(7)), errOrderEventsSkipped, "the next failed recovery should give up at once")
		assert.Equal(t, []string{"5", "6", "7"}, relayedOrderIDs(t, ex), "the rest of the events held back should be relayed before the next event")
		require.NoError(t, ex.wsHandleData(t.Context(), frame(8)), "an event chained to the relayed events must be processed")
		assert.Equal(t, []string{"8"}, relayedOrderIDs(t, ex), "an event chained to the relayed events should be relayed without recovery")
	})
	t.Run("repeated event refused while giving up", func(t *testing.T) {
		t.Parallel()
		ex, _ := newOrderEventsTestExchange(t, func(string) bool { return true })
		require.NoError(t, ex.wsHandleData(t.Context(), frame(1)), "the first event must be processed")
		assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "the first event should be relayed")
		for _, id := range []uint64{4, 5} {
			assert.ErrorIsf(t, ex.wsHandleData(t.Context(), frame(id)), errOrderEventRecovery, "the failed recovery before event %d should be reported", id)
		}
		// Event 4 arrives again as recovery gives up, and the data handler takes it but refuses event 5
		ex.Websocket.DataHandler = stream.NewRelay(1)
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), frame(4)), errOrderEventsSkipped, "the third failed recovery in a row should skip the missed events")
		assert.Equal(t, []string{"4"}, relayedOrderIDs(t, ex), "the repeated event should be relayed among the events held back")
		ex.Websocket.DataHandler = stream.NewRelay(100)
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), frame(6)), errOrderEventsSkipped, "the next failed recovery should give up at once")
		assert.Equal(t, []string{"5", "6"}, relayedOrderIDs(t, ex), "the repeated event should not be relayed again")
	})
	t.Run("events recovered while kept", func(t *testing.T) {
		t.Parallel()
		var failing atomic.Bool
		failing.Store(true)
		ex, _ := newOrderEventsTestExchange(t, func(string) bool { return failing.Load() })
		require.NoError(t, ex.wsHandleData(t.Context(), frame(1)), "the first event must be processed")
		for _, id := range []uint64{4, 5} {
			assert.ErrorIsf(t, ex.wsHandleData(t.Context(), frame(id)), errOrderEventRecovery, "the failed recovery before event %d should be reported", id)
		}
		failing.Store(false)
		require.NoError(t, ex.wsHandleData(t.Context(), frame(6)), "an event after a recovered gap must be processed")
		assert.Equal(t, []string{"1", "2", "3", "4", "5", "6"}, relayedOrderIDs(t, ex), "the missed events and the events held back should be recovered before the event")
		failing.Store(true)
		for _, id := range []uint64{9, 10} {
			assert.ErrorIsf(t, ex.wsHandleData(t.Context(), frame(id)), errOrderEventRecovery, "the failed recovery before event %d should be reported", id)
		}
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), frame(11)), errOrderEventsSkipped, "the third failed recovery in a row should skip the missed events")
		assert.Equal(t, []string{"9", "10", "11"}, relayedOrderIDs(t, ex), "the events a recovery relayed should not be relayed again when recovery gives up")
	})
	t.Run("kept event a recovery does not return", func(t *testing.T) {
		t.Parallel()
		var failing atomic.Bool
		ex := newTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			if failing.Load() {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// The events after the since ID are not available yet, as when the API's event log lags the websocket
			_, err := w.Write([]byte(`[]`))
			assert.NoError(t, err, "Write should not error")
		})
		ex.Websocket.DataHandler = stream.NewRelay(100)
		require.NoError(t, ex.wsHandleData(t.Context(), frame(1)), "the first event must be processed")
		assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "the first event should be relayed")
		failing.Store(true)
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), frame(3)), errOrderEventRecovery, "a failed recovery should be reported")
		failing.Store(false)
		require.NoError(t, ex.wsHandleData(t.Context(), frame(4)), "an event after a recovery must be processed")
		assert.Equal(t, []string{"3", "4"}, relayedOrderIDs(t, ex), "an event kept should be relayed before the event when the recovery does not return it")
		require.NoError(t, ex.wsHandleData(t.Context(), heartbeatReply), "a heartbeat reply with nothing kept must be processed")
		assert.Empty(t, relayedOrderIDs(t, ex), "a heartbeat reply should relay nothing once the kept event was relayed")
	})
	t.Run("kept event refused after a recovery", func(t *testing.T) {
		t.Parallel()
		var failing atomic.Bool
		ex := newTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			if failing.Load() {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// The events after the since ID are not available yet, as when the API's event log lags the websocket
			_, err := w.Write([]byte(`[]`))
			assert.NoError(t, err, "Write should not error")
		})
		ex.Websocket.DataHandler = stream.NewRelay(100)
		require.NoError(t, ex.wsHandleData(t.Context(), frame(1)), "the first event must be processed")
		assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "the first event should be relayed")
		failing.Store(true)
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), frame(3)), errOrderEventRecovery, "a failed recovery should be reported")
		failing.Store(false)
		ex.Websocket.DataHandler = fullDataHandler(t)
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), frame(4)), errOrderEventRecovery, "a refused event should be reported")
		ex.Websocket.DataHandler = stream.NewRelay(100)
		failing.Store(true)
		// The recovery succeeded, so the failures before it no longer count towards giving up
		for _, id := range []uint64{5, 6} {
			err := ex.wsHandleData(t.Context(), frame(id))
			assert.ErrorIsf(t, err, errOrderEventRecovery, "the failed recovery before event %d should be reported", id)
			assert.NotErrorIsf(t, err, errOrderEventsSkipped, "the failed recovery before event %d should not skip the missed events", id)
		}
		assert.Empty(t, relayedOrderIDs(t, ex), "events should be held back while recovery fails")
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), frame(7)), errOrderEventsSkipped, "the third failed recovery in a row should skip the missed events")
		assert.Equal(t, []string{"3", "4", "5", "6", "7"}, relayedOrderIDs(t, ex), "the events kept should be relayed in event order when recovery gives up")
	})
	t.Run("stale event on a heartbeat", func(t *testing.T) {
		t.Parallel()
		var reply atomic.Int32
		var requests atomic.Int32
		ex := newTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			requests.Add(1)
			if reply.Load() == http.StatusBadRequest {
				w.WriteHeader(http.StatusBadRequest)
				_, err := w.Write([]byte(`{"status":"error","reason":"Invalid event ID range: until_id must be greater than since_id.","response_code":"400.085"}`))
				assert.NoError(t, err, "Write should not error")
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
		})
		ex.Websocket.DataHandler = stream.NewRelay(100)
		require.NoError(t, ex.wsHandleData(t.Context(), frame(1)), "the first event must be processed")
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), frame(3)), errOrderEventRecovery, "a failed recovery should be reported")
		// The API now refuses the range, as it does for an event which is not after the checkpoint
		reply.Store(http.StatusBadRequest)
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), heartbeatReply), errStaleOrderEvent, "a heartbeat retry the API refuses the range of should drop the event")
		require.NoError(t, ex.wsHandleData(t.Context(), heartbeatReply), "a heartbeat reply after the event was dropped must be processed")
		assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "a dropped event should not be relayed")
		assert.Equal(t, int32(2), requests.Load(), "a dropped event should not be retried")
	})
	t.Run("heartbeat retry", func(t *testing.T) {
		t.Parallel()
		var failing atomic.Bool
		ex, sinceIDs := newOrderEventsTestExchange(t, func(string) bool { return failing.Load() })
		require.NoError(t, ex.wsHandleData(t.Context(), frame(1)), "the first event must be processed")
		assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "the first event should be relayed")
		failing.Store(true)
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), frame(3)), errOrderEventRecovery, "a failed recovery should be reported")
		failing.Store(false)
		require.NoError(t, ex.wsHandleData(t.Context(), heartbeatReply), "a heartbeat reply must retry the recovery")
		assert.Equal(t, []string{"2", "3"}, relayedOrderIDs(t, ex), "a heartbeat reply should recover the missed events and relay the event held back")
		require.NoError(t, ex.wsHandleData(t.Context(), heartbeatReply), "a heartbeat reply with nothing held back must be processed")
		require.NoError(t, ex.wsHandleData(t.Context(), frame(4)), "an event chained to the relayed events must be processed")
		assert.Equal(t, []string{"4"}, relayedOrderIDs(t, ex), "an event chained to the relayed events should be relayed without recovery")
		assert.Equal(t, []string{"event-1", "event-1"}, sinceIDs(), "the heartbeat reply should retry the recovery once from the last relayed event")
	})
	t.Run("failed heartbeat retries", func(t *testing.T) {
		t.Parallel()
		ex, sinceIDs := newOrderEventsTestExchange(t, func(string) bool { return true })
		require.NoError(t, ex.wsHandleData(t.Context(), frame(1)), "the first event must be processed")
		assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "the first event should be relayed")
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), frame(3)), errOrderEventRecovery, "a failed recovery should be reported")
		err := ex.wsHandleData(t.Context(), heartbeatReply)
		assert.ErrorIs(t, err, errOrderEventRecovery, "a failed heartbeat retry should be reported")
		assert.NotErrorIs(t, err, errOrderEventsSkipped, "the second failed recovery should not skip the missed events")
		assert.Empty(t, relayedOrderIDs(t, ex), "the event should be held back while recovery fails")
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), heartbeatReply), errOrderEventsSkipped, "a failed heartbeat retry should count towards giving up")
		assert.Equal(t, []string{"3"}, relayedOrderIDs(t, ex), "the event held back should be relayed when recovery gives up")
		require.NoError(t, ex.wsHandleData(t.Context(), heartbeatReply), "a heartbeat reply with nothing held back must be processed")
		require.NoError(t, ex.wsHandleData(t.Context(), frame(4)), "an event chained to the relayed event must be processed")
		assert.Equal(t, []string{"4"}, relayedOrderIDs(t, ex), "an event chained to the relayed event should be relayed without recovery")
		assert.Equal(t, []string{"event-1", "event-1", "event-1"}, sinceIDs(), "recovery should be retried once per heartbeat reply until it gives up")
	})
	t.Run("refused event on a heartbeat", func(t *testing.T) {
		t.Parallel()
		ex, sinceIDs := newOrderEventsTestExchange(t, nil)
		require.NoError(t, ex.wsHandleData(t.Context(), frame(1)), "the first event must be processed")
		assert.Equal(t, []string{"1"}, relayedOrderIDs(t, ex), "the first event should be relayed")
		ex.Websocket.DataHandler = fullDataHandler(t)
		assert.Error(t, ex.wsHandleData(t.Context(), frame(2)), "a refused event should error")
		ex.Websocket.DataHandler = stream.NewRelay(100)
		require.NoError(t, ex.wsHandleData(t.Context(), heartbeatReply), "a heartbeat reply must relay a refused event")
		assert.Equal(t, []string{"2"}, relayedOrderIDs(t, ex), "a heartbeat reply should relay the refused event")
		assert.Empty(t, sinceIDs(), "a refused event should be relayed from its kept copy")
	})
}

func TestKeptOrderEvents(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	keptIDs := func() []string {
		ex.orderEventMu.Lock()
		defer ex.orderEventMu.Unlock()
		ids := make([]string, len(ex.orderEventsKept[orderEventsKey]))
		for i := range ex.orderEventsKept[orderEventsKey] {
			ids[i] = ex.orderEventsKept[orderEventsKey][i].eventID
		}
		return ids
	}
	ex.keepOrderEvent(orderEventsKey, &keptOrderEvent{eventID: "event-4"})
	ex.keepOrderEvent(orderEventsKey, &keptOrderEvent{eventID: "event-5"})
	ex.keepOrderEvent(orderEventsKey, &keptOrderEvent{eventID: "event-4"})
	assert.Equal(t, []string{"event-4", "event-5"}, keptIDs(), "keepOrderEvent should keep live events once each, in the order they arrive")
	ex.keepRecoveredOrderEvents(orderEventsKey, []keptOrderEvent{{eventID: "event-2"}, {eventID: "event-3"}, {eventID: "event-4"}})
	assert.Equal(t, []string{"event-2", "event-3", "event-4", "event-5"}, keptIDs(), "keepRecoveredOrderEvents should keep recovered events ahead of the events kept, once each")
	ex.removeKeptOrderEvent(orderEventsKey, "event-3")
	ex.removeKeptOrderEvent(orderEventsKey, "event-9")
	assert.Equal(t, []string{"event-2", "event-4", "event-5"}, keptIDs(), "removeKeptOrderEvent should remove only a kept event")
	for i := range maxKeptOrderEvents {
		ex.keepOrderEvent(orderEventsKey, &keptOrderEvent{eventID: "event-" + strconv.Itoa(10+i)})
	}
	ids := keptIDs()
	require.Len(t, ids, maxKeptOrderEvents, "the kept events must be bounded")
	assert.Equal(t, "event-10", ids[0], "the oldest kept events should be dropped beyond the bound")
	assert.Equal(t, "event-"+strconv.Itoa(9+maxKeptOrderEvents), ids[len(ids)-1], "the newest event should be kept")
}

func TestProcessOrderEventAmounts(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	require.NoError(t, ex.processOrderEvent(t.Context(), orderEventChanged, &OrderEventData{ID: 1, AmountAtCreate: 1, RemainingAmount: 0.7}, wsSpotPair, asset.Spot), "processOrderEvent must not error")
	err := ex.processOrderEvent(t.Context(), orderEventChanged, &OrderEventData{ID: 2, AmountAtCreate: types.Number(math.NaN()), RemainingAmount: 0.7}, wsSpotPair, asset.Spot)
	assert.ErrorIs(t, err, errMalformedOrderEvent, "processOrderEvent should error on an amount which is not a number")
	assert.False(t, isOrderEventRetryable(err), "an event with a malformed amount should not be retried, as its recovered copy has the same amount")
	payloads := dataHandlerPayloads(t, ex)
	require.Len(t, payloads, 1, "the data handler must only receive the event with valid amounts")
	d, ok := payloads[0].(*order.Detail)
	require.True(t, ok, "the payload must be an order detail")
	// A float subtraction leaves 0.30000000000000004
	assert.Equal(t, 0.3, d.ExecutedAmount, "processOrderEvent should subtract the amounts without a float residue")
}

func TestFetchOrderTotals(t *testing.T) {
	t.Parallel()
	const deleted = `{"id":1001,"amount_str":%q,"amount_at_create":"1","price_str":"100","order_type":0,"order_subtype":0,"microtimestamp":"1712131206000000"}`
	const status = `{"id":1001,"datetime":"2024-04-03 08:00:00","type":"0","status":%q,"market":"BTC/USD","transactions":[%s],"amount_remaining":%q}`
	transaction := func(id int, amount, fee string) string {
		return fmt.Sprintf(`{"tid":%d,"price":"100","btc":%q,"usd":"-1","fee":%q,"datetime":"2024-04-03 08:00:0%d","type":2}`, id, amount, fee, id)
	}
	deletedAt, created := time.UnixMicro(1712131206000000), time.Date(2024, 4, 3, 8, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name      string
		event     string
		data      string
		reply     string
		replyCode int
		requests  int64
		totals    *order.Detail
	}{
		{
			name: "filled", event: orderEventDeleted, data: fmt.Sprintf(deleted, "0"),
			reply:    fmt.Sprintf(status, "Finished", transaction(1, "0.4", "0.5")+","+transaction(2, "0.6", "0.7"), "0"),
			requests: 1,
			totals:   &order.Detail{Status: order.Filled, ExecutedAmount: 1, RemainingAmount: 0, Fee: 1.2, Cost: 100, AverageExecutedPrice: 100},
		},
		{
			name: "partly cancelled", event: orderEventDeleted, data: fmt.Sprintf(deleted, "0.7"),
			reply:    fmt.Sprintf(status, "Canceled", transaction(1, "0.3", "0.1"), "0.7"),
			requests: 1,
			totals:   &order.Detail{Status: order.PartiallyCancelled, ExecutedAmount: 0.3, RemainingAmount: 0.7, Fee: 0.1, Cost: 30, AverageExecutedPrice: 100},
		},
		{
			name: "expired", event: orderEventDeleted, data: fmt.Sprintf(deleted, "0.5"),
			reply:    fmt.Sprintf(status, "Expired", transaction(1, "0.5", "0.2"), "0.5"),
			requests: 1,
			totals:   &order.Detail{Status: order.Expired, ExecutedAmount: 0.5, RemainingAmount: 0.5, Fee: 0.2, Cost: 50, AverageExecutedPrice: 100},
		},
		{name: "still open", event: orderEventDeleted, data: fmt.Sprintf(deleted, "0"), reply: fmt.Sprintf(status, "Open", transaction(1, "0.4", "0.5"), "0.6"), requests: 1},
		{name: "still open with the event's executions", event: orderEventDeleted, data: fmt.Sprintf(deleted, "0.6"), reply: fmt.Sprintf(status, "Open", transaction(1, "0.4", "0.5"), "0.6"), requests: 1},
		{name: "less executed", event: orderEventDeleted, data: fmt.Sprintf(deleted, "0"), reply: fmt.Sprintf(status, "Finished", transaction(1, "0.4", "0.5"), "0"), requests: 1},
		{name: "status refused", event: orderEventDeleted, data: fmt.Sprintf(deleted, "0"), replyCode: http.StatusInternalServerError, requests: 1},
		{name: "nothing executed", event: orderEventDeleted, data: fmt.Sprintf(deleted, "1")},
		{name: "client order ID only", event: orderEventDeleted, data: `{"client_order_id":"my-order","amount_str":"0","amount_at_create":"1","price_str":"100","order_type":0,"microtimestamp":"1712131206000000"}`},
		{name: "replaced, read refused", event: orderEventReplaced, data: `{"id":1002,"orig_order_id":1001,"amount_str":"0.5","amount_at_create":"0.5","price_str":"100","order_type":0,"microtimestamp":"1712131206000000"}`, replyCode: http.StatusInternalServerError, requests: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int64
			ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				assert.Equal(t, "/api/v2/order_status/", r.URL.Path, "path should be the order status")
				assert.NoError(t, r.ParseForm(), "ParseForm should not error")
				assert.Equal(t, url.Values{"id": {"1001"}}, r.PostForm, "the closed order's status should be requested")
				if tc.replyCode != 0 {
					w.WriteHeader(tc.replyCode)
					return
				}
				_, err := w.Write([]byte(tc.reply))
				assert.NoError(t, err, "Write should not error")
			})
			ex.Websocket.DataHandler = stream.NewRelay(10)
			var o OrderEventData
			require.NoError(t, json.Unmarshal([]byte(tc.data), &o), "Unmarshal must not error")
			require.NoError(t, ex.processOrderEvent(t.Context(), tc.event, &o, wsSpotPair, asset.Spot), "processOrderEvent must not error")
			payloads := dataHandlerPayloads(t, ex)
			assert.Equal(t, tc.requests, requests.Load(), "the totals should be fetched only for a replaced order or a deleted order which executed")
			if tc.totals == nil {
				assert.Len(t, payloads, 1, "the data handler should only receive the event")
				return
			}
			require.Len(t, payloads, 2, "the data handler must receive the event and the order's totals")
			event, ok := payloads[0].(*order.Detail)
			require.True(t, ok, "the event must be relayed as an order detail")
			exp := &order.Detail{
				Exchange:        ex.Name,
				OrderID:         "1001",
				Type:            order.Limit,
				TimeInForce:     order.GoodTillCancel,
				Side:            order.Buy,
				Status:          event.Status,
				AssetType:       asset.Spot,
				Pair:            wsSpotPair,
				Date:            deletedAt,
				LastUpdated:     deletedAt,
				Price:           100,
				Amount:          1,
				RemainingAmount: tc.totals.RemainingAmount,
				ExecutedAmount:  tc.totals.ExecutedAmount,
			}
			assert.Equal(t, exp, event, "the event should be relayed as it arrived")
			exp.Status = tc.totals.Status
			exp.ExecutedAmount = tc.totals.ExecutedAmount
			exp.Date = created
			exp.Fee = tc.totals.Fee
			exp.FeeAsset = currency.USD
			exp.Cost = tc.totals.Cost
			exp.CostAsset = currency.USD
			exp.AverageExecutedPrice = tc.totals.AverageExecutedPrice
			assert.Equal(t, exp, payloads[1], "the totals should be relayed with the event's detail, last update and no trades")
		})
	}
}

func TestFetchOrderTotalsOneAtATime(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var inFlight, peak int
	ex := newTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		inFlight++
		peak = max(peak, inFlight)
		mu.Unlock()
		defer func() {
			mu.Lock()
			inFlight--
			mu.Unlock()
		}()
		// The request layer only holds back another request for 50ms, so a fetch which is not waited for overlaps this
		time.Sleep(200 * time.Millisecond)
		_, err := w.Write([]byte(`{"id":1,"datetime":"2024-04-03 08:00:00","type":"0","status":"Finished","market":"BTC/USD","transactions":[{"tid":1,"price":"100","btc":"1","usd":"-100","fee":"0.1","datetime":"2024-04-03 08:00:01","type":2}],"amount_remaining":"0"}`))
		assert.NoError(t, err, "Write should not error")
	})
	ex.Websocket.DataHandler = stream.NewRelay(10)
	for id := uint64(1); id <= 2; id++ {
		require.NoErrorf(t, ex.processOrderEvent(t.Context(), orderEventDeleted, &OrderEventData{ID: id, AmountAtCreate: 1}, wsSpotPair, asset.Spot), "processOrderEvent must not error for order %d", id)
	}
	require.Len(t, dataHandlerPayloads(t, ex), 4, "the data handler must receive each event and its totals")
	// The fetches share the request rate limiter with order placements and cancellations, so a burst of them must not
	// queue ahead of those
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, peak, "the totals of deleted orders should be fetched one at a time")
}

func TestFetchReplacedOrderTotals(t *testing.T) {
	t.Parallel()
	const replacement = `{"id":1002,"orig_order_id":1001,"amount_str":"0.5","amount_at_create":"0.5","price_str":"100","order_type":0,"microtimestamp":"1712131206000000"}`
	const status = `{"id":1001,"datetime":"2024-04-03 08:00:00","type":"0","status":%q,"market":"BTC/USD","transactions":[%s],"amount_remaining":%q}`
	const transaction = `{"tid":1,"price":"100","btc":"0.3","usd":"-30","fee":"0.1","datetime":"2024-04-03 08:00:01","type":2}`
	replacedAt, created := time.UnixMicro(1712131206000000), time.Date(2024, 4, 3, 8, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name   string
		reply  string
		totals bool
	}{
		{name: "partly executed", reply: fmt.Sprintf(status, "Canceled", transaction, "0.7"), totals: true},
		{name: "nothing executed", reply: fmt.Sprintf(status, "Canceled", "", "1")},
		{name: "still open", reply: fmt.Sprintf(status, "Open", transaction, "0.7")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int64
			ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				assert.Equal(t, "/api/v2/order_status/", r.URL.Path, "path should be the order status")
				assert.NoError(t, r.ParseForm(), "ParseForm should not error")
				assert.Equal(t, url.Values{"id": {"1001"}}, r.PostForm, "the replaced order's status should be requested")
				_, err := w.Write([]byte(tc.reply))
				assert.NoError(t, err, "Write should not error")
			})
			ex.Websocket.DataHandler = stream.NewRelay(10)
			var o OrderEventData
			require.NoError(t, json.Unmarshal([]byte(replacement), &o), "Unmarshal must not error")
			require.NoError(t, ex.processOrderEvent(t.Context(), orderEventReplaced, &o, wsSpotPair, asset.Spot), "processOrderEvent must not error")
			payloads := dataHandlerPayloads(t, ex)
			assert.Equal(t, int64(1), requests.Load(), "the replaced order's totals should be fetched")
			if !tc.totals {
				assert.Len(t, payloads, 1, "the data handler should only receive the event")
				return
			}
			require.Len(t, payloads, 2, "the data handler must receive the event and the replaced order's totals")
			assert.Equal(t, &order.Detail{
				Exchange:             ex.Name,
				OrderID:              "1001",
				Status:               order.PartiallyCancelled,
				AssetType:            asset.Spot,
				Pair:                 wsSpotPair,
				Date:                 created,
				LastUpdated:          replacedAt,
				ExecutedAmount:       0.3,
				Fee:                  0.1,
				FeeAsset:             currency.USD,
				Cost:                 30,
				CostAsset:            currency.USD,
				AverageExecutedPrice: 100,
			}, payloads[1], "the replaced order's totals should be relayed with its status after the replacement")
		})
	}
}

func TestIsOrderEventRetryable(t *testing.T) {
	t.Parallel()
	assert.False(t, isOrderEventRetryable(nil), "a relayed event should not be retried")
	missingOrderID := fmt.Errorf("%w: order_changed", errOrderIDMissing)
	assert.False(t, isOrderEventRetryable(missingOrderID), "an event without an order ID should not be retried")
	malformedAmount := fmt.Errorf("%w: order_changed", errMalformedOrderEvent)
	assert.False(t, isOrderEventRetryable(malformedAmount), "an event with a malformed amount should not be retried")
	assert.True(t, isOrderEventRetryable(fullDataHandler(t).Send(t.Context(), nil)), "an event the data handler refused should be retried")
}

func TestHandleWebsocketAccountOrderRestart(t *testing.T) {
	t.Parallel()
	var recovered []url.Values
	var mu sync.Mutex
	ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, r.ParseForm(), "ParseForm should not error")
		mu.Lock()
		recovered = append(recovered, r.PostForm)
		mu.Unlock()
		_, err := w.Write([]byte(`[]`))
		assert.NoError(t, err, "Write should not error")
	})
	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-1", "", 1)), "an event after a restart must be processed")
	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-2", "event-1", 2)), "a chained event must be processed")
	mu.Lock()
	assert.Empty(t, recovered, "events should not be recovered without a gap or a known previous event")
	mu.Unlock()
	require.NoError(t, ex.wsHandleData(t.Context(), orderEventFrame("event-3", "", 3)), "an event after a server restart must be processed")
	mu.Lock()
	assert.Equal(t, []url.Values{{"market": {"BTC/USD"}, "order_source": {"orderbook"}, "since_id": {"event-2"}, "until_id": {"event-3"}}}, recovered, "a server restart should recover events from the last processed event")
	mu.Unlock()
}

func TestOrderTypeFromSubtypeID(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		subtype uint8
		typ     order.Type
		tif     order.TimeInForce
	}{
		{0, order.Limit, order.GoodTillCancel},
		{1, order.Market, order.UnknownTIF},
		{2, order.Market, order.UnknownTIF},
		{3, order.Limit, order.GoodTillDay},
		{4, order.Limit, order.ImmediateOrCancel},
		{5, order.Limit, order.PostOnly},
		{6, order.Limit, order.FillOrKill},
		{7, order.Market, order.UnknownTIF},
		{8, order.Limit, order.GoodTillTime},
		{20, order.StopMarket, order.UnknownTIF},
		{21, order.TakeProfitMarket, order.UnknownTIF},
		{22, order.StopLimit, order.UnknownTIF},
		{23, order.TakeProfit | order.Limit, order.UnknownTIF},
		{24, order.TrailingStop, order.UnknownTIF},
		{25, order.TrailingStop, order.UnknownTIF},
		{26, order.TrailingStopLimit, order.UnknownTIF},
		{27, order.TrailingStopLimit, order.UnknownTIF},
		{99, order.UnknownType, order.UnknownTIF},
	} {
		typ, tif := orderTypeFromSubtypeID(tc.subtype)
		assert.Equalf(t, tc.typ, typ, "orderTypeFromSubtypeID should return the order type for %d", tc.subtype)
		assert.Equalf(t, tc.tif, tif, "orderTypeFromSubtypeID should return the time in force for %d", tc.subtype)
	}
}

func TestHandleWebsocketMyTrade(t *testing.T) {
	t.Parallel()
	data := []byte(`{"id":296050733,"id_str":"296050733","trade_uti":"BSTP-20240403-296050733","order_id":1500000001,"client_order_id":"my-order-001","amount":0.1,"price":61200.00,"fee":"0.01","side":"buy","microtimestamp":"1712131200500000","trade_account_id":0,"position_id":null,"is_liquidation":null,"trade_type":"ORDERBOOK"}`)

	ex := newWebsocketTestInstance(t)
	require.NoError(t, ex.handleWSMyTrade(data, wsPerpetualPair, asset.PerpetualContract), "handleWSMyTrade must not error with the fills feed disabled")
	assert.Error(t, ex.handleWSMyTrade([]byte(`{"price":[]}`), wsPerpetualPair, asset.PerpetualContract), "handleWSMyTrade should error on malformed data with the fills feed disabled")
	assert.Empty(t, dataHandlerPayloads(t, ex), "executions should not be sent with the fills feed disabled")

	ex = newFeedTestInstance(t)
	assert.Error(t, ex.handleWSMyTrade([]byte(`{"price":[]}`), wsPerpetualPair, asset.PerpetualContract), "handleWSMyTrade should error on malformed data")
	err := ex.handleWSMyTrade([]byte(`{"id":1,"side":"wibble"}`), wsPerpetualPair, asset.PerpetualContract)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "handleWSMyTrade should error on an invalid side")
	require.NoError(t, ex.handleWSMyTrade(data, wsPerpetualPair, asset.PerpetualContract), "handleWSMyTrade must not error")
	require.NoError(t, ex.handleWSMyTrade([]byte(`{"client_order_id":"my-order-002","amount":0.2,"price":61000,"side":"sell","microtimestamp":"1712131201000000"}`), wsSpotPair, asset.Spot), "handleWSMyTrade must not error without IDs")
	// Executions are only relayed as fills, as one which reaches the order store before its order would be stored
	// without the order's date, status, type and amount
	assert.Equal(t, []any{
		[]fill.Data{{
			Timestamp:     time.UnixMicro(1712131200500000),
			Exchange:      ex.Name,
			AssetType:     asset.PerpetualContract,
			CurrencyPair:  wsPerpetualPair,
			Side:          order.Buy,
			OrderID:       "1500000001",
			ClientOrderID: "my-order-001",
			TradeID:       "296050733",
			Price:         61200,
			Amount:        0.1,
		}},
		[]fill.Data{{
			Timestamp:     time.UnixMicro(1712131201000000),
			Exchange:      ex.Name,
			AssetType:     asset.Spot,
			CurrencyPair:  wsSpotPair,
			Side:          order.Sell,
			ClientOrderID: "my-order-002",
			Price:         61000,
			Amount:        0.2,
		}},
	}, dataHandlerPayloads(t, ex), "executions should only be sent to the data handler as fills")
}

func TestHandleWebsocketSelfTrade(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	assert.Error(t, ex.handleWSSelfTrade(t.Context(), []byte(`{"amount_str":[]}`), wsSpotPair, asset.Spot), "handleWSSelfTrade should error on malformed data")
	data := []byte(`{"buy_order_id":1500000010,"sell_order_id":1500000011,"amount":0.05,"amount_str":"0.05000000","price":61200.00,"price_str":"61200.00","type":0,"timestamp":"1712131205","microtimestamp":"1712131205000000","sellers_trade_account_id":7,"buyers_trade_account_id":0}`)
	require.NoError(t, ex.handleWSSelfTrade(t.Context(), data, wsSpotPair, asset.Spot), "handleWSSelfTrade must not error")
	assert.Equal(t, []any{&WebsocketSelfTrade{
		Pair:                  wsSpotPair,
		AssetType:             asset.Spot,
		BuyOrderID:            1500000010,
		SellOrderID:           1500000011,
		Amount:                0.05,
		Price:                 61200,
		Side:                  OrderSide(order.Buy),
		Timestamp:             types.Time(time.Unix(1712131205, 0)),
		Microtimestamp:        types.Time(time.UnixMicro(1712131205000000)),
		SellersTradeAccountID: 7,
	}}, dataHandlerPayloads(t, ex), "self trades should be sent to the data handler")
}

func TestHandleWebsocketSettlement(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestInstance(t)
	assert.Error(t, ex.handleWSSettlement(t.Context(), "settlement", []byte(`{"as":[]}`)), "handleWSSettlement should error on malformed data")
	for event, data := range map[string]string{
		"settlement":             `{"id":"1234567890","tnx_id":"123123","ts":"1712131200","price":"61200","ccy":"USD","as":"-1.5","ap":"-1","atf":"-0.25","alf":"0","afu":"-0.25","asl":"0","sp":"61000","typ":"C"}`,
		"insurance_fund_premium": `{"tnx_id":"123124","ts":"1712131200","ccy":"USD","amount":"-0.5"}`,
	} {
		require.NoErrorf(t, ex.handleWSSettlement(t.Context(), event, []byte(data)), "handleWSSettlement must not error for %s", event)
	}
	assert.ElementsMatch(t, []any{
		&WebsocketSettlement{
			Event:                 "settlement",
			PositionID:            "1234567890",
			TransactionID:         "123123",
			Timestamp:             types.Time(time.Unix(1712131200, 0)),
			Price:                 61200,
			Currency:              currency.USD,
			AmountSettled:         -1.5,
			AmountFromPrice:       -1,
			AmountFromTradingFees: -0.25,
			AmountFromFunding:     -0.25,
			StrikePrice:           61000,
			SettlementType:        "C",
		},
		&WebsocketSettlement{
			Event:         "insurance_fund_premium",
			TransactionID: "123124",
			Timestamp:     types.Time(time.Unix(1712131200, 0)),
			Currency:      currency.USD,
			Amount:        -0.5,
		},
	}, dataHandlerPayloads(t, ex), "settlements should be sent to the data handler")
}
