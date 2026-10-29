package coinbase

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testutils "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
)

const (
	// canManipulateRealOrders lets live tests with credentials change the account they act for: place, edit and
	// cancel orders priced away from the market, move funds and change settings, restoring them afterwards
	canManipulateRealOrders = false
	// testAPIKey is a placeholder CDP API key name, which the mock responses and the test servers accept
	testAPIKey = "organizations/mock-organisation/apiKeys/mock-key"
	// testDerivativesAccessToken is the placeholder gateway access token mock tests start with, so private gateway
	// requests match their recorded fixtures without exchanging a JWT first
	testDerivativesAccessToken = "mock-derivatives-access-token"
)

var e *Exchange

// testSigningKey is a placeholder CDP secret, an ECDSA key generated for the test binary; nothing verifies it but
// the tests that check the signatures made with it
var testSigningKey = sync.OnceValues(func() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
})

// newTestCredentials returns the placeholder credentials, the placeholder key with the generated secret as SEC1 PEM
func newTestCredentials() (*accounts.Credentials, error) {
	key, err := testSigningKey()
	if err != nil {
		return nil, err
	}
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	return &accounts.Credentials{Key: testAPIKey, Secret: string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}))}, nil
}

// testCredentials returns the placeholder credentials
func testCredentials(tb testing.TB) *accounts.Credentials {
	tb.Helper()
	creds, err := newTestCredentials()
	require.NoError(tb, err, "newTestCredentials must not error")
	return creds
}

// testExchangeConfig is Coinbase's configuration from testdata/configtest.json, loaded once; testexch.Setup loads every
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
	exchCfg, err := cfg.GetExchangeConfig("Coinbase")
	if err != nil {
		return nil, err
	}
	return json.Marshal(exchCfg)
})

// setupTestExchange sets an exchange up as testexch.Setup does, from its own copy of Coinbase's test configuration
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

// setupDerivativesWebsocket gives an exchange a websocket holding only the gateway's connections, so tests connecting
// every connection without subscriptions leave out Advanced Trade's, which Coinbase closes when nothing is subscribed
// within 5 seconds; the manager answers a closed connection by reconnecting every connection
func setupDerivativesWebsocket(ex *Exchange) error {
	raw, err := testExchangeConfig()
	if err != nil {
		return err
	}
	var exchCfg config.Exchange
	if err := json.Unmarshal(raw, &exchCfg); err != nil {
		return err
	}
	ex.Websocket = sharedtestvalues.NewTestWebsocket()
	if err := ex.Websocket.Setup(ex.websocketManagerSetup(&exchCfg)); err != nil {
		return err
	}
	setups, err := ex.websocketConnectionSetups(&exchCfg)
	if err != nil {
		return err
	}
	for _, setup := range setups {
		if setup.MessageFilter != derivativesStreamsConnection && setup.MessageFilter != derivativesPrivateConnection {
			continue
		}
		if err := ex.Websocket.SetupNewConnection(setup); err != nil {
			return err
		}
	}
	return nil
}

// stopWebsocketTestExchange disables an exchange's websocket, so its connection monitor cannot redial a closed mock,
// and shuts it down
func stopWebsocketTestExchange(t *testing.T, ex *Exchange) {
	t.Helper()
	if ex.Websocket.IsEnabled() {
		assert.NoError(t, ex.Websocket.Disable(), "Disable should not error")
	}
	if err := ex.Websocket.Shutdown(); err != nil {
		assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should only fail when the connection monitor has shut the connection down first")
	}
}

// setTestCredentials enables authenticated support with the given credentials
func setTestCredentials(ex *Exchange, creds *accounts.Credentials) {
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(creds)
}

