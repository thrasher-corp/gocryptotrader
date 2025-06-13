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
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Binance is the overarching type across the Binance package
type Binance struct {
	exchange.Base
	obm *orderbookManager
}

const (
	apiURL         = "https://api.binance.com"
	spotAPIURL     = "https://sapi.binance.com"
	cfuturesAPIURL = "https://dapi.binance.com"
	ufuturesAPIURL = "https://fapi.binance.com"
	tradeBaseURL   = "https://www.binance.com/en/"

	testnetSpotURL = "https://testnet.binance.vision/api" //nolint:unused // Can be used for testing via setting useTestNet to true
	testnetFutures = "https://testnet.binancefuture.com"  //nolint:unused // Can be used for testing via setting useTestNet to true

	// Public endpoints
	exchangeInfo      = "/api/v3/exchangeInfo"
	orderBookDepth    = "/api/v3/depth"
	recentTrades      = "/api/v3/trades"
	aggregatedTrades  = "/api/v3/aggTrades"
	candleStick       = "/api/v3/klines"
	averagePrice      = "/api/v3/avgPrice"
	priceChange       = "/api/v3/ticker/24hr"
	symbolPrice       = "/api/v3/ticker/price"
	bestPrice         = "/api/v3/ticker/bookTicker"
	userAccountStream = "/api/v3/userDataStream"
	perpExchangeInfo  = "/fapi/v1/exchangeInfo"
	historicalTrades  = "/api/v3/historicalTrades"

	// Margin endpoints
	marginInterestHistory = "/sapi/v1/margin/interestHistory"

	// Authenticated endpoints
	newOrderTest      = "/api/v3/order/test"
	orderEndpoint     = "/api/v3/order"
	openOrders        = "/api/v3/openOrders"
	allOrders         = "/api/v3/allOrders"
	accountInfo       = "/api/v3/account"
	marginAccountInfo = "/sapi/v1/margin/account"

	// Wallet endpoints
	allCoinsInfo     = "/sapi/v1/capital/config/getall"
	withdrawEndpoint = "/sapi/v1/capital/withdraw/apply"
	depositHistory   = "/sapi/v1/capital/deposit/hisrec"
	withdrawHistory  = "/sapi/v1/capital/withdraw/history"
	depositAddress   = "/sapi/v1/capital/deposit/address"

	// Crypto loan endpoints
	loanIncomeHistory            = "/sapi/v1/loan/income"
	loanBorrow                   = "/sapi/v1/loan/borrow"
	loanBorrowHistory            = "/sapi/v1/loan/borrow/history"
	loanOngoingOrders            = "/sapi/v1/loan/ongoing/orders"
	loanRepay                    = "/sapi/v1/loan/repay"
	loanRepaymentHistory         = "/sapi/v1/loan/repay/history"
	loanAdjustLTV                = "/sapi/v1/loan/adjust/ltv"
	loanLTVAdjustmentHistory     = "/sapi/v1/loan/ltv/adjustment/history"
	loanableAssetsData           = "/sapi/v1/loan/loanable/data"
	loanCollateralAssetsData     = "/sapi/v1/loan/collateral/data"
	loanCheckCollateralRepayRate = "/sapi/v1/loan/repay/collateral/rate"
	loanCustomiseMarginCall      = "/sapi/v1/loan/customize/margin_call"

	// Flexible loan endpoints
	flexibleLoanBorrow               = "/sapi/v1/loan/flexible/borrow"
	flexibleLoanOngoingOrders        = "/sapi/v1/loan/flexible/ongoing/orders"
	flexibleLoanBorrowHistory        = "/sapi/v1/loan/flexible/borrow/history"
	flexibleLoanRepay                = "/sapi/v1/loan/flexible/repay"
	flexibleLoanRepayHistory         = "/sapi/v1/loan/flexible/repay/history"
	flexibleLoanAdjustLTV            = "/sapi/v1/loan/flexible/adjust/ltv"
	flexibleLoanLTVHistory           = "/sapi/v1/loan/flexible/ltv/adjustment/history"
	flexibleLoanAssetsData           = "/sapi/v1/loan/flexible/loanable/data"
	flexibleLoanCollateralAssetsData = "/sapi/v1/loan/flexible/collateral/data"

	defaultRecvWindow = 5 * time.Second
)

