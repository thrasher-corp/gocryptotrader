package deribit

import (
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testutils "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	// canManipulateRealOrders lets live tests with credentials change the account they act for: place, edit and cancel
	// orders priced away from the market, and change and restore account settings
	canManipulateRealOrders = false
	// testClientID and testClientSecret are placeholder credentials, which the mock responses and the test servers accept
	testClientID     = "mock-client-id"
	testClientSecret = "mock-client-secret"
)

var e *Exchange

// testExchangeConfig is Deribit's configuration from testdata/configtest.json, loaded once; testexch.Setup loads every
// exchange's configuration each time, which serialises the hundreds of exchanges the tests set up on its locks
var testExchangeConfig = sync.OnceValues(func() ([]byte, error) {
	root, err := testutils.RootPathFromCWD()
	if err != nil {
		return nil, err
	}
	cfg := &config.Config{}
	if err := cfg.LoadConfig(filepath.Join(root, "testdata", "configtest.json"), true); err != nil {
		return nil, err
	}
	exchCfg, err := cfg.GetExchangeConfig("Deribit")
	if err != nil {
		return nil, err
	}
	return json.Marshal(exchCfg)
})

// setupTestExchange sets an exchange up as testexch.Setup does, from its own copy of Deribit's test configuration
func setupTestExchange(ex *Exchange) error {
	raw, err := testExchangeConfig()
	if err != nil {
		return err
	}
	var exchCfg config.Exchange
	if err := json.Unmarshal(raw, &exchCfg); err != nil {
		return err
	}
	ex.SetDefaults()
	ex.Websocket = sharedtestvalues.NewTestWebsocket()
	if err := ex.Setup(&exchCfg); err != nil {
		return err
	}
	ex.Accounts = accounts.MustNewAccounts(ex)
	return nil
}

// newTestServerExchange returns an exchange, with placeholder credentials, whose REST requests the handler serves, for
// responses the mock recording cannot express, such as malformed payloads and errors
func newTestServerExchange(t *testing.T, handler http.Handler) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, setupTestExchange(ex), "setupTestExchange must not error")
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	ex.Name = t.Name()
	setTestCredentials(ex, &accounts.Credentials{Key: testClientID, Secret: testClientSecret})
	server := httptest.NewTestServer(t, handler)
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "SetRunningURL must not error")
	return ex
}

// newResponseServerExchange returns an exchange whose REST requests all receive one fixed response body
func newResponseServerExchange(t *testing.T, body string) *Exchange {
	t.Helper()
	return newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Writing the response should not error")
	}))
}

// newUnavailableServerExchange returns an exchange whose REST requests all fail
func newUnavailableServerExchange(t *testing.T) *Exchange {
	t.Helper()
	return newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
}

// countingTransport counts the requests sent through it
type countingTransport struct {
	base http.RoundTripper
	sent atomic.Int32
}

// RoundTrip counts a request and sends it
func (c *countingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.sent.Add(1)
	return c.base.RoundTrip(r)
}

// newTimeoutServerExchange returns an exchange, with placeholder credentials, whose REST requests are never answered and
// so time out after a short HTTP client timeout, and the transport counting the requests its client sends
func newTimeoutServerExchange(t *testing.T) (*Exchange, *countingTransport) {
	t.Helper()
	unanswered := make(chan struct{})
	server := httptest.NewTestServer(t, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-unanswered:
		}
	}))
	t.Cleanup(func() { close(unanswered) })
	ex := new(Exchange)
	require.NoError(t, setupTestExchange(ex), "setupTestExchange must not error")
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	ex.Name = t.Name()
	setTestCredentials(ex, &accounts.Credentials{Key: testClientID, Secret: testClientSecret})
	transport := &countingTransport{base: server.Client().Transport}
	require.NoError(t, ex.SetHTTPClient(&http.Client{Transport: transport, Timeout: 100 * time.Millisecond}), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "SetRunningURL must not error")
	return ex, transport
}

// setTestCredentials enables authenticated support with the given credentials
func setTestCredentials(ex *Exchange, creds *accounts.Credentials) {
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(creds)
}

