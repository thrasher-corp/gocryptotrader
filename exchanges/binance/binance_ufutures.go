package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (

	// Unauth
	ufuturesServerTime         = "/fapi/v1/time"
	ufuturesExchangeInfo       = "/fapi/v1/exchangeInfo"
	ufuturesOrderbook          = "/fapi/v1/depth"
	ufuturesRecentTrades       = "/fapi/v1/trades"
	ufuturesHistoricalTrades   = "/fapi/v1/historicalTrades"
	ufuturesCompressedTrades   = "/fapi/v1/aggTrades"
	ufuturesKlineData          = "/fapi/v1/klines"
	ufuturesMarkPrice          = "/fapi/v1/premiumIndex"
	ufuturesFundingRateHistory = "/fapi/v1/fundingRate"
	ufuturesFundingRateInfo    = "/fapi/v1/fundingInfo"
	ufuturesTickerPriceStats   = "/fapi/v1/ticker/24hr"
	ufuturesSymbolPriceTicker  = "/fapi/v1/ticker/price"
	ufuturesSymbolOrderbook    = "/fapi/v1/ticker/bookTicker"
	ufuturesOpenInterest       = "/fapi/v1/openInterest"
	ufuturesOpenInterestStats  = "/futures/data/openInterestHist"
	ufuturesTopAccountsRatio   = "/futures/data/topLongShortAccountRatio"
	ufuturesTopPositionsRatio  = "/futures/data/topLongShortPositionRatio"
	ufuturesLongShortRatio     = "/futures/data/globalLongShortAccountRatio"
	ufuturesBuySellVolume      = "/futures/data/takerlongshortRatio"
	ufuturesCompositeIndexInfo = "/fapi/v1/indexInfo"
	fundingRate                = "/fapi/v1/fundingRate"

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
	ufuturesCommissionRate        = "/fapi/v1/commissionRate"
	ufuturesAccountTradeList      = "/fapi/v1/userTrades"
	ufuturesIncomeHistory         = "/fapi/v1/income"
	ufuturesNotionalBracket       = "/fapi/v1/leverageBracket"
	ufuturesUsersForceOrders      = "/fapi/v1/forceOrders"
	ufuturesADLQuantile           = "/fapi/v1/adlQuantile"
	uFuturesMultiAssetsMargin     = "/fapi/v1/multiAssetsMargin"
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
func (b *Binance) UExchangeInfo(ctx context.Context) (*UFuturesExchangeInfo, error) {
	var resp *UFuturesExchangeInfo
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesExchangeInfo, uFuturesDefaultRate, &resp)
}

// UFuturesOrderbook gets orderbook data for usdt margined futures
func (b *Binance) UFuturesOrderbook(ctx context.Context, symbol string, limit int64) (*OrderBook, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}

	params := url.Values{}
	params.Set("symbol", symbol)
	strLimit := strconv.FormatInt(limit, 10)
	if strLimit != "" {
		if !slices.Contains(uValidOBLimits, strLimit) {
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
	err := b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesOrderbook, params), rateBudget, &data)
	if err != nil {
		return nil, err
	}

	resp := OrderBook{
		Symbol:       symbol,
		LastUpdateID: data.LastUpdateID,
		Bids:         make([]OrderbookItem, len(data.Bids)),
		Asks:         make([]OrderbookItem, len(data.Asks)),
	}
	for x := range data.Asks {
		resp.Asks[x] = OrderbookItem{
			Price:    data.Asks[x][0].Float64(),
			Quantity: data.Asks[x][1].Float64(),
		}
	}
	for y := range data.Bids {
		resp.Bids[y] = OrderbookItem{
			Price:    data.Bids[y][0].Float64(),
			Quantity: data.Bids[y][1].Float64(),
		}
	}
	return &resp, nil
}

// URecentTrades gets recent trades for usdt margined futures
func (b *Binance) URecentTrades(ctx context.Context, symbol, fromID string, limit int64) ([]UPublicTradesData, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if fromID != "" {
		params.Set("fromID", fromID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []UPublicTradesData
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesRecentTrades, params), uFuturesDefaultRate, &resp)
}

