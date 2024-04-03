package binance

import (
	"context"
	"encoding/json"
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
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Binance is the overarching type across the Binance package
type Binance struct {
	exchange.Base
	// Valid string list that is required by the exchange
	validLimits []int64
	obm         *orderbookManager

	// isAPIStreamConnected is true if the spot API stream websocket connection is established
	isAPIStreamConnected bool

	isAPIStreamConnectionLock sync.Mutex
}

const (
	apiURL         = "https://api.binance.com"
	spotAPIURL     = "https://sapi.binance.com"
	cfuturesAPIURL = "https://dapi.binance.com"
	ufuturesAPIURL = "https://fapi.binance.com"
	eOptionAPIURL  = "https://eapi.binance.com"
	pMarginAPIURL  = "https://papi.binance.com"

	testnetSpotURL = "https://testnet.binance.vision/api"
	testnetFutures = "https://testnet.binancefuture.com"

	defaultRecvWindow = 5 * time.Second
)

var (
	errLoanCoinMustBeSet                      = errors.New("loan coin must bet set")
	errLoanTermMustBeSet                      = errors.New("loan term must be set")
	errCollateralCoinMustBeSet                = errors.New("collateral coin must be set")
	errOrderIDMustBeSet                       = errors.New("orderID must be set")
	errAmountMustBeSet                        = errors.New("amount must not be <= 0")
	errEitherLoanOrCollateralAmountsMustBeSet = errors.New("either loan or collateral amounts must be set")
	errNilArgument                            = errors.New("nil argument")
	errTimestampInfoRequired                  = errors.New("timestamp information is required")
	errListenKeyIsRequired                    = errors.New("listen key is required")
	errValidEmailRequired                     = errors.New("valid email address is required")
	errPageNumberRequired                     = errors.New("page number is required")
	errLimitNumberRequired                    = errors.New("invalid limit")
	errEmptySubAccountEPIKey                  = errors.New("invalid sub-account API key")
	errInvalidFuturesType                     = errors.New("invalid futures types")
	errInvalidAccountType                     = errors.New("invalid account type specified")
)

var subscriptionNames = map[string]string{
	subscription.TickerChannel:    "ticker",
	subscription.OrderbookChannel: "depth",
	subscription.CandlesChannel:   "kline",
	subscription.AllTradesChannel: "trade",
}

// GetExchangeServerTime retrieves the server time.
func (b *Binance) GetExchangeServerTime(ctx context.Context) (time.Time, error) {
	resp := &struct {
		ServerTime convert.ExchangeTime `json:"serverTime"`
	}{}
	return resp.ServerTime.Time(), b.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, "/api/v3/time", spotExchangeInfo, resp)
}

// GetExchangeInfo returns exchange information. Check binance_types for more
// information
func (b *Binance) GetExchangeInfo(ctx context.Context) (*ExchangeInfo, error) {
	var resp *ExchangeInfo
	return resp, b.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary, "/api/v3/exchangeInfo", spotExchangeInfo, &resp)
}

// GetOrderBook returns full orderbook information
//
// OrderBookDataRequestParams contains the following members
// symbol: string of currency pair
// limit: returned limit amount
func (b *Binance) GetOrderBook(ctx context.Context, obd OrderBookDataRequestParams) (*OrderBook, error) {
	if err := b.CheckLimit(obd.Limit); err != nil {
		return nil, err
	}

	symbol, err := b.FormatSymbol(obd.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("limit", strconv.FormatInt(obd.Limit, 10))

	var resp OrderBookData
	if err := b.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		"/api/v3/depth?"+params.Encode(),
		orderbookLimit(obd.Limit), &resp); err != nil {
		return nil, err
	}

	orderbook := OrderBook{
		Bids:         make([]OrderbookItem, len(resp.Bids)),
		Asks:         make([]OrderbookItem, len(resp.Asks)),
		LastUpdateID: resp.LastUpdateID,
	}
	for x := range resp.Bids {
		orderbook.Bids[x] = OrderbookItem{
			Price:    resp.Bids[x][0].Float64(),
			Quantity: resp.Bids[x][1].Float64(),
		}
	}
	for x := range resp.Asks {
		orderbook.Asks[x] = OrderbookItem{
			Price:    resp.Asks[x][0].Float64(),
			Quantity: resp.Asks[x][1].Float64(),
		}
	}
	return &orderbook, nil
}

// GetMostRecentTrades returns recent trade activity
// limit: Up to 500 results returned
func (b *Binance) GetMostRecentTrades(ctx context.Context, rtr RecentTradeRequestParams) ([]RecentTrade, error) {
	symbol, err := b.FormatSymbol(rtr.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("limit", strconv.FormatInt(rtr.Limit, 10))
	var resp []RecentTrade
	return resp, b.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary, "/api/v3/trades?"+params.Encode(), spotDefaultRate, &resp)
}

// GetHistoricalTrades returns historical trade activity
//
// symbol: string of currency pair
// limit: Optional. Default 500; max 1000.
// fromID:
func (b *Binance) GetHistoricalTrades(ctx context.Context, symbol string, limit int, fromID int64) ([]HistoricalTrade, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("limit", strconv.Itoa(limit))
	// else return most recent trades
	if fromID > 0 {
		params.Set("fromId", strconv.FormatInt(fromID, 10))
	}
	var resp []HistoricalTrade
	return resp,
		b.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues("/api/v3/historicalTrades", params), spotDefaultRate, &resp)
}

