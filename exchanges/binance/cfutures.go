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
)

var (
	errBatchModifyPriceMatch          = errors.New("modify multiple orders takes no price match")
	errBatchClosePosition             = errors.New("place multiple orders takes no close position orders")
	errSymbolOrPairRequired           = errors.New("symbol or pair is required")
	errSymbolAndPairMutuallyExclusive = errors.New("symbol and pair cannot be sent together")
	errOrderIDRequiresSymbol          = errors.New("order ID can only be sent with symbol")
	errFromIDMutuallyExclusive        = errors.New("from ID cannot be sent with pair, start time or end time")
)

// GetFuturesAccountInfo returns the coin margined futures Account Information
func (e *Exchange) GetFuturesAccountInfo(ctx context.Context) (*FuturesAccountInformationResponse, error) {
	var resp *FuturesAccountInformationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/account", nil, cFuturesAccountInformationRate, &resp)
}

// GetFuturesAccountBalance returns the coin margined Futures Account Balance
func (e *Exchange) GetFuturesAccountBalance(ctx context.Context) ([]FuturesAccountBalanceData, error) {
	var resp []FuturesAccountBalanceData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/balance", nil, cFuturesDefaultRate, &resp)
}

// FuturesIncomeHistory returns the coin margined futures Income History
func (e *Exchange) FuturesIncomeHistory(ctx context.Context, req *CFuturesIncomeHistoryRequest) ([]FuturesIncomeHistoryData, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
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
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Page != 0 {
		params.Set("page", strconv.FormatUint(req.Page, 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []FuturesIncomeHistoryData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/income", params, cFuturesIncomeHistoryRate, &resp)
}

// FuturesNotionalBracket returns the coin margined futures Notional Bracket for Pair, such as BTCUSD, or for every
// pair when it is empty. Binance discourages this endpoint because a pair's brackets can differ per symbol
func (e *Exchange) FuturesNotionalBracket(ctx context.Context, pair currency.Code) ([]NotionalBracketData, error) {
	params := url.Values{}
	if !pair.IsEmpty() {
		params.Set("pair", pair.Upper().String())
	}
	var resp []NotionalBracketData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/leverageBracket", params, cFuturesDefaultRate, &resp)
}

// GetFuturesBasisData returns the coin margined futures Basis
func (e *Exchange) GetFuturesBasisData(ctx context.Context, req *CFuturesStatisticsRequest) ([]FuturesBasisData, error) {
	params, err := e.futuresStatisticsParams(req, true)
	if err != nil {
		return nil, err
	}
	var resp []FuturesBasisData
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/futures/data/basis", params), futuresDataRate, &resp)
}

// futuresStatisticsParams builds the parameters shared by the /futures/data statistics endpoints
func (e *Exchange) futuresStatisticsParams(req *CFuturesStatisticsRequest, contractTypeRequired bool) (url.Values, error) {
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

// CFuturesServerTime returns the coin margined futures server time from Check Server Time
func (e *Exchange) CFuturesServerTime(ctx context.Context) (time.Time, error) {
	var resp *ServerTimeResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestCoinMargined, "/dapi/v1/time", cFuturesDefaultRate, &resp); err != nil {
		return time.Time{}, err
	}
	return resp.ServerTime.Time(), nil
}

// GetFuturesAggregatedTradesList returns the coin margined futures Compressed/Aggregate Trades List
func (e *Exchange) GetFuturesAggregatedTradesList(ctx context.Context, req *CFuturesAggregatedTradesRequest) ([]CFuturesAggregatedTrade, error) {
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
	var resp []CFuturesAggregatedTrade
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/aggTrades", params), cFuturesAggregateTradesRate, &resp)
}

// GetContinuousKlineData returns the coin margined futures Continuous Contract Kline/Candlestick Data
func (e *Exchange) GetContinuousKlineData(ctx context.Context, req *CFuturesContinuousKlineRequest) ([]CFuturesCandleStick, error) {
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
	var resp []CFuturesCandleStick
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/continuousKlines", params), cFuturesKlineLimit(req.Limit), &resp)
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

// FuturesExchangeInfo returns the coin margined futures Exchange Information
func (e *Exchange) FuturesExchangeInfo(ctx context.Context) (*CFuturesExchangeInfoResponse, error) {
	var resp *CFuturesExchangeInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, "/dapi/v1/exchangeInfo", cFuturesDefaultRate, &resp)
}

// FuturesGetFundingHistory returns the coin margined futures Funding Rate History of Perpetual Futures; delivery
// symbols return no records
func (e *Exchange) FuturesGetFundingHistory(ctx context.Context, req *CFuturesFundingRateHistoryRequest) ([]CFuturesFundingRateHistory, error) {
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
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []CFuturesFundingRateHistory
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/fundingRate", params), cFuturesDefaultRate, &resp)
}

// GetFundingRateInfo returns the coin margined futures Funding Rate Info, which only lists symbols whose funding rate
// cap, floor or interval was adjusted
func (e *Exchange) GetFundingRateInfo(ctx context.Context) ([]CFuturesFundingRateInfo, error) {
	var resp []CFuturesFundingRateInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, "/dapi/v1/fundingInfo", cFuturesDefaultRate, &resp)
}

// GetIndexAndMarkPrice returns the coin margined futures Index Price and Mark Price of a symbol, of every symbol of a
// pair such as BTCUSD, or of every symbol when both are empty. Binance returns every symbol whatever the pair, so the
// pair is applied to the reply
func (e *Exchange) GetIndexAndMarkPrice(ctx context.Context, symbol currency.Pair, pair currency.Code) ([]IndexMarkPrice, error) {
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
	var resp []IndexMarkPrice
	if err := e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/premiumIndex", params), cFuturesIndexMarkPriceRate, &resp); err != nil {
		return nil, err
	}
	if !pair.IsEmpty() {
		resp = slices.DeleteFunc(resp, func(m IndexMarkPrice) bool { return !strings.EqualFold(m.Pair, pair.String()) })
	}
	return resp, nil
}

// GetIndexPriceKlines returns the coin margined futures Index Price Kline/Candlestick Data
func (e *Exchange) GetIndexPriceKlines(ctx context.Context, req *CFuturesIndexPriceKlineRequest) ([]CFuturesPriceCandleStick, error) {
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
	var resp []CFuturesPriceCandleStick
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/indexPriceKlines", params), cFuturesKlineLimit(req.Limit), &resp)
}

// GetFuturesKlineData returns the coin margined futures Kline/Candlestick Data
func (e *Exchange) GetFuturesKlineData(ctx context.Context, req *CFuturesKlineRequest) ([]CFuturesCandleStick, error) {
	params, err := e.symbolKlineParams(req)
	if err != nil {
		return nil, err
	}
	var resp []CFuturesCandleStick
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/klines", params), cFuturesKlineLimit(req.Limit), &resp)
}

// symbolKlineParams builds the parameters of the coin margined futures kline endpoints that take a symbol
func (e *Exchange) symbolKlineParams(req *CFuturesKlineRequest) (url.Values, error) {
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

// GetMarketRatio returns the coin margined futures Long/Short Ratio of every account
func (e *Exchange) GetMarketRatio(ctx context.Context, req *CFuturesStatisticsRequest) ([]TopTraderAccountRatio, error) {
	params, err := e.futuresStatisticsParams(req, false)
	if err != nil {
		return nil, err
	}
	var resp []TopTraderAccountRatio
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/futures/data/globalLongShortAccountRatio", params), futuresDataRate, &resp)
}

// GetMarkPriceKline returns the coin margined futures Mark Price Kline/Candlestick Data
func (e *Exchange) GetMarkPriceKline(ctx context.Context, req *CFuturesKlineRequest) ([]CFuturesPriceCandleStick, error) {
	params, err := e.symbolKlineParams(req)
	if err != nil {
		return nil, err
	}
	var resp []CFuturesPriceCandleStick
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/markPriceKlines", params), cFuturesKlineLimit(req.Limit), &resp)
}

// GetFuturesHistoricalTrades returns coin margined futures trades older than the recent trades list, through Old
// Trades Lookup, which requires an API key
func (e *Exchange) GetFuturesHistoricalTrades(ctx context.Context, symbol currency.Pair, fromID, limit uint64) ([]FuturesPublicTradesData, error) {
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
	var resp []FuturesPublicTradesData
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, common.EncodeURLValues("/dapi/v1/historicalTrades", params), cFuturesHistoricalTradesRate, &resp)
}

// OpenInterest returns the coin margined futures Open Interest of a symbol
func (e *Exchange) OpenInterest(ctx context.Context, symbol currency.Pair) (*CFuturesOpenInterestResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	s, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", s)
	var resp *CFuturesOpenInterestResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/openInterest", params), cFuturesDefaultRate, &resp)
}

// GetOpenInterestStats returns the coin margined futures Open Interest Statistics; an empty contract type aggregates
// every contract type
func (e *Exchange) GetOpenInterestStats(ctx context.Context, req *CFuturesStatisticsRequest) ([]OpenInterestStats, error) {
	params, err := e.futuresStatisticsParams(req, false)
	if err != nil {
		return nil, err
	}
	var resp []OpenInterestStats
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/futures/data/openInterestHist", params), futuresDataRate, &resp)
}

// GetFuturesOrderbook returns the coin margined futures Order Book of a symbol
func (e *Exchange) GetFuturesOrderbook(ctx context.Context, symbol currency.Pair, limit uint64) (*CFuturesOrderBookResponse, error) {
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
	var resp *CFuturesOrderBookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/depth", params), cFuturesOrderbookLimit(limit), &resp)
}

// GetPremiumIndexKlineData returns the coin margined futures Premium index Kline Data
func (e *Exchange) GetPremiumIndexKlineData(ctx context.Context, req *CFuturesKlineRequest) ([]CFuturesPremiumIndexCandleStick, error) {
	params, err := e.symbolKlineParams(req)
	if err != nil {
		return nil, err
	}
	var resp []CFuturesPremiumIndexCandleStick
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/premiumIndexKlines", params), cFuturesKlineLimit(req.Limit), &resp)
}

// GetCFuturesIndexPriceConstituents returns the coin margined futures Index Price Constituents of a pair such as
// BTCUSD, which the API calls the symbol
func (e *Exchange) GetCFuturesIndexPriceConstituents(ctx context.Context, pair currency.Code) (*CFuturesIndexPriceConstituentsResponse, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", pair.Upper().String())
	var resp *CFuturesIndexPriceConstituentsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/constituents", params), cFuturesDefaultRate, &resp)
}

// GetFuturesPublicTrades returns the coin margined futures Recent Trades List
func (e *Exchange) GetFuturesPublicTrades(ctx context.Context, symbol currency.Pair, limit uint64) ([]FuturesPublicTradesData, error) {
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
	var resp []FuturesPublicTradesData
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/trades", params), cFuturesRecentTradesRate, &resp)
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

// GetFuturesOrderbookTicker returns the coin margined futures Symbol Order Book Ticker of a symbol, of every symbol of
// a pair, or of every symbol when both are empty
func (e *Exchange) GetFuturesOrderbookTicker(ctx context.Context, symbol currency.Pair, pair currency.Code) ([]SymbolOrderBookTicker, error) {
	params, err := e.symbolOrPairParams(symbol, pair)
	if err != nil {
		return nil, err
	}
	rateLimit := cFuturesOrderbookTickerAllRate
	if !symbol.IsEmpty() {
		rateLimit = cFuturesBookTickerRate
	}
	var resp []SymbolOrderBookTicker
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/ticker/bookTicker", params), rateLimit, &resp)
}

// GetFuturesSymbolPriceTicker returns the coin margined futures Symbol Price Ticker of a symbol, of every symbol of a
// pair, or of every symbol when both are empty
func (e *Exchange) GetFuturesSymbolPriceTicker(ctx context.Context, symbol currency.Pair, pair currency.Code) ([]SymbolPriceTicker, error) {
	params, err := e.symbolOrPairParams(symbol, pair)
	if err != nil {
		return nil, err
	}
	rateLimit := cFuturesSymbolPriceAllRate
	if !symbol.IsEmpty() {
		rateLimit = cFuturesDefaultRate
	}
	var resp []SymbolPriceTicker
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/ticker/price", params), rateLimit, &resp)
}

// GetFuturesTakerVolume returns the coin margined futures Taker Buy/Sell Volume
func (e *Exchange) GetFuturesTakerVolume(ctx context.Context, req *CFuturesStatisticsRequest) ([]TakerBuySellVolume, error) {
	params, err := e.futuresStatisticsParams(req, true)
	if err != nil {
		return nil, err
	}
	var resp []TakerBuySellVolume
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/futures/data/takerBuySellVol", params), futuresDataRate, &resp)
}

// GetFuturesSwapTickerChangeStats returns the coin margined futures 24hr Ticker Price Change Statistics of a symbol,
// of every symbol of a pair, or of every symbol when both are empty
func (e *Exchange) GetFuturesSwapTickerChangeStats(ctx context.Context, symbol currency.Pair, pair currency.Code) ([]CFuturesPriceChangeStats, error) {
	params, err := e.symbolOrPairParams(symbol, pair)
	if err != nil {
		return nil, err
	}
	rateLimit := cFuturesTicker24HourAllRate
	if !symbol.IsEmpty() {
		rateLimit = cFuturesDefaultRate
	}
	var resp []CFuturesPriceChangeStats
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/dapi/v1/ticker/24hr", params), rateLimit, &resp)
}

// GetTraderFuturesAccountRatio returns the coin margined futures Top Trader Long/Short Account Ratio
func (e *Exchange) GetTraderFuturesAccountRatio(ctx context.Context, req *CFuturesStatisticsRequest) ([]TopTraderAccountRatio, error) {
	params, err := e.futuresStatisticsParams(req, false)
	if err != nil {
		return nil, err
	}
	var resp []TopTraderAccountRatio
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/futures/data/topLongShortAccountRatio", params), futuresDataRate, &resp)
}

// GetTraderFuturesPositionsRatio returns the coin margined futures Top Trader Long/Short Position Ratio
func (e *Exchange) GetTraderFuturesPositionsRatio(ctx context.Context, req *CFuturesStatisticsRequest) ([]TopTraderPositionRatio, error) {
	params, err := e.futuresStatisticsParams(req, false)
	if err != nil {
		return nil, err
	}
	var resp []TopTraderPositionRatio
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/futures/data/topLongShortPositionRatio", params), futuresDataRate, &resp)
}

// FuturesTradeHistory returns the coin margined futures Account Trade List of a symbol or pair
func (e *Exchange) FuturesTradeHistory(ctx context.Context, req *CFuturesAccountTradeListRequest) ([]FuturesAccountTradeList, error) {
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
	var resp []FuturesAccountTradeList
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/userTrades", params, cFuturesAccountTradeListRate, &resp)
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

// GetAllFuturesOrders returns All Orders of a coin margined futures symbol or pair, whether active, cancelled or
// filled
func (e *Exchange) GetAllFuturesOrders(ctx context.Context, req *CFuturesAllOrdersRequest) ([]FuturesOrderData, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if err := checkSymbolOrPair(req.Symbol, req.Pair, req.OrderID); err != nil {
		return nil, err
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
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []FuturesOrderData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/allOrders", params, cFuturesAllOrdersRate, &resp)
}

// AutoCancelAllOpenOrders sets the coin margined futures Auto-Cancel All Open Orders countdown of a symbol, after
// which its open orders are cancelled unless the countdown is renewed; a zero countdown cancels the timer
func (e *Exchange) AutoCancelAllOpenOrders(ctx context.Context, symbol currency.Pair, countdown time.Duration) (*CFuturesAutoCancelAllOpenOrdersResponse, error) {
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
	var resp *CFuturesAutoCancelAllOpenOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, "/dapi/v1/countdownCancelAll", params, cFuturesCountdownCancelRate, &resp)
}

// FuturesCancelAllOpenOrders cancels all coin margined futures open orders of a symbol through Cancel All Open Orders
func (e *Exchange) FuturesCancelAllOpenOrders(ctx context.Context, symbol currency.Pair) (*CFuturesStatusResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	s, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", s)
	var resp *CFuturesStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodDelete, "/dapi/v1/allOpenOrders", params, cFuturesDefaultRate, &resp)
}

// FuturesModifyMultipleOrders modifies up to five coin margined futures limit orders through Modify Multiple Orders,
// which takes no price match; an entry Binance rejects carries its error code and message instead of the order
func (e *Exchange) FuturesModifyMultipleOrders(ctx context.Context, orders []CFuturesModifyOrderRequest) ([]CFuturesModifyBatchOrderResponse, error) {
	if len(orders) == 0 {
		return nil, common.ErrEmptyParams
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
	var resp []CFuturesModifyBatchOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPut, "/dapi/v1/batchOrders", params, cFuturesBatchOrdersRate, &resp)
}

// FuturesBatchOrder places up to five coin margined futures orders through Place Multiple Orders; an entry Binance
// rejects carries its error code and message instead of the order. Binance is moving stop, take profit and trailing
// stop orders to its algo order service, after which they are rejected with -4120
func (e *Exchange) FuturesBatchOrder(ctx context.Context, orders []FuturesNewOrderRequest) ([]CFuturesBatchOrderData, error) {
	if len(orders) == 0 {
		return nil, common.ErrEmptyParams
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
	var resp []CFuturesBatchOrderData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, "/dapi/v1/batchOrders", params, cFuturesBatchOrdersRate, &resp)
}

// FuturesBatchCancelOrders cancels coin margined futures orders of a symbol by order ID or client order ID through
// Cancel Multiple Orders; an entry Binance rejects carries its error code and message instead of the order
func (e *Exchange) FuturesBatchCancelOrders(ctx context.Context, symbol currency.Pair, orderIDs []uint64, origClientOrderIDs []string) ([]CFuturesBatchOrderData, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if len(orderIDs) == 0 && len(origClientOrderIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
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
	var resp []CFuturesBatchOrderData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodDelete, "/dapi/v1/batchOrders", params, cFuturesDefaultRate, &resp)
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

// FuturesGetOrderData returns a coin margined futures order through Query Order
func (e *Exchange) FuturesGetOrderData(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*FuturesOrderDetailResponse, error) {
	params, err := e.orderIdentityParams(symbol, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *FuturesOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/order", params, cFuturesDefaultRate, &resp)
}

// FuturesModifyOrder modifies a coin margined futures limit order's price and quantity through Modify Order, which
// moves the order to the back of the matching queue
func (e *Exchange) FuturesModifyOrder(ctx context.Context, req *CFuturesModifyOrderRequest) (*CFuturesModifyOrderResponse, error) {
	params, err := e.futuresModifyOrderParams(req)
	if err != nil {
		return nil, err
	}
	var resp *CFuturesModifyOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPut, "/dapi/v1/order", params, cFuturesOrdersDefaultRate, &resp)
}

// futuresModifyOrderParams converts a modify request into the parameters Modify Order and Modify Multiple Orders share
func (e *Exchange) futuresModifyOrderParams(req *CFuturesModifyOrderRequest) (url.Values, error) {
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

// FuturesNewOrder places a coin margined futures order through New Order. Binance is moving stop, take profit and
// trailing stop orders to its algo order service, after which they are rejected with -4120
func (e *Exchange) FuturesNewOrder(ctx context.Context, req *FuturesNewOrderRequest) (*FuturesOrderResponse, error) {
	params, err := e.futuresOrderParams(req)
	if err != nil {
		return nil, err
	}
	var resp *FuturesOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, "/dapi/v1/order", params, cFuturesOrdersDefaultRate, &resp)
}

// futuresOrderParams builds the parameters of a coin margined futures order, which New Order sends as they are and
// Place Multiple Orders sends as one JSON object per order
func (e *Exchange) futuresOrderParams(req *FuturesNewOrderRequest) (url.Values, error) {
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
	if req.NewOrderRespType != "" {
		params.Set("newOrderRespType", req.NewOrderRespType)
	}
	if req.PriceMatch != "" {
		params.Set("priceMatch", req.PriceMatch)
	}
	if req.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", req.SelfTradePreventionMode)
	}
	return params, nil
}

// FuturesCancelOrder cancels a coin margined futures order through Cancel Order
func (e *Exchange) FuturesCancelOrder(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*FuturesOrderResponse, error) {
	params, err := e.orderIdentityParams(symbol, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *FuturesOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodDelete, "/dapi/v1/order", params, cFuturesDefaultRate, &resp)
}

// FuturesChangeInitialLeverage sets the initial leverage of a coin margined futures symbol through Change Initial
// Leverage
func (e *Exchange) FuturesChangeInitialLeverage(ctx context.Context, symbol currency.Pair, leverage uint64) (*CFuturesChangeLeverageResponse, error) {
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
	var resp *CFuturesChangeLeverageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, "/dapi/v1/leverage", params, cFuturesDefaultRate, &resp)
}

// FuturesChangeMarginType sets the margin type, ISOLATED or CROSSED, of a coin margined futures symbol through Change
// Margin Type
func (e *Exchange) FuturesChangeMarginType(ctx context.Context, symbol currency.Pair, marginType string) (*CFuturesStatusResponse, error) {
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
	var resp *CFuturesStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, "/dapi/v1/marginType", params, cFuturesDefaultRate, &resp)
}

