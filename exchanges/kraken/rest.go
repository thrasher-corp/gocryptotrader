package kraken

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with Kraken
type Exchange struct {
	exchange.Base
	assetNames assetNames
	// wsToken is the token private websocket subscriptions and requests carry
	wsToken   string
	wsTokenMu sync.RWMutex
	// bookDepths holds the depth each order book is subscribed with, which updates are truncated to
	bookDepths   map[key.PairAsset]int
	bookDepthsMu sync.Mutex
	// level3Books holds each level 3 order book, which updates are applied to before it is loaded whole
	level3Books   map[key.PairAsset]*level3Book
	level3BooksMu sync.Mutex
	// futuresChallenge is the signed challenge private futures feeds are subscribed with on a connection
	futuresChallenge   futuresChallenge
	futuresChallengeMu sync.Mutex
	// futuresRequestMu sends futures websocket requests one at a time, as Kraken answers them without a request ID
	futuresRequestMu sync.Mutex
}

const (
	spotAPIURL                 = "https://api.kraken.com"
	futuresAPIURL              = "https://futures.kraken.com/derivatives"
	futuresSupplementaryAPIURL = "https://futures.kraken.com/api/"
	tradeBaseURL               = "https://pro.kraken.com/app/trade/"
	tradeFuturesURL            = "https://futures.kraken.com/trade/futures/"

	krakenRateInterval = time.Second
	krakenRequestRate  = 1
)

// SendHTTPRequest sends an unauthenticated request to a Spot REST public endpoint and decodes its result
func (e *Exchange) SendHTTPRequest(ctx context.Context, path string, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(exchange.RestSpot)
	if err != nil {
		return err
	}
	var raw json.RawMessage
	err = e.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return &request.Item{
			Method:                 http.MethodGet,
			Path:                   endpoint + path,
			Result:                 &raw,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.UnauthenticatedRequest)
	return e.decodeSpotResponse(raw, err, result)
}

// SendAuthenticatedHTTPRequest sends a signed request to a Spot REST private endpoint and decodes its result. body is
// sent as JSON with the nonce added; query holds the parameters Kraken documents as query parameters, such as
// account_id
func (e *Exchange) SendAuthenticatedHTTPRequest(ctx context.Context, path string, query url.Values, body map[string]any, result any) error {
	var raw json.RawMessage
	err := e.sendSpotPrivateRequest(ctx, path, query, body, &raw)
	return e.decodeSpotResponse(raw, err, result)
}

// sendSpotPrivateRequest signs and sends a Spot REST private request, storing the response body in out, which may be a
// *json.RawMessage or, for an endpoint replying with a file, a *[]byte. Kraken signs the request URI, query included,
// followed by the SHA-256 of the nonce and the body
func (e *Exchange) sendSpotPrivateRequest(ctx context.Context, path string, query url.Values, body map[string]any, out any) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := e.API.Endpoints.GetURL(exchange.RestSpot)
	if err != nil {
		return err
	}
	requestURI := common.EncodeURLValues(path, query)
	return e.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		n := e.Requester.GetNonce(nonce.UnixNano)
		payload := make(map[string]any, len(body)+1)
		maps.Copy(payload, body)
		payload["nonce"] = int64(n)
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		signature, err := spotSignature(creds.Secret, requestURI, n.String(), data)
		if err != nil {
			return nil, err
		}
		return &request.Item{
			Method: http.MethodPost,
			Path:   endpoint + requestURI,
			Headers: map[string]string{
				"API-Key":      creds.Key,
				"API-Sign":     signature,
				"Content-Type": "application/json",
			},
			Body:                   bytes.NewReader(data),
			Result:                 out,
			NonceEnabled:           true,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.AuthenticatedRequest)
}

