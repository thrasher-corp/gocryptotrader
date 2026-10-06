package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"net/http/httptest"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// wsAPIFixture is a WebSocket API method's recorded response to the parameters that request it
type wsAPIFixture struct {
	Signed bool            `json:"signed"`
	Params map[string]any  `json:"params"`
	Status int64           `json:"status"`
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
}

var loadWsAPIFixtures = sync.OnceValues(func() (map[string][]wsAPIFixture, error) {
	raw, err := os.ReadFile("testdata/wsSpotAPI.json")
	if err != nil {
		return nil, err
	}
	var fixtures map[string][]wsAPIFixture
	return fixtures, json.Unmarshal(raw, &fixtures)
})

// wsAPIMock answers each WebSocket API request with the fixture whose parameters equal the request's, after checking
// that a SIGNED request carries the placeholder API key, a timestamp and a valid signature over its other parameters
func wsAPIMock(tb testing.TB, msg []byte, w *gws.Conn) error {
	tb.Helper()
	fixtures, err := loadWsAPIFixtures()
	require.NoError(tb, err, "loadWsAPIFixtures must not error")
	var req struct {
		ID     string         `json:"id"`
		Method string         `json:"method"`
		Params map[string]any `json:"params"`
	}
	require.NoError(tb, json.Unmarshal(msg, &req), "request must decode")
	params := maps.Clone(req.Params)
	if params == nil {
		params = map[string]any{}
	}
	signature, signed := params["signature"]
	if signed {
		delete(params, "signature")
		payload, err := wsParams(params).signaturePayload()
		require.NoError(tb, err, "signaturePayload must not error")
		mac := hmac.New(sha256.New, []byte("mock-api-secret"))
		mac.Write([]byte(payload))
		assert.Equalf(tb, hex.EncodeToString(mac.Sum(nil)), signature, "%s signature should cover every other parameter", req.Method)
		assert.Equalf(tb, "mock-api-key", params["apiKey"], "%s should send the API key", req.Method)
		assert.Positivef(tb, params["timestamp"], "%s should send a timestamp", req.Method)
		delete(params, "apiKey")
		delete(params, "timestamp")
	}
	for _, f := range fixtures[req.Method] {
		fixtureParams := f.Params
		if fixtureParams == nil {
			fixtureParams = map[string]any{}
		}
		if f.Signed != signed || !reflect.DeepEqual(fixtureParams, params) {
			continue
		}
		resp := map[string]any{
			"id":         req.ID,
			"status":     200,
			"rateLimits": []map[string]any{{"rateLimitType": "REQUEST_WEIGHT", "interval": "MINUTE", "intervalNum": 1, "limit": 6000, "count": 2}},
		}
		if f.Status != 0 {
			resp["status"] = f.Status
		}
		if len(f.Error) != 0 {
			resp["error"] = f.Error
		} else {
			resp["result"] = f.Result
		}
		data, err := json.Marshal(resp)
		require.NoError(tb, err, "response must encode")
		return w.WriteMessage(gws.TextMessage, data)
	}
	tb.Errorf("no %s fixture matches signed %t parameters %v", req.Method, signed, params)
	return w.WriteMessage(gws.TextMessage, fmt.Appendf(nil, `{"id":%q,"status":400,"error":{"code":-1,"msg":"no fixture"}}`, req.ID))
}

// wsAPIFixtureResult decodes the result of a method's fixture into the type the method returns. Methods returning a
// type they share with REST endpoints are compared with it, since the type's REST tests compare every decoded field
func wsAPIFixtureResult[T any](tb testing.TB, method string, index int) T {
	tb.Helper()
	fixtures, err := loadWsAPIFixtures()
	require.NoError(tb, err, "loadWsAPIFixtures must not error")
	require.Lessf(tb, index, len(fixtures[method]), "%s must have fixture %d", method, index)
	var result T
	require.NoErrorf(tb, json.Unmarshal(fixtures[method][index].Result, &result), "%s fixture %d result must decode", method, index)
	return result
}

// wsAPITimeRange returns the time range the fixtures were recorded with, or the last hour in live tests
func wsAPITimeRange() (start, end time.Time) {
	if mockTests {
		return time.UnixMilli(1744103854944), time.UnixMilli(1744190254944)
	}
	end = time.Now()
	return end.Add(-time.Hour), end
}

// newWsAPITestExchange returns an exchange connected only to the WebSocket API, which the fixtures serve in mock tests
// and Binance in live tests. Live tests of SIGNED methods skip before connecting without credentials, or without
// permission to manipulate orders when canManipulateOrders is passed
func newWsAPITestExchange(tb testing.TB, signed bool, canManipulateOrders ...bool) *Exchange {
	tb.Helper()
	ex := newTestExchange(tb)
	if signed && !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(tb, ex, canManipulateOrders...)
	}
	setup := ex.spotWebsocketAPIConnectionSetup(ex.Config)
	if mockTests {
		useMockWebsocket(tb, ex, setup, wsAPIMock)
	} else {
		useWebsocket(tb, ex, setup)
	}
	require.NoError(tb, ex.Websocket.Connect(tb.Context()), "Connect must not error")
	return ex
}

// userDataAssets returns the assets of the exchange's User Data Stream subscriptions by subscription ID
func userDataAssets(ex *Exchange) map[uint64]asset.Item {
	ex.userData.mu.Lock()
	defer ex.userData.mu.Unlock()
	return maps.Clone(ex.userData.subscriptions)
}

func TestSignWsAPIParams(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ex.SkipAuthCheck = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	params := wsParams{
		"symbol": "BTCUSDT", "side": "SELL", "type": "LIMIT", "timeInForce": "GTC", "quantity": "0.01000000",
		"price": "52000.00", "recvWindow": uint64(100), "symbols": []string{"BTCUSDT", "ETHUSDT"}, "omitZeroBalances": true,
	}
	require.NoError(t, ex.signWsAPIParams(t.Context(), params), "signWsAPIParams must not error")
	timestamp, ok := params["timestamp"].(int64)
	require.True(t, ok, "timestamp must be an integer")
	assert.Equal(t, "mock-api-key", params["apiKey"], "apiKey should be the API key")
	payload := "apiKey=mock-api-key&omitZeroBalances=true&price=52000.00&quantity=0.01000000&recvWindow=100&side=SELL" +
		`&symbol=BTCUSDT&symbols=["BTCUSDT","ETHUSDT"]&timeInForce=GTC&timestamp=` + strconv.FormatInt(timestamp, 10) + "&type=LIMIT"
	mac := hmac.New(sha256.New, []byte("mock-api-secret"))
	mac.Write([]byte(payload))
	assert.Equal(t, hex.EncodeToString(mac.Sum(nil)), params["signature"], "signature should sign the parameters sorted by name")

	// Signing the same parameters again, as a retry does, must not sign the earlier signature
	require.NoError(t, ex.signWsAPIParams(t.Context(), params), "signWsAPIParams must not error signing again")
	unsigned := maps.Clone(params)
	delete(unsigned, "signature")
	payload, err := unsigned.signaturePayload()
	require.NoError(t, err, "signaturePayload must not error")
	mac = hmac.New(sha256.New, []byte("mock-api-secret"))
	mac.Write([]byte(payload))
	assert.Equal(t, hex.EncodeToString(mac.Sum(nil)), params["signature"], "signing again should sign every parameter but the signature")

	ex.SkipAuthCheck = false
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{})
	assert.ErrorIs(t, ex.signWsAPIParams(t.Context(), wsParams{}), exchange.ErrCredentialsAreEmpty, "signWsAPIParams should error without credentials")

	_, err = wsParams{"invalid": func() {}}.signaturePayload()
	assert.ErrorIs(t, err, errWebsocketAPIRequestFailed, "signaturePayload should reject a value JSON cannot encode")
}

func TestRateLimitAndSignWsAPIParams(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ex.SkipAuthCheck = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	params := wsParams{"symbol": "BTCUSDT"}
	require.NoError(t, ex.rateLimitAndSignWsAPIParams(t.Context(), spotOrderRate, params), "rateLimitAndSignWsAPIParams must not error")
	assert.NotEmpty(t, params["signature"], "rateLimitAndSignWsAPIParams should sign the parameters")

	params = wsParams{"symbol": "BTCUSDT"}
	err := ex.rateLimitAndSignWsAPIParams(t.Context(), request.Unset, params)
	assert.ErrorIs(t, err, common.ErrNilPointer, "rateLimitAndSignWsAPIParams should return the error of a limit without a limiter")
	assert.NotContains(t, params, "signature", "rateLimitAndSignWsAPIParams should not sign a request it cannot rate limit")

	ex.SkipAuthCheck = false
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{})
	err = ex.rateLimitAndSignWsAPIParams(t.Context(), spotOrderRate, wsParams{})
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "rateLimitAndSignWsAPIParams should return the signing error")
}

// useRateLimitedWsAPI connects an exchange's WebSocket API connection to mock with the rate limits Setup gives it
func useRateLimitedWsAPI(t *testing.T, ex *Exchange, mock mockws.WsMockFunc) {
	t.Helper()
	s := httptest.NewTestServer(t, mockws.CurryWsMockUpgrader(t, mock))
	s.Start()
	setup := ex.spotWebsocketAPIConnectionSetup(ex.Config)
	setup.URL = "ws" + strings.TrimPrefix(s.URL, "http")
	ex.Config.ConnectionMonitorDelay = time.Hour
	ex.Websocket = sharedtestvalues.NewTestWebsocket()
	require.NoError(t, ex.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig:               ex.Config,
		Features:                     &ex.Features.Supports.WebsocketCapabilities,
		UseMultiConnectionManagement: true,
		RateLimitDefinitions:         ex.Requester.GetRateLimiterDefinitions(),
	}), "Websocket Setup must not error")
	ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
	require.NoError(t, ex.Websocket.SetupNewConnection(setup), "SetupNewConnection must not error")
	t.Cleanup(func() {
		if ex.Websocket.IsConnected() {
			assert.NoError(t, ex.Websocket.Shutdown(), "Shutdown should not error")
		}
	})
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
}

// orderPlacedMock answers order.place requests with a placed order
func orderPlacedMock(tb testing.TB, msg []byte, w *gws.Conn) error {
	tb.Helper()
	var req WsAPIRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return err
	}
	return w.WriteMessage(gws.TextMessage, fmt.Appendf(nil, `{"id":%q,"status":200,"result":{"symbol":"BTCUSDT","orderId":1,"orderListId":-1,"clientOrderId":"x","transactTime":1791284980615}}`, req.ID))
}

func TestSendSignedWsAPIRequestSignsAfterRateLimit(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	// The server rejects a timestamp older than a one second recvWindow on arrival, as Binance does with -1021, while
	// the order pool admits ten orders a second, so the queue of orders outlasts the recvWindow
	const recvWindow = time.Second
	var mu sync.Mutex
	var first, last time.Time
	useRateLimitedWsAPI(t, ex, func(tb testing.TB, msg []byte, w *gws.Conn) error {
		tb.Helper()
		var req WsAPIRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			return err
		}
		timestamp, ok := req.Params["timestamp"].(float64)
		if !ok {
			return fmt.Errorf("no timestamp in %s", msg)
		}
		now := time.Now()
		mu.Lock()
		if first.IsZero() {
			first = now
		}
		last = now
		mu.Unlock()
		if now.Sub(time.UnixMilli(int64(timestamp))) > recvWindow {
			return w.WriteMessage(gws.TextMessage, fmt.Appendf(nil, `{"id":%q,"status":400,"error":{"code":-1021,"msg":"Timestamp for this request is outside of the recvWindow."}}`, req.ID))
		}
		return orderPlacedMock(tb, msg, w)
	})
	ex.SkipAuthCheck = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})

	const orders = 25
	errs := make(chan error, orders)
	var wg sync.WaitGroup
	for range orders {
		wg.Go(func() {
			_, err := ex.WsPlaceNewOrder(t.Context(), &WsPlaceOrderRequest{Symbol: currency.NewBTCUSDT(), Side: "BUY", Type: "LIMIT", TimeInForce: "GTC", Price: 1, Quantity: 1})
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		assert.NoError(t, err, "WsPlaceNewOrder should sign each order after its wait, within the recvWindow")
	}
	mu.Lock()
	defer mu.Unlock()
	assert.Greater(t, last.Sub(first), recvWindow, "the orders should have queued for longer than the recvWindow")
}

func TestSendSignedWsAPIRequestChargesRateLimitOnce(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	useRateLimitedWsAPI(t, ex, orderPlacedMock)
	ex.SkipAuthCheck = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	// A fresh order pool admits one order at once, so charging an order twice would need a delay
	_, err := ex.WsPlaceNewOrder(request.WithDelayNotAllowed(t.Context()), &WsPlaceOrderRequest{Symbol: currency.NewBTCUSDT(), Side: "BUY", Type: "LIMIT", TimeInForce: "GTC", Price: 1, Quantity: 1})
	require.NoError(t, err, "WsPlaceNewOrder must charge the order pool once")
	_, err = ex.WsPlaceNewOrder(request.WithDelayNotAllowed(t.Context()), &WsPlaceOrderRequest{Symbol: currency.NewBTCUSDT(), Side: "BUY", Type: "LIMIT", TimeInForce: "GTC", Price: 1, Quantity: 1})
	assert.ErrorIs(t, err, request.ErrDelayNotAllowed, "WsPlaceNewOrder should wait for the order pool before a second order")
}

func TestSendSignedWsAPIRequestCancelled(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	var received atomic.Int32
	useRateLimitedWsAPI(t, ex, func(tb testing.TB, msg []byte, w *gws.Conn) error {
		tb.Helper()
		received.Add(1)
		return orderPlacedMock(tb, msg, w)
	})
	ex.SkipAuthCheck = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := ex.WsPlaceNewOrder(ctx, &WsPlaceOrderRequest{Symbol: currency.NewBTCUSDT(), Side: "BUY", Type: "LIMIT", TimeInForce: "GTC", Price: 1, Quantity: 1})
	assert.ErrorIs(t, err, context.Canceled, "WsPlaceNewOrder should return the cancellation of an order cancelled before it was sent")
	// The server answers requests in order, so a cancelled order sent before this one would be counted by its reply
	_, err = ex.WsPlaceNewOrder(t.Context(), &WsPlaceOrderRequest{Symbol: currency.NewBTCUSDT(), Side: "BUY", Type: "LIMIT", TimeInForce: "GTC", Price: 1, Quantity: 1})
	require.NoError(t, err, "WsPlaceNewOrder must not error")
	assert.Equal(t, int32(1), received.Load(), "an order cancelled before it was sent should not reach the server")
}

func TestUserDataStreamSubscriptionLateResponse(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ex.SkipAuthCheck = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	received, release := make(chan struct{}, 1), make(chan struct{})
	var releaseOnce sync.Once
	releaseResponse := func() { releaseOnce.Do(func() { close(release) }) }
	useMockWebsocket(t, ex, ex.spotWebsocketAPIConnectionSetup(ex.Config), func(tb testing.TB, msg []byte, w *gws.Conn) error {
		tb.Helper()
		var req WsAPIRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			return err
		}
		select {
		case received <- struct{}{}:
		default:
		}
		<-release
		return w.WriteMessage(gws.TextMessage, fmt.Appendf(nil, `{"id":%q,"status":200,"result":{"subscriptionId":7}}`, req.ID))
	})
	t.Cleanup(releaseResponse)
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	ctx, cancel := context.WithCancel(t.Context())
	errs := make(chan error, 1)
	go func() {
		_, err := ex.WsSubscribeUserDataStream(ctx)
		errs <- err
	}()
	<-received
	cancel()
	assert.ErrorIs(t, <-errs, context.Canceled, "WsSubscribeUserDataStream should return the cancellation")
	// Binance made the subscription, so its events must still find their account
	releaseResponse()
	assert.Eventually(t, func() bool {
		a, ok := ex.userData.asset(7)
		return ok && a == asset.Spot
	}, 5*time.Second, 10*time.Millisecond, "a subscription answered after its request was cancelled should be recorded")
}

