package bybit

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with Bybit
type Exchange struct {
	exchange.Base
}

const (
	bybitAPIURL  = "https://api.bybit.com"
	bybitDemoURL = "https://api-demo.bybit.com"
	tradeBaseURL = "https://www.bybit.com/"

	defaultRecvWindow = "5000" // 5000 milli second

	sideBuy  = "Buy"
	sideSell = "Sell"

	cSpot, cLinear, cOption, cInverse = "spot", "linear", "option", "inverse"
	tradingStatus                     = "Trading"

	// longDatedFormat is how Bybit writes expiry dates in symbols and expDate parameters, with no leading zero on the
	// day: 6NOV26 matches and 06NOV26 does not
	longDatedFormat = "2Jan06"
)

var (
	errCategoryNotSet                     = errors.New("category not set")
	errBaseNotSet                         = errors.New("base coin not set when category is option")
	errMissingUserID                      = errors.New("sub user id missing")
	errMissingUsername                    = errors.New("username is missing")
	errInvalidMemberType                  = errors.New("invalid member type")
	errMissingTransferID                  = errors.New("transfer ID is required")
	errMemberIDRequired                   = errors.New("member ID is required")
	errEitherOrderIDOROrderLinkIDRequired = errors.New("either orderId or orderLinkId required")
	errNoOrderPassed                      = errors.New("no order passed")
	errTakeProfitOrStopLossModeMissing    = errors.New("TP/SL mode missing")
	errMissingAccountType                 = errors.New("account type not specified")
	errMembersIDsNotSet                   = errors.New("members IDs not set")
	errMissingChainType                   = errors.New("missing chain type is empty")
	errMissingChainInformation            = errors.New("missing transfer chain")
	errMissingAddressInfo                 = errors.New("address is required")
	errMissingWithdrawalID                = errors.New("missing withdrawal id")
	errTimeWindowRequired                 = errors.New("time window is required")
	errQuantityLimitRequired              = errors.New("quantity limit required")
	errInvalidLeverage                    = errors.New("leverage can't be zero or less then it")
	errInvalidMode                        = errors.New("mode can't be empty or missing")
	errEitherSymbolOrCoinRequired         = errors.New("either symbol or coin required")
	errOrderLinkIDMissing                 = errors.New("order link id missing")
	errSymbolMissing                      = errors.New("symbol missing")
	errDisconnectTimeWindowNotSet         = errors.New("disconnect time window not set")
	errAPIResponse                        = errors.New("bybit API error")
	errFeeRateNotFound                    = errors.New("fee rate not found")
	errTriggerPriceRequired               = errors.New("trigger price is required")
	errCancelAllOrdersFailed              = errors.New("cancel all orders failed")
	errBodyWithGet                        = errors.New("signed GET requests cannot carry a body")
	errQueryWithBody                      = errors.New("signed requests with a body cannot carry query parameters")
	errInvalidContractLength              = errors.New("contract length cannot be less than or equal to zero")
)

var (
	intervalMap         = map[kline.Interval]string{kline.OneMin: "1", kline.ThreeMin: "3", kline.FiveMin: "5", kline.FifteenMin: "15", kline.ThirtyMin: "30", kline.OneHour: "60", kline.TwoHour: "120", kline.FourHour: "240", kline.SixHour: "360", kline.TwelveHour: "720", kline.OneDay: "D", kline.OneWeek: "W", kline.OneMonth: "M"}
	stringToIntervalMap = map[string]kline.Interval{"1": kline.OneMin, "3": kline.ThreeMin, "5": kline.FiveMin, "15": kline.FifteenMin, "30": kline.ThirtyMin, "60": kline.OneHour, "120": kline.TwoHour, "240": kline.FourHour, "360": kline.SixHour, "720": kline.TwelveHour, "D": kline.OneDay, "W": kline.OneWeek, "M": kline.OneMonth}
)

// SendHTTPRequest sends an unauthenticated GET request; path carries its query string
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

