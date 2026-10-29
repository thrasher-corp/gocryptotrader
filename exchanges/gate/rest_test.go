package gate

import (
	"bytes"
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
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
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

	for _, res := range []any{&result, nil} {
		err := ex.sendUnversionedHTTPRequest(t.Context(), exchange.EdgeCase1, publicIsolatedMarginPoolLoansEPL, "/apiw/v2/coded", res)
		require.Error(t, err, "sendUnversionedHTTPRequest must error on a coded error body")
		assert.Contains(t, err.Error(), "label: INVALID_KEY code: 10001", "error should carry the error label and code")
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

	for _, unlabelled := range []string{`{"value":1}`, `{"label":"BAD_REQUEST","message":1}`} {
		assert.ErrorIsf(t, decodeReply(json.RawMessage(unlabelled), new(refusingDecoder)), errRefusedDecode, "decodeReply should keep the decode error of %s", unlabelled)
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
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	ex.SkipAuthCheck = true
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := routes[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			body = `{"label":"SERVER_ERROR","message":"unavailable"}`
		}
		_, err := fmt.Fprint(w, body)
		assert.NoError(t, err, "writing the response should not error")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	for _, u := range []exchange.URL{exchange.RestSpot, exchange.RestFutures} {
		require.NoError(t, ex.API.Endpoints.SetRunningURL(u.String(), server.URL), "SetRunningURL must not error")
	}
	return ex
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
		require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
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

func TestGenerateSignature(t *testing.T) {
	t.Parallel()
	// Expected values follow Gate's documented scheme: HMAC-SHA512 over method, path, query, the SHA512 of the body and
	// the Unix timestamp, each separated by a newline
	ts := time.Unix(1700000000, 0)
	for _, tc := range []struct {
		method, path, query string
		body                any
		exp                 string
	}{
		{
			method: http.MethodGet,
			path:   "/api/v4/spot/orders",
			query:  "currency_pair=BTC_USDT&status=open",
			exp:    "777f9b7385b58e88af6ce1aaae82f217f3e3dd0b9406e98522498001de299a9636bc797e3291aa1c49df79d8decf51eca463bd6082937c4c8742175c938eee56",
		},
		{
			method: http.MethodPost,
			path:   "/api/v4/spot/orders",
			body:   map[string]string{"currency_pair": "BTC_USDT"},
			exp:    "dac4c0d44277469bbb4d2dd7297111b89ce2039ebc9b4cebfaa8cd8a361738ca6904a75ebc28df1a1c041af50c104884dddeeae9bbbe89064513a3837363c886",
		},
	} {
		sig, err := e.GenerateSignature("mock-api-secret", tc.method, tc.path, tc.query, tc.body, ts)
		require.NoErrorf(t, err, "GenerateSignature must not error for %s %s", tc.method, tc.path)
		assert.Equalf(t, tc.exp, sig, "GenerateSignature should sign %s %s as documented", tc.method, tc.path)
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

func TestGetFutureTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesTickers(t.Context(), currency.EMPTYCODE, getPair(t, asset.CoinMarginedFutures))
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetFuturesTickers must reject an empty settlement currency")

	result, err := e.GetFuturesTickers(t.Context(), currency.BTC, getPair(t, asset.CoinMarginedFutures))
	require.NoError(t, err, "GetFuturesTickers must not error for a BTC contract")
	assert.Len(t, result, 1, "GetFuturesTickers should return the BTC contract's ticker")

	result, err = e.GetFuturesTickers(t.Context(), currency.USDT, getPair(t, asset.USDTMarginedFutures))
	require.NoError(t, err, "GetFuturesTickers must not error for a USDT contract")
	require.Len(t, result, 1, "GetFuturesTickers must return the USDT contract's ticker")
	if mockTests {
		exp := &FuturesTickerResponse{
			Contract:              currency.NewPairWithDelimiter("BTC", "USDT", "_"),
			ChangePercentage:      0.37,
			Last:                  67477.9,
			Low24Hour:             66580.1,
			High24Hour:            67815.7,
			TotalSize:             577702480,
			Volume24Hour:          327232988,
			Volume24HourBase:      32723,
			Volume24HourQuote:     2196541426,
			Volume24HourSettle:    2196541426,
			QuantoMultiplier:      0.0001,
			MarkPrice:             67478,
			FundingRate:           0.000083,
			FundingRateIndicative: 0.000083,
			IndexPrice:            67503.92,
			LowestAsk:             67478,
			LowestAskSize:         57437,
			HighestBid:            67477.9,
			HighestBidSize:        95272,
			ChangePrice:           248.1,
		}
		assert.Equal(t, exp, result[0], "GetFuturesTickers should decode every field")
	}
}

func TestDeliveryOrderCreateParamsMarshalJSON(t *testing.T) {
	t.Parallel()
	pair := currency.NewPairWithDelimiter("BTC", "USDT_20260626", currency.UnderscoreDelimiter)
	for _, tc := range []struct {
		name   string
		params *DeliveryOrderCreateRequest
		exp    string
	}{
		{
			name:   "market order",
			params: &DeliveryOrderCreateRequest{Contract: pair, Size: -1, TimeInForce: iocTIF, Settle: currency.USDT},
			exp:    `{"contract":"BTC_USDT_20260626","size":-1,"iceberg":0,"price":"0","tif":"ioc"}`,
		},
		{
			name:   "close position",
			params: &DeliveryOrderCreateRequest{Contract: pair, ClosePosition: true, TimeInForce: iocTIF, Settle: currency.USDT},
			exp:    `{"contract":"BTC_USDT_20260626","size":0,"iceberg":0,"price":"0","tif":"ioc","close":true}`,
		},
		{
			name:   "close hedge mode side",
			params: &DeliveryOrderCreateRequest{Contract: pair, AutoSize: "close_long", TimeInForce: iocTIF, Settle: currency.USDT},
			exp:    `{"contract":"BTC_USDT_20260626","size":0,"iceberg":0,"price":"0","tif":"ioc","auto_size":"close_long"}`,
		},
		{
			name: "limit order",
			params: &DeliveryOrderCreateRequest{
				Contract:                  pair,
				Size:                      6024,
				Iceberg:                   10,
				Price:                     3765.5,
				TimeInForce:               gtcTIF,
				Text:                      "t-my-custom-id",
				ReduceOnly:                true,
				Settle:                    currency.USDT,
				SelfTradePreventionAction: "cn",
			},
			exp: `{"contract":"BTC_USDT_20260626","size":6024,"iceberg":10,"price":"3765.5","tif":"gtc","text":"t-my-custom-id","reduce_only":true,"stp_act":"cn"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tc.params)
			require.NoError(t, err, "Marshal must not error")
			assert.JSONEq(t, tc.exp, string(got), "Marshal should send the documented encoding")
		})
	}
}

func TestFuturesOrderCreateParamsMarshalJSON(t *testing.T) {
	t.Parallel()
	contract := currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
	for _, tc := range []struct {
		name string
		arg  FuturesOrderCreateRequest
		exp  string
	}{
		{
			name: "market order sends a zero price and iceberg",
			arg:  FuturesOrderCreateRequest{Contract: contract, Size: 10, TimeInForce: iocTIF, Text: "t-x"},
			exp:  `{"contract":"BTC_USDT","size":"10","iceberg":"0","price":"0","tif":"ioc","text":"t-x"}`,
		},
		{
			name: "close order sends a zero size",
			arg:  FuturesOrderCreateRequest{Contract: contract, TimeInForce: iocTIF, ClosePosition: true, ReduceOnly: true},
			exp:  `{"contract":"BTC_USDT","size":"0","iceberg":"0","price":"0","tif":"ioc","close":true,"reduce_only":true}`,
		},
		{
			name: "limit order sends every optional field",
			arg: FuturesOrderCreateRequest{
				Contract: contract, Size: -1.5, Iceberg: 0.5, Price: 31503.3, TimeInForce: gtcTIF, Text: "t-y", AutoSize: "close_long",
				Settle: currency.USDT, SelfTradePreventionAction: "cn", PositionID: 7, MarketOrderSlipRatio: 0.03, PositionMarginMode: "isolated",
				ActionMode: "ACK", TakeProfitTriggerPrice: 32000, StopLossTriggerPrice: 31000,
			},
			exp: `{"contract":"BTC_USDT","size":"-1.5","iceberg":"0.5","price":"31503.3","tif":"gtc","text":"t-y","auto_size":"close_long","stp_act":"cn","pid":7,"market_order_slip_ratio":"0.03","pos_margin_mode":"isolated","action_mode":"ACK","tpsl_tp_trigger_price":"32000","tpsl_sl_trigger_price":"31000"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			payload, err := json.Marshal(tc.arg)
			require.NoError(t, err, "Marshal must not error")
			assert.Equal(t, tc.exp, string(payload), "FuturesOrderCreateRequest should marshal to the documented body")
			pointerPayload, err := json.Marshal(&tc.arg)
			require.NoError(t, err, "Marshal must not error for a pointer")
			assert.Equal(t, tc.exp, string(pointerPayload), "a FuturesOrderCreateRequest pointer should marshal like a value")
		})
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
	for _, timeout := range []time.Duration{0, 5 * time.Second, time.Hour} {
		assert.NoErrorf(t, validateCountdownTimeout(timeout), "validateCountdownTimeout should accept %s", timeout)
	}
	for _, timeout := range []time.Duration{-time.Second, time.Nanosecond, 4999 * time.Millisecond, 5500 * time.Millisecond} {
		assert.ErrorIsf(t, validateCountdownTimeout(timeout), errInvalidTimeout, "validateCountdownTimeout should reject %s", timeout)
	}
}

func TestCheckWholeSeconds(t *testing.T) {
	t.Parallel()
	for _, d := range []time.Duration{0, time.Second, 24 * time.Hour} {
		assert.NoErrorf(t, checkWholeSeconds(types.DurationSeconds(d), "expiry"), "checkWholeSeconds should accept %s", d)
	}
	for _, d := range []time.Duration{time.Nanosecond, 1500 * time.Millisecond, -time.Nanosecond, -time.Second, -1500 * time.Millisecond} {
		assert.ErrorIsf(t, checkWholeSeconds(types.DurationSeconds(d), "expiry"), errNotWholeSeconds, "checkWholeSeconds should reject %s", d)
	}
}

func TestGetUnderlyingFromCurrencyPair(t *testing.T) {
	t.Parallel()
	uly, err := e.GetUnderlyingFromCurrencyPair(currency.Pair{Delimiter: currency.UnderscoreDelimiter, Base: currency.BTC, Quote: currency.NewCode("USDT_LLK")})
	require.NoError(t, err)
	assert.True(t, uly.Equal(currency.NewBTCUSDT()))
}

func TestTickerHandlersSkipEmptyBatches(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	require.NoError(t, ex.processTicker(t.Context(), []byte(`{"currency_pair":"UNTRACKED_USDT","last":"1"}`), time.Now()), "processTicker must not error for an untracked pair")
	require.NoError(t, ex.processFuturesTickers(t.Context(), []byte(`{"time":1541659086,"channel":"futures.tickers","event":"update","result":[]}`), asset.USDTMarginedFutures), "processFuturesTickers must not error for an empty result")
	assert.Empty(t, ex.Websocket.DataHandler.C, "ticker handlers should not relay an empty batch")
}

func TestFuturesPositionBOBCapturedPayloads(t *testing.T) {
	t.Parallel()
	payloads := [][]byte{
		[]byte(`{"time":1788148634,"time_ms":1788148634648,"channel":"futures.positions","event":"update","result":[{"time":1788148634,"time_ms":1788148634648,"size":"112","user":"12870774","leverage_max":10,"mode":"single","update_id":2498,"cross_leverage_limit":20,"liq_price":0,"maintenance_rate":0.9,"risk_limit":2500000,"last_close_pnl":-0.2543647646,"entry_price":0.005032,"leverage":1,"realised_pnl":-0.027052032,"contract":"BOB_USDT","history_point":0,"margin":56.4006688,"realised_point":0,"history_pnl":-274.6956184238,"pos_margin_mode":"isolated","lever":"1"}]}`),
		[]byte(`{"time":1788148634,"time_ms":1788148634648,"channel":"futures.positions","event":"update","result":[{"time":1788148634,"time_ms":1788148634648,"size":"126","user":"12870774","leverage_max":10,"mode":"single","update_id":2499,"cross_leverage_limit":20,"liq_price":0,"maintenance_rate":0.9,"risk_limit":2500000,"last_close_pnl":-0.2543647646,"entry_price":0.005032444444,"leverage":1,"realised_pnl":-0.030436224,"contract":"BOB_USDT","history_point":0,"margin":63.4563566,"realised_point":0,"history_pnl":-274.6956184238,"pos_margin_mode":"isolated","lever":"1"}]}`),
	}
	wantSizes := []string{"112", "126"}
	wantUpdateIDs := []int64{2498, 2499}
	wantMargins := []float64{56.4006688, 63.4563566}
	wantOpeningPrices := []float64{0.005032, 0.005032444444}
	wantRealisedPNLs := []float64{-0.027052032, -0.030436224}

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.USDTMarginedFutures,
		currency.Pairs{currency.NewPair(currency.NewCode("BOB"), currency.USDT)}, false),
		"StorePairs must register the captured contract")
	for i := range payloads {
		response := struct {
			Time      types.Time                 `json:"time"`
			TimeMilli types.Time                 `json:"time_ms"`
			Channel   string                     `json:"channel"`
			Event     string                     `json:"event"`
			Result    []WebsocketFuturesPosition `json:"result"`
		}{}
		decoder := json.NewDecoder(bytes.NewReader(payloads[i]))
		decoder.DisallowUnknownFields()
		require.NoError(t, decoder.Decode(&response), "captured BOB position payload must unmarshal without unknown fields")
		require.Len(t, response.Result, 1, "captured BOB payload must contain one position")
		assert.Equal(t, "BOB_USDT", response.Result[0].Contract, "captured contract should be preserved")

		require.NoError(t, ex.WebsocketHandleFuturesData(t.Context(), nil, payloads[i], asset.USDTMarginedFutures),
			"WebsocketHandleFuturesData must process the captured position update")
		message := <-ex.Websocket.DataHandler.C
		positions, ok := message.Data.([]futures.Position)
		require.True(t, ok, "captured payload must emit canonical futures positions")
		require.Len(t, positions, 1, "captured payload must emit one position")
		position := positions[0]
		assert.Equal(t, order.Open, position.Status, "position status should be open")
		assert.Equal(t, order.Long, position.LatestDirection, "positive position size should be long")
		assert.Equal(t, wantSizes[i], position.LatestSize.String(), "position size should be preserved")
		assert.Equal(t, "1", position.Leverage.String(), "isolated position should use the quoted lever field")
		assert.Equal(t, wantUpdateIDs[i], position.UpdateID, "position update ID should be preserved")
		assert.Equal(t, time.UnixMilli(1788148634648), position.LastUpdated, "position update time should preserve milliseconds")
		assert.True(t, position.OpeningPrice.Equal(decimal.MustFromFloat(wantOpeningPrices[i])), "position opening price should be correct")
		assert.True(t, position.PositionMargin.Equal(decimal.MustFromFloat(wantMargins[i])), "position margin should be correct")
		assert.True(t, position.RealisedPNL.Equal(decimal.MustFromFloat(wantRealisedPNLs[i])), "position realised PNL should be correct")
	}
}

func TestFuturesPositionCapturedPayload(t *testing.T) {
	t.Parallel()
	payload := []byte(`{"time":1787206135,"time_ms":1787206135040,"channel":"futures.positions","event":"update","result":[{"time":1787206135,"time_ms":1787206135038,"size":"-55","user":"12870774","leverage_max":25,"mode":"single","update_id":267,"cross_leverage_limit":40,"liq_price":0.023554,"maintenance_rate":0.9,"risk_limit":2500000,"last_close_pnl":-0.802254544,"entry_price":0.012071295652,"leverage":1,"realised_pnl":0.079661896,"contract":"GPS_USDT","history_point":0,"margin":66.491714276087,"realised_point":0,"history_pnl":17.1379873471,"pos_margin_mode":"isolated","lever":"1"}]}`)
	response := struct {
		Time      types.Time                 `json:"time"`
		TimeMilli types.Time                 `json:"time_ms"`
		Channel   string                     `json:"channel"`
		Event     string                     `json:"event"`
		Result    []WebsocketFuturesPosition `json:"result"`
	}{}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	require.NoError(t, decoder.Decode(&response), "captured position payload must unmarshal without unknown fields")
	require.Len(t, response.Result, 1, "captured payload must contain one position")
	assert.Equal(t, time.UnixMilli(1787206135038), response.Result[0].Time.Time(), "position timestamp should preserve millisecond precision")

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	require.NoError(t, e.processFuturesPositionsNotification(t.Context(), payload, asset.USDTMarginedFutures))
	message := <-e.Websocket.DataHandler.C
	positions, ok := message.Data.([]futures.Position)
	require.True(t, ok, "captured payload must emit canonical futures positions")
	require.Len(t, positions, 1, "captured payload must emit one position")
	position := positions[0]
	assert.Equal(t, "GPS_USDT", position.Pair.String(), "position pair should be normalised")
	assert.Equal(t, order.Open, position.Status, "position status should be open")
	assert.Equal(t, order.Short, position.LatestDirection, "position direction should be short")
	assert.Equal(t, "55", position.LatestSize.String(), "position size should be absolute")
	assert.Equal(t, "1", position.Leverage.String(), "position leverage should use the isolated leverage field")
	assert.Equal(t, int64(267), position.UpdateID, "position update ID should be preserved")
	assert.Equal(t, time.UnixMilli(1787206135038), position.LastUpdated, "position update time should be preserved")
	assert.True(t, position.OpeningPrice.Equal(decimal.MustFromFloat(0.012071295652)), "position opening price should be correct")
	assert.True(t, position.PositionMargin.Equal(decimal.MustFromFloat(66.491714276087)), "position margin should be correct")
	assert.True(t, position.MaintenanceMarginFraction.Equal(decimal.MustFromFloat(0.9)), "position maintenance rate should be correct")
	assert.True(t, position.EstimatedLiquidationPrice.Equal(decimal.MustFromFloat(0.023554)), "position liquidation price should be correct")
	assert.True(t, position.RealisedPNL.Equal(decimal.MustFromFloat(0.079661896)), "position realised PNL should be correct")

	classicCrossPayload := []byte(strings.NewReplacer(
		`"leverage":1`, `"leverage":0`,
		`"pos_margin_mode":"isolated"`, `"pos_margin_mode":""`,
		`"lever":"1"`, `"lever":"0"`,
	).Replace(string(payload)))
	require.NoError(t, e.processFuturesPositionsNotification(t.Context(), classicCrossPayload, asset.USDTMarginedFutures),
		"processFuturesPositionsNotification must process classic cross margin")
	message = <-e.Websocket.DataHandler.C
	positions, ok = message.Data.([]futures.Position)
	require.True(t, ok, "classic cross-margin payload must emit canonical futures positions")
	require.Len(t, positions, 1, "classic cross-margin payload must emit one position")
	assert.Equal(t, "40", positions[0].Leverage.String(),
		"classic cross-margin leverage should use the cross leverage limit")

	closedPayload := []byte(strings.Replace(string(payload), `"size":"-55"`, `"size":"0"`, 1))
	require.NoError(t, e.processFuturesPositionsNotification(t.Context(), closedPayload, asset.USDTMarginedFutures),
		"processFuturesPositionsNotification must process a zero-size update")
	message = <-e.Websocket.DataHandler.C
	positions, ok = message.Data.([]futures.Position)
	require.True(t, ok, "zero-size update must emit canonical futures positions")
	require.Len(t, positions, 1, "zero-size update must emit one position")
	assert.Equal(t, order.Closed, positions[0].Status, "zero-size position status should be closed")
	assert.Equal(t, order.UnknownSide, positions[0].LatestDirection, "zero-size single-mode direction should be unknown")
	assert.Equal(t, time.UnixMilli(1787206135038), positions[0].CloseDate, "zero-size position close time should be preserved")

	for _, tt := range []struct {
		mode      string
		direction order.Side
	}{
		{mode: "dual_long", direction: order.Long},
		{mode: "dual_short", direction: order.Short},
	} {
		t.Run(tt.mode, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
			dualPayload := []byte(strings.Replace(string(closedPayload), `"mode":"single"`, `"mode":"`+tt.mode+`"`, 1))
			require.NoError(t, ex.processFuturesPositionsNotification(t.Context(), dualPayload, asset.USDTMarginedFutures),
				"processFuturesPositionsNotification must process a zero-size dual-mode update")
			message := <-ex.Websocket.DataHandler.C
			positions, ok := message.Data.([]futures.Position)
			require.True(t, ok, "zero-size dual-mode update must emit canonical futures positions")
			require.Len(t, positions, 1, "zero-size dual-mode update must emit one position")
			assert.Equal(t, tt.direction, positions[0].LatestDirection, "zero-size dual-mode direction should be preserved")
		})
	}

	t.Run("delivery collateral", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
		deliveryPairs, err := ex.GetEnabledPairs(asset.DeliveryFutures)
		require.NoError(t, err, "GetEnabledPairs must return delivery pairs")
		require.NotEmpty(t, deliveryPairs, "delivery pairs must not be empty")
		deliveryPair := deliveryPairs[0]
		formattedPair, err := ex.FormatExchangeCurrency(deliveryPair, asset.DeliveryFutures)
		require.NoError(t, err, "FormatExchangeCurrency must format the delivery pair")
		deliveryPayload := []byte(strings.Replace(string(payload), "GPS_USDT", formattedPair.String(), 1))
		require.NoError(t, ex.processFuturesPositionsNotification(t.Context(), deliveryPayload, asset.DeliveryFutures),
			"processFuturesPositionsNotification must process a delivery position")
		message := <-ex.Websocket.DataHandler.C
		positions, ok := message.Data.([]futures.Position)
		require.True(t, ok, "delivery update must emit canonical futures positions")
		require.Len(t, positions, 1, "delivery update must emit one position")
		assert.Equal(t, currency.USDT, positions[0].CollateralCurrency,
			"delivery collateral currency should exclude the dated contract suffix")
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

// BenchmarkTimeInForceFromString-8         1528101               767.3 ns/op             0 B/op          0 allocs/op
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

func TestOrderCreateParamsValidateNilGuard(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, (*FuturesOrderCreateRequest)(nil).validate(true), common.ErrNilPointer)
	require.ErrorIs(t, (*DeliveryOrderCreateRequest)(nil).validate(true), common.ErrNilPointer)
}

// TestOrderCreateParamsValidateClosePosition checks that a futures order may close a position with a zero size,
// which Gate documents for single position mode as close with a size of 0
func TestOrderCreateParamsValidateClosePosition(t *testing.T) {
	t.Parallel()
	futuresContract := currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
	deliveryContract := currency.NewPairWithDelimiter("BTC", "USDT_20260626", currency.UnderscoreDelimiter)
	for _, tc := range []struct {
		name      string
		close     bool
		autoSize  string
		size      types.Number
		expectErr error
	}{
		{name: "close with a zero size", close: true},
		{name: "close one side of a dual mode position", autoSize: "close_long"},
		{name: "close with a size", close: true, size: 1, expectErr: errInvalidOrderSize},
		{name: "zero size without closing", expectErr: errInvalidOrderSize},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, isRest := range []bool{true, false} {
				perpetual := &FuturesOrderCreateRequest{Contract: futuresContract, Size: tc.size, TimeInForce: iocTIF, ClosePosition: tc.close, AutoSize: tc.autoSize, ReduceOnly: true, Settle: currency.USDT}
				assert.ErrorIsf(t, perpetual.validate(isRest), tc.expectErr, "futures validate should return the expected error when isRest is %t", isRest)
				delivery := &DeliveryOrderCreateRequest{Contract: deliveryContract, Size: tc.size, TimeInForce: iocTIF, ClosePosition: tc.close, AutoSize: tc.autoSize, ReduceOnly: true, Settle: currency.USDT}
				assert.ErrorIsf(t, delivery.validate(isRest), tc.expectErr, "delivery validate should return the expected error when isRest is %t", isRest)
			}
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
