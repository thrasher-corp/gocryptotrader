package bingx

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	testAPIKey    = "test-api-key"
	testAPISecret = "test-api-secret"
)

// livePositionStateCodes are the codes BingX documents for a margin adjustment the account's positions or balances
// cannot take: no position on the symbol (109420), a position not in isolated margin mode (101479), and too little
// margin or balance (101204, 101206, 101212, 101253)
var livePositionStateCodes = []int64{109420, 101479, 101204, 101206, 101212, 101253}

// requireAPIErrorCode requires err to be a BingX API error with one of codes
func requireAPIErrorCode(tb testing.TB, err error, codes ...int64) {
	tb.Helper()
	var apiErr *APIError
	require.ErrorAs(tb, err, &apiErr, "error must be a BingX API error")
	require.Containsf(tb, codes, apiErr.Code, "BingX must answer with one of codes %v", codes)
}

// newTestServerExchange returns an exchange named after the test whose REST endpoints are served by handler, with test
// credentials and no rate limiting
func newTestServerExchange(tb testing.TB, handler http.HandlerFunc) *Exchange {
	tb.Helper()
	ex := new(Exchange)
	require.NoError(tb, testexch.Setup(ex), "Setup must not error")
	ex.Name = tb.Name()
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: testAPIKey, Secret: testAPISecret})
	server := httptest.NewTestServer(tb, handler)
	require.NoError(tb, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	for _, ePath := range []exchange.URL{exchange.RestSpot, exchange.RestUSDTMargined, exchange.RestCoinMargined} {
		require.NoErrorf(tb, ex.API.Endpoints.SetRunningURL(ePath.String(), server.URL), "SetRunningURL must not error for %s", ePath)
	}
	require.NoError(tb, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	return ex
}

// orderedQuery splits a raw query into its keys and decoded values in the order they were sent
func orderedQuery(tb testing.TB, rawQuery string) (keys, values []string) {
	tb.Helper()
	for pair := range strings.SplitSeq(rawQuery, "&") {
		k, v, _ := strings.Cut(pair, "=")
		decoded, err := url.QueryUnescape(v)
		require.NoErrorf(tb, err, "QueryUnescape must not error for %q", v)
		keys = append(keys, k)
		values = append(values, decoded)
	}
	return keys, values
}

// verifyQuerySignature checks a signed query the way BingX does: the signature over every other parameter, in the
// order sent, with decoded values
func verifyQuerySignature(tb testing.TB, rawQuery string) {
	tb.Helper()
	keys, values := orderedQuery(tb, rawQuery)
	require.NotEmpty(tb, keys, "query must not be empty")
	require.Equal(tb, "signature", keys[len(keys)-1], "signature must be the last parameter")
	parts := make([]string, 0, len(keys)-1)
	for i := range keys[:len(keys)-1] {
		parts = append(parts, keys[i]+"="+values[i])
	}
	mac := hmac.New(sha256.New, []byte(testAPISecret))
	mac.Write([]byte(strings.Join(parts, "&")))
	assert.Equal(tb, hex.EncodeToString(mac.Sum(nil)), values[len(values)-1], "signature should match the parameters sent")
}

func TestGetSystemTime(t *testing.T) {
	t.Parallel()
	result, err := e.GetSystemTime(t.Context())
	require.NoError(t, err, "GetSystemTime must not error")
	require.NotNil(t, result, "GetSystemTime must return a result")
	if mockTests {
		assert.Equal(t, time.UnixMilli(1794406598194), result.ServerTime.Time(), "ServerTime should be the recorded time")
	} else {
		assert.WithinDuration(t, time.Now(), result.ServerTime.Time(), time.Minute, "ServerTime should be close to now")
	}
}

func TestSendHTTPRequest(t *testing.T) {
	t.Parallel()
	var calls int
	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, http.MethodGet, r.Method, "method should be GET")
		assert.Equal(t, "/openApi/test", r.URL.Path, "path should be the requested path")
		assert.Equal(t, "symbol=BTC-USDT", r.URL.RawQuery, "query should be sent as given")
		assert.Empty(t, r.Header.Get(keyHeader), "API key header should not be sent")
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"serverTime":1794406412817}}`)
	})
	var resp *SystemTimeResponse
	require.NoError(t, ex.SendHTTPRequest(t.Context(), exchange.RestSpot, "/openApi/test?symbol=BTC-USDT", systemTimeRate, &resp), "SendHTTPRequest must not error")
	require.NotNil(t, resp, "SendHTTPRequest must decode the data member")
	assert.Equal(t, time.UnixMilli(1794406412817), resp.ServerTime.Time(), "ServerTime should decode")
	assert.Equal(t, 1, calls, "one request should be sent")

	err := ex.SendHTTPRequest(t.Context(), exchange.RestFutures, "/openApi/test", systemTimeRate, &resp)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "SendHTTPRequest should error for an unset endpoint")
}

func TestSendTimestampedHTTPRequest(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, "BTC-USDT", q.Get("symbol"), "symbol should be sent")
		ts, err := strconv.ParseInt(q.Get("timestamp"), 10, 64)
		assert.NoError(t, err, "timestamp should be an integer")
		assert.WithinDuration(t, time.Now(), time.UnixMilli(ts), time.Minute, "timestamp should be the current time")
		assert.Empty(t, q.Get("signature"), "signature should not be sent")
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"serverTime":1794406598194}}`)
	})
	params := url.Values{"symbol": {"BTC-USDT"}}
	var resp *SystemTimeResponse
	require.NoError(t, ex.SendTimestampedHTTPRequest(t.Context(), exchange.RestSpot, "/openApi/test", params, systemTimeRate, &resp), "SendTimestampedHTTPRequest must not error")
	assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}}, params, "the caller's parameters should be left unchanged")

	err := ex.SendTimestampedHTTPRequest(t.Context(), exchange.RestFutures, "/openApi/test", nil, systemTimeRate, &resp)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "SendTimestampedHTTPRequest should error for an unset endpoint")

	noParams := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, []string{"timestamp"}, slices.Collect(maps.Keys(r.URL.Query())), "only the timestamp should be sent")
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"serverTime":1794406598194}}`)
	})
	require.NoError(t, noParams.SendTimestampedHTTPRequest(t.Context(), exchange.RestSpot, "/openApi/test", nil, systemTimeRate, &resp), "SendTimestampedHTTPRequest must not error without parameters")
}

func TestSendAPIKeyHTTPRequest(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method, "method should be the requested method")
		assert.Equal(t, testAPIKey, r.Header.Get(keyHeader), "API key header should be sent")
		assert.Equal(t, "listenKey=abc", r.URL.RawQuery, "query should hold only the parameters given")
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{}}`)
	})
	params := url.Values{"listenKey": {"abc"}}
	require.NoError(t, ex.SendAPIKeyHTTPRequest(t.Context(), exchange.RestSpot, http.MethodPut, "/openApi/test", params, extendListenKeyRate, nil), "SendAPIKeyHTTPRequest must not error")

	err := ex.SendAPIKeyHTTPRequest(t.Context(), exchange.RestFutures, http.MethodPut, "/openApi/test", params, extendListenKeyRate, nil)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "SendAPIKeyHTTPRequest should error for an unset endpoint")

	noCreds := newTestServerExchange(t, func(http.ResponseWriter, *http.Request) {
		assert.Fail(t, "a request without credentials should not be sent")
	})
	noCreds.SetCredentials(&accounts.Credentials{})
	err = noCreds.SendAPIKeyHTTPRequest(t.Context(), exchange.RestSpot, http.MethodPut, "/openApi/test", params, extendListenKeyRate, nil)
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "SendAPIKeyHTTPRequest should error without credentials")
}

