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
	errInvalidCountdown                 = errors.New("invalid countdown, Countdown time, in seconds At least 5 seconds, 0 means cancel the countdown")
	errInvalidTriggerRule               = errors.New("invalid price trigger rule")
	errInvalidOrderStatus               = errors.New("invalid order status")
	errInvalidLoanID                    = errors.New("missing loan ID")
	errMissingPreviewID                 = errors.New("missing required parameter: preview_id")
	errInvalidAutoSize                  = errors.New("invalid autoSize")
	errTooManyOrderRequest              = errors.New("too many order creation request")
	errInvalidTimeout                   = errors.New("invalid timeout, should be in seconds At least 5 seconds, 0 means cancel the countdown")
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

// unixSeconds returns t in Unix seconds for Gate's unsigned request fields; a time before 1970 returns 0 rather than
// wrapping around
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

// *****************************************  Spot **************************************

// ListSpotCurrencies retrieves every spot currency, including the deposit and withdrawal status of each of its chains
func (e *Exchange) ListSpotCurrencies(ctx context.Context) ([]*CurrencyInfoResponse, error) {
	var resp []*CurrencyInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicCurrenciesSpotEPL, "spot/currencies", &resp)
}

// GetCurrencyDetail retrieves the details of a single spot currency
func (e *Exchange) GetCurrencyDetail(ctx context.Context, ccy currency.Code) (*CurrencyInfoResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *CurrencyInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicCurrenciesSpotEPL, "spot/currencies/"+ccy.String(), &resp)
}

// ListSpotCurrencyPairs retrieves every spot currency pair
func (e *Exchange) ListSpotCurrencyPairs(ctx context.Context) ([]*CurrencyPairDetailResponse, error) {
	var resp []*CurrencyPairDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicListCurrencyPairsSpotEPL, "spot/currency_pairs", &resp)
}

// GetCurrencyPairDetail retrieves the details of a single spot currency pair
func (e *Exchange) GetCurrencyPairDetail(ctx context.Context, currencyPair currency.Pair) (*CurrencyPairDetailResponse, error) {
	if currencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *CurrencyPairDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicCurrencyPairDetailSpotEPL, "spot/currency_pairs/"+currencyPair.String(), &resp)
}

// GetTickers retrieves ticker information for every currency pair, or only the supplied pair when one is set
// timezone selects the 24 hour change windows returned: utc0, utc8 or all; Gate's default applies when it is empty
// lowest_size and highest_size are only returned when a currency pair is supplied
func (e *Exchange) GetTickers(ctx context.Context, currencyPair currency.Pair, timezone string) ([]*TickerResponse, error) {
	params := url.Values{}
	if !currencyPair.IsEmpty() {
		params.Set("currency_pair", currencyPair.String())
	}
	switch timezone {
	case "":
	case utc0TimeZone, utc8TimeZone, allTimeZones:
		params.Set("timezone", timezone)
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidTimezone, timezone)
	}
	var tickers []*TickerResponse
	return tickers, e.SendHTTPRequest(ctx, exchange.RestSpot, publicTickersSpotEPL, common.EncodeURLValues("spot/tickers", params), &tickers)
}

// GetTicker retrieves a single ticker information for a currency pair.
func (e *Exchange) GetTicker(ctx context.Context, currencyPair currency.Pair, timezone string) (*TickerResponse, error) {
	if currencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	tickers, err := e.GetTickers(ctx, currencyPair, timezone)
	if err != nil {
		return nil, err
	}
	if len(tickers) > 0 {
		return tickers[0], nil
	}
	return nil, fmt.Errorf("%w for currency pair %v", errNoTickerData, currencyPair)
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

// GetOrderbook returns the orderbook for the supplied currency pair; interval merges price levels by the given
// precision and withOrderbookID requests the orderbook update ID
func (e *Exchange) GetOrderbook(ctx context.Context, pairString currency.Pair, interval string, limit uint64, withOrderbookID bool) (*OrderbookResponse, error) {
	if pairString.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if interval != "" {
		params.Set("interval", interval)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	params.Set("currency_pair", pairString.String())
	params.Set("with_id", strconv.FormatBool(withOrderbookID))
	var response *SpotOrderbookResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, publicOrderbookSpotEPL, common.EncodeURLValues("spot/order_book", params), &response); err != nil {
		return nil, err
	}
	return response.MakeOrderbook(), nil
}

// GetMarketTrades retrieves public market trades for a currency pair, filtered either by time range or by lastID
// Gate ignores the time range when lastID is set and only applies reverse alongside lastID
func (e *Exchange) GetMarketTrades(ctx context.Context, pairString currency.Pair, lastID string, reverse bool, startTime, endTime time.Time, page, limit uint64) ([]*TradeResponse, error) {
	if pairString.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("currency_pair", pairString.String())
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if reverse {
		params.Set("reverse", strconv.FormatBool(reverse))
	}
	if err := setUnixTimeRangeParams(&params, startTime, endTime); err != nil {
		return nil, err
	}
	if page != 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*TradeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicMarketTradesSpotEPL, common.EncodeURLValues("spot/trades", params), &resp)
}

// GetCandlesticks retrieves market candlesticks, returning at most 1000 per request
func (e *Exchange) GetCandlesticks(ctx context.Context, currencyPair currency.Pair, limit uint64, from, to time.Time, interval kline.Interval) ([]*CandlestickResponse, error) {
	if currencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if interval.Duration().Microseconds() != 0 {
		intervalString, err := getIntervalString(interval)
		if err != nil {
			return nil, err
		}
		params.Set("interval", intervalString)
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	params.Set("currency_pair", currencyPair.String())
	var candles []*CandlestickResponse
	return candles, e.SendHTTPRequest(ctx, exchange.RestSpot, publicCandleStickSpotEPL, common.EncodeURLValues("spot/candlesticks", params), &candles)
}

// GetAccountBatchFeeRates retrieves the account's spot fee rates for up to 50 currency pairs, keyed by currency pair
func (e *Exchange) GetAccountBatchFeeRates(ctx context.Context, currencyPairs []string) (map[string]*SpotTradingFeeRateResponse, error) {
	if len(currencyPairs) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	if len(currencyPairs) > 50 {
		return nil, fmt.Errorf("%w: %d supplied, maximum is 50", errTooManyCurrencyPairs, len(currencyPairs))
	}
	params := url.Values{}
	params.Set("currency_pairs", strings.Join(currencyPairs, ","))
	var resp map[string]*SpotTradingFeeRateResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotTradingFeeEPL, http.MethodGet, "spot/batch_fee", params, nil, &resp)
}

// GetSpotAccounts retrieves the spot account balances, optionally for a single currency
func (e *Exchange) GetSpotAccounts(ctx context.Context, ccy currency.Code) ([]*SpotAccountResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var response []*SpotAccountResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountsEPL, http.MethodGet, "spot/accounts", params, nil, &response)
}

// GetSpotAccountBook retrieves the spot account's balance change history
// Gate rejects ranges longer than 30 days, and code takes priority over type when both are set
func (e *Exchange) GetSpotAccountBook(ctx context.Context, arg *GetSpotAccountBookRequest) ([]*SpotAccountBookResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	// Gate receives whole seconds, so the 30 day cap is checked in the units sent
	if !arg.From.IsZero() && !arg.To.IsZero() && arg.To.Unix()-arg.From.Unix() > int64(30*24*time.Hour/time.Second) {
		return nil, fmt.Errorf("%w: account book range must not exceed 30 days", errTimeRangeTooLong)
	}
	params := url.Values{}
	if !arg.Currency.IsEmpty() {
		params.Set("currency", arg.Currency.String())
	}
	if err := setUnixTimeRangeParams(&params, arg.From, arg.To); err != nil {
		return nil, err
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Type != "" {
		params.Set("type", arg.Type)
	}
	if arg.Code != "" {
		params.Set("code", arg.Code)
	}
	var resp []*SpotAccountBookResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAccountBookEPL, http.MethodGet, "spot/account_book", params, nil, &resp)
}

// GetUnifiedAccount retrieves unified account.
func (e *Exchange) GetUnifiedAccount(ctx context.Context, ccy currency.Code, subAccountUserID string) (*UnifiedUserAccountResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	var resp *UnifiedUserAccountResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateUnifiedSpotEPL, http.MethodGet, "unified/accounts", params, nil, &resp)
}

// GetMaximumBorrowableAmountUnifiedAccount query maximum borrowable amount for unified account
func (e *Exchange) GetMaximumBorrowableAmountUnifiedAccount(ctx context.Context, ccy currency.Code) (*CurrencyAndAmountResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var resp *CurrencyAndAmountResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateUnifiedSpotEPL, http.MethodGet, "unified/borrowable", params, nil, &resp)
}

// GetUnifiedAccountMaximumTransferableAmount query maximum transferable amount for unified account
func (e *Exchange) GetUnifiedAccountMaximumTransferableAmount(ctx context.Context, ccy currency.Code) (*CurrencyAndAmountResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var resp *CurrencyAndAmountResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateUnifiedSpotEPL, http.MethodGet, "unified/transferable", params, nil, &resp)
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

// GetMultipleTransferableAmountForUnifiedAccounts batch queries the maximum transferable amount of up to 100 currencies
// in the unified account. A withdrawal changes the transferable amount of every currency.
func (e *Exchange) GetMultipleTransferableAmountForUnifiedAccounts(ctx context.Context, currencies ...currency.Code) ([]*CurrencyAndAmountResponse, error) {
	currencyList, err := currencyListParam(currencies, 100)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("currencies", currencyList)
	var resp []*CurrencyAndAmountResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateUnifiedSpotEPL, http.MethodGet, "unified/transferables", params, nil, &resp)
}

// GetBatchUnifiedAccountMaximumBorrowableAmount batch queries the maximum borrowable amount of up to 10 currencies in the
// unified account
func (e *Exchange) GetBatchUnifiedAccountMaximumBorrowableAmount(ctx context.Context, currencies ...currency.Code) ([]*CurrencyAndAmountResponse, error) {
	currencyList, err := currencyListParam(currencies, 10)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("currencies", currencyList)
	var resp []*CurrencyAndAmountResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, privateUnifiedSpotEPL, http.MethodGet, "unified/batch_borrowable", params, nil, &resp)
}

// BorrowOrRepay borrows or repays a currency in the unified account. A borrowed amount must not be below the currency's
// minimum borrowing threshold or exceed the borrowing limits set by the platform and user. Loan interest is deducted
// from the account automatically. Set RepaidAll, without an amount, to repay a loan in full.
func (e *Exchange) BorrowOrRepay(ctx context.Context, arg *BorrowOrRepayRequest) (*BorrowOrRepayResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	switch arg.Type {
	case "":
		return nil, errLoanTypeIsRequired
	case loanTypeBorrow, loanTypeRepay:
	default:
		return nil, fmt.Errorf("%w: %q must be \"borrow\" or \"repay\"", errInvalidLoanType, arg.Type)
	}
	if arg.RepaidAll {
		if arg.Type != loanTypeRepay {
			return nil, errInvalidRepaidAllOperation
		}
		if arg.Amount != 0 {
			return nil, errAmountOverriddenByRepaidAll
		}
	} else if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w: borrow or repay amount must be greater than 0", order.ErrAmountIsInvalid)
	}
	var resp *BorrowOrRepayResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedBorrowOrRepayEPL, http.MethodPost, "unified/loans", nil, arg, &resp)
}

// GetLoans retrieves the unified account's outstanding loans. Loan type is "platform" or "margin"; both are returned
// when it is empty.
func (e *Exchange) GetLoans(ctx context.Context, arg *UnifiedLoansRequest) ([]*UnifiedLoanResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Type != "" && arg.Type != "platform" && arg.Type != "margin" {
		return nil, fmt.Errorf("%w: %q must be \"platform\" or \"margin\"", errInvalidLoanType, arg.Type)
	}
	var resp []*UnifiedLoanResponse
	return resp, e.getLoans(ctx, unifiedLoansEPL, "unified/loans", arg, &resp)
}

// GetLoanRecords retrieves the unified account's borrow and repayment records. Record type is "borrow" or "repay"; both
// are returned when it is empty.
func (e *Exchange) GetLoanRecords(ctx context.Context, arg *UnifiedLoansRequest) ([]*UnifiedLoanRecordResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Type != "" && arg.Type != loanTypeBorrow && arg.Type != loanTypeRepay {
		return nil, fmt.Errorf("%w: %q must be \"borrow\" or \"repay\"", errInvalidLoanType, arg.Type)
	}
	var resp []*UnifiedLoanRecordResponse
	return resp, e.getLoans(ctx, unifiedLoanRecordsEPL, "unified/loan_records", arg, &resp)
}

// getLoans sends a unified loan or loan record query, which share their filters and paging
func (e *Exchange) getLoans(ctx context.Context, epl request.EndpointLimit, path string, arg *UnifiedLoansRequest, result any) error {
	if arg.Limit > 100 {
		return fmt.Errorf("%w: maximum 100", errInvalidLimit)
	}
	params := url.Values{}
	if !arg.Currency.IsEmpty() {
		params.Set("currency", arg.Currency.String())
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Type != "" {
		params.Set("type", arg.Type)
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, epl, http.MethodGet, path, params, nil, result)
}

// GetInterestDeductionRecords retrieves the unified account's interest deduction records
func (e *Exchange) GetInterestDeductionRecords(ctx context.Context, arg *UnifiedInterestRecordsRequest) ([]*InterestDeductionRecordResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Limit > 100 {
		return nil, fmt.Errorf("%w: maximum 100", errInvalidLimit)
	}
	if arg.Type != "" && arg.Type != "platform" && arg.Type != "margin" {
		return nil, fmt.Errorf("%w: %q must be \"platform\" or \"margin\"", errInvalidLoanType, arg.Type)
	}
	params := url.Values{}
	if err := setUnixTimeRangeParams(&params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if !arg.Currency.IsEmpty() {
		params.Set("currency", arg.Currency.String())
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Type != "" {
		params.Set("type", arg.Type)
	}
	var resp []*InterestDeductionRecordResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedInterestRecordsEPL, http.MethodGet, "unified/interest_records", params, nil, &resp)
}

// SetUnifiedAccountMode sets the unified account mode, optionally switching the mode's settings in the same request.
// Modes are "classic", "multi_currency" (cross-currency margin), "portfolio" and "single_currency".
func (e *Exchange) SetUnifiedAccountMode(ctx context.Context, arg *UnifiedAccountModeRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.Mode == "" {
		return errMissingUnifiedAccountMode
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedUpdateUnifiedModeEPL, http.MethodPut, "unified/unified_mode", nil, arg, nil)
}

// GetUnifiedAccountMode query mode of the unified account
func (e *Exchange) GetUnifiedAccountMode(ctx context.Context) (*UnifiedAccountModeResponse, error) {
	var resp *UnifiedAccountModeResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedGetUnifiedModeEPL, http.MethodGet, "unified/unified_mode", nil, nil, &resp)
}

// GetUnifiedAccountEstimatedInterestRate retrieves the estimated hourly interest rate of up to 10 currencies in the
// unified account. Rates change hourly with lending depth, and an unsupported currency's rate is returned as zero.
func (e *Exchange) GetUnifiedAccountEstimatedInterestRate(ctx context.Context, currencies []currency.Code) (map[currency.Code]types.Number, error) {
	currencyList, err := currencyListParam(currencies, 10)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("currencies", currencyList)
	var resp map[currency.Code]types.Number
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedEstimateRateEPL, http.MethodGet, "unified/estimate_rate", params, nil, &resp)
}

// GetUnifiedAccountTiered retrieves the unified account's collateral discount tiers
func (e *Exchange) GetUnifiedAccountTiered(ctx context.Context) ([]*UnifiedAccountTieredDetailResponse, error) {
	var resp []*UnifiedAccountTieredDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unifiedCurrencyDiscountTiersEPL, "unified/currency_discount_tiers", &resp)
}

// GetUnifiedAccountTieredLoanMargin retrieves the unified account's borrowing margin tiers
func (e *Exchange) GetUnifiedAccountTieredLoanMargin(ctx context.Context) ([]*UnifiedAccountLoanMarginResponse, error) {
	var resp []*UnifiedAccountLoanMarginResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unifiedLoanMarginTiersEPL, "unified/loan_margin_tiers", &resp)
}

// CalculatePortfolioMargin calculates the portfolio margin requirements of a simulated portfolio of positions and
// orders. Market orders are not supported.
func (e *Exchange) CalculatePortfolioMargin(ctx context.Context, arg *PortfolioMarginCalculatorRequest) (*PortfolioMarginCalculationResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	for _, sb := range arg.SpotBalances {
		if err := common.NilGuard(sb); err != nil {
			return nil, err
		}
		if sb.Currency.IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
		if sb.Equity == 0 {
			return nil, errInvalidEquity
		}
	}
	for _, so := range arg.SpotOrders {
		if err := common.NilGuard(so); err != nil {
			return nil, err
		}
		if so.CurrencyPair.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if so.OrderPrice <= 0 {
			return nil, fmt.Errorf("%w: order price must be greater than 0", limits.ErrPriceBelowMin)
		}
		if so.Left <= 0 {
			return nil, fmt.Errorf("%w: left, unfilled quantity size must be greater than 0", errInvalidOrderSize)
		}
		if so.Side != order.Buy && so.Side != order.Sell {
			return nil, fmt.Errorf("%w: %q must be buy or sell", order.ErrSideIsInvalid, so.Side)
		}
	}
	for _, fPosition := range arg.FuturesPositions {
		if err := common.NilGuard(fPosition); err != nil {
			return nil, err
		}
		if fPosition.Contract.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if fPosition.Size == 0 {
			return nil, fmt.Errorf("%w: size must not be 0", errInvalidOrderSize)
		}
	}
	for _, fOrder := range arg.FuturesOrders {
		if err := common.NilGuard(fOrder); err != nil {
			return nil, err
		}
		if fOrder.Contract.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if fOrder.Size == 0 || fOrder.Left == 0 {
			return nil, fmt.Errorf("%w: size and left, the unfilled quantity, must not be 0", errInvalidOrderSize)
		}
	}
	for _, op := range arg.OptionsPositions {
		if err := common.NilGuard(op); err != nil {
			return nil, err
		}
		if op.OptionsName.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if op.Size == 0 {
			return nil, fmt.Errorf("%w: size must not be 0", errInvalidOrderSize)
		}
	}
	for _, oo := range arg.OptionsOrders {
		if err := common.NilGuard(oo); err != nil {
			return nil, err
		}
		if oo.OptionsName.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if oo.Size == 0 || oo.Left == 0 {
			return nil, fmt.Errorf("%w: size and left, the unfilled quantity, must not be 0", errInvalidOrderSize)
		}
	}
	var resp *PortfolioMarginCalculationResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedPortfolioCalculatorEPL, http.MethodPost, "unified/portfolio_calculator", nil, arg, &resp)
}

// SetAllLoanCurrenciesLeverage sets the leverage of every currency the unified account can borrow and returns the
// currencies that could not be updated. Currencies with outstanding loans keep their leverage, a leverage above a
// currency's limit is capped at that limit, and successful updates are not rolled back when others fail.
func (e *Exchange) SetAllLoanCurrenciesLeverage(ctx context.Context, leverage float64) ([]*LeverageFailedCurrencyResponse, error) {
	if leverage <= 0 {
		return nil, fmt.Errorf("%w: leverage must be greater than 0", errInvalidLeverage)
	}
	var resp []*LeverageFailedCurrencyResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedLeverageUserSettingEPL, http.MethodPost, "unified/leverage/user_setting", nil, &UnifiedLeverageRequest{Leverage: types.Number(leverage)}, &resp)
}

// GetMaxMinCurrencyLeverage retrieves the maximum and minimum currency leverage that can be set
// for the unified account.
func (e *Exchange) GetMaxMinCurrencyLeverage(ctx context.Context, ccy currency.Code) (*UnifiedCurrencyLeverageConfigResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var resp *UnifiedCurrencyLeverageConfigResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedLeverageUserCurrencyConfigEPL, http.MethodGet, "unified/leverage/user_currency_config", params, nil, &resp)
}

// GetUserCurrencyLeverage retrieves the leverage multiplier settings for borrowing currencies
// in the unified account. If currency is empty, all currencies are returned.
func (e *Exchange) GetUserCurrencyLeverage(ctx context.Context, ccy currency.Code) ([]*UnifiedLeverageSettingResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var resp []*UnifiedLeverageSettingResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedGetLeverageUserCurrencySettingEPL, http.MethodGet, "unified/leverage/user_currency_setting", params, nil, &resp)
}

// SetLoanCurrencyLeverage sets the loan currency leverage multiplier for the unified account.
func (e *Exchange) SetLoanCurrencyLeverage(ctx context.Context, arg *UnifiedLeverageSettingResponse) error {
	if arg == nil {
		return common.ErrNilPointer
	}
	if arg.Currency.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if arg.Leverage <= 0 {
		return errInvalidLeverage
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedCreateLeverageUserCurrencySettingEPL, http.MethodPost, "unified/leverage/user_currency_setting", nil, arg, nil)
}

// GetUnifiedLoanCurrencies retrieves the list of loan currencies supported by the unified account.
// Optionally filter by currency. This endpoint does not require authentication.
func (e *Exchange) GetUnifiedLoanCurrencies(ctx context.Context, ccy currency.Code) ([]*UnifiedLoanCurrencyResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var resp []*UnifiedLoanCurrencyResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unifiedCurrenciesEPL, common.EncodeURLValues("unified/currencies", params), &resp)
}

// GetUnifiedHistoricalLendingRates retrieves a currency's hourly lending rates in the unified account, newest first.
// This endpoint does not require authentication.
func (e *Exchange) GetUnifiedHistoricalLendingRates(ctx context.Context, arg *UnifiedHistoricalLendingRatesRequest) (*UnifiedHistoricalLendingRateResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Limit > 100 {
		return nil, fmt.Errorf("%w: maximum 100", errInvalidLimit)
	}
	params := url.Values{}
	params.Set("currency", arg.Currency.String())
	if arg.Tier != "" {
		params.Set("tier", arg.Tier)
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp *UnifiedHistoricalLendingRateResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, unifiedHistoryLoanRateEPL, common.EncodeURLValues("unified/history_loan_rate", params), &resp)
}

// SetUnifiedCollateralCurrency sets collateral currencies for the unified account.
// collateralType: 0 = all currencies, 1 = custom currencies.
// When collateralType is 0, enableList and disableList are invalid.
func (e *Exchange) SetUnifiedCollateralCurrency(ctx context.Context, arg *UnifiedSetCollateralRequest) (*UnifiedSetCollateralResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.CollateralType > 1 {
		return nil, fmt.Errorf("%w: %d must be 0 for all currencies or 1 for custom currencies", errInvalidCollateralType, arg.CollateralType)
	}
	var resp *UnifiedSetCollateralResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedCollateralCurrenciesEPL, http.MethodPost, "unified/collateral_currencies", nil, arg, &resp)
}

// EstimateQuickRepaymentDetails retrieves the estimated quick repayment details for the unified account.
// Available for unified account cross-currency margin mode and portfolio margin mode.
func (e *Exchange) EstimateQuickRepaymentDetails(ctx context.Context) (*UnifiedQuickEstimateRepaymentResponse, error) {
	var resp *UnifiedQuickEstimateRepaymentResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedEstimateQuickRepaymentEPL, http.MethodGet, "unified/estimated_quick_repayment", nil, nil, &resp)
}

// QuickRepayment performs a quick repayment for the unified account.
// Available for unified account cross-currency margin mode and portfolio margin mode.
func (e *Exchange) QuickRepayment(ctx context.Context, arg *UnifiedQuickRepaymentRequest) (*UnifiedQuickRepaymentResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if len(arg.DebtCurrencies) == 0 {
		return nil, fmt.Errorf("%w: debt currencies are required for quick repayment", currency.ErrCurrencyCodesEmpty)
	}
	if len(arg.AvailableCurrencies) == 0 {
		return nil, fmt.Errorf("%w: available currencies are required for quick repayment", currency.ErrCurrencyCodesEmpty)
	}
	if slices.ContainsFunc(arg.DebtCurrencies, currency.Code.IsEmpty) || slices.ContainsFunc(arg.AvailableCurrencies, currency.Code.IsEmpty) {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *UnifiedQuickRepaymentResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedQuickRepaymentEPL, http.MethodPost, "unified/quick_repayment", nil, arg, &resp)
}