// UFuturesHistoricalTrades gets historical public trades for USDTMarginedFutures
func (b *Binance) UFuturesHistoricalTrades(ctx context.Context, symbol, fromID string, limit int64) ([]UPublicTradesData, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if fromID != "" {
		params.Set("fromID", fromID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []UPublicTradesData
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesHistoricalTrades, params, uFuturesHistoricalTradesRate, nil, &resp)
}

// UCompressedTrades gets compressed public trades for usdt margined futures
func (b *Binance) UCompressedTrades(ctx context.Context, symbol, fromID string, limit int64, startTime, endTime time.Time) ([]UCompressedTradeData, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if fromID != "" {
		params.Set("fromID", fromID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []UCompressedTradeData
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesCompressedTrades, params), uFuturesHistoricalTradesRate, &resp)
}

// UKlineData gets kline data for usdt margined futures
func (b *Binance) UKlineData(ctx context.Context, symbol, interval string, limit int64, startTime, endTime time.Time) ([]FuturesCandleStick, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if !slices.Contains(validFuturesIntervals, interval) {
		return nil, kline.ErrUnsupportedInterval
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", interval)
	var err error
	if !startTime.IsZero() && !endTime.IsZero() {
		err = common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", timeString(startTime))
		params.Set("endTime", timeString(endTime))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
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

	var data [][12]types.Number
	err = b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesKlineData, params), rateBudget, &data)
	if err != nil {
		return nil, err
	}
	resp := make([]FuturesCandleStick, len(data))
	for x := range data {
		resp[x] = FuturesCandleStick{
			OpenTime:                time.UnixMilli(data[x][0].Int64()),
			Open:                    data[x][1].Float64(),
			High:                    data[x][2].Float64(),
			Low:                     data[x][3].Float64(),
			Close:                   data[x][4].Float64(),
			Volume:                  data[x][5].Float64(),
			CloseTime:               time.UnixMilli(data[x][6].Int64()),
			QuoteAssetVolume:        data[x][7].Float64(),
			NumberOfTrades:          data[x][8].Int64(),
			TakerBuyVolume:          data[x][9].Float64(),
			TakerBuyBaseAssetVolume: data[x][10].Float64(),
		}
	}
	return resp, nil
}

// GetUFuturesContinuousKlineData kline/candlestick bars for a specific contract type.
// Klines are uniquely identified by their open time.
func (b *Binance) GetUFuturesContinuousKlineData(ctx context.Context, pair currency.Pair, contractType, interval string, startTime, endTime time.Time, limit int64) (interface{}, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if contractType == "" {
		return nil, errContractTypeIsRequired
	}
	if !slices.Contains(validFuturesIntervals, interval) {
		return nil, kline.ErrUnsupportedInterval
	}
	params := url.Values{}
	params.Set("pair", pair.String())
	params.Set("contractType", contractType)
	params.Set("interval", interval)
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", timeString(startTime))
		params.Set("endTime", timeString(endTime))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
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
	var resp [][12]types.Number
	err := b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/continuousKlines", params), rateBudget, &resp)
	if err != nil {
		return nil, err
	}
	result := make([]FuturesCandleStick, len(resp))
	for x := range resp {
		result[x] = FuturesCandleStick{
			OpenTime:                time.UnixMilli(resp[x][0].Int64()),
			Open:                    resp[x][1].Float64(),
			High:                    resp[x][2].Float64(),
			Low:                     resp[x][3].Float64(),
			Close:                   resp[x][4].Float64(),
			Volume:                  resp[x][5].Float64(),
			CloseTime:               time.UnixMilli(resp[x][6].Int64()),
			QuoteAssetVolume:        resp[x][7].Float64(),
			NumberOfTrades:          resp[x][8].Int64(),
			TakerBuyVolume:          resp[x][9].Float64(),
			TakerBuyBaseAssetVolume: resp[x][10].Float64(),
		}
	}
	return result, nil
}

// GetIndexOrCandlesticPriceKlineData kline/candlestick bars for the index price of a pair.
// Klines are uniquely identified by their open time.
func (b *Binance) GetIndexOrCandlesticPriceKlineData(ctx context.Context, pair currency.Pair, interval string, startTime, endTime time.Time, limit int64) ([]FuturesCandleStick, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !slices.Contains(validFuturesIntervals, interval) {
		return nil, kline.ErrUnsupportedInterval
	}
	params := url.Values{}
	params.Set("pair", pair.String())
	params.Set("interval", interval)
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
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
	var resp [][12]types.Number
	err := b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/indexPriceKlines", params), rateBudget, &resp)
	if err != nil {
		return nil, err
	}
	result := make([]FuturesCandleStick, len(resp))
	for x := range resp {
		result[x] = FuturesCandleStick{
			OpenTime:  time.UnixMilli(resp[x][0].Int64()),
			Open:      resp[x][1].Float64(),
			High:      resp[x][2].Float64(),
			Low:       resp[x][3].Float64(),
			Close:     resp[x][4].Float64(),
			CloseTime: time.UnixMilli(resp[x][6].Int64()),
		}
	}
	return result, nil
}

// GetMarkPriceKlineCandlesticks kline/candlestick bars for the mark price of a symbol.
// Klines are uniquely identified by their open time.
func (b *Binance) GetMarkPriceKlineCandlesticks(ctx context.Context, symbol, interval string, startTime, endTime time.Time, limit int64) ([]FuturesCandleStick, error) {
	return b.getKlineCandlesticks(ctx, symbol, interval, "/fapi/v1/markPriceKlines", startTime, endTime, limit)
}

// GetPremiumIndexKlineCandlesticks premium index kline bars of a symbol.
// Klines are uniquely identified by their open time.
func (b *Binance) GetPremiumIndexKlineCandlesticks(ctx context.Context, symbol, interval string, startTime, endTime time.Time, limit int64) ([]FuturesCandleStick, error) {
	return b.getKlineCandlesticks(ctx, symbol, interval, "/fapi/v1/premiumIndexKlines", startTime, endTime, limit)
}

func (b *Binance) getKlineCandlesticks(ctx context.Context, symbol, interval, path string, startTime, endTime time.Time, limit int64) ([]FuturesCandleStick, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if !slices.Contains(validFuturesIntervals, interval) {
		return nil, kline.ErrUnsupportedInterval
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", interval)
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", timeString(startTime))
		params.Set("endTime", timeString(endTime))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp [][12]types.Number
	err := b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(path, params), uFuturesDefaultRate, &resp)
	if err != nil {
		return nil, err
	}
	result := make([]FuturesCandleStick, len(resp))
	for x := range resp {
		result[x] = FuturesCandleStick{
			OpenTime:  time.UnixMilli(resp[x][0].Int64()),
			Open:      resp[x][1].Float64(),
			High:      resp[x][2].Float64(),
			Low:       resp[x][3].Float64(),
			Close:     resp[x][4].Float64(),
			CloseTime: time.UnixMilli(resp[x][6].Int64()),
		}
	}
	return result, nil
}

// UGetMarkPrice gets mark price data for USDTMarginedFutures
func (b *Binance) UGetMarkPrice(ctx context.Context, symbol string) ([]UMarkPrice, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
		var tempResp UMarkPrice
		err := b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesMarkPrice, params), uFuturesDefaultRate, &tempResp)
		if err != nil {
			return nil, err
		}
		return []UMarkPrice{tempResp}, nil
	}
	var resp []UMarkPrice
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesMarkPrice, params), uFuturesDefaultRate, &resp)
}

