package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (

	// Unauth
	ufuturesServerTime         = "/fapi/v1/time"
	ufuturesExchangeInfo       = "/fapi/v1/exchangeInfo?"
	ufuturesOrderbook          = "/fapi/v1/depth?"
	ufuturesRecentTrades       = "/fapi/v1/trades?"
	ufuturesHistoricalTrades   = "/fapi/v1/historicalTrades"
	ufuturesCompressedTrades   = "/fapi/v1/aggTrades?"
	ufuturesKlineData          = "/fapi/v1/klines?"
	ufuturesMarkPrice          = "/fapi/v1/premiumIndex?"
	ufuturesFundingRateHistory = "/fapi/v1/fundingRate?"
	ufuturesTickerPriceStats   = "/fapi/v1/ticker/24hr?"
	ufuturesSymbolPriceTicker  = "/fapi/v1/ticker/price?"
	ufuturesSymbolOrderbook    = "/fapi/v1/ticker/bookTicker?"
	ufuturesOpenInterest       = "/fapi/v1/openInterest?"
	ufuturesOpenInterestStats  = "/futures/data/openInterestHist?"
	ufuturesTopAccountsRatio   = "/futures/data/topLongShortAccountRatio?"
	ufuturesTopPositionsRatio  = "/futures/data/topLongShortPositionRatio?"
	ufuturesLongShortRatio     = "/futures/data/globalLongShortAccountRatio?"
	ufuturesBuySellVolume      = "/futures/data/takerlongshortRatio?"
	ufuturesCompositeIndexInfo = "/fapi/v1/indexInfo?"
	fundingRate                = "/fapi/v1/fundingRate?"

	// Auth
	ufuturesOrder                 = "/fapi/v1/order"
	ufuturesBatchOrder            = "/fapi/v1/batchOrders"
	ufuturesCancelAllOrders       = "/fapi/v1/allOpenOrders"
	ufuturesCountdownCancel       = "/fapi/v1/countdownCancelAll"
	ufuturesOpenOrder             = "/fapi/v1/openOrder"
	ufuturesAllOpenOrders         = "/fapi/v1/openOrders"
	ufuturesAllOrders             = "/fapi/v1/allOrders"
	ufuturesAccountBalance        = "/fapi/v2/balance"
	ufuturesAccountInfo           = "/fapi/v2/account"
	ufuturesChangeInitialLeverage = "/fapi/v1/leverage"
	ufuturesChangeMarginType      = "/fapi/v1/marginType"
	ufuturesModifyMargin          = "/fapi/v1/positionMargin"
	ufuturesMarginChangeHistory   = "/fapi/v1/positionMargin/history"
	ufuturesPositionInfo          = "/fapi/v2/positionRisk"
	ufuturesAccountTradeList      = "/fapi/v1/userTrades"
	ufuturesIncomeHistory         = "/fapi/v1/income"
	ufuturesNotionalBracket       = "/fapi/v1/leverageBracket"
	ufuturesUsersForceOrders      = "/fapi/v1/forceOrders"
	ufuturesADLQuantile           = "/fapi/v1/adlQuantile"
)

// UServerTime gets the server time
func (b *Binance) UServerTime(ctx context.Context) (time.Time, error) {
	var data struct {
		ServerTime int64 `json:"serverTime"`
	}
	err := b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesServerTime, uFuturesDefaultRate, &data)
	if err != nil {
		return time.Time{}, err
	}
	return time.UnixMilli(data.ServerTime), nil
}

// UExchangeInfo stores usdt margined futures data
func (b *Binance) UExchangeInfo(ctx context.Context) (UFuturesExchangeInfo, error) {
	var resp UFuturesExchangeInfo
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesExchangeInfo, uFuturesDefaultRate, &resp)
}

