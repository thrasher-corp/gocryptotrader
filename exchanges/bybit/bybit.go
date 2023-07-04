package bybit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Bybit is the overarching type across this package
type Bybit struct {
	exchange.Base
}

const (
	bybitAPIURL     = "https://api.bybit.com"
	bybitAPIVersion = "/v5/"

	defaultRecvWindow = "5000" // 5000 milli second

	sideBuy  = "Buy"
	sideSell = "Sell"

	// Public endpoints
	bybitSpotGetSymbols   = "/spot/v1/symbols"
	bybitCandlestickChart = "/spot/quote/v1/kline"
	bybitOrderBook        = "/spot/quote/v1/depth"
	bybitRecentTrades     = "/spot/quote/v1/trades"
	bybit24HrsChange      = "/spot/quote/v1/ticker/24hr"
	bybitLastTradedPrice  = "/spot/quote/v1/ticker/price"
	bybitMergedOrderBook  = "/spot/quote/v1/depth/merged"
	bybitBestBidAskPrice  = "/spot/quote/v1/ticker/book_ticker"
	// bybitGetTickersV5     = "/v5/market/tickers"

	// Authenticated endpoints
	bybitSpotOrder                = "/spot/v1/order" // create, query, cancel
	bybitFastCancelSpotOrder      = "/spot/v1/order/fast"
	bybitBatchCancelSpotOrder     = "/spot/order/batch-cancel"
	bybitFastBatchCancelSpotOrder = "/spot/order/batch-fast-cancel"
	bybitBatchCancelByIDs         = "/spot/order/batch-cancel-by-ids"
	bybitOpenOrder                = "/spot/v1/open-orders"
	bybitPastOrder                = "/spot/v1/history-orders"
	bybitTradeHistory             = "/spot/v1/myTrades"
	bybitWalletBalance            = "/spot/v1/account"
	bybitServerTime               = "/spot/v1/time"
	bybitAccountFee               = "/v5/account/fee-rate"

	// Account asset endpoint
	bybitGetDepositAddress = "/asset/v1/private/deposit/address"
	bybitWithdrawFund      = "/asset/v1/private/withdraw"

	// --------- New ----------------------------------------------------------------
	instrumentsInfo            = "market/instruments-info"
	klineInfos                 = "market/kline"
	markPriceKline             = "market/mark-price-kline"
	indexPriceKline            = "market/index-price-kline"
	marketOrderbook            = "market/orderbook"
	marketTicker               = "market/tickers"
	marketFundingRateHistory   = "market/funding/history"
	marketRecentTrade          = "market/recent-trade"
	marketOpenInterest         = "market/open-interest"
	marketHistoricalVolatility = "market/historical-volatility"
	marketInsurance            = "market/insurance"
	marketRiskLimit            = "market/risk-limit"
	marketDeliveryPrice        = "market/delivery-price"

	// Trade
	placeOrder            = "/v5/order/create"
	amendOrder            = "/v5/order/amend"
	cancelOrder           = "/v5/order/cancel"
	openOrders            = "/v5/order/realtime"
	cancelAllOrders       = "/v5/order/cancel-all"
	tradeOrderhistory     = "/v5/order/history"
	placeBatchTradeOrders = "/v5/order/create-batch"
	amendBatchOrder       = "/v5/order/amend-batch"
	cancelBatchOrder      = "/v5/order/cancel-batch"
	tradeBorrowCheck      = "/v5/order/spot-borrow-check"

	// Position endpoints
	getPositionList          = "/v5/position/list"
	positionSetLeverage      = "/v5/position/set-leverage"
	switchTradeMode          = "/v5/position/switch-isolated"
	setTPSLMode              = "/v5/position/set-tpsl-mode"
	switchPositionMode       = "/v5/position/switch-mode"
	setPositionRiskLimit     = "/v5/position/set-risk-limit"
	stopTradingPosition      = "/v5/position/trading-stop"
	positionSetAutoAddMargin = "/v5/position/set-auto-add-margin"
)

var (
	errCategoryNotSet                     = errors.New("category not set")
	errBaseNotSet                         = errors.New("base coin not set when category is option")
	errInvalidTriggerDirection            = errors.New("invalid trigger direction")
	errInvalidTriggerPriceType            = errors.New("invalid trigger price type")
	errNilArgument                        = errors.New("nil argument")
	errNonePointerArgument                = errors.New("argument must be pointer")
	errEitherOrderIDOROrderLinkIDRequired = errors.New("either orderId or orderLinkId required")
	errNoOrderPassed                      = errors.New("no order passed")
	errSymbolOrSettleCoinRequired         = errors.New("provide symbol or settleCoin at least one")
	errInvalidTradeModeValue              = errors.New("invalid trade mode value")
	errTakeProfitOrStopLossModeMissing    = errors.New("TP/SL mode missing")
)

func intervalToString(interval kline.Interval) (string, error) {
	switch interval {
	case kline.OneMin:
		return "1", nil
	case kline.ThreeMin:
		return "3", nil
	case kline.FiveMin:
		return "5", nil
	case kline.FifteenMin:
		return "15", nil
	case kline.ThirtyMin:
		return "30", nil
	case kline.OneHour:
		return "60", nil
	case kline.TwoHour:
		return "120", nil
	case kline.FourHour:
		return "240", nil
	case kline.SixHour:
		return "360", nil
	case kline.SevenHour:
		return "720", nil
	case kline.OneDay:
		return "D", nil
	case kline.OneMonth:
		return "M", nil
	case kline.OneWeek:
		return "W", nil
	default:
		return "", kline.ErrUnsupportedInterval
	}
}

// stringToInterval returns a kline.Interval instance from string.
func stringToInterval(s string) (kline.Interval, error) {
	switch s {
	case "1":
		return kline.OneMin, nil
	case "3":
		return kline.ThreeMin, nil
	case "5":
		return kline.FiveMin, nil
	case "15":
		return kline.FifteenMin, nil
	case "30":
		return kline.ThirtyMin, nil
	case "60":
		return kline.OneHour, nil
	case "120":
		return kline.TwoHour, nil
	case "240":
		return kline.FourHour, nil
	case "360":
		return kline.SixHour, nil
	case "720":
		return kline.SevenHour, nil
	case "D":
		return kline.OneDay, nil
	case "M":
		return kline.OneMonth, nil
	case "W":
		return kline.OneWeek, nil
	default:
		return 0, kline.ErrInvalidInterval
	}
}

