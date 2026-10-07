package binance

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/orderbookmanager"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with Binance
type Exchange struct {
	exchange.Base
	userData userDataStreams
	// orderbookSync synchronises every asset's websocket diff depth with REST snapshots
	orderbookSync *orderbookmanager.UpdateManager
}

const (
	apiURL         = "https://api.binance.com"
	spotAPIURL     = "https://sapi.binance.com"
	cfuturesAPIURL = "https://dapi.binance.com"
	ufuturesAPIURL = "https://fapi.binance.com"
	eOptionAPIURL  = "https://eapi.binance.com"
	pMarginAPIURL  = "https://papi.binance.com"
	tradeBaseURL   = "https://www.binance.com/en/"

	defaultRecvWindow = 5 * time.Second
)

var (
	errInvalidParameterCombination   = errors.New("invalid combination of optional parameters")
	errAggregatedTradesStartRequired = errors.New("fromID or start time required to collect more than 1000 aggregate trades")
	errInvalidWindowSize             = errors.New("invalid window size")
)

// GetExchangeInfo returns the current exchange trading rules and symbol information. A nil request returns every
// symbol with SPOT, MARGIN or LEVERAGED permission
func (e *Exchange) GetExchangeInfo(ctx context.Context, req *ExchangeInfoRequest) (*ExchangeInfoResponse, error) {
	params := url.Values{}
	if req != nil {
		if len(req.Symbols) > 0 && (len(req.Permissions) > 0 || req.SymbolStatus != "") {
			return nil, fmt.Errorf("%w: permissions and symbolStatus cannot be sent with symbols", errInvalidParameterCombination)
		}
		if err := e.setSymbolParams(params, req.Symbols); err != nil {
			return nil, err
		}
		switch len(req.Permissions) {
		case 0:
		case 1:
			params.Set("permissions", req.Permissions[0])
		default:
			permissions, err := json.Marshal(req.Permissions)
			if err != nil {
				return nil, err
			}
			params.Set("permissions", string(permissions))
		}
		if req.ShowPermissionSets != nil {
			params.Set("showPermissionSets", strconv.FormatBool(*req.ShowPermissionSets))
		}
		if req.SymbolStatus != "" {
			params.Set("symbolStatus", req.SymbolStatus)
		}
	}
	var resp *ExchangeInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues("/api/v3/exchangeInfo", params), spotExchangeInfoRate, &resp)
}

// GetExchangeServerTime checks the server time
func (e *Exchange) GetExchangeServerTime(ctx context.Context) (*ServerTimeResponse, error) {
	var resp *ServerTimeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, "/api/v3/time", spotDefaultRate, &resp)
}

// GetAggregatedTrades returns the compressed/aggregate trades list. A limit above the endpoint's maximum of 1000 is
// collected over several requests, paging on from FromID or StartTime until the limit or EndTime is reached
func (e *Exchange) GetAggregatedTrades(ctx context.Context, req *AggregatedTradeRequest) ([]AggregatedTrade, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.FromID != 0 && (!req.StartTime.IsZero() || !req.EndTime.IsZero()) {
		return nil, fmt.Errorf("%w: fromId cannot be sent with startTime or endTime", errInvalidParameterCombination)
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if req.FromID != 0 {
		params.Set("fromId", strconv.FormatUint(req.FromID, 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit > 1000 {
		if req.FromID == 0 && req.StartTime.IsZero() {
			return nil, errAggregatedTradesStartRequired
		}
		return e.batchAggregateTrades(ctx, req, params)
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []AggregatedTrade
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues("/api/v3/aggTrades", params), aggTradesRate, &resp)
}

// batchAggregateTrades collects aggregate trades a thousand at a time. The first page starts at FromID or StartTime
// and each later page starts at the last trade collected, which the endpoint returns again since fromId is inclusive
func (e *Exchange) batchAggregateTrades(ctx context.Context, req *AggregatedTradeRequest, params url.Values) ([]AggregatedTrade, error) {
	params.Set("limit", "1000")
	var trades []AggregatedTrade
	for uint64(len(trades)) < req.Limit {
		var page []AggregatedTrade
		if err := e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues("/api/v3/aggTrades", params), aggTradesRate, &page); err != nil {
			return nil, fmt.Errorf("error fetching aggregate trades for %s: %w", req.Symbol, err)
		}
		if len(trades) > 0 && len(page) > 0 && page[0].AggregateTradeID == trades[len(trades)-1].AggregateTradeID {
			page = page[1:]
		}
		if !req.EndTime.IsZero() {
			// Pages requested by fromId run past EndTime, which only the first request sends
			if afterEnd := sort.Search(len(page), func(i int) bool {
				return page[i].Timestamp.Time().After(req.EndTime)
			}); afterEnd < len(page) {
				trades = append(trades, page[:afterEnd]...)
				break
			}
		}
		if len(page) == 0 {
			break
		}
		trades = append(trades, page...)
		params.Del("startTime")
		params.Del("endTime")
		params.Set("fromId", strconv.FormatUint(trades[len(trades)-1].AggregateTradeID, 10))
	}
	if uint64(len(trades)) > req.Limit {
		trades = trades[:req.Limit]
	}
	return trades, nil
}

// GetAveragePrice returns the current average price for a symbol
func (e *Exchange) GetAveragePrice(ctx context.Context, symbol currency.Pair) (*AveragePriceResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	var resp *AveragePriceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues("/api/v3/avgPrice", params), getCurrentAveragePriceRate, &resp)
}

// GetOrderBook returns the order book of a symbol. A limit of zero returns the default 100 levels per side, and a limit
// above 5000 returns 5000
func (e *Exchange) GetOrderBook(ctx context.Context, req *OrderBookRequest) (*OrderBookResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.SymbolStatus != "" {
		params.Set("symbolStatus", req.SymbolStatus)
	}
	var resp *OrderBookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues("/api/v3/depth", params), spotOrderbookLimit(req.Limit), &resp)
}

// GetMostRecentTrades returns the recent trades list of a symbol
func (e *Exchange) GetMostRecentTrades(ctx context.Context, req *RecentTradeRequest) ([]RecentTrade, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []RecentTrade
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues("/api/v3/trades", params), getRecentTradesListRate, &resp)
}

