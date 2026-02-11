package binance

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	// Unauth
	ufuturesMarkPrice          = "/fapi/v1/premiumIndex"
	ufuturesTickerPriceStats   = "/fapi/v1/ticker/24hr"
	ufuturesCompositeIndexInfo = "/fapi/v1/indexInfo"

	// Auth
	ufuturesOrder             = "/fapi/v1/order"
	ufuturesBatchOrder        = "/fapi/v1/batchOrders"
	uFuturesMultiAssetsMargin = "/fapi/v1/multiAssetsMargin"
)

// UServerTime gets the server time
func (e *Exchange) UServerTime(ctx context.Context) (time.Time, error) {
	var data struct {
		ServerTime types.Time `json:"serverTime"`
	}
	err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, "/fapi/v1/time", uFuturesDefaultRate, &data)
	if err != nil {
		return time.Time{}, err
	}
	return data.ServerTime.Time(), nil
}

// UExchangeInfo stores usdt margined futures data
func (e *Exchange) UExchangeInfo(ctx context.Context) (*UFuturesExchangeInfo, error) {
	var resp *UFuturesExchangeInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, "/fapi/v1/exchangeInfo", uFuturesDefaultRate, &resp)
}

// UFuturesOrderbook gets orderbook data for usdt margined futures
func (e *Exchange) UFuturesOrderbook(ctx context.Context, symbol currency.Pair, limit int64) (*OrderBook, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	strLimit := strconv.FormatInt(limit, 10)
	if strLimit != "" {
		if !slices.Contains(uValidOBLimits, strLimit) {
			return nil, errLimitNumberRequired
		}
		params.Set("limit", strLimit)
	}
	rateBudget := uFuturesOrderbook1000Rate
	switch {
	case limit == 5, limit == 10, limit == 20, limit == 50:
		rateBudget = uFuturesOrderbook50Rate
	case limit >= 100 && limit < 500:
		rateBudget = uFuturesOrderbook100Rate
	case limit == 0, limit >= 500 && limit < 1000:
		rateBudget = uFuturesOrderbook500Rate
	}
	params.Set("symbol", symbol.String())

	var resp *OrderBook
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/depth", params), rateBudget, &resp)
}

