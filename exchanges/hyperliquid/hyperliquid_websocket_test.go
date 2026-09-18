package hyperliquid

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	ws "github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

var errWebsocketFixture = errors.New("websocket fixture failure")

type websocketConnectionFixture struct {
	ws.Connection

	mu       sync.Mutex
	dialErr  error
	sendErr  error
	sendHook func(websocketRequest, error)
	messages []ws.Response
	sent     []websocketRequest
	ping     ws.PingHandler
	url      string

	dialCalls     atomic.Int32
	readCalls     atomic.Int32
	pingCalls     atomic.Int32
	shutdownCalls atomic.Int32
}

func (f *websocketConnectionFixture) Dial(context.Context, *gws.Dialer, http.Header, url.Values) error {
	f.dialCalls.Add(1)
	return f.dialErr
}

func (f *websocketConnectionFixture) ReadMessage() ws.Response {
	f.readCalls.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.messages) == 0 {
		return ws.Response{}
	}
	message := f.messages[0]
	f.messages = f.messages[1:]
	return message
}

func (f *websocketConnectionFixture) SetupPingHandler(_ request.EndpointLimit, handler ws.PingHandler) {
	f.pingCalls.Add(1)
	f.mu.Lock()
	f.ping = handler
	f.mu.Unlock()
}

func (f *websocketConnectionFixture) SendJSONMessage(_ context.Context, _ request.EndpointLimit, payload any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	wsRequest, ok := payload.(websocketRequest)
	if !ok {
		return common.GetTypeAssertError("websocketRequest", payload)
	}
	if f.sendHook != nil {
		f.sendHook(wsRequest, f.sendErr)
	}
	if f.sendErr != nil {
		return f.sendErr
	}
	f.sent = append(f.sent, wsRequest)
	return nil
}

func (f *websocketConnectionFixture) SetURL(value string) {
	f.mu.Lock()
	f.url = value
	f.mu.Unlock()
}

func (f *websocketConnectionFixture) GetURL() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.url
}

func (f *websocketConnectionFixture) Shutdown() error {
	f.shutdownCalls.Add(1)
	return nil
}

func (f *websocketConnectionFixture) sentRequests() []websocketRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]websocketRequest(nil), f.sent...)
}

func waitForHyperliquidWebsocketReaders(t *testing.T, group *sync.WaitGroup) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		group.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Websocket readers did not exit")
	}
}

func newWebsocketConnectTestExchange(t *testing.T) *Exchange {
	t.Helper()
	ex := new(Exchange)
	ex.SetDefaults()
	cfg, err := ex.GetStandardConfig()
	require.NoError(t, err, "Getting websocket test config must not error")
	cfg.Features = &config.FeaturesConfig{Enabled: config.FeaturesEnabledConfig{Websocket: true}}
	require.NoError(t, ex.Setup(cfg), "Setting up websocket test exchange must not error")
	return ex
}

func newWebsocketHandlerTestExchange(t *testing.T) *Exchange {
	t.Helper()
	ex := newTradingTestExchange(t, nil, nil)
	ex.Name += "-" + strings.ReplaceAll(t.Name(), "/", "-")
	require.NoError(t, ex.UpdatePairs(currency.Pairs{testPerpetualPair}, asset.PerpetualContract, false), "Updating available perpetual test pairs must not error")
	require.NoError(t, ex.UpdatePairs(currency.Pairs{testPerpetualPair}, asset.PerpetualContract, true), "Updating enabled perpetual test pairs must not error")
	require.NoError(t, ex.UpdatePairs(currency.Pairs{testSpotPair}, asset.Spot, false), "Updating available spot test pairs must not error")
	require.NoError(t, ex.UpdatePairs(currency.Pairs{testSpotPair}, asset.Spot, true), "Updating enabled spot test pairs must not error")
	ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
	ex.SetFillsFeedStatus(true)
	ex.Websocket.Fills.Setup(true, ex.Websocket.DataHandler)
	return ex
}

func websocketAcknowledgement(t *testing.T, req *websocketRequest) []byte {
	t.Helper()
	response, err := json.Marshal(struct {
		Channel string                        `json:"channel"`
		Data    websocketSubscriptionResponse `json:"data"`
	}{
		Channel: wsChannelSubscriptionResponse,
		Data:    websocketSubscriptionResponse(*req),
	})
	require.NoError(t, err, "Encoding websocket acknowledgement must not error")
	return response
}

func installWebsocketAcknowledgements(t *testing.T, ex *Exchange, connection *websocketConnectionFixture, authenticated bool) {
	t.Helper()
	connection.sendHook = func(req websocketRequest, sendErr error) {
		if sendErr != nil {
			return
		}
		require.NoError(t, ex.websocketHandleDataForConnection(t.Context(), websocketAcknowledgement(t, &req), authenticated), "Handling websocket acknowledgement must not error")
	}
}

func receiveWebsocketData(t *testing.T, ex *Exchange) any {
	t.Helper()
	select {
	case payload := <-ex.Websocket.DataHandler.C:
		return payload.Data
	case <-time.After(time.Second):
		t.Fatal("Timed out waiting for websocket data")
		return nil
	}
}

func TestConnectWebsocket(t *testing.T) {
	ex := new(Exchange)
	require.ErrorIs(t, ex.connectWebsocket(t.Context(), nil), common.ErrNilPointer, "Nil websocket connection must return the expected error")

	failed := &websocketConnectionFixture{dialErr: errWebsocketFixture}
	require.ErrorIs(t, ex.connectWebsocket(t.Context(), failed), errWebsocketFixture, "Websocket dial failure must be returned")
	assert.Zero(t, failed.pingCalls.Load(), "Failed websocket connection should not install a ping handler")

	connection := new(websocketConnectionFixture)
	require.NoError(t, ex.connectWebsocket(t.Context(), connection), "Connecting a websocket fixture must not error")
	assert.Equal(t, int32(1), connection.dialCalls.Load(), "Websocket should dial once")
	assert.Equal(t, int32(1), connection.pingCalls.Load(), "Websocket should install one ping handler")
	assert.JSONEq(t, `{"method":"ping"}`, string(connection.ping.Message), "Ping handler should send the application ping")
	assert.Equal(t, websocketPingInterval, connection.ping.Delay, "Ping handler should use the documented interval")
}