// skipLiveWithoutCredentials skips a live test of a private method when no credentials are set
func skipLiveWithoutCredentials(t *testing.T) {
	t.Helper()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
}

// skipLiveUnlessCanManipulate skips a live test that changes the account unless credentials are set and
// canManipulateRealOrders allows it
func skipLiveUnlessCanManipulate(t *testing.T) {
	t.Helper()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
}

func TestSendHTTPRequest(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "SendHTTPRequest should send a POST request")
		assert.Equal(t, "/api/v2/public/get_time", r.URL.Path, "SendHTTPRequest should request the method's path")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "SendHTTPRequest should send a JSON body")
		assert.Empty(t, r.Header.Get("Authorization"), "SendHTTPRequest should not sign the request")
		body, err := io.ReadAll(r.Body)
		if assert.NoError(t, err, "Reading the request body should not error") {
			assert.JSONEq(t, `{"jsonrpc":"2.0","method":"public/get_time","params":{"key":"value"}}`, string(body), "SendHTTPRequest should send the JSON-RPC request without an ID")
		}
		_, err = w.Write([]byte(`{"jsonrpc":"2.0","result":1792105300181,"testnet":false}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	var result types.Time
	require.NoError(t, ex.SendHTTPRequest(t.Context(), nonMatchingEPL, "public/get_time", map[string]string{"key": "value"}, &result), "SendHTTPRequest must not error")
	assert.Equal(t, time.UnixMilli(1792105300181), result.Time(), "SendHTTPRequest should decode the result")

	for _, tc := range []struct {
		name   string
		status int
	}{
		{name: "error status", status: http.StatusBadRequest},
		{name: "success status", status: http.StatusOK},
	} {
		server := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
			_, err := w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32602,"data":{"reason":"invalid index","param":"index_name"},"message":"Invalid params"},"testnet":false}`))
			assert.NoError(t, err, "Writing the response should not error")
		}))
		err := server.SendHTTPRequest(t.Context(), nonMatchingEPL, "public/get_index_price", nil, &result)
		var apiErr *APIError
		require.ErrorAsf(t, err, &apiErr, "SendHTTPRequest must return the error object sent with an %s", tc.name)
		exp := &APIError{Code: -32602, Message: "Invalid params", Data: json.RawMessage(`{"reason":"invalid index","param":"index_name"}`)}
		assert.Equalf(t, exp, apiErr, "SendHTTPRequest should decode the error object sent with an %s", tc.name)
	}

	err := newUnavailableServerExchange(t).SendHTTPRequest(t.Context(), nonMatchingEPL, "public/get_time", nil, &result)
	assert.ErrorIs(t, err, request.ErrBadStatus, "SendHTTPRequest should return a request failure without an error object")

	assert.Error(t, e.SendHTTPRequest(t.Context(), nonMatchingEPL, "public/get_time", make(chan int), &result), "SendHTTPRequest should error for params that cannot be marshalled")

	ex = new(Exchange)
	ex.SetDefaults()
	ex.API.Endpoints = ex.NewEndpoints()
	err = ex.SendHTTPRequest(t.Context(), nonMatchingEPL, "public/get_time", nil, &result)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "SendHTTPRequest should error without an endpoint")
}

