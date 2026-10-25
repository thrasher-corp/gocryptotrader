package cryptodotcom

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	// canManipulateRealOrders lets live tests with credentials change the account they act for: place, amend and
	// cancel orders priced away from the market, and change and restore account settings
	canManipulateRealOrders = false
	// testAPIKey and testAPISecret are placeholder credentials, which the mock responses and the test servers accept
	testAPIKey    = "mock-api-key"
	testAPISecret = "mock-api-secret"
	// testServerRequestLimit is the most requests a test server answers, several times as many as any test sends one,
	// so a pager no longer advancing fails its test rather than hangs it
	testServerRequestLimit = 100
	// mockRequestTimeout bounds a paging call the recorded mock serves, for the same reason, as the tests share that
	// mock and so cannot count one test's requests
	mockRequestTimeout = 30 * time.Second
)

var e *Exchange

// testExchangeConfig is Crypto.com's configuration from testdata/configtest.json, loaded once; testexch.Setup loads
// every exchange's configuration each time, which serialises the many exchanges the tests set up on its locks
var testExchangeConfig = sync.OnceValues(func() ([]byte, error) {
	root, err := testutils.RootPathFromCWD()
	if err != nil {
		return nil, err
	}
	cfg := &config.Config{}
	if err := cfg.LoadConfig(filepath.Join(root, "testdata", "configtest.json"), true); err != nil {
		return nil, err
	}
	exchCfg, err := cfg.GetExchangeConfig("Cryptodotcom")
	if err != nil {
		return nil, err
	}
	return json.Marshal(exchCfg)
})

// setupTestExchange sets an exchange up as testexch.Setup does, from its own copy of Crypto.com's test configuration;
// its websocket connections send at once, as only the live venue pro-rates a connection's first second
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
	if mockTests {
		ex.websocketConnectDelay = 0
	}
	ex.Accounts = accounts.MustNewAccounts(ex)
	return nil
}

// setTestCredentials enables authenticated support with the given credentials
func setTestCredentials(ex *Exchange, creds *accounts.Credentials) {
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(creds)
}

// newTestServerExchange returns an exchange, with placeholder credentials, whose REST requests the handler serves, for
// responses the mock recording cannot express, such as malformed payloads and errors
func newTestServerExchange(t *testing.T, handler http.Handler) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, setupTestExchange(ex), "setupTestExchange must not error")
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	ex.Name = t.Name()
	setTestCredentials(ex, &accounts.Credentials{Key: testAPIKey, Secret: testAPISecret})
	ex.SkipAuthCheck = true
	var requests atomic.Int64
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n := requests.Add(1); n > testServerRequestLimit {
			// Reported once, as a client that keeps sending would otherwise fill the test's log
			if n == testServerRequestLimit+1 {
				assert.Failf(t, "the test server should receive no more requests than its limit", "at most %d, not %s %s", testServerRequestLimit, r.Method, r.URL.Path)
			}
			http.Error(w, "too many requests", http.StatusBadRequest)
			return
		}
		handler.ServeHTTP(w, r)
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")
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

// newUnavailableServerExchange returns an exchange whose REST requests all fail without a body Crypto.com would send
func newUnavailableServerExchange(t *testing.T) *Exchange {
	t.Helper()
	return newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
}

// preciseNumber returns the precise number of a test's numeric literal, panicking should the literal not parse
func preciseNumber(s string) types.PreciseNumber {
	n, err := types.NewPreciseNumberFromString(s)
	if err != nil {
		panic(err)
	}
	return n
}

// testRequest is a request a test server received: its method, and its params, a public request's query string and a
// private request's params object
type testRequest struct {
	Method string
	Params string
}

// testRoute answers a method's requests: it returns the result of the request with the params given, left out of the
// response when empty, or an error, which Crypto.com's error envelope carries
type testRoute func(params string) (result string, err *APIError)

// testResult returns a route answering every request with result
func testResult(result string) testRoute {
	return func(string) (string, *APIError) { return result, nil }
}

