package htx

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestWSSubscriptionKeyMatch(t *testing.T) {
	t.Parallel()
	h := new(Exchange)
	require.NoError(t, testexch.Setup(h), "exchange setup must succeed")
	conn, err := h.Websocket.CreateTestConnection(exchange.WebsocketSpot)
	require.NoError(t, err, "first connection must be created")
	other, err := h.Websocket.CreateTestConnection(exchange.WebsocketSpot)
	require.NoError(t, err, "second connection must be created")
	sub := &subscription.Subscription{Asset: asset.Spot, Channel: subscription.TickerChannel, QualifiedChannel: "market.btcusdt.detail", Pairs: currency.Pairs{btcusdtPair}}
	key := wsSubscriptionKey{ExactKey: &subscription.ExactKey{Subscription: sub}, connection: conn}
	for _, tc := range []struct {
		name  string
		query wsSubscriptionKey
		other subscription.MatchableKey
		want  bool
	}{
		{name: "same connection and topic", query: key, other: key, want: true},
		{name: "different connection", query: key, other: wsSubscriptionKey{ExactKey: key.ExactKey, connection: other}},
		{name: "nil connection cannot select managed topic", query: wsSubscriptionKey{ExactKey: key.ExactKey}, other: key},
		{name: "different topic", query: key, other: wsSubscriptionKey{ExactKey: &subscription.ExactKey{Subscription: &subscription.Subscription{QualifiedChannel: "orders.*"}}, connection: conn}},
		{name: "nil query key", query: wsSubscriptionKey{}, other: key},
		{name: "nil query subscription", query: wsSubscriptionKey{ExactKey: &subscription.ExactKey{}}, other: key},
		{name: "nil candidate key", query: key, other: wsSubscriptionKey{}},
		{name: "nil candidate subscription", query: key, other: wsSubscriptionKey{ExactKey: &subscription.ExactKey{}}},
		{name: "ordinary key", query: key, other: key.ExactKey},
		{name: "nil candidate", query: key},
		{name: "typed nil candidate", query: key, other: (*wsSubscriptionKey)(nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.query.Match(tc.other), "topic lookup should remain scoped to its connection")
		})
	}
	t.Run("regenerated settings retain registration", func(t *testing.T) {
		t.Parallel()
		store := subscription.NewStore()
		registered := sub.Clone()
		registered.SetKey(wsSubscriptionKey{ExactKey: &subscription.ExactKey{Subscription: registered}, connection: conn})
		require.NoError(t, store.Add(registered), "scoped subscription must register")
		fresh := registered.Clone()
		fresh.SetKey(nil)
		adds, removes := store.Diff(subscription.List{fresh})
		assert.Empty(t, adds, "equivalent regenerated settings should not subscribe again")
		assert.Empty(t, removes, "equivalent regenerated settings should not unsubscribe")
		fresh.Levels++
		adds, removes = store.Diff(subscription.List{fresh})
		assert.Len(t, adds, 1, "changed settings should require a new subscription")
		assert.Len(t, removes, 1, "changed settings should retire the previous subscription")
	})
}

func TestWSHandleSubscriptionResponse(t *testing.T) {
	t.Parallel()
	for _, protocol := range []string{"public", "spot private", "derivative"} {
		for _, operation := range []string{wsSubOp, wsUnsubOp} {
			for _, rejected := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/rejected=%t", protocol, operation, rejected), func(t *testing.T) {
					t.Parallel()
					h := new(Exchange)
					require.NoError(t, testexch.Setup(h), "exchange setup must succeed")
					h.Name = t.Name()
					conn, err := h.Websocket.CreateTestConnection(exchange.WebsocketSpot)
					require.NoError(t, err, "connection must be created")
					require.NoError(t, h.Websocket.TrackTestConnection(exchange.WebsocketSpot, conn), "connection must be tracked")
					channel := "market.btcusdt.detail"
					sub := &subscription.Subscription{Asset: asset.Spot, Channel: subscription.TickerChannel, QualifiedChannel: channel, Pairs: currency.Pairs{btcusdtPair}}
					sub.SetKey(wsSubscriptionKey{ExactKey: &subscription.ExactKey{Subscription: sub}, connection: conn})
					require.NoError(t, h.Websocket.AddSubscriptions(conn, sub), "pending subscription must register")
					if operation == wsUnsubOp {
						require.NoError(t, sub.SetState(subscription.SubscribedState), "unsubscribe must start with an accepted subscription")
					}
					responses, err := conn.MatchReturnResponses(t.Context(), operation+":"+channel, 1)
					require.NoError(t, err, "acknowledgement waiter must register")
					status, code := "ok", 200
					if rejected {
						status, code = "error", 1000
					}
					var raw string
					switch protocol {
					case "public":
						raw = fmt.Sprintf(`{"id":%q,"status":%q,"code":%d}`, operation+":"+channel, status, code)
					case "spot private":
						raw = fmt.Sprintf(`{"action":%q,"ch":%q,"code":%d}`, operation, channel, code)
					case "derivative":
						raw = fmt.Sprintf(`{"op":%q,"topic":%q,"code":%d}`, operation, channel, code)
					}
					// Process the next push without waiting for the request goroutine or batch to finish.
					require.NoError(t, h.wsHandleData(t.Context(), conn, []byte(raw)), "acknowledgement must update registration before returning")
					retained := operation == wsSubOp && !rejected || operation == wsUnsubOp && rejected
					if retained {
						assert.Equal(t, subscription.SubscribedState, sub.State(), "accepted subscription should already be active")
					} else {
						assert.Equal(t, subscription.UnsubscribedState, sub.State(), "removed subscription should already be inactive")
					}
					assert.Empty(t, conn.Subscriptions().List(), "routing should not depend on batch connection bookkeeping")
					err = h.wsHandleData(t.Context(), conn, []byte(`{"ch":"market.btcusdt.detail","tick":{"close":100}}`))
					if retained {
						require.NoError(t, err, "immediate push must resolve its accepted subscription")
						stored, err := ticker.GetTicker(h.Name, btcusdtPair, asset.Spot)
						require.NoError(t, err, "immediate ticker must reach the shared store")
						assert.Equal(t, 100.0, stored.Last, "immediate push should retain its price")
					} else {
						require.ErrorIs(t, err, subscription.ErrNotFound, "push for removed subscription must not route")
					}
					select {
					case response := <-responses:
						require.NoError(t, response.Err, "acknowledgement must reach the request waiter")
						assert.Equal(t, []byte(raw), response.Responses[0], "request waiter should receive the original reply")
					case <-time.After(time.Second):
						require.FailNow(t, "request waiter must receive its acknowledgement")
					}
				})
			}
		}
	}
	require.ErrorIs(t, new(Exchange).wsHandleSubscriptionResponse(nil, wsSubOp, "orders.*", nil), common.ErrNilPointer, "acknowledgement must require a connection")
	var typedNil *struct{ websocket.Connection }
	require.ErrorIs(t, new(Exchange).wsHandleSubscriptionResponse(typedNil, wsSubOp, "orders.*", nil), common.ErrNilPointer, "acknowledgement must reject a typed nil connection")
}
