package kraken

import (
	"context"
	"net/http"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/types"
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
	return `{"feed":"` + feed + `","account":"DemoUser","fills":[` + futuresFill(orderID, fillID, instrument, quantity, remaining, price, fee, feeCurrency) + `]}`
}

// futuresFill returns a fill of a buy order, as the fills feed reports it
func futuresFill(orderID, fillID, instrument string, quantity, remaining, price, fee float64, feeCurrency string) string {
	f := func(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
	return `{"instrument":"` + instrument + `","time":1600256966528,"price":` + f(price) + `,"seq":1,"buy":true,"qty":` + f(quantity) +
		`,"remaining_order_qty":` + f(remaining) + `,"order_id":"` + orderID + `","fill_id":"` + fillID + `","fill_type":"taker","fee_paid":` + f(fee) +
		`,"fee_currency":"` + feeCurrency + `","order_type":"lmt"}`
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

// assertFuturesOrderTotalsUnset asserts that an order holds no average price, cost or fee, which the futures feeds
// leave unset as they cannot show the fills they report are all of an order's
func assertFuturesOrderTotalsUnset(t *testing.T, d *order.Detail, of string) {
	t.Helper()
	assert.Zerof(t, d.AverageExecutedPrice, "%s should hold no average price", of)
	assert.Zerof(t, d.Cost, "%s should hold no cost", of)
	assert.Zerof(t, d.Fee, "%s should hold no fee", of)
	assert.Truef(t, d.FeeAsset.IsEmpty(), "%s should hold no fee currency", of)
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
	assert.Equal(t, order.UnknownStatus, details[0].Status, "a fill leaving some of the order remaining should leave its status to open_orders")
	mergeOrderUpdates(t, stored, details)
	assert.Equal(t, order.Filled, stored.Status, "the order should be stored filled")
	assert.Equal(t, 10.0, stored.Amount, "the order should keep its quantity")
	assert.Equal(t, 10.0, stored.ExecutedAmount, "the order should be stored with all of it executed")
	// Every fill was reported as it happened, but the tracker cannot tell that from a sequence that missed some, so it
	// leaves even these totals unset
	assertFuturesOrderTotalsUnset(t, stored, "an order whose fills were all reported as they happened")

	// The fills feed can repeat a fill, which must not be counted again
	repeated := futuresOrderUpdates(t, ex, futuresFillMessage(wsFuturesFeedFills, "o-1", "f-2", "PF_XBTUSD", 6, 0, 20010, 0.3, "USD"))
	require.Len(t, repeated, 1, "the repeated fill must relay an order update")
	assert.Equal(t, 10.0, repeated[0].ExecutedAmount, "a repeated fill should relay the same executed quantity")
	repeated = futuresOrderUpdates(t, ex, futuresFillMessage(wsFuturesFeedFillsSnapshot, "o-1", "f-2", "PF_XBTUSD", 6, 0, 20010, 0.3, "USD"))
	require.Len(t, repeated, 1, "the fill a fills snapshot repeats must relay an order update")
	assert.Equal(t, 10.0, repeated[0].ExecutedAmount, "a fill a fills snapshot repeats should relay the same executed quantity")
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
	assertFuturesOrderTotalsUnset(t, stored, "an order placed elsewhere")
	assert.Len(t, stored.Trades, 2, "the order should be stored with each fill's trade, which holds the fill's fee")
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
	for i, d := range details[2:] {
		assert.Equalf(t, 10.0, d.ExecutedAmount, "fill %d should not relay less executed than the removal did", i)
		assert.Equalf(t, order.Filled, d.Status, "fill %d should relay the order as it was removed", i)
	}
	assertFuturesOrderTotalsUnset(t, details[3], "the last fill of an order removed as filled")
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
		Exchange:        ex.Name,
		AssetType:       asset.Futures,
		OrderID:         "o-3",
		Status:          order.PartiallyFilledCancelled,
		Amount:          10,
		ExecutedAmount:  4,
		RemainingAmount: 6,
	}, details[3], "a cancellation after a fill should relay the order partly filled, with what it executed")
}

func TestFuturesOrderFillReportedBeforeItsUpdate(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	// 4 of the order filled before the feeds subscribed, in a fill the fills snapshot leaves out, and its next fill is
	// reported before open_orders reports it, when that fill alone makes up all the feeds have reported filled
	details := futuresOrderUpdates(t, ex,
		`{"feed":"open_orders_snapshot","account":"DemoUser","orders":[`+futuresOpenOrder("o-40", 10, 4)+`]}`,
		futuresEmptyFillsSnapshot,
		futuresFillMessage(wsFuturesFeedFills, "o-40", "f-2", "PF_XBTUSD", 5, 1, 21000, 0.5, "USD"),
		futuresOpenOrderMessage("o-40", 10, 9, "partial_fill"),
		futuresRemovalMessage("o-40", "cancelled_by_user"),
	)
	require.Len(t, details, 4, "the snapshot, the fill, the update and the removal must each relay an order update")
	stored := details[0]
	mergeOrderUpdates(t, stored, details[1:])
	stored.LastUpdated = time.Time{}
	exp := &order.Detail{
		Exchange:        ex.Name,
		AssetType:       asset.Futures,
		OrderID:         "o-40",
		Pair:            futuresTestPair,
		Side:            order.Buy,
		Type:            order.Limit,
		Status:          order.PartiallyFilledCancelled,
		Price:           20000,
		Amount:          10,
		ExecutedAmount:  9,
		RemainingAmount: 1,
		Date:            time.UnixMilli(1612275024153),
		Trades:          []order.TradeHistory{{TID: "f-2", Price: 21000, Amount: 5, Fee: 0.5, FeeAsset: "USD", Exchange: ex.Name, Type: order.Limit, Side: order.Buy, Timestamp: time.UnixMilli(1600256966528)}},
	}
	assert.Equal(t, exp, stored, "the order should be stored with what it executed, and its fill's price and fee on the fill's trade alone")
}

