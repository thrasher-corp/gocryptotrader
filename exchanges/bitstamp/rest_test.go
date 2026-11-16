package bitstamp

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
	"regexp"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Please supply your own keys in live_test.go for due diligence testing
const (
	canManipulateRealOrders = false
	// useTestNet directs live REST tests at the sandbox, which requires sandbox API credentials
	useTestNet = false
)

var (
	e             *Exchange
	spotPair      = currency.NewBTCUSD()
	perpetualPair = currency.NewPair(currency.BTC, currency.NewCode("USD-PERP"))
	// unlistedCurrency is a currency Bitstamp does not list, so live requests naming it are refused
	unlistedCurrency = currency.NewCode("GCTUNLISTED")
	// invalidIBAN has the check digits 00, which no IBAN has
	invalidIBAN = "IT00X0542811101000000123456"
)

// parseRFC3339 parses a time as the JSON decoder does, since a zero offset takes the local zone when that zone is UTC
func parseRFC3339(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, s)
	require.NoError(t, err, "Parse must not error")
	return tm
}

// skipIfLiveWithoutCredentials skips authenticated tests when live testing without credentials, or without permission
// to manipulate orders when canManipulateOrders is supplied
func skipIfLiveWithoutCredentials(t *testing.T, canManipulateOrders ...bool) {
	t.Helper()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateOrders...)
	}
}

// refusedRegardlessOfRequest matches the replies the API documents for refusing a request whatever it asks: failed
// authentication, missing permissions, unverified or restricted accounts and API keys, rate limits, unavailable or failing
// services, and the reply to an unknown endpoint
var refusedRegardlessOfRequest = regexp.MustCompile(`(?i)\bAPI(?:00\d\d|500[5-7]|5011|5502)\b|\b(?:400\.(?:002|003|067|068)|(?:403|405|410|500)\.\d{3})\b|permission|not available for your account|not allowed to withdraw|sub account API keys|user not verified|temporarily|currently (?:disabled|unavailable)|try (?:again )?later|something went wrong|reach out to support|contact support|: Not found\.$`)

// malformedRequest matches the replies the API documents for a request without a parameter it requires, or with a
// parameter it cannot read or an amount with more decimal places than it takes
var malformedRequest = regexp.MustCompile(`(?i)\bmissing\b[^;]*\bparam|parameter\(s\) is (?:missing|malformed)|this field is required|parameter is required|must set this field|enter a number|decimal places`)

// requireRefusedOnMerits checks that the API refused a live request which was built to be refused for what it asks
func requireRefusedOnMerits(t *testing.T, err error) {
	t.Helper()
	require.ErrorIs(t, err, errAPIResponse, "the API must refuse the request")
	assert.NotRegexp(t, refusedRegardlessOfRequest, err.Error(), "the API should refuse the request for what it asks")
	assert.NotRegexp(t, malformedRequest, err.Error(), "the request should carry every parameter the API requires in a form it reads")
}

// newLiveTestInstance returns an exchange which sends its REST requests where the package exchange does
func newLiveTestInstance(t *testing.T) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	endpoint, err := e.API.Endpoints.GetURL(exchange.RestSpot)
	require.NoError(t, err, "GetURL must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), endpoint), "SetRunningURL must not error")
	return ex
}

// newPlaceholderCredentialsTestInstance returns a live exchange whose requests are signed for an API key no account holds
func newPlaceholderCredentialsTestInstance(t *testing.T) *Exchange {
	t.Helper()
	ex := newLiveTestInstance(t)
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "placeholderplaceholderplaceholde", Secret: "placeholderplaceholderplaceholde"})
	return ex
}

// requireUnknownAPIKey checks that the API refused a request for its API key, which the API checks only once the
// endpoint exists, as it answers an unknown endpoint with "Not found." first
func requireUnknownAPIKey(t *testing.T, err error) {
	t.Helper()
	require.ErrorIs(t, err, errAPIResponse, "the API must refuse the request")
	assert.ErrorContains(t, err, "API0001 API key not found", "the API should refuse the request for its unknown API key")
}

// maxTestServerRequests caps the requests a test server answers. It is above the 231 requests the longest pager test
// makes, so a pager which never stops fails its test instead of hanging it
const maxTestServerRequests = 250

