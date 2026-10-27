package binance

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errBatchModifyPriceMatch          = errors.New("modify multiple orders takes no price match")
	errBatchClosePosition             = errors.New("place multiple orders takes no close position orders")
	errSymbolOrPairRequired           = errors.New("symbol or pair is required")
	errSymbolAndPairMutuallyExclusive = errors.New("symbol and pair cannot be sent together")
	errOrderIDRequiresSymbol          = errors.New("order ID can only be sent with symbol")
	errFromIDMutuallyExclusive        = errors.New("from ID cannot be sent with pair, start time or end time")
)

// GetCoinMarginedFuturesAccountInfo returns the coin margined futures Account Information
func (e *Exchange) GetCoinMarginedFuturesAccountInfo(ctx context.Context) (*CoinMarginedFuturesAccountInformationResponse, error) {
	var resp *CoinMarginedFuturesAccountInformationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/account", nil, coinMarginedFuturesAccountInformationRate, &resp)
}

// GetCoinMarginedFuturesAccountBalance returns the coin margined Futures Account Balance
func (e *Exchange) GetCoinMarginedFuturesAccountBalance(ctx context.Context) ([]CoinMarginedFuturesAccountBalanceResponse, error) {
	var resp []CoinMarginedFuturesAccountBalanceResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/balance", nil, coinMarginedFuturesDefaultRate, &resp)
}

// CoinMarginedFuturesIncomeHistory returns the coin margined futures Income History
func (e *Exchange) CoinMarginedFuturesIncomeHistory(ctx context.Context, req *CoinMarginedFuturesIncomeHistoryRequest) ([]CoinMarginedFuturesIncomeHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if !req.Symbol.IsEmpty() {
		symbol, err := e.FormatSymbol(req.Symbol, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbol)
	}
	if req.IncomeType != "" {
		params.Set("incomeType", req.IncomeType)
	}
	if req.Page != 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []CoinMarginedFuturesIncomeHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/income", params, coinMarginedFuturesIncomeHistoryRate, &resp)
}

// CoinMarginedFuturesNotionalBracket returns the coin margined futures Notional Bracket for Pair, such as BTCUSD, or
// for every pair when it is empty. Binance discourages this endpoint because a pair's brackets can differ per symbol
func (e *Exchange) CoinMarginedFuturesNotionalBracket(ctx context.Context, pair currency.Code) ([]NotionalBracketResponse, error) {
	params := url.Values{}
	if !pair.IsEmpty() {
		params.Set("pair", pair.Upper().String())
	}
	var resp []NotionalBracketResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/leverageBracket", params, coinMarginedFuturesDefaultRate, &resp)
}

// GetCoinMarginedFuturesBasisData returns the coin margined futures Basis
func (e *Exchange) GetCoinMarginedFuturesBasisData(ctx context.Context, req *CoinMarginedFuturesStatisticsRequest) ([]CoinMarginedFuturesBasisResponse, error) {
	params, err := e.futuresStatisticsParams(req, true)
	if err != nil {
		return nil, err
	}
	var resp []CoinMarginedFuturesBasisResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/futures/data/basis", params), futuresDataRate, &resp)
}

// futuresStatisticsParams builds the parameters shared by the /futures/data statistics endpoints
func (e *Exchange) futuresStatisticsParams(req *CoinMarginedFuturesStatisticsRequest, contractTypeRequired bool) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if contractTypeRequired && req.ContractType == "" {
		return nil, errContractTypeIsRequired
	}
	if req.Period == 0 {
		return nil, errInvalidPeriodOrInterval
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("pair", req.Pair.Upper().String())
	if req.ContractType != "" {
		params.Set("contractType", req.ContractType)
	}
	params.Set("period", e.FormatExchangeKlineInterval(req.Period))
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	return params, nil
}

// CoinMarginedFuturesServerTime returns the coin margined futures server time from Check Server Time
func (e *Exchange) CoinMarginedFuturesServerTime(ctx context.Context) (time.Time, error) {
	var resp *ServerTimeResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestCoinMargined, "/dapi/v1/time", coinMarginedFuturesDefaultRate, &resp); err != nil {
		return time.Time{}, err
	}
	return resp.ServerTime.Time(), nil
}