// newRoutesServerExchange returns an exchange whose REST requests a test server answers by method, from routes, after
// checking a private request is a signed POST; a request for a method routes does not hold fails the test. The function
// returned gives the requests received, in arrival order
func newRoutesServerExchange(t *testing.T, routes map[string]testRoute) (ex *Exchange, received func() []testRequest) {
	t.Helper()
	var mu sync.Mutex
	var requests []testRequest
	ex = newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := strings.TrimPrefix(r.URL.Path, apiPath)
		route, ok := routes[method]
		if !assert.Truef(t, ok, "the test server should only receive requests for its routes, not for %s", method) {
			http.Error(w, "no route", http.StatusNotFound)
			return
		}
		params := r.URL.RawQuery
		if r.Method == http.MethodPost {
			body, ok := readSignedRequest(t, r)
			if !ok {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			assert.Equal(t, method, body.Method, "The request should name its method")
			params = string(body.Params)
		}
		mu.Lock()
		requests = append(requests, testRequest{Method: method, Params: params})
		mu.Unlock()
		// The envelope leaves out the method, which the client does not read, so nothing from the request is echoed
		result, apiErr := route(params)
		response := `{"id":1,"code":0}`
		switch {
		case apiErr != nil:
			response = `{"id":1,"code":` + strconv.FormatInt(apiErr.Code, 10) + `,"message":"` + apiErr.Message + `"}`
		case result != "":
			response = `{"id":1,"code":0,"result":` + result + `}`
		}
		_, err := w.Write([]byte(response))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	received = func() []testRequest {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(requests)
	}
	return ex, received
}

// testResponse returns a successful response envelope for method holding result, which is left out when empty, as
// Crypto.com leaves out a result it does not have
func testResponse(method, result string) string {
	if result == "" {
		return `{"id":1,"method":"` + method + `","code":0}`
	}
	return `{"id":1,"method":"` + method + `","code":0,"result":` + result + `}`
}

// newSignedRecorder returns an exchange whose requests a test server answers with result, which is left out of the
// response when empty, after checking each is a signed POST of method, and a function returning the params of the
// requests received, in arrival order
func newSignedRecorder(t *testing.T, method, result string) (ex *Exchange, sent func() []string) {
	t.Helper()
	response := testResponse(method, result)
	var mu sync.Mutex
	var params []string
	ex = newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "The request should be a POST")
		assert.Equal(t, apiPath+method, r.URL.Path, "The request should be sent to its method's path")
		body, ok := readSignedRequest(t, r)
		if !ok {
			return
		}
		assert.Equal(t, method, body.Method, "The request should name its method")
		mu.Lock()
		params = append(params, string(body.Params))
		mu.Unlock()
		_, err := w.Write([]byte(response))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	sent = func() []string {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(params)
	}
	return ex, sent
}

// newSignedParamsServer returns an exchange whose requests a test server answers with response, after checking each is
// a signed POST of method whose params equal expParams
func newSignedParamsServer(t *testing.T, method, expParams, response string) *Exchange {
	t.Helper()
	return newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "The request should be a POST")
		assert.Equal(t, apiPath+method, r.URL.Path, "The request should be sent to its method's path")
		body, ok := readSignedRequest(t, r)
		if !ok {
			return
		}
		assert.Equal(t, method, body.Method, "The request should name its method")
		assert.JSONEqf(t, expParams, string(body.Params), "%s should send the expected params", method)
		_, err := w.Write([]byte(response))
		assert.NoError(t, err, "Writing the response should not error")
	}))
}

