//go:build !mock_test_off

// This will build unless build tag mock_test_off is parsed and will do mock testing
// using all tests in (exchange)_test.go
package deribit

import (
	"errors"
	"fmt"
	"log"
	"net/http/httptest"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mock"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

const (
	// mockAccessToken is the access token the mock websocket authenticates a connection with
	mockAccessToken = "mock-access-token"
)

var errWebsocketVenueRequest = errors.New("unexpected request to the mock websocket")

var mockTests = true

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := setupMockExchange(e); err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

// setupMockExchange serves an exchange from testdata/http.json, whose public method responses are trimmed from live
// recordings and whose private method responses follow Deribit's documented examples, as no account has been recorded
func setupMockExchange(ex *Exchange) error {
	if err := setupTestExchange(ex); err != nil {
		return err
	}
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: testClientID, Secret: testClientSecret})
	if err := testexch.MockHTTPInstance(ex); err != nil {
		return err
	}
	// Mock responses are served locally, so throttling protects nothing
	return ex.DisableRateLimiter()
}

// newTestExchange returns a mock exchange named after the test, so the ticker and orderbook stores it writes do not
// collide with other tests'
func newTestExchange(tb testing.TB) *Exchange {
	tb.Helper()
	ex := new(Exchange)
	require.NoError(tb, setupMockExchange(ex), "setupMockExchange must not error")
	ex.Name = tb.Name()
	return ex
}

// newWebsocketTestExchange returns a mock exchange named after the test whose websocket is connected to a new mock of
// Deribit's websocket and authenticated, and the requests the mock receives
func newWebsocketTestExchange(t *testing.T) (*Exchange, *websocketVenue) {
	t.Helper()
	ex, venue := connectWebsocketTestExchange(t, true)
	require.True(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "the mock websocket must authenticate the connection")
	return ex, venue
}

// newUnauthenticatedWebsocketTestExchange returns a mock exchange named after the test whose websocket is connected to
// a new mock of Deribit's websocket without authenticated websocket support, so it does not authenticate, and the
// requests the mock receives
func newUnauthenticatedWebsocketTestExchange(t *testing.T) (*Exchange, *websocketVenue) {
	t.Helper()
	ex, venue := connectWebsocketTestExchange(t, false)
	require.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "the connection must not be authenticated")
	return ex, venue
}

// connectWebsocketTestExchange returns a mock exchange named after the test whose websocket is connected to a new mock
// of Deribit's websocket, with authenticated websocket support if authenticated is set, so it authenticates when it
// connects, and the requests the mock receives
func connectWebsocketTestExchange(t *testing.T, authenticated bool) (*Exchange, *websocketVenue) {
	t.Helper()
	url, venue := startWebsocketVenue(t)
	ex := newTestExchange(t)
	require.NoError(t, ex.Websocket.SetWebsocketURL(url, false, true), "SetWebsocketURL must not error")
	ex.Features.Subscriptions = subscription.List{}
	ex.Websocket.SetSubscriptionsNotRequired()
	ex.API.AuthenticatedWebsocketSupport = authenticated
	ex.Websocket.SetCanUseAuthenticatedEndpoints(authenticated)
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must connect to the mock websocket")
	t.Cleanup(func() {
		// Disabling stops the connection monitor, which would otherwise redial the mock after the test, and which may then
		// shut the connection down first
		assert.NoError(t, ex.Websocket.Disable(), "Disable should not error")
		if err := ex.Websocket.Shutdown(); err != nil {
			assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should only fail when the connection monitor has shut the connection down first")
		}
	})
	return ex, venue
}

// websocketVenue is a mock of Deribit's websocket, which answers requests as Deribit does and records them
type websocketVenue struct {
	mu       sync.Mutex
	requests []websocketVenueRequest
	// refusals counts the requests of each subscription method still to be refused
	refusals map[string]int
	// delays holds how long the mock waits to answer the next request of a method, answering others meanwhile
	delays map[string]time.Duration
	// gates holds the gate the mock waits on before answering the next request of a method, answering others meanwhile
	gates map[string]chan struct{}
	// notices holds the notification the mock sends ahead of its answer to the next request of a method
	notices map[string][]byte
	// writeMu serialises the mock's answers, which a delayed answer writes from its own goroutine
	writeMu sync.Mutex
}