// SendHeaderAuthenticatedHTTPRequest sends a signed request to the Funding (Beta) or Affiliate REST API and decodes its
// response. These APIs take the nonce in the API-Nonce header, sign the path with its query string, reply with bare
// objects rather than Spot REST's envelope and report errors as problem details with an error status. A non-nil body
// is sent as JSON
func (e *Exchange) SendHeaderAuthenticatedHTTPRequest(ctx context.Context, method, path string, query url.Values, body, result any) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := e.API.Endpoints.GetURL(exchange.RestSpot)
	if err != nil {
		return err
	}
	var data []byte
	if body != nil {
		if data, err = json.Marshal(body); err != nil {
			return err
		}
	}
	requestURI := common.EncodeURLValues(path, query)
	var raw json.RawMessage
	err = e.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		n := e.Requester.GetNonce(nonce.UnixNano).String()
		signature, err := spotSignature(creds.Secret, requestURI, n, data)
		if err != nil {
			return nil, err
		}
		headers := map[string]string{
			"API-Key":   creds.Key,
			"API-Nonce": n,
			"API-Sign":  signature,
		}
		item := &request.Item{
			Method:                 method,
			Path:                   endpoint + requestURI,
			Headers:                headers,
			Result:                 &raw,
			NonceEnabled:           true,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}
		if data != nil {
			headers["Content-Type"] = "application/json"
			item.Body = bytes.NewReader(data)
		}
		return item, nil
	}, request.AuthenticatedRequest)
	if err != nil {
		if apiErr := problemError(raw); apiErr != nil {
			return fmt.Errorf("%w: %w", apiErr, err)
		}
		return err
	}
	if result == nil {
		return nil
	}
	if len(raw) == 0 || string(raw) == "null" {
		return common.ErrNoResponse
	}
	return json.Unmarshal(raw, result)
}

// SendFuturesHTTPRequest sends an unauthenticated request to a Derivatives REST endpoint and decodes its response. ep
// is RestFutures for the v3 API, or RestFuturesSupplementary for the history and charts APIs
func (e *Exchange) SendFuturesHTTPRequest(ctx context.Context, ep exchange.URL, path string, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	var raw json.RawMessage
	err = e.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return &request.Item{
			Method:                 http.MethodGet,
			Path:                   strings.TrimSuffix(endpoint, "/") + path,
			Result:                 &raw,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.UnauthenticatedRequest)
	return decodeFuturesResponse(raw, err, result)
}

// SendFuturesAuthenticatedHTTPRequest sends a signed request to a Derivatives REST endpoint and decodes its response.
// Kraken documents every parameter as a query parameter, whatever the method, and signs the URL-encoded query string as
// sent, the nonce and the endpoint path without its /derivatives prefix. headers adds the optional headers some
// endpoints take, such as algoId
func (e *Exchange) SendFuturesAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method, path string, params url.Values, headers map[string]string, result any) error {
	_, err := e.sendFuturesAuthenticatedRequest(ctx, ep, method, path, params, headers, result, false)
	return err
}

// sendFuturesAuthenticatedRequest signs and sends a Derivatives REST request. When raw is true the response body is
// returned as is, for endpoints replying with a file, and result is not decoded
func (e *Exchange) sendFuturesAuthenticatedRequest(ctx context.Context, ep exchange.URL, method, path string, params url.Values, headers map[string]string, result any, raw bool) ([]byte, error) {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return nil, err
	}
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return nil, err
	}
	target := strings.TrimSuffix(endpoint, "/") + path
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	signingPath := strings.TrimPrefix(u.Path, "/derivatives")
	// Kraken hashes the query string as it appears in the request, where a space is %20
	postData := strings.ReplaceAll(params.Encode(), "+", "%20")
	if postData != "" {
		target += "?" + postData
	}
	var body []byte
	err = e.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		n := strconv.FormatInt(time.Now().UnixNano(), 10)
		authent, err := futuresSignature(creds.Secret, postData, n, signingPath)
		if err != nil {
			return nil, err
		}
		h := make(map[string]string, len(headers)+3)
		maps.Copy(h, headers)
		h["APIKey"] = creds.Key
		h["Authent"] = authent
		h["Nonce"] = n
		item := &request.Item{
			Method:                 method,
			Path:                   target,
			Headers:                h,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}
		if raw {
			item.Result = &body
		} else {
			item.Result = (*json.RawMessage)(&body)
		}
		return item, nil
	}, request.AuthenticatedRequest)
	if raw {
		if err != nil {
			if apiErr := futuresError(body, true); apiErr != nil {
				return nil, fmt.Errorf("%w: %w", apiErr, err)
			}
			return nil, err
		}
		return body, nil
	}
	return nil, decodeFuturesResponse(body, err, result)
}

// spotSignature returns the API-Sign of a Spot REST private, Funding (Beta) or Affiliate REST request: the
// HMAC-SHA512, keyed by the decoded secret, of the request URI followed by the SHA-256 of the nonce and the body
func spotSignature(secret, requestURI, nonceValue string, body []byte) (string, error) {
	shasum := sha256.Sum256(append([]byte(nonceValue), body...))
	hmac, err := crypto.GetHMAC(crypto.HashSHA512, append([]byte(requestURI), shasum[:]...), []byte(secret))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(hmac), nil
}

// futuresSignature returns the Authent of a Derivatives REST request: the HMAC-SHA512, keyed by the decoded secret, of
// the SHA-256 of the URL-encoded parameters, the nonce and the endpoint path
func futuresSignature(secret, postData, nonceValue, endpointPath string) (string, error) {
	return futuresChallengeSignature(secret, postData+nonceValue+endpointPath)
}

