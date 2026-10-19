package coinbase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
)

func TestWebsocketAPISubscribe(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAPI().Subscribe(t.Context(), nil)
	require.ErrorIs(t, err, errDerivativesChannelsRequired, "WebsocketAPI.Subscribe must refuse no channels")

	ex, _ := newDerivativesWebsocketTestExchange(t)
	channels := []string{"platform_state.public_methods_state"}
	subscribed, err := ex.WebsocketAPI().Subscribe(t.Context(), channels)
	require.NoError(t, err, "WebsocketAPI.Subscribe must not error")
	assert.Equal(t, channels, subscribed, "WebsocketAPI.Subscribe should return the channels subscribed")
	unsubscribed, err := ex.WebsocketAPI().Unsubscribe(t.Context(), channels)
	require.NoError(t, err, "WebsocketAPI.Unsubscribe must not error")
	assert.Equal(t, channels, unsubscribed, "WebsocketAPI.Unsubscribe should return the channels unsubscribed")

	_, err = newTestExchange(t).WebsocketAPI().Subscribe(t.Context(), channels)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.Subscribe should error without the gateway's streams connected")
}

func TestWebsocketAPIUnsubscribe(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAPI().Unsubscribe(t.Context(), nil)
	assert.ErrorIs(t, err, errDerivativesChannelsRequired, "WebsocketAPI.Unsubscribe should refuse no channels")
}

func TestWebsocketAPISubscribePrivate(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAPI().SubscribePrivate(t.Context(), nil)
	require.ErrorIs(t, err, errDerivativesChannelsRequired, "WebsocketAPI.SubscribePrivate must refuse a nil request")
	_, err = e.WebsocketAPI().SubscribePrivate(t.Context(), &DerivativesPrivateSubscribeRequest{Channels: []string{"user.lock"}, Label: "seventeen-letters"})
	require.ErrorIs(t, err, errDerivativesSubscriptionLabelTooLong, "WebsocketAPI.SubscribePrivate must refuse a label longer than 16 characters")

	skipLiveWithoutCredentials(t)
	ex, venue := newDerivativesWebsocketTestExchange(t)
	req := &DerivativesPrivateSubscribeRequest{Channels: []string{"user.lock"}, Label: "gct"}
	subscribed, err := ex.WebsocketAPI().SubscribePrivate(t.Context(), req)
	require.NoError(t, err, "WebsocketAPI.SubscribePrivate must not error")
	assert.Equal(t, req.Channels, subscribed, "WebsocketAPI.SubscribePrivate should return the channels subscribed")
	if mockTests {
		requests := venue.requestsFor("private/subscribe")
		require.Len(t, requests, 1, "the mock must receive the private subscription")
		assert.JSONEq(t, `{"channels":["user.lock"],"label":"gct"}`, string(requests[0].Params), "WebsocketAPI.SubscribePrivate should send the channels and label")
	}
	unsubscribed, err := ex.WebsocketAPI().UnsubscribePrivate(t.Context(), req.Channels)
	require.NoError(t, err, "WebsocketAPI.UnsubscribePrivate must not error")
	assert.Equal(t, req.Channels, unsubscribed, "WebsocketAPI.UnsubscribePrivate should return the channels unsubscribed")

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().SubscribePrivate(t.Context(), req)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.SubscribePrivate should refuse an unauthenticated connection")
}

func TestWebsocketAPIUnsubscribePrivate(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAPI().UnsubscribePrivate(t.Context(), nil)
	assert.ErrorIs(t, err, errDerivativesChannelsRequired, "WebsocketAPI.UnsubscribePrivate should refuse no channels")
}

func TestWebsocketAPIUnsubscribeAll(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("the live tests share one connection, whose other tests' channels unsubscribing all would drop")
	}
	ex, venue := newDerivativesWebsocketTestExchange(t)
	require.NoError(t, ex.WebsocketAPI().UnsubscribeAll(t.Context()), "WebsocketAPI.UnsubscribeAll must not error")
	assert.Len(t, venue.requestsFor("public/unsubscribe_all"), 1, "WebsocketAPI.UnsubscribeAll should be sent on the gateway's streams")
	require.NoError(t, ex.WebsocketAPI().UnsubscribeAllPrivate(t.Context()), "WebsocketAPI.UnsubscribeAllPrivate must not error")
	assert.Len(t, venue.requestsFor("private/unsubscribe_all"), 1, "WebsocketAPI.UnsubscribeAllPrivate should be sent on the gateway's private connection")
}

func TestWebsocketAPIUnsubscribeAllResult(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("the mock gateway websocket answers with the result this test needs")
	}
	ex, venue := newDerivativesWebsocketTestExchange(t)
	venue.override("public/unsubscribe_all", json.RawMessage(`{"result":"not ok"}`))
	venue.override("private/unsubscribe_all", json.RawMessage(`{"result":"not ok"}`))
	assert.ErrorIs(t, ex.WebsocketAPI().UnsubscribeAll(t.Context()), errDerivativesUnexpectedResult, "WebsocketAPI.UnsubscribeAll should error for a result other than ok")
	assert.ErrorIs(t, ex.WebsocketAPI().UnsubscribeAllPrivate(t.Context()), errDerivativesUnexpectedResult, "WebsocketAPI.UnsubscribeAllPrivate should error for a result other than ok")
}
