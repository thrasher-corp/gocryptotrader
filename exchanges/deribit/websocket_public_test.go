package deribit

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestWebsocketHandleOrderbook(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	perp := currency.NewPairWithDelimiter("BTC", "PERPETUAL", currency.DashDelimiter)
	const channel = "book.BTC-PERPETUAL.100ms"

	// Trimmed from a live snapshot, then a change following it
	require.NoError(t, ex.wsHandleOrderbook(t.Context(), channel, notification(channel, `{"timestamp":1792107433616,"type":"snapshot","change_id":178645851489,"instrument_name":"BTC-PERPETUAL","bids":[["new",82141.0,66900.0],["new",82140.5,10.0]],"asks":[["new",82141.5,59940.0],["new",82142.0,10.0]]}`)), "wsHandleOrderbook must not error for a snapshot")
	require.NoError(t, ex.wsHandleOrderbook(t.Context(), channel, notification(channel, `{"timestamp":1792107433716,"type":"change","prev_change_id":178645851489,"change_id":178645851490,"instrument_name":"BTC-PERPETUAL","bids":[["delete",82140.5,0.0]],"asks":[["change",82141.5,50000.0],["new",82143.0,1000.0]]}`)), "wsHandleOrderbook must not error for a change following the snapshot")
	book, err := ex.Websocket.Orderbook.GetOrderbook(perp, asset.Futures)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.Equal(t, orderbook.Levels{{Price: 82141, Amount: 66900}}, book.Bids, "wsHandleOrderbook should remove a deleted bid")
	assert.Equal(t, orderbook.Levels{{Price: 82141.5, Amount: 50000}, {Price: 82142, Amount: 10}, {Price: 82143, Amount: 1000}}, book.Asks, "wsHandleOrderbook should change and add asks")
	assert.Equal(t, int64(178645851490), book.LastUpdateID, "wsHandleOrderbook should record the change ID")
	assert.Equal(t, time.UnixMilli(1792107433716), book.LastUpdated, "wsHandleOrderbook should record the change time")

	// A change with neither side moved still advances the change ID the next one follows
	require.NoError(t, ex.wsHandleOrderbook(t.Context(), channel, notification(channel, `{"timestamp":1792107433816,"type":"change","prev_change_id":178645851490,"change_id":178645851491,"instrument_name":"BTC-PERPETUAL","bids":[],"asks":[]}`)), "wsHandleOrderbook must not error for an empty change")
	lastChangeID, err := ex.Websocket.Orderbook.LastUpdateID(perp, asset.Futures)
	require.NoError(t, err, "LastUpdateID must not error")
	assert.Equal(t, int64(178645851491), lastChangeID, "an empty change should advance the change ID")

	err = ex.wsHandleOrderbook(t.Context(), channel, notification(channel, `{"timestamp":1792107433916,"type":"change","prev_change_id":178645851000,"change_id":178645851492,"instrument_name":"BTC-PERPETUAL","bids":[["new",82140.0,5.0]],"asks":[]}`))
	require.ErrorIs(t, err, errOrderbookOutOfSequence, "wsHandleOrderbook must reject a change that does not follow the last one applied")
	_, err = ex.Websocket.Orderbook.GetOrderbook(perp, asset.Futures)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "wsHandleOrderbook should invalidate a book that missed a change")

	// A new snapshot recovers the book
	require.NoError(t, ex.wsHandleOrderbook(t.Context(), channel, notification(channel, `{"timestamp":1792107434016,"type":"snapshot","change_id":178645851600,"instrument_name":"BTC-PERPETUAL","bids":[["new",82141.0,1.0]],"asks":[["new",82141.5,2.0]]}`)), "wsHandleOrderbook must not error for a new snapshot")
	book, err = ex.Websocket.Orderbook.GetOrderbook(perp, asset.Futures)
	require.NoError(t, err, "GetOrderbook must not error after a new snapshot")
	assert.Equal(t, int64(178645851600), book.LastUpdateID, "a new snapshot should replace the invalid book")

	other := newTestExchange(t)
	err = other.wsHandleOrderbook(t.Context(), "book.ETH-PERPETUAL.100ms", notification("book.ETH-PERPETUAL.100ms", `{"timestamp":1792107433716,"type":"change","prev_change_id":1,"change_id":2,"instrument_name":"ETH-PERPETUAL","bids":[["new",2500.0,1.0]],"asks":[]}`))
	assert.ErrorIs(t, err, orderbook.ErrDepthNotFound, "wsHandleOrderbook should reject a change before any snapshot")

	err = other.wsHandleOrderbook(t.Context(), channel, notification(channel, `{"type":"bogus","instrument_name":"BTC-PERPETUAL","bids":[],"asks":[]}`))
	assert.ErrorIs(t, err, errMalformedNotification, "wsHandleOrderbook should reject an unknown notification type")
	err = other.wsHandleOrderbook(t.Context(), channel, notification(channel, `{"type":"snapshot","instrument_name":"BTC","bids":[],"asks":[]}`))
	assert.ErrorIs(t, err, errUnsupportedInstrumentFormat, "wsHandleOrderbook should reject an instrument it cannot resolve")
	err = other.wsHandleOrderbook(t.Context(), channel, notification(channel, `{"type":"snapshot","instrument_name":"BTC-PERPETUAL","bids":[["new",1]],"asks":[]}`))
	assert.ErrorIs(t, err, errMalformedNotification, "wsHandleOrderbook should reject a malformed level")

	// Trimmed from a live grouped book, which every notification sends whole
	const grouped = "book.BTC-PERPETUAL.none.10.100ms"
	require.NoError(t, other.wsHandleOrderbook(t.Context(), grouped, notification(grouped, `{"timestamp":1792107433899,"change_id":178645851772,"instrument_name":"BTC-PERPETUAL","bids":[[82141.0,66900.0],[82140.5,10.0]],"asks":[[82141.5,59940.0],[82142.0,10.0]]}`)), "wsHandleOrderbook must not error for a grouped book")
	book, err = other.Websocket.Orderbook.GetOrderbook(perp, asset.Futures)
	require.NoError(t, err, "GetOrderbook must not error for a grouped book")
	assert.Equal(t, orderbook.Levels{{Price: 82141, Amount: 66900}, {Price: 82140.5, Amount: 10}}, book.Bids, "wsHandleOrderbook should load a grouped book's bids")
	assert.Equal(t, orderbook.Levels{{Price: 82141.5, Amount: 59940}, {Price: 82142, Amount: 10}}, book.Asks, "wsHandleOrderbook should load a grouped book's asks")
	assert.Equal(t, int64(178645851772), book.LastUpdateID, "wsHandleOrderbook should record a grouped book's change ID")
	err = other.wsHandleOrderbook(t.Context(), grouped, notification(grouped, `{"instrument_name":"BTC","bids":[],"asks":[]}`))
	assert.ErrorIs(t, err, errUnsupportedInstrumentFormat, "wsHandleOrderbook should reject a grouped book's unresolved instrument")
	err = other.wsHandleOrderbook(t.Context(), grouped, notification(grouped, `{"instrument_name":"BTC-PERPETUAL","bids":[[1]],"asks":[]}`))
	assert.ErrorIs(t, err, errMalformedNotification, "wsHandleOrderbook should reject a grouped book's malformed level")
}

func TestOrderbookChangeLevels(t *testing.T) {
	t.Parallel()
	levels := orderbookChangeLevels([]WebsocketOrderbookChange{
		{Action: "new", Price: 100, Amount: 1},
		{Action: "change", Price: 101, Amount: 2},
		{Action: bookDelete, Price: 102, Amount: 3},
	})
	assert.Equal(t, orderbook.Levels{{Price: 100, Amount: 1}, {Price: 101, Amount: 2}, {Price: 102}}, levels, "orderbookChangeLevels should zero a deleted level's amount, which removes it")
	assert.Empty(t, orderbookChangeLevels(nil), "orderbookChangeLevels should return no levels for no changes")
}

func TestResubscribeChannel(t *testing.T) {
	t.Parallel()
	// Without a held subscription for the channel nothing is sent
	newTestExchange(t).resubscribeChannel(t.Context(), "book.BTC-PERPETUAL.100ms")
	if !mockTests {
		return
	}
	ex, venue := newWebsocketTestExchange(t)
	sub := &subscription.Subscription{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": "btc_usd"}}
	require.NoError(t, ex.Subscribe(subscription.List{sub}), "Subscribe must not error")
	held := ex.Websocket.GetSubscriptions()
	require.Len(t, held, 1, "Subscribe must store the subscription")
	ex.resubscribeChannel(t.Context(), held[0].QualifiedChannel)
	assert.Eventually(t, func() bool {
		return len(venue.received("public/unsubscribe")) == 1 && len(venue.received("public/subscribe")) == 2
	}, 5*time.Second, 10*time.Millisecond, "resubscribeChannel should unsubscribe and subscribe the channel again")

	t.Run("unsubscribe refused", func(t *testing.T) {
		t.Parallel()
		ex, venue, sub := newSubscribedTestExchange(t)
		venue.refuse("public/unsubscribe", 1)
		ex.resubscribeChannel(t.Context(), sub.QualifiedChannel)
		assert.Eventually(t, func() bool {
			return len(venue.received("public/unsubscribe")) == 1 && sub.State() == subscription.SubscribedState
		}, 5*time.Second, 10*time.Millisecond, "a channel Deribit did not unsubscribe should be left subscribed")
		ex.resubscribeChannel(t.Context(), sub.QualifiedChannel)
		assert.Eventually(t, func() bool {
			return len(venue.received("public/unsubscribe")) == 2 && len(venue.received("public/subscribe")) == 2
		}, 5*time.Second, 10*time.Millisecond, "the next change out of sequence should resubscribe a channel Deribit did not unsubscribe")
	})

	t.Run("subscribe refused", func(t *testing.T) {
		t.Parallel()
		ex, venue, sub := newSubscribedTestExchange(t)
		venue.refuse("public/subscribe", 1)
		ex.resubscribeChannel(t.Context(), sub.QualifiedChannel)
		assert.Eventually(t, func() bool {
			return len(venue.received("public/subscribe")) == 3 && sub.State() == subscription.SubscribedState
		}, 5*time.Second, 10*time.Millisecond, "a refused resubscription should be subscribed again")
		assert.Len(t, ex.Websocket.GetSubscriptions(), 1, "a refused resubscription should be held again")
	})

	t.Run("change during a resubscription", func(t *testing.T) {
		t.Parallel()
		ex, venue := newWebsocketTestExchange(t)
		book := func(base string) *subscription.Subscription {
			return &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{currency.NewPairWithDelimiter(base, "PERPETUAL", currency.DashDelimiter)}, Interval: kline.HundredMilliseconds}
		}
		require.NoError(t, ex.Subscribe(subscription.List{book("BTC"), book("ETH")}), "Subscribe must not error")
		btc, eth := heldSubscription(ex, "book.BTC-PERPETUAL.100ms"), heldSubscription(ex, "book.ETH-PERPETUAL.100ms")
		require.NotNil(t, btc, "Subscribe must hold the BTC-PERPETUAL book")
		require.NotNil(t, eth, "Subscribe must hold the ETH-PERPETUAL book")
		// A change the reader cannot apply arrives for another book while the resubscription waits for its answer
		venue.notifyBeforeNext("public/unsubscribe", notification("book.ETH-PERPETUAL.100ms", `{"timestamp":1792107433716,"type":"change","prev_change_id":1,"change_id":2,"instrument_name":"ETH-PERPETUAL","bids":[],"asks":[]}`))
		require.NoError(t, ex.resubscribe(t.Context(), markedResubscribing(t, ex, btc), ex.connectionGeneration.Load()), "resubscribe must not wait out its answer while the reader resubscribes another book")
		assert.Eventually(t, func() bool {
			return len(venue.received("public/unsubscribe")) == 2 && len(venue.received("public/subscribe")) == 3 && eth.State() == subscription.SubscribedState
		}, 5*time.Second, 10*time.Millisecond, "the other book should be resubscribed once the resubscription under way is made")
	})
}

