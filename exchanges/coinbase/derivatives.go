package coinbase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// derivativesPath prefixes every gateway method over HTTP
	derivativesPath = "/api/v2/"
	jsonRPCVersion  = "2.0"
	// derivativesPrivatePrefix starts the name of every gateway method that needs authentication
	derivativesPrivatePrefix = "private/"
	derivativesResultOK      = "ok"

	// derivativesAuthMethod exchanges a CDP JWT for an access token over HTTP, or authenticates a websocket connection
	derivativesAuthMethod = "public/auth"
	// derivativesGrantType is the public/auth grant a CDP API key's JWT is exchanged with
	derivativesGrantType = "coinbase_cdp"
	// derivativesTokenRefreshMargin is how long before an access token expires it is replaced, so a request made with it
	// cannot reach the gateway after it expires
	derivativesTokenRefreshMargin = time.Minute

	// derivativesErrorCodeRetry asks the client to send the request again, as the gateway could not process it yet
	derivativesErrorCodeRetry = 10040
	// derivativesErrorCodeUnauthorised is the gateway's answer to a private request whose access token it no longer
	// accepts
	derivativesErrorCodeUnauthorised = 13009
	// derivativesMaximumRequestAttempts bounds how often a request the gateway asks to retry is sent
	derivativesMaximumRequestAttempts = 5
)

var (
	errInstrumentNameRequired      = errors.New("instrument name required")
	errDerivativesUnexpectedResult = errors.New("unexpected result")
	errDerivativesAccessToken      = errors.New("gateway issued no access token")
	errDerivativesTimeBeforeEpoch  = errors.New("time before the Unix epoch")
)

// derivativesQueryMethods are the gateway methods that only answer an HTTP GET request with their params in the query
// string; the gateway answers a JSON-RPC body sent to them with "Method not found"
var derivativesQueryMethods = map[string]bool{
	"public/get_tradingview_chart_data": true,
}

// GlobalDerivativesAPI calls the Global Derivatives gateway's JSON-RPC methods over HTTP. The gateway runs Deribit's
// API for Coinbase accounts, authenticated with the same CDP API key as Advanced Trade; WebsocketAPI calls the same
// methods over the websocket
type GlobalDerivativesAPI struct {
	e *Exchange
}

// GlobalDerivatives returns the methods called on the Global Derivatives gateway over HTTP
func (e *Exchange) GlobalDerivatives() GlobalDerivativesAPI {
	return GlobalDerivativesAPI{e: e}
}

// derivativesRequestID returns a gateway websocket request's ID, the next of the exchange's message sequence
func (e *Exchange) derivativesRequestID() uint64 {
	return uint64(e.MessageSequence()) //nolint:gosec // The message sequence counts up from one
}

// derivativesUnixMilli returns a time in Unix milliseconds, as the gateway's timestamps are, or zero for an unset time,
// which leaves an optional timestamp out of a request; a time before 1970 has no timestamp, so it is refused
func derivativesUnixMilli(t time.Time) (uint64, error) {
	if t.IsZero() {
		return 0, nil
	}
	milli := t.UnixMilli()
	if milli < 0 {
		return 0, fmt.Errorf("%w: %s", errDerivativesTimeBeforeEpoch, t)
	}
	return uint64(milli), nil
}

// derivativesTokenLifetime returns a token's lifetime in seconds as a duration, capped at the longest a duration holds
func derivativesTokenLifetime(expiresIn uint64) time.Duration {
	return time.Duration(min(expiresIn, uint64(math.MaxInt64/time.Second))) * time.Second
}

// derivativesUnixMilliRange returns a time range's ends in Unix milliseconds, as derivativesUnixMilli does
func derivativesUnixMilliRange(start, end time.Time) (startMilli, endMilli uint64, err error) {
	if startMilli, err = derivativesUnixMilli(start); err != nil {
		return 0, 0, err
	}
	if endMilli, err = derivativesUnixMilli(end); err != nil {
		return 0, 0, err
	}
	return startMilli, endMilli, nil
}

