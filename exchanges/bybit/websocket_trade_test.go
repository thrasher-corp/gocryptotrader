package bybit

import (
	"context"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testutils "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// mockTradeRequest is a request the mock trade and private streams receive
type mockTradeRequest struct {
	RequestID       string            `json:"reqId"`
	StreamRequestID string            `json:"req_id"`
	Header          map[string]string `json:"header"`
	Operation       string            `json:"op"`
	Arguments       []json.RawMessage `json:"args"`
}

// tradeStreamMock serves the trade and private streams: it answers authentication, records order requests and answers
// each with the reply its reply func returns, after pushing the order update that reply returns on the private stream
type tradeStreamMock struct {
	mu       sync.Mutex
	private  *gws.Conn
	requests []*mockTradeRequest
	reply    func(req *mockTradeRequest) (reply, push string)
}

func (m *tradeStreamMock) handle(_ testing.TB, msg []byte, c *gws.Conn) error {
	var req mockTradeRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	switch req.Operation {
	case authOperation:
		if req.StreamRequestID != "" {
			m.private = c
			return c.WriteMessage(gws.TextMessage, []byte(`{"success":true,"ret_msg":"","op":"auth","conn_id":"cejreaspqfh3sjdnldmg-p","req_id":"`+req.StreamRequestID+`"}`))
		}
		return c.WriteMessage(gws.TextMessage, []byte(`{"retCode":0,"retMsg":"OK","op":"auth","connId":"cnt5leec0hvan15eukcg-2t"}`))
	case "ping":
		return nil
	}
	m.requests = append(m.requests, &req)
	reply, push := m.reply(&req)
	if push != "" {
		if m.private == nil {
			return errors.New("private stream not authenticated")
		}
		if err := m.private.WriteMessage(gws.TextMessage, []byte(push)); err != nil {
			return err
		}
	}
	return c.WriteMessage(gws.TextMessage, []byte(reply))
}

// setReply changes how the mock answers order requests
func (m *tradeStreamMock) setReply(reply func(req *mockTradeRequest) (reply, push string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reply = reply
}

// received returns the order requests the mock received
func (m *tradeStreamMock) received() []*mockTradeRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.requests)
}

// newTradeStreamExchange returns an exchange whose websocket connections, including the authenticated trade and private
// connections, are served by m
func newTradeStreamExchange(t *testing.T, m *tradeStreamMock) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	for a, pairs := range map[asset.Item]currency.Pairs{
		asset.Spot:                {currency.NewBTCUSDT()},
		asset.USDTMarginedFutures: {currency.NewBTCUSDT()},
		asset.Options:             {fixtureOptionPair},
	} {
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, pairs, false), "StorePairs must not error for %s", a)
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, pairs, true), "StorePairs must not error for %s", a)
	}
	ex.Features.Subscriptions = subscription.List{}
	s := httptest.NewTestServer(t, mockws.CurryWsMockUpgrader(t, m.handle))
	s.Start()
	require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(s.URL, "http")), "SetAllConnectionURLs must not error")
	ex.Websocket.SetSubscriptionsNotRequired()
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	// The connections outlive the test context until the cleanup stops them, so its end sets off no reconnection
	require.NoError(t, ex.Websocket.Connect(context.WithoutCancel(t.Context())), "Connect must not error")
	t.Cleanup(func() { stopWebsocket(t, ex) })
	return ex
}

// orderReply returns a successful reply to an order request
func orderReply(req *mockTradeRequest, data string) string {
	return `{"reqId":"` + req.RequestID + `","retCode":0,"retMsg":"OK","op":"` + req.Operation + `","data":` + data + `,"retExtInfo":{},"header":{"X-Bapi-Limit":"10","X-Bapi-Limit-Status":"9","X-Bapi-Limit-Reset-Timestamp":"1711001595208","Traceid":"38b7977b430f9bd228f4b19724794dfd","Timenow":"1711001595209"},"connId":"cnt5leec0hvan15eukcg-2v"}`
}

// requestArgument decodes the order a mock request carries
func requestArgument(t *testing.T, req *mockTradeRequest) map[string]any {
	t.Helper()
	require.Len(t, req.Arguments, 1, "an order request must carry one argument")
	var arg map[string]any
	require.NoError(t, json.Unmarshal(req.Arguments[0], &arg), "order argument must decode")
	return arg
}

func TestWebsocketAuthenticateTradeConnection(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	err := ex.WebsocketAuthenticateTradeConnection(t.Context(), &FixtureConnection{})
	require.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "WebsocketAuthenticateTradeConnection must error without authentication support")

	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	conn := &FixtureConnection{sendMessageReturnResponseOverride: []byte(`{"retCode":0,"retMsg":"OK","op":"auth","connId":"d2a641kgcg7ab33b7mdg-4x6a"}`)}
	require.NoError(t, ex.WebsocketAuthenticateTradeConnection(ctx, conn), "WebsocketAuthenticateTradeConnection must not error")
	require.Len(t, conn.sent, 1, "WebsocketAuthenticateTradeConnection must send one request")
	req, ok := conn.sent[0].(*WebsocketTradeRequest)
	require.True(t, ok, "WebsocketAuthenticateTradeConnection must send a WebsocketTradeRequest")
	assert.Equal(t, authOperation, req.Operation, "request operation should be auth")
	assert.Empty(t, req.RequestID, "request should carry no request ID, so the reply is matched on its operation")
	assert.Len(t, req.Arguments, 3, "request should carry the key, expiry and signature")

	err = ex.WebsocketAuthenticateTradeConnection(ctx, &FixtureConnection{sendMessageReturnResponseOverride: []byte(`{"retCode":10004,"retMsg":"Invalid sign","op":"auth","connId":"d2a63t6p49kk82nefh90-4ye8"}`)})
	require.ErrorIs(t, err, errWebsocketAuthFailed, "WebsocketAuthenticateTradeConnection must report a rejected authentication")
	require.ErrorIs(t, err, errAPIResponse, "WebsocketAuthenticateTradeConnection must report Bybit's error")
	assert.ErrorContains(t, err, "Invalid sign", "error should carry Bybit's message")
	err = ex.WebsocketAuthenticateTradeConnection(ctx, &FixtureConnection{sendMessageReturnResponseOverride: []byte(`{"retCode":"0"}`)})
	require.Error(t, err, "WebsocketAuthenticateTradeConnection must error for an undecodable reply")
	err = ex.WebsocketAuthenticateTradeConnection(ctx, &FixtureConnection{reply: func(any) ([]byte, error) { return nil, errors.New("connection lost") }})
	require.ErrorContains(t, err, "connection lost", "WebsocketAuthenticateTradeConnection must return a failed send")
}

func TestWsHandleTradeData(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input string
		match string
		err   error
	}{
		{input: `{"reqId":"test-005","retCode":0,"retMsg":"OK","op":"order.create","data":{"orderId":"a4c1718e-fe53-4659-a118-1f6ecce04ad9","orderLinkId":""}}`, match: "test-005"},
		{input: `{"retCode":0,"retMsg":"OK","op":"auth","connId":"cnt5leec0hvan15eukcg-2t"}`, match: "auth"},
		{input: `{"retCode":0,"retMsg":"OK","op":"pong","data":["1711002002529"],"connId":"cnt5leec0hvan15eukcg-2v"}`},
		{input: `{"op":"pewpewpew"}`, err: errUnhandledStreamData},
		{input: `{"reqId":"noMatch","retCode":0,"op":"order.create"}`, err: websocket.ErrSignatureNotMatched},
		{input: `{"reqId":1}`, err: errors.New("reqId")},
		{input: `{"op":1}`, err: errors.New("op")},
	} {
		conn := &FixtureConnection{match: websocket.NewMatch()}
		var ch <-chan []byte
		if tc.match != "" {
			var err error
			ch, err = conn.match.Set(tc.match, 1)
			require.NoError(t, err, "match.Set must not error")
		}
		err := e.wsHandleTradeData(conn, []byte(tc.input))
		switch {
		case tc.err == nil:
			require.NoError(t, err, "wsHandleTradeData must not error")
		case tc.err.Error() == "reqId", tc.err.Error() == "op":
			require.Errorf(t, err, "wsHandleTradeData must error for %s", tc.input)
		default:
			require.ErrorIs(t, err, tc.err, "wsHandleTradeData must return the expected error")
		}
		if tc.match != "" {
			require.Len(t, ch, 1, "the reply must reach its request")
			assert.Equal(t, tc.input, string(<-ch), "the reply should reach its request unchanged")
		}
	}
}

func TestWebsocketCreateOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketCreateOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketCreateOrder must reject a nil request")
	_, err = e.WebsocketCreateOrder(t.Context(), &PlaceOrderRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "WebsocketCreateOrder must require a category")
	_, err = e.WebsocketCreateOrder(t.Context(), &PlaceOrderRequest{Category: "spread"})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "WebsocketCreateOrder must refuse a category websocket order entry does not support")
	_, err = e.WebsocketCreateOrder(t.Context(), &PlaceOrderRequest{Category: cSpot})
	require.ErrorIs(t, err, errSymbolMissing, "WebsocketCreateOrder must validate the request")
	_, err = e.WebsocketCreateOrder(t.Context(), &PlaceOrderRequest{Category: cSpot, Symbol: "BTCUSDT", Side: "Buy", OrderType: "Limit", Quantity: 0.0001})
	require.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketCreateOrder must not fall back to REST without a trade connection")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ex := getWebsocketInstance(t)
		got, err := ex.WebsocketCreateOrder(t.Context(), &PlaceOrderRequest{Category: cSpot, Symbol: "BTCUSDT", Side: "Buy", OrderType: "Limit", Price: 55000, Quantity: 0.0001, TimeInForce: "PostOnly"})
		require.NoError(t, err, "WebsocketCreateOrder must not error")
		assert.NotEmpty(t, got.OrderID, "WebsocketCreateOrder should return the order ID")
		return
	}
	m := &tradeStreamMock{reply: func(req *mockTradeRequest) (string, string) {
		return orderReply(req, `{"orderId":"c1956690-b731-4191-97c0-94b00422231b","orderLinkId":"linear-tp-001"}`), ""
	}}
	ex := newTradeStreamExchange(t, m)
	got, err := ex.WebsocketCreateOrder(t.Context(), &PlaceOrderRequest{Category: cLinear, Symbol: "BTCUSDT", Side: "Sell", OrderType: "Limit", Quantity: 0.5, Price: 82600, TimeInForce: "GTC", OrderLinkID: "linear-tp-001"})
	require.NoError(t, err, "WebsocketCreateOrder must not error")
	assert.Equal(t, &OrderResponse{OrderID: "c1956690-b731-4191-97c0-94b00422231b", OrderLinkID: "linear-tp-001"}, got, "WebsocketCreateOrder should return the order IDs Bybit accepted")
	requests := m.received()
	require.Len(t, requests, 1, "WebsocketCreateOrder must send one request")
	assert.Equal(t, "order.create", requests[0].Operation, "request operation should be order.create")
	assert.NotEmpty(t, requests[0].RequestID, "request should carry a request ID")
	assert.NotEmpty(t, requests[0].Header["X-BAPI-TIMESTAMP"], "request should carry a timestamp header")
	exp := map[string]any{"category": cLinear, "symbol": "BTCUSDT", "side": "Sell", "orderType": "Limit", "qty": "0.5", "price": "82600", "timeInForce": "GTC", "orderLinkId": "linear-tp-001"}
	assert.Equal(t, exp, requestArgument(t, requests[0]), "request should carry the Place Order body")

	m.setReply(func(req *mockTradeRequest) (string, string) {
		return orderReply(req, `{"orderId":"c1956690-b731-4191-97c0-94b00422231c","orderLinkId":""}`), ""
	})
	got, err = ex.WebsocketCreateOrder(t.Context(), &PlaceOrderRequest{Category: cLinear, Symbol: "BTCUSDT", Side: "Sell", OrderType: "Limit", Quantity: 0.5, Price: 82600})
	require.NoError(t, err, "WebsocketCreateOrder must not error without a client order ID")
	assert.Equal(t, &OrderResponse{OrderID: "c1956690-b731-4191-97c0-94b00422231c"}, got, "WebsocketCreateOrder should return the order ID Bybit accepted")
	assert.NotContains(t, requestArgument(t, m.received()[1]), "orderLinkId", "WebsocketCreateOrder should not invent a client order ID")

	m.setReply(func(req *mockTradeRequest) (string, string) {
		return `{"reqId":"` + req.RequestID + `","retCode":10404,"retMsg":"category is not correct/supported","op":"order.create","connId":"cnt5leec0hvan15eukcg-2v"}`, ""
	})
	_, err = ex.WebsocketCreateOrder(t.Context(), &PlaceOrderRequest{Category: cLinear, Symbol: "BTCUSDT", Side: "Sell", OrderType: "Limit", Quantity: 0.5, Price: 82600, OrderLinkID: "linear-tp-002"})
	require.ErrorIs(t, err, errAPIResponse, "WebsocketCreateOrder must report Bybit's error")
	assert.ErrorContains(t, err, "10404", "error should carry Bybit's code")
}

func TestWebsocketAmendOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAmendOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketAmendOrder must reject a nil request")
	_, err = e.WebsocketAmendOrder(t.Context(), &AmendOrderRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "WebsocketAmendOrder must require a category")
	_, err = e.WebsocketAmendOrder(t.Context(), &AmendOrderRequest{Category: "spread"})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "WebsocketAmendOrder must refuse a category websocket order entry does not support")
	_, err = e.WebsocketAmendOrder(t.Context(), &AmendOrderRequest{Category: cSpot, Symbol: "BTCUSDT"})
	require.ErrorIs(t, err, errEitherOrderIDOROrderLinkIDRequired, "WebsocketAmendOrder must validate the request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ex := getWebsocketInstance(t)
		got, err := ex.WebsocketAmendOrder(t.Context(), &AmendOrderRequest{Category: cSpot, Symbol: "BTCUSDT", OrderID: "1793353687809485568", Quantity: 0.0002})
		require.NoError(t, err, "WebsocketAmendOrder must not error")
		assert.NotEmpty(t, got.OrderID, "WebsocketAmendOrder should return the order ID")
		return
	}
	m := &tradeStreamMock{reply: func(req *mockTradeRequest) (string, string) {
		return orderReply(req, `{"orderId":"c1956690-b731-4191-97c0-94b00422231b","orderLinkId":"linear-tp-001"}`), ""
	}}
	ex := newTradeStreamExchange(t, m)
	got, err := ex.WebsocketAmendOrder(t.Context(), &AmendOrderRequest{Category: cLinear, Symbol: "BTCUSDT", OrderID: "c1956690-b731-4191-97c0-94b00422231b", Quantity: 0.5})
	require.NoError(t, err, "WebsocketAmendOrder must not error")
	assert.Equal(t, &OrderResponse{OrderID: "c1956690-b731-4191-97c0-94b00422231b", OrderLinkID: "linear-tp-001"}, got, "WebsocketAmendOrder should return the order IDs Bybit accepted")
	requests := m.received()
	require.Len(t, requests, 1, "WebsocketAmendOrder must send one request")
	assert.Equal(t, "order.amend", requests[0].Operation, "request operation should be order.amend")
	exp := map[string]any{"category": cLinear, "symbol": "BTCUSDT", "orderId": "c1956690-b731-4191-97c0-94b00422231b", "qty": "0.5"}
	assert.Equal(t, exp, requestArgument(t, requests[0]), "request should carry the Amend Order body")
}

func TestWebsocketCancelTradeOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketCancelTradeOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketCancelTradeOrder must reject a nil request")
	_, err = e.WebsocketCancelTradeOrder(t.Context(), &CancelOrderRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "WebsocketCancelTradeOrder must require a category")
	_, err = e.WebsocketCancelTradeOrder(t.Context(), &CancelOrderRequest{Category: "spread"})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "WebsocketCancelTradeOrder must refuse a category websocket order entry does not support")
	_, err = e.WebsocketCancelTradeOrder(t.Context(), &CancelOrderRequest{Category: cSpot, Symbol: "BTCUSDT"})
	require.ErrorIs(t, err, errEitherOrderIDOROrderLinkIDRequired, "WebsocketCancelTradeOrder must validate the request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ex := getWebsocketInstance(t)
		got, err := ex.WebsocketCancelTradeOrder(t.Context(), &CancelOrderRequest{Category: cSpot, Symbol: "BTCUSDT", OrderID: "1793353687809485568"})
		require.NoError(t, err, "WebsocketCancelTradeOrder must not error")
		assert.NotEmpty(t, got.OrderID, "WebsocketCancelTradeOrder should return the order ID")
		return
	}
	m := &tradeStreamMock{reply: func(req *mockTradeRequest) (string, string) {
		return orderReply(req, `{"orderId":"2290000000123456789","orderLinkId":"spot-buy-001"}`), ""
	}}
	ex := newTradeStreamExchange(t, m)
	got, err := ex.WebsocketCancelTradeOrder(t.Context(), &CancelOrderRequest{Category: cSpot, Symbol: "BTCUSDT", OrderLinkID: "spot-buy-001"})
	require.NoError(t, err, "WebsocketCancelTradeOrder must not error")
	assert.Equal(t, &OrderResponse{OrderID: "2290000000123456789", OrderLinkID: "spot-buy-001"}, got, "WebsocketCancelTradeOrder should return the order IDs Bybit accepted")
	requests := m.received()
	require.Len(t, requests, 1, "WebsocketCancelTradeOrder must send one request")
	assert.Equal(t, "order.cancel", requests[0].Operation, "request operation should be order.cancel")
	exp := map[string]any{"category": cSpot, "symbol": "BTCUSDT", "orderLinkId": "spot-buy-001"}
	assert.Equal(t, exp, requestArgument(t, requests[0]), "request should carry the Cancel Order body")

	m.setReply(func(req *mockTradeRequest) (string, string) {
		return `{"reqId":"` + req.RequestID + `","retCode":110001,"retMsg":"Order does not exist","op":"order.cancel","connId":"cnt5leec0hvan15eukcg-2v"}`, ""
	})
	_, err = ex.WebsocketCancelTradeOrder(t.Context(), &CancelOrderRequest{Category: cSpot, Symbol: "BTCUSDT", OrderLinkID: "spot-buy-002"})
	require.ErrorIs(t, err, order.ErrOrderNotFound, "WebsocketCancelTradeOrder must map Bybit's order not found code")
}

// batchReply returns a reply to a batch order request listing two orders, the second rejected
func batchReply(req *mockTradeRequest, createAt string) string {
	return `{"reqId":"` + req.RequestID + `","retCode":0,"retMsg":"OK","op":"` + req.Operation + `","data":{"list":[{"category":"linear","symbol":"BTCUSDT","orderId":"0f2a4c6e-8b1d-4f3a-9c5e-7d9b1f3a5c7e","orderLinkId":"batch-000","createAt":"` + createAt + `"},{"category":"linear","symbol":"BTCUSDT","orderId":"","orderLinkId":"batch-001","createAt":""}]},"retExtInfo":{"list":[{"code":0,"msg":"OK"},{"code":10001,"msg":"position idx not match position mode"}]},"header":{"Timenow":"1740453408556","X-Bapi-Limit":"150","X-Bapi-Limit-Status":"147","X-Bapi-Limit-Reset-Timestamp":"1740453408555","Traceid":"0e32b551b3e17aae77651aadf6a5be80"},"connId":"cupviqn88smf24t2kpb0-536o"}`
}

// batchResults returns the results batchReply holds
func batchResults(createAt time.Time) []WebsocketBatchOrderResult {
	return []WebsocketBatchOrderResult{
		{Category: cLinear, Symbol: "BTCUSDT", OrderID: "0f2a4c6e-8b1d-4f3a-9c5e-7d9b1f3a5c7e", OrderLinkID: "batch-000", CreateAt: createAt, Message: "OK"},
		{Category: cLinear, Symbol: "BTCUSDT", OrderLinkID: "batch-001", Code: 10001, Message: "position idx not match position mode"},
	}
}

func TestWebsocketCreateBatchOrders(t *testing.T) {
	t.Parallel()
	item := BatchPlaceOrderItem{Symbol: "BTCUSDT", Side: "Buy", OrderType: "Limit", Quantity: 0.001, Price: 82000, OrderLinkID: "batch-000"}
	_, err := e.WebsocketCreateBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketCreateBatchOrders must reject a nil request")
	_, err = e.WebsocketCreateBatchOrders(t.Context(), &BatchPlaceOrderRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "WebsocketCreateBatchOrders must require a category")
	_, err = e.WebsocketCreateBatchOrders(t.Context(), &BatchPlaceOrderRequest{Category: "spread"})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "WebsocketCreateBatchOrders must refuse a category websocket order entry does not support")
	_, err = e.WebsocketCreateBatchOrders(t.Context(), &BatchPlaceOrderRequest{Category: cLinear})
	require.ErrorIs(t, err, errNoOrderPassed, "WebsocketCreateBatchOrders must require orders")
	for _, tc := range []struct {
		mutate func(*BatchPlaceOrderItem)
		err    error
	}{
		{mutate: func(i *BatchPlaceOrderItem) { i.Symbol = "" }, err: errSymbolMissing},
		{mutate: func(i *BatchPlaceOrderItem) { i.Side = "" }, err: order.ErrSideIsInvalid},
		{mutate: func(i *BatchPlaceOrderItem) { i.OrderType = "" }, err: order.ErrTypeIsInvalid},
		{mutate: func(i *BatchPlaceOrderItem) { i.Quantity = 0 }, err: limits.ErrAmountBelowMin},
	} {
		bad := item
		tc.mutate(&bad)
		_, err = e.WebsocketCreateBatchOrders(t.Context(), &BatchPlaceOrderRequest{Category: cLinear, Request: []BatchPlaceOrderItem{item, bad}})
		require.ErrorIs(t, err, tc.err, "WebsocketCreateBatchOrders must validate every order")
	}
	_, err = e.WebsocketCreateBatchOrders(t.Context(), &BatchPlaceOrderRequest{Category: cLinear, Request: []BatchPlaceOrderItem{item}})
	require.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketCreateBatchOrders must not fall back to REST without a trade connection")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ex := getWebsocketInstance(t)
		got, err := ex.WebsocketCreateBatchOrders(t.Context(), &BatchPlaceOrderRequest{Category: cLinear, Request: []BatchPlaceOrderItem{item}})
		require.NoError(t, err, "WebsocketCreateBatchOrders must not error")
		assert.Len(t, got, 1, "WebsocketCreateBatchOrders should return each order's result")
		return
	}
	m := &tradeStreamMock{reply: func(req *mockTradeRequest) (string, string) { return batchReply(req, "1740453408555"), "" }}
	ex := newTradeStreamExchange(t, m)
	second := item
	second.OrderLinkID = "batch-001"
	got, err := ex.WebsocketCreateBatchOrders(t.Context(), &BatchPlaceOrderRequest{Category: cLinear, Request: []BatchPlaceOrderItem{item, second}})
	require.NoError(t, err, "WebsocketCreateBatchOrders must not error")
	assert.Equal(t, batchResults(time.UnixMilli(1740453408555)), got, "WebsocketCreateBatchOrders should return each order's result and outcome")
	requests := m.received()
	require.Len(t, requests, 1, "WebsocketCreateBatchOrders must send one request")
	assert.Equal(t, "order.create-batch", requests[0].Operation, "request operation should be order.create-batch")
	arg := requestArgument(t, requests[0])
	assert.Equal(t, cLinear, arg["category"], "request should carry the category")
	assert.Len(t, arg["request"], 2, "request should carry both orders")

	m.setReply(func(req *mockTradeRequest) (string, string) {
		return strings.Replace(batchReply(req, ""), `{"code":0,"msg":"OK"},`, "", 1), ""
	})
	_, err = ex.WebsocketCreateBatchOrders(t.Context(), &BatchPlaceOrderRequest{Category: cLinear, Request: []BatchPlaceOrderItem{item, second}})
	require.ErrorIs(t, err, errBatchOutcomeMismatch, "WebsocketCreateBatchOrders must not report results it cannot pair with outcomes")
	// Nor can a reply holding two orders' results be matched to a batch of one
	m.setReply(func(req *mockTradeRequest) (string, string) { return batchReply(req, "1740453408555"), "" })
	_, err = ex.WebsocketCreateBatchOrders(t.Context(), &BatchPlaceOrderRequest{Category: cLinear, Request: []BatchPlaceOrderItem{item}})
	require.ErrorIs(t, err, errBatchOutcomeMismatch, "WebsocketCreateBatchOrders must reject a reply for more orders than it sent")

	m.setReply(func(req *mockTradeRequest) (string, string) {
		return `{"reqId":"` + req.RequestID + `","retCode":10404,"retMsg":"category is not correct/supported","op":"order.create-batch","connId":"cupviqn88smf24t2kpb0-536o"}`, ""
	})
	_, err = ex.WebsocketCreateBatchOrders(t.Context(), &BatchPlaceOrderRequest{Category: cLinear, Request: []BatchPlaceOrderItem{item}})
	require.ErrorIs(t, err, errAPIResponse, "WebsocketCreateBatchOrders must report Bybit's error")
}

