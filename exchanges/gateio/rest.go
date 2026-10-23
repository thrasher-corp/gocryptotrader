package gateio

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/orderbookmanager"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	gateioAPIVersion                    = "/api/v4/"
	gateioTradeURL                      = "https://api.gateio.ws"
	gateioFuturesTestnetTrading         = "https://fx-api-testnet.gateio.ws"
	gateioFuturesLiveTradingAlternative = "https://fx-api.gateio.ws"
	testnetRESTURL                      = "https://api-testnet.gateapi.io"
	tradeBaseURL                        = "https://www.gate.io/"

	// Account names as expected on the wire; these are decoupled from asset.Item String values
	spotAccount        = "spot"
	crossMarginAccount = "cross_margin"
	futuresAccount     = "futures"
	deliveryAccount    = "delivery"
	unifiedAccount     = "unified"
)

const (
	utc0TimeZone = "utc0"
	utc8TimeZone = "utc8"
	allTimeZones = "all"

	// sizeDecimalHeader opts in to decimal sizes; without it futures sizes are truncated to whole
	// contracts, which silently loses precision and reports sub-unit orderbook levels as zero
	sizeDecimalHeader = "X-Gate-Size-Decimal"
)

var (
	errEmptyOrInvalidSettlementCurrency = errors.New("empty or invalid settlement currency")
	errLoanTypeIsRequired               = errors.New("loan type is required")
	errUserIDRequired                   = errors.New("user id is required")
	errSTPGroupNameRequired             = errors.New("self-trade prevention group name required")
	errSTPGroupIDRequired               = errors.New("self-trade prevention group id required")
	errPlanIDRequired                   = errors.New("plan ID required")
	errInvalidCurrencyChain             = errors.New("name of the chain used for withdrawal must be specified")
	errInvalidUnderlying                = errors.New("missing underlying")
	errInvalidOrderSize                 = errors.New("invalid order size")
	errInvalidSubAccount                = errors.New("invalid or empty subaccount")
	errInvalidTransferDirection         = errors.New("invalid transfer direction")
	errDifferentAccount                 = errors.New("account type must be identical for all orders")
	errNoValidParameterPassed           = errors.New("no valid parameter passed")
	errPeriodTypeRequired               = errors.New("period type required")
	errInvalidPlanPeriodDay             = errors.New("invalid plan period day")
	errInvalidPlanPeriodHour            = errors.New("invalid plan period hour")
	errInvalidTriggerRule               = errors.New("invalid price trigger rule")
	errInvalidOrderStatus               = errors.New("invalid order status")
	errInvalidLoanID                    = errors.New("missing loan ID")
	errMissingPreviewID                 = errors.New("missing required parameter: preview_id")
	errInvalidAutoSize                  = errors.New("invalid autoSize")
	errTooManyOrderRequest              = errors.New("too many order creation request")
	errInvalidTimeout                   = errors.New("invalid countdown timeout, which must be whole seconds of at least 5, or zero to stop the countdown")
	errNotWholeSeconds                  = errors.New("duration must be whole seconds")
	errNoTickerData                     = errors.New("no ticker data available")
	errInvalidTimezone                  = errors.New("invalid timezone")
	errTooManyBatchOrders               = errors.New("too many orders in batch request")
	errBusinessTypeRequired             = errors.New("business type is required")
	errParticipationRateRequired        = errors.New("participation rate is required")
	errTimeToLiveRequired               = errors.New("time to live is required")
	errMissingWithdrawalID              = errors.New("missing withdrawal ID")
	errTransactionIDRequired            = errors.New("transaction ID is required")
	errTransferStatusIDRequired         = errors.New("client order ID or transaction ID is required")
	errInvalidSubAccountUserID          = errors.New("sub-account user id is required")
	errSubAccountTransferHistoryStart   = errors.New("sub-account transfer history starts on 2020-04-10")
	errSubAccountTransferHistoryRange   = errors.New("sub-account transfer history range exceeds 30 days")
	errTimeRangeTooLong                 = errors.New("time range exceeds the maximum allowed span")
	errInvalidSettlementQuote           = errors.New("symbol quote currency does not match asset settlement currency")
	errInvalidSettlementBase            = errors.New("symbol base currency does not match asset settlement currency")
	errMissingAPIKey                    = errors.New("missing API key information")
	errInvalidTextPrefix                = errors.New("invalid text value, requires prefix `t-`")
	errSingleAssetRequired              = errors.New("single asset type required")
	errCancelOutcomeMissing             = errors.New("order cancellation outcome missing from response")
	errMissingUnifiedAccountMode        = errors.New("unified account mode is required")
	errInvalidCollateralType            = errors.New("invalid collateral type")
	errInvalidEquity                    = errors.New("equity must not be 0")
	errTooManyCurrencyCodes             = errors.New("too many currency codes supplied")
	errTooManyCurrencyPairs             = errors.New("too many currency pairs supplied")
	errFetchingOrderbook                = errors.New("error fetching orderbook")
	errNoSpotInstrument                 = errors.New("no spot instrument available")
	errInvalidLeverage                  = errors.New("invalid leverage value")
	errChaseOrderIDOrTextRequired       = errors.New("either id or text is required to stop a chase order")
	errInvalidChaseSortBy               = errors.New("invalid sort_by value: must be 1 (ORDER_SORT_CREATED_AT) or 2 (ORDER_SORT_FINISHED_AT)")
	errCurrencyTypeRequired             = errors.New("currency type is required")
	errOrderRefundRequestIDRequired     = errors.New("order-refund request ID is required")
	errRecordIDRequired                 = errors.New("record ID is required")
	errPageRequired                     = errors.New("page is required")
	errInvalidReinvestStatus            = errors.New("invalid reinvest status")
	errUniChartRangeExceeded            = errors.New("uni chart time range exceeds 30 days")
	errFixedRateTermRequired            = errors.New("fixed rate loan term is required")
	errFixedRateRequired                = errors.New("fixed interest rate is required")
	errProductIDRequired                = errors.New("product ID is missing")
	errPlanStatusRequired               = errors.New("plan status is required")
	errHistoryTypeRequired              = errors.New("history type is required")
	errInvalidLoanType                  = errors.New("invalid loan type")
)

