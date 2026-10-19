package coinbase

import (
	"bytes"
	"context"
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
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Exchange implements exchange.IBotExchange and holds the methods of Coinbase's Advanced Trade API, for spot and US
// derivatives; the Coinbase App API is reached through App, and the Global Derivatives gateway through
// GlobalDerivatives over HTTP and WebsocketAPI over the websocket
type Exchange struct {
	exchange.Base
	// rateLimits are the limiters the requester and the websocket share, so the gateway's credits are counted once
	// across HTTP and the websocket
	rateLimits       request.RateLimitDefinitions
	pairAliases      pairAliases
	derivativesToken derivativesAccessToken
	wsConnections    websocketConnections
	// orderSizes holds the sizes of the account's live Advanced Trade orders, which the user channel leaves out
	orderSizes advancedTradeOrderSizes
	// tickerMergeMu holds a ticker's merge with the stored one and its storing together
	tickerMergeMu sync.Mutex
	// derivativesIncrementalTickers holds each instrument's incremental ticker so far, which a change is merged onto
	derivativesIncrementalTickers   map[string]*DerivativesWebsocketIncrementalTicker
	derivativesIncrementalTickersMu sync.Mutex
	// derivativesSessions holds which of the gateway's private connections are authenticated
	derivativesSessions derivativesWebsocketSessions
	// perpetualFutures holds the product IDs of the US derivatives paying funding, which product data loads
	perpetualFutures   map[string]struct{}
	perpetualFuturesMu sync.RWMutex
}

const (
	// apiURL serves the Advanced Trade API under /api/v3/brokerage and the Coinbase App API under /v2
	apiURL = "https://api.coinbase.com"
	// sandboxAPIURL serves static responses for the Advanced Trade accounts, orders and portfolios endpoints, without
	// authentication
	sandboxAPIURL = "https://api-sandbox.coinbase.com"
	// exchangeAPIURL is Coinbase Exchange's API, whose currency endpoints alone list each currency's networks; saved
	// configurations hold it with a trailing slash
	exchangeAPIURL = "https://api.exchange.coinbase.com/"
	// derivativesAPIURL is the Global Derivatives gateway, which serves its JSON-RPC methods under /api/v2
	derivativesAPIURL = "https://drb.coinbase.com"

	// advancedTradePath prefixes every Advanced Trade endpoint
	advancedTradePath = "/api/v3/brokerage/"
	// appPath prefixes every Coinbase App endpoint
	appPath = "/v2/"

	// apiVersion is the Coinbase App API version requests ask for: the docs ask for the date an implementation was
	// written against, so the API's behaviour cannot change under it
	apiVersion = "2026-10-19"
	// jwtLifetime is how long a JWT is valid for; Coinbase refuses one that expires more than two minutes after it
	// starts
	jwtLifetime = 2 * time.Minute
)

var (
	errProductIDRequired   = errors.New("product ID required")
	errAccountIDRequired   = errors.New("account ID required")
	errPortfolioIDRequired = errors.New("portfolio ID required")
	errInvalidAmount       = errors.New("amount must be greater than zero")
	errInvalidPrice        = errors.New("price must be greater than zero")
	errInvalidLimit        = errors.New("invalid limit")
	errDecodingPrivateKey  = errors.New("error decoding private key")
	errEndpointPathInvalid = errors.New("endpoint must be an absolute URL")
)

// APIError is an error the Advanced Trade API reports in a response body
type APIError struct {
	// ErrorType names the failure, such as NOT_FOUND, INVALID_ARGUMENT or unknown
	ErrorType    string `json:"error"`
	Code         int64  `json:"code"`
	Message      string `json:"message"`
	ErrorDetails string `json:"error_details"`
	// Details holds the failure's google.rpc details, which Coinbase rarely sends
	Details []json.RawMessage `json:"details"`
}

// Error returns the error's type and message, followed by its details when they add to the message
func (a *APIError) Error() string {
	msg := a.ErrorType
	if a.Code != 0 {
		msg += " (" + strconv.FormatInt(a.Code, 10) + ")"
	}
	if a.Message != "" {
		msg += ": " + a.Message
	}
	if a.ErrorDetails != "" && a.ErrorDetails != a.Message {
		msg += ": " + a.ErrorDetails
	}
	return msg
}

// AppError is an error the Coinbase App API reports, an id naming the failure and a message describing it
type AppError struct {
	ID      string `json:"id"`
	Message string `json:"message"`
	URL     string `json:"url"`
}

// AppErrors are the errors the Coinbase App API reports in a response body; only validation errors report several
type AppErrors []AppError

// Error returns each error's id and message
func (a AppErrors) Error() string {
	msgs := make([]string, len(a))
	for i := range a {
		msgs[i] = a[i].ID + ": " + a[i].Message
	}
	return strings.Join(msgs, "; ")
}

// ExchangeAPIError is an error Coinbase Exchange's API reports in a response body
type ExchangeAPIError struct {
	Message string `json:"message"`
}

// Error returns the error's message
func (e *ExchangeAPIError) Error() string {
	return e.Message
}

// decodesAs reports whether a body decodes into v; a body that is not JSON, such as the plain text Coinbase answers an
// unauthenticated request with, reports no error of its own
func decodesAs(body []byte, v any) bool {
	return json.Unmarshal(body, v) == nil
}

// advancedTradeError returns the error an Advanced Trade response body reports, or nil when it reports none
func advancedTradeError(body []byte) error {
	var apiErr APIError
	if !decodesAs(body, &apiErr) || (apiErr.ErrorType == "" && apiErr.Message == "") {
		return nil
	}
	return &apiErr
}

// appError returns the errors a Coinbase App response body reports, or nil when it reports none
func appError(body []byte) error {
	var resp struct {
		Errors AppErrors `json:"errors"`
	}
	if !decodesAs(body, &resp) || len(resp.Errors) == 0 {
		return nil
	}
	return resp.Errors
}

// exchangeAPIError returns the error a Coinbase Exchange response body reports, or nil when it reports none
func exchangeAPIError(body []byte) error {
	var apiErr ExchangeAPIError
	if !decodesAs(body, &apiErr) || apiErr.Message == "" {
		return nil
	}
	return &apiErr
}

// errorParser returns the error an API reports in a response body, or nil when it reports none
type errorParser func(body []byte) error

// errorParserForPath returns the error parser of the API a path belongs to
func errorParserForPath(path string) errorParser {
	if strings.HasPrefix(path, appPath) {
		return appError
	}
	return advancedTradeError
}

// decodeResponse returns the error a failed request's body reports, else the request's error, else decodes the body
// into result. A response body is only checked for an error when the request failed, as Coinbase reports failures
// with an error status
func decodeResponse(body json.RawMessage, requestErr error, parse errorParser, result any) error {
	if requestErr != nil {
		apiErr := parse(body)
		if apiErr == nil {
			return requestErr
		}
		// The request system marks every failed authenticated request, which callers tell auth failures by
		if errors.Is(requestErr, request.ErrAuthRequestFailed) {
			return fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, apiErr)
		}
		return apiErr
	}
	if result == nil || len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	return json.Unmarshal(body, result)
}

