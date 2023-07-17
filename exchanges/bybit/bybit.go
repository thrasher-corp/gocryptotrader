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
	addOrRemoveMargin        = "/v5/position/add-margin"
	positionExecutionList    = "/v5/execution/list"
	positionClosedPNL        = "/v5/position/closed-pnl"

	// Pre-Upgrade endpoints
	preUpgradeOrderHistory          = "/v5/pre-upgrade/order/history"
	preUpgradeExecutionList         = "/v5/pre-upgrade/execution/list"
	preUpgradePositionClosedPNL     = "/v5/pre-upgrade/position/closed-pnl"
	preUpgradeAccountTransactionLog = "/v5/pre-upgrade/account/transaction-log"
	preUpgradeAssetDeliveryRecord   = "/v5/pre-upgrade/asset/delivery-record"
	preUpgradeAssetSettlementRecord = "/v5/pre-upgrade/asset/settlement-record"
	accountWalletBalanceRequired    = "/v5/account/wallet-balance"
	accountUpgradeToUTA             = "/v5/account/upgrade-to-uta"
)

var (
	errCategoryNotSet                     = errors.New("category not set")
	errBaseNotSet                         = errors.New("base coin not set when category is option")
	errInvalidTriggerDirection            = errors.New("invalid trigger direction")
	errInvalidTriggerPriceType            = errors.New("invalid trigger price type")
	errNilArgument                        = errors.New("nil argument")
	errMissingUserID                      = errors.New("sub user id missing")
	errMissingusername                    = errors.New("username is missing")
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
func (by *Bybit) GetTradeOrderHistory(ctx context.Context, category, symbol, orderID, orderLinkID,
	baseCoin, settleCoin, orderFilter, orderStatus, cursor string,
	startTime, endTime time.Time, limit int64) (*TradeOrders, error) {
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
func (by *Bybit) SetAutoAddMargin(ctx context.Context, arg *AddRemoveMarginParams) error {
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

// AddOrReduceMargin manually add or reduce margin for isolated margin position
func (by *Bybit) AddOrReduceMargin(ctx context.Context, arg *AddRemoveMarginParams) (*AddOrReduceMargin, error) {
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
	if arg.Symbol.IsEmpty() {
		return nil, errSymbolMissing
	}
	if arg.AutoAddmargin != 0 && arg.AutoAddmargin != 1 {
		return nil, errInvalidAutoAddMarginValue
	}
	if arg.PositionMode < 0 || arg.PositionMode > 2 {
		return nil, errInvalidPositionMode
	}
	var resp AddOrReduceMargin
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, addOrRemoveMargin, nil, arg, &resp, privateSpotRate)
}

// GetExecution retrives users' execution records, sorted by execTime in descending order. However, for Normal spot, they are sorted by execId in descending order.
func (by *Bybit) GetExecution(ctx context.Context, category, symbol, orderID, orderLinkID, baseCoin, cursor string, startTime, endTime time.Time, limit int64) (*ExecutionResponse, error) {
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
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp ExecutionResponse
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, positionExecutionList, params, nil, &resp, privateSpotRate)
}

// GetClosedPnL retrives user's closed profit and loss records. The results are sorted by createdTime in descending order.
func (by *Bybit) GetClosedPnL(ctx context.Context, category, symbol, cursor string, startTime, endTime time.Time, limit int64) (*ClosedProfitAndLossResponse, error) {
	params, err := fillOrderAndExecutionFetchParams(paramsConfig{Linear: true, Inverse: true}, category, symbol, "", "", "", "", "", cursor, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	var resp ClosedProfitAndLossResponse
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, positionClosedPNL, params, nil, &resp, privateSpotRate)
}

// ---------------------------------------------------------------- Pre-Upgrade ----------------------------------------------------------------

func fillOrderAndExecutionFetchParams(ac paramsConfig, category, symbol, baseCoin, orderID, orderLinkID, orderFilter, orderStatus, cursor string, startTime, endTime time.Time, limit int64) (url.Values, error) {
	params := url.Values{}
	switch {
	case !ac.OptionalCategory && category == "":
		return nil, errCategoryNotSet
	case ac.OptionalCategory && category == "":
	case (!ac.Linear && category == "linear") ||
		(!ac.Option && category == "option") ||
		(!ac.Inverse && category == "inverse") ||
		(!ac.Spot && category == "spot"):
		return nil, fmt.Errorf("%w, category: %s", errInvalidCategory, category)
	default:
		params.Set("category", category)
	}
	if ac.MendatorySymbol && symbol == "" {
		return nil, errSymbolMissing
	} else if symbol != "" {
		params.Set("symbol", symbol)
	}
	if category == "option" && baseCoin == "" && !ac.OptionalBaseCoin {
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

// GetPreUpgradeOrderHistory the account is upgraded to a Unified account, you can get the orders which occurred before the upgrade.
func (by *Bybit) GetPreUpgradeOrderHistory(ctx context.Context, category, symbol, baseCoin, orderID, orderLinkID, orderFilter, orderStatus, cursor string, startTime, endTime time.Time, limit int64) (*TradeOrders, error) {
	params, err := fillOrderAndExecutionFetchParams(paramsConfig{Linear: true, Option: true, Inverse: true}, category, symbol, baseCoin, orderID, orderLinkID, orderFilter, orderStatus, cursor, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	var resp TradeOrders
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, preUpgradeOrderHistory, params, nil, &resp, privateSpotRate)
}

// GetPreUpgradeTradeHistory retrieves users' execution records which occurred before you upgraded the account to a Unified account, sorted by execTime in descending order
func (by *Bybit) GetPreUpgradeTradeHistory(ctx context.Context, category, symbol, orderID, orderLinkID, baseCoin, executionType, cursor string, startTime, endTime time.Time, limit int64) (*ExecutionResponse, error) {
	params, err := fillOrderAndExecutionFetchParams(paramsConfig{Linear: true, Option: false, Inverse: true}, category, symbol, baseCoin, orderID, orderLinkID, "", "", cursor, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	if executionType != "" {
		params.Set("executionType", executionType)
	}
	var resp ExecutionResponse
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, preUpgradeExecutionList, params, nil, &resp, privateSpotRate)
}

// GetPreUpgradeClosedPnL retrives user's closed profit and loss records from before you upgraded the account to a Unified account. The results are sorted by createdTime in descending order.
func (by *Bybit) GetPreUpgradeClosedPnL(ctx context.Context, category, symbol, cursor string, startTime, endTime time.Time, limit int64) (*ClosedProfitAndLossResponse, error) {
	params, err := fillOrderAndExecutionFetchParams(paramsConfig{Linear: true, Inverse: true, MendatorySymbol: true}, category, symbol, "", "", "", "", "", cursor, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	var resp ClosedProfitAndLossResponse
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, preUpgradePositionClosedPNL, params, nil, &resp, privateSpotRate)
}

// GetPreUpgradeTransactionLog retrives transaction logs which occurred in the USDC Derivatives wallet before the account was upgraded to a Unified account.
func (by *Bybit) GetPreUpgradeTransactionLog(ctx context.Context, category, baseCoin, transactionType, cursor string, startTime, endTime time.Time, limit int64) (*TransactionLog, error) {
	params, err := fillOrderAndExecutionFetchParams(paramsConfig{Linear: true, Inverse: true}, category, "", baseCoin, "", "", "", "", cursor, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	if transactionType != "" {
		params.Set("type", transactionType)
	}
	var resp TransactionLog
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, preUpgradeAccountTransactionLog, params, nil, &resp, privateSpotRate)
}

// GetPreUpgradeOptionDeliveryRecord retrives delivery records of Option before you upgraded the account to a Unified account, sorted by deliveryTime in descending order
func (by *Bybit) GetPreUpgradeOptionDeliveryRecord(ctx context.Context, category, symbol, cursor string, expiryDate time.Time, limit int64) (*PreUpdateOptionDeliveryRecord, error) {
	params, err := fillOrderAndExecutionFetchParams(paramsConfig{OptionalBaseCoin: true, Option: true}, category, symbol, "", "", "", "", "", cursor, time.Time{}, time.Time{}, limit)
	if err != nil {
		return nil, err
	}
	if !expiryDate.IsZero() {
		params.Set("expData", expiryDate.Format("02Jan06"))
	}
	var resp PreUpdateOptionDeliveryRecord
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, preUpgradeAssetDeliveryRecord, params, nil, &resp, privateSpotRate)
}

// GetPreUpgradeUSDCSessionSettlement retrives session settlement records of USDC perpetual before you upgrade the account to Unified account.
func (by *Bybit) GetPreUpgradeUSDCSessionSettlement(ctx context.Context, category, symbol, cursor string, limit int64) (*SettlementSession, error) {
	params, err := fillOrderAndExecutionFetchParams(paramsConfig{Linear: true}, category, symbol, "", "", "", "", "", cursor, time.Time{}, time.Time{}, limit)
	if err != nil {
		return nil, err
	}
	var resp SettlementSession
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, preUpgradeAssetSettlementRecord, params, nil, &resp, privateSpotRate)
}

// ---------------------------------------------------------------- Account Endpoints ----------------------------------------------------------------

// GetWalletBalance represents wallet balance, query asset information of each currency, and account risk rate information.
// By default, currency information with assets or liabilities of 0 is not returned.
// Unified account: UNIFIED (trade spot/linear/options), CONTRACT(trade inverse)
// Normal account: CONTRACT, SPOT
func (by *Bybit) GetWalletBalance(ctx context.Context, accountType, coin string) (*WalletBalance, error) {
	params := url.Values{}
	if accountType == "" {
		return nil, errMissingAccountType
	} else {
		params.Set("accountType", accountType)
	}
	if coin != "" {
		params.Set("coin", coin)
	}
	var resp WalletBalance
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, accountWalletBalanceRequired, params, nil, &resp, privateSpotRate)
}

