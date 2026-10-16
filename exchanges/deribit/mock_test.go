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
	url, venue := startWebsocketVenue(t)
	ex := newTestExchange(t)
	require.NoError(t, ex.Websocket.SetWebsocketURL(url, false, true), "SetWebsocketURL must not error")
	ex.Features.Subscriptions = subscription.List{}
	ex.Websocket.SetSubscriptionsNotRequired()
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
	require.True(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "the mock websocket must authenticate the connection")
	return ex, venue
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

// record keeps a request the mock received
func (v *websocketVenue) record(req *websocketVenueRequest) {
	v.mu.Lock()
	v.requests = append(v.requests, *req)
	v.mu.Unlock()
}

// answer replies to a request as Deribit does: it authenticates a client signature of the test client ID and rejects
// any other, acknowledges every channel of a subscription request and answers any other request with the
// testdata/http.json response recorded for the same method and params; a request without one is answered with an error,
// so the request fails at once, and reported
func (v *websocketVenue) answer(_ testing.TB, msg []byte, conn *gws.Conn) error {
	var req websocketVenueRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return fmt.Errorf("%w: %s: %w", errWebsocketVenueRequest, msg, err)
	}
	v.record(&req)
	if req.Method == "private/logout" {
		return conn.Close()
	}
	result, answerErr := websocketVenueResult(&req)
	if answerErr != nil {
		result = json.RawMessage(`{"error":{"code":-32601,"message":"no mock response"}}`)
	}
	frame, err := websocketVenueFrame(req.ID, result)
	if err != nil {
		return err
	}
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
		var params subscriptionParams
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