// URecentTrades gets recent trades for usdt margined futures
func (e *Exchange) URecentTrades(ctx context.Context, symbol currency.Pair, fromID string, limit int64) ([]*UPublicTradesData, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if fromID != "" {
		params.Set("fromID", fromID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []*UPublicTradesData
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/trades", params), uFuturesDefaultRate, &resp)
}

// UFuturesHistoricalTrades gets historical public trades for USDTMarginedFutures
func (e *Exchange) UFuturesHistoricalTrades(ctx context.Context, symbol currency.Pair, fromID string, limit int64) ([]*UPublicTradesData, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if fromID != "" {
		params.Set("fromID", fromID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []*UPublicTradesData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/historicalTrades", params, uFuturesHistoricalTradesRate, nil, &resp)
}

// UCompressedTrades gets compressed public trades for usdt margined futures
func (e *Exchange) UCompressedTrades(ctx context.Context, symbol currency.Pair, fromID string, limit int64, startTime, endTime time.Time) ([]*UCompressedTradeData, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if fromID != "" {
		params.Set("fromID", fromID)
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
	var resp []*UCompressedTradeData
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/aggTrades", params), uFuturesHistoricalTradesRate, &resp)
}

// UKlineData gets kline data for usdt margined futures
func (e *Exchange) UKlineData(ctx context.Context, symbol currency.Pair, interval string, limit uint64, startTime, endTime time.Time) ([]*UFuturesCandleStick, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !slices.Contains(validFuturesIntervals, interval) {
		return nil, kline.ErrUnsupportedInterval
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	params.Set("interval", interval)
	if !startTime.IsZero() {
		params.Set("startTime", timeString(startTime))
	}
	if !endTime.IsZero() {
		params.Set("endTime", timeString(endTime))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
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
	var data []*UFuturesCandleStick
	return data, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/klines", params), rateBudget, &data)
}

// GetUFuturesContinuousKlineData kline/candlestick bars for a specific contract type.
// Klines are uniquely identified by their open time.
func (e *Exchange) GetUFuturesContinuousKlineData(ctx context.Context, pair currency.Pair, contractType, interval string, startTime, endTime time.Time, limit int64) (interface{}, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if contractType == "" {
		return nil, errContractTypeIsRequired
	}
	if !slices.Contains(validFuturesIntervals, interval) {
		return nil, kline.ErrUnsupportedInterval
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("pair", pair.String())
	params.Set("contractType", contractType)
	params.Set("interval", interval)
	if !startTime.IsZero() {
		params.Set("startTime", timeString(startTime))
	}
	if !endTime.IsZero() {
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
	var resp []*UFuturesCandleStick
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/continuousKlines", params), rateBudget, &resp)
}

// GetIndexOrCandlesticPriceKlineData kline/candlestick bars for the index price of a pair.
// Klines are uniquely identified by their open time.
func (e *Exchange) GetIndexOrCandlesticPriceKlineData(ctx context.Context, pair currency.Pair, interval string, startTime, endTime time.Time, limit int64) ([]*UFuturesCandleStick, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !slices.Contains(validFuturesIntervals, interval) {
		return nil, kline.ErrUnsupportedInterval
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("pair", pair.String())
	params.Set("interval", interval)
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
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
	var resp []*UFuturesCandleStick
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/indexPriceKlines", params), rateBudget, &resp)
}

// GetMarkPriceKlineCandlesticks kline/candlestick bars for the mark price of a symbol.
// Klines are uniquely identified by their open time.
func (e *Exchange) GetMarkPriceKlineCandlesticks(ctx context.Context, symbol currency.Pair, interval string, startTime, endTime time.Time, limit int64) ([]*UFuturesCandleStick, error) {
	return e.getKlineCandlesticks(ctx, symbol, interval, "/fapi/v1/markPriceKlines", startTime, endTime, limit)
}

// GetPremiumIndexKlineCandlesticks premium index kline bars of a symbol.
// Klines are uniquely identified by their open time.
func (e *Exchange) GetPremiumIndexKlineCandlesticks(ctx context.Context, symbol currency.Pair, interval string, startTime, endTime time.Time, limit int64) ([]*UFuturesCandleStick, error) {
	return e.getKlineCandlesticks(ctx, symbol, interval, "/fapi/v1/premiumIndexKlines", startTime, endTime, limit)
}

func (e *Exchange) getKlineCandlesticks(ctx context.Context, symbol currency.Pair, interval, path string, startTime, endTime time.Time, limit int64) ([]*UFuturesCandleStick, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !slices.Contains(validFuturesIntervals, interval) {
		return nil, kline.ErrUnsupportedInterval
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	params.Set("interval", interval)
	if !startTime.IsZero() {
		params.Set("startTime", timeString(startTime))
	}
	if !endTime.IsZero() {
		params.Set("endTime", timeString(endTime))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []*UFuturesCandleStick
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(path, params), uFuturesDefaultRate, &resp)
}

// UGetMarkPrice gets mark price data for USDTMarginedFutures
func (e *Exchange) UGetMarkPrice(ctx context.Context, symbol currency.Pair) ([]*UMarkPrice, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
		var tempResp UMarkPrice
		if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesMarkPrice, params), uFuturesDefaultRate, &tempResp); err != nil {
			return nil, err
		}
		return []*UMarkPrice{&tempResp}, nil
	}
	var resp []*UMarkPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesMarkPrice, params), uFuturesDefaultRate, &resp)
}

// UGetFundingRateInfo returns extra details about funding rates
func (e *Exchange) UGetFundingRateInfo(ctx context.Context) ([]*FundingRateInfoResponse, error) {
	var resp []*FundingRateInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, "/fapi/v1/fundingInfo", uFuturesDefaultRate, &resp)
}

// UGetFundingHistory gets funding history for USDTMarginedFutures
func (e *Exchange) UGetFundingHistory(ctx context.Context, symbol currency.Pair, limit int64, startTime, endTime time.Time) ([]*FundingRateHistory, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []*FundingRateHistory
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/fundingRate", params), uFuturesDefaultRate, &resp)
}

// U24HTickerPriceChangeStats gets 24hr ticker price change stats for USDTMarginedFutures
func (e *Exchange) U24HTickerPriceChangeStats(ctx context.Context, symbol currency.Pair) ([]*U24HrPriceChangeStats, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		var tempResp U24HrPriceChangeStats
		if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesTickerPriceStats, params), uFuturesDefaultRate, &tempResp); err != nil {
			return nil, err
		}
		return []*U24HrPriceChangeStats{&tempResp}, err
	}
	var resp []*U24HrPriceChangeStats
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesTickerPriceStats, params), uFuturesTickerPriceHistoryRate, &resp)
}