// delayNext has the mock answer the next request of a method only after a delay, answering any request it receives
// meanwhile first
func (v *websocketVenue) delayNext(method string, delay time.Duration) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.delays == nil {
		v.delays = make(map[string]time.Duration)
	}
	v.delays[method] = delay
}

// holdNext has the mock answer the next request of a method only once the returned function is called, answering other
// requests meanwhile as usual; the test's cleanup calls it too, so an answer held is never left waiting
func (v *websocketVenue) holdNext(t *testing.T, method string) func() {
	t.Helper()
	gate := make(chan struct{})
	release := sync.OnceFunc(func() { close(gate) })
	t.Cleanup(release)
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.gates == nil {
		v.gates = make(map[string]chan struct{})
	}
	v.gates[method] = gate
	return release
}

// notifyBeforeNext has the mock send a notification ahead of its answer to the next request of a method, as Deribit
// sends one it had sent already before answering
func (v *websocketVenue) notifyBeforeNext(method string, notice []byte) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.notices == nil {
		v.notices = make(map[string][]byte)
	}
	v.notices[method] = notice
}

// refuse has the mock refuse the next count requests of a subscription method, acknowledging none of their channels as
// Deribit acknowledges no channel it does not subscribe or unsubscribe
func (v *websocketVenue) refuse(method string, count int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.refusals == nil {
		v.refusals = make(map[string]int)
	}
	v.refusals[method] = count
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

// startWebsocketVenue starts a mock of Deribit's websocket and returns its URL and the requests it receives; it listens
// on loopback, as the exchange dials with a plain dialer that cannot reach the in-memory network
func startWebsocketVenue(t *testing.T) (string, *websocketVenue) {
	t.Helper()
	venue := new(websocketVenue)
	server := httptest.NewTestServer(t, mockws.CurryWsMockUpgrader(t, venue.answer))
	server.Start()
	return "ws" + strings.TrimPrefix(server.URL, "http"), venue
}

// record keeps a request the mock received, and returns whether to refuse it, how long to wait to answer it, the gate
// to wait on before answering it and the notification to send ahead of the answer
func (v *websocketVenue) record(req *websocketVenueRequest) (refused bool, delay time.Duration, hold <-chan struct{}, notice []byte) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.requests = append(v.requests, *req)
	delay = v.delays[req.Method]
	delete(v.delays, req.Method)
	if gate, ok := v.gates[req.Method]; ok {
		hold = gate
		delete(v.gates, req.Method)
	}
	notice = v.notices[req.Method]
	delete(v.notices, req.Method)
	if v.refusals[req.Method] > 0 {
		v.refusals[req.Method]--
		return true, delay, hold, notice
	}
	return false, delay, hold, notice
}

// answer replies to a request as Deribit does: it authenticates a client signature of the test client ID and rejects
// any other, acknowledges every channel of a subscription request unless told to refuse it and answers any other
// request with the testdata/http.json response recorded for the same method and params; a request without one is
// answered with an error, so the request fails at once, and reported
func (v *websocketVenue) answer(_ testing.TB, msg []byte, conn *gws.Conn) error {
	var req websocketVenueRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return fmt.Errorf("%w: %s: %w", errWebsocketVenueRequest, msg, err)
	}
	refused, delay, hold, notice := v.record(&req)
	if req.Method == "private/logout" {
		return conn.Close()
	}
	if notice != nil {
		v.writeMu.Lock()
		err := conn.WriteMessage(gws.TextMessage, notice)
		v.writeMu.Unlock()
		if err != nil {
			return err
		}
	}
	result, answerErr := websocketVenueResult(&req)
	switch {
	case answerErr != nil:
		result = json.RawMessage(`{"error":{"code":-32601,"message":"no mock response"}}`)
	case refused:
		result = json.RawMessage(`{"result":[]}`)
	}
	frame, err := websocketVenueFrame(req.ID, result)
	if err != nil {
		return err
	}
	if delay > 0 || hold != nil {
		go func() {
			if hold != nil {
				<-hold
			}
			time.Sleep(delay)
			v.writeMu.Lock()
			defer v.writeMu.Unlock()
			_ = conn.WriteMessage(gws.TextMessage, frame) // The connection may have closed meanwhile
		}()
		return answerErr
	}
	v.writeMu.Lock()
	defer v.writeMu.Unlock()
	if err := conn.WriteMessage(gws.TextMessage, frame); err != nil {
		return err
	}
	return answerErr
}

