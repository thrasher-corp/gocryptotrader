//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package lbank

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
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
)

var mockTests = true

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := setupMockExchange(e); err != nil {
		log.Fatalf("Lbank mock setup error: %s", err)
	}
	os.Exit(m.Run())
}

// setupMockExchange serves an exchange from testdata/http.json, whose public endpoint responses are live recordings
// and whose authenticated endpoint responses follow LBank's documented examples, signing with placeholder credentials
// so the authenticated endpoints run against their recorded responses too
func setupMockExchange(ex *Exchange) error {
	if err := setupTestExchange(ex); err != nil {
		return err
	}
	setTestCredentials(ex, &accounts.Credentials{Key: testAPIKey, Secret: testAPISecret})
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
	require.NoError(tb, setupMockExchange(ex), "setupMockExchange must not error")
	ex.Name = tb.Name()
	return ex
}

var errWebsocketVenueRequest = errors.New("unexpected request to the mock websocket")

// unknownSubscribeKeyFrame is LBank's refusal of an account channel subscription with a subscribe key it doesn't know,
// recorded live
const unknownSubscribeKeyFrame = `{"SERVER":"V2","message":"Not existing or expired ['subscribeKey']","status":"error","TS":"2026-10-25T11:46:39.507"}`

// newWebsocketTestExchange returns a mock exchange named after the test whose websocket is connected to a new mock of
// LBank's websocket, with its account connection, and subs subscribed, or none, and the mock, which records the
// requests it receives
func newWebsocketTestExchange(t *testing.T, subs subscription.List) (*Exchange, *websocketVenue) {
	t.Helper()
	ex := newTestExchange(t)
	return ex, connectWebsocketTestExchange(t, ex, subs)
}

// connectWebsocketTestExchange connects an exchange's websocket to a new mock of LBank's websocket, with its account
// connection when the exchange supports authentication, and subs subscribed, or none, returning the mock. The websocket
// is disabled and shut down when the test ends, so its connection monitor does not redial the closed mock
func connectWebsocketTestExchange(t *testing.T, ex *Exchange, subs subscription.List) *websocketVenue {
	t.Helper()
	venue := new(websocketVenue)
	server := httptest.NewTestServer(t, venue.handler(t))
	// The exchange dials with its own dialer, which reaches a listener but not the in-memory network
	server.Start()
	// Cleanups run in reverse: the mock stops reporting errors, the exchange shuts down, then the mock closes
	t.Cleanup(venue.close)
	require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(server.URL, "http")), "SetAllConnectionURLs must not error")
	ex.Features.Subscriptions = subs
	if len(subs) == 0 {
		ex.Websocket.SetSubscriptionsNotRequired()
	}
	ex.Websocket.SetCanUseAuthenticatedEndpoints(ex.API.AuthenticatedWebsocketSupport)
	startRelayRecorder(t, ex)
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	t.Cleanup(func() { stopWebsocketTestExchange(t, ex) })
	t.Cleanup(venue.closing)
	return venue
}

// websocketVenue mocks LBank's websocket. It answers pings, and follows a subscription with the frames
// testdata/wsPublic.json and testdata/wsAccount.json recorded for its channel, or with the refusal LBank sends for a
// pair it doesn't list or a subscribe key it doesn't know. LBank confirms no subscription or unsubscription
type websocketVenue struct {
	mu       sync.Mutex
	requests []websocketSubscribeRequest
	// conns are the connections the mock serves, and handlers its goroutines serving them, which outlive the test
	// server, as it does not track the connections it upgrades
	conns    []*gws.Conn
	closed   bool
	handlers sync.WaitGroup
}

var venueUpgrader = gws.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

