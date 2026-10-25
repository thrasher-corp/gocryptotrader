package lbank

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/md5" //nolint:gosec // LBank's signature scheme signs an MD5 digest of the request parameters
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testutils "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
)

const (
	// canManipulateRealOrders lets live tests with credentials change the account they act for: place and cancel orders
	// priced away from the market, and withdraw
	canManipulateRealOrders = false
	// testAPIKey and testAPISecret are placeholder credentials, which the mock responses and the test servers accept
	testAPIKey    = "mock-api-key"
	testAPISecret = "mock-api-secret"
	// testServerRequestLimit is the most requests a test server answers, several times as many as any test sends one,
	// so a pager no longer advancing fails its test rather than hangs it
	testServerRequestLimit = 100
)

var (
	e               *Exchange
	spotTestPair    = currency.NewBTCUSDT().Format(currency.PairFormat{Delimiter: currency.UnderscoreDelimiter})
	futuresTestPair = currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter})
)

// testExchangeConfig is LBank's configuration from testdata/configtest.json, loaded once; testexch.Setup loads every
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
	exchCfg, err := cfg.GetExchangeConfig("Lbank")
	if err != nil {
		return nil, err
	}
	return json.Marshal(exchCfg)
})

// setupTestExchange sets an exchange up as testexch.Setup does, from its own copy of LBank's test configuration, with
// the test pairs available and enabled
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
	return storeTestPairs(ex)
}

// setTestCredentials enables authenticated support with the given credentials
func setTestCredentials(ex *Exchange, creds *accounts.Credentials) {
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(creds)
}

// storeTestPairs makes the test pairs available and enabled
func storeTestPairs(ex *Exchange) error {
	for a, p := range map[asset.Item]currency.Pair{asset.Spot: spotTestPair, asset.USDTMarginedFutures: futuresTestPair} {
		available, err := ex.CurrencyPairs.GetPairs(a, false)
		if err != nil {
			return err
		}
		if !available.Contains(p, true) {
			if err := ex.CurrencyPairs.StorePairs(a, append(available, p), false); err != nil {
				return err
			}
		}
		enabled, err := ex.CurrencyPairs.GetPairs(a, true)
		if err != nil {
			return err
		}
		if !enabled.Contains(p, true) {
			if err := ex.CurrencyPairs.StorePairs(a, append(enabled, p), true); err != nil {
				return err
			}
		}
	}
	return nil
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
	for _, ep := range []exchange.URL{exchange.RestSpot, exchange.RestUSDTMargined} {
		require.NoErrorf(t, ex.API.Endpoints.SetRunningURL(ep.String(), server.URL), "SetRunningURL must not error for %s", ep)
	}
	return ex
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

// writeResponse writes a reply in a test server's handler
func writeResponse(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	_, err := fmt.Fprint(w, body)
	assert.NoError(t, err, "writing the response should not error")
}

// hmacSignature independently computes LBank's HmacSHA256 signature of a request's parameters
func hmacSignature(params url.Values, secret string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + "=" + params.Get(k)
	}
	digest := md5.Sum([]byte(strings.Join(parts, "&"))) //nolint:gosec // LBank's signature scheme signs an MD5 digest of the request parameters
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strings.ToUpper(hex.EncodeToString(digest[:]))))
	return hex.EncodeToString(mac.Sum(nil))
}

// pkcs8Secret returns a private key as the bare base64 PKCS #8 DER LBank shows as an RSA secret
func pkcs8Secret(t *testing.T, key any) string {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err, "MarshalPKCS8PrivateKey must not error")
	return base64.StdEncoding.EncodeToString(der)
}