func TestUserDataStreamsEndWithConnection(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ex.SkipAuthCheck = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	useMockWebsocket(t, ex, ex.spotWebsocketAPIConnectionSetup(ex.Config), wsAPIMock)
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	pending := func() int {
		ex.userData.mu.Lock()
		defer ex.userData.mu.Unlock()
		return len(ex.userData.pending)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := ex.WsSubscribeUserDataStreamWithListenToken(ctx, "listen-token")
	require.ErrorIs(t, err, context.Canceled, "WsSubscribeUserDataStreamWithListenToken must return the cancellation")
	assert.Zero(t, pending(), "a subscription request cancelled before it was sent should not be pending")

	// The connection reconnects without authenticating, so only the new connection can end the last one's state
	ex.userData.expect("answered", asset.Spot)
	ex.userData.recordResponse("answered", []byte(`{"id":"answered","status":200,"result":{"subscriptionId":3}}`))
	ex.userData.expect("unanswered", asset.Margin)
	require.NoError(t, ex.Websocket.Shutdown(), "Shutdown must not error")
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	assert.Zero(t, pending(), "a new connection should end the last one's subscription requests")
	assert.Empty(t, userDataAssets(ex), "a new connection should end the last one's subscriptions")
}

func TestWsParamsSetters(t *testing.T) {
	t.Parallel()
	params := wsParams{}
	params.setString("string", "")
	params.setUint("uint", 0)
	params.setInt("int", 0)
	params.setFloat("float", 0)
	params.setTime("time", time.Time{})
	params.setBool("bool", false)
	assert.Empty(t, params, "unset values should not be sent")

	params.setString("string", "GTC")
	params.setUint("uint", 12569099453)
	params.setInt("int", -1)
	params.setFloat("float", 0.00000001)
	params.setTime("time", time.UnixMilli(1744103854944))
	params.setBool("bool", true)
	exp := wsParams{"string": "GTC", "uint": uint64(12569099453), "int": int64(-1), "float": "0.00000001", "time": int64(1744103854944), "bool": true}
	assert.Equal(t, exp, params, "set values should be sent as the documented JSON types, decimals without exponents")
}

func TestUserDataStreams(t *testing.T) {
	t.Parallel()
	var u userDataStreams
	u.recordResponse("unknown", []byte(`{"id":"unknown","status":200,"result":{"subscriptionId":3}}`))
	_, ok := u.asset(3)
	assert.False(t, ok, "a response to a request that is not a subscription should not record one")

	u.expect("spot", asset.Spot)
	u.expect("margin", asset.Margin)
	u.expect("rejected", asset.Spot)
	u.recordResponse("spot", []byte(`{"id":"spot","status":200,"result":{"subscriptionId":0}}`))
	u.recordResponse("margin", []byte(`{"id":"margin","status":200,"result":{"subscriptionId":1,"expirationTime":1749094553955907}}`))
	u.recordResponse("rejected", []byte(`{"id":"rejected","status":400,"error":{"code":-2015,"msg":"Invalid API-key, IP, or permissions for action."}}`))
	assert.Equal(t, map[uint64]asset.Item{0: asset.Spot, 1: asset.Margin}, u.subscriptions, "successful responses should record their subscriptions")
	assert.Empty(t, u.pending, "responses should end their requests' pending state")

	a, ok := u.asset(0)
	assert.True(t, ok, "subscription 0 should be known")
	assert.Equal(t, asset.Spot, a, "subscription 0 should deliver the spot account's events")
	u.remove(0)
	_, ok = u.asset(0)
	assert.False(t, ok, "remove should forget the subscription")
	u.expect("unanswered", asset.Spot)
	u.reset()
	_, ok = u.asset(1)
	assert.False(t, ok, "reset should forget every subscription")
	assert.Empty(t, u.pending, "reset should end requests still waiting for their responses")
}

func TestSendWsAPIRequest(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	err := ex.SendWsAPIRequest(t.Context(), spotDefaultRate, "session.status", nil, nil)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "SendWsAPIRequest should error without a connection")
	err = ex.SendSignedWsAPIRequest(t.Context(), spotDefaultRate, "account.status", nil, nil)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "SendSignedWsAPIRequest should error without a connection")

	responses := map[string]string{
		"error":      `"status":400,"error":{"code":-1121,"msg":"Invalid symbol."}`,
		"error.data": `"status":418,"error":{"code":-1003,"msg":"Way too much request weight used; IP banned until 1659146400000.","data":{"serverTime":1659142907531,"retryAfter":1659146400000}}`,
		"status":     `"status":429`,
		"null":       `"status":200,"result":null`,
		"result":     `"status":200,"result":{"subscriptionId":4}`,
	}
	useMockWebsocket(t, ex, ex.spotWebsocketAPIConnectionSetup(ex.Config), func(tb testing.TB, msg []byte, w *gws.Conn) error {
		tb.Helper()
		var req WsAPIRequest
		require.NoError(tb, json.Unmarshal(msg, &req), "request must decode")
		return w.WriteMessage(gws.TextMessage, fmt.Appendf(nil, `{"id":%q,%s}`, req.ID, responses[req.Method]))
	})
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	for _, method := range []string{"error", "error.data", "status"} {
		err = ex.SendWsAPIRequest(t.Context(), spotDefaultRate, method, nil, nil)
		assert.ErrorIsf(t, err, errWebsocketAPIRequestFailed, "SendWsAPIRequest should return the %s response's error", method)
	}
	err = ex.SendWsAPIRequest(t.Context(), spotDefaultRate, "error", nil, nil)
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "SendWsAPIRequest should return the error the code maps to")
	err = ex.SendWsAPIRequest(t.Context(), spotDefaultRate, "error.data", nil, nil)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "SendWsAPIRequest must return an APIError")
	assert.Equal(t, &APIError{Code: -1003, Message: "Way too much request weight used; IP banned until 1659146400000.", Data: json.RawMessage(`{"serverTime":1659142907531,"retryAfter":1659146400000}`)}, apiErr, "SendWsAPIRequest should return the error's code, message and data")
	var result WsUserDataStreamSubscriptionResponse
	err = ex.SendWsAPIRequest(t.Context(), spotDefaultRate, "null", nil, &result)
	assert.ErrorIs(t, err, common.ErrNoResponse, "SendWsAPIRequest should error on a null result")
	require.NoError(t, ex.SendWsAPIRequest(t.Context(), spotDefaultRate, "result", nil, nil), "SendWsAPIRequest must not error when the result is not wanted")
	require.NoError(t, ex.SendWsAPIRequest(t.Context(), spotDefaultRate, "result", nil, &result), "SendWsAPIRequest must not error")
	assert.Equal(t, WsUserDataStreamSubscriptionResponse{SubscriptionID: 4}, result, "SendWsAPIRequest should decode the result")
}

func TestSendSignedWsAPIRequest(t *testing.T) {
	t.Parallel()
	ex := newWsAPITestExchange(t, true)
	var resp []WsRateLimit
	require.NoError(t, ex.SendSignedWsAPIRequest(t.Context(), currentOrderCountUsageRate, "account.rateLimits.orders", nil, &resp), "SendSignedWsAPIRequest must sign a request without parameters")
	assert.NotEmpty(t, resp, "SendSignedWsAPIRequest should decode the result")
}

func TestGetWsAveragePrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetWsAveragePrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetWsAveragePrice must reject an empty pair")

	ex := newWsAPITestExchange(t, false)
	got, err := ex.GetWsAveragePrice(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "GetWsAveragePrice must not error")
	if mockTests {
		exp := &WsAveragePriceResponse{
			Minutes:   5,
			Price:     86109.15045074,
			CloseTime: types.Time(time.Unix(1791285061, 0)),
		}
		assert.Equal(t, exp, got, "GetWsAveragePrice should return the average price")
	} else {
		assert.Positive(t, got.Price.Float64(), "GetWsAveragePrice should return a price")
	}
	_, err = ex.GetWsAveragePrice(t.Context(), currency.NewPair(currency.BTC, currency.NewCode("XYZ")))
	assert.ErrorIs(t, err, errWebsocketAPIRequestFailed, "GetWsAveragePrice should return the API's error for an unknown symbol")
}

func TestGetWsOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetWsOrderbook(t.Context(), currency.EMPTYPAIR, 0, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetWsOrderbook must reject an empty pair")

	ex := newWsAPITestExchange(t, false)
	got, err := ex.GetWsOrderbook(t.Context(), currency.NewBTCUSDT(), 5, "TRADING")
	require.NoError(t, err, "GetWsOrderbook must not error")
	if mockTests {
		assert.Equal(t, wsAPIFixtureResult[*OrderBookResponse](t, "depth", 0), got, "GetWsOrderbook should return the order book")
	} else {
		assert.NotEmpty(t, got, "GetWsOrderbook should return an order book")
	}
}

func TestGetWsCandlestick(t *testing.T) {
	t.Parallel()
	start, end := wsAPITimeRange()
	_, err := e.GetWsCandlestick(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetWsCandlestick must reject a nil request")
	_, err = e.GetWsCandlestick(t.Context(), &WsKlinesRequest{Interval: kline.OneMin})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetWsCandlestick must reject an empty pair")
	_, err = e.GetWsCandlestick(t.Context(), &WsKlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.TenMin})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetWsCandlestick must reject an unsupported interval")
	_, err = e.GetWsCandlestick(t.Context(), &WsKlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.OneMin, StartTime: end, EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetWsCandlestick must reject a start time after the end time")

	ex := newWsAPITestExchange(t, false)
	got, err := ex.GetWsCandlestick(t.Context(), &WsKlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.OneMin, StartTime: start, EndTime: end, TimeZone: "8", Limit: 2})
	require.NoError(t, err, "GetWsCandlestick must not error")
	if mockTests {
		assert.Equal(t, wsAPIFixtureResult[[]CandleStick](t, "klines", 0), got, "GetWsCandlestick should return the klines")
	} else {
		assert.Len(t, got, 2, "GetWsCandlestick should return the requested number of klines")
	}
}