// handler upgrades each request to a websocket and answers its requests until the client or close ends it. An error
// answering is reported only while the mock is open, as the client going away ends the frames being written
func (v *websocketVenue) handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		v.mu.Lock()
		if v.closed {
			v.mu.Unlock()
			http.Error(w, "mock closed", http.StatusServiceUnavailable)
			return
		}
		v.handlers.Add(1)
		v.mu.Unlock()
		defer v.handlers.Done()
		conn, err := venueUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		v.mu.Lock()
		v.conns = append(v.conns, conn)
		v.mu.Unlock()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if err := v.answer(msg, conn); err != nil {
				v.mu.Lock()
				closed := v.closed
				v.mu.Unlock()
				if !closed {
					assert.NoError(t, err, "the mock websocket should answer each request")
				}
			}
		}
	}
}

// closing stops the mock reporting errors writing to clients, which are going away
func (v *websocketVenue) closing() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.closed = true
}

// close closes the mock's connections and waits for its goroutines to end
func (v *websocketVenue) close() {
	v.mu.Lock()
	v.closed = true
	for _, conn := range v.conns {
		_ = conn.Close()
	}
	v.mu.Unlock()
	v.handlers.Wait()
}

// received returns the requests of an action, such as subscribe, in arrival order
func (v *websocketVenue) received(action string) []websocketSubscribeRequest {
	if v == nil {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	var got []websocketSubscribeRequest
	for _, req := range v.requests {
		if req.Action == action {
			got = append(got, req)
		}
	}
	return got
}

// answer replies to a request as LBank does
func (v *websocketVenue) answer(msg []byte, conn *gws.Conn) error {
	var req websocketSubscribeRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return fmt.Errorf("%w: %s: %w", errWebsocketVenueRequest, msg, err)
	}
	switch req.Action {
	case "ping":
		var ping websocketHeartbeatRequest
		if err := json.Unmarshal(msg, &ping); err != nil {
			return err
		}
		return conn.WriteJSON(&websocketHeartbeatRequest{Action: "pong", Pong: ping.Ping})
	case "subscribe", "unsubscribe":
		v.mu.Lock()
		v.requests = append(v.requests, req)
		v.mu.Unlock()
		if req.Action == "subscribe" {
			return v.subscribe(&req, conn)
		}
		return nil
	}
	return fmt.Errorf("%w: %s", errWebsocketVenueRequest, msg)
}

// subscribe streams the frames recorded for a subscription's channel, or LBank's refusal of a pair it doesn't list or
// a subscribe key it doesn't know
func (v *websocketVenue) subscribe(req *websocketSubscribeRequest, conn *gws.Conn) error {
	if slices.Contains(accountChannels, req.Subscribe) && req.SubscribeKey != testSubscribeKey {
		return conn.WriteMessage(gws.TextMessage, []byte(unknownSubscribeKeyFrame))
	}
	if req.Pair == "notapair_usdt" {
		return conn.WriteMessage(gws.TextMessage, []byte(liveInvalidPairFrame))
	}
	frames, err := recordedFrames()
	if err != nil {
		return err
	}
	for _, frame := range frames[req.Subscribe] {
		if err := conn.WriteMessage(gws.TextMessage, frame); err != nil {
			return err
		}
	}
	return nil
}

// recordedFrames holds the frames testdata/wsPublic.json and testdata/wsAccount.json recorded, by the channel type they
// carry
var recordedFrames = sync.OnceValues(func() (map[string][][]byte, error) {
	frames := make(map[string][][]byte)
	for _, path := range []string{"testdata/wsPublic.json", "testdata/wsAccount.json"} {
		recorded, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		for line := range bytes.Lines(recorded) {
			var frame struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(line, &frame); err != nil {
				return nil, err
			}
			if frame.Type != "" {
				frames[frame.Type] = append(frames[frame.Type], bytes.TrimSpace(line))
			}
		}
	}
	return frames, nil
})

// stopWebsocketTestExchange disables an exchange's websocket, so its connection monitor stops, then shuts it down
func stopWebsocketTestExchange(t *testing.T, ex *Exchange) {
	t.Helper()
	if ex.Websocket.IsEnabled() {
		assert.NoError(t, ex.Websocket.Disable(), "Disable should not error")
	}
	if err := ex.Websocket.Shutdown(); err != nil {
		assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should only fail when the connection monitor has shut the connection down first")
	}
}