func TestWebsocketAmendBatchOrders(t *testing.T) {
	t.Parallel()
	item := BatchAmendOrderItem{Symbol: "BTCUSDT", OrderLinkID: "batch-000", Price: 82100}
	_, err := e.WebsocketAmendBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketAmendBatchOrders must reject a nil request")
	_, err = e.WebsocketAmendBatchOrders(t.Context(), &BatchAmendOrderRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "WebsocketAmendBatchOrders must require a category")
	_, err = e.WebsocketAmendBatchOrders(t.Context(), &BatchAmendOrderRequest{Category: "spread"})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "WebsocketAmendBatchOrders must refuse a category websocket order entry does not support")
	_, err = e.WebsocketAmendBatchOrders(t.Context(), &BatchAmendOrderRequest{Category: cLinear})
	require.ErrorIs(t, err, errNoOrderPassed, "WebsocketAmendBatchOrders must require orders")
	_, err = e.WebsocketAmendBatchOrders(t.Context(), &BatchAmendOrderRequest{Category: cLinear, Request: []BatchAmendOrderItem{item, {OrderID: "1"}}})
	require.ErrorIs(t, err, errSymbolMissing, "WebsocketAmendBatchOrders must require each order's symbol")
	_, err = e.WebsocketAmendBatchOrders(t.Context(), &BatchAmendOrderRequest{Category: cLinear, Request: []BatchAmendOrderItem{item, {Symbol: "BTCUSDT"}}})
	require.ErrorIs(t, err, errEitherOrderIDOROrderLinkIDRequired, "WebsocketAmendBatchOrders must require each order's ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ex := getWebsocketInstance(t)
		got, err := ex.WebsocketAmendBatchOrders(t.Context(), &BatchAmendOrderRequest{Category: cLinear, Request: []BatchAmendOrderItem{item}})
		require.NoError(t, err, "WebsocketAmendBatchOrders must not error")
		assert.Len(t, got, 1, "WebsocketAmendBatchOrders should return each order's result")
		return
	}
	m := &tradeStreamMock{reply: func(req *mockTradeRequest) (string, string) { return batchReply(req, ""), "" }}
	ex := newTradeStreamExchange(t, m)
	second := item
	second.OrderLinkID = "batch-001"
	got, err := ex.WebsocketAmendBatchOrders(t.Context(), &BatchAmendOrderRequest{Category: cLinear, Request: []BatchAmendOrderItem{item, second}})
	require.NoError(t, err, "WebsocketAmendBatchOrders must not error")
	assert.Equal(t, batchResults(time.Time{}), got, "WebsocketAmendBatchOrders should return each order's result and outcome")
	requests := m.received()
	require.Len(t, requests, 1, "WebsocketAmendBatchOrders must send one request")
	assert.Equal(t, "order.amend-batch", requests[0].Operation, "request operation should be order.amend-batch")
	// The reply holds two orders' results, which cannot be matched to a batch of one
	_, err = ex.WebsocketAmendBatchOrders(t.Context(), &BatchAmendOrderRequest{Category: cLinear, Request: []BatchAmendOrderItem{item}})
	require.ErrorIs(t, err, errBatchOutcomeMismatch, "WebsocketAmendBatchOrders must reject a reply for more orders than it sent")
}

func TestWebsocketCancelBatchOrders(t *testing.T) {
	t.Parallel()
	item := BatchCancelOrderItem{Symbol: "BTCUSDT", OrderLinkID: "batch-000"}
	_, err := e.WebsocketCancelBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketCancelBatchOrders must reject a nil request")
	_, err = e.WebsocketCancelBatchOrders(t.Context(), &BatchCancelOrderRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "WebsocketCancelBatchOrders must require a category")
	_, err = e.WebsocketCancelBatchOrders(t.Context(), &BatchCancelOrderRequest{Category: "spread"})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "WebsocketCancelBatchOrders must refuse a category websocket order entry does not support")
	_, err = e.WebsocketCancelBatchOrders(t.Context(), &BatchCancelOrderRequest{Category: cLinear})
	require.ErrorIs(t, err, errNoOrderPassed, "WebsocketCancelBatchOrders must require orders")
	_, err = e.WebsocketCancelBatchOrders(t.Context(), &BatchCancelOrderRequest{Category: cLinear, Request: []BatchCancelOrderItem{item, {OrderID: "1"}}})
	require.ErrorIs(t, err, errSymbolMissing, "WebsocketCancelBatchOrders must require each order's symbol")
	_, err = e.WebsocketCancelBatchOrders(t.Context(), &BatchCancelOrderRequest{Category: cLinear, Request: []BatchCancelOrderItem{item, {Symbol: "BTCUSDT"}}})
	require.ErrorIs(t, err, errEitherOrderIDOROrderLinkIDRequired, "WebsocketCancelBatchOrders must require each order's ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ex := getWebsocketInstance(t)
		got, err := ex.WebsocketCancelBatchOrders(t.Context(), &BatchCancelOrderRequest{Category: cLinear, Request: []BatchCancelOrderItem{item}})
		require.NoError(t, err, "WebsocketCancelBatchOrders must not error")
		assert.Len(t, got, 1, "WebsocketCancelBatchOrders should return each order's result")
		return
	}
	m := &tradeStreamMock{reply: func(req *mockTradeRequest) (string, string) { return batchReply(req, ""), "" }}
	ex := newTradeStreamExchange(t, m)
	second := item
	second.OrderLinkID = "batch-001"
	got, err := ex.WebsocketCancelBatchOrders(t.Context(), &BatchCancelOrderRequest{Category: cLinear, Request: []BatchCancelOrderItem{item, second}})
	require.NoError(t, err, "WebsocketCancelBatchOrders must not error")
	assert.Equal(t, batchResults(time.Time{}), got, "WebsocketCancelBatchOrders should return each order's result and outcome")
	requests := m.received()
	require.Len(t, requests, 1, "WebsocketCancelBatchOrders must send one request")
	assert.Equal(t, "order.cancel-batch", requests[0].Operation, "request operation should be order.cancel-batch")
	// The reply holds two orders' results, which cannot be matched to a batch of one
	_, err = ex.WebsocketCancelBatchOrders(t.Context(), &BatchCancelOrderRequest{Category: cLinear, Request: []BatchCancelOrderItem{item}})
	require.ErrorIs(t, err, errBatchOutcomeMismatch, "WebsocketCancelBatchOrders must reject a reply for more orders than it sent")
}