func TestSendAuthHTTPRequest(t *testing.T) {
	t.Parallel()
	var queries []string
	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "method should be the requested method")
		assert.Equal(t, testAPIKey, r.Header.Get(keyHeader), "API key header should be sent")
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "reading the body should not error")
		assert.Empty(t, body, "body should be empty")
		queries = append(queries, r.URL.RawQuery)
		verifyQuerySignature(t, r.URL.RawQuery)
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"serverTime":1794406598194}}`)
	})
	params := url.Values{"type": {"LIMIT"}, "symbol": {"BTC-USDT"}, "clientOrderId": {"a b&c=d+e"}, "data": {`[{"a":1}]`}}
	original := url.Values{"type": {"LIMIT"}, "symbol": {"BTC-USDT"}, "clientOrderId": {"a b&c=d+e"}, "data": {`[{"a":1}]`}}
	var resp *SystemTimeResponse
	require.NoError(t, ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, http.MethodPost, "/openApi/test", params, swapPlaceOrderRate, &resp), "SendAuthHTTPRequest must not error")
	assert.Equal(t, original, params, "the caller's parameters should be left unchanged")
	require.Len(t, queries, 1, "one request must be sent")
	keys, values := orderedQuery(t, queries[0])
	assert.Equal(t, []string{"clientOrderId", "data", "recvWindow", "symbol", "timestamp", "type", "signature"}, keys, "parameters should be sorted by key, timestamp included, with the signature last")
	assert.Equal(t, "a b&c=d+e", values[0], "a value with reserved characters should arrive intact")
	assert.Equal(t, "5000", values[2], "recvWindow should default to 5000")
	const encodedClientOrderID = "clientOrderId=a%20b%26c%3Dd%2Be"
	assert.Contains(t, queries[0], encodedClientOrderID, "values should be percent-encoded, spaces included")

	params.Set("recvWindow", "10000")
	require.NoError(t, ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, http.MethodPost, "/openApi/test", params, swapPlaceOrderRate, &resp), "SendAuthHTTPRequest must not error")
	require.Len(t, queries, 2, "a second request must be sent")
	_, values = orderedQuery(t, queries[1])
	assert.Equal(t, "10000", values[2], "an explicit recvWindow should be kept")

	require.NoError(t, ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, http.MethodPost, "/openApi/test", nil, swapPlaceOrderRate, nil), "SendAuthHTTPRequest must not error without parameters")

	err := ex.SendAuthHTTPRequest(t.Context(), exchange.RestFutures, http.MethodPost, "/openApi/test", nil, swapPlaceOrderRate, nil)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "SendAuthHTTPRequest should error for an unset endpoint")

	ex.SetCredentials(&accounts.Credentials{})
	err = ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, http.MethodPost, "/openApi/test", nil, swapPlaceOrderRate, nil)
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "SendAuthHTTPRequest should error without credentials")
}

func TestSendAuthJSONBodyHTTPRequest(t *testing.T) {
	t.Parallel()
	type body struct {
		SubUID      uint64   `json:"subUid"`
		Note        string   `json:"note"`
		Permissions []uint64 `json:"permissions"`
		Freeze      bool     `json:"freeze"`
	}
	var received map[string]json.RawMessage
	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "method should be the requested method")
		assert.Equal(t, testAPIKey, r.Header.Get(keyHeader), "API key header should be sent")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "content type should be JSON")
		assert.Empty(t, r.URL.RawQuery, "query should be empty")
		raw, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "reading the body should not error")
		assert.NoError(t, json.Unmarshal(raw, &received), "the body should be a JSON object")
		_, _ = io.WriteString(w, `{"code":0,"timestamp":1,"data":{"serverTime":1794406598194}}`)
	})
	req := &body{SubUID: 12345678, Note: "a&b c", Permissions: []uint64{1, 2}, Freeze: true}
	var resp *SystemTimeResponse
	require.NoError(t, ex.SendAuthJSONBodyHTTPRequest(t.Context(), exchange.RestSpot, http.MethodPost, "/openApi/test", req, createSubAccountRate, &resp), "SendAuthJSONBodyHTTPRequest must not error")
	require.NotNil(t, received, "a body must be received")
	assert.JSONEq(t, `12345678`, string(received["subUid"]), "subUid should be sent")
	assert.JSONEq(t, `[1,2]`, string(received["permissions"]), "permissions should be sent as an array")
	assert.JSONEq(t, `5000`, string(received["recvWindow"]), "recvWindow should default to 5000")
	require.Contains(t, received, "timestamp", "timestamp must be sent")
	var signature string
	require.NoError(t, json.Unmarshal(received["signature"], &signature), "signature must be a string")
	expected := "freeze=true&note=a&b c&permissions=[1,2]&recvWindow=5000&subUid=12345678&timestamp=" + string(received["timestamp"])
	mac := hmac.New(sha256.New, []byte(testAPISecret))
	mac.Write([]byte(expected))
	assert.Equal(t, hex.EncodeToString(mac.Sum(nil)), signature, "signature should cover the sorted members with strings unquoted")

	err := ex.SendAuthJSONBodyHTTPRequest(t.Context(), exchange.RestSpot, http.MethodPost, "/openApi/test", []string{"a"}, createSubAccountRate, nil)
	assert.ErrorIs(t, err, errSigningPayloadObject, "SendAuthJSONBodyHTTPRequest should reject a body that is not an object")

	err = ex.SendAuthJSONBodyHTTPRequest(t.Context(), exchange.RestFutures, http.MethodPost, "/openApi/test", req, createSubAccountRate, nil)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "SendAuthJSONBodyHTTPRequest should error for an unset endpoint")

	ex.SetCredentials(&accounts.Credentials{})
	err = ex.SendAuthJSONBodyHTTPRequest(t.Context(), exchange.RestSpot, http.MethodPost, "/openApi/test", req, createSubAccountRate, nil)
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "SendAuthJSONBodyHTTPRequest should error without credentials")
}

func TestSendRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		status  int
		body    string
		errIs   []error
		errText string
		exp     *SystemTimeResponse
	}{
		{name: "enveloped data", status: http.StatusOK, body: `{"code":0,"msg":"","data":{"serverTime":1794406598194}}`, exp: &SystemTimeResponse{ServerTime: types.Time(time.UnixMilli(1794406598194))}},
		{name: "API error on 200", status: http.StatusOK, body: `{"code":109400,"msg":"Invalid parameters","data":{}}`, errIs: []error{errAPIResponse}, errText: "code 109400 msg Invalid parameters"},
		{name: "API error on 400", status: http.StatusBadRequest, body: `{"code":104414,"msg":"Invalid parameter","retryable":0}`, errIs: []error{errAPIResponse, request.ErrBadStatus}, errText: "code 104414"},
		{name: "bad status without API error", status: http.StatusBadRequest, body: `not json`, errIs: []error{request.ErrBadStatus}},
		{name: "null data", status: http.StatusOK, body: `{"code":0,"msg":"","data":null}`, errIs: []error{common.ErrNoResponse}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			})
			var resp *SystemTimeResponse
			err := ex.SendHTTPRequest(t.Context(), exchange.RestSpot, "/openApi/test", systemTimeRate, &resp)
			if len(tc.errIs) == 0 {
				require.NoError(t, err, "sendRequest must not error")
				assert.Equal(t, tc.exp, resp, "sendRequest should decode the response")
				return
			}
			for _, target := range tc.errIs {
				assert.ErrorIs(t, err, target, "sendRequest should return the expected error")
			}
			assert.NotErrorIs(t, err, errRequestNotSent, "sendRequest should not report a request the server answered as not sent")
			if tc.errText != "" {
				assert.ErrorContains(t, err, tc.errText, "the error should carry the code and message")
			}
		})
	}

	t.Run("endpoint budget alone", func(t *testing.T) {
		t.Parallel()
		var requests atomic.Int32
		server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requests.Add(1)
			_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{}}`)
		}))
		// Every budget but the tested endpoints' own is spent, so a request that also drew on another, such as an IP
		// group's, would be refused
		tested := []request.EndpointLimit{swapPlaceOrderRate, spotPlaceOrderRate, coinMarginedPlaceOrderRate, depositAddressRate}
		definitions := make(request.RateLimitDefinitions)
		for f := range GetRateLimits() {
			definitions[f] = request.NewRateLimitWithWeight(time.Hour, 1, 1)
		}
		ex := new(Exchange)
		var err error
		ex.Requester, err = request.New(t.Name(), server.Client(), request.WithLimiter(definitions))
		require.NoError(t, err, "request.New must not error")
		for k := range definitions {
			f, ok := k.(request.EndpointLimit)
			require.Truef(t, ok, "definition key %v must be an endpoint limit", k)
			if !slices.Contains(tested, f) {
				require.NoErrorf(t, ex.Requester.InitiateRateLimit(t.Context(), f), "InitiateRateLimit must spend the budget of endpoint %d", f)
			}
		}
		send := func(f request.EndpointLimit) error {
			return ex.sendRequest(request.WithDelayNotAllowed(t.Context()), f, request.AuthenticatedRequest, nil, func() (*request.Item, error) {
				return &request.Item{Method: http.MethodGet, Path: server.URL + "/openApi/test"}, nil
			})
		}
		for _, f := range tested {
			assert.NoErrorf(t, send(f), "sendRequest should admit endpoint %d by its own budget alone", f)
			assert.ErrorIsf(t, send(f), request.ErrDelayNotAllowed, "sendRequest should refuse endpoint %d once its own budget is spent", f)
		}
		assert.EqualValues(t, len(tested), requests.Load(), "sendRequest should send each request its endpoint's budget admitted")
	})

	t.Run("item error", func(t *testing.T) {
		t.Parallel()
		err := e.sendRequest(t.Context(), systemTimeRate, request.UnauthenticatedRequest, nil, func() (*request.Item, error) {
			return nil, errInvalidTimeRange
		})
		assert.ErrorIs(t, err, errInvalidTimeRange, "sendRequest should return the request generator's error")
		assert.ErrorIs(t, err, errRequestNotSent, "sendRequest should report a request it could not build as not sent")
	})

	t.Run("rate limit wait past the deadline", func(t *testing.T) {
		t.Parallel()
		var requests atomic.Int32
		server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requests.Add(1)
			_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{}}`)
		}))
		ex := new(Exchange)
		var err error
		ex.Requester, err = request.New(t.Name(), server.Client(), request.WithLimiter(request.RateLimitDefinitions{spotCancelOrderRate: request.NewRateLimitWithWeight(time.Hour, 1, 1)}))
		require.NoError(t, err, "request.New must not error")
		require.NoError(t, ex.Requester.InitiateRateLimit(t.Context(), spotCancelOrderRate), "InitiateRateLimit must spend the endpoint's budget")
		ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
		defer cancel()
		err = ex.sendRequest(ctx, spotCancelOrderRate, request.AuthenticatedRequest, nil, func() (*request.Item, error) {
			return &request.Item{Method: http.MethodPost, Path: server.URL + "/openApi/test"}, nil
		})
		assert.ErrorIs(t, err, errRequestNotSent, "sendRequest should report a request the rate limiter refused as not sent")
		assert.ErrorIs(t, err, context.DeadlineExceeded, "sendRequest should return the rate limiter's error")
		assert.Zero(t, requests.Load(), "sendRequest should send nothing the rate limiter refused")
	})

	t.Run("context ended", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		err := e.sendRequest(ctx, systemTimeRate, request.UnauthenticatedRequest, nil, func() (*request.Item, error) {
			return &request.Item{Method: http.MethodGet, Path: "http://localhost/openApi/test"}, nil
		})
		assert.ErrorIs(t, err, context.Canceled, "sendRequest should return an ended context's error")
		assert.NotErrorIs(t, err, errRequestNotSent, "sendRequest should return an ended context's error as it is, so callers matching it find it alone")
	})
}

// TestTimedOutRequestRetries checks that a write BingX may already have carried out is not sent again after its reply
// times out, nor a USDT-M cancellation, as BingX asks that those not be resubmitted, while reads, other cancellations,
// amendments and settings are
func TestTimedOutRequestRetries(t *testing.T) {
	t.Parallel()
	coinMarginedPair := currency.NewBTCUSD()
	for _, tc := range []struct {
		name    string
		route   string
		retried bool
		call    func(context.Context, *Exchange) error
	}{
		{name: "PlaceSpotOrder", route: "POST /openApi/spot/v1/trade/order", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.PlaceSpotOrder(ctx, &PlaceSpotOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Type: OrderTypeLimit, Price: 60000, Quantity: 0.001})
			return err
		}},
		{name: "PlaceSpotBatchOrders", route: "POST /openApi/spot/v1/trade/batchOrders", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.PlaceSpotBatchOrders(ctx, &PlaceSpotBatchOrdersRequest{Orders: []PlaceSpotOrderRequest{{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Type: OrderTypeLimit, Price: 60000, Quantity: 0.001}}})
			return err
		}},
		{name: "CancelReplaceSpotOrder", route: "POST /openApi/spot/v1/trade/order/cancelReplace", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CancelReplaceSpotOrder(ctx, &CancelReplaceSpotOrderRequest{Symbol: currency.NewBTCUSDT(), CancelOrderID: 1, CancelReplaceMode: CancelReplaceModeStopOnFailure, Side: SideBuy, Type: OrderTypeLimit, Price: 60000, Quantity: 0.001})
			return err
		}},
		{name: "PlaceSpotOCOOrder", route: "POST /openApi/spot/v1/oco/order", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.PlaceSpotOCOOrder(ctx, &PlaceSpotOCOOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Quantity: 0.001, LimitPrice: 48000, OrderPrice: 88000, TriggerPrice: 87000})
			return err
		}},
		{name: "CancelSpotOpenOrders", route: "POST /openApi/spot/v1/trade/cancelOpenOrders", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CancelSpotOpenOrders(ctx, currency.NewBTCUSDT())
			return err
		}},
		{name: "CancelAllSpotOrdersAfter", route: "POST /openApi/spot/v1/trade/cancelAllAfter", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CancelAllSpotOrdersAfter(ctx, CancelAllAfterTypeActivate, time.Minute)
			return err
		}},
		{name: "CancelSwapOrder", route: "DELETE /openApi/swap/v2/trade/order", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CancelSwapOrder(ctx, currency.NewBTCUSDT(), 1, "")
			return err
		}},
		{name: "CancelSwapBatchOrders", route: "DELETE /openApi/swap/v2/trade/batchOrders", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CancelSwapBatchOrders(ctx, currency.NewBTCUSDT(), []uint64{1}, nil)
			return err
		}},
		{name: "CancelSwapBatchOrders by client order ID", route: "DELETE /openApi/swap/v2/trade/batchOrders", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CancelSwapBatchOrders(ctx, currency.NewBTCUSDT(), nil, []string{"c1"})
			return err
		}},
		{name: "CancelSwapBatchOrders of several orders", route: "DELETE /openApi/swap/v2/trade/batchOrders", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CancelSwapBatchOrders(ctx, currency.NewBTCUSDT(), []uint64{1, 2}, []string{"c1", "c2"})
			return err
		}},
		{name: "CancelSwapBatchOrders of several order IDs", route: "DELETE /openApi/swap/v2/trade/batchOrders", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CancelSwapBatchOrders(ctx, currency.NewBTCUSDT(), []uint64{1, 2}, nil)
			return err
		}},
		{name: "CancelSwapBatchOrders of several client order IDs", route: "DELETE /openApi/swap/v2/trade/batchOrders", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CancelSwapBatchOrders(ctx, currency.NewBTCUSDT(), nil, []string{"c1", "c2"})
			return err
		}},
		{name: "CancelSwapTWAPOrder", route: "POST /openApi/swap/v1/twap/cancelOrder", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CancelSwapTWAPOrder(ctx, "1")
			return err
		}},
		{name: "CancelAllSwapOpenOrders", route: "DELETE /openApi/swap/v2/trade/allOpenOrders", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CancelAllSwapOpenOrders(ctx, currency.NewBTCUSDT(), "")
			return err
		}},
		{name: "CancelAllSwapOrdersAfter", route: "POST /openApi/swap/v2/trade/cancelAllAfter", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CancelAllSwapOrdersAfter(ctx, CancelAllAfterTypeActivate, time.Minute)
			return err
		}},
		{name: "CancelAllCoinMarginedOrders", route: "DELETE /openApi/cswap/v1/trade/allOpenOrders", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CancelAllCoinMarginedOrders(ctx, coinMarginedPair)
			return err
		}},
		{name: "PlaceSwapOrder", route: "POST /openApi/swap/v2/trade/order", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.PlaceSwapOrder(ctx, swapTradeMarketOrder())
			return err
		}},
		{name: "PlaceSwapBatchOrders", route: "POST /openApi/swap/v2/trade/batchOrders", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.PlaceSwapBatchOrders(ctx, []SwapOrderRequest{*swapTradeMarketOrder()})
			return err
		}},
		{name: "CancelReplaceSwapOrder", route: "POST /openApi/swap/v1/trade/cancelReplace", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CancelReplaceSwapOrder(ctx, &CancelReplaceSwapOrderRequest{SwapOrderRequest: *swapTradeMarketOrder(), CancelReplaceMode: CancelReplaceModeStopOnFailure, CancelOrderID: 1})
			return err
		}},
		{name: "CancelReplaceSwapBatchOrders", route: "POST /openApi/swap/v1/trade/batchCancelReplace", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CancelReplaceSwapBatchOrders(ctx, []CancelReplaceSwapOrderRequest{{SwapOrderRequest: *swapTradeMarketOrder(), CancelReplaceMode: CancelReplaceModeStopOnFailure, CancelOrderID: 1}})
			return err
		}},
		{name: "CloseSwapPosition", route: "POST /openApi/swap/v1/trade/closePosition", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CloseSwapPosition(ctx, 1)
			return err
		}},
		{name: "CloseAllSwapPositions", route: "POST /openApi/swap/v2/trade/closeAllPositions", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CloseAllSwapPositions(ctx, currency.NewBTCUSDT())
			return err
		}},
		{name: "ReverseSwapPosition", route: "POST /openApi/swap/v1/trade/reverse", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.ReverseSwapPosition(ctx, &ReverseSwapPositionRequest{Type: ReverseTypeReverse, Symbol: currency.NewBTCUSDT()})
			return err
		}},
		{name: "ModifySwapIsolatedPositionMargin", route: "POST /openApi/swap/v2/trade/positionMargin", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.ModifySwapIsolatedPositionMargin(ctx, &ModifySwapIsolatedPositionMarginRequest{Symbol: currency.NewBTCUSDT(), Amount: 3, Type: SwapMarginAdjustmentTypeIncrease, PositionSide: PositionSideLong})
			return err
		}},
		{name: "PlaceSwapTWAPOrder", route: "POST /openApi/swap/v1/twap/order", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.PlaceSwapTWAPOrder(ctx, &SwapTWAPOrderRequest{
				Symbol: currency.NewBTCUSDT(), Side: SideBuy, PositionSide: PositionSideLong, PriceType: TWAPPriceTypeConstant, PriceVariance: 2000,
				TriggerPrice: 68000, Interval: 8 * time.Second, AmountPerOrder: 0.111, TotalAmount: 0.511,
			})
			return err
		}},
		{name: "ApplySwapVST", route: "POST /openApi/swap/v2/trade/getVst", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.ApplySwapVST(ctx, VSTAdjustTypeIncrease, 500000)
			return err
		}},
		{name: "PlaceCoinMarginedOrder", route: "POST /openApi/cswap/v2/trade/order", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.PlaceCoinMarginedOrder(ctx, &PlaceCoinMarginedOrderRequest{Symbol: coinMarginedPair, Type: OrderTypeLimit, Side: SideBuy, PositionSide: PositionSideLong, Price: 60000, Quantity: 1})
			return err
		}},
		{name: "CloseAllCoinMarginedPositions", route: "POST /openApi/cswap/v1/trade/closeAllPositions", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CloseAllCoinMarginedPositions(ctx, coinMarginedPair)
			return err
		}},
		{name: "AdjustCoinMarginedIsolatedMargin", route: "POST /openApi/cswap/v1/trade/positionMargin", call: func(ctx context.Context, ex *Exchange) error {
			return ex.AdjustCoinMarginedIsolatedMargin(ctx, &AdjustCoinMarginedIsolatedMarginRequest{Symbol: coinMarginedPair, Amount: 0.01, Type: CoinMarginedMarginAdjustmentTypeIncrease, PositionSide: PositionSideLong})
		}},
		{name: "Withdraw", route: "POST /openApi/wallets/v1/capital/withdraw/apply", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.Withdraw(ctx, &WithdrawRequest{Coin: currency.USDT, Network: "TRC20", Address: "TR****zc", Amount: 12.5, WalletType: WalletTypeSpot})
			return err
		}},
		{name: "AssetTransfer", route: "POST /openApi/api/asset/v1/transfer", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.AssetTransfer(ctx, &AssetTransferRequest{FromAccount: TransferAccountFund, ToAccount: TransferAccountSpot, Asset: currency.USDT, Amount: 1095})
			return err
		}},
		{name: "MainAccountInternalTransfer", route: "POST /openApi/wallets/v1/capital/innerTransfer/apply", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.MainAccountInternalTransfer(ctx, &InternalTransferRequest{Coin: currency.USDT, UserAccountType: UserAccountTypeUID, UserAccount: "16689999", Amount: 0.5, WalletType: WalletTypeSpot})
			return err
		}},
		{name: "SubAccountInternalTransfer", route: "POST /openApi/wallets/v1/capital/subAccountInnerTransfer/apply", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SubAccountInternalTransfer(ctx, &InternalTransferRequest{Coin: currency.USDT, UserAccountType: UserAccountTypeUID, UserAccount: "16689999", Amount: 20, WalletType: WalletTypeFund})
			return err
		}},
		{name: "SubMotherAccountAssetTransfer", route: "POST /openApi/account/transfer/v1/subAccount/transferAsset", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SubMotherAccountAssetTransfer(ctx, &SubMotherAccountAssetTransferRequest{
				AssetName: currency.USDT, TransferAmount: 1.1, FromUID: 25472377, FromType: SubAccountTypeSubAccount, FromAccountType: WalletTypeStandardFutures,
				ToUID: 25316652, ToType: SubAccountTypeParentAccount, ToAccountType: WalletTypeFund, Remark: "rebalance",
			})
			return err
		}},
		{name: "CreateSubAccount", route: "POST /openApi/subAccount/v1/create", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CreateSubAccount(ctx, &CreateSubAccountRequest{SubAccountString: "abc123456"})
			return err
		}},
		{name: "CreateSubAccountAPIKey", route: "POST /openApi/subAccount/v1/apiKey/create", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CreateSubAccountAPIKey(ctx, &CreateSubAccountAPIKeyRequest{SubUID: 14189999, Note: "abc6798", Permissions: []APIKeyPermission{APIKeyPermissionRead}})
			return err
		}},
		{name: "CreateSignal", route: "POST /openApi/social/v1/swap/signal", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CreateSignal(ctx, &CreateSignalRequest{Symbol: currency.NewBTCUSDT(), Side: SignalSideLong, EntryPrice: 65000, TakeProfitPrice: 68000, StopLossPrice: 62000.5, OrderType: OrderTypeLimit, Leverage: 20, SizePercentage: 10})
			return err
		}},
		{name: "PlaceDualCurrencyOrder", route: "POST /openApi/wealth/v1/product/dual-currency/order", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.PlaceDualCurrencyOrder(ctx, &DualCurrencyOrderRequest{
				Amount: 0.01, SelectAccounts: []DualCurrencyAccount{DualCurrencyAccountFund}, SKU: "DUAL-BTC-USDT-CALL-8H", ProductID: 1234567890,
				StrikePrice: 77900.5, QuoteID: "quote_id_from_pre_order", SettleTargetAccount: DualCurrencyAccountSpot,
			})
			return err
		}},
		{name: "SellCopyTradingSpotOrder", route: "POST /openApi/copyTrading/v1/spot/trader/sellOrder", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SellCopyTradingSpotOrder(ctx, 1253517936071234567)
			return err
		}},
		{name: "generateListenKey", route: "POST /openApi/user/auth/userDataStream", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.generateListenKey(ctx, exchange.RestSpot)
			return err
		}},
		{name: "GetSpotAccountAssets", route: "GET /openApi/spot/v1/account/balance", retried: true, call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.GetSpotAccountAssets(ctx)
			return err
		}},
		{name: "CancelSpotOrder", route: "POST /openApi/spot/v1/trade/cancel", retried: true, call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CancelSpotOrder(ctx, &CancelSpotOrderRequest{Symbol: currency.NewBTCUSDT(), OrderID: 1})
			return err
		}},
		{name: "CancelCoinMarginedOrder", route: "DELETE /openApi/cswap/v1/trade/cancelOrder", retried: true, call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CancelCoinMarginedOrder(ctx, coinMarginedPair, 1, "")
			return err
		}},
		{name: "ModifySwapOrder", route: "POST /openApi/swap/v1/trade/amend", retried: true, call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.ModifySwapOrder(ctx, &ModifySwapOrderRequest{Symbol: currency.NewBTCUSDT(), OrderID: 1, Quantity: 2})
			return err
		}},
		{name: "SetSwapLeverage", route: "POST /openApi/swap/v2/trade/leverage", retried: true, call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.SetSwapLeverage(ctx, currency.NewBTCUSDT(), PositionSideLong, 5)
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var routes []string
			received := func() []string {
				mu.Lock()
				defer mu.Unlock()
				return slices.Clone(routes)
			}
			ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
				// The server notices the client giving up, which ends the request's context, only once the body is read
				_, err := io.Copy(io.Discard, r.Body)
				assert.NoError(t, err, "reading the request body should not error")
				mu.Lock()
				routes = append(routes, r.Method+" "+r.URL.Path)
				first := len(routes) == 1
				mu.Unlock()
				if first {
					select {
					case <-r.Context().Done():
					case <-time.After(5 * time.Second):
					}
					return
				}
				w.WriteHeader(http.StatusBadRequest)
			})
			// Long enough for a request to arrive while the package's parallel tests load the machine
			require.NoError(t, ex.SetHTTPClientTimeout(time.Second), "SetHTTPClientTimeout must not error")
			err := tc.call(t.Context(), ex)
			// A request is sent again before the call returns, but the first may reach the server after the client gave up
			require.Eventuallyf(t, func() bool { return len(received()) != 0 }, 5*time.Second, 10*time.Millisecond, "%s must send a request", tc.name)
			got := received()
			assert.Equalf(t, tc.route, got[0], "%s should send its own request first", tc.name)
			if tc.retried {
				assert.ErrorIsf(t, err, request.ErrBadStatus, "%s should return its retry's reply", tc.name)
				assert.Equalf(t, []string{tc.route, tc.route}, got, "%s should send a timed-out request again", tc.name)
				return
			}
			assert.ErrorIsf(t, err, context.DeadlineExceeded, "%s should return the timeout", tc.name)
			assert.Lenf(t, got, 1, "%s should send a timed-out request once", tc.name)
		})
	}
}

func TestCloneParams(t *testing.T) {
	t.Parallel()
	assert.Equal(t, url.Values{}, cloneParams(nil), "cloneParams should return empty values for nil")
	params := url.Values{"a": {"1"}}
	clone := cloneParams(params)
	clone.Set("b", "2")
	assert.Equal(t, url.Values{"a": {"1"}}, params, "setting a value on the clone should leave the original unchanged")
}

func TestSigningString(t *testing.T) {
	t.Parallel()
	params := url.Values{"symbol": {"BTC-USDT"}, "recvWindow": {"0"}, "timestamp": {"1696751141337"}, "type": {"LIMIT"}, "signature": {"ignored"}, "data": {`[{"a":"b c"}]`}}
	assert.Equal(t, `data=[{"a":"b c"}]&recvWindow=0&symbol=BTC-USDT&timestamp=1696751141337&type=LIMIT`, signingString(params), "signingString should sort every parameter, timestamp included, with raw values and no signature")
	assert.Empty(t, signingString(nil), "signingString should be empty without parameters")
}

func TestEncodedQuery(t *testing.T) {
	t.Parallel()
	params := url.Values{"symbol": {"BTC-USDT"}, "timestamp": {"1"}, "data": {`[{"a":"b c"}]`}, "signature": {"ignored"}}
	const exp = "data=%5B%7B%22a%22%3A%22b%20c%22%7D%5D&symbol=BTC-USDT&timestamp=1"
	assert.Equal(t, exp, encodedQuery(params), "encodedQuery should encode values in signing order without the signature")
}

func TestSignQuery(t *testing.T) {
	t.Parallel()
	params := url.Values{"recvWindow": {"0"}, "symbol": {"BTC-USDT"}, "timestamp": {"1696751141337"}}
	query := signQuery(params, "SECRET_KEY")
	mac := hmac.New(sha256.New, []byte("SECRET_KEY"))
	mac.Write([]byte("recvWindow=0&symbol=BTC-USDT&timestamp=1696751141337"))
	assert.Equal(t, "recvWindow=0&symbol=BTC-USDT&timestamp=1696751141337&signature="+hex.EncodeToString(mac.Sum(nil)), query, "signQuery should append the signature over the documented example")
}

func TestSign(t *testing.T) {
	t.Parallel()
	signature := sign("recvWindow=0&symbol=BTC-USDT&timestamp=1696751141337", "SECRET_KEY")
	assert.Len(t, signature, 64, "sign should return 64 hex characters")
	assert.Equal(t, strings.ToLower(signature), signature, "sign should return lowercase hex")
	// The output of the openssl command the documentation signs this example with
	assert.Equal(t, "fe041f159118c90ac13eab4d32f9e2d75b80ca6fe17ca8acd290aba864753ce2", signature, "sign should match the documented HMAC-SHA256 command")
}

func TestJSONBodyFields(t *testing.T) {
	t.Parallel()
	fields, err := jsonBodyFields(nil)
	require.NoError(t, err, "jsonBodyFields must not error for nil")
	assert.Empty(t, fields, "jsonBodyFields should return no fields for nil")

	fields, err = jsonBodyFields(map[string]any{"a": 1, "b": "x"})
	require.NoError(t, err, "jsonBodyFields must not error for an object")
	assert.Equal(t, map[string]json.RawMessage{"a": json.RawMessage("1"), "b": json.RawMessage(`"x"`)}, fields, "jsonBodyFields should return the object's members")

	var nilMap map[string]any
	fields, err = jsonBodyFields(nilMap)
	require.NoError(t, err, "jsonBodyFields must not error for a nil map")
	assert.NotNil(t, fields, "jsonBodyFields should return an empty map for a nil map")

	_, err = jsonBodyFields([]int{1})
	assert.ErrorIs(t, err, errSigningPayloadObject, "jsonBodyFields should reject an array")

	_, err = jsonBodyFields(func() {})
	assert.Error(t, err, "jsonBodyFields should error for a value that cannot be encoded")
}

func TestSignJSONBody(t *testing.T) {
	t.Parallel()
	fields := map[string]json.RawMessage{"b": json.RawMessage(`"x y"`), "a": json.RawMessage(`[1,{"c":2}]`), "timestamp": json.RawMessage("7"), "signature": json.RawMessage(`"old"`)}
	payload, err := signJSONBody(fields, "SECRET_KEY")
	require.NoError(t, err, "signJSONBody must not error")
	var out map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(payload, &out), "signJSONBody must return a JSON object")
	mac := hmac.New(sha256.New, []byte("SECRET_KEY"))
	mac.Write([]byte(`a=[1,{"c":2}]&b=x y&timestamp=7`))
	assert.JSONEq(t, strconv.Quote(hex.EncodeToString(mac.Sum(nil))), string(out["signature"]), "signature should replace any given one and cover the sorted members")
	assert.JSONEq(t, `"old"`, string(fields["signature"]), "the given fields should be left unchanged")

	_, err = signJSONBody(map[string]json.RawMessage{"a": json.RawMessage(`"unterminated`)}, "SECRET_KEY")
	assert.Error(t, err, "signJSONBody should error for a member that is not valid JSON")
}