func TestFuturesOrderFillsCountedAcrossReports(t *testing.T) {
	t.Parallel()
	update := func(quantity, filled float64, reason string) string {
		return futuresOpenOrderMessage("o-50", quantity, filled, reason)
	}
	fill := func(feed, fillID string, quantity, remaining float64) string {
		return futuresFillMessage(feed, "o-50", fillID, "PF_XBTUSD", quantity, remaining, 20000, 0.1, "USD")
	}
	snapshot := func(filled float64) string {
		return `{"feed":"open_orders_snapshot","account":"DemoUser","orders":[` + futuresOpenOrder("o-50", 10, filled) + `]}`
	}
	for _, tc := range []struct {
		name string
		msgs []string
		// quantity, executed and remaining are what the last update relays of the order
		quantity, executed, remaining float64
	}{
		{
			name:     "partial_fill update",
			msgs:     []string{update(10, 0, "new_placed_order_by_user"), fill(wsFuturesFeedFills, "f-1", 4, 6), update(10, 4, "partial_fill"), fill(wsFuturesFeedFills, "f-2", 3, 3)},
			quantity: 10, executed: 7, remaining: 3,
		},
		{
			name:     "partial_fill update reported after both fills",
			msgs:     []string{update(10, 0, "new_placed_order_by_user"), fill(wsFuturesFeedFills, "f-1", 4, 6), fill(wsFuturesFeedFills, "f-2", 3, 3), update(10, 4, "partial_fill")},
			quantity: 10, executed: 7, remaining: 3,
		},
		{
			name:     "placement reported after the first fill",
			msgs:     []string{fill(wsFuturesFeedFills, "f-1", 4, 6), update(10, 0, "new_placed_order_by_user"), fill(wsFuturesFeedFills, "f-2", 3, 3)},
			quantity: 10, executed: 7, remaining: 3,
		},
		{
			name:     "amended up",
			msgs:     []string{update(10, 0, "new_placed_order_by_user"), fill(wsFuturesFeedFills, "f-1", 4, 6), update(20, 4, "edited_by_user"), fill(wsFuturesFeedFills, "f-2", 3, 13)},
			quantity: 20, executed: 7, remaining: 13,
		},
		{
			name:     "amended up after a partial_fill update",
			msgs:     []string{update(10, 0, "new_placed_order_by_user"), fill(wsFuturesFeedFills, "f-1", 4, 6), update(10, 4, "partial_fill"), update(20, 4, "edited_by_user"), fill(wsFuturesFeedFills, "f-2", 3, 13)},
			quantity: 20, executed: 7, remaining: 13,
		},
		{
			name:     "amended down",
			msgs:     []string{update(20, 0, "new_placed_order_by_user"), fill(wsFuturesFeedFills, "f-1", 4, 16), update(10, 4, "edited_by_user"), fill(wsFuturesFeedFills, "f-2", 3, 3)},
			quantity: 10, executed: 7, remaining: 3,
		},
		{
			name:     "open_orders snapshot",
			msgs:     []string{update(10, 0, "new_placed_order_by_user"), fill(wsFuturesFeedFills, "f-1", 4, 6), snapshot(4), fill(wsFuturesFeedFills, "f-2", 3, 3)},
			quantity: 10, executed: 7, remaining: 3,
		},
		{
			name:     "fills snapshot before the open_orders snapshot",
			msgs:     []string{fill(wsFuturesFeedFillsSnapshot, "f-1", 4, 6), snapshot(4), fill(wsFuturesFeedFills, "f-2", 3, 3)},
			quantity: 10, executed: 7, remaining: 3,
		},
		{
			name:     "fills snapshot repeating the first fill after its partial_fill update",
			msgs:     []string{update(10, 0, "new_placed_order_by_user"), fill(wsFuturesFeedFills, "f-1", 4, 6), update(10, 4, "partial_fill"), fill(wsFuturesFeedFillsSnapshot, "f-1", 4, 6), fill(wsFuturesFeedFills, "f-2", 3, 3)},
			quantity: 10, executed: 7, remaining: 3,
		},
		{
			name:     "fractional quantities",
			msgs:     []string{update(1, 0, "new_placed_order_by_user"), fill(wsFuturesFeedFills, "f-1", 0.5, 0.5), update(1, 0.5, "partial_fill"), fill(wsFuturesFeedFills, "f-2", 0.25, 0.25)},
			quantity: 1, executed: 0.75, remaining: 0.25,
		},
		{
			name:     "open_orders snapshot reported after both fills",
			msgs:     []string{update(10, 0, "new_placed_order_by_user"), fill(wsFuturesFeedFills, "f-1", 4, 6), fill(wsFuturesFeedFills, "f-2", 3, 3), snapshot(4)},
			quantity: 10, executed: 7, remaining: 3,
		},
		{
			name:     "placement reported after both fills",
			msgs:     []string{fill(wsFuturesFeedFills, "f-1", 4, 6), fill(wsFuturesFeedFills, "f-2", 3, 3), update(10, 0, "new_placed_order_by_user")},
			quantity: 10, executed: 7, remaining: 3,
		},
		{
			name:     "amended up, reported after both fills",
			msgs:     []string{update(10, 0, "new_placed_order_by_user"), fill(wsFuturesFeedFills, "f-1", 4, 6), fill(wsFuturesFeedFills, "f-2", 3, 13), update(20, 4, "edited_by_user")},
			quantity: 20, executed: 7, remaining: 13,
		},
		{
			name:     "amended down, reported after both fills",
			msgs:     []string{update(20, 0, "new_placed_order_by_user"), fill(wsFuturesFeedFills, "f-1", 4, 16), fill(wsFuturesFeedFills, "f-2", 3, 3), update(10, 4, "edited_by_user")},
			quantity: 10, executed: 7, remaining: 3,
		},
		{
			name:     "partial_fill update reported before the second fill",
			msgs:     []string{update(10, 0, "new_placed_order_by_user"), fill(wsFuturesFeedFills, "f-1", 4, 6), update(10, 7, "partial_fill"), fill(wsFuturesFeedFills, "f-2", 3, 3)},
			quantity: 10, executed: 7, remaining: 3,
		},
		{
			name:     "open_orders snapshot of nothing filled reported after the first fill",
			msgs:     []string{fill(wsFuturesFeedFills, "f-1", 4, 6), snapshot(0), fill(wsFuturesFeedFills, "f-2", 3, 3)},
			quantity: 10, executed: 7, remaining: 3,
		},
		{
			name:     "fills snapshot repeating the first fill after the second",
			msgs:     []string{update(10, 0, "new_placed_order_by_user"), fill(wsFuturesFeedFills, "f-1", 4, 6), update(10, 4, "partial_fill"), fill(wsFuturesFeedFills, "f-2", 3, 3), fill(wsFuturesFeedFillsSnapshot, "f-1", 4, 6)},
			quantity: 10, executed: 7, remaining: 3,
		},
		{
			name:     "fills snapshots holding each fill",
			msgs:     []string{update(10, 0, "new_placed_order_by_user"), fill(wsFuturesFeedFillsSnapshot, "f-1", 4, 6), update(10, 4, "partial_fill"), fill(wsFuturesFeedFillsSnapshot, "f-2", 3, 3)},
			quantity: 10, executed: 7, remaining: 3,
		},
		{
			name:     "fills snapshot repeating the first fill after an open_orders snapshot",
			msgs:     []string{update(10, 0, "new_placed_order_by_user"), fill(wsFuturesFeedFills, "f-1", 4, 6), snapshot(4), fill(wsFuturesFeedFillsSnapshot, "f-1", 4, 6), fill(wsFuturesFeedFills, "f-2", 3, 3)},
			quantity: 10, executed: 7, remaining: 3,
		},
		{
			name:     "fills snapshot repeating the first fill after its placement is reported",
			msgs:     []string{fill(wsFuturesFeedFills, "f-1", 4, 6), update(10, 0, "new_placed_order_by_user"), fill(wsFuturesFeedFillsSnapshot, "f-1", 4, 6), fill(wsFuturesFeedFills, "f-2", 3, 3)},
			quantity: 10, executed: 7, remaining: 3,
		},
		{
			name:     "fills snapshot holding the fill an open_orders snapshot reported, after an amendment",
			msgs:     []string{snapshot(4), update(20, 4, "edited_by_user"), fill(wsFuturesFeedFillsSnapshot, "f-1", 4, 6), fill(wsFuturesFeedFills, "f-2", 3, 13)},
			quantity: 20, executed: 7, remaining: 13,
		},
		{
			name:     "amended up between the fills, which add up to the quantity reported before, reported after both",
			msgs:     []string{update(10, 0, "new_placed_order_by_user"), fill(wsFuturesFeedFills, "f-1", 4, 6), fill(wsFuturesFeedFills, "f-2", 6, 10), update(20, 4, "edited_by_user")},
			quantity: 20, executed: 10, remaining: 10,
		},
		{
			name:     "amended up between the fills, reported after both and after a partial_fill update of the first",
			msgs:     []string{update(10, 0, "new_placed_order_by_user"), fill(wsFuturesFeedFills, "f-1", 4, 6), fill(wsFuturesFeedFills, "f-2", 8, 8), update(10, 4, "partial_fill"), update(20, 4, "edited_by_user")},
			quantity: 20, executed: 12, remaining: 8,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestExchange(t)
			seedTestAssetNames(ex)
			details := futuresOrderUpdates(t, ex, tc.msgs...)
			require.Len(t, details, len(tc.msgs), "each message must relay an order update")
			last := details[len(details)-1]
			assert.Equal(t, tc.quantity, last.Amount, "the last update should relay the order's quantity")
			assert.Equal(t, tc.executed, last.ExecutedAmount, "the last update should relay both fills executed")
			assert.Equal(t, tc.remaining, last.RemainingAmount, "the last update should relay what remains after both fills")
			stored := details[0]
			mergeOrderUpdates(t, stored, details[1:])
			assert.Equal(t, order.PartiallyFilled, stored.Status, "the order should be stored partly filled")

			cancellation := futuresOrderUpdates(t, ex, futuresRemovalMessage("o-50", "cancelled_by_user"))
			require.Len(t, cancellation, 1, "the cancellation must relay an order update")
			exp := &order.Detail{
				Exchange:        ex.Name,
				AssetType:       asset.Futures,
				OrderID:         "o-50",
				Status:          order.PartiallyFilledCancelled,
				Amount:          tc.quantity,
				ExecutedAmount:  tc.executed,
				RemainingAmount: tc.remaining,
			}
			assert.Equal(t, exp, cancellation[0], "a cancellation after both fills should relay the order partly filled, with both fills executed")
		})
	}
}