// UFuturesOrderbook gets orderbook data for usdt margined futures
func (b *Binance) UFuturesOrderbook(ctx context.Context, symbol currency.Pair, limit int64) (*OrderBook, error) {
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("symbol", symbolValue)
	strLimit := strconv.FormatInt(limit, 10)
	if strLimit != "" {
		if !common.StringDataCompare(uValidOBLimits, strLimit) {
			return nil, fmt.Errorf("invalid limit: %v", limit)
		}
		params.Set("limit", strLimit)
	}

	rateBudget := uFuturesDefaultRate
	switch {
	case limit == 5, limit == 10, limit == 20, limit == 50:
		rateBudget = uFuturesOrderbook50Rate
	case limit >= 100 && limit < 500:
		rateBudget = uFuturesOrderbook100Rate
	case limit >= 500 && limit < 1000:
		rateBudget = uFuturesOrderbook500Rate
	case limit == 1000:
		rateBudget = uFuturesOrderbook1000Rate
	}

	var data OrderbookData
	err = b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesOrderbook+params.Encode(), rateBudget, &data)
	if err != nil {
		return nil, err
	}

	resp := OrderBook{
		Symbol:       symbolValue,
		LastUpdateID: data.LastUpdateID,
		Bids:         make([]OrderbookItem, len(data.Bids)),
		Asks:         make([]OrderbookItem, len(data.Asks)),
	}

	var price, quantity float64
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

// URecentTrades gets recent trades for usdt margined futures
func (b *Binance) URecentTrades(ctx context.Context, symbol currency.Pair, fromID string, limit int64) ([]UPublicTradesData, error) {
	var resp []UPublicTradesData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
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
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesRecentTrades+params.Encode(), uFuturesDefaultRate, &resp)
}

// UFuturesHistoricalTrades gets historical public trades for USDTMarginedFutures
func (b *Binance) UFuturesHistoricalTrades(ctx context.Context, symbol currency.Pair, fromID string, limit int64) ([]interface{}, error) {
	var resp []interface{}
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
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
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesHistoricalTrades, params, uFuturesHistoricalTradesRate, &resp)
}

// UCompressedTrades gets compressed public trades for usdt margined futures
func (b *Binance) UCompressedTrades(ctx context.Context, symbol currency.Pair, fromID string, limit int64, startTime, endTime time.Time) ([]UCompressedTradeData, error) {
	var resp []UCompressedTradeData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
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
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesCompressedTrades+params.Encode(), uFuturesHistoricalTradesRate, &resp)
}

// UKlineData gets kline data for usdt margined futures
func (b *Binance) UKlineData(ctx context.Context, symbol currency.Pair, interval string, limit int64, startTime, endTime time.Time) ([]FuturesCandleStick, error) {
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbolValue)
	if !common.StringDataCompare(validFuturesIntervals, interval) {
		return nil, errors.New("invalid interval")
	}
	params.Set("interval", interval)
	if limit > 0 && limit <= 1500 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return nil, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", timeString(startTime))
		params.Set("endTime", timeString(endTime))
	}
	rateBudget := uFuturesDefaultRate
	switch {
	case limit > 0 && limit <= 100:
		rateBudget = uFuturesKline100Rate
	case limit > 100 && limit <= 500:
		rateBudget = uFuturesKline500Rate
	case limit > 500 && limit <= 1000:
		rateBudget = uFuturesKline1000Rate
	case limit > 1000:
		rateBudget = uFuturesKlineMaxRate
	}

	var data [][10]interface{}
	err = b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesKlineData+params.Encode(), rateBudget, &data)
	if err != nil {
		return nil, err
	}

	resp := make([]FuturesCandleStick, len(data))
	for x := range data {
		var tempData FuturesCandleStick
		var floatData float64
		var strData string
		var ok bool

		floatData, ok = data[x][0].(float64)
		if !ok {
			return nil, errors.New("type assertion failed for opentime")
		}
		tempData.OpenTime, err = convert.TimeFromUnixTimestampFloat(floatData)
		if err != nil {
			return nil, err
		}
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
		tempData.CloseTime, err = convert.TimeFromUnixTimestampFloat(floatData)
		if err != nil {
			return resp, err
		}
		strData, ok = data[x][7].(string)
		if !ok {
			return nil, errors.New("type assertion failed base asset volume")
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

// UGetMarkPrice gets mark price data for USDTMarginedFutures
func (b *Binance) UGetMarkPrice(ctx context.Context, symbol currency.Pair) ([]UMarkPrice, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		var tempResp UMarkPrice
		err = b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesMarkPrice+params.Encode(), uFuturesDefaultRate, &tempResp)
		if err != nil {
			return nil, err
		}
		return []UMarkPrice{tempResp}, nil
	}
	var resp []UMarkPrice
	err := b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesMarkPrice+params.Encode(), uFuturesDefaultRate, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// UGetFundingHistory gets funding history for USDTMarginedFutures
func (b *Binance) UGetFundingHistory(ctx context.Context, symbol currency.Pair, limit int64, startTime, endTime time.Time) ([]FundingRateHistory, error) {
	var resp []FundingRateHistory
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
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
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesFundingRateHistory+params.Encode(), uFuturesDefaultRate, &resp)
}