// GetHistoricalTrades looks up older trades of a symbol, starting from fromID when it is set and otherwise returning the
// most recent trades
func (e *Exchange) GetHistoricalTrades(ctx context.Context, symbol currency.Pair, limit, fromID uint64) ([]RecentTrade, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if fromID > 0 {
		params.Set("fromId", strconv.FormatUint(fromID, 10))
	}
	var resp []RecentTrade
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues("/api/v3/historicalTrades", params), getOldTradeLookupRate, &resp)
}

// GetSpotKline returns the kline/candlestick bars of a symbol
func (e *Exchange) GetSpotKline(ctx context.Context, req *KlinesRequest) ([]CandleStick, error) {
	return e.getKlines(ctx, req, "/api/v3/klines")
}

// GetTickerData returns rolling window price change statistics. The window's open time starts on a minute while its
// close time is the time of the request, so the window can be up to 59999ms wider than the requested WindowSize
func (e *Exchange) GetTickerData(ctx context.Context, req *RollingWindowTickerRequest) ([]TickerStatistics, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if len(req.Symbols) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	params := url.Values{}
	if err := e.setSymbolParams(params, req.Symbols); err != nil {
		return nil, err
	}
	if req.WindowSize != 0 {
		windowSize, err := formatWindowSize(req.WindowSize)
		if err != nil {
			return nil, err
		}
		params.Set("windowSize", windowSize)
	}
	if req.Type != "" {
		params.Set("type", req.Type)
	}
	if req.SymbolStatus != "" {
		params.Set("symbolStatus", req.SymbolStatus)
	}
	var resp objectOrArray[TickerStatistics]
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues("/api/v3/ticker", params), spotTickerSymbolsLimit(len(req.Symbols)), &resp)
}

// formatWindowSize renders a window size in the largest whole unit of days, hours or minutes, since units cannot be
// combined
func formatWindowSize(windowSize time.Duration) (string, error) {
	switch {
	case windowSize <= 0 || windowSize%time.Minute != 0:
		return "", fmt.Errorf("%w: %s is not a whole number of minutes", errInvalidWindowSize, windowSize)
	case windowSize%(24*time.Hour) == 0:
		return strconv.FormatInt(int64(windowSize/(24*time.Hour)), 10) + "d", nil
	case windowSize%time.Hour == 0:
		return strconv.FormatInt(int64(windowSize/time.Hour), 10) + "h", nil
	default:
		return strconv.FormatInt(int64(windowSize/time.Minute), 10) + "m", nil
	}
}

// GetPriceChangeStats returns 24hr ticker price change statistics. A request without symbols returns every symbol, at
// a much higher weight
func (e *Exchange) GetPriceChangeStats(ctx context.Context, req *PriceChangeStatsRequest) ([]PriceChangeStats, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := e.setSymbolParams(params, req.Symbols); err != nil {
		return nil, err
	}
	if req.Type != "" {
		params.Set("type", req.Type)
	}
	if req.SymbolStatus != "" {
		params.Set("symbolStatus", req.SymbolStatus)
	}
	var resp objectOrArray[PriceChangeStats]
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues("/api/v3/ticker/24hr", params), spotTicker24HourLimit(len(req.Symbols)), &resp)
}