// newTestServerExchange returns an exchange, with placeholder credentials and gateway access token, whose REST
// requests to every API the handler serves, for responses the mock recording cannot express, such as malformed
// payloads and errors
func newTestServerExchange(t *testing.T, handler http.Handler) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, setupTestExchange(ex), "setupTestExchange must not error")
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	ex.Name = t.Name()
	setTestCredentials(ex, testCredentials(t))
	ex.SkipAuthCheck = true
	ex.derivativesToken.set(testAPIKey, testDerivativesAccessToken, time.Now().Add(time.Hour))
	// A pager that stops advancing asks forever, so the server refuses requests past what any test sends, failing the
	// test rather than hanging it
	var requests atomic.Int32
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) > testServerRequestLimit {
			assert.Failf(t, "the test server should not be asked forever", "request %s", r.URL)
			http.Error(w, `{"error":"INTERNAL","message":"request limit reached"}`, http.StatusInternalServerError)
			return
		}
		handler.ServeHTTP(w, r)
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	for _, ep := range []exchange.URL{exchange.RestSpot, exchange.RestSpotSupplementary, exchange.RestFutures} {
		require.NoErrorf(t, ex.API.Endpoints.SetRunningURL(ep.String(), server.URL), "SetRunningURL must not error for %s", ep)
	}
	return ex
}

// testServerRequestLimit is the most requests a test server answers, well above the 18 the busiest test sends
const testServerRequestLimit = 100

// newResponseServerExchange returns an exchange whose REST requests all receive one fixed response body
func newResponseServerExchange(t *testing.T, body string) *Exchange {
	t.Helper()
	return newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Writing the response should not error")
	}))
}

// newUnavailableServerExchange returns an exchange whose REST requests all fail without a body Coinbase would send
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

// jwtParts is a JWT's decoded header and claims
type jwtParts struct {
	Header map[string]any
	Claims map[string]any
}

// parseTestJWT verifies a JWT's signature with the placeholder signing key and returns its header and claims
func parseTestJWT(token string) (jwtParts, error) {
	var decoded jwtParts
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return decoded, fmt.Errorf("%w: %d parts", errTestJWT, len(parts))
	}
	for i, dst := range []*map[string]any{&decoded.Header, &decoded.Claims} {
		raw, err := base64.RawURLEncoding.DecodeString(parts[i])
		if err != nil {
			return decoded, err
		}
		if err := json.Unmarshal(raw, dst); err != nil {
			return decoded, err
		}
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return decoded, err
	}
	if len(sig) != 64 {
		return decoded, fmt.Errorf("%w: %d byte signature", errTestJWT, len(sig))
	}
	key, err := testSigningKey()
	if err != nil {
		return decoded, err
	}
	hash := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if !ecdsa.Verify(&key.PublicKey, hash[:], new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])) {
		return decoded, fmt.Errorf("%w: signature does not verify", errTestJWT)
	}
	return decoded, nil
}

var errTestJWT = errors.New("invalid test JWT")

// verifyTestJWT verifies a JWT's signature with the placeholder signing key and returns its header and claims
func verifyTestJWT(t *testing.T, token string) jwtParts {
	t.Helper()
	decoded, err := parseTestJWT(token)
	require.NoError(t, err, "the JWT must verify with the signing key")
	return decoded
}