// U24HTickerPriceChangeStats gets 24hr ticker price change stats for USDTMarginedFutures
func (b *Binance) U24HTickerPriceChangeStats(ctx context.Context, symbol currency.Pair) ([]U24HrPriceChangeStats, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		var tempResp U24HrPriceChangeStats
		err = b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesTickerPriceStats+params.Encode(), uFuturesDefaultRate, &tempResp)
		if err != nil {
			return nil, err
		}
		return []U24HrPriceChangeStats{tempResp}, err
	}
	var resp []U24HrPriceChangeStats
	err := b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesTickerPriceStats+params.Encode(), uFuturesTickerPriceHistoryRate, &resp)
	return resp, err
}

// USymbolPriceTicker gets symbol price ticker for USDTMarginedFutures
func (b *Binance) USymbolPriceTicker(ctx context.Context, symbol currency.Pair) ([]USymbolPriceTicker, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		var tempResp USymbolPriceTicker
		err = b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesSymbolPriceTicker+params.Encode(), uFuturesDefaultRate, &tempResp)
		if err != nil {
			return nil, err
		}
		return []USymbolPriceTicker{tempResp}, err
	}
	var resp []USymbolPriceTicker
	err := b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesSymbolPriceTicker+params.Encode(), uFuturesOrderbookTickerAllRate, &resp)
	return resp, err
}

// USymbolOrderbookTicker gets symbol orderbook ticker
func (b *Binance) USymbolOrderbookTicker(ctx context.Context, symbol currency.Pair) ([]USymbolOrderbookTicker, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		var tempResp USymbolOrderbookTicker
		err = b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesSymbolOrderbook+params.Encode(), uFuturesDefaultRate, &tempResp)
		if err != nil {
			return nil, err
		}
		return []USymbolOrderbookTicker{tempResp}, err
	}
	var resp []USymbolOrderbookTicker
	err := b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesTickerPriceStats+params.Encode(), uFuturesOrderbookTickerAllRate, &resp)
	return resp, err
}

// UOpenInterest gets open interest data for USDTMarginedFutures
func (b *Binance) UOpenInterest(ctx context.Context, symbol currency.Pair) (UOpenInterestData, error) {
	var resp UOpenInterestData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesOpenInterest+params.Encode(), uFuturesDefaultRate, &resp)
}

// UOpenInterestStats gets open interest stats for USDTMarginedFutures
func (b *Binance) UOpenInterestStats(ctx context.Context, symbol currency.Pair, period string, limit int64, startTime, endTime time.Time) ([]UOpenInterestStats, error) {
	var resp []UOpenInterestStats
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if !common.StringDataCompare(uValidPeriods, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	if limit > 0 && limit < 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesOpenInterestStats+params.Encode(), uFuturesDefaultRate, &resp)
}

// UTopAcccountsLongShortRatio gets long/short ratio data for top trader accounts in ufutures
func (b *Binance) UTopAcccountsLongShortRatio(ctx context.Context, symbol currency.Pair, period string, limit int64, startTime, endTime time.Time) ([]ULongShortRatio, error) {
	var resp []ULongShortRatio
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if !common.StringDataCompare(uValidPeriods, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	if limit > 0 && limit < 500 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesTopAccountsRatio+params.Encode(), uFuturesDefaultRate, &resp)
}

// UTopPostionsLongShortRatio gets long/short ratio data for top positions' in ufutures
func (b *Binance) UTopPostionsLongShortRatio(ctx context.Context, symbol currency.Pair, period string, limit int64, startTime, endTime time.Time) ([]ULongShortRatio, error) {
	var resp []ULongShortRatio
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if !common.StringDataCompare(uValidPeriods, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	if limit > 0 && limit < 500 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesTopPositionsRatio+params.Encode(), uFuturesDefaultRate, &resp)
}

