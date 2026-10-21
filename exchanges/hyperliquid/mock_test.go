//go:build !mock_test_off

// This will build unless build tag mock_test_off is parsed and will do mock testing
// using all tests in (exchange)_test.go
package hyperliquid

import (
	"errors"
	"fmt"
	"log"
	"net/http/httptest"
	"os"
	"reflect"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
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

// setupMockExchange serves an exchange from testdata/http.json, signing with the Python SDK's public test key: its info
// responses are trimmed from live recordings, and its exchange endpoint responses follow the documented shapes, as
// Hyperliquid has not been sent that key's actions
func setupMockExchange(ex *Exchange) error {
	if err := testexch.Setup(ex); err != nil {
		return err
	}
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: testAccountAddress, Secret: testPrivateKey})
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

// newUserSignedTestExchange returns a mock exchange whose next nonce is the one the user-signed action fixtures carry,
// mockUserSignedNonce, a synthesised time in the year 2100
func newUserSignedTestExchange(t *testing.T) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, setupMockExchange(ex), "setupMockExchange must not error")
	ex.lastNonce.Store(mockUserSignedNonce - 1)
	return ex
}

// websocketPostFixture is an answer to a websocket post: a post response, or, for a request Hyperliquid cannot parse, an
// error message quoting the request. Info answers and the parse failure are live recordings, trimmed; action answers are
// the exchange endpoint fixtures of the same actions in the documented post envelope, as Hyperliquid has not been sent
// a signed action over the websocket
type websocketPostFixture struct {
	Request      websocketVenuePostBody `json:"request"`
	Response     json.RawMessage        `json:"response"`
	ParseFailure bool                   `json:"parseFailure"`
}

// loadWebsocketPostFixtures reads the post answers in testdata/websocket_post.json
var loadWebsocketPostFixtures = sync.OnceValues(func() ([]websocketPostFixture, error) {
	raw, err := os.ReadFile("testdata/websocket_post.json")
	if err != nil {
		return nil, err
	}
	var fixtures []websocketPostFixture
	return fixtures, json.Unmarshal(raw, &fixtures)
})

// newVenueWebsocketTestExchange returns a mock exchange named after the test whose websocket connections reach a new
// mock of Hyperliquid's websocket, and the requests the mock receives; nothing is subscribed, so only the public
// connection, which posts, connects until a test subscribes an account feed
func newVenueWebsocketTestExchange(t *testing.T) (*Exchange, *websocketVenueRequests) {
	t.Helper()
	url, venue := startVenueWebsocketMock(t)
	ex := newTestExchange(t)
	require.NoError(t, ex.Websocket.SetAllConnectionURLs(url), "SetAllConnectionURLs must not error")
	ex.Features.Subscriptions = subscription.List{}
	require.NoError(t, ex.Websocket.Enable(t.Context()), "Enable must connect to the mock websocket")
	t.Cleanup(func() {
		// Disabling stops the connection monitor, which would otherwise redial the mock after the test, and which may then
		// shut the connection down first
		assert.NoError(t, ex.Websocket.Disable(), "Disable should not error")
		if err := shutdownWebsocketWithin(t, ex); err != nil {
			assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should only fail when the connection monitor has shut the connection down first")
		}
	})
	return ex, venue
}

// venueWebsocketURL returns the URL of a new mock of Hyperliquid's websocket
func venueWebsocketURL(t *testing.T) string {
	t.Helper()
	url, _ := startVenueWebsocketMock(t)
	return url
}

// startVenueWebsocketMock starts a mock of Hyperliquid's websocket and returns its URL and the requests it receives; it
// listens on loopback, as the exchange dials with a plain dialer that cannot reach the in-memory network
func startVenueWebsocketMock(t *testing.T) (string, *websocketVenueRequests) {
	t.Helper()
	venue := new(websocketVenueRequests)
	server := httptest.NewTestServer(t, mockws.CurryWsMockUpgrader(t, venue.answer))
	server.Start()
	return "ws" + strings.TrimPrefix(server.URL, "http"), venue
}

// answer replies to a request as Hyperliquid does: a pong to a ping, the canonical acknowledgement to a subscription
// request and the fixture answer to a post; it records every request but pings, which the connections send on a timer
func (v *websocketVenueRequests) answer(_ testing.TB, msg []byte, conn *gws.Conn) error {
	var req websocketVenueRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return fmt.Errorf("%w: %s: %w", errWebsocketVenueRequest, msg, err)
	}
	if req.Method == "ping" {
		return conn.WriteMessage(gws.TextMessage, []byte(`{"channel":"pong"}`))
	}
	v.record(msg)
	switch req.Method {
	case wsMethodSubscribe, wsMethodUnsubscribe:
		for _, frame := range acknowledge(msg) {
			if err := conn.WriteMessage(gws.TextMessage, []byte(frame)); err != nil {
				return err
			}
		}
		return nil
	case wsMethodPost:
		frame, answerErr := websocketPostAnswer(msg, &req)
		if err := conn.WriteMessage(gws.TextMessage, []byte(frame)); err != nil {
			return err
		}
		return answerErr
	default:
		return fmt.Errorf("%w: %s", errWebsocketVenueRequest, msg)
	}
}

// websocketPostAnswer returns the frame answering a post with its fixture answer; a post without one is answered with an
// error frame, so the request fails at once, and reported
func websocketPostAnswer(msg []byte, req *websocketVenueRequest) (string, error) {
	id := strconv.FormatUint(req.ID, 10)
	unanswered := `{"channel":"post","data":{"id":` + id + `,"response":{"type":"error","payload":"no fixture answer"}}}`
	fixtures, err := loadWebsocketPostFixtures()
	if err != nil {
		return unanswered, err
	}
	payload, err := websocketVenuePayload(&req.Request)
	if err != nil {
		return unanswered, err
	}
	for i := range fixtures {
		var answered map[string]any
		if err := json.Unmarshal(fixtures[i].Request.Payload, &answered); err != nil {
			return unanswered, err
		}
		if fixtures[i].Request.Type != req.Request.Type || !reflect.DeepEqual(answered, payload) {
			continue
		}
		if fixtures[i].ParseFailure {
			quoted, err := json.Marshal(wsErrorParseFailure + string(msg))
			return `{"channel":"error","data":` + string(quoted) + `}`, err
		}
		return `{"channel":"post","data":{"id":` + id + `,"response":` + string(fixtures[i].Response) + `}}`, nil
	}
	return unanswered, fmt.Errorf("%w: no fixture answer to %s", errWebsocketVenueRequest, msg)
}

// websocketVenuePayload decodes a post's payload to compare with the fixture payloads; a signed action's nonce and
// signature change with every request, so they are checked for presence and left out
func websocketVenuePayload(body *websocketVenuePostBody) (map[string]any, error) {
	var payload map[string]any
	if err := json.Unmarshal(body.Payload, &payload); err != nil {
		return nil, err
	}
	if body.Type != wsPostTypeAction {
		return payload, nil
	}
	nonce, _ := payload["nonce"].(float64)
	signature, _ := payload["signature"].(map[string]any)
	r, _ := signature["r"].(string)
	s, _ := signature["s"].(string)
	if nonce <= 0 || r == "" || s == "" || signature["v"] == nil {
		return nil, fmt.Errorf("%w: action without a nonce and signature", errWebsocketVenueRequest)
	}
	delete(payload, "nonce")
	delete(payload, "signature")
	return payload, nil
}