func TestSigningValue(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		raw, exp string
	}{
		{`"plain"`, "plain"},
		{` "with \"quote\"" `, `with "quote"`},
		{`12.5`, "12.5"},
		{`true`, "true"},
		{`[1,"a"]`, `[1,"a"]`},
		{`{"k":"v"}`, `{"k":"v"}`},
	} {
		value, err := signingValue(json.RawMessage(tc.raw))
		require.NoErrorf(t, err, "signingValue must not error for %s", tc.raw)
		assert.Equalf(t, tc.exp, value, "signingValue should render %s", tc.raw)
	}
	_, err := signingValue(json.RawMessage(`"bad`))
	assert.Error(t, err, "signingValue should error for an unterminated string")
}

func TestCheckErrorResponse(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		raw  string
		exp  error
	}{
		{"empty", ``, nil},
		{"array", `[1]`, nil},
		{"no code", `{"listenKey":"a"}`, nil},
		{"zero code", `{"code":0,"msg":""}`, nil},
		{"zero string code", `{"code":"0"}`, nil},
		{"bad number code", `{"code":1.5}`, nil},
		{"bad string code", `{"code":"x"}`, nil},
		{"bool code", `{"code":true}`, nil},
		{"error", `{"code":109400,"msg":"Invalid parameters","data":{}}`, &APIError{Code: 109400, Message: "Invalid parameters"}},
		{"string code", `{"code":"100400","msg":"bad"}`, &APIError{Code: 100400, Message: "bad"}},
		{"error data", `{"code":100400,"msg":"partial","data":{"cancelResult":"FAIL"}}`, &APIError{Code: 100400, Message: "partial", Data: json.RawMessage(`{"cancelResult":"FAIL"}`)}},
		{"empty array data", `{"code":109419,"msg":"m","data":[]}`, &APIError{Code: 109419, Message: "m"}},
		{"null data", `{"code":1,"msg":"m","data":null}`, &APIError{Code: 1, Message: "m"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := checkErrorResponse(json.RawMessage(tc.raw))
			if tc.exp == nil {
				assert.NoError(t, err, "checkErrorResponse should not error")
				return
			}
			assert.Equal(t, tc.exp, err, "checkErrorResponse should return the API error")
		})
	}
}