func TestWebsocketOrderEPL(t *testing.T) {
	t.Parallel()
	for category, exp := range map[string]any{cSpot: wsCreateSpotOrderEPL, cLinear: wsCreateOrderEPL, cInverse: wsCreateOrderEPL, cOption: wsCreateOptionOrderEPL} {
		epl, err := websocketOrderEPL("order.create", category)
		require.NoErrorf(t, err, "websocketOrderEPL must not error for %s", category)
		assert.Equalf(t, exp, epl, "websocketOrderEPL should return the %s limit", category)
	}
	_, err := websocketOrderEPL("order.create", "")
	require.ErrorIs(t, err, errCategoryNotSet, "websocketOrderEPL must require a category")
	_, err = websocketOrderEPL("order.create", "spread")
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "websocketOrderEPL must refuse spread trading")
	_, err = websocketOrderEPL("order.replace", cLinear)
	require.ErrorIs(t, err, errUnknownOrderOperation, "websocketOrderEPL must refuse an unknown operation")
}

func TestSendWebsocketOrderRequest(t *testing.T) {
	t.Parallel()
	payload := map[string]any{"category": cLinear, "symbol": "BTCUSDT", "orderId": "c1956690-b731-4191-97c0-94b00422231b"}
	_, err := websocketTestInstance(t).sendWebsocketOrderRequest(t.Context(), "order.cancel", payload, wsCancelOrderEPL)
	require.ErrorIs(t, err, websocket.ErrNotConnected, "sendWebsocketOrderRequest must error without a trade connection")

	m := &tradeStreamMock{reply: func(req *mockTradeRequest) (string, string) {
		return orderReply(req, `{"orderId":"c1956690-b731-4191-97c0-94b00422231b","orderLinkId":"linear-tp-001"}`), ""
	}}
	ex := newTradeStreamExchange(t, m)
	got, err := ex.sendWebsocketOrderRequest(t.Context(), "order.cancel", payload, wsCancelOrderEPL)
	require.NoError(t, err, "sendWebsocketOrderRequest must not error")
	assert.Equal(t, &OrderResponse{OrderID: "c1956690-b731-4191-97c0-94b00422231b", OrderLinkID: "linear-tp-001"}, got, "sendWebsocketOrderRequest should return the order IDs Bybit accepted")
	requests := m.received()
	require.Len(t, requests, 1, "sendWebsocketOrderRequest must send one request over the trade connection")
	assert.Equal(t, "order.cancel", requests[0].Operation, "sendWebsocketOrderRequest should send the operation")
	assert.Equal(t, map[string]any{"category": cLinear, "symbol": "BTCUSDT", "orderId": "c1956690-b731-4191-97c0-94b00422231b"}, requestArgument(t, requests[0]), "sendWebsocketOrderRequest should send the payload as the request's argument")

	m.setReply(func(req *mockTradeRequest) (string, string) {
		return `{"reqId":"` + req.RequestID + `","retCode":110001,"retMsg":"Order does not exist","op":"order.cancel","connId":"cnt5leec0hvan15eukcg-2v"}`, ""
	})
	got, err = ex.sendWebsocketOrderRequest(t.Context(), "order.cancel", payload, wsCancelOrderEPL)
	require.ErrorIs(t, err, errAPIResponse, "sendWebsocketOrderRequest must report Bybit's error")
	require.ErrorIs(t, err, order.ErrOrderNotFound, "sendWebsocketOrderRequest must map Bybit's order not found code")
	assert.Nil(t, got, "sendWebsocketOrderRequest should return no order IDs for a refused request")
}

