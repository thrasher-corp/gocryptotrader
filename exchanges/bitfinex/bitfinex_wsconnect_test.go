package bitfinex

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

type wsConnectFixtureConnection struct {
	websocket.Connection
	dialErr     error
	sendErr     error
	responseRaw []byte
	sent        []any

	dialCalls     atomic.Int32
	readCalls     atomic.Int32
	sendJSONCalls atomic.Int32
}

func (f *wsConnectFixtureConnection) Dial(context.Context, *gws.Dialer, http.Header, url.Values) error {
	f.dialCalls.Add(1)
	return f.dialErr
}

func (f *wsConnectFixtureConnection) ReadMessage() websocket.Response {
	f.readCalls.Add(1)
	return websocket.Response{}
}

func (f *wsConnectFixtureConnection) SendJSONMessage(_ context.Context, _ request.EndpointLimit, payload any) error {
	f.sendJSONCalls.Add(1)
	f.sent = append(f.sent, payload)
	return f.sendErr
}

func (f *wsConnectFixtureConnection) SendMessageReturnResponse(_ context.Context, _ request.EndpointLimit, _, payload any) ([]byte, error) {
	f.sent = append(f.sent, payload)
	return f.responseRaw, f.sendErr
}

func TestWsConnect(t *testing.T) {
	t.Parallel()

	t.Run("success configures connection", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		conn := &wsConnectFixtureConnection{}

		require.NoError(t, ex.wsConnect(t.Context(), conn), "wsConnect must not error")
		assert.Equal(t, int32(1), conn.dialCalls.Load(), "connection should dial once")
		assert.Equal(t, int32(1), conn.sendJSONCalls.Load(), "wsConnect should configure the connection")
	})

	t.Run("dial failure skips configuration", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		errDialFailed := errors.New("dial failed")
		conn := &wsConnectFixtureConnection{dialErr: errDialFailed}

		err := ex.wsConnect(t.Context(), conn)
		require.ErrorIs(t, err, errDialFailed, "wsConnect must return the dial error")
		assert.Equal(t, int32(1), conn.dialCalls.Load(), "connection should dial once")
		assert.Equal(t, int32(0), conn.sendJSONCalls.Load(), "ConfigureWS should not send when dial fails")
	})
}

func TestConfigureWS(t *testing.T) {
	t.Parallel()

	errSend := errors.New("send failure")
	conn := &wsConnectFixtureConnection{sendErr: errSend}
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")

	assert.ErrorIs(t, ex.ConfigureWS(t.Context(), conn), errSend, "ConfigureWS should return the send failure")
	assert.Equal(t, int32(1), conn.sendJSONCalls.Load(), "ConfigureWS should send one request")
}

func TestGeneratePublicSubscriptions(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	subs, err := ex.generatePublicSubscriptions()
	require.NoError(t, err, "generatePublicSubscriptions must not error")
	require.NotEmpty(t, subs, "generatePublicSubscriptions must return subscriptions")
	for _, sub := range subs {
		assert.False(t, sub.Authenticated, "generatePublicSubscriptions should return only public subscriptions")
	}
}

func TestGeneratePrivateSubscriptions(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	subs, err := ex.generatePrivateSubscriptions()
	require.NoError(t, err, "generatePrivateSubscriptions must not error")
	assert.Empty(t, subs, "generatePrivateSubscriptions should return no subscriptions when Bitfinex has no explicit private channels")
}

func TestSubscribeToChan(t *testing.T) {
	t.Parallel()

	t.Run("requires one subscription", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		assert.ErrorIs(t, ex.subscribeToChan(t.Context(), testexch.GetMockConn(t, ex, ""), nil), subscription.ErrNotSinglePair, "subscribeToChan should require one subscription")
	})

	t.Run("rejects invalid qualified channel", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		err := ex.subscribeToChan(t.Context(), testexch.GetMockConn(t, ex, ""), subscription.List{{QualifiedChannel: "{"}})
		require.Error(t, err, "subscribeToChan must reject invalid JSON")
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		conn := &wsConnectFixtureConnection{
			Connection:  testexch.GetMockConn(t, ex, authenticatedBitfinexWebsocketEndpoint),
			responseRaw: []byte(`{"event":"subscribed"}`),
		}
		sub := &subscription.Subscription{QualifiedChannel: `{"channel":"ticker","symbol":"tBTCUSD"}`}

		require.NoError(t, ex.subscribeToChan(t.Context(), conn, subscription.List{sub}), "subscribeToChan must not error")
		assert.NotNil(t, ex.Websocket.GetSubscription(sub.Key), "subscribeToChan should store the temporary subscription")
	})
}