// newTestExchange returns an exchange whose REST requests are served by handler
func newTestExchange(t *testing.T, handler http.HandlerFunc) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "key", Secret: "secret"})
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	var requests atomic.Int64
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assert.LessOrEqualf(t, requests.Add(1), int64(maxTestServerRequests), "test server should receive at most %d requests", maxTestServerRequests) {
			http.Error(w, "too many requests", http.StatusBadRequest)
			return
		}
		handler(w, r)
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/api"), "SetRunningURL must not error")
	return ex
}

// preciseNumber returns a PreciseNumber holding a decimal string
func preciseNumber(t *testing.T, s string) types.PreciseNumber {
	t.Helper()
	n, err := types.NewPreciseNumberFromString(s)
	require.NoErrorf(t, err, "NewPreciseNumberFromString must not error for %s", s)
	return n
}

// newResponseTestExchange returns an exchange whose REST requests are all answered with the same status and body
func newResponseTestExchange(t *testing.T, status int, body string) *Exchange {
	t.Helper()
	return newTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Write should not error")
	})
}

func TestSendAuthenticatedHTTPRequest(t *testing.T) {
	t.Parallel()
	t.Run("account balances", func(t *testing.T) {
		t.Parallel()
		skipIfLiveWithoutCredentials(t)
		var resp []AccountBalanceResponse
		require.NoError(t, e.SendAuthenticatedHTTPRequest(t.Context(), exchange.RestSpot, http.MethodPost, "/v2/account_balances/", nil, nil, &resp), "SendAuthenticatedHTTPRequest must not error")
		assert.NotEmpty(t, resp, "SendAuthenticatedHTTPRequest should decode the balances")
	})
	t.Run("signature", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name                string
			method              string
			params              url.Values
			body                any
			subAccount          string
			expectedContentType string
			expectedBody        string
			expectedQuery       string
		}{
			{name: "empty POST", method: http.MethodPost},
			{name: "empty form", method: http.MethodPost, body: url.Values{}},
			{name: "form", method: http.MethodPost, body: url.Values{"offset": {"1"}}, expectedContentType: formContentType, expectedBody: "offset=1"},
			{name: "JSON", method: http.MethodPost, body: &closePositionBody{PositionID: "1234567890"}, expectedContentType: jsonContentType, expectedBody: `{"position_id":"1234567890"}`},
			{name: "query", method: http.MethodGet, params: url.Values{"limit": {"10"}, "sort": {"asc"}}, expectedQuery: "limit=10&sort=asc"},
			{name: "sub account", method: http.MethodPost, subAccount: "990129"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var captured *http.Request
				var capturedBody []byte
				ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
					captured = r
					var err error
					capturedBody, err = io.ReadAll(r.Body)
					assert.NoError(t, err, "ReadAll should not error")
					_, err = w.Write([]byte(`{"status":"ok"}`))
					assert.NoError(t, err, "Write should not error")
				})
				creds := &accounts.Credentials{Key: "key", Secret: "secret", SubAccount: tc.subAccount}
				ex.SetCredentials(creds)
				err := ex.SendAuthenticatedHTTPRequest(t.Context(), exchange.RestSpot, tc.method, "/v2/user_transactions/", tc.params, tc.body, nil)
				require.NoError(t, err, "SendAuthenticatedHTTPRequest must not error")
				require.NotNil(t, captured, "request must be captured")

				assert.Equal(t, tc.method, captured.Method, "method should be correct")
				assert.Equal(t, tc.expectedBody, string(capturedBody), "body should be correct")
				assert.Equal(t, tc.expectedQuery, captured.URL.RawQuery, "query should be correct")
				assert.Equal(t, tc.expectedContentType, captured.Header.Get("Content-Type"), "Content-Type should only be set with a body")
				assert.Equal(t, "BITSTAMP key", captured.Header.Get("X-Auth"), "X-Auth should be correct")
				assert.Equal(t, "v2", captured.Header.Get("X-Auth-Version"), "X-Auth-Version should be correct")
				assert.Equal(t, tc.subAccount, captured.Header.Get("X-Auth-Subaccount-Id"), "X-Auth-Subaccount-Id should be correct")
				nonce := captured.Header.Get("X-Auth-Nonce")
				assert.Len(t, nonce, 36, "X-Auth-Nonce should be a UUID")
				timestamp := captured.Header.Get("X-Auth-Timestamp")
				ms, err := strconv.ParseInt(timestamp, 10, 64)
				require.NoError(t, err, "X-Auth-Timestamp must be an integer")
				assert.WithinDuration(t, time.Now(), time.UnixMilli(ms), time.Minute, "X-Auth-Timestamp should be in milliseconds")

				query := ""
				if tc.expectedQuery != "" {
					query = "?" + tc.expectedQuery
				}
				message := "BITSTAMP key" + tc.method + captured.Host + "/api/v2/user_transactions/" + query + tc.expectedContentType + nonce + timestamp + "v2" + tc.expectedBody
				mac := hmac.New(sha256.New, []byte("secret"))
				_, err = mac.Write([]byte(message))
				require.NoError(t, err, "Write must not error")
				assert.Equal(t, hex.EncodeToString(mac.Sum(nil)), captured.Header.Get("X-Auth-Signature"), "X-Auth-Signature should sign the documented message")
			})
		}
	})
	t.Run("error responses", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name     string
			status   int
			response string
			contains string
		}{
			{name: "status error", status: http.StatusOK, response: `{"status":"error","reason":"Order not found","code":"404.002"}`, contains: "404.002 Order not found"},
			{name: "field errors", status: http.StatusOK, response: `{"status":"error","reason":{"__all__":["Minimum order size is 10.0 USD."],"price":["Enter a number."]}}`, contains: "Minimum order size is 10.0 USD.; price: Enter a number."},
			{name: "simple error", status: http.StatusOK, response: `{"error":"No permission found"}`, contains: "No permission found"},
			{name: "auth failure", status: http.StatusForbidden, response: `{"status":"error","reason":"Invalid signature","code":"API0005"}`, contains: "API0005 Invalid signature"},
			{name: "derivatives error", status: http.StatusBadRequest, response: `{"code":"API5506","message":"Trade account does not support derivatives.","field":"market"}`, contains: "API5506 Trade account does not support derivatives.; field: market"},
			{name: "unauthorised", status: http.StatusUnauthorized, response: `{"message":"Unauthorized"}`, contains: "Unauthorized"},
			{name: "response code", status: http.StatusOK, response: `{"status":"error","reason":"Request rejected","response_code":"400.002","response_explanation":"Request rejected due to exceeded rate limit."}`, contains: "400.002 Request rejected; Request rejected due to exceeded rate limit."},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(tc.status)
					_, err := w.Write([]byte(tc.response))
					assert.NoError(t, err, "Write should not error")
				})
				err := ex.SendAuthenticatedHTTPRequest(t.Context(), exchange.RestSpot, http.MethodPost, "/v2/order_status/", nil, nil, nil)
				require.ErrorIs(t, err, errAPIResponse, "SendAuthenticatedHTTPRequest must return an API error")
				assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "SendAuthenticatedHTTPRequest should flag an authenticated request failure")
				assert.ErrorContains(t, err, tc.contains, "error should contain the reported details")
			})
		}

		ex := newTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			_, err := w.Write([]byte(`<html>bad gateway</html>`))
			assert.NoError(t, err, "Write should not error")
		})
		err := ex.SendAuthenticatedHTTPRequest(t.Context(), exchange.RestSpot, http.MethodPost, "/v2/order_status/", nil, nil, nil)
		assert.ErrorIs(t, err, request.ErrBadStatus, "SendAuthenticatedHTTPRequest should return the status error for an unrecognised body")
		assert.NotErrorIs(t, err, errAPIResponse, "SendAuthenticatedHTTPRequest should not report an API error for an unrecognised body")
	})
	t.Run("undecodable successful response", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name   string
			status int
			body   string
			err    bool
		}{
			{name: "no content", status: http.StatusNoContent},
			{name: "ok status", status: http.StatusOK, body: `{"status":"ok"}`},
			{name: "error status", status: http.StatusOK, body: `{"status":"error","reason":"Insufficient balance"}`, err: true},
			{name: "HTML", status: http.StatusOK, body: `<html><body>Request unsuccessful.</body></html>`, err: true},
			{name: "truncated JSON", status: http.StatusOK, body: `{"status":"ok"`, err: true},
			{name: "empty body", status: http.StatusOK, err: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				err := newResponseTestExchange(t, tc.status, tc.body).TransferToMainAccount(t.Context(), &TransferRequest{Amount: 1, Currency: currency.BTC, SubAccount: 1})
				if tc.err {
					assert.Errorf(t, err, "TransferToMainAccount should error on a %d response with body %q", tc.status, tc.body)
					return
				}
				assert.NoErrorf(t, err, "TransferToMainAccount should not error on a %d response with body %q", tc.status, tc.body)
			})
		}
	})
}

