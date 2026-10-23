package hyperliquid

import (
	"bytes"
	"compress/flate"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"maps"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
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
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var errWebsocketFixture = errors.New("websocket fixture failure")

// websocketConnectionFixture stands in for a websocket connection: it records each request it sends, has the exchange
// handle the frames reply answers a request with as if they were read from the connection, and serves queued messages
// to a reader
type websocketConnectionFixture struct {
	websocket.Connection

	exchange *Exchange

	mu      sync.Mutex
	dialErr error
	sendErr error
	reply   func(request []byte) []string
	sent    []string
	// handleErrs holds the exchange's result of handling each reply frame
	handleErrs []error
	messages   []websocket.Response
	ping       websocket.PingHandler
	match      *websocket.Match
	subs       *subscription.Store

	dialCalls atomic.Int32
	readCalls atomic.Int32
	pingCalls atomic.Int32
}

func (f *websocketConnectionFixture) Dial(context.Context, *gws.Dialer, http.Header, url.Values) error {
	f.dialCalls.Add(1)
	return f.dialErr
}

func (f *websocketConnectionFixture) GetURL() string {
	return "fixture"
}

func (f *websocketConnectionFixture) ReadMessage() websocket.Response {
	f.readCalls.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.messages) == 0 {
		return websocket.Response{}
	}
	message := f.messages[0]
	f.messages = f.messages[1:]
	return message
}

func (f *websocketConnectionFixture) SetupPingHandler(_ request.EndpointLimit, handler websocket.PingHandler) {
	f.pingCalls.Add(1)
	f.mu.Lock()
	f.ping = handler
	f.mu.Unlock()
}

// matcher returns the fixture's response matcher, as a connection has its own under multi-connection management
func (f *websocketConnectionFixture) matcher() *websocket.Match {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.match == nil {
		f.match = websocket.NewMatch()
	}
	return f.match
}

func (f *websocketConnectionFixture) RequireMatchWithData(signature any, incoming []byte) error {
	return f.matcher().RequireMatchWithData(signature, incoming)
}

func (f *websocketConnectionFixture) IncomingWithData(signature any, data []byte) bool {
	return f.matcher().IncomingWithData(signature, data)
}

func (f *websocketConnectionFixture) Subscriptions() *subscription.Store {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.subs == nil {
		f.subs = subscription.NewStore()
	}
	return f.subs
}

// SendJSONMessage records the request and handles its reply frames before returning sendErr, so a reply can settle an
// operation whose send then reports a failure
func (f *websocketConnectionFixture) SendJSONMessage(ctx context.Context, _ request.EndpointLimit, payload any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	f.mu.Lock()
	f.sent = append(f.sent, string(encoded))
	reply, sendErr := f.reply, f.sendErr
	f.mu.Unlock()
	if reply != nil {
		for _, frame := range reply(encoded) {
			handleErr := f.exchange.wsHandleData(ctx, f, []byte(frame))
			f.mu.Lock()
			f.handleErrs = append(f.handleErrs, handleErr)
			f.mu.Unlock()
		}
	}
	return sendErr
}

// SendMessageReturnResponse sends a request and returns the response the match system received while its reply was
// handled, or times out at once, as the reply has then already been handled
func (f *websocketConnectionFixture) SendMessageReturnResponse(ctx context.Context, epl request.EndpointLimit, signature, payload any) ([]byte, error) {
	responses, err := f.matcher().Set(signature, 1)
	if err != nil {
		return nil, err
	}
	if err := f.SendJSONMessage(ctx, epl, payload); err != nil {
		f.matcher().RemoveSignature(signature)
		return nil, err
	}
	select {
	case response := <-responses:
		return response, nil
	default:
		f.matcher().RemoveSignature(signature)
		return nil, websocket.ErrSignatureTimeout
	}
}

func (f *websocketConnectionFixture) sentRequests() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.sent...)
}

func (f *websocketConnectionFixture) handled() []error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]error(nil), f.handleErrs...)
}

// websocketCanonicalDefaults contains the options Hyperliquid adds to the subscriptions it echoes, by subscription type
var websocketCanonicalDefaults = map[string]map[string]any{
	wsChannelOrderbook:          {"nSigFigs": nil, "mantissa": nil, "fast": false},
	wsChannelUserFills:          {"aggregateByTime": false},
	wsChannelSpotState:          {"ignorePortfolioMargin": false},
	wsChannelClearinghouseState: {"dex": ""},
	wsChannelOpenOrders:         {"dex": ""},
	wsChannelTWAPStates:         {"dex": ""},
}

// websocketSubscriptionRequestFields is a subscription request with its subscription kept as fields, so acknowledge can
// echo it with the options Hyperliquid adds
type websocketSubscriptionRequestFields struct {
	Method       string         `json:"method"`
	Subscription map[string]any `json:"subscription"`
}

// acknowledge answers a subscription request as Hyperliquid does, echoing its subscription in the canonical form, with
// the user lower case and the defaults of absent options present; it does not answer other requests
func acknowledge(sent []byte) []string {
	var req websocketSubscriptionRequestFields
	if err := json.Unmarshal(sent, &req); err != nil || req.Subscription == nil {
		return nil
	}
	canonical := req.Subscription
	if subscriptionType, ok := canonical["type"].(string); ok {
		for option, value := range websocketCanonicalDefaults[subscriptionType] {
			if _, ok := canonical[option]; !ok {
				canonical[option] = value
			}
		}
	}
	if user, ok := canonical["user"].(string); ok {
		canonical["user"] = strings.ToLower(user)
	}
	frame, err := json.Marshal(map[string]any{
		"channel": wsChannelSubscriptionResponse,
		"data":    map[string]any{"method": req.Method, "subscription": canonical},
	})
	if err != nil {
		return nil
	}
	return []string{string(frame)}
}

// newWebsocketTestExchange returns a trading server exchange with its markets enabled, the trade and fills feeds
// enabled, account feeds usable, and fixture connections that acknowledge every subscription request, the public one
// carrying WebsocketAPI posts
func newWebsocketTestExchange(t *testing.T) *Exchange {
	t.Helper()
	ex := newTradingServerExchange(t, nil, nil)
	setTestPairs(t, ex, asset.PerpetualContract, testPerpetualMapping, testBuilderMapping)
	setTestPairs(t, ex, asset.Spot, testSpotMapping)
	ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
	ex.SetFillsFeedStatus(true)
	ex.Websocket.Fills.Setup(true, ex.Websocket.DataHandler)
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	ex.websocketPostConnection = publicConn(t, ex)
	return ex
}

// websocketTestConnections holds the fixture connections standing in for each test exchange's public and account
// connections until the test first using them ends; they are not managed, so the subscriptions added on them are held
// in the manager's own store
var websocketTestConnections sync.Map

// websocketTestConnectionPair is a test exchange's public and account fixture connections
type websocketTestConnectionPair struct {
	public, account *websocketConnectionFixture
}

// websocketTestFixtures returns the exchange's fixture connections, which acknowledge every subscription request
func websocketTestFixtures(t *testing.T, ex *Exchange) *websocketTestConnectionPair {
	t.Helper()
	pair, loaded := websocketTestConnections.LoadOrStore(ex, &websocketTestConnectionPair{
		public:  &websocketConnectionFixture{exchange: ex, reply: acknowledge},
		account: &websocketConnectionFixture{exchange: ex, reply: acknowledge},
	})
	if !loaded {
		t.Cleanup(func() { websocketTestConnections.Delete(ex) })
	}
	connections, ok := pair.(*websocketTestConnectionPair)
	require.True(t, ok, "the fixture connections must be a websocketTestConnectionPair")
	return connections
}

// publicConn returns the fixture standing in for the exchange's public connection
func publicConn(t *testing.T, ex *Exchange) *websocketConnectionFixture {
	t.Helper()
	return websocketTestFixtures(t, ex).public
}

// accountConn returns the fixture standing in for the exchange's account connection
func accountConn(t *testing.T, ex *Exchange) *websocketConnectionFixture {
	t.Helper()
	return websocketTestFixtures(t, ex).account
}

// websocketTestConn returns the exchange's account fixture connection when account is set, else its public one
func websocketTestConn(t *testing.T, ex *Exchange, account bool) *websocketConnectionFixture {
	t.Helper()
	if account {
		return accountConn(t, ex)
	}
	return publicConn(t, ex)
}

// subscribeWebsocketTest subscribes to feeds as the manager does: expanded and keyed, then each on the connection its
// account flag selects
func subscribeWebsocketTest(t *testing.T, ex *Exchange, subs subscription.List) error {
	t.Helper()
	if slices.Contains(subs, nil) {
		return common.ErrNilPointer
	}
	expanded, errs := subs.ExpandTemplates(ex)
	for _, sub := range expanded {
		errs = common.AppendError(errs, ex.websocketSubscribe(t.Context(), websocketTestConn(t, ex, sub.Authenticated), subscription.List{sub}))
	}
	return errs
}

// unsubscribeWebsocketTest unsubscribes from the held variants of feeds, each on the connection its account flag selects
func unsubscribeWebsocketTest(t *testing.T, ex *Exchange, subs subscription.List) error {
	t.Helper()
	if slices.Contains(subs, nil) {
		return common.ErrNilPointer
	}
	expanded, errs := subs.ExpandTemplates(websocketTemplateExpander{ex})
	for _, sub := range expanded {
		feed := newWebsocketSubscriptionKey(sub)
		held := ex.Websocket.GetSubscription(feed)
		if held == nil {
			errs = common.AppendError(errs, fmt.Errorf("%w: %s", subscription.ErrNotFound, feed))
			continue
		}
		errs = common.AppendError(errs, ex.websocketUnsubscribe(t.Context(), websocketTestConn(t, ex, held.Authenticated), subscription.List{held}))
	}
	return errs
}

// receiveWebsocketData returns the next relayed message; handlers relay synchronously, so it must already be waiting
func receiveWebsocketData(t *testing.T, ex *Exchange) any {
	t.Helper()
	select {
	case payload := <-ex.Websocket.DataHandler.C:
		return payload.Data
	default:
		require.FailNow(t, "a relayed message must be waiting")
		return nil
	}
}

// receivePendingResult returns the result a settled operation sent; settling sends it before returning, so the test
// fails at once rather than waiting when none was sent
func receivePendingResult(t *testing.T, pending *websocketPendingOperation) error {
	t.Helper()
	select {
	case err := <-pending.done:
		return err
	default:
		require.FailNow(t, "a settled operation's result must be waiting")
		return nil
	}
}

// receivePostResponse returns the response delivered to a post request, failing the test at once when none was
// delivered
func receivePostResponse(t *testing.T, responses <-chan []byte) []byte {
	t.Helper()
	select {
	case response := <-responses:
		return response
	default:
		require.FailNow(t, "a post response must be waiting")
		return nil
	}
}

// assertNoWebsocketData checks nothing more was relayed
func assertNoWebsocketData(t *testing.T, ex *Exchange) {
	t.Helper()
	select {
	case payload := <-ex.Websocket.DataHandler.C:
		assert.Failf(t, "nothing should be relayed", "relayed %T: %v", payload.Data, payload.Data)
	default:
	}
}

// addWebsocketPending prepares a subscription as manageWebsocketSubscription does before sending its request, and
// registers the operation awaiting its acknowledgement under key, on the public fixture connection unless key names one
func addWebsocketPending(t *testing.T, ex *Exchange, method string, sub *subscription.Subscription, key *websocketPendingKey) *websocketPendingOperation {
	t.Helper()
	if key.connection == nil {
		key.connection = publicConn(t, ex)
	}
	connection := key.connection
	previousState := sub.State()
	if method == wsMethodSubscribe {
		require.NoError(t, ex.Websocket.AddSubscriptions(connection, sub), "AddSubscriptions must not error")
	} else {
		require.NoError(t, sub.SetState(subscription.UnsubscribingState), "SetState must not error")
	}
	pending := &websocketPendingOperation{
		method:        method,
		connection:    connection,
		subscription:  sub,
		previousState: previousState,
		done:          make(chan error, 1),
	}
	registerWebsocketPending(ex, key, pending)
	return pending
}

// registerWebsocketPending registers an operation under key, on the operation's connection unless key names one, without
// preparing its subscription
func registerWebsocketPending(ex *Exchange, key *websocketPendingKey, pending *websocketPendingOperation) {
	if key.connection == nil {
		key.connection = pending.connection
	}
	ex.websocketPendingMu.Lock()
	if ex.websocketPending == nil {
		ex.websocketPending = make(map[websocketPendingKey]*websocketPendingOperation)
	}
	ex.websocketPending[*key] = pending
	ex.websocketPendingMu.Unlock()
}

// pendingWebsocketOperation returns the operation pending under key, if any
func pendingWebsocketOperation(ex *Exchange, key *websocketPendingKey) *websocketPendingOperation {
	ex.websocketPendingMu.Lock()
	defer ex.websocketPendingMu.Unlock()
	return ex.websocketPending[*key]
}

// newSubscribedWebsocketSubscription returns a subscription stored as subscribed on the public connection
func newSubscribedWebsocketSubscription(t *testing.T, ex *Exchange, sub *subscription.Subscription) *subscription.Subscription {
	t.Helper()
	require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(publicConn(t, ex), sub), "AddSuccessfulSubscriptions must not error")
	return sub
}

// deflateBase64 compresses data with raw DEFLATE and encodes it in base64, as the fastAssetCtxs feed sends it
func deflateBase64(t *testing.T, data []byte) string {
	t.Helper()
	var compressed bytes.Buffer
	writer, err := flate.NewWriter(&compressed, flate.BestCompression)
	require.NoError(t, err, "flate.NewWriter must not error")
	_, err = writer.Write(data)
	require.NoError(t, err, "Write must not error")
	require.NoError(t, writer.Close(), "Close must not error")
	return base64.StdEncoding.EncodeToString(compressed.Bytes())
}

// websocketVenueRequests records the requests other than pings that the mock of Hyperliquid's websocket receives; the
// live venue records nothing
type websocketVenueRequests struct {
	mu       sync.Mutex
	requests []string
}

// received returns the recorded requests in arrival order
func (v *websocketVenueRequests) received() []string {
	v.mu.Lock()
	defer v.mu.Unlock()
	return slices.Clone(v.requests)
}

// record keeps a request the mock received, without the newline the connection's JSON encoder ends it with
func (v *websocketVenueRequests) record(msg []byte) {
	v.mu.Lock()
	v.requests = append(v.requests, strings.TrimSpace(string(msg)))
	v.mu.Unlock()
}

// acknowledgeSubscriptions answers a request as Hyperliquid does on a connection used only for subscriptions: a pong to
// a ping and the canonical acknowledgement to a subscription request; it records every request but pings
func (v *websocketVenueRequests) acknowledgeSubscriptions(_ testing.TB, msg []byte, conn *gws.Conn) error {
	var req websocketVenueRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return err
	}
	if req.Method == "ping" {
		return conn.WriteMessage(gws.TextMessage, []byte(`{"channel":"pong"}`))
	}
	v.record(msg)
	frames := acknowledge(msg)
	if len(frames) == 0 {
		return fmt.Errorf("%w: unanswered request %s", errWebsocketFixture, msg)
	}
	for _, frame := range frames {
		if err := conn.WriteMessage(gws.TextMessage, []byte(frame)); err != nil {
			return err
		}
	}
	return nil
}

// newLocalWebsocketTestExchange returns an exchange named after the test that watches the test account, with its
// websocket pointed at a new local server acknowledging each subscription request as Hyperliquid does, and the requests
// the server receives; it subscribes the configured subscriptions when enabled. Manager flows run against it in both test
// modes, as they would disturb the connection the live tests share
func newLocalWebsocketTestExchange(t *testing.T, configured subscription.List) (*Exchange, *websocketVenueRequests) {
	t.Helper()
	venue := new(websocketVenueRequests)
	server := httptest.NewTestServer(t, mockws.CurryWsMockUpgrader(t, venue.acknowledgeSubscriptions))
	server.Start()
	ex := newTestExchange(t)
	ex.API.AuthenticatedWebsocketSupport = true
	setTestCredentials(ex, &accounts.Credentials{Key: testAccountAddress})
	require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(server.URL, "http")), "SetAllConnectionURLs must not error")
	ex.Features.Subscriptions = configured
	t.Cleanup(func() {
		if !ex.Websocket.IsEnabled() {
			return
		}
		// Disabling stops the connection monitor, which would otherwise redial the server after the test
		assert.NoError(t, ex.Websocket.Disable(), "Disable should not error")
		if err := shutdownWebsocketWithin(t, ex); err != nil {
			assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should only fail when the connection monitor has shut the connection down first")
		}
	})
	return ex, venue
}

// shutdownWebsocketWithin shuts the websocket down, failing the test once Shutdown has not returned within five seconds:
// the connection monitor holds the manager's lock while it waits for a lost connection's reader, so a reader that never
// reports it has stopped would otherwise block the test's cleanup until go test's timeout
func shutdownWebsocketWithin(t *testing.T, ex *Exchange) error {
	t.Helper()
	return websocketCallWithin(t, "Shutdown", ex.Websocket.Shutdown)
}

// websocketCallWithin returns what call returns, failing the test once call has not returned within five seconds: a call
// that retires a connection waits for its reader while holding the manager's lock, which every later call needing the
// lock then waits for, so a reader that never reports it has stopped would otherwise block the test until go test's
// timeout
func websocketCallWithin(t *testing.T, name string, call func() error) error {
	t.Helper()
	result := make(chan error, 1)
	go func() { result <- call() }()
	select {
	case err := <-result:
		return err
	case <-time.After(5 * time.Second):
		assert.Fail(t, name+" should return within five seconds")
		return nil
	}
}

// websocketVenueRequest is a request as Hyperliquid's websocket receives it: a ping, a subscription request or a post
type websocketVenueRequest struct {
	Method  string                 `json:"method"`
	ID      uint64                 `json:"id"`
	Request websocketVenuePostBody `json:"request"`
}

// websocketVenuePostBody is a post's request type and its payload, kept as JSON
type websocketVenuePostBody struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// assertWebsocketActionPost checks one request was sent, a post of the signed action with a nonce and a signature
func assertWebsocketActionPost(t *testing.T, received []string, action string) {
	t.Helper()
	require.Len(t, received, 1, "one request must be sent")
	var post websocketVenueRequest
	require.NoError(t, json.Unmarshal([]byte(received[0]), &post), "the request must decode")
	assert.Equal(t, wsMethodPost, post.Method, "the request should be a post")
	assert.NotZero(t, post.ID, "the post should carry an ID")
	assert.Equal(t, wsPostTypeAction, post.Request.Type, "the post should carry an action")
	var signed SignedActionRequest
	require.NoError(t, json.Unmarshal(post.Request.Payload, &signed), "the signed action must decode")
	encoded, err := json.Marshal(signed.Action)
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, action, string(encoded), "the post should carry the action")
	assert.NotZero(t, signed.Nonce, "the action should carry a nonce")
	assert.NotEmpty(t, signed.Signature.R, "the action should carry a signature's r")
	assert.NotEmpty(t, signed.Signature.S, "the action should carry a signature's s")
	assert.Contains(t, []uint8{27, 28}, signed.Signature.V, "the action should carry a signature's v")
}

// websocketTestOrder returns the order the websocket action tests place: the fixtures' BTC buy in mock mode, and live
// liveRestingOrder, which is cancelled when the test ends
func websocketTestOrder(t *testing.T) OrderRequest {
	t.Helper()
	if mockTests {
		return OrderRequest{Asset: 0, IsBuy: true, Price: 85000, Size: 0.001, Limit: &LimitOrderType{TimeInForce: TimeInForceGTC}, ClientOrderID: testClientOrderID}
	}
	skipLiveWithoutPrivateKey(t)
	return liveRestingOrder(t)
}

// websocketTestRestingOrder returns the order the websocket cancel and modify tests act on and its order ID: the
// fixtures' order in mock mode, and live a new resting order placed through the exchange endpoint
func websocketTestRestingOrder(t *testing.T) (placed OrderRequest, orderID uint64) {
	t.Helper()
	if mockTests {
		return websocketTestOrder(t), testOrderID
	}
	skipLiveWithoutPrivateKey(t)
	return placeLiveRestingOrder(t)
}

func TestConnectWebsocket(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	disabled := new(websocketConnectionFixture)
	require.ErrorIs(t, ex.connectWebsocket(t.Context(), disabled), websocket.ErrWebsocketNotEnabled, "connectWebsocket must refuse a disabled exchange")
	assert.Zero(t, disabled.dialCalls.Load(), "connectWebsocket should not dial for a disabled exchange")
	ex.SetEnabled(true)
	require.ErrorIs(t, ex.connectWebsocket(t.Context(), nil), common.ErrNilPointer, "connectWebsocket must reject a nil connection")

	failed := &websocketConnectionFixture{dialErr: errWebsocketFixture}
	err := ex.connectWebsocket(t.Context(), failed)
	require.ErrorIs(t, err, errWebsocketFixture, "connectWebsocket must return the dial error")
	assert.ErrorContains(t, err, "dialling fixture", "connectWebsocket should name the URL it could not dial")
	assert.Zero(t, failed.pingCalls.Load(), "connectWebsocket should not set up pings after a failed dial")

	connection := new(websocketConnectionFixture)
	require.NoError(t, ex.connectWebsocket(t.Context(), connection), "connectWebsocket must not error")
	assert.Equal(t, int32(1), connection.dialCalls.Load(), "connectWebsocket should dial once")
	exp := websocket.PingHandler{MessageType: gws.TextMessage, Message: []byte(`{"method":"ping"}`), Delay: websocketPingInterval}
	assert.Equal(t, exp, connection.ping, "connectWebsocket should send application pings")

	venue := new(Exchange)
	require.NoError(t, testexch.Setup(venue), "Setup must not error")
	require.NoError(t, venue.Websocket.SetAllConnectionURLs(venueWebsocketURL(t)), "SetAllConnectionURLs must not error")
	dialled, err := venue.Websocket.CreateTestConnection(websocketPublicConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	require.NoError(t, venue.connectWebsocket(t.Context(), dialled), "connectWebsocket must dial the venue")
	assert.NoError(t, dialled.Shutdown(), "Shutdown should close the venue connection and stop its pings")
}

func TestConnectPublicWebsocket(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	failed := &websocketConnectionFixture{exchange: ex, dialErr: errWebsocketFixture}
	require.ErrorIs(t, ex.connectPublicWebsocket(t.Context(), failed), errWebsocketFixture, "connectPublicWebsocket must return the dial error")
	assert.Nil(t, ex.websocketPostConnection, "connectPublicWebsocket should not post on a connection it could not dial")

	connection := &websocketConnectionFixture{exchange: ex}
	require.NoError(t, ex.connectPublicWebsocket(t.Context(), connection), "connectPublicWebsocket must not error")
	assert.Equal(t, int32(1), connection.pingCalls.Load(), "connectPublicWebsocket should send application pings")
	assert.Equal(t, websocket.Connection(connection), ex.websocketPostConnection, "connectPublicWebsocket should post on the connection it dialled")
}

func TestConnectAccountWebsocket(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	failed := &websocketConnectionFixture{exchange: ex, dialErr: errWebsocketFixture}
	require.ErrorIs(t, ex.connectAccountWebsocket(t.Context(), failed), errWebsocketFixture, "connectAccountWebsocket must return the dial error")
	assert.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "connectAccountWebsocket should not make account feeds usable without a connection")
	assert.Nil(t, ex.websocketAccountConnection, "connectAccountWebsocket should not hold a connection it could not dial")

	connection := &websocketConnectionFixture{exchange: ex}
	require.NoError(t, ex.connectAccountWebsocket(t.Context(), connection), "connectAccountWebsocket must not error")
	assert.Equal(t, int32(1), connection.pingCalls.Load(), "connectAccountWebsocket should send application pings")
	assert.True(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "connectAccountWebsocket should make account feeds usable")
	assert.Equal(t, websocket.Connection(connection), ex.websocketAccountConnection, "connectAccountWebsocket should hold the connection it dialled")
}

func TestPublicWebsocketDisconnected(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	current := publicConn(t, ex)
	sub := &subscription.Subscription{Channel: wsChannelAllMids, QualifiedChannel: wsChannelAllMids}
	key := websocketPendingKey{connection: current, subscription: WebsocketSubscription{Type: wsChannelAllMids}}
	pending := addWebsocketPending(t, ex, wsMethodSubscribe, sub, &key)
	ex.expectTradeReplay("BTC")
	record := ex.websocketWatermarks.orders.touch(1, time.Now())
	record.covered = true

	ex.publicWebsocketDisconnected(t.Context(), &websocketConnectionFixture{exchange: ex})
	assert.Same(t, pending, pendingWebsocketOperation(ex, &key), "publicWebsocketDisconnected should leave the current connection's pending operations")
	assert.Equal(t, map[string]struct{}{"BTC": {}}, ex.websocketTradeReplays, "publicWebsocketDisconnected should keep the current connection's awaited replays when an older connection stops")
	assert.Equal(t, websocket.Connection(current), ex.websocketPostConnection, "publicWebsocketDisconnected should keep posting on the current connection when an older one stops")

	ex.publicWebsocketDisconnected(t.Context(), current)
	require.ErrorIs(t, receivePendingResult(t, pending), websocket.ErrNotConnected, "publicWebsocketDisconnected must fail the connection's pending operations")
	assert.Nil(t, ex.Websocket.GetSubscription(sub), "publicWebsocketDisconnected should roll back the connection's pending subscribe")
	assert.Empty(t, ex.websocketTradeReplays, "publicWebsocketDisconnected should stop awaiting the replays of the closed connection")
	assert.Nil(t, ex.websocketPostConnection, "publicWebsocketDisconnected should stop posting on the closed connection")
	assert.True(t, record.covered, "publicWebsocketDisconnected should keep the orders' fills covered")
}

func TestAccountWebsocketDisconnected(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	current := accountConn(t, ex)
	require.NoError(t, ex.connectAccountWebsocket(t.Context(), current), "connectAccountWebsocket must not error")
	sub := &subscription.Subscription{Channel: wsChannelNotification, Authenticated: true, QualifiedChannel: wsChannelNotification}
	key := websocketPendingKey{connection: current, subscription: WebsocketSubscription{Type: wsChannelNotification, User: testAccountAddress}}
	pending := addWebsocketPending(t, ex, wsMethodSubscribe, sub, &key)
	record := ex.websocketWatermarks.orders.touch(1, time.Now())
	record.covered = true

	ex.accountWebsocketDisconnected(t.Context(), &websocketConnectionFixture{exchange: ex})
	assert.Same(t, pending, pendingWebsocketOperation(ex, &key), "accountWebsocketDisconnected should leave the current connection's pending operations")
	assert.True(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "accountWebsocketDisconnected should leave account feeds usable when an older connection stops")
	assert.True(t, record.covered, "accountWebsocketDisconnected should keep the orders' fills covered when an older connection stops")

	ex.accountWebsocketDisconnected(t.Context(), current)
	require.ErrorIs(t, receivePendingResult(t, pending), websocket.ErrNotConnected, "accountWebsocketDisconnected must fail the connection's pending operations")
	assert.Nil(t, ex.Websocket.GetSubscription(sub), "accountWebsocketDisconnected should roll back the connection's pending subscribe")
	assert.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "accountWebsocketDisconnected should stop account feeds when the account connection stops")
	assert.Nil(t, ex.websocketAccountConnection, "accountWebsocketDisconnected should forget the closed connection")
	assert.False(t, record.covered, "accountWebsocketDisconnected should mark the orders' fills incomplete")
}

// dropWebsocketAccountConnection connects the exchange's account fixture connection and drops it
func dropWebsocketAccountConnection(t *testing.T, ex *Exchange) {
	t.Helper()
	require.NoError(t, ex.connectAccountWebsocket(t.Context(), accountConn(t, ex)), "connectAccountWebsocket must not error")
	ex.accountWebsocketDisconnected(t.Context(), accountConn(t, ex))
}

// newAccountRecoveryTestExchange returns an exchange watching the test account and subscribing its orders, whose public
// and account connections reach one local server that refuses the upgrades refuse reports, counted from 1, and when the
// server was asked for each; the websocket is disabled and shut down when the test ends
func newAccountRecoveryTestExchange(t *testing.T, refuse func(attempt int) bool, monitorDelay time.Duration) (ex *Exchange, attempts func() []time.Time) {
	t.Helper()
	venue := new(websocketVenueRequests)
	var mu sync.Mutex
	var times []time.Time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		times = append(times, time.Now())
		attempt := len(times)
		mu.Unlock()
		if refuse(attempt) {
			http.Error(w, "refused", http.StatusServiceUnavailable)
			return
		}
		mockws.WsMockUpgrader(t, w, r, func(_ testing.TB, msg []byte, conn *gws.Conn) error {
			if string(msg) == "drop" {
				return conn.Close()
			}
			return venue.acknowledgeSubscriptions(t, msg, conn)
		})
	}))
	t.Cleanup(server.Close)
	ex = new(Exchange)
	ex.SetDefaults()
	ex.Name = t.Name()
	cfg, err := ex.GetStandardConfig()
	require.NoError(t, err, "GetStandardConfig must not error")
	cfg.Features = &config.FeaturesConfig{Enabled: config.FeaturesEnabledConfig{Websocket: true}}
	cfg.ConnectionMonitorDelay, cfg.WebsocketTrafficTimeout = monitorDelay, time.Hour
	require.NoError(t, ex.Setup(cfg), "Setup must not error")
	ex.API.AuthenticatedWebsocketSupport = true
	setTestCredentials(ex, &accounts.Credentials{Key: testAccountAddress})
	ex.Features.Subscriptions = subscription.List{{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true}}
	require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(server.URL, "http")), "SetAllConnectionURLs must not error")
	t.Cleanup(func() {
		// Disabling stops the connection monitor, which may then shut the connection down first
		assert.NoError(t, ex.Websocket.Disable(), "Disable should not error")
		if err := shutdownWebsocketWithin(t, ex); err != nil {
			assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should only fail when the websocket is already shut down")
		}
	})
	return ex, func() []time.Time {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(times)
	}
}

func TestWebsocketAccountConnectionRecovery(t *testing.T) {
	t.Parallel()
	// The public connection dials first and is accepted; the account connection's first dial and first retry are refused
	ex, attempts := newAccountRecoveryTestExchange(t, func(attempt int) bool { return attempt == 2 || attempt == 3 }, 20*time.Millisecond)
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must keep the public connection when the account connection fails")
	assert.True(t, ex.Websocket.IsConnected(), "Connect should leave the public connection up")
	assert.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "Connect should not make account feeds usable while the account connection is down")
	// The connector makes account feeds usable before the feeds are subscribed
	accountFeed := newWebsocketSubscriptionKey(&subscription.Subscription{Channel: subscription.MyOrdersChannel, Authenticated: true})
	require.Eventually(t, func() bool {
		sub := ex.Websocket.GetSubscription(accountFeed)
		return sub != nil && sub.State() == subscription.SubscribedState
	}, 10*time.Second, 5*time.Millisecond, "the manager must connect the account connection and subscribe the configured account feed once its endpoint accepts it")
	assert.True(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "the account connection should make account feeds usable")
	upgrades := attempts()
	require.Len(t, upgrades, 4, "the server must see the public dial, the refused account dial, a refused retry and the accepted retry")
	// Retries wait the monitor delay, 20ms, after the first failure and double after each further one, as timers never
	// fire early
	assert.GreaterOrEqual(t, upgrades[3].Sub(upgrades[2]), 40*time.Millisecond, "the second retry should wait double the first")
}

// TestWebsocketConnectionLoss has the venue drop each connection: losing the account connection fails its pending
// subscription operations and replaces it alone, and losing the public connection fails its own and reconnects
func TestWebsocketConnectionLoss(t *testing.T) {
	t.Parallel()
	ex, attempts := newAccountRecoveryTestExchange(t, func(int) bool { return false }, 20*time.Millisecond)
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	ex.websocketConnectionsMu.Lock()
	public, account := ex.websocketPostConnection, ex.websocketAccountConnection
	ex.websocketConnectionsMu.Unlock()
	require.NotNil(t, public, "Connect must connect the public connection")
	require.NotNil(t, account, "Connect must connect the account connection")
	publicKey := websocketPendingKey{connection: public, subscription: WebsocketSubscription{Type: wsChannelAllMids}}
	publicPending := addWebsocketPending(t, ex, wsMethodSubscribe, &subscription.Subscription{Channel: wsChannelAllMids, QualifiedChannel: wsChannelAllMids}, &publicKey)
	accountKey := websocketPendingKey{connection: account, subscription: WebsocketSubscription{Type: wsChannelNotification, User: testAccountAddress}}
	accountPending := addWebsocketPending(t, ex, wsMethodSubscribe, &subscription.Subscription{Channel: wsChannelNotification, Authenticated: true, QualifiedChannel: wsChannelNotification}, &accountKey)
	settled := func(pending *websocketPendingOperation) error {
		select {
		case err := <-pending.done:
			return err
		case <-time.After(5 * time.Second):
			require.FailNow(t, "a pending operation on a lost connection must settle")
			return nil
		}
	}

	require.NoError(t, account.SendRawMessage(t.Context(), request.Unset, gws.TextMessage, []byte("drop")), "SendRawMessage must not error")
	assert.ErrorIs(t, settled(accountPending), websocket.ErrNotConnected, "losing the account connection should fail its pending operations")
	require.Eventually(t, func() bool {
		ex.websocketConnectionsMu.Lock()
		defer ex.websocketConnectionsMu.Unlock()
		return ex.websocketAccountConnection != nil && ex.websocketAccountConnection != account
	}, 10*time.Second, 5*time.Millisecond, "the manager must replace a lost account connection")
	ex.websocketConnectionsMu.Lock()
	assert.Equal(t, public, ex.websocketPostConnection, "losing the account connection should leave the public connection up")
	ex.websocketConnectionsMu.Unlock()
	assert.Same(t, publicPending, pendingWebsocketOperation(ex, &publicKey), "losing the account connection should leave the public connection's pending operations")
	assert.Len(t, attempts(), 3, "the server should see the public dial, the account dial and the account's replacement")

	require.NoError(t, public.SendRawMessage(t.Context(), request.Unset, gws.TextMessage, []byte("drop")), "SendRawMessage must not error")
	assert.ErrorIs(t, settled(publicPending), websocket.ErrNotConnected, "losing the public connection should fail its pending operations")
	assert.Eventually(t, func() bool {
		ex.websocketConnectionsMu.Lock()
		defer ex.websocketConnectionsMu.Unlock()
		return ex.websocketPostConnection != nil && ex.websocketPostConnection != public
	}, 10*time.Second, 5*time.Millisecond, "the manager should reconnect once the public connection is lost")
}

func TestWebsocketHandleData(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(websocketConnectionEstablished)), "wsHandleData should ignore the connection greeting")
	assert.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"pong"}`)), "wsHandleData should ignore a pong")
	assert.ErrorContains(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`invalid`)), ex.Name+" decoding websocket message", "wsHandleData should name the exchange of a message it cannot decode")
	err := ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"l2Book","data":{"coin":"ETH","time":1791278882097,"levels":[[],[]]}}`))
	assert.ErrorIs(t, err, errPairMappingNotFound, "wsHandleData should return the handler's error")
	assert.ErrorContains(t, err, ex.Name+" l2Book message", "wsHandleData should name the exchange and channel of a message that fails")
	err = ex.wsHandleData(t.Context(), publicConn(t, ex), websocketErrorFrame(t, "Subscription limit reached"))
	assert.ErrorIs(t, err, errWebsocketServer, "wsHandleData should return the server error")
	assert.ErrorContains(t, err, ex.Name+" error message", "wsHandleData should identify the server error channel")
	assert.ErrorContains(t, err, "Subscription limit reached", "wsHandleData should retain the server's message")
	assertNoWebsocketData(t, ex)
}

func TestWebsocketHandleChannel(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.NoError(t, ex.wsHandleChannel(t.Context(), publicConn(t, ex), &WebsocketMessage{Channel: wsChannelPong}, []byte(`{"channel":"pong"}`)), "wsHandleChannel should ignore a pong")
	for _, channel := range []string{
		wsChannelSubscriptionResponse, wsChannelPost, wsChannelError, wsChannelAllMids, wsChannelNotification, wsChannelWebData3,
		wsChannelTWAPStates, wsChannelClearinghouseState, wsChannelOpenOrders, wsChannelCandle, wsChannelOrderbook, wsChannelTrades,
		wsChannelOrderUpdates, wsChannelUserEvents, wsChannelUserFills, wsChannelUserFundings, wsChannelUserNonFundingLedgerUpdates,
		wsChannelActiveAssetContext, wsChannelActiveSpotAssetContext, wsChannelActiveAssetData, wsChannelUserTWAPSliceFills,
		wsChannelUserTWAPHistory, wsChannelBestBidOffer, wsChannelSpotState, wsChannelAllDEXsClearinghouseState,
		wsChannelAllDEXsAssetContexts, wsChannelOutcomeMetaUpdates, wsChannelFastAssetContexts,
	} {
		raw := []byte(`{"channel":"` + channel + `","data":"invalid!"}`)
		assert.Errorf(t, ex.wsHandleChannel(t.Context(), publicConn(t, ex), &WebsocketMessage{Channel: channel, Data: json.RawMessage(`"invalid!"`)}, raw), "wsHandleChannel should pass %s data to its handler", channel)
	}
	assertNoWebsocketData(t, ex)

	unhandled := `{"channel":"futureChannel","data":{}}`
	require.NoError(t, ex.wsHandleChannel(t.Context(), publicConn(t, ex), &WebsocketMessage{Channel: "futureChannel", Data: json.RawMessage(`{}`)}, []byte(unhandled)), "wsHandleChannel must not error for an unhandled channel")
	exp := websocket.UnhandledMessageWarning{Message: ex.Name + websocket.UnhandledMessage + unhandled}
	assert.Equal(t, exp, receiveWebsocketData(t, ex), "wsHandleChannel should relay a warning for an unhandled channel")
}

// websocketClearinghouseStateJSON is the clearinghouse state of the websocket fixtures' vault on the default DEX
const websocketClearinghouseStateJSON = `{"marginSummary":{"accountValue":"3155203.7660619998","totalNtlPos":"4345291.8850560002","totalRawUsd":"3182148.2545679999","totalMarginUsed":"217264.594197"},"crossMarginSummary":{"accountValue":"3155203.7660619998","totalNtlPos":"4345291.8850560002","totalRawUsd":"3182148.2545679999","totalMarginUsed":"217264.594197"},"crossMaintenanceMarginUsed":"43452.918775","withdrawable":"2720674.5776","assetPositions":[{"type":"oneWay","position":{"coin":"SAGA","szi":"-103774.5","leverage":{"type":"cross","value":20},"entryPx":"0.01662","positionValue":"2527.94682","unrealizedPnl":"-802.226337","returnOnEquity":"-9.2972917098","liquidationPx":"29.713170189","marginUsed":"126.397341","maxLeverage":50,"cumFunding":{"allTime":"-3524.80395","sinceOpen":"81.316838","sinceChange":"114.927323"}}},{"type":"oneWay","position":{"coin":"SOL","szi":"-212.78","leverage":{"type":"isolated","value":10,"rawUsd":"27985.6"},"entryPx":"119.51","positionValue":"25429.33","unrealizedPnl":"-0.21","returnOnEquity":"-0.0000826","liquidationPx":"130.2","marginUsed":"2542.93","maxLeverage":20,"cumFunding":{"allTime":"-12.75","sinceOpen":"-0.32","sinceChange":"-0.32"}}}],"time":1791279140401}`

// websocketEmptyClearinghouseStateJSON is the clearinghouse state of the websocket fixtures' vault on the xyz DEX
const websocketEmptyClearinghouseStateJSON = `{"marginSummary":{"accountValue":"0.0","totalNtlPos":"0.0","totalRawUsd":"0.0","totalMarginUsed":"0.0"},"crossMarginSummary":{"accountValue":"0.0","totalNtlPos":"0.0","totalRawUsd":"0.0","totalMarginUsed":"0.0"},"crossMaintenanceMarginUsed":"0.0","withdrawable":"0.0","assetPositions":[],"time":1791280776159}`

// testWebsocketClearinghouseState is websocketClearinghouseStateJSON decoded
func testWebsocketClearinghouseState() ClearinghouseStateResponse {
	summary := MarginSummary{AccountValue: 3155203.7660619998, TotalNotionalPosition: 4345291.8850560002, TotalRawUSD: 3182148.2545679999, TotalMarginUsed: 217264.594197}
	return ClearinghouseStateResponse{
		MarginSummary:              summary,
		CrossMarginSummary:         summary,
		CrossMaintenanceMarginUsed: 43452.918775,
		Withdrawable:               2720674.5776,
		AssetPositions: []AssetPosition{
			{Type: "oneWay", Position: Position{
				Coin:              "SAGA",
				SignedSize:        -103774.5,
				Leverage:          Leverage{Type: "cross", Value: 20},
				EntryPrice:        0.01662,
				PositionValue:     2527.94682,
				UnrealisedPNL:     -802.226337,
				ReturnOnEquity:    -9.2972917098,
				LiquidationPrice:  29.713170189,
				MarginUsed:        126.397341,
				MaxLeverage:       50,
				CumulativeFunding: CumulativeFunding{AllTime: -3524.80395, SinceOpen: 81.316838, SinceChange: 114.927323},
			}},
			{Type: "oneWay", Position: Position{
				Coin:              "SOL",
				SignedSize:        -212.78,
				Leverage:          Leverage{Type: "isolated", Value: 10, RawUSD: 27985.6},
				EntryPrice:        119.51,
				PositionValue:     25429.33,
				UnrealisedPNL:     -0.21,
				ReturnOnEquity:    -0.0000826,
				LiquidationPrice:  130.2,
				MarginUsed:        2542.93,
				MaxLeverage:       20,
				CumulativeFunding: CumulativeFunding{AllTime: -12.75, SinceOpen: -0.32, SinceChange: -0.32},
			}},
		},
		Time: milli(1791279140401),
	}
}

// testWebsocketEmptyClearinghouseState is websocketEmptyClearinghouseStateJSON decoded
func testWebsocketEmptyClearinghouseState() ClearinghouseStateResponse {
	return ClearinghouseStateResponse{AssetPositions: []AssetPosition{}, Time: milli(1791280776159)}
}

func TestWebsocketRelay(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.Error(t, wsRelay[WebsocketAllMids](t.Context(), ex, []byte(`"invalid"`)), "wsRelay should reject invalid data")
	for _, tc := range []struct {
		name  string
		frame string
		exp   any
	}{
		{
			name:  "allMids",
			frame: `{"channel":"allMids","data":{"mids":{"BTC":"85801.5","PURR/USDC":"0.14988","@107":"92.9315","#14880":"0.111485"}}}`,
			exp:   &WebsocketAllMids{MidPrices: map[string]types.Number{"BTC": 85801.5, "PURR/USDC": 0.14988, "@107": 92.9315, "#14880": 0.111485}},
		},
		{
			name:  "builder DEX allMids",
			frame: `{"channel":"allMids","data":{"dex":"xyz","mids":{"xyz:XYZ100":"31169.5"}}}`,
			exp:   &WebsocketAllMids{DEX: "xyz", MidPrices: map[string]types.Number{"xyz:XYZ100": 31169.5}},
		},
		{
			name:  "notification",
			frame: `{"channel":"notification","data":{"notification":"Resting order filled: Bought 1017.6 ARB at $0.20138"}}`,
			exp:   &WebsocketNotification{Notification: "Resting order filled: Bought 1017.6 ARB at $0.20138"},
		},
		{
			name:  "webData3",
			frame: `{"channel":"webData3","data":{"userState":{"agentAddress":"0x0000000000000000000000000000000000000008","agentValidUntil":1792311392771,"cumLedger":"-2686.97","serverTime":1791280777681,"isVault":false,"user":"0x0000000000000000000000000000000000000003","dexAbstractionEnabled":true,"abstraction":"unifiedAccount","optOutOfSpotDusting":true},"perpDexStates":[{"totalVaultEquity":"1250000.5","perpsAtOpenInterestCap":["CANTO","FTM"]},{"totalVaultEquity":"15.25","perpsAtOpenInterestCap":["xyz:XYZ100"],"leadingVaults":[{"address":"0x0000000000000000000000000000000000000009","name":"Hyperliquidity Provider (HLP)"}]}]}}`,
			exp: &WebsocketWebData3{
				UserState: WebsocketWebData3UserState{
					AgentAddress:          "0x0000000000000000000000000000000000000008",
					AgentValidUntil:       milli(1792311392771),
					CumulativeLedger:      -2686.97,
					ServerTime:            milli(1791280777681),
					User:                  "0x0000000000000000000000000000000000000003",
					OptOutOfSpotDusting:   true,
					DEXAbstractionEnabled: true,
					Abstraction:           AccountAbstractionUnifiedAccount,
				},
				PerpetualDEXStates: []WebsocketPerpetualDEXState{
					{TotalVaultEquity: 1250000.5, PerpetualsAtOpenInterestCap: []string{"CANTO", "FTM"}},
					{
						TotalVaultEquity:            15.25,
						PerpetualsAtOpenInterestCap: []string{"xyz:XYZ100"},
						LeadingVaults:               []WebsocketLeadingVault{{Address: "0x0000000000000000000000000000000000000009", Name: "Hyperliquidity Provider (HLP)"}},
					},
				},
			},
		},
		{
			name:  "vault webData3",
			frame: `{"channel":"webData3","data":{"userState":{"agentAddress":null,"agentValidUntil":null,"cumLedger":"-1190433.52","serverTime":1791280775688,"isVault":true,"user":"0x0000000000000000000000000000000000000001"},"perpDexStates":[{"totalVaultEquity":"0.0","perpsAtOpenInterestCap":["CANTO","FTM"]},{"totalVaultEquity":"0.0"}]}}`,
			exp: &WebsocketWebData3{
				UserState: WebsocketWebData3UserState{
					CumulativeLedger: -1190433.52,
					ServerTime:       milli(1791280775688),
					IsVault:          true,
					User:             "0x0000000000000000000000000000000000000001",
				},
				PerpetualDEXStates: []WebsocketPerpetualDEXState{{PerpetualsAtOpenInterestCap: []string{"CANTO", "FTM"}}, {}},
			},
		},
		{
			name:  "twapStates",
			frame: `{"channel":"twapStates","data":{"dex":"","user":"0x0000000000000000000000000000000000000006","states":[[2285948,{"coin":"@107","user":"0x0000000000000000000000000000000000000006","side":"A","sz":"1000.0","executedSz":"333.29","executedNtl":"31171.44245","minutes":4320,"reduceOnly":true,"randomize":true,"timestamp":1791192756084,"trigger":{"px":"90.0","above":true},"stopPx":"92.0"}]]}}`,
			exp: &WebsocketTWAPStates{
				User: "0x0000000000000000000000000000000000000006",
				States: []TWAPStateEntry{{
					TWAPID: 2285948,
					State: TWAPState{
						Coin:             "@107",
						User:             "0x0000000000000000000000000000000000000006",
						Side:             "A",
						Size:             1000,
						ExecutedSize:     333.29,
						ExecutedNotional: 31171.44245,
						Minutes:          4320,
						ReduceOnly:       true,
						Randomise:        true,
						Timestamp:        milli(1791192756084),
						Trigger:          &TWAPTrigger{Price: 90, Above: true},
						StopPrice:        92,
					},
				}},
			},
		},
		{
			name:  "builder DEX twapStates",
			frame: `{"channel":"twapStates","data":{"dex":"xyz","user":"0x0000000000000000000000000000000000000003","states":[]}}`,
			exp:   &WebsocketTWAPStates{DEX: "xyz", User: "0x0000000000000000000000000000000000000003", States: []TWAPStateEntry{}},
		},
		{
			name:  "clearinghouseState",
			frame: `{"channel":"clearinghouseState","data":{"dex":"","user":"0x0000000000000000000000000000000000000001","clearinghouseState":` + websocketClearinghouseStateJSON + `}}`,
			exp:   &WebsocketClearinghouseState{User: "0x0000000000000000000000000000000000000001", ClearinghouseState: testWebsocketClearinghouseState()},
		},
		{
			name:  "builder DEX clearinghouseState",
			frame: `{"channel":"clearinghouseState","data":{"dex":"xyz","user":"0x0000000000000000000000000000000000000001","clearinghouseState":` + websocketEmptyClearinghouseStateJSON + `}}`,
			exp:   &WebsocketClearinghouseState{DEX: "xyz", User: "0x0000000000000000000000000000000000000001", ClearinghouseState: testWebsocketEmptyClearinghouseState()},
		},
		{
			name:  "openOrders",
			frame: `{"channel":"openOrders","data":{"dex":"xyz","user":"0x000000000000000000000000000000000000000a","orders":[{"coin":"xyz:XYZ100","side":"A","limitPx":"31175.0","sz":"0.0296","oid":566617880209,"timestamp":1791280999484,"triggerCondition":"N/A","isTrigger":false,"triggerPx":"0.0","children":[],"isPositionTpsl":false,"reduceOnly":false,"orderType":"Limit","origSz":"0.032","tif":"Alo","cloid":"0x000000000000000000065d291ba7c026"},{"coin":"xyz:XYZ100","side":"B","limitPx":"31180.0","sz":"0.0296","oid":566617880210,"timestamp":1791280999484,"triggerCondition":"Price above 31180","isTrigger":true,"triggerPx":"31180.0","children":[{"coin":"xyz:XYZ100","side":"B","limitPx":"30000.0","sz":"0.0296","oid":566617880211,"timestamp":1791280999484,"triggerCondition":"Price below 30000","isTrigger":true,"triggerPx":"30000.0","children":[],"isPositionTpsl":true,"reduceOnly":true,"orderType":"Stop Market","origSz":"0.032","tif":"FrontendMarket","cloid":"0x000000000000000000065d291ba7c028"}],"isPositionTpsl":true,"reduceOnly":true,"orderType":"Take Profit Market","origSz":"0.032","tif":"FrontendMarket","cloid":"0x000000000000000000065d291ba7c027"}]}}`,
			exp: &WebsocketOpenOrders{
				DEX:  "xyz",
				User: "0x000000000000000000000000000000000000000a",
				Orders: []FrontendOpenOrderResponse{
					{
						Coin:             "xyz:XYZ100",
						Side:             "A",
						LimitPrice:       31175,
						Size:             0.0296,
						OrderID:          566617880209,
						Timestamp:        milli(1791280999484),
						OriginalSize:     0.032,
						ClientOrderID:    "0x000000000000000000065d291ba7c026",
						TriggerCondition: "N/A",
						Children:         []FrontendOpenOrderResponse{},
						OrderType:        "Limit",
						TimeInForce:      "Alo",
					},
					{
						Coin:             "xyz:XYZ100",
						Side:             "B",
						LimitPrice:       31180,
						Size:             0.0296,
						OrderID:          566617880210,
						Timestamp:        milli(1791280999484),
						OriginalSize:     0.032,
						ClientOrderID:    "0x000000000000000000065d291ba7c027",
						TriggerCondition: "Price above 31180",
						IsTrigger:        true,
						TriggerPrice:     31180,
						Children: []FrontendOpenOrderResponse{{
							Coin:             "xyz:XYZ100",
							Side:             "B",
							LimitPrice:       30000,
							Size:             0.0296,
							OrderID:          566617880211,
							Timestamp:        milli(1791280999484),
							OriginalSize:     0.032,
							ClientOrderID:    "0x000000000000000000065d291ba7c028",
							TriggerCondition: "Price below 30000",
							IsTrigger:        true,
							TriggerPrice:     30000,
							Children:         []FrontendOpenOrderResponse{},
							IsPositionTPSL:   true,
							ReduceOnly:       true,
							OrderType:        "Stop Market",
							TimeInForce:      "FrontendMarket",
						}},
						IsPositionTPSL: true,
						ReduceOnly:     true,
						OrderType:      "Take Profit Market",
						TimeInForce:    "FrontendMarket",
					},
				},
			},
		},
		{
			name:  "activeAssetData",
			frame: `{"channel":"activeAssetData","data":{"user":"0x0000000000000000000000000000000000000001","coin":"BTC","leverage":{"type":"isolated","value":10,"rawUsd":"-4999.5"},"maxTradeSzs":["683.30131","684.41969"],"availableToTrade":["2937170.6810349999","2941978.0374650001"],"markPx":"85970.0"}}`,
			exp: &ActiveAssetDataResponse{
				User:             "0x0000000000000000000000000000000000000001",
				Coin:             "BTC",
				Leverage:         Leverage{Type: "isolated", Value: 10, RawUSD: -4999.5},
				MaxTradeSizes:    [2]types.Number{683.30131, 684.41969},
				AvailableToTrade: [2]types.Number{2937170.6810349999, 2941978.0374650001},
				MarkPrice:        85970,
			},
		},
		{
			name:  "bbo",
			frame: `{"channel":"bbo","data":{"coin":"BTC","time":1791278884013,"bbo":[{"px":"85811.0","sz":"9.48256","n":43},{"px":"85812.0","sz":"4.09277","n":1}]}}`,
			exp: &WebsocketBestBidOffer{
				Coin:         "BTC",
				Time:         milli(1791278884013),
				BestBidOffer: [2]*L2Level{{Price: 85811, Size: 9.48256, NumberOfOrders: 43}, {Price: 85812, Size: 4.09277, NumberOfOrders: 1}},
			},
		},
		{
			name:  "bbo without bids",
			frame: `{"channel":"bbo","data":{"coin":"@107","time":1791279271233,"bbo":[null,{"px":"92.638","sz":"12.89","n":1}]}}`,
			exp:   &WebsocketBestBidOffer{Coin: "@107", Time: milli(1791279271233), BestBidOffer: [2]*L2Level{nil, {Price: 92.638, Size: 12.89, NumberOfOrders: 1}}},
		},
		{
			name:  "spotState",
			frame: `{"channel":"spotState","data":{"user":"0x0000000000000000000000000000000000000002","spotState":{"balances":[{"coin":"USDC","token":0,"total":"100508.11272667","hold":"49999.89","entryNtl":"1.0"},{"coin":"VAULT","token":106,"total":"0.95437326","hold":"0.5","entryNtl":"0.00095437"}],"tokenToAvailableAfterMaintenance":[[0,"100508.11272667"]]}}}`,
			exp: &WebsocketSpotState{
				User: "0x0000000000000000000000000000000000000002",
				SpotState: SpotClearinghouseStateResponse{
					Balances: []SpotBalance{
						{Coin: currency.USDC, Total: 100508.11272667, Hold: 49999.89, EntryNotional: 1},
						{Coin: currency.NewCode("VAULT"), Token: 106, Total: 0.95437326, Hold: 0.5, EntryNotional: 0.00095437},
					},
					TokenToAvailableAfterMaintenance: []TokenAmount{{Amount: 100508.11272667}},
				},
			},
		},
		{
			name:  "allDexsClearinghouseState",
			frame: `{"channel":"allDexsClearinghouseState","data":{"user":"0x0000000000000000000000000000000000000001","clearinghouseStates":[["",` + websocketClearinghouseStateJSON + `],["xyz",` + websocketEmptyClearinghouseStateJSON + `]]}}`,
			exp: &WebsocketAllDEXsClearinghouseState{
				User: "0x0000000000000000000000000000000000000001",
				ClearinghouseStates: []DEXClearinghouseState{
					{ClearinghouseState: testWebsocketClearinghouseState()},
					{DEX: "xyz", ClearinghouseState: testWebsocketEmptyClearinghouseState()},
				},
			},
		},
		{
			name:  "allDexsAssetCtxs",
			frame: `{"channel":"allDexsAssetCtxs","data":{"ctxs":[["",[{"funding":"0.0000125","openInterest":"39988.35486","prevDayPx":"85840.0","dayNtlVlm":"1821731901.6448168755","premium":"-0.0000932097","oraclePx":"85828.0","markPx":"85814.0","midPx":"85818.0","impactPxs":["85816.0","85820.0"],"dayBaseVlm":"21234.78753"}]],["xyz",[{"funding":"0.00000625","openInterest":"7177.89","prevDayPx":"30751.0","dayNtlVlm":"276887160.4848001003","premium":"0.0000855208","oraclePx":"31162.0","markPx":"31164.0","midPx":"31164.5","impactPxs":["31164.0","31165.33"],"dayBaseVlm":"8935.3045"}]]]}}`,
			exp: &WebsocketAllDEXsAssetContexts{Contexts: []DEXAssetContexts{
				{AssetContexts: []PerpetualAssetContext{{
					Funding:           0.0000125,
					OpenInterest:      39988.35486,
					PreviousDayPrice:  85840,
					DayNotionalVolume: 1821731901.6448168755,
					Premium:           -0.0000932097,
					OraclePrice:       85828,
					MarkPrice:         85814,
					MidPrice:          85818,
					ImpactPrices:      []types.Number{85816, 85820},
					DayBaseVolume:     21234.78753,
				}}},
				{DEX: "xyz", AssetContexts: []PerpetualAssetContext{{
					Funding:           0.00000625,
					OpenInterest:      7177.89,
					PreviousDayPrice:  30751,
					DayNotionalVolume: 276887160.4848001003,
					Premium:           0.0000855208,
					OraclePrice:       31162,
					MarkPrice:         31164,
					MidPrice:          31164.5,
					ImpactPrices:      []types.Number{31164, 31165.33},
					DayBaseVolume:     8935.3045,
				}}},
			}},
		},
		{
			name:  "allDexsAssetCtxs without a book",
			frame: `{"channel":"allDexsAssetCtxs","data":{"ctxs":[["",[{"funding":"0.0000125","openInterest":"39988.35486","prevDayPx":"85840.0","dayNtlVlm":"1821731901.6448168755","premium":null,"oraclePx":"85828.0","markPx":"85814.0","midPx":null,"impactPxs":null,"dayBaseVlm":"21234.78753"}]]]}}`,
			exp: &WebsocketAllDEXsAssetContexts{Contexts: []DEXAssetContexts{{AssetContexts: []PerpetualAssetContext{{
				Funding:           0.0000125,
				OpenInterest:      39988.35486,
				PreviousDayPrice:  85840,
				DayNotionalVolume: 1821731901.6448168755,
				OraclePrice:       85828,
				MarkPrice:         85814,
				DayBaseVolume:     21234.78753,
			}}}}},
		},
	} {
		require.NoErrorf(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(tc.frame)), "wsHandleData must not error for %s", tc.name)
		assert.Equalf(t, tc.exp, receiveWebsocketData(t, ex), "wsHandleData should relay the decoded %s message", tc.name)
	}
	assertNoWebsocketData(t, ex)
}

func TestWebsocketMessageUnmarshal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		data   string
		target any
		exp    any
	}{
		{
			name:   "fast l2Book",
			data:   `{"coin":"SOL","time":1791279270407,"levels":[[{"px":"119.19","sz":"896.3","n":7},{"px":"119.18","sz":"157.98","n":5}],[{"px":"119.2","sz":"684.82","n":11},{"px":"119.21","sz":"862.57","n":17}]],"fast":true}`,
			target: new(WebsocketL2Book),
			exp: &WebsocketL2Book{
				Coin: "SOL",
				Time: milli(1791279270407),
				Levels: [][]L2Level{
					{{Price: 119.19, Size: 896.3, NumberOfOrders: 7}, {Price: 119.18, Size: 157.98, NumberOfOrders: 5}},
					{{Price: 119.2, Size: 684.82, NumberOfOrders: 11}, {Price: 119.21, Size: 862.57, NumberOfOrders: 17}},
				},
				Fast: true,
			},
		},
		{
			name:   "aggregated l2Book",
			data:   `{"coin":"BTC","time":1791279269350,"levels":[[{"px":"85874.0","sz":"17.86492","n":63}],[{"px":"85876.0","sz":"1.22481","n":6}]],"spread":"1.0"}`,
			target: new(WebsocketL2Book),
			exp: &WebsocketL2Book{
				Coin:   "BTC",
				Time:   milli(1791279269350),
				Levels: [][]L2Level{{{Price: 85874, Size: 17.86492, NumberOfOrders: 63}}, {{Price: 85876, Size: 1.22481, NumberOfOrders: 6}}},
				Spread: 1,
			},
		},
		{
			name:   "perpetual activeAssetCtx",
			data:   `{"coin":"BTC","ctx":{"funding":"0.0000125","openInterest":"39980.76392","prevDayPx":"85840.0","dayNtlVlm":"1821282705.4506571293","premium":"-0.0001864194","oraclePx":"85828.0","markPx":"85814.0","midPx":"85811.5","impactPxs":["85811.0","85812.0"],"dayBaseVlm":"21229.55909"}}`,
			target: new(WebsocketPerpetualAssetContext),
			exp: &WebsocketPerpetualAssetContext{Coin: "BTC", Context: PerpetualAssetContext{
				Funding:           0.0000125,
				OpenInterest:      39980.76392,
				PreviousDayPrice:  85840,
				DayNotionalVolume: 1821282705.4506571293,
				Premium:           -0.0001864194,
				OraclePrice:       85828,
				MarkPrice:         85814,
				MidPrice:          85811.5,
				ImpactPrices:      []types.Number{85811, 85812},
				DayBaseVolume:     21229.55909,
			}},
		},
		{
			name:   "spot activeAssetCtx",
			data:   `{"coin":"@107","ctx":{"prevDayPx":"93.124","dayNtlVlm":"64552049.6973299459","markPx":"92.949","midPx":"92.945","circulatingSupply":"298462370.4079165459","coin":"@107","totalSupply":"998882916.6303099394","dayBaseVlm":"690824.1400000008"}}`,
			target: new(WebsocketSpotAssetContext),
			exp: &WebsocketSpotAssetContext{Coin: "@107", Context: SpotAssetContext{
				PreviousDayPrice:  93.124,
				DayNotionalVolume: 64552049.6973299459,
				MarkPrice:         92.949,
				MidPrice:          92.945,
				CirculatingSupply: 298462370.4079165459,
				Coin:              "@107",
				TotalSupply:       998882916.6303099394,
				DayBaseVolume:     690824.1400000008,
			}},
		},
		{
			name:   "userFills",
			data:   `{"isSnapshot":true,"user":"0x000000000000000000000000000000000000000a","fills":[` + websocketFillJSON + `]}`,
			target: new(WebsocketUserFills),
			exp:    &WebsocketUserFills{IsSnapshot: true, User: "0x000000000000000000000000000000000000000a", Fills: []FillResponse{testWebsocketFill()}},
		},
		{
			name:   "userTwapSliceFills",
			data:   `{"isSnapshot":true,"user":"0x0000000000000000000000000000000000000006","twapSliceFills":[{"fill":` + websocketFillJSON + `,"twapId":2288616}]}`,
			target: new(WebsocketUserTWAPSliceFills),
			exp: &WebsocketUserTWAPSliceFills{
				IsSnapshot:     true,
				User:           "0x0000000000000000000000000000000000000006",
				TWAPSliceFills: []TWAPSliceFillResponse{{Fill: testWebsocketFill(), TWAPID: 2288616}},
			},
		},
		{
			name:   "userEvents",
			data:   `{"fills":[` + websocketFillJSON + `],"funding":{"time":1791280800037,"coin":"BTC","usdc":"-0.601087","szi":"0.55919","fundingRate":"0.0000125","nSamples":24},"liquidation":{"lid":123456,"liquidator":"0x0000000000000000000000000000000000000009","liquidated_user":"0x000000000000000000000000000000000000000b","liquidated_ntl_pos":"356.80015","liquidated_account_value":"2.956494"},"nonUserCancel":[{"coin":"PNUT","oid":566615769247}],"twapSliceFills":[{"fill":` + websocketFillJSON + `,"twapId":2288616}]}`,
			target: new(WebsocketUserEvent),
			exp: &WebsocketUserEvent{
				Fills:          []FillResponse{testWebsocketFill()},
				Funding:        &WebsocketUserFunding{Time: milli(1791280800037), Coin: "BTC", USDC: -0.601087, SignedSize: 0.55919, FundingRate: 0.0000125, NumberOfSamples: 24},
				Liquidation:    &WebsocketLiquidation{LiquidationID: 123456, Liquidator: "0x0000000000000000000000000000000000000009", LiquidatedUser: "0x000000000000000000000000000000000000000b", LiquidatedNotionalPosition: 356.80015, LiquidatedAccountValue: 2.956494},
				NonUserCancel:  []WebsocketNonUserCancel{{Coin: "PNUT", OrderID: 566615769247}},
				TWAPSliceFills: []TWAPSliceFillResponse{{Fill: testWebsocketFill(), TWAPID: 2288616}},
			},
		},
	} {
		require.NoErrorf(t, json.Unmarshal([]byte(tc.data), tc.target), "Unmarshal must not error for %s", tc.name)
		assert.Equalf(t, tc.exp, tc.target, "Unmarshal should decode every %s field", tc.name)
	}
}

// websocketFillJSON is a liquidation fill from the websocket fixtures, with every field set
const websocketFillJSON = `{"coin":"BTC","px":"86006.0","sz":"0.07905","side":"B","time":1791280942711,"startPosition":"-0.14136","dir":"Close Short","closedPnl":"-2.600745","hash":"0x0da3f51a7423740a0f1d0445f5707e02017400000f2692dcb16ca06d33274df4","oid":566617233807,"crossed":true,"fee":"-0.067987","tid":669583051366496,"cloid":"0x000000000000000000065d2917ff6969","feeToken":"USDC","twapId":2288616,"builderFee":"0.0001","liquidation":{"liquidatedUser":"0x000000000000000000000000000000000000000b","markPx":"86010.0","method":"backstop"}}`

// websocketSpotFillJSON is a TWAP slice fill of the HYPE spot market from the websocket fixtures
const websocketSpotFillJSON = `{"coin":"@107","px":"92.7","sz":"0.41","side":"A","time":1791279150013,"startPosition":"54377.71000602","dir":"Sell","closedPnl":"5.11885001","hash":"0x0000000000000000000000000000000000000000000000000000000000000000","oid":566598918312,"crossed":true,"fee":"0.01787849","tid":499584618036466,"feeToken":"USDC","twapId":null}`

// websocketLaterFillJSON is websocketFillJSON a block later, as a fill a reconnect misses
var websocketLaterFillJSON = strings.NewReplacer(`"time":1791280942711`, `"time":1791280950000`, `"tid":669583051366496`, `"tid":669583051366497`).Replace(websocketFillJSON)

// websocketLaterSpotFillJSON is websocketSpotFillJSON a slice later, as a slice fill a reconnect misses
var websocketLaterSpotFillJSON = strings.NewReplacer(`"time":1791279150013`, `"time":1791279180013`, `"tid":499584618036466`, `"tid":499584618036467`).Replace(websocketSpotFillJSON)

// testWebsocketFill is websocketFillJSON decoded
func testWebsocketFill() FillResponse {
	return FillResponse{
		Coin:          "BTC",
		Price:         86006,
		Size:          0.07905,
		Side:          "B",
		Time:          milli(1791280942711),
		StartPosition: -0.14136,
		Direction:     "Close Short",
		ClosedPNL:     -2.600745,
		Hash:          "0x0da3f51a7423740a0f1d0445f5707e02017400000f2692dcb16ca06d33274df4",
		OrderID:       566617233807,
		Crossed:       true,
		Fee:           -0.067987,
		TradeID:       669583051366496,
		ClientOrderID: "0x000000000000000000065d2917ff6969",
		Liquidation:   &FillLiquidation{LiquidatedUser: "0x000000000000000000000000000000000000000b", MarkPrice: 86010, Method: "backstop"},
		FeeToken:      currency.USDC,
		BuilderFee:    0.0001,
		TWAPID:        2288616,
	}
}

// testWebsocketFillData is websocketFillJSON converted to a fill
func testWebsocketFillData(exchangeName string) fill.Data {
	return fill.Data{
		ID:            "669583051366496",
		Timestamp:     time.UnixMilli(1791280942711).UTC(),
		Exchange:      exchangeName,
		AssetType:     asset.PerpetualContract,
		CurrencyPair:  perpetualPair,
		Side:          order.Buy,
		OrderID:       "566617233807",
		ClientOrderID: "0x000000000000000000065d2917ff6969",
		TradeID:       "669583051366496",
		Price:         86006,
		Amount:        0.07905,
	}
}

// testWebsocketSpotFillData is websocketSpotFillJSON converted to a fill
func testWebsocketSpotFillData(exchangeName string) fill.Data {
	return fill.Data{
		ID:           "499584618036466",
		Timestamp:    time.UnixMilli(1791279150013).UTC(),
		Exchange:     exchangeName,
		AssetType:    asset.Spot,
		CurrencyPair: spotPair,
		Side:         order.Sell,
		OrderID:      "566598918312",
		TradeID:      "499584618036466",
		Price:        92.7,
		Amount:       0.41,
	}
}

// testWebsocketFillUpdate is websocketFillJSON relayed to the order stream as a trade of an order no row has shown, so
// dated by its fill
func testWebsocketFillUpdate(exchangeName string) order.Detail {
	filled := time.UnixMilli(1791280942711).UTC()
	price, size := 86006.0, 0.07905
	return order.Detail{
		Exchange:      exchangeName,
		OrderID:       "566617233807",
		ClientOrderID: "0x000000000000000000065d2917ff6969",
		Side:          order.Buy,
		AssetType:     asset.PerpetualContract,
		Date:          filled,
		LastUpdated:   filled,
		Pair:          perpetualPair,
		Trades:        []order.TradeHistory{{Price: price, Amount: size, Fee: -0.067987, Exchange: exchangeName, TID: "669583051366496", Side: order.Buy, Timestamp: filled, FeeAsset: "USDC", Total: price * size}},
	}
}

// testWebsocketSpotFillUpdate is websocketSpotFillJSON relayed to the order stream as a trade of an order no row has
// shown, so dated by its fill
func testWebsocketSpotFillUpdate(exchangeName string) order.Detail {
	filled := time.UnixMilli(1791279150013).UTC()
	price, size := 92.7, 0.41
	return order.Detail{
		Exchange:    exchangeName,
		OrderID:     "566598918312",
		Side:        order.Sell,
		AssetType:   asset.Spot,
		Date:        filled,
		LastUpdated: filled,
		Pair:        spotPair,
		Trades:      []order.TradeHistory{{Price: price, Amount: size, Fee: 0.01787849, Exchange: exchangeName, TID: "499584618036466", Side: order.Sell, Timestamp: filled, FeeAsset: "USDC", Total: price * size}},
	}
}

// receiveTicker returns the relayed ticker, checking and then clearing its current time stamp
func receiveTicker(t *testing.T, ex *Exchange) *ticker.Price {
	t.Helper()
	price, ok := receiveWebsocketData(t, ex).(*ticker.Price)
	require.True(t, ok, "the relayed message must be a ticker")
	assert.WithinDuration(t, time.Now(), price.LastUpdated, time.Minute, "the ticker should be stamped with the current time")
	price.LastUpdated = time.Time{}
	return price
}

func TestWebsocketHandleCandle(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.Error(t, ex.wsHandleCandle(t.Context(), []byte(`"invalid"`)), "wsHandleCandle should reject invalid data")
	for _, tc := range []struct {
		frame string
		exp   kline.Item
	}{
		{
			frame: `{"channel":"candle","data":{"t":1791278880000,"T":1791278939999,"s":"BTC","i":"1m","o":"85827.0","c":"85812.0","h":"85827.0","l":"85801.0","v":"0.73617","n":20}}`,
			exp: kline.Item{
				Exchange: ex.Name,
				Pair:     perpetualPair,
				Asset:    asset.PerpetualContract,
				Interval: kline.OneMin,
				Candles:  []kline.Candle{{Time: time.UnixMilli(1791278880000).UTC(), Open: 85827, High: 85827, Low: 85801, Close: 85812, Volume: 0.73617}},
			},
		},
		{
			frame: `{"channel":"candle","data":{"t":1791279240000,"T":1791279299999,"s":"@107","i":"1m","o":"92.652","c":"92.621","h":"92.653","l":"92.581","v":"2115.03","n":57}}`,
			exp: kline.Item{
				Exchange: ex.Name,
				Pair:     spotPair,
				Asset:    asset.Spot,
				Interval: kline.OneMin,
				Candles:  []kline.Candle{{Time: time.UnixMilli(1791279240000).UTC(), Open: 92.652, High: 92.653, Low: 92.581, Close: 92.621, Volume: 2115.03}},
			},
		},
	} {
		require.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(tc.frame)), "wsHandleData must not error")
		assert.Equalf(t, tc.exp, receiveWebsocketData(t, ex), "wsHandleCandle should relay the %s %s candle", tc.exp.Asset, tc.exp.Pair)
	}
	err := ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"candle","data":{"s":"ETH","i":"1m"}}`))
	assert.ErrorIs(t, err, errPairMappingNotFound, "wsHandleCandle should reject an unknown coin")
	err = ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"candle","data":{"s":"BTC","i":"7m"}}`))
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "wsHandleCandle should reject an unsupported interval")
	assertNoWebsocketData(t, ex)
}

func TestWebsocketHandleOrderbook(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.Error(t, ex.wsHandleOrderbook(t.Context(), []byte(`"invalid"`)), "wsHandleOrderbook should reject invalid data")
	frame := `{"channel":"l2Book","data":{"coin":"BTC","time":1791278882097,"levels":[[{"px":"85821.0","sz":"0.96592","n":4},{"px":"85818.0","sz":"0.80629","n":1}],[{"px":"85822.0","sz":"26.98831","n":71},{"px":"85823.0","sz":"4.42262","n":4}]]}}`
	require.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(frame)), "wsHandleData must not error")
	_, ok := receiveWebsocketData(t, ex).(*orderbook.Depth)
	require.True(t, ok, "wsHandleOrderbook must relay the orderbook depth")
	book, err := ex.Websocket.Orderbook.GetOrderbook(perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.Equal(t, orderbook.Levels{{Price: 85821, Amount: 0.96592, OrderCount: 4}, {Price: 85818, Amount: 0.80629, OrderCount: 1}}, book.Bids, "wsHandleOrderbook should load the bids")
	assert.Equal(t, orderbook.Levels{{Price: 85822, Amount: 26.98831, OrderCount: 71}, {Price: 85823, Amount: 4.42262, OrderCount: 4}}, book.Asks, "wsHandleOrderbook should load the asks")
	assert.True(t, book.LastUpdated.Equal(time.UnixMilli(1791278882097)), "wsHandleOrderbook should load the book time")

	err = ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"l2Book","data":{"coin":"BTC","time":1791278882097,"levels":[[]]}}`))
	assert.ErrorIs(t, err, errInvalidBookLevelCount, "wsHandleOrderbook should reject a book without two sides")
	err = ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"l2Book","data":{"coin":"ETH","time":1791278882097,"levels":[[],[]]}}`))
	assert.ErrorIs(t, err, errPairMappingNotFound, "wsHandleOrderbook should reject an unknown coin")
	err = ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"l2Book","data":{"coin":"@107","time":1791278882097,"levels":[[{"px":"0","sz":"1","n":1}],[{"px":"92.7","sz":"1","n":1}]]}}`))
	assert.ErrorIs(t, err, orderbook.ErrPriceZero, "wsHandleOrderbook should reject an invalid book")
	assertNoWebsocketData(t, ex)
}

func TestWebsocketHandleTrades(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.Error(t, ex.wsHandleTrades([]byte(`"invalid"`)), "wsHandleTrades should reject invalid data")
	frame := `{"channel":"trades","data":[{"coin":"BTC","side":"B","px":"85827.0","sz":"0.02721","time":1791278878126,"hash":"0xe7a30278f69a68bee91c0445f4fe140206c5005e919d87908b6badcbb59e42a9","tid":188095306343783,"users":["0x0000000000000000000000000000000000000004","0x0000000000000000000000000000000000000005"]},{"coin":"BTC","side":"A","px":"85826.0","sz":"0.00897","time":1791278878126,"hash":"0x7e3eb751c3147c9c7fb80445f4fe140206cd00375e179b6e220762a482185687","tid":250009747508244,"users":["0x0000000000000000000000000000000000000004","0x0000000000000000000000000000000000000005"]}]}`
	acknowledgement := func(coin string) []byte {
		return []byte(`{"channel":"subscriptionResponse","data":{"method":"subscribe","subscription":{"type":"trades","coin":"` + coin + `"}}}`)
	}
	require.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), acknowledgement("BTC")), "wsHandleData must not error for the acknowledgement")
	require.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(frame)), "wsHandleData must not error for the replayed trades")
	assertNoWebsocketData(t, ex)

	require.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(frame)), "wsHandleData must not error")
	exp := []trade.Data{
		{TID: "188095306343783", Exchange: ex.Name, CurrencyPair: perpetualPair, AssetType: asset.PerpetualContract, Side: order.Buy, Price: 85827, Amount: 0.02721, Timestamp: time.UnixMilli(1791278878126).UTC()},
		{TID: "250009747508244", Exchange: ex.Name, CurrencyPair: perpetualPair, AssetType: asset.PerpetualContract, Side: order.Sell, Price: 85826, Amount: 0.00897, Timestamp: time.UnixMilli(1791278878126).UTC()},
	}
	assert.Equal(t, exp, receiveWebsocketData(t, ex), "wsHandleTrades should relay the trades after the replay")

	require.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), acknowledgement("BTC")), "wsHandleData must not error for a reconnect's acknowledgement")
	reconnect := `{"channel":"trades","data":[{"coin":"BTC","side":"B","px":"85827.0","sz":"0.02721","time":1791278878126,"tid":188095306343783},{"coin":"BTC","side":"B","px":"85827.0","sz":"0.5","time":1791278878126,"tid":260000000000000},{"coin":"BTC","side":"A","px":"85830.0","sz":"0.01","time":1791278880000,"tid":3}]}`
	require.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(reconnect)), "wsHandleData must not error for a reconnect's replay")
	missed := []trade.Data{
		{TID: "260000000000000", Exchange: ex.Name, CurrencyPair: perpetualPair, AssetType: asset.PerpetualContract, Side: order.Buy, Price: 85827, Amount: 0.5, Timestamp: time.UnixMilli(1791278878126).UTC()},
		{TID: "3", Exchange: ex.Name, CurrencyPair: perpetualPair, AssetType: asset.PerpetualContract, Side: order.Sell, Price: 85830, Amount: 0.01, Timestamp: time.UnixMilli(1791278880000).UTC()},
	}
	assert.Equal(t, missed, receiveWebsocketData(t, ex), "wsHandleTrades should relay only the replayed trades after the last relayed one")

	// A market without recent trades replays an empty batch, which must not leave its replay awaited
	require.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), acknowledgement("@107")), "wsHandleData must not error for the acknowledgement")
	require.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"trades","data":[]}`)), "wsHandleData must not error for an empty replay")
	assertNoWebsocketData(t, ex)
	require.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"trades","data":[{"coin":"@107","side":"A","px":"92.7","sz":"0.41","time":1791279150013,"tid":499584618036466}]}`)), "wsHandleData must not error")
	firstLive := []trade.Data{{TID: "499584618036466", Exchange: ex.Name, CurrencyPair: spotPair, AssetType: asset.Spot, Side: order.Sell, Price: 92.7, Amount: 0.41, Timestamp: time.UnixMilli(1791279150013).UTC()}}
	assert.Equal(t, firstLive, receiveWebsocketData(t, ex), "wsHandleTrades should relay a market's first live trades after its empty replay")
	require.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"trades","data":[]}`)), "wsHandleData must not error for an empty batch")
	assertNoWebsocketData(t, ex)

	mixed := `{"channel":"trades","data":[{"coin":"BTC","side":"X","px":"85827.0","sz":"1","time":1791278878126,"tid":1},{"coin":"BTC","side":"B","px":"85827.0","sz":"0.02721","time":1791278878126,"tid":188095306343783}]}`
	assert.ErrorIs(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(mixed)), order.ErrSideIsInvalid, "wsHandleTrades should report a trade that cannot convert")
	assert.Equal(t, exp[:1], receiveWebsocketData(t, ex), "wsHandleTrades should relay the trades that convert")

	err := ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"trades","data":[{"coin":"ETH","side":"B","px":"2710.8","sz":"1","time":1791278878126,"tid":1}]}`))
	assert.ErrorIs(t, err, errPairMappingNotFound, "wsHandleTrades should reject an unknown coin")
	assertNoWebsocketData(t, ex)
}

func TestExpectTradeReplay(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.expectTradeReplay("BTC")
	ex.expectTradeReplay("ETH")
	assert.Equal(t, map[string]struct{}{"BTC": {}, "ETH": {}}, ex.websocketTradeReplays, "expectTradeReplay should mark each coin, creating the marks when needed")
}

func TestConsumeTradeReplay(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	assert.False(t, ex.consumeTradeReplay("BTC"), "consumeTradeReplay should report no replay before one is expected")
	ex.expectTradeReplay("BTC")
	assert.False(t, ex.consumeTradeReplay("ETH"), "consumeTradeReplay should report no replay for another coin")
	assert.True(t, ex.consumeTradeReplay("BTC"), "consumeTradeReplay should report the expected replay")
	assert.False(t, ex.consumeTradeReplay("BTC"), "consumeTradeReplay should report one replay per subscription")
}

func TestClearTradeReplay(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.clearTradeReplay("BTC")
	assert.Empty(t, ex.websocketTradeReplays, "clearTradeReplay should ignore a coin without a mark")
	ex.expectTradeReplay("BTC")
	ex.expectTradeReplay("ETH")
	ex.clearTradeReplay("BTC")
	assert.Equal(t, map[string]struct{}{"ETH": {}}, ex.websocketTradeReplays, "clearTradeReplay should remove only the coin's mark")
}

func TestConsumeEmptyTradeReplay(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.consumeEmptyTradeReplay()
	assert.Empty(t, ex.websocketTradeReplays, "consumeEmptyTradeReplay should ignore an empty message without an awaited replay")
	assert.Empty(t, ex.websocketWatermarks.trades, "consumeEmptyTradeReplay should record no watermark without an awaited replay")
	ex.expectTradeReplay("@189")
	ex.consumeEmptyTradeReplay()
	assert.Empty(t, ex.websocketTradeReplays, "consumeEmptyTradeReplay should consume the awaited replay")
	assert.Equal(t, map[string]websocketPosition{"@189": {}}, ex.websocketWatermarks.trades, "consumeEmptyTradeReplay should record that the coin's replays hold no trade relayed before")
	ex.expectTradeReplay("@189")
	ex.expectTradeReplay("BTC")
	ex.consumeEmptyTradeReplay()
	assert.Equal(t, map[string]struct{}{"@189": {}, "BTC": {}}, ex.websocketTradeReplays, "consumeEmptyTradeReplay should consume no replay when it cannot tell which one an empty message answers")
	assert.Equal(t, map[string]websocketPosition{"@189": {}}, ex.websocketWatermarks.trades, "consumeEmptyTradeReplay should record no watermark when it cannot tell which replay an empty message answers")

	live := websocketPosition{time: time.UnixMilli(1791279150013), tradeID: 499584618036466}
	ex.websocketWatermarks.trades["BTC"] = live
	ex.clearTradeReplays()
	ex.expectTradeReplay("BTC")
	ex.consumeEmptyTradeReplay()
	assert.Equal(t, live, ex.websocketWatermarks.trades["BTC"], "consumeEmptyTradeReplay should keep the watermark of a coin whose trades were relayed")
}

// TestWebsocketTradesAfterEmptyReplay has a market without recent trades replay none, then resubscribes it after its
// connection dropped before any live trade
func TestWebsocketTradesAfterEmptyReplay(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	acknowledgement := []byte(`{"channel":"subscriptionResponse","data":{"method":"subscribe","subscription":{"type":"trades","coin":"@107"}}}`)
	require.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), acknowledgement), "wsHandleData must not error for the acknowledgement")
	require.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"trades","data":[]}`)), "wsHandleData must not error for the empty replay")
	assertNoWebsocketData(t, ex)
	ex.clearTradeReplays()
	require.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), acknowledgement), "wsHandleData must not error for the resubscription's acknowledgement")
	require.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"trades","data":[{"coin":"@107","side":"A","px":"92.7","sz":"0.41","time":1791279150013,"tid":499584618036466}]}`)), "wsHandleData must not error for the resubscription's replay")
	exp := []trade.Data{{TID: "499584618036466", Exchange: ex.Name, CurrencyPair: spotPair, AssetType: asset.Spot, Side: order.Sell, Price: 92.7, Amount: 0.41, Timestamp: time.UnixMilli(1791279150013).UTC()}}
	assert.Equal(t, exp, receiveWebsocketData(t, ex), "wsHandleTrades should relay the trade made while the connection was down")
}

func TestClearTradeReplays(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.clearTradeReplays()
	assert.Empty(t, ex.websocketTradeReplays, "clearTradeReplays should ignore missing marks")
	ex.expectTradeReplay("BTC")
	ex.expectTradeReplay("ETH")
	ex.clearTradeReplays()
	assert.Empty(t, ex.websocketTradeReplays, "clearTradeReplays should remove every mark")
}

func TestUnrelayedTrades(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	relayed := RecentTradeResponse{Coin: "BTC", Time: milli(1791278878126), TradeID: 188095306343783}
	missed := RecentTradeResponse{Coin: "BTC", Time: milli(1791278880000), TradeID: 3}
	ex.expectTradeReplay("BTC")
	assert.Empty(t, ex.unrelayedTrades([]RecentTradeResponse{relayed}), "unrelayedTrades should relay none of the first subscription's replay")
	assert.False(t, ex.consumeTradeReplay("BTC"), "unrelayedTrades should consume the replay mark")
	assert.Equal(t, []RecentTradeResponse{relayed}, ex.unrelayedTrades([]RecentTradeResponse{relayed}), "unrelayedTrades should relay a live message")
	ex.expectTradeReplay("BTC")
	assert.Equal(t, []RecentTradeResponse{missed}, ex.unrelayedTrades([]RecentTradeResponse{relayed, missed}), "unrelayedTrades should relay the replayed trades after the last relayed one")
}

func TestUnrelayedEntries(t *testing.T) {
	t.Parallel()
	position := func(p *websocketPosition) websocketPosition { return *p }
	early := websocketPosition{time: time.UnixMilli(1791278878126), tradeID: 5}
	late := websocketPosition{time: time.UnixMilli(1791278880000), tradeID: 1}
	later := websocketPosition{time: time.UnixMilli(1791278890000), tradeID: 1}
	watermarks := make(map[string]websocketPosition)
	assert.Empty(t, unrelayedEntries(watermarks, "BTC", []websocketPosition{late, early}, true, position), "unrelayedEntries should relay none of a replay before the feed has a watermark")
	assert.Equal(t, late, watermarks["BTC"], "unrelayedEntries should move the watermark to the newest entry")
	assert.Equal(t, []websocketPosition{later}, unrelayedEntries(watermarks, "BTC", []websocketPosition{early, later, late}, true, position), "unrelayedEntries should relay a replay's entries after the watermark")
	assert.Equal(t, later, watermarks["BTC"], "unrelayedEntries should move the watermark to the replay's newest entry")
	assert.Equal(t, []websocketPosition{early}, unrelayedEntries(watermarks, "BTC", []websocketPosition{early}, false, position), "unrelayedEntries should relay every entry of a live message")
	assert.Equal(t, later, watermarks["BTC"], "unrelayedEntries should keep a watermark newer than the message")
	assert.Empty(t, unrelayedEntries(watermarks, "ETH", []websocketPosition{later}, true, position), "unrelayedEntries should keep each feed's watermark apart")
}

func TestClearWebsocketWatermark(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.clearWebsocketWatermark(&WebsocketSubscription{Type: wsChannelTrades, Coin: "BTC"})
	assert.Equal(t, websocketWatermarks{}, ex.websocketWatermarks, "clearWebsocketWatermark should ignore a feed without a watermark")
	position := websocketPosition{time: time.UnixMilli(1791278878126), tradeID: 1}
	fillsFeed := websocketFillsFeed{channel: wsChannelUserFills, user: testAccountAddress}
	sliceFillsFeed := websocketFillsFeed{channel: wsChannelUserTWAPSliceFills, user: testAccountAddress}
	ex.websocketWatermarks = websocketWatermarks{
		trades: map[string]websocketPosition{"BTC": position, "ETH": position},
		fills:  map[websocketFillsFeed]websocketPosition{fillsFeed: position, sliceFillsFeed: position},
	}
	ex.clearWebsocketWatermark(&WebsocketSubscription{Type: wsChannelTrades, Coin: "BTC"})
	ex.clearWebsocketWatermark(&WebsocketSubscription{Type: wsChannelUserFills, User: testAccountAddress})
	ex.clearWebsocketWatermark(&WebsocketSubscription{Type: wsChannelOrderbook, Coin: "ETH"})
	exp := websocketWatermarks{
		trades: map[string]websocketPosition{"ETH": position},
		fills:  map[websocketFillsFeed]websocketPosition{sliceFillsFeed: position},
	}
	assert.Equal(t, exp, ex.websocketWatermarks, "clearWebsocketWatermark should forget only the unsubscribed feeds' watermarks")
	ex.clearWebsocketWatermark(&WebsocketSubscription{Type: wsChannelUserTWAPSliceFills, User: testAccountAddress})
	assert.Empty(t, ex.websocketWatermarks.fills, "clearWebsocketWatermark should forget a TWAP slice fills feed's watermark")
}

func TestWebsocketHandleOrderUpdates(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.Error(t, ex.wsHandleOrderUpdates(t.Context(), []byte(`"invalid"`)), "wsHandleOrderUpdates should reject invalid data")
	frame := `{"channel":"orderUpdates","data":[{"order":{"coin":"@107","side":"B","limitPx":"93.218","sz":"3.28","oid":566617964508,"timestamp":1791281008463,"origSz":"3.28","cloid":"0x000000000000000000065d291c305ae0"},"status":"open","statusTimestamp":1791281008463},{"order":{"coin":"@107","side":"B","limitPx":"93.218","sz":"0.0","oid":566617964508,"timestamp":1791281008463,"origSz":"3.28","cloid":"0x000000000000000000065d291c305ae0"},"status":"filled","statusTimestamp":1791281008463},{"order":{"coin":"xyz:XYZ100","side":"A","limitPx":"31169.5","sz":"0.1","oid":566617840215,"timestamp":1791280994709,"origSz":"0.1","cloid":"0x000000000000000000065d291b5eb630"},"status":"canceled","statusTimestamp":1791281001374}]}`
	require.NoError(t, ex.wsHandleData(t.Context(), accountConn(t, ex), []byte(frame)), "wsHandleData must not error")
	placed := time.UnixMilli(1791281008463).UTC()
	exp := []order.Detail{
		{
			Price:           93.218,
			Amount:          3.28,
			RemainingAmount: 3.28,
			Exchange:        ex.Name,
			OrderID:         "566617964508",
			ClientOrderID:   "0x000000000000000000065d291c305ae0",
			Side:            order.Buy,
			Status:          order.Open,
			AssetType:       asset.Spot,
			Date:            placed,
			LastUpdated:     placed,
			Pair:            spotPair,
		},
		{
			Price:          93.218,
			Amount:         3.28,
			ExecutedAmount: 3.28,
			Exchange:       ex.Name,
			OrderID:        "566617964508",
			ClientOrderID:  "0x000000000000000000065d291c305ae0",
			Side:           order.Buy,
			Status:         order.Filled,
			AssetType:      asset.Spot,
			Date:           placed,
			LastUpdated:    placed,
			Pair:           spotPair,
		},
		{
			Price:           31169.5,
			Amount:          0.1,
			RemainingAmount: 0.1,
			Exchange:        ex.Name,
			OrderID:         "566617840215",
			ClientOrderID:   "0x000000000000000000065d291b5eb630",
			Side:            order.Sell,
			Status:          order.Cancelled,
			AssetType:       asset.PerpetualContract,
			Date:            time.UnixMilli(1791280994709).UTC(),
			LastUpdated:     time.UnixMilli(1791281001374).UTC(),
			Pair:            testBuilderPair,
		},
	}
	assert.Equal(t, exp, receiveWebsocketData(t, ex), "wsHandleOrderUpdates should relay the converted orders")

	mixed := `{"channel":"orderUpdates","data":[{"order":{"coin":"ETH","side":"B","oid":1,"timestamp":1791281008463},"status":"open","statusTimestamp":1791281008463},{"order":{"coin":"BTC","side":"B","oid":2,"timestamp":1791281008463},"status":"unknownStatus","statusTimestamp":1791281008463},{"order":{"coin":"@107","side":"B","limitPx":"93.218","sz":"3.28","oid":566617964508,"timestamp":1791281008463,"origSz":"3.28","cloid":"0x000000000000000000065d291c305ae0"},"status":"open","statusTimestamp":1791281008463}]}`
	err := ex.wsHandleData(t.Context(), accountConn(t, ex), []byte(mixed))
	assert.ErrorIs(t, err, errPairMappingNotFound, "wsHandleOrderUpdates should report an order of an unknown coin")
	assert.ErrorIs(t, err, errUnsupportedOrderStatus, "wsHandleOrderUpdates should report an order with an unknown status")
	assert.Equal(t, exp[:1], receiveWebsocketData(t, ex), "wsHandleOrderUpdates should relay the orders that convert")

	outcome := `{"channel":"orderUpdates","data":[{"order":{"coin":"#90700","side":"B","limitPx":"0.52","sz":"100.0","oid":570870444900,"timestamp":1791281008463,"origSz":"100.0"},"status":"open","statusTimestamp":1791281008463}]}`
	require.NoError(t, ex.wsHandleData(t.Context(), accountConn(t, ex), []byte(outcome)), "wsHandleData must not error for an outcome order, which no supported asset holds")
	require.NoError(t, ex.wsHandleData(t.Context(), accountConn(t, ex), []byte(`{"channel":"orderUpdates","data":[]}`)), "wsHandleData must not error for an empty batch")
	assertNoWebsocketData(t, ex)
}

// websocketOrderClientID is the client order ID of the order events tests' orders
const websocketOrderClientID = "0x000000000000000000065d291c305ae1"

// websocketOrderPlaced is when the order events tests' orders are placed
const websocketOrderPlaced = int64(1791281008463)

// websocketOrderRowJSON returns an orderUpdates row of a BTC buy limited at 85000 and placed at websocketOrderPlaced
func websocketOrderRowJSON(orderID uint64, status, size, originalSize string, statusTimestamp int64) string {
	return `{"order":{"coin":"BTC","side":"B","limitPx":"85000.0","sz":"` + size + `","oid":` + strconv.FormatUint(orderID, 10) +
		`,"timestamp":` + strconv.FormatInt(websocketOrderPlaced, 10) + `,"origSz":"` + originalSize + `","cloid":"` + websocketOrderClientID +
		`"},"status":"` + status + `","statusTimestamp":` + strconv.FormatInt(statusTimestamp, 10) + `}`
}

// websocketOrderFillJSON returns a fill of a websocketOrderRowJSON order: a taker fill paying a 0.1 USDC fee, or a maker
// fill earning a 0.05 USDC rebate
func websocketOrderFillJSON(orderID, tradeID uint64, price, size string, filled int64, taker bool) string {
	fee := "-0.05"
	if taker {
		fee = "0.1"
	}
	return `{"coin":"BTC","px":"` + price + `","sz":"` + size + `","side":"B","time":` + strconv.FormatInt(filled, 10) +
		`,"startPosition":"0.0","dir":"Open Long","closedPnl":"0.0","hash":"0x0","oid":` + strconv.FormatUint(orderID, 10) +
		`,"crossed":` + strconv.FormatBool(taker) + `,"fee":"` + fee + `","tid":` + strconv.FormatUint(tradeID, 10) +
		`,"cloid":"` + websocketOrderClientID + `","feeToken":"USDC"}`
}

// websocketOrderTrade returns a websocketOrderFillJSON fill as the trade its order update carries
func websocketOrderTrade(exchangeName string, tradeID uint64, price, size float64, filled int64, taker bool) order.TradeHistory {
	fee := -0.05
	if taker {
		fee = 0.1
	}
	return order.TradeHistory{
		Price:     price,
		Amount:    size,
		Fee:       fee,
		Exchange:  exchangeName,
		TID:       strconv.FormatUint(tradeID, 10),
		Side:      order.Buy,
		Timestamp: time.UnixMilli(filled).UTC(),
		IsMaker:   !taker,
		FeeAsset:  "USDC",
		Total:     price * size,
	}
}

// websocketOrderUpdatesFrame returns an orderUpdates message holding the rows
func websocketOrderUpdatesFrame(rows ...string) []byte {
	return []byte(`{"channel":"orderUpdates","data":[` + strings.Join(rows, ",") + `]}`)
}

// websocketUserFillsFrame returns a userEvents message holding the fills
func websocketUserFillsFrame(fills ...string) []byte {
	return []byte(`{"channel":"user","data":{"fills":[` + strings.Join(fills, ",") + `]}}`)
}

// handleWebsocketFrames has the exchange handle messages read from the account connection
func handleWebsocketFrames(t *testing.T, ex *Exchange, frames ...[]byte) {
	t.Helper()
	for _, frame := range frames {
		require.NoErrorf(t, ex.wsHandleData(t.Context(), accountConn(t, ex), frame), "wsHandleData must not error for %s", frame)
	}
}

// receiveOrderUpdates returns the order updates of the next relayed message
func receiveOrderUpdates(t *testing.T, ex *Exchange) []order.Detail {
	t.Helper()
	updates, ok := receiveWebsocketData(t, ex).([]order.Detail)
	require.True(t, ok, "the relayed message must hold order updates")
	return updates
}

// mergeOrderUpdates merges one order's relayed updates as the engine does: the first adds the order, and each later one
// updates it through UpdateOrderFromDetail
func mergeOrderUpdates(t *testing.T, updates ...order.Detail) order.Detail {
	t.Helper()
	merged := updates[0].Copy()
	for i := 1; i < len(updates); i++ {
		require.NoError(t, merged.UpdateOrderFromDetail(&updates[i]), "UpdateOrderFromDetail must not error")
	}
	return merged
}

// newWebsocketOrderEventsExchange returns a websocket test exchange relaying fills only to the order stream
func newWebsocketOrderEventsExchange(t *testing.T) *Exchange {
	t.Helper()
	ex := newWebsocketTestExchange(t)
	ex.SetFillsFeedStatus(false)
	return ex
}

// TestWebsocketOrderEvents has orderUpdates rows and fills handled in the orders Hyperliquid sends them in, and merges the
// relayed updates as the engine does, to show what the engine holds of each order
func TestWebsocketOrderEvents(t *testing.T) {
	t.Parallel()
	placed := time.UnixMilli(websocketOrderPlaced).UTC()
	at := func(offset int64) time.Time { return time.UnixMilli(websocketOrderPlaced + offset).UTC() }
	// Expected sizes and fees are worked out in float64, as the handlers work them out from the decoded values
	originalSize, takerFee, makerFee := 0.02, 0.1, -0.05
	row := func(exchangeName, orderID string) order.Detail {
		return order.Detail{
			Price:         85000,
			Amount:        0.02,
			Exchange:      exchangeName,
			OrderID:       orderID,
			ClientOrderID: websocketOrderClientID,
			Side:          order.Buy,
			AssetType:     asset.PerpetualContract,
			Date:          placed,
			Pair:          perpetualPair,
		}
	}
	opened := func(exchangeName, orderID string) order.Detail {
		update := row(exchangeName, orderID)
		update.RemainingAmount, update.Status, update.LastUpdated = 0.02, order.Open, placed
		return update
	}

	t.Run("placement row, partial fills, then the filled row", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketOrderEventsExchange(t)
		handleWebsocketFrames(t, ex,
			websocketOrderUpdatesFrame(websocketOrderRowJSON(1001, "open", "0.02", "0.02", websocketOrderPlaced)),
			websocketUserFillsFrame(websocketOrderFillJSON(1001, 11, "84990.0", "0.012", websocketOrderPlaced+100, false)),
			websocketUserFillsFrame(websocketOrderFillJSON(1001, 12, "84995.0", "0.008", websocketOrderPlaced+200, true)),
			websocketOrderUpdatesFrame(websocketOrderRowJSON(1001, "filled", "0.0", "0.02", websocketOrderPlaced+200)),
		)
		price1, size1, price2, size2 := 84990.0, 0.012, 84995.0, 0.008
		cost := float64(price1*size1) + float64(price2*size2)
		executed := size1 + size2
		partial := row(ex.Name, "1001")
		partial.AverageExecutedPrice, partial.ExecutedAmount, partial.RemainingAmount, partial.Cost = price1*size1/size1, size1, originalSize-size1, price1*size1
		partial.Fee, partial.FeeAsset, partial.Status, partial.LastUpdated = -0.05, currency.USDC, order.PartiallyFilled, at(100)
		partial.Trades = []order.TradeHistory{websocketOrderTrade(ex.Name, 11, price1, size1, websocketOrderPlaced+100, false)}
		filledByFills := row(ex.Name, "1001")
		filledByFills.AverageExecutedPrice, filledByFills.ExecutedAmount, filledByFills.Cost = cost/executed, executed, cost
		filledByFills.Fee, filledByFills.FeeAsset, filledByFills.Status, filledByFills.LastUpdated = makerFee+takerFee, currency.USDC, order.Filled, at(200)
		filledByFills.Trades = []order.TradeHistory{websocketOrderTrade(ex.Name, 12, price2, size2, websocketOrderPlaced+200, true)}
		filledRow := row(ex.Name, "1001")
		filledRow.AverageExecutedPrice, filledRow.ExecutedAmount, filledRow.Cost, filledRow.Status, filledRow.LastUpdated = cost/executed, 0.02, cost, order.Filled, at(200)
		filledRow.Fee, filledRow.FeeAsset = makerFee+takerFee, currency.USDC
		exp := []order.Detail{opened(ex.Name, "1001"), partial, filledByFills, filledRow}
		relayed := make([]order.Detail, 0, len(exp))
		for range exp {
			relayed = append(relayed, receiveOrderUpdates(t, ex)...)
		}
		assertNoWebsocketData(t, ex)
		assert.Equal(t, exp, relayed, "the order stream should report the placement, each fill's cumulative execution and the filled row's cost")
		merged := mergeOrderUpdates(t, relayed...)
		assert.Equal(t, 0.02, merged.ExecutedAmount, "the engine should hold the order's executed size")
		assert.Equal(t, cost, merged.Cost, "the engine should hold the order's cost")
		assert.Equal(t, cost/executed, merged.AverageExecutedPrice, "the engine should hold the order's average price")
		assert.Equal(t, makerFee+takerFee, merged.Fee, "the engine should hold the order's total fee")
		assert.Equal(t, order.Filled, merged.Status, "the engine should hold the order filled")
		assert.Len(t, merged.Trades, 2, "the engine should hold each trade of the order")
		assert.Nil(t, ex.websocketWatermarks.orders.byOrderID[1001], "the filled row should forget the order")
	})

	t.Run("fill before its placement row", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketOrderEventsExchange(t)
		handleWebsocketFrames(t, ex,
			websocketUserFillsFrame(websocketOrderFillJSON(1002, 21, "85010.0", "0.005", websocketOrderPlaced, true)),
			websocketOrderUpdatesFrame(websocketOrderRowJSON(1002, "open", "0.02", "0.02", websocketOrderPlaced)),
			websocketOrderUpdatesFrame(websocketOrderRowJSON(1002, "canceled", "0.015", "0.02", websocketOrderPlaced+300)),
		)
		price, size := 85010.0, 0.005
		tradeOnly := order.Detail{
			Exchange:      ex.Name,
			OrderID:       "1002",
			ClientOrderID: websocketOrderClientID,
			Side:          order.Buy,
			AssetType:     asset.PerpetualContract,
			Date:          placed,
			LastUpdated:   placed,
			Pair:          perpetualPair,
			Trades:        []order.TradeHistory{websocketOrderTrade(ex.Name, 21, price, size, websocketOrderPlaced, true)},
		}
		ahead := row(ex.Name, "1002")
		ahead.AverageExecutedPrice, ahead.ExecutedAmount, ahead.RemainingAmount, ahead.Cost = price*size/size, size, originalSize-size, price*size
		ahead.Fee, ahead.FeeAsset, ahead.Status, ahead.LastUpdated = takerFee, currency.USDC, order.PartiallyFilled, placed
		cancelled := row(ex.Name, "1002")
		cancelled.AverageExecutedPrice, cancelled.ExecutedAmount, cancelled.RemainingAmount, cancelled.Cost = price*size/size, originalSize-0.015, 0.015, price*size
		cancelled.Fee, cancelled.FeeAsset, cancelled.Status, cancelled.LastUpdated = takerFee, currency.USDC, order.PartiallyFilledCancelled, at(300)
		relayed := slices.Concat(receiveOrderUpdates(t, ex), receiveOrderUpdates(t, ex), receiveOrderUpdates(t, ex))
		assertNoWebsocketData(t, ex)
		assert.Equal(t, []order.Detail{tradeOnly, ahead, cancelled}, relayed, "the order stream should relay the early fill alone, then the placement row with the fill's execution")
		merged := mergeOrderUpdates(t, relayed...)
		assert.Equal(t, cancelled.ExecutedAmount, merged.ExecutedAmount, "the engine should hold the order's executed size")
		assert.Equal(t, price*size, merged.Cost, "the engine should hold the order's cost")
		assert.Equal(t, order.PartiallyFilledCancelled, merged.Status, "the engine should hold the order partly filled and cancelled")
		assert.Equal(t, placed, merged.Date, "the engine should date the order by its placement")
	})

	t.Run("untracked order whose first row shows earlier fills", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketOrderEventsExchange(t)
		handleWebsocketFrames(t, ex,
			websocketOrderUpdatesFrame(websocketOrderRowJSON(1003, "open", "0.015", "0.02", websocketOrderPlaced)),
			websocketUserFillsFrame(websocketOrderFillJSON(1003, 31, "85000.0", "0.003", websocketOrderPlaced+100, true)),
			websocketOrderUpdatesFrame(websocketOrderRowJSON(1003, "canceled", "0.012", "0.02", websocketOrderPlaced+300)),
		)
		partial := row(ex.Name, "1003")
		partial.ExecutedAmount, partial.RemainingAmount, partial.Status, partial.LastUpdated = originalSize-0.015, 0.015, order.PartiallyFilled, placed
		tradeOnly := order.Detail{
			Exchange:      ex.Name,
			OrderID:       "1003",
			ClientOrderID: websocketOrderClientID,
			Side:          order.Buy,
			AssetType:     asset.PerpetualContract,
			Date:          placed,
			LastUpdated:   at(100),
			Pair:          perpetualPair,
			Trades:        []order.TradeHistory{websocketOrderTrade(ex.Name, 31, 85000, 0.003, websocketOrderPlaced+100, true)},
		}
		cancelled := row(ex.Name, "1003")
		cancelled.ExecutedAmount, cancelled.RemainingAmount, cancelled.Status, cancelled.LastUpdated = originalSize-0.012, 0.012, order.PartiallyFilledCancelled, at(300)
		relayed := slices.Concat(receiveOrderUpdates(t, ex), receiveOrderUpdates(t, ex), receiveOrderUpdates(t, ex))
		assertNoWebsocketData(t, ex)
		assert.Equal(t, []order.Detail{partial, tradeOnly, cancelled}, relayed, "the order stream should relay the fill of an order whose earlier fills it lacks alone, and the rows without a cost")
		learnt := relayed[0].Copy()
		learnt.ExecutedAmount = 0.007
		require.NoError(t, learnt.UpdateOrderFromDetail(&relayed[1]), "UpdateOrderFromDetail must not error")
		assert.Equal(t, 0.007, learnt.ExecutedAmount, "the engine should keep an executed size it learnt elsewhere when a fill is relayed alone")
		merged := mergeOrderUpdates(t, relayed...)
		assert.Equal(t, cancelled.ExecutedAmount, merged.ExecutedAmount, "the engine should hold the executed size the final row shows")
		assert.Zero(t, merged.Cost, "the engine should hold no cost the fills cannot cover")
		assert.Equal(t, order.PartiallyFilledCancelled, merged.Status, "the engine should hold the order partly filled and cancelled")
	})

	t.Run("cancel after a partial fill", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketOrderEventsExchange(t)
		handleWebsocketFrames(t, ex,
			websocketOrderUpdatesFrame(websocketOrderRowJSON(1004, "open", "0.02", "0.02", websocketOrderPlaced)),
			websocketUserFillsFrame(websocketOrderFillJSON(1004, 41, "84980.0", "0.006", websocketOrderPlaced+100, false)),
			websocketOrderUpdatesFrame(websocketOrderRowJSON(1004, "canceled", "0.014", "0.02", websocketOrderPlaced+300)),
		)
		price, size := 84980.0, 0.006
		partial := row(ex.Name, "1004")
		partial.AverageExecutedPrice, partial.ExecutedAmount, partial.RemainingAmount, partial.Cost = price*size/size, size, originalSize-size, price*size
		partial.Fee, partial.FeeAsset, partial.Status, partial.LastUpdated = -0.05, currency.USDC, order.PartiallyFilled, at(100)
		partial.Trades = []order.TradeHistory{websocketOrderTrade(ex.Name, 41, price, size, websocketOrderPlaced+100, false)}
		cancelled := row(ex.Name, "1004")
		cancelled.AverageExecutedPrice, cancelled.ExecutedAmount, cancelled.RemainingAmount, cancelled.Cost = price*size/size, originalSize-0.014, 0.014, price*size
		cancelled.Fee, cancelled.FeeAsset, cancelled.Status, cancelled.LastUpdated = makerFee, currency.USDC, order.PartiallyFilledCancelled, at(300)
		relayed := slices.Concat(receiveOrderUpdates(t, ex), receiveOrderUpdates(t, ex), receiveOrderUpdates(t, ex))
		assertNoWebsocketData(t, ex)
		assert.Equal(t, []order.Detail{opened(ex.Name, "1004"), partial, cancelled}, relayed, "the order stream should report the cancel of a partly filled order with its cost")
		merged := mergeOrderUpdates(t, relayed...)
		assert.Equal(t, cancelled.ExecutedAmount, merged.ExecutedAmount, "the engine should hold the order's executed size")
		assert.Equal(t, price*size, merged.Cost, "the engine should hold the order's cost")
		assert.Equal(t, price*size/size, merged.AverageExecutedPrice, "the engine should hold the order's average price")
		assert.Equal(t, order.PartiallyFilledCancelled, merged.Status, "the engine should hold the order partly filled and cancelled")
	})

	t.Run("duplicate fill from userEvents and userFills", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		fillJSON := websocketOrderFillJSON(1005, 51, "85000.0", "0.01", websocketOrderPlaced+100, true)
		handleWebsocketFrames(t, ex,
			websocketOrderUpdatesFrame(websocketOrderRowJSON(1005, "open", "0.02", "0.02", websocketOrderPlaced)),
			websocketUserFillsFrame(fillJSON),
			[]byte(`{"channel":"userFills","data":{"user":"`+testAccountAddress+`","fills":[`+fillJSON+`]}}`),
		)
		price, size := 85000.0, 0.01
		partial := row(ex.Name, "1005")
		partial.AverageExecutedPrice, partial.ExecutedAmount, partial.RemainingAmount, partial.Cost = price*size/size, size, originalSize-size, price*size
		partial.Fee, partial.FeeAsset, partial.Status, partial.LastUpdated = 0.1, currency.USDC, order.PartiallyFilled, at(100)
		partial.Trades = []order.TradeHistory{websocketOrderTrade(ex.Name, 51, price, size, websocketOrderPlaced+100, true)}
		relayed := slices.Concat(receiveOrderUpdates(t, ex), receiveOrderUpdates(t, ex))
		assert.Equal(t, []order.Detail{opened(ex.Name, "1005"), partial}, relayed, "the order stream should relay a fill both feeds deliver once")
		fillData := fill.Data{
			ID:            "51",
			Timestamp:     at(100),
			Exchange:      ex.Name,
			AssetType:     asset.PerpetualContract,
			CurrencyPair:  perpetualPair,
			Side:          order.Buy,
			OrderID:       "1005",
			ClientOrderID: websocketOrderClientID,
			TradeID:       "51",
			Price:         price,
			Amount:        size,
		}
		assert.Equal(t, []fill.Data{fillData}, receiveWebsocketData(t, ex), "the fills feed should relay the userEvents fill")
		assert.Equal(t, []fill.Data{fillData}, receiveWebsocketData(t, ex), "the fills feed should relay the userFills fill as it did before")
		assertNoWebsocketData(t, ex)
		assert.Equal(t, size, mergeOrderUpdates(t, relayed...).ExecutedAmount, "the engine should count a fill both feeds deliver once")
	})

	t.Run("final row before the fills that complete it", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketOrderEventsExchange(t)
		handleWebsocketFrames(t, ex,
			websocketOrderUpdatesFrame(websocketOrderRowJSON(1006, "open", "0.02", "0.02", websocketOrderPlaced)),
			websocketUserFillsFrame(websocketOrderFillJSON(1006, 61, "85000.0", "0.01", websocketOrderPlaced+100, true)),
			websocketOrderUpdatesFrame(websocketOrderRowJSON(1006, "filled", "0.0", "0.02", websocketOrderPlaced+300)),
			websocketUserFillsFrame(websocketOrderFillJSON(1006, 62, "85005.0", "0.004", websocketOrderPlaced+250, true)),
			websocketUserFillsFrame(websocketOrderFillJSON(1006, 63, "85010.0", "0.006", websocketOrderPlaced+260, false)),
		)
		price1, size1, price2, size2, price3, size3 := 85000.0, 0.01, 85005.0, 0.004, 85010.0, 0.006
		cost := float64(price1*size1) + float64(price2*size2) + float64(price3*size3)
		executed := size1 + size2 + size3
		partial := row(ex.Name, "1006")
		partial.AverageExecutedPrice, partial.ExecutedAmount, partial.RemainingAmount, partial.Cost = price1*size1/size1, size1, originalSize-size1, price1*size1
		partial.Fee, partial.FeeAsset, partial.Status, partial.LastUpdated = 0.1, currency.USDC, order.PartiallyFilled, at(100)
		partial.Trades = []order.TradeHistory{websocketOrderTrade(ex.Name, 61, price1, size1, websocketOrderPlaced+100, true)}
		filledRow := row(ex.Name, "1006")
		filledRow.ExecutedAmount, filledRow.Status, filledRow.LastUpdated = 0.02, order.Filled, at(300)
		lagging := order.Detail{
			Exchange:      ex.Name,
			OrderID:       "1006",
			ClientOrderID: websocketOrderClientID,
			Side:          order.Buy,
			AssetType:     asset.PerpetualContract,
			Date:          placed,
			LastUpdated:   at(250),
			Pair:          perpetualPair,
			Trades:        []order.TradeHistory{websocketOrderTrade(ex.Name, 62, price2, size2, websocketOrderPlaced+250, true)},
		}
		completed := filledRow
		completed.AverageExecutedPrice, completed.Cost, completed.Fee, completed.FeeAsset = cost/executed, cost, takerFee+takerFee+makerFee, currency.USDC
		completed.Trades = []order.TradeHistory{websocketOrderTrade(ex.Name, 63, price3, size3, websocketOrderPlaced+260, false)}
		relayed := make([]order.Detail, 0, 5)
		for range 5 {
			relayed = append(relayed, receiveOrderUpdates(t, ex)...)
		}
		assertNoWebsocketData(t, ex)
		assert.Equal(t, []order.Detail{opened(ex.Name, "1006"), partial, filledRow, lagging, completed}, relayed, "the order stream should complete a final row with its cost once the fills reach its executed size")
		merged := mergeOrderUpdates(t, relayed[:4]...)
		assert.Equal(t, order.Filled, merged.Status, "the engine should keep the order filled while its last fills arrive")
		assert.Equal(t, 0.02, merged.ExecutedAmount, "the engine should keep the final row's executed size while its last fills arrive")
		merged = mergeOrderUpdates(t, relayed...)
		assert.Equal(t, 0.02, merged.ExecutedAmount, "the engine should hold the order's executed size")
		assert.Equal(t, cost, merged.Cost, "the engine should hold the order's cost")
		assert.Equal(t, takerFee+takerFee+makerFee, merged.Fee, "the engine should hold the order's total fee")
		assert.Equal(t, order.Filled, merged.Status, "the engine should hold the order filled")
		assert.Len(t, merged.Trades, 3, "the engine should hold each trade of the order")
		assert.Nil(t, ex.websocketWatermarks.orders.byOrderID[1006], "the completing fill should forget the order")
	})

	t.Run("outcome orders and fills are skipped", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketOrderEventsExchange(t)
		handleWebsocketFrames(t, ex,
			[]byte(`{"channel":"orderUpdates","data":[{"order":{"coin":"#90700","side":"B","limitPx":"0.52","sz":"100.0","oid":1007,"timestamp":1791281008463,"origSz":"100.0"},"status":"open","statusTimestamp":1791281008463}]}`),
			[]byte(`{"channel":"user","data":{"fills":[{"coin":"#90700","px":"0.52","sz":"40.0","side":"B","time":1791281008563,"oid":1007,"crossed":true,"fee":"0.01","tid":71,"feeToken":"USDC"}]}}`),
		)
		assertNoWebsocketData(t, ex)
		assert.Empty(t, ex.websocketWatermarks.orders.byOrderID, "an outcome order should not be recorded")
	})

	t.Run("position TP/SL row with zero sizes", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketOrderEventsExchange(t)
		handleWebsocketFrames(t, ex,
			websocketOrderUpdatesFrame(websocketOrderRowJSON(1008, "open", "0.0", "0.0", websocketOrderPlaced)),
			websocketUserFillsFrame(websocketOrderFillJSON(1008, 81, "84000.0", "0.02", websocketOrderPlaced+100, true)),
		)
		zeroSized := row(ex.Name, "1008")
		zeroSized.Amount, zeroSized.Status, zeroSized.LastUpdated = 0, order.Open, placed
		tradeOnly := order.Detail{
			Exchange:      ex.Name,
			OrderID:       "1008",
			ClientOrderID: websocketOrderClientID,
			Side:          order.Buy,
			AssetType:     asset.PerpetualContract,
			Date:          placed,
			LastUpdated:   at(100),
			Pair:          perpetualPair,
			Trades:        []order.TradeHistory{websocketOrderTrade(ex.Name, 81, 84000, 0.02, websocketOrderPlaced+100, true)},
		}
		assert.Equal(t, []order.Detail{zeroSized, tradeOnly}, slices.Concat(receiveOrderUpdates(t, ex), receiveOrderUpdates(t, ex)), "the order stream should relay a zero sized row as converted and its order's fills alone")
		assertNoWebsocketData(t, ex)
	})

	t.Run("dropped account connection", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketOrderEventsExchange(t)
		handleWebsocketFrames(t, ex, websocketOrderUpdatesFrame(websocketOrderRowJSON(1009, "open", "0.02", "0.02", websocketOrderPlaced)))
		dropWebsocketAccountConnection(t, ex)
		handleWebsocketFrames(t, ex,
			websocketUserFillsFrame(websocketOrderFillJSON(1009, 91, "85000.0", "0.005", websocketOrderPlaced+100, true)),
			websocketOrderUpdatesFrame(websocketOrderRowJSON(1009, "canceled", "0.015", "0.02", websocketOrderPlaced+300)),
		)
		tradeOnly := order.Detail{
			Exchange:      ex.Name,
			OrderID:       "1009",
			ClientOrderID: websocketOrderClientID,
			Side:          order.Buy,
			AssetType:     asset.PerpetualContract,
			Date:          placed,
			LastUpdated:   at(100),
			Pair:          perpetualPair,
			Trades:        []order.TradeHistory{websocketOrderTrade(ex.Name, 91, 85000, 0.005, websocketOrderPlaced+100, true)},
		}
		cancelled := row(ex.Name, "1009")
		price, size := 85000.0, 0.005
		cancelled.AverageExecutedPrice, cancelled.ExecutedAmount, cancelled.RemainingAmount, cancelled.Cost = price*size/size, originalSize-0.015, 0.015, price*size
		cancelled.Fee, cancelled.FeeAsset, cancelled.Status, cancelled.LastUpdated = takerFee, currency.USDC, order.PartiallyFilledCancelled, at(300)
		relayed := slices.Concat(receiveOrderUpdates(t, ex), receiveOrderUpdates(t, ex), receiveOrderUpdates(t, ex))
		assertNoWebsocketData(t, ex)
		assert.Equal(t, []order.Detail{opened(ex.Name, "1009"), tradeOnly, cancelled}, relayed, "the order stream should relay a fill alone after the account connection dropped, until a row shows the fills cover the order")
	})

	// The fills summing to a final row's executed size are all of the order's fills, however many arrived before the row
	price1, size1, price2, size2 := 85000.0, 0.005, 85010.0, 0.015
	cost, executed := float64(price1*size1)+float64(price2*size2), size1+size2
	fillOnly := func(exchangeName, orderID string, tradeID uint64, price, size float64, offset int64, date time.Time) order.Detail {
		return order.Detail{
			Exchange:      exchangeName,
			OrderID:       orderID,
			ClientOrderID: websocketOrderClientID,
			Side:          order.Buy,
			AssetType:     asset.PerpetualContract,
			Date:          date,
			LastUpdated:   at(offset),
			Pair:          perpetualPair,
			Trades:        []order.TradeHistory{websocketOrderTrade(exchangeName, tradeID, price, size, websocketOrderPlaced+offset, true)},
		}
	}
	filledRow := func(exchangeName, orderID string) order.Detail {
		update := row(exchangeName, orderID)
		update.ExecutedAmount, update.Status, update.LastUpdated = originalSize, order.Filled, at(300)
		return update
	}
	completed := func(exchangeName, orderID string, tradeID uint64) order.Detail {
		update := filledRow(exchangeName, orderID)
		update.AverageExecutedPrice, update.Cost, update.Fee, update.FeeAsset = cost/executed, cost, takerFee+takerFee, currency.USDC
		update.Trades = []order.TradeHistory{websocketOrderTrade(exchangeName, tradeID, price2, size2, websocketOrderPlaced+250, true)}
		return update
	}
	assertCompleted := func(t *testing.T, ex *Exchange, orderID uint64, relayed []order.Detail) {
		t.Helper()
		merged := mergeOrderUpdates(t, relayed...)
		assert.Equal(t, originalSize, merged.ExecutedAmount, "the engine should hold the order's executed size")
		assert.Equal(t, cost, merged.Cost, "the engine should hold the cost of every fill")
		assert.Equal(t, cost/executed, merged.AverageExecutedPrice, "the engine should hold the average price of every fill")
		assert.Equal(t, takerFee+takerFee, merged.Fee, "the engine should hold the fees of every fill")
		assert.Equal(t, order.Filled, merged.Status, "the engine should hold the order filled")
		assert.Nil(t, ex.websocketWatermarks.orders.byOrderID[orderID], "the completing fill should forget the order")
	}

	t.Run("final row of an unseen order before its fills", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketOrderEventsExchange(t)
		handleWebsocketFrames(t, ex,
			websocketOrderUpdatesFrame(websocketOrderRowJSON(1010, "filled", "0.0", "0.02", websocketOrderPlaced+300)),
			websocketUserFillsFrame(websocketOrderFillJSON(1010, 101, "85000.0", "0.005", websocketOrderPlaced+100, true)),
			websocketUserFillsFrame(websocketOrderFillJSON(1010, 102, "85010.0", "0.015", websocketOrderPlaced+250, true)),
		)
		relayed := slices.Concat(receiveOrderUpdates(t, ex), receiveOrderUpdates(t, ex), receiveOrderUpdates(t, ex))
		assertNoWebsocketData(t, ex)
		exp := []order.Detail{filledRow(ex.Name, "1010"), fillOnly(ex.Name, "1010", 101, price1, size1, 100, placed), completed(ex.Name, "1010", 102)}
		assert.Equal(t, exp, relayed, "the order stream should complete the final row of an order first seen in it once its fills arrive")
		assertCompleted(t, ex, 1010, relayed)
	})

	t.Run("fill, final row, then the last fill", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketOrderEventsExchange(t)
		handleWebsocketFrames(t, ex,
			websocketUserFillsFrame(websocketOrderFillJSON(1011, 111, "85000.0", "0.005", websocketOrderPlaced+100, true)),
			websocketOrderUpdatesFrame(websocketOrderRowJSON(1011, "filled", "0.0", "0.02", websocketOrderPlaced+300)),
			websocketUserFillsFrame(websocketOrderFillJSON(1011, 112, "85010.0", "0.015", websocketOrderPlaced+250, true)),
		)
		relayed := slices.Concat(receiveOrderUpdates(t, ex), receiveOrderUpdates(t, ex), receiveOrderUpdates(t, ex))
		assertNoWebsocketData(t, ex)
		exp := []order.Detail{fillOnly(ex.Name, "1011", 111, price1, size1, 100, at(100)), filledRow(ex.Name, "1011"), completed(ex.Name, "1011", 112)}
		assert.Equal(t, exp, relayed, "the order stream should complete a final row that arrived between an order's fills")
		assertCompleted(t, ex, 1011, relayed)
	})

	t.Run("dropped account connection before the final row and the last fill", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketOrderEventsExchange(t)
		handleWebsocketFrames(t, ex,
			websocketOrderUpdatesFrame(websocketOrderRowJSON(1012, "open", "0.02", "0.02", websocketOrderPlaced)),
			websocketUserFillsFrame(websocketOrderFillJSON(1012, 121, "85000.0", "0.005", websocketOrderPlaced+100, true)),
		)
		dropWebsocketAccountConnection(t, ex)
		handleWebsocketFrames(t, ex,
			websocketOrderUpdatesFrame(websocketOrderRowJSON(1012, "filled", "0.0", "0.02", websocketOrderPlaced+300)),
			websocketUserFillsFrame(websocketOrderFillJSON(1012, 122, "85010.0", "0.015", websocketOrderPlaced+250, true)),
		)
		partial := row(ex.Name, "1012")
		partial.AverageExecutedPrice, partial.ExecutedAmount, partial.RemainingAmount, partial.Cost = price1*size1/size1, size1, originalSize-size1, price1*size1
		partial.Fee, partial.FeeAsset, partial.Status, partial.LastUpdated = 0.1, currency.USDC, order.PartiallyFilled, at(100)
		partial.Trades = []order.TradeHistory{websocketOrderTrade(ex.Name, 121, price1, size1, websocketOrderPlaced+100, true)}
		relayed := make([]order.Detail, 0, 4)
		for range 4 {
			relayed = append(relayed, receiveOrderUpdates(t, ex)...)
		}
		assertNoWebsocketData(t, ex)
		exp := []order.Detail{opened(ex.Name, "1012"), partial, filledRow(ex.Name, "1012"), completed(ex.Name, "1012", 122)}
		assert.Equal(t, exp, relayed, "the order stream should complete a final row after the account connection dropped, once the fills reach its executed size")
		assertCompleted(t, ex, 1012, relayed)
	})
	t.Run("fills feed resubscribed", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketOrderEventsExchange(t)
		fills := subscription.List{{Channel: wsSubscriptionUserEvents, Authenticated: true}}
		require.NoError(t, subscribeWebsocketTest(t, ex, fills), "Subscribe must not error")
		handleWebsocketFrames(t, ex,
			websocketOrderUpdatesFrame(websocketOrderRowJSON(1013, "open", "0.02", "0.02", websocketOrderPlaced)),
			websocketUserFillsFrame(websocketOrderFillJSON(1013, 131, "84000", "0.005", websocketOrderPlaced+100, true)),
		)
		merged := mergeOrderUpdates(t, append(receiveOrderUpdates(t, ex), receiveOrderUpdates(t, ex)...)...)
		require.NoError(t, unsubscribeWebsocketTest(t, ex, fills), "Unsubscribe must not error")
		rest := merged.Copy()
		rest.ExecutedAmount, rest.RemainingAmount, rest.Cost = 0.015, 0.005, 1260
		require.NoError(t, merged.UpdateOrderFromDetail(&rest), "UpdateOrderFromDetail must not error")
		require.NoError(t, subscribeWebsocketTest(t, ex, subscription.List{{Channel: wsSubscriptionUserEvents, Authenticated: true}}), "Subscribe must not error again")
		handleWebsocketFrames(t, ex, websocketUserFillsFrame(websocketOrderFillJSON(1013, 133, "84000", "0.002", websocketOrderPlaced+300, true)))
		latest := receiveOrderUpdates(t, ex)[0]
		assert.Zero(t, latest.ExecutedAmount, "wsHandleSubscriptionResponse should prevent incomplete cumulative execution")
		require.NoError(t, merged.UpdateOrderFromDetail(&latest), "UpdateOrderFromDetail must not error for the fill")
		assert.Equal(t, 0.015, merged.ExecutedAmount, "UpdateOrderFromDetail should retain execution learnt during the gap")
	})
	t.Run("aggregated userFills refused beside userEvents", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketOrderEventsExchange(t)
		require.NoError(t, subscribeWebsocketTest(t, ex, subscription.List{{Channel: wsSubscriptionUserEvents, Authenticated: true}}), "Subscribe must not error for userEvents")
		handleWebsocketFrames(t, ex,
			websocketOrderUpdatesFrame(websocketOrderRowJSON(1015, "open", "0.02", "0.02", websocketOrderPlaced)),
			websocketUserFillsFrame(websocketOrderFillJSON(1015, 151, "84000", "0.005", websocketOrderPlaced+100, true)),
		)
		receiveOrderUpdates(t, ex)
		receiveOrderUpdates(t, ex)
		aggregated := subscription.List{{Channel: wsChannelUserFills, Authenticated: true, Params: map[string]any{"aggregateByTime": true}}}
		require.ErrorIs(t, subscribeWebsocketTest(t, ex, aggregated), subscription.ErrExclusiveSubscription, "Subscribe must refuse aggregated userFills beside userEvents, which would relay every fill twice")
		assert.Nil(t, ex.Websocket.GetSubscription(newWebsocketSubscriptionKey(aggregated[0])), "Subscribe should not hold the refused aggregated userFills")
		handleWebsocketFrames(t, ex, websocketUserFillsFrame(websocketOrderFillJSON(1015, 152, "84000", "0.002", websocketOrderPlaced+200, true)))
		assert.InDelta(t, 0.007, receiveOrderUpdates(t, ex)[0].ExecutedAmount, 1e-12, "a refused aggregated userFills should leave the order's fills covered")
	})
	for _, feed := range []string{wsChannelUserFills, wsChannelUserTWAPSliceFills} {
		t.Run(feed+" subscribed after the order's row", func(t *testing.T) {
			t.Parallel()
			ex := newWebsocketOrderEventsExchange(t)
			require.NoError(t, subscribeWebsocketTest(t, ex, subscription.List{{Channel: wsChannelOrderUpdates, Authenticated: true}}), "Subscribe must not error for orderUpdates")
			handleWebsocketFrames(t, ex, websocketOrderUpdatesFrame(websocketOrderRowJSON(1014, "open", "0.02", "0.02", websocketOrderPlaced)))
			receiveOrderUpdates(t, ex)
			require.NoErrorf(t, subscribeWebsocketTest(t, ex, subscription.List{{Channel: feed, Authenticated: true}}), "Subscribe must not error for %s", feed)
			frame := func(snapshot bool, fill string) []byte {
				if feed == wsChannelUserFills {
					return []byte(`{"channel":"userFills","data":{"isSnapshot":` + strconv.FormatBool(snapshot) + `,"user":"` + testAccountAddress + `","fills":[` + fill + `]}}`)
				}
				return []byte(`{"channel":"userTwapSliceFills","data":{"isSnapshot":` + strconv.FormatBool(snapshot) + `,"user":"` + testAccountAddress + `","twapSliceFills":[{"fill":` + fill + `,"twapId":5}]}}`)
			}
			handleWebsocketFrames(t, ex,
				frame(true, websocketOrderFillJSON(1014, 141, "84000", "0.01", websocketOrderPlaced+100, true)),
				frame(false, websocketOrderFillJSON(1014, 142, "84000", "0.002", websocketOrderPlaced+200, true)),
			)
			assert.Zerof(t, receiveOrderUpdates(t, ex)[0].ExecutedAmount, "wsHandleSubscriptionResponse should prevent incomplete cumulative execution after a %s snapshot", feed)
		})
	}
}

func TestApplyWebsocketOrderRow(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	now := time.Now()
	placed := time.UnixMilli(websocketOrderPlaced).UTC()
	opened := order.Detail{Price: 85000, Amount: 0.02, RemainingAmount: 0.02, OrderID: "1", Status: order.Open, Date: placed}
	cancelled := order.Detail{Price: 85000, Amount: 0.02, ExecutedAmount: 0.005, RemainingAmount: 0.015, OrderID: "1", Status: order.PartiallyFilledCancelled, Date: placed}
	orders := &ex.websocketWatermarks.orders

	apply := func(orderID uint64, row order.Detail) order.Detail {
		ex.applyWebsocketOrderRow(orderID, &row, now)
		return row
	}
	unfilled := order.Detail{Price: 85000, Amount: 0.02, RemainingAmount: 0.02, OrderID: "1", Status: order.Cancelled, Date: placed}
	assert.Equal(t, unfilled, apply(1, unfilled), "applyWebsocketOrderRow should relay the final row of an unrecorded order that executed nothing as converted")
	assert.Empty(t, orders.byOrderID, "applyWebsocketOrderRow should not record an order whose final row executed nothing")

	assert.Equal(t, opened, apply(1, opened), "applyWebsocketOrderRow should relay an order's first row as converted")
	record := orders.byOrderID[1]
	require.NotNil(t, record, "applyWebsocketOrderRow must record the order")
	exp := websocketOrder{orderID: 1, touched: now, rowSeen: true, originalSize: 0.02, limitPrice: 85000, placed: placed, covered: true}
	assert.Equal(t, exp, *record, "applyWebsocketOrderRow should record the row and cover an order whose first row shows nothing executed")

	executed, cost := 0.005, 425.0
	record.executed, record.cost, record.sizeTolerance = executed, cost, 0.000005
	enriched := cancelled
	enriched.Cost, enriched.AverageExecutedPrice = cost, cost/executed
	assert.Equal(t, enriched, apply(1, cancelled), "applyWebsocketOrderRow should add the cost and average price of the fills that cover a row's executed size")
	assert.Empty(t, orders.byOrderID, "applyWebsocketOrderRow should forget an order its final row ends")

	record = orders.touch(2, now)
	record.executed, record.cost, record.sizeTolerance = executed, cost, 0.000005
	ahead := opened
	ahead.OrderID = "2"
	cumulative := ahead
	cumulative.ExecutedAmount, cumulative.RemainingAmount, cumulative.Cost, cumulative.AverageExecutedPrice, cumulative.Status = executed, opened.Amount-executed, cost, cost/executed, order.PartiallyFilled
	assert.Equal(t, cumulative, apply(2, ahead), "applyWebsocketOrderRow should report the fills that arrived ahead of an order's first row, which they follow")
	short := cancelled
	short.OrderID, short.ExecutedAmount = "2", 0.004
	assert.Equal(t, short, apply(2, short), "applyWebsocketOrderRow should relay a final row its fills exceed as converted")
	assert.Empty(t, orders.byOrderID, "applyWebsocketOrderRow should forget an order whose final row its fills exceed")

	record = orders.touch(3, now)
	record.rowSeen, record.covered, record.originalSize, record.executed, record.cost, record.sizeTolerance = true, true, 0.02, 0.03, 2550, 0.000005
	excess := opened
	excess.OrderID = "3"
	assert.Equal(t, excess, apply(3, excess), "applyWebsocketOrderRow should relay a row as converted when the fills exceed the order's size")

	record = orders.touch(4, now)
	record.rowSeen, record.covered, record.executed, record.cost, record.sizeTolerance = true, true, 0.01, 850, 0.000005
	filled := order.Detail{Price: 85000, Amount: 0.02, ExecutedAmount: 0.02, OrderID: "4", Status: order.Filled, Date: placed}
	assert.Equal(t, filled, apply(4, filled), "applyWebsocketOrderRow should relay a final row its fills have yet to reach as converted")
	assert.Equal(t, &filled, record.final, "applyWebsocketOrderRow should keep a final row its fills have yet to reach")
	assert.Same(t, record, orders.byOrderID[4], "applyWebsocketOrderRow should keep the order of a final row its fills have yet to reach")

	record = orders.touch(5, now)
	record.rowSeen, record.executed, record.cost, record.sizeTolerance = true, 0.001, 85, 0.000005
	uncovered := cancelled
	uncovered.OrderID = "5"
	assert.Equal(t, uncovered, apply(5, uncovered), "applyWebsocketOrderRow should relay a row as converted when its order's fills miss executions")
	assert.Equal(t, &uncovered, record.final, "applyWebsocketOrderRow should keep a final row its fills have yet to reach though they missed executions, as fills reaching its executed size are all of the order's fills")
	assert.Same(t, record, orders.byOrderID[5], "applyWebsocketOrderRow should keep the order of a final row its fills have yet to reach")

	zeroSized := order.Detail{Price: 85000, OrderID: "6", Status: order.Open, Date: placed}
	record = orders.touch(6, now)
	record.executed, record.cost, record.sizeTolerance = 0.02, 1700, 0.000005
	assert.Equal(t, zeroSized, apply(6, zeroSized), "applyWebsocketOrderRow should relay a zero sized row as converted")

	unseen := cancelled
	unseen.OrderID = "7"
	assert.Equal(t, unseen, apply(7, unseen), "applyWebsocketOrderRow should relay the final row of an unrecorded order as converted")
	record = orders.byOrderID[7]
	require.NotNil(t, record, "applyWebsocketOrderRow must record an order first seen in a final row with an executed size")
	assert.Equal(t, &unseen, record.final, "applyWebsocketOrderRow should keep the final row of an order first seen in it until its fills arrive")

	record = orders.touch(8, now)
	record.rowSeen, record.originalSize, record.executed, record.cost, record.sizeTolerance = true, 0.02, executed, cost, 0.000005
	uncoveredAhead := opened
	uncoveredAhead.OrderID = "8"
	assert.Equal(t, uncoveredAhead, apply(8, uncoveredAhead), "applyWebsocketOrderRow should report no cumulative execution for an order whose fills may miss executions")
}

func TestApplyWebsocketOrderFill(t *testing.T) {
	t.Parallel()
	ex := &Exchange{Name: "TestApplyWebsocketOrderFill"}
	now := time.Now()
	placed := time.UnixMilli(websocketOrderPlaced).UTC()
	orders := &ex.websocketWatermarks.orders
	mapping := testPerpetualMapping
	source := func(orderID, tradeID uint64, price, size float64, offset int64, taker bool) *FillResponse {
		fee := -0.05
		if taker {
			fee = 0.1
		}
		return &FillResponse{Coin: "BTC", Price: types.Number(price), Size: types.Number(size), Side: "B", Time: milli(websocketOrderPlaced + offset), OrderID: orderID, Crossed: taker, Fee: types.Number(fee), TradeID: tradeID, ClientOrderID: websocketOrderClientID, FeeToken: currency.USDC}
	}
	tradeOnly := func(orderID string, tradeID uint64, price, size float64, offset int64, taker bool, date time.Time) order.Detail {
		trade := websocketOrderTrade(ex.Name, tradeID, price, size, websocketOrderPlaced+offset, taker)
		return order.Detail{
			Exchange:      ex.Name,
			OrderID:       orderID,
			ClientOrderID: websocketOrderClientID,
			Side:          order.Buy,
			AssetType:     asset.PerpetualContract,
			Date:          date,
			LastUpdated:   trade.Timestamp,
			Pair:          perpetualPair,
			Trades:        []order.TradeHistory{trade},
		}
	}

	update, ok := ex.applyWebsocketOrderFill(source(1, 11, 85000, 0.005, 100, true), &mapping, asset.PerpetualContract, order.Buy, now)
	require.True(t, ok, "applyWebsocketOrderFill must relay a new fill")
	assert.Equal(t, tradeOnly("1", 11, 85000, 0.005, 100, true, time.UnixMilli(websocketOrderPlaced+100).UTC()), update, "applyWebsocketOrderFill should relay the fill of an order no row has shown alone, dated by the fill")
	record := orders.byOrderID[1]
	require.NotNil(t, record, "applyWebsocketOrderFill must record the order")
	price1, size1, price2, size2 := 85000.0, 0.005, 85010.0, 0.015
	exp := websocketOrder{orderID: 1, touched: now, sizeTolerance: 0.000005, executed: size1, cost: price1 * size1, fee: 0.1, feeAsset: currency.USDC, tradeIDs: map[uint64]struct{}{11: {}}}
	assert.Equal(t, exp, *record, "applyWebsocketOrderFill should add the fill to the order's record")
	_, ok = ex.applyWebsocketOrderFill(source(1, 11, 85000, 0.005, 100, true), &mapping, asset.PerpetualContract, order.Buy, now)
	assert.False(t, ok, "applyWebsocketOrderFill should not relay a fill twice")
	assert.Equal(t, 0.005, record.executed, "applyWebsocketOrderFill should count a fill once")

	record.rowSeen, record.covered, record.originalSize, record.limitPrice, record.placed = true, true, 0.02, 85000, placed
	update, ok = ex.applyWebsocketOrderFill(source(1, 12, 85010, 0.015, 200, false), &mapping, asset.PerpetualContract, order.Buy, now)
	require.True(t, ok, "applyWebsocketOrderFill must relay a new fill")
	cost := float64(price1*size1) + float64(price2*size2)
	filled := tradeOnly("1", 12, price2, size2, 200, false, placed)
	filled.Price, filled.Amount, filled.ExecutedAmount, filled.Cost, filled.AverageExecutedPrice, filled.Status = 85000, 0.02, size1+size2, cost, cost/(size1+size2), order.Filled
	filled.Fee, filled.FeeAsset = 0.1+-0.05, currency.USDC
	assert.Equal(t, filled, update, "applyWebsocketOrderFill should report the cumulative execution and fees of an order whose fills cover it")

	update, ok = ex.applyWebsocketOrderFill(source(1, 13, 85020, 0.01, 300, true), &mapping, asset.PerpetualContract, order.Buy, now)
	require.True(t, ok, "applyWebsocketOrderFill must relay a new fill")
	assert.Equal(t, tradeOnly("1", 13, 85020, 0.01, 300, true, placed), update, "applyWebsocketOrderFill should relay a fill alone, dated by the placement, when the fills exceed the order's size")

	record = orders.touch(2, now)
	record.placed = placed
	record.final = &order.Detail{Price: 85000, Amount: 0.02, ExecutedAmount: 0.02, OrderID: "2", Status: order.Filled, Date: placed, LastUpdated: time.UnixMilli(websocketOrderPlaced + 300).UTC()}
	update, ok = ex.applyWebsocketOrderFill(source(2, 21, 85000, 0.01, 250, true), &mapping, asset.PerpetualContract, order.Buy, now)
	require.True(t, ok, "applyWebsocketOrderFill must relay a new fill")
	assert.Equal(t, tradeOnly("2", 21, 85000, 0.01, 250, true, placed), update, "applyWebsocketOrderFill should relay a fill alone while the fills have yet to reach a final row's executed size")
	update, ok = ex.applyWebsocketOrderFill(source(2, 22, 85010, 0.01, 400, false), &mapping, asset.PerpetualContract, order.Buy, now)
	require.True(t, ok, "applyWebsocketOrderFill must relay a new fill")
	price3, size3, price4, size4 := 85000.0, 0.01, 85010.0, 0.01
	cost = float64(price3*size3) + float64(price4*size4)
	completed := *record.final
	completed.Cost, completed.AverageExecutedPrice, completed.Fee, completed.FeeAsset = cost, cost/(size3+size4), 0.1+-0.05, currency.USDC
	completed.Trades = []order.TradeHistory{websocketOrderTrade(ex.Name, 22, 85010, 0.01, websocketOrderPlaced+400, false)}
	completed.LastUpdated = time.UnixMilli(websocketOrderPlaced + 400).UTC()
	assert.Equal(t, completed, update, "applyWebsocketOrderFill should complete a final row with the cost of the fills that reach its executed size, as last updated by the later of the two")
	assert.Nil(t, orders.byOrderID[2], "applyWebsocketOrderFill should forget an order once it completes its final row")

	record = orders.touch(3, now)
	record.rowSeen, record.covered, record.originalSize, record.limitPrice, record.placed = true, true, 0.03, 85000, placed
	_, ok = ex.applyWebsocketOrderFill(source(3, 31, 85000, 0.01, 100, true), &mapping, asset.PerpetualContract, order.Buy, now)
	require.True(t, ok, "applyWebsocketOrderFill must relay a new fill")
	otherToken := source(3, 32, 85000, 0.01, 200, false)
	otherToken.FeeToken = currency.HYPE
	update, ok = ex.applyWebsocketOrderFill(otherToken, &mapping, asset.PerpetualContract, order.Buy, now)
	require.True(t, ok, "applyWebsocketOrderFill must relay a new fill")
	assert.Equal(t, order.PartiallyFilled, update.Status, "applyWebsocketOrderFill should report the cumulative execution of fills paid in different tokens")
	assert.Zero(t, update.Fee, "applyWebsocketOrderFill should report no fee total for fills paid in different tokens")
	assert.True(t, update.FeeAsset.IsEmpty(), "applyWebsocketOrderFill should report no fee asset for fills paid in different tokens")
	update, ok = ex.applyWebsocketOrderFill(source(3, 33, 85000, 0.01, 300, true), &mapping, asset.PerpetualContract, order.Buy, now)
	require.True(t, ok, "applyWebsocketOrderFill must relay a new fill")
	assert.Equal(t, order.Filled, update.Status, "applyWebsocketOrderFill should report the cumulative execution of a later fill")
	assert.Zero(t, update.Fee, "applyWebsocketOrderFill should report no fee total once fills were paid in different tokens, whatever later fills pay")
	assert.True(t, record.feeUnknown, "applyWebsocketOrderFill should keep an order's fee total unknown once its fills were paid in different tokens")
	assert.Equal(t, 0.1, record.fee, "applyWebsocketOrderFill should stop summing fees once they were paid in different tokens")
}

func TestWebsocketOrderSetFillState(t *testing.T) {
	t.Parallel()
	originalSize, executed, cost := 0.02, 0.005, 425.0
	record := websocketOrder{originalSize: originalSize, limitPrice: 85000, executed: executed, cost: cost, fee: 0.15, feeAsset: currency.USDC, sizeTolerance: 0.000005}
	base := order.Detail{OrderID: "1"}
	exp := base
	exp.Price, exp.Amount, exp.ExecutedAmount, exp.RemainingAmount, exp.Cost, exp.AverageExecutedPrice, exp.Status = 85000, originalSize, executed, originalSize-executed, cost, cost/executed, order.PartiallyFilled
	exp.Fee, exp.FeeAsset = 0.15, currency.USDC
	update := base
	record.setFillState(&update)
	assert.Equal(t, exp, update, "setFillState should report a partly filled order's cumulative execution")

	executed, cost = 0.020001, 1700.085
	record.executed, record.cost = executed, cost
	exp.ExecutedAmount, exp.RemainingAmount, exp.Cost, exp.AverageExecutedPrice, exp.Status = executed, 0, cost, cost/executed, order.Filled
	update = base
	record.setFillState(&update)
	assert.Equal(t, exp, update, "setFillState should report an order filled when its fills are within half a size step of its size")

	record.executed = 0.03
	update = base
	record.setFillState(&update)
	assert.Equal(t, base, update, "setFillState should leave an update unchanged when the fills exceed the order's size")

	residual := websocketOrder{originalSize: 0.00008, limitPrice: 85000, sizeTolerance: 0.000005, feeAsset: currency.USDC}
	for _, size := range []float64{0.00001, 0.00007} {
		residual.executed += size
		residual.cost += float64(85000 * size)
	}
	require.Positive(t, residual.originalSize-residual.executed, "the fills must sum to just below the order's size, leaving a rounding residual")
	exp = base
	exp.Price, exp.Amount, exp.ExecutedAmount, exp.Cost, exp.AverageExecutedPrice, exp.Status = 85000, residual.originalSize, residual.executed, residual.cost, residual.cost/residual.executed, order.Filled
	exp.FeeAsset = currency.USDC
	update = base
	residual.setFillState(&update)
	assert.Equal(t, exp, update, "setFillState should report an order filled with nothing remaining when its fills fall short of its size only by rounding")

	mixedTokens := websocketOrder{originalSize: originalSize, limitPrice: 85000, executed: 0.005, cost: 425, fee: 0.15, feeAsset: currency.USDC, feeUnknown: true, sizeTolerance: 0.000005}
	update = base
	mixedTokens.setFillState(&update)
	assert.Equal(t, order.PartiallyFilled, update.Status, "setFillState should report the cumulative execution of fills paid in different tokens")
	assert.Zero(t, update.Fee, "setFillState should report no fee total for fills paid in different tokens")
	assert.True(t, update.FeeAsset.IsEmpty(), "setFillState should report no fee asset for fills paid in different tokens")
}

func TestWebsocketOrderSetFee(t *testing.T) {
	t.Parallel()
	record := websocketOrder{fee: -0.05, feeAsset: currency.USDC}
	update := order.Detail{OrderID: "1"}
	record.setFee(&update)
	assert.Equal(t, order.Detail{OrderID: "1", Fee: -0.05, FeeAsset: currency.USDC}, update, "setFee should report the fees of the order's fills")

	record.feeUnknown = true
	update = order.Detail{OrderID: "1"}
	record.setFee(&update)
	assert.Equal(t, order.Detail{OrderID: "1"}, update, "setFee should leave an update unchanged when the fills paid their fees in different tokens")
}

func TestUncoverWebsocketOrders(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.uncoverWebsocketOrders()
	now := time.Now()
	first, second := ex.websocketWatermarks.orders.touch(1, now), ex.websocketWatermarks.orders.touch(2, now)
	first.covered, second.covered = true, true
	ex.uncoverWebsocketOrders()
	assert.False(t, first.covered, "uncoverWebsocketOrders should mark every order's fills incomplete")
	assert.False(t, second.covered, "uncoverWebsocketOrders should mark every order's fills incomplete")
}

func TestWebsocketOrdersTouch(t *testing.T) {
	t.Parallel()
	var orders websocketOrders
	now := time.Now()
	first := orders.touch(1, now)
	second := orders.touch(2, now.Add(time.Second))
	assert.Equal(t, uint64(1), first.orderID, "touch should record the order")
	assert.Same(t, first, orders.oldest, "touch should keep the least recently touched record oldest")
	assert.Same(t, second, orders.newest, "touch should make a new record the newest")
	assert.Same(t, first, orders.touch(1, now.Add(2*time.Second)), "touch should return an order's existing record")
	assert.Same(t, second, orders.oldest, "touch should move a touched record from the oldest end")
	assert.Same(t, first, orders.newest, "touch should make a touched record the newest")
	assert.Equal(t, now.Add(2*time.Second), first.touched, "touch should mark the record touched")

	third := orders.touch(3, now.Add(2*time.Second+websocketOrderLifetime))
	assert.Equal(t, map[uint64]*websocketOrder{1: first, 3: third}, orders.byOrderID, "touch should forget a record untouched for its lifetime")
	orders.touch(4, now.Add(3*time.Second+websocketOrderLifetime))
	assert.Nil(t, orders.byOrderID[1], "touch should forget each record past its lifetime")

	var idle websocketOrders
	idled := idle.touch(1, now)
	idled.executed, idled.cost, idled.covered, idled.tradeIDs = 0.005, 425, true, map[uint64]struct{}{91: {}}
	assert.Same(t, idled, idle.touch(1, now.Add(websocketOrderLifetime)), "touch should keep a record touched within its lifetime")
	fresh := idle.touch(1, now.Add(2*websocketOrderLifetime+time.Second))
	assert.NotSame(t, idled, fresh, "touch should start a fresh record for an order idle past its lifetime")
	assert.Equal(t, websocketOrder{orderID: 1, touched: now.Add(2*websocketOrderLifetime + time.Second)}, *fresh, "touch should forget the executions, cost, coverage and trade IDs of an order idle past its lifetime")
	assert.Same(t, fresh, idle.byOrderID[1], "touch should hold the fresh record")
	assert.Same(t, fresh, idle.oldest, "touch should leave the fresh record alone in the touch order")

	var bounded websocketOrders
	for orderID := range uint64(maximumWebsocketOrders + 1) {
		bounded.touch(orderID, now)
	}
	assert.Len(t, bounded.byOrderID, maximumWebsocketOrders, "touch should hold no more records than the limit")
	assert.Nil(t, bounded.byOrderID[0], "touch should forget the least recently touched record beyond the limit")
	assert.NotNil(t, bounded.byOrderID[maximumWebsocketOrders], "touch should keep the record it touched")
}

func TestWebsocketOrdersRemove(t *testing.T) {
	t.Parallel()
	var orders websocketOrders
	now := time.Now()
	first, second, third := orders.touch(1, now), orders.touch(2, now), orders.touch(3, now)
	orders.remove(second)
	assert.Equal(t, map[uint64]*websocketOrder{1: first, 3: third}, orders.byOrderID, "remove should forget the record")
	assert.Same(t, third, first.newer, "remove should link the records either side of the removed one")
	orders.remove(first)
	orders.remove(third)
	assert.Empty(t, orders.byOrderID, "remove should forget every removed record")
	assert.Nil(t, orders.oldest, "remove should leave no oldest record")
	assert.Nil(t, orders.newest, "remove should leave no newest record")
}

func TestWebsocketOrdersUnlink(t *testing.T) {
	t.Parallel()
	var orders websocketOrders
	now := time.Now()
	first, second, third := orders.touch(1, now), orders.touch(2, now), orders.touch(3, now)
	orders.unlink(second)
	assert.Same(t, third, first.newer, "unlink should link the next newer record to the older one")
	assert.Same(t, first, third.older, "unlink should link the next older record to the newer one")
	assert.Nil(t, second.older, "unlink should clear the record's older link")
	assert.Nil(t, second.newer, "unlink should clear the record's newer link")
	assert.Same(t, second, orders.byOrderID[2], "unlink should keep the record held")
	orders.unlink(first)
	assert.Same(t, third, orders.oldest, "unlink should move the oldest end past an unlinked oldest record")
	orders.unlink(third)
	assert.Nil(t, orders.oldest, "unlink should leave no oldest record once the last is unlinked")
	assert.Nil(t, orders.newest, "unlink should leave no newest record once the last is unlinked")
}

func TestWebsocketHandleUserEvent(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.Error(t, ex.wsHandleUserEvent(t.Context(), []byte(`"invalid"`)), "wsHandleUserEvent should reject invalid data")
	require.NoError(t, ex.wsHandleData(t.Context(), accountConn(t, ex), []byte(`{"channel":"user","data":{"fills":[`+websocketFillJSON+`]}}`)), "wsHandleData must not error for fills")
	assert.Equal(t, []order.Detail{testWebsocketFillUpdate(ex.Name)}, receiveWebsocketData(t, ex), "wsHandleUserEvent should relay fills to the order stream")
	assert.Equal(t, []fill.Data{testWebsocketFillData(ex.Name)}, receiveWebsocketData(t, ex), "wsHandleUserEvent should relay fills to the fills feed")

	require.NoError(t, ex.wsHandleData(t.Context(), accountConn(t, ex), []byte(`{"channel":"user","data":{"twapSliceFills":[{"fill":`+websocketSpotFillJSON+`,"twapId":2278664}]}}`)), "wsHandleData must not error for TWAP slice fills")
	assert.Equal(t, []order.Detail{testWebsocketSpotFillUpdate(ex.Name)}, receiveWebsocketData(t, ex), "wsHandleUserEvent should relay TWAP slice fills to the order stream")
	assert.Equal(t, []fill.Data{testWebsocketSpotFillData(ex.Name)}, receiveWebsocketData(t, ex), "wsHandleUserEvent should relay TWAP slice fills to the fills feed")

	require.NoError(t, ex.wsHandleData(t.Context(), accountConn(t, ex), []byte(`{"channel":"user","data":{"funding":{"time":1791280800037,"coin":"BTC","usdc":"-0.601087","szi":"0.55919","fundingRate":"0.0000125","nSamples":null}}}`)), "wsHandleData must not error for a funding payment")
	assert.Equal(t, &WebsocketUserFunding{Time: milli(1791280800037), Coin: "BTC", USDC: -0.601087, SignedSize: 0.55919, FundingRate: 0.0000125}, receiveWebsocketData(t, ex), "wsHandleUserEvent should relay a funding payment")

	require.NoError(t, ex.wsHandleData(t.Context(), accountConn(t, ex), []byte(`{"channel":"user","data":{"liquidation":{"lid":123456,"liquidator":"0x0000000000000000000000000000000000000009","liquidated_user":"0x000000000000000000000000000000000000000b","liquidated_ntl_pos":"356.80015","liquidated_account_value":"2.956494"}}}`)), "wsHandleData must not error for a liquidation")
	assert.Equal(t, &WebsocketLiquidation{LiquidationID: 123456, Liquidator: "0x0000000000000000000000000000000000000009", LiquidatedUser: "0x000000000000000000000000000000000000000b", LiquidatedNotionalPosition: 356.80015, LiquidatedAccountValue: 2.956494}, receiveWebsocketData(t, ex), "wsHandleUserEvent should relay a liquidation")

	require.NoError(t, ex.wsHandleData(t.Context(), accountConn(t, ex), []byte(`{"channel":"user","data":{"nonUserCancel":[{"coin":"PNUT","oid":566615769247}]}}`)), "wsHandleData must not error for cancels")
	assert.Equal(t, []WebsocketNonUserCancel{{Coin: "PNUT", OrderID: 566615769247}}, receiveWebsocketData(t, ex), "wsHandleUserEvent should relay the orders the exchange cancelled")

	require.NoError(t, ex.wsHandleData(t.Context(), accountConn(t, ex), []byte(`{"channel":"user","data":{}}`)), "wsHandleData must not error for an empty event")
	assertNoWebsocketData(t, ex)
}

func TestWebsocketHandleUserFills(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.Error(t, ex.wsHandleUserFills(t.Context(), []byte(`"invalid"`)), "wsHandleUserFills should reject invalid data")
	require.NoError(t, ex.wsHandleData(t.Context(), accountConn(t, ex), []byte(`{"channel":"userFills","data":{"isSnapshot":true,"user":"0x000000000000000000000000000000000000000a","fills":[`+websocketFillJSON+`]}}`)), "wsHandleData must not error for the snapshot")
	assertNoWebsocketData(t, ex)

	require.NoError(t, ex.wsHandleData(t.Context(), accountConn(t, ex), []byte(`{"channel":"userFills","data":{"user":"0x000000000000000000000000000000000000000a","fills":[`+websocketFillJSON+`]}}`)), "wsHandleData must not error")
	assert.Equal(t, []order.Detail{testWebsocketFillUpdate(ex.Name)}, receiveWebsocketData(t, ex), "wsHandleUserFills should relay new fills to the order stream")
	assert.Equal(t, []fill.Data{testWebsocketFillData(ex.Name)}, receiveWebsocketData(t, ex), "wsHandleUserFills should relay new fills to the fills feed")

	require.NoError(t, ex.wsHandleData(t.Context(), accountConn(t, ex), []byte(`{"channel":"userFills","data":{"isSnapshot":true,"user":"0x000000000000000000000000000000000000000A","fills":[`+websocketLaterFillJSON+`,`+websocketFillJSON+`]}}`)), "wsHandleData must not error for a reconnect's snapshot")
	missedUpdate := testWebsocketFillUpdate(ex.Name)
	missedUpdate.Date, missedUpdate.LastUpdated = time.UnixMilli(1791280950000).UTC(), time.UnixMilli(1791280950000).UTC()
	missedUpdate.Trades[0].TID, missedUpdate.Trades[0].Timestamp = "669583051366497", time.UnixMilli(1791280950000).UTC()
	assert.Equal(t, []order.Detail{missedUpdate}, receiveWebsocketData(t, ex), "wsHandleUserFills should relay only the snapshot's fills after the last relayed one to the order stream")
	missed := testWebsocketFillData(ex.Name)
	missed.ID, missed.TradeID, missed.Timestamp = "669583051366497", "669583051366497", time.UnixMilli(1791280950000).UTC()
	assert.Equal(t, []fill.Data{missed}, receiveWebsocketData(t, ex), "wsHandleUserFills should relay only the snapshot's fills after the last relayed one to the fills feed")

	require.NoError(t, ex.wsHandleData(t.Context(), accountConn(t, ex), []byte(`{"channel":"userFills","data":{"isSnapshot":true,"user":"0x000000000000000000000000000000000000000b","fills":[`+websocketLaterFillJSON+`]}}`)), "wsHandleData must not error for another account's snapshot")
	assertNoWebsocketData(t, ex)

	// An aggregated fill carries the trade ID of one of the fills it combines, so it would be counted beside them
	aggregated := newWebsocketTestExchange(t)
	require.NoError(t, subscribeWebsocketTest(t, aggregated, subscription.List{{Channel: wsChannelUserFills, Authenticated: true, Params: map[string]any{"aggregateByTime": true}}}), "Subscribe must not error")
	require.NoError(t, aggregated.wsHandleData(t.Context(), accountConn(t, aggregated), []byte(`{"channel":"userFills","data":{"user":"`+testAccountAddress+`","fills":[`+websocketFillJSON+`]}}`)), "wsHandleData must not error for an aggregated fill")
	assert.Equal(t, []fill.Data{testWebsocketFillData(aggregated.Name)}, receiveWebsocketData(t, aggregated), "wsHandleUserFills should relay aggregated fills only to the fills feed")
	assertNoWebsocketData(t, aggregated)

	unheld, account := acknowledgeUserFillsLate(t, map[string]any{"aggregateByTime": true})
	require.NoError(t, unheld.wsHandleData(t.Context(), account, []byte(`{"channel":"userFills","data":{"user":"`+testAccountAddress+`","fills":[`+websocketFillJSON+`]}}`)), "wsHandleData must not error for an aggregated fill")
	assert.Equal(t, []fill.Data{testWebsocketFillData(unheld.Name)}, receiveWebsocketData(t, unheld), "wsHandleUserFills should relay the fills of an aggregated feed acknowledged late only to the fills feed")
	assertNoWebsocketData(t, unheld)
}

// acknowledgeUserFillsLate returns an exchange whose userFills subscribe with the options in params timed out, and
// which Hyperliquid then acknowledged, leaving the feed sending without a held subscription
func acknowledgeUserFillsLate(t *testing.T, params map[string]any) (*Exchange, *websocketConnectionFixture) {
	t.Helper()
	ex := newWebsocketTestExchange(t)
	account := accountConn(t, ex)
	account.mu.Lock()
	account.reply = nil
	account.mu.Unlock()
	limit := ex.WebsocketResponseMaxLimit
	ex.WebsocketResponseMaxLimit = time.Millisecond
	require.ErrorIs(t, subscribeWebsocketTest(t, ex, subscription.List{{Channel: wsChannelUserFills, Authenticated: true, Params: params}}), websocket.ErrSignatureTimeout, "Subscribe must time out without an acknowledgement")
	ex.WebsocketResponseMaxLimit = limit
	sent := account.sentRequests()
	require.NotEmpty(t, sent, "Subscribe must send the userFills subscribe")
	for _, frame := range acknowledge([]byte(sent[len(sent)-1])) {
		require.NoError(t, ex.wsHandleData(t.Context(), account, []byte(frame)), "wsHandleData must not error for the late acknowledgement")
	}
	require.Empty(t, ex.Websocket.GetSubscriptions(), "the late acknowledgement must leave the feed unheld")
	account.mu.Lock()
	account.reply = acknowledge
	account.mu.Unlock()
	return ex, account
}

func TestAggregatesUserFills(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.False(t, ex.aggregatesUserFills(testAccountAddress), "aggregatesUserFills should report no combining without a held userFills feed")
	newSubscribedWebsocketSubscription(t, ex, &subscription.Subscription{Channel: wsChannelUserFills, Authenticated: true, Params: map[string]any{"aggregateByTime": true}, QualifiedChannel: wsChannelUserFills})
	assert.False(t, ex.aggregatesUserFills(testAccountAddress), "aggregatesUserFills should ignore a held subscription without a subscribed feed")

	aggregated := newWebsocketTestExchange(t)
	require.NoError(t, subscribeWebsocketTest(t, aggregated, subscription.List{{Channel: wsChannelUserFills, Authenticated: true, Params: map[string]any{"aggregateByTime": true}}}), "Subscribe must not error")
	assert.True(t, aggregated.aggregatesUserFills(testAccountAddress), "aggregatesUserFills should report a held userFills feed that combines fills")
	assert.False(t, aggregated.aggregatesUserFills(testOtherAddress), "aggregatesUserFills should report no combining for another user")

	plain := newWebsocketTestExchange(t)
	require.NoError(t, subscribeWebsocketTest(t, plain, subscription.List{{Channel: wsChannelUserFills, Authenticated: true}}), "Subscribe must not error")
	assert.False(t, plain.aggregatesUserFills(testAccountAddress), "aggregatesUserFills should report a held userFills feed that sends each fill")

	plainLate, _ := acknowledgeUserFillsLate(t, nil)
	assert.False(t, plainLate.aggregatesUserFills(testAccountAddress), "aggregatesUserFills should report a feed acknowledged late that sends each fill")

	unheld, account := acknowledgeUserFillsLate(t, map[string]any{"aggregateByTime": true})
	assert.True(t, unheld.aggregatesUserFills(testAccountAddress), "aggregatesUserFills should report an aggregated feed acknowledged late")
	assert.False(t, unheld.aggregatesUserFills(testOtherAddress), "aggregatesUserFills should report no combining for another user")
	assert.ErrorIs(t, unheld.failWebsocketPending(account, errWebsocketServer), errWebsocketServer, "failWebsocketPending should return the cause")
	assert.ErrorIs(t, unheld.failWebsocketPending(publicConn(t, unheld), websocket.ErrNotConnected), websocket.ErrNotConnected, "failWebsocketPending should return the cause")
	assert.True(t, unheld.aggregatesUserFills(testAccountAddress), "aggregatesUserFills should keep an aggregated feed acknowledged late while its connection stays up")
	assert.ErrorIs(t, unheld.failWebsocketPending(account, websocket.ErrNotConnected), websocket.ErrNotConnected, "failWebsocketPending should return the cause")
	assert.False(t, unheld.aggregatesUserFills(testAccountAddress), "aggregatesUserFills should forget an aggregated feed acknowledged late once its connection is lost")

	aggregatedFeed := `{"type":"userFills","user":"` + testAccountAddress + `","aggregateByTime":true}`
	aggregatedFills := subscription.List{{Channel: wsChannelUserFills, Authenticated: true, Params: map[string]any{"aggregateByTime": true}}}
	stopped, account := acknowledgeUserFillsLate(t, map[string]any{"aggregateByTime": true})
	assert.ErrorIs(t, stopped.wsHandleData(t.Context(), account, websocketErrorFrame(t, wsErrorAlreadyUnsubscribed+aggregatedFeed)), errWebsocketServer, "wsHandleData should report an already unsubscribed answer naming no operation")
	assert.False(t, stopped.aggregatesUserFills(testAccountAddress), "aggregatesUserFills should forget an aggregated feed acknowledged late once Hyperliquid reports it unsubscribed")

	retried, account := acknowledgeUserFillsLate(t, map[string]any{"aggregateByTime": true})
	account.mu.Lock()
	account.reply = nil
	account.mu.Unlock()
	retried.WebsocketResponseMaxLimit = time.Millisecond
	require.ErrorIs(t, subscribeWebsocketTest(t, retried, aggregatedFills), websocket.ErrSignatureTimeout, "Subscribe must time out without an answer")
	require.NoError(t, retried.wsHandleData(t.Context(), account, websocketErrorFrame(t, wsErrorAlreadySubscribed+aggregatedFeed)), "wsHandleData must not error for a late already subscribed answer")
	assert.True(t, retried.aggregatesUserFills(testAccountAddress), "aggregatesUserFills should report an aggregated feed whose takeover Hyperliquid answered late")
	require.NoError(t, retried.wsHandleData(t.Context(), account, []byte(`{"channel":"subscriptionResponse","data":{"method":"unsubscribe","subscription":`+aggregatedFeed+`}}`)), "wsHandleData must not error for an unsubscribe acknowledgement naming no operation")
	assert.False(t, retried.aggregatesUserFills(testAccountAddress), "aggregatesUserFills should forget an aggregated feed once Hyperliquid acknowledges its unsubscribe")

	takenOver, account := acknowledgeUserFillsLate(t, map[string]any{"aggregateByTime": true})
	// Hyperliquid answers a subscribe of the feed the late acknowledgement left active as already subscribed
	alreadySubscribed := string(websocketErrorFrame(t, wsErrorAlreadySubscribed+aggregatedFeed))
	account.mu.Lock()
	account.reply = func(request []byte) []string {
		if strings.Contains(string(request), `"method":"subscribe"`) {
			return []string{alreadySubscribed}
		}
		return acknowledge(request)
	}
	account.mu.Unlock()
	require.NoError(t, subscribeWebsocketTest(t, takenOver, aggregatedFills), "Subscribe must take over the feed the late acknowledgement left active")
	require.NoError(t, unsubscribeWebsocketTest(t, takenOver, aggregatedFills), "Unsubscribe must not error")
	assert.False(t, takenOver.aggregatesUserFills(testAccountAddress), "aggregatesUserFills should forget an aggregated feed acknowledged late once it is unsubscribed")
}

func TestUnrelayedFills(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	position := func(f *FillResponse) websocketPosition {
		return websocketPosition{time: f.Time.Time(), tradeID: f.TradeID}
	}
	feed := websocketFillsFeed{channel: wsChannelUserFills, user: testAccountAddress}
	relayed := FillResponse{Time: milli(1791280942711), TradeID: 669583051366496}
	missed := FillResponse{Time: milli(1791280950000), TradeID: 669583051366497}
	assert.Empty(t, unrelayedFills(ex, feed, []FillResponse{relayed}, true, position), "unrelayedFills should relay none of the first snapshot")
	assert.Equal(t, []FillResponse{missed}, unrelayedFills(ex, feed, []FillResponse{relayed, missed}, true, position), "unrelayedFills should relay the snapshot's fills after the last relayed one")
	sliceFillsFeed := websocketFillsFeed{channel: wsChannelUserTWAPSliceFills, user: testAccountAddress}
	assert.Empty(t, unrelayedFills(ex, sliceFillsFeed, []FillResponse{missed}, true, position), "unrelayedFills should keep each feed's watermark apart")
	assert.Equal(t, []FillResponse{relayed}, unrelayedFills(ex, feed, []FillResponse{relayed}, false, position), "unrelayedFills should relay a live message")
}

func TestRelayFills(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	var spotFill FillResponse
	require.NoError(t, json.Unmarshal([]byte(websocketSpotFillJSON), &spotFill), "Unmarshal must not error")
	fills := []FillResponse{{Coin: "ETH", Side: "B"}, {Coin: "#90700", Side: "B", OrderID: 570870444900, TradeID: 1}, testWebsocketFill()}
	sliceFills := []TWAPSliceFillResponse{{Fill: FillResponse{Coin: "BTC", Side: "X"}, TWAPID: 7}, {Fill: spotFill, TWAPID: 2278664}}
	err := ex.relayFills(t.Context(), fills, sliceFills, true)
	assert.ErrorIs(t, err, errPairMappingNotFound, "relayFills should report a fill of an unknown coin")
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "relayFills should report a TWAP slice fill with an invalid side")
	assert.ErrorContains(t, err, "fill 0", "relayFills should identify the fill that failed")
	assert.ErrorContains(t, err, "TWAP 7 slice fill 0", "relayFills should identify the TWAP slice fill that failed")
	assert.NotContains(t, err.Error(), "fill 1", "relayFills should skip an outcome fill, which no supported asset holds")
	assert.Equal(t, []order.Detail{testWebsocketFillUpdate(ex.Name), testWebsocketSpotFillUpdate(ex.Name)}, receiveWebsocketData(t, ex), "relayFills should relay the fills that convert to the order stream")
	assert.Equal(t, []fill.Data{testWebsocketFillData(ex.Name), testWebsocketSpotFillData(ex.Name)}, receiveWebsocketData(t, ex), "relayFills should relay the fills that convert to the fills feed")

	require.NoError(t, ex.relayFills(t.Context(), []FillResponse{testWebsocketFill()}, nil, true), "relayFills must not error for a fill relayed before")
	assert.Equal(t, []fill.Data{testWebsocketFillData(ex.Name)}, receiveWebsocketData(t, ex), "relayFills should relay a fill relayed before to the fills feed only, as another feed delivered it to the order stream")

	later := testWebsocketFill()
	later.TradeID = 669583051366497
	require.NoError(t, ex.relayFills(t.Context(), []FillResponse{later}, nil, false), "relayFills must not error")
	laterData := testWebsocketFillData(ex.Name)
	laterData.ID, laterData.TradeID = "669583051366497", "669583051366497"
	assert.Equal(t, []fill.Data{laterData}, receiveWebsocketData(t, ex), "relayFills should relay fills kept from the order stream to the fills feed")
	assertNoWebsocketData(t, ex)

	assert.NoError(t, ex.relayFills(t.Context(), nil, nil, true), "relayFills should not error without fills")
	ex.SetFillsFeedStatus(false)
	require.NoError(t, ex.relayFills(t.Context(), []FillResponse{later}, nil, true), "relayFills must not error while the fills feed is disabled")
	laterUpdate := testWebsocketFillUpdate(ex.Name)
	laterUpdate.Trades[0].TID = "669583051366497"
	assert.Equal(t, []order.Detail{laterUpdate}, receiveWebsocketData(t, ex), "relayFills should relay fills to the order stream while the fills feed is disabled")
	assert.NoError(t, ex.relayFills(t.Context(), []FillResponse{{Coin: "ETH", Side: "B"}}, nil, false), "relayFills should ignore fills it relays nowhere")
	assertNoWebsocketData(t, ex)
}

func TestConvertFill(t *testing.T) {
	t.Parallel()
	ex := &Exchange{Name: "TestConvertFill"}
	source := testWebsocketFill()
	assert.Equal(t, testWebsocketFillData(ex.Name), ex.convertFill(&source, perpetualPair, asset.PerpetualContract, order.Buy), "convertFill should identify a fill by its trade ID")
}

func TestWebsocketHandleUserFundings(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.Error(t, ex.wsHandleUserFundings(t.Context(), []byte(`"invalid"`)), "wsHandleUserFundings should reject invalid data")
	frame := `{"channel":"userFundings","data":{"isSnapshot":true,"user":"0x0000000000000000000000000000000000000003","fundings":[{"time":1791216000037,"coin":"ENA","usdc":"-2.38878","szi":"777946.0","fundingRate":"0.0000125","nSamples":1},{"time":1791219600012,"coin":"BTC","usdc":"0.238633","szi":"-0.2238","fundingRate":"0.0000125","nSamples":null}]}}`
	require.NoError(t, ex.wsHandleData(t.Context(), accountConn(t, ex), []byte(frame)), "wsHandleData must not error")
	exp := &WebsocketUserFundings{
		IsSnapshot: true,
		User:       "0x0000000000000000000000000000000000000003",
		Fundings: []WebsocketUserFunding{
			{Time: milli(1791216000037), Coin: "ENA", USDC: -2.38878, SignedSize: 777946, FundingRate: 0.0000125, NumberOfSamples: 1},
			{Time: milli(1791219600012), Coin: "BTC", USDC: 0.238633, SignedSize: -0.2238, FundingRate: 0.0000125},
		},
	}
	assert.Equal(t, exp, receiveWebsocketData(t, ex), "wsHandleUserFundings should relay the funding payments")
	require.NoError(t, ex.wsHandleData(t.Context(), accountConn(t, ex), []byte(`{"channel":"userFundings","data":{"user":"0x0000000000000000000000000000000000000003","fundings":[]}}`)), "wsHandleData must not error without payments")
	assertNoWebsocketData(t, ex)
}

func TestWebsocketHandleUserNonFundingLedgerUpdates(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.Error(t, ex.wsHandleUserNonFundingLedgerUpdates(t.Context(), []byte(`"invalid"`)), "wsHandleUserNonFundingLedgerUpdates should reject invalid data")
	frame := `{"channel":"userNonFundingLedgerUpdates","data":{"isSnapshot":true,"user":"0x0000000000000000000000000000000000000001","nonFundingLedgerUpdates":[{"time":1748135719014,"hash":"0x2674629937983fbe452ec47df9c991d29c136ff7c79c3c1630d600d6707524d0","delta":{"type":"deposit","usdc":"200079.847397"}},{"time":1677618623525,"hash":"0x651383a8fda3daaf31cc0308597c0000f7f51388f97acf0df28797ee97ae278a","delta":{"type":"liquidation","liquidatedNtlPos":"356.80015","accountValue":"2.956494","leverageType":"Cross","liquidatedPositions":[{"coin":"ATOM","szi":"29.21"}]}}]}}`
	require.NoError(t, ex.wsHandleData(t.Context(), accountConn(t, ex), []byte(frame)), "wsHandleData must not error")
	exp := &WebsocketUserNonFundingLedgerUpdates{
		IsSnapshot: true,
		User:       "0x0000000000000000000000000000000000000001",
		NonFundingLedgerUpdates: []UserLedgerUpdateResponse{
			{Delta: UserLedgerDelta{Type: "deposit", USDC: 200079.847397}, Hash: "0x2674629937983fbe452ec47df9c991d29c136ff7c79c3c1630d600d6707524d0", Time: milli(1748135719014)},
			{
				Delta: UserLedgerDelta{
					Type:                       "liquidation",
					AccountValue:               2.956494,
					LeverageType:               "Cross",
					LiquidatedNotionalPosition: 356.80015,
					LiquidatedPositions:        []LiquidatedPosition{{Coin: "ATOM", SignedSize: 29.21}},
				},
				Hash: "0x651383a8fda3daaf31cc0308597c0000f7f51388f97acf0df28797ee97ae278a",
				Time: milli(1677618623525),
			},
		},
	}
	assert.Equal(t, exp, receiveWebsocketData(t, ex), "wsHandleUserNonFundingLedgerUpdates should relay the ledger updates")
	require.NoError(t, ex.wsHandleData(t.Context(), accountConn(t, ex), []byte(`{"channel":"userNonFundingLedgerUpdates","data":{"user":"0x0000000000000000000000000000000000000001","nonFundingLedgerUpdates":[]}}`)), "wsHandleData must not error without updates")
	assertNoWebsocketData(t, ex)
}

func TestWebsocketHandlePerpetualTicker(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.Error(t, ex.wsHandlePerpetualTicker(t.Context(), []byte(`"invalid"`)), "wsHandlePerpetualTicker should reject invalid data")
	frame := `{"channel":"activeAssetCtx","data":{"coin":"BTC","ctx":{"funding":"0.0000125","openInterest":"39980.76392","prevDayPx":"85840.0","dayNtlVlm":"1821282705.4506571293","premium":"-0.0001864194","oraclePx":"85828.0","markPx":"85814.0","midPx":"85811.5","impactPxs":["85811.0","85812.0"],"dayBaseVlm":"21229.55909"}}}`
	require.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(frame)), "wsHandleData must not error")
	exp := &ticker.Price{
		Last:         85811.5,
		Open:         85840,
		BaseVolume:   21229.55909,
		QuoteVolume:  1821282705.4506571293,
		OpenInterest: 39980.76392,
		MarkPrice:    85814,
		IndexPrice:   85828,
		Pair:         perpetualPair,
		ExchangeName: ex.Name,
		AssetType:    asset.PerpetualContract,
	}
	assert.Equal(t, exp, receiveTicker(t, ex), "wsHandlePerpetualTicker should relay the converted ticker")
	stored, err := ticker.GetTicker(ex.Name, perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetTicker must not error")
	assert.Equal(t, 85811.5, stored.Last, "wsHandlePerpetualTicker should store the ticker")

	err = ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"activeAssetCtx","data":{"coin":"@107","ctx":{}}}`))
	assert.ErrorIs(t, err, errWebsocketAssetMismatch, "wsHandlePerpetualTicker should reject a spot coin")
	err = ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"activeAssetCtx","data":{"coin":"ETH","ctx":{}}}`))
	assert.ErrorIs(t, err, errPairMappingNotFound, "wsHandlePerpetualTicker should reject an unknown coin")
	ex.setPairMappings(asset.PerpetualContract, []pairMapping{{coin: "UNPAIRED"}})
	err = ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"activeAssetCtx","data":{"coin":"UNPAIRED","ctx":{"markPx":"1.0"}}}`))
	assert.Error(t, err, "wsHandlePerpetualTicker should return the error of a ticker that cannot be stored")
	assertNoWebsocketData(t, ex)
}

func TestWebsocketHandleSpotTicker(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.Error(t, ex.wsHandleSpotTicker(t.Context(), []byte(`"invalid"`)), "wsHandleSpotTicker should reject invalid data")
	frame := `{"channel":"activeSpotAssetCtx","data":{"coin":"@107","ctx":{"prevDayPx":"93.124","dayNtlVlm":"64552049.6973299459","markPx":"92.949","midPx":"92.945","circulatingSupply":"298462370.4079165459","coin":"@107","totalSupply":"998882916.6303099394","dayBaseVlm":"690824.1400000008"}}}`
	require.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(frame)), "wsHandleData must not error")
	exp := &ticker.Price{
		Last:         92.945,
		Open:         93.124,
		BaseVolume:   690824.1400000008,
		QuoteVolume:  64552049.6973299459,
		MarkPrice:    92.949,
		Pair:         spotPair,
		ExchangeName: ex.Name,
		AssetType:    asset.Spot,
	}
	assert.Equal(t, exp, receiveTicker(t, ex), "wsHandleSpotTicker should relay the converted ticker")
	stored, err := ticker.GetTicker(ex.Name, spotPair, asset.Spot)
	require.NoError(t, err, "GetTicker must not error")
	assert.Equal(t, 92.945, stored.Last, "wsHandleSpotTicker should store the ticker")

	err = ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"activeSpotAssetCtx","data":{"coin":"BTC","ctx":{}}}`))
	assert.ErrorIs(t, err, errWebsocketAssetMismatch, "wsHandleSpotTicker should reject a perpetual coin")
	err = ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"activeSpotAssetCtx","data":{"coin":"@1","ctx":{}}}`))
	assert.ErrorIs(t, err, errPairMappingNotFound, "wsHandleSpotTicker should reject an unknown coin")
	ex.setPairMappings(asset.Spot, []pairMapping{{coin: "@999"}})
	err = ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"activeSpotAssetCtx","data":{"coin":"@999","ctx":{"markPx":"1.0"}}}`))
	assert.Error(t, err, "wsHandleSpotTicker should return the error of a ticker that cannot be stored")
	assertNoWebsocketData(t, ex)
}

func TestWebsocketHandleUserTWAPSliceFills(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.Error(t, ex.wsHandleUserTWAPSliceFills(t.Context(), []byte(`"invalid"`)), "wsHandleUserTWAPSliceFills should reject invalid data")
	require.NoError(t, ex.wsHandleData(t.Context(), accountConn(t, ex), []byte(`{"channel":"userTwapSliceFills","data":{"isSnapshot":true,"user":"0x0000000000000000000000000000000000000007","twapSliceFills":[{"fill":`+websocketSpotFillJSON+`,"twapId":2278664}]}}`)), "wsHandleData must not error for the snapshot")
	assertNoWebsocketData(t, ex)

	require.NoError(t, ex.wsHandleData(t.Context(), accountConn(t, ex), []byte(`{"channel":"userTwapSliceFills","data":{"user":"0x0000000000000000000000000000000000000007","twapSliceFills":[{"fill":`+websocketSpotFillJSON+`,"twapId":2278664}]}}`)), "wsHandleData must not error")
	assert.Equal(t, []order.Detail{testWebsocketSpotFillUpdate(ex.Name)}, receiveWebsocketData(t, ex), "wsHandleUserTWAPSliceFills should relay new TWAP slice fills to the order stream")
	assert.Equal(t, []fill.Data{testWebsocketSpotFillData(ex.Name)}, receiveWebsocketData(t, ex), "wsHandleUserTWAPSliceFills should relay new TWAP slice fills to the fills feed")

	require.NoError(t, ex.wsHandleData(t.Context(), accountConn(t, ex), []byte(`{"channel":"userTwapSliceFills","data":{"isSnapshot":true,"user":"0x0000000000000000000000000000000000000007","twapSliceFills":[{"fill":`+websocketSpotFillJSON+`,"twapId":2278664},{"fill":`+websocketLaterSpotFillJSON+`,"twapId":2278664}]}}`)), "wsHandleData must not error for a reconnect's snapshot")
	missedUpdate := testWebsocketSpotFillUpdate(ex.Name)
	missedUpdate.Date, missedUpdate.LastUpdated = time.UnixMilli(1791279180013).UTC(), time.UnixMilli(1791279180013).UTC()
	missedUpdate.Trades[0].TID, missedUpdate.Trades[0].Timestamp = "499584618036467", time.UnixMilli(1791279180013).UTC()
	assert.Equal(t, []order.Detail{missedUpdate}, receiveWebsocketData(t, ex), "wsHandleUserTWAPSliceFills should relay only the snapshot's slice fills after the last relayed one to the order stream")
	missed := testWebsocketSpotFillData(ex.Name)
	missed.ID, missed.TradeID, missed.Timestamp = "499584618036467", "499584618036467", time.UnixMilli(1791279180013).UTC()
	assert.Equal(t, []fill.Data{missed}, receiveWebsocketData(t, ex), "wsHandleUserTWAPSliceFills should relay only the snapshot's slice fills after the last relayed one to the fills feed")
}

func TestWebsocketHandleUserTWAPHistory(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.Error(t, ex.wsHandleUserTWAPHistory(t.Context(), []byte(`"invalid"`)), "wsHandleUserTWAPHistory should reject invalid data")
	frame := `{"channel":"userTwapHistory","data":{"isSnapshot":true,"user":"0x0000000000000000000000000000000000000007","history":[{"time":1769165674,"state":{"coin":"@107","user":"0x0000000000000000000000000000000000000007","side":"B","sz":"2000.0","executedSz":"2000.0","executedNtl":"41959.56859","minutes":30,"reduceOnly":true,"randomize":true,"timestamp":1769163872991,"trigger":{"px":"88.0","above":true},"stopPx":"85.5"},"status":{"status":"finished","description":"finished"},"twapId":1530771},{"time":1770013630,"state":{"coin":"@151","user":"0x0000000000000000000000000000000000000007","side":"B","sz":"86.0","executedSz":"49.0413","executedNtl":"107736.21719","minutes":60,"reduceOnly":false,"randomize":false,"timestamp":1770011558151,"trigger":null,"stopPx":null},"status":{"status":"error","description":"Insufficient spot balance"},"twapId":1560286}]}}`
	require.NoError(t, ex.wsHandleData(t.Context(), accountConn(t, ex), []byte(frame)), "wsHandleData must not error")
	exp := &WebsocketUserTWAPHistory{
		IsSnapshot: true,
		User:       "0x0000000000000000000000000000000000000007",
		History: []TWAPHistoryEntryResponse{
			{
				Time: types.Time(time.Unix(1769165674, 0)),
				State: TWAPState{
					Coin:             "@107",
					User:             "0x0000000000000000000000000000000000000007",
					Side:             "B",
					Size:             2000,
					ExecutedSize:     2000,
					ExecutedNotional: 41959.56859,
					Minutes:          30,
					ReduceOnly:       true,
					Randomise:        true,
					Timestamp:        milli(1769163872991),
					Trigger:          &TWAPTrigger{Price: 88, Above: true},
					StopPrice:        85.5,
				},
				Status: TWAPStatus{Status: "finished", Description: "finished"},
				TWAPID: 1530771,
			},
			{
				Time: types.Time(time.Unix(1770013630, 0)),
				State: TWAPState{
					Coin:             "@151",
					User:             "0x0000000000000000000000000000000000000007",
					Side:             "B",
					Size:             86,
					ExecutedSize:     49.0413,
					ExecutedNotional: 107736.21719,
					Minutes:          60,
					Timestamp:        milli(1770011558151),
				},
				Status: TWAPStatus{Status: "error", Description: "Insufficient spot balance"},
				TWAPID: 1560286,
			},
		},
	}
	assert.Equal(t, exp, receiveWebsocketData(t, ex), "wsHandleUserTWAPHistory should relay the TWAP status changes")
	require.NoError(t, ex.wsHandleData(t.Context(), accountConn(t, ex), []byte(`{"channel":"userTwapHistory","data":{"user":"0x0000000000000000000000000000000000000007","history":[]}}`)), "wsHandleData must not error without changes")
	assertNoWebsocketData(t, ex)
}

func TestWebsocketHandleOutcomeMetaUpdates(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.Error(t, ex.wsHandleOutcomeMetaUpdates(t.Context(), []byte(`"invalid"`)), "wsHandleOutcomeMetaUpdates should reject invalid data")
	frame := `{"channel":"outcomeMetaUpdates","data":[{"outcomeCreated":{"outcome":9115,"name":"template:binaryPrice","description":"perp:HYPE|priceDescription:HYPE-USDC perp mark|seconds:60|threshold:93.058|time:20261006-1015","sideSpecs":[{"name":"template:Yes"},{"name":"template:No"}],"quoteToken":"USDC","venue":"out","deployerFeeScale":"1.0"}},{"outcomeSettled":9112},{"questionUpdated":{"question":400,"name":"template:sportsContestResult","description":"competition:English Premier League|contestType:Match|participantA:Coventry City|participantB:Newcastle United","fallbackOutcome":9068,"namedOutcomes":[9069,9070,9071],"settledNamedOutcomes":[9069]}},{"questionSettled":400}]}`
	require.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(frame)), "wsHandleData must not error")
	exp := []WebsocketOutcomeMetaUpdate{
		{OutcomeCreated: &OutcomeSpecification{
			Outcome:            9115,
			Name:               "template:binaryPrice",
			Description:        "perp:HYPE|priceDescription:HYPE-USDC perp mark|seconds:60|threshold:93.058|time:20261006-1015",
			SideSpecifications: []OutcomeSideSpecification{{Name: "template:Yes"}, {Name: "template:No"}},
			QuoteToken:         currency.USDC,
			Venue:              "out",
			DeployerFeeScale:   1,
		}},
		{OutcomeSettled: new(uint64(9112))},
		{QuestionUpdated: &OutcomeQuestion{
			Question:             400,
			Name:                 "template:sportsContestResult",
			Description:          "competition:English Premier League|contestType:Match|participantA:Coventry City|participantB:Newcastle United",
			FallbackOutcome:      9068,
			NamedOutcomes:        []uint64{9069, 9070, 9071},
			SettledNamedOutcomes: []uint64{9069},
		}},
		{QuestionSettled: new(uint64(400))},
	}
	assert.Equal(t, exp, receiveWebsocketData(t, ex), "wsHandleOutcomeMetaUpdates should relay the outcome changes")
	require.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"outcomeMetaUpdates","data":[]}`)), "wsHandleData must not error without changes")
	assertNoWebsocketData(t, ex)
}

func TestWebsocketHandleFastAssetContexts(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	for _, tc := range []struct {
		name string
		data string
		exp  map[string]WebsocketFastAssetContext
	}{
		{
			name: "snapshot",
			data: `"q1ZyCnFWsqpWyk0syg6oULJSsjC1MDTRM1DSUcrNTIGLGOqZKtXqKDkYGpijqLY00rM0sURSDBYAq1U2NLGwMEBRbaBnaAgUNUVSDxcC6qiorLKKiIwyNEDVZWxoaGaO4iKgiLkByEW1AA=="`,
			exp: map[string]WebsocketFastAssetContext{
				"BTC":        {MarkPrice: 85814, MidPrice: 85811.5, MarkPriceSet: true, MidPriceSet: true},
				"@107":       {MarkPrice: 92.949, MidPrice: 92.945, MarkPriceSet: true, MidPriceSet: true},
				"#14880":     {MarkPrice: 0.111485, MidPrice: 0.111485, MarkPriceSet: true, MidPriceSet: true},
				"xyz:XYZ100": {MarkPrice: 31167, MidPrice: 31170.5, MarkPriceSet: true, MidPriceSet: true},
			},
		},
		{
			name: "update",
			data: `"q1ZyCnFWsqpWys1MCahQslKyMLUwNNIzVarVUXIwNDAHSyUWZYPlLI30LMEyGZV5iVauIR4oskYmBgZ6lko6MKPySnNyamsB"`,
			exp: map[string]WebsocketFastAssetContext{
				"BTC":      {MidPrice: 85812.5, MidPriceSet: true},
				"@107":     {MarkPrice: 92.95, MarkPriceSet: true},
				"hyna:ETH": {MarkPrice: 2400.9, MarkPriceSet: true, MidPriceSet: true},
			},
		},
		{
			name: "maximum sized message",
			data: `"` + deflateBase64(t, append([]byte(`{}`), bytes.Repeat([]byte(" "), maximumFastAssetContextsSize-2)...)) + `"`,
			exp:  map[string]WebsocketFastAssetContext{},
		},
	} {
		require.NoErrorf(t, ex.wsHandleFastAssetContexts(t.Context(), []byte(tc.data)), "wsHandleFastAssetContexts must decode the %s", tc.name)
		assert.Equalf(t, tc.exp, receiveWebsocketData(t, ex), "wsHandleFastAssetContexts should relay the %s", tc.name)
	}

	for _, tc := range []struct {
		name string
		data string
		err  error
	}{
		{name: "an object", data: `{}`},
		{name: "invalid base64", data: `"!!!"`},
		{name: "invalid DEFLATE data", data: `"` + base64.StdEncoding.EncodeToString([]byte("not compressed")) + `"`},
		{name: "a JSON array", data: `"` + deflateBase64(t, []byte(`[1]`)) + `"`},
		{name: "an oversized message", data: `"` + deflateBase64(t, bytes.Repeat([]byte(" "), maximumFastAssetContextsSize+1)) + `"`, err: errWebsocketServer},
	} {
		err := ex.wsHandleFastAssetContexts(t.Context(), []byte(tc.data))
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "wsHandleFastAssetContexts should reject %s", tc.name)
		} else {
			assert.Errorf(t, err, "wsHandleFastAssetContexts should reject %s", tc.name)
		}
	}
	assertNoWebsocketData(t, ex)
}

func TestTWAPStateEntryUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var entry TWAPStateEntry
	require.NoError(t, json.Unmarshal([]byte(`[2285948,{"coin":"@107","minutes":4320,"trigger":{"px":"90.0","above":true}}]`), &entry), "Unmarshal must not error")
	exp := TWAPStateEntry{TWAPID: 2285948, State: TWAPState{Coin: "@107", Minutes: 4320, Trigger: &TWAPTrigger{Price: 90, Above: true}}}
	assert.Equal(t, exp, entry, "TWAPStateEntry should decode its TWAP ID and state")
	assert.ErrorIs(t, json.Unmarshal([]byte(`[2285948]`), &entry), errInvalidTupleLength, "TWAPStateEntry should reject a short tuple")
}

func TestDEXClearinghouseStateUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var state DEXClearinghouseState
	require.NoError(t, json.Unmarshal([]byte(`["xyz",`+websocketEmptyClearinghouseStateJSON+`]`), &state), "Unmarshal must not error")
	assert.Equal(t, DEXClearinghouseState{DEX: "xyz", ClearinghouseState: testWebsocketEmptyClearinghouseState()}, state, "DEXClearinghouseState should decode its DEX and state")
	assert.ErrorIs(t, json.Unmarshal([]byte(`["xyz"]`), &state), errInvalidTupleLength, "DEXClearinghouseState should reject a short tuple")
}

func TestDEXAssetContextsUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var contexts DEXAssetContexts
	require.NoError(t, json.Unmarshal([]byte(`["xyz",[{"funding":"0.00000625","openInterest":"7177.89","markPx":"31164.0","impactPxs":["31164.0","31165.33"]}]]`), &contexts), "Unmarshal must not error")
	exp := DEXAssetContexts{DEX: "xyz", AssetContexts: []PerpetualAssetContext{{Funding: 0.00000625, OpenInterest: 7177.89, MarkPrice: 31164, ImpactPrices: []types.Number{31164, 31165.33}}}}
	assert.Equal(t, exp, contexts, "DEXAssetContexts should decode its DEX and asset contexts")
	assert.ErrorIs(t, json.Unmarshal([]byte(`["xyz"]`), &contexts), errInvalidTupleLength, "DEXAssetContexts should reject a short tuple")
}

func TestWebsocketPositionAfter(t *testing.T) {
	t.Parallel()
	position := websocketPosition{time: time.UnixMilli(1791278878126), tradeID: 200}
	for _, tc := range []struct {
		other websocketPosition
		exp   bool
	}{
		{other: websocketPosition{time: time.UnixMilli(1791278878125), tradeID: 900}, exp: true},
		{other: websocketPosition{time: time.UnixMilli(1791278878126), tradeID: 100}, exp: true},
		{other: websocketPosition{time: time.UnixMilli(1791278878126), tradeID: 200}},
		{other: websocketPosition{time: time.UnixMilli(1791278878126), tradeID: 300}},
		{other: websocketPosition{time: time.UnixMilli(1791278878127)}},
	} {
		assert.Equalf(t, tc.exp, position.after(tc.other), "after should order %v against %v by time, then trade ID", position, tc.other)
	}
}

func TestWebsocketFastAssetContextUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		data string
		exp  WebsocketFastAssetContext
	}{
		{data: `{"markPx":"85814.0","midPx":"85811.5"}`, exp: WebsocketFastAssetContext{MarkPrice: 85814, MidPrice: 85811.5, MarkPriceSet: true, MidPriceSet: true}},
		{data: `{"midPx":"85812.5"}`, exp: WebsocketFastAssetContext{MidPrice: 85812.5, MidPriceSet: true}},
		{data: `{"markPx":"2400.9","midPx":null}`, exp: WebsocketFastAssetContext{MarkPrice: 2400.9, MarkPriceSet: true, MidPriceSet: true}},
		{data: `{}`},
	} {
		var fastContext WebsocketFastAssetContext
		require.NoErrorf(t, json.Unmarshal([]byte(tc.data), &fastContext), "Unmarshal must not error for %s", tc.data)
		assert.Equalf(t, tc.exp, fastContext, "Unmarshal should record which prices %s sets", tc.data)
	}
	for _, data := range []string{`[]`, `{"markPx":true}`, `{"midPx":{}}`} {
		var fastContext WebsocketFastAssetContext
		assert.Errorf(t, json.Unmarshal([]byte(data), &fastContext), "Unmarshal should reject %s", data)
	}
}

// replyToPost returns a reply that answers a post request with the response, echoing the request's ID
func replyToPost(response string) func([]byte) []string {
	return func(request []byte) []string {
		var post WebsocketPostRequest
		if err := json.Unmarshal(request, &post); err != nil {
			return nil
		}
		return []string{`{"channel":"post","data":{"id":` + strconv.FormatUint(post.ID, 10) + `,"response":` + response + `}}`}
	}
}

func TestWebsocketAPIPostInfo(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, e.WebsocketAPI().PostInfo(t.Context(), nil, new(L2BookResponse)), common.ErrNilPointer, "WebsocketAPI.PostInfo should require a payload")
	assert.ErrorIs(t, e.WebsocketAPI().PostInfo(t.Context(), &InfoRequest{Type: "l2Book", Coin: "ETH"}, nil), common.ErrNilPointer, "WebsocketAPI.PostInfo should require a result")

	venue, requests := newVenueWebsocketTestExchange(t)
	var book *L2BookResponse
	require.NoError(t, venue.WebsocketAPI().PostInfo(t.Context(), &InfoRequest{Type: "l2Book", Coin: "ETH", NumberOfSignificantFigures: 5}, &book), "WebsocketAPI.PostInfo must not error")
	if mockTests {
		exp := &L2BookResponse{
			Coin:   "ETH",
			Time:   milli(1791279145054),
			Levels: [][]L2Level{{{Price: 2702.5, Size: 138.1974, NumberOfOrders: 17}}, {{Price: 2702.6, Size: 84.4867, NumberOfOrders: 17}}},
			Spread: 0.1,
		}
		assert.Equal(t, exp, book, "WebsocketAPI.PostInfo should decode the info response")
		assert.Equal(t, []string{`{"method":"post","id":1,"request":{"type":"info","payload":{"type":"l2Book","coin":"ETH","nSigFigs":5}}}`}, requests.received(), "WebsocketAPI.PostInfo should post the info request")
	} else {
		require.NotNil(t, book, "WebsocketAPI.PostInfo must return the ETH book")
		require.Len(t, book.Levels, 2, "WebsocketAPI.PostInfo must return both sides of the ETH book")
		assert.NotEmpty(t, book.Levels[0], "WebsocketAPI.PostInfo should return the ETH bids")
		assert.NotEmpty(t, book.Levels[1], "WebsocketAPI.PostInfo should return the ETH asks")
		assert.Positive(t, book.Spread.Float64(), "WebsocketAPI.PostInfo should return the aggregated book's spread")
	}
	book = nil
	require.NoError(t, venue.WebsocketAPI().PostInfo(t.Context(), &InfoRequest{Type: "l2Book", Coin: "NOTACOIN"}, &book), "WebsocketAPI.PostInfo must not error for an unknown coin")
	assert.Nil(t, book, "WebsocketAPI.PostInfo should leave the result unset for an unknown coin")
	assert.ErrorIs(t, venue.WebsocketAPI().PostInfo(t.Context(), &InfoRequest{Type: "l2Book", Coin: "BTC", NumberOfSignificantFigures: 7}, &book), errWebsocketPost, "WebsocketAPI.PostInfo should return a rejected request's error")

	// The venue sends only well-formed info payloads, so a fixture connection supplies malformed ones
	ex := newWebsocketTestExchange(t)
	connection := publicConn(t, ex)
	connection.reply = replyToPost(`{"type":"info","payload":"invalid"}`)
	assert.ErrorContains(t, ex.WebsocketAPI().PostInfo(t.Context(), &InfoRequest{Type: "l2Book", Coin: "ETH"}, &book), "decoding info post payload", "WebsocketAPI.PostInfo should reject an invalid info payload")
	connection.reply = replyToPost(`{"type":"info","payload":{"type":"l2Book","data":"invalid"}}`)
	assert.ErrorContains(t, ex.WebsocketAPI().PostInfo(t.Context(), &InfoRequest{Type: "l2Book", Coin: "ETH"}, &book), "decoding l2Book info response", "WebsocketAPI.PostInfo should reject a response that does not fit the result")
	connection.reply = replyToPost(`{"type":"info","payload":null}`)
	assert.ErrorIs(t, ex.WebsocketAPI().PostInfo(t.Context(), &InfoRequest{Type: "l2Book", Coin: "ETH"}, &book), common.ErrNoResponse, "WebsocketAPI.PostInfo should reject an empty info payload")
}

func TestWebsocketAPIPlaceOrders(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAPI().PlaceOrders(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "WebsocketAPI.PlaceOrders should reject a nil request")
	_, err = e.WebsocketAPI().PlaceOrders(t.Context(), &PlaceOrdersRequest{})
	assert.ErrorIs(t, err, errNoActionItems, "WebsocketAPI.PlaceOrders should require an order")
	_, err = e.WebsocketAPI().PlaceOrders(t.Context(), &PlaceOrdersRequest{Orders: []OrderRequest{{Price: 1}}})
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid, "WebsocketAPI.PlaceOrders should reject an invalid order")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	ex, venue := newVenueWebsocketTestExchange(t)
	placed := websocketTestOrder(t)
	statuses, err := ex.WebsocketAPI().PlaceOrders(t.Context(), &PlaceOrdersRequest{Orders: []OrderRequest{placed}})
	require.NoError(t, err, "WebsocketAPI.PlaceOrders must not error")
	require.Len(t, statuses, 1, "WebsocketAPI.PlaceOrders must return the order's status")
	require.NotNil(t, statuses[0].Resting, "the order must rest")
	assert.Equal(t, placed.ClientOrderID, statuses[0].Resting.ClientOrderID, "WebsocketAPI.PlaceOrders should return the order's client order ID")
	if mockTests {
		assert.Equal(t, []OrderActionStatusResponse{{Resting: &RestingOrderStatus{OrderID: testOrderID, ClientOrderID: testClientOrderID}}}, statuses, "WebsocketAPI.PlaceOrders should return the fixture status")
		assertWebsocketActionPost(t, venue.received(), `{"type":"order","orders":[{"a":0,"b":true,"p":"85000","s":"0.001","r":false,"t":{"limit":{"tif":"Gtc"}},"c":"`+testClientOrderID+`"}],"grouping":"na"}`)
	}
}

func TestWebsocketAPICancelOrders(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAPI().CancelOrders(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "WebsocketAPI.CancelOrders should reject a nil request")
	_, err = e.WebsocketAPI().CancelOrders(t.Context(), &CancelOrdersRequest{})
	assert.ErrorIs(t, err, errNoActionItems, "WebsocketAPI.CancelOrders should require a cancel")
	_, err = e.WebsocketAPI().CancelOrders(t.Context(), &CancelOrdersRequest{Cancels: []CancelRequest{{Asset: 1}}})
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet, "WebsocketAPI.CancelOrders should require an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	ex, venue := newVenueWebsocketTestExchange(t)
	placed, orderID := websocketTestRestingOrder(t)
	statuses, err := ex.WebsocketAPI().CancelOrders(t.Context(), &CancelOrdersRequest{Cancels: []CancelRequest{{Asset: placed.Asset, OrderID: orderID}}})
	require.NoError(t, err, "WebsocketAPI.CancelOrders must not error")
	assert.Equal(t, []CancelActionStatusResponse{{}}, statuses, "WebsocketAPI.CancelOrders should cancel the order")
	if mockTests {
		assertWebsocketActionPost(t, venue.received(), `{"type":"cancel","cancels":[{"a":0,"o":`+strconv.FormatUint(testOrderID, 10)+`}]}`)
	}
}

func TestWebsocketAPICancelOrdersByClientOrderID(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAPI().CancelOrdersByClientOrderID(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "WebsocketAPI.CancelOrdersByClientOrderID should reject a nil request")
	_, err = e.WebsocketAPI().CancelOrdersByClientOrderID(t.Context(), &CancelOrdersByClientOrderIDRequest{})
	assert.ErrorIs(t, err, errNoActionItems, "WebsocketAPI.CancelOrdersByClientOrderID should require a cancel")
	_, err = e.WebsocketAPI().CancelOrdersByClientOrderID(t.Context(), &CancelOrdersByClientOrderIDRequest{Cancels: []CancelByClientOrderIDRequest{{ClientOrderID: "0x1"}}})
	assert.ErrorIs(t, err, errClientOrderIDInvalid, "WebsocketAPI.CancelOrdersByClientOrderID should reject an invalid client order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	ex, venue := newVenueWebsocketTestExchange(t)
	placed, _ := websocketTestRestingOrder(t)
	statuses, err := ex.WebsocketAPI().CancelOrdersByClientOrderID(t.Context(), &CancelOrdersByClientOrderIDRequest{Cancels: []CancelByClientOrderIDRequest{{Asset: placed.Asset, ClientOrderID: placed.ClientOrderID}}})
	require.NoError(t, err, "WebsocketAPI.CancelOrdersByClientOrderID must not error")
	assert.Equal(t, []CancelActionStatusResponse{{}}, statuses, "WebsocketAPI.CancelOrdersByClientOrderID should cancel the order")
	if mockTests {
		assertWebsocketActionPost(t, venue.received(), `{"type":"cancelByCloid","cancels":[{"asset":0,"cloid":"`+testClientOrderID+`"}]}`)
	}
}

func TestWebsocketAPIModifySingleOrder(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, e.WebsocketAPI().ModifySingleOrder(t.Context(), nil), common.ErrNilPointer, "WebsocketAPI.ModifySingleOrder should reject a nil request")
	assert.ErrorIs(t, e.WebsocketAPI().ModifySingleOrder(t.Context(), &ModifyOrderRequest{}), order.ErrOrderIDNotSet, "WebsocketAPI.ModifySingleOrder should require an order identifier")
	assert.ErrorIs(t, e.WebsocketAPI().ModifySingleOrder(t.Context(), &ModifyOrderRequest{OrderID: 1, Order: OrderRequest{Price: 1, Size: 1}}), errOrderTypeRequired, "WebsocketAPI.ModifySingleOrder should reject an invalid replacement")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	ex, venue := newVenueWebsocketTestExchange(t)
	placed, orderID := websocketTestRestingOrder(t)
	replacement := placed
	replacement.Price = 84500
	if !mockTests {
		replacement.Price = math.Round(placed.Price * 0.99)
	}
	require.NoError(t, ex.WebsocketAPI().ModifySingleOrder(t.Context(), &ModifyOrderRequest{OrderID: orderID, Order: replacement}), "WebsocketAPI.ModifySingleOrder must not error")
	if mockTests {
		assertWebsocketActionPost(t, venue.received(), `{"type":"modify","oid":`+strconv.FormatUint(testOrderID, 10)+`,"order":{"a":0,"b":true,"p":"84500","s":"0.001","r":false,"t":{"limit":{"tif":"Gtc"}},"c":"`+testClientOrderID+`"}}`)
	}
}

func TestWebsocketAPIModifyOrders(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAPI().ModifyOrders(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "WebsocketAPI.ModifyOrders should reject a nil request")
	_, err = e.WebsocketAPI().ModifyOrders(t.Context(), &ModifyOrdersRequest{})
	assert.ErrorIs(t, err, errNoActionItems, "WebsocketAPI.ModifyOrders should require a replacement")
	_, err = e.WebsocketAPI().ModifyOrders(t.Context(), &ModifyOrdersRequest{Modifies: []OrderModification{{OrderID: 1, ClientOrderID: testClientOrderID}}})
	assert.ErrorIs(t, err, errOrderIdentifiersConflict, "WebsocketAPI.ModifyOrders should reject both order identifiers")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	ex, venue := newVenueWebsocketTestExchange(t)
	placed, orderID := websocketTestRestingOrder(t)
	replacement := placed
	replacement.Price = 84000
	if !mockTests {
		replacement.Price = math.Round(placed.Price * 0.98)
	}
	statuses, err := ex.WebsocketAPI().ModifyOrders(t.Context(), &ModifyOrdersRequest{Modifies: []OrderModification{{OrderID: orderID, Order: replacement}}})
	require.NoError(t, err, "WebsocketAPI.ModifyOrders must not error")
	require.Len(t, statuses, 1, "WebsocketAPI.ModifyOrders must return the replacement's status")
	require.NotNil(t, statuses[0].Resting, "the replacement must rest")
	if mockTests {
		assert.Equal(t, []OrderActionStatusResponse{{Resting: &RestingOrderStatus{OrderID: 566563303537, ClientOrderID: testClientOrderID}}}, statuses, "WebsocketAPI.ModifyOrders should return the fixture status")
		assertWebsocketActionPost(t, venue.received(), `{"type":"batchModify","modifies":[{"oid":`+strconv.FormatUint(testOrderID, 10)+`,"order":{"a":0,"b":true,"p":"84000","s":"0.001","r":false,"t":{"limit":{"tif":"Gtc"}},"c":"`+testClientOrderID+`"}}]}`)
	}
}

// TestSendWebsocketAction answers through a fixture connection, as the venue answers only validly signed actions; the
// websocket order tests make the venue round trips
func TestSendWebsocketAction(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	connection := publicConn(t, ex)
	action := &SignedActionRequest{Action: EmptyAction{Type: "noop"}, Nonce: 1}
	connection.reply = replyToPost(`{"type":"action","payload":{"status":"ok","response":{"type":"default"}}}`)
	response, err := ex.sendWebsocketAction(t.Context(), action)
	require.NoError(t, err, "sendWebsocketAction must not error")
	assert.JSONEq(t, `{"type":"default"}`, string(response), "sendWebsocketAction should return the accepted action's response")
	assert.Equal(t, []string{
		`{"method":"post","id":1,"request":{"type":"action","payload":{"action":{"type":"noop"},"nonce":1,"signature":{"r":"","s":"","v":0}}}}`,
	}, connection.sentRequests(), "sendWebsocketAction should post the signed action request as an action")

	connection.reply = replyToPost(`{"type":"action","payload":{"status":"err","response":"Order has invalid price."}}`)
	_, err = ex.sendWebsocketAction(t.Context(), action)
	assert.ErrorIs(t, err, errActionResponse, "sendWebsocketAction should return a rejected action's error")
	assert.NotErrorIs(t, err, errSignerRejected, "sendWebsocketAction should not report another rejection as the signer's")

	connection.reply = replyToPost(`{"type":"action","payload":{"status":"err","response":"User or API Wallet 0x0000000000000000000000000000000000000001 does not exist."}}`)
	_, err = ex.sendWebsocketAction(t.Context(), action)
	assert.ErrorIs(t, err, errSignerRejected, "sendWebsocketAction should report a rejected signer")

	connection.reply = replyToPost(`{"type":"action","payload":null}`)
	_, err = ex.sendWebsocketAction(t.Context(), action)
	assert.ErrorIs(t, err, common.ErrNoResponse, "sendWebsocketAction should reject an empty action response")

	connection.reply = replyToPost(`{"type":"action","payload":"invalid"}`)
	_, err = ex.sendWebsocketAction(t.Context(), action)
	assert.ErrorContains(t, err, "decoding action response", "sendWebsocketAction should reject an undecodable action response")

	connection.reply = replyToPost(`{"type":"error","payload":"422 Unprocessable Entity"}`)
	_, err = ex.sendWebsocketAction(t.Context(), action)
	assert.ErrorIs(t, err, errWebsocketPost, "sendWebsocketAction should return a rejected post's error")
}

// TestSendSignedActionOverWebsocket sends actions over a fixture connection while the exchange endpoint counts what it
// receives, so it shows the websocket option bypasses the endpoint and that a signer rejected over the websocket
// invalidates the cached authority
func TestSendSignedActionOverWebsocket(t *testing.T) {
	t.Parallel()
	var exchangeRequests atomic.Int32
	ex := newTradingServerExchange(t, nil, func(map[string]any) string {
		exchangeRequests.Add(1)
		return defaultActionResponse
	})
	connection := &websocketConnectionFixture{exchange: ex, reply: replyToPost(`{"type":"action","payload":{"status":"ok","response":{"type":"default"}}}`)}
	ex.websocketPostConnection = connection
	response, err := ex.SendSignedAction(t.Context(), EmptyAction{Type: "noop"}, &L1ActionOptionsRequest{Websocket: true})
	require.NoError(t, err, "SendSignedAction must not error over the websocket")
	assert.JSONEq(t, `{"type":"default"}`, string(response), "SendSignedAction should return the websocket response")
	assertWebsocketActionPost(t, connection.sentRequests(), `{"type":"noop"}`)
	assert.True(t, authorityCached(t, ex), "SendSignedAction should cache the validated authority")

	connection.reply = replyToPost(`{"type":"action","payload":{"status":"err","response":"User or API Wallet ` + testAccountAddress + ` does not exist."}}`)
	_, err = ex.SendSignedAction(t.Context(), EmptyAction{Type: "noop"}, &L1ActionOptionsRequest{Websocket: true})
	assert.ErrorIs(t, err, errSignerRejected, "SendSignedAction should report a signer rejected over the websocket")
	assert.False(t, authorityCached(t, ex), "SendSignedAction should invalidate the cached authority after a signer rejection over the websocket")

	connection.reply = replyToPost(`{"type":"action","payload":{"status":"err","response":"Order has invalid price."}}`)
	_, err = ex.SendSignedAction(t.Context(), EmptyAction{Type: "noop"}, &L1ActionOptionsRequest{Websocket: true})
	assert.ErrorIs(t, err, errActionResponse, "SendSignedAction should return an action rejected over the websocket")
	assert.True(t, authorityCached(t, ex), "SendSignedAction should revalidate the authority and keep it after another rejection")
	assert.Zero(t, exchangeRequests.Load(), "SendSignedAction should not post to the exchange endpoint")
}

type postResponseConnection struct {
	websocket.Connection
	response []byte
}

func (c *postResponseConnection) SendMessageReturnResponse(context.Context, request.EndpointLimit, any, any) ([]byte, error) {
	return c.response, nil
}

func TestSendMessageReturnResponse(t *testing.T) {
	t.Parallel()
	for _, response := range [][]byte{nil, []byte(`[]`), []byte(`{"response":{}}`)} {
		connection := &postResponseConnection{response: response}
		result, err := connection.SendMessageReturnResponse(t.Context(), request.Unset, uint64(1), nil)
		require.NoError(t, err, "SendMessageReturnResponse must return the stub response")
		assert.Equal(t, response, result, "SendMessageReturnResponse should preserve response bytes")
	}
}

func TestSendWebsocketPost(t *testing.T) {
	t.Parallel()
	venue, _ := newVenueWebsocketTestExchange(t)
	payload, err := venue.sendWebsocketPost(t.Context(), wsPostTypeInfo, &InfoRequest{Type: "l2Book", Coin: "BTC"})
	require.NoError(t, err, "sendWebsocketPost must not error")
	if mockTests {
		assert.JSONEq(t, `{"type":"l2Book","data":{"coin":"BTC","time":1791279151499,"levels":[[{"px":"85895.0","sz":"0.00271","n":2}],[{"px":"85896.0","sz":"8.0337","n":32}]]}}`, string(payload), "sendWebsocketPost should return the response payload")
	} else {
		var info WebsocketPostInfoResponse
		require.NoError(t, json.Unmarshal(payload, &info), "the response payload must decode")
		var book *L2BookResponse
		require.NoError(t, json.Unmarshal(info.Data, &book), "the BTC book must decode")
		require.NotNil(t, book, "sendWebsocketPost must return the BTC book")
		assert.NotEmpty(t, book.Levels, "sendWebsocketPost should return the BTC book's levels")
	}
	_, err = venue.sendWebsocketPost(t.Context(), wsPostTypeInfo, &InfoRequest{Type: "bogusInfoType"})
	assert.ErrorIs(t, err, errWebsocketPost, "sendWebsocketPost should return the error of a request the server cannot parse")
	assert.ErrorContains(t, err, wsErrorParseFailure, "sendWebsocketPost should return the server's parse error")
	_, err = venue.sendWebsocketPost(t.Context(), wsPostTypeInfo, &InfoRequest{Type: "l2Book", Coin: "BTC", NumberOfSignificantFigures: 7})
	assert.ErrorIs(t, err, errWebsocketPost, "sendWebsocketPost should return a rejected request's error")
	assert.ErrorContains(t, err, "500 Internal Server Error", "sendWebsocketPost should return the server's message")

	// The venue cannot be made to answer with a structured error or another response type, to stay silent, to fail a
	// send, or to accept an action that is not validly signed, so a fixture connection supplies those replies
	ex := newWebsocketTestExchange(t)
	connection := publicConn(t, ex)
	connection.reply = replyToPost(`{"type":"info","payload":{"type":"l2Book","data":null}}`)
	_, err = ex.sendWebsocketPost(t.Context(), wsPostTypeInfo, &InfoRequest{Type: "l2Book", Coin: "BTC"})
	require.NoError(t, err, "sendWebsocketPost must not error")
	action := &SignedActionRequest{Action: EmptyAction{Type: "noop"}, Nonce: 1}
	connection.reply = replyToPost(`{"type":"action","payload":{"status":"ok","response":{"type":"default"}}}`)
	payload, err = ex.sendWebsocketPost(t.Context(), wsPostTypeAction, action)
	require.NoError(t, err, "sendWebsocketPost must not error for an action")
	assert.JSONEq(t, `{"status":"ok","response":{"type":"default"}}`, string(payload), "sendWebsocketPost should return an action's response payload")
	assert.Equal(t, []string{
		`{"method":"post","id":1,"request":{"type":"info","payload":{"type":"l2Book","coin":"BTC"}}}`,
		`{"method":"post","id":2,"request":{"type":"action","payload":{"action":{"type":"noop"},"nonce":1,"signature":{"r":"","s":"","v":0}}}}`,
	}, connection.sentRequests(), "sendWebsocketPost should send each post with a new ID and its request type")

	connection.reply = replyToPost(`{"type":"info","payload":{}}`)
	_, err = ex.sendWebsocketPost(t.Context(), wsPostTypeAction, action)
	assert.ErrorIs(t, err, errWebsocketPost, "sendWebsocketPost should reject an info response to an action")
	assert.ErrorContains(t, err, `unexpected response type "info"`, "sendWebsocketPost should name the unexpected response type")

	connection.reply = replyToPost(`{"type":"error","payload":"422 Unprocessable Entity"}`)
	_, err = ex.sendWebsocketPost(t.Context(), wsPostTypeAction, action)
	assert.ErrorIs(t, err, errWebsocketPost, "sendWebsocketPost should return a rejected action's error")
	assert.ErrorContains(t, err, "422 Unprocessable Entity", "sendWebsocketPost should return the server's message for an action")

	connection.reply = replyToPost(`{"type":"action","payload":{}}`)
	_, err = ex.sendWebsocketPost(t.Context(), wsPostTypeInfo, &InfoRequest{Type: "l2Book", Coin: "BTC"})
	assert.ErrorIs(t, err, errWebsocketPost, "sendWebsocketPost should reject an action response to an info request")
	assert.ErrorContains(t, err, `unexpected response type "action"`, "sendWebsocketPost should name the unexpected response type")

	connection.reply = replyToPost(`{"type":"error","payload":{"code":500}}`)
	_, err = ex.sendWebsocketPost(t.Context(), wsPostTypeInfo, &InfoRequest{Type: "l2Book", Coin: "BTC"})
	assert.ErrorIs(t, err, errWebsocketPost, "sendWebsocketPost should return a rejected request's error")
	assert.ErrorContains(t, err, `{"code":500}`, "sendWebsocketPost should return a structured error as it was sent")

	connection.reply = replyToPost(`{"type":"info","payload":{"type":"l2Book","data":null}}`)
	ex.websocketPostID.Store(math.MaxUint64 - 1)
	_, err = ex.sendWebsocketPost(t.Context(), wsPostTypeInfo, &InfoRequest{Type: "l2Book", Coin: "BTC"})
	require.NoError(t, err, "sendWebsocketPost must match the largest request ID")
	sent := connection.sentRequests()
	var post websocketVenueRequest
	require.NoError(t, json.Unmarshal([]byte(sent[len(sent)-1]), &post), "the post must decode")
	assert.Equal(t, uint64(math.MaxUint64), post.ID, "sendWebsocketPost should send the largest request ID")

	connection.reply = nil
	_, err = ex.sendWebsocketPost(t.Context(), wsPostTypeInfo, &InfoRequest{Type: "l2Book", Coin: "BTC"})
	assert.ErrorIs(t, err, websocket.ErrSignatureTimeout, "sendWebsocketPost should time out without a response")

	connection.sendErr = errWebsocketFixture
	_, err = ex.sendWebsocketPost(t.Context(), wsPostTypeInfo, &InfoRequest{Type: "l2Book", Coin: "BTC"})
	assert.ErrorIs(t, err, errWebsocketFixture, "sendWebsocketPost should return a send error")
	assert.ErrorContains(t, err, "posting info request", "sendWebsocketPost should name the request it could not post")

	// The normal dispatcher decodes this envelope first; exercise the defensive decode at the connection boundary.
	ex.websocketPostConnection = &postResponseConnection{response: []byte(`[]`)}
	_, err = ex.sendWebsocketPost(t.Context(), wsPostTypeInfo, &InfoRequest{Type: "l2Book", Coin: "BTC"})
	assert.ErrorContains(t, err, "decoding post response", "sendWebsocketPost should report invalid connection response JSON")
	ex.websocketPostConnection = &postResponseConnection{response: []byte(`null`)}
	_, err = ex.sendWebsocketPost(t.Context(), wsPostTypeInfo, &InfoRequest{Type: "l2Book", Coin: "BTC"})
	assert.ErrorIs(t, err, common.ErrNoResponse, "sendWebsocketPost should reject an empty connection response")

	ex.websocketPostConnection = nil
	_, err = ex.sendWebsocketPost(t.Context(), wsPostTypeInfo, &InfoRequest{Type: "l2Book", Coin: "BTC"})
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "sendWebsocketPost should require a connection")
}

func TestWebsocketHandlePostResponse(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.Error(t, ex.wsHandlePostResponse(publicConn(t, ex), []byte(`"invalid"`)), "wsHandlePostResponse should reject invalid data")
	assert.ErrorIs(t, ex.wsHandlePostResponse(publicConn(t, ex), []byte(`null`)), common.ErrNoResponse, "wsHandlePostResponse should reject an empty response")
	assert.ErrorIs(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"post","data":{"id":7,"response":{"type":"info","payload":{}}}}`)), websocket.ErrSignatureNotMatched, "wsHandlePostResponse should reject a response without a request")
	responses, err := publicConn(t, ex).matcher().Set(uint64(7), 1)
	require.NoError(t, err, "Set must not error")
	data := `{"id":7,"response":{"type":"info","payload":{}}}`
	require.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"post","data":`+data+`}`)), "wsHandleData must not error")
	assert.JSONEq(t, data, string(receivePostResponse(t, responses)), "wsHandlePostResponse should deliver the response to its request")
}

// testOrderbookSubscription returns an unstored BTC orderbook subscription and its pending operation key
func testOrderbookSubscription() (*subscription.Subscription, websocketPendingKey) {
	return &subscription.Subscription{
		Channel:          subscription.OrderbookChannel,
		Asset:            asset.PerpetualContract,
		Pairs:            currency.Pairs{perpetualPair},
		QualifiedChannel: "l2Book:perpetualcontract:BTC-USDC",
	}, websocketPendingKey{subscription: WebsocketSubscription{Type: wsChannelOrderbook, Coin: "BTC"}}
}

// testTradesSubscription returns an unstored BTC trades subscription and its pending operation key
func testTradesSubscription() (*subscription.Subscription, websocketPendingKey) {
	return &subscription.Subscription{
		Channel:          subscription.AllTradesChannel,
		Asset:            asset.PerpetualContract,
		Pairs:            currency.Pairs{perpetualPair},
		QualifiedChannel: "trades:perpetualcontract:BTC-USDC",
	}, websocketPendingKey{subscription: WebsocketSubscription{Type: wsChannelTrades, Coin: "BTC"}}
}

func TestWebsocketHandleSubscriptionResponse(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	for _, data := range []string{
		`"invalid"`,
		`{"method":"post","subscription":{"type":"trades","coin":"BTC"}}`,
		`{"method":"subscribe","subscription":{"type":" "}}`,
		`null`,
	} {
		err := ex.wsHandleSubscriptionResponse(publicConn(t, ex), []byte(data))
		assert.ErrorIsf(t, err, errWebsocketSubscription, "wsHandleSubscriptionResponse should reject %s", data)
	}
	assert.ErrorIs(t, ex.wsHandleSubscriptionResponse(publicConn(t, ex), []byte(`null`)), common.ErrNoResponse, "wsHandleSubscriptionResponse should reject an empty acknowledgement")
	assert.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"subscriptionResponse","data":{"method":"subscribe","subscription":{"type":"trades","coin":"ETH"}}}`)), "wsHandleSubscriptionResponse should ignore an acknowledgement without an operation")

	sub, key := testOrderbookSubscription()
	pending := addWebsocketPending(t, ex, wsMethodSubscribe, sub, &key)
	ack := []byte(`{"channel":"subscriptionResponse","data":{"method":"subscribe","subscription":{"type":"l2Book","coin":"BTC","nSigFigs":null,"mantissa":null,"fast":false}}}`)
	require.NoError(t, ex.wsHandleData(t.Context(), accountConn(t, ex), ack), "wsHandleData must not error for an acknowledgement on the other connection")
	assert.Same(t, pending, pendingWebsocketOperation(ex, &key), "wsHandleSubscriptionResponse should not settle another connection's operation")
	require.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), ack), "wsHandleData must not error")
	require.NoError(t, receivePendingResult(t, pending), "the acknowledged subscribe must succeed")
	assert.Equal(t, subscription.SubscribedState, sub.State(), "wsHandleSubscriptionResponse should mark the subscription subscribed")
	assert.Nil(t, pendingWebsocketOperation(ex, &key), "wsHandleSubscriptionResponse should remove the settled operation")
	assert.Equal(t, map[string]struct{}{"ETH": {}}, ex.websocketTradeReplays, "wsHandleSubscriptionResponse should await the replay of an acknowledged trades subscription, with or without an operation, and none for a feed that sends none")

	require.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"subscriptionResponse","data":{"method":"subscribe","subscription":{"type":"trades","coin":"BTC"}}}`)), "wsHandleData must not error for a trades acknowledgement")
	assert.Equal(t, map[string]struct{}{"ETH": {}, "BTC": {}}, ex.websocketTradeReplays, "wsHandleSubscriptionResponse should await the replay each trades subscription's acknowledgement precedes")
	require.NoError(t, ex.wsHandleData(t.Context(), publicConn(t, ex), []byte(`{"channel":"subscriptionResponse","data":{"method":"unsubscribe","subscription":{"type":"trades","coin":"ETH"}}}`)), "wsHandleData must not error for a trades unsubscribe acknowledgement")
	assert.Equal(t, map[string]struct{}{"BTC": {}}, ex.websocketTradeReplays, "wsHandleSubscriptionResponse should stop awaiting the replay of an unsubscribed trades feed")

	for _, tc := range []struct {
		feed                    string
		aggregated, invalidates bool
	}{
		{feed: wsSubscriptionUserEvents, invalidates: true},
		{feed: wsChannelUserTWAPSliceFills, invalidates: true},
		{feed: wsChannelUserFills, invalidates: true},
		{feed: wsChannelUserFills, aggregated: true},
		{feed: wsChannelOrderUpdates},
		{feed: wsChannelTrades},
	} {
		for _, pendingOperation := range []bool{false, true} {
			for _, method := range []string{wsMethodSubscribe, wsMethodUnsubscribe} {
				t.Run(fmt.Sprintf("%s/aggregated=%t/pending=%t/%s", tc.feed, tc.aggregated, pendingOperation, method), func(t *testing.T) {
					t.Parallel()
					ex := newWebsocketOrderEventsExchange(t)
					fillJSON := websocketOrderFillJSON(1013, 131, "84000", "0.005", websocketOrderPlaced+100, true)
					handleWebsocketFrames(t, ex,
						websocketOrderUpdatesFrame(websocketOrderRowJSON(1013, "open", "0.02", "0.02", websocketOrderPlaced)),
						websocketUserFillsFrame(fillJSON),
					)
					receiveOrderUpdates(t, ex)
					receiveOrderUpdates(t, ex)
					ex.websocketPendingMu.Lock()
					record := ex.websocketWatermarks.orders.byOrderID[1013]
					exp := *record
					ex.websocketPendingMu.Unlock()
					require.True(t, exp.covered, "applyWebsocketOrderRow must establish coverage")
					payload := WebsocketSubscription{Type: tc.feed, User: testAccountAddress, AggregateByTime: tc.aggregated}
					var pending *websocketPendingOperation
					if pendingOperation {
						sub := &subscription.Subscription{Channel: tc.feed, Authenticated: true, Params: map[string]any{"aggregateByTime": tc.aggregated}}
						if method == wsMethodUnsubscribe {
							require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(accountConn(t, ex), sub), "AddSuccessfulSubscriptions must store the feed")
						}
						pending = addWebsocketPending(t, ex, method, sub, &websocketPendingKey{connection: accountConn(t, ex), subscription: payload})
					}
					raw, err := json.Marshal(WebsocketSubscriptionResponse{Method: method, Subscription: payload})
					require.NoError(t, err, "Marshal must encode the acknowledgement")
					require.NoError(t, ex.wsHandleSubscriptionResponse(accountConn(t, ex), raw), "wsHandleSubscriptionResponse must accept the acknowledgement")
					if pending != nil {
						require.NoError(t, receivePendingResult(t, pending), "wsHandleSubscriptionResponse must complete the pending operation")
					}
					exp.covered = method != wsMethodSubscribe || !tc.invalidates
					ex.websocketPendingMu.Lock()
					assert.Equal(t, exp, *record, "wsHandleSubscriptionResponse should change only coverage for a started fills feed")
					ex.websocketPendingMu.Unlock()
					handleWebsocketFrames(t, ex, websocketUserFillsFrame(fillJSON))
					assertNoWebsocketData(t, ex)
					handleWebsocketFrames(t, ex, websocketOrderUpdatesFrame(websocketOrderRowJSON(1013, "open", "0.015", "0.02", websocketOrderPlaced+200)))
					receiveOrderUpdates(t, ex)
					handleWebsocketFrames(t, ex, websocketUserFillsFrame(websocketOrderFillJSON(1013, 132, "84000", "0.002", websocketOrderPlaced+300, true)))
					assert.InDelta(t, 0.007, receiveOrderUpdates(t, ex)[0].ExecutedAmount, 1e-12, "wsHandleSubscriptionResponse should allow cumulative execution after a matching row restores coverage")
				})
			}
		}
	}
}

func TestCompleteWebsocketPending(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	sub, key := testOrderbookSubscription()
	settled, err := ex.completeWebsocketPending(publicConn(t, ex), wsMethodSubscribe, &key.subscription)
	assert.False(t, settled, "completeWebsocketPending should report an unknown operation as not settled")
	assert.NoError(t, err, "completeWebsocketPending should ignore an unknown operation")

	pending := addWebsocketPending(t, ex, wsMethodSubscribe, sub, &key)
	settled, err = ex.completeWebsocketPending(publicConn(t, ex), wsMethodUnsubscribe, &key.subscription)
	assert.True(t, settled, "completeWebsocketPending should report an operation completed with another method as settled")
	assert.ErrorIs(t, err, errWebsocketSubscription, "completeWebsocketPending should reject a completion with another method")
	assert.Same(t, pending, pendingWebsocketOperation(ex, &key), "completeWebsocketPending should keep an operation completed with another method")

	settled, err = ex.completeWebsocketPending(publicConn(t, ex), wsMethodSubscribe, &key.subscription)
	assert.True(t, settled, "completeWebsocketPending should report the operation as settled")
	require.NoError(t, err, "completeWebsocketPending must not error")
	require.NoError(t, receivePendingResult(t, pending), "the completed subscribe must succeed")
	assert.Equal(t, subscription.SubscribedState, sub.State(), "completeWebsocketPending should mark the subscription subscribed")

	pending = addWebsocketPending(t, ex, wsMethodUnsubscribe, sub, &key)
	_, err = ex.completeWebsocketPending(publicConn(t, ex), wsMethodUnsubscribe, &key.subscription)
	require.NoError(t, err, "completeWebsocketPending must not error for an unsubscribe")
	require.NoError(t, receivePendingResult(t, pending), "the completed unsubscribe must succeed")
	assert.Nil(t, ex.Websocket.GetSubscription(sub), "completeWebsocketPending should remove the unsubscribed subscription")

	conflict := newSubscribedWebsocketSubscription(t, ex, &subscription.Subscription{Channel: wsChannelAllMids, QualifiedChannel: wsChannelAllMids})
	conflictKey := websocketPendingKey{subscription: WebsocketSubscription{Type: wsChannelAllMids}}
	pending = &websocketPendingOperation{method: wsMethodSubscribe, connection: publicConn(t, ex), subscription: conflict, done: make(chan error, 1)}
	registerWebsocketPending(ex, &conflictKey, pending)
	_, err = ex.completeWebsocketPending(publicConn(t, ex), wsMethodSubscribe, &conflictKey.subscription)
	assert.ErrorIs(t, err, subscription.ErrInStateAlready, "completeWebsocketPending should return a state conflict")
	assert.ErrorIs(t, receivePendingResult(t, pending), subscription.ErrInStateAlready, "completeWebsocketPending should fail the operation with the state conflict")
	assert.Nil(t, ex.Websocket.GetSubscription(conflict), "completeWebsocketPending should roll back a subscribe it cannot complete")

	unstored := &subscription.Subscription{Channel: wsChannelOutcomeMetaUpdates, QualifiedChannel: wsChannelOutcomeMetaUpdates}
	require.NoError(t, unstored.SetState(subscription.SubscribedState), "SetState must not error")
	unstoredKey := websocketPendingKey{subscription: WebsocketSubscription{Type: wsChannelOutcomeMetaUpdates}}
	pending = addWebsocketPending(t, ex, wsMethodUnsubscribe, unstored, &unstoredKey)
	_, err = ex.completeWebsocketPending(publicConn(t, ex), wsMethodUnsubscribe, &unstoredKey.subscription)
	assert.ErrorIs(t, err, subscription.ErrNotFound, "completeWebsocketPending should return a store error")
	assert.ErrorIs(t, receivePendingResult(t, pending), subscription.ErrNotFound, "completeWebsocketPending should fail the operation with the store error")
	assert.Equal(t, subscription.SubscribedState, unstored.State(), "completeWebsocketPending should restore the state of an unsubscribe it cannot complete")
}

func TestFailWebsocketPendingOperation(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	sub, key := testOrderbookSubscription()
	assert.False(t, ex.failWebsocketPendingOperation(publicConn(t, ex), wsMethodSubscribe, &key.subscription, errWebsocketFixture), "failWebsocketPendingOperation should report an unknown operation as not failed")
	pending := addWebsocketPending(t, ex, wsMethodSubscribe, sub, &key)
	assert.False(t, ex.failWebsocketPendingOperation(publicConn(t, ex), wsMethodUnsubscribe, &key.subscription, errWebsocketFixture), "failWebsocketPendingOperation should not fail an operation of another method")
	assert.False(t, ex.failWebsocketPendingOperation(accountConn(t, ex), wsMethodSubscribe, &key.subscription, errWebsocketFixture), "failWebsocketPendingOperation should not fail another connection's operation")
	assert.Same(t, pending, pendingWebsocketOperation(ex, &key), "failWebsocketPendingOperation should keep an operation it does not fail")
	assert.True(t, ex.failWebsocketPendingOperation(publicConn(t, ex), wsMethodSubscribe, &key.subscription, errWebsocketFixture), "failWebsocketPendingOperation should fail the operation")
	assert.ErrorIs(t, receivePendingResult(t, pending), errWebsocketFixture, "failWebsocketPendingOperation should fail the operation with the cause")
	assert.Nil(t, ex.Websocket.GetSubscription(sub), "failWebsocketPendingOperation should roll back the subscribe")
	assert.Nil(t, pendingWebsocketOperation(ex, &key), "failWebsocketPendingOperation should remove the operation")
}

func TestRollbackWebsocketPending(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.ErrorIs(t, ex.rollbackWebsocketPending(nil), common.ErrNilPointer, "rollbackWebsocketPending should reject a nil operation")
	assert.ErrorIs(t, ex.rollbackWebsocketPending(&websocketPendingOperation{}), common.ErrNilPointer, "rollbackWebsocketPending should reject an operation without a subscription")

	sub, _ := testOrderbookSubscription()
	require.NoError(t, ex.Websocket.AddSubscriptions(publicConn(t, ex), sub), "AddSubscriptions must not error")
	require.NoError(t, ex.rollbackWebsocketPending(&websocketPendingOperation{method: wsMethodSubscribe, connection: publicConn(t, ex), subscription: sub}), "rollbackWebsocketPending must not error for a subscribe")
	assert.Nil(t, ex.Websocket.GetSubscription(sub), "rollbackWebsocketPending should remove a subscribe's subscription")

	unsubscribe := &subscription.Subscription{Channel: wsChannelAllMids}
	require.NoError(t, unsubscribe.SetState(subscription.SubscribedState), "SetState must not error")
	operation := &websocketPendingOperation{method: wsMethodUnsubscribe, connection: publicConn(t, ex), subscription: unsubscribe, previousState: subscription.SubscribedState}
	require.NoError(t, ex.rollbackWebsocketPending(operation), "rollbackWebsocketPending must not error for a subscription in its previous state")
	require.NoError(t, unsubscribe.SetState(subscription.UnsubscribingState), "SetState must not error")
	require.NoError(t, ex.rollbackWebsocketPending(operation), "rollbackWebsocketPending must not error for an unsubscribe")
	assert.Equal(t, subscription.SubscribedState, unsubscribe.State(), "rollbackWebsocketPending should restore an unsubscribe's previous state")

	operation.method = "post"
	assert.ErrorIs(t, ex.rollbackWebsocketPending(operation), common.ErrNotYetImplemented, "rollbackWebsocketPending should reject another method")
}

func TestFailWebsocketPending(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.NoError(t, ex.failWebsocketPending(publicConn(t, ex), nil), "failWebsocketPending should not error without operations or a cause")

	publicSub, publicKey := testOrderbookSubscription()
	publicPending := addWebsocketPending(t, ex, wsMethodSubscribe, publicSub, &publicKey)
	invalidKey := websocketPendingKey{subscription: WebsocketSubscription{Type: wsChannelAllMids}}
	invalidPending := &websocketPendingOperation{method: "post", connection: publicConn(t, ex), subscription: &subscription.Subscription{}, done: make(chan error, 1)}
	registerWebsocketPending(ex, &invalidKey, invalidPending)
	authenticatedSub := &subscription.Subscription{Channel: subscription.MyOrdersChannel, Authenticated: true, QualifiedChannel: wsChannelOrderUpdates}
	authenticatedKey := websocketPendingKey{connection: accountConn(t, ex), subscription: WebsocketSubscription{Type: wsChannelOrderUpdates, User: testAccountAddress}}
	authenticatedPending := addWebsocketPending(t, ex, wsMethodSubscribe, authenticatedSub, &authenticatedKey)

	err := ex.failWebsocketPending(publicConn(t, ex), errWebsocketFixture)
	assert.ErrorIs(t, err, errWebsocketFixture, "failWebsocketPending should return the cause")
	assert.ErrorIs(t, err, common.ErrNotYetImplemented, "failWebsocketPending should return rollback errors")
	assert.ErrorIs(t, receivePendingResult(t, publicPending), errWebsocketFixture, "failWebsocketPending should fail each operation with the cause")
	assert.ErrorIs(t, receivePendingResult(t, invalidPending), common.ErrNotYetImplemented, "failWebsocketPending should fail an operation with its rollback error")
	assert.Nil(t, ex.Websocket.GetSubscription(publicSub), "failWebsocketPending should roll back the failed subscribe")
	assert.Same(t, authenticatedPending, pendingWebsocketOperation(ex, &authenticatedKey), "failWebsocketPending should keep the other connection's operations")

	assert.ErrorIs(t, ex.failWebsocketPending(accountConn(t, ex), errWebsocketServer), errWebsocketServer, "failWebsocketPending should return the cause")
	assert.ErrorIs(t, receivePendingResult(t, authenticatedPending), errWebsocketServer, "failWebsocketPending should fail the account connection's operations")
	assert.Nil(t, ex.Websocket.GetSubscription(authenticatedSub), "failWebsocketPending should roll back the account subscribe")
	assert.Empty(t, ex.websocketPending, "failWebsocketPending should remove every failed operation")
}

func TestAbortWebsocketPending(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	sub, key := testOrderbookSubscription()
	pending := addWebsocketPending(t, ex, wsMethodSubscribe, sub, &key)
	assert.ErrorIs(t, ex.abortWebsocketPending(&key, pending, errWebsocketFixture), errWebsocketFixture, "abortWebsocketPending should return the cause")
	assert.Nil(t, pendingWebsocketOperation(ex, &key), "abortWebsocketPending should remove the operation")
	assert.Nil(t, ex.Websocket.GetSubscription(sub), "abortWebsocketPending should roll back the operation")

	settled := &websocketPendingOperation{done: make(chan error, 1)}
	settled.done <- errWebsocketServer
	err := ex.abortWebsocketPending(&key, settled, errWebsocketFixture)
	assert.ErrorIs(t, err, errWebsocketServer, "abortWebsocketPending should return the outcome of an operation the server settled")
	assert.NotErrorIs(t, err, errWebsocketFixture, "abortWebsocketPending should discard the cause of an operation the server settled")
}

// websocketErrorFrame returns an error frame holding the message
func websocketErrorFrame(t *testing.T, message string) []byte {
	t.Helper()
	encoded, err := json.Marshal(message)
	require.NoError(t, err, "Marshal must not error")
	return []byte(`{"channel":"error","data":` + string(encoded) + `}`)
}

func TestWebsocketHandleError(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	tradesSub, tradesKey := testTradesSubscription()
	pending := addWebsocketPending(t, ex, wsMethodSubscribe, tradesSub, &tradesKey)
	require.NoError(t, ex.wsHandleError(publicConn(t, ex), []byte(strconv.Quote(wsErrorAlreadySubscribed+`{"type":"trades","coin":"BTC"}`))), "wsHandleError must not error for a duplicate subscribe")
	require.NoError(t, receivePendingResult(t, pending), "wsHandleError must settle a duplicate subscribe")
	assert.Equal(t, subscription.SubscribedState, tradesSub.State(), "wsHandleError should mark a duplicate subscribe subscribed")
	assert.False(t, ex.consumeTradeReplay("BTC"), "wsHandleError should not await a replay for a duplicate trades subscribe, as none follows")

	pending = addWebsocketPending(t, ex, wsMethodUnsubscribe, tradesSub, &tradesKey)
	require.NoError(t, ex.wsHandleError(publicConn(t, ex), []byte(strconv.Quote(wsErrorAlreadyUnsubscribed+`{"type":"trades","coin":"BTC"}`))), "wsHandleError must not error for a duplicate unsubscribe")
	require.NoError(t, receivePendingResult(t, pending), "wsHandleError must settle a duplicate unsubscribe")
	assert.Nil(t, ex.Websocket.GetSubscription(tradesSub), "wsHandleError should remove a duplicate unsubscribe's subscription")

	for _, message := range []string{
		wsErrorAlreadySubscribed + `{"type":"allMids"}`,
		wsErrorAlreadySubscribed + `invalid`,
		wsErrorAlreadyUnsubscribed + `{"type":"trades","coin":"ETH"}`,
		wsErrorAlreadyUnsubscribed + `invalid`,
	} {
		err := ex.wsHandleError(publicConn(t, ex), []byte(strconv.Quote(message)))
		assert.ErrorIsf(t, err, errWebsocketServer, "wsHandleError should return %q without an operation", message)
		assert.ErrorContainsf(t, err, message, "wsHandleError should return the server's message %q", message)
	}

	bookSub, bookKey := testOrderbookSubscription()
	bookPending := addWebsocketPending(t, ex, wsMethodSubscribe, bookSub, &bookKey)
	assetDataSub := &subscription.Subscription{Channel: wsChannelActiveAssetData, Authenticated: true, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}, QualifiedChannel: "activeAssetData:perpetualcontract:BTC-USDC"}
	assetDataKey := websocketPendingKey{connection: accountConn(t, ex), subscription: WebsocketSubscription{Type: wsChannelActiveAssetData, Coin: "BTC", User: testAccountAddress}}
	assetDataPending := addWebsocketPending(t, ex, wsMethodSubscribe, assetDataSub, &assetDataKey)
	err := ex.wsHandleError(accountConn(t, ex), []byte(strconv.Quote(wsErrorInvalidSubscription+`{"type":"activeAssetData","user":"`+testAccountAddress+`","coin":"BTC"}`)))
	assert.ErrorIs(t, err, errWebsocketServer, "wsHandleError should return an invalid subscription's error")
	assert.ErrorIs(t, receivePendingResult(t, assetDataPending), errWebsocketServer, "wsHandleError should fail the invalid subscription")
	assert.Nil(t, ex.Websocket.GetSubscription(assetDataSub), "wsHandleError should roll back the invalid subscription")
	assert.Same(t, bookPending, pendingWebsocketOperation(ex, &bookKey), "wsHandleError should keep the operations an error does not answer")

	candleSub := &subscription.Subscription{Channel: subscription.CandlesChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}, Interval: kline.OneMin, QualifiedChannel: "candle:perpetualcontract:BTC-USDC:1m"}
	candleKey := websocketPendingKey{subscription: WebsocketSubscription{Type: wsChannelCandle, Coin: "BTC", Interval: "1m"}}
	candlePending := addWebsocketPending(t, ex, wsMethodSubscribe, candleSub, &candleKey)
	err = ex.wsHandleError(publicConn(t, ex), []byte(strconv.Quote(wsErrorParseFailure+`{"method":"subscribe","subscription":{"type":"candle","coin":"BTC","interval":"1m"}}`)))
	assert.ErrorIs(t, err, errWebsocketServer, "wsHandleError should return a parse failure's error")
	assert.ErrorIs(t, receivePendingResult(t, candlePending), errWebsocketServer, "wsHandleError should fail the subscription the server could not parse")
	assert.Same(t, bookPending, pendingWebsocketOperation(ex, &bookKey), "wsHandleError should keep the operations a parse failure does not answer")

	responses, err := publicConn(t, ex).matcher().Set(uint64(4), 1)
	require.NoError(t, err, "Set must not error")
	postFailure := wsErrorParseFailure + `{"method":"post","id":4,"request":{"type":"info","payload":{"type":"bogusInfoType"}}}`
	require.NoError(t, ex.wsHandleError(publicConn(t, ex), []byte(strconv.Quote(postFailure))), "wsHandleError must not error for a post the server could not parse")
	var response WebsocketPostResponse
	require.NoError(t, json.Unmarshal(receivePostResponse(t, responses), &response), "Unmarshal must not error")
	encodedFailure, err := json.Marshal(postFailure)
	require.NoError(t, err, "Marshal must not error")
	assert.Equal(t, WebsocketPostResponse{ID: 4, Response: WebsocketPostResponseBody{Type: wsPostTypeError, Payload: encodedFailure}}, response, "wsHandleError should answer the post with the server's error")
	err = ex.wsHandleError(publicConn(t, ex), []byte(strconv.Quote(postFailure)))
	assert.ErrorIs(t, err, errWebsocketServer, "wsHandleError should return a post parse failure without a request")
	assert.Same(t, bookPending, pendingWebsocketOperation(ex, &bookKey), "wsHandleError should keep the operations a post parse failure does not answer")
	require.True(t, ex.failWebsocketPendingOperation(publicConn(t, ex), wsMethodSubscribe, &bookKey.subscription, errWebsocketFixture), "failWebsocketPendingOperation must fail the remaining operation")
	require.ErrorIs(t, receivePendingResult(t, bookPending), errWebsocketFixture, "failWebsocketPendingOperation must fail the remaining operation")

	for _, message := range []string{
		wsErrorInvalidSubscription + `{"type":"activeAssetData","user":"` + testAccountAddress + `","coin":"@107"}`,
		wsErrorParseFailure + `{"method":"subscribe","subscription":{"type":"bogus"}}`,
		wsErrorParseFailure + `not json`,
		"Websocket server error",
	} {
		sub, key := testOrderbookSubscription()
		pending := addWebsocketPending(t, ex, wsMethodSubscribe, sub, &key)
		err := ex.wsHandleError(publicConn(t, ex), []byte(strconv.Quote(message)))
		assert.ErrorIsf(t, err, errWebsocketServer, "wsHandleError should return %q", message)
		assert.ErrorIsf(t, receivePendingResult(t, pending), errWebsocketServer, "wsHandleError should fail every operation on the connection for %q", message)
		assert.Nilf(t, ex.Websocket.GetSubscription(sub), "wsHandleError should roll back every operation on the connection for %q", message)
	}

	err = ex.wsHandleError(publicConn(t, ex), []byte(`null`))
	assert.ErrorIs(t, err, errWebsocketServer, "wsHandleError should return an empty error")
	assert.ErrorContains(t, err, "unspecified error", "wsHandleError should describe an empty error")
	err = ex.wsHandleError(publicConn(t, ex), []byte(`{"message":"bad request"}`))
	assert.ErrorContains(t, err, `{"message":"bad request"}`, "wsHandleError should return a structured error as it was sent")
}

func TestDecodeWebsocketSubscription(t *testing.T) {
	t.Parallel()
	sub, err := decodeWebsocketSubscription(`{"type":"l2Book","coin":"BTC","nSigFigs":5,"mantissa":2,"fast":false}`)
	require.NoError(t, err, "decodeWebsocketSubscription must not error")
	assert.Equal(t, WebsocketSubscription{Type: wsChannelOrderbook, Coin: "BTC", NumberOfSignificantFigures: 5, Mantissa: 2}, sub, "decodeWebsocketSubscription should decode the canonical subscription")
	_, err = decodeWebsocketSubscription("invalid")
	assert.Error(t, err, "decodeWebsocketSubscription should reject an invalid subscription")
}

func TestWebsocketChannelName(t *testing.T) {
	t.Parallel()
	_, err := websocketChannelName(nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "websocketChannelName should reject a nil subscription")
	for channel, exp := range map[string]string{
		subscription.TickerChannel:    wsChannelActiveAssetContext,
		subscription.OrderbookChannel: wsChannelOrderbook,
		subscription.AllTradesChannel: wsChannelTrades,
		subscription.CandlesChannel:   wsChannelCandle,
		subscription.MyOrdersChannel:  wsChannelOrderUpdates,
		subscription.MyTradesChannel:  wsSubscriptionUserEvents,
		wsChannelBestBidOffer:         wsChannelBestBidOffer,
		wsChannelFastAssetContexts:    wsChannelFastAssetContexts,
	} {
		name, err := websocketChannelName(&subscription.Subscription{Channel: channel})
		require.NoErrorf(t, err, "websocketChannelName must not error for %s", channel)
		assert.Equalf(t, exp, name, "websocketChannelName should name the subscription type of %s", channel)
	}
	_, err = websocketChannelName(&subscription.Subscription{Channel: wsChannelUserEvents})
	assert.ErrorIs(t, err, subscription.ErrNotSupported, "websocketChannelName should reject a data channel that is not a subscription type")
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	public, err := ex.expandWebsocketSubscriptions(false)
	require.NoError(t, err, "expandWebsocketSubscriptions must not error for the public feeds")
	account, err := ex.expandWebsocketSubscriptions(true)
	require.NoError(t, err, "expandWebsocketSubscriptions must not error for the account feeds")
	subs := slices.Concat(public, account)
	for _, sub := range subs {
		assert.Equalf(t, newWebsocketSubscriptionKey(sub), sub.EnsureKeyed(), "expandWebsocketSubscriptions should key %s by its feed", sub)
	}
	stored := &subscription.Subscription{Channel: wsChannelAllMids, QualifiedChannel: wsChannelAllMids}
	held := newWebsocketSubscriptionKey(stored)
	held.feed = WebsocketSubscription{Type: wsChannelAllMids}
	stored.SetKey(held)
	ex.Features.Subscriptions = subscription.List{stored}
	_, err = ex.expandWebsocketSubscriptions(false)
	require.NoError(t, err, "expandWebsocketSubscriptions must not error for an expanded subscription")
	assert.Equal(t, held, stored.EnsureKeyed(), "expandWebsocketSubscriptions should keep the key of a subscription keyed by its feed, which may be held")

	// The configuration replaces the held subscriptions, so a variant of a held book is no conflict
	require.NoError(t, subscribeWebsocketTest(t, ex, subscription.List{testBookSubscription(20, nil)}), "Subscribe must not error")
	fast := testBookSubscription(5, nil)
	fast.Enabled = true
	ex.Features.Subscriptions = subscription.List{fast}
	replacement, err := ex.expandWebsocketSubscriptions(false)
	require.NoError(t, err, "expandWebsocketSubscriptions must not validate the configuration beside the held subscriptions it replaces")
	require.Len(t, replacement, 1, "expandWebsocketSubscriptions must expand the configured book")
	assert.Equal(t, WebsocketSubscription{Type: wsChannelOrderbook, Fast: true}, newWebsocketSubscriptionKey(replacement[0]).options, "expandWebsocketSubscriptions should generate the configured variant")
	slow := testBookSubscription(20, nil)
	slow.Enabled = true
	ex.Features.Subscriptions = subscription.List{fast, slow}
	conflicting, err := ex.expandWebsocketSubscriptions(false)
	assert.ErrorIs(t, err, subscription.ErrExclusiveSubscription, "expandWebsocketSubscriptions should refuse a configuration holding two variants of a market's l2Book")
	assert.Nil(t, conflicting, "expandWebsocketSubscriptions should generate nothing of a refused configuration")
	ex.Features.Subscriptions = subscription.List{{Enabled: true, Channel: "bogus"}, fast}
	partial, err := ex.expandWebsocketSubscriptions(false)
	assert.ErrorIs(t, err, subscription.ErrNotSupported, "expandWebsocketSubscriptions should report a configured subscription it cannot expand")
	assert.Len(t, partial, 1, "expandWebsocketSubscriptions should generate the configured subscriptions that expand")
	exp := make(subscription.List, 0, 14)
	for _, standard := range []struct {
		channel, name string
		interval      kline.Interval
	}{
		{channel: subscription.TickerChannel, name: wsChannelActiveAssetContext},
		{channel: subscription.OrderbookChannel, name: wsChannelOrderbook},
		{channel: subscription.AllTradesChannel, name: wsChannelTrades},
		{channel: subscription.CandlesChannel, name: wsChannelCandle, interval: kline.OneMin},
	} {
		for _, market := range []struct {
			a    asset.Item
			pair currency.Pair
		}{
			{a: asset.PerpetualContract, pair: perpetualPair},
			{a: asset.PerpetualContract, pair: testBuilderPair},
			{a: asset.Spot, pair: spotPair},
		} {
			pair := market.pair.Format(dashFormat)
			qualifiedChannel := standard.name + ":" + market.a.String() + ":" + pair.String()
			if standard.interval != 0 {
				qualifiedChannel += ":1m"
			}
			exp = append(exp, &subscription.Subscription{
				Enabled:          true,
				Channel:          standard.channel,
				Asset:            market.a,
				Pairs:            currency.Pairs{pair},
				Interval:         standard.interval,
				QualifiedChannel: qualifiedChannel,
			})
		}
	}
	exp = append(
		exp,
		&subscription.Subscription{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true, QualifiedChannel: wsChannelOrderUpdates},
		&subscription.Subscription{Enabled: true, Channel: subscription.MyTradesChannel, Authenticated: true, QualifiedChannel: wsSubscriptionUserEvents},
	)
	testsubs.EqualLists(t, exp, subs)
}

func TestGenerateConnectionSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	public, err := ex.generatePublicSubscriptions()
	require.NoError(t, err, "generatePublicSubscriptions must not error")
	require.Len(t, public, 12, "generatePublicSubscriptions must generate each market feed of each enabled market")
	for _, sub := range public {
		assert.Falsef(t, sub.Authenticated, "generatePublicSubscriptions should leave out the account feed %s", sub)
	}
	ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
	account, err := ex.generateAccountSubscriptions()
	require.NoError(t, err, "generateAccountSubscriptions must not error")
	require.Len(t, account, 2, "generateAccountSubscriptions must generate the account feeds while the account connection is down")
	for _, sub := range account {
		assert.Truef(t, sub.Authenticated, "generateAccountSubscriptions should leave out the market feed %s", sub)
	}

	ex.API.AuthenticatedWebsocketSupport = false
	account, err = ex.generateAccountSubscriptions()
	require.NoError(t, err, "generateAccountSubscriptions must not error without authenticated websocket support")
	assert.Empty(t, account, "generateAccountSubscriptions should generate nothing without authenticated websocket support")
	ex.API.AuthenticatedWebsocketSupport = true
	setTestCredentials(ex, &accounts.Credentials{Key: "invalid"})
	account, err = ex.generateAccountSubscriptions()
	require.NoError(t, err, "generateAccountSubscriptions must not error without an account address")
	assert.Empty(t, account, "generateAccountSubscriptions should generate nothing without an account address")
	setTestCredentials(ex, &accounts.Credentials{Key: testAccountAddress, SubAccount: testSubAccountAddress})
	account, err = ex.generateAccountSubscriptions()
	require.NoError(t, err, "generateAccountSubscriptions must not error for a subaccount")
	assert.Len(t, account, 2, "generateAccountSubscriptions should generate the account feeds of a subaccount")
}

func TestGenerateConnectionSubscriptionsApart(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	book := testBookSubscription(20, nil)
	book.Enabled = true
	userFills := &subscription.Subscription{Enabled: true, Channel: wsChannelUserFills, Authenticated: true}
	aggregated := &subscription.Subscription{Enabled: true, Channel: wsChannelUserFills, Authenticated: true, Params: map[string]any{"aggregateByTime": true}}
	ex.Features.Subscriptions = subscription.List{book, userFills, aggregated, {Enabled: true, Channel: "bogus", Authenticated: true}}
	public, err := ex.generatePublicSubscriptions()
	require.NoError(t, err, "generatePublicSubscriptions must not fail for the account feeds' configuration")
	require.Len(t, public, 1, "generatePublicSubscriptions must generate the market feed")
	_, err = ex.generateAccountSubscriptions()
	assert.ErrorIs(t, err, subscription.ErrExclusiveSubscription, "generateAccountSubscriptions should refuse two userFills variants")
	ex.API.AuthenticatedWebsocketSupport = false
	account, err := ex.generateAccountSubscriptions()
	require.NoError(t, err, "generateAccountSubscriptions must not error without authenticated websocket support")
	assert.Empty(t, account, "generateAccountSubscriptions should generate nothing without authenticated websocket support")
}

func TestAccountFeedsAvailable(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	require.NoError(t, ex.accountFeedsAvailable(t.Context()), "accountFeedsAvailable must not error with an account address and support enabled")
	ex.API.AuthenticatedWebsocketSupport = false
	assert.ErrorIs(t, ex.accountFeedsAvailable(t.Context()), errWebsocketAccountFeedsDisabled, "accountFeedsAvailable should require authenticated websocket support")
	ex.API.AuthenticatedWebsocketSupport = true
	setTestCredentials(ex, &accounts.Credentials{Key: "invalid"})
	assert.ErrorIs(t, ex.accountFeedsAvailable(t.Context()), errInvalidAddress, "accountFeedsAvailable should require an account address")
}

// reservedAccountFeedTypes returns the subscription type of each account feed the exchange keeps room for
func reservedAccountFeedTypes(ex *Exchange) []string {
	ex.websocketPendingMu.Lock()
	defer ex.websocketPendingMu.Unlock()
	feedTypes := make([]string, 0, len(ex.websocketAccountFeeds))
	for _, feed := range ex.websocketAccountFeeds {
		feedTypes = append(feedTypes, feed.options.Type)
	}
	return feedTypes
}

func TestReserveAccountFeeds(t *testing.T) {
	t.Parallel()
	myOrders := &subscription.Subscription{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true}
	for _, tc := range []struct {
		name       string
		configured subscription.List
		exp        []string
	}{
		{name: "defaults", configured: defaultSubscriptions.Clone(), exp: []string{wsChannelOrderUpdates, wsSubscriptionUserEvents}},
		{name: "one feed configured twice", configured: subscription.List{myOrders.Clone(), {Enabled: true, Channel: wsChannelOrderUpdates, Authenticated: true}}, exp: []string{wsChannelOrderUpdates}},
		{name: "a registered DEX", configured: subscription.List{{Enabled: true, Channel: wsChannelOpenOrders, Authenticated: true, Params: map[string]any{"dex": "xyz"}}}, exp: []string{wsChannelOpenOrders}},
		{name: "an unregistered DEX, checked when subscribed", configured: subscription.List{myOrders.Clone(), {Enabled: true, Channel: wsChannelOpenOrders, Authenticated: true, Params: map[string]any{"dex": "abc"}}}, exp: []string{wsChannelOrderUpdates, wsChannelOpenOrders}},
		{name: "an invalid option", configured: subscription.List{myOrders.Clone(), {Enabled: true, Channel: wsChannelUserFills, Authenticated: true, Params: map[string]any{"aggregateByTime": "bad"}}}, exp: []string{wsChannelOrderUpdates}},
		{name: "a market feed set authenticated", configured: subscription.List{myOrders.Clone(), {Enabled: true, Channel: wsChannelAllMids, Authenticated: true}}, exp: []string{wsChannelOrderUpdates}},
		{name: "an empty pair", configured: subscription.List{myOrders.Clone(), {Enabled: true, Channel: wsChannelActiveAssetData, Authenticated: true, Asset: asset.PerpetualContract, Pairs: currency.Pairs{currency.EMPTYPAIR}}}, exp: []string{wsChannelOrderUpdates}},
		{name: "an unsupported channel", configured: subscription.List{myOrders.Clone(), {Enabled: true, Channel: "bogus", Authenticated: true}}, exp: []string{}},
		{name: "exclusive variants", configured: subscription.List{myOrders.Clone(), {Enabled: true, Channel: wsChannelUserFills, Authenticated: true}, {Enabled: true, Channel: wsChannelUserFills, Authenticated: true, Params: map[string]any{"aggregateByTime": true}}}, exp: []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newWebsocketTestExchange(t)
			ex.Features.Subscriptions = tc.configured
			_, err := ex.generatePublicSubscriptions()
			require.NoError(t, err, "generatePublicSubscriptions must not error")
			assert.Equal(t, tc.exp, reservedAccountFeedTypes(ex), "generatePublicSubscriptions should keep room for each account feed the account connection subscribes, once")
		})
	}

	// No request delays the public connection, as the DEX an account feed names is checked when it is subscribed
	counted, requests := newCountingServerExchange(t, nil)
	counted.API.AuthenticatedWebsocketSupport = true
	counted.Features.Subscriptions = subscription.List{{Enabled: true, Channel: wsChannelOpenOrders, Authenticated: true, Params: map[string]any{"dex": "xyz"}}}
	_, err := counted.generatePublicSubscriptions()
	require.NoError(t, err, "generatePublicSubscriptions must not error")
	assert.Equal(t, []string{wsChannelOpenOrders}, reservedAccountFeedTypes(counted), "generatePublicSubscriptions should keep room for an account feed naming a DEX")
	assert.Empty(t, requests(), "generatePublicSubscriptions should make no request to keep room for the account feeds")

	ex := newWebsocketTestExchange(t)
	_, err = ex.generatePublicSubscriptions()
	require.NoError(t, err, "generatePublicSubscriptions must not error")
	require.Len(t, reservedAccountFeedTypes(ex), 2, "generatePublicSubscriptions must keep room for the default account feeds")
	ex.Features.Subscriptions = subscription.List{myOrders.Clone()}
	_, err = ex.generatePublicSubscriptions()
	require.NoError(t, err, "generatePublicSubscriptions must not error")
	assert.Equal(t, []string{wsChannelOrderUpdates}, reservedAccountFeedTypes(ex), "generatePublicSubscriptions should keep room only for the account feed still configured")
	ex.API.AuthenticatedWebsocketSupport = false
	_, err = ex.generatePublicSubscriptions()
	require.NoError(t, err, "generatePublicSubscriptions must not error without authenticated websocket support")
	assert.Empty(t, reservedAccountFeedTypes(ex), "generatePublicSubscriptions should keep no room without authenticated websocket support")
	ex.API.AuthenticatedWebsocketSupport = true
	setTestCredentials(ex, &accounts.Credentials{Key: "invalid"})
	_, err = ex.generatePublicSubscriptions()
	require.NoError(t, err, "generatePublicSubscriptions must not error without an account address")
	assert.Empty(t, reservedAccountFeedTypes(ex), "generatePublicSubscriptions should keep no room without an account address")
}

func TestGeneratePublicSubscriptionsKeepsRoomForAccountFeeds(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	market := make(subscription.List, maximumWebsocketSubscriptions)
	for i := range market {
		market[i] = &subscription.Subscription{Enabled: true, Channel: wsChannelAllMids, Params: map[string]any{"dex": "d" + strconv.Itoa(i)}}
	}
	ex.Features.Subscriptions = slices.Concat(market, subscription.List{
		{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true},
		{Enabled: true, Channel: subscription.MyTradesChannel, Authenticated: true},
	})
	subs, err := ex.generatePublicSubscriptions()
	assert.ErrorIs(t, err, websocket.ErrSubscriptionPartial, "generatePublicSubscriptions should report the market feeds it leaves out")
	require.Len(t, subs, maximumWebsocketSubscriptions-2, "generatePublicSubscriptions must leave room for the two account feeds")
	assert.Equal(t, "d997", subs[len(subs)-1].Params["dex"], "generatePublicSubscriptions should leave out the market feeds configured last")

	ex.API.AuthenticatedWebsocketSupport = false
	subs, err = ex.generatePublicSubscriptions()
	require.NoError(t, err, "generatePublicSubscriptions must not error while no account feed needs room")
	assert.Len(t, subs, maximumWebsocketSubscriptions, "generatePublicSubscriptions should generate every market feed while no account feed needs room")

	// A feed configured twice takes no slot, and an invalid one is left out, so every valid market feed is kept
	ex.API.AuthenticatedWebsocketSupport = true
	ex.Features.Subscriptions = slices.Concat(market[:maximumWebsocketSubscriptions-2], subscription.List{
		market[0],
		{Enabled: true, Channel: wsChannelAllMids, Params: map[string]any{"dex": 42}},
		{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true},
		{Enabled: true, Channel: subscription.MyTradesChannel, Authenticated: true},
	})
	subs, err = ex.generatePublicSubscriptions()
	assert.ErrorIs(t, err, websocket.ErrSubscriptionPartial, "generatePublicSubscriptions should report an invalid market feed it leaves out as a partial generation")
	assert.ErrorIs(t, err, errWebsocketParameter, "generatePublicSubscriptions should report why it refused an invalid market feed")
	assert.Len(t, subs, maximumWebsocketSubscriptions-1, "generatePublicSubscriptions should keep every valid market feed that fits and a feed configured twice")

	account := make(subscription.List, maximumWebsocketSubscriptions+1)
	for i := range account {
		account[i] = &subscription.Subscription{Enabled: true, Channel: wsChannelOpenOrders, Authenticated: true, Params: map[string]any{"dex": "d" + strconv.Itoa(i)}}
	}
	ex.Features.Subscriptions = slices.Concat(market[:1], account)
	subs, err = ex.generatePublicSubscriptions()
	assert.ErrorIs(t, err, websocket.ErrSubscriptionPartial, "generatePublicSubscriptions should report a market feed left out by account feeds filling the limit")
	assert.Empty(t, subs, "generatePublicSubscriptions should leave out every market feed while the account feeds fill the limit")
}

func TestWebsocketFeedIdentity(t *testing.T) {
	t.Parallel()
	subs := subscription.List{
		{Channel: wsChannelAllMids},
		{Channel: wsChannelAllMids, Params: map[string]any{"dex": "xyz"}},
		{Channel: wsChannelAllMids, Asset: asset.Spot},
		{Channel: subscription.AllTradesChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}},
		{Channel: subscription.AllTradesChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{currency.NewPairWithDelimiter("btc", "usdc", "/")}},
		{Channel: subscription.AllTradesChannel, Asset: asset.Spot, Pairs: currency.Pairs{perpetualPair}},
		{Channel: subscription.AllTradesChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{testBuilderPair}},
		{Channel: subscription.TickerChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}},
		{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotPair}},
		{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPair(currency.HYPE, currency.USDE)}},
		{Channel: subscription.CandlesChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}, Interval: kline.OneMin},
		{Channel: subscription.CandlesChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}, Interval: kline.FiveMin},
		{Channel: subscription.CandlesChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}, Interval: kline.OneHour},
		{Channel: subscription.CandlesChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}, Interval: kline.OneMonth},
		{Channel: subscription.OrderbookChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}, Levels: 20},
		{Channel: subscription.OrderbookChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}, Levels: 5},
		{Channel: wsChannelActiveAssetData, Authenticated: true, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}},
	}
	for _, a := range subs {
		key := newWebsocketSubscriptionKey(a)
		id, ok := key.identity()
		require.Truef(t, ok, "identity must exist for %s", a)
		for _, b := range subs {
			other := newWebsocketSubscriptionKey(b)
			otherID, _ := other.identity()
			assert.Equalf(t, key.Match(other), id == otherID, "identity should agree with Match for %s and %s", a, b)
		}
	}
	_, ok := newWebsocketSubscriptionKey(&subscription.Subscription{Channel: "bogus"}).identity()
	assert.False(t, ok, "identity should not exist for a key that matches nothing")
	_, ok = newWebsocketSubscriptionKey(&subscription.Subscription{Channel: subscription.AllTradesChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair, testBuilderPair}}).identity()
	assert.False(t, ok, "identity should not exist for a market feed naming two markets")
}

func TestCapMarketFeeds(t *testing.T) {
	t.Parallel()
	feed := func(dex any) *subscription.Subscription {
		return &subscription.Subscription{Channel: wsChannelAllMids, Params: map[string]any{"dex": dex}}
	}
	a, b, c, d, invalid := feed("a"), feed("b"), feed("c"), feed("d"), feed(42)
	// Refused for its asset, though its key matches c's, so keeping it would keep a held c
	shield := feed("c")
	shield.Asset = asset.Spot
	noAsset := &subscription.Subscription{Channel: subscription.AllTradesChannel, Pairs: currency.Pairs{perpetualPair}}
	// The invalid entry comes first, so that it would take a slot while there is room if refused entries counted
	subs := subscription.List{invalid, a, b, a.Clone(), shield, noAsset, c, c.Clone(), d}
	kept, leftOut, refused := e.capMarketFeeds(subs, 2)
	assert.Equal(t, subscription.List{a, b, subs[3]}, kept, "capMarketFeeds should keep the feeds that fit and a feed configured twice")
	assert.Equal(t, 2, leftOut, "capMarketFeeds should count each feed it leaves out once")
	assert.ErrorIs(t, refused, errWebsocketParameter, "capMarketFeeds should report a feed with an invalid parameter")
	assert.ErrorIs(t, refused, subscription.ErrNotSupported, "capMarketFeeds should report a feed that takes no asset set with one")
	assert.ErrorIs(t, refused, asset.ErrNotSupported, "capMarketFeeds should report a market feed without an asset")
	kept, leftOut, _ = e.capMarketFeeds(subs, 0)
	assert.Empty(t, kept, "capMarketFeeds should leave out every feed without room")
	assert.Equal(t, 4, leftOut, "capMarketFeeds should count every distinct valid feed it leaves out")
}

func TestAwaitedAccountFeeds(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	_, err := ex.generatePublicSubscriptions()
	require.NoError(t, err, "generatePublicSubscriptions must not error")
	myOrders := &subscription.Subscription{Channel: subscription.MyOrdersChannel, Authenticated: true}
	myTrades := &subscription.Subscription{Channel: subscription.MyTradesChannel, Authenticated: true}
	notification := &subscription.Subscription{Channel: wsChannelNotification, Authenticated: true}
	market := &subscription.Subscription{Channel: wsChannelAllMids}
	for _, tc := range []struct {
		name string
		held subscription.List
		exp  int
	}{
		{name: "nothing held", exp: 2},
		{name: "a market feed held", held: subscription.List{market}, exp: 2},
		{name: "a reserved feed's channel held as a market feed", held: subscription.List{{Channel: subscription.MyOrdersChannel}}, exp: 2},
		{name: "one reserved feed held", held: subscription.List{market, myOrders}, exp: 1},
		{name: "every reserved feed held", held: subscription.List{myOrders, myTrades}, exp: 0},
		// An account feed subscribed at runtime stays held while a reserved feed is subscribed again at runtime
		{name: "an unconfigured account feed held", held: subscription.List{notification}, exp: 2},
		{name: "a reserved and an unconfigured account feed held", held: subscription.List{market, myOrders, notification}, exp: 1},
	} {
		assert.Equalf(t, tc.exp, ex.awaitedAccountFeeds(t.Context(), tc.held), "awaitedAccountFeeds should leave %d slots with %s", tc.exp, tc.name)
	}

	// myTrades stays held after the configuration drops it until the account connection's flush removes it, which comes
	// after the market feeds subscribe
	ex.Features.Subscriptions = subscription.List{{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true}}
	_, err = ex.generatePublicSubscriptions()
	require.NoError(t, err, "generatePublicSubscriptions must not error")
	assert.Equal(t, 1, ex.awaitedAccountFeeds(t.Context(), subscription.List{market}), "awaitedAccountFeeds should leave a slot for the one reserved feed")
	assert.Equal(t, 1, ex.awaitedAccountFeeds(t.Context(), subscription.List{myTrades}), "awaitedAccountFeeds should leave the reserved feed's slot beside a held feed the configuration dropped")
	assert.Zero(t, ex.awaitedAccountFeeds(t.Context(), subscription.List{myOrders, myTrades}), "awaitedAccountFeeds should leave no slots once the reserved feed is held")
	ex.API.AuthenticatedWebsocketSupport = false
	assert.Zero(t, ex.awaitedAccountFeeds(t.Context(), nil), "awaitedAccountFeeds should leave no slots without authenticated websocket support")
}

func TestKeyWebsocketSubscriptions(t *testing.T) {
	t.Parallel()
	fresh := &subscription.Subscription{Channel: wsChannelAllMids, Params: map[string]any{"dex": "xyz"}}
	held := &subscription.Subscription{Channel: wsChannelAllMids}
	heldKey := newWebsocketSubscriptionKey(held)
	heldKey.feed = WebsocketSubscription{Type: wsChannelAllMids}
	held.SetKey(heldKey)
	keyWebsocketSubscriptions(subscription.List{fresh, nil, held})
	assert.Equal(t, newWebsocketSubscriptionKey(fresh), fresh.EnsureKeyed(), "keyWebsocketSubscriptions should key a subscription by the feed it selects")
	assert.Same(t, heldKey, held.EnsureKeyed(), "keyWebsocketSubscriptions should keep the key of a subscription keyed by its feed, which may be held")
}

func TestSubscribeToWebsocketChannels(t *testing.T) {
	t.Parallel()
	ex, venue := newLocalWebsocketTestExchange(t, subscription.List{})
	require.NoError(t, ex.Websocket.Enable(t.Context()), "Enable must connect to the local websocket")
	assert.ErrorIs(t, ex.SubscribeToWebsocketChannels(subscription.List{nil}), common.ErrNilPointer, "SubscribeToWebsocketChannels should reject a nil subscription")
	plain := &subscription.Subscription{Channel: wsChannelAllMids}
	variant := &subscription.Subscription{Channel: wsChannelAllMids, Params: map[string]any{"dex": "xyz"}}
	require.NoError(t, ex.SubscribeToWebsocketChannels(subscription.List{plain}), "SubscribeToWebsocketChannels must not error")
	require.NoError(t, ex.SubscribeToWebsocketChannels(subscription.List{variant}), "SubscribeToWebsocketChannels must subscribe a variant beside the held feed")
	assert.NotNil(t, ex.Websocket.GetSubscription(newWebsocketSubscriptionKey(plain)), "SubscribeToWebsocketChannels should hold the plain feed")
	assert.NotNil(t, ex.Websocket.GetSubscription(newWebsocketSubscriptionKey(variant)), "SubscribeToWebsocketChannels should hold the variant")
	duplicate := &subscription.Subscription{Channel: wsChannelAllMids, Params: map[string]any{"dex": "xyz"}}
	assert.ErrorIs(t, ex.SubscribeToWebsocketChannels(subscription.List{duplicate}), subscription.ErrDuplicate, "SubscribeToWebsocketChannels should refuse a held variant")
	exp := []string{
		`{"method":"subscribe","subscription":{"type":"allMids"}}`,
		`{"method":"subscribe","subscription":{"type":"allMids","dex":"xyz"}}`,
	}
	assert.Equal(t, exp, venue.received(), "SubscribeToWebsocketChannels should send each feed's subscribe")

	fast, slow := testBookSubscription(5, nil), testBookSubscription(20, nil)
	fast.QualifiedChannel, slow.QualifiedChannel = "l2Book:fast", "l2Book:slow"
	assert.ErrorIs(t, ex.SubscribeToWebsocketChannels(subscription.List{fast, slow}), subscription.ErrExclusiveSubscription, "SubscribeToWebsocketChannels should validate subscriptions already expanded")
	tooMany := make(subscription.List, maximumWebsocketSubscriptions-1)
	for i := range tooMany {
		tooMany[i] = &subscription.Subscription{Channel: wsChannelAllMids, Params: map[string]any{"dex": "d" + strconv.Itoa(i)}}
	}
	assert.ErrorIs(t, ex.SubscribeToWebsocketChannels(tooMany), errWebsocketSubscriptionsLimit, "SubscribeToWebsocketChannels should refuse subscriptions past Hyperliquid's limit before sending any")
	assert.Equal(t, exp, venue.received(), "SubscribeToWebsocketChannels should send nothing it refuses")

	ex.API.AuthenticatedWebsocketSupport = false
	for _, tc := range []struct{ accountFeed, marketFeed *subscription.Subscription }{
		{&subscription.Subscription{Channel: wsChannelNotification, Authenticated: true}, &subscription.Subscription{Channel: wsChannelAllMids, Params: map[string]any{"dex": "flx"}}},
		{&subscription.Subscription{Channel: wsChannelNotification, Authenticated: true, QualifiedChannel: wsChannelNotification}, testBookSubscription(0, nil)},
	} {
		assert.ErrorIsf(t, ex.SubscribeToWebsocketChannels(subscription.List{tc.accountFeed}), errWebsocketAccountFeedsDisabled, "SubscribeToWebsocketChannels should refuse account feed %q without authenticated websocket support", tc.accountFeed.QualifiedChannel)
		assert.ErrorIsf(t, ex.SubscribeToWebsocketChannels(subscription.List{tc.accountFeed, tc.marketFeed}), errWebsocketAccountFeedsDisabled, "SubscribeToWebsocketChannels should refuse account feed %q beside a market feed without authenticated websocket support", tc.accountFeed.QualifiedChannel)
	}
	exp = slices.Concat(exp, []string{`{"method":"subscribe","subscription":{"type":"allMids","dex":"flx"}}`, `{"method":"subscribe","subscription":{"type":"l2Book","coin":"BTC"}}`})
	assert.Equal(t, exp, venue.received(), "SubscribeToWebsocketChannels should subscribe only the market feeds beside the account feeds it refuses")
	ex.API.AuthenticatedWebsocketSupport = true
	assert.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "the account connection should not be up before an account feed is subscribed")
	notification := &subscription.Subscription{Channel: wsChannelNotification, Authenticated: true}
	require.NoError(t, ex.SubscribeToWebsocketChannels(subscription.List{notification}), "SubscribeToWebsocketChannels must open the account connection for an account feed")
	assert.True(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "SubscribeToWebsocketChannels should make account feeds usable once their connection is up")
	assert.NotNil(t, ex.Websocket.GetSubscription(newWebsocketSubscriptionKey(notification)), "SubscribeToWebsocketChannels should hold the account feed")
	received := venue.received()
	assert.JSONEq(t, `{"method":"subscribe","subscription":{"type":"notification","user":"`+testAccountAddress+`"}}`, received[len(received)-1], "SubscribeToWebsocketChannels should subscribe the account feed")
}

func TestUnsubscribeToWebsocketChannels(t *testing.T) {
	t.Parallel()
	ex, venue := newLocalWebsocketTestExchange(t, subscription.List{})
	require.NoError(t, ex.Websocket.Enable(t.Context()), "Enable must connect to the local websocket")
	plain := &subscription.Subscription{Channel: wsChannelAllMids}
	variant := &subscription.Subscription{Channel: wsChannelAllMids, Params: map[string]any{"dex": "xyz"}}
	require.NoError(t, ex.SubscribeToWebsocketChannels(subscription.List{plain, variant, testBookSubscription(0, nil)}), "SubscribeToWebsocketChannels must not error")
	assert.ErrorIs(t, ex.UnsubscribeToWebsocketChannels(subscription.List{nil}), common.ErrNilPointer, "UnsubscribeToWebsocketChannels should reject a nil subscription")
	unheld := &subscription.Subscription{Channel: wsChannelAllMids, Params: map[string]any{"dex": "flx"}}
	assert.ErrorIs(t, ex.UnsubscribeToWebsocketChannels(subscription.List{unheld}), subscription.ErrNotFound, "UnsubscribeToWebsocketChannels should report a variant that is not held")
	require.NoError(t, ex.UnsubscribeToWebsocketChannels(subscription.List{variant.Clone()}), "UnsubscribeToWebsocketChannels must remove the held variant")
	assert.NotNil(t, ex.Websocket.GetSubscription(newWebsocketSubscriptionKey(plain)), "UnsubscribeToWebsocketChannels should leave the plain feed held")
	assert.Nil(t, ex.Websocket.GetSubscription(newWebsocketSubscriptionKey(variant)), "UnsubscribeToWebsocketChannels should remove the variant")
	equivalent := &subscription.Subscription{Channel: wsChannelOrderbook, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}, Levels: 20}
	require.NoError(t, ex.UnsubscribeToWebsocketChannels(subscription.List{equivalent}), "UnsubscribeToWebsocketChannels must find a book held under its standard channel by its Hyperliquid name")
	require.NoError(t, ex.UnsubscribeToWebsocketChannels(subscription.List{plain.Clone()}), "UnsubscribeToWebsocketChannels must remove the plain feed")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "UnsubscribeToWebsocketChannels should remove each feed it unsubscribes")
	exp := []string{
		`{"method":"subscribe","subscription":{"type":"allMids"}}`,
		`{"method":"subscribe","subscription":{"type":"allMids","dex":"xyz"}}`,
		`{"method":"subscribe","subscription":{"type":"l2Book","coin":"BTC"}}`,
		`{"method":"unsubscribe","subscription":{"type":"allMids","dex":"xyz"}}`,
		`{"method":"unsubscribe","subscription":{"type":"l2Book","coin":"BTC"}}`,
		`{"method":"unsubscribe","subscription":{"type":"allMids"}}`,
	}
	assert.Equal(t, exp, venue.received(), "UnsubscribeToWebsocketChannels should send the unsubscribe of each held feed it removes")
}

// TestWebsocketFlushReplacesFeedVariants configures another variant of a held feed and flushes the subscriptions through
// the manager, which generates them, unsubscribes the held variant and subscribes the configured one
func TestWebsocketFlushReplacesFeedVariants(t *testing.T) {
	t.Parallel()
	userFills := func(params map[string]any) *subscription.Subscription {
		return &subscription.Subscription{Enabled: true, Channel: wsChannelUserFills, Authenticated: true, Params: params}
	}
	for _, tc := range []struct {
		name       string
		held, next *subscription.Subscription
		requests   []string
	}{
		{
			name: "l2Book depth",
			held: testBookSubscription(20, nil),
			next: testBookSubscription(5, nil),
			requests: []string{
				`{"method":"subscribe","subscription":{"type":"l2Book","coin":"BTC"}}`,
				`{"method":"unsubscribe","subscription":{"type":"l2Book","coin":"BTC"}}`,
				`{"method":"subscribe","subscription":{"type":"l2Book","coin":"BTC","fast":true}}`,
			},
		},
		{
			name: "userFills aggregateByTime",
			held: userFills(nil),
			next: userFills(map[string]any{"aggregateByTime": true}),
			requests: []string{
				`{"method":"subscribe","subscription":{"type":"userFills","user":"` + testAccountAddress + `"}}`,
				`{"method":"unsubscribe","subscription":{"type":"userFills","user":"` + testAccountAddress + `"}}`,
				`{"method":"subscribe","subscription":{"type":"userFills","user":"` + testAccountAddress + `","aggregateByTime":true}}`,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.held.Enabled, tc.next.Enabled = true, true
			ex, venue := newLocalWebsocketTestExchange(t, subscription.List{tc.held})
			require.NoError(t, ex.Websocket.Enable(t.Context()), "Enable must subscribe the configured feed")
			ex.Features.Subscriptions = subscription.List{tc.next}
			require.NoError(t, ex.Websocket.FlushChannels(t.Context()), "FlushChannels must replace the held variant with the configured one")
			assert.Nil(t, ex.Websocket.GetSubscription(newWebsocketSubscriptionKey(tc.held)), "FlushChannels should remove the held variant")
			assert.NotNil(t, ex.Websocket.GetSubscription(newWebsocketSubscriptionKey(tc.next)), "FlushChannels should hold the configured variant")
			assert.Equal(t, tc.requests, venue.received(), "FlushChannels should unsubscribe the held variant, then subscribe the configured one")
		})
	}
}

func TestWebsocketConnectRefusesConflictingConfiguration(t *testing.T) {
	t.Parallel()
	fast, slow := testBookSubscription(5, nil), testBookSubscription(20, nil)
	fast.Enabled, slow.Enabled = true, true
	ex, venue := newLocalWebsocketTestExchange(t, subscription.List{fast, slow})
	assert.ErrorIs(t, ex.Websocket.Enable(t.Context()), subscription.ErrExclusiveSubscription, "Enable should refuse a configuration holding two variants of a market's l2Book")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "Enable should hold nothing of a refused configuration")
	assert.Empty(t, venue.received(), "Enable should send nothing of a refused configuration")
}

func TestGetSubscriptionTemplate(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	tmpl, err := ex.GetSubscriptionTemplate(nil)
	require.NoError(t, err, "GetSubscriptionTemplate must not error")
	require.NotNil(t, tmpl, "GetSubscriptionTemplate must return a template")
	subs, err := subscription.List{
		{Channel: wsChannelAllMids},
		{Channel: wsChannelBestBidOffer, Asset: asset.Spot},
		{Channel: wsChannelUserFills, Authenticated: true},
	}.ExpandTemplates(ex)
	require.NoError(t, err, "ExpandTemplates must not error")
	spotPairs, err := ex.GetEnabledPairs(asset.Spot)
	require.NoError(t, err, "GetEnabledPairs must not error")
	exp := subscription.List{
		{Channel: wsChannelAllMids, QualifiedChannel: wsChannelAllMids},
		{Channel: wsChannelBestBidOffer, Asset: asset.Spot, Pairs: spotPairs, QualifiedChannel: "bbo:spot:HYPE-USDC"},
		{Channel: wsChannelUserFills, Authenticated: true, QualifiedChannel: wsChannelUserFills},
	}
	testsubs.EqualLists(t, exp, subs)

	_, err = subscription.List{{Channel: "bogus"}}.ExpandTemplates(ex)
	assert.ErrorIs(t, err, subscription.ErrNotSupported, "the template should reject an unsupported channel")
}

// testBookSubscription returns a BTC perpetual orderbook subscription of the levels and parameters
func testBookSubscription(levels int, params map[string]any) *subscription.Subscription {
	return &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}, Levels: levels, Params: params}
}

func TestWebsocketSubscriptionKeyMatch(t *testing.T) {
	t.Parallel()
	allMids := &subscription.Subscription{Channel: wsChannelAllMids}
	builderAllMids := &subscription.Subscription{Channel: wsChannelAllMids, Params: map[string]any{"dex": "xyz"}}
	account := func(channel string, params map[string]any) *subscription.Subscription {
		return &subscription.Subscription{Channel: channel, Authenticated: true, Params: params}
	}
	candles := func(interval kline.Interval) *subscription.Subscription {
		return &subscription.Subscription{Channel: subscription.CandlesChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}, Interval: interval}
	}
	trades := &subscription.Subscription{Channel: subscription.AllTradesChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}}
	for _, tc := range []struct {
		name string
		a, b *subscription.Subscription
		exp  bool
	}{
		{name: "allMids without a dex and with an empty dex", a: allMids, b: &subscription.Subscription{Channel: wsChannelAllMids, Params: map[string]any{"dex": ""}}, exp: true},
		{name: "allMids without a dex and with a null dex", a: allMids, b: &subscription.Subscription{Channel: wsChannelAllMids, Params: map[string]any{"dex": nil}}, exp: true},
		{name: "allMids of two DEXs", a: allMids, b: builderAllMids},
		{name: "openOrders of two DEXs", a: account(wsChannelOpenOrders, nil), b: account(wsChannelOpenOrders, map[string]any{"dex": "xyz"})},
		{name: "clearinghouseState of two DEXs", a: account(wsChannelClearinghouseState, nil), b: account(wsChannelClearinghouseState, map[string]any{"dex": "xyz"})},
		{name: "twapStates of two DEXs", a: account(wsChannelTWAPStates, nil), b: account(wsChannelTWAPStates, map[string]any{"dex": "xyz"})},
		{name: "userFills with and without aggregation", a: account(wsChannelUserFills, nil), b: account(wsChannelUserFills, map[string]any{"aggregateByTime": true})},
		{name: "userFills without aggregation and with it turned off", a: account(wsChannelUserFills, nil), b: account(wsChannelUserFills, map[string]any{"aggregateByTime": false}), exp: true},
		{name: "spotState with and without portfolio margin", a: account(wsChannelSpotState, nil), b: account(wsChannelSpotState, map[string]any{"ignorePortfolioMargin": true})},
		{name: "l2Book of two significant figure settings", a: testBookSubscription(0, map[string]any{"nSigFigs": 5}), b: testBookSubscription(0, map[string]any{"nSigFigs": 4})},
		{name: "l2Book significant figures as an integer and a JSON number", a: testBookSubscription(0, map[string]any{"nSigFigs": 5}), b: testBookSubscription(0, map[string]any{"nSigFigs": 5.0}), exp: true},
		{name: "l2Book significant figures as signed and unsigned integers", a: testBookSubscription(0, map[string]any{"nSigFigs": 5}), b: testBookSubscription(0, map[string]any{"nSigFigs": uint64(5)}), exp: true},
		{name: "l2Book of two mantissas", a: testBookSubscription(0, map[string]any{"nSigFigs": 5, "mantissa": 2}), b: testBookSubscription(0, map[string]any{"nSigFigs": 5, "mantissa": 5})},
		{name: "l2Book of 20 levels by default and by count", a: testBookSubscription(0, nil), b: testBookSubscription(20, nil), exp: true},
		{name: "fast and slow l2Book", a: testBookSubscription(5, nil), b: testBookSubscription(20, nil)},
		{name: "l2Book by its standard and its Hyperliquid channel name", a: testBookSubscription(0, nil), b: &subscription.Subscription{Channel: wsChannelOrderbook, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}}, exp: true},
		{name: "l2Book of two markets", a: testBookSubscription(0, nil), b: &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{testBuilderPair}}},
		{name: "l2Book of one pair of two asset types", a: &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotPair}}, b: &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{spotPair}}},
		{name: "candles of two intervals", a: candles(kline.OneMin), b: candles(kline.FiveMin)},
		{name: "candles of one interval", a: candles(kline.OneMin), b: candles(kline.OneMin), exp: true},
		{name: "trades whatever levels and interval, which they do not take", a: trades, b: &subscription.Subscription{Channel: wsChannelTrades, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}, Levels: 5, Interval: kline.OneMin}, exp: true},
		{name: "userFills whatever a dex, which it does not take", a: account(wsChannelUserFills, nil), b: account(wsChannelUserFills, map[string]any{"dex": "xyz"}), exp: true},
		{name: "allMids whatever pairs, which it does not take", a: allMids, b: &subscription.Subscription{Channel: wsChannelAllMids, Pairs: currency.Pairs{perpetualPair}}, exp: true},
		{name: "an invalid parameter, which matches nothing", a: &subscription.Subscription{Channel: wsChannelAllMids, Params: map[string]any{"dex": 1}}, b: &subscription.Subscription{Channel: wsChannelAllMids, Params: map[string]any{"dex": 1}}},
		{name: "an unsupported channel, which matches nothing", a: &subscription.Subscription{Channel: "bogus"}, b: &subscription.Subscription{Channel: "bogus"}},
	} {
		a, b := newWebsocketSubscriptionKey(tc.a), newWebsocketSubscriptionKey(tc.b)
		assert.Equalf(t, tc.exp, a.Match(b), "Match should compare the feeds of %s", tc.name)
		assert.Equalf(t, tc.exp, b.Match(a), "Match should compare the feeds of %s either way round", tc.name)
	}
	assert.True(t, newWebsocketSubscriptionKey(builderAllMids).Match(&subscription.ExactKey{Subscription: builderAllMids}), "Match should compare the feed of another key type's subscription")
	assert.False(t, newWebsocketSubscriptionKey(allMids).Match(&subscription.ExactKey{Subscription: builderAllMids}), "Match should tell apart the feed of another key type's subscription")
	assert.False(t, newWebsocketSubscriptionKey(allMids).Match(nil), "Match should not match a nil key")
	assert.False(t, newWebsocketSubscriptionKey(allMids).Match(subscription.ExactKey{}), "Match should not match a key without a subscription")
	assert.False(t, newWebsocketSubscriptionKey(allMids).Match((*websocketSubscriptionKey)(nil)), "Match should not match a nil feed key")
}

func TestWebsocketSubscriptionKeyGetSubscription(t *testing.T) {
	t.Parallel()
	sub := &subscription.Subscription{Channel: wsChannelAllMids}
	assert.Same(t, sub, newWebsocketSubscriptionKey(sub).GetSubscription(), "GetSubscription should return the keyed subscription")
}

func TestWebsocketSubscriptionKeyString(t *testing.T) {
	t.Parallel()
	allMids := &subscription.Subscription{Channel: wsChannelAllMids}
	assert.Equal(t, subscription.ExactKey{Subscription: allMids}.String(), newWebsocketSubscriptionKey(allMids).String(), "String should name a default feed as ExactKey does")
	for _, tc := range []struct {
		sub *subscription.Subscription
		exp string
	}{
		{sub: &subscription.Subscription{Channel: wsChannelAllMids, Params: map[string]any{"dex": "xyz"}}, exp: " [dex=xyz]"},
		{sub: testBookSubscription(5, map[string]any{"nSigFigs": 5, "mantissa": 2}), exp: " [nSigFigs=5 mantissa=2 fast]"},
		{sub: &subscription.Subscription{Channel: wsChannelUserFills, Authenticated: true, Params: map[string]any{"aggregateByTime": true}}, exp: " [aggregateByTime]"},
		{sub: &subscription.Subscription{Channel: wsChannelSpotState, Authenticated: true, Params: map[string]any{"ignorePortfolioMargin": true}}, exp: " [ignorePortfolioMargin]"},
	} {
		assert.Equalf(t, subscription.ExactKey{Subscription: tc.sub}.String()+tc.exp, newWebsocketSubscriptionKey(tc.sub).String(), "String should name the options that set %s's feed apart", tc.sub.Channel)
	}
}

func TestNewWebsocketSubscriptionKey(t *testing.T) {
	t.Parallel()
	sub := &subscription.Subscription{Channel: wsChannelAllMids, Params: map[string]any{"dex": "xyz"}}
	exp := &websocketSubscriptionKey{Subscription: sub, options: WebsocketSubscription{Type: wsChannelAllMids, DEX: "xyz"}, valid: true}
	assert.Equal(t, exp, newWebsocketSubscriptionKey(sub), "newWebsocketSubscriptionKey should key a subscription by its feed's options")
	invalid := &subscription.Subscription{Channel: wsChannelAllMids, Params: map[string]any{"dex": 1}}
	assert.False(t, newWebsocketSubscriptionKey(invalid).valid, "newWebsocketSubscriptionKey should mark a subscription with an invalid parameter as matching nothing")
	assert.False(t, newWebsocketSubscriptionKey(nil).valid, "newWebsocketSubscriptionKey should mark a nil subscription as matching nothing")
}

func TestWebsocketFeedOptions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		sub  *subscription.Subscription
		exp  WebsocketSubscription
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "unsupported channel", sub: &subscription.Subscription{Channel: "bogus"}, err: subscription.ErrNotSupported},
		{name: "invalid dex", sub: &subscription.Subscription{Channel: wsChannelAllMids, Params: map[string]any{"dex": 1}}, err: errWebsocketParameter},
		{name: "unsupported interval", sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.TenMin}, err: kline.ErrUnsupportedInterval},
		{name: "unsupported orderbook levels", sub: testBookSubscription(10, nil), err: subscription.ErrInvalidLevel},
		{name: "invalid aggregateByTime", sub: &subscription.Subscription{Channel: wsChannelUserFills, Params: map[string]any{"aggregateByTime": "yes"}}, err: errWebsocketParameter},
		{name: "invalid ignorePortfolioMargin", sub: &subscription.Subscription{Channel: wsChannelSpotState, Params: map[string]any{"ignorePortfolioMargin": 1}}, err: errWebsocketParameter},
		{name: "ticker", sub: &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotPair}}, exp: WebsocketSubscription{Type: wsChannelActiveAssetContext}},
		{name: "candles", sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.OneMonth}, exp: WebsocketSubscription{Type: wsChannelCandle, Interval: "1M"}},
		{name: "fast aggregated orderbook", sub: testBookSubscription(5, map[string]any{"nSigFigs": 5.0, "mantissa": 2}), exp: WebsocketSubscription{Type: wsChannelOrderbook, NumberOfSignificantFigures: 5, Mantissa: 2, Fast: true}},
		{name: "builder DEX allMids", sub: &subscription.Subscription{Channel: wsChannelAllMids, Params: map[string]any{"dex": "xyz"}}, exp: WebsocketSubscription{Type: wsChannelAllMids, DEX: "xyz"}},
		{name: "aggregated userFills", sub: &subscription.Subscription{Channel: wsChannelUserFills, Params: map[string]any{"aggregateByTime": true, "dex": "xyz"}}, exp: WebsocketSubscription{Type: wsChannelUserFills, AggregateByTime: true}},
		{name: "spotState without portfolio margin", sub: &subscription.Subscription{Channel: wsChannelSpotState, Params: map[string]any{"ignorePortfolioMargin": true}}, exp: WebsocketSubscription{Type: wsChannelSpotState, IgnorePortfolioMargin: true}},
		{name: "trades with levels and an interval", sub: &subscription.Subscription{Channel: wsChannelTrades, Levels: 5, Interval: kline.OneMin}, exp: WebsocketSubscription{Type: wsChannelTrades}},
	} {
		options, err := websocketFeedOptions(tc.sub)
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "websocketFeedOptions should reject the %s subscription", tc.name)
			continue
		}
		require.NoErrorf(t, err, "websocketFeedOptions must not error for the %s subscription", tc.name)
		assert.Equalf(t, tc.exp, options, "websocketFeedOptions should return the options of the %s subscription", tc.name)
	}
}

// TestValidateSubscriptions validates through ExpandTemplates, which runs a ListValidator against the subscriptions it
// expands and the held ones
func TestValidateSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	builderBook := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{testBuilderPair}, Levels: 20}
	userFills := func(params map[string]any) *subscription.Subscription {
		return &subscription.Subscription{Channel: wsChannelUserFills, Authenticated: true, Params: params}
	}
	userEvents := &subscription.Subscription{Channel: wsSubscriptionUserEvents, Authenticated: true}
	for _, tc := range []struct {
		name string
		subs subscription.List
		err  error
	}{
		{name: "an l2Book of each depth for one market", subs: subscription.List{testBookSubscription(5, nil), testBookSubscription(20, nil)}, err: subscription.ErrExclusiveSubscription},
		{name: "l2Books of two significant figure settings for one market", subs: subscription.List{testBookSubscription(0, map[string]any{"nSigFigs": 5}), testBookSubscription(0, nil)}, err: subscription.ErrExclusiveSubscription},
		{name: "l2Books of two mantissas for one market", subs: subscription.List{testBookSubscription(0, map[string]any{"nSigFigs": 5, "mantissa": 2}), testBookSubscription(0, map[string]any{"nSigFigs": 5})}, err: subscription.ErrExclusiveSubscription},
		{name: "standard and Hyperliquid named l2Books of two depths", subs: subscription.List{testBookSubscription(5, nil), {Channel: wsChannelOrderbook, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}}}, err: subscription.ErrExclusiveSubscription},
		{name: "two userFills variants", subs: subscription.List{userFills(map[string]any{"aggregateByTime": true}), userFills(nil)}, err: subscription.ErrExclusiveSubscription},
		{name: "an aggregated userFills after userEvents", subs: subscription.List{userEvents, userFills(map[string]any{"aggregateByTime": true})}, err: subscription.ErrExclusiveSubscription},
		{name: "userEvents after an aggregated userFills", subs: subscription.List{userFills(map[string]any{"aggregateByTime": true}), userEvents}, err: subscription.ErrExclusiveSubscription},
		{name: "an aggregated userFills beside the standard account trades", subs: subscription.List{{Channel: subscription.MyTradesChannel, Authenticated: true}, userFills(map[string]any{"aggregateByTime": true})}, err: subscription.ErrExclusiveSubscription},
		{name: "userFills beside userEvents", subs: subscription.List{userEvents, userFills(nil), userEvents}},
		{name: "l2Books of two depths for two markets", subs: subscription.List{testBookSubscription(5, nil), builderBook}},
		{name: "l2Books of two depths for one pair of two asset types", subs: subscription.List{{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotPair}, Levels: 5}, {Channel: subscription.OrderbookChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{spotPair}, Levels: 20}}},
		{name: "one l2Book twice", subs: subscription.List{testBookSubscription(5, nil), testBookSubscription(5, nil)}},
		{name: "one userFills variant twice", subs: subscription.List{userFills(nil), userFills(map[string]any{"aggregateByTime": false})}},
		{name: "an invalid l2Book beside a valid one, which subscribing reports", subs: subscription.List{testBookSubscription(10, nil), testBookSubscription(5, nil)}},
		{name: "an authenticated l2Book beside another depth, which subscribing reports", subs: subscription.List{func() *subscription.Subscription {
			s := testBookSubscription(20, nil)
			s.Authenticated = true
			return s
		}(), testBookSubscription(5, nil)}},
	} {
		_, err := tc.subs.ExpandTemplates(ex)
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "ValidateSubscriptions should refuse %s", tc.name)
			continue
		}
		assert.NoErrorf(t, err, "ValidateSubscriptions should accept %s", tc.name)
	}

	require.NoError(t, subscribeWebsocketTest(t, ex, subscription.List{testBookSubscription(20, nil)}), "Subscribe must not error")
	assert.ErrorIs(t, subscribeWebsocketTest(t, ex, subscription.List{testBookSubscription(5, nil)}), subscription.ErrExclusiveSubscription, "ValidateSubscriptions should refuse a variant of a held l2Book")
	assert.NoError(t, ex.ValidateSubscriptions(nil), "ValidateSubscriptions should accept no subscriptions")
}

// TestWebsocketSubscribeRefusesExclusiveFeeds has the subscriber meet a feed exclusive of a held one, as a runtime
// subscription can once the manager has reconnected the account connection with its configured feeds first
func TestWebsocketSubscribeRefusesExclusiveFeeds(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	require.NoError(t, subscribeWebsocketTest(t, ex, subscription.List{{Channel: subscription.MyTradesChannel, Authenticated: true}}), "Subscribe must not error for the account trades")
	sent := len(accountConn(t, ex).sentRequests())
	aggregated, err := subscription.List{{Channel: wsChannelUserFills, Authenticated: true, Params: map[string]any{"aggregateByTime": true}}}.ExpandTemplates(websocketTemplateExpander{ex})
	require.NoError(t, err, "ExpandTemplates must not error")
	assert.ErrorIs(t, ex.websocketSubscribe(t.Context(), accountConn(t, ex), aggregated), subscription.ErrExclusiveSubscription, "websocketSubscribe should refuse a feed exclusive of a held one")
	assert.Len(t, accountConn(t, ex).sentRequests(), sent, "websocketSubscribe should send nothing for a refused feed")
	assert.Len(t, ex.Websocket.GetSubscriptions(), 1, "websocketSubscribe should leave only the account trades held")

	// A wrongly authenticated variant is refused when it is subscribed, so it must not refuse the feeds beside it
	require.NoError(t, subscribeWebsocketTest(t, ex, subscription.List{testBookSubscription(20, nil)}), "Subscribe must not error for the BTC book")
	wrong := testBookSubscription(5, nil)
	wrong.Authenticated = true
	batch, err := subscription.List{wrong, {Channel: wsChannelOrderUpdates, Authenticated: true}}.ExpandTemplates(websocketTemplateExpander{ex})
	require.NoError(t, err, "ExpandTemplates must not error")
	assert.ErrorIs(t, ex.websocketSubscribe(t.Context(), accountConn(t, ex), batch), subscription.ErrNotSupported, "websocketSubscribe should refuse a wrongly authenticated feed")
	assert.NotNil(t, ex.Websocket.GetSubscription(newWebsocketSubscriptionKey(batch[1])), "websocketSubscribe should subscribe the feeds beside a wrongly authenticated one")

	// An abandoned subscribe may yet be acknowledged, so its feed is treated as held until it is settled
	connection := publicConn(t, ex)
	connection.mu.Lock()
	connection.reply = nil
	connection.mu.Unlock()
	ex.WebsocketResponseMaxLimit = time.Millisecond
	builderBook := func(levels int) subscription.List {
		return subscription.List{{Channel: subscription.OrderbookChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{testBuilderPair}, Levels: levels}}
	}
	require.ErrorIs(t, subscribeWebsocketTest(t, ex, builderBook(20)), websocket.ErrSignatureTimeout, "Subscribe must time out without an acknowledgement")
	variant, err := builderBook(5).ExpandTemplates(websocketTemplateExpander{ex})
	require.NoError(t, err, "ExpandTemplates must not error")
	assert.ErrorIs(t, ex.websocketSubscribe(t.Context(), connection, variant), subscription.ErrExclusiveSubscription, "websocketSubscribe should refuse a variant of an abandoned subscribe's feed")
}

// TestSubscribeHoldsFeedVariantsApart subscribes a variant beside a feed it differs from only in a parameter selecting
// another feed, then unsubscribes the variant, through fixture connections that acknowledge each request
func TestSubscribeHoldsFeedVariantsApart(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		feed, variant *subscription.Subscription
		requests      []string
	}{
		{
			name:    "allMids dex",
			feed:    &subscription.Subscription{Channel: wsChannelAllMids},
			variant: &subscription.Subscription{Channel: wsChannelAllMids, Params: map[string]any{"dex": "xyz"}},
			requests: []string{
				`{"method":"subscribe","subscription":{"type":"allMids"}}`,
				`{"method":"subscribe","subscription":{"type":"allMids","dex":"xyz"}}`,
				`{"method":"unsubscribe","subscription":{"type":"allMids","dex":"xyz"}}`,
			},
		},
		{
			name:    "openOrders dex",
			feed:    &subscription.Subscription{Channel: wsChannelOpenOrders, Authenticated: true},
			variant: &subscription.Subscription{Channel: wsChannelOpenOrders, Authenticated: true, Params: map[string]any{"dex": "xyz"}},
			requests: []string{
				`{"method":"subscribe","subscription":{"type":"openOrders","user":"` + testAccountAddress + `"}}`,
				`{"method":"subscribe","subscription":{"type":"openOrders","user":"` + testAccountAddress + `","dex":"xyz"}}`,
				`{"method":"unsubscribe","subscription":{"type":"openOrders","user":"` + testAccountAddress + `","dex":"xyz"}}`,
			},
		},
		{
			name:    "clearinghouseState dex",
			feed:    &subscription.Subscription{Channel: wsChannelClearinghouseState, Authenticated: true},
			variant: &subscription.Subscription{Channel: wsChannelClearinghouseState, Authenticated: true, Params: map[string]any{"dex": "xyz"}},
			requests: []string{
				`{"method":"subscribe","subscription":{"type":"clearinghouseState","user":"` + testAccountAddress + `"}}`,
				`{"method":"subscribe","subscription":{"type":"clearinghouseState","user":"` + testAccountAddress + `","dex":"xyz"}}`,
				`{"method":"unsubscribe","subscription":{"type":"clearinghouseState","user":"` + testAccountAddress + `","dex":"xyz"}}`,
			},
		},
		{
			name:    "twapStates dex",
			feed:    &subscription.Subscription{Channel: wsChannelTWAPStates, Authenticated: true},
			variant: &subscription.Subscription{Channel: wsChannelTWAPStates, Authenticated: true, Params: map[string]any{"dex": "xyz"}},
			requests: []string{
				`{"method":"subscribe","subscription":{"type":"twapStates","user":"` + testAccountAddress + `"}}`,
				`{"method":"subscribe","subscription":{"type":"twapStates","user":"` + testAccountAddress + `","dex":"xyz"}}`,
				`{"method":"unsubscribe","subscription":{"type":"twapStates","user":"` + testAccountAddress + `","dex":"xyz"}}`,
			},
		},
		{
			name:    "spotState ignorePortfolioMargin",
			feed:    &subscription.Subscription{Channel: wsChannelSpotState, Authenticated: true},
			variant: &subscription.Subscription{Channel: wsChannelSpotState, Authenticated: true, Params: map[string]any{"ignorePortfolioMargin": true}},
			requests: []string{
				`{"method":"subscribe","subscription":{"type":"spotState","user":"` + testAccountAddress + `"}}`,
				`{"method":"subscribe","subscription":{"type":"spotState","user":"` + testAccountAddress + `","ignorePortfolioMargin":true}}`,
				`{"method":"unsubscribe","subscription":{"type":"spotState","user":"` + testAccountAddress + `","ignorePortfolioMargin":true}}`,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newWebsocketTestExchange(t)
			connection := publicConn(t, ex)
			if tc.feed.Authenticated {
				connection = accountConn(t, ex)
			}
			require.NoError(t, subscribeWebsocketTest(t, ex, subscription.List{tc.feed}), "Subscribe must not error for the feed")
			require.NoError(t, subscribeWebsocketTest(t, ex, subscription.List{tc.variant}), "Subscribe must hold the variant beside the feed")
			assert.ErrorIs(t, subscribeWebsocketTest(t, ex, subscription.List{tc.variant}), subscription.ErrDuplicate, "Subscribe should refuse the held variant again")
			require.NoError(t, unsubscribeWebsocketTest(t, ex, subscription.List{tc.variant}), "Unsubscribe must not error for the variant")
			assert.NotNil(t, ex.Websocket.GetSubscription(newWebsocketSubscriptionKey(tc.feed)), "Unsubscribe should leave the feed held")
			assert.Nil(t, ex.Websocket.GetSubscription(newWebsocketSubscriptionKey(tc.variant)), "Unsubscribe should remove the variant")
			assert.ErrorIs(t, unsubscribeWebsocketTest(t, ex, subscription.List{tc.variant}), subscription.ErrNotFound, "Unsubscribe should report a variant that is not held")
			assert.NotNil(t, ex.Websocket.GetSubscription(newWebsocketSubscriptionKey(tc.feed)), "Unsubscribing a variant that is not held should leave the feed held")
			assert.Equal(t, tc.requests, connection.sentRequests(), "Subscribe and Unsubscribe should send the variant's own requests")
		})
	}

	// A second variant of these feeds is refused as exclusive, and unsubscribing it must still leave the feed held
	for _, tc := range []struct {
		name          string
		feed, variant *subscription.Subscription
		request       string
	}{
		{name: "l2Book nSigFigs", feed: testBookSubscription(0, nil), variant: testBookSubscription(0, map[string]any{"nSigFigs": 5}), request: `{"method":"subscribe","subscription":{"type":"l2Book","coin":"BTC"}}`},
		{name: "l2Book mantissa", feed: testBookSubscription(0, map[string]any{"nSigFigs": 5}), variant: testBookSubscription(0, map[string]any{"nSigFigs": 5, "mantissa": 2}), request: `{"method":"subscribe","subscription":{"type":"l2Book","coin":"BTC","nSigFigs":5}}`},
		{name: "l2Book fast", feed: testBookSubscription(20, nil), variant: testBookSubscription(5, nil), request: `{"method":"subscribe","subscription":{"type":"l2Book","coin":"BTC"}}`},
		{name: "userFills aggregateByTime", feed: &subscription.Subscription{Channel: wsChannelUserFills, Authenticated: true}, variant: &subscription.Subscription{Channel: wsChannelUserFills, Authenticated: true, Params: map[string]any{"aggregateByTime": true}}, request: `{"method":"subscribe","subscription":{"type":"userFills","user":"` + testAccountAddress + `"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newWebsocketTestExchange(t)
			connection := publicConn(t, ex)
			if tc.feed.Authenticated {
				connection = accountConn(t, ex)
			}
			require.NoError(t, subscribeWebsocketTest(t, ex, subscription.List{tc.feed}), "Subscribe must not error for the feed")
			assert.ErrorIs(t, subscribeWebsocketTest(t, ex, subscription.List{tc.variant}), subscription.ErrExclusiveSubscription, "Subscribe should refuse a second variant of an exclusive feed")
			assert.ErrorIs(t, unsubscribeWebsocketTest(t, ex, subscription.List{tc.variant}), subscription.ErrNotFound, "Unsubscribe should report a variant that is not held")
			assert.NotNil(t, ex.Websocket.GetSubscription(newWebsocketSubscriptionKey(tc.feed)), "Unsubscribing a variant that is not held should leave the feed held")
			assert.Equal(t, []string{tc.request}, connection.sentRequests(), "Subscribe and Unsubscribe should send nothing for the variant")
		})
	}
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, e.SubscribeToWebsocketChannels(subscription.List{nil}), common.ErrNilPointer, "SubscribeToWebsocketChannels should reject a nil subscription")
	assert.ErrorIs(t, e.SubscribeToWebsocketChannels(subscription.List{{Channel: "bogus"}}), subscription.ErrNotSupported, "SubscribeToWebsocketChannels should report a subscription it cannot expand")

	ex, venue := newVenueWebsocketTestExchange(t)
	subs := subscription.List{
		{Channel: subscription.OrderbookChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}, Levels: 5, Params: map[string]any{"nSigFigs": 5, "mantissa": 2}},
		{Channel: wsChannelAllMids, Params: map[string]any{"dex": "xyz"}},
		{Channel: "bogus"},
	}
	assert.ErrorIs(t, ex.SubscribeToWebsocketChannels(subs), subscription.ErrNotSupported, "SubscribeToWebsocketChannels should report a subscription it cannot expand and subscribe the rest")
	subscribed := subs[:2]
	t.Cleanup(func() {
		err := websocketCallWithin(t, "UnsubscribeToWebsocketChannels", func() error { return ex.UnsubscribeToWebsocketChannels(subscribed) })
		assert.NoError(t, err, "UnsubscribeToWebsocketChannels should remove the test's subscriptions")
	})
	for _, sub := range subscribed {
		stored := ex.Websocket.GetSubscription(sub)
		require.NotNilf(t, stored, "SubscribeToWebsocketChannels must store %s", sub)
		assert.Equalf(t, subscription.SubscribedState, stored.State(), "SubscribeToWebsocketChannels should mark %s subscribed", sub)
	}
	if mockTests {
		assert.Equal(t, []string{
			`{"method":"subscribe","subscription":{"type":"l2Book","coin":"BTC","nSigFigs":5,"mantissa":2,"fast":true}}`,
			`{"method":"subscribe","subscription":{"type":"allMids","dex":"xyz"}}`,
		}, venue.received(), "SubscribeToWebsocketChannels should send each subscription request")
	}

	t.Run("account feed", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		}
		accountSubs := subscription.List{{Channel: wsChannelUserFills, Authenticated: true, Params: map[string]any{"aggregateByTime": true}}}
		require.NoError(t, ex.SubscribeToWebsocketChannels(accountSubs), "SubscribeToWebsocketChannels must not error for an account feed")
		t.Cleanup(func() {
			// Removing the account connection's only feed retires the connection, which waits for its reader
			err := websocketCallWithin(t, "UnsubscribeToWebsocketChannels", func() error { return ex.UnsubscribeToWebsocketChannels(accountSubs) })
			assert.NoError(t, err, "UnsubscribeToWebsocketChannels should remove the account feed")
		})
		stored := ex.Websocket.GetSubscription(accountSubs[0])
		require.NotNil(t, stored, "SubscribeToWebsocketChannels must store the account feed")
		assert.Equal(t, subscription.SubscribedState, stored.State(), "SubscribeToWebsocketChannels should mark the account feed subscribed")
		if mockTests {
			received := venue.received()
			assert.JSONEq(t, `{"method":"subscribe","subscription":{"type":"userFills","user":"`+testAccountAddress+`","aggregateByTime":true}}`, received[len(received)-1], "SubscribeToWebsocketChannels should send the account feed's subscription request")
		}
	})
}

func TestUnsubscribe(t *testing.T) {
	t.Parallel()
	_, validates := any(e).(subscription.ListValidator)
	assert.True(t, validates, "the exchange should validate the subscriptions it expands")
	_, validates = any(websocketTemplateExpander{e}).(subscription.ListValidator)
	assert.False(t, validates, "UnsubscribeToWebsocketChannels should expand the subscriptions to remove without validating them")
	assert.ErrorIs(t, e.UnsubscribeToWebsocketChannels(subscription.List{nil}), common.ErrNilPointer, "UnsubscribeToWebsocketChannels should reject a nil subscription")
	sub, _ := testOrderbookSubscription()
	assert.ErrorIs(t, e.UnsubscribeToWebsocketChannels(subscription.List{sub}), subscription.ErrNotFound, "UnsubscribeToWebsocketChannels should reject a subscription it does not hold")

	ex, venue := newVenueWebsocketTestExchange(t)
	require.NoError(t, ex.SubscribeToWebsocketChannels(subscription.List{sub}), "SubscribeToWebsocketChannels must not error")
	require.NoError(t, ex.UnsubscribeToWebsocketChannels(subscription.List{sub}), "UnsubscribeToWebsocketChannels must not error")
	assert.Nil(t, ex.Websocket.GetSubscription(sub), "UnsubscribeToWebsocketChannels should remove the subscription")
	if mockTests {
		assert.Equal(t, []string{
			`{"method":"subscribe","subscription":{"type":"l2Book","coin":"BTC"}}`,
			`{"method":"unsubscribe","subscription":{"type":"l2Book","coin":"BTC"}}`,
		}, venue.received(), "UnsubscribeToWebsocketChannels should send the unsubscribe request")
	}
}

// delayedSubscriptionKey exposes a stored subscription after the initial duplicate check, modelling another caller
// registering the same feed before AddSubscriptions without scheduling a concurrent mutation.
type delayedSubscriptionKey struct {
	subscription.ExactKey
	lookups int
}

func (k *delayedSubscriptionKey) GetSubscription() *subscription.Subscription {
	k.lookups++
	if k.lookups == 1 {
		return nil
	}
	return k.Subscription
}

func TestGetSubscription(t *testing.T) {
	t.Parallel()
	sub := &subscription.Subscription{Channel: wsChannelAllMids}
	key := new(delayedSubscriptionKey)
	key.Subscription = sub
	assert.Nil(t, key.GetSubscription(), "GetSubscription should hide the first lookup")
	assert.Same(t, sub, key.GetSubscription(), "GetSubscription should expose the second lookup")
	assert.Same(t, sub, key.GetSubscription(), "GetSubscription should retain subsequent lookups")
}

// TestManageWebsocketSubscription injects failed sends, missing acknowledgements and colliding operations, which the
// venue cannot produce, through fixture connections; TestSubscribe and TestUnsubscribe cover the venue round trips
func TestManageWebsocketSubscription(t *testing.T) {
	t.Parallel()
	t.Run("subscription limit", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		held := make(subscription.List, maximumWebsocketSubscriptions)
		for i := range held {
			held[i] = &subscription.Subscription{Channel: wsChannelAllMids, QualifiedChannel: wsChannelAllMids, Params: map[string]any{"dex": "d" + strconv.Itoa(i)}}
		}
		keyWebsocketSubscriptions(held)
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(publicConn(t, ex), held...), "AddSuccessfulSubscriptions must not error")
		sub := &subscription.Subscription{Channel: wsChannelAllMids, QualifiedChannel: wsChannelAllMids}
		assert.ErrorIs(t, ex.manageWebsocketSubscription(t.Context(), publicConn(t, ex), wsMethodSubscribe, sub), errWebsocketSubscriptionsLimit, "manageWebsocketSubscription should refuse a subscription past Hyperliquid's limit")
		assert.Empty(t, publicConn(t, ex).sentRequests(), "manageWebsocketSubscription should send nothing past Hyperliquid's limit")
	})
	t.Run("room kept for account feeds", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		_, err := ex.generatePublicSubscriptions()
		require.NoError(t, err, "generatePublicSubscriptions must not error")
		// The default account feeds keep two slots, so these market feeds leave room for one more
		held := make(subscription.List, maximumWebsocketSubscriptions-3)
		for i := range held {
			held[i] = &subscription.Subscription{Channel: wsChannelAllMids, QualifiedChannel: wsChannelAllMids, Params: map[string]any{"dex": "d" + strconv.Itoa(i)}}
		}
		keyWebsocketSubscriptions(held)
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(publicConn(t, ex), held...), "AddSuccessfulSubscriptions must not error")
		last := &subscription.Subscription{Channel: wsChannelAllMids, QualifiedChannel: wsChannelAllMids}
		require.NoError(t, ex.manageWebsocketSubscription(t.Context(), publicConn(t, ex), wsMethodSubscribe, last), "manageWebsocketSubscription must subscribe a market feed into the room left beside the account feeds")
		market := &subscription.Subscription{Channel: wsChannelAllMids, QualifiedChannel: wsChannelAllMids, Params: map[string]any{"dex": "xyz"}}
		assert.ErrorIs(t, ex.manageWebsocketSubscription(t.Context(), publicConn(t, ex), wsMethodSubscribe, market), errWebsocketSubscriptionsLimit, "manageWebsocketSubscription should keep the room of the account feeds not held")
		assert.Len(t, publicConn(t, ex).sentRequests(), 1, "manageWebsocketSubscription should send no market feed into the account feeds' room")
		for _, channel := range []string{subscription.MyOrdersChannel, subscription.MyTradesChannel} {
			sub := &subscription.Subscription{Channel: channel, Authenticated: true, QualifiedChannel: channel}
			require.NoErrorf(t, ex.manageWebsocketSubscription(t.Context(), accountConn(t, ex), wsMethodSubscribe, sub), "manageWebsocketSubscription must subscribe the account feed %s into its room", channel)
		}
		notification := &subscription.Subscription{Channel: wsChannelNotification, Authenticated: true, QualifiedChannel: wsChannelNotification}
		assert.ErrorIs(t, ex.manageWebsocketSubscription(t.Context(), accountConn(t, ex), wsMethodSubscribe, notification), errWebsocketSubscriptionsLimit, "manageWebsocketSubscription should refuse an account feed past Hyperliquid's limit")
		assert.Len(t, accountConn(t, ex).sentRequests(), 2, "manageWebsocketSubscription should send no account feed past Hyperliquid's limit")
	})
	t.Run("subscription added after duplicate check", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		stored := &subscription.Subscription{Channel: wsChannelAllMids, QualifiedChannel: wsChannelAllMids}
		key := new(delayedSubscriptionKey)
		key.Subscription = stored
		stored.SetKey(key)
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(publicConn(t, ex), stored), "AddSuccessfulSubscriptions must store the competing subscription")
		sub := &subscription.Subscription{Channel: wsChannelAllMids, QualifiedChannel: wsChannelAllMids}
		err := ex.manageWebsocketSubscription(t.Context(), websocketTestConn(t, ex, sub.Authenticated), wsMethodSubscribe, sub)
		assert.ErrorIs(t, err, subscription.ErrDuplicate, "manageWebsocketSubscription should return AddSubscriptions errors")
		assert.Equal(t, 2, key.lookups, "manageWebsocketSubscription should reach the second duplicate check")
		assert.Empty(t, publicConn(t, ex).sentRequests(), "manageWebsocketSubscription should send no failed registration")
		assert.Nil(t, pendingWebsocketOperation(ex, &websocketPendingKey{subscription: WebsocketSubscription{Type: wsChannelAllMids}}), "manageWebsocketSubscription should register no pending operation")
	})

	ex := newWebsocketTestExchange(t)
	connection := publicConn(t, ex)
	assert.ErrorIs(t, ex.manageWebsocketSubscription(t.Context(), publicConn(t, ex), wsMethodSubscribe, nil), common.ErrNilPointer, "manageWebsocketSubscription should reject a nil subscription")
	sub, _ := testOrderbookSubscription()
	assert.ErrorIs(t, ex.manageWebsocketSubscription(t.Context(), websocketTestConn(t, ex, sub.Authenticated), "post", sub), common.ErrNotYetImplemented, "manageWebsocketSubscription should reject another method")
	assert.ErrorIs(t, ex.manageWebsocketSubscription(t.Context(), publicConn(t, ex), wsMethodSubscribe, &subscription.Subscription{Channel: "bogus"}), subscription.ErrNotSupported, "manageWebsocketSubscription should reject an unsupported channel")

	assert.ErrorIs(t, ex.manageWebsocketSubscription(t.Context(), nil, wsMethodSubscribe, sub), common.ErrNilPointer, "manageWebsocketSubscription should require a connection")

	require.NoError(t, ex.manageWebsocketSubscription(t.Context(), websocketTestConn(t, ex, sub.Authenticated), wsMethodSubscribe, sub), "manageWebsocketSubscription must not error")
	assert.Equal(t, subscription.SubscribedState, sub.State(), "manageWebsocketSubscription should mark an acknowledged subscribe subscribed")
	assert.NotNil(t, ex.Websocket.GetSubscription(sub.Clone()), "manageWebsocketSubscription should store the subscription")
	assert.ErrorIs(t, ex.manageWebsocketSubscription(t.Context(), websocketTestConn(t, ex, sub.Authenticated), wsMethodSubscribe, sub), subscription.ErrDuplicate, "manageWebsocketSubscription should reject a duplicate subscribe")
	require.NoError(t, ex.manageWebsocketSubscription(t.Context(), websocketTestConn(t, ex, sub.Authenticated), wsMethodUnsubscribe, sub), "manageWebsocketSubscription must not error for an unsubscribe")
	assert.Nil(t, ex.Websocket.GetSubscription(sub), "manageWebsocketSubscription should remove an acknowledged unsubscribe")
	assert.Equal(t, []string{
		`{"method":"subscribe","subscription":{"type":"l2Book","coin":"BTC"}}`,
		`{"method":"unsubscribe","subscription":{"type":"l2Book","coin":"BTC"}}`,
	}, connection.sentRequests(), "manageWebsocketSubscription should send each request")
	for _, err := range connection.handled() {
		assert.NoError(t, err, "the acknowledgements should be handled without error")
	}

	trades, _ := testTradesSubscription()
	require.NoError(t, ex.manageWebsocketSubscription(t.Context(), websocketTestConn(t, ex, trades.Authenticated), wsMethodSubscribe, trades), "manageWebsocketSubscription must not error for trades")
	assert.True(t, ex.consumeTradeReplay("BTC"), "an acknowledged trades subscribe should await the trades it replays")
	ex.expectTradeReplay("BTC")
	watermarks := map[string]websocketPosition{"BTC": {time: time.UnixMilli(1791278878126), tradeID: 1}}
	ex.websocketWatermarks.trades = maps.Clone(watermarks)
	require.NoError(t, ex.manageWebsocketSubscription(t.Context(), websocketTestConn(t, ex, trades.Authenticated), wsMethodUnsubscribe, trades), "manageWebsocketSubscription must not error for a trades unsubscribe")
	assert.False(t, ex.consumeTradeReplay("BTC"), "an acknowledged trades unsubscribe should stop awaiting a replay")
	assert.Empty(t, ex.websocketWatermarks.trades, "manageWebsocketSubscription should forget an unsubscribed feed's watermark")

	connection.sendErr = errWebsocketFixture
	require.NoError(t, ex.manageWebsocketSubscription(t.Context(), websocketTestConn(t, ex, trades.Authenticated), wsMethodSubscribe, trades), "manageWebsocketSubscription must keep a subscribe acknowledged before its send failed")
	assert.Equal(t, subscription.SubscribedState, trades.State(), "manageWebsocketSubscription should keep an acknowledged subscribe")
	connection.reply = nil
	ex.websocketWatermarks.trades = maps.Clone(watermarks)
	err := ex.manageWebsocketSubscription(t.Context(), websocketTestConn(t, ex, trades.Authenticated), wsMethodUnsubscribe, trades)
	assert.Equal(t, watermarks, ex.websocketWatermarks.trades, "manageWebsocketSubscription should keep the watermark of a feed it failed to unsubscribe")
	assert.ErrorIs(t, err, errWebsocketFixture, "manageWebsocketSubscription should return a send error")
	assert.ErrorContains(t, err, "sending unsubscribe request", "manageWebsocketSubscription should name the request it could not send")
	assert.Equal(t, subscription.SubscribedState, trades.State(), "manageWebsocketSubscription should restore the state of an unsubscribe that was not sent")
	connection.sendErr = nil
	connection.reply = acknowledge
	require.NoError(t, ex.manageWebsocketSubscription(t.Context(), websocketTestConn(t, ex, trades.Authenticated), wsMethodUnsubscribe, trades), "manageWebsocketSubscription must not error when retrying an unsubscribe")

	connection.sendErr = errWebsocketFixture
	connection.reply = nil
	assert.ErrorIs(t, ex.manageWebsocketSubscription(t.Context(), websocketTestConn(t, ex, trades.Authenticated), wsMethodSubscribe, trades), errWebsocketFixture, "manageWebsocketSubscription should return a send error")
	assert.Nil(t, ex.Websocket.GetSubscription(trades), "manageWebsocketSubscription should roll back a subscribe that was not sent")
	assert.False(t, ex.consumeTradeReplay("BTC"), "a trades subscribe without an acknowledgement should await no replay")

	removed, _ := testTradesSubscription()
	connection.reply = func([]byte) []string {
		assert.NoError(t, ex.Websocket.RemoveSubscriptions(connection, removed), "RemoveSubscriptions should not error")
		return nil
	}
	err = ex.manageWebsocketSubscription(t.Context(), websocketTestConn(t, ex, removed.Authenticated), wsMethodSubscribe, removed)
	assert.ErrorIs(t, err, errWebsocketFixture, "manageWebsocketSubscription should return a send error")
	assert.ErrorIs(t, err, subscription.ErrNotFound, "manageWebsocketSubscription should return a rollback error")
	connection.sendErr = nil
	connection.reply = acknowledge

	unsubscribing := &subscription.Subscription{Channel: wsChannelOutcomeMetaUpdates, QualifiedChannel: wsChannelOutcomeMetaUpdates}
	require.NoError(t, ex.manageWebsocketSubscription(t.Context(), websocketTestConn(t, ex, unsubscribing.Authenticated), wsMethodSubscribe, unsubscribing), "manageWebsocketSubscription must not error for outcomeMetaUpdates")
	require.NoError(t, unsubscribing.SetState(subscription.UnsubscribingState), "SetState must not error")
	assert.ErrorIs(t, ex.manageWebsocketSubscription(t.Context(), websocketTestConn(t, ex, unsubscribing.Authenticated), wsMethodUnsubscribe, unsubscribing), subscription.ErrInStateAlready, "manageWebsocketSubscription should reject an unsubscribe already in progress")
	unrecorded := newSubscribedWebsocketSubscription(t, ex, &subscription.Subscription{Channel: wsChannelFastAssetContexts, QualifiedChannel: wsChannelFastAssetContexts})
	assert.ErrorIs(t, ex.manageWebsocketSubscription(t.Context(), websocketTestConn(t, ex, unrecorded.Authenticated), wsMethodUnsubscribe, unrecorded), errWebsocketFeedUnknown, "manageWebsocketSubscription should reject an unsubscribe of a subscription it did not subscribe")

	// A market that has left the metadata, as a delisting removes it, is still unsubscribed with the feed it subscribed
	ethPair := currency.NewPair(currency.ETH, currency.USDC)
	ex.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping, testBuilderMapping, {pair: ethPair, coin: "ETH", sizeDecimals: 4, maxLeverage: 25}})
	delisted := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{ethPair}, QualifiedChannel: "l2Book:perpetualcontract:ETH-USDC"}
	require.NoError(t, ex.manageWebsocketSubscription(t.Context(), websocketTestConn(t, ex, delisted.Authenticated), wsMethodSubscribe, delisted), "manageWebsocketSubscription must not error for the ETH book")
	ex.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping, testBuilderMapping})
	require.NoError(t, ex.manageWebsocketSubscription(t.Context(), websocketTestConn(t, ex, delisted.Authenticated), wsMethodUnsubscribe, delisted), "manageWebsocketSubscription must unsubscribe a market that has left the metadata")
	assert.Nil(t, ex.Websocket.GetSubscription(delisted), "manageWebsocketSubscription should remove the subscription of a market that has left the metadata")
	sent := connection.sentRequests()
	assert.JSONEq(t, `{"method":"unsubscribe","subscription":{"type":"l2Book","coin":"ETH"}}`, sent[len(sent)-1], "manageWebsocketSubscription should unsubscribe the feed the market subscribed")

	collisionKey := websocketPendingKey{connection: publicConn(t, ex), subscription: WebsocketSubscription{Type: wsChannelAllMids}}
	collision := &websocketPendingOperation{method: wsMethodUnsubscribe, done: make(chan error, 1)}
	registerWebsocketPending(ex, &collisionKey, collision)
	allMids := &subscription.Subscription{Channel: wsChannelAllMids, QualifiedChannel: wsChannelAllMids}
	assert.ErrorIs(t, ex.manageWebsocketSubscription(t.Context(), websocketTestConn(t, ex, allMids.Authenticated), wsMethodSubscribe, allMids), errWebsocketSubscription, "manageWebsocketSubscription should reject an operation already pending for the feed")
	assert.Nil(t, ex.Websocket.GetSubscription(allMids), "manageWebsocketSubscription should roll back an operation it rejects")
	assert.Same(t, collision, pendingWebsocketOperation(ex, &collisionKey), "manageWebsocketSubscription should keep the pending operation")

	ex.websocketPending = nil
	authenticated := &subscription.Subscription{Channel: subscription.MyOrdersChannel, Authenticated: true, QualifiedChannel: wsChannelOrderUpdates}
	require.NoError(t, ex.manageWebsocketSubscription(t.Context(), websocketTestConn(t, ex, authenticated.Authenticated), wsMethodSubscribe, authenticated), "manageWebsocketSubscription must not error for an account feed")
	assert.Equal(t, []string{
		`{"method":"subscribe","subscription":{"type":"orderUpdates","user":"` + testAccountAddress + `"}}`,
	}, accountConn(t, ex).sentRequests(), "manageWebsocketSubscription should send account feeds on the account connection")

	// An account feed is unsubscribed for the address it subscribed, after the configured address changes, and
	// subscribing it again subscribes the new address
	userFills := &subscription.Subscription{Channel: wsChannelUserFills, Authenticated: true, QualifiedChannel: wsChannelUserFills}
	require.NoError(t, ex.manageWebsocketSubscription(t.Context(), websocketTestConn(t, ex, userFills.Authenticated), wsMethodSubscribe, userFills), "manageWebsocketSubscription must not error for userFills")
	setTestCredentials(ex, &accounts.Credentials{Key: testOtherAddress})
	require.NoError(t, ex.manageWebsocketSubscription(t.Context(), websocketTestConn(t, ex, userFills.Authenticated), wsMethodUnsubscribe, userFills), "manageWebsocketSubscription must unsubscribe userFills after the address changes")
	require.NoError(t, ex.manageWebsocketSubscription(t.Context(), websocketTestConn(t, ex, userFills.Authenticated), wsMethodSubscribe, userFills), "manageWebsocketSubscription must subscribe userFills again")
	require.NoError(t, ex.manageWebsocketSubscription(t.Context(), websocketTestConn(t, ex, userFills.Authenticated), wsMethodUnsubscribe, userFills), "manageWebsocketSubscription must unsubscribe userFills again")
	assert.Equal(t, []string{
		`{"method":"subscribe","subscription":{"type":"orderUpdates","user":"` + testAccountAddress + `"}}`,
		`{"method":"subscribe","subscription":{"type":"userFills","user":"` + testAccountAddress + `"}}`,
		`{"method":"unsubscribe","subscription":{"type":"userFills","user":"` + testAccountAddress + `"}}`,
		`{"method":"subscribe","subscription":{"type":"userFills","user":"` + testOtherAddress + `"}}`,
		`{"method":"unsubscribe","subscription":{"type":"userFills","user":"` + testOtherAddress + `"}}`,
	}, accountConn(t, ex).sentRequests(), "manageWebsocketSubscription should unsubscribe each address it subscribed")
	assert.ErrorIs(t, ex.manageWebsocketSubscription(t.Context(), nil, wsMethodUnsubscribe, authenticated), common.ErrNilPointer, "manageWebsocketSubscription should require the account connection")
	assert.Equal(t, subscription.SubscribedState, authenticated.State(), "manageWebsocketSubscription should not change a subscription it cannot send")
}

// TestAwaitWebsocketPending injects settled operations, cancelled contexts and missing acknowledgements, which the
// venue cannot produce, through a fixture connection
func TestAwaitWebsocketPending(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	connection := publicConn(t, ex)
	connection.reply = nil
	subscribeRequest := &WebsocketSubscriptionRequest{Method: wsMethodSubscribe, Subscription: WebsocketSubscription{Type: wsChannelOrderbook, Coin: "BTC"}}

	sub, key := testOrderbookSubscription()
	pending := addWebsocketPending(t, ex, wsMethodSubscribe, sub, &key)
	pending.done <- errWebsocketServer
	assert.ErrorIs(t, ex.awaitWebsocketPending(t.Context(), connection, &key, pending, subscribeRequest), errWebsocketServer, "awaitWebsocketPending should return the server's outcome")

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	assert.ErrorIs(t, ex.awaitWebsocketPending(ctx, connection, &key, pending, subscribeRequest), context.Canceled, "awaitWebsocketPending should stop when its context ends")
	assert.Nil(t, ex.Websocket.GetSubscription(sub), "awaitWebsocketPending should roll back an operation whose context ended")

	sub, key = testOrderbookSubscription()
	pending = addWebsocketPending(t, ex, wsMethodSubscribe, sub, &key)
	ex.WebsocketResponseMaxLimit = time.Millisecond
	assert.ErrorIs(t, ex.awaitWebsocketPending(t.Context(), connection, &key, pending, subscribeRequest), websocket.ErrSignatureTimeout, "awaitWebsocketPending should time out without an acknowledgement")
	assert.Nil(t, ex.Websocket.GetSubscription(sub), "awaitWebsocketPending should roll back an operation that timed out")
	assert.Same(t, pending, pendingWebsocketOperation(ex, &key), "awaitWebsocketPending should keep a subscribe that timed out pending, as Hyperliquid may still answer it")
	ex.websocketPendingMu.Lock()
	abandoned := pending.abandoned
	ex.websocketPendingMu.Unlock()
	assert.True(t, abandoned, "awaitWebsocketPending should mark a subscribe that timed out abandoned")

	held, heldKey := testTradesSubscription()
	held = newSubscribedWebsocketSubscription(t, ex, held)
	unsubscribing := addWebsocketPending(t, ex, wsMethodUnsubscribe, held, &heldKey)
	unsubscribeRequest := &WebsocketSubscriptionRequest{Method: wsMethodUnsubscribe, Subscription: heldKey.subscription}
	assert.ErrorIs(t, ex.awaitWebsocketPending(t.Context(), connection, &heldKey, unsubscribing, unsubscribeRequest), websocket.ErrSignatureTimeout, "awaitWebsocketPending should time out an unsubscribe without an acknowledgement")
	assert.Nil(t, pendingWebsocketOperation(ex, &heldKey), "awaitWebsocketPending should remove an unsubscribe that timed out")
	assert.Equal(t, subscription.SubscribedState, held.State(), "awaitWebsocketPending should restore the state of an unsubscribe that timed out")
}

// TestAbandonedWebsocketSubscribe has Hyperliquid answer a subscribe after its caller gave up on it
func TestAbandonedWebsocketSubscribe(t *testing.T) {
	t.Parallel()
	trades := func() subscription.List {
		return subscription.List{{Channel: subscription.AllTradesChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}}}
	}
	feed := `{"type":"trades","coin":"BTC"}`
	acknowledgement := func(method string) []byte {
		return []byte(`{"channel":"subscriptionResponse","data":{"method":"` + method + `","subscription":` + feed + `}}`)
	}
	// abandon times out a BTC trades subscribe on an exchange already holding held market feeds
	abandon := func(t *testing.T, held int) (*Exchange, *websocketConnectionFixture, *websocketPendingKey) {
		t.Helper()
		ex := newWebsocketTestExchange(t)
		conn := publicConn(t, ex)
		if held != 0 {
			subs := make(subscription.List, held)
			for i := range subs {
				subs[i] = &subscription.Subscription{Channel: wsChannelAllMids, QualifiedChannel: wsChannelAllMids, Params: map[string]any{"dex": "d" + strconv.Itoa(i)}}
			}
			keyWebsocketSubscriptions(subs)
			require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, subs...), "AddSuccessfulSubscriptions must not error")
		}
		conn.mu.Lock()
		conn.reply = nil
		conn.mu.Unlock()
		ex.WebsocketResponseMaxLimit = time.Millisecond
		require.ErrorIs(t, subscribeWebsocketTest(t, ex, trades()), websocket.ErrSignatureTimeout, "Subscribe must time out without an acknowledgement")
		require.Len(t, ex.Websocket.GetSubscriptions(), held, "Subscribe must roll back the subscription it gave up on")
		key := &websocketPendingKey{connection: conn, subscription: WebsocketSubscription{Type: wsChannelTrades, Coin: "BTC"}}
		require.NotNil(t, pendingWebsocketOperation(ex, key), "Subscribe must keep the abandoned subscribe pending")
		conn.mu.Lock()
		conn.reply = acknowledge
		conn.mu.Unlock()
		return ex, conn, key
	}
	t.Run("acknowledged late", func(t *testing.T) {
		t.Parallel()
		ex, conn, key := abandon(t, 0)
		abandoned := pendingWebsocketOperation(ex, key)
		assert.ErrorIs(t, subscribeWebsocketTest(t, ex, trades()), errWebsocketSubscription, "Subscribe should refuse the feed while its abandoned subscribe awaits an answer")
		sent := len(conn.sentRequests())
		require.NoError(t, ex.wsHandleData(t.Context(), conn, acknowledgement(wsMethodSubscribe)), "wsHandleData must not error for the late acknowledgement")
		assert.Nil(t, pendingWebsocketOperation(ex, key), "the late acknowledgement should settle the abandoned subscribe")
		assert.Len(t, conn.sentRequests(), sent, "the late acknowledgement should leave the feed subscribed")
		assert.Empty(t, ex.Websocket.GetSubscriptions(), "the abandoned subscribe should stay rolled back")
		assert.Equal(t, subscription.UnsubscribedState, abandoned.subscription.State(), "the abandoned subscription should stay unsubscribed")
		assert.Empty(t, abandoned.done, "the abandoned operation should not send another result")
		// Hyperliquid answers a subscribe of the feed the late acknowledgement left active as already subscribed
		alreadySubscribed := string(websocketErrorFrame(t, wsErrorAlreadySubscribed+feed))
		conn.mu.Lock()
		conn.reply = func([]byte) []string { return []string{alreadySubscribed} }
		conn.mu.Unlock()
		require.NoError(t, subscribeWebsocketTest(t, ex, trades()), "Subscribe must take over the feed the late acknowledgement left active")
		assert.Len(t, ex.Websocket.GetSubscriptions(), 1, "Subscribe should hold the feed it took over")
		require.NoError(t, ex.wsHandleData(t.Context(), conn, acknowledgement(wsMethodUnsubscribe)), "a stale unsubscribe acknowledgement must be ignored")
		assert.Len(t, ex.Websocket.GetSubscriptions(), 1, "a stale unsubscribe acknowledgement should leave the later subscription held")
	})
	t.Run("already subscribed", func(t *testing.T) {
		t.Parallel()
		ex, conn, key := abandon(t, 0)
		abandoned := pendingWebsocketOperation(ex, key)
		sent := len(conn.sentRequests())
		err := ex.wsHandleData(t.Context(), conn, websocketErrorFrame(t, wsErrorAlreadySubscribed+feed))
		require.NoError(t, err, "wsHandleData must not error for a late already subscribed answer")
		assert.Nil(t, pendingWebsocketOperation(ex, key), "an already subscribed answer should settle the abandoned subscribe")
		assert.Len(t, conn.sentRequests(), sent, "an already subscribed answer should leave the feed subscribed")
		assert.Empty(t, ex.Websocket.GetSubscriptions(), "the abandoned subscribe should stay rolled back")
		assert.Equal(t, subscription.UnsubscribedState, abandoned.subscription.State(), "the abandoned subscription should stay unsubscribed")
		assert.Empty(t, abandoned.done, "the abandoned operation should not send another result")
	})
	t.Run("refused late", func(t *testing.T) {
		t.Parallel()
		ex, conn, key := abandon(t, 0)
		abandoned := pendingWebsocketOperation(ex, key)
		sent := len(conn.sentRequests())
		err := ex.wsHandleData(t.Context(), conn, websocketErrorFrame(t, wsErrorInvalidSubscription+feed))
		assert.ErrorIs(t, err, errWebsocketServer, "wsHandleData should report the late refusal")
		assert.Nil(t, pendingWebsocketOperation(ex, key), "a late refusal should free the feed's slot")
		assert.Empty(t, abandoned.done, "the abandoned operation should not send another result")
		assert.Len(t, conn.sentRequests(), sent, "a late refusal should send nothing")
	})
	t.Run("refused late while subscribed again", func(t *testing.T) {
		t.Parallel()
		ex, conn, key := abandon(t, 0)
		abandoned := pendingWebsocketOperation(ex, key)
		// A subscribe of the feed stores its subscription before it finds the abandoned subscribe pending
		retry, err := trades().ExpandTemplates(ex)
		require.NoError(t, err, "ExpandTemplates must not error")
		retry[0].SetKey(abandoned.subscription.EnsureKeyed())
		require.NoError(t, ex.Websocket.AddSubscriptions(conn, retry...), "AddSubscriptions must not error")
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), conn, websocketErrorFrame(t, wsErrorInvalidSubscription+feed)), errWebsocketServer, "wsHandleData should report the late refusal")
		assert.Same(t, retry[0], ex.Websocket.GetSubscription(retry[0]), "a late refusal should leave the subscription the feed's subscribe stored since")
	})
	t.Run("unrelated error", func(t *testing.T) {
		t.Parallel()
		ex, conn, key := abandon(t, 0)
		other, otherKey := testOrderbookSubscription()
		otherPending := addWebsocketPending(t, ex, wsMethodSubscribe, other, &otherKey)
		assert.ErrorIs(t, ex.wsHandleData(t.Context(), conn, websocketErrorFrame(t, "unexpected")), errWebsocketServer, "wsHandleData should report the error")
		assert.ErrorIs(t, receivePendingResult(t, otherPending), errWebsocketServer, "an error naming no operation should fail the connection's operations")
		assert.NotNil(t, pendingWebsocketOperation(ex, key), "an error naming no operation should keep the abandoned subscribe, whose feed may yet be acknowledged")
	})
	t.Run("connection lost", func(t *testing.T) {
		t.Parallel()
		ex, conn, key := abandon(t, 0)
		assert.Equal(t, websocket.ErrNotConnected, ex.failWebsocketPending(conn, websocket.ErrNotConnected), "failWebsocketPending should return the cause alone, as an abandoned subscribe is already rolled back")
		assert.Nil(t, pendingWebsocketOperation(ex, key), "a lost connection should free the feed's slot")
	})
	t.Run("subscription limit", func(t *testing.T) {
		t.Parallel()
		ex, conn, _ := abandon(t, maximumWebsocketSubscriptions-1)
		other := subscription.List{{Channel: subscription.AllTradesChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{testBuilderPair}}}
		assert.ErrorIs(t, subscribeWebsocketTest(t, ex, other), errWebsocketSubscriptionsLimit, "Subscribe should count the abandoned subscribe's slot")
		require.ErrorIs(t, ex.wsHandleData(t.Context(), conn, websocketErrorFrame(t, wsErrorInvalidSubscription+feed)), errWebsocketServer, "wsHandleData must report the late refusal")
		assert.NoError(t, subscribeWebsocketTest(t, ex, other), "Subscribe should use the slot the refused subscribe freed")
	})
	t.Run("account feed", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		// The default account feeds, myOrders and myTrades, each keep a slot
		_, err := ex.generatePublicSubscriptions()
		require.NoError(t, err, "generatePublicSubscriptions must not error")
		account := accountConn(t, ex)
		account.mu.Lock()
		account.reply = nil
		account.mu.Unlock()
		ex.WebsocketResponseMaxLimit = time.Millisecond
		require.ErrorIs(t, subscribeWebsocketTest(t, ex, subscription.List{{Channel: subscription.MyTradesChannel, Authenticated: true}}), websocket.ErrSignatureTimeout, "Subscribe must time out without an acknowledgement")
		held := make(subscription.List, maximumWebsocketSubscriptions-3)
		for i := range held {
			held[i] = &subscription.Subscription{Channel: wsChannelAllMids, QualifiedChannel: wsChannelAllMids, Params: map[string]any{"dex": "d" + strconv.Itoa(i)}}
		}
		keyWebsocketSubscriptions(held)
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(publicConn(t, ex), held...), "AddSuccessfulSubscriptions must not error")
		assert.NoError(t, subscribeWebsocketTest(t, ex, trades()), "Subscribe should count the abandoned account feed in its own slot rather than keep another for it")
		other := subscription.List{{Channel: subscription.AllTradesChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{testBuilderPair}}}
		assert.ErrorIs(t, subscribeWebsocketTest(t, ex, other), errWebsocketSubscriptionsLimit, "Subscribe should keep myOrders' slot")
	})
	t.Run("account feed of an address no longer watched", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		account := accountConn(t, ex)
		account.mu.Lock()
		account.reply = nil
		account.mu.Unlock()
		ex.WebsocketResponseMaxLimit = time.Millisecond
		require.ErrorIs(t, subscribeWebsocketTest(t, ex, subscription.List{{Channel: subscription.MyTradesChannel, Authenticated: true}}), websocket.ErrSignatureTimeout, "Subscribe must time out without an acknowledgement")
		setTestCredentials(ex, &accounts.Credentials{Key: testOtherAddress})
		// The default account feeds of the newly watched address, myOrders and myTrades, each keep a slot
		_, err := ex.generatePublicSubscriptions()
		require.NoError(t, err, "generatePublicSubscriptions must not error")
		held := make(subscription.List, maximumWebsocketSubscriptions-3)
		for i := range held {
			held[i] = &subscription.Subscription{Channel: wsChannelAllMids, QualifiedChannel: wsChannelAllMids, Params: map[string]any{"dex": "d" + strconv.Itoa(i)}}
		}
		keyWebsocketSubscriptions(held)
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(publicConn(t, ex), held...), "AddSuccessfulSubscriptions must not error")
		assert.ErrorIs(t, subscribeWebsocketTest(t, ex, trades()), errWebsocketSubscriptionsLimit, "Subscribe should keep the watched address's myTrades slot beside the abandoned feed of the address watched before")
		account.mu.Lock()
		account.reply = acknowledge
		account.mu.Unlock()
		assert.NoError(t, subscribeWebsocketTest(t, ex, subscription.List{{Channel: subscription.MyTradesChannel, Authenticated: true}}), "Subscribe should subscribe the watched address's myTrades in its slot")
	})
	t.Run("cancelled", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		conn := publicConn(t, ex)
		conn.mu.Lock()
		conn.reply = nil
		conn.mu.Unlock()
		expanded, err := trades().ExpandTemplates(ex)
		require.NoError(t, err, "ExpandTemplates must not error")
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		assert.ErrorIs(t, ex.websocketSubscribe(ctx, conn, expanded), context.Canceled, "websocketSubscribe should stop when its context ends")
		assert.Empty(t, ex.Websocket.GetSubscriptions(), "websocketSubscribe should roll back the subscribe it gave up on")
		assert.NotNil(t, pendingWebsocketOperation(ex, &websocketPendingKey{connection: conn, subscription: WebsocketSubscription{Type: wsChannelTrades, Coin: "BTC"}}), "websocketSubscribe should keep a sent subscribe whose context ended pending")
	})
}

func TestCheckWebsocketFeed(t *testing.T) {
	t.Parallel()
	// TestWebsocketSubscriptionPayload cannot show these checks, as mapping the pair to a coin refuses each of them too
	for _, tc := range []struct {
		name string
		sub  *subscription.Subscription
		err  error
	}{
		{name: "an empty pair", sub: &subscription.Subscription{Channel: wsChannelActiveAssetData, Authenticated: true, Asset: asset.PerpetualContract, Pairs: currency.Pairs{currency.EMPTYPAIR}}, err: currency.ErrCurrencyPairEmpty},
		{name: "no asset", sub: &subscription.Subscription{Channel: subscription.AllTradesChannel, Pairs: currency.Pairs{perpetualPair}}, err: asset.ErrNotSupported},
		{name: "an unsupported asset", sub: &subscription.Subscription{Channel: subscription.AllTradesChannel, Asset: asset.Futures, Pairs: currency.Pairs{perpetualPair}}, err: asset.ErrNotSupported},
	} {
		_, err := e.checkWebsocketFeed(tc.sub)
		assert.ErrorIsf(t, err, tc.err, "checkWebsocketFeed should refuse a feed with %s", tc.name)
	}
}

func TestWebsocketSubscriptionPayload(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	for _, tc := range []struct {
		name string
		sub  *subscription.Subscription
		exp  WebsocketSubscription
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "unsupported channel", sub: &subscription.Subscription{Channel: "bogus"}, err: subscription.ErrNotSupported},
		{name: "authenticated market feed", sub: &subscription.Subscription{Channel: subscription.TickerChannel, Authenticated: true}, err: subscription.ErrNotSupported},
		{name: "unauthenticated account feed", sub: &subscription.Subscription{Channel: subscription.MyOrdersChannel}, err: subscription.ErrNotSupported},
		{name: "market feed without a pair", sub: &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.PerpetualContract}, err: subscription.ErrNotSinglePair},
		{name: "market feed with two pairs", sub: &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair, testBuilderPair}}, err: subscription.ErrNotSinglePair},
		{name: "unknown pair", sub: &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{currency.NewPair(currency.ETH, currency.USDC)}}, err: errPairMappingNotFound},
		{name: "spot activeAssetData", sub: &subscription.Subscription{Channel: wsChannelActiveAssetData, Authenticated: true, Asset: asset.Spot, Pairs: currency.Pairs{spotPair}}, err: asset.ErrNotSupported},
		{name: "feed with an asset", sub: &subscription.Subscription{Channel: wsChannelAllMids, Asset: asset.Spot}, err: subscription.ErrNotSupported},
		{name: "unknown DEX", sub: &subscription.Subscription{Channel: wsChannelAllMids, Params: map[string]any{"dex": "abc"}}, err: errInvalidPerpetualDEX},
		{name: "unsupported interval", sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}, Interval: kline.TenMin}, err: kline.ErrUnsupportedInterval},
		{name: "unsupported orderbook levels", sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}, Levels: 10}, err: subscription.ErrInvalidLevel},
		{name: "invalid aggregateByTime", sub: &subscription.Subscription{Channel: wsChannelUserFills, Authenticated: true, Params: map[string]any{"aggregateByTime": "yes"}}, err: errWebsocketParameter},
		{name: "invalid ignorePortfolioMargin", sub: &subscription.Subscription{Channel: wsChannelSpotState, Authenticated: true, Params: map[string]any{"ignorePortfolioMargin": 1}}, err: errWebsocketParameter},
		{
			name: "perpetual ticker",
			sub:  &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}},
			exp:  WebsocketSubscription{Type: wsChannelActiveAssetContext, Coin: "BTC"},
		},
		{
			name: "spot ticker",
			sub:  &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotPair}},
			exp:  WebsocketSubscription{Type: wsChannelActiveAssetContext, Coin: "@107"},
		},
		{
			name: "builder DEX trades",
			sub:  &subscription.Subscription{Channel: subscription.AllTradesChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{testBuilderPair}},
			exp:  WebsocketSubscription{Type: wsChannelTrades, Coin: "xyz:XYZ100"},
		},
		{
			name: "candles",
			sub:  &subscription.Subscription{Channel: subscription.CandlesChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotPair}, Interval: kline.OneMonth},
			exp:  WebsocketSubscription{Type: wsChannelCandle, Coin: "@107", Interval: "1M"},
		},
		{
			name: "orderbook",
			sub:  &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}, Levels: 20},
			exp:  WebsocketSubscription{Type: wsChannelOrderbook, Coin: "BTC"},
		},
		{
			name: "fast aggregated orderbook",
			sub:  &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}, Levels: 5, Params: map[string]any{"nSigFigs": 5.0, "mantissa": 2.0}},
			exp:  WebsocketSubscription{Type: wsChannelOrderbook, Coin: "BTC", NumberOfSignificantFigures: 5, Mantissa: 2, Fast: true},
		},
		{
			name: "best bid and offer",
			sub:  &subscription.Subscription{Channel: wsChannelBestBidOffer, Asset: asset.Spot, Pairs: currency.Pairs{spotPair}},
			exp:  WebsocketSubscription{Type: wsChannelBestBidOffer, Coin: "@107"},
		},
		{
			name: "activeAssetData",
			sub:  &subscription.Subscription{Channel: wsChannelActiveAssetData, Authenticated: true, Asset: asset.PerpetualContract, Pairs: currency.Pairs{testBuilderPair}},
			exp:  WebsocketSubscription{Type: wsChannelActiveAssetData, Coin: "xyz:XYZ100", User: testAccountAddress},
		},
		{
			name: "allMids",
			sub:  &subscription.Subscription{Channel: wsChannelAllMids, Params: map[string]any{"dex": ""}},
			exp:  WebsocketSubscription{Type: wsChannelAllMids},
		},
		{
			name: "builder DEX allMids",
			sub:  &subscription.Subscription{Channel: wsChannelAllMids, Params: map[string]any{"dex": "xyz"}},
			exp:  WebsocketSubscription{Type: wsChannelAllMids, DEX: "xyz"},
		},
		{
			name: "my orders",
			sub:  &subscription.Subscription{Channel: subscription.MyOrdersChannel, Authenticated: true},
			exp:  WebsocketSubscription{Type: wsChannelOrderUpdates, User: testAccountAddress},
		},
		{
			name: "my trades",
			sub:  &subscription.Subscription{Channel: subscription.MyTradesChannel, Authenticated: true},
			exp:  WebsocketSubscription{Type: wsSubscriptionUserEvents, User: testAccountAddress},
		},
		{
			name: "builder DEX clearinghouseState",
			sub:  &subscription.Subscription{Channel: wsChannelClearinghouseState, Authenticated: true, Params: map[string]any{"dex": "xyz"}},
			exp:  WebsocketSubscription{Type: wsChannelClearinghouseState, User: testAccountAddress, DEX: "xyz"},
		},
		{
			name: "userFills aggregated by time",
			sub:  &subscription.Subscription{Channel: wsChannelUserFills, Authenticated: true, Params: map[string]any{"aggregateByTime": true}},
			exp:  WebsocketSubscription{Type: wsChannelUserFills, User: testAccountAddress, AggregateByTime: true},
		},
		{
			name: "spotState without portfolio margin",
			sub:  &subscription.Subscription{Channel: wsChannelSpotState, Authenticated: true, Params: map[string]any{"ignorePortfolioMargin": true}},
			exp:  WebsocketSubscription{Type: wsChannelSpotState, User: testAccountAddress, IgnorePortfolioMargin: true},
		},
		{
			name: "fastAssetCtxs",
			sub:  &subscription.Subscription{Channel: wsChannelFastAssetContexts},
			exp:  WebsocketSubscription{Type: wsChannelFastAssetContexts},
		},
	} {
		payload, err := ex.websocketSubscriptionPayload(t.Context(), tc.sub)
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "websocketSubscriptionPayload should reject the %s subscription", tc.name)
			continue
		}
		require.NoErrorf(t, err, "websocketSubscriptionPayload must not error for the %s subscription", tc.name)
		assert.Equalf(t, tc.exp, payload, "websocketSubscriptionPayload should build the %s subscription", tc.name)
	}

	watchless := newWebsocketTestExchange(t)
	watchless.SetCredentials(&accounts.Credentials{})
	_, err := watchless.websocketSubscriptionPayload(t.Context(), &subscription.Subscription{Channel: subscription.MyOrdersChannel, Authenticated: true})
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "websocketSubscriptionPayload should require an account address for an account feed")
}

func TestWebsocketOrderbookOptions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		levels int
		params map[string]any
		exp    WebsocketSubscription
		err    error
	}{
		{exp: WebsocketSubscription{}},
		{levels: 20, params: map[string]any{"nSigFigs": 4}, exp: WebsocketSubscription{NumberOfSignificantFigures: 4}},
		{levels: 5, params: map[string]any{"nSigFigs": uint64(5), "mantissa": 5}, exp: WebsocketSubscription{NumberOfSignificantFigures: 5, Mantissa: 5, Fast: true}},
		{levels: 10, err: subscription.ErrInvalidLevel},
		{params: map[string]any{"nSigFigs": "5"}, err: errWebsocketParameter},
		{params: map[string]any{"mantissa": -1}, err: errWebsocketParameter},
		{params: map[string]any{"nSigFigs": 7}, err: errInvalidSignificantFigures},
		{params: map[string]any{"nSigFigs": 4, "mantissa": 2}, err: errInvalidMantissa},
		{params: map[string]any{"nSigFigs": 5, "mantissa": 1}, err: errInvalidMantissa},
	} {
		var payload WebsocketSubscription
		err := websocketOrderbookOptions(&subscription.Subscription{Levels: tc.levels, Params: tc.params}, &payload)
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "websocketOrderbookOptions should reject %d levels with %v", tc.levels, tc.params)
			continue
		}
		require.NoErrorf(t, err, "websocketOrderbookOptions must not error for %d levels with %v", tc.levels, tc.params)
		assert.Equalf(t, tc.exp, payload, "websocketOrderbookOptions should set the options of %d levels with %v", tc.levels, tc.params)
	}
}

func TestWebsocketDEXName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		params map[string]any
		exp    string
		err    error
	}{
		{},
		{params: map[string]any{"dex": nil}},
		{params: map[string]any{"dex": ""}},
		{params: map[string]any{"dex": "xyz"}, exp: "xyz"},
		{params: map[string]any{"dex": "XYZ"}, exp: "XYZ"},
		{params: map[string]any{"dex": 1}, err: errWebsocketParameter},
	} {
		dex, err := websocketDEXName(tc.params)
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "websocketDEXName should reject %v", tc.params)
			continue
		}
		require.NoErrorf(t, err, "websocketDEXName must not error for %v", tc.params)
		assert.Equalf(t, tc.exp, dex, "websocketDEXName should return the DEX of %v", tc.params)
	}
}

func TestValidateWebsocketDEX(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	assert.NoError(t, ex.validateWebsocketDEX(t.Context(), "xyz"), "validateWebsocketDEX should accept a registered DEX")
	assert.ErrorIs(t, ex.validateWebsocketDEX(t.Context(), "XYZ"), errInvalidPerpetualDEX, "validateWebsocketDEX should reject a DEX registered under another case")
	assert.ErrorIs(t, ex.validateWebsocketDEX(t.Context(), "abc"), errInvalidPerpetualDEX, "validateWebsocketDEX should reject an unregistered DEX")
	assert.ErrorIs(t, newUnavailableServerExchange(t).validateWebsocketDEX(t.Context(), "xyz"), request.ErrBadStatus, "validateWebsocketDEX should return a registry error")
}

func TestWebsocketUintParameter(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		value any
		exp   uint64
		err   error
	}{
		{},
		{value: uint64(5), exp: 5},
		{value: 5, exp: 5},
		{value: 5.0, exp: 5},
		{value: -1, err: errWebsocketParameter},
		{value: -1.0, err: errWebsocketParameter},
		{value: 5.5, err: errWebsocketParameter},
		{value: float64(math.MaxUint32) + 1, err: errWebsocketParameter},
		{value: "5", err: errWebsocketParameter},
	} {
		params := map[string]any{}
		if tc.value != nil {
			params["nSigFigs"] = tc.value
		}
		value, err := websocketUintParameter(params, "nSigFigs")
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "websocketUintParameter should reject %v", tc.value)
			continue
		}
		require.NoErrorf(t, err, "websocketUintParameter must not error for %v", tc.value)
		assert.Equalf(t, tc.exp, value, "websocketUintParameter should return the value of %v", tc.value)
	}
}

func TestWebsocketBoolParameter(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		value any
		exp   bool
		err   error
	}{
		{},
		{value: true, exp: true},
		{value: false},
		{value: "true", err: errWebsocketParameter},
	} {
		params := map[string]any{}
		if tc.value != nil {
			params["aggregateByTime"] = tc.value
		}
		value, err := websocketBoolParameter(params, "aggregateByTime")
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "websocketBoolParameter should reject %v", tc.value)
			continue
		}
		require.NoErrorf(t, err, "websocketBoolParameter must not error for %v", tc.value)
		assert.Equalf(t, tc.exp, value, "websocketBoolParameter should return the value of %v", tc.value)
	}
}