// USymbolPriceTickerV1 gets symbol price ticker for USDTMarginedFutures V1
func (e *Exchange) USymbolPriceTickerV1(ctx context.Context, symbol currency.Pair) ([]*USymbolPriceTicker, error) {
	return e.uSymbolPriceTicker(ctx, symbol, "/fapi/v1/ticker/price")
}

// USymbolPriceTickerV2 gets symbol price ticker for USDTMarginedFutures V2
func (e *Exchange) USymbolPriceTickerV2(ctx context.Context, symbol currency.Pair) ([]*USymbolPriceTicker, error) {
	return e.uSymbolPriceTicker(ctx, symbol, "/fapi/v2/ticker/price")
}

func (e *Exchange) uSymbolPriceTicker(ctx context.Context, symbol currency.Pair, path string) ([]*USymbolPriceTicker, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		var tempResp USymbolPriceTicker
		if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(path, params), uFuturesDefaultRate, &tempResp); err != nil {
			return nil, err
		}
		return []*USymbolPriceTicker{&tempResp}, err
	}
	var resp []*USymbolPriceTicker
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(path, params), uFuturesOrderbookTickerAllRate, &resp)
}

// USymbolOrderbookTicker gets symbol orderbook ticker
func (e *Exchange) USymbolOrderbookTicker(ctx context.Context, symbol currency.Pair) ([]*USymbolOrderbookTicker, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		var tempResp USymbolOrderbookTicker
		if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/ticker/bookTicker", params), uFuturesDefaultRate, &tempResp); err != nil {
			return nil, err
		}
		return []*USymbolOrderbookTicker{&tempResp}, err
	}
	var resp []*USymbolOrderbookTicker
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesTickerPriceStats, params), uFuturesOrderbookTickerAllRate, &resp)
}

// UOpenInterest gets open interest data for USDTMarginedFutures
func (e *Exchange) UOpenInterest(ctx context.Context, symbol currency.Pair) (*UOpenInterestData, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	var resp *UOpenInterestData
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/openInterest", params), uFuturesDefaultRate, &resp)
}

// GetQuarterlyContractSettlementPrice retrieves quarterly contract settlement price
func (e *Exchange) GetQuarterlyContractSettlementPrice(ctx context.Context, pair currency.Pair) ([]*SettlementPrice, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp []*SettlementPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, "/futures/data/delivery-price?pair="+pair.String(), uFuturesDefaultRate, &resp)
}