// UpgradeToUnifiedAccount upgrades the account to unified account.
func (by *Bybit) UpgradeToUnifiedAccount(ctx context.Context) (*UnifiedAccountUpgradeResponse, error) {
	var resp UnifiedAccountUpgradeResponse
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, accountUpgradeToUTA, nil, nil, &resp, privateSpotRate)
}

// GetBorrowHistory retrives interest records, sorted in reverse order of creation time.
func (by *Bybit) GetBorrowHistory(ctx context.Context, currency, cursor string, startTime, endTime time.Time, limit int64) (*BorrowHistory, error) {
	params := url.Values{}
	if currency != "" {
		params.Set("currency", currency)
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
	var resp BorrowHistory
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/borrow-history", params, nil, &resp, privateSpotRate)
}

// GetCollateralInfo retrives the collateral information of the current unified margin account,
// including loan interest rate, loanable amount, collateral conversion rate,
// whether it can be mortgaged as margin, etc.
func (by *Bybit) GetCollateralInfo(ctx context.Context, currency string) (*CollateralInfo, error) {
	params := url.Values{}
	if currency != "" {
		params.Set("currency", currency)
	}
	var resp CollateralInfo
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/collateral-info", params, nil, &resp, privateSpotRate)
}

// GetCoinGreeks retrives current account Greeks information
func (by *Bybit) GetCoinGreeks(ctx context.Context, baseCoin string) (*CoinGreeks, error) {
	params := url.Values{}
	if baseCoin != "" {
		params.Set("baseCoin", baseCoin)
	}
	var resp CoinGreeks
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/coin-greeks", params, nil, &resp, privateSpotRate)
}

// GetFeeRate retrives the trading fee rate.
func (by *Bybit) GetFeeRate(ctx context.Context, category, symbol, baseCoin string) ([]Fee, error) {
	params := url.Values{}
	if !common.StringDataContains(validCategory, category) {
		// NOTE: Opted to fail here because if the user passes in an invalid
		// category the error returned is this
		// `Bybit raw response: {"retCode":10005,"retMsg":"Permission denied, please check your API key permissions.","result":{},"retExtInfo":{},"time":1683694010783}`
		return nil, fmt.Errorf("%w, valid category values are %v", errInvalidCategory, validCategory)
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if baseCoin != "" {
		params.Set("baseCoin", baseCoin)
	}
	var resp AccountFee
	return resp.List, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/fee-rate", params, nil, &resp, privateSpotRate)
}

// GetAccountInfo retrieves the margin mode configuration of the account.
func (by *Bybit) GetAccountInfo(ctx context.Context) (*AccountInfo, error) {
	var resp AccountInfo
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/info", nil, nil, &resp, privateSpotRate)
}

// GetTransactionLog retrieves transaction logs in Unified account.
func (by *Bybit) GetTransactionLog(ctx context.Context, category, baseCoin, transactionType, cursor string, startTime, endTime time.Time, limit int64) (*TransactionLog, error) {
	params, err := fillOrderAndExecutionFetchParams(paramsConfig{OptionalBaseCoin: true, OptionalCategory: true, Linear: true, Option: true, Spot: true}, category, "", baseCoin, "", "", "", "", cursor, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	if transactionType != "" {
		params.Set("type", transactionType)
	}
	var resp TransactionLog
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/account/transaction-log", params, nil, &resp, privateSpotRate)
}

// SetMarginMode set margin mode to  either of ISOLATED_MARGIN, REGULAR_MARGIN(i.e. Cross margin), PORTFOLIO_MARGIN
func (by *Bybit) SetMarginMode(ctx context.Context, marginMode string) (*SetMarginModeResponse, error) {
	if marginMode == "" {
		return nil, fmt.Errorf("%w, margin mode should be either of ISOLATED_MARGIN, REGULAR_MARGIN, or PORTFOLIO_MARGIN", errInvalidMode)
	}
	arg := &struct {
		SetMarginMode string `json:"setMarginMode"`
	}{SetMarginMode: marginMode}

	var resp SetMarginModeResponse
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/set-margin-mode", nil, arg, &resp, privateSpotRate)
}

// SetMMP Market Maker Protection (MMP) is an automated mechanism designed to protect market makers (MM) against liquidity risks and over-exposure in the market.
// It prevents simultaneous trade executions on quotes provided by the MM within a short time span.
// The MM can automatically pull their quotes if the number of contracts traded for an underlying asset exceeds the configured threshold within a certain time frame.
// Once MMP is triggered, any pre-existing MMP orders will be automatically canceled, and new orders tagged as MMP will be rejected for a specific duration — known as the frozen period — so that MM can reassess the market and modify the quotes.
//
// How to enable MMP​
// Send an email to Bybit (financial.inst@bybit.com) or contact your business development (BD) manager to apply for MMP. After processed, the default settings are as below table:
func (by *Bybit) SetMMP(ctx context.Context, arg *MMPRequestParam) error {
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
	var resp interface{}
	return by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/mmp-modify", nil, arg, &resp, privateSpotRate)
}

// ResetMMP resets MMP.
// once the mmp triggered, you can unfreeze the account by this endpoint
func (by *Bybit) ResetMMP(ctx context.Context, baseCoin string) error {
	if baseCoin == "" {
		return errBaseNotSet
	}
	arg := &struct {
		BaseCoin string `json:"baseCoin"`
	}{BaseCoin: baseCoin}
	var resp interface{}
	return by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/mmp-reset", nil, arg, &resp, privateSpotRate)
}

// GetMMPState retrive Market Maker Protection (MMP) states for different coins.
func (by *Bybit) GetMMPState(ctx context.Context, baseCoin string) (*MMPStates, error) {
	if baseCoin == "" {
		return nil, errBaseNotSet
	}
	arg := &struct {
		BaseCoin string `json:"baseCoin"`
	}{BaseCoin: baseCoin}
	var resp MMPStates
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/account/mmp-state", nil, arg, &resp, privateSpotRate)
}

// ---------------------------------------------------------------- Assets ----------------------------------------------------------------

// GetCoinExchangeRecords queries the coin exchange records.
func (by *Bybit) GetCoinExchangeRecords(ctx context.Context, fromCoin, toCoin, cursor string, limit int64) (*CoinExchangeRecords, error) {
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
	var resp CoinExchangeRecords
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/exchange/order-record", params, nil, &resp, privateSpotRate)
}

// GetDeliveryRecord retrieves delivery records of USDC futures and Options, sorted by deliveryTime in descending order
func (by *Bybit) GetDeliveryRecord(ctx context.Context, category, symbol, cursor string, expiryDate time.Time, limit int64) (*DeliveryRecord, error) {
	validCategory = []string{"linear", "option"}
	if !common.StringDataContains(validCategory, category) {
		return nil, fmt.Errorf("%w, valid category values are %v", errInvalidCategory, validCategory)
	}
	params := url.Values{}
	params.Set("category", category)
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if !expiryDate.IsZero() {
		params.Set("expData", expiryDate.Format("02Jan06"))
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp DeliveryRecord
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/delivery-record", params, nil, &resp, privateSpotRate)
}

// GetUSDCSessionSettlement retrieves session settlement records of USDC perpetual and futures
func (by *Bybit) GetUSDCSessionSettlement(ctx context.Context, category, symbol, cursor string, limit int64) (*SettlementSession, error) {
	validCategory = []string{"linear"}
	if !common.StringDataContains(validCategory, category) {
		return nil, fmt.Errorf("%w, valid category values are %v", errInvalidCategory, validCategory)
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
	var resp SettlementSession
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/settlement-record", params, nil, &resp, privateSpotRate)
}

// GetSpotAccountInfo retrieves asset information
func (by *Bybit) GetAssetInfo(ctx context.Context, accountType, coin string) (AccountInfos, error) {
	if accountType == "" {
		return nil, errMissingAccountType
	}
	params := url.Values{}
	params.Set("accountType", accountType)
	if coin != "" {
		params.Set("coin", coin)
	}
	var resp AccountInfos
	return resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/transfer/query-asset-info", params, nil, &resp, privateSpotRate)
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
func (by *Bybit) GetAllCoinBalance(ctx context.Context, accountType, memberID, coin string, withBonus int64) (*CoinBalances, error) {
	params, err := fillCoinBalanceFetchParams(accountType, memberID, coin, withBonus, false)
	if err != nil {
		return nil, err
	}
	var resp CoinBalances
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/transfer/query-account-coins-balance", params, nil, &resp, privateSpotRate)
}

// GetSingleCoinBalance retrieves the balance of a specific coin in a specific account type. Supports querying sub UID's balance.
func (by *Bybit) GetSingleCoinBalance(ctx context.Context, accountType, coin, memberID string, withBonus, withTransferSafeAmount int64) (*CoinBalances, error) {
	params, err := fillCoinBalanceFetchParams(accountType, memberID, coin, withBonus, true)
	if err != nil {
		return nil, err
	}
	if withTransferSafeAmount > 0 {
		params.Set("withTransferSafeAmount", strconv.FormatInt(withTransferSafeAmount, 10))
	}
	var resp CoinBalances
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/transfer/query-account-coin-balance", params, nil, &resp, privateSpotRate)
}

// GetTransferableCoin the transferable coin list between each account type
func (by *Bybit) GetTransferableCoin(ctx context.Context, fromAccountType, toAccountType string) (*TransferableCoins, error) {
	if fromAccountType == "" {
		return nil, fmt.Errorf("%w, from account type not specified", errMissingAccountType)
	}
	if toAccountType == "" {
		return nil, fmt.Errorf("%w, to account type not specified", errMissingAccountType)
	}
	params := url.Values{}
	params.Set("fromAccountType", fromAccountType)
	params.Set("toAccountType", toAccountType)
	var resp TransferableCoins
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/transfer/query-transfer-coin-list", params, nil, &resp, privateSpotRate)
}

// CreateInternalTransfer create the internal transfer between different account types under the same UID.
// Each account type has its own acceptable coins, e.g, you cannot transfer USDC from SPOT to CONTRACT.
// Please refer to transferable coin list API to find out more.
func (by *Bybit) CreateInternalTransfer(ctx context.Context, arg *TransferParams) (string, error) {
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
	resp := &struct {
		TransferID string `json:"transferId"`
	}{}
	return resp.TransferID, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/transfer/inter-transfer", nil, arg, &resp, privateSpotRate)
}

// GetInternalTransferRecords retrieves the internal transfer records between different account types under the same UID.
func (by *Bybit) GetInternalTransferRecords(ctx context.Context, transferID, coin, status, cursor string, startTime, endTime time.Time, limit int64) (*TransferResponse, error) {
	params, err := fillTransferQueryParams(transferID, coin, status, cursor, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	var resp TransferResponse
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/transfer/query-inter-transfer-list", params, nil, &resp, privateSpotRate)
}

// GetSubUID retrieves the sub UIDs under a main UID
func (by *Bybit) GetSubUID(ctx context.Context) (*SubUID, error) {
	var resp SubUID
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/transfer/query-sub-member-list", nil, nil, &resp, privateSpotRate)
}

// EnableUniversalTransferForSubUID Transfer between sub-sub or main-sub
// Use this endpoint to enable a subaccount to take part in a universal transfer. It is a one-time switch which, once thrown, enables a subaccount permanently.
// If not set, your subaccount cannot use universal transfers.
func (by *Bybit) EnableUniversalTransferForSubUID(ctx context.Context, subMemberIDS ...string) error {
	if len(subMemberIDS) == 0 {
		return errMembersIDsNotSet
	}
	arg := map[string][]string{
		"subMemberIds": subMemberIDS,
	}
	var resp interface{}
	return by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/transfer/save-transfer-sub-member", nil, &arg, &resp, privateSpotRate)
}

// CreateUniversalTransfer transfer between sub-sub or main-sub. Please make sure you have enabled universal transfer on your sub UID in advance.
// To use sub acct api key, it must have "SubMemberTransferList" permission
// When use sub acct api key, it can only transfer to main account
// You can not transfer between the same UID
func (by *Bybit) CreateUniversalTransfer(ctx context.Context, arg *TransferParams) (string, error) {
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
	resp := &struct {
		TransferID string `json:"transferId"`
	}{}
	return resp.TransferID, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/transfer/universal-transfer", nil, arg, &resp, privateSpotRate)
}

func fillTransferQueryParams(transferID, coin, status, cursor string, startTime, endTime time.Time, limit int64) (url.Values, error) {
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
	return params, nil
}

// GetUniversalTransferRecords query universal transfer records
// Main acct api key or Sub acct api key are both supported
// Main acct api key needs "SubMemberTransfer" permission
// Sub acct api key needs "SubMemberTransferList" permission
func (by *Bybit) GetUniversalTransferRecords(ctx context.Context, transferID, coin, status, cursor string, startTime, endTime time.Time, limit int64) (*TransferResponse, error) {
	params, err := fillTransferQueryParams(transferID, coin, status, cursor, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	var resp TransferResponse
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/transfer/query-inter-transfer-list", params, nil, &resp, privateSpotRate)
}

// GetAllowedDepositCoinInfo retrieves allowed deposit coin information. To find out paired chain of coin, please refer coin info api.
func (by *Bybit) GetAllowedDepositCoinInfo(ctx context.Context, coin, chain, cursor string, limit int64) (*AllowedDepositCoinInfo, error) {
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
	var resp AllowedDepositCoinInfo
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/deposit/query-allowed-list", params, nil, &resp, privateSpotRate)
}

// SetDepositAccount sets auto transfer account after deposit. The same function as the setting for Deposit on web GUI
func (by *Bybit) SetDepositAccount(ctx context.Context, accountType string) (*StatusResponse, error) {
	if accountType == "" {
		return nil, errMissingAccountType
	}
	arg := &struct {
		AccountType string `json:"accountType"`
	}{
		AccountType: accountType,
	}
	var resp StatusResponse
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/deposit/deposit-to-account", nil, &arg, &resp, privateSpotRate)
}

func fillDepositRecordsParams(coin, cursor string, startTime, endTime time.Time, limit int64) (url.Values, error) {
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
	return params, nil
}

// GetDepositRecords query deposit records.
func (by *Bybit) GetDepositRecords(ctx context.Context, coin, cursor string, startTime, endTime time.Time, limit int64) (*DepositRecords, error) {
	params, err := fillDepositRecordsParams(coin, cursor, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	var resp DepositRecords
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/deposit/query-record", params, nil, &resp, privateSpotRate)
}

// GetSubDepositRecords query subaccount's deposit records by main UID's API key. on chain
func (by *Bybit) GetSubDepositRecords(ctx context.Context, subMemberID, coin, cursor string, startTime, endTime time.Time, limit int64) (*DepositRecords, error) {
	if subMemberID == "" {
		return nil, errMembersIDsNotSet
	}
	params, err := fillDepositRecordsParams(coin, cursor, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	params.Set("subMemberId", subMemberID)
	var resp DepositRecords
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/deposit/query-sub-member-record", params, nil, &resp, privateSpotRate)
}

// GetInternalDepositRecordsOffChain retrieves deposit records within the Bybit platform. These transactions are not on the blockchain.
func (by *Bybit) GetInternalDepositRecordsOffChain(ctx context.Context, coin, cursor string, startTime, endTime time.Time, limit int64) (*InternalDepositRecords, error) {
	params, err := fillDepositRecordsParams(coin, cursor, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	var resp InternalDepositRecords
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/deposit/query-internal-record", params, nil, &resp, privateSpotRate)
}

// GetMasterDepositAddress retrieves the deposit address information of MASTER account.
func (by *Bybit) GetMasterDepositAddress(ctx context.Context, coin currency.Code, chainType string) (*DepositAddresses, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", coin.String())
	if chainType != "" {
		params.Set("chainType", chainType)
	}
	var resp DepositAddresses
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/deposit/query-address", params, nil, &resp, privateSpotRate)
}

// GetSubDepositAddress retrieves the deposit address information of SUB account.
func (by *Bybit) GetSubDepositAddress(ctx context.Context, coin currency.Code, chainType, subMemberID string) (*DepositAddresses, error) {
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
	var resp DepositAddresses
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/deposit/query-sub-member-address", params, nil, &resp, privateSpotRate)
}

// GetCoinInfo retrieves coin information, including chain information, withdraw and deposit status.
func (by *Bybit) GetCoinInfo(ctx context.Context, coin currency.Code) (*CoinInfo, error) {
	params := url.Values{}
	if coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	var resp CoinInfo
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/coin/query-info", params, nil, &resp, privateSpotRate)
}

// GetWithdrawalRecords query withdrawal records.
// endTime - startTime should be less than 30 days. Query last 30 days records by default.
// Can query by the master UID's api key only
func (by *Bybit) GetWithdrawalRecords(ctx context.Context, coin currency.Code, withdrawalID, withdrawType, cursor string, startTime, endTime time.Time, limit int64) (*WithdrawalRecords, error) {
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
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp WithdrawalRecords
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/withdraw/query-record", params, nil, &resp, privateSpotRate)
}

// GetWithdrawableAmount retrieves withdrawable amount information using currency code
func (by *Bybit) GetWithdrawableAmount(ctx context.Context, coin currency.Code) (*WithdrawableAmount, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", coin.String())
	var resp WithdrawableAmount
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/asset/withdraw/withdrawable-amount", params, nil, &resp, privateSpotRate)
}

// WithdrawCurrency Withdraw assets from your Bybit account. You can make an off-chain transfer if the target wallet address is from Bybit. This means that no blockchain fee will be charged.
func (by *Bybit) WithdrawCurrency(ctx context.Context, arg *WithdrawalParam) (string, error) {
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
		return "", order.ErrAmountBelowMin
	}
	if arg.Timestamp == 0 {
		arg.Timestamp = time.Now().UnixMilli()
	}
	resp := &struct {
		ID string `json:"id"`
	}{}
	return resp.ID, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/withdraw/create", nil, arg, &resp, privateSpotRate)
}

// CancelWithdrawal cancel the withdrawal
func (by *Bybit) CancelWithdrawal(ctx context.Context, id string) (*StatusResponse, error) {
	if id == "" {
		return nil, errMissingWithdrawalID
	}
	arg := &struct {
		ID string `json:"id"`
	}{
		ID: id,
	}
	var resp StatusResponse
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/asset/withdraw/cancel", nil, arg, &resp, privateSpotRate)
}

// -------------------------- User endpoints --------------------

// CreateNewSubUserID created a new sub user id. Use master user's api key only.
func (by *Bybit) CreateNewSubUserID(ctx context.Context, arg *CreateSubUserParams) (*SubUserItem, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.Username == "" {
		return nil, errMissingusername
	}
	if arg.MemberType <= 0 {
		return nil, errInvalidMemberType
	}
	var resp SubUserItem
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/create-sub-member", nil, &arg, &resp, privateSpotRate)
}

// CreateSubUIDAPIKey create new API key for those newly created sub UID. Use master user's api key only.
func (by *Bybit) CreateSubUIDAPIKey(ctx context.Context, arg *SubUIDAPIKeyParam) (*SubUIDAPIResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.Subuid <= 0 {
		return nil, fmt.Errorf("%w, subuid", errMissingUserID)
	}
	var resp SubUIDAPIResponse
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/query-sub-members", nil, arg, &resp, privateSpotRate)
}

// GetSubUIDList get all sub uid of master account. Use master user's api key only.
func (by *Bybit) GetSubUIDList(ctx context.Context) ([]SubUserItem, error) {
	resp := &struct {
		SubMembers []SubUserItem `json:"subMembers"`
	}{}
	return resp.SubMembers, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/user/query-sub-members", nil, nil, &resp, privateSpotRate)
}

// FreezeSubUID freeze Sub UID. Use master user's api key only.
func (by *Bybit) FreezeSubUID(ctx context.Context, subUID string, frozen bool) error {
	params := url.Values{}
	if subUID == "" {
		return fmt.Errorf("%w, subuid", errMissingUserID)
	}
	params.Set("subuid", subUID)
	if frozen {
		params.Set("frozen", "0")
	} else {
		params.Set("frozen", "1")
	}
	var resp interface{}
	return by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/user/frozen-sub-member", params, nil, &resp, privateSpotRate)
}

// GetAPIKeyInformation retrieves the information of the api key.
// Use the api key pending to be checked to call the endpoint.
// Both master and sub user's api key are applicable.
func (by *Bybit) GetAPIKeyInformation(ctx context.Context) (*SubUIDAPIResponse, error) {
	var resp SubUIDAPIResponse
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/user/query-api", nil, nil, &resp, privateSpotRate)
}

// GetUIDWalletType retrieves available wallet types for the master account or sub account
func (by *Bybit) GetUIDWalletType(ctx context.Context, memberIDS string) (*WalletType, error) {
	if memberIDS == "" {
		return nil, errMembersIDsNotSet
	}
	var resp WalletType
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/user/get-member-type", nil, nil, &resp, privateSpotRate)
}

// ModifyMasterAPIKey modify the settings of master api key.
// Use the api key pending to be modified to call the endpoint. Use master user's api key only.
func (by *Bybit) ModifyMasterAPIKey(ctx context.Context, arg *SubUIDAPIKeyUpdateParam) (*SubUIDAPIResponse, error) {
	if arg == nil || reflect.DeepEqual(*arg, SubUIDAPIKeyUpdateParam{}) {
		return nil, errNilArgument
	}
	var resp SubUIDAPIResponse
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/update-api", nil, arg, &resp, privateSpotRate)
}

// ModifySubAPIKey modifies the settings of sub api key. Use the api key pending to be modified to call the endpoint. Use sub user's api key only.
func (by *Bybit) ModifySubAPIKey(ctx context.Context, arg *SubUIDAPIKeyUpdateParam) (*SubUIDAPIResponse, error) {
	if arg == nil || reflect.DeepEqual(*arg, SubUIDAPIKeyUpdateParam{}) {
		return nil, errNilArgument
	}
	var resp SubUIDAPIResponse
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/update-sub-api", nil, &arg, &resp, privateSpotRate)
}

// DeleteMasterAPIKey delete the api key of master account.
// Use the api key pending to be delete to call the endpoint. Use master user's api key only.
func (by *Bybit) DeleteMasterAPIKey(ctx context.Context) error {
	var resp interface{}
	return by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/delete-api", nil, nil, &resp, privateSpotRate)
}

// DeleteSubAccountAPIKey delete the api key of sub account.
// Use the api key pending to be delete to call the endpoint. Use sub user's api key only.
func (by *Bybit) DeleteSubAccountAPIKey(ctx context.Context, subAccountUID string) error {
	if subAccountUID == "" {
		return fmt.Errorf("%w, sub-account id missing", errMissingUserID)
	}
	arg := &struct {
		UID string `json:"uid"`
	}{
		UID: subAccountUID,
	}
	var resp interface{}
	return by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/user/delete-sub-api", nil, arg, &resp, privateSpotRate)
}

// GetAffiliateUserInfo the API is used for affiliate to get their users information
// The master account uid of affiliate's client
func (by *Bybit) GetAffiliateUserInfo(ctx context.Context, uid string) (*AffiliateCustomerInfo, error) {
	if uid == "" {
		return nil, errMissingUserID
	}
	params := url.Values{}
	params.Set("uid", uid)
	var resp AffiliateCustomerInfo
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/user/aff-customer-info", params, nil, &resp, privateSpotRate)
}

// ----------------------------------------------------------------------------  Spot Leverage Token ----------------------------------------------------------------

// GetLeverageTokenInfo query leverage token information
// Abbreviation of the LT, such as BTC3L
func (by *Bybit) GetLeverageTokenInfo(ctx context.Context, ltCoin currency.Code) ([]LeverageTokenInfo, error) {
	params := url.Values{}
	if !ltCoin.IsEmpty() {
		params.Set("ltCoin", ltCoin.String())
	}
	resp := &struct {
		List []LeverageTokenInfo `json:"list"`
	}{}
	return resp.List, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-lever-token/info", params, nil, &resp, privateSpotRate)
}

// GetLeveragedTokenMarket retrieves leverage token market information
func (by *Bybit) GetLeveragedTokenMarket(ctx context.Context, ltCoin currency.Code) (*LeveragedTokenMarket, error) {
	if ltCoin.IsEmpty() {
		return nil, fmt.Errorf("%w, 'ltCoin' is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("ltCoin", ltCoin.String())
	var resp LeveragedTokenMarket
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-lever-token/reference", params, nil, &resp, privateSpotRate)
}

// PurchaseLeverageToken purcases a leverage token.
func (by *Bybit) PurchaseLeverageToken(ctx context.Context, ltCoin currency.Code, amount float64, serialNumber string) (*LeverageToken, error) {
	if ltCoin.IsEmpty() {
		return nil, fmt.Errorf("%w, 'ltCoin' is required", currency.ErrCurrencyCodeEmpty)
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
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
	var resp LeverageToken
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/spot-lever-token/purchase", nil, arg, &resp, privateSpotRate)
}

// RedeemLeverageToken redeem leverage token
func (by *Bybit) RedeemLeverageToken(ctx context.Context, ltCoin currency.Code, quantity float64, serialNumber string) (*RedeemToken, error) {
	if ltCoin.IsEmpty() {
		return nil, fmt.Errorf("%w, 'ltCoin' is required", currency.ErrCurrencyCodeEmpty)
	}
	if quantity <= 0 {
		return nil, fmt.Errorf("%w, quantity=%f", order.ErrAmountBelowMin, quantity)
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
	var resp RedeemToken
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/spot-lever-token/redeem", nil, &arg, &resp, privateSpotRate)
}

// GetPurchaseAndRedemptionRecords retrieves purchase or redeem history.
// ltOrderType	false	integer	LT order type. 1: purchase, 2: redemption
func (by *Bybit) GetPurchaseAndRedemptionRecords(ctx context.Context, ltCoin currency.Code, orderID, serialNo string, startTime, endTime time.Time, ltOrderType, limit int64) ([]RedeemPurchaseRecord, error) {
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
	resp := &struct {
		List []RedeemPurchaseRecord `json:"list"`
	}{}
	return resp.List, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-lever-token/order-record", params, nil, &resp, privateSpotRate)
}

// ---------------------------------------------------------------- Spot Margin Trade (UTA) ----------------------------------------------------------------

// ToggleMarginTrade turn on / off spot margin trade
// Your account needs to activate spot margin first; i.e., you must have finished the quiz on web / app.
// spotMarginMode '1': on, '0': off
func (by *Bybit) ToggleMarginTrade(ctx context.Context, spotMarginMode bool) (*SpotMarginMode, error) {
	arg := &SpotMarginMode{}
	if spotMarginMode {
		arg.SpotMarginMode = "1"
	} else {
		arg.SpotMarginMode = "0"
	}
	var resp SpotMarginMode
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/spot-margin-trade/switch-mode", nil, arg, &resp, privateSpotRate)
}

// SetSpotMarginTradeLeverage set the user's maximum leverage in spot cross margin
func (by *Bybit) SetSpotMarginTradeLeverage(ctx context.Context, leverage float64) error {
	if leverage <= 2 {
		return fmt.Errorf("%w, Leverage. [2, 10].", errInvalidLeverage)
	}
	var resp interface{}
	return by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/spot-margin-trade/set-leverage", nil, &map[string]string{"leverage": strconv.FormatFloat(leverage, 'f', -1, 64)}, &resp, privateSpotRate)
}

// ---------------------------------------------------------------- Spot Margin Trade (Normal) ----------------------------------------------------------------

// GetMarginCoinInfo retrieves margin coin information.
func (by *Bybit) GetMarginCoinInfo(ctx context.Context, coin currency.Code) ([]MarginCoinInfo, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	resp := &struct {
		List []MarginCoinInfo `json:"list"`
	}{}
	return resp.List, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-cross-margin-trade/pledge-token", params, nil, &resp, privateSpotRate)
}

// GetBorrowableCoinInfo retrieves borrowable coin info list.
func (by *Bybit) GetBorrowableCoinInfo(ctx context.Context, coin currency.Code) ([]BorrowableCoinInfo, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	resp := &struct {
		List []BorrowableCoinInfo `json:"list"`
	}{}
	return resp.List, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-cross-margin-trade/borrow-token", params, nil, &resp, privateSpotRate)
}

// GetInterestAndQuota retrieves interest and quota information.
func (by *Bybit) GetInterestAndQuota(ctx context.Context, coin currency.Code) (*InterestAndQuota, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", coin.String())
	var resp InterestAndQuota
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-cross-margin-trade/loan-info", params, nil, &resp, privateSpotRate)
}

// GetLoanAccountInfo retrieves loan account information.
func (by *Bybit) GetLoanAccountInfo(ctx context.Context) (*AccountLoanInfo, error) {
	var resp AccountLoanInfo
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-cross-margin-trade/account", nil, nil, &resp, privateSpotRate)
}

// Borrow borrows a coin.
func (by *Bybit) Borrow(ctx context.Context, arg *LendArgument) (*BorrowResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.AmountToBorrow <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	var resp BorrowResponse
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/spot-cross-margin-trade/loan", nil, arg, &resp, privateSpotRate)
}

// Repay repay a debt.
func (by *Bybit) Repay(ctx context.Context, arg *LendArgument) (*RepayResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.AmountToBorrow <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	var resp RepayResponse
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/spot-cross-margin-trade/repay", nil, arg, &resp, privateSpotRate)
}

// GetBorrowOrderDetail represents the borrow order detail.
// Status '0'(default)：get all kinds of status '1'：uncleared '2'：cleared
func (by *Bybit) GetBorrowOrderDetail(ctx context.Context, startTime, endTime time.Time, coin currency.Code, status, limit int64) ([]BorrowOrderDetail, error) {
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
	resp := &struct {
		List []BorrowOrderDetail `json:"list"`
	}{}
	return resp.List, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-cross-margin-trade/orders", params, nil, &resp, privateSpotRate)
}

// GetRepaymentOrderDetail retrieves repayment order detail.
func (by *Bybit) GetRepaymentOrderDetail(ctx context.Context, startTime, endTime time.Time, coin currency.Code, limit int64) ([]CoinRepaymentResponse, error) {
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
	resp := &struct {
		List []CoinRepaymentResponse `json:"list"`
	}{}
	return resp.List, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/spot-cross-margin-trade/repay-history", params, nil, &resp, privateSpotRate)
}

// ToggleMarginTradeNormal turn on / off spot margin trade
// Your account needs to activate spot margin first; i.e., you must have finished the quiz on web / app.
// spotMarginMode '1': on, '0': off
func (by *Bybit) ToggleMarginTradeNormal(ctx context.Context, spotMarginMode bool) (*SpotMarginMode, error) {
	arg := &SpotMarginMode{}
	if spotMarginMode {
		arg.SpotMarginMode = "1"
	} else {
		arg.SpotMarginMode = "0"
	}
	var resp SpotMarginMode
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/spot-margin-trade/switch-mode", nil, arg, &resp, privateSpotRate)
}

// ----------------------------------------- Institutional Lending -----------------------------------------

// GetProductInfo represents a product info.
func (by *Bybit) GetProductInfo(ctx context.Context, productID string) (*InstitutionalProductInfo, error) {
	params := url.Values{}
	if productID != "" {
		params.Set("productId", productID)
	}
	var resp InstitutionalProductInfo
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/ins-loan/product-infos", params, nil, &resp, privateSpotRate)
}

// GetInstitutionalLengingMarginCoinInfo retrieves institutional lending margin coin information.
// ProductId. If not passed, then return all product margin coin. For spot, it returns coin that convertRation greater than 0.
func (by *Bybit) GetInstitutionalLengingMarginCoinInfo(ctx context.Context, productID string) (*InstitutionalMarginCoinInfo, error) {
	params := url.Values{}
	if productID != "" {
		params.Set("productId", productID)
	}
	var resp InstitutionalMarginCoinInfo
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/ins-loan/ensure-tokens-convert", params, nil, &resp, privateSpotRate)
}

// GetInstitutionalLoanOrders retrieves institutional loan orders.
func (by *Bybit) GetInstitutionalLoanOrders(ctx context.Context, orderID string, startTime, endTime time.Time, limit int64) ([]LoanOrderDetails, error) {
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
	resp := &struct {
		Loans []LoanOrderDetails `json:"loanInfo"`
	}{}
	return resp.Loans, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/ins-loan/loan-order", params, nil, &resp, privateSpotRate)
}

// GetInstitutionalRepayOrders retrieves list of repaid order information.
func (by *Bybit) GetInstitutionalRepayOrders(ctx context.Context, startTime, endTime time.Time, limit int64) ([]OrderRepayInfo, error) {
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
	resp := &struct {
		RepayInfo []OrderRepayInfo `json:"repayInfo"`
	}{}
	return resp.RepayInfo, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/ins-loan/repaid-history", params, nil, &resp, privateSpotRate)
}

// GetLTV retrieves a loan-to-value(LTV)
func (by *Bybit) GetLTV(ctx context.Context) (*LTVInfo, error) {
	var resp LTVInfo
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/ins-loan/ltv-convert", nil, nil, &resp, privateSpotRate)
}

// --------------------------------------------------------- Contract-to-contract lending ----------------------------------------------------

// GetC2CLendingCoinInfo retrieves C2C basic information of lending coins
func (by *Bybit) GetC2CLendingCoinInfo(ctx context.Context, coin currency.Code) ([]C2CLendingCoinInfo, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	resp := &struct {
		List []C2CLendingCoinInfo `json:"list"`
	}{}
	return resp.List, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/lending/info", params, nil, &resp, privateSpotRate)
}

// C2CDepositFunds lending funds to Bybit asset pool
func (by *Bybit) C2CDepositFunds(ctx context.Context, arg *C2CLendingFundsParams) (*C2CLendingFundResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Quantity <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	var resp C2CLendingFundResponse
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/lending/purchase", nil, &arg, &resp, privateSpotRate)
}

// C2CRedeemFunds withdraw funds from the Bybit asset pool.
func (by *Bybit) C2CRedeemFunds(ctx context.Context, arg *C2CLendingFundsParams) (*C2CLendingFundResponse, error) {
	if arg == nil {
		return nil, errNilArgument
	}
	if arg.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Quantity <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	var resp C2CLendingFundResponse
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodPost, "/v5/lending/redeem", nil, &arg, &resp, privateSpotRate)
}

// GetC2CLendingOrderRecords retrieves lending or redeem history
func (by *Bybit) GetC2CLendingOrderRecords(ctx context.Context, coin currency.Code, orderID, orderType string, startTime, endTime time.Time, limit int64) ([]C2CLendingFundResponse, error) {
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
	resp := &struct {
		List []C2CLendingFundResponse `json:"list"`
	}{}
	return resp.List, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/lending/history-order", params, nil, &resp, privateSpotRate)
}

// GetC2CLendingAccountInfo retrieves C2C lending account information.
func (by *Bybit) GetC2CLendingAccountInfo(ctx context.Context, coin currency.Code) (*LendingAccountInfo, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	var resp LendingAccountInfo
	return &resp, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/lending/account", params, nil, &resp, privateSpotRate)
}

//  ---------------------------------------------------------------- Broker endoint ----------------------------------------------------------------

// GetBrokerEarning exchange broker master account to query
// The data can support up to past 6 months until T-1
// startTime & endTime are either entered at the same time or not entered
// Business type. 'SPOT', 'DERIVATIVES', 'OPTIONS'
func (by *Bybit) GetBrokerEarning(ctx context.Context, businessType, cursor string, startTime, endTime time.Time, limit int64) ([]BrokerEarningItem, error) {
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
	resp := &struct {
		List []BrokerEarningItem `json:"list"`
	}{}
	return resp.List, by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, "/v5/broker/earning-record", params, nil, &resp, privateSpotRate)
}

// GetFeeRate returns user account fee
// Valid  category: "spot", "linear", "inverse", "option"
// func (by *Bybit) GetFeeRate(ctx context.Context, category, symbol, baseCoin string) (*AccountFee, error) {
// 	if category == "" {
// 		return nil, errCategoryNotSet
// 	}

// 	if !common.StringDataContains(validCategory, category) {
// 		// NOTE: Opted to fail here because if the user passes in an invalid
// 		// category the error returned is this
// 		// `Bybit raw response: {"retCode":10005,"retMsg":"Permission denied, please check your API key permissions.","result":{},"retExtInfo":{},"time":1683694010783}`
// 		return nil, fmt.Errorf("%w, valid category values are %v", errInvalidCategory, validCategory)
// 	}

// 	params := url.Values{}
// 	params.Set("category", category)
// 	if symbol != "" {
// 		params.Set("symbol", symbol)
// 	}
// 	if baseCoin != "" {
// 		params.Set("baseCoin", baseCoin)
// 	}

// 	result := struct {
// 		Data *AccountFee `json:"result"`
// 		Error
// 	}{}

// 	err := by.SendAuthHTTPRequestV5(ctx, exchange.RestSpot, http.MethodGet, bybitAccountFee, params, nil, &result, privateFeeRate)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return result.Data, nil
// }

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
