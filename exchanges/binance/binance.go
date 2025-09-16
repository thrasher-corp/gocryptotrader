package binance

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with Binance
type Exchange struct {
	exchange.Base
	// Valid string list that is required by the exchange
	validLimits []int64
	obm         *orderbookManager

	// isAPIStreamConnected is true if the spot API stream websocket connection is established
	isAPIStreamConnected bool

	isAPIStreamConnectionLock sync.Mutex
}

const (
	apiURL         = "https://api4.binance.com"
	cfuturesAPIURL = "https://dapi.binance.com"
	ufuturesAPIURL = "https://fapi.binance.com"
	eOptionAPIURL  = "https://eapi.binance.com"
	pMarginAPIURL  = "https://papi.binance.com"
	tradeBaseURL   = "https://www.binance.com/en/"

	defaultRecvWindow = 5 * time.Second
)

// GetExchangeServerTime retrieves the server time.
func (e *Exchange) GetExchangeServerTime(ctx context.Context) (types.Time, error) {
	resp := &struct {
		ServerTime types.Time `json:"serverTime"`
	}{}
	return resp.ServerTime, e.SendHTTPRequest(ctx, exchange.RestSpot, "/api/v3/time", spotDefaultRate, resp)
}

// GetExchangeInfo returns exchange information. Check binance_types for more
// information
func (e *Exchange) GetExchangeInfo(ctx context.Context) (*ExchangeInfo, error) {
	var resp *ExchangeInfo
	return resp, e.SendHTTPRequest(ctx,
		exchange.RestSpot, "/api/v3/exchangeInfo", spotExchangeInfo, &resp)
}

// GetOrderBook returns full orderbook information
//
// OrderBookDataRequestParams contains the following members
// symbol: string of currency pair
// limit: returned limit amount
func (e *Exchange) GetOrderBook(ctx context.Context, obd OrderBookDataRequestParams) (*OrderBook, error) {
	if err := e.CheckLimit(obd.Limit); err != nil {
		return nil, err
	}
	symbol, err := e.FormatSymbol(obd.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("limit", strconv.FormatInt(obd.Limit, 10))
	var resp *OrderBook
	return resp, e.SendHTTPRequest(ctx,
		exchange.RestSpot,
		common.EncodeURLValues("/api/v3/depth", params),
		orderbookLimit(obd.Limit), &resp)
}

// GetMostRecentTrades returns recent trade activity
// limit: Up to 500 results returned
func (e *Exchange) GetMostRecentTrades(ctx context.Context, rtr *RecentTradeRequestParams) ([]RecentTrade, error) {
	if *rtr == (RecentTradeRequestParams{}) {
		return nil, common.ErrEmptyParams
	}
	symbol, err := e.FormatSymbol(rtr.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("limit", strconv.FormatInt(rtr.Limit, 10))
	var resp []RecentTrade
	return resp, e.SendHTTPRequest(ctx,
		exchange.RestSpot, common.EncodeURLValues("/api/v3/trades", params), getRecentTradesListRate, &resp)
}

// GetHistoricalTrades returns historical trade activity
//
// symbol: string of currency pair
// limit: Optional. Default 500; max 1000.
// fromID:
func (e *Exchange) GetHistoricalTrades(ctx context.Context, symbol string, limit, fromID int64) ([]HistoricalTrade, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("limit", strconv.FormatInt(limit, 10))
	// else return most recent trades
	if fromID > 0 {
		params.Set("fromId", strconv.FormatInt(fromID, 10))
	}
	var resp []HistoricalTrade
	return resp,
		e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/api/v3/historicalTrades", params), getOldTradeLookupRate, &resp)
}

// GetAggregatedTrades returns aggregated trade activity.
// If more than one hour of data is requested or asked limit is not supported by exchange
// then the trades are collected with multiple backend requests.
// https://binance-docs.github.io/apidocs/spot/en/#compressed-aggregate-trades-list
func (e *Exchange) GetAggregatedTrades(ctx context.Context, arg *AggregatedTradeRequestParams) ([]AggregatedTrade, error) {
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", arg.Symbol)
	// If the user request is directly not supported by the exchange, we might be able to fulfill it
	// by merging results from multiple API requests
	needBatch := true // Need to batch unless user has specified a limit
	if arg.Limit > 0 && arg.Limit <= 1000 {
		needBatch = false
		params.Set("limit", strconv.Itoa(arg.Limit))
	}
	if arg.FromID != 0 {
		params.Set("fromId", strconv.FormatInt(arg.FromID, 10))
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}

	// startTime and endTime are set and time between startTime and endTime is more than 1 hour
	needBatch = needBatch || (!arg.StartTime.IsZero() && !arg.EndTime.IsZero() && arg.EndTime.Sub(arg.StartTime) > time.Hour)
	// Fall back to batch requests, if possible and necessary
	if needBatch {
		// fromId or start time must be set
		canBatch := (arg.FromID == 0) != arg.StartTime.IsZero()
		if canBatch {
			// Split the request into multiple
			return e.batchAggregateTrades(ctx, arg, params)
		}
		// Can not handle this request locally or remotely
		// We would receive {"code":-1128,"msg":"Combination of optional parameters invalid."}
		return nil, errors.New("either StartTime or FromId must be provided")
	}
	var resp []AggregatedTrade
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/api/v3/aggTrades", params), aggTradesRate, &resp)
}

// batchAggregateTrades fetches trades in multiple requests
// first phase, hourly requests until the first trade (or end time) is reached
// second phase, limit requests from previous trade until end time (or limit) is reached
func (e *Exchange) batchAggregateTrades(ctx context.Context, arg *AggregatedTradeRequestParams, params url.Values) ([]AggregatedTrade, error) {
	// prepare first request with only first hour and max limit
	if arg.Limit == 0 || arg.Limit > 1000 {
		// Extend from the default of 500
		params.Set("limit", "1000")
	}
	var resp []AggregatedTrade
	var fromID int64
	if arg.FromID > 0 {
		fromID = arg.FromID
	} else {
		// Only 10 seconds is used to prevent limit of 1000 being reached in the first request,
		// cutting off trades for high activity pairs
		increment := time.Second * 10
		for start := arg.StartTime; len(resp) == 0; start = start.Add(increment) {
			if !arg.EndTime.IsZero() && start.After(arg.EndTime) {
				// All requests returned empty
				return nil, nil
			}
			params.Set("startTime", timeString(start))
			params.Set("endTime", timeString(start.Add(increment)))
			err := e.SendHTTPRequest(ctx,
				exchange.RestSpot,
				common.EncodeURLValues("/api/v3/aggTrades", params),
				getAggregateTradeListRate, &resp)
			if err != nil {
				return resp, fmt.Errorf("%w %v", err, arg.Symbol)
			}
		}
		fromID = resp[len(resp)-1].ATradeID
	}

	// other requests follow from the last aggregate trade id and have no time window
	params.Del("startTime")
	params.Del("endTime")
	// while we haven't reached the limit
	for ; arg.Limit == 0 || len(resp) < arg.Limit; fromID = resp[len(resp)-1].ATradeID {
		// Keep requesting new data after last retrieved trade
		params.Set("fromId", strconv.FormatInt(fromID, 10))
		var additionalTrades []AggregatedTrade
		err := e.SendHTTPRequest(ctx,
			exchange.RestSpot,
			common.EncodeURLValues("/api/v3/aggTrades", params),
			spotDefaultRate,
			&additionalTrades)
		if err != nil {
			return resp, fmt.Errorf("%w %v", err, arg.Symbol)
		}
		lastIndex := len(additionalTrades)
		if !arg.EndTime.IsZero() {
			// get index for truncating to end time
			lastIndex = sort.Search(len(additionalTrades), func(i int) bool {
				return arg.EndTime.Before(additionalTrades[i].TimeStamp.Time())
			})
		}
		// don't include the first as the request was inclusive from last ATradeID
		resp = append(resp, additionalTrades[1:lastIndex]...)
		// If only the starting trade is returned or if we received trades after end time
		if len(additionalTrades) == 1 || lastIndex < len(additionalTrades) {
			// We found the end
			break
		}
	}
	// Truncate if necessary
	if arg.Limit > 0 && len(resp) > arg.Limit {
		resp = resp[:arg.Limit]
	}
	return resp, nil
}

// GetSpotKline returns kline data
//
// KlinesRequestParams supports 5 parameters
// symbol: the symbol to get the kline data for
// limit: optional
// interval: the interval time for the data
// startTime: startTime filter for kline data
// endTime: endTime filter for the kline data
func (e *Exchange) GetSpotKline(ctx context.Context, arg *KlinesRequestParams) ([]CandleStick, error) {
	return e.retrieveSpotKline(ctx, arg, "/api/v3/klines")
}

// GetUIKline return modified kline data, optimized for presentation of candlestick charts.
func (e *Exchange) GetUIKline(ctx context.Context, arg *KlinesRequestParams) ([]CandleStick, error) {
	return e.retrieveSpotKline(ctx, arg, "/api/v3/uiKlines")
}

func (e *Exchange) retrieveSpotKline(ctx context.Context, arg *KlinesRequestParams, urlPath string) ([]CandleStick, error) {
	symbol, err := e.FormatSymbol(arg.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", arg.Interval)
	if arg.Limit != 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	if !arg.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
	}
	var resp []CandleStick
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues(urlPath, params), getKlineRate, &resp)
}

// GetAveragePrice returns current average price for a symbol.
//
// symbol: string of currency pair
func (e *Exchange) GetAveragePrice(ctx context.Context, symbol currency.Pair) (*AveragePrice, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	var resp *AveragePrice
	return resp, e.SendHTTPRequest(ctx,
		exchange.RestSpot, common.EncodeURLValues("/api/v3/avgPrice", params), getCurrentAveragePriceRate, &resp)
}

// GetPriceChangeStats returns price change statistics for the last 24 hours
//
// symbol: string of currency pair
func (e *Exchange) GetPriceChangeStats(ctx context.Context, symbol currency.Pair, symbols currency.Pairs) ([]PriceChangeStats, error) {
	params := url.Values{}
	rateLimit := spotPriceChangeAllRate
	switch {
	case !symbol.IsEmpty(),
		symbol.IsEmpty() && len(symbols) == 1 && !symbols[0].IsEmpty():
		if symbol.IsEmpty() && len(symbols) == 1 {
			symbol = symbols[0]
		}
		rateLimit = get24HrTickerPriceChangeStatisticsRate
		symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	case len(symbols) > 1:
		rateLimit = getTickers20Rate
		if len(symbols) > 100 {
			rateLimit = getTickersMoreThan100Rate
		} else if len(symbols) > 20 {
			rateLimit = getTickers100Rate
		}
		val, err := json.Marshal(symbols.Strings())
		if err != nil {
			return nil, err
		}
		params.Set("symbols", string(val))
	}
	var resp PriceChangesWrapper
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/api/v3/ticker/24hr", params), rateLimit, &resp)
}

// GetTradingDayTicker retrieves the price change statistics for the trading day
// possible tickerType values: FULL or MINI
func (e *Exchange) GetTradingDayTicker(ctx context.Context, symbols currency.Pairs, timeZone, tickerType string) ([]PriceChangeStats, error) {
	if len(symbols) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	params := url.Values{}
	switch {
	case len(symbols) > 1:
		params.Set("symbols", "["+strings.Join(symbols.Strings(), "")+"]")
	case len(symbols) == 1 && !symbols[0].IsEmpty():
		params.Set("symbol", symbols[0].String())
	default:
		return nil, currency.ErrCurrencyPairEmpty
	}
	if timeZone != "" {
		params.Set("timeZone", timeZone)
	}
	if tickerType != "" {
		params.Set("type", tickerType)
	}
	var resp PriceChangesWrapper
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/api/v3/ticker/tradingDay", params), spotPriceChangeAllRate, &resp)
}

// GetLatestSpotPrice returns latest spot price of symbol
//
// symbol: string of currency pair
func (e *Exchange) GetLatestSpotPrice(ctx context.Context, symbol currency.Pair, symbols currency.Pairs) (*SymbolPrice, error) {
	params := url.Values{}
	rateLimit := spotTickerAllRate
	if !symbol.IsEmpty() {
		rateLimit = spotSymbolPriceRate
		symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	} else if len(symbols) > 0 {
		params.Set("symbols", "["+strings.Join(symbols.Strings(), "")+"]")
	}
	var resp *SymbolPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/api/v3/ticker/price", params), rateLimit, &resp)
}

// GetBestPrice returns the latest best price for symbol
//
// symbol: string of currency pair
func (e *Exchange) GetBestPrice(ctx context.Context, symbol currency.Pair, symbols currency.Pairs) (*BestPrice, error) {
	params := url.Values{}
	rateLimit := spotOrderbookTickerAllRate
	if !symbol.IsEmpty() || (symbol.IsEmpty() && len(symbols) == 1 && !symbols[0].IsEmpty()) {
		if symbol.IsEmpty() && len(symbols) == 1 {
			symbol = symbols[0]
		}
		rateLimit = spotBookTickerRate
		symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	} else if len(symbols) > 1 {
		params.Set("symbols", "["+strings.Join(symbols.Strings(), "")+"]")
	}
	var resp *BestPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/api/v3/ticker/bookTicker", params), rateLimit, &resp)
}

// GetTickerData openTime always starts on a minute, while the closeTime is the current time of the request.
// As such, the effective window will be up to 59999ms wider than windowSize.
// possible windowSize values are FULL and MINI
func (e *Exchange) GetTickerData(ctx context.Context, symbols currency.Pairs, windowSize time.Duration, tickerType string) ([]PriceChangeStats, error) {
	if len(symbols) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	params := url.Values{}
	switch {
	case len(symbols) > 1:
		params.Set("symbols", "["+strings.Join(symbols.Strings(), "")+"]")
	case len(symbols) == 1 && !symbols[0].IsEmpty():
		params.Set("symbol", symbols[0].String())
	default:
		return nil, currency.ErrCurrencyPairEmpty
	}
	switch {
	case windowSize > time.Hour*24:
		params.Set("windowSize", strconv.FormatInt(int64(windowSize/(time.Hour*24)), 10)+"d")
	case windowSize >= time.Hour:
		params.Set("windowSize", strconv.FormatInt(int64(windowSize/(time.Hour)), 10)+"h")
	case windowSize >= time.Minute:
		params.Set("windowSize", strconv.FormatInt(int64(windowSize/time.Minute), 10)+"m")
	}
	if tickerType != "" {
		params.Set("type", tickerType)
	}
	var resp PriceChangesWrapper
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/api/v3/ticker", params), spotDefaultRate, &resp)
}

// NewOrder sends a new order to Binance
func (e *Exchange) NewOrder(ctx context.Context, o *NewOrderRequest) (NewOrderResponse, error) {
	var resp NewOrderResponse
	if err := e.newOrder(ctx, "/api/v3/order", o, &resp, false); err != nil {
		return resp, err
	}
	if resp.Code != 0 {
		return resp, errors.New(resp.Msg)
	}
	return resp, nil
}

// NewOrderTest sends a new test order to Binance
func (e *Exchange) NewOrderTest(ctx context.Context, o *NewOrderRequest, computeCommisionRates bool) error {
	var resp NewOrderResponse
	return e.newOrder(ctx, "/api/v3/order/test", o, &resp, computeCommisionRates)
}

func (e *Exchange) newOrder(ctx context.Context, api string, o *NewOrderRequest, resp *NewOrderResponse, computeCommissionRate bool) error {
	ratelimit := spotOrderRate
	if computeCommissionRate {
		ratelimit = testNewOrderWithCommissionRate
	}
	symbol, err := e.FormatSymbol(o.Symbol, asset.Spot)
	if err != nil {
		return err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", o.Side)
	params.Set("type", o.TradeType)
	if o.QuoteOrderQty > 0 {
		params.Set("quoteOrderQty", strconv.FormatFloat(o.QuoteOrderQty, 'f', -1, 64))
	} else {
		params.Set("quantity", strconv.FormatFloat(o.Quantity, 'f', -1, 64))
	}
	if o.TradeType == order.Limit.String() {
		params.Set("price", strconv.FormatFloat(o.Price, 'f', -1, 64))
	}
	if o.TimeInForce != "" {
		params.Set("timeInForce", o.TimeInForce)
	}
	if o.NewClientOrderID != "" {
		params.Set("newClientOrderId", o.NewClientOrderID)
	}
	if o.StopPrice != 0 {
		params.Set("stopPrice", strconv.FormatFloat(o.StopPrice, 'f', -1, 64))
	}
	if o.IcebergQty != 0 {
		params.Set("icebergQty", strconv.FormatFloat(o.IcebergQty, 'f', -1, 64))
	}
	if o.NewOrderRespType != "" {
		params.Set("newOrderRespType", o.NewOrderRespType)
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, api, params, ratelimit, nil, resp)
}

// CancelExistingOrder sends a cancel order to Binance
func (e *Exchange) CancelExistingOrder(ctx context.Context, symbol currency.Pair, orderID int64, origClientOrderID string) (*CancelOrderResponse, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	var resp *CancelOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, "/api/v3/order", params, spotOrderRate, nil, &resp)
}

// OpenOrders Current open orders. Get all open orders on a symbol.
// Careful when accessing this with no symbol: The number of requests counted
// against the rate limiter is significantly higher
func (e *Exchange) OpenOrders(ctx context.Context, pair currency.Pair) ([]TradeOrder, error) {
	var (
		p   string
		err error
	)
	params := url.Values{}
	if !pair.IsEmpty() {
		p, err = e.FormatSymbol(pair, asset.Spot)
		if err != nil {
			return nil, err
		}
		params.Add("symbol", p)
	} else {
		// extend the receive window when all currencies to prevent "recvwindow"
		// error
		params.Set("recvWindow", "10000")
	}
	var resp []TradeOrder
	return resp, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpot, http.MethodGet,
		"/api/v3/openOrders", params, openOrdersLimit(p), nil, &resp)
}

// CancelAllOpenOrderOnSymbol cancels all active orders on a symbol.
func (e *Exchange) CancelAllOpenOrderOnSymbol(ctx context.Context, symbol string) ([]SymbolOrders, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp []SymbolOrders
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, "/api/v3/openOrders", params, spotOrderRate, nil, &resp)
}

// AllOrders Get all account orders; active, canceled, or filled.
// orderId optional param
// limit optional param, default 500; max 500
func (e *Exchange) AllOrders(ctx context.Context, symbol currency.Pair, orderID, limit string) ([]TradeOrder, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if limit != "" {
		params.Set("limit", limit)
	}
	var resp []TradeOrder
	return resp, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpot, http.MethodGet, "/api/v3/allOrders",
		params, spotAllOrdersRate, nil, &resp)
}

