package gate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/asset"
	"github.com/thrasher-corp/gocryptotrader/exchange/kline"
	"github.com/thrasher-corp/gocryptotrader/exchange/order"
	"github.com/thrasher-corp/gocryptotrader/exchange/request"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const canManipulateRealOrders = false

// errRefusedDecode is what refusingDecoder returns, so a test can follow a decode error through a sender
var errRefusedDecode = errors.New("decode refused")

// refusingDecoder refuses every reply, standing in for a result a reply cannot decode into
type refusingDecoder struct{}

func (*refusingDecoder) UnmarshalJSON([]byte) error {
	return errRefusedDecode
}

const (
	// fractionalSeconds holds part of a second, which Gate refuses where it reads a duration in whole seconds
	fractionalSeconds = types.DurationSeconds(1500 * time.Millisecond)
	// negativeSeconds is a whole number of seconds below zero, which Gate gives no meaning where it reads a duration
	negativeSeconds = types.DurationSeconds(-time.Second)
)

// apiCredentials holds the credentials used for due diligence testing; please supply your own
var apiCredentials = &accounts.Credentials{
	Key:    "",
	Secret: "",
}

var (
	e *Exchange
	// enabledAssetPair holds each asset's test pair, set by TestMain for mock tests and by ensureTestPairs for live tests
	enabledAssetPair map[asset.Item]currency.Pair
)

// mockTestPairs returns the pair each asset's recorded fixtures in testdata/http.json are keyed on
func mockTestPairs() map[asset.Item]currency.Pair {
	btcUSDT := currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
	return map[asset.Item]currency.Pair{
		asset.Spot:                btcUSDT,
		asset.Margin:              btcUSDT,
		asset.CrossMargin:         btcUSDT,
		asset.USDTMarginedFutures: btcUSDT,
		asset.CoinMarginedFutures: currency.NewPairWithDelimiter("BTC", "USD", currency.UnderscoreDelimiter),
		asset.DeliveryFutures:     currency.NewPairWithDelimiter("BTC", "USDT_20260626", currency.UnderscoreDelimiter),
		asset.Options:             currency.NewPairWithDelimiter("BTC", "USDT-20260925-75000-P", currency.UnderscoreDelimiter),
	}
}

// storeTestPairs makes the mock test pairs the only available and enabled pairs of each asset
func storeTestPairs(e *Exchange) error {
	for a, p := range mockTestPairs() {
		if err := e.CurrencyPairs.StorePairs(a, []currency.Pair{p}, false); err != nil {
			return err
		}
		if err := e.CurrencyPairs.StorePairs(a, []currency.Pair{p}, true); err != nil {
			return err
		}
	}
	return nil
}

func getTime() (startTime, endTime time.Time) {
	if mockTests {
		return time.UnixMilli(1744103854944), time.UnixMilli(1744190254944)
	}
	return time.Now().Add(-time.Hour * 48), time.Now()
}

func TestSetUnixTimeRangeParams(t *testing.T) {
	t.Parallel()
	from := time.Unix(1710000000, 0)
	to := from.Add(time.Hour)
	for _, tc := range []struct {
		name           string
		from           time.Time
		to             time.Time
		expectedParams url.Values
		expectedErr    error
	}{
		{
			name:           "both set",
			from:           from,
			to:             to,
			expectedParams: url.Values{"from": {strconv.FormatInt(from.Unix(), 10)}, "to": {strconv.FormatInt(to.Unix(), 10)}},
		},
		{
			name:           "from only",
			from:           from,
			expectedParams: url.Values{"from": {strconv.FormatInt(from.Unix(), 10)}},
		},
		{
			name:           "to only",
			to:             to,
			expectedParams: url.Values{"to": {strconv.FormatInt(to.Unix(), 10)}},
		},
		{
			name:           "both zero",
			expectedParams: url.Values{},
		},
		{
			name:           "start after end",
			from:           to,
			to:             from,
			expectedParams: url.Values{},
			expectedErr:    common.ErrStartAfterEnd,
		},
		{
			name:           "start equals end",
			from:           from,
			to:             from,
			expectedParams: url.Values{"from": {strconv.FormatInt(from.Unix(), 10)}, "to": {strconv.FormatInt(from.Unix(), 10)}},
		},
		{
			name:           "start after current time",
			from:           time.Date(2222, 1, 1, 0, 0, 0, 0, time.UTC),
			to:             time.Date(2222, 1, 2, 0, 0, 0, 0, time.UTC),
			expectedParams: url.Values{},
			expectedErr:    common.ErrStartAfterTimeNow,
		},
		{
			name:           "equal times after current time",
			from:           time.Date(2222, 1, 1, 0, 0, 0, 0, time.UTC),
			to:             time.Date(2222, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedParams: url.Values{},
			expectedErr:    common.ErrStartAfterTimeNow,
		},
		{
			name:           "unix epoch",
			from:           time.Unix(0, 0),
			to:             to,
			expectedParams: url.Values{},
			expectedErr:    common.ErrDateUnset,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params := url.Values{}
			err := setUnixTimeRangeParams(&params, tc.from, tc.to)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr, "setUnixTimeRangeParams must return the expected error")
			} else {
				require.NoError(t, err, "setUnixTimeRangeParams must not error")
			}
			assert.Equal(t, tc.expectedParams, params, "params should match expected values")
		})
	}
}

func TestUnixSeconds(t *testing.T) {
	t.Parallel()
	assert.Equal(t, uint64(1767153378), unixSeconds(time.Unix(1767153378, 999999999)), "unixSeconds should return whole Unix seconds")
	assert.Zero(t, unixSeconds(time.Unix(0, 0)), "unixSeconds should return 0 at the Unix epoch")
	assert.Zero(t, unixSeconds(time.Unix(-1, 0)), "unixSeconds should return 0 for a time before 1970")
	assert.Zero(t, unixSeconds(time.Time{}), "unixSeconds should return 0 for a zero time")
}