func TestMarkResubscribing(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	const channel = "deribit_price_index.btc_usd"
	assert.Nil(t, ex.markResubscribing(channel), "markResubscribing should return nil without a held subscription")
	sub := &subscription.Subscription{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": "btc_usd"}, QualifiedChannel: channel}
	sub.SetKey(qualifiedChannelKey{sub})
	require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(ex.Websocket.Conn, sub), "AddSuccessfulSubscriptions must not error")
	assert.Same(t, sub, ex.markResubscribing(channel), "markResubscribing should return the held subscription")
	assert.Equal(t, subscription.ResubscribingState, sub.State(), "markResubscribing should mark the subscription as resubscribing")
	assert.Nil(t, ex.markResubscribing(channel), "markResubscribing should leave a subscription already resubscribing to that resubscription")
}

// markedResubscribing marks a held subscription as resubscribing, as a book change the reader cannot apply does, and
// returns it
func markedResubscribing(t *testing.T, ex *Exchange, sub *subscription.Subscription) *subscription.Subscription {
	t.Helper()
	require.Same(t, sub, ex.markResubscribing(sub.QualifiedChannel), "markResubscribing must mark the held subscription")
	return sub
}

// disconnectWebsocket disables an exchange's websocket and shuts its connection down; the connection monitor shuts a
// disabled websocket down itself at its next check, which may come first
func disconnectWebsocket(t *testing.T, ex *Exchange) {
	t.Helper()
	require.NoError(t, ex.Websocket.Disable(), "Disable must not error")
	if err := ex.Websocket.Shutdown(); err != nil {
		require.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown must only fail when the connection monitor has shut the connection down first")
	}
}