func TestWsConnect(t *testing.T) {
	disabledWebsocket := new(Exchange)
	disabledWebsocket.SetDefaults()
	require.ErrorIs(t, disabledWebsocket.WsConnect(), ws.ErrWebsocketNotEnabled, "Disabled websocket must return the expected error")

	disabledExchange := newWebsocketConnectTestExchange(t)
	disabledExchange.SetEnabled(false)
	require.ErrorIs(t, disabledExchange.WsConnect(), ws.ErrWebsocketNotEnabled, "Disabled exchange must return the expected websocket error")

	publicFailure := newWebsocketConnectTestExchange(t)
	publicFailure.Websocket.Conn = &websocketConnectionFixture{dialErr: errWebsocketFixture}
	require.ErrorIs(t, publicFailure.WsConnect(), errWebsocketFixture, "Public websocket dial failure must be returned")

	publicOnly := newWebsocketConnectTestExchange(t)
	publicOnly.API.AuthenticatedWebsocketSupport = false
	publicConnection := new(websocketConnectionFixture)
	publicOnly.Websocket.Conn = publicConnection
	require.NoError(t, publicOnly.WsConnect(), "Public-only websocket connection must not error")
	waitForHyperliquidWebsocketReaders(t, &publicOnly.Websocket.Wg)
	assert.Equal(t, int32(1), publicConnection.readCalls.Load(), "Public websocket reader should start")

	missingCredentials := newWebsocketConnectTestExchange(t)
	missingCredentials.API.AuthenticatedWebsocketSupport = true
	missingCredentials.Websocket.SetCanUseAuthenticatedEndpoints(true)
	missingPublic := new(websocketConnectionFixture)
	missingAuth := new(websocketConnectionFixture)
	missingCredentials.Websocket.Conn = missingPublic
	missingCredentials.Websocket.AuthConn = missingAuth
	require.NoError(t, missingCredentials.WsConnect(), "Missing address must degrade to public websocket only")
	waitForHyperliquidWebsocketReaders(t, &missingCredentials.Websocket.Wg)
	assert.False(t, missingCredentials.Websocket.CanUseAuthenticatedEndpoints(), "Missing address should disable account-scoped feeds")
	assert.Zero(t, missingAuth.dialCalls.Load(), "Missing address should not dial the account connection")

	authFailure := newWebsocketConnectTestExchange(t)
	authFailure.API.AuthenticatedWebsocketSupport = true
	setTestCredentials(authFailure, &accounts.Credentials{Key: officialSigningAddress})
	authFailure.Websocket.SetCanUseAuthenticatedEndpoints(true)
	authFailure.Websocket.Conn = new(websocketConnectionFixture)
	failedAuthConnection := &websocketConnectionFixture{dialErr: errWebsocketFixture}
	authFailure.Websocket.AuthConn = failedAuthConnection
	require.NoError(t, authFailure.WsConnect(), "Account websocket dial failure must retain the public stream")
	waitForHyperliquidWebsocketReaders(t, &authFailure.Websocket.Wg)
	assert.False(t, authFailure.Websocket.CanUseAuthenticatedEndpoints(), "Account dial failure should disable account-scoped feeds")
	assert.Zero(t, failedAuthConnection.readCalls.Load(), "Failed account connection should not start a reader")

	authenticated := newWebsocketConnectTestExchange(t)
	authenticated.API.AuthenticatedWebsocketSupport = true
	setTestCredentials(authenticated, &accounts.Credentials{Key: officialSigningAddress})
	authenticated.Websocket.Conn = new(websocketConnectionFixture)
	authenticatedConnection := new(websocketConnectionFixture)
	authenticated.Websocket.AuthConn = authenticatedConnection
	require.NoError(t, authenticated.WsConnect(), "Public and account websocket connections must not error")
	waitForHyperliquidWebsocketReaders(t, &authenticated.Websocket.Wg)
	assert.True(t, authenticated.Websocket.CanUseAuthenticatedEndpoints(), "Configured address should enable account-scoped feeds")
	assert.Equal(t, int32(1), authenticatedConnection.readCalls.Load(), "Account websocket reader should start")
}

func TestWebsocketReadLoop(t *testing.T) {
	ex := newWebsocketHandlerTestExchange(t)
	connection := &websocketConnectionFixture{messages: []ws.Response{{Raw: []byte(`invalid`)}, {}}}
	ex.Websocket.Wg.Add(1)
	ex.websocketReadLoop(t.Context(), connection, false)
	waitForHyperliquidWebsocketReaders(t, &ex.Websocket.Wg)
	relayed, ok := receiveWebsocketData(t, ex).(error)
	require.True(t, ok, "Read-loop message must be an error")
	assert.Error(t, relayed, "Read-loop handler error should be relayed")
	assert.Equal(t, int32(2), connection.readCalls.Load(), "Read loop should stop on an empty message")

	fullRelay := newWebsocketHandlerTestExchange(t)
	fullRelay.Websocket.DataHandler = stream.NewRelay(1)
	require.NoError(t, fullRelay.Websocket.DataHandler.Send(t.Context(), "filler"), "Filling the websocket relay must not error")
	fullConnection := &websocketConnectionFixture{messages: []ws.Response{{Raw: []byte(`invalid`)}, {}}}
	fullRelay.Websocket.Wg.Add(1)
	fullRelay.websocketReadLoop(t.Context(), fullConnection, false)
	waitForHyperliquidWebsocketReaders(t, &fullRelay.Websocket.Wg)
	assert.Equal(t, int32(2), fullConnection.readCalls.Load(), "Read loop should continue after a full data relay")
}

func TestWebsocketHandleDataRouting(t *testing.T) {
	ex := newWebsocketHandlerTestExchange(t)
	require.Error(t, ex.websocketHandleData(t.Context(), []byte(`invalid`)), "Invalid websocket envelope must error")
	require.NoError(t, ex.websocketHandleData(t.Context(), []byte(websocketConnectionEstablished)), "Connection acknowledgement must be ignored")
	require.ErrorIs(t, ex.websocketHandleData(t.Context(), []byte(`{"channel":"subscriptionResponse","data":{}}`)), errWebsocketSubscription, "Malformed subscription response must error")
	require.NoError(t, ex.websocketHandleData(t.Context(), []byte(`{"channel":"pong","data":{}}`)), "Pong must be ignored")
	err := ex.websocketHandleData(t.Context(), []byte(`{"channel":"error","data":"bad subscription"}`))
	require.ErrorIs(t, err, errWebsocketServer, "Websocket error channel must return the expected error")
	assert.ErrorContains(t, err, "bad subscription", "Websocket error should retain the server message")
	err = ex.websocketHandleData(t.Context(), []byte(`{"channel":"error","data":{"message":"bad request"}}`))
	require.ErrorIs(t, err, errWebsocketServer, "Structured websocket error must return the expected error")
	assert.ErrorContains(t, err, `"message":"bad request"`, "Structured websocket error should retain the server payload")
	err = ex.websocketHandleData(t.Context(), []byte(`{"channel":"error","data":null}`))
	require.ErrorIs(t, err, errWebsocketServer, "Empty websocket error must return the expected error")
	assert.ErrorContains(t, err, "unspecified error", "Empty websocket error should use a safe fallback")

	for _, channel := range []string{
		wsChannelActiveAssetContext,
		wsChannelActiveSpotAssetContext,
		wsChannelOrderbook,
		wsChannelTrades,
		wsChannelCandle,
		wsChannelOrderUpdates,
		wsChannelUserFills,
	} {
		err := ex.websocketHandleData(t.Context(), []byte(`{"channel":"`+channel+`","data":"invalid"}`))
		require.Error(t, err, "Invalid routed channel data must error")
	}

	require.NoError(t, ex.websocketHandleData(t.Context(), []byte(`{"channel":"futureChannel","data":{}}`)), "Unhandled websocket message must be relayed as a warning")
	_, ok := receiveWebsocketData(t, ex).(ws.UnhandledMessageWarning)
	assert.True(t, ok, "Unhandled channel should relay the expected warning type")
}