// GetKlines query for historical klines (also known as candles/candlesticks). Charts are returned in groups based on the requested interval.
func (by *Bybit) GetKlines(ctx context.Context, category, symbol string, interval kline.Interval, startTime, endTime time.Time, limit int64) ([]KlineItem, error) {
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
	err = by.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(klineInfos, params), publicSpotRate, &resp)
	if err != nil {
		return nil, err
	}
	return processKlineResponse(resp.List)
}

func processKlineResponse(in [][]string) ([]KlineItem, error) {
	klines := make([]KlineItem, len(in))
	for x := range in {
		startTimestamp, err := strconv.ParseInt(in[x][0], 10, 64)
		if err != nil {
			return nil, err
		}
		klineData := KlineItem{StartTime: time.UnixMilli(startTimestamp)}
		klineData.Open, err = strconv.ParseFloat(in[x][1], 64)
		if err != nil {
			return nil, err
		}
		klineData.High, err = strconv.ParseFloat(in[x][2], 64)
		if err != nil {
			return nil, err
		}
		klineData.Low, err = strconv.ParseFloat(in[x][3], 64)
		if err != nil {
			return nil, err
		}
		klineData.Close, err = strconv.ParseFloat(in[x][4], 64)
		if err != nil {
			return nil, err
		}
		if len(in) == 7 {
			klineData.TradeVolume, err = strconv.ParseFloat(in[x][5], 64)
			if err != nil {
				return nil, err
			}
			klineData.Turnover, err = strconv.ParseFloat(in[x][6], 64)
			if err != nil {
				return nil, err
			}
		}
	}
	return klines, nil
}

// GetInstruments retrives the list of instrument details given the category and symbol.
func (by *Bybit) GetInstruments(ctx context.Context, category, symbol, status, baseCoin, cursor string, limit int64) (*InstrumentsInfo, error) {
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
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp InstrumentsInfo
	return &resp, by.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(instrumentsInfo, params), publicSpotRate, &resp)
}

// GetMarkPriceKline query for historical mark price klines. Charts are returned in groups based on the requested interval.
func (by *Bybit) GetMarkPriceKline(ctx context.Context, category, symbol string, interval kline.Interval, startTime, endTime time.Time, limit int64) ([]KlineItem, error) {
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
	err = by.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(markPriceKline, params), publicSpotRate, &resp)
	if err != nil {
		return nil, err
	}
	return processKlineResponse(resp.List)
}

// GetIndexPriceKline query for historical index price klines. Charts are returned in groups based on the requested interval.
func (by *Bybit) GetIndexPriceKline(ctx context.Context, category, symbol string, interval kline.Interval, startTime, endTime time.Time, limit int64) ([]KlineItem, error) {
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
	err = by.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(indexPriceKline, params), publicSpotRate, &resp)
	if err != nil {
		return nil, err
	}
	return processKlineResponse(resp.List)
}

// GetOrderBook retrives for orderbook depth data.
func (by *Bybit) GetOrderBook(ctx context.Context, category, symbol string, limit int64) (*Orderbook, error) {
	params, err := fillCategoryAndSymbol(category, symbol)
	if err != nil {
		return nil, err
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp orderbookResponse
	err = by.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(marketOrderbook, params), publicSpotRate, &resp)
	if err != nil {
		return nil, err
	}
	return constructOrderbook(&resp)
}

func fillCategoryAndSymbol(category, symbol string, optionalSymbol ...bool) (url.Values, error) {
	if category == "" {
		return nil, errCategoryNotSet
	} else if category != "spot" && category != "linear" && category != "inverse" && category != "option" {
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
func (by *Bybit) GetTickers(ctx context.Context, category, symbol, baseCoin string, expiryDate time.Time) (*TickerData, error) {
	params, err := fillCategoryAndSymbol(category, symbol, true)
	if err != nil {
		return nil, err
	}
	if category == "option" && baseCoin == "" {
		return nil, errBaseNotSet
	}
	if baseCoin != "" {
		params.Set("baseCoin", baseCoin)
	}
	if !expiryDate.IsZero() {
		params.Set("expData", expiryDate.Format("02Jan06"))
	}
	var resp TickerData
	return &resp, by.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(marketTicker, params), publicSpotRate, &resp)
}

// GetFundingRateHistory retrives historical funding rates. Each symbol has a different funding interval.
// For example, if the interval is 8 hours and the current time is UTC 12, then it returns the last funding rate, which settled at UTC 8.
func (by *Bybit) GetFundingRateHistory(ctx context.Context, category, symbol string, startTime, endTime time.Time, limit int64) (*FundingRateHistory, error) {
	if category == "" {
		return nil, errCategoryNotSet
	} else if category != "linear" && category != "inverse" {
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
	var resp FundingRateHistory
	return &resp, by.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(marketFundingRateHistory, params), publicSpotRate, &resp)
}

// GetPublicTradingHistory retrives recent public trading data.
// Option type. 'Call' or 'Put'. For option only
func (by *Bybit) GetPublicTradingHistory(ctx context.Context, category, symbol, baseCoin, optionType string, limit int64) (*TradingHistory, error) {
	params, err := fillCategoryAndSymbol(category, symbol)
	if err != nil {
		return nil, err
	}
	if category == "option" && baseCoin == "" {
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
	var resp TradingHistory
	return &resp, by.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(marketRecentTrade, params), publicSpotRate, &resp)
}

// GetOpenInterest retrives open interest of each symbol.
func (by *Bybit) GetOpenInterest(ctx context.Context, category, symbol, intervalTime string, startTime, endTime time.Time, limit int64, cursor string) (*OpenInterest, error) {
	if category == "" {
		return nil, errCategoryNotSet
	} else if category != "linear" && category != "inverse" {
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
	var resp OpenInterest
	return &resp, by.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(marketOpenInterest, params), publicSpotRate, &resp)
}

// GetHistoricalValatility retrives option historical volatility.
// The data is hourly.
// If both 'startTime' and 'endTime' are not specified, it will return the most recent 1 hours worth of data.
// 'startTime' and 'endTime' are a pair of params. Either both are passed or they are not passed at all.
// This endpoint can query the last 2 years worth of data, but make sure [endTime - startTime] <= 30 days.
func (by *Bybit) GetHistoricalValatility(ctx context.Context, category, baseCoin string, period int64, startTime, endTime time.Time) ([]HistoricVolatility, error) {
	if category == "" {
		return nil, errCategoryNotSet
	} else if category != "option" {
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
	if err = common.StartEndTimeCheck(startTime, endTime); !(startTime.IsZero() && endTime.IsZero()) && err != nil {
		return nil, err
	} else if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []HistoricVolatility
	return resp, by.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(marketHistoricalVolatility, params), publicSpotRate, &resp)
}

// GetInsurance retrives insurance pool data (BTC/USDT/USDC etc). The data is updated every 24 hours.
func (by *Bybit) GetInsurance(ctx context.Context, coin string) (*InsuranceHistory, error) {
	params := url.Values{}
	if coin != "" {
		params.Set("coin", coin)
	}
	var resp InsuranceHistory
	return &resp, by.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(marketInsurance, params), publicSpotRate, &resp)
}

