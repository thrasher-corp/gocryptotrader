//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package coinbase

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
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
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mock"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

var mockTests = true

var errDerivativesVenueRequest = errors.New("unexpected request to the mock gateway websocket")

func TestMain(m *testing.M) {
	creds, err := newTestCredentials()
	if err != nil {
		log.Fatalf("Coinbase mock credentials error: %s", err)
	}
	e = new(Exchange)
	if err := setupMockExchange(e, creds); err != nil {
		log.Fatalf("Coinbase mock setup error: %s", err)
	}
	os.Exit(m.Run())
}

// setupMockExchange serves an exchange from testdata/http.json, whose public endpoint responses are trimmed from live
// recordings and whose authenticated endpoint responses come from Coinbase's sandbox or documented examples, with
// placeholder credentials and gateway access token, so the authenticated endpoints run against their recorded
// responses too
func setupMockExchange(ex *Exchange, creds *accounts.Credentials) error {
	if err := setupTestExchange(ex); err != nil {
		return err
	}
	setTestCredentials(ex, creds)
	ex.derivativesToken.set(creds.Key, testDerivativesAccessToken, time.Now().Add(24*time.Hour))
	if err := testexch.MockHTTPInstance(ex); err != nil {
		return err
	}
	// Mock responses are served locally, so throttling protects nothing
	return ex.DisableRateLimiter()
}

// newTestExchange returns a mock exchange named after the test, so the ticker, order book and account stores it
// writes do not collide with other tests'
func newTestExchange(tb testing.TB) *Exchange {
	tb.Helper()
	ex := new(Exchange)
	require.NoError(tb, setupMockExchange(ex, testCredentials(tb)), "setupMockExchange must not error")
	ex.Name = tb.Name()
	return ex
}

// newDerivativesWebsocketTestExchange returns a mock exchange named after the test whose gateway private websocket
// connection is connected, and authenticated, to a new mock of the gateway's websocket, and the mock, which records
// the requests it receives
func newDerivativesWebsocketTestExchange(t *testing.T) (*Exchange, *derivativesVenue) {
	t.Helper()
	ex, venue := connectDerivativesWebsocketTestExchange(t, true)
	require.True(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "the mock gateway websocket must authenticate the connection")
	return ex, venue
}

// newDedicatedDerivativesWebsocketTestExchange returns a mock exchange of the test's own, as
// newDerivativesWebsocketTestExchange does for every mock test, connected and authenticated to a new mock of the
// gateway's websocket, and the mock
func newDedicatedDerivativesWebsocketTestExchange(t *testing.T) (*Exchange, *derivativesVenue) {
	t.Helper()
	return newDerivativesWebsocketTestExchange(t)
}

// newUnauthenticatedDerivativesWebsocketTestExchange returns a mock exchange named after the test whose gateway
// private websocket connection is connected to a new mock of the gateway's websocket without authenticating
func newUnauthenticatedDerivativesWebsocketTestExchange(t *testing.T) (*Exchange, *derivativesVenue) {
	t.Helper()
	ex, venue := connectDerivativesWebsocketTestExchange(t, false)
	require.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "the connection must not be authenticated")
	return ex, venue
}

// connectDerivativesWebsocketTestExchange connects a new mock exchange's websocket to a new mock of the gateway's
// websocket, authenticating when authenticated is set. The websocket is disabled and shut down when the test ends,
// so its connection monitor does not redial the closed mock
func connectDerivativesWebsocketTestExchange(t *testing.T, authenticated bool) (*Exchange, *derivativesVenue) {
	t.Helper()
	venue := new(derivativesVenue)
	server := httptest.NewTestServer(t, mockws.CurryWsMockUpgrader(t, venue.answer))
	// The exchange dials with its own dialer, which reaches a listener but not the in-memory network
	server.Start()
	ex := newTestExchange(t)
	require.NoError(t, setupDerivativesWebsocket(ex), "setupDerivativesWebsocket must not error")
	require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(server.URL, "http")), "SetAllConnectionURLs must not error")
	ex.Features.Subscriptions = subscription.List{}
	ex.Websocket.SetSubscriptionsNotRequired()
	ex.Websocket.SetCanUseAuthenticatedEndpoints(authenticated)
	t.Cleanup(func() { stopWebsocketTestExchange(t, ex) })
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must connect to the mock gateway websocket")
	return ex, venue
}