func TestSendWebsocketBatchOrderRequest(t *testing.T) {
	t.Parallel()
	payload := map[string]any{"category": cLinear, "request": []any{
		map[string]any{"symbol": "BTCUSDT", "side": "Buy", "orderType": "Limit", "qty": "0.001", "price": "82000", "orderLinkId": "batch-000"},
		map[string]any{"symbol": "BTCUSDT", "side": "Buy", "orderType": "Limit", "qty": "0.001", "price": "82000", "orderLinkId": "batch-001"},
	}}
	_, err := websocketTestInstance(t).sendWebsocketBatchOrderRequest(t.Context(), "order.create-batch", cLinear, 2, payload, wsCreateBatchOrderEPL)
	require.ErrorIs(t, err, websocket.ErrNotConnected, "sendWebsocketBatchOrderRequest must error without a trade connection")

	m := &tradeStreamMock{reply: func(req *mockTradeRequest) (string, string) { return batchReply(req, "1740453408555"), "" }}
	ex := newTradeStreamExchange(t, m)
	got, err := ex.sendWebsocketBatchOrderRequest(t.Context(), "order.create-batch", cLinear, 2, payload, wsCreateBatchOrderEPL)
	require.NoError(t, err, "sendWebsocketBatchOrderRequest must not error")
	assert.Equal(t, batchResults(time.UnixMilli(1740453408555)), got, "sendWebsocketBatchOrderRequest should pair each order's result with its outcome")
	requests := m.received()
	require.Len(t, requests, 1, "sendWebsocketBatchOrderRequest must send one request over the trade connection")
	assert.Equal(t, "order.create-batch", requests[0].Operation, "sendWebsocketBatchOrderRequest should send the operation")
	assert.Equal(t, payload, requestArgument(t, requests[0]), "sendWebsocketBatchOrderRequest should send the payload as the request's argument")

	// Bybit counts each order of a linear batch against a budget of 10 a second, so a batch of 11 never fits it
	_, err = ex.sendWebsocketBatchOrderRequest(t.Context(), "order.create-batch", cLinear, 11, payload, wsCreateBatchOrderEPL)
	require.ErrorIs(t, err, request.ErrWeightExceedsBudget, "sendWebsocketBatchOrderRequest must not send a batch of more orders than its budget admits")
	assert.Len(t, m.received(), 1, "sendWebsocketBatchOrderRequest should send nothing when the rate limit refuses the batch's orders")

	m.setReply(func(req *mockTradeRequest) (string, string) {
		return strings.Replace(batchReply(req, "1740453408555"), `{"code":0,"msg":"OK"},`, "", 1), ""
	})
	got, err = ex.sendWebsocketBatchOrderRequest(t.Context(), "order.create-batch", cLinear, 2, payload, wsCreateBatchOrderEPL)
	require.ErrorIs(t, err, errBatchOutcomeMismatch, "sendWebsocketBatchOrderRequest must not report results it cannot pair with outcomes")
	assert.Nil(t, got, "sendWebsocketBatchOrderRequest should return no results it cannot pair with outcomes")
	m.setReply(func(req *mockTradeRequest) (string, string) {
		return strings.Replace(batchReply(req, "1740453408555"), `{"code":0,"msg":"OK"},`, `{"code":0,"msg":"OK"},{"code":0,"msg":"OK"},`, 1), ""
	})
	got, err = ex.sendWebsocketBatchOrderRequest(t.Context(), "order.create-batch", cLinear, 2, payload, wsCreateBatchOrderEPL)
	require.ErrorIs(t, err, errBatchOutcomeMismatch, "sendWebsocketBatchOrderRequest must not report results when Bybit sends more outcomes than results")
	assert.Nil(t, got, "sendWebsocketBatchOrderRequest should return no results when Bybit sends more outcomes than results")
	// An outcome for each order does not make up for a missing result
	m.setReply(func(req *mockTradeRequest) (string, string) {
		return strings.Replace(batchReply(req, "1740453408555"), `,{"category":"linear","symbol":"BTCUSDT","orderId":"","orderLinkId":"batch-001","createAt":""}`, "", 1), ""
	})
	got, err = ex.sendWebsocketBatchOrderRequest(t.Context(), "order.create-batch", cLinear, 2, payload, wsCreateBatchOrderEPL)
	require.ErrorIs(t, err, errBatchOutcomeMismatch, "sendWebsocketBatchOrderRequest must not report results when Bybit sends fewer results than orders")
	assert.Nil(t, got, "sendWebsocketBatchOrderRequest should return no results when Bybit sends fewer results than orders")
	// The reply holds two results and two outcomes, which cannot be matched to a batch of any other size
	m.setReply(func(req *mockTradeRequest) (string, string) { return batchReply(req, "1740453408555"), "" })
	for _, orders := range []int{1, 3} {
		got, err = ex.sendWebsocketBatchOrderRequest(t.Context(), "order.create-batch", cLinear, orders, payload, wsCreateBatchOrderEPL)
		require.ErrorIsf(t, err, errBatchOutcomeMismatch, "sendWebsocketBatchOrderRequest must not report two results for a batch of %d orders", orders)
		assert.Nilf(t, got, "sendWebsocketBatchOrderRequest should return no results for a batch of %d orders", orders)
	}

	m.setReply(func(req *mockTradeRequest) (string, string) {
		return `{"reqId":"` + req.RequestID + `","retCode":10404,"retMsg":"category is not correct/supported","op":"order.create-batch","connId":"cupviqn88smf24t2kpb0-536o"}`, ""
	})
	got, err = ex.sendWebsocketBatchOrderRequest(t.Context(), "order.create-batch", cLinear, 2, payload, wsCreateBatchOrderEPL)
	require.ErrorIs(t, err, errAPIResponse, "sendWebsocketBatchOrderRequest must report Bybit's error")
	assert.Nil(t, got, "sendWebsocketBatchOrderRequest should return no results for a refused request")
}

