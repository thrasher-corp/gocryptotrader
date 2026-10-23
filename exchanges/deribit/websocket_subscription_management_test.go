package deribit

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
)

// testPublicChannel is a public channel every subscription test can subscribe to without disturbing the others, which
// subscribe to their own
const testPublicChannel = "deribit_price_index.btc_usd"

func TestWebsocketAPISubscribe(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	_, err := ex.WebsocketAPI().Subscribe(t.Context(), nil)
	require.ErrorIs(t, err, errChannelsRequired, "WebsocketAPI.Subscribe must reject an empty channel list")

	channels := []string{"deribit_price_index.eth_usd"}
	result, err := ex.WebsocketAPI().Subscribe(t.Context(), channels)
	require.NoError(t, err, "WebsocketAPI.Subscribe must not error")
	assert.Equal(t, channels, result, "WebsocketAPI.Subscribe should return the subscribed channels")
	result, err = ex.WebsocketAPI().Unsubscribe(t.Context(), channels)
	require.NoError(t, err, "WebsocketAPI.Unsubscribe must not error")
	assert.Equal(t, channels, result, "WebsocketAPI.Unsubscribe should return the unsubscribed channels")

	_, err = newTestExchange(t).WebsocketAPI().Subscribe(t.Context(), channels)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.Subscribe should error without a websocket connection")
}

func TestWebsocketAPISubscribePrivate(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, venue := newWebsocketTestExchange(t)
	_, err := ex.WebsocketAPI().SubscribePrivate(t.Context(), nil)
	require.ErrorIs(t, err, errChannelsRequired, "WebsocketAPI.SubscribePrivate must reject a nil request")
	_, err = ex.WebsocketAPI().SubscribePrivate(t.Context(), &PrivateSubscribeRequest{})
	require.ErrorIs(t, err, errChannelsRequired, "WebsocketAPI.SubscribePrivate must reject an empty channel list")
	_, err = ex.WebsocketAPI().SubscribePrivate(t.Context(), &PrivateSubscribeRequest{Channels: []string{"user.lock"}, Label: strings.Repeat("a", 17)})
	require.ErrorIs(t, err, errSubscriptionLabelTooLong, "WebsocketAPI.SubscribePrivate must reject a label over 16 characters")

	channels := []string{"user.lock"}
	result, err := ex.WebsocketAPI().SubscribePrivate(t.Context(), &PrivateSubscribeRequest{Channels: channels, Label: "gct"})
	require.NoError(t, err, "WebsocketAPI.SubscribePrivate must not error")
	assert.Equal(t, channels, result, "WebsocketAPI.SubscribePrivate should return the subscribed channels")
	if mockTests {
		requests := venue.received("private/subscribe")
		require.Len(t, requests, 1, "WebsocketAPI.SubscribePrivate must send one request")
		assert.JSONEq(t, `{"channels":["user.lock"],"label":"gct"}`, string(requests[0].Params), "WebsocketAPI.SubscribePrivate should send the channels and label")
	}
	result, err = ex.WebsocketAPI().UnsubscribePrivate(t.Context(), channels)
	require.NoError(t, err, "WebsocketAPI.UnsubscribePrivate must not error")
	assert.Equal(t, channels, result, "WebsocketAPI.UnsubscribePrivate should return the unsubscribed channels")

	_, err = newTestExchange(t).WebsocketAPI().SubscribePrivate(t.Context(), &PrivateSubscribeRequest{Channels: channels})
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.SubscribePrivate should error without an authenticated connection")
}

func TestWebsocketAPIUnsubscribe(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	_, err := ex.WebsocketAPI().Unsubscribe(t.Context(), nil)
	require.ErrorIs(t, err, errChannelsRequired, "WebsocketAPI.Unsubscribe must reject an empty channel list")

	channels := []string{testPublicChannel}
	_, err = ex.WebsocketAPI().Subscribe(t.Context(), channels)
	require.NoError(t, err, "WebsocketAPI.Subscribe must not error")
	result, err := ex.WebsocketAPI().Unsubscribe(t.Context(), channels)
	require.NoError(t, err, "WebsocketAPI.Unsubscribe must not error")
	assert.Equal(t, channels, result, "WebsocketAPI.Unsubscribe should return the unsubscribed channels")

	_, err = newTestExchange(t).WebsocketAPI().Unsubscribe(t.Context(), channels)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.Unsubscribe should error without a websocket connection")
}

func TestWebsocketAPIUnsubscribePrivate(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	_, err := ex.WebsocketAPI().UnsubscribePrivate(t.Context(), nil)
	require.ErrorIs(t, err, errChannelsRequired, "WebsocketAPI.UnsubscribePrivate must reject an empty channel list")

	channels := []string{"user.access_log"}
	_, err = ex.WebsocketAPI().SubscribePrivate(t.Context(), &PrivateSubscribeRequest{Channels: channels})
	require.NoError(t, err, "WebsocketAPI.SubscribePrivate must not error")
	result, err := ex.WebsocketAPI().UnsubscribePrivate(t.Context(), channels)
	require.NoError(t, err, "WebsocketAPI.UnsubscribePrivate must not error")
	assert.Equal(t, channels, result, "WebsocketAPI.UnsubscribePrivate should return the unsubscribed channels")

	_, err = newTestExchange(t).WebsocketAPI().UnsubscribePrivate(t.Context(), channels)
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.UnsubscribePrivate should error without an authenticated connection")
}

func TestWebsocketAPIUnsubscribeAll(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.UnsubscribeAll would unsubscribe the channels other live tests subscribe on the shared connection")
	}
	ex, venue := newWebsocketTestExchange(t)
	require.NoError(t, ex.WebsocketAPI().UnsubscribeAll(t.Context()), "WebsocketAPI.UnsubscribeAll must not error")
	assert.Len(t, venue.received("public/unsubscribe_all"), 1, "WebsocketAPI.UnsubscribeAll should send one request")

	err := newTestExchange(t).WebsocketAPI().UnsubscribeAll(t.Context())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.UnsubscribeAll should error without a websocket connection")
}

func TestWebsocketAPIUnsubscribeAllPrivate(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("WebsocketAPI.UnsubscribeAllPrivate would unsubscribe the channels other live tests subscribe on the shared connection")
	}
	ex, venue := newWebsocketTestExchange(t)
	require.NoError(t, ex.WebsocketAPI().UnsubscribeAllPrivate(t.Context()), "WebsocketAPI.UnsubscribeAllPrivate must not error")
	assert.Len(t, venue.received("private/unsubscribe_all"), 1, "WebsocketAPI.UnsubscribeAllPrivate should send one request")

	err := newTestExchange(t).WebsocketAPI().UnsubscribeAllPrivate(t.Context())
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.UnsubscribeAllPrivate should error without an authenticated connection")
}