func TestFuturesOrderFillsCountedAcrossRemoval(t *testing.T) {
	t.Parallel()
	fill := func(feed, fillID string, quantity, remaining float64) string {
		return futuresFillMessage(feed, "o-53", fillID, "PF_XBTUSD", quantity, remaining, 20000, 0.1, "USD")
	}
	for _, tc := range []struct {
		name   string
		reason string
		status order.Status
		// reported has open_orders report 7 filled before the removal, ahead of the fills feed reporting the second fill
		reported bool
	}{
		{name: "cancelled", reason: "cancelled_by_user", status: order.PartiallyFilledCancelled},
		{name: "cancelled after open_orders reported both fills", reason: "cancelled_by_user", status: order.PartiallyFilledCancelled, reported: true},
		{name: "expired", reason: "contract_expired", status: order.Expired},
		{name: "expired after open_orders reported both fills", reason: "contract_expired", status: order.Expired, reported: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestExchange(t)
			seedTestAssetNames(ex)
			futuresOrderUpdates(t, ex, futuresOpenOrderMessage("o-53", 10, 0, "new_placed_order_by_user"), fill(wsFuturesFeedFills, "f-1", 4, 6))
			executed := 4.0
			if tc.reported {
				futuresOrderUpdates(t, ex, futuresOpenOrderMessage("o-53", 10, 7, "partial_fill"))
				executed = 7
			}
			removal := futuresOrderUpdates(t, ex, futuresRemovalMessage("o-53", tc.reason))
			require.Len(t, removal, 1, "the removal must relay an order update")
			exp := &order.Detail{
				Exchange:        ex.Name,
				AssetType:       asset.Futures,
				OrderID:         "o-53",
				Status:          tc.status,
				Amount:          10,
				ExecutedAmount:  executed,
				RemainingAmount: 10 - executed,
			}
			assert.Equal(t, exp, removal[0], "the removal should relay the fills counted, or what open_orders reported filled if more")

			late := futuresOrderUpdates(t, ex,
				fill(wsFuturesFeedFills, "f-2", 3, 3),
				fill(wsFuturesFeedFillsSnapshot, "f-1", 4, 6),
				fill(wsFuturesFeedFillsSnapshot, "f-2", 3, 3),
			)
			late = append(late, futuresOrderUpdatesOn(t, ex, &futuresTestConn{name: "new"}, futuresEmptySnapshot, fill(wsFuturesFeedFillsSnapshot, "f-1", 4, 6))...)
			assert.Len(t, late, 4, "each fill after the removal, on its connection or a new one, should relay an order update")
			for i, d := range late {
				assert.Equalf(t, tc.status, d.Status, "fill %d after the removal should relay the order as it was removed", i)
				assert.Equalf(t, 7.0, d.ExecutedAmount, "fill %d after the removal should relay both fills executed, each counted once", i)
				assert.Equalf(t, 3.0, d.RemainingAmount, "fill %d after the removal should relay what remained after both fills", i)
			}
		})
	}
}

