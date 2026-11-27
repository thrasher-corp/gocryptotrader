package binanceus

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

// Please supply your own keys here for due diligence testing
const canManipulateRealOrders = false

// apiCredentials holds the credentials used for due diligence testing; please supply your own
var apiCredentials = &accounts.Credentials{
	Key:    "",
	Secret: "",
}

var (
	e *Exchange

	// spotTradablePair is an enabled and active pair used to test endpoints
	spotTradablePair currency.Pair
)

// newTestServerExchange returns an exchange named after the test whose REST requests go to handler, signed with
// placeholder credentials and without rate limiting
func newTestServerExchange(t *testing.T, handler http.HandlerFunc) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "test-api-key", Secret: "test-api-secret"})
	server := httptest.NewTestServer(t, capRequests(t, handler))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpotSupplementary.String(), server.URL), "SetRunningURL must not error")
	return ex
}

// capRequests fails a test whose fake venue is asked for more requests than any test needs, as a pager that never stops
// would ask, rather than letting the test run until go test's timeout
func capRequests(t *testing.T, handler http.HandlerFunc) http.HandlerFunc {
	t.Helper()
	var requests atomic.Int32
	return func(w http.ResponseWriter, r *http.Request) {
		if n := requests.Add(1); n > 100 {
			assert.Failf(t, "too many requests", "%s should not be requested %d times", r.URL.Path, n)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		handler(w, r)
	}
}

// writeResponse writes a fake venue's response body
func writeResponse(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	_, err := w.Write([]byte(body))
	assert.NoError(t, err, "Write should not error")
}

// skipIfCredentialsUnset skips a live test of an authenticated endpoint when no credentials are set; mock tests run it
// against the recorded responses
func skipIfCredentialsUnset(t *testing.T) {
	t.Helper()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
}

// skipUnlessLive skips a wrapper test's live case, which runs on the package's exchange: in mock mode, as the recorded
// responses do not cover the wrapper's requests, and in live mode without the credentials or the permission to
// manipulate orders the case needs
func skipUnlessLive(t *testing.T, authenticated bool, canManipulateOrders ...bool) {
	t.Helper()
	if mockTests {
		t.Skip("the recorded responses do not cover the wrapper's requests")
	}
	if authenticated {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateOrders...)
	}
	ensureTradablePairs(t)
}

func TestSendHTTPRequest(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "SendHTTPRequest should send a GET request")
		assert.Equal(t, "/api/v3/avgPrice", r.URL.Path, "SendHTTPRequest should request the path")
		assert.Equal(t, "symbol=BTCUSDT", r.URL.RawQuery, "SendHTTPRequest should send the query unchanged")
		assert.Empty(t, r.Header.Get("X-MBX-APIKEY"), "SendHTTPRequest should not send the API key")
		writeResponse(t, w, `{"mins":5,"price":"73672.97372917","closeTime":1794390776701}`)
	})
	var result map[string]any
	require.NoError(t, ex.SendHTTPRequest(t.Context(), exchange.RestSpotSupplementary, "/api/v3/avgPrice?symbol=BTCUSDT", defaultRate, &result), "SendHTTPRequest must not error")
	assert.Equal(t, map[string]any{"mins": float64(5), "price": "73672.97372917", "closeTime": float64(1794390776701)}, result, "SendHTTPRequest should decode the response")

	t.Run("through the shared exchange", func(t *testing.T) {
		t.Parallel()
		var result map[string]any
		require.NoError(t, e.SendHTTPRequest(t.Context(), exchange.RestSpotSupplementary, "/api/v3/ping", defaultRate, &result), "SendHTTPRequest must not error")
		assert.Empty(t, result, "SendHTTPRequest should decode the ping's empty object")
	})
}

func TestSendRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		statusCode int
		body       string
		errs       []error
		notErrs    []error
		msg        string
	}{
		{
			name:       "mapped invalid symbol code",
			statusCode: http.StatusBadRequest,
			body:       `{"code":-1121,"msg":"Invalid symbol."}`,
			errs:       []error{errAPIResponse, currency.ErrPairNotFound, request.ErrBadStatus},
			msg:        "code -1121 msg Invalid symbol.",
		},
		{
			name:       "mapped unauthorised code",
			statusCode: http.StatusUnauthorized,
			body:       `{"code":-1002,"msg":"You are not authorized to execute this request."}`,
			errs:       []error{errAPIResponse, request.ErrAuthRequestFailed, request.ErrBadStatus},
			msg:        "code -1002 msg You are not authorized to execute this request.",
		},
		{
			name:       "mapped unknown order code",
			statusCode: http.StatusBadRequest,
			body:       `{"code":-2013,"msg":"Order does not exist."}`,
			errs:       []error{errAPIResponse, order.ErrOrderNotFound, request.ErrBadStatus},
			msg:        "code -2013 msg Order does not exist.",
		},
		{
			name:       "unmapped code",
			statusCode: http.StatusBadRequest,
			body:       `{"code":-2011,"msg":"Unknown order sent."}`,
			errs:       []error{errAPIResponse, request.ErrBadStatus},
			notErrs:    []error{currency.ErrPairNotFound, request.ErrAuthRequestFailed, order.ErrOrderNotFound},
			msg:        "code -2011 msg Unknown order sent.",
		},
		{
			name:       "negative code with a success status",
			statusCode: http.StatusOK,
			body:       `{"code":-1121,"msg":"Invalid symbol."}`,
			errs:       []error{errAPIResponse, currency.ErrPairNotFound},
			notErrs:    []error{request.ErrBadStatus},
			msg:        "code -1121 msg Invalid symbol.",
		},
		{
			name:       "success flag of false",
			statusCode: http.StatusOK,
			body:       `{"success":false,"msg":"Sub-account does not exist."}`,
			errs:       []error{errAPIResponse},
			notErrs:    []error{request.ErrBadStatus},
			msg:        "code 0 msg Sub-account does not exist.",
		},
		{
			name:       "envelope with a success flag of false",
			statusCode: http.StatusOK,
			body:       `{"code":"100001","message":"Staking is not available.","success":false}`,
			errs:       []error{errAPIResponse},
			notErrs:    []error{request.ErrBadStatus},
			msg:        "code 100001 msg Staking is not available.",
		},
		{
			name:       "error status without an error object",
			statusCode: http.StatusInternalServerError,
			body:       `{"someField":"value"}`,
			errs:       []error{request.ErrBadStatus},
			notErrs:    []error{errAPIResponse},
		},
		{
			name:       "error status without a JSON body",
			statusCode: http.StatusBadGateway,
			body:       `<html><body>Bad Gateway</body></html>`,
			errs:       []error{request.ErrBadStatus},
			notErrs:    []error{errAPIResponse},
		},
		{
			name:       "null body",
			statusCode: http.StatusOK,
			body:       `null`,
			errs:       []error{common.ErrNoResponse},
			notErrs:    []error{errAPIResponse},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.statusCode)
				writeResponse(t, w, tc.body)
			})
			var result *ErrorResponse
			err := ex.SendHTTPRequest(t.Context(), exchange.RestSpotSupplementary, "/api/v3/time", defaultRate, &result)
			for _, want := range tc.errs {
				require.ErrorIs(t, err, want, "SendHTTPRequest must return the expected error")
			}
			for _, unwanted := range tc.notErrs {
				assert.NotErrorIs(t, err, unwanted, "SendHTTPRequest should not return an unrelated error")
			}
			if tc.msg != "" {
				assert.ErrorContains(t, err, tc.msg, "SendHTTPRequest should report the code and message")
			}
		})
	}

	for _, body := range []string{
		`{"code":200,"msg":"success"}`,
		`{"msg":"Normal","success":true}`,
		`{"code":"000000","message":"success","data":{"result":"SUCCESS"},"success":true}`,
	} {
		t.Run(body, func(t *testing.T) {
			t.Parallel()
			ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
				writeResponse(t, w, body)
			})
			var result *ErrorResponse
			require.NoError(t, ex.SendHTTPRequest(t.Context(), exchange.RestSpotSupplementary, "/sapi/v1/test", defaultRate, &result), "SendHTTPRequest must not error")
			require.NotNil(t, result, "SendHTTPRequest must decode the response")
		})
	}

	t.Run("an item that cannot be built", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, func(http.ResponseWriter, *http.Request) {
			assert.Fail(t, "a request whose item cannot be built should not be sent")
		})
		err := ex.sendRequest(t.Context(), defaultRate, request.UnauthenticatedRequest, nil, func() (*request.Item, error) {
			return nil, errAPIResponse
		})
		assert.ErrorIs(t, err, errAPIResponse, "sendRequest should return the item's error")
	})

	t.Run("a nil result", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			body string
			err  error
		}{
			{name: "empty object", body: `{}`},
			{name: "empty array", body: `[]`},
			{name: "null", body: `null`},
			{name: "data", body: `{"listenKey":"pqia91ma19a5s61cv6a81va65sdf19v8a65a1a5s61cv6a81va65sdf19v8a65a1"}`, err: errUnexpectedResponseData},
			{name: "error object", body: `{"code":-1121,"msg":"Invalid symbol."}`, err: errAPIResponse},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
					writeResponse(t, w, tc.body)
				})
				endpoint, err := ex.API.Endpoints.GetURL(exchange.RestSpotSupplementary)
				require.NoError(t, err, "GetURL must not error")
				assert.ErrorIs(t, ex.sendRequest(t.Context(), defaultRate, request.UnauthenticatedRequest, nil, func() (*request.Item, error) {
					return &request.Item{Method: http.MethodGet, Path: endpoint + "/api/v3/ping"}, nil
				}), tc.err, "sendRequest should return the expected error when no result is wanted")
			})
		}
	})
}

