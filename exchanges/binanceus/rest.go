package binanceus

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/orderbookmanager"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with Binance.US
type Exchange struct {
	exchange.Base
	// orderbookSync synchronises the diff depth streams' books with REST snapshots
	orderbookSync *orderbookmanager.UpdateManager
	// depthUpdateIDs tracks the diff depth events passed to orderbookSync
	depthUpdateIDs depthUpdateIDs
	// orderbookStreamGeneration counts the connections of the market data streams and the order book streams'
	// unsubscriptions, so that orderbookSync discards a snapshot fetched across either
	orderbookStreamGeneration atomic.Uint64
	// wsAPI is the WebSocket API connection requests are sent on
	wsAPI websocketAPIConnection
	// websocketAPIKeyRejected records that the WebSocket API refused a request with the configured account's key, after
	// which the wrapper sends its requests over REST
	websocketAPIKeyRejected atomic.Bool
	// userData tracks the WebSocket API connection's User Data Stream subscriptions
	userData userDataStreams
	// userDataRetry tries a failed User Data Stream subscription again
	userDataRetry userDataRetries
	// balances orders the spot balances User Data Stream events store against those REST polls store
	balances balanceOrdering
}

const (
	// apiURL is Binance.US's REST endpoint. https://api1.binance.us and https://api2.binance.us serve the same API through
	// fewer infrastructure layers, and either can be configured as the RestSpotURL endpoint
	apiURL       = "https://api.binance.us"
	tradeBaseURL = "https://www.binance.us/spot-trade/"

	defaultRecvWindow = 5 * time.Second
)

var (
	errAPIResponse                 = errors.New("Binance.US API error")
	errInvalidParameterCombination = errors.New("invalid combination of optional parameters")
	errUnexpectedResponseData      = errors.New("response carries data but no result was given to decode it into")
)

// SendHTTPRequest sends an unauthenticated request
func (e *Exchange) SendHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	return e.sendRequest(ctx, f, request.UnauthenticatedRequest, result, func() (*request.Item, error) {
		return &request.Item{
			Method:                 http.MethodGet,
			Path:                   endpoint + path,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	})
}

// SendAPIKeyHTTPRequest sends a request carrying the API key without a signature, which USER_STREAM endpoints require
func (e *Exchange) SendAPIKeyHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, f request.EndpointLimit, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	return e.sendRequest(ctx, f, request.AuthenticatedRequest, result, func() (*request.Item, error) {
		return &request.Item{
			Method:                 method,
			Path:                   endpoint + path,
			Headers:                map[string]string{"X-MBX-APIKEY": creds.Key},
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	})
}

// SendAuthHTTPRequest sends a signed request, carrying every parameter in the query string. Binance.US signs the
// percent-encoded query, which is the query sent, so a value needing encoding signs as it is sent. A recvWindow the
// caller set is kept
func (e *Exchange) SendAuthHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, params url.Values, f request.EndpointLimit, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	if params == nil {
		params = url.Values{}
	}
	if !params.Has("recvWindow") {
		params.Set("recvWindow", strconv.FormatInt(defaultRecvWindow.Milliseconds(), 10))
	}
	return e.sendRequest(ctx, f, request.AuthenticatedRequest, result, func() (*request.Item, error) {
		// The timestamp is renewed for every attempt, so a retried request is not rejected as outside recvWindow
		params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
		signature, err := crypto.GetHMAC(crypto.HashSHA256, []byte(params.Encode()), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		return &request.Item{
			Method:                 method,
			Path:                   common.EncodeURLValues(endpoint+path, params) + "&signature=" + hex.EncodeToString(signature),
			Headers:                map[string]string{"X-MBX-APIKEY": creds.Key},
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	})
}

// sendPublicRequest sends a public request over REST to path, or with useWebsocket over the WebSocket API as method
func (e *Exchange) sendPublicRequest(ctx context.Context, useWebsocket bool, path, method string, params url.Values, f request.EndpointLimit, result any) error {
	if useWebsocket {
		return e.SendWebsocketAPIRequest(ctx, f, method, params, result)
	}
	return e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues(path, params), f, result)
}

// sendSignedRequest sends a signed request over REST as an httpMethod request to path, or with useWebsocket over the
// WebSocket API as method
func (e *Exchange) sendSignedRequest(ctx context.Context, useWebsocket bool, httpMethod, path, method string, params url.Values, f request.EndpointLimit, result any) error {
	if useWebsocket {
		return e.SendSignedWebsocketAPIRequest(ctx, f, method, params, result)
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, httpMethod, path, params, f, result)
}

