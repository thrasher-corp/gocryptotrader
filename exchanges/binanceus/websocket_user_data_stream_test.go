package binanceus

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// userDataStreamVenue answers User Data Stream requests as Binance.US does, failing the first failures subscriptions,
// and records each method's requests
type userDataStreamVenue struct {
	mu       sync.Mutex
	failures int
	// notActive answers unsubscriptions as Binance.US answers those of ended subscriptions
	notActive    bool
	nextID       uint64
	requests     map[string]int
	unsubscribed []map[string]string
}

func (v *userDataStreamVenue) handle(tb testing.TB, msg []byte, w *gws.Conn) error {
	tb.Helper()
	var req WebsocketAPIRequest
	require.NoError(tb, json.Unmarshal(msg, &req), "request must decode")
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.requests == nil {
		v.requests = make(map[string]int)
	}
	v.requests[req.Method]++
	var resp string
	switch req.Method {
	case "userDataStream.subscribe.signature":
		assert.NotEmpty(tb, req.Params["signature"], "a subscription should be signed")
		if v.failures > 0 {
			v.failures--
			resp = fmt.Sprintf(`{"id":%q,"status":400,"error":{"code":-1022,"msg":"Signature for this request is not valid."}}`, req.ID)
			break
		}
		resp = fmt.Sprintf(`{"id":%q,"status":200,"result":{"subscriptionId":%d}}`, req.ID, v.nextID)
		v.nextID++
	case "userDataStream.unsubscribe":
		v.unsubscribed = append(v.unsubscribed, req.Params)
		resp = fmt.Sprintf(`{"id":%q,"status":200,"result":{}}`, req.ID)
		if v.notActive {
			resp = fmt.Sprintf(`{"id":%q,"status":400,"error":{"code":-2036,"msg":"User Data Stream subscription not active."}}`, req.ID)
		}
	case "session.subscriptions":
		resp = fmt.Sprintf(`{"id":%q,"status":200,"result":[{"subscriptionId":0},{"subscriptionId":1}]}`, req.ID)
	default:
		tb.Errorf("unexpected %s request", req.Method)
		resp = fmt.Sprintf(`{"id":%q,"status":400,"error":{"code":-1,"msg":"unexpected"}}`, req.ID)
	}
	return w.WriteMessage(gws.TextMessage, []byte(resp))
}

// count returns the number of requests of method
func (v *userDataStreamVenue) count(method string) int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.requests[method]
}

// newUserDataTestExchange returns an exchange whose WebSocket API connection is served by venue, without subscribing
// the configured account's stream
func newUserDataTestExchange(t *testing.T, venue *userDataStreamVenue) *Exchange {
	t.Helper()
	ex := newWebsocketTestExchange(t)
	setup := ex.websocketAPIConnectionSetup(ex.Config)
	useMockWebsocket(t, ex, setup, venue.handle)
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	// Set after connecting, so that tests subscribe themselves
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	return ex
}

// configuredCredentials returns a copy of the exchange's configured credentials
func configuredCredentials(t *testing.T, ex *Exchange) *accounts.Credentials {
	t.Helper()
	creds := ex.GetDefaultCredentials()
	require.NotNil(t, creds, "the exchange must have configured credentials")
	return creds
}

// relayed returns the next payload the exchange relayed, failing the test when there is none
func relayed(t *testing.T, ex *Exchange) any {
	t.Helper()
	select {
	case p := <-ex.Websocket.DataHandler.C:
		return p.Data
	case <-time.After(5 * time.Second):
		require.FailNow(t, "a payload must be relayed")
		return nil
	}
}

// waitForBalanceStoreWaiter waits until a goroutine the calling goroutine started is blocked taking the balance store
// lock, reading every goroutine's stack, as nothing else shows a goroutine waiting for a mutex
func waitForBalanceStoreWaiter(t *testing.T) {
	t.Helper()
	self := make([]byte, 64)
	self = self[:runtime.Stack(self, false)]
	id, _, ok := bytes.Cut(bytes.TrimPrefix(self, []byte("goroutine ")), []byte(" "))
	require.True(t, ok, "the calling goroutine's stack must start with its ID")
	creator := []byte(" in goroutine " + string(id) + "\n")
	require.Eventually(t, func() bool {
		stacks := make([]byte, 1<<20)
		for {
			n := runtime.Stack(stacks, true)
			if n < len(stacks) {
				stacks = stacks[:n]
				break
			}
			stacks = make([]byte, 2*len(stacks))
		}
		for goroutine := range bytes.SplitSeq(stacks, []byte("\n\n")) {
			if bytes.Contains(goroutine, []byte("[sync.Mutex.Lock")) && bytes.Contains(goroutine, []byte("(*Exchange).withBalanceStore")) && bytes.Contains(goroutine, creator) {
				return true
			}
		}
		return false
	}, 5*time.Second, time.Millisecond, "a goroutine the test started must wait for the balance store lock")
}

func TestUserDataStreamsExpect(t *testing.T) {
	t.Parallel()
	var u userDataStreams
	creds := &accounts.Credentials{Key: "k"}
	u.expect("1", creds)
	assert.Same(t, creds, u.pending["1"], "expect should record the request as pending")
	assert.Same(t, creds, u.awaiting["1"], "expect should record the request as awaited")
}

func TestUserDataStreamsReturned(t *testing.T) {
	t.Parallel()
	var u userDataStreams
	u.expect("1", &accounts.Credentials{Key: "k"})
	u.returned("1")
	assert.NotContains(t, u.awaiting, "1", "returned should stop the request being awaited")
	assert.Contains(t, u.pending, "1", "returned should keep the request pending for a late response")
}

func TestUserDataStreamsRecordResponse(t *testing.T) {
	t.Parallel()
	var u userDataStreams
	assert.Nil(t, u.recordResponse("unknown", []byte(`{"result":{"subscriptionId":1}}`)), "recordResponse should return no credentials for a response to no subscription request")
	assert.Empty(t, u.subscriptions, "recordResponse should ignore a response to no subscription request")

	creds := &accounts.Credentials{Key: "k"}
	u.expect("1", creds)
	assert.Same(t, creds, u.recordResponse("1", []byte(`{"id":"1","status":200,"result":{"subscriptionId":0}}`)), "recordResponse should return the request's credentials")
	assert.Same(t, creds, u.subscriptions[0], "recordResponse should record a subscription, ID 0 included")
	assert.NotContains(t, u.pending, "1", "recordResponse should end the request's pending state")
	assert.NotContains(t, u.awaiting, "1", "recordResponse should end the request's awaited state")

	u.expect("2", creds)
	assert.Same(t, creds, u.recordResponse("2", []byte(`{"id":"2","status":400,"error":{"code":-1022,"msg":"x"}}`)), "recordResponse should return a failed request's credentials")
	assert.Len(t, u.subscriptions, 1, "recordResponse should record no subscription for a failed request")
	assert.NotContains(t, u.pending, "2", "recordResponse should end a failed request's pending state")
}

func TestParseSubscriptionID(t *testing.T) {
	t.Parallel()
	id, err := parseSubscriptionID([]byte(`{"result":{"subscriptionId":7}}`), "result", "subscriptionId")
	require.NoError(t, err, "parseSubscriptionID must not error")
	assert.Equal(t, uint64(7), id, "parseSubscriptionID should return the subscription ID")
	_, err = parseSubscriptionID([]byte(`{"result":{}}`), "result", "subscriptionId")
	assert.Error(t, err, "parseSubscriptionID should error without a subscription ID")
	_, err = parseSubscriptionID([]byte(`{"subscriptionId":-1}`), "subscriptionId")
	assert.Error(t, err, "parseSubscriptionID should error for a subscription ID that is not an unsigned integer")
}

func TestUserDataStreamsCredentials(t *testing.T) {
	t.Parallel()
	var u userDataStreams
	_, ok := u.credentials(0)
	assert.False(t, ok, "credentials should report an unknown subscription")
	creds := &accounts.Credentials{Key: "k"}
	u.expect("1", creds)
	u.recordResponse("1", []byte(`{"result":{"subscriptionId":0}}`))
	got, ok := u.credentials(0)
	require.True(t, ok, "credentials must report a recorded subscription")
	assert.Same(t, creds, got, "credentials should return the subscription's credentials")
}

func TestUserDataStreamsHas(t *testing.T) {
	t.Parallel()
	var u userDataStreams
	creds := &accounts.Credentials{Key: "k"}
	assert.False(t, u.has(nil), "has should report no subscription for no credentials")
	assert.False(t, u.has(creds), "has should report no subscription before any")
	u.expect("1", &accounts.Credentials{Key: "k"})
	assert.True(t, u.has(creds), "has should count an awaited request with equal credentials")
	u.returned("1")
	assert.False(t, u.has(creds), "has should not count a request whose sender gave up")
	u.recordResponse("1", []byte(`{"result":{"subscriptionId":2}}`))
	assert.True(t, u.has(creds), "has should count a subscription with equal credentials")
	assert.False(t, u.has(&accounts.Credentials{Key: "other"}), "has should not count another account's subscription")
}

func TestUserDataStreamsRemove(t *testing.T) {
	t.Parallel()
	var u userDataStreams
	u.expect("1", &accounts.Credentials{Key: "k"})
	u.recordResponse("1", []byte(`{"result":{"subscriptionId":2}}`))
	u.remove(2)
	_, ok := u.credentials(2)
	assert.False(t, ok, "remove should remove the subscription")
}

func TestUserDataStreamsReset(t *testing.T) {
	t.Parallel()
	var u userDataStreams
	u.expect("1", &accounts.Credentials{Key: "k"})
	u.recordResponse("1", []byte(`{"result":{"subscriptionId":2}}`))
	u.expect("3", &accounts.Credentials{Key: "k"})
	u.setWanted(false)
	u.reset()
	assert.Empty(t, u.pending, "reset should forget pending requests")
	assert.Empty(t, u.awaiting, "reset should forget awaited requests")
	assert.Empty(t, u.subscriptions, "reset should forget subscriptions")
	assert.False(t, u.isWanted(), "reset should keep a caller's unsubscription")
}

func TestUserDataStreamsSetWanted(t *testing.T) {
	t.Parallel()
	var u userDataStreams
	u.setWanted(false)
	assert.True(t, u.unsubscribed, "setWanted false should record the unsubscription")
	u.setWanted(true)
	assert.False(t, u.unsubscribed, "setWanted true should clear the unsubscription")
}

func TestUserDataStreamsIsWanted(t *testing.T) {
	t.Parallel()
	var u userDataStreams
	assert.True(t, u.isWanted(), "isWanted should report the stream wanted unless a caller unsubscribed it")
	u.setWanted(false)
	assert.False(t, u.isWanted(), "isWanted should report an unsubscribed stream unwanted")
}

func TestWebsocketAuthenticateAPI(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("mock tests serve the User Data Stream from a local venue")
		}
		ex := newWebsocketAPITestExchange(t, true)
		require.NoError(t, ex.WebsocketAuthenticateAPI(t.Context(), ex.wsAPI.get()), "WebsocketAuthenticateAPI must not error")
		subs, err := ex.WebsocketAPI().GetUserDataStreamSubscriptions(t.Context())
		require.NoError(t, err, "GetUserDataStreamSubscriptions must not error")
		assert.Len(t, subs, 1, "WebsocketAuthenticateAPI should subscribe the configured account's stream")
	})

	venue := new(userDataStreamVenue)
	ex := newUserDataTestExchange(t, venue)
	conn := ex.wsAPI.get()
	require.NoError(t, ex.WebsocketAuthenticateAPI(t.Context(), conn), "WebsocketAuthenticateAPI must not error")
	creds, ok := ex.userData.credentials(0)
	require.True(t, ok, "WebsocketAuthenticateAPI must subscribe the configured account's stream")
	assert.True(t, ex.isConfiguredAccount(creds), "the subscription should be the configured account's")

	unwanted := new(userDataStreamVenue)
	ex = newUserDataTestExchange(t, unwanted)
	ex.userData.setWanted(false)
	require.NoError(t, ex.WebsocketAuthenticateAPI(t.Context(), ex.wsAPI.get()), "WebsocketAuthenticateAPI must not error for an unwanted stream")
	assert.Zero(t, unwanted.count("userDataStream.subscribe.signature"), "WebsocketAuthenticateAPI should not subscribe a stream a caller unsubscribed")

	failing := &userDataStreamVenue{failures: 1}
	ex = newUserDataTestExchange(t, failing)
	ex.userDataRetry.delay = 10 * time.Millisecond
	require.NoError(t, ex.WebsocketAuthenticateAPI(t.Context(), ex.wsAPI.get()), "WebsocketAuthenticateAPI must not return a failed subscription")
	reported, ok := relayed(t, ex).(error)
	require.True(t, ok, "WebsocketAuthenticateAPI must relay an error")
	assert.ErrorIs(t, reported, errAPIResponse, "WebsocketAuthenticateAPI should report the failed subscription")
	assert.Eventually(t, func() bool {
		_, ok := ex.userData.credentials(0)
		return ok
	}, 5*time.Second, 10*time.Millisecond, "WebsocketAuthenticateAPI should subscribe again in the background")
}

func TestSubscribeUserDataStream(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("mock tests serve the User Data Stream from a local venue")
		}
		ex := newWebsocketAPITestExchange(t, true)
		resp, err := ex.subscribeUserDataStream(t.Context(), ex.wsAPI.get())
		require.NoError(t, err, "subscribeUserDataStream must not error")
		require.NotNil(t, resp, "subscribeUserDataStream must return the subscription")
		creds, ok := ex.userData.credentials(resp.SubscriptionID)
		require.True(t, ok, "subscribeUserDataStream must record the subscription")
		assert.True(t, ex.isConfiguredAccount(creds), "the subscription should be recorded under the configured account")
	})

	venue := new(userDataStreamVenue)
	ex := newUserDataTestExchange(t, venue)
	conn := ex.wsAPI.get()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := ex.subscribeUserDataStream(ctx, conn)
	require.ErrorIs(t, err, context.Canceled, "subscribeUserDataStream must not send a cancelled request")
	assert.Empty(t, ex.userData.pending, "a cancelled request should not be pending")

	other := &accounts.Credentials{Key: "other-api-key", Secret: "other-api-secret"}
	resp, err := ex.subscribeUserDataStream(accounts.DeployCredentialsToContext(t.Context(), other), conn)
	require.NoError(t, err, "subscribeUserDataStream must not error")
	require.NotNil(t, resp, "subscribeUserDataStream must return the subscription")
	creds, ok := ex.userData.credentials(resp.SubscriptionID)
	require.True(t, ok, "subscribeUserDataStream must record the subscription")
	assert.Equal(t, other.Key, creds.Key, "the subscription should be recorded under the credentials that signed it")
}

func TestSubscribeWantedUserDataStream(t *testing.T) {
	t.Parallel()
	venue := new(userDataStreamVenue)
	ex := newUserDataTestExchange(t, venue)
	require.NoError(t, ex.subscribeWantedUserDataStream(t.Context(), ex.wsAPI.get()), "subscribeWantedUserDataStream must not error")
	assert.Equal(t, 1, venue.count("userDataStream.subscribe.signature"), "subscribeWantedUserDataStream should subscribe a wanted stream")
	assert.Zero(t, venue.count("userDataStream.unsubscribe"), "subscribeWantedUserDataStream should keep a wanted stream's subscription")

	// A caller unsubscribing while the subscription is under way leaves the stream unwanted when its response arrives
	raced := new(userDataStreamVenue)
	ex = newUserDataTestExchange(t, raced)
	ex.userData.setWanted(false)
	require.NoError(t, ex.subscribeWantedUserDataStream(t.Context(), ex.wsAPI.get()), "subscribeWantedUserDataStream must not error for a stream unwanted meanwhile")
	assert.Equal(t, []map[string]string{{"subscriptionId": "0"}}, raced.unsubscribed, "subscribeWantedUserDataStream should end a subscription made after a caller's unsubscription")

	ended := &userDataStreamVenue{notActive: true}
	ex = newUserDataTestExchange(t, ended)
	ex.userData.setWanted(false)
	assert.NoError(t, ex.subscribeWantedUserDataStream(t.Context(), ex.wsAPI.get()), "a subscription the caller's unsubscription already ended should not error")

	failing := &userDataStreamVenue{failures: 1}
	ex = newUserDataTestExchange(t, failing)
	assert.ErrorIs(t, ex.subscribeWantedUserDataStream(t.Context(), ex.wsAPI.get()), errAPIResponse, "subscribeWantedUserDataStream should return a failed subscription")
}

func TestRetryUserDataStream(t *testing.T) {
	t.Parallel()
	venue := &userDataStreamVenue{failures: 2}
	ex := newUserDataTestExchange(t, venue)
	ex.userDataRetry.delay = 10 * time.Millisecond
	conn := ex.wsAPI.get()
	ex.retryUserDataStream(t.Context(), conn, true)
	ex.retryUserDataStream(t.Context(), conn, true)
	assert.Eventually(t, func() bool {
		_, ok := ex.userData.credentials(0)
		return ok
	}, 5*time.Second, 10*time.Millisecond, "retryUserDataStream should subscribe once the venue accepts")
	assert.Eventually(t, func() bool {
		ex.userDataRetry.mu.Lock()
		defer ex.userDataRetry.mu.Unlock()
		return ex.userDataRetry.running == nil
	}, 5*time.Second, 10*time.Millisecond, "retryUserDataStream should stop once subscribed")
	assert.Equal(t, 3, venue.count("userDataStream.subscribe.signature"), "one retry should run at a time on a connection")

	unwanted := new(userDataStreamVenue)
	ex = newUserDataTestExchange(t, unwanted)
	ex.userData.setWanted(false)
	ex.retryUserDataStream(t.Context(), ex.wsAPI.get(), true)
	assert.Eventually(t, func() bool {
		ex.userDataRetry.mu.Lock()
		defer ex.userDataRetry.mu.Unlock()
		return ex.userDataRetry.running == nil
	}, 5*time.Second, 10*time.Millisecond, "retryUserDataStream should stop for an unwanted stream")
	assert.Zero(t, unwanted.count("userDataStream.subscribe.signature"), "retryUserDataStream should not subscribe an unwanted stream")

	waiting := new(userDataStreamVenue)
	ex = newUserDataTestExchange(t, waiting)
	ex.userDataRetry.delay = time.Hour
	ex.retryUserDataStream(t.Context(), ex.wsAPI.get(), false)
	require.NoError(t, ex.Websocket.Shutdown(), "Shutdown must not error")
	ex.userDataRetry.mu.Lock()
	assert.Nil(t, ex.userDataRetry.running, "a websocket shutdown should end a waiting retry before it returns")
	ex.userDataRetry.mu.Unlock()
	assert.Zero(t, waiting.count("userDataStream.subscribe.signature"), "a retry the websocket shut down should not subscribe")

	unauthenticated := new(userDataStreamVenue)
	ex = newUserDataTestExchange(t, unauthenticated)
	ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
	ex.retryUserDataStream(t.Context(), ex.wsAPI.get(), true)
	assert.Eventually(t, func() bool {
		ex.userDataRetry.mu.Lock()
		defer ex.userDataRetry.mu.Unlock()
		return ex.userDataRetry.running == nil
	}, 5*time.Second, 10*time.Millisecond, "retryUserDataStream should stop while authenticated websocket use is off")
	assert.Zero(t, unauthenticated.count("userDataStream.subscribe.signature"), "retryUserDataStream should not subscribe while authenticated websocket use is off")
}

func TestNextUserDataRetryWait(t *testing.T) {
	t.Parallel()
	assert.Equal(t, time.Second, nextUserDataRetryWait(0, time.Second), "the first wait after an immediate attempt should be the retry delay")
	assert.Equal(t, 2*time.Second, nextUserDataRetryWait(time.Second, time.Second), "a later wait should double the last one")
	assert.Equal(t, userDataRetryMaxDelay, nextUserDataRetryWait(userDataRetryMaxDelay, time.Second), "a wait should not exceed the maximum")
}

func TestReportUserDataStreamError(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	ex.Websocket.DataHandler = stream.NewRelay(1)
	ex.reportUserDataStreamError(t.Context(), errAPIResponse)
	reported, ok := relayed(t, ex).(error)
	require.True(t, ok, "reportUserDataStreamError must relay an error")
	assert.ErrorIs(t, reported, errAPIResponse, "reportUserDataStreamError should relay the error")
	require.NoError(t, ex.Websocket.DataHandler.Send(t.Context(), "fill"), "Send must not error")
	assert.NotPanics(t, func() { ex.reportUserDataStreamError(t.Context(), errAPIResponse) }, "reportUserDataStreamError should log an error the data handler cannot take")
}