func TestSendAuthenticatedHTTPRequest(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "SendAuthenticatedHTTPRequest should send a POST request")
		assert.Equal(t, "/api/v2/private/get_account_summary", r.URL.Path, "SendAuthenticatedHTTPRequest should request the method's path")
		body, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err, "Reading the request body should not error") {
			return
		}
		assert.JSONEq(t, `{"jsonrpc":"2.0","method":"private/get_account_summary","params":{"currency":"BTC"}}`, string(body), "SendAuthenticatedHTTPRequest should send the JSON-RPC request")
		assertHTTPSignature(t, r.Header.Get("Authorization"), r.URL.Path, body)
		_, err = w.Write([]byte(`{"jsonrpc":"2.0","result":{"currency":"BTC"},"testnet":false}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	var result map[string]string
	require.NoError(t, ex.SendAuthenticatedHTTPRequest(t.Context(), nonMatchingEPL, "private/get_account_summary", map[string]string{"currency": "BTC"}, &result), "SendAuthenticatedHTTPRequest must not error")
	assert.Equal(t, map[string]string{"currency": "BTC"}, result, "SendAuthenticatedHTTPRequest should decode the result")

	rejected := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":13004,"message":"invalid_credentials"}}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	err := rejected.SendAuthenticatedHTTPRequest(t.Context(), nonMatchingEPL, "private/get_account_summary", nil, &result)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "SendAuthenticatedHTTPRequest must return the error object Deribit sends")
	assert.Equal(t, int64(13004), apiErr.Code, "SendAuthenticatedHTTPRequest should decode the error object")
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "SendAuthenticatedHTTPRequest should report the error object as an authentication failure")

	err = newUnavailableServerExchange(t).SendAuthenticatedHTTPRequest(t.Context(), nonMatchingEPL, "private/get_account_summary", nil, &result)
	assert.ErrorIs(t, err, request.ErrBadStatus, "SendAuthenticatedHTTPRequest should return a request failure")
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "SendAuthenticatedHTTPRequest should report a request failure as an authentication failure")

	ex = newTestServerExchange(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		assert.Fail(t, "SendAuthenticatedHTTPRequest should not send a request without credentials")
	}))
	ex.SkipAuthCheck = false
	ex.SetCredentials(&accounts.Credentials{})
	err = ex.SendAuthenticatedHTTPRequest(t.Context(), nonMatchingEPL, "private/get_account_summary", nil, &result)
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "SendAuthenticatedHTTPRequest should report missing credentials as an authentication failure")
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "SendAuthenticatedHTTPRequest should return the credentials error")

	for _, tc := range []struct {
		method string
		epl    request.EndpointLimit
		sent   int
	}{
		{method: "private/buy", epl: matchingEPL, sent: 1},
		{method: "private/mass_quote", epl: massQuoteEPL, sent: 1},
		{method: "private/withdraw", epl: nonMatchingEPL, sent: 1},
		{method: "private/submit_transfer_to_user", epl: nonMatchingEPL, sent: 1},
		{method: "private/get_account_summary", epl: nonMatchingEPL, sent: request.MaxRetryAttempts + 1},
	} {
		unanswered, transport := newTimeoutServerExchange(t)
		err := unanswered.SendAuthenticatedHTTPRequest(t.Context(), tc.epl, tc.method, nil, &result)
		assert.Errorf(t, err, "SendAuthenticatedHTTPRequest should return the timeout of %s", tc.method)
		assert.Equalf(t, tc.sent, int(transport.sent.Load()), "SendAuthenticatedHTTPRequest should make %d attempts at %s when its response times out", tc.sent, tc.method)
	}
	var attempts atomic.Int32
	retried := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		body := `{"jsonrpc":"2.0","result":{"currency":"BTC"}}`
		if attempts.Add(1) == 1 {
			body = `{"jsonrpc":"2.0","error":{"code":10040,"message":"retry"}}`
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	require.NoError(t, retried.SendAuthenticatedHTTPRequest(t.Context(), matchingEPL, "private/buy", nil, &result), "SendAuthenticatedHTTPRequest must succeed once Deribit stops asking for a retry")
	assert.Equal(t, int32(2), attempts.Load(), "SendAuthenticatedHTTPRequest should send a matching engine request again when Deribit asks for a retry")

	quick := newResponseServerExchange(t, `{"jsonrpc":"2.0","result":{"currency":"BTC"}}`)
	const signedRequests = 10
	start := time.Now()
	for range signedRequests {
		require.NoError(t, quick.SendAuthenticatedHTTPRequest(t.Context(), nonMatchingEPL, "private/get_account_summary", nil, &result), "SendAuthenticatedHTTPRequest must not error")
	}
	// A signed request that waited out the timed lock its nonce takes would spend the lock's whole duration
	assert.Less(t, time.Since(start), signedRequests*request.DefaultMutexLockTimeout/2, "SendAuthenticatedHTTPRequest should not wait out the timed lock taken with its nonce")
}

func TestResendable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		method string
		exp    bool
	}{
		{method: "public/get_time", exp: true},
		{method: "public/auth", exp: true},
		{method: "private/get_account_summary", exp: true},
		{method: "private/list_api_keys", exp: true},
		{method: "private/simulate_portfolio", exp: true},
		{method: "private/buy"},
		{method: "private/cancel_all"},
		{method: "private/withdraw"},
		{method: "private/submit_transfer_to_subaccount"},
		{method: "private/verify_block_trade"},
		{method: "private/getaway"},
		{method: "get_time", exp: true},
	} {
		assert.Equalf(t, tc.exp, resendable(tc.method), "resendable should report whether %s may be sent again", tc.method)
	}
}

func TestHTTPRequestItem(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	var result map[string]string
	for _, tc := range []struct {
		name  string
		path  string
		body  string
		creds *accounts.Credentials
	}{
		{name: "an unsigned request", path: "/api/v2/public/get_time", body: `{"jsonrpc":"2.0","method":"public/get_time"}`},
		{name: "a signed request", path: "/api/v2/private/get_account_summary", body: `{"jsonrpc":"2.0","method":"private/get_account_summary","params":{"currency":"BTC"}}`, creds: &accounts.Credentials{Key: testClientID, Secret: testClientSecret}},
	} {
		item, err := ex.httpRequestItem(apiURL, tc.path, []byte(tc.body), &result, tc.creds)
		require.NoErrorf(t, err, "httpRequestItem must not error for %s", tc.name)
		if tc.creds != nil {
			assertHTTPSignature(t, item.Headers["Authorization"], tc.path, []byte(tc.body))
			delete(item.Headers, "Authorization")
		}
		exp := &request.Item{
			Method:                 http.MethodPost,
			Path:                   apiURL + tc.path,
			Headers:                map[string]string{"Content-Type": "application/json"},
			Body:                   bytes.NewReader([]byte(tc.body)),
			Result:                 &result,
			NonceEnabled:           tc.creds != nil,
			Verbose:                ex.Verbose,
			HTTPDebugging:          ex.HTTPDebugging,
			HTTPRecording:          ex.HTTPRecording,
			HTTPMockDataSliceLimit: ex.HTTPMockDataSliceLimit,
		}
		assert.Equalf(t, exp, item, "httpRequestItem should build %s", tc.name)
	}
}

func TestSendHTTPQueryRequest(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "sendHTTPQueryRequest should send a GET request")
		assert.Equal(t, "/api/v2/public/get_index_price", r.URL.Path, "sendHTTPQueryRequest should request the method's path")
		assert.Equal(t, "index_name=btc_usd", r.URL.RawQuery, "sendHTTPQueryRequest should send the params in the query string")
		assert.Empty(t, r.Header.Get("Authorization"), "sendHTTPQueryRequest should not sign the request")
		_, err := w.Write([]byte(`{"jsonrpc":"2.0","result":{"index_price":82285.99,"estimated_delivery_price":82285.99}}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	var result *IndexPriceResponse
	require.NoError(t, ex.sendHTTPQueryRequest(t.Context(), "public/get_index_price", map[string][]string{"index_name": {marketDataIndexName}}, &result), "sendHTTPQueryRequest must not error")
	assert.Equal(t, &IndexPriceResponse{IndexPrice: 82285.99, EstimatedDeliveryPrice: 82285.99}, result, "sendHTTPQueryRequest should decode the result")

	ex = newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32602,"data":{"reason":"unsupported resolution","param":"resolution"},"message":"Invalid params"}}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	err := ex.sendHTTPQueryRequest(t.Context(), "public/get_tradingview_chart_data", nil, &result)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "sendHTTPQueryRequest must return the error object sent with an error status")
	assert.Equal(t, &APIError{Code: -32602, Message: "Invalid params", Data: json.RawMessage(`{"reason":"unsupported resolution","param":"resolution"}`)}, apiErr, "sendHTTPQueryRequest should decode the error object")

	var attempts atomic.Int32
	ex = newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		body := `{"jsonrpc":"2.0","result":{"index_price":1}}`
		if attempts.Add(1) == 1 {
			body = `{"jsonrpc":"2.0","error":{"code":10040,"message":"retry"}}`
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	require.NoError(t, ex.sendHTTPQueryRequest(t.Context(), "public/get_index_price", nil, &result), "sendHTTPQueryRequest must succeed once Deribit stops asking for a retry")
	assert.Equal(t, int32(2), attempts.Load(), "sendHTTPQueryRequest should send the request again when Deribit asks for a retry")

	err = newUnavailableServerExchange(t).sendHTTPQueryRequest(t.Context(), "public/get_index_price", nil, &result)
	assert.ErrorIs(t, err, request.ErrBadStatus, "sendHTTPQueryRequest should return a request failure without an error object")
}