func TestGetWsRollingWindowPriceChanges(t *testing.T) {
	t.Parallel()
	_, err := e.GetWsRollingWindowPriceChanges(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetWsRollingWindowPriceChanges must reject a nil request")
	_, err = e.GetWsRollingWindowPriceChanges(t.Context(), &WsRollingWindowTickerRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "GetWsRollingWindowPriceChanges must require symbols")
	_, err = e.GetWsRollingWindowPriceChanges(t.Context(), &WsRollingWindowTickerRequest{Symbols: currency.Pairs{currency.EMPTYPAIR}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetWsRollingWindowPriceChanges must reject an empty pair")
	_, err = e.GetWsRollingWindowPriceChanges(t.Context(), &WsRollingWindowTickerRequest{Symbols: currency.Pairs{currency.NewBTCUSDT()}, WindowSize: 90 * time.Minute})
	require.ErrorIs(t, err, errInvalidWindowSize, "GetWsRollingWindowPriceChanges must reject an unsupported window size")

	ex := newWsAPITestExchange(t, false)
	got, err := ex.GetWsRollingWindowPriceChanges(t.Context(), &WsRollingWindowTickerRequest{Symbols: currency.Pairs{currency.NewBTCUSDT()}, WindowSize: 2 * time.Hour, Type: "FULL", SymbolStatus: "TRADING"})
	require.NoError(t, err, "GetWsRollingWindowPriceChanges must not error for a symbol")
	if mockTests {
		assert.Equal(t, []PriceChangeStats{wsAPIFixtureResult[PriceChangeStats](t, "ticker", 0)}, got, "GetWsRollingWindowPriceChanges should return the symbol's statistics")
	} else {
		assert.Len(t, got, 1, "GetWsRollingWindowPriceChanges should return the symbol's statistics")
	}
	got, err = ex.GetWsRollingWindowPriceChanges(t.Context(), &WsRollingWindowTickerRequest{Symbols: currency.Pairs{currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USDT)}, WindowSize: 7 * 24 * time.Hour, Type: "MINI"})
	require.NoError(t, err, "GetWsRollingWindowPriceChanges must not error for several symbols")
	if mockTests {
		assert.Equal(t, wsAPIFixtureResult[[]PriceChangeStats](t, "ticker", 1), got, "GetWsRollingWindowPriceChanges should return the symbols' statistics")
	} else {
		assert.Len(t, got, 2, "GetWsRollingWindowPriceChanges should return the symbols' statistics")
	}
}

func TestWindowSizeToString(t *testing.T) {
	t.Parallel()
	for windowSize, exp := range map[time.Duration]string{
		time.Minute:        "1m",
		59 * time.Minute:   "59m",
		time.Hour:          "1h",
		23 * time.Hour:     "23h",
		24 * time.Hour:     "1d",
		7 * 24 * time.Hour: "7d",
	} {
		got, err := windowSizeToString(windowSize)
		require.NoErrorf(t, err, "windowSizeToString must not error for %s", windowSize)
		assert.Equalf(t, exp, got, "windowSizeToString should format %s", windowSize)
	}
	for _, windowSize := range []time.Duration{0, 30 * time.Second, 90 * time.Minute, 25 * time.Hour, 8 * 24 * time.Hour} {
		_, err := windowSizeToString(windowSize)
		assert.ErrorIsf(t, err, errInvalidWindowSize, "windowSizeToString should reject %s", windowSize)
	}
}

func TestGetWs24HourPriceChanges(t *testing.T) {
	t.Parallel()
	_, err := e.GetWs24HourPriceChanges(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetWs24HourPriceChanges must reject a nil request")
	_, err = e.GetWs24HourPriceChanges(t.Context(), &WsTickerRequest{Symbols: currency.Pairs{currency.EMPTYPAIR}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetWs24HourPriceChanges must reject an empty pair")

	ex := newWsAPITestExchange(t, false)
	got, err := ex.GetWs24HourPriceChanges(t.Context(), &WsTickerRequest{Symbols: currency.Pairs{currency.NewBTCUSDT()}})
	require.NoError(t, err, "GetWs24HourPriceChanges must not error for a symbol")
	if mockTests {
		assert.Equal(t, []PriceChangeStats{wsAPIFixtureResult[PriceChangeStats](t, "ticker.24hr", 0)}, got, "GetWs24HourPriceChanges should return the symbol's statistics")
	} else {
		assert.Len(t, got, 1, "GetWs24HourPriceChanges should return the symbol's statistics")
	}
	got, err = ex.GetWs24HourPriceChanges(t.Context(), &WsTickerRequest{Symbols: currency.Pairs{currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USDT)}, Type: "MINI", SymbolStatus: "TRADING"})
	require.NoError(t, err, "GetWs24HourPriceChanges must not error for several symbols")
	if mockTests {
		assert.Equal(t, wsAPIFixtureResult[[]PriceChangeStats](t, "ticker.24hr", 1), got, "GetWs24HourPriceChanges should return the symbols' statistics")
	} else {
		assert.Len(t, got, 2, "GetWs24HourPriceChanges should return the symbols' statistics")
	}
	if mockTests {
		// Every symbol's statistics weigh 80, too much for live tests
		got, err = ex.GetWs24HourPriceChanges(t.Context(), &WsTickerRequest{})
		require.NoError(t, err, "GetWs24HourPriceChanges must not error for every symbol")
		assert.Equal(t, wsAPIFixtureResult[[]PriceChangeStats](t, "ticker.24hr", 2), got, "GetWs24HourPriceChanges should return every symbol's statistics")
	}
}

func TestGetWsSymbolOrderbookTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetWsSymbolOrderbookTicker(t.Context(), currency.Pairs{currency.EMPTYPAIR}, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetWsSymbolOrderbookTicker must reject an empty pair")

	ex := newWsAPITestExchange(t, false)
	got, err := ex.GetWsSymbolOrderbookTicker(t.Context(), currency.Pairs{currency.NewBTCUSDT()}, "TRADING")
	require.NoError(t, err, "GetWsSymbolOrderbookTicker must not error for a symbol")
	if mockTests {
		exp := []WsBookTicker{
			{
				Symbol:      "BTCUSDT",
				BidPrice:    86102,
				BidQuantity: 11.56871,
				AskPrice:    86102.01,
				AskQuantity: 0.08287,
			},
		}
		assert.Equal(t, exp, got, "GetWsSymbolOrderbookTicker should return the symbol's best bid and ask")
	} else {
		assert.Len(t, got, 1, "GetWsSymbolOrderbookTicker should return the symbol's best bid and ask")
	}
	got, err = ex.GetWsSymbolOrderbookTicker(t.Context(), currency.Pairs{currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USDT)}, "")
	require.NoError(t, err, "GetWsSymbolOrderbookTicker must not error for several symbols")
	if mockTests {
		exp := []WsBookTicker{
			{
				Symbol:      "BTCUSDT",
				BidPrice:    86102,
				BidQuantity: 11.56824,
				AskPrice:    86102.01,
				AskQuantity: 0.06961,
			},
			{
				Symbol:      "ETHUSDT",
				BidPrice:    2711.38,
				BidQuantity: 55.022,
				AskPrice:    2711.39,
				AskQuantity: 6.6252,
			},
		}
		assert.Equal(t, exp, got, "GetWsSymbolOrderbookTicker should return the symbols' best bids and asks")
	} else {
		assert.Len(t, got, 2, "GetWsSymbolOrderbookTicker should return the symbols' best bids and asks")
	}
}

func TestGetWsSymbolPriceTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetWsSymbolPriceTicker(t.Context(), currency.Pairs{currency.EMPTYPAIR}, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetWsSymbolPriceTicker must reject an empty pair")

	ex := newWsAPITestExchange(t, false)
	got, err := ex.GetWsSymbolPriceTicker(t.Context(), currency.Pairs{currency.NewBTCUSDT()}, "TRADING")
	require.NoError(t, err, "GetWsSymbolPriceTicker must not error for a symbol")
	if mockTests {
		exp := []WsPriceTicker{
			{
				Symbol: "BTCUSDT",
				Price:  86100.06,
			},
		}
		assert.Equal(t, exp, got, "GetWsSymbolPriceTicker should return the symbol's price")
	} else {
		assert.Len(t, got, 1, "GetWsSymbolPriceTicker should return the symbol's price")
	}
	got, err = ex.GetWsSymbolPriceTicker(t.Context(), currency.Pairs{currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USDT)}, "")
	require.NoError(t, err, "GetWsSymbolPriceTicker must not error for several symbols")
	if mockTests {
		exp := []WsPriceTicker{
			{
				Symbol: "BTCUSDT",
				Price:  86102,
			},
			{
				Symbol: "ETHUSDT",
				Price:  2711.38,
			},
		}
		assert.Equal(t, exp, got, "GetWsSymbolPriceTicker should return the symbols' prices")
	} else {
		assert.Len(t, got, 2, "GetWsSymbolPriceTicker should return the symbols' prices")
	}
}

func TestGetWsTradingDayTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetWsTradingDayTickers(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetWsTradingDayTickers must reject a nil request")
	_, err = e.GetWsTradingDayTickers(t.Context(), &WsTradingDayTickerRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "GetWsTradingDayTickers must require symbols")
	_, err = e.GetWsTradingDayTickers(t.Context(), &WsTradingDayTickerRequest{Symbols: currency.Pairs{currency.EMPTYPAIR}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetWsTradingDayTickers must reject an empty pair")

	ex := newWsAPITestExchange(t, false)
	got, err := ex.GetWsTradingDayTickers(t.Context(), &WsTradingDayTickerRequest{Symbols: currency.Pairs{currency.NewBTCUSDT()}, TimeZone: "8", Type: "FULL", SymbolStatus: "TRADING"})
	require.NoError(t, err, "GetWsTradingDayTickers must not error for a symbol")
	if mockTests {
		assert.Equal(t, []PriceChangeStats{wsAPIFixtureResult[PriceChangeStats](t, "ticker.tradingDay", 0)}, got, "GetWsTradingDayTickers should return the symbol's statistics")
	} else {
		assert.Len(t, got, 1, "GetWsTradingDayTickers should return the symbol's statistics")
	}
	got, err = ex.GetWsTradingDayTickers(t.Context(), &WsTradingDayTickerRequest{Symbols: currency.Pairs{currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USDT)}})
	require.NoError(t, err, "GetWsTradingDayTickers must not error for several symbols")
	if mockTests {
		assert.Equal(t, wsAPIFixtureResult[[]PriceChangeStats](t, "ticker.tradingDay", 1), got, "GetWsTradingDayTickers should return the symbols' statistics")
	} else {
		assert.Len(t, got, 2, "GetWsTradingDayTickers should return the symbols' statistics")
	}
}

func TestGetWsAggregatedTrades(t *testing.T) {
	t.Parallel()
	start, end := wsAPITimeRange()
	_, err := e.GetWsAggregatedTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetWsAggregatedTrades must reject a nil request")
	_, err = e.GetWsAggregatedTrades(t.Context(), &WsAggregatedTradesRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetWsAggregatedTrades must reject an empty pair")
	_, err = e.GetWsAggregatedTrades(t.Context(), &WsAggregatedTradesRequest{Symbol: currency.NewBTCUSDT(), StartTime: end, EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetWsAggregatedTrades must reject a start time after the end time")

	ex := newWsAPITestExchange(t, false)
	got, err := ex.GetWsAggregatedTrades(t.Context(), &WsAggregatedTradesRequest{Symbol: currency.NewBTCUSDT(), FromID: 4081866525, Limit: 2})
	require.NoError(t, err, "GetWsAggregatedTrades must not error from a trade ID")
	if mockTests {
		assert.Equal(t, wsAPIFixtureResult[[]AggregatedTrade](t, "trades.aggregate", 0), got, "GetWsAggregatedTrades should return the trades from the ID")
	} else {
		assert.Len(t, got, 2, "GetWsAggregatedTrades should return the requested number of trades")
	}
	got, err = ex.GetWsAggregatedTrades(t.Context(), &WsAggregatedTradesRequest{Symbol: currency.NewBTCUSDT(), StartTime: start, EndTime: end, Limit: 1000})
	require.NoError(t, err, "GetWsAggregatedTrades must not error for a time range")
	if mockTests {
		assert.Equal(t, wsAPIFixtureResult[[]AggregatedTrade](t, "trades.aggregate", 1), got, "GetWsAggregatedTrades should return the trades in the time range")
	} else {
		assert.NotEmpty(t, got, "GetWsAggregatedTrades should return the trades in the time range")
	}
}

func TestGetWsMostRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetWsMostRecentTrades(t.Context(), currency.EMPTYPAIR, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetWsMostRecentTrades must reject an empty pair")

	ex := newWsAPITestExchange(t, false)
	got, err := ex.GetWsMostRecentTrades(t.Context(), currency.NewBTCUSDT(), 2)
	require.NoError(t, err, "GetWsMostRecentTrades must not error")
	if mockTests {
		assert.Equal(t, wsAPIFixtureResult[[]RecentTrade](t, "trades.recent", 0), got, "GetWsMostRecentTrades should return the trades")
	} else {
		assert.Len(t, got, 2, "GetWsMostRecentTrades should return the requested number of trades")
	}
}

func TestGetWsOptimizedCandlestick(t *testing.T) {
	t.Parallel()
	_, err := e.GetWsOptimizedCandlestick(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetWsOptimizedCandlestick must reject a nil request")

	ex := newWsAPITestExchange(t, false)
	got, err := ex.GetWsOptimizedCandlestick(t.Context(), &WsKlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.OneDay, Limit: 2})
	require.NoError(t, err, "GetWsOptimizedCandlestick must not error")
	if mockTests {
		assert.Equal(t, wsAPIFixtureResult[[]CandleStick](t, "uiKlines", 0), got, "GetWsOptimizedCandlestick should return the klines")
	} else {
		assert.Len(t, got, 2, "GetWsOptimizedCandlestick should return the requested number of klines")
	}
}

func TestWsLogOutOfSession(t *testing.T) {
	t.Parallel()
	ex := newWsAPITestExchange(t, false)
	got, err := ex.WsLogOutOfSession(t.Context())
	require.NoError(t, err, "WsLogOutOfSession must not error")
	if mockTests {
		exp := &WsSessionStatusResponse{
			ConnectedSince: types.Time(time.UnixMilli(1649729873021)),
			ServerTime:     types.Time(time.UnixMilli(1649730611671)),
			UserDataStream: new(false),
		}
		assert.Equal(t, exp, got, "WsLogOutOfSession should return the session's status")
	} else {
		assert.Empty(t, got.APIKey, "WsLogOutOfSession should leave the session without an API key")
	}
}

func TestGetWsSessionStatus(t *testing.T) {
	t.Parallel()
	ex := newWsAPITestExchange(t, false)
	got, err := ex.GetWsSessionStatus(t.Context())
	require.NoError(t, err, "GetWsSessionStatus must not error")
	if mockTests {
		exp := &WsSessionStatusResponse{
			APIKey:           "mock-api-key",
			AuthorizedSince:  types.Time(time.UnixMilli(1649729878532)),
			ConnectedSince:   types.Time(time.UnixMilli(1649729873021)),
			ReturnRateLimits: true,
			ServerTime:       types.Time(time.UnixMilli(1649730611671)),
			UserDataStream:   new(true),
		}
		assert.Equal(t, exp, got, "GetWsSessionStatus should return the session's status")
	} else {
		assert.False(t, got.ConnectedSince.Time().IsZero(), "GetWsSessionStatus should return when the session connected")
	}
}

func TestWsSessionStatusUserDataStream(t *testing.T) {
	t.Parallel()
	for raw, exp := range map[string]*bool{
		`{"userDataStream":true}`:  new(true),
		`{"userDataStream":false}`: new(false),
		`{"userDataStream":null}`:  nil,
		`{}`:                       nil,
	} {
		var got WsSessionStatusResponse
		require.NoErrorf(t, json.Unmarshal([]byte(raw), &got), "Unmarshal must not error for %s", raw)
		assert.Equalf(t, exp, got.UserDataStream, "UserDataStream should tell %s apart from the other states", raw)
	}
}

func TestWsCancelOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.WsCancelOpenOrders(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "WsCancelOpenOrders must reject an empty pair")

	ex := newWsAPITestExchange(t, true, canManipulateRealOrders)
	got, err := ex.WsCancelOpenOrders(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "WsCancelOpenOrders must not error")
	if mockTests {
		exp := []WsCancelOrderResponse{
			{
				Symbol:                     "BTCUSDT",
				OriginalClientOrderID:      "4d96324ff9d44481926157",
				OrderID:                    12569099453,
				OrderListID:                -1,
				ClientOrderID:              "91fe37ce9e69c90d6358c0",
				TransactTime:               types.Time(time.UnixMilli(1684804350068)),
				Price:                      23416.1,
				OriginalQuantity:           0.00847,
				ExecutedQuantity:           0.00001,
				OriginalQuoteOrderQuantity: 198.335215,
				CummulativeQuoteQuantity:   0.234161,
				Status:                     "CANCELED",
				TimeInForce:                "GTC",
				Type:                       "STOP_LOSS_LIMIT",
				Side:                       "SELL",
				StopPrice:                  23400,
				TrailingDelta:              10,
				TrailingTime:               1684804350001,
				IcebergQuantity:            0.001,
				StrategyID:                 37463720,
				StrategyType:               1000000,
				SelfTradePreventionMode:    "EXPIRE_MAKER",
				PreventedMatchID:           3,
				PreventedQuantity:          1.2,
				UsedSOR:                    true,
				WorkingFloor:               "SOR",
				PegPriceType:               "PRIMARY_PEG",
				PegOffsetType:              "PRICE_LEVEL",
				PegOffsetValue:             5,
				PeggedPrice:                23416.1,
				ExpiryReason:               "INSUFFICIENT_LIQUIDITY",
			},
			{
				Symbol:            "BTCUSDT",
				OrderListID:       19431,
				ContingencyType:   "OCO",
				ListStatusType:    "ALL_DONE",
				ListOrderStatus:   "ALL_DONE",
				ListClientOrderID: "iuVNVJYYrByz6C4yGOPPK0",
				TransactionTime:   types.Time(time.UnixMilli(1660803702431)),
				Orders: []WsOrderListOrder{
					{
						Symbol:        "BTCUSDT",
						OrderID:       12569099453,
						ClientOrderID: "bX5wROblo6YeDwa9iTLeyY",
					},
					{
						Symbol:        "BTCUSDT",
						OrderID:       12569099454,
						ClientOrderID: "Tnu2IP0J5Y4mxw3IATBfmW",
					},
				},
				OrderReports: []WsOrderReport{
					{
						Symbol:                  "BTCUSDT",
						OriginalClientOrderID:   "bX5wROblo6YeDwa9iTLeyY",
						OrderID:                 12569099453,
						OrderListID:             19431,
						ClientOrderID:           "OFFXQtxVFZ6Nbcg4PgE2DA",
						TransactTime:            types.Time(time.UnixMilli(1684804350068)),
						Price:                   23450.5,
						OriginalQuantity:        0.0085,
						Status:                  "CANCELED",
						TimeInForce:             "GTC",
						Type:                    "STOP_LOSS_LIMIT",
						Side:                    "BUY",
						StopPrice:               23430,
						SelfTradePreventionMode: "NONE",
					},
					{
						Symbol:                  "BTCUSDT",
						OriginalClientOrderID:   "Tnu2IP0J5Y4mxw3IATBfmW",
						OrderID:                 12569099454,
						OrderListID:             19431,
						ClientOrderID:           "OFFXQtxVFZ6Nbcg4PgE2DA",
						TransactTime:            types.Time(time.UnixMilli(1684804350068)),
						Price:                   23400,
						OriginalQuantity:        0.0085,
						Status:                  "CANCELED",
						TimeInForce:             "GTC",
						Type:                    "LIMIT_MAKER",
						Side:                    "BUY",
						SelfTradePreventionMode: "NONE",
					},
				},
			},
		}
		assert.Equal(t, exp, got, "WsCancelOpenOrders should return the cancelled orders and order lists")
	}
}

func TestWsCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WsCancelOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsCancelOrder must reject a nil request")
	_, err = e.WsCancelOrder(t.Context(), &WsCancelOrderRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "WsCancelOrder must require an order ID or client order ID")
	_, err = e.WsCancelOrder(t.Context(), &WsCancelOrderRequest{OrderID: 12569099453})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "WsCancelOrder must reject an empty pair")

	ex := newWsAPITestExchange(t, true, canManipulateRealOrders)
	got, err := ex.WsCancelOrder(t.Context(), &WsCancelOrderRequest{
		Symbol:                currency.NewBTCUSDT(),
		OrderID:               12569099453,
		OriginalClientOrderID: "4d96324ff9d44481926157",
		NewClientOrderID:      "91fe37ce9e69c90d6358c0",
		CancelRestrictions:    "ONLY_NEW",
	})
	require.NoError(t, err, "WsCancelOrder must not error")
	if mockTests {
		exp := &WsCancelOrderResponse{
			Symbol:                     "BTCUSDT",
			OriginalClientOrderID:      "4d96324ff9d44481926157",
			OrderID:                    12569099453,
			OrderListID:                -1,
			ClientOrderID:              "91fe37ce9e69c90d6358c0",
			TransactTime:               types.Time(time.UnixMilli(1684804350068)),
			Price:                      23416.1,
			OriginalQuantity:           0.00847,
			ExecutedQuantity:           0.00001,
			OriginalQuoteOrderQuantity: 198.335215,
			CummulativeQuoteQuantity:   0.234161,
			Status:                     "CANCELED",
			TimeInForce:                "GTC",
			Type:                       "STOP_LOSS_LIMIT",
			Side:                       "SELL",
			StopPrice:                  23400,
			TrailingDelta:              10,
			TrailingTime:               1684804350001,
			IcebergQuantity:            0.001,
			StrategyID:                 37463720,
			StrategyType:               1000000,
			SelfTradePreventionMode:    "EXPIRE_MAKER",
			PreventedMatchID:           3,
			PreventedQuantity:          1.2,
			UsedSOR:                    true,
			WorkingFloor:               "SOR",
			PegPriceType:               "PRIMARY_PEG",
			PegOffsetType:              "PRICE_LEVEL",
			PegOffsetValue:             5,
			PeggedPrice:                23416.1,
			ExpiryReason:               "INSUFFICIENT_LIQUIDITY",
		}
		assert.Equal(t, exp, got, "WsCancelOrder should return the cancelled order")
	}
	got, err = ex.WsCancelOrder(t.Context(), &WsCancelOrderRequest{Symbol: currency.NewBTCUSDT(), OrderID: 12569099454})
	require.NoError(t, err, "WsCancelOrder must not error for an order list's leg")
	if mockTests {
		exp := &WsCancelOrderResponse{
			Symbol:            "BTCUSDT",
			OrderListID:       19431,
			ContingencyType:   "OCO",
			ListStatusType:    "ALL_DONE",
			ListOrderStatus:   "ALL_DONE",
			ListClientOrderID: "iuVNVJYYrByz6C4yGOPPK0",
			TransactionTime:   types.Time(time.UnixMilli(1660803702431)),
			Orders: []WsOrderListOrder{
				{
					Symbol:        "BTCUSDT",
					OrderID:       12569099453,
					ClientOrderID: "bX5wROblo6YeDwa9iTLeyY",
				},
				{
					Symbol:        "BTCUSDT",
					OrderID:       12569099454,
					ClientOrderID: "Tnu2IP0J5Y4mxw3IATBfmW",
				},
			},
			OrderReports: []WsOrderReport{
				{
					Symbol:                  "BTCUSDT",
					OriginalClientOrderID:   "bX5wROblo6YeDwa9iTLeyY",
					OrderID:                 12569099453,
					OrderListID:             19431,
					ClientOrderID:           "OFFXQtxVFZ6Nbcg4PgE2DA",
					TransactTime:            types.Time(time.UnixMilli(1684804350068)),
					Price:                   23450.5,
					OriginalQuantity:        0.0085,
					Status:                  "CANCELED",
					TimeInForce:             "GTC",
					Type:                    "STOP_LOSS_LIMIT",
					Side:                    "BUY",
					StopPrice:               23430,
					SelfTradePreventionMode: "NONE",
				},
				{
					Symbol:                  "BTCUSDT",
					OriginalClientOrderID:   "Tnu2IP0J5Y4mxw3IATBfmW",
					OrderID:                 12569099454,
					OrderListID:             19431,
					ClientOrderID:           "OFFXQtxVFZ6Nbcg4PgE2DA",
					TransactTime:            types.Time(time.UnixMilli(1684804350068)),
					Price:                   23400,
					OriginalQuantity:        0.0085,
					Status:                  "CANCELED",
					TimeInForce:             "GTC",
					Type:                    "LIMIT_MAKER",
					Side:                    "BUY",
					SelfTradePreventionMode: "NONE",
				},
			},
		}
		assert.Equal(t, exp, got, "WsCancelOrder should return the cancelled order list")
	}
}

func TestWsCancelAndReplaceTradeOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WsCancelAndReplaceTradeOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsCancelAndReplaceTradeOrder must reject a nil request")
	arg := &WsCancelReplaceOrderRequest{}
	_, err = e.WsCancelAndReplaceTradeOrder(t.Context(), arg)
	require.ErrorIs(t, err, errCancelReplaceModeRequired, "WsCancelAndReplaceTradeOrder must require a cancel replace mode")
	arg.CancelReplaceMode = "ALLOW_FAILURE"
	_, err = e.WsCancelAndReplaceTradeOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "WsCancelAndReplaceTradeOrder must require the order ID or client order ID to cancel")
	arg.CancelOrderID = 125690984230
	_, err = e.WsCancelAndReplaceTradeOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "WsCancelAndReplaceTradeOrder must require a side")
	arg.Side = "SELL"
	_, err = e.WsCancelAndReplaceTradeOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "WsCancelAndReplaceTradeOrder must require an order type")
	arg.Type = "STOP_LOSS_LIMIT"
	_, err = e.WsCancelAndReplaceTradeOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "WsCancelAndReplaceTradeOrder must reject an empty pair")

	ex := newWsAPITestExchange(t, true, canManipulateRealOrders)
	got, err := ex.WsCancelAndReplaceTradeOrder(t.Context(), &WsCancelReplaceOrderRequest{
		Symbol:                      currency.NewBTCUSDT(),
		CancelReplaceMode:           "ALLOW_FAILURE",
		CancelOrderID:               125690984230,
		CancelOriginalClientOrderID: "4d96324ff9d44481926157",
		CancelNewClientOrderID:      "91fe37ce9e69c90d6358c0",
		Side:                        "SELL",
		Type:                        "STOP_LOSS_LIMIT",
		TimeInForce:                 "GTC",
		Price:                       23416.1,
		Quantity:                    0.00847,
		QuoteOrderQuantity:          198.335215,
		NewClientOrderID:            "bX5wROblo6YeDwa9iTLeyY",
		NewOrderResponseType:        "FULL",
		StopPrice:                   23400,
		TrailingDelta:               10,
		IcebergQuantity:             0.001,
		StrategyID:                  37463720,
		StrategyType:                1000000,
		SelfTradePreventionMode:     "EXPIRE_MAKER",
		CancelRestrictions:          "ONLY_NEW",
		OrderRateLimitExceededMode:  "CANCEL_ONLY",
		PegPriceType:                "PRIMARY_PEG",
		PegOffsetValue:              5,
		PegOffsetType:               "PRICE_LEVEL",
	})
	require.NoError(t, err, "WsCancelAndReplaceTradeOrder must not error")
	if mockTests {
		exp := &WsCancelReplaceOrderResponse{
			CancelResult:   "SUCCESS",
			NewOrderResult: "SUCCESS",
			CancelResponse: WsCancelReplaceCancelResponse{
				Symbol:                     "BTCUSDT",
				OriginalClientOrderID:      "4d96324ff9d44481926157",
				OrderID:                    12569099453,
				OrderListID:                -1,
				ClientOrderID:              "91fe37ce9e69c90d6358c0",
				TransactTime:               types.Time(time.UnixMilli(1684804350068)),
				Price:                      23416.1,
				OriginalQuantity:           0.00847,
				ExecutedQuantity:           0.00001,
				OriginalQuoteOrderQuantity: 198.335215,
				CummulativeQuoteQuantity:   0.234161,
				Status:                     "CANCELED",
				TimeInForce:                "GTC",
				Type:                       "STOP_LOSS_LIMIT",
				Side:                       "SELL",
				StopPrice:                  23400,
				TrailingDelta:              10,
				TrailingTime:               1684804350001,
				IcebergQuantity:            0.001,
				StrategyID:                 37463720,
				StrategyType:               1000000,
				SelfTradePreventionMode:    "EXPIRE_MAKER",
				PreventedMatchID:           3,
				PreventedQuantity:          1.2,
				UsedSOR:                    true,
				WorkingFloor:               "SOR",
				PegPriceType:               "PRIMARY_PEG",
				PegOffsetType:              "PRICE_LEVEL",
				PegOffsetValue:             5,
				PeggedPrice:                23416.1,
				ExpiryReason:               "INSUFFICIENT_LIQUIDITY",
			},
			NewOrderResponse: WsCancelReplaceNewOrderResponse{
				Symbol:                     "BTCUSDT",
				OrderID:                    12569099453,
				OrderListID:                -1,
				ClientOrderID:              "4d96324ff9d44481926157ec08158a40",
				TransactTime:               types.Time(time.UnixMilli(1660801715793)),
				Price:                      23416.1,
				OriginalQuantity:           0.00847,
				ExecutedQuantity:           0.00847,
				OriginalQuoteOrderQuantity: 198.335215,
				CummulativeQuoteQuantity:   198.335215,
				Status:                     "FILLED",
				TimeInForce:                "GTC",
				Type:                       "LIMIT",
				Side:                       "SELL",
				WorkingTime:                1660801715793,
				SelfTradePreventionMode:    "EXPIRE_MAKER",
				Fills: []WsOrderFill{
					{
						Price:           23416.1,
						Quantity:        0.00635,
						Commission:      0.0000011,
						CommissionAsset: currency.BNB,
						TradeID:         1650422481,
					},
					{
						Price:           23416.5,
						Quantity:        0.00212,
						Commission:      0.0000004,
						CommissionAsset: currency.BNB,
						TradeID:         1650422482,
					},
				},
				IcebergQuantity:   0.001,
				PreventedMatchID:  3,
				PreventedQuantity: 1.2,
				StopPrice:         23400,
				StrategyID:        37463720,
				StrategyType:      1000000,
				TrailingDelta:     10,
				TrailingTime:      -1,
				UsedSOR:           true,
				WorkingFloor:      "EXCHANGE",
				PegPriceType:      "PRIMARY_PEG",
				PegOffsetType:     "PRICE_LEVEL",
				PegOffsetValue:    5,
				PeggedPrice:       23416.1,
				ExpiryReason:      "INSUFFICIENT_LIQUIDITY",
			},
		}
		assert.Equal(t, exp, got, "WsCancelAndReplaceTradeOrder should return the cancellation and the new order")
	}

	partial := newTestExchange(t)
	useMockWebsocket(t, partial, partial.spotWebsocketAPIConnectionSetup(partial.Config), func(tb testing.TB, msg []byte, w *gws.Conn) error {
		tb.Helper()
		var req WsAPIRequest
		require.NoError(tb, json.Unmarshal(msg, &req), "request must decode")
		return w.WriteMessage(gws.TextMessage, fmt.Appendf(nil, `{"id":%q,"status":409,"error":{"code":-2021,"msg":"Order cancel-replace partially failed.","data":{"cancelResult":"SUCCESS","newOrderResult":"FAILURE","cancelResponse":{"symbol":"BTCUSDT","origClientOrderId":"4d96324ff9d44481926157","orderId":125690984230,"orderListId":-1,"clientOrderId":"91fe37ce9e69c90d6358c0","transactTime":1684804350068},"newOrderResponse":{"code":-2010,"msg":"Order would immediately match and take."}}}}`, req.ID))
	})
	require.NoError(t, partial.Websocket.Connect(t.Context()), "Connect must not error")
	got, err = partial.WsCancelAndReplaceTradeOrder(t.Context(), &WsCancelReplaceOrderRequest{Symbol: currency.NewBTCUSDT(), CancelReplaceMode: "ALLOW_FAILURE", CancelOrderID: 125690984230, Side: order.Sell.String(), Type: order.Limit.String(), TimeInForce: "GTC", Price: 23416.1, Quantity: 0.00847})
	require.ErrorIs(t, err, errWebsocketAPIRequestFailed, "WsCancelAndReplaceTradeOrder must return the API's error")
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "WsCancelAndReplaceTradeOrder must return an APIError")
	assert.Equal(t, int64(-2021), apiErr.Code, "WsCancelAndReplaceTradeOrder should return the API's error code")
	exp := &WsCancelReplaceOrderResponse{
		CancelResult:     "SUCCESS",
		NewOrderResult:   "FAILURE",
		CancelResponse:   WsCancelReplaceCancelResponse{Symbol: "BTCUSDT", OriginalClientOrderID: "4d96324ff9d44481926157", OrderID: 125690984230, OrderListID: -1, ClientOrderID: "91fe37ce9e69c90d6358c0", TransactTime: types.Time(time.UnixMilli(1684804350068))},
		NewOrderResponse: WsCancelReplaceNewOrderResponse{Code: -2010, Message: "Order would immediately match and take."},
	}
	assert.Equal(t, exp, got, "WsCancelAndReplaceTradeOrder should return the outcome of both halves")
}

func TestWsCancelOCOOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WsCancelOCOOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsCancelOCOOrder must reject a nil request")
	_, err = e.WsCancelOCOOrder(t.Context(), &WsCancelOrderListRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, errOrderListIDRequired, "WsCancelOCOOrder must require an order list ID or client order list ID")
	_, err = e.WsCancelOCOOrder(t.Context(), &WsCancelOrderListRequest{OrderListID: 1274512})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "WsCancelOCOOrder must reject an empty pair")

	ex := newWsAPITestExchange(t, true, canManipulateRealOrders)
	got, err := ex.WsCancelOCOOrder(t.Context(), &WsCancelOrderListRequest{
		Symbol:            currency.NewBTCUSDT(),
		OrderListID:       1274512,
		ListClientOrderID: "6023531d7edaad348f5aff",
		NewClientOrderID:  "tTHrs4n3dWgqWQ1mXFZrzQ",
	})
	require.NoError(t, err, "WsCancelOCOOrder must not error")
	if mockTests {
		exp := &WsCancelOrderResponse{
			Symbol:            "BTCUSDT",
			OrderListID:       1274512,
			ContingencyType:   "OCO",
			ListStatusType:    "ALL_DONE",
			ListOrderStatus:   "ALL_DONE",
			ListClientOrderID: "6023531d7edaad348f5aff",
			TransactionTime:   types.Time(time.UnixMilli(1660801720215)),
			Orders: []WsOrderListOrder{
				{
					Symbol:        "BTCUSDT",
					OrderID:       12569138901,
					ClientOrderID: "BqtFCj5odMoWtSqGk2X9tU",
				},
				{
					Symbol:        "BTCUSDT",
					OrderID:       12569138902,
					ClientOrderID: "jLnZpj5enfMXTuhKB1d0us",
				},
			},
			OrderReports: []WsOrderReport{
				{
					Symbol:                  "BTCUSDT",
					OriginalClientOrderID:   "BqtFCj5odMoWtSqGk2X9tU",
					OrderID:                 12569138901,
					OrderListID:             1274512,
					ClientOrderID:           "tTHrs4n3dWgqWQ1mXFZrzQ",
					TransactTime:            types.Time(time.UnixMilli(1660801720215)),
					Price:                   23410,
					OriginalQuantity:        0.0065,
					Status:                  "CANCELED",
					TimeInForce:             "GTC",
					Type:                    "STOP_LOSS_LIMIT",
					Side:                    "SELL",
					StopPrice:               23405,
					SelfTradePreventionMode: "NONE",
				},
				{
					Symbol:                  "BTCUSDT",
					OriginalClientOrderID:   "jLnZpj5enfMXTuhKB1d0us",
					OrderID:                 12569138902,
					OrderListID:             1274512,
					ClientOrderID:           "tTHrs4n3dWgqWQ1mXFZrzQ",
					TransactTime:            types.Time(time.UnixMilli(1660801720215)),
					Price:                   23420,
					OriginalQuantity:        0.0065,
					Status:                  "CANCELED",
					TimeInForce:             "GTC",
					Type:                    "LIMIT_MAKER",
					Side:                    "SELL",
					SelfTradePreventionMode: "NONE",
				},
			},
		}
		assert.Equal(t, exp, got, "WsCancelOCOOrder should return the cancelled order list")
	}
}

func TestWsPlaceOCOOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WsPlaceOCOOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsPlaceOCOOrder must reject a nil request")
	arg := &WsPlaceOCOOrderRequest{}
	_, err = e.WsPlaceOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "WsPlaceOCOOrder must require a side")
	arg.Side = "SELL"
	_, err = e.WsPlaceOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "WsPlaceOCOOrder must require a price")
	arg.Price = 23420
	_, err = e.WsPlaceOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "WsPlaceOCOOrder must require a quantity")
	arg.Quantity = 0.0065
	_, err = e.WsPlaceOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, errStopPriceOrTrailingDeltaRequired, "WsPlaceOCOOrder must require a stop price or trailing delta")
	arg.StopPrice = 23410
	_, err = e.WsPlaceOCOOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "WsPlaceOCOOrder must reject an empty pair")

	ex := newWsAPITestExchange(t, true, canManipulateRealOrders)
	got, err := ex.WsPlaceOCOOrder(t.Context(), &WsPlaceOCOOrderRequest{
		Symbol:                  currency.NewBTCUSDT(),
		Side:                    "SELL",
		Price:                   23420,
		Quantity:                0.0065,
		ListClientOrderID:       "08985fedd9ea2cf6b28996",
		LimitClientOrderID:      "jLnZpj5enfMXTuhKB1d0us",
		LimitIcebergQuantity:    0.001,
		LimitStrategyID:         1,
		LimitStrategyType:       1000000,
		StopPrice:               23410,
		TrailingDelta:           10,
		StopClientOrderID:       "BqtFCj5odMoWtSqGk2X9tU",
		StopLimitPrice:          23405,
		StopLimitTimeInForce:    "GTC",
		StopIcebergQuantity:     0.001,
		StopStrategyID:          2,
		StopStrategyType:        1000001,
		NewOrderResponseType:    "RESULT",
		SelfTradePreventionMode: "EXPIRE_TAKER",
	})
	require.NoError(t, err, "WsPlaceOCOOrder must not error")
	if mockTests {
		exp := &WsOrderListResponse{
			OrderListID:       1274512,
			ContingencyType:   "OCO",
			ListStatusType:    "EXEC_STARTED",
			ListOrderStatus:   "EXECUTING",
			ListClientOrderID: "08985fedd9ea2cf6b28996",
			TransactionTime:   types.Time(time.UnixMilli(1660801713793)),
			Symbol:            "BTCUSDT",
			Orders: []WsOrderListOrder{
				{
					Symbol:        "BTCUSDT",
					OrderID:       12569138901,
					ClientOrderID: "BqtFCj5odMoWtSqGk2X9tU",
				},
				{
					Symbol:        "BTCUSDT",
					OrderID:       12569138902,
					ClientOrderID: "jLnZpj5enfMXTuhKB1d0us",
				},
			},
			OrderReports: []WsOrderResponse{
				{
					Symbol:                  "BTCUSDT",
					OrderID:                 12569138901,
					OrderListID:             1274512,
					ClientOrderID:           "BqtFCj5odMoWtSqGk2X9tU",
					TransactTime:            types.Time(time.UnixMilli(1660801713793)),
					Price:                   23410,
					OriginalQuantity:        0.0065,
					Status:                  "NEW",
					TimeInForce:             "GTC",
					Type:                    "STOP_LOSS_LIMIT",
					Side:                    "SELL",
					WorkingTime:             -1,
					SelfTradePreventionMode: "NONE",
					StopPrice:               23405,
				},
				{
					Symbol:                  "BTCUSDT",
					OrderID:                 12569138902,
					OrderListID:             1274512,
					ClientOrderID:           "jLnZpj5enfMXTuhKB1d0us",
					TransactTime:            types.Time(time.UnixMilli(1660801713793)),
					Price:                   23420,
					OriginalQuantity:        0.0065,
					Status:                  "NEW",
					TimeInForce:             "GTC",
					Type:                    "LIMIT_MAKER",
					Side:                    "SELL",
					WorkingTime:             1660801713793,
					SelfTradePreventionMode: "NONE",
				},
			},
		}
		assert.Equal(t, exp, got, "WsPlaceOCOOrder should return the order list")
	}
}

func TestWsPlaceOCOOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.WsPlaceOCOOrderList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsPlaceOCOOrderList must reject a nil request")
	arg := &OCOOrderListRequest{}
	_, err = e.WsPlaceOCOOrderList(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "WsPlaceOCOOrderList must reject a request without a side")
	arg.Side = "SELL"
	_, err = e.WsPlaceOCOOrderList(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "WsPlaceOCOOrderList must reject a request without a quantity")
	arg.Quantity = 0.0065
	_, err = e.WsPlaceOCOOrderList(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "WsPlaceOCOOrderList must reject a request without an above order type")
	arg.Above.Type = "TAKE_PROFIT_LIMIT"
	_, err = e.WsPlaceOCOOrderList(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "WsPlaceOCOOrderList must reject a request without a below order type")
	arg.Below.Type = "STOP_LOSS"
	_, err = e.WsPlaceOCOOrderList(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "WsPlaceOCOOrderList must reject a request without a symbol")

	ex := newWsAPITestExchange(t, true, canManipulateRealOrders)
	got, err := ex.WsPlaceOCOOrderList(t.Context(), &OCOOrderListRequest{
		Symbol:            currency.NewBTCUSDT(),
		ListClientOrderID: "08985fedd9ea2cf6b28996",
		Side:              "SELL",
		Quantity:          0.0065,
		Above: OCOOrderListLeg{
			Type:            "TAKE_PROFIT_LIMIT",
			ClientOrderID:   "jLnZpj5enfMXTuhKB1d0us",
			IcebergQuantity: 1,
			Price:           23500,
			StopPrice:       23480,
			TimeInForce:     "GTC",
			StrategyID:      1,
			StrategyType:    1000000,
			PegPriceType:    "PRIMARY_PEG",
			PegOffsetType:   "PRICE_LEVEL",
			PegOffsetValue:  5,
		},
		Below: OCOOrderListLeg{
			Type:          "STOP_LOSS",
			ClientOrderID: "BqtFCj5odMoWtSqGk2X9tU",
			StopPrice:     23405,
			TrailingDelta: 10,
			StrategyID:    2,
			StrategyType:  1000001,
		},
		NewOrderResponseType:    "RESULT",
		SelfTradePreventionMode: "EXPIRE_TAKER",
	})
	require.NoError(t, err, "WsPlaceOCOOrderList must not error")
	if mockTests {
		exp := &WsOrderListResponse{
			OrderListID:       1274513,
			ContingencyType:   "OCO",
			ListStatusType:    "EXEC_STARTED",
			ListOrderStatus:   "EXECUTING",
			ListClientOrderID: "08985fedd9ea2cf6b28996",
			TransactionTime:   types.Time(time.UnixMilli(1711062760648)),
			Symbol:            "BTCUSDT",
			Orders: []WsOrderListOrder{
				{Symbol: "BTCUSDT", OrderID: 12569138903, ClientOrderID: "BqtFCj5odMoWtSqGk2X9tU"},
				{Symbol: "BTCUSDT", OrderID: 12569138904, ClientOrderID: "jLnZpj5enfMXTuhKB1d0us"},
			},
			OrderReports: []WsOrderResponse{
				{
					Symbol:                  "BTCUSDT",
					OrderID:                 12569138903,
					OrderListID:             1274513,
					ClientOrderID:           "BqtFCj5odMoWtSqGk2X9tU",
					TransactTime:            types.Time(time.UnixMilli(1711062760648)),
					OriginalQuantity:        0.0065,
					Status:                  "NEW",
					TimeInForce:             "GTC",
					Type:                    "STOP_LOSS",
					Side:                    "SELL",
					WorkingTime:             -1,
					SelfTradePreventionMode: "NONE",
					StopPrice:               23405,
					StrategyID:              2,
					StrategyType:            1000001,
					TrailingDelta:           10,
					TrailingTime:            -1,
				},
				{
					Symbol:                  "BTCUSDT",
					OrderID:                 12569138904,
					OrderListID:             1274513,
					ClientOrderID:           "jLnZpj5enfMXTuhKB1d0us",
					TransactTime:            types.Time(time.UnixMilli(1711062760648)),
					Price:                   23500,
					OriginalQuantity:        0.0065,
					Status:                  "NEW",
					TimeInForce:             "GTC",
					Type:                    "TAKE_PROFIT_LIMIT",
					Side:                    "SELL",
					WorkingTime:             -1,
					SelfTradePreventionMode: "NONE",
					IcebergQuantity:         0.001,
					StopPrice:               23480,
					StrategyID:              1,
					StrategyType:            1000000,
					PegPriceType:            "PRIMARY_PEG",
					PegOffsetType:           "PRICE_LEVEL",
					PegOffsetValue:          5,
					PeggedPrice:             23500,
				},
			},
		}
		assert.Equal(t, exp, got, "WsPlaceOCOOrderList should return the order list")
	}
}

func TestWsPlaceNewOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WsPlaceNewOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsPlaceNewOrder must reject a nil request")
	arg := &WsPlaceOrderRequest{}
	_, err = e.WsPlaceNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "WsPlaceNewOrder must require a side")
	arg.Side = "SELL"
	_, err = e.WsPlaceNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "WsPlaceNewOrder must require an order type")
	arg.Type = "LIMIT"
	_, err = e.WsPlaceNewOrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "WsPlaceNewOrder must reject an empty pair")

	ex := newWsAPITestExchange(t, true, canManipulateRealOrders)
	got, err := ex.WsPlaceNewOrder(t.Context(), &WsPlaceOrderRequest{
		Symbol:                  currency.NewBTCUSDT(),
		Side:                    "SELL",
		Type:                    "LIMIT",
		TimeInForce:             "GTC",
		Price:                   23416.1,
		Quantity:                0.00847,
		NewClientOrderID:        "4d96324ff9d44481926157ec08158a40",
		NewOrderResponseType:    "FULL",
		IcebergQuantity:         0.001,
		StrategyID:              37463720,
		StrategyType:            1000000,
		SelfTradePreventionMode: "EXPIRE_MAKER",
		PegPriceType:            "PRIMARY_PEG",
		PegOffsetValue:          5,
		PegOffsetType:           "PRICE_LEVEL",
	})
	require.NoError(t, err, "WsPlaceNewOrder must not error for a limit order")
	if mockTests {
		exp := &WsOrderResponse{
			Symbol:                     "BTCUSDT",
			OrderID:                    12569099453,
			OrderListID:                -1,
			ClientOrderID:              "4d96324ff9d44481926157ec08158a40",
			TransactTime:               types.Time(time.UnixMilli(1660801715793)),
			Price:                      23416.1,
			OriginalQuantity:           0.00847,
			ExecutedQuantity:           0.00847,
			OriginalQuoteOrderQuantity: 198.335215,
			CummulativeQuoteQuantity:   198.335215,
			Status:                     "FILLED",
			TimeInForce:                "GTC",
			Type:                       "LIMIT",
			Side:                       "SELL",
			WorkingTime:                1660801715793,
			SelfTradePreventionMode:    "EXPIRE_MAKER",
			Fills: []WsOrderFill{
				{
					Price:           23416.1,
					Quantity:        0.00635,
					Commission:      0.0000011,
					CommissionAsset: currency.BNB,
					TradeID:         1650422481,
				},
				{
					Price:           23416.5,
					Quantity:        0.00212,
					Commission:      0.0000004,
					CommissionAsset: currency.BNB,
					TradeID:         1650422482,
				},
			},
			IcebergQuantity:   0.001,
			PreventedMatchID:  3,
			PreventedQuantity: 1.2,
			StopPrice:         23400,
			StrategyID:        37463720,
			StrategyType:      1000000,
			TrailingDelta:     10,
			TrailingTime:      -1,
			UsedSOR:           true,
			WorkingFloor:      "EXCHANGE",
			PegPriceType:      "PRIMARY_PEG",
			PegOffsetType:     "PRICE_LEVEL",
			PegOffsetValue:    5,
			PeggedPrice:       23416.1,
			ExpiryReason:      "INSUFFICIENT_LIQUIDITY",
		}
		assert.Equal(t, exp, got, "WsPlaceNewOrder should return the order and its fills")
	}
	if !mockTests {
		return
	}
	got, err = ex.WsPlaceNewOrder(t.Context(), &WsPlaceOrderRequest{
		Symbol:               currency.NewBTCUSDT(),
		Side:                 "BUY",
		Type:                 "STOP_LOSS",
		Quantity:             0.00847,
		StopPrice:            23400,
		TrailingDelta:        10,
		NewOrderResponseType: "ACK",
	})
	require.NoError(t, err, "WsPlaceNewOrder must not error for a stop order")
	exp := &WsOrderResponse{
		Symbol:        "BTCUSDT",
		OrderID:       12569099455,
		OrderListID:   -1,
		ClientOrderID: "x4hXc0n8Fs3UXwQnFxIjGq",
		TransactTime:  types.Time(time.UnixMilli(1660801715639)),
	}
	assert.Equal(t, exp, got, "WsPlaceNewOrder should return the acknowledgement")
	got, err = ex.WsPlaceNewOrder(t.Context(), &WsPlaceOrderRequest{Symbol: currency.NewBTCUSDT(), Side: "BUY", Type: "MARKET", QuoteOrderQuantity: 100})
	require.NoError(t, err, "WsPlaceNewOrder must not error for a market order by quote quantity")
	assert.Equal(t, uint64(12569099456), got.OrderID, "WsPlaceNewOrder should return the order's ID")
}

func TestWsTestNewOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WsTestNewOrder(t.Context(), nil, false)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsTestNewOrder must reject a nil request")

	ex := newWsAPITestExchange(t, true)
	arg := &WsPlaceOrderRequest{Symbol: currency.NewBTCUSDT(), Side: "SELL", Type: "LIMIT", TimeInForce: "GTC", Price: 23416.1, Quantity: 0.00847}
	got, err := ex.WsTestNewOrder(t.Context(), arg, false)
	require.NoError(t, err, "WsTestNewOrder must not error")
	assert.Nil(t, got, "WsTestNewOrder should return no commission rates unless asked")
	got, err = ex.WsTestNewOrder(t.Context(), arg, true)
	require.NoError(t, err, "WsTestNewOrder must not error computing commission rates")
	if mockTests {
		exp := &WsOrderCommissionRatesResponse{
			StandardCommissionForOrder: WsOrderCommissionRates{
				Maker: 0.00000112,
				Taker: 0.00000114,
			},
			SpecialCommissionForOrder: WsOrderCommissionRates{
				Maker: 0.05,
				Taker: 0.06,
			},
			TaxCommissionForOrder: WsOrderCommissionRates{
				Maker: 0.00000112,
				Taker: 0.00000114,
			},
			Discount: WsCommissionDiscount{
				EnabledForAccount: true,
				EnabledForSymbol:  true,
				DiscountAsset:     currency.BNB,
				Discount:          0.25,
			},
		}
		assert.Equal(t, exp, got, "WsTestNewOrder should return the order's commission rates")
	} else {
		assert.NotNil(t, got, "WsTestNewOrder should return the order's commission rates")
	}
}

func TestWsPlaceNewSOROrder(t *testing.T) {
	t.Parallel()
	_, err := e.WsPlaceNewSOROrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsPlaceNewSOROrder must reject a nil request")
	arg := &WsSOROrderRequest{}
	_, err = e.WsPlaceNewSOROrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "WsPlaceNewSOROrder must require a side")
	arg.Side = "BUY"
	_, err = e.WsPlaceNewSOROrder(t.Context(), arg)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "WsPlaceNewSOROrder must require an order type")
	arg.Type = "LIMIT"
	_, err = e.WsPlaceNewSOROrder(t.Context(), arg)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "WsPlaceNewSOROrder must require a quantity")
	arg.Quantity = 0.5
	_, err = e.WsPlaceNewSOROrder(t.Context(), arg)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "WsPlaceNewSOROrder must reject an empty pair")

	ex := newWsAPITestExchange(t, true, canManipulateRealOrders)
	got, err := ex.WsPlaceNewSOROrder(t.Context(), &WsSOROrderRequest{
		Symbol:                  currency.NewBTCUSDT(),
		Side:                    "BUY",
		Type:                    "LIMIT",
		TimeInForce:             "GTC",
		Price:                   31000,
		Quantity:                0.5,
		NewClientOrderID:        "sBI1KM6nNtOfj5tccZSKly",
		NewOrderResponseType:    "FULL",
		IcebergQuantity:         0.1,
		StrategyID:              37463720,
		StrategyType:            1000000,
		SelfTradePreventionMode: "EXPIRE_MAKER",
	})
	require.NoError(t, err, "WsPlaceNewSOROrder must not error")
	if mockTests {
		exp := []WsSOROrderResponse{
			{
				Symbol:                     "BTCUSDT",
				OrderID:                    2,
				OrderListID:                -1,
				ClientOrderID:              "sBI1KM6nNtOfj5tccZSKly",
				TransactTime:               types.Time(time.UnixMilli(1689149087774)),
				Price:                      31000,
				OriginalQuantity:           0.5,
				ExecutedQuantity:           0.5,
				OriginalQuoteOrderQuantity: 15500,
				CummulativeQuoteQuantity:   14000,
				Status:                     "FILLED",
				TimeInForce:                "GTC",
				Type:                       "LIMIT",
				Side:                       "BUY",
				WorkingTime:                1689149087774,
				Fills: []WsSOROrderFill{
					{
						MatchType:       "ONE_PARTY_TRADE_REPORT",
						Price:           28000,
						Quantity:        0.5,
						Commission:      0.0001,
						CommissionAsset: currency.BTC,
						TradeID:         -1,
						AllocationID:    7,
					},
				},
				WorkingFloor:            "SOR",
				SelfTradePreventionMode: "EXPIRE_MAKER",
				UsedSOR:                 true,
				StopPrice:               30900,
				TrailingDelta:           10,
				IcebergQuantity:         0.1,
				StrategyID:              37463720,
				StrategyType:            1000000,
				PreventedMatchID:        3,
				PreventedQuantity:       1.2,
				TrailingTime:            -1,
				PegPriceType:            "PRIMARY_PEG",
				PegOffsetType:           "PRICE_LEVEL",
				PegOffsetValue:          5,
				PeggedPrice:             31000,
				ExpiryReason:            "INSUFFICIENT_LIQUIDITY",
			},
		}
		assert.Equal(t, exp, got, "WsPlaceNewSOROrder should return the order and its fills")
	}
}

func TestWsTestNewOrderUsingSOR(t *testing.T) {
	t.Parallel()
	_, err := e.WsTestNewOrderUsingSOR(t.Context(), nil, false)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsTestNewOrderUsingSOR must reject a nil request")

	ex := newWsAPITestExchange(t, true)
	arg := &WsSOROrderRequest{Symbol: currency.NewBTCUSDT(), Side: "BUY", Type: "MARKET", Quantity: 0.5}
	got, err := ex.WsTestNewOrderUsingSOR(t.Context(), arg, false)
	require.NoError(t, err, "WsTestNewOrderUsingSOR must not error")
	assert.Nil(t, got, "WsTestNewOrderUsingSOR should return no commission rates unless asked")
	got, err = ex.WsTestNewOrderUsingSOR(t.Context(), arg, true)
	require.NoError(t, err, "WsTestNewOrderUsingSOR must not error computing commission rates")
	if mockTests {
		exp := &WsSOROrderCommissionRatesResponse{
			StandardCommissionForOrder: WsOrderCommissionRates{
				Maker: 0.00000112,
				Taker: 0.00000114,
			},
			TaxCommissionForOrder: WsOrderCommissionRates{
				Maker: 0.00000112,
				Taker: 0.00000114,
			},
			Discount: WsCommissionDiscount{
				EnabledForAccount: true,
				EnabledForSymbol:  true,
				DiscountAsset:     currency.BNB,
				Discount:          0.25,
			},
		}
		assert.Equal(t, exp, got, "WsTestNewOrderUsingSOR should return the order's commission rates")
	} else {
		assert.NotNil(t, got, "WsTestNewOrderUsingSOR should return the order's commission rates")
	}
}

