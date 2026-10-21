package bitstamp

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"uuid"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	bitstampAPIURL = "https://www.bitstamp.net/api"
	tradeBaseURL   = "https://www.bitstamp.net/trade/"

	bitstampRateInterval = time.Minute * 10
	bitstampRequestRate  = 10000

	maxWithdrawalRequestsTimeDelta = 50000000 * time.Second
	maxResultLimit                 = 1000
	maxResultOffset                = 200000
	maxFundingRateHistoryLimit     = 100
	maxClientOrderIDLength         = 180
	maxTrailingDelta               = 2000
	minLeverage                    = 1
	maxLeverage                    = 25

	formContentType = "application/x-www-form-urlencoded"
	jsonContentType = "application/json"
	trueValue       = "True"
	falseValue      = "False"
)

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with Bitstamp
type Exchange struct {
	exchange.Base

	// websocketUserID is the user ID of the last websocket token, which names the private channels
	websocketUserID atomic.Uint64
	// orderEventCheckpoints holds the last private order event handled for each market and order source
	orderEventCheckpoints map[string]orderEventCheckpoint
	// orderEventsKept holds the received private order events which have not been relayed, in event order, for each
	// market and order source
	orderEventsKept map[string][]keptOrderEvent
	orderEventMu    sync.Mutex
	// orderTotalFetches tracks the fetches of deleted and replaced orders' totals, which run off the websocket's read loop
	// and its wait group, so a reconnection does not wait for them
	orderTotalFetches sync.WaitGroup
	// orderTotalFetchMu runs the fetches one at a time, so that a burst of them holds at most one place in the request
	// rate limiter ahead of order placements, cancellations and the read loop's own recovery requests
	orderTotalFetchMu sync.Mutex
}

var (
	errAPIResponse              = errors.New("API returned an error")
	errNoPermission             = errors.New("API key lacks permission")
	errInvalidEventIDRange      = errors.New("invalid event ID range")
	errMalformedOrderbookLevel  = errors.New("malformed orderbook level")
	errMalformedTransaction     = errors.New("malformed transaction")
	errNullNumber               = errors.New("number is null")
	errInvalidOrderbookGrouping = errors.New("invalid orderbook grouping")
	errInvalidTransactionPeriod = errors.New("invalid transaction time period")
	errInvalidInterval          = errors.New("invalid interval")
	errInvalidLimit             = errors.New("invalid limit")
	errInvalidOffset            = errors.New("invalid offset")
	errInvalidTimeDelta         = errors.New("time delta must be between 1 and 50000000 seconds")
	errInvalidDateTime          = errors.New("invalid date time")
	errNoResponse               = errors.New("no response data")
	errOrderSourceRequired      = errors.New("order source is required")
	errNetworkRequired          = errors.New("network is required")
	errWithdrawalIDRequired     = errors.New("withdrawal ID is required")
	errSubAccountRequired       = errors.New("sub account is required")
	errDepositIDRequired        = errors.New("deposit ID is required")
	errContactIDRequired        = errors.New("contact ID is required")
	errSatoshiTestIDRequired    = errors.New("satoshi test ID is required")
	errRegistrationIDRequired   = errors.New("registration ID is required")
	errPublicKeyRequired        = errors.New("extended public key is required")
	errContactInfoRequired      = errors.New("retail or corporate contact information is required")
	errEarnTypeRequired         = errors.New("earn type is required")
	errEarnTermRequired         = errors.New("earn term is required")
	errEarnSettingRequired      = errors.New("earn setting is required")
	errMarginModeRequired       = errors.New("margin mode is required")
	errLeverageRequired         = errors.New("leverage is required")
	errOrderTypeRequired        = errors.New("order type is required")
	errPositionIDRequired       = errors.New("position ID is required")
	errPairOrOrderIDRequired    = errors.New("currency pair or order ID is required")
	errCollateralRequired       = errors.New("exactly one of target collateral or collateral deltas is required")
	errBankDetailsRequired      = errors.New("bank account details are required")
	errContactNameRequired      = errors.New("retail contact first and last names are required")
	errCompanyNameRequired      = errors.New("corporate contact company name is required")
	errIdentificationRequired   = errors.New("contact identification is required")
	errBeneficiaryRequired      = errors.New("beneficiary ID or details are required for a third party withdrawal")
	errOriginatorRequired       = errors.New("originator ID or details are required for a third party deposit")
	errContactUUIDRequired      = errors.New("contact UUID is required for a third party address")
	errOrderFlagsConflict       = errors.New("only one of the daily, immediate or cancel, fill or kill, maker or cancel and good till date flags may be set")
	errLimitPriceConflict       = errors.New("limit price cannot be combined with an order flag")
	errInvalidPrice             = errors.New("price must not be negative")
	errInvalidOrderSubtype      = errors.New("invalid order subtype")
	errInvalidTrailingDelta     = errors.New("trailing delta must be between 1 and 2000 basis points")
	errTrailingDeltaRequired    = errors.New("trailing delta is required with an activation price")
	errClientOrderIDTooLong     = errors.New("client order ID exceeds 180 characters")
	errReduceOnlyUnsupported    = errors.New("reduce only is not supported with these order parameters")
	errInvalidLeverage          = errors.New("leverage must be between 1 and 25")
	errInvalidCollateral        = errors.New("collateral amounts must not be negative")
)

