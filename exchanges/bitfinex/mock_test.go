//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package bitfinex

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
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
		log.Fatalf("Bitfinex mock setup error: %s", err)
	}
	os.Exit(m.Run())
}

// setupMockExchange serves an exchange from testdata/http.json, whose public endpoint responses are trimmed from live
// recordings and whose authenticated endpoint responses follow Bitfinex's documented examples, signing with
// placeholder credentials so the authenticated endpoints run against their recorded responses too
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

// newWebsocketTestExchange returns a mock exchange named after the test whose websocket is connected, authenticated, to
// a new mock of Bitfinex's websocket with subs subscribed, or none, and the mock, which records the requests it
// receives
func newWebsocketTestExchange(t *testing.T, subs subscription.List, opts ...websocketVenueOption) (*Exchange, *websocketVenue) {
	t.Helper()
	ex := newTestExchange(t)
	return ex, connectWebsocketTestExchange(t, ex, subs, opts...)
}

// websocketVenueOption sets the mock of Bitfinex's websocket up before an exchange connects to it
type websocketVenueOption func(*websocketVenue)

// withBadChecksum makes a recorded channel's first stream send a checksum its book cannot match
func withBadChecksum(fixtureKey string) websocketVenueOption {
	return func(v *websocketVenue) { v.badChecksums[fixtureKey] = true }
}

// withRefusedAuthNonces makes the mock refuse the nonces of the first n authentications, as Bitfinex does a nonce no
// larger than the key's last
func withRefusedAuthNonces(n int) websocketVenueOption {
	return func(v *websocketVenue) { v.refusedAuthNonces = n }
}

// withRefusedResubscriptions makes the mock refuse the subscriptions to a recorded channel after its first, n times
func withRefusedResubscriptions(fixtureKey string, n int) websocketVenueOption {
	return func(v *websocketVenue) { v.refusedResubscriptions[fixtureKey] = n }
}

