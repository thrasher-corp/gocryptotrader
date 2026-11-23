package okx

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestSendHTTPRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, payload, errorText string
		errorCode                int64
		slice, text              bool
	}{
		{name: "object", payload: `{"code":"0","data":{"instId":"BTC-USDT-SWAP"}}`},
		{name: "single item", payload: `{"code":"0","data":[{"instId":"BTC-USDT-SWAP"}]}`},
		{name: "full array", payload: `{"code":"0","data":[{"instId":"BTC-USDT-SWAP"},{"instId":"ETH-USDT-SWAP"}]}`, slice: true},
		{name: "escaped string", payload: `{"code":"0","data":"a\"b\\c\n"}`, text: true},
		{name: "single escaped string", payload: `{"code":"0","data":["a\"b\\c\n"]}`, text: true},
		{name: "empty object result", payload: `{"code":"0","data":[]}`, errorText: "got 0 items"},
		{name: "multiple object results", payload: `{"code":"0","data":[{},{}]}`, errorText: "got 2 items"},
		{name: "invalid object", payload: `{"code":"0","data":{"fundingRate":"invalid"}}`, errorText: "cannot unmarshal response data directly"},
		{name: "invalid single item", payload: `{"code":"0","data":[{"fundingRate":"invalid"}]}`, errorText: "as the first array item"},
		{name: "missing data", payload: `{"code":"0"}`, errorText: "cannot unmarshal response data directly"},
		{name: "API error before incompatible data", payload: `{"code":"50001","msg":"service unavailable","data":42}`, errorText: "service unavailable", errorCode: 50001},
		{name: "unknown API error", payload: `{"code":"999999","data":[]}`, errorText: "999999", errorCode: 999999},
		{name: "documented API error without a message", payload: `{"code":"51603","data":[]}`, errorText: "order does not exist", errorCode: 51603},
		{name: "invalid header", payload: `{"code":"invalid","data":[]}`, errorText: "invalid"},
		{name: "malformed JSON", payload: `{"code":"0","data":[}`, errorText: "invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "exchange must initialise")
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, err := w.Write([]byte(tc.payload))
				assert.NoError(t, err, "response should write")
			}))
			require.NoError(t, ex.SetHTTPClient(server.Client()), "HTTP client must use the test server")
			require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/"), "endpoint must be local")
			if tc.slice {
				var result []FundingRateResponse
				require.NoError(t, ex.SendHTTPRequest(t.Context(), exchange.RestSpot, getFundingRateHistoryEPL, http.MethodGet, "public/funding-rate", nil, &result, request.UnauthenticatedRequest), "array response must decode")
				require.Len(t, result, 2, "both response items must survive")
				assert.Equal(t, "ETH-USDT-SWAP", result[1].InstrumentID, "second instrument should survive")
				return
			}
			if tc.text {
				var result string
				require.NoError(t, ex.SendHTTPRequest(t.Context(), exchange.RestSpot, getFundingRateHistoryEPL, http.MethodGet, "public/funding-rate", nil, &result, request.UnauthenticatedRequest), "string response must decode")
				assert.Equal(t, "a\"b\\c\n", result, "string escapes should survive envelope extraction")
				return
			}
			var result *FundingRateResponse
			err := ex.SendHTTPRequest(t.Context(), exchange.RestSpot, getFundingRateHistoryEPL, http.MethodGet, "public/funding-rate", nil, &result, request.UnauthenticatedRequest)
			if tc.errorText != "" {
				require.ErrorContains(t, err, tc.errorText, "invalid response must retain useful error context")
				if tc.errorCode != 0 {
					var respErr *ResponseError
					require.ErrorAs(t, err, &respErr, "an API error must keep OKX's error code")
					assert.Equal(t, tc.errorCode, respErr.Code, "the API error should carry OKX's error code")
				}
				return
			}
			require.NoError(t, err, "typed response must decode")
			require.NotNil(t, result, "response must be populated")
			assert.Equal(t, "BTC-USDT-SWAP", result.InstrumentID, "instrument should decode")
		})
	}
}

func TestResponseError(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		err     *ResponseError
		text    string
		wrapped error
	}{
		{"message", &ResponseError{Code: 51603, Message: "Order does not exist"}, "error code: `51603`; message: \"Order does not exist\"", ErrorCodes["51603"]},
		{"documented code without a message", &ResponseError{Code: 51603}, "order does not exist", ErrorCodes["51603"]},
		{"undocumented code without a message", &ResponseError{Code: 999999}, "error code: `999999`", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.text, tc.err.Error(), "Error should report the code and message")
			assert.Equal(t, tc.wrapped, tc.err.Unwrap(), "Unwrap should return the code's documented error")
		})
	}
}

