package kraken

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
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

	spotTestPair    = currency.NewPair(currency.XBT, currency.USD)
	futuresTestPair = currency.NewPairWithDelimiter("PF", "XBTUSD", currency.UnderscoreDelimiter)
)

// configureTestExchange serves an exchange from the recorded responses in testdata/http.json in mock tests, signing
// with placeholder credentials so authenticated endpoints run against their recorded responses too. Live tests use
// authenticated endpoints when credentials are supplied
func configureTestExchange(ex *Exchange) error {
	if !mockTests {
		if apiCredentials.Key != "" && apiCredentials.Secret != "" {
			ex.API.AuthenticatedSupport = true
			ex.API.AuthenticatedWebsocketSupport = true
			ex.SetCredentials(apiCredentials)
		}
		return nil
	}
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	// Kraken secrets are base64 encoded
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "bW9jay1hcGktc2VjcmV0"})
	if err := testexch.MockHTTPInstance(ex); err != nil {
		return err
	}
	// Mock responses are served locally, so throttling protects nothing
	return ex.DisableRateLimiter()
}

// newTestExchange returns an exchange set up from the test configuration under the test's name, so the stores it
// writes do not collide with other tests'. In mock tests its REST requests are served from the recorded responses
func newTestExchange(tb testing.TB) *Exchange {
	tb.Helper()
	ex := new(Exchange)
	require.NoError(tb, testexch.Setup(ex), "Setup must not error")
	ex.Name = tb.Name()
	require.NoError(tb, configureTestExchange(ex), "configureTestExchange must not error")
	return ex
}

// testSecret is the private key of Kraken's documented signature example
const testSecret = "kQH5HW/8p1uGOVjbgWA7FunAmGO8lsSUXNsu3eow76sz84Q18fWxnyRzBHCd3pd5nE9qa99HAZtuZuj6F1huXg=="

// newHTTPTestExchange returns an exchange named after the test whose every REST endpoint handler serves, signing with
// testSecret. The futures URL keeps Kraken's /derivatives prefix and the supplementary one its trailing slash, as the
// defaults do
func newHTTPTestExchange(t *testing.T, handler http.HandlerFunc) *Exchange {
	t.Helper()
	server := httptest.NewTestServer(t, handler)
	client := server.Client()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "test-key", Secret: testSecret})
	require.NoError(t, ex.SetHTTPClient(client), "SetHTTPClient must not error")
	for u, suffix := range map[exchange.URL]string{
		exchange.RestSpot:                 "",
		exchange.RestFutures:              "/derivatives",
		exchange.RestFuturesSupplementary: "/api/",
	} {
		require.NoErrorf(t, ex.API.Endpoints.SetRunningURL(u.String(), server.URL+suffix), "SetRunningURL must not error for %s", u)
	}
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	return ex
}

func decodedTestSecret(t *testing.T) string {
	t.Helper()
	secret, err := base64.StdEncoding.DecodeString(testSecret)
	assert.NoError(t, err, "DecodeString should not error")
	return string(secret)
}

func TestSpotSignature(t *testing.T) {
	t.Parallel()
	secret := decodedTestSecret(t)
	for _, tc := range []struct {
		name, requestURI, nonce, body, exp string
	}{
		{
			name:       "documented example",
			requestURI: "/0/private/AddOrder",
			nonce:      "1616492376594",
			body:       "nonce=1616492376594&ordertype=limit&pair=XBTUSD&price=37500&type=buy&volume=1.25",
			exp:        "4/dpxb3iT4tp/ZCVEwSnEsLxx0bqyhLpdfOpc6fn7OR8+UClSV5n9E6aSS8MPtnRfp32bAb0nmbRn6H8ndwLUQ==",
		},
		{
			name:       "JSON body and query string",
			requestURI: "/0/private/BalanceEx?account_id=WX6V-JUKW-KKPB-QE36",
			nonce:      "1616492376594",
			body:       `{"nonce":1616492376594}`,
			exp:        "mDKg80CLShRldrIGu7taGp/GfMXUwqa53OEktcFfiiAJpU89rGT78n4oS/sEceeBmue9I87upd8KZ0OhfvlPxg==",
		},
		{
			name:       "nested query string without a body",
			requestURI: "/funding/v1/methods/withdraw?asset%5Bclass%5D=currency&asset%5Bname%5D=USDC&limit=50",
			nonce:      "1791504602000000000",
			exp:        "ZDMDcR+vhL/pc5gQTi+RiuIVfTVW4HY/I8OHyuNzm+gVzPDwsnUDc6zH/BiYQaXblbP8LTrLbkGYA2oOzwaFvQ==",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := spotSignature(secret, tc.requestURI, tc.nonce, []byte(tc.body))
			require.NoError(t, err, "spotSignature must not error")
			assert.Equal(t, tc.exp, got, "spotSignature should match the independently computed signature")
		})
	}
}