// UGetFundingRateInfo returns extra details about funding rates
func (b *Binance) UGetFundingRateInfo(ctx context.Context) ([]FundingRateInfoResponse, error) {
	var resp []FundingRateInfoResponse
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, ufuturesFundingRateInfo, uFuturesDefaultRate, &resp)
}

// UGetFundingHistory gets funding history for USDTMarginedFutures
func (b *Binance) UGetFundingHistory(ctx context.Context, symbol string, limit int64, startTime, endTime time.Time) ([]FundingRateHistory, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []FundingRateHistory
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesFundingRateHistory, params), uFuturesDefaultRate, &resp)
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
		err = b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesTickerPriceStats, params), uFuturesDefaultRate, &tempResp)
		if err != nil {
			return nil, err
		}
		return []U24HrPriceChangeStats{tempResp}, err
	}
	var resp []U24HrPriceChangeStats
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesTickerPriceStats, params), uFuturesTickerPriceHistoryRate, &resp)
}

// USymbolPriceTickerV1 gets symbol price ticker for USDTMarginedFutures V1
func (b *Binance) USymbolPriceTickerV1(ctx context.Context, symbol currency.Pair) ([]USymbolPriceTicker, error) {
	return b.uSymbolPriceTicker(ctx, symbol, ufuturesSymbolPriceTicker)
}

// USymbolPriceTickerV2 gets symbol price ticker for USDTMarginedFutures V2
func (b *Binance) USymbolPriceTickerV2(ctx context.Context, symbol currency.Pair) ([]USymbolPriceTicker, error) {
	return b.uSymbolPriceTicker(ctx, symbol, "/fapi/v2/ticker/price")
}
func (b *Binance) uSymbolPriceTicker(ctx context.Context, symbol currency.Pair, path string) ([]USymbolPriceTicker, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		var tempResp USymbolPriceTicker
		err = b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(path, params), uFuturesDefaultRate, &tempResp)
		if err != nil {
			return nil, err
		}
		return []USymbolPriceTicker{tempResp}, err
	}
	var resp []USymbolPriceTicker
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(path, params), uFuturesOrderbookTickerAllRate, &resp)
}

// GetSymbolPriceTicker retrieves latest price for symbol or symbols
// func (b *Binance) GetSymbolPriceTicker(ctx context.Context, symbol string) ([]USymbolPriceTicker, )

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
		err = b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesSymbolOrderbook, params), uFuturesDefaultRate, &tempResp)
		if err != nil {
			return nil, err
		}
		return []USymbolOrderbookTicker{tempResp}, err
	}
	var resp []USymbolOrderbookTicker
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesTickerPriceStats, params), uFuturesOrderbookTickerAllRate, &resp)
}

// UOpenInterest gets open interest data for USDTMarginedFutures
func (b *Binance) UOpenInterest(ctx context.Context, symbol string) (*UOpenInterestData, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *UOpenInterestData
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesOpenInterest, params), uFuturesDefaultRate, &resp)
}

// GetQuarterlyContractSettlementPrice retrieves quarterly contract settlement price
func (b *Binance) GetQuarterlyContractSettlementPrice(ctx context.Context, pair currency.Pair) ([]SettlementPrice, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp []SettlementPrice
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, "/futures/data/delivery-price?pair="+pair.String(), uFuturesDefaultRate, &resp)
}

// UOpenInterestStats gets open interest stats for USDTMarginedFutures
func (b *Binance) UOpenInterestStats(ctx context.Context, symbol, period string, limit int64, startTime, endTime time.Time) ([]UOpenInterestStats, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if !slices.Contains(uValidPeriods, period) {
		return nil, errInvalidPeriodOrInterval
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("period", period)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []UOpenInterestStats
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesOpenInterestStats, params), uFuturesDefaultRate, &resp)
}