func TestUnmarshalResponseData(t *testing.T) {
	t.Parallel()
	type item struct {
		InstrumentID string `json:"instId"`
		Count        uint64 `json:"count"`
	}
	intoPointer := func(data json.RawMessage) (any, error) {
		var result *item
		err := unmarshalResponseData(data, &result)
		return result, err
	}
	intoSlice := func(data json.RawMessage) (any, error) {
		var result []item
		err := unmarshalResponseData(data, &result)
		return result, err
	}
	intoString := func(data json.RawMessage) (any, error) {
		var result string
		err := unmarshalResponseData(data, &result)
		return result, err
	}
	for _, tc := range []struct {
		name   string
		data   string
		decode func(json.RawMessage) (any, error)
		exp    any
		err    string
	}{
		{name: "object", data: `{"instId":"BTC-USDT","count":2}`, decode: intoPointer, exp: &item{InstrumentID: "BTC-USDT", Count: 2}},
		{name: "single item array", data: `[{"instId":"BTC-USDT","count":2}]`, decode: intoPointer, exp: &item{InstrumentID: "BTC-USDT", Count: 2}},
		{name: "null", data: `null`, decode: intoPointer, exp: (*item)(nil)},
		{name: "full array", data: `[{"instId":"BTC-USDT"},{"instId":"ETH-USDT"}]`, decode: intoSlice, exp: []item{{InstrumentID: "BTC-USDT"}, {InstrumentID: "ETH-USDT"}}},
		{name: "empty array into a slice", data: `[]`, decode: intoSlice, exp: []item{}},
		{name: "string", data: `"a\"b"`, decode: intoString, exp: `a"b`},
		{name: "single string array", data: `["a\"b"]`, decode: intoString, exp: `a"b`},
		{name: "empty array into a single item", data: `[]`, decode: intoPointer, err: "or as a single-item array (got 0 items)"},
		{name: "several items into a single item", data: `[{},{}]`, decode: intoPointer, err: "or as a single-item array (got 2 items)"},
		{name: "undecodable object", data: `{"count":"x"}`, decode: intoPointer, err: "or as an array"},
		{name: "undecodable single item", data: `[{"count":"x"}]`, decode: intoPointer, err: "or as the first array item"},
		{name: "number into a single item", data: `42`, decode: intoPointer, err: "or as an array"},
		{name: "missing data", decode: intoPointer, err: "or as an array"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := tc.decode(json.RawMessage(tc.data))
			if tc.err != "" {
				require.ErrorContains(t, err, "cannot unmarshal response data directly", "unmarshalResponseData must report the direct decoding failure")
				assert.ErrorContains(t, err, tc.err, "unmarshalResponseData should report why the array fallback failed")
				return
			}
			require.NoError(t, err, "unmarshalResponseData must not error")
			assert.Equal(t, tc.exp, result, "unmarshalResponseData should decode the data")
		})
	}
}

func TestGetStatusError(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		statusCode    uint64
		statusMessage string
		err           string
	}{
		{name: "success", statusMessage: "Order placed"},
		{name: "success without a message"},
		{name: "failure", statusCode: 51008, statusMessage: "Order failed. Insufficient USDT balance in account.", err: "status code: `51008` status message: \"Order failed. Insufficient USDT balance in account.\""},
		{name: "failure without a message", statusCode: 1, err: "status code: `1` status message: \"\""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := getStatusError(tc.statusCode, tc.statusMessage)
			if tc.err == "" {
				assert.NoError(t, err, "getStatusError should report no error for a zero status code")
				return
			}
			assert.EqualError(t, err, tc.err, "getStatusError should report the status code and message")
		})
	}
}

// Please supply your own keys here to do authenticated endpoint testing
const (
	canManipulateRealOrders = false
	useTestNet              = false
)

// Please supply your own credentials here to do authenticated endpoint testing
var apiCredentials = &accounts.Credentials{
	Key:      "",
	Secret:   "",
	ClientID: "", // passphrase
}

var e *Exchange

var mainPair = currency.NewPairWithDelimiter("BTC", "USDT", "-") // Is used for spot, margin symbols and underlying contracts

var optionsPair = currency.NewPairWithDelimiter("BTC", "USD", "-")

var futuresPair = currency.NewPairWithDelimiter("BTC", "USD", "-") // OKX lists no delivery futures for BTC-USDT

var perpetualSwapPair = currency.NewPairWithDelimiter("BTC", "USDT-SWAP", "-")