// validTimesInForce holds a list of supported time-in-force values and corresponding string representations.
// slice iteration outperforms map with this few elements
var validTimesInForce = []struct {
	String      string
	TimeInForce order.TimeInForce
}{
	{gtcTIF, order.GoodTillCancel}, {iocTIF, order.ImmediateOrCancel}, {pocTIF, order.PostOnly}, {fokTIF, order.FillOrKill},
}

func timeInForceFromString(tif string) (order.TimeInForce, error) {
	for a := range validTimesInForce {
		if validTimesInForce[a].String == tif {
			return validTimesInForce[a].TimeInForce, nil
		}
	}
	return order.UnknownTIF, fmt.Errorf("%w: %q", order.ErrUnsupportedTimeInForce, tif)
}

// setUnixTimeRangeParams validates fully populated ranges before setting Unix timestamp parameters.
func setUnixTimeRangeParams(params *url.Values, from, to time.Time) error {
	if !from.IsZero() && !to.IsZero() {
		if err := common.StartEndTimeCheck(from, to); err != nil {
			if !errors.Is(err, common.ErrStartEqualsEnd) {
				return err
			}
			if from.After(time.Now()) {
				return common.ErrStartAfterTimeNow
			}
		}
	}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.Unix(), 10))
	}
	return nil
}

// unixSeconds returns t in Unix seconds for Gate's unsigned request fields. An unset time, or any time before 1970,
// returns 0 rather than wrapping around, so an omitempty field leaves it out
func unixSeconds(t time.Time) uint64 {
	return uint64(max(t.Unix(), 0))
}

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with GateIO
type Exchange struct {
	exchange.Base

	messageIDSeq  common.Counter
	wsOBResubMgr  *wsOBResubManager
	wsOBUpdateMgr *orderbookmanager.UpdateManager

	futuresUserIDMu sync.RWMutex
	futuresUserIDs  map[string]string
}