func TestWebsocketHandleSubscriptionAcknowledgement(t *testing.T) {
	ex := newWebsocketHandlerTestExchange(t)
	connection := new(websocketConnectionFixture)
	ex.Websocket.Conn = connection

	require.ErrorIs(
		t,
		ex.websocketHandleDataForConnection(t.Context(), []byte(`{"channel":"subscriptionResponse","data":"invalid"}`), false),
		errWebsocketSubscription,
		"Invalid acknowledgement payload must return the expected error")
	require.ErrorIs(
		t,
		ex.websocketHandleDataForConnection(t.Context(), websocketAcknowledgement(t, &websocketRequest{
			Method:       "invalid",
			Subscription: websocketSubscription{Type: wsChannelTrades, Coin: "BTC"},
		}), false),
		errWebsocketSubscription,
		"Invalid acknowledgement method must return the expected error")
	require.ErrorIs(
		t,
		ex.websocketHandleDataForConnection(t.Context(), websocketAcknowledgement(t, &websocketRequest{Method: wsMethodSubscribe}), false),
		errWebsocketSubscription,
		"Acknowledgement without a subscription type must return the expected error")

	unknown := websocketRequest{
		Method:       wsMethodSubscribe,
		Subscription: websocketSubscription{Type: wsChannelTrades, Coin: "UNKNOWN"},
	}
	require.NoError(t, ex.websocketHandleDataForConnection(t.Context(), websocketAcknowledgement(t, &unknown), false), "Unknown or late acknowledgement must be ignored")

	wrongMethodSub := &subscription.Subscription{Channel: subscription.AllTradesChannel}
	require.NoError(t, wrongMethodSub.SetState(subscription.SubscribingState), "Preparing wrong-method subscription state must not error")
	wrongMethodPayload := websocketSubscription{Type: wsChannelTrades, Coin: "BTC"}
	wrongMethodKey := websocketPendingKey{subscription: wrongMethodPayload}
	wrongMethodPending := &websocketPendingOperation{
		method:        wsMethodSubscribe,
		connection:    connection,
		subscription:  wrongMethodSub,
		previousState: subscription.InactiveState,
		done:          make(chan error, 1),
	}
	ex.websocketPending[wrongMethodKey] = wrongMethodPending
	err := ex.websocketHandleDataForConnection(t.Context(), websocketAcknowledgement(t, &websocketRequest{
		Method:       wsMethodUnsubscribe,
		Subscription: wrongMethodPayload,
	}), false)
	require.ErrorIs(t, err, errWebsocketSubscription, "Crossed acknowledgement method must fail closed")
	assert.Same(t, wrongMethodPending, ex.websocketPending[wrongMethodKey], "Crossed acknowledgement should not remove the pending operation")
	delete(ex.websocketPending, wrongMethodKey)

	stateConflictSub := &subscription.Subscription{
		Channel:          subscription.OrderbookChannel,
		Asset:            asset.PerpetualContract,
		Pairs:            currency.Pairs{testPerpetualPair},
		QualifiedChannel: "ack-state-conflict",
	}
	require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(connection, stateConflictSub), "Preparing subscribed acknowledgement fixture must not error")
	stateConflictPayload := websocketSubscription{Type: wsChannelOrderbook, Coin: "BTC"}
	stateConflictPending := &websocketPendingOperation{
		method:        wsMethodSubscribe,
		connection:    connection,
		subscription:  stateConflictSub,
		previousState: subscription.InactiveState,
		done:          make(chan error, 1),
	}
	ex.websocketPending[websocketPendingKey{subscription: stateConflictPayload}] = stateConflictPending
	err = ex.websocketHandleDataForConnection(t.Context(), websocketAcknowledgement(t, &websocketRequest{
		Method:       wsMethodSubscribe,
		Subscription: stateConflictPayload,
	}), false)
	require.ErrorIs(t, err, subscription.ErrInStateAlready, "Conflicting subscribe acknowledgement state must be returned")
	require.ErrorIs(t, <-stateConflictPending.done, subscription.ErrInStateAlready, "Subscribe waiter must receive the state conflict")
	assert.Nil(t, ex.Websocket.GetSubscription(stateConflictSub), "Conflicting subscribe acknowledgement should roll back the stored subscription")

	missingUnsubscribeSub := &subscription.Subscription{Channel: subscription.CandlesChannel}
	require.NoError(t, missingUnsubscribeSub.SetState(subscription.SubscribedState), "Preparing unsubscribe fixture state must not error")
	require.NoError(t, missingUnsubscribeSub.SetState(subscription.UnsubscribingState), "Preparing unsubscribe transition must not error")
	missingUnsubscribePayload := websocketSubscription{Type: wsChannelCandle, Coin: "BTC", Interval: "1m"}
	missingUnsubscribePending := &websocketPendingOperation{
		method:        wsMethodUnsubscribe,
		connection:    connection,
		subscription:  missingUnsubscribeSub,
		previousState: subscription.SubscribedState,
		done:          make(chan error, 1),
	}
	ex.websocketPending[websocketPendingKey{subscription: missingUnsubscribePayload}] = missingUnsubscribePending
	err = ex.websocketHandleDataForConnection(t.Context(), websocketAcknowledgement(t, &websocketRequest{
		Method:       wsMethodUnsubscribe,
		Subscription: missingUnsubscribePayload,
	}), false)
	require.ErrorIs(t, err, subscription.ErrNotFound, "Acknowledged unknown unsubscribe must return the store error")
	require.ErrorIs(t, <-missingUnsubscribePending.done, subscription.ErrNotFound, "Unsubscribe waiter must receive the store error")
	assert.Equal(t, subscription.SubscribedState, missingUnsubscribeSub.State(), "Failed unsubscribe acknowledgement should restore the prior state")

	authSub := &subscription.Subscription{Channel: subscription.MyOrdersChannel, Authenticated: true}
	require.NoError(t, authSub.SetState(subscription.SubscribingState), "Preparing authenticated subscription state must not error")
	authPayload := websocketSubscription{Type: wsChannelOrderUpdates, User: officialSigningAddress}
	authKey := websocketPendingKey{authenticated: true, subscription: authPayload}
	authPending := &websocketPendingOperation{
		method:        wsMethodSubscribe,
		connection:    connection,
		subscription:  authSub,
		previousState: subscription.InactiveState,
		done:          make(chan error, 1),
	}
	ex.websocketPending[authKey] = authPending
	authAck := websocketAcknowledgement(t, &websocketRequest{Method: wsMethodSubscribe, Subscription: authPayload})
	require.NoError(t, ex.websocketHandleDataForConnection(t.Context(), authAck, false), "Acknowledgement on the wrong connection must be ignored")
	assert.Same(t, authPending, ex.websocketPending[authKey], "Wrong-connection acknowledgement should not mutate authenticated state")
	require.NoError(t, ex.websocketHandleDataForConnection(t.Context(), authAck, true), "Authenticated acknowledgement must complete its exact operation")
	require.NoError(t, <-authPending.done, "Authenticated subscription waiter must receive success")
	assert.Equal(t, subscription.SubscribedState, authSub.State(), "Authenticated acknowledgement should mark the subscription active")
	require.NoError(t, ex.websocketHandleDataForConnection(t.Context(), authAck, true), "Duplicate late acknowledgement must be ignored")
}

func TestRollbackWebsocketPending(t *testing.T) {
	var nilExchange *Exchange
	require.ErrorIs(t, nilExchange.rollbackWebsocketPending(nil), common.ErrNilPointer, "Nil rollback inputs must return the expected error")

	ex := newWebsocketHandlerTestExchange(t)
	connection := new(websocketConnectionFixture)
	subscribeSub := &subscription.Subscription{Channel: subscription.AllTradesChannel, QualifiedChannel: "rollback-subscribe"}
	require.NoError(t, ex.Websocket.AddSubscriptions(connection, subscribeSub), "Preparing subscribe rollback must not error")
	require.NoError(t, ex.rollbackWebsocketPending(&websocketPendingOperation{
		method:       wsMethodSubscribe,
		connection:   connection,
		subscription: subscribeSub,
	}), "Rolling back a subscribe must not error")
	assert.Nil(t, ex.Websocket.GetSubscription(subscribeSub), "Subscribe rollback should remove the subscription")

	unsubscribeSub := &subscription.Subscription{Channel: subscription.AllTradesChannel}
	require.NoError(t, unsubscribeSub.SetState(subscription.SubscribedState), "Preparing unsubscribe rollback state must not error")
	require.NoError(t, ex.rollbackWebsocketPending(&websocketPendingOperation{
		method:        wsMethodUnsubscribe,
		connection:    connection,
		subscription:  unsubscribeSub,
		previousState: subscription.SubscribedState,
	}), "Rollback already in the previous unsubscribe state must be a no-op")
	require.NoError(t, unsubscribeSub.SetState(subscription.UnsubscribingState), "Preparing active unsubscribe rollback must not error")
	require.NoError(t, ex.rollbackWebsocketPending(&websocketPendingOperation{
		method:        wsMethodUnsubscribe,
		connection:    connection,
		subscription:  unsubscribeSub,
		previousState: subscription.SubscribedState,
	}), "Rolling back an unsubscribe must restore its previous state")
	assert.Equal(t, subscription.SubscribedState, unsubscribeSub.State(), "Unsubscribe rollback should restore the previous state")

	require.ErrorIs(t, ex.rollbackWebsocketPending(&websocketPendingOperation{
		method:       "invalid",
		connection:   connection,
		subscription: unsubscribeSub,
	}), common.ErrNotYetImplemented, "Unknown rollback method must return the expected error")
}