func TestResponseCode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		raw  string
		code int64
		ok   bool
	}{
		{``, 0, false},
		{`[1]`, 0, false},
		{`{"listenKey":"a"}`, 0, false},
		{`{"code":109400}`, 109400, true},
		{`{"code":"100400"}`, 100400, true},
		{`{"code":1.5}`, 0, false},
		{`{"code":"x"}`, 0, false},
		{`{"code":true}`, 0, false},
	} {
		code, ok := responseCode(json.RawMessage(tc.raw))
		assert.Equalf(t, tc.ok, ok, "responseCode should report %t for %q", tc.ok, tc.raw)
		assert.Equalf(t, tc.code, code, "responseCode should return %d for %q", tc.code, tc.raw)
	}
}

func TestBadStatusCode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{name: "firewall page", status: http.StatusForbidden, body: "<html><title>403 Forbidden</title></html>"},
		{name: "rate limit page", status: http.StatusTooManyRequests, body: "<html><title>429 Too Many Requests</title></html>"},
		{name: "gateway timeout page", status: http.StatusGatewayTimeout, body: "<html><title>504 Gateway Time-out</title></html>"},
		{name: "API error on an error status", status: http.StatusBadRequest, body: `{"code":104414,"msg":"Invalid parameter"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			})
			err := ex.SendAuthHTTPRequest(request.WithRetryNotAllowed(t.Context()), exchange.RestSpot, http.MethodPost, "/openApi/test", nil, spotCancelOrderRate, nil)
			require.ErrorIs(t, err, request.ErrBadStatus, "SendAuthHTTPRequest must return the requester's bad status error")
			status, ok := badStatusCode(err)
			require.True(t, ok, "badStatusCode must find the reply's status")
			assert.Equal(t, tc.status, status, "badStatusCode should return the reply's HTTP status")
		})
	}
	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "another error", err: context.DeadlineExceeded},
		{name: "bad status without its text", err: request.ErrBadStatus},
		{name: "bad status without a number", err: fmt.Errorf("%w: unknown", request.ErrBadStatus)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, ok := badStatusCode(tc.err)
			assert.False(t, ok, "badStatusCode should find no status in an error without one")
		})
	}
}

func TestIsEmptyJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		value string
		exp   bool
	}{{`{}`, true}, {" [] ", true}, {`""`, true}, {`{"a":1}`, false}, {`0`, false}} {
		assert.Equalf(t, tc.exp, isEmptyJSON([]byte(tc.value)), "isEmptyJSON should report %t for %q", tc.exp, tc.value)
	}
}

func TestDecodeResponse(t *testing.T) {
	t.Parallel()
	type pair struct {
		Total uint64            `json:"total"`
		Rows  []json.RawMessage `json:"rows"`
	}
	type margin struct {
		Amount float64 `json:"amount"`
		Type   uint64  `json:"type"`
	}
	t.Run("data member", func(t *testing.T) {
		t.Parallel()
		var resp *ListenKeyResponse
		require.NoError(t, decodeResponse(json.RawMessage(`{"code":0,"msg":"","data":{"listenKey":"a"}}`), &resp), "decodeResponse must not error")
		assert.Equal(t, &ListenKeyResponse{ListenKey: "a"}, resp, "decodeResponse should decode the data member")
	})
	t.Run("bare object", func(t *testing.T) {
		t.Parallel()
		var resp *ListenKeyResponse
		require.NoError(t, decodeResponse(json.RawMessage(`{"listenKey":"b"}`), &resp), "decodeResponse must not error")
		assert.Equal(t, &ListenKeyResponse{ListenKey: "b"}, resp, "decodeResponse should decode a body without the envelope")
	})
	t.Run("bare array", func(t *testing.T) {
		t.Parallel()
		var resp []ListenKeyResponse
		require.NoError(t, decodeResponse(json.RawMessage(`[{"listenKey":"c"}]`), &resp), "decodeResponse must not error")
		assert.Equal(t, []ListenKeyResponse{{ListenKey: "c"}}, resp, "decodeResponse should decode a bare array")
	})
	t.Run("rows and total", func(t *testing.T) {
		t.Parallel()
		var resp *pair
		require.NoError(t, decodeResponse(json.RawMessage(`{"total":1,"rows":[{}]}`), &resp), "decodeResponse must not error")
		assert.Equal(t, &pair{Total: 1, Rows: []json.RawMessage{json.RawMessage(`{}`)}}, resp, "decodeResponse should decode the whole body")
	})
	t.Run("top level fields beside code", func(t *testing.T) {
		t.Parallel()
		var resp *margin
		require.NoError(t, decodeResponse(json.RawMessage(`{"amount":1.5,"code":0,"msg":"","type":1}`), &resp), "decodeResponse must not error")
		assert.Equal(t, &margin{Amount: 1.5, Type: 1}, resp, "decodeResponse should decode fields beside the code")
	})
	t.Run("no result wanted", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, decodeResponse(json.RawMessage(`{"code":0,"msg":""}`), nil), "decodeResponse should not error when no result is wanted")
	})
	t.Run("empty and null bodies", func(t *testing.T) {
		t.Parallel()
		var resp *ListenKeyResponse
		assert.ErrorIs(t, decodeResponse(nil, &resp), common.ErrNoResponse, "decodeResponse should error for an empty body")
		assert.ErrorIs(t, decodeResponse(json.RawMessage(`null`), &resp), common.ErrNoResponse, "decodeResponse should error for a null body")
		assert.ErrorIs(t, decodeResponse(json.RawMessage(`{"code":0,"data":null}`), &resp), common.ErrNoResponse, "decodeResponse should error for null data")
		assert.ErrorIs(t, decodeResponse(json.RawMessage(`{"code":0,"data":null}`), &resp), errNullData, "decodeResponse should say the data is null")
		assert.NotErrorIs(t, decodeResponse(nil, &resp), errNullData, "decodeResponse should not report an empty body as null data")
		assert.NotErrorIs(t, decodeResponse(json.RawMessage(`null`), &resp), errNullData, "decodeResponse should not report a null body as null data")
	})
	t.Run("malformed envelope", func(t *testing.T) {
		t.Parallel()
		var resp *ListenKeyResponse
		assert.ErrorIs(t, decodeResponse(json.RawMessage(`{"data":`), &resp), errUnexpectedEnvelope, "decodeResponse should error for a truncated envelope")
		assert.Error(t, decodeResponse(json.RawMessage(`{"data":"x"}`), &resp), "decodeResponse should error when data does not decode into the result")
	})
}

func TestAPIError(t *testing.T) {
	t.Parallel()
	err := &APIError{Code: 109421, Message: "order not found"}
	assert.Equal(t, "BingX API error: order not found: code 109421 msg order not found", err.Error(), "Error should name the mapped error, code and message")
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "a mapped code should match its package error")
	assert.ErrorIs(t, err, errAPIResponse, "every API error should match errAPIResponse")

	err = &APIError{Code: 100404, Message: "not found", Data: json.RawMessage(`{"a":1}`)}
	assert.Equal(t, `BingX API error: code 100404 msg not found data {"a":1}`, err.Error(), "Error should add the data")
	assert.NotErrorIs(t, err, order.ErrOrderNotFound, "an ambiguous code should not be mapped")
	assert.Equal(t, []error{errAPIResponse}, err.Unwrap(), "Unwrap should return errAPIResponse alone for an unmapped code")

	for code, target := range errorCodeToErrorMap {
		assert.ErrorIsf(t, &APIError{Code: code}, target, "code %d should match its mapped error", code)
	}
	assert.ErrorIs(t, &APIError{Code: 109425}, currency.ErrPairNotFound, "an unknown pair code should match ErrPairNotFound")
	assert.ErrorIs(t, &APIError{Code: 100001}, request.ErrAuthRequestFailed, "a signature failure should match ErrAuthRequestFailed")
}

func TestAPIErrorOutcomeUnknown(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		code    int64
		unknown bool
	}{
		{code: 100500, unknown: true},
		{code: 109500, unknown: true},
		{code: 109701, unknown: true},
		{code: 110500, unknown: true},
		{code: 100400},
		{code: 100404},
		{code: 100410},
		{code: 100421},
		{code: 109400},
		{code: 109421},
		{code: 110400},
	} {
		t.Run(strconv.FormatInt(tc.code, 10), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.unknown, (&APIError{Code: tc.code}).outcomeUnknown(), "outcomeUnknown should report only the busy, internal and network failure codes")
		})
	}
}

func TestIntervalToString(t *testing.T) {
	t.Parallel()
	for interval, exp := range map[kline.Interval]string{
		kline.OneMin: "1m", kline.ThreeMin: "3m", kline.FiveMin: "5m", kline.FifteenMin: "15m", kline.ThirtyMin: "30m",
		kline.OneHour: "1h", kline.TwoHour: "2h", kline.FourHour: "4h", kline.SixHour: "6h", kline.EightHour: "8h",
		kline.TwelveHour: "12h", kline.OneDay: "1d", kline.ThreeDay: "3d", kline.OneWeek: "1w", kline.OneMonth: "1M",
	} {
		s, err := intervalToString(interval)
		require.NoErrorf(t, err, "intervalToString must not error for %s", interval)
		assert.Equalf(t, exp, s, "intervalToString should return %s for %s", exp, interval)
	}
	_, err := intervalToString(kline.TenMin)
	assert.ErrorIs(t, err, errUnsupportedInterval, "intervalToString should reject an unsupported interval")
}

func TestSpotWebsocketIntervalToString(t *testing.T) {
	t.Parallel()
	for interval, exp := range map[kline.Interval]string{
		kline.OneMin: "1min", kline.ThreeMin: "3min", kline.FiveMin: "5min", kline.FifteenMin: "15min", kline.ThirtyMin: "30min",
		kline.OneHour: "60min", kline.TwoHour: "2hour", kline.FourHour: "4hour", kline.SixHour: "6hour", kline.EightHour: "8hour",
		kline.TwelveHour: "12hour", kline.OneDay: "1day", kline.ThreeDay: "3day", kline.OneWeek: "1week", kline.OneMonth: "1mon",
	} {
		s, err := spotWebsocketIntervalToString(interval)
		require.NoErrorf(t, err, "spotWebsocketIntervalToString must not error for %s", interval)
		assert.Equalf(t, exp, s, "spotWebsocketIntervalToString should return %s for %s", exp, interval)
	}
	_, err := spotWebsocketIntervalToString(kline.TenMin)
	assert.ErrorIs(t, err, errUnsupportedInterval, "spotWebsocketIntervalToString should reject an unsupported interval")
}

func TestSetTimeRange(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1000), time.UnixMilli(2000)
	params := url.Values{}
	require.NoError(t, setTimeRange(params, "startTs", "endTs", start, end), "setTimeRange must not error")
	assert.Equal(t, url.Values{"startTs": {"1000"}, "endTs": {"2000"}}, params, "setTimeRange should set both bounds under the given keys")

	params = url.Values{}
	require.NoError(t, setTimeRange(params, "startTime", "endTime", start, time.Time{}), "setTimeRange must not error for an open end")
	assert.Equal(t, url.Values{"startTime": {"1000"}}, params, "setTimeRange should leave an unset bound out")

	require.NoError(t, setTimeRange(params, "startTime", "endTime", start, start), "setTimeRange must not error for equal bounds")
	err := setTimeRange(url.Values{}, "startTime", "endTime", end, start)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd, "setTimeRange should reject a reversed range")
	assert.ErrorIs(t, err, errInvalidTimeRange, "setTimeRange should wrap errInvalidTimeRange")
}

func TestNumericIDUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		raw string
		exp NumericID
	}{
		{`1970374651259387904`, 1970374651259387904},
		{`"1970374651259387904"`, 1970374651259387904},
		{`null`, 0},
		{`""`, 0},
		{`18446744073709551615`, 18446744073709551615},
	} {
		var id NumericID
		require.NoErrorf(t, json.Unmarshal([]byte(tc.raw), &id), "Unmarshal must not error for %s", tc.raw)
		assert.Equalf(t, tc.exp, id, "Unmarshal should decode %s exactly", tc.raw)
		direct := NumericID(99)
		require.NoErrorf(t, direct.UnmarshalJSON([]byte(tc.raw)), "UnmarshalJSON must not error for %s", tc.raw)
		assert.Equalf(t, tc.exp, direct, "UnmarshalJSON should decode %s exactly", tc.raw)
	}
	for _, raw := range []string{`"abc"`, `-1`, `1.5`, `18446744073709551616`} {
		var id NumericID
		assert.ErrorIsf(t, json.Unmarshal([]byte(raw), &id), errInvalidNumericID, "Unmarshal should reject %s", raw)
		assert.ErrorIsf(t, id.UnmarshalJSON([]byte(raw)), errInvalidNumericID, "UnmarshalJSON should reject %s", raw)
	}
}

func TestNumericIDString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1970374651259387904", NumericID(1970374651259387904).String(), "String should return the decimal form")
}

func TestObjectOrArrayUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var list objectOrArray[ListenKeyResponse]
	require.NoError(t, json.Unmarshal([]byte(`{"listenKey":"a"}`), &list), "Unmarshal must not error for an object")
	assert.Equal(t, objectOrArray[ListenKeyResponse]{{ListenKey: "a"}}, list, "an object should decode as one element")
	require.NoError(t, json.Unmarshal([]byte(` [{"listenKey":"b"},{"listenKey":"c"}]`), &list), "Unmarshal must not error for an array")
	assert.Equal(t, objectOrArray[ListenKeyResponse]{{ListenKey: "b"}, {ListenKey: "c"}}, list, "an array should decode element by element")
	assert.Error(t, json.Unmarshal([]byte(`{"listenKey":1}`), &list), "Unmarshal should error for a bad object")
	assert.Error(t, json.Unmarshal([]byte(`"x"`), &list), "Unmarshal should error for a value that is neither an object nor an array")

	var direct objectOrArray[ListenKeyResponse]
	require.NoError(t, direct.UnmarshalJSON([]byte(" \n{\"listenKey\":\"d\"} ")), "UnmarshalJSON must not error for an object")
	assert.Equal(t, objectOrArray[ListenKeyResponse]{{ListenKey: "d"}}, direct, "UnmarshalJSON should decode an object surrounded by whitespace as one element")
	require.NoError(t, direct.UnmarshalJSON([]byte(`[{"listenKey":"e"},{"listenKey":"f"}]`)), "UnmarshalJSON must not error for an array")
	assert.Equal(t, objectOrArray[ListenKeyResponse]{{ListenKey: "e"}, {ListenKey: "f"}}, direct, "UnmarshalJSON should replace the list with the array's elements")
	assert.Error(t, direct.UnmarshalJSON([]byte(`{"listenKey":1}`)), "UnmarshalJSON should error for a bad object")
	assert.Error(t, direct.UnmarshalJSON([]byte(`"x"`)), "UnmarshalJSON should error for a value that is neither an object nor an array")
}

func TestGetRateLimits(t *testing.T) {
	t.Parallel()
	limits := GetRateLimits()
	for f := systemTimeRate; f <= websocketHeartbeatRate; f++ {
		assert.NotNilf(t, limits[f], "endpoint limit %d should have a definition", f)
	}
	assert.Len(t, limits, int(websocketHeartbeatRate)+1, "GetRateLimits should define the endpoint and websocket budgets alone")
	assert.True(t, limits[spotOrderBookRate].SharesBudgetWith(limits[swapKlinesRate]), "public market data endpoints should share one budget")
	assert.True(t, limits[coinMarginedTickerRate].SharesBudgetWith(limits[systemTimeRate]), "public market data endpoints should share one budget")
	assert.False(t, limits[depositAddressRate].SharesBudgetWith(limits[systemTimeRate]), "the signed deposit address endpoint should keep its own budget")
	assert.InDelta(t, 50, float64(limits[systemTimeRate].Limit()+limits[depositAddressRate].Limit()), 1e-9, "group 1's market data and deposit address budgets should together allow 500 requests each 10 seconds")
	undocumented := []request.EndpointLimit{swapMultiAssetsRulesRate, subAccountTransferHistoryRate, subAccountTransferableAmountRate, subAccountAssetTransferRate}
	for i, f := range undocumented {
		for _, other := range undocumented[i+1:] {
			assert.Falsef(t, limits[f].SharesBudgetWith(limits[other]), "endpoints %d and %d should have budgets of their own", f, other)
		}
	}
	for f, exp := range map[request.EndpointLimit]float64{
		systemTimeRate:                   49,
		swapPlaceOrderRate:               10,
		spotCancelOrderRate:              5,
		swapTradingScheduleRate:          2,
		activityListRate:                 10,
		createSignalRate:                 1,
		swapTradingRulesRate:             5,
		depositAddressRate:               1,
		swapMultiAssetsRulesRate:         5,
		subAccountTransferHistoryRate:    5,
		subAccountTransferableAmountRate: 5,
		subAccountAssetTransferRate:      5,
	} {
		assert.InDeltaf(t, exp, float64(limits[f].Limit()), 1e-9, "endpoint %d should allow %v requests a second", f, exp)
	}
}

func TestGenerateListenKey(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "method should be POST")
		assert.Equal(t, testAPIKey, r.Header.Get(keyHeader), "API key header should be sent")
		_, _ = io.WriteString(w, `{"listenKey":""}`)
	})
	_, err := ex.GenerateListenKey(t.Context())
	assert.ErrorIs(t, err, errListenKeyEmpty, "GenerateListenKey should reject an empty key")

	ex = newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"code":100413,"msg":"Incorrect apiKey"}`)
	})
	_, err = ex.GenerateListenKey(t.Context())
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "GenerateListenKey should return a refused key's error")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	key, err := e.GenerateListenKey(t.Context())
	require.NoError(t, err, "GenerateListenKey must not error")
	if mockTests {
		assert.Equal(t, "YOUR_LISTEN_KEY", key, "GenerateListenKey should return the documented key")
	} else {
		assert.NotEmpty(t, key, "GenerateListenKey should return a key")
	}
}