// futuresChallengeSignature signs a Derivatives websocket challenge: the HMAC-SHA512, keyed by the decoded secret, of
// the challenge's SHA-256
func futuresChallengeSignature(secret, challenge string) (string, error) {
	shasum := sha256.Sum256([]byte(challenge))
	hmac, err := crypto.GetHMAC(crypto.HashSHA512, shasum[:], []byte(secret))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(hmac), nil
}

// spotMessages holds the error array of a Spot REST response, whose entries begin with their severity, E for an error
// or W for a warning. A single message sent as a string is accepted too
type spotMessages []string

// UnmarshalJSON decodes an error array or a single error string
func (s *spotMessages) UnmarshalJSON(data []byte) error {
	if len(data) != 0 && data[0] == '"' {
		var msg string
		if err := json.Unmarshal(data, &msg); err != nil {
			return err
		}
		*s = spotMessages{msg}
		return nil
	}
	var msgs []string
	if err := json.Unmarshal(data, &msgs); err != nil {
		return err
	}
	*s = msgs
	return nil
}

// spotResponse is the envelope of every Spot REST response
type spotResponse struct {
	Error  spotMessages    `json:"error"`
	Result json.RawMessage `json:"result"`
}

// decodeSpotResponse decodes a Spot REST response envelope into result, returning the errors it holds as an APIError
// and logging its warnings
func (e *Exchange) decodeSpotResponse(raw json.RawMessage, sendErr error, result any) error {
	var resp spotResponse
	if len(raw) != 0 {
		if err := json.Unmarshal(raw, &resp); err != nil && sendErr == nil {
			return fmt.Errorf("error decoding Spot REST response: %w", err)
		}
	}
	var apiErr *APIError
	for _, msg := range resp.Error {
		if strings.HasPrefix(msg, "W") {
			log.Warnf(log.ExchangeSys, "%s REST request warning: %s", e.Name, msg)
			continue
		}
		if apiErr == nil {
			apiErr = new(APIError)
		}
		apiErr.Errors = append(apiErr.Errors, msg)
	}
	switch {
	case apiErr != nil && sendErr != nil:
		return fmt.Errorf("%w: %w", apiErr, sendErr)
	case apiErr != nil:
		return apiErr
	case sendErr != nil:
		return sendErr
	case result == nil:
		return nil
	case len(resp.Result) == 0 || string(resp.Result) == "null":
		return common.ErrNoResponse
	}
	return json.Unmarshal(resp.Result, result)
}

// problemDetails is an error body of the Funding (Beta) and Affiliate REST APIs. The gateway in front of them answers a
// route it does not know as Spot REST does, with an error array
type problemDetails struct {
	Type   string       `json:"type"`
	Title  string       `json:"title"`
	Detail string       `json:"detail"`
	Error  spotMessages `json:"error"`
}

// problemError returns the error a Funding (Beta) or Affiliate REST error body describes, or nil when it describes none
func problemError(raw json.RawMessage) *APIError {
	var resp problemDetails
	if !decodeErrorBody(raw, &resp) {
		return nil
	}
	if resp.Title == "" && resp.Detail == "" {
		if len(resp.Error) == 0 {
			return nil
		}
		return &APIError{Errors: resp.Error}
	}
	msg := resp.Title
	if resp.Detail != "" {
		if msg != "" {
			msg += ": "
		}
		msg += resp.Detail
	}
	return &APIError{Errors: []string{msg}, Type: resp.Type}
}

// decodeErrorBody decodes an error body into resp, reporting whether it is a JSON object, so a body that is not one
// leaves the caller's decoding to report it
func decodeErrorBody(raw json.RawMessage, resp any) bool {
	return len(raw) != 0 && raw[0] == '{' && json.Unmarshal(raw, resp) == nil
}

// futuresErrorResponse holds the fields the Derivatives REST APIs report errors in: result, error and errors on the v3
// API, error and message on the authentication API, and status and reason on the history API
type futuresErrorResponse struct {
	Result  string            `json:"result"`
	Error   string            `json:"error"`
	Errors  []json.RawMessage `json:"errors"`
	Message string            `json:"message"`
	Status  string            `json:"status"`
	Reason  string            `json:"reason"`
}

// futuresErrorCode is an entry of the errors array some Derivatives REST v3 errors carry instead of a string. The
// charts API's entries are analytics problems instead
type futuresErrorCode struct {
	Code    *int64 `json:"code"`
	Message string `json:"message"`
}