func TestSendHTTPRequest(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "SendHTTPRequest should send a GET request")
		assert.Empty(t, r.Header.Get("Authorization"), "SendHTTPRequest should not authenticate")
		switch r.URL.Path {
		case "/api/v3/brokerage/market/products/BTC-USD":
			assert.Equal(t, "a=1", r.URL.RawQuery, "SendHTTPRequest should send the query string")
			_, err := w.Write([]byte(`{"product_id":"BTC-USD"}`))
			assert.NoError(t, err, "Writing the response should not error")
		case "/currencies":
			_, err := w.Write([]byte(`[{"id":"BTC"}]`))
			assert.NoError(t, err, "Writing the response should not error")
		default:
			assert.Failf(t, "unexpected request", "path %s", r.URL.Path)
		}
	}))
	var product struct {
		ProductID string `json:"product_id"`
	}
	require.NoError(t, ex.SendHTTPRequest(t.Context(), exchange.RestSpot, advancedTradePublicEPL, advancedTradePath+"market/products/BTC-USD", map[string][]string{"a": {"1"}}, &product), "SendHTTPRequest must not error")
	assert.Equal(t, "BTC-USD", product.ProductID, "SendHTTPRequest should decode the response")

	// Saved configurations hold the Exchange API's URL with a trailing slash, which must not double the path's
	endpoint, err := ex.API.Endpoints.GetURL(exchange.RestSpotSupplementary)
	require.NoError(t, err, "GetURL must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpotSupplementary.String(), endpoint+"/"), "SetRunningURL must not error")
	var currencies []struct {
		ID string `json:"id"`
	}
	require.NoError(t, ex.SendHTTPRequest(t.Context(), exchange.RestSpotSupplementary, exchangeAPIPublicEPL, "/currencies", nil, &currencies), "SendHTTPRequest must not error with a trailing slash")
	assert.Len(t, currencies, 1, "SendHTTPRequest should decode the response")

	errorServer := func(status int, body string) *Exchange {
		return newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			_, err := w.Write([]byte(body))
			assert.NoError(t, err, "Writing the response should not error")
		}))
	}
	err = errorServer(http.StatusNotFound, `{"error":"NOT_FOUND","error_details":"Product NOPE-USD not supported","message":"Product NOPE-USD not supported"}`).
		SendHTTPRequest(t.Context(), exchange.RestSpot, advancedTradePublicEPL, advancedTradePath+"market/products/NOPE-USD", nil, &product)
	apiErr, ok := errors.AsType[*APIError](err)
	require.True(t, ok, "SendHTTPRequest must return the error Advanced Trade reports")
	assert.Equal(t, &APIError{ErrorType: "NOT_FOUND", Message: "Product NOPE-USD not supported", ErrorDetails: "Product NOPE-USD not supported"}, apiErr, "SendHTTPRequest should decode the reported error")

	err = errorServer(http.StatusNotFound, `{"errors":[{"id":"not_found","message":"Not found"}]}`).
		SendHTTPRequest(t.Context(), exchange.RestSpot, appEPL, appPath+"currencies/nope", nil, &product)
	appErrs, ok := errors.AsType[AppErrors](err)
	require.True(t, ok, "SendHTTPRequest must return the errors the Coinbase App API reports")
	assert.Equal(t, AppErrors{{ID: "not_found", Message: "Not found"}}, appErrs, "SendHTTPRequest should decode the reported errors")

	err = errorServer(http.StatusNotFound, `{"message":"NotFound"}`).
		SendHTTPRequest(t.Context(), exchange.RestSpotSupplementary, exchangeAPIPublicEPL, "/currencies/NOPE", nil, &product)
	exchangeErr, ok := errors.AsType[*ExchangeAPIError](err)
	require.True(t, ok, "SendHTTPRequest must return the error Coinbase Exchange reports")
	assert.Equal(t, &ExchangeAPIError{Message: "NotFound"}, exchangeErr, "SendHTTPRequest should decode the reported error")

	err = newUnavailableServerExchange(t).SendHTTPRequest(t.Context(), exchange.RestSpot, advancedTradePublicEPL, advancedTradePath+"time", nil, &product)
	assert.ErrorIs(t, err, request.ErrBadStatus, "SendHTTPRequest should return a request failure without a reported error")

	ex = new(Exchange)
	ex.SetDefaults()
	ex.API.Endpoints = ex.NewEndpoints()
	err = ex.SendHTTPRequest(t.Context(), exchange.RestSpot, advancedTradePublicEPL, advancedTradePath+"time", nil, &product)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound, "SendHTTPRequest should error without an endpoint")
}