// GetCoinMarginedFuturesAggregatedTradesList returns the coin margined futures Compressed/Aggregate Trades List
func (e *Exchange) GetCoinMarginedFuturesAggregatedTradesList(ctx context.Context, req *CoinMarginedFuturesAggregatedTradesRequest) ([]CoinMarginedFuturesAggregatedTradeResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if req.FromID != 0 {
		params.Set("fromId", strconv.FormatUint(req.FromID, 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []CoinMarginedFuturesAggregatedTradeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/aggTrades", params), coinMarginedFuturesAggregateTradesRate, &resp)
}

// GetCoinMarginedFuturesContinuousKlineData returns the coin margined futures Continuous Contract Kline/Candlestick
// Data
func (e *Exchange) GetCoinMarginedFuturesContinuousKlineData(ctx context.Context, req *CoinMarginedFuturesContinuousKlineRequest) ([]CoinMarginedFuturesCandleStickResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.ContractType == "" {
		return nil, errContractTypeIsRequired
	}
	params, err := e.klineParams(req.Interval, req.StartTime, req.EndTime, req.Limit)
	if err != nil {
		return nil, err
	}
	params.Set("pair", req.Pair.Upper().String())
	params.Set("contractType", req.ContractType)
	var resp []CoinMarginedFuturesCandleStickResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/continuousKlines", params), coinMarginedFuturesKlineLimit(req.Limit), &resp)
}

// klineParams builds the interval, time window and limit parameters shared by the coin margined futures kline
// endpoints
func (e *Exchange) klineParams(interval kline.Interval, startTime, endTime time.Time, limit uint64) (url.Values, error) {
	if interval == 0 {
		return nil, kline.ErrInvalidInterval
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("interval", e.FormatExchangeKlineInterval(interval))
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit != 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	return params, nil
}

// CoinMarginedFuturesExchangeInfo returns the coin margined futures Exchange Information
func (e *Exchange) CoinMarginedFuturesExchangeInfo(ctx context.Context) (*CoinMarginedFuturesExchangeInfoResponse, error) {
	var resp *CoinMarginedFuturesExchangeInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, "/dapi/v1/exchangeInfo", coinMarginedFuturesDefaultRate, &resp)
}

// CoinMarginedFuturesGetFundingHistory returns the coin margined futures Funding Rate History of Perpetual Futures;
// delivery symbols return no records
func (e *Exchange) CoinMarginedFuturesGetFundingHistory(ctx context.Context, req *CoinMarginedFuturesFundingRateHistoryRequest) ([]CoinMarginedFuturesFundingRateHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []CoinMarginedFuturesFundingRateHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/fundingRate", params), coinMarginedFuturesDefaultRate, &resp)
}

// GetCoinMarginedFuturesFundingRateInfo returns the coin margined futures Funding Rate Info, which only lists symbols
// whose funding rate cap, floor or interval was adjusted
func (e *Exchange) GetCoinMarginedFuturesFundingRateInfo(ctx context.Context) ([]CoinMarginedFuturesFundingRateInfoResponse, error) {
	var resp []CoinMarginedFuturesFundingRateInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, "/dapi/v1/fundingInfo", coinMarginedFuturesDefaultRate, &resp)
}

// GetCoinMarginedFuturesIndexAndMarkPrice returns the coin margined futures Index Price and Mark Price of a symbol, of
// every symbol of a pair such as BTCUSD, or of every symbol when both are empty. Binance returns every symbol whatever
// the pair, so the pair is applied to the reply
func (e *Exchange) GetCoinMarginedFuturesIndexAndMarkPrice(ctx context.Context, symbol currency.Pair, pair currency.Code) ([]IndexMarkPriceResponse, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		s, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", s)
	}
	if !pair.IsEmpty() {
		params.Set("pair", pair.Upper().String())
	}
	var resp []IndexMarkPriceResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/premiumIndex", params), coinMarginedFuturesIndexMarkPriceRate, &resp); err != nil {
		return nil, err
	}
	if !pair.IsEmpty() {
		resp = slices.DeleteFunc(resp, func(m IndexMarkPriceResponse) bool { return !strings.EqualFold(m.Pair, pair.String()) })
	}
	return resp, nil
}

// GetCoinMarginedFuturesIndexPriceKlines returns the coin margined futures Index Price Kline/Candlestick Data
func (e *Exchange) GetCoinMarginedFuturesIndexPriceKlines(ctx context.Context, req *CoinMarginedFuturesIndexPriceKlineRequest) ([]CoinMarginedFuturesPriceCandleStickResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params, err := e.klineParams(req.Interval, req.StartTime, req.EndTime, req.Limit)
	if err != nil {
		return nil, err
	}
	params.Set("pair", req.Pair.Upper().String())
	var resp []CoinMarginedFuturesPriceCandleStickResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/indexPriceKlines", params), coinMarginedFuturesKlineLimit(req.Limit), &resp)
}

// GetCoinMarginedFuturesKlineData returns the coin margined futures Kline/Candlestick Data
func (e *Exchange) GetCoinMarginedFuturesKlineData(ctx context.Context, req *CoinMarginedFuturesKlineRequest) ([]CoinMarginedFuturesCandleStickResponse, error) {
	params, err := e.symbolKlineParams(req)
	if err != nil {
		return nil, err
	}
	var resp []CoinMarginedFuturesCandleStickResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/klines", params), coinMarginedFuturesKlineLimit(req.Limit), &resp)
}

// symbolKlineParams builds the parameters of the coin margined futures kline endpoints that take a symbol
func (e *Exchange) symbolKlineParams(req *CoinMarginedFuturesKlineRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params, err := e.klineParams(req.Interval, req.StartTime, req.EndTime, req.Limit)
	if err != nil {
		return nil, err
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	return params, nil
}

// GetCoinMarginedFuturesMarketRatio returns the coin margined futures Long/Short Ratio of every account
func (e *Exchange) GetCoinMarginedFuturesMarketRatio(ctx context.Context, req *CoinMarginedFuturesStatisticsRequest) ([]CoinMarginedFuturesLongShortAccountRatioResponse, error) {
	params, err := e.futuresStatisticsParams(req, false)
	if err != nil {
		return nil, err
	}
	var resp []CoinMarginedFuturesLongShortAccountRatioResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/futures/data/globalLongShortAccountRatio", params), futuresDataRate, &resp)
}

// GetCoinMarginedFuturesMarkPriceKline returns the coin margined futures Mark Price Kline/Candlestick Data
func (e *Exchange) GetCoinMarginedFuturesMarkPriceKline(ctx context.Context, req *CoinMarginedFuturesKlineRequest) ([]CoinMarginedFuturesPriceCandleStickResponse, error) {
	params, err := e.symbolKlineParams(req)
	if err != nil {
		return nil, err
	}
	var resp []CoinMarginedFuturesPriceCandleStickResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/markPriceKlines", params), coinMarginedFuturesKlineLimit(req.Limit), &resp)
}

// GetCoinMarginedFuturesHistoricalTrades returns coin margined futures trades older than the recent trades list,
// through Old Trades Lookup, which requires an API key
func (e *Exchange) GetCoinMarginedFuturesHistoricalTrades(ctx context.Context, symbol currency.Pair, fromID, limit uint64) ([]CoinMarginedFuturesPublicTradesResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	s, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", s)
	if limit != 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	if fromID != 0 {
		params.Set("fromId", strconv.FormatUint(fromID, 10))
	}
	var resp []CoinMarginedFuturesPublicTradesResponse
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, common.EncodeURLValues("/dapi/v1/historicalTrades", params), coinMarginedFuturesHistoricalTradesRate, &resp)
}

