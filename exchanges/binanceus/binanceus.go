package binanceus

import (
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Binanceus is the overarching type across this package
type Binanceus struct {
	validLimits []int
	exchange.Base
}

const (
	binanceusAPIURL     = "https://api.binance.us"
	binanceusAPIVersion = "/v3"

	// Public endpoints
	exchangeInfo     = "/api/v3/exchangeInfo"
	recentTrades     = "/api/v3/trades"
	aggregatedTrades = "/api/v3/aggTrades"
	orderBookDepth   = "/api/v3/depth"
	candleStick      = "/api/v3/klines"
	tickerPrice      = "/api/v3/ticker/price"
	averagePrice     = "/api/v3/avgPrice"
	bestPrice        = "/api/v3/ticker/bookTicker"
	priceChange      = "/api/v3/ticker/24hr"

	historicalTrades = "/api/v3/historicalTrades"

	// Withdraw API endpoints
	accountStatus = "/wapi/v3/accountStatus.html"
	tradingStatus = "/wapi/v3/apiTradingStatus.html"

	accountInfo = "/api/v3/account"

	// Authenticated endpoints

	// Other Consts
	defaultRecvWindow = 5 * time.Second
)

// Start implementing public and private exchange API funcs below
func (b *Binanceus) GetExchangeInfo(ctx context.Context) (ExchangeInfo, error) {
	var respo ExchangeInfo
	return respo, b.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, exchangeInfo, spotExchangeInfo, &respo)
}

