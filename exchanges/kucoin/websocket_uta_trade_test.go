package kucoin

import (
	"context"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var utaReplyID = regexp.MustCompile(`"id":"[0-9a-f]+"`)

// utaTradeServer records the messages a mock UTA trade server receives
type utaTradeServer struct {
	mu       sync.Mutex
	received []map[string]any
}

// messages returns the messages the server has received
func (s *utaTradeServer) messages() []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.received
}

// reply answers a trade connection message from the order operation fixtures, giving an operation's reply the
// operation's ID. Client order IDs select the failures: a rejection, a malformed reply or no reply
func (s *utaTradeServer) reply(lines []string) func(msg map[string]any) []string {
	return func(msg map[string]any) []string {
		s.mu.Lock()
		s.received = append(s.received, msg)
		s.mu.Unlock()
		var line string
		switch msg["op"] {
		case "auth":
			return []string{lines[0]}
		case "uta.order":
			line = lines[1]
			if args, ok := msg["args"].(map[string]any); ok {
				switch args["clientOid"] {
				case "rejected-order":
					line = lines[4]
				case "malformed-reply":
					line = `{"code":"200000","data":[],"id":"0"}`
				case "unanswered-order":
					return nil
				}
			}
		case "uta.cancel":
			line = lines[2]
		case "uta.amend":
			line = lines[3]
		default:
			return nil
		}
		id, _ := msg["id"].(string)
		return []string{utaReplyID.ReplaceAllString(line, `"id":"`+id+`"`)}
	}
}

// newUTATradeTestConnection connects an exchange's UTA trade connection to a mock trade server answering from the order
// operation fixtures or, with -tags mock_test_off, to KuCoin
func newUTATradeTestConnection(t *testing.T, ex *Exchange) *utaTradeServer {
	t.Helper()
	server := new(utaTradeServer)
	wsURL := kucoinUTATradeWebsocketURL
	if mockTests {
		wsURL = newUTAMockServer(t, false, server.reply(utaFixtureLines(t, "testdata/wsUTAOrderOperations.jsonl")))
	}
	newUTAWebsocket(t, ex, ex.newUTATradeConnection(ex.Config, wsURL))
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	return server
}

// wsPlaceUTALiveOrder places a live test's order over the trade connection, which is cancelled if the test fails
func wsPlaceUTALiveOrder(t *testing.T, ex *Exchange, r *PlaceUTAOrderRequest) *UTAWebsocketOrderResponse {
	t.Helper()
	cancelUTALiveOrderOnFailure(t, r)
	resp, err := ex.WebsocketAddUTAOrder(t.Context(), r)
	require.NoError(t, err, "WebsocketAddUTAOrder must not error")
	require.NotNil(t, resp, "WebsocketAddUTAOrder must return a response")
	require.NotEmpty(t, resp.Data.OrderID, "WebsocketAddUTAOrder must return the order ID")
	return resp
}

func TestAuthenticateUTATradeWebsocket(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	welcome := utaFixtureLines(t, "testdata/wsUTAOrderOperations.jsonl")[0]
	conn := &utaFakeConnection{reply: []byte(welcome)}
	require.NoError(t, ex.AuthenticateUTATradeWebsocket(t.Context(), conn), "AuthenticateUTATradeWebsocket must not error")
	require.Len(t, conn.sent, 1, "AuthenticateUTATradeWebsocket must send one message")
	req, ok := conn.sent[0].(*UTAWebsocketAuthRequest)
	require.True(t, ok, "AuthenticateUTATradeWebsocket must send an authentication request")
	assert.Empty(t, req.ID, "the authentication message should carry no ID, as the reply carries none")
	assert.Equal(t, "auth", req.Operation, "the authentication message should be an auth operation")
	assert.Equal(t, utaTradeAuthSignature, conn.signatures[0], "the reply should be matched as the authentication reply")

	rejected := utaFixtureLines(t, "testdata/wsUTAOrderOperations.jsonl")[5]
	err := ex.AuthenticateUTATradeWebsocket(t.Context(), &utaFakeConnection{reply: []byte(rejected)})
	assert.ErrorIs(t, err, errUTAAuthenticationFailed, "AuthenticateUTATradeWebsocket should error when KuCoin rejects the sign-in")
	assert.ErrorContains(t, err, "KC-API-KEY not exists.", "AuthenticateUTATradeWebsocket should report KuCoin's message")
	err = ex.AuthenticateUTATradeWebsocket(t.Context(), &utaFakeConnection{sendErr: websocket.ErrSignatureTimeout})
	assert.ErrorIs(t, err, websocket.ErrSignatureTimeout, "AuthenticateUTATradeWebsocket should return a send error")
	assert.Error(t, ex.AuthenticateUTATradeWebsocket(t.Context(), &utaFakeConnection{reply: []byte(`{`)}), "AuthenticateUTATradeWebsocket should error on a malformed reply")
	assert.ErrorIs(t, ex.AuthenticateUTATradeWebsocket(t.Context(), &utaFakeConnection{reply: []byte(`null`)}), common.ErrNoResponse, "AuthenticateUTATradeWebsocket should error on a null reply")

	ex.SkipAuthCheck = false // The mock instance skips credential checks
	ex.SetCredentials(&accounts.Credentials{})
	assert.ErrorIs(t, ex.AuthenticateUTATradeWebsocket(t.Context(), &utaFakeConnection{}), exchange.ErrCredentialsAreEmpty, "AuthenticateUTATradeWebsocket should error without credentials")
}