// CoinMarginedFuturesOpenInterest returns the coin margined futures Open Interest of a symbol
func (e *Exchange) CoinMarginedFuturesOpenInterest(ctx context.Context, symbol currency.Pair) (*CoinMarginedFuturesOpenInterestResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	s, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", s)
	var resp *CoinMarginedFuturesOpenInterestResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/openInterest", params), coinMarginedFuturesDefaultRate, &resp)
}

// GetCoinMarginedFuturesOpenInterestStats returns the coin margined futures Open Interest Statistics; an empty contract
// type aggregates every contract type
func (e *Exchange) GetCoinMarginedFuturesOpenInterestStats(ctx context.Context, req *CoinMarginedFuturesStatisticsRequest) ([]OpenInterestStatsResponse, error) {
	params, err := e.futuresStatisticsParams(req, false)
	if err != nil {
		return nil, err
	}
	var resp []OpenInterestStatsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/futures/data/openInterestHist", params), futuresDataRate, &resp)
}

// GetCoinMarginedFuturesOrderbook returns the coin margined futures Order Book of a symbol
func (e *Exchange) GetCoinMarginedFuturesOrderbook(ctx context.Context, symbol currency.Pair, limit uint64) (*CoinMarginedFuturesOrderBookResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	s, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", s)
	if limit != 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp *CoinMarginedFuturesOrderBookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/depth", params), coinMarginedFuturesOrderbookLimit(limit), &resp)
}

// GetCoinMarginedFuturesPremiumIndexKlineData returns the coin margined futures Premium index Kline Data
func (e *Exchange) GetCoinMarginedFuturesPremiumIndexKlineData(ctx context.Context, req *CoinMarginedFuturesKlineRequest) ([]CoinMarginedFuturesPremiumIndexCandleStickResponse, error) {
	params, err := e.symbolKlineParams(req)
	if err != nil {
		return nil, err
	}
	var resp []CoinMarginedFuturesPremiumIndexCandleStickResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/premiumIndexKlines", params), coinMarginedFuturesKlineLimit(req.Limit), &resp)
}

// GetCoinMarginedFuturesIndexPriceConstituents returns the coin margined futures Index Price Constituents of a pair
// such as BTCUSD, which the API calls the symbol
func (e *Exchange) GetCoinMarginedFuturesIndexPriceConstituents(ctx context.Context, pair currency.Code) (*CoinMarginedFuturesIndexPriceConstituentsResponse, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", pair.Upper().String())
	var resp *CoinMarginedFuturesIndexPriceConstituentsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/constituents", params), coinMarginedFuturesDefaultRate, &resp)
}

// GetCoinMarginedFuturesPublicTrades returns the coin margined futures Recent Trades List
func (e *Exchange) GetCoinMarginedFuturesPublicTrades(ctx context.Context, symbol currency.Pair, limit uint64) ([]CoinMarginedFuturesPublicTradesResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	s, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", s)
	if limit != 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []CoinMarginedFuturesPublicTradesResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/trades", params), coinMarginedFuturesRecentTradesRate, &resp)
}

// symbolOrPairParams builds the symbol or pair filter of the coin margined futures endpoints that reject both
// together
func (e *Exchange) symbolOrPairParams(symbol currency.Pair, pair currency.Code) (url.Values, error) {
	if !symbol.IsEmpty() && !pair.IsEmpty() {
		return nil, errSymbolAndPairMutuallyExclusive
	}
	params := url.Values{}
	if !symbol.IsEmpty() {
		s, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", s)
	}
	if !pair.IsEmpty() {
		params.Set("pair", pair.Upper().String())
	}
	return params, nil
}

// GetCoinMarginedFuturesOrderbookTicker returns the coin margined futures Symbol Order Book Ticker of a symbol, of
// every symbol of a pair, or of every symbol when both are empty
func (e *Exchange) GetCoinMarginedFuturesOrderbookTicker(ctx context.Context, symbol currency.Pair, pair currency.Code) ([]SymbolOrderBookTickerResponse, error) {
	params, err := e.symbolOrPairParams(symbol, pair)
	if err != nil {
		return nil, err
	}
	rateLimit := coinMarginedFuturesOrderbookTickerAllRate
	if !symbol.IsEmpty() {
		rateLimit = coinMarginedFuturesBookTickerRate
	}
	var resp []SymbolOrderBookTickerResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/ticker/bookTicker", params), rateLimit, &resp)
}

// GetCoinMarginedFuturesSymbolPriceTicker returns the coin margined futures Symbol Price Ticker of a symbol, of every
// symbol of a pair, or of every symbol when both are empty
func (e *Exchange) GetCoinMarginedFuturesSymbolPriceTicker(ctx context.Context, symbol currency.Pair, pair currency.Code) ([]SymbolPriceTickerResponse, error) {
	params, err := e.symbolOrPairParams(symbol, pair)
	if err != nil {
		return nil, err
	}
	rateLimit := coinMarginedFuturesSymbolPriceAllRate
	if !symbol.IsEmpty() {
		rateLimit = coinMarginedFuturesDefaultRate
	}
	var resp []SymbolPriceTickerResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/ticker/price", params), rateLimit, &resp)
}

// GetCoinMarginedFuturesTakerVolume returns the coin margined futures Taker Buy/Sell Volume
func (e *Exchange) GetCoinMarginedFuturesTakerVolume(ctx context.Context, req *CoinMarginedFuturesStatisticsRequest) ([]TakerBuySellVolumeResponse, error) {
	params, err := e.futuresStatisticsParams(req, true)
	if err != nil {
		return nil, err
	}
	var resp []TakerBuySellVolumeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/futures/data/takerBuySellVol", params), futuresDataRate, &resp)
}