// SetUnifiedDeltaNeutral enables or disables the account Delta-neutral strategy mode and returns the resulting state.
// Enabling it requires VIP level 4 or above and the cross-currency margin mode.
func (e *Exchange) SetUnifiedDeltaNeutral(ctx context.Context, enabled bool) (*DeltaNeutralResponse, error) {
	var resp *DeltaNeutralResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedSetDeltaNeutralEPL, http.MethodPost, "unified/delta_neutral", nil, &DeltaNeutralRequest{Enabled: enabled}, &resp)
}

// GetUnifiedDeltaNeutral retrieves whether the account Delta-neutral strategy mode is enabled
func (e *Exchange) GetUnifiedDeltaNeutral(ctx context.Context) (*DeltaNeutralResponse, error) {
	var resp *DeltaNeutralResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, unifiedGetDeltaNeutralEPL, http.MethodGet, "unified/delta_neutral", nil, nil, &resp)
}

// CreateBatchOrders places up to 10 orders on each of up to 4 currency pairs in a single request
// Every order must carry a custom text ID and use the same account; Gate reports each order's outcome individually
func (e *Exchange) CreateBatchOrders(ctx context.Context, args []CreateOrderRequest) ([]*SpotOrderResponse, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	ordersPerPair := make(map[string]int)
	for x := range args {
		if x != 0 && args[x-1].Account != args[x].Account {
			return nil, errDifferentAccount
		}
		if args[x].Text == "" {
			return nil, order.ErrClientOrderIDMustBeSet
		} else if !strings.HasPrefix(args[x].Text, "t-") {
			return nil, errInvalidTextPrefix
		}
		if args[x].CurrencyPair.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if args[x].Side != order.Buy.Lower() && args[x].Side != order.Sell.Lower() {
			return nil, order.ErrSideIsInvalid
		}
		if args[x].Account != asset.Spot &&
			args[x].Account != asset.CrossMargin &&
			args[x].Account != asset.Margin {
			return nil, fmt.Errorf("%w: only spot, margin, and cross_margin are allowed", asset.ErrInvalidAsset)
		}
		if args[x].Amount <= 0 {
			return nil, order.ErrAmountIsInvalid
		}
		if err := validateSpotOrderPrice(args[x].Type, args[x].Price.Float64()); err != nil {
			return nil, err
		}
		pair := args[x].CurrencyPair.String()
		ordersPerPair[pair]++
		if ordersPerPair[pair] > 10 {
			return nil, fmt.Errorf("%w: at most 10 orders are allowed per currency pair, %s has more", errTooManyBatchOrders, pair)
		}
	}
	if len(ordersPerPair) > 4 {
		return nil, fmt.Errorf("%w: at most 4 currency pairs are allowed, %d supplied", errTooManyBatchOrders, len(ordersPerPair))
	}
	var response []*SpotOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotBatchOrdersEPL, http.MethodPost, "spot/batch_orders", nil, &args, &response)
}

// validateSpotOrderPrice requires a positive price for limit orders, which is Gate's default order type, while market
// orders ignore the price
func validateSpotOrderPrice(orderType string, price float64) error {
	if price < 0 || (price == 0 && orderType != order.Market.Lower()) {
		return fmt.Errorf("%w: %v", limits.ErrPriceBelowMin, price)
	}
	return nil
}

// GetSpotOpenOrders retrieves open orders across all currency pairs, optionally for a single account (spot, margin,
// cross_margin or unified)
// Pagination applies per currency pair, not to the number of currency pairs returned.
// Gate rejects an account filter of cross_margin for some account types, so callers needing every account type should
// leave account empty and filter the returned orders by their Account field.
func (e *Exchange) GetSpotOpenOrders(ctx context.Context, page, limit uint64, account string) ([]*SpotOrdersDetailResponse, error) {
	params := url.Values{}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if account != "" {
		params.Set("account", account)
	}
	var response []*SpotOrdersDetailResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetOpenOrdersEPL, http.MethodGet, "spot/open_orders", params, nil, &response)
}

// SpotClosePositionWhenCrossCurrencyDisabled places a cross margin buy order to close a position in a currency whose
// cross-currency trading is disabled
func (e *Exchange) SpotClosePositionWhenCrossCurrencyDisabled(ctx context.Context, arg *ClosePositionRequest) (*SpotOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.CurrencyPair.IsInvalid() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.Price <= 0 {
		return nil, limits.ErrPriceBelowMin
	}
	var response *SpotOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotClosePositionEPL, http.MethodPost, "spot/cross_liquidate_orders", nil, arg, &response)
}

// PlaceSpotOrder places an order using the spot, margin or cross margin account selected by Account
func (e *Exchange) PlaceSpotOrder(ctx context.Context, arg *CreateOrderRequest) (*SpotOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.CurrencyPair.IsInvalid() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side != order.Buy.Lower() && arg.Side != order.Sell.Lower() {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Account != asset.Spot &&
		arg.Account != asset.CrossMargin &&
		arg.Account != asset.Margin {
		return nil, fmt.Errorf("%w: only 'spot', 'cross_margin', and 'margin' are allowed", asset.ErrInvalidAsset)
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if err := validateSpotOrderPrice(arg.Type, arg.Price.Float64()); err != nil {
		return nil, err
	}
	var response *SpotOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotPlaceOrderEPL, http.MethodPost, "spot/orders", nil, arg, &response)
}

// GetSpotOrders retrieves open or finished spot orders
// Open orders require a currency pair and only support page and limit; side and the time range filter finished orders
func (e *Exchange) GetSpotOrders(ctx context.Context, arg *GetSpotOrdersRequest) ([]*SpotOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Status != statusOpen && arg.Status != statusFinished {
		return nil, fmt.Errorf("%w: %q, only 'open' and 'finished' are supported", errInvalidOrderStatus, arg.Status)
	}
	if arg.Status == statusOpen && arg.CurrencyPair.IsEmpty() {
		return nil, fmt.Errorf("%w: open orders require a currency pair", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	params.Set("status", arg.Status)
	if !arg.CurrencyPair.IsEmpty() {
		params.Set("currency_pair", arg.CurrencyPair.String())
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Account != "" {
		params.Set("account", arg.Account)
	}
	if err := setUnixTimeRangeParams(&params, arg.From, arg.To); err != nil {
		return nil, err
	}
	switch arg.Side {
	case order.UnknownSide, order.AnySide:
	case order.Buy, order.Sell:
		params.Set("side", arg.Side.Lower())
	default:
		return nil, fmt.Errorf("%w: %q", order.ErrSideIsInvalid, arg.Side)
	}
	var response []*SpotOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetOrdersEPL, http.MethodGet, "spot/orders", params, nil, &response)
}

// CancelAllOpenOrdersSpecifiedCurrencyPair cancels all open orders, optionally limited to a currency pair, side,
// account and unified market quote currency; Gate cancels the orders of every pair when the currency pair is empty
func (e *Exchange) CancelAllOpenOrdersSpecifiedCurrencyPair(ctx context.Context, arg *CancelAllSpotOrdersRequest) ([]*SpotOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if !arg.CurrencyPair.IsEmpty() {
		params.Set("currency_pair", arg.CurrencyPair.String())
	}
	switch arg.Side {
	case order.Buy, order.Sell:
		params.Set("side", arg.Side.Lower())
	case order.UnknownSide, order.AnySide:
		// Do not set side. Gate will cancel both bids and asks.
	default:
		return nil, fmt.Errorf("%w: %q", order.ErrSideIsInvalid, arg.Side)
	}
	if arg.Account != "" {
		params.Set("account", arg.Account)
	}
	if !arg.TradeQuote.IsEmpty() {
		params.Set("trade_quote", arg.TradeQuote.String())
	}
	if arg.ActionMode != "" {
		params.Set("action_mode", arg.ActionMode)
	}
	var response []*SpotOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelAllOpenOrdersEPL, http.MethodDelete, "spot/orders", params, nil, &response)
}

// CancelBatchOrdersWithIDList cancels up to 20 orders, which may span several currency pairs, by order ID
func (e *Exchange) CancelBatchOrdersWithIDList(ctx context.Context, args []CancelOrderByIDRequest) ([]*CancelOrderByIDResponse, error) {
	if len(args) == 0 {
		return nil, errNoValidParameterPassed
	} else if len(args) > 20 {
		return nil, fmt.Errorf("%w maximum order size to cancel is 20", errInvalidOrderSize)
	}
	for x := range args {
		if args[x].CurrencyPair.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if args[x].ID == "" {
			return nil, order.ErrOrderIDNotSet
		}
	}
	var response []*CancelOrderByIDResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelBatchOrdersEPL, http.MethodPost, "spot/cancel_batch_orders", nil, &args, &response)
}

// GetSpotOrder retrieves a single spot order by order ID or custom text ID
func (e *Exchange) GetSpotOrder(ctx context.Context, orderID string, currencyPair currency.Pair, a asset.Item) (*SpotOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if currencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("currency_pair", currencyPair.String())
	if accountType := a.String(); accountType != "" {
		params.Set("account", accountType)
	}
	var response *SpotOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetOrderEPL, http.MethodGet, "spot/orders/"+orderID, params, nil, &response)
}

// AmendSpotOrder amends the price, amount or take profit and stop loss of an open order
// Spot, unified and margin account orders are amended by default; cross margin orders require an account of
// cross_margin. Only reducing the amount keeps the order's matching priority.
func (e *Exchange) AmendSpotOrder(ctx context.Context, orderID string, arg *AmendSpotOrderRequest) (*SpotOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if err := arg.validate(); err != nil {
		return nil, err
	}
	var resp *SpotOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAmendOrderEPL, http.MethodPatch, "spot/orders/"+orderID, nil, arg, &resp)
}

// validate checks the fields Gate requires to amend a spot order, so single and batch amendments share the same rules
func (a *AmendSpotOrderRequest) validate() error {
	if err := common.NilGuard(a); err != nil {
		return err
	}
	if a.CurrencyPair.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if a.Price <= 0 && a.Amount <= 0 {
		return fmt.Errorf("%w %w: either price or amount has to be set", order.ErrAmountIsInvalid, limits.ErrPriceBelowMin)
	}
	return nil
}

// CancelSingleSpotOrder cancels a single order
// Spot, unified and margin account orders are cancelled by default; cross margin orders require isCrossMarginAccount
func (e *Exchange) CancelSingleSpotOrder(ctx context.Context, orderID string, currencyPair currency.Pair, isCrossMarginAccount bool) (*SpotOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if currencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if isCrossMarginAccount {
		params.Set("account", crossMarginAccount)
	}
	params.Set("currency_pair", currencyPair.String())
	var response *SpotOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelSingleOrderEPL, http.MethodDelete, "spot/orders/"+orderID, params, nil, &response)
}

// GetMySpotTradingHistory retrieves the account's spot trades
// Without a time range Gate returns the last 7 days, and a range must not exceed 30 days
func (e *Exchange) GetMySpotTradingHistory(ctx context.Context, arg *GetMySpotTradingHistoryRequest) ([]*SpotPersonalTradeHistoryResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.OrderID != "" && arg.CurrencyPair.IsEmpty() {
		return nil, fmt.Errorf("%w: filtering by order ID requires a currency pair", currency.ErrCurrencyPairEmpty)
	}
	// Gate receives whole seconds, so the 30 day cap is checked in the units sent
	if !arg.From.IsZero() && !arg.To.IsZero() && arg.To.Unix()-arg.From.Unix() > int64(30*24*time.Hour/time.Second) {
		return nil, fmt.Errorf("%w: trading history range must not exceed 30 days", errTimeRangeTooLong)
	}
	params := url.Values{}
	if err := setUnixTimeRangeParams(&params, arg.From, arg.To); err != nil {
		return nil, err
	}
	if !arg.CurrencyPair.IsEmpty() {
		params.Set("currency_pair", arg.CurrencyPair.String())
	}
	if arg.OrderID != "" {
		params.Set("order_id", arg.OrderID)
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var response []*SpotPersonalTradeHistoryResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotTradingHistoryEPL, http.MethodGet, "spot/my_trades", params, nil, &response)
}

// GetServerTime retrieves current server time
func (e *Exchange) GetServerTime(ctx context.Context, _ asset.Item) (time.Time, error) {
	var resp serverTimeReply
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, publicGetServerTimeEPL, "spot/time", &resp); err != nil {
		return time.Time{}, err
	}
	return resp.ServerTime.Time(), nil
}

// CountdownCancelSpotOrders sets, renews or stops a countdown that cancels open spot orders when it expires
// Calling it again before the timeout renews the countdown, and a timeout of 0 stops it. Orders on every currency pair
// are cancelled when the currency pair is empty.
func (e *Exchange) CountdownCancelSpotOrders(ctx context.Context, arg *CountdownCancelOrderRequest) (*TriggerTimeResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Timeout > 0 && arg.Timeout < 5 {
		return nil, errInvalidCountdown
	}
	var response *TriggerTimeResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCountdownCancelEPL, http.MethodPost, "spot/countdown_cancel_all", nil, arg, &response)
}

// AmendBatchSpotOrders amends up to 5 open orders in one request
// Gate amends the orders in the order supplied, continues past failures and reports each order's outcome individually
func (e *Exchange) AmendBatchSpotOrders(ctx context.Context, args []*BatchAmendSpotOrderRequest) ([]*SpotOrderResponse, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	if len(args) > 5 {
		return nil, fmt.Errorf("%w: at most 5 orders can be amended at once, %d supplied", errTooManyBatchOrders, len(args))
	}
	for _, arg := range args {
		if err := common.NilGuard(arg); err != nil {
			return nil, err
		}
		if arg.OrderID == "" {
			return nil, order.ErrOrderIDNotSet
		}
		if err := arg.validate(); err != nil {
			return nil, err
		}
	}
	var resp []*SpotOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotAmendBatchOrdersEPL, http.MethodPost, "spot/amend_batch_orders", nil, args, &resp)
}

// GetSpotInsuranceHistory retrieves the historical balances of the margin or unified account insurance fund
func (e *Exchange) GetSpotInsuranceHistory(ctx context.Context, arg *GetSpotInsuranceHistoryRequest) ([]*SpotInsuranceHistoryResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Business == "" {
		return nil, errBusinessTypeRequired
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := common.StartEndTimeCheck(arg.From, arg.To); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("business", arg.Business)
	params.Set("currency", arg.Currency.String())
	params.Set("from", strconv.FormatInt(arg.From.Unix(), 10))
	params.Set("to", strconv.FormatInt(arg.To.Unix(), 10))
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []*SpotInsuranceHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicInsuranceHistorySpotEPL, common.EncodeURLValues("spot/insurance_history", params), &resp)
}

// CreatePriceTriggeredOrder creates a price-triggered order, which places the Put order once the market price meets
// the Trigger rule
// Put.Account defaults to normal, the spot account; margin and unified are also accepted
func (e *Exchange) CreatePriceTriggeredOrder(ctx context.Context, arg *PriceTriggeredOrderRequest) (*TriggerOrderIDResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Market.IsEmpty() {
		return nil, fmt.Errorf("%w: market is required", currency.ErrCurrencyPairEmpty)
	}
	if arg.Trigger.Price <= 0 {
		return nil, fmt.Errorf("%w: trigger price must be positive, got %v", limits.ErrPriceBelowMin, arg.Trigger.Price)
	}
	if arg.Trigger.Rule != ">=" && arg.Trigger.Rule != "<=" {
		return nil, fmt.Errorf("%w: %q, expected '>=' or '<='", errInvalidTriggerRule, arg.Trigger.Rule)
	}
	arg.Put.Side = strings.ToLower(arg.Put.Side)
	if arg.Put.Side != order.Buy.Lower() && arg.Put.Side != order.Sell.Lower() {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Put.Price < 0 {
		return nil, fmt.Errorf("%w: put price must not be negative", limits.ErrPriceBelowMin)
	}
	if arg.Put.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	arg.Put.TimeInForce = strings.ToLower(arg.Put.TimeInForce)
	if arg.Put.TimeInForce != gtcTIF && arg.Put.TimeInForce != iocTIF {
		return nil, fmt.Errorf("%w: %q, only 'gtc' and 'ioc' are supported", order.ErrInvalidTimeInForce, arg.Put.TimeInForce)
	}
	arg.Put.Type = strings.ToLower(arg.Put.Type)
	arg.Put.Account = strings.ToLower(arg.Put.Account)
	if arg.Put.Account == "" {
		arg.Put.Account = "normal"
	}
	var response *TriggerOrderIDResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCreateTriggerOrderEPL, http.MethodPost, "spot/price_orders", nil, arg, &response)
}

// GetPriceTriggeredOrderList retrieves open or finished price-triggered orders, optionally filtered by market and
// account (normal, margin or unified)
func (e *Exchange) GetPriceTriggeredOrderList(ctx context.Context, status string, market currency.Pair, account string, offset, limit uint64) ([]*SpotPriceTriggeredOrderResponse, error) {
	if status != statusOpen && status != statusFinished {
		return nil, fmt.Errorf("%w status %s", errInvalidOrderStatus, status)
	}
	params := url.Values{}
	params.Set("status", status)
	if market.IsPopulated() {
		params.Set("market", market.String())
	}
	if account != "" {
		params.Set("account", account)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	var response []*SpotPriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetTriggerOrderListEPL, http.MethodGet, "spot/price_orders", params, nil, &response)
}

// CancelMultipleSpotOpenOrders cancels all open price-triggered orders, optionally limited to a market and account
// (normal, margin or unified)
func (e *Exchange) CancelMultipleSpotOpenOrders(ctx context.Context, market currency.Pair, account string) ([]*SpotPriceTriggeredOrderResponse, error) {
	params := url.Values{}
	if market.IsPopulated() {
		params.Set("market", market.String())
	}
	if account != "" {
		params.Set("account", account)
	}
	var response []*SpotPriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelTriggerOrdersEPL, http.MethodDelete, "spot/price_orders", params, nil, &response)
}

// GetSinglePriceTriggeredOrder retrieves a single price-triggered order
func (e *Exchange) GetSinglePriceTriggeredOrder(ctx context.Context, orderID string) (*SpotPriceTriggeredOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var response *SpotPriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetTriggerOrderEPL, http.MethodGet, "spot/price_orders/"+orderID, nil, nil, &response)
}

// CancelPriceTriggeredOrder cancels a single price-triggered order
func (e *Exchange) CancelPriceTriggeredOrder(ctx context.Context, orderID string) (*SpotPriceTriggeredOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var response *SpotPriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelTriggerOrderEPL, http.MethodDelete, "spot/price_orders/"+orderID, nil, nil, &response)
}

// CreateSpotPOVOrder creates a percentage of volume (POV) order, which trades a target share of market volume until
// it fills or its time to live expires
func (e *Exchange) CreateSpotPOVOrder(ctx context.Context, arg *SpotPOVOrderRequest) (*SpotPOVOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.CurrencyPair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side != order.Buy.Lower() && arg.Side != order.Sell.Lower() {
		return nil, fmt.Errorf("%w: %q", order.ErrSideIsInvalid, arg.Side)
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if arg.ParticipationRate == 0 {
		return nil, errParticipationRateRequired
	}
	if arg.TimeToLive == "" {
		return nil, errTimeToLiveRequired
	}
	var resp *SpotPOVOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCreatePOVOrderEPL, http.MethodPost, "spot/pov_orders", nil, arg, &resp)
}

// GetSpotPOVOrders retrieves open or finished POV orders
func (e *Exchange) GetSpotPOVOrders(ctx context.Context, arg *GetSpotPOVOrdersRequest) ([]*SpotPOVOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Status != statusOpen && arg.Status != statusFinished {
		return nil, fmt.Errorf("%w: %q, only 'open' and 'finished' are supported", errInvalidOrderStatus, arg.Status)
	}
	params := url.Values{}
	params.Set("status", arg.Status)
	if !arg.CurrencyPair.IsEmpty() {
		params.Set("currency_pair", arg.CurrencyPair.String())
	}
	switch arg.Side {
	case order.UnknownSide, order.AnySide:
	case order.Buy, order.Sell:
		params.Set("side", arg.Side.Lower())
	default:
		return nil, fmt.Errorf("%w: %q", order.ErrSideIsInvalid, arg.Side)
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []*SpotPOVOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetPOVOrdersEPL, http.MethodGet, "spot/pov_orders", params, nil, &resp)
}

// CancelSpotPOVOrders cancels all POV orders, or only those on the supplied currency pair
func (e *Exchange) CancelSpotPOVOrders(ctx context.Context, currencyPair currency.Pair) ([]*SpotPOVOrderResponse, error) {
	params := url.Values{}
	if !currencyPair.IsEmpty() {
		params.Set("currency_pair", currencyPair.String())
	}
	var resp []*SpotPOVOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelPOVOrdersEPL, http.MethodDelete, "spot/pov_orders", params, nil, &resp)
}

// GetSpotPOVOrder retrieves a single POV order by order ID or custom text ID
func (e *Exchange) GetSpotPOVOrder(ctx context.Context, orderID string) (*SpotPOVOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *SpotPOVOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotGetPOVOrderEPL, http.MethodGet, "spot/pov_orders/"+orderID, nil, nil, &resp)
}

// CancelSpotPOVOrder cancels a single POV order by order ID or custom text ID
func (e *Exchange) CancelSpotPOVOrder(ctx context.Context, orderID string) (*SpotPOVOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *SpotPOVOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, spotCancelPOVOrderEPL, http.MethodDelete, "spot/pov_orders/"+orderID, nil, nil, &resp)
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

// *********************************** Withdrawals ******************************

// WithdrawCurrency withdraws a currency. No fee is charged when the recipient address also belongs to Gate
func (e *Exchange) WithdrawCurrency(ctx context.Context, arg *WithdrawalRequest) (*WithdrawalResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w: withdrawal amount must be greater than zero", order.ErrAmountIsInvalid)
	}
	if arg.Currency.IsEmpty() {
		return nil, fmt.Errorf("%w: withdrawal currency must be specified", currency.ErrCurrencyCodeEmpty)
	}
	if arg.Chain == "" {
		return nil, errInvalidCurrencyChain
	}
	var resp *WithdrawalResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletWithdrawEPL, http.MethodPost, "withdrawals", nil, arg, &resp)
}

// UIDTransfer transfers funds between the spot accounts of two main accounts by UID. Neither party can be a sub-account
func (e *Exchange) UIDTransfer(ctx context.Context, arg *UIDTransferRequest) (*UIDTransferResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if arg.RecipientUserID == 0 {
		return nil, fmt.Errorf("%w: recipient UID", errUserIDRequired)
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *UIDTransferResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletUIDTransferEPL, http.MethodPost, "withdrawals/push", nil, arg, &resp)
}

// CancelWithdrawalWithSpecifiedID requests the cancellation of a withdrawal. The returned record status shows whether
// the cancellation took effect
func (e *Exchange) CancelWithdrawalWithSpecifiedID(ctx context.Context, withdrawalID string) (*WithdrawalResponse, error) {
	if withdrawalID == "" {
		return nil, errMissingWithdrawalID
	}
	var response *WithdrawalResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletCancelWithdrawEPL, http.MethodDelete, "withdrawals/"+withdrawalID, nil, nil, &response)
}

// *********************************** Wallet ***********************************

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

// ListCurrencyChain retrieves the chains supported by a currency
func (e *Exchange) ListCurrencyChain(ctx context.Context, ccy currency.Code) ([]*CurrencyChainResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var resp []*CurrencyChainResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicListCurrencyChainEPL, common.EncodeURLValues("wallet/currency_chains", params), &resp)
}

// GenerateCurrencyDepositAddress generates a deposit address for a currency
func (e *Exchange) GenerateCurrencyDepositAddress(ctx context.Context, ccy currency.Code) (*CurrencyDepositAddressInfoResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var response *CurrencyDepositAddressInfoResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletDepositAddressEPL, http.MethodGet, "wallet/deposit_address", params, nil, &response)
}