// sendRequest sends a request and decodes its response into result. Binance.US reports errors as {"code":-N,"msg":"..."},
// usually with an HTTP error status but on some endpoints with a 200, and some /sapi endpoints as a success flag of
// false, so a body is checked for both before it is decoded
func (e *Exchange) sendRequest(ctx context.Context, f request.EndpointLimit, auth request.AuthType, result any, newItem func() (*request.Item, error)) error {
	var raw json.RawMessage
	err := e.SendPayload(ctx, f, func() (*request.Item, error) {
		item, err := newItem()
		if err != nil {
			return nil, err
		}
		raw = nil
		item.Result = &raw
		return item, nil
	}, auth)
	if apiErr := checkErrorResponse(raw); apiErr != nil {
		if err != nil {
			return fmt.Errorf("%w: %w", apiErr, err)
		}
		return apiErr
	}
	if err != nil {
		return err
	}
	if result == nil {
		return checkNoData(raw)
	}
	// The request layer rejects an empty body itself, so only a null body is left to catch
	if string(raw) == "null" {
		return common.ErrNoResponse
	}
	return json.Unmarshal(raw, result)
}

// checkNoData returns errUnexpectedResponseData unless a response is empty, null, an empty object or an empty array. A
// request sent without a result is for an endpoint that replies without data, so data a reply does carry is reported
// rather than dropped
func checkNoData(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var value any
	if json.Unmarshal(raw, &value) == nil {
		switch v := value.(type) {
		case nil:
			return nil
		case map[string]any:
			if len(v) == 0 {
				return nil
			}
		case []any:
			if len(v) == 0 {
				return nil
			}
		}
	}
	return fmt.Errorf("%w: %d bytes", errUnexpectedResponseData, len(raw))
}

// checkErrorResponse returns an error when a response body is a Binance.US error object
func checkErrorResponse(raw json.RawMessage) error {
	if len(raw) == 0 || raw[0] != '{' {
		return nil
	}
	// A body that does not decode as an error object is left for the caller's decoding to report
	var resp *ErrorResponse
	if json.Unmarshal(raw, &resp) == nil && resp != nil && (resp.Code.Int64() < 0 || resp.Success != nil && !*resp.Success) {
		msg := resp.Message
		if msg == "" {
			msg = resp.EnvelopeMessage
		}
		return &APIError{Code: resp.Code.Int64(), Message: msg, Data: resp.Data}
	}
	return nil
}

// APIError is an error Binance.US replied with, over REST or the WebSocket API
type APIError struct {
	Code    int64
	Message string
	// Data holds the details some endpoints add, such as the outcome of each half of a cancel-replace
	Data json.RawMessage
}

// Error returns the error's code and message, and its data when it has any
func (a *APIError) Error() string {
	msg := errAPIResponse.Error()
	if mapped, ok := errorCodeToErrorMap[a.Code]; ok {
		msg += ": " + mapped.Error()
	}
	msg += ": code " + strconv.FormatInt(a.Code, 10) + " msg " + a.Message
	if len(a.Data) != 0 {
		msg += " data " + string(a.Data)
	}
	return msg
}

// Unwrap returns errAPIResponse and the package error the code maps to, if any, so callers can match either
func (a *APIError) Unwrap() []error {
	if mapped, ok := errorCodeToErrorMap[a.Code]; ok {
		return []error{errAPIResponse, mapped}
	}
	return []error{errAPIResponse}
}

// errorCodeToErrorMap maps Binance.US error codes to package errors callers can match
var errorCodeToErrorMap = map[int64]error{
	-1002: request.ErrAuthRequestFailed,
	-1121: currency.ErrPairNotFound,
	-2013: order.ErrOrderNotFound,
}

// setTimeRangeParams validates a query's time range and sets its bounds in milliseconds. An unset bound is left out, so
// the endpoint applies its documented default window. Binance.US takes both bounds as inclusive, so equal bounds ask for
// the records of one millisecond; only a reversed window is rejected
func setTimeRangeParams(params url.Values, startTime, endTime time.Time) error {
	if !startTime.IsZero() && !endTime.IsZero() && startTime.After(endTime) {
		return common.ErrStartAfterEnd
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return nil
}

// formatSymbol returns the request symbol of a pair, refusing an empty one
func (e *Exchange) formatSymbol(pair currency.Pair) (string, error) {
	if pair.IsEmpty() {
		return "", currency.ErrCurrencyPairEmpty
	}
	return e.FormatSymbol(pair, asset.Spot)
}

// setSymbolParams sends one pair as symbol and several as symbols, a JSON array, the way the exchange information and
// ticker endpoints take them; no pairs leaves both unset
func (e *Exchange) setSymbolParams(params url.Values, pairs currency.Pairs) error {
	symbols := make([]string, len(pairs))
	for i := range pairs {
		var err error
		if symbols[i], err = e.formatSymbol(pairs[i]); err != nil {
			return err
		}
	}
	switch len(symbols) {
	case 0:
	case 1:
		params.Set("symbol", symbols[0])
	default:
		list, err := json.Marshal(symbols)
		if err != nil {
			return err
		}
		params.Set("symbols", string(list))
	}
	return nil
}

// formatFloat formats a decimal parameter in plain notation, since Binance.US rejects exponents
func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