func TestIsConfiguredAccount(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.True(t, ex.isConfiguredAccount(configuredCredentials(t, ex)), "isConfiguredAccount should report the configured credentials")
	assert.False(t, ex.isConfiguredAccount(&accounts.Credentials{Key: "other"}), "isConfiguredAccount should not report other credentials")
	assert.False(t, ex.isConfiguredAccount(nil), "isConfiguredAccount should not report no credentials")
}

func TestWebsocketAPISubscribeUserDataStream(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("mock tests serve the User Data Stream from a local venue")
		}
		ex := newWebsocketAPITestExchange(t, true)
		resp, err := ex.WebsocketAPI().SubscribeUserDataStream(t.Context())
		require.NoError(t, err, "SubscribeUserDataStream must not error")
		require.NotNil(t, resp, "SubscribeUserDataStream must return the subscription")
		subs, err := ex.WebsocketAPI().GetUserDataStreamSubscriptions(t.Context())
		require.NoError(t, err, "GetUserDataStreamSubscriptions must not error")
		assert.Contains(t, subs, *resp, "GetUserDataStreamSubscriptions should list the subscription")
		assert.NoError(t, ex.WebsocketAPI().UnsubscribeUserDataStream(t.Context(), resp.SubscriptionID), "UnsubscribeUserDataStream should not error")
	})

	_, err := newWebsocketTestExchange(t).WebsocketAPI().SubscribeUserDataStream(t.Context())
	require.ErrorIs(t, err, websocket.ErrNotConnected, "SubscribeUserDataStream must error while disconnected")

	venue := new(userDataStreamVenue)
	ex := newUserDataTestExchange(t, venue)
	ex.userData.setWanted(false)
	resp, err := ex.WebsocketAPI().SubscribeUserDataStream(t.Context())
	require.NoError(t, err, "SubscribeUserDataStream must not error")
	assert.Equal(t, &WebsocketUserDataStreamSubscriptionResponse{SubscriptionID: 0}, resp, "SubscribeUserDataStream should return the subscription")
	assert.True(t, ex.userData.isWanted(), "subscribing the configured account should want its stream again")
}

func TestWebsocketAPIUnsubscribeUserDataStream(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("mock tests serve the User Data Stream from a local venue")
		}
		err := newWebsocketAPITestExchange(t, false).WebsocketAPI().UnsubscribeUserDataStream(t.Context(), 0)
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr, "UnsubscribeUserDataStream must return Binance.US's refusal on a connection without subscriptions")
		assert.Equal(t, int64(userDataStreamNotActiveCode), apiErr.Code, "UnsubscribeUserDataStream should be refused as the subscription is not active")
	})

	venue := new(userDataStreamVenue)
	ex := newUserDataTestExchange(t, venue)
	resp, err := ex.WebsocketAPI().SubscribeUserDataStream(t.Context())
	require.NoError(t, err, "SubscribeUserDataStream must not error")
	require.NoError(t, ex.WebsocketAPI().UnsubscribeUserDataStream(t.Context(), resp.SubscriptionID), "UnsubscribeUserDataStream must not error")
	assert.Equal(t, []map[string]string{{"subscriptionId": "0"}}, venue.unsubscribed, "UnsubscribeUserDataStream should name the subscription")
	assert.False(t, ex.userData.isWanted(), "unsubscribing the configured account should leave its stream unwanted")
}

func TestWebsocketAPIUnsubscribeAllUserDataStreams(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("mock tests serve the User Data Stream from a local venue")
		}
		err := newWebsocketAPITestExchange(t, false).WebsocketAPI().UnsubscribeAllUserDataStreams(t.Context())
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr, "UnsubscribeAllUserDataStreams must return Binance.US's refusal on a connection without subscriptions")
		assert.Equal(t, int64(userDataStreamNotActiveCode), apiErr.Code, "UnsubscribeAllUserDataStreams should be refused as no subscription is active")
	})

	venue := new(userDataStreamVenue)
	ex := newUserDataTestExchange(t, venue)
	require.NoError(t, ex.WebsocketAPI().UnsubscribeAllUserDataStreams(t.Context()), "UnsubscribeAllUserDataStreams must not error")
	require.Len(t, venue.unsubscribed, 1, "UnsubscribeAllUserDataStreams must send one request")
	assert.Empty(t, venue.unsubscribed[0], "UnsubscribeAllUserDataStreams should name no subscription")
	assert.False(t, ex.userData.isWanted(), "unsubscribing every stream should leave the configured account's unwanted")
}

func TestWebsocketAPIGetUserDataStreamSubscriptions(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("mock tests serve the User Data Stream from a local venue")
		}
		resp, err := newWebsocketAPITestExchange(t, false).WebsocketAPI().GetUserDataStreamSubscriptions(t.Context())
		require.NoError(t, err, "GetUserDataStreamSubscriptions must not error")
		assert.Empty(t, resp, "a new connection should have no User Data Stream subscription")
	})

	venue := new(userDataStreamVenue)
	ex := newUserDataTestExchange(t, venue)
	resp, err := ex.WebsocketAPI().GetUserDataStreamSubscriptions(t.Context())
	require.NoError(t, err, "GetUserDataStreamSubscriptions must not error")
	assert.Equal(t, []WebsocketUserDataStreamSubscriptionResponse{{SubscriptionID: 0}, {SubscriptionID: 1}}, resp, "GetUserDataStreamSubscriptions should decode every subscription")
}

// websocketAPIListenKey is the listen key of the documented WebSocket API examples the user data stream fixtures record
const websocketAPIListenKey = "xs0mRXdAKlIPDRFrlPcw0qI41Eh3ixNntmymGyhrhgqo7L6FuLaWArTD7RLP"

func TestWebsocketAPICreateUserDataStream(t *testing.T) {
	t.Parallel()
	ex := newWebsocketAPITestExchange(t, true)
	result, err := ex.WebsocketAPI().CreateUserDataStream(t.Context())
	require.NoError(t, err, "CreateUserDataStream must not error")
	require.NotNil(t, result, "CreateUserDataStream must return a listen key")
	if !mockTests {
		assert.NotEmpty(t, result.ListenKey, "CreateUserDataStream should return a listen key")
		return
	}
	assert.Equal(t, &ListenKeyResponse{ListenKey: websocketAPIListenKey}, result, "CreateUserDataStream should decode the listen key")
}

// TestWebsocketAPIExtendUserDataStream is not parallel: an account has one listen key at a time, which
// TestWebsocketAPICloseUserDataStream closes in live tests
func TestWebsocketAPIExtendUserDataStream(t *testing.T) {
	ex := newWebsocketAPITestExchange(t, true)
	listenKey := testListenKey
	if !mockTests {
		resp, err := ex.WebsocketAPI().CreateUserDataStream(t.Context())
		require.NoError(t, err, "CreateUserDataStream must not error")
		require.NotNil(t, resp, "CreateUserDataStream must return a listen key")
		listenKey = resp.ListenKey
	}
	assert.NoError(t, ex.WebsocketAPI().ExtendUserDataStream(t.Context(), listenKey), "ExtendUserDataStream should not error")
}

// TestWebsocketAPICloseUserDataStream is not parallel: an account has one listen key at a time, which
// TestWebsocketAPIExtendUserDataStream extends in live tests
func TestWebsocketAPICloseUserDataStream(t *testing.T) {
	ex := newWebsocketAPITestExchange(t, true)
	listenKey := websocketAPIListenKey
	if !mockTests {
		resp, err := ex.WebsocketAPI().CreateUserDataStream(t.Context())
		require.NoError(t, err, "CreateUserDataStream must not error")
		require.NotNil(t, resp, "CreateUserDataStream must return a listen key")
		listenKey = resp.ListenKey
	}
	assert.NoError(t, ex.WebsocketAPI().CloseUserDataStream(t.Context(), listenKey), "CloseUserDataStream should not error")
}

// subscribedUserDataTestExchange returns an exchange with a User Data Stream subscription 0 of its configured account
// and a relay to read what it relays
func subscribedUserDataTestExchange(t *testing.T) *Exchange {
	t.Helper()
	ex := newWebsocketTestExchange(t)
	ex.Websocket = sharedtestvalues.NewTestWebsocket()
	ex.Websocket.DataHandler = stream.NewRelay(sharedtestvalues.WebsocketRelayBufferCapacity)
	ex.userData.expect("1", configuredCredentials(t, ex))
	ex.userData.recordResponse("1", []byte(`{"result":{"subscriptionId":0}}`))
	return ex
}

func TestHandleUserDataEvent(t *testing.T) {
	t.Parallel()
	ex := subscribedUserDataTestExchange(t)
	creds := configuredCredentials(t, ex)
	conn := newFakeConnection()
	handle := func(eventType, event string) error {
		return ex.handleUserDataEvent(t.Context(), conn, []byte(`{"subscriptionId":0,"event":`+event+`}`), []byte(event), eventType)
	}

	require.NoError(t, handle("outboundAccountPosition", `{"e":"outboundAccountPosition","E":1564034571105,"u":1564034571073,"B":[{"a":"ETH","f":"10000.000000","l":"0.500000"}]}`), "an outboundAccountPosition event must not error")
	_, ok := relayed(t, ex).(accounts.SubAccounts)
	assert.True(t, ok, "an outboundAccountPosition event should be relayed as sub-accounts")
	stored, err := ex.Accounts.GetBalance("", creds, asset.Spot, currency.ETH)
	require.NoError(t, err, "GetBalance must not error")
	assert.Equal(t, 10000.5, stored.Total, "an outboundAccountPosition event should store the subscription's account balance")

	require.NoError(t, handle("balanceUpdate", `{"e":"balanceUpdate","E":1573200697110,"a":"BTC","d":"100.00000000","T":1573200697068}`), "a balanceUpdate event must not error")
	assert.Equal(t, &WebsocketBalanceUpdate{
		EventType:    "balanceUpdate",
		EventTime:    types.Time(time.UnixMilli(1573200697110)),
		Asset:        currency.BTC,
		BalanceDelta: 100,
		ClearTime:    types.Time(time.UnixMilli(1573200697068)),
	}, relayed(t, ex), "a balanceUpdate event should be relayed decoded")

	require.NoError(t, handle("executionReport", `{"e":"executionReport","E":1499405658658,"s":"BTCUSDT","c":"mUvoqJxFIILMdfAW5iGSOW","S":"BUY","o":"LIMIT","f":"GTC","q":"1","p":"0.1","x":"NEW","X":"NEW","i":4293153,"T":1499405658657,"O":1499405658657}`), "an executionReport event must not error")
	detail, ok := relayed(t, ex).(*order.Detail)
	require.True(t, ok, "an executionReport event must be relayed as an order detail")
	assert.Equal(t, "4293153", detail.OrderID, "an executionReport event should be relayed as its order's detail")

	require.NoError(t, handle("listStatus", `{"e":"listStatus","E":1564035303637,"s":"BTCUSDT","g":2,"c":"OCO","l":"EXEC_STARTED","L":"EXECUTING","r":"NONE","C":"F4QN4G8DlFATFlIUQ0cjdD","T":1564035303625,"O":[{"s":"BTCUSDT","i":17,"c":"AJYsMjErWJesZvqlJCTUgL"},{"s":"BTCUSDT","i":18,"c":"bfYPSQdLoqAJeNrOr9adzq"}]}`), "a listStatus event must not error")
	listStatus, ok := relayed(t, ex).(*WebsocketListStatus)
	require.True(t, ok, "a listStatus event must be relayed decoded")
	assert.Equal(t, uint64(2), listStatus.OrderListID, "a listStatus event should be relayed as its order list's")
	detail, ok = relayed(t, ex).(*order.Detail)
	require.True(t, ok, "an OCO order list's listStatus event must be relayed as an order detail")
	assert.Equal(t, "list-2", detail.OrderID, "an OCO order list's listStatus event should be relayed as the order GCT tracks")

	require.NoError(t, handle("externalLockUpdate", `{"e":"externalLockUpdate","E":1728973001334,"a":"BTC","d":"-0.10000000","T":1728973001335}`), "an externalLockUpdate event must not error")
	assert.Equal(t, &WebsocketExternalLockUpdate{
		EventType:       "externalLockUpdate",
		EventTime:       types.Time(time.UnixMilli(1728973001334)),
		Asset:           currency.BTC,
		Delta:           -0.1,
		TransactionTime: types.Time(time.UnixMilli(1728973001335)),
	}, relayed(t, ex), "an externalLockUpdate event should be relayed decoded")

	assert.ErrorIs(t, handle("unknownEvent", `{"e":"unknownEvent","E":1}`), errUnhandledMessage, "an unknown event should be unhandled")
	err = ex.handleUserDataEvent(t.Context(), conn, []byte(`{"event":{"e":"balanceUpdate","E":1}}`), []byte(`{"e":"balanceUpdate","E":1}`), "balanceUpdate")
	assert.ErrorIs(t, err, errUserDataEventWithoutSubscription, "an event without a subscription ID should error")
	err = ex.handleUserDataEvent(t.Context(), conn, []byte(`{"subscriptionId":9,"event":{"e":"balanceUpdate","E":1}}`), []byte(`{"e":"balanceUpdate","E":1}`), "balanceUpdate")
	assert.ErrorIs(t, err, errUnknownUserDataStreamSubscription, "an event of an unknown subscription should error")

	ex.userData.setWanted(false)
	require.NoError(t, handle("eventStreamTerminated", `{"e":"eventStreamTerminated","E":1728973001334}`), "an eventStreamTerminated event must not error")
	assert.Equal(t, &WebsocketEventStreamTerminated{EventType: "eventStreamTerminated", EventTime: types.Time(time.UnixMilli(1728973001334))}, relayed(t, ex), "an eventStreamTerminated event should be relayed decoded")
	_, ok = ex.userData.credentials(0)
	assert.False(t, ok, "an eventStreamTerminated event should end its subscription")
	ex.userDataRetry.mu.Lock()
	assert.Nil(t, ex.userDataRetry.running, "an eventStreamTerminated event should not resubscribe an unwanted stream")
	ex.userDataRetry.mu.Unlock()

	t.Run("a terminated wanted stream is subscribed again", func(t *testing.T) {
		t.Parallel()
		venue := new(userDataStreamVenue)
		ex := newUserDataTestExchange(t, venue)
		conn := ex.wsAPI.get()
		require.NoError(t, ex.WebsocketAuthenticateAPI(t.Context(), conn), "WebsocketAuthenticateAPI must not error")
		event := `{"e":"eventStreamTerminated","E":1728973001334}`
		require.NoError(t, ex.handleUserDataEvent(t.Context(), conn, []byte(`{"subscriptionId":0,"event":`+event+`}`), []byte(event), "eventStreamTerminated"), "an eventStreamTerminated event must not error")
		assert.Eventually(t, func() bool {
			_, ok := ex.userData.credentials(1)
			return ok
		}, 5*time.Second, 10*time.Millisecond, "an eventStreamTerminated event should resubscribe the configured account's wanted stream")
	})
}

func TestWithBalanceStore(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, ex.withBalanceStore(func() error {
		assert.False(t, ex.balances.storeMu.TryLock(), "withBalanceStore should hold the balance store lock")
		return nil
	}), "withBalanceStore must not error")
	assert.True(t, ex.balances.storeMu.TryLock(), "withBalanceStore should release the lock")
	ex.balances.storeMu.Unlock()
	assert.ErrorIs(t, ex.withBalanceStore(func() error { return errAPIResponse }), errAPIResponse, "withBalanceStore should return f's error")
}

func TestBalanceOrderingTimesOf(t *testing.T) {
	t.Parallel()
	recorded := func() *accounts.Credentials {
		return &accounts.Credentials{Key: "account-a-key", Secret: "account-a-secret", ClientID: "account-a-client", PEMKey: "account-a-pem", SubAccount: "account-a-sub", OneTimePassword: "account-a-otp"}
	}
	record := func(b *balanceOrdering) *balanceTimes {
		times := b.timesOf(recorded())
		times.eventTimes = map[*currency.Item]time.Time{currency.BTC.Item: time.UnixMilli(1000)}
		times.snapshotTime = time.UnixMilli(2000)
		return times
	}

	t.Run("the account the times were recorded for has them", func(t *testing.T) {
		t.Parallel()
		var b balanceOrdering
		times := record(&b)
		assert.Same(t, times, b.timesOf(recorded()), "timesOf should return the times recorded for the account")
	})

	for _, tc := range []struct {
		name  string
		other func(*accounts.Credentials)
	}{
		{name: "an account with another key has none", other: func(c *accounts.Credentials) { c.Key = "account-b-key" }},
		{name: "an account with another secret has none", other: func(c *accounts.Credentials) { c.Secret = "account-b-secret" }},
		{name: "an account with another client ID has none", other: func(c *accounts.Credentials) { c.ClientID = "account-b-client" }},
		{name: "an account with another PEM key has none", other: func(c *accounts.Credentials) { c.PEMKey = "account-b-pem" }},
		{name: "an account with another sub-account has none", other: func(c *accounts.Credentials) { c.SubAccount = "account-b-sub" }},
		{name: "an account with another one-time password has none", other: func(c *accounts.Credentials) { c.OneTimePassword = "account-b-otp" }},
		{name: "an account with its secret decoded from base64 has none", other: func(c *accounts.Credentials) { c.SecretBase64Decoded = true }},
		{name: "a nil account has none"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var b balanceOrdering
			record(&b)
			var other *accounts.Credentials
			if tc.other != nil {
				other = recorded()
				tc.other(other)
			}
			got := b.timesOf(other)
			assert.Empty(t, got.eventTimes, "timesOf should return no event times for another account")
			assert.Zero(t, got.snapshotTime, "timesOf should return no snapshot time for another account")
			assert.Equal(t, map[*currency.Item]time.Time{currency.BTC.Item: time.UnixMilli(1000)}, b.timesOf(recorded()).eventTimes, "timesOf should keep the event times recorded for the first account")
			assert.Equal(t, time.UnixMilli(2000), b.timesOf(recorded()).snapshotTime, "timesOf should keep the snapshot time recorded for the first account")
		})
	}

	t.Run("an account keeps its times beside two others", func(t *testing.T) {
		t.Parallel()
		var b balanceOrdering
		times := record(&b)
		b.timesOf(&accounts.Credentials{Key: "account-b-key", Secret: "account-b-secret"}).snapshotTime = time.UnixMilli(3000)
		b.timesOf(&accounts.Credentials{Key: "account-c-key", Secret: "account-c-secret"}).snapshotTime = time.UnixMilli(4000)
		assert.Same(t, times, b.timesOf(recorded()), "timesOf should return the first account's times after two others'")
		assert.Equal(t, time.UnixMilli(2000), times.snapshotTime, "timesOf should keep the first account's snapshot time")
	})
}

// balanceTestAccount returns the account the balance ordering tests record times for
func balanceTestAccount() *accounts.Credentials {
	return &accounts.Credentials{Key: "account-a-key", Secret: "account-a-secret", ClientID: "account-a-client", SubAccount: "account-a-sub"}
}

// otherBalanceTestAccounts returns accounts differing from balanceTestAccount's in one credential each, by what differs
func otherBalanceTestAccounts() []struct {
	differs string
	account *accounts.Credentials
} {
	others := []struct {
		differs string
		account *accounts.Credentials
	}{{differs: "key"}, {differs: "secret"}, {differs: "client ID"}, {differs: "sub-account"}}
	for i := range others {
		others[i].account = balanceTestAccount()
		switch others[i].differs {
		case "key":
			others[i].account.Key = "account-b-key"
		case "secret":
			others[i].account.Secret = "account-b-secret"
		case "client ID":
			others[i].account.ClientID = "account-b-client"
		case "sub-account":
			others[i].account.SubAccount = "account-b-sub"
		}
	}
	return others
}

func TestRecordBalanceEvent(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	account := balanceTestAccount()
	later, earlier := time.UnixMilli(2000), time.UnixMilli(1000)
	ex.recordBalanceEvent(account, later, currency.BTC)
	ex.recordBalanceEvent(account, earlier, currency.BTC, currency.ETH)
	assert.Equal(t, map[*currency.Item]time.Time{currency.BTC.Item: later, currency.ETH.Item: earlier}, ex.balanceEventTimes(account), "recordBalanceEvent should keep each currency's latest event")

	for _, other := range otherBalanceTestAccounts() {
		t.Run("keeps the latest events of an account with another "+other.differs+" apart", func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			ex.recordBalanceEvent(balanceTestAccount(), time.UnixMilli(2000), currency.BTC)
			ex.recordBalanceEvent(other.account, time.UnixMilli(1000), currency.BTC, currency.ETH)
			ex.recordBalanceEvent(other.account, time.UnixMilli(500), currency.BTC)
			assert.Equal(t, map[*currency.Item]time.Time{currency.BTC.Item: time.UnixMilli(1000), currency.ETH.Item: time.UnixMilli(1000)}, ex.balanceEventTimes(other.account), "recordBalanceEvent should keep the other account's latest events apart from the first account's")
			assert.Equal(t, map[*currency.Item]time.Time{currency.BTC.Item: time.UnixMilli(2000)}, ex.balanceEventTimes(balanceTestAccount()), "recordBalanceEvent should keep the first account's events apart from the other account's")
		})
	}
}