// newSubscribedTestExchange returns an exchange connected to a mock of Deribit's websocket, the mock, and the
// subscription to an index channel the exchange holds
func newSubscribedTestExchange(t *testing.T) (*Exchange, *websocketVenue, *subscription.Subscription) {
	t.Helper()
	ex, venue := newWebsocketTestExchange(t)
	require.NoError(t, ex.Subscribe(subscription.List{{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": "btc_usd"}}}), "Subscribe must not error")
	held := ex.Websocket.GetSubscriptions()
	require.Len(t, held, 1, "Subscribe must store the subscription")
	return ex, venue, held[0]
}

func TestResubscribe(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("the mock websocket refuses requests on demand; live tests share Deribit's websocket")
	}

	t.Run("resubscribed", func(t *testing.T) {
		t.Parallel()
		ex, venue, sub := newSubscribedTestExchange(t)
		require.NoError(t, ex.resubscribe(t.Context(), markedResubscribing(t, ex, sub), ex.connectionGeneration.Load()), "resubscribe must not error")
		assert.Len(t, venue.received("public/unsubscribe"), 1, "resubscribe should unsubscribe the channel")
		assert.Len(t, venue.received("public/subscribe"), 2, "resubscribe should subscribe the channel again")
		assert.Equal(t, subscription.SubscribedState, sub.State(), "resubscribe should leave the subscription subscribed")
		assert.Len(t, ex.Websocket.GetSubscriptions(), 1, "resubscribe should leave the subscription held")
	})

	t.Run("unsubscribe refused", func(t *testing.T) {
		t.Parallel()
		ex, venue, sub := newSubscribedTestExchange(t)
		venue.refuse("public/unsubscribe", 1)
		assert.ErrorIs(t, ex.resubscribe(t.Context(), markedResubscribing(t, ex, sub), ex.connectionGeneration.Load()), errUnacknowledgedSubscription, "resubscribe should return the refused unsubscribe")
		assert.Len(t, venue.received("public/subscribe"), 1, "resubscribe should not subscribe a channel it could not unsubscribe")
		assert.Equal(t, subscription.SubscribedState, sub.State(), "resubscribe should mark the subscription it could not unsubscribe as subscribed")
		assert.Len(t, ex.Websocket.GetSubscriptions(), 1, "resubscribe should leave the subscription it could not unsubscribe held")
		require.NoError(t, ex.resubscribe(t.Context(), markedResubscribing(t, ex, sub), ex.connectionGeneration.Load()), "a later resubscription must not error")
		assert.Len(t, venue.received("public/subscribe"), 2, "a later resubscription should subscribe the channel again")
	})

	t.Run("subscribe refused", func(t *testing.T) {
		t.Parallel()
		ex, venue, sub := newSubscribedTestExchange(t)
		venue.refuse("public/subscribe", 1)
		require.NoError(t, ex.resubscribe(t.Context(), markedResubscribing(t, ex, sub), ex.connectionGeneration.Load()), "resubscribe must not error once a retried subscribe succeeds")
		assert.Len(t, venue.received("public/subscribe"), 3, "resubscribe should retry the refused subscribe")
		assert.Equal(t, subscription.SubscribedState, sub.State(), "resubscribe should leave the subscription subscribed")
		assert.Len(t, ex.Websocket.GetSubscriptions(), 1, "resubscribe should hold the subscription again")
	})

	t.Run("subscribe not retried", func(t *testing.T) {
		t.Parallel()
		ex, venue, sub := newSubscribedTestExchange(t)
		venue.refuse("public/subscribe", 1)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		err := ex.resubscribe(ctx, markedResubscribing(t, ex, sub), ex.connectionGeneration.Load())
		assert.ErrorIs(t, err, errUnacknowledgedSubscription, "resubscribe should return the refused subscribe")
		assert.ErrorIs(t, err, context.Canceled, "resubscribe should return why the subscribe was not retried")
		assert.Len(t, venue.received("public/subscribe"), 2, "resubscribe should not retry the subscribe once the context ends")
		assert.Empty(t, ex.Websocket.GetSubscriptions(), "resubscribe should not hold a subscription Deribit refused")
	})

	t.Run("removed meanwhile", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			remove func(context.Context, *Exchange, *subscription.Subscription) error
		}{
			{"unsubscribed", func(_ context.Context, ex *Exchange, sub *subscription.Subscription) error {
				return ex.Unsubscribe(subscription.List{sub})
			}},
			{"unsubscribed by an equal subscription", func(_ context.Context, ex *Exchange, _ *subscription.Subscription) error {
				return ex.Unsubscribe(subscription.List{{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": "btc_usd"}}})
			}},
			{"flushed", func(ctx context.Context, ex *Exchange, _ *subscription.Subscription) error {
				ex.Features.Supports.WebsocketCapabilities.Subscribe = true
				ex.Features.Supports.WebsocketCapabilities.Unsubscribe = true
				ex.Websocket.GenerateSubs = func() (subscription.List, error) { return subscription.List{}, nil }
				return ex.Websocket.FlushChannels(ctx)
			}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex, venue, sub := newSubscribedTestExchange(t)
				markedResubscribing(t, ex, sub)
				require.NoError(t, tc.remove(t.Context(), ex, sub), "removing a subscription resubscribing must not error")
				require.Empty(t, ex.Websocket.GetSubscriptions(), "removing a subscription resubscribing must leave none held")
				require.NoError(t, ex.resubscribe(t.Context(), sub, ex.connectionGeneration.Load()), "resubscribe must not error for a subscription removed meanwhile")
				assert.Len(t, venue.received("public/subscribe"), 1, "resubscribe should not subscribe a channel removed meanwhile again")
				assert.NotEqual(t, subscription.ResubscribingState, sub.State(), "resubscribe should not leave a subscription removed meanwhile marked as resubscribing")
				assert.Empty(t, ex.Websocket.GetSubscriptions(), "resubscribe should not hold a subscription removed meanwhile")
			})
		}
	})

	t.Run("connection gone", func(t *testing.T) {
		t.Parallel()
		ex, venue, sub := newSubscribedTestExchange(t)
		generation := ex.connectionGeneration.Load()
		markedResubscribing(t, ex, sub)
		disconnectWebsocket(t, ex)
		require.NoError(t, ex.resubscribe(t.Context(), sub, generation), "resubscribe must leave a connection that has gone to the new one")
		assert.Empty(t, venue.received("public/unsubscribe"), "resubscribe should send nothing for a connection that has gone")
		assert.Equal(t, subscription.SubscribedState, sub.State(), "resubscribe should mark a subscription it did not resubscribe as subscribed again")
		require.NoError(t, ex.Websocket.Enable(t.Context()), "Enable must connect the websocket again")
	})

	t.Run("held again by a later connection", func(t *testing.T) {
		t.Parallel()
		ex, venue, sub := newSubscribedTestExchange(t)
		// A resubscription started on an earlier connection finds the subscription held, and marked as resubscribing,
		// by the current connection, as a later connection may hold the same subscription again
		require.NoError(t, ex.resubscribe(t.Context(), markedResubscribing(t, ex, sub), ex.connectionGeneration.Load()-1), "resubscribe must leave a connection that has gone to the new one")
		assert.Empty(t, venue.received("public/unsubscribe"), "resubscribe should send nothing for a connection that has gone")
		assert.Equal(t, subscription.ResubscribingState, sub.State(), "resubscribe should leave a subscription a later connection holds to that connection's resubscription")
	})

	t.Run("unsubscribe unanswered", func(t *testing.T) {
		t.Parallel()
		ex, venue := newWebsocketTestExchange(t)
		// Connect again with a short response limit, so the unsubscribe whose answer the mock holds back times out quickly
		disconnectWebsocket(t, ex)
		require.NoError(t, ex.Websocket.SetupNewConnection(&websocket.ConnectionSetup{URL: ex.Websocket.GetWebsocketURL(), ResponseCheckTimeout: ex.WebsocketResponseCheckTimeout, ResponseMaxLimit: time.Second}), "SetupNewConnection must not error")
		require.NoError(t, ex.Websocket.Enable(t.Context()), "Enable must connect the websocket again")
		require.NoError(t, ex.Subscribe(subscription.List{{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": "btc_usd"}}}), "Subscribe must not error")
		sub := heldSubscription(ex, "deribit_price_index.btc_usd")
		require.NotNil(t, sub, "Subscribe must hold the subscription")
		venue.holdNext(t, "public/unsubscribe")
		err := ex.resubscribe(t.Context(), markedResubscribing(t, ex, sub), ex.connectionGeneration.Load())
		assert.ErrorIs(t, err, websocket.ErrSignatureTimeout, "resubscribe should return its unanswered unsubscribe")
		assert.Len(t, venue.received("public/subscribe"), 1, "resubscribe should not subscribe a channel it takes to be subscribed still")
		assert.Equal(t, subscription.SubscribedState, sub.State(), "resubscribe should mark the subscription it takes to be subscribed still as subscribed")
		assert.Same(t, sub, heldSubscription(ex, sub.QualifiedChannel), "resubscribe should leave the subscription it takes to be subscribed still held")
	})

	t.Run("unsubscribe not sent", func(t *testing.T) {
		t.Parallel()
		ex, venue := newUnauthenticatedWebsocketTestExchange(t)
		sub := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{currency.NewPairWithDelimiter("BTC", "PERPETUAL", currency.DashDelimiter)}, Interval: kline.Raw, Authenticated: true, QualifiedChannel: "book.BTC-PERPETUAL.raw"}
		sub.SetKey(qualifiedChannelKey{sub})
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(ex.Websocket.Conn, sub), "AddSuccessfulSubscriptions must not error")
		err := ex.resubscribe(t.Context(), markedResubscribing(t, ex, sub), ex.connectionGeneration.Load())
		assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "resubscribe should return why its unsubscribe was not sent")
		assert.Empty(t, venue.received("private/unsubscribe"), "resubscribe should not send a private unsubscribe from an unauthenticated connection")
		assert.Equal(t, subscription.SubscribedState, sub.State(), "resubscribe should mark the subscription it could not unsubscribe as subscribed")
		assert.Same(t, sub, heldSubscription(ex, sub.QualifiedChannel), "resubscribe should leave the subscription it could not unsubscribe held")
	})

	t.Run("private", func(t *testing.T) {
		t.Parallel()
		ex, venue := newWebsocketTestExchange(t)
		require.NoError(t, ex.Subscribe(subscription.List{{Channel: subscription.OrderbookChannel, Asset: asset.Futures, Pairs: currency.Pairs{currency.NewPairWithDelimiter("BTC", "PERPETUAL", currency.DashDelimiter)}, Interval: kline.Raw, Authenticated: true}}), "Subscribe must not error")
		sub := heldSubscription(ex, "book.BTC-PERPETUAL.raw")
		require.NotNil(t, sub, "Subscribe must hold the raw book")
		require.NoError(t, ex.resubscribe(t.Context(), markedResubscribing(t, ex, sub), ex.connectionGeneration.Load()), "resubscribe must not error")
		assert.Len(t, venue.received("private/unsubscribe"), 1, "resubscribe should unsubscribe a private subscription through private/unsubscribe")
		assert.Len(t, venue.received("private/subscribe"), 2, "resubscribe should subscribe a private subscription again through private/subscribe")
		assert.Empty(t, venue.received("public/unsubscribe"), "resubscribe should not unsubscribe a private subscription through public/unsubscribe")
		assert.Equal(t, subscription.SubscribedState, sub.State(), "resubscribe should leave the private subscription subscribed")
	})

	t.Run("held while resubscribing", func(t *testing.T) {
		t.Parallel()
		ex, venue, sub := newSubscribedTestExchange(t)
		markedResubscribing(t, ex, sub)
		release := venue.holdNext(t, "public/subscribe")
		done := make(chan error, 1)
		go func() { done <- ex.resubscribe(t.Context(), sub, ex.connectionGeneration.Load()) }()
		require.Eventually(t, func() bool { return len(venue.received("public/subscribe")) == 2 }, 5*time.Second, time.Millisecond, "resubscribe must send its subscribe")
		assert.Same(t, sub, heldSubscription(ex, sub.QualifiedChannel), "resubscribe should hold the subscription while its subscribe is answered")
		assert.Equal(t, subscription.ResubscribingState, sub.State(), "resubscribe should leave the subscription marked as resubscribing until its subscribe is answered")
		release()
		require.NoError(t, resubscribed(t, ex, done), "resubscribe must not error")
		assert.Equal(t, subscription.SubscribedState, sub.State(), "resubscribe should leave the subscription subscribed")
	})

	t.Run("flushed while its subscribe is answered", func(t *testing.T) {
		t.Parallel()
		ex, venue, sub := newSubscribedTestExchange(t)
		markedResubscribing(t, ex, sub)
		ex.Features.Supports.WebsocketCapabilities.Subscribe = true
		ex.Features.Supports.WebsocketCapabilities.Unsubscribe = true
		ex.Websocket.GenerateSubs = func() (subscription.List, error) { return subscription.List{}, nil }
		venue.delayNext("public/subscribe", 500*time.Millisecond)
		done := make(chan error, 1)
		go func() { done <- ex.resubscribe(t.Context(), sub, ex.connectionGeneration.Load()) }()
		require.Eventually(t, func() bool { return len(venue.received("public/subscribe")) == 2 }, 5*time.Second, time.Millisecond, "resubscribe must send its subscribe")
		require.NoError(t, ex.Websocket.FlushChannels(t.Context()), "FlushChannels must not error while a resubscription waits for its subscribe")
		assert.Empty(t, ex.Websocket.GetSubscriptions(), "FlushChannels should remove a subscription resubscribing")
		assert.Len(t, venue.received("public/unsubscribe"), 2, "FlushChannels should unsubscribe the channel the resubscription subscribed")
		assert.Equal(t, subscription.UnsubscribedState, sub.State(), "FlushChannels should wait for the resubscription's subscribe, then leave the subscription unsubscribed")
		require.NoError(t, resubscribed(t, ex, done), "resubscribe must not error")
		assert.Empty(t, ex.Websocket.GetSubscriptions(), "resubscribe should not hold a subscription flushed meanwhile")
		assert.Equal(t, subscription.UnsubscribedState, sub.State(), "resubscribe should leave a subscription flushed meanwhile unsubscribed")
	})

	t.Run("subscribed meanwhile", func(t *testing.T) {
		t.Parallel()
		ex, venue, sub := newSubscribedTestExchange(t)
		markedResubscribing(t, ex, sub)
		// The resubscription's subscribe is refused late, so the subscribe below waits for it and is made in its backoff
		venue.refuse("public/subscribe", 1)
		venue.delayNext("public/subscribe", 100*time.Millisecond)
		done := make(chan error, 1)
		go func() { done <- ex.resubscribe(t.Context(), sub, ex.connectionGeneration.Load()) }()
		require.Eventually(t, func() bool { return len(venue.received("public/subscribe")) == 2 }, 5*time.Second, time.Millisecond, "resubscribe must send its subscribe")
		// The websocket manager subscribes a subscription marked as resubscribing even though it is held
		err := ex.Websocket.SubscribeToChannels(t.Context(), ex.Websocket.Conn, subscription.List{sub})
		assert.ErrorIs(t, err, subscription.ErrDuplicate, "SubscribeToChannels should refuse to hold the subscription twice")
		require.NoError(t, resubscribed(t, ex, done), "resubscribe must not error for a subscription subscribed meanwhile")
		assert.Equal(t, subscription.SubscribedState, sub.State(), "resubscribe should leave a subscription subscribed meanwhile subscribed")
		assert.Same(t, sub, heldSubscription(ex, sub.QualifiedChannel), "resubscribe should leave a subscription subscribed meanwhile held")
		assert.Len(t, venue.received("public/subscribe"), 3, "resubscribe should not subscribe a subscription subscribed meanwhile again")
	})

	t.Run("subscribed meanwhile, then stopped", func(t *testing.T) {
		t.Parallel()
		ex, venue, sub := newSubscribedTestExchange(t)
		markedResubscribing(t, ex, sub)
		venue.refuse("public/subscribe", 1)
		venue.delayNext("public/subscribe", 100*time.Millisecond)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- ex.resubscribe(ctx, sub, ex.connectionGeneration.Load()) }()
		require.Eventually(t, func() bool { return len(venue.received("public/subscribe")) == 2 }, 5*time.Second, time.Millisecond, "resubscribe must send its subscribe")
		err := ex.Websocket.SubscribeToChannels(t.Context(), ex.Websocket.Conn, subscription.List{sub})
		assert.ErrorIs(t, err, subscription.ErrDuplicate, "SubscribeToChannels should refuse to hold the subscription twice")
		// Stopped during its backoff, the resubscription would remove a subscription still resubscribing
		cancel()
		if err := resubscribed(t, ex, done); err != nil {
			assert.ErrorIs(t, err, context.Canceled, "resubscribe should only return why it stopped")
		}
		assert.Equal(t, subscription.SubscribedState, sub.State(), "resubscribe should leave a subscription subscribed meanwhile subscribed")
		assert.Same(t, sub, heldSubscription(ex, sub.QualifiedChannel), "resubscribe should not remove a subscription subscribed meanwhile")
	})

	t.Run("connection gone while its subscribe is retried", func(t *testing.T) {
		t.Parallel()
		ex, venue, sub := newSubscribedTestExchange(t)
		generation := ex.connectionGeneration.Load()
		markedResubscribing(t, ex, sub)
		// The connection goes before the retry, as it may during a retry's backoff, and the manager clears its stores
		disconnectWebsocket(t, ex)
		done := make(chan error, 1)
		go func() { done <- ex.subscribeAgain(t.Context(), sub, generation) }()
		assert.ErrorIs(t, resubscribed(t, ex, done), websocket.ErrNotConnected, "subscribeAgain should stop retrying once the connection has gone")
		assert.Len(t, venue.received("public/subscribe"), 1, "subscribeAgain should send nothing once the connection has gone")
		assert.Equal(t, subscription.UnsubscribedState, sub.State(), "subscribeAgain should leave a subscription whose connection has gone unsubscribed")
		require.NoError(t, ex.Websocket.Enable(t.Context()), "Enable must connect the websocket again")
	})

	t.Run("unsubscribed while its subscribe is retried", func(t *testing.T) {
		t.Parallel()
		ex, venue, sub := newSubscribedTestExchange(t)
		markedResubscribing(t, ex, sub)
		venue.refuse("public/subscribe", 2)
		done := make(chan error, 1)
		go func() { done <- ex.resubscribe(t.Context(), sub, ex.connectionGeneration.Load()) }()
		require.Eventually(t, func() bool { return len(venue.received("public/subscribe")) >= 2 }, 5*time.Second, time.Millisecond, "resubscribe must send its subscribe")
		require.NoError(t, ex.Websocket.UnsubscribeChannels(t.Context(), ex.Websocket.Conn, subscription.List{sub}), "UnsubscribeChannels must remove a subscription whose subscribe is retried")
		subscribes := len(venue.received("public/subscribe"))
		require.NoError(t, resubscribed(t, ex, done), "resubscribe must not error once its subscription is removed")
		assert.Len(t, venue.received("public/subscribe"), subscribes, "resubscribe should stop retrying the subscribe of a subscription removed meanwhile")
		assert.Empty(t, ex.Websocket.GetSubscriptions(), "resubscribe should not hold a subscription removed while its subscribe is retried")
		assert.Equal(t, subscription.UnsubscribedState, sub.State(), "resubscribe should leave a subscription removed meanwhile unsubscribed")
	})
}

// resubscribed returns the error of a resubscription running in the background, failing the test rather than leaving it
// to hang once the resubscription has run longer than its requests could on a connection using the exchange's response
// limit: an unsubscribe and a subscribe, then the subscribe until the attempts run out, each answered or timed out,
// with the backoff between them; the mock venue never asks for a request to be retried
func resubscribed(t *testing.T, ex *Exchange, done <-chan error) error {
	t.Helper()
	limit := time.Duration(maximumRequestAttempts+1)*ex.WebsocketResponseMaxLimit + time.Duration(maximumRequestAttempts)*time.Second
	select {
	case err := <-done:
		return err
	case <-time.After(limit):
		require.FailNow(t, "the resubscription must return within "+limit.String())
		return nil
	}
}

func TestSubscribeAgain(t *testing.T) {
	t.Parallel()
	newSub := func() *subscription.Subscription {
		sub := &subscription.Subscription{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": "btc_usd"}, QualifiedChannel: "deribit_price_index.btc_usd"}
		sub.SetKey(qualifiedChannelKey{sub})
		return sub
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	assert.ErrorIs(t, newTestExchange(t).subscribeAgain(ctx, newSub(), 0), context.Canceled, "subscribeAgain should stop when the context ends")
	assert.ErrorIs(t, newTestExchange(t).subscribeAgain(t.Context(), newSub(), 0), websocket.ErrNotConnected, "subscribeAgain should leave subscribing again to the reconnection")

	if !mockTests {
		return
	}
	// A resubscription leaves the subscription whose channel it unsubscribed held, marked as resubscribing
	held := func(t *testing.T, ex *Exchange) *subscription.Subscription {
		t.Helper()
		sub := newSub()
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(ex.Websocket.Conn, sub), "AddSuccessfulSubscriptions must not error")
		return markedResubscribing(t, ex, sub)
	}
	ex, venue := newWebsocketTestExchange(t)
	retried := held(t, ex)
	venue.refuse("public/subscribe", 1)
	require.NoError(t, ex.subscribeAgain(t.Context(), retried, ex.connectionGeneration.Load()), "subscribeAgain must not error once a subscribe succeeds")
	assert.Len(t, venue.received("public/subscribe"), 2, "subscribeAgain should retry a refused subscribe")
	assert.Equal(t, subscription.SubscribedState, retried.State(), "subscribeAgain should leave the subscription subscribed")
	assert.Same(t, retried, heldSubscription(ex, retried.QualifiedChannel), "subscribeAgain should leave the subscription held")

	ex, venue = newWebsocketTestExchange(t)
	refused := held(t, ex)
	venue.refuse("public/subscribe", maximumRequestAttempts)
	assert.ErrorIs(t, ex.subscribeAgain(t.Context(), refused, ex.connectionGeneration.Load()), errUnacknowledgedSubscription, "subscribeAgain should return the last refusal")
	assert.Len(t, venue.received("public/subscribe"), maximumRequestAttempts-1, "subscribeAgain should stop retrying after the attempts a resubscription has left")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "subscribeAgain should remove a subscription Deribit refused")
	assert.Equal(t, subscription.UnsubscribedState, refused.State(), "subscribeAgain should leave a subscription Deribit refused unsubscribed")

	ex, venue = newWebsocketTestExchange(t)
	stopped := held(t, ex)
	assert.ErrorIs(t, ex.subscribeAgain(ctx, stopped, ex.connectionGeneration.Load()), context.Canceled, "subscribeAgain should stop when the context ends")
	assert.Empty(t, venue.received("public/subscribe"), "subscribeAgain should not subscribe once the context ends")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "subscribeAgain should remove a subscription it stops retrying")

	ex, venue = newWebsocketTestExchange(t)
	assert.NoError(t, ex.subscribeAgain(t.Context(), newSub(), ex.connectionGeneration.Load()), "subscribeAgain should stop for a subscription no longer held")
	assert.Empty(t, venue.received("public/subscribe"), "subscribeAgain should not subscribe a subscription no longer held")
	replacement := held(t, ex)
	assert.ErrorIs(t, ex.subscribeAgain(ctx, newSub(), ex.connectionGeneration.Load()), context.Canceled, "subscribeAgain should stop when the context ends")
	assert.Same(t, replacement, heldSubscription(ex, replacement.QualifiedChannel), "subscribeAgain should leave a subscription held in place of its own")

	ex, venue = newWebsocketTestExchange(t)
	assert.ErrorIs(t, ex.subscribeAgain(t.Context(), held(t, ex), ex.connectionGeneration.Load()-1), websocket.ErrNotConnected, "subscribeAgain should leave a connection that has gone to the new one")
	assert.Empty(t, venue.received("public/subscribe"), "subscribeAgain should send nothing for a connection that has gone")
}

func TestRemoveUnsubscribed(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("the test holds subscriptions the live connection has not subscribed; live tests share Deribit's websocket")
	}
	ex, _ := newWebsocketTestExchange(t)
	resubscribing := func(t *testing.T, held bool) *subscription.Subscription {
		t.Helper()
		sub := &subscription.Subscription{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": "btc_usd"}, QualifiedChannel: "deribit_price_index.btc_usd"}
		sub.SetKey(qualifiedChannelKey{sub})
		if held {
			require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(ex.Websocket.Conn, sub), "AddSuccessfulSubscriptions must not error")
		}
		require.NoError(t, sub.SetState(subscription.ResubscribingState), "SetState must not error")
		return sub
	}
	generation := ex.connectionGeneration.Load()

	sub := resubscribing(t, true)
	unrelated := &subscription.Subscription{Channel: ChannelPriceIndex, Params: map[string]any{"index_name": "eth_usd"}, QualifiedChannel: "deribit_price_index.eth_usd"}
	unrelated.SetKey(qualifiedChannelKey{unrelated})
	require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(ex.Websocket.Conn, unrelated), "AddSuccessfulSubscriptions must not error")
	require.NoError(t, ex.removeUnsubscribed(generation, sub), "removeUnsubscribed must not error")
	assert.Nil(t, heldSubscription(ex, sub.QualifiedChannel), "removeUnsubscribed should remove a subscription still resubscribing")
	assert.Equal(t, subscription.UnsubscribedState, sub.State(), "removeUnsubscribed should leave the subscription it removes unsubscribed")
	assert.Same(t, unrelated, heldSubscription(ex, unrelated.QualifiedChannel), "removeUnsubscribed should leave unrelated subscriptions held")
	assert.Equal(t, subscription.SubscribedState, unrelated.State(), "removeUnsubscribed should leave unrelated subscriptions subscribed")
	require.NoError(t, ex.Websocket.RemoveSubscriptions(ex.Websocket.Conn, unrelated), "RemoveSubscriptions must not error")

	orphan := resubscribing(t, false)
	require.NoError(t, ex.removeUnsubscribed(generation-1, orphan), "removeUnsubscribed must not error once its connection has gone")
	assert.Equal(t, subscription.UnsubscribedState, orphan.State(), "removeUnsubscribed should mark a subscription its connection no longer holds as unsubscribed")
	require.NoError(t, orphan.SetState(subscription.SubscribedState), "SetState must not error")
	require.NoError(t, ex.removeUnsubscribed(generation-1, orphan), "removeUnsubscribed must not error once its connection has gone")
	assert.Equal(t, subscription.SubscribedState, orphan.State(), "removeUnsubscribed should leave a subscription no longer resubscribing as it is")

	replaced := resubscribing(t, false)
	other := resubscribing(t, true)
	require.NoError(t, ex.removeUnsubscribed(generation-1, replaced), "removeUnsubscribed must not error once its connection has gone")
	assert.Equal(t, subscription.UnsubscribedState, replaced.State(), "removeUnsubscribed should mark a subscription another is held in place of as unsubscribed")
	assert.Equal(t, subscription.ResubscribingState, other.State(), "removeUnsubscribed should leave the subscription held in its place alone")
	require.NoError(t, ex.Websocket.RemoveSubscriptions(ex.Websocket.Conn, other), "RemoveSubscriptions must not error")

	held := resubscribing(t, true)
	require.NoError(t, ex.removeUnsubscribed(generation-1, held), "removeUnsubscribed must not error once its connection has gone")
	assert.Same(t, held, heldSubscription(ex, held.QualifiedChannel), "removeUnsubscribed should leave a subscription a later connection holds")
	assert.Equal(t, subscription.ResubscribingState, held.State(), "removeUnsubscribed should leave a subscription a later connection holds to that connection's resubscription")

	require.NoError(t, held.SetState(subscription.SubscribedState), "SetState must not error")
	require.NoError(t, ex.removeUnsubscribed(generation, held), "removeUnsubscribed must not error for a subscription subscribed meanwhile")
	assert.Same(t, held, heldSubscription(ex, held.QualifiedChannel), "removeUnsubscribed should not remove a subscription subscribed meanwhile")
	assert.Equal(t, subscription.SubscribedState, held.State(), "removeUnsubscribed should leave a subscription subscribed meanwhile subscribed")
}