// derivativesVenue mocks the gateway's websocket: it authenticates a CDP JWT signed with the placeholder key, sets
// heartbeats, acknowledges every channel of a subscription and answers any other request with the testdata/http.json
// response recorded for the same method and params
type derivativesVenue struct {
	mu       sync.Mutex
	requests []derivativesVenueRequest
	// overrides holds the response body the mock answers a method with in place of its usual one
	overrides map[string]json.RawMessage
	// respond, when set, answers the requests it reports handling, ahead of the overrides
	respond func(method string, params []byte) (body []byte, handled bool)
}

// respondWith has the mock answer the requests respond handles with the body it returns for each
func (v *derivativesVenue) respondWith(respond func(method string, params []byte) ([]byte, bool)) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.respond = respond
}

// override has the mock answer a method with a response body holding its result or error
func (v *derivativesVenue) override(method string, body json.RawMessage) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.overrides == nil {
		v.overrides = make(map[string]json.RawMessage)
	}
	v.overrides[method] = body
}

// derivativesVenueRequest is a JSON-RPC request the mock received
type derivativesVenueRequest struct {
	ID     uint64          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// requestsFor returns the requests the mock received for a method
func (v *derivativesVenue) requestsFor(method string) []derivativesVenueRequest {
	v.mu.Lock()
	defer v.mu.Unlock()
	return slices.DeleteFunc(slices.Clone(v.requests), func(r derivativesVenueRequest) bool { return r.Method != method })
}

// derivativesVenueRequestLimit is how many requests a mock of the gateway's websocket answers before refusing any but a
// logout, well above the 23 the busiest test exchange's mock receives in a run of the package
const derivativesVenueRequestLimit = 100

// answer replies to a request as the gateway does; a request without a recorded response is answered with an error,
// so it fails at once, and reported. A pager that stops advancing asks forever, so past derivativesVenueRequestLimit a
// request other than a logout is refused the same way, failing the test rather than hanging it
func (v *derivativesVenue) answer(_ testing.TB, msg []byte, conn *gws.Conn) error {
	var req derivativesVenueRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return fmt.Errorf("%w: %s: %w", errDerivativesVenueRequest, msg, err)
	}
	v.mu.Lock()
	v.requests = append(v.requests, req)
	received := len(v.requests)
	v.mu.Unlock()
	if req.Method == "private/logout" {
		return conn.Close()
	}
	var result json.RawMessage
	var answerErr error
	if received > derivativesVenueRequestLimit {
		answerErr = fmt.Errorf("%w: %s past the limit of %d requests", errDerivativesVenueRequest, req.Method, derivativesVenueRequestLimit)
	} else {
		result, answerErr = v.result(&req)
	}
	if answerErr != nil {
		result = json.RawMessage(`{"error":{"code":-32601,"message":"no mock response"}}`)
	}
	frame, err := derivativesVenueFrame(req.ID, result)
	if err != nil {
		return err
	}
	// A reply to a client that has closed the connection, as one does when its test ends during a reconnection, goes
	// nowhere, and reporting it would fail a test that has already finished
	_ = conn.WriteMessage(gws.TextMessage, frame)
	return answerErr
}

