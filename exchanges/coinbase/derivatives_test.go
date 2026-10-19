package coinbase

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// writeJSONRPC writes a JSON-RPC response body, with the HTTP status the gateway answers an error with
func writeJSONRPC(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	if strings.Contains(body, `"error"`) {
		w.WriteHeader(http.StatusBadRequest)
	}
	_, err := w.Write([]byte(body))
	assert.NoError(t, err, "Writing the response should not error")
}

// receivedJSONRPC is a JSON-RPC request a test server received
type receivedJSONRPC struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// readJSONRPC reads a JSON-RPC request body, reporting a failure with assert, as it runs in a handler
func readJSONRPC(t *testing.T, r *http.Request) (receivedJSONRPC, bool) {
	t.Helper()
	var req receivedJSONRPC
	body, err := io.ReadAll(r.Body)
	if !assert.NoError(t, err, "Reading the request body should not error") {
		return req, false
	}
	return req, assert.NoError(t, json.Unmarshal(body, &req), "The request body should be JSON")
}

// testDerivativesAuthHandler answers public/auth with an access token, for test servers of private methods
func testDerivativesAuthHandler(t *testing.T, w http.ResponseWriter, token string) {
	t.Helper()
	writeJSONRPC(t, w, `{"jsonrpc":"2.0","result":{"access_token":"`+token+`","expires_in":900,"scope":"trade view","token_type":"bearer"}}`)
}

func TestGlobalDerivativesAuthenticate(t *testing.T) {
	t.Parallel()
	var host string
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "Authenticate should post its request")
		assert.Equal(t, "/api/v2/public/auth", r.URL.Path, "Authenticate should call public/auth")
		assert.Empty(t, r.Header.Get("Authorization"), "Authenticate should not send an access token")
		req, ok := readJSONRPC(t, r)
		if !ok {
			return
		}
		assert.Equal(t, "2.0", req.JSONRPC, "Authenticate should send a JSON-RPC 2.0 envelope")
		assert.Zero(t, req.ID, "Authenticate should leave the ID out over HTTP")
		assert.Equal(t, derivativesAuthMethod, req.Method, "Authenticate should name the method")
		var auth derivativesAuthParams
		if assert.NoError(t, json.Unmarshal(req.Params, &auth), "The params should decode") {
			assert.Equal(t, derivativesGrantType, auth.GrantType, "Authenticate should exchange a CDP JWT")
			jwt, err := parseTestJWT(auth.Token)
			if assert.NoError(t, err, "The JWT should verify") {
				assert.Equal(t, "POST "+host+"/api/v2/public/auth", jwt.Claims["uri"], "Authenticate should bind the JWT to the public/auth request")
			}
		}
		writeJSONRPC(t, w, `{"jsonrpc":"2.0","result":{"access_token":"token","expires_in":900,"refresh_token":"refresh","scope":"trade view","token_type":"bearer","enabled_features":[]},"testnet":false}`)
	}))
	endpoint, err := ex.API.Endpoints.GetURL(exchange.RestFutures)
	require.NoError(t, err, "GetURL must not error")
	host, err = endpointHost(endpoint)
	require.NoError(t, err, "endpointHost must not error")
	resp, err := ex.GlobalDerivatives().Authenticate(t.Context())
	require.NoError(t, err, "Authenticate must not error")
	exp := &DerivativesAuthResponse{AccessToken: "token", ExpiresIn: 900, RefreshToken: "refresh", Scope: "trade view", TokenType: "bearer", EnabledFeatures: []string{}}
	assert.Equal(t, exp, resp, "Authenticate should decode every field")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().Authenticate(t.Context())
	assert.ErrorIs(t, err, errDerivativesAccessToken, "Authenticate should error when no token is issued")
	_, err = newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSONRPC(t, w, `{"jsonrpc":"2.0","error":{"code":13004,"message":"invalid_credentials"}}`)
	})).GlobalDerivatives().Authenticate(t.Context())
	derivativesErr, ok := errors.AsType[*DerivativesError](err)
	require.True(t, ok, "Authenticate must return the error the gateway reports")
	assert.Equal(t, &DerivativesError{Code: 13004, Message: "invalid_credentials"}, derivativesErr, "Authenticate should decode the error")
}

