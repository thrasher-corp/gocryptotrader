package kucoin

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// orderAPITestCredentials are the credentials the local order API server checks signatures with
var orderAPITestCredentials = &accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret", ClientID: "mock-passphrase"}

// orderAPIServer is a local websocket order API, which checks the connection's signatures, answers each operation with
// its recorded reply and records the requests it receives
type orderAPIServer struct {
	t        *testing.T
	mu       sync.Mutex
	session  string
	welcome  string
	replies  map[string]json.RawMessage
	requests chan *WsOrderMessage
	pings    chan struct{}
}

// setWelcome sets the message the server welcomes the next session with
func (s *orderAPIServer) setWelcome(welcome string) {
	s.mu.Lock()
	s.welcome = welcome
	s.mu.Unlock()
}

func (s *orderAPIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/v1/private" {
		http.NotFound(w, r)
		return
	}
	q := r.URL.Query()
	sign, err := crypto.GetHMAC(crypto.HashSHA256, []byte(q.Get("apikey")+q.Get("timestamp")), []byte(orderAPITestCredentials.Secret))
	assert.NoError(s.t, err, "signing should not error")
	assert.Equal(s.t, orderAPITestCredentials.Key, q.Get("apikey"), "the connection should be made with the API key")
	assert.Equal(s.t, base64.StdEncoding.EncodeToString(sign), q.Get("sign"), "the connection should be signed with the key and the time")
	c, err := (&gws.Upgrader{}).Upgrade(w, r, nil)
	if !assert.NoError(s.t, err, "Upgrade should not error") {
		return
	}
	defer c.Close()
	c.SetPingHandler(func(string) error {
		select {
		case s.pings <- struct{}{}:
		default:
		}
		return nil
	})
	s.mu.Lock()
	welcome := s.welcome
	s.mu.Unlock()
	if !assert.NoError(s.t, c.WriteMessage(gws.TextMessage, []byte(s.session)), "writing the session should not error") {
		return
	}
	_, signature, err := c.ReadMessage()
	if err != nil {
		return
	}
	sessionSign, err := crypto.GetHMAC(crypto.HashSHA256, []byte(s.session), []byte(orderAPITestCredentials.Secret))
	assert.NoError(s.t, err, "signing should not error")
	assert.Equal(s.t, base64.StdEncoding.EncodeToString(sessionSign), string(signature), "the session should be signed with the secret")
	if !assert.NoError(s.t, c.WriteMessage(gws.TextMessage, []byte(welcome)), "writing the welcome should not error") {
		return
	}
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			return
		}
		var req WsOrderMessage
		if !assert.NoError(s.t, json.Unmarshal(msg, &req), "a request should decode") {
			return
		}
		select {
		case s.requests <- &req:
		default:
		}
		key := req.Operation
		if strings.Contains(string(msg), `"FAIL`) {
			key = "placeError"
		}
		var reply map[string]any
		if !assert.NoError(s.t, json.Unmarshal(s.replies[key], &reply), "the reply should decode") {
			return
		}
		reply["id"] = req.ID
		reply["op"] = req.Operation
		out, err := json.Marshal(reply)
		if !assert.NoError(s.t, err, "the reply should encode") {
			return
		}
		if !assert.NoError(s.t, c.WriteMessage(gws.TextMessage, out), "writing the reply should not error") {
			return
		}
	}
}

// newOrderAPIServer starts a local websocket order API and returns it with its URL
func newOrderAPIServer(t *testing.T) (api *orderAPIServer, wsURL string) {
	t.Helper()
	s := &orderAPIServer{
		t:        t,
		session:  string(wsFixture(t, "wsOrderAPI.json", "session")),
		welcome:  string(wsFixture(t, "wsOrderAPI.json", "welcome")),
		replies:  make(map[string]json.RawMessage),
		requests: make(chan *WsOrderMessage, 8),
		pings:    make(chan struct{}, 8),
	}
	for _, key := range wsFixtureKeys(t, "wsOrderAPI.json") {
		s.replies[key] = wsFixture(t, "wsOrderAPI.json", key)
	}
	server := httptest.NewTestServer(t, s)
	server.Start()
	return s, "ws" + strings.TrimPrefix(server.URL, "http") + "/v1/private"
}