func Test_generateListenKey(t *testing.T) {
	t.Parallel()
	spot := newTestListenKeyServer(t)
	coinMargined := newTestListenKeyServer(t)
	ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	testRESTEndpoints(t, ex, spot.URL)
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestCoinMargined.String(), coinMargined.URL), "SetRunningURL must not error")

	key, err := ex.generateListenKey(t.Context(), exchange.RestCoinMargined)
	require.NoError(t, err, "generateListenKey must not error")
	assert.Equal(t, testListenKey+"-1", key, "generateListenKey should return the key the product's REST host created")
	created, _ := coinMargined.counts()
	assert.Equal(t, 1, created, "generateListenKey should create the key through the product's REST host")
	created, _ = spot.counts()
	assert.Zero(t, created, "generateListenKey should create no key through another REST host")

	coinMargined.setReplies(`{"code":0,"msg":"","data":{"listenKey":"enveloped"}}`, "")
	key, err = ex.generateListenKey(t.Context(), exchange.RestCoinMargined)
	require.NoError(t, err, "generateListenKey must not error for a key in a data member")
	assert.Equal(t, "enveloped", key, "generateListenKey should return a key sent in a data member")

	coinMargined.setReplies(`{"listenKey":""}`, "")
	_, err = ex.generateListenKey(t.Context(), exchange.RestCoinMargined)
	assert.ErrorIs(t, err, errListenKeyEmpty, "generateListenKey should reject an empty key")

	_, err = ex.generateListenKey(t.Context(), exchange.RestFutures)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "generateListenKey should error for an unset endpoint")
}