func TestFailWebsocketPending(t *testing.T) {
	ex := newWebsocketHandlerTestExchange(t)
	require.ErrorIs(t, ex.failWebsocketPending(false, nil), common.ErrNilPointer, "Nil pending failure cause must return the expected error")

	connection := new(websocketConnectionFixture)
	publicSub := &subscription.Subscription{Channel: subscription.AllTradesChannel, QualifiedChannel: "fail-public"}
	authSub := &subscription.Subscription{Channel: subscription.MyOrdersChannel, Authenticated: true, QualifiedChannel: "fail-auth"}
	require.NoError(t, ex.Websocket.AddSubscriptions(connection, publicSub, authSub), "Preparing pending failure subscriptions must not error")
	publicPending := &websocketPendingOperation{
		method:       wsMethodSubscribe,
		connection:   connection,
		subscription: publicSub,
		done:         make(chan error, 1),
	}
	authPending := &websocketPendingOperation{
		method:       wsMethodSubscribe,
		connection:   connection,
		subscription: authSub,
		done:         make(chan error, 1),
	}
	invalidPending := &websocketPendingOperation{
		method:       "invalid",
		connection:   connection,
		subscription: &subscription.Subscription{},
		done:         make(chan error, 1),
	}
	publicKey := websocketPendingKey{subscription: websocketSubscription{Type: wsChannelTrades, Coin: "BTC"}}
	authKey := websocketPendingKey{authenticated: true, subscription: websocketSubscription{Type: wsChannelOrderUpdates, User: officialSigningAddress}}
	invalidKey := websocketPendingKey{subscription: websocketSubscription{Type: wsChannelCandle, Coin: "BTC", Interval: "1m"}}
	ex.websocketPending[publicKey] = publicPending
	ex.websocketPending[authKey] = authPending
	ex.websocketPending[invalidKey] = invalidPending

	err := ex.failWebsocketPending(false, errWebsocketFixture)
	require.ErrorIs(t, err, errWebsocketFixture, "Pending failure must retain the connection cause")
	require.ErrorIs(t, err, common.ErrNotYetImplemented, "Pending failure must retain rollback errors")
	require.ErrorIs(t, <-publicPending.done, errWebsocketFixture, "Public pending waiter must receive the connection cause")
	require.ErrorIs(t, <-invalidPending.done, common.ErrNotYetImplemented, "Invalid pending waiter must receive its rollback error")
	assert.Nil(t, ex.Websocket.GetSubscription(publicSub), "Failed public subscribe should be rolled back")
	assert.Same(t, authPending, ex.websocketPending[authKey], "Public failure should not consume authenticated pending operations")

	err = ex.failWebsocketPending(true, errWebsocketServer)
	require.ErrorIs(t, err, errWebsocketServer, "Authenticated pending failure must retain its cause")
	require.ErrorIs(t, <-authPending.done, errWebsocketServer, "Authenticated pending waiter must receive the connection cause")
	assert.Empty(t, ex.websocketPending, "All matching pending operations should be removed")
	assert.Nil(t, ex.Websocket.GetSubscription(authSub), "Failed authenticated subscribe should be rolled back")
}

func TestAbortWebsocketPending(t *testing.T) {
	var nilExchange *Exchange
	require.ErrorIs(t, nilExchange.abortWebsocketPending(nil, nil, nil), common.ErrNilPointer, "Nil abort inputs must return the expected error")

	ex := newWebsocketHandlerTestExchange(t)
	connection := new(websocketConnectionFixture)
	sub := &subscription.Subscription{Channel: subscription.AllTradesChannel, QualifiedChannel: "abort-owned"}
	require.NoError(t, ex.Websocket.AddSubscriptions(connection, sub), "Preparing owned pending abort must not error")
	key := websocketPendingKey{subscription: websocketSubscription{Type: wsChannelTrades, Coin: "BTC"}}
	pending := &websocketPendingOperation{
		method:       wsMethodSubscribe,
		connection:   connection,
		subscription: sub,
		done:         make(chan error, 1),
	}
	ex.websocketPending[key] = pending
	err := ex.abortWebsocketPending(&key, pending, errWebsocketFixture)
	require.ErrorIs(t, err, errWebsocketFixture, "Owned pending abort must retain its cause")
	assert.NotContains(t, ex.websocketPending, key, "Owned pending abort should remove the operation")
	assert.Nil(t, ex.Websocket.GetSubscription(sub), "Owned pending abort should roll back the subscription")

	completed := &websocketPendingOperation{done: make(chan error, 1)}
	completed.done <- errWebsocketServer
	err = ex.abortWebsocketPending(&key, completed, errWebsocketFixture)
	require.ErrorIs(t, err, errWebsocketServer, "Completed pending abort must return its authoritative concurrent result")
	assert.NotErrorIs(t, err, errWebsocketFixture, "Completed pending abort should discard a losing local cause")
}

func TestWebsocketHandleTicker(t *testing.T) {
	ex := newWebsocketHandlerTestExchange(t)
	require.Error(t, ex.websocketHandleTicker(t.Context(), []byte(`invalid`), asset.Spot), "Invalid ticker update must error")

	perpetual := `{"coin":"BTC","ctx":{"openInterest":"10","prevDayPx":"90","dayNtlVlm":"1000","oraclePx":"99","markPx":"101","midPx":"100","dayBaseVlm":"5"}}`
	require.NoError(t, ex.websocketHandleTicker(t.Context(), []byte(perpetual), asset.PerpetualContract), "Valid perpetual ticker must not error")
	perpetualTicker, ok := receiveWebsocketData(t, ex).(*ticker.Price)
	require.True(t, ok, "Perpetual ticker must relay the expected type")
	assert.Equal(t, 100.0, perpetualTicker.Last, "Perpetual midpoint should be used as last")
	assert.Equal(t, 10.0, perpetualTicker.OpenInterest, "Perpetual open interest should be decoded")
	assert.Equal(t, 99.0, perpetualTicker.IndexPrice, "Perpetual oracle price should be used as index")

	spot := `{"coin":"@107","ctx":{"prevDayPx":"9","dayNtlVlm":"100","markPx":"10","midPx":"0","dayBaseVlm":"5"}}`
	require.NoError(t, ex.websocketHandleTicker(t.Context(), []byte(spot), asset.Spot), "Valid spot ticker must not error")
	spotTicker, ok := receiveWebsocketData(t, ex).(*ticker.Price)
	require.True(t, ok, "Spot ticker must relay the expected type")
	assert.Equal(t, 10.0, spotTicker.Last, "Spot mark price should be the zero-midpoint fallback")
	assert.Equal(t, 5.0, spotTicker.Volume, "Spot base volume should be decoded")

	require.ErrorIs(t, ex.websocketHandleTicker(t.Context(), []byte(perpetual), asset.Spot), errWebsocketAssetMismatch, "Ticker channel asset mismatch must return the expected error")
	require.Error(t, ex.websocketHandleTicker(t.Context(), []byte(`{"coin":"BTC","ctx":"bad"}`), asset.PerpetualContract), "Invalid perpetual ticker context must error")
	require.Error(t, ex.websocketHandleTicker(t.Context(), []byte(`{"coin":"@107","ctx":"bad"}`), asset.Spot), "Invalid spot ticker context must error")
	require.ErrorIs(t, ex.websocketHandleTicker(t.Context(), []byte(`{"coin":"MISSING","ctx":{}}`), asset.Spot), errPairMappingNotFound, "Unknown ticker market must return the expected error")

	perpetualFallback := `{"coin":"BTC","ctx":{"prevDayPx":"99","dayNtlVlm":"1000","oraclePx":"100","markPx":"101","midPx":"0","dayBaseVlm":"10"}}`
	require.NoError(t, ex.websocketHandleTicker(t.Context(), []byte(perpetualFallback), asset.PerpetualContract), "Perpetual ticker without a midpoint must not error")
	fallbackTicker, ok := receiveWebsocketData(t, ex).(*ticker.Price)
	require.True(t, ok, "Fallback ticker must relay the expected type")
	assert.Equal(t, 101.0, fallbackTicker.Last, "Perpetual ticker should fall back to the mark price")
}