func (b *Binanceus) GetMostRecentTrades(ctx context.Context, rtr RecentTradeRequestParams) ([]RecentTrade, error) {
	params := url.Values{}
	symbol, err := b.FormatSymbol(rtr.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	params.Set("limit", fmt.Sprintf("%d", rtr.Limit))
	path := recentTrades + "?" + params.Encode()
	var resp []RecentTrade
	return resp, b.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
}

// GetHistoricalTrades returns historical trade activity
//
// symbol: string of currency pair
// limit: Optional. Default 500; max 1000.
func (b *Binanceus) GetHistoricalTrades(ctx context.Context, hist HistoricalTradeParams) ([]HistoricalTrade, error) {
	var resp []HistoricalTrade
	params := url.Values{}
	params.Set("symbol", hist.Symbol)
	params.Set("limit", fmt.Sprintf("%d", hist.Limit))
	if hist.FromID > 0 {
		params.Set("fromId", fmt.Sprintf("%d", hist.FromID))
	}
	path := historicalTrades + "?" + params.Encode()
	return resp, b.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
}

func (b *Binanceus) GetAggregateTrades(ctx context.Context, agg *AggregatedTradeRequestParams) ([]AggregatedTrade, error) {
	params := url.Values{}
	symbol, err := b.FormatSymbol(agg.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	needBatch := false
	if agg.Limit > 0 {
		if agg.Limit > 1000 {
			needBatch = true
		} else {
			params.Set("limit", strconv.Itoa(agg.Limit))
		}
	}
	if agg.FromID != 0 {
		params.Set("fromId", strconv.FormatInt(agg.FromID, 10))
	}
	startTime := time.UnixMilli(int64(agg.StartTime))
	endTime := time.UnixMilli(int64(agg.EndTime))

	if (endTime.UnixNano() - startTime.UnixNano()) >= int64(time.Hour) {
		endTime = startTime.Add(time.Minute * 59)
	}

	if !startTime.IsZero() {
		params.Set("startTime", strconv.Itoa(int(agg.StartTime)))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.Itoa(int(agg.EndTime)))
	}
	needBatch = needBatch || (!startTime.IsZero() && !endTime.IsZero() && endTime.Sub(startTime) > time.Hour)
	if needBatch {
		// fromId xor start time must be set
		canBatch := agg.FromID == 0 != startTime.IsZero()
		if canBatch {
			return b.batchAggregateTrades(ctx, agg, params)
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

// batchAggregateTrades fetches trades in multiple requests   <-- copied and amended from the  binance
// first phase, hourly requests until the first trade (or end time) is reached
// second phase, limit requests from previous trade until end time (or limit) is reached
func (b *Binanceus) batchAggregateTrades(ctx context.Context, arg *AggregatedTradeRequestParams, params url.Values) ([]AggregatedTrade, error) {
	var resp []AggregatedTrade
	// prepare first request with only first hour and max limit
	if arg.Limit == 0 || arg.Limit > 1000 {
		// Extend from the default of 500
		params.Set("limit", "1000")
	}
	startTime := time.UnixMilli(int64(arg.StartTime))
	endTime := time.UnixMilli(int64(arg.EndTime))
	var fromID int64
	if arg.FromID > 0 {
		fromID = arg.FromID
	} else {
		// Only 10 seconds is used to prevent limit of 1000 being reached in the first request,
		// cutting off trades for high activity pairs
		increment := time.Second * 10
		for len(resp) == 0 {
			startTime = startTime.Add(increment)
			if !endTime.IsZero() && !startTime.Before(endTime) {
				// All requests returned empty
				return nil, nil
			}
			params.Set("startTime", strconv.Itoa(int(startTime.UnixMilli())))
			params.Set("endTime", strconv.Itoa(int(startTime.Add(increment).UnixMilli())))
			path := aggregatedTrades + "?" + params.Encode()
			println(path)
			err := b.SendHTTPRequest(ctx,
				exchange.RestSpotSupplementary, path, spotDefaultRate, &resp)
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
		err := b.SendHTTPRequest(ctx,
			exchange.RestSpotSupplementary,
			path,
			spotDefaultRate,
			&additionalTrades)
		if err != nil {
			return resp, err
		}
		lastIndex := len(additionalTrades)
		if !endTime.IsZero() {
			// get index for truncating to end time
			lastIndex = sort.Search(len(additionalTrades), func(i int) bool {
				return endTime.Before(time.UnixMilli(int64(additionalTrades[i].TimeStamp)))
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

func (b *Binanceus) GetOrderBookDepth(ctx context.Context, arg *OrderBookDataRequestParams) (*OrderBook, error) {
	if err := b.CheckLimit(arg.Limit); err != nil {
		return nil, err
	}
	params := url.Values{}
	symbol, err := b.FormatSymbol(arg.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	params.Set("limit", fmt.Sprintf("%d", arg.Limit))
	var resp OrderBookData
	if err := b.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		orderBookDepth+"?"+params.Encode(),
		orderbookLimit(arg.Limit), &resp); err != nil {
		return nil, err
	}
	orderbook := OrderBook{
		Bids:         make([]OrderbookItem, len(resp.Bids)),
		Asks:         make([]OrderbookItem, len(resp.Asks)),
		LastUpdateID: resp.LastUpdateID,
	}
	for x := range resp.Bids {
		price, err := strconv.ParseFloat(resp.Bids[x][0], 64)
		if err != nil {
			return nil, err
		}
		amount, err := strconv.ParseFloat(resp.Bids[x][1], 64)
		if err != nil {
			return nil, err
		}
		orderbook.Bids[x] = OrderbookItem{
			Price:    price,
			Quantity: amount,
		}
	}
	for x := range resp.Asks {
		price, err := strconv.ParseFloat(resp.Asks[x][0], 64)
		if err != nil {
			return nil, err
		}
		amount, err := strconv.ParseFloat(resp.Asks[x][1], 64)
		if err != nil {
			return nil, err
		}
		orderbook.Asks[x] = OrderbookItem{
			Price:    price,
			Quantity: amount,
		}
	}
	return &orderbook, nil
}

// CheckLimit checks value against a variable list
func (b *Binanceus) CheckLimit(limit int) error {
	for x := range b.validLimits {
		if b.validLimits[x] == limit {
			return nil
		}
	}
	return errors.New("incorrect limit values - valid values are 5, 10, 20, 50, 100, 500, 1000")
}

func (b *Binanceus) GetSpotKline(ctx context.Context, arg *KlinesRequestParams) ([]CandleStick, error) {
	symbol, err := b.FormatSymbol(arg.Symbol, asset.Spot)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", arg.Interval)
	if arg.Limit != 0 {
		params.Set("limit", strconv.Itoa(arg.Limit))
	}
	if !arg.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt((arg.StartTime).UnixMilli(), 10))
	}
	if !arg.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt((arg.EndTime).UnixMilli(), 10))
	}

	path := candleStick + "?" + params.Encode()
	var resp interface{}

	err = b.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary,
		path,
		spotDefaultRate,
		&resp)
	if err != nil {
		return nil, err
	}
	responseData, ok := resp.([]interface{})
	if !ok {
		return nil, errors.New("unable to type assert responseData")
	}

	klineData := make([]CandleStick, len(responseData))
	for x := range responseData {
		individualData, ok := responseData[x].([]interface{})
		if !ok {
			return nil, errors.New("unable to type assert individualData")
		}
		if len(individualData) != 12 {
			return nil, errors.New("unexpected kline data length")
		}
		var candle CandleStick
		candle.OpenTime, ok = individualData[0].(float64)
		if !ok {
			return nil, errors.New("invalid open time error")
		}
		if candle.Open, err = convert.FloatFromString(individualData[1]); err != nil {
			return nil, err
		}
		if candle.High, err = convert.FloatFromString(individualData[2]); err != nil {
			return nil, err
		}
		if candle.Low, err = convert.FloatFromString(individualData[3]); err != nil {
			return nil, err
		}
		if candle.Close, err = convert.FloatFromString(individualData[4]); err != nil {
			return nil, err
		}
		if candle.Volume, err = convert.FloatFromString(individualData[5]); err != nil {
			return nil, err
		}
		if candle.CloseTime, ok = individualData[6].(float64); !ok {
			return nil, errors.New("invalid close time error")
		}
		if candle.QuoteAssetVolume, err = convert.FloatFromString(individualData[7]); err != nil {
			return nil, err
		}
		if candle.TradeCount, ok = individualData[8].(float64); !ok {
			return nil, errors.New("unable to type assert trade count")
		}
		if candle.TakerBuyAssetVolume, err = convert.FloatFromString(individualData[9]); err != nil {
			return nil, err
		}
		if candle.TakerBuyQuoteAssetVolume, err = convert.FloatFromString(individualData[10]); err != nil {
			return nil, err
		}
		klineData[x] = candle
	}
	return klineData, nil
}

func (b *Binanceus) GetSinglePriceData(ctx context.Context, symbol currency.Pair) (SymbolPrice, error) {
	var res SymbolPrice
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.Spot)
	if err != nil {
		return res, err
	}
	params.Set("symbol", symbolValue)
	path := tickerPrice + "?" + params.Encode()
	return res, b.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, path, spotDefaultRate, &res)
}

func (b *Binanceus) GetPriceDatas(ctx context.Context) (SymbolPrices, error) {
	var res SymbolPrices
	return res, b.SendHTTPRequest(ctx, exchange.RestSpotSupplementary, tickerPrice, spotDefaultRate, &res)
}

// GetAveragePrice returns current average price for a symbol.
//
// symbol: string of currency pair
func (b *Binanceus) GetAveragePrice(ctx context.Context, symbol currency.Pair) (AveragePrice, error) {
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

// GetBestPrice returns the latest best price for symbol
//
// symbol: string of currency pair
func (b *Binanceus) GetBestPrice(ctx context.Context, symbol currency.Pair) (BestPrice, error) {
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

// GetPriceChangeStats returns price change statistics for the last 24 hours
//
// symbol: string of currency pair
func (b *Binanceus) GetPriceChangeStats(ctx context.Context, symbol currency.Pair) (PriceChangeStats, error) {
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

	return resp, b.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary, path, rateLimit, &resp)
}

// GetTickers returns the ticker data for the last 24 hrs
func (b *Binanceus) GetTickers(ctx context.Context) ([]PriceChangeStats, error) {
	var resp []PriceChangeStats
	return resp, b.SendHTTPRequest(ctx,
		exchange.RestSpotSupplementary, priceChange, spotPriceChangeAllRate, &resp)
}

// GetAccount returns binance user accounts
func (b *Binanceus) GetAccount(ctx context.Context) (*Account, error) {
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

func (b *Binanceus) GetUserAccountStatus(ctx context.Context, recvWindow uint) (*AccountStatusResponse, error) {
	var resp AccountStatusResponse
	params := url.Values{}
	timestamp := time.Now().UnixMilli()
	params.Set("timestamp", strconv.Itoa(int(timestamp)))
	if recvWindow > 0 {
		if recvWindow < 2000 {
			recvWindow += 1500
		}
	} else {
		recvWindow = uint(defaultRecvWindow)
	}
	params.Set("recvWindow", strconv.Itoa(int(recvWindow)))
	return &resp,
		b.SendAuthHTTPRequest(ctx,
			exchange.RestSpotSupplementary,
			http.MethodGet,
			accountStatus,
			params,
			spotAccountInformationRate,
			&resp)
}

func (b *Binanceus) GetUserAPITradingStatus(ctx context.Context, recvWindow uint) (*TradeStatus, error) {
	type response struct {
		Success bool        `json:"success"`
		TC      TradeStatus `json:"status"`
	}
	var resp response
	params := url.Values{}
	timestamp := time.Now().UnixMilli()
	params.Set("timestamp", strconv.Itoa(int(timestamp)))
	if recvWindow > 0 {
		if recvWindow < 2000 {
			recvWindow += 1500
		}
	} else {
		recvWindow = uint(defaultRecvWindow)
	}
	params.Set("recvWindow", strconv.Itoa(int(recvWindow)))
	return &(resp.TC),
		b.SendAuthHTTPRequest(ctx,
			exchange.RestSpotSupplementary,
			http.MethodGet,
			tradingStatus,
			params,
			spotAccountInformationRate,
			&resp)
}

func (b *Binanceus) SendHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result interface{}) error {
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
	})
}

func (b *Binanceus) SendAPIKeyHTTPRequest(ctx context.Context, ePath exchange.URL, path string, f request.EndpointLimit, result interface{}) error {
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
	})
}

// SendAuthHTTPRequest sends an authenticated HTTP request
func (b *Binanceus) SendAuthHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, params url.Values, f request.EndpointLimit, result interface{}) error {
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
			AuthRequest:   true,
			Verbose:       b.Verbose,
			HTTPDebugging: b.HTTPDebugging,
			HTTPRecording: b.HTTPRecording}, nil
	})
	if err != nil {
		return err
	}
	errCap := struct {
		Success bool   `json:"success"`
		Message string `json:"msg"`
		Code    int64  `json:"code"`
	}{}

	println(interim)
	if err := json.Unmarshal(interim, &errCap); err == nil {
		if !errCap.Success && errCap.Message != "" && errCap.Code != 200 {
			return errors.New(errCap.Message)
		}
	}
	return json.Unmarshal(interim, result)
}
