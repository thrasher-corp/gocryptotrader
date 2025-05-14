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
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with Binance
type Exchange struct {
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
func (e *Exchange) GetExchangeInfo(ctx context.Context) (ExchangeInfo, error) {
	var resp ExchangeInfo
	return resp, e.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary, exchangeInfo, spotExchangeInfo, &resp)
}

// GetOrderBook returns full orderbook information
func (e *Exchange) GetOrderBook(ctx context.Context, pair currency.Pair, limit uint64) (*OrderBookResponse, error) {
	symbol, err := e.FormatSymbol(pair, asset.Spot)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("limit", strconv.FormatUint(limit, 10))

	var resp *OrderBookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues(orderBookDepth, params), orderbookLimit(limit), &resp)
}

// GetMostRecentTrades returns recent trade activity
// limit: Up to 500 results returned
func (e *Exchange) GetMostRecentTrades(ctx context.Context, rtr RecentTradeRequestParams) ([]RecentTrade, error) {
	params := url.Values{}
	symbol, err := e.FormatSymbol(rtr.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	params.Set("limit", strconv.Itoa(rtr.Limit))

	path := recentTrades + "?" + params.Encode()

	var resp []RecentTrade
	return resp, e.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
}

// GetHistoricalTrades returns historical trade activity
//
// symbol: string of currency pair
// limit: Optional. Default 500; max 1000.
// fromID:
func (e *Exchange) GetHistoricalTrades(ctx context.Context, symbol string, limit int, fromID int64) ([]HistoricalTrade, error) {
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
		e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
}

// GetUserMarginInterestHistory returns margin interest history for the user
func (e *Exchange) GetUserMarginInterestHistory(ctx context.Context, assetCurrency currency.Code, isolatedSymbol currency.Pair, startTime, endTime time.Time, currentPage, size int64, archived bool) (*UserMarginInterestHistoryResponse, error) {
	params := url.Values{}

	if !assetCurrency.IsEmpty() {
		params.Set("asset", assetCurrency.String())
	}
	if !isolatedSymbol.IsEmpty() {
		fPair, err := e.FormatSymbol(isolatedSymbol, asset.Margin)
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
	return &resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
}

// GetAggregatedTrades returns aggregated trade activity.
// If more than one hour of data is requested or asked limit is not supported by exchange
// then the trades are collected with multiple backend requests.
// https://binance-docs.github.io/apidocs/spot/en/#compressed-aggregate-trades-list
func (e *Exchange) GetAggregatedTrades(ctx context.Context, arg *AggregatedTradeRequestParams) ([]AggregatedTrade, error) {
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
			return e.batchAggregateTrades(ctx, arg, params)
		}

		// Can't handle this request locally or remotely
		// We would receive {"code":-1128,"msg":"Combination of optional parameters invalid."}
		return nil, errors.New("please set StartTime or FromId, but not both")
	}
	var resp []AggregatedTrade
	path := aggregatedTrades + "?" + params.Encode()
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
}