// orderAPITestInstance returns an exchange whose websocket connects only the order API, to a local server
func orderAPITestInstance(t *testing.T) (*Exchange, *orderAPIServer) {
	t.Helper()
	server, u := newOrderAPIServer(t)
	ex := wsTestInstance(t)
	ex.SkipAuthCheck = true
	ex.SetCredentials(orderAPITestCredentials)
	ex.Features.Subscriptions = subscription.List{}
	require.NoError(t, ex.Websocket.SetupNewConnection(ex.newOrderConnectionSetup(&config.Exchange{WebsocketResponseMaxLimit: 5 * time.Second}, u)), "SetupNewConnection must not error")
	connectTestWebsocket(t, ex)
	return ex, server
}

// orderAPIRequest returns the request the local order API last received
func orderAPIRequest(t *testing.T, s *orderAPIServer) *WsOrderMessage {
	t.Helper()
	select {
	case req := <-s.requests:
		return req
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the order API received no request")
	}
	return nil
}

// assertOrderAPIArgs asserts the arguments of the request the local order API received
func assertOrderAPIArgs(t *testing.T, s *orderAPIServer, op, expArgs string) {
	t.Helper()
	req := orderAPIRequest(t, s)
	assert.Equal(t, op, req.Operation, "the request should name the operation")
	assert.NotEmpty(t, req.ID, "the request should carry an ID")
	args, err := json.Marshal(req.Args)
	require.NoError(t, err, "the arguments must encode")
	assert.JSONEq(t, expArgs, string(args), "the request should carry the arguments")
}

func TestNewOrderConnectionSetup(t *testing.T) {
	t.Parallel()
	s := e.newOrderConnectionSetup(&config.Exchange{WebsocketResponseCheckTimeout: time.Second, WebsocketResponseMaxLimit: 2 * time.Second}, kucoinOrderWebsocketURL)
	assert.Equal(t, kucoinOrderWebsocketURL, s.URL, "URL should be the order API URL")
	assert.Equal(t, orderConnection, s.MessageFilter, "MessageFilter should name the order connection")
	assert.Equal(t, time.Second, s.ResponseCheckTimeout, "ResponseCheckTimeout should come from the config")
	assert.Equal(t, 2*time.Second, s.ResponseMaxLimit, "ResponseMaxLimit should come from the config")
	assert.True(t, s.SubscriptionsNotRequired, "the order connection should need no subscriptions")
	assert.NotNil(t, s.Connector, "Connector should be set")
	assert.NotNil(t, s.Handler, "Handler should be set")
}