func TestFuturesOrderFillsCountedAcrossResubscriptions(t *testing.T) {
	t.Parallel()
	snapshot := `{"feed":"open_orders_snapshot","account":"DemoUser","orders":[` + futuresOpenOrder("o-52", 10, 4) + `]}`
	repeated := futuresFillMessage(wsFuturesFeedFillsSnapshot, "o-52", "f-1", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD")
	for _, tc := range []struct {
		name string
		// unsubscribed are the feeds unsubscribed and subscribed again between the fills, none for a new connection, and
		// resubscribed are the snapshots reported after them
		unsubscribed []string
		resubscribed []string
	}{
		{name: "new connection, its fills snapshot repeating the first fill", resubscribed: []string{snapshot, repeated}},
		{name: "new connection, its fills snapshot leaving out the first fill", resubscribed: []string{snapshot, futuresEmptyFillsSnapshot}},
		{name: "open_orders resubscribed", unsubscribed: []string{wsFuturesFeedOpenOrders}, resubscribed: []string{snapshot}},
		{name: "fills resubscribed, its snapshot repeating the first fill", unsubscribed: []string{wsFuturesFeedFills}, resubscribed: []string{repeated}},
		{name: "fills resubscribed, its snapshot leaving out the first fill", unsubscribed: []string{wsFuturesFeedFills}, resubscribed: []string{futuresEmptyFillsSnapshot}},
		{name: "both resubscribed, the fills snapshot repeating the first fill", unsubscribed: []string{wsFuturesFeedOpenOrders, wsFuturesFeedFills}, resubscribed: []string{snapshot, repeated}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, conn := newFuturesTestExchange(t, new(mockFuturesServer))
			seedTestAssetNames(ex)
			subs := map[string]*subscription.Subscription{
				wsFuturesFeedOpenOrders: futuresSubscription(subscription.MyOrdersChannel, wsFuturesFeedOpenOrders),
				wsFuturesFeedFills:      futuresSubscription(subscription.MyTradesChannel, wsFuturesFeedFills),
			}
			require.NoError(t, ex.futuresSubscribe(t.Context(), conn, subscription.List{subs[wsFuturesFeedOpenOrders], subs[wsFuturesFeedFills]}), "futuresSubscribe must not error")
			futuresOrderUpdatesOn(t, ex, conn,
				futuresEmptySnapshot,
				futuresEmptyFillsSnapshot,
				futuresOpenOrderMessage("o-52", 10, 0, "new_placed_order_by_user"),
				futuresFillMessage(wsFuturesFeedFills, "o-52", "f-1", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD"),
				futuresOpenOrderMessage("o-52", 10, 4, "partial_fill"),
			)
			on := conn
			if tc.unsubscribed == nil {
				on = &futuresTestConn{name: "new"}
			}
			for _, feed := range tc.unsubscribed {
				require.NoError(t, ex.futuresUnsubscribe(t.Context(), conn, subscription.List{subs[feed]}), "futuresUnsubscribe must not error")
			}
			for _, feed := range tc.unsubscribed {
				require.NoError(t, ex.futuresSubscribe(t.Context(), conn, subscription.List{subs[feed]}), "futuresSubscribe must not error subscribing again")
			}
			futuresOrderUpdatesOn(t, ex, on, tc.resubscribed...)
			details := futuresOrderUpdatesOn(t, ex, on,
				futuresFillMessage(wsFuturesFeedFills, "o-52", "f-2", "PF_XBTUSD", 3, 3, 20000, 0.15, "USD"),
				futuresRemovalMessage("o-52", "cancelled_by_user"),
			)
			require.Len(t, details, 2, "the fill and the cancellation must each relay an order update")
			assert.Equal(t, 7.0, details[0].ExecutedAmount, "a fill after the feeds were subscribed again should add to the fills counted before")
			assert.Equal(t, 3.0, details[0].RemainingAmount, "a fill after the feeds were subscribed again should relay what remains after both fills")
			exp := &order.Detail{
				Exchange:        ex.Name,
				AssetType:       asset.Futures,
				OrderID:         "o-52",
				Status:          order.PartiallyFilledCancelled,
				Amount:          10,
				ExecutedAmount:  7,
				RemainingAmount: 3,
			}
			assert.Equal(t, exp, details[1], "a cancellation after the feeds were subscribed again should relay the order partly filled, with both fills executed")
		})
	}
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
	assertFuturesOrderTotalsUnset(t, stored, "an order filled after its cancellation")
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

	early := futuresOrderUpdates(t, ex,
		futuresOpenOrderMessage("o-60", 10, 0, "new_placed_order_by_user"),
		futuresRemovalMessage("o-60", "stop_order_triggered"),
		futuresFillMessage(wsFuturesFeedFills, "o-60", "f-1", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD"),
		futuresOpenOrderMessage("o-60", 10, 0, "limit_order_from_stop"),
		futuresFillMessage(wsFuturesFeedFills, "o-60", "f-2", "PF_XBTUSD", 3, 3, 20000, 0.15, "USD"),
	)
	require.Len(t, early, 5, "each message must relay an order update")
	assert.Equal(t, 4.0, early[3].ExecutedAmount, "the order a stop placed, reported open after its fill, should relay that fill executed")
	assert.Equal(t, 7.0, early[4].ExecutedAmount, "a later fill should add to the fill reported before the order a stop placed was reported open")

	// The order a stop places can fill and be removed without open_orders reporting it open
	market := futuresOrderUpdates(t, ex,
		futuresOpenOrderMessage("o-61", 10, 0, "new_placed_order_by_user"),
		futuresRemovalMessage("o-61", "stop_order_triggered"),
		futuresFillMessage(wsFuturesFeedFills, "o-61", "f-1", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD"),
		futuresRemovalMessage("o-61", "full_fill"),
		futuresFillMessage(wsFuturesFeedFills, "o-61", "f-2", "PF_XBTUSD", 6, 0, 20000, 0.3, "USD"),
	)
	require.Len(t, market, 5, "each message must relay an order update")
	for i, d := range market[3:] {
		assert.Equalf(t, order.Filled, d.Status, "update %d from the removal as filled on should relay the order filled", i)
		assert.Equalf(t, 10.0, d.ExecutedAmount, "update %d from the removal as filled on should relay the order's quantity executed", i)
	}
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
	assert.Equal(t, 16.0, details[2].RemainingAmount, "the amendment should relay what remains of the amended quantity")
	assert.Equal(t, 16.0, details[3].RemainingAmount, "a fill should relay what remains of the amended quantity, not what remained when it filled")
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
	assert.Equal(t, 4.0, details[2].ExecutedAmount, "a repeated fill should not be counted again")
	assert.Equal(t, order.Filled, details[3].Status, "the fill leaving nothing remaining should relay the order filled")
	assert.Equal(t, 10.0, details[3].ExecutedAmount, "the live fill should relay the order filled in full")

	// The fills snapshot can come before the open orders snapshot
	early := futuresOrderUpdates(t, ex,
		futuresFillMessage(wsFuturesFeedFillsSnapshot, "o-9", "f-3", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD"),
		`{"feed":"open_orders_snapshot","account":"DemoUser","orders":[`+futuresOpenOrder("o-9", 10, 4)+`]}`,
	)
	require.Len(t, early, 2, "the fill and the snapshot must relay an order update")
	assert.Zero(t, early[0].ExecutedAmount, "a fill of an order not yet reported should relay its trade alone")

	missing := futuresOrderUpdates(t, ex,
		`{"feed":"open_orders_snapshot","account":"DemoUser","orders":[`+futuresOpenOrder("o-10", 10, 4)+`]}`,
		futuresFillMessage(wsFuturesFeedFills, "o-10", "f-4", "PF_XBTUSD", 6, 0, 20000, 0.3, "USD"),
	)
	require.Len(t, missing, 2, "the snapshot and the fill must relay an order update")
	assert.Equal(t, 10.0, missing[1].ExecutedAmount, "a fill leaving nothing remaining should relay the order filled in full")
	assertFuturesOrderTotalsUnset(t, missing[1], "an order whose earlier fill was not reported")

	// A live fill can be reported before open_orders reports it, when only the fills counted show what has executed
	ahead := futuresOrderUpdates(t, ex,
		`{"feed":"open_orders_snapshot","account":"DemoUser","orders":[`+futuresOpenOrder("o-41", 10, 4)+`]}`,
		futuresFillMessage(wsFuturesFeedFillsSnapshot, "o-41", "f-5", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD"),
		futuresFillMessage(wsFuturesFeedFills, "o-41", "f-6", "PF_XBTUSD", 3, 3, 20000, 0.15, "USD"),
	)
	require.Len(t, ahead, 3, "the snapshot and each fill must relay an order update")
	assert.Equal(t, 7.0, ahead[2].ExecutedAmount, "a live fill reported before open_orders reports it should add to the fills snapshot's fill")
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

func TestFuturesOrderFilledBeforeOpenOrdersUnsubscribed(t *testing.T) {
	t.Parallel()
	ex, conn := newFuturesTestExchange(t, new(mockFuturesServer))
	seedTestAssetNames(ex)
	orders := futuresSubscription(subscription.MyOrdersChannel, wsFuturesFeedOpenOrders)
	require.NoError(t, ex.futuresSubscribe(t.Context(), conn, subscription.List{orders, futuresSubscription(subscription.MyTradesChannel, wsFuturesFeedFills)}), "futuresSubscribe must not error")
	last := futuresFillMessage(wsFuturesFeedFills, "o-62", "f-2", "PF_XBTUSD", 6, 0, 20000, 0.3, "USD")
	futuresOrderUpdatesOn(t, ex, conn,
		futuresEmptySnapshot,
		futuresEmptyFillsSnapshot,
		futuresOpenOrderMessage("o-62", 10, 0, "new_placed_order_by_user"),
		futuresFillMessage(wsFuturesFeedFills, "o-62", "f-1", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD"),
		last,
	)
	require.NoError(t, ex.futuresUnsubscribe(t.Context(), conn, subscription.List{orders}), "futuresUnsubscribe must not error")
	repeated := futuresOrderUpdatesOn(t, ex, conn, last)
	require.Len(t, repeated, 1, "the repeated fill must relay an order update")
	assert.Equal(t, order.Filled, repeated[0].Status, "a fill repeated after open_orders was unsubscribed should relay the order its fills filled in full")
	assert.Equal(t, 10.0, repeated[0].ExecutedAmount, "a fill repeated after open_orders was unsubscribed should relay every fill of the order executed")
	assert.Zero(t, repeated[0].RemainingAmount, "a fill repeated after open_orders was unsubscribed should relay nothing remaining")
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
	// An inverse contract's quantity is in US dollars, and its fees are charged in bitcoin
	assert.Equal(t, 2000.0, details[1].ExecutedAmount, "the fill leaving nothing remaining should relay the order filled in full")
	assertFuturesOrderTotalsUnset(t, details[1], "an inverse contract's order")
	for i, fee := range []float64{0.00001, 0.00002} {
		require.Lenf(t, details[i].Trades, 1, "fill %d must relay its trade", i)
		assert.Equalf(t, fee, details[i].Trades[0].Fee, "fill %d's trade should hold its fee", i)
		assert.Equalf(t, "XBT", details[i].Trades[0].FeeAsset, "fill %d's trade should hold its fee currency by its alternative name", i)
	}
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
	assert.Equal(t, 0.1, details[1].ExecutedAmount, "a fill leaving some of the order remaining should relay what it executed")
	assert.Equal(t, 0.8, details[2].ExecutedAmount, "a fill leaving nothing remaining should relay the order's quantity as executed")
}

func TestFuturesOrderFillBeyondReportedQuantity(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	// Fills can follow an amendment open_orders has yet to report, executing more than the quantity it last reported
	details := futuresOrderUpdates(t, ex,
		futuresOpenOrderMessage("o-42", 10, 0, "new_placed_order_by_user"),
		futuresFillMessage(wsFuturesFeedFills, "o-42", "f-1", "PF_XBTUSD", 12, 1, 20000, 0.6, "USD"),
		futuresFillMessage(wsFuturesFeedFills, "o-42", "f-2", "PF_XBTUSD", 1, 0, 20000, 0.05, "USD"),
		futuresRemovalMessage("o-42", "full_fill"),
	)
	require.Len(t, details, 4, "each message must relay an order update")
	assert.Equal(t, 12.0, details[1].ExecutedAmount, "a fill should relay the fills counted, beyond the quantity open_orders last reported")
	assert.Zero(t, details[1].RemainingAmount, "a fill beyond the quantity open_orders last reported should relay no remaining amount rather than a negative one")
	assert.Equal(t, order.UnknownStatus, details[1].Status, "a fill beyond the quantity open_orders last reported should leave the status to open_orders while some of the order remains")
	assert.Equal(t, order.Filled, details[2].Status, "the fill leaving nothing remaining should relay the order filled")
	assert.Equal(t, 13.0, details[2].ExecutedAmount, "the fill leaving nothing remaining should relay all the order's fills executed")
	assert.Zero(t, details[2].RemainingAmount, "the fill leaving nothing remaining should relay no remaining amount")
	assert.Equal(t, 13.0, details[3].ExecutedAmount, "a removal as filled should relay all the order's fills executed, beyond the quantity open_orders last reported")
	stored := details[0]
	mergeOrderUpdates(t, stored, details[1:])
	assert.Equal(t, order.Filled, stored.Status, "the order should be stored filled")
	assert.Equal(t, 13.0, stored.ExecutedAmount, "the order should be stored with all its fills executed")
	assert.Zero(t, stored.RemainingAmount, "an order filled beyond the quantity open_orders last reported should be stored with nothing remaining")

	single := futuresOrderUpdates(t, ex,
		futuresOpenOrderMessage("o-43", 10, 0, "new_placed_order_by_user"),
		futuresFillMessage(wsFuturesFeedFills, "o-43", "f-3", "PF_XBTUSD", 12, 0, 20000, 0.6, "USD"),
	)
	require.Len(t, single, 2, "each message must relay an order update")
	assert.Equal(t, order.Filled, single[1].Status, "a single fill leaving nothing remaining should relay the order filled")
	assert.Equal(t, 12.0, single[1].ExecutedAmount, "a single fill should relay all it executed, beyond the quantity open_orders last reported")
	assert.Zero(t, single[1].RemainingAmount, "a single fill beyond the quantity open_orders last reported should relay no remaining amount")

	partlyFilled := futuresOrderUpdates(t, ex,
		futuresOpenOrderMessage("o-44", 10, 0, "new_placed_order_by_user"),
		futuresFillMessage(wsFuturesFeedFills, "o-44", "f-4", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD"),
		futuresOpenOrderMessage("o-44", 10, 4, "partial_fill"),
		futuresFillMessage(wsFuturesFeedFills, "o-44", "f-5", "PF_XBTUSD", 8, 1, 20000, 0.4, "USD"),
	)
	require.Len(t, partlyFilled, 4, "each message must relay an order update")
	assert.Equal(t, 12.0, partlyFilled[3].ExecutedAmount, "a fill beyond the quantity open_orders last reported filled in part should relay the fills counted")
	assert.Zero(t, partlyFilled[3].RemainingAmount, "a fill beyond the quantity open_orders last reported filled in part should relay no remaining amount")
	assert.Equal(t, order.UnknownStatus, partlyFilled[3].Status, "a fill beyond the quantity open_orders last reported filled in part should leave the status to open_orders while some of the order remains")

	removedFirst := futuresOrderUpdates(t, ex,
		futuresOpenOrderMessage("o-48", 10, 0, "new_placed_order_by_user"),
		futuresFillMessage(wsFuturesFeedFills, "o-48", "f-6", "PF_XBTUSD", 12, 1, 20000, 0.6, "USD"),
		futuresRemovalMessage("o-48", "full_fill"),
		futuresFillMessage(wsFuturesFeedFills, "o-48", "f-7", "PF_XBTUSD", 1, 0, 20000, 0.05, "USD"),
	)
	require.Len(t, removedFirst, 4, "each message must relay an order update")
	assert.Equal(t, 13.0, removedFirst[3].ExecutedAmount, "a fill reported after the removal as filled should add to the fills counted, beyond the quantity open_orders last reported")
}

func TestFuturesOrderFillReportedAfterTheLast(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	details := futuresOrderUpdates(t, ex,
		futuresOpenOrderMessage("o-45", 10, 0, "new_placed_order_by_user"),
		futuresFillMessage(wsFuturesFeedFills, "o-45", "f-2", "PF_XBTUSD", 6, 0, 20000, 0.3, "USD"),
		futuresOpenOrderMessage("o-45", 10, 4, "partial_fill"),
		futuresFillMessage(wsFuturesFeedFillsSnapshot, "o-45", "f-1", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD"),
	)
	require.Len(t, details, 4, "each message must relay an order update")
	for _, i := range []int{1, 3} {
		assert.Equalf(t, order.Filled, details[i].Status, "update %d, a fill, should relay the order filled once a fill has left nothing of it remaining", i)
		assert.Equalf(t, 10.0, details[i].ExecutedAmount, "update %d, a fill, should relay the order's quantity as executed", i)
		assert.Zerof(t, details[i].RemainingAmount, "update %d, a fill, should relay nothing remaining", i)
	}
}

func TestFuturesOrderFillsOfOrdersInOneMessage(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	fills := `{"feed":"fills","account":"DemoUser","fills":[` + futuresFill("o-46", "f-1", "PF_XBTUSD", 4, 6, 20000, 0.2, "USD") + `,` +
		futuresFill("o-47", "f-2", "PF_XBTUSD", 3, 17, 20000, 0.15, "USD") + `,` + futuresFill("o-46", "f-3", "PF_XBTUSD", 6, 0, 20000, 0.3, "USD") + `]}`
	details := futuresOrderUpdates(t, ex,
		futuresOpenOrderMessage("o-46", 10, 0, "new_placed_order_by_user"),
		futuresOpenOrderMessage("o-47", 20, 0, "new_placed_order_by_user"),
		fills,
	)
	require.Len(t, details, 5, "each placement and each fill must relay an order update")
	type quantities struct {
		orderID                     string
		amount, executed, remaining float64
	}
	got := make([]quantities, 0, len(details)-2)
	for _, d := range details[2:] {
		got = append(got, quantities{d.OrderID, d.Amount, d.ExecutedAmount, d.RemainingAmount})
	}
	assert.Equal(t, []quantities{{"o-46", 10, 4, 6}, {"o-47", 20, 3, 17}, {"o-46", 10, 10, 0}}, got, "each fill in a message should relay its own order's quantities, after the fills before it")
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
	tracker.fill(nil, &WebsocketFuturesFill{Instrument: "PF_XBTUSD", OrderID: "filled only", FillID: "f-1", Quantity: 1, RemainingOrderQuantity: 1}, true)
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
		return tracker.fill(nil, &WebsocketFuturesFill{Instrument: "PF_XBTUSD", OrderID: "repeated", FillID: fillID, Quantity: 1, RemainingOrderQuantity: 1}, live)
	}
	require.True(t, fill("f-2", true).liveOnly, "a fill reported as it happened on a covered connection must be counted live")
	assert.False(t, fill("f-3", false).liveOnly, "a fill not reported as it happened should end an order's live fills")

	// An order an open_orders snapshot holds can have filled before the fills feed's snapshot
	tracker.snapshot(nil, []WebsocketFuturesOpenOrder{open("listed")})
	assert.False(t, tracker.get("listed").liveOnly, "a snapshot should not take an order it holds as having all its fills reported as they happened")
}

// futuresTrackerCoverage is what an order tracker holds of the connection the feeds last reported on, and the quantity
// it holds of o-1, the order coveredFuturesOrderTracker records
type futuresTrackerCoverage struct {
	conn                                  websocket.Connection
	ordersSnapshotSeen, fillsSnapshotSeen bool
	quantity                              float64
}

// trackerCoverage returns what a tracker holds of the connection the feeds last reported on, and of o-1's quantity. The
// caller holds the tracker's lock
func trackerCoverage(tracker *futuresOrderTracker) futuresTrackerCoverage {
	c := futuresTrackerCoverage{conn: tracker.conn, ordersSnapshotSeen: tracker.ordersSnapshotSeen, fillsSnapshotSeen: tracker.fillsSnapshotSeen}
	if s, ok := tracker.orders["o-1"]; ok {
		c.quantity = s.quantity
	}
	return c
}

// coveredFuturesOrderTracker returns a tracker both feeds' snapshots covered on conn, the open_orders snapshot holding
// o-1, an open order of 10
func coveredFuturesOrderTracker(conn websocket.Connection) *futuresOrderTracker {
	tracker := new(futuresOrderTracker)
	tracker.snapshot(conn, []WebsocketFuturesOpenOrder{{Instrument: "PF_XBTUSD", Quantity: 10, OrderID: "o-1"}})
	tracker.fillsSnapshot(conn)
	return tracker
}

func TestFuturesOrderTrackerReportedOn(t *testing.T) {
	t.Parallel()
	first, second := &futuresTestConn{name: "first"}, &futuresTestConn{name: "second"}
	tracker := coveredFuturesOrderTracker(first)
	tracker.mu.Lock()
	tracker.reportedOn(first)
	same := trackerCoverage(tracker)
	tracker.reportedOn(second)
	other := trackerCoverage(tracker)
	tracker.mu.Unlock()
	assert.Equal(t, futuresTrackerCoverage{first, true, true, 10}, same, "a report on the connection the feeds last reported on should keep its coverage and the open orders' quantities")
	assert.Equal(t, futuresTrackerCoverage{second, false, false, 0}, other, "a report on another connection should start its coverage, forgetting the open orders' quantities")
}

func TestFuturesOrderTrackerUnsubscribed(t *testing.T) {
	t.Parallel()
	first, second := &futuresTestConn{name: "first"}, &futuresTestConn{name: "second"}
	for _, tc := range []struct {
		name string
		conn websocket.Connection
		feed string
		exp  futuresTrackerCoverage
	}{
		{name: "open_orders", conn: first, feed: wsFuturesFeedOpenOrders, exp: futuresTrackerCoverage{conn: first, fillsSnapshotSeen: true}},
		{name: "fills", conn: first, feed: wsFuturesFeedFills, exp: futuresTrackerCoverage{conn: first, ordersSnapshotSeen: true, quantity: 10}},
		{name: "balances", conn: first, feed: wsFuturesFeedBalances, exp: futuresTrackerCoverage{conn: first, ordersSnapshotSeen: true, fillsSnapshotSeen: true, quantity: 10}},
		{name: "open_orders on another connection", conn: second, feed: wsFuturesFeedOpenOrders, exp: futuresTrackerCoverage{conn: first, ordersSnapshotSeen: true, fillsSnapshotSeen: true, quantity: 10}},
	} {
		tracker := coveredFuturesOrderTracker(first)
		tracker.unsubscribed(tc.conn, tc.feed)
		tracker.mu.Lock()
		got := trackerCoverage(tracker)
		tracker.mu.Unlock()
		assert.Equalf(t, tc.exp, got, "unsubscribing %s should leave the feeds' coverage and the open orders' quantities as expected", tc.name)
	}
}

func TestFuturesOrderTrackerForgetQuantities(t *testing.T) {
	t.Parallel()
	var tracker futuresOrderTracker
	open := func(orderID string) WebsocketFuturesOpenOrder {
		return WebsocketFuturesOpenOrder{Instrument: "PF_XBTUSD", Quantity: 10, Filled: 4, OrderID: orderID}
	}
	tracker.snapshot(nil, []WebsocketFuturesOpenOrder{open("open"), open("triggered"), open("filled"), open("cancelled")})
	tracker.finish(nil, "triggered", "stop_order_triggered")
	tracker.finish(nil, "filled", "full_fill")
	tracker.finish(nil, "cancelled", "cancelled_by_user")
	tracker.mu.Lock()
	tracker.forgetQuantities()
	got := make(map[string][2]float64, len(tracker.orders))
	for id, s := range tracker.orders {
		got[id] = [2]float64{s.quantity, s.filled}
	}
	tracker.mu.Unlock()
	// The order a stop places when it triggers takes its ID, so its quantity can still change; a finished order's cannot
	exp := map[string][2]float64{"open": {0, 4}, "triggered": {0, 4}, "filled": {10, 4}, "cancelled": {10, 4}}
	assert.Equal(t, exp, got, "forgetQuantities should forget the quantity of each order whose quantity can still change, keeping what filled")
}

func TestFuturesOrderTrackerState(t *testing.T) {
	t.Parallel()
	first, second := &futuresTestConn{name: "first"}, &futuresTestConn{name: "second"}
	tracker := coveredFuturesOrderTracker(first)
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	made := tracker.state("o-2", first)
	assert.Same(t, made, tracker.state("o-2", first), "state should return the record it made of an order")
	got := *made
	got.updated = time.Time{}
	assert.Equal(t, futuresOrderState{fillIDs: map[string]struct{}{}, liveOnly: true, fillsCoverage: 1}, got, "state should make the record of an order first reported on a covered connection as reported live")
	got = *tracker.state("o-3", second)
	got.updated = time.Time{}
	assert.Equal(t, futuresOrderState{fillIDs: map[string]struct{}{}, fillsCoverage: 1}, got, "state should make the record of an order first reported on a new connection as not reported live")
	assert.Equal(t, futuresTrackerCoverage{second, false, false, 0}, trackerCoverage(tracker), "a report on a new connection should start its coverage, forgetting the open orders' quantities")
}

func TestFuturesOrderQuantityCoverage(t *testing.T) {
	t.Parallel()
	snapshot := func(quantity, filled float64) string {
		return `{"feed":"open_orders_snapshot","account":"DemoUser","orders":[` + futuresOpenOrder("o-30", quantity, filled) + `]}`
	}
	lastFill := func(feed string, quantity float64) string {
		return futuresFillMessage(feed, "o-30", "f-last", "PF_XBTUSD", quantity, 0, 20000, 0.25, "USD")
	}
	initial := []string{snapshot(10, 0), futuresEmptyFillsSnapshot}
	// report is what the feeds report on the subscribed connection, or on a new connection after it
	type report struct {
		onNew bool
		msgs  []string
		// unsubscribe has open_orders unsubscribed on the subscribed connection after the messages
		unsubscribe bool
	}
	for _, tc := range []struct {
		name    string
		reports []report
		// filled is the order's last fill, the last message, which leaves nothing of the order remaining
		filled float64
		status order.Status
		// amount is the order's quantity and executed quantity the last fill relays, zero when it relays its trade alone
		amount float64
	}{
		// The order was amended from 10 to 5 and filled while the connection was down
		{
			name:    "fills snapshot first on a new connection",
			reports: []report{{msgs: initial}, {onNew: true, msgs: []string{lastFill(wsFuturesFeedFillsSnapshot, 5)}}},
			filled:  5,
		},
		{
			name:    "fill before the snapshots on a new connection",
			reports: []report{{msgs: initial}, {onNew: true, msgs: []string{lastFill(wsFuturesFeedFills, 5)}}},
			filled:  5,
		},
		{
			name:    "open_orders snapshot first on a new connection",
			reports: []report{{msgs: initial}, {onNew: true, msgs: []string{futuresEmptySnapshot, lastFill(wsFuturesFeedFillsSnapshot, 5)}}},
			filled:  5,
		},
		// The order was amended from 10 to 5, of which 2 filled, while the connection was down
		{
			name:    "listed again by an open_orders snapshot after the fills snapshot on a new connection",
			reports: []report{{msgs: initial}, {onNew: true, msgs: []string{futuresEmptyFillsSnapshot, snapshot(5, 2), lastFill(wsFuturesFeedFills, 3)}}},
			filled:  3,
			status:  order.Filled,
			amount:  5,
		},
		// The order was amended from 10 to 5 and filled while open_orders was unsubscribed
		{
			name:    "open_orders unsubscribed",
			reports: []report{{msgs: initial, unsubscribe: true}, {msgs: []string{lastFill(wsFuturesFeedFills, 5)}}},
			filled:  5,
		},
		// 4 of the order filled before the fills snapshot, which need not hold that fill
		{
			name:    "listed by an open_orders snapshot after the fills snapshot, then open_orders unsubscribed",
			reports: []report{{msgs: []string{futuresEmptyFillsSnapshot, snapshot(10, 4)}, unsubscribe: true}, {msgs: []string{lastFill(wsFuturesFeedFills, 6)}}},
			filled:  6,
		},
		{
			name:    "listed by an open_orders snapshot after the fills snapshot and amended, then open_orders unsubscribed",
			reports: []report{{msgs: []string{futuresEmptyFillsSnapshot, snapshot(10, 4), futuresOpenOrderMessage("o-30", 20, 4, "edited_by_user")}, unsubscribe: true}, {msgs: []string{lastFill(wsFuturesFeedFills, 16)}}},
			filled:  16,
		},
		// The stop triggered, and the order it placed under its ID was amended from 10 to 5 and filled while the
		// connection was down
		{
			name:    "stop removed as triggered",
			reports: []report{{msgs: append(slices.Clone(initial), futuresRemovalMessage("o-30", "stop_order_triggered"))}, {onNew: true, msgs: []string{lastFill(wsFuturesFeedFillsSnapshot, 5)}}},
			filled:  5,
			status:  order.Closed,
		},
		// A finished order's quantity cannot change, so the fills Kraken reports late count against it
		{
			name:    "filled",
			reports: []report{{msgs: append(slices.Clone(initial), futuresRemovalMessage("o-30", "full_fill"))}, {onNew: true, msgs: []string{lastFill(wsFuturesFeedFillsSnapshot, 6)}}},
			filled:  6,
			status:  order.Filled,
			amount:  10,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, conn := newFuturesTestExchange(t, new(mockFuturesServer))
			seedTestAssetNames(ex)
			orders := futuresSubscription(subscription.MyOrdersChannel, wsFuturesFeedOpenOrders)
			require.NoError(t, ex.futuresSubscribe(t.Context(), conn, subscription.List{orders, futuresSubscription(subscription.MyTradesChannel, wsFuturesFeedFills)}), "futuresSubscribe must not error")
			newConn := &futuresTestConn{name: "new"}
			var details []*order.Detail
			for _, r := range tc.reports {
				on := conn
				if r.onNew {
					on = newConn
				}
				details = futuresOrderUpdatesOn(t, ex, on, r.msgs...)
				if r.unsubscribe {
					require.NoError(t, ex.futuresUnsubscribe(t.Context(), conn, subscription.List{orders}), "futuresUnsubscribe must not error")
				}
			}
			require.NotEmpty(t, details, "the last fill must relay an order update")
			fillTime := time.UnixMilli(1600256966528)
			exp := &order.Detail{
				Exchange:       ex.Name,
				AssetType:      asset.Futures,
				OrderID:        "o-30",
				Pair:           futuresTestPair,
				Side:           order.Buy,
				Type:           order.Limit,
				Status:         tc.status,
				Amount:         tc.amount,
				ExecutedAmount: tc.amount,
				LastUpdated:    fillTime,
				Trades:         []order.TradeHistory{{TID: "f-last", Price: 20000, Amount: tc.filled, Fee: 0.25, FeeAsset: "USD", Exchange: ex.Name, Type: order.Limit, Side: order.Buy, Timestamp: fillTime}},
			}
			assert.Equal(t, exp, details[len(details)-1], "the last fill should relay the order's quantity only while open_orders reports the order's changes, or once it has finished")
		})
	}
}

func TestFuturesOrderRestingPastRetention(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	seedTestAssetNames(ex)
	futuresOrderUpdates(t, ex, `{"feed":"open_orders_snapshot","account":"DemoUser","orders":[`+futuresOpenOrder("resting", 10, 0)+`]}`)
	ex.futuresOrders.mu.Lock()
	ex.futuresOrders.orders["resting"].updated = time.Now().Add(-futuresOrderRetention - time.Second)
	ex.futuresOrders.mu.Unlock()
	// Another order's removal prunes the orders the feeds have not reported for futuresOrderRetention
	futuresOrderUpdates(t, ex, futuresRemovalMessage("other", "cancelled_by_user"))
	got := futuresOrderUpdates(t, ex, futuresRemovalMessage("resting", "full_fill"))
	require.Len(t, got, 1, "the removal must relay one order update")
	exp := &order.Detail{Exchange: ex.Name, AssetType: asset.Futures, OrderID: "resting", Status: order.Filled, Amount: 10, ExecutedAmount: 10}
	assert.Equal(t, exp, got[0], "an order resting past futuresOrderRetention should keep the quantity open_orders reported, and be relayed filled in full")
}

// setFuturesMarginCurrencies records the REST margin accounts' currencies for the default credentials, as
// UpdateAccountBalances does
func setFuturesMarginCurrencies(ex *Exchange, currencies map[string]currency.Code) {
	ex.futuresMarginCurrenciesMu.Lock()
	defer ex.futuresMarginCurrenciesMu.Unlock()
	ex.futuresMarginCurrencies = map[accounts.Credentials]futuresMarginRead{*ex.GetDefaultCredentials(): {currencies: currencies}}
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

func TestWebsocketFuturesMarginAccountFreeClamped(t *testing.T) {
	t.Parallel()
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		if !assert.Equal(t, "/derivatives/api/v3/accounts", r.URL.Path, "only the futures accounts should be requested") {
			http.Error(w, "unexpected request", http.StatusNotFound)
			return
		}
		_, err := w.Write([]byte(`{"result":"success","accounts":{"fi_xbtusd":{"type":"marginAccount","currency":"xbt","balances":{"xbt":"1"},` +
			`"auxiliary":{"af":0.5},"marginRequirements":{},"triggerEstimates":{}},"cash":{"type":"cashAccount","balances":{"usd":"100"}},` +
			`"flex":{"type":"multiCollateralMarginAccount","currencies":{}}},"serverTime":"2026-10-16T00:00:00.000Z"}`))
		assert.NoError(t, err, "Write should not error")
	})
	_, err := ex.UpdateAccountBalances(t.Context(), asset.Futures)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	for _, tc := range []struct {
		available float64
		exp       accounts.Balance
	}{
		{available: 1.2, exp: accounts.Balance{Currency: currency.XBT, Total: 0.75, Free: 0.75}},
		{available: -0.1, exp: accounts.Balance{Currency: currency.XBT, Total: 0.75, Hold: 0.75}},
	} {
		update := `{"feed":"balances","account":"DemoUser","futures":{"F-XBT:USD":{"name":"F-XBT:USD","pair":"XBT/USD","unit":"XBT","balance":0.75,"available":` +
			strconv.FormatFloat(tc.available, 'f', -1, 64) + `}},"timestamp":1640995200000,"seq":2}`
		require.NoErrorf(t, ex.wsFuturesHandleData(t.Context(), nil, []byte(update)), "wsFuturesHandleData must not error for available %v", tc.available)
		relayed(ex)
		b, err := ex.Accounts.GetBalance("fi_xbtusd", ex.GetDefaultCredentials(), asset.Futures, currency.XBT)
		require.NoError(t, err, "GetBalance must not error")
		b.UpdatedAt = time.Time{}
		assert.Equalf(t, tc.exp, b, "the free balance stored for available %v should stay within the balance", tc.available)
	}
}

// pausedCredentialsContext holds the first credentials lookup made with it until resume is called, closing reached as
// it starts, so a test can hold a balances message after it is numbered and before it is stored
type pausedCredentialsContext struct {
	context.Context       //nolint:containedctx // pausedCredentialsContext implements context.Context
	reached, resumed      chan struct{}
	pauseOnce, resumeOnce sync.Once
}

// newPausedCredentialsContext returns a pausedCredentialsContext of the test's context, resumed when the test ends
func newPausedCredentialsContext(t *testing.T) *pausedCredentialsContext {
	t.Helper()
	ctx := &pausedCredentialsContext{Context: t.Context(), reached: make(chan struct{}), resumed: make(chan struct{})}
	t.Cleanup(ctx.resume)
	return ctx
}

// resume releases the held credentials lookup
func (c *pausedCredentialsContext) resume() {
	c.resumeOnce.Do(func() { close(c.resumed) })
}

// Value holds the first credentials lookup until resume is called
func (c *pausedCredentialsContext) Value(key any) any {
	if key == accounts.ContextCredentialsFlag {
		c.pauseOnce.Do(func() {
			close(c.reached)
			<-c.resumed
		})
	}
	return c.Context.Value(key)
}

func TestWebsocketFuturesProcessBalancesKeepsNewerRead(t *testing.T) {
	t.Parallel()
	const (
		olderRead = `{"result":"success","accounts":{"cash":{"type":"cashAccount","balances":{"usd":"1000"}},` +
			`"flex":{"type":"multiCollateralMarginAccount","currencies":{}}},"serverTime":"2026-10-19T00:00:00.000Z"}`
		newerRead = `{"result":"success","accounts":{"cash":{"type":"cashAccount","balances":{"usd":"500"}},` +
			`"flex":{"type":"multiCollateralMarginAccount","currencies":{}}},"serverTime":"2026-10-19T00:00:01.000Z"}`
	)
	for _, feed := range []string{wsFuturesFeedBalancesSnapshot, wsFuturesFeedBalances} {
		t.Run(feed, func(t *testing.T) {
			t.Parallel()
			ex, readArrived, releaseRead := newHeldFuturesAccountsTestExchange(t, olderRead, newerRead)
			ctx := newPausedCredentialsContext(t)
			msg := `{"feed":"` + feed + `","account":"DemoUser","holding":{"USD":200},"timestamp":1640995200000,"seq":1}`
			stored := make(chan error, 1)
			go func() { stored <- ex.wsFuturesProcessBalances(ctx, feed, []byte(msg)) }()
			awaitClosed(t, ctx.reached, "the message must be numbered and held before its store")
			older := make(chan error, 1)
			go func() {
				_, err := ex.UpdateAccountBalances(t.Context(), asset.Futures)
				older <- err
			}()
			awaitClosed(t, readArrived, "the older read must reach the server")
			newer := make(chan error, 1)
			go func() {
				_, err := ex.UpdateAccountBalances(t.Context(), asset.Futures)
				newer <- err
			}()
			require.NoError(t, awaitError(t, newer, "the newer read must finish while the older one is held"), "the newer UpdateAccountBalances must not error")
			ctx.resume()
			require.NoError(t, awaitError(t, stored, "the message must be processed once resumed"), "wsFuturesProcessBalances must not error")
			expRelayed := []any{&WebsocketFuturesBalances{
				Feed:      feed,
				Account:   "DemoUser",
				Timestamp: types.Time(time.UnixMilli(1640995200000)),
				Sequence:  1,
				Holding:   map[string]float64{"USD": 200},
			}}
			assert.Equal(t, expRelayed, relayed(ex), "wsFuturesProcessBalances should relay the message alone when it stores nothing")
			releaseRead()
			require.NoError(t, awaitError(t, older, "the older read must finish once released"), "the older UpdateAccountBalances must not error")
			exp := map[string]accounts.CurrencyBalances{
				"cash": {currency.USD: {Currency: currency.USD, Total: 500, Free: 500}},
				"flex": {},
			}
			assert.Equal(t, exp, storedFuturesWallets(t, ex), "a message and a read requested before a newer read should not be stored over it")
		})
	}
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

	noCreds := newTestExchange(t)
	noCreds.SkipAuthCheck = false
	noCreds.API.AuthenticatedSupport = true
	noCreds.SetCredentials(&accounts.Credentials{})
	_, ok = noCreds.futuresMarginAccountName(t.Context(), &WebsocketFuturesMarginAccount{Name: "F-XBT:USD", Pair: "XBT/USD", Unit: "XBT"})
	assert.False(t, ok, "futuresMarginAccountName should match no account without credentials")
}

func TestSaveFuturesFeedBalances(t *testing.T) {
	t.Parallel()
	ex := newHTTPTestExchange(t, func(http.ResponseWriter, *http.Request) {
		assert.Fail(t, "saveFuturesFeedBalances should make no request")
	})
	noCreds := newTestExchange(t)
	noCreds.SkipAuthCheck = false
	noCreds.API.AuthenticatedSupport = true
	noCreds.SetCredentials(&accounts.Credentials{})
	subAccts, err := noCreds.saveFuturesFeedBalances(t.Context(), 1, true, nil)
	require.NoError(t, err, "saveFuturesFeedBalances must not error without wallets to store")
	assert.Nil(t, subAccts, "saveFuturesFeedBalances should store nothing without wallets")
	cash := func(balances accounts.CurrencyBalances) accounts.SubAccounts {
		return accounts.SubAccounts{{ID: "cash", AssetType: asset.Futures, Balances: balances}}
	}
	_, err = noCreds.saveFuturesFeedBalances(t.Context(), 2, true, cash(accounts.CurrencyBalances{}))
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "saveFuturesFeedBalances should reject missing credentials")

	require.NoError(t, ex.Accounts.Save(t.Context(), accounts.SubAccounts{
		{ID: "cash", AssetType: asset.Futures, Balances: accounts.CurrencyBalances{currency.USD: {Total: 1, Free: 1}, currency.EUR: {Total: 1, Free: 1}}},
		{ID: "flex", AssetType: asset.Futures, Balances: accounts.CurrencyBalances{currency.USD: {Total: 1, Free: 1}}},
		{ID: "fi_xbtusd", AssetType: asset.Futures, Balances: accounts.CurrencyBalances{currency.XBT: {Total: 1, Free: 1}, currency.XRP: {Total: 1, Free: 1}}},
	}, true), "Save must not error")
	subAccts, err = ex.saveFuturesFeedBalances(t.Context(), 10, true, accounts.SubAccounts{
		{ID: "cash", AssetType: asset.Futures, Balances: accounts.CurrencyBalances{currency.USD: {Currency: currency.USD, Total: 5, Free: 5}}},
		{ID: "flex", AssetType: asset.Futures, Balances: accounts.CurrencyBalances{}},
		{ID: "fi_xbtusd", AssetType: asset.Futures, Balances: accounts.CurrencyBalances{currency.XBT: {Currency: currency.XBT, Total: 0.75, Hold: 0.25, Free: 0.5}}},
	})
	require.NoError(t, err, "saveFuturesFeedBalances must not error for a snapshot")
	expReturned := map[string]accounts.CurrencyBalances{
		"cash":      {currency.USD: {Currency: currency.USD, Total: 5, Free: 5}},
		"flex":      {},
		"fi_xbtusd": {currency.XBT: {Currency: currency.XBT, Total: 0.75, Hold: 0.25, Free: 0.5}},
	}
	assert.Equal(t, expReturned, futuresWalletBalances(subAccts), "saveFuturesFeedBalances should return the wallets as stored")
	exp := map[string]accounts.CurrencyBalances{
		"cash": {currency.USD: {Currency: currency.USD, Total: 5, Free: 5}},
		"flex": {},
		"fi_xbtusd": {
			currency.XBT: {Currency: currency.XBT, Total: 0.75, Hold: 0.25, Free: 0.5},
			currency.XRP: {Currency: currency.XRP, Total: 1, Free: 1},
		},
	}
	assert.Equal(t, exp, storedFuturesWallets(t, ex), "a snapshot's cash and multi-collateral wallets should be stored whole, and a margin account a currency at a time")

	subAccts, err = ex.saveFuturesFeedBalances(t.Context(), 11, false, cash(accounts.CurrencyBalances{currency.EUR: {Currency: currency.EUR, Total: 2, Free: 2}}))
	require.NoError(t, err, "saveFuturesFeedBalances must not error for an update")
	assert.Equal(t, map[string]accounts.CurrencyBalances{"cash": {currency.EUR: {Currency: currency.EUR, Total: 2, Free: 2}}}, futuresWalletBalances(subAccts), "saveFuturesFeedBalances should return the balances an update stored")
	exp["cash"][currency.EUR] = accounts.Balance{Currency: currency.EUR, Total: 2, Free: 2}
	assert.Equal(t, exp, storedFuturesWallets(t, ex), "an update should store a wallet a currency at a time")

	subAccts, err = ex.saveFuturesFeedBalances(t.Context(), 9, false, cash(accounts.CurrencyBalances{currency.USD: {Currency: currency.USD, Total: 7, Free: 7}}))
	require.NoError(t, err, "saveFuturesFeedBalances must not error for an update that arrived before its wallet was stored whole")
	assert.Nil(t, subAccts, "saveFuturesFeedBalances should store nothing of an update that arrived before its wallet was stored whole")
	subAccts, err = ex.saveFuturesFeedBalances(t.Context(), 9, true, accounts.SubAccounts{
		{ID: "cash", AssetType: asset.Futures, Balances: accounts.CurrencyBalances{currency.USD: {Currency: currency.USD, Total: 8, Free: 8}}},
		{ID: "flex", AssetType: asset.Futures, Balances: accounts.CurrencyBalances{currency.USD: {Currency: currency.USD, Total: 8, Free: 8}}},
	})
	require.NoError(t, err, "saveFuturesFeedBalances must not error for a snapshot that arrived before its wallets were stored whole")
	assert.Nil(t, subAccts, "saveFuturesFeedBalances should store nothing of a snapshot that arrived before its wallets were stored whole")
	assert.Equal(t, exp, storedFuturesWallets(t, ex), "saveFuturesFeedBalances should leave balances stored after a message arrived as stored")

	subAccts, err = ex.saveFuturesFeedBalances(t.Context(), 10, false, cash(accounts.CurrencyBalances{
		currency.USD: {Currency: currency.USD, Total: 6, Free: 6},
		currency.EUR: {Currency: currency.EUR, Total: 9, Free: 9},
		currency.GBP: {Currency: currency.GBP, Total: 3, Free: 3},
	}))
	require.NoError(t, err, "saveFuturesFeedBalances must not error for an update")
	exp["cash"] = accounts.CurrencyBalances{
		currency.USD: {Currency: currency.USD, Total: 6, Free: 6},
		currency.EUR: {Currency: currency.EUR, Total: 2, Free: 2},
		currency.GBP: {Currency: currency.GBP, Total: 3, Free: 3},
	}
	assert.Equal(t, map[string]accounts.CurrencyBalances{"cash": {currency.USD: exp["cash"][currency.USD], currency.GBP: exp["cash"][currency.GBP]}}, futuresWalletBalances(subAccts),
		"saveFuturesFeedBalances should store an update's currencies except one stored on its own after it arrived")
	assert.Equal(t, exp, storedFuturesWallets(t, ex), "saveFuturesFeedBalances should keep a currency stored on its own after an update arrived")
	subAccts, err = ex.saveFuturesFeedBalances(t.Context(), 11, false, cash(accounts.CurrencyBalances{currency.EUR: {Currency: currency.EUR, Total: 4, Free: 4}}))
	require.NoError(t, err, "saveFuturesFeedBalances must not error for an update numbered as its currency's last store")
	exp["cash"][currency.EUR] = accounts.Balance{Currency: currency.EUR, Total: 4, Free: 4}
	assert.Equal(t, map[string]accounts.CurrencyBalances{"cash": {currency.EUR: exp["cash"][currency.EUR]}}, futuresWalletBalances(subAccts), "saveFuturesFeedBalances should store an update numbered as its currency's last store")

	subAccts, err = ex.saveFuturesFeedBalances(t.Context(), 10, true, cash(accounts.CurrencyBalances{currency.USD: {Currency: currency.USD, Total: 9, Free: 9}}))
	require.NoError(t, err, "saveFuturesFeedBalances must not error for a snapshot numbered as its wallet's last store")
	exp["cash"] = accounts.CurrencyBalances{
		currency.USD: {Currency: currency.USD, Total: 9, Free: 9},
		currency.EUR: {Currency: currency.EUR, Total: 4, Free: 4},
	}
	assert.Equal(t, map[string]accounts.CurrencyBalances{"cash": exp["cash"]}, futuresWalletBalances(subAccts), "saveFuturesFeedBalances should return a snapshot's wallet as stored")
	assert.Equal(t, exp, storedFuturesWallets(t, ex), "a snapshot should store its wallet whole, keeping a currency stored on its own after it arrived")

	unsupported := accounts.SubAccounts{{ID: "cash", AssetType: asset.Empty, Balances: accounts.CurrencyBalances{}}}
	_, err = ex.saveFuturesFeedBalances(t.Context(), 10, true, unsupported)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "saveFuturesFeedBalances should return the error reading a balance a snapshot keeps")
	_, err = ex.saveFuturesFeedBalances(t.Context(), 12, true, unsupported)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "saveFuturesFeedBalances should return the error storing a wallet whole")
	_, err = ex.saveFuturesFeedBalances(t.Context(), 12, false, cash(accounts.CurrencyBalances{currency.EMPTYCODE: {Total: 1, Free: 1}}))
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "saveFuturesFeedBalances should return the error storing a balance")
	expStored := map[string]*futuresWalletStores{
		"cash":      {whole: 10, currencies: map[currency.Code]uint64{currency.USD: 10, currency.EUR: 11, currency.GBP: 10}},
		"flex":      {whole: 10, currencies: map[currency.Code]uint64{}},
		"fi_xbtusd": {currencies: map[currency.Code]uint64{currency.XBT: 10}},
	}
	ex.futuresBalancesMu.Lock()
	assert.Equal(t, expStored, ex.futuresWalletsStored[*ex.GetDefaultCredentials()], "saveFuturesFeedBalances should record the message each store came from, and nothing it did not store")
	ex.futuresBalancesMu.Unlock()
}