func TestWebsocketHandleOrderbook(t *testing.T) {
	ex := newWebsocketHandlerTestExchange(t)
	require.Error(t, ex.websocketHandleOrderbook(t.Context(), []byte(`invalid`)), "Invalid orderbook update must error")
	require.ErrorIs(t, ex.websocketHandleOrderbook(t.Context(), []byte(`{"coin":"BTC","levels":[],"time":1700000000000}`)), errInvalidBookLevelCount, "Invalid orderbook side count must return the expected error")
	require.ErrorIs(t, ex.websocketHandleOrderbook(t.Context(), []byte(`{"coin":"MISSING","levels":[[],[]],"time":1700000000000}`)), errPairMappingNotFound, "Unknown orderbook market must return the expected error")

	raw := `{"coin":"BTC","levels":[[{"px":"100","sz":"2","n":1}],[{"px":"101","sz":"3","n":1}]],"time":1700000000000}`
	require.NoError(t, ex.websocketHandleOrderbook(t.Context(), []byte(raw)), "Valid orderbook snapshot must not error")
	_, ok := receiveWebsocketData(t, ex).(*orderbook.Depth)
	require.True(t, ok, "Orderbook snapshot must relay a depth")
	book, err := ex.Websocket.Orderbook.GetOrderbook(testPerpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "Getting stored websocket orderbook must not error")
	require.Len(t, book.Bids, 1, "Stored orderbook must contain one bid")
	require.Len(t, book.Asks, 1, "Stored orderbook must contain one ask")
	assert.Equal(t, 100.0, book.Bids[0].Price, "Stored bid price should match")

	require.Error(t, ex.websocketHandleOrderbook(t.Context(), []byte(`{"coin":"@107","levels":[[{"px":"0","sz":"1"}],[{"px":"1","sz":"1"}]],"time":1700000000000}`)), "Invalid orderbook snapshot must return validation error")
}

func TestWebsocketHandleTrades(t *testing.T) {
	ex := newWebsocketHandlerTestExchange(t)
	require.Error(t, ex.websocketHandleTrades(t.Context(), []byte(`invalid`)), "Invalid trade update must error")
	require.ErrorIs(t, ex.websocketHandleTrades(t.Context(), []byte(`[{"coin":"MISSING","side":"B"}]`)), errPairMappingNotFound, "Unknown trade market must return the expected error")
	require.ErrorIs(t, ex.websocketHandleTrades(t.Context(), []byte(`[{"coin":"BTC","side":"X"}]`)), order.ErrSideIsInvalid, "Invalid trade side must return the expected error")
	require.NoError(t, ex.websocketHandleTrades(t.Context(), []byte(`[]`)), "Empty trade update must not error")

	raw := `[{"coin":"BTC","side":"A","px":"100","sz":"2","time":1700000000000,"tid":7},{"coin":"@107","side":"B","px":"10","sz":"3","time":1700000001000,"tid":8}]`
	require.NoError(t, ex.websocketHandleTrades(t.Context(), []byte(raw)), "Valid trade update must not error")
	trades, ok := receiveWebsocketData(t, ex).([]trade.Data)
	require.True(t, ok, "Trade update must relay the expected type")
	require.Len(t, trades, 2, "Both trades must be relayed")
	assert.Equal(t, order.Sell, trades[0].Side, "Ask trade should be converted to sell")
	assert.Equal(t, order.Buy, trades[1].Side, "Bid trade should be converted to buy")
	assert.Equal(t, "8", trades[1].TID, "Trade ID should be converted")
}

func TestWebsocketHandleCandle(t *testing.T) {
	ex := newWebsocketHandlerTestExchange(t)
	require.Error(t, ex.websocketHandleCandle(t.Context(), []byte(`invalid`)), "Invalid candle update must error")
	require.ErrorIs(t, ex.websocketHandleCandle(t.Context(), []byte(`{"s":"MISSING","i":"1m"}`)), errPairMappingNotFound, "Unknown candle market must return the expected error")
	require.ErrorIs(t, ex.websocketHandleCandle(t.Context(), []byte(`{"s":"BTC","i":"bad"}`)), kline.ErrUnsupportedInterval, "Unsupported candle interval must return the expected error")

	raw := `{"t":1700000000000,"T":1700000059999,"s":"BTC","i":"1m","o":"100","c":"101","h":"102","l":"99","v":"5","n":3}`
	require.NoError(t, ex.websocketHandleCandle(t.Context(), []byte(raw)), "Valid candle update must not error")
	item, ok := receiveWebsocketData(t, ex).(kline.Item)
	require.True(t, ok, "Candle update must relay the expected type")
	assert.Equal(t, kline.OneMin, item.Interval, "Candle interval should be converted")
	require.Len(t, item.Candles, 1, "Candle update must contain one candle")
	assert.Equal(t, 101.0, item.Candles[0].Close, "Candle close should be decoded")
}

func TestWebsocketHandleOrderUpdates(t *testing.T) {
	ex := newWebsocketHandlerTestExchange(t)
	require.Error(t, ex.websocketHandleOrderUpdates(t.Context(), []byte(`invalid`)), "Invalid order update must error")
	require.ErrorIs(t, ex.websocketHandleOrderUpdates(t.Context(), []byte(`[{"order":{"coin":"MISSING","side":"B","timestamp":1700000000000,"orderType":"Limit","tif":"Gtc"},"status":"open","statusTimestamp":1700000001000}]`)), errPairMappingNotFound, "Invalid order update conversion must return the expected error")
	require.ErrorIs(t, ex.websocketHandleOrderUpdates(t.Context(), []byte(`[{"order":{"coin":"BTC","side":"X","timestamp":1700000000000,"orderType":"Limit","tif":"Gtc"},"status":"open","statusTimestamp":1700000001000}]`)), order.ErrSideIsInvalid, "Invalid mapped order update must return its conversion error")

	raw := `[{"order":{"coin":"BTC","side":"B","limitPx":"100","sz":"1","origSz":"2","oid":7,"timestamp":1700000000000,"isTrigger":false,"reduceOnly":false,"orderType":"Limit","tif":"Gtc"},"status":"open","statusTimestamp":1700000001000}]`
	require.NoError(t, ex.websocketHandleOrderUpdates(t.Context(), []byte(raw)), "Valid order update must not error")
	orders, ok := receiveWebsocketData(t, ex).([]order.Detail)
	require.True(t, ok, "Order update must relay the expected type")
	require.Len(t, orders, 1, "One order update must be relayed")
	assert.Equal(t, "7", orders[0].OrderID, "Order update ID should be converted")

	mixed := `[{"order":{"coin":"MISSING","side":"B","timestamp":1700000000000,"orderType":"Limit","tif":"Gtc"},"status":"open","statusTimestamp":1700000001000},` + raw[1:]
	err := ex.websocketHandleOrderUpdates(t.Context(), []byte(mixed))
	require.ErrorIs(t, err, errPairMappingNotFound, "Mixed order-update batch must return the invalid entry error")
	orders, ok = receiveWebsocketData(t, ex).([]order.Detail)
	require.True(t, ok, "Mixed order-update batch must relay valid entries")
	require.Len(t, orders, 1, "Mixed order-update batch must retain its valid entry")
	assert.Equal(t, "7", orders[0].OrderID, "Mixed order-update batch should retain the valid order")
}