// assertHTTPSignature checks an Authorization header signs the request's timestamp, nonce, path and body with the test
// secret
func assertHTTPSignature(t *testing.T, authorisation, path string, body []byte) {
	t.Helper()
	fields, ok := strings.CutPrefix(authorisation, "deri-hmac-sha256 ")
	if !assert.True(t, ok, "the Authorization header should use the deri-hmac-sha256 scheme") {
		return
	}
	values := map[string]string{}
	for field := range strings.SplitSeq(fields, ",") {
		key, value, found := strings.Cut(field, "=")
		if assert.Truef(t, found, "the Authorization header field %q should hold a value", field) {
			values[key] = value
		}
	}
	assert.Equal(t, testClientID, values["id"], "the Authorization header should name the client ID")
	payload := values["ts"] + "\n" + values["nonce"] + "\n" + http.MethodPost + "\n" + path + "\n" + string(body) + "\n"
	signature, err := crypto.GetHMAC(crypto.HashSHA256, []byte(payload), []byte(testClientSecret))
	if assert.NoError(t, err, "GetHMAC should not error") {
		assert.Equal(t, hex.EncodeToString(signature), values["sig"], "the Authorization header should sign the request")
	}
}

func TestSignHTTPRequest(t *testing.T) {
	t.Parallel()
	body := []byte(`{"jsonrpc":"2.0","method":"private/get_account_summary","params":{"currency":"BTC"}}`)
	// The signature was computed independently with Python's hmac module from Deribit's documented formula
	authorisation, err := signHTTPRequest(&accounts.Credentials{Key: testClientID, Secret: testClientSecret}, "/api/v2/private/get_account_summary", body, time.UnixMilli(1700000000000), "abc123")
	require.NoError(t, err, "signHTTPRequest must not error")
	assert.Equal(t, "deri-hmac-sha256 id=mock-client-id,ts=1700000000000,sig=a9b8827fcb95caa4b15a44e43316fc9decef156cd13a8dbb71872b98454076f2,nonce=abc123", authorisation, "signHTTPRequest should sign the timestamp, nonce, method, path and body")
}

