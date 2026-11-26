package bingx

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Exchange implements exchange.IBotExchange and contains additional specific API methods for interacting with BingX
type Exchange struct {
	exchange.Base

	wsState websocketState
	// positionModes and coinMarginedPositionModes cache each account's USDT-M and coin-M position modes, which every
	// perpetual order's position side follows
	positionModes             positionModeCache
	coinMarginedPositionModes positionModeCache
	// orderModifications serialises the modifications of each order
	orderModifications orderModificationLocks
	// balances orders the stream's balance writes against REST balance snapshots and runs the refreshes perpetual
	// balance events ask for
	balances balanceState
}

const (
	apiURL                   = "https://open-api.bingx.com"
	spotWebsocketURL         = "wss://open-api-ws.bingx.com/market"
	usdtMarginedWebsocketURL = "wss://open-api-swap.bingx.com/swap-market"
	coinMarginedWebsocketURL = "wss://open-api-cswap-ws.bingx.com/market"
	tradeBaseURL             = "https://bingx.com/en/"

	keyHeader = "X-BX-APIKEY"

	defaultRecvWindow = 5 * time.Second

	jsonNull = "null"
)

var (
	errAPIResponse          = errors.New("BingX API error")
	errRequestNotSent       = errors.New("request not sent")
	errUnexpectedEnvelope   = errors.New("unexpected response envelope")
	errSigningPayloadObject = errors.New("signed request body must encode as a JSON object")
	errPositionSideRequired = errors.New("position side required")
	errNullData             = errors.New("null data")
)

// GetSystemTime returns the server time
func (e *Exchange) GetSystemTime(ctx context.Context) (*SystemTimeResponse, error) {
	var resp *SystemTimeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, "/openApi/swap/v2/server/time", systemTimeRate, &resp)
}

// SendHTTPRequest sends an unauthenticated GET request to a public endpoint
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

// SendTimestampedHTTPRequest sends an unsigned GET request to a public endpoint that refuses requests without a
// timestamp
func (e *Exchange) SendTimestampedHTTPRequest(ctx context.Context, ePath exchange.URL, path string, params url.Values, f request.EndpointLimit, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	params = cloneParams(params)
	return e.sendRequest(ctx, f, request.UnauthenticatedRequest, result, func() (*request.Item, error) {
		// Renewed for every attempt, so a retried request is not rejected as stale
		params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
		return &request.Item{
			Method:                 http.MethodGet,
			Path:                   common.EncodeURLValues(endpoint+path, params),
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	})
}

// SendAPIKeyHTTPRequest sends a request carrying the API key without a signature, as the documented listen key requests
// do
func (e *Exchange) SendAPIKeyHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, params url.Values, f request.EndpointLimit, result any) error {
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
			Path:                   common.EncodeURLValues(endpoint+path, params),
			Headers:                map[string]string{keyHeader: creds.Key},
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	})
}

// SendAuthHTTPRequest sends a signed request carrying every parameter in the query string, the transport BingX's
// documentation uses for every method apart from its JSON body endpoints. A request without recvWindow is sent with
// the default 5 second window
func (e *Exchange) SendAuthHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, params url.Values, f request.EndpointLimit, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	params = cloneParams(params)
	if !params.Has("recvWindow") {
		params.Set("recvWindow", strconv.FormatInt(defaultRecvWindow.Milliseconds(), 10))
	}
	return e.sendRequest(ctx, f, request.AuthenticatedRequest, result, func() (*request.Item, error) {
		// The timestamp is renewed for every attempt, so a retried request is not rejected as outside recvWindow
		params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
		return &request.Item{
			Method:                 method,
			Path:                   endpoint + path + "?" + signQuery(params, creds.Secret),
			Headers:                map[string]string{keyHeader: creds.Key},
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	})
}

