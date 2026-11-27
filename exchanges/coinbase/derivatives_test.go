package coinbase

import (
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
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
	ID      uint64          `json:"id"`
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

// derivativesRequestRefusalCodes are the codes of the gateway's errors that refuse or fail a request itself rather than
// answer what it asks for, such as those for a malformed request, invalid input or an invalid instrument, an unknown
// method, refused credentials or permissions, an account verification not completed or a subaccount restriction, a
// rate limit or an outage, from the error list of Deribit's API, which the gateway runs. Any other refusal is an
// answer, as a request naming nothing that exists, or another account's user, draws one
var derivativesRequestRefusalCodes = []int64{
	-32700, -32602, -32600, -32601, -32000, 10000, 10001, 10019, 10020, 10022, 10023, 10024, 10025, 10026, 10027, 10028,
	10030, 10031, 10033, 10040, 10041, 10043, 10044, 10047, 10048, 10066, 11029, 11031, 11042, 11043, 11047, 11049, 11050,
	11051, 11056, 11094, 11095, 11098, 12003, 12004, 12005, 12998, 13004, 13005, 13006, 13007, 13008, 13009, 13010, 13011,
	13012, 13015, 13016, 13018, 13020, 13021, 13025, 13028, 13031, 13037, 13403, 13503, 13666, 13777, 13778, 13780, 13792,
	13793, 13888, 13903, 13904,
}

// derivativesInvalidInputCodes are the request refusal codes for invalid input, whose data can name the parameter at
// fault
var derivativesInvalidInputCodes = []int64{-32602, 11029, 11043, 11049, 13016}

// derivativesInvalidParamsData is the data of the gateway's invalid input errors, which can name the parameter at fault
type derivativesInvalidParamsData struct {
	Param string `json:"param"`
}

// requireDerivativesAnsweredOnMerits fails a live case unless the gateway answered its request on the merits: the
// request succeeded, or the gateway refused what it asked for, as when the account lacks what the method needs. It
// reports whether the request succeeded
func requireDerivativesAnsweredOnMerits(t *testing.T, err error, method string) bool {
	t.Helper()
	if err == nil {
		return true
	}
	derivativesErr, ok := errors.AsType[*DerivativesError](err)
	require.Truef(t, ok, "%s must return the gateway's answer, not %v", method, err)
	require.NotContainsf(t, derivativesRequestRefusalCodes, derivativesErr.Code, "%s must be answered on its merits rather than refused as a request", method)
	return false
}

// requireDerivativesRefusedOnMerits fails a live case built to be refused unless its request fails and
// requireDerivativesAnsweredOnMerits accepts the error. The case fills foreignParams with values no account holds,
// which the gateway can refuse as invalid input, so an invalid input error naming one of them is accepted too
func requireDerivativesRefusedOnMerits(t *testing.T, err error, method string, foreignParams ...string) {
	t.Helper()
	require.Errorf(t, err, "%s must be refused, as its request is built to be refused", method)
	if derivativesErr, ok := errors.AsType[*DerivativesError](err); ok && slices.Contains(derivativesInvalidInputCodes, derivativesErr.Code) {
		var data derivativesInvalidParamsData
		if json.Unmarshal(derivativesErr.Data, &data) == nil && slices.Contains(foreignParams, data.Param) {
			return
		}
	}
	requireDerivativesAnsweredOnMerits(t, err, method)
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
		var auth DerivativesAuthWireRequest
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
	exp := &DerivativesAuthResponse{AccessToken: "token", ExpiresIn: types.DurationSeconds(15 * time.Minute), RefreshToken: "refresh", Scope: "trade view", TokenType: "bearer", EnabledFeatures: []string{}}
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
		// Past the attempts the request succeeds, so a retry that loses its limit fails the checks below rather than
		// retrying until go test's timeout
		if attempts > derivativesMaximumRequestAttempts {
			return nil
		}
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

	t.Run("adds nothing for an error sent without data", func(t *testing.T) {
		t.Parallel()
		var derivativesErr DerivativesError
		require.NoError(t, json.Unmarshal([]byte(`{"code":13009,"message":"unauthorized"}`), &derivativesErr), "Unmarshal must not error")
		assert.Equal(t, "code 13009: unauthorized", derivativesErr.Error(), "Error should add nothing to an error the gateway sent without data")
	})
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
	var params DerivativesAuthWireRequest
	require.NoError(t, json.Unmarshal(auth[0].Params, &params), "the auth params must decode")
	jwt := verifyTestJWT(t, params.Token)
	assert.NotContains(t, jwt.Claims, "uri", "the websocket JWT should not be bound to a request")
	// The gateway's public streams and its private connection each ask for heartbeats
	assert.Eventually(t, func() bool { return len(venue.requestsFor("public/set_heartbeat")) == 2 }, 30*time.Second, time.Millisecond, "each gateway connection should ask for heartbeats")

	var result string
	require.NoError(t, ex.WebsocketAPI().SendRequest(t.Context(), derivativesNonMatchingEPL, "public/set_heartbeat", &DerivativesHeartbeatWireRequest{Interval: 30}, &result), "SendRequest must not error")
	assert.Equal(t, "ok", result, "SendRequest should decode the result")
	err := ex.WebsocketAPI().SendRequest(t.Context(), derivativesNonMatchingEPL, derivativesAuthMethod, &DerivativesAuthWireRequest{GrantType: "client_credentials"}, &result)
	derivativesErr, ok := errors.AsType[*DerivativesError](err)
	require.True(t, ok, "SendRequest must return the error the gateway reports")
	assert.Equal(t, int64(13004), derivativesErr.Code, "SendRequest should decode the error")

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	err = unauthenticated.WebsocketAPI().SendRequest(t.Context(), derivativesNonMatchingEPL, "private/get_positions", nil, &result)
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "SendRequest should refuse a private method on an unauthenticated connection")
	err = newTestExchange(t).WebsocketAPI().SendRequest(t.Context(), derivativesNonMatchingEPL, "public/test", nil, &result)
	assert.Error(t, err, "SendRequest should error without a connection")
}

