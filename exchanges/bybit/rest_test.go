package bybit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Please supply your own keys here to do authenticated endpoint testing
const canManipulateRealOrders = false

var apiCredentials = &accounts.Credentials{
	Key:    "",
	Secret: "",
}

var (
	e *Exchange

	spotTradablePair, usdcMarginedTradablePair, usdtMarginedTradablePair, inverseTradablePair, optionsTradablePair currency.Pair
)

func TestStringToOrderStatus(t *testing.T) {
	t.Parallel()
	input := []struct {
		OrderStatus string
		Expectation order.Status
	}{
		{
			OrderStatus: "",
			Expectation: order.UnknownStatus,
		},
		{
			OrderStatus: "UNKNOWN",
			Expectation: order.UnknownStatus,
		},
		{
			OrderStatus: "Cancelled",
			Expectation: order.Cancelled,
		},
		{
			OrderStatus: "ACTIVE",
			Expectation: order.Active,
		},
		{
			OrderStatus: "NEW",
			Expectation: order.New,
		},
		{
			OrderStatus: "FILLED",
			Expectation: order.Filled,
		},
		{
			OrderStatus: "UNTRIGGERED",
			Expectation: order.Pending,
		},
		// Bybit's documented orderStatus values, spelt as the API sends them
		{
			OrderStatus: "New",
			Expectation: order.New,
		},
		{
			OrderStatus: "PartiallyFilled",
			Expectation: order.PartiallyFilled,
		},
		{
			OrderStatus: "Untriggered",
			Expectation: order.Pending,
		},
		{
			OrderStatus: "Rejected",
			Expectation: order.Rejected,
		},
		{
			OrderStatus: "PartiallyFilledCanceled",
			Expectation: order.PartiallyFilledCancelled,
		},
		{
			OrderStatus: "Filled",
			Expectation: order.Filled,
		},
		{
			OrderStatus: "Triggered",
			Expectation: order.Open,
		},
		{
			OrderStatus: "Deactivated",
			Expectation: order.Cancelled,
		},
		// Other spellings the function accepts
		{
			OrderStatus: "Created",
			Expectation: order.Open,
		},
		{
			OrderStatus: "PARTIALLY_FILLED",
			Expectation: order.PartiallyFilled,
		},
		{
			OrderStatus: "PENDING_CANCEL",
			Expectation: order.PendingCancel,
		},
		{
			OrderStatus: "Canceled",
			Expectation: order.Cancelled,
		},
	}
	for x := range input {
		assert.Equalf(t, input[x].Expectation, StringToOrderStatus(input[x].OrderStatus), "StringToOrderStatus should return %s for %q", input[x].Expectation, input[x].OrderStatus)
	}
}

// newTransportTestExchange returns an exchange whose REST requests reach handler, with placeholder credentials
func newTransportTestExchange(t *testing.T, handler http.HandlerFunc) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	server := httptest.NewTestServer(t, handler)
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	return ex
}

// newPagingTestExchange returns an exchange whose server answers the first request for each cursor in pages with that
// page as the result, and refuses any other request, so a listing that pages on past its last page fails rather than
// asking forever
func newPagingTestExchange(t *testing.T, pages map[string]string) *Exchange {
	t.Helper()
	var mu sync.Mutex
	asked := make(map[string]bool)
	ex := newTransportTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		cursor := r.URL.Query().Get("cursor")
		mu.Lock()
		again := asked[cursor]
		asked[cursor] = true
		mu.Unlock()
		reply := `{"retCode":10001,"retMsg":"params error","result":{},"retExtInfo":{},"time":1760000000000}`
		if page, ok := pages[cursor]; ok && !again {
			reply = `{"retCode":0,"retMsg":"success","result":` + page + `,"retExtInfo":{},"time":1760000000000}`
		}
		_, err := fmt.Fprint(w, reply)
		assert.NoError(t, err, "writing the reply should not error")
	})
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	return ex
}

