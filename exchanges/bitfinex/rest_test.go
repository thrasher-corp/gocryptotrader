package bitfinex

import (
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
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
	// canManipulateRealOrders lets live tests with credentials change the account they act for: place, update and
	// cancel orders priced away from the market, and change and restore account settings
	canManipulateRealOrders = false
	// testAPIKey and testAPISecret are placeholder credentials, which the mock responses and the test servers accept
	testAPIKey    = "mock-api-key"
	testAPISecret = "mock-api-secret"
)

var e *Exchange

// testExchangeConfig is Bitfinex's configuration from testdata/configtest.json, loaded once; testexch.Setup loads every
// exchange's configuration each time, which serialises the many exchanges the tests set up on its locks
var testExchangeConfig = sync.OnceValues(func() ([]byte, error) {
	root, err := testutils.RootPathFromCWD()
	if err != nil {
		return nil, err
	}
	cfg := &config.Config{}
	if err := cfg.LoadConfig(filepath.Join(root, "testdata", "configtest.json"), true); err != nil {
		return nil, err
	}
	exchCfg, err := cfg.GetExchangeConfig("Bitfinex")
	if err != nil {
		return nil, err
	}
	return json.Marshal(exchCfg)
})

// setupTestExchange sets an exchange up as testexch.Setup does, from its own copy of Bitfinex's test configuration
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
	server := httptest.NewTestServer(t, handler)
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

// newUnavailableServerExchange returns an exchange whose REST requests all fail without a body Bitfinex would send
func newUnavailableServerExchange(t *testing.T) *Exchange {
	t.Helper()
	return newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
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
		switch r.Method {
		case http.MethodGet:
			assert.Equal(t, "/v2/ticker/tBTCUSD", r.URL.Path, "SendHTTPRequest should request the path under the version path")
			assert.Equal(t, "a=1", r.URL.RawQuery, "SendHTTPRequest should send the query string")
			assert.Empty(t, r.Header.Get("Content-Type"), "SendHTTPRequest should not send a content type without a body")
			assert.Empty(t, r.Header.Get("bfx-apikey"), "SendHTTPRequest should not sign the request")
		case http.MethodPost:
			assert.Equal(t, "/v2/calc/fx", r.URL.Path, "SendHTTPRequest should request the path under the version path")
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "SendHTTPRequest should send a body as JSON")
			body, err := io.ReadAll(r.Body)
			if assert.NoError(t, err, "Reading the request body should not error") {
				assert.JSONEq(t, `{"ccy1":"BTC","ccy2":"USD"}`, string(body), "SendHTTPRequest should send the body")
			}
		}
		_, err := w.Write([]byte(`[1.5]`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	var result []float64
	require.NoError(t, ex.SendHTTPRequest(t.Context(), tickerEPL, http.MethodGet, "ticker/tBTCUSD?a=1", nil, &result), "SendHTTPRequest must not error")
	assert.Equal(t, []float64{1.5}, result, "SendHTTPRequest should decode the response")
	require.NoError(t, ex.SendHTTPRequest(t.Context(), foreignExchangeRateEPL, http.MethodPost, "calc/fx", map[string]string{"ccy1": "BTC", "ccy2": "USD"}, &result), "SendHTTPRequest must not error with a body")

	err := newResponseServerExchange(t, `["error",10020,"symbol: invalid"]`).SendHTTPRequest(t.Context(), tickerEPL, http.MethodGet, "ticker/tBTCUSD", nil, &result)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "SendHTTPRequest must return the error Bitfinex reports")
	assert.Equal(t, &APIError{Code: 10020, Message: "symbol: invalid"}, apiErr, "SendHTTPRequest should decode the reported error")
	err = newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte(`["error",10020,"symbol: invalid"]`))
		assert.NoError(t, err, "Writing the response should not error")
	})).SendHTTPRequest(t.Context(), tickerEPL, http.MethodGet, "ticker/tBTCUSD", nil, &result)
	require.ErrorAs(t, err, &apiErr, "SendHTTPRequest must return the error Bitfinex reports with an error status")
	assert.Equal(t, int64(10020), apiErr.Code, "SendHTTPRequest should decode the error code reported with an error status")

	err = newUnavailableServerExchange(t).SendHTTPRequest(t.Context(), tickerEPL, http.MethodGet, "ticker/tBTCUSD", nil, &result)
	assert.ErrorIs(t, err, request.ErrBadStatus, "SendHTTPRequest should return a request failure without a reported error")
	assert.Error(t, e.SendHTTPRequest(t.Context(), tickerEPL, http.MethodPost, "calc/fx", make(chan int), &result), "SendHTTPRequest should error for a body that cannot be marshalled")

	ex = new(Exchange)
	ex.SetDefaults()
	ex.API.Endpoints = ex.NewEndpoints()
	err = ex.SendHTTPRequest(t.Context(), tickerEPL, http.MethodGet, "ticker/tBTCUSD", nil, &result)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "SendHTTPRequest should error without an endpoint")
}

