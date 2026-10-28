//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package cryptodotcom

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mock"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

const (
	// mockHTTPFile holds the recorded REST responses: public methods' trimmed from live recordings, private methods'
	// following Crypto.com's documented examples
	mockHTTPFile = "testdata/http.json"
	// mockWebsocketAPIFiles hold the results the mock websocket answers each method with, one file per docs section
	mockWebsocketAPIFiles = "testdata/websocketapi/*.json"
)

var mockTests = true

var errWebsocketVenueRequest = errors.New("unexpected request to the mock websocket")

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := setupMockExchange(e); err != nil {
		log.Fatalf("Cryptodotcom mock setup error: %s", err)
	}
	os.Exit(m.Run())
}

// setupMockExchange serves an exchange from the recorded REST responses, signing with placeholder credentials so the
// private methods run against their recorded responses too; each private request's signature is checked before the
// mock matches it
func setupMockExchange(ex *Exchange) error {
	if err := setupTestExchange(ex); err != nil {
		return err
	}
	setTestCredentials(ex, &accounts.Credentials{Key: testAPIKey, Secret: testAPISecret})
	serverURL, client, err := mock.NewVCRServer(mockHTTPFile)
	if err != nil {
		return fmt.Errorf("error starting NewVCRServer: %w", err)
	}
	client.Transport = &signedRequestNormaliser{next: client.Transport}
	ex.SkipAuthCheck = true
	if err := ex.SetHTTPClient(client); err != nil {
		return err
	}
	for k := range ex.API.Endpoints.GetURLMap() {
		if err := ex.API.Endpoints.SetRunningURL(k, serverURL); err != nil {
			return err
		}
	}
	log.Printf(sharedtestvalues.MockTesting, ex.Name)
	// Mock responses are served locally, so throttling protects nothing
	return ex.DisableRateLimiter()
}

// newTestExchange returns a mock exchange named after the test, so the ticker, orderbook and account stores it writes
// do not collide with other tests'
func newTestExchange(tb testing.TB) *Exchange {
	tb.Helper()
	ex := new(Exchange)
	require.NoError(tb, setupMockExchange(ex), "setupMockExchange must not error")
	ex.Name = tb.Name()
	return ex
}

// signedRequestNormaliser checks a POST request's signature against the placeholder secret, then sends the mock its
// API key, if any, and each of its params as a body field, a nested object or array as its canonical JSON text, so the
// recorded responses, which leave out the ID, nonce and signature that change with every request, match it
type signedRequestNormaliser struct {
	next http.RoundTripper
}

// RoundTrip normalises a POST request's body and sends it on
func (n *signedRequestNormaliser) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Method != http.MethodPost || r.Body == nil {
		return n.next.RoundTrip(r)
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if err := r.Body.Close(); err != nil {
		return nil, err
	}
	normalised, err := normaliseSignedRequest(raw)
	if err != nil {
		return nil, err
	}
	out := r.Clone(r.Context())
	out.Body = io.NopCloser(bytes.NewReader(normalised))
	out.ContentLength = int64(len(normalised))
	out.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(normalised)), nil }
	return n.next.RoundTrip(out)
}