func TestExtendListenKey(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, e.ExtendListenKey(t.Context(), ""), errListenKeyEmpty, "ExtendListenKey should reject an empty key")

	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method, "method should be PUT")
		assert.Equal(t, "listenKey=k", r.URL.RawQuery, "the listen key should be sent")
		w.WriteHeader(http.StatusNoContent)
	})
	assert.ErrorIs(t, ex.ExtendListenKey(t.Context(), "k"), common.ErrNoResponse, "ExtendListenKey should treat the documented 204 as a failure")

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !mockTests {
		t.Skip("extending a listen key live needs a key from GenerateListenKey; TestGenerateListenKey covers it")
	}
	require.NoError(t, e.ExtendListenKey(t.Context(), "YOUR_LISTEN_KEY"), "ExtendListenKey must not error")
}

func Test_extendListenKey(t *testing.T) {
	t.Parallel()
	spot := newTestListenKeyServer(t)
	swap := newTestListenKeyServer(t)
	ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	testRESTEndpoints(t, ex, spot.URL)
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), swap.URL), "SetRunningURL must not error")

	require.NoError(t, ex.extendListenKey(t.Context(), exchange.RestUSDTMargined, testListenKey), "extendListenKey must not error")
	_, extended := swap.counts()
	assert.Equal(t, 1, extended, "extendListenKey should extend the key through the product's REST host")
	_, extended = spot.counts()
	assert.Zero(t, extended, "extendListenKey should extend no key through another REST host")

	swap.setReplies("", `{"code":100400,"msg":"listen key does not exist"}`)
	assert.ErrorIs(t, ex.extendListenKey(t.Context(), exchange.RestUSDTMargined, testListenKey), errAPIResponse, "extendListenKey should return a refused extension's error")
	assert.ErrorIs(t, ex.extendListenKey(t.Context(), exchange.RestUSDTMargined, ""), errListenKeyEmpty, "extendListenKey should reject an empty key")
	assert.ErrorIs(t, ex.extendListenKey(t.Context(), exchange.RestFutures, testListenKey), exchange.ErrEndpointPathNotFound, "extendListenKey should error for an unset endpoint")
	_, extended = swap.counts()
	assert.Equal(t, 2, extended, "extendListenKey should send nothing for an empty key or an unset endpoint")
}