// GetWithdrawalRecords retrieves withdrawal records. The time range cannot exceed 30 days and defaults to the last 7 days
func (e *Exchange) GetWithdrawalRecords(ctx context.Context, arg *WithdrawalRecordsRequest) ([]*WithdrawalResponse, error) {
	params := url.Values{}
	if arg != nil {
		if err := setUnixTimeRangeParams(&params, arg.From, arg.To); err != nil {
			return nil, err
		}
		if !arg.Currency.IsEmpty() {
			params.Set("currency", arg.Currency.String())
		}
		if arg.WithdrawalID != "" {
			params.Set("withdraw_id", arg.WithdrawalID)
		}
		if arg.AssetClass != "" {
			params.Set("asset_class", arg.AssetClass)
		}
		if arg.WithdrawalOrderID != "" {
			params.Set("withdraw_order_id", arg.WithdrawalOrderID)
		}
		if arg.Limit > 0 {
			params.Set("limit", strconv.FormatUint(arg.Limit, 10))
		}
		if arg.Offset > 0 {
			params.Set("offset", strconv.FormatUint(arg.Offset, 10))
		}
	}
	var withdrawals []*WithdrawalResponse
	return withdrawals, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletWithdrawalRecordsEPL, http.MethodGet, "wallet/withdrawals", params, nil, &withdrawals)
}

// GetDepositRecords retrieves deposit records. The time range cannot exceed 30 days and defaults to the last 7 days
func (e *Exchange) GetDepositRecords(ctx context.Context, ccy currency.Code, from, to time.Time, offset, limit uint64) ([]*DepositRecordResponse, error) {
	params := url.Values{}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	var depositHistories []*DepositRecordResponse
	return depositHistories, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletDepositRecordsEPL, http.MethodGet, "wallet/deposits", params, nil, &depositHistories)
}

// TransferCurrency transfers funds between the user's trading accounts. Gate supports transfers between the spot
// account and the margin, perpetual futures, delivery futures and options accounts
func (e *Exchange) TransferCurrency(ctx context.Context, arg *TransferCurrencyRequest) (*TransactionIDResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	from, err := walletAccountType(arg.From)
	if err != nil {
		return nil, fmt.Errorf("%w: source account", err)
	}
	to, err := walletAccountType(arg.To)
	if err != nil {
		return nil, fmt.Errorf("%w: destination account", err)
	}
	if (from == "margin" || to == "margin") && arg.CurrencyPair.IsEmpty() {
		return nil, fmt.Errorf("%w: currency pair is required for margin account transfer", currency.ErrCurrencyPairEmpty)
	}
	if (from == futuresAccount || from == deliveryAccount || to == futuresAccount || to == deliveryAccount) && arg.Settle.IsEmpty() {
		return nil, fmt.Errorf("%w: settle is required for futures account transfer", currency.ErrCurrencyCodeEmpty)
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	var response *TransactionIDResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletTransferCurrencyEPL, http.MethodPost, "wallet/transfers", nil, arg, &response)
}

// GetTradingAccountTransfer retrieves a transfer between the user's trading accounts by the transaction ID returned
// from TransferCurrency
func (e *Exchange) GetTradingAccountTransfer(ctx context.Context, transactionID string) (*TradingAccountTransferResponse, error) {
	if transactionID == "" {
		return nil, errTransactionIDRequired
	}
	params := url.Values{}
	params.Set("tx_id", transactionID)
	var resp *TradingAccountTransferResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletGetTransferEPL, http.MethodGet, "wallet/transfers", params, nil, &resp)
}

// SubAccountTransfer transfers funds between the main account and a sub-account's trading account. The main
// account side always uses its spot account, whichever sub-account account is chosen
func (e *Exchange) SubAccountTransfer(ctx context.Context, arg *SubAccountTransferRequest) (*TransactionIDResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.SubAccount == "" {
		return nil, errInvalidSubAccount
	}
	arg.Direction = strings.ToLower(arg.Direction)
	if arg.Direction != "to" && arg.Direction != "from" {
		return nil, errInvalidTransferDirection
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	switch arg.SubAccountType {
	case "", spotAccount, futuresAccount, deliveryAccount, "options":
	default:
		return nil, fmt.Errorf("%w %q for SubAccountTransfer; Supported: [spot, futures, delivery, options]", asset.ErrNotSupported, arg.SubAccountType)
	}
	var resp *TransactionIDResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountTransferEPL, http.MethodPost, "wallet/sub_account_transfers", nil, arg, &resp)
}

// GetSubAccountTransferHistory retrieves transfer records between main and sub accounts.
// Records begin on 2020-04-10 and the query range cannot exceed 30 days.
func (e *Exchange) GetSubAccountTransferHistory(ctx context.Context, subAccountUserID string, from, to time.Time, offset, limit uint64) ([]*SubAccountTransferResponse, error) {
	if !from.IsZero() && from.Before(time.Date(2020, time.April, 10, 0, 0, 0, 0, time.UTC)) {
		return nil, errSubAccountTransferHistoryStart
	}
	params := url.Values{}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	// Only whole seconds are sent, so the limit applies to those
	if !from.IsZero() && !to.IsZero() && to.Truncate(time.Second).Sub(from.Truncate(time.Second)) > 30*24*time.Hour {
		return nil, errSubAccountTransferHistoryRange
	}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var response []*SubAccountTransferResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountTransferHistoryEPL, http.MethodGet, "wallet/sub_account_transfers", params, nil, &response)
}

// SubAccountTransferToSubAccount transfers funds between two sub-accounts of the same main account, using either the
// main account's API key or the source sub-account's
func (e *Exchange) SubAccountTransferToSubAccount(ctx context.Context, arg *InterSubAccountTransferRequest) (*TransactionIDResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.SubAccountFromUserID == "" {
		return nil, fmt.Errorf("%w: sub-account from user-id is required", errInvalidSubAccountUserID)
	}
	if err := checkSubAccountTransferAccountType(arg.SubAccountFromAssetType); err != nil {
		return nil, fmt.Errorf("%w: source sub-account", err)
	}
	if arg.SubAccountToUserID == "" {
		return nil, fmt.Errorf("%w: sub-account to user-id is required", errInvalidSubAccountUserID)
	}
	if err := checkSubAccountTransferAccountType(arg.SubAccountToAssetType); err != nil {
		return nil, fmt.Errorf("%w: destination sub-account", err)
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	var resp *TransactionIDResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountToSubAccountTransferEPL, http.MethodPost, "wallet/sub_account_to_sub_account", nil, arg, &resp)
}

// checkSubAccountTransferAccountType checks that a sub-account to sub-account transfer uses an account Gate supports
func checkSubAccountTransferAccountType(a asset.Item) error {
	account, err := walletAccountType(a)
	if err != nil {
		return err
	}
	if account != spotAccount && account != futuresAccount && account != deliveryAccount {
		return fmt.Errorf("%w: %q", asset.ErrNotSupported, a)
	}
	return nil
}

// GetTransferOrderStatus retrieves the status of a main and sub-account transfer by its client order ID or by the
// transaction ID the transfer returned
func (e *Exchange) GetTransferOrderStatus(ctx context.Context, clientOrderID, transactionID string) (*TransferStatusResponse, error) {
	if clientOrderID == "" && transactionID == "" {
		return nil, errTransferStatusIDRequired
	}
	params := url.Values{}
	if clientOrderID != "" {
		params.Set("client_order_id", clientOrderID)
	}
	if transactionID != "" {
		params.Set("tx_id", transactionID)
	}
	var resp *TransferStatusResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletTransferOrderStatusEPL, http.MethodGet, "wallet/order_status", params, nil, &resp)
}

// GetWithdrawalStatus retrieves withdrawal fees and limits, for all currencies when ccy is empty
func (e *Exchange) GetWithdrawalStatus(ctx context.Context, ccy currency.Code) ([]*WithdrawalStatusResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var response []*WithdrawalStatusResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletWithdrawStatusEPL, http.MethodGet, "wallet/withdraw_status", params, nil, &response)
}

// GetSubAccountBalances retrieves sub-account spot balances. subAccountUserID accepts several comma-separated user IDs
// and all sub-accounts are returned when it is empty
func (e *Exchange) GetSubAccountBalances(ctx context.Context, subAccountUserID string, page, limit uint64) ([]*SubAccountBalancesResponse, error) {
	params := url.Values{}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*SubAccountBalancesResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountBalancesEPL, http.MethodGet, "wallet/sub_account_balances", params, nil, &resp)
}

// GetSubAccountMarginBalances retrieves sub-account isolated margin balances. subAccountUserID accepts several
// comma-separated user IDs and all sub-accounts are returned when it is empty
func (e *Exchange) GetSubAccountMarginBalances(ctx context.Context, subAccountUserID string) ([]*SubAccountMarginBalanceResponse, error) {
	params := url.Values{}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	var response []*SubAccountMarginBalanceResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountMarginBalancesEPL, http.MethodGet, "wallet/sub_account_margin_balances", params, nil, &response)
}

// GetSubAccountFuturesBalances retrieves sub-account perpetual futures balances. subAccountUserID accepts several
// comma-separated user IDs and all sub-accounts are returned when it is empty
func (e *Exchange) GetSubAccountFuturesBalances(ctx context.Context, subAccountUserID string, settle currency.Code) ([]*FuturesSubAccountBalanceResponse, error) {
	params := url.Values{}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	if !settle.IsEmpty() {
		params.Set("settle", settle.Item.Lower)
	}
	var response []*FuturesSubAccountBalanceResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountFuturesBalancesEPL, http.MethodGet, "wallet/sub_account_futures_balances", params, nil, &response)
}

// GetSubAccountCrossMarginBalances retrieves sub-account cross margin balances. subAccountUserID accepts several
// comma-separated user IDs and all sub-accounts are returned when it is empty
func (e *Exchange) GetSubAccountCrossMarginBalances(ctx context.Context, subAccountUserID string) ([]*SubAccountCrossMarginInfoResponse, error) {
	params := url.Values{}
	if subAccountUserID != "" {
		params.Set("sub_uid", subAccountUserID)
	}
	var response []*SubAccountCrossMarginInfoResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSubAccountCrossMarginBalancesEPL, http.MethodGet, "wallet/sub_account_cross_margin_balances", params, nil, &response)
}

// GetSavedAddresses retrieves saved withdrawal addresses
func (e *Exchange) GetSavedAddresses(ctx context.Context, arg *SavedAddressesRequest) ([]*WalletSavedAddressResponse, error) {
	params := url.Values{}
	if arg != nil {
		if !arg.Currency.IsEmpty() {
			params.Set("currency", arg.Currency.String())
		}
		if arg.Chain != "" {
			params.Set("chain", arg.Chain)
		}
		if arg.Verified != "" {
			params.Set("verified", arg.Verified)
		}
		if arg.Limit > 0 {
			params.Set("limit", strconv.FormatUint(arg.Limit, 10))
		}
		if arg.Page > 0 {
			params.Set("page", strconv.FormatUint(arg.Page, 10))
		}
	}
	var response []*WalletSavedAddressResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSavedAddressesEPL, http.MethodGet, "wallet/saved_address", params, nil, &response)
}

// GetPersonalTradingFee retrieves the user's trading fee rates. Fees are usually the same for every currency pair and
// settlement currency, which narrow the query when set
func (e *Exchange) GetPersonalTradingFee(ctx context.Context, currencyPair currency.Pair, settle currency.Code) (*PersonalTradingFeeResponse, error) {
	params := url.Values{}
	if currencyPair.IsPopulated() {
		// specify a currency pair to retrieve precise fee rate
		params.Set("currency_pair", currencyPair.String())
	}
	if !settle.IsEmpty() {
		params.Set("settle", settle.Upper().String())
	}
	var response *PersonalTradingFeeResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletTradingFeeEPL, http.MethodGet, "wallet/fee", params, nil, &response)
}

// GetUsersTotalBalance retrieves the estimated value of every account converted to ccy, which accepts BTC, CNY, USD
// and USDT and defaults to USDT. Gate caches the values for up to a minute, so use the account balance endpoints for
// real-time figures
func (e *Exchange) GetUsersTotalBalance(ctx context.Context, ccy currency.Code) (*UsersAllAccountBalanceResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var response *UsersAllAccountBalanceResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletTotalBalanceEPL, http.MethodGet, "wallet/total_balance", params, nil, &response)
}

// GetConvertibleSmallBalanceCurrencies retrieves the small balances that can be converted into GT
func (e *Exchange) GetConvertibleSmallBalanceCurrencies(ctx context.Context) ([]*ConvertibleSmallBalanceResponse, error) {
	var resp []*ConvertibleSmallBalanceResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSmallBalancesEPL, http.MethodGet, "wallet/small_balance", nil, nil, &resp)
}

// ConvertSmallBalances converts small balances of provided currencies into GT.
// If no currencies are provided, all supported currencies will be converted
// See [this documentation](https://www.gate.io/help/guide/functional_guidelines/22367) for details and restrictions.
func (e *Exchange) ConvertSmallBalances(ctx context.Context, currs ...currency.Code) error {
	currencyList := make([]string, len(currs))
	for i := range currs {
		if currs[i].IsEmpty() {
			return currency.ErrCurrencyCodeEmpty
		}
		currencyList[i] = currs[i].Upper().String()
	}

	payload := &convertSmallBalancesRequest{
		Currency: currencyList,
		IsAll:    len(currs) == 0,
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletConvertSmallBalancesEPL, http.MethodPost, "wallet/small_balance", nil, payload, nil)
}

// GetConvertibleSmallBalanceCurrencyHistory retrieves small balance conversion records
func (e *Exchange) GetConvertibleSmallBalanceCurrencyHistory(ctx context.Context, ccy currency.Code, page, limit uint64) ([]*SmallCurrencyBalanceResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*SmallCurrencyBalanceResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletSmallBalanceHistoryEPL, http.MethodGet, "wallet/small_balance_history", params, nil, &resp)
}

// GetUIDTransferHistory retrieves UID transfer records. The time range defaults to the last 7 days
func (e *Exchange) GetUIDTransferHistory(ctx context.Context, arg *UIDTransferHistoryRequest) ([]*UIDTransferRecordResponse, error) {
	params := url.Values{}
	if arg != nil {
		if err := setUnixTimeRangeParams(&params, arg.From, arg.To); err != nil {
			return nil, err
		}
		if arg.ID != 0 {
			params.Set("id", strconv.FormatUint(arg.ID, 10))
		}
		if arg.Limit > 0 {
			params.Set("limit", strconv.FormatUint(arg.Limit, 10))
		}
		if arg.Offset > 0 {
			params.Set("offset", strconv.FormatUint(arg.Offset, 10))
		}
		if arg.TransactionType != "" {
			params.Set("transaction_type", arg.TransactionType)
		}
	}
	var resp []*UIDTransferRecordResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletUIDTransferHistoryEPL, http.MethodGet, "wallet/push", params, nil, &resp)
}

// GetLowCapExchangeList retrieves the low-liquidity or low-cap currencies that the API does not support, which must be
// managed through the website or app instead
func (e *Exchange) GetLowCapExchangeList(ctx context.Context) ([]currency.Code, error) {
	var resp []currency.Code
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, walletLowCapExchangeListEPL, http.MethodGet, "wallet/getLowCapExchangeList", nil, nil, &resp)
}

// ********************************* Margin *******************************************

// Gate no longer documents the cross margin endpoints below, having folded cross margin into the unified account. They
// still answer, and the wrapper relies on them for the CrossMargin asset.

// CurrencySupportedByCrossMargin currencies supported by cross margin.
func (e *Exchange) CurrencySupportedByCrossMargin(ctx context.Context) ([]*CrossMarginCurrenciesResponse, error) {
	var response []*CrossMarginCurrenciesResponse
	return response, e.SendHTTPRequest(ctx, exchange.RestSpot, marginSupportedCurrencyCrossListEPL, "margin/cross/currencies", &response)
}

// GetCrossMarginSupportedCurrencyDetail retrieve detail of one single currency supported by cross margin
func (e *Exchange) GetCrossMarginSupportedCurrencyDetail(ctx context.Context, ccy currency.Code) (*CrossMarginCurrenciesResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var response *CrossMarginCurrenciesResponse
	return response, e.SendHTTPRequest(ctx, exchange.RestSpot, marginSupportedCurrencyCrossEPL, "margin/cross/currencies/"+ccy.String(), &response)
}

// GetCrossMarginAccounts retrieve cross margin account
func (e *Exchange) GetCrossMarginAccounts(ctx context.Context) (*CrossMarginAccountResponse, error) {
	var response *CrossMarginAccountResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginAccountsEPL, http.MethodGet, "margin/cross/accounts", nil, nil, &response)
}

// GetCrossMarginAccountChangeHistory retrieve cross margin account change history
// Record time range cannot exceed 30 days
// possible values of account change types are "in", "out", "repay", "new_order", "order_fill", "referral_fee", "order_fee", and "unknown"
func (e *Exchange) GetCrossMarginAccountChangeHistory(ctx context.Context, ccy currency.Code, from, to time.Time, page, limit uint64, accountChangeType string) ([]*CrossMarginAccountHistoryItemResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if accountChangeType != "" {
		params.Set("type", accountChangeType)
	}
	var response []*CrossMarginAccountHistoryItemResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginAccountHistoryEPL, http.MethodGet, "margin/cross/account_book", params, nil, &response)
}

// CreateCrossMarginBorrowLoan create a cross margin borrow loan
// Borrow amount cannot be less than currency minimum borrow amount
func (e *Exchange) CreateCrossMarginBorrowLoan(ctx context.Context, arg *CrossMarginBorrowLoanRequest) (*CrossMarginLoanResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w, borrow amount must be greater than 0", order.ErrAmountIsInvalid)
	}
	var resp *CrossMarginLoanResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginCreateCrossBorrowLoanEPL, http.MethodPost, "margin/cross/loans", nil, arg, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// ExecuteRepayment when the liquidity of the currency is insufficient and the transaction risk is high, the currency will be disabled,
// and funds cannot be transferred.When the available balance of cross-margin is insufficient, the balance of the spot account can be used for repayment.
// Please ensure that the balance of the spot account is sufficient, and system uses cross-margin account for repayment first
func (e *Exchange) ExecuteRepayment(ctx context.Context, arg CurrencyAndAmountResponse) ([]*CrossMarginLoanResponse, error) {
	if arg.Currency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, fmt.Errorf("%w, repay amount must be greater than 0", order.ErrAmountIsInvalid)
	}
	var response []*CrossMarginLoanResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginExecuteRepaymentsEPL, http.MethodPost, "margin/cross/repayments", nil, &arg, &response)
}

// GetCrossMarginRepayments retrieves list of cross margin repayments
func (e *Exchange) GetCrossMarginRepayments(ctx context.Context, ccy currency.Code, loanID string, limit, offset uint64, reverse bool) ([]*CrossMarginLoanResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if loanID != "" {
		params.Set("loan_id", loanID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if reverse {
		params.Set("reverse", "true")
	}
	var response []*CrossMarginLoanResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetCrossMarginRepaymentsEPL, http.MethodGet, "margin/cross/repayments", params, nil, &response)
}

// GetMaxTransferableAmountForSpecificCrossMarginCurrency get the max transferable amount for a specific cross margin currency
func (e *Exchange) GetMaxTransferableAmountForSpecificCrossMarginCurrency(ctx context.Context, ccy currency.Code) (*CurrencyAndAmountResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	var response *CurrencyAndAmountResponse
	params.Set("currency", ccy.String())
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetMaxTransferCrossEPL, http.MethodGet, "margin/cross/transferable", params, nil, &response)
}

// GetMaxBorrowableAmountForSpecificCrossMarginCurrency returns the max borrowable amount for a specific cross margin currency
func (e *Exchange) GetMaxBorrowableAmountForSpecificCrossMarginCurrency(ctx context.Context, ccy currency.Code) (*CurrencyAndAmountResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("currency", ccy.String())
	var response *CurrencyAndAmountResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetMaxBorrowCrossEPL, http.MethodGet, "margin/cross/borrowable", params, nil, &response)
}

// GetCrossMarginBorrowHistory retrieves cross margin borrow history sorted by creation time in descending order by default.
// Set reverse=false to return ascending results.
func (e *Exchange) GetCrossMarginBorrowHistory(ctx context.Context, status uint64, ccy currency.Code, limit, offset uint64, reverse bool) ([]*CrossMarginLoanResponse, error) {
	if status < 1 || status > 3 {
		return nil, fmt.Errorf("%s %w, only allowed status values are 1:failed, 2:borrowed, and 3:repayment", e.Name, errInvalidOrderStatus)
	}
	params := url.Values{}
	params.Set("status", strconv.FormatUint(status, 10))
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if reverse {
		params.Set("reverse", strconv.FormatBool(reverse))
	}
	var response []*CrossMarginLoanResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetCrossBorrowHistoryEPL, http.MethodGet, "margin/cross/loans", params, nil, &response)
}

// GetSingleBorrowLoanDetail retrieve single borrow loan detail
func (e *Exchange) GetSingleBorrowLoanDetail(ctx context.Context, loanID string) (*CrossMarginLoanResponse, error) {
	if loanID == "" {
		return nil, errInvalidLoanID
	}
	var response *CrossMarginLoanResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, marginGetBorrowEPL, http.MethodGet, "margin/cross/loans/"+loanID, nil, nil, &response)
}

// *********************************Futures***************************************

var (
	errInvalidNumericString         = errors.New("invalid numeric string")
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

// GetAllFutureContracts retrieves the futures contracts of a settlement currency, all of which are returned when no
// limit is supplied
func (e *Exchange) GetAllFutureContracts(ctx context.Context, settle currency.Code, limit, offset uint64) ([]*FuturesContractResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	var contracts []*FuturesContractResponse
	return contracts, e.SendHTTPRequest(ctx, exchange.RestSpot, publicFuturesContractsEPL, common.EncodeURLValues("futures/"+settle.Item.Lower+"/contracts", params), &contracts)
}

// GetAllFutureContractsIncludingDelisted retrieves the futures contracts of a settlement currency, including
// delisted ones
func (e *Exchange) GetAllFutureContractsIncludingDelisted(ctx context.Context, settle currency.Code, limit, offset uint64) ([]*FuturesContractResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	var contracts []*FuturesContractResponse
	return contracts, e.SendHTTPRequest(ctx, exchange.RestSpot, publicFuturesContractsAllEPL, common.EncodeURLValues("futures/"+settle.Item.Lower+"/contracts_all", params), &contracts)
}

// GetFuturesContract returns a single futures contract info for the specified settle and Currency Pair (contract << in this case)
func (e *Exchange) GetFuturesContract(ctx context.Context, settle currency.Code, contract string) (*FuturesContractResponse, error) {
	if contract == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	var futureContract *FuturesContractResponse
	return futureContract, e.SendHTTPRequest(ctx, exchange.RestSpot, publicFuturesContractsEPL, "futures/"+settle.Item.Lower+"/contracts/"+contract, &futureContract)
}

// GetFuturesADLRiskStates retrieves the auto-deleveraging risk state of every futures market of a settlement currency
func (e *Exchange) GetFuturesADLRiskStates(ctx context.Context, settle currency.Code) (*FuturesADLRiskStatesResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	var resp *FuturesADLRiskStatesResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicFuturesADLRiskStatesEPL, "futures/"+settle.Item.Lower+"/adl_risk_states", &resp)
}

// GetFuturesOrderbook retrieves futures order book data
func (e *Exchange) GetFuturesOrderbook(ctx context.Context, settle currency.Code, contract currency.Pair, interval string, limit uint64, withOrderbookID bool) (*OrderbookResponse, error) {
	if contract.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if interval != "" {
		params.Set("interval", interval)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if withOrderbookID {
		params.Set("with_id", "true")
	}
	var response *OrderbookResponse
	return response, e.SendHTTPRequest(ctx, exchange.RestSpot, publicOrderbookFuturesEPL, common.EncodeURLValues("futures/"+settle.Item.Lower+"/order_book", params), &response)
}

// GetFuturesTradingHistory retrieves futures trading history
func (e *Exchange) GetFuturesTradingHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit, offset uint64, lastID string, from, to time.Time) ([]*TradingHistoryItemResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("contract", contract.Upper().String())
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	var response []*TradingHistoryItemResponse
	return response, e.SendHTTPRequest(ctx, exchange.RestSpot, publicTradingHistoryFuturesEPL, common.EncodeURLValues("futures/"+settle.Item.Lower+"/trades", params), &response)
}