// normaliseSignedRequest checks a request body's signature, if it is signed, and returns the body the mock matches: its
// API key and its params as fields
func normaliseSignedRequest(raw []byte) ([]byte, error) {
	var body requestBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	params, err := paramString(body.Params)
	if err != nil {
		return nil, err
	}
	if body.APIKey != "" || body.Signature != "" {
		signature, err := signRequest(testAPISecret, body.Method, body.ID, body.APIKey, params, body.Nonce)
		if err != nil {
			return nil, err
		}
		if body.APIKey != testAPIKey || signature != body.Signature {
			return nil, fmt.Errorf("%s: request not signed with the test credentials", body.Method)
		}
	}
	fields := make(map[string]string)
	if body.APIKey != "" {
		fields["api_key"] = body.APIKey
	}
	if err := jsonparser.ObjectEach(body.Params, func(key, value []byte, dataType jsonparser.ValueType, _ int) error {
		k, err := jsonparser.ParseString(key)
		if err != nil {
			return err
		}
		switch dataType {
		case jsonparser.String:
			v, err := jsonparser.ParseString(value)
			if err != nil {
				return err
			}
			fields[k] = v
		case jsonparser.Object, jsonparser.Array:
			var v any
			if err := json.Unmarshal(value, &v); err != nil {
				return err
			}
			canonical, err := json.Marshal(v)
			if err != nil {
				return err
			}
			fields[k] = string(canonical)
		default:
			fields[k] = string(value)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return json.Marshal(fields)
}

// newWebsocketTestExchange returns a mock exchange named after the test whose trade connection is connected to a new
// mock of Crypto.com's user API websocket and authenticated, and the mock, which records the requests it receives
func newWebsocketTestExchange(t *testing.T) (*Exchange, *websocketVenue) {
	t.Helper()
	venue := newWebsocketVenue(t)
	ex := newTestExchange(t)
	connectWebsocketVenue(t, ex, venue)
	return ex, venue
}

// connectWebsocketVenue connects an exchange's trade connection to a mock of Crypto.com's user API websocket and
// authenticates it
func connectWebsocketVenue(t *testing.T, ex *Exchange, venue *websocketVenue) {
	t.Helper()
	require.NoError(t, ex.Websocket.SetAllConnectionURLs(venue.url), "SetAllConnectionURLs must not error")
	ex.Features.Subscriptions = subscription.List{}
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must connect to the mock websocket")
	t.Cleanup(func() {
		// Disabling stops the connection monitor, which would otherwise redial the mock after the test, and which may then
		// shut the connection down first
		assert.NoError(t, ex.Websocket.Disable(), "Disable should not error")
		if err := ex.Websocket.Shutdown(); err != nil {
			assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should only fail when the connection monitor has shut the connection down first")
		}
	})
	// Cleanups run in reverse, so the mock stops reporting errors before the exchange shuts its connections down
	t.Cleanup(venue.closing)
	require.True(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "the mock websocket must authenticate the trade connection")
}

// websocketVenueRequest is a request the mock websocket received
type websocketVenueRequest struct {
	ID     int64
	Method string
	Params json.RawMessage
}

// websocketVenue mocks Crypto.com's user API websocket. It authenticates a public/auth request signed with the
// placeholder credentials, acknowledges subscriptions, unless a test set an error refusing subscribe requests as a
// whole, and answers every other method with the error a test set for it, else the result a test set for it, else the
// result recorded for it in testdata/websocketapi; it records every request. As a mock of the market data websocket it
// answers a channel's subscriptions with the error or the frames set for the channel in place of an acknowledgement
type websocketVenue struct {
	url      string
	mu       sync.Mutex
	requests []websocketVenueRequest
	results  map[string]json.RawMessage
	errs     map[string]*APIError
	// answers holds the frames each channel's subscriptions are answered with, a set for each subscription in turn, the
	// last set answering any after
	answers map[string][][]json.RawMessage
	// rejections holds the errors channels' subscriptions are answered with
	rejections map[string]*APIError
	// hooks holds the functions run on receiving a method's request, before it is answered
	hooks map[string]func()
	// closed stops the mock reporting errors answering, once the exchange going away ends the frames being written
	closed bool
}

var venueUpgrader = gws.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

// newWebsocketVenue starts a mock of Crypto.com's user API websocket, loaded with the recorded results; it listens on
// loopback, as the exchange dials with a plain dialer that cannot reach the in-memory network
func newWebsocketVenue(t *testing.T) *websocketVenue {
	t.Helper()
	venue := &websocketVenue{
		results:    make(map[string]json.RawMessage),
		errs:       make(map[string]*APIError),
		answers:    make(map[string][][]json.RawMessage),
		rejections: make(map[string]*APIError),
		hooks:      make(map[string]func()),
	}
	files, err := filepath.Glob(mockWebsocketAPIFiles)
	require.NoError(t, err, "Glob must not error")
	for _, file := range files {
		raw, err := os.ReadFile(file)
		require.NoErrorf(t, err, "ReadFile must not error for %s", file)
		var results map[string]json.RawMessage
		require.NoErrorf(t, json.Unmarshal(raw, &results), "the recorded results in %s must decode", file)
		for method, result := range results {
			_, dup := venue.results[method]
			require.Falsef(t, dup, "method %s must be recorded once", method)
			venue.results[method] = result
		}
	}
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := venueUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if err := venue.answer(conn, msg); err != nil {
				venue.mu.Lock()
				closed := venue.closed
				venue.mu.Unlock()
				if !closed {
					assert.NoError(t, err, "the mock websocket should answer each request")
				}
			}
		}
	}))
	server.Start()
	venue.url = "ws" + strings.TrimPrefix(server.URL, "http")
	return venue
}

// closing stops the mock reporting errors writing to clients, which are going away
func (v *websocketVenue) closing() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.closed = true
}

// respond has the mock answer a method with result in place of the recorded one
func (v *websocketVenue) respond(method, result string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.results[method] = json.RawMessage(result)
}

// fail has the mock answer a method with an error
func (v *websocketVenue) fail(method string, err *APIError) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.errs[method] = err
}