// batchAggregateTrades fetches trades in multiple requests
// If no args.FromID is passed, requests are made intervals doubling from 10s until we get a viable starting position.
// Once the starting set is found, we continue scanning until we have all the trades in the time window
func (e *Exchange) batchAggregateTrades(ctx context.Context, args *AggregatedTradeRequestParams, params url.Values) ([]AggregatedTrade, error) {
	var resp []AggregatedTrade
	// prepare first request with only first hour and max limit
	if args.Limit == 0 || args.Limit > 1000 {
		// Extend from the default of 500
		params.Set("limit", "1000")
	}

	var fromID int64
	if args.FromID > 0 {
		fromID = args.FromID
	} else {
		// Only 10 seconds is used to prevent limit of 1000 being reached in the first request, cutting off trades for high activity pairs
		// If we don't find anything we keep increasing the window by doubling the interval and scanning again
		increment := time.Second * 10
		for start := args.StartTime; len(resp) == 0; start, increment = start.Add(increment), increment*2 {
			if !args.EndTime.IsZero() && start.After(args.EndTime) || increment <= 0 {
				// All requests returned empty or we searched until increment overflowed
				return nil, nil
			}
			params.Set("startTime", strconv.FormatInt(start.UnixMilli(), 10))
			end := start.Add(increment)
			if !args.EndTime.IsZero() && end.After(args.EndTime) {
				end = args.EndTime
			}
			params.Set("endTime", strconv.FormatInt(end.UnixMilli(), 10))
			path := aggregatedTrades + "?" + params.Encode()
			err := e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
			if err != nil {
				return resp, fmt.Errorf("%w %v", err, args.Symbol)
			}
		}
		fromID = resp[len(resp)-1].ATradeID
	}

	// other requests follow from the last aggregate trade id and have no time window
	params.Del("startTime")
	params.Del("endTime")
outer:
	for ; args.Limit == 0 || len(resp) < args.Limit; fromID = resp[len(resp)-1].ATradeID {
		// Keep requesting new data after last retrieved trade
		params.Set("fromId", strconv.FormatInt(fromID, 10))
		path := aggregatedTrades + "?" + params.Encode()
		var additionalTrades []AggregatedTrade
		if err := e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, spotDefaultRate, &additionalTrades); err != nil {
			return resp, fmt.Errorf("%w %v", err, args.Symbol)
		}
		switch len(additionalTrades) {
		case 0, 1:
			break outer // We only got the one we already have
		default:
			additionalTrades = additionalTrades[1:] // Remove the record we already have
		}
		if !args.EndTime.IsZero() {
			// Check if only some of the results are before EndTime
			if afterEnd := sort.Search(len(additionalTrades), func(i int) bool {
				return args.EndTime.Before(additionalTrades[i].TimeStamp.Time())
			}); afterEnd < len(additionalTrades) {
				resp = append(resp, additionalTrades[:afterEnd]...)
				break
			}
		}
		resp = append(resp, additionalTrades...)
	}
	if args.Limit > 0 && len(resp) > args.Limit {
		resp = resp[:args.Limit]
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
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, common.EncodeURLValues(candleStick, params), spotDefaultRate, &resp)
}

// GetAveragePrice returns current average price for a symbol.
//
// symbol: string of currency pair
func (e *Exchange) GetAveragePrice(ctx context.Context, symbol currency.Pair) (AveragePrice, error) {
	resp := AveragePrice{}
	params := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)

	path := averagePrice + "?" + params.Encode()

	return resp, e.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
}

// GetPriceChangeStats returns price change statistics for the last 24 hours
//
// symbol: string of currency pair
func (e *Exchange) GetPriceChangeStats(ctx context.Context, symbol currency.Pair) (*PriceChangeStats, error) {
	resp := PriceChangeStats{}
	params := url.Values{}
	rateLimit := spotTickerAllRate
	if !symbol.IsEmpty() {
		rateLimit = spotTicker1Rate
		symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	path := priceChange + "?" + params.Encode()

	return &resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, rateLimit, &resp)
}

// GetTickers returns the ticker data for the last 24 hrs
func (e *Exchange) GetTickers(ctx context.Context, symbols ...currency.Pair) ([]PriceChangeStats, error) {
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
			symbolValue, err := e.FormatSymbol(symbols[i], asset.Spot)
			if err != nil {
				return resp, err
			}
			symbolValues[i] = "\"" + symbolValue + "\""
		}
		params.Set("symbols", "["+strings.Join(symbolValues, ",")+"]")
		path += "?" + params.Encode()
	}
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, rl, &resp)
}

// GetLatestSpotPrice returns latest spot price of symbol
//
// symbol: string of currency pair
func (e *Exchange) GetLatestSpotPrice(ctx context.Context, symbol currency.Pair) (SymbolPrice, error) {
	resp := SymbolPrice{}
	params := url.Values{}
	rateLimit := spotTickerAllRate
	if !symbol.IsEmpty() {
		rateLimit = spotDefaultRate
		symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	path := symbolPrice + "?" + params.Encode()

	return resp,
		e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, rateLimit, &resp)
}