func TestWebsocketHandleUserFills(t *testing.T) {
	ex := newWebsocketHandlerTestExchange(t)
	ex.SetFillsFeedStatus(false)
	require.NoError(t, ex.websocketHandleUserFills(t.Context(), []byte(`invalid`)), "Disabled fills feed must ignore updates")

	ex.SetFillsFeedStatus(true)
	ex.Websocket.Fills.Setup(true, ex.Websocket.DataHandler)
	require.Error(t, ex.websocketHandleUserFills(t.Context(), []byte(`invalid`)), "Invalid fill update must error")
	require.NoError(t, ex.websocketHandleUserFills(t.Context(), []byte(`{"isSnapshot":true,"fills":[]}`)), "Fill snapshot must be ignored")
	require.ErrorIs(t, ex.websocketHandleUserFills(t.Context(), []byte(`{"fills":[{"coin":"MISSING","side":"B"}]}`)), errPairMappingNotFound, "Unknown fill market must return the expected error")
	require.ErrorIs(t, ex.websocketHandleUserFills(t.Context(), []byte(`{"fills":[{"coin":"BTC","side":"X"}]}`)), order.ErrSideIsInvalid, "Invalid fill side must return the expected error")

	raw := `{"isSnapshot":false,"user":"` + officialSigningAddress + `","fills":[{"coin":"BTC","px":"100","sz":"2","side":"A","time":1700000000000,"hash":"0x1","oid":7,"tid":8},{"coin":"@107","px":"10","sz":"3","side":"B","time":1700000001000,"hash":"0x1","oid":9,"tid":10,"cloid":"` + validClientOrderID + `"}]}`
	require.NoError(t, ex.websocketHandleUserFills(t.Context(), []byte(raw)), "Valid fill update must not error")
	fills, ok := receiveWebsocketData(t, ex).([]fill.Data)
	require.True(t, ok, "Fill update must relay the expected type")
	require.Len(t, fills, 2, "Both fills must be relayed")
	assert.Equal(t, order.Sell, fills[0].Side, "Ask fill should be converted to sell")
	assert.Equal(t, order.Buy, fills[1].Side, "Bid fill should be converted to buy")
	assert.Equal(t, validClientOrderID, fills[1].ClientOrderID, "Fill client order ID should be retained")
	assert.Equal(t, "10", fills[1].TradeID, "Fill trade ID should be converted")
	assert.Equal(t, "8", fills[0].ID, "Fill ID should use Hyperliquid's unique trade ID")
	assert.Equal(t, "10", fills[1].ID, "Fills sharing a transaction hash should retain distinct IDs")

	mixed := `{"fills":[{"coin":"BTC","side":"X"},{"coin":"@107","px":"10","sz":"3","side":"B","time":1700000001000,"hash":"0x1","oid":9,"tid":10}]}`
	err := ex.websocketHandleUserFills(t.Context(), []byte(mixed))
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "Mixed fill batch must return the invalid entry error")
	fills, ok = receiveWebsocketData(t, ex).([]fill.Data)
	require.True(t, ok, "Mixed fill batch must relay valid entries")
	require.Len(t, fills, 1, "Mixed fill batch must retain its valid entry")
	assert.Equal(t, "10", fills[0].TradeID, "Mixed fill batch should retain the valid fill")

	require.NoError(t, ex.websocketHandleUserFills(t.Context(), []byte(`{"fills":[]}`)), "Empty fill update must not error")
}

func TestParseWebsocketInterval(t *testing.T) {
	for _, interval := range []kline.Interval{
		kline.OneMin,
		kline.ThreeMin,
		kline.FiveMin,
		kline.FifteenMin,
		kline.ThirtyMin,
		kline.OneHour,
		kline.TwoHour,
		kline.FourHour,
		kline.EightHour,
		kline.TwelveHour,
		kline.OneDay,
		kline.ThreeDay,
		kline.OneWeek,
		kline.OneMonth,
	} {
		formatted, err := formatInterval(interval)
		require.NoError(t, err, "Formatting websocket interval fixture must not error")
		result, err := parseWebsocketInterval(formatted)
		require.NoError(t, err, "Parsing supported websocket interval must not error")
		assert.Equal(t, interval, result, "Parsed websocket interval should round trip")
	}
	_, err := parseWebsocketInterval("bad")
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "Unsupported websocket interval must return the expected error")
}

func TestWebsocketSubscriptionTemplates(t *testing.T) {
	ex := newWebsocketHandlerTestExchange(t)
	name, err := websocketChannelName(&subscription.Subscription{Channel: subscription.OrderbookChannel})
	require.NoError(t, err, "Getting a supported channel name must not error")
	assert.Equal(t, wsChannelOrderbook, name, "Channel name should map to Hyperliquid")
	_, err = websocketChannelName(nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "A nil channel must return the expected error")
	_, err = websocketChannelName(&subscription.Subscription{Channel: "unsupported"})
	require.ErrorIs(t, err, subscription.ErrNotSupported, "An unsupported channel must return the expected error")

	template, err := ex.GetSubscriptionTemplate(nil)
	require.NoError(t, err, "Getting subscription template must not error")
	assert.NotNil(t, template, "Subscription template should be returned")

	subscriptions, err := ex.generateSubscriptions()
	require.NoError(t, err, "Generating default subscriptions must not error")
	assert.NotEmpty(t, subscriptions, "Default subscriptions should expand")
}