// answerSubscriptions has the mock answer a channel's subscriptions with a set of frames each, in turn, the last set
// answering any after, in place of an acknowledgement; a frame with a positive ID, the answer to the request it was
// recorded for, is sent with the ID of the request it answers
func (v *websocketVenue) answerSubscriptions(channel string, sets ...[]json.RawMessage) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.answers[channel] = sets
}

// beforeAnswering has the mock run fn on receiving a method's request, before answering it
func (v *websocketVenue) beforeAnswering(method string, fn func()) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.hooks[method] = fn
}

// rejectSubscriptions has the mock answer a channel's subscriptions with an error
func (v *websocketVenue) rejectSubscriptions(channel string, err *APIError) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.rejections[channel] = err
}

// received returns the requests received for a method, in arrival order
func (v *websocketVenue) received(method string) []websocketVenueRequest {
	v.mu.Lock()
	defer v.mu.Unlock()
	var requests []websocketVenueRequest
	for i := range v.requests {
		if v.requests[i].Method == method {
			requests = append(requests, v.requests[i])
		}
	}
	return requests
}

// answer records a request and writes the mock's answer to it. A subscribe request's answers are chosen as the request
// is recorded, so a test that has seen the request knows what it was answered with
func (v *websocketVenue) answer(conn *gws.Conn, msg []byte) error {
	var req websocketRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return err
	}
	var channels []string
	if req.Method == websocketMethodSubscribe {
		var err error
		if channels, err = requestChannels(req.Params); err != nil {
			return err
		}
	}
	v.mu.Lock()
	v.requests = append(v.requests, websocketVenueRequest{ID: req.ID, Method: req.Method, Params: req.Params})
	result, hasResult := v.results[req.Method]
	apiErr := v.errs[req.Method]
	var answers []subscriptionAnswer
	if apiErr == nil {
		for _, channel := range channels {
			answers = append(answers, v.nextSubscriptionAnswer(channel))
		}
	}
	hook := v.hooks[req.Method]
	v.mu.Unlock()
	if hook != nil {
		hook()
	}
	resp := map[string]any{"id": req.ID, "method": req.Method, "code": 0}
	switch req.Method {
	case websocketMethodRespondHeartbeat:
		return nil
	case websocketMethodAuth:
		signature, err := signRequest(testAPISecret, req.Method, uint64(req.ID), req.APIKey, "", req.Nonce) //nolint:gosec // Request IDs count up from 1
		if err != nil {
			return err
		}
		if req.APIKey != testAPIKey || signature != req.Signature {
			resp["code"], resp["message"] = 40101, "Unauthorized"
		}
	case websocketMethodSubscribe:
		if apiErr != nil {
			// A request refused as a whole is answered once, naming none of its channels
			resp["code"], resp["message"] = apiErr.Code, apiErr.Message
			return conn.WriteJSON(resp)
		}
		for i := range answers {
			if err := answers[i].write(conn, req.ID); err != nil {
				return err
			}
		}
		return nil
	case websocketMethodUnsubscribe:
		if apiErr != nil {
			resp["code"], resp["message"] = apiErr.Code, apiErr.Message
		}
	default:
		switch {
		case apiErr != nil:
			resp["code"], resp["message"] = apiErr.Code, apiErr.Message
		case hasResult:
			resp["result"] = result
		default:
			resp["code"], resp["message"] = 40002, "METHOD_NOT_FOUND"
			if err := conn.WriteJSON(resp); err != nil {
				return err
			}
			return fmt.Errorf("%w: %s has no recorded result", errWebsocketVenueRequest, req.Method)
		}
	}
	return conn.WriteJSON(resp)
}

// subscriptionAnswer is the mock's answer to a subscription request's channel: the error set for the channel, else the
// frames set for it, else an acknowledgement
type subscriptionAnswer struct {
	channel string
	err     *APIError
	frames  []json.RawMessage
}

// nextSubscriptionAnswer returns the answer to a channel's next subscription, which takes the next set of frames set for
// the channel, the last set answering any after; the caller holds the mock's lock
func (v *websocketVenue) nextSubscriptionAnswer(channel string) subscriptionAnswer {
	a := subscriptionAnswer{channel: channel, err: v.rejections[channel]}
	if sets := v.answers[channel]; len(sets) != 0 {
		a.frames = sets[0]
		if len(sets) > 1 {
			v.answers[channel] = sets[1:]
		}
	}
	return a
}