func TestIsNil(t *testing.T) {
	t.Parallel()
	coin := "BTC"
	for _, tc := range []struct {
		v   any
		exp bool
	}{
		{v: nil, exp: true},
		{v: (*string)(nil), exp: true},
		{v: map[string]any(nil), exp: true},
		{v: []string(nil), exp: true},
		{v: &coin, exp: false},
		{v: map[string]any{}, exp: false},
		{v: []string{}, exp: false},
		{v: struct{}{}, exp: false},
		{v: "", exp: false},
	} {
		assert.Equalf(t, tc.exp, isNil(tc.v), "isNil should report whether %#v is nil", tc.v)
	}
}

func TestSendHTTPRequest(t *testing.T) {
	t.Parallel()
	var body string
	ex := newTransportTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "public requests should use GET")
		assert.Equal(t, "/v5/market/time", r.URL.Path, "the path should reach the server unchanged")
		assert.Equal(t, "category=spot", r.URL.RawQuery, "the query string should reach the server unchanged")
		assert.Empty(t, r.Header.Get("X-BAPI-API-KEY"), "public requests should not carry the API key")
		_, err := fmt.Fprint(w, body)
		assert.NoError(t, err, "writing the response should not error")
	})

	body = `{"retCode":0,"retMsg":"OK","result":{"timeSecond":"1688639403"},"retExtInfo":{},"time":1688639403423}`
	var result struct {
		TimeSecond types.Time `json:"timeSecond"`
	}
	require.NoError(t, ex.SendHTTPRequest(t.Context(), exchange.RestSpot, "/v5/market/time?category=spot", defaultEPL, &result), "SendHTTPRequest must not error")
	assert.Equal(t, time.Unix(1688639403, 0), result.TimeSecond.Time(), "SendHTTPRequest should decode the envelope's result")

	require.NoError(t, ex.SendHTTPRequest(t.Context(), exchange.RestSpot, "/v5/market/time?category=spot", defaultEPL, nil), "SendHTTPRequest must not error with a nil result")

	body = `{"retCode":0,"retMsg":"OK","result":null,"retExtInfo":{},"time":1688639403423}`
	err := ex.SendHTTPRequest(t.Context(), exchange.RestSpot, "/v5/market/time?category=spot", defaultEPL, &result)
	assert.ErrorIs(t, err, common.ErrNoResponse, "SendHTTPRequest should reject a null result")

	body = `{"retCode":170121,"retMsg":"Invalid symbol.","result":{},"retExtInfo":{},"time":1688639403423}`
	err = ex.SendHTTPRequest(t.Context(), exchange.RestSpot, "/v5/market/time?category=spot", defaultEPL, &result)
	assert.ErrorIs(t, err, errAPIResponse, "SendHTTPRequest should return an API error")
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "SendHTTPRequest should map Invalid symbol to ErrPairNotFound")
	assert.NotErrorIs(t, err, request.ErrAuthRequestFailed, "a public request's API error should not be an authentication failure")

	err = ex.SendHTTPRequest(t.Context(), exchange.RestSwap, "/v5/market/time", defaultEPL, &result)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "SendHTTPRequest should error for an unset URL key")
}

var errBodyNotEncodable = errors.New("body cannot be encoded")

// unencodableBody is a request body whose JSON encoding fails
type unencodableBody struct{}

// MarshalJSON returns errBodyNotEncodable
func (unencodableBody) MarshalJSON() ([]byte, error) {
	return nil, errBodyNotEncodable
}