// GetBestPrice returns the symbol order book ticker: the best bid and ask price and quantity of each symbol, or of
// every symbol when none are given
func (e *Exchange) GetBestPrice(ctx context.Context, symbols currency.Pairs, symbolStatus string) ([]BookTicker, error) {
	params := url.Values{}
	if err := e.setSymbolParams(params, symbols); err != nil {
		return nil, err
	}
	if symbolStatus != "" {
		params.Set("symbolStatus", symbolStatus)
	}
	rateLimit := spotOrderbookTickerAllRate
	if len(symbols) == 1 {
		rateLimit = spotBookTickerRate
	}
	var resp objectOrArray[BookTicker]
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues("/api/v3/ticker/bookTicker", params), rateLimit, &resp)
}

// GetLatestSpotPrice returns the symbol price ticker: the latest price of each symbol, or of every symbol when none are
// given
func (e *Exchange) GetLatestSpotPrice(ctx context.Context, symbols currency.Pairs, symbolStatus string) ([]SymbolPrice, error) {
	params := url.Values{}
	if err := e.setSymbolParams(params, symbols); err != nil {
		return nil, err
	}
	if symbolStatus != "" {
		params.Set("symbolStatus", symbolStatus)
	}
	rateLimit := spotSymbolPriceAllRate
	if len(symbols) == 1 {
		rateLimit = spotSymbolPriceRate
	}
	var resp objectOrArray[SymbolPrice]
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues("/api/v3/ticker/price", params), rateLimit, &resp)
}

// GetTradingDayTicker returns the price change statistics of the current trading day for each symbol
func (e *Exchange) GetTradingDayTicker(ctx context.Context, req *TradingDayTickerRequest) ([]TickerStatistics, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if len(req.Symbols) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	params := url.Values{}
	if err := e.setSymbolParams(params, req.Symbols); err != nil {
		return nil, err
	}
	if req.TimeZone != "" {
		params.Set("timeZone", req.TimeZone)
	}
	if req.Type != "" {
		params.Set("type", req.Type)
	}
	if req.SymbolStatus != "" {
		params.Set("symbolStatus", req.SymbolStatus)
	}
	var resp objectOrArray[TickerStatistics]
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues("/api/v3/ticker/tradingDay", params), spotTickerSymbolsLimit(len(req.Symbols)), &resp)
}

// GetUIKline returns UIKlines: kline data modified for presentation in candlestick charts
func (e *Exchange) GetUIKline(ctx context.Context, req *KlinesRequest) ([]CandleStick, error) {
	return e.getKlines(ctx, req, "/api/v3/uiKlines")
}

// getKlines sends a klines or uiKlines request, which take the same parameters and return the same response
func (e *Exchange) getKlines(ctx context.Context, req *KlinesRequest, path string) ([]CandleStick, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.Interval == "" {
		return nil, kline.ErrInvalidInterval
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", req.Interval)
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.TimeZone != "" {
		params.Set("timeZone", req.TimeZone)
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []CandleStick
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues(path, params), getKlineRate, &resp)
}

// setSymbolParams sends one pair as symbol and several as symbols, a JSON array, the way the exchange information and
// ticker endpoints take them; no pairs leaves both unset
func (e *Exchange) setSymbolParams(params url.Values, pairs currency.Pairs) error {
	symbols := make([]string, len(pairs))
	for i := range pairs {
		if pairs[i].IsEmpty() {
			return currency.ErrCurrencyPairEmpty
		}
		var err error
		if symbols[i], err = e.FormatSymbol(pairs[i], asset.Spot); err != nil {
			return err
		}
	}
	switch len(symbols) {
	case 0:
	case 1:
		params.Set("symbol", symbols[0])
	default:
		list, err := json.Marshal(symbols)
		if err != nil {
			return err
		}
		params.Set("symbols", string(list))
	}
	return nil
}

// CancelAllOpenOrderOnSymbol cancels all open orders on a symbol, including orders that are part of an order list
func (e *Exchange) CancelAllOpenOrderOnSymbol(ctx context.Context, symbol currency.Pair) ([]CancelledOpenOrder, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	var resp []CancelledOpenOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodDelete, "/api/v3/openOrders", params, spotDefaultRate, &resp)
}

// NewOrder sends in a new order
func (e *Exchange) NewOrder(ctx context.Context, req *NewOrderRequest) (*NewOrderResponse, error) {
	params, err := e.newOrderParams(req)
	if err != nil {
		return nil, err
	}
	var resp *NewOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/api/v3/order", params, spotOrderRate, &resp)
}