// GetFuturesCandlesticks retrieves specified contract candlesticks. A limit cannot be combined with a time range,
// and timezone, one of all, utc0 or utc8, defaults to utc0 when empty.
func (e *Exchange) GetFuturesCandlesticks(ctx context.Context, settle currency.Code, contract currency.Pair, from, to time.Time, limit uint64, interval kline.Interval, timezone string) ([]*FuturesCandlestickResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if limit > 0 && (!from.IsZero() || !to.IsZero()) {
		return nil, errLimitWithTimeRange
	}
	if timezone != "" && timezone != "all" && timezone != utc0TimeZone && timezone != utc8TimeZone {
		return nil, fmt.Errorf("%w: %q", errInvalidTimezone, timezone)
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if interval.Duration().Microseconds() != 0 {
		intervalString, err := getIntervalString(interval)
		if err != nil {
			return nil, err
		}
		params.Set("interval", intervalString)
	}
	if timezone != "" {
		params.Set("timezone", timezone)
	}
	var candlesticks []*FuturesCandlestickResponse
	return candlesticks, e.SendHTTPRequest(ctx, exchange.RestFutures, publicCandleSticksFuturesEPL, common.EncodeURLValues("futures/"+settle.Item.Lower+"/candlesticks", params), &candlesticks)
}

// PremiumIndexKline retrieves premium Index K-Line
// Maximum of 1000 points can be returned in a query. Be sure not to exceed the limit when specifying from, to and interval
// A limit cannot be combined with a time range
func (e *Exchange) PremiumIndexKline(ctx context.Context, settleCurrency currency.Code, contract currency.Pair, from, to time.Time, limit uint64, interval kline.Interval) ([]*FuturesPremiumIndexKlineResponse, error) {
	if settleCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", errEmptyOrInvalidSettlementCurrency)
	}
	if contract.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if limit > 0 && (!from.IsZero() || !to.IsZero()) {
		return nil, errLimitWithTimeRange
	}
	intervalString, err := getIntervalString(interval)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	params.Set("interval", intervalString)
	var resp []*FuturesPremiumIndexKlineResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, publicPremiumIndexEPL, common.EncodeURLValues("futures/"+settleCurrency.Item.Lower+"/premium_index", params), &resp)
}

// GetFuturesTickers retrieves futures ticker information for a specific settle and contract info.
func (e *Exchange) GetFuturesTickers(ctx context.Context, settle currency.Code, contract currency.Pair) ([]*FuturesTickerResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	var tickers []*FuturesTickerResponse
	return tickers, e.SendHTTPRequest(ctx, exchange.RestSpot, publicTickersFuturesEPL, common.EncodeURLValues("futures/"+settle.Item.Lower+"/tickers", params), &tickers)
}

// GetFutureFundingRates retrieves funding rate information.
func (e *Exchange) GetFutureFundingRates(ctx context.Context, settle currency.Code, contract currency.Pair, limit uint64, from, to time.Time) ([]*FuturesFundingRateResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsInvalid() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	var rates []*FuturesFundingRateResponse
	return rates, e.SendHTTPRequest(ctx, exchange.RestSpot, publicFundingRatesEPL, common.EncodeURLValues("futures/"+settle.Item.Lower+"/funding_rate", params), &rates)
}

// GetBatchFuturesFundingRates retrieves the funding rate history of several perpetual contracts at once
func (e *Exchange) GetBatchFuturesFundingRates(ctx context.Context, settle currency.Code, contracts currency.Pairs) ([]*BatchFundingRatesResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if len(contracts) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	names := make([]string, len(contracts))
	for i := range contracts {
		if contracts[i].IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		names[i] = contracts[i].String()
	}
	var resp []*BatchFundingRatesResponse
	return resp, e.sendPublicPostRequest(ctx, publicBatchFundingRatesEPL, "futures/"+settle.Item.Lower+"/funding_rates", &batchFundingRatesRequest{Contracts: names}, &resp)
}

// GetFuturesInsuranceBalanceHistory retrieves futures insurance balance history
func (e *Exchange) GetFuturesInsuranceBalanceHistory(ctx context.Context, settle currency.Code, limit uint64) ([]*InsuranceBalanceResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var balances []*InsuranceBalanceResponse
	return balances, e.SendHTTPRequest(ctx, exchange.RestSpot, publicInsuranceFuturesEPL, common.EncodeURLValues("futures/"+settle.Item.Lower+"/insurance", params), &balances)
}

// GetFutureStats retrieves futures stats
func (e *Exchange) GetFutureStats(ctx context.Context, settle currency.Code, contract currency.Pair, from time.Time, interval kline.Interval, limit uint64) ([]*ContractStatResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w: settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsInvalid() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.Unix(), 10))
	}
	if int64(interval) != 0 {
		intervalString, err := getIntervalString(interval)
		if err != nil {
			return nil, err
		}
		params.Set("interval", intervalString)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var stats []*ContractStatResponse
	return stats, e.SendHTTPRequest(ctx, exchange.RestSpot, publicStatsFuturesEPL, common.EncodeURLValues("futures/"+settle.Item.Lower+"/contract_stats", params), &stats)
}

// GetIndexConstituent retrieves index constituents
func (e *Exchange) GetIndexConstituent(ctx context.Context, settle currency.Code, index string) (*IndexConstituentResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if index == "" {
		return nil, fmt.Errorf("%w: index pair string is required", currency.ErrCurrencyPairEmpty)
	}
	indexString := strings.ToUpper(index)
	var constituents *IndexConstituentResponse
	return constituents, e.SendHTTPRequest(ctx, exchange.RestSpot, publicIndexConstituentsEPL, "futures/"+settle.Item.Lower+"/index_constituents/"+indexString, &constituents)
}

// GetLiquidationHistory retrieves the public liquidation orders of a settlement currency, optionally for one contract.
// A time range may span at most an hour.
func (e *Exchange) GetLiquidationHistory(ctx context.Context, settle currency.Code, contract currency.Pair, from, to time.Time, limit uint64) ([]*LiquidationHistoryResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if !from.IsZero() && !to.IsZero() && to.Sub(from) > time.Hour {
		return nil, errLiquidationTimeRangeTooLarge
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var histories []*LiquidationHistoryResponse
	return histories, e.SendHTTPRequest(ctx, exchange.RestSpot, publicLiquidationHistoryEPL, common.EncodeURLValues("futures/"+settle.Item.Lower+"/liq_orders", params), &histories)
}

// QueryFuturesAccount retrieves futures account
func (e *Exchange) QueryFuturesAccount(ctx context.Context, settle currency.Code) (*FuturesAccountResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w: settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	var response *FuturesAccountResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualAccountEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/accounts", nil, nil, &response)
}

// GetFuturesAccountBooks retrieves account books. A contract only filters records from 2023-10-30 onwards.
func (e *Exchange) GetFuturesAccountBooks(ctx context.Context, settle currency.Code, contract currency.Pair, limit, offset uint64, from, to time.Time, changingType string) ([]*AccountBookItemResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if changingType != "" {
		params.Set("type", changingType)
	}
	var response []*AccountBookItemResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualAccountBooksEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/account_book", params, nil, &response)
}

// GetAllFuturesPositionsOfUsers list all positions of users. Every position is returned when no limit is supplied,
// and a supplied limit may not exceed 100.
func (e *Exchange) GetAllFuturesPositionsOfUsers(ctx context.Context, settle currency.Code, realPositionsOnly bool, limit, offset uint64) ([]*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if limit > 100 {
		return nil, fmt.Errorf("%w: %d exceeds the maximum of 100 positions", errInvalidLimit, limit)
	}
	params := url.Values{}
	if realPositionsOnly {
		params.Set("holding", "true")
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	var response []*PositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualPositionsEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/positions", params, nil, &response)
}

// GetFuturesHistoricalPositions retrieves a contract's historical position snapshots by time range
func (e *Exchange) GetFuturesHistoricalPositions(ctx context.Context, settle currency.Code, contract currency.Pair, limit, offset uint64, from, to time.Time) ([]*FuturesHistoricalPositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	var resp []*FuturesHistoricalPositionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualPositionsTimeRangeEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/positions_timerange", params, nil, &resp)
}

// GetSinglePosition returns a single position. A contract holding both a long and a short position in hedge mode is
// queried with GetPositionDetailInDualMode.
func (e *Exchange) GetSinglePosition(ctx context.Context, settle currency.Code, contract currency.Pair) (*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	var response *PositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualPositionEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/positions/"+contract.String(), nil, nil, &response)
}

// GetFuturesLeverage retrieves a contract's leverage for a position margin mode, isolated or cross, and a hedge mode
// side, dual_long or dual_short
func (e *Exchange) GetFuturesLeverage(ctx context.Context, settle currency.Code, contract currency.Pair, positionMarginMode, dualSide string) (*FuturesLeverageResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if positionMarginMode != "isolated" && positionMarginMode != "cross" {
		return nil, fmt.Errorf("%w: %q", errInvalidPositionMarginMode, positionMarginMode)
	}
	if dualSide != "dual_long" && dualSide != "dual_short" {
		return nil, fmt.Errorf("%w: 'dual_side' should be 'dual_short' or 'dual_long'", order.ErrSideIsInvalid)
	}
	params := url.Values{}
	params.Set("pos_margin_mode", positionMarginMode)
	params.Set("dual_side", dualSide)
	var resp *FuturesLeverageResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetLeverageEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/get_leverage/"+contract.String(), params, nil, &resp)
}

// UpdateFuturesPositionMargin adds margin to a futures position, or removes it with a negative change
func (e *Exchange) UpdateFuturesPositionMargin(ctx context.Context, settle currency.Code, change float64, contract currency.Pair) (*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if change == 0 {
		return nil, errInvalidMarginChange
	}
	params := url.Values{}
	params.Set("change", strconv.FormatFloat(change, 'f', -1, 64))
	var response *PositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateMarginEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/positions/"+contract.String()+"/margin", params, nil, &response)
}

// UpdateFuturesPositionLeverage update position leverage. A non-zero leverage sets isolated margin and ignores the
// cross leverage limit, while a zero leverage sets cross margin at the cross leverage limit.
func (e *Exchange) UpdateFuturesPositionLeverage(ctx context.Context, settle currency.Code, contract currency.Pair, leverage, crossLeverageLimit float64, positionID uint64) (*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if leverage < 0 {
		return nil, fmt.Errorf("%w: %f", order.ErrSubmitLeverageNotSupported, leverage)
	}
	params := url.Values{}
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	if leverage == 0 && crossLeverageLimit > 0 {
		params.Set("cross_leverage_limit", strconv.FormatFloat(crossLeverageLimit, 'f', -1, 64))
	}
	if positionID > 0 {
		params.Set("pid", strconv.FormatUint(positionID, 10))
	}
	var resp positionReply
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateLeverageEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/positions/"+contract.String()+"/leverage", params, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Position == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Position, nil
}

// SetFuturesLeverage sets a contract's leverage for a margin mode, isolated or cross, optionally for one hedge mode
// side, dual_long or dual_short
func (e *Exchange) SetFuturesLeverage(ctx context.Context, settle currency.Code, contract currency.Pair, leverage float64, marginMode, dualSide string) (*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if leverage <= 0 {
		return nil, fmt.Errorf("%w: %f", errInvalidLeverage, leverage)
	}
	if marginMode != "isolated" && marginMode != "cross" {
		return nil, fmt.Errorf("%w: %q", errInvalidPositionMarginMode, marginMode)
	}
	if dualSide != "" && dualSide != "dual_long" && dualSide != "dual_short" {
		return nil, fmt.Errorf("%w: 'dual_side' should be 'dual_short' or 'dual_long'", order.ErrSideIsInvalid)
	}
	params := url.Values{}
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	params.Set("margin_mode", marginMode)
	if dualSide != "" {
		params.Set("dual_side", dualSide)
	}
	var resp *PositionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualSetLeverageEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/positions/"+contract.String()+"/set_leverage", params, nil, &resp)
}

// SwitchFuturesPositionMarginMode switches a futures position between ISOLATED and CROSS margin
func (e *Exchange) SwitchFuturesPositionMarginMode(ctx context.Context, settle currency.Code, arg *FuturesPositionMarginModeRequest) (*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := arg.validate(); err != nil {
		return nil, err
	}
	var resp *PositionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualSwitchMarginModeEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/positions/cross_mode", nil, arg, &resp)
}

// SwitchFuturesDualModePositionMarginMode switches a contract's hedge mode positions between ISOLATED and CROSS margin
func (e *Exchange) SwitchFuturesDualModePositionMarginMode(ctx context.Context, settle currency.Code, arg *FuturesPositionMarginModeRequest) ([]*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := arg.validate(); err != nil {
		return nil, err
	}
	var resp []*PositionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualSwitchDualModeMarginModeEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/dual_comp/positions/cross_mode", nil, arg, &resp)
}

// validate checks a position margin mode switch before it is sent
func (f *FuturesPositionMarginModeRequest) validate() error {
	if err := common.NilGuard(f); err != nil {
		return err
	}
	if f.Mode != "ISOLATED" && f.Mode != "CROSS" {
		return fmt.Errorf("%w: %q", errInvalidPositionMarginMode, f.Mode)
	}
	if f.Contract.IsEmpty() {
		return fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	return nil
}

// EnableOrDisableDualMode enable or disable dual mode
// Before setting dual mode, make sure all positions are closed and no orders are open
func (e *Exchange) EnableOrDisableDualMode(ctx context.Context, settle currency.Code, dualMode bool) (*FuturesAccountResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("dual_mode", strconv.FormatBool(dualMode))
	var response *FuturesAccountResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualToggleDualModeEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/dual_mode", params, nil, &response)
}

// SetFuturesPositionMode sets the position holding mode to single, dual or dual_plus, superseding
// EnableOrDisableDualMode. Every position must be closed and every order cancelled beforehand.
func (e *Exchange) SetFuturesPositionMode(ctx context.Context, settle currency.Code, positionMode string) (*FuturesAccountResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if positionMode != "single" && positionMode != "dual" && positionMode != "dual_plus" {
		return nil, fmt.Errorf("%w: %q", errInvalidPositionMode, positionMode)
	}
	params := url.Values{}
	params.Set("position_mode", positionMode)
	var resp *FuturesAccountResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualSetPositionModeEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/set_position_mode", params, nil, &resp)
}

// GetPositionDetailInDualMode retrieve position detail in dual mode
func (e *Exchange) GetPositionDetailInDualMode(ctx context.Context, settle currency.Code, contract currency.Pair) ([]*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	var response []*PositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualPositionsDualModeEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/dual_comp/positions/"+contract.String(), nil, nil, &response)
}

// UpdatePositionMarginInDualMode adds margin to one side of a hedge mode position, or removes it with a negative
// change
func (e *Exchange) UpdatePositionMarginInDualMode(ctx context.Context, settle currency.Code, contract currency.Pair, change float64, dualSide string) ([]*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if change == 0 {
		return nil, errInvalidMarginChange
	}
	if dualSide != "dual_long" && dualSide != "dual_short" {
		return nil, fmt.Errorf("%w: 'dual_side' should be 'dual_short' or 'dual_long'", order.ErrSideIsInvalid)
	}
	params := url.Values{}
	params.Set("change", strconv.FormatFloat(change, 'f', -1, 64))
	params.Set("dual_side", dualSide)
	var response []*PositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateMarginDualModeEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/dual_comp/positions/"+contract.String()+"/margin", params, nil, &response)
}

// UpdatePositionLeverageInDualMode update position leverage in dual mode
func (e *Exchange) UpdatePositionLeverageInDualMode(ctx context.Context, settle currency.Code, contract currency.Pair, leverage, crossLeverageLimit float64) ([]*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if leverage < 0 {
		return nil, fmt.Errorf("%w: %f", order.ErrSubmitLeverageNotSupported, leverage)
	}
	params := url.Values{}
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	if leverage == 0 && crossLeverageLimit > 0 {
		params.Set("cross_leverage_limit", strconv.FormatFloat(crossLeverageLimit, 'f', -1, 64))
	}
	var response []*PositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateLeverageDualModeEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/dual_comp/positions/"+contract.String()+"/leverage", params, nil, &response)
}

// UpdatePositionRiskLimitInDualMode update position risk limit in dual mode
// Risk Limit as of GateIO: https://www.gate.com/en/help/futures/futures-logic/22162
func (e *Exchange) UpdatePositionRiskLimitInDualMode(ctx context.Context, settle currency.Code, contract currency.Pair, riskLimit float64) ([]*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if riskLimit <= 0 {
		return nil, errInvalidRiskLimit
	}
	params := url.Values{}
	params.Set("risk_limit", strconv.FormatFloat(riskLimit, 'f', -1, 64))
	var response []*PositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateRiskDualModeEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/dual_comp/positions/"+contract.String()+"/risk_limit", params, nil, &response)
}

// PlaceFuturesOrder creates futures order
// Create a futures order
// Creating futures orders requires size, which is number of contracts instead of currency amount. You can use quanto_multiplier in contract detail response to know how much currency 1 size contract represents
// Zero-filled order cannot be retrieved 10 minutes after order cancellation. You will get a 404 not found for such orders
// Set reduce_only to true can keep the position from changing side when reducing position size
// In single position mode, to close a position, you need to set size to 0 and close to true
// In dual position mode, to close one side position, you need to set auto_size side, reduce_only to true and size to 0
func (e *Exchange) PlaceFuturesOrder(ctx context.Context, arg *FuturesOrderCreateRequest) (*FuturesOrderResponse, error) {
	if err := arg.validate(true); err != nil {
		return nil, err
	}
	var response *FuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualSubmitOrderEPL, http.MethodPost, "futures/"+arg.Settle.Item.Lower+"/orders", nil, &arg, &response)
}

// GetFuturesOrders retrieves list of futures orders
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (e *Exchange) GetFuturesOrders(ctx context.Context, contract currency.Pair, status, lastID string, settle currency.Code, limit, offset uint64) ([]*FuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if status != statusOpen && status != statusFinished {
		return nil, fmt.Errorf("%w, only 'open' and 'finished' status are supported", errInvalidOrderStatus)
	}
	params := url.Values{}
	params.Set("status", status)
	if !contract.IsEmpty() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	var response []*FuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetOrdersEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/orders", params, nil, &response)
}

// CancelMultipleFuturesOpenOrders cancels every open futures order matching the request's filters
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (e *Exchange) CancelMultipleFuturesOpenOrders(ctx context.Context, settle currency.Code, arg *CancelFuturesOrdersRequest) ([]*FuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Side != "" && arg.Side != order.Bid.Lower() && arg.Side != order.Ask.Lower() {
		return nil, fmt.Errorf("%w: %q; only 'bid' and 'ask' are supported", order.ErrSideIsInvalid, arg.Side)
	}
	params := url.Values{}
	if arg.Contract.IsPopulated() {
		params.Set("contract", arg.Contract.String())
	}
	if arg.Side != "" {
		params.Set("side", arg.Side)
	}
	if arg.ActionMode != "" {
		params.Set("action_mode", arg.ActionMode)
	}
	if arg.ExcludeReduceOnly {
		params.Set("exclude_reduce_only", "true")
	}
	if arg.Text != "" {
		params.Set("text", arg.Text)
	}
	var response []*FuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualCancelOrdersEPL, http.MethodDelete, "futures/"+settle.Item.Lower+"/orders", params, nil, &response)
}

// GetFuturesOrdersByTimeRange retrieves futures orders by time range, including those older than the six months
// GetFuturesOrders covers
func (e *Exchange) GetFuturesOrdersByTimeRange(ctx context.Context, settle currency.Code, contract currency.Pair, limit, offset uint64, from, to time.Time) ([]*FuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	var resp []*FuturesOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetOrdersByTimeRangeEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/orders_timerange", params, nil, &resp)
}

// PlaceBatchFuturesOrders creates a list of futures orders
// Up to 10 orders per request
// If any of the order's parameters are missing or in the wrong format, all of them will not be executed, and a http status 400 error will be returned directly
// If the parameters are checked and passed, all are executed. Even if there is a business logic error in the middle (such as insufficient funds), it will not affect other execution orders
// The returned result is in array format, and the order corresponds to the orders in the request body
// In the returned result, the succeeded field of type bool indicates whether the execution was successful or not
// If the execution is successful, the normal order content is included; if the execution fails, the label field is included to indicate the cause of the error
// In the rate limiting, each order is counted individually
func (e *Exchange) PlaceBatchFuturesOrders(ctx context.Context, settle currency.Code, args []FuturesOrderCreateRequest) ([]*BatchFuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if len(args) == 0 {
		return nil, errOrdersEmpty
	}
	if len(args) > 10 {
		return nil, errTooManyOrderRequest
	}
	for x := range args {
		if err := args[x].validate(true); err != nil {
			return nil, err
		}
	}
	var response []*BatchFuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualSubmitBatchOrdersEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/batch_orders", nil, &args, &response)
}

// GetSingleFuturesOrder retrieves a single order by its identifier
func (e *Exchange) GetSingleFuturesOrder(ctx context.Context, settle currency.Code, orderID string) (*FuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w, 'order_id' cannot be empty", order.ErrOrderIDNotSet)
	}
	var response *FuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualFetchOrderEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/orders/"+orderID, nil, nil, &response)
}

// CancelSingleFuturesOrder cancels a single order by its ID or custom text. An action mode of ACK, RESULT or FULL
// selects the fields returned.
func (e *Exchange) CancelSingleFuturesOrder(ctx context.Context, settle currency.Code, orderID, actionMode string) (*FuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w, 'order_id' cannot be empty", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	if actionMode != "" {
		params.Set("action_mode", actionMode)
	}
	var response *FuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualCancelOrderEPL, http.MethodDelete, "futures/"+settle.Item.Lower+"/orders/"+orderID, params, nil, &response)
}

// AmendFuturesOrder amends an existing futures order's size, price or both
func (e *Exchange) AmendFuturesOrder(ctx context.Context, settle currency.Code, orderID string, arg *AmendFuturesOrderRequest) (*FuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w, 'order_id' cannot be empty", order.ErrOrderIDNotSet)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Size == 0 && arg.Price == 0 {
		return nil, fmt.Errorf("%w: size or price is required", errNoOrderAmendment)
	}
	if arg.Price < 0 {
		return nil, fmt.Errorf("%w: %v", limits.ErrPriceBelowMin, arg.Price)
	}
	var response *FuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualAmendOrderEPL, http.MethodPut, "futures/"+settle.Item.Lower+"/orders/"+orderID, nil, arg, &response)
}