// GetRiskLimit retrives risk limit history
func (by *Bybit) GetRiskLimit(ctx context.Context, category, symbol string) (*RiskLimitHistory, error) {
	if category == "" {
		return nil, errCategoryNotSet
	} else if category != "linear" && category != "inverse" {
		return nil, fmt.Errorf("%w, category: %s", errInvalidCategory, category)
	}
	params := url.Values{}
	params.Set("category", category)
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp RiskLimitHistory
	return &resp, by.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(marketRiskLimit, params), publicSpotRate, &resp)
}

// GetDeliveryPrice retrives delivery price.
func (by *Bybit) GetDeliveryPrice(ctx context.Context, category, symbol, baseCoin, cursor string, limit int64) (*DeliveryPrice, error) {
	if category == "" {
		return nil, errCategoryNotSet
	} else if category != "linear" && category != "inverse" && category != "option" {
		return nil, fmt.Errorf("%w, category: %s", errInvalidCategory, category)
	}
	params := url.Values{}
	params.Set("category", category)
	if baseCoin != "" {
		params.Set("baseCoin", baseCoin)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	var resp DeliveryPrice
	return &resp, by.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(marketDeliveryPrice, params), publicSpotRate, &resp)
}

// ----------------- Trade Endpints ----------------

func isValidCategory(category string) error {
	switch category {
	case "spot", "option", "linear", "inverse":
		return nil
	case "":
		return errCategoryNotSet
	default:
		return fmt.Errorf("%w, category: %s", errInvalidCategory, category)
	}
}

// PlaceOrder creates an order for spot, spot margin, USDT perpetual, USDC perpetual, USDC futures, inverse futures and options.
func (by *Bybit) PlaceOrder(ctx context.Context, arg *PlaceOrderParams) (*OrderResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	err := isValidCategory(arg.Category)
	if err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.WhetherToBorrow {
		arg.IsLeverage = 1
	}
	// specifies whether to borrow or to trade.
	if arg.IsLeverage != 0 && arg.IsLeverage != 1 {
		return nil, errors.New("please provide a valid isLeverage value; must be 0 for unified spot and 1 for margin trading")
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.OrderType == "" { // Market and Limit order types are allowed
		return nil, order.ErrTypeIsInvalid
	}
	if arg.OrderQuantity <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	switch arg.TriggerDirection {
	case 0, 1, 2:
	default:
		return nil, fmt.Errorf("%w, triggerDirection: %d", errInvalidTriggerDirection, arg.TriggerDirection)
	}
	if arg.OrderFilter != "" && arg.Category == "spot" { //
		switch arg.OrderFilter {
		case "Order", "tpslOrder":
		default:
			return nil, fmt.Errorf("%w, orderFilter=%s", errInvalidOrderFilter, arg.OrderFilter)
		}
	}
	switch arg.TriggerPriceType {
	case "", "LastPrice", "IndexPrice", "MarkPrice":
	default:
		return nil, errInvalidTriggerPriceType
	}
	var resp OrderResponse
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, placeOrder, nil, arg, &resp, privateSpotRate)
}

// AmendOrder amends an open unfilled or partially filled orders.
func (by *Bybit) AmendOrder(ctx context.Context, arg *AmendOrderParams) (*OrderResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.OrderID == "" && arg.OrderLinkID == "" {
		return nil, errEitherOrderIDOROrderLinkIDRequired
	}
	err := isValidCategory(arg.Category)
	if err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp OrderResponse
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, amendOrder, nil, arg, &resp, privateSpotRate)
}

// CancelOrder cancels an open unfilled or partially filled order.
func (by *Bybit) CancelTradeOrder(ctx context.Context, arg *CancelOrderParams) (*OrderResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.OrderID == "" && arg.OrderLinkID == "" {
		return nil, errEitherOrderIDOROrderLinkIDRequired
	}
	err := isValidCategory(arg.Category)
	if err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.OrderFilter != "" && arg.Category == "spot" { //
		switch arg.OrderFilter {
		case "Order", "tpslOrder":
		default:
			return nil, fmt.Errorf("%w, orderFilter=%s", errInvalidOrderFilter, arg.OrderFilter)
		}
	}
	var resp *OrderResponse
	return resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, cancelOrder, nil, arg, &resp, privateSpotRate)
}

// GetOpenOrders retrives unfilled or partially filled orders in real-time. To query older order records, please use the order history interface.
func (by *Bybit) GetOpenOrders(ctx context.Context, category, symbol, baseCoin, settleCoin, orderID, orderLinkID, orderFilter, cursor string, openOnly, limit int64) (*TradeOrders, error) {
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
	var resp TradeOrders
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, openOrders, params, nil, &resp, privateSpotRate)
}

// CancelAllTradeOrders cancel all open orders
func (by *Bybit) CancelAllTradeOrders(ctx context.Context, arg *CancelAllOrdersParam) ([]OrderResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	err := isValidCategory(arg.Category)
	if err != nil {
		return nil, err
	}
	var resp CancelAllResponse
	return resp.List, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, cancelAllOrders, nil, arg, &resp, privateSpotRate)
}

// GetTradeOrderHistory retrives order history. As order creation/cancellation is asynchronous, the data returned from this endpoint may delay.
// If you want to get real-time order information, you could query this endpoint or rely on the websocket stream (recommended).
func (by *Bybit) GetTradeOrderHistory(ctx context.Context, category, symbol, orderID, orderLinkID, baseCoin, settleCoin, orderFilter, orderStatus, cursor string, startTime, endTime time.Time, limit int64) (*TradeOrders, error) {
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
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
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
	var resp TradeOrders
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, tradeOrderhistory, params, nil, &resp, privateSpotRate)
}