// UOpenInterestStats gets open interest stats for USDTMarginedFutures
func (e *Exchange) UOpenInterestStats(ctx context.Context, symbol currency.Pair, period string, limit int64, startTime, endTime time.Time) ([]*UOpenInterestStats, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !slices.Contains(uValidPeriods, period) {
		return nil, errInvalidPeriodOrInterval
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	params.Set("period", period)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []*UOpenInterestStats
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/futures/data/openInterestHist", params), uFuturesDefaultRate, &resp)
}

// UTopAcccountsLongShortRatio gets long/short ratio data for top trader accounts in ufutures
func (e *Exchange) UTopAcccountsLongShortRatio(ctx context.Context, symbol currency.Pair, period string, limit int64, startTime, endTime time.Time) ([]*ULongShortRatio, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !slices.Contains(uValidPeriods, period) {
		return nil, errInvalidPeriodOrInterval
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	params.Set("period", period)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []*ULongShortRatio
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/futures/data/topLongShortAccountRatio", params), uFuturesDefaultRate, &resp)
}

// UTopPostionsLongShortRatio gets long/short ratio data for top positions' in ufutures
func (e *Exchange) UTopPostionsLongShortRatio(ctx context.Context, symbol currency.Pair, period string, limit int64, startTime, endTime time.Time) ([]*ULongShortRatio, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !slices.Contains(uValidPeriods, period) {
		return nil, errInvalidPeriodOrInterval
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	params.Set("period", period)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []*ULongShortRatio
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/futures/data/topLongShortPositionRatio", params), uFuturesDefaultRate, &resp)
}

// UGlobalLongShortRatio gets the global long/short ratio data for USDTMarginedFutures
func (e *Exchange) UGlobalLongShortRatio(ctx context.Context, symbol currency.Pair, period string, limit int64, startTime, endTime time.Time) ([]*ULongShortRatio, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !slices.Contains(uValidPeriods, period) {
		return nil, errInvalidPeriodOrInterval
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	params.Set("period", period)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	var resp []*ULongShortRatio
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/futures/data/globalLongShortAccountRatio", params), uFuturesDefaultRate, &resp)
}

// UTakerBuySellVol gets takers' buy/sell ratio for USDTMarginedFutures
func (e *Exchange) UTakerBuySellVol(ctx context.Context, symbol currency.Pair, period string, limit int64, startTime, endTime time.Time) ([]*UTakerVolumeData, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !slices.Contains(uValidPeriods, period) {
		return nil, errInvalidPeriodOrInterval
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	params.Set("symbol", symbol.String())
	params.Set("period", period)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []*UTakerVolumeData
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/futures/data/takerlongshortRatio", params), uFuturesDefaultRate, &resp)
}

// GetBasis retrieves the basis price difference between the index price and futures trading of pairs.
func (e *Exchange) GetBasis(ctx context.Context, pair currency.Pair, contractType, period string, startTime, endTime time.Time, limit int64) (interface{}, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if contractType == "" {
		return nil, errContractTypeIsRequired
	}
	if !slices.Contains(uValidPeriods, period) {
		return nil, errInvalidPeriodOrInterval
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	params.Set("pair", pair.String())
	params.Set("contractType", contractType)
	params.Set("period", period)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []*BasisInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/futures/data/basis", params), uFuturesDefaultRate, &resp)
}

// GetHistoricalBLVTNAVCandlesticks retrieves BLVT (Exchange Leveraged Tokens) NAV (Net Asset Value) leveraged tokens candlestic data.
func (e *Exchange) GetHistoricalBLVTNAVCandlesticks(ctx context.Context, symbol currency.Pair, interval string, startTime, endTime time.Time, limit int64) ([]*UFuturesCandleStick, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !slices.Contains(validFuturesIntervals, interval) {
		return nil, kline.ErrUnsupportedInterval
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	params.Set("interval", interval)
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []*UFuturesCandleStick
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/lvtKlines", params), uFuturesDefaultRate, &resp)
}

// UCompositeIndexInfo stores composite indexs' info for usdt margined futures
func (e *Exchange) UCompositeIndexInfo(ctx context.Context, symbol currency.Pair) ([]*UCompositeIndexInfoData, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		var tempResp UCompositeIndexInfoData
		if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesCompositeIndexInfo, params), uFuturesDefaultRate, &tempResp); err != nil {
			return nil, err
		}
		return []*UCompositeIndexInfoData{&tempResp}, err
	}
	var resp []*UCompositeIndexInfoData
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(ufuturesCompositeIndexInfo, params), uFuturesDefaultRate, &resp)
}

// GetMultiAssetModeAssetIndex asset index for multi-asset mode
func (e *Exchange) GetMultiAssetModeAssetIndex(ctx context.Context, symbol currency.Pair) ([]*AssetIndex, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	var resp AssetIndexResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/assetIndex", params), uFuturesDefaultRate, &resp)
}

// GetIndexPriceConstituents retrieves an index price constituents for a specified symbol
func (e *Exchange) GetIndexPriceConstituents(ctx context.Context, symbol string) (*IndexPriceConstituent, error) {
	if symbol == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp *IndexPriceConstituent
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, "/fapi/v1/constituents?symbol="+symbol, uFuturesDefaultRate, &resp)
}

// UFuturesNewOrder sends a new order for USDTMarginedFutures
func (e *Exchange) UFuturesNewOrder(ctx context.Context, data *UFuturesNewOrderRequest) (*UOrderData, error) {
	if err := common.NilGuard(data); err != nil {
		return nil, err
	}
	var err error
	data.Symbol, err = e.FormatExchangeCurrency(data.Symbol, asset.USDTMarginedFutures)
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
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesOrder, nil, uFuturesOrdersDefaultRate, data, &resp)
}

func (e *Exchange) validatePlaceOrder(arg *USDTOrderUpdateParams) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.OrderID == 0 && arg.OrigClientOrderID == "" {
		return order.ErrOrderIDNotSet
	}
	if arg.Symbol.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return order.ErrSideIsInvalid
	}
	if arg.Amount <= 0 {
		return limits.ErrAmountBelowMin
	}
	if arg.Price <= 0 {
		return limits.ErrPriceBelowMin
	}
	return nil
}

// UModifyOrder order modify function, currently only LIMIT order modification is supported, modified orders will be reordered in the match queue
// Weight: 1 on 10s order rate limit(X-MBX-ORDER-COUNT-10S); 1 on 1min order rate limit(X-MBX-ORDER-COUNT-1M); 1 on IP rate limit(x-mbx-used-weight-1m);
// PriceMatch: only available for LIMIT/STOP/TAKE_PROFIT order; can be set to OPPONENT/ OPPONENT_5/ OPPONENT_10/ OPPONENT_20: /QUEUE/ QUEUE_5/ QUEUE_10/ QUEUE_20; Can't be passed together with price
func (e *Exchange) UModifyOrder(ctx context.Context, arg *USDTOrderUpdateParams) (*UOrderData, error) {
	err := e.validatePlaceOrder(arg)
	if err != nil {
		return nil, err
	}
	arg.Symbol, err = e.FormatExchangeCurrency(arg.Symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	var resp *UOrderData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPut, ufuturesOrder, nil, uFuturesDefaultRate, arg, &resp)
}