// GetUserMarginInterestHistory returns margin interest history for the user
func (b *Binance) GetUserMarginInterestHistory(ctx context.Context, assetCurrency currency.Code, isolatedSymbol currency.Pair, startTime, endTime time.Time, currentPage, size int64, archived bool) (*UserMarginInterestHistoryResponse, error) {
	params := url.Values{}
	if !assetCurrency.IsEmpty() {
		params.Set("asset", assetCurrency.String())
	}
	if !isolatedSymbol.IsEmpty() {
		fPair, err := b.FormatSymbol(isolatedSymbol, asset.Margin)
		if err != nil {
			return nil, err
		}
		params.Set("isolatedSymbol", fPair)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if currentPage > 0 {
		params.Set("current", strconv.FormatInt(currentPage, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	if archived {
		params.Set("archived", "true")
	}

	var resp *UserMarginInterestHistoryResponse
	return resp, b.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues("/sapi/v1/margin/interestHistory", params), spotDefaultRate, &resp)
}

// GetAggregatedTrades returns aggregated trade activity.
// If more than one hour of data is requested or asked limit is not supported by exchange
// then the trades are collected with multiple backend requests.
// https://binance-docs.github.io/apidocs/spot/en/#compressed-aggregate-trades-list
func (b *Binance) GetAggregatedTrades(ctx context.Context, arg *AggregatedTradeRequestParams) ([]AggregatedTrade, error) {
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
	if !arg.StartTime.IsZero() {
		params.Set("startTime", timeString(arg.StartTime))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", timeString(arg.EndTime))
	}

	// startTime and endTime are set and time between startTime and endTime is more than 1 hour
	needBatch = needBatch || (!arg.StartTime.IsZero() && !arg.EndTime.IsZero() && arg.EndTime.Sub(arg.StartTime) > time.Hour)
	// Fall back to batch requests, if possible and necessary
	if needBatch {
		// fromId or start time must be set
		canBatch := arg.FromID == 0 != arg.StartTime.IsZero()
		if canBatch {
			// Split the request into multiple
			return b.batchAggregateTrades(ctx, arg, params)
		}

		// Can't handle this request locally or remotely
		// We would receive {"code":-1128,"msg":"Combination of optional parameters invalid."}
		return nil, errors.New("please set StartTime or FromId, but not both")
	}
	var resp []AggregatedTrade
	path := "/api/v3/aggTrades?" + params.Encode()
	return resp, b.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
}

// batchAggregateTrades fetches trades in multiple requests
// first phase, hourly requests until the first trade (or end time) is reached
// second phase, limit requests from previous trade until end time (or limit) is reached
func (b *Binance) batchAggregateTrades(ctx context.Context, arg *AggregatedTradeRequestParams, params url.Values) ([]AggregatedTrade, error) {
	var resp []AggregatedTrade
	// prepare first request with only first hour and max limit
	if arg.Limit == 0 || arg.Limit > 1000 {
		// Extend from the default of 500
		params.Set("limit", "1000")
	}

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
			path := "/api/v3/aggTrades?" + params.Encode()
			err := b.SendHTTPRequest(ctx,
				exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
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
		path := "/api/v3/aggTrades?" + params.Encode()
		var additionalTrades []AggregatedTrade
		err := b.SendHTTPRequest(ctx,
			exchange.RestSpotSupplementary,
			path,
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
func (b *Binance) GetSpotKline(ctx context.Context, arg *KlinesRequestParams) ([]CandleStick, error) {
	return b.retrieveSpotKline(ctx, arg, "/api/v3/klines")
}

// GetUIKline return modified kline data, optimized for presentation of candlestick charts.
func (b *Binance) GetUIKline(ctx context.Context, arg *KlinesRequestParams) ([]CandleStick, error) {
	return b.retrieveSpotKline(ctx, arg, "/api/v3/uiKlines")
}

func (b *Binance) retrieveSpotKline(ctx context.Context, arg *KlinesRequestParams, urlPath string) ([]CandleStick, error) {
	symbol, err := b.FormatSymbol(arg.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", arg.Interval)
	if arg.Limit != 0 {
		params.Set("limit", strconv.FormatInt(arg.Limit, 10))
	}
	if !arg.StartTime.IsZero() {
		params.Set("startTime", timeString(arg.StartTime))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", timeString(arg.EndTime))
	}

	path := urlPath + "?" + params.Encode()
	var resp [][]types.Number
	err = b.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		path,
		spotDefaultRate,
		&resp)
	if err != nil {
		return nil, err
	}
	klineData := make([]CandleStick, len(resp))
	for x := range resp {
		if len(resp[x]) != 12 {
			return nil, errors.New("unexpected kline data length")
		}
		klineData[x] = CandleStick{
			OpenTime:                 time.UnixMilli(resp[x][0].Int64()),
			Open:                     resp[x][1].Float64(),
			High:                     resp[x][2].Float64(),
			Low:                      resp[x][3].Float64(),
			Close:                    resp[x][4].Float64(),
			Volume:                   resp[x][5].Float64(),
			CloseTime:                time.UnixMilli(resp[x][6].Int64()),
			QuoteAssetVolume:         resp[x][7].Float64(),
			TradeCount:               resp[x][8].Float64(),
			TakerBuyAssetVolume:      resp[x][9].Float64(),
			TakerBuyQuoteAssetVolume: resp[x][10].Float64(),
		}
	}
	return klineData, nil
}

// GetAveragePrice returns current average price for a symbol.
//
// symbol: string of currency pair
func (b *Binance) GetAveragePrice(ctx context.Context, symbol currency.Pair) (*AveragePrice, error) {
	symbolValue, err := b.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	var resp *AveragePrice
	return resp, b.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary, "/api/v3/avgPrice?"+params.Encode(), spotDefaultRate, &resp)
}

// GetPriceChangeStats returns price change statistics for the last 24 hours
//
// symbol: string of currency pair
func (b *Binance) GetPriceChangeStats(ctx context.Context, symbol currency.Pair) (*PriceChangeStats, error) {
	params := url.Values{}
	rateLimit := spotPriceChangeAllRate
	if !symbol.IsEmpty() {
		rateLimit = spotDefaultRate
		symbolValue, err := b.FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp *PriceChangeStats
	return resp, b.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary, "/api/v3/ticker/24hr?"+params.Encode(), rateLimit, &resp)
}

// GetTickers returns the ticker data for the last 24 hrs
func (b *Binance) GetTickers(ctx context.Context) ([]PriceChangeStats, error) {
	var resp []PriceChangeStats
	return resp, b.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary, "/api/v3/ticker/24hr", spotPriceChangeAllRate, &resp)
}

// GetTradingDayTicker retrieves the price change statistics for the trading day
// possible tickerType values: FULL or MINI
func (b *Binance) GetTradingDayTicker(ctx context.Context, symbols currency.Pairs, timeZone, tickerType string) ([]PriceChangeStats, error) {
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
	return resp, b.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues("/api/v3/ticker/tradingDay", params), spotPriceChangeAllRate, &resp)
}

// GetLatestSpotPrice returns latest spot price of symbol
//
// symbol: string of currency pair
func (b *Binance) GetLatestSpotPrice(ctx context.Context, symbol currency.Pair) (*SymbolPrice, error) {
	params := url.Values{}
	rateLimit := spotSymbolPriceAllRate
	if !symbol.IsEmpty() {
		rateLimit = spotDefaultRate
		symbolValue, err := b.FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp *SymbolPrice
	return resp,
		b.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, "/api/v3/ticker/price?"+params.Encode(), rateLimit, &resp)
}

// GetBestPrice returns the latest best price for symbol
//
// symbol: string of currency pair
func (b *Binance) GetBestPrice(ctx context.Context, symbol currency.Pair) (*BestPrice, error) {
	params := url.Values{}
	rateLimit := spotOrderbookTickerAllRate
	if !symbol.IsEmpty() {
		rateLimit = spotDefaultRate
		symbolValue, err := b.FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp *BestPrice
	return resp,
		b.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, "/api/v3/ticker/bookTicker?"+params.Encode(), rateLimit, &resp)
}

// GetTickerData openTime always starts on a minute, while the closeTime is the current time of the request.
// As such, the effective window will be up to 59999ms wider than windowSize.
// possible windowSize values are FULL and MINI
func (b *Binance) GetTickerData(ctx context.Context, symbols currency.Pairs, windowSize time.Duration, tickerType string) ([]PriceChangeStats, error) {
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
	if windowSize < time.Minute {
		params.Set("windowSize", strconv.FormatInt(int64(windowSize/time.Minute), 10)+"m")
	} else if windowSize > (time.Hour * 24) {
		params.Set("windowSize", strconv.FormatInt(int64(windowSize/(time.Hour*24)), 10)+"h")
	}
	if tickerType != "" {
		params.Set("type", tickerType)
	}
	var resp PriceChangesWrapper
	return resp, b.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues("/api/v3/ticker", params), spotDefaultRate, &resp)
}

// NewOrder sends a new order to Binance
func (b *Binance) NewOrder(ctx context.Context, o *NewOrderRequest) (NewOrderResponse, error) {
	var resp NewOrderResponse
	if err := b.newOrder(ctx, "/api/v3/order", o, &resp); err != nil {
		return resp, err
	}

	if resp.Code != 0 {
		return resp, errors.New(resp.Msg)
	}

	return resp, nil
}

// NewOrderTest sends a new test order to Binance
func (b *Binance) NewOrderTest(ctx context.Context, o *NewOrderRequest) error {
	var resp NewOrderResponse
	return b.newOrder(ctx, "/api/v3/order/test", o, &resp)
}

func (b *Binance) newOrder(ctx context.Context, api string, o *NewOrderRequest, resp *NewOrderResponse) error {
	symbol, err := b.FormatSymbol(o.Symbol, asset.Spot)
	if err != nil {
		return err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", o.Side)
	params.Set("type", string(o.TradeType))
	if o.QuoteOrderQty > 0 {
		params.Set("quoteOrderQty", strconv.FormatFloat(o.QuoteOrderQty, 'f', -1, 64))
	} else {
		params.Set("quantity", strconv.FormatFloat(o.Quantity, 'f', -1, 64))
	}
	if o.TradeType == BinanceRequestParamsOrderLimit {
		params.Set("price", strconv.FormatFloat(o.Price, 'f', -1, 64))
	}
	if o.TimeInForce != "" {
		params.Set("timeInForce", string(o.TimeInForce))
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
	return b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, api, params, spotOrderRate, resp)
}

// CancelExistingOrder sends a cancel order to Binance
func (b *Binance) CancelExistingOrder(ctx context.Context, symbol currency.Pair, orderID int64, origClientOrderID string) (*CancelOrderResponse, error) {
	symbolValue, err := b.FormatSymbol(symbol, asset.Spot)
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
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodDelete, "/api/v3/order", params, spotOrderRate, &resp)
}

// OpenOrders Current open orders. Get all open orders on a symbol.
// Careful when accessing this with no symbol: The number of requests counted
// against the rate limiter is significantly higher
func (b *Binance) OpenOrders(ctx context.Context, pair currency.Pair) ([]TradeOrder, error) {
	var p string
	var err error
	params := url.Values{}
	if !pair.IsEmpty() {
		p, err = b.FormatSymbol(pair, asset.Spot)
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
	return resp, b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet,
		"/api/v3/openOrders",
		params, openOrdersLimit(p), &resp)
}

// AllOrders Get all account orders; active, canceled, or filled.
// orderId optional param
// limit optional param, default 500; max 500
func (b *Binance) AllOrders(ctx context.Context, symbol currency.Pair, orderID, limit string) ([]TradeOrder, error) {
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbolValue)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if limit != "" {
		params.Set("limit", limit)
	}
	var resp []TradeOrder
	return resp, b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet,
		"/api/v3/allOrders",
		params,
		spotAllOrdersRate,
		&resp)
}

// NewOCOOrder places a new one-cancel-other trade order.
func (b *Binance) NewOCOOrder(ctx context.Context, arg *OCOOrderParam) (*OCOOrderResponse, error) {
	if arg == nil || *arg == (OCOOrderParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if arg.Price <= 0 {
		return nil, order.ErrPriceBelowMin
	}
	if arg.StopPrice <= 0 {
		return nil, fmt.Errorf("%w stop price is required", order.ErrPriceBelowMin)
	}
	params := url.Values{}
	params.Set("symbol", arg.Symbol.String())
	params.Set("side", arg.Side)
	params.Set("amount", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	params.Set("stopPrice", strconv.FormatFloat(arg.StopPrice, 'f', -1, 64))
	if arg.ListClientOrderID != "" {
		params.Set("listClientOrderId", arg.ListClientOrderID)
	}
	if arg.LimitClientOrderID != "" {
		params.Set("limitClientOrderId", arg.LimitClientOrderID)
	}
	if arg.LimitStrategyID != "" {
		params.Set("limitStrategyId", arg.LimitStrategyID)
	}
	if arg.LimitStrategyType != "" {
		params.Set("limitStrategyType", arg.LimitStrategyType)
	}
	if arg.LimitIcebergQuantity > 0 {
		params.Set("limitIcebergQty", strconv.FormatFloat(arg.LimitIcebergQuantity, 'f', -1, 64))
	}
	if arg.TrailingDelta > 0 {
		params.Set("trailingDelta", strconv.FormatInt(arg.TrailingDelta, 10))
	}
	if arg.StopClientOrderID != "" {
		params.Set("stopClientOrderId", arg.StopClientOrderID)
	}
	if arg.StopStrategyID > 0 {
		params.Set("stopStrategyId", strconv.FormatInt(arg.StopStrategyID, 10))
	}
	if arg.StopStrategyType > 0 {
		params.Set("stopStrategyType", strconv.FormatInt(arg.StopStrategyType, 10))
	}
	if arg.StopLimitPrice > 0 {
		params.Set("stopLimitPrice", strconv.FormatFloat(arg.StopLimitPrice, 'f', -1, 64))
	}
	if arg.StopIcebergQuantity > 0 {
		params.Set("stopIcebergQty", strconv.FormatFloat(arg.StopIcebergQuantity, 'f', -1, 64))
	}
	if arg.StopLimitTimeInForce != "" {
		params.Set("stopLimitTimeInForce", arg.StopLimitTimeInForce)
	}
	if arg.NewOrderRespType != "" {
		params.Set("newOrderRespType", arg.NewOrderRespType)
	}
	if arg.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", arg.SelfTradePreventionMode)
	}
	var resp *OCOOrderResponse
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/api/v3/order/oco", params, spotDefaultRate, &resp)
}

// CancelOCOOrderList cancels an entire Order List.
func (b *Binance) CancelOCOOrderList(ctx context.Context, symbol, orderListID, listClientOrderID, newClientOrderID string) (*OCOOrderResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if orderListID == "" && listClientOrderID == "" {
		return nil, errOrderIDMustBeSet
	}
	if orderListID != "" {
		params.Set("orderListId", orderListID)
	}
	if listClientOrderID != "" {
		params.Set("listClientOrderId", listClientOrderID)
	}
	if newClientOrderID != "" {
		params.Set("newClientOrderId", newClientOrderID)
	}
	var resp *OCOOrderResponse
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, "/api/v3/orderList", params, spotDefaultRate, &resp)
}