// PlaceBatchOrder place batch or trade order.
func (by *Bybit) PlaceBatchOrder(ctx context.Context, arg *PlaceBatchOrderParam) ([]BatchOrderResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	err := isValidCategory(arg.Category)
	if err != nil {
		return nil, err
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
			return nil, order.ErrAmountBelowMin
		}
	}
	var resp BatchOrdersList
	return resp.List, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, placeBatchTradeOrders, nil, arg, &resp, privateSpotRate)
}

// BatchAmendOrder represents a batch amend order.
func (by *Bybit) BatchAmendOrder(ctx context.Context, arg *BatchAmendOrderParams) (*BatchOrderResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	err := isValidCategory(arg.Category)
	if err != nil {
		return nil, err
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
	var resp BatchOrderResponse
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, amendBatchOrder, nil, arg, &resp, privateSpotRate)
}

// CancelBatchOrder cancel more than one open order in a single request.
func (by *Bybit) CancelBatchOrder(ctx context.Context, arg *CancelBatchOrder) ([]CancelBatchResponseItem, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	err := isValidCategory(arg.Category)
	if err != nil {
		return nil, err
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
	return resp.List, by.SendAuthHTTPRequestV5(context.Background(), exchange.RestSpot, http.MethodPost, cancelBatchOrder, nil, arg, &resp, privateSpotRate)
}

// GetBorrowQuota retrives the qty and amount of borrowable coins in spot account.
func (by *Bybit) GetBorrowQuota(ctx context.Context, category, symbol, side string) (*BorrowQuota, error) {
	if category == "" {
		return nil, errCategoryNotSet
	} else if category != "spot" {
		return nil, fmt.Errorf("%w, category: %s", errInvalidCategory, category)
	}
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	params.Set("category", category)
	if side == "" {
		return nil, order.ErrSideIsInvalid
	}
	params.Set("side", side)
	var resp BorrowQuota
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, tradeBorrowCheck, params, nil, &resp, privateSpotRate)
}

// SetDisconnectCancelAll You can use this endpoint to get your current DCP configuration.
// Your private websocket connection must subscribe "dcp" topic in order to trigger DCP successfully
func (by *Bybit) SetDisconnectCancelAll(ctx context.Context, arg *SetDCPParams) error {
	if arg == nil {
		return errNilArgument
	}
	if arg.TimeWindow == 0 {
		return errDisconnectTimeWindowNotSet
	}
	var resp interface{}
	return by.SendAuthHTTPRequestV5(context.Background(), exchange.RestSpot, http.MethodPost, "/v5/order/disconnected-cancel-all", nil, arg, &resp, privateSpotRate)
}

// -------------------------------------------------  Position Endpoints ---------------------------------------------------

// GetPositionInfo retrives real-time position data, such as position size, cumulative realizedPNL.
func (by *Bybit) GetPositionInfo(ctx context.Context, category, symbol, baseCoin, settleCoin, cursor string, limit int64) (*PositionInfoList, error) {
	if category == "" {
		return nil, errCategoryNotSet
	} else if category != "linear" && category != "inverse" && category != "option" {
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
	var resp PositionInfoList
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, getPositionList, params, nil, &resp, privateSpotRate)
}

// SetLeverage sets a leverage from 0 to max leverage of corresponding risk limit
func (by *Bybit) SetLeverage(ctx context.Context, arg *SetLeverageParams) error {
	if arg == nil {
		return errNilArgument
	}
	switch arg.Category {
	case "":
		return errCategoryNotSet
	case "linear", "inverse":
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
	var resp interface{}
	return by.SendAuthHTTPRequestV5(context.Background(), exchange.RestSpot, http.MethodPost, positionSetLeverage, nil, arg, &resp, privateSpotRate)
}

// SwitchTradeMode sets the trade mode value either to 'cross' or 'isolated'.
// Select cross margin mode or isolated margin mode per symbol level
func (by *Bybit) SwitchTradeMode(ctx context.Context, arg *SwitchTradeModeParams) error {
	if arg == nil {
		return errNilArgument
	}
	switch arg.Category {
	case "":
		return errCategoryNotSet
	case "linear", "inverse":
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
	var resp interface{}
	return by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, switchTradeMode, nil, arg, &resp, privateSpotRate)
}

// SetTakeProfitStopLossMode set partial TP/SL mode, you can set the TP/SL size smaller than position size.
func (by *Bybit) SetTakeProfitStopLossMode(ctx context.Context, arg *TPSLModeParams) (*TPSLModeResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	switch arg.Category {
	case "":
		return nil, errCategoryNotSet
	case "linear", "inverse":
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
	return resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, setTPSLMode, nil, arg, &resp, privateSpotRate)
}

// SwitchPositionMode switch the position mode for USDT perpetual and Inverse futures.
// If you are in one-way Mode, you can only open one position on Buy or Sell side.
// If you are in hedge mode, you can open both Buy and Sell side positions simultaneously.
// switches mode between MergedSingle: One-Way Mode or BothSide: Hedge Mode
func (by *Bybit) SwitchPositionMode(ctx context.Context, arg *SwitchPositionModeParams) error {
	if arg == nil {
		return errNilArgument
	}
	switch arg.Category {
	case "":
		return errCategoryNotSet
	case "linear", "inverse":
	default:
		return fmt.Errorf("%w, category: %s", errInvalidCategory, arg.Category)
	}
	if arg.Symbol.IsEmpty() && arg.Coin.IsEmpty() {
		return errEitherSymbolOrCoinRequired
	}
	var resp interface{}
	return by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, switchPositionMode, nil, arg, &resp, privateSpotRate)
}

// SetRiskLimit risk limit will limit the maximum position value you can hold under different margin requirements.
// If you want to hold a bigger position size, you need more margin. This interface can set the risk limit of a single position.
// If the order exceeds the current risk limit when placing an order, it will be rejected.
// '0': one-way mode '1': hedge-mode Buy side '2': hedge-mode Sell side
func (by *Bybit) SetRiskLimit(ctx context.Context, arg *SetRiskLimitParam) (*RiskLimitResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	switch arg.Category {
	case "":
		return nil, errCategoryNotSet
	case "linear", "inverse":
	default:
		return nil, fmt.Errorf("%w, category: %s", errInvalidCategory, arg.Category)
	}
	if arg.PositionMode < 0 || arg.PositionMode > 2 {
		return nil, errInvalidPositionMode
	}
	if arg.Symbol.IsEmpty() {
		return nil, errSymbolMissing
	}
	var resp RiskLimitResponse
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, setPositionRiskLimit, nil, arg, &resp, privateSpotRate)
}