// newOrderParams validates a new order and builds its parameters, which a test order takes too
func (e *Exchange) newOrderParams(req *NewOrderRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if req.Type == "" {
		return nil, order.ErrTypeIsInvalid
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", req.Side)
	params.Set("type", req.Type)
	if req.TimeInForce != "" {
		params.Set("timeInForce", req.TimeInForce)
	}
	if req.Quantity > 0 {
		params.Set("quantity", strconv.FormatFloat(req.Quantity, 'f', -1, 64))
	}
	if req.QuoteOrderQuantity > 0 {
		params.Set("quoteOrderQty", strconv.FormatFloat(req.QuoteOrderQuantity, 'f', -1, 64))
	}
	if req.Price > 0 {
		params.Set("price", strconv.FormatFloat(req.Price, 'f', -1, 64))
	}
	if req.NewClientOrderID != "" {
		params.Set("newClientOrderId", req.NewClientOrderID)
	}
	if req.StrategyID != 0 {
		params.Set("strategyId", strconv.FormatUint(req.StrategyID, 10))
	}
	if req.StrategyType != 0 {
		params.Set("strategyType", strconv.FormatUint(req.StrategyType, 10))
	}
	if req.StopPrice > 0 {
		params.Set("stopPrice", strconv.FormatFloat(req.StopPrice, 'f', -1, 64))
	}
	if req.TrailingDelta != 0 {
		params.Set("trailingDelta", strconv.FormatUint(req.TrailingDelta, 10))
	}
	if req.IcebergQuantity > 0 {
		params.Set("icebergQty", strconv.FormatFloat(req.IcebergQuantity, 'f', -1, 64))
	}
	if req.NewOrderResponseType != "" {
		params.Set("newOrderRespType", req.NewOrderResponseType)
	}
	if req.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", req.SelfTradePreventionMode)
	}
	if req.PegPriceType != "" {
		params.Set("pegPriceType", req.PegPriceType)
	}
	if req.PegOffsetValue != 0 {
		params.Set("pegOffsetValue", strconv.FormatUint(req.PegOffsetValue, 10))
	}
	if req.PegOffsetType != "" {
		params.Set("pegOffsetType", req.PegOffsetType)
	}
	return params, nil
}

// CancelExistingOrder cancels an active order
func (e *Exchange) CancelExistingOrder(ctx context.Context, req *CancelOrderRequest) (*CancelOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.OrderID == 0 && req.OriginalClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if req.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	}
	if req.OriginalClientOrderID != "" {
		params.Set("origClientOrderId", req.OriginalClientOrderID)
	}
	if req.NewClientOrderID != "" {
		params.Set("newClientOrderId", req.NewClientOrderID)
	}
	if req.CancelRestrictions != "" {
		params.Set("cancelRestrictions", req.CancelRestrictions)
	}
	var resp *CancelOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodDelete, "/api/v3/order", params, spotDefaultRate, &resp)
}

// CancelOCOOrder cancels an entire order list. Cancelling any order of a list cancels the whole list
func (e *Exchange) CancelOCOOrder(ctx context.Context, req *CancelOrderListRequest) (*OCOOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.OrderListID == 0 && req.ListClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if req.OrderListID != 0 {
		params.Set("orderListId", strconv.FormatUint(req.OrderListID, 10))
	}
	if req.ListClientOrderID != "" {
		params.Set("listClientOrderId", req.ListClientOrderID)
	}
	if req.NewClientOrderID != "" {
		params.Set("newClientOrderId", req.NewClientOrderID)
	}
	var resp *OCOOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodDelete, "/api/v3/orderList", params, spotDefaultRate, &resp)
}

