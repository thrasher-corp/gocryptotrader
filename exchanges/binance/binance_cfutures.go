package binance

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
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
	cfuturesMarkPrice          = "/dapi/v1/premiumIndex?"
	cfuturesFundingRateHistory = "/dapi/v1/fundingRate?"
	cfuturesTickerPriceStats   = "/dapi/v1/ticker/24hr?"
	cfuturesSymbolPriceTicker  = "/dapi/v1/ticker/price?"
	cfuturesSymbolOrderbook    = "/dapi/v1/ticker/bookTicker?"
	cfuturesLiquidationOrders  = "/dapi/v1/allForceOrders?"
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
)

// FuturesExchangeInfo stores CoinMarginedFutures, data
func (b *Binance) FuturesExchangeInfo() (CExchangeInfo, error) {
	var resp CExchangeInfo
	return resp, b.SendHTTPRequest(exchange.RestCoinMargined, cfuturesExchangeInfo, cFuturesDefaultRate, &resp)
}

// GetFuturesOrderbook gets orderbook data for CoinMarginedFutures,
func (b *Binance) GetFuturesOrderbook(symbol currency.Pair, limit int64) (OrderBook, error) {
	var resp OrderBook
	var data OrderbookData
	params := url.Values{}
	rateBudget := cFuturesDefaultRate
	switch {
	case limit == 5, limit == 10, limit == 20, limit == 50:
		rateBudget = cFuturesOrderbook50Rate
	case limit >= 100 && limit < 500:
		rateBudget = cFuturesOrderbook100Rate
	case limit >= 500 && limit < 1000:
		rateBudget = cFuturesOrderbook500Rate
	case limit == 1000:
		rateBudget = cFuturesOrderbook1000Rate
	}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	err = b.SendHTTPRequest(exchange.RestCoinMargined, cfuturesOrderbook+params.Encode(), rateBudget, &data)
	if err != nil {
		return resp, err
	}
	var price, quantity float64
	for x := range data.Asks {
		price, err = strconv.ParseFloat(data.Asks[x][0], 64)
		if err != nil {
			return resp, err
		}
		quantity, err = strconv.ParseFloat(data.Asks[x][1], 64)
		if err != nil {
			return resp, err
		}
		resp.Asks = append(resp.Asks, OrderbookItem{
			Price:    price,
			Quantity: quantity,
		})
	}
	for y := range data.Bids {
		price, err = strconv.ParseFloat(data.Bids[y][0], 64)
		if err != nil {
			return resp, err
		}
		quantity, err = strconv.ParseFloat(data.Bids[y][1], 64)
		if err != nil {
			return resp, err
		}
		resp.Bids = append(resp.Bids, OrderbookItem{
			Price:    price,
			Quantity: quantity,
		})
	}
	return resp, nil
}

// GetFuturesPublicTrades gets recent public trades for CoinMarginedFutures,
func (b *Binance) GetFuturesPublicTrades(symbol currency.Pair, limit int64) ([]FuturesPublicTradesData, error) {
	var resp []FuturesPublicTradesData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp, b.SendHTTPRequest(exchange.RestCoinMargined, cfuturesRecentTrades+params.Encode(), cFuturesDefaultRate, &resp)
}

// GetFuturesHistoricalTrades gets historical public trades for CoinMarginedFutures,
func (b *Binance) GetFuturesHistoricalTrades(symbol currency.Pair, fromID string, limit int64) ([]UPublicTradesData, error) {
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
	if limit > 0 && limit < 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp, b.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, cfuturesHistoricalTrades, params, cFuturesHistoricalTradesRate, &resp)
}

// GetPastPublicTrades gets past public trades for CoinMarginedFutures,
func (b *Binance) GetPastPublicTrades(symbol currency.Pair, limit, fromID int64) ([]FuturesPublicTradesData, error) {
	var resp []FuturesPublicTradesData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if fromID != 0 {
		params.Set("fromID", strconv.FormatInt(fromID, 10))
	}
	return resp, b.SendHTTPRequest(exchange.RestCoinMargined, cfuturesRecentTrades+params.Encode(), cFuturesDefaultRate, &resp)
}