func TestFuturesSignature(t *testing.T) {
	t.Parallel()
	got, err := futuresSignature(decodedTestSecret(t), "cliOrdId=my%20order&orderType=lmt&side=buy&size=1&symbol=PF_XBTUSD", "1791504602000000000", "/api/v3/sendorder")
	require.NoError(t, err, "futuresSignature must not error")
	assert.Equal(t, "JaLvzoQkt+4mGbjR2lbd0+v+qrg/EI45lbWYKYx0d4YF3mUu1aOyyN3tPuQCQzrrZf/zvGBld0CFsf6Xum++jQ==", got, "futuresSignature should match the independently computed signature")
}

// checkSpotSignedRequest asserts a Spot REST private, Funding (Beta) or Affiliate REST request carries the key and a
// signature over its URI, nonce and body, returning the body. Handlers run on the server's goroutine, so it only asserts
func checkSpotSignedRequest(t *testing.T, r *http.Request, nonce func(body []byte) string) []byte {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	assert.NoError(t, err, "ReadAll should not error")
	assert.Equal(t, "test-key", r.Header.Get("API-Key"), "API-Key should be the key")
	expected, err := spotSignature(decodedTestSecret(t), r.URL.RequestURI(), nonce(body), body)
	assert.NoError(t, err, "spotSignature should not error")
	assert.Equal(t, expected, r.Header.Get("API-Sign"), "API-Sign should sign the request URI, nonce and body")
	return body
}