// SendHTTPRequest sends an unauthenticated HTTP GET request
func (e *Exchange) SendHTTPRequest(ctx context.Context, ep exchange.URL, path string, params url.Values, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	var interim json.RawMessage
	err = e.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return &request.Item{
			Method:                 http.MethodGet,
			Path:                   common.EncodeURLValues(endpoint+path, params),
			Result:                 &interim,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.UnauthenticatedRequest)
	return processResponse(interim, err, result, false)
}

// SendAuthenticatedHTTPRequest sends an HTTP request signed with Bitstamp's v2 authentication scheme
// params are sent in the query string. body is form encoded when it is url.Values and JSON encoded otherwise; a nil or
// empty body is sent without a Content-Type header, as the API rejects requests with an empty body and Content-Type
func (e *Exchange) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method, path string, params url.Values, body, result any) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	target, err := url.Parse(common.EncodeURLValues(endpoint+path, params))
	if err != nil {
		return err
	}

	var payload []byte
	var contentType string
	switch b := body.(type) {
	case nil:
	case url.Values:
		if len(b) > 0 {
			payload = []byte(b.Encode())
			contentType = formContentType
		}
	default:
		if payload, err = json.Marshal(b); err != nil {
			return err
		}
		contentType = jsonContentType
	}

	// The query component of the signature includes its leading question mark, as the URL is built by concatenating it
	var query string
	if target.RawQuery != "" {
		query = "?" + target.RawQuery
	}

	var interim json.RawMessage
	err = e.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		nonce := uuid.NewV4().String()
		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		authHeader := "BITSTAMP " + creds.Key
		message := authHeader + method + target.Host + target.Path + query + contentType + nonce + timestamp + "v2" + string(payload)
		signature, err := crypto.GetHMAC(crypto.HashSHA256, []byte(message), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		headers := map[string]string{
			"X-Auth":           authHeader,
			"X-Auth-Signature": hex.EncodeToString(signature),
			"X-Auth-Nonce":     nonce,
			"X-Auth-Timestamp": timestamp,
			"X-Auth-Version":   "v2",
		}
		if contentType != "" {
			headers["Content-Type"] = contentType
		}
		if creds.SubAccount != "" {
			headers["X-Auth-Subaccount-Id"] = creds.SubAccount
		}
		var reader io.Reader
		if len(payload) > 0 {
			reader = bytes.NewReader(payload)
		}
		return &request.Item{
			Method:                 method,
			Path:                   target.String(),
			Headers:                headers,
			Body:                   reader,
			Result:                 &interim,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.AuthenticatedRequest)
	return processResponse(interim, err, result, true)
}