func TestSendAuthenticatedHTTPRequest(t *testing.T) {
	t.Parallel()
	var host string
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !assert.True(t, ok, "SendAuthenticatedHTTPRequest should send a bearer JWT") {
			return
		}
		jwt := verifyTestJWT(t, auth)
		assert.Equal(t, r.Method+" "+host+r.URL.Path, jwt.Claims["uri"], "SendAuthenticatedHTTPRequest should sign the JWT for the request's method, host and path")
		assert.Equal(t, apiVersion, r.Header.Get("CB-VERSION"), "SendAuthenticatedHTTPRequest should ask for the API version")
		switch r.Method {
		case http.MethodGet:
			assert.Equal(t, "/api/v3/brokerage/accounts", r.URL.Path, "SendAuthenticatedHTTPRequest should request the path")
			assert.Equal(t, "limit=1", r.URL.RawQuery, "SendAuthenticatedHTTPRequest should send the query string")
			assert.Empty(t, r.Header.Get("Content-Type"), "SendAuthenticatedHTTPRequest should not send a content type without a body")
		case http.MethodPost:
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "SendAuthenticatedHTTPRequest should send a body as JSON")
			body, err := io.ReadAll(r.Body)
			if assert.NoError(t, err, "Reading the request body should not error") {
				assert.JSONEq(t, `{"name":"x"}`, string(body), "SendAuthenticatedHTTPRequest should send the body")
			}
		}
		_, err := w.Write([]byte(`{"has_next":false}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	endpoint, err := ex.API.Endpoints.GetURL(exchange.RestSpot)
	require.NoError(t, err, "GetURL must not error")
	host, err = endpointHost(endpoint)
	require.NoError(t, err, "endpointHost must not error")
	var result struct {
		HasNext bool `json:"has_next"`
	}
	require.NoError(t, ex.SendAuthenticatedHTTPRequest(t.Context(), advancedTradePrivateEPL, http.MethodGet, advancedTradePath+"accounts", map[string][]string{"limit": {"1"}}, nil, &result), "SendAuthenticatedHTTPRequest must not error")
	require.NoError(t, ex.SendAuthenticatedHTTPRequest(t.Context(), advancedTradePrivateEPL, http.MethodPost, advancedTradePath+"portfolios", nil, map[string]string{"name": "x"}, &result), "SendAuthenticatedHTTPRequest must not error with a body")
	assert.Error(t, ex.SendAuthenticatedHTTPRequest(t.Context(), advancedTradePrivateEPL, http.MethodPost, advancedTradePath+"portfolios", nil, make(chan int), &result), "SendAuthenticatedHTTPRequest should error for a body that cannot be marshalled")

	err = newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, err := w.Write([]byte(`{"errors":[{"id":"authentication_error","message":"invalid signature"}]}`))
		assert.NoError(t, err, "Writing the response should not error")
	})).SendAuthenticatedHTTPRequest(t.Context(), appEPL, http.MethodGet, appPath+"accounts", nil, nil, &result)
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "SendAuthenticatedHTTPRequest should mark a failed authenticated request")
	_, ok := errors.AsType[AppErrors](err)
	assert.True(t, ok, "SendAuthenticatedHTTPRequest should return the errors the Coinbase App API reports")
}

func TestDecodeResponse(t *testing.T) {
	t.Parallel()
	var result map[string]any
	require.NoError(t, decodeResponse(json.RawMessage(`{"a":1}`), nil, advancedTradeError, &result), "decodeResponse must not error for a successful response")
	assert.Equal(t, map[string]any{"a": float64(1)}, result, "decodeResponse should decode a successful response")
	assert.NoError(t, decodeResponse(json.RawMessage(``), nil, advancedTradeError, &result), "decodeResponse should not error for an empty successful response")
	assert.NoError(t, decodeResponse(json.RawMessage(`{"a":1}`), nil, advancedTradeError, nil), "decodeResponse should not error without a result")
	// A successful response is never read as an error, even with an error member
	assert.NoError(t, decodeResponse(json.RawMessage(`{"error":"NOT_FOUND"}`), nil, advancedTradeError, &result), "decodeResponse should not read a successful response as an error")
	assert.ErrorIs(t, decodeResponse(json.RawMessage(`Unauthorized`), request.ErrBadStatus, advancedTradeError, &result), request.ErrBadStatus, "decodeResponse should return the request error for a body reporting none")
	assert.Error(t, decodeResponse(json.RawMessage(`{`), nil, advancedTradeError, &result), "decodeResponse should error for an undecodable response")
}

func TestErrorStrings(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "NOT_FOUND (5): Product NOPE-USD not supported", (&APIError{ErrorType: "NOT_FOUND", Code: 5, Message: "Product NOPE-USD not supported", ErrorDetails: "Product NOPE-USD not supported"}).Error(), "APIError should not repeat details matching the message")
	assert.Equal(t, "INVALID_ARGUMENT: bad: limit", (&APIError{ErrorType: "INVALID_ARGUMENT", Message: "bad", ErrorDetails: "limit"}).Error(), "APIError should add details that differ from the message")
	assert.Equal(t, "validation_error: a; param_required: b", AppErrors{{ID: "validation_error", Message: "a"}, {ID: "param_required", Message: "b"}}.Error(), "AppErrors should join each error")
	assert.Equal(t, "NotFound", (&ExchangeAPIError{Message: "NotFound"}).Error(), "ExchangeAPIError should return its message")
}

func TestEndpointHost(t *testing.T) {
	t.Parallel()
	host, err := endpointHost("https://api.coinbase.com")
	require.NoError(t, err, "endpointHost must not error")
	assert.Equal(t, "api.coinbase.com", host, "endpointHost should return the host")
	host, err = endpointHost("http://127.0.0.1:8080/prefix")
	require.NoError(t, err, "endpointHost must not error")
	assert.Equal(t, "127.0.0.1:8080/prefix", host, "endpointHost should keep a path prefix")
	_, err = endpointHost("api.coinbase.com")
	assert.ErrorIs(t, err, errEndpointPathInvalid, "endpointHost should error for an endpoint without a scheme")
}

func TestGetJWT(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, setupTestExchange(ex), "setupTestExchange must not error")
	setTestCredentials(ex, testCredentials(t))
	before := time.Now().Truncate(time.Second)
	token, expires, err := ex.GetJWT(t.Context(), "GET api.coinbase.com/api/v3/brokerage/accounts")
	require.NoError(t, err, "GetJWT must not error")
	jwt := verifyTestJWT(t, token)
	assert.Equal(t, testAPIKey, jwt.Header["kid"], "GetJWT should name the key")
	assert.Equal(t, "JWT", jwt.Header["typ"], "GetJWT should set the type")
	assert.Equal(t, "ES256", jwt.Header["alg"], "GetJWT should sign an ECDSA key with ES256")
	assert.Len(t, jwt.Header["nonce"], 16, "GetJWT should send a 16 character nonce")
	assert.Equal(t, "cdp", jwt.Claims["iss"], "GetJWT should set the issuer")
	assert.Equal(t, testAPIKey, jwt.Claims["sub"], "GetJWT should set the subject to the key")
	assert.Equal(t, "GET api.coinbase.com/api/v3/brokerage/accounts", jwt.Claims["uri"], "GetJWT should bind the JWT to the URI")
	nbf, ok := jwt.Claims["nbf"].(float64)
	require.True(t, ok, "nbf must be a number")
	exp, ok := jwt.Claims["exp"].(float64)
	require.True(t, ok, "exp must be a number")
	assert.Equal(t, jwtLifetime.Seconds(), exp-nbf, "GetJWT should expire the JWT two minutes after it starts")
	assert.WithinRange(t, time.Unix(int64(nbf), 0), before, time.Now(), "GetJWT should start the JWT now")
	assert.Equal(t, int64(exp), expires.Unix(), "GetJWT should return the JWT's expiry")

	token, _, err = ex.GetJWT(t.Context(), "")
	require.NoError(t, err, "GetJWT must not error without a URI")
	assert.NotContains(t, verifyTestJWT(t, token).Claims, "uri", "GetJWT should leave the URI out of a websocket JWT")

	ex.SetCredentials(&accounts.Credentials{Key: testAPIKey, Secret: "not a key"})
	_, _, err = ex.GetJWT(t.Context(), "")
	assert.ErrorIs(t, err, errDecodingPrivateKey, "GetJWT should error for a secret that is not a key")
}

func TestParseSigningKey(t *testing.T) {
	t.Parallel()
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err, "GenerateKey must not error")
	sec1, err := x509.MarshalECPrivateKey(ecKey)
	require.NoError(t, err, "MarshalECPrivateKey must not error")
	ecPKCS8, err := x509.MarshalPKCS8PrivateKey(ecKey)
	require.NoError(t, err, "MarshalPKCS8PrivateKey must not error")
	_, edKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err, "GenerateKey must not error")
	edPKCS8, err := x509.MarshalPKCS8PrivateKey(edKey)
	require.NoError(t, err, "MarshalPKCS8PrivateKey must not error")
	toPEM := func(blockType string, der []byte) string {
		return string(pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der}))
	}
	for _, tc := range []struct {
		name    string
		secret  string
		wantAlg string
		wantKey crypto.PrivateKey
	}{
		{name: "SEC1 PEM ECDSA", secret: toPEM("EC PRIVATE KEY", sec1), wantAlg: "ES256", wantKey: ecKey},
		{name: "PKCS#8 PEM ECDSA", secret: toPEM("PRIVATE KEY", ecPKCS8), wantAlg: "ES256", wantKey: ecKey},
		{name: "PKCS#8 PEM Ed25519", secret: toPEM("PRIVATE KEY", edPKCS8), wantAlg: "EdDSA", wantKey: edKey},
		{name: "base64 Ed25519 private key", secret: base64.StdEncoding.EncodeToString(edKey), wantAlg: "EdDSA", wantKey: edKey},
		{name: "base64 Ed25519 seed", secret: base64.StdEncoding.EncodeToString(edKey.Seed()), wantAlg: "EdDSA", wantKey: edKey},
		{name: "not base64 or PEM", secret: "not a key"},
		{name: "base64 of wrong length", secret: base64.StdEncoding.EncodeToString([]byte("short"))},
		{name: "PEM with invalid DER", secret: toPEM("PRIVATE KEY", []byte("garbage"))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			key, alg, err := parseSigningKey(tc.secret)
			if tc.wantKey == nil {
				require.Nil(t, key, "parseSigningKey must not return a key for an invalid secret")
				assert.ErrorIs(t, err, errDecodingPrivateKey, "parseSigningKey should reject an invalid secret")
				return
			}
			require.NoError(t, err, "parseSigningKey must not error")
			assert.Equal(t, tc.wantAlg, alg, "parseSigningKey should return the key's algorithm")
			eq, ok := key.(interface{ Equal(crypto.PrivateKey) bool })
			require.True(t, ok, "parsed key must support Equal")
			assert.True(t, eq.Equal(tc.wantKey), "parsed key should equal the generated key")
		})
	}
}

func TestSignJWT(t *testing.T) {
	t.Parallel()
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err, "GenerateKey must not error")
	sig, err := signJWT(ecKey, "input")
	require.NoError(t, err, "signJWT must not error")
	require.Len(t, sig, 64, "ECDSA signature must be raw R||S")
	hash := sha256.Sum256([]byte("input"))
	s := new(big.Int).SetBytes(sig[32:])
	assert.True(t, ecdsa.Verify(&ecKey.PublicKey, hash[:], new(big.Int).SetBytes(sig[:32]), s), "ECDSA signature should verify against the public key")
	assert.LessOrEqual(t, s.Cmp(new(big.Int).Rsh(ecKey.Params().N, 1)), 0, "ECDSA signature should have a low S value")

	p384Key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err, "GenerateKey must not error")
	_, err = signJWT(p384Key, "input")
	assert.ErrorIs(t, err, errDecodingPrivateKey, "signJWT should reject non-P-256 ECDSA keys")

	pub, edKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err, "GenerateKey must not error")
	sig, err = signJWT(edKey, "input")
	require.NoError(t, err, "signJWT must not error")
	assert.True(t, ed25519.Verify(pub, []byte("input"), sig), "Ed25519 signature should verify against the public key")

	_, err = signJWT("not a key", "input")
	assert.ErrorIs(t, err, common.ErrTypeAssertFailure, "signJWT should reject a value that is not a key")
}

func TestFormatHelpers(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "0.00000001", formatFloat(1e-8), "formatFloat should not use exponent notation")
	assert.Equal(t, "150000", formatFloat(1.5e5), "formatFloat should not use exponent notation")
	assert.Equal(t, "2026-10-19T01:02:03Z", formatTime(time.Date(2026, 10, 19, 12, 2, 3, 0, time.FixedZone("AEDT", 11*60*60))), "formatTime should format in UTC")
	assert.NoError(t, checkWindow(time.Time{}, time.Now()), "checkWindow should allow an unset start")
	assert.NoError(t, checkWindow(time.Now(), time.Time{}), "checkWindow should allow an unset end")
	assert.ErrorIs(t, checkWindow(time.Unix(2, 0), time.Unix(1, 0)), common.ErrStartAfterEnd, "checkWindow should reject a reversed window")
}

func TestAppErrorPublicShape(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, err := w.Write([]byte(`{"error":"rpc error: code = NotFound desc = not found","code":5,"message":"rpc error: code = NotFound desc = not found"}`))
		assert.NoError(t, err, "writing the response should not error")
	}))
	_, err := ex.App().GetSpotPrice(t.Context(), &AppSpotPriceRequest{Pair: currency.NewPair(currency.NewCode("NOTACOINXYZ"), currency.USD)})
	require.Error(t, err, "GetSpotPrice must error")
	apiErr, ok := errors.AsType[*APIError](err)
	require.True(t, ok, "GetSpotPrice's error must be an APIError")
	assert.EqualValues(t, 5, apiErr.Code, "Code should be NotFound's")
}

func TestNullReplies(t *testing.T) {
	t.Parallel()
	ex := newResponseServerExchange(t, `null`)
	candles := &ProductCandlesRequest{ProductID: testSpotProductID, Start: testCandlesStart, End: testCandlesStart.Add(time.Hour), Granularity: GranularityOneHour}
	for name, call := range map[string]func() error{
		"ListFuturesPositions":    func() error { _, err := ex.ListFuturesPositions(t.Context()); return err },
		"ListFuturesSweeps":       func() error { _, err := ex.ListFuturesSweeps(t.Context()); return err },
		"ListPaymentMethods":      func() error { _, err := ex.ListPaymentMethods(t.Context()); return err },
		"ListPortfolios":          func() error { _, err := ex.ListPortfolios(t.Context(), ""); return err },
		"GetBestBidAsk":           func() error { _, err := ex.GetBestBidAsk(t.Context(), []string{testSpotProductID}); return err },
		"GetProductCandles":       func() error { _, err := ex.GetProductCandles(t.Context(), candles); return err },
		"GetPublicProductCandles": func() error { _, err := ex.GetPublicProductCandles(t.Context(), candles); return err },
	} {
		assert.ErrorIsf(t, call(), common.ErrNoResponse, "%s should error for a null response", name)
	}
}