func TestSendAuthenticatedHTTPRequest(t *testing.T) {
	t.Parallel()
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "Method should be POST")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "Content-Type should be JSON")
		body := checkSpotSignedRequest(t, r, func(body []byte) string {
			var payload struct {
				Nonce json.RawMessage `json:"nonce"`
			}
			assert.NoError(t, json.Unmarshal(body, &payload), "Unmarshal should not error")
			return string(payload.Nonce)
		})
		var payload map[string]any
		assert.NoError(t, json.Unmarshal(body, &payload), "Unmarshal should not error")
		assert.Contains(t, payload, "nonce", "body should carry the nonce")
		switch r.URL.Path {
		case "/0/private/Success":
			assert.Equal(t, "WX6V-JUKW-KKPB-QE36", r.URL.Query().Get("account_id"), "account_id should be sent in the query")
			assert.Equal(t, "XBTUSD", payload["pair"], "body should carry the parameters")
			_, _ = w.Write([]byte(`{"error":["WGeneral:Deprecated field"],"result":{"count":2}}`))
		case "/0/private/Errors":
			_, _ = w.Write([]byte(`{"error":["EOrder:Unknown order","EGeneral:Invalid arguments:volume"]}`))
		case "/0/private/ErrorString":
			_, _ = w.Write([]byte(`{"error":"EAPI:Invalid key"}`))
		case "/0/private/Null":
			_, _ = w.Write([]byte(`{"error":[],"result":null}`))
		case "/0/private/Status":
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":["EService:Unavailable"]}`))
		case "/0/private/Raw":
			_, _ = w.Write([]byte("PK\x03\x04zip"))
		}
	})

	var result struct {
		Count uint64 `json:"count"`
	}
	err := ex.SendAuthenticatedHTTPRequest(t.Context(), "/0/private/Success", url.Values{"account_id": {"WX6V-JUKW-KKPB-QE36"}}, map[string]any{"pair": "XBTUSD"}, &result)
	require.NoError(t, err, "SendAuthenticatedHTTPRequest must not error when only warned")
	assert.Equal(t, uint64(2), result.Count, "SendAuthenticatedHTTPRequest should decode the result")

	err = ex.SendAuthenticatedHTTPRequest(t.Context(), "/0/private/Errors", nil, nil, &result)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "SendAuthenticatedHTTPRequest must return an APIError")
	assert.Equal(t, []string{"EOrder:Unknown order", "EGeneral:Invalid arguments:volume"}, apiErr.Errors, "APIError should hold every error")
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "SendAuthenticatedHTTPRequest should match a mapped error")
	assert.ErrorIs(t, err, errAPIResponse, "SendAuthenticatedHTTPRequest should match errAPIResponse")

	err = ex.SendAuthenticatedHTTPRequest(t.Context(), "/0/private/ErrorString", nil, nil, &result)
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "SendAuthenticatedHTTPRequest should decode an error sent as a string")

	err = ex.SendAuthenticatedHTTPRequest(t.Context(), "/0/private/Null", nil, nil, &result)
	assert.ErrorIs(t, err, common.ErrNoResponse, "SendAuthenticatedHTTPRequest should reject a null result")

	err = ex.SendAuthenticatedHTTPRequest(t.Context(), "/0/private/Status", nil, nil, &result)
	assert.ErrorIs(t, err, errAPIResponse, "SendAuthenticatedHTTPRequest should return the API error of an error status")
	assert.ErrorIs(t, err, request.ErrBadStatus, "SendAuthenticatedHTTPRequest should keep the status error")

	var raw []byte
	require.NoError(t, ex.sendSpotPrivateRequest(t.Context(), "/0/private/Raw", nil, nil, &raw), "sendSpotPrivateRequest must not error for a file")
	assert.Equal(t, "PK\x03\x04zip", string(raw), "sendSpotPrivateRequest should return the file as sent")
}

func TestSendHTTPRequest(t *testing.T) {
	t.Parallel()
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "Method should be GET")
		assert.Empty(t, r.Header.Get("API-Key"), "a public request should not carry the key")
		reply := `{"error":["EQuery:Unknown asset pair"]}`
		if r.URL.Path == "/0/public/Time" {
			reply = `{"error":[],"result":{"unixtime":1791504602,"rfc1123":"Fri, 09 Oct 26 00:10:02 +0000"}}`
		}
		_, _ = w.Write([]byte(reply))
	})
	var result ServerTimeResponse
	require.NoError(t, ex.SendHTTPRequest(t.Context(), "/0/public/Time", &result), "SendHTTPRequest must not error")
	assert.Equal(t, ServerTimeResponse{UnixTime: types.Time(time.Unix(1791504602, 0)), RFC1123: "Fri, 09 Oct 26 00:10:02 +0000"}, result, "SendHTTPRequest should decode the result")
	err := ex.SendHTTPRequest(t.Context(), "/0/public/Ticker?pair=MOONUSD", &result)
	assert.ErrorIs(t, err, errAPIResponse, "SendHTTPRequest should return the error Kraken replies with")
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "SendHTTPRequest should map the error Kraken replies with")
}