func TestUTAWebsocketTradeAuthResponseDecoding(t *testing.T) {
	t.Parallel()
	lines := utaFixtureLines(t, "testdata/wsUTAOrderOperations.jsonl")
	exp := []UTAWebsocketTradeAuthResponse{
		{SessionID: "cbb32567-489d-4102-a35e-9dcd61dc0919", Data: "welcome", PingInterval: 18000, PingTimeout: 10000},
		{Code: "400003", Message: "KC-API-KEY not exists.", InTime: types.Time(time.UnixMilli(1741589088843)), OutTime: types.Time(time.UnixMilli(1741589088848))},
	}
	got := make([]UTAWebsocketTradeAuthResponse, 2)
	for i, line := range []string{lines[0], lines[5]} {
		require.NoErrorf(t, json.Unmarshal([]byte(line), &got[i]), "authentication reply %d must decode", i)
	}
	assert.Equal(t, exp, got, "the welcome and rejection replies should decode every field")
}

func TestWebsocketHandleUTATradeData(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	conn := &utaFakeConnection{}
	require.NoError(t, ex.wsHandleUTATradeData(t.Context(), conn, []byte(`{"id":"ping","op":"pong","timestamp":1773135391743}`)), "wsHandleUTATradeData must ignore a pong")
	assert.Empty(t, conn.matched, "wsHandleUTATradeData should not match a pong to a request")
	lines := utaFixtureLines(t, "testdata/wsUTAOrderOperations.jsonl")
	require.NoError(t, ex.wsHandleUTATradeData(t.Context(), conn, []byte(lines[1])), "wsHandleUTATradeData must route an operation reply")
	require.NoError(t, ex.wsHandleUTATradeData(t.Context(), conn, []byte(lines[0])), "wsHandleUTATradeData must route the welcome frame")
	require.NoError(t, ex.wsHandleUTATradeData(t.Context(), conn, []byte(lines[5])), "wsHandleUTATradeData must route an authentication rejection")
	assert.Equal(t, []any{"946ffd6b29904649b199a50885e114be", utaTradeAuthSignature, utaTradeAuthSignature}, conn.matched, "wsHandleUTATradeData should match replies by ID and authentication replies by their signature")
	conn.matchErr = websocket.ErrSignatureNotMatched
	assert.ErrorIs(t, ex.wsHandleUTATradeData(t.Context(), conn, []byte(lines[2])), websocket.ErrSignatureNotMatched, "wsHandleUTATradeData should return an unmatched reply's error")
	assert.Error(t, ex.wsHandleUTATradeData(t.Context(), conn, []byte(`{`)), "wsHandleUTATradeData should error for a malformed frame")
}

// TestUTATradeConnection signs in the UTA trade connection, against a mock server or, with -tags mock_test_off, against
// KuCoin when credentials are set
func TestUTATradeConnection(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ex.SetCredentials(apiCredentials)
	}
	server := newUTATradeTestConnection(t, ex)
	conn, err := ex.Websocket.GetConnection(utaTradeConnection)
	require.NoError(t, err, "GetConnection must not error once the trade connection signs in")
	require.NotNil(t, conn, "GetConnection must return the trade connection")
	if mockTests {
		received := server.messages()
		require.Len(t, received, 1, "the trade connection must send its authentication message alone")
		assert.Equal(t, "auth", received[0]["op"], "the trade connection should sign in")
		assert.NotContains(t, received[0], "id", "the trade connection's authentication message should carry no ID")
	}
}

