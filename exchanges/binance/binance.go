package binance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Binance is the overarching type across the Binance package
type Binance struct {
	exchange.Base
	// Valid string list that is required by the exchange
	validLimits []int
	obm         *orderbookManager
}

const (
	apiURL         = "https://api.binance.com"
	spotAPIURL     = "https://sapi.binance.com"
	cfuturesAPIURL = "https://dapi.binance.com"
	ufuturesAPIURL = "https://fapi.binance.com"

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

	// Authenticated endpoints
	newOrderTest      = "/api/v3/order/test"
	orderEndpoint     = "/api/v3/order"
	openOrders        = "/api/v3/openOrders"
	allOrders         = "/api/v3/allOrders"
	accountInfo       = "/api/v3/account"
	marginAccountInfo = "/sapi/v1/margin/account"

	// Withdraw API endpoints
	withdrawEndpoint                       = "/wapi/v3/withdraw.html"
	depositHistory                         = "/wapi/v3/depositHistory.html"
	withdrawalHistory                      = "/wapi/v3/withdrawHistory.html"
	depositAddress                         = "/wapi/v3/depositAddress.html"
	accountStatus                          = "/wapi/v3/accountStatus.html"
	systemStatus                           = "/wapi/v3/systemStatus.html"
	dustLog                                = "/wapi/v3/userAssetDribbletLog.html"
	tradeFee                               = "/wapi/v3/tradeFee.html"
	assetDetail                            = "/wapi/v3/assetDetail.html"
	undocumentedInterestHistory            = "/gateway-api/v1/public/isolated-margin/pair/vip-level"
	undocumentedCrossMarginInterestHistory = "/gateway-api/v1/friendly/margin/vip/spec/list-all"
)

