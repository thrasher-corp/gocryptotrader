package cryptodotcom

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/cache"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with
// Crypto.com's Exchange API v1, which trades spot, margin and derivatives through one account
type Exchange struct {
	exchange.Base
	// websocketRequestID counts the websocket's request IDs, which its responses echo
	websocketRequestID atomic.Int64
	// websocketConnectDelay is how long a websocket connection waits after dialling before it sends anything
	websocketConnectDelay time.Duration
	// bookResubscribeDelay is how long a book channel that could not be subscribed again for a new snapshot waits before
	// its first retry
	bookResubscribeDelay time.Duration
	// marginOrders holds the orders on spot instruments known to trade on margin, for their trades, which do not say so,
	// to be filled as margin
	marginOrders *cache.LRUCache
}

const (
	// apiURL serves every REST method under apiPath, and uatAPIURL is Crypto.com's UAT sandbox
	apiURL    = "https://api.crypto.com"
	uatAPIURL = "https://uat-api.3ona.co"
	apiPath   = "/exchange/v1/"

	websocketMarketDataURL    = "wss://stream.crypto.com/exchange/v1/market"
	websocketUserURL          = "wss://stream.crypto.com/exchange/v1/user"
	uatWebsocketMarketDataURL = "wss://uat-stream.3ona.co/exchange/v1/market"
	uatWebsocketUserURL       = "wss://uat-stream.3ona.co/exchange/v1/user"

	tradeBaseURL = "https://crypto.com/exchange/trade/"
)

var (
	errInstrumentNameRequired = errors.New("instrument name required")
	errOrderIDRequired        = errors.New("order ID or client order ID required")
	errListIDRequired         = errors.New("list ID required")
	errInvalidQuantity        = errors.New("quantity must be greater than zero")
	errInvalidPrice           = errors.New("price must be greater than zero")
	errInvalidAmount          = errors.New("amount must be greater than zero")
	errInvalidSide            = errors.New("invalid order side")
	errInvalidOrderType       = errors.New("invalid order type")
	errInvalidTimeInForce     = errors.New("invalid time in force")
	errInvalidCount           = errors.New("invalid count")
	errInvalidLeverage        = errors.New("invalid leverage")
	errAccountIDRequired      = errors.New("account ID required")
	errIsolationIDRequired    = errors.New("isolation ID required")
	errAddressRequired        = errors.New("address required")
	errPaymentNetworkRequired = errors.New("payment network required")
	errOrderListEmpty         = errors.New("order list empty")
	errOrderListTooLong       = errors.New("order list too long")
	errTimeBeforeUnixEpoch    = errors.New("time before the Unix epoch")
)

// Response codes of a request Crypto.com refuses as a whole, whatever it asks
const (
	codeUnauthorised    = 40101
	codeInvalidNonce    = 40102
	codeIPIllegal       = 40103
	codeUserTierInvalid = 40104
	codeRequestTimeout  = 40801
	codeTooManyRequests = 42901
	codeInternalError   = 50001
)

// APIError is an error Crypto.com reports in a response's code and message. The original request Crypto.com echoes
// with some errors is left out, as it carries the API key and signature
type APIError struct {
	Code    int64
	Message string
}

// Error returns the error's code and message
func (a *APIError) Error() string {
	if a.Message == "" {
		return strconv.FormatInt(a.Code, 10)
	}
	return strconv.FormatInt(a.Code, 10) + " " + a.Message
}

// apiResponse is the envelope of every response, whose code is 0 when the request succeeded; Crypto.com sends the
// code as a JSON number although its docs show a string, so either is decoded
type apiResponse struct {
	Code    types.Number `json:"code"`
	Message string       `json:"message"`
	Result  any          `json:"result"`
}

// err returns the error a response reports, or nil when it reports none
func (r *apiResponse) err() error {
	if r.Code == 0 {
		return nil
	}
	return &APIError{Code: r.Code.Int64(), Message: r.Message}
}

// dataResponse is a result holding only a list, which methods return by itself
type dataResponse[T any] struct {
	Data []T `json:"data"`
}

// requestBody is the body of a POST request; a private method's also carries the API key and the signature
type requestBody struct {
	ID        uint64          `json:"id"`
	Method    string          `json:"method"`
	Params    json.RawMessage `json:"params"`
	Nonce     uint64          `json:"nonce"`
	APIKey    string          `json:"api_key,omitempty"`
	Signature string          `json:"sig,omitempty"`
}