func TestSendAuthenticatedHTTPRequest(t *testing.T) {
	t.Parallel()
	var nonces []int64
	var mu sync.Mutex
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "SendAuthenticatedHTTPRequest should send a POST request")
		assert.Equal(t, "/v2/auth/r/wallets", r.URL.Path, "SendAuthenticatedHTTPRequest should request the path under the version path")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "SendAuthenticatedHTTPRequest should send a JSON body")
		assert.Equal(t, testAPIKey, r.Header.Get("bfx-apikey"), "SendAuthenticatedHTTPRequest should send the API key")
		body, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err, "Reading the request body should not error") {
			return
		}
		nonce := r.Header.Get("bfx-nonce")
		signature, err := crypto.GetHMAC(crypto.HashSHA512_384, []byte("/api/v2/auth/r/wallets"+nonce+string(body)), []byte(testAPISecret))
		if assert.NoError(t, err, "GetHMAC should not error") {
			assert.Equal(t, hex.EncodeToString(signature), r.Header.Get("bfx-signature"), "SendAuthenticatedHTTPRequest should sign the path, nonce and body")
		}
		n, err := strconv.ParseInt(nonce, 10, 64)
		if assert.NoError(t, err, "The nonce should be an integer") {
			mu.Lock()
			nonces = append(nonces, n)
			mu.Unlock()
		}
		switch string(body) {
		case "{}":
		default:
			assert.JSONEq(t, `{"id":[1,2]}`, string(body), "SendAuthenticatedHTTPRequest should send the body")
		}
		_, err = w.Write([]byte(`[["exchange","BTC",1,0,1,null,null]]`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	var result json.RawMessage
	require.NoError(t, ex.SendAuthenticatedHTTPRequest(t.Context(), walletsEPL, "auth/r/wallets", nil, &result), "SendAuthenticatedHTTPRequest must not error")
	assert.JSONEq(t, `[["exchange","BTC",1,0,1,null,null]]`, string(result), "SendAuthenticatedHTTPRequest should decode the response")
	require.NoError(t, ex.SendAuthenticatedHTTPRequest(t.Context(), walletsEPL, "auth/r/wallets", map[string][]int{"id": {1, 2}}, &result), "SendAuthenticatedHTTPRequest must not error with a body")
	require.Len(t, nonces, 2, "SendAuthenticatedHTTPRequest must have sent two requests")
	assert.Greater(t, nonces[1], nonces[0], "SendAuthenticatedHTTPRequest should send increasing nonces")
	// A key keeps the largest nonce it has seen, and the nonces GCT signed with were the time in nanoseconds
	assert.Greater(t, nonces[0], time.Now().Add(-time.Hour).UnixNano(), "SendAuthenticatedHTTPRequest should send nanosecond nonces")

	err := newResponseServerExchange(t, `["error",10100,"apikey: invalid"]`).SendAuthenticatedHTTPRequest(t.Context(), walletsEPL, "auth/r/wallets", nil, &result)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "SendAuthenticatedHTTPRequest must return the error Bitfinex reports")
	assert.Equal(t, &APIError{Code: 10100, Message: "apikey: invalid"}, apiErr, "SendAuthenticatedHTTPRequest should decode the reported error")
	err = newUnavailableServerExchange(t).SendAuthenticatedHTTPRequest(t.Context(), walletsEPL, "auth/r/wallets", nil, &result)
	assert.ErrorIs(t, err, request.ErrBadStatus, "SendAuthenticatedHTTPRequest should return a request failure without a reported error")
	assert.Error(t, ex.SendAuthenticatedHTTPRequest(t.Context(), walletsEPL, "auth/r/wallets", make(chan int), &result), "SendAuthenticatedHTTPRequest should error for a body that cannot be marshalled")

	unauthenticated := new(Exchange)
	require.NoError(t, setupTestExchange(unauthenticated), "setupTestExchange must not error")
	err = unauthenticated.SendAuthenticatedHTTPRequest(t.Context(), walletsEPL, "auth/r/wallets", nil, &result)
	assert.ErrorIs(t, err, exchange.ErrAuthenticationSupportNotEnabled, "SendAuthenticatedHTTPRequest should error without authentication support")
}