func TestSendUnversionedHTTPRequest(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body string
		switch r.URL.Path {
		case "/apiw/v2/ok":
			body = `{"value":"pool"}`
		case "/apiw/v2/coded":
			body = `{"label":"INVALID_KEY","code":"10001","message":"invalid key"}`
		case "/apiw/v2/labelled":
			body = `{"Label":"BAD_REQUEST","Message":"meme token not found"}`
		case "/apiw/v2/status":
			body = `{"code":4,"msg":"boom","trace_id":"trace-1","data":null}`
		default:
			assert.Failf(t, "unexpected request path", "request path %q should be one the test sends", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Write should not error")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.EdgeCase1.String(), server.URL), "SetRunningURL must not error")

	var result struct {
		Value string `json:"value"`
	}
	require.NoError(t, ex.sendUnversionedHTTPRequest(t.Context(), exchange.EdgeCase1, publicIsolatedMarginPoolLoansEPL, "/apiw/v2/ok", &result), "sendUnversionedHTTPRequest must not error")
	assert.Equal(t, "pool", result.Value, "sendUnversionedHTTPRequest should decode the response at the unprefixed path")

	assert.NoError(t, ex.sendUnversionedHTTPRequest(t.Context(), exchange.EdgeCase1, publicIsolatedMarginPoolLoansEPL, "/apiw/v2/ok", nil), "sendUnversionedHTTPRequest should not error without a result")

	ctx := t.Context()
	// Cases share ex's server and the result decoded above, and run in order
	for _, tc := range []struct {
		name   string
		result any
	}{
		{name: "coded error with a result", result: &result},
		{name: "coded error without a result", result: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ex.sendUnversionedHTTPRequest(ctx, exchange.EdgeCase1, publicIsolatedMarginPoolLoansEPL, "/apiw/v2/coded", tc.result)
			require.Error(t, err, "sendUnversionedHTTPRequest must error on a coded error body")
			assert.Contains(t, err.Error(), "label: INVALID_KEY code: 10001", "error should carry the error label and code")
		})
	}

	var list []struct {
		Value string `json:"value"`
	}
	err := ex.sendUnversionedHTTPRequest(t.Context(), exchange.EdgeCase1, publicIsolatedMarginPoolLoansEPL, "/apiw/v2/labelled", &list)
	require.Error(t, err, "sendUnversionedHTTPRequest must error on a labelled error body in place of the result")
	assert.Contains(t, err.Error(), "label: BAD_REQUEST message: meme token not found", "error should carry the error label and message")

	err = ex.sendUnversionedHTTPRequest(t.Context(), exchange.EdgeCase1, publicIsolatedMarginPoolLoansEPL, "/apiw/v2/labelled", new(refusingDecoder))
	assert.ErrorIs(t, err, errRefusedDecode, "sendUnversionedHTTPRequest should keep the decode error of a labelled error body")

	var status *BotCreateResponse
	err = ex.sendUnversionedHTTPRequest(t.Context(), exchange.EdgeCase1, publicIsolatedMarginPoolLoansEPL, "/apiw/v2/status", &status)
	require.Error(t, err, "sendUnversionedHTTPRequest must surface an error carried in the response body")
	assert.Contains(t, err.Error(), "trace-1", "error should carry the trace ID")

	err = ex.sendUnversionedHTTPRequest(t.Context(), exchange.RestUSDCMargined, publicIsolatedMarginPoolLoansEPL, "/apiw/v2/ok", &result)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "sendUnversionedHTTPRequest should error on an endpoint the exchange has no URL for")

	for _, tc := range []struct {
		name, path, served string
	}{
		{name: "versioned path under an endpoint with a path", path: gateAPIVersion + "spot/time", served: "/gateway/spot/time"},
		{name: "unversioned path under an endpoint with a path", path: "apiw/v2/ok", served: "/gateway/apiw/v2/ok"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			routed := newRoutedExchange(t, map[string]string{tc.served: `{"value":"routed"}`})
			routeEndpoint(t, routed, "/gateway/")
			var reply struct {
				Value string `json:"value"`
			}
			require.NoError(t, routed.sendUnversionedHTTPRequest(t.Context(), exchange.RestSpot, publicIsolatedMarginPoolLoansEPL, tc.path, &reply), "sendUnversionedHTTPRequest must reach the path under the endpoint")
			assert.Equal(t, "routed", reply.Value, "sendUnversionedHTTPRequest should decode the reply to the path under the endpoint")
		})
	}
}

func TestJoinPath(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, endpoint, path, expected string
	}{
		{name: "bare host", endpoint: "https://api.gateio.ws", path: "/api/v4/spot/time", expected: "https://api.gateio.ws/api/v4/spot/time"},
		{name: "bare host with a port", endpoint: "http://localhost:8080", path: "/api/v4/spot/time", expected: "http://localhost:8080/api/v4/spot/time"},
		{name: "host ending in a slash", endpoint: "https://proxy.example/", path: "/api/v4/spot/time", expected: "https://proxy.example/spot/time"},
		{name: "host ending in slashes", endpoint: "http://localhost:8080//", path: "/api/v4/spot/time", expected: "http://localhost:8080//spot/time"},
		{name: "endpoint saved with the version prefix", endpoint: "https://api.gateio.ws/api/v4/", path: "/api/v4/spot/time", expected: "https://api.gateio.ws/api/v4/spot/time"},
		{name: "versioned proxy route without a trailing slash", endpoint: "https://proxy.example/gate-fx/api/v4", path: "/api/v4/futures/usdt/contracts", expected: "https://proxy.example/gate-fx/api/v4/futures/usdt/contracts"},
		{name: "proxy route", endpoint: "https://proxy.example/gateway/", path: "/api/v4/spot/time", expected: "https://proxy.example/gateway/spot/time"},
		{name: "proxy route without a trailing slash", endpoint: "https://proxy.example/gateway", path: "/api/v4/spot/time", expected: "https://proxy.example/gateway/spot/time"},
		{name: "proxy route with a query", endpoint: "https://proxy.example/gateway/", path: "/api/v4/futures/usdt/candlesticks?contract=BTC_USDT&interval=1m", expected: "https://proxy.example/gateway/futures/usdt/candlesticks?contract=BTC_USDT&interval=1m"},
		{name: "unversioned path", endpoint: "https://www.gate.com/apiw/v2/", path: "spot_loan/margin/margin_loan_info", expected: "https://www.gate.com/apiw/v2/spot_loan/margin/margin_loan_info"},
		{name: "unversioned path holding the version prefix further in", endpoint: "https://www.gate.com/apiw/v2/", path: "zone/api/v4/info", expected: "https://www.gate.com/apiw/v2/zone/api/v4/info"},
		{name: "unversioned path on a bare host", endpoint: "https://proxy.example", path: "/apiw/v2/ok", expected: "https://proxy.example/apiw/v2/ok"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, joinPath(tc.endpoint, tc.path), "joinPath should join the path to the endpoint")
		})
	}
}

func TestIsBareHost(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		endpoint string
		expected bool
	}{
		{endpoint: "https://api.gateio.ws", expected: true},
		{endpoint: "http://localhost:8080", expected: true},
		{endpoint: "https://proxy.example:8443/gateway/"},
		{endpoint: "https://proxy.example/"},
		{endpoint: "https://proxy.example/gateway"},
		{endpoint: "https://api.gateio.ws/api/v4/"},
		{endpoint: "api.gateio.ws"},
	} {
		t.Run(tc.endpoint, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, isBareHost(tc.endpoint), "isBareHost should report whether the endpoint is a scheme and host alone")
		})
	}
}