var spreadPair = currency.NewPairWithDelimiter("BTC", "USDT_BTC-USDT-SWAP", currency.DashDelimiter)

// contextGenerate sends an optional value to allow test requests
// named this way, so it shows up in auto-complete and reminds you to use it
func contextGenerate() context.Context {
	return context.WithValue(context.Background(), testNetVal, useTestNet)
}

const listingSettlePeriod = time.Minute

func nextListedExpiry(instruments []InstrumentResponse, now time.Time) time.Time {
	pendingExpiries := make(map[string]bool)
	settled := now.Add(-listingSettlePeriod)
	for i := range instruments {
		if listTime := instruments[i].ListTime.Time(); !strings.EqualFold(instruments[i].State, instrumentStateLive) || listTime.IsZero() || listTime.After(settled) {
			pendingExpiries[instruments[i].ExpiryTime.Time().UTC().Format(expiryDateLayout)] = true
		}
	}
	today := now.UTC().Truncate(24 * time.Hour)
	for i := range instruments {
		expTime := instruments[i].ExpiryTime.Time()
		if !expTime.UTC().Truncate(24*time.Hour).After(today) || pendingExpiries[expTime.UTC().Format(expiryDateLayout)] {
			continue
		}
		return expTime
	}
	return time.Time{}
}

func TestNextListedExpiry(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.October, 5, 23, 56, 0, 0, time.UTC)
	nextDay := time.Date(2026, time.October, 6, 8, 0, 0, 0, time.UTC)
	later := nextDay.AddDate(0, 0, 1)
	listed := now.Add(-time.Hour)
	option := func(expiry time.Time, state string, listTime time.Time) InstrumentResponse {
		return InstrumentResponse{ExpiryTime: types.Time(expiry), State: state, ListTime: types.Time(listTime)}
	}
	for _, tc := range []struct {
		name        string
		instruments []InstrumentResponse
		expected    time.Time
	}{
		{"nearest expiry", []InstrumentResponse{option(nextDay, "live", listed), option(later, "live", listed)}, nextDay},
		{"preopen strike", []InstrumentResponse{option(nextDay, "live", listed), option(nextDay, "preopen", now.Add(5*time.Minute)), option(later, "live", listed)}, later},
		{"strike listing later", []InstrumentResponse{option(nextDay, "live", listed), option(nextDay, "live", now.Add(time.Minute)), option(later, "live", listed)}, later},
		{"strike without listing time", []InstrumentResponse{option(nextDay, "live", listed), option(nextDay, "live", time.Time{}), option(later, "live", listed)}, later},
		{"strike listed now", []InstrumentResponse{option(nextDay, "live", listed), option(nextDay, "live", now), option(later, "live", listed)}, later},
		{"strike listed seconds ago", []InstrumentResponse{option(nextDay, "live", listed), option(nextDay, "live", now.Add(-8*time.Second)), option(later, "live", listed)}, later},
		{"strike listed one settle period ago", []InstrumentResponse{option(nextDay, "live", listed), option(nextDay, "live", now.Add(-listingSettlePeriod)), option(later, "live", listed)}, nextDay},
		{"strike listed just under one settle period ago", []InstrumentResponse{option(nextDay, "live", listed), option(nextDay, "live", now.Add(-listingSettlePeriod+time.Millisecond)), option(later, "live", listed)}, later},
		{"preopen strike past its listing time", []InstrumentResponse{option(nextDay, "live", listed), option(nextDay, "preopen", listed), option(later, "live", listed)}, later},
		{"preopen strike in another time zone", []InstrumentResponse{option(nextDay, "live", listed), option(nextDay.In(time.FixedZone("UTC-12", -12*60*60)), "preopen", now.Add(5*time.Minute)), option(later, "live", listed)}, later},
		{"state in upper case", []InstrumentResponse{option(nextDay, "LIVE", listed), option(later, "live", listed)}, nextDay},
		{"expiry later today", []InstrumentResponse{option(now.Add(3*time.Minute), "live", listed), option(nextDay, "live", listed)}, nextDay},
		{"expired", []InstrumentResponse{option(now.Add(-time.Minute), "live", listed), option(nextDay, "live", listed)}, nextDay},
		{"no listed expiry", []InstrumentResponse{option(nextDay, "preopen", now.Add(5*time.Minute))}, time.Time{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, nextListedExpiry(tc.instruments, now), "nextListedExpiry should return the expected expiry")
		})
	}
}

// fakeRequestBudget is far above the requests any funding fake legitimately serves, retries included, so only a
// paging loop exhausts it
const fakeRequestBudget = 50