// GetCoinMarginedFuturesSwapTickerChangeStats returns the coin margined futures 24hr Ticker Price Change Statistics of
// a symbol, of every symbol of a pair, or of every symbol when both are empty
func (e *Exchange) GetCoinMarginedFuturesSwapTickerChangeStats(ctx context.Context, symbol currency.Pair, pair currency.Code) ([]CoinMarginedFuturesPriceChangeStatsResponse, error) {
	params, err := e.symbolOrPairParams(symbol, pair)
	if err != nil {
		return nil, err
	}
	rateLimit := coinMarginedFuturesTicker24HourAllRate
	if !symbol.IsEmpty() {
		rateLimit = coinMarginedFuturesDefaultRate
	}
	var resp []CoinMarginedFuturesPriceChangeStatsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/ticker/24hr", params), rateLimit, &resp)
}

// GetCoinMarginedFuturesTraderAccountRatio returns the coin margined futures Top Trader Long/Short Account Ratio
func (e *Exchange) GetCoinMarginedFuturesTraderAccountRatio(ctx context.Context, req *CoinMarginedFuturesStatisticsRequest) ([]CoinMarginedFuturesLongShortAccountRatioResponse, error) {
	params, err := e.futuresStatisticsParams(req, false)
	if err != nil {
		return nil, err
	}
	var resp []CoinMarginedFuturesLongShortAccountRatioResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/futures/data/topLongShortAccountRatio", params), futuresDataRate, &resp)
}

// GetCoinMarginedFuturesTraderPositionsRatio returns the coin margined futures Top Trader Long/Short Position Ratio
func (e *Exchange) GetCoinMarginedFuturesTraderPositionsRatio(ctx context.Context, req *CoinMarginedFuturesStatisticsRequest) ([]TopTraderPositionRatioResponse, error) {
	params, err := e.futuresStatisticsParams(req, false)
	if err != nil {
		return nil, err
	}
	var resp []TopTraderPositionRatioResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/futures/data/topLongShortPositionRatio", params), futuresDataRate, &resp)
}