func TestWsAccountCommissionRates(t *testing.T) {
	t.Parallel()
	_, err := e.WsAccountCommissionRates(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "WsAccountCommissionRates must reject an empty pair")

	ex := newWsAPITestExchange(t, true)
	got, err := ex.WsAccountCommissionRates(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "WsAccountCommissionRates must not error")
	if mockTests {
		exp := &WsAccountCommissionResponse{
			Symbol: "BTCUSDT",
			StandardCommission: WsAccountCommissionRates{
				Maker:  0.0000001,
				Taker:  0.0000002,
				Buyer:  0.0000003,
				Seller: 0.0000004,
			},
			SpecialCommission: WsAccountCommissionRates{
				Maker:  0.01,
				Taker:  0.02,
				Buyer:  0.03,
				Seller: 0.04,
			},
			TaxCommission: WsAccountCommissionRates{
				Maker:  0.00000112,
				Taker:  0.00000114,
				Buyer:  0.00000118,
				Seller: 0.00000116,
			},
			Discount: WsCommissionDiscount{
				EnabledForAccount: true,
				EnabledForSymbol:  true,
				DiscountAsset:     currency.BNB,
				Discount:          0.75,
			},
		}
		assert.Equal(t, exp, got, "WsAccountCommissionRates should return the commission rates")
	} else {
		assert.Equal(t, "BTCUSDT", got.Symbol, "WsAccountCommissionRates should return the symbol's commission rates")
	}
}

func TestWsQueryAccountOrderRateLimits(t *testing.T) {
	t.Parallel()
	ex := newWsAPITestExchange(t, true)
	got, err := ex.WsQueryAccountOrderRateLimits(t.Context())
	require.NoError(t, err, "WsQueryAccountOrderRateLimits must not error")
	if mockTests {
		exp := []WsRateLimit{
			{
				RateLimitType:  "ORDERS",
				Interval:       "SECOND",
				IntervalNumber: 10,
				Limit:          50,
				Count:          3,
			},
			{
				RateLimitType:  "ORDERS",
				Interval:       "DAY",
				IntervalNumber: 1,
				Limit:          160000,
				Count:          4043,
			},
		}
		assert.Equal(t, exp, got, "WsQueryAccountOrderRateLimits should return the unfilled order counts")
	} else {
		assert.NotEmpty(t, got, "WsQueryAccountOrderRateLimits should return the unfilled order counts")
	}
}

func TestGetWsAccountInfo(t *testing.T) {
	t.Parallel()
	ex := newWsAPITestExchange(t, true)
	got, err := ex.GetWsAccountInfo(t.Context(), true)
	require.NoError(t, err, "GetWsAccountInfo must not error")
	if mockTests {
		assert.Equal(t, wsAPIFixtureResult[*AccountResponse](t, "account.status", 0), got, "GetWsAccountInfo should return the account")
	} else {
		assert.NotNil(t, got, "GetWsAccountInfo should return the account")
	}
}

func TestWsQueryAccountOCOOrderHistory(t *testing.T) {
	t.Parallel()
	start, end := wsAPITimeRange()
	_, err := e.WsQueryAccountOCOOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsQueryAccountOCOOrderHistory must reject a nil request")
	_, err = e.WsQueryAccountOCOOrderHistory(t.Context(), &WsOrderListHistoryRequest{StartTime: end, EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "WsQueryAccountOCOOrderHistory must reject a start time after the end time")

	ex := newWsAPITestExchange(t, true)
	got, err := ex.WsQueryAccountOCOOrderHistory(t.Context(), &WsOrderListHistoryRequest{StartTime: start, EndTime: end, Limit: 2})
	require.NoError(t, err, "WsQueryAccountOCOOrderHistory must not error for a time range")
	if mockTests {
		assert.Equal(t, wsAPIFixtureResult[[]OCOOrderResponse](t, "allOrderLists", 0), got, "WsQueryAccountOCOOrderHistory should return the order lists in the time range")
	}
	got, err = ex.WsQueryAccountOCOOrderHistory(t.Context(), &WsOrderListHistoryRequest{FromID: 1274512})
	require.NoError(t, err, "WsQueryAccountOCOOrderHistory must not error from an order list ID")
	if mockTests {
		assert.Equal(t, wsAPIFixtureResult[[]OCOOrderResponse](t, "allOrderLists", 1), got, "WsQueryAccountOCOOrderHistory should return the order lists from the ID")
	}
}

func TestWsQueryAccountOrderHistory(t *testing.T) {
	t.Parallel()
	start, end := wsAPITimeRange()
	_, err := e.WsQueryAccountOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsQueryAccountOrderHistory must reject a nil request")
	_, err = e.WsQueryAccountOrderHistory(t.Context(), &WsOrderHistoryRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "WsQueryAccountOrderHistory must reject an empty pair")
	_, err = e.WsQueryAccountOrderHistory(t.Context(), &WsOrderHistoryRequest{Symbol: currency.NewBTCUSDT(), StartTime: end, EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "WsQueryAccountOrderHistory must reject a start time after the end time")

	ex := newWsAPITestExchange(t, true)
	got, err := ex.WsQueryAccountOrderHistory(t.Context(), &WsOrderHistoryRequest{Symbol: currency.NewBTCUSDT(), OrderID: 12569099453, Limit: 2})
	require.NoError(t, err, "WsQueryAccountOrderHistory must not error from an order ID")
	if mockTests {
		assert.Equal(t, wsAPIFixtureResult[[]TradeOrderResponse](t, "allOrders", 0), got, "WsQueryAccountOrderHistory should return the orders from the ID")
	}
	got, err = ex.WsQueryAccountOrderHistory(t.Context(), &WsOrderHistoryRequest{Symbol: currency.NewBTCUSDT(), StartTime: start, EndTime: end})
	require.NoError(t, err, "WsQueryAccountOrderHistory must not error for a time range")
	if mockTests {
		assert.Equal(t, wsAPIFixtureResult[[]TradeOrderResponse](t, "allOrders", 1), got, "WsQueryAccountOrderHistory should return the orders in the time range")
	}
}

func TestWsAccountAllocation(t *testing.T) {
	t.Parallel()
	start, end := wsAPITimeRange()
	_, err := e.WsAccountAllocation(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsAccountAllocation must reject a nil request")
	_, err = e.WsAccountAllocation(t.Context(), &WsAllocationsRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "WsAccountAllocation must reject an empty pair")
	_, err = e.WsAccountAllocation(t.Context(), &WsAllocationsRequest{Symbol: currency.NewBTCUSDT(), StartTime: end, EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "WsAccountAllocation must reject a start time after the end time")

	ex := newWsAPITestExchange(t, true)
	got, err := ex.WsAccountAllocation(t.Context(), &WsAllocationsRequest{Symbol: currency.NewBTCUSDT(), StartTime: start, EndTime: end, Limit: 2})
	require.NoError(t, err, "WsAccountAllocation must not error for a time range")
	if mockTests {
		exp := []WsAllocation{
			{
				Symbol:          "BTCUSDT",
				AllocationID:    1,
				AllocationType:  "SOR",
				OrderID:         500,
				OrderListID:     -1,
				Price:           1,
				Quantity:        0.1,
				QuoteQuantity:   0.1,
				Commission:      0.0000001,
				CommissionAsset: currency.BTC,
				Time:            types.Time(time.UnixMilli(1687319487614)),
				IsBuyer:         true,
				IsMaker:         true,
				IsAllocator:     true,
			},
		}
		assert.Equal(t, exp, got, "WsAccountAllocation should return the allocations in the time range")
		got, err = ex.WsAccountAllocation(t.Context(), &WsAllocationsRequest{Symbol: currency.NewBTCUSDT(), OrderID: 500, FromAllocationID: 1})
		require.NoError(t, err, "WsAccountAllocation must not error for an order")
		assert.Equal(t, exp, got, "WsAccountAllocation should return the order's allocations")
	}
}

func TestWsAccountPreventedMatches(t *testing.T) {
	t.Parallel()
	_, err := e.WsAccountPreventedMatches(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsAccountPreventedMatches must reject a nil request")
	_, err = e.WsAccountPreventedMatches(t.Context(), &WsPreventedMatchesRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "WsAccountPreventedMatches must require a prevented match ID or an order ID")
	_, err = e.WsAccountPreventedMatches(t.Context(), &WsPreventedMatchesRequest{PreventedMatchID: 1})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "WsAccountPreventedMatches must reject an empty pair")

	ex := newWsAPITestExchange(t, true)
	got, err := ex.WsAccountPreventedMatches(t.Context(), &WsPreventedMatchesRequest{Symbol: currency.NewBTCUSDT(), PreventedMatchID: 1})
	require.NoError(t, err, "WsAccountPreventedMatches must not error for a prevented match ID")
	if mockTests {
		exp := []WsPreventedMatch{
			{
				Symbol:                  "BTCUSDT",
				PreventedMatchID:        1,
				TakerOrderID:            5,
				MakerSymbol:             "BTCUSDT",
				MakerOrderID:            3,
				TradeGroupID:            1,
				SelfTradePreventionMode: "EXPIRE_MAKER",
				Price:                   1.1,
				MakerPreventedQuantity:  1.3,
				TransactTime:            types.Time(time.UnixMilli(1669101687094)),
			},
		}
		assert.Equal(t, exp, got, "WsAccountPreventedMatches should return the prevented match")
		got, err = ex.WsAccountPreventedMatches(t.Context(), &WsPreventedMatchesRequest{Symbol: currency.NewBTCUSDT(), OrderID: 5, FromPreventedMatchID: 1, Limit: 2})
		require.NoError(t, err, "WsAccountPreventedMatches must not error for an order")
		assert.Equal(t, exp, got, "WsAccountPreventedMatches should return the order's prevented matches")
	}
}

func TestWsAccountTradeHistory(t *testing.T) {
	t.Parallel()
	start, end := wsAPITimeRange()
	_, err := e.WsAccountTradeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WsAccountTradeHistory must reject a nil request")
	_, err = e.WsAccountTradeHistory(t.Context(), &WsAccountTradesRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "WsAccountTradeHistory must reject an empty pair")
	_, err = e.WsAccountTradeHistory(t.Context(), &WsAccountTradesRequest{Symbol: currency.NewBTCUSDT(), StartTime: end, EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "WsAccountTradeHistory must reject a start time after the end time")

	ex := newWsAPITestExchange(t, true)
	got, err := ex.WsAccountTradeHistory(t.Context(), &WsAccountTradesRequest{Symbol: currency.NewBTCUSDT(), OrderID: 12569099453, Limit: 2})
	require.NoError(t, err, "WsAccountTradeHistory must not error for an order")
	if mockTests {
		exp := []WsAccountTrade{
			{
				Symbol:          "BTCUSDT",
				ID:              1650422481,
				OrderID:         12569099453,
				OrderListID:     -1,
				Price:           23416.1,
				Quantity:        0.00635,
				QuoteQuantity:   148.692235,
				Commission:      0.0000011,
				CommissionAsset: currency.BNB,
				Time:            types.Time(time.UnixMilli(1660801715793)),
				IsBuyer:         true,
				IsMaker:         true,
				IsBestMatch:     true,
			},
		}
		assert.Equal(t, exp, got, "WsAccountTradeHistory should return the order's trades")
		got, err = ex.WsAccountTradeHistory(t.Context(), &WsAccountTradesRequest{Symbol: currency.NewBTCUSDT(), FromID: 1650422481})
		require.NoError(t, err, "WsAccountTradeHistory must not error from a trade ID")
		assert.Equal(t, exp, got, "WsAccountTradeHistory should return the trades from the ID")
	}
	got, err = ex.WsAccountTradeHistory(t.Context(), &WsAccountTradesRequest{Symbol: currency.NewBTCUSDT(), StartTime: start, EndTime: end})
	require.NoError(t, err, "WsAccountTradeHistory must not error for a time range")
	if mockTests {
		assert.Empty(t, got, "WsAccountTradeHistory should return no trades for a time range without any")
	}
}

func TestWsCurrentOpenOCOOrders(t *testing.T) {
	t.Parallel()
	ex := newWsAPITestExchange(t, true)
	got, err := ex.WsCurrentOpenOCOOrders(t.Context())
	require.NoError(t, err, "WsCurrentOpenOCOOrders must not error")
	if mockTests {
		assert.Equal(t, wsAPIFixtureResult[[]OCOOrderResponse](t, "openOrderLists.status", 0), got, "WsCurrentOpenOCOOrders should return the open order lists")
	}
}

func TestWsCurrentOpenOrders(t *testing.T) {
	t.Parallel()
	ex := newWsAPITestExchange(t, true)
	got, err := ex.WsCurrentOpenOrders(t.Context(), currency.NewBTCUSDT())
	require.NoError(t, err, "WsCurrentOpenOrders must not error for a symbol")
	if mockTests {
		assert.Equal(t, wsAPIFixtureResult[[]TradeOrderResponse](t, "openOrders.status", 0), got, "WsCurrentOpenOrders should return the symbol's open orders")
	}
	got, err = ex.WsCurrentOpenOrders(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "WsCurrentOpenOrders must not error for every symbol")
	if mockTests {
		assert.Equal(t, wsAPIFixtureResult[[]TradeOrderResponse](t, "openOrders.status", 1), got, "WsCurrentOpenOrders should return every open order")
	}
}

func TestWsQueryOCOOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WsQueryOCOOrder(t.Context(), 0, "")
	require.ErrorIs(t, err, errOrderListIDRequired, "WsQueryOCOOrder must require an order list ID or client order list ID")

	ex := newWsAPITestExchange(t, true)
	got, err := ex.WsQueryOCOOrder(t.Context(), 1274512, "")
	require.NoError(t, err, "WsQueryOCOOrder must not error for an order list ID")
	if mockTests {
		assert.Equal(t, wsAPIFixtureResult[*OCOOrderResponse](t, "orderList.status", 0), got, "WsQueryOCOOrder should return the order list")
		got, err = ex.WsQueryOCOOrder(t.Context(), 0, "08985fedd9ea2cf6b28996")
		require.NoError(t, err, "WsQueryOCOOrder must not error for a client order list ID")
		assert.Equal(t, wsAPIFixtureResult[*OCOOrderResponse](t, "orderList.status", 1), got, "WsQueryOCOOrder should return the order list")
	}
}

func TestWsQueryOrder(t *testing.T) {
	t.Parallel()
	_, err := e.WsQueryOrder(t.Context(), currency.NewBTCUSDT(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "WsQueryOrder must require an order ID or client order ID")
	_, err = e.WsQueryOrder(t.Context(), currency.EMPTYPAIR, 12569099453, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "WsQueryOrder must reject an empty pair")

	ex := newWsAPITestExchange(t, true)
	got, err := ex.WsQueryOrder(t.Context(), currency.NewBTCUSDT(), 12569099453, "4d96324ff9d44481926157")
	require.NoError(t, err, "WsQueryOrder must not error")
	if mockTests {
		assert.Equal(t, wsAPIFixtureResult[*TradeOrderResponse](t, "order.status", 0), got, "WsQueryOrder should return the order")
	}
}

func TestWsSubscribeUserDataStream(t *testing.T) {
	t.Parallel()
	_, err := e.WsSubscribeUserDataStream(t.Context())
	require.ErrorIs(t, err, websocket.ErrNotConnected, "WsSubscribeUserDataStream must error without a connection")

	ex := newWsAPITestExchange(t, true)
	got, err := ex.WsSubscribeUserDataStream(t.Context())
	require.NoError(t, err, "WsSubscribeUserDataStream must not error")
	if mockTests {
		exp := &WsUserDataStreamSubscriptionResponse{
			SubscriptionID: 1,
		}
		assert.Equal(t, exp, got, "WsSubscribeUserDataStream should return the subscription")
	}
	assert.Equal(t, map[uint64]asset.Item{got.SubscriptionID: asset.Spot}, userDataAssets(ex), "the subscription should deliver the spot account's events")
}

func TestWsSubscribeUserDataStreamWithListenToken(t *testing.T) {
	t.Parallel()
	_, err := e.WsSubscribeUserDataStreamWithListenToken(t.Context(), "6xXxePXwZRjVSHKhzUCCGnmN3fkvMTXru+pYJS8RwijXk9Vcyr3rkwfVOTcP2OkONqciYA")
	require.ErrorIs(t, err, websocket.ErrNotConnected, "WsSubscribeUserDataStreamWithListenToken must error without a connection")

	ex := newWsAPITestExchange(t, true)
	_, err = ex.WsSubscribeUserDataStreamWithListenToken(t.Context(), "")
	require.ErrorIs(t, err, errListenTokenRequired, "WsSubscribeUserDataStreamWithListenToken must require a listen token")
	listenToken := "6xXxePXwZRjVSHKhzUCCGnmN3fkvMTXru+pYJS8RwijXk9Vcyr3rkwfVOTcP2OkONqciYA"
	if !mockTests {
		token, err := ex.CreateMarginListenToken(t.Context(), &MarginListenTokenRequest{})
		require.NoError(t, err, "CreateMarginListenToken must not error")
		listenToken = token.Token
	}
	got, err := ex.WsSubscribeUserDataStreamWithListenToken(t.Context(), listenToken)
	require.NoError(t, err, "WsSubscribeUserDataStreamWithListenToken must not error")
	if mockTests {
		exp := &WsListenTokenSubscriptionResponse{
			SubscriptionID: 2,
			ExpirationTime: types.Time(time.UnixMicro(1749094553955907)),
		}
		assert.Equal(t, exp, got, "WsSubscribeUserDataStreamWithListenToken should return the subscription")
	}
	assert.Equal(t, map[uint64]asset.Item{got.SubscriptionID: asset.Margin}, userDataAssets(ex), "the subscription should deliver the margin account's events")
}

func TestWsAuthenticateAPI(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
	}
	setup := ex.spotWebsocketAPIConnectionSetup(ex.Config)
	if mockTests {
		useMockWebsocket(t, ex, setup, wsAPIMock)
	} else {
		useWebsocket(t, ex, setup)
	}
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	if mockTests {
		assert.Equal(t, map[uint64]asset.Item{1: asset.Spot, 2: asset.Margin}, userDataAssets(ex), "Authenticate should subscribe to the spot and margin accounts' User Data Streams")
	} else {
		assert.Contains(t, slices.Collect(maps.Values(userDataAssets(ex))), asset.Spot, "Authenticate should subscribe to the spot account's User Data Stream")
	}

	ex = newTestExchange(t)
	ex.SkipAuthCheck = false
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{})
	useMockWebsocket(t, ex, ex.spotWebsocketAPIConnectionSetup(ex.Config), wsAPIMock)
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error when the User Data Streams are unavailable")
	assert.True(t, ex.IsAPIStreamConnected(), "the WebSocket API should stay connected without User Data Streams")
	assert.Empty(t, userDataAssets(ex), "Authenticate should not record failed subscriptions")
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		require.NotEmptyf(t, ex.Websocket.DataHandler.C, "relay must hold the %s User Data Stream error", a)
		err, ok := (<-ex.Websocket.DataHandler.C).Data.(error)
		require.Truef(t, ok, "relay must hold an error for %s", a)
		assert.ErrorIsf(t, err, exchange.ErrCredentialsAreEmpty, "the %s User Data Stream error should be the missing credentials", a)
		assert.ErrorContainsf(t, err, a.String(), "the error should name %s", a)
	}

	ex = newTestExchange(t)
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Spot, false), "SetAssetEnabled must not error")
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Margin, false), "SetAssetEnabled must not error")
	require.NoError(t, ex.WsAuthenticateAPI(t.Context(), nil), "WsAuthenticateAPI must not error with spot and margin disabled")
	assert.Empty(t, userDataAssets(ex), "WsAuthenticateAPI should not subscribe for disabled assets")
}