// UTopAcccountsLongShortRatio gets long/short ratio data for top trader accounts in ufutures
func (b *Binance) UTopAcccountsLongShortRatio(ctx context.Context, symbol, period string, limit int64, startTime, endTime time.Time) ([]ULongShortRatio, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if !slices.Contains(uValidPeriods, period) {
		return nil, errInvalidPeriodOrInterval
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("period", period)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []ULongShortRatio
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesTopAccountsRatio, params), uFuturesDefaultRate, &resp)
}

// UTopPostionsLongShortRatio gets long/short ratio data for top positions' in ufutures
func (b *Binance) UTopPostionsLongShortRatio(ctx context.Context, symbol, period string, limit int64, startTime, endTime time.Time) ([]ULongShortRatio, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if !slices.Contains(uValidPeriods, period) {
		return nil, errInvalidPeriodOrInterval
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("period", period)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []ULongShortRatio
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesTopPositionsRatio, params), uFuturesDefaultRate, &resp)
}

// UGlobalLongShortRatio gets the global long/short ratio data for USDTMarginedFutures
func (b *Binance) UGlobalLongShortRatio(ctx context.Context, symbol, period string, limit int64, startTime, endTime time.Time) ([]ULongShortRatio, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if !slices.Contains(uValidPeriods, period) {
		return nil, errInvalidPeriodOrInterval
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("period", period)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []ULongShortRatio
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesLongShortRatio, params), uFuturesDefaultRate, &resp)
}

// UTakerBuySellVol gets takers' buy/sell ratio for USDTMarginedFutures
func (b *Binance) UTakerBuySellVol(ctx context.Context, symbol, period string, limit int64, startTime, endTime time.Time) ([]UTakerVolumeData, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if !slices.Contains(uValidPeriods, period) {
		return nil, errInvalidPeriodOrInterval
	}
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	params.Set("symbol", symbol)
	params.Set("period", period)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []UTakerVolumeData
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesBuySellVolume, params), uFuturesDefaultRate, &resp)
}

// GetBasis retrieves the basis price difference between the index price and futures trading of pairs.
func (b *Binance) GetBasis(ctx context.Context, pair currency.Pair, contractType, period string, startTime, endTime time.Time, limit int64) (interface{}, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if contractType == "" {
		return nil, errContractTypeIsRequired
	}
	if !slices.Contains(uValidPeriods, period) {
		return nil, errInvalidPeriodOrInterval
	}
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	params.Set("pair", pair.String())
	params.Set("contractType", contractType)
	params.Set("period", period)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []BasisInfo
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/futures/data/basis", params), uFuturesDefaultRate, &resp)
}

// GetHistoricalBLVTNAVCandlesticks retrieves BLVT (Binance Leveraged Tokens) NAV (Net Asset Value) leveraged tokens candlestic data.
func (b *Binance) GetHistoricalBLVTNAVCandlesticks(ctx context.Context, symbol, interval string, startTime, endTime time.Time, limit int64) ([]FuturesCandleStick, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if !slices.Contains(validFuturesIntervals, interval) {
		return nil, kline.ErrUnsupportedInterval
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", interval)
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
	var resp [][12]types.Number
	err := b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/lvtKlines", params), uFuturesDefaultRate, &resp)
	if err != nil {
		return nil, err
	}
	result := make([]FuturesCandleStick, len(resp))
	for x := range resp {
		result[x] = FuturesCandleStick{
			OpenTime:  time.UnixMilli(resp[x][0].Int64()),
			Open:      resp[x][1].Float64(),
			High:      resp[x][2].Float64(),
			Low:       resp[x][3].Float64(),
			Close:     resp[x][4].Float64(),
			CloseTime: time.UnixMilli(resp[x][6].Int64()),
		}
	}
	return result, nil
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
		err = b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesCompositeIndexInfo, params), uFuturesDefaultRate, &tempResp)
		if err != nil {
			return nil, err
		}
		return []UCompositeIndexInfoData{tempResp}, err
	}
	var resp []UCompositeIndexInfoData
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesCompositeIndexInfo, params), uFuturesDefaultRate, &resp)
}

// GetMultiAssetModeAssetIndex asset index for multi-asset mode
func (b *Binance) GetMultiAssetModeAssetIndex(ctx context.Context, symbol string) ([]AssetIndex, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp AssetIndexResponse
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/assetIndex", params), uFuturesDefaultRate, &resp)
}

// GetIndexPriceConstituents retrieves an index price constituents for a specified symbol
func (b *Binance) GetIndexPriceConstituents(ctx context.Context, symbol string) (*IndexPriceConstituent, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *IndexPriceConstituent
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, "/fapi/v1/constituents?symbol="+symbol, uFuturesDefaultRate, &resp)
}