// GetMyFuturesTradingHistory retrieves the account's futures fills from the past six months; older fills are
// retrieved with GetMyFuturesTradingHistoryByTimeRange
func (e *Exchange) GetMyFuturesTradingHistory(ctx context.Context, settle currency.Code, lastID, orderID string, contract currency.Pair, limit, offset uint64) ([]*TradingHistoryItemResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if orderID != "" {
		params.Set("order", orderID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	var response []*TradingHistoryItemResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualTradingHistoryEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/my_trades", params, nil, &response)
}

// GetMyFuturesTradingHistoryByTimeRange retrieves the account's futures fills by time range, optionally only those
// in a role of maker or taker
func (e *Exchange) GetMyFuturesTradingHistoryByTimeRange(ctx context.Context, settle currency.Code, contract currency.Pair, role string, limit, offset uint64, from, to time.Time) ([]*FuturesTradeByTimeRangeResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if role != "" && role != "maker" && role != "taker" {
		return nil, fmt.Errorf("%w: %q", errInvalidTradeRole, role)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if role != "" {
		params.Set("role", role)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	var resp []*FuturesTradeByTimeRangeResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualTradingHistoryByTimeRangeEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/my_trades_timerange", params, nil, &resp)
}

// GetFuturesPositionCloseHistory lists position close history, optionally filtered by a side of long or short and
// by profit or loss
func (e *Exchange) GetFuturesPositionCloseHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit, offset uint64, from, to time.Time, side, pnl string) ([]*PositionCloseHistoryResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if side != "" && side != order.Long.Lower() && side != order.Short.Lower() {
		return nil, fmt.Errorf("%w: %q; only 'long' and 'short' are supported", order.ErrSideIsInvalid, side)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if side != "" {
		params.Set("side", side)
	}
	if pnl != "" {
		params.Set("pnl", pnl)
	}
	var response []*PositionCloseHistoryResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualClosePositionEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/position_close", params, nil, &response)
}

// GetFuturesLiquidationHistory lists the account's liquidations, optionally those at a given time
func (e *Exchange) GetFuturesLiquidationHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit, offset uint64, from, to, at time.Time) ([]*LiquidationHistoryItemResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if !at.IsZero() {
		params.Set("at", strconv.FormatInt(at.Unix(), 10))
	}
	var response []*LiquidationHistoryItemResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualLiquidationHistoryEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/liquidates", params, nil, &response)
}

// GetFuturesAutoDeleverages lists the auto-deleveraging of the account's positions, optionally those at a given time
func (e *Exchange) GetFuturesAutoDeleverages(ctx context.Context, settle currency.Code, contract currency.Pair, limit, offset uint64, from, to, at time.Time) ([]*FuturesAutoDeleverageResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if !at.IsZero() {
		params.Set("at", strconv.FormatInt(at.Unix(), 10))
	}
	var resp []*FuturesAutoDeleverageResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualAutoDeleveragesEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/auto_deleverages", params, nil, &resp)
}

// CountdownCancelFuturesOrders sets a countdown, of at least 5 seconds, after which the account's open futures
// orders are cancelled unless the countdown is renewed. A zero timeout cancels the countdown.
func (e *Exchange) CountdownCancelFuturesOrders(ctx context.Context, settle currency.Code, arg *CountdownRequest) (*TriggerTimeResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", errEmptyOrInvalidSettlementCurrency)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Timeout < 0 || (arg.Timeout > 0 && arg.Timeout < 5) {
		return nil, errInvalidTimeout
	}
	return e.sendCountdownCancelOrdersRequest(ctx, perpetualCancelTriggerOrdersEPL, "futures/"+settle.Item.Lower+"/countdown_cancel_all", arg)
}

func (e *Exchange) sendCountdownCancelOrdersRequest(ctx context.Context, epl request.EndpointLimit, path string, arg any) (*TriggerTimeResponse, error) {
	var response *TriggerTimeResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, epl, http.MethodPost, path, nil, arg, &response)
}

// GetFuturesTradingFees retrieves the account's futures trading fee rates keyed by contract, optionally for one
// contract
func (e *Exchange) GetFuturesTradingFees(ctx context.Context, settle currency.Code, contract currency.Pair) (map[string]*FuturesTradingFeeResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	var resp map[string]*FuturesTradingFeeResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualTradingFeeEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/fee", params, nil, &resp)
}

// CancelBatchFuturesOrders cancels up to 20 futures orders by their IDs
func (e *Exchange) CancelBatchFuturesOrders(ctx context.Context, settle currency.Code, orderIDs []string) ([]*CancelFuturesOrderResultResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if len(orderIDs) == 0 {
		return nil, errNoOrdersToCancel
	}
	if len(orderIDs) > 20 {
		return nil, fmt.Errorf("%w: %d exceeds the maximum of 20", errTooManyOrders, len(orderIDs))
	}
	if slices.Contains(orderIDs, "") {
		return nil, order.ErrOrderIDNotSet
	}
	var resp []*CancelFuturesOrderResultResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualCancelBatchOrdersEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/batch_cancel_orders", nil, orderIDs, &resp)
}

// AmendBatchFuturesOrders amends up to 10 futures orders, each identified by its order ID or custom text
func (e *Exchange) AmendBatchFuturesOrders(ctx context.Context, settle currency.Code, args []*BatchAmendFuturesOrderRequest) ([]*BatchFuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if len(args) == 0 {
		return nil, errOrdersEmpty
	}
	if len(args) > 10 {
		return nil, fmt.Errorf("%w: %d exceeds the maximum of 10", errTooManyOrders, len(args))
	}
	for _, arg := range args {
		if err := common.NilGuard(arg); err != nil {
			return nil, err
		}
		if arg.OrderID == 0 && arg.Text == "" {
			return nil, fmt.Errorf("%w: order ID or text is required", order.ErrOrderIDNotSet)
		}
		if arg.Size == 0 && arg.Price == 0 {
			return nil, fmt.Errorf("%w: size or price is required", errNoOrderAmendment)
		}
	}
	var resp []*BatchFuturesOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualAmendBatchOrdersEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/batch_amend_orders", nil, args, &resp)
}

// PlaceFuturesBBOOrder places a futures order priced at a level of the order book, the bids for a sell and the asks
// for a buy
func (e *Exchange) PlaceFuturesBBOOrder(ctx context.Context, settle currency.Code, arg *FuturesBBOOrderRequest) (*FuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if arg.Size == 0 && !arg.ClosePosition && arg.AutoSize == "" {
		return nil, errInvalidOrderSize
	}
	if arg.Direction != order.Buy.Lower() && arg.Direction != order.Sell.Lower() {
		return nil, fmt.Errorf("%w: %q; only 'buy' and 'sell' are supported", order.ErrSideIsInvalid, arg.Direction)
	}
	if arg.Level < 1 || arg.Level > 20 {
		return nil, fmt.Errorf("%w: %d", errInvalidOrderBookLevel, arg.Level)
	}
	if arg.TimeInForce != "" {
		if _, err := timeInForceFromString(arg.TimeInForce); err != nil {
			return nil, err
		}
	}
	var resp *FuturesOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualSubmitBBOOrderEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/bbo_orders", nil, arg, &resp)
}

// CreateTrailOrder creates a futures trailing stop order
func (e *Exchange) CreateTrailOrder(ctx context.Context, settle currency.Code, arg *CreateTrailOrderRequest) (*CreateTrailOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if arg.Amount == 0 {
		return nil, errInvalidOrderSize
	}
	if arg.PriceType > 3 {
		return nil, fmt.Errorf("%w: %d", errInvalidTriggerPriceType, arg.PriceType)
	}
	var resp gateioAPIResponse[*CreateTrailOrderResponse]
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualCreateTrailOrderEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/autoorder/v1/trail/create", nil, arg, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// StopTrailOrder stops a futures trail order identified by its ID or custom text
func (e *Exchange) StopTrailOrder(ctx context.Context, settle currency.Code, arg *StopTrailOrderRequest) (*TrailOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.ID == 0 && arg.Text == "" {
		return nil, errTrailOrderIDOrTextRequired
	}
	var resp trailOrderReply
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualStopTrailOrderEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/autoorder/v1/trail/stop", nil, arg, &resp); err != nil {
		return nil, err
	}
	if resp.Order == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Order, nil
}

// StopAllTrailOrders stops the futures trail orders matching the request's filters
func (e *Exchange) StopAllTrailOrders(ctx context.Context, settle currency.Code, arg *StopAllTrailOrdersRequest) ([]*TrailOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.RelatedPosition > 2 {
		return nil, fmt.Errorf("%w: related position %d", errInvalidTrailOrderFilter, arg.RelatedPosition)
	}
	var resp trailOrdersReply
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualStopAllTrailOrdersEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/autoorder/v1/trail/stop_all", nil, arg, &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

// GetTrailOrders retrieves the futures trail orders matching the request's filters
func (e *Exchange) GetTrailOrders(ctx context.Context, settle currency.Code, arg *GetTrailOrdersRequest) ([]*TrailOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	switch {
	case arg.SortBy > 2:
		return nil, fmt.Errorf("%w: sort by %d", errInvalidTrailOrderFilter, arg.SortBy)
	case arg.RelatedPosition > 2:
		return nil, fmt.Errorf("%w: related position %d", errInvalidTrailOrderFilter, arg.RelatedPosition)
	case arg.ReduceOnly > 2:
		return nil, fmt.Errorf("%w: reduce only %d", errInvalidTrailOrderFilter, arg.ReduceOnly)
	case arg.Side > 2:
		return nil, fmt.Errorf("%w: side %d", errInvalidTrailOrderFilter, arg.Side)
	}
	if !arg.StartAt.IsZero() && !arg.EndAt.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartAt, arg.EndAt); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if arg.Contract.IsPopulated() {
		params.Set("contract", arg.Contract.String())
	}
	if arg.IsFinished {
		params.Set("is_finished", "true")
	}
	if !arg.StartAt.IsZero() {
		params.Set("start_at", strconv.FormatInt(arg.StartAt.Unix(), 10))
	}
	if !arg.EndAt.IsZero() {
		params.Set("end_at", strconv.FormatInt(arg.EndAt.Unix(), 10))
	}
	if arg.PageNumber > 0 {
		params.Set("page_num", strconv.FormatUint(arg.PageNumber, 10))
	}
	if arg.PageSize > 0 {
		params.Set("page_size", strconv.FormatUint(arg.PageSize, 10))
	}
	if arg.SortBy > 0 {
		params.Set("sort_by", strconv.FormatUint(uint64(arg.SortBy), 10))
	}
	if arg.HideCancelled {
		params.Set("hide_cancel", "true")
	}
	if arg.RelatedPosition > 0 {
		params.Set("related_position", strconv.FormatUint(uint64(arg.RelatedPosition), 10))
	}
	if arg.SortByTrigger {
		params.Set("sort_by_trigger", "true")
	}
	if arg.ReduceOnly > 0 {
		params.Set("reduce_only", strconv.FormatUint(uint64(arg.ReduceOnly), 10))
	}
	if arg.Side > 0 {
		params.Set("side", strconv.FormatUint(uint64(arg.Side), 10))
	}
	var resp trailOrdersReply
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetTrailOrdersEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/autoorder/v1/trail/list", params, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

// GetTrailOrderDetail retrieves a futures trail order's detail
func (e *Exchange) GetTrailOrderDetail(ctx context.Context, settle currency.Code, orderID uint64) (*TrailOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == 0 {
		return nil, fmt.Errorf("%w: trail order ID is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("id", strconv.FormatUint(orderID, 10))
	var resp gateioAPIResponse[*trailOrderReply]
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetTrailOrderDetailEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/autoorder/v1/trail/detail", params, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Data == nil || resp.Data.Order == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data.Order, nil
}

// UpdateTrailOrder amends a futures trail order, leaving parameters with empty values unchanged
func (e *Exchange) UpdateTrailOrder(ctx context.Context, settle currency.Code, arg *UpdateTrailOrderRequest) (*TrailOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.ID == 0 {
		return nil, fmt.Errorf("%w: trail order ID is required", order.ErrOrderIDNotSet)
	}
	if arg.IsGreaterThanEqual != "" && arg.IsGreaterThanEqual != "true" && arg.IsGreaterThanEqual != "false" {
		return nil, fmt.Errorf("%w: is_gte_str %q", errInvalidTrailOrderFilter, arg.IsGreaterThanEqual)
	}
	if arg.PriceType > 3 {
		return nil, fmt.Errorf("%w: %d", errInvalidTriggerPriceType, arg.PriceType)
	}
	var resp trailOrderReply
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualUpdateTrailOrderEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/autoorder/v1/trail/update", nil, arg, &resp); err != nil {
		return nil, err
	}
	if resp.Order == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Order, nil
}

// GetTrailOrderChangeLog retrieves the creation and modification records of a futures trail order
func (e *Exchange) GetTrailOrderChangeLog(ctx context.Context, settle currency.Code, orderID, pageNumber, pageSize uint64) ([]*TrailOrderChangeResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == 0 {
		return nil, fmt.Errorf("%w: trail order ID is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("id", strconv.FormatUint(orderID, 10))
	if pageNumber > 0 {
		params.Set("page_num", strconv.FormatUint(pageNumber, 10))
	}
	if pageSize > 0 {
		params.Set("page_size", strconv.FormatUint(pageSize, 10))
	}
	var resp trailOrderChangeLogReply
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetTrailOrderChangeLogEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/autoorder/v1/trail/change_log", params, nil, &resp); err != nil {
		return nil, err
	}
	return resp.ChangeLog, nil
}

// CreateChaseOrder creates a futures chase limit order that automatically adjusts its price to track the market
// until the full amount is filled or the order is stopped. A chase may be bounded by a price limit or by an offset
// limit, but not both.
func (e *Exchange) CreateChaseOrder(ctx context.Context, arg *CreateChaseOrderRequest) (*CreateChaseOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Settle.IsEmpty() {
		return nil, fmt.Errorf("%w: settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.Contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if arg.Amount == 0 {
		return nil, errInvalidOrderSize
	}
	if arg.PriceLimit != 0 && arg.OffsetLimit != 0 {
		return nil, errChasePriceAndOffsetLimit
	}
	var resp *CreateChaseOrderResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualCreateChaseOrderEPL, http.MethodPost, "futures/"+arg.Settle.Item.Lower+"/autoorder/v1/chase/create", nil, arg, &resp)
}

// StopChaseOrder stops a running futures chase limit order. Either the order ID or the custom
// text label must be provided to identify the order to stop.
func (e *Exchange) StopChaseOrder(ctx context.Context, settle currency.Code, arg *StopChaseOrderRequest) (*ChaseOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w: settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.ID == "" && arg.Text == "" {
		return nil, errChaseOrderIDOrTextRequired
	}
	var resp chaseOrderReply
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualStopChaseOrderEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/autoorder/v1/chase/stop", nil, arg, &resp); err != nil {
		return nil, err
	}
	if resp.Order == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Order, nil
}

// StopAllChaseOrders stops all running futures chase limit orders for the given settlement
// currency. The request may optionally scope the cancellation to a specific contract or margin mode.
func (e *Exchange) StopAllChaseOrders(ctx context.Context, settle currency.Code, arg *StopAllChaseOrdersRequest) ([]*ChaseOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w: settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	var resp chaseOrdersReply
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualStopAllChaseOrdersEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/autoorder/v1/chase/stop_all", nil, arg, &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

// GetChaseOrders retrieves a page of futures chase limit orders, sorted by creation time (1) or finish time (2).
// Finished orders are listed within a required time range.
func (e *Exchange) GetChaseOrders(ctx context.Context, settle currency.Code, arg *GetChaseOrdersRequest) ([]*ChaseOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w: settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.SortBy != 1 && arg.SortBy != 2 {
		return nil, errInvalidChaseSortBy
	}
	if arg.IsFinished && (arg.StartAt.IsZero() || arg.EndAt.IsZero()) {
		return nil, errChaseTimeRangeRequired
	}
	if !arg.StartAt.IsZero() && !arg.EndAt.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartAt, arg.EndAt); err != nil {
			return nil, err
		}
	}
	switch {
	case arg.PageSize > 100:
		return nil, fmt.Errorf("%w: %d exceeds the maximum page size of 100", errInvalidLimit, arg.PageSize)
	case arg.ReduceOnly > 2:
		return nil, fmt.Errorf("%w: reduce only %d", errInvalidChaseOrderFilter, arg.ReduceOnly)
	case arg.Side > 2:
		return nil, fmt.Errorf("%w: side %d", errInvalidChaseOrderFilter, arg.Side)
	}
	params := url.Values{}
	if arg.Contract.IsPopulated() {
		params.Set("contract", arg.Contract.String())
	}
	if arg.IsFinished {
		params.Set("is_finished", "true")
	}
	if !arg.StartAt.IsZero() {
		params.Set("start_at", strconv.FormatInt(arg.StartAt.Unix(), 10))
	}
	if !arg.EndAt.IsZero() {
		params.Set("end_at", strconv.FormatInt(arg.EndAt.Unix(), 10))
	}
	if arg.PageNumber > 0 {
		params.Set("page_num", strconv.FormatUint(arg.PageNumber, 10))
	}
	if arg.PageSize > 0 {
		params.Set("page_size", strconv.FormatUint(arg.PageSize, 10))
	}
	params.Set("sort_by", strconv.FormatUint(uint64(arg.SortBy), 10))
	if arg.HideCancelled {
		params.Set("hide_cancel", "true")
	}
	if arg.ReduceOnly > 0 {
		params.Set("reduce_only", strconv.FormatUint(uint64(arg.ReduceOnly), 10))
	}
	if arg.Side > 0 {
		params.Set("side", strconv.FormatUint(uint64(arg.Side), 10))
	}
	var resp chaseOrdersReply
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetChaseOrdersEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/autoorder/v1/chase/list", params, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

// GetChaseOrderDetail retrieves the full detail of a single futures chase limit order by its ID.
func (e *Exchange) GetChaseOrderDetail(ctx context.Context, settle currency.Code, orderID string) (*ChaseOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w: settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w: chase order ID is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("id", orderID)
	var resp chaseOrderReply
	if err := e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetChaseOrderDetailEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/autoorder/v1/chase/detail", params, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Order == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Order, nil
}

// CreatePriceTriggeredFuturesOrder create a price-triggered order
func (e *Exchange) CreatePriceTriggeredFuturesOrder(ctx context.Context, settle currency.Code, arg *FuturesPriceTriggeredOrderRequest) (*TriggerOrderIDResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := arg.validate(); err != nil {
		return nil, err
	}
	var response *TriggerOrderIDResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualSubmitTriggerOrderEPL, http.MethodPost, "futures/"+settle.Item.Lower+"/price_orders", nil, arg, &response)
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
	// close-long-order and close-short-order are read-only types Gate reports for an order's own take profit and stop loss
	switch p.OrderType {
	case "", "close-long-position", "close-short-position", "plan-close-long-position", "plan-close-short-position":
		return nil
	default:
		return fmt.Errorf("%w: %q", errInvalidAutoOrderType, p.OrderType)
	}
}

// ListAllFuturesAutoOrders lists futures price triggered orders, optionally only those with an open or finished
// status
func (e *Exchange) ListAllFuturesAutoOrders(ctx context.Context, status string, settle currency.Code, contract currency.Pair, limit, offset uint64) ([]*PriceTriggeredOrderResponse, error) {
	if status != "" && status != statusOpen && status != statusFinished {
		return nil, fmt.Errorf("%w status: %s", errInvalidOrderStatus, status)
	}
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if status != "" {
		params.Set("status", status)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	var response []*PriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualListOpenOrdersEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/price_orders", params, nil, &response)
}

// CancelAllFuturesOpenOrders cancels every open futures price triggered order, optionally for one contract
func (e *Exchange) CancelAllFuturesOpenOrders(ctx context.Context, settle currency.Code, contract currency.Pair) ([]*PriceTriggeredOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	var response []*PriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualCancelOpenOrdersEPL, http.MethodDelete, "futures/"+settle.Item.Lower+"/price_orders", params, nil, &response)
}

// GetSingleFuturesPriceTriggeredOrder retrieves a single price triggered order
func (e *Exchange) GetSingleFuturesPriceTriggeredOrder(ctx context.Context, settle currency.Code, orderID string) (*PriceTriggeredOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var response *PriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualGetTriggerOrderEPL, http.MethodGet, "futures/"+settle.Item.Lower+"/price_orders/"+orderID, nil, nil, &response)
}

// CancelFuturesPriceTriggeredOrder cancel a price-triggered order
func (e *Exchange) CancelFuturesPriceTriggeredOrder(ctx context.Context, settle currency.Code, orderID string) (*PriceTriggeredOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var response *PriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualCancelTriggerOrderEPL, http.MethodDelete, "futures/"+settle.Item.Lower+"/price_orders/"+orderID, nil, nil, &response)
}

// AmendFuturesPriceTriggeredOrder amends a pending futures price triggered order
func (e *Exchange) AmendFuturesPriceTriggeredOrder(ctx context.Context, settle currency.Code, arg *AmendFuturesPriceTriggeredOrderRequest) (*TriggerOrderIDResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.OrderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.Size == nil && arg.Amount == 0 && arg.Price == nil && arg.TriggerPrice == 0 && arg.PriceType == nil && arg.AutoSize == "" && !arg.Close {
		return nil, errNoOrderAmendment
	}
	if arg.Price != nil && *arg.Price < 0 {
		return nil, fmt.Errorf("%w: %v", limits.ErrPriceBelowMin, *arg.Price)
	}
	if arg.PriceType != nil && (*arg.PriceType < 0 || *arg.PriceType > 2) {
		return nil, fmt.Errorf("%w: %d; 0: latest trade price, 1: mark price, 2: index price", errInvalidTriggerPriceType, *arg.PriceType)
	}
	var resp *TriggerOrderIDResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, perpetualAmendTriggerOrderEPL, http.MethodPut, "futures/"+settle.Item.Lower+"/price_orders/amend", nil, arg, &resp)
}

// *************************************** Delivery ***************************************

// GetAllDeliveryContracts retrieves all futures contracts
func (e *Exchange) GetAllDeliveryContracts(ctx context.Context, settle currency.Code) ([]*DeliveryContractResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	var contracts []*DeliveryContractResponse
	return contracts, e.SendHTTPRequest(ctx, exchange.RestSpot, publicDeliveryContractsEPL, "delivery/"+settle.Item.Lower+"/contracts", &contracts)
}

// GetDeliveryContract retrieves a single delivery contract instance
func (e *Exchange) GetDeliveryContract(ctx context.Context, settle currency.Code, contract currency.Pair) (*DeliveryContractResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	var deliveryContract *DeliveryContractResponse
	return deliveryContract, e.SendHTTPRequest(ctx, exchange.RestSpot, publicDeliveryContractsEPL, "delivery/"+settle.Item.Lower+"/contracts/"+contract.String(), &deliveryContract)
}

// GetDeliveryOrderbook delivery orderbook
func (e *Exchange) GetDeliveryOrderbook(ctx context.Context, settle currency.Code, interval string, contract currency.Pair, limit uint64, withOrderbookID bool) (*OrderbookResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if interval != "" {
		params.Set("interval", interval)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if withOrderbookID {
		params.Set("with_id", strconv.FormatBool(withOrderbookID))
	}
	var orderbook *OrderbookResponse
	return orderbook, e.SendHTTPRequest(ctx, exchange.RestSpot, publicOrderbookDeliveryEPL, common.EncodeURLValues("delivery/"+settle.Item.Lower+"/order_book", params), &orderbook)
}

// GetDeliveryTradingHistory retrieves futures trading history
func (e *Exchange) GetDeliveryTradingHistory(ctx context.Context, settle currency.Code, lastID string, contract currency.Pair, limit uint64, from, to time.Time) ([]*TradingHistoryItemResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	var histories []*TradingHistoryItemResponse
	return histories, e.SendHTTPRequest(ctx, exchange.RestSpot, publicTradingHistoryDeliveryEPL, common.EncodeURLValues("delivery/"+settle.Item.Lower+"/trades", params), &histories)
}

// GetDeliveryFuturesCandlesticks retrieves specified contract candlesticks
// A limit returns the most recent candles, so Gate rejects one sent alongside a from or to time
func (e *Exchange) GetDeliveryFuturesCandlesticks(ctx context.Context, settle currency.Code, contract currency.Pair, from, to time.Time, limit uint64, interval kline.Interval) ([]*FuturesCandlestickResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if limit > 0 && (!from.IsZero() || !to.IsZero()) {
		return nil, errLimitWithTimeRange
	}
	params := url.Values{}
	if int64(interval) != 0 {
		intervalString, err := getIntervalString(interval)
		if err != nil {
			return nil, err
		}
		params.Set("interval", intervalString)
	}
	params.Set("contract", contract.Upper().String())
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var candlesticks []*FuturesCandlestickResponse
	return candlesticks, e.SendHTTPRequest(ctx, exchange.RestSpot, publicCandleSticksDeliveryEPL, common.EncodeURLValues("delivery/"+settle.Item.Lower+"/candlesticks", params), &candlesticks)
}

// GetDeliveryFutureTickers retrieves futures ticker information for a specific settle and contract info.
func (e *Exchange) GetDeliveryFutureTickers(ctx context.Context, settle currency.Code, contract currency.Pair) ([]*FuturesTickerResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	var tickers []*FuturesTickerResponse
	return tickers, e.SendHTTPRequest(ctx, exchange.RestSpot, publicTickersDeliveryEPL, common.EncodeURLValues("delivery/"+settle.Item.Lower+"/tickers", params), &tickers)
}

// GetDeliveryInsuranceBalanceHistory retrieves delivery futures insurance balance history
func (e *Exchange) GetDeliveryInsuranceBalanceHistory(ctx context.Context, settle currency.Code, limit uint64) ([]*InsuranceBalanceResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var balances []*InsuranceBalanceResponse
	return balances, e.SendHTTPRequest(ctx, exchange.RestSpot, publicInsuranceDeliveryEPL, common.EncodeURLValues("delivery/"+settle.Item.Lower+"/insurance", params), &balances)
}

// GetDeliveryFuturesAccounts retrieves futures account
func (e *Exchange) GetDeliveryFuturesAccounts(ctx context.Context, settle currency.Code) (*FuturesAccountResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	var response *FuturesAccountResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryAccountEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/accounts", nil, nil, &response)
}

// GetDeliveryAccountBooks retrieves account books
func (e *Exchange) GetDeliveryAccountBooks(ctx context.Context, settle currency.Code, limit uint64, from, to time.Time, changingType string) ([]*AccountBookItemResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if changingType != "" {
		params.Set("type", changingType)
	}
	var response []*AccountBookItemResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryAccountBooksEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/account_book", params, nil, &response)
}

// GetAllDeliveryPositionsOfUser retrieves all positions of user
func (e *Exchange) GetAllDeliveryPositionsOfUser(ctx context.Context, settle currency.Code) ([]*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	var response []*PositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryPositionsEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/positions", nil, nil, &response)
}

// GetSingleDeliveryPosition get single position
func (e *Exchange) GetSingleDeliveryPosition(ctx context.Context, settle currency.Code, contract currency.Pair) (*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	var response *PositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryPositionsEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/positions/"+contract.String(), nil, nil, &response)
}

// UpdateDeliveryPositionMargin updates position margin, adding margin for a positive change and removing it for a negative one
func (e *Exchange) UpdateDeliveryPositionMargin(ctx context.Context, settle currency.Code, change float64, contract currency.Pair) (*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if change == 0 {
		return nil, errInvalidMarginChange
	}
	params := url.Values{}
	params.Set("change", strconv.FormatFloat(change, 'f', -1, 64))
	var response *PositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryUpdateMarginEPL, http.MethodPost, "delivery/"+settle.Item.Lower+"/positions/"+contract.String()+"/margin", params, nil, &response)
}