func TestUTAWebsocketTradeStatusTradeStatus(t *testing.T) {
	t.Parallel()
	resp := &UTAWebsocketOrderResponse{Code: "200000"}
	assert.Same(t, &resp.UTAWebsocketTradeStatus, resp.tradeStatus(), "tradeStatus should return the reply's own status")
}

func TestWebsocketAddUTAOrder(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	_, err := ex.WebsocketAddUTAOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketAddUTAOrder must reject a nil request")
	req := &PlaceUTAOrderRequest{
		TradeType:     "FUTURES",
		ClientOrderID: "313e793a-a88b-4311-a590-a3b3bac00d2f-8df",
		Symbol:        "XBTUSDTM",
		Side:          "BUY",
		OrderType:     "LIMIT",
		Size:          1,
		SizeUnit:      "UNIT",
		Price:         63310,
		TimeInForce:   "GTC",
		Tags:          "KuCoin API Service",
	}
	_, err = ex.WebsocketAddUTAOrder(t.Context(), req)
	require.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAddUTAOrder must error without a trade connection")
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ex.SetCredentials(apiCredentials)
		newUTATradeTestConnection(t, ex)
		live := newUTALiveFuturesOrder(t)
		placed := wsPlaceUTALiveOrder(t, ex, live)
		_, err = ex.WebsocketCancelUTAOrder(t.Context(), &CancelUTAOrderRequest{TradeType: live.TradeType, Symbol: live.Symbol, OrderID: placed.Data.OrderID})
		require.NoError(t, err, "WebsocketCancelUTAOrder must cancel the test's order")
		return
	}
	server := newUTATradeTestConnection(t, ex)
	resp, err := ex.WebsocketAddUTAOrder(t.Context(), req)
	require.NoError(t, err, "WebsocketAddUTAOrder must not error")
	require.NotNil(t, resp, "WebsocketAddUTAOrder must return a response")
	received := server.messages()
	require.Len(t, received, 2, "the mock server must receive the authentication and order messages")
	exp := map[string]any{
		"id": resp.ID,
		"op": "uta.order",
		"args": map[string]any{
			"clientOid":   "313e793a-a88b-4311-a590-a3b3bac00d2f-8df",
			"orderType":   "LIMIT",
			"price":       "63310",
			"side":        "BUY",
			"size":        "1",
			"sizeUnit":    "UNIT",
			"symbol":      "XBTUSDTM",
			"tags":        "KuCoin API Service",
			"timeInForce": "GTC",
			"tradeType":   "FUTURES",
		},
	}
	assert.Equal(t, exp, received[1], "WebsocketAddUTAOrder should send the Place Order body as the Add Order arguments")
	assert.Len(t, resp.ID, 32, "WebsocketAddUTAOrder's request ID should fit KuCoin's 32 byte limit")
	expResp := &UTAWebsocketOrderResponse{
		ID:            resp.ID,
		Operation:     "uta.order",
		Code:          "200000",
		InTime:        types.Time(time.Unix(0, 1773135391711213000)),
		OutTime:       types.Time(time.Unix(0, 1773135391743642000)),
		UserRateLimit: UTAWebsocketRateLimit{Limit: 2000, Remaining: 1999, Reset: 14406},
		Data:          UTAOrderResponse{ClientOrderID: "313e793a-a88b-4311-a590-a3b3bac00d2f-8df", OrderID: "420781761635647489", TradeType: "FUTURES", Timestamp: types.Time(time.Unix(0, 1773135391743000000))},
	}
	assert.Equal(t, expResp, resp, "WebsocketAddUTAOrder should decode every field of the reply")
}

