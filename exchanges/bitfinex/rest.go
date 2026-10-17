package bitfinex

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with Bitfinex
type Exchange struct {
	exchange.Base
	wsChannels       wsChannelRegistry
	wsSharedChannels wsSharedChannels
	wsClientOrderID  nonce.Nonce
	wsInputMu        sync.Mutex
	// wsRenewMu keeps a channel's renewal apart from the manager unsubscribing it
	wsRenewMu        sync.RWMutex
	wsFundingBookIDs wsFundingBookIDs
	wsBookResync     wsBookResync
	wsWalletCalcs    wsWalletCalcs
}

const (
	// apiURL serves both the public and the authenticated endpoints; the docs suggest api-pub.bitfinex.com for public
	// ones, but saved configurations hold this URL as RestSpotURL for all of them
	apiURL       = "https://api.bitfinex.com"
	tradeBaseURL = "https://trading.bitfinex.com"
	// apiVersionPath prefixes every endpoint path, and the authenticated request signature
	apiVersionPath = "/v2/"
	// cidDateLayout is the layout of the cid_date parameter, the UTC day a client order ID belongs to
	cidDateLayout = time.DateOnly
	// tifLayout is the layout of the tif parameter, the UTC time an order is cancelled at
	tifLayout = time.DateTime
)

var (
	errSymbolRequired         = errors.New("symbol required")
	errTradingSymbolRequired  = errors.New("trading pair symbol required, such as tBTCUSD")
	errFundingSymbolRequired  = errors.New("funding currency symbol required, such as fUSD")
	errUnknownSymbolType      = errors.New("symbol is neither a trading pair nor a funding currency")
	errOrderIDRequired        = errors.New("order ID required")
	errClientOrderIDDateUnset = errors.New("client order ID date must be set with a client order ID")
	errInvalidAmount          = errors.New("invalid amount")
	errInvalidPrice           = errors.New("invalid price")
	errInvalidSort            = errors.New("invalid sort, want 1 for ascending or -1 for descending")
	errNotificationFailed     = errors.New("request failed")
)

// APIError is an error Bitfinex reports in a response body, as an ["error", CODE, MESSAGE] array or as an object
// holding an error
type APIError struct {
	Code    int64
	Message string
}

// Error returns the error's code and message
func (a *APIError) Error() string {
	if a.Code == 0 {
		return a.Message
	}
	return strconv.FormatInt(a.Code, 10) + " " + a.Message
}

// errorTuple is the ["error", CODE, MESSAGE] array Bitfinex reports errors in
type errorTuple struct {
	Event   string `tuple:"0"`
	Code    int64  `tuple:"1"`
	Message string `tuple:"2"`
}

// UnmarshalJSON decodes an error array
func (e *errorTuple) UnmarshalJSON(data []byte) error { return unmarshalTuple(data, e) }

// responseError returns the error a response body reports, or nil when it reports none
func responseError(body []byte) error {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil
	}
	switch body[0] {
	case '[':
		// Only the first element is read, so a large response is not decoded twice
		if event, dataType, _, _ := jsonparser.Get(body, "[0]"); dataType != jsonparser.String || string(event) != "error" {
			return nil
		}
		var e errorTuple
		if err := json.Unmarshal(body, &e); err != nil {
			return fmt.Errorf("decoding error response %s: %w", body, err)
		}
		return &APIError{Code: e.Code, Message: e.Message}
	case '{':
		// Some endpoints, and the rate limiter, report errors in an object; one without an error member reports none
		message, _ := jsonparser.GetString(body, "error")
		if message == "" {
			return nil
		}
		code, _ := jsonparser.GetInt(body, "code")
		if detail, _ := jsonparser.GetString(body, "message"); detail != "" && detail != message {
			message += ": " + detail
		}
		return &APIError{Code: code, Message: message}
	}
	return nil
}

// decodeResponse returns the error a response body reports, else the request's error, else decodes the body into
// result. Bitfinex reports errors in the body with any status, and the recorded mock responses always succeed
func decodeResponse(body json.RawMessage, requestErr error, result any) error {
	if err := responseError(body); err != nil {
		// The request system marks every failed authenticated request, which callers tell auth failures by
		if errors.Is(requestErr, request.ErrAuthRequestFailed) {
			return fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, err)
		}
		return err
	}
	if requestErr != nil {
		return requestErr
	}
	if result == nil || len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, result)
}

// SendHTTPRequest sends an unauthenticated request. path follows the version path and may hold a query string, and a
// non-nil body is sent as JSON
func (e *Exchange) SendHTTPRequest(ctx context.Context, epl request.EndpointLimit, method, path string, body, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(exchange.RestSpot)
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
		item := &request.Item{
			Method:                 method,
			Path:                   endpoint + apiVersionPath + path,
			Result:                 &response,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}
		if payload != nil {
			item.Headers = map[string]string{"Content-Type": "application/json"}
			item.Body = bytes.NewReader(payload)
		}
		return item, nil
	}, request.UnauthenticatedRequest)
	return decodeResponse(response, err, result)
}