func TestResponseError(t *testing.T) {
	t.Parallel()
	for body, want := range map[string]error{
		`["error",10020,"symbol: invalid"]`:        &APIError{Code: 10020, Message: "symbol: invalid"},
		` ["error", 10114, "nonce: small"]`:        &APIError{Code: 10114, Message: "nonce: small"},
		`{"error":"ERR_RATE_LIMIT"}`:               &APIError{Message: "ERR_RATE_LIMIT"},
		`{"error":"ERR","message":"detail"}`:       &APIError{Message: "ERR: detail"},
		`{"error":"ERR","code":5,"message":"ERR"}`: &APIError{Code: 5, Message: "ERR"},
		`[1]`:                    nil,
		`[["error",1,"nested"]]`: nil,
		`["tBTCUSD",1]`:          nil,
		`{"id":1}`:               nil,
		`{"error":""}`:           nil,
		``:                       nil,
		`null`:                   nil,
	} {
		assert.Equalf(t, want, responseError([]byte(body)), "responseError should report the error in %s", body)
	}
	assert.Error(t, responseError([]byte(`["error","code","message"]`)), "responseError should report an error array it cannot decode")
	assert.Equal(t, "10020 symbol: invalid", (&APIError{Code: 10020, Message: "symbol: invalid"}).Error(), "APIError should give its code and message")
	assert.Equal(t, "ERR_RATE_LIMIT", (&APIError{Message: "ERR_RATE_LIMIT"}).Error(), "APIError should give only its message without a code")
}

func TestDecodeResponse(t *testing.T) {
	t.Parallel()
	var result []int64
	assert.ErrorIs(t, decodeResponse(json.RawMessage(`[1]`), common.ErrNoResponse, &result), common.ErrNoResponse, "decodeResponse should return the request error without a reported one")
	var apiErr *APIError
	assert.ErrorAs(t, decodeResponse(json.RawMessage(`["error",1,"x"]`), common.ErrNoResponse, &result), &apiErr, "decodeResponse should return the reported error over the request's")
	err := decodeResponse(json.RawMessage(`["error",10100,"apikey: digest invalid"]`), fmt.Errorf("%w: status 500", request.ErrAuthRequestFailed), &result)
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "decodeResponse should keep an authenticated request's failure")
	assert.ErrorAs(t, err, &apiErr, "decodeResponse should return the error Bitfinex reported for an authenticated request")
	require.NoError(t, decodeResponse(json.RawMessage(`[1,2]`), nil, &result), "decodeResponse must not error")
	assert.Equal(t, []int64{1, 2}, result, "decodeResponse should decode the body")
	assert.NoError(t, decodeResponse(json.RawMessage(`[1]`), nil, nil), "decodeResponse should not error without a result")
	assert.NoError(t, decodeResponse(nil, nil, &result), "decodeResponse should not error for an empty body")
	assert.Error(t, decodeResponse(json.RawMessage(`["x"]`), nil, &result), "decodeResponse should return a decoding error")
}

func TestTimeParam(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1700000000123", timeParam(time.UnixMilli(1700000000123)), "timeParam should format Unix milliseconds")
}