// futuresError returns the error a Derivatives REST response body describes, or nil when it describes none. A v3
// response says so with result "error", whatever its status; the other fields only describe an error alongside an
// error status
func futuresError(raw json.RawMessage, errorStatus bool) *APIError {
	var resp futuresErrorResponse
	if !decodeErrorBody(raw, &resp) {
		return nil
	}
	if resp.Result != "error" && !errorStatus {
		return nil
	}
	var msgs []string
	if resp.Error != "" {
		msgs = append(msgs, resp.Error)
	}
	if resp.Message != "" {
		msgs = append(msgs, resp.Message)
	}
	for _, e := range resp.Errors {
		var msg string
		if json.Unmarshal(e, &msg) == nil {
			msgs = append(msgs, msg)
			continue
		}
		var code futuresErrorCode
		if json.Unmarshal(e, &code) == nil && (code.Code != nil || code.Message != "") {
			if code.Code != nil {
				msgs = append(msgs, "code "+strconv.FormatInt(*code.Code, 10)+": "+code.Message)
			} else {
				msgs = append(msgs, code.Message)
			}
			continue
		}
		var problem futuresAnalyticsError
		if json.Unmarshal(e, &problem) == nil {
			msgs = append(msgs, problem.message())
		}
	}
	apiErr := &APIError{Type: resp.Status}
	if resp.Reason != "" {
		msgs = append(msgs, resp.Reason)
	}
	if len(msgs) == 0 {
		switch {
		case resp.Status != "":
			msgs = append(msgs, resp.Status)
		case resp.Result == "error":
			msgs = append(msgs, string(raw))
		default:
			return nil
		}
	}
	apiErr.Errors = msgs
	return apiErr
}

// decodeFuturesResponse decodes a Derivatives REST response into result, returning any error it describes as an
// APIError
func decodeFuturesResponse(raw json.RawMessage, sendErr error, result any) error {
	if apiErr := futuresError(raw, sendErr != nil); apiErr != nil {
		if sendErr != nil {
			return fmt.Errorf("%w: %w", apiErr, sendErr)
		}
		return apiErr
	}
	if sendErr != nil {
		return sendErr
	}
	if result == nil {
		return nil
	}
	if len(raw) == 0 || string(raw) == "null" {
		return common.ErrNoResponse
	}
	return json.Unmarshal(raw, result)
}

// APIError is an error Kraken replied with
type APIError struct {
	// Errors holds each error as Kraken sent it, such as "EOrder:Unknown order", "authenticationError" or "API: Invalid
	// key"
	Errors []string
	// Type is the problem type of a Funding (Beta) or Affiliate REST error, such as "tag:kraken.com,2025:InvalidKey", or
	// the status of a Derivatives history error, such as "unauthorized"
	Type string
}

// Error returns the errors Kraken replied with
func (a *APIError) Error() string {
	msg := errAPIResponse.Error() + ": " + strings.Join(a.Errors, ", ")
	if a.Type != "" {
		msg += " (" + a.Type + ")"
	}
	return msg
}

// Unwrap returns errAPIResponse and the package errors the replied errors map to, so callers can match either
func (a *APIError) Unwrap() []error {
	errs := []error{errAPIResponse}
	if mapped, ok := errorToErrorMap[a.Type]; ok {
		errs = append(errs, mapped)
	}
	for _, msg := range a.Errors {
		if mapped, ok := errorToErrorMap[msg]; ok {
			errs = append(errs, mapped)
		}
	}
	return errs
}

// errorToErrorMap maps the errors and problem types Kraken replies with to package errors callers can match
var errorToErrorMap = map[string]error{
	"EAPI:Invalid key":               request.ErrAuthRequestFailed,
	"EAPI:Invalid signature":         request.ErrAuthRequestFailed,
	"EAPI:Invalid nonce":             request.ErrAuthRequestFailed,
	"EGeneral:Permission denied":     request.ErrAuthRequestFailed,
	"tag:kraken.com,2025:InvalidKey": request.ErrAuthRequestFailed,
	"authenticationError":            request.ErrAuthRequestFailed,
	"nonceBelowThreshold":            request.ErrAuthRequestFailed,
	"nonceDuplicate":                 request.ErrAuthRequestFailed,
	"api_key_not_found":              request.ErrAuthRequestFailed,
	"unauthorized":                   request.ErrAuthRequestFailed,
	"EOrder:Unknown order":           order.ErrOrderNotFound,
	"EQuery:Unknown asset pair":      currency.ErrPairNotFound,
	"contractNotFound":               currency.ErrPairNotFound,
}

var errAPIResponse = errors.New("API error response")