func TestWebsocketCancelUTAOrder(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	_, err := ex.WebsocketCancelUTAOrder(t.Context(), &CancelUTAOrderRequest{TradeType: "FUTURES", Symbol: "XBTUSDTM"})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "WebsocketCancelUTAOrder must reject a request identifying no order")
	req := &CancelUTAOrderRequest{TradeType: "FUTURES", Symbol: "XBTUSDTM", OrderID: "420781761635647489"}
	_, err = ex.WebsocketCancelUTAOrder(t.Context(), req)
	require.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketCancelUTAOrder must error without a trade connection")
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ex.SetCredentials(apiCredentials)
		newUTATradeTestConnection(t, ex)
		live := newUTALiveFuturesOrder(t)
		placed := wsPlaceUTALiveOrder(t, ex, live)
		resp, err := ex.WebsocketCancelUTAOrder(t.Context(), &CancelUTAOrderRequest{TradeType: live.TradeType, Symbol: live.Symbol, OrderID: placed.Data.OrderID})
		require.NoError(t, err, "WebsocketCancelUTAOrder must not error")
		require.NotNil(t, resp, "WebsocketCancelUTAOrder must return a response")
		assert.Equal(t, placed.Data.OrderID, resp.Data.OrderID, "WebsocketCancelUTAOrder should return the cancelled order's ID")
		return
	}
	server := newUTATradeTestConnection(t, ex)
	resp, err := ex.WebsocketCancelUTAOrder(t.Context(), req)
	require.NoError(t, err, "WebsocketCancelUTAOrder must not error")
	require.NotNil(t, resp, "WebsocketCancelUTAOrder must return a response")
	received := server.messages()
	require.Len(t, received, 2, "the mock server must receive the authentication and cancel messages")
	exp := map[string]any{
		"id":   resp.ID,
		"op":   "uta.cancel",
		"args": map[string]any{"orderId": "420781761635647489", "symbol": "XBTUSDTM", "tradeType": "FUTURES"},
	}
	assert.Equal(t, exp, received[1], "WebsocketCancelUTAOrder should send the Cancel Order body as the operation's arguments")
	expResp := &UTAWebsocketOrderResponse{
		ID:            resp.ID,
		Operation:     "uta.cancel",
		Code:          "200000",
		InTime:        types.Time(time.Unix(0, 1773135434165975000)),
		OutTime:       types.Time(time.Unix(0, 1773135434167140000)),
		UserRateLimit: UTAWebsocketRateLimit{Limit: 2000, Remaining: 1998, Reset: 14363},
		Data:          UTAOrderResponse{ClientOrderID: "313e793a-a88b-4311-a590-a3b3bac00d2f-8df", OrderID: "420781761635647489", TradeType: "FUTURES", Timestamp: types.Time(time.Unix(0, 1773135434166000000))},
	}
	assert.Equal(t, expResp, resp, "WebsocketCancelUTAOrder should decode every field of the reply")
}

func TestWebsocketAmendUTAOrder(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	_, err := ex.WebsocketAmendUTAOrder(t.Context(), &AmendUTAOrderRequest{OrderID: "454538143408951297"})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "WebsocketAmendUTAOrder must reject a request without a symbol")
	req := &AmendUTAOrderRequest{Symbol: "DOGSUSDTM", OrderID: "454538143408951297", ClientOrderID: "11", NewSize: 9, NewPrice: 0.0000433, SizeUnit: "UNIT"}
	_, err = ex.WebsocketAmendUTAOrder(t.Context(), req)
	require.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAmendUTAOrder must error without a trade connection")
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ex.SetCredentials(apiCredentials)
		newUTATradeTestConnection(t, ex)
		live := newUTALiveFuturesOrder(t)
		placed := wsPlaceUTALiveOrder(t, ex, live)
		resp, err := ex.WebsocketAmendUTAOrder(t.Context(), &AmendUTAOrderRequest{Symbol: live.Symbol, OrderID: placed.Data.OrderID, NewPrice: utaLiveFuturesPrice(t, 0.75)})
		require.NoError(t, err, "WebsocketAmendUTAOrder must not error")
		require.NotNil(t, resp, "WebsocketAmendUTAOrder must return a response")
		assert.NotEmpty(t, resp.Data.OrderID, "WebsocketAmendUTAOrder should return the order ID")
		// The client order ID finds the order whether or not the amendment gave it a new order ID
		_, err = ex.WebsocketCancelUTAOrder(t.Context(), &CancelUTAOrderRequest{TradeType: live.TradeType, Symbol: live.Symbol, ClientOrderID: live.ClientOrderID})
		require.NoError(t, err, "WebsocketCancelUTAOrder must cancel the test's order")
		return
	}
	server := newUTATradeTestConnection(t, ex)
	resp, err := ex.WebsocketAmendUTAOrder(t.Context(), req)
	require.NoError(t, err, "WebsocketAmendUTAOrder must not error")
	require.NotNil(t, resp, "WebsocketAmendUTAOrder must return a response")
	received := server.messages()
	require.Len(t, received, 2, "the mock server must receive the authentication and amend messages")
	exp := map[string]any{
		"id":   resp.ID,
		"op":   "uta.amend",
		"args": map[string]any{"clientOid": "11", "newPrice": 0.0000433, "newSize": 9.0, "orderId": "454538143408951297", "sizeUnit": "UNIT", "symbol": "DOGSUSDTM"},
	}
	assert.Equal(t, exp, received[1], "WebsocketAmendUTAOrder should send the Amend Order body as the operation's arguments, the decimals as bare numbers")
	expResp := &UTAWebsocketAmendOrderResponse{
		ID:            resp.ID,
		Operation:     "uta.amend",
		Code:          "200000",
		InTime:        types.Time(time.Unix(0, 1768909952943183000)),
		OutTime:       types.Time(time.Unix(0, 1768909952945083000)),
		UserRateLimit: UTAWebsocketRateLimit{Limit: 2000, Remaining: 1997, Reset: 21361},
		Data:          AmendUTAOrderResponse{ClientOrderID: "11", OrderID: "454538143408951297"},
	}
	assert.Equal(t, expResp, resp, "WebsocketAmendUTAOrder should decode every field of the reply")
}