func TestSendHTTPRequest(t *testing.T) {
	t.Parallel()
	var requested []string
	var mu sync.Mutex
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requested = append(requested, r.Method+" "+r.URL.RequestURI())
		mu.Unlock()
		switch r.URL.Path {
		case "/envelope":
			writeResponse(t, w, `{"result":"true","data":{"symbol":"btc_usdt","price":"1.5"},"error_code":0,"ts":1}`)
		case "/failed":
			writeResponse(t, w, `{"result":"false","msg":"Invalid parameter","error_code":10003,"ts":1}`)
		case "/bare":
			writeResponse(t, w, `[{"symbol":"btc_usdt","price":"2.5"}]`)
		case "/malformed":
			writeResponse(t, w, `{"result":true,"error_code":{}}`)
		default:
			http.NotFound(w, r)
		}
	}))

	params := url.Values{}
	params.Set("symbol", "btc_usdt")
	var result *LatestPriceResponse
	require.NoError(t, ex.SendHTTPRequest(t.Context(), exchange.RestSpot, defaultEPL, http.MethodGet, "/envelope", params, &result), "SendHTTPRequest must not error")
	assert.Equal(t, &LatestPriceResponse{Symbol: "btc_usdt", Price: 1.5}, result, "SendHTTPRequest should decode the data the envelope carries")

	var bare []LatestPriceResponse
	require.NoError(t, ex.SendHTTPRequest(t.Context(), exchange.RestSpot, defaultEPL, http.MethodGet, "/bare", nil, &bare), "SendHTTPRequest must not error")
	assert.Equal(t, []LatestPriceResponse{{Symbol: "btc_usdt", Price: 2.5}}, bare, "SendHTTPRequest should decode a reply without an envelope")

	err := ex.SendHTTPRequest(t.Context(), exchange.RestSpot, defaultEPL, http.MethodGet, "/failed", nil, &bare)
	require.ErrorIs(t, err, errRequestFailed, "SendHTTPRequest must report a failed reply")
	assert.ErrorContains(t, err, ex.Name+": request failed: Invalid parameter (error_code 10003)", "SendHTTPRequest should report the exchange, message and code")
	assert.NotErrorIs(t, err, request.ErrAuthRequestFailed, "SendHTTPRequest should not report an unauthenticated failure as an authentication failure")

	err = ex.SendHTTPRequest(t.Context(), exchange.RestSpot, defaultEPL, http.MethodGet, "/malformed", nil, &bare)
	assert.ErrorContains(t, err, "decoding response envelope", "SendHTTPRequest should report an envelope it cannot decode")

	err = ex.SendHTTPRequest(t.Context(), exchange.URL(255), defaultEPL, http.MethodGet, "/envelope", nil, &bare)
	assert.Error(t, err, "SendHTTPRequest should error for an endpoint the exchange has no URL for")

	assert.Equal(t, []string{"GET /envelope?symbol=btc_usdt", "GET /bare", "GET /failed", "GET /malformed"}, requested, "SendHTTPRequest should send each request once to its path")
}

func TestSendAuthHTTPRequest(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var timestamps []int64
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assert.NoError(t, r.ParseForm(), "the request body should parse as a form") {
			return
		}
		assert.Equal(t, http.MethodPost, r.Method, "a signed request should be a POST")
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"), "a signed request should send a form")
		assert.Equal(t, signatureMethodHMAC, r.Header.Get("signature_method"), "a signed request should name its signature method")
		echostr := r.Header.Get("echostr")
		assert.Len(t, echostr, echostrLength, "a signed request should send an echostr of LBank's length")
		assert.Equal(t, -1, strings.IndexFunc(echostr, func(c rune) bool {
			return (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9')
		}), "the echostr should hold only letters and digits")
		ts, err := strconv.ParseInt(r.Header.Get("timestamp"), 10, 64)
		assert.NoError(t, err, "the timestamp header should be Unix milliseconds")
		mu.Lock()
		timestamps = append(timestamps, ts)
		mu.Unlock()
		for _, k := range []string{"timestamp", "echostr", "signature_method"} {
			assert.NotContainsf(t, r.PostForm, k, "the body should not carry %s, which is sent as a header", k)
		}
		signed := url.Values{}
		for k, v := range r.PostForm {
			if k != "sign" {
				signed[k] = v
			}
		}
		signed.Set("timestamp", r.Header.Get("timestamp"))
		signed.Set("echostr", echostr)
		signed.Set("signature_method", signatureMethodHMAC)
		assert.Equal(t, hmacSignature(signed, testAPISecret), r.PostForm.Get("sign"), "the signature should cover the raw parameters and the signing headers")
		exp := url.Values{"api_key": {testAPIKey}, "symbol": {"a b&c"}, "sign": {r.PostForm.Get("sign")}}
		assert.Equal(t, exp, r.PostForm, "the body should carry the parameters, the key and the signature")
		switch r.URL.Path {
		case "/ok":
			writeResponse(t, w, `{"result":true,"data":{"symbol":"btc_usdt","price":"3"},"error_code":0,"ts":1}`)
		case "/refused":
			writeResponse(t, w, `{"result":"false","msg":"Invalid signature","error_code":10007,"ts":1}`)
		case "/undecodable":
			writeResponse(t, w, `{"result":true,"data":{"symbol":7},"error_code":0,"ts":1}`)
		}
	}))
	params := url.Values{}
	params.Set("symbol", "a b&c")
	called := time.Now().UnixMilli()
	var result *LatestPriceResponse
	require.NoError(t, ex.SendAuthHTTPRequest(t.Context(), defaultEPL, "/ok", params, &result), "SendAuthHTTPRequest must not error")
	returned := time.Now().UnixMilli()
	assert.Equal(t, &LatestPriceResponse{Symbol: "btc_usdt", Price: 3}, result, "SendAuthHTTPRequest should decode the data the reply carries")
	assert.Equal(t, url.Values{"symbol": {"a b&c"}}, params, "SendAuthHTTPRequest should not change the caller's parameters")
	require.Len(t, timestamps, 1, "SendAuthHTTPRequest must send one request")
	assert.True(t, timestamps[0] >= called && timestamps[0] <= returned, "the timestamp should be the time of the request")

	err := ex.SendAuthHTTPRequest(t.Context(), defaultEPL, "/refused", params, &result)
	require.ErrorIs(t, err, request.ErrAuthRequestFailed, "SendAuthHTTPRequest must report a refused request as an authentication failure")
	assert.ErrorIs(t, err, errRequestFailed, "SendAuthHTTPRequest should report the reply's failure")

	err = ex.SendAuthHTTPRequest(t.Context(), defaultEPL, "/undecodable", params, &result)
	require.Error(t, err, "SendAuthHTTPRequest must report a reply it cannot decode")
	assert.NotErrorIs(t, err, request.ErrAuthRequestFailed, "SendAuthHTTPRequest should not report a decoding failure as an authentication failure")

	noCreds := newTestServerExchange(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		assert.Fail(t, "a request without credentials should not be sent")
	}))
	noCreds.SkipAuthCheck = false
	noCreds.SetCredentials(&accounts.Credentials{})
	assert.ErrorIs(t, noCreds.SendAuthHTTPRequest(t.Context(), defaultEPL, "/ok", nil, nil), exchange.ErrCredentialsAreEmpty, "SendAuthHTTPRequest should require credentials")
}