// processResponse returns any error reported in a response body before decoding the body into result
// A successful response whose body cannot be decoded is an error, as only the 204 responses the API documents for calls
// without a result carry no body, and those are not decoded
func processResponse(data json.RawMessage, requestErr error, result any, authenticated bool) error {
	if apiErr := checkResponseError(data, requestErr != nil); apiErr != nil {
		if authenticated {
			return fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, apiErr)
		}
		return apiErr
	}
	data = bytes.TrimSpace(data)
	empty := len(data) == 0 || bytes.Equal(data, []byte("null"))
	if requestErr != nil {
		return requestErr
	}
	if result == nil {
		return nil
	}
	if empty {
		// A missing body would otherwise leave a pointer result nil for callers to dereference
		if v := reflect.ValueOf(result); v.Kind() == reflect.Pointer && v.Elem().Kind() == reflect.Pointer {
			return errNoResponse
		}
		return nil
	}
	return json.Unmarshal(data, result)
}

// checkResponseError returns an error describing an error response body
// Successful responses may contain status, code and message fields, so they are only treated as errors when they match
// a documented error shape, such as the 400.043 response code which closing positions returns with HTTP 200, whereas any
// recognised field is reported for responses with an unsuccessful HTTP status
func checkResponseError(data json.RawMessage, unsuccessfulStatus bool) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return nil
	}
	var resp errorResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil //nolint:nilerr // Malformed bodies are reported by the caller's decode or by the request error
	}
	code := formatErrorDetail(resp.Code)
	message := formatErrorDetail(resp.Message)
	errorDetail := formatErrorDetail(resp.Error)
	responseCode := formatErrorDetail(resp.ResponseCode)
	if !unsuccessfulStatus && formatErrorDetail(resp.Status) != "error" && errorDetail == "" && (code == "" || message == "") && formatErrorDetail(resp.Errors) == "" && !isErrorResponseCode(responseCode) {
		return nil
	}

	var codes, details []string
	for _, c := range []string{code, responseCode} {
		if c != "" {
			codes = append(codes, c)
		}
	}
	for _, d := range []string{formatErrorDetail(resp.Reason), errorDetail, message, formatErrorDetail(resp.ResponseExplanation), formatErrorDetail(resp.Errors)} {
		if d != "" {
			details = append(details, d)
		}
	}
	if field := formatErrorDetail(resp.Field); field != "" {
		details = append(details, "field: "+field)
	}
	if orderID := formatErrorDetail(resp.OrderID); orderID != "" {
		details = append(details, "order ID: "+orderID)
	}
	apiErr := errAPIResponse
	switch {
	case code == "API0003":
		// The API documents API0003 as an API key without permission for the endpoint, which retrying cannot fix
		apiErr = fmt.Errorf("%w: %w", errAPIResponse, errNoPermission)
	case responseCode == "400.085":
		// The API documents 400.085 as an event ID range whose until ID is not after its since ID
		apiErr = fmt.Errorf("%w: %w", errAPIResponse, errInvalidEventIDRange)
	case responseCode == "400.043":
		// The API documents 400.043 as there being no positions to close
		apiErr = fmt.Errorf("%w: %w", errAPIResponse, futures.ErrNoPositionsFound)
	}
	switch {
	case len(codes) == 0 && len(details) == 0:
		if unsuccessfulStatus {
			return nil
		}
		return fmt.Errorf("%w: %s", errAPIResponse, data)
	case len(codes) == 0:
		return fmt.Errorf("%w: %s", errAPIResponse, strings.Join(details, "; "))
	case len(details) == 0:
		return fmt.Errorf("%w: %s", apiErr, strings.Join(codes, " "))
	default:
		return fmt.Errorf("%w: %s %s", apiErr, strings.Join(codes, " "), strings.Join(details, "; "))
	}
}

