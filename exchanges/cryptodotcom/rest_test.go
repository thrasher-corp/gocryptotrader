package cryptodotcom

import (
	"bytes"
	"context"
	"io"
	"math"
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

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
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
	// cancel orders priced away from the market, and cancel the account's ETHUSD-PERP orders; create, update, pause,
	// resume and terminate trading bots, a funding arbitrage bot among them, which trades at the market; stake, unstake
	// and convert staked tokens; request OTC quotes and deals; move funds to a sub-account and to isolated positions'
	// margin, and back; withdraw BTC to the donation address and fiat to the first linked bank account; set the tests'
	// websocket connection to cancel its orders when it disconnects; and write leverage and account settings back as
	// they found them
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

// newRefusingServerExchange returns an exchange whose first REST request fails without a body Crypto.com would send,
// and whose later ones receive an empty list, so a pager retrying the failure returns without it rather than hangs its
// test
func newRefusingServerExchange(t *testing.T) *Exchange {
	t.Helper()
	var requests atomic.Int64
	return newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		_, err := w.Write([]byte(`{"id":1,"code":0,"result":{"data":[]}}`))
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

// testRefusedOnce returns a route refusing its first request and answering later ones with result, so a caller retrying
// the refusal goes on and fails its test rather than hangs it
func testRefusedOnce(result string) testRoute {
	var requests atomic.Int64
	return func(string) (string, *APIError) {
		if requests.Add(1) == 1 {
			return "", &APIError{Code: 50001, Message: "INTERNAL_SERVER_ERROR"}
		}
		return result, nil
	}
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

// newEnvelopeRecorder returns an exchange whose signed requests of method a test server records whole, raw and decoded,
// and answers with result
func newEnvelopeRecorder(t *testing.T, method, result string) (ex *Exchange, sent func() ([]string, []requestBody)) {
	t.Helper()
	response := testResponse(method, result)
	var mu sync.Mutex
	var raws []string
	var bodies []requestBody
	ex = newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err, "Reading the request body should not error") {
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(raw))
		body, ok := readSignedRequest(t, r)
		if !ok {
			return
		}
		mu.Lock()
		raws = append(raws, string(raw))
		bodies = append(bodies, body)
		mu.Unlock()
		_, err = w.Write([]byte(response))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	sent = func() ([]string, []requestBody) {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(raws), slices.Clone(bodies)
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

// newAPIKeyRecorder returns an exchange whose REST requests all receive one fixed response body. The function returned
// gives each request's API key, in arrival order; no signature is checked, as a request carrying other credentials is
// signed with them
func newAPIKeyRecorder(t *testing.T, body string) (ex *Exchange, keys func() []string) {
	t.Helper()
	var mu sync.Mutex
	var received []string
	ex = newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "Reading the request body should not error")
		var sent requestBody
		assert.NoError(t, json.Unmarshal(raw, &sent), "The request body should decode")
		mu.Lock()
		received = append(received, sent.APIKey)
		mu.Unlock()
		_, err = w.Write([]byte(body))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	keys = func() []string {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(received)
	}
	return ex, keys
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

// subtestName names a subtest after s, or "empty" when s is empty, which t.Run would otherwise name "#00"
func subtestName(s string) string {
	if s == "" {
		return "empty"
	}
	return s
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
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, v := range signatureVectors {
		t.Run(v.name, func(t *testing.T) {
			got, err := paramString([]byte(v.params))
			require.NoErrorf(t, err, "paramString must not error for %s", v.name)
			assert.Equalf(t, v.paramsStr, got, "paramString should build the params string of %s", v.name)
		})
	}
	for name, params := range map[string]string{
		"truncated":                 `{"a":`,
		"invalid escape in a value": `{"a":"\uZZZZ"}`,
		"invalid escape in a list":  `{"a":["\uZZZZ"]}`,
		"invalid escape in a key":   `{"\uZZZZ":1}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := paramString([]byte(params))
			assert.Errorf(t, err, "paramString should error for malformed params %s", params)
		})
	}
}

func TestAppendParamString(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		value    string
		dataType jsonparser.ValueType
		exp      string
	}{
		{name: "object keys in ascending order", value: `{"b":"2","a":{"d":true,"c":null}}`, dataType: jsonparser.Object, exp: "acnulldtrueb2"},
		{name: "array elements in order", value: `["x",1,[2,"y"]]`, dataType: jsonparser.Array, exp: "x12y"},
		{name: "string unescaped", value: `a\"b`, dataType: jsonparser.String, exp: `a"b`},
		{name: "number as sent", value: `1.50`, dataType: jsonparser.Number, exp: "1.50"},
		{name: "boolean", value: `false`, dataType: jsonparser.Boolean, exp: "false"},
		{name: "null", value: `null`, dataType: jsonparser.Null, exp: "null"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var sb strings.Builder
			require.NoErrorf(t, appendParamString(&sb, []byte(tc.value), tc.dataType), "appendParamString must not error for %s", tc.value)
			assert.Equalf(t, tc.exp, sb.String(), "appendParamString should append the params string of %s", tc.value)
		})
	}
	for _, tc := range []struct {
		name     string
		value    string
		dataType jsonparser.ValueType
	}{
		{name: "truncated object", value: `{"a":`, dataType: jsonparser.Object},
		{name: "invalid key escape", value: `{"\uZZZZ":1}`, dataType: jsonparser.Object},
		{name: "truncated array", value: `[1,`, dataType: jsonparser.Array},
		{name: "invalid element escape", value: `["\uZZZZ"]`, dataType: jsonparser.Array},
		{name: "invalid string escape", value: `\uZZZZ`, dataType: jsonparser.String},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var sb strings.Builder
			assert.Errorf(t, appendParamString(&sb, []byte(tc.value), tc.dataType), "appendParamString should error for %s", tc.value)
		})
	}
}

func TestRequestNonce(t *testing.T) {
	t.Parallel()
	before := uint64(time.Now().UnixMilli())
	got := requestNonce()
	after := uint64(time.Now().UnixMilli())
	assert.GreaterOrEqual(t, got, before, "requestNonce should be the time in milliseconds when it is called")
	assert.LessOrEqual(t, got, after, "requestNonce should be the time in milliseconds when it is called")
}

func TestSignRequest(t *testing.T) {
	t.Parallel()
	for _, v := range signatureVectors {
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			params, err := paramString([]byte(v.params))
			require.NoErrorf(t, err, "paramString must not error for %s", v.name)
			got, err := signRequest(v.secret, v.method, v.id, v.apiKey, params, v.nonce)
			require.NoErrorf(t, err, "signRequest must not error for %s", v.name)
			assert.Equalf(t, v.signature, got, "signRequest should sign %s as the docs' example does", v.name)
		})
	}
}

func TestEncodeParams(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same function
	t.Parallel()
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, tc := range []struct {
		name   string
		params any
		exp    string
	}{
		{name: "nil", params: nil, exp: `{}`},
		{name: "nil pointer", params: (*struct{})(nil), exp: `{}`},
		{name: "empty struct", params: &struct{}{}, exp: `{}`},
		{name: "map", params: map[string]string{"a": "b"}, exp: `{"a":"b"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := encodeParams(tc.params)
			require.NoErrorf(t, err, "encodeParams must not error for %v", tc.params)
			assert.Equalf(t, tc.exp, string(got), "encodeParams should encode %v", tc.params)
		})
	}
	_, err := encodeParams(make(chan int))
	assert.Error(t, err, "encodeParams should error for params that cannot be marshalled")
}

func TestAPIErrorError(t *testing.T) {
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

func TestAPIResponseErr(t *testing.T) {
	t.Parallel()
	assert.NoError(t, (&apiResponse{Message: "ignored without a code"}).err(), "err should not error for code 0")
	err := (&apiResponse{Code: 40001, Message: "BAD_REQUEST"}).err()
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "err must return an APIError for a nonzero code")
	assert.Equal(t, &APIError{Code: 40001, Message: "BAD_REQUEST"}, apiErr, "err should report the response's code and message")
}

func TestDecodeResult(t *testing.T) {
	t.Parallel()
	t.Run("errors before decoding", func(t *testing.T) {
		t.Parallel()
		var result *OrderIDResponse
		assert.ErrorIs(t, decodeResult(&apiResponse{Result: json.RawMessage(`{"order_id":"1"}`)}, request.ErrBadStatus, "private/get-order-detail", &result, request.UnauthenticatedRequest), request.ErrBadStatus, "decodeResult should return the request's error")
		var apiErr *APIError
		require.ErrorAs(t, decodeResult(&apiResponse{Code: 40401, Message: "NOT_FOUND", Result: json.RawMessage(`{"order_id":"1"}`)}, nil, "private/get-order-detail", &result, request.UnauthenticatedRequest), &apiErr, "decodeResult must return the response's error")
		assert.Equal(t, &APIError{Code: 40401, Message: "NOT_FOUND"}, apiErr, "decodeResult should report the response's code and message")
		assert.Nil(t, result, "decodeResult should not decode the result of a failed request")
		require.ErrorAs(t, decodeResult(&apiResponse{Code: 40401, Message: "NOT_FOUND", Result: json.RawMessage(`["1"]`)}, nil, "private/get-order-detail", &result, request.UnauthenticatedRequest), &apiErr, "decodeResult must return the response's error over a result of the wrong type")
	})
	t.Run("missing result", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct{ name, result string }{
			{name: "absent"},
			{name: "null", result: "null"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				resp := &apiResponse{Result: json.RawMessage(tc.result)}
				var pointer *OrderIDResponse
				var value OrderIDResponse
				var list []OrderIDResponse
				held := &OrderIDResponse{OrderID: "1"}
				for _, destination := range []struct {
					name   string
					result any
				}{
					{name: "pointer", result: &pointer},
					{name: "struct", result: &value},
					{name: "list", result: &list},
					{name: "pointer holding an earlier result", result: &held},
				} {
					assert.ErrorIsf(t, decodeResult(resp, nil, "private/get-order-detail", destination.result, request.UnauthenticatedRequest), common.ErrNoResponse, "decodeResult should require a wanted result for a %s", destination.name)
				}
				assert.Equal(t, &OrderIDResponse{OrderID: "1"}, held, "decodeResult should leave the destination of a missing result as it was")
				assert.ErrorContains(t, decodeResult(resp, nil, "private/get-order-detail", &pointer, request.UnauthenticatedRequest), "private/get-order-detail", "decodeResult should name the method whose result is missing")
				assert.NoError(t, decodeResult(resp, nil, "private/get-order-detail", nil, request.UnauthenticatedRequest), "decodeResult should accept no result when none is wanted")
			})
		}
	})
	t.Run("empty list", func(t *testing.T) {
		t.Parallel()
		var result []OrderIDResponse
		require.NoError(t, decodeResult(&apiResponse{Result: json.RawMessage(`[]`)}, nil, "private/get-order-detail", &result, request.UnauthenticatedRequest), "decodeResult must accept an empty list as a result")
		assert.Equal(t, []OrderIDResponse{}, result, "decodeResult should decode an empty list")
	})
	t.Run("empty object", func(t *testing.T) {
		t.Parallel()
		var result *OrderIDResponse
		require.NoError(t, decodeResult(&apiResponse{Result: json.RawMessage(`{}`)}, nil, "private/get-order-detail", &result, request.UnauthenticatedRequest), "decodeResult must accept an empty object as a result")
		assert.Equal(t, &OrderIDResponse{}, result, "decodeResult should decode an empty object")
	})
	t.Run("object", func(t *testing.T) {
		t.Parallel()
		var result *OrderIDResponse
		require.NoError(t, decodeResult(&apiResponse{Result: json.RawMessage(`{"order_id":"1","client_oid":"c"}`)}, nil, "private/get-order-detail", &result, request.UnauthenticatedRequest), "decodeResult must not error")
		assert.Equal(t, &OrderIDResponse{OrderID: "1", ClientOrderID: "c"}, result, "decodeResult should decode the result")
	})
	t.Run("malformed result", func(t *testing.T) {
		t.Parallel()
		resp := &apiResponse{Result: json.RawMessage(`["1"]`)}
		var result *OrderIDResponse
		err := decodeResult(resp, nil, "private/get-order-detail", &result, request.UnauthenticatedRequest)
		require.Error(t, err, "decodeResult must error for a result of the wrong type")
		assert.NotErrorIs(t, err, request.ErrAuthRequestFailed, "decodeResult should not mark an unauthenticated request's result")
		assert.ErrorIs(t, decodeResult(resp, nil, "private/get-order-detail", &result, request.AuthenticatedRequest), request.ErrAuthRequestFailed, "decodeResult should mark an authenticated request's result it cannot decode")
	})
	t.Run("unusable destination", func(t *testing.T) {
		t.Parallel()
		resp := &apiResponse{Result: json.RawMessage(`{"order_id":"1"}`)}
		var typedNil *OrderIDResponse
		assert.Error(t, decodeResult(resp, nil, "private/get-order-detail", typedNil, request.UnauthenticatedRequest), "decodeResult should error for a nil pointer destination")
		assert.Error(t, decodeResult(resp, nil, "private/get-order-detail", OrderIDResponse{}, request.UnauthenticatedRequest), "decodeResult should error for a destination that is not a pointer")
	})
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

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		var tickers dataResponse[TickerResponse]
		require.NoError(t, e.SendHTTPRequest(t.Context(), publicEPL, "public/get-tickers", nil, &tickers), "SendHTTPRequest must not error")
		assert.NotEmpty(t, tickers.Data, "SendHTTPRequest should decode the tickers")
	})
	t.Run("missing result respects the destination", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct{ name, body string }{
			{name: "missing", body: `{"code":0}`},
			{name: "null", body: `{"code":0,"result":null}`},
			{name: "null between spaces", body: `{"code":0,"result" : null }`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newResponseServerExchange(t, tc.body)
				var result *OrderIDResponse
				assert.ErrorIs(t, ex.SendHTTPRequest(t.Context(), publicEPL, "public/get-tickers", nil, &result), common.ErrNoResponse, "SendHTTPRequest should require a wanted result")
				var value OrderIDResponse
				assert.ErrorIs(t, ex.SendHTTPRequest(t.Context(), publicEPL, "public/get-tickers", nil, &value), common.ErrNoResponse, "SendHTTPRequest should require a wanted result of a destination that cannot be nil")
				assert.NoError(t, ex.SendHTTPRequest(t.Context(), publicEPL, "public/get-tickers", nil, nil), "SendHTTPRequest should accept no result when none is wanted")
			})
		}
	})
	t.Run("empty list is a result", func(t *testing.T) {
		t.Parallel()
		var result []OrderIDResponse
		require.NoError(t, newResponseServerExchange(t, `{"code":0,"result":[]}`).SendHTTPRequest(t.Context(), publicEPL, "public/get-tickers", nil, &result), "SendHTTPRequest must accept an empty list as a result")
		assert.Empty(t, result, "SendHTTPRequest should decode an empty list")
	})
	t.Run("empty object is a result", func(t *testing.T) {
		t.Parallel()
		var result *OrderIDResponse
		require.NoError(t, newResponseServerExchange(t, `{"code":0,"result":{}}`).SendHTTPRequest(t.Context(), publicEPL, "public/get-tickers", nil, &result), "SendHTTPRequest must accept an empty object as a result")
		assert.Equal(t, &OrderIDResponse{}, result, "SendHTTPRequest should decode an empty object")
	})
	t.Run("reported error beats the result", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct{ name, body string }{
			{name: "result after the code", body: `{"code":40401,"message":"NOT_FOUND","result":{"order_id":"1"}}`},
			{name: "result before the code", body: `{"result":{"order_id":"1"},"code":40401,"message":"NOT_FOUND"}`},
			{name: "result of the wrong type before the code", body: `{"result":["1"],"code":40401,"message":"NOT_FOUND"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var result *OrderIDResponse
				err := newResponseServerExchange(t, tc.body).SendHTTPRequest(t.Context(), publicEPL, "public/get-tickers", nil, &result)
				var apiErr *APIError
				require.ErrorAs(t, err, &apiErr, "SendHTTPRequest must return the error Crypto.com reports")
				assert.Equal(t, &APIError{Code: 40401, Message: "NOT_FOUND"}, apiErr, "SendHTTPRequest should report the response's code and message")
				assert.Nil(t, result, "SendHTTPRequest should not decode the result of a failed request")
			})
		}
	})
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

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		skipLiveWithoutCredentials(t)
		var balances dataResponse[UserBalanceResponse]
		require.NoError(t, e.SendAuthenticatedHTTPRequest(t.Context(), privateEPL, "private/user-balance", nil, &balances), "SendAuthenticatedHTTPRequest must not error")
		assert.NotEmpty(t, balances.Data, "SendAuthenticatedHTTPRequest should decode the balances")
	})
	t.Run("missing result respects the destination", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct{ name, body string }{
			{name: "missing", body: `{"code":0}`},
			{name: "null", body: `{"code":0,"result":null}`},
			{name: "null between spaces", body: `{"code":0,"result" : null }`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newResponseServerExchange(t, tc.body)
				var result *OrderIDResponse
				assert.ErrorIs(t, ex.SendAuthenticatedHTTPRequest(t.Context(), getOrderDetailEPL, "private/get-order-detail", nil, &result), common.ErrNoResponse, "SendAuthenticatedHTTPRequest should require a wanted result")
				var value OrderIDResponse
				assert.ErrorIs(t, ex.SendAuthenticatedHTTPRequest(t.Context(), getOrderDetailEPL, "private/get-order-detail", nil, &value), common.ErrNoResponse, "SendAuthenticatedHTTPRequest should require a wanted result of a destination that cannot be nil")
				assert.NoError(t, ex.SendAuthenticatedHTTPRequest(t.Context(), getOrderDetailEPL, "private/get-order-detail", nil, nil), "SendAuthenticatedHTTPRequest should accept no result when none is wanted")
			})
		}
	})
	t.Run("empty list is a result", func(t *testing.T) {
		t.Parallel()
		var result []OrderIDResponse
		require.NoError(t, newResponseServerExchange(t, `{"code":0,"result":[]}`).SendAuthenticatedHTTPRequest(t.Context(), getOrderDetailEPL, "private/get-order-detail", nil, &result), "SendAuthenticatedHTTPRequest must accept an empty list as a result")
		assert.Empty(t, result, "SendAuthenticatedHTTPRequest should decode an empty list")
	})
	t.Run("empty object is a result", func(t *testing.T) {
		t.Parallel()
		var result *OrderIDResponse
		require.NoError(t, newResponseServerExchange(t, `{"code":0,"result":{}}`).SendAuthenticatedHTTPRequest(t.Context(), getOrderDetailEPL, "private/get-order-detail", nil, &result), "SendAuthenticatedHTTPRequest must accept an empty object as a result")
		assert.Equal(t, &OrderIDResponse{}, result, "SendAuthenticatedHTTPRequest should decode an empty object")
	})
	t.Run("reported error beats the result", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct{ name, body string }{
			{name: "result after the code", body: `{"code":40401,"message":"NOT_FOUND","result":{"order_id":"1"}}`},
			{name: "result before the code", body: `{"result":{"order_id":"1"},"code":40401,"message":"NOT_FOUND"}`},
			{name: "result of the wrong type before the code", body: `{"result":["1"],"code":40401,"message":"NOT_FOUND"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var result *OrderIDResponse
				err := newResponseServerExchange(t, tc.body).SendAuthenticatedHTTPRequest(t.Context(), getOrderDetailEPL, "private/get-order-detail", nil, &result)
				var apiErr *APIError
				require.ErrorAs(t, err, &apiErr, "SendAuthenticatedHTTPRequest must return the error Crypto.com reports")
				assert.Equal(t, &APIError{Code: 40401, Message: "NOT_FOUND"}, apiErr, "SendAuthenticatedHTTPRequest should report the response's code and message")
				assert.Nil(t, result, "SendAuthenticatedHTTPRequest should not decode the result of a failed request")
			})
		}
	})
	t.Run("result of the wrong type keeps the requester's mark", func(t *testing.T) {
		t.Parallel()
		var result *OrderIDResponse
		err := newResponseServerExchange(t, `{"code":0,"result":["1"]}`).SendAuthenticatedHTTPRequest(t.Context(), getOrderDetailEPL, "private/get-order-detail", nil, &result)
		assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "SendAuthenticatedHTTPRequest should mark a result it cannot decode as a failed authenticated request")
	})
}