func TestOnConnection(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("the test disconnects its websocket; live tests share Deribit's websocket")
	}
	ex, _ := newWebsocketTestExchange(t)
	newSub := func() *subscription.Subscription {
		sub := &subscription.Subscription{Channel: ChannelPlatformState, QualifiedChannel: ChannelPlatformState}
		sub.SetKey(qualifiedChannelKey{sub})
		return sub
	}
	sub := newSub()
	generation := ex.connectionGeneration.Load()
	var changes int
	change := func() error {
		changes++
		return nil
	}
	require.NoError(t, sub.SetState(subscription.ResubscribingState), "SetState must not error")
	assert.ErrorIs(t, ex.onConnection(generation, sub, change), errNoLongerResubscribing, "onConnection should not make a change to a subscription it does not hold")
	assert.Equal(t, subscription.UnsubscribedState, sub.State(), "onConnection should mark a subscription resubscribing it no longer holds as unsubscribed")
	require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(ex.Websocket.Conn, sub), "AddSuccessfulSubscriptions must not error")
	assert.ErrorIs(t, ex.onConnection(generation, sub, change), errNoLongerResubscribing, "onConnection should not make a change to a subscription subscribed again meanwhile")
	assert.Equal(t, subscription.SubscribedState, sub.State(), "onConnection should leave a subscription subscribed again meanwhile subscribed")
	require.NoError(t, sub.SetState(subscription.UnsubscribingState), "SetState must not error")
	assert.ErrorIs(t, ex.onConnection(generation, sub, change), errNoLongerResubscribing, "onConnection should not make a change to a held subscription in any other state than resubscribing")
	require.NoError(t, sub.SetState(subscription.ResubscribingState), "SetState must not error")
	replaced := newSub()
	require.NoError(t, replaced.SetState(subscription.ResubscribingState), "SetState must not error")
	assert.ErrorIs(t, ex.onConnection(generation, replaced, change), errNoLongerResubscribing, "onConnection should not make a change to a subscription another is held in place of")
	assert.Zero(t, changes, "onConnection should make no change to a subscription it does not hold or that is not resubscribing")
	assert.ErrorIs(t, ex.onConnection(generation+1, sub, change), websocket.ErrNotConnected, "onConnection should not make a change on another connection")
	require.NoError(t, ex.onConnection(generation, sub, change), "onConnection must make a change on its connection")
	assert.Equal(t, 1, changes, "onConnection should make the change once")
	var connectionLocked, subscriptionLocked bool
	require.NoError(t, ex.onConnection(generation, sub, func() error {
		if connectionLocked = !ex.connectionMtx.TryLock(); !connectionLocked {
			ex.connectionMtx.Unlock()
		}
		if subscriptionLocked = !ex.subscriptionMtx.TryLock(); !subscriptionLocked {
			ex.subscriptionMtx.Unlock()
		}
		return nil
	}), "onConnection must make a change on its connection")
	assert.True(t, connectionLocked, "onConnection should hold a new connection back while it makes a change")
	assert.True(t, subscriptionLocked, "onConnection should hold other subscription changes back while it makes a change")
	errChange := errors.New("change failed")
	assert.ErrorIs(t, ex.onConnection(generation, sub, func() error { return errChange }), errChange, "onConnection should return the change's error")

	disconnectWebsocket(t, ex)
	assert.ErrorIs(t, ex.onConnection(generation, sub, change), websocket.ErrNotConnected, "onConnection should not make a change without a connection")
	assert.Equal(t, 1, changes, "onConnection should make no change without a connection")
	require.NoError(t, ex.Websocket.Enable(t.Context()), "Enable must connect the websocket again")
	assert.Greater(t, ex.connectionGeneration.Load(), generation, "a new connection should start a new generation")
}