// result returns the response body answering a request, holding its result or error
func (v *derivativesVenue) result(req *derivativesVenueRequest) (json.RawMessage, error) {
	v.mu.Lock()
	respond := v.respond
	body, ok := v.overrides[req.Method]
	v.mu.Unlock()
	if respond != nil {
		if body, handled := respond(req.Method, req.Params); handled {
			return json.RawMessage(body), nil
		}
	}
	if ok {
		return body, nil
	}
	switch req.Method {
	case derivativesAuthMethod:
		var params DerivativesAuthWireRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, err
		}
		if params.GrantType != derivativesGrantType || params.Token == "" {
			return json.RawMessage(`{"error":{"code":13004,"message":"invalid_credentials"}}`), nil
		}
		// A websocket JWT is not bound to a request, so one carrying a URI is refused
		if jwt, err := parseTestJWT(params.Token); err != nil || jwt.Claims["uri"] != nil {
			return json.RawMessage(`{"error":{"code":13004,"message":"invalid_credentials"}}`), nil //nolint:nilerr // the mock answers a JWT it cannot verify as the gateway does
		}
		return json.RawMessage(`{"result":{"access_token":"mock-websocket-session","expires_in":3000,"refresh_token":"mock-refresh-token","scope":"connection mainaccount","enabled_features":[],"token_type":"bearer"}}`), nil
	case "public/set_heartbeat", "public/disable_heartbeat", "public/unsubscribe_all", "private/unsubscribe_all":
		return json.RawMessage(`{"result":"ok"}`), nil
	case "public/subscribe", "private/subscribe", "public/unsubscribe", "private/unsubscribe":
		var params struct {
			Channels []string `json:"channels"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, err
		}
		channels, err := json.Marshal(params.Channels)
		if err != nil {
			return nil, err
		}
		return json.RawMessage(`{"result":` + string(channels) + `}`), nil
	}
	return recordedDerivativesResponse(req.Method, req.Params)
}

// derivativesVenueFrame returns the frame answering a request with its ID and a response body's result or error
func derivativesVenueFrame(id uint64, body json.RawMessage) ([]byte, error) {
	var response map[string]json.RawMessage
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	frame := map[string]json.RawMessage{"jsonrpc": json.RawMessage(`"2.0"`)}
	if id != 0 {
		frame["id"] = json.RawMessage(strconv.FormatUint(id, 10))
	}
	for _, field := range []string{"result", "error"} {
		if value, ok := response[field]; ok {
			frame[field] = value
		}
	}
	return json.Marshal(frame)
}

// recordedDerivativesResponses holds the gateway responses testdata/http.json records, by method
var recordedDerivativesResponses = sync.OnceValues(func() (map[string][]mock.HTTPResponse, error) {
	raw, err := os.ReadFile("testdata/http.json")
	if err != nil {
		return nil, err
	}
	var vcr mock.VCRMock
	if err := json.Unmarshal(raw, &vcr); err != nil {
		return nil, err
	}
	responses := make(map[string][]mock.HTTPResponse)
	for path, methods := range vcr.Routes {
		method, ok := strings.CutPrefix(path, derivativesPath)
		if !ok {
			continue
		}
		responses[method] = append(methods["POST"], methods["GET"]...)
	}
	return responses, nil
})

// recordedDerivativesResponse returns the response testdata/http.json records for a gateway method and params, sent
// over HTTP as a JSON-RPC body or a query string
func recordedDerivativesResponse(method string, params json.RawMessage) (json.RawMessage, error) {
	responses, err := recordedDerivativesResponses()
	if err != nil {
		return nil, err
	}
	want, err := decodeDerivativesParams(params)
	if err != nil {
		return nil, err
	}
	for _, response := range responses[method] {
		recorded := map[string]any{}
		if response.BodyParams != "" {
			var body derivativesVenueRequest
			if err := json.Unmarshal([]byte(response.BodyParams), &body); err != nil {
				return nil, err
			}
			if recorded, err = decodeDerivativesParams(body.Params); err != nil {
				return nil, err
			}
		} else {
			query, err := url.ParseQuery(response.QueryString)
			if err != nil {
				return nil, err
			}
			for k := range query {
				recorded[k] = query.Get(k)
			}
			// A query string holds every param as text, so the request's are compared as text too
			for k, v := range want {
				switch v := v.(type) {
				case string:
				case float64:
					// %v writes a millisecond timestamp in exponent form, which a query string never holds
					want[k] = strconv.FormatFloat(v, 'f', -1, 64)
				default:
					want[k] = fmt.Sprint(v)
				}
			}
		}
		if reflect.DeepEqual(recorded, want) {
			return response.Data, nil
		}
	}
	return nil, fmt.Errorf("%w: no recorded response for %s %s", errDerivativesVenueRequest, method, params)
}

// decodeDerivativesParams decodes params for comparison, treating absent params as empty
func decodeDerivativesParams(params json.RawMessage) (map[string]any, error) {
	decoded := map[string]any{}
	if len(params) == 0 || string(params) == "null" {
		return decoded, nil
	}
	return decoded, json.Unmarshal(params, &decoded)
}

func TestDerivativesVenueRequestIDs(t *testing.T) {
	t.Parallel()
	// The mock gateway decodes every ID a request can carry
	body, err := json.Marshal(&JSONRPCRequest{JSONRPC: jsonRPCVersion, ID: math.MaxUint64, Method: "public/test"})
	require.NoError(t, err, "a request with the largest ID must encode")
	var req derivativesVenueRequest
	require.NoError(t, json.Unmarshal(body, &req), "the mock must decode a request with the largest ID")
	assert.Equal(t, uint64(math.MaxUint64), req.ID, "the mock should decode the request's ID")
	frame, err := derivativesVenueFrame(req.ID, json.RawMessage(`{"result":"ok"}`))
	require.NoError(t, err, "the mock must answer a request with the largest ID")
	var envelope DerivativesEnvelope
	require.NoError(t, json.Unmarshal(frame, &envelope), "the mock's answer must decode")
	assert.Equal(t, uint64(math.MaxUint64), envelope.ID, "the mock should answer with the request's ID")
}

func TestDerivativesVenueAnswer(t *testing.T) {
	t.Parallel()
	// serve starts a mock of the gateway's websocket answering through the venue. Each connection it accepts sends the
	// error answer returns for each request on a channel of its own rather than failing the test, so the refusals can
	// be checked
	serve := func(t *testing.T, venue *derivativesVenue) (dial func() (*gws.Conn, <-chan error)) {
		t.Helper()
		accepted := make(chan chan error, 4)
		server, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handled := make(chan error, 2*derivativesVenueRequestLimit)
			accepted <- handled
			mockws.WsMockUpgrader(t, w, r, func(_ testing.TB, msg []byte, conn *gws.Conn) error {
				handled <- venue.answer(t, msg, conn)
				return nil
			})
		}))
		dialer.HandshakeTimeout = 30 * time.Second
		return func() (*gws.Conn, <-chan error) {
			conn, resp, err := dialer.DialContext(t.Context(), "ws"+strings.TrimPrefix(server.URL, "http"), nil)
			require.NoError(t, err, "DialContext must not error")
			require.NoError(t, resp.Body.Close(), "closing the handshake response's body must not error")
			t.Cleanup(func() { _ = conn.Close() })
			select {
			case handled := <-accepted:
				return conn, handled
			case <-time.After(30 * time.Second):
				require.FailNow(t, "the mock must accept the connection")
				return nil, nil
			}
		}
	}
	send := func(t *testing.T, conn *gws.Conn, id uint64, method string) {
		t.Helper()
		body, err := json.Marshal(&JSONRPCRequest{JSONRPC: jsonRPCVersion, ID: id, Method: method})
		require.NoError(t, err, "the request must encode")
		require.NoError(t, conn.SetWriteDeadline(time.Now().Add(30*time.Second)), "SetWriteDeadline must not error")
		require.NoError(t, conn.WriteMessage(gws.TextMessage, body), "WriteMessage must not error")
	}
	handledError := func(t *testing.T, handled <-chan error) error {
		t.Helper()
		select {
		case err := <-handled:
			return err
		case <-time.After(30 * time.Second):
			require.FailNow(t, "the mock must finish handling the request")
			return nil
		}
	}
	// reply returns the mock's reply to the connection's last request and the error answering it returned
	reply := func(t *testing.T, conn *gws.Conn, handled <-chan error) (*JSONRPCResponse, error) {
		t.Helper()
		require.NoError(t, conn.SetReadDeadline(time.Now().Add(30*time.Second)), "SetReadDeadline must not error")
		_, frame, err := conn.ReadMessage()
		require.NoError(t, err, "ReadMessage must not error")
		var resp *JSONRPCResponse
		require.NoError(t, json.Unmarshal(frame, &resp), "the reply must decode")
		return resp, handledError(t, handled)
	}
	closed := func(t *testing.T, conn *gws.Conn, handled <-chan error) {
		t.Helper()
		require.NoError(t, conn.SetReadDeadline(time.Now().Add(30*time.Second)), "SetReadDeadline must not error")
		_, frame, err := conn.ReadMessage()
		var closeErr *gws.CloseError
		assert.ErrorAsf(t, err, &closeErr, "the mock should close the connection on a logout rather than answer %s", frame)
		assert.NoError(t, handledError(t, handled), "the mock should close the connection on a logout without an error")
	}
	answered := func(id uint64) *JSONRPCResponse {
		return &JSONRPCResponse{JSONRPC: jsonRPCVersion, ID: id, Result: "ok"}
	}
	refused := func(id uint64) *JSONRPCResponse {
		return &JSONRPCResponse{JSONRPC: jsonRPCVersion, ID: id, Error: &DerivativesError{Code: -32601, Message: "no mock response"}}
	}

	t.Run("answers requests up to the limit and refuses the rest but a logout", func(t *testing.T) {
		t.Parallel()
		var lookups atomic.Int64
		venue := new(derivativesVenue)
		venue.respondWith(func(string, []byte) ([]byte, bool) {
			lookups.Add(1)
			return []byte(`{"result":"ok"}`), true
		})
		conn, handled := serve(t, venue)()
		for id := uint64(1); id <= derivativesVenueRequestLimit+2; id++ {
			method := "private/get_order_history_by_currency"
			if id%2 == 0 {
				method = "public/get_time"
			}
			send(t, conn, id, method)
			resp, err := reply(t, conn, handled)
			if id <= derivativesVenueRequestLimit {
				require.NoErrorf(t, err, "the mock must answer request %d", id)
				require.Equalf(t, answered(id), resp, "the mock must answer request %d with its result", id)
				continue
			}
			assert.ErrorIsf(t, err, errDerivativesVenueRequest, "the mock should report request %d past the limit", id)
			assert.Equalf(t, refused(id), resp, "the mock should refuse request %d past the limit", id)
		}
		assert.Equal(t, int64(derivativesVenueRequestLimit), lookups.Load(), "the mock should look up no response for a request past the limit")
		send(t, conn, derivativesVenueRequestLimit+3, "private/logout")
		closed(t, conn, handled)
	})

	t.Run("counts the requests of every connection and method together", func(t *testing.T) {
		t.Parallel()
		entered, release := make(chan struct{}), make(chan struct{})
		releaseOnce := sync.OnceFunc(func() { close(release) })
		var lookups atomic.Int64
		var heldPastDeadline atomic.Bool
		venue := new(derivativesVenue)
		venue.respondWith(func(method string, _ []byte) ([]byte, bool) {
			lookups.Add(1)
			if method == "public/status" {
				close(entered)
				select {
				case <-release:
				case <-time.After(30 * time.Second):
					heldPastDeadline.Store(true)
				}
			}
			return []byte(`{"result":"ok"}`), true
		})
		dial := serve(t, venue)
		first, firstHandled := dial()
		second, secondHandled := dial()
		t.Cleanup(releaseOnce)
		// Each connection numbers its own requests, so the two reuse IDs
		for n := uint64(1); n < derivativesVenueRequestLimit; n++ {
			conn, handled, id, method := first, firstHandled, (n+1)/2, "public/test"
			if n%2 == 0 {
				conn, handled, id, method = second, secondHandled, n/2, "private/get_positions"
			}
			send(t, conn, id, method)
			resp, err := reply(t, conn, handled)
			require.NoErrorf(t, err, "the mock must answer request %d", n)
			require.Equalf(t, answered(id), resp, "the mock must answer request %d with its result", n)
		}
		// The request at the limit is held until one past it on the other connection is refused
		send(t, first, 51, "public/status")
		select {
		case <-entered:
		case <-time.After(30 * time.Second):
			require.FailNow(t, "the mock must start answering the request at the limit")
		}
		send(t, second, 50, "private/get_positions")
		resp, err := reply(t, second, secondHandled)
		assert.ErrorIs(t, err, errDerivativesVenueRequest, "the mock should report a request past the limit on another connection")
		assert.Equal(t, refused(50), resp, "the mock should refuse a request past the limit on another connection")
		releaseOnce()
		resp, err = reply(t, first, firstHandled)
		require.NoError(t, err, "the mock must answer the request at the limit")
		assert.Equal(t, answered(51), resp, "the mock should answer the request at the limit with its result")
		assert.False(t, heldPastDeadline.Load(), "the mock should refuse the request past the limit while the one at it is held")
		send(t, first, 52, "public/test")
		resp, err = reply(t, first, firstHandled)
		assert.ErrorIs(t, err, errDerivativesVenueRequest, "the mock should report a later request past the limit")
		assert.Equal(t, refused(52), resp, "the mock should refuse a later request past the limit")
		assert.Equal(t, int64(derivativesVenueRequestLimit), lookups.Load(), "the mock should look up no response for a request past the limit")
	})

	t.Run("counts a logout toward the limit and refuses requests past it without a responder", func(t *testing.T) {
		t.Parallel()
		dial := serve(t, new(derivativesVenue))
		first, firstHandled := dial()
		for id := uint64(1); id < derivativesVenueRequestLimit; id++ {
			method := "public/unsubscribe_all"
			if id%2 == 0 {
				method = "private/unsubscribe_all"
			}
			send(t, first, id, method)
			resp, err := reply(t, first, firstHandled)
			require.NoErrorf(t, err, "the mock must answer request %d", id)
			require.Equalf(t, answered(id), resp, "the mock must answer request %d with its result", id)
		}
		send(t, first, derivativesVenueRequestLimit, "private/logout")
		closed(t, first, firstHandled)
		second, secondHandled := dial()
		for i, method := range []string{"public/unsubscribe_all", "private/unsubscribe_all"} {
			id := uint64(i + 1)
			send(t, second, id, method)
			resp, err := reply(t, second, secondHandled)
			assert.ErrorIsf(t, err, errDerivativesVenueRequest, "the mock should report a %s request past the limit", method)
			assert.Equalf(t, refused(id), resp, "the mock should refuse a %s request past the limit", method)
		}
	})
}