func TestSendHTTPRequest(t *testing.T) {
	t.Parallel()

	t.Run("versioned path", func(t *testing.T) {
		t.Parallel()
		ex := newRoutedExchange(t, map[string]string{"/api/v4/spot/time": `{"server_time":1774386128160}`})
		var resp serverTimeReply
		require.NoError(t, ex.SendHTTPRequest(t.Context(), exchange.RestSpot, publicGetServerTimeEPL, "spot/time", &resp), "SendHTTPRequest must not error")
		assert.Equal(t, time.UnixMilli(1774386128160), resp.ServerTime.Time(), "SendHTTPRequest should decode the reply to the path under the API version")
	})

	for _, route := range []string{"/gateway/", "/api/v4/", "/"} {
		t.Run("endpoint with the path "+route, func(t *testing.T) {
			t.Parallel()
			ex := newRoutedExchange(t, map[string]string{route + "spot/time": `{"server_time":1774386128160}`})
			routeEndpoint(t, ex, route)
			var resp serverTimeReply
			require.NoError(t, ex.SendHTTPRequest(t.Context(), exchange.RestSpot, publicGetServerTimeEPL, "spot/time", &resp), "SendHTTPRequest must not error")
			assert.Equal(t, time.UnixMilli(1774386128160), resp.ServerTime.Time(), "SendHTTPRequest should decode the reply to the path under the endpoint")
		})
	}

	t.Run("endpoint without a URL", func(t *testing.T) {
		t.Parallel()
		ex := newRoutedExchange(t, nil)
		err := ex.SendHTTPRequest(t.Context(), exchange.RestUSDCMargined, publicGetServerTimeEPL, "spot/time", new(serverTimeReply))
		assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "SendHTTPRequest should error on an endpoint the exchange has no URL for")
	})

	t.Run("server time from the API", func(t *testing.T) {
		t.Parallel()
		var resp serverTimeReply
		require.NoError(t, e.SendHTTPRequest(t.Context(), exchange.RestSpot, publicGetServerTimeEPL, "spot/time", &resp), "SendHTTPRequest must not error")
		if mockTests {
			assert.Equal(t, time.UnixMilli(1774386128160), resp.ServerTime.Time(), "SendHTTPRequest should decode the recorded server time")
			return
		}
		assert.WithinDuration(t, time.Now(), resp.ServerTime.Time(), time.Minute, "SendHTTPRequest should decode the current server time")
	})
}

func TestSendPublicPostRequest(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "request method should be POST")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "request should carry a JSON body")
		assert.Empty(t, r.Header.Get("KEY"), "request should not be signed")
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err, "ReadAll should not error")
		assert.JSONEq(t, `{"contracts":["BTC_USDT"]}`, string(body), "request body should be the marshalled data")
		var resp string
		switch r.URL.Path {
		case "/api/v4/ok":
			resp = `[{"contract":"BTC_USDT"}]`
		case "/api/v4/label":
			resp = `{"label":"INVALID_PARAM_VALUE","message":"invalid contract"}`
		case "/api/v4/coded":
			resp = `{"code":4,"msg":"boom","trace_id":"trace-1"}`
		case "/api/v4/mistyped":
			resp = `[{"contract":1}]`
		default:
			resp = `{`
		}
		_, err = w.Write([]byte(resp))
		assert.NoError(t, err, "Write should not error")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")
	body := &struct {
		Contracts []string `json:"contracts"`
	}{Contracts: []string{"BTC_USDT"}}

	var result []struct {
		Contract string `json:"contract"`
	}
	require.NoError(t, ex.sendPublicPostRequest(t.Context(), publicBatchFundingRatesEPL, "ok", body, &result), "sendPublicPostRequest must not error")
	assert.Equal(t, []struct {
		Contract string `json:"contract"`
	}{{Contract: "BTC_USDT"}}, result, "sendPublicPostRequest should decode the response")

	err := ex.sendPublicPostRequest(t.Context(), publicBatchFundingRatesEPL, "label", body, &result)
	require.Error(t, err, "sendPublicPostRequest must error on a labelled error body")
	assert.Contains(t, err.Error(), "label: INVALID_PARAM_VALUE message: invalid contract", "error should carry the error label and message")

	var coded *BotCreateResponse
	err = ex.sendPublicPostRequest(t.Context(), publicBatchFundingRatesEPL, "coded", body, &coded)
	require.Error(t, err, "sendPublicPostRequest must surface an error carried in the response body")
	assert.Contains(t, err.Error(), "trace-1", "error should carry the trace ID")

	require.Error(t, ex.sendPublicPostRequest(t.Context(), publicBatchFundingRatesEPL, "malformed", body, &result), "sendPublicPostRequest must error on a malformed response")

	err = ex.sendPublicPostRequest(t.Context(), publicBatchFundingRatesEPL, "mistyped", body, &result)
	require.Error(t, err, "sendPublicPostRequest must error on a response that does not decode")
	assert.NotContains(t, err.Error(), "label:", "error should not carry a label the reply lacks")

	assert.Error(t, ex.sendPublicPostRequest(t.Context(), publicBatchFundingRatesEPL, "ok", make(chan int), &result), "sendPublicPostRequest should error on a body it cannot marshal")

	unset := new(Exchange)
	require.NoError(t, testexch.Setup(unset), "Setup must not error")
	unset.API.Endpoints = unset.NewEndpoints()
	err = unset.sendPublicPostRequest(t.Context(), publicBatchFundingRatesEPL, "ok", body, &result)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "sendPublicPostRequest should error without a REST URL")

	t.Run("endpoint with a path", func(t *testing.T) {
		t.Parallel()
		ex := newRoutedExchange(t, map[string]string{"/gateway/ok": `[{"contract":"BTC_USDT"}]`})
		routeEndpoint(t, ex, "/gateway/")
		var routed []struct {
			Contract string `json:"contract"`
		}
		require.NoError(t, ex.sendPublicPostRequest(t.Context(), publicBatchFundingRatesEPL, "ok", body, &routed), "sendPublicPostRequest must not error")
		require.Len(t, routed, 1, "sendPublicPostRequest must decode the reply to the path under the endpoint")
		assert.Equal(t, "BTC_USDT", routed[0].Contract, "sendPublicPostRequest should decode the reply to the path under the endpoint")
	})
}