func TestSendAuthHTTPRequest(t *testing.T) {
	t.Parallel()
	var body string
	var gotMethod, gotQuery, gotBody string
	var gotHeader http.Header
	ex := newTransportTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotQuery, gotHeader = r.Method, r.URL.RawQuery, r.Header
		raw, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "reading the request body should not error")
		gotBody = string(raw)
		_, err = fmt.Fprint(w, body)
		assert.NoError(t, err, "writing the response should not error")
	})

	err := ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, http.MethodGet, "/v5/order/realtime", nil, struct{}{}, nil, defaultEPL)
	assert.ErrorIs(t, err, errBodyWithGet, "SendAuthHTTPRequest should reject a GET with a body")
	err = ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, http.MethodPost, "/v5/order/create", url.Values{"category": {"spot"}}, map[string]string{"category": "spot"}, nil, defaultEPL)
	assert.ErrorIs(t, err, errQueryWithBody, "SendAuthHTTPRequest should reject a POST with both query parameters and a body")

	sign := func(timestamp, payload string) string {
		mac := hmac.New(sha256.New, []byte("secret"))
		_, err := mac.Write([]byte(timestamp + "key" + defaultRecvWindow + payload))
		require.NoError(t, err, "hashing must not error")
		return hex.EncodeToString(mac.Sum(nil))
	}

	body = `{"retCode":0,"retMsg":"OK","result":{"orderId":"1"},"retExtInfo":{},"time":1688639403423}`
	var result struct {
		OrderID string `json:"orderId"`
	}
	params := url.Values{"category": {"spot"}, "symbol": {"BTCUSDT"}}
	before := time.Now().Truncate(time.Millisecond)
	require.NoError(t, ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, http.MethodGet, "/v5/order/realtime", params, nil, &result, defaultEPL), "SendAuthHTTPRequest must not error")
	assert.Equal(t, "1", result.OrderID, "SendAuthHTTPRequest should decode the envelope's result")
	assert.Equal(t, http.MethodGet, gotMethod, "the request should keep its method")
	assert.Equal(t, "category=spot&symbol=BTCUSDT", gotQuery, "a GET request should carry its parameters in the query string")
	assert.Empty(t, gotBody, "a GET request should not carry a body")
	assert.Equal(t, "key", gotHeader.Get("X-BAPI-API-KEY"), "the request should carry the API key")
	assert.Equal(t, defaultRecvWindow, gotHeader.Get("X-BAPI-RECV-WINDOW"), "the request should carry the receive window")
	assert.Equal(t, sign(gotHeader.Get("X-BAPI-TIMESTAMP"), gotQuery), gotHeader.Get("X-BAPI-SIGN"), "a GET request should sign its query string")
	// Bybit refuses a request whose timestamp, in milliseconds, falls outside its receive window
	sentAt, err := strconv.ParseInt(gotHeader.Get("X-BAPI-TIMESTAMP"), 10, 64)
	require.NoError(t, err, "the timestamp header must be an integer")
	assert.WithinRange(t, time.UnixMilli(sentAt), before, time.Now(), "the request should carry the current time in milliseconds")

	before = time.Now().Truncate(time.Millisecond)
	require.NoError(t, ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, http.MethodPost, "/v5/order/create", nil, map[string]string{"category": "spot"}, &result, defaultEPL), "SendAuthHTTPRequest must not error")
	sentAt, err = strconv.ParseInt(gotHeader.Get("X-BAPI-TIMESTAMP"), 10, 64)
	require.NoError(t, err, "the timestamp header must be an integer")
	assert.WithinRange(t, time.UnixMilli(sentAt), before, time.Now(), "a POST request should carry the current time in milliseconds")
	assert.JSONEq(t, `{"category":"spot"}`, gotBody, "a POST request should send its body as JSON")
	assert.Empty(t, gotQuery, "a POST request should not carry a query string")
	assert.Equal(t, "application/json", gotHeader.Get("Content-Type"), "a POST request should declare a JSON body")
	assert.Equal(t, sign(gotHeader.Get("X-BAPI-TIMESTAMP"), gotBody), gotHeader.Get("X-BAPI-SIGN"), "a POST request should sign its body")

	body = `{"retCode":0,"retMsg":"OK","retExtInfo":{},"time":1688639403423}`
	require.NoError(t, ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, http.MethodPost, "/v5/account/mmp-reset", nil, nil, nil, defaultEPL), "SendAuthHTTPRequest must not error without a result")
	assert.Equal(t, "{}", gotBody, "a POST request without a body should send an empty object")
	require.NoError(t, ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, http.MethodPost, "/v5/account/mmp-reset", url.Values{}, nil, nil, defaultEPL), "SendAuthHTTPRequest must not error with empty query parameters")
	assert.Equal(t, "{}", gotBody, "a POST request with empty query parameters should send an empty object")
	assert.Empty(t, gotQuery, "a POST request with empty query parameters should not carry a query string")
	// A nil map, pointer or slice holds no body, as a nil interface does
	for _, typedNil := range []any{map[string]any(nil), (*struct{ Coin string })(nil), []any(nil)} {
		require.NoErrorf(t, ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, http.MethodPost, "/v5/account/mmp-reset", nil, typedNil, nil, defaultEPL), "SendAuthHTTPRequest must not error with a %T body", typedNil)
		assert.Equalf(t, "{}", gotBody, "a POST request with a %T body should send an empty object", typedNil)
		assert.NoErrorf(t, ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, http.MethodGet, "/v5/order/realtime", nil, typedNil, nil, defaultEPL), "SendAuthHTTPRequest should send a GET with a %T body", typedNil)
	}

	require.NoError(t, ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, http.MethodPost, "/v5/card/reward/point/cashback/detail", url.Values{"bizTxnId": {"TXN20230101001"}}, nil, nil, defaultEPL), "SendAuthHTTPRequest must not error")
	assert.Equal(t, http.MethodPost, gotMethod, "a POST request with query parameters should keep its method")
	assert.Equal(t, "bizTxnId=TXN20230101001", gotQuery, "a POST request should carry its query parameters in the query string")
	assert.Empty(t, gotBody, "a POST request with query parameters should not carry a body")
	assert.Empty(t, gotHeader.Get("Content-Type"), "a POST request without a body should not declare one")
	assert.Equal(t, sign(gotHeader.Get("X-BAPI-TIMESTAMP"), ""), gotHeader.Get("X-BAPI-SIGN"), "a POST request with query parameters should sign its empty body")

	body = `{"retCode":0,"retMsg":"OK","result":{"list":[{"orderId":"1"}]},"retExtInfo":{"list":[{"code":0,"msg":"OK"}]},"time":1688639403423}`
	var envelope RestResponse
	require.NoError(t, ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, http.MethodPost, "/v5/order/create-batch", nil, nil, &envelope, defaultEPL), "SendAuthHTTPRequest must not error")
	exp := RestResponse{
		ReturnMessage:      "OK",
		Result:             json.RawMessage(`{"list":[{"orderId":"1"}]}`),
		ReturnExtendedInfo: json.RawMessage(`{"list":[{"code":0,"msg":"OK"}]}`),
		Time:               types.Time(time.UnixMilli(1688639403423)),
	}
	assert.Equal(t, exp, envelope, "a *RestResponse result should receive the whole envelope")
	gotMethod = ""
	err = ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, http.MethodPost, "/v5/order/create-batch", nil, nil, (*RestResponse)(nil), defaultEPL)
	assert.ErrorIs(t, err, common.ErrNilPointer, "SendAuthHTTPRequest should refuse a nil *RestResponse result")
	assert.Empty(t, gotMethod, "SendAuthHTTPRequest should send nothing for a result it could not fill")

	body = `{"retCode":110001,"retMsg":"Order does not exist","result":{},"retExtInfo":{},"time":1688639403423}`
	err = ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, http.MethodPost, "/v5/order/cancel", nil, nil, &result, defaultEPL)
	assert.ErrorIs(t, err, errAPIResponse, "SendAuthHTTPRequest should return an API error")
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "SendAuthHTTPRequest should map Order does not exist to ErrOrderNotFound")
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "an authenticated request's API error should be an authentication failure")
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "the error must be an APIError")
	assert.Equal(t, int64(110001), apiErr.Code, "the APIError should keep the code")
	assert.Equal(t, "Order does not exist", apiErr.Message, "the APIError should keep the message")

	gotMethod = ""
	err = ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, http.MethodPost, "/v5/order/create", nil, unencodableBody{}, &result, defaultEPL)
	assert.ErrorIs(t, err, errBodyNotEncodable, "SendAuthHTTPRequest should return the body's encoding error")
	err = ex.SendAuthHTTPRequest(t.Context(), exchange.RestSwap, http.MethodGet, "/v5/order/realtime", nil, nil, &result, defaultEPL)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "SendAuthHTTPRequest should error for an unset URL key")
	assert.Empty(t, gotMethod, "SendAuthHTTPRequest should send nothing when it cannot encode the body or find the URL")

	ex.SetCredentials(&accounts.Credentials{})
	err = ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, http.MethodGet, "/v5/order/realtime", nil, nil, &result, defaultEPL)
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "SendAuthHTTPRequest should require credentials")
}