// GetBestPrice returns the latest best price for symbol
//
// symbol: string of currency pair
func (e *Exchange) GetBestPrice(ctx context.Context, symbol currency.Pair) (BestPrice, error) {
	resp := BestPrice{}
	params := url.Values{}
	rateLimit := spotOrderbookTickerAllRate
	if !symbol.IsEmpty() {
		rateLimit = spotDefaultRate
		symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	path := bestPrice + "?" + params.Encode()

	return resp,
		e.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, rateLimit, &resp)
}

// NewOrder sends a new order to Binance
func (e *Exchange) NewOrder(ctx context.Context, o *NewOrderRequest) (NewOrderResponse, error) {
	var resp NewOrderResponse
	if err := e.newOrder(ctx, orderEndpoint, o, &resp); err != nil {
		return resp, err
	}

	if resp.Code != 0 {
		return resp, errors.New(resp.Msg)
	}

	return resp, nil
}

// NewOrderTest sends a new test order to Binance
func (e *Exchange) NewOrderTest(ctx context.Context, o *NewOrderRequest) error {
	var resp NewOrderResponse
	return e.newOrder(ctx, newOrderTest, o, &resp)
}

func (e *Exchange) newOrder(ctx context.Context, api string, o *NewOrderRequest, resp *NewOrderResponse) error {
	params := url.Values{}
	symbol, err := e.FormatSymbol(o.Symbol, asset.Spot)
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
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, api, params, spotOrderRate, resp)
}

// CancelExistingOrder sends a cancel order to Binance
func (e *Exchange) CancelExistingOrder(ctx context.Context, symbol currency.Pair, orderID int64, origClientOrderID string) (CancelOrderResponse, error) {
	var resp CancelOrderResponse

	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
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
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodDelete, orderEndpoint, params, spotOrderRate, &resp)
}