func TestCheckNoData(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		raw  string
		err  error
		msg  string
	}{
		{name: "no body"},
		{name: "null", raw: `null`},
		{name: "empty object", raw: `{}`},
		{name: "empty object holding whitespace", raw: "{ \n}"},
		{name: "empty array", raw: `[]`},
		{name: "object", raw: `{"listenKey":"pqia91ma19a5s61cv6a81va65sdf19v8a65a1a5s61cv6a81va65sdf19v8a65a1"}`, err: errUnexpectedResponseData},
		{name: "object holding a null", raw: `{"listenKey":null}`, err: errUnexpectedResponseData},
		{name: "array", raw: `[1]`, err: errUnexpectedResponseData},
		{name: "array holding an empty object", raw: `[{}]`, err: errUnexpectedResponseData},
		{name: "array holding a null", raw: `[null]`, err: errUnexpectedResponseData},
		{name: "multibyte text counted in bytes", raw: `{"fee":"€"}`, err: errUnexpectedResponseData, msg: "13 bytes"},
		{name: "string", raw: `"ok"`, err: errUnexpectedResponseData},
		{name: "empty string", raw: `""`, err: errUnexpectedResponseData},
		{name: "zero", raw: `0`, err: errUnexpectedResponseData},
		{name: "false", raw: `false`, err: errUnexpectedResponseData},
		{name: "true", raw: `true`, err: errUnexpectedResponseData},
		{name: "not JSON", raw: `{`, err: errUnexpectedResponseData},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := checkNoData(json.RawMessage(tc.raw))
			assert.ErrorIs(t, err, tc.err, "checkNoData should return the expected error")
			if tc.msg != "" {
				assert.ErrorContains(t, err, tc.msg, "checkNoData should report the response's size in bytes")
			}
		})
	}
}

func TestSendAuthHTTPRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		method     string
		recvWindow string
		exp        string
	}{
		{name: "default receive window", method: http.MethodDelete, exp: "5000"},
		{name: "receive window set by the caller", method: http.MethodPost, recvWindow: "10000", exp: "10000"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tc.method, r.Method, "SendAuthHTTPRequest should send the method")
				assert.Equal(t, "/api/v3/order", r.URL.Path, "SendAuthHTTPRequest should request the path")
				assert.Equal(t, "test-api-key", r.Header.Get("X-MBX-APIKEY"), "SendAuthHTTPRequest should send the API key")
				payload, signature, found := strings.Cut(r.URL.RawQuery, "&signature=")
				assert.True(t, found, "SendAuthHTTPRequest should send the signature last")
				mac := hmac.New(sha256.New, []byte("test-api-secret"))
				_, err := mac.Write([]byte(payload))
				assert.NoError(t, err, "Write should not error")
				assert.Equal(t, hex.EncodeToString(mac.Sum(nil)), signature, "SendAuthHTTPRequest should sign the encoded parameters with HMAC-SHA256")
				// Binance.US signs the percent-encoded payload, so every value needing encoding is encoded where it is signed
				for _, encoded := range []string{"newClientOrderId=a%2Bb%2Fc+d", "email=sub%40example.com", "symbols=%5B%22BTCUSDT%22%2C%22ETHUSDT%22%5D"} {
					assert.Contains(t, payload, encoded, "SendAuthHTTPRequest should sign and send the percent-encoded parameters")
				}
				query := r.URL.Query()
				assert.Equal(t, "a+b/c d", query.Get("newClientOrderId"), "SendAuthHTTPRequest should send the parameters")
				assert.Equal(t, tc.exp, query.Get("recvWindow"), "SendAuthHTTPRequest should send the receive window")
				timestamp, err := strconv.ParseInt(query.Get("timestamp"), 10, 64)
				assert.NoError(t, err, "ParseInt should not error")
				assert.WithinDuration(t, time.Now(), time.UnixMilli(timestamp), time.Minute, "SendAuthHTTPRequest should send the current timestamp")
				writeResponse(t, w, `{"code":200,"msg":"success"}`)
			})
			params := url.Values{}
			params.Set("newClientOrderId", "a+b/c d")
			params.Set("email", "sub@example.com")
			params.Set("symbols", `["BTCUSDT","ETHUSDT"]`)
			if tc.recvWindow != "" {
				params.Set("recvWindow", tc.recvWindow)
			}
			var result *ErrorResponse
			require.NoError(t, ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpotSupplementary, tc.method, "/api/v3/order", params, defaultRate, &result), "SendAuthHTTPRequest must not error")
			assert.Equal(t, &ErrorResponse{Code: 200, Message: "success"}, result, "SendAuthHTTPRequest should decode the response")
		})
	}

	t.Run("through the shared exchange", func(t *testing.T) {
		t.Parallel()
		skipIfCredentialsUnset(t)
		var resp *AccountResponse
		require.NoError(t, e.SendAuthHTTPRequest(t.Context(), exchange.RestSpotSupplementary, http.MethodGet, "/api/v3/account", nil, accountInformationRate, &resp), "SendAuthHTTPRequest must not error")
		require.NotNil(t, resp, "SendAuthHTTPRequest must decode the response")
		assert.NotEmpty(t, resp.AccountType, "SendAuthHTTPRequest should decode the account")
	})

	t.Run("a retry is signed with a new timestamp", func(t *testing.T) {
		t.Parallel()
		var attempts atomic.Int32
		var timestamps [2]string
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			n := attempts.Add(1)
			if n <= 2 {
				timestamps[n-1] = r.URL.Query().Get("timestamp")
			}
			if n == 1 {
				time.Sleep(2 * time.Millisecond)
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			writeResponse(t, w, `{}`)
		})
		require.NoError(t, ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpotSupplementary, http.MethodGet, "/api/v3/account", nil, defaultRate, nil), "SendAuthHTTPRequest must not error once a retry succeeds")
		require.Equal(t, int32(2), attempts.Load(), "SendAuthHTTPRequest must retry the rate limited request once")
		assert.NotEqual(t, timestamps[0], timestamps[1], "SendAuthHTTPRequest should sign a retry with a new timestamp")
	})
}

func TestSendAPIKeyHTTPRequest(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method, "SendAPIKeyHTTPRequest should send the method")
		assert.Equal(t, "test-api-key", r.Header.Get("X-MBX-APIKEY"), "SendAPIKeyHTTPRequest should send the API key")
		assert.Equal(t, "listenKey=pqia91ma19a5s61cv6a81va65sdf19v8a65a1a5s61cv6a81va65sdf19v8a65a1", r.URL.RawQuery, "SendAPIKeyHTTPRequest should send neither a timestamp nor a signature")
		writeResponse(t, w, `{}`)
	})
	var result *ErrorResponse
	require.NoError(t, ex.SendAPIKeyHTTPRequest(t.Context(), exchange.RestSpotSupplementary, http.MethodPut, "/api/v3/userDataStream?listenKey=pqia91ma19a5s61cv6a81va65sdf19v8a65a1a5s61cv6a81va65sdf19v8a65a1", defaultRate, &result), "SendAPIKeyHTTPRequest must not error")
	assert.Equal(t, &ErrorResponse{}, result, "SendAPIKeyHTTPRequest should decode the response")

	t.Run("through the shared exchange", func(t *testing.T) {
		t.Parallel()
		skipIfCredentialsUnset(t)
		var resp *ListenKeyResponse
		require.NoError(t, e.SendAPIKeyHTTPRequest(t.Context(), exchange.RestSpotSupplementary, http.MethodPost, "/api/v3/userDataStream", defaultRate, &resp), "SendAPIKeyHTTPRequest must not error")
		require.NotNil(t, resp, "SendAPIKeyHTTPRequest must decode the response")
		assert.NotEmpty(t, resp.ListenKey, "SendAPIKeyHTTPRequest should decode the listen key")
	})
}

