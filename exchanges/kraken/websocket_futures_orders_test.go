package kraken

import (
	"net/http"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

// futuresTestConn stands in for a futures connection, which the order tracker tells apart by identity
type futuresTestConn struct {
	websocket.Connection
	name string
}

// futuresEmptySnapshot and futuresEmptyFillsSnapshot are open_orders and fills snapshots holding nothing. The fills of
// an order first reported after both on one connection all follow them, so they can make up the order's quantity
const (
	futuresEmptySnapshot      = `{"feed":"open_orders_snapshot","account":"DemoUser","orders":[]}`
	futuresEmptyFillsSnapshot = `{"feed":"fills_snapshot","account":"DemoUser","fills":[]}`
)

// futuresOrderUpdates sends futures feed messages, returning the order updates they relay
func futuresOrderUpdates(t *testing.T, ex *Exchange, msgs ...string) []*order.Detail {
	t.Helper()
	return futuresOrderUpdatesOn(t, ex, nil, msgs...)
}

// futuresOrderUpdatesOn sends futures feed messages on a connection, returning the order updates they relay
func futuresOrderUpdatesOn(t *testing.T, ex *Exchange, conn websocket.Connection, msgs ...string) []*order.Detail {
	t.Helper()
	for _, msg := range msgs {
		require.NoErrorf(t, ex.wsFuturesHandleData(t.Context(), conn, []byte(msg)), "wsFuturesHandleData must not error for %s", msg)
	}
	var details []*order.Detail
	for _, payload := range relayed(ex) {
		if d, ok := payload.(*order.Detail); ok {
			details = append(details, d)
		}
	}
	return details
}

// mergeOrderUpdates merges order updates into a stored order, as the engine merges each update the websocket relays
func mergeOrderUpdates(t *testing.T, stored *order.Detail, details []*order.Detail) {
	t.Helper()
	for _, d := range details {
		require.NoError(t, stored.UpdateOrderFromDetail(d), "UpdateOrderFromDetail must not error")
	}
}

// futuresFillMessage returns a fills feed message holding one fill of a buy order
func futuresFillMessage(feed, orderID, fillID, instrument string, quantity, remaining, price, fee float64, feeCurrency string) string {
	f := func(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
	return `{"feed":"` + feed + `","account":"DemoUser","fills":[{"instrument":"` + instrument + `","time":1600256966528,"price":` + f(price) +
		`,"seq":1,"buy":true,"qty":` + f(quantity) + `,"remaining_order_qty":` + f(remaining) + `,"order_id":"` + orderID + `","fill_id":"` + fillID +
		`","fill_type":"taker","fee_paid":` + f(fee) + `,"fee_currency":"` + feeCurrency + `","order_type":"lmt"}]}`
}

// futuresOpenOrder returns an open order of PF_XBTUSD at 20000, as the open_orders feed reports it
func futuresOpenOrder(orderID string, quantity, filled float64) string {
	f := func(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
	return `{"instrument":"PF_XBTUSD","time":1612275024153,"last_update_time":1612275178153,"qty":` + f(quantity) + `,"filled":` + f(filled) +
		`,"limit_price":20000,"stop_price":0,"type":"limit","order_id":"` + orderID + `","direction":0,"reduce_only":false}`
}

// futuresOpenOrderMessage returns an open_orders message holding a limit buy order of PF_XBTUSD at 20000
func futuresOpenOrderMessage(orderID string, quantity, filled float64, reason string) string {
	return `{"feed":"open_orders","order":` + futuresOpenOrder(orderID, quantity, filled) + `,"is_cancel":false,"reason":"` + reason + `"}`
}

// futuresRemovalMessage returns an open_orders message removing an order, which names the order alone
func futuresRemovalMessage(orderID, reason string) string {
	return `{"feed":"open_orders","order_id":"` + orderID + `","is_cancel":true,"reason":"` + reason + `"}`
}

func TestFuturesOrderFilledAtOnce(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	// A market order rests in no book, so open_orders reports nothing of it until it removes it as filled
	stored := &order.Detail{Exchange: ex.Name, AssetType: asset.Futures, OrderID: "o-1", Pair: futuresTestPair, Side: order.Buy, Type: order.Market, Status: order.New, Amount: 10}
	details := futuresOrderUpdates(t, ex,
		futuresEmptySnapshot,
		futuresEmptyFillsSnapshot,
		futuresFillMessage(wsFuturesFeedFills, "o-1", "f-1", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD"),
		futuresFillMessage(wsFuturesFeedFills, "o-1", "f-2", "PF_XBTUSD", 6, 0, 20010, 0.3, "USD"),
		futuresRemovalMessage("o-1", "full_fill"),
	)
	require.Len(t, details, 3, "each fill and the order's removal must relay an order update")
	assert.Zero(t, details[0].ExecutedAmount, "a fill should relay no executed quantity while the order's quantity is unknown")
	assert.Zero(t, details[0].Cost, "a fill should relay no cost while the order's quantity is unknown")
	assert.Equal(t, currency.USD, details[0].FeeAsset, "a fill should relay its fee currency, which the order store keeps from the update that first stores an order")
	mergeOrderUpdates(t, stored, details)
	assert.Equal(t, order.Filled, stored.Status, "the order should be stored filled")
	assert.Equal(t, 10.0, stored.Amount, "the order should keep its quantity")
	assert.Equal(t, 10.0, stored.ExecutedAmount, "the order should be stored with all of it executed")
	assert.InDelta(t, 200060.0, stored.Cost, 1e-6, "the order should be stored with its fills' cost")
	assert.InDelta(t, 20006.0, stored.AverageExecutedPrice, 1e-9, "the order should be stored with its fills' average price")
	assert.InDelta(t, 0.5, stored.Fee, 1e-12, "the order should be stored with its fills' fees")
	assert.Equal(t, currency.USD, details[1].FeeAsset, "the fee currency should be relayed with the fee")

	// The fills feed can repeat a fill, which must not be counted again
	repeated := futuresOrderUpdates(t, ex, futuresFillMessage(wsFuturesFeedFills, "o-1", "f-2", "PF_XBTUSD", 6, 0, 20010, 0.3, "USD"))
	require.Len(t, repeated, 1, "the repeated fill must relay an order update")
	assert.Equal(t, 10.0, repeated[0].ExecutedAmount, "a repeated fill should relay the same executed quantity")
	assert.InDelta(t, 0.5, repeated[0].Fee, 1e-12, "a repeated fill's fee should not be counted again")
}

func TestFuturesOrderPlacedElsewhere(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	// The engine stores an order it has not placed from its first update and merges the later ones into it
	details := futuresOrderUpdates(t, ex,
		futuresEmptySnapshot,
		futuresEmptyFillsSnapshot,
		futuresFillMessage(wsFuturesFeedFills, "o-18", "f-1", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD"),
		futuresFillMessage(wsFuturesFeedFills, "o-18", "f-2", "PF_XBTUSD", 6, 0, 20010, 0.3, "USD"),
	)
	require.Len(t, details, 2, "each fill must relay an order update")
	stored := details[0]
	mergeOrderUpdates(t, stored, details[1:])
	assert.Equal(t, order.Filled, stored.Status, "the order should be stored filled")
	assert.Equal(t, 10.0, stored.ExecutedAmount, "the order should be stored with its fills executed")
	assert.InDelta(t, 0.5, stored.Fee, 1e-12, "the order should be stored with its fills' fees")
	assert.Equal(t, currency.USD, stored.FeeAsset, "the order should be stored with its fee currency")
}

func TestFuturesOrderFillsAfterRemoval(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	details := futuresOrderUpdates(t, ex,
		`{"feed":"open_orders_snapshot","account":"DemoUser","orders":[`+futuresOpenOrder("o-2", 10, 0)+`]}`,
		// The fills feed can report the fills after open_orders reports the order filled
		futuresRemovalMessage("o-2", "full_fill"),
		futuresFillMessage(wsFuturesFeedFills, "o-2", "f-1", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD"),
		futuresFillMessage(wsFuturesFeedFills, "o-2", "f-2", "PF_XBTUSD", 6, 0, 20000, 0.3, "USD"),
	)
	require.Len(t, details, 4, "the snapshot, the removal and each fill must relay an order update")
	removal := details[1]
	assert.Equal(t, order.Filled, removal.Status, "the removal should relay the order filled")
	assert.Equal(t, 10.0, removal.ExecutedAmount, "the removal should relay the order's quantity as executed")
	assert.Zero(t, removal.Cost, "the removal should relay no cost before the fills are counted")
	for i, d := range details[2:] {
		assert.Equalf(t, 10.0, d.ExecutedAmount, "fill %d should not relay less executed than the removal did", i)
		assert.Equalf(t, order.Filled, d.Status, "fill %d should relay the order as it was removed", i)
	}
	assert.Zero(t, details[2].Cost, "the first fill should relay no cost, as it is not all the order executed")
	assert.Equal(t, 200000.0, details[3].Cost, "the last fill should relay the order's cost")
	assert.InDelta(t, 0.5, details[3].Fee, 1e-12, "the last fill should relay the order's fees")
}

func TestFuturesOrderPartlyFilledCancellation(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	details := futuresOrderUpdates(t, ex,
		futuresOpenOrderMessage("o-3", 10, 0, "new_placed_order_by_user"),
		futuresFillMessage(wsFuturesFeedFills, "o-3", "f-1", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD"),
		futuresOpenOrderMessage("o-3", 10, 4, "partial_fill"),
		futuresRemovalMessage("o-3", "cancelled_by_user"),
	)
	require.Len(t, details, 4, "each message must relay an order update")
	assert.Equal(t, &order.Detail{
		Exchange:             ex.Name,
		AssetType:            asset.Futures,
		OrderID:              "o-3",
		Status:               order.PartiallyFilledCancelled,
		Amount:               10,
		ExecutedAmount:       4,
		RemainingAmount:      6,
		Cost:                 80000,
		AverageExecutedPrice: 20000,
		Fee:                  0.2,
		FeeAsset:             currency.USD,
	}, details[3], "a cancellation after a fill should relay the order partly filled, with what it executed")
}

func TestFuturesOrderFillsAfterCancellation(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	// The fills feed can report a fill after open_orders reports the order cancelled
	details := futuresOrderUpdates(t, ex,
		futuresOpenOrderMessage("o-4", 10, 0, "new_placed_order_by_user"),
		futuresRemovalMessage("o-4", "cancelled_by_user"),
		futuresFillMessage(wsFuturesFeedFills, "o-4", "f-1", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD"),
	)
	require.Len(t, details, 3, "each message must relay an order update")
	assert.Equal(t, order.Cancelled, details[1].Status, "the cancellation should relay the order cancelled while nothing is known filled")
	stored := &order.Detail{Exchange: ex.Name, AssetType: asset.Futures, OrderID: "o-4", Status: order.Open, Amount: 10}
	mergeOrderUpdates(t, stored, details)
	assert.Equal(t, order.PartiallyFilledCancelled, stored.Status, "a fill after the cancellation should make the order partly filled")
	assert.Equal(t, 4.0, stored.ExecutedAmount, "a fill after the cancellation should relay what the order executed")
	assert.Equal(t, 80000.0, stored.Cost, "a fill after the cancellation should relay the order's cost")
}

func TestFuturesOrderRemovedWithOrder(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	// A removal can carry the order as last reported, before its last fill
	removal := `{"feed":"open_orders","order":` + futuresOpenOrder("o-5", 10, 4) + `,"is_cancel":true,"reason":"full_fill"}`
	details := futuresOrderUpdates(t, ex, futuresOpenOrderMessage("o-5", 10, 0, "new_placed_order_by_user"), removal)
	require.Len(t, details, 2, "the placement and the removal must relay an order update")
	assert.Equal(t, order.Filled, details[1].Status, "a removal as filled should relay the order filled")
	assert.Equal(t, 10.0, details[1].ExecutedAmount, "a removal as filled should relay the order's quantity as executed")
	assert.Zero(t, details[1].RemainingAmount, "a removal as filled should relay nothing remaining")
	state := ex.futuresOrders.get("o-5")
	assert.False(t, state.open, "a removal carrying the order should record the order as removed")
	assert.Equal(t, "full_fill", state.reason, "a removal carrying the order should record why it was removed")
}

func TestFuturesOrderStopTriggered(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	// A stop is removed as triggered, and the order it places is reported open
	details := futuresOrderUpdates(t, ex,
		futuresOpenOrderMessage("o-6", 10, 0, "new_placed_order_by_user"),
		futuresRemovalMessage("o-6", "stop_order_triggered"),
		futuresOpenOrderMessage("o-6", 10, 0, "limit_order_from_stop"),
		futuresFillMessage(wsFuturesFeedFills, "o-6", "f-1", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD"),
	)
	require.Len(t, details, 4, "each message must relay an order update")
	assert.Equal(t, order.UnknownStatus, details[3].Status, "a fill of an order open again should leave its status to open_orders")
	assert.Equal(t, 4.0, details[3].ExecutedAmount, "a fill of an order open again should relay what it executed")
}

func TestFuturesOrderAmendedQuantity(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	details := futuresOrderUpdates(t, ex,
		futuresOpenOrderMessage("o-7", 10, 0, "new_placed_order_by_user"),
		futuresFillMessage(wsFuturesFeedFills, "o-7", "f-1", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD"),
		futuresOpenOrderMessage("o-7", 20, 4, "edited_by_user"),
		// A fill's remaining quantity is of the quantity before the amendment
		futuresFillMessage(wsFuturesFeedFillsSnapshot, "o-7", "f-1", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD"),
	)
	require.Len(t, details, 4, "each message must relay an order update")
	for i, d := range details[1:] {
		assert.Equalf(t, 4.0, d.ExecutedAmount, "update %d should relay what the order executed, whatever its quantity", i)
	}
	assert.Equal(t, 20.0, details[3].Amount, "a fill should relay the order's amended quantity")
	assert.InDelta(t, 0.2, details[3].Fee, 1e-12, "a repeated fill's fee should not be counted again")
}

func TestFuturesOrderFillsSnapshot(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	// Filled in part before the connection, the order's earlier fill comes in the fills snapshot
	details := futuresOrderUpdates(t, ex,
		`{"feed":"open_orders_snapshot","account":"DemoUser","orders":[`+futuresOpenOrder("o-8", 10, 4)+`]}`,
		futuresFillMessage(wsFuturesFeedFillsSnapshot, "o-8", "f-1", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD"),
		// Resubscribing repeats the recent fills
		futuresFillMessage(wsFuturesFeedFillsSnapshot, "o-8", "f-1", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD"),
		futuresFillMessage(wsFuturesFeedFills, "o-8", "f-2", "PF_XBTUSD", 6, 0, 20000, 0.3, "USD"),
	)
	require.Len(t, details, 4, "the snapshot and each fill must relay an order update")
	assert.Equal(t, 80000.0, details[1].Cost, "the fills snapshot's fill should relay the cost of what the order executed before the connection")
	assert.Equal(t, 80000.0, details[2].Cost, "a repeated fill should not be counted again")
	assert.InDelta(t, 0.2, details[2].Fee, 1e-12, "a repeated fill's fee should not be counted again")
	assert.Equal(t, order.Filled, details[3].Status, "the fill leaving nothing remaining should relay the order filled")
	assert.Equal(t, 10.0, details[3].ExecutedAmount, "the live fill should relay the order filled in full")
	assert.Equal(t, 200000.0, details[3].Cost, "the live fill should relay both fills' cost")
	assert.InDelta(t, 0.5, details[3].Fee, 1e-12, "the live fill should relay both fills' fees")

	// The fills snapshot can come before the open orders snapshot
	early := futuresOrderUpdates(t, ex,
		futuresFillMessage(wsFuturesFeedFillsSnapshot, "o-9", "f-3", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD"),
		`{"feed":"open_orders_snapshot","account":"DemoUser","orders":[`+futuresOpenOrder("o-9", 10, 4)+`]}`,
	)
	require.Len(t, early, 2, "the fill and the snapshot must relay an order update")
	assert.Zero(t, early[0].ExecutedAmount, "a fill of an order not yet reported should relay its trade alone")
	assert.Equal(t, 80000.0, early[1].Cost, "the open orders snapshot should relay the cost of the fill counted before it")

	missing := futuresOrderUpdates(t, ex,
		`{"feed":"open_orders_snapshot","account":"DemoUser","orders":[`+futuresOpenOrder("o-10", 10, 4)+`]}`,
		futuresFillMessage(wsFuturesFeedFills, "o-10", "f-4", "PF_XBTUSD", 6, 0, 20000, 0.3, "USD"),
	)
	require.Len(t, missing, 2, "the snapshot and the fill must relay an order update")
	assert.Equal(t, 10.0, missing[1].ExecutedAmount, "a fill leaving nothing remaining should relay the order filled in full")
	assert.Zero(t, missing[1].Cost, "an order whose earlier fill was not reported should relay no cost")
	assert.Zero(t, missing[1].Fee, "an order whose earlier fill was not reported should relay no fee")
}

func TestFuturesOrderFillsOfAnUnnamedOrder(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	// An order open_orders has not named, whose first fill came in a fills snapshot, may have had earlier fills, even
	// after an open_orders snapshot
	details := futuresOrderUpdates(t, ex,
		futuresEmptySnapshot,
		futuresFillMessage(wsFuturesFeedFillsSnapshot, "o-11", "f-1", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD"),
		futuresFillMessage(wsFuturesFeedFills, "o-11", "f-2", "PF_XBTUSD", 6, 0, 20000, 0.3, "USD"),
	)
	require.Len(t, details, 2, "each fill must relay an order update")
	for i, d := range details {
		assert.Zerof(t, d.ExecutedAmount, "fill %d of an order known only by fills partly from a snapshot should relay no executed quantity", i)
		assert.Zerof(t, d.Amount, "fill %d of an order known only by fills partly from a snapshot should relay no quantity", i)
		assert.Lenf(t, d.Trades, 1, "fill %d should relay its trade", i)
	}
	assert.Equal(t, order.UnknownStatus, details[1].Status, "an order known only by fills partly from a snapshot should not be relayed filled")
}

func TestFuturesOrderFillsWithoutOpenOrders(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	// With myTrades subscribed alone there is no open_orders snapshot, so a live fill can follow fills of the order the
	// fills snapshot left out
	stored := &order.Detail{Exchange: ex.Name, AssetType: asset.Futures, OrderID: "o-15", Pair: futuresTestPair, Side: order.Buy, Status: order.PartiallyFilled, Amount: 10, ExecutedAmount: 4}
	details := futuresOrderUpdates(t, ex, futuresEmptyFillsSnapshot, futuresFillMessage(wsFuturesFeedFills, "o-15", "f-2", "PF_XBTUSD", 6, 0, 20010, 0.3, "USD"))
	require.Len(t, details, 1, "the fill must relay an order update")
	assert.Zero(t, details[0].ExecutedAmount, "a fill without an open_orders snapshot should relay no executed quantity")
	mergeOrderUpdates(t, stored, details)
	assert.Equal(t, 10.0, stored.Amount, "the stored order should keep its quantity")
	assert.Equal(t, 4.0, stored.ExecutedAmount, "the stored order should keep what it executed")
	assert.Len(t, stored.Trades, 1, "the stored order should gain the fill's trade")
}

func TestFuturesOrderFillsCoverage(t *testing.T) {
	t.Parallel()
	first, second := &futuresTestConn{name: "first"}, &futuresTestConn{name: "second"}
	snapshots := []string{futuresEmptySnapshot, futuresEmptyFillsSnapshot}
	partFill := futuresFillMessage(wsFuturesFeedFills, "o-16", "f-1", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD")
	type report struct {
		conn websocket.Connection
		msgs []string
	}
	for _, tc := range []struct {
		name     string
		reports  []report
		conn     websocket.Connection
		executed float64
	}{
		{"after both snapshots", []report{{first, snapshots}}, first, 6},
		{"after both snapshots, fills first", []report{{first, []string{futuresEmptyFillsSnapshot, futuresEmptySnapshot}}}, first, 6},
		{"fills after both snapshots", []report{{first, append(slices.Clone(snapshots), partFill)}}, first, 10},
		{"without snapshots", nil, first, 0},
		{"after the open_orders snapshot alone", []report{{first, []string{futuresEmptySnapshot}}}, first, 0},
		{"after the fills snapshot alone", []report{{first, []string{futuresEmptyFillsSnapshot}}}, first, 0},
		// Reconnecting, the fill can come on the new connection before its snapshots
		{"after snapshots on another connection", []report{{first, snapshots}}, second, 0},
		{"after snapshots split between connections", []report{{first, []string{futuresEmptySnapshot}}, {second, []string{futuresEmptyFillsSnapshot}}}, second, 0},
		// A removal before the snapshots can follow fills that went unseen, which fills reported after them don't show
		{"of an order removed before the snapshots", []report{{first, []string{futuresRemovalMessage("o-16", "full_fill"), futuresEmptySnapshot, futuresEmptyFillsSnapshot}}}, first, 0},
		{"of an order removed between the snapshots", []report{{first, []string{futuresEmptySnapshot, futuresRemovalMessage("o-16", "full_fill"), futuresEmptyFillsSnapshot}}}, first, 0},
		{"of an order removed after the snapshots", []report{{first, append(slices.Clone(snapshots), futuresRemovalMessage("o-16", "full_fill"))}}, first, 6},
		{"of an order filled before the snapshots", []report{{first, []string{futuresEmptySnapshot, partFill, futuresEmptyFillsSnapshot}}}, first, 0},
		{"of an order filled on a connection before", []report{{first, append(slices.Clone(snapshots), partFill)}}, second, 0},
		// Fills of the order can have come between the connections, which the new fills snapshot can leave out
		{"of an order filled on a connection before its snapshots", []report{{first, append(slices.Clone(snapshots), partFill)}, {second, snapshots}}, second, 0},
		{"either side of another open_orders snapshot", []report{{first, append(slices.Clone(snapshots), partFill, futuresEmptySnapshot)}}, first, 10},
		{"either side of another fills snapshot", []report{{first, append(slices.Clone(snapshots), partFill, futuresEmptyFillsSnapshot)}}, first, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestExchange(t)
			seedTestAssetNames(ex)
			for _, r := range tc.reports {
				futuresOrderUpdatesOn(t, ex, r.conn, r.msgs...)
			}
			details := futuresOrderUpdatesOn(t, ex, tc.conn, futuresFillMessage(wsFuturesFeedFills, "o-16", "f-2", "PF_XBTUSD", 6, 0, 20010, 0.3, "USD"))
			require.Len(t, details, 1, "the fill must relay an order update")
			assert.Equal(t, tc.executed, details[0].ExecutedAmount, "a fill leaving nothing remaining should relay the order's fills as executed only when every fill of it was seen")
		})
	}
}

func TestFuturesOrderFeedUnsubscribed(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		unsubscribed []string
		// refused has Kraken refuse the unsubscription
		refused bool
		// resubscribed are the snapshots of the feeds subscribed to again
		resubscribed []string
		// filledBefore has the order fill in part before the unsubscription
		filledBefore bool
		executed     float64
	}{
		{name: "open_orders", unsubscribed: []string{wsFuturesFeedOpenOrders}},
		{name: "fills", unsubscribed: []string{wsFuturesFeedFills}},
		{name: "open_orders resubscribed", unsubscribed: []string{wsFuturesFeedOpenOrders}, resubscribed: []string{futuresEmptySnapshot}, executed: 6},
		{name: "fills resubscribed", unsubscribed: []string{wsFuturesFeedFills}, resubscribed: []string{futuresEmptyFillsSnapshot}, executed: 6},
		// An order placed while both were unsubscribed, which the new fills snapshot left out
		{name: "both, fills resubscribed", unsubscribed: []string{wsFuturesFeedOpenOrders, wsFuturesFeedFills}, resubscribed: []string{futuresEmptyFillsSnapshot}},
		// The fills feed stayed subscribed, so every fill of an order it covered came live
		{name: "open_orders, filled either side", unsubscribed: []string{wsFuturesFeedOpenOrders}, filledBefore: true, executed: 10},
		{name: "open_orders resubscribed, filled either side", unsubscribed: []string{wsFuturesFeedOpenOrders}, resubscribed: []string{futuresEmptySnapshot}, filledBefore: true, executed: 10},
		{name: "fills resubscribed, filled either side", unsubscribed: []string{wsFuturesFeedFills}, resubscribed: []string{futuresEmptyFillsSnapshot}, filledBefore: true},
		{name: "fills refused", unsubscribed: []string{wsFuturesFeedFills}, refused: true, filledBefore: true, executed: 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := new(mockFuturesServer)
			ex, conn := newFuturesTestExchange(t, server)
			seedTestAssetNames(ex)
			subs := map[string]*subscription.Subscription{
				wsFuturesFeedOpenOrders: futuresSubscription(subscription.MyOrdersChannel, wsFuturesFeedOpenOrders),
				wsFuturesFeedFills:      futuresSubscription(subscription.MyTradesChannel, wsFuturesFeedFills),
			}
			require.NoError(t, ex.futuresSubscribe(t.Context(), conn, subscription.List{subs[wsFuturesFeedOpenOrders], subs[wsFuturesFeedFills]}), "futuresSubscribe must not error")
			futuresOrderUpdatesOn(t, ex, conn, futuresEmptySnapshot, futuresEmptyFillsSnapshot)
			if tc.filledBefore {
				futuresOrderUpdatesOn(t, ex, conn, futuresFillMessage(wsFuturesFeedFills, "o-21", "f-1", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD"))
			}
			var unsubscribe subscription.List
			for _, feed := range tc.unsubscribed {
				unsubscribe = append(unsubscribe, subs[feed])
			}
			if tc.refused {
				server.m.Lock()
				server.reply = `{"event":"unsubscribed_failed","feed":"` + tc.unsubscribed[0] + `"}`
				server.m.Unlock()
				require.ErrorIs(t, ex.futuresUnsubscribe(t.Context(), conn, unsubscribe), errFuturesSubscriptionFailed, "futuresUnsubscribe must error when Kraken refuses")
			} else {
				require.NoError(t, ex.futuresUnsubscribe(t.Context(), conn, unsubscribe), "futuresUnsubscribe must not error")
			}
			futuresOrderUpdatesOn(t, ex, conn, tc.resubscribed...)
			details := futuresOrderUpdatesOn(t, ex, conn, futuresFillMessage(wsFuturesFeedFills, "o-21", "f-2", "PF_XBTUSD", 6, 0, 20010, 0.3, "USD"))
			require.Len(t, details, 1, "the fill must relay an order update")
			assert.Equal(t, tc.executed, details[0].ExecutedAmount, "a fill leaving nothing remaining should relay the order's fills as executed only when every fill of it was seen")
		})
	}
}

func TestFuturesOrderFillsPartlyUnsubscribed(t *testing.T) {
	t.Parallel()
	server := new(mockFuturesServer)
	ex, conn := newFuturesTestExchange(t, server)
	seedTestAssetNames(ex)
	fills := futuresSubscription(subscription.MyTradesChannel, wsFuturesFeedFills, futuresTestPair, piXBTUSDPair)
	orders := futuresSubscription(subscription.MyOrdersChannel, wsFuturesFeedOpenOrders)
	require.NoError(t, ex.futuresSubscribe(t.Context(), conn, subscription.List{orders, fills}), "futuresSubscribe must not error")
	futuresOrderUpdatesOn(t, ex, conn, futuresEmptySnapshot, futuresEmptyFillsSnapshot)
	// Kraken unsubscribes one product of the fills subscription and refuses the other, so that product's fills go
	// unseen
	server.m.Lock()
	server.alerts = map[string]string{"PI_XBTUSD": "Unsubscribe failed"}
	server.m.Unlock()
	require.ErrorIs(t, ex.futuresUnsubscribe(t.Context(), conn, subscription.List{fills}), errFuturesSubscriptionFailed, "futuresUnsubscribe must error for the product Kraken refuses")
	details := futuresOrderUpdatesOn(t, ex, conn, futuresFillMessage(wsFuturesFeedFills, "o-22", "f-1", "PF_XBTUSD", 6, 0, 20010, 0.3, "USD"))
	require.Len(t, details, 1, "the fill must relay an order update")
	assert.Zero(t, details[0].ExecutedAmount, "a fill after part of the fills feed was unsubscribed should relay no executed quantity")
}

func TestFuturesOrderInverseContract(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	details := futuresOrderUpdates(t, ex,
		futuresEmptySnapshot,
		futuresEmptyFillsSnapshot,
		futuresFillMessage(wsFuturesFeedFills, "o-12", "f-1", "PI_XBTUSD", 1000, 1000, 10000, 0.00001, "BTC"),
		futuresFillMessage(wsFuturesFeedFills, "o-12", "f-2", "PI_XBTUSD", 1000, 0, 20000, 0.00002, "BTC"),
	)
	require.Len(t, details, 2, "each fill must relay an order update")
	// An inverse contract's quantity is in US dollars, so its average price is what it bought over the bitcoin it cost
	assert.InDelta(t, 2000/(0.1+0.05), details[1].AverageExecutedPrice, 1e-9, "an inverse contract's average price should weight its fills by their base value")
	assert.Zero(t, details[1].Cost, "an inverse contract's cost should not be relayed")
	assert.InDelta(t, 0.00003, details[1].Fee, 1e-15, "the fills' fees should be relayed")
	assert.Equal(t, currency.XBT, details[1].FeeAsset, "the fee currency should be relayed by its alternative name")
}

func TestFuturesOrderMixedFeeCurrencies(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	details := futuresOrderUpdates(t, ex,
		futuresEmptySnapshot,
		futuresEmptyFillsSnapshot,
		futuresFillMessage(wsFuturesFeedFills, "o-13", "f-1", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD"),
		futuresFillMessage(wsFuturesFeedFills, "o-13", "f-2", "PF_XBTUSD", 6, 0, 20000, 0.3, "EUR"),
	)
	require.Len(t, details, 2, "each fill must relay an order update")
	assert.Equal(t, 200000.0, details[1].Cost, "the order's cost should be relayed")
	assert.Zero(t, details[1].Fee, "fees charged in more than one currency should relay no fee")
	assert.True(t, details[1].FeeAsset.IsEmpty(), "fees charged in more than one currency should relay no fee currency")
}

func TestFuturesOrderFillsAddUp(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	// 0.1 and 0.7 add up to 0.7999999999999999 in floating point, short of the order's 0.8
	details := futuresOrderUpdates(t, ex,
		futuresOpenOrderMessage("o-14", 0.8, 0, "new_placed_order_by_user"),
		futuresFillMessage(wsFuturesFeedFills, "o-14", "f-1", "PF_XBTUSD", 0.1, 0.7, 20000, 0.01, "USD"),
		futuresFillMessage(wsFuturesFeedFills, "o-14", "f-2", "PF_XBTUSD", 0.7, 0, 20000, 0.07, "USD"),
	)
	require.Len(t, details, 3, "each message must relay an order update")
	assert.Equal(t, 0.8, details[2].ExecutedAmount, "a fill leaving nothing remaining should relay the order's quantity as executed")
	assert.InDelta(t, 16000.0, details[2].Cost, 1e-9, "fills adding up to the order's quantity but for rounding should relay its cost")
}

func TestFuturesOrderTracker(t *testing.T) {
	t.Parallel()
	var tracker futuresOrderTracker
	open := func(orderID string) WebsocketFuturesOpenOrder {
		return WebsocketFuturesOpenOrder{Instrument: "PF_XBTUSD", Quantity: 10, OrderID: orderID}
	}
	tracker.snapshot(nil, []WebsocketFuturesOpenOrder{open("left"), open("finished"), open("stale")})
	assert.Equal(t, 10.0, tracker.get("left").quantity, "a snapshot should record its orders")
	tracker.finish(nil, "finished", "cancelled_by_user")
	tracker.finish(nil, "stale", "cancelled_by_user")
	tracker.fill(nil, &WebsocketFuturesFill{Instrument: "PF_XBTUSD", OrderID: "filled only", FillID: "f-1", Quantity: 1, RemainingOrderQuantity: 1}, true, currency.USD)
	tracker.mu.Lock()
	tracker.orders["stale"].updated = time.Now().Add(-futuresOrderRetention - time.Second)
	tracker.mu.Unlock()

	// An open order missing from a later snapshot left the open orders unseen
	tracker.snapshot(nil, nil)
	assert.Zero(t, tracker.get("left").quantity, "a snapshot should forget an open order it leaves out")
	assert.Equal(t, 10.0, tracker.get("finished").quantity, "a snapshot should keep a removed order for its later fills")
	assert.Equal(t, 1.0, tracker.get("filled only").fillQuantity, "a snapshot should keep an order known only by its fills")
	assert.Zero(t, tracker.get("stale").quantity, "a removed order should be forgotten after futuresOrderRetention")

	state := tracker.finish(nil, "unknown", "cancelled_by_user")
	assert.Equal(t, "cancelled_by_user", state.reason, "removing an order the tracker does not know should record why, for its later fills")

	// A fill not reported as it happened can follow fills that went unseen
	tracker.fillsSnapshot(nil)
	fill := func(fillID string, live bool) futuresOrderState {
		return tracker.fill(nil, &WebsocketFuturesFill{Instrument: "PF_XBTUSD", OrderID: "repeated", FillID: fillID, Quantity: 1, RemainingOrderQuantity: 1}, live, currency.USD)
	}
	require.True(t, fill("f-2", true).liveOnly, "a fill reported as it happened on a covered connection must be counted live")
	assert.False(t, fill("f-3", false).liveOnly, "a fill not reported as it happened should end an order's live fills")
}

func TestSameQuantity(t *testing.T) {
	t.Parallel()
	assert.True(t, sameQuantity(0.1+0.2, 0.3), "sameQuantity should accept the rounding of adding fills")
	assert.True(t, sameQuantity(0, 0), "sameQuantity should accept two zeros")
	assert.False(t, sameQuantity(0.3, 0.30001), "sameQuantity should tell different quantities apart")
	assert.False(t, sameQuantity(4, 10), "sameQuantity should tell different quantities apart")
}

// setFuturesMarginCurrencies records the REST margin accounts' currencies for the default credentials, as
// UpdateAccountBalances does
func setFuturesMarginCurrencies(ex *Exchange, currencies map[string]currency.Code) {
	ex.futuresMarginCurrenciesMu.Lock()
	defer ex.futuresMarginCurrenciesMu.Unlock()
	ex.futuresMarginCurrencies = map[accounts.Credentials]map[string]currency.Code{*ex.GetDefaultCredentials(): currencies}
}

func TestWebsocketFuturesProcessBalancesSnapshot(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	useTestCredentials(ex)
	require.NoError(t, ex.Accounts.Save(t.Context(), accounts.SubAccounts{
		{ID: "cash", AssetType: asset.Futures, Balances: accounts.CurrencyBalances{currency.USD: {Total: 1, Free: 1}, currency.EUR: {Total: 2, Free: 2}}},
		{ID: "flex", AssetType: asset.Futures, Balances: accounts.CurrencyBalances{currency.USD: {Total: 3, Free: 3}}},
		{ID: "fi_xbtusd", AssetType: asset.Futures, Balances: accounts.CurrencyBalances{currency.XBT: {Total: 1, Free: 1}, currency.XRP: {Total: 9, Free: 9}}},
	}, true), "Save must not error")
	setFuturesMarginCurrencies(ex, map[string]currency.Code{"fi_xbtusd": currency.NewCode("xbt"), "fi_etheur": currency.ETH})
	stored := func(id string) accounts.CurrencyBalances {
		subAccts, err := ex.Accounts.SubAccounts(ex.GetDefaultCredentials(), asset.Futures)
		require.NoError(t, err, "SubAccounts must not error")
		for _, s := range subAccts {
			if s.ID == id {
				for c, b := range s.Balances {
					b.UpdatedAt = time.Time{}
					s.Balances[c] = b
				}
				return s.Balances
			}
		}
		return nil
	}

	snapshot := `{"feed":"balances_snapshot","account":"DemoUser","holding":{"usd":100},"flex_futures":{"currencies":{}},"futures":{` +
		`"F-XBT:USD":{"name":"F-XBT:USD","pair":"XBT/USD","unit":"XBT","balance":0.5,"available":0.25},` +
		`"F-ETH:EUR":{"name":"F-ETH:EUR","pair":"ETH/EUR","unit":"EUR","balance":7,"available":7}},"timestamp":1640995200000,"seq":1}`
	require.NoError(t, ex.wsFuturesHandleData(t.Context(), nil, []byte(snapshot)), "wsFuturesHandleData must not error for a balances snapshot")
	relayed(ex)
	assert.Equal(t, accounts.CurrencyBalances{currency.USD: {Currency: currency.USD, Total: 100, Free: 100}}, stored("cash"), "a snapshot should store its cash wallet whole, removing a currency it no longer lists")
	assert.Empty(t, stored("flex"), "a snapshot carrying an empty multi-collateral wallet should empty it")
	assert.Equal(t, accounts.CurrencyBalances{
		currency.XBT: {Currency: currency.XBT, Total: 0.5, Hold: 0.25, Free: 0.25},
		currency.XRP: {Currency: currency.XRP, Total: 9, Free: 9},
	}, stored("fi_xbtusd"), "a snapshot should store a margin account under its REST name, keeping the currencies it does not report")
	assert.Nil(t, stored("fi_etheur"), "a margin account whose currency differs from the REST account's should be left to REST")

	update := `{"feed":"balances","account":"DemoUser","flex_futures":{"currencies":{"XBT":{"quantity":0.1,"available":0.1}}},"timestamp":1640995200001,"seq":2}`
	require.NoError(t, ex.wsFuturesHandleData(t.Context(), nil, []byte(update)), "wsFuturesHandleData must not error for a balances update")
	relayed(ex)
	assert.Equal(t, accounts.CurrencyBalances{currency.USD: {Currency: currency.USD, Total: 100, Free: 100}}, stored("cash"), "an update leaving the cash wallet out should keep it")
	assert.Equal(t, accounts.CurrencyBalances{currency.XBT: {Currency: currency.XBT, Total: 0.1, Free: 0.1}}, stored("flex"), "an update should store the balances it holds")

	require.NoError(t, ex.wsFuturesHandleData(t.Context(), nil, []byte(`{"feed":"balances_snapshot","account":"DemoUser","holding":{"usd":100},"timestamp":1640995200002,"seq":3}`)), "wsFuturesHandleData must not error for a balances snapshot")
	relayed(ex)
	assert.Equal(t, accounts.CurrencyBalances{currency.XBT: {Currency: currency.XBT, Total: 0.1, Free: 0.1}}, stored("flex"), "a snapshot leaving a wallet out should keep it")

	require.NoError(t, ex.wsFuturesHandleData(t.Context(), nil, []byte(`{"feed":"balances_snapshot","account":"DemoUser","holding":{},"timestamp":1640995200003,"seq":4}`)), "wsFuturesHandleData must not error for a balances snapshot")
	relayed(ex)
	assert.Empty(t, stored("cash"), "a snapshot carrying an empty cash wallet should empty it")
}

func TestWebsocketFuturesProcessBalancesMarginAccount(t *testing.T) {
	t.Parallel()
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		if !assert.Equal(t, "/derivatives/api/v3/accounts", r.URL.Path, "only the futures accounts should be requested") {
			http.Error(w, "unexpected request", http.StatusNotFound)
			return
		}
		// Another account's credentials hold no margin account
		accounts := `"fi_xbtusd":{"type":"marginAccount","currency":"xbt","balances":{"xbt":"1","xrp":"9","FI_XBTUSD_171215":"5000"},` +
			`"auxiliary":{"af":0.5},"marginRequirements":{},"triggerEstimates":{}},`
		if r.Header.Get("APIKey") != "test-key" {
			accounts = ""
		}
		_, err := w.Write([]byte(`{"result":"success","accounts":{` + accounts + `"cash":{"type":"cashAccount","balances":{"usd":"100"}},` +
			`"flex":{"type":"multiCollateralMarginAccount","currencies":{}}},"serverTime":"2026-10-16T00:00:00.000Z"}`))
		assert.NoError(t, err, "Write should not error")
	})
	_, err := ex.UpdateAccountBalances(t.Context(), asset.Futures)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	other := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "other-key", Secret: testSecret})
	_, err = ex.UpdateAccountBalances(other, asset.Futures)
	require.NoError(t, err, "UpdateAccountBalances must not error for other credentials")

	update := `{"feed":"balances","account":"DemoUser","futures":{"F-XBT:USD":{"name":"F-XBT:USD","pair":"XBT/USD","unit":"XBT","balance":0.75,"available":0.5}},"timestamp":1640995200000,"seq":2}`
	require.NoError(t, ex.wsFuturesHandleData(t.Context(), nil, []byte(update)), "wsFuturesHandleData must not error for a balances update")
	got := relayed(ex)
	require.Len(t, got, 2, "the update must relay the margin account and the message")
	subAccts, ok := got[0].(accounts.SubAccounts)
	require.True(t, ok, "the margin account must be relayed as a sub-account")
	require.Len(t, subAccts, 1, "one margin account must be relayed")
	assert.Equal(t, "fi_xbtusd", subAccts[0].ID, "the margin account should be relayed under its REST name")
	b, err := ex.Accounts.GetBalance("fi_xbtusd", ex.GetDefaultCredentials(), asset.Futures, currency.XBT)
	require.NoError(t, err, "GetBalance must not error")
	b.UpdatedAt = time.Time{}
	assert.Equal(t, accounts.Balance{Currency: currency.XBT, Total: 0.75, Hold: 0.25, Free: 0.5}, b, "the margin account's balance should be stored under its REST name, whatever another account's REST read held")
	b, err = ex.Accounts.GetBalance("fi_xbtusd", ex.GetDefaultCredentials(), asset.Futures, currency.XRP)
	require.NoError(t, err, "GetBalance must not error")
	assert.Equal(t, 9.0, b.Total, "the margin account's other currencies should be kept")
}

func TestFuturesMarginAccountName(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	useTestCredentials(ex)
	setFuturesMarginCurrencies(ex, map[string]currency.Code{"fi_xbtusd": currency.XBT})
	for _, tc := range []struct {
		account WebsocketFuturesMarginAccount
		exp     bool
	}{
		{WebsocketFuturesMarginAccount{Name: "F-XBT:USD", Pair: "XBT/USD", Unit: "XBT"}, true},
		{WebsocketFuturesMarginAccount{Name: "F-XBT:USD", Pair: "XBT/USD", Unit: "xbt"}, true},
		{WebsocketFuturesMarginAccount{Name: "F-XBT:USD", Pair: "XBT/USD", Unit: "USD"}, false},
		{WebsocketFuturesMarginAccount{Name: "F-ETH:USD", Pair: "XBT/USD", Unit: "XBT"}, false},
		{WebsocketFuturesMarginAccount{Name: "F-ETH:USD", Pair: "ETH/USD", Unit: "ETH"}, false},
		{WebsocketFuturesMarginAccount{Name: "F-XBT", Pair: "XBTUSD", Unit: "XBT"}, false},
	} {
		name, ok := ex.futuresMarginAccountName(t.Context(), &tc.account)
		assert.Equalf(t, tc.exp, ok, "futuresMarginAccountName should match %+v only to a REST account of its pair and currency", tc.account)
		if ok {
			assert.Equal(t, "fi_xbtusd", name, "futuresMarginAccountName should return the REST account's name")
		}
	}
	other := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "other-key", Secret: testSecret})
	_, ok := ex.futuresMarginAccountName(other, &WebsocketFuturesMarginAccount{Name: "F-XBT:USD", Pair: "XBT/USD", Unit: "XBT"})
	assert.False(t, ok, "futuresMarginAccountName should not match an account another account's REST read recorded")
}