// UFuturesNewOrder sends a new order for USDTMarginedFutures
func (b *Binance) UFuturesNewOrder(ctx context.Context, data *UFuturesNewOrderRequest) (*UOrderData, error) {
	if *data == (UFuturesNewOrderRequest{}) {
		return nil, errNilArgument
	}
	var err error
	data.Symbol, err = b.FormatExchangeCurrency(data.Symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	if data.PositionSide != "" && !slices.Contains(validPositionSide, data.PositionSide) {
		return nil, errInvalidPositionSide
	}
	if data.WorkingType != "" && !slices.Contains(validWorkingType, data.WorkingType) {
		return nil, errInvalidWorkingType
	}
	if data.NewOrderRespType != "" && !slices.Contains(validNewOrderRespType, data.NewOrderRespType) {
		return nil, errInvalidNewOrderResponseType
	}
	var resp *UOrderData
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesOrder, nil, uFuturesOrdersDefaultRate, data, &resp)
}

func (b *Binance) validatePlaceOrder(arg *USDTOrderUpdateParams) error {
	if arg.OrderID == 0 && arg.OrigClientOrderID == "" {
		return order.ErrOrderIDNotSet
	}
	if arg.Symbol.IsEmpty() {
		return currency.ErrSymbolStringEmpty
	}
	if arg.Side == "" {
		return order.ErrSideIsInvalid
	}
	if arg.Amount <= 0 {
		return order.ErrAmountBelowMin
	}
	if arg.Price <= 0 {
		return order.ErrPriceBelowMin
	}
	return nil
}

// UModifyOrder order modify function, currently only LIMIT order modification is supported, modified orders will be reordered in the match queue
// Weight: 1 on 10s order rate limit(X-MBX-ORDER-COUNT-10S); 1 on 1min order rate limit(X-MBX-ORDER-COUNT-1M); 1 on IP rate limit(x-mbx-used-weight-1m);
// PriceMatch: only available for LIMIT/STOP/TAKE_PROFIT order; can be set to OPPONENT/ OPPONENT_5/ OPPONENT_10/ OPPONENT_20: /QUEUE/ QUEUE_5/ QUEUE_10/ QUEUE_20; Can't be passed together with price
func (b *Binance) UModifyOrder(ctx context.Context, arg *USDTOrderUpdateParams) (*UOrderData, error) {
	if *arg == (USDTOrderUpdateParams{}) {
		return nil, errNilArgument
	}
	err := b.validatePlaceOrder(arg)
	if err != nil {
		return nil, err
	}
	arg.Symbol, err = b.FormatExchangeCurrency(arg.Symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	var resp *UOrderData
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPut, ufuturesOrder, nil, uFuturesDefaultRate, arg, &resp)
}

// UPlaceBatchOrders places batch orders
func (b *Binance) UPlaceBatchOrders(ctx context.Context, data []PlaceBatchOrderData) ([]UOrderData, error) {
	if len(data) == 0 {
		return nil, errNilArgument
	}
	var err error
	for x := range data {
		data[x].Symbol, err = b.FormatExchangeCurrency(data[x].Symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		if data[x].PositionSide != "" {
			if !slices.Contains(validPositionSide, data[x].PositionSide) {
				return nil, errInvalidPositionSide
			}
		}
		if data[x].WorkingType != "" {
			if !slices.Contains(validWorkingType, data[x].WorkingType) {
				return nil, errInvalidWorkingType
			}
		}
		if data[x].NewOrderRespType != "" {
			if !slices.Contains(validNewOrderRespType, data[x].NewOrderRespType) {
				return nil, errInvalidNewOrderResponseType
			}
		}
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("batchOrders", string(jsonData))
	var resp []UOrderData
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesBatchOrder, params, uFuturesBatchOrdersRate, nil, &resp)
}

// UModifyMultipleOrders applies a modification to a batch of usdt margined futures orders.
func (b *Binance) UModifyMultipleOrders(ctx context.Context, args []USDTOrderUpdateParams) ([]UOrderData, error) {
	if len(args) == 0 {
		return nil, errNilArgument
	}
	for a := range args {
		err := b.validatePlaceOrder(&args[a])
		if err != nil {
			return nil, err
		}
		args[a].Symbol, err = b.FormatExchangeCurrency(args[a].Symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
	}
	jsonData, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("batchOrders", string(jsonData))
	var resp []UOrderData
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPut, "/fapi/v1/batchOrders", params, uFuturesDefaultRate, nil, &resp)
}

// GetUSDTOrderModifyHistory retrieves order modification history
func (b *Binance) GetUSDTOrderModifyHistory(ctx context.Context, symbol currency.Pair, origClientOrderID string, orderID, limit int64, startTime, endTime time.Time) ([]USDTAmendInfo, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	if orderID <= 0 && origClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if orderID <= 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
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
	var resp []USDTAmendInfo
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/orderAmendment", params, uFuturesDefaultRate, nil, &resp)
}

// UGetOrderData gets order data for USDTMarginedFutures
func (b *Binance) UGetOrderData(ctx context.Context, symbol, orderID, cliOrderID string) (*UOrderData, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if cliOrderID != "" {
		params.Set("origClientOrderId", cliOrderID)
	}
	var resp *UOrderData
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesOrder, params, uFuturesOrdersDefaultRate, nil, &resp)
}