// SendAuthJSONBodyHTTPRequest sends a signed request carrying its parameters, recvWindow, timestamp and signature in a
// JSON body, which the sub-account management, internal transfer and signal endpoints require. body must encode as a
// JSON object, or be nil
func (e *Exchange) SendAuthJSONBodyHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, body any, f request.EndpointLimit, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	fields, err := jsonBodyFields(body)
	if err != nil {
		return err
	}
	if _, ok := fields["recvWindow"]; !ok {
		fields["recvWindow"] = json.RawMessage(strconv.FormatInt(defaultRecvWindow.Milliseconds(), 10))
	}
	return e.sendRequest(ctx, f, request.AuthenticatedRequest, result, func() (*request.Item, error) {
		fields["timestamp"] = json.RawMessage(strconv.FormatInt(time.Now().UnixMilli(), 10))
		payload, err := signJSONBody(fields, creds.Secret)
		if err != nil {
			return nil, err
		}
		return &request.Item{
			Method: method,
			Path:   endpoint + path,
			Headers: map[string]string{
				keyHeader:      creds.Key,
				"Content-Type": "application/json",
			},
			Body:                   bytes.NewReader(payload),
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	})
}

// sendRequest sends a request and decodes its response into result. BingX reports errors as a non-zero code beside msg,
// on an HTTP 200 or an error status, so the body is checked whichever status it came with. A failure before the request
// was built while ctx is live, such as the rate limiter refusing a wait past ctx's deadline, returns errRequestNotSent,
// as nothing was sent; the error of an ended ctx is returned as it is, so callers matching it find it alone
func (e *Exchange) sendRequest(ctx context.Context, f request.EndpointLimit, auth request.AuthType, result any, newItem func() (*request.Item, error)) error {
	var raw json.RawMessage
	var built bool
	err := e.SendPayload(ctx, f, func() (*request.Item, error) {
		raw = nil
		item, err := newItem()
		if err != nil {
			return nil, err
		}
		built = true
		item.Result = &raw
		return item, nil
	}, auth)
	if err != nil && !built && ctx.Err() == nil {
		return fmt.Errorf("%w: %w", errRequestNotSent, err)
	}
	if apiErr := checkErrorResponse(raw); apiErr != nil {
		if err != nil {
			return fmt.Errorf("%w: %w", apiErr, err)
		}
		return apiErr
	}
	if err != nil {
		return err
	}
	return decodeResponse(raw, result)
}

// cloneParams returns a copy of params a request can add its defaults, timestamp and signature to, so the caller's
// values are left as they were
func cloneParams(params url.Values) url.Values {
	if params == nil {
		return url.Values{}
	}
	return maps.Clone(params)
}

// signingString returns the string BingX signs: every parameter, timestamp included, sorted by key with its value
// unencoded
func signingString(params url.Values) string {
	var sb strings.Builder
	for _, k := range slices.Sorted(maps.Keys(params)) {
		if k == "signature" {
			continue
		}
		for _, v := range params[k] {
			if sb.Len() > 0 {
				sb.WriteByte('&')
			}
			sb.WriteString(k)
			sb.WriteByte('=')
			sb.WriteString(v)
		}
	}
	return sb.String()
}

// encodedQuery returns the parameters in signing order with their values percent-encoded, spaces as %20. BingX decodes
// values before checking the signature over them, so encoding every value keeps reserved characters from splitting
// the query
func encodedQuery(params url.Values) string {
	var sb strings.Builder
	for _, k := range slices.Sorted(maps.Keys(params)) {
		if k == "signature" {
			continue
		}
		for _, v := range params[k] {
			if sb.Len() > 0 {
				sb.WriteByte('&')
			}
			sb.WriteString(k)
			sb.WriteByte('=')
			sb.WriteString(strings.ReplaceAll(url.QueryEscape(v), "+", "%20"))
		}
	}
	return sb.String()
}

// signQuery returns the query string of a signed request: the parameters, then the signature over them
func signQuery(params url.Values, secret string) string {
	return encodedQuery(params) + "&signature=" + sign(signingString(params), secret)
}

// sign returns the lowercase hex HMAC-SHA256 of payload keyed with secret
func sign(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload)) // hash.Hash writes never return an error
	return hex.EncodeToString(mac.Sum(nil))
}

// jsonBodyFields returns the members of a request body's JSON object
func jsonBodyFields(body any) (map[string]json.RawMessage, error) {
	fields := make(map[string]json.RawMessage)
	if body == nil {
		return fields, nil
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	if string(encoded) == jsonNull {
		return fields, nil
	}
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, fmt.Errorf("%w: %w", errSigningPayloadObject, err)
	}
	return fields, nil
}