var (
	errLoanCoinMustBeSet                      = errors.New("loan coin must bet set")
	errLoanTermMustBeSet                      = errors.New("loan term must be set")
	errCollateralCoinMustBeSet                = errors.New("collateral coin must be set")
	errOrderIDMustBeSet                       = errors.New("orderID must be set")
	errAmountMustBeSet                        = errors.New("amount must not be <= 0")
	errEitherLoanOrCollateralAmountsMustBeSet = errors.New("either loan or collateral amounts must be set")
)

// GetExchangeInfo returns exchange information. Check binance_types for more
// information
func (b *Binance) GetExchangeInfo(ctx context.Context) (ExchangeInfo, error) {
	var resp ExchangeInfo
	return resp, b.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary, exchangeInfo, spotExchangeInfo, &resp)
}

// GetOrderBook returns full orderbook information
//
// OrderBookDataRequestParams contains the following members
// symbol: string of currency pair
// limit: returned limit amount
func (b *Binance) GetOrderBook(ctx context.Context, obd OrderBookDataRequestParams) (*OrderBook, error) {
	params := url.Values{}
	symbol, err := b.FormatSymbol(obd.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	params.Set("limit", strconv.Itoa(obd.Limit))

	var resp *OrderBookData
	if err := b.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues(orderBookDepth, params), orderbookLimit(obd.Limit), &resp); err != nil {
		return nil, err
	}

	ob := &OrderBook{
		Bids:         make([]OrderbookItem, len(resp.Bids)),
		Asks:         make([]OrderbookItem, len(resp.Asks)),
		LastUpdateID: resp.LastUpdateID,
	}
	for x := range resp.Bids {
		ob.Bids[x].Price = resp.Bids[x][0].Float64()
		ob.Bids[x].Quantity = resp.Bids[x][1].Float64()
	}

	for x := range resp.Asks {
		ob.Asks[x].Price = resp.Asks[x][0].Float64()
		ob.Asks[x].Quantity = resp.Asks[x][1].Float64()
	}

	return ob, nil
}

// GetMostRecentTrades returns recent trade activity
// limit: Up to 500 results returned
func (b *Binance) GetMostRecentTrades(ctx context.Context, rtr RecentTradeRequestParams) ([]RecentTrade, error) {
	params := url.Values{}
	symbol, err := b.FormatSymbol(rtr.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	params.Set("limit", strconv.Itoa(rtr.Limit))

	path := recentTrades + "?" + params.Encode()

	var resp []RecentTrade
	return resp, b.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
}

// GetHistoricalTrades returns historical trade activity
//
// symbol: string of currency pair
// limit: Optional. Default 500; max 1000.
// fromID:
func (b *Binance) GetHistoricalTrades(ctx context.Context, symbol string, limit int, fromID int64) ([]HistoricalTrade, error) {
	var resp []HistoricalTrade
	params := url.Values{}

	params.Set("symbol", symbol)
	params.Set("limit", strconv.Itoa(limit))
	// else return most recent trades
	if fromID > 0 {
		params.Set("fromId", strconv.FormatInt(fromID, 10))
	}

	path := historicalTrades + "?" + params.Encode()
	return resp,
		b.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
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

	path := marginInterestHistory + "?" + params.Encode()
	var resp UserMarginInterestHistoryResponse
	return &resp, b.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
}

// GetAggregatedTrades returns aggregated trade activity.
// If more than one hour of data is requested or asked limit is not supported by exchange
// then the trades are collected with multiple backend requests.
// https://binance-docs.github.io/apidocs/spot/en/#compressed-aggregate-trades-list
func (b *Binance) GetAggregatedTrades(ctx context.Context, arg *AggregatedTradeRequestParams) ([]AggregatedTrade, error) {
	params := url.Values{}
	params.Set("symbol", arg.Symbol.String())
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
		params.Set("startTime", strconv.FormatInt(arg.StartTime.UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(arg.EndTime.UnixMilli(), 10))
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
	path := aggregatedTrades + "?" + params.Encode()
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
			params.Set("startTime", strconv.FormatInt(start.UnixMilli(), 10))
			params.Set("endTime", strconv.FormatInt(start.Add(increment).UnixMilli(), 10))
			path := aggregatedTrades + "?" + params.Encode()
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
		path := aggregatedTrades + "?" + params.Encode()
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
	symbol, err := b.FormatSymbol(arg.Symbol, asset.Spot)
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
	return resp, b.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues(candleStick, params), spotDefaultRate, &resp)
}

// GetAveragePrice returns current average price for a symbol.
//
// symbol: string of currency pair
func (b *Binance) GetAveragePrice(ctx context.Context, symbol currency.Pair) (AveragePrice, error) {
	resp := AveragePrice{}
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)

	path := averagePrice + "?" + params.Encode()

	return resp, b.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
}