func TestUnsubscribeFromChan(t *testing.T) {
	t.Parallel()

	t.Run("rejects batching", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		assert.ErrorIs(t, ex.unsubscribeFromChan(t.Context(), testexch.GetMockConn(t, ex, ""), nil), subscription.ErrBatchingNotSupported, "unsubscribeFromChan should reject batching")
	})

	t.Run("rejects non-integer key", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		err := ex.unsubscribeFromChan(t.Context(), testexch.GetMockConn(t, ex, ""), subscription.List{{Key: "invalid"}})
		require.Error(t, err, "unsubscribeFromChan must reject a non-integer key")
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		conn := &wsConnectFixtureConnection{
			Connection:  testexch.GetMockConn(t, ex, authenticatedBitfinexWebsocketEndpoint),
			responseRaw: []byte(`{"event":"unsubscribed"}`),
		}
		sub := &subscription.Subscription{Key: 42}
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, sub), "AddSuccessfulSubscriptions must not error")

		require.NoError(t, ex.unsubscribeFromChan(t.Context(), conn, subscription.List{sub}), "unsubscribeFromChan must not error")
		assert.Nil(t, ex.Websocket.GetSubscription(42), "unsubscribeFromChan should remove the subscription")
	})
}

func TestSubscribeForConnection(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	conn := &wsConnectFixtureConnection{
		Connection:  testexch.GetMockConn(t, ex, authenticatedBitfinexWebsocketEndpoint),
		responseRaw: []byte(`{"event":"subscribed"}`),
	}
	sub := &subscription.Subscription{
		Asset:            asset.Spot,
		Channel:          subscription.TickerChannel,
		Pairs:            currency.Pairs{currency.NewBTCUSD()},
		QualifiedChannel: `{"channel":"ticker","symbol":"tBTCUSD"}`,
	}

	require.NoError(t, ex.subscribeForConnection(t.Context(), conn, subscription.List{sub}), "subscribeForConnection must not error")
	assert.Len(t, conn.sent, 1, "subscribeForConnection should send one request")
	assert.NotNil(t, ex.Websocket.GetSubscription(sub.Key), "subscribeForConnection should store a temporary subscription")
}

func TestUnsubscribeForConnection(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	conn := &wsConnectFixtureConnection{
		Connection:  testexch.GetMockConn(t, ex, authenticatedBitfinexWebsocketEndpoint),
		responseRaw: []byte(`{"event":"unsubscribed"}`),
	}
	sub := &subscription.Subscription{Key: 42, QualifiedChannel: `{"channel":"ticker","symbol":"tBTCUSD"}`}
	require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, sub), "AddSuccessfulSubscriptions must not error")

	require.NoError(t, ex.unsubscribeForConnection(t.Context(), conn, subscription.List{sub}), "unsubscribeForConnection must not error")
	assert.Len(t, conn.sent, 1, "unsubscribeForConnection should send one request")
	assert.Nil(t, ex.Websocket.GetSubscription(42), "unsubscribeForConnection should remove the subscription")
}

func TestWsSendAuthConn(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials("key", "secret", "", "", "", "")
	conn := &wsConnectFixtureConnection{}

	require.NoError(t, ex.wsSendAuthConn(t.Context(), conn), "wsSendAuthConn must not error")
	assert.Equal(t, int32(1), conn.sendJSONCalls.Load(), "wsSendAuthConn should send one request")
}

func TestResubOrderbook(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	assert.ErrorIs(t, ex.resubOrderbook(t.Context(), testexch.GetMockConn(t, ex, ""), nil), common.ErrNilPointer, "resubOrderbook should reject a nil subscription")
	assert.ErrorIs(t, ex.resubOrderbook(t.Context(), testexch.GetMockConn(t, ex, ""), &subscription.Subscription{}), subscription.ErrNotSinglePair, "resubOrderbook should require exactly one pair")
}
