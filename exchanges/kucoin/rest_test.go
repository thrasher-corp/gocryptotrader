package kucoin

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// Please supply your own keys here to do authenticated endpoint testing
const canManipulateRealOrders = false

// Please supply your own credentials here to do authenticated endpoint testing
var apiCredentials = &accounts.Credentials{
	Key:      "",
	Secret:   "",
	ClientID: "", // passphrase
}

var (
	e                                                         *Exchange
	spotTradablePair, marginTradablePair, futuresTradablePair currency.Pair
	assertToTradablePairMap                                   map[asset.Item]currency.Pair
)

func BenchmarkIntervalToString(b *testing.B) {
	for b.Loop() {
		_, err := IntervalToString(kline.OneWeek)
		require.NoError(b, err, "IntervalToString must not error")
	}
}

func TestIntervalToString(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		interval kline.Interval
		exp      string
		err      error
	}{
		{interval: kline.OneMin, exp: "1min"},
		{interval: kline.ThreeMin, exp: "3min"},
		{interval: kline.FiveMin, exp: "5min"},
		{interval: kline.TenMin, err: kline.ErrUnsupportedInterval},
	} {
		got, err := IntervalToString(tc.interval)
		require.ErrorIsf(t, err, tc.err, "IntervalToString must return the expected error for %s", tc.interval)
		assert.Equalf(t, tc.exp, got, "IntervalToString should map %s", tc.interval)
	}
}

// cancelledContext returns a context that is already cancelled, so a request fails before it is sent
func cancelledContext(tb testing.TB) context.Context {
	tb.Helper()
	ctx, cancel := context.WithCancel(tb.Context())
	cancel()
	return ctx
}

func TestSetTimeRange(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1765900000000), time.UnixMilli(1766000000000)
	for _, tc := range []struct {
		name       string
		start, end time.Time
		exp        url.Values
		err        error
	}{
		{name: "unset", exp: url.Values{}},
		{name: "start", start: start, exp: url.Values{"startAt": {"1765900000000"}}},
		{name: "end", end: end, exp: url.Values{"endAt": {"1766000000000"}}},
		{name: "both", start: start, end: end, exp: url.Values{"startAt": {"1765900000000"}, "endAt": {"1766000000000"}}},
		{name: "start after end", start: end, end: start, exp: url.Values{}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params := url.Values{}
			require.ErrorIs(t, setTimeRange(params, "startAt", "endAt", tc.start, tc.end), tc.err, "setTimeRange must return the expected error")
			assert.Equal(t, tc.exp, params, "setTimeRange should set the ends given")
		})
	}
	params := url.Values{}
	require.NoError(t, setTimeRange(params, "startTime", "endTime", start, end), "setTimeRange must not error for other keys")
	assert.Equal(t, url.Values{"startTime": {"1765900000000"}, "endTime": {"1766000000000"}}, params, "setTimeRange should set the endpoint's keys")
}

func TestHMACSign(t *testing.T) {
	t.Parallel()
	// KuCoin's broker signing example
	assert.Equal(t, "CN1imIGUz/USkPuhOtGWi5DlZ08VeuVfknJNOPqUEac=", hmacSign("1680885532722goodbroker6422da9c97b45100018c6e62", "e8512b82-a4aa"), "hmacSign should match the docs' example signature")
}

func TestSignaturePath(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		path, exp string
	}{
		{"/v1/accounts", "/v1/accounts"},
		{"/v1/accounts/ledgers?currency=BTC%2CUSDT", "/v1/accounts/ledgers?currency=BTC,USDT"},
		// The authentication docs' own example
		{"/v1/sub/api-key?apiKey=67%2Ab3&passphrase=abc%21%40%2311&subName=test", "/v1/sub/api-key?apiKey=67*b3&passphrase=abc!@#11&subName=test"},
		{"/v1/orders?remark=a+b", "/v1/orders?remark=a b"},
	} {
		got, err := signaturePath(tc.path)
		require.NoErrorf(t, err, "signaturePath must not error for %s", tc.path)
		assert.Equalf(t, tc.exp, got, "signaturePath should unescape the query of %s", tc.path)
	}
	_, err := signaturePath("/v1/accounts?currency=%zz")
	var escapeErr url.EscapeError
	assert.ErrorAs(t, err, &escapeErr, "signaturePath should return the unescaping error of a malformed query")
}

// signedRequest is what a local server records of a request
type signedRequest struct {
	method, requestURI, body              string
	key, sign, timestamp, passphrase, ver string
}