// GetFuturesAllOpenOrders returns the coin margined futures Current All Open Orders of a symbol, of every symbol of a
// pair, or of every symbol when both are empty
func (e *Exchange) GetFuturesAllOpenOrders(ctx context.Context, symbol currency.Pair, pair currency.Code) ([]FuturesOrderDetailResponse, error) {
	params := url.Values{}
	rateLimit := cFuturesGetAllOpenOrdersRate
	if !symbol.IsEmpty() {
		rateLimit = cFuturesDefaultRate
		s, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", s)
	}
	if !pair.IsEmpty() {
		params.Set("pair", pair.Upper().String())
	}
	var resp []FuturesOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/openOrders", params, rateLimit, &resp)
}

// GetFuturesOrderModifyHistory returns a coin margined futures order's modifications through Get Order Modify History;
// only the last three months are kept
func (e *Exchange) GetFuturesOrderModifyHistory(ctx context.Context, req *CFuturesOrderModifyHistoryRequest) ([]FuturesOrderAmendment, error) {
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
	var resp []FuturesOrderAmendment
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/orderAmendment", params, cFuturesDefaultRate, &resp)
}

// FuturesMarginChangeHistory returns the coin margined futures Position Margin Change History of a symbol
func (e *Exchange) FuturesMarginChangeHistory(ctx context.Context, req *CFuturesPositionMarginHistoryRequest) ([]GetPositionMarginChangeHistoryData, error) {
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
	var resp []GetPositionMarginChangeHistoryData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/positionMargin/history", params, cFuturesDefaultRate, &resp)
}