// UPlaceBatchOrders places batch orders
func (e *Exchange) UPlaceBatchOrders(ctx context.Context, data []PlaceBatchOrderData) ([]*UOrderData, error) {
	if len(data) == 0 {
		return nil, common.ErrEmptyParams
	}
	var err error
	for x := range data {
		data[x].Symbol, err = e.FormatExchangeCurrency(data[x].Symbol, asset.USDTMarginedFutures)
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
	var resp []*UOrderData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesBatchOrder, params, uFuturesBatchOrdersRate, nil, &resp)
}

// UModifyMultipleOrders applies a modification to a batch of usdt margined futures orders.
func (e *Exchange) UModifyMultipleOrders(ctx context.Context, args []USDTOrderUpdateParams) ([]*UOrderData, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	for a := range args {
		err := e.validatePlaceOrder(&args[a])
		if err != nil {
			return nil, err
		}
		args[a].Symbol, err = e.FormatExchangeCurrency(args[a].Symbol, asset.USDTMarginedFutures)
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
	var resp []*UOrderData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPut, "/fapi/v1/batchOrders", params, uFuturesDefaultRate, nil, &resp)
}

// GetUSDTOrderModifyHistory retrieves order modification history
func (e *Exchange) GetUSDTOrderModifyHistory(ctx context.Context, symbol currency.Pair, origClientOrderID string, orderID, limit int64, startTime, endTime time.Time) ([]*USDTAmendInfo, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if orderID <= 0 && origClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if orderID <= 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
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
	var resp []*USDTAmendInfo
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/orderAmendment", params, uFuturesDefaultRate, nil, &resp)
}

// UGetOrderData gets order data for USDTMarginedFutures
func (e *Exchange) UGetOrderData(ctx context.Context, symbol currency.Pair, orderID, cliOrderID string) (*UOrderData, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if cliOrderID != "" {
		params.Set("origClientOrderId", cliOrderID)
	}
	var resp *UOrderData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesOrder, params, uFuturesOrdersDefaultRate, nil, &resp)
}

// UCancelOrder cancel an order for USDTMarginedFutures
func (e *Exchange) UCancelOrder(ctx context.Context, symbol currency.Pair, orderID, cliOrderID string) (*UOrderData, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if cliOrderID != "" {
		params.Set("origClientOrderId", cliOrderID)
	}
	var resp *UOrderData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodDelete, ufuturesOrder, params, uFuturesOrdersDefaultRate, nil, &resp)
}

// UCancelAllOpenOrders cancels all open orders for a symbol ufutures
func (e *Exchange) UCancelAllOpenOrders(ctx context.Context, symbol currency.Pair) (*GenericAuthResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	var resp *GenericAuthResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodDelete, "/fapi/v1/allOpenOrders", params, uFuturesOrdersDefaultRate, nil, &resp)
}

// UCancelBatchOrders cancel batch order for USDTMarginedFutures
func (e *Exchange) UCancelBatchOrders(ctx context.Context, symbol currency.Pair, orderIDList, origCliOrdIDList []string) ([]*UOrderData, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
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
	var resp []*UOrderData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodDelete, ufuturesBatchOrder, params, uFuturesOrdersDefaultRate, nil, &resp)
}

// UAutoCancelAllOpenOrders auto cancels all ufutures open orders for a symbol after the set countdown time
func (e *Exchange) UAutoCancelAllOpenOrders(ctx context.Context, symbol currency.Pair, countdownTime int64) (*AutoCancelAllOrdersData, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	params.Set("countdownTime", strconv.FormatInt(countdownTime, 10))
	var resp *AutoCancelAllOrdersData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/countdownCancelAll", params, uFuturesCountdownCancelRate, nil, &resp)
}

// UFetchOpenOrder sends a request to fetch open order data for USDTMarginedFutures
func (e *Exchange) UFetchOpenOrder(ctx context.Context, symbol currency.Pair, orderID, origClientOrderID string) (*UOrderData, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
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
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/openOrder", params, uFuturesOrdersDefaultRate, nil, &resp)
}