// readSignedRequest decodes a test server's POST request body, checking its signature against the placeholder
// secret, and returns it; a failed check fails the test and returns false
func readSignedRequest(t *testing.T, r *http.Request) (requestBody, bool) {
	t.Helper()
	var body requestBody
	raw, err := io.ReadAll(r.Body)
	if !assert.NoError(t, err, "Reading the request body should not error") {
		return body, false
	}
	if !assert.NoError(t, json.Unmarshal(raw, &body), "The request body should decode") {
		return body, false
	}
	params, err := paramString(body.Params)
	if !assert.NoError(t, err, "paramString should not error for the params sent") {
		return body, false
	}
	signature, err := signRequest(testAPISecret, body.Method, body.ID, testAPIKey, params, body.Nonce)
	if !assert.NoError(t, err, "signRequest should not error") {
		return body, false
	}
	return body, assert.Equal(t, signature, body.Signature, "The request should be signed with the test secret")
}

// mockRequestContext returns the test's context, ending after mockRequestTimeout when the recorded mock serves the test,
// so a pager no longer advancing fails rather than hangs it; live calls are left as long as Crypto.com's rate limits need
func mockRequestContext(t *testing.T) context.Context {
	t.Helper()
	if !mockTests {
		return t.Context()
	}
	ctx, cancel := context.WithTimeout(t.Context(), mockRequestTimeout)
	t.Cleanup(cancel)
	return ctx
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

// signatureVectors are requests signed with the docs' JavaScript example, run with node's crypto in place of
// crypto-js; ccxt's signer, which Crypto.com accepts live, gives the same params strings for the docs' examples and
// for order lists holding execution instructions
var signatureVectors = []struct {
	name      string
	method    string
	id        uint64
	apiKey    string
	secret    string
	nonce     uint64
	params    string
	paramsStr string
	signature string
}{
	{
		name:      "docs get-order-detail",
		method:    "private/get-order-detail",
		id:        11,
		apiKey:    "token",
		secret:    "secretKey",
		nonce:     1587846358253,
		params:    `{"order_id":"53287421324"}`,
		paramsStr: "order_id53287421324",
		signature: "02ef0a52c9428e5d3dcc5dd24d534ca39ef73f35acd3f6945f139a2364ef67a9",
	},
	{
		name:      "docs public auth",
		method:    "public/auth",
		id:        11,
		apiKey:    "token",
		secret:    "secretKey",
		nonce:     1589594102779,
		params:    `{}`,
		signature: "9dcebf6eeec155f829227ee447dee73120e0aead42fab74d38ed5d8271793dc8",
	},
	{
		name:      "docs create-order-list",
		method:    "private/create-order-list",
		id:        14,
		apiKey:    "API_KEY",
		secret:    "SECRET_KEY",
		nonce:     1691385000000,
		params:    `{"contingency_type":"LIST","order_list":[{"instrument_name":"ONE_USDT","side":"BUY","type":"LIMIT","price":"0.24","quantity":"1.0"},{"instrument_name":"ONE_USDT","side":"BUY","type":"STOP_LIMIT","price":"0.27","quantity":"1.0","trigger_price":"0.26"}]}`,
		paramsStr: "contingency_typeLISTorder_listinstrument_nameONE_USDTprice0.24quantity1.0sideBUYtypeLIMITinstrument_nameONE_USDTprice0.27quantity1.0sideBUYtrigger_price0.26typeSTOP_LIMIT",
		signature: "0413b528fcf90bbc799864fdfdbb5475b0d299db80b3dc25367172aec55101fd",
	},
	{
		name:      "nested lists objects and scalars",
		method:    "private/advanced/create-oto",
		id:        1792890000000,
		apiKey:    "mock-api-key",
		secret:    "mock-api-secret",
		nonce:     1792890000000,
		params:    `{"order_list":[{"instrument_name":"BTC_USD","side":"BUY","type":"LIMIT","price":"80000","quantity":"0.001","exec_inst":["POST_ONLY","SMART_POST_ONLY"]},{"instrument_name":"BTC_USD","side":"SELL","type":"TAKE_PROFIT","ref_price":"90000","quantity":"0.001"}],"notify":true,"leverage":10,"empty":[],"nothing":null,"settings":{"b":"2","a":{"d":["x",{"z":"1","y":"2"}],"c":false}}}`,
		paramsStr: "emptyleverage10nothingnullnotifytrueorder_listexec_instPOST_ONLYSMART_POST_ONLYinstrument_nameBTC_USDprice80000quantity0.001sideBUYtypeLIMITinstrument_nameBTC_USDquantity0.001ref_price90000sideSELLtypeTAKE_PROFITsettingsacfalsedxy2z1b2",
		signature: "bd54b18339c2e1396fedb1d00d1124de1bdc0e1f98ace0b305b53c9fc862fa57",
	},
}

func TestParamString(t *testing.T) {
	t.Parallel()
	for _, v := range signatureVectors {
		got, err := paramString([]byte(v.params))
		require.NoErrorf(t, err, "paramString must not error for %s", v.name)
		assert.Equalf(t, v.paramsStr, got, "paramString should build the params string of %s", v.name)
	}
	for _, params := range []string{`{"a":`, `{"a":"\uZZZZ"}`, `{"a":["\uZZZZ"]}`, `{"\uZZZZ":1}`} {
		_, err := paramString([]byte(params))
		assert.Errorf(t, err, "paramString should error for malformed params %s", params)
	}
}

func TestSignRequest(t *testing.T) {
	t.Parallel()
	for _, v := range signatureVectors {
		params, err := paramString([]byte(v.params))
		require.NoErrorf(t, err, "paramString must not error for %s", v.name)
		got, err := signRequest(v.secret, v.method, v.id, v.apiKey, params, v.nonce)
		require.NoErrorf(t, err, "signRequest must not error for %s", v.name)
		assert.Equalf(t, v.signature, got, "signRequest should sign %s as the docs' example does", v.name)
	}
}

func TestEncodeParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		params any
		exp    string
	}{
		{params: nil, exp: `{}`},
		{params: (*struct{})(nil), exp: `{}`},
		{params: &struct{}{}, exp: `{}`},
		{params: map[string]string{"a": "b"}, exp: `{"a":"b"}`},
	} {
		got, err := encodeParams(tc.params)
		require.NoErrorf(t, err, "encodeParams must not error for %v", tc.params)
		assert.Equalf(t, tc.exp, string(got), "encodeParams should encode %v", tc.params)
	}
	_, err := encodeParams(make(chan int))
	assert.Error(t, err, "encodeParams should error for params that cannot be marshalled")
}