// UCancelOrder cancel an order for USDTMarginedFutures
func (b *Binance) UCancelOrder(ctx context.Context, symbol, orderID, cliOrderID string) (*UOrderData, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if cliOrderID != "" {
		params.Set("origClientOrderId", cliOrderID)
	}
	var resp *UOrderData
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodDelete, ufuturesOrder, params, uFuturesOrdersDefaultRate, nil, &resp)
}

// UCancelAllOpenOrders cancels all open orders for a symbol ufutures
func (b *Binance) UCancelAllOpenOrders(ctx context.Context, symbol string) (*GenericAuthResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *GenericAuthResponse
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodDelete, ufuturesCancelAllOrders, params, uFuturesOrdersDefaultRate, nil, &resp)
}

// UCancelBatchOrders cancel batch order for USDTMarginedFutures
func (b *Binance) UCancelBatchOrders(ctx context.Context, symbol string, orderIDList, origCliOrdIDList []string) ([]UOrderData, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if len(orderIDList) != 0 {
		jsonOrders, err := json.Marshal(orderIDList)
		if err != nil {
			return nil, err
		}
		params.Set("orderIdList", string(jsonOrders))
	}
	if len(origCliOrdIDList) != 0 {
		jsonCliOrders, err := json.Marshal(origCliOrdIDList)
		if err != nil {
			return nil, err
		}
		params.Set("origClientOrderIdList", string(jsonCliOrders))
	}
	var resp []UOrderData
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodDelete, ufuturesBatchOrder, params, uFuturesOrdersDefaultRate, nil, &resp)
}

// UAutoCancelAllOpenOrders auto cancels all ufutures open orders for a symbol after the set countdown time
func (b *Binance) UAutoCancelAllOpenOrders(ctx context.Context, symbol string, countdownTime int64) (*AutoCancelAllOrdersData, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("countdownTime", strconv.FormatInt(countdownTime, 10))
	var resp *AutoCancelAllOrdersData
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesCountdownCancel, params, uFuturesCountdownCancelRate, nil, &resp)
}

// UFetchOpenOrder sends a request to fetch open order data for USDTMarginedFutures
func (b *Binance) UFetchOpenOrder(ctx context.Context, symbol currency.Pair, orderID, origClientOrderID string) (*UOrderData, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	var resp *UOrderData
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesOpenOrder, params, uFuturesOrdersDefaultRate, nil, &resp)
}

// UAllAccountOpenOrders gets all account's orders for USDTMarginedFutures
func (b *Binance) UAllAccountOpenOrders(ctx context.Context, symbol currency.Pair) ([]UOrderData, error) {
	params := url.Values{}
	rateLimit := uFuturesGetAllOpenOrdersRate
	if !symbol.IsEmpty() {
		rateLimit = uFuturesOrdersDefaultRate
		p, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", p)
	} else {
		// extend the receive window when all currencies to prevent "recvwindow" error
		params.Set("recvWindow", "10000")
	}
	var resp []UOrderData
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesAllOpenOrders, params, rateLimit, nil, &resp)
}

// UAllAccountOrders gets all account's orders for USDTMarginedFutures
func (b *Binance) UAllAccountOrders(ctx context.Context, symbol string, orderID, limit int64, startTime, endTime time.Time) ([]UFuturesOrderData, error) {
	params := url.Values{}
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []UFuturesOrderData
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesAllOrders, params, uFuturesGetAllOrdersRate, nil, &resp)
}

// UAccountBalanceV2 gets V2 account balance data
func (b *Binance) UAccountBalanceV2(ctx context.Context) ([]UAccountBalanceV2Data, error) {
	var resp []UAccountBalanceV2Data
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesAccountBalance, nil, uFuturesOrdersDefaultRate, nil, &resp)
}

// UAccountInformationV2 gets V2 account balance data
func (b *Binance) UAccountInformationV2(ctx context.Context) (*UAccountInformationV2Data, error) {
	var resp *UAccountInformationV2Data
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesAccountInfo, nil, uFuturesAccountInformationRate, nil, &resp)
}

// UChangeInitialLeverageRequest sends a request to change account's initial leverage
func (b *Binance) UChangeInitialLeverageRequest(ctx context.Context, symbol string, leverage float64) (*UChangeInitialLeverage, error) {
	if leverage < 1 || leverage > 125 {
		return nil, order.ErrSubmitLeverageNotSupported
	}
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	var resp *UChangeInitialLeverage
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesChangeInitialLeverage, params, uFuturesDefaultRate, nil, &resp)
}

// UChangeInitialMarginType sends a request to change account's initial margin type
func (b *Binance) UChangeInitialMarginType(ctx context.Context, symbol currency.Pair, marginType string) error {
	symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	if !slices.Contains(validMarginType, marginType) {
		return margin.ErrInvalidMarginType
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	params.Set("marginType", marginType)
	return b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesChangeMarginType, params, uFuturesDefaultRate, nil, &struct{}{})
}