func TestCloseListenKey(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, e.CloseListenKey(t.Context(), ""), errListenKeyEmpty, "CloseListenKey should reject an empty key")
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	if !mockTests {
		t.Skip("closing a live listen key would end other clients' account streams")
	}
	require.NoError(t, e.CloseListenKey(t.Context(), "YOUR_LISTEN_KEY"), "CloseListenKey must not error")
}

func TestPercentageUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		raw string
		exp Percentage
	}{
		{`"-1.05%"`, -1.05},
		{`"2.91%"`, 2.91},
		{`"-1.0300%"`, -1.03},
		{`"-1.05"`, -1.05},
		{`-1.05`, -1.05},
		{`"0%"`, 0},
		{`"-1.04%"`, -1.04},
		{`"-0.02"`, -0.02},
		{`2.5`, 2.5},
		{`""`, 0},
		{`null`, 0},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			t.Parallel()
			p := Percentage(99)
			require.NoError(t, json.Unmarshal([]byte(tc.raw), &p), "Unmarshal must not error")
			assert.Equal(t, tc.exp, p, "Unmarshal should decode the percentage shown")
			direct := Percentage(99)
			require.NoError(t, direct.UnmarshalJSON([]byte(tc.raw)), "UnmarshalJSON must not error")
			assert.Equal(t, tc.exp, direct, "UnmarshalJSON should decode the percentage shown")
		})
	}
	for _, raw := range []string{`"abc%"`, `"%"`, `"1.05%%"`, `"1.0x%"`, `true`} {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			var p Percentage
			assert.ErrorIs(t, json.Unmarshal([]byte(raw), &p), errInvalidPercentage, "Unmarshal should reject a value that is not a percentage")
			assert.ErrorIs(t, p.UnmarshalJSON([]byte(raw)), errInvalidPercentage, "UnmarshalJSON should reject a value that is not a percentage")
		})
	}
}

func TestPercentageFloat64(t *testing.T) {
	t.Parallel()
	assert.Equal(t, -1.05, Percentage(-1.05).Float64(), "Float64 should return the percentage")
}