func TestSendAuthenticatedHTTPRequest(t *testing.T) {
	t.Parallel()
	ex := newRoutedExchange(t, map[string]string{
		"/api/v4/ok":       `{"value":"pool"}`,
		"/api/v4/coded":    `{"label":"INVALID_KEY","code":"10001","message":"invalid key"}`,
		"/api/v4/labelled": `{"Label":"BAD_REQUEST","Message":"meme token not found"}`,
		"/api/v4/mistyped": `{"value":1}`,
	})
	var result struct {
		Value string `json:"value"`
	}
	require.NoError(t, ex.SendAuthenticatedHTTPRequest(t.Context(), exchange.RestSpot, spotAccountsEPL, http.MethodGet, "ok", nil, nil, &result), "SendAuthenticatedHTTPRequest must not error")
	assert.Equal(t, "pool", result.Value, "SendAuthenticatedHTTPRequest should decode the response")

	err := ex.SendAuthenticatedHTTPRequest(t.Context(), exchange.RestSpot, spotAccountsEPL, http.MethodGet, "coded", nil, nil, &result)
	require.Error(t, err, "SendAuthenticatedHTTPRequest must error on a coded error body")
	assert.Contains(t, err.Error(), "label: INVALID_KEY code: 10001 message: invalid key", "error should carry the error label, code and message")

	var list []struct {
		Value string `json:"value"`
	}
	err = ex.SendAuthenticatedHTTPRequest(t.Context(), exchange.RestSpot, spotAccountsEPL, http.MethodGet, "labelled", nil, nil, &list)
	require.ErrorIs(t, err, request.ErrAuthRequestFailed, "SendAuthenticatedHTTPRequest must fail a labelled error body in place of the result")
	assert.Contains(t, err.Error(), "label: BAD_REQUEST message: meme token not found", "error should carry the error label and message")

	err = ex.SendAuthenticatedHTTPRequest(t.Context(), exchange.RestSpot, spotAccountsEPL, http.MethodGet, "labelled", nil, nil, new(refusingDecoder))
	require.ErrorIs(t, err, request.ErrAuthRequestFailed, "SendAuthenticatedHTTPRequest must fail a labelled error body the result refuses")
	assert.ErrorIs(t, err, errRefusedDecode, "SendAuthenticatedHTTPRequest should keep the decode error of a labelled error body")

	err = ex.SendAuthenticatedHTTPRequest(t.Context(), exchange.RestSpot, spotAccountsEPL, http.MethodGet, "mistyped", nil, nil, &result)
	require.ErrorIs(t, err, request.ErrAuthRequestFailed, "SendAuthenticatedHTTPRequest must fail a reply that does not decode")
	assert.NotContains(t, err.Error(), "label:", "error should not carry a label the reply lacks")

	err = ex.SendAuthenticatedHTTPRequest(t.Context(), exchange.RestUSDCMargined, spotAccountsEPL, http.MethodGet, "ok", nil, nil, &result)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "SendAuthenticatedHTTPRequest should error on an endpoint the exchange has no URL for")

	assert.Error(t, ex.SendAuthenticatedHTTPRequest(t.Context(), exchange.RestSpot, spotAccountsEPL, http.MethodPost, "ok", nil, make(chan int), &result), "SendAuthenticatedHTTPRequest should error on a body it cannot marshal")

	noCredentials := new(Exchange)
	require.NoError(t, testexch.Setup(noCredentials), "Setup must not error")
	noCredentials.API.AuthenticatedSupport = true
	err = noCredentials.SendAuthenticatedHTTPRequest(t.Context(), exchange.RestSpot, spotAccountsEPL, http.MethodGet, "ok", nil, nil, &result)
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "SendAuthenticatedHTTPRequest should error without credentials")

	noContent := newRoutedExchange(t, nil)
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Status", "204")
		if r.URL.Path == "/api/v4/stray-body" {
			_, err := fmt.Fprint(w, `{"value":"pool"}`)
			assert.NoError(t, err, "writing the response should not error")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	require.NoError(t, noContent.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, noContent.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")
	assert.NoError(t, noContent.SendAuthenticatedHTTPRequest(t.Context(), exchange.RestSpot, spotAccountsEPL, http.MethodDelete, "empty", nil, nil, nil), "SendAuthenticatedHTTPRequest should accept an empty reply when it expects no result")
	err = noContent.SendAuthenticatedHTTPRequest(t.Context(), exchange.RestSpot, spotAccountsEPL, http.MethodDelete, "empty", nil, nil, &result)
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "SendAuthenticatedHTTPRequest should fail an empty reply when it expects a result")
	err = noContent.SendAuthenticatedHTTPRequest(t.Context(), exchange.RestSpot, spotAccountsEPL, http.MethodDelete, "stray-body", nil, nil, nil)
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "SendAuthenticatedHTTPRequest should fail a reply marked empty that carries a body")

	t.Run("endpoint with a path", func(t *testing.T) {
		t.Parallel()
		routed := newServedExchange(t, func(w http.ResponseWriter, r *http.Request) {
			// The route forwards to Gate's versioned API, so the signature covers Gate's path, not the route's
			timestamp, err := strconv.ParseInt(r.Header.Get("TIMESTAMP"), 10, 64)
			assert.NoError(t, err, "TIMESTAMP header should hold the signing time")
			sig, err := new(Exchange).GenerateSignature("mock-api-secret", r.Method, "/api/v4/ok", r.URL.RawQuery, nil, time.Unix(timestamp, 0))
			assert.NoError(t, err, "GenerateSignature should not error")
			if r.URL.Path != "/gateway/ok" || r.URL.RawQuery != "currency=BTC" || r.Header.Get("SIGN") != sig {
				w.WriteHeader(http.StatusNotFound)
				_, err = fmt.Fprint(w, `{"label":"NOT_FOUND","message":""}`)
				assert.NoError(t, err, "writing the response should not error")
				return
			}
			_, err = fmt.Fprint(w, `{"value":"pool"}`)
			assert.NoError(t, err, "writing the response should not error")
		})
		routeEndpoint(t, routed, "/gateway/")
		var reply struct {
			Value string `json:"value"`
		}
		require.NoError(t, routed.SendAuthenticatedHTTPRequest(t.Context(), exchange.RestSpot, spotAccountsEPL, http.MethodGet, "ok", url.Values{"currency": {"BTC"}}, nil, &reply), "SendAuthenticatedHTTPRequest must reach the path under the endpoint, signed for Gate's path")
		assert.Equal(t, "pool", reply.Value, "SendAuthenticatedHTTPRequest should decode the response")
	})
}