func TestNextConnectionGeneration(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("the test makes subscription changes on its own connection; live tests share Deribit's websocket")
	}
	ex, _ := newWebsocketTestExchange(t)
	sub := &subscription.Subscription{Channel: ChannelPlatformState, QualifiedChannel: ChannelPlatformState}
	sub.SetKey(qualifiedChannelKey{sub})
	require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(ex.Websocket.Conn, sub), "AddSuccessfulSubscriptions must not error")
	require.NoError(t, sub.SetState(subscription.ResubscribingState), "SetState must not error")
	var made, started atomic.Int64
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for range 200 {
				generation := ex.connectionGeneration.Load()
				_ = ex.onConnection(generation, sub, func() error {
					made.Add(1)
					runtime.Gosched()
					if ex.connectionGeneration.Load() != generation {
						started.Add(1)
					}
					return nil
				})
			}
		})
		wg.Go(func() {
			for range 200 {
				ex.nextConnectionGeneration()
				runtime.Gosched()
			}
		})
	}
	wg.Wait()
	require.Positive(t, made.Load(), "onConnection must make subscription changes for the test to check")
	assert.Zero(t, started.Load(), "nextConnectionGeneration should not start a generation while a subscription change is made")
	assert.Equal(t, uint64(801), ex.connectionGeneration.Load(), "nextConnectionGeneration should start one generation a call")
}

func TestWebsocketOrderbookChangeUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var change WebsocketOrderbookChange
	require.NoError(t, json.Unmarshal([]byte(`["delete",5041.94,0]`), &change), "Unmarshal must not error")
	assert.Equal(t, WebsocketOrderbookChange{Action: bookDelete, Price: 5041.94}, change, "UnmarshalJSON should decode an [action, price, amount] tuple")
	for _, input := range []string{`["new",1]`, `[1,2,3]`, `["new","1",2]`, `["new",1,"2"]`, `{}`} {
		assert.ErrorIsf(t, json.Unmarshal([]byte(input), &change), errInvalidTuple, "UnmarshalJSON should reject %s", input)
	}
}