// NewOCOOrder places a new one-cancel-other trade order.
func (e *Exchange) NewOCOOrder(ctx context.Context, arg *OCOOrderParam) (*OCOOrder, error) {
	if *arg == (OCOOrderParam{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.Price <= 0 {
		return nil, limits.ErrPriceBelowMin
	}
	if arg.StopPrice <= 0 {
		return nil, fmt.Errorf("%w stop price is required", limits.ErrPriceBelowMin)
	}
	params := url.Values{}
	params.Set("quantity", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	params.Set("stopPrice", strconv.FormatFloat(arg.StopPrice, 'f', -1, 64))
	var resp *OCOOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/api/v3/order/oco", params, spotDefaultRate, arg, &resp)
}

// NewOCOOrderList send in an one-cancels-the-other (OCO) pair, where activation of one order immediately cancels the other.
// An OCO has 2 legs called the above leg and below leg.
// One of the legs must be a LIMIT_MAKER order and the other leg must be STOP_LOSS or STOP_LOSS_LIMIT order.
// Price restrictions:
//
//	If the OCO is on the SELL side: LIMIT_MAKER price > Last Traded Price > stopPrice
//	If the OCO is on the BUY side: LIMIT_MAKER price < Last Traded Price < stopPrice
//
// OCO counts as 2 orders against the order rate limit.
func (e *Exchange) NewOCOOrderList(ctx context.Context, arg *OCOOrderListParams) (*OCOListOrderResponse, error) {
	if *arg == (OCOOrderListParams{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Quantity <= 0 {
		return nil, fmt.Errorf("%w: quantity must be greater than 0", limits.ErrAmountBelowMin)
	}
	if arg.AboveType == "" {
		return nil, fmt.Errorf("%w: aboveType is required", order.ErrTypeIsInvalid)
	}
	if arg.BelowType == "" {
		return nil, fmt.Errorf("%w: belowType is required", order.ErrTypeIsInvalid)
	}
	var resp *OCOListOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/api/v3/orderList/oco", nil, spotOrderRate, arg, &resp)
}

// CancelOCOOrder cancels an entire Order List.
func (e *Exchange) CancelOCOOrder(ctx context.Context, symbol, orderListID, listClientOrderID, newClientOrderID string) (*OCOOrder, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if orderListID == "" && listClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if orderListID != "" {
		params.Set("orderListId", orderListID)
	}
	if listClientOrderID != "" {
		params.Set("listClientOrderId", listClientOrderID)
	}
	if newClientOrderID != "" {
		params.Set("newClientOrderId", newClientOrderID)
	}
	var resp *OCOOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, "/api/v3/orderList", params, spotDefaultRate, nil, &resp)
}

// GetOCOOrders retrieves a specific OCO based on provided optional parameters
func (e *Exchange) GetOCOOrders(ctx context.Context, orderListID, origiClientOrderID string) (*OCOOrder, error) {
	if orderListID == "" && origiClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	if orderListID != "" {
		params.Set("orderListId", orderListID)
	}
	if origiClientOrderID != "" {
		params.Set("origClientOrderId", origiClientOrderID)
	}
	var resp *OCOOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/api/v3/orderList", params, getOCOListRate, nil, &resp)
}

// GetAllOCOOrders retrieves all OCO based on provided optional parameters
func (e *Exchange) GetAllOCOOrders(ctx context.Context, fromID string, startTime, endTime time.Time, limit int64) ([]OCOOrder, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if fromID != "" {
		params.Set("fromId", fromID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []OCOOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/api/v3/allOrderList", params, getAllOCOOrdersRate, nil, &resp)
}

// GetOpenOCOList retrieves an open OCO orders.
func (e *Exchange) GetOpenOCOList(ctx context.Context) ([]OCOOrder, error) {
	var resp []OCOOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/api/v3/openOrderList", nil, getOpenOCOListRate, nil, &resp)
}

// ----------------------------------------------------- Smart Order Routing (SOR) -----------------------------------------------------------

// NewOrderUsingSOR places an order using smart order routing (SOR).
func (e *Exchange) NewOrderUsingSOR(ctx context.Context, arg *SOROrderRequestParams) (*SOROrderResponse, error) {
	return e.newOrderUsingSOR(ctx, arg, "/api/v3/sor/order")
}

// NewOrderUsingSORTest test new order creation and signature/recvWindow using smart order routing (SOR).
// Creates and validates a new order but does not send it into the matching engine.
func (e *Exchange) NewOrderUsingSORTest(ctx context.Context, arg *SOROrderRequestParams) (*SOROrderResponse, error) {
	return e.newOrderUsingSOR(ctx, arg, "/api/v3/sor/order/test")
}

func (e *Exchange) newOrderUsingSOR(ctx context.Context, arg *SOROrderRequestParams, path string) (*SOROrderResponse, error) {
	if *arg == (SOROrderRequestParams{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if arg.Quantity <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	params := url.Values{}
	params.Set("symbol", arg.Symbol.String())
	params.Set("side", arg.Side)
	params.Set("type", arg.OrderType)
	params.Set("quantity", strconv.FormatFloat(arg.Quantity, 'f', -1, 64))
	if arg.Price <= 0 {
		params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	}
	if arg.IcebergQuantity != 0 {
		params.Set("icebergQty", strconv.FormatFloat(arg.IcebergQuantity, 'f', -1, 64))
	}
	var resp *SOROrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, params, spotOrderRate, arg, &resp)
}

// QueryOrder returns information on a past order
func (e *Exchange) QueryOrder(ctx context.Context, symbol currency.Pair, origClientOrderID string, orderID int64) (*TradeOrder, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	var resp *TradeOrder
	if err := e.SendAuthHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodGet, "/api/v3/order",
		params, spotOrderQueryRate,
		nil, &resp); err != nil {
		return resp, err
	}
	if resp.Code != 0 {
		return resp, errors.New(resp.Msg)
	}
	return resp, nil
}

// CancelExistingOrderAndSendNewOrder cancels an existing order and places a new order on the same symbol.
// Filters and Order Count are evaluated before the processing of the cancellation and order placement occurs.
// A new order that was not attempted (i.e. when newOrderResult: NOT_ATTEMPTED), will still increase the order count by 1.
func (e *Exchange) CancelExistingOrderAndSendNewOrder(ctx context.Context, arg *CancelReplaceOrderParams) (*CancelAndReplaceResponse, error) {
	if *arg == (CancelReplaceOrderParams{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if arg.CancelReplaceMode == "" {
		return nil, errCancelReplaceModeRequired
	}
	var resp *CancelAndReplaceResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/api/v3/order/cancelReplace", nil, spotOrderRate, arg, &resp)
}

// GetAccount returns binance user accounts
func (e *Exchange) GetAccount(ctx context.Context, omitZeroBalances bool) (*Account, error) {
	type response struct {
		Response
		Account
	}
	params := url.Values{}
	if omitZeroBalances {
		params.Set("omitZeroBalances", "true")
	}
	var resp response
	if err := e.SendAuthHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodGet, "/api/v3/account",
		params, spotAccountInformationRate,
		nil, &resp); err != nil {
		return &resp.Account, err
	}
	if resp.Code != 0 {
		return nil, errors.New(resp.Msg)
	}
	return &resp.Account, nil
}

// GetAccountTradeList retrieves trades for a specific account and symbol.
func (e *Exchange) GetAccountTradeList(ctx context.Context, symbol, orderID string, startTime, endTime time.Time, fromID, limit int64) ([]AccountTradeItem, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if fromID != 0 {
		params.Set("fromId", strconv.FormatInt(fromID, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []AccountTradeItem
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/api/v3/myTrades", params, accountTradeListRate, nil, &resp)
}

// GetCurrentOrderCountUsage displays the user's current order count usage for all intervals.
func (e *Exchange) GetCurrentOrderCountUsage(ctx context.Context) ([]CurrentOrderCountUsage, error) {
	var resp []CurrentOrderCountUsage
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/api/v3/rateLimit/order", nil, currentOrderCountUsageRate, nil, &resp)
}

// GetPreventedMatches displays the list of orders that were expired because of STP.
func (e *Exchange) GetPreventedMatches(ctx context.Context, symbol string, preventedMatchID, orderID, fromPreventedMatchID, limit int64) ([]PreventedMatches, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if preventedMatchID == 0 && orderID == 0 {
		return nil, fmt.Errorf("%w: either preventedMatchID or orderID", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	rateLimit := queryPreventedMatchsWithRate
	if preventedMatchID != 0 {
		params.Set("preventedMatchId", strconv.FormatInt(preventedMatchID, 10))
	}
	if orderID != 0 {
		rateLimit = preventedMatchesByOrderIDRate
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if fromPreventedMatchID != 0 {
		params.Set("fromPreventedMatchId", strconv.FormatInt(fromPreventedMatchID, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []PreventedMatches
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/api/v3/myPreventedMatches", params, rateLimit, nil, &resp)
}

// GetAllocations retrieves allocations resulting from SOR order placement.
func (e *Exchange) GetAllocations(ctx context.Context, symbol string, startTime, endTime time.Time, fromAllocationID, orderID, limit int64) ([]Allocation, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if fromAllocationID > 0 {
		params.Set("fromAllocationId", strconv.FormatInt(fromAllocationID, 10))
	}
	if orderID > 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []Allocation
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/api/v3/myAllocations", params, getAllocationsRate, nil, &resp)
}

// GetCommissionRates retrieves current account commission rates.
func (e *Exchange) GetCommissionRates(ctx context.Context, symbol string) (*AccountCommissionRate, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *AccountCommissionRate
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/api/v3/account/commission", params, getCommissionRate, nil, &resp)
}

// MarginAccountBorrowRepay represents a margin account borrow/repay
// lending type: possible values BORROW or REPAY
// Symbol is valid only for Isolated margin
func (e *Exchange) MarginAccountBorrowRepay(ctx context.Context, assetName currency.Code, symbol, lendingType string, isIsolated bool, amount float64) (string, error) {
	if symbol == "" {
		return "", currency.ErrSymbolStringEmpty
	}
	if assetName.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if lendingType == "" {
		return "", errLendingTypeRequired
	}
	if amount <= 0 {
		return "", limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("asset", assetName.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	if isIsolated {
		params.Set("isIsolated", "TRUE")
		// Default is FALSE for Cross Margin
	}
	resp := &struct {
		TransactionID string `json:"tranId"`
	}{}
	return resp.TransactionID, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/margin/borrow-repay", params, marginAccountBorrowRepayRate, nil, &resp)
}

// GetBorrowOrRepayRecordsInMarginAccount retrieves borrow/repay records in Margin account.
// tranId in POST /sapi/v1/margin/loan
func (e *Exchange) GetBorrowOrRepayRecordsInMarginAccount(ctx context.Context, assetName currency.Code, isolatedSymbol, lendingType string, transactionID, current, size int64, startTime, endTime time.Time) (*MarginAccountBorrowRepayRecords, error) {
	params := url.Values{}
	if !assetName.IsEmpty() {
		params.Set("asset", assetName.String())
	}
	if isolatedSymbol != "" {
		params.Set("isolatedSymbol", isolatedSymbol)
	}
	if transactionID != 0 {
		params.Set("txId", strconv.FormatInt(transactionID, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if current > 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	if lendingType != "" {
		params.Set("type", lendingType)
	}
	var resp *MarginAccountBorrowRepayRecords
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/borrow-repay", params, borrowRepayRecordsInMarginAccountRate, nil, &resp)
}

// GetAllMarginAssets retrieves all margin assets
func (e *Exchange) GetAllMarginAssets(ctx context.Context, assetName currency.Code) ([]MarginAsset, error) {
	params := url.Values{}
	if !assetName.IsEmpty() {
		params.Set("asset", assetName.String())
	}
	var resp []MarginAsset
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, common.EncodeURLValues("/sapi/v1/margin/allAssets", params), sapiDefaultRate, &resp)
}

// GetAllCrossMarginPairs retrieves all cross-margin pairs
func (e *Exchange) GetAllCrossMarginPairs(ctx context.Context, symbol string) ([]CrossMarginPairInfo, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []CrossMarginPairInfo
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, common.EncodeURLValues("/sapi/v1/margin/allPairs", params), sapiDefaultRate, &resp)
}

// GetMarginPriceIndex retrieves margin price index
func (e *Exchange) GetMarginPriceIndex(ctx context.Context, symbol string) (*MarginPriceIndex, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *MarginPriceIndex
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, common.EncodeURLValues("/sapi/v1/margin/priceIndex", params), getPriceMarginIndexRate, &resp)
}

// PostMarginAccountOrder post a new order for margin account.
// autoRepayAtCancel is suggested to set as “FALSE" to keep liability unrepaid under high frequent new order/cancel order execution
func (e *Exchange) PostMarginAccountOrder(ctx context.Context, arg *MarginAccountOrderParam) (*MarginAccountOrder, error) {
	if *arg == (MarginAccountOrderParam{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	arg.Side = strings.ToUpper(arg.Side)
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	var resp *MarginAccountOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/margin/order", nil, marginAccountNewOrderRate, arg, &resp)
}

// CancelMarginAccountOrder cancels an active order for margin account.
func (e *Exchange) CancelMarginAccountOrder(ctx context.Context, symbol, origClientOrderID, newClientOrderID string, isIsolated bool, orderID int64) (*MarginAccountOrder, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if orderID == 0 && origClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if isIsolated {
		params.Set("isIsolated", "TRUE")
	}
	if orderID > 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if newClientOrderID != "" {
		params.Set("newClientOrderId", newClientOrderID)
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	var resp *MarginAccountOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, "/sapi/v1/margin/order", params, marginAccountCancelOrderRate, nil, &resp)
}

// MarginAccountCancelAllOpenOrdersOnSymbol cancels all active orders on a symbol for margin account.
// This includes OCO orders.
func (e *Exchange) MarginAccountCancelAllOpenOrdersOnSymbol(ctx context.Context, symbol string, isIsolated bool) ([]MarginAccountOrderDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if isIsolated {
		params.Set("isIsolated", "TRUE")
	}
	var resp []MarginAccountOrderDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, "/sapi/v1/margin/openOrders", params, sapiDefaultRate, nil, &resp)
}

// AdjustCrossMarginMaxLeverage adjusts cross-margin account max leverage
// Can only adjust 3 , 5 or 10，Example:
// maxLeverage=10 for Cross Margin Pro ，
// maxLeverage = 5 or 3 for Cross Margin Classic
// The margin level need higher than the initial risk ratio of adjusted leverage, the initial risk ratio of 3x is 1.5 ,
// the initial risk ratio of 5x is 1.25, the initial risk ratio of 10x is 2.5.
func (e *Exchange) AdjustCrossMarginMaxLeverage(ctx context.Context, maxLeverage int64) (bool, error) {
	if maxLeverage <= 0 {
		return false, fmt.Errorf("%w: possible leverage values are 3, 5 are 10", order.ErrSubmitLeverageNotSupported)
	}
	params := url.Values{}
	params.Set("maxLeverage", strconv.FormatInt(maxLeverage, 10))
	resp := &struct {
		Success bool `json:"success"`
	}{}
	return resp.Success, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/margin/max-leverage", params, adjustCrossMarginMaxLeverageRate, nil, &resp)
}

// GetCrossMarginTransferHistory retrieves cross-margin transfer history in a descending order.
//
// The max interval between startTime and endTime is 30 days.
// Returns data for last 7 days by default
// Transfer Type possible values: ROLL_IN, ROLL_OUT
func (e *Exchange) GetCrossMarginTransferHistory(ctx context.Context, assetName currency.Code, transferType, isolatedSymbol string, startTime, endTime time.Time, current, size int64) (*CrossMarginTransferHistory, error) {
	params, err := fillMarginInterestAndTransferHistoryParams(assetName, transferType, isolatedSymbol, startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	var resp *CrossMarginTransferHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/transfer", params, sapiDefaultRate, nil, &resp)
}

func fillMarginInterestAndTransferHistoryParams(assetName currency.Code, transferType, isolatedSymbol string, startTime, endTime time.Time, current, size int64) (url.Values, error) {
	params := url.Values{}
	if !assetName.IsEmpty() {
		params.Set("asset", assetName.String())
	}
	if transferType != "" {
		params.Set("type", transferType)
	}
	if isolatedSymbol != "" {
		params.Set("isolatedSymbol", isolatedSymbol)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if current > 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	return params, nil
}

// GetUserMarginInterestHistory returns margin interest history for the user
func (e *Exchange) GetUserMarginInterestHistory(ctx context.Context, assetName currency.Code, isolatedSymbol string, startTime, endTime time.Time, current, size int64) (*UserMarginInterestHistoryResponse, error) {
	params, err := fillMarginInterestAndTransferHistoryParams(assetName, "", isolatedSymbol, startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	var resp *UserMarginInterestHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/interestHistory", params, sapiDefaultRate, nil, &resp)
}

// GetForceLiquidiationRecord retrieves force liquidiation records
func (e *Exchange) GetForceLiquidiationRecord(ctx context.Context, startTime, endTime time.Time, isolatedSymbol string, current, size int64) (*LiquidiationRecord, error) {
	params := url.Values{}
	if isolatedSymbol != "" {
		params.Set("isolatedSymbol", isolatedSymbol)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if current > 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	var resp *LiquidiationRecord
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/forceLiquidationRec", params, sapiDefaultRate, nil, &resp)
}

// GetCrossMarginAccountDetail retrieves cross-margin account details
func (e *Exchange) GetCrossMarginAccountDetail(ctx context.Context) (*CrossMarginAccount, error) {
	var resp *CrossMarginAccount
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot,
		http.MethodGet, "/sapi/v1/margin/account", nil,
		getCrossMarginAccountDetailRate, nil, &resp)
}

// GetMarginAccountsOrder retrieves margin account's order
//
// Either orderId or origClientOrderId must be sent.
// For some historical orders cummulativeQuoteQty will be < 0, meaning the data is not available at this time.
func (e *Exchange) GetMarginAccountsOrder(ctx context.Context, symbol, origClientOrderID string, isIsolated bool, orderID int64) (*TradeOrder, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if orderID == 0 && origClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if isIsolated {
		params.Set("isIsolated", "true")
	}
	if orderID > 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	var resp *TradeOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/order ", params, getCrossMarginAccountOrderRate, nil, &resp)
}

// GetMarginAccountsOpenOrders retrieves margin account's open orders
func (e *Exchange) GetMarginAccountsOpenOrders(ctx context.Context, symbol string, isIsolated bool) ([]TradeOrder, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if isIsolated {
		params.Set("isIsolated", "true")
	}
	var resp []TradeOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/openOrders", nil, getMarginAccountsOpenOrdersRate, nil, &resp)
}

// GetMarginAccountAllOrders retrieves all margin account's orders.
func (e *Exchange) GetMarginAccountAllOrders(ctx context.Context, symbol string, isIsolated bool, startTime, endTime time.Time, orderID, limit int64) ([]TradeOrder, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if isIsolated {
		params.Set("isIsolated", "TRUE")
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if orderID > 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []TradeOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/allOrders", nil, marginAccountsAllOrdersRate, nil, &resp)
}

// NewMarginAccountOCOOrder send in a new OCO for a margin account
//
// Price Restrictions:
// SELL: Limit Price > Last Price > Stop Price
// BUY: Limit Price < Last Price < Stop Price
// Quantity Restrictions:
// Both legs must have the same quantity
// ICEBERG quantities however do not have to be the same.
// Order Rate Limit
// OCO counts as 2 orders against the order rate limit.
// autoRepayAtCancel is suggested to set as “FALSE" to keep liability unrepaid under high frequent new order/cancel order execution
func (e *Exchange) NewMarginAccountOCOOrder(ctx context.Context, arg *MarginOCOOrderParam) (*OCOOrder, error) {
	if *arg == (MarginOCOOrderParam{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Quantity <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.Price <= 0 {
		return nil, limits.ErrPriceBelowMin
	}
	if arg.StopPrice <= 0 {
		return nil, fmt.Errorf("%w: stopPrice is required", limits.ErrPriceBelowMin)
	}
	var resp *OCOOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/margin/order/oco", nil, marginOCOOrderRate, arg, &resp)
}

// CancelMarginAccountOCOOrder cancel an entire Order List for a margin account.
func (e *Exchange) CancelMarginAccountOCOOrder(ctx context.Context, symbol, listClientOrderID, newClientOrderID string, isIsolated bool, orderListID int64) (*OCOOrder, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if isIsolated {
		params.Set("isIsolated", "TRUE")
	}
	if orderListID > 0 {
		params.Set("orderListId", strconv.FormatInt(orderListID, 10))
	}
	if listClientOrderID != "" {
		params.Set("listClientOrderId", listClientOrderID)
	}
	if newClientOrderID != "" {
		params.Set("newClientOrderId", newClientOrderID)
	}
	var resp *OCOOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, "/sapi/v1/margin/orderList", params, sapiDefaultRate, nil, &resp)
}

// GetMarginAccountOCOOrder retrieves a specific OCO based on provided optional parameters
func (e *Exchange) GetMarginAccountOCOOrder(ctx context.Context, symbol, origClientOrderID string, orderListID int64, isIsolated bool) (*OCOOrder, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if isIsolated {
		params.Set("isIsolated", "TRUE")
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	if orderListID > 0 {
		params.Set("orderListId", strconv.FormatInt(orderListID, 10))
	}
	var resp *OCOOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/orderList", params, getMarginAccountOCOOrderRate, nil, &resp)
}

func ocoOrdersAndTradeParams(symbol string, isIsolated bool, startTime, endTime time.Time, orderID, fromID, limit int64) (url.Values, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if isIsolated {
		params.Set("isIsolated", "TRUE")
	}
	if fromID > 0 {
		params.Set("fromId", strconv.FormatInt(fromID, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if orderID > 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	return params, nil
}

// GetMarginAccountAllOCO retrieves all OCO for a specific margin account based on provided optional parameters
func (e *Exchange) GetMarginAccountAllOCO(ctx context.Context, symbol string, isIsolated bool, startTime, endTime time.Time, fromID, limit int64) ([]OCOOrder, error) {
	params, err := ocoOrdersAndTradeParams(symbol, isIsolated, startTime, endTime, 0, fromID, limit)
	if err != nil {
		return nil, err
	}
	var resp []OCOOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/allOrderList", params, getMarginAccountAllOCORate, nil, &resp)
}

// GetMarginAccountsOpenOCOOrder retrieves margin account's open OCO order
func (e *Exchange) GetMarginAccountsOpenOCOOrder(ctx context.Context, isIsolated bool, symbol string) ([]OCOOrder, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if isIsolated {
		params.Set("isIsolated", "TRUE")
	}
	var resp []OCOOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/openOrderList", params, marginAccountOpenOCOOrdersRate, nil, &resp)
}

// GetMarginAccountTradeList retrieves margin accounts trade list
func (e *Exchange) GetMarginAccountTradeList(ctx context.Context, symbol string, isIsolated bool, startTime, endTime time.Time, orderID, fromID, limit int64) ([]TradeHistory, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params, err := ocoOrdersAndTradeParams(symbol, isIsolated, startTime, endTime, orderID, fromID, limit)
	if err != nil {
		return nil, err
	}
	var resp []TradeHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/myTrades", params, marginAccountTradeListRate, nil, &resp)
}

// GetMaxBorrow represents a maximum borrowable amount.
func (e *Exchange) GetMaxBorrow(ctx context.Context, assetName currency.Code, isolatedSymbol string) (*MaxBorrow, error) {
	if assetName.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("asset", assetName.String())
	if isolatedSymbol != "" {
		params.Set("isolatedSymbol", isolatedSymbol)
	}
	var resp *MaxBorrow
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/maxBorrowable", params, marginMaxBorrowRate, nil, &resp)
}

// GetMaxTransferOutAmount retrieves the maximum amount to transfer out of margin account.
// If isolatedSymbol is not sent, crossed margin data will be sent.
func (e *Exchange) GetMaxTransferOutAmount(ctx context.Context, assetName currency.Code, isolatedSymbol string) (float64, error) {
	if assetName.IsEmpty() {
		return 0, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("asset", assetName.String())
	if isolatedSymbol != "" {
		params.Set("isolatedSymbol", isolatedSymbol)
	}
	resp := &struct {
		Amount float64 `json:"amount"`
	}{}
	return resp.Amount, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/maxTransferable", params, maxTransferOutRate, nil, &resp)
}

// GetSummaryOfMarginAccount retrieves a margin account summary
func (e *Exchange) GetSummaryOfMarginAccount(ctx context.Context) (*MarginAccountSummary, error) {
	var resp *MarginAccountSummary
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/tradeCoeff", nil, marginAccountSummaryRate, nil, &resp)
}

// GetIsolatedMarginAccountInfo retrieves isolated margin account info
func (e *Exchange) GetIsolatedMarginAccountInfo(ctx context.Context, symbols []string) (*IsolatedMarginAccountInfo, error) {
	params := url.Values{}
	if len(symbols) > 0 {
		params.Set("symbols", strings.Join(symbols, ","))
	}
	var resp *IsolatedMarginAccountInfo
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/isolated/account", params, getIsolatedMarginAccountInfoRate, nil, &resp)
}

// DisableIsolatedMarginAccount disable isolated margin account for a specific symbol. Each trading pair can only be deactivated once every 24 hours.
func (e *Exchange) DisableIsolatedMarginAccount(ctx context.Context, symbol string) (*IsolatedMarginResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *IsolatedMarginResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, "/sapi/v1/margin/isolated/account", params, deleteIsolatedMarginAccountRate, nil, &resp)
}

// EnableIsolatedMarginAccount enable isolated margin account for a specific symbol(Only supports activation of previously disabled accounts).
func (e *Exchange) EnableIsolatedMarginAccount(ctx context.Context, symbol string) (*IsolatedMarginResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *IsolatedMarginResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/margin/isolated/account", params, enableIsolatedMarginAccountRate, nil, &resp)
}

// GetEnabledIsolatedMarginAccountLimit retrieves enabled isolated margin account limit.
func (e *Exchange) GetEnabledIsolatedMarginAccountLimit(ctx context.Context) (*IsolatedMarginAccountLimit, error) {
	var resp *IsolatedMarginAccountLimit
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/isolated/accountLimit", nil, sapiDefaultRate, nil, &resp)
}

// GetAllIsolatedMarginSymbols retrieves all isolated margin symbols
func (e *Exchange) GetAllIsolatedMarginSymbols(ctx context.Context, symbol string) ([]IsolatedMarginAccount, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []IsolatedMarginAccount
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/isolated/allPairs", params, allIsolatedMarginSymbol, nil, &resp)
}

// ToggleBNBBurn toggles BNB burn on spot trade and margin interest
func (e *Exchange) ToggleBNBBurn(ctx context.Context, spotBNBBurn, interestBNBBurn bool) (*BNBBurnOnSpotAndMarginInterest, error) {
	params := url.Values{}
	if spotBNBBurn {
		params.Set("spotBNBBurn", "true")
	} else {
		params.Set("spotBNBBurn", "false")
	}
	if interestBNBBurn {
		params.Set("interestBNBBurn", "true")
	} else {
		params.Set("interestBNBBurn", "false")
	}
	var resp *BNBBurnOnSpotAndMarginInterest
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/bnbBurn", params, sapiDefaultRate, nil, &resp)
}

// GetBNBBurnStatus retrieves BNB Burn status
func (e *Exchange) GetBNBBurnStatus(ctx context.Context) (*BNBBurnOnSpotAndMarginInterest, error) {
	var resp *BNBBurnOnSpotAndMarginInterest
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/bnbBurn", nil, sapiDefaultRate, nil, &resp)
}

// GetMarginInterestRateHistory retrieves margin interest rate history
func (e *Exchange) GetMarginInterestRateHistory(ctx context.Context, assetName currency.Code, vipLevel int64, startTime, endTime time.Time) ([]MarginInterestRate, error) {
	if assetName.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("asset", assetName.String())
	if vipLevel > 0 {
		params.Set("vipLevel", strconv.FormatInt(vipLevel, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []MarginInterestRate
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/interestRateHistory", params, sapiDefaultRate, nil, &resp)
}

// GetCrossMarginFeeData get cross margin fee data collection with any vip level or user's current specific data as https://www.binance.com/en/margin-fee
func (e *Exchange) GetCrossMarginFeeData(ctx context.Context, vipLevel int64, coin currency.Code) ([]CrossMarginFeeData, error) {
	params := url.Values{}
	if vipLevel > 0 {
		params.Set("vipLevel", strconv.FormatInt(vipLevel, 10))
	}
	endpointLimiter := allCrossMarginFeeDataRate
	if !coin.IsEmpty() {
		endpointLimiter = sapiDefaultRate
		params.Set("coin", coin.String())
	}
	var resp []CrossMarginFeeData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/crossMarginData", params, endpointLimiter, nil, &resp)
}

// GetIsolatedMaringFeeData represents an isolated margin fee data.
// get isolated margin fee data collection with any vip level or user's current specific data as https://www.binance.com/en/margin-fee
func (e *Exchange) GetIsolatedMaringFeeData(ctx context.Context, vipLevel int64, symbol string) ([]IsolatedMarginFeeData, error) {
	endpointLimiter := allIsolatedMarginFeeDataRate
	params := url.Values{}
	if vipLevel > 0 {
		params.Set("vipLevel", strconv.FormatInt(vipLevel, 10))
	}
	if symbol != "" {
		endpointLimiter = sapiDefaultRate
		params.Set("symbol", symbol)
	}
	var resp []IsolatedMarginFeeData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/isolatedMarginData", params, endpointLimiter, nil, &resp)
}

// GetIsolatedMarginTierData get isolated margin tier data collection with any tier as https://www.binance.com/en/margin-data
func (e *Exchange) GetIsolatedMarginTierData(ctx context.Context, symbol string, tier int64) ([]IsolatedMarginTierInfo, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if tier > 0 {
		params.Set("tier", strconv.FormatInt(tier, 10))
	}
	var resp []IsolatedMarginTierInfo
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/isolatedMarginTier", params, sapiDefaultRate, nil, &resp)
}

// GetCurrencyMarginOrderCountUsage displays the user's current margin order count usage for all intervals.
func (e *Exchange) GetCurrencyMarginOrderCountUsage(ctx context.Context, isIsolated bool, symbol string) ([]MarginOrderCount, error) {
	params := url.Values{}
	if isIsolated {
		params.Set("isIsolated", "TRUE")
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []MarginOrderCount
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/rateLimit/order", params, marginCurrentOrderCountUsageRate, nil, &resp)
}

// GetCrossMarginCollateralRatio retrieves collaterals for list of assets.
func (e *Exchange) GetCrossMarginCollateralRatio(ctx context.Context) ([]CrossMarginCollateralRatio, error) {
	var resp []CrossMarginCollateralRatio
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/crossMarginCollateralRatio", crossMarginCollateralRatioRate, &resp)
}

// GetSmallLiabilityExchangeCoinList query the coins which can be small liability exchange
func (e *Exchange) GetSmallLiabilityExchangeCoinList(ctx context.Context) ([]SmallLiabilityCoin, error) {
	var resp []SmallLiabilityCoin
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/exchange-small-liability", nil, getSmallLiabilityExchangeCoinListRate, nil, &resp)
}

// MarginSmallLiabilityExchange set cross-margin small liability exchange
func (e *Exchange) MarginSmallLiabilityExchange(ctx context.Context, assetNames []string) ([]SmallLiabilityCoin, error) {
	if len(assetNames) == 0 {
		return nil, errEmptyCurrencyCodes
	}
	params := url.Values{}
	params.Set("assetNames", strings.Join(assetNames, ","))
	var resp []SmallLiabilityCoin
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/margin/exchange-small-liability", params, smallLiabilityExchangeCoinListRate, nil, &resp)
}

// GetSmallLiabilityExchangeHistory retrieves small liability exchange history
func (e *Exchange) GetSmallLiabilityExchangeHistory(ctx context.Context, current, size int64, startTime, endTime time.Time) (*SmallLiabilityExchange, error) {
	if current <= 0 {
		return nil, fmt.Errorf("%w: current page is empty", errPageNumberRequired)
	}
	if size <= 0 {
		return nil, errPageSizeRequired
	}
	params := url.Values{}
	params.Set("current", strconv.FormatInt(current, 10))
	params.Set("size", strconv.FormatInt(size, 10))
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp *SmallLiabilityExchange
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/exchange-small-liability-history", params, getSmallLiabilityExchangeRate, nil, &resp)
}

// GetFutureHourlyInterestRate retrieves user the next hourly estimate interest
// isIsolated: for isolated margin or not, "TRUE", "FALSE"
func (e *Exchange) GetFutureHourlyInterestRate(ctx context.Context, assets []string, isIsolated bool) ([]HourlyInterestrate, error) {
	params := url.Values{}
	if len(assets) == 0 {
		params.Set("assets", strings.Join(assets, ","))
	}
	if isIsolated {
		params.Set("isIsolated", "TRUE")
	} else {
		params.Set("isIsolated", "FALSE")
	}
	var resp []HourlyInterestrate
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/next-hourly-interest-rate", params, marginHourlyInterestRate, nil, &resp)
}

// GetCrossOrIsolatedMarginCapitalFlow retrieves cross or isolated margin capital flow
// type: represents a capital flow type.
// Supported types:
//
//	TRANSFER("Transfer")
//	BORROW("Borrow")
//	REPAY("Repay")
//	BUY_INCOME("Buy-Trading Income")
//	BUY_EXPENSE("Buy-Trading Expense")
//	SELL_INCOME("Sell-Trading Income")
//	SELL_EXPENSE("Sell-Trading Expense")
//	TRADING_COMMISSION("Trading Commission")
//	BUY_LIQUIDATION("Buy by Liquidation")
//	SELL_LIQUIDATION("Sell by Liquidation")
//	REPAY_LIQUIDATION("Repay by Liquidation")
//	OTHER_LIQUIDATION("Other Liquidation")
//	LIQUIDATION_FEE("Liquidation Fee")
//	SMALL_BALANCE_CONVERT("Small Balance Convert")
//	COMMISSION_RETURN("Commission Return")
//	SMALL_CONVERT("Small Convert")
func (e *Exchange) GetCrossOrIsolatedMarginCapitalFlow(ctx context.Context, assetName currency.Code, symbol, flowType string, startTime, endTime time.Time, fromID, limit int64) ([]MarginCapitalFlow, error) {
	params := url.Values{}
	if !assetName.IsEmpty() {
		params.Set("asset", assetName.String())
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if flowType != "" {
		params.Set("type", flowType)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if fromID > 0 {
		params.Set("fromId", strconv.FormatInt(fromID, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []MarginCapitalFlow
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/capital-flow", params, marginCapitalFlowRate, nil, &resp)
}

// GetTokensOrSymbolsDelistSchedule retrieves tokens or symbols delist schedule for cross-margin and isolated-margin accounts.
func (e *Exchange) GetTokensOrSymbolsDelistSchedule(ctx context.Context) ([]MarginDelistSchedule, error) {
	var resp []MarginDelistSchedule
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/delist-schedule", nil, marginTokensAndSymbolsDelistScheduleRate, nil, &resp)
}

// GetMarginAvailableInventory retrieves margin account available inventory
func (e *Exchange) GetMarginAvailableInventory(ctx context.Context, marginType string) ([]MarginInventory, error) {
	if marginType == "" {
		return nil, margin.ErrInvalidMarginType
	}
	params := url.Values{}
	params.Set("type", marginType)
	var resp []MarginInventory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/available-inventory", params, marginAvailableInventoryRate, nil, &resp)
}

// MarginManualLiquidiation margin manual liquidation
// marginType possible values are 'MARGIN','ISOLATED'
func (e *Exchange) MarginManualLiquidiation(ctx context.Context, marginType, symbol string) ([]SmallLiabilityCoin, error) {
	if marginType == "" {
		return nil, margin.ErrInvalidMarginType
	}
	params := url.Values{}
	params.Set("type", marginType)
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []SmallLiabilityCoin
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/margin/manual-liquidation", params, marginManualLiquidiationRate, nil, &resp)
}

// GetLiabilityCoinLeverageBracketInCrossMarginProMode retrieve liability Coin Leverage Bracket in Cross Margin Pro Mode
func (e *Exchange) GetLiabilityCoinLeverageBracketInCrossMarginProMode(ctx context.Context) ([]LiabilityCoinLeverageBracket, error) {
	var resp []LiabilityCoinLeverageBracket
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/leverageBracket", nil, sapiDefaultRate, nil, &resp)
}

// GetMarginAccount returns account information for margin accounts
func (e *Exchange) GetMarginAccount(ctx context.Context) (*MarginAccount, error) {
	var resp *MarginAccount
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/margin/account", nil, marginAccountInformationRate, nil, &resp)
}

// SendHTTPRequest sends an unauthenticated request
func (e *Exchange) SendHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result any) error {
	endpointPath, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	var responseJSON json.RawMessage
	err = e.SendPayload(ctx, f, func() (*request.Item, error) {
		return &request.Item{
			Method:                 http.MethodGet,
			Path:                   endpointPath + path,
			Result:                 &responseJSON,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.UnauthenticatedRequest)
	if err != nil {
		var errorResponse *ErrResponse
		err = json.Unmarshal(responseJSON, &errorResponse)
		if err == nil && errorResponse.Code.Int64() != 0 {
			errorMsg, okay := errorCodeToErrorMap[errorResponse.Code.Int64()]
			if okay {
				return fmt.Errorf("err %w code: %d msg: %s", errorMsg, errorResponse.Code.Int64(), errorResponse.Message)
			}
			return fmt.Errorf("err code: %d msg: %s", errorResponse.Code.Int64(), errorResponse.Message)
		}
		return err
	}
	err = json.Unmarshal(responseJSON, result)
	if err != nil {
		return err
	}
	if result == nil {
		return common.ErrNoResponse
	}
	return nil
}

// errorCodeToErrorMap represents common error messages.
var errorCodeToErrorMap = map[int64]error{
	-1121: currency.ErrSymbolStringEmpty,
	-1002: request.ErrAuthRequestFailed,
}

// SendAPIKeyHTTPRequest is a special API request where the api key is
// appended to the headers without a secret
func (e *Exchange) SendAPIKeyHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, f request.EndpointLimit, result interface{}) error {
	endpointPath, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	headers := make(map[string]string)
	headers["X-MBX-APIKEY"] = creds.Key
	item := &request.Item{
		Method:                 method,
		Path:                   endpointPath + path,
		Headers:                headers,
		Result:                 result,
		Verbose:                e.Verbose,
		HTTPDebugging:          e.HTTPDebugging,
		HTTPRecording:          e.HTTPRecording,
		HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
	}

	return e.SendPayload(ctx, f, func() (*request.Item, error) {
		return item, nil
	}, request.AuthenticatedRequest)
}

// interfaceToParams convert interface into url.Values instance.
func interfaceToParams(val interface{}) (url.Values, error) {
	dMap := make(map[string]interface{})
	data, err := json.Marshal(val)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &dMap)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	for key, value := range dMap {
		params.Set(key, fmt.Sprintf("%v", value))
	}
	return params, nil
}

// SendAuthHTTPRequest sends an authenticated HTTP request
func (e *Exchange) SendAuthHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, params url.Values, f request.EndpointLimit, arg, result interface{}) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}

	endpointPath, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	if params == nil {
		params = url.Values{}
	}
	if arg != nil && method == http.MethodPost {
		var newParams url.Values
		newParams, err = interfaceToParams(arg)
		if err != nil {
			return err
		}
		for k := range newParams {
			if !params.Has(k) {
				params.Set(k, newParams.Get(k))
			}
		}
	}
	if params.Get("recvWindow") == "" {
		params.Set("recvWindow", strconv.FormatInt(defaultRecvWindow.Milliseconds(), 10))
	}

	interim := json.RawMessage{}
	err = e.SendPayload(ctx, f, func() (*request.Item, error) {
		params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
		hmacSigned, err := crypto.GetHMAC(crypto.HashSHA256, []byte(params.Encode()), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		headers := make(map[string]string)
		headers["X-MBX-APIKEY"] = creds.Key
		fullPath := common.EncodeURLValues(endpointPath+path, params) + "&signature=" + hex.EncodeToString(hmacSigned)
		return &request.Item{
			Method:                 method,
			Path:                   fullPath,
			Headers:                headers,
			Result:                 &interim,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, request.AuthenticatedRequest)
	if err != nil {
		return err
	}
	errCap := struct {
		Success bool   `json:"success"`
		Message string `json:"msg"`
		Code    int64  `json:"code"`
	}{}

	if err := json.Unmarshal(interim, &errCap); err == nil {
		if !errCap.Success && errCap.Message != "" && errCap.Code != 200 {
			return errors.New(errCap.Message)
		}
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(interim, result)
}

// CheckLimit checks value against a variable list
func (e *Exchange) CheckLimit(limit int64) error {
	for x := range e.validLimits {
		if e.validLimits[x] == limit {
			return nil
		}
	}
	return fmt.Errorf("%w: incorrect limit values - valid values are 5, 10, 20, 50, 100, 500, 1000", errLimitNumberRequired)
}

// SetValues sets the default valid values
func (e *Exchange) SetValues() {
	e.validLimits = []int64{5, 10, 20, 50, 100, 500, 1000, 5000}
}

// GetFee returns an estimate of fee based on type of transaction
func (e *Exchange) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		multiplier, err := e.getMultiplier(ctx, feeBuilder.IsMaker)
		if err != nil {
			return 0, err
		}
		fee = calculateTradingFee(feeBuilder.PurchasePrice, feeBuilder.Amount, multiplier)
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

// getMultiplier retrieves account based taker/maker fees
func (e *Exchange) getMultiplier(ctx context.Context, isMaker bool) (float64, error) {
	var multiplier float64
	account, err := e.GetAccount(ctx, false)
	if err != nil {
		return 0, err
	}
	if isMaker {
		multiplier = float64(account.MakerCommission)
	} else {
		multiplier = float64(account.TakerCommission)
	}
	return multiplier, nil
}

// calculateTradingFee returns the fee for trading any currency on Binance
func calculateTradingFee(purchasePrice, amount, multiplier float64) float64 {
	return (multiplier / 100) * purchasePrice * amount
}

// getCryptocurrencyWithdrawalFee returns the fee for withdrawing from the exchange
func getCryptocurrencyWithdrawalFee(c currency.Code) float64 {
	return WithdrawalFees[c]
}

// GetSystemStatus fetch system status.
// 0: normal，1：system maintenance
// "normal", "system_maintenance"
func (e *Exchange) GetSystemStatus(ctx context.Context) (*SystemStatus, error) {
	var resp *SystemStatus
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/sapi/v1/system/status", sapiDefaultRate, &resp)
}

// GetAllCoinsInfo returns details about all supported coins(available for deposit and withdraw)
func (e *Exchange) GetAllCoinsInfo(ctx context.Context) ([]CoinInfo, error) {
	var resp []CoinInfo
	return resp, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpot, http.MethodGet,
		"/sapi/v1/capital/config/getall",
		nil, allCoinInfoRate, nil, &resp)
}

// GetDailyAccountSnapshot retrieves daily account snapshot
func (e *Exchange) GetDailyAccountSnapshot(ctx context.Context, tradeType string, startTime, endTime time.Time, limit int64) (*DailyAccountSnapshot, error) {
	if tradeType == "" {
		return nil, fmt.Errorf("%w type: %s", asset.ErrInvalidAsset, tradeType)
	}
	params := url.Values{}
	params.Set("type", tradeType)
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *DailyAccountSnapshot
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/accountSnapshot", params, dailyAccountSnapshotRate, nil, &resp)
}

// DisableFastWithdrawalSwitch disables fast withdrawal switch
// This request will disable fastwithdraw switch under your account.
// You need to enable "trade" option for the api key which requests this endpoint.
func (e *Exchange) DisableFastWithdrawalSwitch(ctx context.Context) error {
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/account/disableFastWithdrawSwitch", nil, sapiDefaultRate, nil, &struct{}{})
}

// EnableFastWithdrawalSwitch enable fastwithdraw switch under your account.
func (e *Exchange) EnableFastWithdrawalSwitch(ctx context.Context) error {
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/account/enableFastWithdrawSwitch", nil, sapiDefaultRate, nil, &struct{}{})
}

// WithdrawCrypto sends cryptocurrency to the address of your choosing
func (e *Exchange) WithdrawCrypto(ctx context.Context, cryptoAsset currency.Code, withdrawOrderID, network, address, addressTag, name string, amount float64, transactionFeeFlag bool) (string, error) {
	if cryptoAsset.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if address == "" {
		return "", errAddressRequired
	}
	if amount <= 0 {
		return "", limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("coin", cryptoAsset.String())
	params.Set("address", address)
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	// optional params
	if withdrawOrderID != "" {
		params.Set("withdrawOrderId", withdrawOrderID)
	}
	if network != "" {
		params.Set("network", network)
	}
	if addressTag != "" {
		params.Set("addressTag", addressTag)
	}
	if transactionFeeFlag {
		params.Set("transactionFeeFlag", "true")
	}
	if name != "" {
		params.Set("name", url.QueryEscape(name))
	}
	var resp *WithdrawResponse
	return resp.ID, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodPost, "/sapi/v1/capital/withdraw/apply",
		params, fundWithdrawalRate, nil, &resp)
}

// DepositHistory returns the deposit history based on the supplied params
// status `param` used as string to prevent default value 0 (for int) interpreting as EmailSent status
func (e *Exchange) DepositHistory(ctx context.Context, c currency.Code, status string, startTime, endTime time.Time, offset, limit int) ([]DepositHistory, error) {
	params := url.Values{}
	if status != "" {
		i, err := strconv.Atoi(status)
		if err != nil {
			return nil, fmt.Errorf("wrong param (status): %s. Error: %v", status, err)
		}
		switch i {
		case EmailSent, Cancelled, AwaitingApproval, Rejected, Processing, Failure, Completed:
		default:
			return nil, fmt.Errorf("wrong param (status): %s", status)
		}
		params.Set("status", status)
	}
	if !c.IsEmpty() {
		params.Set("coin", c.String())
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UTC().UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UTC().UnixMilli(), 10))
	}
	if offset != 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if limit != 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	var response []DepositHistory
	return response, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpot, http.MethodGet,
		"/sapi/v1/capital/deposit/hisrec",
		params, sapiDefaultRate, nil, &response)
}

// WithdrawHistory gets the status of recent withdrawals
// status `param` used as string to prevent default value 0 (for int) interpreting as EmailSent status
func (e *Exchange) WithdrawHistory(ctx context.Context, c currency.Code, status string, startTime, endTime time.Time, offset, limit int) ([]WithdrawStatusResponse, error) {
	params := url.Values{}
	if !c.IsEmpty() {
		params.Set("coin", c.String())
	}

	if status != "" {
		i, err := strconv.Atoi(status)
		if err != nil {
			return nil, fmt.Errorf("wrong param (status): %s. Error: %v", status, err)
		}

		switch i {
		case EmailSent, Cancelled, AwaitingApproval, Rejected, Processing, Failure, Completed:
		default:
			return nil, fmt.Errorf("wrong param (status): %s", status)
		}
		params.Set("status", status)
	}

	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UTC().UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UTC().UnixMilli(), 10))
	}
	if offset != 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if limit != 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	var withdrawStatus []WithdrawStatusResponse
	return withdrawStatus, e.SendAuthHTTPRequest(ctx,
		exchange.RestSpot, http.MethodGet,
		"/sapi/v1/capital/withdraw/history", params, withdrawalHistoryRate, nil, &withdrawStatus)
}

// GetDepositAddressForCurrency retrieves the wallet address for a given currency
func (e *Exchange) GetDepositAddressForCurrency(ctx context.Context, coin currency.Code, chain string) (*DepositAddress, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", coin.String())
	if chain != "" {
		params.Set("network", chain)
	}
	params.Set("recvWindow", "10000")
	var d *DepositAddress
	return d, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/capital/deposit/address", params, depositAddressesRate, nil, &d)
}

// GetAssetsThatCanBeConvertedIntoBNB retrieves assets that can be converted into BNB
func (e *Exchange) GetAssetsThatCanBeConvertedIntoBNB(ctx context.Context, accountType string) (*AssetsDust, error) {
	params := url.Values{}
	if accountType != "" {
		params.Set("accountType", accountType)
	}
	var resp *AssetsDust
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/asset/dust-btc", params, sapiDefaultRate, nil, &resp)
}

// DustTransfer convert dust assets to BNB.
func (e *Exchange) DustTransfer(ctx context.Context, assets []string, accountType string) (*Dusts, error) {
	if len(assets) == 0 {
		return nil, fmt.Errorf("%w: assets must not be empty", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("assets", strings.Join(assets, ","))
	if accountType != "" {
		params.Set("accountType", accountType)
	}
	var resp *Dusts
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/asset/dust", params, dustTransferRate, nil, &resp)
}

// GetAssetDevidendRecords query asset dividend record.
func (e *Exchange) GetAssetDevidendRecords(ctx context.Context, ccy currency.Code, startTime, endTime time.Time, limit int64) (interface{}, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("asset", ccy.String())
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *AssetDividendRecord
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/asset/assetDividend", params, assetDividendRecordRate, nil, &resp)
}

// GetAssetDetail fetches details of assets supported on Binance
func (e *Exchange) GetAssetDetail(ctx context.Context) (map[string]DividendAsset, error) {
	var resp map[string]DividendAsset
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/asset/assetDetail", nil, sapiDefaultRate, nil, &resp)
}

// GetTradeFees fetch trade fee
func (e *Exchange) GetTradeFees(ctx context.Context, symbol currency.Pair) ([]TradeFee, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	var resp []TradeFee
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/asset/tradeFee", params, sapiDefaultRate, nil, &resp)
}

// UserUniversalTransfer transfers an asset
// You need to enable Permits Universal Transfer option for the API Key which requests this endpoint.
// fromSymbol must be sent when type are ISOLATEDMARGIN_MARGIN and ISOLATEDMARGIN_ISOLATEDMARGIN
// toSymbol must be sent when type are MARGIN_ISOLATEDMARGIN and ISOLATEDMARGIN_ISOLATEDMARGIN
func (e *Exchange) UserUniversalTransfer(ctx context.Context, transferType TransferTypes, amount float64, ccy currency.Code, fromSymbol, toSymbol string) (string, error) {
	if transferType == 0 {
		return "", errTransferTypeRequired
	}
	if ccy.IsEmpty() {
		return "", fmt.Errorf("asset %w", currency.ErrCurrencyCodeEmpty)
	}
	if amount <= 0 {
		return "", limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	params.Set("type", transferType.String())
	params.Set("asset", ccy.String())
	if fromSymbol == "" {
		params.Set("fromSymbol", fromSymbol)
	}
	if toSymbol == "" {
		params.Set("toSymbol", toSymbol)
	}
	resp := &struct {
		TransferID string `json:"tranId"`
	}{}
	return resp.TransferID, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/asset/transfer", params, userUniversalTransferRate, nil, &resp)
}

// GetUserUniversalTransferHistory retrieves user universal transfer history
func (e *Exchange) GetUserUniversalTransferHistory(ctx context.Context, transferType TransferTypes, startTime, endTime time.Time, current, size int64, fromSymbol, toSymbol string) (*UniversalTransferHistory, error) {
	if transferType == 0 {
		return nil, errTransferTypeRequired
	}
	if size <= 0 {
		return nil, fmt.Errorf("%w: 'size' is required", limits.ErrAmountBelowMin)
	}
	params := url.Values{}
	params.Set("type", transferType.String())
	params.Set("size", strconv.FormatInt(size, 10))
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if current != 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if fromSymbol != "" {
		params.Set("fromSymbol", fromSymbol)
	}
	if toSymbol != "" {
		params.Set("toSymbol", toSymbol)
	}
	var resp *UniversalTransferHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/asset/transfer", params, sapiDefaultRate, nil, &resp)
}

// GetFundingAssets funding wallet
func (e *Exchange) GetFundingAssets(ctx context.Context, ccy currency.Code, needBTCValuation bool) ([]FundingAsset, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("asset", ccy.String())
	}
	if needBTCValuation {
		params.Set("needBtcValuation", "true")
	}
	var resp []FundingAsset
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/asset/get-funding-asset", params, sapiDefaultRate, nil, &resp)
}

// GetUserAssets get user assets, just for positive data.
func (e *Exchange) GetUserAssets(ctx context.Context, ccy currency.Code, needBTCValuation bool) ([]FundingAsset, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("asset", ccy.String())
	}
	if needBTCValuation {
		params.Set("needBtcValuation", "true")
	}
	var resp []FundingAsset
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v3/asset/getUserAsset", params, userAssetsRate, nil, &resp)
}

// ConvertBUSD convert transfer, convert between BUSD and stablecoins.
// accountType: possible values are MAIN and CARD
func (e *Exchange) ConvertBUSD(ctx context.Context, clientTransactionID, accountType string, assetCcy, targetAsset currency.Code, amount float64) (*AssetConverResponse, error) {
	if clientTransactionID == "" {
		return nil, errTransactionIDRequired
	}
	if assetCcy.IsEmpty() {
		return nil, fmt.Errorf("%w assetCcy is empty", currency.ErrCurrencyCodeEmpty)
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if targetAsset.IsEmpty() {
		return nil, fmt.Errorf("%w targetAsset is empty", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("clientTranId", clientTransactionID)
	params.Set("asset", assetCcy.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	params.Set("targetAsset", targetAsset.String())
	if accountType != "" {
		params.Set("accountType", accountType)
	}
	var resp *AssetConverResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/asset/convert-transfer", params, busdConvertRate, nil, &resp)
}

// BUSDConvertHistory convert transfer, convert between BUSD and stablecoins.
func (e *Exchange) BUSDConvertHistory(ctx context.Context, transactionID, clientTransactionID, accountType string, assetCcy currency.Code, startTime, endTime time.Time, current, size int64) (*BUSDConvertHistory, error) {
	err := common.StartEndTimeCheck(startTime, endTime)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	if transactionID != "" {
		params.Set("tranid", transactionID)
	}
	if clientTransactionID != "" {
		params.Set("clientTranId", clientTransactionID)
	}
	if !assetCcy.IsEmpty() {
		params.Set("asset", assetCcy.String())
	}
	if accountType != "" {
		params.Set("accountType", accountType)
	}
	if current > 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	var resp *BUSDConvertHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/asset/convert-transfer/queryByPage", params, busdConvertHistoryRate, nil, &resp)
}

// GetCloudMiningPaymentAndRefundHistory retrieves cloud-mining payment and refund history
func (e *Exchange) GetCloudMiningPaymentAndRefundHistory(ctx context.Context, clientTransactionID string, assetCcy currency.Code, startTime, endTime time.Time, transactionID, size, current int64) (*CloudMiningPR, error) {
	err := common.StartEndTimeCheck(startTime, endTime)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	if transactionID != 0 {
		params.Set("tranId", strconv.FormatInt(transactionID, 10))
	}
	if clientTransactionID != "" {
		params.Set("clientTranId", clientTransactionID)
	}
	if !assetCcy.IsEmpty() {
		params.Set("asset", assetCcy.String())
	}
	if current > 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	var resp *CloudMiningPR
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/asset/ledger-transfer/cloud-mining/queryByPage", params, cloudMiningPaymentAndRefundHistoryRate, nil, &resp)
}

// GetUserAccountInfo retrieves users account information
func (e *Exchange) GetUserAccountInfo(ctx context.Context) (interface{}, error) {
	var resp *AccountInfo
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/account/info", nil, request.Auth, nil, &resp)
}

// GetAPIKeyPermission retrieves API key ermissions detail.
func (e *Exchange) GetAPIKeyPermission(ctx context.Context) (*APIKeyPermissions, error) {
	var resp *APIKeyPermissions
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/account/apiRestrictions", nil, sapiDefaultRate, nil, &resp)
}

// GetAutoConvertingStableCoins a user's auto-conversion settings in deposit/withdrawal
func (e *Exchange) GetAutoConvertingStableCoins(ctx context.Context) (*AutoConvertingStableCoins, error) {
	var resp *AutoConvertingStableCoins
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/capital/contract/convertible-coins", nil, autoConvertingStableCoinsRate, nil, &resp)
}

// SwitchOnOffBUSDAndStableCoinsConversion user can use it to turn on or turn off the BUSD auto-conversion from/to a specific stable coin.
func (e *Exchange) SwitchOnOffBUSDAndStableCoinsConversion(ctx context.Context, coin currency.Code, enable bool) error {
	if coin.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", coin.String())
	params.Set("enable", strconv.FormatBool(enable))
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/capital/contract/convertible-coins", params, autoConvertingStableCoinsRate, nil, &struct{}{})
}

// OneClickArrivalDepositApply apply deposit credit for expired address
func (e *Exchange) OneClickArrivalDepositApply(ctx context.Context, transactionID string, subAccountID, subUserID, depositID int64) (bool, error) {
	params := map[string]string{}
	if transactionID != "" {
		params["txId"] = transactionID
	}
	if depositID != 0 {
		params["depositId"] = strconv.FormatInt(depositID, 10)
	}
	if subAccountID != 0 {
		params["subAccountId"] = strconv.FormatInt(subAccountID, 10)
	}
	if subUserID != 0 {
		params["subUserID"] = strconv.FormatInt(subUserID, 10)
	}
	var resp bool
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/capital/deposit/credit-apply", nil, sapiDefaultRate, params, &resp)
}

// GetDepositAddressListWithNetwork fetch deposit address list with network.
func (e *Exchange) GetDepositAddressListWithNetwork(ctx context.Context, coin currency.Code, network string) ([]DepositAddressAndNetwork, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", coin.String())
	if network != "" {
		params.Set("network", network)
	}
	var resp []DepositAddressAndNetwork
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/capital/deposit/address/list", params, getDepositAddressListInNetworkRate, nil, &resp)
}

// GetUserWalletBalance retrieves user wallet balance.
func (e *Exchange) GetUserWalletBalance(ctx context.Context, quoteAsset currency.Code) ([]UserWalletBalance, error) {
	params := url.Values{}
	if quoteAsset.IsEmpty() {
		params.Set("quoteAsset", quoteAsset.String())
	}
	var resp []UserWalletBalance
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/asset/wallet/balance", params, getUserWalletBalanceRate, nil, &resp)
}

// GetUserDelegationHistory query User Delegation History for Master account.
// The delegation type has two values: delegated or undelegated.
func (e *Exchange) GetUserDelegationHistory(ctx context.Context, email, delegation string, startTime, endTime time.Time, ccy currency.Code, current int64, size float64) (*UserDelegationHistory, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	err := common.StartEndTimeCheck(startTime, endTime)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("email", email)
	params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	if !ccy.IsEmpty() {
		params.Set("asset", ccy.String())
	}
	if delegation != "" {
		params.Set("type", delegation)
	}
	if current != 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if size != 0 {
		params.Set("size", strconv.FormatFloat(size, 'f', -1, 64))
	}
	var resp *UserDelegationHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/asset/custody/transfer-history", params, getUserDelegationHistoryRate, nil, &resp)
}

// GetSymbolsDelistScheduleForSpot symbols delist schedule for spot
func (e *Exchange) GetSymbolsDelistScheduleForSpot(ctx context.Context) ([]DelistSchedule, error) {
	var resp []DelistSchedule
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/spot/delist-schedule", nil, symbolDelistScheduleForSpotRate, nil, &resp)
}

// GetWithdrawAddressList retrieves withdraw address list
func (e *Exchange) GetWithdrawAddressList(ctx context.Context) ([]WithdrawAddress, error) {
	var resp []WithdrawAddress
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/capital/withdraw/address/list", nil, withdrawAddressListRate, nil, &resp)
}

// --------------------------------------------------  Sub-Account Endpoints  --------------------------------------------------------------

// CreateVirtualSubAccount creates a virtual subaccount information.
func (e *Exchange) CreateVirtualSubAccount(ctx context.Context, subAccountString string) (*VirtualSubAccount, error) {
	params := url.Values{}
	if subAccountString != "" {
		params.Set("subAccountString", subAccountString)
	}
	var resp *VirtualSubAccount
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/sub-account/virtualSubAccount", params, sapiDefaultRate, nil, &resp)
}

// GetSubAccountList retrieves sub-account list for Master Account
func (e *Exchange) GetSubAccountList(ctx context.Context, email string, isFreeze bool, page, limit int64) (*SubAccountList, error) {
	params := url.Values{}
	if common.MatchesEmailPattern(email) {
		params.Set("email", email)
	}
	if isFreeze {
		params.Set("isFreeze", "true")
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *SubAccountList
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/list", params, sapiDefaultRate, nil, &resp)
}

// GetSubAccountSpotAssetTransferHistory represents sub-account spot asset transfer history for master account
func (e *Exchange) GetSubAccountSpotAssetTransferHistory(ctx context.Context, fromEmail, toEmail string, startTime, endTime time.Time, page, limit int64) ([]SubAccountSpotAsset, error) {
	params := url.Values{}
	if common.MatchesEmailPattern(fromEmail) {
		params.Set("fromEmail", fromEmail)
	}
	if common.MatchesEmailPattern(toEmail) {
		params.Set("toEmail", toEmail)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []SubAccountSpotAsset
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/sub/transfer/history", params, sapiDefaultRate, nil, &resp)
}

// GetSubAccountFuturesAssetTransferHistory Query Sub-account Futures Asset Transfer History For Master Account
func (e *Exchange) GetSubAccountFuturesAssetTransferHistory(ctx context.Context, email string, startTime, endTime time.Time, futuresType, page, limit int64) (*AssetTransferHistory, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	if futuresType != 1 && futuresType != 2 {
		return nil, errInvalidFuturesType
	}
	params := url.Values{}
	params.Set("email", email)
	params.Set("futuresType", strconv.FormatInt(futuresType, 10))
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page != 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *AssetTransferHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/futures/internalTransfer", params, sapiDefaultRate, nil, &resp)
}

// SubAccountFuturesAssetTransfer sub-account futures asset transfer for master account
// futuresType: 1:USDT-margined Futures，2: Coin-margined Futures
func (e *Exchange) SubAccountFuturesAssetTransfer(ctx context.Context, fromEmail, toEmail string, futuresType int64, ccy currency.Code, amount float64) (*FuturesAssetTransfer, error) {
	if !common.MatchesEmailPattern(fromEmail) {
		return nil, fmt.Errorf("%w: fromEmail=%s", errValidEmailRequired, fromEmail)
	}
	if !common.MatchesEmailPattern(toEmail) {
		return nil, fmt.Errorf("%w: toEmail=%s", errValidEmailRequired, toEmail)
	}
	if futuresType != 0 && futuresType != 1 {
		return nil, fmt.Errorf("%w 1: USDT-margined Futures or 2: Coin-margined Futures", errInvalidFuturesType)
	}
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("fromEmail", fromEmail)
	params.Set("toEmail", toEmail)
	params.Set("futuresType", strconv.FormatInt(futuresType, 10))
	params.Set("asset", ccy.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *FuturesAssetTransfer
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/sub-account/futures/internalTransfer", params, sapiDefaultRate, nil, &resp)
}

// GetSubAccountAssets sub-account assets for master account
func (e *Exchange) GetSubAccountAssets(ctx context.Context, email string) (*SubAccountAssets, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	var resp *SubAccountAssets
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v4/sub-account/assets", params, getSubAccountAssetRate, nil, &resp)
}

// GetManagedSubAccountList retrieves investor's managed sub-account list.
func (e *Exchange) GetManagedSubAccountList(ctx context.Context, email string, page, limit int64) (*ManagedSubAccountList, error) {
	params := url.Values{}
	if common.MatchesEmailPattern(email) {
		params.Set("email", email)
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *ManagedSubAccountList
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/managed-subaccount/info", params, getManagedSubAccountListRate, nil, &resp)
}

// GetSubAccountTransactionStatistics retrieves sub-account Transaction statistics (For Master Account)(USER_DATA).
func (e *Exchange) GetSubAccountTransactionStatistics(ctx context.Context, email string) ([]SubAccountTransactionStatistics, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	var resp []SubAccountTransactionStatistics
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/transaction-statistics", params, getSubAccountTransactionStatisticsRate, nil, &resp)
}

// GetManagedSubAccountDepositAddress retrieves investor's managed sub-account deposit address.
// get managed sub-account deposit address (For Investor Master Account) (USER_DATA)
// network: can be found in this endpoint /sapi/v1/capital/deposit/address
func (e *Exchange) GetManagedSubAccountDepositAddress(ctx context.Context, coin currency.Code, email, network string) (*ManagedSubAccountDepositAddres, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", coin.String())
	params.Set("email", email)
	if network != "" {
		params.Set("network", network)
	}
	var resp *ManagedSubAccountDepositAddres
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/managed-subaccount/deposit/address", params, sapiDefaultRate, nil, &resp)
}

// EnableOptionsForSubAccount enables options for sub-account(For master account)
func (e *Exchange) EnableOptionsForSubAccount(ctx context.Context, email string) (*OptionsEnablingResponse, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	var resp *OptionsEnablingResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/sub-account/eoptions/enable", params, sapiDefaultRate, nil, &resp)
}

// GetManagedSubAccountTransferLog retrieves managed sub account transfer Log (For Trading Team Sub Account)
// transfers: Transfer Direction (FROM/TO)
// transferFunctionAccountType: Transfer function account type (SPOT/MARGIN/ISOLATED_MARGIN/USDT_FUTURE/COIN_FUTURE)
func (e *Exchange) GetManagedSubAccountTransferLog(ctx context.Context, startTime, endTime time.Time, page, limit int64, transfers, transferFunctionAccountType string) (*ManagedSubAccountTransferLog, error) {
	err := common.StartEndTimeCheck(startTime, endTime)
	if err != nil {
		return nil, err
	}
	if page < 0 {
		return nil, errPageNumberRequired
	}
	if limit < 0 {
		return nil, errLimitNumberRequired
	}
	params := url.Values{}
	params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	params.Set("page", strconv.FormatInt(page, 10))
	params.Set("limit", strconv.FormatInt(limit, 10))
	if transfers != "" {
		params.Set("transfers", transfers)
	}
	if transferFunctionAccountType != "" {
		params.Set("transferFunctionAccountType", transferFunctionAccountType)
	}
	var resp *ManagedSubAccountTransferLog
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/managed-subaccount/query-trans-log", params, managedSubAccountTransferLogRate, nil, &resp)
}

// GetSubAccountSpotAssetsSummary retrieves BTC valued asset summary of subaccounts.
func (e *Exchange) GetSubAccountSpotAssetsSummary(ctx context.Context, email string, page, size int64) (*SubAccountSpotSummary, error) {
	params := url.Values{}
	if common.MatchesEmailPattern(email) {
		params.Set("email", email)
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	var resp *SubAccountSpotSummary
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/spotSummary", params, sapiDefaultRate, nil, &resp)
}

// GetSubAccountDepositAddress sub-account deposit address
func (e *Exchange) GetSubAccountDepositAddress(ctx context.Context, email, coin, network string, amount float64) (*SubAccountDepositAddress, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	if coin == "" {
		return nil, fmt.Errorf("%w: coin=%s", currency.ErrCurrencyCodeEmpty, coin)
	}
	params := url.Values{}
	params.Set("email", email)
	params.Set("coin", coin)
	if network != "" {
		params.Set("network", network)
	}
	if amount > 0 {
		params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	}
	var resp *SubAccountDepositAddress
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/capital/deposit/subAddress", params, sapiDefaultRate, nil, &resp)
}

// GetSubAccountDepositHistory retrieves sub-account deposit history
func (e *Exchange) GetSubAccountDepositHistory(ctx context.Context, email, coin string, startTime, endTime time.Time, status, offset, limit int64) (*SubAccountDepositHistory, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if coin != "" {
		params.Set("coin", coin)
	}
	if status != 0 {
		params.Set("status", strconv.FormatInt(status, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	var resp *SubAccountDepositHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/capital/deposit/subHisrec", params, sapiDefaultRate, nil, &resp)
}

// GetSubAccountStatusOnMarginFutures sub-account's status on Margin/Futures for master account
func (e *Exchange) GetSubAccountStatusOnMarginFutures(ctx context.Context, email string) (*SubAccountStatus, error) {
	params := url.Values{}
	if common.MatchesEmailPattern(email) {
		params.Set("email", email)
	}
	var resp *SubAccountStatus
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/status", params, getSubAccountStatusOnMarginOrFuturesRate, nil, &resp)
}

// EnableMarginForSubAccount Enable Margin for Sub-account For Master Account
func (e *Exchange) EnableMarginForSubAccount(ctx context.Context, email string) (*MarginEnablingResponse, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	var resp *MarginEnablingResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/sub-account/margin/enable", params, sapiDefaultRate, nil, &resp)
}

// GetDetailOnSubAccountMarginAccount retrieves Detail on Sub-account's Margin Account For Master Account
func (e *Exchange) GetDetailOnSubAccountMarginAccount(ctx context.Context, email string) (*SubAccountMarginAccountDetail, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	var resp *SubAccountMarginAccountDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/margin/account", params, subAccountMarginAccountDetailRate, nil, &resp)
}

// GetSummaryOfSubAccountMarginAccount retrieves summary of sub-account's margin account for master account
func (e *Exchange) GetSummaryOfSubAccountMarginAccount(ctx context.Context) (*SubAccountMarginAccount, error) {
	var resp *SubAccountMarginAccount
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/margin/accountSummary", nil, getSubAccountSummaryOfMarginAccountRate, nil, &resp)
}

// EnableFuturesSubAccount enables futures for Sub-account for master account
func (e *Exchange) EnableFuturesSubAccount(ctx context.Context, email string) (*FuturesEnablingResponse, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	var resp *FuturesEnablingResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/futures/enable", params, sapiDefaultRate, nil, &resp)
}

// GetDetailSubAccountFuturesAccount retrieves detail on sub-account's futures account for master account
func (e *Exchange) GetDetailSubAccountFuturesAccount(ctx context.Context, email string) (*SubAccountsFuturesAccount, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	var resp *SubAccountsFuturesAccount
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/futures/account", params, getDetailSubAccountFuturesAccountRate, nil, &resp)
}

// GetSummaryOfSubAccountFuturesAccount retrieves summary of sub-account's futures account for master account
func (e *Exchange) GetSummaryOfSubAccountFuturesAccount(ctx context.Context) (*SubAccountsFuturesAccount, error) {
	var resp *SubAccountsFuturesAccount
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/futures/accountSummary", nil, sapiDefaultRate, nil, &resp)
}

// GetV1FuturesPositionRiskSubAccount retrieves V1 position-risk of sub-account's futures account.
func (e *Exchange) GetV1FuturesPositionRiskSubAccount(ctx context.Context, email string) (*SubAccountFuturesPositionRisk, error) {
	return e.getFuturesPositionRiskSubAccount(ctx, email, "/sapi/v1/sub-account/futures/positionRisk", -1, getFuturesPositionRiskOfSubAccountV1Rate)
}

// GetV2FuturesPositionRiskSubAccount retrieves futures position-risk of sub-account for master account
func (e *Exchange) GetV2FuturesPositionRiskSubAccount(ctx context.Context, email string, futuresType int64) (*SubAccountFuturesPositionRisk, error) {
	return e.getFuturesPositionRiskSubAccount(ctx, email, "/sapi/v2/sub-account/futures/positionRisk", futuresType, sapiDefaultRate)
}

func (e *Exchange) getFuturesPositionRiskSubAccount(ctx context.Context, email, path string, futuresType int64, endpointLimit request.EndpointLimit) (*SubAccountFuturesPositionRisk, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	if futuresType < 0 {
		return nil, errInvalidFuturesType
	}
	params := url.Values{}
	params.Set("email", email)
	var resp *SubAccountFuturesPositionRisk
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, endpointLimit, nil, &resp)
}

// EnableLeverageTokenForSubAccount enables leverage token sub-account form master account.
func (e *Exchange) EnableLeverageTokenForSubAccount(ctx context.Context, email string, enableElvt bool) (*LeverageToken, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	if enableElvt {
		params.Set("enableElvt", "true")
	} else {
		params.Set("enableElvt", "false")
	}
	var resp *LeverageToken
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/sub-account/blvt/enable", params, sapiDefaultRate, nil, &resp)
}

// GetIPRestrictionForSubAccountAPIKeyV2 retrieves list of IP addresses restricted for the sub account API key(for master account).
func (e *Exchange) GetIPRestrictionForSubAccountAPIKeyV2(ctx context.Context, email, subAccountAPIKey string) (*APIRestrictions, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	if subAccountAPIKey == "" {
		return nil, errEmptySubAccountAPIKey
	}
	params := url.Values{}
	params.Set("email", email)
	params.Set("subAccountApiKey", subAccountAPIKey)
	var resp *APIRestrictions
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/subAccountApi/ipRestriction", params, ipRestrictionForSubAccountAPIKeyRate, nil, &resp)
}

// DeleteIPListForSubAccountAPIKey delete IP list for a sub-account API key (For Master Account)
func (e *Exchange) DeleteIPListForSubAccountAPIKey(ctx context.Context, email, subAccountAPIKey, ipAddress string) (*APIRestrictions, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	if subAccountAPIKey == "" {
		return nil, errEmptySubAccountAPIKey
	}
	params := url.Values{}
	params.Set("email", email)
	params.Set("subAccountApiKey", subAccountAPIKey)
	if ipAddress != "" {
		params.Set("ipAddress", ipAddress)
	}
	var resp *APIRestrictions
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, "/sapi/v1/sub-account/subAccountApi/ipRestriction/ipList", params, deleteIPListForSubAccountAPIKeyRate, nil, &resp)
}

// AddIPRestrictionForSubAccountAPIkey adds an IP address into the restricted IP addresses for the subaccount
func (e *Exchange) AddIPRestrictionForSubAccountAPIkey(ctx context.Context, email, subAccountAPIKey, ipAddress string, restricted bool) (*APIRestrictions, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	if subAccountAPIKey == "" {
		return nil, errEmptySubAccountAPIKey
	}
	params := url.Values{}
	if restricted {
		params.Set("status", "2")
	} else {
		params.Set("status", "1")
	}
	params.Set("email", email)
	params.Set("subAccountApiKey", subAccountAPIKey)
	if ipAddress != "" {
		params.Set("ipAddress", ipAddress)
	}
	var resp *APIRestrictions
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v2/sub-account/subAccountApi/ipRestriction", params, addIPRestrictionSubAccountAPIKey, nil, &resp)
}

// DepositAssetsIntoTheManagedSubAccount deposits an asset into managed sub-account (for investor master account).
func (e *Exchange) DepositAssetsIntoTheManagedSubAccount(ctx context.Context, toEmail string, ccy currency.Code, amount float64) (string, error) {
	if !common.MatchesEmailPattern(toEmail) {
		return "", fmt.Errorf("%w: toEmail = %s", errValidEmailRequired, toEmail)
	}
	if ccy.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return "", limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("toEmail", toEmail)
	params.Set("asset", ccy.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	resp := &struct {
		TransactionID string `json:"tranId"`
	}{}
	return resp.TransactionID, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/managed-subaccount/deposit", params, sapiDefaultRate, nil, &resp)
}

// GetManagedSubAccountAssetsDetails retrieves managed sub-account assets details for investor master accounts.
func (e *Exchange) GetManagedSubAccountAssetsDetails(ctx context.Context, email string) ([]ManagedSubAccountAssetInfo, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	var resp []ManagedSubAccountAssetInfo
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/managed-subaccount/asset", params, sapiDefaultRate, nil, &resp)
}

// WithdrawAssetsFromManagedSubAccount withdraws an asset from managed sub-account(for investor master account).
func (e *Exchange) WithdrawAssetsFromManagedSubAccount(ctx context.Context, fromEmail string, ccy currency.Code, amount float64, transferDate time.Time) (string, error) {
	if !common.MatchesEmailPattern(fromEmail) {
		return "", fmt.Errorf("%w fromEmail=%s", errValidEmailRequired, fromEmail)
	}
	if ccy.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return "", limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("fromEmail", fromEmail)
	params.Set("asset", ccy.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	if !transferDate.IsZero() {
		params.Set("transferData", strconv.FormatInt(transferDate.UnixMilli(), 10))
	}
	resp := &struct {
		TransactionID string `json:"tranId"`
	}{}
	return resp.TransactionID, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/managed-subaccount/withdraw", params, sapiDefaultRate, nil, &resp)
}

// GetManagedSubAccountSnapshot retrieves managed sub-account snapshot for investor master account.
// assetType possible values: "SPOT", "MARGIN"（cross）, "FUTURES"（UM)
func (e *Exchange) GetManagedSubAccountSnapshot(ctx context.Context, email, assetType string, startTime, endTime time.Time, limit int64) (*SubAccountAssetsSnapshot, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, fmt.Errorf("%w email=%s", errValidEmailRequired, email)
	}
	if assetType == "" {
		return nil, asset.ErrInvalidAsset
	}
	params := url.Values{}
	params.Set("email", email)
	params.Set("type", assetType)
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *SubAccountAssetsSnapshot
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/managed-subaccount/accountSnapshot", params, getManagedSubAccountSnapshotRate, nil, &resp)
}

// GetManagedSubAccountTransferLogForInvestorMasterAccount retrieves managed sub account transfer log. This endpoint is available for investor of Managed Sub-Account.
// A Managed Sub-Account is an account type for investors who value flexibility in asset allocation and account application,
// while delegating trades to a professional trading team.
func (e *Exchange) GetManagedSubAccountTransferLogForInvestorMasterAccount(ctx context.Context, email, transfers, transferFunctionAccountType string, startTime, endTime time.Time, page, limit int64) (*SubAccountTransferLog, error) {
	return e.getManagedSubAccountTransferLog(ctx, email, transfers, transferFunctionAccountType, "/sapi/v1/managed-subaccount/queryTransLogForInvestor", startTime, endTime, page, limit, sapiDefaultRate)
}

// GetManagedSubAccountTransferLogForTradingTeam retrieves managed sub account transfer log.
// This endpoint is available for investor of Managed Sub-Account. A Managed Sub-Account is an account type for investors who value flexibility in asset allocation and account application,
// while delegating trades to a professional trading team.
// transfers: Transfer Direction (FROM/TO)
// transferFunctionAccountType: Transfer function account type (SPOT/MARGIN/ISOLATED_MARGIN/USDT_FUTURE/COIN_FUTURE)
func (e *Exchange) GetManagedSubAccountTransferLogForTradingTeam(ctx context.Context, email, transfers, transferFunctionAccountType string, startTime, endTime time.Time, page, limit int64) (*SubAccountTransferLog, error) {
	return e.getManagedSubAccountTransferLog(ctx, email, transfers, transferFunctionAccountType, "/sapi/v1/managed-subaccount/queryTransLogForTradeParent", startTime, endTime, page, limit, managedSubAccountTransferLogRate)
}

func (e *Exchange) getManagedSubAccountTransferLog(ctx context.Context, email, transfers, transferFunctionAccountType, path string, startTime, endTime time.Time, page, limit int64, epl request.EndpointLimit) (*SubAccountTransferLog, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, fmt.Errorf("%w: email = %s", errValidEmailRequired, email)
	}
	err := common.StartEndTimeCheck(startTime, endTime)
	if err != nil {
		return nil, err
	}
	if page < 0 {
		return nil, errPageNumberRequired
	}
	if limit <= 0 {
		return nil, errLimitNumberRequired
	}
	params := url.Values{}
	params.Set("email", email)
	params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	params.Set("page", strconv.FormatInt(page, 10))
	params.Set("limit", strconv.FormatInt(limit, 10))
	if transfers != "" {
		params.Set("transfers", transfers)
	}
	if transferFunctionAccountType != "" {
		params.Set("transferFunctionAccountType", transferFunctionAccountType)
	}
	var resp *SubAccountTransferLog
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, epl, nil, &resp)
}

// GetManagedSubAccountFutureesAssetDetails retrieves managed sub account futures asset details(For Investor Master Account）(USER_DATA)
func (e *Exchange) GetManagedSubAccountFutureesAssetDetails(ctx context.Context, email string) (*ManagedSubAccountFuturesAssetDetail, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	var resp *ManagedSubAccountFuturesAssetDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/managed-subaccount/fetch-future-asset", params, managedSubAccountFuturesAssetDetailRate, nil, &resp)
}

// GetManagedSubAccountMarginAssetDetails retrieves managed sub-account margin asset details.
func (e *Exchange) GetManagedSubAccountMarginAssetDetails(ctx context.Context, email string) (*SubAccountMarginAsset, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	var resp *SubAccountMarginAsset
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/managed-subaccount/marginAsset", params, sapiDefaultRate, nil, &resp)
}

// FuturesTransferSubAccount transfers futures for sub-account( from master account only)
// 1: transfer from subaccount's spot account to its USDT-margined futures account 2: transfer from subaccount's USDT-margined futures account to its spot account
// 3: transfer from subaccount's spot account to its COIN-margined futures account 4:transfer from subaccount's COIN-margined futures account to its spot account
func (e *Exchange) FuturesTransferSubAccount(ctx context.Context, email string, ccy currency.Code, amount float64, transferType int64) (string, error) {
	return e.transferSubAccount(ctx, email, "/sapi/v1/sub-account/futures/transfer", ccy, amount, transferType)
}

// MarginTransferForSubAccount margin Transfer for Sub-account (For Master Account)
// transferType: 1: transfer from subaccount's spot account to margin account 2: transfer from subaccount's margin account to its spot account
func (e *Exchange) MarginTransferForSubAccount(ctx context.Context, email string, ccy currency.Code, amount float64, transferType int64) (string, error) {
	return e.transferSubAccount(ctx, email, "/sapi/v1/sub-account/margin/transfer", ccy, amount, transferType)
}

func (e *Exchange) transferSubAccount(ctx context.Context, email, path string, ccy currency.Code, amount float64, transferType int64) (string, error) {
	if !common.MatchesEmailPattern(email) {
		return "", errValidEmailRequired
	}
	if ccy.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return "", limits.ErrAmountBelowMin
	}
	if transferType != 1 && transferType != 2 && transferType != 3 && transferType != 4 {
		return "", errTransferTypeRequired
	}
	params := url.Values{}
	params.Set("email", email)
	params.Set("asset", ccy.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	params.Set("type", strconv.FormatInt(transferType, 10))
	resp := struct {
		TransactionID string `json:"txnId"`
	}{}
	return resp.TransactionID, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, params, sapiDefaultRate, nil, &resp)
}

// GetSubAccountAssetsV3 retrieves sub-account assets
func (e *Exchange) GetSubAccountAssetsV3(ctx context.Context, email string) (*SubAccountAssets, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, fmt.Errorf("%w: provided %s", errValidEmailRequired, email)
	}
	params := url.Values{}
	params.Set("email", email)
	var resp *SubAccountAssets
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v3/sub-account/assets", params, getV3SubAccountAssetsRate, nil, &resp)
}

// TransferToSubAccountOfSameMaster Transfer to Sub-account of Same Master (For Sub-account)
func (e *Exchange) TransferToSubAccountOfSameMaster(ctx context.Context, toEmail string, ccy currency.Code, amount float64) (string, error) {
	if !common.MatchesEmailPattern(toEmail) {
		return "", errValidEmailRequired
	}
	if ccy.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return "", limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("toEmail", toEmail)
	params.Set("asset", ccy.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	resp := &struct {
		TransactionID string `json:"txnId"`
	}{}
	return resp.TransactionID, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/sub-account/transfer/subToSub", params, sapiDefaultRate, nil, &resp)
}

// FromSubAccountTransferToMaster Transfer to Master (For Sub-account)
// need to open Enable Spot & Margin Trading permission for the API Key which requests this endpoint.
func (e *Exchange) FromSubAccountTransferToMaster(ctx context.Context, ccy currency.Code, amount float64) (string, error) {
	if ccy.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return "", limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("asset", ccy.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	resp := &struct {
		TransactionID string `json:"txnId"`
	}{}
	return resp.TransactionID, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/transfer/subToMaster", params, sapiDefaultRate, nil, &resp)
}

// SubAccountTransferHistory retrieves Sub-account Transfer History (For Sub-account)
func (e *Exchange) SubAccountTransferHistory(ctx context.Context, ccy currency.Code, transferType, limit int64, startTime, endTime time.Time) (*SubAccountTransferHistory, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("asset", ccy.String())
	}
	if transferType != 1 && transferType != 2 {
		params.Set("type", strconv.FormatInt(transferType, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *SubAccountTransferHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/transfer/subUserHistory", params, sapiDefaultRate, nil, &resp)
}

// SubAccountTransferHistoryForSubAccount represents a sub-account transfer history for sub accounts.
func (e *Exchange) SubAccountTransferHistoryForSubAccount(ctx context.Context, ccy currency.Code, transferType, limit int64, startTime, endTime time.Time, returnFailHistory bool) (*SubAccountTransferHistoryItem, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("asset", ccy.String())
	}
	if transferType != 0 {
		params.Set("type", strconv.FormatInt(transferType, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if returnFailHistory {
		params.Set("returnFailHistory", "true")
	}
	var resp *SubAccountTransferHistoryItem
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/transfer/subUserHistory", params, sapiDefaultRate, nil, &resp)
}

// UniversalTransferForMasterAccount submits a universal transfer using the master account.
func (e *Exchange) UniversalTransferForMasterAccount(ctx context.Context, arg *UniversalTransferParams) (*UniversalTransferResponse, error) {
	if *arg == (UniversalTransferParams{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.FromAccountType == "" {
		return nil, fmt.Errorf("%w: fromAccountType=%s", errInvalidAccountType, arg.FromAccountType)
	}
	if arg.ToAccountType == "" {
		return nil, fmt.Errorf("%w: toAccountType = %s", errInvalidAccountType, arg.ToAccountType)
	}
	if arg.Asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("fromAccountType", arg.FromAccountType)
	params.Set("toAccountType", arg.ToAccountType)
	params.Set("asset", arg.Asset.String())
	params.Set("amount", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	if arg.FromEmail != "" {
		params.Set("fromEmail", arg.FromEmail)
	}
	if arg.ToEmail != "" {
		params.Set("toEmail", arg.ToEmail)
	}
	if arg.ClientTransactionID != "" {
		params.Set("clientTranId", arg.ClientTransactionID)
	}
	if arg.Symbol != "" {
		params.Set("symbol", arg.Symbol)
	}
	var resp *UniversalTransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/sub-account/universalTransfer", params, sapiDefaultRate, nil, &resp)
}

// GetUniversalTransferHistoryForMasterAccount retrieves universal transfer history for master account.
func (e *Exchange) GetUniversalTransferHistoryForMasterAccount(ctx context.Context, fromEmail, toEmail, clientTransactionID string, startTime, endTime time.Time, page, limit int64) (*UniversalTransfersDetail, error) {
	params := url.Values{}
	if fromEmail != "" {
		params.Set("fromEmail", fromEmail)
	}
	if toEmail != "" {
		params.Set("toEmail", toEmail)
	}
	if clientTransactionID != "" {
		params.Set("clientTranId", clientTransactionID)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *UniversalTransfersDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/universalTransfer", params, sapiDefaultRate, nil, &resp)
}

// GetDetailOnSubAccountsFuturesAccountV2 retrieves detail on sub-account's futures account V2 for master account
func (e *Exchange) GetDetailOnSubAccountsFuturesAccountV2(ctx context.Context, email string, futuresType int64) (*MarginedFuturesAccount, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	if futuresType == 0 {
		return nil, errInvalidFuturesType
	}
	params := url.Values{}
	params.Set("email", email)
	params.Set("futuresType", strconv.FormatInt(futuresType, 10))
	var resp *MarginedFuturesAccount
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v2/sub-account/futures/account", params, sapiDefaultRate, nil, &resp)
}

// GetSummaryOfSubAccountsFuturesAccountV2 retrieves the summary of sub-account's futures account v2 for master account
func (e *Exchange) GetSummaryOfSubAccountsFuturesAccountV2(ctx context.Context, futuresType, page, limit int64) (*AccountSummary, error) {
	if futuresType == 0 {
		return nil, errInvalidFuturesType
	}
	params := url.Values{}
	params.Set("futuresType", strconv.FormatInt(futuresType, 10))
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *AccountSummary
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v2/sub-account/futures/accountSummary", params, getFuturesSubAccountSummaryV2Rate, nil, &resp)
}

// GetAccountStatus fetch account status detail.
func (e *Exchange) GetAccountStatus(ctx context.Context) (string, error) {
	resp := &struct {
		Data string `json:"data"`
	}{}
	return resp.Data, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/account/status", nil, sapiDefaultRate, nil, &resp)
}

// GetAccountTradingAPIStatus fetch account api trading status detail.
func (e *Exchange) GetAccountTradingAPIStatus(ctx context.Context) (*TradingAPIAccountStatus, error) {
	var resp *TradingAPIAccountStatus
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/account/apiTradingStatus", nil, sapiDefaultRate, nil, &resp)
}

// GetDustLog retrieves record of small or fractional amounts of assets that accumulate in a user's account
func (e *Exchange) GetDustLog(ctx context.Context, accountType string, startTime, endTime time.Time) (*DustLog, error) {
	params := url.Values{}
	if accountType == "" {
		params.Set("accountType", accountType)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp *DustLog
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/asset/dribblet", params, sapiDefaultRate, nil, &resp)
}

// GetWsAuthStreamKey will retrieve a key to use for authorised WS streaming
func (e *Exchange) GetWsAuthStreamKey(ctx context.Context) (string, error) {
	endpointPath, err := e.API.Endpoints.GetURL(exchange.RestSpot)
	if err != nil {
		return "", err
	}

	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return "", err
	}

	var resp UserAccountStream
	headers := make(map[string]string)
	headers["X-MBX-APIKEY"] = creds.Key
	item := &request.Item{
		Method:                 http.MethodPost,
		Path:                   endpointPath + "/api/v3/userDataStream",
		Headers:                headers,
		Result:                 &resp,
		Verbose:                e.Verbose,
		HTTPDebugging:          e.HTTPDebugging,
		HTTPRecording:          e.HTTPRecording,
		HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
	}

	err = e.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	}, request.AuthenticatedRequest)
	if err != nil {
		return "", err
	}
	return resp.ListenKey, nil
}

// MaintainWsAuthStreamKey will keep the key alive
func (e *Exchange) MaintainWsAuthStreamKey(ctx context.Context) error {
	endpointPath, err := e.API.Endpoints.GetURL(exchange.RestSpot)
	if err != nil {
		return err
	}
	if listenKey == "" {
		listenKey, err = e.GetWsAuthStreamKey(ctx)
		return err
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	path := endpointPath + "/api/v3/userDataStream"
	params := url.Values{}
	params.Set("listenKey", listenKey)
	path = common.EncodeURLValues(path, params)
	headers := make(map[string]string)
	headers["X-MBX-APIKEY"] = creds.Key
	item := &request.Item{
		Method:                 http.MethodPut,
		Path:                   path,
		Headers:                headers,
		Verbose:                e.Verbose,
		HTTPDebugging:          e.HTTPDebugging,
		HTTPRecording:          e.HTTPRecording,
		HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
	}
	return e.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	}, request.AuthenticatedRequest)
}

// FetchExchangeLimits fetches order execution limits filtered by asset
func (e *Exchange) FetchExchangeLimits(ctx context.Context, a asset.Item) ([]limits.MinMaxLevel, error) {
	if a != asset.Spot && a != asset.Margin {
		return nil, fmt.Errorf("%w %q", asset.ErrNotSupported, a)
	}

	resp, err := e.GetExchangeInfo(ctx)
	if err != nil {
		return nil, err
	}

	aUpper := strings.ToUpper(a.String())

	l := make([]limits.MinMaxLevel, 0, len(resp.Symbols))
	for _, s := range resp.Symbols {
		var cp currency.Pair
		cp, err = currency.NewPairFromStrings(s.BaseAsset, s.QuoteAsset)
		if err != nil {
			return nil, err
		}
		var hasPermission bool
		for _, permissionSet := range s.PermissionSets {
			if slices.Contains(permissionSet, aUpper) {
				hasPermission = true
				break
			}
		}
		if !hasPermission {
			continue
		}

		mml := limits.MinMaxLevel{
			Key: key.NewExchangeAssetPair(e.Name, a, cp),
		}

		for _, f := range s.Filters {
			// TODO: Unhandled filters:
			// maxPosition, trailingDelta, percentPriceBySide, maxNumAlgoOrders
			switch f.FilterType {
			case priceFilter:
				mml.MinPrice = f.MinPrice
				mml.MaxPrice = f.MaxPrice
				mml.PriceStepIncrementSize = f.TickSize
			case percentPriceFilter:
				mml.MultiplierUp = f.MultiplierUp
				mml.MultiplierDown = f.MultiplierDown
				mml.AveragePriceMinutes = f.AvgPriceMinutes
			case lotSizeFilter:
				mml.MaximumBaseAmount = f.MaxQty
				mml.MinimumBaseAmount = f.MinQty
				mml.AmountStepIncrementSize = f.StepSize
			case notionalFilter:
				mml.MinNotional = f.MinNotional
			case icebergPartsFilter:
				mml.MaxIcebergParts = f.Limit
			case marketLotSizeFilter:
				mml.MarketMinQty = f.MinQty
				mml.MarketMaxQty = f.MaxQty
				mml.MarketStepIncrementSize = f.StepSize
			case maxNumOrdersFilter:
				mml.MaxTotalOrders = f.MaxNumOrders
				mml.MaxAlgoOrders = f.MaxNumAlgoOrders
			}
		}

		l = append(l, mml)
	}
	return l, nil
}

// CryptoLoanIncomeHistory returns crypto loan income history
func (e *Exchange) CryptoLoanIncomeHistory(ctx context.Context, curr currency.Code, loanType string, startTime, endTime time.Time, limit int64) ([]CryptoLoansIncomeHistory, error) {
	params := url.Values{}
	if !curr.IsEmpty() {
		params.Set("asset", curr.String())
	}
	if loanType != "" {
		params.Set("type", loanType)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []CryptoLoansIncomeHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/loan/income", params, cryptoLoansIncomeHistory, nil, &resp)
}

// CryptoLoanBorrow borrows crypto
func (e *Exchange) CryptoLoanBorrow(ctx context.Context, loanCoin currency.Code, loanAmount float64, collateralCoin currency.Code, collateralAmount float64, loanTerm int64) ([]CryptoLoanBorrow, error) {
	if loanCoin.IsEmpty() {
		return nil, errLoanCoinMustBeSet
	}
	if collateralCoin.IsEmpty() {
		return nil, errCollateralCoinMustBeSet
	}
	if loanTerm <= 0 {
		return nil, errLoanTermMustBeSet
	}
	if loanAmount == 0 && collateralAmount == 0 {
		return nil, fmt.Errorf("%w: either loan or collateral amounts must be set", limits.ErrAmountBelowMin)
	}
	params := url.Values{}
	params.Set("loanCoin", loanCoin.String())
	if loanAmount != 0 {
		params.Set("loanAmount", strconv.FormatFloat(loanAmount, 'f', -1, 64))
	}
	params.Set("collateralCoin", collateralCoin.String())
	if collateralAmount != 0 {
		params.Set("collateralAmount", strconv.FormatFloat(collateralAmount, 'f', -1, 64))
	}
	params.Set("loanTerm", strconv.FormatInt(loanTerm, 10))
	var resp []CryptoLoanBorrow
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/loan/borrow", params, sapiDefaultRate, nil, &resp)
}

// CryptoLoanBorrowHistory gets loan borrow history
func (e *Exchange) CryptoLoanBorrowHistory(ctx context.Context, orderID int64, loanCoin, collateralCoin currency.Code, startTime, endTime time.Time, current, limit int64) (*LoanBorrowHistory, error) {
	params, err := fillHistoryParams(startTime, endTime, current, 0)
	if err != nil {
		return nil, err
	}
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if !loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.String())
	}
	if !collateralCoin.IsEmpty() {
		params.Set("collateralCoin", collateralCoin.String())
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *LoanBorrowHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/loan/borrow/history", params, getLoanBorrowHistoryRate, nil, &resp)
}

// CryptoLoanOngoingOrders obtains ongoing loan orders
func (e *Exchange) CryptoLoanOngoingOrders(ctx context.Context, orderID int64, loanCoin, collateralCoin currency.Code, current, limit int64) (*CryptoLoanOngoingOrder, error) {
	params := url.Values{}
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if !loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.String())
	}
	if !collateralCoin.IsEmpty() {
		params.Set("collateralCoin", collateralCoin.String())
	}
	if current != 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *CryptoLoanOngoingOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/loan/ongoing/orders", params, getBorrowOngoingOrdersRate, nil, &resp)
}

// CryptoLoanRepay repays a crypto loan
func (e *Exchange) CryptoLoanRepay(ctx context.Context, orderID int64, amount float64, repayType int64, collateralReturn bool) ([]CryptoLoanRepay, error) {
	if orderID <= 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("orderId", strconv.FormatInt(orderID, 10))
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	if repayType != 0 {
		params.Set("type", strconv.FormatInt(repayType, 10))
	}
	params.Set("collateralReturn", strconv.FormatBool(collateralReturn))
	var resp []CryptoLoanRepay
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/loan/repay", params, cryptoRepayLoanRate, nil, &resp)
}

// CryptoLoanRepaymentHistory gets the crypto loan repayment history
func (e *Exchange) CryptoLoanRepaymentHistory(ctx context.Context, orderID int64, loanCoin, collateralCoin currency.Code, startTime, endTime time.Time, current, limit int64) (*CryptoLoanRepayHistory, error) {
	params, err := fillHistoryParams(startTime, endTime, current, 0)
	if err != nil {
		return nil, err
	}
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if !loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.String())
	}
	if !collateralCoin.IsEmpty() {
		params.Set("collateralCoin", collateralCoin.String())
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *CryptoLoanRepayHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/loan/repay/history", params, repaymentHistoryRate, nil, &resp)
}

// CryptoLoanAdjustLTV adjusts the LTV of a crypto loan
func (e *Exchange) CryptoLoanAdjustLTV(ctx context.Context, orderID int64, reduce bool, amount float64) (*CryptoLoanAdjustLTV, error) {
	if orderID <= 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("orderId", strconv.FormatInt(orderID, 10))
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	direction := "ADDITIONAL"
	if reduce {
		direction = "REDUCED"
	}
	params.Set("direction", direction)
	var resp *CryptoLoanAdjustLTV
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/loan/adjust/ltv", params, adjustLTVRate, nil, &resp)
}

// CryptoLoanLTVAdjustmentHistory gets the crypto loan LTV adjustment history
func (e *Exchange) CryptoLoanLTVAdjustmentHistory(ctx context.Context, orderID int64, loanCoin, collateralCoin currency.Code, startTime, endTime time.Time, current, limit int64) (*CryptoLoanLTVAdjustmentHistory, error) {
	params, err := fillHistoryParams(startTime, endTime, current, 0)
	if err != nil {
		return nil, err
	}
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if !loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.String())
	}
	if !collateralCoin.IsEmpty() {
		params.Set("collateralCoin", collateralCoin.String())
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *CryptoLoanLTVAdjustmentHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/loan/ltv/adjustment/history", params, getLoanLTVAdjustmentHistoryRate, nil, &resp)
}

// CryptoLoanAssetsData gets the loanable assets data
func (e *Exchange) CryptoLoanAssetsData(ctx context.Context, loanCoin currency.Code, vipLevel int64) (*LoanableAssetsData, error) {
	params := url.Values{}
	if !loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.String())
	}
	if vipLevel != 0 {
		params.Set("vipLevel", strconv.FormatInt(vipLevel, 10))
	}
	var resp *LoanableAssetsData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/loan/loanable/data", params, getLoanableAssetsDataRate, nil, &resp)
}

// CryptoLoanCollateralAssetsData gets the collateral assets data
func (e *Exchange) CryptoLoanCollateralAssetsData(ctx context.Context, collateralCoin currency.Code, vipLevel int64) (*CollateralAssetData, error) {
	params := url.Values{}
	if !collateralCoin.IsEmpty() {
		params.Set("collateralCoin", collateralCoin.String())
	}
	if vipLevel != 0 {
		params.Set("vipLevel", strconv.FormatInt(vipLevel, 10))
	}
	var resp *CollateralAssetData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/loan/collateral/data", params, collateralAssetsDataRate, nil, &resp)
}

// CryptoLoanCheckCollateralRepayRate checks the collateral repay rate
func (e *Exchange) CryptoLoanCheckCollateralRepayRate(ctx context.Context, loanCoin, collateralCoin currency.Code, amount float64) (*CollateralRepayRate, error) {
	if loanCoin.IsEmpty() {
		return nil, errLoanCoinMustBeSet
	}
	if collateralCoin.IsEmpty() {
		return nil, errCollateralCoinMustBeSet
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("loanCoin", loanCoin.String())
	params.Set("collateralCoin", collateralCoin.String())
	params.Set("repayAmount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *CollateralRepayRate
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/loan/repay/collateral/rate", params, checkCollateralRepayRate, nil, &resp)
}

// CryptoLoanCustomiseMarginCall customises a loan's margin call
func (e *Exchange) CryptoLoanCustomiseMarginCall(ctx context.Context, orderID int64, collateralCoin currency.Code, marginCallValue float64) (*CustomiseMarginCall, error) {
	if marginCallValue <= 0 {
		return nil, fmt.Errorf("%w: marginCallValue must not be <= 0", errMarginCallValueRequired)
	}
	params := url.Values{}
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if !collateralCoin.IsEmpty() {
		params.Set("collateralCoin", collateralCoin.String())
	}
	params.Set("marginCall", strconv.FormatFloat(marginCallValue, 'f', -1, 64))
	var resp *CustomiseMarginCall
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/loan/customize/margin_call", params, cryptoLoanCustomizeMarginRate, nil, &resp)
}

// FlexibleLoanBorrow creates a flexible loan
func (e *Exchange) FlexibleLoanBorrow(ctx context.Context, loanCoin, collateralCoin currency.Code, loanAmount, collateralAmount float64) (*FlexibleLoanBorrow, error) {
	if loanCoin.IsEmpty() {
		return nil, errLoanCoinMustBeSet
	}
	if collateralCoin.IsEmpty() {
		return nil, errCollateralCoinMustBeSet
	}
	if loanAmount == 0 && collateralAmount == 0 {
		return nil, fmt.Errorf("%w: either loan or collateral amounts must be set", limits.ErrAmountBelowMin)
	}
	params := url.Values{}
	params.Set("loanCoin", loanCoin.String())
	if loanAmount != 0 {
		params.Set("loanAmount", strconv.FormatFloat(loanAmount, 'f', -1, 64))
	}
	params.Set("collateralCoin", collateralCoin.String())
	if collateralAmount != 0 {
		params.Set("collateralAmount", strconv.FormatFloat(collateralAmount, 'f', -1, 64))
	}
	var resp *FlexibleLoanBorrow
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v2/loan/flexible/borrow", params, borrowFlexibleRate, nil, &resp)
}

// FlexibleLoanOngoingOrders gets the flexible loan ongoing orders
func (e *Exchange) FlexibleLoanOngoingOrders(ctx context.Context, loanCoin, collateralCoin currency.Code, current, limit int64) (*FlexibleLoanOngoingOrder, error) {
	params := url.Values{}
	if !loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.String())
	}
	if !collateralCoin.IsEmpty() {
		params.Set("collateralCoin", collateralCoin.String())
	}
	if current != 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *FlexibleLoanOngoingOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v2/loan/flexible/ongoing/orders", params, getFlexibleLoanOngoingOrdersRate, nil, &resp)
}

// FlexibleLoanBorrowHistory gets the flexible loan borrow history
func (e *Exchange) FlexibleLoanBorrowHistory(ctx context.Context, loanCoin, collateralCoin currency.Code, startTime, endTime time.Time, current, limit int64) (*FlexibleLoanBorrowHistory, error) {
	params, err := fillHistoryParams(startTime, endTime, current, 0)
	if err != nil {
		return nil, err
	}
	if !loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.String())
	}
	if !collateralCoin.IsEmpty() {
		params.Set("collateralCoin", collateralCoin.String())
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *FlexibleLoanBorrowHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v2/loan/flexible/borrow/history", params, flexibleBorrowHistoryRate, nil, &resp)
}

// FlexibleLoanRepay repays a flexible loan
func (e *Exchange) FlexibleLoanRepay(ctx context.Context, loanCoin, collateralCoin currency.Code, amount float64, collateralReturn, fullRepayment bool) (*FlexibleLoanRepay, error) {
	if loanCoin.IsEmpty() {
		return nil, errLoanCoinMustBeSet
	}
	if collateralCoin.IsEmpty() {
		return nil, errCollateralCoinMustBeSet
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("loanCoin", loanCoin.String())
	params.Set("collateralCoin", collateralCoin.String())
	params.Set("repayAmount", strconv.FormatFloat(amount, 'f', -1, 64))
	params.Set("collateralReturn", strconv.FormatBool(collateralReturn))
	if fullRepayment {
		params.Set("fullRepayment", "true")
	}
	var resp *FlexibleLoanRepay
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v2/loan/flexible/repay", params, repayFlexibleLoanHistoryRate, nil, &resp)
}

// FlexibleLoanRepayHistory gets the flexible loan repayment history
func (e *Exchange) FlexibleLoanRepayHistory(ctx context.Context, loanCoin, collateralCoin currency.Code, startTime, endTime time.Time, current, limit int64) (*FlexibleLoanRepayHistory, error) {
	params, err := fillHistoryParams(startTime, endTime, current, 0)
	if err != nil {
		return nil, err
	}
	if !loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.String())
	}
	if !collateralCoin.IsEmpty() {
		params.Set("collateralCoin", collateralCoin.String())
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *FlexibleLoanRepayHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v2/loan/flexible/repay/history", params, flexibleLoanRepaymentHistoryRate, nil, &resp)
}

// FlexibleLoanCollateralRepayment flexible loan collateral repayment
func (e *Exchange) FlexibleLoanCollateralRepayment(ctx context.Context, loanCoin, collateralCoin currency.Code, repaymentAmount float64, fullRepayment bool) (*FlexibleLoanCollateralRepaymentResponse, error) {
	if loanCoin.IsEmpty() {
		return nil, errLoanCoinMustBeSet
	}
	if collateralCoin.IsEmpty() {
		return nil, errCollateralCoinMustBeSet
	}
	if repaymentAmount <= 0 {
		return nil, fmt.Errorf("%w: repayment amount is required", limits.ErrAmountBelowMin)
	}
	params := url.Values{}
	params.Set("loanCoin", loanCoin.String())
	params.Set("collateralCoin", collateralCoin.String())
	params.Set("repaymentAmount", strconv.FormatFloat(repaymentAmount, 'f', -1, 64))
	if fullRepayment {
		params.Set("fullRepayment", "true")
	}
	var resp *FlexibleLoanCollateralRepaymentResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v2/loan/flexible/repay/collateral", params, flexibleLoanCollateralRepaymentRate, nil, &resp)
}

// CheckCollateralRepayRate checks collateral loan repayment rate of the account
func (e *Exchange) CheckCollateralRepayRate(ctx context.Context, loanCoin, collateralCoin currency.Code) (*CollateralRepayRate, error) {
	if loanCoin.IsEmpty() {
		return nil, errLoanCoinMustBeSet
	}
	if collateralCoin.IsEmpty() {
		return nil, errCollateralCoinMustBeSet
	}
	params := url.Values{}
	params.Set("loanCoin", loanCoin.String())
	params.Set("collateralCoin", collateralCoin.String())
	var resp *CollateralRepayRate
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v2/loan/flexible/repay/rate", params, checkCollateralRepayRate, nil, &resp)
}

// GetFlexibleLoanLiquidiationHistory retrieves flexible loan liquidiation history of an account
func (e *Exchange) GetFlexibleLoanLiquidiationHistory(ctx context.Context, loanCoin, collateralCoin currency.Code, startTime, endTime time.Time, current, limit int64) (*FlexibleLoanLiquidiationhistory, error) {
	params, err := fillHistoryParams(startTime, endTime, current, 0)
	if err != nil {
		return nil, err
	}
	if !loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.String())
	}
	if !collateralCoin.IsEmpty() {
		params.Set("collateralCoin", collateralCoin.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *FlexibleLoanLiquidiationhistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v2/loan/flexible/liquidation/history", params, flexibleLoanLiquidiationHistoryRate, nil, &resp)
}

// FlexibleLoanAdjustLTV adjusts the LTV of a flexible loan
func (e *Exchange) FlexibleLoanAdjustLTV(ctx context.Context, loanCoin, collateralCoin currency.Code, amount float64, reduce bool) (*FlexibleLoanAdjustLTV, error) {
	if loanCoin.IsEmpty() {
		return nil, errLoanCoinMustBeSet
	}
	if collateralCoin.IsEmpty() {
		return nil, errCollateralCoinMustBeSet
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	direction := "ADDITIONAL"
	if reduce {
		direction = "REDUCED"
	}
	params := url.Values{}
	params.Set("loanCoin", loanCoin.String())
	params.Set("collateralCoin", collateralCoin.String())
	params.Set("adjustmentAmount", strconv.FormatFloat(amount, 'f', -1, 64))
	params.Set("direction", direction)
	var resp *FlexibleLoanAdjustLTV
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v2/loan/flexible/adjust/ltv", params, adjustFlexibleLoanRate, nil, &resp)
}

// FlexibleLoanLTVAdjustmentHistory gets the flexible loan LTV adjustment history
func (e *Exchange) FlexibleLoanLTVAdjustmentHistory(ctx context.Context, loanCoin, collateralCoin currency.Code, startTime, endTime time.Time, current, limit int64) (*FlexibleLoanLTVAdjustmentHistory, error) {
	params, err := fillHistoryParams(startTime, endTime, current, 0)
	if err != nil {
		return nil, err
	}
	if !loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.String())
	}
	if !collateralCoin.IsEmpty() {
		params.Set("collateralCoin", collateralCoin.String())
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *FlexibleLoanLTVAdjustmentHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v2/loan/flexible/ltv/adjustment/history", params, flexibleLoanAdjustLTVRate, nil, &resp)
}

// FlexibleLoanAssetsData gets the flexible loan assets data
func (e *Exchange) FlexibleLoanAssetsData(ctx context.Context, loanCoin currency.Code) (*FlexibleLoanAssetsData, error) {
	params := url.Values{}
	if !loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.String())
	}
	var resp *FlexibleLoanAssetsData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v2/loan/flexible/loanable/data", params, flexibleLoanAssetDataRate, nil, &resp)
}

// FlexibleCollateralAssetsData gets the flexible loan collateral assets data
func (e *Exchange) FlexibleCollateralAssetsData(ctx context.Context, collateralCoin currency.Code) (*FlexibleCollateralAssetsData, error) {
	params := url.Values{}
	if !collateralCoin.IsEmpty() {
		params.Set("collateralCoin", collateralCoin.String())
	}
	var resp *FlexibleCollateralAssetsData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v2/loan/flexible/collateral/data", params, flexibleLoanCollateralAssetRate, nil, &resp)
}

// ----------------------------------  Simple Earn Endpoints -------------------------------
// The endpoints below allow you to interact with Binance Simple Earn.

// GetSimpleEarnFlexibleProductList retrieves available simple earn flexible product list.
func (e *Exchange) GetSimpleEarnFlexibleProductList(ctx context.Context, assetName currency.Code, current, size int64) (*SimpleEarnProducts, error) {
	params := url.Values{}
	if !assetName.IsEmpty() {
		params.Set("asset", assetName.String())
	}
	if current > 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	var resp *SimpleEarnProducts
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/simple-earn/flexible/list", params, simpleEarnProductsRate, nil, &resp)
}

// GetSimpleEarnLockedProducts retrieves available Simple Earn locked product list
func (e *Exchange) GetSimpleEarnLockedProducts(ctx context.Context, assetName currency.Code, current, size int64) (*LockedSimpleEarnProducts, error) {
	params := url.Values{}
	if !assetName.IsEmpty() {
		params.Set("asset", assetName.String())
	}
	if current > 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	var resp *LockedSimpleEarnProducts
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/simple-earn/locked/list", params, simpleEarnProductsRate, nil, &resp)
}

// SubscribeToFlexibleProducts subscribe to simple earn flexible product instance.
// You need to open Enable Spot & Margin Trading permission for the API Key which requests this endpoint.
// sourceAccount: possible values are- SPOT, FUND, ALL, default SPOT
func (e *Exchange) SubscribeToFlexibleProducts(ctx context.Context, productID, sourceAccount string, amount float64, autoSubscribe bool) (*SimpleEarnSubscriptionResponse, error) {
	if productID == "" {
		return nil, errProductIDRequired
	}
	return e.subscribeToFlexibleAndLockedProducts(ctx, productID, "", sourceAccount, "/sapi/v1/simple-earn/flexible/subscribe", amount, autoSubscribe)
}

// SubscribeToLockedProducts subscribes to locked products
func (e *Exchange) SubscribeToLockedProducts(ctx context.Context, projectID, sourceAccount string, amount float64, autoSubscribe bool) (*SimpleEarnSubscriptionResponse, error) {
	if projectID == "" {
		return nil, errProjectIDRequired
	}
	return e.subscribeToFlexibleAndLockedProducts(ctx, "", projectID, sourceAccount, "/sapi/v1/simple-earn/locked/subscribe", amount, autoSubscribe)
}

func (e *Exchange) subscribeToFlexibleAndLockedProducts(ctx context.Context, productID, projectID, sourceAccount, path string, amount float64, autoSubscribe bool) (*SimpleEarnSubscriptionResponse, error) {
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	if productID != "" {
		params.Set("productId", productID)
	}
	if projectID != "" {
		params.Set("projectId", projectID)
	}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	if autoSubscribe {
		params.Set("autoSubscribe", "true")
	}
	if sourceAccount != "" {
		params.Set("sourceAccount", sourceAccount)
	}
	var resp *SimpleEarnSubscriptionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, params, sapiDefaultRate, nil, &resp)
}

// RedeemFlexibleProduct redeems flexible products
// destinationAccount: possible values SPOT, FUND, default SPOT
func (e *Exchange) RedeemFlexibleProduct(ctx context.Context, productID, destinationAccount string, redeemAll bool, amount float64) (*RedeemResponse, error) {
	if productID == "" {
		return nil, errProductIDRequired
	}
	params := url.Values{}
	params.Set("productId", productID)
	if destinationAccount != "" {
		params.Set("destAccount", destinationAccount)
	}
	if redeemAll {
		params.Set("redeemAll", "true")
	}
	if amount != 0 {
		params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	}
	var resp *RedeemResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/simple-earn/flexible/redeem", params, sapiDefaultRate, nil, &resp)
}

// RedeemLockedProduct posts a redeem locked product
func (e *Exchange) RedeemLockedProduct(ctx context.Context, positionID int64) (*RedeemResponse, error) {
	if positionID == 0 {
		return nil, errPositionIDRequired
	}
	params := url.Values{}
	params.Set("positionId", strconv.FormatInt(positionID, 10))
	var resp *RedeemResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/simple-earn/locked/redeem", params, sapiDefaultRate, nil, &resp)
}

// GetFlexibleProductPosition retrieves flexible product position
func (e *Exchange) GetFlexibleProductPosition(ctx context.Context, assetName currency.Code, productID string, current, size int64) (*FlexibleProductPosition, error) {
	params := url.Values{}
	if !assetName.IsEmpty() {
		params.Set("asset", assetName.String())
	}
	if productID != "" {
		params.Set("productId", productID)
	}
	if current > 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	var resp *FlexibleProductPosition
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/simple-earn/flexible/position", params, getFlexibleSimpleEarnProductPositionRate, nil, &resp)
}

// GetLockedProductPosition retrieves locked product positions.
func (e *Exchange) GetLockedProductPosition(ctx context.Context, assetName currency.Code, positionID, projectID string, current, size int64) (*LockedProductPosition, error) {
	params := url.Values{}
	if !assetName.IsEmpty() {
		params.Set("asset", assetName.String())
	}
	if positionID != "" {
		params.Set("positionId", positionID)
	}
	if projectID != "" {
		params.Set("projectId", projectID)
	}
	if current > 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	var resp *LockedProductPosition
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/simple-earn/locked/position", params, getSimpleEarnProductPositionRate, nil, &resp)
}

// SimpleAccount retrieves simple account instance.
func (e *Exchange) SimpleAccount(ctx context.Context) (*SimpleAccount, error) {
	var resp *SimpleAccount
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/simple-earn/account", nil, simpleAccountRate, nil, &resp)
}

// GetFlexibleSubscriptionRecord retrieves flexible subscription record.
func (e *Exchange) GetFlexibleSubscriptionRecord(ctx context.Context, productID, purchaseID string, assetName currency.Code, startTime, endTime time.Time, current, size int64) (*FlexibleSubscriptionRecord, error) {
	params, err := fillSubscriptionAndRedemptionRecord(productID, purchaseID, "", "", assetName, startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	var resp *FlexibleSubscriptionRecord
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/simple-earn/flexible/history/subscriptionRecord", params, getFlexibleSubscriptionRecordRate, nil, &resp)
}

// GetLockedSubscriptionsRecords retrieves locked subscriptions records
func (e *Exchange) GetLockedSubscriptionsRecords(ctx context.Context, purchaseID string, assetName currency.Code, startTime, endTime time.Time, current, size int64) (*LockedSubscriptions, error) {
	params, err := fillSubscriptionAndRedemptionRecord(purchaseID, "", "", "", assetName, startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	var resp *LockedSubscriptions
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/simple-earn/locked/history/subscriptionRecord", params, getLockedSubscriptionRecordsRate, nil, &resp)
}

// GetFlexibleRedemptionRecord retrieves flexible redemption record
func (e *Exchange) GetFlexibleRedemptionRecord(ctx context.Context, productID, redeemID string, assetName currency.Code, startTime, endTime time.Time, current, size int64) (*RedemptionRecord, error) {
	params, err := fillSubscriptionAndRedemptionRecord(productID, "", redeemID, "", assetName, startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	var resp *RedemptionRecord
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/simple-earn/flexible/history/redemptionRecord", params, getRedemptionRecordRate, nil, &resp)
}

// GetLockedRedemptionRecord retrieves locked redemptions record list
func (e *Exchange) GetLockedRedemptionRecord(ctx context.Context, productID, redeemID string, assetName currency.Code, startTime, endTime time.Time, current, size int64) (*LockedRedemptionRecord, error) {
	params, err := fillSubscriptionAndRedemptionRecord(productID, "", redeemID, "", assetName, startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	var resp *LockedRedemptionRecord
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/simple-earn/locked/history/redemptionRecord", params, getRedemptionRecordRate, nil, &resp)
}

func fillSubscriptionAndRedemptionRecord(productID, purchaseID, redeemID, rewardType string, assetName currency.Code, startTime, endTime time.Time, current, size int64) (url.Values, error) {
	params, err := fillHistoryParams(startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	if productID != "" {
		params.Set("productId", productID)
	}
	if purchaseID != "" {
		params.Set("purchaseId", purchaseID)
	}
	if rewardType != "" {
		params.Set("rewardType", rewardType)
	}
	if redeemID != "" {
		params.Set("redeemId", redeemID)
	}
	if !assetName.IsEmpty() {
		params.Set("asset", assetName.String())
	}
	return params, nil
}

// GetFlexibleRewardHistory retrieves flexible rewards history
func (e *Exchange) GetFlexibleRewardHistory(ctx context.Context, productID, rewardType string, assetName currency.Code, startTime, endTime time.Time, current, size int64) (*FlexibleReward, error) {
	params, err := fillSubscriptionAndRedemptionRecord(productID, "", "", rewardType, assetName, startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	var resp *FlexibleReward
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/simple-earn/flexible/history/rewardsRecord", params, getRewardHistoryRate, nil, &resp)
}

// GetLockedRewardHistory retrieves locked rewards history
func (e *Exchange) GetLockedRewardHistory(ctx context.Context, positionID string, assetName currency.Code, startTime, endTime time.Time, current, size int64) (*LockedRewards, error) {
	params, err := fillSubscriptionAndRedemptionRecord(positionID, "", "", "", assetName, startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	var resp *LockedRewards
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/simple-earn/locked/history/rewardsRecord", params, getRewardHistoryRate, nil, &resp)
}

// SetFlexibleAutoSusbcribe sets auto subscribe on to flexible products
func (e *Exchange) SetFlexibleAutoSusbcribe(ctx context.Context, productID string, autoSubscribe bool) (bool, error) {
	if productID == "" {
		return false, errProductIDRequired
	}
	params := url.Values{}
	params.Set("productId", productID)
	if autoSubscribe {
		params.Set("autoSubscribe", "true")
	} else {
		params.Set("autoSubscribe", "false")
	}
	resp := &struct {
		Success bool `json:"success"`
	}{}
	return resp.Success, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/simple-earn/flexible/setAutoSubscribe", params, setAutoSubscribeRate, nil, &resp)
}

// SetLockedAutoSubscribe sets auto subscribe to locked products
func (e *Exchange) SetLockedAutoSubscribe(ctx context.Context, positionID string, autoSubscribe bool) (bool, error) {
	if positionID == "" {
		return false, errPositionIDRequired
	}
	params := url.Values{}
	params.Set("positionId", positionID)
	if autoSubscribe {
		params.Set("autoSubscribe", "true")
	} else {
		params.Set("autoSubscribe", "false")
	}
	resp := &struct {
		Success bool `json:"success"`
	}{}
	return resp.Success, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/simple-earn/locked/setAutoSubscribe", params, setAutoSubscribeRate, nil, &resp)
}

// GetFlexiblePersonalLeftQuota retrieves flexible personal left quota
func (e *Exchange) GetFlexiblePersonalLeftQuota(ctx context.Context, productID string) (*PersonalLeftQuota, error) {
	params := url.Values{}
	if productID != "" {
		params.Set("productId", productID)
	}
	var resp *PersonalLeftQuota
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/simple-earn/flexible/personalLeftQuota", params, personalLeftQuotaRate, nil, &resp)
}

// GetLockedPersonalLeftQuota retrieves flexible personal left quota
func (e *Exchange) GetLockedPersonalLeftQuota(ctx context.Context, projectID string) (*PersonalLeftQuota, error) {
	params := url.Values{}
	if projectID != "" {
		params.Set("projectId", projectID)
	}
	var resp *PersonalLeftQuota
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/simple-earn/locked/personalLeftQuota", params, personalLeftQuotaRate, nil, &resp)
}

// GetFlexibleSubscriptionPreview retrieves flexible subscription preview
func (e *Exchange) GetFlexibleSubscriptionPreview(ctx context.Context, productID string, amount float64) (*FlexibleSubscriptionPreview, error) {
	if productID == "" {
		return nil, errProductIDRequired
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("productId", productID)
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *FlexibleSubscriptionPreview
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/simple-earn/flexible/subscriptionPreview", params, subscriptionPreviewRate, nil, &resp)
}

// GetLockedSubscriptionPreview retrieves locked subscription preview.
func (e *Exchange) GetLockedSubscriptionPreview(ctx context.Context, projectID string, amount float64, autoSubscribe bool) ([]LockedSubscriptionPreview, error) {
	if projectID == "" {
		return nil, errProjectIDRequired
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("projectId", projectID)
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	if autoSubscribe {
		params.Set("autoSubscribe", "true")
	} else {
		params.Set("autoSubscribe", "false")
	}
	var resp []LockedSubscriptionPreview
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/simple-earn/locked/subscriptionPreview", params, subscriptionPreviewRate, nil, &resp)
}

// SetLockedProductRedeemOption possible values of redeemTo are 'SPOT' and 'FLEXIBLE'.
func (e *Exchange) SetLockedProductRedeemOption(ctx context.Context, positionID, redeemTo string) (interface{}, error) {
	if positionID == "" {
		return nil, errPositionIDRequired
	}
	if redeemTo == "" {
		return nil, errRedemptionAccountRequired
	}
	params := url.Values{}
	params.Set("positionId", positionID)
	params.Set("redeemTo", redeemTo)
	resp := &struct {
		Success bool `json:"success"`
	}{}
	return resp.Success, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/simple-earn/locked/setRedeemOption", params, request.Auth, nil, &resp)
}

// GetSimpleEarnRatehistory retrieves rate history for simple-rean products
func (e *Exchange) GetSimpleEarnRatehistory(ctx context.Context, projectID string, startTime, endTime time.Time, current, size int64) (*SimpleEarnRateHistory, error) {
	params, err := fillHistoryParams(startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	if projectID != "" {
		params.Set("projectId", projectID)
	}
	var resp *SimpleEarnRateHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/simple-earn/flexible/history/rateHistory", params, simpleEarnRateHistoryRate, nil, &resp)
}

// GetSimpleEarnCollateralRecord retrieves simple earn collateral records
func (e *Exchange) GetSimpleEarnCollateralRecord(ctx context.Context, productID string, startTime, endTime time.Time, current, size int64) (*SimpleEarnCollateralRecords, error) {
	params, err := fillHistoryParams(startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	if productID != "" {
		params.Set("productId", productID)
	}
	var resp *SimpleEarnCollateralRecords
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/simple-earn/flexible/history/collateralRecord", params, sapiDefaultRate, nil, &resp)
}

// ------------------------------------------- Dual Investment Endpoints  -----------------------------------------------------

// GetDualInvestmentProductList retrieves a dual investment product list
// possible optionType values: 'CALL' and 'PUT'
func (e *Exchange) GetDualInvestmentProductList(ctx context.Context, optionType string, exerciseCoin, investCoin currency.Code, pageSize, pageIndex int64) (*DualInvestmentProduct, error) {
	if optionType == "" {
		return nil, errOptionTypeRequired
	}
	if exerciseCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: exerciseCoin is required", currency.ErrCurrencyCodeEmpty)
	}
	if investCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: investCoin is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("optionType", optionType)
	params.Set("exerciseCoin", exerciseCoin.String())
	params.Set("investCoin", investCoin.String())
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	if pageIndex > 0 {
		params.Set("pageIndex", strconv.FormatInt(pageIndex, 10))
	}
	var resp *DualInvestmentProduct
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/dci/product/list", params, sapiDefaultRate, nil, &resp)
}

// SubscribeDualInvestmentProducts represents dual investment products
// id: get id from /sapi/v1/dci/product/list
// orderId: get orderId from /sapi/v1/dci/product/list
// possible autoCompoundPlan values: NONE: switch off the plan, STANDARD:standard plan, ADVANCED:advanced plan
// Products are not available. // this means APR changes to lower value, or orders are not unavailable.
// Failed. This means System or network errors.
func (e *Exchange) SubscribeDualInvestmentProducts(ctx context.Context, id, orderID, autoCompoundPlan string, depositAmount float64) (*DualInvestmentProductSubscription, error) {
	if id == "" {
		return nil, errProductIDRequired
	}
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if depositAmount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if autoCompoundPlan == "" {
		return nil, fmt.Errorf("%w: accountCompoundPlan is required", errPlanTypeRequired)
	}
	params := url.Values{}
	params.Set("id", id)
	params.Set("orderId", orderID)
	params.Set("depositAmount", strconv.FormatFloat(depositAmount, 'f', -1, 64))
	params.Set("autoCompoundPlan", autoCompoundPlan)
	var resp *DualInvestmentProductSubscription
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/dci/product/subscribe", params, sapiDefaultRate, nil, &resp)
}

// GetDualInvestmentPositions get Dual Investment positions (batch)
// PENDING:Products are purchasing, will give results later;PURCHASE_SUCCESS:purchase successfully;SETTLED: Products are finish settling;PURCHASE_FAIL:fail to purchase;REFUNDING:refund ongoing;REFUND_SUCCESS:refund to spot account successfully; SETTLING:Products are settling.
// If don't fill this field, will response all the position status.
func (e *Exchange) GetDualInvestmentPositions(ctx context.Context, status string, pageSize, pageIndex int64) (*DualInvestmentPositions, error) {
	params := url.Values{}
	if status != "" {
		params.Set("status", status)
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	if pageIndex > 0 {
		params.Set("pageIndex", strconv.FormatInt(pageIndex, 10))
	}
	var resp *DualInvestmentPositions
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/dci/product/positions", params, sapiDefaultRate, nil, &resp)
}

// CheckDualInvestmentAccounts checks dual investment accounts
func (e *Exchange) CheckDualInvestmentAccounts(ctx context.Context) (*DualInvestmentAccount, error) {
	var resp *DualInvestmentAccount
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/dci/product/accounts", nil, sapiDefaultRate, nil, &resp)
}

// ChangeAutoCompoundStatus change Auto-Compound status
// autoCompoundPlan possible values: NONE, STANDARD,ADVANCED
// get positionId from /sapi/v1/dci/product/positions
func (e *Exchange) ChangeAutoCompoundStatus(ctx context.Context, positionID, autoCompoundPlan string) (*AutoCompoundStatus, error) {
	if positionID == "" {
		return nil, errPositionIDRequired
	}
	params := url.Values{}
	params.Set("positionId", positionID)
	if autoCompoundPlan != "" {
		params.Set("autoCompoundPlan", autoCompoundPlan)
	}
	var resp *AutoCompoundStatus
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/dci/product/auto_compound/edit-status", params, sapiDefaultRate, nil, &resp)
}

// ------------------------------------------   Auto-Invest Endpoints  ----------------------------------------------------

// GetTargetAssetList retrieves auto-invest
func (e *Exchange) GetTargetAssetList(ctx context.Context, targetAsset currency.Code, size, current int64) (*AutoInvestmentAsset, error) {
	params := url.Values{}
	if !targetAsset.IsEmpty() {
		params.Set("targetAsset", targetAsset.String())
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	if current > 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	var resp *AutoInvestmentAsset
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/lending/auto-invest/target-asset/list", params, sapiDefaultRate, nil, &resp)
}

// GetTargetAssetROIData retrieves return-on-investment(ROI) return list for target asset
// FIVE_YEAR,THREE_YEAR,ONE_YEAR,SIX_MONTH,THREE_MONTH,SEVEN_DAY
func (e *Exchange) GetTargetAssetROIData(ctx context.Context, targetAsset currency.Code, hisRoiType string) ([]ROIAssetData, error) {
	params := url.Values{}
	if !targetAsset.IsEmpty() {
		params.Set("targetAsset", targetAsset.String())
	}
	if hisRoiType != "" {
		params.Set("hisRoiType", hisRoiType)
	}
	var resp []ROIAssetData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/lending/auto-invest/target-asset/roi/list", params, sapiDefaultRate, nil, &resp)
}

// GetAllSourceAssetAndTargetAsset retrieves all source assets and target assets
func (e *Exchange) GetAllSourceAssetAndTargetAsset(ctx context.Context) (*AutoInvestAssets, error) {
	var resp *AutoInvestAssets
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/lending/auto-invest/all/asset", nil, sapiDefaultRate, nil, &resp)
}

// GetSourceAssetList retrieves assets to be used for investment
// usageType: "RECURRING", "ONE_TIME"
func (e *Exchange) GetSourceAssetList(ctx context.Context, targetAsset currency.Code, indexID int64, usageType, sourceType string, flexibleAllowedToUse bool) (*SourceAssetsList, error) {
	if usageType == "" {
		return nil, errUsageTypeRequired
	}
	params := url.Values{}
	params.Set("usageType", usageType)
	if !targetAsset.IsEmpty() {
		params.Set("targetAsset", targetAsset.String())
	}
	if indexID > 0 {
		params.Set("indexId", strconv.FormatInt(indexID, 10))
	}
	if flexibleAllowedToUse {
		params.Set("flexibleAllowedToUse", "true")
	}
	if sourceType != "" {
		params.Set("sourceType", sourceType)
	}
	var resp *SourceAssetsList
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/lending/auto-invest/source-asset/list", params, sapiDefaultRate, nil, &resp)
}

// InvestmentPlanCreation creates an investment plan
func (e *Exchange) InvestmentPlanCreation(ctx context.Context, arg *InvestmentPlanParams) (*InvestmentPlanResponse, error) {
	if arg == nil {
		return nil, common.ErrEmptyParams
	}
	if arg.SourceType == "" {
		return nil, errSourceTypeRequired
	}
	if arg.PlanType == "" {
		return nil, errPlanTypeRequired
	}
	if arg.SubscriptionAmount <= 0 {
		return nil, fmt.Errorf("%w: subscriptionAmount valid is %f", limits.ErrAmountBelowMin, arg.SubscriptionAmount)
	}
	if arg.SubscriptionStartDay <= 0 {
		return nil, errInvalidSubscriptionStartTime
	}
	if arg.SubscriptionStartTime < 0 {
		return nil, errInvalidSubscriptionStartTime
	}
	if arg.SourceAsset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if len(arg.Details) == 0 {
		return nil, errPortfolioDetailRequired
	}
	params := url.Values{}
	for a := range arg.Details {
		if arg.Details[a].TargetAsset.IsEmpty() {
			return nil, fmt.Errorf("%w: targetAsset is required", currency.ErrCurrencyCodeEmpty)
		}
		if arg.Details[a].Percentage < 0 {
			return nil, errInvalidPercentageAmount
		}
		params.Add("targetAsset", arg.Details[a].TargetAsset.String())
		params.Add("percentage", strconv.FormatInt(arg.Details[a].Percentage, 10))
	}
	var resp *InvestmentPlanResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/lending/auto-invest/plan/add", params, sapiDefaultRate, arg, &resp)
}

// InvestmentPlanAdjustment query Source Asset to be used for investment
func (e *Exchange) InvestmentPlanAdjustment(ctx context.Context, arg *AdjustInvestmentPlan) (*InvestmentPlanResponse, error) {
	if arg == nil {
		return nil, common.ErrEmptyParams
	}
	if arg.PlanID == 0 {
		return nil, errPlanIDRequired
	}
	if arg.SubscriptionAmount <= 0 {
		return nil, fmt.Errorf("%w: subscriptionAmount valid is %f", limits.ErrAmountBelowMin, arg.SubscriptionAmount)
	}
	if !slices.Contains(subscriptionCycleList, arg.SubscriptionCycle) {
		return nil, fmt.Errorf("%w: subscription cycle %s", errInvalidSubscriptionCycle, arg.SubscriptionCycle)
	}
	if arg.SubscriptionStartTime < 0 {
		return nil, errInvalidSubscriptionStartTime
	}
	if arg.SourceAsset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if len(arg.Details) == 0 {
		return nil, errPortfolioDetailRequired
	}
	params := url.Values{}
	for a := range arg.Details {
		if arg.Details[a].TargetAsset.IsEmpty() {
			return nil, fmt.Errorf("%w: targetAsset is required", currency.ErrCurrencyCodeEmpty)
		}
		if arg.Details[a].Percentage < 0 {
			return nil, errInvalidPercentageAmount
		}
		params.Add("targetAsset", arg.Details[a].TargetAsset.String())
		params.Add("percentage", strconv.FormatInt(arg.Details[a].Percentage, 10))
	}
	var resp *InvestmentPlanResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/lending/auto-invest/plan/edit", params, sapiDefaultRate, arg, &resp)
}

// ChangePlanStatus change Plan Status
// status: “ONGOING","PAUSED","REMOVED"
func (e *Exchange) ChangePlanStatus(ctx context.Context, planID int64, status string) (*ChangePlanStatusResponse, error) {
	if planID == 0 {
		return nil, errPlanIDRequired
	}
	if status == "" {
		return nil, errPlanStatusRequired
	}
	params := url.Values{}
	params.Set("planId", strconv.FormatInt(planID, 10))
	params.Set("status", status)
	var resp *ChangePlanStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/lending/auto-invest/plan/edit-status", params, sapiDefaultRate, nil, &resp)
}

// GetListOfPlans retrieves list of plans
func (e *Exchange) GetListOfPlans(ctx context.Context, planType string) (*InvestmentPlans, error) {
	if planType == "" {
		return nil, errPlanTypeRequired
	}
	params := url.Values{}
	params.Set("planType", planType)
	var resp *InvestmentPlans
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/lending/auto-invest/plan/list", params, sapiDefaultRate, nil, &resp)
}

// GetHoldingDetailsOfPlan query holding details of the plan
func (e *Exchange) GetHoldingDetailsOfPlan(ctx context.Context, planID int64, requestID string) (*InvestmentPlanHoldingDetail, error) {
	params := url.Values{}
	if planID > 0 {
		params.Set("planId", strconv.FormatInt(planID, 10))
	}
	if requestID != "" {
		params.Set("requestId", requestID)
	}
	var resp *InvestmentPlanHoldingDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/lending/auto-invest/plan/id", params, sapiDefaultRate, nil, &resp)
}

// GetSubscriptionsTransactionHistory query subscription transaction history of a plan
// planType: SINGLE, PORTFOLIO, INDEX, ALL
func (e *Exchange) GetSubscriptionsTransactionHistory(ctx context.Context, planID, size, current int64, startTime, endTime time.Time, targetAsset currency.Code, planType string) (*AutoInvestSubscriptionTransactionResponse, error) {
	params := url.Values{}
	if planID > 0 {
		params.Set("planId", strconv.FormatInt(planID, 10))
	}
	params, err := fillHistoryParams(startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	if planType != "" {
		params.Set("planType", planType)
	}
	if !targetAsset.IsEmpty() {
		params.Set("targetAsset", targetAsset.String())
	}
	var resp *AutoInvestSubscriptionTransactionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/lending/auto-invest/history/list", params, sapiDefaultRate, nil, &resp)
}

// GetIndexDetail retrieves index details
func (e *Exchange) GetIndexDetail(ctx context.Context, indexID int64) (*AutoInvestmentIndexDetail, error) {
	if indexID == 0 {
		return nil, errIndexIDIsRequired
	}
	params := url.Values{}
	params.Set("indexId", strconv.FormatInt(indexID, 10))
	var resp *AutoInvestmentIndexDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/lending/auto-invest/index/info", params, sapiDefaultRate, nil, &resp)
}

// GetIndexLinkedPlanPositionDetails retrieves details on users Index-Linked plan position details
func (e *Exchange) GetIndexLinkedPlanPositionDetails(ctx context.Context, indexID int64) (*IndexLinkedPlanPositionDetail, error) {
	if indexID == 0 {
		return nil, errIndexIDIsRequired
	}
	params := url.Values{}
	params.Set("indexId", strconv.FormatInt(indexID, 10))
	var resp *IndexLinkedPlanPositionDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/lending/auto-invest/index/user-summary", params, sapiDefaultRate, nil, &resp)
}

// OneTimeTransaction posts one time transactions
// sourceType possible values are "MAIN_SITE" for Binance,“TR" for Binance Turkey
func (e *Exchange) OneTimeTransaction(ctx context.Context, arg *OneTimeTransactionParams) (*OneTimeTransactionResponse, error) {
	if arg == nil {
		return nil, common.ErrEmptyParams
	}
	if arg.SourceType == "" {
		return nil, errSourceTypeRequired
	}
	if arg.SubscriptionAmount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.SourceAsset.IsEmpty() {
		return nil, fmt.Errorf("%w: sourceAsset is required", currency.ErrCurrencyCodeEmpty)
	}
	if len(arg.Details) == 0 {
		return nil, errPortfolioDetailRequired
	}
	params := url.Values{}
	for a := range arg.Details {
		if arg.Details[a].TargetAsset.IsEmpty() {
			return nil, fmt.Errorf("%w: targetAsset is required", currency.ErrCurrencyCodeEmpty)
		}
		if arg.Details[a].Percentage <= 0 {
			return nil, errInvalidPercentageAmount
		}
		params.Add("targetAsset", arg.Details[a].TargetAsset.String())
		params.Add("percentage", strconv.FormatInt(arg.Details[a].Percentage, 10))
	}
	var resp *OneTimeTransactionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/lending/auto-invest/one-off", params, sapiDefaultRate, arg, &resp)
}

// GetOneTimeTransactionStatus retrieves transaction status of one-time transaction
//
// transactionID: PORTFOLIO plan's Id
// requestID: sourceType + unique, transactionId and requestId cannot be empty at the same time
func (e *Exchange) GetOneTimeTransactionStatus(ctx context.Context, transactionID int64, requestID string) (*OneTimeTransactionResponse, error) {
	if transactionID == 0 {
		return nil, errTransactionIDRequired
	}
	params := url.Values{}
	params.Set("transactionId", strconv.FormatInt(transactionID, 10))
	if requestID != "" {
		params.Set("requestId", requestID)
	}
	var resp *OneTimeTransactionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/lending/auto-invest/one-off/status", params, sapiDefaultRate, nil, &resp)
}

// IndexLinkedPlanRedemption returns an identifier for this redemption after redeeming index-Linked plan holdings.
// redemptionPercentage: user redeem percentage,10/20/100..
func (e *Exchange) IndexLinkedPlanRedemption(ctx context.Context, indexID, redemptionPercentage int64, requestID string) (int64, error) {
	if indexID == 0 {
		return 0, errIndexIDIsRequired
	}
	if redemptionPercentage <= 0 {
		return 0, fmt.Errorf("%w: invalid redemption percentage value %v", errInvalidPercentageAmount, redemptionPercentage)
	}
	params := url.Values{}
	params.Set("indexId", strconv.FormatInt(indexID, 10))
	params.Set("redemptionPercentage", strconv.FormatInt(redemptionPercentage, 10))
	if requestID != "" {
		params.Set("requestId", requestID)
	}
	resp := &struct {
		RedemptionID int64 `json:"redemptionId"`
	}{}
	return resp.RedemptionID, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/lending/auto-invest/redeem", params, sapiDefaultRate, nil, &resp)
}

// GetIndexLinkedPlanRedemption get the history of Index Linked Plan Redemption transactions
func (e *Exchange) GetIndexLinkedPlanRedemption(ctx context.Context, requestID string, startTime, endTime time.Time, assetName currency.Code, current, size int64) ([]PlanRedemption, error) {
	if requestID == "" {
		return nil, errRequestIDRequired
	}
	params, err := fillHistoryParams(startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	params.Set("requestId", requestID)
	if !assetName.IsEmpty() {
		params.Set("asset", assetName.String())
	}
	var resp []PlanRedemption
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/lending/auto-invest/redeem/history", params, sapiDefaultRate, nil, &resp)
}

// GetIndexLinkedPlanRebalanceDetails retrieves the history of Index Linked Plan Redemption transactions
func (e *Exchange) GetIndexLinkedPlanRebalanceDetails(ctx context.Context, startTime, endTime time.Time, current, size int64) ([]IndexLinkedPlanRebalanceDetail, error) {
	params, err := fillHistoryParams(startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	var resp []IndexLinkedPlanRebalanceDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/lending/auto-invest/rebalance/history", params, sapiDefaultRate, nil, &resp)
}

// ---------------------------------------- Staking Endpoints  ------------------------------------------------------

// GetSubscribeETHStaking subscribes to staking endpoints.
// Amount in ETH, limit 4 decimals
func (e *Exchange) GetSubscribeETHStaking(ctx context.Context, amount float64) (bool, error) {
	if amount <= 0 {
		return false, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	resp := &struct {
		Success bool `json:"success"`
	}{}
	return resp.Success, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/eth-staking/eth/stake", params, subscribeETHStakingRate, nil, &resp)
}

// SusbcribeETHStakingV2 stake ETH to get WBETH
func (e *Exchange) SusbcribeETHStakingV2(ctx context.Context, amount float64) (*StakingSubscriptionResponse, error) {
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *StakingSubscriptionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v2/eth-staking/eth/stake", params, subscribeETHStakingRate, nil, &resp)
}

// RedeemETH redeem WBETH or BETH and get ETH
func (e *Exchange) RedeemETH(ctx context.Context, amount float64, assetName currency.Code) (*StakingRedemptionResponse, error) {
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	if !assetName.IsEmpty() {
		params.Set("asset", assetName.String())
	}
	var resp *StakingRedemptionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/eth-staking/eth/redeem", params, etherumStakingRedemptionRate, nil, &resp)
}

// GetETHStakingHistory retrieves ETH staking history
func (e *Exchange) GetETHStakingHistory(ctx context.Context, startTime, endTime time.Time, current, size int64) (*ETHStakingHistory, error) {
	params, err := fillHistoryParams(startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	var resp *ETHStakingHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/eth-staking/eth/history/stakingHistory", params, ethStakingHistoryRate, nil, &resp)
}

// GetETHRedemptionHistory retrieves ETH redemption history
func (e *Exchange) GetETHRedemptionHistory(ctx context.Context, startTime, endTime time.Time, current, size int64) (*ETHRedemptionHistory, error) {
	params, err := fillHistoryParams(startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	var resp *ETHRedemptionHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/eth-staking/eth/history/redemptionHistory", params, ethRedemptionHistoryRate, nil, &resp)
}

// GetBETHRewardsDistributionHistory retrieves BETH reward distribution history
func (e *Exchange) GetBETHRewardsDistributionHistory(ctx context.Context, startTime, endTime time.Time, current, size int64) (*BETHRewardDistribution, error) {
	params, err := fillHistoryParams(startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	var resp *BETHRewardDistribution
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/eth-staking/eth/history/rewardsHistory", params, bethRewardDistributionHistoryRate, nil, &resp)
}

// GetCurrentETHStakingQuota retrieves current ETH staking quota
func (e *Exchange) GetCurrentETHStakingQuota(ctx context.Context) (*ETHStakingQuota, error) {
	var resp *ETHStakingQuota
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/eth-staking/eth/quota", nil, currentETHStakingQuotaRate, nil, &resp)
}

// GetWBETHRateHistory retrieves WBETH rate history
func (e *Exchange) GetWBETHRateHistory(ctx context.Context, startTime, endTime time.Time, current, size int64) (*WBETHRateHistory, error) {
	params, err := fillHistoryParams(startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	var resp *WBETHRateHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/eth-staking/eth/history/rateHistory", params, getWBETHRateHistoryRate, nil, &resp)
}

func fillHistoryParams(startTime, endTime time.Time, current, size int64) (url.Values, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if current > 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	return params, nil
}

// GetETHStakingAccount retrieves ETH staking account detail.
func (e *Exchange) GetETHStakingAccount(ctx context.Context) (*ETHStakingAccountDetail, error) {
	var resp *ETHStakingAccountDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/eth-staking/account", nil, ethStakingAccountRate, nil, &resp)
}

// GetETHStakingAccountV2 retrieves V2 ETH staking account detail.
func (e *Exchange) GetETHStakingAccountV2(ctx context.Context) (*StakingAccountV2Response, error) {
	var resp *StakingAccountV2Response
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v2/eth-staking/account", nil, ethStakingAccountRate, nil, &resp)
}

// WrapBETH creates wrapped version of BETH
// amount: Amount in BETH, limit 4 decimals
func (e *Exchange) WrapBETH(ctx context.Context, amount float64) (*WrapBETHResponse, error) {
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *WrapBETHResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/eth-staking/wbeth/wrap", params, wrapBETHRate, nil, &resp)
}

// GetWBETHWrapHistory retrieves a wrap BETH history
func (e *Exchange) GetWBETHWrapHistory(ctx context.Context, startTime, endTime time.Time, current, size int64) (*WBETHWrapHistory, error) {
	return e.getWBETHWrapOrUnwrapHistory(ctx, startTime, endTime, current, size, "/sapi/v1/eth-staking/wbeth/history/wrapHistory")
}

// GetWBETHUnwrapHistory retrieves a WEBTH unwrap BETH history
func (e *Exchange) GetWBETHUnwrapHistory(ctx context.Context, startTime, endTime time.Time, current, size int64) (*WBETHWrapHistory, error) {
	return e.getWBETHWrapOrUnwrapHistory(ctx, startTime, endTime, current, size, "/sapi/v1/eth-staking/wbeth/history/unwrapHistory")
}

func (e *Exchange) getWBETHWrapOrUnwrapHistory(ctx context.Context, startTime, endTime time.Time, current, size int64, path string) (*WBETHWrapHistory, error) {
	params, err := fillHistoryParams(startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	var resp *WBETHWrapHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, wbethWrapOrUnwrapHistoryRate, nil, &resp)
}

// GetWBETHRewardHistory retrieves WBETH rewards history
func (e *Exchange) GetWBETHRewardHistory(ctx context.Context, startTime, endTime time.Time, current, size int64) (*WBETHRewardHistory, error) {
	params, err := fillHistoryParams(startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	var resp *WBETHRewardHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/eth-staking/eth/history/wbethRewardsHistory", params, wbethRewardsHistoryRate, nil, &resp)
}

// GetSOLStakingAccount retrieves SOL staking account
func (e *Exchange) GetSOLStakingAccount(ctx context.Context) (*SOLStakingAccountDetail, error) {
	var resp *SOLStakingAccountDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sol-staking/account", nil, solStakingAccountRate, nil, &resp)
}

// GetSOLStakingQuotaDetails retrieves SOL staking quota
func (e *Exchange) GetSOLStakingQuotaDetails(ctx context.Context) (*SOLStakingQuotaDetail, error) {
	var resp *SOLStakingQuotaDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sol-staking/sol/quota", nil, solStakingQuotaDetailsRate, nil, &resp)
}

// SubscribeToSOLStaking subscribes to SOL staking
func (e *Exchange) SubscribeToSOLStaking(ctx context.Context, amount float64) (*SOLStakingSubscriptionResponse, error) {
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *SOLStakingSubscriptionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/sol-staking/sol/stake", params, subscribeSOLStakingRate, nil, &resp)
}

// RedeemSOL redeem BNSOL and SOL
func (e *Exchange) RedeemSOL(ctx context.Context, amount float64) (*SOLRedemptionResponse, error) {
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *SOLRedemptionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/sol-staking/sol/redeem", nil, redeemSOLRate, nil, &resp)
}

// ClaimBoostRewards claim boost APR airdrop rewards
func (e *Exchange) ClaimBoostRewards(ctx context.Context) (bool, error) {
	resp := &struct {
		Success bool `json:"success"`
	}{}
	return resp.Success, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/sol-staking/sol/claim", nil, claimbBoostReqardsRate, nil, &resp)
}

// GetSOLStakingHistory retrieves SOL staking history
func (e *Exchange) GetSOLStakingHistory(ctx context.Context, startTime, endTime time.Time, current, size int64) (*SOLStakingHistory, error) {
	params, err := fillHistoryParams(startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	var resp *SOLStakingHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sol-staking/sol/history/stakingHistory", params, solStakingHistoryRate, nil, &resp)
}

// GetSOLRedemptionHistory retrieves SOL redemption history
func (e *Exchange) GetSOLRedemptionHistory(ctx context.Context, startTime, endTime time.Time, current, size int64) (*SOLStakingHistory, error) {
	params, err := fillHistoryParams(startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	var resp *SOLStakingHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sol-staking/sol/history/redemptionHistory", params, solRedemptionHistoryRate, nil, &resp)
}

// GetBNSOLRewardsHistory retrieves a BNSOL rewards history
func (e *Exchange) GetBNSOLRewardsHistory(ctx context.Context, startTime, endTime time.Time, current, size int64) (*BNSOLRewardHistory, error) {
	params, err := fillHistoryParams(startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	var resp *BNSOLRewardHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sol-staking/sol/history/bnsolRewardsHistory", params, bnsolRewardsHistoryRate, nil, &resp)
}

// GetBNSOLRateHistory retrieves BNSOL rate history
func (e *Exchange) GetBNSOLRateHistory(ctx context.Context, startTime, endTime time.Time, current, size int64) (*BNSOLRewardHistory, error) {
	params, err := fillHistoryParams(startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	var resp *BNSOLRewardHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sol-staking/sol/history/rateHistory", params, bnsolRateHistory, nil, &resp)
}

// GetBoostRewardsHistory retrieves boosts reward history
func (e *Exchange) GetBoostRewardsHistory(ctx context.Context, rewardType string, startTime, endTime time.Time, current, size int64) (*RewardBoostResponse, error) {
	if rewardType == "" {
		return nil, errRewardTypeMissing
	}
	params, err := fillHistoryParams(startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	params.Set("type", rewardType)
	var resp *RewardBoostResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sol-staking/sol/history/boostRewardsHistory", params, boostRewardsHistoryRate, nil, &resp)
}

// GetUnclaimedRewards get unclaimed rewards
func (e *Exchange) GetUnclaimedRewards(ctx context.Context) ([]Reward, error) {
	var resp []Reward
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sol-staking/sol/history/unclaimedRewards", nil, unclaimedRewardsRate, nil, &resp)
}

// -----------------------------------  Mining Endpoints  -----------------------------
// The endpoints below allow to interact with Binance Pool.
// For more information on this, please refer to the Binance Pool page

// AcquiringAlgorithm retrieves list of algorithms
func (e *Exchange) AcquiringAlgorithm(ctx context.Context) (*AlgorithmsList, error) {
	var resp *AlgorithmsList
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/sapi/v1/mining/pub/algoList", sapiDefaultRate, &resp)
}

// GetCoinNames retrieves coin names
func (e *Exchange) GetCoinNames(ctx context.Context) (*CoinNames, error) {
	var resp *CoinNames
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, "/sapi/v1/mining/pub/coinList", sapiDefaultRate, &resp)
}

// GetDetailMinerList retrieves list of miners name and other details.
func (e *Exchange) GetDetailMinerList(ctx context.Context, algorithm, userName, workerName string) (*MinersDetailList, error) {
	if workerName == "" {
		return nil, fmt.Errorf("%w: worker's name is required", errNameRequired)
	}
	params, err := fillMinersRetrivalParams(algorithm, userName, workerName)
	if err != nil {
		return nil, err
	}
	var resp *MinersDetailList
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/mining/worker/detail", params, getMinersListRate, nil, &resp)
}

// GetMinersList retrieves miners info
func (e *Exchange) GetMinersList(ctx context.Context, algorithm, userName string, sortInNegativeSequence bool, pageIndex, sortColumn, workerStatus int64) (*MinerLists, error) {
	params, err := fillMinersRetrivalParams(algorithm, userName, "")
	if err != nil {
		return nil, err
	}
	if sortInNegativeSequence {
		params.Set("sort", "1")
	}
	if pageIndex > 0 {
		params.Set("pageIndex", strconv.FormatInt(pageIndex, 10))
	}
	// Sort by( default 1):
	// 	1: miner name
	// 	2: real-time computing power
	// 	3: daily average computing power
	// 	4: real-time rejection rate
	// 	5: last submission time
	if sortColumn > 0 {
		params.Set("sortColumn", strconv.FormatInt(sortColumn, 10))
	}
	if workerStatus > 0 {
		params.Set("workerStatus", strconv.FormatInt(workerStatus, 10))
	}
	var resp *MinerLists
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/mining/worker/list", params, getMinersListRate, nil, &resp)
}

func fillMinersRetrivalParams(algorithm, userName, workerName string) (url.Values, error) {
	if algorithm == "" {
		return nil, errTransferAlgorithmRequired
	}
	if userName == "" {
		return nil, fmt.Errorf("%w: mining account name is missing", errNameRequired)
	}
	params := url.Values{}
	params.Set("algo", algorithm)
	params.Set("userName", userName)
	if workerName != "" {
		params.Set("workerName", workerName)
	}
	return params, nil
}

func fillMiningParams(transferAlgorithm, userName string, coin currency.Code, startDate, endDate time.Time, pageIndex, pageSize int64) (url.Values, error) {
	if transferAlgorithm == "" {
		return nil, errTransferAlgorithmRequired
	}
	if userName == "" {
		return nil, errUsernameRequired
	}
	params := url.Values{}
	params.Set("algo", transferAlgorithm)
	params.Set("userName", userName)
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	if !startDate.IsZero() && !endDate.IsZero() {
		err := common.StartEndTimeCheck(startDate, endDate)
		if err != nil {
			return nil, err
		}
		params.Set("startDate", strconv.FormatInt(startDate.UnixMilli(), 10))
		params.Set("endDate", strconv.FormatInt(endDate.UnixMilli(), 10))
	}
	if pageIndex > 0 {
		params.Set("pageIndex", strconv.FormatInt(pageIndex, 10))
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	return params, nil
}

// GetEarningList retrieves list of earning list
func (e *Exchange) GetEarningList(ctx context.Context, transferAlgorithm, userName string, coin currency.Code, startDate, endDate time.Time, pageIndex, pageSize int64) (*EarningList, error) {
	params, err := fillMiningParams(transferAlgorithm, userName, coin, startDate, endDate, pageIndex, pageSize)
	if err != nil {
		return nil, err
	}
	var resp *EarningList
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/mining/payment/list", params, getEarningsListRate, nil, &resp)
}

// ExtraBonousList retrieves extra bonus list
func (e *Exchange) ExtraBonousList(ctx context.Context, transferAlgorithm, userName string, coin currency.Code, startDate, endDate time.Time, pageIndex, pageSize int64) (*ExtraBonus, error) {
	params, err := fillMiningParams(transferAlgorithm, userName, coin, startDate, endDate, pageIndex, pageSize)
	if err != nil {
		return nil, err
	}
	var resp *ExtraBonus
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/mining/payment/other", params, extraBonusListRate, nil, &resp)
}

// GetHashrateRescaleList represents hashrate rescale list
func (e *Exchange) GetHashrateRescaleList(ctx context.Context, pageIndex, pageSize int64) (*HashrateHashTransfers, error) {
	params := url.Values{}
	if pageIndex > 0 {
		params.Set("pageIndex", strconv.FormatInt(pageIndex, 10))
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp *HashrateHashTransfers
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/mining/hash-transfer/config/details/list", params, getHashrateRescaleRate, nil, &resp)
}

// GetHashRateRescaleDetail retrieves a hashrate rescale detail
func (e *Exchange) GetHashRateRescaleDetail(ctx context.Context, configID, userName string, pageIndex, pageSize int64) (*HashrateRescaleDetail, error) {
	if configID == "" {
		return nil, errConfigIDRequired
	}
	if userName == "" {
		return nil, errUsernameRequired
	}
	params := url.Values{}
	params.Set("configId", configID)
	params.Set("userName", userName)
	if pageIndex > 0 {
		params.Set("pageIndex", strconv.FormatInt(pageIndex, 10))
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp *HashrateRescaleDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/mining/hash-transfer/profit/details", params, getHashrateRescaleDetailRate, nil, &resp)
}

// HashRateRescaleRequest retrieves a hashrate rescale request
func (e *Exchange) HashRateRescaleRequest(ctx context.Context, userName, algorithm, toPoolUser string, startTime, endTime time.Time, hashRate int64) (*HashrateRescalResponse, error) {
	if userName == "" {
		return nil, errUsernameRequired
	}
	if algorithm == "" {
		return nil, errTransferAlgorithmRequired
	}
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startDate", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endDate", strconv.FormatInt(endTime.UnixMilli(), 10))
	} else {
		return nil, fmt.Errorf("%w: start time and end time are required", common.ErrDateUnset)
	}
	if toPoolUser == "" {
		return nil, fmt.Errorf("%w: receiver mining account is required", errAccountRequired)
	}
	if hashRate <= 0 {
		return nil, errHashRateRequired
	}
	params.Set("userName", userName)
	params.Set("algo", algorithm)
	params.Set("toPoolUser", toPoolUser)
	params.Set("hashRate", strconv.FormatInt(hashRate, 10))
	var resp *HashrateRescalResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/mining/hash-transfer/config", params, getHasrateRescaleRequestRate, nil, &resp)
}

// CancelHashrateRescaleConfiguration retrieves cancel hashrate rescale configuration
func (e *Exchange) CancelHashrateRescaleConfiguration(ctx context.Context, configID, userName string) (*HashrateRescalResponse, error) {
	if configID == "" {
		return nil, errConfigIDRequired
	}
	if userName == "" {
		return nil, errUsernameRequired
	}
	params := url.Values{}
	params.Set("configId", configID)
	params.Set("userName", userName)
	var resp *HashrateRescalResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/mining/hash-transfer/config/cancel", params, cancelHashrateResaleConfigurationRate, nil, &resp)
}

// StatisticsList represents a statistics list
func (e *Exchange) StatisticsList(ctx context.Context, algorithm, userName string) (*UserStatistics, error) {
	if algorithm == "" {
		return nil, errTransferAlgorithmRequired
	}
	if userName == "" {
		return nil, errUsernameRequired
	}
	params := url.Values{}
	params.Set("algo", algorithm)
	params.Set("userName", userName)
	var resp *UserStatistics
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/mining/statistics/user/status", params, statisticsListRate, nil, &resp)
}

// GetAccountList retrieves account list
func (e *Exchange) GetAccountList(ctx context.Context, algorithm, userName string) (*MiningAccounts, error) {
	if algorithm == "" {
		return nil, errTransferAlgorithmRequired
	}
	if userName == "" {
		return nil, errUsernameRequired
	}
	params := url.Values{}
	params.Set("algo", algorithm)
	params.Set("userName", userName)
	var resp *MiningAccounts
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/mining/statistics/user/list", params, miningAccountListRate, nil, &resp)
}

// GetMiningAccountEarningRate represents a mining account earning rate
func (e *Exchange) GetMiningAccountEarningRate(ctx context.Context, algorithm string, startTime, endTime time.Time, pageIndex, pageSize int64) (*MiningAccountEarnings, error) {
	if algorithm == "" {
		return nil, errTransferAlgorithmRequired
	}
	params := url.Values{}
	params.Set("algo", algorithm)
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startDate", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endDate", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if pageIndex > 0 {
		params.Set("pageIndex", strconv.FormatInt(pageIndex, 10))
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp *MiningAccountEarnings
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/mining/payment/uid", params, miningAccountEarningRate, nil, &resp)
}

// ---------------------------------- Futures Endpoints --------------------------------

// NewFuturesAccountTransfer execute transfer between spot account and futures account.
// transferType
// 1: transfer from spot account to USDT-Ⓜ futures account.
// 2: transfer from USDT-Ⓜ futures account to spot account.
// 3: transfer from spot account to COIN-Ⓜ futures account.
// 4: transfer from COIN-Ⓜ futures account to spot account.
func (e *Exchange) NewFuturesAccountTransfer(ctx context.Context, assetName currency.Code, amount float64, transferType int64) (*FundTransferResponse, error) {
	if assetName.IsEmpty() {
		return nil, fmt.Errorf("%w: assetName is required", currency.ErrCurrencyCodeEmpty)
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if transferType == 0 {
		return nil, errTransferTypeRequired
	}
	params := url.Values{}
	params.Set("asset", assetName.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	params.Set("type", strconv.FormatInt(transferType, 10))
	var resp *FundTransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/futures/transfer", params, sapiDefaultRate, nil, &resp)
}

// GetFuturesAccountTransactionHistoryList retrieves list of futures account transfer transactions.
//
// Support query within the last 6 months only
func (e *Exchange) GetFuturesAccountTransactionHistoryList(ctx context.Context, assetName currency.Code, startTime, endTime time.Time, current, size int64) (*FutureFundTransfers, error) {
	if startTime.IsZero() {
		return nil, errStartTimeRequired
	}
	params, err := fillHistoryParams(startTime, endTime, current, size)
	if err != nil {
		return nil, err
	}
	if !assetName.IsEmpty() {
		params.Set("asset", assetName.String())
	}
	var resp *FutureFundTransfers
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/futures/transfer", params, futuresFundTransfersFetchRate, nil, &resp)
}

// GetFutureTickLevelOrderbookHistoricalDataDownloadLink retrieves the orderbook historical data download link.
//
// dataType: possible values are 'T_DEPTH' for ticklevel orderbook data, 'S_DEPTH' for orderbook snapshot data
func (e *Exchange) GetFutureTickLevelOrderbookHistoricalDataDownloadLink(ctx context.Context, symbol, dataType string, startTime, endTime time.Time) (*HistoricalOrderbookDownloadLink, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if dataType == "" {
		return nil, errors.New("dataType is required, possible values are 'T_DEPTH', and 'S_DEPTH'")
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("dataType", dataType)
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	} else {
		return nil, fmt.Errorf("%w: start time and end time are required", errStartTimeRequired)
	}
	var resp *HistoricalOrderbookDownloadLink
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/futures/histDataLink", params, futureTickLevelOrderbookHistoricalDataDownloadLinkRate, nil, &resp)
}

// ------------------------------  Futures Algo Endpoints  ----------------------------------
//
// Binance Futures Execution Algorithm API solution aims to provide users ability to programmatically
// leverage Binance in-house algorithmic trading capability to automate order execution strategy,
// improve execution transparency and give users smart access to the available market liquidity.

// VolumeParticipationNewOrder send in a VP new order. Only support on USDⓈ-M Contracts.
//
// You need to enable Futures Trading Permission for the api key which requests this endpoint.
// Base URL: https://api.binance.com
func (e *Exchange) VolumeParticipationNewOrder(ctx context.Context, arg *VolumeParticipationOrderParams) (*AlgoOrderResponse, error) {
	if *arg == (VolumeParticipationOrderParams{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Quantity <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.Urgency == "" {
		// Possible values of 'LOW', 'MEDIUM', and 'HIGH'
		return nil, errPossibleValuesRequired
	}
	var resp *AlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/algo/futures/newOrderVp", nil, placeVPOrderRate, arg, &resp)
}

// FuturesTWAPOrder placed futures time-weighted average price(TWAP) order.
func (e *Exchange) FuturesTWAPOrder(ctx context.Context, arg *TWAPOrderParams) (*AlgoOrderResponse, error) {
	if *arg == (TWAPOrderParams{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Quantity <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.Duration == 0 {
		return nil, fmt.Errorf("%w: duration for TWAP orders in seconds. [300, 86400]", errDurationRequired)
	}
	var resp *AlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/algo/futures/newOrderTwap", nil, placeTWAveragePriceNewOrderRate, arg, &resp)
}

// CancelFuturesAlgoOrder cancels futures an active algo order.
//
// You need to enable Futures Trading Permission for the api key which requests this endpoint.
// Base URL: https://api.binance.com
func (e *Exchange) CancelFuturesAlgoOrder(ctx context.Context, algoID int64) (*AlgoOrderResponse, error) {
	return e.cancelAlgoOrder(ctx, algoID, "/sapi/v1/algo/futures/order")
}

func (e *Exchange) cancelAlgoOrder(ctx context.Context, algoID int64, path string) (*AlgoOrderResponse, error) {
	if algoID == 0 {
		return nil, fmt.Errorf("%w: algoId is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("algoId", strconv.FormatInt(algoID, 10))
	var resp *AlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, path, params, sapiDefaultRate, nil, &resp)
}

// GetFuturesCurrentAlgoOpenOrders retrieves futures current algo open orders
//
// You need to enable Futures Trading Permission for the api key which requests this endpoint.
// Base URL: https://api.binance.com
func (e *Exchange) GetFuturesCurrentAlgoOpenOrders(ctx context.Context) (*AlgoOrders, error) {
	var resp *AlgoOrders
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/algo/futures/openOrders", nil, sapiDefaultRate, nil, &resp)
}

// GetFuturesHistoricalAlgoOrders represents a historical algo order instance.
func (e *Exchange) GetFuturesHistoricalAlgoOrders(ctx context.Context, symbol, side string, startTime, endTime time.Time, page, pageSize int64) (*AlgoOrders, error) {
	return e.getHistoricalAlgoOrders(ctx, symbol, side, "/sapi/v1/algo/futures/historicalOrders", startTime, endTime, page, pageSize)
}

func (e *Exchange) getHistoricalAlgoOrders(ctx context.Context, symbol, side, path string, startTime, endTime time.Time, page, pageSize int64) (*AlgoOrders, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if side != "" {
		params.Set("side", side)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp *AlgoOrders
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, sapiDefaultRate, nil, &resp)
}

// GetFuturesSubOrders get respective sub orders for a specified algoId
//
// You need to enable Futures Trading Permission for the api key which requests this endpoint.
// Base URL: https://api.binance.com
func (e *Exchange) GetFuturesSubOrders(ctx context.Context, algoID, page, pageSize int64) (*AlgoSubOrders, error) {
	return e.getSubOrders(ctx, algoID, page, pageSize, "/sapi/v1/algo/futures/subOrders")
}

func (e *Exchange) getSubOrders(ctx context.Context, algoID, page, pageSize int64, path string) (*AlgoSubOrders, error) {
	if algoID == 0 {
		return nil, fmt.Errorf("%w: algoId is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("algoId", strconv.FormatInt(algoID, 10))
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if pageSize > 0 {
		params.Set("pageSize", strconv.FormatInt(pageSize, 10))
	}
	var resp *AlgoSubOrders
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, sapiDefaultRate, nil, &resp)
}

// -----------------------------------------  Spot Algo Endpoints  --------------------------------------------
// Binance Spot Execution Algorithm API solution aims to provide users ability to programmatically leverage Binance in-house algorithmic trading capability to automate order execution strategy,
// improve execution transparency and give users smart access to the available market liquidity. During the introductory period, there will be no additional fees for TWAP orders.
// Standard trading fees apply. Order size exceeds to maximum API supported size (100,000 USDT). Please contact liquidity@binance.com for larger sizes.

// SpotTWAPNewOrder puts spot Time-Weighted Average Price(TWAP) orders
func (e *Exchange) SpotTWAPNewOrder(ctx context.Context, arg *SpotTWAPOrderParam) (*AlgoOrderResponse, error) {
	if *arg == (SpotTWAPOrderParam{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Quantity <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.Duration == 0 {
		return nil, fmt.Errorf("%w: duration for TWAP orders in seconds. [300, 86400]", errDurationRequired)
	}
	var resp *AlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/algo/spot/newOrderTwap", nil, spotTwapNewOrderRate, arg, &resp)
}

// CancelSpotAlgoOrder cancels an open spot TWAP order
func (e *Exchange) CancelSpotAlgoOrder(ctx context.Context, algoID int64) (*AlgoOrderResponse, error) {
	return e.cancelAlgoOrder(ctx, algoID, "/sapi/v1/algo/spot/order")
}

// GetCurrentSpotAlgoOpenOrder retrieves all open SPOT TWAP orders.
func (e *Exchange) GetCurrentSpotAlgoOpenOrder(ctx context.Context) (*AlgoOrders, error) {
	var resp *AlgoOrders
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/algo/spot/openOrders", nil, sapiDefaultRate, nil, &resp)
}

// GetSpotHistoricalAlgoOrders retrieves all historical SPOT TWAP Orders
func (e *Exchange) GetSpotHistoricalAlgoOrders(ctx context.Context, symbol, side string, startTime, endTime time.Time, page, pageSize int64) (*AlgoOrders, error) {
	return e.getHistoricalAlgoOrders(ctx, symbol, side, "/sapi/v1/algo/spot/historicalOrders", startTime, endTime, page, pageSize)
}

// GetSpotSubOrders get respective sub orders for a specified algoId
func (e *Exchange) GetSpotSubOrders(ctx context.Context, algoID, page, pageSize int64) (*AlgoSubOrders, error) {
	return e.getSubOrders(ctx, algoID, page, pageSize, "/sapi/v1/algo/spot/subOrders")
}

// -------------------------------------- Classic Portfolio Margin Endpoints -------------------------------------------
// The Binance Classic Portfolio Margin Program is a cross-asset margin program supporting consolidated margin balance across trading products with over 200+ effective crypto collaterals.
// It is designed for professional traders, market makers, and institutional users looking to actively trade & hedge cross-asset and optimize risk-management in a consolidated setup.
// Only Classic Portfolio Margin Account is accessible to these endpoints.

// GetClassicPortfolioMarginAccountInfo retrieves classic portfolio margin account information.
func (e *Exchange) GetClassicPortfolioMarginAccountInfo(ctx context.Context) (*ClassicPMAccountInfo, error) {
	var resp *ClassicPMAccountInfo
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/portfolio/account", nil, classicPMAccountInfoRate, nil, &resp)
}

// GetClassicPortfolioMarginCollateralRate retrieves classic Portfolio Margin Collateral Rate
func (e *Exchange) GetClassicPortfolioMarginCollateralRate(ctx context.Context) ([]PMCollateralRate, error) {
	var resp []PMCollateralRate
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/portfolio/collateralRate", nil, classicPMCollateralRate, nil, &resp)
}

// GetClassicPortfolioMarginBankruptacyLoanAmount query Classic Portfolio Margin Bankruptcy Loan Amount
func (e *Exchange) GetClassicPortfolioMarginBankruptacyLoanAmount(ctx context.Context) (*PMBankruptacyLoanAmount, error) {
	var resp *PMBankruptacyLoanAmount
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/portfolio/pmLoan", nil, getClassicPMBankruptacyLoanAmountRate, nil, &resp)
}

// RepayClassicPMBankruptacyLoan repay Classic Portfolio Margin Bankruptcy Loan
// from: SPOT or MARGIN，default SPOT
func (e *Exchange) RepayClassicPMBankruptacyLoan(ctx context.Context, from string) (*FundTransferResponse, error) {
	params := url.Values{}
	if from != "" {
		params.Set("from", from)
	}
	var resp *FundTransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/portfolio/repay", params, repayClassicPMBankruptacyLoanRate, nil, &resp)
}

// GetClassicPMNegativeBalanceInterestHistory query interest history of negative balance for portfolio margin.
func (e *Exchange) GetClassicPMNegativeBalanceInterestHistory(ctx context.Context, assetName currency.Code, startTime, endTime time.Time, size int64) ([]PMNegativeBalaceInterestHistory, error) {
	params := url.Values{}
	if !assetName.IsEmpty() {
		params.Set("asset", assetName.String())
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	var resp []PMNegativeBalaceInterestHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/portfolio/interest-history", params, classicPMNegativeBalanceInterestHistory, nil, &resp)
}

// GetPMAssetIndexPrice query Portfolio Margin Asset Index Price
func (e *Exchange) GetPMAssetIndexPrice(ctx context.Context, assetName currency.Code) ([]PMIndexPrice, error) {
	params := url.Values{}
	endpointLimit := pmAssetIndexPriceRate
	if !assetName.IsEmpty() {
		endpointLimit = sapiDefaultRate
		params.Set("asset", assetName.String())
	}
	var resp []PMIndexPrice
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, common.EncodeURLValues("/sapi/v1/portfolio/asset-index-price", params), endpointLimit, &resp)
}

// ClassicPMFundAutoCollection transfers all assets from Futures Account to Margin account
//
// The BNB would not be collected from UM-PM account to the Portfolio Margin account.
// You can only use this function 500 times per hour in a rolling manner.
func (e *Exchange) ClassicPMFundAutoCollection(ctx context.Context) (*FundAutoCollectionResponse, error) {
	var resp *FundAutoCollectionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/portfolio/auto-collection", nil, fundAutoCollectionRate, nil, &resp)
}

// ClassicFundCollectionByAsset transfers specific asset from Futures Account to Margin account
func (e *Exchange) ClassicFundCollectionByAsset(ctx context.Context, assetName currency.Code) (*FundAutoCollectionResponse, error) {
	if assetName.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("asset", assetName.String())
	var resp *FundAutoCollectionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/portfolio/asset-collection", params, fundCollectionByAssetRate, nil, &resp)
}

// BNBTransferClassic BNB transfer can be between Margin Account and USDM Account
// transferSide: "TO_UM","FROM_UM"
func (e *Exchange) BNBTransferClassic(ctx context.Context, amount float64, transferSide string) (int64, error) {
	return e.bnbTransfer(ctx, amount, transferSide, "/sapi/v1/portfolio/bnb-transfer", transferBNBRate, exchange.RestSpot)
}

// ChangeAutoRepayFuturesStatusClassic change Auto-repay-futures Status
func (e *Exchange) ChangeAutoRepayFuturesStatusClassic(ctx context.Context, autoRepay bool) (string, error) {
	return e.changeAutoRepayFuturesStatus(ctx, autoRepay, exchange.RestSpot, "/sapi/v1/portfolio/repay-futures-switch", changeAutoRepayFuturesStatusRate)
}

// GetAutoRepayFuturesStatusClassic get Auto-repay-futures Status
func (e *Exchange) GetAutoRepayFuturesStatusClassic(ctx context.Context) (*AutoRepayStatus, error) {
	var resp *AutoRepayStatus
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/portfolio/repay-futures-switch", nil, getAutoRepayFuturesStatusRate, nil, &resp)
}

// RepayFuturesNegativeBalanceClassic represents a classic repay futures negative balance
func (e *Exchange) RepayFuturesNegativeBalanceClassic(ctx context.Context) (string, error) {
	resp := &struct {
		Message string `json:"msg"`
	}{}
	return resp.Message, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/portfolio/repay-futures-negative-balance", nil, repayFuturesNegativeBalanceRate, nil, &resp)
}

// GetPortfolioMarginAssetLeverage retrieves portfolio margin asset leverage classic
func (e *Exchange) GetPortfolioMarginAssetLeverage(ctx context.Context) ([]PMAssetLeverage, error) {
	var resp []PMAssetLeverage
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/portfolio/margin-asset-leverage", nil, pmAssetLeverageRate, nil, &resp)
}

// GetUserNegativeBalanceAutoExchangeRecord retrieves user negative balance auto exchange record
func (e *Exchange) GetUserNegativeBalanceAutoExchangeRecord(ctx context.Context, startTime, endTime time.Time) (*UserNegativeBalanceRecord, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	} else {
		return nil, errStartAndEndTimeRequired
	}
	var resp *UserNegativeBalanceRecord
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/papi/v1/portfolio/negative-balance-exchange-record", params, request.UnAuth, nil, &resp)
}

// ----------------------------------  Binance Leverate Token(BLVT) Endpoints  ------------------------------

// GetBLVTInfo retrieves details of binance leverage tokens.
func (e *Exchange) GetBLVTInfo(ctx context.Context, tokenName string) ([]BLVTTokenDetail, error) {
	params := url.Values{}
	if tokenName != "" {
		params.Set("tokenName", tokenName)
	}
	var resp []BLVTTokenDetail
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, common.EncodeURLValues("/sapi/v1/blvt/tokenInfo", params), sapiDefaultRate, &resp)
}

// SubscribeBLVT subscribe to BLVT token
func (e *Exchange) SubscribeBLVT(ctx context.Context, tokenName string, cost float64) (*BLVTSubscriptionResponse, error) {
	if tokenName == "" {
		return nil, fmt.Errorf("%w: tokenName is missing", errNameRequired)
	}
	if cost <= 0 {
		return nil, errCostRequired
	}
	params := url.Values{}
	params.Set("tokenName", tokenName)
	params.Set("cost", strconv.FormatFloat(cost, 'f', -1, 64))
	var resp *BLVTSubscriptionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/blvt/subscribe", params, sapiDefaultRate, nil, &resp)
}

// GetSusbcriptionRecords retrieves BLVT tokens subscriptions
func (e *Exchange) GetSusbcriptionRecords(ctx context.Context, tokenName string, startTime, endTime time.Time, id, limit int64) ([]BLVTTokenSubscriptionItem, error) {
	params := url.Values{}
	if tokenName != "" {
		params.Set("tokenName", tokenName)
	}
	if id > 0 {
		params.Set("id", strconv.FormatInt(id, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []BLVTTokenSubscriptionItem
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/blvt/subscribe/record", params, sapiDefaultRate, nil, &resp)
}

// RedeemBLVT redeems a BLVT token
// You need to openEnable Spot&Margin Trading permission for the API Key which requests this endpoint.
func (e *Exchange) RedeemBLVT(ctx context.Context, symbol string, amount float64) (*BLVTRedemption, error) {
	if symbol == "" {
		return nil, fmt.Errorf("%w: tokenName is missing", currency.ErrSymbolStringEmpty)
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("tokenName", symbol)
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *BLVTRedemption
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/blvt/redeem", params, sapiDefaultRate, nil, &resp)
}

// GetRedemptionRecord retrieves BLVT redemption records
func (e *Exchange) GetRedemptionRecord(ctx context.Context, tokenName string, startTime, endTime time.Time, id, limit int64) ([]BLVTRedemptionItem, error) {
	params := url.Values{}
	if tokenName != "" {
		params.Set("tokenName", tokenName)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if id > 0 {
		params.Set("id", strconv.FormatInt(id, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []BLVTRedemptionItem
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/blvt/redeem/record", params, sapiDefaultRate, nil, &resp)
}

// GetBLVTUserLimitInfo represents a BLVT user limit information.
func (e *Exchange) GetBLVTUserLimitInfo(ctx context.Context, tokenName string) ([]BLVTUserLimitInfo, error) {
	params := url.Values{}
	if tokenName != "" {
		params.Set("tokenName", tokenName)
	}
	var resp []BLVTUserLimitInfo
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/blvt/userLimit", params, sapiDefaultRate, nil, &resp)
}

// TODO: Websocket BLVT Info Streams
// https://binance-docs.github.io/apidocs/spot/en/#get-blvt-user-limit-info-user_data

// --------------------------------------------  Fiat Endpoints  ----------------------------------------------

func fillFiatFetchParams(beginTime, endTime time.Time, transactionType, page, rows int64) (url.Values, error) {
	params := url.Values{}
	params.Set("transactionType", strconv.FormatInt(transactionType, 10))
	if !beginTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(beginTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("beginTime", strconv.FormatInt(beginTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if rows > 0 {
		params.Set("rows", strconv.FormatInt(rows, 10))
	}
	return params, nil
}

// ---------------------------------------------------------------- Fiat endpoints ----------------------------------------------------------------

// GetFiatDepositAndWithdrawalHistory represents a fiat deposit and withdrawal history
// transactionType possible values are 0 for deposit and 1 for withdrawal
func (e *Exchange) GetFiatDepositAndWithdrawalHistory(ctx context.Context, beginTime, endTime time.Time, transactionType, page, rows int64) (*FiatTransactionHistory, error) {
	if transactionType != 0 && transactionType != 1 {
		return nil, fmt.Errorf("%w: possible values are 0 for 'deposit' and '1' for withdrawal", errInvalidTransactionType)
	}
	params, err := fillFiatFetchParams(beginTime, endTime, transactionType, page, rows)
	if err != nil {
		return nil, err
	}
	var resp *FiatTransactionHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/fiat/orders", params, fiatDepositWithdrawHistRate, nil, &resp)
}

// GetFiatPaymentHistory represents a fiat payment history.
//
// paymentMethod: Only when requesting payments history for buy (transactionType=0), response contains paymentMethod representing the way of purchase. Now we have:
// - Cash Balance
// - Credit Card
// - Online Banking
// - Bank Transfer
func (e *Exchange) GetFiatPaymentHistory(ctx context.Context, beginTime, endTime time.Time, transactionType, page, rows int64) (*FiatPaymentHistory, error) {
	if transactionType != 0 && transactionType != 1 {
		return nil, fmt.Errorf("%w: possible values are 0 for 'buy' and 1 for 'sell'", errInvalidTransactionType)
	}
	params, err := fillFiatFetchParams(beginTime, endTime, transactionType, page, rows)
	if err != nil {
		return nil, err
	}
	var resp *FiatPaymentHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/fiat/payments", params, sapiDefaultRate, nil, &resp)
}

// ------------------------------------------------------  Peer-To-Peer(C2C) Endpoints  --------------------------------------------------------------

// GetC2CTradeHistory represents a peer-to-peer trade history
// To view the complete P2P order history, you can download it from https://c2c.binance.com/en/fiatOrder
// possible trade type values: SELL or BUY
func (e *Exchange) GetC2CTradeHistory(ctx context.Context, tradeType string, startTime, endTime time.Time, page, rows int64) (*C2CTransaction, error) {
	if tradeType == "" {
		return nil, errTradeTypeRequired
	}
	params := url.Values{}
	params.Set("tradeType", tradeType)
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTimestamp", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTimestamp", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if rows > 0 {
		params.Set("rows", strconv.FormatInt(rows, 10))
	}
	var resp *C2CTransaction
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/c2c/orderMatch/listUserOrderHistory", params, sapiDefaultRate, nil, &resp)
}

// ------------------------------------------  VIP Loan endpoints ------------------------------------------------

// GetVIPLoanOngoingOrders retrieves VIP loan is available for VIP users only.
func (e *Exchange) GetVIPLoanOngoingOrders(ctx context.Context, orderID, collateralAccountID, current, limit int64, loanCoin, collateralCoin currency.Code) (*VIPLoanOngoingOrders, error) {
	params := url.Values{}
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if collateralAccountID != 0 {
		params.Set("collateralAccountId", strconv.FormatInt(collateralAccountID, 10))
	}
	if loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.String())
	}
	if collateralCoin.IsEmpty() {
		params.Set("collateralCoin", collateralCoin.String())
	}
	if current > 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *VIPLoanOngoingOrders
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/loan/vip/ongoing/orders", params, getVIPLoanOngoingOrdersRate, nil, &resp)
}

// VIPLoanRepay VIP loan is available for VIP users only.
func (e *Exchange) VIPLoanRepay(ctx context.Context, orderID int64, amount float64) (*VIPLoanRepayResponse, error) {
	if orderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("orderId", strconv.FormatInt(orderID, 10))
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *VIPLoanRepayResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/loan/vip/repay", params, vipLoanRepayRate, nil, &resp)
}

// GetVIPLoanRepaymentHistory retrieves VIP loan repayment history
func (e *Exchange) GetVIPLoanRepaymentHistory(ctx context.Context, loanCoin currency.Code, startTime, endTime time.Time, orderID, current, limit int64) (*VIPLoanRepaymentHistoryResponse, error) {
	params, err := fillHistoryParams(startTime, endTime, current, 0)
	if err != nil {
		return nil, err
	}
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if !loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *VIPLoanRepaymentHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/loan/vip/repay/history", params, getVIPLoanRepaymentHistoryRate, nil, &resp)
}

// GetVIPLoanAccruedInterest retrieves VIP loan accrued interest
func (e *Exchange) GetVIPLoanAccruedInterest(ctx context.Context, orderID string, loanCoin currency.Code, startTime, endTime time.Time, current, limit int64) (*VIPLoanAccruedInterests, error) {
	params, err := fillHistoryParams(startTime, endTime, current, 0)
	if err != nil {
		return nil, err
	}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if !loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *VIPLoanAccruedInterests
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/loan/vip/accruedInterest", params, getVIPLoanAccruedInterest, nil, &resp)
}

// VIPLoanRenew represents VIP loan is available for VIP users only.
func (e *Exchange) VIPLoanRenew(ctx context.Context, orderID, longTerm int64) (*LoanRenewResponse, error) {
	if orderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("orderId", strconv.FormatInt(orderID, 10))
	if longTerm != 0 {
		params.Set("longTerm", strconv.FormatInt(longTerm, 10))
	}
	var resp *LoanRenewResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/loan/vip/renew", params, vipLoanRenewRate, nil, &resp)
}

// CheckLockedValueVIPCollateralAccount VIP loan is available for VIP users only.
func (e *Exchange) CheckLockedValueVIPCollateralAccount(ctx context.Context, orderID, collateralAccountID int64) (*LockedValueVIPCollateralAccount, error) {
	if orderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if collateralAccountID == 0 {
		return nil, fmt.Errorf("%w: collateral Account ID is missing", errAccountIDRequired)
	}
	params := url.Values{}
	params.Set("orderId", strconv.FormatInt(orderID, 10))
	params.Set("collateralAccountId", strconv.FormatInt(collateralAccountID, 10))
	var resp *LockedValueVIPCollateralAccount
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/loan/vip/collateral/account", params, checkLockedValueVIPCollateralAccountRate, nil, &resp)
}

// VIPLoanBorrow VIP loan is available for VIP users only.
func (e *Exchange) VIPLoanBorrow(ctx context.Context, loanAccountID, loanTerm int64, loanCoin, collateralCoin currency.Code, loanAmount float64, collateralAccountID string, isFlexibleRate bool) ([]VIPLoanBorrow, error) {
	if loanAccountID == 0 {
		return nil, fmt.Errorf("%w: loanAccountId is required", errAccountIDRequired)
	}
	if loanCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: loanCoin is required", currency.ErrCurrencyCodeEmpty)
	}
	if loanAmount <= 0 {
		return nil, fmt.Errorf("%w: loanAmount is required", limits.ErrAmountBelowMin)
	}
	if collateralAccountID == "" {
		return nil, fmt.Errorf("%w: collateralAccountID is required", errAccountIDRequired)
	}
	if collateralCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: collateralCoin is required", currency.ErrCurrencyCodeEmpty)
	}
	if loanTerm == 0 {
		return nil, errLoanTermMustBeSet
	}
	params := url.Values{}
	params.Set("loanAccountId", strconv.FormatInt(loanAccountID, 10))
	params.Set("loanCoin", loanCoin.String())
	params.Set("loanAmount", strconv.FormatFloat(loanAmount, 'f', -1, 64))
	params.Set("loanTerm", strconv.FormatInt(loanTerm, 10))
	params.Set("collateralAccountId", collateralAccountID)
	params.Set("collateralCoin", collateralCoin.String())
	if isFlexibleRate {
		params.Set("isFlexible", "TRUE")
	} else {
		params.Set("isFlexible", "FALSE")
	}
	var resp []VIPLoanBorrow
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/loan/vip/borrow", params, vipLoanBorrowRate, nil, &resp)
}

// GetVIPLoanableAssetsData get interest rate and borrow limit of loanable assets. The borrow limit is shown in USD value.
func (e *Exchange) GetVIPLoanableAssetsData(ctx context.Context, loanCoin currency.Code, vipLevel int64) (*VIPLoanableAssetsData, error) {
	params := url.Values{}
	if !loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.String())
	}
	if vipLevel > 0 {
		params.Set("vipLevel", strconv.FormatInt(vipLevel, 10))
	}
	var resp *VIPLoanableAssetsData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/loan/vip/loanable/data", params, getVIPLoanableAssetsRate, nil, &resp)
}

// GetVIPCollateralAssetData retrieves Collateral Asset Data
func (e *Exchange) GetVIPCollateralAssetData(ctx context.Context, collateralCoin currency.Code) (*VIPCollateralAssetData, error) {
	params := url.Values{}
	if !collateralCoin.IsEmpty() {
		params.Set("collateralCoin", collateralCoin.String())
	}
	var resp *VIPCollateralAssetData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/loan/vip/collateral/data", params, getCollateralAssetDataRate, nil, &resp)
}

// GetVIPApplicationStatus retrieves a loan application status
func (e *Exchange) GetVIPApplicationStatus(ctx context.Context, current, limit int64) (*LoanApplicationStatus, error) {
	params := url.Values{}
	if current > 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *LoanApplicationStatus
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/loan/vip/request/data", params, getApplicationStatusRate, nil, &resp)
}

// GetVIPBorrowInterestRate represents an interest rates of loaned coin.
func (e *Exchange) GetVIPBorrowInterestRate(ctx context.Context, loanCoin currency.Code) ([]BorrowInterestRate, error) {
	if loanCoin.IsEmpty() {
		return nil, fmt.Errorf("%w: loanCoin is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("loanCoin", loanCoin.String())
	var resp []BorrowInterestRate
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/loan/vip/request/interestRate", params, getVIPBorrowInterestRate, nil, &resp)
}

// GetVIPLoanInterestRateHistory retrieves VIP Loan Interest Rate History
func (e *Exchange) GetVIPLoanInterestRateHistory(ctx context.Context, coin currency.Code, startTime, endTime time.Time, current, limit int64) (*VIPLoanInterestRate, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params, err := fillHistoryParams(startTime, endTime, current, 0)
	if err != nil {
		return nil, err
	}
	params.Set("coin", coin.String())
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *VIPLoanInterestRate
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/loan/vip/interestRateHistory", params, vipLoanInterestRateHistoryRate, nil, &resp)
}

// ------------------------------------------------ Pay Endpoints ----------------------------------------------

// GetPayTradeHistory retrieves pay trade history
// Detail found here: https://binance-docs.github.io/apidocs/spot/en/#pay-endpoints
func (e *Exchange) GetPayTradeHistory(ctx context.Context, startTime, endTime time.Time, limit int64) (*PayTradeHistory, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTimestamp", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTimestamp", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *PayTradeHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/pay/transactions", params, payTradeEndpointsRate, nil, &resp)
}

// ---------------------------------------------------------  Convert Endpoints  -------------------------------------------------------

// GetAllConvertPairs query for all convertible token pairs and the tokens’ respective upper/lower limits
// If not defined for both fromAsset and toAsset, only partial token pairs will be return
func (e *Exchange) GetAllConvertPairs(ctx context.Context, fromAsset, toAsset currency.Code) ([]ConvertPairInfo, error) {
	if fromAsset.IsEmpty() && toAsset.IsEmpty() {
		return nil, fmt.Errorf("%w: either fromAsset or toAsset is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	if !fromAsset.IsEmpty() {
		params.Set("fromAsset", fromAsset.String())
	}
	if !toAsset.IsEmpty() {
		params.Set("toAsset", toAsset.String())
	}
	var resp []ConvertPairInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/sapi/v1/convert/exchangeInfo", params), getAllConvertPairsRate, &resp)
}

// GetOrderQuantityPrecisionPerAsset query for supported asset’s precision information
func (e *Exchange) GetOrderQuantityPrecisionPerAsset(ctx context.Context) ([]OrderQuantityPrecision, error) {
	var resp []OrderQuantityPrecision
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/convert/assetInfo", nil, getOrderQuantityPrecisionPerAssetRate, nil, &resp)
}

// SendQuoteRequest request a quote for the requested token pairs
// validTime possible values: 10s, 30s, 1m, 2m, default 10s
// quoteId will be returned only if you have enough funds to convert
func (e *Exchange) SendQuoteRequest(ctx context.Context, fromAsset, toAsset currency.Code, fromAmount, toAmount float64, walletType, validTime string) (*ConvertQuoteResponse, error) {
	if fromAsset.IsEmpty() {
		return nil, fmt.Errorf("%w: fromAsset is required", currency.ErrCurrencyCodeEmpty)
	}
	if toAsset.IsEmpty() {
		return nil, fmt.Errorf("%w: toAsset is required", currency.ErrCurrencyCodeEmpty)
	}
	if fromAmount <= 0 && toAmount <= 0 {
		return nil, fmt.Errorf("%w: fromAmount or toAmount is required", order.ErrAmountIsInvalid)
	}
	params := url.Values{}
	params.Set("fromAsset", fromAsset.String())
	params.Set("toAsset", toAsset.String())
	if fromAmount > 0 {
		params.Set("fromAmount", strconv.FormatFloat(fromAmount, 'f', -1, 64))
	}
	if toAmount > 0 {
		params.Set("toAmount", strconv.FormatFloat(toAmount, 'f', -1, 64))
	}
	if walletType != "" {
		params.Set("walletType", walletType)
	}
	if validTime != "" {
		params.Set("validTime", validTime)
	}
	var resp *ConvertQuoteResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/convert/getQuote", params, sendQuoteRequestRate, nil, &resp)
}

// AcceptQuote accept the offered quote by quote ID.
func (e *Exchange) AcceptQuote(ctx context.Context, quoteID string) (*QuoteOrderStatus, error) {
	if quoteID == "" {
		return nil, errQuoteIDRequired
	}
	params := url.Values{}
	params.Set("quoteId", quoteID)
	var resp *QuoteOrderStatus
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/convert/acceptQuote", params, acceptQuoteRate, nil, &resp)
}

// GetConvertOrderStatus retrieves order status by order ID.
func (e *Exchange) GetConvertOrderStatus(ctx context.Context, orderID, quoteID string) (*ConvertOrderStatus, error) {
	params := url.Values{}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if quoteID != "" {
		params.Set("quoteId", quoteID)
	}
	var resp *ConvertOrderStatus
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/convert/orderStatus", params, orderStatusRate, nil, &resp)
}

// PlaceLimitOrder enable users to place a limit order
func (e *Exchange) PlaceLimitOrder(ctx context.Context, arg *ConvertPlaceLimitOrderParam) (*OrderStatusResponse, error) {
	if *arg == (ConvertPlaceLimitOrderParam{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.BaseAsset.IsEmpty() {
		return nil, fmt.Errorf("%w: baseAsset is required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.QuoteAsset.IsEmpty() {
		return nil, fmt.Errorf("%w: quoteAsset is required", currency.ErrCurrencyCodeEmpty)
	}
	if arg.LimitPrice <= 0 {
		return nil, fmt.Errorf("%w: limitPrice is required", limits.ErrPriceBelowMin)
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.ExpiredType == "" {
		return nil, errExpiredTypeRequired
	}
	var resp *OrderStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/convert/limit/placeOrder", nil, placeLimitOrderRate, arg, &resp)
}

// CancelLimitOrder cancels a limit order
func (e *Exchange) CancelLimitOrder(ctx context.Context, orderID string) (*OrderStatusResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("orderId", orderID)
	var resp *OrderStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/convert/limit/cancelOrder", params, cancelLimitOrderRate, nil, &resp)
}

// GetLimitOpenOrders represents users to query for all existing limit orders
func (e *Exchange) GetLimitOpenOrders(ctx context.Context) (*LimitOrderHistory, error) {
	var resp *LimitOrderHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/convert/limit/queryOpenOrders", nil, getLimitOpenOrdersRate, nil, &resp)
}

// GetConvertTradeHistory represents a convert trade history
func (e *Exchange) GetConvertTradeHistory(ctx context.Context, startTime, endTime time.Time, limit int64) (*ConvertTradeHistory, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *ConvertTradeHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/convert/tradeFlow", params, convertTradeFlowHistoryRate, nil, &resp)
}

// -----------------------------------------------  Rebate Endpoints  -------------------------------------------------

// GetSpotRebateHistoryRecords represents a rebate history records
func (e *Exchange) GetSpotRebateHistoryRecords(ctx context.Context, startTime, endTime time.Time, page int64) (*RebateHistory, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	var resp *RebateHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/rebate/taxQuery", params, spotRebateHistoryRate, nil, &resp)
}

// -----------------------------------------  NFT Endpoints ------------------------------------------------

func fillNFTFetchParams(startTime, endTime time.Time, limit, page int64) (url.Values, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	return params, nil
}

// GetNFTTransactionHistory represents an NFT transaction history
// orderType: 0: purchase order, 1: sell order, 2: royalty income, 3: primary market order, 4: mint fee
func (e *Exchange) GetNFTTransactionHistory(ctx context.Context, orderType int64, startTime, endTime time.Time, limit, page int64) (*NFTTransactionHistory, error) {
	if orderType < 0 || orderType > 4 {
		return nil, fmt.Errorf("%w 0: purchase order, 1: sell order, 2: royalty income, 3: primary market order, 4: mint fee", order.ErrUnsupportedOrderType)
	}
	params, err := fillNFTFetchParams(startTime, endTime, limit, page)
	if err != nil {
		return nil, err
	}
	params.Set("orderType", strconv.FormatInt(orderType, 10))
	var resp *NFTTransactionHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/nft/history/transactions", params, nftRate, nil, &resp)
}

// ----------------- NFT endpoints ----------------

// GetNFTDepositHistory retrieves list of deposit history
func (e *Exchange) GetNFTDepositHistory(ctx context.Context, startTime, endTime time.Time, limit, page int64) (*NFTDepositHistory, error) {
	params, err := fillNFTFetchParams(startTime, endTime, limit, page)
	if err != nil {
		return nil, err
	}
	var resp *NFTDepositHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/nft/history/deposit", params, nftRate, nil, &resp)
}

// GetNFTWithdrawalHistory retrieves list of withdrawal history
func (e *Exchange) GetNFTWithdrawalHistory(ctx context.Context, startTime, endTime time.Time, limit, page int64) (*NFTWithdrawalHistory, error) {
	params, err := fillNFTFetchParams(startTime, endTime, limit, page)
	if err != nil {
		return nil, err
	}
	var resp *NFTWithdrawalHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/nft/history/withdraw", params, nftRate, nil, &resp)
}

// GetNFTAsset retrieves an NFT assets
func (e *Exchange) GetNFTAsset(ctx context.Context, limit, page int64) (*NFTAssets, error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	var resp *NFTAssets
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/nft/user/getAsset", params, nftRate, nil, &resp)
}

// --------------------------------------------------- Binance Gift Card Endpoints --------------------------------------------------
// Binance Gift Card allows simple crypto transfer and exchange through secured and prepaid codes.
// Binance Gift Card API solution is to facilitate instant creation, redemption and value-checking for Binance Gift Card.
// Binance Gift Card product feature consists of two parts: “Gift Card Number" and “Binance Gift Card Redemption Code".
// The Gift Card Number can be circulated in public, and it is used to verify the validity of the Binance Gift Card;
// Binance Gift Card Redemption Code should be kept confidential, because as long as someone knows the redemption code, that person can redeem it anytime.

// CreateSingleTokenGiftCard creating a Binance Gift Card.
//
// Daily creation volume: 2 BTC / 24H / account
// Daily creation quantity: 200 Gift Cards / 24H / account
func (e *Exchange) CreateSingleTokenGiftCard(ctx context.Context, token currency.Code, amount float64) (*GiftCard, error) {
	if token.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("token", token.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *GiftCard
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/giftcard/createCode", params, sapiDefaultRate, nil, &resp)
}

// CreateDualTokenGiftCard creating a dual-token ( stablecoin-denominated) Binance Gift Card.
func (e *Exchange) CreateDualTokenGiftCard(ctx context.Context, baseToken, faceToken currency.Code, baseTokenAmount, discount float64) (*DualTokenGiftCard, error) {
	if baseToken.IsEmpty() {
		return nil, fmt.Errorf("%w: baseToken is empty", currency.ErrCurrencyCodeEmpty)
	}
	if faceToken.IsEmpty() {
		return nil, fmt.Errorf("%w: faceToken is empty", currency.ErrCurrencyCodeEmpty)
	}
	if baseTokenAmount <= 0 {
		return nil, fmt.Errorf("%w: baseTokenAmount is %f", limits.ErrAmountBelowMin, baseTokenAmount)
	}
	if discount <= 0 {
		return nil, fmt.Errorf("%w: discount must be greater than zero", limits.ErrAmountBelowMin)
	}
	params := url.Values{}
	params.Set("baseToken", baseToken.String())
	params.Set("faceToken", faceToken.String())
	params.Set("baseTokenAmount", strconv.FormatFloat(baseTokenAmount, 'f', -1, 64))
	params.Set("discount", strconv.FormatFloat(discount, 'f', -1, 64))
	var resp *DualTokenGiftCard
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/giftcard/buyCode", params, sapiDefaultRate, nil, &resp)
}

// RedeemBinanaceGiftCard redeeming a Binance Gift Card.
// Once redeemed, the coins will be deposited in your funding wallet.
// Redemption code of Binance Gift Card to be redeemed, supports both Plaintext & Encrypted code.
// Each external unique ID represents a unique user on the partner platform.
func (e *Exchange) RedeemBinanaceGiftCard(ctx context.Context, code, externalUID string) (*RedeemBinanceGiftCard, error) {
	if code == "" {
		return nil, errCodeRequired
	}
	params := url.Values{}
	params.Set("code", code)
	if externalUID != "" {
		params.Set("expternalUid", externalUID)
	}
	var resp *RedeemBinanceGiftCard
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/giftcard/redeemCode", params, sapiDefaultRate, nil, &resp)
}

// VerifyBinanceGiftCardNumber verifying whether the Binance Gift Card is valid or not by entering Gift Card Number.
func (e *Exchange) VerifyBinanceGiftCardNumber(ctx context.Context, referenceNumber string) (*GiftCardVerificationResponse, error) {
	if referenceNumber == "" {
		return nil, errReferenceNumberRequired
	}
	params := url.Values{}
	params.Set("referenceNo", referenceNumber)
	var resp *GiftCardVerificationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/giftcard/verify", params, sapiDefaultRate, nil, &resp)
}

// FetchRSAPublicKey this API is for fetching the RSA Public Key.
// This RSA Public key will be used to encrypt the card code.
func (e *Exchange) FetchRSAPublicKey(ctx context.Context) (*RSAPublicKeyResponse, error) {
	var resp *RSAPublicKeyResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/giftcard/cryptography/rsa-public-key", nil, sapiDefaultRate, nil, &resp)
}

// FetchTokenLimit this API is to help you verify which tokens are available for
// you to create Stablecoin-Denominated gift cards as mentioned in section 2 and its’ limitation.
func (e *Exchange) FetchTokenLimit(ctx context.Context, baseToken currency.Code) (*TokenLimitInfo, error) {
	if baseToken.IsEmpty() {
		return nil, fmt.Errorf("%w: baseToken is empty", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("baseToken", baseToken.String())
	var resp *TokenLimitInfo
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/giftcard/buyCode/token-limit", params, sapiDefaultRate, nil, &resp)
}

// ------------------------------------------------- User Data Stream Endpoints -----------------------------------------------

// CreateSpotListenKey start a new user data stream. The stream will close after 60 minutes unless a keepalive is sent.
// If the account has an active listenKey, that listenKey will be returned and its validity will be extended for 60 minutes.
func (e *Exchange) CreateSpotListenKey(ctx context.Context) (*ListenKeyResponse, error) {
	var resp *ListenKeyResponse
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/api/v3/userDataStream", listenKeyRate, &resp)
}

// CreateMarginListenKey start a margin new user data stream.
func (e *Exchange) CreateMarginListenKey(ctx context.Context) (*ListenKeyResponse, error) {
	var resp *ListenKeyResponse
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/userDataStream", listenKeyRate, &resp)
}

// KeepSpotListenKeyAlive keepalive a user data stream to prevent a time out. User data streams will close after 60 minutes. It's recommended to send a ping about every 30 minutes.
func (e *Exchange) KeepSpotListenKeyAlive(ctx context.Context, listenKey string) error {
	if listenKey == "" {
		return errListenKeyIsRequired
	}
	params := url.Values{}
	params.Set("listenKey", listenKey)
	return e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpot, http.MethodPut, common.EncodeURLValues("/api/v3/userDataStream", params), listenKeyRate, &struct{}{})
}

// KeepMarginListenKeyAlive Keep-alive a margin ListenKey
func (e *Exchange) KeepMarginListenKeyAlive(ctx context.Context, listenKey string) error {
	if listenKey == "" {
		return errListenKeyIsRequired
	}
	params := url.Values{}
	params.Set("listenKey", listenKey)
	return e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpot, http.MethodPut, common.EncodeURLValues("/sapi/v1/userDataStream", params), sapiDefaultRate, &struct{}{})
}

// CloseSpotListenKey close out a user data stream.
func (e *Exchange) CloseSpotListenKey(ctx context.Context, listenKey string) error {
	if listenKey == "" {
		return errListenKeyIsRequired
	}
	params := url.Values{}
	params.Set("listenKey", listenKey)
	return e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, common.EncodeURLValues("/api/v3/userDataStream", params), listenKeyRate, &struct{}{})
}

// CloseMarginListenKey closes a margin account listen key
func (e *Exchange) CloseMarginListenKey(ctx context.Context, listenKey string) error {
	if listenKey == "" {
		return errListenKeyIsRequired
	}
	params := url.Values{}
	params.Set("listenKey", listenKey)
	return e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, common.EncodeURLValues("/sapi/v1/userDataStream", params), sapiDefaultRate, &struct{}{})
}

// CreateCrossMarginListenKey start a cross-margin new user data stream.
func (e *Exchange) CreateCrossMarginListenKey(ctx context.Context, symbol string) (*ListenKeyResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *ListenKeyResponse
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, common.EncodeURLValues("/sapi/v1/userDataStream/isolated", params), listenKeyRate, &resp)
}

// KeepCrossMarginListenKeyAlive keepalive a user data stream to prevent a time out. User data streams will close after 60 minutes. It's recommended to send a ping about every 30 minutes.
func (e *Exchange) KeepCrossMarginListenKeyAlive(ctx context.Context, symbol, listenKey string) error {
	if listenKey == "" {
		return errListenKeyIsRequired
	}
	if symbol == "" {
		return currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("listenKey", listenKey)
	params.Set("symbol", symbol)
	return e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpot, http.MethodPut, common.EncodeURLValues("/sapi/v1/userDataStream/isolated", params), listenKeyRate, &struct{}{})
}

// CloseCrossMarginListenKey closed a cross-margin listen key
func (e *Exchange) CloseCrossMarginListenKey(ctx context.Context, symbol, listenKey string) error {
	if listenKey == "" {
		return errListenKeyIsRequired
	}
	if symbol == "" {
		return currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("listenKey", listenKey)
	params.Set("symbol", symbol)
	return e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, common.EncodeURLValues("/sapi/v1/userDataStream/isolated", params), sapiDefaultRate, &struct{}{})
}

// WithdrawalHistoryV1 fetch withdraw history for local entities that required travel rule through the V1 API.
// for local entities that require travel rule
func (e *Exchange) WithdrawalHistoryV1(ctx context.Context, travelRuleRecordIDs, transactionIDs, withdrawalOrderIDs []string, network, travelRuleStatus string, offset, limit int64, startTime, endTime time.Time) ([]LocalEntityWithdrawalDetail, error) {
	return e.withdrawalHistory(ctx, travelRuleRecordIDs, transactionIDs, withdrawalOrderIDs, network, travelRuleStatus, "/sapi/v1/localentity/withdraw/history", offset, limit, startTime, endTime)
}

// WithdrawalHistoryV2 fetch withdraw history for local entities that required travel rule through the V2 API.
func (e *Exchange) WithdrawalHistoryV2(ctx context.Context, travelRuleRecordIDs, transactionIDs, withdrawalOrderIDs []string, network, travelRuleStatus string, offset, limit int64, startTime, endTime time.Time) ([]LocalEntityWithdrawalDetail, error) {
	return e.withdrawalHistory(ctx, travelRuleRecordIDs, transactionIDs, withdrawalOrderIDs, network, travelRuleStatus, "/sapi/v2/localentity/withdraw/history", offset, limit, startTime, endTime)
}

// GetOnboardedVASPList retrieves the onboarded virtual asset service provider(VASP) list for local entities that required travel rule.
func (e *Exchange) GetOnboardedVASPList(ctx context.Context) ([]VASPItemInfo, error) {
	var resp []VASPItemInfo
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/localentity/vasp", nil, request.Auth, nil, &resp)
}

func (e *Exchange) withdrawalHistory(ctx context.Context, travelRuleRecordIDs, transactionIDs, withdrawalOrderIDs []string, network, travelRuleStatus, path string, offset, limit int64, startTime, endTime time.Time) ([]LocalEntityWithdrawalDetail, error) {
	params := url.Values{}
	if len(travelRuleRecordIDs) != 0 {
		params.Set("trId", strings.Join(travelRuleRecordIDs, ","))
	}
	if len(transactionIDs) != 0 {
		params.Set("txId", strings.Join(transactionIDs, ","))
	}
	if len(withdrawalOrderIDs) != 0 {
		params.Set("withdrawOrderId", strings.Join(withdrawalOrderIDs, ","))
	}
	if network != "" {
		params.Set("network", network)
	}
	if travelRuleStatus != "" {
		params.Set("travelRuleStatus", travelRuleStatus)
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []LocalEntityWithdrawalDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, request.Auth, nil, &resp)
}

// SubmitDepositQuestionnaire submit questionnaire for local entities that require travel rule.
// The questionnaire is only applies to transactions from unhosted wallets or VASPs that are not yet onboarded with GTR.
// details of key and values of questionnaire for each country can be found here: https://developers.binance.com/docs/wallet/travel-rule/withdraw-questionnaire
func (e *Exchange) SubmitDepositQuestionnaire(ctx context.Context, walletTransactionID string, questionnaire any) (*QuestionnaireDepositResponse, error) {
	if walletTransactionID == "" {
		return nil, fmt.Errorf("%w: WalletTransactionID is required", errTransactionIDRequired)
	}
	if questionnaire == "" {
		return nil, errQuestionnaireRequired
	}
	questionnaireJSON, err := json.Marshal(questionnaire)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("tranId", walletTransactionID)
	params.Set("questionnaire", string(questionnaireJSON))
	var resp *QuestionnaireDepositResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPut, "/sapi/v1/localentity/deposit/provide-info", params, request.Auth, nil, &resp)
}

// GetLocalEntitiesDepositHistory fetch deposit history for local entities that required travel rule.
func (e *Exchange) GetLocalEntitiesDepositHistory(ctx context.Context, travelRuleRecordIDs, transactionIDs, walletTransactionIDs []string, network string, coin currency.Code, travelRuleStatus string, pendingQuestionnaire bool, startTime, endTime time.Time, offset, limit int64) ([]LocalEntityDepositDetail, error) {
	params := url.Values{}
	if len(travelRuleRecordIDs) != 0 {
		params.Set("trId", strings.Join(travelRuleRecordIDs, ","))
	}
	if len(transactionIDs) != 0 {
		params.Set("txId", strings.Join(transactionIDs, ","))
	}
	if len(walletTransactionIDs) != 0 {
		params.Set("withdrawOrderId", strings.Join(walletTransactionIDs, ","))
	}
	if network != "" {
		params.Set("network", network)
	}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	if travelRuleStatus != "" {
		params.Set("travelRuleStatus", travelRuleStatus)
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if pendingQuestionnaire {
		params.Set("pendingQuestionnaire", "true")
	}
	var resp []LocalEntityDepositDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/localentity/deposit/history", params, request.Auth, nil, &resp)
}

// --------------------------------- Sub Account endpoints --------------------------------

// CreateSubAccount creates a link sub-account
func (e *Exchange) CreateSubAccount(ctx context.Context, tag string) (*CreatesSubAccount, error) {
	params := url.Values{}
	if tag != "" {
		params.Set("tag", tag)
	}
	var resp *CreatesSubAccount
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/broker/subAccount", nil, createSubAccountRate, nil, &resp)
}

// GetSubAccounts retrieves sub-accounts of the given account
func (e *Exchange) GetSubAccounts(ctx context.Context, subAccountID string, page, size int64) ([]SubAccountInstance, error) {
	params := url.Values{}
	if subAccountID != "" {
		params.Set("subAccountId", subAccountID)
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	var resp []SubAccountInstance
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/broker/subAccount", params, getSubAccountRate, nil, &resp)
}

// EnableFuturesForSubAccount enabled futures for sub-account
func (e *Exchange) EnableFuturesForSubAccount(ctx context.Context, subAccountID string, futures bool) (*FuturesSubAccountEnableResponse, error) {
	if subAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	params := url.Values{}
	params.Set("subAccountId", subAccountID)
	if futures {
		params.Set("futures", "true")
	}
	var resp *FuturesSubAccountEnableResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/broker/subAccount/futures", params, enableFuturesForSubAccountRate, nil, &resp)
}

// CreateAPIKeyForSubAccount creates a new API key for the specified subaccount
func (e *Exchange) CreateAPIKeyForSubAccount(ctx context.Context, subAccountID string, canTrade, marginTrade, futuresTrade bool) (*SubAccountAPIKey, error) {
	if subAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	params := url.Values{}
	if canTrade {
		params.Set("canTrade", "true")
	} else {
		params.Set("canTrade", "false")
	}
	if marginTrade {
		params.Set("marginTrade", "true")
	}
	if futuresTrade {
		params.Set("futuresTrade", "true")
	}
	var resp *SubAccountAPIKey
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/broker/subAccountApi", params, createAPIKeyForSubAccountRate, nil, &resp)
}

// ChangeSubAccountAPIPermission changes sub-account's api permission
func (e *Exchange) ChangeSubAccountAPIPermission(ctx context.Context, subAccountID, subAccountAPIKey string, canTrade, marginTrade, futuresTrade bool) (*SubAccountAPIKey, error) {
	if subAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	if subAccountAPIKey == "" {
		return nil, errEmptySubAccountAPIKey
	}
	params := url.Values{}
	params.Set("subAccountId", subAccountID)
	params.Set("subAccountApiKey", subAccountAPIKey)
	if canTrade {
		params.Set("canTrade", "true")
	}
	if marginTrade {
		params.Set("marginTrade", "true")
	}
	if futuresTrade {
		params.Set("futuresTrade", "true")
	}
	var resp *SubAccountAPIKey
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/broker/subAccountApi/permission", params, request.Auth, nil, &resp)
}

// EnableUniversalTransferPermissionForSubAccountAPIKey enables universal transfer permission for subaccount API key
func (e *Exchange) EnableUniversalTransferPermissionForSubAccountAPIKey(ctx context.Context, subAccountID, subAccountAPIKey string, canUniversalTransfer bool) (*SubAccountUniversalTransferEnableResponse, error) {
	if subAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	if subAccountAPIKey == "" {
		return nil, errEmptySubAccountAPIKey
	}
	params := url.Values{}
	params.Set("subAccountId", subAccountID)
	params.Set("subAccountApiKey", subAccountAPIKey)
	if canUniversalTransfer {
		params.Set("canUniversalTransfer", "true")
	} else {
		params.Set("canUniversalTransfer", "false")
	}
	var resp *SubAccountUniversalTransferEnableResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/broker/subAccountApi/permission/universalTransfer", params, request.Auth, nil, &resp)
}

// UpdateIPRestrictionForSubAccountAPIKey updates IP restriction for sub-account api key
func (e *Exchange) UpdateIPRestrictionForSubAccountAPIKey(ctx context.Context, subAccountID, subAccountAPIKey, status, ipAddress string) (*SubAccountIPRestrictioin, error) {
	if subAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	if subAccountAPIKey == "" {
		return nil, errEmptySubAccountAPIKey
	}
	if status == "" {
		return nil, errSubAccountStatusMissing
	}
	params := url.Values{}
	params.Set("subAccountId", subAccountID)
	params.Set("subAccountApiKey", subAccountAPIKey)
	params.Set("status", status)
	if ipAddress != "" {
		params.Set("ipAddress", ipAddress)
	}
	var resp *SubAccountIPRestrictioin
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v2/broker/subAccountApi/ipRestriction", params, request.Auth, nil, &resp)
}

// DeleteIPRestrictionForSubAccountAPIKey deletes an IP restriction for sub account api key
func (e *Exchange) DeleteIPRestrictionForSubAccountAPIKey(ctx context.Context, subAccountID, subAccountAPIKey, ipAddress string) (*SubAccountIPRestrictioin, error) {
	if subAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	if subAccountAPIKey == "" {
		return nil, errEmptySubAccountAPIKey
	}
	params := url.Values{}
	params.Set("subAccountId", subAccountID)
	params.Set("subAccountApiKey", subAccountAPIKey)
	if ipAddress != "" {
		params.Set("ipAddress", ipAddress)
	}
	var resp *SubAccountIPRestrictioin
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/broker/subAccountApi/ipRestriction/ipList", nil, request.Auth, nil, &resp)
}

// DeleteSubAccountAPIKey delete sub account api key
func (e *Exchange) DeleteSubAccountAPIKey(ctx context.Context, subAccountID, subAccountAPIKey string) (interface{}, error) {
	if subAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	if subAccountAPIKey == "" {
		return nil, errEmptySubAccountAPIKey
	}
	params := url.Values{}
	params.Set("subAccountId", subAccountID)
	params.Set("subAccountApiKey", subAccountAPIKey)
	var resp interface{}
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/broker/subAccountApi", params, request.Auth, nil, &resp)
}

// LinkAccountInformation link account information
func (e *Exchange) LinkAccountInformation(ctx context.Context) (*LinkAccountInformation, error) {
	var resp *LinkAccountInformation
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/broker/info", nil, request.Auth, nil, &resp)
}

// EnableOrDisableBNBBurnForSubAccountSpotAndMargin enables or disables BNB burn for spot and margin subaccounts
func (e *Exchange) EnableOrDisableBNBBurnForSubAccountSpotAndMargin(ctx context.Context, subAccountID string, spotBNBBurn bool) (*BNBBurnToggleSpot, error) {
	params := url.Values{}
	if subAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	params.Set("subAccountId", subAccountID)
	if spotBNBBurn {
		params.Set("spotBNBBurn", "true")
	} else {
		params.Set("spotBNBBurn", "false")
	}
	var resp *BNBBurnToggleSpot
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/broker/subAccount/bnbBurn/spot", params, request.Auth, nil, &resp)
}

// EnableOrDisableBNBBurnForSubAccountMarginInterest toggles to enable or disable BNB burn for sub-account margin interest
// subaccount must be enabled margin before using this switch
func (e *Exchange) EnableOrDisableBNBBurnForSubAccountMarginInterest(ctx context.Context, subAccountID string, interestBNBburn bool) (*BNBBurnToggleResponse, error) {
	if subAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	params := url.Values{}
	params.Set("subAccountId", subAccountID)
	if interestBNBburn {
		params.Set("interestBNBBurn", "true")
	} else {
		params.Set("interestBNBBurn", "false")
	}
	var resp *BNBBurnToggleResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/broker/subAccount/bnbBurn/marginInterest", params, request.Auth, nil, &resp)
}

// GetBNBBurnStatusForSubAccount retrieves BNB burn status for sub-account
func (e *Exchange) GetBNBBurnStatusForSubAccount(ctx context.Context, subAccountID string) (*BNBBurnStatus, error) {
	if subAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	params := url.Values{}
	params.Set("subAccountId", subAccountID)
	var resp *BNBBurnStatus
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/broker/subAccount/bnbBurn/status", params, request.Auth, nil, &resp)
}

// SubAccountTransferWithSpotBroker applies a subaccount transfer through a broker on spot account
func (e *Exchange) SubAccountTransferWithSpotBroker(ctx context.Context, ccy currency.Code, fromID, toID, clientTransferID string, amount float64) (*BrokerSubAccountTransfer, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("asset", ccy.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	if fromID != "" {
		params.Set("fromId", fromID)
	}
	if toID != "" {
		params.Set("toId", toID)
	}
	if clientTransferID != "" {
		params.Set("clientTranId", clientTransferID)
	}
	var resp *BrokerSubAccountTransfer
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/broker/transfer", params, request.Auth, nil, &resp)
}

// GetSpotBrokerSubAccountTransferHistory retrieves sub-account assets transfers of spot account through a broker account
func (e *Exchange) GetSpotBrokerSubAccountTransferHistory(ctx context.Context, fromID, toID, clientTransferID string, showAllStatus bool, startTime, endTime time.Time, page, limit int64) ([]SubAccountTransferRecord, error) {
	params := url.Values{}
	if fromID != "" {
		params.Set("fromId", fromID)
	}
	if toID != "" {
		params.Set("toId", toID)
	}
	if clientTransferID != "" {
		params.Set("clientTranId", clientTransferID)
	}
	if showAllStatus {
		params.Set("showAllStatus", "true")
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	var resp []SubAccountTransferRecord
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/broker/transfer", params, request.Auth, nil, &resp)
}

// SubAccountTransferWithFuturesBroker applies a subaccount transfer through a broker on futures account
func (e *Exchange) SubAccountTransferWithFuturesBroker(ctx context.Context, ccy currency.Code, fromID, toID, clientTransferID string, futuresType int, amount float64) (*BrokerSubAccountTransfer, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("asset", ccy.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	if futuresType != 0 {
		params.Set("futuresType", strconv.Itoa(futuresType))
	}
	if fromID != "" {
		params.Set("fromId", fromID)
	}
	if toID != "" {
		params.Set("toId", toID)
	}
	if clientTransferID != "" {
		params.Set("clientTranId", clientTransferID)
	}
	var resp *BrokerSubAccountTransfer
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/broker/transfer/futures", params, request.Auth, nil, &resp)
}

// GetFuturesBrokerSubAccountTransferHistory retrieves sub-account assets transfers of futures account through a broker account
func (e *Exchange) GetFuturesBrokerSubAccountTransferHistory(ctx context.Context, coinMargined bool, subAccountID, clientTransferID string, startTime, endTime time.Time, page, limit int64) (*FuturesSubAccountTransfers, error) {
	if subAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	params := url.Values{}
	if coinMargined {
		params.Set("futuresType", "2")
	} else {
		params.Set("futuresType", "1")
	}
	if clientTransferID != "" {
		params.Set("clientTranId", clientTransferID)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	var resp *FuturesSubAccountTransfers
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/broker/transfer/futures", params, request.Auth, nil, &resp)
}

// GetSubAccountDepositHistoryWithBroker holds a sub-account deposit history through broker
func (e *Exchange) GetSubAccountDepositHistoryWithBroker(ctx context.Context, subAccountID string, coin currency.Code, startTime, endTime time.Time, status, limit, offset int64) ([]SubAccountTransferWithBroker, error) {
	params := url.Values{}
	if subAccountID != "" {
		params.Set("subAccountId", subAccountID)
	}
	if !coin.IsEmpty() {
		params.Set("coin", coin.String())
	}
	if status >= 0 {
		params.Set("status", strconv.FormatInt(status, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	var resp []SubAccountTransferWithBroker
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/broker/subAccount/depositHist", params, request.Auth, nil, &resp)
}

// GetSubAccountSpotAssetInfo retrieves a spot accounts sub-account asset information
func (e *Exchange) GetSubAccountSpotAssetInfo(ctx context.Context, subAccountID string, page, size int64) (*SpotSubAccountAssetInfo, error) {
	params := url.Values{}
	if subAccountID != "" {
		params.Set("subAccountId", subAccountID)
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	var resp *SpotSubAccountAssetInfo
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/broker/subAccount/spotSummary", params, request.Auth, nil, &resp)
}

// GetSubAccountMarginAssetInfo retrieves a margin account sub-account asset information
func (e *Exchange) GetSubAccountMarginAssetInfo(ctx context.Context, subAccountID string, page, size int64) (*MarginSubAccountAssetInfo, error) {
	params := url.Values{}
	if subAccountID != "" {
		params.Set("subAccountId", subAccountID)
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	var resp *MarginSubAccountAssetInfo
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/broker/subAccount/marginSummary", params, request.Auth, nil, &resp)
}

// GetSubAccountFuturesAssetInfo holds a futures sub-account asset information
func (e *Exchange) GetSubAccountFuturesAssetInfo(ctx context.Context, subAccountID string, coinMargined bool, page, size int64) (*FuturesSubAccountAssetInfo, error) {
	params := url.Values{}
	if subAccountID != "" {
		params.Set("subAccountId", subAccountID)
	}
	if coinMargined {
		params.Set("futuresType", "2")
	} else {
		params.Set("futuresType", "1")
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	var resp *FuturesSubAccountAssetInfo
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v3/broker/subAccount/futuresSummary", params, request.Auth, nil, &resp)
}

// UniversalTransferWithBroker retrieves a universal transfer history with broker
func (e *Exchange) UniversalTransferWithBroker(ctx context.Context, fromAccountType, toAccountType, fromID, toID, clientTransferID string, ccy currency.Code, amount float64) (*UniversalTransferResponse, error) {
	if fromAccountType == "" {
		return nil, fmt.Errorf("%w: fromAccountType=%s", errInvalidAccountType, fromAccountType)
	}
	if toAccountType == "" {
		return nil, fmt.Errorf("%w: toAccountType=%s", errInvalidAccountType, toAccountType)
	}
	if ccy.IsEmpty() {
		return nil, fmt.Errorf("%w: asset is required", currency.ErrCurrencyCodeEmpty)
	}
	if amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("fromAccountType", fromAccountType)
	params.Set("toAccountType", toAccountType)
	params.Set("asset", ccy.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	if fromID != "" {
		params.Set("fromId", fromID)
	}
	if toID != "" {
		params.Set("toId", toID)
	}
	if clientTransferID != "" {
		params.Set("clientTranId", clientTransferID)
	}
	var resp *UniversalTransferResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/broker/universalTransfer", params, request.Auth, nil, &resp)
}

// GetUniversalTransferHistoryThroughBroker retrieves a universal asset transfer history thought broker
func (e *Exchange) GetUniversalTransferHistoryThroughBroker(ctx context.Context, fromID, toID, clientTransferID string, startTime, endTime time.Time, page, limit int64) ([]AssetUniversalTransferDetail, error) {
	params := url.Values{}
	if fromID != "" {
		params.Set("fromId", fromID)
	}
	if toID != "" {
		params.Set("toId", toID)
	}
	if clientTransferID != "" {
		params.Set("clientTranId", clientTransferID)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []AssetUniversalTransferDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/broker/universalTransfer", params, request.Auth, nil, &resp)
}

// CreateBrokerSubAccount creates a new sub-account through broker
func (e *Exchange) CreateBrokerSubAccount(ctx context.Context, tag string) (*BrokerCreateSubAccount, error) {
	params := url.Values{}
	if tag != "" {
		params.Set("tag", tag)
	}
	var resp *BrokerCreateSubAccount
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/broker/subAccount", params, request.Auth, nil, &resp)
}

// GetBrokerSubAccounts retrieves sub-accounts of a user created by broker
func (e *Exchange) GetBrokerSubAccounts(ctx context.Context, subAccountID string, page, size int64) ([]BrokerCreatedSubAccountDetail, error) {
	params := url.Values{}
	if subAccountID != "" {
		params.Set("subAccountId", subAccountID)
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	var resp []BrokerCreatedSubAccountDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/broker/subAccount", params, request.Auth, nil, &resp)
}

// ChangeSubAccountCommission changes subaccount commission
func (e *Exchange) ChangeSubAccountCommission(ctx context.Context, subAccountID string, makerCommission, takerCommission, marginMakerCommission, marginTakerCommission float64) (*SubAccountCommission, error) {
	if subAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	if makerCommission <= 0 {
		return nil, fmt.Errorf("%w: makerCommission is required", errCommissionValueRequired)
	}
	if takerCommission <= 0 {
		return nil, fmt.Errorf("%w: takerCommission is required", errCommissionValueRequired)
	}
	params := url.Values{}
	params.Set("makerCommission", strconv.FormatFloat(makerCommission, 'f', -1, 64))
	params.Set("takerCommission", strconv.FormatFloat(takerCommission, 'f', -1, 64))
	if marginMakerCommission <= 0 {
		params.Set("marginMakerCommission", strconv.FormatFloat(marginMakerCommission, 'f', -1, 64))
	}
	if marginTakerCommission <= 0 {
		params.Set("marginTakerCommission", strconv.FormatFloat(marginTakerCommission, 'f', -1, 64))
	}
	var resp *SubAccountCommission
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/broker/subAccountApi/commission", params, request.Auth, nil, &resp)
}

// ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment changes subaccount USDT margined futures commission adjustment
func (e *Exchange) ChangeSubAccountUSDTMarginedFuturesCommissionAdjustment(ctx context.Context, subAccountID, symbol string, makerAdjustment, takerAdjustment int64) (*SubAccountFuturesUSDMarginedCommissionAdjustment, error) {
	if subAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if makerAdjustment <= 0 {
		return nil, fmt.Errorf("%w: makerAdjustment", limits.ErrAmountBelowMin)
	}
	if takerAdjustment <= 0 {
		return nil, fmt.Errorf("%w: takerAdjustment", limits.ErrAmountBelowMin)
	}
	params := url.Values{}
	params.Set("subAccountId", subAccountID)
	params.Set("symbol", symbol)
	params.Set("makerAdjustment", strconv.FormatInt(makerAdjustment, 10))
	params.Set("takerAdjustment", strconv.FormatInt(takerAdjustment, 10))
	var resp *SubAccountFuturesUSDMarginedCommissionAdjustment
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/broker/subAccountApi/commission/futures", params, request.Auth, nil, &resp)
}

// GetSubAccountUSDMarginedFuturesCommissionAdjustment retrieves subaccount USDT margined futures commission adjustment
func (e *Exchange) GetSubAccountUSDMarginedFuturesCommissionAdjustment(ctx context.Context, subAccountID, symbol string) ([]FuturesSubAccountCommissionAdjustments, error) {
	if subAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	params := url.Values{}
	params.Set("subAccountId", subAccountID)
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []FuturesSubAccountCommissionAdjustments
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/broker/subAccountApi/commission/futures", params, request.Auth, nil, &resp)
}

// ChangeSubAccountCoinMarginedFuturesCommissionAdjustment changes subaccount coind margined futures commission adjustments
func (e *Exchange) ChangeSubAccountCoinMarginedFuturesCommissionAdjustment(ctx context.Context, subAccountID, symbol string, makerAdjustment, takerAdjustment int64) ([]FSubAccountCommissionAdjustment, error) {
	if subAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if makerAdjustment <= 0 {
		return nil, fmt.Errorf("%w: makerAdjustment is required", limits.ErrAmountBelowMin)
	}
	if takerAdjustment <= 0 {
		return nil, fmt.Errorf("%w: takerAdjustment is required", limits.ErrAmountBelowMin)
	}
	params := url.Values{}
	params.Set("subAccountId", subAccountID)
	params.Set("pair", symbol)
	params.Set("makerAdjustment", strconv.FormatInt(makerAdjustment, 10))
	params.Set("takerAdjustment", strconv.FormatInt(takerAdjustment, 10))
	var resp []FSubAccountCommissionAdjustment
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/broker/subAccountApi/commission/coinFutures", params, request.Auth, nil, &resp)
}

// GetSubAccountCoinMarginedFuturesCommissionAdjustment sub-account's COIN-Ⓜ futures commission of a symbol equals to the base commission of the symbol on the sub-account's fee tier plus the commission adjustment.
func (e *Exchange) GetSubAccountCoinMarginedFuturesCommissionAdjustment(ctx context.Context, subAccountID, symbol string) ([]FSubAccountCommissionAdjustment, error) {
	if subAccountID == "" {
		return nil, errSubAccountIDMissing
	}
	params := url.Values{}
	params.Set("subAccountId", subAccountID)
	if symbol != "" {
		params.Set("pair", symbol)
	}
	var resp []FSubAccountCommissionAdjustment
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/broker/subAccountApi/commission/coinFutures", params, request.Auth, nil, &resp)
}

// GetSpotBrokerCommissionRebateRecentRecord retrieves broker commission rebate recent records spot
func (e *Exchange) GetSpotBrokerCommissionRebateRecentRecord(ctx context.Context, subAccountID string, startTime, endTime time.Time, page, limit int64) ([]CommissionRebateRecord, error) {
	params := url.Values{}
	if subAccountID != "" {
		params.Set("subAccountId", subAccountID)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []CommissionRebateRecord
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/broker/rebate/recentRecord", params, request.Auth, nil, &resp)
}

// GetFuturesBrokerCommissionRebateRecentRecord retrieves a broker futures commission rebate record
func (e *Exchange) GetFuturesBrokerCommissionRebateRecentRecord(ctx context.Context, coinMargined, filterResult bool, startTime, endTime time.Time, page, size int64) ([]CommissionRebateRecord, error) {
	params := url.Values{}
	if coinMargined {
		params.Set("futuresType", "2")
	} else {
		params.Set("futuresType", "1")
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if filterResult {
		params.Set("filterResult", "true")
	}
	var resp []CommissionRebateRecord
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/broker/rebate/futures/recentRecord", params, request.Auth, nil, &resp)
}

// ---------------------------------------------------------------- Binance Link endpoints ----------------------------------------------------------------

// GetSpotInfoAboutIfUserIsNew retrieves client details including information about whether the client is new or not
func (e *Exchange) GetSpotInfoAboutIfUserIsNew(ctx context.Context, apiAgentCode string) (*UserIsNewUserDetail, error) {
	if apiAgentCode == "" {
		return nil, fmt.Errorf("%w: apiAgentCode is required", errCodeRequired)
	}
	params := url.Values{}
	params.Set("apiAgentCode", apiAgentCode)
	var resp *UserIsNewUserDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/apiReferral/ifNewUser", params, request.Auth, nil, &resp)
}

// CustomizeSpotPartnerClientID customizes partner's customer ID by user email
func (e *Exchange) CustomizeSpotPartnerClientID(ctx context.Context, customerID, email string) (*CustomerIDResult, error) {
	return e.customizePartnerClientID(ctx, customerID, email, "/sapi/v1/apiReferral/customization")
}

func (e *Exchange) customizePartnerClientID(ctx context.Context, customerID, email, path string) (*CustomerIDResult, error) {
	if customerID == "" {
		return nil, fmt.Errorf("%w: customerID required", order.ErrOrderIDNotSet)
	}
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	params.Set("customerId", customerID)
	var resp *CustomerIDResult
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, params, request.Auth, nil, &resp)
}

// GetSpotClientEmailCustomizedID retrieves client email customized ID details
func (e *Exchange) GetSpotClientEmailCustomizedID(ctx context.Context, customerID, email string) ([]CustomerIDResult, error) {
	return e.getClientEmailCustomizedID(ctx, customerID, email, "/sapi/v1/apiReferral/customization")
}

func (e *Exchange) getClientEmailCustomizedID(ctx context.Context, customerID, email, path string) ([]CustomerIDResult, error) {
	params := url.Values{}
	if customerID != "" {
		params.Set("customerId", customerID)
	}
	if common.MatchesEmailPattern(email) {
		params.Set("email", email)
	}
	var resp []CustomerIDResult
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, request.Auth, nil, &resp)
}

// CustomizeSpotOwnClientID customize your own customer ID by broker ID
func (e *Exchange) CustomizeSpotOwnClientID(ctx context.Context, customerID, apiAgentCode string) (*CustomerIDResult, error) {
	return e.customizeOwnClientID(ctx, customerID, apiAgentCode, "/sapi/v1/apiReferral/userCustomization")
}

func (e *Exchange) customizeOwnClientID(ctx context.Context, customerID, apiAgentCode, path string) (*CustomerIDResult, error) {
	if customerID == "" {
		return nil, fmt.Errorf("%w: customerID required", order.ErrOrderIDNotSet)
	}
	if apiAgentCode == "" {
		return nil, fmt.Errorf("%w: apiAgentCode is required", errCodeRequired)
	}
	params := url.Values{}
	params.Set("customerId", customerID)
	params.Set("apiAgentCode", apiAgentCode)
	var resp *CustomerIDResult
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, params, request.Auth, nil, &resp)
}

// GetSpotUsersCustomizedID retrieves user's customized ID
func (e *Exchange) GetSpotUsersCustomizedID(ctx context.Context, apiAgentCode string) (*CustomerIDResult, error) {
	if apiAgentCode == "" {
		return nil, fmt.Errorf("%w: apiAgentCode is required", errCodeRequired)
	}
	params := url.Values{}
	params.Set("apiAgentCode", apiAgentCode)
	var resp *CustomerIDResult
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/apiReferral/userCustomization", params, request.Auth, nil, &resp)
}

// GetSpotOthersRebateRecentRecord retrieves a list of recent rabate records by customer ID
func (e *Exchange) GetSpotOthersRebateRecentRecord(ctx context.Context, customerID string, startTime, endTime time.Time, limit int64) ([]CustomerRabateRecord, error) {
	if customerID == "" {
		return nil, fmt.Errorf("%w: customerID required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("customerId", customerID)
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []CustomerRabateRecord
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/apiReferral/rebate/recentRecord", params, request.Auth, nil, &resp)
}

// GetSpotOwnRebateRecentRecords retrieves own recent rebate records
func (e *Exchange) GetSpotOwnRebateRecentRecords(ctx context.Context, startTime, endTime time.Time, limit int64) ([]CustomerRabateRecord, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []CustomerRabateRecord
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/apiReferral/kickback/recentRecord", params, request.Auth, nil, &resp)
}

// GetFuturesClientIfNewUser retrieves client if the new user is margin type(mType) value of 1:USDT-margined Futures，2: Coin-margined Futures
func (e *Exchange) GetFuturesClientIfNewUser(ctx context.Context, brokerID string, mType int) (*FuturesNewUserDetail, error) {
	if brokerID == "" {
		return nil, fmt.Errorf("%w: brokerID is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("brokerId", brokerID)
	if mType != 0 {
		params.Set("type", strconv.Itoa(mType))
	}
	var resp *FuturesNewUserDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/fapi/v1/apiReferral/ifNewUser", params, request.Auth, nil, &resp)
}

// CustomizeFuturesPartnerClientID customizes partner's customer ID by user email
func (e *Exchange) CustomizeFuturesPartnerClientID(ctx context.Context, customerID, email string) (*CustomerIDResult, error) {
	return e.customizePartnerClientID(ctx, customerID, email, "/fapi/v1/apiReferral/customization")
}

// GetFuturesClientEmailCustomizedID retrieves client email customized ID details for futures account
func (e *Exchange) GetFuturesClientEmailCustomizedID(ctx context.Context, customerID, email string) ([]CustomerIDResult, error) {
	return e.getClientEmailCustomizedID(ctx, customerID, email, "/fapi/v1/apiReferral/customization")
}

// CustomizeFuturesOwnClientID customize your own customer ID by broker ID for futures account
func (e *Exchange) CustomizeFuturesOwnClientID(ctx context.Context, customerID, apiAgentCode string) (*CustomerIDResult, error) {
	return e.customizeOwnClientID(ctx, customerID, apiAgentCode, "/fapi/v1/apiReferral/userCustomization")
}

// GetFuturesUsersCustomizedID retrieves user's customized ID for futures account
func (e *Exchange) GetFuturesUsersCustomizedID(ctx context.Context, brokerID string) (*FuturesCustomerID, error) {
	if brokerID == "" {
		return nil, fmt.Errorf("%w: brokerId is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("brokerId", brokerID)
	var resp *FuturesCustomerID
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/fapi/v1/apiReferral/userCustomization", params, request.Auth, nil, &resp)
}

// GetFuturesUserIncomeHistory retrieves a futures user's income history
func (e *Exchange) GetFuturesUserIncomeHistory(ctx context.Context, symbol, incomeType string, startTime, endTime time.Time, limit int64) (interface{}, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if incomeType != "" {
		params.Set("incomeType", incomeType)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []FuturesUserIncomeDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/fapi/v1/income", params, request.Auth, nil, &resp)
}

// GetFuturesReferredTradersNumber retrieve the number of new and existing traders associated with their referral program over specific time periods.
// coinMargined is true if the type coin margined futures and false if it is usdt margined futures
func (e *Exchange) GetFuturesReferredTradersNumber(ctx context.Context, coinMargined bool, startTime, endTime time.Time, limit int64) ([]BrokerTradersNumber, error) {
	params := url.Values{}
	if coinMargined {
		params.Set("type", "2")
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []BrokerTradersNumber
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/fapi/v1/apiReferral/traderNum", params, request.Auth, nil, &resp)
}

// GetFuturesRebateDataOverview retrieves an overview of rebate data for brokers, including details like the number of new and existing traders referred, total trading volume, and total rebates earned.
func (e *Exchange) GetFuturesRebateDataOverview(ctx context.Context, coinMargined bool) (*RebateOverview, error) {
	params := url.Values{}
	if coinMargined {
		params.Set("type", "2")
	}
	var resp *RebateOverview
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/fapi/v1/apiReferral/overview", params, request.Auth, nil, &resp)
}

// GetUserTradeVolume retrieves user's trade volume at different timestamps
func (e *Exchange) GetUserTradeVolume(ctx context.Context, coinMargined bool, startTime, endTime time.Time, limit int64) ([]UserTradeVolume, error) {
	params := url.Values{}
	if coinMargined {
		params.Set("type", "2")
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []UserTradeVolume
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/fapi/v1/apiReferral/tradeVol", params, request.Auth, nil, &resp)
}

// GetRebateVolume retrieve rebate volume data for a user's futures trading account
func (e *Exchange) GetRebateVolume(ctx context.Context, coinMargined bool, startTime, endTime time.Time, limit int64) (interface{}, error) {
	params := url.Values{}
	if coinMargined {
		params.Set("type", "2")
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []UserRebateVolume
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/fapi/v1/apiReferral/rebateVol", params, request.Auth, nil, &resp)
}

// GetTraderDetail retrieves detailed trading and rebate volume data for referred traders under the Binance Futures Referral Program
func (e *Exchange) GetTraderDetail(ctx context.Context, customerID string, coinMargined bool, startTime, endTime time.Time, limit int64) ([]TradingAndRebateVolumeData, error) {
	params := url.Values{}
	if customerID != "" {
		params.Set("customerId", customerID)
	}
	if coinMargined {
		params.Set("type", "2")
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []TradingAndRebateVolumeData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/fapi/v1/apiReferral/traderSummary", params, request.Auth, nil, &resp)
}

// GetFuturesClientifNewUser retrieves futures client detail if new user
func (e *Exchange) GetFuturesClientifNewUser(ctx context.Context, brokerID string, coinMargined bool) (*FuturesClientIfNewUser, error) {
	if brokerID == "" {
		return nil, fmt.Errorf("%w: brokerId is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("brokerId", brokerID)
	if coinMargined {
		params.Set("type", "2")
	}
	var resp *FuturesClientIfNewUser
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/papi/v1/apiReferral/ifNewUser", params, request.Auth, nil, &resp)
}

// CustomizeIDForClientToReferredUser allows a broker (referrer) to assign a custom unique identifier (customerId) to a referred user.
func (e *Exchange) CustomizeIDForClientToReferredUser(ctx context.Context, customerID, brokerID string) (*BrokerAndCustomerID, error) {
	if customerID == "" {
		return nil, fmt.Errorf("%w: customerID is required", order.ErrOrderIDNotSet)
	}
	if brokerID == "" {
		return nil, fmt.Errorf("%w: brokerID is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("customerId", customerID)
	params.Set("brokerId", brokerID)
	var resp *BrokerAndCustomerID
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/papi/v1/apiReferral/userCustomization", params, request.Auth, nil, &resp)
}

// GetUsersCustomizeIDs retrieves user's customize ID
func (e *Exchange) GetUsersCustomizeIDs(ctx context.Context, brokerID string) (*BrokerAndCustomerID, error) {
	if brokerID == "" {
		return nil, fmt.Errorf("%w: brokerID is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("brokerId", brokerID)
	var resp *BrokerAndCustomerID
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/papi/v1/apiReferral/userCustomization", params, request.Auth, nil, &resp)
}

// GetFastAPIUserStatus retrieves whether user's futures account is new or existing
func (e *Exchange) GetFastAPIUserStatus(ctx context.Context) (*FuturesUserStatus, error) {
	var resp *FuturesUserStatus
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/v1/api-key/user-status", nil, request.Auth, nil, &resp)
}

// CreateAPIKey creates a new API key
func (e *Exchange) CreateAPIKey(ctx context.Context, apiName, publicKey, status, ipAddress, thirdPartyName string, enableTrade, enableFutureTrade, enableMargin, enableEuropeanOptions bool) (*UserAPIKeyCreationResponse, error) {
	if apiName == "" {
		return nil, errAPIKeyNameRequired
	}
	if publicKey == "" {
		return nil, fmt.Errorf("%w: publicKey is required", errEmptySubAccountAPIKey)
	}
	params := url.Values{}
	params.Set("apiName", apiName)
	params.Set("publicKey", publicKey)
	if enableTrade {
		params.Set("enableTrade", "true")
	} else {
		params.Set("enableTrade", "false")
	}
	if enableFutureTrade {
		params.Set("enableFutureTrade", "true")
	} else {
		params.Set("enableFutureTrade", "false")
	}
	if enableMargin {
		params.Set("enableMargin", "true")
	} else {
		params.Set("enableMargin", "false")
	}
	if enableEuropeanOptions {
		params.Set("enableEuropeanOptions", "true")
	} else {
		params.Set("enableEuropeanOptions", "false")
	}
	if status != "" {
		params.Set("status", status)
	}
	if ipAddress != "" {
		params.Set("ipAddress", ipAddress)
	}
	if thirdPartyName != "" {
		params.Set("thirdPartyName", thirdPartyName)
	}
	var resp *UserAPIKeyCreationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/v1/api-key/create", params, request.Auth, nil, &resp)
}