// GetInterestHistory gets interest history for currency/currencies provided
func (b *Binance) GetInterestHistory() (MarginInfoData, error) {
	var resp MarginInfoData
	if err := b.SendHTTPRequest(exchange.EdgeCase1, undocumentedInterestHistory, spotDefaultRate, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetCrossMarginInterestHistory gets cross-margin interest history for currency/currencies provided
func (b *Binance) GetCrossMarginInterestHistory() (CrossMarginInterestData, error) {
	var resp CrossMarginInterestData
	if err := b.SendHTTPRequest(exchange.EdgeCase1, undocumentedCrossMarginInterestHistory, spotDefaultRate, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetMarginMarkets returns exchange information. Check binance_types for more information
func (b *Binance) GetMarginMarkets() (PerpsExchangeInfo, error) {
	var resp PerpsExchangeInfo
	return resp, b.SendHTTPRequest(exchange.RestSpot, perpExchangeInfo, spotDefaultRate, &resp)
}

// GetExchangeInfo returns exchange information. Check binance_types for more
// information
func (b *Binance) GetExchangeInfo() (ExchangeInfo, error) {
	var resp ExchangeInfo
	return resp, b.SendHTTPRequest(exchange.RestSpotSupplementary, exchangeInfo, spotExchangeInfo, &resp)
}

// GetOrderBook returns full orderbook information
//
// OrderBookDataRequestParams contains the following members
// symbol: string of currency pair
// limit: returned limit amount
func (b *Binance) GetOrderBook(obd OrderBookDataRequestParams) (OrderBook, error) {
	var orderbook OrderBook
	if err := b.CheckLimit(obd.Limit); err != nil {
		return orderbook, err
	}

	params := url.Values{}
	symbol, err := b.FormatSymbol(obd.Symbol, asset.Spot)
	if err != nil {
		return orderbook, err
	}
	params.Set("symbol", symbol)
	params.Set("limit", fmt.Sprintf("%d", obd.Limit))

	var resp OrderBookData
	if err := b.SendHTTPRequest(exchange.RestSpotSupplementary, orderBookDepth+"?"+params.Encode(), orderbookLimit(obd.Limit), &resp); err != nil {
		return orderbook, err
	}

	for x := range resp.Bids {
		price, err := strconv.ParseFloat(resp.Bids[x][0], 64)
		if err != nil {
			return orderbook, err
		}

		amount, err := strconv.ParseFloat(resp.Bids[x][1], 64)
		if err != nil {
			return orderbook, err
		}

		orderbook.Bids = append(orderbook.Bids, OrderbookItem{
			Price:    price,
			Quantity: amount,
		})
	}

	for x := range resp.Asks {
		price, err := strconv.ParseFloat(resp.Asks[x][0], 64)
		if err != nil {
			return orderbook, err
		}

		amount, err := strconv.ParseFloat(resp.Asks[x][1], 64)
		if err != nil {
			return orderbook, err
		}

		orderbook.Asks = append(orderbook.Asks, OrderbookItem{
			Price:    price,
			Quantity: amount,
		})
	}

	orderbook.LastUpdateID = resp.LastUpdateID
	return orderbook, nil
}

// GetMostRecentTrades returns recent trade activity
// limit: Up to 500 results returned
func (b *Binance) GetMostRecentTrades(rtr RecentTradeRequestParams) ([]RecentTrade, error) {
	var resp []RecentTrade

	params := url.Values{}
	symbol, err := b.FormatSymbol(rtr.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	params.Set("limit", fmt.Sprintf("%d", rtr.Limit))

	path := recentTrades + "?" + params.Encode()

	return resp, b.SendHTTPRequest(exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
}

// GetHistoricalTrades returns historical trade activity
//
// symbol: string of currency pair
// limit: Optional. Default 500; max 1000.
// fromID:
func (b *Binance) GetHistoricalTrades(symbol string, limit int, fromID int64) ([]HistoricalTrade, error) {
	var resp []HistoricalTrade
	params := url.Values{}

	params.Set("symbol", symbol)
	params.Set("limit", fmt.Sprintf("%d", limit))
	// else return most recent trades
	if fromID > 0 {
		params.Set("fromId", fmt.Sprintf("%d", fromID))
	}

	path := historicalTrades + "?" + params.Encode()
	return resp, b.SendAPIKeyHTTPRequest(exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
}

// GetAggregatedTrades returns aggregated trade activity.
// If more than one hour of data is requested or asked limit is not supported by exchange
// then the trades are collected with multiple backend requests.
// https://binance-docs.github.io/apidocs/spot/en/#compressed-aggregate-trades-list
func (b *Binance) GetAggregatedTrades(arg *AggregatedTradeRequestParams) ([]AggregatedTrade, error) {
	params := url.Values{}
	symbol, err := b.FormatSymbol(arg.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	// if the user request is directly not supported by the exchange, we might be able to fulfill it
	// by merging results from multiple API requests
	needBatch := false
	if arg.Limit > 0 {
		if arg.Limit > 1000 {
			// remote call doesn't support higher limits
			needBatch = true
		} else {
			params.Set("limit", strconv.Itoa(arg.Limit))
		}
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
		// fromId xor start time must be set
		canBatch := arg.FromID == 0 != arg.StartTime.IsZero()
		if canBatch {
			// Split the request into multiple
			return b.batchAggregateTrades(arg, params)
		}

		// Can't handle this request locally or remotely
		// We would receive {"code":-1128,"msg":"Combination of optional parameters invalid."}
		return nil, errors.New("please set StartTime or FromId, but not both")
	}
	var resp []AggregatedTrade
	path := aggregatedTrades + "?" + params.Encode()
	return resp, b.SendHTTPRequest(exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
}

// batchAggregateTrades fetches trades in multiple requests
// first phase, hourly requests until the first trade (or end time) is reached
// second phase, limit requests from previous trade until end time (or limit) is reached
func (b *Binance) batchAggregateTrades(arg *AggregatedTradeRequestParams, params url.Values) ([]AggregatedTrade, error) {
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
		for start := arg.StartTime; len(resp) == 0; start = start.Add(time.Hour) {
			if !arg.EndTime.IsZero() && !start.Before(arg.EndTime) {
				// All requests returned empty
				return nil, nil
			}
			params.Set("startTime", timeString(start))
			params.Set("endTime", timeString(start.Add(time.Hour)))
			path := aggregatedTrades + "?" + params.Encode()
			err := b.SendHTTPRequest(exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
			if err != nil {
				log.Warn(log.ExchangeSys, err.Error())
				return resp, err
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
		err := b.SendHTTPRequest(exchange.RestSpotSupplementary, path, spotDefaultRate, &additionalTrades)
		if err != nil {
			return resp, err
		}
		lastIndex := len(additionalTrades)
		if !arg.EndTime.IsZero() {
			// get index for truncating to end time
			lastIndex = sort.Search(len(additionalTrades), func(i int) bool {
				return arg.EndTime.Before(additionalTrades[i].TimeStamp)
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
// limit: optinal
// interval: the interval time for the data
// startTime: startTime filter for kline data
// endTime: endTime filter for the kline data
func (b *Binance) GetSpotKline(arg *KlinesRequestParams) ([]CandleStick, error) {
	var resp interface{}
	var klineData []CandleStick

	params := url.Values{}
	symbol, err := b.FormatSymbol(arg.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	params.Set("interval", arg.Interval)
	if arg.Limit != 0 {
		params.Set("limit", strconv.Itoa(arg.Limit))
	}
	if !arg.StartTime.IsZero() {
		params.Set("startTime", timeString(arg.StartTime))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", timeString(arg.EndTime))
	}

	path := candleStick + "?" + params.Encode()

	if err := b.SendHTTPRequest(exchange.RestSpotSupplementary, path, spotDefaultRate, &resp); err != nil {
		return klineData, err
	}

	for _, responseData := range resp.([]interface{}) {
		var candle CandleStick
		for i, individualData := range responseData.([]interface{}) {
			switch i {
			case 0:
				tempTime := individualData.(float64)
				var err error
				candle.OpenTime, err = convert.TimeFromUnixTimestampFloat(tempTime)
				if err != nil {
					return klineData, err
				}
			case 1:
				candle.Open, _ = strconv.ParseFloat(individualData.(string), 64)
			case 2:
				candle.High, _ = strconv.ParseFloat(individualData.(string), 64)
			case 3:
				candle.Low, _ = strconv.ParseFloat(individualData.(string), 64)
			case 4:
				candle.Close, _ = strconv.ParseFloat(individualData.(string), 64)
			case 5:
				candle.Volume, _ = strconv.ParseFloat(individualData.(string), 64)
			case 6:
				tempTime := individualData.(float64)
				var err error
				candle.CloseTime, err = convert.TimeFromUnixTimestampFloat(tempTime)
				if err != nil {
					return klineData, err
				}
			case 7:
				candle.QuoteAssetVolume, _ = strconv.ParseFloat(individualData.(string), 64)
			case 8:
				candle.TradeCount = individualData.(float64)
			case 9:
				candle.TakerBuyAssetVolume, _ = strconv.ParseFloat(individualData.(string), 64)
			case 10:
				candle.TakerBuyQuoteAssetVolume, _ = strconv.ParseFloat(individualData.(string), 64)
			}
		}
		klineData = append(klineData, candle)
	}
	return klineData, nil
}

// GetAveragePrice returns current average price for a symbol.
//
// symbol: string of currency pair
func (b *Binance) GetAveragePrice(symbol currency.Pair) (AveragePrice, error) {
	resp := AveragePrice{}
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)

	path := averagePrice + "?" + params.Encode()

	return resp, b.SendHTTPRequest(exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
}

// GetPriceChangeStats returns price change statistics for the last 24 hours
//
// symbol: string of currency pair
func (b *Binance) GetPriceChangeStats(symbol currency.Pair) (PriceChangeStats, error) {
	resp := PriceChangeStats{}
	params := url.Values{}
	rateLimit := spotPriceChangeAllRate
	if !symbol.IsEmpty() {
		rateLimit = spotDefaultRate
		symbolValue, err := b.FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	path := priceChange + "?" + params.Encode()

	return resp, b.SendHTTPRequest(exchange.RestSpotSupplementary, path, rateLimit, &resp)
}

// GetTickers returns the ticker data for the last 24 hrs
func (b *Binance) GetTickers() ([]PriceChangeStats, error) {
	var resp []PriceChangeStats
	return resp, b.SendHTTPRequest(exchange.RestSpotSupplementary, priceChange, spotPriceChangeAllRate, &resp)
}

// GetLatestSpotPrice returns latest spot price of symbol
//
// symbol: string of currency pair
func (b *Binance) GetLatestSpotPrice(symbol currency.Pair) (SymbolPrice, error) {
	resp := SymbolPrice{}
	params := url.Values{}
	rateLimit := spotSymbolPriceAllRate
	if !symbol.IsEmpty() {
		rateLimit = spotDefaultRate
		symbolValue, err := b.FormatSymbol(symbol, asset.Spot)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	path := symbolPrice + "?" + params.Encode()

	return resp, b.SendHTTPRequest(exchange.RestSpotSupplementary, path, rateLimit, &resp)
}

// GetBestPrice returns the latest best price for symbol
//
// symbol: string of currency pair
func (b *Binance) GetBestPrice(symbol currency.Pair) (BestPrice, error) {
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

	return resp, b.SendHTTPRequest(exchange.RestSpotSupplementary, path, rateLimit, &resp)
}

// NewOrder sends a new order to Binance
func (b *Binance) NewOrder(o *NewOrderRequest) (NewOrderResponse, error) {
	var resp NewOrderResponse
	if err := b.newOrder(orderEndpoint, o, &resp); err != nil {
		return resp, err
	}

	if resp.Code != 0 {
		return resp, errors.New(resp.Msg)
	}

	return resp, nil
}

// NewOrderTest sends a new test order to Binance
func (b *Binance) NewOrderTest(o *NewOrderRequest) error {
	var resp NewOrderResponse
	return b.newOrder(newOrderTest, o, &resp)
}

func (b *Binance) newOrder(api string, o *NewOrderRequest, resp *NewOrderResponse) error {
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
	return b.SendAuthHTTPRequest(exchange.RestSpotSupplementary, http.MethodPost, api, params, spotOrderRate, resp)
}

// CancelExistingOrder sends a cancel order to Binance
func (b *Binance) CancelExistingOrder(symbol currency.Pair, orderID int64, origClientOrderID string) (CancelOrderResponse, error) {
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
	return resp, b.SendAuthHTTPRequest(exchange.RestSpotSupplementary, http.MethodDelete, orderEndpoint, params, spotOrderRate, &resp)
}

// OpenOrders Current open orders. Get all open orders on a symbol.
// Careful when accessing this with no symbol: The number of requests counted against the rate limiter
// is significantly higher
func (b *Binance) OpenOrders(pair currency.Pair) ([]QueryOrderData, error) {
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
		// extend the receive window when all currencies to prevent "recvwindow" error
		params.Set("recvWindow", "10000")
	}
	if err := b.SendAuthHTTPRequest(exchange.RestSpotSupplementary, http.MethodGet, openOrders, params, openOrdersLimit(p), &resp); err != nil {
		return resp, err
	}

	return resp, nil
}

// AllOrders Get all account orders; active, canceled, or filled.
// orderId optional param
// limit optional param, default 500; max 500
func (b *Binance) AllOrders(symbol currency.Pair, orderID, limit string) ([]QueryOrderData, error) {
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
	if err := b.SendAuthHTTPRequest(exchange.RestSpotSupplementary, http.MethodGet, allOrders, params, spotAllOrdersRate, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// QueryOrder returns information on a past order
func (b *Binance) QueryOrder(symbol currency.Pair, origClientOrderID string, orderID int64) (QueryOrderData, error) {
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

	if err := b.SendAuthHTTPRequest(exchange.RestSpotSupplementary, http.MethodGet, orderEndpoint, params, spotOrderQueryRate, &resp); err != nil {
		return resp, err
	}

	if resp.Code != 0 {
		return resp, errors.New(resp.Msg)
	}
	return resp, nil
}

// GetAccount returns binance user accounts
func (b *Binance) GetAccount() (*Account, error) {
	type response struct {
		Response
		Account
	}

	var resp response
	params := url.Values{}

	if err := b.SendAuthHTTPRequest(exchange.RestSpotSupplementary, http.MethodGet, accountInfo, params, spotAccountInformationRate, &resp); err != nil {
		return &resp.Account, err
	}

	if resp.Code != 0 {
		return &resp.Account, errors.New(resp.Msg)
	}

	return &resp.Account, nil
}

// GetMarginAccount returns account information for margin accounts
func (b *Binance) GetMarginAccount() (*MarginAccount, error) {
	var resp MarginAccount
	params := url.Values{}

	if err := b.SendAuthHTTPRequest(exchange.RestSpotSupplementary, http.MethodGet, marginAccountInfo, params, spotAccountInformationRate, &resp); err != nil {
		return &resp, err
	}

	return &resp, nil
}

// SendHTTPRequest sends an unauthenticated request
func (b *Binance) SendHTTPRequest(ePath exchange.URL, path string, f request.EndpointLimit, result interface{}) error {
	endpointPath, err := b.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	return b.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodGet,
		Path:          endpointPath + path,
		Result:        result,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
		Endpoint:      f})
}

// SendAPIKeyHTTPRequest is a special API request where the api key is
// appended to the headers without a secret
func (b *Binance) SendAPIKeyHTTPRequest(ePath exchange.URL, path string, f request.EndpointLimit, result interface{}) error {
	endpointPath, err := b.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	headers := make(map[string]string)
	headers["X-MBX-APIKEY"] = b.API.Credentials.Key
	return b.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodGet,
		Path:          endpointPath + path,
		Headers:       headers,
		Result:        result,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
		Endpoint:      f})
}

// SendAuthHTTPRequest sends an authenticated HTTP request
func (b *Binance) SendAuthHTTPRequest(ePath exchange.URL, method, path string, params url.Values, f request.EndpointLimit, result interface{}) error {
	if !b.AllowAuthenticatedRequest() {
		return fmt.Errorf("%s %w", b.Name, exchange.ErrAuthenticatedRequestWithoutCredentialsSet)
	}
	endpointPath, err := b.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}
	path = endpointPath + path
	if params == nil {
		params = url.Values{}
	}
	recvWindow := 5 * time.Second
	if params.Get("recvWindow") != "" {
		// convert recvWindow value into time.Duration
		var recvWindowParam int64
		recvWindowParam, err = convert.Int64FromString(params.Get("recvWindow"))
		if err != nil {
			return err
		}
		recvWindow = time.Duration(recvWindowParam) * time.Millisecond
	} else {
		params.Set("recvWindow", strconv.FormatInt(convert.RecvWindow(recvWindow), 10))
	}
	params.Set("recvWindow", strconv.FormatInt(convert.RecvWindow(recvWindow), 10))
	params.Set("timestamp", strconv.FormatInt(time.Now().Unix()*1000, 10))
	signature := params.Encode()
	hmacSigned := crypto.GetHMAC(crypto.HashSHA256, []byte(signature), []byte(b.API.Credentials.Secret))
	hmacSignedStr := crypto.HexEncodeToString(hmacSigned)
	headers := make(map[string]string)
	headers["X-MBX-APIKEY"] = b.API.Credentials.Key
	if b.Verbose {
		log.Debugf(log.ExchangeSys, "sent path: %s", path)
	}

	path = common.EncodeURLValues(path, params)
	path += "&signature=" + hmacSignedStr
	interim := json.RawMessage{}
	errCap := struct {
		Success bool   `json:"success"`
		Message string `json:"msg"`
		Code    int64  `json:"code"`
	}{}
	ctx, cancel := context.WithTimeout(context.Background(), recvWindow)
	defer cancel()
	err = b.SendPayload(ctx, &request.Item{
		Method:        method,
		Path:          path,
		Headers:       headers,
		Body:          bytes.NewBuffer(nil),
		Result:        &interim,
		AuthRequest:   true,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
		Endpoint:      f})
	if err != nil {
		return err
	}
	if err := json.Unmarshal(interim, &errCap); err == nil {
		if !errCap.Success && errCap.Message != "" && errCap.Code != 200 {
			return errors.New(errCap.Message)
		}
	}
	return json.Unmarshal(interim, result)
}

// CheckLimit checks value against a variable list
func (b *Binance) CheckLimit(limit int) error {
	for x := range b.validLimits {
		if b.validLimits[x] == limit {
			return nil
		}
	}
	return errors.New("incorrect limit values - valid values are 5, 10, 20, 50, 100, 500, 1000")
}

// SetValues sets the default valid values
func (b *Binance) SetValues() {
	b.validLimits = []int{5, 10, 20, 50, 100, 500, 1000, 5000}
}

// GetFee returns an estimate of fee based on type of transaction
func (b *Binance) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		multiplier, err := b.getMultiplier(feeBuilder.IsMaker)
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
func (b *Binance) getMultiplier(isMaker bool) (float64, error) {
	var multiplier float64
	account, err := b.GetAccount()
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

// calculateTradingFee returns the fee for trading any currency on Bittrex
func calculateTradingFee(purchasePrice, amount, multiplier float64) float64 {
	return (multiplier / 100) * purchasePrice * amount
}

// getCryptocurrencyWithdrawalFee returns the fee for withdrawing from the exchange
func getCryptocurrencyWithdrawalFee(c currency.Code) float64 {
	return WithdrawalFees[c]
}

// WithdrawCrypto sends cryptocurrency to the address of your choosing
func (b *Binance) WithdrawCrypto(asset, address, addressTag, name, amount string) (string, error) {
	var resp WithdrawResponse

	params := url.Values{}
	params.Set("asset", asset)
	params.Set("address", address)
	params.Set("amount", amount)
	if len(name) > 0 {
		params.Set("name", name)
	}
	if len(addressTag) > 0 {
		params.Set("addressTag", addressTag)
	}

	if err := b.SendAuthHTTPRequest(exchange.RestSpotSupplementary, http.MethodPost, withdrawEndpoint, params, spotDefaultRate, &resp); err != nil {
		return "", err
	}

	if !resp.Success {
		return resp.ID, errors.New(resp.Msg)
	}

	return resp.ID, nil
}

// WithdrawStatus gets the status of recent withdrawals
// status `param` used as string to prevent default value 0 (for int) interpreting as EmailSent status
func (b *Binance) WithdrawStatus(c currency.Code, status string, startTime, endTime int64) ([]WithdrawStatusResponse, error) {
	var response struct {
		Success      bool                     `json:"success"`
		WithdrawList []WithdrawStatusResponse `json:"withdrawList"`
	}

	params := url.Values{}
	params.Set("asset", c.String())

	if status != "" {
		i, err := strconv.Atoi(status)
		if err != nil {
			return response.WithdrawList, fmt.Errorf("wrong param (status): %s. Error: %v", status, err)
		}

		switch i {
		case EmailSent, Cancelled, AwaitingApproval, Rejected, Processing, Failure, Completed:
		default:
			return response.WithdrawList, fmt.Errorf("wrong param (status): %s", status)
		}

		params.Set("status", status)
	}

	if startTime > 0 {
		params.Set("startTime", strconv.FormatInt(startTime, 10))
	}

	if endTime > 0 {
		params.Set("endTime", strconv.FormatInt(endTime, 10))
	}

	if err := b.SendAuthHTTPRequest(exchange.RestSpotSupplementary, http.MethodGet, withdrawalHistory, params, spotDefaultRate, &response); err != nil {
		return response.WithdrawList, err
	}

	return response.WithdrawList, nil
}

// GetDepositAddressForCurrency retrieves the wallet address for a given currency
func (b *Binance) GetDepositAddressForCurrency(currency string) (string, error) {
	resp := struct {
		Address    string `json:"address"`
		Success    bool   `json:"success"`
		AddressTag string `json:"addressTag"`
	}{}

	params := url.Values{}
	params.Set("asset", currency)
	params.Set("status", "true")
	params.Set("recvWindow", "10000")

	return resp.Address,
		b.SendAuthHTTPRequest(exchange.RestSpotSupplementary, http.MethodGet, depositAddress, params, spotDefaultRate, &resp)
}

// GetWsAuthStreamKey will retrieve a key to use for authorised WS streaming
func (b *Binance) GetWsAuthStreamKey() (string, error) {
	endpointPath, err := b.API.Endpoints.GetURL(exchange.RestSpotSupplementary)
	if err != nil {
		return "", err
	}
	var resp UserAccountStream
	path := endpointPath + userAccountStream
	headers := make(map[string]string)
	headers["X-MBX-APIKEY"] = b.API.Credentials.Key
	err = b.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodPost,
		Path:          path,
		Headers:       headers,
		Body:          bytes.NewBuffer(nil),
		Result:        &resp,
		AuthRequest:   true,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
	})
	if err != nil {
		return "", err
	}
	return resp.ListenKey, nil
}

// MaintainWsAuthStreamKey will keep the key alive
func (b *Binance) MaintainWsAuthStreamKey() error {
	endpointPath, err := b.API.Endpoints.GetURL(exchange.RestSpotSupplementary)
	if err != nil {
		return err
	}
	if listenKey == "" {
		listenKey, err = b.GetWsAuthStreamKey()
		return err
	}
	path := endpointPath + userAccountStream
	params := url.Values{}
	params.Set("listenKey", listenKey)
	path = common.EncodeURLValues(path, params)
	headers := make(map[string]string)
	headers["X-MBX-APIKEY"] = b.API.Credentials.Key
	return b.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodPut,
		Path:          path,
		Headers:       headers,
		Body:          bytes.NewBuffer(nil),
		AuthRequest:   true,
		Verbose:       b.Verbose,
		HTTPDebugging: b.HTTPDebugging,
		HTTPRecording: b.HTTPRecording,
	})
}

// FetchSpotExchangeLimits fetches spot order execution limits
func (b *Binance) FetchSpotExchangeLimits() ([]order.MinMaxLevel, error) {
	var limits []order.MinMaxLevel
	spot, err := b.GetExchangeInfo()
	if err != nil {
		return nil, err
	}

	for x := range spot.Symbols {
		var cp currency.Pair
		cp, err = currency.NewPairFromStrings(spot.Symbols[x].BaseAsset,
			spot.Symbols[x].QuoteAsset)
		if err != nil {
			return nil, err
		}
		var assets []asset.Item
		for y := range spot.Symbols[x].Permissions {
			switch spot.Symbols[x].Permissions[y] {
			case "SPOT":
				assets = append(assets, asset.Spot)
			case "MARGIN":
				assets = append(assets, asset.Margin)
			case "LEVERAGED": // leveraged tokens not available for spot trading
			default:
				return nil, fmt.Errorf("unhandled asset type for exchange limits loading %s",
					spot.Symbols[x].Permissions[y])
			}
		}

		for z := range assets {
			if len(spot.Symbols[x].Filters) < 8 {
				continue
			}

			limits = append(limits, order.MinMaxLevel{
				Pair:                cp,
				Asset:               assets[z],
				MinPrice:            spot.Symbols[x].Filters[0].MinPrice,
				MaxPrice:            spot.Symbols[x].Filters[0].MaxPrice,
				StepPrice:           spot.Symbols[x].Filters[0].TickSize,
				MultiplierUp:        spot.Symbols[x].Filters[1].MultiplierUp,
				MultiplierDown:      spot.Symbols[x].Filters[1].MultiplierDown,
				AveragePriceMinutes: spot.Symbols[x].Filters[1].AvgPriceMinutes,
				MaxAmount:           spot.Symbols[x].Filters[2].MaxQty,
				MinAmount:           spot.Symbols[x].Filters[2].MinQty,
				StepAmount:          spot.Symbols[x].Filters[2].StepSize,
				MinNotional:         spot.Symbols[x].Filters[3].MinNotional,
				MaxIcebergParts:     spot.Symbols[x].Filters[4].Limit,
				MarketMinQty:        spot.Symbols[x].Filters[5].MinQty,
				MarketMaxQty:        spot.Symbols[x].Filters[5].MaxQty,
				MarketStepSize:      spot.Symbols[x].Filters[5].StepSize,
				MaxTotalOrders:      spot.Symbols[x].Filters[6].MaxNumOrders,
				MaxAlgoOrders:       spot.Symbols[x].Filters[7].MaxNumAlgoOrders,
			})
		}
	}
	return limits, nil
}