var intervalAndStringRepresentations = []struct {
	Interval kline.Interval
	String   string
}{
	{kline.TenMilliseconds, "10ms"},
	{kline.TwentyMilliseconds, "20ms"},
	{kline.HundredMilliseconds, "100ms"},
	{kline.TwoHundredAndFiftyMilliseconds, "250ms"},
	{kline.ThousandMilliseconds, "1000ms"},
	{kline.TenSecond, "10s"},
	{kline.ThirtySecond, "30s"},
	{kline.OneMin, "1m"},
	{kline.FiveMin, "5m"},
	{kline.FifteenMin, "15m"},
	{kline.ThirtyMin, "30m"},
	{kline.OneHour, "1h"},
	{kline.TwoHour, "2h"},
	{kline.FourHour, "4h"},
	{kline.SixHour, "6h"},
	{kline.EightHour, "8h"},
	{kline.TwelveHour, "12h"},
	{kline.OneDay, "1d"},
	{kline.SevenDay, "7d"},
	{kline.OneMonth, "30d"},
}

// GetIntervalFromString returns a kline.Interval representation of the interval string
func (e *Exchange) GetIntervalFromString(interval string) (kline.Interval, error) {
	interval = strings.ToLower(interval)
	for _, result := range intervalAndStringRepresentations {
		if result.String == interval {
			return result.Interval, nil
		}
	}
	return kline.Interval(0), fmt.Errorf("%w: %s", kline.ErrInvalidInterval, interval)
}

// getIntervalString returns a string representation of the interval according to the Gateio exchange representation
func getIntervalString(interval kline.Interval) (string, error) {
	for _, result := range intervalAndStringRepresentations {
		if result.Interval == interval {
			return result.String, nil
		}
	}
	return "", fmt.Errorf("%q: %w", interval.String(), kline.ErrUnsupportedInterval)
}

// currencyListParam validates a batch of currencies and joins them into a comma separated query parameter value
func currencyListParam(currencies []currency.Code, maximum int) (string, error) {
	if len(currencies) == 0 {
		return "", currency.ErrCurrencyCodesEmpty
	}
	if len(currencies) > maximum {
		return "", fmt.Errorf("%w: maximum %d", errTooManyCurrencyCodes, maximum)
	}
	if slices.ContainsFunc(currencies, currency.Code.IsEmpty) {
		return "", currency.ErrCurrencyCodeEmpty
	}
	return currency.Currencies(currencies).Join(), nil
}