// UModifyIsolatedPositionMarginReq sends a request to modify isolated margin for USDTMarginedFutures
func (b *Binance) UModifyIsolatedPositionMarginReq(ctx context.Context, symbol, positionSide, changeType string, amount float64) (*UModifyIsolatedPosMargin, error) {
	cType, ok := validMarginChange[changeType]
	if !ok {
		return nil, errMarginChangeTypeInvalid
	}
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	params.Set("type", strconv.FormatInt(cType, 10))
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	if positionSide != "" {
		params.Set("positionSide", positionSide)
	}
	var resp *UModifyIsolatedPosMargin
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesModifyMargin, params, uFuturesDefaultRate, nil, &resp)
}

// UPositionMarginChangeHistory gets margin change history for USDTMarginedFutures
func (b *Binance) UPositionMarginChangeHistory(ctx context.Context, symbol, changeType string, limit int64, startTime, endTime time.Time) ([]UPositionMarginChangeHistoryData, error) {
	cType, ok := validMarginChange[changeType]
	if !ok {
		return nil, errMarginChangeTypeInvalid
	}
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	params.Set("type", strconv.FormatInt(cType, 10))
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []UPositionMarginChangeHistoryData
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesMarginChangeHistory, params, uFuturesDefaultRate, nil, &resp)
}

// UPositionsInfoV2 gets positions' info for USDTMarginedFutures
func (b *Binance) UPositionsInfoV2(ctx context.Context, symbol currency.Pair) ([]UPositionInformationV2, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp []UPositionInformationV2
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesPositionInfo, params, uFuturesDefaultRate, nil, &resp)
}

// UGetCommissionRates returns the commission rates for USDTMarginedFutures
func (b *Binance) UGetCommissionRates(ctx context.Context, symbol string) (*UPositionInformationV2, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *UPositionInformationV2
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesCommissionRate, params, uFuturesDefaultRate, nil, &resp)
}

// GetUSDTUserRateLimits retrieves users rate limit information.
func (b *Binance) GetUSDTUserRateLimits(ctx context.Context) ([]RateLimitInfo, error) {
	var resp []RateLimitInfo
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/rateLimit/order", nil, uFuturesDefaultRate, nil, &resp)
}

// GetDownloadIDForFuturesTransactionHistory retrieves download ID for futures transaction history
func (b *Binance) GetDownloadIDForFuturesTransactionHistory(ctx context.Context, startTime, endTime time.Time) (*UTransactionDownloadID, error) {
	return b.uFuturesDownloadID(ctx, "/fapi/v1/income/asyn", startTime, endTime)
}

// UFuturesOrderHistoryDownloadID retrieves downloading id futures orders history
func (b *Binance) UFuturesOrderHistoryDownloadID(ctx context.Context, startTime, endTime time.Time) (*UTransactionDownloadID, error) {
	return b.uFuturesDownloadID(ctx, "/fapi/v1/order/asyn", startTime, endTime)
}

func (b *Binance) uFuturesDownloadID(ctx context.Context, path string, startTime, endTime time.Time) (*UTransactionDownloadID, error) {
	err := common.StartEndTimeCheck(startTime, endTime)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	var resp *UTransactionDownloadID
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, path, params, uFuturesDefaultRate, nil, &resp)
}

// GetFuturesTransactionHistoryDownloadLinkByID retrieves futures transaction history download link by id
func (b *Binance) GetFuturesTransactionHistoryDownloadLinkByID(ctx context.Context, downloadID string) (*UTransactionHistoryDownloadLink, error) {
	return b.uFuturesHistoryDownloadLinkByID(ctx, downloadID, "/fapi/v1/income/asyn/id")
}

// GetFuturesOrderHistoryDownloadLinkByID retrieves futures order history download link by id
func (b *Binance) GetFuturesOrderHistoryDownloadLinkByID(ctx context.Context, downloadID string) (*UTransactionHistoryDownloadLink, error) {
	return b.uFuturesHistoryDownloadLinkByID(ctx, downloadID, "/fapi/v1/order/asyn/id")
}

func (b *Binance) uFuturesHistoryDownloadLinkByID(ctx context.Context, downloadID, path string) (*UTransactionHistoryDownloadLink, error) {
	if downloadID == "" {
		return nil, errDownloadIDRequired
	}
	var resp *UTransactionHistoryDownloadLink
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, path, url.Values{"downloadID": {downloadID}}, uFuturesDefaultRate, nil, &resp)
}

// FuturesTradeHistoryDownloadID retrieves download ID for futures trade history
func (b *Binance) FuturesTradeHistoryDownloadID(ctx context.Context, startTime, endTime time.Time) (*UTransactionDownloadID, error) {
	return b.uFuturesDownloadID(ctx, "/fapi/v1/trade/asyn", startTime, endTime)
}

// FuturesTradeDownloadLinkByID retrieves futures trade download link by download id
func (b *Binance) FuturesTradeDownloadLinkByID(ctx context.Context, downloadID string) (*UTransactionHistoryDownloadLink, error) {
	return b.uFuturesHistoryDownloadLinkByID(ctx, downloadID, "/fapi/v1/trade/asyn/id")
}