func TestWebsocketSubscriptionPayload(t *testing.T) {
	ex := newWebsocketHandlerTestExchange(t)
	_, err := ex.websocketSubscriptionPayload(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "Nil subscription must return the expected error")
	_, err = ex.websocketSubscriptionPayload(t.Context(), &subscription.Subscription{Channel: "unsupported"})
	require.ErrorIs(t, err, subscription.ErrNotSupported, "Unsupported subscription channel must return the expected error")

	_, err = ex.websocketSubscriptionPayload(t.Context(), &subscription.Subscription{Channel: subscription.TickerChannel, Authenticated: true})
	require.ErrorIs(t, err, subscription.ErrNotSupported, "Authenticated public channel must return the expected error")
	_, err = ex.websocketSubscriptionPayload(t.Context(), &subscription.Subscription{Channel: subscription.MyOrdersChannel})
	require.ErrorIs(t, err, subscription.ErrNotSupported, "Unauthenticated account channel must return the expected error")

	payload, err := ex.websocketSubscriptionPayload(t.Context(), &subscription.Subscription{Channel: subscription.MyOrdersChannel, Authenticated: true})
	require.NoError(t, err, "Address-scoped order subscription must not error")
	assert.Equal(t, wsChannelOrderUpdates, payload.Type, "Order subscription type should match")
	assert.Equal(t, officialSigningAddress, payload.User, "Order subscription should include configured address")

	payload, err = ex.websocketSubscriptionPayload(t.Context(), &subscription.Subscription{Channel: subscription.MyTradesChannel, Authenticated: true})
	require.NoError(t, err, "Address-scoped fill subscription must not error")
	assert.Equal(t, wsChannelUserFills, payload.Type, "Fill subscription type should match")

	missingCredentials := newWebsocketHandlerTestExchange(t)
	missingCredentials.SetCredentials(nil)
	_, err = missingCredentials.websocketSubscriptionPayload(t.Context(), &subscription.Subscription{Channel: subscription.MyOrdersChannel, Authenticated: true})
	require.Error(t, err, "Address-scoped subscription without credentials must error")

	_, err = ex.websocketSubscriptionPayload(t.Context(), &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.PerpetualContract})
	require.ErrorIs(t, err, subscription.ErrNotSinglePair, "Public subscription without one pair must return the expected error")
	_, err = ex.websocketSubscriptionPayload(t.Context(), &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{testPerpetualPair, testPerpetualPair}})
	require.ErrorIs(t, err, subscription.ErrNotSinglePair, "Public subscription with multiple pairs must return the expected error")
	_, err = ex.websocketSubscriptionPayload(t.Context(), &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{currency.NewPair(currency.ETH, currency.USDC)}})
	require.ErrorIs(t, err, errPairMappingNotFound, "Unknown subscription pair must return the expected error")

	payload, err = ex.websocketSubscriptionPayload(t.Context(), &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{testPerpetualPair}})
	require.NoError(t, err, "Perpetual ticker subscription must not error")
	assert.Equal(t, wsChannelActiveAssetContext, payload.Type, "Perpetual ticker should use activeAssetCtx")
	assert.Equal(t, "BTC", payload.Coin, "Perpetual ticker should use API coin")

	payload, err = ex.websocketSubscriptionPayload(t.Context(), &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{testSpotPair}})
	require.NoError(t, err, "Spot ticker subscription must not error")
	assert.Equal(t, wsChannelActiveAssetContext, payload.Type, "Spot ticker should subscribe through activeAssetCtx")

	payload, err = ex.websocketSubscriptionPayload(t.Context(), &subscription.Subscription{Channel: subscription.CandlesChannel, Asset: asset.Spot, Pairs: currency.Pairs{testSpotPair}, Interval: kline.OneMin})
	require.NoError(t, err, "Candle subscription must not error")
	assert.Equal(t, "1m", payload.Interval, "Candle interval should be formatted")
	_, err = ex.websocketSubscriptionPayload(t.Context(), &subscription.Subscription{Channel: subscription.CandlesChannel, Asset: asset.Spot, Pairs: currency.Pairs{testSpotPair}, Interval: kline.Interval(42)})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "Unsupported candle subscription interval must return the expected error")
}