func TestSendHTTPRequest(t *testing.T) {
	t.Parallel()
	t.Run("ticker", func(t *testing.T) {
		t.Parallel()
		var resp *TickerResponse
		require.NoError(t, e.SendHTTPRequest(t.Context(), exchange.RestSpot, "/v2/ticker/btcusd/", nil, &resp), "SendHTTPRequest must not error")
		require.NotNil(t, resp, "SendHTTPRequest must decode the ticker")
		assert.Positive(t, resp.Last.Float64(), "SendHTTPRequest should decode the last price")
	})
	ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "method should be GET")
		assert.Equal(t, "/api/v2/funding_rate_history/btcusd-perp/", r.URL.Path, "path should be correct")
		assert.Equal(t, "limit=3", r.URL.RawQuery, "query should be correct")
		assert.Empty(t, r.Header.Get("X-Auth"), "public requests should not be authenticated")
		_, err := w.Write([]byte(`{"market":"BTC/USD-PERP","funding_rate_history":[]}`))
		assert.NoError(t, err, "Write should not error")
	})
	var resp *FundingRateHistoryResponse
	require.NoError(t, ex.SendHTTPRequest(t.Context(), exchange.RestSpot, "/v2/funding_rate_history/btcusd-perp/", url.Values{"limit": {"3"}}, &resp), "SendHTTPRequest must not error")
	require.NotNil(t, resp, "response must be decoded")
	assert.Equal(t, "BTC/USD-PERP", resp.Market, "Market should be decoded")

	err := ex.SendHTTPRequest(t.Context(), exchange.RestFutures, "/v2/ticker/", nil, nil)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "SendHTTPRequest should error on an unsupported endpoint")

	ex = newTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	err = ex.SendHTTPRequest(t.Context(), exchange.RestSpot, "/v2/funding_rate/btcusd/", nil, nil)
	assert.ErrorIs(t, err, request.ErrBadStatus, "SendHTTPRequest should return the status error for an empty body")

	ex = newTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte(`{"message":"Cannot fetch funding rate history for market BTC/USD-PERP"}`))
		assert.NoError(t, err, "Write should not error")
	})
	err = ex.SendHTTPRequest(t.Context(), exchange.RestSpot, "/v2/funding_rate_history/btcusd-perp/", nil, nil)
	require.ErrorIs(t, err, errAPIResponse, "SendHTTPRequest must return an API error")
	assert.NotErrorIs(t, err, request.ErrAuthRequestFailed, "SendHTTPRequest should not flag an authenticated request failure")
	assert.ErrorContains(t, err, "Cannot fetch funding rate history", "error should contain the reported message")
}