func TestDecodeReply(t *testing.T) {
	t.Parallel()
	var list []string
	require.NoError(t, decodeReply(json.RawMessage(`["a"]`), &list), "decodeReply must not error on a reply that decodes")
	assert.Equal(t, []string{"a"}, list, "decodeReply should decode the reply into the result")

	var labelled struct {
		Label string `json:"label"`
	}
	require.NoError(t, decodeReply(json.RawMessage(`{"label":"BAD_REQUEST"}`), &labelled), "decodeReply must not error on a labelled reply that decodes into the result")
	assert.Equal(t, "BAD_REQUEST", labelled.Label, "decodeReply should decode a labelled reply the result can hold")

	reply := json.RawMessage(`{"Label":"BAD_REQUEST","Message":"meme token not found"}`)
	var unwanted []string
	decodeErr := json.Unmarshal(reply, &unwanted)
	require.Error(t, decodeErr, "a labelled object must not decode into a list")
	var refused []string
	err := decodeReply(reply, &refused)
	require.Error(t, err, "decodeReply must error on a labelled reply that does not decode")
	assert.Equal(t, "label: BAD_REQUEST message: meme token not found: "+decodeErr.Error(), err.Error(), "decodeReply should add the label and message to the decode error")
	wrapped := errors.Unwrap(err)
	require.Error(t, wrapped, "decodeReply must wrap the decode error")
	assert.Equal(t, decodeErr.Error(), wrapped.Error(), "decodeReply should wrap the decode error unchanged")

	assert.ErrorIs(t, decodeReply(reply, new(refusingDecoder)), errRefusedDecode, "decodeReply should keep the decode error of a labelled reply")

	var mistyped []string
	err = decodeReply(json.RawMessage(`{"value":1}`), &mistyped)
	require.Error(t, err, "decodeReply must error on a reply that does not decode")
	assert.NotContains(t, err.Error(), "label:", "decodeReply should not add a label the reply lacks")

	// An error reply that doesn't decode itself, here with a numeric message, is not trusted for its label
	var malformed []string
	err = decodeReply(json.RawMessage(`{"label":"BAD_REQUEST","message":1}`), &malformed)
	require.Error(t, err, "decodeReply must error on a malformed error reply")
	assert.NotContains(t, err.Error(), "label:", "decodeReply should not add the label of an error reply that does not decode")

	for _, tc := range []struct {
		name       string
		unlabelled string
	}{
		{name: "reply without a label", unlabelled: `{"value":1}`},
		{name: "error reply with a numeric message", unlabelled: `{"label":"BAD_REQUEST","message":1}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, decodeReply(json.RawMessage(tc.unlabelled), new(refusingDecoder)), errRefusedDecode, "decodeReply should keep the decode error of %s", tc.unlabelled)
		})
	}
}

// newFixedResponseExchange returns an exchange whose REST requests reach a server answering path with body, signing
// with placeholder credentials
func newFixedResponseExchange(t *testing.T, path, body string) *Exchange {
	t.Helper()
	return newRoutedExchange(t, map[string]string{path: body})
}

// newRoutedExchange returns an exchange whose REST requests reach a server answering each path with its body and every
// other path with a server error, signing with placeholder credentials
func newRoutedExchange(t *testing.T, routes map[string]string) *Exchange {
	t.Helper()
	return newRefusingRoutedExchange(t, "", routes)
}

// newRefusingRoutedExchange is newRoutedExchange that also answers a request whose query contains refused, when set, with
// a server error, so a test can fail a later request of a method that makes several
func newRefusingRoutedExchange(t *testing.T, refused string, routes map[string]string) *Exchange {
	t.Helper()
	return newServedExchange(t, func(w http.ResponseWriter, r *http.Request) {
		body, ok := routes[r.URL.Path]
		if !ok || (refused != "" && strings.Contains(r.URL.RawQuery, refused)) {
			w.WriteHeader(http.StatusInternalServerError)
			body = `{"label":"SERVER_ERROR","message":"unavailable"}`
		}
		_, err := fmt.Fprint(w, body)
		assert.NoError(t, err, "writing the response should not error")
	})
}

// newServedExchange returns an exchange whose REST requests reach a server answering with handler, signing with
// placeholder credentials
func newServedExchange(t *testing.T, handler http.HandlerFunc) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	ex.SkipAuthCheck = true
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	server := httptest.NewTestServer(t, handler)
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	for _, u := range []exchange.URL{exchange.RestSpot, exchange.RestFutures} {
		require.NoError(t, ex.API.Endpoints.SetRunningURL(u.String(), server.URL), "SetRunningURL must not error")
	}
	return ex
}

// routeEndpoint moves the exchange's spot REST endpoint to a path on its server, as a proxy route forwarding to Gate's
// versioned API would be configured
func routeEndpoint(t *testing.T, ex *Exchange, path string) {
	t.Helper()
	host, err := ex.API.Endpoints.GetURL(exchange.RestSpot)
	require.NoError(t, err, "GetURL must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), host+path), "SetRunningURL must not error")
}

// newTestExchange returns a new instance set up on the shared exchange's endpoints, as a live run against the testnet
// needs every instance there, not only the shared one
func newTestExchange(tb testing.TB) *Exchange {
	tb.Helper()
	ex := new(Exchange)
	useSandbox := e.Config.UseSandbox
	require.NoError(tb, testexch.SetupWithConfig(ex, func(c *config.Exchange) { c.UseSandbox = useSandbox }), "Setup must not error")
	return ex
}

// newIsolatedExchange returns an exchange set up as the shared one is but named after the test, so the ticker,
// orderbook, account and limits stores its wrapper calls write cannot collide with another test's
func newIsolatedExchange(t *testing.T) *Exchange {
	t.Helper()
	ex := newTestExchange(t)
	if mockTests {
		ex.API.AuthenticatedSupport = true
		ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
		require.NoError(t, testexch.MockHTTPInstance(ex), "MockHTTPInstance must not error")
		require.NoError(t, storeTestPairs(ex), "storeTestPairs must not error")
	} else {
		ensureTestPairs(t)
		ex.CurrencyPairs.Load(&e.CurrencyPairs)
		if apiCredentials.Key != "" && apiCredentials.Secret != "" {
			ex.API.AuthenticatedSupport = true
			ex.SetCredentials(apiCredentials)
		}
	}
	ex.Name = t.Name()
	return ex
}

// unsignedExchange returns an exchange allowed to send authenticated requests but holding no credentials, so a request
// that passes its checks fails only at signing, before anything is sent
func unsignedExchange(t *testing.T) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.API.AuthenticatedSupport = true
	return ex
}

// TestFuturesTickerUnmarshal pins the tags futuresBaseVolume reads: constructing the fields
// directly cannot catch one of them being renamed on the wire
func TestFuturesTickerUnmarshal(t *testing.T) {
	t.Parallel()
	var tk FuturesTickerResponse
	require.NoError(t, json.Unmarshal([]byte(`{"contract":"UB_USDT","volume_24h":"8819","volume_24h_base":"8819700","volume_24h_quote":"1234","quanto_multiplier":"1000"}`), &tk), "Unmarshal must not error")
	assert.Equal(t, "UB_USDT", tk.Contract.String(), "Contract should unmarshal")
	assert.Equal(t, 8819.0, tk.Volume24Hour.Float64(), "volume_24h should unmarshal")
	assert.Equal(t, 8819700.0, tk.Volume24HourBase.Float64(), "volume_24h_base should unmarshal")
	assert.Equal(t, 1000.0, tk.QuantoMultiplier.Float64(), "quanto_multiplier should unmarshal")
	assert.Equal(t, 8819700.0, futuresBaseVolume(&tk), "futuresBaseVolume should keep the explicit base volume when volume_24h dropped a part contract")
}

func TestGenerateSignature(t *testing.T) { //nolint:tparallel // Cases call the shared exchange and run in order
	t.Parallel()
	// Expected values follow Gate's documented scheme: HMAC-SHA512 over method, path, query, the SHA512 of the body and
	// the Unix timestamp, each separated by a newline
	ts := time.Unix(1700000000, 0)
	// Cases call the shared exchange, so run in order
	for _, tc := range []struct {
		name, method, path, query string
		body                      any
		exp                       string
	}{
		{
			name:   "GET with a query",
			method: http.MethodGet,
			path:   "/api/v4/spot/orders",
			query:  "currency_pair=BTC_USDT&status=open",
			exp:    "777f9b7385b58e88af6ce1aaae82f217f3e3dd0b9406e98522498001de299a9636bc797e3291aa1c49df79d8decf51eca463bd6082937c4c8742175c938eee56",
		},
		{
			name:   "POST with a body",
			method: http.MethodPost,
			path:   "/api/v4/spot/orders",
			body:   map[string]string{"currency_pair": "BTC_USDT"},
			exp:    "dac4c0d44277469bbb4d2dd7297111b89ce2039ebc9b4cebfaa8cd8a361738ca6904a75ebc28df1a1c041af50c104884dddeeae9bbbe89064513a3837363c886",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sig, err := e.GenerateSignature("mock-api-secret", tc.method, tc.path, tc.query, tc.body, ts)
			require.NoErrorf(t, err, "GenerateSignature must not error for %s %s", tc.method, tc.path)
			assert.Equalf(t, tc.exp, sig, "GenerateSignature should sign %s %s as documented", tc.method, tc.path)
		})
	}
}

func TestWalletAccountType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		asset       asset.Item
		expected    string
		expectedErr error
	}{
		{asset: asset.Spot, expected: "spot"},
		{asset: asset.Margin, expected: "margin"},
		{asset: asset.Futures, expected: "futures"},
		{asset: asset.USDTMarginedFutures, expected: "futures"},
		{asset: asset.CoinMarginedFutures, expected: "futures"},
		{asset: asset.DeliveryFutures, expected: "delivery"},
		{asset: asset.Options, expected: "options"},
		{asset: asset.Empty, expectedErr: asset.ErrInvalidAsset},
		{asset: asset.CrossMargin, expectedErr: asset.ErrNotSupported},
	} {
		t.Run(tc.asset.String(), func(t *testing.T) {
			t.Parallel()
			got, err := walletAccountType(tc.asset)
			require.ErrorIs(t, err, tc.expectedErr, "walletAccountType must return the expected error")
			assert.Equal(t, tc.expected, got, "walletAccountType should return the account Gate expects")
		})
	}
}

// futuresLocalServerExchange returns an authenticated exchange whose REST requests reach handler. Mock tests use it for
// endpoints whose JSON bodies the VCR mock server cannot match, which compares nested arrays and objects in a
// recorded body as Go formatted text but in a request body as JSON
func futuresLocalServerExchange(t *testing.T, handler http.HandlerFunc) *Exchange {
	t.Helper()
	server := httptest.NewTestServer(t, handler)
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "SetRunningURL must not error")
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	return ex
}

// futuresFixturePosition returns the position the futures fixtures record for a contract, position mode and size
func futuresFixturePosition(contract currency.Pair, mode string, size types.Number) *PositionResponse {
	return &PositionResponse{
		User:                       10000,
		Contract:                   contract,
		Size:                       size,
		HedgeStatus:                "partial_hedged",
		HedgedSize:                 12,
		UnhedgedSize:               9428,
		RiskLimit:                  100,
		LeverageMax:                100,
		MaintenanceRate:            0.005,
		Value:                      3568.62,
		Margin:                     4.431548146258,
		EntryPrice:                 3779.55,
		LiquidationPrice:           99999999,
		MarkPrice:                  3780.32,
		InitialMargin:              1.5,
		MaintenanceMargin:          0.75,
		UnrealisedPNL:              -0.000507486844,
		RealisedPNL:                0.045543982432,
		RealisedPNLPosition:        0.045543982432,
		RealisedPNLFundingFee:      0.1,
		RealisedPNLTransactionFees: 0.2,
		HistoryPNL:                 0.3,
		LastClosePNL:               0.4,
		RealisedPNLPoint:           0.5,
		RealisedPNLHistoryPoint:    0.6,
		ADLRanking:                 5,
		PendingOrders:              16,
		CloseOrder:                 CloseOrder{ID: 232323, Price: 3779, IsLiquidation: true},
		Mode:                       mode,
		CrossLeverageLimit:         10,
		UpdateTime:                 types.Time(time.Unix(1684994406, 0)),
		OpenTime:                   types.Time(time.Unix(1684990000, 0)),
		UpdateID:                   1,
		RiskLimitTable:             "BIG_HOT_COIN_50X_V2",
		AverageMaintenanceRate:     0.005,
		PositionID:                 7,
		PositionMarginMode:         "isolated",
		PositionLeverage:           30,
	}
}

// deliveryPositionFixture returns the delivery position recorded in testdata/http.json
func deliveryPositionFixture(contract currency.Pair) *PositionResponse {
	return &PositionResponse{
		User:                       10000,
		Contract:                   contract,
		Size:                       -9440,
		RiskLimit:                  100,
		LeverageMax:                100,
		MaintenanceRate:            0.005,
		Value:                      3568.62,
		Margin:                     4.431548146258,
		EntryPrice:                 3779.55,
		LiquidationPrice:           99999999,
		MarkPrice:                  3780.32,
		InitialMargin:              35.7,
		MaintenanceMargin:          17.84,
		UnrealisedPNL:              -0.000507486844,
		RealisedPNL:                0.045497982432,
		RealisedPNLPosition:        0.045543982432,
		RealisedPNLFundingFee:      -0.000012,
		RealisedPNLTransactionFees: -0.000034,
		HistoryPNL:                 1.5,
		LastClosePNL:               0.5,
		ADLRanking:                 5,
		PendingOrders:              16,
		CloseOrder:                 CloseOrder{ID: 232323, Price: 3779},
		Mode:                       "single",
		CrossLeverageLimit:         10,
		UpdateTime:                 types.Time(time.Unix(1684994406, 0)),
		OpenTime:                   types.Time(time.Unix(1684990000, 0)),
		UpdateID:                   1,
		RiskLimitTable:             "BIG_HOT_COIN_50X_V2",
		AverageMaintenanceRate:     0.005,
	}
}

func TestValidateCountdownTimeout(t *testing.T) {
	t.Parallel()
	// Serial, as the refusals below call validateCountdownTimeout after these
	for _, timeout := range []time.Duration{0, 5 * time.Second, time.Hour} {
		t.Run("accepts "+timeout.String(), func(t *testing.T) {
			assert.NoErrorf(t, validateCountdownTimeout(timeout), "validateCountdownTimeout should accept %s", timeout)
		})
	}
	for _, timeout := range []time.Duration{-time.Second, time.Nanosecond, 4999 * time.Millisecond, 5500 * time.Millisecond} {
		t.Run("rejects "+timeout.String(), func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, validateCountdownTimeout(timeout), errInvalidTimeout, "validateCountdownTimeout should reject %s", timeout)
		})
	}
}

func TestCheckWholeSeconds(t *testing.T) {
	t.Parallel()
	// Serial, as the refusals below call checkWholeSeconds after these
	for _, d := range []time.Duration{0, time.Second, 24 * time.Hour} {
		t.Run("accepts "+d.String(), func(t *testing.T) {
			assert.NoErrorf(t, checkWholeSeconds(types.DurationSeconds(d), "expiry"), "checkWholeSeconds should accept %s", d)
		})
	}
	for _, d := range []time.Duration{time.Nanosecond, 1500 * time.Millisecond, -time.Nanosecond, -time.Second, -1500 * time.Millisecond} {
		t.Run("rejects "+d.String(), func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, checkWholeSeconds(types.DurationSeconds(d), "expiry"), errNotWholeSeconds, "checkWholeSeconds should reject %s", d)
		})
	}
}

func TestValidateOrderCreateParams(t *testing.T) {
	t.Parallel()
	contract := currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
	for _, tc := range []struct {
		name          string
		contract      currency.Pair
		size          types.Number
		price         types.Number
		closePosition bool
		autoSize      string
		timeInForce   string
		text          string
		settle        currency.Code
		isRest        bool
		err           error
	}{
		{name: "REST market order", contract: contract, size: 1, timeInForce: iocTIF, settle: currency.USDT, isRest: true},
		{name: "REST limit order settled in any currency", contract: contract, size: -1, price: 100, timeInForce: gtcTIF, text: "t-1", settle: currency.ETH, isRest: true},
		{name: "websocket order without a settlement currency", contract: contract, size: 1, timeInForce: fokTIF},
		{name: "websocket order settled in BTC", contract: contract, size: 1, timeInForce: iocTIF, settle: currency.BTC},
		{name: "websocket order settled in USDT", contract: contract, size: 1, timeInForce: iocTIF, settle: currency.USDT},
		{name: "close with a zero size", contract: contract, closePosition: true, timeInForce: iocTIF, settle: currency.USDT, isRest: true},
		{name: "auto size closing one side", contract: contract, autoSize: "close_short", timeInForce: iocTIF, settle: currency.USDT, isRest: true},
		{name: "empty contract", size: 1, timeInForce: iocTIF, settle: currency.USDT, isRest: true, err: currency.ErrCurrencyPairEmpty},
		{name: "zero size without closing", contract: contract, timeInForce: iocTIF, settle: currency.USDT, isRest: true, err: errInvalidOrderSize},
		{name: "close with a size", contract: contract, size: 1, closePosition: true, timeInForce: iocTIF, settle: currency.USDT, isRest: true, err: errInvalidOrderSize},
		{name: "unknown time in force", contract: contract, size: 1, price: 100, timeInForce: "gtd", settle: currency.USDT, isRest: true, err: order.ErrUnsupportedTimeInForce},
		{name: "market order neither ioc nor fok", contract: contract, size: 1, timeInForce: gtcTIF, settle: currency.USDT, isRest: true, err: order.ErrUnsupportedTimeInForce},
		{name: "market order without a time in force", contract: contract, size: 1, settle: currency.USDT, isRest: true, err: order.ErrUnsupportedTimeInForce},
		{name: "text without the t- prefix", contract: contract, size: 1, timeInForce: iocTIF, text: "custom", settle: currency.USDT, isRest: true, err: errInvalidTextPrefix},
		{name: "unknown auto size", contract: contract, autoSize: "close_both", timeInForce: iocTIF, settle: currency.USDT, isRest: true, err: errInvalidAutoSize},
		{name: "auto size with a size", contract: contract, size: 1, autoSize: "close_long", timeInForce: iocTIF, settle: currency.USDT, isRest: true, err: order.ErrAmountIsInvalid},
		{name: "REST order without a settlement currency", contract: contract, size: 1, timeInForce: iocTIF, isRest: true, err: currency.ErrCurrencyCodeEmpty},
		{name: "websocket order settled in another currency", contract: contract, size: 1, timeInForce: iocTIF, settle: currency.ETH, err: currency.ErrCurrencyCodeEmpty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateOrderCreateParams(tc.contract, tc.size, tc.price, tc.closePosition, tc.autoSize, tc.timeInForce, tc.text, tc.settle, tc.isRest)
			assert.ErrorIs(t, err, tc.err, "validateOrderCreateParams should return the expected error")
		})
	}
}

func TestGetUnderlyingFromCurrencyPair(t *testing.T) {
	t.Parallel()
	uly, err := e.GetUnderlyingFromCurrencyPair(currency.Pair{Delimiter: currency.UnderscoreDelimiter, Base: currency.BTC, Quote: currency.NewCode("USDT_LLK")})
	require.NoError(t, err)
	assert.True(t, uly.Equal(currency.NewBTCUSDT()))

	t.Run("an underlying itself", func(t *testing.T) {
		t.Parallel()
		uly, err := e.GetUnderlyingFromCurrencyPair(currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter))
		require.NoError(t, err, "GetUnderlyingFromCurrencyPair must allow a pair of two currencies")
		assert.True(t, uly.Equal(currency.NewBTCUSDT()), "GetUnderlyingFromCurrencyPair should return the pair itself")
	})
}

func TestParseGateMilliSecTimeUnmarshal(t *testing.T) {
	t.Parallel()
	var timeWhenTesting int64 = 1684981731098
	timeWhenTestingString := `"1684981731098"` // Normal string
	integerJSON := `{"number": 1684981731098}`
	float64JSON := `{"number": 1684981731.098}`

	tm := time.UnixMilli(timeWhenTesting)
	var in types.Time
	err := json.Unmarshal([]byte(timeWhenTestingString), &in)
	require.NoError(t, err)
	require.Equalf(t, tm, in.Time(), "found %v, but expected %v", in.Time(), tm)

	var inInteger struct {
		Number types.Time `json:"number"`
	}
	err = json.Unmarshal([]byte(integerJSON), &inInteger)
	require.NoError(t, err)
	require.Equalf(t, tm, inInteger.Number.Time(), "found %v, but expected %v", inInteger.Number.Time(), tm)

	var inFloat64 struct {
		Number types.Time `json:"number"`
	}
	err = json.Unmarshal([]byte(float64JSON), &inFloat64)
	require.NoError(t, err)
	require.Equalf(t, tm, inFloat64.Number.Time(), "found %v, but expected %v", inFloat64.Number.Time(), tm)
}

func TestParseTimeUnmarshal(t *testing.T) {
	t.Parallel()
	var timeWhenTesting int64 = 1684981731
	timeWhenTestingString := `"1684981731"`
	integerJSON := `{"number": 1684981731}`
	float64JSON := `{"number": 1684981731.234}`
	timeWhenTestingStringMicroSecond := `"1691122380942.173000"`

	whenTime := time.Unix(timeWhenTesting, 0)
	var in types.Time
	err := json.Unmarshal([]byte(timeWhenTestingString), &in)
	require.NoError(t, err)
	require.Equalf(t, whenTime, in.Time(), "found %v, but expected %v", in.Time(), whenTime)

	var inInteger struct {
		Number types.Time `json:"number"`
	}
	err = json.Unmarshal([]byte(integerJSON), &inInteger)
	require.NoError(t, err)
	require.Equalf(t, whenTime, inInteger.Number.Time(), "found %v, but expected %v", inInteger.Number.Time(), whenTime)

	var inFloat64 struct {
		Number types.Time `json:"number"`
	}
	err = json.Unmarshal([]byte(float64JSON), &inFloat64)
	require.NoError(t, err)

	msTime := time.UnixMilli(1684981731234)
	require.Equalf(t, time.UnixMilli(1684981731234), inFloat64.Number.Time(), "found %v, but expected %v", inFloat64.Number.Time(), msTime)

	var microSeconds types.Time
	err = json.Unmarshal([]byte(timeWhenTestingStringMicroSecond), &microSeconds)
	require.NoError(t, err)
	require.Equalf(t, time.UnixMicro(1691122380942173), microSeconds.Time(), "found %v, but expected %v", microSeconds.Time(), time.UnixMicro(1691122380942173))
}

func TestCurrencyListParam(t *testing.T) {
	t.Parallel()
	_, err := currencyListParam(nil, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodesEmpty, "currencyListParam must reject no currencies")

	_, err = currencyListParam(make([]currency.Code, 11), 10)
	require.ErrorIs(t, err, errTooManyCurrencyCodes, "currencyListParam must reject more currencies than the maximum")

	_, err = currencyListParam([]currency.Code{currency.BTC, currency.EMPTYCODE}, 10)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "currencyListParam must reject an empty currency")

	got, err := currencyListParam([]currency.Code{currency.BTC, currency.USDT, currency.ETH}, 3)
	require.NoError(t, err, "currencyListParam must accept the maximum number of currencies")
	assert.Equal(t, "BTC,USDT,ETH", got, "currencyListParam should join the currencies with commas")
}

type FixtureConnection struct{ websocket.Connection }

func (d *FixtureConnection) SendMessageReturnResponse(context.Context, request.EndpointLimit, any, any) ([]byte, error) {
	return []byte(`{"time":1726121320,"time_ms":1726121320745,"id":1,"conn_id":"f903779a148987ca","trace_id":"d8ee37cd14347e4ed298d44e69aedaa7","channel":"spot.tickers","event":"subscribe","payload":["BRETT_USDT"],"result":{"status":"success"},"requestId":"d8ee37cd14347e4ed298d44e69aedaa7"}`), nil
}

func (d *FixtureConnection) GetURL() string { return "wss://test" }

func getPair(tb testing.TB, a asset.Item) currency.Pair {
	tb.Helper()
	ensureTestPairs(tb)
	return enabledAssetPair[a]
}

func BenchmarkTimeInForceFromString(b *testing.B) {
	for b.Loop() {
		for _, tifString := range []string{gtcTIF, iocTIF, pocTIF, fokTIF} {
			_, err := timeInForceFromString(tifString)
			require.NoError(b, err)
		}
	}
}

func TestTimeInForceFromString(t *testing.T) {
	t.Parallel()
	_, err := timeInForceFromString("abcdef")
	assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce)

	for k, v := range map[string]order.TimeInForce{gtcTIF: order.GoodTillCancel, iocTIF: order.ImmediateOrCancel, pocTIF: order.PostOnly, fokTIF: order.FillOrKill} {
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			tif, err := timeInForceFromString(k)
			require.NoError(t, err)
			assert.Equal(t, v, tif)
		})
	}
}

func TestGetIntervalString(t *testing.T) {
	t.Parallel()
	for key, exp := range map[kline.Interval]string{
		kline.TenMilliseconds:                "10ms",
		kline.TwentyMilliseconds:             "20ms",
		kline.HundredMilliseconds:            "100ms",
		kline.TwoHundredAndFiftyMilliseconds: "250ms",
		kline.ThousandMilliseconds:           "1000ms",
		kline.TenSecond:                      "10s",
		kline.ThirtySecond:                   "30s",
		kline.OneMin:                         "1m",
		kline.FiveMin:                        "5m",
		kline.FifteenMin:                     "15m",
		kline.ThirtyMin:                      "30m",
		kline.OneHour:                        "1h",
		kline.TwoHour:                        "2h",
		kline.FourHour:                       "4h",
		kline.SixHour:                        "6h",
		kline.EightHour:                      "8h",
		kline.TwelveHour:                     "12h",
		kline.OneDay:                         "1d",
		kline.SevenDay:                       "7d",
		kline.OneMonth:                       "30d",
	} {
		t.Run(exp, func(t *testing.T) {
			t.Parallel()
			s, err := getIntervalString(key)
			require.NoError(t, err, "getIntervalString must not error")
			assert.Equal(t, exp, s, "getIntervalString should return Gate's name for the interval")
		})
	}
	_, err := getIntervalString(0)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "0 should be an invalid interval")

	_, err = getIntervalString(kline.FiveDay)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "Any other random interval should also be invalid")
}

func TestGetIntervalFromString(t *testing.T) {
	t.Parallel()
	for str, exp := range map[string]kline.Interval{
		"10ms":  kline.TenMilliseconds,
		"20ms":  kline.TwentyMilliseconds,
		"100ms": kline.HundredMilliseconds,
		"250ms": kline.TwoHundredAndFiftyMilliseconds,
		"10s":   kline.TenSecond,
		"30s":   kline.ThirtySecond,
		"1m":    kline.OneMin,
		"5m":    kline.FiveMin,
		"15m":   kline.FifteenMin,
		"30m":   kline.ThirtyMin,
		"1h":    kline.OneHour,
		"2h":    kline.TwoHour,
		"4h":    kline.FourHour,
		"6h":    kline.SixHour,
		"8h":    kline.EightHour,
		"12h":   kline.TwelveHour,
		"1d":    kline.OneDay,
		"7d":    kline.SevenDay,
		"30d":   kline.OneMonth,
		// Input is lowercased before matching, so mixed case must resolve too
		"1H": kline.OneHour,
	} {
		t.Run(str, func(t *testing.T) {
			t.Parallel()
			interval, err := e.GetIntervalFromString(str)
			require.NoError(t, err, "GetIntervalFromString must not error")
			assert.Equal(t, exp, interval, "GetIntervalFromString should return the interval Gate's name stands for")
		})
	}

	_, err := e.GetIntervalFromString("")
	assert.ErrorIs(t, err, kline.ErrInvalidInterval, "empty interval string should be invalid")

	_, err = e.GetIntervalFromString("3w")
	assert.ErrorIs(t, err, kline.ErrInvalidInterval, "unknown interval string should be invalid")
}

func TestResponseError(t *testing.T) {
	t.Parallel()

	assert.NoError(t, responseError(nil), "nil result should not error")
	assert.NoError(t, responseError(&struct{ Field string }{}), "result without AsError should not error")

	clean := &BotCreateResponse{Code: 200}
	assert.NoError(t, responseError(clean), "successful response should not error")

	failed := &BotCreateResponse{Code: 4, Message: "boom", TraceID: "trace-1"}
	err := responseError(failed)
	require.Error(t, err, "failed response must error")
	assert.Contains(t, err.Error(), "trace-1", "error should carry the trace ID")

	// Results are commonly unmarshalled into a pointer to a pointer, whose method set is empty
	doubled := &AlphaCurrencyQuoteDetailResponse{Label: "BALANCE_NOT_ENOUGH", Message: "insufficient"}
	err = responseError(&doubled)
	require.Error(t, err, "double pointer result must still surface the embedded error")
	assert.Contains(t, err.Error(), "BALANCE_NOT_ENOUGH", "error should carry the alpha label")

	var null *BotCreateResponse
	require.NoError(t, json.Unmarshal([]byte("null"), &null), "Unmarshal must not error")
	assert.NoError(t, responseError(&null), "a null response should not error")
}