func TestSendHeaderAuthenticatedHTTPRequest(t *testing.T) {
	t.Parallel()
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		nonce := r.Header.Get("API-Nonce")
		assert.NotEmpty(t, nonce, "API-Nonce should be sent")
		body := checkSpotSignedRequest(t, r, func([]byte) string { return nonce })
		switch r.URL.Path {
		case "/funding/v1/addresses":
			assert.Equal(t, http.MethodPost, r.Method, "Method should be POST")
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "Content-Type should be JSON")
			assert.JSONEq(t, `{"name":"wallet"}`, string(body), "body should be the JSON request without a nonce")
			assert.Equal(t, "WX6V-JUKW-KKPB-QE36", r.URL.Query().Get("account_id"), "account_id should be sent in the query")
			_, _ = w.Write([]byte(`{"address_id":"AB7J4FF-BGM7G-V2JMIH","verified":true}`))
		case "/funding/v1/networks":
			assert.Equal(t, http.MethodGet, r.Method, "Method should be GET")
			assert.Empty(t, body, "a GET should not carry a body")
			assert.Empty(t, r.Header.Get("Content-Type"), "a GET should not carry a content type")
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"type":"tag:kraken.com,2025:InvalidKey","status":401,"title":"API: Invalid key","data":null,"source":"Gateway"}`))
		}
	})

	var result struct {
		AddressID string `json:"address_id"`
		Verified  bool   `json:"verified"`
	}
	err := ex.SendHeaderAuthenticatedHTTPRequest(t.Context(), http.MethodPost, "/funding/v1/addresses", url.Values{"account_id": {"WX6V-JUKW-KKPB-QE36"}}, map[string]string{"name": "wallet"}, &result)
	require.NoError(t, err, "SendHeaderAuthenticatedHTTPRequest must not error")
	assert.Equal(t, "AB7J4FF-BGM7G-V2JMIH", result.AddressID, "SendHeaderAuthenticatedHTTPRequest should decode the bare response")
	assert.True(t, result.Verified, "SendHeaderAuthenticatedHTTPRequest should decode the bare response")

	err = ex.SendHeaderAuthenticatedHTTPRequest(t.Context(), http.MethodGet, "/funding/v1/networks", nil, nil, &result)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "SendHeaderAuthenticatedHTTPRequest must return an APIError for problem details")
	assert.Equal(t, &APIError{Errors: []string{"API: Invalid key"}, Type: "tag:kraken.com,2025:InvalidKey"}, apiErr, "APIError should hold the problem's title and type")
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "SendHeaderAuthenticatedHTTPRequest should match the problem type's mapped error")
	assert.ErrorIs(t, err, request.ErrBadStatus, "SendHeaderAuthenticatedHTTPRequest should keep the status error")
}

func TestSendFuturesAuthenticatedHTTPRequest(t *testing.T) {
	t.Parallel()
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-key", r.Header.Get("APIKey"), "APIKey should be the key")
		nonce := r.Header.Get("Nonce")
		assert.NotEmpty(t, nonce, "Nonce should be sent")
		expected, err := futuresSignature(decodedTestSecret(t), r.URL.RawQuery, nonce, strings.TrimPrefix(r.URL.Path, "/derivatives"))
		assert.NoError(t, err, "futuresSignature should not error")
		assert.Equal(t, expected, r.Header.Get("Authent"), "Authent should sign the query string, nonce and endpoint path")
		switch r.URL.Path {
		case "/derivatives/api/v3/sendorder":
			assert.Equal(t, http.MethodPost, r.Method, "Method should be POST")
			const expQuery = "cliOrdId=my%20order&symbol=PF_XBTUSD"
			assert.Equal(t, expQuery, r.URL.RawQuery, "a space should be percent-encoded rather than sent as a plus sign")
			assert.Equal(t, "algo-1", r.Header.Get("algoId"), "the optional headers should be sent")
			_, _ = w.Write([]byte(`{"result":"success","serverTime":"2026-10-09T00:07:39.15Z","sendStatus":{"status":"placed"}}`))
		case "/derivatives/api/v3/accounts":
			_, _ = w.Write([]byte(`{"result":"error","error":"authenticationError","serverTime":"2026-10-09T00:07:39.15Z"}`))
		case "/derivatives/api/v3/codes":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"result":"error","serverTime":"2026-10-09T00:07:39.15Z","errors":[{"code":95,"message":"Account is not permitted"},"invalidArgument"]}`))
		case "/api/history/v3/executions":
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"status":"unauthorized","reason":"You are not authorized to access this endpoint."}`))
		case "/api/history/v3/accountlogcsv":
			_, _ = w.Write([]byte("date,uid\n"))
		}
	})

	var result struct {
		SendStatus struct {
			Status string `json:"status"`
		} `json:"sendStatus"`
	}
	err := ex.SendFuturesAuthenticatedHTTPRequest(t.Context(), exchange.RestFutures, http.MethodPost, "/api/v3/sendorder", url.Values{"symbol": {"PF_XBTUSD"}, "cliOrdId": {"my order"}}, map[string]string{"algoId": "algo-1"}, &result)
	require.NoError(t, err, "SendFuturesAuthenticatedHTTPRequest must not error")
	assert.Equal(t, "placed", result.SendStatus.Status, "SendFuturesAuthenticatedHTTPRequest should decode the response")

	err = ex.SendFuturesAuthenticatedHTTPRequest(t.Context(), exchange.RestFutures, http.MethodGet, "/api/v3/accounts", nil, nil, &result)
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "SendFuturesAuthenticatedHTTPRequest should report an error result sent with a success status")

	err = ex.SendFuturesAuthenticatedHTTPRequest(t.Context(), exchange.RestFutures, http.MethodPost, "/api/v3/codes", nil, nil, &result)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "SendFuturesAuthenticatedHTTPRequest must return an APIError")
	assert.Equal(t, []string{"code 95: Account is not permitted", "invalidArgument"}, apiErr.Errors, "APIError should hold coded and plain errors")

	err = ex.SendFuturesAuthenticatedHTTPRequest(t.Context(), exchange.RestFuturesSupplementary, http.MethodGet, "/history/v3/executions", nil, nil, &result)
	require.ErrorAs(t, err, &apiErr, "SendFuturesAuthenticatedHTTPRequest must return an APIError for a history error")
	assert.Equal(t, []string{"You are not authorized to access this endpoint."}, apiErr.Errors, "APIError should hold the history API's reason")
	assert.Equal(t, "unauthorized", apiErr.Type, "APIError should hold the history API's status as its type")
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "an unauthorized history request should be an authentication failure")

	body, err := ex.sendFuturesAuthenticatedRequest(t.Context(), exchange.RestFuturesSupplementary, http.MethodGet, "/history/v3/accountlogcsv", nil, nil, nil, true)
	require.NoError(t, err, "sendFuturesAuthenticatedRequest must not error for a file")
	assert.Equal(t, "date,uid\n", string(body), "sendFuturesAuthenticatedRequest should return the file as sent")
}

func TestSendFuturesHTTPRequest(t *testing.T) {
	t.Parallel()
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("APIKey"), "a public request should not carry the key")
		switch r.URL.Path {
		case "/api/charts/v1/trade":
			_, _ = w.Write([]byte(`["PF_XBTUSD"]`))
		case "/derivatives/api/v3/tickers/PF_NOPE":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"result":"error","error":"contractNotFound","serverTime":"2026-10-09T00:07:39.485Z"}`))
		}
	})
	var markets []string
	require.NoError(t, ex.SendFuturesHTTPRequest(t.Context(), exchange.RestFuturesSupplementary, "/charts/v1/trade", &markets), "SendFuturesHTTPRequest must not error")
	assert.Equal(t, []string{"PF_XBTUSD"}, markets, "SendFuturesHTTPRequest should join the supplementary URL's trailing slash and decode the response")

	err := ex.SendFuturesHTTPRequest(t.Context(), exchange.RestFutures, "/api/v3/tickers/PF_NOPE", &markets)
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "SendFuturesHTTPRequest should match a mapped error")
	assert.ErrorIs(t, err, request.ErrBadStatus, "SendFuturesHTTPRequest should keep the status error")
}

func TestSpotMessagesUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var msgs spotMessages
	require.NoError(t, msgs.UnmarshalJSON([]byte(`["EGeneral:Invalid arguments","WGeneral:Deprecated"]`)), "UnmarshalJSON must not error for an array")
	assert.Equal(t, spotMessages{"EGeneral:Invalid arguments", "WGeneral:Deprecated"}, msgs, "UnmarshalJSON should decode an array")
	require.NoError(t, msgs.UnmarshalJSON([]byte(`"EAPI:Invalid key"`)), "UnmarshalJSON must not error for a string")
	assert.Equal(t, spotMessages{"EAPI:Invalid key"}, msgs, "UnmarshalJSON should decode a string")
	assert.Error(t, msgs.UnmarshalJSON([]byte(`{}`)), "UnmarshalJSON should reject an object")
	assert.Error(t, msgs.UnmarshalJSON([]byte(`"EAPI:Invalid key`)), "UnmarshalJSON should reject an unterminated string")
}

func TestDecodeSpotResponse(t *testing.T) {
	t.Parallel()
	err := e.decodeSpotResponse([]byte(`<html>`), nil, nil)
	assert.ErrorContains(t, err, "error decoding Spot REST response", "decodeSpotResponse should reject a body that is not JSON")
	err = e.decodeSpotResponse([]byte(`<html>`), request.ErrBadStatus, nil)
	assert.ErrorIs(t, err, request.ErrBadStatus, "decodeSpotResponse should return the send error for a body that is not JSON")
	err = e.decodeSpotResponse([]byte(`{"error":[],"result":{}}`), nil, nil)
	assert.NoError(t, err, "decodeSpotResponse should accept a nil result")
	err = e.decodeSpotResponse([]byte(`{"error":[]}`), nil, new(map[string]any))
	assert.ErrorIs(t, err, common.ErrNoResponse, "decodeSpotResponse should reject a missing result")
}