func TestSortParam(t *testing.T) {
	t.Parallel()
	for sort, want := range map[int64]string{1: "1", -1: "-1"} {
		got, err := sortParam(sort)
		require.NoErrorf(t, err, "sortParam must not error for %d", sort)
		assert.Equalf(t, want, got, "sortParam should format %d", sort)
	}
	for _, sort := range []int64{0, 2, -2} {
		_, err := sortParam(sort)
		assert.ErrorIsf(t, err, errInvalidSort, "sortParam should refuse %d", sort)
	}
}

func TestSendNotificationRequest(t *testing.T) {
	t.Parallel()
	ex := newResponseServerExchange(t, `[1678987199,"oc-req",null,null,[1,null,2,"tBTCUSD"],null,"SUCCESS","Submitted"]`)
	resp, err := sendNotificationRequest[Order](t.Context(), ex, cancelOrderEPL, "auth/w/order/cancel", map[string]uint64{"id": 1})
	require.NoError(t, err, "sendNotificationRequest must not error")
	assert.Equal(t, &Notification[Order]{Time: types.Time(time.Unix(1678987199, 0)), Type: "oc-req", Data: Order{ID: 1, ClientOrderID: 2, Symbol: "tBTCUSD"}, Status: NotificationStatusSuccess, Text: "Submitted"}, resp, "sendNotificationRequest should decode the notification")

	_, err = sendNotificationRequest[Order](t.Context(), newResponseServerExchange(t, `[1678987199,"oc-req",null,null,null,null,"ERROR","Order not found."]`), cancelOrderEPL, "auth/w/order/cancel", nil)
	assert.ErrorIs(t, err, errNotificationFailed, "sendNotificationRequest should error for a failed notification")
	_, err = sendNotificationRequest[Order](t.Context(), newResponseServerExchange(t, `null`), cancelOrderEPL, "auth/w/order/cancel", nil)
	assert.ErrorIs(t, err, common.ErrNoResponse, "sendNotificationRequest should error for a null response")
	_, err = sendNotificationRequest[Order](t.Context(), newUnavailableServerExchange(t), cancelOrderEPL, "auth/w/order/cancel", nil)
	assert.ErrorIs(t, err, request.ErrBadStatus, "sendNotificationRequest should return the request failure")
}

func TestSetHistoryParams(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1700000000000), time.UnixMilli(1700000100000)
	params := url.Values{}
	require.NoError(t, setHistoryParams(params, start, end, 25, -1), "setHistoryParams must not error")
	assert.Equal(t, url.Values{"start": {"1700000000000"}, "end": {"1700000100000"}, "limit": {"25"}, "sort": {"-1"}}, params, "setHistoryParams should set every parameter")
	params = url.Values{}
	require.NoError(t, setHistoryParams(params, time.Time{}, time.Time{}, 0, 0), "setHistoryParams must not error with nothing set")
	assert.Empty(t, params, "setHistoryParams should set no unset parameter")
	assert.ErrorIs(t, setHistoryParams(url.Values{}, end, start, 0, 0), common.ErrStartAfterEnd, "setHistoryParams should refuse a reversed window")
	assert.ErrorIs(t, setHistoryParams(url.Values{}, time.Time{}, time.Time{}, 0, 2), errInvalidSort, "setHistoryParams should refuse an invalid sort")
}

func TestNewHistoryBody(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1700000000000), time.UnixMilli(1700000100000)
	body, err := newHistoryBody(start, end, 25, 1)
	require.NoError(t, err, "newHistoryBody must not error")
	assert.Equal(t, historyBody{Start: 1700000000000, End: 1700000100000, Limit: 25, Sort: 1}, body, "newHistoryBody should set every field")
	encoded, err := json.Marshal(historyBody{})
	require.NoError(t, err, "Marshal must not error")
	assert.Equal(t, "{}", string(encoded), "An unset history body should send no field")
	_, err = newHistoryBody(end, start, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd, "newHistoryBody should refuse a reversed window")
	_, err = newHistoryBody(time.Time{}, time.Time{}, 0, -2)
	assert.ErrorIs(t, err, errInvalidSort, "newHistoryBody should refuse an invalid sort")
}