// isErrorResponseCode returns whether a response code reports a failed request, as the API's codes of the form 400.043
// and 500.016 begin with the HTTP status class of the failure
func isErrorResponseCode(responseCode string) bool {
	class, _, ok := strings.Cut(responseCode, ".")
	return ok && len(class) == 3 && (class[0] == '4' || class[0] == '5')
}

// formatErrorDetail flattens an error detail which may be a string, a number, a list or an object of field errors
func formatErrorDetail(data json.RawMessage) string {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return ""
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		return s
	}
	var list []json.RawMessage
	if err := json.Unmarshal(data, &list); err == nil {
		details := make([]string, 0, len(list))
		for _, v := range list {
			if d := formatErrorDetail(v); d != "" {
				details = append(details, d)
			}
		}
		return strings.Join(details, ", ")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err == nil {
		details := make([]string, 0, len(fields))
		for _, k := range slices.Sorted(maps.Keys(fields)) {
			d := formatErrorDetail(fields[k])
			if d == "" {
				continue
			}
			if k == "__all__" {
				details = append(details, d)
				continue
			}
			details = append(details, k+": "+d)
		}
		return strings.Join(details, "; ")
	}
	return string(data)
}

// isPerpetualPair reports whether a pair is a perpetual contract market, e.g. BTC/USD-PERP
func isPerpetualPair(pair currency.Pair) bool {
	return strings.HasSuffix(formatMarketName(pair), perpetualMarketSuffix)
}

// paginationParams returns the query parameters of a page request
func paginationParams(req *PaginationRequest) url.Values {
	params := url.Values{}
	if req.Page != 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.PerPage != 0 {
		params.Set("per_page", strconv.FormatUint(req.PerPage, 10))
	}
	return params
}

// validatePaging checks a limit and offset against the documented maximums
func validatePaging(limit, offset uint64) error {
	if limit > maxResultLimit {
		return fmt.Errorf("%w: %d must not exceed %d", errInvalidLimit, limit, maxResultLimit)
	}
	if offset > maxResultOffset {
		return fmt.Errorf("%w: %d must not exceed %d", errInvalidOffset, offset, maxResultOffset)
	}
	return nil
}

// formatMarketSymbol returns the market symbol used in endpoint paths, e.g. btcusd or btcusd-perp
func formatMarketSymbol(pair currency.Pair) string {
	return currency.EMPTYFORMAT.Format(pair)
}

// formatMarketName returns the market name used in request parameters, e.g. BTC/USD or BTC/USD-PERP
func formatMarketName(pair currency.Pair) string {
	return pair.Format(currency.PairFormat{Uppercase: true, Delimiter: currency.ForwardSlashDelimiter}).String()
}

// formatOrderSide returns the side used in order endpoint paths
func formatOrderSide(side order.Side) (string, error) {
	switch {
	case side.IsLong():
		return "buy", nil
	case side.IsShort():
		return "sell", nil
	default:
		return "", fmt.Errorf("%w: %s", order.ErrSideIsInvalid, side)
	}
}

// formatCollateral formats collateral amounts keyed by currency for JSON request bodies
func formatCollateral(collateral map[currency.Code]float64) map[string]string {
	if len(collateral) == 0 {
		return nil
	}
	formatted := make(map[string]string, len(collateral))
	for c, amount := range collateral {
		formatted[c.Upper().String()] = strconv.FormatFloat(amount, 'f', -1, 64)
	}
	return formatted
}

func setFloat(params url.Values, key string, value float64) {
	if value != 0 {
		params.Set(key, strconv.FormatFloat(value, 'f', -1, 64))
	}
}

func setTrue(params url.Values, key string, value bool) {
	if value {
		params.Set(key, trueValue)
	}
}

// formatBool returns a boolean in the form the API expects in form bodies
func formatBool(value bool) string {
	if value {
		return trueValue
	}
	return falseValue
}