// TestWebsocketHandleTicker covers the ticker channel for each instrument kind, which
// defaultSubscriptions subscribes for every asset. min_price and max_price are the bounds an order
// is clamped to rather than the day's range, and implied_bid and implied_ask are null outside
// combos, so reading them for an option recorded a zero book
func TestWebsocketHandleTicker(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)

	// Each trimmed from GET /api/v2/public/ticker, which serves what the ticker channel pushes
	for _, tc := range []struct {
		instrument string
		a          asset.Item
		data       string
		want       ticker.Price
	}{
		{
			instrument: "ETH-30OCT26-3200-C", a: asset.Options,
			data: `"timestamp":1789024540294,"last_price":0.0135,"mark_price":0.0131,"index_price":2474.47,"best_bid_price":0.0125,"best_bid_amount":1611.0,"best_ask_price":0.0135,"best_ask_amount":643.0,"implied_bid":null,"implied_ask":null,"min_price":0.0001,"max_price":0.042,"stats":{"high":0.0165,"low":0.0135,"volume":2740.0,"volume_usd":110475.0}`,
			want: ticker.Price{Bid: 0.0125, BidSize: 1611, Ask: 0.0135, AskSize: 643, Last: 0.0135, Close: 0.0135, High: 0.0165, Low: 0.0135, BaseVolume: 2740, MarkPrice: 0.0131, IndexPrice: 2474.47, LastUpdated: time.UnixMilli(1789024540294)},
		},
		{
			// A combo's own book is priced apart from the one implied off its legs, and the sizes
			// served are its own book's
			instrument: "BTC-PCAL-18SEP26_11SEP26-75000", a: asset.OptionCombo,
			data: `"timestamp":1789024540975,"last_price":0.0084,"mark_price":0.0086,"index_price":78179.34,"best_bid_price":0.0081,"best_bid_amount":30.0,"best_ask_price":0.0089,"best_ask_amount":15.0,"implied_bid":0.008,"implied_ask":0.0092,"min_price":-0.0064,"max_price":0.0235,"stats":{"high":0.0084,"low":0.008,"volume":0.2,"volume_usd":0.0}`,
			want: ticker.Price{Bid: 0.0081, BidSize: 30, Ask: 0.0089, AskSize: 15, Last: 0.0084, Close: 0.0084, High: 0.0084, Low: 0.008, BaseVolume: 0.2, MarkPrice: 0.0086, IndexPrice: 78179.34, LastUpdated: time.UnixMilli(1789024540975)},
		},
		{
			instrument: "BTC-FS-25SEP26_10SEP26", a: asset.FutureCombo,
			data: `"timestamp":1789024541973,"last_price":null,"mark_price":119.68,"index_price":78178.92,"best_bid_price":83.0,"best_bid_amount":1.0E+5,"best_ask_price":139.0,"best_ask_amount":1.0E+5,"implied_bid":100.0,"implied_ask":130.0,"min_price":-197.0,"max_price":436.0,"stats":{"high":null,"low":null,"volume":0.0,"volume_usd":0.0,"volume_notional":0.0}`,
			want: ticker.Price{Bid: 83, BidSize: 1e5, Ask: 139, AskSize: 1e5, MarkPrice: 119.68, IndexPrice: 78178.92, LastUpdated: time.UnixMilli(1789024541973)},
		},
		{
			instrument: "BTC-PERPETUAL", a: asset.Futures,
			data: `"timestamp":1789024543036,"last_price":78187.0,"mark_price":78187.9,"index_price":78178.92,"best_bid_price":78186.5,"best_bid_amount":25070.0,"best_ask_price":78187.0,"best_ask_amount":5.77E+4,"implied_bid":null,"implied_ask":null,"min_price":77015.0,"max_price":79361.0,"stats":{"high":79779.5,"low":77728.0,"volume":4652.02138044,"volume_usd":366626080.0,"volume_notional":366626080.0}`,
			want: ticker.Price{Bid: 78186.5, BidSize: 25070, Ask: 78187, AskSize: 57700, Last: 78187, Close: 78187, High: 79779.5, Low: 77728, BaseVolume: 4652.02138044, QuoteVolume: 366626080, MarkPrice: 78187.9, IndexPrice: 78178.92, LastUpdated: time.UnixMilli(1789024543036)},
		},
		{
			instrument: "BTC_USDC", a: asset.Spot,
			data: `"timestamp":1789024542984,"last_price":78223.0,"mark_price":78178.92,"index_price":78178.92,"best_bid_price":78140.0,"best_bid_amount":0.0462,"best_ask_price":78249.0,"best_ask_amount":0.04,"implied_bid":null,"implied_ask":null,"min_price":76615.0,"max_price":79743.0,"stats":{"high":79755.0,"low":77785.0,"volume":8.6814,"volume_usd":682942.62,"volume_notional":682976.9009}`,
			want: ticker.Price{Bid: 78140, BidSize: 0.0462, Ask: 78249, AskSize: 0.04, Last: 78223, Close: 78223, High: 79755, Low: 77785, BaseVolume: 8.6814, QuoteVolume: 682976.9009, MarkPrice: 78178.92, IndexPrice: 78178.92, LastUpdated: time.UnixMilli(1789024542984)},
		},
	} {
		payload := `{"params":{"channel":"ticker.` + tc.instrument + `.100ms","data":{"instrument_name":"` + tc.instrument + `",` + tc.data + `}},"method":"subscription","jsonrpc":"2.0"}`
		require.NoErrorf(t, ex.wsHandleData(t.Context(), []byte(payload)), "wsHandleData must not error for %s", tc.instrument)

		p, err := currency.NewPairFromString(tc.instrument)
		require.NoErrorf(t, err, "NewPairFromString must not error for %s", tc.instrument)
		tc.want.ExchangeName, tc.want.Pair, tc.want.AssetType = ex.Name, p, tc.a
		select {
		case msg := <-ex.Websocket.DataHandler.C:
			got, ok := msg.Data.(*ticker.Price)
			require.Truef(t, ok, "the ticker handler must send a ticker price for %s", tc.instrument)
			assert.Equalf(t, &tc.want, got, "the %s ticker should take its book, last price and range from the fields UpdateTicker reads", tc.a)
		default:
			require.Failf(t, "no ticker price sent", "the ticker handler must send a ticker price for %s", tc.instrument)
		}
	}

	err := ex.wsHandleTicker(t.Context(), notification("ticker.BTC.100ms", `{"instrument_name":"BTC","timestamp":1789024543036}`))
	assert.ErrorIs(t, err, errUnsupportedInstrumentFormat, "wsHandleTicker should reject an instrument it cannot resolve")
	err = ex.wsHandleTicker(t.Context(), notification("ticker.BTC-PERPETUAL.100ms", `{"instrument_name":"BTC-PERPETUAL","last_price":"high"}`))
	assert.ErrorIs(t, err, errMalformedNotification, "wsHandleTicker should reject a malformed ticker")
	assertNothingRelayed(t, ex)
}

func TestWebsocketHandleQuote(t *testing.T) {
	t.Parallel()
	// Trimmed from a live quote.BTC-PERPETUAL notification
	const channel = "quote.BTC-PERPETUAL"
	const quote = `{"timestamp":1792107433899,"instrument_name":"BTC-PERPETUAL","best_ask_price":82141.5,"best_bid_price":82141.0,"best_ask_amount":59940.0,"best_bid_amount":6.69e4}`
	perp := currency.NewPairWithDelimiter("BTC", "PERPETUAL", currency.DashDelimiter)

	t.Run("no stored ticker", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		require.NoError(t, ex.wsHandleNotification(t.Context(), channel, notification(channel, quote)), "wsHandleNotification must not error for a quote")
		exp := &ticker.Price{ExchangeName: ex.Name, Pair: perp, AssetType: asset.Futures, Bid: 82141, BidSize: 66900, Ask: 82141.5, AskSize: 59940, LastUpdated: time.UnixMilli(1792107433899)}
		assert.Equal(t, exp, relayed(t, ex), "wsHandleQuote should relay the quote as a ticker")
		stored, err := ticker.GetTicker(ex.Name, perp, asset.Futures)
		require.NoError(t, err, "GetTicker must not error")
		assert.Equal(t, exp, stored, "wsHandleQuote should store the quote as a ticker")
	})

	t.Run("stored ticker", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		// The ticker channel's BTC-PERPETUAL ticker of TestWebsocketHandleTicker, with an open interest
		require.NoError(t, ticker.ProcessTicker(&ticker.Price{
			ExchangeName: ex.Name,
			Pair:         perp,
			AssetType:    asset.Futures,
			Bid:          78186.5,
			BidSize:      25070,
			Ask:          78187,
			AskSize:      57700,
			Last:         78187,
			Close:        78187,
			High:         79779.5,
			Low:          77728,
			BaseVolume:   4652.02138044,
			QuoteVolume:  366626080,
			MarkPrice:    78187.9,
			IndexPrice:   78178.92,
			OpenInterest: 863084480,
			LastUpdated:  time.UnixMilli(1789024543036),
		}), "ProcessTicker must not error")
		require.NoError(t, ex.wsHandleNotification(t.Context(), channel, notification(channel, quote)), "wsHandleNotification must not error for a quote")
		exp := &ticker.Price{
			ExchangeName: ex.Name,
			Pair:         perp,
			AssetType:    asset.Futures,
			Bid:          82141,
			BidSize:      66900,
			Ask:          82141.5,
			AskSize:      59940,
			Last:         78187,
			Close:        78187,
			High:         79779.5,
			Low:          77728,
			BaseVolume:   4652.02138044,
			QuoteVolume:  366626080,
			MarkPrice:    78187.9,
			IndexPrice:   78178.92,
			OpenInterest: 863084480,
			LastUpdated:  time.UnixMilli(1792107433899),
		}
		assert.Equal(t, exp, relayed(t, ex), "wsHandleQuote should relay the stored ticker with the quote merged onto it")
		stored, err := ticker.GetTicker(ex.Name, perp, asset.Futures)
		require.NoError(t, err, "GetTicker must not error")
		assert.Equal(t, exp, stored, "wsHandleQuote should keep the stored ticker's other fields")
	})

	t.Run("errors", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		assert.ErrorIs(t, ex.wsHandleQuote(t.Context(), notification(channel, `[]`)), errMalformedNotification, "wsHandleQuote should reject a malformed quote")
		assert.ErrorIs(t, ex.wsHandleQuote(t.Context(), notification("quote.BTC", `{"instrument_name":"BTC"}`)), errUnsupportedInstrumentFormat, "wsHandleQuote should reject an instrument it cannot resolve")
		err := ex.wsHandleQuote(t.Context(), notification(channel, `{"timestamp":1792107433899,"instrument_name":"BTC-PERPETUAL","best_ask_price":82141.0,"best_bid_price":82141.0,"best_ask_amount":1.0,"best_bid_amount":1.0}`))
		assert.ErrorIs(t, err, ticker.ErrBidEqualsAsk, "wsHandleQuote should return a ticker the store refuses")
		_, err = ticker.GetTicker(ex.Name, perp, asset.Futures)
		assert.ErrorIs(t, err, ticker.ErrTickerNotFound, "wsHandleQuote should not store a ticker the store refuses")
		assertNothingRelayed(t, ex)
	})
}

