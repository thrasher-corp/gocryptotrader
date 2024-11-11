package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (

	// Unauth
	cfuturesExchangeInfo       = "/dapi/v1/exchangeInfo?"
	cfuturesOrderbook          = "/dapi/v1/depth?"
	cfuturesRecentTrades       = "/dapi/v1/trades?"
	cfuturesHistoricalTrades   = "/dapi/v1/historicalTrades"
	cfuturesCompressedTrades   = "/dapi/v1/aggTrades?"
	cfuturesKlineData          = "/dapi/v1/klines?"
	cfuturesContinuousKline    = "/dapi/v1/continuousKlines?"
	cfuturesIndexKline         = "/dapi/v1/indexPriceKlines?"
	cfuturesMarkPriceKline     = "/dapi/v1/markPriceKlines?"
	cfuturesFundingRateInfo    = "/dapi/v1/fundingInfo?"
	cfuturesMarkPrice          = "/dapi/v1/premiumIndex?"
	cfuturesFundingRateHistory = "/dapi/v1/fundingRate?"
	cfuturesTickerPriceStats   = "/dapi/v1/ticker/24hr?"
	cfuturesSymbolPriceTicker  = "/dapi/v1/ticker/price?"
	cfuturesSymbolOrderbook    = "/dapi/v1/ticker/bookTicker?"
	cfuturesOpenInterest       = "/dapi/v1/openInterest?"
	cfuturesOpenInterestStats  = "/futures/data/openInterestHist?"
	cfuturesTopAccountsRatio   = "/futures/data/topLongShortAccountRatio?"
	cfuturesTopPositionsRatio  = "/futures/data/topLongShortPositionRatio?"
	cfuturesLongShortRatio     = "/futures/data/globalLongShortAccountRatio?"
	cfuturesBuySellVolume      = "/futures/data/takerBuySellVol?"
	cfuturesBasis              = "/futures/data/basis?"

	// Auth
	cfuturesOrder                 = "/dapi/v1/order"
	cfuturesBatchOrder            = "/dapi/v1/batchOrders"
	cfuturesCancelAllOrders       = "/dapi/v1/allOpenOrders"
	cfuturesCountdownCancel       = "/dapi/v1/countdownCancelAll"
	cfuturesOpenOrder             = "/dapi/v1/openOrder"
	cfuturesAllOpenOrders         = "/dapi/v1/openOrders"
	cfuturesAllOrders             = "/dapi/v1/allOrders"
	cfuturesAccountBalance        = "/dapi/v1/balance"
	cfuturesAccountInfo           = "/dapi/v1/account"
	cfuturesChangeInitialLeverage = "/dapi/v1/leverage"
	cfuturesChangeMarginType      = "/dapi/v1/marginType"
	cfuturesModifyMargin          = "/dapi/v1/positionMargin"
	cfuturesMarginChangeHistory   = "/dapi/v1/positionMargin/history"
	cfuturesPositionInfo          = "/dapi/v1/positionRisk"
	cfuturesAccountTradeList      = "/dapi/v1/userTrades"
	cfuturesIncomeHistory         = "/dapi/v1/income"
	cfuturesNotionalBracket       = "/dapi/v1/leverageBracket"
	cfuturesUsersForceOrders      = "/dapi/v1/forceOrders"
	cfuturesADLQuantile           = "/dapi/v1/adlQuantile"

	cfuturesLimit              = "LIMIT"
	cfuturesMarket             = "MARKET"
	cfuturesStop               = "STOP"
	cfuturesTakeProfit         = "TAKE_PROFIT"
	cfuturesStopMarket         = "STOP_MARKET"
	cfuturesTakeProfitMarket   = "TAKE_PROFIT_MARKET"
	cfuturesTrailingStopMarket = "TRAILING_STOP_MARKET"
)

// FuturesExchangeInfo stores CoinMarginedFutures, data
func (b *Binance) FuturesExchangeInfo(ctx context.Context) (CExchangeInfo, error) {
	var resp CExchangeInfo
	return resp, b.SendHTTPRequest(ctx, exchange.RestCoinMargined, cfuturesExchangeInfo, cFuturesDefaultRate, &resp)
}

// GetFuturesOrderbook gets orderbook data for CoinMarginedFutures,
func (b *Binance) GetFuturesOrderbook(ctx context.Context, symbol currency.Pair, limit int64) (*OrderBook, error) {
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("symbol", symbolValue)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}

	rateBudget := cFuturesOrderbook1000Rate
	switch {
	case limit == 5, limit == 10, limit == 20, limit == 50:
		rateBudget = cFuturesOrderbook50Rate
	case limit >= 100 && limit < 500:
		rateBudget = cFuturesOrderbook100Rate
	case limit == 0, limit >= 500 && limit < 1000:
		rateBudget = cFuturesOrderbook500Rate
	}

	var data OrderbookData
	err = b.SendHTTPRequest(ctx, exchange.RestCoinMargined, cfuturesOrderbook+params.Encode(), rateBudget, &data)
	if err != nil {
		return nil, err
	}

	var resp OrderBook
	var price, quantity float64
	resp.Asks = make([]OrderbookItem, len(data.Asks))
	for x := range data.Asks {
		price, err = strconv.ParseFloat(data.Asks[x][0], 64)
		if err != nil {
			return nil, err
		}
		quantity, err = strconv.ParseFloat(data.Asks[x][1], 64)
		if err != nil {
			return nil, err
		}
		resp.Asks[x] = OrderbookItem{
			Price:    price,
			Quantity: quantity,
		}
	}

	resp.Bids = make([]OrderbookItem, len(data.Bids))
	for y := range data.Bids {
		price, err = strconv.ParseFloat(data.Bids[y][0], 64)
		if err != nil {
			return nil, err
		}
		quantity, err = strconv.ParseFloat(data.Bids[y][1], 64)
		if err != nil {
			return nil, err
		}
		resp.Bids[y] = OrderbookItem{
			Price:    price,
			Quantity: quantity,
		}
	}
	return &resp, nil
}