// SendAuthenticatedHTTPRequest sends a signed POST request. path follows the version path, and body, which may be nil,
// is sent as JSON
func (e *Exchange) SendAuthenticatedHTTPRequest(ctx context.Context, epl request.EndpointLimit, path string, body, result any) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := e.API.Endpoints.GetURL(exchange.RestSpot)
	if err != nil {
		return err
	}
	payload := []byte("{}")
	if body != nil {
		if payload, err = json.Marshal(body); err != nil {
			return err
		}
	}
	var response json.RawMessage
	err = e.SendPayload(ctx, epl, func() (*request.Item, error) {
		// The docs ask for a nonce within JavaScript's MAX_SAFE_INTEGER, but keys GCT has signed for have only seen
		// nanosecond nonces, and each key refuses a nonce smaller than its last
		n := e.Requester.GetNonce(nonce.UnixNano).String()
		signature, err := crypto.GetHMAC(crypto.HashSHA512_384, []byte("/api"+apiVersionPath+path+n+string(payload)), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		return &request.Item{
			Method: http.MethodPost,
			Path:   endpoint + apiVersionPath + path,
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Accept":        "application/json",
				"bfx-apikey":    creds.Key,
				"bfx-nonce":     n,
				"bfx-signature": hex.EncodeToString(signature),
			},
			Body:                   bytes.NewReader(payload),
			Result:                 &response,
			NonceEnabled:           true,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.AuthenticatedRequest)
	return decodeResponse(response, err, result)
}

// timeParam formats a time as the Unix millisecond timestamp Bitfinex takes
func timeParam(t time.Time) string {
	return strconv.FormatInt(t.UnixMilli(), 10)
}

// sortParam returns the sort parameter for an ascending or descending sort, or an error for any other value
func sortParam(sort int64) (string, error) {
	switch sort {
	case 1, -1:
		return strconv.FormatInt(sort, 10), nil
	default:
		return "", fmt.Errorf("%w: %d", errInvalidSort, sort)
	}
}

// isTradingSymbol reports whether a symbol names a trading pair, which Bitfinex prefixes with t
func isTradingSymbol(symbol string) bool {
	return len(symbol) > 1 && symbol[0] == 't'
}

// isFundingSymbol reports whether a symbol names a funding currency, which Bitfinex prefixes with f
func isFundingSymbol(symbol string) bool {
	return len(symbol) > 1 && symbol[0] == 'f'
}

// sendNotificationRequest sends a signed write request and returns the notification Bitfinex replies with, erroring
// when the notification reports the request failed
func sendNotificationRequest[T any](ctx context.Context, e *Exchange, epl request.EndpointLimit, path string, body any) (*Notification[T], error) {
	var resp *Notification[T]
	if err := e.SendAuthenticatedHTTPRequest(ctx, epl, path, body, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return resp, nil
}

// checkWindow returns an error when a window's start is after its end; either may be unset
func checkWindow(start, end time.Time) error {
	if !start.IsZero() && !end.IsZero() {
		return common.StartEndTimeCheck(start, end)
	}
	return nil
}

// setHistoryParams sets the window, limit and sort of a public history endpoint's query; each is sent only when set,
// and sort must be 1 for oldest first or -1 for newest first
func setHistoryParams(params url.Values, start, end time.Time, limit uint64, sort int64) error {
	if err := checkWindow(start, end); err != nil {
		return err
	}
	if sort != 0 {
		s, err := sortParam(sort)
		if err != nil {
			return err
		}
		params.Set("sort", s)
	}
	if !start.IsZero() {
		params.Set("start", timeParam(start))
	}
	if !end.IsZero() {
		params.Set("end", timeParam(end))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	return nil
}

// historyBody is the window, limit and sort an authenticated history endpoint's body takes, each sent only when set
type historyBody struct {
	Start int64  `json:"start,omitempty"`
	End   int64  `json:"end,omitempty"`
	Limit uint64 `json:"limit,omitempty"`
	Sort  int64  `json:"sort,omitempty"`
}

// newHistoryBody returns the body of a window, limit and sort, erroring when the window is reversed or sort is
// neither 1 for oldest first nor -1 for newest first
func newHistoryBody(start, end time.Time, limit uint64, sort int64) (historyBody, error) {
	if err := checkWindow(start, end); err != nil {
		return historyBody{}, err
	}
	if sort != 0 {
		if _, err := sortParam(sort); err != nil {
			return historyBody{}, err
		}
	}
	body := historyBody{Limit: limit, Sort: sort}
	if !start.IsZero() {
		body.Start = start.UnixMilli()
	}
	if !end.IsZero() {
		body.End = end.UnixMilli()
	}
	return body, nil
}