// UGlobalLongShortRatio gets the global long/short ratio data for USDTMarginedFutures
func (b *Binance) UGlobalLongShortRatio(ctx context.Context, symbol currency.Pair, period string, limit int64, startTime, endTime time.Time) ([]ULongShortRatio, error) {
	var resp []ULongShortRatio
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if !common.StringDataCompare(uValidPeriods, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	if limit > 0 && limit < 500 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesLongShortRatio+params.Encode(), uFuturesDefaultRate, &resp)
}

// UTakerBuySellVol gets takers' buy/sell ratio for USDTMarginedFutures
func (b *Binance) UTakerBuySellVol(ctx context.Context, symbol currency.Pair, period string, limit int64, startTime, endTime time.Time) ([]UTakerVolumeData, error) {
	var resp []UTakerVolumeData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if !common.StringDataCompare(uValidPeriods, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	if limit > 0 && limit < 500 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesBuySellVolume+params.Encode(), uFuturesDefaultRate, &resp)
}

// UCompositeIndexInfo stores composite indexs' info for usdt margined futures
func (b *Binance) UCompositeIndexInfo(ctx context.Context, symbol currency.Pair) ([]UCompositeIndexInfoData, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		var tempResp UCompositeIndexInfoData
		err = b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesCompositeIndexInfo+params.Encode(), uFuturesDefaultRate, &tempResp)
		if err != nil {
			return nil, err
		}
		return []UCompositeIndexInfoData{tempResp}, err
	}
	var resp []UCompositeIndexInfoData
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesCompositeIndexInfo+params.Encode(), uFuturesDefaultRate, &resp)
}

// UFuturesNewOrder sends a new order for USDTMarginedFutures
func (b *Binance) UFuturesNewOrder(ctx context.Context, data *UFuturesNewOrderRequest) (UOrderData, error) {
	var resp UOrderData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(data.Symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	params.Set("side", data.Side)
	if data.PositionSide != "" {
		if !common.StringDataCompare(validPositionSide, data.PositionSide) {
			return resp, errors.New("invalid positionSide")
		}
		params.Set("positionSide", data.PositionSide)
	}
	params.Set("type", data.OrderType)
	params.Set("timeInForce", data.TimeInForce)
	if data.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	if data.NewClientOrderID != "" {
		params.Set("newClientOrderID", data.NewClientOrderID)
	}
	if data.ClosePosition != "" {
		params.Set("closePosition", data.ClosePosition)
	}
	if data.WorkingType != "" {
		if !common.StringDataCompare(validWorkingType, data.WorkingType) {
			return resp, errors.New("invalid workingType")
		}
		params.Set("workingType", data.WorkingType)
	}
	if data.NewOrderRespType != "" {
		if !common.StringDataCompare(validNewOrderRespType, data.NewOrderRespType) {
			return resp, errors.New("invalid newOrderRespType")
		}
		params.Set("newOrderRespType", data.NewOrderRespType)
	}
	if data.Quantity != 0 {
		params.Set("quantity", strconv.FormatFloat(data.Quantity, 'f', -1, 64))
	}
	if data.Price != 0 {
		params.Set("price", strconv.FormatFloat(data.Price, 'f', -1, 64))
	}
	if data.StopPrice != 0 {
		params.Set("stopPrice", strconv.FormatFloat(data.StopPrice, 'f', -1, 64))
	}
	if data.ActivationPrice != 0 {
		params.Set("activationPrice", strconv.FormatFloat(data.ActivationPrice, 'f', -1, 64))
	}
	if data.CallbackRate != 0 {
		params.Set("callbackRate", strconv.FormatFloat(data.CallbackRate, 'f', -1, 64))
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesOrder, params, uFuturesOrdersDefaultRate, &resp)
}

// UPlaceBatchOrders places batch orders
func (b *Binance) UPlaceBatchOrders(ctx context.Context, data []PlaceBatchOrderData) ([]UOrderData, error) {
	var resp []UOrderData
	params := url.Values{}
	for x := range data {
		unformattedPair, err := currency.NewPairFromString(data[x].Symbol)
		if err != nil {
			return resp, err
		}
		formattedPair, err := b.FormatExchangeCurrency(unformattedPair, asset.USDTMarginedFutures)
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
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesBatchOrder, params, uFuturesBatchOrdersRate, &resp)
}

// UGetOrderData gets order data for USDTMarginedFutures
func (b *Binance) UGetOrderData(ctx context.Context, symbol currency.Pair, orderID, cliOrderID string) (UOrderData, error) {
	var resp UOrderData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if cliOrderID != "" {
		params.Set("origClientOrderId", cliOrderID)
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesOrder, params, uFuturesOrdersDefaultRate, &resp)
}

// UCancelOrder cancel an order for USDTMarginedFutures
func (b *Binance) UCancelOrder(ctx context.Context, symbol currency.Pair, orderID, cliOrderID string) (UOrderData, error) {
	var resp UOrderData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if cliOrderID != "" {
		params.Set("origClientOrderId", cliOrderID)
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodDelete, ufuturesOrder, params, uFuturesOrdersDefaultRate, &resp)
}

// UCancelAllOpenOrders cancels all open orders for a symbol ufutures
func (b *Binance) UCancelAllOpenOrders(ctx context.Context, symbol currency.Pair) (GenericAuthResponse, error) {
	var resp GenericAuthResponse
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodDelete, ufuturesCancelAllOrders, params, uFuturesOrdersDefaultRate, &resp)
}