func TestSendAuthHTTPRequestRSA(t *testing.T) {
	t.Parallel()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "GenerateKey must not error")
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assert.NoError(t, r.ParseForm(), "the request body should parse as a form") {
			return
		}
		assert.Equal(t, signatureMethodRSA, r.Header.Get("signature_method"), "an RSA signed request should name its signature method")
		signed := url.Values{"api_key": {testAPIKey}, "symbol": {"btc_usdt"}}
		signed.Set("timestamp", r.Header.Get("timestamp"))
		signed.Set("echostr", r.Header.Get("echostr"))
		signed.Set("signature_method", signatureMethodRSA)
		signature, err := base64.StdEncoding.DecodeString(r.PostForm.Get("sign"))
		if assert.NoError(t, err, "the RSA signature should be base64") {
			digest := md5.Sum([]byte(parameterString(signed))) //nolint:gosec // LBank's signature scheme signs an MD5 digest of the request parameters
			hashed := sha256.Sum256([]byte(strings.ToUpper(hex.EncodeToString(digest[:]))))
			assert.NoError(t, rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, hashed[:], signature), "the RSA signature should verify against the key")
		}
		writeResponse(t, w, `{"result":"true","data":{"status":"1"},"error_code":0,"ts":1}`)
	}))
	ex.SetCredentials(&accounts.Credentials{Key: testAPIKey, Secret: pkcs8Secret(t, key)})
	params := url.Values{}
	params.Set("symbol", "btc_usdt")
	var result *SystemStatusResponse
	require.NoError(t, ex.SendAuthHTTPRequest(t.Context(), defaultEPL, "/rsa", params, &result), "SendAuthHTTPRequest must not error")
	assert.Equal(t, &SystemStatusResponse{Status: "1"}, result, "SendAuthHTTPRequest should decode the reply")
}