// UAllAccountOpenOrders gets all account's orders for USDTMarginedFutures
func (e *Exchange) UAllAccountOpenOrders(ctx context.Context, symbol currency.Pair) ([]*UOrderData, error) {
	params := url.Values{}
	rateLimit := uFuturesGetAllOpenOrdersRate
	if !symbol.IsEmpty() {
		rateLimit = uFuturesOrdersDefaultRate
		p, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", p)
	} else {
		// extend the receive window when all currencies to prevent "recvwindow" error
		params.Set("recvWindow", "10000")
	}
	var resp []*UOrderData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/openOrders", params, rateLimit, nil, &resp)
}

// UAllAccountOrders gets all account's orders for USDTMarginedFutures
func (e *Exchange) UAllAccountOrders(ctx context.Context, symbol currency.Pair, orderID, limit int64, startTime, endTime time.Time) ([]*UFuturesOrderData, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
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
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	var resp []*UFuturesOrderData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/allOrders", params, uFuturesGetAllOrdersRate, nil, &resp)
}

// UAccountBalanceV2 gets V2 account balance data
func (e *Exchange) UAccountBalanceV2(ctx context.Context) ([]*UAccountBalanceV2Data, error) {
	var resp []*UAccountBalanceV2Data
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v2/balance", nil, uFuturesOrdersDefaultRate, nil, &resp)
}

// UAccountInformationV2 gets V2 account balance data
func (e *Exchange) UAccountInformationV2(ctx context.Context) (*UAccountInformationV2Data, error) {
	var resp *UAccountInformationV2Data
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v2/account", nil, uFuturesAccountInformationRate, nil, &resp)
}

// UChangeInitialLeverageRequest sends a request to change account's initial leverage
func (e *Exchange) UChangeInitialLeverageRequest(ctx context.Context, symbol currency.Pair, leverage float64) (*UChangeInitialLeverage, error) {
	if leverage < 1 || leverage > 125 {
		return nil, order.ErrSubmitLeverageNotSupported
	}
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	var resp *UChangeInitialLeverage
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/leverage", params, uFuturesDefaultRate, nil, &resp)
}