func TestGlobalDerivativesSendHTTPRequest(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"), "SendHTTPRequest should not send an access token")
		switch r.URL.Path {
		case "/api/v2/public/ticker":
			assert.Equal(t, http.MethodPost, r.Method, "SendHTTPRequest should post a JSON-RPC body")
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "SendHTTPRequest should send the body as JSON")
			req, ok := readJSONRPC(t, r)
			if ok {
				assert.Equal(t, "public/ticker", req.Method, "SendHTTPRequest should name the method")
				assert.JSONEq(t, `{"instrument_name":"BTC_USDC-PERPETUAL"}`, string(req.Params), "SendHTTPRequest should send the params")
			}
			writeJSONRPC(t, w, `{"jsonrpc":"2.0","result":{"last_price":1.5}}`)
		case "/api/v2/public/get_tradingview_chart_data":
			assert.Equal(t, http.MethodGet, r.Method, "SendHTTPRequest should send a query method as a GET request")
			assert.Equal(t, "end_timestamp=2&instrument_name=BTC_USDC-PERPETUAL&resolution=60&start_timestamp=1", r.URL.Query().Encode(), "SendHTTPRequest should send the params in the query string")
			writeJSONRPC(t, w, `{"jsonrpc":"2.0","result":{"status":"ok"}}`)
		default:
			assert.Failf(t, "unexpected request", "path %s", r.URL.Path)
		}
	}))
	var ticker struct {
		LastPrice float64 `json:"last_price"`
	}
	require.NoError(t, ex.GlobalDerivatives().SendHTTPRequest(t.Context(), derivativesNonMatchingEPL, "public/ticker", map[string]string{"instrument_name": "BTC_USDC-PERPETUAL"}, &ticker), "SendHTTPRequest must not error")
	assert.Equal(t, 1.5, ticker.LastPrice, "SendHTTPRequest should decode the result")
	var chart struct {
		Status string `json:"status"`
	}
	params := map[string]any{"instrument_name": "BTC_USDC-PERPETUAL", "start_timestamp": 1, "end_timestamp": 2, "resolution": "60"}
	require.NoError(t, ex.GlobalDerivatives().SendHTTPRequest(t.Context(), derivativesNonMatchingEPL, "public/get_tradingview_chart_data", params, &chart), "SendHTTPRequest must not error for a query method")
	assert.Equal(t, "ok", chart.Status, "SendHTTPRequest should decode a query method's result")

	err := newResponseServerExchange(t, `{"jsonrpc":"2.0","error":{"code":-32602,"message":"Invalid params"}}`).GlobalDerivatives().SendHTTPRequest(t.Context(), derivativesNonMatchingEPL, "public/ticker", nil, &ticker)
	derivativesErr, ok := errors.AsType[*DerivativesError](err)
	require.True(t, ok, "SendHTTPRequest must return the error the gateway reports")
	assert.Equal(t, int64(-32602), derivativesErr.Code, "SendHTTPRequest should decode the error code")
	err = newUnavailableServerExchange(t).GlobalDerivatives().SendHTTPRequest(t.Context(), derivativesNonMatchingEPL, "public/ticker", nil, &ticker)
	assert.ErrorIs(t, err, request.ErrBadStatus, "SendHTTPRequest should return a request failure without a reported error")
	assert.Error(t, ex.GlobalDerivatives().SendHTTPRequest(t.Context(), derivativesNonMatchingEPL, "public/ticker", make(chan int), &ticker), "SendHTTPRequest should error for params that cannot be marshalled")
	assert.Error(t, ex.GlobalDerivatives().SendHTTPRequest(t.Context(), derivativesNonMatchingEPL, "public/get_tradingview_chart_data", make(chan int), &ticker), "SendHTTPRequest should error for query params that cannot be marshalled")
}