// signJSONBody returns a JSON body holding fields and the signature over them. BingX signs a JSON body as it signs a
// query: the members sorted by key, string values unquoted and other values as their compact JSON
func signJSONBody(fields map[string]json.RawMessage, secret string) ([]byte, error) {
	var sb strings.Builder
	for _, k := range slices.Sorted(maps.Keys(fields)) {
		if k == "signature" {
			continue
		}
		value, err := signingValue(fields[k])
		if err != nil {
			return nil, fmt.Errorf("%w for %q", err, k)
		}
		if sb.Len() > 0 {
			sb.WriteByte('&')
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(value)
	}
	out := maps.Clone(fields)
	out["signature"] = json.RawMessage(strconv.Quote(sign(sb.String(), secret)))
	return json.Marshal(out)
}

// signingValue returns a JSON value as BingX writes it into a signing string: a string unquoted, anything else as the
// compact JSON json.Marshal produced for it
func signingValue(raw json.RawMessage) (string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) > 0 && trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return "", err
		}
		return s, nil
	}
	return string(trimmed), nil
}

// checkErrorResponse returns an error when a response body is a BingX error: an object with a non-zero code
func checkErrorResponse(raw json.RawMessage) error {
	code, ok := responseCode(raw)
	if !ok || code == 0 {
		return nil
	}
	apiErr := &APIError{Code: code}
	apiErr.Message, _ = jsonparser.GetString(raw, "msg")
	if data, dataType, _, err := jsonparser.Get(raw, "data"); err == nil && dataType != jsonparser.Null && !isEmptyJSON(data) {
		apiErr.Data = slices.Clone(data)
	}
	return apiErr
}

// responseCode returns the code of a response object, reporting false for a body without an integer code, which a
// response that is not an error object or not enveloped leaves for its decoding to handle
func responseCode(raw json.RawMessage) (int64, bool) {
	if len(raw) == 0 || raw[0] != '{' {
		return 0, false
	}
	value, dataType, _, err := jsonparser.Get(raw, "code")
	if err != nil {
		return 0, false
	}
	switch dataType {
	case jsonparser.Number:
		code, err := jsonparser.ParseInt(value)
		return code, err == nil
	case jsonparser.String:
		code, err := strconv.ParseInt(string(value), 10, 64)
		return code, err == nil
	default:
		return 0, false
	}
}

// badStatusCode returns the HTTP status of a reply the requester returned request.ErrBadStatus for, which it reports
// only in the error's text, right after the sentinel's own
func badStatusCode(err error) (int, bool) {
	if !errors.Is(err, request.ErrBadStatus) {
		return 0, false
	}
	_, after, ok := strings.Cut(err.Error(), request.ErrBadStatus.Error()+": ")
	if !ok {
		return 0, false
	}
	digits, _, _ := strings.Cut(after, " ")
	status, err := strconv.Atoi(digits)
	return status, err == nil
}

// isEmptyJSON reports whether a JSON value is an empty object or array, which BingX adds to many errors
func isEmptyJSON(value []byte) bool {
	trimmed := string(bytes.TrimSpace(value))
	return trimmed == "{}" || trimmed == "[]" || trimmed == `""`
}

// decodeResponse decodes a successful response into result: the data member of an enveloped response, or the whole
// body of the endpoints that reply without the envelope, such as the transfer and deposit records
func decodeResponse(raw json.RawMessage, result any) error {
	if result == nil {
		return nil
	}
	if len(raw) == 0 || string(raw) == jsonNull {
		return common.ErrNoResponse
	}
	if raw[0] == '{' {
		data, dataType, _, err := jsonparser.Get(raw, "data")
		switch {
		case err == nil && dataType == jsonparser.Null:
			return fmt.Errorf("%w: %w", common.ErrNoResponse, errNullData)
		case err == nil:
			return json.Unmarshal(data, result)
		case !errors.Is(err, jsonparser.KeyPathNotFoundError):
			return fmt.Errorf("%w: %w", errUnexpectedEnvelope, err)
		}
	}
	return json.Unmarshal(raw, result)
}