// UCancelBatchOrders cancel batch order for USDTMarginedFutures
func (b *Binance) UCancelBatchOrders(ctx context.Context, symbol currency.Pair, orderIDList, origCliOrdIDList []string) ([]UOrderData, error) {
	var resp []UOrderData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if len(orderIDList) != 0 {
		jsonOrders, err := json.Marshal(orderIDList)
		if err != nil {
			return resp, err
		}
		params.Set("orderIdList", string(jsonOrders))
	}
	if len(origCliOrdIDList) != 0 {
		jsonCliOrders, err := json.Marshal(origCliOrdIDList)
		if err != nil {
			return resp, err
		}
		params.Set("origClientOrderIdList", string(jsonCliOrders))
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodDelete, ufuturesBatchOrder, params, uFuturesOrdersDefaultRate, &resp)
}

// UAutoCancelAllOpenOrders auto cancels all ufutures open orders for a symbol after the set countdown time
func (b *Binance) UAutoCancelAllOpenOrders(ctx context.Context, symbol currency.Pair, countdownTime int64) (AutoCancelAllOrdersData, error) {
	var resp AutoCancelAllOrdersData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	params.Set("countdownTime", strconv.FormatInt(countdownTime, 10))
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesCountdownCancel, params, uFuturesCountdownCancelRate, &resp)
}

// UFetchOpenOrder sends a request to fetch open order data for USDTMarginedFutures
func (b *Binance) UFetchOpenOrder(ctx context.Context, symbol currency.Pair, orderID, origClientOrderID string) (UOrderData, error) {
	var resp UOrderData
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesOpenOrder, params, uFuturesOrdersDefaultRate, &resp)
}

// UAllAccountOpenOrders gets all account's orders for USDTMarginedFutures
func (b *Binance) UAllAccountOpenOrders(ctx context.Context, symbol currency.Pair) ([]UOrderData, error) {
	var resp []UOrderData
	params := url.Values{}
	rateLimit := uFuturesGetAllOpenOrdersRate
	if !symbol.IsEmpty() {
		rateLimit = uFuturesOrdersDefaultRate
		p, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", p)
	} else {
		// extend the receive window when all currencies to prevent "recvwindow" error
		params.Set("recvWindow", "10000")
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesAllOpenOrders, params, rateLimit, &resp)
}

// UAllAccountOrders gets all account's orders for USDTMarginedFutures
func (b *Binance) UAllAccountOrders(ctx context.Context, symbol currency.Pair, orderID, limit int64, startTime, endTime time.Time) ([]UFuturesOrderData, error) {
	var resp []UFuturesOrderData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if limit > 0 && limit < 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesAllOrders, params, uFuturesGetAllOrdersRate, &resp)
}

// UAccountBalanceV2 gets V2 account balance data
func (b *Binance) UAccountBalanceV2(ctx context.Context) ([]UAccountBalanceV2Data, error) {
	var resp []UAccountBalanceV2Data
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesAccountBalance, nil, uFuturesOrdersDefaultRate, &resp)
}