func TestSignRequest(t *testing.T) {
	t.Parallel()
	signer := &requestSigner{method: signatureMethodHMAC, secret: []byte(testAPISecret)}
	params := url.Values{"symbol": {"btc_usdt"}, "custom_id": {"a&b c"}}
	now := time.UnixMilli(1792887560015)
	body, headers, err := signRequest(signer, &accounts.Credentials{Key: testAPIKey, Secret: testAPISecret}, params, now)
	require.NoError(t, err, "signRequest must not error")
	require.Len(t, headers["echostr"], echostrLength, "signRequest must send an echostr of LBank's length")
	signed := url.Values{"api_key": {testAPIKey}, "symbol": {"btc_usdt"}, "custom_id": {"a&b c"}, "timestamp": {"1792887560015"}, "echostr": {headers["echostr"]}, "signature_method": {signatureMethodHMAC}}
	exp := url.Values{"api_key": {testAPIKey}, "symbol": {"btc_usdt"}, "custom_id": {"a&b c"}, "sign": {hmacSignature(signed, testAPISecret)}}
	assert.Equal(t, exp.Encode(), body, "signRequest should send the parameters, the key and the signature of the raw values")
	assert.Equal(t, map[string]string{
		"Content-Type":     "application/x-www-form-urlencoded",
		"timestamp":        "1792887560015",
		"signature_method": signatureMethodHMAC,
		"echostr":          headers["echostr"],
	}, headers, "signRequest should send the signing values as headers")
	assert.Equal(t, url.Values{"symbol": {"btc_usdt"}, "custom_id": {"a&b c"}}, params, "signRequest should not change the caller's parameters")
}

func TestParameterString(t *testing.T) {
	t.Parallel()
	params := url.Values{"timestamp": {"1"}, "api_key": {"k"}, "echostr": {"e"}, "amount": {"0.1"}, "custom_id": {"a&b=c d"}}
	assert.Equal(t, "amount=0.1&api_key=k&custom_id=a&b=c d&echostr=e&timestamp=1", parameterString(params), "parameterString should sort the parameters and keep their raw values")
	assert.Empty(t, parameterString(url.Values{}), "parameterString should return nothing for no parameters")
}

func TestRequestSignerSign(t *testing.T) {
	t.Parallel()
	// The test account and the signature LBank's contract docs give for their example request
	signer := &requestSigner{method: signatureMethodHMAC, secret: []byte("093F44F700FC48F17DDB67390C895CE5")}
	sign, err := signer.sign("api_key=fb4e39e5-6a06-4291-9f80-d10176a0badd&asset=USDT&echostr=echostr123456789012345678901234567890&productGroup=SwapU&signature_method=HmacSHA256&timestamp=1665990154559")
	require.NoError(t, err, "sign must not error")
	assert.Equal(t, "809133cb69a17beba0be076b99b4d90de872476e36da87978ab2889970ccd06d", sign, "sign should reproduce LBank's documented HmacSHA256 signature")

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "GenerateKey must not error")
	rsaSigner := &requestSigner{method: signatureMethodRSA, rsaKey: key}
	sign, err = rsaSigner.sign("hello123")
	require.NoError(t, err, "sign must not error")
	signature, err := base64.StdEncoding.DecodeString(sign)
	require.NoError(t, err, "the RSA signature must be base64")
	digest := md5.Sum([]byte("hello123")) //nolint:gosec // LBank's signature scheme signs an MD5 digest of the request parameters
	hashed := sha256.Sum256([]byte(strings.ToUpper(hex.EncodeToString(digest[:]))))
	assert.NoError(t, rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, hashed[:], signature), "sign should sign the upper case MD5 digest with SHA256withRSA")
}

func TestGetSigner(t *testing.T) {
	t.Parallel()
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "GenerateKey must not error")
	secret := pkcs8Secret(t, rsaKey)
	der, err := x509.MarshalPKCS8PrivateKey(rsaKey)
	require.NoError(t, err, "MarshalPKCS8PrivateKey must not error")
	pemSecret := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err, "GenerateKey must not error")
	pkcs1 := base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PrivateKey(rsaKey))

	ex := new(Exchange)
	hmacSigner, err := ex.getSigner("093F44F700FC48F17DDB67390C895CE5")
	require.NoError(t, err, "getSigner must not error for a hexadecimal secret")
	assert.Equal(t, &requestSigner{method: signatureMethodHMAC, secret: []byte("093F44F700FC48F17DDB67390C895CE5")}, hmacSigner, "getSigner should sign a hexadecimal secret with HmacSHA256")

	for name, s := range map[string]string{"base64 PKCS #8 key": secret, "PEM PKCS #8 key": pemSecret} {
		signer, err := ex.getSigner(s)
		require.NoErrorf(t, err, "getSigner must not error for a %s", name)
		assert.Equalf(t, signatureMethodRSA, signer.method, "getSigner should sign a %s with RSA", name)
		assert.Truef(t, rsaKey.Equal(signer.rsaKey), "getSigner should parse the %s", name)
	}
	cached, ok := ex.rsaKeys.Load(secret)
	require.True(t, ok, "getSigner must cache a parsed RSA key")
	again, err := ex.getSigner(secret)
	require.NoError(t, err, "getSigner must not error for a cached key")
	assert.Same(t, cached, again.rsaKey, "getSigner should reuse the key it cached")

	for _, tc := range []struct {
		name   string
		secret string
		err    error
	}{
		{name: "not base64", secret: "MI!!not base64", err: errPEMBlockIsNil},
		{name: "PKCS #1 key", secret: pkcs1, err: errUnableToParsePrivateKey},
		{name: "ECDSA key", secret: pkcs8Secret(t, ecdsaKey), err: common.ErrTypeAssertFailure},
	} {
		_, err := ex.getSigner(tc.secret)
		assert.ErrorIsf(t, err, tc.err, "getSigner should reject a %s", tc.name)
	}
}