// CoinMarginedFuturesTradeHistory returns the coin margined futures Account Trade List of a symbol or pair
func (e *Exchange) CoinMarginedFuturesTradeHistory(ctx context.Context, req *CoinMarginedFuturesAccountTradeListRequest) ([]CoinMarginedFuturesAccountTradeListResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := checkSymbolOrPair(req.Symbol, req.Pair, req.OrderID); err != nil {
		return nil, err
	}
	if req.FromID != 0 && (!req.Pair.IsEmpty() || !req.StartTime.IsZero() || !req.EndTime.IsZero()) {
		return nil, errFromIDMutuallyExclusive
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params, err := e.symbolOrPairParams(req.Symbol, req.Pair)
	if err != nil {
		return nil, err
	}
	if req.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.FromID != 0 {
		params.Set("fromId", strconv.FormatUint(req.FromID, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []CoinMarginedFuturesAccountTradeListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/userTrades", params, coinMarginedFuturesAccountTradeListRate, &resp)
}

// checkSymbolOrPair checks the documented rules of the coin margined futures order and trade history endpoints:
// exactly one of symbol and pair, and an order ID only with a symbol
func checkSymbolOrPair(symbol currency.Pair, pair currency.Code, orderID uint64) error {
	switch {
	case symbol.IsEmpty() && pair.IsEmpty():
		return errSymbolOrPairRequired
	case orderID != 0 && symbol.IsEmpty():
		return errOrderIDRequiresSymbol
	}
	return nil
}

// GetAllCoinMarginedFuturesOrders returns All Orders of a coin margined futures symbol or pair, whether active,
// cancelled or filled
func (e *Exchange) GetAllCoinMarginedFuturesOrders(ctx context.Context, req *CoinMarginedFuturesAllOrdersRequest) ([]CoinMarginedFuturesAllOrdersResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := checkSymbolOrPair(req.Symbol, req.Pair, req.OrderID); err != nil {
		return nil, err
	}
	params, err := e.symbolOrPairParams(req.Symbol, req.Pair)
	if err != nil {
		return nil, err
	}
	if err := setTimeRangeParams(params, req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []CoinMarginedFuturesAllOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/allOrders", params, coinMarginedFuturesAllOrdersRate, &resp)
}

// CoinMarginedFuturesAutoCancelAllOpenOrders sets the coin margined futures Auto-Cancel All Open Orders countdown of a
// symbol, after which its open orders are cancelled unless the countdown is renewed; a zero countdown cancels the
// timer. Renewals draw on a rate limit share of their own, which USDⓈ-M renewals share, while it can take them at once
func (e *Exchange) CoinMarginedFuturesAutoCancelAllOpenOrders(ctx context.Context, symbol currency.Pair, countdown time.Duration) (*CoinMarginedFuturesAutoCancelAllOpenOrdersResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	s, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", s)
	params.Set("countdownTime", strconv.FormatInt(countdown.Milliseconds(), 10))
	var resp *CoinMarginedFuturesAutoCancelAllOpenOrdersResponse
	return resp, e.sendOnShare(ctx, coinMarginedFuturesCountdownCancelRate, coinMarginedFuturesHeartbeatOverflowRate, func(ctx context.Context, f request.EndpointLimit) error {
		return e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, "/dapi/v1/countdownCancelAll", params, f, &resp)
	})
}

// CoinMarginedFuturesCancelAllOpenOrders cancels all coin margined futures open orders of a symbol through Cancel All
// Open Orders
func (e *Exchange) CoinMarginedFuturesCancelAllOpenOrders(ctx context.Context, symbol currency.Pair) (*CoinMarginedFuturesStatusResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	s, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", s)
	var resp *CoinMarginedFuturesStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodDelete, "/dapi/v1/allOpenOrders", params, coinMarginedFuturesCancelRate, &resp)
}

// CoinMarginedFuturesModifyMultipleOrders modifies up to five coin margined futures limit orders through Modify
// Multiple Orders, which takes no price match; an entry Binance rejects carries its error code and message instead of
// the order
func (e *Exchange) CoinMarginedFuturesModifyMultipleOrders(ctx context.Context, orders []CoinMarginedFuturesModifyOrderRequest) ([]CoinMarginedFuturesModifyBatchOrderResponse, error) {
	if len(orders) == 0 {
		return nil, common.ErrEmptyParams
	}
	if err := checkListLength("orders", len(orders), batchOrdersLimit); err != nil {
		return nil, err
	}
	batch := make([]map[string]string, len(orders))
	for i := range orders {
		if orders[i].PriceMatch != "" {
			return nil, errBatchModifyPriceMatch
		}
		params, err := e.futuresModifyOrderParams(&orders[i])
		if err != nil {
			return nil, err
		}
		batch[i] = make(map[string]string, len(params))
		for k := range params {
			batch[i][k] = params.Get(k)
		}
	}
	batchOrders, err := json.Marshal(batch)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("batchOrders", string(batchOrders))
	var resp []CoinMarginedFuturesModifyBatchOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPut, "/dapi/v1/batchOrders", params, coinMarginedFuturesBatchOrdersRate, &resp)
}

// CoinMarginedFuturesBatchOrder places up to five coin margined futures orders through Place Multiple Orders; an entry
// Binance rejects carries its error code and message instead of the order. Binance is moving stop, take profit and
// trailing stop orders to its algo order service, after which they are rejected with -4120
func (e *Exchange) CoinMarginedFuturesBatchOrder(ctx context.Context, orders []CoinMarginedFuturesNewOrderRequest) ([]CoinMarginedFuturesBatchOrderResponse, error) {
	if len(orders) == 0 {
		return nil, common.ErrEmptyParams
	}
	if err := checkListLength("orders", len(orders), batchOrdersLimit); err != nil {
		return nil, err
	}
	batch := make([]map[string]string, len(orders))
	for i := range orders {
		// Batch orders take no close position flag and always need a quantity, unlike New Order
		if orders[i].ClosePosition {
			return nil, errBatchClosePosition
		}
		if orders[i].Quantity <= 0 {
			return nil, limits.ErrAmountBelowMin
		}
		params, err := e.futuresOrderParams(&orders[i])
		if err != nil {
			return nil, err
		}
		batch[i] = make(map[string]string, len(params))
		for k := range params {
			batch[i][k] = params.Get(k)
		}
	}
	batchOrders, err := json.Marshal(batch)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("batchOrders", string(batchOrders))
	var resp []CoinMarginedFuturesBatchOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, "/dapi/v1/batchOrders", params, coinMarginedFuturesBatchOrdersRate, &resp)
}

// CoinMarginedFuturesBatchCancelOrders cancels up to ten coin margined futures orders of a symbol by order ID and up to
// ten by client order ID through Cancel Multiple Orders; an entry Binance rejects carries its error code and message
// instead of the order
func (e *Exchange) CoinMarginedFuturesBatchCancelOrders(ctx context.Context, symbol currency.Pair, orderIDs []uint64, origClientOrderIDs []string) ([]CoinMarginedFuturesBatchOrderResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if len(orderIDs) == 0 && len(origClientOrderIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if err := checkListLength("order IDs", len(orderIDs), batchCancelLimit); err != nil {
		return nil, err
	}
	if err := checkListLength("client order IDs", len(origClientOrderIDs), batchCancelLimit); err != nil {
		return nil, err
	}
	s, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", s)
	if len(orderIDs) != 0 {
		orderIDList, err := json.Marshal(orderIDs)
		if err != nil {
			return nil, err
		}
		params.Set("orderIdList", string(orderIDList))
	}
	if len(origClientOrderIDs) != 0 {
		clientOrderIDList, err := json.Marshal(origClientOrderIDs)
		if err != nil {
			return nil, err
		}
		params.Set("origClientOrderIdList", string(clientOrderIDList))
	}
	var resp []CoinMarginedFuturesBatchOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodDelete, "/dapi/v1/batchOrders", params, coinMarginedFuturesCancelRate, &resp)
}

// orderIdentityParams builds the symbol and order identity parameters of the single coin margined futures order
// endpoints, which require an order ID or a client order ID
func (e *Exchange) orderIdentityParams(symbol currency.Pair, orderID uint64, origClientOrderID string) (url.Values, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if orderID == 0 && origClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	s, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", s)
	if orderID != 0 {
		params.Set("orderId", strconv.FormatUint(orderID, 10))
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	return params, nil
}

// CoinMarginedFuturesGetOrderData returns a coin margined futures order through Query Order
func (e *Exchange) CoinMarginedFuturesGetOrderData(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*CoinMarginedFuturesOrderDetailResponse, error) {
	params, err := e.orderIdentityParams(symbol, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *CoinMarginedFuturesOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/order", params, coinMarginedFuturesDefaultRate, &resp)
}

// CoinMarginedFuturesModifyOrder modifies a coin margined futures limit order's price and quantity through Modify
// Order, which moves the order to the back of the matching queue
func (e *Exchange) CoinMarginedFuturesModifyOrder(ctx context.Context, req *CoinMarginedFuturesModifyOrderRequest) (*CoinMarginedFuturesModifyOrderResponse, error) {
	params, err := e.futuresModifyOrderParams(req)
	if err != nil {
		return nil, err
	}
	var resp *CoinMarginedFuturesModifyOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPut, "/dapi/v1/order", params, coinMarginedFuturesOrdersDefaultRate, &resp)
}

// futuresModifyOrderParams converts a modify request into the parameters Modify Order and Modify Multiple Orders share
func (e *Exchange) futuresModifyOrderParams(req *CoinMarginedFuturesModifyOrderRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := e.orderIdentityParams(req.Symbol, req.OrderID, req.OrigClientOrderID)
	if err != nil {
		return nil, err
	}
	if req.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if req.Quantity <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if req.PriceMatch != "" && req.Price != 0 {
		return nil, errPriceMatchWithPrice
	}
	if req.PriceMatch == "" && req.Price <= 0 {
		return nil, limits.ErrPriceBelowMin
	}
	params.Set("side", req.Side)
	params.Set("quantity", strconv.FormatFloat(req.Quantity, 'f', -1, 64))
	if req.Price != 0 {
		params.Set("price", strconv.FormatFloat(req.Price, 'f', -1, 64))
	}
	if req.PriceMatch != "" {
		params.Set("priceMatch", req.PriceMatch)
	}
	if req.ModifyID != 0 {
		params.Set("modifyId", strconv.FormatUint(req.ModifyID, 10))
	}
	return params, nil
}

// CoinMarginedFuturesNewOrder places a coin margined futures order through New Order. Since the CM-UM integration,
// stop, take profit and trailing stop orders are placed with CoinMarginedFuturesNewAlgoOrder, and New Order rejects
// them with -4120 once Binance's announced cut-over takes effect
func (e *Exchange) CoinMarginedFuturesNewOrder(ctx context.Context, req *CoinMarginedFuturesNewOrderRequest) (*CoinMarginedFuturesOrderResponse, error) {
	params, err := e.futuresOrderParams(req)
	if err != nil {
		return nil, err
	}
	var resp *CoinMarginedFuturesOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, "/dapi/v1/order", params, coinMarginedFuturesOrdersDefaultRate, &resp)
}

// futuresOrderParams builds the parameters of a coin margined futures order, which New Order sends as they are and
// Place Multiple Orders sends as one JSON object per order
func (e *Exchange) futuresOrderParams(req *CoinMarginedFuturesNewOrderRequest) (url.Values, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if req.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", req.Side)
	if req.PositionSide != "" {
		params.Set("positionSide", req.PositionSide)
	}
	params.Set("type", req.OrderType)
	if req.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	if req.Quantity != 0 {
		params.Set("quantity", strconv.FormatFloat(req.Quantity, 'f', -1, 64))
	}
	if req.Price != 0 {
		params.Set("price", strconv.FormatFloat(req.Price, 'f', -1, 64))
	}
	if req.NewClientOrderID != "" {
		params.Set("newClientOrderId", req.NewClientOrderID)
	}
	if req.StopPrice != 0 {
		params.Set("stopPrice", strconv.FormatFloat(req.StopPrice, 'f', -1, 64))
	}
	if req.ClosePosition {
		params.Set("closePosition", "true")
	}
	if req.ActivationPrice != 0 {
		params.Set("activationPrice", strconv.FormatFloat(req.ActivationPrice, 'f', -1, 64))
	}
	if req.CallbackRate != 0 {
		params.Set("callbackRate", strconv.FormatFloat(req.CallbackRate, 'f', -1, 64))
	}
	if req.TimeInForce != "" {
		params.Set("timeInForce", req.TimeInForce)
	}
	if req.WorkingType != "" {
		params.Set("workingType", req.WorkingType)
	}
	if req.PriceProtect {
		params.Set("priceProtect", "true")
	}
	if req.NewOrderResponseType != "" {
		params.Set("newOrderRespType", req.NewOrderResponseType)
	}
	if req.PriceMatch != "" {
		params.Set("priceMatch", req.PriceMatch)
	}
	if req.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", req.SelfTradePreventionMode)
	}
	return params, nil
}