func TestSendUTATradeRequest(t *testing.T) {
	t.Parallel()
	ex := newUTATestExchange(t)
	args := map[string]any{"tradeType": "FUTURES", "symbol": "XBTUSDTM", "side": "BUY", "orderType": "LIMIT", "price": "63310", "size": "1", "sizeUnit": "UNIT"}
	err := ex.sendUTATradeRequest(t.Context(), placeUTAOrderEPL, "uta.order", args, new(UTAWebsocketOrderResponse))
	require.ErrorIs(t, err, websocket.ErrNotConnected, "sendUTATradeRequest must error without a trade connection")
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ex.SetCredentials(apiCredentials)
		newUTATradeTestConnection(t, ex)
		live := newUTALiveFuturesOrder(t)
		args, err = live.Payload()
		require.NoError(t, err, "Payload must not error")
		cancelUTALiveOrderOnFailure(t, live)
		resp := new(UTAWebsocketOrderResponse)
		require.NoError(t, ex.sendUTATradeRequest(t.Context(), placeUTAOrderEPL, "uta.order", args, resp), "sendUTATradeRequest must not error")
		require.NotEmpty(t, resp.Data.OrderID, "sendUTATradeRequest must decode the reply into the result")
		_, err = ex.WebsocketCancelUTAOrder(t.Context(), &CancelUTAOrderRequest{TradeType: live.TradeType, Symbol: live.Symbol, OrderID: resp.Data.OrderID})
		require.NoError(t, err, "WebsocketCancelUTAOrder must cancel the test's order")
		return
	}
	server := newUTATradeTestConnection(t, ex)
	resp := new(UTAWebsocketOrderResponse)
	require.NoError(t, ex.sendUTATradeRequest(t.Context(), placeUTAOrderEPL, "uta.order", args, resp), "sendUTATradeRequest must not error")
	assert.Equal(t, "200000", resp.Code, "sendUTATradeRequest should decode the reply into the result")

	args["clientOid"] = "rejected-order"
	err = ex.sendUTATradeRequest(t.Context(), placeUTAOrderEPL, "uta.order", args, new(UTAWebsocketOrderResponse))
	assert.ErrorIs(t, err, errUTARequestRejected, "sendUTATradeRequest should error when KuCoin rejects the operation")
	assert.ErrorContains(t, err, "100003", "sendUTATradeRequest should report KuCoin's code")
	args["clientOid"] = "malformed-reply"
	assert.Error(t, ex.sendUTATradeRequest(t.Context(), placeUTAOrderEPL, "uta.order", args, new(UTAWebsocketOrderResponse)), "sendUTATradeRequest should error for a reply it cannot decode")

	// KuCoin's rate limiter, which the mock instance disables, holds no limit for a generic endpoint limit
	require.NoError(t, ex.EnableRateLimiter(), "EnableRateLimiter must not error")
	args["clientOid"] = "unlimited-order"
	sent := len(server.messages())
	err = ex.sendUTATradeRequest(t.Context(), request.Unset, "uta.order", args, new(UTAWebsocketOrderResponse))
	assert.ErrorIs(t, err, common.ErrNilPointer, "sendUTATradeRequest should return the rate limiter's error")
	assert.Empty(t, server.messages()[sent:], "sendUTATradeRequest should not send an operation the rate limiter refuses")
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")

	args["clientOid"] = "unanswered-order"
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	err = ex.sendUTATradeRequest(ctx, placeUTAOrderEPL, "uta.order", args, new(UTAWebsocketOrderResponse))
	assert.ErrorIs(t, err, context.DeadlineExceeded, "sendUTATradeRequest should return the error waiting for a reply")
}