// UAccountInformationV2 gets V2 account balance data
func (b *Binance) UAccountInformationV2(ctx context.Context) (UAccountInformationV2Data, error) {
	var resp UAccountInformationV2Data
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesAccountInfo, nil, uFuturesAccountInformationRate, &resp)
}

// UChangeInitialLeverageRequest sends a request to change account's initial leverage
func (b *Binance) UChangeInitialLeverageRequest(ctx context.Context, symbol currency.Pair, leverage int64) (UChangeInitialLeverage, error) {
	var resp UChangeInitialLeverage
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if leverage < 1 || leverage > 125 {
		return resp, errors.New("invalid leverage")
	}
	params.Set("leverage", strconv.FormatInt(leverage, 10))
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesChangeInitialLeverage, params, uFuturesDefaultRate, &resp)
}

// UChangeInitialMarginType sends a request to change account's initial margin type
func (b *Binance) UChangeInitialMarginType(ctx context.Context, symbol currency.Pair, marginType string) error {
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	params.Set("symbol", symbolValue)
	if !common.StringDataCompare(validMarginType, marginType) {
		return errors.New("invalid marginType")
	}
	params.Set("marginType", marginType)
	return b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesChangeMarginType, params, uFuturesDefaultRate, nil)
}

// UModifyIsolatedPositionMarginReq sends a request to modify isolated margin for USDTMarginedFutures
func (b *Binance) UModifyIsolatedPositionMarginReq(ctx context.Context, symbol currency.Pair, positionSide, changeType string, amount float64) (UModifyIsolatedPosMargin, error) {
	var resp UModifyIsolatedPosMargin
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if positionSide != "" {
		if !common.StringDataCompare(validPositionSide, positionSide) {
			return resp, errors.New("invalid margin changeType")
		}
	}
	cType, ok := validMarginChange[changeType]
	if !ok {
		return resp, errors.New("invalid margin changeType")
	}
	params.Set("type", strconv.FormatInt(cType, 10))
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesModifyMargin, params, uFuturesDefaultRate, &resp)
}

// UPositionMarginChangeHistory gets margin change history for USDTMarginedFutures
func (b *Binance) UPositionMarginChangeHistory(ctx context.Context, symbol currency.Pair, changeType string, limit int64, startTime, endTime time.Time) ([]UPositionMarginChangeHistoryData, error) {
	var resp []UPositionMarginChangeHistoryData
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	cType, ok := validMarginChange[changeType]
	if !ok {
		return resp, errors.New("invalid margin changeType")
	}
	params.Set("type", strconv.FormatInt(cType, 10))
	if limit > 0 && limit < 500 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesMarginChangeHistory, params, uFuturesDefaultRate, &resp)
}

// UPositionsInfoV2 gets positions' info for USDTMarginedFutures
func (b *Binance) UPositionsInfoV2(ctx context.Context, symbol currency.Pair) ([]UPositionInformationV2, error) {
	var resp []UPositionInformationV2
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesPositionInfo, params, uFuturesDefaultRate, &resp)
}

// UAccountTradesHistory gets account's trade history data for USDTMarginedFutures
func (b *Binance) UAccountTradesHistory(ctx context.Context, symbol currency.Pair, fromID string, limit int64, startTime, endTime time.Time) ([]UAccountTradeHistory, error) {
	var resp []UAccountTradeHistory
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
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
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesAccountTradeList, params, uFuturesAccountInformationRate, &resp)
}

// UAccountIncomeHistory gets account's income history data for USDTMarginedFutures
func (b *Binance) UAccountIncomeHistory(ctx context.Context, symbol currency.Pair, incomeType string, limit int64, startTime, endTime time.Time) ([]UAccountIncomeHistory, error) {
	var resp []UAccountIncomeHistory
	params := url.Values{}
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if incomeType != "" {
		if !common.StringDataCompare(validIncomeType, incomeType) {
			return resp, errors.New("invalid incomeType")
		}
		params.Set("incomeType", incomeType)
	}
	if limit > 0 && limit < 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesIncomeHistory, params, uFuturesIncomeHistoryRate, &resp)
}