func TestSendAuthHTTPRequestSignsEachAttempt(t *testing.T) {
	t.Parallel()
	// The requester builds each attempt after its rate limit wait and a retry's backoff, and Bybit refuses a timestamp
	// older than its receive window, so each attempt must be signed when it is sent
	var mu sync.Mutex
	var timestamps, signatures []string
	ex := newTransportTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		timestamps = append(timestamps, r.Header.Get("X-BAPI-TIMESTAMP"))
		signatures = append(signatures, r.Header.Get("X-BAPI-SIGN"))
		first := len(timestamps) == 1
		mu.Unlock()
		if first {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, err := fmt.Fprint(w, `{"retCode":0,"retMsg":"OK","result":{},"retExtInfo":{},"time":1688639403423}`)
		assert.NoError(t, err, "writing the response should not error")
	})
	require.NoError(t, ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, http.MethodGet, "/v5/account/info", url.Values{"category": {"spot"}}, nil, nil, defaultEPL), "SendAuthHTTPRequest must not error once its retry succeeds")
	mu.Lock()
	defer mu.Unlock()
	require.Len(t, timestamps, 2, "SendAuthHTTPRequest must send the request again after a 429")
	first, err := strconv.ParseInt(timestamps[0], 10, 64)
	require.NoError(t, err, "the first timestamp must be an integer")
	retried, err := strconv.ParseInt(timestamps[1], 10, 64)
	require.NoError(t, err, "the retried timestamp must be an integer")
	assert.Greater(t, retried, first, "a retried request should carry the time it is sent at")
	assert.Equal(t, getSign(timestamps[1]+"key"+defaultRecvWindow+"category=spot", "secret"), signatures[1], "a retried request should be signed with its own timestamp")
}