func TestCheckResponseError(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name               string
		data               string
		unsuccessfulStatus bool
		err                error
		contains           string
	}{
		{name: "array", data: `[{"status":"error"}]`},
		{name: "malformed", data: `{"status":`},
		{name: "empty", data: ``},
		{name: "null", data: `null`},
		{name: "cancelled order", data: `{"id":1,"status":"Canceled"}`},
		{name: "transfer", data: `{"status":"ok"}`},
		{name: "code without message", data: `{"code":"123"}`},
		{name: "status error without details", data: `{"status":"error"}`, err: errAPIResponse, contains: `{"status":"error"}`},
		{name: "reason list", data: `{"status":"error","reason":["a","b"]}`, err: errAPIResponse, contains: "a, b"},
		{name: "numeric code", data: `{"code":4009,"message":"Connection is unauthorized."}`, err: errAPIResponse, contains: "4009 Connection is unauthorized."},
		{name: "error object", data: `{"error":{"amount":["Required"]}}`, err: errAPIResponse, contains: "amount: Required"},
		{name: "field", data: `{"status":"error","reason":"Invalid amount","field":"amount"}`, err: errAPIResponse, contains: "Invalid amount; field: amount"},
		{name: "order timeout", data: `{"status":"error","reason":"Your request timed out before order confirmation. Please check order status with id in \"order_id\" field.","response_code":"500.016","order_id":"1458532827766784"}`, unsuccessfulStatus: true, err: errAPIResponse, contains: "order ID: 1458532827766784"},
		{name: "order timeout numeric ID", data: `{"status":"error","reason":"Your request timed out before order confirmation.","response_code":"500.016","order_id":1458532827766784}`, err: errAPIResponse, contains: "order ID: 1458532827766784"},
		{name: "code only on failure", data: `{"code":"API0001"}`, unsuccessfulStatus: true, err: errAPIResponse, contains: "API0001"},
		{name: "no permission", data: `{"status":"error","reason":"No permission found","code":"API0003"}`, unsuccessfulStatus: true, err: errNoPermission, contains: "API0003 No permission found"},
		{name: "no permission code only", data: `{"code":"API0003"}`, unsuccessfulStatus: true, err: errNoPermission, contains: "API0003"},
		{name: "invalid event ID range", data: `{"status":"error","reason":"Invalid event ID range: until_id must be greater than since_id.","response_code":"400.085"}`, unsuccessfulStatus: true, err: errInvalidEventIDRange, contains: "400.085 Invalid event ID range"},
		{name: "invalid event ID range code only", data: `{"response_code":"400.085"}`, unsuccessfulStatus: true, err: errInvalidEventIDRange, contains: "400.085"},
		{name: "no positions", data: `{"response_code":"400.043","response_explanation":"No positions present."}`, err: futures.ErrNoPositionsFound, contains: "400.043 No positions present."},
		{name: "error response code", data: `{"response_code":"500.016","response_explanation":"Your request timed out before order confirmation."}`, err: errAPIResponse, contains: "500.016 Your request timed out before order confirmation."},
		{name: "response code without an error class", data: `{"response_code":"API5504"}`},
		{name: "unrecognised failure", data: `{"detail":"nope"}`, unsuccessfulStatus: true},
		{name: "mismatched types", data: `{"status":["error"],"reason":1}`, unsuccessfulStatus: true, err: errAPIResponse, contains: "API returned an error: 1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := checkResponseError(json.RawMessage(tc.data), tc.unsuccessfulStatus)
			if tc.err == nil {
				assert.NoError(t, err, "checkResponseError should not error")
				return
			}
			assert.ErrorIs(t, err, tc.err, "checkResponseError should return the correct error")
			assert.ErrorIs(t, err, errAPIResponse, "checkResponseError should report an API error")
			assert.ErrorContains(t, err, tc.contains, "checkResponseError should include the details")
		})
	}
	for _, tc := range []struct {
		name string
		data string
	}{
		{name: "API0001 code", data: `{"code":"API0001"}`},
		{name: "API0004 code", data: `{"code":"API0004"}`},
		{name: "API00030 code", data: `{"code":"API00030"}`},
		{name: "API0003 response code", data: `{"response_code":"API0003"}`},
		{name: "400.085 code", data: `{"code":"400.085"}`},
		{name: "400.084 response code", data: `{"response_code":"400.084"}`},
		{name: "400.0850 response code", data: `{"response_code":"400.0850"}`},
		{name: "400.043 code", data: `{"code":"400.043"}`},
		{name: "400.042 response code", data: `{"response_code":"400.042"}`},
		{name: "400.0430 response code", data: `{"response_code":"400.0430"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := checkResponseError(json.RawMessage(tc.data), true)
			require.ErrorIsf(t, err, errAPIResponse, "checkResponseError must report %s as an API error", tc.data)
			assert.NotErrorIsf(t, err, errNoPermission, "checkResponseError should not report %s as a permission error", tc.data)
			assert.NotErrorIsf(t, err, errInvalidEventIDRange, "checkResponseError should not report %s as an invalid event ID range", tc.data)
			assert.NotErrorIsf(t, err, futures.ErrNoPositionsFound, "checkResponseError should not report %s as no positions", tc.data)
		})
	}
}

func TestIsErrorResponseCode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		code string
		exp  bool
	}{
		{name: "client error", code: "400.043", exp: true},
		{name: "not found", code: "404.001", exp: true},
		{name: "server error", code: "500.016", exp: true},
		{name: "empty", code: "", exp: false},
		{name: "success", code: "200.000", exp: false},
		{name: "redirection", code: "300.001", exp: false},
		{name: "class without a detail", code: "400", exp: false},
		{name: "two digit class", code: "40.043", exp: false},
		{name: "four digit class", code: "4000.43", exp: false},
		{name: "API code", code: "API0003", exp: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, isErrorResponseCode(tc.code), "isErrorResponseCode should return %t for %q", tc.exp, tc.code)
		})
	}
}

func TestProcessResponse(t *testing.T) {
	t.Parallel()
	errRequest := errors.New("request failed")
	assert.ErrorIs(t, processResponse(nil, errRequest, nil, false), errRequest, "processResponse should return the error of a request without a response")
	assert.ErrorIs(t, processResponse(nil, errRequest, new(any), false), errRequest, "processResponse should return the decoding error of an empty response when a result is expected")
	assert.ErrorIs(t, processResponse(nil, errRequest, nil, true), errRequest, "processResponse should return the decoding error of a successful response which cannot be decoded")
	assert.NoError(t, processResponse(json.RawMessage(" null "), nil, nil, true), "processResponse should accept a null successful response when no result is expected")
	badStatus := fmt.Errorf("%w: 500", request.ErrBadStatus)
	assert.ErrorIs(t, processResponse(nil, badStatus, nil, true), request.ErrBadStatus, "processResponse should return the status error of an unsuccessful empty response")
	assert.NoError(t, processResponse(nil, nil, new(any), true), "processResponse should not error on an empty body when a result is optional")

	var resp *AccountBalanceResponse
	assert.ErrorIs(t, processResponse(json.RawMessage("null"), nil, &resp, true), errNoResponse, "processResponse should error on a null body when a response is expected")
	assert.ErrorIs(t, processResponse(nil, nil, &resp, true), errNoResponse, "processResponse should error on an empty body when a response is expected")
	var list []AccountBalanceResponse
	assert.NoError(t, processResponse(json.RawMessage("null"), nil, &list, true), "processResponse should accept a null list")
	assert.Nil(t, list, "processResponse should leave a null list empty")

	require.NoError(t, processResponse(json.RawMessage(`{"currency":"btc","total":"1.5"}`), nil, &resp, true), "processResponse must not error")
	require.NotNil(t, resp, "response must be decoded")
	assert.Equal(t, 1.5, resp.Total.Float64(), "Total should be decoded")

	err := processResponse(json.RawMessage(`{"status":"error","reason":"nope"}`), nil, &resp, false)
	assert.ErrorIs(t, err, errAPIResponse, "processResponse should return an API error")
	assert.NotErrorIs(t, err, request.ErrAuthRequestFailed, "processResponse should not flag an unauthenticated request")
	err = processResponse(json.RawMessage(`{"status":"error","reason":"nope"}`), nil, nil, true)
	assert.ErrorIs(t, err, errAPIResponse, "processResponse should return an API error when no result is expected")
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "processResponse should flag an authenticated request")
}

func TestIsPerpetualPair(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		pair currency.Pair
		exp  bool
	}{
		{name: "spot pair", pair: spotPair, exp: false},
		{name: "perpetual pair", pair: perpetualPair, exp: true},
		{name: "lower case perpetual pair with another delimiter", pair: currency.NewPairWithDelimiter("eth", "usd-perp", "_"), exp: true},
		{name: "perpetual suffix without its hyphen", pair: currency.NewPair(currency.BTC, currency.NewCode("USDPERP")), exp: false},
		{name: "perpetual suffix inside the quote", pair: currency.NewPair(currency.BTC, currency.NewCode("USD-PERPX")), exp: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, isPerpetualPair(tc.pair), "isPerpetualPair should return %t for %s", tc.exp, tc.pair)
		})
	}
}

func TestPaginationParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *PaginationRequest
		exp  url.Values
	}{
		{name: "no paging", req: &PaginationRequest{}, exp: url.Values{}},
		{name: "page", req: &PaginationRequest{Page: 2}, exp: url.Values{"page": {"2"}}},
		{name: "page size", req: &PaginationRequest{PerPage: 50}, exp: url.Values{"per_page": {"50"}}},
		{name: "page and page size", req: &PaginationRequest{Page: 3, PerPage: 100}, exp: url.Values{"page": {"3"}, "per_page": {"100"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, paginationParams(tc.req), "paginationParams should set only the paging parameters given")
		})
	}
}

func TestValidatePaging(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		limit  uint64
		offset uint64
		err    error
	}{
		{name: "no limit or offset"},
		{name: "maximum limit and offset", limit: maxResultLimit, offset: maxResultOffset},
		{name: "limit above the maximum", limit: maxResultLimit + 1, err: errInvalidLimit},
		{name: "offset above the maximum", offset: maxResultOffset + 1, err: errInvalidOffset},
		{name: "limit and offset above the maximum", limit: maxResultLimit + 1, offset: maxResultOffset + 1, err: errInvalidLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, validatePaging(tc.limit, tc.offset), tc.err, "validatePaging should return the correct error")
		})
	}
}

func TestFormatMarketSymbol(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		pair currency.Pair
		exp  string
	}{
		{name: "spot pair", pair: spotPair, exp: "btcusd"},
		{name: "perpetual pair", pair: perpetualPair, exp: "btcusd-perp"},
		{name: "lower case pair with another delimiter", pair: currency.NewPairWithDelimiter("eth", "usd-perp", "_"), exp: "ethusd-perp"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, formatMarketSymbol(tc.pair), "formatMarketSymbol should format %s", tc.pair)
		})
	}
}

func TestFormatMarketName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		pair currency.Pair
		exp  string
	}{
		{name: "spot pair", pair: spotPair, exp: "BTC/USD"},
		{name: "perpetual pair", pair: perpetualPair, exp: "BTC/USD-PERP"},
		{name: "lower case pair with another delimiter", pair: currency.NewPairWithDelimiter("eth", "usd-perp", "_"), exp: "ETH/USD-PERP"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, formatMarketName(tc.pair), "formatMarketName should format %s", tc.pair)
		})
	}
}

func TestFormatOrderSide(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		side order.Side
		exp  string
	}{
		{name: "buy", side: order.Buy, exp: "buy"},
		{name: "bid", side: order.Bid, exp: "buy"},
		{name: "long", side: order.Long, exp: "buy"},
		{name: "sell", side: order.Sell, exp: "sell"},
		{name: "ask", side: order.Ask, exp: "sell"},
		{name: "short", side: order.Short, exp: "sell"},
	} {
		// Serial and in order, as the invalid side below goes to the same function
		t.Run(tc.name, func(t *testing.T) {
			s, err := formatOrderSide(tc.side)
			require.NoErrorf(t, err, "formatOrderSide must not error for %s", tc.side)
			assert.Equalf(t, tc.exp, s, "formatOrderSide should format %s", tc.side)
		})
	}
	t.Run("invalid side", func(t *testing.T) {
		t.Parallel()
		_, err := formatOrderSide(order.AnySide)
		assert.ErrorIs(t, err, order.ErrSideIsInvalid, "formatOrderSide should reject an invalid side")
	})
}

func TestFormatCollateral(t *testing.T) {
	t.Parallel()
	assert.Nil(t, formatCollateral(nil), "formatCollateral should return nil without collateral")
	collateral := map[currency.Code]float64{currency.USD: 120, currency.NewCode("btc"): 0.00000001}
	assert.Equal(t, map[string]string{"USD": "120", "BTC": "0.00000001"}, formatCollateral(collateral), "formatCollateral should key amounts by upper case currency")
}

func TestFormatErrorDetail(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data string
		exp  string
	}{
		{name: "empty", data: ``, exp: ""},
		{name: "padded null", data: ` null `, exp: ""},
		{name: "string", data: `"Order not found"`, exp: "Order not found"},
		{name: "list with an empty entry", data: `["a","","b"]`, exp: "a, b"},
		{name: "list of field errors", data: `[{"price":["Enter a number."]}]`, exp: "price: Enter a number."},
		{name: "object of field errors", data: `{"__all__":["Minimum order size is 10.0 USD."],"price":["Enter a number."],"amount":null}`, exp: "Minimum order size is 10.0 USD.; price: Enter a number."},
		{name: "number", data: `4009`, exp: "4009"},
		{name: "boolean", data: `true`, exp: "true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, formatErrorDetail(json.RawMessage(tc.data)), "formatErrorDetail should format %s", tc.data)
		})
	}
}

func TestSetFloat(t *testing.T) {
	t.Parallel()
	params := url.Values{}
	for _, tc := range []struct {
		name  string
		value float64
		exp   string
	}{
		{name: "decimal", value: 1.5, exp: "1.5"},
		{name: "small decimal without an exponent", value: 0.00000001, exp: "0.00000001"},
		{name: "negative", value: -2, exp: "-2"},
		{name: "large value without an exponent", value: 100000000, exp: "100000000"},
	} {
		// Serial, as each case replaces the amount the case before it set in params
		t.Run(tc.name, func(t *testing.T) {
			setFloat(params, "amount", tc.value)
			assert.Equalf(t, tc.exp, params.Get("amount"), "setFloat should format %v", tc.value)
		})
	}
	t.Run("zero", func(t *testing.T) {
		t.Parallel()
		params := url.Values{}
		setFloat(params, "amount", 0)
		assert.Empty(t, params, "setFloat should not set a zero value")
	})
}

func TestSetTrue(t *testing.T) {
	t.Parallel()
	params := url.Values{}
	setTrue(params, "reduce_only", false)
	assert.Empty(t, params, "setTrue should not set a false value")
	setTrue(params, "reduce_only", true)
	assert.Equal(t, url.Values{"reduce_only": {trueValue}}, params, "setTrue should set a true value")
}

func TestFormatBool(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		value bool
		exp   string
	}{
		{name: "true", value: true, exp: "True"},
		{name: "false", value: false, exp: "False"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, formatBool(tc.value), "formatBool should return the form the API expects")
		})
	}
}

// testIsolatedPosition is the isolated position of the mock open positions
var testIsolatedPosition = PositionResponse{
	ID:                        "1234567890",
	Market:                    "BTC/USD-PERP",
	MarketType:                MarketTypePerpetual,
	MarginMode:                MarginModeIsolated,
	SettlementCurrency:        currency.USD,
	EntryPrice:                80000,
	PNLPercentage:             5,
	PNLRealised:               -2.5,
	PNLSettledSinceInception:  -2.5,
	Leverage:                  3,
	PNL:                       37.5,
	CumulativePricePNL:        40,
	CumulativeTradingFees:     1.6,
	CumulativeFunding:         -0.9,
	Size:                      0.01,
	PNLUnrealised:             40,
	PNLInSettlement:           12.12,
	ImpliedLeverage:           2.8,
	InitialMargin:             266.67,
	InitialMarginRatio:        25,
	CurrentMargin:             306.67,
	CollateralReserved:        300,
	MaintenanceMargin:         133.33,
	MaintenanceMarginRatio:    12.5,
	EstimatedLiquidationPrice: 55000,
	EstimatedClosingFeeAmount: 0.34,
	MarkPrice:                 84000,
	CurrentValue:              840,
	EntryValue:                800,
	StrikePrice:               84000,
	Side:                      PositionSideShort,
	MarginTier:                "1",
}

// testClosedPosition is the mock position closed by ClosePosition
var testClosedPosition = ClosedPositionResponse{
	ID:                    "1234567890",
	Market:                "BTC/USD-PERP",
	MarketType:            MarketTypePerpetual,
	MarginMode:            MarginModeIsolated,
	PNLCurrency:           currency.USD,
	EntryPrice:            80000,
	PNLPercentage:         5,
	PNLRealised:           40,
	PNLSettled:            40,
	Leverage:              3,
	PNL:                   40,
	CumulativePricePNL:    42,
	CumulativeTradingFees: 1.6,
	CumulativeFunding:     -0.4,
	AmountDelta:           0.01,
	TimeOpened:            types.Time(time.UnixMicro(1750068000000000)),
	TimeClosed:            types.Time(time.UnixMicro(1750154400000000)),
	Status:                PositionStatusSettled,
	ExitPrice:             84000,
	SettlementPrice:       84000,
	ClosingFeeAmount:      0.34,
}

var testContact = ContactResponse{
	ID:          "258b8c23-48c0-4916-bba9-0dd47bcdd7cf",
	Description: "Cold wallet owner",
	RetailInfo:  &RetailInfo{FirstName: "John", LastName: "Doe", DateOfBirth: time.Date(1990, 1, 31, 0, 0, 0, 0, time.UTC), Country: "AU"},
}

var testSatoshiTest = SatoshiTestResponse{
	ID:             "6f1c2a4e-7b3d-4e5f-9a8b-1c2d3e4f5a6b",
	Network:        "bitcoin",
	Currency:       currency.BTC,
	Amount:         0.00001234,
	UserAddress:    core.BitcoinDonationAddress,
	DepositAddress: "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
	Status:         "pending",
	Expires:        types.Time(time.Unix(1759995000, 0)),
}

const testExtendedPublicKeyRegistrationID = "9e8d7c6b-5a4f-4e3d-8c2b-1a0f9e8d7c6b"