// APIError is an error BingX replied with
type APIError struct {
	Code    int64
	Message string
	// Data holds the details some errors carry, such as the outcome of each half of a cancel-replace
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

// outcomeUnknown reports whether the error's code is one BingX documents as a busy, internal or network failure, which
// leaves unknown whether the request was carried out: 100500 system busy, 109500 internal server error, 109701 network
// request exception and 110500 order system busy
func (a *APIError) outcomeUnknown() bool {
	switch a.Code {
	case 100500, 109500, 109701, 110500:
		return true
	default:
		return false
	}
}

// errorCodeToErrorMap maps the BingX error codes that mean the same on every product to package errors callers can
// match. Codes such as 100404 and 100421 mean different things on spot and elsewhere, so the methods they apply to map
// them instead
var errorCodeToErrorMap = map[int64]error{
	100001: request.ErrAuthRequestFailed,
	100412: request.ErrAuthRequestFailed,
	100413: request.ErrAuthRequestFailed,
	100419: request.ErrAuthRequestFailed,
	109421: order.ErrOrderNotFound,
	109425: currency.ErrPairNotFound,
}

var (
	errUnsupportedInterval = errors.New("unsupported interval")
	errInvalidTimeRange    = errors.New("invalid time range")
)

// intervalToString returns the interval parameter the REST candle endpoints and the perpetual candle streams take
func intervalToString(interval kline.Interval) (string, error) {
	switch interval {
	case kline.OneMin:
		return "1m", nil
	case kline.ThreeMin:
		return "3m", nil
	case kline.FiveMin:
		return "5m", nil
	case kline.FifteenMin:
		return "15m", nil
	case kline.ThirtyMin:
		return "30m", nil
	case kline.OneHour:
		return "1h", nil
	case kline.TwoHour:
		return "2h", nil
	case kline.FourHour:
		return "4h", nil
	case kline.SixHour:
		return "6h", nil
	case kline.EightHour:
		return "8h", nil
	case kline.TwelveHour:
		return "12h", nil
	case kline.OneDay:
		return "1d", nil
	case kline.ThreeDay:
		return "3d", nil
	case kline.OneWeek:
		return "1w", nil
	case kline.OneMonth:
		return "1M", nil
	}
	return "", fmt.Errorf("%w: %s", errUnsupportedInterval, interval)
}

// spotWebsocketIntervalToString returns the interval the spot candle streams take, which spell intervals out
func spotWebsocketIntervalToString(interval kline.Interval) (string, error) {
	switch interval {
	case kline.OneMin:
		return "1min", nil
	case kline.ThreeMin:
		return "3min", nil
	case kline.FiveMin:
		return "5min", nil
	case kline.FifteenMin:
		return "15min", nil
	case kline.ThirtyMin:
		return "30min", nil
	case kline.OneHour:
		return "60min", nil
	case kline.TwoHour:
		return "2hour", nil
	case kline.FourHour:
		return "4hour", nil
	case kline.SixHour:
		return "6hour", nil
	case kline.EightHour:
		return "8hour", nil
	case kline.TwelveHour:
		return "12hour", nil
	case kline.OneDay:
		return "1day", nil
	case kline.ThreeDay:
		return "3day", nil
	case kline.OneWeek:
		return "1week", nil
	case kline.OneMonth:
		return "1mon", nil
	}
	return "", fmt.Errorf("%w: %s", errUnsupportedInterval, interval)
}

// setTimeRange validates a query's time range and sets its bounds in milliseconds under the keys the endpoint uses.
// An unset bound is left out, so the endpoint applies its documented default
func setTimeRange(params url.Values, startKey, endKey string, start, end time.Time) error {
	if !start.IsZero() && !end.IsZero() && start.After(end) {
		return fmt.Errorf("%w: %w", errInvalidTimeRange, common.ErrStartAfterEnd)
	}
	if !start.IsZero() {
		params.Set(startKey, strconv.FormatInt(start.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set(endKey, strconv.FormatInt(end.UnixMilli(), 10))
	}
	return nil
}