// SendAuthHTTPRequest sends a signed request. A GET request carries params in its query string; any other method
// sends body as JSON, or an empty object when body is nil, since Bybit signs the query string or the body but not both
func (e *Exchange) SendAuthHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, params url.Values, body, result any, f request.EndpointLimit) error {
	if method == http.MethodGet && body != nil {
		return fmt.Errorf("%w: GET %s", errBodyWithGet, path)
	}
	if method != http.MethodGet && len(params) != 0 {
		return fmt.Errorf("%w: %s %s", errQueryWithBody, method, path)
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpoint, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	var payload []byte
	signedPart := params.Encode()
	if method != http.MethodGet {
		payload = []byte("{}")
		if body != nil {
			if payload, err = json.Marshal(body); err != nil {
				return err
			}
		}
		signedPart = string(payload)
	}
	return e.sendRequest(ctx, f, request.AuthenticatedRequest, result, func() (*request.Item, error) {
		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		sign := getSign(timestamp+creds.Key+defaultRecvWindow+signedPart, creds.Secret)
		headers := map[string]string{
			"X-BAPI-API-KEY":     creds.Key,
			"X-BAPI-TIMESTAMP":   timestamp,
			"X-BAPI-RECV-WINDOW": defaultRecvWindow,
			"X-BAPI-SIGN":        sign,
		}
		var reqBody io.Reader
		if payload != nil {
			headers["Content-Type"] = "application/json"
			reqBody = bytes.NewReader(payload)
		}
		return &request.Item{
			Method:                 method,
			Path:                   endpoint + common.EncodeURLValues(path, params),
			Headers:                headers,
			Body:                   reqBody,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	})
}

// sendRequest sends the request newItem builds and decodes the envelope's result into result. A *RestResponse result
// receives the whole envelope instead, for endpoints that report per-item outcomes in retExtInfo
func (e *Exchange) sendRequest(ctx context.Context, f request.EndpointLimit, auth request.AuthType, result any, newItem func() (*request.Item, error)) error {
	var resp RestResponse
	err := e.SendPayload(ctx, f, func() (*request.Item, error) {
		item, err := newItem()
		if err != nil {
			return nil, err
		}
		resp = RestResponse{}
		item.Result = &resp
		return item, nil
	}, auth)
	if apiErr := resp.apiError(); apiErr != nil {
		if err != nil {
			return fmt.Errorf("%w: %w", apiErr, err)
		}
		if auth == request.AuthenticatedRequest {
			return fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, apiErr)
		}
		return apiErr
	}
	if err != nil {
		return err
	}
	if envelope, ok := result.(*RestResponse); ok {
		*envelope = resp
		return nil
	}
	if result == nil {
		return nil
	}
	if len(resp.Result) == 0 || string(resp.Result) == "null" {
		return common.ErrNoResponse
	}
	return json.Unmarshal(resp.Result, result)
}

// apiError returns the envelope's error, or nil when it reports success
func (r *RestResponse) apiError() error {
	code, msg := r.RetCode, r.RetMsg
	if code == 0 && r.SnakeRetCode != 0 {
		code, msg = r.SnakeRetCode, r.SnakeRetMsg
	}
	if code == 0 {
		return nil
	}
	return &APIError{Code: code, Message: msg, ExtInfo: r.RetExtInfo}
}

// APIError is an error Bybit replied with
type APIError struct {
	Code    int64
	Message string
	// ExtInfo holds the retExtInfo details some endpoints add, such as the outcome of each order in a batch
	ExtInfo json.RawMessage
}

// Error returns the error's code and message, and its details when there are any
func (a *APIError) Error() string {
	msg := errAPIResponse.Error()
	if mapped, ok := errorCodeToErrorMap[a.Code]; ok {
		msg += ": " + mapped.Error()
	}
	msg += ": code " + strconv.FormatInt(a.Code, 10) + " msg " + a.Message
	if len(a.ExtInfo) != 0 && string(a.ExtInfo) != "{}" && string(a.ExtInfo) != `""` && string(a.ExtInfo) != "null" {
		msg += " retExtInfo " + string(a.ExtInfo)
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

// errorCodeToErrorMap maps Bybit error codes to package errors callers can match
var errorCodeToErrorMap = map[int64]error{
	110001: order.ErrOrderNotFound,   // Order does not exist
	170213: order.ErrOrderNotFound,   // Order does not exist. (spot)
	170121: currency.ErrPairNotFound, // Invalid symbol. (spot)
}

// StringToOrderStatus returns order status from string
func StringToOrderStatus(status string) order.Status {
	status = strings.ToUpper(status)
	switch status {
	case "CREATED":
		return order.Open
	case "NEW":
		return order.New
	case "REJECTED":
		return order.Rejected
	case "PARTIALLYFILLED", "PARTIALLY_FILLED":
		return order.PartiallyFilled
	case "PARTIALLYFILLEDCANCELED":
		return order.PartiallyFilledCancelled
	case "PENDING_CANCEL":
		return order.PendingCancel
	case "FILLED":
		return order.Filled
	case "CANCELED", "CANCELLED":
		return order.Cancelled
	case "UNTRIGGERED":
		return order.Pending
	case "TRIGGERED":
		return order.Open
	case "DEACTIVATED":
		// Bybit deactivates a spot TP/SL, conditional or OCO order that is cancelled before it triggers
		return order.Cancelled
	case "ACTIVE":
		return order.Active
	default:
		return order.UnknownStatus
	}
}

// getSign returns the hex encoded HMAC-SHA256 of message keyed by secret, which signs REST and websocket requests
func getSign(message, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message)) // writing to a hash never fails
	return hex.EncodeToString(mac.Sum(nil))
}