func TestGlobalDerivativesSendAuthenticatedHTTPRequest(t *testing.T) {
	t.Parallel()
	var auths, refusals atomic.Int64
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/public/auth":
			testDerivativesAuthHandler(t, w, "fresh-token-"+strconv.FormatInt(auths.Add(1), 10))
		case "/api/v2/private/get_positions":
			switch r.Header.Get("Authorization") {
			case "Bearer " + testDerivativesAccessToken:
				// The cached token has been revoked, so the request is answered as the gateway answers it
				refusals.Add(1)
				writeJSONRPC(t, w, `{"jsonrpc":"2.0","error":{"code":13009,"message":"unauthorized","data":{"reason":"invalid_token"}}}`)
			case "Bearer fresh-token-1":
				writeJSONRPC(t, w, `{"jsonrpc":"2.0","result":[]}`)
			default:
				assert.Failf(t, "unexpected token", "header %q", r.Header.Get("Authorization"))
			}
		default:
			assert.Failf(t, "unexpected request", "path %s", r.URL.Path)
		}
	}))
	var positions []json.RawMessage
	require.NoError(t, ex.GlobalDerivatives().SendAuthenticatedHTTPRequest(t.Context(), derivativesNonMatchingEPL, "private/get_positions", nil, &positions), "SendAuthenticatedHTTPRequest must replace a refused token")
	assert.Equal(t, int64(1), refusals.Load(), "SendAuthenticatedHTTPRequest should send the cached token first")
	assert.Equal(t, int64(1), auths.Load(), "SendAuthenticatedHTTPRequest should exchange a JWT for one new token")
	token, ok := ex.derivativesToken.get(testAPIKey)
	require.True(t, ok, "the new token must be cached")
	assert.Equal(t, "fresh-token-1", token, "SendAuthenticatedHTTPRequest should cache the new token")
	require.NoError(t, ex.GlobalDerivatives().SendAuthenticatedHTTPRequest(t.Context(), derivativesNonMatchingEPL, "private/get_positions", nil, &positions), "SendAuthenticatedHTTPRequest must not error with the cached token")
	assert.Equal(t, int64(1), auths.Load(), "SendAuthenticatedHTTPRequest should reuse the cached token")

	// A token refused again after its replacement is reported, not replaced forever
	err := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/public/auth" {
			testDerivativesAuthHandler(t, w, "refused-token")
			return
		}
		writeJSONRPC(t, w, `{"jsonrpc":"2.0","error":{"code":13009,"message":"unauthorized","data":{"reason":"invalid_token"}}}`)
	})).GlobalDerivatives().SendAuthenticatedHTTPRequest(t.Context(), derivativesNonMatchingEPL, "private/get_positions", nil, &positions)
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "SendAuthenticatedHTTPRequest should mark the failed authenticated request")
	derivativesErr, ok := errors.AsType[*DerivativesError](err)
	require.True(t, ok, "SendAuthenticatedHTTPRequest must return the error the gateway reports")
	assert.Equal(t, int64(derivativesErrorCodeUnauthorised), derivativesErr.Code, "SendAuthenticatedHTTPRequest should return the refusal")

	// A failed token exchange fails the request as an authentication failure
	authErrEx := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSONRPC(t, w, `{"jsonrpc":"2.0","error":{"code":13004,"message":"invalid_credentials"}}`)
	}))
	authErrEx.derivativesToken.clear(testAPIKey, testDerivativesAccessToken)
	err = authErrEx.GlobalDerivatives().SendAuthenticatedHTTPRequest(t.Context(), derivativesNonMatchingEPL, "private/get_positions", nil, &positions)
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "SendAuthenticatedHTTPRequest should report a failed token exchange as an authentication failure")
}

func TestGlobalDerivativesSendRequest(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/public/test":
			assert.Empty(t, r.Header.Get("Authorization"), "sendRequest should not send an access token for a public method")
			writeJSONRPC(t, w, `{"jsonrpc":"2.0","result":"ok"}`)
		case "/api/v2/private/enable_cancel_on_disconnect":
			assert.Equal(t, "Bearer "+testDerivativesAccessToken, r.Header.Get("Authorization"), "sendRequest should send the access token for a private method")
			writeJSONRPC(t, w, `{"jsonrpc":"2.0","result":"not ok"}`)
		}
	}))
	assert.NoError(t, ex.GlobalDerivatives().sendOKRequest(t.Context(), false, "public/test", nil), "sendOKRequest should accept an ok result")
	err := ex.GlobalDerivatives().sendOKRequest(t.Context(), false, "private/enable_cancel_on_disconnect", nil)
	assert.ErrorIs(t, err, errDerivativesUnexpectedResult, "sendOKRequest should refuse any other result")
	err = newTestExchange(t).GlobalDerivatives().sendOKRequest(t.Context(), true, "private/enable_cancel_on_disconnect", nil)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "sendOKRequest should send over the websocket when asked")
}

func TestDerivativesRetry(t *testing.T) {
	t.Parallel()
	retry := &DerivativesError{Code: derivativesErrorCodeRetry, Message: "retry"}
	var attempts int
	err := derivativesRetry(t.Context(), func() error {
		attempts++
		if attempts < 3 {
			return retry
		}
		return nil
	})
	require.NoError(t, err, "derivativesRetry must stop once a retried request succeeds")
	assert.Equal(t, 3, attempts, "derivativesRetry should retry a request the gateway asks to retry")

	attempts = 0
	err = derivativesRetry(t.Context(), func() error {
		attempts++
		return retry
	})
	assert.ErrorIs(t, err, retry, "derivativesRetry should return the last refusal")
	assert.Equal(t, derivativesMaximumRequestAttempts, attempts, "derivativesRetry should give up after its attempts")

	attempts = 0
	other := &DerivativesError{Code: 10001, Message: "error"}
	assert.ErrorIs(t, derivativesRetry(t.Context(), func() error { attempts++; return other }), other, "derivativesRetry should return any other error at once")
	assert.Equal(t, 1, attempts, "derivativesRetry should not retry any other error")

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	assert.ErrorIs(t, derivativesRetry(ctx, func() error { return retry }), context.Canceled, "derivativesRetry should stop waiting when the context ends")
}