// ModifyIsolatedPositionMargin adds margin to or removes margin from a coin margined futures isolated position through
// Modify Isolated Position Margin
func (e *Exchange) ModifyIsolatedPositionMargin(ctx context.Context, req *CFuturesModifyIsolatedPositionMarginRequest) (*FuturesMarginUpdatedResponse, error) {
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
	var resp *FuturesMarginUpdatedResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, "/dapi/v1/positionMargin", params, cFuturesDefaultRate, &resp)
}

// FuturesPositionsADLEstimate returns the coin margined futures Position ADL Quantile Estimation of a symbol, or of
// every symbol when it is empty
func (e *Exchange) FuturesPositionsADLEstimate(ctx context.Context, symbol currency.Pair) ([]ADLEstimateData, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		s, err := e.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", s)
	}
	var resp []ADLEstimateData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/adlQuantile", params, cFuturesADLQuantileRate, &resp)
}

// FuturesPositionsInfo returns the coin margined futures Position Information of a margin asset or a pair such as
// BTCUSD, or of every trading symbol when both are empty
func (e *Exchange) FuturesPositionsInfo(ctx context.Context, marginAsset, pair currency.Code) ([]FuturesPositionInformation, error) {
	params := url.Values{}
	if !marginAsset.IsEmpty() {
		params.Set("marginAsset", marginAsset.Upper().String())
	}
	if !pair.IsEmpty() {
		params.Set("pair", pair.Upper().String())
	}
	var resp []FuturesPositionInformation
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/positionRisk", params, cFuturesDefaultRate, &resp)
}