func TestAPIError(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "40001 BAD_REQUEST", (&APIError{Code: 40001, Message: "BAD_REQUEST"}).Error(), "APIError should give its code and message")
	assert.Equal(t, "40001", (&APIError{Code: 40001}).Error(), "APIError should give only its code without a message")
}

func TestResponseError(t *testing.T) {
	t.Parallel()
	assert.NoError(t, responseError(&apiResponse{}, nil), "responseError should not error for a successful response")
	assert.ErrorIs(t, responseError(&apiResponse{}, request.ErrBadStatus), request.ErrBadStatus, "responseError should return the request error without a reported one")
	err := responseError(&apiResponse{Code: 40101, Message: "UNAUTHORIZED"}, request.ErrAuthRequestFailed)
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "responseError should keep an authenticated request's failure")
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "responseError must return the reported error")
	assert.Equal(t, &APIError{Code: 40101, Message: "UNAUTHORIZED"}, apiErr, "responseError should report the response's code and message")
	err = responseError(&apiResponse{Code: 40001, Message: "BAD_REQUEST"}, request.ErrBadStatus)
	require.ErrorAs(t, err, &apiErr, "responseError must return the reported error over the request's")
	assert.NotErrorIs(t, err, request.ErrBadStatus, "responseError should report the error Crypto.com explains over the bad status")
}