func TestSendAuthenticatedRequest(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var sent []string
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := readSignedRequest(t, r)
		if !ok {
			return
		}
		mu.Lock()
		sent = append(sent, body.OnBehalfOfAccountUUID)
		mu.Unlock()
		_, err := w.Write([]byte(`{"id":1,"method":"private/fiat/fiat-access","code":0,"result":{"order_id":"1"}}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	var result *OrderIDResponse
	require.NoError(t, ex.sendAuthenticatedRequest(t.Context(), privateEPL, "private/fiat/fiat-access", "", nil, &result), "sendAuthenticatedRequest must not error for the account's own request")
	require.NoError(t, ex.sendAuthenticatedRequest(t.Context(), privateEPL, "private/fiat/fiat-access", "8f3e8f7e-1a2b-4c5d-9e0f-123456789abc", nil, &result), "sendAuthenticatedRequest must not error for a broker's request")
	assert.Equal(t, &OrderIDResponse{OrderID: "1"}, result, "sendAuthenticatedRequest should decode the result")
	assert.Equal(t, []string{"", "8f3e8f7e-1a2b-4c5d-9e0f-123456789abc"}, sent, "sendAuthenticatedRequest should send the broker's account UUID only when given")

	unauthenticated := new(Exchange)
	require.NoError(t, setupTestExchange(unauthenticated), "setupTestExchange must not error")
	assert.ErrorIs(t, unauthenticated.sendAuthenticatedRequest(t.Context(), privateEPL, "private/fiat/fiat-access", "", nil, &result), exchange.ErrAuthenticationSupportNotEnabled, "sendAuthenticatedRequest should error without credentials")
}

func TestSendPostRequest(t *testing.T) {
	t.Parallel()
	t.Run("unsigned", func(t *testing.T) {
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
		require.NoError(t, ex.sendPostRequest(t.Context(), stakingEPL, "public/staking/get-conversion-rate", nil, &result, nil, ""), "sendPostRequest must not error unsigned")
		assert.Equal(t, OrderIDResponse{OrderID: "2"}, result, "sendPostRequest should decode the result")
		err := newResponseServerExchange(t, `{"code":0,"result":["2"]}`).sendPostRequest(t.Context(), stakingEPL, "public/staking/get-conversion-rate", nil, &result, nil, "")
		require.Error(t, err, "sendPostRequest must error for a result of the wrong type")
		assert.NotErrorIs(t, err, request.ErrAuthRequestFailed, "sendPostRequest should not mark an unsigned request's result as a failed authenticated request")
	})
}

// TestTimedOutRequestRetries checks that a write Crypto.com may already have processed is not sent again after a
// timeout, while reads, cancels and changes to absolute values are
func TestTimedOutRequestRetries(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		retried bool
		call    func(context.Context, *Exchange) error
	}{
		{name: "CreateOrder", call: func(ctx context.Context, ex *Exchange) error {
			req := tradingTestOrder()
			req.ClientOrderID = ""
			_, err := ex.CreateOrder(ctx, req)
			return err
		}},
		{name: "ClosePosition", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.ClosePosition(ctx, &ClosePositionRequest{InstrumentName: "BTCUSD-PERP", OrderType: OrderTypeMarket})
			return err
		}},
		{name: "CreateOrderList", call: func(ctx context.Context, ex *Exchange) error {
			orders := tradingTestOrderList()
			for i := range orders {
				orders[i].ClientOrderID = ""
			}
			_, err := ex.CreateOrderList(ctx, orders)
			return err
		}},
		{name: "CreateIsolatedMarginTransfer", call: func(ctx context.Context, ex *Exchange) error {
			return ex.CreateIsolatedMarginTransfer(ctx, &CreateIsolatedMarginTransferRequest{IsolationID: tradingTestIsolationID, Direction: IsolatedMarginTransferDirectionCredit, Amount: 50})
		}},
		{name: "CreateAdvancedOrder", call: func(ctx context.Context, ex *Exchange) error {
			req := advancedTriggerOrder()
			req.ClientOrderID = ""
			_, err := ex.CreateAdvancedOrder(ctx, &req)
			return err
		}},
		{name: "CreateOneCancelsTheOtherOrder", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CreateOneCancelsTheOtherOrder(ctx, ocoOrders())
			return err
		}},
		{name: "CreateOneTriggersTheOtherOrder", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CreateOneTriggersTheOtherOrder(ctx, otoOrders())
			return err
		}},
		{name: "CreateOneTriggersAOneCancelsTheOtherOrder", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CreateOneTriggersAOneCancelsTheOtherOrder(ctx, otocoOrders())
			return err
		}},
		{name: "CreateAdvancedOrderList", call: func(ctx context.Context, ex *Exchange) error {
			orders := advancedOrderListOrders()
			for i := range orders {
				orders[i].ClientOrderID = ""
			}
			_, err := ex.CreateAdvancedOrderList(ctx, orders)
			return err
		}},
		{name: "CreateWithdrawal", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CreateWithdrawal(ctx, &WithdrawalRequest{Currency: currency.BTC, Amount: 1, Address: "2NBqqD5GRJ8wHy1PYyCXTe9ke5226FhavBf"})
			return err
		}},
		{name: "CreateFiatWithdrawal", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CreateFiatWithdrawal(ctx, &FiatWithdrawalRequest{BankAccountID: "b2", Amount: 100, Currency: currency.EUR, PaymentNetwork: PaymentNetworkOpenPaydSEPA})
			return err
		}},
		{name: "CreateFiatBankAccount", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CreateFiatBankAccount(ctx, &FiatBankAccountRequest{PlaidAccountID: "a2", PlaidPublicToken: "public-sandbox-2", Currency: currency.USD})
			return err
		}},
		{name: "CreateSubAccountTransfer", call: func(ctx context.Context, ex *Exchange) error {
			return ex.CreateSubAccountTransfer(ctx, &SubAccountTransferRequest{FromAccountUUID: "a", ToAccountUUID: "b", Currency: currency.CRO, Amount: 1})
		}},
		{name: "Stake", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.Stake(ctx, &StakeRequest{InstrumentName: "SOL.staked", Quantity: 1})
			return err
		}},
		{name: "Unstake", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.Unstake(ctx, &UnstakeRequest{InstrumentName: "SOL.staked", Quantity: 0.5})
			return err
		}},
		{name: "ConvertStakedToken", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.ConvertStakedToken(ctx, &ConvertStakedTokenRequest{FromInstrumentName: "ETH.staked", ToInstrumentName: "CDCETH", ExpectedRate: 1.0203, FromQuantity: 3.14159265, SlippageToleranceBasisPoints: 3})
			return err
		}},
		{name: "RequestOTCQuote", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.RequestOTCQuote(ctx, validOTCQuoteRequest())
			return err
		}},
		{name: "RequestOTCDeal", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.RequestOTCDeal(ctx, validOTCDealRequest())
			return err
		}},
		{name: "CreateTradingBot", call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CreateTradingBot(ctx, &CreateTradingBotRequest{BotType: TradingBotTypeDollarCostAveraging, NotifyOnBotChange: true, Settings: dcaBotSettings()})
			return err
		}},
		{name: "GetUserBalance", retried: true, call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.GetUserBalance(ctx)
			return err
		}},
		{name: "CancelExistingOrder", retried: true, call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.CancelExistingOrder(ctx, &CancelOrderRequest{OrderID: tradingTestOrderID})
			return err
		}},
		{name: "AmendOrder", retried: true, call: func(ctx context.Context, ex *Exchange) error {
			_, err := ex.AmendOrder(ctx, &AmendOrderRequest{OrderID: tradingTestOrderID, NewPrice: 82000, NewQuantity: 0.0002})
			return err
		}},
		{name: "ChangeAccountLeverage", retried: true, call: func(ctx context.Context, ex *Exchange) error {
			return ex.ChangeAccountLeverage(ctx, &ChangeAccountLeverageRequest{AccountID: tradingTestAccountID, Leverage: 10})
		}},
		{name: "PauseTradingBot", retried: true, call: func(ctx context.Context, ex *Exchange) error {
			return ex.PauseTradingBot(ctx, &TradingBotRequest{BotID: "4611686018431234561", BotType: TradingBotTypeDollarCostAveraging})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int64
			ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// The server notices the client giving up, which ends the request's context, only once the body is read
				_, err := io.Copy(io.Discard, r.Body)
				assert.NoError(t, err, "Reading the request body should not error")
				if requests.Add(1) == 1 {
					select {
					case <-r.Context().Done():
					case <-time.After(time.Second):
					}
					return
				}
				w.WriteHeader(http.StatusBadRequest)
			}))
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

func TestUnixMilliParam(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		t    time.Time
		exp  uint64
		err  error
	}{
		{name: "zero time"},
		{name: "after the epoch", t: time.UnixMilli(1700000000123), exp: 1700000000123},
		{name: "before the epoch", t: time.UnixMilli(-1), err: errTimeBeforeUnixEpoch},
		{name: "the epoch", t: time.Unix(0, 0)},
		{name: "the last millisecond an int64 holds", t: time.UnixMilli(math.MaxInt64), exp: math.MaxInt64},
		{name: "after the last millisecond an int64 holds", t: time.UnixMilli(math.MaxInt64).Add(time.Millisecond), err: errTimeTooLate},
		{name: "so long before the epoch its milliseconds would wrap", t: time.Date(-300000000, 1, 1, 0, 0, 0, 0, time.UTC), err: errTimeBeforeUnixEpoch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := unixMilliParam(tc.t)
			require.ErrorIsf(t, err, tc.err, "unixMilliParam must return the expected error for %s", tc.t)
			assert.Equalf(t, tc.exp, got, "unixMilliParam should give %s in Unix milliseconds", tc.t)
		})
	}
}

func TestUnixNanoParam(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		t    time.Time
		exp  uint64
		err  error
	}{
		{name: "zero time"},
		{name: "after the epoch", t: time.Unix(0, 1700000000123456789), exp: 1700000000123456789},
		{name: "before the epoch", t: time.Unix(0, -1), err: errTimeBeforeUnixEpoch},
		{name: "the last nanosecond an int64 holds", t: time.Unix(0, math.MaxInt64), exp: math.MaxInt64},
		{name: "after the last nanosecond an int64 holds", t: time.Unix(0, math.MaxInt64).Add(time.Nanosecond), err: errTimeTooLate},
		{name: "a year an int64 of nanoseconds wraps", t: time.Date(2600, 1, 1, 0, 0, 0, 0, time.UTC), err: errTimeTooLate},
		{name: "a year before the first an int64 of nanoseconds holds", t: time.Date(1600, 1, 1, 0, 0, 0, 0, time.UTC), err: errTimeBeforeUnixEpoch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := unixNanoParam(tc.t)
			require.ErrorIsf(t, err, tc.err, "unixNanoParam must return the expected error for %s", tc.t)
			assert.Equalf(t, tc.exp, got, "unixNanoParam should give %s in Unix nanoseconds", tc.t)
		})
	}
}

// TestUnrecordedResponseFields decodes the response fields the recorded responses leave zero or out, each on its own, so
// that a wrong JSON tag on one fails a test
func TestUnrecordedResponseFields(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data string
		got  any
		exp  any
	}{
		{"Account.Suspended", `{"suspended":true}`, new(Account), &Account{Suspended: true}},
		{"Account.Terminated", `{"terminated":true}`, new(Account), &Account{Terminated: true}},
		{"BalanceSummary.IsLiquidating", `{"is_liquidating":true}`, new(BalanceSummary), &BalanceSummary{IsLiquidating: true}},
		{"BalanceSummary.Terminatable", `{"terminatable":true}`, new(BalanceSummary), &BalanceSummary{Terminatable: true}},
		{"FiatBankAccountResponse.OK", `{"ok":"true"}`, new(FiatBankAccountResponse), &FiatBankAccountResponse{OK: "true"}},
		{"FiatBankAccountResponse.Code", `{"code":"0"}`, new(FiatBankAccountResponse), &FiatBankAccountResponse{Code: "0"}},
		{"FiatBankAccountResponse.Message", `{"message":"success"}`, new(FiatBankAccountResponse), &FiatBankAccountResponse{Message: "success"}},
		{"FiatBankAccountResponse.BankCity", `{"bank_city":"West Reading"}`, new(FiatBankAccountResponse), &FiatBankAccountResponse{BankCity: "West Reading"}},
		{"FiatBankAccountResponse.BankAccountHolderName", `{"bank_account_holder_name":"Smoke Test Co"}`, new(FiatBankAccountResponse), &FiatBankAccountResponse{BankAccountHolderName: "Smoke Test Co"}},
		{"FiatBankAccountResponse.AccountType", `{"account_type":"business"}`, new(FiatBankAccountResponse), &FiatBankAccountResponse{AccountType: "business"}},
		{"FiatBankAccountResponse.WithdrawalPaymentNetworks", `{"withdrawal_payment_networks":["usd_cubix"]}`, new(FiatBankAccountResponse), &FiatBankAccountResponse{WithdrawalPaymentNetworks: []string{PaymentNetworkUSDCubix}}},
		{"FiatBankAccountDetails.IntermediateBankOptions", `{"intermediate_bank_options":[{"bank_name":"Example Bank"}]}`, new(FiatBankAccountDetails), &FiatBankAccountDetails{IntermediateBankOptions: []json.RawMessage{json.RawMessage(`{"bank_name":"Example Bank"}`)}}},
		{"FiatTransactionHistoryResponse.Page", `{"page":"3"}`, new(FiatTransactionHistoryResponse), &FiatTransactionHistoryResponse{Page: 3}},
		{"FiatTransactionQuotaResponse.CurrencyTransactionsPerMonth", `{"currency_transactions_per_month":200}`, new(FiatTransactionQuotaResponse), &FiatTransactionQuotaResponse{CurrencyTransactionsPerMonth: new(types.Number(200))}},
		{"FiatTransactionQuotaResponse.CurrencyRemainingTransactionsMonthlyCount", `{"currency_remaining_transactions_monthly_count":189}`, new(FiatTransactionQuotaResponse), &FiatTransactionQuotaResponse{CurrencyRemainingTransactionsMonthlyCount: new(types.Number(189))}},
		{"RiskParametersResponse.DefaultCollateralCap", `{"default_collateral_cap":"5000000.0"}`, new(RiskParametersResponse), &RiskParametersResponse{DefaultCollateralCap: 5000000}},
		{"BaseCurrencyRiskConfig.MaximumProductLeverageForFutures", `{"max_product_leverage_for_futures":"25.0"}`, new(BaseCurrencyRiskConfig), &BaseCurrencyRiskConfig{MaximumProductLeverageForFutures: 25}},
		{"InstrumentResponse.BetaProduct", `{"beta_product":true}`, new(InstrumentResponse), &InstrumentResponse{BetaProduct: true}},
		{"DepositInstrumentResponse.Tradable", `{"tradable":true}`, new(DepositInstrumentResponse), &DepositInstrumentResponse{Tradable: true}},
		{"DepositInstrumentResponse.ExpiryTime", `{"expiry_timestamp_ms":1793347200000}`, new(DepositInstrumentResponse), &DepositInstrumentResponse{ExpiryTime: types.Time(time.UnixMilli(1793347200000))}},
		{"DepositInstrumentResponse.BetaProduct", `{"beta_product":true}`, new(DepositInstrumentResponse), &DepositInstrumentResponse{BetaProduct: true}},
		{"DepositInstrumentResponse.MarginBuyEnabled", `{"margin_buy_enabled":true}`, new(DepositInstrumentResponse), &DepositInstrumentResponse{MarginBuyEnabled: true}},
		{"DepositInstrumentResponse.MarginSellEnabled", `{"margin_sell_enabled":true}`, new(DepositInstrumentResponse), &DepositInstrumentResponse{MarginSellEnabled: true}},
		{"DepositInstrumentResponse.InstrumentID", `{"inst_id":501}`, new(DepositInstrumentResponse), &DepositInstrumentResponse{InstrumentID: 501}},
		{"OTCQuoteRequestResponse.FirmQuote", `{"firm_quote":true}`, new(OTCQuoteRequestResponse), &OTCQuoteRequestResponse{FirmQuote: true}},
		{"OTCQuoteLeg.Notional", `{"notional":"1000"}`, new(OTCQuoteLeg), &OTCQuoteLeg{Notional: 1000}},
		{"OTCDealLeg.Notional", `{"notional":"1000"}`, new(OTCDealLeg), &OTCDealLeg{Notional: preciseNumber("1000")}},
		{"StakingInstrumentResponse.OutOfStock", `{"out_of_stock":true}`, new(StakingInstrumentResponse), &StakingInstrumentResponse{OutOfStock: true}},
		{"StakingInstrumentResponse.BlockUnstake", `{"block_unstake":true}`, new(StakingInstrumentResponse), &StakingInstrumentResponse{BlockUnstake: true}},
		{"StakingInstrumentResponse.PreStakeChargeEnabled", `{"pre_stake_charge_enable":true}`, new(StakingInstrumentResponse), &StakingInstrumentResponse{PreStakeChargeEnabled: true}},
		{"StakingInstrumentResponse.PreStakeChargeRateBasisPoints", `{"pre_stake_charge_rate_in_bps":"50"}`, new(StakingInstrumentResponse), &StakingInstrumentResponse{PreStakeChargeRateBasisPoints: 50}},
		{"StakingInstrumentResponse.IsRestaked", `{"is_restaked":true}`, new(StakingInstrumentResponse), &StakingInstrumentResponse{IsRestaked: true}},
		{"UserTradeResponse.MatchIndex", `{"match_index":"1"}`, new(UserTradeResponse), &UserTradeResponse{MatchIndex: 1}},
		{"UserTradeResponse.IsolationID", `{"isolation_id":"6530219466236720402"}`, new(UserTradeResponse), &UserTradeResponse{IsolationID: "6530219466236720402"}},
		{"UserTradeResponse.IsolationType", `{"isolation_type":"ISOLATED_MARGIN"}`, new(UserTradeResponse), &UserTradeResponse{IsolationType: IsolationTypeIsolatedMargin}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.NoErrorf(t, json.Unmarshal([]byte(tc.data), tc.got), "Unmarshal must not error for %T %s", tc.exp, tc.data)
			assert.Equalf(t, tc.exp, tc.got, "Unmarshal should decode %T %s", tc.exp, tc.data)
		})
	}
}