// UpdateDeliveryPositionLeverage updates position leverage
func (e *Exchange) UpdateDeliveryPositionLeverage(ctx context.Context, settle currency.Code, contract currency.Pair, leverage float64) (*PositionResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if leverage < 0 {
		return nil, order.ErrSubmitLeverageNotSupported
	}
	params := url.Values{}
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	var response *PositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx,
		exchange.RestSpot, deliveryUpdateLeverageEPL, http.MethodPost, "delivery/"+settle.Item.Lower+"/positions/"+contract.String()+"/leverage", params, nil, &response)
}

// PlaceDeliveryOrder create a futures order
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (e *Exchange) PlaceDeliveryOrder(ctx context.Context, arg *DeliveryOrderCreateRequest) (*FuturesOrderResponse, error) {
	if err := arg.validate(true); err != nil {
		return nil, err
	}
	var response *FuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliverySubmitOrderEPL, http.MethodPost, "delivery/"+arg.Settle.Item.Lower+"/orders", nil, arg, &response)
}

// GetDeliveryOrders list futures orders
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (e *Exchange) GetDeliveryOrders(ctx context.Context, contract currency.Pair, status string, settle currency.Code, lastID string, limit, offset uint64, countTotal bool) ([]*FuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if status != statusOpen && status != statusFinished {
		return nil, fmt.Errorf("%w, only 'open' and 'finished' status are supported", errInvalidOrderStatus)
	}
	params := url.Values{}
	params.Set("status", status)
	if !contract.IsEmpty() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if countTotal && status != statusOpen {
		params.Set("count_total", "1")
	}
	var response []*FuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryGetOrdersEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/orders", params, nil, &response)
}

// CancelMultipleDeliveryOrders cancel all open orders matched
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (e *Exchange) CancelMultipleDeliveryOrders(ctx context.Context, contract currency.Pair, side string, settle currency.Code) ([]*FuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsInvalid() {
		return nil, fmt.Errorf("%w, currency pair for contract must not be empty", currency.ErrCurrencyPairEmpty)
	}
	// Dropping an unrecognised side would widen the cancellation to both sides of the book
	if side != "" && side != order.Ask.Lower() && side != order.Bid.Lower() {
		return nil, fmt.Errorf("%w: %q; only 'ask' and 'bid' are supported", order.ErrSideIsInvalid, side)
	}
	params := url.Values{}
	if side != "" {
		params.Set("side", side)
	}
	params.Set("contract", contract.String())
	var response []*FuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryCancelOrdersEPL, http.MethodDelete, "delivery/"+settle.Item.Lower+"/orders", params, nil, &response)
}

// GetSingleDeliveryOrder Get a single order
// Zero-filled order cannot be retrieved 10 minutes after order cancellation
func (e *Exchange) GetSingleDeliveryOrder(ctx context.Context, settle currency.Code, orderID string) (*FuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w, 'order_id' cannot be empty", order.ErrOrderIDNotSet)
	}
	var response *FuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryGetOrderEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/orders/"+orderID, nil, nil, &response)
}

// CancelSingleDeliveryOrder cancel a single order
func (e *Exchange) CancelSingleDeliveryOrder(ctx context.Context, settle currency.Code, orderID string) (*FuturesOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, fmt.Errorf("%w, 'order_id' cannot be empty", order.ErrOrderIDNotSet)
	}
	var response *FuturesOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryCancelOrderEPL, http.MethodDelete, "delivery/"+settle.Item.Lower+"/orders/"+orderID, nil, nil, &response)
}

// GetMyDeliveryTradingHistory retrieves authenticated account delivery futures trading history
func (e *Exchange) GetMyDeliveryTradingHistory(ctx context.Context, settle currency.Code, orderID string, contract currency.Pair, limit, offset, countTotal uint64, lastID string) ([]*TradingHistoryItemResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if orderID != "" {
		params.Set("order", orderID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if lastID != "" {
		params.Set("last_id", lastID)
	}
	if countTotal == 1 {
		params.Set("count_total", strconv.FormatUint(countTotal, 10))
	}
	var response []*TradingHistoryItemResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryTradingHistoryEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/my_trades", params, nil, &response)
}

// GetDeliveryPositionCloseHistory retrieves position history
func (e *Exchange) GetDeliveryPositionCloseHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit uint64) ([]*PositionCloseHistoryResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var response []*PositionCloseHistoryResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryCloseHistoryEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/position_close", params, nil, &response)
}

// GetDeliveryLiquidationHistory lists liquidation history
func (e *Exchange) GetDeliveryLiquidationHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit uint64, at time.Time) ([]*LiquidationHistoryItemResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if !at.IsZero() {
		params.Set("at", strconv.FormatInt(at.Unix(), 10))
	}
	var response []*LiquidationHistoryItemResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryLiquidationHistoryEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/liquidates", params, nil, &response)
}

// GetDeliverySettlementHistory retrieves settlement history
func (e *Exchange) GetDeliverySettlementHistory(ctx context.Context, settle currency.Code, contract currency.Pair, limit uint64, at time.Time) ([]*SettlementHistoryItemResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if !at.IsZero() {
		params.Set("at", strconv.FormatInt(at.Unix(), 10))
	}
	var response []*SettlementHistoryItemResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliverySettlementHistoryEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/settlements", params, nil, &response)
}

// CreatePriceTriggeredDeliveryOrder creates a price-triggered order
// An initial size of zero closes the whole position and an initial price of zero places a market order
func (e *Exchange) CreatePriceTriggeredDeliveryOrder(ctx context.Context, settle currency.Code, arg *FuturesPriceTriggeredOrderRequest) (*TriggerOrderIDResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.Initial.Contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if arg.Initial.Price < 0 {
		return nil, fmt.Errorf("%w: initial price must not be negative", limits.ErrPriceBelowMin)
	}
	if arg.Initial.TimeInForce != "" && arg.Initial.TimeInForce != gtcTIF && arg.Initial.TimeInForce != iocTIF {
		return nil, fmt.Errorf("%w: %q; only 'gtc' and 'ioc' are allowed", order.ErrUnsupportedTimeInForce, arg.Initial.TimeInForce)
	}
	if arg.Trigger.StrategyType != 0 && arg.Trigger.StrategyType != 1 {
		return nil, fmt.Errorf("%w: %d", errInvalidTriggerStrategyType, arg.Trigger.StrategyType)
	}
	if arg.Trigger.PriceType > 2 {
		return nil, fmt.Errorf("%w: %d", errInvalidTriggerPriceType, arg.Trigger.PriceType)
	}
	if arg.Trigger.Rule != 1 && arg.Trigger.Rule != 2 {
		return nil, fmt.Errorf("%w: %d", errInvalidTriggerRule, arg.Trigger.Rule)
	}
	if arg.Trigger.Price <= 0 {
		return nil, fmt.Errorf("%w: trigger price must be positive", limits.ErrPriceBelowMin)
	}
	switch arg.OrderType {
	case "", "close-long-position", "close-short-position", "plan-close-long-position", "plan-close-short-position":
	default:
		// close-long-order and close-short-order are read only, so Gate rejects them on creation
		return nil, fmt.Errorf("%w: %q", errInvalidAutoOrderType, arg.OrderType)
	}
	var response *TriggerOrderIDResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliverySubmitTriggerOrderEPL, http.MethodPost, "delivery/"+settle.Item.Lower+"/price_orders", nil, arg, &response)
}

// GetDeliveryAllAutoOrder retrieves all auto orders
func (e *Exchange) GetDeliveryAllAutoOrder(ctx context.Context, status string, settle currency.Code, contract currency.Pair, limit, offset uint64) ([]*PriceTriggeredOrderResponse, error) {
	if status != statusOpen && status != statusFinished {
		return nil, fmt.Errorf("%w status %s", errInvalidOrderStatus, status)
	}
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("status", status)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	var response []*PriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryAutoOrdersEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/price_orders", params, nil, &response)
}

// CancelAllDeliveryPriceTriggeredOrder cancels all delivery price triggered orders
func (e *Exchange) CancelAllDeliveryPriceTriggeredOrder(ctx context.Context, settle currency.Code, contract currency.Pair) ([]*PriceTriggeredOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	var response []*PriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryCancelTriggerOrdersEPL, http.MethodDelete, "delivery/"+settle.Item.Lower+"/price_orders", params, nil, &response)
}

// GetSingleDeliveryPriceTriggeredOrder retrieves a single price triggered order
func (e *Exchange) GetSingleDeliveryPriceTriggeredOrder(ctx context.Context, settle currency.Code, orderID string) (*PriceTriggeredOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var response *PriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryGetTriggerOrderEPL, http.MethodGet, "delivery/"+settle.Item.Lower+"/price_orders/"+orderID, nil, nil, &response)
}

// CancelDeliveryPriceTriggeredOrder cancel a price-triggered order
func (e *Exchange) CancelDeliveryPriceTriggeredOrder(ctx context.Context, settle currency.Code, orderID string) (*PriceTriggeredOrderResponse, error) {
	if settle.IsEmpty() {
		return nil, fmt.Errorf("%w; settlement currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var response *PriceTriggeredOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, deliveryCancelTriggerOrderEPL, http.MethodDelete, "delivery/"+settle.Item.Lower+"/price_orders/"+orderID, nil, nil, &response)
}

// ********************************** Options ***************************************************

var (
	errInvalidMMPWindow       = errors.New("market maker protection window must be whole milliseconds of at most 5 seconds")
	errInvalidMMPFrozenPeriod = errors.New("market maker protection frozen period must be non-negative whole milliseconds")
	errInvalidMMPLimit        = errors.New("invalid market maker protection limit")
)

// GetAllOptionsUnderlyings retrieves all option underlyings
func (e *Exchange) GetAllOptionsUnderlyings(ctx context.Context) ([]*OptionUnderlyingResponse, error) {
	var response []*OptionUnderlyingResponse
	return response, e.SendHTTPRequest(ctx, exchange.RestSpot, publicUnderlyingOptionsEPL, "options/underlyings", &response)
}

// GetExpirationTimes returns every expiration time listed for the provided underlying, in the unsorted order Gate
// returns them
func (e *Exchange) GetExpirationTimes(ctx context.Context, underlying string) ([]time.Time, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	var timestamps []types.Time
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, publicExpirationOptionsEPL, common.EncodeURLValues("options/expirations", params), &timestamps); err != nil {
		return nil, err
	}
	expirations := make([]time.Time, len(timestamps))
	for i := range timestamps {
		expirations[i] = timestamps[i].Time()
	}
	return expirations, nil
}

// GetAllContractOfUnderlyingWithinExpiryDate retrieves list of contracts of the specified underlying and expiry time.
func (e *Exchange) GetAllContractOfUnderlyingWithinExpiryDate(ctx context.Context, underlying string, expTime time.Time) ([]*OptionContractResponse, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	if !expTime.IsZero() {
		params.Set("expiration", strconv.FormatInt(expTime.Unix(), 10))
	}
	var contracts []*OptionContractResponse
	return contracts, e.SendHTTPRequest(ctx, exchange.RestSpot, publicContractsOptionsEPL, common.EncodeURLValues("options/contracts", params), &contracts)
}

// GetOptionsSpecifiedContractDetail query specified contract detail
func (e *Exchange) GetOptionsSpecifiedContractDetail(ctx context.Context, contract currency.Pair) (*OptionContractResponse, error) {
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	var contr *OptionContractResponse
	return contr, e.SendHTTPRequest(ctx, exchange.RestSpot, publicContractsOptionsEPL, "options/contracts/"+contract.String(), &contr)
}

// GetSettlementHistory retrieves list of settlement history
func (e *Exchange) GetSettlementHistory(ctx context.Context, underlying string, offset, limit uint64, from, to time.Time) ([]*OptionSettlementResponse, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	var settlements []*OptionSettlementResponse
	return settlements, e.SendHTTPRequest(ctx, exchange.RestSpot, publicSettlementOptionsEPL, common.EncodeURLValues("options/settlements", params), &settlements)
}

// GetOptionsSpecifiedContractsSettlement retrieves a contract's settlement at the given settlement time
func (e *Exchange) GetOptionsSpecifiedContractsSettlement(ctx context.Context, contract currency.Pair, underlying string, at time.Time) (*OptionSettlementResponse, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if at.IsZero() {
		return nil, fmt.Errorf("%w: settlement time is required", common.ErrDateUnset)
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	params.Set("at", strconv.FormatInt(at.Unix(), 10))
	var settlement *OptionSettlementResponse
	return settlement, e.SendHTTPRequest(ctx, exchange.RestSpot, publicSettlementOptionsEPL, common.EncodeURLValues("options/settlements/"+contract.String(), params), &settlement)
}

// GetMyOptionsSettlements retrieves accounts option settlements.
func (e *Exchange) GetMyOptionsSettlements(ctx context.Context, underlying string, contract currency.Pair, offset, limit uint64, from, to time.Time) ([]*MyOptionSettlementResponse, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	var settlements []*MyOptionSettlementResponse
	return settlements, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsSettlementsEPL, http.MethodGet, "options/my_settlements", params, nil, &settlements)
}

// GetOptionsOrderbook returns the orderbook data for the given contract.
func (e *Exchange) GetOptionsOrderbook(ctx context.Context, contract currency.Pair, interval string, limit uint64, withOrderbookID bool) (*OrderbookResponse, error) {
	if contract.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("contract", strings.ToUpper(contract.String()))
	if interval != "" {
		params.Set("interval", interval)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	params.Set("with_id", strconv.FormatBool(withOrderbookID))
	var response *OrderbookResponse
	return response, e.SendHTTPRequest(ctx, exchange.RestSpot, publicOrderbookOptionsEPL, common.EncodeURLValues("options/order_book", params), &response)
}

// GetOptionsTickers lists tickers of options contracts
func (e *Exchange) GetOptionsTickers(ctx context.Context, underlying string) ([]*OptionsTickerResponse, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params := url.Values{}
	params.Set("underlying", strings.ToUpper(underlying))
	var response []*OptionsTickerResponse
	return response, e.SendHTTPRequest(ctx, exchange.RestSpot, publicTickerOptionsEPL, common.EncodeURLValues("options/tickers", params), &response)
}

// GetOptionUnderlyingTickers retrieves options underlying ticker
func (e *Exchange) GetOptionUnderlyingTickers(ctx context.Context, underlying string) (*OptionsUnderlyingTickerResponse, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	var respos *OptionsUnderlyingTickerResponse
	return respos, e.SendHTTPRequest(ctx, exchange.RestSpot, publicUnderlyingTickerOptionsEPL, "options/underlying/tickers/"+underlying, &respos)
}

// GetOptionFuturesCandlesticks retrieves option futures candlesticks
func (e *Exchange) GetOptionFuturesCandlesticks(ctx context.Context, contract currency.Pair, interval kline.Interval, from, to time.Time, limit uint64) ([]*FuturesCandlestickResponse, error) {
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	params.Set("contract", contract.String())
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if int64(interval) != 0 {
		intervalString, err := getIntervalString(interval)
		if err != nil {
			return nil, err
		}
		params.Set("interval", intervalString)
	}
	var candles []*FuturesCandlestickResponse
	return candles, e.SendHTTPRequest(ctx, exchange.RestSpot, publicCandleSticksOptionsEPL, common.EncodeURLValues("options/candlesticks", params), &candles)
}

// GetOptionFuturesMarkPriceCandlesticks retrieves index price candlesticks of an underlying
func (e *Exchange) GetOptionFuturesMarkPriceCandlesticks(ctx context.Context, underlying string, limit uint64, from, to time.Time, interval kline.Interval) ([]*FuturesCandlestickResponse, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if int64(interval) != 0 {
		intervalString, err := getIntervalString(interval)
		if err != nil {
			return nil, err
		}
		params.Set("interval", intervalString)
	}
	var candles []*FuturesCandlestickResponse
	return candles, e.SendHTTPRequest(ctx, exchange.RestSpot, publicMarkpriceCandleSticksOptionsEPL, common.EncodeURLValues("options/underlying/candlesticks", params), &candles)
}

// GetOptionsTradeHistory retrieves options trade history
func (e *Exchange) GetOptionsTradeHistory(ctx context.Context, contract currency.Pair, callType string, offset, limit uint64, from, to time.Time) ([]*OptionTradeResponse, error) {
	params := url.Values{}
	callType = strings.ToUpper(callType)
	if callType == "C" || callType == "P" {
		params.Set("type", callType)
	}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	var trades []*OptionTradeResponse
	return trades, e.SendHTTPRequest(ctx, exchange.RestSpot, publicTradeHistoryOptionsEPL, common.EncodeURLValues("options/trades", params), &trades)
}

// GetOptionAccount lists option accounts
func (e *Exchange) GetOptionAccount(ctx context.Context) (*OptionAccountResponse, error) {
	var resp *OptionAccountResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsAccountsEPL, http.MethodGet, "options/accounts", nil, nil, &resp)
}

// GetAccountChangingHistory retrieves list of account changing history
// possible change type values are: "dnw": Deposit & Withdrawal, "prem": Trading premium, "fee": Trading fee, "refr": Referrer rebate, and "set": Settlement P&L
func (e *Exchange) GetAccountChangingHistory(ctx context.Context, offset, limit uint64, from, to time.Time, changingType string) ([]*AccountBookResponse, error) {
	params := url.Values{}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if changingType != "" {
		params.Set("type", changingType)
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var accountBook []*AccountBookResponse
	return accountBook, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsAccountBooksEPL, http.MethodGet, "options/account_book", params, nil, &accountBook)
}

// GetUsersPositionSpecifiedUnderlying lists user's positions of specified underlying
func (e *Exchange) GetUsersPositionSpecifiedUnderlying(ctx context.Context, underlying string) ([]*UsersPositionForUnderlyingResponse, error) {
	params := url.Values{}
	if underlying != "" {
		params.Set("underlying", underlying)
	}
	var response []*UsersPositionForUnderlyingResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsPositions, http.MethodGet, "options/positions", params, nil, &response)
}

// GetSpecifiedContractPosition retrieves specified contract position
func (e *Exchange) GetSpecifiedContractPosition(ctx context.Context, contract currency.Pair) (*UsersPositionForUnderlyingResponse, error) {
	if contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	var response *UsersPositionForUnderlyingResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsPositions, http.MethodGet, "options/positions/"+contract.String(), nil, nil, &response)
}

// GetUsersLiquidationHistoryForSpecifiedUnderlying retrieves user's liquidation history of specified underlying
func (e *Exchange) GetUsersLiquidationHistoryForSpecifiedUnderlying(ctx context.Context, underlying string, contract currency.Pair) ([]*ContractClosePositionResponse, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	var response []*ContractClosePositionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsLiquidationHistoryEPL, http.MethodGet, "options/position_close", params, nil, &response)
}

// PlaceOptionOrder creates an options order
// A zero size with Close set closes the position, and a zero price with an IOC time in force places a market order
func (e *Exchange) PlaceOptionOrder(ctx context.Context, arg *OptionOrderRequest) (*OptionOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if arg.Close != (arg.OrderSize == 0) {
		return nil, fmt.Errorf("%w: size must be zero when closing a position and non-zero otherwise", errInvalidOrderSize)
	}
	if arg.Price < 0 {
		return nil, fmt.Errorf("%w: price must not be negative", limits.ErrPriceBelowMin)
	}
	var response *OptionOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsSubmitOrderEPL, http.MethodPost, "options/orders", nil, arg, &response)
}

// GetOptionFuturesOrders retrieves options orders with the given status
func (e *Exchange) GetOptionFuturesOrders(ctx context.Context, contract currency.Pair, underlying, status string, offset, limit uint64, from, to time.Time) ([]*OptionOrderResponse, error) {
	status = strings.ToLower(status)
	if status != statusOpen && status != statusFinished {
		return nil, fmt.Errorf("%w: %q; only 'open' and 'finished' are supported", errInvalidOrderStatus, status)
	}
	params := url.Values{}
	params.Set("status", status)
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if underlying != "" {
		params.Set("underlying", underlying)
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var response []*OptionOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsOrdersEPL, http.MethodGet, "options/orders", params, nil, &response)
}

// CancelMultipleOptionOpenOrders cancels all open orders matched
func (e *Exchange) CancelMultipleOptionOpenOrders(ctx context.Context, contract currency.Pair, underlying, side string) ([]*OptionOrderResponse, error) {
	// Dropping an unrecognised side would widen the cancellation to both sides of the book
	if side != "" && side != order.Ask.Lower() && side != order.Bid.Lower() {
		return nil, fmt.Errorf("%w: %q; only 'ask' and 'bid' are supported", order.ErrSideIsInvalid, side)
	}
	params := url.Values{}
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if underlying != "" {
		params.Set("underlying", underlying)
	}
	if side != "" {
		params.Set("side", side)
	}
	var response []*OptionOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsCancelOrdersEPL, http.MethodDelete, "options/orders", params, nil, &response)
}

// AmendOptionOrder amends the price and size of an open options order
func (e *Exchange) AmendOptionOrder(ctx context.Context, orderID string, arg *AmendOptionOrderRequest) (*OptionOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Contract.IsEmpty() {
		return nil, fmt.Errorf("%w: contract pair is required", currency.ErrCurrencyPairEmpty)
	}
	if arg.Price <= 0 {
		return nil, fmt.Errorf("%w: price must be positive", limits.ErrPriceBelowMin)
	}
	if arg.Size == 0 {
		return nil, fmt.Errorf("%w: size must not be zero", errInvalidOrderSize)
	}
	var response *OptionOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsAmendOrderEPL, http.MethodPut, "options/orders/"+orderID, nil, arg, &response)
}

// GetSingleOptionOrder retrieves a single option order
func (e *Exchange) GetSingleOptionOrder(ctx context.Context, orderID string) (*OptionOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var o *OptionOrderResponse
	return o, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsOrderEPL, http.MethodGet, "options/orders/"+orderID, nil, nil, &o)
}