// UAccountTradesHistory gets account's trade history data for USDTMarginedFutures
func (b *Binance) UAccountTradesHistory(ctx context.Context, symbol, fromID string, limit int64, startTime, endTime time.Time) ([]UAccountTradeHistory, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if fromID != "" {
		params.Set("fromID", fromID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []UAccountTradeHistory
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesAccountTradeList, params, uFuturesAccountInformationRate, nil, &resp)
}

// UAccountIncomeHistory gets account's income history data for USDTMarginedFutures
func (b *Binance) UAccountIncomeHistory(ctx context.Context, symbol, incomeType string, limit int64, startTime, endTime time.Time) ([]UAccountIncomeHistory, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if incomeType != "" {
		if !slices.Contains(validIncomeType, incomeType) {
			return nil, errIncomeTypeRequired
		}
		params.Set("incomeType", incomeType)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []UAccountIncomeHistory
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesIncomeHistory, params, uFuturesIncomeHistoryRate, nil, &resp)
}

// UGetNotionalAndLeverageBrackets gets account's notional and leverage brackets for USDTMarginedFutures
func (b *Binance) UGetNotionalAndLeverageBrackets(ctx context.Context, symbol currency.Pair) ([]UNotionalLeverageAndBrakcetsData, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp []UNotionalLeverageAndBrakcetsData
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesNotionalBracket, params, uFuturesDefaultRate, nil, &resp)
}

// UPositionsADLEstimate gets estimated ADL data for USDTMarginedFutures positions
func (b *Binance) UPositionsADLEstimate(ctx context.Context, symbol currency.Pair) (*UPositionADLEstimationData, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp *UPositionADLEstimationData
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesADLQuantile, params, uFuturesAccountInformationRate, nil, &resp)
}

// UAccountForcedOrders gets account's forced (liquidation) orders for USDTMarginedFutures
func (b *Binance) UAccountForcedOrders(ctx context.Context, symbol currency.Pair, autoCloseType string, limit int64, startTime, endTime time.Time) ([]UForceOrdersData, error) {
	params := url.Values{}
	rateLimit := uFuturesAllForceOrdersRate
	if !symbol.IsEmpty() {
		rateLimit = uFuturesCurrencyForceOrdersRate
		symbolValue, err := b.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	if autoCloseType != "" {
		if !slices.Contains(validAutoCloseTypes, autoCloseType) {
			return nil, errInvalidAutoCloseType
		}
		params.Set("autoCloseType", autoCloseType)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []UForceOrdersData
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesUsersForceOrders, params, rateLimit, nil, &resp)
}

// UFuturesTradingWuantitativeRulesIndicators retrieves rules that regulate general trading based on the quantitative indicators
func (b *Binance) UFuturesTradingWuantitativeRulesIndicators(ctx context.Context, symbol currency.Pair) (*TradingQuantitativeRulesIndicators, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	var resp *TradingQuantitativeRulesIndicators
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/apiTradingStatus", params, uFuturesDefaultRate, nil, &resp)
}

// GetPerpMarkets returns exchange information. Check binance_types for more information
func (b *Binance) GetPerpMarkets(ctx context.Context) (*PerpsExchangeInfo, error) {
	var resp *PerpsExchangeInfo
	return resp, b.SendHTTPRequest(ctx, exchange.RestUSDTMargined, "/fapi/v1/exchangeInfo", uFuturesDefaultRate, &resp)
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

// SetAssetsMode sets the current asset margin type, true for multi, false for single
func (b *Binance) SetAssetsMode(ctx context.Context, multiMargin bool) error {
	params := url.Values{
		"multiAssetsMargin": {strconv.FormatBool(multiMargin)},
	}
	return b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, uFuturesMultiAssetsMargin, params, uFuturesDefaultRate, nil, nil)
}

// GetAssetsMode returns the current asset margin type, true for multi, false for single
func (b *Binance) GetAssetsMode(ctx context.Context) (bool, error) {
	var result struct {
		MultiAssetsMargin bool `json:"multiAssetsMargin"`
	}
	return result.MultiAssetsMargin, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, uFuturesMultiAssetsMargin, nil, uFuturesDefaultRate, nil, &result)
}

// ------------------------------------------ Account/Trade Endpoints -----------------------------------------------------

// ChangePositionMode change user's position mode (Hedge Mode or One-way Mode ) on EVERY symbol
func (b *Binance) ChangePositionMode(ctx context.Context, dualPositionMode bool) error {
	params := url.Values{}
	if dualPositionMode {
		params.Set("dualPositionMode", "true")
	} else {
		params.Set("dualPositionMode", "false")
	}
	return b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/positionSide/dual", params, uFuturesDefaultRate, nil, &struct{}{})
}

// GetCurrentPositionMode retrieves the current position mode
func (b *Binance) GetCurrentPositionMode(ctx context.Context) (*PositionMode, error) {
	var resp *PositionMode
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/positionSide/dual", nil, uFuturesDefaultRate, nil, &resp)
}