// GetOCOOrders retrieves a specific OCO based on provided optional parameters
func (b *Binance) GetOCOOrders(ctx context.Context, orderListID, origiClientOrderID string) (*OCOOrderResponse, error) {
	if orderListID == "" && origiClientOrderID == "" {
		return nil, errOrderIDMustBeSet
	}
	params := url.Values{}
	if orderListID != "" {
		params.Set("orderListId", orderListID)
	}
	if origiClientOrderID != "" {
		params.Set("origClientOrderId", origiClientOrderID)
	}
	var resp *OCOOrderResponse
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/api/v3/orderList", params, spotDefaultRate, &resp)
}

// GetAllOCOOrders represents a list of OCO orders based on provided optional parameters.
func (b *Binance) GetAllOCOOrders(ctx context.Context, fromID string, startTime, endTime time.Time, limit int64) ([]OCOOrderResponse, error) {
	params := url.Values{}
	if fromID != "" {
		params.Set("fromId", fromID)
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
	var resp []OCOOrderResponse
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/api/v3/allOrderList", params, spotDefaultRate, &resp)
}

// GetOpenOCOList retrieves an open OCO orders.
func (b *Binance) GetOpenOCOList(ctx context.Context) ([]OCOOrderResponse, error) {
	var resp []OCOOrderResponse
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/api/v3/openOrderList", nil, spotDefaultRate, &resp)
}

// ----------------------------------------------------- Smart Order Routing (SOR) -----------------------------------------------------------

// NewOrderUsingSOR places an order using smart order routing (SOR).
func (b *Binance) NewOrderUsingSOR(ctx context.Context, arg *SOROrderRequestParams) (interface{}, error) {
	return b.newOrderUsingSOR(ctx, arg, "/api/v3/sor/order")
}

// NewOrderUsingSORTest est new order creation and signature/recvWindow using smart order routing (SOR).
// Creates and validates a new order but does not send it into the matching engine.
func (b *Binance) NewOrderUsingSORTest(ctx context.Context, arg *SOROrderRequestParams) (interface{}, error) {
	return b.newOrderUsingSOR(ctx, arg, "/api/v3/sor/order/test")
}

func (b *Binance) newOrderUsingSOR(ctx context.Context, arg *SOROrderRequestParams, path string) (interface{}, error) {
	if arg == nil || *arg == (SOROrderRequestParams{}) {
		return nil, common.ErrNilPointer
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
	if arg.TimeInForce != "" {
		params.Set("timeInForce", arg.TimeInForce)
	}
	if arg.Price <= 0 {
		params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	}
	if arg.NewClientOrderID != "" {
		params.Set("newClientOrderId", arg.NewClientOrderID)
	}
	if arg.StrategyID != 0 {
		params.Set("strategyId", strconv.FormatInt(arg.StrategyID, 10))
	}
	if arg.StrategyType != 0 {
		params.Set("strategyType", strconv.FormatInt(arg.StrategyType, 10))
	}
	if arg.IcebergQuantity != 0 {
		params.Set("icebergQty", strconv.FormatFloat(arg.IcebergQuantity, 'f', -1, 64))
	}
	if arg.NewOrderResponseType != "" {
		params.Set("newOrderRespType", arg.NewOrderResponseType)
	}
	if arg.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", arg.SelfTradePreventionMode)
	}
	var resp *SOROrderResponse
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, path, params, spotDefaultRate, &resp)
}

// QueryOrder returns information on a past order
func (b *Binance) QueryOrder(ctx context.Context, symbol currency.Pair, origClientOrderID string, orderID int64) (*TradeOrder, error) {
	symbolValue, err := b.FormatSymbol(symbol, asset.Spot)
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
	if err := b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet, "/api/v3/order",
		params, spotOrderQueryRate,
		&resp); err != nil {
		return resp, err
	}
	if resp.Code != 0 {
		return resp, errors.New(resp.Msg)
	}
	return resp, nil
}

// GetAccount returns binance user accounts
func (b *Binance) GetAccount(ctx context.Context) (*Account, error) {
	type response struct {
		Response
		Account
	}

	var resp response
	if err := b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet, "/api/v3/account",
		nil, spotAccountInformationRate,
		&resp); err != nil {
		return &resp.Account, err
	}

	if resp.Code != 0 {
		return &resp.Account, errors.New(resp.Msg)
	}
	return &resp.Account, nil
}

// GetAccountTradeList retrieves trades for a specific account and symbol.
func (b *Binance) GetAccountTradeList(ctx context.Context, symbol, orderID string, startTime, endTime time.Time, fromID, limit int64) ([]AccountTradeItem, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if fromID != 0 {
		params.Set("fromId", strconv.FormatInt(fromID, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []AccountTradeItem
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/api/v3/myTrades", params, spotDefaultRate, &resp)
}

// GetMarginAccount returns account information for margin accounts
func (b *Binance) GetMarginAccount(ctx context.Context) (*MarginAccount, error) {
	var resp *MarginAccount
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/account", nil, spotAccountInformationRate, &resp)
}

// SendHTTPRequest sends an unauthenticated request
func (b *Binance) SendHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result interface{}) error {
	endpointPath, err := b.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	item := &request.Item{
		Method:        http.MethodGet,
		Path:          endpointPath + path,
		Result:        result,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording}

	return b.SendPayload(ctx, f, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

// SendAPIKeyHTTPRequest is a special API request where the api key is
// appended to the headers without a secret
func (b *Binance) SendAPIKeyHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result interface{}) error {
	endpointPath, err := b.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}

	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return err
	}

	headers := make(map[string]string)
	headers["X-MBX-APIKEY"] = creds.Key
	item := &request.Item{
		Method:        http.MethodGet,
		Path:          endpointPath + path,
		Headers:       headers,
		Result:        result,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording}

	return b.SendPayload(ctx, f, func() (*request.Item, error) {
		return item, nil
	}, request.AuthenticatedRequest)
}

// SendAuthHTTPRequest sends an authenticated HTTP request
func (b *Binance) SendAuthHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, params url.Values, f request.EndpointLimit, result interface{}) error {
	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return err
	}

	endpointPath, err := b.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}

	if params == nil {
		params = url.Values{}
	}

	if params.Get("recvWindow") == "" {
		params.Set("recvWindow", strconv.FormatInt(defaultRecvWindow.Milliseconds(), 10))
	}

	interim := json.RawMessage{}
	err = b.SendPayload(ctx, f, func() (*request.Item, error) {
		fullPath := endpointPath + path
		params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
		signature := params.Encode()
		var hmacSigned []byte
		hmacSigned, err = crypto.GetHMAC(crypto.HashSHA256,
			[]byte(signature),
			[]byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		hmacSignedStr := crypto.HexEncodeToString(hmacSigned)
		headers := make(map[string]string)
		headers["X-MBX-APIKEY"] = creds.Key
		fullPath = common.EncodeURLValues(fullPath, params)
		fullPath += "&signature=" + hmacSignedStr
		return &request.Item{
			Method:        method,
			Path:          fullPath,
			Headers:       headers,
			Result:        &interim,
			Verbose:       b.Verbose,
			HTTPDebugging: b.HTTPDebugging,
			HTTPRecording: b.HTTPRecording}, nil
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
func (b *Binance) CheckLimit(limit int64) error {
	for x := range b.validLimits {
		if b.validLimits[x] == limit {
			return nil
		}
	}
	return errors.New("incorrect limit values - valid values are 5, 10, 20, 50, 100, 500, 1000")
}

// SetValues sets the default valid values
func (b *Binance) SetValues() {
	b.validLimits = []int64{5, 10, 20, 50, 100, 500, 1000, 5000}
}

// GetFee returns an estimate of fee based on type of transaction
func (b *Binance) GetFee(ctx context.Context, feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		multiplier, err := b.getMultiplier(ctx, feeBuilder.IsMaker)
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
func (b *Binance) getMultiplier(ctx context.Context, isMaker bool) (float64, error) {
	var multiplier float64
	account, err := b.GetAccount(ctx)
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
func (b *Binance) GetSystemStatus(ctx context.Context) (*SystemStatus, error) {
	var resp *SystemStatus
	return resp, b.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, "/sapi/v1/system/status", walletSystemStatus, &resp)
}

// GetAllCoinsInfo returns details about all supported coins(available for deposit and withdraw)
func (b *Binance) GetAllCoinsInfo(ctx context.Context) ([]CoinInfo, error) {
	var resp []CoinInfo
	if err := b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet,
		"/sapi/v1/capital/config/getall",
		nil,
		spotDefaultRate,
		&resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetDailyAccountSnapshot retrieves daily account snapshot
func (b *Binance) GetDailyAccountSnapshot(ctx context.Context, tradeType string, startTime, endTime time.Time, limit int64) (*DailyAccountSnapshot, error) {
	if tradeType == "" {
		return nil, fmt.Errorf("%w type: %s", asset.ErrInvalidAsset, tradeType)
	}
	params := url.Values{}
	params.Set("type", tradeType)
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *DailyAccountSnapshot
	return resp, b.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/sapi/v1/accountSnapshot", params), spotDefaultRate, &resp)
}

// DisableFastWithdrawalSwitch disables fast withdrawal switch
// This request will disable fastwithdraw switch under your account.
// You need to enable "trade" option for the api key which requests this endpoint.
func (b *Binance) DisableFastWithdrawalSwitch(ctx context.Context) error {
	return b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/account/disableFastWithdrawSwitch", nil, spotDefaultRate, &struct{}{})
}

// EnableFastWithdrawalSwitch enable fastwithdraw switch under your account.
func (b *Binance) EnableFastWithdrawalSwitch(ctx context.Context) error {
	return b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/account/enableFastWithdrawSwitch", nil, spotDefaultRate, &struct{}{})
}

// WithdrawCrypto sends cryptocurrency to the address of your choosing
func (b *Binance) WithdrawCrypto(ctx context.Context, cryptoAsset, withdrawOrderID, network, address, addressTag, name, amount string, transactionFeeFlag bool) (string, error) {
	if cryptoAsset == "" || address == "" || amount == "" {
		return "", errors.New("asset, address and amount must not be empty")
	}

	params := url.Values{}
	params.Set("coin", cryptoAsset)
	params.Set("address", address)
	params.Set("amount", amount)

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
	if err := b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodPost,
		"/sapi/v1/capital/withdraw/apply",
		params,
		spotDefaultRate,
		&resp); err != nil {
		return "", err
	}

	if resp.ID == "" {
		return "", errors.New("ID is nil")
	}

	return resp.ID, nil
}

// DepositHistory returns the deposit history based on the supplied params
// status `param` used as string to prevent default value 0 (for int) interpreting as EmailSent status
func (b *Binance) DepositHistory(ctx context.Context, c currency.Code, status string, startTime, endTime time.Time, offset, limit int) ([]DepositHistory, error) {
	var response []DepositHistory

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

	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UTC().UnixMilli(), 10))
	}

	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UTC().UnixMilli(), 10))
	}

	if offset != 0 {
		params.Set("offset", strconv.Itoa(offset))
	}

	if limit != 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	if err := b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet,
		"/sapi/v1/capital/deposit/hisrec",
		params,
		spotDefaultRate,
		&response); err != nil {
		return nil, err
	}

	return response, nil
}