func TestProblemError(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		raw  string
		exp  *APIError
	}{
		{name: "empty"},
		{name: "not JSON", raw: "<html>"},
		{name: "no title or detail", raw: `{"type":"tag:kraken.com,2025:InvalidKey"}`},
		{name: "title", raw: `{"type":"tag:kraken.com,2025:InvalidKey","title":"API: Invalid key"}`, exp: &APIError{Errors: []string{"API: Invalid key"}, Type: "tag:kraken.com,2025:InvalidKey"}},
		{name: "title and detail", raw: `{"title":"Bad request","detail":"limit is outside 1 to 200"}`, exp: &APIError{Errors: []string{"Bad request: limit is outside 1 to 200"}}},
		{name: "detail", raw: `{"detail":"limit is outside 1 to 200"}`, exp: &APIError{Errors: []string{"limit is outside 1 to 200"}}},
		{name: "unknown route", raw: `{"error":["EGeneral:Unknown method"]}`, exp: &APIError{Errors: []string{"EGeneral:Unknown method"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, problemError(json.RawMessage(tc.raw)), "problemError should describe the body")
		})
	}
}

func TestFuturesError(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		raw         string
		errorStatus bool
		exp         *APIError
	}{
		{name: "empty"},
		{name: "not JSON", raw: "[]", errorStatus: true},
		{name: "success", raw: `{"result":"success","error":"ignored"}`},
		{name: "error result", raw: `{"result":"error","error":"apiLimitExceeded"}`, exp: &APIError{Errors: []string{"apiLimitExceeded"}}},
		{name: "error result without detail", raw: `{"result":"error"}`, exp: &APIError{Errors: []string{`{"result":"error"}`}}},
		{name: "error status without detail", raw: `{"elements":[]}`, errorStatus: true},
		{name: "authentication error", raw: `{"error":"api_key_not_found","message":"API key not found","suberror":null}`, errorStatus: true, exp: &APIError{Errors: []string{"api_key_not_found", "API key not found"}}},
		{name: "history error", raw: `{"status":"unauthorized"}`, errorStatus: true, exp: &APIError{Errors: []string{"unauthorized"}, Type: "unauthorized"}},
		{name: "history error with a reason", raw: `{"status":"unauthorized","reason":"You are not authorized to access this endpoint."}`, errorStatus: true, exp: &APIError{Errors: []string{"You are not authorized to access this endpoint."}, Type: "unauthorized"}},
		{name: "analytics problem", raw: `{"result":null,"errors":[{"severity":"E","error_class":"General","type":"Invalid arguments","msg":"must be one of","value":"7","field":"interval"}]}`, errorStatus: true, exp: &APIError{Errors: []string{"EGeneral:Invalid arguments:interval:must be one of:7"}}},
		{name: "coded error without a code", raw: `{"result":"error","errors":[{"code":null,"message":"Not permitted"}]}`, exp: &APIError{Errors: []string{"Not permitted"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, futuresError(json.RawMessage(tc.raw), tc.errorStatus), "futuresError should describe the body")
		})
	}
}