// UChangeInitialMarginType sends a request to change account's initial margin type
func (e *Exchange) UChangeInitialMarginType(ctx context.Context, symbol currency.Pair, marginType string) error {
	symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	if !slices.Contains(validMarginType, marginType) {
		return margin.ErrInvalidMarginType
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	params.Set("marginType", marginType)
	return e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/marginType", params, uFuturesDefaultRate, nil, &struct{}{})
}

// UModifyIsolatedPositionMarginReq sends a request to modify isolated margin for USDTMarginedFutures
func (e *Exchange) UModifyIsolatedPositionMarginReq(ctx context.Context, symbol currency.Pair, positionSide, changeType string, amount float64) (*UModifyIsolatedPosMargin, error) {
	cType, ok := validMarginChange[changeType]
	if !ok {
		return nil, errMarginChangeTypeInvalid
	}
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	params.Set("type", strconv.FormatInt(cType, 10))
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	if positionSide != "" {
		params.Set("positionSide", positionSide)
	}
	var resp *UModifyIsolatedPosMargin
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/positionMargin", params, uFuturesDefaultRate, nil, &resp)
}

// UPositionMarginChangeHistory gets margin change history for USDTMarginedFutures
func (e *Exchange) UPositionMarginChangeHistory(ctx context.Context, symbol currency.Pair, changeType string, limit int64, startTime, endTime time.Time) ([]*UPositionMarginChangeHistoryData, error) {
	cType, ok := validMarginChange[changeType]
	if !ok {
		return nil, errMarginChangeTypeInvalid
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	params.Set("type", strconv.FormatInt(cType, 10))
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []*UPositionMarginChangeHistoryData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/positionMargin/history", params, uFuturesDefaultRate, nil, &resp)
}

// UPositionsInfoV2 gets positions' info for USDTMarginedFutures
func (e *Exchange) UPositionsInfoV2(ctx context.Context, symbol currency.Pair) ([]*UPositionInformationV2, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp []*UPositionInformationV2
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v2/positionRisk", params, uFuturesDefaultRate, nil, &resp)
}

// UGetCommissionRates returns the commission rates for USDTMarginedFutures
func (e *Exchange) UGetCommissionRates(ctx context.Context, symbol currency.Pair) (*UPositionInformationV2, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	var resp *UPositionInformationV2
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/commissionRate", params, uFuturesDefaultRate, nil, &resp)
}

// GetUSDTUserRateLimits retrieves users rate limit information.
func (e *Exchange) GetUSDTUserRateLimits(ctx context.Context) ([]*RateLimitInfo, error) {
	var resp []*RateLimitInfo
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/rateLimit/order", nil, uFuturesDefaultRate, nil, &resp)
}

// GetDownloadIDForFuturesTransactionHistory retrieves download ID for futures transaction history
func (e *Exchange) GetDownloadIDForFuturesTransactionHistory(ctx context.Context, startTime, endTime time.Time) (*UTransactionDownloadID, error) {
	return e.uFuturesDownloadID(ctx, "/fapi/v1/income/asyn", startTime, endTime)
}

// UFuturesOrderHistoryDownloadID retrieves downloading id futures orders history
func (e *Exchange) UFuturesOrderHistoryDownloadID(ctx context.Context, startTime, endTime time.Time) (*UTransactionDownloadID, error) {
	return e.uFuturesDownloadID(ctx, "/fapi/v1/order/asyn", startTime, endTime)
}

func (e *Exchange) uFuturesDownloadID(ctx context.Context, path string, startTime, endTime time.Time) (*UTransactionDownloadID, error) {
	if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	var resp *UTransactionDownloadID
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, path, params, uFuturesDefaultRate, nil, &resp)
}

// GetFuturesTransactionHistoryDownloadLinkByID retrieves futures transaction history download link by id
func (e *Exchange) GetFuturesTransactionHistoryDownloadLinkByID(ctx context.Context, downloadID string) (*UTransactionHistoryDownloadLink, error) {
	return e.uFuturesHistoryDownloadLinkByID(ctx, downloadID, "/fapi/v1/income/asyn/id")
}

// GetFuturesOrderHistoryDownloadLinkByID retrieves futures order history download link by id
func (e *Exchange) GetFuturesOrderHistoryDownloadLinkByID(ctx context.Context, downloadID string) (*UTransactionHistoryDownloadLink, error) {
	return e.uFuturesHistoryDownloadLinkByID(ctx, downloadID, "/fapi/v1/order/asyn/id")
}

func (e *Exchange) uFuturesHistoryDownloadLinkByID(ctx context.Context, downloadID, path string) (*UTransactionHistoryDownloadLink, error) {
	if downloadID == "" {
		return nil, errDownloadIDRequired
	}
	var resp *UTransactionHistoryDownloadLink
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, path, url.Values{"downloadID": {downloadID}}, uFuturesDefaultRate, nil, &resp)
}

// FuturesTradeHistoryDownloadID retrieves download ID for futures trade history
func (e *Exchange) FuturesTradeHistoryDownloadID(ctx context.Context, startTime, endTime time.Time) (*UTransactionDownloadID, error) {
	return e.uFuturesDownloadID(ctx, "/fapi/v1/trade/asyn", startTime, endTime)
}

// FuturesTradeDownloadLinkByID retrieves futures trade download link by download id
func (e *Exchange) FuturesTradeDownloadLinkByID(ctx context.Context, downloadID string) (*UTransactionHistoryDownloadLink, error) {
	return e.uFuturesHistoryDownloadLinkByID(ctx, downloadID, "/fapi/v1/trade/asyn/id")
}