// WithdrawHistory gets the status of recent withdrawals
// status `param` used as string to prevent default value 0 (for int) interpreting as EmailSent status
func (b *Binance) WithdrawHistory(ctx context.Context, c currency.Code, status string, startTime, endTime time.Time, offset, limit int) ([]WithdrawStatusResponse, error) {
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

	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UTC().UnixMilli(), 10))
	}

	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UTC().UnixMilli(), 10))
	}

	if offset != 0 {
		params.Set("offset", strconv.Itoa(offset))
	}

	if limit != 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	var withdrawStatus []WithdrawStatusResponse
	if err := b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet,
		"/sapi/v1/capital/withdraw/history",
		params,
		spotDefaultRate,
		&withdrawStatus); err != nil {
		return nil, err
	}

	return withdrawStatus, nil
}

// GetDepositAddressForCurrency retrieves the wallet address for a given currency
func (b *Binance) GetDepositAddressForCurrency(ctx context.Context, currency, chain string) (*DepositAddress, error) {
	params := url.Values{}
	params.Set("coin", currency)
	if chain != "" {
		params.Set("network", chain)
	}
	params.Set("recvWindow", "10000")
	var d *DepositAddress
	return d,
		b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/capital/deposit/address", params, spotDefaultRate, &d)
}

// GetAssetsThatCanBeConvertedIntoBNB retrieves assets that can be converted into BNB
func (b *Binance) GetAssetsThatCanBeConvertedIntoBNB(ctx context.Context, accountType string) (*AssetsDust, error) {
	params := url.Values{}
	if accountType != "" {
		params.Set("accountType", accountType)
	}
	var resp *AssetsDust
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/asset/dust-btc", params, spotDefaultRate, &resp)
}

// DustTransfer convert dust assets to BNB.
func (b *Binance) DustTransfer(ctx context.Context, assets []string, accountType string) (*Dusts, error) {
	if len(assets) == 0 {
		return nil, fmt.Errorf("%w, assets must not be empty", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("assets", strings.Join(assets, ","))
	if accountType != "" {
		params.Set("accountType", accountType)
	}
	var resp *Dusts
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/asset/dust", params, spotDefaultRate, &resp)
}

// GetAssetDevidendRecords query asset dividend record.
func (b *Binance) GetAssetDevidendRecords(ctx context.Context, asset currency.Code, startTime, endTime time.Time, limit int64) (interface{}, error) {
	if asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("asset", asset.String())
	if startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *AssetDividendRecord
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/asset/assetDividend", params, spotDefaultRate, &resp)
}

// GetAssetDetail fetches details of assets supported on Binance
func (b *Binance) GetAssetDetail(ctx context.Context, asset currency.Code) (map[string]DividendAsset, error) {
	params := url.Values{}
	if !asset.IsEmpty() {
		params.Set("asset", asset.String())
	}
	var resp map[string]DividendAsset
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/asset/assetDetail", params, spotDefaultRate, &resp)
}

// GetTradeFees fetch trade fee
func (b *Binance) GetTradeFees(ctx context.Context, symbol currency.Pair) ([]TradeFee, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	var resp []TradeFee
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/asset/tradeFee", params, spotDefaultRate, &resp)
}

// UserUniversalTransfer transfers an asset
// You need to enable Permits Universal Transfer option for the API Key which requests this endpoint.
// fromSymbol must be sent when type are ISOLATEDMARGIN_MARGIN and ISOLATEDMARGIN_ISOLATEDMARGIN
// toSymbol must be sent when type are MARGIN_ISOLATEDMARGIN and ISOLATEDMARGIN_ISOLATEDMARGIN
func (b *Binance) UserUniversalTransfer(ctx context.Context, transferType TransferTypes, amount float64, asset currency.Code, fromSymbol, toSymbol string) (string, error) {
	if transferType == 0 {
		return "", errors.New("transfer type is required")
	}
	if asset.IsEmpty() {
		return "", fmt.Errorf("asset %w", currency.ErrCurrencyCodeEmpty)
	}
	if amount <= 0 {
		return "", order.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	params.Set("type", transferType.String())
	params.Set("asset", asset.String())
	if fromSymbol == "" {
		params.Set("fromSymbol", fromSymbol)
	}
	if toSymbol == "" {
		params.Set("toSymbol", toSymbol)
	}
	resp := &struct {
		TransferID string `json:"tranId"`
	}{}
	return resp.TransferID, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/asset/transfer", params, spotDefaultRate, &resp)
}

// GetUserUniversalTransferHistory retrieves user universal transfer history
func (b *Binance) GetUserUniversalTransferHistory(ctx context.Context, transferType TransferTypes, startTime, endTime time.Time, current int64, size float64, fromSymbol, toSymbol string) (*UniversalTransferHistory, error) {
	if transferType == 0 {
		return nil, errors.New("transfer type is required")
	}
	params := url.Values{}
	params.Set("type", transferType.String())
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if current != 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatFloat(size, 'f', -1, 64))
	}
	if fromSymbol != "" {
		params.Set("fromSymbol", fromSymbol)
	}
	if toSymbol != "" {
		params.Set("toSymbol", toSymbol)
	}
	var resp *UniversalTransferHistory
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/asset/transfer", params, spotDefaultRate, &resp)
}

// GetFundingAssets funding wallet
func (b *Binance) GetFundingAssets(ctx context.Context, asset currency.Code, needBTCValuation bool) ([]FundingAsset, error) {
	params := url.Values{}
	if !asset.IsEmpty() {
		params.Set("asset", asset.String())
	}
	if needBTCValuation {
		params.Set("needBtcValuation", "true")
	}
	var resp []FundingAsset
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/asset/get-funding-asset", params, spotDefaultRate, &resp)
}

// GetUserAssets get user assets, just for positive data.
func (b *Binance) GetUserAssets(ctx context.Context, ccy currency.Code, needBTCValuation bool) ([]FundingAsset, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("asset", ccy.String())
	}
	if needBTCValuation {
		params.Set("needBtcValuation", "true")
	}
	var resp []FundingAsset
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v3/asset/getUserAsset", params, spotDefaultRate, &resp)
}