func TestRecordBalanceSnapshot(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	account := balanceTestAccount()
	ex.recordBalanceSnapshot(account, time.UnixMilli(2000))
	ex.recordBalanceSnapshot(account, time.UnixMilli(1000))
	assert.Equal(t, time.UnixMilli(2000), ex.balanceSnapshotTime(account), "recordBalanceSnapshot should keep the latest snapshot's time")

	for _, other := range otherBalanceTestAccounts() {
		t.Run("keeps the latest snapshot of an account with another "+other.differs+" apart", func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			ex.recordBalanceSnapshot(balanceTestAccount(), time.UnixMilli(2000))
			ex.recordBalanceSnapshot(other.account, time.UnixMilli(1500))
			ex.recordBalanceSnapshot(other.account, time.UnixMilli(1000))
			assert.Equal(t, time.UnixMilli(1500), ex.balanceSnapshotTime(other.account), "recordBalanceSnapshot should keep the other account's latest snapshot apart from the first account's")
			assert.Equal(t, time.UnixMilli(2000), ex.balanceSnapshotTime(balanceTestAccount()), "recordBalanceSnapshot should keep the first account's snapshot apart from the other account's")
		})
	}
}

func TestBalanceSnapshotTime(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	account := balanceTestAccount()
	assert.Zero(t, ex.balanceSnapshotTime(account), "balanceSnapshotTime should be zero before any snapshot")
	ex.recordBalanceSnapshot(account, time.UnixMilli(1000))
	assert.Equal(t, time.UnixMilli(1000), ex.balanceSnapshotTime(account), "balanceSnapshotTime should return the snapshot's time")

	for _, other := range otherBalanceTestAccounts() {
		t.Run("is zero for an account with another "+other.differs+" without a snapshot", func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			ex.recordBalanceSnapshot(balanceTestAccount(), time.UnixMilli(2000))
			assert.Zero(t, ex.balanceSnapshotTime(other.account), "balanceSnapshotTime should be zero for the other account")
			assert.Equal(t, time.UnixMilli(2000), ex.balanceSnapshotTime(balanceTestAccount()), "balanceSnapshotTime should return the first account's snapshot time")
		})
	}
}

func TestBalanceEventTimes(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	account := balanceTestAccount()
	ex.recordBalanceEvent(account, time.UnixMilli(1000), currency.BTC)
	times := ex.balanceEventTimes(account)
	require.Equal(t, map[*currency.Item]time.Time{currency.BTC.Item: time.UnixMilli(1000)}, times, "balanceEventTimes must return the recorded event times")
	times[currency.ETH.Item] = time.UnixMilli(1)
	assert.NotContains(t, ex.balanceEventTimes(account), currency.ETH.Item, "balanceEventTimes should return a copy")

	for _, other := range otherBalanceTestAccounts() {
		t.Run("is empty for an account with another "+other.differs+" without events", func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			ex.recordBalanceEvent(balanceTestAccount(), time.UnixMilli(2000), currency.BTC)
			assert.Empty(t, ex.balanceEventTimes(other.account), "balanceEventTimes should return no times for the other account")
			assert.Equal(t, map[*currency.Item]time.Time{currency.BTC.Item: time.UnixMilli(2000)}, ex.balanceEventTimes(balanceTestAccount()), "balanceEventTimes should return the first account's event times")
		})
	}
}

func TestKeepStreamedSpotBalances(t *testing.T) {
	t.Parallel()
	ex := subscribedUserDataTestExchange(t)
	creds := configuredCredentials(t, ex)
	ctx := t.Context()
	asOf := time.UnixMilli(2000)
	stored := accounts.NewSubAccount(asset.Spot, "")
	stored.Balances.Set(currency.BTC, accounts.Balance{Total: 1, Free: 1})
	stored.Balances.Set(currency.ETH, accounts.Balance{Total: 5, Free: 5})
	stored.Balances.Set(currency.LTC, accounts.Balance{})
	require.NoError(t, ex.Accounts.Save(ctx, accounts.SubAccounts{stored}, true), "Save must not error")
	// BTC and LTC, which reached zero, were streamed after the snapshot, ETH in its millisecond and USDT before it
	ex.recordBalanceEvent(creds, time.UnixMilli(3000), currency.BTC, currency.LTC)
	ex.recordBalanceEvent(creds, asOf, currency.ETH)
	ex.recordBalanceEvent(creds, time.UnixMilli(1000), currency.USDT)

	snapshot := accounts.NewSubAccount(asset.Spot, "")
	snapshot.Balances.Set(currency.BTC, accounts.Balance{Total: 9, Free: 9})
	snapshot.Balances.Set(currency.LTC, accounts.Balance{Total: 4, Free: 4})
	snapshot.Balances.Set(currency.USDT, accounts.Balance{Total: 7, Free: 7})
	polled := maps.Clone(snapshot.Balances)
	ex.keepStreamedSpotBalances(creds, snapshot, time.Time{})
	assert.Equal(t, polled, snapshot.Balances, "a snapshot without a time should be left as it is")
	ex.keepStreamedSpotBalances(&accounts.Credentials{Key: "other-api-key", Secret: "other-api-secret"}, snapshot, asOf)
	assert.Equal(t, polled, snapshot.Balances, "a snapshot of an account without events should be left as it is")
	ex.keepStreamedSpotBalances(creds, snapshot, asOf)
	assert.Equal(t, 1.0, snapshot.Balances[currency.BTC].Total, "a currency streamed after the snapshot should keep its stored balance")
	require.Contains(t, snapshot.Balances, currency.LTC, "a currency streamed to zero after the snapshot must stay in the snapshot")
	assert.Zero(t, snapshot.Balances[currency.LTC].Total, "a currency streamed to zero after the snapshot should keep its zero balance")
	assert.Equal(t, 5.0, snapshot.Balances[currency.ETH].Total, "a currency the snapshot left out but streamed in its millisecond should keep its stored balance")
	assert.Equal(t, 7.0, snapshot.Balances[currency.USDT].Total, "a currency streamed before the snapshot should take the snapshot's balance")
}

func TestProcessAccountPosition(t *testing.T) {
	t.Parallel()
	ex := subscribedUserDataTestExchange(t)
	creds := configuredCredentials(t, ex)
	ctx := accounts.DeployCredentialsToContext(t.Context(), creds)
	event := []byte(`{"e":"outboundAccountPosition","E":1564034571105,"u":1564034571073,"B":[{"a":"ETH","f":"10000.000000","l":"0.500000"}]}`)
	require.NoError(t, ex.processAccountPosition(ctx, creds, event), "processAccountPosition must not error")
	subAccounts, ok := relayed(t, ex).(accounts.SubAccounts)
	require.True(t, ok, "processAccountPosition must relay the balances as sub-accounts")
	require.Len(t, subAccounts, 1, "processAccountPosition must relay one sub-account")
	balance := subAccounts[0].Balances[currency.ETH]
	assert.Equal(t, 10000.5, balance.Total, "the total should add the free and locked balances")
	assert.Equal(t, 10000.0, balance.Free, "the free balance should be the event's")
	assert.Equal(t, 0.5, balance.Hold, "the held balance should be the event's locked balance")
	stored, err := ex.Accounts.GetBalance("", creds, asset.Spot, currency.ETH)
	require.NoError(t, err, "GetBalance must not error")
	assert.Equal(t, 10000.5, stored.Total, "processAccountPosition should store the balance")
	assert.Equal(t, map[*currency.Item]time.Time{currency.ETH.Item: time.UnixMilli(1564034571073)}, ex.balanceEventTimes(creds), "processAccountPosition should record the event's account update time")

	ex.recordBalanceSnapshot(creds, time.UnixMilli(1564034571074))
	stale := []byte(`{"e":"outboundAccountPosition","E":1564034571105,"u":1564034571073,"B":[{"a":"ETH","f":"1","l":"0"}]}`)
	require.NoError(t, ex.processAccountPosition(ctx, creds, stale), "processAccountPosition must not error for an event predating the snapshot")
	stored, err = ex.Accounts.GetBalance("", creds, asset.Spot, currency.ETH)
	require.NoError(t, err, "GetBalance must not error")
	assert.Equal(t, 10000.5, stored.Total, "an event predating the stored snapshot should be left out")
	assert.Empty(t, ex.Websocket.DataHandler.C, "an event left out should relay nothing")

	assert.Error(t, ex.processAccountPosition(ctx, creds, []byte(`{"B":"x"}`)), "processAccountPosition should error for an event that does not decode")

	for _, tc := range []struct {
		name     string
		snapshot int64
		events   []string
		btc, usd float64
	}{
		{
			name:     "an event sharing the stored snapshot's account update time is stored",
			snapshot: 1795087200000,
			events:   []string{`{"e":"outboundAccountPosition","E":1795087200003,"u":1795087200000,"B":[{"a":"BTC","f":"1.50000000","l":"0.00000000"}]}`},
			btc:      1.5,
		},
		{
			name: "a later event in the same millisecond replaces the balance",
			events: []string{
				`{"e":"outboundAccountPosition","E":1795087200003,"u":1795087200000,"B":[{"a":"BTC","f":"1.25000000","l":"0.00000000"}]}`,
				`{"e":"outboundAccountPosition","E":1795087200004,"u":1795087200000,"B":[{"a":"BTC","f":"1.50000000","l":"0.00000000"}]}`,
			},
			btc: 1.5,
		},
		{
			name: "a later event published in the same millisecond replaces the balance",
			events: []string{
				`{"e":"outboundAccountPosition","E":1795087200000,"u":1795087200000,"B":[{"a":"BTC","f":"1.25000000","l":"0.00000000"}]}`,
				`{"e":"outboundAccountPosition","E":1795087200000,"u":1795087200000,"B":[{"a":"BTC","f":"1.50000000","l":"0.00000000"}]}`,
			},
			btc: 1.5,
		},
		{
			name:   "each currency of an event is stored",
			events: []string{`{"e":"outboundAccountPosition","E":1795087200003,"u":1795087200000,"B":[{"a":"BTC","f":"1.25000000","l":"0.00000000"},{"a":"USD","f":"623.00000000","l":"15.00000000"}]}`},
			btc:    1.25,
			usd:    638,
		},
		{
			name: "a balance reaching zero is stored",
			events: []string{
				`{"e":"outboundAccountPosition","E":1795087200003,"u":1795087200000,"B":[{"a":"BTC","f":"1.25000000","l":"0.00000000"}]}`,
				`{"e":"outboundAccountPosition","E":1795087201003,"u":1795087201000,"B":[{"a":"BTC","f":"0.00000000","l":"0.00000000"}]}`,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := subscribedUserDataTestExchange(t)
			creds := configuredCredentials(t, ex)
			ctx := accounts.DeployCredentialsToContext(t.Context(), creds)
			if tc.snapshot != 0 {
				ex.recordBalanceSnapshot(creds, time.UnixMilli(tc.snapshot))
			}
			for i, event := range tc.events {
				require.NoErrorf(t, ex.processAccountPosition(ctx, creds, []byte(event)), "processAccountPosition must not error for event %d", i)
				_, ok := relayed(t, ex).(accounts.SubAccounts)
				require.Truef(t, ok, "processAccountPosition must relay event %d's balances", i)
			}
			stored, err := ex.Accounts.GetBalance("", creds, asset.Spot, currency.BTC)
			require.NoError(t, err, "GetBalance must not error for BTC")
			assert.Equal(t, tc.btc, stored.Total, "processAccountPosition should store the latest BTC balance")
			if tc.usd != 0 {
				stored, err = ex.Accounts.GetBalance("", creds, asset.Spot, currency.USD)
				require.NoError(t, err, "GetBalance must not error for USD")
				assert.Equal(t, tc.usd, stored.Total, "processAccountPosition should store each currency of the event")
				updated := time.UnixMilli(1795087200000)
				assert.Equal(t, map[*currency.Item]time.Time{currency.BTC.Item: updated, currency.USD.Item: updated}, ex.balanceEventTimes(creds), "processAccountPosition should record the event's account update time for each of its currencies")
			}
		})
	}

	t.Run("another account's event is stored for that account and leaves the configured account's event times alone", func(t *testing.T) {
		t.Parallel()
		ex := subscribedUserDataTestExchange(t)
		creds := configuredCredentials(t, ex)
		require.NoError(t, ex.processAccountPosition(accounts.DeployCredentialsToContext(t.Context(), creds), creds, []byte(`{"e":"outboundAccountPosition","E":1795087199003,"u":1795087199000,"B":[{"a":"USD","f":"623.00000000","l":"0.00000000"}]}`)), "processAccountPosition must not error for the configured account's event")
		_, ok := relayed(t, ex).(accounts.SubAccounts)
		require.True(t, ok, "processAccountPosition must relay the configured account's balances")
		other := &accounts.Credentials{Key: "other-api-key", Secret: "other-api-secret"}
		otherCtx := accounts.DeployCredentialsToContext(t.Context(), other)
		require.NoError(t, ex.processAccountPosition(otherCtx, other, []byte(`{"e":"outboundAccountPosition","E":1795087200003,"u":1795087200000,"B":[{"a":"BTC","f":"1.00000000","l":"0.00000000"}]}`)), "processAccountPosition must not error for another account's event")
		_, ok = relayed(t, ex).(accounts.SubAccounts)
		require.True(t, ok, "processAccountPosition must relay another account's balances")
		stored, err := ex.Accounts.GetBalance("", other, asset.Spot, currency.BTC)
		require.NoError(t, err, "GetBalance must not error for the other account")
		assert.Equal(t, 1.0, stored.Total, "another account's balance should be stored for that account")
		assert.Equal(t, map[*currency.Item]time.Time{currency.USD.Item: time.UnixMilli(1795087199000)}, ex.balanceEventTimes(creds), "another account's event should leave the configured account's event times alone")
		assert.Empty(t, ex.balanceEventTimes(other), "another account's event should record no event times")
	})

	t.Run("a newly configured account's event is stored after the previous account's later snapshot", func(t *testing.T) {
		t.Parallel()
		ex := subscribedUserDataTestExchange(t)
		previous := configuredCredentials(t, ex)
		ex.recordBalanceSnapshot(previous, time.UnixMilli(1795087200000))
		next := &accounts.Credentials{Key: "next-api-key", Secret: "next-api-secret"}
		ex.SetCredentials(next)
		require.NoError(t, ex.processAccountPosition(accounts.DeployCredentialsToContext(t.Context(), next), next, []byte(`{"e":"outboundAccountPosition","E":1795087100003,"u":1795087100000,"B":[{"a":"BTC","f":"2.00000000","l":"0.00000000"}]}`)), "processAccountPosition must not error for the newly configured account's event")
		_, ok := relayed(t, ex).(accounts.SubAccounts)
		require.True(t, ok, "processAccountPosition must relay the newly configured account's balances")
		stored, err := ex.Accounts.GetBalance("", next, asset.Spot, currency.BTC)
		require.NoError(t, err, "GetBalance must not error for the newly configured account")
		assert.Equal(t, 2.0, stored.Total, "the newly configured account's event should be stored though the previous account's snapshot is later")
		assert.Equal(t, map[*currency.Item]time.Time{currency.BTC.Item: time.UnixMilli(1795087100000)}, ex.balanceEventTimes(next), "the newly configured account's event should be recorded as its own")
		assert.Empty(t, ex.balanceEventTimes(previous), "the newly configured account's event should leave the previous account's event times alone")
	})

	for _, tc := range []struct {
		name string
		// configured is whether the event is the configured account's as it arrives; otherwise another account's event
		// arrives, and that account is configured and stores its snapshot while the event waits
		configured bool
		// updated is the event's account update time, against its account's snapshot of 3 BTC at 1795087300000
		updated int64
		// stored is the event account's BTC balance afterwards, and relayed whether the event is relayed
		stored  float64
		relayed bool
	}{
		{name: "the configured account's event older than its snapshot is left out", configured: true, updated: 1795087200000, stored: 3},
		{name: "the configured account's event newer than its snapshot is stored as its own", configured: true, updated: 1795087400000, stored: 2, relayed: true},
		{name: "another account's event older than that account's snapshot is left out once that account is configured", updated: 1795087200000, stored: 3},
		{name: "another account's event newer than that account's snapshot is stored as its own once that account is configured", updated: 1795087400000, stored: 2, relayed: true},
	} {
		t.Run(tc.name+", the configured account changing while it waits to be stored", func(t *testing.T) {
			t.Parallel()
			ex := subscribedUserDataTestExchange(t)
			account, next := configuredCredentials(t, ex), &accounts.Credentials{Key: "next-api-key", Secret: "next-api-secret"}
			if !tc.configured {
				account = next
			}
			ctx := accounts.DeployCredentialsToContext(t.Context(), account)
			snapshot := func() {
				subAccount := accounts.NewSubAccount(asset.Spot, "")
				subAccount.Balances.Set(currency.BTC, accounts.Balance{Total: 3, Free: 3})
				require.NoError(t, ex.Accounts.Save(ctx, accounts.SubAccounts{subAccount}, true), "Save must not error")
				ex.recordBalanceSnapshot(account, time.UnixMilli(1795087300000))
			}
			if tc.configured {
				snapshot()
			}
			held, release := make(chan struct{}), make(chan struct{})
			var releaseOnce sync.Once
			t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
			holder, processed := make(chan error, 1), make(chan error, 1)
			go func() {
				holder <- ex.withBalanceStore(func() error {
					close(held)
					<-release
					return nil
				})
			}()
			select {
			case <-held:
			case <-time.After(5 * time.Second):
				require.FailNow(t, "the test must hold the balance store lock")
			}
			go func() {
				processed <- ex.processAccountPosition(ctx, account, []byte(`{"e":"outboundAccountPosition","E":`+strconv.FormatInt(tc.updated+3, 10)+`,"u":`+strconv.FormatInt(tc.updated, 10)+`,"B":[{"a":"BTC","f":"2.00000000","l":"0.00000000"}]}`))
			}()
			waitForBalanceStoreWaiter(t)
			ex.SetCredentials(next)
			if !tc.configured {
				snapshot()
			}
			releaseOnce.Do(func() { close(release) })
			for _, done := range []chan error{holder, processed} {
				select {
				case err := <-done:
					require.NoError(t, err, "the balance store's holders must not error")
				case <-time.After(5 * time.Second):
					require.FailNow(t, "the balance store's holders must finish")
				}
			}
			stored, err := ex.Accounts.GetBalance("", account, asset.Spot, currency.BTC)
			require.NoError(t, err, "GetBalance must not error")
			assert.Equal(t, tc.stored, stored.Total, "the event's account should hold the latest of its event and its snapshot")
			if !tc.relayed {
				assert.Empty(t, ex.Websocket.DataHandler.C, "an event left out should relay nothing")
				assert.Empty(t, ex.balanceEventTimes(account), "an event left out should record no event times")
				return
			}
			_, ok := relayed(t, ex).(accounts.SubAccounts)
			assert.True(t, ok, "the event should be relayed as sub-accounts")
			assert.Equal(t, map[*currency.Item]time.Time{currency.BTC.Item: time.UnixMilli(tc.updated)}, ex.balanceEventTimes(account), "the event should be recorded for its own account")
			if tc.configured {
				assert.Empty(t, ex.balanceEventTimes(next), "the event should record nothing for the newly configured account")
			}
		})
	}

	t.Run("an event of three currencies with amounts to the eighth decimal place is relayed and stored exactly, held rather than borrowed", func(t *testing.T) {
		t.Parallel()
		ex := subscribedUserDataTestExchange(t)
		creds := configuredCredentials(t, ex)
		ctx := accounts.DeployCredentialsToContext(t.Context(), creds)
		require.NoError(t, ex.processAccountPosition(ctx, creds, []byte(`{"e":"outboundAccountPosition","E":1795087200003,"u":1795087200000,"B":[{"a":"BTC","f":"1.23456789","l":"0.87654321"},{"a":"USD","f":"623.12345678","l":"15.00000001"},{"a":"BNB","f":"0.12345678","l":"0.00000001"}]}`)), "processAccountPosition must not error")
		subAccounts, ok := relayed(t, ex).(accounts.SubAccounts)
		require.True(t, ok, "processAccountPosition must relay the balances as sub-accounts")
		require.Len(t, subAccounts, 1, "processAccountPosition must relay one sub-account")
		require.Len(t, subAccounts[0].Balances, 3, "processAccountPosition must relay each currency of the event")
		for _, exp := range []accounts.Balance{
			{Currency: currency.BTC, Total: 2.1111111, Hold: 0.87654321, Free: 1.23456789},
			{Currency: currency.USD, Total: 638.12345679, Hold: 15.00000001, Free: 623.12345678},
			{Currency: currency.BNB, Total: 0.12345679, Hold: 0.00000001, Free: 0.12345678},
		} {
			stored, err := ex.Accounts.GetBalance("", creds, asset.Spot, exp.Currency)
			require.NoErrorf(t, err, "GetBalance must not error for %s", exp.Currency)
			exp.UpdatedAt = stored.UpdatedAt
			assert.Equalf(t, exp, stored, "the %s balance should be stored as the event's amounts", exp.Currency)
			assert.Equalf(t, exp, subAccounts[0].Balances[exp.Currency], "the %s balance should be relayed as the event's amounts", exp.Currency)
		}
		require.NoError(t, ex.processAccountPosition(ctx, creds, []byte(`{"e":"outboundAccountPosition","E":1795087201003,"u":1795087201000,"B":[{"a":"BTC","f":"1.23456789","l":"0.87654321"},{"a":"USD","f":"623.12345678","l":"15.00000001"},{"a":"BNB","f":"0.12345678","l":"0.00000001"}]}`)), "processAccountPosition must not error for the same balances again")
		subAccounts, ok = relayed(t, ex).(accounts.SubAccounts)
		require.True(t, ok, "processAccountPosition must relay the same balances again as sub-accounts")
		require.Len(t, subAccounts, 1, "processAccountPosition must relay one sub-account for the same balances again")
		assert.Len(t, subAccounts[0].Balances, 3, "an event repeating unchanged balances should relay each of its currencies")
	})

	t.Run("each currency of an event is relayed as spot, reserving and releasing a balance moves it between free and held at the same total, and a balance reaching zero is relayed", func(t *testing.T) {
		t.Parallel()
		ex := subscribedUserDataTestExchange(t)
		creds := configuredCredentials(t, ex)
		ctx := accounts.DeployCredentialsToContext(t.Context(), creds)
		relayedBalances := func(event, which string) accounts.CurrencyBalances {
			t.Helper()
			require.NoErrorf(t, ex.processAccountPosition(ctx, creds, []byte(event)), "processAccountPosition must not error for the %s", which)
			subAccounts, ok := relayed(t, ex).(accounts.SubAccounts)
			require.Truef(t, ok, "processAccountPosition must relay the %s as sub-accounts", which)
			require.Lenf(t, subAccounts, 1, "processAccountPosition must relay one sub-account for the %s", which)
			assert.Equalf(t, asset.Spot, subAccounts[0].AssetType, "the %s should be relayed as spot balances", which)
			assert.Emptyf(t, subAccounts[0].ID, "the %s should be relayed for the main account", which)
			return subAccounts[0].Balances
		}
		amounts := func(b accounts.Balance) [3]float64 {
			return [3]float64{b.Total, b.Free, b.Hold}
		}
		storedBalance := func(code currency.Code) accounts.Balance {
			t.Helper()
			b, err := ex.Accounts.GetBalance("", creds, asset.Spot, code)
			require.NoErrorf(t, err, "GetBalance must not error for %s", code)
			return b
		}
		stored := func(code currency.Code) [3]float64 {
			t.Helper()
			return amounts(storedBalance(code))
		}
		balances := relayedBalances(`{"e":"outboundAccountPosition","E":1795087200003,"u":1795087200000,"B":[{"a":"BTC","f":"1.25000000","l":"0.00000000"},{"a":"USD","f":"623.00000000","l":"15.00000000"}]}`, "event of two currencies")
		require.Len(t, balances, 2, "the event of two currencies must be relayed with both")
		assert.Equal(t, [3]float64{1.25, 1.25, 0}, amounts(balances[currency.BTC]), "the BTC balance should be relayed as the event's")
		assert.Equal(t, [3]float64{638, 623, 15}, amounts(balances[currency.USD]), "the USD balance should be relayed as the event's")
		assert.Equal(t, storedBalance(currency.BTC), balances[currency.BTC], "the BTC balance should be relayed as stored")
		assert.Equal(t, storedBalance(currency.USD), balances[currency.USD], "the USD balance should be relayed as stored")
		eventTimes := func(btc int64) map[*currency.Item]time.Time {
			return map[*currency.Item]time.Time{currency.BTC.Item: time.UnixMilli(btc), currency.USD.Item: time.UnixMilli(1795087200000)}
		}
		assert.Equal(t, eventTimes(1795087200000), ex.balanceEventTimes(creds), "the event of two currencies should record its account update time for both")
		first := maps.Clone(balances)
		firstRelay := balances
		balances = relayedBalances(`{"e":"outboundAccountPosition","E":1795087201003,"u":1795087201000,"B":[{"a":"BTC","f":"0.00000000","l":"1.25000000"}]}`, "reservation")
		require.Len(t, balances, 1, "the reservation must be relayed with its own currency alone")
		assert.Equal(t, [3]float64{1.25, 0, 1.25}, amounts(balances[currency.BTC]), "the reservation should be relayed with the BTC balance locked")
		assert.Equal(t, [3]float64{1.25, 0, 1.25}, stored(currency.BTC), "the reservation should be stored with the BTC balance locked")
		assert.Equal(t, [3]float64{638, 623, 15}, stored(currency.USD), "the reservation should leave the USD balance stored as it was")
		assert.Equal(t, eventTimes(1795087201000), ex.balanceEventTimes(creds), "the reservation should record its account update time for BTC alone")
		balances = relayedBalances(`{"e":"outboundAccountPosition","E":1795087202003,"u":1795087202000,"B":[{"a":"BTC","f":"1.25000000","l":"0.00000000"}]}`, "release")
		assert.Equal(t, [3]float64{1.25, 1.25, 0}, amounts(balances[currency.BTC]), "the release should be relayed with the BTC balance free")
		assert.Equal(t, [3]float64{1.25, 1.25, 0}, stored(currency.BTC), "the release should be stored with the BTC balance free")
		assert.Equal(t, storedBalance(currency.BTC), balances[currency.BTC], "the release should be relayed as stored")
		assert.Equal(t, eventTimes(1795087202000), ex.balanceEventTimes(creds), "the release should record its account update time for BTC alone")
		balances = relayedBalances(`{"e":"outboundAccountPosition","E":1795087203003,"u":1795087203000,"B":[{"a":"BTC","f":"0.00000000","l":"0.00000000"}]}`, "balance reaching zero")
		require.Len(t, balances, 1, "the balance reaching zero must be relayed")
		assert.Equal(t, [3]float64{0, 0, 0}, amounts(balances[currency.BTC]), "the BTC balance reaching zero should be relayed as zero")
		assert.Equal(t, [3]float64{0, 0, 0}, stored(currency.BTC), "the BTC balance reaching zero should be stored as zero")
		assert.Equal(t, storedBalance(currency.BTC), balances[currency.BTC], "the balance reaching zero should be relayed as stored")
		assert.Equal(t, eventTimes(1795087203000), ex.balanceEventTimes(creds), "the balance reaching zero should record its account update time for BTC")
		assert.Equal(t, first, firstRelay, "the event of two currencies should be left as it was relayed after the later events")
	})
}