// CoinMarginedFuturesCancelOrder cancels a coin margined futures order through Cancel Order
func (e *Exchange) CoinMarginedFuturesCancelOrder(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*CoinMarginedFuturesOrderResponse, error) {
	params, err := e.orderIdentityParams(symbol, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *CoinMarginedFuturesOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodDelete, "/dapi/v1/order", params, coinMarginedFuturesCancelRate, &resp)
}

// CoinMarginedFuturesQueryAlgoOrder returns a coin margined futures algo (conditional) order's state from Query Algo
// Order, by its algo ID or, when algoID is 0, its client algo ID. Binance gives COIN-M the USDⓈ-M algo order endpoints,
// taking the same parameters and replying alike, so the USDⓈ-M algo order types serve both
func (e *Exchange) CoinMarginedFuturesQueryAlgoOrder(ctx context.Context, algoID uint64, clientAlgoID string) (*USDTMarginedFuturesAlgoOrderResponse, error) {
	return e.futuresQueryAlgoOrder(ctx, algoID, clientAlgoID, coinMarginedFuturesDefaultRate)
}

// futuresQueryAlgoOrder queries a coin margined futures algo order drawing on f, so that the lookup a cancellation
// makes can draw on the cancel pool as the cancellation does, rather than queue behind order book snapshots on the IP
// pool
func (e *Exchange) futuresQueryAlgoOrder(ctx context.Context, algoID uint64, clientAlgoID string, f request.EndpointLimit) (*USDTMarginedFuturesAlgoOrderResponse, error) {
	params, err := algoOrderIDParams(algoID, clientAlgoID)
	if err != nil {
		return nil, err
	}
	var resp *USDTMarginedFuturesAlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/algoOrder", params, f, &resp)
}

// CoinMarginedFuturesNewAlgoOrder places a coin margined futures algo (conditional) order with New Algo Order, which is
// how stop, take profit and trailing stop orders are placed since the CM-UM integration moved them off New Order. The
// quantity is in contracts
func (e *Exchange) CoinMarginedFuturesNewAlgoOrder(ctx context.Context, arg *USDTMarginedFuturesNewAlgoOrderRequest) (*USDTMarginedFuturesNewAlgoOrderResponse, error) {
	params, err := e.newAlgoOrderParams(arg, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	var resp *USDTMarginedFuturesNewAlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, "/dapi/v1/algoOrder", params, coinMarginedFuturesOrdersDefaultRate, &resp)
}

// CoinMarginedFuturesCancelAlgoOrder cancels an active coin margined futures algo (conditional) order with Cancel Algo
// Order, by its algo ID or, when algoID is 0, its client algo ID
func (e *Exchange) CoinMarginedFuturesCancelAlgoOrder(ctx context.Context, algoID uint64, clientAlgoID string) (*USDTMarginedFuturesCancelAlgoOrderResponse, error) {
	params, err := algoOrderIDParams(algoID, clientAlgoID)
	if err != nil {
		return nil, err
	}
	var resp *USDTMarginedFuturesCancelAlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodDelete, "/dapi/v1/algoOrder", params, coinMarginedFuturesCancelRate, &resp)
}