func TestDerivativesQueryParams(t *testing.T) {
	t.Parallel()
	values, err := derivativesQueryParams(nil)
	require.NoError(t, err, "derivativesQueryParams must not error for nil params")
	assert.Empty(t, values, "derivativesQueryParams should send nothing for nil params")
	values, err = derivativesQueryParams(map[string]any{"instrument_name": "BTC-PERPETUAL", "start_timestamp": int64(1792368000000), "flag": true, "price": 0.5})
	require.NoError(t, err, "derivativesQueryParams must not error")
	assert.Equal(t, "flag=true&instrument_name=BTC-PERPETUAL&price=0.5&start_timestamp=1792368000000", values.Encode(), "derivativesQueryParams should send strings as they are and other values as JSON")
	_, err = derivativesQueryParams(make(chan int))
	assert.Error(t, err, "derivativesQueryParams should error for params that cannot be marshalled")
	_, err = derivativesQueryParams([]int{1})
	assert.Error(t, err, "derivativesQueryParams should error for params that are not an object")
}

func TestDerivativesAccessToken(t *testing.T) {
	t.Parallel()
	var tokens derivativesAccessToken
	_, ok := tokens.get("key")
	assert.False(t, ok, "get should report no token for an unknown key")
	tokens.set("key", "token", time.Now().Add(time.Hour))
	token, ok := tokens.get("key")
	assert.True(t, ok, "get should return a cached token")
	assert.Equal(t, "token", token, "get should return the key's token")
	_, ok = tokens.get("other")
	assert.False(t, ok, "get should not return another key's token")
	tokens.set("key", "expiring", time.Now().Add(derivativesTokenRefreshMargin/2))
	_, ok = tokens.get("key")
	assert.False(t, ok, "get should not return a token about to expire")
	tokens.set("key", "token", time.Now().Add(time.Hour))
	tokens.clear("key", "older")
	_, ok = tokens.get("key")
	assert.True(t, ok, "clear should keep a token another request replaced")
	tokens.clear("key", "token")
	_, ok = tokens.get("key")
	assert.False(t, ok, "clear should drop the given token")
}

func TestDerivativesErrorError(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "code 13009: unauthorized {\"reason\":\"invalid_token\"}", (&DerivativesError{Code: 13009, Message: "unauthorized", Data: json.RawMessage(`{"reason":"invalid_token"}`)}).Error(), "Error should add the error's data")
	assert.Equal(t, "code -32601: Method not found", (&DerivativesError{Code: -32601, Message: "Method not found", Data: json.RawMessage(`null`)}).Error(), "Error should leave null data out")
}

func TestIsDerivativesPrivateMethod(t *testing.T) {
	t.Parallel()
	assert.True(t, isDerivativesPrivateMethod("private/buy"), "isDerivativesPrivateMethod should report a private method")
	assert.False(t, isDerivativesPrivateMethod("public/ticker"), "isDerivativesPrivateMethod should not report a public method")
}

func TestWebsocketAPISendRequest(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("the mock gateway websocket records the requests sent; live tests have none")
	}
	ex, venue := newDerivativesWebsocketTestExchange(t)
	auth := venue.requestsFor(derivativesAuthMethod)
	require.Len(t, auth, 1, "the connection must authenticate once")
	var params derivativesAuthParams
	require.NoError(t, json.Unmarshal(auth[0].Params, &params), "the auth params must decode")
	jwt := verifyTestJWT(t, params.Token)
	assert.NotContains(t, jwt.Claims, "uri", "the websocket JWT should not be bound to a request")
	// The gateway's public streams and its private connection each ask for heartbeats
	assert.Eventually(t, func() bool { return len(venue.requestsFor("public/set_heartbeat")) == 2 }, time.Second, time.Millisecond, "each gateway connection should ask for heartbeats")

	var result string
	require.NoError(t, ex.WebsocketAPI().SendRequest(t.Context(), derivativesNonMatchingEPL, "public/set_heartbeat", &derivativesHeartbeatParams{Interval: 30}, &result), "SendRequest must not error")
	assert.Equal(t, "ok", result, "SendRequest should decode the result")
	err := ex.WebsocketAPI().SendRequest(t.Context(), derivativesNonMatchingEPL, derivativesAuthMethod, &derivativesAuthParams{GrantType: "client_credentials"}, &result)
	derivativesErr, ok := errors.AsType[*DerivativesError](err)
	require.True(t, ok, "SendRequest must return the error the gateway reports")
	assert.Equal(t, int64(13004), derivativesErr.Code, "SendRequest should decode the error")

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	err = unauthenticated.WebsocketAPI().SendRequest(t.Context(), derivativesNonMatchingEPL, "private/get_positions", nil, &result)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "SendRequest should refuse a private method on an unauthenticated connection")
	err = newTestExchange(t).WebsocketAPI().SendRequest(t.Context(), derivativesNonMatchingEPL, "public/test", nil, &result)
	assert.Error(t, err, "SendRequest should error without a connection")
}