// GetPriceChangeStats returns price change statistics for the last 24 hours
//
// symbol: string of currency pair
func (b *Binance) GetPriceChangeStats(ctx context.Context, symbol currency.Pair) (*PriceChangeStats, error) {
	resp := PriceChangeStats{}
	params := url.Values{}
	rateLimit := spotTickerAllRate
	if !symbol.IsEmpty() {
		rateLimit = spotTicker1Rate
		symbolValue, err := b.FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	path := priceChange + "?" + params.Encode()

	return &resp, b.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, rateLimit, &resp)
}

// GetTickers returns the ticker data for the last 24 hrs
func (b *Binance) GetTickers(ctx context.Context, symbols ...currency.Pair) ([]PriceChangeStats, error) {
	var resp []PriceChangeStats
	symbolLength := len(symbols)
	params := url.Values{}
	var rl request.EndpointLimit
	switch {
	case symbolLength == 1:
		rl = spotTicker1Rate
	case symbolLength > 1 && symbolLength <= 20:
		rl = spotTicker20Rate
	case symbolLength > 20 && symbolLength <= 100:
		rl = spotTicker100Rate
	case symbolLength > 100, symbolLength == 0:
		rl = spotTickerAllRate
	}
	path := priceChange
	if symbolLength > 0 {
		symbolValues := make([]string, symbolLength)
		for i := range symbols {
			symbolValue, err := b.FormatSymbol(symbols[i], asset.Spot)
			if err != nil {
				return resp, err
			}
			symbolValues[i] = "\"" + symbolValue + "\""
		}
		params.Set("symbols", "["+strings.Join(symbolValues, ",")+"]")
		path += "?" + params.Encode()
	}
	return resp, b.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, rl, &resp)
}

// GetLatestSpotPrice returns latest spot price of symbol
//
// symbol: string of currency pair
func (b *Binance) GetLatestSpotPrice(ctx context.Context, symbol currency.Pair) (SymbolPrice, error) {
	resp := SymbolPrice{}
	params := url.Values{}
	rateLimit := spotTickerAllRate
	if !symbol.IsEmpty() {
		rateLimit = spotDefaultRate
		symbolValue, err := b.FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	path := symbolPrice + "?" + params.Encode()

	return resp,
		b.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, rateLimit, &resp)
}

// GetBestPrice returns the latest best price for symbol
//
// symbol: string of currency pair
func (b *Binance) GetBestPrice(ctx context.Context, symbol currency.Pair) (BestPrice, error) {
	resp := BestPrice{}
	params := url.Values{}
	rateLimit := spotOrderbookTickerAllRate
	if !symbol.IsEmpty() {
		rateLimit = spotDefaultRate
		symbolValue, err := b.FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	path := bestPrice + "?" + params.Encode()

	return resp,
		b.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, rateLimit, &resp)
}

// NewOrder sends a new order to Binance
func (b *Binance) NewOrder(ctx context.Context, o *NewOrderRequest) (NewOrderResponse, error) {
	var resp NewOrderResponse
	if err := b.newOrder(ctx, orderEndpoint, o, &resp); err != nil {
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
	return b.newOrder(ctx, newOrderTest, o, &resp)
}

func (b *Binance) newOrder(ctx context.Context, api string, o *NewOrderRequest, resp *NewOrderResponse) error {
	params := url.Values{}
	symbol, err := b.FormatSymbol(o.Symbol, asset.Spot)
	if err != nil {
		return err
	}
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
	return b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, api, params, spotOrderRate, resp)
}

