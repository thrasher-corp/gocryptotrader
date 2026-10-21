package gateio

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

// wsAPIFixture is a WebSocket API channel's response to the request parameters that select it
type wsAPIFixture struct {
	Params json.RawMessage `json:"params"`
	Ack    bool            `json:"ack"`
	Result json.RawMessage `json:"result"`
	Errs   json.RawMessage `json:"errs"`
}

var loadWsAPIFixtures = sync.OnceValues(func() (map[string][]wsAPIFixture, error) {
	raw, err := os.ReadFile("testdata/wsAPI.json")
	if err != nil {
		return nil, err
	}
	var fixtures map[string][]wsAPIFixture
	return fixtures, json.Unmarshal(raw, &fixtures)
})

// wsAPIRequest is a WebSocket request as Gate receives it. Subscriptions carry a list as their payload, so the payload
// is only decoded for api requests
type wsAPIRequest struct {
	Time       int64            `json:"time"`
	ID         int64            `json:"id"`
	Channel    string           `json:"channel"`
	Event      string           `json:"event"`
	RawPayload json.RawMessage  `json:"payload"`
	Payload    WebsocketPayload `json:"-"`
}

// wsAPIMock serves Gate's WebSocket API from testdata/wsAPI.json. It accepts subscriptions, checks each login is signed
// with the placeholder credentials, and answers each api request with the fixture whose parameters equal the request's,
// preceded by an ack where Gate sends one
func wsAPIMock(tb testing.TB, msg []byte, w *gws.Conn) error {
	tb.Helper()
	var req wsAPIRequest
	require.NoError(tb, json.Unmarshal(msg, &req), "request must decode")
	switch req.Event {
	case subscribeEvent, unsubscribeEvent:
		return w.WriteJSON(map[string]any{"time": req.Time, "id": req.ID, "channel": req.Channel, "event": req.Event, "result": map[string]string{"status": "success"}})
	case "api":
		require.NoError(tb, json.Unmarshal(req.RawPayload, &req.Payload), "api request payload must decode")
	default:
		return nil // Pings need no reply
	}
	if strings.HasSuffix(req.Channel, ".login") {
		mac := hmac.New(sha512.New, []byte("mock-api-secret"))
		_, err := mac.Write([]byte("api\n" + req.Channel + "\n\n" + req.Payload.Timestamp))
		require.NoError(tb, err, "mac.Write must not error")
		assert.Equalf(tb, hex.EncodeToString(mac.Sum(nil)), req.Payload.Signature, "%s should be signed with the API secret", req.Channel)
		assert.Equalf(tb, "mock-api-key", req.Payload.APIKey, "%s should send the API key", req.Channel)
		return writeWsAPIResponse(w, &req, false, json.RawMessage(`{"api_key":"mock-api-key","uid":"10001"}`), nil)
	}
	fixtures, err := loadWsAPIFixtures()
	require.NoError(tb, err, "loadWsAPIFixtures must not error")
	for _, f := range fixtures[req.Channel] {
		if !jsonEqual(tb, f.Params, req.Payload.RequestParam) {
			continue
		}
		if f.Ack {
			ack, err := json.Marshal(map[string]any{"req_id": req.Payload.RequestID, "req_header": nil, "req_param": req.Payload.RequestParam})
			require.NoError(tb, err, "Marshal must not error")
			if err := writeWsAPIResponse(w, &req, true, ack, nil); err != nil {
				return err
			}
		}
		return writeWsAPIResponse(w, &req, false, f.Result, f.Errs)
	}
	tb.Errorf("no %s fixture matches parameters %s", req.Channel, req.Payload.RequestParam)
	return writeWsAPIResponse(w, &req, false, nil, json.RawMessage(`{"label":"NO_FIXTURE","message":"no fixture matches the request"}`))
}