// SetTradingStop set the take profit, stop loss or trailing stop for the position.
func (by *Bybit) SetTradingStop(ctx context.Context, arg *TradingStopParams) error {
	if arg == nil {
		return errNilArgument
	}
	switch arg.Category {
	case "":
		return errCategoryNotSet
	case "linear", "inverse":
	default:
		return fmt.Errorf("%w, category: %s", errInvalidCategory, arg.Category)
	}
	if arg.Symbol.IsEmpty() {
		return errSymbolMissing
	}
	var resp interface{}
	return by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, stopTradingPosition, nil, arg, &resp, privateSpotRate)
}

// SetAutoAddMargin sets auto add margin
func (by *Bybit) SetAutoAddMargin(ctx context.Context, arg *AutoAddMarginParams) error {
	if arg == nil {
		return errNilArgument
	}
	switch arg.Category {
	case "":
		return errCategoryNotSet
	case "linear", "inverse":
	default:
		return fmt.Errorf("%w, category: %s", errInvalidCategory, arg.Category)
	}
	if arg.Symbol.IsEmpty() {
		return errSymbolMissing
	}
	if arg.AutoAddmargin != 0 && arg.AutoAddmargin != 1 {
		return errInvalidAutoAddMarginValue
	}
	if arg.PositionMode < 0 || arg.PositionMode > 2 {
		return errInvalidPositionMode
	}
	var resp interface{}
	return by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, positionSetAutoAddMargin, nil, arg, &resp, privateSpotRate)
}

// ---------------------------------------------------------------- Old ----------------------------------------------------------------

// GetAllSpotPairs gets all pairs on the exchange
func (by *Bybit) GetAllSpotPairs(ctx context.Context) ([]PairData, error) {
	resp := struct {
		Data []PairData `json:"result"`
		Error
	}{}
	return resp.Data, by.SendHTTPRequest(ctx, exchange.RestSpot, bybitSpotGetSymbols, publicSpotRate, &resp)
}

func processOB(ob [][2]string) ([]orderbook.Item, error) {
	o := make([]orderbook.Item, len(ob))
	for x := range ob {
		var price, amount float64
		amount, err := strconv.ParseFloat(ob[x][1], 64)
		if err != nil {
			return nil, err
		}
		price, err = strconv.ParseFloat(ob[x][0], 64)
		if err != nil {
			return nil, err
		}
		o[x] = orderbook.Item{
			Price:  price,
			Amount: amount,
		}
	}
	return o, nil
}

// GetMergedOrderBook gets orderbook for a given market with a given depth (default depth 100)
func (by *Bybit) GetMergedOrderBook(ctx context.Context, symbol string, scale, depth int64) (*Orderbook, error) {
	var o orderbookResponse
	params := url.Values{}
	if scale > 0 {
		params.Set("scale", strconv.FormatInt(scale, 10))
	}

	strDepth := "100" // default depth
	if depth > 0 && depth <= 200 {
		strDepth = strconv.FormatInt(depth, 10)
	}

	params.Set("symbol", symbol)
	params.Set("limit", strDepth)
	path := common.EncodeURLValues(bybitMergedOrderBook, params)
	err := by.SendHTTPRequest(ctx, exchange.RestSpot, path, publicSpotRate, &o)
	if err != nil {
		return nil, err
	}

	return constructOrderbook(&o)
}

// GetTrades gets recent trades from the exchange
func (by *Bybit) GetTrades(ctx context.Context, symbol string, limit int64) ([]TradeItem, error) {
	resp := struct {
		Data []struct {
			Price        convert.StringToFloat64 `json:"price"`
			Time         bybitTimeMilliSec       `json:"time"`
			Quantity     convert.StringToFloat64 `json:"qty"`
			IsBuyerMaker bool                    `json:"isBuyerMaker"`
		} `json:"result"`
		Error
	}{}

	params := url.Values{}
	params.Set("symbol", symbol)

	strLimit := "60" // default limit
	if limit > 0 && limit < 60 {
		strLimit = strconv.FormatInt(limit, 10)
	}
	params.Set("limit", strLimit)
	path := common.EncodeURLValues(bybitRecentTrades, params)
	err := by.SendHTTPRequest(ctx, exchange.RestSpot, path, publicSpotRate, &resp)
	if err != nil {
		return nil, err
	}

	trades := make([]TradeItem, len(resp.Data))
	for x := range resp.Data {
		var tradeSide string
		if resp.Data[x].IsBuyerMaker {
			tradeSide = order.Buy.String()
		} else {
			tradeSide = order.Sell.String()
		}

		trades[x] = TradeItem{
			CurrencyPair: symbol,
			Price:        resp.Data[x].Price.Float64(),
			Side:         tradeSide,
			Volume:       resp.Data[x].Quantity.Float64(),
			Time:         resp.Data[x].Time.Time(),
		}
	}
	return trades, nil
}