// GetFuturesPublicTrades gets recent public trades for CoinMarginedFutures,
func (b *Binance) GetFuturesPublicTrades(ctx context.Context, symbol currency.Pair, limit int64) ([]FuturesPublicTradesData, error) {
	var resp []FuturesPublicTradesData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp, b.SendHTTPRequest(ctx, exchange.RestCoinMargined, cfuturesRecentTrades+params.Encode(), cFuturesDefaultRate, &resp)
}

// GetFuturesHistoricalTrades gets historical public trades for CoinMarginedFutures,
func (b *Binance) GetFuturesHistoricalTrades(ctx context.Context, symbol currency.Pair, fromID string, limit int64) ([]UPublicTradesData, error) {
	var resp []UPublicTradesData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if fromID != "" {
		params.Set("fromID", fromID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, cfuturesHistoricalTrades, params, cFuturesHistoricalTradesRate, &resp)
}

// GetPastPublicTrades gets past public trades for CoinMarginedFutures,
func (b *Binance) GetPastPublicTrades(ctx context.Context, symbol currency.Pair, limit, fromID int64) ([]FuturesPublicTradesData, error) {
	var resp []FuturesPublicTradesData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if fromID != 0 {
		params.Set("fromID", strconv.FormatInt(fromID, 10))
	}
	return resp, b.SendHTTPRequest(ctx, exchange.RestCoinMargined, cfuturesRecentTrades+params.Encode(), cFuturesDefaultRate, &resp)
}

// GetFuturesAggregatedTradesList gets aggregated trades list for CoinMarginedFutures,
func (b *Binance) GetFuturesAggregatedTradesList(ctx context.Context, symbol currency.Pair, fromID, limit int64, startTime, endTime time.Time) ([]AggregatedTrade, error) {
	var resp []AggregatedTrade
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if fromID != 0 {
		params.Set("fromID", strconv.FormatInt(fromID, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return resp, b.SendHTTPRequest(ctx, exchange.RestCoinMargined, cfuturesCompressedTrades+params.Encode(), cFuturesHistoricalTradesRate, &resp)
}

// GetIndexAndMarkPrice gets index and mark prices  for CoinMarginedFutures,
func (b *Binance) GetIndexAndMarkPrice(ctx context.Context, symbol, pair string) ([]IndexMarkPrice, error) {
	var resp []IndexMarkPrice
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	return resp, b.SendHTTPRequest(ctx, exchange.RestCoinMargined, cfuturesMarkPrice+params.Encode(), cFuturesIndexMarkPriceRate, &resp)
}

// GetFundingRateInfo returns extra details about funding rates
func (b *Binance) GetFundingRateInfo(ctx context.Context) ([]FundingRateInfoResponse, error) {
	params := url.Values{}
	var resp []FundingRateInfoResponse
	return resp, b.SendHTTPRequest(ctx, exchange.RestCoinMargined, cfuturesFundingRateInfo+params.Encode(), uFuturesDefaultRate, &resp)
}

// GetFuturesKlineData gets futures kline data for CoinMarginedFutures,
func (b *Binance) GetFuturesKlineData(ctx context.Context, symbol currency.Pair, interval string, limit int64, startTime, endTime time.Time) ([]FuturesCandleStick, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !slices.Contains(validFuturesIntervals, interval) {
		return nil, errors.New("invalid interval parsed")
	}
	params.Set("interval", interval)
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return nil, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}

	var data [][10]interface{}
	rateBudget := getKlineRateBudget(limit)
	err := b.SendHTTPRequest(ctx, exchange.RestCoinMargined, cfuturesKlineData+params.Encode(), rateBudget, &data)
	if err != nil {
		return nil, err
	}

	resp := make([]FuturesCandleStick, len(data))
	for x := range data {
		var floatData float64
		var strData string
		var ok bool
		var tempData FuturesCandleStick
		floatData, ok = data[x][0].(float64)
		if !ok {
			return nil, errors.New("type assertion failed for open time")
		}
		tempData.OpenTime = time.Unix(int64(floatData), 0)
		strData, ok = data[x][1].(string)
		if !ok {
			return nil, errors.New("type assertion failed for open")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.Open = floatData
		strData, ok = data[x][2].(string)
		if !ok {
			return nil, errors.New("type assertion failed for high")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.High = floatData
		strData, ok = data[x][3].(string)
		if !ok {
			return nil, errors.New("type assertion failed for low")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.Low = floatData
		strData, ok = data[x][4].(string)
		if !ok {
			return nil, errors.New("type assertion failed for close")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.Close = floatData
		strData, ok = data[x][5].(string)
		if !ok {
			return nil, errors.New("type assertion failed for volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.Volume = floatData
		floatData, ok = data[x][6].(float64)
		if !ok {
			return nil, errors.New("type assertion failed for close time")
		}
		tempData.CloseTime = time.Unix(int64(floatData), 0)
		strData, ok = data[x][7].(string)
		if !ok {
			return nil, errors.New("type assertion failed for base asset volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.BaseAssetVolume = floatData
		floatData, ok = data[x][8].(float64)
		if !ok {
			return nil, errors.New("type assertion failed for taker buy volume")
		}
		tempData.TakerBuyVolume = floatData
		strData, ok = data[x][9].(string)
		if !ok {
			return nil, errors.New("type assertion failed for taker buy base asset volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.TakerBuyBaseAssetVolume = floatData
		resp[x] = tempData
	}
	return resp, nil
}

// GetContinuousKlineData gets continuous kline data
func (b *Binance) GetContinuousKlineData(ctx context.Context, pair, contractType, interval string, limit int64, startTime, endTime time.Time) ([]FuturesCandleStick, error) {
	params := url.Values{}
	params.Set("pair", pair)
	if !slices.Contains(validContractType, contractType) {
		return nil, errors.New("invalid contractType")
	}
	params.Set("contractType", contractType)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !slices.Contains(validFuturesIntervals, interval) {
		return nil, errors.New("invalid interval parsed")
	}
	params.Set("interval", interval)
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return nil, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}

	rateBudget := getKlineRateBudget(limit)
	var data [][10]interface{}
	err := b.SendHTTPRequest(ctx, exchange.RestCoinMargined, cfuturesContinuousKline+params.Encode(), rateBudget, &data)
	if err != nil {
		return nil, err
	}

	resp := make([]FuturesCandleStick, len(data))
	for x := range data {
		var floatData float64
		var strData string
		var ok bool
		var tempData FuturesCandleStick
		floatData, ok = data[x][0].(float64)
		if !ok {
			return nil, errors.New("type assertion failed for open time")
		}
		tempData.OpenTime = time.Unix(int64(floatData), 0)
		strData, ok = data[x][1].(string)
		if !ok {
			return nil, errors.New("type assertion failed for open")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.Open = floatData
		strData, ok = data[x][2].(string)
		if !ok {
			return nil, errors.New("type assertion failed for high")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.High = floatData
		strData, ok = data[x][3].(string)
		if !ok {
			return nil, errors.New("type assertion failed for low")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.Low = floatData
		strData, ok = data[x][4].(string)
		if !ok {
			return nil, errors.New("type assertion failed for close")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.Close = floatData
		strData, ok = data[x][5].(string)
		if !ok {
			return nil, errors.New("type assertion failed for volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.Volume = floatData
		floatData, ok = data[x][6].(float64)
		if !ok {
			return nil, errors.New("type assertion failed for close time")
		}
		tempData.CloseTime = time.Unix(int64(floatData), 0)
		strData, ok = data[x][7].(string)
		if !ok {
			return nil, errors.New("type assertion failed for base asset volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.BaseAssetVolume = floatData
		floatData, ok = data[x][8].(float64)
		if !ok {
			return nil, errors.New("type assertion failed for taker buy volume")
		}
		tempData.TakerBuyVolume = floatData
		strData, ok = data[x][9].(string)
		if !ok {
			return nil, errors.New("type assertion failed for taker buy base asset volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.TakerBuyBaseAssetVolume = floatData
		resp[x] = tempData
	}
	return resp, nil
}

// GetIndexPriceKlines gets continuous kline data
func (b *Binance) GetIndexPriceKlines(ctx context.Context, pair, interval string, limit int64, startTime, endTime time.Time) ([]FuturesCandleStick, error) {
	params := url.Values{}
	params.Set("pair", pair)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !slices.Contains(validFuturesIntervals, interval) {
		return nil, errors.New("invalid interval parsed")
	}
	params.Set("interval", interval)
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return nil, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}

	rateBudget := getKlineRateBudget(limit)
	var data [][10]interface{}
	err := b.SendHTTPRequest(ctx, exchange.RestCoinMargined, cfuturesIndexKline+params.Encode(), rateBudget, &data)
	if err != nil {
		return nil, err
	}

	resp := make([]FuturesCandleStick, len(data))
	for x := range data {
		var floatData float64
		var strData string
		var ok bool
		var tempData FuturesCandleStick
		floatData, ok = data[x][0].(float64)
		if !ok {
			return nil, errors.New("type assertion failed for open time")
		}
		tempData.OpenTime = time.Unix(int64(floatData), 0)
		strData, ok = data[x][1].(string)
		if !ok {
			return nil, errors.New("type assertion failed for open")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.Open = floatData
		strData, ok = data[x][2].(string)
		if !ok {
			return nil, errors.New("type assertion failed for high")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.High = floatData
		strData, ok = data[x][3].(string)
		if !ok {
			return nil, errors.New("type assertion failed for low")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.Low = floatData
		strData, ok = data[x][4].(string)
		if !ok {
			return nil, errors.New("type assertion failed for close")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.Close = floatData
		strData, ok = data[x][5].(string)
		if !ok {
			return nil, errors.New("type assertion failed for volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.Volume = floatData
		floatData, ok = data[x][6].(float64)
		if !ok {
			return nil, errors.New("type assertion failed for close time")
		}
		tempData.CloseTime = time.Unix(int64(floatData), 0)
		strData, ok = data[x][7].(string)
		if !ok {
			return nil, errors.New("type assertion failed for base asset volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.BaseAssetVolume = floatData
		floatData, ok = data[x][8].(float64)
		if !ok {
			return nil, errors.New("type assertion failed for taker buy volume")
		}
		tempData.TakerBuyVolume = floatData
		strData, ok = data[x][9].(string)
		if !ok {
			return nil, errors.New("type assertion failed for taker buy base asset volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.TakerBuyBaseAssetVolume = floatData
		resp[x] = tempData
	}
	return resp, nil
}

// GetMarkPriceKline gets mark price kline data
func (b *Binance) GetMarkPriceKline(ctx context.Context, symbol currency.Pair, interval string, limit int64, startTime, endTime time.Time) ([]FuturesCandleStick, error) {
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !slices.Contains(validFuturesIntervals, interval) {
		return nil, errors.New("invalid interval parsed")
	}
	params.Set("interval", interval)
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return nil, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}

	var data [][10]interface{}
	rateBudget := getKlineRateBudget(limit)
	err = b.SendHTTPRequest(ctx, exchange.RestCoinMargined, cfuturesMarkPriceKline+params.Encode(), rateBudget, &data)
	if err != nil {
		return nil, err
	}

	resp := make([]FuturesCandleStick, len(data))
	for x := range data {
		var floatData float64
		var strData string
		var ok bool
		var tempData FuturesCandleStick
		floatData, ok = data[x][0].(float64)
		if !ok {
			return nil, errors.New("type assertion failed for open time")
		}
		tempData.OpenTime = time.Unix(int64(floatData), 0)
		strData, ok = data[x][1].(string)
		if !ok {
			return nil, errors.New("type assertion failed for open")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.Open = floatData
		strData, ok = data[x][2].(string)
		if !ok {
			return nil, errors.New("type assertion failed for high")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.High = floatData
		strData, ok = data[x][3].(string)
		if !ok {
			return nil, errors.New("type assertion failed for low")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.Low = floatData
		strData, ok = data[x][4].(string)
		if !ok {
			return nil, errors.New("type assertion failed for close")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.Close = floatData
		strData, ok = data[x][5].(string)
		if !ok {
			return nil, errors.New("type assertion failed for volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.Volume = floatData
		floatData, ok = data[x][6].(float64)
		if !ok {
			return nil, errors.New("type assertion failed for close time")
		}
		tempData.CloseTime = time.Unix(int64(floatData), 0)
		strData, ok = data[x][7].(string)
		if !ok {
			return nil, errors.New("type assertion failed for base asset volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.BaseAssetVolume = floatData
		floatData, ok = data[x][8].(float64)
		if !ok {
			return nil, errors.New("type assertion failed for taker buy volume")
		}
		tempData.TakerBuyVolume = floatData
		strData, ok = data[x][9].(string)
		if !ok {
			return nil, errors.New("type assertion failed for taker buy base asset volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return nil, err
		}
		tempData.TakerBuyBaseAssetVolume = floatData
		resp[x] = tempData
	}
	return resp, nil
}

func getKlineRateBudget(limit int64) request.EndpointLimit {
	rateBudget := cFuturesDefaultRate
	switch {
	case limit > 0 && limit < 100:
		rateBudget = cFuturesDefaultRate
	case limit >= 100 && limit < 500:
		rateBudget = cFuturesKline500Rate
	case limit >= 500 && limit < 1000:
		rateBudget = cFuturesKline1000Rate
	case limit >= 1000:
		rateBudget = cFuturesKlineMaxRate
	}
	return rateBudget
}

// GetFuturesSwapTickerChangeStats gets 24hr ticker change stats for CoinMarginedFutures,
func (b *Binance) GetFuturesSwapTickerChangeStats(ctx context.Context, symbol currency.Pair, pair string) ([]PriceChangeStats, error) {
	var resp []PriceChangeStats
	params := url.Values{}
	rateLimit := cFuturesTickerPriceHistoryRate
	if !symbol.IsEmpty() {
		rateLimit = cFuturesDefaultRate
		symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	return resp, b.SendHTTPRequest(ctx, exchange.RestCoinMargined, cfuturesTickerPriceStats+params.Encode(), rateLimit, &resp)
}

// FuturesGetFundingHistory gets funding history for CoinMarginedFutures,
func (b *Binance) FuturesGetFundingHistory(ctx context.Context, symbol currency.Pair, limit int64, startTime, endTime time.Time) ([]FundingRateHistory, error) {
	var resp []FundingRateHistory
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return resp, b.SendHTTPRequest(ctx, exchange.RestCoinMargined, cfuturesFundingRateHistory+params.Encode(), cFuturesDefaultRate, &resp)
}

// GetFuturesSymbolPriceTicker gets price ticker for symbol
func (b *Binance) GetFuturesSymbolPriceTicker(ctx context.Context, symbol currency.Pair, pair string) ([]SymbolPriceTicker, error) {
	var resp []SymbolPriceTicker
	params := url.Values{}
	rateLimit := cFuturesOrderbookTickerAllRate
	if !symbol.IsEmpty() {
		rateLimit = cFuturesDefaultRate
		symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	return resp, b.SendHTTPRequest(ctx, exchange.RestCoinMargined, cfuturesSymbolPriceTicker+params.Encode(), rateLimit, &resp)
}

// GetFuturesOrderbookTicker gets orderbook ticker for symbol
func (b *Binance) GetFuturesOrderbookTicker(ctx context.Context, symbol currency.Pair, pair string) ([]SymbolOrderBookTicker, error) {
	var resp []SymbolOrderBookTicker
	params := url.Values{}
	rateLimit := cFuturesOrderbookTickerAllRate
	if !symbol.IsEmpty() {
		rateLimit = cFuturesDefaultRate
		symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	return resp, b.SendHTTPRequest(ctx, exchange.RestCoinMargined, cfuturesSymbolOrderbook+params.Encode(), rateLimit, &resp)
}

// OpenInterest gets open interest data for a symbol
func (b *Binance) OpenInterest(ctx context.Context, symbol currency.Pair) (OpenInterestData, error) {
	var resp OpenInterestData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	return resp, b.SendHTTPRequest(ctx, exchange.RestCoinMargined, cfuturesOpenInterest+params.Encode(), cFuturesDefaultRate, &resp)
}

// GetOpenInterestStats gets open interest stats for a symbol
func (b *Binance) GetOpenInterestStats(ctx context.Context, pair, contractType, period string, limit int64, startTime, endTime time.Time) ([]OpenInterestStats, error) {
	var resp []OpenInterestStats
	params := url.Values{}
	if pair != "" {
		params.Set("pair", pair)
	}
	if !slices.Contains(validContractType, contractType) {
		return resp, errors.New("invalid contractType")
	}
	params.Set("contractType", contractType)
	if !slices.Contains(validFuturesIntervals, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return resp, b.SendHTTPRequest(ctx, exchange.RestCoinMargined, cfuturesOpenInterestStats+params.Encode(), cFuturesDefaultRate, &resp)
}

// GetTraderFuturesAccountRatio gets a traders futures account long/short ratio
func (b *Binance) GetTraderFuturesAccountRatio(ctx context.Context, pair, period string, limit int64, startTime, endTime time.Time) ([]TopTraderAccountRatio, error) {
	var resp []TopTraderAccountRatio
	params := url.Values{}
	params.Set("pair", pair)
	if !slices.Contains(validFuturesIntervals, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return resp, b.SendHTTPRequest(ctx, exchange.RestCoinMargined, cfuturesTopAccountsRatio+params.Encode(), cFuturesDefaultRate, &resp)
}

// GetTraderFuturesPositionsRatio gets a traders futures positions' long/short ratio
func (b *Binance) GetTraderFuturesPositionsRatio(ctx context.Context, pair, period string, limit int64, startTime, endTime time.Time) ([]TopTraderPositionRatio, error) {
	var resp []TopTraderPositionRatio
	params := url.Values{}
	params.Set("pair", pair)
	if !slices.Contains(validFuturesIntervals, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return resp, b.SendHTTPRequest(ctx, exchange.RestCoinMargined, cfuturesTopPositionsRatio+params.Encode(), cFuturesDefaultRate, &resp)
}

// GetMarketRatio gets global long/short ratio
func (b *Binance) GetMarketRatio(ctx context.Context, pair, period string, limit int64, startTime, endTime time.Time) ([]TopTraderPositionRatio, error) {
	var resp []TopTraderPositionRatio
	params := url.Values{}
	params.Set("pair", pair)
	if !slices.Contains(validFuturesIntervals, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return resp, b.SendHTTPRequest(ctx, exchange.RestCoinMargined, cfuturesLongShortRatio+params.Encode(), cFuturesDefaultRate, &resp)
}

// GetFuturesTakerVolume gets futures taker buy/sell volumes
func (b *Binance) GetFuturesTakerVolume(ctx context.Context, pair, contractType, period string, limit int64, startTime, endTime time.Time) ([]TakerBuySellVolume, error) {
	var resp []TakerBuySellVolume
	params := url.Values{}
	params.Set("pair", pair)
	if !slices.Contains(validContractType, contractType) {
		return resp, errors.New("invalid contractType")
	}
	params.Set("contractType", contractType)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !slices.Contains(validFuturesIntervals, period) {
		return resp, errors.New("invalid period parsed")
	}
	params.Set("period", period)
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return resp, b.SendHTTPRequest(ctx, exchange.RestCoinMargined, cfuturesBuySellVolume+params.Encode(), cFuturesDefaultRate, &resp)
}

// GetFuturesBasisData gets futures basis data
func (b *Binance) GetFuturesBasisData(ctx context.Context, pair, contractType, period string, limit int64, startTime, endTime time.Time) ([]FuturesBasisData, error) {
	var resp []FuturesBasisData
	params := url.Values{}
	params.Set("pair", pair)
	if !slices.Contains(validContractType, contractType) {
		return resp, errors.New("invalid contractType")
	}
	params.Set("contractType", contractType)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !slices.Contains(validFuturesIntervals, period) {
		return resp, errors.New("invalid period parsed")
	}
	params.Set("period", period)
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return resp, b.SendHTTPRequest(ctx, exchange.RestCoinMargined, cfuturesBasis+params.Encode(), cFuturesDefaultRate, &resp)
}

// FuturesNewOrder sends a new futures order to the exchange
func (b *Binance) FuturesNewOrder(ctx context.Context, x *FuturesNewOrderRequest) (
	FuturesOrderPlaceData,
	error,
) {
	var resp FuturesOrderPlaceData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(x.Symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	params.Set("side", x.Side)
	if x.PositionSide != "" {
		if !slices.Contains(validPositionSide, x.PositionSide) {
			return resp, errors.New("invalid positionSide")
		}
		params.Set("positionSide", x.PositionSide)
	}
	params.Set("type", x.OrderType)
	if string(x.TimeInForce) != "" {
		params.Set("timeInForce", string(x.TimeInForce))
	}
	if x.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	if x.NewClientOrderID != "" {
		params.Set("newClientOrderID", x.NewClientOrderID)
	}
	if x.ClosePosition != "" {
		params.Set("closePosition", x.ClosePosition)
	}
	if x.WorkingType != "" {
		if !slices.Contains(validWorkingType, x.WorkingType) {
			return resp, errors.New("invalid workingType")
		}
		params.Set("workingType", x.WorkingType)
	}
	if x.NewOrderRespType != "" {
		if !slices.Contains(validNewOrderRespType, x.NewOrderRespType) {
			return resp, errors.New("invalid newOrderRespType")
		}
		params.Set("newOrderRespType", x.NewOrderRespType)
	}
	if x.Quantity != 0 {
		params.Set("quantity", strconv.FormatFloat(x.Quantity, 'f', -1, 64))
	}
	if x.Price != 0 {
		params.Set("price", strconv.FormatFloat(x.Price, 'f', -1, 64))
	}
	if x.StopPrice != 0 {
		params.Set("stopPrice", strconv.FormatFloat(x.StopPrice, 'f', -1, 64))
	}
	if x.ActivationPrice != 0 {
		params.Set("activationPrice", strconv.FormatFloat(x.ActivationPrice, 'f', -1, 64))
	}
	if x.CallbackRate != 0 {
		params.Set("callbackRate", strconv.FormatFloat(x.CallbackRate, 'f', -1, 64))
	}
	if x.PriceProtect {
		params.Set("priceProtect", "TRUE")
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, cfuturesOrder, params, cFuturesOrdersDefaultRate, &resp)
}

// FuturesBatchOrder sends a batch order request
func (b *Binance) FuturesBatchOrder(ctx context.Context, data []PlaceBatchOrderData) ([]FuturesOrderPlaceData, error) {
	var resp []FuturesOrderPlaceData
	params := url.Values{}
	for x := range data {
		unformattedPair, err := currency.NewPairFromString(data[x].Symbol)
		if err != nil {
			return resp, err
		}
		formattedPair, err := b.FormatExchangeCurrency(unformattedPair, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		data[x].Symbol = formattedPair.String()
		if data[x].PositionSide != "" {
			if !slices.Contains(validPositionSide, data[x].PositionSide) {
				return resp, errors.New("invalid positionSide")
			}
		}
		if data[x].WorkingType != "" {
			if !slices.Contains(validWorkingType, data[x].WorkingType) {
				return resp, errors.New("invalid workingType")
			}
		}
		if data[x].NewOrderRespType != "" {
			if !slices.Contains(validNewOrderRespType, data[x].NewOrderRespType) {
				return resp, errors.New("invalid newOrderRespType")
			}
		}
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return resp, err
	}
	params.Set("batchOrders", string(jsonData))
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, cfuturesBatchOrder, params, cFuturesBatchOrdersRate, &resp)
}

// FuturesBatchCancelOrders sends a batch request to cancel orders
func (b *Binance) FuturesBatchCancelOrders(ctx context.Context, symbol currency.Pair, orderList, origClientOrderIDList []string) ([]BatchCancelOrderData, error) {
	var resp []BatchCancelOrderData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if len(orderList) != 0 {
		jsonOrderList, err := json.Marshal(orderList)
		if err != nil {
			return resp, err
		}
		params.Set("orderIdList", string(jsonOrderList))
	}
	if len(origClientOrderIDList) != 0 {
		jsonCliOrdIDList, err := json.Marshal(origClientOrderIDList)
		if err != nil {
			return resp, err
		}
		params.Set("origClientOrderIdList", string(jsonCliOrdIDList))
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodDelete, cfuturesBatchOrder, params, cFuturesOrdersDefaultRate, &resp)
}

// FuturesGetOrderData gets futures order data
func (b *Binance) FuturesGetOrderData(ctx context.Context, symbol currency.Pair, orderID, origClientOrderID string) (FuturesOrderGetData, error) {
	var resp FuturesOrderGetData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, cfuturesOrder, params, cFuturesOrdersDefaultRate, &resp)
}

// FuturesCancelOrder cancels a futures order
func (b *Binance) FuturesCancelOrder(ctx context.Context, symbol currency.Pair, orderID, origClientOrderID string) (FuturesOrderGetData, error) {
	var resp FuturesOrderGetData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodDelete, cfuturesOrder, params, cFuturesOrdersDefaultRate, &resp)
}

// FuturesCancelAllOpenOrders cancels a futures order
func (b *Binance) FuturesCancelAllOpenOrders(ctx context.Context, symbol currency.Pair) (GenericAuthResponse, error) {
	var resp GenericAuthResponse
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodDelete, cfuturesCancelAllOrders, params, cFuturesOrdersDefaultRate, &resp)
}

// AutoCancelAllOpenOrders cancels all open futures orders
// countdownTime 1000 = 1s, example - to cancel all orders after 30s (countdownTime: 30000)
func (b *Binance) AutoCancelAllOpenOrders(ctx context.Context, symbol currency.Pair, countdownTime int64) (AutoCancelAllOrdersData, error) {
	var resp AutoCancelAllOrdersData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	params.Set("countdownTime", strconv.FormatInt(countdownTime, 10))
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, cfuturesCountdownCancel, params, cFuturesCancelAllOrdersRate, &resp)
}

// FuturesOpenOrderData gets open order data for CoinMarginedFutures,
func (b *Binance) FuturesOpenOrderData(ctx context.Context, symbol currency.Pair, orderID, origClientOrderID string) (FuturesOrderGetData, error) {
	var resp FuturesOrderGetData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, cfuturesOpenOrder, params, cFuturesOrdersDefaultRate, &resp)
}

// GetFuturesAllOpenOrders gets all open orders data for CoinMarginedFutures,
func (b *Binance) GetFuturesAllOpenOrders(ctx context.Context, symbol currency.Pair, pair string) ([]FuturesOrderData, error) {
	var resp []FuturesOrderData
	params := url.Values{}
	var p string
	var err error
	rateLimit := cFuturesGetAllOpenOrdersRate
	if !symbol.IsEmpty() {
		rateLimit = cFuturesOrdersDefaultRate
		p, err = b.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", p)
	} else {
		// extend the receive window when all currencies to prevent "recvwindow" error
		params.Set("recvWindow", "10000")
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, cfuturesAllOpenOrders, params, rateLimit, &resp)
}

// GetAllFuturesOrders gets all orders active cancelled or filled
func (b *Binance) GetAllFuturesOrders(ctx context.Context, symbol, pair currency.Pair, startTime, endTime time.Time, orderID, limit int64) ([]FuturesOrderData, error) {
	var resp []FuturesOrderData
	params := url.Values{}
	rateLimit := cFuturesPairOrdersRate
	if !symbol.IsEmpty() {
		rateLimit = cFuturesSymbolOrdersRate
		symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	if !pair.IsEmpty() {
		params.Set("pair", pair.String())
	}
	if orderID != 0 {
		params.Set("orderID", strconv.FormatInt(orderID, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, cfuturesAllOrders, params, rateLimit, &resp)
}

// GetFuturesAccountBalance gets account balance data for CoinMarginedFutures, account
func (b *Binance) GetFuturesAccountBalance(ctx context.Context) ([]FuturesAccountBalanceData, error) {
	var resp []FuturesAccountBalanceData
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, cfuturesAccountBalance, nil, cFuturesDefaultRate, &resp)
}

// GetFuturesAccountInfo gets account info data for CoinMarginedFutures, account
func (b *Binance) GetFuturesAccountInfo(ctx context.Context) (FuturesAccountInformation, error) {
	var resp FuturesAccountInformation
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, cfuturesAccountInfo, nil, cFuturesAccountInformationRate, &resp)
}

// FuturesChangeInitialLeverage changes initial leverage for the account
func (b *Binance) FuturesChangeInitialLeverage(ctx context.Context, symbol currency.Pair, leverage float64) (FuturesLeverageData, error) {
	var resp FuturesLeverageData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if leverage < 1 || leverage > 125 {
		return resp, errors.New("invalid leverage")
	}
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, cfuturesChangeInitialLeverage, params, cFuturesDefaultRate, &resp)
}

// FuturesChangeMarginType changes margin type
func (b *Binance) FuturesChangeMarginType(ctx context.Context, symbol currency.Pair, marginType string) (GenericAuthResponse, error) {
	var resp GenericAuthResponse
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if !slices.Contains(validMarginType, marginType) {
		return resp, errors.New("invalid marginType")
	}
	params.Set("marginType", marginType)
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, cfuturesChangeMarginType, params, cFuturesDefaultRate, &resp)
}

// ModifyIsolatedPositionMargin changes margin for an isolated position
func (b *Binance) ModifyIsolatedPositionMargin(ctx context.Context, symbol currency.Pair, positionSide, changeType string, amount float64) (FuturesMarginUpdatedResponse, error) {
	var resp FuturesMarginUpdatedResponse
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if positionSide != "" {
		if !slices.Contains(validPositionSide, positionSide) {
			return resp, errors.New("invalid positionSide")
		}
		params.Set("positionSide", positionSide)
	}
	cType, ok := validMarginChange[changeType]
	if !ok {
		return resp, errors.New("invalid changeType")
	}
	params.Set("type", strconv.FormatInt(cType, 10))
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, cfuturesModifyMargin, params, cFuturesDefaultRate, &resp)
}

// FuturesMarginChangeHistory gets past margin changes for positions
func (b *Binance) FuturesMarginChangeHistory(ctx context.Context, symbol currency.Pair, changeType string, startTime, endTime time.Time, limit int64) ([]GetPositionMarginChangeHistoryData, error) {
	var resp []GetPositionMarginChangeHistoryData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	cType, ok := validMarginChange[changeType]
	if !ok {
		return resp, errors.New("invalid changeType")
	}
	params.Set("type", strconv.FormatInt(cType, 10))
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, cfuturesMarginChangeHistory, params, cFuturesDefaultRate, &resp)
}

// FuturesPositionsInfo gets futures positions info
// "pair" for coinmarginedfutures in GCT terms is the pair base
// eg ADAUSD_PERP the "pair" parameter is ADAUSD
func (b *Binance) FuturesPositionsInfo(ctx context.Context, marginAsset, pair string) ([]FuturesPositionInformation, error) {
	var resp []FuturesPositionInformation
	params := url.Values{}
	if marginAsset != "" {
		params.Set("marginAsset", marginAsset)
	}

	if pair != "" {
		params.Set("pair", pair)
	}

	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, cfuturesPositionInfo, params, cFuturesDefaultRate, &resp)
}

// FuturesTradeHistory gets trade history for CoinMarginedFutures, account
func (b *Binance) FuturesTradeHistory(ctx context.Context, symbol currency.Pair, pair string, startTime, endTime time.Time, limit, fromID int64) ([]FuturesAccountTradeList, error) {
	var resp []FuturesAccountTradeList
	params := url.Values{}
	rateLimit := cFuturesPairOrdersRate
	if !symbol.IsEmpty() {
		rateLimit = cFuturesSymbolOrdersRate
		symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if fromID != 0 {
		params.Set("fromId", strconv.FormatInt(fromID, 10))
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, cfuturesAccountTradeList, params, rateLimit, &resp)
}

// FuturesIncomeHistory gets income history for CoinMarginedFutures,
func (b *Binance) FuturesIncomeHistory(ctx context.Context, symbol currency.Pair, incomeType string, startTime, endTime time.Time, limit int64) ([]FuturesIncomeHistoryData, error) {
	var resp []FuturesIncomeHistoryData
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	if incomeType != "" {
		if !slices.Contains(validIncomeType, incomeType) {
			return resp, fmt.Errorf("invalid incomeType: %v", incomeType)
		}
		params.Set("incomeType", incomeType)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, cfuturesIncomeHistory, params, cFuturesIncomeHistoryRate, &resp)
}

// FuturesNotionalBracket gets futures notional bracket
func (b *Binance) FuturesNotionalBracket(ctx context.Context, pair string) ([]NotionalBracketData, error) {
	var resp []NotionalBracketData
	params := url.Values{}
	if pair != "" {
		params.Set("pair", pair)
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, cfuturesNotionalBracket, params, cFuturesDefaultRate, &resp)
}

// FuturesForceOrders gets futures forced orders
func (b *Binance) FuturesForceOrders(ctx context.Context, symbol currency.Pair, autoCloseType string, startTime, endTime time.Time) ([]ForcedOrdersData, error) {
	var resp []ForcedOrdersData
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	if autoCloseType != "" {
		if !slices.Contains(validAutoCloseTypes, autoCloseType) {
			return resp, errors.New("invalid autoCloseType")
		}
		params.Set("autoCloseType", autoCloseType)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, cfuturesUsersForceOrders, params, cFuturesDefaultRate, &resp)
}

// FuturesPositionsADLEstimate estimates ADL on positions
func (b *Binance) FuturesPositionsADLEstimate(ctx context.Context, symbol currency.Pair) ([]ADLEstimateData, error) {
	var resp []ADLEstimateData
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, cfuturesADLQuantile, params, cFuturesAccountInformationRate, &resp)
}

// FetchCoinMarginExchangeLimits fetches coin margined order execution limits
func (b *Binance) FetchCoinMarginExchangeLimits(ctx context.Context) ([]order.MinMaxLevel, error) {
	coinFutures, err := b.FuturesExchangeInfo(ctx)
	if err != nil {
		return nil, err
	}

	limits := make([]order.MinMaxLevel, 0, len(coinFutures.Symbols))
	for x := range coinFutures.Symbols {
		symbol := strings.Split(coinFutures.Symbols[x].Symbol, currency.UnderscoreDelimiter)
		var cp currency.Pair
		cp, err = currency.NewPairFromStrings(symbol[0], symbol[1])
		if err != nil {
			return nil, err
		}

		if len(coinFutures.Symbols[x].Filters) < 6 {
			continue
		}

		limits = append(limits, order.MinMaxLevel{
			Pair:                    cp,
			Asset:                   asset.CoinMarginedFutures,
			MinPrice:                coinFutures.Symbols[x].Filters[0].MinPrice,
			MaxPrice:                coinFutures.Symbols[x].Filters[0].MaxPrice,
			PriceStepIncrementSize:  coinFutures.Symbols[x].Filters[0].TickSize,
			MaximumBaseAmount:       coinFutures.Symbols[x].Filters[1].MaxQty,
			MinimumBaseAmount:       coinFutures.Symbols[x].Filters[1].MinQty,
			AmountStepIncrementSize: coinFutures.Symbols[x].Filters[1].StepSize,
			MarketMinQty:            coinFutures.Symbols[x].Filters[2].MinQty,
			MarketMaxQty:            coinFutures.Symbols[x].Filters[2].MaxQty,
			MarketStepIncrementSize: coinFutures.Symbols[x].Filters[2].StepSize,
			MaxTotalOrders:          coinFutures.Symbols[x].Filters[3].Limit,
			MaxAlgoOrders:           coinFutures.Symbols[x].Filters[4].Limit,
			MultiplierUp:            coinFutures.Symbols[x].Filters[5].MultiplierUp,
			MultiplierDown:          coinFutures.Symbols[x].Filters[5].MultiplierDown,
			MultiplierDecimal:       coinFutures.Symbols[x].Filters[5].MultiplierDecimal,
		})
	}
	return limits, nil
}