// CancelExistingOrder sends a cancel order to Binance
func (b *Binance) CancelExistingOrder(ctx context.Context, symbol currency.Pair, orderID int64, origClientOrderID string) (CancelOrderResponse, error) {
	var resp CancelOrderResponse

	symbolValue, err := b.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)

	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}

	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodDelete, orderEndpoint, params, spotOrderRate, &resp)
}

// OpenOrders Current open orders. Get all open orders on a symbol.
// Careful when accessing this with no symbol: The number of requests counted
// against the rate limiter is significantly higher
func (b *Binance) OpenOrders(ctx context.Context, pair currency.Pair) ([]QueryOrderData, error) {
	var resp []QueryOrderData
	params := url.Values{}
	var p string
	var err error
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
	if err := b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet,
		openOrders,
		params,
		openOrdersLimit(p),
		&resp); err != nil {
		return resp, err
	}

	return resp, nil
}

// AllOrders Get all account orders; active, canceled, or filled.
// orderId optional param
// limit optional param, default 500; max 500
func (b *Binance) AllOrders(ctx context.Context, symbol currency.Pair, orderID, limit string) ([]QueryOrderData, error) {
	var resp []QueryOrderData

	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if limit != "" {
		params.Set("limit", limit)
	}
	if err := b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet,
		allOrders,
		params,
		spotAllOrdersRate,
		&resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// QueryOrder returns information on a past order
func (b *Binance) QueryOrder(ctx context.Context, symbol currency.Pair, origClientOrderID string, orderID int64) (QueryOrderData, error) {
	var resp QueryOrderData

	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}

	if err := b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet, orderEndpoint,
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
	params := url.Values{}

	if err := b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet, accountInfo,
		params, spotAccountInformationRate,
		&resp); err != nil {
		return &resp.Account, err
	}

	if resp.Code != 0 {
		return &resp.Account, errors.New(resp.Msg)
	}

	return &resp.Account, nil
}

// GetMarginAccount returns account information for margin accounts
func (b *Binance) GetMarginAccount(ctx context.Context) (*MarginAccount, error) {
	var resp MarginAccount
	params := url.Values{}

	if err := b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet, marginAccountInfo,
		params, spotAccountInformationRate,
		&resp); err != nil {
		return &resp, err
	}

	return &resp, nil
}

// SendHTTPRequest sends an unauthenticated request
func (b *Binance) SendHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result any) error {
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
		HTTPRecording: b.HTTPRecording,
	}

	return b.SendPayload(ctx, f, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

// SendAPIKeyHTTPRequest is a special API request where the api key is
// appended to the headers without a secret
func (b *Binance) SendAPIKeyHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result any) error {
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
		HTTPRecording: b.HTTPRecording,
	}

	return b.SendPayload(ctx, f, func() (*request.Item, error) {
		return item, nil
	}, request.AuthenticatedRequest)
}