func TestCheckErrorResponse(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		body string
		exp  *APIError
	}{
		{name: "empty body", body: ``},
		{name: "array", body: `[{"code":-1}]`},
		{name: "not an error object", body: `{"code":"x"}`},
		{name: "success code", body: `{"code":200,"msg":"success"}`},
		{name: "success envelope", body: `{"code":"000000","message":"success","success":true}`},
		{name: "negative code", body: `{"code":-1100,"msg":"Illegal characters found in a parameter.","data":{"a":1}}`, exp: &APIError{Code: -1100, Message: "Illegal characters found in a parameter.", Data: json.RawMessage(`{"a":1}`)}},
		{name: "success flag of false", body: `{"success":false,"msg":"failed"}`, exp: &APIError{Message: "failed"}},
		{name: "envelope message", body: `{"code":"100001","message":"failed","success":false}`, exp: &APIError{Code: 100001, Message: "failed"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := checkErrorResponse(json.RawMessage(tc.body))
			if tc.exp == nil {
				assert.NoError(t, err, "checkErrorResponse should not return an error")
				return
			}
			require.ErrorIs(t, err, errAPIResponse, "checkErrorResponse must return an API error")
			assert.Equal(t, tc.exp, err, "checkErrorResponse should return the code, message and data")
		})
	}
}

func TestAPIErrorError(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "Binance.US API error: order not found: code -2013 msg Order does not exist.", (&APIError{Code: -2013, Message: "Order does not exist."}).Error(), "Error should name the mapped error, code and message")
	assert.Equal(t, "Binance.US API error: code -2021 msg Order cancel-replace partially failed. data {\"cancelResult\":\"FAILURE\"}", (&APIError{Code: -2021, Message: "Order cancel-replace partially failed.", Data: json.RawMessage(`{"cancelResult":"FAILURE"}`)}).Error(), "Error should add the data")
}

func TestAPIErrorUnwrap(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []error{errAPIResponse, order.ErrOrderNotFound}, (&APIError{Code: -2013}).Unwrap(), "Unwrap should return the mapped error")
	assert.Equal(t, []error{errAPIResponse}, (&APIError{Code: -2011}).Unwrap(), "Unwrap should return the API error alone for an unmapped code")
}

func TestSetTimeRangeParams(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1794390000000), time.UnixMilli(1794390834232)
	for _, tc := range []struct {
		name       string
		start, end time.Time
		exp        url.Values
		err        error
	}{
		{name: "neither", exp: url.Values{}},
		{name: "start only", start: start, exp: url.Values{"startTime": {"1794390000000"}}},
		{name: "end only", end: end, exp: url.Values{"endTime": {"1794390834232"}}},
		{name: "both", start: start, end: end, exp: url.Values{"startTime": {"1794390000000"}, "endTime": {"1794390834232"}}},
		{name: "equal", start: end, end: end, exp: url.Values{"startTime": {"1794390834232"}, "endTime": {"1794390834232"}}},
		{name: "reversed", start: end, end: start, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params := url.Values{}
			err := setTimeRangeParams(params, tc.start, tc.end)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err, "setTimeRangeParams must reject a reversed range")
				assert.Empty(t, params, "setTimeRangeParams should set nothing for a reversed range")
				return
			}
			require.NoError(t, err, "setTimeRangeParams must not error")
			assert.Equal(t, tc.exp, params, "setTimeRangeParams should set the bounds in milliseconds")
		})
	}
}

func TestFormatSymbol(t *testing.T) {
	t.Parallel()
	_, err := e.formatSymbol(currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "formatSymbol must refuse an empty pair")
	symbol, err := e.formatSymbol(currency.NewPairWithDelimiter("btc", "usdt", "-"))
	require.NoError(t, err, "formatSymbol must not error")
	assert.Equal(t, "BTCUSDT", symbol, "formatSymbol should return the request formatted symbol")
}

func TestSetSymbolParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		pairs currency.Pairs
		exp   url.Values
		err   error
	}{
		{name: "none", exp: url.Values{}},
		{name: "one", pairs: currency.Pairs{currency.NewBTCUSDT()}, exp: url.Values{"symbol": {"BTCUSDT"}}},
		{name: "several", pairs: currency.Pairs{currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USD)}, exp: url.Values{"symbols": {`["BTCUSDT","ETHUSD"]`}}},
		{name: "empty pair", pairs: currency.Pairs{currency.NewBTCUSDT(), currency.EMPTYPAIR}, err: currency.ErrCurrencyPairEmpty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params := url.Values{}
			err := e.setSymbolParams(params, tc.pairs)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err, "setSymbolParams must refuse an empty pair")
				return
			}
			require.NoError(t, err, "setSymbolParams must not error")
			assert.Equal(t, tc.exp, params, "setSymbolParams should set symbol or symbols")
		})
	}
}

func TestFormatFloat(t *testing.T) {
	t.Parallel()
	for in, exp := range map[float64]string{1.9: "1.9", 0.00001234: "0.00001234", 1e-8: "0.00000001", 25: "25", 123456789.123: "123456789.123"} {
		t.Run(exp, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, exp, formatFloat(in), "formatFloat should format %v in plain notation", in)
		})
	}
}

func TestSendPublicRequest(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v3/time", r.URL.Path, "a REST request should go to the path")
		writeResponse(t, w, `{"serverTime":1794466395877}`)
	})
	var resp *ServerTimeResponse
	require.NoError(t, ex.sendPublicRequest(t.Context(), false, "/api/v3/time", "time", nil, defaultRate, &resp), "sendPublicRequest must not error over REST")
	assert.Equal(t, time.UnixMilli(1794466395877), resp.ServerTime.Time(), "sendPublicRequest should decode the REST response")

	resp = nil
	ws := newWebsocketAPITestExchange(t, false)
	require.NoError(t, ws.sendPublicRequest(t.Context(), true, "/api/v3/time", "time", nil, defaultRate, &resp), "sendPublicRequest must not error over the WebSocket API")
	assert.NotNil(t, resp, "sendPublicRequest should decode the WebSocket API result")
}

func TestSendSignedRequest(t *testing.T) {
	t.Parallel()
	ex := signedRequestExchange(t, http.MethodGet, "/api/v3/account", url.Values{}, `{"uid":7}`)
	var resp *AccountResponse
	require.NoError(t, ex.sendSignedRequest(t.Context(), false, http.MethodGet, "/api/v3/account", "account.status", nil, accountInformationRate, &resp), "sendSignedRequest must not error over REST")
	assert.Equal(t, uint64(7), resp.UID, "sendSignedRequest should decode the REST response")

	resp = nil
	ws := newWebsocketAPITestExchange(t, true)
	require.NoError(t, ws.sendSignedRequest(t.Context(), true, http.MethodGet, "/api/v3/account", "account.status", nil, accountInformationRate, &resp), "sendSignedRequest must not error over the WebSocket API")
	assert.NotNil(t, resp, "sendSignedRequest should decode the WebSocket API result")
}