// UGetNotionalAndLeverageBrackets gets account's notional and leverage brackets for USDTMarginedFutures
func (b *Binance) UGetNotionalAndLeverageBrackets(ctx context.Context, symbol currency.Pair) ([]UNotionalLeverageAndBrakcetsData, error) {
	var resp []UNotionalLeverageAndBrakcetsData
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesNotionalBracket, params, uFuturesDefaultRate, &resp)
}

// UPositionsADLEstimate gets estimated ADL data for USDTMarginedFutures positions
func (b *Binance) UPositionsADLEstimate(ctx context.Context, symbol currency.Pair) (UPositionADLEstimationData, error) {
	var resp UPositionADLEstimationData
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesADLQuantile, params, uFuturesAccountInformationRate, &resp)
}

// UAccountForcedOrders gets account's forced (liquidation) orders for USDTMarginedFutures
func (b *Binance) UAccountForcedOrders(ctx context.Context, symbol currency.Pair, autoCloseType string, limit int64, startTime, endTime time.Time) ([]UForceOrdersData, error) {
	var resp []UForceOrdersData
	params := url.Values{}
	rateLimit := uFuturesAllForceOrdersRate
	if !symbol.IsEmpty() {
		rateLimit = uFuturesCurrencyForceOrdersRate
		symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	}
	if autoCloseType != "" {
		if !common.StringDataCompare(validAutoCloseTypes, autoCloseType) {
			return resp, errors.New("invalid incomeType")
		}
		params.Set("autoCloseType", autoCloseType)
	}
	if limit > 0 && limit < 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesUsersForceOrders, params, rateLimit, &resp)
}

// GetPerpMarkets returns exchange information. Check binance_types for more information
func (b *Binance) GetPerpMarkets(ctx context.Context) (PerpsExchangeInfo, error) {
	var resp PerpsExchangeInfo
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, perpExchangeInfo, uFuturesDefaultRate, &resp)
}

// FetchUSDTMarginExchangeLimits fetches USDT margined order execution limits
func (b *Binance) FetchUSDTMarginExchangeLimits(ctx context.Context) ([]order.MinMaxLevel, error) {
	usdtFutures, err := b.UExchangeInfo(ctx)
	if err != nil {
		return nil, err
	}

	limits := make([]order.MinMaxLevel, 0, len(usdtFutures.Symbols))
	for x := range usdtFutures.Symbols {
		var cp currency.Pair
		cp, err = currency.NewPairFromStrings(usdtFutures.Symbols[x].BaseAsset,
			usdtFutures.Symbols[x].QuoteAsset)
		if err != nil {
			return nil, err
		}

		if len(usdtFutures.Symbols[x].Filters) < 7 {
			continue
		}

		limits = append(limits, order.MinMaxLevel{
			Pair:                    cp,
			Asset:                   asset.USDTMarginedFutures,
			MinPrice:                usdtFutures.Symbols[x].Filters[0].MinPrice,
			MaxPrice:                usdtFutures.Symbols[x].Filters[0].MaxPrice,
			PriceStepIncrementSize:  usdtFutures.Symbols[x].Filters[0].TickSize,
			MaximumBaseAmount:       usdtFutures.Symbols[x].Filters[1].MaxQty,
			MinimumBaseAmount:       usdtFutures.Symbols[x].Filters[1].MinQty,
			AmountStepIncrementSize: usdtFutures.Symbols[x].Filters[1].StepSize,
			MarketMinQty:            usdtFutures.Symbols[x].Filters[2].MinQty,
			MarketMaxQty:            usdtFutures.Symbols[x].Filters[2].MaxQty,
			MarketStepIncrementSize: usdtFutures.Symbols[x].Filters[2].StepSize,
			MaxTotalOrders:          usdtFutures.Symbols[x].Filters[3].Limit,
			MaxAlgoOrders:           usdtFutures.Symbols[x].Filters[4].Limit,
			MinNotional:             usdtFutures.Symbols[x].Filters[5].Notional,
			MultiplierUp:            usdtFutures.Symbols[x].Filters[6].MultiplierUp,
			MultiplierDown:          usdtFutures.Symbols[x].Filters[6].MultiplierDown,
			MultiplierDecimal:       usdtFutures.Symbols[x].Filters[6].MultiplierDecimal,
		})
	}
	return limits, nil
}