// CancelExistingOrderAndSendNewOrder cancels an existing order and places a new order on the same symbol. Filters and
// the unfilled order count are evaluated before either happens, and a new order that is not attempted still adds one
// to the unfilled order count
func (e *Exchange) CancelExistingOrderAndSendNewOrder(ctx context.Context, req *CancelReplaceOrderRequest) (*CancelAndReplaceResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if req.Type == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if req.CancelReplaceMode == "" {
		return nil, errCancelReplaceModeRequired
	}
	if req.CancelOrderID == 0 && req.CancelOriginalClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", req.Side)
	params.Set("type", req.Type)
	params.Set("cancelReplaceMode", req.CancelReplaceMode)
	if req.TimeInForce != "" {
		params.Set("timeInForce", req.TimeInForce)
	}
	if req.Quantity > 0 {
		params.Set("quantity", strconv.FormatFloat(req.Quantity, 'f', -1, 64))
	}
	if req.QuoteOrderQuantity > 0 {
		params.Set("quoteOrderQty", strconv.FormatFloat(req.QuoteOrderQuantity, 'f', -1, 64))
	}
	if req.Price > 0 {
		params.Set("price", strconv.FormatFloat(req.Price, 'f', -1, 64))
	}
	if req.CancelNewClientOrderID != "" {
		params.Set("cancelNewClientOrderId", req.CancelNewClientOrderID)
	}
	if req.CancelOriginalClientOrderID != "" {
		params.Set("cancelOrigClientOrderId", req.CancelOriginalClientOrderID)
	}
	if req.CancelOrderID != 0 {
		params.Set("cancelOrderId", strconv.FormatUint(req.CancelOrderID, 10))
	}
	if req.NewClientOrderID != "" {
		params.Set("newClientOrderId", req.NewClientOrderID)
	}
	if req.StrategyID != 0 {
		params.Set("strategyId", strconv.FormatUint(req.StrategyID, 10))
	}
	if req.StrategyType != 0 {
		params.Set("strategyType", strconv.FormatUint(req.StrategyType, 10))
	}
	if req.StopPrice > 0 {
		params.Set("stopPrice", strconv.FormatFloat(req.StopPrice, 'f', -1, 64))
	}
	if req.TrailingDelta != 0 {
		params.Set("trailingDelta", strconv.FormatUint(req.TrailingDelta, 10))
	}
	if req.IcebergQuantity > 0 {
		params.Set("icebergQty", strconv.FormatFloat(req.IcebergQuantity, 'f', -1, 64))
	}
	if req.NewOrderResponseType != "" {
		params.Set("newOrderRespType", req.NewOrderResponseType)
	}
	if req.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", req.SelfTradePreventionMode)
	}
	if req.CancelRestrictions != "" {
		params.Set("cancelRestrictions", req.CancelRestrictions)
	}
	if req.OrderRateLimitExceededMode != "" {
		params.Set("orderRateLimitExceededMode", req.OrderRateLimitExceededMode)
	}
	if req.PegPriceType != "" {
		params.Set("pegPriceType", req.PegPriceType)
	}
	if req.PegOffsetValue != 0 {
		params.Set("pegOffsetValue", strconv.FormatUint(req.PegOffsetValue, 10))
	}
	if req.PegOffsetType != "" {
		params.Set("pegOffsetType", req.PegOffsetType)
	}
	var resp *CancelAndReplaceResponse
	err = e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/api/v3/order/cancelReplace", params, spotOrderRate, &resp)
	return cancelReplaceOutcome(resp, err)
}

// cancelReplaceOutcome returns a cancel-replace's outcome with its error. When a half fails, Binance puts the outcome
// of both halves in the error's data, which the caller needs to learn whether a new order was still placed
func cancelReplaceOutcome[T any](resp *T, err error) (*T, error) {
	var apiErr *APIError
	if !errors.As(err, &apiErr) || len(apiErr.Data) == 0 {
		return resp, err
	}
	if decodeErr := json.Unmarshal(apiErr.Data, &resp); decodeErr != nil {
		return nil, common.AppendError(err, fmt.Errorf("error decoding cancel-replace outcome: %w", decodeErr))
	}
	return resp, err
}

// NewOCOOrderList sends in a one-cancels-the-other (OCO) order list of an above and a below order, where the
// activation of one order immediately cancels the other. One order must be LIMIT_MAKER, TAKE_PROFIT or
// TAKE_PROFIT_LIMIT and the other STOP_LOSS or STOP_LOSS_LIMIT, and an OCO counts as two orders against the order rate
// limit
func (e *Exchange) NewOCOOrderList(ctx context.Context, req *OCOOrderListRequest) (*OCOOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if req.Quantity <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if req.Above.Type == "" {
		return nil, fmt.Errorf("%w: aboveType is required", order.ErrTypeIsInvalid)
	}
	if req.Below.Type == "" {
		return nil, fmt.Errorf("%w: belowType is required", order.ErrTypeIsInvalid)
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", req.Side)
	params.Set("quantity", strconv.FormatFloat(req.Quantity, 'f', -1, 64))
	if req.ListClientOrderID != "" {
		params.Set("listClientOrderId", req.ListClientOrderID)
	}
	setOCOLegParams(params, "above", &req.Above)
	setOCOLegParams(params, "below", &req.Below)
	if req.NewOrderResponseType != "" {
		params.Set("newOrderRespType", req.NewOrderResponseType)
	}
	if req.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", req.SelfTradePreventionMode)
	}
	var resp *OCOOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/api/v3/orderList/oco", params, spotOCOOrderRate, &resp)
}