func TestTimedOutRequestRetries(t *testing.T) {
	t.Parallel()
	newOrder := &NewOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Type: OrderTypeLimit, TimeInForce: TimeInForceGoodTillCancel, Quantity: 0.001, Price: 70000}
	custodialTransfer := &CustodialAssetTransferRequest{Rail: custodialTestRail, Asset: currency.BTC, Amount: 1}
	for _, tc := range []struct {
		name    string
		retried bool
		call    func(context.Context, *Exchange) error
	}{
		{name: "NewOrder", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.NewOrder(ctx, newOrder)
			return err
		}},
		{name: "CancelExistingOrderAndSendNewOrder", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CancelExistingOrderAndSendNewOrder(ctx, &CancelReplaceOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideBuy, Type: OrderTypeLimit, CancelReplaceMode: CancelReplaceModeAllowFailure, CancelOrderID: 9, Quantity: 0.001, Price: 70000})
			return err
		}},
		{name: "NewOneCancelsTheOtherOrder", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.NewOneCancelsTheOtherOrder(ctx, &OneCancelsTheOtherOrderRequest{Symbol: currency.NewBTCUSDT(), Side: SideSell, Quantity: 0.002, Price: 80000, StopPrice: 70000})
			return err
		}},
		{name: "CustodialNewOrder", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CustodialNewOrder(ctx, &CustodialNewOrderRequest{Rail: custodialTestRail, Symbol: currency.NewBTCUSD(), Side: SideBuy, Type: OrderTypeMarket, Quantity: 0.5})
			return err
		}},
		{name: "CustodialNewOneCancelsTheOtherOrder", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CustodialNewOneCancelsTheOtherOrder(ctx, &CustodialNewOneCancelsTheOtherOrderRequest{Rail: custodialTestRail, Symbol: currency.NewPair(currency.LTC, currency.BTC), Side: SideBuy, Quantity: 0.624363, Price: 0.036435, StopPrice: 0.960664})
			return err
		}},
		{name: "TransferFromExchangeWallet", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.TransferFromExchangeWallet(ctx, custodialTransfer)
			return err
		}},
		{name: "TransferFromCustodian", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.TransferFromCustodian(ctx, custodialTransfer)
			return err
		}},
		{name: "UndoCustodialTransfer", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.UndoCustodialTransfer(ctx, custodialTestRail, "2022092709232937542366")
			return err
		}},
		{name: "ExecuteCreditLineTransfer", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.ExecuteCreditLineTransfer(ctx, &CreditLineTransferRequest{TransferType: CreditLineTransferTypeTransferOut, TransferAssetType: currency.BTC, Quantity: 0.00012345})
			return err
		}},
		{name: "ConvertDust", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.ConvertDust(ctx, []currency.Code{currency.BTC, currency.ETH}, currency.BNB)
			return err
		}},
		{name: "ExecuteSubAccountTransfer", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.ExecuteSubAccountTransfer(ctx, &SubAccountTransferRequest{FromEmail: "master@test.com", ToEmail: "sub@test.com", Asset: currency.BTC, Amount: 0.25})
			return err
		}},
		{name: "RequestForQuote", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.RequestForQuote(ctx, &QuoteRequest{FromCoin: currency.BTC, ToCoin: currency.USDT, RequestCoin: currency.BTC, RequestAmount: 1})
			return err
		}},
		{name: "PlaceOverTheCounterTradeOrder", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.PlaceOverTheCounterTradeOrder(ctx, "15848701022")
			return err
		}},
		{name: "StakeAsset", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.StakeAsset(ctx, &StakeAssetRequest{StakingAsset: currency.BNB, Amount: 0.01})
			return err
		}},
		{name: "UnstakeAsset", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.UnstakeAsset(ctx, currency.BNB, 0.01)
			return err
		}},
		{name: "WithdrawFiat", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.WithdrawFiat(ctx, &WithdrawFiatRequest{PaymentMethod: PaymentMethodBitGo, PaymentAccount: "4a992541-c12d-4cca-bbd6-df637f801526", Amount: 100})
			return err
		}},
		{name: "WithdrawCrypto", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.WithdrawCrypto(ctx, &WithdrawRequest{Coin: currency.BTC, Network: "BTC", Address: core.BitcoinDonationAddress, Amount: 0.001})
			return err
		}},
		{name: "GetAccount", retried: true, call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.GetAccount(ctx)
			return err
		}},
		{name: "NewOrderTest", retried: true, call: func(ctx context.Context, ex *Exchange) error {
			return ex.NewOrderTest(ctx, newOrder)
		}},
		{name: "CreateUserDataStream", retried: true, call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CreateUserDataStream(ctx)
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int64
			ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
				// The server notices the client giving up, which ends the request's context, only once the body is read
				_, err := io.Copy(io.Discard, r.Body)
				assert.NoError(t, err, "reading the request body should not error")
				if requests.Add(1) == 1 {
					select {
					case <-r.Context().Done():
					case <-time.After(time.Second):
					}
					return
				}
				w.WriteHeader(http.StatusBadRequest)
			})
			require.NoError(t, ex.SetHTTPClientTimeout(250*time.Millisecond), "SetHTTPClientTimeout must not error")
			err := tc.call(t.Context(), ex)
			if tc.retried {
				assert.ErrorIsf(t, err, request.ErrBadStatus, "%s should return its retry's reply", tc.name)
				assert.Equalf(t, int64(2), requests.Load(), "%s should send a timed-out request again", tc.name)
				return
			}
			assert.ErrorIsf(t, err, context.DeadlineExceeded, "%s should return the timeout", tc.name)
			assert.Equalf(t, int64(1), requests.Load(), "%s should send a timed-out request once", tc.name)
		})
	}
}
