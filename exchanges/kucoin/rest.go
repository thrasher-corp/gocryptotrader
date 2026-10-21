package kucoin

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/orderbookmanager"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with Kucoin
type Exchange struct {
	exchange.Base
	wsOBUpdateMgr *orderbookmanager.UpdateManager
	// marginOrderIDs holds the IDs of the live margin orders this exchange placed or listed, as spot and margin order
	// pushes share one topic and do not say which asset an order is
	marginOrderIDs sync.Map
	// spotAccountBalances holds each set of credentials' main and trade account balances. The spot balance is the two
	// together, while REST lists them apart and a balance push carries one account's figures alone
	spotAccountBalances map[accounts.Credentials]spotAccountShares
	// spotAccountsListedAt holds when each set of credentials' accounts were last listed, as a listing that leaves a
	// currency out of an account leaves no share of it to tell a later push from an earlier one by
	spotAccountsListedAt  map[accounts.Credentials]time.Time
	spotAccountBalancesMu sync.Mutex
	// marginBalanceMu makes saving REST's cross margin snapshot and a margin balance push, which keeps the saved
	// liability, one step each
	marginBalanceMu sync.Mutex
	// tickerMergeMu serialises websocket ticker merges, which load the stored ticker and store it back, as the pushes
	// of one pair can arrive on several connections at once
	tickerMergeMu sync.Mutex
}

// KuCoin's REST hosts, and the parts of its trade page URLs the wrapper builds links from
const (
	kucoinAPIURL       = "https://api.kucoin.com/api"
	kucoinBrokerAPIURL = "https://api-broker.kucoin.com/api"
	tradeBaseURL       = "https://www.kucoin.com/"
	tradeSpot          = "trade/"
	tradeMargin        = "margin/"
	tradeFutures       = "futures/"
)