func TestGlobalDerivativesAPIAccessToken(t *testing.T) {
	t.Parallel()

	t.Run("concurrent requests share one renewal", func(t *testing.T) {
		t.Parallel()
		var authCalls atomic.Int64
		ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "Reading the request body should not error")
			if strings.Contains(string(body), derivativesAuthMethod) {
				authCalls.Add(1)
				// A renewal in flight lets the other requests find the cache empty, as concurrent requests do
				time.Sleep(20 * time.Millisecond)
				_, err = w.Write([]byte(`{"jsonrpc":"2.0","result":{"access_token":"renewed","expires_in":900}}`))
				assert.NoError(t, err, "Writing the response should not error")
				return
			}
			assert.Equal(t, "Bearer renewed", r.Header.Get("Authorization"), "every request should carry the renewed access token")
			_, err = w.Write([]byte(`{"jsonrpc":"2.0","result":"ok"}`))
			assert.NoError(t, err, "Writing the response should not error")
		}))
		ex.derivativesToken.clear(testAPIKey, testDerivativesAccessToken)
		// A renewal that is never released fails the requests waiting for it rather than holding the test
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		var wg sync.WaitGroup
		start := make(chan struct{})
		for range 10 {
			wg.Go(func() {
				<-start
				var resp string
				assert.NoError(t, ex.GlobalDerivatives().SendAuthenticatedHTTPRequest(ctx, derivativesNonMatchingEPL, "private/get_positions", nil, &resp), "SendAuthenticatedHTTPRequest should not error")
			})
		}
		close(start)
		wg.Wait()
		require.EqualValues(t, 1, authCalls.Load(), "concurrent requests must share one public/auth")
	})

	t.Run("a request waiting on another renewal stops when its context ends", func(t *testing.T) {
		t.Parallel()
		entered, release := make(chan struct{}), make(chan struct{})
		var authCalls atomic.Int32
		ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authCalls.Add(1) == 1 {
				close(entered)
				if !awaitRelease(t, r, release) {
					return
				}
			}
			writeJSONRPC(t, w, `{"jsonrpc":"2.0","result":{"access_token":"renewed","expires_in":900}}`)
		}))
		ex.derivativesToken.clear(testAPIKey, testDerivativesAccessToken)
		first := make(chan error, 1)
		go func() {
			_, _, err := ex.GlobalDerivatives().accessToken(t.Context())
			first <- err
		}()
		awaitClosed(t, entered, "the renewal must reach the gateway")
		// A request waiting for the renewal under way stops when its context ends
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		waited := make(chan error, 1)
		go func() {
			_, _, err := ex.GlobalDerivatives().accessToken(ctx)
			waited <- err
		}()
		select {
		case err := <-waited:
			assert.ErrorIs(t, err, context.Canceled, "accessToken should return its context's error while another renewal is under way")
		case <-time.After(30 * time.Second):
			assert.Fail(t, "accessToken should stop waiting for another renewal once its context ends")
		}
		close(release)
		select {
		case err := <-first:
			require.NoError(t, err, "the renewal under way must complete")
		case <-time.After(30 * time.Second):
			require.FailNow(t, "the renewal under way must complete")
		}
		assert.EqualValues(t, 1, authCalls.Load(), "the request whose context ended should not renew the token")
		_, token, err := ex.GlobalDerivatives().accessToken(t.Context())
		require.NoError(t, err, "accessToken must not error once a token is renewed")
		assert.Equal(t, "renewed", token, "accessToken should return the renewed token")
		assert.EqualValues(t, 1, authCalls.Load(), "accessToken should reuse the renewed token")
	})

	t.Run("a failed renewal lets the next request renew", func(t *testing.T) {
		t.Parallel()
		var authCalls atomic.Int32
		ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if authCalls.Add(1) == 1 {
				writeJSONRPC(t, w, `{"jsonrpc":"2.0","error":{"code":13004,"message":"invalid_credentials"}}`)
				return
			}
			writeJSONRPC(t, w, `{"jsonrpc":"2.0","result":{"access_token":"renewed","expires_in":900}}`)
		}))
		ex.derivativesToken.clear(testAPIKey, testDerivativesAccessToken)
		_, _, err := ex.GlobalDerivatives().accessToken(t.Context())
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "accessToken must return the refused renewal's error")
		// The failed renewal releases its turn, so the next request renews rather than waiting for it
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		_, token, err := ex.GlobalDerivatives().accessToken(ctx)
		require.NoError(t, err, "accessToken must renew the token after a failed renewal")
		assert.Equal(t, "renewed", token, "accessToken should return the renewed token")
		assert.EqualValues(t, 2, authCalls.Load(), "accessToken should renew once for each request")
	})

	t.Run("refuses a token whose lifetime is longer than a duration holds", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSONRPC(t, w, `{"jsonrpc":"2.0","result":{"access_token":"lasting","expires_in":18446744073709551615}}`)
		}))
		ex.derivativesToken.clear(testAPIKey, testDerivativesAccessToken)
		_, _, err := ex.GlobalDerivatives().accessToken(t.Context())
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "accessToken must refuse a lifetime longer than a duration holds")
		_, ok := ex.derivativesToken.get(testAPIKey)
		assert.False(t, ok, "accessToken should not keep a token whose lifetime it cannot hold")
	})

	t.Run("caches a token for the lifetime issued", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name, expiresIn string
			lifetime        time.Duration
		}{
			{"an hour", "3600", time.Hour},
			{"15 minutes", "900", 15 * time.Minute},
			{"a fractional lifetime", "1800.75", 30*time.Minute + 750*time.Millisecond},
			{"a quoted lifetime", `"1800"`, 30 * time.Minute},
			{"no lifetime", "0", 0},
			{"a negative lifetime", "-1", -time.Second},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					writeJSONRPC(t, w, `{"jsonrpc":"2.0","result":{"access_token":"issued","expires_in":`+tc.expiresIn+`}}`)
				}))
				ex.derivativesToken.clear(testAPIKey, testDerivativesAccessToken)
				before := time.Now()
				_, token, err := ex.GlobalDerivatives().accessToken(t.Context())
				after := time.Now()
				require.NoError(t, err, "accessToken must not error")
				assert.Equal(t, "issued", token, "accessToken should return the token issued")
				ex.derivativesToken.m.Lock()
				cached, ok := ex.derivativesToken.tokens[testAPIKey]
				ex.derivativesToken.m.Unlock()
				require.True(t, ok, "accessToken must cache the token issued")
				assert.Equal(t, "issued", cached.token, "accessToken should cache the token issued")
				assert.False(t, cached.expires.Before(before.Add(tc.lifetime)), "accessToken should cache the token until its lifetime ends")
				assert.False(t, cached.expires.After(after.Add(tc.lifetime)), "accessToken should cache the token no longer than its lifetime")
				_, reusable := ex.derivativesToken.get(testAPIKey)
				assert.Equal(t, tc.lifetime >= derivativesTokenRefreshMargin, reusable, "accessToken should let later requests reuse only a token lasting beyond the refresh margin")
			})
		}
	})

	t.Run("keeps a token lasting 9223372036 seconds, the most whole seconds a duration holds", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSONRPC(t, w, `{"jsonrpc":"2.0","result":{"access_token":"lasting","expires_in":9223372036}}`)
		}))
		ex.derivativesToken.clear(testAPIKey, testDerivativesAccessToken)
		_, token, err := ex.GlobalDerivatives().accessToken(t.Context())
		require.NoError(t, err, "accessToken must accept the most whole seconds a duration holds")
		assert.Equal(t, "lasting", token, "accessToken should return the token issued")
		token, ok := ex.derivativesToken.get(testAPIKey)
		assert.True(t, ok, "accessToken should keep a token lasting the most whole seconds a duration holds")
		assert.Equal(t, "lasting", token, "accessToken should keep the token issued")
		ex.derivativesToken.m.Lock()
		cached := ex.derivativesToken.tokens[testAPIKey]
		ex.derivativesToken.m.Unlock()
		assert.Greater(t, time.Until(cached.expires), 9223372036*time.Second-time.Hour, "accessToken should cache the token for its whole lifetime")
	})

	t.Run("refuses a token lasting 9223372037 seconds, past what a duration holds", func(t *testing.T) {
		t.Parallel()
		ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSONRPC(t, w, `{"jsonrpc":"2.0","result":{"access_token":"lasting","expires_in":9223372037}}`)
		}))
		ex.derivativesToken.clear(testAPIKey, testDerivativesAccessToken)
		_, token, err := ex.GlobalDerivatives().accessToken(t.Context())
		require.ErrorIs(t, err, request.ErrAuthRequestFailed, "accessToken must refuse a lifetime longer than a duration holds")
		assert.Empty(t, token, "accessToken should return no token for a lifetime it cannot hold")
		ex.derivativesToken.m.Lock()
		_, ok := ex.derivativesToken.tokens[testAPIKey]
		ex.derivativesToken.m.Unlock()
		assert.False(t, ok, "accessToken should cache no token whose lifetime it cannot hold")
	})
}