// GenerateSignature returns hash for authenticated requests
func (e *Exchange) GenerateSignature(secret, method, path, query string, body any, dtime time.Time) (string, error) {
	rawQuery, err := url.QueryUnescape(query)
	if err != nil {
		return "", err
	}

	h := sha512.New()
	if body != nil {
		val, err := json.Marshal(body)
		if err != nil {
			return "", err
		}
		h.Write(val)
	}

	h.Write(nil)
	hashedPayload := hex.EncodeToString(h.Sum(nil))
	msg := method + "\n" + path + "\n" + rawQuery + "\n" + hashedPayload + "\n" + strconv.FormatInt(dtime.Unix(), 10)
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// walletAccountType returns the account name the wallet endpoints expect for an asset, since Gate names its perpetual
// futures account "futures" while GCT splits it into USDT and coin margined assets
func walletAccountType(a asset.Item) (string, error) {
	switch a {
	case asset.Spot:
		return spotAccount, nil
	case asset.Margin:
		return "margin", nil
	case asset.Futures, asset.USDTMarginedFutures, asset.CoinMarginedFutures:
		return futuresAccount, nil
	case asset.DeliveryFutures:
		return deliveryAccount, nil
	case asset.Options:
		return "options", nil
	case asset.Empty:
		return "", asset.ErrInvalidAsset
	default:
		return "", fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
	}
}

var (
	errInvalidNumericString         = errors.New("invalid numeric string")
	errInvalidDisplayNumber         = errors.New("invalid display number")
	errInvalidDate                  = errors.New("invalid date")
	errLimitWithTimeRange           = errors.New("limit cannot be combined with a from or to time")
	errLiquidationTimeRangeTooLarge = errors.New("liquidation order time range exceeds one hour")
	errInvalidMarginChange          = errors.New("margin change must be non-zero")
	errInvalidPositionMarginMode    = errors.New("invalid position margin mode")
	errInvalidPositionMode          = errors.New("invalid position mode")
	errNoOrderAmendment             = errors.New("no order amendment supplied")
	errTooManyOrders                = errors.New("too many orders in one request")
	errInvalidOrderBookLevel        = errors.New("order book level must be between 1 and 20")
	errInvalidTradeRole             = errors.New("invalid trade role")
	errInvalidTriggerStrategyType   = errors.New("invalid price trigger strategy type")
	errInvalidTriggerPriceType      = errors.New("invalid price trigger price type")
	errInvalidAutoOrderType         = errors.New("invalid auto order type")
	errTrailOrderIDOrTextRequired   = errors.New("either id or text is required to stop a trail order")
	errInvalidTrailOrderFilter      = errors.New("invalid trail order filter")
	errChasePriceAndOffsetLimit     = errors.New("price_limit and offset_limit cannot both be set on a chase order")
	errChaseTimeRangeRequired       = errors.New("start and end times are required to list finished chase orders")
	errInvalidChaseOrderFilter      = errors.New("invalid chase order filter")
)

// validateCountdownTimeout checks a countdown cancel timeout, which Gate needs in whole seconds of at least 5, or zero
// to stop the countdown
func validateCountdownTimeout(timeout time.Duration) error {
	if timeout < 0 || timeout != 0 && timeout < 5*time.Second || timeout%time.Second != 0 {
		return fmt.Errorf("%w: %s", errInvalidTimeout, timeout)
	}
	return nil
}

// checkWholeSeconds refuses a duration Gate takes as an integer number of seconds when it holds a fraction of one, as
// marshalling it would send a decimal Gate rejects
func checkWholeSeconds(d types.DurationSeconds, field string) error {
	if d.Duration()%time.Second != 0 {
		return fmt.Errorf("%w: %s is %s", errNotWholeSeconds, field, d)
	}
	return nil
}

// validate checks a futures price triggered order before it is sent
func (p *FuturesPriceTriggeredOrderRequest) validate() error {
	if err := common.NilGuard(p); err != nil {
		return err
	}
	if p.Initial.Contract.IsEmpty() {
		return fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if p.Initial.Price < 0 {
		return fmt.Errorf("%w, price must be greater than 0", limits.ErrPriceBelowMin)
	}
	if p.Initial.TimeInForce != "" && p.Initial.TimeInForce != gtcTIF && p.Initial.TimeInForce != iocTIF {
		return fmt.Errorf("%w: %q; only 'gtc' and 'ioc' are allowed", order.ErrInvalidTimeInForce, p.Initial.TimeInForce)
	}
	if p.Trigger.StrategyType != 0 && p.Trigger.StrategyType != 1 {
		return fmt.Errorf("%w: %d; 0: by price, 1: by price gap", errInvalidTriggerStrategyType, p.Trigger.StrategyType)
	}
	if p.Trigger.PriceType != 0 && p.Trigger.PriceType != 1 && p.Trigger.PriceType != 2 {
		return fmt.Errorf("%w: %d; 0: latest trade price, 1: mark price, 2: index price", errInvalidTriggerPriceType, p.Trigger.PriceType)
	}
	if p.Trigger.Price <= 0 {
		return fmt.Errorf("%w: trigger price is required", limits.ErrPriceBelowMin)
	}
	if p.Trigger.Rule != 1 && p.Trigger.Rule != 2 {
		return fmt.Errorf("%w: %d; 1: at or above the trigger price, 2: at or below it", errInvalidTriggerRule, p.Trigger.Rule)
	}
	if err := checkWholeSeconds(p.Trigger.ExpireAfter, "trigger expiry"); err != nil {
		return err
	}
	// close-long-order and close-short-order are read-only types Gate reports for an order's own take profit and stop loss
	switch p.OrderType {
	case "", "close-long-position", "close-short-position", "plan-close-long-position", "plan-close-short-position":
		return nil
	default:
		return fmt.Errorf("%w: %q", errInvalidAutoOrderType, p.OrderType)
	}
}

// GetUnderlyingFromCurrencyPair returns an underlying string from a currency pair
func (e *Exchange) GetUnderlyingFromCurrencyPair(p currency.Pair) (currency.Pair, error) {
	pairString := strings.ReplaceAll(p.Upper().String(), currency.DashDelimiter, currency.UnderscoreDelimiter)
	ccies := strings.Split(pairString, currency.UnderscoreDelimiter)
	if len(ccies) < 2 {
		return currency.EMPTYPAIR, fmt.Errorf("invalid currency pair %v", p)
	}
	return currency.Pair{Base: currency.NewCode(ccies[0]), Delimiter: currency.UnderscoreDelimiter, Quote: currency.NewCode(ccies[1])}, nil
}

// validateOrderCreateParams validates common order creation parameters shared by futures and delivery orders.
func validateOrderCreateParams(contract currency.Pair, size, price types.Number, closePosition bool, autoSize, timeInForce, text string, settle currency.Code, isRest bool) error {
	if contract.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	// Gate closes a position with a zero size, closing the whole position with close or one side of a dual mode
	// position with auto_size
	if size == 0 && !closePosition && autoSize == "" {
		return errInvalidOrderSize
	}
	if closePosition && size != 0 {
		return fmt.Errorf("%w: size must be zero when closing a position", errInvalidOrderSize)
	}
	if timeInForce != "" {
		if _, err := timeInForceFromString(timeInForce); err != nil {
			return err
		}
	}
	if price == 0 && timeInForce != iocTIF && timeInForce != fokTIF {
		return fmt.Errorf("%w: %q; only 'ioc' and 'fok' allowed for market order", order.ErrUnsupportedTimeInForce, timeInForce)
	}
	if text != "" && !strings.HasPrefix(text, "t-") {
		return errInvalidTextPrefix
	}
	if autoSize != "" {
		if autoSize != "close_long" && autoSize != "close_short" {
			return fmt.Errorf("%w: %q", errInvalidAutoSize, autoSize)
		}
		if size != 0 {
			return fmt.Errorf("%w: size must be zero when auto size is set", order.ErrAmountIsInvalid)
		}
	}
	// REST requests require a settlement currency, but it can be anything
	// Websocket requests may have an empty settlement currency, or it must be BTC or USDT
	if (isRest && settle.IsEmpty()) ||
		(!isRest && !settle.IsEmpty() && !settle.Equal(currency.BTC) && !settle.Equal(currency.USDT)) {
		return currency.ErrCurrencyCodeEmpty
	}
	return nil
}

// SendAuthenticatedHTTPRequest sends authenticated requests to the Gateio API
func (e *Exchange) SendAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, epl request.EndpointLimit, method, endpoint string, param url.Values, data, result any) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	ePoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	respHeaders := make(http.Header)
	var intermediary json.RawMessage
	if err := e.SendPayload(ctx, epl, func() (*request.Item, error) {
		headers := make(map[string]string)
		urlPath := gateioAPIVersion + endpoint
		timestamp := time.Now()
		var paramValue string
		if param != nil {
			paramValue = param.Encode()
		}
		var sig string
		sig, err = e.GenerateSignature(creds.Secret, method, urlPath, paramValue, data, timestamp)
		if err != nil {
			return nil, err
		}
		headers["Content-Type"] = "application/json"
		headers["KEY"] = creds.Key
		headers["TIMESTAMP"] = strconv.FormatInt(timestamp.Unix(), 10)
		headers["Accept"] = "application/json"
		headers["SIGN"] = sig
		headers[sizeDecimalHeader] = "1"
		urlPath = ePoint + urlPath
		if param != nil {
			urlPath = common.EncodeURLValues(urlPath, param)
		}
		var payload string
		if data != nil {
			var byteData []byte
			byteData, err = json.Marshal(data)
			if err != nil {
				return nil, err
			}
			payload = string(byteData)
		}
		return &request.Item{
			Method:                 method,
			Path:                   urlPath,
			Headers:                headers,
			Body:                   strings.NewReader(payload),
			Result:                 &intermediary,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
			HeaderResponse:         &respHeaders,
		}, nil
	}, request.AuthenticatedRequest); err != nil {
		return err
	}
	if respHeaders.Get("Status") == "204" { // 204 No Content is returned with empty body, so intermediary will be empty but it is not an error
		if len(intermediary) != 0 {
			return fmt.Errorf("%s %w, expected empty response body but got %s", e.Name, request.ErrAuthRequestFailed, string(intermediary))
		}
		if result == nil {
			return nil
		}
		return fmt.Errorf("%s %w, empty response body with non-nil result", e.Name, request.ErrAuthRequestFailed)
	}
	var errCap codedErrorReply
	if err := json.Unmarshal(intermediary, &errCap); err == nil && errCap.Code != "" {
		return fmt.Errorf("%s auth request error, code: %s message: %s", e.Name, errCap.Label, errCap.Message)
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(intermediary, result); err != nil {
		return fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, err)
	}
	if err := responseError(result); err != nil {
		return fmt.Errorf("%w: %w", request.ErrAuthRequestFailed, err)
	}
	return nil
}