// ConvertBUSD convert transfer, convert between BUSD and stablecoins.
// accountType: possible values are MAIN and CARD
func (b *Binance) ConvertBUSD(ctx context.Context, clientTransactionID, accountType string, assetCcy, targetAsset currency.Code, amount float64) (*AssetConverResponse, error) {
	if clientTransactionID == "" {
		return nil, errors.New("client transaction ID is required")
	}
	if assetCcy.IsEmpty() {
		return nil, fmt.Errorf("%w assetCcy is empty", currency.ErrCurrencyCodeEmpty)
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
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
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/asset/convert-transfer", params, spotDefaultRate, &resp)
}

// BUSDConvertHistory convert transfer, convert between BUSD and stablecoins.
func (b *Binance) BUSDConvertHistory(ctx context.Context, transactionID, clientTransactionID, accountType string, assetCcy, targetAsset currency.Code, amount float64) (*BUSDConvertHistory, error) {
	params := url.Values{}
	if transactionID != "" {
		params.Set("tranid", transactionID)
	}
	if clientTransactionID != "" {
		params.Set("clientTranId", clientTransactionID)
	}
	if !assetCcy.IsEmpty() {
		params.Set("asset", assetCcy.String())
	}
	if amount != 0 {
		params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	}
	if !targetAsset.IsEmpty() {
		params.Set("targetAsset", targetAsset.String())
	}
	if accountType != "" {
		params.Set("accountType", accountType)
	}
	var resp *BUSDConvertHistory
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/asset/convert-transfer/queryByPage", params, spotDefaultRate, &resp)
}

// GetCloudMiningPaymentAndRefundHistory retrieves cloud-mining payment and refund history
func (b *Binance) GetCloudMiningPaymentAndRefundHistory(ctx context.Context, transactionID, current int64, clientTransactionID string, assetCcy currency.Code, startTime, endTime time.Time, size float64) (*CloudMiningPR, error) {
	params := url.Values{}
	if transactionID != 0 {
		params.Set("tranId", strconv.FormatInt(transactionID, 10))
	}
	if current != 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if clientTransactionID != "" {
		params.Set("clientTranId", clientTransactionID)
	}
	if !assetCcy.IsEmpty() {
		params.Set("asset", assetCcy.String())
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if size != 0 {
		params.Set("size", strconv.FormatFloat(size, 'f', -1, 64))
	}
	var resp *CloudMiningPR
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/asset/ledger-transfer/cloud-mining/queryByPage", params, spotDefaultRate, &resp)
}

// GetAPIKeyPermission retrieves API key ermissions detail.
func (b *Binance) GetAPIKeyPermission(ctx context.Context) (*APIKeyPermissions, error) {
	var resp *APIKeyPermissions
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/account/apiRestrictions", nil, spotDefaultRate, &resp)
}

// GetAutoConvertingStableCoins a user's auto-conversion settings in deposit/withdrawal
func (b *Binance) GetAutoConvertingStableCoins(ctx context.Context) (*AutoConvertingStableCoins, error) {
	var resp *AutoConvertingStableCoins
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/capital/contract/convertible-coins", nil, spotDefaultRate, &resp)
}

// SwitchOnOffBUSDAndStableCoinsConversion user can use it to turn on or turn off the BUSD auto-conversion from/to a specific stable coin.
func (b *Binance) SwitchOnOffBUSDAndStableCoinsConversion(ctx context.Context, coin currency.Code, enable bool) error {
	if coin.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", coin.String())
	params.Set("enable", strconv.FormatBool(enable))
	return b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/capital/contract/convertible-coins", params, spotDefaultRate, &struct{}{})
}

// OneClickArrivalDepositApply apply deposit credit for expired address
func (b *Binance) OneClickArrivalDepositApply(ctx context.Context, transactionID string, subAccountID, subUserID, depositID int64) (bool, error) {
	params := url.Values{}
	if transactionID != "" {
		params.Set("txId", transactionID)
	}
	if depositID != 0 {
		params.Set("depositId", strconv.FormatInt(depositID, 10))
	}
	if subAccountID != 0 {
		params.Set("subAccountId", strconv.FormatInt(subAccountID, 10))
	}
	if subUserID != 0 {
		params.Set("subUserID", strconv.FormatInt(subUserID, 10))
	}
	var resp bool
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/capital/deposit/credit-apply", params, spotDefaultRate, &resp)
}

// GetDepositAddressListWithNetwork fetch deposit address list with network.
func (b *Binance) GetDepositAddressListWithNetwork(ctx context.Context, coin currency.Code, network string) ([]DepositAddressAndNetwork, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", coin.String())
	if network != "" {
		params.Set("network", network)
	}
	var resp []DepositAddressAndNetwork
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/capital/deposit/address/list", params, spotDefaultRate, &resp)
}

// GetUserWalletBalance retrieves user wallet balance.
func (b *Binance) GetUserWalletBalance(ctx context.Context) ([]UserWalletBalance, error) {
	var resp []UserWalletBalance
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/asset/wallet/balance", nil, spotDefaultRate, &resp)
}

// GetUserDelegationHistory query User Delegation History for Master account.
// The delegation type has two values: delegated or undelegated.
func (b *Binance) GetUserDelegationHistory(ctx context.Context, email, delegation string, startTime, endTime time.Time, asset currency.Code, current int64, size float64) (*UserDelegationHistory, error) {
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
	if !asset.IsEmpty() {
		params.Set("asset", asset.String())
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
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/asset/custody/transfer-history", params, spotDefaultRate, &resp)
}

// GetSymbolsDelistScheduleForSpot symbols delist schedule for spot
func (b *Binance) GetSymbolsDelistScheduleForSpot(ctx context.Context) ([]DelistSchedule, error) {
	var resp []DelistSchedule
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/spot/delist-schedule", nil, spotDefaultRate, &resp)
}

// --------------------------------------------------    ---------------------------------------------------------------------------------

// CreateVirtualSubAccount creates a virtual subaccount information.
func (b *Binance) CreateVirtualSubAccount(ctx context.Context, subAccountString string) (*VirtualSubAccount, error) {
	params := url.Values{}
	if subAccountString != "" {
		params.Set("subAccountString", subAccountString)
	}
	var resp *VirtualSubAccount
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/virtualSubAccount", params, spotDefaultRate, &resp)
}

// GetSubAccountList retrieves sub-account list for Master Account
func (b *Binance) GetSubAccountList(ctx context.Context, email string, isFreeze bool, page, limit int64) (*SubAccountList, error) {
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
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/list", params, spotDefaultRate, &resp)
}

// GetSubAccountSpotAssetTransferHistory represents sub-account spot asset transfer history for master account
func (b *Binance) GetSubAccountSpotAssetTransferHistory(ctx context.Context, fromEmail, toEmail string,
	startTime, endTime time.Time, page, limit int64) ([]SubAccountSpotAsset, error) {
	params := url.Values{}
	if common.MatchesEmailPattern(fromEmail) {
		params.Set("fromEmail", fromEmail)
	}
	if common.MatchesEmailPattern(toEmail) {
		params.Set("toEmail", toEmail)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []SubAccountSpotAsset
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/sub/transfer/history", params, spotDefaultRate, &resp)
}

// GetSubAccountFuturesAssetTransferHistory Query Sub-account Futures Asset Transfer History For Master Account
func (b *Binance) GetSubAccountFuturesAssetTransferHistory(ctx context.Context, email string, startTime, endTime time.Time, futuresType, page, limit int64) (*AssetTransferHistory, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	if futuresType != 1 && futuresType != 2 {
		return nil, errInvalidFuturesType
	}
	params := url.Values{}
	params.Set("email", email)
	params.Set("futuresType", strconv.FormatInt(futuresType, 10))
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page != 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *AssetTransferHistory
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/futures/internalTransfer", params, spotDefaultRate, &resp)
}

// SubAccountFuturesAssetTransfer sub-account futures asset transfer for master account
// futuresType: 1:USDT-margined Futures，2: Coin-margined Futures
func (b *Binance) SubAccountFuturesAssetTransfer(ctx context.Context, fromEmail, toEmail string, futuresType int64,
	asset currency.Code, amount float64) (*FuturesAssetTransfer, error) {
	if !common.MatchesEmailPattern(fromEmail) {
		return nil, fmt.Errorf("%w, fromEmail=%s", errValidEmailRequired, fromEmail)
	}
	if !common.MatchesEmailPattern(toEmail) {
		return nil, fmt.Errorf("%w, toEmail=%s", errValidEmailRequired, toEmail)
	}
	if futuresType != 0 && futuresType != 1 {
		return nil, fmt.Errorf("%w 1: USDT-margined Futures or 2: Coin-margined Futures", errInvalidFuturesType)
	}
	if !asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	params := url.Values{}
	var resp *FuturesAssetTransfer
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/sub-account/futures/internalTransfer", params, spotDefaultRate, &resp)
}

// GetSubAccountAssets sub-account assets for master account
func (b *Binance) GetSubAccountAssets(ctx context.Context, email string) (*SubAccountAssets, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	var resp *SubAccountAssets
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v4/sub-account/assets", params, spotDefaultRate, &resp)
}

// GetManagedSubAccountList retrieves investor's managed sub-account list.
func (b *Binance) GetManagedSubAccountList(ctx context.Context, email string, page, limit int64) (*ManagedSubAccountList, error) {
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
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/managed-subaccount/info", params, spotDefaultRate, &resp)
}

// GetSubAccountTransactionStatistics retrieves sub-account Transaction statistics (For Master Account)(USER_DATA).
func (b *Binance) GetSubAccountTransactionStatistics(ctx context.Context, email string) ([]SubAccountTransactionStatistics, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	var resp []SubAccountTransactionStatistics
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/transaction-statistics", params, spotDefaultRate, &resp)
}

// GetManagedSubAccountDepositAddress retrieves investor's managed sub-account deposit address.
// get managed sub-account deposit address (For Investor Master Account) (USER_DATA)
func (b *Binance) GetManagedSubAccountDepositAddress(ctx context.Context, coin currency.Code, email, network string) (*ManagedSubAccountDepositAddres, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", coin.String())
	params.Set("email", email)
	params.Set("network", network)
	var resp *ManagedSubAccountDepositAddres
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/managed-subaccount/deposit/address", params, spotDefaultRate, &resp)
}

// EnableOptionsForSubAccount enables options for sub-account(For master account)
func (b *Binance) EnableOptionsForSubAccount(ctx context.Context, email string) (*OptionsEnablingResponse, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	var resp *OptionsEnablingResponse
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/sub-account/eoptions/enable", params, spotDefaultRate, &resp)
}

// GetManagedSubAccountTransferLog retrieves managed sub account transfer Log (For Trading Team Sub Account)
// transfers: Transfer Direction (FROM/TO)
// transferFunctionAccountType: Transfer function account type (SPOT/MARGIN/ISOLATED_MARGIN/USDT_FUTURE/COIN_FUTURE)
func (b *Binance) GetManagedSubAccountTransferLog(ctx context.Context, startTime, endTime time.Time,
	page, limit int64, transfers, transferFunctionAccountType string) (*ManagedSubAccountTransferLog, error) {
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
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/managed-subaccount/query-trans-log", params, spotDefaultRate, &resp)
}

// GetSubAccountSpotAssetsSummary retrieves BTC valued asset summary of subaccounts.
func (b *Binance) GetSubAccountSpotAssetsSummary(ctx context.Context, email string, page, size int64) (*SubAccountSpotSummary, error) {
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
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/spotSummary", params, spotDefaultRate, &resp)
}

// GetSubAccountDepositAddress sub-account deposit address
func (b *Binance) GetSubAccountDepositAddress(ctx context.Context, email, coin, network string, amount float64) (*SubAccountDepositAddress, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	if coin == "" {
		return nil, fmt.Errorf("%w, coin=%s", currency.ErrCurrencyCodeEmpty, coin)
	}
	params := url.Values{}
	if network != "" {
		params.Set("network", network)
	}
	if amount > 0 {
		params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	}
	var resp *SubAccountDepositAddress
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/capital/deposit/subAddress", params, spotDefaultRate, &resp)
}

// GetSubAccountDepositHistory retrieves sub-account deposit history
func (b *Binance) GetSubAccountDepositHistory(ctx context.Context, email, coin string,
	startTime, endTime time.Time, status, offset, limit int64) (*SubAccountDepositHistory, error) {
	params := url.Values{}
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params.Set("email", email)
	if coin != "" {
		params.Set("coin", coin)
	}
	if status != 0 {
		params.Set("status", strconv.FormatInt(status, 10))
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
	if offset > 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	var resp *SubAccountDepositHistory
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/capital/deposit/subHisrec", params, spotDefaultRate, &resp)
}

// GetSubAccountStatusOnMarginFutures sub-account's status on Margin/Futures for master account
func (b *Binance) GetSubAccountStatusOnMarginFutures(ctx context.Context, email string) (*SubAccountStatus, error) {
	params := url.Values{}
	if common.MatchesEmailPattern(email) {
		params.Set("email", email)
	}
	var resp *SubAccountStatus
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/status", params, spotDefaultRate, &resp)
}

// EnableMarginForSubAccount Enable Margin for Sub-account For Master Account
func (b *Binance) EnableMarginForSubAccount(ctx context.Context, email string) (*MarginEnablingResponse, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	var resp *MarginEnablingResponse
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/sub-account/margin/enable", params, spotDefaultRate, &resp)
}

// GetDetailOnSubAccountMarginAccount retrieves Detail on Sub-account's Margin Account For Master Account
func (b *Binance) GetDetailOnSubAccountMarginAccount(ctx context.Context, email string) (*SubAccountMarginAccountDetail, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	var resp *SubAccountMarginAccountDetail
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/margin/account", params, spotDefaultRate, &resp)
}

// GetSummaryOfSubAccountMarginAccount retrieves summary of sub-account's margin account for master account
func (b *Binance) GetSummaryOfSubAccountMarginAccount(ctx context.Context) (*SubAccountMarginAccount, error) {
	var resp *SubAccountMarginAccount
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/margin/accountSummary", nil, spotDefaultRate, &resp)
}

// EnableFuturesSubAccount enables futures for Sub-account for master account
func (b *Binance) EnableFuturesSubAccount(ctx context.Context, email string) (*FuturesEnablingResponse, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	var resp *FuturesEnablingResponse
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/futures/enable", params, spotDefaultRate, &resp)
}

// GetDetailSubAccountFuturesAccount retrieves detail on sub-account's futures account for master account
func (b *Binance) GetDetailSubAccountFuturesAccount(ctx context.Context, email string) (*SubAccountsFuturesAccount, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	var resp *SubAccountsFuturesAccount
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/futures/accountSummary", params, spotDefaultRate, &resp)
}

// GetSummaryOfSubAccountFuturesAccount retrieves summary of sub-account's futures account for master account
func (b *Binance) GetSummaryOfSubAccountFuturesAccount(ctx context.Context) (*SubAccountsFuturesAccount, error) {
	var resp *SubAccountsFuturesAccount
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/futures/accountSummary", nil, spotDefaultRate, &resp)
}

// GetFuturesPositionRiskSubAccount retrieves futures position-risk of sub-account for master account
func (b *Binance) GetFuturesPositionRiskSubAccount(ctx context.Context, email string) (*SubAccountFuturesPositionRisk, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	var resp *SubAccountFuturesPositionRisk
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v2/sub-account/futures/positionRisk", params, spotDefaultRate, &resp)
}

// EnableLeverageTokenForSubAccount enables leverage token sub-account form master account.
func (b *Binance) EnableLeverageTokenForSubAccount(ctx context.Context, email string, enableElvt bool) (*LeverageToken, error) {
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
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/sub-account/blvt/enable", params, spotDefaultRate, &resp)
}

// GetIPRestrictionForSubAccountAPIKey retrieves list of IP addresses restricted for the sub account API key(for master account).
func (b *Binance) GetIPRestrictionForSubAccountAPIKey(ctx context.Context, email, subAccountAPIKey string) (*APIRestrictions, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	if subAccountAPIKey == "" {
		return nil, errEmptySubAccountEPIKey
	}
	params := url.Values{}
	params.Set("email", email)
	params.Set("subAccountApiKey", subAccountAPIKey)
	var resp *APIRestrictions
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/subAccountApi/ipRestriction", params, spotDefaultRate, &resp)
}

// DeleteIPListForSubAccountAPIKey delete IP list for a sub-account API key (For Master Account)
func (b *Binance) DeleteIPListForSubAccountAPIKey(ctx context.Context, email, subAccountAPIKey, ipAddress string) (*APIRestrictions, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	if subAccountAPIKey == "" {
		return nil, errEmptySubAccountEPIKey
	}
	params := url.Values{}
	params.Set("email", email)
	params.Set("subAccountApiKey", subAccountAPIKey)
	if ipAddress != "" {
		params.Set("ipAddress", ipAddress)
	}
	var resp *APIRestrictions
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodDelete, "/sapi/v1/sub-account/subAccountApi/ipRestriction/ipList", params, spotDefaultRate, &resp)
}

// AddIPRestrictionForSubAccountAPIkey adds an IP address into the restricted IP addresses for the subaccount
func (b *Binance) AddIPRestrictionForSubAccountAPIkey(ctx context.Context, email, subAccountAPIKey, ipAddress string, restricted bool) (*APIRestrictions, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	if subAccountAPIKey == "" {
		return nil, errEmptySubAccountEPIKey
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
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/sub-account/subAccountApi/ipRestriction/ipList", params, spotDefaultRate, &resp)
}

// DepositAssetsIntoTheManagedSubAccount deposits an asset into managed sub-account (for investor master account).
func (b *Binance) DepositAssetsIntoTheManagedSubAccount(ctx context.Context, toEmail string, asset currency.Code, amount float64) (string, error) {
	if !common.MatchesEmailPattern(toEmail) {
		return "", fmt.Errorf("%w, toEmail = %s", errValidEmailRequired, toEmail)
	}
	if asset.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return "", errAmountMustBeSet
	}
	params := url.Values{}
	params.Set("toEmail", toEmail)
	params.Set("asset", asset.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	resp := &struct {
		TransactionID string `json:"tranId"`
	}{}
	return resp.TransactionID, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/managed-subaccount/deposit", params, spotDefaultRate, &resp)
}

// GetManagedSubAccountAssetsDetails retrieves managed sub-account assets details for investor master accounts.
func (b *Binance) GetManagedSubAccountAssetsDetails(ctx context.Context, email string) ([]ManagedSubAccountAssetInfo, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	var resp []ManagedSubAccountAssetInfo
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/managed-subaccount/asset", params, spotDefaultRate, &resp)
}

// WithdrawAssetsFromManagedSubAccount withdraws an asset from managed sub-account(for investor master account).
func (b *Binance) WithdrawAssetsFromManagedSubAccount(ctx context.Context, fromEmail string, asset currency.Code, amount float64, transferDate time.Time) (string, error) {
	if !common.MatchesEmailPattern(fromEmail) {
		return "", fmt.Errorf("%w fromEmail=%s", errValidEmailRequired, fromEmail)
	}
	if asset.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return "", errAmountMustBeSet
	}
	params := url.Values{}
	params.Set("fromEmail", fromEmail)
	params.Set("asset", asset.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	if !transferDate.IsZero() {
		params.Set("transferData", strconv.FormatInt(transferDate.UnixMilli(), 10))
	}
	resp := &struct {
		TransactionID string `json:"tranId"`
	}{}
	return resp.TransactionID, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/managed-subaccount/withdraw", params, spotDefaultRate, &resp)
}

// GetManagedSubAccountSnapshot retrieves managed sub-account snapshot for investor master account.
func (b *Binance) GetManagedSubAccountSnapshot(ctx context.Context, email, assetType string, startTime, endTime time.Time, limit int64) (*SubAccountAssetsSnapshot, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, fmt.Errorf("%w email=%s", errValidEmailRequired, email)
	}
	if assetType == "" {
		return nil, errors.New("invalid assets type")
	}
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
	var resp *SubAccountAssetsSnapshot
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/managed-subaccount/accountSnapshot", params, spotDefaultRate, &resp)
}

// GetManagedSubAccountTransferLogForInvestorMasterAccount retrieves  managed sub account transfer log. This endpoint is available for investor of Managed Sub-Account.
// A Managed Sub-Account is an account type for investors who value flexibility in asset allocation and account application,
// while delegating trades to a professional trading team.
func (b *Binance) GetManagedSubAccountTransferLogForInvestorMasterAccount(ctx context.Context, email string, startTime, endTime time.Time, page, limit int64) (*SubAccountTransferLog, error) {
	return b.getManagedSubAccountTransferLog(ctx, email, "/sapi/v1/managed-subaccount/queryTransLogForInvestor", startTime, endTime, page, limit)
}

// GetManagedSubAccountTransferLogForTradingTeam retrieves managed sub account transfer log.
// This endpoint is available for investor of Managed Sub-Account. A Managed Sub-Account is an account type for investors who value flexibility in asset allocation and account application,
// while delegating trades to a professional trading team.
func (b *Binance) GetManagedSubAccountTransferLogForTradingTeam(ctx context.Context, email string, startTime, endTime time.Time, page, limit int64) (*SubAccountTransferLog, error) {
	return b.getManagedSubAccountTransferLog(ctx, email, "/sapi/v1/managed-subaccount/queryTransLogForTradeParent", startTime, endTime, page, limit)
}

func (b *Binance) getManagedSubAccountTransferLog(ctx context.Context, email, path string, startTime, endTime time.Time, page, limit int64) (*SubAccountTransferLog, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, fmt.Errorf("%w, email = %s", errValidEmailRequired, email)
	}
	err := common.StartEndTimeCheck(startTime, endTime)
	if err != nil {
		return nil, err
	}
	if page < 0 {
		return nil, errors.New("page is required")
	}
	if limit <= 0 {
		return nil, errors.New("limit is required")
	}
	params := url.Values{}
	params.Set("email", email)
	params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	params.Set("page", strconv.FormatInt(page, 10))
	params.Set("limit", strconv.FormatInt(limit, 10))
	var resp *SubAccountTransferLog
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, spotDefaultRate, &resp)
}

// GetManagedSubAccountFutureesAssetDetails retrieves managed sub account futures asset details(For Investor Master Account）(USER_DATA)
func (b *Binance) GetManagedSubAccountFutureesAssetDetails(ctx context.Context, email string) (*ManagedSubAccountFuturesAssetDetail, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	var resp *ManagedSubAccountFuturesAssetDetail
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/managed-subaccount/fetch-future-asset", params, spotDefaultRate, &resp)
}

// GetManagedSubAccountMarginAssetDetails retrieves managed sub-account margin asset details.
func (b *Binance) GetManagedSubAccountMarginAssetDetails(ctx context.Context, email string) (*SubAccountMarginAsset, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	params := url.Values{}
	params.Set("email", email)
	var resp *SubAccountMarginAsset
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/managed-subaccount/marginAsset", params, spotDefaultRate, &resp)
}

// FuturesTransferSubAccount transfers futures for sub-account( from master account only)
// 1: transfer from subaccount's spot account to its USDT-margined futures account 2: transfer from subaccount's USDT-margined futures account to its spot account
// 3: transfer from subaccount's spot account to its COIN-margined futures account 4:transfer from subaccount's COIN-margined futures account to its spot account
func (b *Binance) FuturesTransferSubAccount(ctx context.Context, email string, asset currency.Code, amount float64, transferType int64) (string, error) {
	return b.transferSubAccount(ctx, email, "/sapi/v1/sub-account/futures/transfer", asset, amount, transferType)
}

// MarginTransferForSubAccount margin Transfer for Sub-account (For Master Account)
// transferType: 1: transfer from subaccount's spot account to margin account 2: transfer from subaccount's margin account to its spot account
func (b *Binance) MarginTransferForSubAccount(ctx context.Context, email string, asset currency.Code, amount float64, transferType int64) (string, error) {
	return b.transferSubAccount(ctx, email, "/sapi/v1/sub-account/margin/transfer", asset, amount, transferType)
}

func (b *Binance) transferSubAccount(ctx context.Context, email, path string, asset currency.Code, amount float64, transferType int64) (string, error) {
	if !common.MatchesEmailPattern(email) {
		return "", errValidEmailRequired
	}
	if asset.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return "", order.ErrAmountBelowMin
	}
	if transferType != 1 && transferType != 2 && transferType != 3 && transferType != 4 {
		return "", errors.New("transfer type is required")
	}
	params := url.Values{}
	params.Set("email", email)
	params.Set("asset", asset.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	params.Set("type", strconv.FormatInt(transferType, 10))
	resp := struct {
		TransactionID string `json:"txnId"`
	}{}
	return resp.TransactionID, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, spotDefaultRate, &resp)
}

// TransferToSubAccountOfSameMaster Transfer to Sub-account of Same Master (For Sub-account)
func (b *Binance) TransferToSubAccountOfSameMaster(ctx context.Context, toEmail string, asset currency.Code, amount float64) (string, error) {
	if !common.MatchesEmailPattern(toEmail) {
		return "", errValidEmailRequired
	}
	if asset.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return "", order.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("toEmail", toEmail)
	params.Set("asset", asset.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	resp := &struct {
		TransactionID string `json:"txnId"`
	}{}
	return resp.TransactionID, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/transfer/subToSub", params, spotDefaultRate, &resp)
}

// FromSubAccountTransferToMaster Transfer to Master (For Sub-account)
// need to open Enable Spot & Margin Trading permission for the API Key which requests this endpoint.
func (b *Binance) FromSubAccountTransferToMaster(ctx context.Context, asset currency.Code, amount float64) (string, error) {
	if asset.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return "", order.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("asset", asset.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	resp := &struct {
		TransactionID string `json:"txnId"`
	}{}
	return resp.TransactionID, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/transfer/subToMaster", params, spotDefaultRate, &resp)
}

// SubAccountTransferHistory retrieves Sub-account Transfer History (For Sub-account)
func (b *Binance) SubAccountTransferHistory(ctx context.Context, asset currency.Code, transferType, limit int64, startTime, endTime time.Time) (*SubAccountTransferHistory, error) {
	params := url.Values{}
	if !asset.IsEmpty() {
		params.Set("asset", asset.String())
	}
	if transferType != 1 && transferType != 2 {
		params.Set("type", strconv.FormatInt(transferType, 10))
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
	var resp *SubAccountTransferHistory
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/transfer/subUserHistory", params, spotDefaultRate, &resp)
}

// SubAccountTransferHistoryForSubAccount represents a sub-account transfer history for sub accounts.
func (b *Binance) SubAccountTransferHistoryForSubAccount(ctx context.Context, asset currency.Code, transferType, limit int64, startTime, endTime time.Time, returnFailHistory bool) (*SubAccountTransferHistoryItem, error) {
	params := url.Values{}
	if !asset.IsEmpty() {
		params.Set("asset", asset.String())
	}
	if transferType != 0 {
		params.Set("type", strconv.FormatInt(transferType, 10))
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
	if returnFailHistory {
		params.Set("returnFailHistory", "true")
	}
	var resp *SubAccountTransferHistoryItem
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/transfer/subUserHistory", params, spotDefaultRate, &resp)
}

// UniversalTransferForMasterAccount submits a universal transfer using the master account.
func (b *Binance) UniversalTransferForMasterAccount(ctx context.Context, arg *UniversalTransferParams) (*UniversalTransferResponse, error) {
	if arg == nil || *arg == (UniversalTransferParams{}) {
		return nil, common.ErrNilPointer
	}
	if arg.FromAccountType == "" {
		return nil, fmt.Errorf("%w, fromAccountType=%s", errInvalidAccountType, arg.FromAccountType)
	}
	if arg.ToAccountType == "" {
		return nil, fmt.Errorf("%w, toAccountType = %s", errInvalidAccountType, arg.ToAccountType)
	}
	if arg.Asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountBelowMin
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
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodPost, "/sapi/v1/sub-account/universalTransfer", params, spotDefaultRate, &resp)
}

// GetUniversalTransferHistoryForMasterAccount retrieves universal transfer history for master account.
func (b *Binance) GetUniversalTransferHistoryForMasterAccount(ctx context.Context, fromEmail, toEmail, clientTransactionID string,
	startTime, endTime time.Time, page, limit int64) (*UniversalTransfersDetail, error) {
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
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *UniversalTransfersDetail
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/sub-account/universalTransfer", params, spotDefaultRate, &resp)
}

// GetDetailOnSubAccountsFuturesAccountV2 retrieves detail on sub-account's futures account V2 for master account
func (b *Binance) GetDetailOnSubAccountsFuturesAccountV2(ctx context.Context, email string, futuresType int64) (*MarginedFuturesAccount, error) {
	if !common.MatchesEmailPattern(email) {
		return nil, errValidEmailRequired
	}
	if futuresType != 0 {
		return nil, errInvalidFuturesType
	}
	params := url.Values{}
	var resp *MarginedFuturesAccount
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v2/sub-account/futures/account", params, spotDefaultRate, &resp)
}

// GetSummaryOfSubAccountsFuturesAccountV2 retrieves the summary of sub-account's futures account v2 for master account
func (b *Binance) GetSummaryOfSubAccountsFuturesAccountV2(ctx context.Context, futuresType, page, limit int64) (*AccountSummary, error) {
	if futuresType != 0 {
		return nil, errInvalidFuturesType
	}
	params := url.Values{}
	if page > 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *AccountSummary
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v2/sub-account/futures/accountSummary", params, spotDefaultRate, &resp)
}

// GetAccountStatus fetch account status detail.
func (b *Binance) GetAccountStatus(ctx context.Context) (string, error) {
	resp := &struct {
		Data string `json:"data"`
	}{}
	return resp.Data, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/account/status", nil, spotDefaultRate, &resp)
}

// GetAccountTradingAPIStatus fetch account api trading status detail.
func (b *Binance) GetAccountTradingAPIStatus(ctx context.Context) (*TradingAPIAccountStatus, error) {
	var resp *TradingAPIAccountStatus
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/account/apiTradingStatus", nil, spotDefaultRate, &resp)
}

// GetDustLog retrieves record of small or fractional amounts of assets that accumulate in a user's account
func (b *Binance) GetDustLog(ctx context.Context, accountType string, startTime, endTime time.Time) (*DustLog, error) {
	params := url.Values{}
	if accountType == "" {
		params.Set("accountType", accountType)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp *DustLog
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/asset/dribblet", params, spotDefaultRate, &resp)
}

// GetWsAuthStreamKey will retrieve a key to use for authorised WS streaming
func (b *Binance) GetWsAuthStreamKey(ctx context.Context) (string, error) {
	endpointPath, err := b.API.Endpoints.GetURL(exchange.RestSpotSupplementary)
	if err != nil {
		return "", err
	}

	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return "", err
	}

	var resp UserAccountStream
	headers := make(map[string]string)
	headers["X-MBX-APIKEY"] = creds.Key
	item := &request.Item{
		Method:        http.MethodPost,
		Path:          endpointPath + "/api/v3/userDataStream",
		Headers:       headers,
		Result:        &resp,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
	}

	err = b.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	}, request.AuthenticatedRequest)
	if err != nil {
		return "", err
	}
	return resp.ListenKey, nil
}

// MaintainWsAuthStreamKey will keep the key alive
func (b *Binance) MaintainWsAuthStreamKey(ctx context.Context) error {
	endpointPath, err := b.API.Endpoints.GetURL(exchange.RestSpotSupplementary)
	if err != nil {
		return err
	}
	if listenKey == "" {
		listenKey, err = b.GetWsAuthStreamKey(ctx)
		return err
	}

	creds, err := b.GetCredentials(ctx)
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
		Method:        http.MethodPut,
		Path:          path,
		Headers:       headers,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
	}

	return b.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		return item, nil
	}, request.AuthenticatedRequest)
}