// FuturesOpenOrderData returns an open coin margined futures order through Query Current Open Order
func (e *Exchange) FuturesOpenOrderData(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*FuturesOrderDetailResponse, error) {
	params, err := e.orderIdentityParams(symbol, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *FuturesOrderDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/openOrder", params, cFuturesDefaultRate, &resp)
}

// FuturesForceOrders returns the coin margined futures orders Binance placed to liquidate or deleverage the account's
// positions, through User's Force Orders
func (e *Exchange) FuturesForceOrders(ctx context.Context, req *CFuturesForceOrdersRequest) ([]ForcedOrdersData, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	rateLimit := cFuturesAllForceOrdersRate
	if !req.Symbol.IsEmpty() {
		rateLimit = cFuturesSymbolForceOrdersRate
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
	var resp []ForcedOrdersData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/dapi/v1/forceOrders", params, rateLimit, &resp)
}

// CFuturesQuarterlyContractSettlementPrice returns a COIN-M pair's quarterly contract settlement prices. Binance only
// documents the endpoint for USDⓈ-M futures, but the COIN-M host serves its own pairs (BTCUSD) from the same path
func (e *Exchange) CFuturesQuarterlyContractSettlementPrice(ctx context.Context, pair currency.Code) ([]SettlementPrice, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("pair", pair.Upper().String())
	var resp []SettlementPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestCoinMargined, common.EncodeURLValues("/futures/data/delivery-price", params), futuresDataRate, &resp)
}

// KeepaliveCFuturesUserDataStream extends the validity of the account's COIN-M user data stream listen key by 60
// minutes. A -1125 error means the key has expired and StartCFuturesUserDataStream must create a new one
func (e *Exchange) KeepaliveCFuturesUserDataStream(ctx context.Context) (*FuturesListenKeyResponse, error) {
	var resp *FuturesListenKeyResponse
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPut, "/dapi/v1/listenKey", cFuturesDefaultRate, &resp)
}

// StartCFuturesUserDataStream starts a COIN-M user data stream. An account with an active listen key gets that key
// back with its validity extended by 60 minutes
func (e *Exchange) StartCFuturesUserDataStream(ctx context.Context) (*FuturesListenKeyResponse, error) {
	var resp *FuturesListenKeyResponse
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, "/dapi/v1/listenKey", cFuturesDefaultRate, &resp)
}

// CloseCFuturesUserDataStream closes the account's COIN-M user data stream and invalidates its listen key
func (e *Exchange) CloseCFuturesUserDataStream(ctx context.Context) error {
	return e.SendAPIKeyHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodDelete, "/dapi/v1/listenKey", cFuturesDefaultRate, nil)
}
