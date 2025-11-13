package bybit

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with Bybit
type Exchange struct {
	exchange.Base

	account accountTypeHolder
}

const (
	bybitAPIURL     = "https://api.bybit.com"
	bybitAPIVersion = "/v5/"
	tradeBaseURL    = "https://www.bybit.com/"

	defaultRecvWindow = "5000" // 5000 milli second

	sideBuy  = "Buy"
	sideSell = "Sell"

	cSpot, cLinear, cOption, cInverse = "spot", "linear", "option", "inverse"

	accountTypeNormal  AccountType = 1
	accountTypeUnified AccountType = 2

	longDatedFormat = "02Jan06"
)

var (
	errCategoryNotSet                     = errors.New("category not set")
	errBaseNotSet                         = errors.New("base coin not set when category is option")
	errInvalidTriggerDirection            = errors.New("invalid trigger direction")
	errInvalidTriggerPriceType            = errors.New("invalid trigger price type")
	errNilArgument                        = errors.New("nil argument")
	errMissingUserID                      = errors.New("sub user id missing")
	errMissingUsername                    = errors.New("username is missing")
	errInvalidMemberType                  = errors.New("invalid member type")
	errMissingTransferID                  = errors.New("transfer ID is required")
	errMemberIDRequired                   = errors.New("member ID is required")
	errNonePointerArgument                = errors.New("argument must be pointer")
	errEitherOrderIDOROrderLinkIDRequired = errors.New("either orderId or orderLinkId required")
	errNoOrderPassed                      = errors.New("no order passed")
	errSymbolOrSettleCoinRequired         = errors.New("provide symbol or settleCoin at least one")
	errInvalidTradeModeValue              = errors.New("invalid trade mode value")
	errTakeProfitOrStopLossModeMissing    = errors.New("TP/SL mode missing")
	errMissingAccountType                 = errors.New("account type not specified")
	errMembersIDsNotSet                   = errors.New("members IDs not set")
	errMissingChainType                   = errors.New("missing chain type is empty")
	errMissingChainInformation            = errors.New("missing transfer chain")
	errMissingAddressInfo                 = errors.New("address is required")
	errMissingWithdrawalID                = errors.New("missing withdrawal id")
	errTimeWindowRequired                 = errors.New("time window is required")
	errFrozenPeriodRequired               = errors.New("frozen period required")
	errQuantityLimitRequired              = errors.New("quantity limit required")
	errInvalidLeverage                    = errors.New("leverage can't be zero or less then it")
	errInvalidPositionMode                = errors.New("position mode is invalid")
	errInvalidMode                        = errors.New("mode can't be empty or missing")
	errInvalidOrderFilter                 = errors.New("invalid order filter")
	errInvalidCategory                    = errors.New("invalid category")
	errEitherSymbolOrCoinRequired         = errors.New("either symbol or coin required")
	errOrderLinkIDMissing                 = errors.New("order link id missing")
	errSymbolMissing                      = errors.New("symbol missing")
	errInvalidAutoAddMarginValue          = errors.New("invalid add auto margin value")
	errDisconnectTimeWindowNotSet         = errors.New("disconnect time window not set")
	errAPIKeyIsNotUnified                 = errors.New("api key is not unified")
	errInvalidContractLength              = errors.New("contract length cannot be less than or equal to zero")
)

var (
	intervalMap         = map[kline.Interval]string{kline.OneMin: "1", kline.ThreeMin: "3", kline.FiveMin: "5", kline.FifteenMin: "15", kline.ThirtyMin: "30", kline.OneHour: "60", kline.TwoHour: "120", kline.FourHour: "240", kline.SixHour: "360", kline.SevenHour: "720", kline.OneDay: "D", kline.OneWeek: "W", kline.OneMonth: "M"}
	stringToIntervalMap = map[string]kline.Interval{"1": kline.OneMin, "3": kline.ThreeMin, "5": kline.FiveMin, "15": kline.FifteenMin, "30": kline.ThirtyMin, "60": kline.OneHour, "120": kline.TwoHour, "240": kline.FourHour, "360": kline.SixHour, "720": kline.SevenHour, "D": kline.OneDay, "W": kline.OneWeek, "M": kline.OneMonth}
	validCategory       = []string{cSpot, cLinear, cOption, cInverse}
)

func intervalToString(interval kline.Interval) (string, error) {
	inter, okay := intervalMap[interval]
	if okay {
		return inter, nil
	}
	return "", kline.ErrUnsupportedInterval
}

// stringToInterval returns a kline.Interval instance from string.
func stringToInterval(s string) (kline.Interval, error) {
	interval, okay := stringToIntervalMap[s]
	if okay {
		return interval, nil
	}
	return 0, kline.ErrInvalidInterval
}

// GetBybitServerTime retrieves bybit server time
func (e *Exchange) GetBybitServerTime(ctx context.Context) (*ServerTime, error) {
	var resp *ServerTime
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "market/time", defaultEPL, &resp)
}