// FetchExchangeLimits fetches order execution limits filtered by asset
func (b *Binance) FetchExchangeLimits(ctx context.Context, a asset.Item) ([]order.MinMaxLevel, error) {
	if a != asset.Spot && a != asset.Margin {
		return nil, fmt.Errorf("%w %v", asset.ErrNotSupported, a)
	}

	resp, err := b.GetExchangeInfo(ctx)
	if err != nil {
		return nil, err
	}

	aUpper := strings.ToUpper(a.String())

	limits := make([]order.MinMaxLevel, 0, len(resp.Symbols))
	for _, s := range resp.Symbols {
		var cp currency.Pair
		cp, err = currency.NewPairFromStrings(s.BaseAsset, s.QuoteAsset)
		if err != nil {
			return nil, err
		}

		if !slices.Contains(s.Permissions, aUpper) {
			continue
		}

		l := order.MinMaxLevel{
			Pair:  cp,
			Asset: a,
		}

		for _, f := range s.Filters {
			// TODO: Unhandled filters:
			// maxPosition, trailingDelta, percentPriceBySide, maxNumAlgoOrders
			switch f.FilterType {
			case priceFilter:
				l.MinPrice = f.MinPrice
				l.MaxPrice = f.MaxPrice
				l.PriceStepIncrementSize = f.TickSize
			case percentPriceFilter:
				l.MultiplierUp = f.MultiplierUp
				l.MultiplierDown = f.MultiplierDown
				l.AveragePriceMinutes = f.AvgPriceMinutes
			case lotSizeFilter:
				l.MaximumBaseAmount = f.MaxQty
				l.MinimumBaseAmount = f.MinQty
				l.AmountStepIncrementSize = f.StepSize
			case notionalFilter:
				l.MinNotional = f.MinNotional
			case icebergPartsFilter:
				l.MaxIcebergParts = f.Limit
			case marketLotSizeFilter:
				l.MarketMinQty = f.MinQty
				l.MarketMaxQty = f.MaxQty
				l.MarketStepIncrementSize = f.StepSize
			case maxNumOrdersFilter:
				l.MaxTotalOrders = f.MaxNumOrders
				l.MaxAlgoOrders = f.MaxNumAlgoOrders
			}
		}

		limits = append(limits, l)
	}
	return limits, nil
}