// GetFuturesAggregatedTradesList gets aggregated trades list for CoinMarginedFutures,
func (b *Binance) GetFuturesAggregatedTradesList(symbol currency.Pair, fromID, limit int64, startTime, endTime time.Time) ([]AggregatedTrade, error) {
	var resp []AggregatedTrade
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if fromID != 0 {
		params.Set("fromID", strconv.FormatInt(fromID, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.RestCoinMargined, cfuturesCompressedTrades+params.Encode(), cFuturesHistoricalTradesRate, &resp)
}

// GetIndexAndMarkPrice gets index and mark prices  for CoinMarginedFutures,
func (b *Binance) GetIndexAndMarkPrice(symbol, pair string) ([]IndexMarkPrice, error) {
	var resp []IndexMarkPrice
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	return resp, b.SendHTTPRequest(exchange.RestCoinMargined, cfuturesMarkPrice+params.Encode(), cFuturesIndexMarkPriceRate, &resp)
}

// GetFuturesKlineData gets futures kline data for CoinMarginedFutures,
func (b *Binance) GetFuturesKlineData(symbol currency.Pair, interval string, limit int64, startTime, endTime time.Time) ([]FuturesCandleStick, error) {
	var data [][10]interface{}
	var resp []FuturesCandleStick
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	if limit > 0 && limit <= 1500 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesIntervals, interval) {
		return resp, errors.New("invalid interval parsed")
	}
	params.Set("interval", interval)
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	rateBudget := getKlineRateBudget(limit)
	err := b.SendHTTPRequest(exchange.RestCoinMargined, cfuturesKlineData+params.Encode(), rateBudget, &data)
	if err != nil {
		return resp, err
	}
	var floatData float64
	var strData string
	var ok bool
	var tempData FuturesCandleStick
	for x := range data {
		floatData, ok = data[x][0].(float64)
		if !ok {
			return resp, errors.New("type assertion failed for open time")
		}
		tempData.OpenTime = time.Unix(int64(floatData), 0)
		strData, ok = data[x][1].(string)
		if !ok {
			return resp, errors.New("type assertion failed for open")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Open = floatData
		strData, ok = data[x][2].(string)
		if !ok {
			return resp, errors.New("type assertion failed for high")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.High = floatData
		strData, ok = data[x][3].(string)
		if !ok {
			return resp, errors.New("type assertion failed for low")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Low = floatData
		strData, ok = data[x][4].(string)
		if !ok {
			return resp, errors.New("type assertion failed for close")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Close = floatData
		strData, ok = data[x][5].(string)
		if !ok {
			return resp, errors.New("type assertion failed for volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Volume = floatData
		floatData, ok = data[x][6].(float64)
		if !ok {
			return resp, errors.New("type assertion failed for close time")
		}
		tempData.CloseTime = time.Unix(int64(floatData), 0)
		strData, ok = data[x][7].(string)
		if !ok {
			return resp, errors.New("type assertion failed for base asset volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.BaseAssetVolume = floatData
		floatData, ok = data[x][8].(float64)
		if !ok {
			return resp, errors.New("type assertion failed for taker buy volume")
		}
		tempData.TakerBuyVolume = floatData
		strData, ok = data[x][9].(string)
		if !ok {
			return resp, errors.New("type assertion failed for taker buy base asset volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.TakerBuyBaseAssetVolume = floatData
		resp = append(resp, tempData)
	}
	return resp, nil
}

// GetContinuousKlineData gets continuous kline data
func (b *Binance) GetContinuousKlineData(pair, contractType, interval string, limit int64, startTime, endTime time.Time) ([]FuturesCandleStick, error) {
	var data [][10]interface{}
	var resp []FuturesCandleStick
	params := url.Values{}
	params.Set("pair", pair)
	if !common.StringDataCompare(validContractType, contractType) {
		return resp, errors.New("invalid contractType")
	}
	params.Set("contractType", contractType)
	if limit > 0 && limit <= 1500 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesIntervals, interval) {
		return resp, errors.New("invalid interval parsed")
	}
	params.Set("interval", interval)
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}

	rateBudget := getKlineRateBudget(limit)
	err := b.SendHTTPRequest(exchange.RestCoinMargined, cfuturesContinuousKline+params.Encode(), rateBudget, &data)
	if err != nil {
		return resp, err
	}
	var floatData float64
	var strData string
	var ok bool
	var tempData FuturesCandleStick
	for x := range data {
		floatData, ok = data[x][0].(float64)
		if !ok {
			return resp, errors.New("type assertion failed for open time")
		}
		tempData.OpenTime = time.Unix(int64(floatData), 0)
		strData, ok = data[x][1].(string)
		if !ok {
			return resp, errors.New("type assertion failed for open")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Open = floatData
		strData, ok = data[x][2].(string)
		if !ok {
			return resp, errors.New("type assertion failed for high")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.High = floatData
		strData, ok = data[x][3].(string)
		if !ok {
			return resp, errors.New("type assertion failed for low")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Low = floatData
		strData, ok = data[x][4].(string)
		if !ok {
			return resp, errors.New("type assertion failed for close")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Close = floatData
		strData, ok = data[x][5].(string)
		if !ok {
			return resp, errors.New("type assertion failed for volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Volume = floatData
		floatData, ok = data[x][6].(float64)
		if !ok {
			return resp, errors.New("type assertion failed for close time")
		}
		tempData.CloseTime = time.Unix(int64(floatData), 0)
		strData, ok = data[x][7].(string)
		if !ok {
			return resp, errors.New("type assertion failed for base asset volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.BaseAssetVolume = floatData
		floatData, ok = data[x][8].(float64)
		if !ok {
			return resp, errors.New("type assertion failed for taker buy volume")
		}
		tempData.TakerBuyVolume = floatData
		strData, ok = data[x][9].(string)
		if !ok {
			return resp, errors.New("type assertion failed for taker buy base asset volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.TakerBuyBaseAssetVolume = floatData
		resp = append(resp, tempData)
	}
	return resp, nil
}

// GetIndexPriceKlines gets continuous kline data
func (b *Binance) GetIndexPriceKlines(pair, interval string, limit int64, startTime, endTime time.Time) ([]FuturesCandleStick, error) {
	var data [][10]interface{}
	var resp []FuturesCandleStick
	params := url.Values{}
	params.Set("pair", pair)
	if limit > 0 && limit <= 1500 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesIntervals, interval) {
		return resp, errors.New("invalid interval parsed")
	}
	params.Set("interval", interval)
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	rateBudget := getKlineRateBudget(limit)
	err := b.SendHTTPRequest(exchange.RestCoinMargined, cfuturesIndexKline+params.Encode(), rateBudget, &data)
	if err != nil {
		return resp, err
	}
	var floatData float64
	var strData string
	var ok bool
	var tempData FuturesCandleStick
	for x := range data {
		floatData, ok = data[x][0].(float64)
		if !ok {
			return resp, errors.New("type assertion failed for open time")
		}
		tempData.OpenTime = time.Unix(int64(floatData), 0)
		strData, ok = data[x][1].(string)
		if !ok {
			return resp, errors.New("type assertion failed for open")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Open = floatData
		strData, ok = data[x][2].(string)
		if !ok {
			return resp, errors.New("type assertion failed for high")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.High = floatData
		strData, ok = data[x][3].(string)
		if !ok {
			return resp, errors.New("type assertion failed for low")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Low = floatData
		strData, ok = data[x][4].(string)
		if !ok {
			return resp, errors.New("type assertion failed for close")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Close = floatData
		strData, ok = data[x][5].(string)
		if !ok {
			return resp, errors.New("type assertion failed for volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Volume = floatData
		floatData, ok = data[x][6].(float64)
		if !ok {
			return resp, errors.New("type assertion failed for close time")
		}
		tempData.CloseTime = time.Unix(int64(floatData), 0)
		strData, ok = data[x][7].(string)
		if !ok {
			return resp, errors.New("type assertion failed for base asset volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.BaseAssetVolume = floatData
		floatData, ok = data[x][8].(float64)
		if !ok {
			return resp, errors.New("type assertion failed for taker buy volume")
		}
		tempData.TakerBuyVolume = floatData
		strData, ok = data[x][9].(string)
		if !ok {
			return resp, errors.New("type assertion failed for taker buy base asset volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.TakerBuyBaseAssetVolume = floatData
		resp = append(resp, tempData)
	}
	return resp, nil
}

// GetMarkPriceKline gets mark price kline data
func (b *Binance) GetMarkPriceKline(symbol currency.Pair, interval string, limit int64, startTime, endTime time.Time) ([]FuturesCandleStick, error) {
	var data [][10]interface{}
	var resp []FuturesCandleStick
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 1500 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesIntervals, interval) {
		return resp, errors.New("invalid interval parsed")
	}
	params.Set("interval", interval)
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	rateBudget := getKlineRateBudget(limit)
	err = b.SendHTTPRequest(exchange.RestCoinMargined, cfuturesMarkPriceKline+params.Encode(), rateBudget, &data)
	if err != nil {
		return resp, err
	}
	var floatData float64
	var strData string
	var ok bool
	var tempData FuturesCandleStick
	for x := range data {
		floatData, ok = data[x][0].(float64)
		if !ok {
			return resp, errors.New("type assertion failed for open time")
		}
		tempData.OpenTime = time.Unix(int64(floatData), 0)
		strData, ok = data[x][1].(string)
		if !ok {
			return resp, errors.New("type assertion failed for open")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Open = floatData
		strData, ok = data[x][2].(string)
		if !ok {
			return resp, errors.New("type assertion failed for high")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.High = floatData
		strData, ok = data[x][3].(string)
		if !ok {
			return resp, errors.New("type assertion failed for low")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Low = floatData
		strData, ok = data[x][4].(string)
		if !ok {
			return resp, errors.New("type assertion failed for close")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Close = floatData
		strData, ok = data[x][5].(string)
		if !ok {
			return resp, errors.New("type assertion failed for volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.Volume = floatData
		floatData, ok = data[x][6].(float64)
		if !ok {
			return resp, errors.New("type assertion failed for close time")
		}
		tempData.CloseTime = time.Unix(int64(floatData), 0)
		strData, ok = data[x][7].(string)
		if !ok {
			return resp, errors.New("type assertion failed for base asset volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.BaseAssetVolume = floatData
		floatData, ok = data[x][8].(float64)
		if !ok {
			return resp, errors.New("type assertion failed for taker buy volume")
		}
		tempData.TakerBuyVolume = floatData
		strData, ok = data[x][9].(string)
		if !ok {
			return resp, errors.New("type assertion failed for taker buy base asset volume")
		}
		floatData, err = strconv.ParseFloat(strData, 64)
		if err != nil {
			return resp, err
		}
		tempData.TakerBuyBaseAssetVolume = floatData
		resp = append(resp, tempData)
	}
	return resp, nil
}

func getKlineRateBudget(limit int64) request.EndpointLimit {
	rateBudget := cFuturesDefaultRate
	switch {
	case limit > 0 && limit < 100:
		rateBudget = cFuturesKline100Rate
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
func (b *Binance) GetFuturesSwapTickerChangeStats(symbol currency.Pair, pair string) ([]PriceChangeStats, error) {
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
	return resp, b.SendHTTPRequest(exchange.RestCoinMargined, cfuturesTickerPriceStats+params.Encode(), rateLimit, &resp)
}

// FuturesGetFundingHistory gets funding history for CoinMarginedFutures,
func (b *Binance) FuturesGetFundingHistory(symbol currency.Pair, limit int64, startTime, endTime time.Time) ([]FundingRateHistory, error) {
	var resp []FundingRateHistory
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	if limit > 0 && limit < 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.RestCoinMargined, cfuturesFundingRateHistory+params.Encode(), cFuturesDefaultRate, &resp)
}

// GetFuturesSymbolPriceTicker gets price ticker for symbol
func (b *Binance) GetFuturesSymbolPriceTicker(symbol currency.Pair, pair string) ([]SymbolPriceTicker, error) {
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
	return resp, b.SendHTTPRequest(exchange.RestCoinMargined, cfuturesSymbolPriceTicker+params.Encode(), rateLimit, &resp)
}

// GetFuturesOrderbookTicker gets orderbook ticker for symbol
func (b *Binance) GetFuturesOrderbookTicker(symbol currency.Pair, pair string) ([]SymbolOrderBookTicker, error) {
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
	return resp, b.SendHTTPRequest(exchange.RestCoinMargined, cfuturesSymbolOrderbook+params.Encode(), rateLimit, &resp)
}

// GetFuturesLiquidationOrders gets forced liquidation orders
func (b *Binance) GetFuturesLiquidationOrders(symbol currency.Pair, pair string, limit int64, startTime, endTime time.Time) ([]AllLiquidationOrders, error) {
	var resp []AllLiquidationOrders
	params := url.Values{}
	rateLimit := cFuturesAllForceOrdersRate
	if !symbol.IsEmpty() {
		rateLimit = cFuturesCurrencyForceOrdersRate
		symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.RestCoinMargined, cfuturesLiquidationOrders+params.Encode(), rateLimit, &resp)
}

// GetOpenInterest gets open interest data for a symbol
func (b *Binance) GetOpenInterest(symbol currency.Pair) (OpenInterestData, error) {
	var resp OpenInterestData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	return resp, b.SendHTTPRequest(exchange.RestCoinMargined, cfuturesOpenInterest+params.Encode(), cFuturesDefaultRate, &resp)
}

// GetOpenInterestStats gets open interest stats for a symbol
func (b *Binance) GetOpenInterestStats(pair, contractType, period string, limit int64, startTime, endTime time.Time) ([]OpenInterestStats, error) {
	var resp []OpenInterestStats
	params := url.Values{}
	if pair != "" {
		params.Set("pair", pair)
	}
	if !common.StringDataCompare(validContractType, contractType) {
		return resp, errors.New("invalid contractType")
	}
	params.Set("contractType", contractType)
	if !common.StringDataCompare(validFuturesIntervals, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.RestCoinMargined, cfuturesOpenInterestStats+params.Encode(), cFuturesDefaultRate, &resp)
}

// GetTraderFuturesAccountRatio gets a traders futures account long/short ratio
func (b *Binance) GetTraderFuturesAccountRatio(pair, period string, limit int64, startTime, endTime time.Time) ([]TopTraderAccountRatio, error) {
	var resp []TopTraderAccountRatio
	params := url.Values{}
	params.Set("pair", pair)
	if !common.StringDataCompare(validFuturesIntervals, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.RestCoinMargined, cfuturesTopAccountsRatio+params.Encode(), cFuturesDefaultRate, &resp)
}

// GetTraderFuturesPositionsRatio gets a traders futures positions' long/short ratio
func (b *Binance) GetTraderFuturesPositionsRatio(pair, period string, limit int64, startTime, endTime time.Time) ([]TopTraderPositionRatio, error) {
	var resp []TopTraderPositionRatio
	params := url.Values{}
	params.Set("pair", pair)
	if !common.StringDataCompare(validFuturesIntervals, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.RestCoinMargined, cfuturesTopPositionsRatio+params.Encode(), cFuturesDefaultRate, &resp)
}

// GetMarketRatio gets global long/short ratio
func (b *Binance) GetMarketRatio(pair, period string, limit int64, startTime, endTime time.Time) ([]TopTraderPositionRatio, error) {
	var resp []TopTraderPositionRatio
	params := url.Values{}
	params.Set("pair", pair)
	if !common.StringDataCompare(validFuturesIntervals, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.RestCoinMargined, cfuturesLongShortRatio+params.Encode(), cFuturesDefaultRate, &resp)
}

// GetFuturesTakerVolume gets futures taker buy/sell volumes
func (b *Binance) GetFuturesTakerVolume(pair, contractType, period string, limit int64, startTime, endTime time.Time) ([]TakerBuySellVolume, error) {
	var resp []TakerBuySellVolume
	params := url.Values{}
	params.Set("pair", pair)
	if !common.StringDataCompare(validContractType, contractType) {
		return resp, errors.New("invalid contractType")
	}
	params.Set("contractType", contractType)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesIntervals, period) {
		return resp, errors.New("invalid period parsed")
	}
	params.Set("period", period)
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.RestCoinMargined, cfuturesBuySellVolume+params.Encode(), cFuturesDefaultRate, &resp)
}

// GetFuturesBasisData gets futures basis data
func (b *Binance) GetFuturesBasisData(pair, contractType, period string, limit int64, startTime, endTime time.Time) ([]FuturesBasisData, error) {
	var resp []FuturesBasisData
	params := url.Values{}
	params.Set("pair", pair)
	if !common.StringDataCompare(validContractType, contractType) {
		return resp, errors.New("invalid contractType")
	}
	params.Set("contractType", contractType)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesIntervals, period) {
		return resp, errors.New("invalid period parsed")
	}
	params.Set("period", period)
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendHTTPRequest(exchange.RestCoinMargined, cfuturesBasis+params.Encode(), cFuturesDefaultRate, &resp)
}

// FuturesNewOrder sends a new futures order to the exchange
func (b *Binance) FuturesNewOrder(symbol currency.Pair, side, positionSide, orderType, timeInForce,
	newClientOrderID, closePosition, workingType, newOrderRespType string,
	quantity, price, stopPrice, activationPrice, callbackRate float64, reduceOnly bool) (FuturesOrderPlaceData, error) {
	var resp FuturesOrderPlaceData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	params.Set("side", side)
	if positionSide != "" {
		if !common.StringDataCompare(validPositionSide, positionSide) {
			return resp, errors.New("invalid positionSide")
		}
		params.Set("positionSide", positionSide)
	}
	params.Set("type", orderType)
	params.Set("timeInForce", timeInForce)
	if reduceOnly {
		params.Set("reduceOnly", "true")
	}
	if newClientOrderID != "" {
		params.Set("newClientOrderID", newClientOrderID)
	}
	if closePosition != "" {
		params.Set("closePosition", closePosition)
	}
	if workingType != "" {
		if !common.StringDataCompare(validWorkingType, workingType) {
			return resp, errors.New("invalid workingType")
		}
		params.Set("workingType", workingType)
	}
	if newOrderRespType != "" {
		if !common.StringDataCompare(validNewOrderRespType, newOrderRespType) {
			return resp, errors.New("invalid newOrderRespType")
		}
		params.Set("newOrderRespType", newOrderRespType)
	}
	if quantity != 0 {
		params.Set("quantity", strconv.FormatFloat(quantity, 'f', -1, 64))
	}
	if price != 0 {
		params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	}
	if stopPrice != 0 {
		params.Set("stopPrice", strconv.FormatFloat(stopPrice, 'f', -1, 64))
	}
	if activationPrice != 0 {
		params.Set("activationPrice", strconv.FormatFloat(activationPrice, 'f', -1, 64))
	}
	if callbackRate != 0 {
		params.Set("callbackRate", strconv.FormatFloat(callbackRate, 'f', -1, 64))
	}
	return resp, b.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, cfuturesOrder, params, cFuturesOrdersDefaultRate, &resp)
}

// FuturesBatchOrder sends a batch order request
func (b *Binance) FuturesBatchOrder(data []PlaceBatchOrderData) ([]FuturesOrderPlaceData, error) {
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
			if !common.StringDataCompare(validPositionSide, data[x].PositionSide) {
				return resp, errors.New("invalid positionSide")
			}
		}
		if data[x].WorkingType != "" {
			if !common.StringDataCompare(validWorkingType, data[x].WorkingType) {
				return resp, errors.New("invalid workingType")
			}
		}
		if data[x].NewOrderRespType != "" {
			if !common.StringDataCompare(validNewOrderRespType, data[x].NewOrderRespType) {
				return resp, errors.New("invalid newOrderRespType")
			}
		}
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return resp, err
	}
	params.Set("batchOrders", string(jsonData))
	return resp, b.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, cfuturesBatchOrder, params, cFuturesBatchOrdersRate, &resp)
}

// FuturesBatchCancelOrders sends a batch request to cancel orders
func (b *Binance) FuturesBatchCancelOrders(symbol currency.Pair, orderList, origClientOrderIDList []string) ([]BatchCancelOrderData, error) {
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
	return resp, b.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodDelete, cfuturesBatchOrder, params, cFuturesOrdersDefaultRate, &resp)
}

// FuturesGetOrderData gets futures order data
func (b *Binance) FuturesGetOrderData(symbol currency.Pair, orderID, origClientOrderID string) (FuturesOrderGetData, error) {
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
	return resp, b.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, cfuturesOrder, params, cFuturesOrdersDefaultRate, &resp)
}

// FuturesCancelOrder cancels a futures order
func (b *Binance) FuturesCancelOrder(symbol currency.Pair, orderID, origClientOrderID string) (FuturesOrderGetData, error) {
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
	return resp, b.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodDelete, cfuturesOrder, params, cFuturesOrdersDefaultRate, &resp)
}

// FuturesCancelAllOpenOrders cancels a futures order
func (b *Binance) FuturesCancelAllOpenOrders(symbol currency.Pair) (GenericAuthResponse, error) {
	var resp GenericAuthResponse
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	return resp, b.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodDelete, cfuturesCancelAllOrders, params, cFuturesOrdersDefaultRate, &resp)
}

// AutoCancelAllOpenOrders cancels all open futures orders
// countdownTime 1000 = 1s, example - to cancel all orders after 30s (countdownTime: 30000)
func (b *Binance) AutoCancelAllOpenOrders(symbol currency.Pair, countdownTime int64) (AutoCancelAllOrdersData, error) {
	var resp AutoCancelAllOrdersData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	params.Set("countdownTime", strconv.FormatInt(countdownTime, 10))
	return resp, b.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, cfuturesCountdownCancel, params, cFuturesCancelAllOrdersRate, &resp)
}

// FuturesOpenOrderData gets open order data for CoinMarginedFutures,
func (b *Binance) FuturesOpenOrderData(symbol currency.Pair, orderID, origClientOrderID string) (FuturesOrderGetData, error) {
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
	return resp, b.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, cfuturesOpenOrder, params, cFuturesOrdersDefaultRate, &resp)
}

// GetFuturesAllOpenOrders gets all open orders data for CoinMarginedFutures,
func (b *Binance) GetFuturesAllOpenOrders(symbol currency.Pair, pair string) ([]FuturesOrderData, error) {
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
	return resp, b.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, cfuturesAllOpenOrders, params, rateLimit, &resp)
}

// GetAllFuturesOrders gets all orders active cancelled or filled
func (b *Binance) GetAllFuturesOrders(symbol currency.Pair, pair string, startTime, endTime time.Time, orderID, limit int64) ([]FuturesOrderData, error) {
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
	if pair != "" {
		params.Set("pair", pair)
	}
	if orderID != 0 {
		params.Set("orderID", strconv.FormatInt(orderID, 10))
	}
	if limit > 0 && limit <= 100 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, cfuturesAllOrders, params, rateLimit, &resp)
}

// GetFuturesAccountBalance gets account balance data for CoinMarginedFutures, account
func (b *Binance) GetFuturesAccountBalance() ([]FuturesAccountBalanceData, error) {
	var resp []FuturesAccountBalanceData
	return resp, b.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, cfuturesAccountBalance, nil, cFuturesDefaultRate, &resp)
}

// GetFuturesAccountInfo gets account info data for CoinMarginedFutures, account
func (b *Binance) GetFuturesAccountInfo() (FuturesAccountInformation, error) {
	var resp FuturesAccountInformation
	return resp, b.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, cfuturesAccountInfo, nil, cFuturesAccountInformationRate, &resp)
}

// FuturesChangeInitialLeverage changes initial leverage for the account
func (b *Binance) FuturesChangeInitialLeverage(symbol currency.Pair, leverage int64) (FuturesLeverageData, error) {
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
	params.Set("leverage", strconv.FormatInt(leverage, 10))
	return resp, b.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, cfuturesChangeInitialLeverage, params, cFuturesDefaultRate, &resp)
}

// FuturesChangeMarginType changes margin type
func (b *Binance) FuturesChangeMarginType(symbol currency.Pair, marginType string) (GenericAuthResponse, error) {
	var resp GenericAuthResponse
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if !common.StringDataCompare(validMarginType, marginType) {
		return resp, errors.New("invalid marginType")
	}
	params.Set("marginType", marginType)
	return resp, b.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, cfuturesChangeMarginType, params, cFuturesDefaultRate, &resp)
}

// ModifyIsolatedPositionMargin changes margin for an isolated position
func (b *Binance) ModifyIsolatedPositionMargin(symbol currency.Pair, positionSide, changeType string, amount float64) (GenericAuthResponse, error) {
	var resp GenericAuthResponse
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if !common.StringDataCompare(validPositionSide, positionSide) {
		return resp, errors.New("invalid positionSide")
	}
	params.Set("positionSide", positionSide)
	cType, ok := validMarginChange[changeType]
	if !ok {
		return resp, errors.New("invalid changeType")
	}
	params.Set("type", strconv.FormatInt(cType, 10))
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	return resp, b.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, cfuturesModifyMargin, params, cFuturesDefaultRate, &resp)
}

// FuturesMarginChangeHistory gets past margin changes for positions
func (b *Binance) FuturesMarginChangeHistory(symbol currency.Pair, changeType string, startTime, endTime time.Time, limit int64) ([]GetPositionMarginChangeHistoryData, error) {
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
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp, b.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, cfuturesMarginChangeHistory, params, cFuturesDefaultRate, &resp)
}

// FuturesPositionsInfo gets futures positions info
func (b *Binance) FuturesPositionsInfo(marginAsset, pair string) ([]FuturesPositionInformation, error) {
	var resp []FuturesPositionInformation
	params := url.Values{}
	if marginAsset != "" {
		params.Set("marginAsset", marginAsset)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	return resp, b.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, cfuturesPositionInfo, params, cFuturesDefaultRate, &resp)
}

// FuturesTradeHistory gets trade history for CoinMarginedFutures, account
func (b *Binance) FuturesTradeHistory(symbol currency.Pair, pair string, startTime, endTime time.Time, limit, fromID int64) ([]FuturesAccountTradeList, error) {
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
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if fromID != 0 {
		params.Set("fromId", strconv.FormatInt(fromID, 10))
	}
	return resp, b.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, cfuturesAccountTradeList, params, rateLimit, &resp)
}

// FuturesIncomeHistory gets income history for CoinMarginedFutures,
func (b *Binance) FuturesIncomeHistory(symbol currency.Pair, incomeType string, startTime, endTime time.Time, limit int64) ([]FuturesIncomeHistoryData, error) {
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
		if !common.StringDataCompare(validIncomeType, incomeType) {
			return resp, fmt.Errorf("invalid incomeType: %v", incomeType)
		}
		params.Set("incomeType", incomeType)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp, b.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, cfuturesIncomeHistory, params, cFuturesIncomeHistoryRate, &resp)
}

// FuturesNotionalBracket gets futures notional bracket
func (b *Binance) FuturesNotionalBracket(pair string) ([]NotionalBracketData, error) {
	var resp []NotionalBracketData
	params := url.Values{}
	if pair != "" {
		params.Set("pair", pair)
	}
	return resp, b.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, cfuturesNotionalBracket, params, cFuturesDefaultRate, &resp)
}

// FuturesForceOrders gets futures forced orders
func (b *Binance) FuturesForceOrders(symbol currency.Pair, autoCloseType string, startTime, endTime time.Time) ([]ForcedOrdersData, error) {
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
		if !common.StringDataCompare(validAutoCloseTypes, autoCloseType) {
			return resp, errors.New("invalid autoCloseType")
		}
		params.Set("autoCloseType", autoCloseType)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, b.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, cfuturesUsersForceOrders, params, cFuturesDefaultRate, &resp)
}

// FuturesPositionsADLEstimate estimates ADL on positions
func (b *Binance) FuturesPositionsADLEstimate(symbol currency.Pair) ([]ADLEstimateData, error) {
	var resp []ADLEstimateData
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	return resp, b.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, cfuturesADLQuantile, params, cFuturesAccountInformationRate, &resp)
}

// FetchCoinMarginExchangeLimits fetches coin margined order execution limits
func (b *Binance) FetchCoinMarginExchangeLimits() ([]order.MinMaxLevel, error) {
	var limits []order.MinMaxLevel
	coinFutures, err := b.FuturesExchangeInfo()
	if err != nil {
		return nil, err
	}

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
			Pair:              cp,
			Asset:             asset.CoinMarginedFutures,
			MinPrice:          coinFutures.Symbols[x].Filters[0].MinPrice,
			MaxPrice:          coinFutures.Symbols[x].Filters[0].MaxPrice,
			StepPrice:         coinFutures.Symbols[x].Filters[0].TickSize,
			MaxAmount:         coinFutures.Symbols[x].Filters[1].MaxQty,
			MinAmount:         coinFutures.Symbols[x].Filters[1].MinQty,
			StepAmount:        coinFutures.Symbols[x].Filters[1].StepSize,
			MarketMinQty:      coinFutures.Symbols[x].Filters[2].MinQty,
			MarketMaxQty:      coinFutures.Symbols[x].Filters[2].MaxQty,
			MarketStepSize:    coinFutures.Symbols[x].Filters[2].StepSize,
			MaxTotalOrders:    coinFutures.Symbols[x].Filters[3].Limit,
			MaxAlgoOrders:     coinFutures.Symbols[x].Filters[4].Limit,
			MultiplierUp:      coinFutures.Symbols[x].Filters[5].MultiplierUp,
			MultiplierDown:    coinFutures.Symbols[x].Filters[5].MultiplierDown,
			MultiplierDecimal: coinFutures.Symbols[x].Filters[5].MultiplierDecimal,
		})
	}
	return limits, nil
}