func TestWsHandleSpotAPIData(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ex.SkipAuthCheck = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	ex.userData.expect("spot", asset.Spot)
	ex.userData.expect("margin", asset.Margin)
	ex.userData.recordResponse("spot", []byte(`{"id":"spot","status":200,"result":{"subscriptionId":0}}`))
	ex.userData.recordResponse("margin", []byte(`{"id":"margin","status":200,"result":{"subscriptionId":1,"expirationTime":1749094553955907}}`))
	for _, line := range fixtureLines(t, "testdata/wsSpotUserData.json") {
		require.NoErrorf(t, ex.wsHandleSpotAPIData(t.Context(), nil, line), "wsHandleSpotAPIData must not error for %s", line)
	}
	pair := currency.NewPairWithDelimiter("BTC", "USDT", "-")
	exp := []any{
		accounts.SubAccounts{{AssetType: asset.Spot, Balances: accounts.CurrencyBalances{
			currency.BTC:  {Currency: currency.BTC, Total: 0.26, Hold: 0.01, Free: 0.25},
			currency.USDT: {Currency: currency.USDT, Total: 1250, Hold: 250, Free: 1000},
		}}},
		accounts.SubAccounts{{AssetType: asset.Margin, Balances: accounts.CurrencyBalances{
			currency.BTC: {Currency: currency.BTC, Total: 0.52, Hold: 0.02, Free: 0.5, AvailableWithoutBorrow: 0.5},
		}}},
		&WsBalanceUpdate{
			EventType:    "balanceUpdate",
			EventTime:    types.Time(time.UnixMilli(1791285058331)),
			Asset:        currency.USDT,
			BalanceDelta: 100,
			ClearTime:    types.Time(time.UnixMilli(1791285058329)),
		},
		&order.Detail{
			Price:           86000,
			Amount:          0.01,
			RemainingAmount: 0.01,
			CostAsset:       currency.USDT,
			Exchange:        t.Name(),
			OrderID:         "5340845958",
			ClientOrderID:   "web_6a1b2c3d4e5f",
			Type:            order.Limit,
			Side:            order.Buy,
			Status:          order.New,
			AssetType:       asset.Spot,
			Date:            time.UnixMilli(1791285058999),
			LastUpdated:     time.UnixMilli(1791285058999),
			Pair:            pair,
			TimeInForce:     order.GoodTillCancel,
		},
		&order.Detail{
			Price:                86000,
			Amount:               0.01,
			AverageExecutedPrice: 86000,
			ExecutedAmount:       0.004,
			RemainingAmount:      0.006,
			Cost:                 344,
			CostAsset:            currency.USDT,
			FeeAsset:             currency.BTC,
			Exchange:             t.Name(),
			OrderID:              "5340845958",
			ClientOrderID:        "web_6a1b2c3d4e5f",
			Type:                 order.Limit,
			Side:                 order.Buy,
			Status:               order.PartiallyFilled,
			AssetType:            asset.Spot,
			Date:                 time.UnixMilli(1791285058999),
			LastUpdated:          time.UnixMilli(1791285059999),
			Pair:                 pair,
			TimeInForce:          order.GoodTillCancel,
			Trades: []order.TradeHistory{{
				Price:     86000,
				Amount:    0.004,
				Fee:       0.000004,
				FeeAsset:  "BTC",
				Exchange:  t.Name(),
				TID:       "6739450700",
				Type:      order.Limit,
				Side:      order.Buy,
				Timestamp: time.UnixMilli(1791285059999),
				IsMaker:   true,
				Total:     344,
			}},
		},
		&order.Detail{
			Price:           87000,
			Amount:          0.02,
			RemainingAmount: 0.02,
			CostAsset:       currency.USDT,
			Exchange:        t.Name(),
			OrderID:         "5340845960",
			ClientOrderID:   "web_1a2b3c",
			Type:            order.LimitMaker,
			Side:            order.Sell,
			Status:          order.Cancelled,
			AssetType:       asset.Margin,
			Date:            time.UnixMilli(1791285050000),
			LastUpdated:     time.UnixMilli(1791285060999),
			Pair:            pair,
			TimeInForce:     order.GoodTillCancel,
		},
		&order.Detail{
			Price:           90000,
			Amount:          0.01,
			TriggerPrice:    89000,
			RemainingAmount: 0.01,
			CostAsset:       currency.USDT,
			Exchange:        t.Name(),
			OrderID:         "5340845961",
			ClientOrderID:   "oto_pending_leg",
			Type:            order.TakeProfitLimit,
			Side:            order.Sell,
			Status:          order.Pending,
			AssetType:       asset.Spot,
			Date:            time.UnixMilli(1791285061999),
			LastUpdated:     time.UnixMilli(1791285061999),
			Pair:            pair,
			TimeInForce:     order.GoodTillCancel,
		},
		&order.Detail{
			Price:                0.1026441,
			Amount:               1,
			TriggerPrice:         0.102,
			AverageExecutedPrice: 0.1026441,
			QuoteAmount:          0.05,
			ExecutedAmount:       0.3,
			RemainingAmount:      0.7,
			Cost:                 0.03079323,
			CostAsset:            currency.USDT,
			FeeAsset:             currency.BNB,
			Exchange:             t.Name(),
			OrderID:              "4293153",
			ClientOrderID:        "mUvoqJxFIILMdfAW5iGSOW",
			Type:                 order.StopLimit,
			Side:                 order.Sell,
			Status:               order.Expired,
			AssetType:            asset.Spot,
			Date:                 time.UnixMilli(1499405658657),
			LastUpdated:          time.UnixMilli(1499405658657),
			Pair:                 pair,
			TimeInForce:          order.GoodTillCancel,
		},
		&WsListStatus{
			EventType:         "listStatus",
			EventTime:         types.Time(time.UnixMilli(1564035303637)),
			Symbol:            "BTCUSDT",
			OrderListID:       2,
			ContingencyType:   "OCO",
			ListStatusType:    "EXEC_STARTED",
			ListOrderStatus:   "EXECUTING",
			ListRejectReason:  "NONE",
			ListClientOrderID: "F4QN4G8DlFATFlIUQ0cjdD",
			TransactionTime:   types.Time(time.UnixMilli(1564035303625)),
			Orders: []WsListStatusOrder{
				{
					Symbol:        "BTCUSDT",
					OrderID:       17,
					ClientOrderID: "AJYsMjErWJesZvqlJCTUgL",
				},
				{
					Symbol:        "BTCUSDT",
					OrderID:       18,
					ClientOrderID: "bfYPSQdLoqAJeNrOr9adzq",
				},
			},
		},
		&order.Detail{
			Exchange:      t.Name(),
			OrderID:       "2",
			ClientOrderID: "F4QN4G8DlFATFlIUQ0cjdD",
			Type:          order.OCO,
			Status:        order.Active,
			AssetType:     asset.Spot,
			Date:          time.UnixMilli(1564035303625),
			LastUpdated:   time.UnixMilli(1564035303625),
			Pair:          pair,
		},
		&WsExternalLockUpdate{
			EventType:       "externalLockUpdate",
			EventTime:       types.Time(time.UnixMilli(1581557507324)),
			Asset:           currency.BNB,
			Delta:           10,
			TransactionTime: types.Time(time.UnixMilli(1581557507268)),
		},
	}
	for i := range exp {
		require.NotEmptyf(t, ex.Websocket.DataHandler.C, "relay must hold payload %d", i)
		got := (<-ex.Websocket.DataHandler.C).Data
		if subAccounts, ok := got.(accounts.SubAccounts); ok {
			// Balances are stamped with their arrival time
			for _, subAccount := range subAccounts {
				for c, balance := range subAccount.Balances {
					assert.WithinDurationf(t, time.Now(), balance.UpdatedAt, time.Minute, "%s balance should be stamped with its arrival time", c)
					balance.UpdatedAt = time.Time{}
					subAccount.Balances[c] = balance
				}
			}
		}
		assert.Equalf(t, exp[i], got, "relayed payload %d should match", i)
	}
	assert.Empty(t, ex.Websocket.DataHandler.C, "relay should hold nothing else")

	creds := &accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"}
	balance, err := ex.Accounts.GetBalance("", creds, asset.Spot, currency.USDT)
	require.NoError(t, err, "GetBalance must not error")
	assert.Equal(t, 1250.0, balance.Total, "the spot balance should be stored")
	balance, err = ex.Accounts.GetBalance("", creds, asset.Margin, currency.BTC)
	require.NoError(t, err, "GetBalance must not error")
	assert.Equal(t, 0.52, balance.Total, "the margin balance should be stored")
}