var errRequestItem = errors.New("request item cannot be built")

func TestSendRequest(t *testing.T) {
	t.Parallel()
	// serve returns an exchange whose server answers every request with status and body, and a builder of requests
	// to that server
	serve := func(t *testing.T, status int, body string) (*Exchange, func() (*request.Item, error)) {
		t.Helper()
		ex := newTransportTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			_, err := fmt.Fprint(w, body)
			assert.NoError(t, err, "writing the response should not error")
		})
		endpoint, err := ex.API.Endpoints.GetURL(exchange.RestSpot)
		require.NoError(t, err, "GetURL must not error")
		return ex, func() (*request.Item, error) {
			return &request.Item{Method: http.MethodGet, Path: endpoint + "/v5/market/time"}, nil
		}
	}

	t.Run("result", func(t *testing.T) {
		t.Parallel()
		ex, newItem := serve(t, http.StatusOK, `{"retCode":0,"retMsg":"OK","result":{"timeSecond":"1688639403","timeNano":"1688639403423213947"},"retExtInfo":{},"time":1688639403423}`)
		var result *ServerTimeResponse
		require.NoError(t, ex.sendRequest(t.Context(), defaultEPL, request.UnauthenticatedRequest, &result, newItem), "sendRequest must not error")
		exp := &ServerTimeResponse{
			TimeSecond: types.Time(time.Unix(1688639403, 0)),
			TimeNano:   types.Time(time.Unix(0, 1688639403423213947)),
		}
		assert.Equal(t, exp, result, "sendRequest should decode the envelope's result")
	})

	t.Run("no nonce lock", func(t *testing.T) {
		t.Parallel()
		for _, auth := range []request.AuthType{request.UnauthenticatedRequest, request.AuthenticatedRequest} {
			ex, newItem := serve(t, http.StatusOK, `{"retCode":0,"retMsg":"OK","result":{"timeSecond":"1688639403","timeNano":"1688639403423213947"},"retExtInfo":{},"time":1688639403423}`)
			var sent *request.Item
			var result *ServerTimeResponse
			require.NoErrorf(t, ex.sendRequest(t.Context(), defaultEPL, auth, &result, func() (*request.Item, error) {
				item, err := newItem()
				sent = item
				return item, err
			}), "sendRequest must not error for auth type %d", auth)
			require.NotNilf(t, sent, "sendRequest must build its request for auth type %d", auth)
			assert.Truef(t, sent.NonceEnabled, "sendRequest should not queue a request of auth type %d for the requester's nonce lock, which would move it past its booked send time", auth)
		}
	})

	t.Run("nil result", func(t *testing.T) {
		t.Parallel()
		var sent atomic.Bool
		ex := newTransportTestExchange(t, func(http.ResponseWriter, *http.Request) { sent.Store(true) })
		endpoint, err := ex.API.Endpoints.GetURL(exchange.RestSpot)
		require.NoError(t, err, "GetURL must not error")
		for _, result := range []any{(*ServerTimeResponse)(nil), (**ServerTimeResponse)(nil), (*RestResponse)(nil), map[string]any(nil), []any(nil)} {
			err := ex.sendRequest(t.Context(), defaultEPL, request.UnauthenticatedRequest, result, func() (*request.Item, error) {
				return &request.Item{Method: http.MethodGet, Path: endpoint + "/v5/market/time"}, nil
			})
			assert.ErrorIsf(t, err, common.ErrNilPointer, "sendRequest should refuse a nil %T result", result)
		}
		assert.False(t, sent.Load(), "sendRequest should send nothing for a result it could not decode into")
	})

	t.Run("whole envelope", func(t *testing.T) {
		t.Parallel()
		ex, newItem := serve(t, http.StatusOK, `{"retCode":0,"retMsg":"OK","result":{"list":[{"orderId":"1"}]},"retExtInfo":{"list":[{"code":0,"msg":"OK"}]},"time":1688639403423}`)
		var result RestResponse
		require.NoError(t, ex.sendRequest(t.Context(), defaultEPL, request.AuthenticatedRequest, &result, newItem), "sendRequest must not error")
		exp := RestResponse{
			ReturnMessage:      "OK",
			Result:             json.RawMessage(`{"list":[{"orderId":"1"}]}`),
			ReturnExtendedInfo: json.RawMessage(`{"list":[{"code":0,"msg":"OK"}]}`),
			Time:               types.Time(time.UnixMilli(1688639403423)),
		}
		assert.Equal(t, exp, result, "sendRequest should give a *RestResponse result the whole envelope")
	})

	t.Run("no result wanted", func(t *testing.T) {
		t.Parallel()
		ex, newItem := serve(t, http.StatusOK, `{"retCode":0,"retMsg":"OK","result":{},"retExtInfo":{},"time":1688639403423}`)
		assert.NoError(t, ex.sendRequest(t.Context(), defaultEPL, request.AuthenticatedRequest, nil, newItem), "sendRequest should not error without a result")
	})

	for _, tc := range []struct {
		name, body string
	}{
		{name: "null result", body: `{"retCode":0,"retMsg":"OK","result":null,"retExtInfo":{},"time":1688639403423}`},
		{name: "missing result", body: `{"retCode":0,"retMsg":"OK","retExtInfo":{},"time":1688639403423}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, newItem := serve(t, http.StatusOK, tc.body)
			var result *ServerTimeResponse
			err := ex.sendRequest(t.Context(), defaultEPL, request.UnauthenticatedRequest, &result, newItem)
			assert.ErrorIs(t, err, common.ErrNoResponse, "sendRequest should reject a response without a result")
		})
	}

	t.Run("result of the wrong type", func(t *testing.T) {
		t.Parallel()
		ex, newItem := serve(t, http.StatusOK, `{"retCode":0,"retMsg":"OK","result":"maintenance","retExtInfo":{},"time":1688639403423}`)
		var result *ServerTimeResponse
		err := ex.sendRequest(t.Context(), defaultEPL, request.UnauthenticatedRequest, &result, newItem)
		// The decoding error's type depends on the JSON backend, standard library or Sonic, so only its presence is checked
		assert.Error(t, err, "sendRequest should return the result's decoding error")
	})

	t.Run("public API error", func(t *testing.T) {
		t.Parallel()
		ex, newItem := serve(t, http.StatusOK, `{"retCode":10001,"retMsg":"params error","result":{},"retExtInfo":{},"time":1688639403423}`)
		err := ex.sendRequest(t.Context(), defaultEPL, request.UnauthenticatedRequest, nil, newItem)
		assert.ErrorIs(t, err, errAPIResponse, "sendRequest should return the API error")
		assert.NotErrorIs(t, err, request.ErrAuthRequestFailed, "a public request's API error should not be an authentication failure")
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr, "the error must be an APIError")
		assert.Equal(t, &APIError{Code: 10001, Message: "params error", ExtendedInfo: json.RawMessage(`{}`), Result: json.RawMessage(`{}`)}, apiErr, "the APIError should keep the code, message, details and result")
	})

	t.Run("signed API error", func(t *testing.T) {
		t.Parallel()
		ex, newItem := serve(t, http.StatusOK, `{"retCode":10004,"retMsg":"error sign!","result":{},"retExtInfo":{},"time":1688639403423}`)
		err := ex.sendRequest(t.Context(), defaultEPL, request.AuthenticatedRequest, nil, newItem)
		assert.ErrorIs(t, err, errAPIResponse, "sendRequest should return the API error")
		assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "a signed request's API error should be an authentication failure")
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr, "the error must be an APIError")
		assert.Equal(t, int64(10004), apiErr.Code, "the APIError should keep the code")
	})

	t.Run("API error with an error status", func(t *testing.T) {
		t.Parallel()
		ex, newItem := serve(t, http.StatusUnauthorized, `{"retCode":10003,"retMsg":"API key is invalid.","result":{},"retExtInfo":{},"time":1688639403423}`)
		err := ex.sendRequest(t.Context(), defaultEPL, request.AuthenticatedRequest, nil, newItem)
		assert.ErrorIs(t, err, errAPIResponse, "sendRequest should return the API error")
		assert.ErrorIs(t, err, request.ErrBadStatus, "sendRequest should keep the HTTP status error")
		assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "a signed request's error should be an authentication failure")
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr, "the error must be an APIError")
		assert.Equal(t, int64(10003), apiErr.Code, "the APIError should keep the code")
	})

	t.Run("error status without an envelope", func(t *testing.T) {
		t.Parallel()
		ex, newItem := serve(t, http.StatusForbidden, "403 Forbidden")
		err := ex.sendRequest(t.Context(), defaultEPL, request.UnauthenticatedRequest, nil, newItem)
		assert.ErrorIs(t, err, request.ErrBadStatus, "sendRequest should return the HTTP status error")
		assert.NotErrorIs(t, err, errAPIResponse, "sendRequest should not report an API error the response does not carry")
	})

	t.Run("request cannot be built", func(t *testing.T) {
		t.Parallel()
		ex := newTransportTestExchange(t, func(http.ResponseWriter, *http.Request) {
			assert.Fail(t, "sendRequest should not send a request it cannot build")
		})
		err := ex.sendRequest(t.Context(), defaultEPL, request.UnauthenticatedRequest, nil, func() (*request.Item, error) {
			return nil, errRequestItem
		})
		assert.ErrorIs(t, err, errRequestItem, "sendRequest should return the request builder's error")
	})
}

func TestRestResponseAPIError(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		resp RestResponse
		exp  error
	}{
		{name: "success", resp: RestResponse{ReturnMessage: "OK"}},
		{name: "camel case", resp: RestResponse{ReturnCode: 10001, ReturnMessage: "params error", Result: json.RawMessage(`{}`), ReturnExtendedInfo: json.RawMessage(`{}`)}, exp: &APIError{Code: 10001, Message: "params error", ExtendedInfo: json.RawMessage(`{}`), Result: json.RawMessage(`{}`)}},
		{name: "snake case", resp: RestResponse{SnakeReturnCode: 912100005, SnakeReturnMessage: "item not exist"}, exp: &APIError{Code: 912100005, Message: "item not exist"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, tc.resp.apiError(), "apiError should return the envelope's error")
		})
	}
}

func TestAPIError(t *testing.T) {
	t.Parallel()
	err := &APIError{Code: 110001, Message: "Order does not exist", ExtendedInfo: json.RawMessage(`{"list":[{"code":110001,"msg":"Order does not exist"}]}`)}
	assert.Equal(t, `bybit API error: order not found: code 110001 msg Order does not exist retExtInfo {"list":[{"code":110001,"msg":"Order does not exist"}]}`, err.Error(), "Error should include the mapped error, code, message and details")
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "a mapped code should unwrap to its package error")
	assert.ErrorIs(t, err, errAPIResponse, "every APIError should unwrap to errAPIResponse")

	err = &APIError{Code: 10001, Message: "params error", ExtendedInfo: json.RawMessage(`{}`)}
	assert.Equal(t, "bybit API error: code 10001 msg params error", err.Error(), "Error should leave out empty details")
	assert.Equal(t, []error{errAPIResponse}, err.Unwrap(), "an unmapped code should unwrap to errAPIResponse only")
	for _, empty := range []string{`""`, "null", ""} {
		err = &APIError{Code: 10001, Message: "params error", ExtendedInfo: json.RawMessage(empty)}
		assert.Equalf(t, "bybit API error: code 10001 msg params error", err.Error(), "Error should leave out details sent as %q", empty)
	}
}

func TestDecodeErrorResult(t *testing.T) {
	t.Parallel()
	failure := &APIError{
		Code: 3400045, Message: "Set margin mode failed",
		Result: json.RawMessage(`{"reasons":[{"reasonCode":"3400000","reasonMsg":"Equity needs to be equal to or greater than 1000 USDC"}]}`),
	}
	for _, tc := range []struct {
		name string
		err  error
		exp  *SetMarginModeResponse
	}{
		{name: "no error"},
		{name: "not an API error", err: errRequestItem},
		{name: "API error without a result", err: &APIError{Code: 10001, Message: "params error"}},
		{name: "API error with a null result", err: &APIError{Code: 10001, Message: "params error", Result: json.RawMessage(`null`)}},
		{
			name: "wrapped API error with a result",
			err:  fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, failure),
			exp:  &SetMarginModeResponse{Reasons: []SetMarginModeReason{{ReasonCode: "3400000", ReasonMessage: "Equity needs to be equal to or greater than 1000 USDC"}}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var result *SetMarginModeResponse
			assert.Equal(t, tc.err, decodeErrorResult(tc.err, &result), "decodeErrorResult should return the error unchanged")
			assert.Equal(t, tc.exp, result, "decodeErrorResult should decode the result the API error carries")
		})
	}

	t.Run("undecodable result", func(t *testing.T) {
		t.Parallel()
		apiErr := &APIError{Code: 3400045, Message: "Set margin mode failed", Result: json.RawMessage(`{"reasons":"none"}`)}
		var result *SetMarginModeResponse
		err := decodeErrorResult(apiErr, &result)
		assert.ErrorIs(t, err, apiErr, "decodeErrorResult should keep the API error")
		// The decoding error's type depends on the JSON backend, standard library or Sonic, so only its context is checked
		assert.ErrorContains(t, err, "decoding its result", "decodeErrorResult should add the result's decoding error")
	})
}