func TestWsConnectOrderAPI(t *testing.T) {
	t.Parallel()
	server, u := newOrderAPIServer(t)
	server.setWelcome(`{"sessionId":"92f2aec4-d87e-47cc-917d-4e7c93911bdc","data":"welcome","pingInterval":100,"pingTimeout":10000}`)
	ex := wsTestInstance(t)
	ex.SkipAuthCheck = true
	ex.SetCredentials(orderAPITestCredentials)
	require.NoError(t, ex.Websocket.SetupNewConnection(ex.newOrderConnectionSetup(&config.Exchange{}, u)), "SetupNewConnection must not error")
	conn, err := ex.Websocket.CreateTestConnection(orderConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	require.NoError(t, ex.WsConnectOrderAPI(t.Context(), conn), "WsConnectOrderAPI must not error")
	select {
	case <-server.pings:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the order API connection sent no ping")
	}
	require.NoError(t, conn.Shutdown(), "Shutdown must not error")

	fake := newFakeConnection(`{"sessionId":"92f2aec4-d87e-47cc-917d-4e7c93911bdc","timestamp":1742175983882}`, `{"sessionId":"92f2aec4-d87e-47cc-917d-4e7c93911bdc","data":"welcome"}`)
	require.NoError(t, ex.WsConnectOrderAPI(t.Context(), fake), "WsConnectOrderAPI must not error for a welcome without a ping interval")
	assert.Equal(t, websocket.PingHandler{MessageType: gws.PingMessage, Delay: defaultPingDelay}, fake.pingHandler, "a session welcomed without a ping interval should be pinged at the default delay")

	conn, err = ex.Websocket.CreateTestConnection(orderConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	conn.SetURL(strings.TrimSuffix(u, "/v1/private") + "/v1/missing")
	err = ex.WsConnectOrderAPI(t.Context(), conn)
	require.ErrorIs(t, err, errWebsocketDialFailed, "WsConnectOrderAPI must return a dial failure")
	assert.ErrorIs(t, err, gws.ErrBadHandshake, "a dial failure should keep the dial error")
	assert.NotContains(t, err.Error(), orderAPITestCredentials.Key, "a dial failure should not repeat the API key")
	assert.NotContains(t, err.Error(), "sign=", "a dial failure should not repeat the signature")
	assert.NotContains(t, err.Error(), "passphrase=", "a dial failure should not repeat the signed passphrase")

	server.setWelcome(string(wsFixture(t, "wsOrderAPI.json", "connectionError")))
	conn, err = ex.Websocket.CreateTestConnection(orderConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	require.ErrorIs(t, ex.WsConnectOrderAPI(t.Context(), conn), errOrderSessionNotWelcomed, "WsConnectOrderAPI must reject a session that is not welcomed")

	ex.SkipAuthCheck = false
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{})
	require.ErrorIs(t, ex.WsConnectOrderAPI(t.Context(), newFakeConnection()), exchange.ErrCredentialsAreEmpty, "WsConnectOrderAPI must need credentials")
}

func TestOrderAPIQuery(t *testing.T) {
	t.Parallel()
	got := orderAPIQuery(orderAPITestCredentials, time.UnixMilli(1791899445520))
	exp := url.Values{
		"apikey":     {"mock-api-key"},
		"timestamp":  {"1791899445520"},
		"sign":       {"DLyuv6/qhmguSHIP0Wja2kF65XirI/mvPMioTpg3/ik="},
		"passphrase": {"UoArgMFMpw+bMnJs/PmxRr/gfaitkhbNgN+ULAC/f+U="},
	}
	assert.Equal(t, exp, got, "orderAPIQuery should sign the key and time and the passphrase")
}

func TestAuthenticateOrderSession(t *testing.T) {
	t.Parallel()
	session := `{"sessionId":"92f2aec4-d87e-47cc-917d-4e7c93911bdc","timestamp":1742175983882}`
	welcome := `{"sessionId":"92f2aec4-d87e-47cc-917d-4e7c93911bdc","data":"welcome","pingInterval":18000,"pingTimeout":10000}`
	ex := wsTestInstance(t)
	conn := newFakeConnection(session, welcome)
	got, err := ex.authenticateOrderSession(t.Context(), conn, orderAPITestCredentials.Secret)
	require.NoError(t, err, "authenticateOrderSession must not error")
	assert.Equal(t, &WsOrderWelcome{SessionID: "92f2aec4-d87e-47cc-917d-4e7c93911bdc", Data: "welcome", PingInterval: 18000, PingTimeout: 10000}, got, "authenticateOrderSession should return the welcome")
	assert.Equal(t, []string{"8q146CxA6r6/y2+Xp+84WYNluke8FzKHpanF5WWxGcU="}, conn.rawSent, "authenticateOrderSession should send the session's signature")

	errSend := errors.New("send failed")
	for _, tc := range []struct {
		name     string
		messages []string
		sendErr  error
		err      error
	}{
		{name: "no session", err: errWelcomeNotReceived},
		{name: "undecodable session", messages: []string{`{"sessionId":`}},
		{name: "error instead of a session", messages: []string{`{"code":"400003","msg":"KC-API-KEY not exists."}`}, err: errOrderSessionNotWelcomed},
		{name: "session signature not sent", messages: []string{session, welcome}, sendErr: errSend, err: errSend},
		{name: "no welcome", messages: []string{session}, err: errWelcomeNotReceived},
		{name: "undecodable welcome", messages: []string{session, `{"data":`}},
		{name: "error instead of a welcome", messages: []string{session, `{"code":"400003","msg":"KC-API-KEY not exists."}`}, err: errOrderSessionNotWelcomed},
		{name: "welcome for another session", messages: []string{session, `{"sessionId":"another","data":"welcome"}`}, err: errOrderSessionNotWelcomed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := wsTestInstance(t)
			ex.WebsocketResponseMaxLimit = 10 * time.Millisecond
			conn := newFakeConnection(tc.messages...)
			conn.sendErr = tc.sendErr
			t.Cleanup(func() { assert.NoError(t, conn.Shutdown(), "Shutdown should not error") })
			_, err := ex.authenticateOrderSession(t.Context(), conn, orderAPITestCredentials.Secret)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err, "authenticateOrderSession must return the expected error")
				return
			}
			require.Error(t, err, "authenticateOrderSession must reject an undecodable message")
		})
	}
}