func TestUnwrapV2Response(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		payload  string
		expected string
		err      string
		code     int64
	}{
		{name: "string result true", payload: `{"result":"true","data":[1,2,3]}`, expected: `[1,2,3]`},
		{name: "boolean result true", payload: `{"result":true,"data":[1,2,3]}`, expected: `[1,2,3]`},
		{name: "string result false", payload: `{"result":"false","msg":"Invalid parameter","error_code":10003}`, err: "request failed: Invalid parameter (error_code 10003)", code: 10003},
		{name: "boolean result false", payload: `{"result":false,"msg":"Invalid parameter","error_code":10003}`, err: "request failed: Invalid parameter (error_code 10003)", code: 10003},
		{name: "failed envelope carrying data is still a failure", payload: `{"result":false,"msg":"Invalid parameter","error_code":10003,"data":[1,2,3]}`, err: "request failed: Invalid parameter (error_code 10003)", code: 10003},
		{name: "failed envelope without error code", payload: `{"result":false,"msg":"instrument not found"}`, err: "request failed: instrument not found (error_code 0)"},
		{name: "quoted error code", payload: `{"result":"false","msg":"Illegal parameter","error_code":"10005"}`, err: "request failed: Illegal parameter (error_code 10005)", code: 10005},
		{name: "negative error code", payload: `{"result":"false","msg":"System exception","error_code":-99}`, err: "request failed: System exception (error_code -99)", code: -99},
		{name: "absent result is not a failure", payload: `{"data":[1,2,3]}`, expected: `[1,2,3]`},
		{name: "null result is not a failure", payload: `{"result":null,"data":[1,2,3]}`, expected: `[1,2,3]`},
		{name: "envelope without data returns the payload", payload: `{"result":"true"}`, expected: `{"result":"true"}`},
		{name: "non envelope payload is unchanged", payload: `[{"symbol":"btc_usdt"}]`, expected: `[{"symbol":"btc_usdt"}]`},
		{name: "non json payload is unchanged", payload: `not json at all`, expected: `not json at all`},
		{name: "object with a reshaped field is an error", payload: `{"result":true,"error_code":{}}`, err: "decoding response envelope:"},
		{name: "object with leading whitespace and a reshaped field is an error", payload: " \n\t{\"result\":true,\"error_code\":{}}", err: "decoding response envelope:"},
		{name: "truncated object is an error", payload: `{"result":true,oops}`, err: "decoding response envelope:"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			payload, err := unwrapV2Response([]byte(tc.payload))
			if tc.err == "" {
				require.NoError(t, err, "unwrapV2Response must not error")
				assert.Equal(t, tc.expected, string(payload), "unwrapV2Response should return the expected payload")
				return
			}
			require.ErrorContains(t, err, tc.err, "unwrapV2Response must report the expected error")
			if !strings.HasPrefix(tc.err, "request failed") {
				assert.NotErrorIs(t, err, errRequestFailed, "a malformed envelope should not be reported as a failed request")
				return
			}
			var reqErr *requestError
			require.ErrorAs(t, err, &reqErr, "unwrapV2Response must report a failed request as a requestError")
			assert.Equal(t, tc.code, reqErr.code, "unwrapV2Response should report LBank's error code")
			assert.ErrorIs(t, err, errRequestFailed, "unwrapV2Response should report a matchable request failure")
		})
	}
}

func TestDecodeResponse(t *testing.T) {
	t.Parallel()
	require.NoError(t, decodeResponse([]byte(`{"result":"true"}`), nil), "decodeResponse must accept a nil result")
	var resp *SystemStatusResponse
	require.NoError(t, decodeResponse([]byte(`{"result":"true","data":{"status":"0"}}`), &resp), "decodeResponse must not error")
	assert.Equal(t, &SystemStatusResponse{Status: "0"}, resp, "decodeResponse should decode the reply's data")
	assert.Error(t, decodeResponse([]byte(`[{]`), &resp), "decodeResponse should report a payload it cannot decode")
}