// newSigningTestInstance returns an instance with placeholder credentials whose REST hosts point at a local TLS server
// answering with reply, and the requests the server records
func newSigningTestInstance(t *testing.T, reply string) (*Exchange, *[]signedRequest) {
	t.Helper()
	var mu sync.Mutex
	var got []signedRequest
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "the request body should read")
		mu.Lock()
		got = append(got, signedRequest{
			method: r.Method, requestURI: r.RequestURI, body: string(body),
			key: r.Header.Get("KC-API-KEY"), sign: r.Header.Get("KC-API-SIGN"), timestamp: r.Header.Get("KC-API-TIMESTAMP"),
			passphrase: r.Header.Get("KC-API-PASSPHRASE"), ver: r.Header.Get("KC-API-KEY-VERSION"),
		})
		mu.Unlock()
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(srv.Close)
	ex := testInstance(t)
	require.NoError(t, ex.SetHTTPClient(srv.Client()), "SetHTTPClient must not error")
	for _, ep := range []exchange.URL{exchange.RestSpot, exchange.RestFutures, exchange.RestSpotSupplementary} {
		require.NoErrorf(t, ex.API.Endpoints.SetRunningURL(ep.String(), srv.URL+"/api"), "SetRunningURL must not error for %s", ep)
	}
	ex.API.AuthenticatedSupport = true
	ex.API.CredentialsValidator.RequiresBase64DecodeSecret = false
	ex.SetCredentials(&accounts.Credentials{Key: "test-key", Secret: "test-secret", ClientID: "test-passphrase"})
	return ex, &got
}

// sentRequest returns the request call sends to a local server answering success without data. The recorded mock
// matches a body's values whatever their JSON type, so tests pin the JSON types KuCoin documents against this server
func sentRequest(t *testing.T, call func(*Exchange) error) signedRequest {
	t.Helper()
	ex, got := newSigningTestInstance(t, `{"code":"200000","data":null}`)
	require.NoError(t, call(ex), "the request must not error against the local server")
	require.Len(t, *got, 1, "the local server must receive one request")
	return (*got)[0]
}