// cappedHandler fails the test, rather than leaving it to go test's timeout, when the code under test keeps
// requesting past fakeRequestBudget, as a paging regression would
func cappedHandler(t *testing.T, next http.HandlerFunc) http.HandlerFunc {
	t.Helper()
	var calls atomic.Int32
	return func(w http.ResponseWriter, r *http.Request) {
		if n := calls.Add(1); n > fakeRequestBudget {
			assert.Failf(t, "unexpected request", "request %d to %s should not exceed the fake's budget of %d", n, r.URL.Path, fakeRequestBudget)
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		next(w, r)
	}
}

func (e *Exchange) instrumentFamilyFromInstID(instrumentType, instID string) (string, error) {
	e.instrumentsInfoMapLock.Lock()
	defer e.instrumentsInfoMapLock.Unlock()
	if instrumentType != "" {
		insts, okay := e.instrumentsInfoMap[instrumentType]
		if !okay {
			return "", errInvalidInstrumentType
		}
		for a := range insts {
			if insts[a].InstrumentID.String() == instID {
				return insts[a].InstrumentFamily, nil
			}
		}
	} else {
		for _, insts := range e.instrumentsInfoMap {
			for a := range insts {
				if insts[a].InstrumentID.String() == instID {
					return insts[a].InstrumentFamily, nil
				}
			}
		}
	}
	return "", fmt.Errorf("instrument family not found for instrument %s", instID)
}

func TestCappedHandler(t *testing.T) {
	t.Parallel()
	calls := 0
	handler := cappedHandler(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, "/funding", r.URL.Path, "request path should reach the handler")
		w.WriteHeader(http.StatusNoContent)
	})
	for range fakeRequestBudget {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/funding", http.NoBody))
		assert.Equal(t, http.StatusNoContent, response.Code, "requests within the budget should reach the handler")
	}
	assert.Equal(t, fakeRequestBudget, calls, "the full legitimate budget should reach the handler")
}

// writeOKXData writes a successful OKX JSON response carrying data.
func writeOKXData(t *testing.T, w http.ResponseWriter, data any) {
	t.Helper()
	encoded, err := json.Marshal(map[string]any{"code": "0", "msg": "", "data": data})
	if err != nil {
		t.Errorf("marshalling mock response should not error: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(encoded)
}

// mockRequestLimit caps the requests one mock exchange answers. A pager whose
// stop regresses asks for the same page forever, so past the cap the mock fails
// the request and the test fails instead of hanging until go test's timeout.
// The busiest mock, which TestCancelAllOrders' filter subtests share, answers
// 121.
const mockRequestLimit = 200

// limitMockRequests serves the first mockRequestLimit requests with h and fails
// the rest.
func limitMockRequests(t *testing.T, h http.Handler) http.Handler {
	t.Helper()
	var served atomic.Int64
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if served.Add(1) > mockRequestLimit {
			t.Errorf("mock exchange received more than %d requests, the last for %s", mockRequestLimit, r.URL)
			http.Error(w, "mock request limit exceeded", http.StatusBadRequest)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// newMockExchange returns an exchange whose REST endpoints all point at a test
// server running h.
func newMockExchange(t *testing.T, h http.Handler) *Exchange {
	t.Helper()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	require.NoError(t, e.DisableRateLimiter(), "the limiter must disable: the mock answers at once, so the rate limiters every test in the package shares would only hold its requests back behind other tests")
	srv := httptest.NewTestServer(t, limitMockRequests(t, h))
	b := e.GetBase()
	b.SkipAuthCheck = true
	require.NoError(t, e.SetHTTPClient(srv.Client()), "SetHTTPClient must not error")
	for k := range b.API.Endpoints.GetURLMap() {
		require.NoErrorf(t, b.API.Endpoints.SetRunningURL(k, srv.URL+"/"), "Setup must point endpoint %s at the mock server", k)
	}
	return e
}

// newRequestPinExchange returns an exchange whose mock server checks each request is a POST to path carrying the JSON
// body want, and answers it with data
func newRequestPinExchange(t *testing.T, path, want string, data any) *Exchange {
	t.Helper()
	return newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "request method should be POST")
		assert.Equal(t, path, r.URL.Path, "request path should match")
		body, err := io.ReadAll(r.Body)
		if assert.NoError(t, err, "reading the request body should not error") {
			assert.JSONEq(t, want, string(body), "request body should match")
		}
		writeOKXData(t, w, data)
	}))
}