// OpenOrders Current open orders. Get all open orders on a symbol.
// Careful when accessing this with no symbol: The number of requests counted
// against the rate limiter is significantly higher
func (e *Exchange) OpenOrders(ctx context.Context, pair currency.Pair) ([]QueryOrderData, error) {
	var resp []QueryOrderData
	params := url.Values{}
	var p string
	var err error
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
	if err := e.SendAuthHTTPRequest(ctx,
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
func (e *Exchange) AllOrders(ctx context.Context, symbol currency.Pair, orderID, limit string) ([]QueryOrderData, error) {
	var resp []QueryOrderData

	params := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
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
	if err := e.SendAuthHTTPRequest(ctx,
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
func (e *Exchange) QueryOrder(ctx context.Context, symbol currency.Pair, origClientOrderID string, orderID int64) (QueryOrderData, error) {
	var resp QueryOrderData

	params := url.Values{}
	symbolValue, err := e.FormatSymbol(symbol, asset.Spot)
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

	if err := e.SendAuthHTTPRequest(ctx,
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
func (e *Exchange) GetAccount(ctx context.Context) (*Account, error) {
	type response struct {
		Response
		Account
	}

	var resp response
	params := url.Values{}

	if err := e.SendAuthHTTPRequest(ctx,
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
func (e *Exchange) GetMarginAccount(ctx context.Context) (*MarginAccount, error) {
	var resp MarginAccount
	params := url.Values{}

	if err := e.SendAuthHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		http.MethodGet, marginAccountInfo,
		params, spotAccountInformationRate,
		&resp); err != nil {
		return &resp, err
	}

	return &resp, nil
}

// SendHTTPRequest sends an unauthenticated request
func (e *Exchange) SendHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result any) error {
	endpointPath, err := e.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	item := &request.Item{
		Method:                 http.MethodGet,
		Path:                   endpointPath + path,
		Result:                 result,
		Verbose:                e.Verbose,
		HTTPDebugging:          e.HTTPDebugging,
		HTTPRecording:          e.HTTPRecording,
		HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
	}

	return e.SendPayload(ctx, f, func() (*request.Item, error) {
		return item, nil
	}, request.UnauthenticatedRequest)
}

// SendAPIKeyHTTPRequest is a special API request where the api key is
// appended to the headers without a secret
func (e *Exchange) SendAPIKeyHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result any) error {
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
		Method:                 http.MethodGet,
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

// SendAuthHTTPRequest sends an authenticated HTTP request
func (e *Exchange) SendAuthHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, params url.Values, f request.EndpointLimit, result any) error {
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
	a, err := e.GetAccount(ctx)
	if err != nil {
		return 0, err
	}
	if isMaker {
		return float64(a.MakerCommission), nil
	}
	return float64(a.TakerCommission), nil
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
func (e *Exchange) GetAllCoinsInfo(ctx context.Context) ([]CoinInfo, error) {
	var resp []CoinInfo
	if err := e.SendAuthHTTPRequest(ctx,
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
func (e *Exchange) WithdrawCrypto(ctx context.Context, cryptoAsset, withdrawOrderID, network, address, addressTag, name, amount string, transactionFeeFlag bool) (string, error) {
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
	if err := e.SendAuthHTTPRequest(ctx,
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
func (e *Exchange) DepositHistory(ctx context.Context, c currency.Code, status string, startTime, endTime time.Time, offset, limit int) ([]DepositHistory, error) {
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

	if err := e.SendAuthHTTPRequest(ctx,
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
	if err := e.SendAuthHTTPRequest(ctx,
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
func (e *Exchange) GetDepositAddressForCurrency(ctx context.Context, coin, chain string) (*DepositAddress, error) {
	params := url.Values{}
	params.Set("coin", coin)
	if chain != "" {
		params.Set("network", chain)
	}
	params.Set("recvWindow", "10000")
	var d DepositAddress
	return &d,
		e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, depositAddress, params, spotDefaultRate, &d)
}

// GetWsAuthStreamKey will retrieve a key to use for authorised WS streaming
func (e *Exchange) GetWsAuthStreamKey(ctx context.Context) (string, error) {
	endpointPath, err := e.API.Endpoints.GetURL(exchange.RestSpotSupplementary)
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
		Path:                   endpointPath + userAccountStream,
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
	endpointPath, err := e.API.Endpoints.GetURL(exchange.RestSpotSupplementary)
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

	path := endpointPath + userAccountStream
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

		for i := range s.PermissionSets {
			if !slices.Contains(s.PermissionSets[i], aUpper) {
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
			break
		}
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
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, loanIncomeHistory, params, spotDefaultRate, &resp)
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
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, loanBorrow, params, spotDefaultRate, &resp)
}

// CryptoLoanBorrowHistory gets loan borrow history
func (e *Exchange) CryptoLoanBorrowHistory(ctx context.Context, orderID int64, loanCoin, collateralCoin currency.Code, startTime, endTime time.Time, current, limit int64) (*LoanBorrowHistory, error) {
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
	return &resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, loanBorrowHistory, params, spotDefaultRate, &resp)
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

	var resp CryptoLoanOngoingOrder
	return &resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, loanOngoingOrders, params, spotDefaultRate, &resp)
}

// CryptoLoanRepay repays a crypto loan
func (e *Exchange) CryptoLoanRepay(ctx context.Context, orderID int64, amount float64, repayType int64, collateralReturn bool) ([]CryptoLoanRepay, error) {
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
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, loanRepay, params, spotDefaultRate, &resp)
}

// CryptoLoanRepaymentHistory gets the crypto loan repayment history
func (e *Exchange) CryptoLoanRepaymentHistory(ctx context.Context, orderID int64, loanCoin, collateralCoin currency.Code, startTime, endTime time.Time, current, limit int64) (*CryptoLoanRepayHistory, error) {
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
	return &resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, loanRepaymentHistory, params, spotDefaultRate, &resp)
}

// CryptoLoanAdjustLTV adjusts the LTV of a crypto loan
func (e *Exchange) CryptoLoanAdjustLTV(ctx context.Context, orderID int64, reduce bool, amount float64) (*CryptoLoanAdjustLTV, error) {
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
	return &resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, loanAdjustLTV, params, spotDefaultRate, &resp)
}

// CryptoLoanLTVAdjustmentHistory gets the crypto loan LTV adjustment history
func (e *Exchange) CryptoLoanLTVAdjustmentHistory(ctx context.Context, orderID int64, loanCoin, collateralCoin currency.Code, startTime, endTime time.Time, current, limit int64) (*CryptoLoanLTVAdjustmentHistory, error) {
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
	return &resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, loanLTVAdjustmentHistory, params, spotDefaultRate, &resp)
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

	var resp LoanableAssetsData
	return &resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, loanableAssetsData, params, spotDefaultRate, &resp)
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

	var resp CollateralAssetData
	return &resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, loanCollateralAssetsData, params, spotDefaultRate, &resp)
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
		return nil, errAmountMustBeSet
	}

	params := url.Values{}
	params.Set("loanCoin", loanCoin.String())
	params.Set("collateralCoin", collateralCoin.String())
	params.Set("repayAmount", strconv.FormatFloat(amount, 'f', -1, 64))

	var resp CollateralRepayRate
	return &resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, loanCheckCollateralRepayRate, params, spotDefaultRate, &resp)
}

// CryptoLoanCustomiseMarginCall customises a loan's margin call
func (e *Exchange) CryptoLoanCustomiseMarginCall(ctx context.Context, orderID int64, collateralCoin currency.Code, marginCallValue float64) (*CustomiseMarginCall, error) {
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
	return &resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, loanCustomiseMarginCall, params, spotDefaultRate, &resp)
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
	return &resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, flexibleLoanBorrow, params, spotDefaultRate, &resp)
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

	var resp FlexibleLoanOngoingOrder
	return &resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, flexibleLoanOngoingOrders, params, spotDefaultRate, &resp)
}