// CoinMarginedFuturesCancelAllAlgoOpenOrders cancels a coin margined futures symbol's open algo (conditional) orders
// with Cancel All Algo Open Orders
func (e *Exchange) CoinMarginedFuturesCancelAllAlgoOpenOrders(ctx context.Context, symbol currency.Pair) error {
	if symbol.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	s, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return err
	}
	params := url.Values{}
	params.Set("symbol", s)
	return e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodDelete, "/dapi/v1/algoOpenOrders", params, coinMarginedFuturesCancelRate, nil)
}

// CoinMarginedFuturesCurrentAllAlgoOpenOrders returns open coin margined futures algo (conditional) orders from Current
// All Algo Open Orders; each argument is an optional filter, and an empty symbol returns every symbol's orders
func (e *Exchange) CoinMarginedFuturesCurrentAllAlgoOpenOrders(ctx context.Context, algoType string, symbol currency.Pair, algoID uint64) ([]USDTMarginedFuturesAlgoOrderResponse, error) {
	return e.futuresCurrentAllAlgoOpenOrders(ctx, algoType, symbol, algoID, coinMarginedFuturesOpenAlgoOrdersAllRate)
}

// futuresCurrentAllAlgoOpenOrders lists open coin margined futures algo orders, drawing on allSymbols for a listing of
// every symbol's, so the listing a cancellation of every symbol's orders makes can draw on the cancel pool
func (e *Exchange) futuresCurrentAllAlgoOpenOrders(ctx context.Context, algoType string, symbol currency.Pair, algoID uint64, allSymbols request.EndpointLimit) ([]USDTMarginedFuturesAlgoOrderResponse, error) {
	params := url.Values{}
	rateLimit := allSymbols
	if algoType != "" {
		params.Set("algoType", algoType)
	}
	if !symbol.IsEmpty() {
		s, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", s)
		rateLimit = coinMarginedFuturesDefaultRate
	}
	if algoID > 0 {
		params.Set("algoId", strconv.FormatUint(algoID, 10))
	}
	var resp []USDTMarginedFuturesAlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/openAlgoOrders", params, rateLimit, &resp)
}

