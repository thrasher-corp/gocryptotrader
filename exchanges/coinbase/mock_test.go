//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package coinbase

import (
	"errors"
	"fmt"
	"log"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
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
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must connect to the mock gateway websocket")
	t.Cleanup(func() { stopWebsocketTestExchange(t, ex) })
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
	ID     int64           `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// requestsFor returns the requests the mock received for a method
func (v *derivativesVenue) requestsFor(method string) []derivativesVenueRequest {
	v.mu.Lock()
	defer v.mu.Unlock()
	return slices.DeleteFunc(slices.Clone(v.requests), func(r derivativesVenueRequest) bool { return r.Method != method })
}

// answer replies to a request as the gateway does; a request without a recorded response is answered with an error,
// so it fails at once, and reported
func (v *derivativesVenue) answer(_ testing.TB, msg []byte, conn *gws.Conn) error {
	var req derivativesVenueRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return fmt.Errorf("%w: %s: %w", errDerivativesVenueRequest, msg, err)
	}
	v.mu.Lock()
	v.requests = append(v.requests, req)
	v.mu.Unlock()
	if req.Method == "private/logout" {
		return conn.Close()
	}
	result, answerErr := v.result(&req)
	if answerErr != nil {
		result = json.RawMessage(`{"error":{"code":-32601,"message":"no mock response"}}`)
	}
	frame, err := derivativesVenueFrame(req.ID, result)
	if err != nil {
		return err
	}
	if err := conn.WriteMessage(gws.TextMessage, frame); err != nil {
		return err
	}
	return answerErr
}

// result returns the response body answering a request, holding its result or error
func (v *derivativesVenue) result(req *derivativesVenueRequest) (json.RawMessage, error) {
	v.mu.Lock()
	body, ok := v.overrides[req.Method]
	v.mu.Unlock()
	if ok {
		return body, nil
	}
	switch req.Method {
	case derivativesAuthMethod:
		var params derivativesAuthParams
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
func derivativesVenueFrame(id int64, body json.RawMessage) ([]byte, error) {
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