func TestWebsocketTradeRequestHeaderJSON(t *testing.T) {
	t.Parallel()
	data, err := json.Marshal(&WebsocketTradeRequestHeader{Timestamp: "1711001595207", ReceiveWindow: "8000", Referer: "broker-001"})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"X-BAPI-TIMESTAMP":"1711001595207","X-BAPI-RECV-WINDOW":"8000","Referer":"broker-001"}`, string(data), "WebsocketTradeRequestHeader should marshal to Bybit's header names")
	data, err = json.Marshal(&WebsocketTradeRequestHeader{Timestamp: "1711001595207"})
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, `{"X-BAPI-TIMESTAMP":"1711001595207"}`, string(data), "WebsocketTradeRequestHeader should leave out an unset receive window and referer")
}

func TestTimedTradeRequestStampSendTime(t *testing.T) {
	t.Parallel()
	newRequest := func(timestamp string) *WebsocketTradeRequest {
		return &WebsocketTradeRequest{
			RequestID: "test-001",
			Header:    &WebsocketTradeRequestHeader{Timestamp: timestamp, ReceiveWindow: "8000"},
			Operation: "order.create",
			Arguments: []any{map[string]any{"category": cLinear}},
		}
	}
	r := newRequest("1711001595207")
	assert.Equal(t, newRequest("1711001600123"), timedTradeRequest{r}.StampSendTime(time.Unix(0, 1711001600123456789)), "StampSendTime should return the request stamped with its send time in milliseconds")
	assert.Equal(t, newRequest("1711001595207"), r, "StampSendTime should leave the request unchanged")
	// A connection that does not stamp requests sends the timed request as it is
	timed, err := json.Marshal(timedTradeRequest{r})
	require.NoError(t, err, "Marshal must not error")
	plain, err := json.Marshal(r)
	require.NoError(t, err, "Marshal must not error")
	assert.JSONEq(t, string(plain), string(timed), "a timed request should marshal as the request it holds")
	_, stamped := any(r).(websocket.SendTimeStamper)
	assert.False(t, stamped, "WebsocketTradeRequest should not implement SendTimeStamper, so that a type embedding it is sent whole")
}

func TestSendWebsocketTradeRequest(t *testing.T) {
	t.Parallel()
	conn := &FixtureConnection{sendMessageReturnResponseOverride: []byte(`{"reqId":"1","retCode":0,"retMsg":"OK","op":"order.create","data":{"orderId":"a4c1718e-fe53-4659-a118-1f6ecce04ad9","orderLinkId":"test-005"},"retExtInfo":{},"header":{"X-Bapi-Limit":"10","X-Bapi-Limit-Status":"9","X-Bapi-Limit-Reset-Timestamp":"1711001595208","Traceid":"38b7977b430f9bd228f4b19724794dfd","Timenow":"1711001595209"},"connId":"cnt5leec0hvan15eukcg-2v"}`)}
	resp, err := sendWebsocketTradeRequest[OrderResponse](t.Context(), e, conn, "order.create", map[string]any{"category": cLinear}, wsCreateOrderEPL)
	require.NoError(t, err, "sendWebsocketTradeRequest must not error")
	exp := &WebsocketTradeResponse[OrderResponse]{
		RequestID:     "1",
		ReturnMessage: "OK",
		Operation:     "order.create",
		Data:          OrderResponse{OrderID: "a4c1718e-fe53-4659-a118-1f6ecce04ad9", OrderLinkID: "test-005"},
		Header: WebsocketTradeResponseHeader{
			TraceID:             "38b7977b430f9bd228f4b19724794dfd",
			TimeNow:             types.Time(time.UnixMilli(1711001595209)),
			Limit:               10,
			LimitStatus:         9,
			LimitResetTimestamp: types.Time(time.UnixMilli(1711001595208)),
		},
		ConnectionID: "cnt5leec0hvan15eukcg-2v",
	}
	assert.Equal(t, exp, resp, "sendWebsocketTradeRequest should decode every reply field")
	require.Len(t, conn.sent, 1, "sendWebsocketTradeRequest must send one request")
	sent, ok := conn.sent[0].(timedTradeRequest)
	require.True(t, ok, "sendWebsocketTradeRequest must send a request the trade connection stamps when it sends it")
	req := sent.WebsocketTradeRequest
	assert.Equal(t, "order.create", req.Operation, "request operation should be kept")
	assert.NotEmpty(t, req.RequestID, "request should carry a request ID")
	require.NotNil(t, req.Header, "request must carry headers")
	assert.NotEmpty(t, req.Header.Timestamp, "request should carry a timestamp header")
	assert.Equal(t, []any{map[string]any{"category": cLinear}}, req.Arguments, "request should carry its argument")

	_, err = sendWebsocketTradeRequest[OrderResponse](t.Context(), e, &FixtureConnection{reply: func(any) ([]byte, error) { return nil, errors.New("connection lost") }}, "order.create", nil, wsCreateOrderEPL)
	require.ErrorContains(t, err, "connection lost", "sendWebsocketTradeRequest must return a failed send")
	_, err = sendWebsocketTradeRequest[OrderResponse](t.Context(), e, &FixtureConnection{sendMessageReturnResponseOverride: []byte(`{"retCode":"0"}`)}, "order.create", nil, wsCreateOrderEPL)
	require.Error(t, err, "sendWebsocketTradeRequest must error for an undecodable reply")
}

func TestSendWebsocketTradeRequestStampsWhenSent(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	stamps := make(map[string]string)
	m := &tradeStreamMock{reply: func(req *mockTradeRequest) (string, string) {
		mu.Lock()
		stamps[string(req.Arguments[0])] = req.Header["X-BAPI-TIMESTAMP"]
		mu.Unlock()
		return orderReply(req, `{"orderId":"c1956690-b731-4191-97c0-94b00422231b","orderLinkId":""}`), ""
	}}
	ex := newTradeStreamExchange(t, m)
	conn, err := ex.Websocket.GetConnection(OutboundTradeConnection)
	require.NoError(t, err, "GetConnection must not error")
	// A rate limit barrier holds the first request back until the second joins it, as a rate limit wait would. Their
	// endpoint has no budget, so they take nothing from the order budgets other tests draw on. The deadline ends both
	// requests should the barrier never admit them
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	contexts, err := request.NewRateLimitBarrierContexts(ctx, 2)
	require.NoError(t, err, "NewRateLimitBarrierContexts must not error")
	held := make(chan error, 1)
	go func() {
		_, err := sendWebsocketTradeRequest[OrderResponse](contexts[0], ex, conn, "order.create", "held", wsSubscriptionEPL)
		held <- err
	}()
	time.Sleep(100 * time.Millisecond)
	joined := time.Now()
	_, err = sendWebsocketTradeRequest[OrderResponse](contexts[1], ex, conn, "order.create", "joining", wsSubscriptionEPL)
	require.NoError(t, err, "sendWebsocketTradeRequest must not error")
	require.NoError(t, <-held, "sendWebsocketTradeRequest must not error for the held request")
	mu.Lock()
	defer mu.Unlock()
	stamp, err := strconv.ParseInt(stamps[`"held"`], 10, 64)
	require.NoError(t, err, "the held request's timestamp must be Unix milliseconds")
	assert.GreaterOrEqual(t, stamp, joined.UnixMilli(), "a request held back by its rate limit should be stamped when it is sent, not when it was made")
}

func TestWebsocketSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketSubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketSubmitOrder must reject a nil order")
	_, err = e.WebsocketSubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.Margin, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "WebsocketSubmitOrder must refuse an asset websocket order entry does not trade before sending")
	s := &order.Submit{Exchange: e.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Buy, Type: order.Market, Amount: 0.0001}
	_, err = e.WebsocketSubmitOrder(t.Context(), s)
	require.ErrorIs(t, err, order.ErrAmountMustBeSet, "WebsocketSubmitOrder must require a quote amount for a spot market buy")
	_, err = e.WebsocketSubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Price: 82650, Amount: 0.0001})
	require.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketSubmitOrder must not fall back to REST without a trade connection")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ex := getWebsocketInstance(t)
		s.Type = order.Limit
		s.Price = 55000
		got, err := ex.WebsocketSubmitOrder(t.Context(), s)
		require.NoError(t, err, "WebsocketSubmitOrder must not error")
		assert.NotEmpty(t, got.OrderID, "WebsocketSubmitOrder should return the order ID")
		return
	}
	m := &tradeStreamMock{reply: func(req *mockTradeRequest) (string, string) {
		return orderReply(req, `{"orderId":"2290000000123456789","orderLinkId":"spot-buy-001"}`), ""
	}}
	ex := newTradeStreamExchange(t, m)
	s.Exchange = ex.Name
	s.Type = order.Limit
	s.Price = 82650
	s.ClientOrderID = "spot-buy-001"
	got, err := ex.WebsocketSubmitOrder(t.Context(), s)
	require.NoError(t, err, "WebsocketSubmitOrder must not error")
	assert.Equal(t, "2290000000123456789", got.OrderID, "WebsocketSubmitOrder should return the order ID")
	assert.Equal(t, "spot-buy-001", got.ClientOrderID, "WebsocketSubmitOrder should return the client order ID")
	assert.Equal(t, order.New, got.Status, "WebsocketSubmitOrder should report a new order, as SubmitOrder does")
	requests := m.received()
	require.Len(t, requests, 1, "WebsocketSubmitOrder must send one request over the trade connection")
	assert.Equal(t, "order.create", requests[0].Operation, "WebsocketSubmitOrder should create the order over the trade connection")
	// A limit order with a trigger price is placed as a conditional order, and reported as the stop limit order it is
	conditional := *s
	conditional.TriggerPrice = 90000
	got, err = ex.WebsocketSubmitOrder(t.Context(), &conditional)
	require.NoError(t, err, "WebsocketSubmitOrder must not error for a limit order with a trigger price")
	assert.Equal(t, order.StopLimit, got.Type, "WebsocketSubmitOrder should report a limit order with a trigger price as a stop limit order")
	// A market order is not filled when Bybit accepts it, so it is reported as a new order too
	market := *s
	market.Type = order.Market
	market.Side = order.Sell
	market.Price = 0
	got, err = ex.WebsocketSubmitOrder(t.Context(), &market)
	require.NoError(t, err, "WebsocketSubmitOrder must not error for a market order")
	assert.Equal(t, order.New, got.Status, "WebsocketSubmitOrder should report a market order as a new order")
	// Bybit requires a client order ID on every options order, so WebsocketSubmitOrder returns the one it generated and sent
	option := &order.Submit{Exchange: ex.Name, Pair: currency.NewPairWithDelimiter("BTC", "26NOV24-92000-C", "-"), AssetType: asset.Options, Side: order.Sell, Type: order.Limit, Price: 500, Amount: 0.01}
	got, err = ex.WebsocketSubmitOrder(t.Context(), option)
	require.NoError(t, err, "WebsocketSubmitOrder must not error for an options order without a client order ID")
	requests = m.received()
	sent := requestArgument(t, requests[len(requests)-1])["orderLinkId"]
	assert.NotEmpty(t, sent, "WebsocketSubmitOrder should send a client order ID with an options order")
	assert.Equal(t, sent, got.ClientOrderID, "WebsocketSubmitOrder should return the client order ID it sent")

	// Bybit refuses an order the account cannot pay for
	m.setReply(func(req *mockTradeRequest) (string, string) {
		return `{"reqId":"` + req.RequestID + `","retCode":170131,"retMsg":"Balance insufficient","op":"order.create","connId":"cnt5leec0hvan15eukcg-2v"}`, ""
	})
	_, err = ex.WebsocketSubmitOrder(t.Context(), s)
	require.ErrorIs(t, err, errAPIResponse, "WebsocketSubmitOrder must return Bybit's refusal")
	assert.ErrorContains(t, err, "Balance insufficient", "WebsocketSubmitOrder should return Bybit's reason")
	m.setReply(func(req *mockTradeRequest) (string, string) {
		return orderReply(req, `{"orderId":"","orderLinkId":"spot-buy-001"}`), ""
	})
	_, err = ex.WebsocketSubmitOrder(t.Context(), s)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "WebsocketSubmitOrder must not report an order accepted without Bybit's order ID")
}

func TestWebsocketModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketModifyOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketModifyOrder must reject a nil modification")
	_, err = e.WebsocketModifyOrder(t.Context(), &order.Modify{Pair: currency.NewBTCUSDT(), AssetType: asset.Margin, OrderID: "1"})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "WebsocketModifyOrder must refuse an asset websocket order entry does not trade before sending")
	mod := &order.Modify{Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, Amount: 0.5, OrderID: "c1956690-b731-4191-97c0-94b00422231b"}
	_, err = e.WebsocketModifyOrder(t.Context(), &order.Modify{Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, Amount: 0.5})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "WebsocketModifyOrder must require an order ID or client order ID")
	_, err = e.WebsocketModifyOrder(t.Context(), mod)
	require.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketModifyOrder must not fall back to REST without a trade connection")
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ex := getWebsocketInstance(t)
		got, err := ex.WebsocketModifyOrder(t.Context(), mod)
		require.NoError(t, err, "WebsocketModifyOrder must not error")
		assert.NotEmpty(t, got.OrderID, "WebsocketModifyOrder should return the order ID")
		return
	}
	m := &tradeStreamMock{reply: func(req *mockTradeRequest) (string, string) {
		return orderReply(req, `{"orderId":"c1956690-b731-4191-97c0-94b00422231b","orderLinkId":"linear-tp-001"}`), ""
	}}
	ex := newTradeStreamExchange(t, m)
	got, err := ex.WebsocketModifyOrder(t.Context(), mod)
	require.NoError(t, err, "WebsocketModifyOrder must not error")
	assert.Equal(t, "c1956690-b731-4191-97c0-94b00422231b", got.OrderID, "WebsocketModifyOrder should return the order ID")
	assert.Equal(t, 0.5, got.Amount, "WebsocketModifyOrder should return the requested amount")
	requests := m.received()
	require.Len(t, requests, 1, "WebsocketModifyOrder must send one request over the trade connection")
	assert.Equal(t, "order.amend", requests[0].Operation, "WebsocketModifyOrder should amend the order over the trade connection")

	// An amendment by client order ID alone takes the order ID from Bybit's acceptance
	got, err = ex.WebsocketModifyOrder(t.Context(), &order.Modify{Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures, Amount: 0.5, ClientOrderID: "linear-tp-001"})
	require.NoError(t, err, "WebsocketModifyOrder must not error for an amendment by client order ID")
	assert.Equal(t, "c1956690-b731-4191-97c0-94b00422231b", got.OrderID, "WebsocketModifyOrder should return the order ID Bybit amended")

	// A filled order can no longer be amended
	m.setReply(func(req *mockTradeRequest) (string, string) {
		return `{"reqId":"` + req.RequestID + `","retCode":110001,"retMsg":"Order does not exist","op":"order.amend","connId":"cnt5leec0hvan15eukcg-2v"}`, ""
	})
	_, err = ex.WebsocketModifyOrder(t.Context(), mod)
	require.ErrorIs(t, err, order.ErrOrderNotFound, "WebsocketModifyOrder must report an order Bybit cannot find")
}

func TestWebsocketCancelOrder(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.WebsocketCancelOrder(t.Context(), nil), common.ErrNilPointer, "WebsocketCancelOrder must reject a nil cancellation")
	err := e.WebsocketCancelOrder(t.Context(), &order.Cancel{OrderID: "1", Pair: currency.NewBTCUSDT(), AssetType: asset.Margin})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "WebsocketCancelOrder must refuse an asset websocket order entry does not trade before sending")
	cancel := &order.Cancel{OrderID: "2290000000123456789", Pair: currency.NewBTCUSDT(), AssetType: asset.Spot}
	err = e.WebsocketCancelOrder(t.Context(), &order.Cancel{Pair: currency.NewBTCUSDT(), AssetType: asset.Spot})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "WebsocketCancelOrder must require an order ID or client order ID")
	require.ErrorIs(t, e.WebsocketCancelOrder(t.Context(), cancel), websocket.ErrNotConnected, "WebsocketCancelOrder must not fall back to REST without a trade connection")
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ex := getWebsocketInstance(t)
		require.NoError(t, ex.WebsocketCancelOrder(t.Context(), cancel), "WebsocketCancelOrder must not error")
		return
	}
	m := &tradeStreamMock{reply: func(req *mockTradeRequest) (string, string) {
		return orderReply(req, `{"orderId":"2290000000123456789","orderLinkId":"spot-buy-001"}`), ""
	}}
	ex := newTradeStreamExchange(t, m)
	require.NoError(t, ex.WebsocketCancelOrder(t.Context(), cancel), "WebsocketCancelOrder must not error")
	requests := m.received()
	require.Len(t, requests, 1, "WebsocketCancelOrder must send one request over the trade connection")
	assert.Equal(t, "order.cancel", requests[0].Operation, "WebsocketCancelOrder should cancel the order over the trade connection")

	// A filled order can no longer be cancelled
	m.setReply(func(req *mockTradeRequest) (string, string) {
		return `{"reqId":"` + req.RequestID + `","retCode":110001,"retMsg":"Order does not exist","op":"order.cancel","connId":"cnt5leec0hvan15eukcg-2v"}`, ""
	})
	require.ErrorIs(t, ex.WebsocketCancelOrder(t.Context(), cancel), order.ErrOrderNotFound, "WebsocketCancelOrder must report an order Bybit cannot find")

	// A spot stop order Bybit cannot find among TP/SL orders is looked for among conditional orders
	var mu sync.Mutex
	var filters []string
	m.setReply(func(req *mockTradeRequest) (string, string) {
		var arg struct {
			OrderFilter string `json:"orderFilter"`
		}
		assert.NoError(t, json.Unmarshal(req.Arguments[0], &arg), "decoding the cancellation should not error")
		mu.Lock()
		filters = append(filters, arg.OrderFilter)
		mu.Unlock()
		if arg.OrderFilter == "StopOrder" {
			return orderReply(req, `{"orderId":"2290000000123456790","orderLinkId":""}`), ""
		}
		return `{"reqId":"` + req.RequestID + `","retCode":170213,"retMsg":"Order does not exist.","op":"order.cancel","connId":"cnt5leec0hvan15eukcg-2v"}`, ""
	})
	stop := &order.Cancel{OrderID: "2290000000123456790", Pair: currency.NewBTCUSDT(), AssetType: asset.Spot, Type: order.StopMarket}
	require.NoError(t, ex.WebsocketCancelOrder(t.Context(), stop), "WebsocketCancelOrder must cancel a spot conditional order")
	mu.Lock()
	assert.Equal(t, []string{"tpslOrder", "StopOrder"}, filters, "WebsocketCancelOrder should look for a spot stop order among TP/SL orders and then conditional orders")
	mu.Unlock()
}

// getWebsocketInstance returns a websocket instance copy for live bi-directional testing
func getWebsocketInstance(t *testing.T) *Exchange {
	t.Helper()
	cfg := &config.Config{}
	root, err := testutils.RootPathFromCWD()
	require.NoError(t, err, "RootPathFromCWD must not error")
	require.NoError(t, cfg.LoadConfig(filepath.Join(root, "testdata", "configtest.json"), true), "LoadConfig must not error")
	pairs := &e.CurrencyPairs
	ex := new(Exchange)
	ex.SetDefaults()
	bConf, err := cfg.GetExchangeConfig("Bybit")
	require.NoError(t, err, "GetExchangeConfig must not error")
	bConf.API.AuthenticatedSupport = true
	bConf.API.AuthenticatedWebsocketSupport = true
	bConf.API.Credentials.Key = apiCredentials.Key
	bConf.API.Credentials.Secret = apiCredentials.Secret
	require.NoError(t, ex.Setup(bConf), "Setup must not error")
	ex.CurrencyPairs.Load(pairs)
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	return ex
}