// CoinMarginedFuturesQueryAllAlgoOrders returns a coin margined futures symbol's active, cancelled, triggered and
// finished algo (conditional) orders from Query All Algo Orders. A time window may span at most seven days
func (e *Exchange) CoinMarginedFuturesQueryAllAlgoOrders(ctx context.Context, arg *USDTMarginedFuturesAllAlgoOrdersRequest) ([]USDTMarginedFuturesAlgoOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	s, err := e.FormatSymbol(arg.Symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", s)
	if arg.AlgoID > 0 {
		params.Set("algoId", strconv.FormatUint(arg.AlgoID, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []USDTMarginedFuturesAlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/allAlgoOrders", params, coinMarginedFuturesAllAlgoOrdersRate, &resp)
}

// CoinMarginedFuturesChangeInitialLeverage sets the initial leverage of a coin margined futures symbol through Change
// Initial Leverage
func (e *Exchange) CoinMarginedFuturesChangeInitialLeverage(ctx context.Context, symbol currency.Pair, leverage uint64) (*CoinMarginedFuturesChangeLeverageResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if leverage == 0 {
		return nil, errLeverageRequired
	}
	s, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", s)
	params.Set("leverage", strconv.FormatUint(leverage, 10))
	var resp *CoinMarginedFuturesChangeLeverageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, "/dapi/v1/leverage", params, coinMarginedFuturesDefaultRate, &resp)
}

// CoinMarginedFuturesChangeMarginType sets the margin type, ISOLATED or CROSSED, of a coin margined futures symbol
// through Change Margin Type
func (e *Exchange) CoinMarginedFuturesChangeMarginType(ctx context.Context, symbol currency.Pair, marginType string) (*CoinMarginedFuturesStatusResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if marginType == "" {
		return nil, margin.ErrInvalidMarginType
	}
	s, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", s)
	params.Set("marginType", marginType)
	var resp *CoinMarginedFuturesStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, "/dapi/v1/marginType", params, coinMarginedFuturesDefaultRate, &resp)
}

// GetCoinMarginedFuturesAllOpenOrders returns the coin margined futures Current All Open Orders of a symbol, of every
// symbol of a pair, or of every symbol when both are empty
func (e *Exchange) GetCoinMarginedFuturesAllOpenOrders(ctx context.Context, symbol currency.Pair, pair currency.Code) ([]CoinMarginedFuturesOrderDetailResponse, error) {
	return e.getFuturesAllOpenOrders(ctx, symbol, pair, coinMarginedFuturesGetAllOpenOrdersRate)
}

// getFuturesAllOpenOrders lists coin margined futures open orders, drawing on allSymbols for a listing of every
// symbol's, so the listing a cancellation of every symbol's orders makes can draw on the cancel pool
func (e *Exchange) getFuturesAllOpenOrders(ctx context.Context, symbol currency.Pair, pair currency.Code, allSymbols request.EndpointLimit) ([]CoinMarginedFuturesOrderDetailResponse, error) {
	params := url.Values{}
	rateLimit := allSymbols
	if !symbol.IsEmpty() {
		rateLimit = coinMarginedFuturesDefaultRate
		s, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", s)
	}
	if !pair.IsEmpty() {
		params.Set("pair", pair.Upper().String())
	}
	var resp []CoinMarginedFuturesOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/openOrders", params, rateLimit, &resp)
}

// GetCoinMarginedFuturesOrderModifyHistory returns a coin margined futures order's modifications through Get Order
// Modify History; only the last three months are kept
func (e *Exchange) GetCoinMarginedFuturesOrderModifyHistory(ctx context.Context, req *CoinMarginedFuturesOrderModifyHistoryRequest) ([]FuturesOrderAmendmentResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := e.orderIdentityParams(req.Symbol, req.OrderID, req.OrigClientOrderID)
	if err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []FuturesOrderAmendmentResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/orderAmendment", params, coinMarginedFuturesDefaultRate, &resp)
}

// CoinMarginedFuturesMarginChangeHistory returns the coin margined futures Position Margin Change History of a symbol
func (e *Exchange) CoinMarginedFuturesMarginChangeHistory(ctx context.Context, req *CoinMarginedFuturesPositionMarginHistoryRequest) ([]GetPositionMarginChangeHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if req.Type != 0 {
		params.Set("type", strconv.FormatUint(req.Type, 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []GetPositionMarginChangeHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/positionMargin/history", params, coinMarginedFuturesDefaultRate, &resp)
}

// CoinMarginedFuturesModifyIsolatedPositionMargin adds margin to or removes margin from a coin margined futures
// isolated position through Modify Isolated Position Margin
func (e *Exchange) CoinMarginedFuturesModifyIsolatedPositionMargin(ctx context.Context, req *CoinMarginedFuturesModifyIsolatedPositionMarginRequest) (*CoinMarginedFuturesMarginUpdatedResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.Amount <= 0 {
		return nil, order.ErrAmountIsInvalid
	}
	if req.Type == 0 {
		return nil, errMarginChangeTypeInvalid
	}
	symbol, err := e.FormatSymbol(req.Symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if req.PositionSide != "" {
		params.Set("positionSide", req.PositionSide)
	}
	params.Set("amount", strconv.FormatFloat(req.Amount, 'f', -1, 64))
	params.Set("type", strconv.FormatUint(req.Type, 10))
	var resp *CoinMarginedFuturesMarginUpdatedResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, "/dapi/v1/positionMargin", params, coinMarginedFuturesDefaultRate, &resp)
}

// CoinMarginedFuturesPositionsAutoDeleveragingEstimate returns the coin margined futures Position ADL Quantile
// Estimation of a symbol, or of every symbol when it is empty
func (e *Exchange) CoinMarginedFuturesPositionsAutoDeleveragingEstimate(ctx context.Context, symbol currency.Pair) ([]AutoDeleveragingEstimateResponse, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		s, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", s)
	}
	var resp []AutoDeleveragingEstimateResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/adlQuantile", params, coinMarginedFuturesAutoDeleveragingQuantileRate, &resp)
}

// CoinMarginedFuturesPositionsInfo returns the coin margined futures Position Information of a margin asset or a pair
// such as BTCUSD, or of every trading symbol when both are empty
func (e *Exchange) CoinMarginedFuturesPositionsInfo(ctx context.Context, marginAsset, pair currency.Code) ([]CoinMarginedFuturesPositionInformationResponse, error) {
	params := url.Values{}
	if !marginAsset.IsEmpty() {
		params.Set("marginAsset", marginAsset.Upper().String())
	}
	if !pair.IsEmpty() {
		params.Set("pair", pair.Upper().String())
	}
	var resp []CoinMarginedFuturesPositionInformationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/positionRisk", params, coinMarginedFuturesDefaultRate, &resp)
}

// CoinMarginedFuturesOpenOrderData returns an open coin margined futures order through Query Current Open Order
func (e *Exchange) CoinMarginedFuturesOpenOrderData(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*CoinMarginedFuturesOrderDetailResponse, error) {
	params, err := e.orderIdentityParams(symbol, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *CoinMarginedFuturesOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/openOrder", params, coinMarginedFuturesDefaultRate, &resp)
}

// CoinMarginedFuturesForceOrders returns the coin margined futures orders Binance placed to liquidate or deleverage the
// account's positions, through User's Force Orders
func (e *Exchange) CoinMarginedFuturesForceOrders(ctx context.Context, req *CoinMarginedFuturesForceOrdersRequest) ([]ForcedOrdersResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	rateLimit := coinMarginedFuturesAllForceOrdersRate
	if !req.Symbol.IsEmpty() {
		rateLimit = coinMarginedFuturesSymbolForceOrdersRate
		symbol, err := e.FormatSymbol(req.Symbol, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbol)
	}
	if req.AutoCloseType != "" {
		params.Set("autoCloseType", req.AutoCloseType)
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []ForcedOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/forceOrders", params, rateLimit, &resp)
}

// CoinMarginedFuturesQuarterlyContractSettlementPrice returns a COIN-M pair's quarterly contract settlement prices.
// Binance only documents the endpoint for USDⓈ-M futures, but the COIN-M host serves its own pairs (BTCUSD) from the
// same path
func (e *Exchange) CoinMarginedFuturesQuarterlyContractSettlementPrice(ctx context.Context, pair currency.Code) ([]SettlementPriceResponse, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("pair", pair.Upper().String())
	var resp []SettlementPriceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/futures/data/delivery-price", params), futuresDataRate, &resp)
}

// KeepaliveCoinMarginedFuturesUserDataStream extends the validity of the account's COIN-M user data stream listen key
// by 60 minutes. A -1125 error means the key has expired and StartCoinMarginedFuturesUserDataStream must create a new
// one
func (e *Exchange) KeepaliveCoinMarginedFuturesUserDataStream(ctx context.Context) (*FuturesListenKeyResponse, error) {
	var resp *FuturesListenKeyResponse
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPut, "/dapi/v1/listenKey", coinMarginedFuturesDefaultRate, &resp)
}

// StartCoinMarginedFuturesUserDataStream starts a COIN-M user data stream. An account with an active listen key gets
// that key back with its validity extended by 60 minutes
func (e *Exchange) StartCoinMarginedFuturesUserDataStream(ctx context.Context) (*FuturesListenKeyResponse, error) {
	var resp *FuturesListenKeyResponse
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, "/dapi/v1/listenKey", coinMarginedFuturesDefaultRate, &resp)
}

// CloseCoinMarginedFuturesUserDataStream closes the account's COIN-M user data stream and invalidates its listen key
func (e *Exchange) CloseCoinMarginedFuturesUserDataStream(ctx context.Context) error {
	return e.SendAPIKeyHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodDelete, "/dapi/v1/listenKey", coinMarginedFuturesDefaultRate, nil)
}
