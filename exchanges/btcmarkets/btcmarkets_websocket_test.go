package btcmarkets

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

type subscriptionConnection struct {
	websocket.Connection
	messages []WsSubscribe
	onSend   func() error
}

func (c *subscriptionConnection) SendJSONMessage(_ context.Context, _ request.EndpointLimit, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var message WsSubscribe
	if err := json.Unmarshal(raw, &message); err != nil {
		return err
	}
	c.messages = append(c.messages, message)
	if c.onSend != nil {
		callback := c.onSend
		c.onSend = nil
		return callback()
	}
	return nil
}

func newSubscriptionConnection(t *testing.T, e *Exchange) *subscriptionConnection {
	t.Helper()
	conn, err := e.Websocket.CreateTestConnection(asset.Spot)
	require.NoError(t, err, "managed test connection must initialise")
	capture := &subscriptionConnection{Connection: conn}
	require.NoError(t, e.Websocket.TrackTestConnection(asset.Spot, capture), "test connection must register")
	return capture
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	t.Run("initial message belongs to each connection", func(t *testing.T) {
		t.Parallel()
		e := new(Exchange)
		require.NoError(t, testexch.Setup(e), "exchange must initialise")
		first, second := newSubscriptionConnection(t, e), newSubscriptionConnection(t, e)
		firstSub := &subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: tick, Asset: asset.Spot, Pairs: currency.Pairs{spotTestPair}}
		require.NoError(t, e.Subscribe(t.Context(), first, subscription.List{firstSub}), "first subscription must succeed")
		require.NoError(t, first.Subscriptions().Add(firstSub), "manager-owned connection store must record subscription")
		secondSubs := subscription.List{
			{Channel: subscription.OrderbookChannel, QualifiedChannel: wsOrderbookUpdate, Asset: asset.Spot, Pairs: currency.Pairs{spotTestPair}},
			{Channel: subscription.HeartbeatChannel, QualifiedChannel: heartbeat},
		}
		require.NoError(t, e.Subscribe(t.Context(), second, secondSubs), "second connection subscriptions must succeed")
		require.Len(t, second.messages, 2, "different pair groups must produce two messages")
		assert.Equal(t, subscribe, second.messages[0].MessageType, "new socket should start with subscribe despite other active sockets")
		assert.Empty(t, second.messages[0].ClientType, "initial subscribe should omit clientType")
		assert.Equal(t, addSubscription, second.messages[1].MessageType, "next successful batch should add subscriptions")
		assert.Equal(t, clientType, second.messages[1].ClientType, "addSubscription should include clientType")
	})
	t.Run("private subscription is signed before sending", func(t *testing.T) {
		t.Parallel()
		e := new(Exchange)
		require.NoError(t, testexch.Setup(e), "exchange must initialise")
		e.SkipAuthCheck = true
		e.SetCredentials(&accounts.Credentials{Key: "mock-key", Secret: "bW9jay1zZWNyZXQ="})
		conn := newSubscriptionConnection(t, e)
		require.NoError(t, e.Subscribe(t.Context(), conn, subscription.List{{Channel: subscription.MyOrdersChannel, QualifiedChannel: orderChange, Authenticated: true}}), "private subscription must succeed")
		require.Len(t, conn.messages, 1, "private request must be sent once")
		assert.Equal(t, "mock-key", conn.messages[0].Key, "private request should include credentials")
		assert.NotEmpty(t, conn.messages[0].Signature, "private request should already be signed")
		assert.NotEmpty(t, conn.messages[0].Timestamp, "private request should include its signed timestamp")
	})
}

func TestUnsubscribe(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "exchange must initialise")
	conn := newSubscriptionConnection(t, e)
	sub := &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: wsOrderbookUpdate, Asset: asset.Spot, Pairs: currency.Pairs{spotTestPair}}
	require.NoError(t, e.Websocket.AddSuccessfulSubscriptions(conn, sub), "subscription must register")
	require.NoError(t, e.Unsubscribe(t.Context(), conn, subscription.List{sub}), "unsubscription must succeed")
	require.Len(t, conn.messages, 1, "one unsubscription must be sent")
	assert.Equal(t, []string{wsOrderbookUpdate}, conn.messages[0].Channels, "unsubscription should use the wire channel")
	assert.Nil(t, e.Websocket.GetSubscription(sub), "subscription should leave its managed store")
}