// SendHTTPRequest sends a public method's GET request with params as its query string, and decodes its result
func (e *Exchange) SendHTTPRequest(ctx context.Context, epl request.EndpointLimit, method string, params url.Values, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(exchange.RestSpot)
	if err != nil {
		return err
	}
	path := common.EncodeURLValues(endpoint+apiPath+method, params)
	resp := &apiResponse{Result: result}
	err = e.SendPayload(ctx, epl, func() (*request.Item, error) {
		return &request.Item{
			Method:                 http.MethodGet,
			Path:                   path,
			Result:                 resp,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.UnauthenticatedRequest)
	return responseError(resp, err)
}

// SendAuthenticatedHTTPRequest sends a private method's POST request, signed with the configured credentials, with
// params, which may be nil, as its params object, and decodes its result
func (e *Exchange) SendAuthenticatedHTTPRequest(ctx context.Context, epl request.EndpointLimit, method string, params, result any) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	return e.sendPostRequest(ctx, epl, method, params, result, creds)
}

// sendPostRequest sends a POST request with params as its params object, signed when credentials are given, and
// decodes its result
func (e *Exchange) sendPostRequest(ctx context.Context, epl request.EndpointLimit, method string, params, result any, creds *accounts.Credentials) error {
	endpoint, err := e.API.Endpoints.GetURL(exchange.RestSpot)
	if err != nil {
		return err
	}
	encodedParams, err := encodeParams(params)
	if err != nil {
		return err
	}
	signedParams, err := paramString(encodedParams)
	if err != nil {
		return err
	}
	requestType := request.AuthType(request.UnauthenticatedRequest)
	if creds != nil {
		requestType = request.AuthenticatedRequest
	}
	resp := &apiResponse{Result: result}
	err = e.SendPayload(ctx, epl, func() (*request.Item, error) {
		body := requestBody{
			// The ID only has to be unique to tell responses apart, so the requester's nonce, which takes its timed lock,
			// serves; the nonce Crypto.com checks is the time in milliseconds
			ID:     uint64(e.Requester.GetNonce(nonce.UnixNano)), //nolint:gosec // A nanosecond nonce is positive
			Method: method,
			Params: encodedParams,
			Nonce:  requestNonce(),
		}
		if creds != nil {
			body.APIKey = creds.Key
			signature, err := signRequest(creds.Secret, method, body.ID, creds.Key, signedParams, body.Nonce)
			if err != nil {
				return nil, err
			}
			body.Signature = signature
		}
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		return &request.Item{
			Method:                 http.MethodPost,
			Path:                   endpoint + apiPath + method,
			Headers:                map[string]string{"Content-Type": "application/json"},
			Body:                   bytes.NewReader(payload),
			Result:                 resp,
			NonceEnabled:           true,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, requestType)
	return responseError(resp, err)
}

// responseError returns the error a response reports, else the request's. Crypto.com answers a failed request with
// an error status as well as its code, which explains the failure better; the requester marks a failed authenticated
// request, which callers tell authentication failures by, so that mark is kept
func responseError(resp *apiResponse, requestErr error) error {
	if err := resp.err(); err != nil {
		if errors.Is(requestErr, request.ErrAuthRequestFailed) {
			return fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, err)
		}
		return err
	}
	return requestErr
}

// encodeParams returns params as the JSON object a request sends, an empty object when params is nil, as Crypto.com
// wants params on every request
func encodeParams(params any) (json.RawMessage, error) {
	if params == nil {
		return json.RawMessage("{}"), nil
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	if string(encoded) == "null" {
		return json.RawMessage("{}"), nil
	}
	return encoded, nil
}

// requestNonce returns a request's nonce, the current time in milliseconds, which Crypto.com checks is within a minute
// of its own clock
func requestNonce() uint64 {
	return uint64(time.Now().UnixMilli())
}

// signRequest returns a request's signature: the hex HMAC-SHA256 of its method, ID, API key, params string and nonce,
// keyed by the API secret
func signRequest(secret, method string, id uint64, apiKey, params string, requestNonce uint64) (string, error) {
	payload := method + strconv.FormatUint(id, 10) + apiKey + params + strconv.FormatUint(requestNonce, 10)
	signature, err := crypto.GetHMAC(crypto.HashSHA256, []byte(payload), []byte(secret))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(signature), nil
}

// paramString returns the params string a signature covers, as the docs' JavaScript example builds it: an object's keys
// in ascending order, each followed by its value, an array's elements one after another and null as "null"
func paramString(params []byte) (string, error) {
	var sb strings.Builder
	if err := appendParamString(&sb, params, jsonparser.Object); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// appendParamString appends the params string of a JSON value of the given type
func appendParamString(sb *strings.Builder, value []byte, dataType jsonparser.ValueType) error {
	switch dataType {
	case jsonparser.Object:
		type member struct {
			key       string
			value     []byte
			valueType jsonparser.ValueType
		}
		var members []member
		if err := jsonparser.ObjectEach(value, func(key, v []byte, vt jsonparser.ValueType, _ int) error {
			k, err := jsonparser.ParseString(key)
			if err != nil {
				return err
			}
			members = append(members, member{key: k, value: v, valueType: vt})
			return nil
		}); err != nil {
			return err
		}
		slices.SortFunc(members, func(a, b member) int { return strings.Compare(a.key, b.key) })
		for i := range members {
			sb.WriteString(members[i].key)
			if err := appendParamString(sb, members[i].value, members[i].valueType); err != nil {
				return err
			}
		}
	case jsonparser.Array:
		var elementErr error
		if _, err := jsonparser.ArrayEach(value, func(v []byte, vt jsonparser.ValueType, _ int, _ error) {
			if elementErr == nil {
				elementErr = appendParamString(sb, v, vt)
			}
		}); err != nil {
			return err
		}
		return elementErr
	case jsonparser.String:
		s, err := jsonparser.ParseString(value)
		if err != nil {
			return err
		}
		sb.WriteString(s)
	case jsonparser.Null:
		sb.WriteString("null")
	default:
		sb.Write(value)
	}
	return nil
}

// unixMilliParam returns a time as the Unix milliseconds a request param takes, zero, which is left unsent, for the
// zero time, and an error for a time before 1970
func unixMilliParam(t time.Time) (uint64, error) {
	if t.IsZero() {
		return 0, nil
	}
	ms := t.UnixMilli()
	if ms < 0 {
		return 0, fmt.Errorf("%w: %s", errTimeBeforeUnixEpoch, t)
	}
	return uint64(ms), nil
}

// unixNanoParam returns a time as the Unix nanoseconds a request param takes, zero, which is left unsent, for the zero
// time, and an error for a time before 1970; Crypto.com recommends nanoseconds for paging history accurately
func unixNanoParam(t time.Time) (uint64, error) {
	if t.IsZero() {
		return 0, nil
	}
	ns := t.UnixNano()
	if ns < 0 {
		return 0, fmt.Errorf("%w: %s", errTimeBeforeUnixEpoch, t)
	}
	return uint64(ns), nil
}