// connectWebsocketTestExchange connects an exchange's websocket to a new mock of Bitfinex's websocket, authenticated
// when the exchange supports it, with subs subscribed, or none, returning the mock. The websocket is disabled and shut
// down when the test ends, so its connection monitor does not redial the closed mock
func connectWebsocketTestExchange(t *testing.T, ex *Exchange, subs subscription.List, opts ...websocketVenueOption) *websocketVenue {
	t.Helper()
	venue := &websocketVenue{badChecksums: map[string]bool{}, refusedResubscriptions: map[string]int{}, subscribed: map[string]int{}}
	for _, opt := range opts {
		opt(venue)
	}
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

// websocketVenue mocks Bitfinex's websocket. It confirms conf, auth, subscribe and unsubscribe requests, numbering
// each connection's channels, and follows a subscription with the frames testdata/wsPublic.json recorded for its
// channel. It answers input requests with the notifications and events Bitfinex documents
type websocketVenue struct {
	mu          sync.Mutex
	requests    []json.RawMessage
	nextChannel int64
	nextOrderID uint64
	// badChecksums holds the fixture keys of channels whose next stream sends a checksum their book cannot match
	badChecksums map[string]bool
	// refusedResubscriptions holds how many subscriptions to a channel after its first are refused, by fixture key,
	// and subscribed how many subscriptions to each were requested
	refusedResubscriptions map[string]int
	subscribed             map[string]int
	// refusedAuthNonces is how many authentications' nonces are still to be refused
	refusedAuthNonces int
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

// events returns the object requests with an event, in arrival order
func (v *websocketVenue) events(event string) []map[string]any {
	if v == nil {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	var got []map[string]any
	for _, raw := range v.requests {
		var req map[string]any
		if json.Unmarshal(raw, &req) == nil && req["event"] == event {
			got = append(got, req)
		}
	}
	return got
}

// inputs returns the data of the input requests of a type, in arrival order
func (v *websocketVenue) inputs(inputType string) []json.RawMessage {
	if v == nil {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	var got []json.RawMessage
	for _, raw := range v.requests {
		var req []json.RawMessage
		if json.Unmarshal(raw, &req) != nil || len(req) != 4 {
			continue
		}
		var t string
		if json.Unmarshal(req[1], &t) == nil && t == inputType {
			got = append(got, req[3])
		}
	}
	return got
}

// answer replies to a request as Bitfinex does
func (v *websocketVenue) answer(msg []byte, conn *gws.Conn) error {
	v.mu.Lock()
	v.requests = append(v.requests, bytes.Clone(msg))
	v.mu.Unlock()
	if len(msg) > 0 && msg[0] == '[' {
		return v.answerInput(msg, conn)
	}
	var req map[string]any
	if err := json.Unmarshal(msg, &req); err != nil {
		return fmt.Errorf("%w: %s: %w", errWebsocketVenueRequest, msg, err)
	}
	switch req["event"] {
	case "conf":
		return writeVenueFrame(conn, map[string]any{"event": "conf", "status": "OK", "flags": req["flags"]})
	case "auth":
		v.mu.Lock()
		refuse := v.refusedAuthNonces > 0
		if refuse {
			v.refusedAuthNonces--
		}
		v.mu.Unlock()
		if refuse {
			return writeVenueFrame(conn, map[string]any{"event": "auth", "status": "FAILED", "chanId": 0, "code": wsErrorAuthNonce, "msg": "nonce: small"})
		}
		return writeVenueFrame(conn, venueAuthReply(req))
	case "subscribe":
		return v.subscribe(req, conn)
	case "unsubscribe":
		return writeVenueFrame(conn, map[string]any{"event": "unsubscribed", "status": "OK", "chanId": req["chanId"]})
	case "ping":
		return writeVenueFrame(conn, map[string]any{"event": "pong", "cid": req["cid"], "ts": int64(1792128975841)})
	}
	return fmt.Errorf("%w: %s", errWebsocketVenueRequest, msg)
}

// venueAuthReply authenticates a request signed with the placeholder credentials, and rejects any other
func venueAuthReply(req map[string]any) map[string]any {
	payload, _ := req["authPayload"].(string)
	nonce, _ := req["authNonce"].(string)
	signature, err := crypto.GetHMAC(crypto.HashSHA512_384, []byte(payload), []byte(testAPISecret))
	if err != nil || req["apiKey"] != testAPIKey || payload != "AUTH"+nonce || req["authSig"] != hex.EncodeToString(signature) {
		return map[string]any{"event": "auth", "status": "FAILED", "chanId": 0, "code": 10100, "msg": "apikey: invalid"}
	}
	return map[string]any{"event": "auth", "status": "OK", "chanId": 0, "userId": 269312}
}

// subscribe confirms a subscription with a new channel ID, then streams the frames recorded for its channel
func (v *websocketVenue) subscribe(req map[string]any, conn *gws.Conn) error {
	channel, _ := req["channel"].(string)
	symbol, _ := req["symbol"].(string)
	key, _ := req["key"].(string)
	precision, _ := req["prec"].(string)
	fixtureKey := wsFixtureKey(channel, symbol+key, precision)
	v.mu.Lock()
	v.subscribed[fixtureKey]++
	if v.subscribed[fixtureKey] > 1 && v.refusedResubscriptions[fixtureKey] > 0 {
		v.refusedResubscriptions[fixtureKey]--
		v.mu.Unlock()
		return writeVenueFrame(conn, map[string]any{"event": "error", "msg": "subscribe: failed", "code": 10300, "subId": req["subId"], "pair": symbol})
	}
	v.nextChannel++
	channelID := v.nextChannel
	badChecksum := v.badChecksums[fixtureKey]
	delete(v.badChecksums, fixtureKey)
	v.mu.Unlock()
	reply := map[string]any{"event": "subscribed", "channel": channel, "chanId": channelID, "subId": req["subId"]}
	for _, field := range []string{"symbol", "key", "prec", "freq", "len"} {
		if value, ok := req[field]; ok {
			reply[field] = value
		}
	}
	if err := writeVenueFrame(conn, reply); err != nil {
		return err
	}
	fixture, err := wsPublicFixture()
	if err != nil {
		return err
	}
	for _, frame := range fixture.frames[fixtureKey] {
		frame = rechannelFrame(frame, channelID)
		if badChecksum && bytes.Contains(frame, []byte(`,"cs",`)) {
			frame = []byte(`[` + strconv.FormatInt(channelID, 10) + `,"cs",1,1792128959341]`)
		}
		if err := conn.WriteMessage(gws.TextMessage, frame); err != nil {
			return err
		}
	}
	return nil
}

// wsFixtureKey identifies a recorded channel by its name, symbol or key and any book precision, as the fixtures label
// their subscriptions
func wsFixtureKey(channel, symbol, precision string) string {
	if precision == "" {
		return channel + ":" + symbol
	}
	return channel + ":" + symbol + ":" + precision
}

// rechannelFrame returns a recorded channel frame with its channel ID replaced
func rechannelFrame(frame []byte, channelID int64) []byte {
	i := bytes.IndexByte(frame, ',')
	return append([]byte("["+strconv.FormatInt(channelID, 10)), frame[i:]...)
}

// answerInput answers an input request, [0, TYPE, null, DATA], with the notification Bitfinex documents for it and,
// for an accepted order or offer request, the event that follows
func (v *websocketVenue) answerInput(msg []byte, conn *gws.Conn) error {
	var input []json.RawMessage
	if err := json.Unmarshal(msg, &input); err != nil || len(input) != 4 {
		return fmt.Errorf("%w: %s", errWebsocketVenueRequest, msg)
	}
	var inputType string
	if err := json.Unmarshal(input[1], &inputType); err != nil {
		return err
	}
	var data map[string]any
	if inputType != wsCalc {
		if err := json.Unmarshal(input[3], &data); err != nil {
			return err
		}
	}
	switch inputType {
	case wsOrderNew:
		v.mu.Lock()
		v.nextOrderID++
		id := 1185815097 + v.nextOrderID
		v.mu.Unlock()
		order := venueOrder(id, data, "ACTIVE")
		if data["symbol"] == "tNOPEUSD" {
			order[0] = nil
			return writeVenueFrame(conn, venueNotification(wsOrderNewRequest, order, "ERROR", "Invalid order: symbol"))
		}
		if err := writeVenueFrame(conn, venueNotification(wsOrderNewRequest, order, "SUCCESS", "Submitting limit order.")); err != nil {
			return err
		}
		return writeVenueFrame(conn, []any{0, wsOrderNew, order})
	case wsOrderUpdate:
		order := venueOrder(venueID(data), data, "ACTIVE")
		if err := writeVenueFrame(conn, venueNotification(wsOrderUpdateRequest, order, "SUCCESS", "Submitting update to limit order.")); err != nil {
			return err
		}
		return writeVenueFrame(conn, []any{0, wsOrderUpdate, order})
	case wsOrderCancel:
		order := venueOrder(venueID(data), map[string]any{"symbol": "tETHUSD", "amount": "-3", "price": "240", "type": "LIMIT"}, "ACTIVE")
		if err := writeVenueFrame(conn, venueNotification(wsOrderCancelRequest, order, "SUCCESS", "Submitted for cancellation; waiting for confirmation.")); err != nil {
			return err
		}
		order[13] = "CANCELED"
		return writeVenueFrame(conn, []any{0, wsOrderCancel, order})
	case wsOrderCancelMulti:
		return writeVenueFrame(conn, venueNotification(wsOrderCancelMultiRequest, []any{}, "SUCCESS", "Submitting order cancellations."))
	case wsFundingOfferNew:
		offer := make([]any, 21)
		offer[0], offer[4], offer[14], offer[15] = 41238934, venueFloat(data["amount"]), venueFloat(data["rate"]), data["period"]
		return writeVenueFrame(conn, venueNotification(wsFundingOfferNewRequest, offer, "SUCCESS", "Submitting funding offer."))
	case wsFundingOfferCancel:
		offer := make([]any, 21)
		offer[0], offer[1], offer[10] = venueID(data), "fUST", "ACTIVE"
		return writeVenueFrame(conn, venueNotification(wsFundingOfferCancelRequest, offer, "SUCCESS", "Submitted for cancellation; waiting for confirmation."))
	case wsCalc:
		return nil
	}
	return fmt.Errorf("%w: %s", errWebsocketVenueRequest, msg)
}

// venueOrder returns an order array for an order request's fields
func venueOrder(id uint64, data map[string]any, status string) []any {
	order := make([]any, 32)
	order[0], order[2], order[3], order[4], order[5] = id, data["cid"], data["symbol"], int64(1575287438515), int64(1575287438519)
	order[6], order[7], order[8], order[13] = venueFloat(data["amount"]), venueFloat(data["amount"]), data["type"], status
	order[16], order[28] = venueFloat(data["price"]), "API>BFX"
	return order
}

// venueNotification returns a notification frame
func venueNotification(notificationType string, data any, status, text string) []any {
	return []any{0, "n", []any{int64(1575287438515), notificationType, nil, nil, data, nil, status, text}}
}

// venueID returns a request's ID
func venueID(data map[string]any) uint64 {
	id, _ := data["id"].(float64)
	return uint64(id)
}

// venueFloat returns a decimal string request field as a number
func venueFloat(v any) float64 {
	s, _ := v.(string)
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// writeVenueFrame writes a frame to a mock connection
func writeVenueFrame(conn *gws.Conn, frame any) error {
	b, err := json.Marshal(frame)
	if err != nil {
		return err
	}
	return conn.WriteMessage(gws.TextMessage, b)
}

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