// setOCOLegParams sets the parameters of an order list's above or below order, which carry the leg's name as a prefix
func setOCOLegParams(params url.Values, leg string, arg *OCOOrderListLeg) {
	params.Set(leg+"Type", arg.Type)
	if arg.ClientOrderID != "" {
		params.Set(leg+"ClientOrderId", arg.ClientOrderID)
	}
	if arg.IcebergQuantity != 0 {
		params.Set(leg+"IcebergQty", strconv.FormatUint(arg.IcebergQuantity, 10))
	}
	if arg.Price > 0 {
		params.Set(leg+"Price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	}
	if arg.StopPrice > 0 {
		params.Set(leg+"StopPrice", strconv.FormatFloat(arg.StopPrice, 'f', -1, 64))
	}
	if arg.TrailingDelta != 0 {
		params.Set(leg+"TrailingDelta", strconv.FormatUint(arg.TrailingDelta, 10))
	}
	if arg.TimeInForce != "" {
		params.Set(leg+"TimeInForce", arg.TimeInForce)
	}
	if arg.StrategyID != 0 {
		params.Set(leg+"StrategyId", strconv.FormatUint(arg.StrategyID, 10))
	}
	if arg.StrategyType != 0 {
		params.Set(leg+"StrategyType", strconv.FormatUint(arg.StrategyType, 10))
	}
	if arg.PegPriceType != "" {
		params.Set(leg+"PegPriceType", arg.PegPriceType)
	}
	if arg.PegOffsetType != "" {
		params.Set(leg+"PegOffsetType", arg.PegOffsetType)
	}
	if arg.PegOffsetValue != 0 {
		params.Set(leg+"PegOffsetValue", strconv.FormatUint(arg.PegOffsetValue, 10))
	}
}

// NewOrderTest tests new order creation and the request's signature and recvWindow without sending the order to the
// matching engine. With computeCommissionRates the response holds the commission rates the order would pay
func (e *Exchange) NewOrderTest(ctx context.Context, req *NewOrderRequest, computeCommissionRates bool) (*OrderTestResponse, error) {
	params, err := e.newOrderParams(req)
	if err != nil {
		return nil, err
	}
	rateLimit := spotDefaultRate
	if computeCommissionRates {
		params.Set("computeCommissionRates", "true")
		rateLimit = testNewOrderWithCommissionRate
	}
	var resp *OrderTestResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/api/v3/order/test", params, rateLimit, &resp)
}

// NewOrderUsingSOR places an order using smart order routing (SOR)
func (e *Exchange) NewOrderUsingSOR(ctx context.Context, req *SOROrderRequest) (*SOROrderResponse, error) {
	params, err := e.sorOrderParams(req)
	if err != nil {
		return nil, err
	}
	var resp *SOROrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/api/v3/sor/order", params, spotOrderRate, &resp)
}

// NewOrderUsingSORTest tests new order creation and the request's signature and recvWindow using smart order routing
// (SOR) without sending the order to the matching engine. With computeCommissionRates the response holds the commission
// rates the order would pay
func (e *Exchange) NewOrderUsingSORTest(ctx context.Context, req *SOROrderRequest, computeCommissionRates bool) (*OrderTestResponse, error) {
	params, err := e.sorOrderParams(req)
	if err != nil {
		return nil, err
	}
	rateLimit := spotDefaultRate
	if computeCommissionRates {
		params.Set("computeCommissionRates", "true")
		rateLimit = testNewOrderWithCommissionRate
	}
	var resp *OrderTestResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/api/v3/sor/order/test", params, rateLimit, &resp)
}

// sorOrderParams validates an order using smart order routing and builds its parameters
func (e *Exchange) sorOrderParams(req *SOROrderRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if req.Type == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if req.Quantity <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", req.Side)
	params.Set("type", req.Type)
	params.Set("quantity", strconv.FormatFloat(req.Quantity, 'f', -1, 64))
	if req.TimeInForce != "" {
		params.Set("timeInForce", req.TimeInForce)
	}
	if req.Price > 0 {
		params.Set("price", strconv.FormatFloat(req.Price, 'f', -1, 64))
	}
	if req.NewClientOrderID != "" {
		params.Set("newClientOrderId", req.NewClientOrderID)
	}
	if req.StrategyID != 0 {
		params.Set("strategyId", strconv.FormatUint(req.StrategyID, 10))
	}
	if req.StrategyType != 0 {
		params.Set("strategyType", strconv.FormatUint(req.StrategyType, 10))
	}
	if req.IcebergQuantity > 0 {
		params.Set("icebergQty", strconv.FormatFloat(req.IcebergQuantity, 'f', -1, 64))
	}
	if req.NewOrderResponseType != "" {
		params.Set("newOrderRespType", req.NewOrderResponseType)
	}
	if req.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", req.SelfTradePreventionMode)
	}
	return params, nil
}

// GetCommissionRates queries the account's current commission rates for a symbol
func (e *Exchange) GetCommissionRates(ctx context.Context, symbol currency.Pair) (*AccountCommissionResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	var resp *AccountCommissionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/api/v3/account/commission", params, getCommissionRate, &resp)
}

// GetAllOCOOrders queries all order lists. A nil request returns the most recent order lists
func (e *Exchange) GetAllOCOOrders(ctx context.Context, req *AllOrderListsRequest) ([]OCOOrderResponse, error) {
	params := url.Values{}
	if req != nil {
		if req.FromID != 0 && (!req.StartTime.IsZero() || !req.EndTime.IsZero()) {
			return nil, fmt.Errorf("%w: fromId cannot be sent with startTime or endTime", errInvalidParameterCombination)
		}
		if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
			if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
				return nil, err
			}
		}
		if req.FromID != 0 {
			params.Set("fromId", strconv.FormatUint(req.FromID, 10))
		}
		if !req.StartTime.IsZero() {
			params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
		}
		if !req.EndTime.IsZero() {
			params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
		}
		if req.Limit > 0 {
			params.Set("limit", strconv.FormatUint(req.Limit, 10))
		}
	}
	var resp []OCOOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/api/v3/allOrderList", params, getAllOCOOrdersRate, &resp)
}