// TestWebsocketHandleIncrementalTicker covers incremental_ticker, where Deribit sends one snapshot and then
// only the fields that moved: part of stats or none of it, an explicit 0 for a book side that
// empties, and a null for a range that empties. Built afresh, the first change zeroed every volume
// and the day's range, since the store overwrites a pair wholesale
func TestWebsocketHandleIncrementalTicker(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)

	send := func(instrument, data string) error {
		payload := `{"params":{"channel":"incremental_ticker.` + instrument + `","data":{"instrument_name":"` + instrument + `",` + data + `}},"method":"subscription","jsonrpc":"2.0"}`
		return ex.wsHandleData(t.Context(), []byte(payload))
	}
	received := func() *ticker.Price {
		t.Helper()
		select {
		case msg := <-ex.Websocket.DataHandler.C:
			got, ok := msg.Data.(*ticker.Price)
			require.True(t, ok, "the incremental ticker handler must send a ticker price")
			return got
		default:
			require.Fail(t, "no ticker price sent", "the incremental ticker handler must send one for each message")
			return nil
		}
	}

	// Consecutive messages from incremental_ticker.BTC-PERPETUAL
	for _, data := range []string{
		`"timestamp":1789023748046,"type":"snapshot","stats":{"high":79779.5,"low":77728.0,"price_change":-1.1276,"volume":4662.27109873,"volume_usd":367468920.0,"volume_notional":367468920.0},"index_price":78291.88,"last_price":78301.0,"min_price":77129.0,"max_price":79479.0,"mark_price":78304.05,"best_ask_price":78301.5,"best_bid_price":78301.0,"best_ask_amount":573260.0,"best_bid_amount":163420.0`,
		`"timestamp":1789023748464,"type":"change","stats":{"volume":4662.27109873,"volume_usd":367468920.0,"volume_notional":367468920.0},"index_price":78293.92,"last_price":7.83e4,"min_price":77131.0,"max_price":79481.0,"mark_price":78306.09,"best_ask_price":78300.5,"best_bid_price":7.83e4,"best_ask_amount":66980.0,"best_bid_amount":190.0`,
		`"timestamp":1789023748827,"type":"change","min_price":77130.5,"max_price":79480.5,"mark_price":78305.55,"best_ask_amount":64270.0,"best_bid_amount":530.0`,
	} {
		require.NoError(t, send("BTC-PERPETUAL", data), "wsHandleData must not error")
		received()
	}
	perp := currency.NewPairWithDelimiter("BTC", "PERPETUAL", "-")

	// The data handler's consumer writes the store behind the reader, so an older ticker there must
	// not be what the next change merges onto
	require.NoError(t, ticker.ProcessTicker(&ticker.Price{ExchangeName: ex.Name, Pair: perp, AssetType: asset.Futures, Last: 78200, BaseVolume: 1}), "ProcessTicker must not error")
	require.NoError(t, send("BTC-PERPETUAL", `"timestamp":1789023749000,"type":"change","index_price":78290.5`), "wsHandleData must not error")
	assert.Equal(t, &ticker.Price{
		ExchangeName: ex.Name,
		Pair:         perp,
		AssetType:    asset.Futures,
		Bid:          78300,
		BidSize:      530,
		Ask:          78300.5,
		AskSize:      64270,
		Last:         78300,
		Close:        78300,
		High:         79779.5,
		Low:          77728,
		BaseVolume:   4662.27109873,
		QuoteVolume:  367468920,
		MarkPrice:    78305.55,
		IndexPrice:   78290.5,
		LastUpdated:  time.UnixMilli(1789023749000),
	}, received(), "each change should merge onto the ticker last sent rather than replace it or read the store")

	// Consecutive messages from incremental_ticker.BTC-10SEP26-78000-P, whose only bid is pulled
	require.NoError(t, send("BTC-10SEP26-78000-P", `"timestamp":1789026089754,"type":"snapshot","stats":{"high":0.0055,"low":0.0001,"price_change":-95.6522,"volume":55.9,"volume_usd":8221.17},"index_price":77955.54,"last_price":0.0001,"min_price":0.0001,"max_price":0.0225,"mark_price":0.0009,"best_ask_price":0.0014,"best_bid_price":0.0001,"best_ask_amount":16.0,"best_bid_amount":16.0`), "wsHandleData must not error")
	snapshot := received()
	require.Equal(t, 0.0001, snapshot.Bid, "the snapshot's bid must be recorded")
	require.NoError(t, send("BTC-10SEP26-78000-P", `"timestamp":1789026091615,"type":"change","best_bid_price":0.0,"best_bid_amount":0.0`), "wsHandleData must not error")
	emptied := received()
	assert.Zero(t, emptied.Bid, "a bid sent as 0 should clear the one before it rather than read as absent")
	assert.Zero(t, emptied.BidSize, "a bid size sent as 0 should clear the one before it")
	assert.Equal(t, 0.0014, emptied.Ask, "the side the change leaves out should keep its price")

	// Consecutive messages from incremental_ticker.ETH-25DEC26-1200-P as its only trade left the 24
	// hour window, which Deribit sends by nulling the day's range
	require.NoError(t, send("ETH-25DEC26-1200-P", `"timestamp":1789109461717,"type":"snapshot","state":"open","stats":{"high":0.0034,"low":0.0034,"price_change":0.0,"volume":2.0,"volume_usd":16.86},"index_price":2466.06,"last_price":0.0034,"open_interest":6764.0,"mark_price":0.0035,"best_ask_price":0.0038,"best_bid_price":0.0033,"best_ask_amount":30.0,"best_bid_amount":117.0`), "wsHandleData must not error")
	require.Equal(t, 0.0034, received().High, "the snapshot's high must be recorded")
	require.NoError(t, send("ETH-25DEC26-1200-P", `"timestamp":1789110000494,"type":"change","stats":{"high":null,"low":null,"price_change":null,"volume":0.0,"volume_usd":0.0},"greeks":{"vega":0.67225,"theta":-0.23727,"rho":-0.17539},"index_price":2466.01,"estimated_delivery_price":2466.01,"bid_iv":73.23,"underlying_price":2491.08`), "wsHandleData must not error")
	rolled := received()
	assert.Zero(t, rolled.High, "a high sent as null should clear the one before it rather than read as absent")
	assert.Zero(t, rolled.Low, "a low sent as null should clear the one before it")
	assert.Zero(t, rolled.BaseVolume, "a volume sent as 0 should clear the one before it")
	assert.Equal(t, 0.0034, rolled.Last, "the last price the change leaves out should be kept")

	// A change with no snapshot before it has nothing to merge onto
	assert.ErrorIs(t, send("ETH-PERPETUAL", `"timestamp":1789023748827,"type":"change","mark_price":2474.47`), errNoTickerSnapshot, "a change with no snapshot before it should error")
}

func TestWebsocketIncrementalTickerUnmarshal(t *testing.T) {
	t.Parallel()
	// A live snapshot from incremental_ticker.BTC-12SEP26-80000-C: the first message on the channel
	// carries the whole ticker
	const snapshot = `
{
  "timestamp": 1789101556714,
  "type": "snapshot",
  "state": "open",
  "stats": {"high": 0.0065, "low": 0.0008, "price_change": -80.0, "volume": 906.5, "volume_usd": 118930.6},
  "greeks": {"delta": 0.0924, "gamma": 8.0e-5, "vega": 7.13308, "theta": -90.84534, "rho": 0.21957},
  "index_price": 77086.35,
  "instrument_name": "BTC-12SEP26-80000-C",
  "last_price": 0.0013,
  "settlement_price": 0.00515494,
  "min_price": 0.0001,
  "max_price": 0.0215,
  "open_interest": 696.4,
  "mark_price": 0.0012,
  "interest_rate": 0.0,
  "estimated_delivery_price": 77086.35,
  "best_ask_price": 0.0014,
  "best_bid_price": 0.001,
  "mark_iv": 49.45,
  "bid_iv": 47.47,
  "ask_iv": 51.77,
  "underlying_price": 77092.18,
  "underlying_index": "BTC-12SEP26",
  "best_ask_amount": 106.5,
  "best_bid_amount": 38.0
}
`

	var x WebsocketIncrementalTicker
	require.NoError(t, json.Unmarshal([]byte(snapshot), &x), "Unmarshal must not error")
	exp := WebsocketIncrementalTicker{
		Type:           "snapshot",
		Timestamp:      types.Time(time.UnixMilli(1789101556714)),
		InstrumentName: "BTC-12SEP26-80000-C",
		Stats: WebsocketIncrementalTickerStats{
			High:        new(0.0065),
			Low:         new(0.0008),
			PriceChange: new(-80.0),
			Volume:      new(906.5),
			VolumeUSD:   new(118930.6),
		},
		Greeks: WebsocketIncrementalTickerGreeks{
			Delta: new(0.0924),
			Gamma: new(8.0e-5),
			Rho:   new(0.21957),
			Theta: new(-90.84534),
			Vega:  new(7.13308),
		},
		State:                  new("open"),
		UnderlyingIndex:        new("BTC-12SEP26"),
		BestBidPrice:           new(0.001),
		BestBidAmount:          new(38.0),
		BestAskPrice:           new(0.0014),
		BestAskAmount:          new(106.5),
		LastPrice:              new(0.0013),
		MarkPrice:              new(0.0012),
		IndexPrice:             new(77086.35),
		MinPrice:               new(0.0001),
		MaxPrice:               new(0.0215),
		EstimatedDeliveryPrice: new(77086.35),
		SettlementPrice:        new(0.00515494),
		UnderlyingPrice:        new(77092.18),
		OpenInterest:           new(696.4),
		BidImpliedVolatility:   new(47.47),
		AskImpliedVolatility:   new(51.77),
		MarkImpliedVolatility:  new(49.45),
		InterestRate:           new(0.0),
	}
	assert.Equal(t, exp, x, "WebsocketIncrementalTicker should unmarshal a snapshot correctly")

	// A live change from incremental_ticker.BTC-10SEP26-78000-P as its only bid was pulled: the side
	// arrives as an explicit 0, and everything that did not move is left out
	const change = `
{
  "type": "change",
  "timestamp": 1789026091615,
  "instrument_name": "BTC-10SEP26-78000-P",
  "best_bid_price": 0.0,
  "best_bid_amount": 0.0
}
`

	x = WebsocketIncrementalTicker{}
	require.NoError(t, json.Unmarshal([]byte(change), &x), "Unmarshal must not error")
	exp = WebsocketIncrementalTicker{
		Type:           "change",
		Timestamp:      types.Time(time.UnixMilli(1789026091615)),
		InstrumentName: "BTC-10SEP26-78000-P",
		BestBidPrice:   new(0.0),
		BestBidAmount:  new(0.0),
	}
	assert.Equal(t, exp, x, "WebsocketIncrementalTicker should leave every field a change omits nil")

	// A live snapshot from incremental_ticker.BTC-FS-30OCT26_2OCT26. A combo nobody has traded sends
	// its last price and range as null, which read as nil, and a spread can be priced below zero
	const combo = `
{
  "timestamp": 1789100704971,
  "type": "snapshot",
  "state": "open",
  "stats": {"high": null, "low": null, "price_change": null, "volume": 0.0, "volume_usd": 0.0, "volume_notional": 0.0},
  "index_price": 77061.95,
  "instrument_name": "BTC-FS-30OCT26_2OCT26",
  "last_price": null,
  "min_price": -56.0,
  "max_price": 579.0,
  "mark_price": 261.58,
  "combo_state": "active",
  "best_ask_price": 310.0,
  "best_bid_price": 220.0,
  "implied_ask": 282.5,
  "implied_bid": 247.5,
  "best_ask_amount": 3.0e5,
  "best_bid_amount": 3.25e5
}
`

	x = WebsocketIncrementalTicker{}
	require.NoError(t, json.Unmarshal([]byte(combo), &x), "Unmarshal must not error")
	exp = WebsocketIncrementalTicker{
		Type:           "snapshot",
		Timestamp:      types.Time(time.UnixMilli(1789100704971)),
		InstrumentName: "BTC-FS-30OCT26_2OCT26",
		Stats: WebsocketIncrementalTickerStats{
			Volume:         new(0.0),
			VolumeUSD:      new(0.0),
			VolumeNotional: new(0.0),
		},
		State:         new("open"),
		ComboState:    new("active"),
		BestBidPrice:  new(220.0),
		BestBidAmount: new(3.25e5),
		BestAskPrice:  new(310.0),
		BestAskAmount: new(3.0e5),
		MarkPrice:     new(261.58),
		IndexPrice:    new(77061.95),
		MinPrice:      new(-56.0),
		MaxPrice:      new(579.0),
		ImpliedBid:    new(247.5),
		ImpliedAsk:    new(282.5),
	}
	assert.Equal(t, exp, x, "WebsocketIncrementalTicker should read a null as nil")
}