func TestWsHandleOrderData(t *testing.T) {
	t.Parallel()
	ex := wsTestInstance(t)
	conn := newFakeConnection()
	require.NoError(t, ex.wsHandleOrderData(t.Context(), conn, []byte(`{"id":"ping-1","op":"pong","timestamp":1742175983882}`)), "wsHandleOrderData must not error for a pong")
	assert.Empty(t, conn.matched, "a pong should not be matched to a request")
	reply := wsFixture(t, "wsOrderAPI.json", "placeError")
	require.NoError(t, ex.wsHandleOrderData(t.Context(), conn, reply), "wsHandleOrderData must not error for a reply")
	assert.Equal(t, map[any][]byte{"759ad5add86b4b09b35145a4d6f49488": reply}, conn.matched, "a reply should be matched to its request")
	err := ex.wsHandleOrderData(t.Context(), conn, wsFixture(t, "wsOrderAPI.json", "validationError"))
	require.ErrorIs(t, err, errWsOrderRequestFailed, "wsHandleOrderData must return an error answered without an ID")
	assert.ErrorContains(t, err, "4001002", "wsHandleOrderData should name the error code")
	require.Error(t, ex.wsHandleOrderData(t.Context(), conn, []byte(`{"id":`)), "wsHandleOrderData must reject an undecodable message")
}

func TestWsAddSpotOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WsAddSpotOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsAddSpotOrder must reject a nil request")
	req := &PlaceHFOrderRequest{ClientOrderID: "e4b2bab8-7b55-433e-a72f-78efc8976e1a-456", Symbol: "BTC-USDT", Type: "limit", Side: "buy", Price: 65000, Size: 0.00001, Remark: "KuCoin API Service"}
	_, err = e.WsAddSpotOrder(t.Context(), req)
	require.ErrorIs(t, err, websocket.ErrNotConnected, "WsAddSpotOrder must need the order connection")

	ex, server := orderAPITestInstance(t)
	got, err := ex.WsAddSpotOrder(t.Context(), req)
	require.NoError(t, err, "WsAddSpotOrder must not error")
	assertOrderAPIArgs(t, server, "spot.order", `{"clientOid":"e4b2bab8-7b55-433e-a72f-78efc8976e1a-456","price":"65000","remark":"KuCoin API Service","side":"buy","size":"0.00001","symbol":"BTC-USDT","type":"limit"}`)
	assert.Equal(t, &WsAddOrderResponse{ClientOrderID: "e4b2bab8-7b55-433e-a72f-78efc8976e1a-456", OrderID: "69afcfa5598e6800072afba3"}, got, "WsAddSpotOrder should decode every field")
}

func TestWsAddSpotOrderSync(t *testing.T) {
	t.Parallel()
	_, err := e.WsAddSpotOrderSync(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsAddSpotOrderSync must reject a nil request")

	ex, server := orderAPITestInstance(t)
	req := &PlaceHFOrderRequest{ClientOrderID: "c8c89703-b2de-488f-8836-e129ab98c748-9a2", Symbol: "BTC-USDT", Type: "limit", Side: "buy", Price: 65000, Size: 0.00001}
	got, err := ex.WsAddSpotOrderSync(t.Context(), req)
	require.NoError(t, err, "WsAddSpotOrderSync must not error")
	assertOrderAPIArgs(t, server, "spot.sync_order", `{"clientOid":"c8c89703-b2de-488f-8836-e129ab98c748-9a2","price":"65000","side":"buy","size":"0.00001","symbol":"BTC-USDT","type":"limit"}`)
	exp := &WsAddSpotOrderSyncResponse{
		CanceledSize:  0.000001,
		ClientOrderID: "c8c89703-b2de-488f-8836-e129ab98c748-9a2",
		DealSize:      0.000002,
		MatchTime:     types.Time(time.UnixMilli(1773131628649)),
		OrderID:       "69afd76c3378e900079a9455",
		OrderTime:     types.Time(time.UnixMilli(1773131628626)),
		OriginSize:    0.00001,
		RemainSize:    0.000007,
		Status:        "open",
	}
	assert.Equal(t, exp, got, "WsAddSpotOrderSync should decode every field")
}

func TestWsAddMarginOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WsAddMarginOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsAddMarginOrder must reject a nil request")

	ex, server := orderAPITestInstance(t)
	req := &PlaceMarginHFOrderRequest{ClientOrderID: "6216aa42-894b-4cdc-94ac-7c46c6b976d4-78c", Symbol: "BTC-USDT", Type: "limit", Side: "buy", Price: 65000, Size: 0.00001, AutoBorrow: true}
	got, err := ex.WsAddMarginOrder(t.Context(), req)
	require.NoError(t, err, "WsAddMarginOrder must not error")
	assertOrderAPIArgs(t, server, "margin.order", `{"autoBorrow":true,"clientOid":"6216aa42-894b-4cdc-94ac-7c46c6b976d4-78c","price":"65000","side":"buy","size":"0.00001","symbol":"BTC-USDT","type":"limit"}`)
	exp := &WsAddMarginOrderResponse{BorrowSize: 10.2, ClientOrderID: "6216aa42-894b-4cdc-94ac-7c46c6b976d4-78c", LoanApplyID: "600656d9a33ac90009de4f6f", OrderID: "69afda52f5c2aa00072ee487"}
	assert.Equal(t, exp, got, "WsAddMarginOrder should decode every field")
}

func TestWsAddFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WsAddFuturesOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsAddFuturesOrder must reject a nil request")

	ex, server := orderAPITestInstance(t)
	req := &FuturesOrderRequest{ClientOrderID: "70e7d33c-ee8f-477d-b2a3-ce700bb9362e-e22", Symbol: "XBTUSDTM", Leverage: 100, Type: "limit", Side: "buy", Price: 70000, Size: 1}
	got, err := ex.WsAddFuturesOrder(t.Context(), req)
	require.NoError(t, err, "WsAddFuturesOrder must not error")
	assertOrderAPIArgs(t, server, "futures.order", `{"clientOid":"70e7d33c-ee8f-477d-b2a3-ce700bb9362e-e22","leverage":100,"price":"70000","side":"buy","size":1,"symbol":"XBTUSDTM","type":"limit"}`)
	assert.Equal(t, &WsAddOrderResponse{ClientOrderID: "70e7d33c-ee8f-477d-b2a3-ce700bb9362e-e22", OrderID: "420780675814141952"}, got, "WsAddFuturesOrder should decode every field")

	req.Remark = "FAIL"
	_, err = ex.WsAddFuturesOrder(t.Context(), req)
	require.ErrorIs(t, err, errWsOrderRequestFailed, "WsAddFuturesOrder must return the error the order API answers with")
	assert.ErrorContains(t, err, "100003", "the error should name the code")
}

func TestWsBatchAddFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.WsBatchAddFuturesOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "WsBatchAddFuturesOrders must reject no orders")

	ex, server := orderAPITestInstance(t)
	reqs := []FuturesOrderRequest{
		{ClientOrderID: "ab7f92df-8f7e-47a0-820b-7912b95195be-d9d", Symbol: "XBTUSDTM", Leverage: 100, Type: "limit", Side: "buy", Price: 70000, Size: 1},
		{ClientOrderID: "485ca98d-2567-4aba-afa2-8114c327fe03-6d4", Symbol: "XBTUSDTM", Leverage: 100, Type: "limit", Side: "sell", Price: 72000, Size: 1},
	}
	got, err := ex.WsBatchAddFuturesOrders(t.Context(), reqs)
	require.NoError(t, err, "WsBatchAddFuturesOrders must not error")
	assertOrderAPIArgs(t, server, "futures.multi_order", `[{"clientOid":"ab7f92df-8f7e-47a0-820b-7912b95195be-d9d","leverage":100,"price":"70000","side":"buy","size":1,"symbol":"XBTUSDTM","type":"limit"},{"clientOid":"485ca98d-2567-4aba-afa2-8114c327fe03-6d4","leverage":100,"price":"72000","side":"sell","size":1,"symbol":"XBTUSDTM","type":"limit"}]`)
	exp := []WsFuturesOrderResult{
		{ClientOrderID: "ab7f92df-8f7e-47a0-820b-7912b95195be-d9d", Code: "200000", Message: "success", OrderID: "420778397380227073", Symbol: "XBTUSDTM"},
		{ClientOrderID: "485ca98d-2567-4aba-afa2-8114c327fe03-6d4", Code: "100001", Message: "Price parameter invalid.", OrderID: "420778397413781504", Symbol: "XBTUSDTM"},
	}
	assert.Equal(t, exp, got, "WsBatchAddFuturesOrders should return each order's result")
}

func TestWsCancelSpotOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WsCancelSpotOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsCancelSpotOrder must reject a nil request")
	_, err = e.WsCancelSpotOrder(t.Context(), &WsCancelOrderRequest{OrderID: "69afcfa5598e6800072afba3"})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "WsCancelSpotOrder must need a symbol")

	ex, server := orderAPITestInstance(t)
	got, err := ex.WsCancelSpotOrder(t.Context(), &WsCancelOrderRequest{Symbol: "BTC-USDT", ClientOrderID: "e4b2bab8-7b55-433e-a72f-78efc8976e1a-456"})
	require.NoError(t, err, "WsCancelSpotOrder must not error")
	assertOrderAPIArgs(t, server, "spot.cancel", `{"symbol":"BTC-USDT","clientOid":"e4b2bab8-7b55-433e-a72f-78efc8976e1a-456"}`)
	assert.Equal(t, &WsCancelOrderResponse{ClientOrderID: "e4b2bab8-7b55-433e-a72f-78efc8976e1a-456", OrderID: "69afcfa5598e6800072afba3"}, got, "WsCancelSpotOrder should decode every field")
}

func TestWsCancelSpotOrderSync(t *testing.T) {
	t.Parallel()
	_, err := e.WsCancelSpotOrderSync(t.Context(), &WsCancelOrderRequest{Symbol: "BTC-USDT"})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "WsCancelSpotOrderSync must need an order ID")

	ex, server := orderAPITestInstance(t)
	got, err := ex.WsCancelSpotOrderSync(t.Context(), &WsCancelOrderRequest{Symbol: "BTC-USDT", OrderID: "69afd76c3378e900079a9455"})
	require.NoError(t, err, "WsCancelSpotOrderSync must not error")
	assertOrderAPIArgs(t, server, "spot.sync_cancel", `{"symbol":"BTC-USDT","orderId":"69afd76c3378e900079a9455"}`)
	exp := &WsCancelSpotOrderSyncResponse{
		CanceledSize:  0.00001,
		ClientOrderID: "c8c89703-b2de-488f-8836-e129ab98c748-9a2",
		DealSize:      0.000002,
		OrderID:       "69afd76c3378e900079a9455",
		OriginSize:    0.000012,
		Status:        "done",
	}
	assert.Equal(t, exp, got, "WsCancelSpotOrderSync should decode every field")
}

func TestWsCancelMarginOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WsCancelMarginOrder(t.Context(), &WsCancelOrderRequest{ClientOrderID: "6216aa42-894b-4cdc-94ac-7c46c6b976d4-78c"})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "WsCancelMarginOrder must need a symbol")

	ex, server := orderAPITestInstance(t)
	got, err := ex.WsCancelMarginOrder(t.Context(), &WsCancelOrderRequest{Symbol: "BTC-USDT", OrderID: "69afda52f5c2aa00072ee487", ClientOrderID: "6216aa42-894b-4cdc-94ac-7c46c6b976d4-78c"})
	require.NoError(t, err, "WsCancelMarginOrder must not error")
	assertOrderAPIArgs(t, server, "margin.cancel", `{"symbol":"BTC-USDT","orderId":"69afda52f5c2aa00072ee487","clientOid":"6216aa42-894b-4cdc-94ac-7c46c6b976d4-78c"}`)
	assert.Equal(t, &WsCancelOrderResponse{ClientOrderID: "6216aa42-894b-4cdc-94ac-7c46c6b976d4-78c", OrderID: "69afda52f5c2aa00072ee487"}, got, "WsCancelMarginOrder should decode every field")
}

func TestWsCancelFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WsCancelFuturesOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsCancelFuturesOrder must reject a nil request")
	_, err = e.WsCancelFuturesOrder(t.Context(), &WsCancelOrderRequest{ClientOrderID: "085410bdb2f34e3bb4aa29a50856a0909d3e7475"})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "WsCancelFuturesOrder must need a symbol with a client order ID")

	ex, server := orderAPITestInstance(t)
	got, err := ex.WsCancelFuturesOrder(t.Context(), &WsCancelOrderRequest{OrderID: "420772522842234880"})
	require.NoError(t, err, "WsCancelFuturesOrder must not error without a symbol for an order ID")
	assertOrderAPIArgs(t, server, "futures.cancel", `{"orderId":"420772522842234880"}`)
	assert.Equal(t, &WsCancelFuturesOrderResponse{CancelledOrderIDs: []string{"420772522842234880"}, ClientOrderID: "085410bdb2f34e3bb4aa29a50856a0909d3e7475"}, got, "WsCancelFuturesOrder should decode every field")
}

func TestWsBatchCancelFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.WsBatchCancelFuturesOrders(t.Context(), &WsBatchCancelFuturesOrdersRequest{})
	require.ErrorIs(t, err, errNoOrders, "WsBatchCancelFuturesOrders must reject no orders")

	ex, server := orderAPITestInstance(t)
	got, err := ex.WsBatchCancelFuturesOrders(t.Context(), &WsBatchCancelFuturesOrdersRequest{ClientOrders: []WsFuturesClientOrder{
		{Symbol: "XBTUSDTM", ClientOrderID: "ab7f92df-8f7e-47a0-820b-7912b95195be-d9d"},
		{Symbol: "XBTUSDTM", ClientOrderID: "485ca98d-2567-4aba-afa2-8114c327fe03-6d4"},
	}})
	require.NoError(t, err, "WsBatchCancelFuturesOrders must not error")
	assertOrderAPIArgs(t, server, "futures.multi_cancel", `{"clientOidsList":[{"symbol":"XBTUSDTM","clientOid":"ab7f92df-8f7e-47a0-820b-7912b95195be-d9d"},{"symbol":"XBTUSDTM","clientOid":"485ca98d-2567-4aba-afa2-8114c327fe03-6d4"}]}`)
	exp := []WsFuturesOrderResult{
		{ClientOrderID: "ab7f92df-8f7e-47a0-820b-7912b95195be-d9d", Code: "200", Message: "success", OrderID: "420778397380227073", Symbol: "XBTUSDTM"},
		{ClientOrderID: "485ca98d-2567-4aba-afa2-8114c327fe03-6d4", Code: "300000", Message: "order does not exist", OrderID: "420778397413781504", Symbol: "XBTUSDTM"},
	}
	assert.Equal(t, exp, got, "WsBatchCancelFuturesOrders should return each order's result")
}

func TestWsCancelOrderRequestArgs(t *testing.T) {
	t.Parallel()
	var nilRequest *WsCancelOrderRequest
	_, err := nilRequest.args(true)
	require.ErrorIs(t, err, common.ErrNilPointer, "args must reject a nil request")
	_, err = (&WsCancelOrderRequest{OrderID: "1"}).args(true)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "args must need a symbol when told to")
	_, err = (&WsCancelOrderRequest{Symbol: "BTC-USDT"}).args(true)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "args must need an order ID or client order ID")
	got, err := (&WsCancelOrderRequest{OrderID: "1"}).args(false)
	require.NoError(t, err, "args must not error")
	assert.Equal(t, map[string]any{"orderId": "1"}, got, "args should leave out an empty symbol and client order ID")
	got, err = (&WsCancelOrderRequest{Symbol: "BTC-USDT", OrderID: "1", ClientOrderID: "2"}).args(true)
	require.NoError(t, err, "args must not error")
	assert.Equal(t, map[string]any{"symbol": "BTC-USDT", "orderId": "1", "clientOid": "2"}, got, "args should carry every ID given")
}

func TestWsBatchCancelFuturesOrdersRequestArgs(t *testing.T) {
	t.Parallel()
	var nilRequest *WsBatchCancelFuturesOrdersRequest
	_, err := nilRequest.args()
	require.ErrorIs(t, err, common.ErrNilPointer, "args must reject a nil request")
	_, err = (&WsBatchCancelFuturesOrdersRequest{}).args()
	require.ErrorIs(t, err, errNoOrders, "args must need an order")
	_, err = (&WsBatchCancelFuturesOrdersRequest{ClientOrders: []WsFuturesClientOrder{{ClientOrderID: "1"}}}).args()
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "args must need each client order's symbol")
	_, err = (&WsBatchCancelFuturesOrdersRequest{ClientOrders: []WsFuturesClientOrder{{Symbol: "XBTUSDTM"}}}).args()
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "args must need each client order's ID")
	got, err := (&WsBatchCancelFuturesOrdersRequest{OrderIDs: []string{"1", "2"}, ClientOrders: []WsFuturesClientOrder{{Symbol: "XBTUSDTM", ClientOrderID: "3"}}}).args()
	require.NoError(t, err, "args must not error")
	exp := map[string]any{"orderIdsList": []string{"1", "2"}, "clientOidsList": []map[string]any{{"symbol": "XBTUSDTM", "clientOid": "3"}}}
	assert.Equal(t, exp, got, "args should list the order IDs and client orders")
}