// CancelOptionSingleOrder cancel a single order.
func (e *Exchange) CancelOptionSingleOrder(ctx context.Context, orderID string) (*OptionOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var response *OptionOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsCancelOrderEPL, http.MethodDelete, "options/orders/"+orderID, nil, nil, &response)
}

// CountdownCancelOptionOrders sets a countdown after which Gate cancels open options orders unless the countdown is
// renewed before it ends; a zero timeout clears the countdown
func (e *Exchange) CountdownCancelOptionOrders(ctx context.Context, arg *CountdownCancelOptionOrdersRequest) (*TriggerTimeResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Timeout < 0 || (arg.Timeout != 0 && arg.Timeout < 5*time.Second) {
		return nil, fmt.Errorf("%w: %s", errInvalidTimeout, arg.Timeout)
	}
	var response *TriggerTimeResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsCountdownCancelEPL, http.MethodPost, "options/countdown_cancel_all", nil, arg, &response)
}

// GetMyOptionsTradingHistory retrieves authenticated account's option trading history
func (e *Exchange) GetMyOptionsTradingHistory(ctx context.Context, underlying string, contract currency.Pair, offset, limit uint64, from, to time.Time) ([]*OptionTradingHistoryResponse, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	params := url.Values{}
	params.Set("underlying", underlying)
	if contract.IsPopulated() {
		params.Set("contract", contract.String())
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatUint(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if err := setUnixTimeRangeParams(&params, from, to); err != nil {
		return nil, err
	}
	var resp []*OptionTradingHistoryResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsTradingHistoryEPL, http.MethodGet, "options/my_trades", params, nil, &resp)
}

// SetOptionsMarketMakerProtection sets the market maker protection of an options underlying
func (e *Exchange) SetOptionsMarketMakerProtection(ctx context.Context, arg *OptionsMarketMakerProtectionRequest) (*OptionsMarketMakerProtectionResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Underlying == "" {
		return nil, errInvalidUnderlying
	}
	// Gate takes whole milliseconds, and truncating a sub-millisecond window to zero would disable protection
	if arg.Window < 0 || arg.Window > 5*time.Second || arg.Window%time.Millisecond != 0 {
		return nil, fmt.Errorf("%w: %s", errInvalidMMPWindow, arg.Window)
	}
	if arg.FrozenPeriod < 0 || arg.FrozenPeriod%time.Millisecond != 0 {
		return nil, fmt.Errorf("%w: %s", errInvalidMMPFrozenPeriod, arg.FrozenPeriod)
	}
	if arg.QuantityLimit <= 0 {
		return nil, fmt.Errorf("%w: quantity limit must be positive", errInvalidMMPLimit)
	}
	if arg.DeltaLimit <= 0 {
		return nil, fmt.Errorf("%w: delta limit must be positive", errInvalidMMPLimit)
	}
	var response *OptionsMarketMakerProtectionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsSetMMPEPL, http.MethodPost, "options/mmp", nil, arg, &response)
}

// GetOptionsMarketMakerProtection retrieves the market maker protection of an options underlying, or of every
// underlying when underlying is empty
func (e *Exchange) GetOptionsMarketMakerProtection(ctx context.Context, underlying string) ([]*OptionsMarketMakerProtectionResponse, error) {
	params := url.Values{}
	if underlying != "" {
		params.Set("underlying", underlying)
	}
	var response []*OptionsMarketMakerProtectionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsGetMMPEPL, http.MethodGet, "options/mmp", params, nil, &response)
}

// ResetOptionsMarketMakerProtection resets the market maker protection of an options underlying, unfreezing trading
// after the protection triggers
func (e *Exchange) ResetOptionsMarketMakerProtection(ctx context.Context, underlying string) (*OptionsMarketMakerProtectionResponse, error) {
	if underlying == "" {
		return nil, errInvalidUnderlying
	}
	var response *OptionsMarketMakerProtectionResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, optionsResetMMPEPL, http.MethodPost, "options/mmp/reset", nil, map[string]string{"underlying": underlying}, &response)
}

// ********************************** Flash_SWAP *************************

// GetSupportedFlashSwapCurrencyPairs retrieves the flash swap currency pairs, where BTC_USDT means selling BTC for USDT
// A pair being swappable in one direction does not mean the reverse swap is supported
func (e *Exchange) GetSupportedFlashSwapCurrencyPairs(ctx context.Context, ccy currency.Code, limit, page uint64) ([]*FlashSwapCurrencyPairResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	var pairs []*FlashSwapCurrencyPairResponse
	return pairs, e.SendHTTPRequest(ctx, exchange.RestSpot, publicFlashSwapEPL, common.EncodeURLValues("flash_swap/currency_pairs", params), &pairs)
}

// CreateFlashSwapOrder creates a flash swap order from a preview returned by InitiateFlashSwapOrderReview
func (e *Exchange) CreateFlashSwapOrder(ctx context.Context, arg *FlashSwapOrderRequest) (*FlashSwapOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.PreviewID == "" {
		return nil, errMissingPreviewID
	}
	if arg.BuyCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, buy currency can not empty", currency.ErrCurrencyCodeEmpty)
	}
	if arg.SellCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, sell currency can not empty", currency.ErrCurrencyCodeEmpty)
	}
	if arg.SellAmount <= 0 {
		return nil, fmt.Errorf("%w, sell_amount can not be less than or equal to 0", order.ErrAmountIsInvalid)
	}
	if arg.BuyAmount <= 0 {
		return nil, fmt.Errorf("%w, buy_amount amount can not be less than or equal to 0", order.ErrAmountIsInvalid)
	}
	var response *FlashSwapOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, flashSwapOrderEPL, http.MethodPost, "flash_swap/orders", nil, arg, &response)
}

// GetAllFlashSwapOrders retrieves flash swap orders, optionally filtered by status (1 success, 2 failure) and currencies
// reverse is always sent, so false lists the oldest orders first rather than applying Gate's newest first default
func (e *Exchange) GetAllFlashSwapOrders(ctx context.Context, status uint64, sellCurrency, buyCurrency currency.Code, reverse bool, limit, page uint64) ([]*FlashSwapOrderResponse, error) {
	params := url.Values{}
	if status == 1 || status == 2 {
		params.Set("status", strconv.FormatUint(status, 10))
	}
	if !sellCurrency.IsEmpty() {
		params.Set("sell_currency", sellCurrency.String())
	}
	if !buyCurrency.IsEmpty() {
		params.Set("buy_currency", buyCurrency.String())
	}
	params.Set("reverse", strconv.FormatBool(reverse))
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var response []*FlashSwapOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, flashGetOrdersEPL, http.MethodGet, "flash_swap/orders", params, nil, &response)
}

// GetSingleFlashSwapOrder retrieves a single flash swap order
func (e *Exchange) GetSingleFlashSwapOrder(ctx context.Context, orderID string) (*FlashSwapOrderResponse, error) {
	if orderID == "" {
		return nil, fmt.Errorf("%w, flash order order_id must not be empty", order.ErrOrderIDNotSet)
	}
	var response *FlashSwapOrderResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, flashGetOrderEPL, http.MethodGet, "flash_swap/orders/"+orderID, nil, nil, &response)
}

// InitiateFlashSwapOrderReview previews a flash swap, quoting the amount received for SellAmount or the amount needed for
// BuyAmount; the returned preview ID is required to create the order
func (e *Exchange) InitiateFlashSwapOrderReview(ctx context.Context, arg *FlashSwapPreviewRequest) (*InitFlashSwapOrderPreviewResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.SellCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, sell currency can not empty", currency.ErrCurrencyCodeEmpty)
	}
	if arg.BuyCurrency.IsEmpty() {
		return nil, fmt.Errorf("%w, buy currency can not empty", currency.ErrCurrencyCodeEmpty)
	}
	if arg.SellAmount <= 0 && arg.BuyAmount <= 0 {
		return nil, fmt.Errorf("%w: either sell_amount or buy_amount is required", order.ErrAmountIsInvalid)
	}
	var response *InitFlashSwapOrderPreviewResponse
	return response, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, flashOrderReviewEPL, http.MethodPost, "flash_swap/orders/preview", nil, arg, &response)
}

// GetDualInvestmentProductList retrieves the dual investment product list
func (e *Exchange) GetDualInvestmentProductList(ctx context.Context, arg *DualInvestmentProductListRequest) ([]*DualInvestmentPlanResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if arg.PlanID != 0 {
		params.Set("plan_id", strconv.FormatUint(arg.PlanID, 10))
	}
	if !arg.Coin.IsEmpty() {
		params.Set("coin", arg.Coin.String())
	}
	if arg.Type != "" {
		params.Set("type", arg.Type)
	}
	if !arg.QuoteCurrency.IsEmpty() {
		params.Set("quote_currency", arg.QuoteCurrency.String())
	}
	if arg.Sort != "" {
		params.Set("sort", arg.Sort)
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.PageSize > 0 {
		params.Set("page_size", strconv.FormatUint(arg.PageSize, 10))
	}
	var resp []*DualInvestmentPlanResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, earnDualInvestmentPlanEPL, common.EncodeURLValues("earn/dual/investment_plan", params), &resp)
}

// GetDualInvestmentOrderList retrieves dual investment orders
func (e *Exchange) GetDualInvestmentOrderList(ctx context.Context, arg *DualInvestmentOrderListRequest) ([]*DualInvestmentOrderDetailResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	// The window filters settlement times, which open positions have in the future, so only its order is checked
	if !arg.From.IsZero() && !arg.To.IsZero() && arg.To.Before(arg.From) {
		return nil, common.ErrStartAfterEnd
	}
	params := url.Values{}
	if !arg.From.IsZero() {
		params.Set("from", strconv.FormatInt(arg.From.Unix(), 10))
	}
	if !arg.To.IsZero() {
		params.Set("to", strconv.FormatInt(arg.To.Unix(), 10))
	}
	if arg.Type != "" {
		params.Set("type", arg.Type)
	}
	if arg.Status != "" {
		params.Set("status", arg.Status)
	}
	if !arg.Coin.IsEmpty() {
		params.Set("coin", arg.Coin.String())
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []*DualInvestmentOrderDetailResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnDualOrdersEPL, http.MethodGet, "earn/dual/orders", params, nil, &resp)
}

// PlaceDualInvestmentOrder places a dual investment order
func (e *Exchange) PlaceDualInvestmentOrder(ctx context.Context, arg *DualInvestmentOrderRequest) (*DualInvestmentOrderDetailResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.PlanID == "" {
		return nil, errPlanIDRequired
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if arg.Text != "" && !strings.HasPrefix(arg.Text, "t-") {
		return nil, errInvalidTextPrefix
	}
	var resp *DualInvestmentOrderDetailResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnDualPlaceOrderEPL, http.MethodPost, "earn/dual/orders", nil, arg, &resp)
}

// GetDualCurrencyEarningAssets retrieves the user's dual investment assets and interest
func (e *Exchange) GetDualCurrencyEarningAssets(ctx context.Context) (*DualCurrencyEarningAssetResponse, error) {
	var resp *DualCurrencyEarningAssetResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnDualBalanceEPL, http.MethodGet, "earn/dual/balance", nil, nil, &resp)
}

// GetDualCurrencyEarlyRedemptionPreview previews a dual investment order's early redemption; the returned request
// ID is required by RedeemDualCurrencyOrder
func (e *Exchange) GetDualCurrencyEarlyRedemptionPreview(ctx context.Context, orderID string) (*DualCurrencyEarlyRedemptionResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("order_id", orderID)
	var resp *DualCurrencyEarlyRedemptionResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnDualOrderRefundPreviewEPL, http.MethodGet, "earn/dual/order-refund-preview", params, nil, &resp)
}

// RedeemDualCurrencyOrder redeems a dual investment order early using the request ID from its redemption preview
func (e *Exchange) RedeemDualCurrencyOrder(ctx context.Context, orderID, requestID string) error {
	if orderID == "" {
		return order.ErrOrderIDNotSet
	}
	if requestID == "" {
		return errOrderRefundRequestIDRequired
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnDualOrderRefundEPL, http.MethodPost, "earn/dual/order-refund", nil, map[string]string{
		"order_id": orderID,
		"req_id":   requestID,
	}, nil)
}

// ModifyDualCurrencyOrderReinvest switches a dual investment order's reinvestment on or off
func (e *Exchange) ModifyDualCurrencyOrderReinvest(ctx context.Context, arg *DualModifyOrderReinvestRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.OrderID == 0 {
		return order.ErrOrderIDNotSet
	}
	if arg.Status > 1 {
		return fmt.Errorf("%w: 0 switches reinvestment off and 1 switches it on", errInvalidReinvestStatus)
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnDualModifyOrderReinvestEPL, http.MethodPost, "earn/dual/modify-order-reinvest", nil, arg, nil)
}

// GetDualCurrencyRecommendedProjects retrieves recommended dual investment products
// mode is normal, senior, apy_up, ep_down, ep_up, dt_down or dt_up; orderType is call or put; excludeProductIDs
// lists previously recommended products to leave out
func (e *Exchange) GetDualCurrencyRecommendedProjects(ctx context.Context, mode string, coin currency.Code, orderType string, excludeProductIDs []uint64) ([]*DualRecommendedProjectResponse, error) {
	params := url.Values{}
	if mode != "" {
		params.Set("mode", mode)
	}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if len(excludeProductIDs) > 0 {
		productIDs := make([]string, len(excludeProductIDs))
		for i, productID := range excludeProductIDs {
			productIDs[i] = strconv.FormatUint(productID, 10)
		}
		params.Set("history_pids", strings.Join(productIDs, ","))
	}
	var resp []*DualRecommendedProjectResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnDualProjectRecommendEPL, http.MethodGet, "earn/dual/project-recommend", params, nil, &resp)
}

// CreateAutoInvestPlan creates an auto invest plan
func (e *Exchange) CreateAutoInvestPlan(ctx context.Context, arg *CreateAutoInvestPlanRequest) (*AutoInvestPlanResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.PlanMoney.IsEmpty() {
		return nil, fmt.Errorf("%w: plan pricing currency is required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.PlanAmount <= 0 {
		return nil, fmt.Errorf("%w: plan amount is required", limits.ErrAmountBelowMin)
	}
	// The period day only applies to monthly and weekly cycles; daily and hourly cycles ignore it
	switch arg.PlanPeriodType {
	case "":
		return nil, errPeriodTypeRequired
	case "monthly":
		if arg.PlanPeriodDay < 1 || arg.PlanPeriodDay > 30 {
			return nil, fmt.Errorf("%w: monthly plans run on a day from 1 to 30", errInvalidPlanPeriodDay)
		}
	case "weekly", "biweekly":
		if arg.PlanPeriodDay < 1 || arg.PlanPeriodDay > 7 {
			return nil, fmt.Errorf("%w: weekly plans run on a weekday from 1 to 7", errInvalidPlanPeriodDay)
		}
	}
	if arg.PlanPeriodHour > 23 {
		return nil, fmt.Errorf("%w: must be an hour from 0 to 23", errInvalidPlanPeriodHour)
	}
	if err := validateAutoInvestPlanItems(arg.Items); err != nil {
		return nil, err
	}
	var resp *AutoInvestPlanResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnAutoInvestCreatePlanEPL, http.MethodPost, "earn/autoinvest/plans/create", nil, arg, &resp)
}

// validateAutoInvestPlanItems checks each portfolio item names a currency and its share of the portfolio
func validateAutoInvestPlanItems(items []*AutoInvestPlanItem) error {
	if len(items) == 0 {
		return fmt.Errorf("%w: at least one portfolio item is required", errNoValidParameterPassed)
	}
	for _, item := range items {
		if item == nil {
			return fmt.Errorf("%w: portfolio item", common.ErrNilPointer)
		}
		if item.Asset.IsEmpty() {
			return currency.ErrCurrencyCodeEmpty
		}
		if item.Ratio <= 0 {
			return fmt.Errorf("%w: proportion of this currency in the portfolio is required", order.ErrAmountMustBeSet)
		}
	}
	return nil
}

// UpdateAutoInvestPlan updates an auto invest plan's fund source and fund flow
func (e *Exchange) UpdateAutoInvestPlan(ctx context.Context, arg *AutoInvestPlanUpdateRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.PlanID == 0 {
		return errPlanIDRequired
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnAutoInvestUpdatePlanEPL, http.MethodPost, "earn/autoinvest/plans/update", nil, arg, nil)
}

// StopAutoInvestPlan stops an auto invest plan
func (e *Exchange) StopAutoInvestPlan(ctx context.Context, planID uint64) error {
	if planID == 0 {
		return errPlanIDRequired
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnAutoInvestStopPlanEPL, http.MethodPost, "earn/autoinvest/plans/stop", nil, &AutoInvestPlanStopRequest{PlanID: planID}, nil)
}

// AddAutoInvestPlanPosition invests an amount into an auto invest plan immediately
func (e *Exchange) AddAutoInvestPlanPosition(ctx context.Context, planID uint64, amount float64) error {
	if planID == 0 {
		return errPlanIDRequired
	}
	if amount <= 0 {
		return order.ErrAmountIsInvalid
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnAutoInvestAddPositionEPL, http.MethodPost, "earn/autoinvest/plans/add_position", nil, &AutoInvestPlanAddPositionRequest{
		PlanID: planID,
		Amount: types.Number(amount),
	}, nil)
}

// GetAutoInvestSupportedCoins retrieves the currencies supported by auto invest for a pricing currency of USDT or BTC
// The pricing currency defaults to USDT when empty
func (e *Exchange) GetAutoInvestSupportedCoins(ctx context.Context, planMoney currency.Code) ([]*AutoInvestCoinItemResponse, error) {
	params := url.Values{}
	if !planMoney.IsEmpty() {
		params.Set("plan_money", planMoney.String())
	}
	var resp []*AutoInvestCoinItemResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnAutoInvestCoinsEPL, http.MethodGet, "earn/autoinvest/coins", params, nil, &resp)
}

// GetAutoInvestMinimumAmount retrieves the minimum investment amount for a portfolio
func (e *Exchange) GetAutoInvestMinimumAmount(ctx context.Context, arg *AutoInvestMinAmountRequest) (*AutoInvestMinAmountResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Money.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if err := validateAutoInvestPlanItems(arg.Items); err != nil {
		return nil, err
	}
	var resp *AutoInvestMinAmountResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnAutoInvestMinAmountEPL, http.MethodPost, "earn/autoinvest/min_invest_amount", nil, arg, &resp)
}

// GetAutoInvestPlanExecutionRecords retrieves an auto invest plan's execution records; pageSize is at most 100
func (e *Exchange) GetAutoInvestPlanExecutionRecords(ctx context.Context, planID, page, pageSize uint64) (*AutoInvestPlanExecutionRecordsResponse, error) {
	if planID == 0 {
		return nil, errPlanIDRequired
	}
	params := url.Values{}
	params.Set("plan_id", strconv.FormatUint(planID, 10))
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if pageSize > 0 {
		params.Set("page_size", strconv.FormatUint(pageSize, 10))
	}
	var resp *AutoInvestPlanExecutionRecordsResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnAutoInvestPlanRecordsEPL, http.MethodGet, "earn/autoinvest/plans/records", params, nil, &resp)
}

// GetAutoInvestPlanOrderDetails retrieves the orders placed by an auto invest plan execution record
func (e *Exchange) GetAutoInvestPlanOrderDetails(ctx context.Context, planID, recordID uint64) ([]*AutoInvestOrderItemResponse, error) {
	if planID == 0 {
		return nil, errPlanIDRequired
	}
	if recordID == 0 {
		return nil, errRecordIDRequired
	}
	params := url.Values{}
	params.Set("plan_id", strconv.FormatUint(planID, 10))
	params.Set("record_id", strconv.FormatUint(recordID, 10))
	var resp []*AutoInvestOrderItemResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnAutoInvestOrdersEPL, http.MethodGet, "earn/autoinvest/orders", params, nil, &resp)
}

// GetAutoInvestCurrencyConfig retrieves the maximum investment amount of each auto invest pricing currency
func (e *Exchange) GetAutoInvestCurrencyConfig(ctx context.Context) ([]*AutoInvestConfigItemResponse, error) {
	var resp []*AutoInvestConfigItemResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnAutoInvestConfigEPL, http.MethodGet, "earn/autoinvest/config", nil, nil, &resp)
}

// GetAutoInvestPlanDetails retrieves an auto invest plan
func (e *Exchange) GetAutoInvestPlanDetails(ctx context.Context, planID uint64) (*AutoInvestPlanDetailsResponse, error) {
	if planID == 0 {
		return nil, errPlanIDRequired
	}
	params := url.Values{}
	params.Set("plan_id", strconv.FormatUint(planID, 10))
	var resp *AutoInvestPlanDetailsResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnAutoInvestPlanDetailEPL, http.MethodGet, "earn/autoinvest/plans/detail", params, nil, &resp)
}

// GetAutoInvestPlanList retrieves auto invest plans; status is active or history and pageSize is at most 100
func (e *Exchange) GetAutoInvestPlanList(ctx context.Context, status string, page, pageSize uint64) (*AutoInvestPlanListResponse, error) {
	if status == "" {
		return nil, errPlanStatusRequired
	}
	params := url.Values{}
	params.Set("status", status)
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if pageSize > 0 {
		params.Set("page_size", strconv.FormatUint(pageSize, 10))
	}
	var resp *AutoInvestPlanListResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnAutoInvestPlanListEPL, http.MethodGet, "earn/autoinvest/plans/list_info", params, nil, &resp)
}

// GetFixedTermProducts retrieves fixed-term earn products; productType is 1 for regular or 2 for VIP products and is
// not filtered on when 0
func (e *Exchange) GetFixedTermProducts(ctx context.Context, ccy currency.Code, productType uint8, page, limit uint64) (*FixedTermProductsResponse, error) {
	if page == 0 {
		return nil, errPageRequired
	}
	if limit == 0 {
		return nil, fmt.Errorf("%w: limit is required", errInvalidLimit)
	}
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("asset", ccy.String())
	}
	if productType != 0 {
		params.Set("type", strconv.FormatUint(uint64(productType), 10))
	}
	params.Set("page", strconv.FormatUint(page, 10))
	params.Set("limit", strconv.FormatUint(limit, 10))
	var resp gateioAPIResponse[*FixedTermProductsResponse]
	return resp.Data, e.SendHTTPRequest(ctx, exchange.RestSpot, earnFixedTermProductEPL, common.EncodeURLValues("earn/fixed-term/product", params), &resp)
}

// GetFixedTermProductsByAsset retrieves a currency's fixed-term earn products in ascending order of term
// productType is empty or "1" for regular products, "2" for VIP products and "0" for all products
func (e *Exchange) GetFixedTermProductsByAsset(ctx context.Context, ccy currency.Code, productType string) (*FixedTermProductsByAssetResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	if productType != "" {
		params.Set("type", productType)
	}
	var resp gateioAPIResponse[*FixedTermProductsByAssetResponse]
	return resp.Data, e.SendHTTPRequest(ctx, exchange.RestSpot, earnFixedTermProductListEPL, common.EncodeURLValues("earn/fixed-term/product/"+ccy.String()+"/list", params), &resp)
}