// UAccountTradesHistory gets account's trade history data for USDTMarginedFutures
func (e *Exchange) UAccountTradesHistory(ctx context.Context, symbol currency.Pair, fromID string, limit int64, startTime, endTime time.Time) ([]*UAccountTradeHistory, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if fromID != "" {
		params.Set("fromID", fromID)
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
	var resp []*UAccountTradeHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/userTrades", params, uFuturesAccountInformationRate, nil, &resp)
}

// UAccountIncomeHistory gets account's income history data for USDTMarginedFutures
func (e *Exchange) UAccountIncomeHistory(ctx context.Context, symbol currency.Pair, incomeType string, limit int64, startTime, endTime time.Time) ([]*UAccountIncomeHistory, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
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
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []*UAccountIncomeHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/income", params, uFuturesIncomeHistoryRate, nil, &resp)
}

// UGetNotionalAndLeverageBrackets gets account's notional and leverage brackets for USDTMarginedFutures
func (e *Exchange) UGetNotionalAndLeverageBrackets(ctx context.Context, symbol currency.Pair) ([]*UNotionalLeverageAndBrakcetsData, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp []*UNotionalLeverageAndBrakcetsData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/leverageBracket", params, uFuturesDefaultRate, nil, &resp)
}

// UPositionsADLEstimate gets estimated ADL data for USDTMarginedFutures positions
func (e *Exchange) UPositionsADLEstimate(ctx context.Context, symbol currency.Pair) (*UPositionADLEstimationData, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp *UPositionADLEstimationData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/adlQuantile", params, uFuturesAccountInformationRate, nil, &resp)
}

// UAccountForcedOrders gets account's forced (liquidation) orders for USDTMarginedFutures
func (e *Exchange) UAccountForcedOrders(ctx context.Context, symbol currency.Pair, autoCloseType string, limit int64, startTime, endTime time.Time) ([]*UForceOrdersData, error) {
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	rateLimit := uFuturesAllForceOrdersRate
	if !symbol.IsEmpty() {
		rateLimit = uFuturesCurrencyForceOrdersRate
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
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
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []*UForceOrdersData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/forceOrders", params, rateLimit, nil, &resp)
}

// UFuturesTradingWuantitativeRulesIndicators retrieves rules that regulate general trading based on the quantitative indicators
func (e *Exchange) UFuturesTradingWuantitativeRulesIndicators(ctx context.Context, symbol currency.Pair) (*TradingQuantitativeRulesIndicators, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	var resp *TradingQuantitativeRulesIndicators
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/apiTradingStatus", params, uFuturesDefaultRate, nil, &resp)
}

// GetPerpMarkets returns exchange information. Check binance_types for more information
func (e *Exchange) GetPerpMarkets(ctx context.Context) (*PerpsExchangeInfo, error) {
	var resp *PerpsExchangeInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, "/fapi/v1/exchangeInfo", uFuturesDefaultRate, &resp)
}

// FetchUSDTMarginExchangeLimits fetches USDT margined order execution limits
func (e *Exchange) FetchUSDTMarginExchangeLimits(ctx context.Context) ([]limits.MinMaxLevel, error) {
	usdtFutures, err := e.UExchangeInfo(ctx)
	if err != nil {
		return nil, err
	}
	l := make([]limits.MinMaxLevel, 0, len(usdtFutures.Symbols))
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

		l = append(l, limits.MinMaxLevel{
			Key:                     key.NewExchangeAssetPair(e.Name, asset.USDTMarginedFutures, cp),
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
	return l, nil
}

// SetAssetsMode sets the current asset margin type, true for multi, false for single
func (e *Exchange) SetAssetsMode(ctx context.Context, multiMargin bool) error {
	params := url.Values{
		"multiAssetsMargin": {strconv.FormatBool(multiMargin)},
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, uFuturesMultiAssetsMargin, params, uFuturesDefaultRate, nil, nil)
}

// GetAssetsMode returns the current asset margin type, true for multi, false for single
func (e *Exchange) GetAssetsMode(ctx context.Context) (bool, error) {
	var result struct {
		MultiAssetsMargin bool `json:"multiAssetsMargin"`
	}
	return result.MultiAssetsMargin, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, uFuturesMultiAssetsMargin, nil, uFuturesDefaultRate, nil, &result)
}

// ------------------------------------------ Account/Trade Endpoints -----------------------------------------------------

// ChangePositionMode change user's position mode (Hedge Mode or One-way Mode ) on EVERY symbol
func (e *Exchange) ChangePositionMode(ctx context.Context, dualPositionMode bool) error {
	params := url.Values{}
	if dualPositionMode {
		params.Set("dualPositionMode", "true")
	} else {
		params.Set("dualPositionMode", "false")
	}
	return e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/positionSide/dual", params, uFuturesDefaultRate, nil, &struct{}{})
}

// GetCurrentPositionMode retrieves the current position mode
func (e *Exchange) GetCurrentPositionMode(ctx context.Context) (*PositionMode, error) {
	var resp *PositionMode
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/positionSide/dual", nil, uFuturesDefaultRate, nil, &resp)
}

// -----------------------------------  Copy Trading endpoints  -----------------------------------------

// GetFuturesLeadTraderStatus retrieves futures lead trader status
func (e *Exchange) GetFuturesLeadTraderStatus(ctx context.Context) (*LeadTraderStatus, error) {
	var resp *LeadTraderStatus
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/copyTrading/futures/userStatus", nil, request.UnAuth, nil, &resp)
}

// GetFuturesLeadTradingSymbolWhitelist retrieves a futures lead trading symbol whitelist
func (e *Exchange) GetFuturesLeadTradingSymbolWhitelist(ctx context.Context) ([]*LeadTradingSymbolItem, error) {
	var resp []*LeadTradingSymbolItem
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, "/sapi/v1/copyTrading/futures/leadSymbol", nil, request.UnAuth, nil, &resp)
}