// SendAuthHTTPRequest sends an authenticated HTTP request
func (b *Binance) SendAuthHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, params url.Values, f request.EndpointLimit, result any) error {
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
		params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
		hmacSigned, err := crypto.GetHMAC(crypto.HashSHA256, []byte(params.Encode()), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}
		headers := make(map[string]string)
		headers["X-MBX-APIKEY"] = creds.Key
		fullPath := common.EncodeURLValues(endpointPath+path, params) + "&signature=" + hex.EncodeToString(hmacSigned)
		return &request.Item{
			Method:        method,
			Path:          fullPath,
			Headers:       headers,
			Result:        &interim,
			Verbose:       b.Verbose,
			HTTPDebugging: b.HTTPDebugging,
			HTTPRecording: b.HTTPRecording,
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

// GetAllCoinsInfo returns details about all supported coins
func (b *Binance) GetAllCoinsInfo(ctx context.Context) ([]CoinInfo, error) {
	var resp []CoinInfo
	if err := b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet,
		allCoinsInfo,
		nil,
		spotDefaultRate,
		&resp); err != nil {
		return nil, err
	}
	return resp, nil
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

	var resp WithdrawResponse
	if err := b.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodPost,
		withdrawEndpoint,
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
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}

	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
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
		depositHistory,
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
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}

	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
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
		withdrawHistory,
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
	var d DepositAddress
	return &d,
		b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, depositAddress, params, spotDefaultRate, &d)
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
		Path:          endpointPath + userAccountStream,
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

	path := endpointPath + userAccountStream
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

		for i := range s.PermissionSets {
			if !slices.Contains(s.PermissionSets[i], aUpper) {
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
			break
		}
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
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, loanIncomeHistory, params, spotDefaultRate, &resp)
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
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, loanBorrow, params, spotDefaultRate, &resp)
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

	var resp LoanBorrowHistory
	return &resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, loanBorrowHistory, params, spotDefaultRate, &resp)
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

	var resp CryptoLoanOngoingOrder
	return &resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, loanOngoingOrders, params, spotDefaultRate, &resp)
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
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, loanRepay, params, spotDefaultRate, &resp)
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

	var resp CryptoLoanRepayHistory
	return &resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, loanRepaymentHistory, params, spotDefaultRate, &resp)
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

	var resp CryptoLoanAdjustLTV
	return &resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, loanAdjustLTV, params, spotDefaultRate, &resp)
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

	var resp CryptoLoanLTVAdjustmentHistory
	return &resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, loanLTVAdjustmentHistory, params, spotDefaultRate, &resp)
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

	var resp LoanableAssetsData
	return &resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, loanableAssetsData, params, spotDefaultRate, &resp)
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

	var resp CollateralAssetData
	return &resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, loanCollateralAssetsData, params, spotDefaultRate, &resp)
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

	var resp CollateralRepayRate
	return &resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, loanCheckCollateralRepayRate, params, spotDefaultRate, &resp)
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

	var resp CustomiseMarginCall
	return &resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, loanCustomiseMarginCall, params, spotDefaultRate, &resp)
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

	var resp FlexibleLoanBorrow
	return &resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, flexibleLoanBorrow, params, spotDefaultRate, &resp)
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

	var resp FlexibleLoanOngoingOrder
	return &resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, flexibleLoanOngoingOrders, params, spotDefaultRate, &resp)
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

	var resp FlexibleLoanBorrowHistory
	return &resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, flexibleLoanBorrowHistory, params, spotDefaultRate, &resp)
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

	var resp FlexibleLoanRepay
	return &resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, flexibleLoanRepay, params, spotDefaultRate, &resp)
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

	var resp FlexibleLoanRepayHistory
	return &resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, flexibleLoanRepayHistory, params, spotDefaultRate, &resp)
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

	var resp FlexibleLoanAdjustLTV
	return &resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, flexibleLoanAdjustLTV, params, spotDefaultRate, &resp)
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

	var resp FlexibleLoanLTVAdjustmentHistory
	return &resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, flexibleLoanLTVHistory, params, spotDefaultRate, &resp)
}

// FlexibleLoanAssetsData gets the flexible loan assets data
func (b *Binance) FlexibleLoanAssetsData(ctx context.Context, loanCoin currency.Code) (*FlexibleLoanAssetsData, error) {
	params := url.Values{}
	if !loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.String())
	}

	var resp FlexibleLoanAssetsData
	return &resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, flexibleLoanAssetsData, params, spotDefaultRate, &resp)
}

// FlexibleCollateralAssetsData gets the flexible loan collateral assets data
func (b *Binance) FlexibleCollateralAssetsData(ctx context.Context, collateralCoin currency.Code) (*FlexibleCollateralAssetsData, error) {
	params := url.Values{}
	if !collateralCoin.IsEmpty() {
		params.Set("collateralCoin", collateralCoin.String())
	}

	var resp FlexibleCollateralAssetsData
	return &resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, flexibleLoanCollateralAssetsData, params, spotDefaultRate, &resp)
}