func TestSendWsOrderRequest(t *testing.T) {
	t.Parallel()
	var resp *WsAddOrderResponse
	require.ErrorIs(t, e.sendWsOrderRequest(t.Context(), request.Unset, "spot.order", nil, &resp), websocket.ErrNotConnected, "sendWsOrderRequest must need the order connection")

	ex, server := orderAPITestInstance(t)
	// A REST limit of one order an hour, already used, would hold the next order past the caller's deadline
	requester, err := request.New(ex.Name, new(http.Client), request.WithLimiter(request.RateLimitDefinitions{placeHFOrderEPL: request.NewWeightedRateLimitByDuration(time.Hour)}))
	require.NoError(t, err, "request.New must not error")
	ex.Requester = requester
	require.NoError(t, ex.Requester.InitiateRateLimit(t.Context(), placeHFOrderEPL), "the first order must be admitted at once")
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	err = ex.sendWsOrderRequest(ctx, placeHFOrderEPL, "spot.order", nil, &resp)
	require.ErrorIs(t, err, context.DeadlineExceeded, "sendWsOrderRequest must return a rate limit error")
	assert.Empty(t, server.requests, "an order the REST order limit refuses should not reach the order API")
}

// failingOrderConnection answers order requests with a scripted reply or error
type failingOrderConnection struct {
	websocket.Connection
	reply []byte
	err   error
}

func (c *failingOrderConnection) SendMessageReturnResponse(context.Context, request.EndpointLimit, any, any) ([]byte, error) {
	return c.reply, c.err
}

func TestSendWsOrderRequestOnConnection(t *testing.T) {
	t.Parallel()
	ex := wsTestInstance(t)
	var resp *WsAddOrderResponse
	errSend := errors.New("send failed")
	require.ErrorIs(t, ex.sendWsOrderRequestOnConnection(t.Context(), &failingOrderConnection{err: errSend}, "spot.order", nil, &resp), errSend, "sendWsOrderRequestOnConnection must return a send error")
	require.Error(t, ex.sendWsOrderRequestOnConnection(t.Context(), &failingOrderConnection{reply: []byte(`{"code":`)}, "spot.order", nil, &resp), "sendWsOrderRequestOnConnection must reject an undecodable reply")
	err := ex.sendWsOrderRequestOnConnection(t.Context(), &failingOrderConnection{reply: wsFixture(t, "wsOrderAPI.json", "placeError")}, "futures.order", nil, &resp)
	require.ErrorIs(t, err, errWsOrderRequestFailed, "sendWsOrderRequestOnConnection must return the error the order API answers with")
	assert.ErrorContains(t, err, "Contract parameter invalid.", "the error should carry the message")
	require.Error(t, ex.sendWsOrderRequestOnConnection(t.Context(), &failingOrderConnection{reply: []byte(`{"code":"200000","data":[]}`)}, "spot.order", nil, &resp), "sendWsOrderRequestOnConnection must reject undecodable data")
	require.NoError(t, ex.sendWsOrderRequestOnConnection(t.Context(), &failingOrderConnection{reply: wsFixture(t, "wsOrderAPI.json", "spot.order")}, "spot.order", nil, &resp), "sendWsOrderRequestOnConnection must not error")
	assert.Equal(t, &WsAddOrderResponse{ClientOrderID: "e4b2bab8-7b55-433e-a72f-78efc8976e1a-456", OrderID: "69afcfa5598e6800072afba3"}, resp, "sendWsOrderRequestOnConnection should decode the result")
}

// TestWsOrderResponseDecoding decodes a recorded order API reply's envelope
func TestWsOrderResponseDecoding(t *testing.T) {
	t.Parallel()
	var got WsOrderResponse
	require.NoError(t, json.Unmarshal(wsFixture(t, "wsOrderAPI.json", "placeError"), &got), "the reply must decode")
	exp := WsOrderResponse{
		Code:      "100003",
		Message:   "Contract parameter invalid.",
		ID:        "759ad5add86b4b09b35145a4d6f49488",
		Operation: "futures.order",
		InTime:    types.Time(time.UnixMilli(1741587922123)),
		OutTime:   types.Time(time.UnixMilli(1741587922128)),
		RateLimit: WsOrderRateLimit{Limit: 1600, Remaining: 1528, Reset: 15244},
	}
	assert.Equal(t, exp, got, "an error reply should decode every field")
	require.NoError(t, json.Unmarshal(wsFixture(t, "wsOrderAPI.json", "spot.order"), &got), "the reply must decode")
	assert.Equal(t, WsOrderRateLimit{Limit: 20000, Remaining: 19999, Reset: 5892}, got.UserRateLimit, "a reply should decode the user's rate limit")
	assert.Equal(t, types.Time(time.Unix(0, 1773129637307039000)), got.InTime, "a reply should decode nanosecond times")
}
