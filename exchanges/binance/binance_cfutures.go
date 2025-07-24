package binance

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// FuturesExchangeInfo stores CoinMarginedFutures, data
func (e *Exchange) FuturesExchangeInfo(ctx context.Context) (*CExchangeInfo, error) {
	var resp *CExchangeInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, "/dapi/v1/exchangeInfo", cFuturesDefaultRate, &resp)
}

// GetFuturesOrderbook gets orderbook data for CoinMarginedFutures,
func (e *Exchange) GetFuturesOrderbook(ctx context.Context, symbol currency.Pair, limit uint64) (*OrderBook, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
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
	params.Set("symbol", symbolValue)
	var resp *OrderBook
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/depth", params), rateBudget, &resp)
}

// GetFuturesPublicTrades gets recent public trades for CoinMarginedFutures,
func (e *Exchange) GetFuturesPublicTrades(ctx context.Context, symbol currency.Pair, limit uint64) ([]FuturesPublicTradesData, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []FuturesPublicTradesData
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/trades", params), cFuturesDefaultRate, &resp)
}

// GetFuturesHistoricalTrades gets historical public trades for CoinMarginedFutures,
func (e *Exchange) GetFuturesHistoricalTrades(ctx context.Context, symbol currency.Pair, fromID string, limit uint64) ([]UPublicTradesData, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if fromID != "" {
		params.Set("fromID", fromID)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []UPublicTradesData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/historicalTrades", params, cFuturesHistoricalTradesRate, nil, &resp)
}

// GetPastPublicTrades gets past public trades for CoinMarginedFutures,
func (e *Exchange) GetPastPublicTrades(ctx context.Context, symbol currency.Pair, limit, fromID uint64) ([]FuturesPublicTradesData, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if fromID != 0 {
		params.Set("fromID", strconv.FormatUint(fromID, 10))
	}
	var resp []FuturesPublicTradesData
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/trades", params), cFuturesDefaultRate, &resp)
}

// GetFuturesAggregatedTradesList gets aggregated trades list for CoinMarginedFutures,
func (e *Exchange) GetFuturesAggregatedTradesList(ctx context.Context, symbol currency.Pair, fromID, limit uint64, startTime, endTime time.Time) ([]AggregatedTrade, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if fromID != 0 {
		params.Set("fromID", strconv.FormatUint(fromID, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []AggregatedTrade
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/aggTrades", params), cFuturesHistoricalTradesRate, &resp)
}

// GetIndexAndMarkPrice gets index and mark prices  for CoinMarginedFutures,
func (e *Exchange) GetIndexAndMarkPrice(ctx context.Context, symbol, pair string) ([]IndexMarkPrice, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	var resp []IndexMarkPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/premiumIndex", params), cFuturesIndexMarkPriceRate, &resp)
}

// GetFundingRateInfo retrieves funding rate info for symbols that had FundingRateCap/ FundingRateFloor / fundingIntervalHours adjustment
func (e *Exchange) GetFundingRateInfo(ctx context.Context) ([]FundingRateInfoResponse, error) {
	var resp []FundingRateInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, "/dapi/v1/fundingInfo", uFuturesDefaultRate, &resp)
}

// GetFuturesKlineData gets futures kline data for CoinMarginedFutures,
func (e *Exchange) GetFuturesKlineData(ctx context.Context, symbol currency.Pair, interval string, limit uint64, startTime, endTime time.Time) ([]CFuturesCandleStick, error) {
	if !slices.Contains(validFuturesIntervals, interval) {
		return nil, kline.ErrInvalidInterval
	}
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	params.Set("interval", interval)
	var data []CFuturesCandleStick
	rateBudget := getKlineRateBudget(limit)
	return data, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/klines", params), rateBudget, &data)
}

// GetContinuousKlineData gets continuous kline data
func (e *Exchange) GetContinuousKlineData(ctx context.Context, pair, contractType, interval string, limit uint64, startTime, endTime time.Time) ([]CFuturesCandleStick, error) {
	if pair == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if !slices.Contains(validContractType, contractType) {
		return nil, errContractTypeIsRequired
	}
	if !slices.Contains(validFuturesIntervals, interval) {
		return nil, kline.ErrInvalidInterval
	}
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	params.Set("pair", pair)
	params.Set("contractType", contractType)
	params.Set("interval", interval)
	rateBudget := getKlineRateBudget(limit)
	var data []CFuturesCandleStick
	return data, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/continuousKlines", params), rateBudget, &data)
}

// GetIndexPriceKlines gets continuous kline data
func (e *Exchange) GetIndexPriceKlines(ctx context.Context, pair, interval string, limit uint64, startTime, endTime time.Time) ([]CFuturesCandleStick, error) {
	if !slices.Contains(validFuturesIntervals, interval) {
		return nil, kline.ErrInvalidInterval
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	params.Set("interval", interval)
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	params.Set("pair", pair)
	rateBudget := getKlineRateBudget(limit)
	var data []CFuturesCandleStick
	return data, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/indexPriceKlines", params), rateBudget, &data)
}

// GetMarkPriceKline gets mark price kline data
func (e *Exchange) GetMarkPriceKline(ctx context.Context, symbol currency.Pair, interval string, limit uint64, startTime, endTime time.Time) ([]CFuturesCandleStick, error) {
	return e.getKline(ctx, symbol, interval, "/dapi/v1/markPriceKlines", limit, startTime, endTime)
}

// GetPremiumIndexKlineData premium index kline bars of a symbol.
// Klines are uniquely identified by their open time.
func (e *Exchange) GetPremiumIndexKlineData(ctx context.Context, symbol currency.Pair, interval string, limit uint64, startTime, endTime time.Time) ([]CFuturesCandleStick, error) {
	return e.getKline(ctx, symbol, interval, "/dapi/v1/premiumIndexKlines", limit, startTime, endTime)
}

func (e *Exchange) getKline(ctx context.Context, symbol currency.Pair, interval, path string, limit uint64, startTime, endTime time.Time) ([]CFuturesCandleStick, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(validFuturesIntervals, interval) {
		return nil, kline.ErrInvalidInterval
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	params.Set("interval", interval)
	if !startTime.IsZero() && !endTime.IsZero() {
		err = common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	params.Set("symbol", symbolValue)
	rateBudget := getKlineRateBudget(limit)
	var data []CFuturesCandleStick
	return data, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues(path, params), rateBudget, &data)
}

func getKlineRateBudget(limit uint64) request.EndpointLimit {
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
func (e *Exchange) GetFuturesSwapTickerChangeStats(ctx context.Context, symbol currency.Pair, pair string) ([]PriceChangeStats, error) {
	params := url.Values{}
	rateLimit := cFuturesTickerPriceHistoryRate
	if !symbol.IsEmpty() {
		rateLimit = cFuturesDefaultRate
		symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	var resp []PriceChangeStats
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/ticker/24hr", params), rateLimit, &resp)
}

// FuturesGetFundingHistory gets funding history for CoinMarginedFutures,
func (e *Exchange) FuturesGetFundingHistory(ctx context.Context, symbol currency.Pair, limit int, startTime, endTime time.Time) ([]FundingRateHistory, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []FundingRateHistory
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/fundingRate", params), cFuturesDefaultRate, &resp)
}

// GetFuturesSymbolPriceTicker gets price ticker for symbol
func (e *Exchange) GetFuturesSymbolPriceTicker(ctx context.Context, symbol currency.Pair, pair string) ([]SymbolPriceTicker, error) {
	params := url.Values{}
	rateLimit := cFuturesOrderbookTickerAllRate
	if !symbol.IsEmpty() {
		rateLimit = cFuturesDefaultRate
		symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	var resp []SymbolPriceTicker
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/ticker/price", params), rateLimit, &resp)
}

// GetFuturesOrderbookTicker gets orderbook ticker for symbol
func (e *Exchange) GetFuturesOrderbookTicker(ctx context.Context, symbol currency.Pair, pair string) ([]SymbolOrderBookTicker, error) {
	params := url.Values{}
	rateLimit := cFuturesOrderbookTickerAllRate
	if !symbol.IsEmpty() {
		rateLimit = cFuturesDefaultRate
		symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	var resp []SymbolOrderBookTicker
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/ticker/bookTicker", params), rateLimit, &resp)
}

// GetCFuturesIndexPriceConstituents retrieved index price constituents detail
func (e *Exchange) GetCFuturesIndexPriceConstituents(ctx context.Context, symbol currency.Pair) (*CFuturesIndexPriceConstituents, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	var resp *CFuturesIndexPriceConstituents
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/constituents", params), cFuturesDefaultRate, &resp)
}

// OpenInterest gets open interest data for a symbol
func (e *Exchange) OpenInterest(ctx context.Context, symbol currency.Pair) (*OpenInterestData, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	var resp *OpenInterestData
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, "/dapi/v1/openInterest?symbol="+symbolValue, cFuturesDefaultRate, &resp)
}

// CFuturesQuarterlyContractSettlementPrice retrieves coin margined futures quarterly contract settlement price
func (e *Exchange) CFuturesQuarterlyContractSettlementPrice(ctx context.Context, pair currency.Pair) ([]SettlementPrice, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp []SettlementPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, "/futures/data/delivery-price?pair="+pair.String(), cFuturesDefaultRate, &resp)
}

// GetOpenInterestStats gets open interest stats for a symbol
func (e *Exchange) GetOpenInterestStats(ctx context.Context, pair, contractType, period string, limit uint64, startTime, endTime time.Time) ([]OpenInterestStats, error) {
	if !slices.Contains(validContractType, contractType) {
		return nil, fmt.Errorf("%w: invalid interval %s", errContractTypeIsRequired, contractType)
	}
	if !slices.Contains(validFuturesIntervals, period) {
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
	params.Set("contractType", contractType)
	params.Set("period", period)
	if pair != "" {
		params.Set("pair", pair)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []OpenInterestStats
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/futures/data/openInterestHist", params), cFuturesDefaultRate, &resp)
}

// GetTraderFuturesAccountRatio gets a traders futures account long/short ratio
func (e *Exchange) GetTraderFuturesAccountRatio(ctx context.Context, pair currency.Pair, period string, limit uint64, startTime, endTime time.Time) ([]TopTraderAccountRatio, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !slices.Contains(validFuturesIntervals, period) {
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
	params.Set("period", period)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []TopTraderAccountRatio
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/futures/data/topLongShortAccountRatio", params), cFuturesDefaultRate, &resp)
}

// GetTraderFuturesPositionsRatio gets a traders futures positions' long/short ratio
func (e *Exchange) GetTraderFuturesPositionsRatio(ctx context.Context, pair currency.Pair, period string, limit uint64, startTime, endTime time.Time) ([]TopTraderPositionRatio, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if !slices.Contains(validFuturesIntervals, period) {
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
	params.Set("period", period)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []TopTraderPositionRatio
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/futures/data/topLongShortPositionRatio", params), cFuturesDefaultRate, &resp)
}

// GetMarketRatio gets global long/short ratio
func (e *Exchange) GetMarketRatio(ctx context.Context, pair currency.Pair, period string, limit uint64, startTime, endTime time.Time) ([]TopTraderPositionRatio, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !slices.Contains(validFuturesIntervals, period) {
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
	params.Set("period", period)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []TopTraderPositionRatio
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/futures/data/globalLongShortAccountRatio", params), cFuturesDefaultRate, &resp)
}

// GetFuturesTakerVolume gets futures taker buy/sell volumes
func (e *Exchange) GetFuturesTakerVolume(ctx context.Context, pair currency.Pair, contractType, period string, limit uint64, startTime, endTime time.Time) ([]TakerBuySellVolume, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !slices.Contains(validContractType, contractType) {
		return nil, errContractTypeIsRequired
	}
	if !slices.Contains(validFuturesIntervals, period) {
		return nil, kline.ErrInvalidInterval
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
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	params.Set("period", period)
	var resp []TakerBuySellVolume
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/futures/data/takerBuySellVol", params), cFuturesDefaultRate, &resp)
}

// GetFuturesBasisData gets futures basis data
func (e *Exchange) GetFuturesBasisData(ctx context.Context, pair currency.Pair, contractType, period string, limit uint64, startTime, endTime time.Time) ([]FuturesBasisData, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !slices.Contains(validContractType, contractType) {
		return nil, errContractTypeIsRequired
	}
	if !slices.Contains(validFuturesIntervals, period) {
		return nil, errInvalidPeriodOrInterval
	}
	params := url.Values{}
	params.Set("pair", pair.String())
	params.Set("contractType", contractType)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	params.Set("period", period)
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []FuturesBasisData
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/futures/data/basis", params), cFuturesDefaultRate, &resp)
}

// FuturesNewOrder sends a new futures order to the exchange
func (e *Exchange) FuturesNewOrder(ctx context.Context, x *FuturesNewOrderRequest) (*FuturesOrderPlaceData, error) {
	var err error
	x.Symbol, err = e.FormatExchangeCurrency(x.Symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	if x.PositionSide != "" && !slices.Contains(validPositionSide, x.PositionSide) {
		return nil, fmt.Errorf("%w %s", errInvalidPositionSide, x.PositionSide)
	}
	if x.WorkingType != "" && !slices.Contains(validWorkingType, x.WorkingType) {
		return nil, errInvalidWorkingType
	}
	if x.NewOrderRespType != "" && !slices.Contains(validNewOrderRespType, x.NewOrderRespType) {
		return nil, errInvalidNewOrderResponseType
	}
	var resp *FuturesOrderPlaceData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, "/dapi/v1/order", nil, cFuturesOrdersDefaultRate, x, &resp)
}

// FuturesBatchOrder sends a batch order request
func (e *Exchange) FuturesBatchOrder(ctx context.Context, data []PlaceBatchOrderData) ([]FuturesOrderPlaceData, error) {
	if len(data) == 0 {
		return nil, common.ErrEmptyParams
	}
	var err error
	for x := range data {
		data[x].Symbol, err = e.FormatExchangeCurrency(data[x].Symbol, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		if data[x].PositionSide != "" && !slices.Contains(validPositionSide, data[x].PositionSide) {
			return nil, fmt.Errorf("%w %s", errInvalidPositionSide, data[x].PositionSide)
		}
		if data[x].WorkingType != "" && !slices.Contains(validWorkingType, data[x].WorkingType) {
			return nil, errInvalidWorkingType
		}
		if data[x].NewOrderRespType != "" && !slices.Contains(validNewOrderRespType, data[x].NewOrderRespType) {
			return nil, errInvalidNewOrderResponseType
		}
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("batchOrders", string(jsonData))
	var resp []FuturesOrderPlaceData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, "/dapi/v1/batchOrders", params, cFuturesBatchOrdersRate, nil, &resp)
}

// FuturesBatchCancelOrders sends a batch request to cancel orders
func (e *Exchange) FuturesBatchCancelOrders(ctx context.Context, symbol currency.Pair, orderList, origClientOrderIDList []string) ([]BatchCancelOrderData, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if len(orderList) != 0 {
		jsonOrderList, err := json.Marshal(orderList)
		if err != nil {
			return nil, err
		}
		params.Set("orderIdList", string(jsonOrderList))
	}
	if len(origClientOrderIDList) != 0 {
		jsonCliOrdIDList, err := json.Marshal(origClientOrderIDList)
		if err != nil {
			return nil, err
		}
		params.Set("origClientOrderIdList", string(jsonCliOrdIDList))
	}
	var resp []BatchCancelOrderData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodDelete, "/dapi/v1/batchOrders", params, cFuturesOrdersDefaultRate, nil, &resp)
}

// FuturesGetOrderData gets futures order data
func (e *Exchange) FuturesGetOrderData(ctx context.Context, symbol currency.Pair, orderID, origClientOrderID string) (*FuturesOrderGetData, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	var resp *FuturesOrderGetData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/order", params, cFuturesOrdersDefaultRate, nil, &resp)
}

// FuturesCancelOrder cancels a futures order
func (e *Exchange) FuturesCancelOrder(ctx context.Context, symbol currency.Pair, orderID, origClientOrderID string) (*FuturesOrderGetData, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	var resp *FuturesOrderGetData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodDelete, "/dapi/v1/order", params, cFuturesOrdersDefaultRate, nil, &resp)
}

// FuturesCancelAllOpenOrders cancels a futures order
func (e *Exchange) FuturesCancelAllOpenOrders(ctx context.Context, symbol currency.Pair) (*GenericAuthResponse, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	var resp *GenericAuthResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodDelete, "/dapi/v1/allOpenOrders", params, cFuturesOrdersDefaultRate, nil, &resp)
}

// AutoCancelAllOpenOrders cancels all open futures orders
// countdownTime 1000 = 1s, example - to cancel all orders after 30s (countdownTime: 30000)
func (e *Exchange) AutoCancelAllOpenOrders(ctx context.Context, symbol currency.Pair, countdownTime int64) (*AutoCancelAllOrdersData, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	params.Set("countdownTime", strconv.FormatInt(countdownTime, 10))
	var resp *AutoCancelAllOrdersData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, "/dapi/v1/countdownCancelAll", params, cFuturesCancelAllOrdersRate, nil, &resp)
}

// FuturesOpenOrderData gets open order data for CoinMarginedFutures
func (e *Exchange) FuturesOpenOrderData(ctx context.Context, symbol currency.Pair, orderID, origClientOrderID string) (*FuturesOrderGetData, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	var resp *FuturesOrderGetData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/openOrder", params, cFuturesOrdersDefaultRate, nil, &resp)
}

// GetFuturesAllOpenOrders gets all open orders data for CoinMarginedFutures,
func (e *Exchange) GetFuturesAllOpenOrders(ctx context.Context, symbol currency.Pair, pair string) ([]FuturesOrderData, error) {
	var p string
	var err error
	rateLimit := cFuturesGetAllOpenOrdersRate
	params := url.Values{}
	if !symbol.IsEmpty() {
		rateLimit = cFuturesOrdersDefaultRate
		p, err = e.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", p)
	} else {
		// extend the receive window when all currencies to prevent "recvwindow" error
		params.Set("recvWindow", "10000")
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	var resp []FuturesOrderData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/openOrders", params, rateLimit, nil, &resp)
}

// GetAllFuturesOrders gets all orders active cancelled or filled
func (e *Exchange) GetAllFuturesOrders(ctx context.Context, symbol, pair currency.Pair, startTime, endTime time.Time, orderID, limit uint64) ([]FuturesOrderData, error) {
	params := url.Values{}
	rateLimit := cFuturesPairOrdersRate
	if !symbol.IsEmpty() {
		rateLimit = cFuturesSymbolOrdersRate
		symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	if !pair.IsEmpty() {
		params.Set("pair", pair.String())
	}
	if orderID != 0 {
		params.Set("orderID", strconv.FormatUint(orderID, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []FuturesOrderData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/allOrders", params, rateLimit, nil, &resp)
}

// GetFuturesAccountBalance gets account balance data for CoinMarginedFutures, account
func (e *Exchange) GetFuturesAccountBalance(ctx context.Context) ([]FuturesAccountBalanceData, error) {
	var resp []FuturesAccountBalanceData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/balance", nil, cFuturesDefaultRate, nil, &resp)
}

// GetFuturesAccountInfo gets account info data for CoinMarginedFutures, account
func (e *Exchange) GetFuturesAccountInfo(ctx context.Context) (*FuturesAccountInformation, error) {
	var resp *FuturesAccountInformation
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/account", nil, cFuturesAccountInformationRate, nil, &resp)
}

// FuturesChangeInitialLeverage changes initial leverage for the account
func (e *Exchange) FuturesChangeInitialLeverage(ctx context.Context, symbol currency.Pair, leverage float64) (*FuturesLeverageData, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	if leverage < 1 || leverage > 125 {
		return nil, fmt.Errorf("%w: leverage value should range between 1 and 125", order.ErrSubmitLeverageNotSupported)
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	var resp *FuturesLeverageData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, "/dapi/v1/leverage", params, cFuturesDefaultRate, nil, &resp)
}

// FuturesChangeMarginType changes margin type
func (e *Exchange) FuturesChangeMarginType(ctx context.Context, symbol currency.Pair, marginType string) (*GenericAuthResponse, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(validMarginType, marginType) {
		return nil, margin.ErrInvalidMarginType
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	params.Set("marginType", marginType)
	var resp *GenericAuthResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, "/dapi/v1/marginType", params, cFuturesDefaultRate, nil, &resp)
}

// ModifyIsolatedPositionMargin changes margin for an isolated position
func (e *Exchange) ModifyIsolatedPositionMargin(ctx context.Context, symbol currency.Pair, positionSide, changeType string, amount float64) (*FuturesMarginUpdatedResponse, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if changeType != "" {
		cType, ok := validMarginChange[changeType]
		if !ok {
			return nil, fmt.Errorf("%w: possible position margin are 1: add position margin 2: reduce position margin", errMarginChangeTypeInvalid)
		}
		params.Set("type", strconv.FormatInt(cType, 10))
	}
	if positionSide != "" {
		if !slices.Contains(validPositionSide, positionSide) {
			return nil, errInvalidPositionSide
		}
		params.Set("positionSide", positionSide)
	}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	var resp *FuturesMarginUpdatedResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, "/dapi/v1/positionMargin", params, cFuturesDefaultRate, nil, &resp)
}

// FuturesMarginChangeHistory gets past margin changes for positions
func (e *Exchange) FuturesMarginChangeHistory(ctx context.Context, symbol currency.Pair, changeType string, startTime, endTime time.Time, limit int64) ([]GetPositionMarginChangeHistoryData, error) {
	symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	cType, ok := validMarginChange[changeType]
	if !ok {
		return nil, errMarginChangeTypeInvalid
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	params.Set("type", strconv.FormatInt(cType, 10))
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []GetPositionMarginChangeHistoryData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/positionMargin/history", params, cFuturesDefaultRate, nil, &resp)
}

// FuturesPositionsInfo gets futures positions info
// "pair" for coinmarginedfutures in GCT terms is the pair base
// eg ADAUSD_PERP the "pair" parameter is ADAUSD
func (e *Exchange) FuturesPositionsInfo(ctx context.Context, marginAsset, pair string) ([]FuturesPositionInformation, error) {
	params := url.Values{}
	if marginAsset != "" {
		params.Set("marginAsset", marginAsset)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	var resp []FuturesPositionInformation
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/positionRisk", params, cFuturesDefaultRate, nil, &resp)
}

// FuturesTradeHistory gets trade history for CoinMarginedFutures, account
func (e *Exchange) FuturesTradeHistory(ctx context.Context, symbol currency.Pair, pair string, startTime, endTime time.Time, limit, fromID int64) ([]FuturesAccountTradeList, error) {
	params := url.Values{}
	rateLimit := cFuturesPairOrdersRate
	if !symbol.IsEmpty() {
		rateLimit = cFuturesSymbolOrdersRate
		symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
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
	var resp []FuturesAccountTradeList
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/userTrades", params, rateLimit, nil, &resp)
}

// FuturesIncomeHistory gets income history for CoinMarginedFutures,
func (e *Exchange) FuturesIncomeHistory(ctx context.Context, symbol currency.Pair, incomeType string, startTime, endTime time.Time, limit int64) ([]FuturesIncomeHistoryData, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	if incomeType != "" {
		if !slices.Contains(validIncomeType, incomeType) {
			return nil, fmt.Errorf("invalid incomeType: %v", incomeType)
		}
		params.Set("incomeType", incomeType)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []FuturesIncomeHistoryData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/income", params, cFuturesIncomeHistoryRate, nil, &resp)
}

// FuturesNotionalBracket gets futures notional bracket
func (e *Exchange) FuturesNotionalBracket(ctx context.Context, pair string) ([]NotionalBracketData, error) {
	params := url.Values{}
	if pair != "" {
		params.Set("pair", pair)
	}
	var resp []NotionalBracketData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/leverageBracket", params, cFuturesDefaultRate, nil, &resp)
}

// FuturesForceOrders gets futures forced orders
func (e *Exchange) FuturesForceOrders(ctx context.Context, symbol currency.Pair, autoCloseType string, startTime, endTime time.Time) ([]ForcedOrdersData, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
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
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp []ForcedOrdersData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/forceOrders", params, cFuturesDefaultRate, nil, &resp)
}

// FuturesPositionsADLEstimate estimates ADL on positions
func (e *Exchange) FuturesPositionsADLEstimate(ctx context.Context, symbol currency.Pair) ([]ADLEstimateData, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp []ADLEstimateData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/adlQuantile", params, cFuturesAccountInformationRate, nil, &resp)
}

// FetchCoinMarginExchangeLimits fetches coin margined order execution limits
func (e *Exchange) FetchCoinMarginExchangeLimits(ctx context.Context) ([]order.MinMaxLevel, error) {
	coinFutures, err := e.FuturesExchangeInfo(ctx)
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