// CryptoLoanIncomeHistory returns crypto loan income history
func (b *Binance) CryptoLoanIncomeHistory(ctx context.Context, curr currency.Code, loanType string, startTime, endTime time.Time, limit int64) ([]CryptoLoansIncomeHistory, error) {
	params := url.Values{}
	if !curr.IsEmpty() {
		params.Set("asset", curr.String())
	}
	if loanType != "" {
		params.Set("type", loanType)
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []CryptoLoansIncomeHistory
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/income", params, spotDefaultRate, &resp)
}

// CryptoLoanBorrow borrows crypto
func (b *Binance) CryptoLoanBorrow(ctx context.Context, loanCoin currency.Code, loanAmount float64, collateralCoin currency.Code, collateralAmount float64, loanTerm int64) ([]CryptoLoanBorrow, error) {
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
		return nil, errEitherLoanOrCollateralAmountsMustBeSet
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
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/loan/borrow", params, spotDefaultRate, &resp)
}

// CryptoLoanBorrowHistory gets loan borrow history
func (b *Binance) CryptoLoanBorrowHistory(ctx context.Context, orderID int64, loanCoin, collateralCoin currency.Code, startTime, endTime time.Time, current, limit int64) (*LoanBorrowHistory, error) {
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
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if current != 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}

	var resp *LoanBorrowHistory
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/borrow/history", params, spotDefaultRate, &resp)
}