// SendHTTPRequest sends an unauthenticated HTTP request
func (e *Exchange) SendHTTPRequest(ctx context.Context, ePath exchange.URL, epl request.EndpointLimit, path string, result any) error {
	value := reflect.ValueOf(result)
	if value.Kind() != reflect.Pointer {
		return errInvalidResultInterface
	}
	resp, okay := result.(UnmarshalTo)
	if !okay {
		resp = &Response{Data: result}
	}
	endpointPath, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	err = e.SendPayload(ctx, epl, func() (*request.Item, error) {
		return &request.Item{
			Method:                 http.MethodGet,
			Path:                   endpointPath + path,
			Result:                 resp,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.UnauthenticatedRequest)
	if err != nil {
		return err
	}
	return resp.GetError()
}

// SendAuthHTTPRequest sends an authenticated HTTP request
// Request parameters are added to path variable for GET and DELETE request and for other requests its passed in params variable
func (e *Exchange) SendAuthHTTPRequest(ctx context.Context, ePath exchange.URL, epl request.EndpointLimit, method, path string, arg, result any) error {
	value := reflect.ValueOf(result)
	if value.Kind() != reflect.Pointer {
		return errInvalidResultInterface
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	resp, okay := result.(UnmarshalTo)
	if !okay {
		resp = &Response{Data: result}
	}
	endpointPath, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	if value.IsNil() {
		return fmt.Errorf("%w receiver has to be non-nil pointer", errInvalidResponseReceiver)
	}
	err = e.SendPayload(ctx, epl, func() (*request.Item, error) {
		var (
			body    io.Reader
			payload []byte
		)
		if arg != nil {
			payload, err = json.Marshal(arg)
			if err != nil {
				return nil, err
			}
			body = bytes.NewBuffer(payload)
		}
		ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
		signPath, err := signaturePath(path)
		if err != nil {
			return nil, err
		}
		headers := map[string]string{
			"KC-API-KEY":         creds.Key,
			"KC-API-SIGN":        hmacSign(ts+method+"/api"+signPath+string(payload), creds.Secret),
			"KC-API-TIMESTAMP":   ts,
			"KC-API-PASSPHRASE":  hmacSign(creds.ClientID, creds.Secret),
			"KC-API-KEY-VERSION": "3",
			"Content-Type":       "application/json",
		}
		return &request.Item{
			Method:                 method,
			Path:                   endpointPath + path,
			Headers:                headers,
			Body:                   body,
			Result:                 resp,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.AuthenticatedRequest)
	if err != nil {
		return err
	}
	return resp.GetError()
}

// setTimeRange sets an optional time range under the endpoint's start and end keys, in Unix milliseconds, for
// whichever ends are set, after checking the range when both are
func setTimeRange(params url.Values, startKey, endKey string, start, end time.Time) error {
	if !start.IsZero() && !end.IsZero() {
		if err := common.StartEndTimeCheck(start, end); err != nil {
			return err
		}
	}
	if !start.IsZero() {
		params.Set(startKey, strconv.FormatInt(start.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set(endKey, strconv.FormatInt(end.UnixMilli(), 10))
	}
	return nil
}

// hmacSign returns the base64 HMAC-SHA256 of message keyed by secret, the signature KuCoin's request, passphrase and
// websocket authentication all use
func hmacSign(message, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// signaturePath returns the path a request is signed over: KuCoin verifies the signature against the query as it was
// before URL encoding, so a list joined by commas or a value with reserved characters is signed unescaped
func signaturePath(path string) (string, error) {
	base, query, found := strings.Cut(path, "?")
	if !found {
		return path, nil
	}
	unescaped, err := url.QueryUnescape(query)
	if err != nil {
		return "", fmt.Errorf("error unescaping the query to sign: %w", err)
	}
	return base + "?" + unescaped, nil
}

var intervalMap = map[kline.Interval]string{
	kline.OneMin: "1min", kline.ThreeMin: "3min", kline.FiveMin: "5min", kline.FifteenMin: "15min", kline.ThirtyMin: "30min", kline.OneHour: "1hour", kline.TwoHour: "2hour", kline.FourHour: "4hour", kline.SixHour: "6hour", kline.EightHour: "8hour", kline.TwelveHour: "12hour", kline.OneDay: "1day", kline.OneWeek: "1week",
}

// IntervalToString returns a string from kline.Interval input.
func IntervalToString(interval kline.Interval) (string, error) {
	intervalString, okay := intervalMap[interval]
	if okay {
		return intervalString, nil
	}
	return "", fmt.Errorf("%w interval: %v", kline.ErrUnsupportedInterval, interval)
}

// Request validation and response handling errors shared by the package's endpoints
var (
	errUIDRequired                = errors.New("UID is required")
	errStopPriceRequired          = errors.New("stop price is required")
	errLimitPriceRequired         = errors.New("limit price is required")
	errInvalidResponseReceiver    = errors.New("invalid response receiver")
	errInvalidStopPriceType       = errors.New("stopPriceType is required")
	errNoDepositAddress           = errors.New("no deposit address found")
	errAddressRequired            = errors.New("address is required")
	errMultipleDepositAddress     = errors.New("multiple deposit addresses")
	errInvalidResultInterface     = errors.New("result interface has to be pointer")
	errInvalidSubAccountName      = errors.New("invalid sub-account name")
	errRemarkIsRequired           = errors.New("remark with a 24 characters max-length is required")
	errAPIKeyRequired             = errors.New("account API key is required")
	errInvalidPassPhraseInstance  = errors.New("invalid passphrase string")
	errSizeOrFundIsRequired       = errors.New("at least one required among size and funds")
	errInvalidLeverage            = errors.New("invalid leverage value")
	errAccountTypeMissing         = errors.New("account type is required")
	errTransferTypeMissing        = errors.New("transfer type is required")
	errTradeTypeMissing           = errors.New("trade type is missing")
	errTimeInForceRequired        = errors.New("time in force is required")
	errInvalidLimit               = errors.New("invalid limit")
	errInvalidMsgType             = errors.New("message type field not valid")
	errMissingPurchaseOrderNumber = errors.New("missing purchase order number")
	errMissingInterestRate        = errors.New("interest rate is required")
	errAccountIDMissing           = errors.New("account ID is required")
	errProductIDMissing           = errors.New("product ID is missing")
	errStatusMissing              = errors.New("status is missing")
	errInvalidPeriod              = errors.New("invalid period")
	errTransferDirectionRequired  = errors.New("transfer direction cannot be empty")
	errSubUserIDRequired          = errors.New("sub-user ID is required")
	errPageSizeRequired           = errors.New("pageSize is required")
	errCurrentPageRequired        = errors.New("current page value is required")
	errTimeoutRequired            = errors.New("timeout value required")
)