// Get24HrsChange returns price change statistics for the last 24 hours
// If symbol not passed then it will return price change statistics for all pairs
func (by *Bybit) Get24HrsChange(ctx context.Context, symbol string) ([]PriceChangeStats, error) {
	if symbol != "" {
		resp := struct {
			Data PriceChangeStats `json:"result"`
			Error
		}{}

		params := url.Values{}
		params.Set("symbol", symbol)
		path := common.EncodeURLValues(bybit24HrsChange, params)
		err := by.SendHTTPRequest(ctx, exchange.RestSpot, path, publicSpotRate, &resp)
		if err != nil {
			return nil, err
		}
		return []PriceChangeStats{resp.Data}, nil
	}

	resp := struct {
		Data []PriceChangeStats `json:"result"`
		Error
	}{}

	err := by.SendHTTPRequest(ctx, exchange.RestSpot, bybit24HrsChange, publicSpotRate, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// GetLastTradedPrice returns last trading price
// If symbol not passed then it will return last trading price for all pairs
func (by *Bybit) GetLastTradedPrice(ctx context.Context, symbol string) ([]LastTradePrice, error) {
	var lastTradePrices []LastTradePrice
	if symbol != "" {
		resp := struct {
			Data LastTradePrice `json:"result"`
			Error
		}{}

		params := url.Values{}
		params.Set("symbol", symbol)
		path := common.EncodeURLValues(bybitLastTradedPrice, params)
		err := by.SendHTTPRequest(ctx, exchange.RestSpot, path, publicSpotRate, &resp)
		if err != nil {
			return nil, err
		}
		lastTradePrices = append(lastTradePrices, LastTradePrice{
			resp.Data.Symbol,
			resp.Data.Price,
		})
	} else {
		resp := struct {
			Data []LastTradePrice `json:"result"`
			Error
		}{}

		err := by.SendHTTPRequest(ctx, exchange.RestSpot, bybitLastTradedPrice, publicSpotRate, &resp)
		if err != nil {
			return nil, err
		}
		for x := range resp.Data {
			lastTradePrices = append(lastTradePrices, LastTradePrice{
				resp.Data[x].Symbol,
				resp.Data[x].Price,
			})
		}
	}
	return lastTradePrices, nil
}

// GetBestBidAskPrice returns best BID and ASK price
// If symbol not passed then it will return best BID and ASK price for all pairs
func (by *Bybit) GetBestBidAskPrice(ctx context.Context, symbol string) ([]TickerData, error) {
	if symbol != "" {
		resp := struct {
			Data TickerData `json:"result"`
			Error
		}{}

		params := url.Values{}
		params.Set("symbol", symbol)
		path := common.EncodeURLValues(bybitBestBidAskPrice, params)
		err := by.SendHTTPRequest(ctx, exchange.RestSpot, path, publicSpotRate, &resp)
		if err != nil {
			return nil, err
		}
		return []TickerData{resp.Data}, nil
	}

	resp := struct {
		Data []TickerData `json:"result"`
		Error
	}{}

	err := by.SendHTTPRequest(ctx, exchange.RestSpot, bybitBestBidAskPrice, publicSpotRate, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// CreatePostOrder create and post order
func (by *Bybit) CreatePostOrder(ctx context.Context, o *PlaceOrderRequest) (*OrderResponse, error) {
	if o == nil {
		return nil, errInvalidOrderRequest
	}

	params := url.Values{}
	params.Set("symbol", o.Symbol)
	params.Set("qty", strconv.FormatFloat(o.Quantity, 'f', -1, 64))
	params.Set("side", o.Side)
	params.Set("type", o.TradeType)

	if o.TimeInForce != "" {
		params.Set("timeInForce", o.TimeInForce)
	}
	if (o.TradeType == BybitRequestParamsOrderLimit || o.TradeType == BybitRequestParamsOrderLimitMaker) && o.Price == 0 {
		return nil, errMissingPrice
	}
	if o.Price != 0 {
		params.Set("price", strconv.FormatFloat(o.Price, 'f', -1, 64))
	}
	if o.OrderLinkID != "" {
		params.Set("orderLinkId", o.OrderLinkID)
	}

	resp := struct {
		Data OrderResponse `json:"result"`
		Error
	}{}
	return &resp.Data, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, bybitSpotOrder, params, nil, &resp, privateSpotRate)
}

// QueryOrder returns order data based upon orderID or orderLinkID
func (by *Bybit) QueryOrder(ctx context.Context, orderID, orderLinkID string) (*QueryOrderResponse, error) {
	if orderID == "" && orderLinkID == "" {
		return nil, errOrderOrOrderLinkIDMissing
	}

	params := url.Values{}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if orderLinkID != "" {
		params.Set("orderLinkId", orderLinkID)
	}

	resp := struct {
		Data QueryOrderResponse `json:"result"`
		Error
	}{}
	return &resp.Data, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bybitSpotOrder, params, nil, &resp, privateSpotRate)
}

// CancelExistingOrder cancels existing order based upon orderID or orderLinkID
func (by *Bybit) CancelExistingOrder(ctx context.Context, orderID, orderLinkID string) (*CancelOrderResponse, error) {
	if orderID == "" && orderLinkID == "" {
		return nil, errOrderOrOrderLinkIDMissing
	}

	params := url.Values{}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if orderLinkID != "" {
		params.Set("orderLinkId", orderLinkID)
	}

	resp := struct {
		Data CancelOrderResponse `json:"result"`
		Error
	}{}
	err := by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, bybitSpotOrder, params, nil, &resp, privateSpotRate)
	if err != nil {
		return nil, err
	}

	// In case open order is cancelled, this endpoint return status as NEW whereas if we try to cancel a already cancelled order then it's status is returned as CANCELED without any error. So this check is added to prevent this obscurity.
	if resp.Data.Status == "CANCELED" {
		return nil, fmt.Errorf("%s order already cancelled", resp.Data.OrderID)
	}
	return &resp.Data, nil
}

// FastCancelExistingOrder cancels existing order based upon orderID or orderLinkID
func (by *Bybit) FastCancelExistingOrder(ctx context.Context, symbol, orderID, orderLinkID string) (bool, error) {
	resp := struct {
		Data struct {
			IsCancelled bool `json:"isCancelled"`
		} `json:"result"`
		Error
	}{}

	if orderID == "" && orderLinkID == "" {
		return resp.Data.IsCancelled, errOrderOrOrderLinkIDMissing
	}

	params := url.Values{}
	if symbol == "" {
		return resp.Data.IsCancelled, errSymbolMissing
	}
	params.Set("symbolId", symbol)

	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if orderLinkID != "" {
		params.Set("orderLinkId", orderLinkID)
	}

	return resp.Data.IsCancelled, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, bybitFastCancelSpotOrder, params, nil, &resp, privateSpotRate)
}

// BatchCancelOrder cancels orders in batch based upon symbol, side or orderType
func (by *Bybit) BatchCancelOrder(ctx context.Context, symbol, side, orderTypes string) (bool, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if side != "" {
		params.Set("side", side)
	}
	if orderTypes != "" {
		params.Set("orderTypes", orderTypes)
	}

	resp := struct {
		Result struct {
			Success bool `json:"success"`
		} `json:"result"`
		Error
	}{}
	return resp.Result.Success, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, bybitBatchCancelSpotOrder, params, nil, &resp, privateSpotRate)
}

// BatchFastCancelOrder cancels orders in batch based upon symbol, side or orderType
func (by *Bybit) BatchFastCancelOrder(ctx context.Context, symbol, side, orderTypes string) (bool, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if side != "" {
		params.Set("side", side)
	}
	if orderTypes != "" {
		params.Set("orderTypes", orderTypes)
	}

	resp := struct {
		Result struct {
			Success bool `json:"success"`
		} `json:"result"`
		Error
	}{}
	return resp.Result.Success, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, bybitFastBatchCancelSpotOrder, params, nil, &resp, privateSpotRate)
}