func TestDerivativesTimesBeforeEpochRefused(t *testing.T) { //nolint:tparallel // the cases share g, its server and timeRange in order
	t.Parallel()
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Failf(t, "a request with a time before 1970 should not be sent", "requested %s", r.URL.Path)
		http.NotFound(w, r)
	}))
	g := ex.GlobalDerivatives()
	before, after := time.Date(1969, 12, 31, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	timeRange := &DerivativesInstrumentTimeRangeRequest{InstrumentName: "BTC-PERPETUAL", StartTime: before, EndTime: after}
	// The cases share g, its server and timeRange one at a time, so each also checks that a call keeps nothing from the last
	for name, call := range map[string]func() error{
		"GetFundingRateHistory": func() error { _, err := g.GetFundingRateHistory(t.Context(), timeRange); return err },
		"GetFundingRateValue":   func() error { _, err := g.GetFundingRateValue(t.Context(), timeRange); return err },
		"GetMarkPriceHistory":   func() error { _, err := g.GetMarkPriceHistory(t.Context(), timeRange); return err },
		"GetLastSettlementsByCurrency": func() error {
			_, err := g.GetLastSettlementsByCurrency(t.Context(), &DerivativesLastSettlementsByCurrencyRequest{Currency: currency.BTC, SearchStartTime: before})
			return err
		},
		"GetLastSettlementsByInstrument": func() error {
			_, err := g.GetLastSettlementsByInstrument(t.Context(), &DerivativesLastSettlementsByInstrumentRequest{InstrumentName: "BTC-PERPETUAL", SearchStartTime: before})
			return err
		},
		"GetLastTradesByCurrency": func() error {
			_, err := g.GetLastTradesByCurrency(t.Context(), &DerivativesLastTradesByCurrencyRequest{Currency: currency.BTC, StartTime: before, EndTime: after})
			return err
		},
		"GetLastTradesByCurrencyAndTime": func() error {
			_, err := g.GetLastTradesByCurrencyAndTime(t.Context(), &DerivativesLastTradesByCurrencyAndTimeRequest{Currency: currency.BTC, StartTime: before, EndTime: after})
			return err
		},
		"GetLastTradesByInstrument": func() error {
			_, err := g.GetLastTradesByInstrument(t.Context(), &DerivativesLastTradesByInstrumentRequest{InstrumentName: "BTC-PERPETUAL", StartTime: before, EndTime: after})
			return err
		},
		"GetLastTradesByInstrumentAndTime": func() error {
			_, err := g.GetLastTradesByInstrumentAndTime(t.Context(), &DerivativesLastTradesByInstrumentAndTimeRequest{InstrumentName: "BTC-PERPETUAL", StartTime: before, EndTime: after})
			return err
		},
		"GetTradingViewChartData": func() error {
			_, err := g.GetTradingViewChartData(t.Context(), &DerivativesTradingViewChartDataRequest{InstrumentName: "BTC-PERPETUAL", StartTime: before, EndTime: after, Resolution: kline.OneHour})
			return err
		},
		"GetVolatilityIndexData": func() error {
			_, err := g.GetVolatilityIndexData(t.Context(), &DerivativesVolatilityIndexDataRequest{Currency: currency.BTC, StartTime: before, EndTime: after, Resolution: kline.OneHour})
			return err
		},
		"GetTransactionLog": func() error {
			_, err := g.GetTransactionLog(t.Context(), &DerivativesGetTransactionLogRequest{Currency: currency.BTC, StartTime: before, EndTime: after})
			return err
		},
		"GetAnnouncements": func() error {
			_, err := g.GetAnnouncements(t.Context(), &DerivativesGetAnnouncementsRequest{StartTime: before})
			return err
		},
		"Buy": func() error {
			_, err := g.Buy(t.Context(), &DerivativesOrderRequest{InstrumentName: "BTC-PERPETUAL", Amount: 10, Type: DerivativesOrderTypeLimit, Price: 80000, ValidUntil: before})
			return err
		},
		"EditOrder": func() error {
			_, err := g.EditOrder(t.Context(), &DerivativesEditOrderRequest{OrderID: "1", Amount: 10, Price: 80000, ValidUntil: before})
			return err
		},
		"EditOrderByLabel": func() error {
			_, err := g.EditOrderByLabel(t.Context(), &DerivativesEditOrderByLabelRequest{Label: "l", InstrumentName: "BTC-PERPETUAL", Amount: 10, Price: 80000, ValidUntil: before})
			return err
		},
		"GetSettlementHistoryByCurrency": func() error {
			_, err := g.GetSettlementHistoryByCurrency(t.Context(), &DerivativesSettlementHistoryByCurrencyRequest{Currency: currency.BTC, SearchStartTime: before})
			return err
		},
		"GetSettlementHistoryByInstrument": func() error {
			_, err := g.GetSettlementHistoryByInstrument(t.Context(), &DerivativesSettlementHistoryByInstrumentRequest{InstrumentName: "BTC-PERPETUAL", SearchStartTime: before})
			return err
		},
		"GetUserTradesByCurrency": func() error {
			_, err := g.GetUserTradesByCurrency(t.Context(), &DerivativesUserTradesByCurrencyRequest{Currency: currency.BTC, StartTime: before, EndTime: after})
			return err
		},
		"GetUserTradesByCurrencyAndTime": func() error {
			_, err := g.GetUserTradesByCurrencyAndTime(t.Context(), &DerivativesUserTradesByCurrencyAndTimeRequest{Currency: currency.BTC, StartTime: before, EndTime: after})
			return err
		},
		"GetUserTradesByInstrument": func() error {
			_, err := g.GetUserTradesByInstrument(t.Context(), &DerivativesUserTradesByInstrumentRequest{InstrumentName: "BTC-PERPETUAL", StartTime: before, EndTime: after})
			return err
		},
		"GetUserTradesByInstrumentAndTime": func() error {
			_, err := g.GetUserTradesByInstrumentAndTime(t.Context(), &DerivativesUserTradesByInstrumentAndTimeRequest{InstrumentName: "BTC-PERPETUAL", StartTime: before, EndTime: after})
			return err
		},
	} {
		t.Run(subtestName(name), func(t *testing.T) {
			assert.ErrorIsf(t, call(), errDerivativesTimeBeforeEpoch, "%s should refuse a time before 1970", name)
		})
	}
}

func TestDerivativesRequestID(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	first, second := ex.derivativesRequestID(), ex.derivativesRequestID()
	assert.Positive(t, first, "derivativesRequestID should count up from one")
	assert.Equal(t, first+1, second, "derivativesRequestID should give each request the next ID")
}