// CryptoLoanOngoingOrders obtains ongoing loan orders
func (b *Binance) CryptoLoanOngoingOrders(ctx context.Context, orderID int64, loanCoin, collateralCoin currency.Code, current, limit int64) (*CryptoLoanOngoingOrder, error) {
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
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/ongoing/orders", params, spotDefaultRate, &resp)
}

// CryptoLoanRepay repays a crypto loan
func (b *Binance) CryptoLoanRepay(ctx context.Context, orderID int64, amount float64, repayType int64, collateralReturn bool) ([]CryptoLoanRepay, error) {
	if orderID <= 0 {
		return nil, errOrderIDMustBeSet
	}
	if amount <= 0 {
		return nil, errAmountMustBeSet
	}
	params := url.Values{}
	params.Set("orderId", strconv.FormatInt(orderID, 10))
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	if repayType != 0 {
		params.Set("type", strconv.FormatInt(repayType, 10))
	}
	params.Set("collateralReturn", strconv.FormatBool(collateralReturn))
	var resp []CryptoLoanRepay
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/loan/repay", params, spotDefaultRate, &resp)
}

// CryptoLoanRepaymentHistory gets the crypto loan repayment history
func (b *Binance) CryptoLoanRepaymentHistory(ctx context.Context, orderID int64, loanCoin, collateralCoin currency.Code, startTime, endTime time.Time, current, limit int64) (*CryptoLoanRepayHistory, error) {
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
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if current != 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *CryptoLoanRepayHistory
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/repay/history", params, spotDefaultRate, &resp)
}

// CryptoLoanAdjustLTV adjusts the LTV of a crypto loan
func (b *Binance) CryptoLoanAdjustLTV(ctx context.Context, orderID int64, reduce bool, amount float64) (*CryptoLoanAdjustLTV, error) {
	if orderID <= 0 {
		return nil, errOrderIDMustBeSet
	}
	if amount <= 0 {
		return nil, errAmountMustBeSet
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
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/loan/adjust/ltv", params, spotDefaultRate, &resp)
}

// CryptoLoanLTVAdjustmentHistory gets the crypto loan LTV adjustment history
func (b *Binance) CryptoLoanLTVAdjustmentHistory(ctx context.Context, orderID int64, loanCoin, collateralCoin currency.Code, startTime, endTime time.Time, current, limit int64) (*CryptoLoanLTVAdjustmentHistory, error) {
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
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if current != 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *CryptoLoanLTVAdjustmentHistory
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/ltv/adjustment/history", params, spotDefaultRate, &resp)
}

// CryptoLoanAssetsData gets the loanable assets data
func (b *Binance) CryptoLoanAssetsData(ctx context.Context, loanCoin currency.Code, vipLevel int64) (*LoanableAssetsData, error) {
	params := url.Values{}
	if !loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.String())
	}
	if vipLevel != 0 {
		params.Set("vipLevel", strconv.FormatInt(vipLevel, 10))
	}
	var resp *LoanableAssetsData
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/loanable/data", params, spotDefaultRate, &resp)
}

// CryptoLoanCollateralAssetsData gets the collateral assets data
func (b *Binance) CryptoLoanCollateralAssetsData(ctx context.Context, collateralCoin currency.Code, vipLevel int64) (*CollateralAssetData, error) {
	params := url.Values{}
	if !collateralCoin.IsEmpty() {
		params.Set("collateralCoin", collateralCoin.String())
	}
	if vipLevel != 0 {
		params.Set("vipLevel", strconv.FormatInt(vipLevel, 10))
	}
	var resp *CollateralAssetData
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/collateral/data", params, spotDefaultRate, &resp)
}

// CryptoLoanCheckCollateralRepayRate checks the collateral repay rate
func (b *Binance) CryptoLoanCheckCollateralRepayRate(ctx context.Context, loanCoin, collateralCoin currency.Code, amount float64) (*CollateralRepayRate, error) {
	if loanCoin.IsEmpty() {
		return nil, errLoanCoinMustBeSet
	}
	if collateralCoin.IsEmpty() {
		return nil, errCollateralCoinMustBeSet
	}
	if amount <= 0 {
		return nil, errAmountMustBeSet
	}
	params := url.Values{}
	params.Set("loanCoin", loanCoin.String())
	params.Set("collateralCoin", collateralCoin.String())
	params.Set("repayAmount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *CollateralRepayRate
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/repay/collateral/rate", params, spotDefaultRate, &resp)
}

// CryptoLoanCustomiseMarginCall customises a loan's margin call
func (b *Binance) CryptoLoanCustomiseMarginCall(ctx context.Context, orderID int64, collateralCoin currency.Code, marginCallValue float64) (*CustomiseMarginCall, error) {
	if marginCallValue <= 0 {
		return nil, errors.New("marginCallValue must not be <= 0")
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
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/loan/customize/margin_call", params, spotDefaultRate, &resp)
}

// FlexibleLoanBorrow creates a flexible loan
func (b *Binance) FlexibleLoanBorrow(ctx context.Context, loanCoin, collateralCoin currency.Code, loanAmount, collateralAmount float64) (*FlexibleLoanBorrow, error) {
	if loanCoin.IsEmpty() {
		return nil, errLoanCoinMustBeSet
	}
	if collateralCoin.IsEmpty() {
		return nil, errCollateralCoinMustBeSet
	}
	if loanAmount == 0 && collateralAmount == 0 {
		return nil, errEitherLoanOrCollateralAmountsMustBeSet
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
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/loan/flexible/borrow", params, spotDefaultRate, &resp)
}

// FlexibleLoanOngoingOrders gets the flexible loan ongoing orders
func (b *Binance) FlexibleLoanOngoingOrders(ctx context.Context, loanCoin, collateralCoin currency.Code, current, limit int64) (*FlexibleLoanOngoingOrder, error) {
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
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/flexible/ongoing/orders", params, spotDefaultRate, &resp)
}

// FlexibleLoanBorrowHistory gets the flexible loan borrow history
func (b *Binance) FlexibleLoanBorrowHistory(ctx context.Context, loanCoin, collateralCoin currency.Code, startTime, endTime time.Time, current, limit int64) (*FlexibleLoanBorrowHistory, error) {
	params := url.Values{}
	if !loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.String())
	}
	if !collateralCoin.IsEmpty() {
		params.Set("collateralCoin", collateralCoin.String())
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if current != 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}

	var resp *FlexibleLoanBorrowHistory
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/flexible/borrow/history", params, spotDefaultRate, &resp)
}

// FlexibleLoanRepay repays a flexible loan
func (b *Binance) FlexibleLoanRepay(ctx context.Context, loanCoin, collateralCoin currency.Code, amount float64, collateralReturn, fullRepayment bool) (*FlexibleLoanRepay, error) {
	if loanCoin.IsEmpty() {
		return nil, errLoanCoinMustBeSet
	}
	if collateralCoin.IsEmpty() {
		return nil, errCollateralCoinMustBeSet
	}
	if amount <= 0 {
		return nil, errAmountMustBeSet
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
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/loan/flexible/repay", params, spotDefaultRate, &resp)
}

// FlexibleLoanRepayHistory gets the flexible loan repayment history
func (b *Binance) FlexibleLoanRepayHistory(ctx context.Context, loanCoin, collateralCoin currency.Code, startTime, endTime time.Time, current, limit int64) (*FlexibleLoanRepayHistory, error) {
	params := url.Values{}
	if !loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.String())
	}
	if !collateralCoin.IsEmpty() {
		params.Set("collateralCoin", collateralCoin.String())
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if current != 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}

	var resp *FlexibleLoanRepayHistory
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/flexible/repay/history", params, spotDefaultRate, &resp)
}

// FlexibleLoanAdjustLTV adjusts the LTV of a flexible loan
func (b *Binance) FlexibleLoanAdjustLTV(ctx context.Context, loanCoin, collateralCoin currency.Code, amount float64, reduce bool) (*FlexibleLoanAdjustLTV, error) {
	if loanCoin.IsEmpty() {
		return nil, errLoanCoinMustBeSet
	}
	if collateralCoin.IsEmpty() {
		return nil, errCollateralCoinMustBeSet
	}
	if amount <= 0 {
		return nil, errAmountMustBeSet
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
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/loan/flexible/adjust/ltv", params, spotDefaultRate, &resp)
}

// FlexibleLoanLTVAdjustmentHistory gets the flexible loan LTV adjustment history
func (b *Binance) FlexibleLoanLTVAdjustmentHistory(ctx context.Context, loanCoin, collateralCoin currency.Code, startTime, endTime time.Time, current, limit int64) (*FlexibleLoanLTVAdjustmentHistory, error) {
	params := url.Values{}
	if !loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.String())
	}
	if !collateralCoin.IsEmpty() {
		params.Set("collateralCoin", collateralCoin.String())
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if current != 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}

	var resp *FlexibleLoanLTVAdjustmentHistory
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/flexible/ltv/adjustment/history", params, spotDefaultRate, &resp)
}

// FlexibleLoanAssetsData gets the flexible loan assets data
func (b *Binance) FlexibleLoanAssetsData(ctx context.Context, loanCoin currency.Code) (*FlexibleLoanAssetsData, error) {
	params := url.Values{}
	if !loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.String())
	}
	var resp *FlexibleLoanAssetsData
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/flexible/loanable/data", params, spotDefaultRate, &resp)
}

// FlexibleCollateralAssetsData gets the flexible loan collateral assets data
func (b *Binance) FlexibleCollateralAssetsData(ctx context.Context, collateralCoin currency.Code) (*FlexibleCollateralAssetsData, error) {
	params := url.Values{}
	if !collateralCoin.IsEmpty() {
		params.Set("collateralCoin", collateralCoin.String())
	}
	var resp *FlexibleCollateralAssetsData
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/loan/flexible/collateral/data", params, spotDefaultRate, &resp)
}