// writeWsAPIResponse writes a WebSocket API response in Gate's format, failed when errs is set
func writeWsAPIResponse(w *gws.Conn, req *wsAPIRequest, ack bool, result, errs json.RawMessage) error {
	status := "200"
	data := map[string]json.RawMessage{"result": result}
	if errs != nil {
		status = "400"
		data = map[string]json.RawMessage{"errs": errs}
	}
	resp := map[string]any{
		"request_id": req.Payload.RequestID,
		"header": map[string]string{
			"response_time": strconv.FormatInt(time.Now().UnixMilli(), 10),
			"status":        status,
			"channel":       req.Channel,
			"event":         req.Event,
		},
		"data": data,
	}
	if ack {
		resp["ack"] = true
	}
	return w.WriteJSON(resp)
}

// jsonEqual reports whether two JSON documents hold the same values, whatever their key order or spacing
func jsonEqual(tb testing.TB, a, b json.RawMessage) bool {
	tb.Helper()
	var x, y any
	require.NoError(tb, json.Unmarshal(a, &x), "fixture parameters must decode")
	require.NoError(tb, json.Unmarshal(b, &y), "request parameters must decode")
	return reflect.DeepEqual(x, y)
}

// newWsAPITestExchange returns an exchange whose websocket serves Gate's WebSocket API: the fixtures in testdata/wsAPI.json
// in mock tests, and Gate in live tests, which skip without credentials or, when canManipulateOrders is passed, without
// permission to manipulate orders
func newWsAPITestExchange(tb testing.TB, a asset.Item, canManipulateOrders ...bool) *Exchange {
	tb.Helper()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(tb, e, canManipulateOrders...)
		return newExchangeWithWebsocket(tb, a)
	}
	ex := new(Exchange)
	require.NoError(tb, testexch.Setup(ex), "Setup must not error")
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	// Cached futures account IDs keep connecting from looking them up over REST
	for _, settle := range []currency.Code{currency.USDT, currency.BTC} {
		ex.setFuturesUserID("mock-api-key", settle, "20011")
	}
	// Only the asset under test keeps its subscriptions, so only its connection dials; the mock accepts them all
	for _, enabled := range ex.GetAssetTypes(true) {
		if enabled != a {
			require.NoErrorf(tb, ex.CurrencyPairs.SetAssetEnabled(enabled, false), "SetAssetEnabled must not error for %s", enabled)
		}
	}
	s := httptest.NewTestServer(tb, mockws.CurryWsMockUpgrader(tb, wsAPIMock))
	s.Start() // The exchange's connectors dial with their own dialer, which only reaches a listening server
	require.NoError(tb, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(s.URL, "http")), "SetAllConnectionURLs must not error")
	require.NoError(tb, ex.Websocket.Connect(tb.Context()), "Connect must not error")
	// The server closes with the test, so the websocket is disabled first, as its connection monitor would otherwise
	// redial the closed server
	tb.Cleanup(func() {
		if ex.Websocket.IsEnabled() {
			assert.NoError(tb, ex.Websocket.Disable(), "Disable should not error")
		}
		if err := ex.Websocket.Shutdown(); err != nil {
			assert.ErrorIs(tb, err, websocket.ErrNotConnected, "Shutdown should only error when already disconnected")
		}
	})
	return ex
}

func TestWebsocketResponseAckInspector(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		payload string
		final   bool
	}{
		{name: "ack", payload: `{"request_id":"1","ack":true,"data":{"result":{"req_id":"1"}}}`},
		{name: "result", payload: `{"request_id":"1","header":{"status":"200"},"data":{"result":{"id":"1"}}}`, final: true},
		{name: "result mentioning an ack", payload: `{"request_id":"1","header":{"status":"200"},"data":{"result":{"text":"t-ack","stack":true}}}`, final: true},
		{name: "explicit false ack", payload: `{"request_id":"1","ack":false,"data":{"result":{}}}`, final: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.final, wsRespAckInspector{}.IsFinal([]byte(tc.payload)), "IsFinal should only treat a top level ack as an ack")
		})
	}
}