func TestManageWebsocketSubscription(t *testing.T) {
	ex := newWebsocketHandlerTestExchange(t)
	connection := new(websocketConnectionFixture)
	authConnection := new(websocketConnectionFixture)
	ex.Websocket.Conn = connection
	ex.Websocket.AuthConn = authConnection
	installWebsocketAcknowledgements(t, ex, connection, false)
	installWebsocketAcknowledgements(t, ex, authConnection, true)

	require.ErrorIs(t, ex.manageWebsocketSubscription(t.Context(), wsMethodSubscribe, nil), common.ErrNilPointer, "Nil managed subscription must return the expected error")
	sub := &subscription.Subscription{
		Channel:          subscription.OrderbookChannel,
		Asset:            asset.PerpetualContract,
		Pairs:            currency.Pairs{testPerpetualPair},
		QualifiedChannel: "l2Book:perpetualcontract:BTC-USDC",
	}
	require.ErrorIs(t, ex.manageWebsocketSubscription(t.Context(), "invalid", sub), common.ErrNotYetImplemented, "Unsupported subscription method must return the expected error")
	require.ErrorIs(t, ex.manageWebsocketSubscription(t.Context(), wsMethodSubscribe, &subscription.Subscription{Channel: "unsupported"}), subscription.ErrNotSupported, "Unsupported managed subscription must return the expected error")

	ex.Websocket.Conn = nil
	require.ErrorIs(t, ex.manageWebsocketSubscription(t.Context(), wsMethodSubscribe, sub), common.ErrNilPointer, "Nil public connection must return the expected error")
	ex.Websocket.Conn = connection

	require.NoError(t, ex.manageWebsocketSubscription(t.Context(), wsMethodSubscribe, sub), "Subscribing one public channel must not error")
	assert.Equal(t, subscription.SubscribedState, sub.State(), "Successful subscription should be marked subscribed")
	assert.NotNil(t, ex.Websocket.GetSubscription(sub.Clone()), "Equivalent subscription should match the stored exact key")
	require.Len(t, connection.sentRequests(), 1, "One subscribe request must be sent")
	assert.Equal(t, wsMethodSubscribe, connection.sentRequests()[0].Method, "Subscribe request should use the subscribe method")
	require.ErrorIs(t, ex.manageWebsocketSubscription(t.Context(), wsMethodSubscribe, sub), subscription.ErrDuplicate, "Duplicate subscription must return the expected error")

	require.NoError(t, ex.manageWebsocketSubscription(t.Context(), wsMethodUnsubscribe, sub), "Unsubscribing one public channel must not error")
	assert.Nil(t, ex.Websocket.GetSubscription(sub), "Unsubscribed channel should be removed")
	assert.Equal(t, wsMethodUnsubscribe, connection.sentRequests()[1].Method, "Unsubscribe request should use the unsubscribe method")

	acknowledgedThenSendFailed := &subscription.Subscription{
		Channel:          subscription.AllTradesChannel,
		Asset:            asset.PerpetualContract,
		Pairs:            currency.Pairs{testPerpetualPair},
		QualifiedChannel: "trades:acknowledged-before-send-error",
	}
	connection.sendErr = errWebsocketFixture
	connection.sendHook = func(req websocketRequest, _ error) {
		require.NoError(t, ex.websocketHandleDataForConnection(t.Context(), websocketAcknowledgement(t, &req), false), "Handling a concurrent acknowledgement must not error")
	}
	require.NoError(t, ex.manageWebsocketSubscription(t.Context(), wsMethodSubscribe, acknowledgedThenSendFailed), "An acknowledgement committed before a send error must remain authoritative")
	assert.Equal(t, subscription.SubscribedState, acknowledgedThenSendFailed.State(), "Acknowledged subscription should remain committed")
	assert.NotNil(t, ex.Websocket.GetSubscription(acknowledgedThenSendFailed), "Acknowledged subscription should remain tracked")
	connection.sendErr = nil
	installWebsocketAcknowledgements(t, ex, connection, false)
	require.NoError(t, ex.manageWebsocketSubscription(t.Context(), wsMethodUnsubscribe, acknowledgedThenSendFailed), "Cleaning up the acknowledged subscription must not error")

	failing := &subscription.Subscription{
		Channel:          subscription.AllTradesChannel,
		Asset:            asset.PerpetualContract,
		Pairs:            currency.Pairs{testPerpetualPair},
		QualifiedChannel: "trades:perpetualcontract:BTC-USDC",
	}
	connection.sendErr = errWebsocketFixture
	require.ErrorIs(t, ex.manageWebsocketSubscription(t.Context(), wsMethodSubscribe, failing), errWebsocketFixture, "Subscription send failure must be returned")
	assert.Nil(t, ex.Websocket.GetSubscription(failing), "Failed subscription should be removed from the store")
	connection.sendErr = nil

	rollbackFailure := &subscription.Subscription{
		Channel:          subscription.AllTradesChannel,
		Asset:            asset.PerpetualContract,
		Pairs:            currency.Pairs{testPerpetualPair},
		QualifiedChannel: "trades:perpetualcontract:BTC-USDC:rollback-failure",
	}
	var preRollbackErr error
	connection.sendErr = errWebsocketFixture
	connection.sendHook = func(websocketRequest, error) {
		preRollbackErr = ex.Websocket.RemoveSubscriptions(connection, rollbackFailure)
	}
	err := ex.manageWebsocketSubscription(t.Context(), wsMethodSubscribe, rollbackFailure)
	require.NoError(t, preRollbackErr, "Concurrent subscription removal must not error")
	require.ErrorIs(t, err, errWebsocketFixture, "Subscription send failure must remain visible when rollback also fails")
	require.ErrorIs(t, err, subscription.ErrNotFound, "Subscription rollback failure must also be returned")
	connection.sendHook = nil
	connection.sendErr = nil
	installWebsocketAcknowledgements(t, ex, connection, false)

	authSub := &subscription.Subscription{Channel: subscription.MyOrdersChannel, Authenticated: true, QualifiedChannel: wsChannelOrderUpdates}
	require.NoError(t, ex.manageWebsocketSubscription(t.Context(), wsMethodSubscribe, authSub), "Subscribing one address-scoped channel must not error")
	require.Len(t, authConnection.sentRequests(), 1, "Address-scoped subscription must use the account connection")

	authConnection.sendErr = errWebsocketFixture
	require.ErrorIs(t, ex.manageWebsocketSubscription(t.Context(), wsMethodUnsubscribe, authSub), errWebsocketFixture, "Unsubscription send failure must be returned")
	assert.NotNil(t, ex.Websocket.GetSubscription(authSub), "Failed unsubscription should remain tracked")
	authConnection.sendErr = nil
	require.NoError(t, ex.manageWebsocketSubscription(t.Context(), wsMethodUnsubscribe, authSub), "Retrying address-scoped unsubscription must not error")

	stateConflict := &subscription.Subscription{
		Channel:          subscription.AllTradesChannel,
		Asset:            asset.PerpetualContract,
		Pairs:            currency.Pairs{testPerpetualPair},
		QualifiedChannel: "trades:state-conflict",
	}
	require.NoError(t, stateConflict.SetState(subscription.UnsubscribingState), "Preparing unsubscribe state conflict must not error")
	require.ErrorIs(t, ex.manageWebsocketSubscription(t.Context(), wsMethodUnsubscribe, stateConflict), subscription.ErrInStateAlready, "Duplicate unsubscribe state transition must fail closed")

	collision := &subscription.Subscription{
		Channel:          subscription.CandlesChannel,
		Asset:            asset.PerpetualContract,
		Pairs:            currency.Pairs{testPerpetualPair},
		Interval:         kline.OneMin,
		QualifiedChannel: "candle:pending-collision",
	}
	collisionPayload, err := ex.websocketSubscriptionPayload(t.Context(), collision)
	require.NoError(t, err, "Building collision payload must not error")
	collisionKey := websocketPendingKey{subscription: collisionPayload}
	ex.websocketPending[collisionKey] = &websocketPendingOperation{done: make(chan error, 1)}
	err = ex.manageWebsocketSubscription(t.Context(), wsMethodSubscribe, collision)
	require.ErrorIs(t, err, errWebsocketSubscription, "Crossed operation for the same exact subscription must fail closed")
	assert.Nil(t, ex.Websocket.GetSubscription(collision), "Rejected crossed operation should roll back its local subscription state")
	delete(ex.websocketPending, collisionKey)

	ex.websocketPending = nil
	nilMapSub := &subscription.Subscription{
		Channel:          subscription.TickerChannel,
		Asset:            asset.PerpetualContract,
		Pairs:            currency.Pairs{testPerpetualPair},
		QualifiedChannel: "ticker:nil-pending-map",
	}
	require.NoError(t, ex.manageWebsocketSubscription(t.Context(), wsMethodSubscribe, nilMapSub), "Subscription must initialise an absent pending-operation map")
	require.NoError(t, ex.manageWebsocketSubscription(t.Context(), wsMethodUnsubscribe, nilMapSub), "Initialised pending-operation map must support unsubscribe")

	quietConnection := new(websocketConnectionFixture)
	ex.Websocket.Conn = quietConnection
	ex.WebsocketResponseMaxLimit = time.Millisecond
	timeoutSub := &subscription.Subscription{
		Channel:          subscription.AllTradesChannel,
		Asset:            asset.PerpetualContract,
		Pairs:            currency.Pairs{testPerpetualPair},
		QualifiedChannel: "trades:timeout",
	}
	err = ex.manageWebsocketSubscription(t.Context(), wsMethodSubscribe, timeoutSub)
	require.ErrorIs(t, err, ws.ErrSignatureTimeout, "Missing subscription acknowledgement must time out")
	assert.Nil(t, ex.Websocket.GetSubscription(timeoutSub), "Timed-out subscription should be rolled back")

	ex.WebsocketResponseMaxLimit = 0
	cancelledContext, cancel := context.WithCancel(t.Context())
	cancel()
	cancelledSub := &subscription.Subscription{
		Channel:          subscription.OrderbookChannel,
		Asset:            asset.PerpetualContract,
		Pairs:            currency.Pairs{testPerpetualPair},
		QualifiedChannel: "l2Book:cancelled",
	}
	err = ex.manageWebsocketSubscription(cancelledContext, wsMethodSubscribe, cancelledSub)
	require.ErrorIs(t, err, context.Canceled, "Cancelled subscription context must be returned")
	assert.Nil(t, ex.Websocket.GetSubscription(cancelledSub), "Cancelled subscription should be rolled back")
}

func TestSubscribeAndUnsubscribe(t *testing.T) {
	ex := newWebsocketHandlerTestExchange(t)
	connection := new(websocketConnectionFixture)
	ex.Websocket.Conn = connection
	installWebsocketAcknowledgements(t, ex, connection, false)
	sub := &subscription.Subscription{
		Channel:          subscription.OrderbookChannel,
		Asset:            asset.PerpetualContract,
		Pairs:            currency.Pairs{testPerpetualPair},
		QualifiedChannel: "l2Book:perpetualcontract:BTC-USDC",
	}
	require.NoError(t, ex.Subscribe(subscription.List{sub}), "Subscribing through wrapper must not error")
	require.NotEmpty(t, connection.sentRequests(), "Wrapper subscribe must send a request")
	require.NoError(t, ex.Unsubscribe(subscription.List{sub}), "Unsubscribing through wrapper must not error")

	missing := &subscription.Subscription{
		Channel:          subscription.CandlesChannel,
		Asset:            asset.PerpetualContract,
		Pairs:            currency.Pairs{testPerpetualPair},
		Interval:         kline.OneMin,
		QualifiedChannel: "candle:perpetualcontract:BTC-USDC:1m",
	}
	require.ErrorIs(t, ex.Unsubscribe(subscription.List{missing}), subscription.ErrNotFound, "Unsubscribing an unknown channel must return the expected error")
	require.Error(t, ex.Subscribe(subscription.List{nil}), "Nil subscription expansion must error")
	require.ErrorIs(t, ex.Unsubscribe(subscription.List{nil}), common.ErrNilPointer, "Nil unsubscription must return the expected error")
}