// BatchCancelOrderByIDs cancels orders in batch based on comma separated order id's
func (by *Bybit) BatchCancelOrderByIDs(ctx context.Context, orderIDs []string) (bool, error) {
	params := url.Values{}
	if len(orderIDs) == 0 {
		return false, errEmptyOrderIDs
	}
	params.Set("orderIds", strings.Join(orderIDs, ","))

	resp := struct {
		Result struct {
			Success bool `json:"success"`
		} `json:"result"`
		Error
	}{}
	return resp.Result.Success, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, bybitBatchCancelByIDs, params, nil, &resp, privateSpotRate)
}

// ListOpenOrders returns all open orders
func (by *Bybit) ListOpenOrders(ctx context.Context, symbol, orderID string, limit int64) ([]QueryOrderResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}

	resp := struct {
		Data []QueryOrderResponse `json:"result"`
		Error
	}{}
	return resp.Data, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bybitOpenOrder, params, nil, &resp, privateSpotRate)
}

// GetPastOrders returns all past orders from history
func (by *Bybit) GetPastOrders(ctx context.Context, symbol, orderID string, limit int64, startTime, endTime time.Time) ([]QueryOrderResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	resp := struct {
		Data []QueryOrderResponse `json:"result"`
		Error
	}{}
	return resp.Data, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bybitPastOrder, params, nil, &resp, privateSpotRate)
}

// GetTradeHistory returns user trades
func (by *Bybit) GetTradeHistory(ctx context.Context, limit int64, symbol, fromID, toID, orderID string, startTime, endTime time.Time) ([]HistoricalTrade, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if fromID != "" {
		params.Set("fromTicketId", fromID)
	}
	if toID != "" {
		params.Set("toTicketId", toID)
	}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	resp := struct {
		Data []HistoricalTrade `json:"result"`
		Error
	}{}
	return resp.Data, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bybitTradeHistory, params, nil, &resp, privateSpotRate)
}

// GetWalletBalance returns user wallet balance
func (by *Bybit) GetWalletBalance(ctx context.Context) ([]Balance, error) {
	resp := struct {
		Data struct {
			Balances []Balance `json:"balances"`
		} `json:"result"`
		Error
	}{}
	return resp.Data.Balances, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bybitWalletBalance, url.Values{}, nil, &resp, privateSpotRate)
}

// GetSpotServerTime returns server time
func (by *Bybit) GetSpotServerTime(ctx context.Context) (time.Time, error) {
	resp := struct {
		Result struct {
			ServerTime int64 `json:"serverTime"`
		} `json:"result"`
		Error
	}{}
	err := by.SendHTTPRequest(ctx, exchange.RestSpot, bybitServerTime, publicSpotRate, &resp)
	return time.UnixMilli(resp.Result.ServerTime), err
}

// GetDepositAddressForCurrency returns deposit wallet address based upon the coin.
func (by *Bybit) GetDepositAddressForCurrency(ctx context.Context, coin string) (DepositWalletInfo, error) {
	resp := struct {
		Result DepositWalletInfo `json:"result"`
		Error
	}{}

	params := url.Values{}
	if coin == "" {
		return resp.Result, errInvalidCoin
	}
	params.Set("coin", strings.ToUpper(coin))
	return resp.Result, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, bybitGetDepositAddress, params, nil, &resp, publicSpotRate)
}

// WithdrawFund creates request for fund withdrawal.
func (by *Bybit) WithdrawFund(ctx context.Context, coin, chain, address, tag, amount string) (string, error) {
	resp := struct {
		Data struct {
			ID string `json:"id"`
		} `json:"result"`
		Error
	}{}

	params := make(map[string]interface{})
	params["coin"] = coin
	params["chain"] = chain
	params["address"] = address
	params["amount"] = amount
	if tag != "" {
		params["tag"] = tag
	}
	return resp.Data.ID, by.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, bybitWithdrawFund, nil, params, &resp, privateSpotRate)
}

// GetFeeRate returns user account fee
// Valid  category: "spot", "linear", "inverse", "option"
func (by *Bybit) GetFeeRate(ctx context.Context, category, symbol, baseCoin string) (*AccountFee, error) {
	if category == "" {
		return nil, errCategoryNotSet
	}

	if !common.StringDataContains(validCategory, category) {
		// NOTE: Opted to fail here because if the user passes in an invalid
		// category the error returned is this
		// `Bybit raw response: {"retCode":10005,"retMsg":"Permission denied, please check your API key permissions.","result":{},"retExtInfo":{},"time":1683694010783}`
		return nil, fmt.Errorf("%w, valid category values are %v", errInvalidCategory, validCategory)
	}

	params := url.Values{}
	params.Set("category", category)
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if baseCoin != "" {
		params.Set("baseCoin", baseCoin)
	}

	result := struct {
		Data *AccountFee `json:"result"`
		Error
	}{}

	err := by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, bybitAccountFee, params, nil, &result, privateFeeRate)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

// SendHTTPRequest sends an unauthenticated request
func (by *Bybit) SendHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result interface{}) error {
	endpointPath, err := by.API.Endpoints.GetURL(ePath)
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
	err = by.SendPayload(ctx, f, func() (*request.Item, error) {
		return &request.Item{
			Method:        http.MethodGet,
			Path:          endpointPath + bybitAPIVersion + path,
			Result:        response,
			Verbose:       by.Verbose,
			HTTPDebugging: by.HTTPDebugging,
			HTTPRecording: by.HTTPRecording}, nil
	}, request.UnauthenticatedRequest)
	if err != nil {
		return err
	}
	if response.RetCode != 0 && response.RetMsg != "" {
		return fmt.Errorf("code: %d message: %s", response.RetCode, response.RetMsg)
	}
	return nil
}