func TestSendHTTPRequest(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "SendHTTPRequest should send a GET request")
		assert.Equal(t, "/exchange/v1/public/get-tickers", r.URL.Path, "SendHTTPRequest should request the method under the API path")
		assert.Equal(t, "instrument_name=BTC_USD", r.URL.RawQuery, "SendHTTPRequest should send the params as the query string")
		_, err := w.Write([]byte(`{"id":-1,"method":"public/get-tickers","code":0,"result":{"data":[1]}}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	var result struct {
		Data []int64 `json:"data"`
	}
	require.NoError(t, ex.SendHTTPRequest(t.Context(), publicEPL, "public/get-tickers", url.Values{"instrument_name": {"BTC_USD"}}, &result), "SendHTTPRequest must not error")
	assert.Equal(t, []int64{1}, result.Data, "SendHTTPRequest should decode the result")

	err := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte(`{"id":-1,"method":"public/get-tickers","code":"40003","message":"Invalid request"}`))
		assert.NoError(t, err, "Writing the response should not error")
	})).SendHTTPRequest(t.Context(), publicEPL, "public/get-tickers", nil, &result)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "SendHTTPRequest must return the error Crypto.com reports with an error status")
	assert.Equal(t, &APIError{Code: 40003, Message: "Invalid request"}, apiErr, "SendHTTPRequest should decode a quoted error code")
	err = newResponseServerExchange(t, `{"id":-1,"method":"public/get-tickers","code":40401,"message":"NOT_FOUND"}`).SendHTTPRequest(t.Context(), publicEPL, "public/get-tickers", nil, &result)
	require.ErrorAs(t, err, &apiErr, "SendHTTPRequest must return the error Crypto.com reports with a successful status")
	assert.Equal(t, int64(40401), apiErr.Code, "SendHTTPRequest should decode the error code reported with a successful status")
	err = newUnavailableServerExchange(t).SendHTTPRequest(t.Context(), publicEPL, "public/get-tickers", nil, &result)
	assert.ErrorIs(t, err, request.ErrBadStatus, "SendHTTPRequest should return a request failure without a reported error")

	ex = new(Exchange)
	ex.SetDefaults()
	ex.API.Endpoints = ex.NewEndpoints()
	err = ex.SendHTTPRequest(t.Context(), publicEPL, "public/get-tickers", nil, &result)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "SendHTTPRequest should error without an endpoint")
}

func TestSendAuthenticatedHTTPRequest(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var ids []uint64
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "SendAuthenticatedHTTPRequest should send a POST request")
		assert.Equal(t, "/exchange/v1/private/get-order-detail", r.URL.Path, "SendAuthenticatedHTTPRequest should request the method under the API path")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "SendAuthenticatedHTTPRequest should send a JSON body")
		body, ok := readSignedRequest(t, r)
		if !ok {
			return
		}
		assert.Equal(t, "private/get-order-detail", body.Method, "SendAuthenticatedHTTPRequest should name the method in the body")
		assert.Equal(t, testAPIKey, body.APIKey, "SendAuthenticatedHTTPRequest should send the API key")
		assert.InDelta(t, float64(time.Now().UnixMilli()), float64(body.Nonce), float64(time.Minute.Milliseconds()), "SendAuthenticatedHTTPRequest should send the time in milliseconds as the nonce")
		mu.Lock()
		ids = append(ids, body.ID)
		mu.Unlock()
		switch string(body.Params) {
		case `{}`:
		default:
			assert.JSONEq(t, `{"order_id":"1"}`, string(body.Params), "SendAuthenticatedHTTPRequest should send the params")
		}
		_, err := w.Write([]byte(`{"id":1,"method":"private/get-order-detail","code":0,"result":{"order_id":"1"}}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	var result OrderIDResponse
	require.NoError(t, ex.SendAuthenticatedHTTPRequest(t.Context(), getOrderDetailEPL, "private/get-order-detail", nil, &result), "SendAuthenticatedHTTPRequest must not error without params")
	require.NoError(t, ex.SendAuthenticatedHTTPRequest(t.Context(), getOrderDetailEPL, "private/get-order-detail", map[string]string{"order_id": "1"}, &result), "SendAuthenticatedHTTPRequest must not error")
	assert.Equal(t, OrderIDResponse{OrderID: "1"}, result, "SendAuthenticatedHTTPRequest should decode the result")
	require.Len(t, ids, 2, "SendAuthenticatedHTTPRequest must have sent two requests")
	assert.NotEqual(t, ids[0], ids[1], "SendAuthenticatedHTTPRequest should send each request with its own ID")

	err := newResponseServerExchange(t, `{"id":1,"method":"private/get-order-detail","code":40101,"message":"UNAUTHORIZED","original":"{\"api_key\":\"mock-api-key\",\"sig\":\"x\"}"}`).SendAuthenticatedHTTPRequest(t.Context(), getOrderDetailEPL, "private/get-order-detail", nil, &result)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "SendAuthenticatedHTTPRequest must return the error Crypto.com reports")
	assert.Equal(t, &APIError{Code: 40101, Message: "UNAUTHORIZED"}, apiErr, "SendAuthenticatedHTTPRequest should leave out the original request, which carries the API key")
	assert.NotContains(t, err.Error(), testAPIKey, "SendAuthenticatedHTTPRequest should not report the API key")
	err = newUnavailableServerExchange(t).SendAuthenticatedHTTPRequest(t.Context(), getOrderDetailEPL, "private/get-order-detail", nil, &result)
	assert.ErrorIs(t, err, request.ErrBadStatus, "SendAuthenticatedHTTPRequest should return a request failure without a reported error")
	assert.Error(t, ex.SendAuthenticatedHTTPRequest(t.Context(), getOrderDetailEPL, "private/get-order-detail", make(chan int), &result), "SendAuthenticatedHTTPRequest should error for params that cannot be marshalled")

	unauthenticated := new(Exchange)
	require.NoError(t, setupTestExchange(unauthenticated), "setupTestExchange must not error")
	err = unauthenticated.SendAuthenticatedHTTPRequest(t.Context(), getOrderDetailEPL, "private/get-order-detail", nil, &result)
	assert.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "SendAuthenticatedHTTPRequest should error without authentication support")
}