func TestAPIError(t *testing.T) {
	t.Parallel()
	err := &APIError{Errors: []string{"EAPI:Invalid nonce", "EGeneral:Internal error"}}
	assert.Equal(t, "API error response: EAPI:Invalid nonce, EGeneral:Internal error", err.Error(), "Error should list every error")
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "APIError should match a mapped error")
	assert.ErrorIs(t, err, errAPIResponse, "APIError should match errAPIResponse")
	assert.NotErrorIs(t, err, order.ErrOrderNotFound, "APIError should not match an unmapped error")

	err = &APIError{Errors: []string{"API: Invalid key"}, Type: "tag:kraken.com,2025:InvalidKey"}
	assert.Equal(t, "API error response: API: Invalid key (tag:kraken.com,2025:InvalidKey)", err.Error(), "Error should name the problem type")
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "APIError should match the problem type's mapped error")
}

func TestDecodeErrorBody(t *testing.T) {
	t.Parallel()
	var resp futuresErrorResponse
	require.True(t, decodeErrorBody(json.RawMessage(`{"result":"error","error":"apiLimitExceeded"}`), &resp), "decodeErrorBody must decode an object")
	assert.Equal(t, futuresErrorResponse{Result: "error", Error: "apiLimitExceeded"}, resp, "decodeErrorBody should decode the error")
	for _, raw := range []string{``, `[]`, `"error"`, `{"result":1}`} {
		assert.Falsef(t, decodeErrorBody(json.RawMessage(raw), &resp), "decodeErrorBody should not decode %q", raw)
	}
}

func TestDecodeFuturesResponse(t *testing.T) {
	t.Parallel()
	errorBody := json.RawMessage(`{"result":"error","error":"apiLimitExceeded"}`)
	var apiErr *APIError
	err := decodeFuturesResponse(errorBody, nil, nil)
	require.ErrorAs(t, err, &apiErr, "decodeFuturesResponse must return the error a body describes")
	assert.Equal(t, []string{"apiLimitExceeded"}, apiErr.Errors, "decodeFuturesResponse should report the body's error")
	err = decodeFuturesResponse(errorBody, context.DeadlineExceeded, nil)
	assert.ErrorAs(t, err, &apiErr, "decodeFuturesResponse should return the error a failed request's body describes")
	assert.ErrorIs(t, err, context.DeadlineExceeded, "decodeFuturesResponse should keep the request's error beside the body's")
	err = decodeFuturesResponse(json.RawMessage(`<html>`), context.DeadlineExceeded, nil)
	assert.ErrorIs(t, err, context.DeadlineExceeded, "decodeFuturesResponse should return the request's error for a body describing none")
	assert.NoError(t, decodeFuturesResponse(json.RawMessage(`{"result":"success"}`), nil, nil), "decodeFuturesResponse should not decode without a result")
	var result FuturesNotificationsResponse
	for _, raw := range []string{``, `null`} {
		assert.ErrorIsf(t, decodeFuturesResponse(json.RawMessage(raw), nil, &result), common.ErrNoResponse, "decodeFuturesResponse should reject body %q", raw)
	}
	require.NoError(t, decodeFuturesResponse(json.RawMessage(`{"result":"success","serverTime":"2026-10-09T00:00:00.123Z"}`), nil, &result), "decodeFuturesResponse must not error")
	assert.Equal(t, time.Date(2026, 10, 9, 0, 0, 0, 123e6, time.UTC), result.ServerTime, "decodeFuturesResponse should decode the result")
}

func TestAPIErrorUnwrap(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []error{errAPIResponse}, (&APIError{Errors: []string{"EGeneral:Too many requests"}}).Unwrap(), "Unwrap should return only errAPIResponse for an error without a mapping")
	exp := []error{errAPIResponse, request.ErrAuthRequestFailed, order.ErrOrderNotFound, currency.ErrPairNotFound}
	got := (&APIError{Type: "tag:kraken.com,2025:InvalidKey", Errors: []string{"EOrder:Unknown order", "EGeneral:Too many requests", "contractNotFound"}}).Unwrap()
	assert.Equal(t, exp, got, "Unwrap should return the errors the type and each message map to")
}