func TestProcessExecutionReport(t *testing.T) {
	t.Parallel()
	ex := subscribedUserDataTestExchange(t)
	ctx := t.Context()
	newOrder := `{"e":"executionReport","E":1499405658658,"s":"BTCUSDT","c":"mUvoqJxFIILMdfAW5iGSOW","S":"BUY","o":"LIMIT","f":"GTC","q":"1.00000000","p":"0.10264410","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"NEW","X":"NEW","r":"NONE","i":4293153,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1499405658657,"t":-1,"I":8641984,"w":true,"m":false,"M":false,"O":1499405658657,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","V":"EXPIRE_MAKER","W":1499405658657}`
	require.NoError(t, ex.processExecutionReport(ctx, []byte(newOrder)), "processExecutionReport must not error")
	exp := &order.Detail{
		Price:           0.1026441,
		Amount:          1,
		CostAsset:       currency.USDT,
		Exchange:        ex.Name,
		OrderID:         "4293153",
		ClientOrderID:   "mUvoqJxFIILMdfAW5iGSOW",
		RemainingAmount: 1,
		Type:            order.Limit,
		Side:            order.Buy,
		Status:          order.New,
		AssetType:       asset.Spot,
		Date:            time.UnixMilli(1499405658657),
		LastUpdated:     time.UnixMilli(1499405658657),
		Pair:            currency.NewBTCUSDT(),
		TimeInForce:     order.GoodTillCancel,
	}
	assert.Equal(t, exp, relayed(t, ex), "a new order should be relayed as its order detail")

	trade := `{"e":"executionReport","E":1499405658658,"s":"BTCUSDT","c":"mUvoqJxFIILMdfAW5iGSOW","S":"SELL","o":"LIMIT_MAKER","f":"GTC","q":"1.00000000","p":"2.00000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"PARTIALLY_FILLED","r":"NONE","i":4293153,"l":"0.25000000","z":"0.25000000","L":"2.00000000","n":"0.001","N":"BNB","T":1499405658700,"t":77,"I":8641985,"w":true,"m":true,"M":false,"O":1499405658657,"Z":"0.50000000","Y":"0.50000000","Q":"0.00000000","V":"EXPIRE_MAKER","W":1499405658657}`
	require.NoError(t, ex.processExecutionReport(ctx, []byte(trade)), "processExecutionReport must not error for a trade")
	detail, ok := relayed(t, ex).(*order.Detail)
	require.True(t, ok, "a trade must be relayed as an order detail")
	assert.Equal(t, order.PostOnly, detail.TimeInForce, "a LIMIT_MAKER order should read as post only")
	assert.Equal(t, 0.75, detail.RemainingAmount, "the remaining amount should be the unfilled quantity")
	assert.Equal(t, 2.0, detail.AverageExecutedPrice, "the average price should be the quote amount over the quantity filled")
	assert.Equal(t, []order.TradeHistory{{
		Price:     2,
		Amount:    0.25,
		Fee:       0.001,
		FeeAsset:  "BNB",
		Exchange:  ex.Name,
		TID:       "77",
		Type:      order.LimitMaker,
		Side:      order.Sell,
		Timestamp: time.UnixMilli(1499405658700),
		IsMaker:   true,
		Total:     0.5,
	}}, detail.Trades, "a trade should carry its fill with the trade ID")

	cancelled := `{"e":"executionReport","E":1499405658800,"s":"BTCUSDT","c":"cancelRequestID","S":"BUY","o":"LIMIT","f":"GTC","q":"1","p":"1","x":"CANCELED","X":"CANCELED","i":5,"C":"originalClientID","T":1499405658800,"O":1499405658657}`
	require.NoError(t, ex.processExecutionReport(ctx, []byte(cancelled)), "processExecutionReport must not error for a cancellation")
	detail, ok = relayed(t, ex).(*order.Detail)
	require.True(t, ok, "a cancellation must be relayed as an order detail")
	assert.Equal(t, "originalClientID", detail.ClientOrderID, "a cancellation should report the order's own client order ID")
	assert.Empty(t, ex.Websocket.DataHandler.C, "a cancellation should be relayed once")

	for orderType, exp := range map[OrderType]order.Type{OrderTypeStopLossLimit: order.StopLimit, OrderTypeTakeProfitLimit: order.TakeProfitLimit} {
		t.Run(string(orderType), func(t *testing.T) {
			t.Parallel()
			subscribed := subscribedUserDataTestExchange(t)
			frame := `{"e":"executionReport","s":"BTCUSDT","S":"SELL","o":"` + string(orderType) + `","f":"GTC","q":"1","p":"70000","P":"69900","X":"NEW","i":7,"T":1499405658800,"O":1499405658800}`
			require.NoErrorf(t, subscribed.processExecutionReport(t.Context(), []byte(frame)), "processExecutionReport must not error for a %s order", orderType)
			detail, ok := relayed(t, subscribed).(*order.Detail)
			require.Truef(t, ok, "a %s order must be relayed as an order detail", orderType)
			assert.Equalf(t, exp, detail.Type, "a %s order should be relayed with its type", orderType)
			assert.Equalf(t, 69900.0, detail.TriggerPrice, "a %s order should be relayed with its stop price", orderType)
			assert.Emptyf(t, subscribed.Websocket.DataHandler.C, "a %s order should be relayed once", orderType)
		})
	}

	for name, frame := range map[string]string{
		"status":        `{"e":"executionReport","s":"BTCUSDT","S":"BUY","o":"LIMIT","f":"GTC","X":"UNKNOWN","i":6}`,
		"order type":    `{"e":"executionReport","s":"BTCUSDT","S":"BUY","o":"UNKNOWN","f":"GTC","X":"NEW","i":6}`,
		"side":          `{"e":"executionReport","s":"BTCUSDT","S":"UNKNOWN","o":"LIMIT","f":"GTC","X":"NEW","i":6}`,
		"time in force": `{"e":"executionReport","s":"BTCUSDT","S":"BUY","o":"LIMIT","f":"UNKNOWN","X":"NEW","i":6}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			subscribed := subscribedUserDataTestExchange(t)
			require.NoErrorf(t, subscribed.processExecutionReport(t.Context(), []byte(frame)), "processExecutionReport must not error for an unknown %s", name)
			classification, ok := relayed(t, subscribed).(order.ClassificationError)
			require.Truef(t, ok, "an unknown %s must be relayed as a classification error", name)
			assert.Equalf(t, "6", classification.OrderID, "the classification error of an unknown %s should name the order", name)
			assert.Emptyf(t, subscribed.Websocket.DataHandler.C, "an unknown %s should be relayed once", name)
		})
	}

	t.Run("each report of an order of an unknown type is relayed as a classification error", func(t *testing.T) {
		t.Parallel()
		subscribed := subscribedUserDataTestExchange(t)
		report := func(status string) {
			t.Helper()
			frame := `{"e":"executionReport","s":"BTCUSDT","S":"BUY","o":"UNKNOWN","f":"GTC","X":"` + status + `","i":6}`
			require.NoErrorf(t, subscribed.processExecutionReport(t.Context(), []byte(frame)), "processExecutionReport must not error for the %s report of an order of an unknown type", status)
			classification, ok := relayed(t, subscribed).(order.ClassificationError)
			require.Truef(t, ok, "the %s report of an order of an unknown type must be relayed as a classification error", status)
			assert.Equalf(t, "6", classification.OrderID, "the classification error of the %s report should name the order", status)
		}
		report("NEW")
		report("CANCELED")
		assert.Empty(t, subscribed.Websocket.DataHandler.C, "each report should be relayed once")
	})

	t.Run("a report carrying the trade group ID of an account in no trade group is relayed", func(t *testing.T) {
		t.Parallel()
		ungrouped := subscribedUserDataTestExchange(t)
		report := []byte(`{"e":"executionReport","E":1794466396201,"s":"BTCUSDT","c":"restingBuy","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00100000","p":"65900.02000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE_PREVENTION","X":"EXPIRED_IN_MATCH","r":"NONE","i":4293182,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466396200,"t":-1,"I":8642015,"w":false,"m":false,"M":false,"O":1794466396120,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","W":1794466396120,"V":"EXPIRE_MAKER","u":-1,"v":17,"U":4293183,"A":"0.00100000","B":"0.00100000","pl":"0.00100000","pL":"65900.02000000","pY":"65.90002000","eR":"EXPIRED_IN_MATCH"}`)
		var decoded WebsocketExecutionReport
		require.NoError(t, json.Unmarshal(report, &decoded), "the report of an account in no trade group must decode")
		assert.Equal(t, int64(-1), decoded.TradeGroupID, "the report should decode the trade group ID of an account in no trade group")
		assert.Equal(t, uint64(4293183), decoded.CounterOrderID, "the report should decode the counter order ID beside the trade group ID")
		require.NoError(t, json.Unmarshal(bytes.Replace(report, []byte(`"u":-1`), []byte(`"u":2`), 1), &decoded), "the report of an account in trade group 2 must decode")
		assert.Equal(t, int64(2), decoded.TradeGroupID, "the report should decode the trade group ID of an account in a trade group")
		require.NoError(t, ungrouped.processExecutionReport(t.Context(), report), "processExecutionReport must not error for a report of an account in no trade group")
		detail, ok := relayed(t, ungrouped).(*order.Detail)
		require.True(t, ok, "processExecutionReport must relay the report of an account in no trade group as an order detail")
		assert.Equal(t, order.Expired, detail.Status, "the report of an account in no trade group should be relayed as the order's expiry")
	})

	assert.ErrorIs(t, ex.processExecutionReport(ctx, []byte(`{"s":"NOTAPAIR"}`)), currency.ErrPairNotFound, "processExecutionReport should error for an unknown symbol")
	assert.Error(t, ex.processExecutionReport(ctx, []byte(`{"s":1}`)), "processExecutionReport should error for an event that does not decode")

	t.Run("a later fill reports the order's totals and the fill's own amounts", func(t *testing.T) {
		t.Parallel()
		ex := subscribedUserDataTestExchange(t)
		frame := `{"e":"executionReport","E":1499405658801,"s":"BTCUSDT","c":"mUvoqJxFIILMdfAW5iGSOW","S":"SELL","o":"LIMIT_MAKER","f":"GTC","q":"1.10000000","p":"2.00000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"PARTIALLY_FILLED","r":"NONE","i":4293153,"l":"0.25000000","z":"0.50000000","L":"2.10000000","n":"0.002","N":"BNB","T":1499405658800,"t":78,"I":8641986,"w":true,"m":true,"M":false,"O":1499405658657,"Z":"1.02500000","Y":"0.52500000","Q":"0.00000000","V":"EXPIRE_MAKER","W":1499405658657}`
		require.NoError(t, ex.processExecutionReport(t.Context(), []byte(frame)), "processExecutionReport must not error for a later fill")
		assert.Equal(t, &order.Detail{
			Price: 2, Amount: 1.1, AverageExecutedPrice: 2.05, ExecutedAmount: 0.5, RemainingAmount: 0.6, Cost: 1.025,
			CostAsset: currency.USDT, FeeAsset: currency.BNB, Exchange: ex.Name, OrderID: "4293153", ClientOrderID: "mUvoqJxFIILMdfAW5iGSOW",
			Type: order.LimitMaker, Side: order.Sell, Status: order.PartiallyFilled, AssetType: asset.Spot,
			Date: time.UnixMilli(1499405658657), LastUpdated: time.UnixMilli(1499405658800), Pair: currency.NewBTCUSDT(), TimeInForce: order.PostOnly,
			Trades: []order.TradeHistory{{
				Price: 2.1, Amount: 0.25, Fee: 0.002, FeeAsset: "BNB", Exchange: ex.Name, TID: "78", Type: order.LimitMaker,
				Side: order.Sell, Timestamp: time.UnixMilli(1499405658800), IsMaker: true, Total: 0.525,
			}},
		}, relayed(t, ex), "a later fill should report the order's cumulative amounts and the fill's own")
	})

	t.Run("a maker fill completing its order is reported as made", func(t *testing.T) {
		t.Parallel()
		ex := subscribedUserDataTestExchange(t)
		frame := `{"e":"executionReport","E":1499405658901,"s":"BTCUSDT","c":"completedMakerOrder","S":"BUY","o":"LIMIT","f":"GTC","q":"0.50000000","p":"2.00000000","x":"TRADE","X":"FILLED","i":4293155,"l":"0.50000000","z":"0.50000000","L":"2.00000000","n":"0.001","N":"BNB","T":1499405658900,"t":81,"w":false,"m":true,"O":1499405658657,"Z":"1.00000000","Y":"1.00000000"}`
		require.NoError(t, ex.processExecutionReport(t.Context(), []byte(frame)), "processExecutionReport must not error for a filled order")
		detail, ok := relayed(t, ex).(*order.Detail)
		require.True(t, ok, "a filled order must be relayed as an order detail")
		require.Len(t, detail.Trades, 1, "a filled order must be relayed with its fill")
		assert.True(t, detail.Trades[0].IsMaker, "a fill the order made should be reported as the maker's, whether or not the order stays on the book")
	})

	t.Run("a quote sized market order reports its quote amount", func(t *testing.T) {
		t.Parallel()
		ex := subscribedUserDataTestExchange(t)
		frame := `{"e":"executionReport","E":1499405658801,"s":"BTCUSDT","c":"quoteSizedOrder","S":"BUY","o":"MARKET","f":"GTC","q":"0.00150000","p":"0.00000000","P":"0.00000000","x":"TRADE","X":"FILLED","i":4293154,"l":"0.00150000","z":"0.00150000","L":"66600.00000000","n":"0","N":"BTC","T":1499405658800,"t":79,"O":1499405658800,"Z":"99.90000000","Y":"99.90000000","Q":"100.00000000","m":false,"w":false}`
		require.NoError(t, ex.processExecutionReport(t.Context(), []byte(frame)), "processExecutionReport must not error for a quote sized market order")
		detail, ok := relayed(t, ex).(*order.Detail)
		require.True(t, ok, "a quote sized market order must be relayed as an order detail")
		assert.Equal(t, 100.0, detail.QuoteAmount, "a quote sized market order should report the quote amount it was placed for")
		require.Len(t, detail.Trades, 1, "a filled market order must relay its fill")
		assert.False(t, detail.Trades[0].IsMaker, "a market order's fill should be reported as the taker's")
		assert.Equal(t, time.UnixMilli(1499405658800), detail.Date, "a quote sized market order should report when it was created")
	})

	t.Run("each report of an order STP prevented part of leaves the prevented quantity out of the remaining amount only while the order is open or once it is filled", func(t *testing.T) {
		t.Parallel()
		// Binance.US documents neither DECREMENT's nor TRANSFER's stream reports; these give an order STP prevented part of the
		// prevented quantity fields its STP FAQ gives "execution reports for orders involved in STP"
		for _, tc := range []struct {
			name, frame string
			expected    *order.Detail
		}{
			{
				name:     "a market buy filled once a DECREMENT match prevented part of it, paying its commission in BNB",
				frame:    `{"e":"executionReport","E":1794466398101,"s":"BTCUSDT","c":"marketBuy","S":"BUY","o":"MARKET","f":"GTC","q":"0.00600000","p":"0.00000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"FILLED","r":"NONE","i":4293200,"l":"0.00400000","z":"0.00400000","L":"66000.01000000","n":"0.00033000","N":"BNB","T":1794466398100,"t":120,"I":8642050,"w":false,"m":false,"M":true,"O":1794466398099,"Z":"264.00004000","Y":"264.00004000","Q":"0.00000000","V":"DECREMENT","A":"0.00200000"}`,
				expected: &order.Detail{Amount: 0.006, AverageExecutedPrice: 66000.01, ExecutedAmount: 0.004, Cost: 264.00004, CostAsset: currency.USDT, FeeAsset: currency.BNB, OrderID: "4293200", ClientOrderID: "marketBuy", Type: order.Market, Side: order.Buy, Status: order.Filled, Date: time.UnixMilli(1794466398099), LastUpdated: time.UnixMilli(1794466398100), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel, Trades: []order.TradeHistory{{Price: 66000.01, Amount: 0.004, Fee: 0.00033, FeeAsset: "BNB", TID: "120", Type: order.Market, Side: order.Buy, Timestamp: time.UnixMilli(1794466398100), Total: 264.00004}}},
			},
			{
				name:     "a limit sell left open by a DECREMENT match before it rests",
				frame:    `{"e":"executionReport","E":1794466398201,"s":"BTCUSDT","c":"restingSell","S":"SELL","o":"LIMIT","f":"GTC","q":"0.00600000","p":"66000.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE_PREVENTION","X":"NEW","r":"NONE","i":4293201,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466398200,"t":-1,"I":8642051,"w":false,"m":false,"M":false,"O":1794466398200,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","V":"DECREMENT","v":22,"U":4293202,"A":"0.00200000","B":"0.00200000","pl":"0.00200000","pL":"66000.01000000","pY":"132.00002000"}`,
				expected: &order.Detail{Price: 66000.01, Amount: 0.006, RemainingAmount: 0.004, CostAsset: currency.USDT, OrderID: "4293201", ClientOrderID: "restingSell", Type: order.Limit, Side: order.Sell, Status: order.New, Date: time.UnixMilli(1794466398200), LastUpdated: time.UnixMilli(1794466398200), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel},
			},
			{
				name:     "a limit buy expired by a later match once a DECREMENT match had prevented part of it",
				frame:    `{"e":"executionReport","E":1794466398301,"s":"BTCUSDT","c":"expiredBuy","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00600000","p":"66000.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE_PREVENTION","X":"EXPIRED_IN_MATCH","r":"NONE","i":4293203,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466398300,"t":-1,"I":8642052,"w":false,"m":false,"M":false,"O":1794466398250,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","W":1794466398250,"V":"DECREMENT","v":24,"U":4293204,"A":"0.00600000","B":"0.00400000","pl":"0.00400000","pL":"66000.01000000","pY":"264.00004000","eR":"EXPIRED_IN_MATCH"}`,
				expected: &order.Detail{Price: 66000.01, Amount: 0.006, RemainingAmount: 0.006, CostAsset: currency.USDT, OrderID: "4293203", ClientOrderID: "expiredBuy", Type: order.Limit, Side: order.Buy, Status: order.Expired, Date: time.UnixMilli(1794466398250), LastUpdated: time.UnixMilli(1794466398300), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel},
			},
			{
				name:     "a limit buy cancelled after a fill once a DECREMENT match had prevented part of it",
				frame:    `{"e":"executionReport","E":1794466398401,"s":"BTCUSDT","c":"cancelPartlyFilledBuy","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00600000","p":"66000.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"partlyFilledDecrementedBuy","x":"CANCELED","X":"CANCELED","r":"NONE","i":4293205,"l":"0.00000000","z":"0.00100000","L":"0.00000000","n":"0","N":null,"T":1794466398400,"t":-1,"I":8642053,"w":false,"m":false,"M":false,"O":1794466398350,"Z":"66.00001000","Y":"0.00000000","Q":"0.00000000","W":1794466398350,"V":"DECREMENT","A":"0.00200000"}`,
				expected: &order.Detail{Price: 66000.01, Amount: 0.006, AverageExecutedPrice: 66000.01, ExecutedAmount: 0.001, RemainingAmount: 0.005, Cost: 66.00001, CostAsset: currency.USDT, OrderID: "4293205", ClientOrderID: "partlyFilledDecrementedBuy", Type: order.Limit, Side: order.Buy, Status: order.Cancelled, Date: time.UnixMilli(1794466398350), LastUpdated: time.UnixMilli(1794466398400), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel},
			},
			{
				name:     "a limit buy partly filled when a DECREMENT match prevents part of it",
				frame:    `{"e":"executionReport","E":1794466398501,"s":"BTCUSDT","c":"partlyPreventedBuy","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00600000","p":"66000.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE_PREVENTION","X":"PARTIALLY_FILLED","r":"NONE","i":4293206,"l":"0.00000000","z":"0.00100000","L":"0.00000000","n":"0","N":null,"T":1794466398500,"t":-1,"I":8642054,"w":true,"m":false,"M":false,"O":1794466398450,"Z":"66.00001000","Y":"0.00000000","Q":"0.00000000","W":1794466398450,"V":"DECREMENT","v":25,"U":4293207,"A":"0.00200000","B":"0.00200000","pl":"0.00200000","pL":"66000.01000000","pY":"132.00002000"}`,
				expected: &order.Detail{Price: 66000.01, Amount: 0.006, AverageExecutedPrice: 66000.01, ExecutedAmount: 0.001, RemainingAmount: 0.003, Cost: 66.00001, CostAsset: currency.USDT, OrderID: "4293206", ClientOrderID: "partlyPreventedBuy", Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled, Date: time.UnixMilli(1794466398450), LastUpdated: time.UnixMilli(1794466398500), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel},
			},
			{
				name:     "an IOC limit buy expired after a fill once a DECREMENT match had prevented part of it",
				frame:    `{"e":"executionReport","E":1794466398601,"s":"BTCUSDT","c":"iocDecrementedBuy","S":"BUY","o":"LIMIT","f":"IOC","q":"0.00600000","p":"66000.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"EXPIRED","X":"EXPIRED","r":"NONE","i":4293208,"l":"0.00000000","z":"0.00100000","L":"0.00000000","n":"0","N":null,"T":1794466398600,"t":-1,"I":8642055,"w":false,"m":false,"M":false,"O":1794466398600,"Z":"66.00001000","Y":"0.00000000","Q":"0.00000000","V":"DECREMENT","A":"0.00200000","eR":"UNFILLED_IOC_QUANTITY_EXPIRED"}`,
				expected: &order.Detail{Price: 66000.01, Amount: 0.006, AverageExecutedPrice: 66000.01, ExecutedAmount: 0.001, RemainingAmount: 0.005, Cost: 66.00001, CostAsset: currency.USDT, OrderID: "4293208", ClientOrderID: "iocDecrementedBuy", Type: order.Limit, Side: order.Buy, Status: order.Expired, Date: time.UnixMilli(1794466398600), LastUpdated: time.UnixMilli(1794466398600), Pair: currency.NewBTCUSDT(), TimeInForce: order.ImmediateOrCancel},
			},
			{
				name:     "a TRANSFER limit buy filled as the maker, paying its commission in BNB",
				frame:    `{"e":"executionReport","E":1794466398701,"s":"BTCUSDT","c":"transferBuy","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00600000","p":"66000.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"FILLED","r":"NONE","i":4293209,"l":"0.00400000","z":"0.00400000","L":"66000.01000000","n":"0.00033000","N":"BNB","T":1794466398700,"t":121,"I":8642056,"w":false,"m":true,"M":true,"O":1794466398650,"Z":"264.00004000","Y":"264.00004000","Q":"0.00000000","W":1794466398650,"V":"TRANSFER","A":"0.00200000"}`,
				expected: &order.Detail{Price: 66000.01, Amount: 0.006, AverageExecutedPrice: 66000.01, ExecutedAmount: 0.004, Cost: 264.00004, CostAsset: currency.USDT, FeeAsset: currency.BNB, OrderID: "4293209", ClientOrderID: "transferBuy", Type: order.Limit, Side: order.Buy, Status: order.Filled, Date: time.UnixMilli(1794466398650), LastUpdated: time.UnixMilli(1794466398700), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel, Trades: []order.TradeHistory{{Price: 66000.01, Amount: 0.004, Fee: 0.00033, FeeAsset: "BNB", TID: "121", Type: order.Limit, Side: order.Buy, Timestamp: time.UnixMilli(1794466398700), IsMaker: true, Total: 264.00004}}},
			},
			{
				name:     "a limit maker buy filled as the maker",
				frame:    `{"e":"executionReport","E":1794466398801,"s":"BTCUSDT","c":"makerBuy","S":"BUY","o":"LIMIT_MAKER","f":"GTC","q":"0.00600000","p":"66000.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"FILLED","r":"NONE","i":4293210,"l":"0.00400000","z":"0.00400000","L":"66000.01000000","n":"0.00000400","N":"BTC","T":1794466398800,"t":122,"I":8642057,"w":false,"m":true,"M":true,"O":1794466398750,"Z":"264.00004000","Y":"264.00004000","Q":"0.00000000","W":1794466398750,"V":"DECREMENT","A":"0.00200000"}`,
				expected: &order.Detail{Price: 66000.01, Amount: 0.006, AverageExecutedPrice: 66000.01, ExecutedAmount: 0.004, Cost: 264.00004, CostAsset: currency.USDT, FeeAsset: currency.BTC, OrderID: "4293210", ClientOrderID: "makerBuy", Type: order.LimitMaker, Side: order.Buy, Status: order.Filled, Date: time.UnixMilli(1794466398750), LastUpdated: time.UnixMilli(1794466398800), Pair: currency.NewBTCUSDT(), TimeInForce: order.PostOnly, Trades: []order.TradeHistory{{Price: 66000.01, Amount: 0.004, Fee: 0.000004, FeeAsset: "BTC", TID: "122", Type: order.LimitMaker, Side: order.Buy, Timestamp: time.UnixMilli(1794466398800), IsMaker: true, Total: 264.00004}}},
			},
			{
				name:     "a stop loss sell filled after its trigger",
				frame:    `{"e":"executionReport","E":1794466398901,"s":"BTCUSDT","c":"stopSell","S":"SELL","o":"STOP_LOSS","f":"GTC","q":"0.00600000","p":"0.00000000","P":"66000.05000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"FILLED","r":"NONE","i":4293211,"l":"0.00400000","z":"0.00400000","L":"66000.01000000","n":"0.26400004","N":"USDT","T":1794466398900,"t":123,"I":8642058,"w":false,"m":false,"M":true,"O":1794466398800,"Z":"264.00004000","Y":"264.00004000","Q":"0.00000000","V":"DECREMENT","A":"0.00200000"}`,
				expected: &order.Detail{Amount: 0.006, TriggerPrice: 66000.05, AverageExecutedPrice: 66000.01, ExecutedAmount: 0.004, Cost: 264.00004, CostAsset: currency.USDT, FeeAsset: currency.USDT, OrderID: "4293211", ClientOrderID: "stopSell", Type: order.StopMarket, Side: order.Sell, Status: order.Filled, Date: time.UnixMilli(1794466398800), LastUpdated: time.UnixMilli(1794466398900), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel, Trades: []order.TradeHistory{{Price: 66000.01, Amount: 0.004, Fee: 0.26400004, FeeAsset: "USDT", TID: "123", Type: order.StopMarket, Side: order.Sell, Timestamp: time.UnixMilli(1794466398900), Total: 264.00004}}},
			},
			{
				name:     "a stop loss limit sell filled after its trigger",
				frame:    `{"e":"executionReport","E":1794466399001,"s":"BTCUSDT","c":"stopLimitSell","S":"SELL","o":"STOP_LOSS_LIMIT","f":"GTC","q":"0.00600000","p":"66000.01000000","P":"66000.05000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"FILLED","r":"NONE","i":4293212,"l":"0.00400000","z":"0.00400000","L":"66000.01000000","n":"0.26400004","N":"USDT","T":1794466399000,"t":124,"I":8642059,"w":false,"m":false,"M":true,"O":1794466398900,"Z":"264.00004000","Y":"264.00004000","Q":"0.00000000","W":1794466398960,"V":"DECREMENT","A":"0.00200000"}`,
				expected: &order.Detail{Price: 66000.01, Amount: 0.006, TriggerPrice: 66000.05, AverageExecutedPrice: 66000.01, ExecutedAmount: 0.004, Cost: 264.00004, CostAsset: currency.USDT, FeeAsset: currency.USDT, OrderID: "4293212", ClientOrderID: "stopLimitSell", Type: order.StopLimit, Side: order.Sell, Status: order.Filled, Date: time.UnixMilli(1794466398900), LastUpdated: time.UnixMilli(1794466399000), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel, Trades: []order.TradeHistory{{Price: 66000.01, Amount: 0.004, Fee: 0.26400004, FeeAsset: "USDT", TID: "124", Type: order.StopLimit, Side: order.Sell, Timestamp: time.UnixMilli(1794466399000), Total: 264.00004}}},
			},
			{
				name:     "a take profit buy filled after its trigger",
				frame:    `{"e":"executionReport","E":1794466399101,"s":"BTCUSDT","c":"takeProfitBuy","S":"BUY","o":"TAKE_PROFIT","f":"GTC","q":"0.00600000","p":"0.00000000","P":"65999.99000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"FILLED","r":"NONE","i":4293213,"l":"0.00400000","z":"0.00400000","L":"66000.01000000","n":"0.00000400","N":"BTC","T":1794466399100,"t":125,"I":8642060,"w":false,"m":false,"M":true,"O":1794466399000,"Z":"264.00004000","Y":"264.00004000","Q":"0.00000000","V":"DECREMENT","A":"0.00200000"}`,
				expected: &order.Detail{Amount: 0.006, TriggerPrice: 65999.99, AverageExecutedPrice: 66000.01, ExecutedAmount: 0.004, Cost: 264.00004, CostAsset: currency.USDT, FeeAsset: currency.BTC, OrderID: "4293213", ClientOrderID: "takeProfitBuy", Type: order.TakeProfitMarket, Side: order.Buy, Status: order.Filled, Date: time.UnixMilli(1794466399000), LastUpdated: time.UnixMilli(1794466399100), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel, Trades: []order.TradeHistory{{Price: 66000.01, Amount: 0.004, Fee: 0.000004, FeeAsset: "BTC", TID: "125", Type: order.TakeProfitMarket, Side: order.Buy, Timestamp: time.UnixMilli(1794466399100), Total: 264.00004}}},
			},
			{
				name:     "a take profit limit buy filled after its trigger, displaying an iceberg part",
				frame:    `{"e":"executionReport","E":1794466399201,"s":"BTCUSDT","c":"takeProfitLimitBuy","S":"BUY","o":"TAKE_PROFIT_LIMIT","f":"GTC","q":"0.00600000","p":"66000.01000000","P":"65999.99000000","F":"0.00100000","g":-1,"C":"","x":"TRADE","X":"FILLED","r":"NONE","i":4293214,"l":"0.00400000","z":"0.00400000","L":"66000.01000000","n":"0.00000400","N":"BTC","T":1794466399200,"t":126,"I":8642061,"w":false,"m":false,"M":true,"O":1794466399100,"Z":"264.00004000","Y":"264.00004000","Q":"0.00000000","W":1794466399160,"V":"DECREMENT","A":"0.00200000"}`,
				expected: &order.Detail{Price: 66000.01, Amount: 0.006, TriggerPrice: 65999.99, AverageExecutedPrice: 66000.01, ExecutedAmount: 0.004, Cost: 264.00004, CostAsset: currency.USDT, FeeAsset: currency.BTC, OrderID: "4293214", ClientOrderID: "takeProfitLimitBuy", Type: order.TakeProfitLimit, Side: order.Buy, Status: order.Filled, Date: time.UnixMilli(1794466399100), LastUpdated: time.UnixMilli(1794466399200), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel, Trades: []order.TradeHistory{{Price: 66000.01, Amount: 0.004, Fee: 0.000004, FeeAsset: "BTC", TID: "126", Type: order.TakeProfitLimit, Side: order.Buy, Timestamp: time.UnixMilli(1794466399200), Total: 264.00004}}},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := subscribedUserDataTestExchange(t)
				require.NoError(t, ex.processExecutionReport(t.Context(), []byte(tc.frame)), "processExecutionReport must not error")
				expected := tc.expected
				expected.Exchange, expected.AssetType = ex.Name, asset.Spot
				for i := range expected.Trades {
					expected.Trades[i].Exchange = ex.Name
				}
				assert.Equal(t, expected, relayed(t, ex), "processExecutionReport should relay the order detail the report gives")
			})
		}
	})

	t.Run("an exchange relays each report in turn, over its orders' lifecycles", func(t *testing.T) {
		t.Parallel()
		ex := subscribedUserDataTestExchange(t)
		ethUSDT := currency.NewPair(currency.ETH, currency.USDT)
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.NewBTCUSDT(), ethUSDT, currency.NewBTCUSD()}, false), "StorePairs must not error")
		conn := newFakeConnection()
		var subscriptionID uint64
		event := func(frame string) error {
			return ex.wsHandleAPIData(t.Context(), conn, fmt.Appendf(nil, `{"subscriptionId":%d,"event":%s}`, subscriptionID, frame))
		}
		var names []string
		var relays, expected []any
		relay := func(name string, exp any) {
			t.Helper()
			got := relayed(t, ex)
			require.IsTypef(t, exp, got, "%s must be relayed as its type", name)
			assert.Equalf(t, exp, got, "%s should be relayed as expected", name)
			names, relays, expected = append(names, name), append(relays, got), append(expected, exp)
		}
		report := func(frame string, exp *order.Detail) {
			t.Helper()
			exp.Exchange, exp.AssetType = ex.Name, asset.Spot
			for i := range exp.Trades {
				exp.Trades[i].Exchange = ex.Name
			}
			require.NoErrorf(t, event(frame), "wsHandleAPIData must not error for order %s's %s report", exp.OrderID, exp.Status)
			relay(fmt.Sprintf("order %s's %s report", exp.OrderID, exp.Status), exp)
		}
		listStatus := func(frame string, exp *WebsocketListStatus, list *order.Detail) {
			t.Helper()
			list.Exchange, list.Type, list.AssetType = ex.Name, order.OCO, asset.Spot
			require.NoErrorf(t, event(frame), "wsHandleAPIData must not error for order list %d's %s event", exp.OrderListID, exp.ListOrderStatus)
			relay(fmt.Sprintf("order list %d's %s event", exp.OrderListID, exp.ListOrderStatus), exp)
			relay(fmt.Sprintf("order list %d's %s detail", exp.OrderListID, exp.ListOrderStatus), list)
		}
		ocoOrders := []WebsocketListStatusOrder{{Symbol: "BTCUSDT", OrderID: 4293164, ClientOrderID: "ocoStopLossLeg"}, {Symbol: "BTCUSDT", OrderID: 4293165, ClientOrderID: "ocoLimitMakerLeg"}}
		report(`{"e":"executionReport","E":1794466395100,"s":"BTCUSDT","c":"sweptOrder","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00300000","p":"66000.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"NEW","X":"NEW","r":"NONE","i":4293160,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466395099,"t":-1,"I":8641990,"w":false,"m":false,"M":false,"O":1794466395099,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000"}`, &order.Detail{
			Price: 66000.01, Amount: 0.003, RemainingAmount: 0.003, CostAsset: currency.USDT, OrderID: "4293160", ClientOrderID: "sweptOrder", Type: order.Limit, Side: order.Buy,
			Status: order.New, Date: time.UnixMilli(1794466395099), LastUpdated: time.UnixMilli(1794466395099), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel,
		})
		report(`{"e":"executionReport","E":1794466395100,"s":"BTCUSDT","c":"sweptOrder","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00300000","p":"66000.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"PARTIALLY_FILLED","r":"NONE","i":4293160,"l":"0.00100000","z":"0.00100000","L":"65990.00000000","n":"0.00008249","N":"BNB","T":1794466395099,"t":101,"I":8641991,"w":false,"m":false,"M":true,"O":1794466395099,"Z":"65.99000000","Y":"65.99000000","Q":"0.00000000"}`, &order.Detail{
			Price: 66000.01, Amount: 0.003, AverageExecutedPrice: 65990, ExecutedAmount: 0.001, RemainingAmount: 0.002, Cost: 65.99, CostAsset: currency.USDT, FeeAsset: currency.BNB,
			OrderID: "4293160", ClientOrderID: "sweptOrder", Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled, Date: time.UnixMilli(1794466395099),
			LastUpdated: time.UnixMilli(1794466395099), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel, Trades: []order.TradeHistory{{
				Price: 65990, Amount: 0.001, Fee: 0.00008249, FeeAsset: "BNB", TID: "101", Type: order.Limit, Side: order.Buy, Timestamp: time.UnixMilli(1794466395099), Total: 65.99,
			}},
		})
		report(`{"e":"executionReport","E":1794466395100,"s":"BTCUSDT","c":"sweptOrder","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00300000","p":"66000.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"PARTIALLY_FILLED","r":"NONE","i":4293160,"l":"0.00100000","z":"0.00200000","L":"65990.00000000","n":"0.00008249","N":"BNB","T":1794466395099,"t":102,"I":8641992,"w":true,"m":false,"M":true,"O":1794466395099,"Z":"131.98000000","Y":"65.99000000","Q":"0.00000000","W":1794466395099}`, &order.Detail{
			Price: 66000.01, Amount: 0.003, AverageExecutedPrice: 65990, ExecutedAmount: 0.002, RemainingAmount: 0.001, Cost: 131.98, CostAsset: currency.USDT, FeeAsset: currency.BNB,
			OrderID: "4293160", ClientOrderID: "sweptOrder", Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled, Date: time.UnixMilli(1794466395099),
			LastUpdated: time.UnixMilli(1794466395099), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel, Trades: []order.TradeHistory{{
				Price: 65990, Amount: 0.001, Fee: 0.00008249, FeeAsset: "BNB", TID: "102", Type: order.Limit, Side: order.Buy, Timestamp: time.UnixMilli(1794466395099), Total: 65.99,
			}},
		})
		require.NoError(t, event(`{"e":"balanceUpdate","E":1794466395110,"a":"BNB","d":"-0.04991751","T":1794466395109}`), "wsHandleAPIData must not error for a balance update between the reports")
		relay("a balance update between the reports", &WebsocketBalanceUpdate{
			EventType: "balanceUpdate", EventTime: types.Time(time.UnixMilli(1794466395110)), Asset: currency.BNB, BalanceDelta: -0.04991751, ClearTime: types.Time(time.UnixMilli(1794466395109)),
		})
		report(`{"e":"executionReport","E":1794466395120,"s":"BTCUSDT","c":"stopLossOrder","S":"SELL","o":"STOP_LOSS_LIMIT","f":"GTC","q":"0.00200000","p":"64000.02000000","P":"64100.03000000","F":"0.00000000","g":-1,"C":"","x":"NEW","X":"NEW","r":"NONE","i":4293161,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466395119,"t":-1,"I":8641994,"w":false,"m":false,"M":false,"O":1794466395119,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000"}`, &order.Detail{
			Price: 64000.02, Amount: 0.002, TriggerPrice: 64100.03, RemainingAmount: 0.002, CostAsset: currency.USDT, OrderID: "4293161", ClientOrderID: "stopLossOrder", Type: order.StopLimit,
			Side: order.Sell, Status: order.New, Date: time.UnixMilli(1794466395119), LastUpdated: time.UnixMilli(1794466395119), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel,
		})
		report(`{"e":"executionReport","E":1794466395120,"s":"BTCUSDT","c":"takeProfitOrder","S":"SELL","o":"TAKE_PROFIT_LIMIT","f":"GTC","q":"0.00200000","p":"68000.04000000","P":"67900.05000000","F":"0.00000000","g":-1,"C":"","x":"NEW","X":"NEW","r":"NONE","i":4293162,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466395119,"t":-1,"I":8641996,"w":false,"m":false,"M":false,"O":1794466395119,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000"}`, &order.Detail{
			Price: 68000.04, Amount: 0.002, TriggerPrice: 67900.05, RemainingAmount: 0.002, CostAsset: currency.USDT, OrderID: "4293162", ClientOrderID: "takeProfitOrder", Type: order.TakeProfitLimit,
			Side: order.Sell, Status: order.New, Date: time.UnixMilli(1794466395119), LastUpdated: time.UnixMilli(1794466395119), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel,
		})
		report(`{"e":"executionReport","E":1794466395201,"s":"BTCUSDT","c":"cancelStopLoss","S":"SELL","o":"STOP_LOSS_LIMIT","f":"GTC","q":"0.00200000","p":"64000.02000000","P":"64100.03000000","F":"0.00000000","g":-1,"C":"stopLossOrder","x":"CANCELED","X":"CANCELED","r":"NONE","i":4293161,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466395200,"t":-1,"I":8641995,"w":false,"m":false,"M":false,"O":1794466395119,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000"}`, &order.Detail{
			Price: 64000.02, Amount: 0.002, TriggerPrice: 64100.03, RemainingAmount: 0.002, CostAsset: currency.USDT, OrderID: "4293161", ClientOrderID: "stopLossOrder", Type: order.StopLimit,
			Side: order.Sell, Status: order.Cancelled, Date: time.UnixMilli(1794466395119), LastUpdated: time.UnixMilli(1794466395200), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel,
		})
		report(`{"e":"executionReport","E":1794466395231,"s":"BTCUSDT","c":"sweptOrder","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00300000","p":"66000.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"PARTIALLY_FILLED","r":"NONE","i":4293160,"l":"0.00050000","z":"0.00250000","L":"66000.01000000","n":"0.00000050","N":"BTC","T":1794466395230,"t":103,"I":8641993,"w":true,"m":true,"M":true,"O":1794466395099,"Z":"164.98000500","Y":"33.00000500","Q":"0.00000000","W":1794466395099}`, &order.Detail{
			Price: 66000.01, Amount: 0.003, AverageExecutedPrice: 65992.002, ExecutedAmount: 0.0025, RemainingAmount: 0.0005, Cost: 164.980005, CostAsset: currency.USDT, FeeAsset: currency.BTC,
			OrderID: "4293160", ClientOrderID: "sweptOrder", Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled, Date: time.UnixMilli(1794466395099),
			LastUpdated: time.UnixMilli(1794466395230), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel, Trades: []order.TradeHistory{{
				Price: 66000.01, Amount: 0.0005, Fee: 0.0000005, FeeAsset: "BTC", TID: "103", Type: order.Limit, Side: order.Buy, Timestamp: time.UnixMilli(1794466395230), IsMaker: true, Total: 33.000005,
			}},
		})
		report(`{"e":"executionReport","E":1794466395251,"s":"BTCUSDT","c":"cancelSweptOrder","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00300000","p":"66000.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"sweptOrder","x":"CANCELED","X":"CANCELED","r":"NONE","i":4293160,"l":"0.00000000","z":"0.00250000","L":"0.00000000","n":"0","N":null,"T":1794466395250,"t":-1,"I":8641997,"w":false,"m":false,"M":false,"O":1794466395099,"Z":"164.98000500","Y":"0.00000000","Q":"0.00000000","W":1794466395099}`, &order.Detail{
			Price: 66000.01, Amount: 0.003, AverageExecutedPrice: 65992.002, ExecutedAmount: 0.0025, RemainingAmount: 0.0005, Cost: 164.980005, CostAsset: currency.USDT, OrderID: "4293160",
			ClientOrderID: "sweptOrder", Type: order.Limit, Side: order.Buy, Status: order.Cancelled, Date: time.UnixMilli(1794466395099), LastUpdated: time.UnixMilli(1794466395250),
			Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel,
		})
		report(`{"e":"executionReport","E":1794466395301,"s":"BTCUSDT","c":"sweptOrder","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00100000","p":"65900.02000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"NEW","X":"NEW","r":"NONE","i":4293163,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466395300,"t":-1,"I":8641998,"w":true,"m":false,"M":false,"O":1794466395300,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","W":1794466395300}`, &order.Detail{
			Price: 65900.02, Amount: 0.001, RemainingAmount: 0.001, CostAsset: currency.USDT, OrderID: "4293163", ClientOrderID: "sweptOrder", Type: order.Limit, Side: order.Buy,
			Status: order.New, Date: time.UnixMilli(1794466395300), LastUpdated: time.UnixMilli(1794466395300), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel,
		})
		report(`{"e":"executionReport","E":1794466395321,"s":"BTCUSDT","c":"ocoStopLossLeg","S":"SELL","o":"STOP_LOSS_LIMIT","f":"GTC","q":"0.00050000","p":"63800.08000000","P":"63900.07000000","F":"0.00000000","g":2049,"C":"","x":"NEW","X":"NEW","r":"NONE","i":4293164,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466395320,"t":-1,"I":8641999,"w":false,"m":false,"M":false,"O":1794466395320,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000"}`, &order.Detail{
			Price: 63800.08, Amount: 0.0005, TriggerPrice: 63900.07, RemainingAmount: 0.0005, CostAsset: currency.USDT, OrderID: "4293164", ClientOrderID: "ocoStopLossLeg", Type: order.StopLimit,
			Side: order.Sell, Status: order.New, Date: time.UnixMilli(1794466395320), LastUpdated: time.UnixMilli(1794466395320), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel,
		})
		report(`{"e":"executionReport","E":1794466395321,"s":"BTCUSDT","c":"ocoLimitMakerLeg","S":"SELL","o":"LIMIT_MAKER","f":"GTC","q":"0.00050000","p":"68500.06000000","P":"0.00000000","F":"0.00000000","g":2049,"C":"","x":"NEW","X":"NEW","r":"NONE","i":4293165,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466395320,"t":-1,"I":8642000,"w":true,"m":false,"M":false,"O":1794466395320,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","W":1794466395320}`, &order.Detail{
			Price: 68500.06, Amount: 0.0005, RemainingAmount: 0.0005, CostAsset: currency.USDT, OrderID: "4293165", ClientOrderID: "ocoLimitMakerLeg", Type: order.LimitMaker, Side: order.Sell,
			Status: order.New, Date: time.UnixMilli(1794466395320), LastUpdated: time.UnixMilli(1794466395320), Pair: currency.NewBTCUSDT(), TimeInForce: order.PostOnly,
		})
		listStatus(`{"e":"listStatus","E":1794466395322,"s":"BTCUSDT","g":2049,"c":"OCO","l":"EXEC_STARTED","L":"EXECUTING","r":"NONE","C":"ocoSellList","T":1794466395320,"O":[{"s":"BTCUSDT","i":4293164,"c":"ocoStopLossLeg"},{"s":"BTCUSDT","i":4293165,"c":"ocoLimitMakerLeg"}]}`, &WebsocketListStatus{
			EventType: "listStatus", EventTime: types.Time(time.UnixMilli(1794466395322)), Symbol: "BTCUSDT", OrderListID: 2049, ContingencyType: "OCO", ListStatusType: "EXEC_STARTED",
			ListOrderStatus: "EXECUTING", ListRejectReason: "NONE", ListClientOrderID: "ocoSellList", TransactionTime: types.Time(time.UnixMilli(1794466395320)), Orders: ocoOrders,
		}, &order.Detail{
			OrderID: "list-2049", ClientOrderID: "ocoSellList", Status: order.Active, Date: time.UnixMilli(1794466395320), LastUpdated: time.UnixMilli(1794466395320), Pair: currency.NewBTCUSDT(),
		})
		report(`{"e":"executionReport","E":1794466395331,"s":"BTCUSD","c":"dollarBuy","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00100000","p":"65800.03000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"NEW","X":"NEW","r":"NONE","i":1208447,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466395330,"t":-1,"I":3204411,"w":true,"m":false,"M":false,"O":1794466395330,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","W":1794466395330}`, &order.Detail{
			Price: 65800.03, Amount: 0.001, RemainingAmount: 0.001, CostAsset: currency.USD, OrderID: "1208447", ClientOrderID: "dollarBuy", Type: order.Limit, Side: order.Buy,
			Status: order.New, Date: time.UnixMilli(1794466395330), LastUpdated: time.UnixMilli(1794466395330), Pair: currency.NewBTCUSD(), TimeInForce: order.GoodTillCancel,
		})
		report(`{"e":"executionReport","E":1794466395351,"s":"ETHUSDT","c":"etherBuy","S":"BUY","o":"LIMIT","f":"GTC","q":"0.02000000","p":"2500.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"PARTIALLY_FILLED","r":"NONE","i":4293160,"l":"0.01000000","z":"0.01000000","L":"2500.01000000","n":"0.00001000","N":"ETH","T":1794466395350,"t":101,"I":201,"w":true,"m":true,"M":true,"O":1794466395000,"Z":"25.00010000","Y":"25.00010000","Q":"0.00000000","W":1794466395000}`, &order.Detail{
			Price: 2500.01, Amount: 0.02, AverageExecutedPrice: 2500.01, ExecutedAmount: 0.01, RemainingAmount: 0.01, Cost: 25.0001, CostAsset: currency.USDT, FeeAsset: currency.ETH,
			OrderID: "4293160", ClientOrderID: "etherBuy", Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled, Date: time.UnixMilli(1794466395000),
			LastUpdated: time.UnixMilli(1794466395350), Pair: ethUSDT, TimeInForce: order.GoodTillCancel, Trades: []order.TradeHistory{{
				Price: 2500.01, Amount: 0.01, Fee: 0.00001, FeeAsset: "ETH", TID: "101", Type: order.Limit, Side: order.Buy, Timestamp: time.UnixMilli(1794466395350), IsMaker: true, Total: 25.0001,
			}},
		})
		require.NoError(t, event(`{"e":"eventStreamTerminated","E":1794466395380}`), "wsHandleAPIData must not error for the end of the User Data Stream")
		relay("the end of the User Data Stream", &WebsocketEventStreamTerminated{EventType: "eventStreamTerminated", EventTime: types.Time(time.UnixMilli(1794466395380))})
		ex.userData.expect("resubscribe", configuredCredentials(t, ex))
		require.NoError(t, ex.wsHandleAPIData(t.Context(), conn, []byte(`{"id":"resubscribe","status":200,"result":{"subscriptionId":1}}`)), "wsHandleAPIData must not error for the User Data Stream subscribed again")
		ex.userData.expect("otherAccount", &accounts.Credentials{Key: "other-account-key", Secret: "other-account-secret"})
		require.NoError(t, ex.wsHandleAPIData(t.Context(), conn, []byte(`{"id":"otherAccount","status":200,"result":{"subscriptionId":2}}`)), "wsHandleAPIData must not error for another account's User Data Stream subscribed on the connection")
		subscriptionID = 2
		report(`{"e":"executionReport","E":1794466395401,"s":"BTCUSDT","c":"sweptOrder","S":"BUY","o":"LIMIT","f":"IOC","q":"0.00200000","p":"68100.00000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"NEW","X":"NEW","r":"NONE","i":4293170,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466395400,"t":-1,"I":8642002,"w":false,"m":false,"M":false,"O":1794466395400,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000"}`, &order.Detail{
			Price: 68100, Amount: 0.002, RemainingAmount: 0.002, CostAsset: currency.USDT, OrderID: "4293170", ClientOrderID: "sweptOrder", Type: order.Limit, Side: order.Buy, Status: order.New,
			Date: time.UnixMilli(1794466395400), LastUpdated: time.UnixMilli(1794466395400), Pair: currency.NewBTCUSDT(), TimeInForce: order.ImmediateOrCancel,
		})
		subscriptionID = 1
		report(`{"e":"executionReport","E":1794466395401,"s":"BTCUSDT","c":"takeProfitOrder","S":"SELL","o":"TAKE_PROFIT_LIMIT","f":"GTC","q":"0.00200000","p":"68000.04000000","P":"67900.05000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"FILLED","r":"NONE","i":4293162,"l":"0.00200000","z":"0.00200000","L":"68000.04000000","n":"0.13600008","N":"USDT","T":1794466395400,"t":105,"I":8642003,"w":false,"m":true,"M":true,"O":1794466395119,"Z":"136.00008000","Y":"136.00008000","Q":"0.00000000","W":1794466395390}`, &order.Detail{
			Price: 68000.04, Amount: 0.002, TriggerPrice: 67900.05, AverageExecutedPrice: 68000.04, ExecutedAmount: 0.002, Cost: 136.00008, CostAsset: currency.USDT, FeeAsset: currency.USDT,
			OrderID: "4293162", ClientOrderID: "takeProfitOrder", Type: order.TakeProfitLimit, Side: order.Sell, Status: order.Filled, Date: time.UnixMilli(1794466395119),
			LastUpdated: time.UnixMilli(1794466395400), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel, Trades: []order.TradeHistory{{
				Price: 68000.04, Amount: 0.002, Fee: 0.13600008, FeeAsset: "USDT", TID: "105", Type: order.TakeProfitLimit, Side: order.Sell, Timestamp: time.UnixMilli(1794466395400), IsMaker: true, Total: 136.00008,
			}},
		})
		subscriptionID = 2
		report(`{"e":"executionReport","E":1794466395401,"s":"BTCUSDT","c":"sweptOrder","S":"BUY","o":"LIMIT","f":"IOC","q":"0.00200000","p":"68100.00000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"FILLED","r":"NONE","i":4293170,"l":"0.00200000","z":"0.00200000","L":"68000.04000000","n":"0.00000200","N":"BTC","T":1794466395400,"t":105,"I":8642004,"w":false,"m":false,"M":true,"O":1794466395400,"Z":"136.00008000","Y":"136.00008000","Q":"0.00000000"}`, &order.Detail{
			Price: 68100, Amount: 0.002, AverageExecutedPrice: 68000.04, ExecutedAmount: 0.002, Cost: 136.00008, CostAsset: currency.USDT, FeeAsset: currency.BTC, OrderID: "4293170",
			ClientOrderID: "sweptOrder", Type: order.Limit, Side: order.Buy, Status: order.Filled, Date: time.UnixMilli(1794466395400), LastUpdated: time.UnixMilli(1794466395400),
			Pair: currency.NewBTCUSDT(), TimeInForce: order.ImmediateOrCancel, Trades: []order.TradeHistory{{
				Price: 68000.04, Amount: 0.002, Fee: 0.000002, FeeAsset: "BTC", TID: "105", Type: order.Limit, Side: order.Buy, Timestamp: time.UnixMilli(1794466395400), Total: 136.00008,
			}},
		})
		subscriptionID = 1
		report(`{"e":"executionReport","E":1794466395421,"s":"BTCUSDT","c":"ocoLimitMakerLeg","S":"SELL","o":"LIMIT_MAKER","f":"GTC","q":"0.00050000","p":"68500.06000000","P":"0.00000000","F":"0.00000000","g":2049,"C":"","x":"TRADE","X":"FILLED","r":"NONE","i":4293165,"l":"0.00050000","z":"0.00050000","L":"68500.06000000","n":"0.03425003","N":"USDT","T":1794466395420,"t":106,"I":8642006,"w":false,"m":true,"M":true,"O":1794466395320,"Z":"34.25003000","Y":"34.25003000","Q":"0.00000000","W":1794466395320}`, &order.Detail{
			Price: 68500.06, Amount: 0.0005, AverageExecutedPrice: 68500.06, ExecutedAmount: 0.0005, Cost: 34.25003, CostAsset: currency.USDT, FeeAsset: currency.USDT, OrderID: "4293165",
			ClientOrderID: "ocoLimitMakerLeg", Type: order.LimitMaker, Side: order.Sell, Status: order.Filled, Date: time.UnixMilli(1794466395320), LastUpdated: time.UnixMilli(1794466395420),
			Pair: currency.NewBTCUSDT(), TimeInForce: order.PostOnly, Trades: []order.TradeHistory{{
				Price: 68500.06, Amount: 0.0005, Fee: 0.03425003, FeeAsset: "USDT", TID: "106", Type: order.LimitMaker, Side: order.Sell, Timestamp: time.UnixMilli(1794466395420), IsMaker: true, Total: 34.25003,
			}},
		})
		report(`{"e":"executionReport","E":1794466395421,"s":"BTCUSDT","c":"ocoStopLossLeg","S":"SELL","o":"STOP_LOSS_LIMIT","f":"GTC","q":"0.00050000","p":"63800.08000000","P":"63900.07000000","F":"0.00000000","g":2049,"C":"","x":"EXPIRED","X":"EXPIRED","r":"NONE","i":4293164,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466395420,"t":-1,"I":8642007,"w":false,"m":false,"M":false,"O":1794466395320,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","eR":"OCO_TRIGGER"}`, &order.Detail{
			Price: 63800.08, Amount: 0.0005, TriggerPrice: 63900.07, RemainingAmount: 0.0005, CostAsset: currency.USDT, OrderID: "4293164", ClientOrderID: "ocoStopLossLeg", Type: order.StopLimit,
			Side: order.Sell, Status: order.Expired, Date: time.UnixMilli(1794466395320), LastUpdated: time.UnixMilli(1794466395420), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel,
		})
		listStatus(`{"e":"listStatus","E":1794466395422,"s":"BTCUSDT","g":2049,"c":"OCO","l":"ALL_DONE","L":"ALL_DONE","r":"NONE","C":"ocoSellList","T":1794466395420,"O":[{"s":"BTCUSDT","i":4293164,"c":"ocoStopLossLeg"},{"s":"BTCUSDT","i":4293165,"c":"ocoLimitMakerLeg"}]}`, &WebsocketListStatus{
			EventType: "listStatus", EventTime: types.Time(time.UnixMilli(1794466395422)), Symbol: "BTCUSDT", OrderListID: 2049, ContingencyType: "OCO", ListStatusType: "ALL_DONE",
			ListOrderStatus: "ALL_DONE", ListRejectReason: "NONE", ListClientOrderID: "ocoSellList", TransactionTime: types.Time(time.UnixMilli(1794466395420)), Orders: ocoOrders,
		}, &order.Detail{
			OrderID: "list-2049", ClientOrderID: "ocoSellList", Status: order.Closed, Date: time.UnixMilli(1794466395420), LastUpdated: time.UnixMilli(1794466395420), Pair: currency.NewBTCUSDT(),
		})
		for i, got := range relays {
			assert.Equalf(t, expected[i], got, "%s should be left as it was relayed after the later events", names[i])
		}
		assert.Empty(t, ex.Websocket.DataHandler.C, "each report should be relayed once")
	})

	t.Run("an exchange relays reports of iceberg, stop, take profit, self-trade prevented, market, quote sized, IOC and FOK orders in turn", func(t *testing.T) {
		t.Parallel()
		ex := subscribedUserDataTestExchange(t)
		ethBTC := currency.NewPair(currency.ETH, currency.BTC)
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USDT), ethBTC}, false), "StorePairs must not error")
		conn := newFakeConnection()
		var relays, expected []*order.Detail
		report := func(frame string, exp *order.Detail) {
			t.Helper()
			exp.Exchange, exp.AssetType = ex.Name, asset.Spot
			for i := range exp.Trades {
				exp.Trades[i].Exchange = ex.Name
			}
			require.NoErrorf(t, ex.wsHandleAPIData(t.Context(), conn, []byte(`{"subscriptionId":0,"event":`+frame+`}`)), "wsHandleAPIData must not error for order %s's %s report", exp.OrderID, exp.Status)
			detail, ok := relayed(t, ex).(*order.Detail)
			require.Truef(t, ok, "order %s's %s report must be relayed as an order detail", exp.OrderID, exp.Status)
			assert.Equalf(t, exp, detail, "order %s's %s report should be relayed as the order's detail", exp.OrderID, exp.Status)
			relays, expected = append(relays, detail), append(expected, exp)
		}
		report(`{"e":"executionReport","E":1794466396101,"s":"BTCUSDT","c":"icebergBuy","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00300000","p":"65800.01000000","P":"0.00000000","F":"0.00100000","g":-1,"C":"","x":"NEW","X":"NEW","r":"NONE","i":4293180,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466396100,"t":-1,"I":8642010,"w":true,"m":false,"M":false,"O":1794466396100,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","W":1794466396100}`, &order.Detail{
			Price: 65800.01, Amount: 0.003, RemainingAmount: 0.003, CostAsset: currency.USDT, OrderID: "4293180", ClientOrderID: "icebergBuy", Type: order.Limit, Side: order.Buy,
			Status: order.New, Date: time.UnixMilli(1794466396100), LastUpdated: time.UnixMilli(1794466396100), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel,
		})
		report(`{"e":"executionReport","E":1794466396111,"s":"BTCUSDT","c":"stopMarketSell","S":"SELL","o":"STOP_LOSS","f":"GTC","q":"0.00200000","p":"0.00000000","P":"64100.03000000","F":"0.00000000","g":-1,"C":"","x":"NEW","X":"NEW","r":"NONE","i":4293181,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466396110,"t":-1,"I":8642011,"w":false,"m":false,"M":false,"O":1794466396110,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000"}`, &order.Detail{
			Amount: 0.002, TriggerPrice: 64100.03, RemainingAmount: 0.002, CostAsset: currency.USDT, OrderID: "4293181", ClientOrderID: "stopMarketSell", Type: order.StopMarket, Side: order.Sell,
			Status: order.New, Date: time.UnixMilli(1794466396110), LastUpdated: time.UnixMilli(1794466396110), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel,
		})
		report(`{"e":"executionReport","E":1794466396121,"s":"BTCUSDT","c":"restingBuy","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00100000","p":"65900.02000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"NEW","X":"NEW","r":"NONE","i":4293182,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466396120,"t":-1,"I":8642012,"w":true,"m":false,"M":false,"O":1794466396120,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","W":1794466396120}`, &order.Detail{
			Price: 65900.02, Amount: 0.001, RemainingAmount: 0.001, CostAsset: currency.USDT, OrderID: "4293182", ClientOrderID: "restingBuy", Type: order.Limit, Side: order.Buy,
			Status: order.New, Date: time.UnixMilli(1794466396120), LastUpdated: time.UnixMilli(1794466396120), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel,
		})
		marketSell := func(status order.Status, executed, remaining, cost, average float64, fee currency.Code, fills ...order.TradeHistory) *order.Detail {
			return &order.Detail{
				Amount: 0.0015, AverageExecutedPrice: average, ExecutedAmount: executed, RemainingAmount: remaining, Cost: cost, CostAsset: currency.USDT, FeeAsset: fee, OrderID: "4293183",
				ClientOrderID: "marketSell", Type: order.Market, Side: order.Sell, Status: status, Date: time.UnixMilli(1794466396200), LastUpdated: time.UnixMilli(1794466396200),
				Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel, Trades: fills,
			}
		}
		report(`{"e":"executionReport","E":1794466396201,"s":"BTCUSDT","c":"marketSell","S":"SELL","o":"MARKET","f":"GTC","q":"0.00150000","p":"0.00000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"NEW","X":"NEW","r":"NONE","i":4293183,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466396200,"t":-1,"I":8642013,"w":false,"m":false,"M":false,"O":1794466396200,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000"}`, marketSell(order.New, 0, 0.0015, 0, 0, currency.EMPTYCODE))
		report(`{"e":"executionReport","E":1794466396201,"s":"BTCUSDT","c":"restingBuy","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00100000","p":"65900.02000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE_PREVENTION","X":"EXPIRED_IN_MATCH","r":"NONE","i":4293182,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466396200,"t":-1,"I":8642015,"w":false,"m":false,"M":false,"O":1794466396120,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","W":1794466396120,"V":"EXPIRE_MAKER","v":17,"U":4293183,"A":"0.00100000","B":"0.00100000","pl":"0.00100000","pL":"65900.02000000","pY":"65.90002000","eR":"EXPIRED_IN_MATCH"}`, &order.Detail{
			Price: 65900.02, Amount: 0.001, RemainingAmount: 0.001, CostAsset: currency.USDT, OrderID: "4293182", ClientOrderID: "restingBuy", Type: order.Limit, Side: order.Buy,
			Status: order.Expired, Date: time.UnixMilli(1794466396120), LastUpdated: time.UnixMilli(1794466396200), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel,
		})
		report(`{"e":"executionReport","E":1794466396201,"s":"BTCUSDT","c":"marketSell","S":"SELL","o":"MARKET","f":"GTC","q":"0.00150000","p":"0.00000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"PARTIALLY_FILLED","r":"NONE","i":4293183,"l":"0.00050000","z":"0.00050000","L":"65899.99000000","n":"0.03294999","N":"USDT","T":1794466396200,"t":107,"I":8642016,"w":false,"m":false,"M":true,"O":1794466396200,"Z":"32.94999500","Y":"32.94999500","Q":"0.00000000"}`, marketSell(order.PartiallyFilled, 0.0005, 0.001, 32.949995, 65899.99, currency.USDT, order.TradeHistory{
			Price: 65899.99, Amount: 0.0005, Fee: 0.03294999, FeeAsset: "USDT", TID: "107", Type: order.Market, Side: order.Sell, Timestamp: time.UnixMilli(1794466396200), Total: 32.949995,
		}))
		report(`{"e":"executionReport","E":1794466396201,"s":"BTCUSDT","c":"marketSell","S":"SELL","o":"MARKET","f":"GTC","q":"0.00150000","p":"0.00000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"FILLED","r":"NONE","i":4293183,"l":"0.00100000","z":"0.00150000","L":"65890.00000000","n":"0.06589000","N":"USDT","T":1794466396200,"t":108,"I":8642017,"w":false,"m":false,"M":true,"O":1794466396200,"Z":"98.83999500","Y":"65.89000000","Q":"0.00000000"}`, marketSell(order.Filled, 0.0015, 0, 98.839995, 65893.33, currency.USDT, order.TradeHistory{
			Price: 65890, Amount: 0.001, Fee: 0.06589, FeeAsset: "USDT", TID: "108", Type: order.Market, Side: order.Sell, Timestamp: time.UnixMilli(1794466396200), Total: 65.89,
		}))
		report(`{"e":"executionReport","E":1794466396301,"s":"BTCUSDT","c":"quoteBuy","S":"BUY","o":"MARKET","f":"GTC","q":"0.00151000","p":"0.00000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"FILLED","r":"NONE","i":4293184,"l":"0.00151000","z":"0.00151000","L":"66000.00000000","n":"0.00000151","N":"BTC","T":1794466396300,"t":109,"I":8642018,"w":false,"m":false,"M":true,"O":1794466396300,"Z":"99.66000000","Y":"99.66000000","Q":"100.00000000"}`, &order.Detail{
			Amount: 0.00151, AverageExecutedPrice: 66000, QuoteAmount: 100, ExecutedAmount: 0.00151, Cost: 99.66, CostAsset: currency.USDT, FeeAsset: currency.BTC, OrderID: "4293184",
			ClientOrderID: "quoteBuy", Type: order.Market, Side: order.Buy, Status: order.Filled, Date: time.UnixMilli(1794466396300), LastUpdated: time.UnixMilli(1794466396300),
			Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel, Trades: []order.TradeHistory{{
				Price: 66000, Amount: 0.00151, Fee: 0.00000151, FeeAsset: "BTC", TID: "109", Type: order.Market, Side: order.Buy, Timestamp: time.UnixMilli(1794466396300), Total: 99.66,
			}},
		})
		report(`{"e":"executionReport","E":1794466396401,"s":"ETHBTC","c":"etherBitcoinBuy","S":"BUY","o":"MARKET","f":"GTC","q":"0.01250000","p":"0.00000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"FILLED","r":"NONE","i":1200345,"l":"0.01250000","z":"0.01250000","L":"0.02971000","n":"0.00001250","N":"ETH","T":1794466396400,"t":702001,"I":3108721,"w":false,"m":false,"M":true,"O":1794466396400,"Z":"0.00037137","Y":"0.00037137","Q":"0.00000000"}`, &order.Detail{
			Amount: 0.0125, AverageExecutedPrice: 0.0297096, ExecutedAmount: 0.0125, Cost: 0.00037137, CostAsset: currency.BTC, FeeAsset: currency.ETH, OrderID: "1200345",
			ClientOrderID: "etherBitcoinBuy", Type: order.Market, Side: order.Buy, Status: order.Filled, Date: time.UnixMilli(1794466396400), LastUpdated: time.UnixMilli(1794466396400),
			Pair: ethBTC, TimeInForce: order.GoodTillCancel, Trades: []order.TradeHistory{{
				Price: 0.02971, Amount: 0.0125, Fee: 0.0000125, FeeAsset: "ETH", TID: "702001", Type: order.Market, Side: order.Buy, Timestamp: time.UnixMilli(1794466396400), Total: 0.00037137,
			}},
		})
		report(`{"e":"executionReport","E":1794466396411,"s":"BTCUSDT","c":"takeProfitMarketSell","S":"SELL","o":"TAKE_PROFIT","f":"GTC","q":"0.00100000","p":"0.00000000","P":"67000.00000000","F":"0.00000000","g":-1,"C":"","x":"NEW","X":"NEW","r":"NONE","i":4293185,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466396410,"t":-1,"I":8642019,"w":false,"m":false,"M":false,"O":1794466396410,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000"}`, &order.Detail{
			Amount: 0.001, TriggerPrice: 67000, RemainingAmount: 0.001, CostAsset: currency.USDT, OrderID: "4293185", ClientOrderID: "takeProfitMarketSell", Type: order.TakeProfitMarket,
			Side: order.Sell, Status: order.New, Date: time.UnixMilli(1794466396410), LastUpdated: time.UnixMilli(1794466396410), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel,
		})
		report(`{"e":"executionReport","E":1794466396421,"s":"BTCUSDT","c":"quoteSell","S":"SELL","o":"MARKET","f":"GTC","q":"0.00145000","p":"0.00000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"FILLED","r":"NONE","i":4293186,"l":"0.00145000","z":"0.00145000","L":"66000.00000000","n":"0.09570000","N":"USDT","T":1794466396420,"t":110,"I":8642020,"w":false,"m":false,"M":true,"O":1794466396420,"Z":"95.70000000","Y":"95.70000000","Q":"95.70000000"}`, &order.Detail{
			Amount: 0.00145, AverageExecutedPrice: 66000, QuoteAmount: 95.7, ExecutedAmount: 0.00145, Cost: 95.7, CostAsset: currency.USDT, FeeAsset: currency.USDT, OrderID: "4293186",
			ClientOrderID: "quoteSell", Type: order.Market, Side: order.Sell, Status: order.Filled, Date: time.UnixMilli(1794466396420), LastUpdated: time.UnixMilli(1794466396420),
			Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel, Trades: []order.TradeHistory{{
				Price: 66000, Amount: 0.00145, Fee: 0.0957, FeeAsset: "USDT", TID: "110", Type: order.Market, Side: order.Sell, Timestamp: time.UnixMilli(1794466396420), Total: 95.7,
			}},
		})
		iocBuy := func(status order.Status, executed, remaining, cost, average float64, updated int64, fee currency.Code, fills ...order.TradeHistory) *order.Detail {
			return &order.Detail{
				Price: 66010, Amount: 0.003, AverageExecutedPrice: average, ExecutedAmount: executed, RemainingAmount: remaining, Cost: cost, CostAsset: currency.USDT, FeeAsset: fee,
				OrderID: "4293187", ClientOrderID: "iocBuy", Type: order.Limit, Side: order.Buy, Status: status, Date: time.UnixMilli(1794466396430), LastUpdated: time.UnixMilli(updated),
				Pair: currency.NewBTCUSDT(), TimeInForce: order.ImmediateOrCancel, Trades: fills,
			}
		}
		report(`{"e":"executionReport","E":1794466396431,"s":"BTCUSDT","c":"iocBuy","S":"BUY","o":"LIMIT","f":"IOC","q":"0.00300000","p":"66010.00000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"NEW","X":"NEW","r":"NONE","i":4293187,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466396430,"t":-1,"I":8642021,"w":false,"m":false,"M":false,"O":1794466396430,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000"}`, iocBuy(order.New, 0, 0.003, 0, 0, 1794466396430, currency.EMPTYCODE))
		report(`{"e":"executionReport","E":1794466396431,"s":"BTCUSDT","c":"iocBuy","S":"BUY","o":"LIMIT","f":"IOC","q":"0.00300000","p":"66010.00000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"PARTIALLY_FILLED","r":"NONE","i":4293187,"l":"0.00100000","z":"0.00100000","L":"66005.00000000","n":"0.00000100","N":"BTC","T":1794466396430,"t":111,"I":8642022,"w":false,"m":false,"M":true,"O":1794466396430,"Z":"66.00500000","Y":"66.00500000","Q":"0.00000000"}`, iocBuy(order.PartiallyFilled, 0.001, 0.002, 66.005, 66005, 1794466396430, currency.BTC, order.TradeHistory{
			Price: 66005, Amount: 0.001, Fee: 0.000001, FeeAsset: "BTC", TID: "111", Type: order.Limit, Side: order.Buy, Timestamp: time.UnixMilli(1794466396430), Total: 66.005,
		}))
		report(`{"e":"executionReport","E":1794466396431,"s":"BTCUSDT","c":"iocBuy","S":"BUY","o":"LIMIT","f":"IOC","q":"0.00300000","p":"66010.00000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"EXPIRED","X":"EXPIRED","r":"NONE","i":4293187,"l":"0.00000000","z":"0.00100000","L":"0.00000000","n":"0","N":null,"T":1794466396430,"t":-1,"I":8642023,"w":false,"m":false,"M":false,"O":1794466396430,"Z":"66.00500000","Y":"0.00000000","Q":"0.00000000","eR":"UNFILLED_IOC_QUANTITY_EXPIRED"}`, iocBuy(order.Expired, 0.001, 0.002, 66.005, 66005, 1794466396430, currency.EMPTYCODE))
		fokBuy := func(status order.Status) *order.Detail {
			return &order.Detail{
				Price: 65950, Amount: 0.002, RemainingAmount: 0.002, CostAsset: currency.USDT, OrderID: "4293188", ClientOrderID: "fokBuy", Type: order.Limit, Side: order.Buy, Status: status,
				Date: time.UnixMilli(1794466396440), LastUpdated: time.UnixMilli(1794466396440), Pair: currency.NewBTCUSDT(), TimeInForce: order.FillOrKill,
			}
		}
		report(`{"e":"executionReport","E":1794466396441,"s":"BTCUSDT","c":"fokBuy","S":"BUY","o":"LIMIT","f":"FOK","q":"0.00200000","p":"65950.00000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"NEW","X":"NEW","r":"NONE","i":4293188,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466396440,"t":-1,"I":8642024,"w":false,"m":false,"M":false,"O":1794466396440,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000"}`, fokBuy(order.New))
		report(`{"e":"executionReport","E":1794466396441,"s":"BTCUSDT","c":"fokBuy","S":"BUY","o":"LIMIT","f":"FOK","q":"0.00200000","p":"65950.00000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"EXPIRED","X":"EXPIRED","r":"NONE","i":4293188,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466396440,"t":-1,"I":8642025,"w":false,"m":false,"M":false,"O":1794466396440,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","eR":"UNFILLED_FOK_ORDER_EXPIRED"}`, fokBuy(order.Expired))
		partlyFilled := func(status order.Status, updated int64, fee currency.Code, fills ...order.TradeHistory) *order.Detail {
			return &order.Detail{
				Price: 66000.01, Amount: 0.003, AverageExecutedPrice: 66000.01, ExecutedAmount: 0.001, RemainingAmount: 0.002, Cost: 66.00001, CostAsset: currency.USDT,
				FeeAsset: fee, OrderID: "4293190", ClientOrderID: "partlyFilledBuy", Type: order.Limit, Side: order.Buy, Status: status, Date: time.UnixMilli(1794466397100),
				LastUpdated: time.UnixMilli(updated), Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel, Trades: fills,
			}
		}
		report(`{"e":"executionReport","E":1794466397101,"s":"BTCUSDT","c":"partlyFilledBuy","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00300000","p":"66000.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"PARTIALLY_FILLED","r":"NONE","i":4293190,"l":"0.00100000","z":"0.00100000","L":"66000.01000000","n":"0.00000100","N":"BTC","T":1794466397100,"t":112,"I":8642030,"w":true,"m":false,"M":true,"O":1794466397100,"Z":"66.00001000","Y":"66.00001000","Q":"0.00000000","W":1794466397100}`, partlyFilled(order.PartiallyFilled, 1794466397100, currency.BTC, order.TradeHistory{
			Price: 66000.01, Amount: 0.001, Fee: 0.000001, FeeAsset: "BTC", TID: "112", Type: order.Limit, Side: order.Buy, Timestamp: time.UnixMilli(1794466397100), Total: 66.00001,
		}))
		report(`{"e":"executionReport","E":1794466397301,"s":"BTCUSDT","c":"partlyFilledBuy","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00300000","p":"66000.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE_PREVENTION","X":"EXPIRED_IN_MATCH","r":"NONE","i":4293190,"l":"0.00000000","z":"0.00100000","L":"0.00000000","n":"0","N":null,"T":1794466397300,"t":-1,"I":8642032,"w":false,"m":false,"M":false,"O":1794466397100,"Z":"66.00001000","Y":"0.00000000","Q":"0.00000000","W":1794466397100,"V":"EXPIRE_MAKER","v":18,"U":4293191,"A":"0.00200000","B":"0.00200000","pl":"0.00200000","pL":"66000.01000000","pY":"132.00002000","eR":"EXPIRED_IN_MATCH"}`, partlyFilled(order.Expired, 1794466397300, currency.EMPTYCODE))
		// Binance.US documents neither DECREMENT's nor TRANSFER's stream reports; these give an order STP left open the prevented
		// quantity fields its STP FAQ gives "execution reports for orders involved in STP"
		decrementedBuy := func(status order.Status, executed, remaining, cost, average float64, updated int64, fee currency.Code, fills ...order.TradeHistory) *order.Detail {
			return &order.Detail{
				Price: 66000.01, Amount: 0.006, AverageExecutedPrice: average, ExecutedAmount: executed, RemainingAmount: remaining, Cost: cost, CostAsset: currency.USDT, FeeAsset: fee,
				OrderID: "4293192", ClientOrderID: "decrementedBuy", Type: order.Limit, Side: order.Buy, Status: status, Date: time.UnixMilli(1794466397400), LastUpdated: time.UnixMilli(updated),
				Pair: currency.NewBTCUSDT(), TimeInForce: order.GoodTillCancel, Trades: fills,
			}
		}
		ethUSDT := currency.NewPair(currency.ETH, currency.USDT)
		transferredSell := func(status order.Status, remaining float64, updated int64) *order.Detail {
			return &order.Detail{
				Price: 2500.02, Amount: 0.04, RemainingAmount: remaining, CostAsset: currency.USDT, OrderID: "1200346", ClientOrderID: "transferredSell", Type: order.LimitMaker,
				Side: order.Sell, Status: status, Date: time.UnixMilli(1794466397800), LastUpdated: time.UnixMilli(updated), Pair: ethUSDT, TimeInForce: order.PostOnly,
			}
		}
		decrementingTaker := func(orderID, clientOrderID string, pair currency.Pair, side order.Side, price, amount float64, placed int64, status order.Status) *order.Detail {
			return &order.Detail{
				Price: price, Amount: amount, RemainingAmount: amount, CostAsset: currency.USDT, OrderID: orderID, ClientOrderID: clientOrderID, Type: order.Limit, Side: side,
				Status: status, Date: time.UnixMilli(placed), LastUpdated: time.UnixMilli(placed), Pair: pair, TimeInForce: order.GoodTillCancel,
			}
		}
		report(`{"e":"executionReport","E":1794466397401,"s":"BTCUSDT","c":"decrementedBuy","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00600000","p":"66000.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"NEW","X":"NEW","r":"NONE","i":4293192,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466397400,"t":-1,"I":8642033,"w":true,"m":false,"M":false,"O":1794466397400,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","W":1794466397400,"V":"DECREMENT"}`, decrementedBuy(order.New, 0, 0.006, 0, 0, 1794466397400, currency.EMPTYCODE))
		report(`{"e":"executionReport","E":1794466397501,"s":"BTCUSDT","c":"decrementingSell","S":"SELL","o":"LIMIT","f":"GTC","q":"0.00200000","p":"66000.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"NEW","X":"NEW","r":"NONE","i":4293193,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466397500,"t":-1,"I":8642034,"w":false,"m":false,"M":false,"O":1794466397500,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","V":"DECREMENT"}`, decrementingTaker("4293193", "decrementingSell", currency.NewBTCUSDT(), order.Sell, 66000.01, 0.002, 1794466397500, order.New))
		report(`{"e":"executionReport","E":1794466397501,"s":"BTCUSDT","c":"decrementedBuy","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00600000","p":"66000.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE_PREVENTION","X":"NEW","r":"NONE","i":4293192,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466397500,"t":-1,"I":8642035,"w":true,"m":false,"M":false,"O":1794466397400,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","W":1794466397400,"V":"DECREMENT","v":19,"U":4293193,"A":"0.00200000","B":"0.00200000","pl":"0.00200000","pL":"66000.01000000","pY":"132.00002000"}`, decrementedBuy(order.New, 0, 0.004, 0, 0, 1794466397500, currency.EMPTYCODE))
		report(`{"e":"executionReport","E":1794466397501,"s":"BTCUSDT","c":"decrementingSell","S":"SELL","o":"LIMIT","f":"GTC","q":"0.00200000","p":"66000.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE_PREVENTION","X":"EXPIRED_IN_MATCH","r":"NONE","i":4293193,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466397500,"t":-1,"I":8642036,"w":false,"m":false,"M":false,"O":1794466397500,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","V":"DECREMENT","v":19,"U":4293192,"A":"0.00200000","B":"0.00200000","pl":"0.00200000","pL":"66000.01000000","pY":"132.00002000","eR":"EXPIRED_IN_MATCH"}`, decrementingTaker("4293193", "decrementingSell", currency.NewBTCUSDT(), order.Sell, 66000.01, 0.002, 1794466397500, order.Expired))
		report(`{"e":"executionReport","E":1794466397551,"s":"BTCUSDT","c":"secondDecrementingSell","S":"SELL","o":"LIMIT","f":"GTC","q":"0.00100000","p":"66000.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"NEW","X":"NEW","r":"NONE","i":4293194,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466397550,"t":-1,"I":8642037,"w":false,"m":false,"M":false,"O":1794466397550,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","V":"DECREMENT"}`, decrementingTaker("4293194", "secondDecrementingSell", currency.NewBTCUSDT(), order.Sell, 66000.01, 0.001, 1794466397550, order.New))
		report(`{"e":"executionReport","E":1794466397551,"s":"BTCUSDT","c":"decrementedBuy","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00600000","p":"66000.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE_PREVENTION","X":"NEW","r":"NONE","i":4293192,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466397550,"t":-1,"I":8642038,"w":true,"m":false,"M":false,"O":1794466397400,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","W":1794466397400,"V":"DECREMENT","v":20,"U":4293194,"A":"0.00300000","B":"0.00100000","pl":"0.00100000","pL":"66000.01000000","pY":"66.00001000"}`, decrementedBuy(order.New, 0, 0.003, 0, 0, 1794466397550, currency.EMPTYCODE))
		report(`{"e":"executionReport","E":1794466397551,"s":"BTCUSDT","c":"secondDecrementingSell","S":"SELL","o":"LIMIT","f":"GTC","q":"0.00100000","p":"66000.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE_PREVENTION","X":"EXPIRED_IN_MATCH","r":"NONE","i":4293194,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466397550,"t":-1,"I":8642039,"w":false,"m":false,"M":false,"O":1794466397550,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","V":"DECREMENT","v":20,"U":4293192,"A":"0.00100000","B":"0.00100000","pl":"0.00100000","pL":"66000.01000000","pY":"66.00001000","eR":"EXPIRED_IN_MATCH"}`, decrementingTaker("4293194", "secondDecrementingSell", currency.NewBTCUSDT(), order.Sell, 66000.01, 0.001, 1794466397550, order.Expired))
		report(`{"e":"executionReport","E":1794466397601,"s":"BTCUSDT","c":"decrementedBuy","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00600000","p":"66000.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"PARTIALLY_FILLED","r":"NONE","i":4293192,"l":"0.00100000","z":"0.00100000","L":"66000.01000000","n":"0.00000100","N":"BTC","T":1794466397600,"t":113,"I":8642040,"w":true,"m":true,"M":true,"O":1794466397400,"Z":"66.00001000","Y":"66.00001000","Q":"0.00000000","W":1794466397400,"V":"DECREMENT","A":"0.00300000"}`, decrementedBuy(order.PartiallyFilled, 0.001, 0.002, 66.00001, 66000.01, 1794466397600, currency.BTC, order.TradeHistory{
			Price: 66000.01, Amount: 0.001, Fee: 0.000001, FeeAsset: "BTC", TID: "113", Type: order.Limit, Side: order.Buy, Timestamp: time.UnixMilli(1794466397600), IsMaker: true, Total: 66.00001,
		}))
		report(`{"e":"executionReport","E":1794466397701,"s":"BTCUSDT","c":"decrementedBuy","S":"BUY","o":"LIMIT","f":"GTC","q":"0.00600000","p":"66000.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE","X":"FILLED","r":"NONE","i":4293192,"l":"0.00200000","z":"0.00300000","L":"66000.01000000","n":"0.00000200","N":"BTC","T":1794466397700,"t":114,"I":8642041,"w":false,"m":true,"M":true,"O":1794466397400,"Z":"198.00003000","Y":"132.00002000","Q":"0.00000000","W":1794466397400,"V":"DECREMENT","A":"0.00300000"}`, decrementedBuy(order.Filled, 0.003, 0, 198.00003, 66000.01, 1794466397700, currency.BTC, order.TradeHistory{
			Price: 66000.01, Amount: 0.002, Fee: 0.000002, FeeAsset: "BTC", TID: "114", Type: order.Limit, Side: order.Buy, Timestamp: time.UnixMilli(1794466397700), IsMaker: true, Total: 132.00002,
		}))
		report(`{"e":"executionReport","E":1794466397801,"s":"ETHUSDT","c":"transferredSell","S":"SELL","o":"LIMIT_MAKER","f":"GTC","q":"0.04000000","p":"2500.02000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"NEW","X":"NEW","r":"NONE","i":1200346,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466397800,"t":-1,"I":8642042,"w":true,"m":false,"M":false,"O":1794466397800,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","W":1794466397800,"V":"TRANSFER"}`, transferredSell(order.New, 0.04, 1794466397800))
		report(`{"e":"executionReport","E":1794466397901,"s":"ETHUSDT","c":"transferringBuy","S":"BUY","o":"LIMIT","f":"GTC","q":"0.01000000","p":"2500.02000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"NEW","X":"NEW","r":"NONE","i":1200347,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466397900,"t":-1,"I":8642043,"w":false,"m":false,"M":false,"O":1794466397900,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","V":"TRANSFER"}`, decrementingTaker("1200347", "transferringBuy", ethUSDT, order.Buy, 2500.02, 0.01, 1794466397900, order.New))
		report(`{"e":"executionReport","E":1794466397901,"s":"ETHUSDT","c":"transferredSell","S":"SELL","o":"LIMIT_MAKER","f":"GTC","q":"0.04000000","p":"2500.02000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE_PREVENTION","X":"NEW","r":"NONE","i":1200346,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466397900,"t":-1,"I":8642044,"w":true,"m":false,"M":false,"O":1794466397800,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","W":1794466397800,"V":"TRANSFER","v":0,"U":1200347,"A":"0.01000000","B":"0.01000000","pl":"0.01000000","pL":"2500.02000000","pY":"25.00020000"}`, transferredSell(order.New, 0.03, 1794466397900))
		report(`{"e":"executionReport","E":1794466397901,"s":"ETHUSDT","c":"transferringBuy","S":"BUY","o":"LIMIT","f":"GTC","q":"0.01000000","p":"2500.02000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"TRADE_PREVENTION","X":"EXPIRED_IN_MATCH","r":"NONE","i":1200347,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466397900,"t":-1,"I":8642045,"w":false,"m":false,"M":false,"O":1794466397900,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","V":"TRANSFER","v":0,"U":1200346,"A":"0.01000000","B":"0.01000000","pl":"0.01000000","pL":"2500.02000000","pY":"25.00020000","eR":"EXPIRED_IN_MATCH"}`, decrementingTaker("1200347", "transferringBuy", ethUSDT, order.Buy, 2500.02, 0.01, 1794466397900, order.Expired))
		report(`{"e":"executionReport","E":1794466398001,"s":"ETHUSDT","c":"cancelTransferredSell","S":"SELL","o":"LIMIT_MAKER","f":"GTC","q":"0.04000000","p":"2500.02000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"transferredSell","x":"CANCELED","X":"CANCELED","r":"NONE","i":1200346,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466398000,"t":-1,"I":8642046,"w":false,"m":false,"M":false,"O":1794466397800,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","W":1794466397800,"V":"TRANSFER","A":"0.01000000"}`, transferredSell(order.Cancelled, 0.04, 1794466398000))
		report(`{"e":"executionReport","E":1794466398001,"s":"ETHUSDT","c":"replacementSell","S":"SELL","o":"LIMIT_MAKER","f":"GTC","q":"0.04000000","p":"2500.01000000","P":"0.00000000","F":"0.00000000","g":-1,"C":"","x":"REJECTED","X":"REJECTED","r":"WOULD_MATCH_IMMEDIATELY","i":1200348,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1794466398000,"t":-1,"I":8642047,"w":false,"m":false,"M":false,"O":1794466398000,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","V":"TRANSFER"}`, &order.Detail{
			Price: 2500.01, Amount: 0.04, RemainingAmount: 0.04, CostAsset: currency.USDT, OrderID: "1200348", ClientOrderID: "replacementSell", Type: order.LimitMaker, Side: order.Sell,
			Status: order.Rejected, Date: time.UnixMilli(1794466398000), LastUpdated: time.UnixMilli(1794466398000), Pair: ethUSDT, TimeInForce: order.PostOnly,
		})
		for i, detail := range relays {
			assert.Equalf(t, expected[i], detail, "order %s's %s report should be left as it was relayed after the later reports", detail.OrderID, detail.Status)
		}
		assert.Empty(t, ex.Websocket.DataHandler.C, "each report should be relayed once")
	})
}