func TestSendPostRequestUnsigned(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body requestBody
		raw, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err, "Reading the request body should not error") {
			return
		}
		if !assert.NoError(t, json.Unmarshal(raw, &body), "The request body should decode") {
			return
		}
		assert.Empty(t, body.APIKey, "An unsigned request should not send the API key")
		assert.Empty(t, body.Signature, "An unsigned request should not send a signature")
		assert.Equal(t, "public/staking/get-conversion-rate", body.Method, "An unsigned request should name the method in the body")
		_, err = w.Write([]byte(`{"id":1,"method":"public/staking/get-conversion-rate","code":0,"result":{"order_id":"2"}}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	var result OrderIDResponse
	require.NoError(t, ex.sendPostRequest(t.Context(), stakingEPL, "public/staking/get-conversion-rate", nil, &result, nil), "sendPostRequest must not error unsigned")
	assert.Equal(t, OrderIDResponse{OrderID: "2"}, result, "sendPostRequest should decode the result")
}

func TestUnixMilliParam(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		t   time.Time
		exp uint64
		err error
	}{
		{},
		{t: time.UnixMilli(1700000000123), exp: 1700000000123},
		{t: time.UnixMilli(-1), err: errTimeBeforeUnixEpoch},
	} {
		got, err := unixMilliParam(tc.t)
		require.ErrorIsf(t, err, tc.err, "unixMilliParam must return the expected error for %s", tc.t)
		assert.Equalf(t, tc.exp, got, "unixMilliParam should give %s in Unix milliseconds", tc.t)
	}
}

func TestUnixNanoParam(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		t   time.Time
		exp uint64
		err error
	}{
		{},
		{t: time.Unix(0, 1700000000123456789), exp: 1700000000123456789},
		{t: time.Unix(0, -1), err: errTimeBeforeUnixEpoch},
	} {
		got, err := unixNanoParam(tc.t)
		require.ErrorIsf(t, err, tc.err, "unixNanoParam must return the expected error for %s", tc.t)
		assert.Equalf(t, tc.exp, got, "unixNanoParam should give %s in Unix nanoseconds", tc.t)
	}
}