// AllOrders returns all of the account's orders on a symbol: active, cancelled or filled
func (e *Exchange) AllOrders(ctx context.Context, req *AllOrdersRequest) ([]TradeOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if req.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []TradeOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/api/v3/allOrders", params, spotAllOrdersRate, &resp)
}

// GetAccount returns the current account information; omitZeroBalances leaves out zero balances
func (e *Exchange) GetAccount(ctx context.Context, omitZeroBalances bool) (*AccountResponse, error) {
	params := url.Values{}
	if omitZeroBalances {
		params.Set("omitZeroBalances", "true")
	}
	var resp *AccountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/api/v3/account", params, spotAccountInformationRate, &resp)
}

// OpenOrders returns the current open orders on a symbol, or on every symbol for an empty pair, which weighs far more
func (e *Exchange) OpenOrders(ctx context.Context, pair currency.Pair) ([]TradeOrderResponse, error) {
	params := url.Values{}
	var symbol string
	if !pair.IsEmpty() {
		var err error
		if symbol, err = e.FormatSymbol(pair, asset.Spot); err != nil {
			return nil, err
		}
		params.Set("symbol", symbol)
	}
	var resp []TradeOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/api/v3/openOrders", params, spotOpenOrdersLimit(symbol), &resp)
}

// QueryOrder checks an order's status by orderID or origClientOrderID
func (e *Exchange) QueryOrder(ctx context.Context, symbol currency.Pair, origClientOrderID string, orderID uint64) (*TradeOrderResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if orderID == 0 && origClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if orderID != 0 {
		params.Set("orderId", strconv.FormatUint(orderID, 10))
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	var resp *TradeOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/api/v3/order", params, spotOrderQueryRate, &resp)
}

// GetOCOOrders queries an order list by orderListID or by its list client order ID, which the endpoint takes as
// origClientOrderId
func (e *Exchange) GetOCOOrders(ctx context.Context, orderListID uint64, origClientOrderID string) (*OCOOrderResponse, error) {
	if orderListID == 0 && origClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	if orderListID != 0 {
		params.Set("orderListId", strconv.FormatUint(orderListID, 10))
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	var resp *OCOOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/api/v3/orderList", params, getOCOListRate, &resp)
}

// GetAllocations queries the allocations resulting from orders placed using smart order routing (SOR)
func (e *Exchange) GetAllocations(ctx context.Context, req *AllocationsRequest) ([]Allocation, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.FromAllocationID != 0 {
		params.Set("fromAllocationId", strconv.FormatUint(req.FromAllocationID, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if req.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	}
	var resp []Allocation
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/api/v3/myAllocations", params, getAllocationsRate, &resp)
}

// GetPreventedMatches queries the orders that expired because of self-trade prevention (STP), by preventedMatchID or
// by orderID
func (e *Exchange) GetPreventedMatches(ctx context.Context, req *PreventedMatchesRequest) ([]PreventedMatch, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.PreventedMatchID == 0 && req.OrderID == 0 {
		return nil, fmt.Errorf("%w: preventedMatchId or orderId is required", order.ErrOrderIDNotSet)
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	rateLimit := preventedMatchesRate
	if req.PreventedMatchID != 0 {
		params.Set("preventedMatchId", strconv.FormatUint(req.PreventedMatchID, 10))
	}
	if req.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
		rateLimit = preventedMatchesByOrderIDRate
	}
	if req.FromPreventedMatchID != 0 {
		params.Set("fromPreventedMatchId", strconv.FormatUint(req.FromPreventedMatchID, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []PreventedMatch
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/api/v3/myPreventedMatches", params, rateLimit, &resp)
}

// GetAccountTradeList returns the account's trades on a symbol
func (e *Exchange) GetAccountTradeList(ctx context.Context, req *AccountTradeListRequest) ([]AccountTrade, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if req.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.FromID != 0 {
		params.Set("fromId", strconv.FormatUint(req.FromID, 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []AccountTrade
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/api/v3/myTrades", params, spotAccountTradeListLimit(req.OrderID), &resp)
}

// GetOpenOCOList queries the open order lists
func (e *Exchange) GetOpenOCOList(ctx context.Context) ([]OCOOrderResponse, error) {
	var resp []OCOOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/api/v3/openOrderList", nil, getOpenOCOListRate, &resp)
}

// GetCurrentOrderCountUsage queries the account's unfilled order count for every interval
func (e *Exchange) GetCurrentOrderCountUsage(ctx context.Context) ([]RateLimitItem, error) {
	var resp []RateLimitItem
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/api/v3/rateLimit/order", nil, currentOrderCountUsageRate, &resp)
}

// SendHTTPRequest sends an unauthenticated request
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

// SendAPIKeyHTTPRequest sends a request carrying the API key without a signature, which MARKET_DATA and USER_STREAM
// endpoints require
func (e *Exchange) SendAPIKeyHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, f request.EndpointLimit, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	return e.sendRequest(ctx, f, request.AuthenticatedRequest, result, func() (*request.Item, error) {
		return &request.Item{
			Method:                 method,
			Path:                   endpoint + path,
			Headers:                map[string]string{"X-MBX-APIKEY": creds.Key},
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	})
}

// SendAuthHTTPRequest sends a signed request, carrying every parameter in the query string
func (e *Exchange) SendAuthHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, params url.Values, f request.EndpointLimit, result any) error {
	endpoint, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	if params == nil {
		params = url.Values{}
	}
	if !params.Has("recvWindow") {
		params.Set("recvWindow", strconv.FormatInt(defaultRecvWindow.Milliseconds(), 10))
	}
	return e.sendRequest(ctx, f, request.AuthenticatedRequest, result, func() (*request.Item, error) {
		// The timestamp is renewed for every attempt, so a retried request is not rejected as outside recvWindow
		params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
		signature, err := crypto.GetHMAC(crypto.HashSHA256, []byte(params.Encode()), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		return &request.Item{
			Method:                 method,
			Path:                   common.EncodeURLValues(endpoint+path, params) + "&signature=" + hex.EncodeToString(signature),
			Headers:                map[string]string{"X-MBX-APIKEY": creds.Key},
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	})
}

// sendRequest sends a request and decodes its response into result. Binance reports errors as {"code":-N,"msg":"..."},
// usually with an HTTP error status but on some endpoints with a 200, so both are checked for a negative code.
// Successful replies such as {"code":200,"msg":"success"} carry a non-negative code and decode normally
func (e *Exchange) sendRequest(ctx context.Context, f request.EndpointLimit, auth request.AuthType, result any, newItem func() (*request.Item, error)) error {
	var raw json.RawMessage
	err := e.SendPayload(ctx, f, func() (*request.Item, error) {
		item, err := newItem()
		if err != nil {
			return nil, err
		}
		raw = nil
		item.Result = &raw
		return item, nil
	}, auth)
	if apiErr := checkErrorResponse(raw); apiErr != nil {
		if err != nil {
			return fmt.Errorf("%w: %w", apiErr, err)
		}
		return apiErr
	}
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}
	// The request layer rejects an empty body itself, so only a null body is left to catch
	if string(raw) == "null" {
		return common.ErrNoResponse
	}
	return json.Unmarshal(raw, result)
}

// setTimeRangeParams validates a query's time range and sets its bounds in milliseconds. An unset bound is left out,
// so the endpoint applies its documented default window
func setTimeRangeParams(params url.Values, startTime, endTime time.Time) error {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return err
		}
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return nil
}

// checkErrorResponse returns an error when a response body is a Binance error object
func checkErrorResponse(raw json.RawMessage) error {
	if len(raw) == 0 || raw[0] != '{' {
		return nil
	}
	// A body that does not decode as an error object is left for the caller's decoding to report
	var resp ErrResponse
	if json.Unmarshal(raw, &resp) == nil && resp.Code.Int64() < 0 {
		return &APIError{Code: resp.Code.Int64(), Message: resp.Message, Data: resp.Data}
	}
	return nil
}

// APIError is an error Binance replied with, over REST or the WebSocket API
type APIError struct {
	Code    int64
	Message string
	// Data holds the details some endpoints add, such as the outcome of each half of a cancel-replace
	Data json.RawMessage
}

// Error returns the error's code and message, and its data when it has any
func (a *APIError) Error() string {
	msg := errAPIResponse.Error()
	if mapped, ok := errorCodeToErrorMap[a.Code]; ok {
		msg += ": " + mapped.Error()
	}
	msg += ": code " + strconv.FormatInt(a.Code, 10) + " msg " + a.Message
	if len(a.Data) != 0 {
		msg += " data " + string(a.Data)
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

// errorCodeToErrorMap maps Binance error codes to package errors callers can match
var errorCodeToErrorMap = map[int64]error{
	-1002: request.ErrAuthRequestFailed,
	-1121: currency.ErrPairNotFound,
	-2013: order.ErrOrderNotFound,
}