// SendAuthHTTPRequest sends an authenticated HTTP request
// If payload is non-nil then request is considered to be JSON
func (by *Bybit) SendAuthHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, params url.Values, jsonPayload map[string]interface{}, result UnmarshalTo, f request.EndpointLimit) error {
	creds, err := by.GetCredentials(ctx)
	if err != nil {
		return err
	}

	if result == nil {
		result = &Error{}
	}

	endpointPath, err := by.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}

	if params == nil && jsonPayload == nil {
		params = url.Values{}
	}

	if jsonPayload != nil {
		jsonPayload["recvWindow"] = defaultRecvWindow
	} else if params.Get("recvWindow") == "" {
		params.Set("recvWindow", defaultRecvWindow)
	}

	err = by.SendPayload(ctx, f, func() (*request.Item, error) {
		var (
			payload       []byte
			hmacSignedStr string
			headers       = make(map[string]string)
		)

		if jsonPayload != nil {
			headers["Content-Type"] = "application/json"
			jsonPayload["timestamp"] = strconv.FormatInt(time.Now().UnixMilli(), 10)
			jsonPayload["api_key"] = creds.Key
			hmacSignedStr, err = getJSONRequestSignature(jsonPayload, creds.Secret)
			if err != nil {
				return nil, err
			}
			jsonPayload["sign"] = hmacSignedStr
			payload, err = json.Marshal(jsonPayload)
			if err != nil {
				return nil, err
			}
		} else {
			params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
			params.Set("api_key", creds.Key)
			hmacSignedStr, err = getSign(params.Encode(), creds.Secret)
			if err != nil {
				return nil, err
			}
			headers["Content-Type"] = "application/x-www-form-urlencoded"
			switch method {
			case http.MethodPost:
				params.Set("sign", hmacSignedStr)
				payload = []byte(params.Encode())
			default:
				path = common.EncodeURLValues(path, params)
				path += "&sign=" + hmacSignedStr
			}
		}

		return &request.Item{
			Method:        method,
			Path:          endpointPath + path,
			Headers:       headers,
			Body:          bytes.NewBuffer(payload),
			Result:        &result,
			Verbose:       by.Verbose,
			HTTPDebugging: by.HTTPDebugging,
			HTTPRecording: by.HTTPRecording}, nil
	}, request.AuthenticatedRequest)
	if err != nil {
		return err
	}
	return result.GetError(true)
}

// SendAuthHTTPRequestV5 sends an authenticated HTTP request
func (by *Bybit) SendAuthHTTPRequestV5(ctx context.Context, ePath exchange.URL, method, path string, params url.Values, arg interface{}, result interface{}, f request.EndpointLimit) error {
	val := reflect.ValueOf(arg)
	if arg != nil && val.Kind() != reflect.Ptr {
		return errNonePointerArgument
	}
	val = reflect.ValueOf(result)
	if val.Kind() != reflect.Ptr {
		return errNonePointerArgument
	} else if val.IsNil() {
		return errNilArgument
	}
	creds, err := by.GetCredentials(ctx)
	if err != nil {
		return err
	}
	endpointPath, err := by.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	response := &RestResponse{
		Result: result,
	}
	err = by.SendPayload(ctx, f, func() (*request.Item, error) {
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
			Method:        method,
			Path:          endpointPath + common.EncodeURLValues(path, params),
			Headers:       headers,
			Body:          bytes.NewBuffer(payload),
			Result:        &response,
			Verbose:       by.Verbose,
			HTTPDebugging: by.HTTPDebugging,
			HTTPRecording: by.HTTPRecording,
		}, nil
	}, request.AuthenticatedRequest)
	if response.RetCode != 0 && response.RetMsg != "" {
		return fmt.Errorf("code: %d message: %s", response.RetCode, response.RetMsg)
	}
	if response.RetExtInfo != nil {
		embeddedErrors := errorMessages{}
		err := json.Unmarshal(response.RetExtInfo, &embeddedErrors)
		if err == nil {
			var errMessage string
			var failed rune
			for i := range embeddedErrors.List {
				if embeddedErrors.List[i].Code != 0 {
					failed |= 1
					errMessage += fmt.Sprintf("code: %d message: %s ", embeddedErrors.List[i].Code, embeddedErrors.List[i].Message)
				}
			}
			if failed > 0 {
				return errors.New(errMessage)
			}
		}
	}
	return err
}

// Error defines all error information for each request
type Error struct {
	ReturnCode      int64  `json:"ret_code"`
	ReturnMsg       string `json:"ret_msg"`
	ReturnCodeV5    int64  `json:"retCode"`
	ReturnMessageV5 string `json:"retMsg"`
	ExtCode         string `json:"ext_code"`
	ExtMsg          string `json:"ext_info"`
}

// GetError checks and returns an error if it is supplied.
func (e *Error) GetError(isAuthRequest bool) error {
	if e.ReturnCode != 0 && e.ReturnMsg != "" {
		if isAuthRequest {
			return fmt.Errorf("%w %v", request.ErrAuthRequestFailed, e.ReturnMsg)
		}
		return errors.New(e.ReturnMsg)
	}
	if e.ReturnCodeV5 != 0 && e.ReturnMessageV5 != "" {
		return errors.New(e.ReturnMessageV5)
	}
	if e.ExtCode != "" && e.ExtMsg != "" {
		if isAuthRequest {
			return fmt.Errorf("%w %v", request.ErrAuthRequestFailed, e.ExtMsg)
		}
		return errors.New(e.ExtMsg)
	}
	return nil
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

func getTradeType(tradeType string) order.Type {
	switch tradeType {
	case BybitRequestParamsOrderLimit:
		return order.Limit
	case BybitRequestParamsOrderMarket:
		return order.Market
	case BybitRequestParamsOrderLimitMaker:
		return order.Limit
	default:
		return order.UnknownType
	}
}

func getOrderStatus(status string) order.Status {
	switch status {
	case "NEW":
		return order.New
	case "PARTIALLY_FILLED":
		return order.PartiallyFilled
	case "FILLED":
		return order.Filled
	case "CANCELED":
		return order.Cancelled
	case "PENDING_CANCEL":
		return order.PendingCancel
	case "PENDING_NEW":
		return order.Pending
	case "REJECTED":
		return order.Rejected
	default:
		return order.UnknownStatus
	}
}

func getJSONRequestSignature(payload map[string]interface{}, secret string) (string, error) {
	payloadArr := make([]string, len(payload))
	var i int
	for p := range payload {
		payloadArr[i] = p
		i++
	}
	sort.Strings(payloadArr)
	var signStr string
	for _, key := range payloadArr {
		if value, found := payload[key]; found {
			if v, ok := value.(string); ok {
				signStr += key + "=" + v + "&"
			}
		} else {
			return "", errors.New("non-string payload parameter not expected")
		}
	}
	return getSign(signStr[:len(signStr)-1], secret)
}

func getSign(sign, secret string) (string, error) {
	hmacSigned, err := crypto.GetHMAC(crypto.HashSHA256, []byte(sign), []byte(secret))
	if err != nil {
		return "", err
	}
	return crypto.HexEncodeToString(hmacSigned), nil
}