func TestProcessListStatus(t *testing.T) {
	t.Parallel()
	ex := subscribedUserDataTestExchange(t)
	ctx := t.Context()
	event := `{"e":"listStatus","E":1564035303637,"s":"BTCUSDT","g":2,"c":"OCO","l":"EXEC_STARTED","L":"EXECUTING","r":"NONE","C":"F4QN4G8DlFATFlIUQ0cjdD","T":1564035303625,"O":[{"s":"BTCUSDT","i":17,"c":"AJYsMjErWJesZvqlJCTUgL"},{"s":"BTCUSDT","i":18,"c":"bfYPSQdLoqAJeNrOr9adzq"}]}`
	require.NoError(t, ex.processListStatus(ctx, []byte(event)), "processListStatus must not error")
	exp := &WebsocketListStatus{
		EventType:         "listStatus",
		EventTime:         types.Time(time.UnixMilli(1564035303637)),
		Symbol:            "BTCUSDT",
		OrderListID:       2,
		ContingencyType:   "OCO",
		ListStatusType:    "EXEC_STARTED",
		ListOrderStatus:   "EXECUTING",
		ListRejectReason:  "NONE",
		ListClientOrderID: "F4QN4G8DlFATFlIUQ0cjdD",
		TransactionTime:   types.Time(time.UnixMilli(1564035303625)),
		Orders: []WebsocketListStatusOrder{
			{Symbol: "BTCUSDT", OrderID: 17, ClientOrderID: "AJYsMjErWJesZvqlJCTUgL"},
			{Symbol: "BTCUSDT", OrderID: 18, ClientOrderID: "bfYPSQdLoqAJeNrOr9adzq"},
		},
	}
	assert.Equal(t, exp, relayed(t, ex), "processListStatus should relay the event decoded")
	assert.Equal(t, &order.Detail{
		Exchange:      ex.Name,
		OrderID:       "list-2",
		ClientOrderID: "F4QN4G8DlFATFlIUQ0cjdD",
		Type:          order.OCO,
		Status:        order.Active,
		AssetType:     asset.Spot,
		Date:          time.UnixMilli(1564035303625),
		LastUpdated:   time.UnixMilli(1564035303625),
		Pair:          currency.NewBTCUSDT(),
	}, relayed(t, ex), "processListStatus should relay an OCO order list as the order GCT tracks")

	require.NoError(t, ex.processListStatus(ctx, []byte(`{"e":"listStatus","s":"BTCUSDT","g":3,"c":"OTO","L":"EXECUTING"}`)), "processListStatus must not error for another contingency type")
	_, ok := relayed(t, ex).(*WebsocketListStatus)
	assert.True(t, ok, "another contingency type's event should be relayed")
	assert.Empty(t, ex.Websocket.DataHandler.C, "another contingency type should relay no order")

	require.NoError(t, ex.processListStatus(ctx, []byte(`{"e":"listStatus","s":"BTCUSDT","g":4,"c":"OCO","L":"UNKNOWN"}`)), "processListStatus must not error for an unknown list status")
	_ = relayed(t, ex)
	classification, ok := relayed(t, ex).(order.ClassificationError)
	require.True(t, ok, "an unknown list status must be relayed as a classification error")
	assert.Equal(t, "list-4", classification.OrderID, "the classification error should name the order list")

	assert.ErrorIs(t, ex.processListStatus(ctx, []byte(`{"s":"NOTAPAIR","c":"OCO"}`)), currency.ErrPairNotFound, "processListStatus should error for an unknown symbol")
	assert.Error(t, ex.processListStatus(ctx, []byte(`{"s":1}`)), "processListStatus should error for an event that does not decode")

	t.Run("each OCO order list is relayed as an order, on a pair that is not enabled or beside another list on its symbol", func(t *testing.T) {
		t.Parallel()
		lists := subscribedUserDataTestExchange(t)
		ethUSDT := currency.NewPair(currency.ETH, currency.USDT)
		enabled, err := lists.GetEnabledPairs(asset.Spot)
		require.NoError(t, err, "GetEnabledPairs must not error")
		require.False(t, enabled.Contains(ethUSDT, true), "ETH/USDT must not be enabled")
		list := func(frame string, exp *order.Detail) {
			t.Helper()
			exp.Exchange, exp.Type, exp.AssetType, exp.Status = lists.Name, order.OCO, asset.Spot, order.Active
			require.NoErrorf(t, lists.processListStatus(t.Context(), []byte(frame)), "processListStatus must not error for order list %s", exp.OrderID)
			_, ok := relayed(t, lists).(*WebsocketListStatus)
			require.Truef(t, ok, "order list %s's event must be relayed decoded", exp.OrderID)
			assert.Equalf(t, exp, relayed(t, lists), "order list %s should be relayed as the order GCT tracks", exp.OrderID)
		}
		list(`{"e":"listStatus","E":1794466395322,"s":"ETHUSDT","g":1187,"c":"OCO","l":"EXEC_STARTED","L":"EXECUTING","r":"NONE","C":"etherList","T":1794466395320,"O":[{"s":"ETHUSDT","i":81240,"c":"etherStopLeg"},{"s":"ETHUSDT","i":81241,"c":"etherLimitLeg"}]}`, &order.Detail{
			OrderID: "list-1187", ClientOrderID: "etherList", Date: time.UnixMilli(1794466395320), LastUpdated: time.UnixMilli(1794466395320), Pair: ethUSDT,
		})
		list(`{"e":"listStatus","E":1794466395332,"s":"BTCUSDT","g":2049,"c":"OCO","l":"EXEC_STARTED","L":"EXECUTING","r":"NONE","C":"firstList","T":1794466395330,"O":[{"s":"BTCUSDT","i":4293164,"c":"firstStopLeg"},{"s":"BTCUSDT","i":4293165,"c":"firstLimitLeg"}]}`, &order.Detail{
			OrderID: "list-2049", ClientOrderID: "firstList", Date: time.UnixMilli(1794466395330), LastUpdated: time.UnixMilli(1794466395330), Pair: currency.NewBTCUSDT(),
		})
		list(`{"e":"listStatus","E":1794466395342,"s":"BTCUSDT","g":2050,"c":"OCO","l":"EXEC_STARTED","L":"EXECUTING","r":"NONE","C":"secondList","T":1794466395340,"O":[{"s":"BTCUSDT","i":4293166,"c":"secondStopLeg"},{"s":"BTCUSDT","i":4293167,"c":"secondLimitLeg"}]}`, &order.Detail{
			OrderID: "list-2050", ClientOrderID: "secondList", Date: time.UnixMilli(1794466395340), LastUpdated: time.UnixMilli(1794466395340), Pair: currency.NewBTCUSDT(),
		})
		assert.Empty(t, lists.Websocket.DataHandler.C, "each order list should be relayed once")
	})
}