// FlexibleLoanBorrowHistory gets the flexible loan borrow history
func (e *Exchange) FlexibleLoanBorrowHistory(ctx context.Context, loanCoin, collateralCoin currency.Code, startTime, endTime time.Time, current, limit int64) (*FlexibleLoanBorrowHistory, error) {
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
	return &resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, flexibleLoanBorrowHistory, params, spotDefaultRate, &resp)
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
	return &resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, flexibleLoanRepay, params, spotDefaultRate, &resp)
}

// FlexibleLoanRepayHistory gets the flexible loan repayment history
func (e *Exchange) FlexibleLoanRepayHistory(ctx context.Context, loanCoin, collateralCoin currency.Code, startTime, endTime time.Time, current, limit int64) (*FlexibleLoanRepayHistory, error) {
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
	return &resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, flexibleLoanRepayHistory, params, spotDefaultRate, &resp)
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
	return &resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, flexibleLoanAdjustLTV, params, spotDefaultRate, &resp)
}

// FlexibleLoanLTVAdjustmentHistory gets the flexible loan LTV adjustment history
func (e *Exchange) FlexibleLoanLTVAdjustmentHistory(ctx context.Context, loanCoin, collateralCoin currency.Code, startTime, endTime time.Time, current, limit int64) (*FlexibleLoanLTVAdjustmentHistory, error) {
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
	return &resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, flexibleLoanLTVHistory, params, spotDefaultRate, &resp)
}

// FlexibleLoanAssetsData gets the flexible loan assets data
func (e *Exchange) FlexibleLoanAssetsData(ctx context.Context, loanCoin currency.Code) (*FlexibleLoanAssetsData, error) {
	params := url.Values{}
	if !loanCoin.IsEmpty() {
		params.Set("loanCoin", loanCoin.String())
	}

	var resp FlexibleLoanAssetsData
	return &resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, flexibleLoanAssetsData, params, spotDefaultRate, &resp)
}

// FlexibleCollateralAssetsData gets the flexible loan collateral assets data
func (e *Exchange) FlexibleCollateralAssetsData(ctx context.Context, collateralCoin currency.Code) (*FlexibleCollateralAssetsData, error) {
	params := url.Values{}
	if !collateralCoin.IsEmpty() {
		params.Set("collateralCoin", collateralCoin.String())
	}

	var resp FlexibleCollateralAssetsData
	return &resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, flexibleLoanCollateralAssetsData, params, spotDefaultRate, &resp)
}