func TestReSubscribeSpecificOrderbook(t *testing.T) {
	t.Parallel()
	t.Run("legacy entry point resolves the managed connection", func(t *testing.T) {
		t.Parallel()
		e := new(Exchange)
		require.NoError(t, testexch.Setup(e), "exchange must initialise")
		require.ErrorIs(t, e.ReSubscribeSpecificOrderbook(spotTestPair), websocket.ErrNotConnected, "legacy entry point must reject disconnected use")

		server, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
		}))
		e.Websocket = websocket.NewManager()
		require.NoError(t, e.Websocket.Setup(&websocket.ManagerSetup{
			UseMultiConnectionManagement: true,
			ExchangeConfig:               &config.Exchange{Name: e.Name, Features: &config.FeaturesConfig{Enabled: config.FeaturesEnabledConfig{Websocket: true}}, WebsocketTrafficTimeout: time.Hour},
			Features:                     &e.Features.Supports.WebsocketCapabilities,
		}), "test manager must initialise")
		sub := &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: wsOrderbookUpdate, Asset: asset.Spot, Pairs: currency.Pairs{spotTestPair}}
		require.NoError(t, e.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
			URL:                   "ws" + server.URL[len("http"):],
			MessageFilter:         asset.Spot,
			Connector:             func(ctx context.Context, conn websocket.Connection) error { return conn.Dial(ctx, dialer, nil, nil) },
			GenerateSubscriptions: func() (subscription.List, error) { return subscription.List{sub}, nil },
			Subscriber:            e.Subscribe,
			Unsubscriber:          e.Unsubscribe,
			Handler:               func(context.Context, websocket.Connection, []byte) error { return nil },
		}), "test connection must register")
		t.Cleanup(func() {
			assert.NoError(t, e.Websocket.Disable(), "manager should disable")
			assert.NoError(t, e.Websocket.Shutdown(), "manager should close the socket")
		})
		require.NoError(t, e.WsConnect(), "legacy connect entry point must start managed lifecycle")
		conn, err := e.Websocket.GetConnection(asset.Spot)
		require.NoError(t, err, "managed connection must be available")
		require.NoError(t, e.ReSubscribeSpecificOrderbook(spotTestPair), "legacy entry point must resolve the owning connection")
		assert.Same(t, sub, conn.Subscriptions().Get(sub), "legacy recovery should preserve ownership")
		require.ErrorIs(t, e.ReSubscribeSpecificOrderbook(currency.NewPair(currency.ETH, currency.AUD)), subscription.ErrNotFound, "legacy entry point must not refresh an unowned book")
	})

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "exchange must initialise")
	other, owner := newSubscriptionConnection(t, e), newSubscriptionConnection(t, e)
	sub := &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: wsOrderbookUpdate, Asset: asset.Spot, Pairs: currency.Pairs{spotTestPair, currency.NewPair(currency.ETH, currency.AUD)}}
	require.NoError(t, e.Websocket.AddSuccessfulSubscriptions(owner, sub), "grouped subscription must register")
	require.NoError(t, owner.Subscriptions().Add(sub), "connection must own its grouped subscription")
	require.NoError(t, e.reSubscribeSpecificOrderbook(t.Context(), owner, spotTestPair), "owning connection must refresh the book")
	require.Len(t, owner.messages, 2, "refresh must remove then add the affected pair")
	assert.Equal(t, removeSubscription, owner.messages[0].MessageType, "refresh should unsubscribe first")
	assert.Equal(t, addSubscription, owner.messages[1].MessageType, "refresh should add to the existing session")
	for _, message := range owner.messages {
		assert.Equal(t, []string{wsOrderbookUpdate}, message.Channels, "refresh should retain the wire channel")
		assert.Equal(t, []string{spotTestPair.String()}, message.MarketIDs, "refresh should affect only the requested pair")
	}
	assert.Empty(t, other.messages, "other connection should receive no refresh traffic")
	assert.Same(t, sub, owner.Subscriptions().Get(sub), "connection should retain the original grouped subscription")
	assert.Same(t, sub, e.Websocket.GetSubscription(sub), "manager should retain the original grouped subscription")
	t.Run("response before ownership is copied", func(t *testing.T) {
		t.Parallel()
		e := new(Exchange)
		require.NoError(t, testexch.Setup(e), "exchange must initialise")
		conn := newSubscriptionConnection(t, e)
		sub := &subscription.Subscription{Channel: subscription.OrderbookChannel, QualifiedChannel: wsOrderbookUpdate, Asset: asset.Spot, Pairs: currency.Pairs{spotTestPair}}
		conn.onSend = func() error {
			assert.Empty(t, conn.Subscriptions().List(), "response should arrive before ownership is copied")
			return e.reSubscribeSpecificOrderbook(t.Context(), conn, spotTestPair)
		}
		require.NoError(t, e.Subscribe(t.Context(), conn, subscription.List{sub}), "early orderbook recovery must not depend on ownership bookkeeping")
		require.Len(t, conn.messages, 3, "initial subscription must be followed by removal and recovery")
		assert.Equal(t, removeSubscription, conn.messages[1].MessageType, "early recovery should remove the affected pair")
		assert.Equal(t, addSubscription, conn.messages[2].MessageType, "early recovery should add the affected pair")
		assert.Same(t, sub, e.Websocket.GetSubscription(sub), "subscription callback should retain the original manager entry")
		assert.Empty(t, conn.Subscriptions().List(), "recovery should not create premature connection ownership")
		require.NoError(t, conn.Subscriptions().Add(sub), "manager must still be able to copy ownership without duplicates")
	})
}

func TestWsConnect(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "exchange must initialise")
	require.False(t, e.Websocket.IsEnabled(), "test setup must leave websocket disabled")
	require.ErrorIs(t, e.WsConnect(), websocket.ErrWebsocketNotEnabled, "legacy connector must use the manager lifecycle")
}