// responseError surfaces an API level error carried in the body of an otherwise successful
// response. Results are commonly unmarshalled into a pointer to a pointer, whose method set is
// empty, so the pointed-to value is checked as well.
func responseError(result any) error {
	if errType, ok := result.(interface{ AsError() error }); ok {
		return errType.AsError()
	}
	if v := reflect.ValueOf(result); v.Kind() == reflect.Pointer && !v.IsNil() {
		elem := v.Elem()
		// A null response leaves the pointed-to pointer nil, and every AsError reads its receiver
		if elem.Kind() == reflect.Pointer && elem.IsNil() {
			return nil
		}
		if errType, ok := reflect.TypeAssert[interface{ AsError() error }](elem); ok {
			return errType.AsError()
		}
	}
	return nil
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (e *Exchange) SendHTTPRequest(ctx context.Context, ep exchange.URL, epl request.EndpointLimit, path string, result any) error {
	return e.sendUnversionedHTTPRequest(ctx, ep, epl, gateioAPIVersion+path, result)
}

// sendPublicPostRequest sends an unauthenticated POST request carrying a JSON body to a versioned REST endpoint
func (e *Exchange) sendPublicPostRequest(ctx context.Context, epl request.EndpointLimit, path string, data, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(exchange.RestSpot)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	var intermediary json.RawMessage
	if err := e.SendPayload(ctx, epl, func() (*request.Item, error) {
		return &request.Item{
			Method:                 http.MethodPost,
			Path:                   endpoint + gateioAPIVersion + path,
			Headers:                map[string]string{"Content-Type": "application/json", sizeDecimalHeader: "1"},
			Body:                   bytes.NewReader(payload),
			Result:                 &intermediary,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.UnauthenticatedRequest); err != nil {
		return err
	}
	var errCap errorReply
	if err := json.Unmarshal(intermediary, &errCap); err == nil && errCap.Label != "" {
		return fmt.Errorf("%s request error, code: %s message: %s", e.Name, errCap.Label, errCap.Message)
	}
	if err := json.Unmarshal(intermediary, result); err != nil {
		return fmt.Errorf("%s: %w", e.Name, err)
	}
	return responseError(result)
}

// sendUnversionedHTTPRequest sends an unauthenticated HTTP request to path, which is appended to the endpoint verbatim.
// Endpoints outside of the versioned REST API, such as the web front-end, carry their own version prefix.
func (e *Exchange) sendUnversionedHTTPRequest(ctx context.Context, ep exchange.URL, epl request.EndpointLimit, path string, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	var intermediary json.RawMessage
	if err := e.SendPayload(ctx, epl, func() (*request.Item, error) {
		return &request.Item{
			Method:                 http.MethodGet,
			Path:                   endpoint + path,
			Headers:                map[string]string{sizeDecimalHeader: "1"},
			Result:                 &intermediary,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.UnauthenticatedRequest); err != nil {
		return err
	}

	var errCap codedErrorReply
	if err := json.Unmarshal(intermediary, &errCap); err == nil && errCap.Code != "" {
		return fmt.Errorf("%s request error, code: %s message: %s", e.Name, errCap.Label, errCap.Message)
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(intermediary, result); err != nil {
		return fmt.Errorf("%s: %w", e.Name, err)
	}
	return responseError(result)
}