// testSignature returns the base64 HMAC-SHA256 of message keyed by the placeholder secret
func testSignature(message string) string {
	h := hmac.New(sha256.New, []byte("test-secret"))
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func TestSendHTTPRequest(t *testing.T) {
	t.Parallel()
	ex, got := newSigningTestInstance(t, `{"code":"200000","data":{"value":1}}`)
	var notPointer map[string]int
	assert.ErrorIs(t, ex.SendHTTPRequest(t.Context(), exchange.RestSpot, getCurrentServerTimeEPL, "/v1/timestamp", notPointer), errInvalidResultInterface, "SendHTTPRequest should reject a result that is not a pointer")
	assert.ErrorIs(t, ex.SendHTTPRequest(t.Context(), exchange.Invalid, getCurrentServerTimeEPL, "/v1/timestamp", new(map[string]int)), exchange.ErrEndpointPathNotFound, "SendHTTPRequest should reject an unknown host")
	assert.ErrorIs(t, ex.SendHTTPRequest(cancelledContext(t), exchange.RestSpot, getCurrentServerTimeEPL, "/v1/timestamp", new(map[string]int)), context.Canceled, "SendHTTPRequest should return the transport's error")

	var resp map[string]int
	require.NoError(t, ex.SendHTTPRequest(t.Context(), exchange.RestSpot, getCurrentServerTimeEPL, "/v1/timestamp?symbol=BTC-USDT", &resp), "SendHTTPRequest must not error")
	assert.Equal(t, map[string]int{"value": 1}, resp, "SendHTTPRequest should decode the data field")
	require.Len(t, *got, 1, "the server must receive one request")
	assert.Equal(t, signedRequest{method: http.MethodGet, requestURI: "/api/v1/timestamp?symbol=BTC-USDT"}, (*got)[0], "SendHTTPRequest should send an unsigned GET")

	ex, _ = newSigningTestInstance(t, `{"code":"400100","msg":"Parameter error"}`)
	assert.ErrorContains(t, ex.SendHTTPRequest(t.Context(), exchange.RestSpot, getCurrentServerTimeEPL, "/v1/timestamp", &resp), "400100", "SendHTTPRequest should return KuCoin's error code")

	ex, _ = newSigningTestInstance(t, `null`)
	var err error
	require.NotPanics(t, func() {
		err = ex.SendHTTPRequest(t.Context(), exchange.RestSpot, getCurrentServerTimeEPL, "/v1/timestamp", &resp)
	}, "SendHTTPRequest must not panic on a null body")
	assert.Error(t, err, "SendHTTPRequest should reject a null body, which carries no code")
}

func TestSendAuthHTTPRequest(t *testing.T) {
	t.Parallel()
	ex, got := newSigningTestInstance(t, `{"code":"200000","data":{"value":1}}`)
	var notPointer map[string]int
	assert.ErrorIs(t, ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, getAccountSummaryInformationEPL, http.MethodGet, "/v2/user-info", nil, notPointer), errInvalidResultInterface, "SendAuthHTTPRequest should reject a result that is not a pointer")
	var nilPointer *map[string]int
	assert.ErrorIs(t, ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, getAccountSummaryInformationEPL, http.MethodGet, "/v2/user-info", nil, nilPointer), errInvalidResponseReceiver, "SendAuthHTTPRequest should reject a nil result")
	assert.ErrorIs(t, ex.SendAuthHTTPRequest(t.Context(), exchange.Invalid, getAccountSummaryInformationEPL, http.MethodGet, "/v2/user-info", nil, new(map[string]int)), exchange.ErrEndpointPathNotFound, "SendAuthHTTPRequest should reject an unknown host")
	assert.Error(t, ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, getAccountSummaryInformationEPL, http.MethodPost, "/v1/orders", func() {}, new(map[string]int)), "SendAuthHTTPRequest should return the body's encoding error")
	var escapeErr url.EscapeError
	malformedQuery := "/v1/accounts?currency=%zz"
	assert.ErrorAs(t, ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, getAccountSummaryInformationEPL, http.MethodGet, malformedQuery, nil, new(map[string]int)), &escapeErr, "SendAuthHTTPRequest should return the error unescaping a malformed query")
	noCredentials := testInstance(t)
	noCredentials.SkipAuthCheck = false
	noCredentials.API.AuthenticatedSupport = true
	noCredentials.SetCredentials(&accounts.Credentials{})
	assert.ErrorIs(t, noCredentials.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, getAccountSummaryInformationEPL, http.MethodGet, "/v2/user-info", nil, new(map[string]int)), exchange.ErrCredentialsAreEmpty, "SendAuthHTTPRequest should need credentials")
	require.Empty(t, *got, "the server must receive no request that failed before sending")

	var resp map[string]int
	// The docs' signing example: the query goes out encoded but is signed as it was before encoding
	path := common.EncodeURLValues("/v1/sub/api-key", url.Values{"apiKey": {"67*b3"}, "passphrase": {"abc!@#11"}, "subName": {"test"}})
	require.NoError(t, ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, getAccountSummaryInformationEPL, http.MethodDelete, path, nil, &resp), "SendAuthHTTPRequest must not error")
	assert.Equal(t, map[string]int{"value": 1}, resp, "SendAuthHTTPRequest should decode the data field")
	require.NoError(t, ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, getAccountSummaryInformationEPL, http.MethodPost, "/v1/orders", map[string]any{"currencies": "BTC,USDT"}, &resp), "SendAuthHTTPRequest must not error for a body")
	require.Len(t, *got, 2, "the server must receive both requests")
	passphrase := testSignature("test-passphrase")
	for i, exp := range []struct {
		method, requestURI, body, signed string
	}{
		{http.MethodDelete, "/api/v1/sub/api-key?apiKey=67%2Ab3&passphrase=abc%21%40%2311&subName=test", "", "/api/v1/sub/api-key?apiKey=67*b3&passphrase=abc!@#11&subName=test"},
		{http.MethodPost, "/api/v1/orders", `{"currencies":"BTC,USDT"}`, `/api/v1/orders{"currencies":"BTC,USDT"}`},
	} {
		r := (*got)[i]
		assert.Equalf(t, exp.method, r.method, "request %d should use its method", i)
		assert.Equalf(t, exp.requestURI, r.requestURI, "request %d should send the encoded query", i)
		assert.Equalf(t, exp.body, r.body, "request %d should send its body", i)
		assert.Equalf(t, "test-key", r.key, "request %d should send the API key", i)
		assert.Equalf(t, "3", r.ver, "request %d should send the key version", i)
		assert.Equalf(t, passphrase, r.passphrase, "request %d should send the signed passphrase", i)
		assert.Equalf(t, testSignature(r.timestamp+exp.method+exp.signed), r.sign, "request %d should be signed over the unescaped path and body", i)
	}

	ex, _ = newSigningTestInstance(t, `{"code":"400100","msg":"Parameter error"}`)
	assert.ErrorContains(t, ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, getAccountSummaryInformationEPL, http.MethodGet, "/v2/user-info", nil, &resp), "400100", "SendAuthHTTPRequest should return KuCoin's error code")

	// A null body carries no code, so it is an error, whether the result is decoded from data or from the whole reply
	ex, _ = newSigningTestInstance(t, `null`)
	for name, result := range map[string]any{"data": &resp, "whole reply": &Response{Data: &resp}} {
		var err error
		require.NotPanicsf(t, func() {
			err = ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpot, getAccountSummaryInformationEPL, http.MethodGet, "/v2/user-info", nil, result)
		}, "SendAuthHTTPRequest must not panic on a null body decoded into %s", name)
		assert.Errorf(t, err, "SendAuthHTTPRequest should reject a null body decoded into %s", name)
	}
}