func TestWebsocketIncrementalTickerTickerPrice(t *testing.T) {
	t.Parallel()
	pair := currency.NewPairWithDelimiter("BTC", "PERPETUAL", currency.DashDelimiter)
	state := &WebsocketIncrementalTicker{
		Timestamp:     types.Time(time.UnixMilli(1789023748046)),
		Stats:         WebsocketIncrementalTickerStats{High: new(79779.5), Low: new(77728.0), Volume: new(4662.27109873), VolumeUSD: new(367468920.0), VolumeNotional: new(367468920.0)},
		BestBidPrice:  new(78301.0),
		BestBidAmount: new(163420.0),
		LastPrice:     new(78301.0),
		MarkPrice:     new(78304.05),
		IndexPrice:    new(78291.88),
		OpenInterest:  new(863084480.0),
	}
	exp := &ticker.Price{
		ExchangeName: "Deribit",
		Pair:         pair,
		AssetType:    asset.Futures,
		LastUpdated:  time.UnixMilli(1789023748046),
		Bid:          78301,
		BidSize:      163420,
		Last:         78301,
		Close:        78301,
		High:         79779.5,
		Low:          77728,
		BaseVolume:   4662.27109873,
		QuoteVolume:  367468920,
		MarkPrice:    78304.05,
		IndexPrice:   78291.88,
		OpenInterest: 863084480,
	}
	assert.Equal(t, exp, state.tickerPrice("Deribit", pair, asset.Futures), "tickerPrice should map the merged state, leaving the ask the state lacks at zero")
}

func TestWebsocketHandleTrades(t *testing.T) {
	t.Parallel()
	// Trimmed from live trades.BTC-PERPETUAL.100ms and trades.option.BTC.100ms notifications
	const trades = `[{"timestamp":1792107442821,"price":82143.5,"direction":"buy","index_price":82119.59,"instrument_name":"BTC-PERPETUAL","trade_seq":306494199,"amount":10.0,"mark_price":82142.62,"tick_direction":0,"starbase_match_id":236633583815503872,"trade_id":"452873170","contracts":1.0,"starbase_timestamp":1792107442821580489},{"timestamp":1792107567601,"iv":41.34,"price":0.026,"direction":"sell","index_price":82119.46,"instrument_name":"BTC-16OCT26-80000-C","trade_seq":42,"amount":1.0,"mark_price":0.02624559,"tick_direction":2,"trade_id":"452873286","contracts":1.0}]`
	perp := currency.NewPairWithDelimiter("BTC", "PERPETUAL", currency.DashDelimiter)
	option, err := currency.NewPairFromString("BTC-16OCT26-80000-C")
	require.NoError(t, err, "NewPairFromString must not error")

	ex := newTestExchange(t)
	ex.SetTradeFeedStatus(true)
	ex.SetSaveTradeDataStatus(false)
	require.NoError(t, ex.wsHandleTrades(t.Context(), notification("trades.future.BTC.100ms", trades)), "wsHandleTrades must not error")
	exp := []trade.Data{
		{TID: "452873170", Exchange: ex.Name, CurrencyPair: perp, AssetType: asset.Futures, Side: order.Buy, Price: 82143.5, Amount: 10, Timestamp: time.UnixMilli(1792107442821)},
		{TID: "452873286", Exchange: ex.Name, CurrencyPair: option, AssetType: asset.Options, Side: order.Sell, Price: 0.026, Amount: 1, Timestamp: time.UnixMilli(1792107567601)},
	}
	for i := range exp {
		assert.Equalf(t, exp[i], relayed(t, ex), "wsHandleTrades should relay trade %d", i)
	}

	// A trade that cannot be converted is reported, and the rest still relayed
	err = ex.wsHandleTrades(t.Context(), notification("trades.future.any.100ms", `[{"instrument_name":"BTC","direction":"buy"},{"instrument_name":"BTC-PERPETUAL","direction":"sideways"},{"instrument_name":"BTC-PERPETUAL","direction":"sell","trade_id":"1","price":1,"amount":2,"timestamp":1792107442821}]`))
	assert.ErrorIs(t, err, errUnsupportedInstrumentFormat, "wsHandleTrades should report a trade whose instrument cannot be resolved")
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "wsHandleTrades should report a trade whose direction cannot be converted")
	assert.Equal(t, trade.Data{TID: "1", Exchange: ex.Name, CurrencyPair: perp, AssetType: asset.Futures, Side: order.Sell, Price: 1, Amount: 2, Timestamp: time.UnixMilli(1792107442821)}, relayed(t, ex), "wsHandleTrades should relay the trades it converts")
	assertNothingRelayed(t, ex)
	assert.ErrorIs(t, ex.wsHandleTrades(t.Context(), notification("trades.future.any.100ms", `{}`)), errMalformedNotification, "wsHandleTrades should reject malformed trades")

	disabled := newTestExchange(t)
	disabled.SetTradeFeedStatus(false)
	disabled.SetSaveTradeDataStatus(false)
	require.NoError(t, disabled.wsHandleTrades(t.Context(), notification("trades.future.BTC.100ms", trades)), "wsHandleTrades must not error with the trade feed off")
	assertNothingRelayed(t, disabled)
}

func TestWebsocketHandleCandle(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	const channel = "chart.trades.BTC-PERPETUAL.1"
	require.NoError(t, ex.wsHandleCandle(t.Context(), channel, notification(channel, `{"close":82143.0,"high":82149.5,"low":82143.0,"open":82145.0,"tick":1792107420000,"cost":64330.0,"volume":0.78312766}`)), "wsHandleCandle must not error")
	exp := kline.Item{
		Pair:     currency.NewPairWithDelimiter("BTC", "PERPETUAL", currency.DashDelimiter),
		Asset:    asset.Futures,
		Exchange: ex.Name,
		Interval: kline.OneMin,
		Candles: []kline.Candle{{
			Time:        time.UnixMilli(1792107420000),
			Open:        82145,
			High:        82149.5,
			Low:         82143,
			Close:       82143,
			Volume:      0.78312766,
			QuoteVolume: 64330,
		}},
	}
	assert.Equal(t, exp, relayed(t, ex), "wsHandleCandle should relay the candle, its cost as the quote volume, with the interval its channel's resolution names")

	for channel, expErr := range map[string]error{
		"chart.trades":                      errMalformedNotification,
		"chart.trades.BTC-PERPETUAL.2":      kline.ErrUnsupportedInterval,
		"chart.trades.BTC-PERPETUAL.weekly": kline.ErrInvalidInterval,
		"chart.trades.BTC.1":                errUnsupportedInstrumentFormat,
	} {
		assert.ErrorIsf(t, ex.wsHandleCandle(t.Context(), channel, notification(channel, `{}`)), expErr, "wsHandleCandle should reject channel %s", channel)
	}
	assert.ErrorIs(t, ex.wsHandleCandle(t.Context(), channel, notification(channel, `[]`)), errMalformedNotification, "wsHandleCandle should reject a malformed candle")
	assertNothingRelayed(t, ex)
}

func TestResolutionToInterval(t *testing.T) {
	t.Parallel()
	for resolution, exp := range map[string]kline.Interval{
		"1": kline.OneMin, "3": kline.ThreeMin, "5": kline.FiveMin, "10": kline.TenMin, "15": kline.FifteenMin, "30": kline.ThirtyMin,
		"60": kline.OneHour, "120": kline.TwoHour, "180": kline.ThreeHour, "360": kline.SixHour, "720": kline.TwelveHour, "1D": kline.OneDay,
	} {
		interval, err := resolutionToInterval(resolution)
		require.NoErrorf(t, err, "resolutionToInterval must not error for %s", resolution)
		assert.Equalf(t, exp, interval, "resolutionToInterval should convert %s", resolution)
	}
	_, err := resolutionToInterval("2")
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "resolutionToInterval should reject a resolution Deribit does not serve")
	_, err = resolutionToInterval("1W")
	assert.ErrorIs(t, err, kline.ErrInvalidInterval, "resolutionToInterval should reject a resolution that is not a number of minutes")
}