// write writes a subscription's answer, with the ID of the request it answers
func (a *subscriptionAnswer) write(conn *gws.Conn, id int64) error {
	switch {
	case a.err != nil:
		return conn.WriteJSON(map[string]any{"id": id, "method": websocketMethodSubscribe, "code": a.err.Code, "channel": a.channel, "message": a.err.Message})
	case len(a.frames) == 0:
		return conn.WriteJSON(map[string]any{"id": id, "method": websocketMethodSubscribe, "code": 0, "channel": a.channel})
	}
	for _, frame := range a.frames {
		frame, err := withRequestID(frame, id)
		if err != nil {
			return err
		}
		if err := conn.WriteMessage(gws.TextMessage, frame); err != nil {
			return err
		}
	}
	return nil
}

// withRequestID returns a recorded frame with the ID of the request it answers in place of the recorded request's, or
// a push's frame, whose ID is -1, as it is
func withRequestID(frame []byte, id int64) ([]byte, error) {
	recorded, err := jsonparser.GetInt(frame, "id")
	if err != nil || recorded <= 0 {
		return frame, err
	}
	return jsonparser.Set(bytes.Clone(frame), []byte(strconv.FormatInt(id, 10)), "id")
}

// newMarketDataVenue starts a mock of Crypto.com's websocket that answers each channel's subscriptions with the market
// data frames recorded for the channel in testdata/wsPublic.json, else acknowledges them
func newMarketDataVenue(t *testing.T) *websocketVenue {
	t.Helper()
	return newRecordedVenue(t, marketDataFrames)
}

// newUserDataVenue starts a mock of Crypto.com's websocket that answers each user channel's subscriptions with the
// frames documented for the channel in testdata/wsAccount.json, else acknowledges them
func newUserDataVenue(t *testing.T) *websocketVenue {
	t.Helper()
	return newRecordedVenue(t, accountFrames)
}

// newRecordedVenue starts a mock of Crypto.com's websocket that answers each channel's subscriptions with the frames
// recorded for the channel, which frames loads, else acknowledges them
func newRecordedVenue(t *testing.T, frames func() (map[string][][]json.RawMessage, error)) *websocketVenue {
	t.Helper()
	venue := newWebsocketVenue(t)
	recorded, err := frames()
	require.NoError(t, err, "the recorded frames must load")
	for channel, sets := range recorded {
		venue.answerSubscriptions(channel, sets...)
	}
	return venue
}

// newMarketDataTestExchange returns a mock exchange named after the test, with its trade feed enabled and prepared by
// each of prepare, whose market data connections are subscribed to subs on venue, and the error connecting, which
// reports the subscriptions venue rejected; the connections are not authenticated
func newMarketDataTestExchange(t *testing.T, venue *websocketVenue, subs subscription.List, prepare ...func(*Exchange)) (*Exchange, error) {
	t.Helper()
	ex := newTestExchange(t)
	require.NoError(t, ex.Websocket.SetAllConnectionURLs(venue.url), "SetAllConnectionURLs must not error")
	enableTradeFeed(ex)
	for _, p := range prepare {
		p(ex)
	}
	ex.Features.Subscriptions = subs
	ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
	return ex, connectVenue(t, ex, venue)
}

// newUserDataTestExchange returns a mock exchange named after the test, prepared by each of prepare, whose user
// connection is authenticated and subscribed to subs on venue, and the error connecting, which reports the
// subscriptions venue rejected
func newUserDataTestExchange(t *testing.T, venue *websocketVenue, subs subscription.List, prepare ...func(*Exchange)) (*Exchange, error) {
	t.Helper()
	ex := newTestExchange(t)
	require.NoError(t, ex.Websocket.SetAllConnectionURLs(venue.url), "SetAllConnectionURLs must not error")
	for _, p := range prepare {
		p(ex)
	}
	ex.Features.Subscriptions = subs
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	return ex, connectVenue(t, ex, venue)
}

// connectVenue connects an exchange's websocket to venue, disconnecting it when the test ends, and returns the error
// connecting
func connectVenue(t *testing.T, ex *Exchange, venue *websocketVenue) error {
	t.Helper()
	err := ex.Websocket.Connect(t.Context())
	t.Cleanup(func() {
		// Disabling stops the connection monitor, which would otherwise redial the mock after the test, and which may then
		// shut the connection down first
		assert.NoError(t, ex.Websocket.Disable(), "Disable should not error")
		if err := ex.Websocket.Shutdown(); err != nil {
			assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should only fail when the connection monitor has shut the connection down first")
		}
	})
	// Cleanups run in reverse, so the mock stops reporting errors before the exchange shuts its connections down
	t.Cleanup(venue.closing)
	return err
}

// requestChannels returns the channels a subscribe or unsubscribe request names
func requestChannels(params json.RawMessage) ([]string, error) {
	var p subscriptionRequest
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	return p.Channels, nil
}