// GetKlines query for historical klines (also known as candles/candlesticks). Charts are returned in groups based on the requested interval.
func (e *Exchange) GetKlines(ctx context.Context, category, symbol string, interval kline.Interval, startTime, endTime time.Time, limit uint64) ([]KlineItem, error) {
	switch category {
	case "":
		return nil, errCategoryNotSet
	case cSpot, cLinear, cInverse:
	default:
		return nil, fmt.Errorf("%w, category: %s", errInvalidCategory, category)
	}
	if symbol == "" {
		return nil, errSymbolMissing
	}
	params := url.Values{}
	params.Set("category", category)
	params.Set("symbol", symbol)
	intervalString, err := intervalToString(interval)
	if err != nil {
		return nil, err
	}
	params.Set("interval", intervalString)
	if !startTime.IsZero() {
		params.Set("start", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("end", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp KlineResponse
	err = e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("market/kline", params), defaultEPL, &resp)
	if err != nil {
		return nil, err
	}
	return resp.List, nil
}

// GetInstrumentInfo retrieves the list of instrument details given the category and symbol.
func (e *Exchange) GetInstrumentInfo(ctx context.Context, category, symbol, status, baseCoin, cursor string, limit int64) (*InstrumentsInfo, error) {
	params, err := fillCategoryAndSymbol(category, symbol, true)
	if err != nil {
		return nil, err
	}
	if status != "" {
		params.Set("status", status)
	}
	if baseCoin != "" {
		params.Set("baseCoin", baseCoin)
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *InstrumentsInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("market/instruments-info", params), defaultEPL, &resp)
}

// GetMarkPriceKline query for historical mark price klines. Charts are returned in groups based on the requested interval.
func (e *Exchange) GetMarkPriceKline(ctx context.Context, category, symbol string, interval kline.Interval, startTime, endTime time.Time, limit int64) ([]KlineItem, error) {
	params, err := fillCategoryAndSymbol(category, symbol)
	if err != nil {
		return nil, err
	}
	intervalString, err := intervalToString(interval)
	if err != nil {
		return nil, err
	}
	params.Set("interval", intervalString)
	if !startTime.IsZero() {
		params.Set("start", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("end", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp MarkPriceKlineResponse
	err = e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("market/mark-price-kline", params), defaultEPL, &resp)
	if err != nil {
		return nil, err
	}
	return resp.List, nil
}

// GetIndexPriceKline query for historical index price klines. Charts are returned in groups based on the requested interval.
func (e *Exchange) GetIndexPriceKline(ctx context.Context, category, symbol string, interval kline.Interval, startTime, endTime time.Time, limit int64) ([]KlineItem, error) {
	params, err := fillCategoryAndSymbol(category, symbol)
	if err != nil {
		return nil, err
	}
	intervalString, err := intervalToString(interval)
	if err != nil {
		return nil, err
	}
	params.Set("interval", intervalString)
	if !startTime.IsZero() {
		params.Set("start", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("end", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp KlineResponse
	err = e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("market/index-price-kline", params), defaultEPL, &resp)
	if err != nil {
		return nil, err
	}
	return resp.List, nil
}

// GetOrderBook retrieves for orderbook depth data.
func (e *Exchange) GetOrderBook(ctx context.Context, category, symbol string, limit int64) (*Orderbook, error) {
	params, err := fillCategoryAndSymbol(category, symbol)
	if err != nil {
		return nil, err
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp orderbookResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("market/orderbook", params), defaultEPL, &resp); err != nil {
		return nil, err
	}
	return &Orderbook{
		UpdateID:       resp.UpdateID,
		Symbol:         resp.Symbol,
		GenerationTime: resp.Timestamp.Time(),
		Bids:           resp.Bids.Levels(),
		Asks:           resp.Asks.Levels(),
	}, nil
}

func fillCategoryAndSymbol(category, symbol string, optionalSymbol ...bool) (url.Values, error) {
	if category == "" {
		return nil, errCategoryNotSet
	} else if category != cSpot && category != cLinear && category != cInverse && category != cOption {
		return nil, fmt.Errorf("%w, category: %s", errInvalidCategory, category)
	}
	params := url.Values{}
	if symbol == "" && (len(optionalSymbol) == 0 || !optionalSymbol[0]) {
		return nil, errSymbolMissing
	} else if symbol != "" {
		params.Set("symbol", symbol)
	}
	params.Set("category", category)
	return params, nil
}

// GetTickers returns the latest price snapshot, best bid/ask price, and trading volume in the last 24 hours.
func (e *Exchange) GetTickers(ctx context.Context, category, symbol, baseCoin string, expiryDate time.Time) (*TickerData, error) {
	params, err := fillCategoryAndSymbol(category, symbol, true)
	if err != nil {
		return nil, err
	}
	if category == cOption && symbol == "" && baseCoin == "" {
		return nil, errBaseNotSet
	}
	if baseCoin != "" {
		params.Set("baseCoin", baseCoin)
	}
	if !expiryDate.IsZero() {
		params.Set("expData", expiryDate.Format(longDatedFormat))
	}
	var resp *TickerData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("market/tickers", params), defaultEPL, &resp)
}

// GetFundingRateHistory retrieves historical funding rates. Each symbol has a different funding interval.
// For example, if the interval is 8 hours and the current time is UTC 12, then it returns the last funding rate, which settled at UTC 8.
func (e *Exchange) GetFundingRateHistory(ctx context.Context, category, symbol string, startTime, endTime time.Time, limit int64) (*FundingRateHistory, error) {
	if category == "" {
		return nil, errCategoryNotSet
	} else if category != cLinear && category != cInverse {
		return nil, fmt.Errorf("%w, category: %s", errInvalidCategory, category)
	}
	params := url.Values{}
	if symbol == "" {
		return nil, errSymbolMissing
	}
	params.Set("symbol", symbol)
	params.Set("category", category)
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *FundingRateHistory
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("market/funding/history", params), defaultEPL, &resp)
}

// GetPublicTradingHistory retrieves recent public trading data.
// Option type. 'Call' or 'Put'. For option only
func (e *Exchange) GetPublicTradingHistory(ctx context.Context, category, symbol, baseCoin, optionType string, limit int64) (*TradingHistory, error) {
	params, err := fillCategoryAndSymbol(category, symbol)
	if err != nil {
		return nil, err
	}
	if category == cOption && symbol == "" && baseCoin == "" {
		return nil, errBaseNotSet
	}
	if baseCoin != "" {
		params.Set("baseCoin", baseCoin)
	}
	if optionType != "" {
		params.Set("optionType", optionType)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *TradingHistory
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("market/recent-trade", params), defaultEPL, &resp)
}

// GetOpenInterestData retrieves open interest of each symbol.
func (e *Exchange) GetOpenInterestData(ctx context.Context, category, symbol, intervalTime string, startTime, endTime time.Time, limit int64, cursor string) (*OpenInterest, error) {
	if category == "" {
		return nil, errCategoryNotSet
	} else if category != cLinear && category != cInverse {
		return nil, fmt.Errorf("%w, category: %s", errInvalidCategory, category)
	}
	params := url.Values{}
	if symbol == "" {
		return nil, errSymbolMissing
	}
	params.Set("symbol", symbol)
	params.Set("category", category)
	if intervalTime != "" {
		params.Set("intervalTime", intervalTime)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	var resp *OpenInterest
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("market/open-interest", params), defaultEPL, &resp)
}

// GetHistoricalVolatility retrieves option historical volatility.
// The data is hourly.
// If both 'startTime' and 'endTime' are not specified, it will return the most recent 1 hours worth of data.
// 'startTime' and 'endTime' are a pair of params. Either both are passed or they are not passed at all.
// This endpoint can query the last 2 years worth of data, but make sure [endTime - startTime] <= 30 days.
func (e *Exchange) GetHistoricalVolatility(ctx context.Context, category, baseCoin string, period int64, startTime, endTime time.Time) ([]HistoricVolatility, error) {
	if category == "" {
		return nil, errCategoryNotSet
	} else if category != cOption {
		return nil, fmt.Errorf("%w, category: %s", errInvalidCategory, category)
	}
	params := url.Values{}
	params.Set("category", category)
	if baseCoin != "" {
		params.Set("baseCoin", baseCoin)
	}
	if period > 0 {
		params.Set("period", strconv.FormatInt(period, 10))
	}
	var err error
	if !startTime.IsZero() || !endTime.IsZero() {
		err = common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []HistoricVolatility
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("market/historical-volatility", params), defaultEPL, &resp)
}

// GetInsurance retrieves insurance pool data (BTC/USDT/USDC etc). The data is updated every 24 hours.
func (e *Exchange) GetInsurance(ctx context.Context, coin string) (*InsuranceHistory, error) {
	params := url.Values{}
	if coin != "" {
		params.Set("coin", coin)
	}
	var resp *InsuranceHistory
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("market/insurance", params), defaultEPL, &resp)
}

// GetRiskLimit retrieves risk limit history
func (e *Exchange) GetRiskLimit(ctx context.Context, category, symbol string) (*RiskLimitHistory, error) {
	if category == "" {
		return nil, errCategoryNotSet
	} else if category != cLinear && category != cInverse {
		return nil, fmt.Errorf("%w, category: %s", errInvalidCategory, category)
	}
	params := url.Values{}
	params.Set("category", category)
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *RiskLimitHistory
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("market/risk-limit", params), defaultEPL, &resp)
}

// GetDeliveryPrice retrieves delivery price.
func (e *Exchange) GetDeliveryPrice(ctx context.Context, category, symbol, baseCoin, cursor string, limit int64) (*DeliveryPrice, error) {
	if category == "" {
		return nil, errCategoryNotSet
	} else if category != cLinear && category != cInverse && category != cOption {
		return nil, fmt.Errorf("%w, category: %s", errInvalidCategory, category)
	}
	params := url.Values{}
	params.Set("category", category)
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if baseCoin != "" {
		params.Set("baseCoin", baseCoin)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	var resp *DeliveryPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("market/delivery-price", params), defaultEPL, &resp)
}

func isValidCategory(category string) error {
	switch category {
	case cSpot, cOption, cLinear, cInverse:
		return nil
	case "":
		return errCategoryNotSet
	default:
		return fmt.Errorf("%w, category: %s", errInvalidCategory, category)
	}
}

// PlaceOrder creates an order for spot, spot margin, USDT perpetual, USDC perpetual, USDC futures, inverse futures and options.
func (e *Exchange) PlaceOrder(ctx context.Context, arg *PlaceOrderRequest) (*OrderResponse, error) {
	if err := arg.Validate(); err != nil {
		return nil, err
	}
	epl := createOrderEPL
	if arg.Category == cSpot {
		epl = createSpotOrderEPL
	}
	var resp *OrderResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/order/create", nil, arg, &resp, epl)
}

// AmendOrder amends an open unfilled or partially filled orders.
func (e *Exchange) AmendOrder(ctx context.Context, arg *AmendOrderRequest) (*OrderResponse, error) {
	if err := arg.Validate(); err != nil {
		return nil, err
	}
	var resp *OrderResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/order/amend", nil, arg, &resp, amendOrderEPL)
}

// CancelTradeOrder cancels an open unfilled or partially filled order.
func (e *Exchange) CancelTradeOrder(ctx context.Context, arg *CancelOrderRequest) (*OrderResponse, error) {
	if err := arg.Validate(); err != nil {
		return nil, err
	}
	epl := cancelOrderEPL
	if arg.Category == cSpot {
		epl = cancelSpotEPL
	}
	var resp *OrderResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/order/cancel", nil, arg, &resp, epl)
}

// GetOpenOrders retrieves unfilled or partially filled orders in real-time. To query older order records, please use the order history interface.
// orderFilter: possible values are 'Order', 'StopOrder', 'tpslOrder', and 'OcoOrder'
func (e *Exchange) GetOpenOrders(ctx context.Context, category, symbol, baseCoin, settleCoin, orderID, orderLinkID, orderFilter, cursor string, openOnly, limit int64) (*TradeOrders, error) {
	params, err := fillCategoryAndSymbol(category, symbol, true)
	if err != nil {
		return nil, err
	}
	if baseCoin != "" {
		params.Set("baseCoin", baseCoin)
	}
	if settleCoin != "" {
		params.Set("settleCoin", settleCoin)
	}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if orderLinkID != "" {
		params.Set("orderLinkId", orderLinkID)
	}
	if openOnly != 0 {
		params.Set("openOnly", strconv.FormatInt(openOnly, 10))
	}
	if orderFilter != "" {
		params.Set("orderFilter", orderFilter)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	var resp *TradeOrders
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/order/realtime", params, nil, &resp, getOrderEPL)
}

// CancelAllTradeOrders cancel all open orders
func (e *Exchange) CancelAllTradeOrders(ctx context.Context, arg *CancelAllOrdersParam) ([]OrderResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	err := isValidCategory(arg.Category)
	if err != nil {
		return nil, err
	}
	if arg.OrderFilter != "" && (arg.Category != cLinear && arg.Category != cInverse) {
		return nil, fmt.Errorf("%w, only used for category=linear or inverse", errInvalidOrderFilter)
	}
	var resp CancelAllResponse
	epl := cancelAllEPL
	if arg.Category == cSpot {
		epl = cancelAllSpotEPL
	}
	return resp.List, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/order/cancel-all", nil, arg, &resp, epl)
}

// GetTradeOrderHistory retrieves order history. As order creation/cancellation is asynchronous, the data returned from this endpoint may delay.
// If you want to get real-time order information, you could query this endpoint or rely on the websocket stream (recommended).
// orderFilter: possible values are 'Order', 'StopOrder', 'tpslOrder', and 'OcoOrder'
func (e *Exchange) GetTradeOrderHistory(ctx context.Context, category, symbol, orderID, orderLinkID,
	baseCoin, settleCoin, orderFilter, orderStatus, cursor string,
	startTime, endTime time.Time, limit int64,
) (*TradeOrders, error) {
	params, err := fillCategoryAndSymbol(category, symbol, true)
	if err != nil {
		return nil, err
	}
	if baseCoin != "" {
		params.Set("baseCoin", baseCoin)
	}
	if settleCoin != "" {
		params.Set("settleCoin", settleCoin)
	}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if orderLinkID != "" {
		params.Set("orderLinkId", orderLinkID)
	}
	if orderFilter != "" {
		params.Set("orderFilter", orderFilter)
	}
	if orderStatus != "" {
		params.Set("orderStatus", orderStatus)
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *TradeOrders
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/order/history", params, nil, &resp, getOrderHistoryEPL)
}

// PlaceBatchOrder place batch or trade order.
func (e *Exchange) PlaceBatchOrder(ctx context.Context, arg *PlaceBatchOrderParam) ([]BatchOrderResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	switch {
	case arg.Category == "":
		return nil, errCategoryNotSet
	case arg.Category != cOption && arg.Category != cLinear:
		return nil, fmt.Errorf("%w, only 'option' and 'linear' categories are allowed", errInvalidCategory)
	}
	if len(arg.Request) == 0 {
		return nil, errNoOrderPassed
	}
	for a := range arg.Request {
		if arg.Request[a].OrderLinkID == "" {
			return nil, errOrderLinkIDMissing
		}
		if arg.Request[a].Symbol.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
		if arg.Request[a].Side == "" {
			return nil, order.ErrSideIsInvalid
		}
		if arg.Request[a].OrderType == "" { // Market and Limit order types are allowed
			return nil, order.ErrTypeIsInvalid
		}
		if arg.Request[a].OrderQuantity <= 0 {
			return nil, limits.ErrAmountBelowMin
		}
	}
	var resp BatchOrdersList
	return resp.List, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/order/create-batch", nil, arg, &resp, createBatchOrderEPL)
}

// BatchAmendOrder represents a batch amend order.
func (e *Exchange) BatchAmendOrder(ctx context.Context, category string, args []BatchAmendOrderParamItem) (*BatchOrderResponse, error) {
	if len(args) == 0 {
		return nil, errNilArgument
	}
	switch {
	case category == "":
		return nil, errCategoryNotSet
	case category != cOption:
		return nil, fmt.Errorf("%w, only 'option' category is allowed", errInvalidCategory)
	}
	for a := range args {
		if args[a].OrderID == "" && args[a].OrderLinkID == "" {
			return nil, errEitherOrderIDOROrderLinkIDRequired
		}
		if args[a].Symbol.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
	}
	var resp *BatchOrderResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/order/amend-batch", nil, &BatchAmendOrderParams{
		Category: category,
		Request:  args,
	}, &resp, amendBatchOrderEPL)
}

// CancelBatchOrder cancel more than one open order in a single request.
func (e *Exchange) CancelBatchOrder(ctx context.Context, arg *CancelBatchOrder) ([]CancelBatchResponseItem, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.Category != cOption {
		return nil, fmt.Errorf("%w, only 'option' category is allowed", errInvalidCategory)
	}
	if len(arg.Request) == 0 {
		return nil, errNoOrderPassed
	}
	for a := range arg.Request {
		if arg.Request[a].OrderID == "" && arg.Request[a].OrderLinkID == "" {
			return nil, errEitherOrderIDOROrderLinkIDRequired
		}
		if arg.Request[a].Symbol.IsEmpty() {
			return nil, currency.ErrCurrencyPairEmpty
		}
	}
	var resp cancelBatchResponse
	return resp.List, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/order/cancel-batch", nil, arg, &resp, cancelBatchOrderEPL)
}

// GetBorrowQuota retrieves the qty and amount of borrowable coins in spot account.
func (e *Exchange) GetBorrowQuota(ctx context.Context, category, symbol, side string) (*BorrowQuota, error) {
	if category == "" {
		return nil, errCategoryNotSet
	} else if category != cSpot {
		return nil, fmt.Errorf("%w, category: %s", errInvalidCategory, category)
	}
	params := url.Values{}
	if symbol == "" {
		return nil, errSymbolMissing
	}
	params.Set("symbol", symbol)
	params.Set("category", category)
	if side == "" {
		return nil, order.ErrSideIsInvalid
	}
	params.Set("side", side)
	var resp *BorrowQuota
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/order/spot-borrow-check", params, nil, &resp, defaultEPL)
}

// SetDisconnectCancelAll You can use this endpoint to get your current DCP configuration.
// Your private websocket connection must subscribe "dcp" topic in order to trigger DCP successfully
func (e *Exchange) SetDisconnectCancelAll(ctx context.Context, arg *SetDCPParams) error {
	if arg == nil {
		return errNilArgument
	}
	if arg.TimeWindow == 0 {
		return errDisconnectTimeWindowNotSet
	}
	return e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/order/disconnected-cancel-all", nil, arg, &struct{}{}, defaultEPL)
}

// GetPositionInfo retrieves real-time position data, such as position size, cumulative realizedPNL.
func (e *Exchange) GetPositionInfo(ctx context.Context, category, symbol, baseCoin, settleCoin, cursor string, limit int64) (*PositionInfoList, error) {
	if category == "" {
		return nil, errCategoryNotSet
	} else if category != cLinear && category != cInverse && category != cOption {
		return nil, fmt.Errorf("%w, category: %s", errInvalidCategory, category)
	}
	if symbol == "" && settleCoin == "" {
		return nil, errSymbolOrSettleCoinRequired
	}
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	params.Set("category", category)
	if baseCoin != "" {
		params.Set("baseCoin", baseCoin)
	}
	if settleCoin != "" {
		params.Set("settleCoin", settleCoin)
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *PositionInfoList
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/position/list", params, nil, &resp, getPositionListEPL)
}

// SetLeverageLevel sets a leverage from 0 to max leverage of corresponding risk limit
func (e *Exchange) SetLeverageLevel(ctx context.Context, arg *SetLeverageParams) error {
	if arg == nil {
		return errNilArgument
	}
	switch arg.Category {
	case "":
		return errCategoryNotSet
	case cLinear, cInverse:
	default:
		return fmt.Errorf("%w, category: %s", errInvalidCategory, arg.Category)
	}
	if arg.Symbol == "" {
		return errSymbolMissing
	}
	switch {
	case arg.BuyLeverage <= 0:
		return fmt.Errorf("%w code: 10001 msg: invalid buy leverage %f", errInvalidLeverage, arg.BuyLeverage)
	case arg.BuyLeverage != arg.SellLeverage:
		return fmt.Errorf("%w, buy leverage not equal sell leverage", errInvalidLeverage)
	}
	return e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/position/set-leverage", nil, arg, &struct{}{}, postPositionSetLeverageEPL)
}

// SwitchTradeMode sets the trade mode value either to 'cross' or 'isolated'.
// Select cross margin mode or isolated margin mode per symbol level
func (e *Exchange) SwitchTradeMode(ctx context.Context, arg *SwitchTradeModeParams) error {
	if arg == nil {
		return errNilArgument
	}
	switch arg.Category {
	case "":
		return errCategoryNotSet
	case cLinear, cInverse:
	default:
		return fmt.Errorf("%w, category: %s", errInvalidCategory, arg.Category)
	}
	switch {
	case arg.Symbol == "":
		return errSymbolMissing
	case arg.BuyLeverage <= 0:
		return fmt.Errorf("%w code: 10001 msg: invalid buy leverage %f", errInvalidLeverage, arg.BuyLeverage)
	case arg.BuyLeverage != arg.SellLeverage:
		return fmt.Errorf("%w, buy leverage not equal sell leverage", errInvalidLeverage)
	case arg.TradeMode != 0 && arg.TradeMode != 1:
		return errInvalidTradeModeValue
	}
	return e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/position/switch-isolated", nil, arg, &struct{}{}, defaultEPL)
}

// SetTakeProfitStopLossMode set partial TP/SL mode, you can set the TP/SL size smaller than position size.
func (e *Exchange) SetTakeProfitStopLossMode(ctx context.Context, arg *TPSLModeParams) (*TPSLModeResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	switch arg.Category {
	case "":
		return nil, errCategoryNotSet
	case cLinear, cInverse:
	default:
		return nil, fmt.Errorf("%w, category: %s", errInvalidCategory, arg.Category)
	}
	switch {
	case arg.Symbol == "":
		return nil, errSymbolMissing
	case arg.TpslMode == "":
		return nil, errTakeProfitOrStopLossModeMissing
	}
	var resp *TPSLModeResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/position/set-tpsl-mode", nil, arg, &resp, setPositionTPLSModeEPL)
}

// SwitchPositionMode switch the position mode for USDT perpetual and Inverse futures.
// If you are in one-way Mode, you can only open one position on Buy or Sell side.
// If you are in hedge mode, you can open both Buy and Sell side positions simultaneously.
// switches mode between MergedSingle: One-Way Mode or BothSide: Hedge Mode
func (e *Exchange) SwitchPositionMode(ctx context.Context, arg *SwitchPositionModeParams) error {
	if arg == nil {
		return errNilArgument
	}
	switch arg.Category {
	case "":
		return errCategoryNotSet
	case cLinear, cInverse:
	default:
		return fmt.Errorf("%w, category: %s", errInvalidCategory, arg.Category)
	}
	if arg.Symbol.IsEmpty() && arg.Coin.IsEmpty() {
		return errEitherSymbolOrCoinRequired
	}
	return e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/position/switch-mode", nil, arg, &struct{}{}, defaultEPL)
}

// SetRiskLimit risk limit will limit the maximum position value you can hold under different margin requirements.
// If you want to hold a bigger position size, you need more margin. This interface can set the risk limit of a single position.
// If the order exceeds the current risk limit when placing an order, it will be rejected.
// '0': one-way mode '1': hedge-mode Buy side '2': hedge-mode Sell side
func (e *Exchange) SetRiskLimit(ctx context.Context, arg *SetRiskLimitParam) (*RiskLimitResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	switch arg.Category {
	case "":
		return nil, errCategoryNotSet
	case cLinear, cInverse:
	default:
		return nil, fmt.Errorf("%w, category: %s", errInvalidCategory, arg.Category)
	}
	if arg.PositionMode < 0 || arg.PositionMode > 2 {
		return nil, errInvalidPositionMode
	}
	if arg.Symbol.IsEmpty() {
		return nil, errSymbolMissing
	}
	var resp *RiskLimitResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/position/set-risk-limit", nil, arg, &resp, setPositionRiskLimitEPL)
}

// SetTradingStop set the take profit, stop loss or trailing stop for the position.
func (e *Exchange) SetTradingStop(ctx context.Context, arg *TradingStopParams) error {
	if arg == nil {
		return errNilArgument
	}
	switch arg.Category {
	case cLinear, cInverse:
	case "":
		return errCategoryNotSet
	default:
		return fmt.Errorf("%w, category: %s", errInvalidCategory, arg.Category)
	}
	if arg.Symbol.IsEmpty() {
		return errSymbolMissing
	}
	return e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/position/trading-stop", nil, arg, &struct{}{}, stopTradingPositionEPL)
}

// SetAutoAddMargin sets auto add margin
func (e *Exchange) SetAutoAddMargin(ctx context.Context, arg *AutoAddMarginParam) error {
	if arg == nil {
		return errNilArgument
	}
	switch arg.Category {
	case "":
		return errCategoryNotSet
	case cLinear, cInverse:
	default:
		return fmt.Errorf("%w, category: %s", errInvalidCategory, arg.Category)
	}
	if arg.Symbol.IsEmpty() {
		return errSymbolMissing
	}
	if arg.AutoAddmargin != 0 && arg.AutoAddmargin != 1 {
		return errInvalidAutoAddMarginValue
	}
	if arg.PositionIndex < 0 || arg.PositionIndex > 2 {
		return errInvalidPositionMode
	}
	return e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/position/set-auto-add-margin", nil, arg, &struct{}{}, defaultEPL)
}

// AddOrReduceMargin manually add or reduce margin for isolated margin position
func (e *Exchange) AddOrReduceMargin(ctx context.Context, arg *AddOrReduceMarginParam) (*AddOrReduceMargin, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	switch arg.Category {
	case "":
		return nil, errCategoryNotSet
	case cLinear, cInverse:
	default:
		return nil, fmt.Errorf("%w, category: %s", errInvalidCategory, arg.Category)
	}
	if arg.Symbol.IsEmpty() {
		return nil, errSymbolMissing
	}
	if arg.Margin != 10 && arg.Margin != -10 {
		return nil, errInvalidAutoAddMarginValue
	}
	if arg.PositionIndex < 0 || arg.PositionIndex > 2 {
		return nil, errInvalidPositionMode
	}
	var resp *AddOrReduceMargin
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/position/add-margin", nil, arg, &resp, defaultEPL)
}

// GetExecution retrieves users' execution records, sorted by execTime in descending order. However, for Normal spot, they are sorted by execId in descending order.
// Execution Type possible values: 'Trade', 'AdlTrade' Auto-Deleveraging, 'Funding' Funding fee, 'BustTrade' Liquidation, 'Delivery' USDC futures delivery, 'BlockTrade'
// UTA Spot: 'stopOrderType', "" for normal order, "tpslOrder" for TP/SL order, "Stop" for conditional order, "OcoOrder" for OCO order
func (e *Exchange) GetExecution(ctx context.Context, category, symbol, orderID, orderLinkID, baseCoin, executionType, stopOrderType, cursor string, startTime, endTime time.Time, limit int64) (*ExecutionResponse, error) {
	params, err := fillCategoryAndSymbol(category, symbol, true)
	if err != nil {
		return nil, err
	}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if orderLinkID != "" {
		params.Set("orderLinkId", orderLinkID)
	}
	if baseCoin != "" {
		params.Set("baseCoin", baseCoin)
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if executionType != "" {
		params.Set("execType", executionType)
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if stopOrderType != "" {
		params.Set("stopOrderType", stopOrderType)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *ExecutionResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/execution/list", params, nil, &resp, getExecutionListEPL)
}

// GetClosedPnL retrieves user's closed profit and loss records. The results are sorted by createdTime in descending order.
func (e *Exchange) GetClosedPnL(ctx context.Context, category, symbol, cursor string, startTime, endTime time.Time, limit int64) (*ClosedProfitAndLossResponse, error) {
	params, err := fillOrderAndExecutionFetchParams(paramsConfig{Linear: true, Inverse: true}, category, symbol, "", "", "", "", "", cursor, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	var resp *ClosedProfitAndLossResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/position/closed-pnl", params, nil, &resp, getPositionClosedPNLEPL)
}

func fillOrderAndExecutionFetchParams(ac paramsConfig, category, symbol, baseCoin, orderID, orderLinkID, orderFilter, orderStatus, cursor string, startTime, endTime time.Time, limit int64) (url.Values, error) {
	params := url.Values{}
	switch {
	case !ac.OptionalCategory && category == "":
		return nil, errCategoryNotSet
	case ac.OptionalCategory && category == "":
	case (!ac.Linear && category == cLinear) ||
		(!ac.Option && category == cOption) ||
		(!ac.Inverse && category == cInverse) ||
		(!ac.Spot && category == cSpot):
		return nil, fmt.Errorf("%w, category: %s", errInvalidCategory, category)
	default:
		params.Set("category", category)
	}
	if ac.MandatorySymbol && symbol == "" {
		return nil, errSymbolMissing
	} else if symbol != "" {
		params.Set("symbol", symbol)
	}
	if category == cOption && baseCoin == "" && !ac.OptionalBaseCoin {
		return nil, fmt.Errorf("%w, baseCoin is required", errBaseNotSet)
	} else if baseCoin != "" {
		params.Set("baseCoin", baseCoin)
	}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if orderLinkID != "" {
		params.Set("orderLinkId", orderLinkID)
	}
	if orderFilter != "" {
		params.Set("orderFilter", orderFilter)
	}
	if orderStatus != "" {
		params.Set("orderStatus", orderStatus)
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return params, nil
}

// ConfirmNewRiskLimit t is only applicable when the user is marked as only reducing positions
// (please see the isReduceOnly field in the Get Position Info interface).
// After the user actively adjusts the risk level, this interface is called to try to calculate the adjusted risk level,
// and if it passes (retCode=0), the system will remove the position reduceOnly mark.
// You are recommended to call Get Position Info to check isReduceOnly field.
func (e *Exchange) ConfirmNewRiskLimit(ctx context.Context, category, symbol string) error {
	if category == "" {
		return errCategoryNotSet
	}
	if symbol == "" {
		return errSymbolMissing
	}
	arg := &struct {
		Category string `json:"category"`
		Symbol   string `json:"symbol"`
	}{
		Category: category,
		Symbol:   symbol,
	}
	return e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/position/confirm-pending-mmr", nil, arg, &struct{}{}, defaultEPL)
}

// GetPreUpgradeOrderHistory the account is upgraded to a Unified account, you can get the orders which occurred before the upgrade.
func (e *Exchange) GetPreUpgradeOrderHistory(ctx context.Context, category, symbol, baseCoin, orderID, orderLinkID, orderFilter, orderStatus, cursor string, startTime, endTime time.Time, limit int64) (*TradeOrders, error) {
	params, err := fillOrderAndExecutionFetchParams(paramsConfig{Linear: true, Option: true, Inverse: true}, category, symbol, baseCoin, orderID, orderLinkID, orderFilter, orderStatus, cursor, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	err = e.RequiresUnifiedAccount(ctx)
	if err != nil {
		return nil, err
	}
	var resp *TradeOrders
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/pre-upgrade/order/history", params, nil, &resp, defaultEPL)
}

// GetPreUpgradeTradeHistory retrieves users' execution records which occurred before you upgraded the account to a Unified account, sorted by execTime in descending order
func (e *Exchange) GetPreUpgradeTradeHistory(ctx context.Context, category, symbol, orderID, orderLinkID, baseCoin, executionType, cursor string, startTime, endTime time.Time, limit int64) (*ExecutionResponse, error) {
	params, err := fillOrderAndExecutionFetchParams(paramsConfig{Linear: true, Option: false, Inverse: true}, category, symbol, baseCoin, orderID, orderLinkID, "", "", cursor, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	err = e.RequiresUnifiedAccount(ctx)
	if err != nil {
		return nil, err
	}
	if executionType != "" {
		params.Set("executionType", executionType)
	}
	var resp *ExecutionResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/pre-upgrade/execution/list", params, nil, &resp, defaultEPL)
}

// GetPreUpgradeClosedPnL retrieves user's closed profit and loss records from before you upgraded the account to a Unified account. The results are sorted by createdTime in descending order.
func (e *Exchange) GetPreUpgradeClosedPnL(ctx context.Context, category, symbol, cursor string, startTime, endTime time.Time, limit int64) (*ClosedProfitAndLossResponse, error) {
	params, err := fillOrderAndExecutionFetchParams(paramsConfig{Linear: true, Inverse: true, MandatorySymbol: true}, category, symbol, "", "", "", "", "", cursor, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	err = e.RequiresUnifiedAccount(ctx)
	if err != nil {
		return nil, err
	}
	var resp *ClosedProfitAndLossResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/pre-upgrade/position/closed-pnl", params, nil, &resp, defaultEPL)
}

// GetPreUpgradeTransactionLog retrieves transaction logs which occurred in the USDC Derivatives wallet before the account was upgraded to a Unified account.
func (e *Exchange) GetPreUpgradeTransactionLog(ctx context.Context, category, baseCoin, transactionType, cursor string, startTime, endTime time.Time, limit int64) (*TransactionLog, error) {
	params, err := fillOrderAndExecutionFetchParams(paramsConfig{Linear: true, Inverse: true}, category, "", baseCoin, "", "", "", "", cursor, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	err = e.RequiresUnifiedAccount(ctx)
	if err != nil {
		return nil, err
	}
	if transactionType != "" {
		params.Set("type", transactionType)
	}
	var resp *TransactionLog
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/pre-upgrade/account/transaction-log", params, nil, &resp, defaultEPL)
}

// GetPreUpgradeOptionDeliveryRecord retrieves delivery records of Option before you upgraded the account to a Unified account, sorted by deliveryTime in descending order
func (e *Exchange) GetPreUpgradeOptionDeliveryRecord(ctx context.Context, category, symbol, cursor string, expiryDate time.Time, limit int64) (*PreUpdateOptionDeliveryRecord, error) {
	params, err := fillOrderAndExecutionFetchParams(paramsConfig{OptionalBaseCoin: true, Option: true}, category, symbol, "", "", "", "", "", cursor, time.Time{}, time.Time{}, limit)
	if err != nil {
		return nil, err
	}
	err = e.RequiresUnifiedAccount(ctx)
	if err != nil {
		return nil, err
	}
	if !expiryDate.IsZero() {
		params.Set("expData", expiryDate.Format(longDatedFormat))
	}
	var resp *PreUpdateOptionDeliveryRecord
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/pre-upgrade/asset/delivery-record", params, nil, &resp, defaultEPL)
}

// GetPreUpgradeUSDCSessionSettlement retrieves session settlement records of USDC perpetual before you upgrade the account to Unified account.
func (e *Exchange) GetPreUpgradeUSDCSessionSettlement(ctx context.Context, category, symbol, cursor string, limit int64) (*SettlementSession, error) {
	params, err := fillOrderAndExecutionFetchParams(paramsConfig{Linear: true}, category, symbol, "", "", "", "", "", cursor, time.Time{}, time.Time{}, limit)
	if err != nil {
		return nil, err
	}
	err = e.RequiresUnifiedAccount(ctx)
	if err != nil {
		return nil, err
	}
	var resp *SettlementSession
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/pre-upgrade/asset/settlement-record", params, nil, &resp, defaultEPL)
}

// GetWalletBalance represents wallet balance, query asset information of each currency, and account risk rate information.
// By default, currency information with assets or liabilities of 0 is not returned.
// Unified account: UNIFIED (trade spot/linear/options), CONTRACT(trade inverse)
// Normal account: CONTRACT, SPOT
func (e *Exchange) GetWalletBalance(ctx context.Context, accountType, coin string) (*WalletBalance, error) {
	params := url.Values{}
	if accountType == "" {
		return nil, errMissingAccountType
	}
	params.Set("accountType", accountType)
	if coin != "" {
		params.Set("coin", coin)
	}
	var resp *WalletBalance
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/wallet-balance", params, nil, &resp, getAccountWalletBalanceEPL)
}

// UpgradeToUnifiedAccount upgrades the account to unified account.
func (e *Exchange) UpgradeToUnifiedAccount(ctx context.Context) (*UnifiedAccountUpgradeResponse, error) {
	var resp *UnifiedAccountUpgradeResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/upgrade-to-uta", nil, nil, &resp, defaultEPL)
}

// GetBorrowHistory retrieves interest records, sorted in reverse order of creation time.
func (e *Exchange) GetBorrowHistory(ctx context.Context, ccy, cursor string, startTime, endTime time.Time, limit int64) (*BorrowHistory, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	var resp *BorrowHistory
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/borrow-history", params, nil, &resp, defaultEPL)
}

// SetCollateralCoin decide whether the assets in the Unified account needs to be collateral coins.
func (e *Exchange) SetCollateralCoin(ctx context.Context, coin currency.Code, collateralSwitchON bool) error {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	if collateralSwitchON {
		params.Set("collateralSwitch", "ON")
	} else {
		params.Set("collateralSwitch", "OFF")
	}
	return e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/set-collateral-switch", params, nil, &struct{}{}, defaultEPL)
}

// GetCollateralInfo retrieves the collateral information of the current unified margin account,
// including loan interest rate, loanable amount, collateral conversion rate,
// whether it can be mortgaged as margin, etc.
func (e *Exchange) GetCollateralInfo(ctx context.Context, ccy string) (*CollateralInfo, error) {
	params := url.Values{}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	var resp *CollateralInfo
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/collateral-info", params, nil, &resp, defaultEPL)
}

// GetCoinGreeks retrieves current account Greeks information
func (e *Exchange) GetCoinGreeks(ctx context.Context, baseCoin string) (*CoinGreeks, error) {
	params := url.Values{}
	if baseCoin != "" {
		params.Set("baseCoin", baseCoin)
	}
	var resp *CoinGreeks
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/coin-greeks", params, nil, &resp, defaultEPL)
}

// GetFeeRate retrieves the trading fee rate.
func (e *Exchange) GetFeeRate(ctx context.Context, category, symbol, baseCoin string) (*AccountFee, error) {
	params := url.Values{}
	if !slices.Contains(validCategory, category) {
		return nil, fmt.Errorf("%w, valid category values are %v", errInvalidCategory, validCategory)
	}
	if category != "" {
		params.Set("category", category)
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if baseCoin != "" {
		params.Set("baseCoin", baseCoin)
	}
	var resp *AccountFee
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/fee-rate", params, nil, &resp, getAccountFeeEPL)
}

// GetAccountInfo retrieves the margin mode configuration of the account.
// query the margin mode and the upgraded status of account
func (e *Exchange) GetAccountInfo(ctx context.Context) (*AccountInfo, error) {
	var resp *AccountInfo
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/info", nil, nil, &resp, defaultEPL)
}

// GetTransactionLog retrieves transaction logs in Unified account.
func (e *Exchange) GetTransactionLog(ctx context.Context, category, baseCoin, transactionType, cursor string, startTime, endTime time.Time, limit int64) (*TransactionLog, error) {
	params, err := fillOrderAndExecutionFetchParams(paramsConfig{OptionalBaseCoin: true, OptionalCategory: true, Linear: true, Option: true, Spot: true}, category, "", baseCoin, "", "", "", "", cursor, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	if transactionType != "" {
		params.Set("type", transactionType)
	}
	var resp *TransactionLog
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/transaction-log", params, nil, &resp, defaultEPL)
}

// SetMarginMode set margin mode to  either of ISOLATED_MARGIN, REGULAR_MARGIN(i.e. Cross margin), PORTFOLIO_MARGIN
func (e *Exchange) SetMarginMode(ctx context.Context, marginMode string) (*SetMarginModeResponse, error) {
	if marginMode == "" {
		return nil, fmt.Errorf("%w, margin mode should be either of ISOLATED_MARGIN, REGULAR_MARGIN, or PORTFOLIO_MARGIN", errInvalidMode)
	}
	arg := &struct {
		SetMarginMode string `json:"setMarginMode"`
	}{SetMarginMode: marginMode}

	var resp *SetMarginModeResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/set-margin-mode", nil, arg, &resp, defaultEPL)
}

// SetSpotHedging to turn on/off Spot hedging feature in Portfolio margin for Unified account.
func (e *Exchange) SetSpotHedging(ctx context.Context, setHedgingModeOn bool) error {
	resp := struct{}{}
	setHedgingMode := "OFF"
	if setHedgingModeOn {
		setHedgingMode = "ON"
	}
	return e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/set-hedging-mode", nil, &map[string]string{"setHedgingMode": setHedgingMode}, &resp, defaultEPL)
}

// SetMMP Market Maker Protection (MMP) is an automated mechanism designed to protect market makers (MM) against liquidity risks and over-exposure in the market.
func (e *Exchange) SetMMP(ctx context.Context, arg *MMPRequestParam) error {
	if arg == nil {
		return errNilArgument
	}
	if arg.BaseCoin == "" {
		return errBaseNotSet
	}
	if arg.TimeWindowMS <= 0 {
		return errTimeWindowRequired
	}
	if arg.FrozenPeriod <= 0 {
		return errFrozenPeriodRequired
	}
	if arg.TradeQuantityLimit <= 0 {
		return fmt.Errorf("%w, trade quantity limit required", errQuantityLimitRequired)
	}
	if arg.DeltaLimit <= 0 {
		return fmt.Errorf("%w, delta limit is required", errQuantityLimitRequired)
	}
	return e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/mmp-modify", nil, arg, &struct{}{}, defaultEPL)
}

// ResetMMP resets MMP.
// once the mmp triggered, you can unfreeze the account by this endpoint
func (e *Exchange) ResetMMP(ctx context.Context, baseCoin string) error {
	if baseCoin == "" {
		return errBaseNotSet
	}
	arg := &struct {
		BaseCoin string `json:"baseCoin"`
	}{BaseCoin: baseCoin}
	return e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/mmp-reset", nil, arg, &struct{}{}, defaultEPL)
}

// GetMMPState retrieve Market Maker Protection (MMP) states for different coins.
func (e *Exchange) GetMMPState(ctx context.Context, baseCoin string) (*MMPStates, error) {
	if baseCoin == "" {
		return nil, errBaseNotSet
	}
	arg := &struct {
		BaseCoin string `json:"baseCoin"`
	}{BaseCoin: baseCoin}
	var resp *MMPStates
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/mmp-state", nil, arg, &resp, defaultEPL)
}

// GetCoinExchangeRecords queries the coin exchange records.
func (e *Exchange) GetCoinExchangeRecords(ctx context.Context, fromCoin, toCoin, cursor string, limit int64) (*CoinExchangeRecords, error) {
	params := url.Values{}
	if fromCoin != "" {
		params.Set("fromCoin", fromCoin)
	}
	if toCoin != "" {
		params.Set("toCoin", toCoin)
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *CoinExchangeRecords
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/exchange/order-record", params, nil, &resp, getExchangeOrderRecordEPL)
}

// GetDeliveryRecord retrieves delivery records of USDC futures and Options, sorted by deliveryTime in descending order
func (e *Exchange) GetDeliveryRecord(ctx context.Context, category, symbol, cursor string, expiryDate time.Time, limit int64) (*DeliveryRecord, error) {
	if !slices.Contains([]string{cLinear, cOption}, category) {
		return nil, fmt.Errorf("%w, valid category values are %v", errInvalidCategory, []string{cLinear, cOption})
	}
	params := url.Values{}
	params.Set("category", category)
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if !expiryDate.IsZero() {
		params.Set("expData", expiryDate.Format(longDatedFormat))
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *DeliveryRecord
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/delivery-record", params, nil, &resp, defaultEPL)
}

// GetUSDCSessionSettlement retrieves session settlement records of USDC perpetual and futures
func (e *Exchange) GetUSDCSessionSettlement(ctx context.Context, category, symbol, cursor string, limit int64) (*SettlementSession, error) {
	if category != cLinear {
		return nil, fmt.Errorf("%w, valid category value is %v", errInvalidCategory, cLinear)
	}
	params := url.Values{}
	params.Set("category", category)
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *SettlementSession
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/settlement-record", params, nil, &resp, defaultEPL)
}

// GetAssetInfo retrieves asset information
func (e *Exchange) GetAssetInfo(ctx context.Context, accountType, coin string) (*AccountInfos, error) {
	if accountType == "" {
		return nil, errMissingAccountType
	}
	params := url.Values{}
	params.Set("accountType", accountType)
	if coin != "" {
		params.Set("coin", coin)
	}
	var resp *AccountInfos
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/transfer/query-asset-info", params, nil, &resp, getAssetTransferQueryInfoEPL)
}

func fillCoinBalanceFetchParams(accountType, memberID, coin string, withBonus int64, coinRequired bool) (url.Values, error) {
	if accountType == "" {
		return nil, errMissingAccountType
	}
	params := url.Values{}
	params.Set("accountType", accountType)
	if memberID != "" {
		params.Set("memberId", memberID)
	}
	if coinRequired && coin == "" {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if coin != "" {
		params.Set("coin", coin)
	}
	if withBonus > 0 {
		params.Set("withBonus", strconv.FormatInt(withBonus, 10))
	}
	return params, nil
}

// GetAllCoinBalance retrieves all coin balance of all account types under the master account, and sub account.
// It is not allowed to get master account coin balance via sub account api key.
func (e *Exchange) GetAllCoinBalance(ctx context.Context, accountType, memberID, coin string, withBonus int64) (*CoinBalances, error) {
	params, err := fillCoinBalanceFetchParams(accountType, memberID, coin, withBonus, false)
	if err != nil {
		return nil, err
	}
	var resp *CoinBalances
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/transfer/query-account-coins-balance", params, nil, &resp, getAssetAccountCoinBalanceEPL)
}

// GetSingleCoinBalance retrieves the balance of a specific coin in a specific account type. Supports querying sub UID's balance.
func (e *Exchange) GetSingleCoinBalance(ctx context.Context, accountType, coin, memberID string, withBonus, withTransferSafeAmount int64) (*CoinBalance, error) {
	params, err := fillCoinBalanceFetchParams(accountType, memberID, coin, withBonus, true)
	if err != nil {
		return nil, err
	}
	if withTransferSafeAmount > 0 {
		params.Set("withTransferSafeAmount", strconv.FormatInt(withTransferSafeAmount, 10))
	}
	var resp *CoinBalance
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/transfer/query-account-coin-balance", params, nil, &resp, getAssetTransferQueryTransferCoinListEPL)
}

// GetTransferableCoin the transferable coin list between each account type
func (e *Exchange) GetTransferableCoin(ctx context.Context, fromAccountType, toAccountType string) (*TransferableCoins, error) {
	if fromAccountType == "" {
		return nil, fmt.Errorf("%w, from account type not specified", errMissingAccountType)
	}
	if toAccountType == "" {
		return nil, fmt.Errorf("%w, to account type not specified", errMissingAccountType)
	}
	params := url.Values{}
	params.Set("fromAccountType", fromAccountType)
	params.Set("toAccountType", toAccountType)
	var resp *TransferableCoins
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/transfer/query-transfer-coin-list", params, nil, &resp, getAssetTransferQueryTransferCoinListEPL)
}

// CreateInternalTransfer create the internal transfer between different account types under the same UID.
// Each account type has its own acceptable coins, e.g, you cannot transfer USDC from SPOT to CONTRACT.
// Please refer to transferable coin list API to find out more.
func (e *Exchange) CreateInternalTransfer(ctx context.Context, arg *TransferParams) (string, error) {
	if arg == nil {
		return "", errNilArgument
	}
	if arg.TransferID.IsNil() {
		return "", errMissingTransferID
	}
	if arg.Coin.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return "", order.ErrAmountIsInvalid
	}
	if arg.FromAccountType == "" {
		return "", fmt.Errorf("%w, from account type not specified", errMissingAccountType)
	}
	if arg.ToAccountType == "" {
		return "", fmt.Errorf("%w, to account type not specified", errMissingAccountType)
	}
	resp := struct {
		TransferID string `json:"transferId"`
	}{}
	return resp.TransferID, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/transfer/inter-transfer", nil, arg, &resp, interTransferEPL)
}

// GetInternalTransferRecords retrieves the internal transfer records between different account types under the same UID.
func (e *Exchange) GetInternalTransferRecords(ctx context.Context, transferID, coin, status, cursor string, startTime, endTime time.Time, limit int64) (*TransferResponse, error) {
	params := fillTransferQueryParams(transferID, coin, status, cursor, startTime, endTime, limit)
	var resp *TransferResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/transfer/query-inter-transfer-list", params, nil, &resp, getAssetInterTransferListEPL)
}

// GetSubUID retrieves the sub UIDs under a main UID
func (e *Exchange) GetSubUID(ctx context.Context) (*SubUID, error) {
	var resp *SubUID
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/transfer/query-sub-member-list", nil, nil, &resp, getSubMemberListEPL)
}

// EnableUniversalTransferForSubUID Transfer between sub-sub or main-sub
// Use this endpoint to enable a subaccount to take part in a universal transfer. It is a one-time switch which, once thrown, enables a subaccount permanently.
// If not set, your subaccount cannot use universal transfers.
func (e *Exchange) EnableUniversalTransferForSubUID(ctx context.Context, subMemberIDs ...string) error {
	if len(subMemberIDs) == 0 {
		return errMembersIDsNotSet
	}
	arg := map[string][]string{
		"subMemberIDs": subMemberIDs,
	}
	return e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/transfer/save-transfer-sub-member", nil, &arg, &struct{}{}, saveTransferSubMemberEPL)
}

// CreateUniversalTransfer transfer between sub-sub or main-sub. Please make sure you have enabled universal transfer on your sub UID in advance.
// To use sub acct api key, it must have "SubMemberTransferList" permission
// When use sub acct api key, it can only transfer to main account
// You can not transfer between the same UID
func (e *Exchange) CreateUniversalTransfer(ctx context.Context, arg *TransferParams) (string, error) {
	if arg == nil {
		return "", errNilArgument
	}
	if arg.TransferID.IsNil() {
		return "", errMissingTransferID
	}
	if arg.Coin.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return "", order.ErrAmountIsInvalid
	}
	if arg.FromAccountType == "" {
		return "", fmt.Errorf("%w, from account type not specified", errMissingAccountType)
	}
	if arg.ToAccountType == "" {
		return "", fmt.Errorf("%w, to account type not specified", errMissingAccountType)
	}
	if arg.FromMemberID == 0 {
		return "", fmt.Errorf("%w, fromMemberId is missing", errMemberIDRequired)
	}
	if arg.ToMemberID == 0 {
		return "", fmt.Errorf("%w, toMemberId is missing", errMemberIDRequired)
	}
	resp := struct {
		TransferID string `json:"transferId"`
	}{}
	return resp.TransferID, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/transfer/universal-transfer", nil, arg, &resp, universalTransferEPL)
}

func fillTransferQueryParams(transferID, coin, status, cursor string, startTime, endTime time.Time, limit int64) url.Values {
	params := url.Values{}
	if transferID != "" {
		params.Set("transferId", transferID)
	}
	if coin != "" {
		params.Set("coin", coin)
	}
	if status != "" {
		params.Set("status", status)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return params
}

// GetUniversalTransferRecords query universal transfer records
// Main acct api key or Sub acct api key are both supported
// Main acct api key needs "SubMemberTransfer" permission
// Sub acct api key needs "SubMemberTransferList" permission
func (e *Exchange) GetUniversalTransferRecords(ctx context.Context, transferID, coin, status, cursor string, startTime, endTime time.Time, limit int64) (*TransferResponse, error) {
	params := fillTransferQueryParams(transferID, coin, status, cursor, startTime, endTime, limit)
	var resp *TransferResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/transfer/query-universal-transfer-list", params, nil, &resp, getAssetUniversalTransferListEPL)
}

// GetAllowedDepositCoinInfo retrieves allowed deposit coin information. To find out paired chain of coin, please refer coin info api.
func (e *Exchange) GetAllowedDepositCoinInfo(ctx context.Context, coin, chain, cursor string, limit int64) (*AllowedDepositCoinInfo, error) {
	params := url.Values{}
	if coin != "" {
		params.Set("coin", coin)
	}
	if chain != "" {
		params.Set("chain", chain)
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *AllowedDepositCoinInfo
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/deposit/query-allowed-list", params, nil, &resp, defaultEPL)
}

// SetDepositAccount sets the auto transfer account after deposit (mirrors the behaviour available via the Bybit account settings interface)
// account types: CONTRACT Derivatives Account
// 'SPOT' Spot Account 'INVESTMENT' ByFi Account (The service has been offline) 'OPTION' USDC Account 'UNIFIED' UMA or UTA 'FUND' Funding Account
func (e *Exchange) SetDepositAccount(ctx context.Context, accountType string) (*StatusResponse, error) {
	if accountType == "" {
		return nil, errMissingAccountType
	}
	arg := &struct {
		AccountType string `json:"accountType"`
	}{
		AccountType: accountType,
	}
	var resp *StatusResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/deposit/deposit-to-account", nil, &arg, &resp, defaultEPL)
}

func fillDepositRecordsParams(coin, cursor string, startTime, endTime time.Time, limit int64) url.Values {
	params := url.Values{}
	if coin != "" {
		params.Set("coin", coin)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return params
}

// GetDepositRecords query deposit records.
func (e *Exchange) GetDepositRecords(ctx context.Context, coin, cursor string, startTime, endTime time.Time, limit int64) (*DepositRecords, error) {
	params := fillDepositRecordsParams(coin, cursor, startTime, endTime, limit)
	var resp *DepositRecords
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/deposit/query-record", params, nil, &resp, getAssetDepositRecordsEPL)
}

// GetSubDepositRecords query subaccount's deposit records by main UID's API key. on chain
func (e *Exchange) GetSubDepositRecords(ctx context.Context, subMemberID, coin, cursor string, startTime, endTime time.Time, limit int64) (*DepositRecords, error) {
	if subMemberID == "" {
		return nil, errMembersIDsNotSet
	}
	params := fillDepositRecordsParams(coin, cursor, startTime, endTime, limit)
	params.Set("subMemberId", subMemberID)
	var resp *DepositRecords
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/deposit/query-sub-member-record", params, nil, &resp, getAssetDepositSubMemberRecordsEPL)
}

// GetInternalDepositRecordsOffChain retrieves deposit records within the Bybit platform. These transactions are not on the blockchain.
func (e *Exchange) GetInternalDepositRecordsOffChain(ctx context.Context, coin, cursor string, startTime, endTime time.Time, limit int64) (*InternalDepositRecords, error) {
	params := fillDepositRecordsParams(coin, cursor, startTime, endTime, limit)
	var resp *InternalDepositRecords
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/deposit/query-internal-record", params, nil, &resp, defaultEPL)
}

// GetMasterDepositAddress retrieves the deposit address information of MASTER account.
func (e *Exchange) GetMasterDepositAddress(ctx context.Context, coin currency.Code, chainType string) (*DepositAddresses, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", coin.String())
	if chainType != "" {
		params.Set("chainType", chainType)
	}
	var resp *DepositAddresses
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/deposit/query-address", params, nil, &resp, getAssetDepositRecordsEPL)
}

// GetSubDepositAddress retrieves the deposit address information of SUB account.
func (e *Exchange) GetSubDepositAddress(ctx context.Context, coin currency.Code, chainType, subMemberID string) (*DepositAddresses, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if chainType == "" {
		return nil, errMissingChainType
	}
	if subMemberID == "" {
		return nil, errMembersIDsNotSet
	}
	params := url.Values{}
	params.Set("coin", coin.String())
	params.Set("chainType", chainType)
	params.Set("subMemberId", subMemberID)
	var resp *DepositAddresses
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/deposit/query-sub-member-address", params, nil, &resp, getAssetDepositSubMemberAddressEPL)
}

// GetCoinInfo retrieves coin information, including chain information, withdraw and deposit status.
func (e *Exchange) GetCoinInfo(ctx context.Context, coin currency.Code) (*CoinInfo, error) {
	params := url.Values{}
	if coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	var resp *CoinInfo
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/coin/query-info", params, nil, &resp, getAssetCoinInfoEPL)
}

// GetWithdrawalRecords query withdrawal records.
// endTime - startTime should be less than 30 days. Query last 30 days records by default.
// Can query by the master UID's api key only
func (e *Exchange) GetWithdrawalRecords(ctx context.Context, coin currency.Code, withdrawalID, withdrawType, cursor string, startTime, endTime time.Time, limit int64) (*WithdrawalRecords, error) {
	params := url.Values{}
	if withdrawalID != "" {
		params.Set("withdrawID", withdrawalID)
	}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	if withdrawType != "" {
		params.Set("withdrawType", withdrawType)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *WithdrawalRecords
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/withdraw/query-record", params, nil, &resp, getWithdrawRecordsEPL)
}

// GetWithdrawableAmount retrieves withdrawable amount information using currency code
func (e *Exchange) GetWithdrawableAmount(ctx context.Context, coin currency.Code) (*WithdrawableAmount, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", coin.String())
	var resp *WithdrawableAmount
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/withdraw/withdrawable-amount", params, nil, &resp, defaultEPL)
}

// WithdrawCurrency Withdraw assets from your Bybit account. You can make an off-chain transfer if the target wallet address is from Bybit. This means that no blockchain fee will be charged.
func (e *Exchange) WithdrawCurrency(ctx context.Context, arg *WithdrawalParam) (string, error) {
	if arg == nil {
		return "", errNilArgument
	}
	if arg.Coin.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if arg.Chain == "" {
		return "", errMissingChainInformation
	}
	if arg.Address == "" {
		return "", errMissingAddressInfo
	}
	if arg.Amount <= 0 {
		return "", limits.ErrAmountBelowMin
	}
	if arg.Timestamp == 0 {
		arg.Timestamp = time.Now().UnixMilli()
	}
	resp := struct {
		ID string `json:"id"`
	}{}
	return resp.ID, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/withdraw/create", nil, arg, &resp, createWithdrawalEPL)
}

// CancelWithdrawal cancel the withdrawal
func (e *Exchange) CancelWithdrawal(ctx context.Context, id string) (*StatusResponse, error) {
	if id == "" {
		return nil, errMissingWithdrawalID
	}
	arg := &struct {
		ID string `json:"id"`
	}{
		ID: id,
	}
	var resp *StatusResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/withdraw/cancel", nil, arg, &resp, cancelWithdrawalEPL)
}

// CreateNewSubUserID created a new sub user id. Use master user's api key only.
func (e *Exchange) CreateNewSubUserID(ctx context.Context, arg *CreateSubUserParams) (*SubUserItem, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.Username == "" {
		return nil, errMissingUsername
	}
	if arg.MemberType <= 0 {
		return nil, errInvalidMemberType
	}
	var resp *SubUserItem
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/create-sub-member", nil, &arg, &resp, userCreateSubMemberEPL)
}

// CreateSubUIDAPIKey create new API key for those newly created sub UID. Use master user's api key only.
func (e *Exchange) CreateSubUIDAPIKey(ctx context.Context, arg *SubUIDAPIKeyParam) (*SubUIDAPIResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.Subuid <= 0 {
		return nil, fmt.Errorf("%w, subuid", errMissingUserID)
	}
	var resp *SubUIDAPIResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/create-sub-api", nil, arg, &resp, userCreateSubAPIKeyEPL)
}

// GetSubUIDList get all sub uid of master account. Use master user's api key only.
func (e *Exchange) GetSubUIDList(ctx context.Context) ([]SubUserItem, error) {
	resp := struct {
		SubMembers []SubUserItem `json:"subMembers"`
	}{}
	return resp.SubMembers, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/user/query-sub-members", nil, nil, &resp, userQuerySubMembersEPL)
}

// FreezeSubUID freeze Sub UID. Use master user's api key only.
func (e *Exchange) FreezeSubUID(ctx context.Context, subUID string, frozen bool) error {
	if subUID == "" {
		return fmt.Errorf("%w, subuid", errMissingUserID)
	}
	arg := &struct {
		SubUID string `json:"subuid"`
		Frozen int64  `json:"frozen"`
	}{
		SubUID: subUID,
	}
	if frozen {
		arg.Frozen = 0
	} else {
		arg.Frozen = 1
	}
	return e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/frozen-sub-member", nil, arg, &struct{}{}, userFrozenSubMemberEPL)
}

// GetAPIKeyInformation retrieves the information of the api key.
// Use the api key pending to be checked to call the endpoint.
// Both master and sub user's api key are applicable.
func (e *Exchange) GetAPIKeyInformation(ctx context.Context) (*SubUIDAPIResponse, error) {
	var resp *SubUIDAPIResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/user/query-api", nil, nil, &resp, userQueryAPIEPL)
}

// GetSubAccountAllAPIKeys retrieves all api keys information of a sub UID.
func (e *Exchange) GetSubAccountAllAPIKeys(ctx context.Context, subMemberID, cursor string, limit int64) (*SubAccountAPIKeys, error) {
	if subMemberID == "" {
		return nil, errMembersIDsNotSet
	}
	params := url.Values{}
	params.Set("subMemberId", subMemberID)
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *SubAccountAPIKeys
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "  /v5/user/sub-apikeys", params, nil, &resp, defaultEPL)
}

// GetUIDWalletType retrieves available wallet types for the master account or sub account
func (e *Exchange) GetUIDWalletType(ctx context.Context, memberIDs string) (*WalletType, error) {
	if memberIDs == "" {
		return nil, errMembersIDsNotSet
	}
	var resp *WalletType
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/user/get-member-type", nil, nil, &resp, defaultEPL)
}

// ModifyMasterAPIKey modify the settings of master api key.
// Use the api key pending to be modified to call the endpoint. Use master user's api key only.
func (e *Exchange) ModifyMasterAPIKey(ctx context.Context, arg *SubUIDAPIKeyUpdateParam) (*SubUIDAPIResponse, error) {
	if arg == nil || reflect.DeepEqual(*arg, SubUIDAPIKeyUpdateParam{}) {
		return nil, errNilArgument
	}
	if arg.IPs == "" && len(arg.IPAddresses) > 0 {
		arg.IPs = strings.Join(arg.IPAddresses, ",")
	}
	var resp *SubUIDAPIResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/update-api", nil, arg, &resp, userUpdateAPIEPL)
}

// ModifySubAPIKey modifies the settings of sub api key. Use the api key pending to be modified to call the endpoint. Use sub user's api key only.
func (e *Exchange) ModifySubAPIKey(ctx context.Context, arg *SubUIDAPIKeyUpdateParam) (*SubUIDAPIResponse, error) {
	if arg == nil || reflect.DeepEqual(*arg, SubUIDAPIKeyUpdateParam{}) {
		return nil, errNilArgument
	}
	if arg.IPs == "" && len(arg.IPAddresses) > 0 {
		arg.IPs = strings.Join(arg.IPAddresses, ",")
	}
	var resp *SubUIDAPIResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/update-sub-api", nil, &arg, &resp, userUpdateSubAPIEPL)
}

// DeleteSubUID delete a sub UID. Before deleting the UID, please make sure there is no asset.
// Use master user's api key**.
func (e *Exchange) DeleteSubUID(ctx context.Context, subMemberID string) error {
	if subMemberID == "" {
		return errMemberIDRequired
	}
	arg := &struct {
		SubMemberID string `json:"subMemberId"`
	}{SubMemberID: subMemberID}
	var resp any
	return e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/del-submember", nil, arg, &resp, defaultEPL)
}

// DeleteMasterAPIKey delete the api key of master account.
// Use the api key pending to be delete to call the endpoint. Use master user's api key only.
func (e *Exchange) DeleteMasterAPIKey(ctx context.Context) error {
	return e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/delete-api", nil, nil, &struct{}{}, userDeleteAPIEPL)
}

// DeleteSubAccountAPIKey delete the api key of sub account.
// Use the api key pending to be delete to call the endpoint. Use sub user's api key only.
func (e *Exchange) DeleteSubAccountAPIKey(ctx context.Context, subAccountUID string) error {
	if subAccountUID == "" {
		return fmt.Errorf("%w, sub-account id missing", errMissingUserID)
	}
	arg := &struct {
		UID string `json:"uid"`
	}{
		UID: subAccountUID,
	}
	return e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/delete-sub-api", nil, arg, &struct{}{}, userDeleteSubAPIEPL)
}

// GetAffiliateUserInfo the API is used for affiliate to get their users information
// The master account uid of affiliate's client
func (e *Exchange) GetAffiliateUserInfo(ctx context.Context, uid string) (*AffiliateCustomerInfo, error) {
	if uid == "" {
		return nil, errMissingUserID
	}
	params := url.Values{}
	params.Set("uid", uid)
	var resp *AffiliateCustomerInfo
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/user/aff-customer-info", params, nil, &resp, defaultEPL)
}

// GetLeverageTokenInfo query leverage token information
// Abbreviation of the LT, such as BTC3L
func (e *Exchange) GetLeverageTokenInfo(ctx context.Context, ltCoin currency.Code) ([]LeverageTokenInfo, error) {
	params := url.Values{}
	if !ltCoin.IsEmpty() {
		params.Set("ltCoin", ltCoin.String())
	}
	resp := struct {
		List []LeverageTokenInfo `json:"list"`
	}{}
	return resp.List, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-lever-token/info", params, nil, &resp, defaultEPL)
}

// GetLeveragedTokenMarket retrieves leverage token market information
func (e *Exchange) GetLeveragedTokenMarket(ctx context.Context, ltCoin currency.Code) (*LeveragedTokenMarket, error) {
	if ltCoin.IsEmpty() {
		return nil, fmt.Errorf("%w, 'ltCoin' is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("ltCoin", ltCoin.String())
	var resp *LeveragedTokenMarket
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-lever-token/reference", params, nil, &resp, defaultEPL)
}

// PurchaseLeverageToken purcases a leverage token.
func (e *Exchange) PurchaseLeverageToken(ctx context.Context, ltCoin currency.Code, amount float64, serialNumber string) (*LeverageToken, error) {
	if ltCoin.IsEmpty() {
		return nil, fmt.Errorf("%w, 'ltCoin' is required", currency.ErrCurrencyCodeEmpty)
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	arg := &struct {
		LTCoin       string  `json:"ltCoin"`
		LTAmount     float64 `json:"amount,string"`
		SerialNumber string  `json:"serialNo,omitempty"`
	}{
		LTCoin:       ltCoin.String(),
		LTAmount:     amount,
		SerialNumber: serialNumber,
	}
	var resp *LeverageToken
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/spot-lever-token/purchase", nil, arg, &resp, spotLeverageTokenPurchaseEPL)
}

// RedeemLeverageToken redeem leverage token
func (e *Exchange) RedeemLeverageToken(ctx context.Context, ltCoin currency.Code, quantity float64, serialNumber string) (*RedeemToken, error) {
	if ltCoin.IsEmpty() {
		return nil, fmt.Errorf("%w, 'ltCoin' is required", currency.ErrCurrencyCodeEmpty)
	}
	if quantity <= 0 {
		return nil, fmt.Errorf("%w, quantity=%f", limits.ErrAmountBelowMin, quantity)
	}
	arg := &struct {
		LTCoin       string  `json:"ltCoin"`
		Quantity     float64 `json:"quantity,string"`
		SerialNumber string  `json:"serialNo,omitempty"`
	}{
		LTCoin:       ltCoin.String(),
		Quantity:     quantity,
		SerialNumber: serialNumber,
	}
	var resp *RedeemToken
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/spot-lever-token/redeem", nil, &arg, &resp, spotLeverTokenRedeemEPL)
}

// GetPurchaseAndRedemptionRecords retrieves purchase or redeem history.
// ltOrderType	false	integer	LT order type. 1: purchase, 2: redemption
func (e *Exchange) GetPurchaseAndRedemptionRecords(ctx context.Context, ltCoin currency.Code, orderID, serialNo string, startTime, endTime time.Time, ltOrderType, limit int64) ([]RedeemPurchaseRecord, error) {
	params := url.Values{}
	if !ltCoin.IsEmpty() {
		params.Set("ltCoin", ltCoin.String())
	}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if serialNo != "" {
		params.Set("serialNo", serialNo)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if ltOrderType != 0 {
		params.Set("ltOrderType", strconv.FormatInt(ltOrderType, 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	resp := struct {
		List []RedeemPurchaseRecord `json:"list"`
	}{}
	return resp.List, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-lever-token/order-record", params, nil, &resp, getSpotLeverageTokenOrderRecordsEPL)
}

// ToggleMarginTrade turn on / off spot margin trade
// Your account needs to activate spot margin first; i.e., you must have finished the quiz on web / app.
// spotMarginMode '1': on, '0': off
func (e *Exchange) ToggleMarginTrade(ctx context.Context, spotMarginMode bool) (*SpotMarginMode, error) {
	arg := &SpotMarginMode{}
	if spotMarginMode {
		arg.SpotMarginMode = "1"
	} else {
		arg.SpotMarginMode = "0"
	}
	var resp *SpotMarginMode
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/spot-margin-trade/switch-mode", nil, arg, &resp, defaultEPL)
}

// SetSpotMarginTradeLeverage set the user's maximum leverage in spot cross margin
func (e *Exchange) SetSpotMarginTradeLeverage(ctx context.Context, leverage float64) error {
	if leverage <= 2 {
		return fmt.Errorf("%w, leverage. value range from [2  to 10]", errInvalidLeverage)
	}
	return e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/spot-margin-trade/set-leverage", nil, &map[string]string{"leverage": strconv.FormatFloat(leverage, 'f', -1, 64)}, &struct{}{}, defaultEPL)
}

// GetVIPMarginData retrieves public VIP Margin data
func (e *Exchange) GetVIPMarginData(ctx context.Context, vipLevel, ccy string) (*VIPMarginData, error) {
	params := url.Values{}
	if vipLevel != "" {
		params.Set("vipLevel", vipLevel)
	}
	if ccy != "" {
		params.Set("currency", ccy)
	}
	var resp *VIPMarginData
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "spot-cross-margin-trade/data", defaultEPL, &resp)
}

// GetMarginCoinInfo retrieves margin coin information.
func (e *Exchange) GetMarginCoinInfo(ctx context.Context, coin currency.Code) ([]MarginCoinInfo, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	resp := struct {
		List []MarginCoinInfo `json:"list"`
	}{}
	return resp.List, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("spot-cross-margin-trade/pledge-token", params), defaultEPL, &resp)
}

// GetBorrowableCoinInfo retrieves borrowable coin info list.
func (e *Exchange) GetBorrowableCoinInfo(ctx context.Context, coin currency.Code) ([]BorrowableCoinInfo, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	resp := struct {
		List []BorrowableCoinInfo `json:"list"`
	}{}
	return resp.List, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("spot-cross-margin-trade/borrow-token", params), defaultEPL, &resp)
}

// GetInterestAndQuota retrieves interest and quota information.
func (e *Exchange) GetInterestAndQuota(ctx context.Context, coin currency.Code) (*InterestAndQuota, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", coin.String())
	var resp *InterestAndQuota
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-cross-margin-trade/loan-info", params, nil, &resp, getSpotCrossMarginTradeLoanInfoEPL)
}

// GetLoanAccountInfo retrieves loan account information.
func (e *Exchange) GetLoanAccountInfo(ctx context.Context) (*AccountLoanInfo, error) {
	var resp *AccountLoanInfo
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-cross-margin-trade/account", nil, nil, &resp, getSpotCrossMarginTradeAccountEPL)
}

// Borrow borrows a coin.
func (e *Exchange) Borrow(ctx context.Context, arg *LendArgument) (*BorrowResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.AmountToBorrow <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	var resp *BorrowResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/spot-cross-margin-trade/loan", nil, arg, &resp, spotCrossMarginTradeLoanEPL)
}

// Repay repay a debt.
func (e *Exchange) Repay(ctx context.Context, arg *LendArgument) (*RepayResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.AmountToBorrow <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	var resp *RepayResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/spot-cross-margin-trade/repay", nil, arg, &resp, spotCrossMarginTradeRepayEPL)
}

// GetBorrowOrderDetail represents the borrow order detail.
// Status '0'(default)：get all kinds of status '1'：uncleared '2'：cleared
func (e *Exchange) GetBorrowOrderDetail(ctx context.Context, startTime, endTime time.Time, coin currency.Code, status, limit int64) ([]BorrowOrderDetail, error) {
	params := url.Values{}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if status != 0 {
		params.Set("status", strconv.FormatInt(status, 10))
	}
	resp := struct {
		List []BorrowOrderDetail `json:"list"`
	}{}
	return resp.List, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-cross-margin-trade/orders", params, nil, &resp, getSpotCrossMarginTradeOrdersEPL)
}

// GetRepaymentOrderDetail retrieves repayment order detail.
func (e *Exchange) GetRepaymentOrderDetail(ctx context.Context, startTime, endTime time.Time, coin currency.Code, limit int64) ([]CoinRepaymentResponse, error) {
	params := url.Values{}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	resp := struct {
		List []CoinRepaymentResponse `json:"list"`
	}{}
	return resp.List, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-cross-margin-trade/repay-history", params, nil, &resp, getSpotCrossMarginTradeRepayHistoryEPL)
}

// ToggleMarginTradeNormal turn on / off spot margin trade
// Your account needs to activate spot margin first; i.e., you must have finished the quiz on web / app.
// spotMarginMode '1': on, '0': off
func (e *Exchange) ToggleMarginTradeNormal(ctx context.Context, spotMarginMode bool) (*SpotMarginMode, error) {
	arg := &SpotMarginMode{}
	if spotMarginMode {
		arg.SpotMarginMode = "1"
	} else {
		arg.SpotMarginMode = "0"
	}
	var resp *SpotMarginMode
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/spot-cross-margin-trade/switch", nil, arg, &resp, spotCrossMarginTradeSwitchEPL)
}

// GetProductInfo represents a product info.
func (e *Exchange) GetProductInfo(ctx context.Context, productID string) (*InstitutionalProductInfo, error) {
	params := url.Values{}
	if productID != "" {
		params.Set("productId", productID)
	}
	var resp *InstitutionalProductInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("ins-loan/product-infos", params), defaultEPL, &resp)
}

// GetInstitutionalLengingMarginCoinInfo retrieves institutional lending margin coin information.
// ProductId. If not passed, then return all product margin coin. For spot, it returns coin that convertRation greater than 0.
func (e *Exchange) GetInstitutionalLengingMarginCoinInfo(ctx context.Context, productID string) (*InstitutionalMarginCoinInfo, error) {
	params := url.Values{}
	if productID != "" {
		params.Set("productId", productID)
	}
	var resp *InstitutionalMarginCoinInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("ins-loan/ensure-tokens-convert", params), defaultEPL, &resp)
}

// GetInstitutionalLoanOrders retrieves institutional loan orders.
func (e *Exchange) GetInstitutionalLoanOrders(ctx context.Context, orderID string, startTime, endTime time.Time, limit int64) ([]LoanOrderDetails, error) {
	params := url.Values{}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	resp := struct {
		Loans []LoanOrderDetails `json:"loanInfo"`
	}{}
	return resp.Loans, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/ins-loan/loan-order", params, nil, &resp, defaultEPL)
}

// GetInstitutionalRepayOrders retrieves list of repaid order information.
func (e *Exchange) GetInstitutionalRepayOrders(ctx context.Context, startTime, endTime time.Time, limit int64) ([]OrderRepayInfo, error) {
	params := url.Values{}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	resp := struct {
		RepayInfo []OrderRepayInfo `json:"repayInfo"`
	}{}
	return resp.RepayInfo, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/ins-loan/repaid-history", params, nil, &resp, defaultEPL)
}

// GetLTV retrieves a loan-to-value(LTV)
func (e *Exchange) GetLTV(ctx context.Context) (*LTVInfo, error) {
	var resp *LTVInfo
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/ins-loan/ltv-convert", nil, nil, &resp, defaultEPL)
}

// BindOrUnbindUID For the INS loan product, you can bind new UID to risk unit or unbind UID out from risk unit.
// possible 'operate' values: 0: bind, 1: unbind
func (e *Exchange) BindOrUnbindUID(ctx context.Context, uid, operate string) (*BindOrUnbindUIDResponse, error) {
	if uid == "" {
		return nil, fmt.Errorf("%w, uid is required", errMissingUserID)
	}
	if operate != "0" && operate != "1" {
		return nil, errors.New("operation type required; 0:bind and 1:unbind")
	}
	arg := &struct {
		UID     string `json:"uid"`
		Operate string `json:"operate"`
	}{UID: uid, Operate: operate}
	var resp *BindOrUnbindUIDResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/ins-loan/association-uid", nil, arg, &resp, defaultEPL)
}

// GetC2CLendingCoinInfo retrieves C2C basic information of lending coins
func (e *Exchange) GetC2CLendingCoinInfo(ctx context.Context, coin currency.Code) ([]C2CLendingCoinInfo, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	resp := struct {
		List []C2CLendingCoinInfo `json:"list"`
	}{}
	return resp.List, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/lending/info", params, nil, &resp, defaultEPL)
}

// C2CDepositFunds lending funds to Bybit asset pool
func (e *Exchange) C2CDepositFunds(ctx context.Context, arg *C2CLendingFundsParams) (*C2CLendingFundResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Quantity <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	var resp *C2CLendingFundResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/lending/purchase", nil, &arg, &resp, defaultEPL)
}

// C2CRedeemFunds withdraw funds from the Bybit asset pool.
func (e *Exchange) C2CRedeemFunds(ctx context.Context, arg *C2CLendingFundsParams) (*C2CLendingFundResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Quantity <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	var resp *C2CLendingFundResponse
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/lending/redeem", nil, &arg, &resp, defaultEPL)
}

// GetC2CLendingOrderRecords retrieves lending or redeem history
func (e *Exchange) GetC2CLendingOrderRecords(ctx context.Context, coin currency.Code, orderID, orderType string, startTime, endTime time.Time, limit int64) ([]C2CLendingFundResponse, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if orderType != "" {
		params.Set("orderType", orderType)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	resp := struct {
		List []C2CLendingFundResponse `json:"list"`
	}{}
	return resp.List, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/lending/history-order", params, nil, &resp, defaultEPL)
}

// GetC2CLendingAccountInfo retrieves C2C lending account information.
func (e *Exchange) GetC2CLendingAccountInfo(ctx context.Context, coin currency.Code) (*LendingAccountInfo, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	var resp *LendingAccountInfo
	return resp, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/lending/account", params, nil, &resp, defaultEPL)
}

// GetBrokerEarning exchange broker master account to query
// The data can support up to past 6 months until T-1
// startTime & endTime are either entered at the same time or not entered
// Business type. 'SPOT', 'DERIVATIVES', 'OPTIONS'
func (e *Exchange) GetBrokerEarning(ctx context.Context, businessType, cursor string, startTime, endTime time.Time, limit int64) ([]BrokerEarningItem, error) {
	params := url.Values{}
	if businessType != "" {
		params.Set("bizType", businessType)
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	resp := struct {
		List []BrokerEarningItem `json:"list"`
	}{}
	return resp.List, e.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/broker/earning-record", params, nil, &resp, defaultEPL)
}

// SendHTTPRequest sends an unauthenticated request
func (e *Exchange) SendHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result any) error {
	endpointPath, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	value := reflect.ValueOf(result)
	if value.Kind() != reflect.Ptr {
		return fmt.Errorf("expected a pointer, got %T", value)
	}
	response := &RestResponse{
		Result: result,
	}
	err = e.SendPayload(ctx, f, func() (*request.Item, error) {
		return &request.Item{
			Method:                 http.MethodGet,
			Path:                   endpointPath + bybitAPIVersion + path,
			Result:                 response,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.UnauthenticatedRequest)
	if err != nil {
		return err
	}
	if response.RetCode != 0 && response.RetMsg != "" {
		return fmt.Errorf("code: %d message: %s", response.RetCode, response.RetMsg)
	}
	return nil
}

// SendAuthHTTPRequestV5 sends an authenticated HTTP request
func (e *Exchange) SendAuthHTTPRequestV5(ctx context.Context, ePath exchange.URL, method, path string, params url.Values, arg, result any, f request.EndpointLimit) error {
	val := reflect.ValueOf(result)
	if val.Kind() != reflect.Ptr {
		return errNonePointerArgument
	} else if val.IsNil() {
		return errNilArgument
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpointPath, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	response := &RestResponse{
		Result: result,
	}
	err = e.SendPayload(ctx, f, func() (*request.Item, error) {
		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		headers := make(map[string]string)
		headers["X-BAPI-API-KEY"] = creds.Key
		headers["X-BAPI-TIMESTAMP"] = timestamp
		headers["X-BAPI-RECV-WINDOW"] = defaultRecvWindow

		var hmacSignedStr string
		var payload []byte

		switch method {
		case http.MethodGet:
			headers["Content-Type"] = "application/x-www-form-urlencoded"
			hmacSignedStr, err = getSign(timestamp+creds.Key+defaultRecvWindow+params.Encode(), creds.Secret)
		case http.MethodPost:
			headers["Content-Type"] = "application/json"
			payload, err = json.Marshal(arg)
			if err != nil {
				return nil, err
			}
			hmacSignedStr, err = getSign(timestamp+creds.Key+defaultRecvWindow+string(payload), creds.Secret)
		}
		if err != nil {
			return nil, err
		}
		headers["X-BAPI-SIGN"] = hmacSignedStr
		return &request.Item{
			Method:                 method,
			Path:                   endpointPath + common.EncodeURLValues(path, params),
			Headers:                headers,
			Body:                   bytes.NewBuffer(payload),
			Result:                 &response,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.AuthenticatedRequest)
	if response.RetCode != 0 && response.RetMsg != "" {
		return fmt.Errorf("%w code: %d message: %s", request.ErrAuthRequestFailed, response.RetCode, response.RetMsg)
	}
	if len(response.RetExtInfo.List) > 0 && response.RetCode != 0 {
		var errMessage strings.Builder
		var failed bool
		for i := range response.RetExtInfo.List {
			if response.RetExtInfo.List[i].Code != 0 {
				failed = true
				errMessage.WriteString(fmt.Sprintf("code: %d message: %s ", response.RetExtInfo.List[i].Code, response.RetExtInfo.List[i].Message))
			}
		}
		if failed {
			return fmt.Errorf("%w %s", request.ErrAuthRequestFailed, errMessage.String())
		}
	}
	return err
}

func getSide(side string) order.Side {
	switch side {
	case sideBuy:
		return order.Buy
	case sideSell:
		return order.Sell
	default:
		return order.UnknownSide
	}
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
		return order.Closed
	case "ACTIVE":
		return order.Active
	default:
		return order.UnknownStatus
	}
}

func getSign(sign, secret string) (string, error) {
	hmacSigned, err := crypto.GetHMAC(crypto.HashSHA256, []byte(sign), []byte(secret))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hmacSigned), nil
}

// FetchAccountType if not set fetches the account type from the API, stores it and returns it. Else returns the stored account type.
func (e *Exchange) FetchAccountType(ctx context.Context) (AccountType, error) {
	e.account.m.Lock()
	defer e.account.m.Unlock()
	if e.account.accountType == 0 {
		accInfo, err := e.GetAPIKeyInformation(ctx)
		if err != nil {
			return 0, err
		}
		// From endpoint 0：regular account; 1：unified trade account
		// + 1 to make it 1 and 2 so that a zero value can be used to check if the account type has been set or not.
		e.account.accountType = AccountType(accInfo.IsUnifiedTradeAccount + 1)
	}
	return e.account.accountType, nil
}

// String returns the account type as a string
func (a AccountType) String() string {
	switch a {
	case 0:
		return "unset"
	case accountTypeNormal:
		return "normal"
	case accountTypeUnified:
		return "unified"
	default:
		return "unknown"
	}
}

// RequiresUnifiedAccount checks account type and returns error if not unified
func (e *Exchange) RequiresUnifiedAccount(ctx context.Context) error {
	at, err := e.FetchAccountType(ctx)
	if err != nil {
		return nil //nolint:nilerr // if we can't get the account type, we can't check if it's unified or not, fail on call
	}
	if at != accountTypeUnified {
		return fmt.Errorf("%w, account type: %s", errAPIKeyIsNotUnified, at)
	}
	return nil
}

// GetLongShortRatio retrieves long short ratio of an instrument.
func (e *Exchange) GetLongShortRatio(ctx context.Context, category, symbol string, interval kline.Interval, limit int64) ([]InstrumentInfoItem, error) {
	if category == "" {
		return nil, errCategoryNotSet
	} else if category != cLinear && category != cInverse {
		return nil, fmt.Errorf("%w, category: %s", errInvalidCategory, category)
	}
	if symbol == "" {
		return nil, errSymbolMissing
	}
	intervalString, err := intervalToString(interval)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("category", category)
	params.Set("symbol", symbol)
	params.Set("period", intervalString)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	resp := struct {
		List []InstrumentInfoItem `json:"list"`
	}{}
	return resp.List, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("market/account-ratio", params), defaultEPL, &resp)
}