// endpointURL returns an endpoint's URL without the trailing slash saved configurations may hold
func (e *Exchange) endpointURL(ep exchange.URL) (string, error) {
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(endpoint, "/"), nil
}

// endpointHost returns the host a JWT's URI names for an endpoint
func endpointHost(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("%w: %q", errEndpointPathInvalid, endpoint)
	}
	return u.Host + u.Path, nil
}

// SendHTTPRequest sends an unauthenticated GET request to an API's endpoint; path starts with the API's version path
// and params, which may be nil, are sent as the query string
func (e *Exchange) SendHTTPRequest(ctx context.Context, ep exchange.URL, epl request.EndpointLimit, path string, params url.Values, result any) error {
	endpoint, err := e.endpointURL(ep)
	if err != nil {
		return err
	}
	parse := errorParserForPath(path)
	if ep == exchange.RestSpotSupplementary {
		parse = exchangeAPIError
	}
	var response json.RawMessage
	err = e.SendPayload(ctx, epl, func() (*request.Item, error) {
		return &request.Item{
			Method:                 http.MethodGet,
			Path:                   endpoint + common.EncodeURLValues(path, params),
			Result:                 &response,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.UnauthenticatedRequest)
	return decodeResponse(response, err, parse, result)
}

// SendAuthenticatedHTTPRequest sends a request to the Advanced Trade or Coinbase App API with a JWT signed for it;
// path starts with the API's version path, params, which may be nil, are sent as the query string, and a non-nil body
// is sent as JSON
func (e *Exchange) SendAuthenticatedHTTPRequest(ctx context.Context, epl request.EndpointLimit, method, path string, params url.Values, body, result any) error {
	endpoint, err := e.endpointURL(exchange.RestSpot)
	if err != nil {
		return err
	}
	host, err := endpointHost(endpoint)
	if err != nil {
		return err
	}
	var payload []byte
	if body != nil {
		if payload, err = json.Marshal(body); err != nil {
			return err
		}
	}
	var response json.RawMessage
	err = e.SendPayload(ctx, epl, func() (*request.Item, error) {
		// A JWT is signed for one request's method, host and path, and expires two minutes after it is made, so each
		// attempt signs a fresh one
		jwt, _, err := e.GetJWT(ctx, method+" "+host+path)
		if err != nil {
			return nil, err
		}
		item := &request.Item{
			Method: method,
			Path:   endpoint + common.EncodeURLValues(path, params),
			Headers: map[string]string{
				"Authorization": "Bearer " + jwt,
				"CB-VERSION":    apiVersion,
			},
			Result:                 &response,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}
		if payload != nil {
			item.Headers["Content-Type"] = "application/json"
			item.Body = bytes.NewReader(payload)
		}
		return item, nil
	}, request.AuthenticatedRequest)
	return decodeResponse(response, err, errorParserForPath(path), result)
}

// GetJWT returns a JWT signed with the configured CDP API key, and the time it expires. uri, the request's method,
// host and path such as "GET api.coinbase.com/api/v3/brokerage/accounts", binds the JWT to one REST request; the
// websocket takes JWTs without one
func (e *Exchange) GetJWT(ctx context.Context, uri string) (string, time.Time, error) {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return "", time.Time{}, err
	}
	privateKey, alg, err := parseSigningKey(creds.Secret)
	if err != nil {
		return "", time.Time{}, err
	}
	nonce, err := common.GenerateRandomString(16, "1234567890ABCDEF")
	if err != nil {
		return "", time.Time{}, err
	}
	headJSON, err := json.Marshal(map[string]any{
		"kid":   creds.Key,
		"typ":   "JWT",
		"alg":   alg,
		"nonce": nonce,
	})
	if err != nil {
		return "", time.Time{}, err
	}
	now := time.Now()
	expires := now.Add(jwtLifetime)
	claims := map[string]any{
		"iss": "cdp",
		"nbf": now.Unix(),
		"exp": expires.Unix(),
		"sub": creds.Key,
	}
	if uri != "" {
		claims["uri"] = uri
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, err
	}
	signingInput := base64.RawURLEncoding.EncodeToString(headJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	sig, err := signJWT(privateKey, signingInput)
	if err != nil {
		return "", time.Time{}, err
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), expires, nil
}

// parseSigningKey decodes a CDP API secret and returns the private key with the matching JWT algorithm. ECDSA keys
// arrive as SEC1 or PKCS#8 PEM, Ed25519 keys as PKCS#8 PEM or base64, either the 64 byte private key or the 32 byte
// seed
func parseSigningKey(secret string) (crypto.PrivateKey, string, error) {
	block, _ := pem.Decode([]byte(secret))
	if block == nil {
		raw, err := base64.StdEncoding.DecodeString(secret)
		if err != nil {
			return nil, "", fmt.Errorf("%w: %w", errDecodingPrivateKey, err)
		}
		switch len(raw) {
		case ed25519.SeedSize:
			return ed25519.NewKeyFromSeed(raw), "EdDSA", nil
		case ed25519.PrivateKeySize:
			return ed25519.PrivateKey(raw), "EdDSA", nil
		}
		return nil, "", fmt.Errorf("%w: %d bytes is not an Ed25519 key or seed", errDecodingPrivateKey, len(raw))
	}
	if ecKey, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return ecKey, "ES256", nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %w", errDecodingPrivateKey, err)
	}
	switch k := parsed.(type) {
	case *ecdsa.PrivateKey:
		return k, "ES256", nil
	case ed25519.PrivateKey:
		return k, "EdDSA", nil
	}
	return nil, "", fmt.Errorf("%w: unsupported key type %T", errDecodingPrivateKey, parsed)
}

// signJWT signs a JWT's signing input with a private key, an ECDSA signature as the raw R||S JWS takes
func signJWT(privKey crypto.PrivateKey, signingInput string) ([]byte, error) {
	switch k := privKey.(type) {
	case *ecdsa.PrivateKey:
		if k.Curve != elliptic.P256() {
			return nil, fmt.Errorf("%w: ECDSA private key must use P-256", errDecodingPrivateKey)
		}
		hash := sha256.Sum256([]byte(signingInput))
		r, s, err := ecdsa.Sign(rand.Reader, k, hash[:])
		if err != nil {
			return nil, err
		}
		// Some verifiers refuse a high S value, so it is normalised to the lower half of the curve's order
		n := k.Params().N
		if s.Cmp(new(big.Int).Rsh(n, 1)) == 1 {
			s.Sub(n, s)
		}
		sig := make([]byte, 64)
		r.FillBytes(sig[:32])
		s.FillBytes(sig[32:])
		return sig, nil
	case ed25519.PrivateKey:
		return ed25519.Sign(k, []byte(signingInput)), nil
	default:
		return nil, common.GetTypeAssertError("*ecdsa.PrivateKey|ed25519.PrivateKey", privKey)
	}
}

// formatFloat formats a number as the decimal string Coinbase takes, never in exponent notation
func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// formatTime formats a time as the RFC 3339 timestamp Coinbase takes, in UTC
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// checkWindow returns an error when a window's start is after its end; either may be unset
func checkWindow(start, end time.Time) error {
	if !start.IsZero() && !end.IsZero() {
		return common.StartEndTimeCheck(start, end)
	}
	return nil
}