func TestUserDataEventPayloads(t *testing.T) {
	t.Parallel()
	exp := []any{
		&WsAccountPosition{
			EventType:             "outboundAccountPosition",
			EventTime:             types.Time(time.UnixMilli(1791285058131)),
			LastAccountUpdateTime: types.Time(time.UnixMilli(1791285058130)),
			Balances: []WsAccountBalance{
				{
					Asset:  currency.BTC,
					Free:   0.25,
					Locked: 0.01,
				},
				{
					Asset:  currency.USDT,
					Free:   1000,
					Locked: 250,
				},
			},
		},
		&WsAccountPosition{
			EventType:             "outboundAccountPosition",
			EventTime:             types.Time(time.UnixMilli(1791285058231)),
			LastAccountUpdateTime: types.Time(time.UnixMilli(1791285058230)),
			Balances: []WsAccountBalance{
				{
					Asset:  currency.BTC,
					Free:   0.5,
					Locked: 0.02,
				},
			},
		},
		&WsBalanceUpdate{
			EventType:    "balanceUpdate",
			EventTime:    types.Time(time.UnixMilli(1791285058331)),
			Asset:        currency.USDT,
			BalanceDelta: 100,
			ClearTime:    types.Time(time.UnixMilli(1791285058329)),
		},
		&WsExecutionReport{
			EventType:               "executionReport",
			EventTime:               types.Time(time.Unix(1791285059, 0)),
			Symbol:                  "BTCUSDT",
			ClientOrderID:           "web_6a1b2c3d4e5f",
			Side:                    "BUY",
			OrderType:               "LIMIT",
			TimeInForce:             "GTC",
			OrderQuantity:           0.01,
			OrderPrice:              86000,
			OrderListID:             -1,
			CurrentExecutionType:    "NEW",
			CurrentOrderStatus:      "NEW",
			OrderRejectReason:       "NONE",
			OrderID:                 5340845958,
			TransactionTime:         types.Time(time.UnixMilli(1791285058999)),
			TradeID:                 -1,
			ExecutionID:             11388173160,
			IsOrderOnBook:           true,
			OrderCreationTime:       types.Time(time.UnixMilli(1791285058999)),
			WorkingTime:             types.Time(time.UnixMilli(1791285058999)),
			SelfTradePreventionMode: "EXPIRE_MAKER",
		},
		&WsExecutionReport{
			EventType:                              "executionReport",
			EventTime:                              types.Time(time.Unix(1791285060, 0)),
			Symbol:                                 "BTCUSDT",
			ClientOrderID:                          "web_6a1b2c3d4e5f",
			Side:                                   "BUY",
			OrderType:                              "LIMIT",
			TimeInForce:                            "GTC",
			OrderQuantity:                          0.01,
			OrderPrice:                             86000,
			OrderListID:                            -1,
			CurrentExecutionType:                   "TRADE",
			CurrentOrderStatus:                     "PARTIALLY_FILLED",
			OrderRejectReason:                      "NONE",
			OrderID:                                5340845958,
			LastExecutedQuantity:                   0.004,
			CumulativeFilledQuantity:               0.004,
			LastExecutedPrice:                      86000,
			CommissionAmount:                       0.000004,
			CommissionAsset:                        currency.BTC,
			TransactionTime:                        types.Time(time.UnixMilli(1791285059999)),
			TradeID:                                6739450700,
			ExecutionID:                            11388173170,
			IsOrderOnBook:                          true,
			IsMaker:                                true,
			Ignore:                                 true,
			OrderCreationTime:                      types.Time(time.UnixMilli(1791285058999)),
			CumulativeQuoteAssetTransactedQuantity: 344,
			LastQuoteAssetTransactedQuantity:       344,
			WorkingTime:                            types.Time(time.UnixMilli(1791285058999)),
			SelfTradePreventionMode:                "EXPIRE_MAKER",
		},
		&WsExecutionReport{
			EventType:               "executionReport",
			EventTime:               types.Time(time.Unix(1791285061, 0)),
			Symbol:                  "BTCUSDT",
			ClientOrderID:           "cancel_7f8e9d",
			Side:                    "SELL",
			OrderType:               "LIMIT_MAKER",
			TimeInForce:             "GTC",
			OrderQuantity:           0.02,
			OrderPrice:              87000,
			OrderListID:             -1,
			OriginalClientOrderID:   "web_1a2b3c",
			CurrentExecutionType:    "CANCELED",
			CurrentOrderStatus:      "CANCELED",
			OrderRejectReason:       "NONE",
			OrderID:                 5340845960,
			TransactionTime:         types.Time(time.UnixMilli(1791285060999)),
			TradeID:                 -1,
			ExecutionID:             11388173180,
			OrderCreationTime:       types.Time(time.Unix(1791285050, 0)),
			SelfTradePreventionMode: "NONE",
		},
		&WsExecutionReport{
			EventType:               "executionReport",
			EventTime:               types.Time(time.Unix(1791285062, 0)),
			Symbol:                  "BTCUSDT",
			ClientOrderID:           "oto_pending_leg",
			Side:                    "SELL",
			OrderType:               "TAKE_PROFIT_LIMIT",
			TimeInForce:             "GTC",
			OrderQuantity:           0.01,
			OrderPrice:              90000,
			StopPrice:               89000,
			OrderListID:             21,
			CurrentExecutionType:    "NEW",
			CurrentOrderStatus:      "PENDING_NEW",
			OrderRejectReason:       "NONE",
			OrderID:                 5340845961,
			TransactionTime:         types.Time(time.UnixMilli(1791285061999)),
			TradeID:                 -1,
			ExecutionID:             11388173190,
			OrderCreationTime:       types.Time(time.UnixMilli(1791285061999)),
			SelfTradePreventionMode: "NONE",
		},
		&WsExecutionReport{
			EventType:                              "executionReport",
			EventTime:                              types.Time(time.UnixMilli(1499405658658)),
			Symbol:                                 "BTCUSDT",
			ClientOrderID:                          "mUvoqJxFIILMdfAW5iGSOW",
			Side:                                   "SELL",
			OrderType:                              "STOP_LOSS_LIMIT",
			TimeInForce:                            "GTC",
			OrderQuantity:                          1,
			OrderPrice:                             0.1026441,
			StopPrice:                              0.102,
			IcebergQuantity:                        0.1,
			OrderListID:                            2,
			OriginalClientOrderID:                  "iuVNVJYYrByz6C4yGOPPK0",
			CurrentExecutionType:                   "TRADE_PREVENTION",
			CurrentOrderStatus:                     "EXPIRED_IN_MATCH",
			OrderRejectReason:                      "NONE",
			OrderID:                                4293153,
			LastExecutedQuantity:                   0.2,
			CumulativeFilledQuantity:               0.3,
			LastExecutedPrice:                      0.1026441,
			CommissionAmount:                       0.000001,
			CommissionAsset:                        currency.BNB,
			TransactionTime:                        types.Time(time.UnixMilli(1499405658657)),
			TradeID:                                12345,
			PreventedMatchID:                       3,
			ExecutionID:                            8641984,
			IsOrderOnBook:                          true,
			IsMaker:                                true,
			Ignore:                                 true,
			OrderCreationTime:                      types.Time(time.UnixMilli(1499405658657)),
			CumulativeQuoteAssetTransactedQuantity: 0.03079323,
			LastQuoteAssetTransactedQuantity:       0.02052882,
			QuoteOrderQuantity:                     0.05,
			WorkingTime:                            types.Time(time.UnixMilli(1499405658657)),
			SelfTradePreventionMode:                "EXPIRE_TAKER",
			TrailingDelta:                          4,
			TrailingTime:                           1668680518494,
			StrategyID:                             1,
			StrategyType:                           1000000,
			PreventedQuantity:                      3,
			LastPreventedQuantity:                  3,
			TradeGroupID:                           1,
			CounterOrderID:                         37,
			CounterSymbol:                          "BTCUSDT",
			PreventedExecutionQuantity:             2.123456,
			PreventedExecutionPrice:                0.10000001,
			PreventedExecutionQuoteQuantity:        0.21234562,
			MatchType:                              "ONE_PARTY_TRADE_REPORT",
			AllocationID:                           1234,
			WorkingFloor:                           "SOR",
			UsedSOR:                                true,
			PeggedPriceType:                        "PRIMARY_PEG",
			PeggedOffsetType:                       "PRICE_LEVEL",
			PeggedOffsetValue:                      5,
			PeggedPrice:                            1,
			ExpiryReason:                           "INSUFFICIENT_LIQUIDITY",
		},
		&WsListStatus{
			EventType:         "listStatus",
			EventTime:         types.Time(time.UnixMilli(1564035303637)),
			Symbol:            "BTCUSDT",
			OrderListID:       2,
			ContingencyType:   "OCO",
			ListStatusType:    "EXEC_STARTED",
			ListOrderStatus:   "EXECUTING",
			ListRejectReason:  "NONE",
			ListClientOrderID: "F4QN4G8DlFATFlIUQ0cjdD",
			TransactionTime:   types.Time(time.UnixMilli(1564035303625)),
			Orders: []WsListStatusOrder{
				{
					Symbol:        "BTCUSDT",
					OrderID:       17,
					ClientOrderID: "AJYsMjErWJesZvqlJCTUgL",
				},
				{
					Symbol:        "BTCUSDT",
					OrderID:       18,
					ClientOrderID: "bfYPSQdLoqAJeNrOr9adzq",
				},
			},
		},
		&WsExternalLockUpdate{
			EventType:       "externalLockUpdate",
			EventTime:       types.Time(time.UnixMilli(1581557507324)),
			Asset:           currency.BNB,
			Delta:           10,
			TransactionTime: types.Time(time.UnixMilli(1581557507268)),
		},
	}
	lines := fixtureLines(t, "testdata/wsSpotUserData.json")
	require.Len(t, lines, len(exp), "fixture must hold an event per expected payload")
	for i, line := range lines {
		event, _, _, err := jsonparser.Get(line, "event")
		require.NoErrorf(t, err, "event %d must be present", i)
		got := reflect.New(reflect.TypeOf(exp[i]).Elem()).Interface()
		require.NoErrorf(t, json.Unmarshal(event, got), "Unmarshal must not error for event %d", i)
		assert.Equalf(t, exp[i], got, "event %d should decode every field", i)
	}
}

func TestWsHandleSpotAPIDataErrors(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ex.SkipAuthCheck = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	ex.userData.expect("spot", asset.Spot)
	ex.userData.recordResponse("spot", []byte(`{"id":"spot","status":200,"result":{"subscriptionId":0}}`))
	for _, tc := range []struct {
		frame string
		err   error
	}{
		{`{"result":null}`, errUnhandledMessage},
		{`{"event":{"E":1}}`, errUnhandledMessage},
		{`{"event":{"e":"balanceUpdate","E":1}}`, errUserDataEventWithoutSubscription},
		{`{"subscriptionId":9,"event":{"e":"balanceUpdate","E":1}}`, errUnknownUserDataStreamSubscription},
		{`{"subscriptionId":0,"event":{"e":"unknownEvent","E":1}}`, errUnhandledMessage},
		{`{"subscriptionId":0,"event":{"e":"executionReport","E":1791285059000,"s":"BTCXYZ","i":1}}`, currency.ErrPairNotFound},
	} {
		assert.ErrorIsf(t, ex.wsHandleSpotAPIData(t.Context(), nil, []byte(tc.frame)), tc.err, "wsHandleSpotAPIData should error for %s", tc.frame)
	}
	for _, frame := range []string{
		`{"subscriptionId":0,"event":{"e":"outboundAccountPosition","B":"invalid"}}`,
		`{"subscriptionId":0,"event":{"e":"balanceUpdate","d":true}}`,
		`{"subscriptionId":0,"event":{"e":"executionReport","i":"invalid"}}`,
		`{"subscriptionId":0,"event":{"e":"listStatus","O":"invalid"}}`,
		`{"subscriptionId":0,"event":{"e":"externalLockUpdate","d":true}}`,
		`{"subscriptionId":0,"event":{"e":"eventStreamTerminated","E":"invalid"}}`,
	} {
		assert.Errorf(t, ex.wsHandleSpotAPIData(t.Context(), nil, []byte(frame)), "wsHandleSpotAPIData should error for %s", frame)
	}

	for _, tc := range []struct {
		field, value string
		err          error
	}{
		{"X", "UNKNOWN", errUnknownOrderStatus},
		{"o", "UNKNOWN", order.ErrUnrecognisedOrderType},
		{"S", "UNKNOWN", order.ErrSideIsInvalid},
		{"f", "UNKNOWN", order.ErrInvalidTimeInForce},
	} {
		frame := map[string]any{"e": "executionReport", "s": "BTCUSDT", "i": 5340845958, "X": "NEW", "o": "LIMIT", "S": "BUY", "f": "GTC"}
		frame[tc.field] = tc.value
		raw, err := json.Marshal(map[string]any{"subscriptionId": 0, "event": frame})
		require.NoError(t, err, "Marshal must not error")
		require.NoErrorf(t, ex.wsHandleSpotAPIData(t.Context(), nil, raw), "wsHandleSpotAPIData must relay an unclassifiable %s", tc.field)
		require.NotEmpty(t, ex.Websocket.DataHandler.C, "relay must hold the classification error")
		got, ok := (<-ex.Websocket.DataHandler.C).Data.(order.ClassificationError)
		require.True(t, ok, "relay must hold a classification error")
		assert.Equal(t, "5340845958", got.OrderID, "the classification error should name the order")
		assert.ErrorIsf(t, got.Err, tc.err, "the classification error should explain the %s value", tc.field)
	}

	// A list status is relayed as it arrives even when its symbol is unknown, and an unknown list order status is
	// reported to the order manager instead of the order list
	err := ex.wsHandleSpotAPIData(t.Context(), nil, []byte(`{"subscriptionId":0,"event":{"e":"listStatus","s":"BTCXYZ","g":3,"L":"EXECUTING"}}`))
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "wsHandleSpotAPIData should report a list status of an unknown symbol")
	assert.Equal(t, []any{&WsListStatus{EventType: "listStatus", Symbol: "BTCXYZ", OrderListID: 3, ListOrderStatus: "EXECUTING"}}, relayedPayloads(ex), "the list status of an unknown symbol should be relayed")
	require.NoError(t, ex.wsHandleSpotAPIData(t.Context(), nil, []byte(`{"subscriptionId":0,"event":{"e":"listStatus","s":"BTCUSDT","g":3,"L":"UNKNOWN"}}`)), "wsHandleSpotAPIData must relay an unclassifiable list status")
	relayed := relayedPayloads(ex)
	require.Len(t, relayed, 2, "relay must hold the list status and its classification error")
	assert.Equal(t, &WsListStatus{EventType: "listStatus", Symbol: "BTCUSDT", OrderListID: 3, ListOrderStatus: "UNKNOWN"}, relayed[0], "the unclassifiable list status should be relayed")
	got, ok := relayed[1].(order.ClassificationError)
	require.True(t, ok, "relay must hold a classification error")
	assert.Equal(t, "3", got.OrderID, "the classification error should name the order list")
	assert.ErrorIs(t, got.Err, errUnknownOrderStatus, "the classification error should explain the list order status")
}

// TestListStatusUpdatesTrackedOrderList feeds the events of an OCO order list whose limit maker leg fills and whose stop
// leg expires, and applies what is relayed to the order as GCT tracks it after SubmitOrder, by the list's ID
func TestListStatusUpdatesTrackedOrderList(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	ex.userData.expect("spot", asset.Spot)
	ex.userData.recordResponse("spot", []byte(`{"id":"spot","status":200,"result":{"subscriptionId":0}}`))
	pair := currency.NewPairWithDelimiter("BTC", "USDT", "-")
	tracked := &order.Detail{Exchange: ex.Name, OrderID: "1930", ClientOrderID: "gct-oco", Type: order.OCO, Status: order.Active, AssetType: asset.Spot, Pair: pair}
	for _, frame := range []string{
		`{"subscriptionId":0,"event":{"e":"executionReport","E":1791284900000,"s":"BTCUSDT","c":"leg-stop","S":"SELL","o":"STOP_LOSS_LIMIT","f":"GTC","q":"0.00100000","p":"69000.00000000","P":"70000.00000000","F":"0.00000000","g":1930,"C":"","x":"EXPIRED","X":"EXPIRED","r":"NONE","i":5001,"l":"0.00000000","z":"0.00000000","L":"0.00000000","n":"0","N":null,"T":1791284900000,"t":-1,"I":1,"w":false,"m":false,"M":false,"O":1791284813000,"Z":"0.00000000","Y":"0.00000000","Q":"0.00000000","V":"NONE","eR":"OCO_TRIGGER"}}`,
		`{"subscriptionId":0,"event":{"e":"executionReport","E":1791284900000,"s":"BTCUSDT","c":"leg-limit","S":"SELL","o":"LIMIT_MAKER","f":"GTC","q":"0.00100000","p":"90000.00000000","P":"0.00000000","F":"0.00000000","g":1930,"C":"","x":"TRADE","X":"FILLED","r":"NONE","i":5002,"l":"0.00100000","z":"0.00100000","L":"90000.00000000","n":"0.09","N":"USDT","T":1791284900000,"t":77,"I":2,"w":false,"m":true,"M":true,"O":1791284813000,"Z":"90.00000000","Y":"90.00000000","Q":"0.00000000","V":"NONE"}}`,
		`{"subscriptionId":0,"event":{"e":"listStatus","E":1791284900000,"s":"BTCUSDT","g":1930,"c":"OCO","l":"ALL_DONE","L":"ALL_DONE","r":"NONE","C":"gct-oco","T":1791284900001,"O":[{"s":"BTCUSDT","i":5001,"c":"leg-stop"},{"s":"BTCUSDT","i":5002,"c":"leg-limit"}]}}`,
	} {
		require.NoError(t, ex.wsHandleSpotAPIData(t.Context(), nil, []byte(frame)), "wsHandleSpotAPIData must not error")
	}
	var updates int
	for _, payload := range relayedPayloads(ex) {
		// The engine applies an order update to the tracked order with the same ID
		if d, ok := payload.(*order.Detail); ok && d.OrderID == tracked.OrderID {
			require.NoError(t, tracked.UpdateOrderFromDetail(d), "UpdateOrderFromDetail must not error")
			updates++
		}
	}
	require.Equal(t, 1, updates, "the order list's event must update the tracked order once")
	exp := &order.Detail{
		Exchange:      ex.Name,
		OrderID:       "1930",
		ClientOrderID: "gct-oco",
		Type:          order.OCO,
		Status:        order.Closed,
		AssetType:     asset.Spot,
		LastUpdated:   time.UnixMilli(1791284900001),
		Pair:          pair,
	}
	assert.Equal(t, exp, tracked, "the tracked order list should close with its list status")
}

func TestWsHandleSpotAPIDataServerShutdown(t *testing.T) {
	t.Parallel()
	ex := newWsAPITestExchange(t, false)
	conn, err := ex.Websocket.GetConnection(spotWebsocketAPI)
	require.NoError(t, err, "GetConnection must not error")
	require.NoError(t, ex.wsHandleSpotAPIData(t.Context(), conn, []byte(`{"event":{"e":"serverShutdown","E":1770123456789}}`)), "wsHandleSpotAPIData must not error")
	assert.Eventually(t, func() bool { return !ex.Websocket.IsConnected() }, 5*time.Second, 10*time.Millisecond, "serverShutdown should shut the connections down for the monitor to reconnect")
}

func TestWsHandleSpotAPIDataUnmatchedResponse(t *testing.T) {
	t.Parallel()
	ex := newWsAPITestExchange(t, false)
	conn, err := ex.Websocket.GetConnection(spotWebsocketAPI)
	require.NoError(t, err, "GetConnection must not error")
	err = ex.wsHandleSpotAPIData(t.Context(), conn, []byte(`{"id":"unrequested","status":200,"result":{}}`))
	assert.ErrorIs(t, err, websocket.ErrSignatureNotMatched, "wsHandleSpotAPIData should error for a response nothing requested")
}

func TestEventStreamTerminatedResubscribes(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("the subscription IDs Binance assigns cannot be predicted")
	}
	ex := newWsAPITestExchange(t, true)
	conn, err := ex.Websocket.GetConnection(spotWebsocketAPI)
	require.NoError(t, err, "GetConnection must not error")
	ex.userData.expect("spot", asset.Spot)
	ex.userData.expect("margin", asset.Margin)
	ex.userData.recordResponse("spot", []byte(`{"id":"spot","status":200,"result":{"subscriptionId":5}}`))
	ex.userData.recordResponse("margin", []byte(`{"id":"margin","status":200,"result":{"subscriptionId":6}}`))
	for _, id := range []int{5, 6} {
		frame := fmt.Appendf(nil, `{"subscriptionId":%d,"event":{"e":"eventStreamTerminated","E":1728973001334}}`, id)
		require.NoErrorf(t, ex.wsHandleSpotAPIData(t.Context(), conn, frame), "wsHandleSpotAPIData must not error ending subscription %d", id)
	}
	assert.Eventually(t, func() bool {
		return maps.Equal(map[uint64]asset.Item{1: asset.Spot, 2: asset.Margin}, userDataAssets(ex))
	}, 5*time.Second, 10*time.Millisecond, "ended subscriptions should be replaced by new ones for the same accounts")
}