// JSONRPCRequest is the envelope of every gateway request; requests sent over HTTP leave the ID out, as each HTTP
// response answers its own request
type JSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      uint64 `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// JSONRPCResponse is the envelope of every gateway response; Error is set in place of Result when a request fails
type JSONRPCResponse struct {
	JSONRPC string            `json:"jsonrpc"`
	ID      uint64            `json:"id"`
	Result  any               `json:"result"`
	Error   *DerivativesError `json:"error"`
	Testnet bool              `json:"testnet"`
}

// DerivativesError is the error object the gateway returns in place of a result
type DerivativesError struct {
	// Code is negative for JSON-RPC protocol errors and positive for the gateway's own
	Code    int64  `json:"code"`
	Message string `json:"message"`
	// Data holds the error's details, usually the parameter at fault and the reason, although some errors carry other
	// fields, so it is kept as sent
	Data json.RawMessage `json:"data"`
}

// Error returns the error's code and message, followed by its details when the gateway sent any
func (d *DerivativesError) Error() string {
	msg := "code " + strconv.FormatInt(d.Code, 10) + ": " + d.Message
	if data := bytes.TrimSpace(d.Data); len(data) > 0 && !bytes.Equal(data, []byte("null")) {
		msg += " " + string(data)
	}
	return msg
}

// DerivativesAuthWireRequest holds public/auth's params for a CDP API key's JWT
type DerivativesAuthWireRequest struct {
	GrantType string `json:"grant_type"`
	Token     string `json:"token"`
}

// DerivativesAuthResponse is the access token public/auth issues; over the websocket the connection itself is
// authenticated instead, and the token is not needed
type DerivativesAuthResponse struct {
	AccessToken     string   `json:"access_token"`
	EnabledFeatures []string `json:"enabled_features"`
	// ExpiresIn is the token's lifetime in seconds
	ExpiresIn          uint64 `json:"expires_in"`
	GoogleLogin        bool   `json:"google_login"`
	MandatoryTFAStatus string `json:"mandatory_tfa_status"`
	RefreshToken       string `json:"refresh_token"`
	// Scope lists the granted scopes, space separated
	Scope     string `json:"scope"`
	SessionID string `json:"sid"`
	State     string `json:"state"`
	TokenType string `json:"token_type"`
}

// derivativesAccessToken caches the gateway's HTTP access tokens by API key, as a context can carry other credentials
// than the exchange's own
type derivativesAccessToken struct {
	m      sync.Mutex
	tokens map[string]derivativesToken
	// renewal is held while a token is renewed, so requests finding no token share one public/auth rather than each
	// sending one; it is a channel, so a request waiting for it stops when its context ends
	renewal chan struct{}
}

// lockRenewal waits to renew a token, until no other renewal is under way or the context ends, and returns the
// renewal's release
func (d *derivativesAccessToken) lockRenewal(ctx context.Context) (func(), error) {
	d.m.Lock()
	if d.renewal == nil {
		d.renewal = make(chan struct{}, 1)
	}
	renewal := d.renewal
	d.m.Unlock()
	select {
	case renewal <- struct{}{}:
		if err := ctx.Err(); err != nil {
			<-renewal
			return nil, err
		}
		return func() { <-renewal }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// derivativesToken is an access token and the time it expires
type derivativesToken struct {
	token   string
	expires time.Time
}

// get returns an API key's cached access token, when it is still far enough from expiring to use
func (d *derivativesAccessToken) get(apiKey string) (string, bool) {
	d.m.Lock()
	defer d.m.Unlock()
	t, ok := d.tokens[apiKey]
	if !ok || time.Until(t.expires) < derivativesTokenRefreshMargin {
		return "", false
	}
	return t.token, true
}

// set caches an API key's access token until it expires
func (d *derivativesAccessToken) set(apiKey, token string, expires time.Time) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.tokens == nil {
		d.tokens = make(map[string]derivativesToken)
	}
	d.tokens[apiKey] = derivativesToken{token: token, expires: expires}
}

// clear drops an API key's cached access token, when the token it holds is the given one; a token refreshed by a
// concurrent request is kept
func (d *derivativesAccessToken) clear(apiKey, token string) {
	d.m.Lock()
	defer d.m.Unlock()
	if d.tokens[apiKey].token == token {
		delete(d.tokens, apiKey)
	}
}

// Authenticate exchanges a JWT signed with the configured CDP API key for an access token, which authenticates the
// gateway's private methods over HTTP for about 15 minutes; private methods fetch and refresh one themselves
func (g GlobalDerivativesAPI) Authenticate(ctx context.Context) (*DerivativesAuthResponse, error) {
	endpoint, err := g.e.endpointURL(exchange.RestFutures)
	if err != nil {
		return nil, err
	}
	host, err := endpointHost(endpoint)
	if err != nil {
		return nil, err
	}
	jwt, _, err := g.e.GetJWT(ctx, http.MethodPost+" "+host+derivativesPath+derivativesAuthMethod)
	if err != nil {
		return nil, err
	}
	var resp *DerivativesAuthResponse
	if err := g.e.sendDerivativesHTTPRequest(ctx, derivativesNonMatchingEPL, derivativesAuthMethod, &DerivativesAuthWireRequest{GrantType: derivativesGrantType, Token: jwt}, &resp, ""); err != nil {
		return nil, err
	}
	if resp == nil || resp.AccessToken == "" {
		return nil, errDerivativesAccessToken
	}
	return resp, nil
}

// accessToken returns an access token for the context's credentials, exchanging a JWT for one when none is cached
func (g GlobalDerivativesAPI) accessToken(ctx context.Context) (apiKey, token string, err error) {
	creds, err := g.e.GetCredentials(ctx)
	if err != nil {
		return "", "", err
	}
	if token, ok := g.e.derivativesToken.get(creds.Key); ok {
		return creds.Key, token, nil
	}
	release, err := g.e.derivativesToken.lockRenewal(ctx)
	if err != nil {
		return "", "", err
	}
	defer release()
	if token, ok := g.e.derivativesToken.get(creds.Key); ok {
		return creds.Key, token, nil
	}
	resp, err := g.Authenticate(ctx)
	if err != nil {
		return "", "", fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, err)
	}
	g.e.derivativesToken.set(creds.Key, resp.AccessToken, time.Now().Add(derivativesTokenLifetime(resp.ExpiresIn)))
	return creds.Key, resp.AccessToken, nil
}

// SendHTTPRequest sends a JSON-RPC request for a public gateway method over HTTP and decodes its result
func (g GlobalDerivativesAPI) SendHTTPRequest(ctx context.Context, epl request.EndpointLimit, method string, params, result any) error {
	return derivativesRetry(ctx, func() error {
		return g.e.sendDerivativesHTTPRequest(ctx, epl, method, params, result, "")
	})
}

// SendAuthenticatedHTTPRequest sends a JSON-RPC request for a private gateway method over HTTP, with an access token
// for the configured credentials, and decodes its result. An access token the gateway no longer accepts is replaced
// once
func (g GlobalDerivativesAPI) SendAuthenticatedHTTPRequest(ctx context.Context, epl request.EndpointLimit, method string, params, result any) error {
	return derivativesRetry(ctx, func() error {
		for attempt := 0; ; attempt++ {
			apiKey, token, err := g.accessToken(ctx)
			if err != nil {
				return err
			}
			err = g.e.sendDerivativesHTTPRequest(ctx, epl, method, params, result, token)
			if derivativesErr, ok := errors.AsType[*DerivativesError](err); attempt == 0 && ok && derivativesErr.Code == derivativesErrorCodeUnauthorised {
				g.e.derivativesToken.clear(apiKey, token)
				continue
			}
			return err
		}
	})
}

// sendRequest sends a gateway request over the websocket or HTTP, with an access token for a private method over
// HTTP
func (g GlobalDerivativesAPI) sendRequest(ctx context.Context, overWebsocket bool, epl request.EndpointLimit, method string, params, result any) error {
	switch {
	case overWebsocket:
		return g.e.WebsocketAPI().SendRequest(ctx, epl, method, params, result)
	case isDerivativesPrivateMethod(method):
		return g.SendAuthenticatedHTTPRequest(ctx, epl, method, params, result)
	default:
		return g.SendHTTPRequest(ctx, epl, method, params, result)
	}
}

// sendOKRequest sends a gateway request whose result is "ok" when it succeeds, none of which reach the matching engine
func (g GlobalDerivativesAPI) sendOKRequest(ctx context.Context, overWebsocket bool, method string, params any) error {
	var resp string
	if err := g.sendRequest(ctx, overWebsocket, derivativesNonMatchingEPL, method, params, &resp); err != nil {
		return err
	}
	if resp != derivativesResultOK {
		return fmt.Errorf("%w: %s returned %q", errDerivativesUnexpectedResult, method, resp)
	}
	return nil
}

// sendDerivativesHTTPRequest sends a gateway method over HTTP, with the access token when one is given, and returns
// the error object the gateway sends in place of a result. Most methods are posted as a JSON-RPC body; the few that
// only answer a GET request send their params in the query string
func (e *Exchange) sendDerivativesHTTPRequest(ctx context.Context, epl request.EndpointLimit, method string, params, result any, token string) error {
	endpoint, err := e.endpointURL(exchange.RestFutures)
	if err != nil {
		return err
	}
	path := endpoint + derivativesPath + method
	var body []byte
	if derivativesQueryMethods[method] {
		query, err := derivativesQueryParams(params)
		if err != nil {
			return err
		}
		path = common.EncodeURLValues(path, query)
	} else if body, err = json.Marshal(&JSONRPCRequest{JSONRPC: jsonRPCVersion, Method: method, Params: params}); err != nil {
		return err
	}
	requestType := request.AuthType(request.UnauthenticatedRequest)
	if token != "" {
		requestType = request.AuthenticatedRequest
	}
	resp := &JSONRPCResponse{Result: result}
	err = e.SendPayload(ctx, epl, func() (*request.Item, error) {
		item := &request.Item{
			Method:                 http.MethodGet,
			Path:                   path,
			Headers:                map[string]string{},
			Result:                 resp,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}
		if body != nil {
			item.Method = http.MethodPost
			item.Headers["Content-Type"] = "application/json"
			item.Body = bytes.NewReader(body)
		}
		if token != "" {
			item.Headers["Authorization"] = "Bearer " + token
		}
		return item, nil
	}, requestType)
	// The gateway answers a failed request with an error status, so its error object explains the failure better; an
	// authenticated request's failure is still reported as one, as the requester reports it
	if resp.Error != nil {
		if token != "" {
			return fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, resp.Error)
		}
		return resp.Error
	}
	return err
}

// derivativesQueryParams returns a method's params as a query string: strings as they are, other values as JSON
func derivativesQueryParams(params any) (url.Values, error) {
	values := url.Values{}
	if params == nil {
		return values, nil
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, err
	}
	for k, v := range fields {
		var s string
		if json.Unmarshal(v, &s) == nil {
			values.Set(k, s)
			continue
		}
		values.Set(k, string(v))
	}
	return values, nil
}

// isDerivativesPrivateMethod reports whether a gateway method needs authentication
func isDerivativesPrivateMethod(method string) bool {
	return strings.HasPrefix(method, derivativesPrivatePrefix)
}

// derivativesRetry sends a request until the gateway stops asking for it to be retried, backing off between attempts
func derivativesRetry(ctx context.Context, send func() error) error {
	backoff := request.DefaultBackoff()
	for attempt := 1; ; attempt++ {
		err := send()
		derivativesErr, ok := errors.AsType[*DerivativesError](err)
		if attempt == derivativesMaximumRequestAttempts || !ok || derivativesErr.Code != derivativesErrorCodeRetry {
			return err
		}
		timer := time.NewTimer(backoff(attempt))
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// WebsocketAPI calls the Global Derivatives gateway's JSON-RPC methods over its private websocket connection; a method
// with an HTTP form takes that form's name and arguments
type WebsocketAPI struct {
	e *Exchange
}

// WebsocketAPI returns the gateway methods called over the websocket
func (e *Exchange) WebsocketAPI() WebsocketAPI {
	return WebsocketAPI{e: e}
}

var (
	errDerivativesWebsocketNotAuthenticated = errors.New("gateway websocket connection is not authenticated")
	errDerivativesWebsocketOtherAccount     = errors.New("gateway websocket session is authenticated with another API key")
)

// SendRequest sends a JSON-RPC request over the gateway's private websocket connection and decodes its result; a
// private method needs the connection to be authenticated with the API key the context's credentials hold, as the
// session acts for that key's account alone
func (w WebsocketAPI) SendRequest(ctx context.Context, epl request.EndpointLimit, method string, params, result any) error {
	conn, err := w.connection(ctx, method)
	if err != nil {
		return err
	}
	return w.sendRequestOnConnection(ctx, conn, epl, method, params, result)
}

// connection returns the gateway's private websocket connection for a method; a private method needs the connection
// to be authenticated with the API key the context's credentials hold, as the session acts for that key's account
// alone
func (w WebsocketAPI) connection(ctx context.Context, method string) (websocket.Connection, error) {
	conn, err := w.e.Websocket.GetConnection(derivativesPrivateConnection)
	if err != nil {
		return nil, err
	}
	if !isDerivativesPrivateMethod(method) {
		return conn, nil
	}
	key := w.e.derivativesSessions.authenticatedKey(conn)
	if key == "" {
		return nil, fmt.Errorf("%w: %s", errDerivativesWebsocketNotAuthenticated, method)
	}
	creds, err := w.e.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}
	if creds.Key != key {
		return nil, fmt.Errorf("%w: %s", errDerivativesWebsocketOtherAccount, method)
	}
	return conn, nil
}

// sendRequestOnConnection sends a JSON-RPC request on a gateway websocket connection and decodes its result
func (w WebsocketAPI) sendRequestOnConnection(ctx context.Context, conn websocket.Connection, epl request.EndpointLimit, method string, params, result any) error {
	return derivativesRetry(ctx, func() error {
		req := &JSONRPCRequest{JSONRPC: jsonRPCVersion, ID: w.e.derivativesRequestID(), Method: method, Params: params}
		raw, err := conn.SendMessageReturnResponse(ctx, epl, req.ID, req)
		if err != nil {
			return err
		}
		resp := &JSONRPCResponse{Result: result}
		if err := json.Unmarshal(raw, resp); err != nil {
			return err
		}
		if resp.Error != nil {
			return resp.Error
		}
		return nil
	})
}