// SetFixedTermSubscriptionOrder subscribes to a fixed-term earn product
func (e *Exchange) SetFixedTermSubscriptionOrder(ctx context.Context, arg *FixedTermSubscriptionRequest) (*OrderIDResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.ProductID == 0 {
		return nil, errProductIDRequired
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	var resp gateioAPIResponse[*OrderIDResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnFixedTermLendEPL, http.MethodPost, "earn/fixed-term/user/lend", nil, arg, &resp)
}

// GetFixedTermSubscriptionOrders retrieves fixed-term earn subscription orders
func (e *Exchange) GetFixedTermSubscriptionOrders(ctx context.Context, arg *FixedTermSubscriptionOrdersRequest) (*FixedTermSubscriptionOrdersResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.OrderType != 1 && arg.OrderType != 2 {
		return nil, fmt.Errorf("%w: 1 for current orders or 2 for historical orders", order.ErrTypeIsInvalid)
	}
	if arg.Page == 0 {
		return nil, errPageRequired
	}
	if arg.Limit == 0 {
		return nil, fmt.Errorf("%w: limit is required", errInvalidLimit)
	}
	params := url.Values{}
	if err := setFixedTermBusinessFilter(params, arg.BusinessFilter); err != nil {
		return nil, err
	}
	if arg.ProductID != 0 {
		params.Set("product_id", strconv.FormatUint(arg.ProductID, 10))
	}
	if arg.OrderID != 0 {
		params.Set("order_id", strconv.FormatUint(arg.OrderID, 10))
	}
	if !arg.Asset.IsEmpty() {
		params.Set("asset", arg.Asset.String())
	}
	params.Set("order_type", strconv.FormatUint(uint64(arg.OrderType), 10))
	params.Set("page", strconv.FormatUint(arg.Page, 10))
	params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	if arg.SubBusiness != 0 {
		params.Set("sub_business", strconv.FormatUint(arg.SubBusiness, 10))
	}
	var resp gateioAPIResponse[*FixedTermSubscriptionOrdersResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnFixedTermLendsEPL, http.MethodGet, "earn/fixed-term/user/lend", params, nil, &resp)
}

// RedeemFixedTermOrder redeems a fixed-term earn order before it matures
func (e *Exchange) RedeemFixedTermOrder(ctx context.Context, orderID string) error {
	if orderID == "" {
		return order.ErrOrderIDNotSet
	}
	var resp gateioAPIResponse[struct{}]
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnFixedTermPreRedeemEPL, http.MethodPost, "earn/fixed-term/user/pre-redeem", nil, map[string]string{"order_id": orderID}, &resp)
}

// GetFixedTermSubscriptionHistory retrieves fixed-term earn subscription, redemption, interest or bonus history
func (e *Exchange) GetFixedTermSubscriptionHistory(ctx context.Context, arg *FixedTermHistoryRequest) (*FixedTermHistoryResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Type == 0 {
		return nil, errHistoryTypeRequired
	}
	if arg.Type > 4 {
		return nil, fmt.Errorf("%w: 1 subscription, 2 redemption, 3 interest or 4 bonus reward", errHistoryTypeRequired)
	}
	if arg.Page == 0 {
		return nil, errPageRequired
	}
	if arg.Limit == 0 {
		return nil, fmt.Errorf("%w: limit is required", errInvalidLimit)
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if err := setFixedTermBusinessFilter(params, arg.BusinessFilter); err != nil {
		return nil, err
	}
	if arg.ProductID != 0 {
		params.Set("product_id", strconv.FormatUint(arg.ProductID, 10))
	}
	if arg.OrderID != "" {
		params.Set("order_id", arg.OrderID)
	}
	if !arg.Asset.IsEmpty() {
		params.Set("asset", arg.Asset.String())
	}
	params.Set("type", strconv.FormatUint(uint64(arg.Type), 10))
	params.Set("page", strconv.FormatUint(arg.Page, 10))
	params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	if !arg.StartTime.IsZero() {
		params.Set("start_at", strconv.FormatInt(arg.StartTime.Unix(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("end_at", strconv.FormatInt(arg.EndTime.Unix(), 10))
	}
	if arg.SubBusiness != 0 {
		params.Set("sub_business", strconv.FormatUint(arg.SubBusiness, 10))
	}
	var resp gateioAPIResponse[*FixedTermHistoryResponse]
	return resp.Data, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnFixedTermHistoryEPL, http.MethodGet, "earn/fixed-term/user/history", params, nil, &resp)
}

// setFixedTermBusinessFilter encodes fixed-term business filters as the JSON array the business_filter parameter expects
func setFixedTermBusinessFilter(params url.Values, filters []FixedTermBusinessFilterRequest) error {
	if len(filters) == 0 {
		return nil
	}
	filter, err := json.Marshal(filters)
	if err != nil {
		return fmt.Errorf("error encoding fixed-term business filter: %w", err)
	}
	params.Set("business_filter", string(filter))
	return nil
}

// ********************************* Trading Fee calculation ********************************

// GetFee returns an estimate of fee based on type of transaction
func (e *Exchange) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (fee float64, err error) {
	if feeBuilder == nil {
		return 0, fmt.Errorf("%T %w", feeBuilder, common.ErrNilPointer)
	}
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		pair := feeBuilder.Pair
		if pair.IsPopulated() {
			// Callers build fee requests in their own pair format, while Gate only recognises its underscored pairs
			formatted, err := e.FormatExchangeCurrency(pair, asset.Spot)
			if err != nil {
				return 0, err
			}
			pair = formatted
		}
		feePairs, err := e.GetPersonalTradingFee(ctx, pair, currency.EMPTYCODE)
		if err != nil {
			return 0, err
		}
		if feeBuilder.IsMaker {
			fee = calculateTradingFee(feePairs.MakerFee.Float64(),
				feeBuilder.PurchasePrice,
				feeBuilder.Amount)
		} else {
			fee = calculateTradingFee(feePairs.TakerFee.Float64(),
				feeBuilder.PurchasePrice,
				feeBuilder.Amount)
		}
	case exchange.CryptocurrencyWithdrawalFee:
		fee = getCryptocurrencyWithdrawalFee(feeBuilder.Pair.Base)
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}
	return fee, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.002 * price * amount
}

func calculateTradingFee(feeForPair, purchasePrice, amount float64) float64 {
	return feeForPair * purchasePrice * amount
}

func getCryptocurrencyWithdrawalFee(c currency.Code) float64 {
	return WithdrawalFees[c]
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

// GetAccountDetails retrieves account details
func (e *Exchange) GetAccountDetails(ctx context.Context) (*AccountDetailsResponse, error) {
	var resp *AccountDetailsResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountDetailEPL, http.MethodGet, "account/detail", nil, nil, &resp)
}

// GetAccountMainKeys retrieves every API key of the main account
func (e *Exchange) GetAccountMainKeys(ctx context.Context) ([]*AccountMainKeyResponse, error) {
	var resp []*AccountMainKeyResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountMainKeysEPL, http.MethodGet, "account/main_keys", nil, nil, &resp)
}

// GetUserTransactionRateLimitInfo retrieves user transaction rate limit info. Gate notes the endpoint is not yet open
// for use
func (e *Exchange) GetUserTransactionRateLimitInfo(ctx context.Context) ([]*UserTransactionRateLimitInfoResponse, error) {
	var resp []*UserTransactionRateLimitInfoResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountRateLimitEPL, http.MethodGet, "account/rate_limit", nil, nil, &resp)
}

// validate validates the FuturesOrderCreateRequest
func (c *FuturesOrderCreateRequest) validate(isRest bool) error {
	if err := common.NilGuard(c); err != nil {
		return err
	}
	return validateOrderCreateParams(c.Contract, c.Size, c.Price, c.ClosePosition, c.AutoSize, c.TimeInForce, c.Text, c.Settle, isRest)
}

// validate validates the DeliveryOrderCreateRequest
func (c *DeliveryOrderCreateRequest) validate(isRest bool) error {
	if err := common.NilGuard(c); err != nil {
		return err
	}
	return validateOrderCreateParams(c.Contract, c.Size, c.Price, c.ClosePosition, c.AutoSize, c.TimeInForce, c.Text, c.Settle, isRest)
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

// CreateSelfTradePreventionUserGroup creates an STP user group. Only the main account can create groups
func (e *Exchange) CreateSelfTradePreventionUserGroup(ctx context.Context, name string) (*STPUserGroupResponse, error) {
	if name == "" {
		return nil, errSTPGroupNameRequired
	}
	var resp *STPUserGroupResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountCreateSTPGroupEPL, http.MethodPost, "account/stp_groups", nil, map[string]string{"name": name}, &resp)
}

// GetUserSelfTradePreventionGroups retrieves the STP user groups created by the main account, fuzzy matched by name
// when it is set
func (e *Exchange) GetUserSelfTradePreventionGroups(ctx context.Context, name string) ([]*STPUserGroupResponse, error) {
	params := url.Values{}
	if name != "" {
		params.Set("name", name)
	}
	var resp []*STPUserGroupResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountListSTPGroupsEPL, http.MethodGet, "account/stp_groups", params, nil, &resp)
}

// GetUsersInSTPUserGroup retrieves the users in an STP user group created by the main account
func (e *Exchange) GetUsersInSTPUserGroup(ctx context.Context, stpID uint64) ([]*STPUserGroupMemberResponse, error) {
	if stpID == 0 {
		return nil, errSTPGroupIDRequired
	}
	var resp []*STPUserGroupMemberResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountListSTPGroupUsersEPL, http.MethodGet, "account/stp_groups/"+strconv.FormatUint(stpID, 10)+"/users", nil, nil, &resp)
}

// AddUsersToSTPUserGroup adds accounts under the main account to an STP user group and returns the group's users
func (e *Exchange) AddUsersToSTPUserGroup(ctx context.Context, stpID uint64, userIDs []uint64) ([]*STPUserGroupMemberResponse, error) {
	if stpID == 0 {
		return nil, errSTPGroupIDRequired
	}
	if len(userIDs) == 0 || slices.Contains(userIDs, 0) {
		return nil, errUserIDRequired
	}
	var resp []*STPUserGroupMemberResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountAddSTPGroupUsersEPL, http.MethodPost, "account/stp_groups/"+strconv.FormatUint(stpID, 10)+"/users", nil, userIDs, &resp)
}

// DeleteUserFromSTPUserGroup removes accounts under the main account from an STP user group and returns the group's
// remaining users
func (e *Exchange) DeleteUserFromSTPUserGroup(ctx context.Context, stpID uint64, userIDs []uint64) ([]*STPUserGroupMemberResponse, error) {
	if stpID == 0 {
		return nil, errSTPGroupIDRequired
	}
	if len(userIDs) == 0 || slices.Contains(userIDs, 0) {
		return nil, errUserIDRequired
	}
	ids := make([]string, len(userIDs))
	for i := range userIDs {
		ids[i] = strconv.FormatUint(userIDs[i], 10)
	}
	params := url.Values{}
	params.Set("user_id", strings.Join(ids, ","))
	var resp []*STPUserGroupMemberResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountDeleteSTPGroupUsersEPL, http.MethodDelete, "account/stp_groups/"+strconv.FormatUint(stpID, 10)+"/users", params, nil, &resp)
}

// ConfigureGTFeeDeduction enables or disables GT fee deduction for the current account
func (e *Exchange) ConfigureGTFeeDeduction(ctx context.Context, setEnabled bool) error {
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountSetDebitFeeEPL, http.MethodPost, "account/debit_fee", nil, &map[string]bool{"enabled": setEnabled}, nil)
}

// GetGTFeeDeductionConfiguration reports whether GT fee deduction is enabled for the current account
func (e *Exchange) GetGTFeeDeductionConfiguration(ctx context.Context) (bool, error) {
	var resp gtFeeDeductionReply
	return resp.Enabled, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, accountGetDebitFeeEPL, http.MethodGet, "account/debit_fee", nil, nil, &resp)
}

// PlaceMultiCollateralLoanOrder places a multi-currency collateral loan order
func (e *Exchange) PlaceMultiCollateralLoanOrder(ctx context.Context, arg *MultiCollateralLoanOrderRequest) (*OrderIDResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.BorrowCurrency.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.BorrowAmount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if arg.OrderType == "fixed" {
		if arg.FixedType == "" {
			return nil, errFixedRateTermRequired
		}
		if arg.FixedRate <= 0 {
			return nil, errFixedRateRequired
		}
	}
	var resp *OrderIDResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralCreateOrderEPL, http.MethodPost, "loan/multi_collateral/orders", nil, arg, &resp)
}

// GetMultiCollateralLoanOrders retrieves multi-currency collateral loan orders
// sortType is time_desc, ltv_asc or ltv_desc and orderType is current or fixed
func (e *Exchange) GetMultiCollateralLoanOrders(ctx context.Context, page, limit uint64, sortType, orderType string) ([]*MultiCollateralLoanOrderDetailResponse, error) {
	params := url.Values{}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if sortType != "" {
		params.Set("sort", sortType)
	}
	if orderType != "" {
		params.Set("order_type", orderType)
	}
	var resp []*MultiCollateralLoanOrderDetailResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralOrdersEPL, http.MethodGet, "loan/multi_collateral/orders", params, nil, &resp)
}

// GetMultiCollateralLoanOrderDetail retrieves a multi-currency collateral loan order
func (e *Exchange) GetMultiCollateralLoanOrderDetail(ctx context.Context, orderID string) (*MultiCollateralLoanOrderDetailResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *MultiCollateralLoanOrderDetailResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralOrderDetailEPL, http.MethodGet, "loan/multi_collateral/orders/"+orderID, nil, nil, &resp)
}

// RepayMultiCollateralLoan repays a multi-currency collateral loan
func (e *Exchange) RepayMultiCollateralLoan(ctx context.Context, arg *MultiCollateralLoanRepaymentRequest) (*MultiCollateralLoanRepaymentResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.OrderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if len(arg.RepayItems) == 0 {
		return nil, fmt.Errorf("%w: at least one repay item is required", errNoValidParameterPassed)
	}
	for _, item := range arg.RepayItems {
		if item.Currency.IsEmpty() {
			return nil, currency.ErrCurrencyCodeEmpty
		}
		if !item.RepaidAll && item.Amount <= 0 {
			return nil, fmt.Errorf("%w: partial repayments require an amount", order.ErrAmountIsInvalid)
		}
	}
	var resp *MultiCollateralLoanRepaymentResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralRepayEPL, http.MethodPost, "loan/multi_collateral/repay", nil, arg, &resp)
}

// GetMultiCurrencyCollateralRepaymentRecords retrieves multi-currency collateral repayment records
func (e *Exchange) GetMultiCurrencyCollateralRepaymentRecords(ctx context.Context, arg *MultiCollateralRepaymentRecordsRequest) ([]*MultiCurrencyCollateralRepaymentResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Type == "" {
		return nil, errLoanTypeIsRequired
	}
	params := url.Values{}
	if err := setUnixTimeRangeParams(&params, arg.From, arg.To); err != nil {
		return nil, err
	}
	params.Set("type", arg.Type)
	if !arg.BorrowCurrency.IsEmpty() {
		params.Set("borrow_currency", arg.BorrowCurrency.String())
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []*MultiCurrencyCollateralRepaymentResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralRepayRecordsEPL, http.MethodGet, "loan/multi_collateral/repay", params, nil, &resp)
}

// AddOrWithdrawCollateral adds collateral to or withdraws collateral from a multi-currency collateral loan
func (e *Exchange) AddOrWithdrawCollateral(ctx context.Context, arg *AddOrWithdrawCollateralRequest) (*CollateralAddOrRemoveResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.OrderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.OperationType == "" {
		return nil, errLoanTypeIsRequired
	}
	var resp *CollateralAddOrRemoveResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralOperateEPL, http.MethodPost, "loan/multi_collateral/mortgage", nil, arg, &resp)
}

// GetMultiCollateralAdjustmentRecords retrieves multi-currency collateral adjustment records
func (e *Exchange) GetMultiCollateralAdjustmentRecords(ctx context.Context, arg *MultiCollateralAdjustmentRecordsRequest) ([]*MultiCollateralAdjustmentRecordResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setUnixTimeRangeParams(&params, arg.From, arg.To); err != nil {
		return nil, err
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if !arg.CollateralCurrency.IsEmpty() {
		params.Set("collateral_currency", arg.CollateralCurrency.String())
	}
	var resp []*MultiCollateralAdjustmentRecordResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralMortgageEPL, http.MethodGet, "loan/multi_collateral/mortgage", params, nil, &resp)
}

// GetMultiCollateralCurrencyQuota retrieves the user's collateral or borrowing quota per currency
// quotaType is collateral or borrow; borrow quotas accept a single currency
func (e *Exchange) GetMultiCollateralCurrencyQuota(ctx context.Context, quotaType string, currencies []currency.Code) ([]*MultiCollateralCurrencyQuotaResponse, error) {
	if quotaType == "" {
		return nil, errCurrencyTypeRequired
	}
	if len(currencies) == 0 {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if quotaType == "borrow" && len(currencies) > 1 {
		return nil, fmt.Errorf("%w: borrow quotas accept a single currency", errTooManyCurrencyCodes)
	}
	params := url.Values{}
	params.Set("type", quotaType)
	params.Set("currency", currency.Currencies(currencies).Join())
	var resp []*MultiCollateralCurrencyQuotaResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralCurrencyQuotaEPL, http.MethodGet, "loan/multi_collateral/currency_quota", params, nil, &resp)
}

// GetMultiCollateralSupportedCurrencies retrieves the borrowing and collateral currencies supported by multi-currency collateral loans
func (e *Exchange) GetMultiCollateralSupportedCurrencies(ctx context.Context) (*MultiCollateralSupportedCurrenciesResponse, error) {
	var resp *MultiCollateralSupportedCurrenciesResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralCurrenciesEPL, "loan/multi_collateral/currencies", &resp)
}

// GetMultiCollateralisationRatio retrieves the multi-currency collateral ratios, which are the same for every currency
func (e *Exchange) GetMultiCollateralisationRatio(ctx context.Context) (*MultiCollateralisationRatioResponse, error) {
	var resp *MultiCollateralisationRatioResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralLtvEPL, "loan/multi_collateral/ltv", &resp)
}

// GetMultiCollateralFixedRates retrieves each currency's 7-day and 30-day fixed interest rates
func (e *Exchange) GetMultiCollateralFixedRates(ctx context.Context) ([]*CollateralFixRateResponse, error) {
	var resp []*CollateralFixRateResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralFixedRateEPL, "loan/multi_collateral/fixed_rate", &resp)
}

// GetMultiCollateralCurrentRates retrieves the previous hour's interest rate for up to 100 currencies
// vipLevel defaults to 0 when empty
func (e *Exchange) GetMultiCollateralCurrentRates(ctx context.Context, currencies []currency.Code, vipLevel string) ([]*CollateralCurrentRateResponse, error) {
	if len(currencies) == 0 {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if len(currencies) > 100 {
		return nil, fmt.Errorf("%w: maximum 100", errTooManyCurrencyCodes)
	}
	params := url.Values{}
	params.Set("currencies", currency.Currencies(currencies).Join())
	if vipLevel != "" {
		params.Set("vip_level", vipLevel)
	}
	var resp []*CollateralCurrentRateResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, loanMultiCollateralCurrentRateEPL, common.EncodeURLValues("loan/multi_collateral/current_rate", params), &resp)
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

// ----- Earning endpoints -----------------

// GetLendingCurrencyList retrieves the currencies available for lending
func (e *Exchange) GetLendingCurrencyList(ctx context.Context) ([]*LendingCurrencyDetailResponse, error) {
	var resp []*LendingCurrencyDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, earnUniCurrenciesEPL, "earn/uni/currencies", &resp)
}

// GetLendingCurrencyDetail retrieves a single lending currency detail
func (e *Exchange) GetLendingCurrencyDetail(ctx context.Context, ccy currency.Code) (*LendingCurrencyDetailResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *LendingCurrencyDetailResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, earnUniCurrencyEPL, "earn/uni/currencies/"+ccy.Lower().String(), &resp)
}

// CreateLendingOrRedemption lends to or redeems from the lending pool; arg.Type is "lend" or "redeem"
func (e *Exchange) CreateLendingOrRedemption(ctx context.Context, arg *LendingOrRedemptionRequest) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.Currency.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return fmt.Errorf("%w: minimum lending or redemption amount is required", limits.ErrAmountBelowMin)
	}
	if arg.Type == "" {
		return errLoanTypeIsRequired
	}
	// Only lending earns interest, so redemptions carry no minimum rate
	if arg.Type == "lend" && arg.MinRate <= 0 {
		return fmt.Errorf("%w: minimum interest rate is required", limits.ErrAmountBelowMin)
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnUniCreateLendsEPL, http.MethodPost, "earn/uni/lends", nil, arg, nil)
}

// GetUserLendingOrderList retrieves the user's lending orders
func (e *Exchange) GetUserLendingOrderList(ctx context.Context, ccy currency.Code, page, limit uint64) ([]*LendOrderDetailResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if page > 0 {
		params.Set("page", strconv.FormatUint(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []*LendOrderDetailResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnUniGetLendsEPL, http.MethodGet, "earn/uni/lends", params, nil, &resp)
}

// AmendUserLendingInformation amends the hourly minimum interest rate of a currency's lending
func (e *Exchange) AmendUserLendingInformation(ctx context.Context, ccy currency.Code, minRate float64) error {
	if ccy.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if minRate <= 0 {
		return fmt.Errorf("%w: minimum interest rate is required", limits.ErrAmountBelowMin)
	}
	return e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnUniUpdateLendsEPL, http.MethodPatch, "earn/uni/lends", nil, map[string]string{
		"currency": ccy.String(),
		"min_rate": strconv.FormatFloat(minRate, 'f', -1, 64),
	}, nil)
}

// GetLendingTransactionRecords retrieves lending and redemption records
func (e *Exchange) GetLendingTransactionRecords(ctx context.Context, arg *LendingTransactionRecordsRequest) ([]*LendingTransactionRecordResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setUnixTimeRangeParams(&params, arg.From, arg.To); err != nil {
		return nil, err
	}
	if !arg.Currency.IsEmpty() {
		params.Set("currency", arg.Currency.String())
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if arg.Type != "" {
		params.Set("type", arg.Type)
	}
	var resp []*LendingTransactionRecordResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnUniLendRecordsEPL, http.MethodGet, "earn/uni/lend_records", params, nil, &resp)
}

// GetUserTotalInterestIncomePerCurrency retrieves the user's total interest income for a currency
func (e *Exchange) GetUserTotalInterestIncomePerCurrency(ctx context.Context, ccy currency.Code) (*CurrencyAndInterestIncomeResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *CurrencyAndInterestIncomeResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnUniInterestsEPL, http.MethodGet, "earn/uni/interests/"+ccy.Lower().String(), nil, nil, &resp)
}

// GetUserDividendRecords retrieves the user's lending interest records
func (e *Exchange) GetUserDividendRecords(ctx context.Context, arg *UserDividendRecordsRequest) ([]*UserDividendRecordsResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setUnixTimeRangeParams(&params, arg.From, arg.To); err != nil {
		return nil, err
	}
	if !arg.Currency.IsEmpty() {
		params.Set("currency", arg.Currency.String())
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []*UserDividendRecordsResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnUniInterestRecordsEPL, http.MethodGet, "earn/uni/interest_records", params, nil, &resp)
}

// GetCurrencyInterestCompoundingStatus retrieves whether a currency's lending interest is paid out or reinvested
func (e *Exchange) GetCurrencyInterestCompoundingStatus(ctx context.Context, ccy currency.Code) (*CurrencyInterestStatusResponse, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	var resp *CurrencyInterestStatusResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnUniInterestStatusEPL, http.MethodGet, "earn/uni/interest_status/"+ccy.Lower().String(), nil, nil, &resp)
}

// GetUniLoanCurrencyAnnualisedTrendChart retrieves a currency's annualised lending rate trend over a range of at most 30 days
func (e *Exchange) GetUniLoanCurrencyAnnualisedTrendChart(ctx context.Context, from, to time.Time, ccy currency.Code) ([]*UniLoanAssetResponse, error) {
	if err := common.StartEndTimeCheck(from, to); err != nil {
		return nil, err
	}
	if to.Sub(from) > 30*24*time.Hour {
		return nil, errUniChartRangeExceeded
	}
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("from", strconv.FormatInt(from.Unix(), 10))
	params.Set("to", strconv.FormatInt(to.Unix(), 10))
	params.Set("asset", ccy.String())
	var resp []*UniLoanAssetResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnUniChartEPL, http.MethodGet, "earn/uni/chart", params, nil, &resp)
}

// GetCurrencyEstimatedAnnualisedInterestRate retrieves each currency's estimated annualised lending rate
func (e *Exchange) GetCurrencyEstimatedAnnualisedInterestRate(ctx context.Context) ([]*CurrencyEstimatedAnnualInterestRateResponse, error) {
	var resp []*CurrencyEstimatedAnnualInterestRateResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, earnUniRateEPL, http.MethodGet, "earn/uni/rate", nil, nil, &resp)
}