func TestSendRequest(t *testing.T) {
	t.Parallel()
	var authorised, unauthorised atomic.Int32
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			authorised.Add(1)
		} else {
			unauthorised.Add(1)
		}
		_, err := w.Write([]byte(`{"jsonrpc":"2.0","result":"ok"}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	var result string
	require.NoError(t, ex.sendRequest(t.Context(), false, nonMatchingEPL, "public/test", nil, &result), "sendRequest must not error for a public method")
	assert.Equal(t, int32(1), unauthorised.Load(), "sendRequest should send a public method unsigned")
	require.NoError(t, ex.sendRequest(t.Context(), false, nonMatchingEPL, "private/get_positions", nil, &result), "sendRequest must not error for a private method")
	assert.Equal(t, int32(1), authorised.Load(), "sendRequest should sign a private method")

	ws, _ := newWebsocketTestExchange(t)
	var serverTime types.Time
	require.NoError(t, ws.sendRequest(t.Context(), true, nonMatchingEPL, "public/get_time", nil, &serverTime), "sendRequest must not error over the websocket")
	assert.NotZero(t, serverTime, "sendRequest should decode the websocket result")
}

func TestSendOKRequest(t *testing.T) {
	t.Parallel()
	require.NoError(t, newResponseServerExchange(t, `{"jsonrpc":"2.0","result":"ok"}`).sendOKRequest(t.Context(), false, "private/reset_mmp", nil), "sendOKRequest must not error for an ok result")
	err := newResponseServerExchange(t, `{"jsonrpc":"2.0","result":"failed"}`).sendOKRequest(t.Context(), false, "private/reset_mmp", nil)
	assert.ErrorIs(t, err, errUnexpectedResult, "sendOKRequest should reject a result other than ok")
	err = newUnavailableServerExchange(t).sendOKRequest(t.Context(), false, "private/reset_mmp", nil)
	assert.ErrorIs(t, err, request.ErrBadStatus, "sendOKRequest should return the request failure")
	limited, noDelay := newRateLimitedServerExchange(t, `{"jsonrpc":"2.0","result":"ok"}`)
	assert.NoError(t, limited.sendOKRequest(noDelay, false, "private/reset_mmp", nil), "sendOKRequest should send a non-matching engine request")
}

func TestIsPrivateMethod(t *testing.T) {
	t.Parallel()
	assert.True(t, isPrivateMethod("private/buy"), "isPrivateMethod should report a private method")
	assert.False(t, isPrivateMethod("public/get_time"), "isPrivateMethod should not report a public method")
	assert.False(t, isPrivateMethod("privatebuy"), "isPrivateMethod should need the private scope's slash")
}

func TestRetryRequest(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		retry := &APIError{Code: errorCodeRetry, Message: "retry"}
		var attempts int
		err := retryRequest(t.Context(), func() error {
			attempts++
			if attempts < 3 {
				return retry
			}
			return nil
		})
		require.NoError(t, err, "retryRequest must succeed once Deribit stops asking for a retry")
		assert.Equal(t, 3, attempts, "retryRequest should send the request until Deribit stops asking for a retry")

		attempts = 0
		err = retryRequest(t.Context(), func() error {
			attempts++
			return retry
		})
		assert.ErrorIs(t, err, retry, "retryRequest should return the last retry error")
		assert.Equal(t, maximumRequestAttempts, attempts, "retryRequest should stop at the maximum number of attempts")

		attempts = 0
		other := &APIError{Code: 10028, Message: "too_many_requests"}
		err = retryRequest(t.Context(), func() error {
			attempts++
			return other
		})
		assert.ErrorIs(t, err, other, "retryRequest should return an error Deribit does not ask to retry")
		assert.Equal(t, 1, attempts, "retryRequest should not retry an error Deribit does not ask to retry")

		ctx, cancel := context.WithCancel(t.Context())
		err = retryRequest(ctx, func() error {
			cancel()
			return retry
		})
		assert.ErrorIs(t, err, context.Canceled, "retryRequest should stop waiting once the context is done")
	})
}

func TestUnixMilli(t *testing.T) {
	t.Parallel()
	assert.Equal(t, uint64(1700000000123), unixMilli(time.UnixMilli(1700000000123).Add(999*time.Microsecond)), "unixMilli should return the time in whole Unix milliseconds")
	assert.Equal(t, uint64(1), unixMilli(time.UnixMilli(1)), "unixMilli should return the first millisecond after 1970")
	assert.Zero(t, unixMilli(time.Time{}), "unixMilli should return zero for a zero time, which leaves it out")
	for _, early := range []time.Time{time.Unix(0, 999999), time.UnixMilli(0), time.UnixMilli(-1), time.Date(1969, 7, 20, 20, 17, 0, 0, time.UTC), time.Time{}.Add(time.Second)} {
		assert.Equalf(t, uint64(1), unixMilli(early), "unixMilli should return the first millisecond after 1970 for %s, so it is still sent", early)
	}
}