// websocketVenueResult returns the response body answering a request, holding its result or error
func websocketVenueResult(req *websocketVenueRequest) (json.RawMessage, error) {
	switch req.Method {
	case "public/auth":
		var params AuthenticateRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, err
		}
		if params.GrantType != GrantTypeClientSignature {
			break
		}
		if params.ClientID != testClientID || params.Signature == "" {
			return json.RawMessage(`{"error":{"code":13004,"message":"invalid_credentials"}}`), nil
		}
		return json.RawMessage(`{"result":{"access_token":"` + mockAccessToken + `","expires_in":31536000,"refresh_token":"mock-refresh-token","scope":"connection mainaccount","enabled_features":[],"token_type":"bearer"}}`), nil
	case "public/subscribe", "private/subscribe", "public/unsubscribe", "private/unsubscribe":
		var params SubscriptionRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, err
		}
		// Deribit leaves out of its acknowledgement a channel it does not subscribe, which tests name as such
		params.Channels = slices.DeleteFunc(params.Channels, func(channel string) bool {
			return strings.Contains(channel, unacknowledgedChannel)
		})
		channels, err := json.Marshal(params.Channels)
		if err != nil {
			return nil, err
		}
		return json.RawMessage(`{"result":` + string(channels) + `}`), nil
	}
	return recordedResponse(req.Method, req.Params)
}

// websocketVenueFrame returns the frame answering a request with its ID and a response body's result or error
func websocketVenueFrame(id int64, body json.RawMessage) ([]byte, error) {
	var response map[string]json.RawMessage
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	frame := map[string]json.RawMessage{"jsonrpc": json.RawMessage(`"2.0"`)}
	if id != 0 {
		frame["id"] = json.RawMessage(strconv.FormatInt(id, 10))
	}
	for _, field := range []string{"result", "error"} {
		if value, ok := response[field]; ok {
			frame[field] = value
		}
	}
	return json.Marshal(frame)
}

// recordedResponses holds the recorded responses of testdata/http.json by method
var recordedResponses = sync.OnceValues(func() (map[string][]mock.HTTPResponse, error) {
	raw, err := os.ReadFile("testdata/http.json")
	if err != nil {
		return nil, err
	}
	var vcr mock.VCRMock
	if err := json.Unmarshal(raw, &vcr); err != nil {
		return nil, err
	}
	responses := make(map[string][]mock.HTTPResponse, len(vcr.Routes))
	for path, methods := range vcr.Routes {
		responses[strings.TrimPrefix(path, apiPath)] = methods["POST"]
	}
	return responses, nil
})

// recordedResponse returns the response testdata/http.json records for a method and params
func recordedResponse(method string, params json.RawMessage) (json.RawMessage, error) {
	responses, err := recordedResponses()
	if err != nil {
		return nil, err
	}
	want, err := decodeParams(params)
	if err != nil {
		return nil, err
	}
	for _, response := range responses[method] {
		var body websocketVenueRequest
		if err := json.Unmarshal([]byte(response.BodyParams), &body); err != nil {
			return nil, err
		}
		recorded, err := decodeParams(body.Params)
		if err != nil {
			return nil, err
		}
		if reflect.DeepEqual(recorded, want) {
			return response.Data, nil
		}
	}
	return nil, fmt.Errorf("%w: no recorded response for %s %s", errWebsocketVenueRequest, method, params)
}

// decodeParams decodes params for comparison, treating absent params as empty
func decodeParams(params json.RawMessage) (map[string]any, error) {
	decoded := map[string]any{}
	if len(params) == 0 || string(params) == "null" {
		return decoded, nil
	}
	return decoded, json.Unmarshal(params, &decoded)
}