// newQueryPinExchange returns an exchange whose mock server checks each request is a GET to path carrying the query
// want, and answers it with data
func newQueryPinExchange(t *testing.T, path string, want url.Values, data any) *Exchange {
	t.Helper()
	return newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "request method should be GET")
		assert.Equal(t, path, r.URL.Path, "request path should match")
		assert.Equal(t, want, r.URL.Query(), "request query should carry the expected parameters")
		writeOKXData(t, w, data)
	}))
}

// newEntryCountExchange returns a mock exchange which asserts that each request it receives carries count entries in
// the array under the body's key, and answers with data
func newEntryCountExchange(t *testing.T, key string, count int, data any) *Exchange {
	t.Helper()
	return newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sent map[string]json.RawMessage
		if assert.NoError(t, json.NewDecoder(r.Body).Decode(&sent), "the request body should decode") {
			var entries []json.RawMessage
			if assert.NoErrorf(t, json.Unmarshal(sent[key], &entries), "the request body's %s should be an array", key) {
				assert.Lenf(t, entries, count, "the request should carry every entry under %s", key)
			}
		}
		writeOKXData(t, w, data)
	}))
}

// tradingBotAlgoIDData is a reply row carrying only an algo ID, enough for any trading bot response to decode
var tradingBotAlgoIDData = []map[string]string{{"algoId": "1"}}

// newMockWebsocketExchange returns an exchange whose authenticated websocket is
// available to the wrapper, its order connections logged in with the
// exchange's own credentials, with every REST endpoint pointed at the same test
// server and mainPair's instrument code cached. rest answers REST requests, and
// wsReply returns the code and data of OKX's reply to each websocket operation
// other than a login, which the mock accepts.
func newMockWebsocketExchange(t *testing.T, rest http.HandlerFunc, wsReply func(op string, args json.RawMessage) (code string, data any)) *Exchange {
	t.Helper()
	limitedRest := limitMockRequests(t, rest)
	e := testexch.MockWebsocketInstance[Exchange](t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") != "websocket" {
			limitedRest.ServeHTTP(w, r)
			return
		}
		mockws.WsMockUpgrader(t, w, r, func(_ testing.TB, msg []byte, c *gws.Conn) error {
			if string(msg) == "ping" {
				return nil
			}
			var req struct {
				ID        string          `json:"id"`
				Operation string          `json:"op"`
				Arguments json.RawMessage `json:"args"`
			}
			if err := json.Unmarshal(msg, &req); err != nil {
				return err
			}
			switch req.Operation {
			case operationLogin:
				return c.WriteMessage(gws.TextMessage, []byte(`{"event":"login","code":"0","msg":"","connId":"mock"}`))
			case operationSubscribe, operationUnsubscribe:
				var args []json.RawMessage
				if err := json.Unmarshal(req.Arguments, &args); err != nil {
					return err
				}
				for _, arg := range args {
					ack, err := json.Marshal(map[string]any{"id": req.ID, "event": req.Operation, "arg": arg, "connId": "mock"})
					if err != nil {
						return err
					}
					if err := c.WriteMessage(gws.TextMessage, ack); err != nil {
						return err
					}
				}
				return nil
			}
			code, data := wsReply(req.Operation, req.Arguments)
			reply, err := json.Marshal(map[string]any{"id": req.ID, "op": req.Operation, "code": code, "msg": "", "data": data})
			if err != nil {
				return err
			}
			return c.WriteMessage(gws.TextMessage, reply)
		})
	}))
	mockURL, err := e.API.Endpoints.GetURL(exchange.RestSpot)
	require.NoError(t, err, "GetURL must not error for RestSpot")
	for k := range e.API.Endpoints.GetURLMap() {
		require.NoErrorf(t, e.API.Endpoints.SetRunningURL(k, mockURL+"/"), "SetRunningURL must not error for %s", k)
	}
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	require.True(t, e.Websocket.CanUseAuthenticatedWebsocketForWrapper(), "the mock websocket must be available to the wrapper")
	e.instrumentIDCodeMap = map[string]uint64{mainPair.String(): 12345}
	require.NoError(t, e.DisableRateLimiter(), "DisableRateLimiter must not error")
	loginMockWebsocket(t, e)
	return e
}

// loginMockWebsocket logs a mock websocket exchange's private and business connections in with its own credentials,
// as OKX acts for the account a connection logged in as
func loginMockWebsocket(t *testing.T, e *Exchange) {
	t.Helper()
	for _, connection := range []string{privateConnection, businessLoginConnection} {
		conn, err := e.Websocket.GetConnection(connection)
		require.NoErrorf(t, err, "the %s connection must be available", connection)
		require.NoErrorf(t, e.wsAuthenticateConnection(t.Context(), conn), "the %s connection must log in", connection)
	}
}
