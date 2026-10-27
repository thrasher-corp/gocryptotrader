package binance

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"sync"
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
	errPeriodRequired            = errors.New("period is required")
	errClosePositionWithQuantity = errors.New("close position cannot be sent with quantity or reduce only")
	errFromIDWithTimeWindow      = errors.New("from ID cannot be sent with a start or end time")
	errInvalidLeverage           = errors.New("leverage must be a whole number of at least 1")
	errMarginAmountRequired      = errors.New("margin amount must be greater than zero")
	errInvalidMarginAmount       = errors.New("margin amount must be a finite number")
	errBatchModifyReduceOnly     = errors.New("reduce only is not accepted by Modify Multiple Orders")
)

// USDTMarginedFuturesAccountInformationV2 returns the account's balances, positions and configuration from Account
// Information V2
func (e *Exchange) USDTMarginedFuturesAccountInformationV2(ctx context.Context) (*USDTMarginedFuturesAccountInformationV2Response, error) {
	var resp *USDTMarginedFuturesAccountInformationV2Response
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v2/account", nil, usdtMarginedFuturesAccountInformationRate, &resp)
}

// USDTMarginedFuturesAccountInformationV3 returns the account's balances and the positions of symbols with a position
// or open orders from Account Information V3
func (e *Exchange) USDTMarginedFuturesAccountInformationV3(ctx context.Context) (*USDTMarginedFuturesAccountInformationV3Response, error) {
	var resp *USDTMarginedFuturesAccountInformationV3Response
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v3/account", nil, usdtMarginedFuturesAccountInformationRate, &resp)
}

// USDTMarginedFuturesAccountBalanceV3 returns the account's asset balances from Futures Account Balance V3
func (e *Exchange) USDTMarginedFuturesAccountBalanceV3(ctx context.Context) ([]USDTMarginedFuturesAccountBalanceResponse, error) {
	var resp []USDTMarginedFuturesAccountBalanceResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v3/balance", nil, usdtMarginedFuturesAccountInformationRate, &resp)
}

// USDTMarginedFuturesTradingQuantitativeRulesIndicators returns the account's Futures Trading Quantitative Rules
// indicators from Futures Trading Quantitative Rules Indicators, for one symbol or, with an empty symbol, every symbol
func (e *Exchange) USDTMarginedFuturesTradingQuantitativeRulesIndicators(ctx context.Context, symbol currency.Pair) (*TradingQuantitativeRulesIndicatorsResponse, error) {
	params := url.Values{}
	rateLimit := usdtMarginedFuturesTradingStatusAllRate
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		rateLimit = usdtMarginedFuturesDefaultRate
	}
	var resp *TradingQuantitativeRulesIndicatorsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/apiTradingStatus", params, rateLimit, &resp)
}

// GetUSDTMarginedFuturesAssetsMode returns whether the account is in Multi-Assets Mode, from Get Current Multi-Assets
// Mode
func (e *Exchange) GetUSDTMarginedFuturesAssetsMode(ctx context.Context) (bool, error) {
	var resp *USDTMarginedFuturesMultiAssetsModeResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/multiAssetsMargin", nil, usdtMarginedFuturesMultiAssetMarginRate, &resp); err != nil {
		return false, err
	}
	return resp.MultiAssetsMargin, nil
}

// GetUSDTMarginedFuturesCurrentPositionMode returns whether the account is in Hedge Mode, from Get Current Position
// Mode
func (e *Exchange) GetUSDTMarginedFuturesCurrentPositionMode(ctx context.Context) (*USDTMarginedFuturesPositionModeResponse, error) {
	return e.getCurrentPositionMode(ctx, usdtMarginedFuturesPositionModeRate)
}

// getCurrentPositionMode fetches the position mode drawing on f, so that the lookup a Long or Short order makes can
// draw on the cancel pool rather than queue behind order book snapshots on the IP pool
func (e *Exchange) getCurrentPositionMode(ctx context.Context, f request.EndpointLimit) (*USDTMarginedFuturesPositionModeResponse, error) {
	var resp *USDTMarginedFuturesPositionModeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/positionSide/dual", nil, f, &resp)
}

// positionModeCache holds the position mode of each account Long and Short futures orders were placed for, keyed by
// API key since context credentials can switch the account per call; USDⓈ-M and COIN-M have shared one mode since
// the CM-UM integration. A lookup keeps the mode it fetched only when the account's mode was not forgotten while it was
// in flight, so a lookup overlapping a change of mode cannot keep the mode the change replaced
type positionModeCache struct {
	mu          sync.Mutex
	hedgeMode   map[string]bool
	generations map[string]uint64
}

// get returns the account's cached mode, whether it has one, and the generation a lookup started now keeps its mode at
func (c *positionModeCache) get(key string) (hedgeMode, ok bool, generation uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	hedgeMode, ok = c.hedgeMode[key]
	return hedgeMode, ok, c.generations[key]
}

// set keeps the mode a lookup started at generation fetched, unless the account's mode was forgotten since
func (c *positionModeCache) set(key string, generation uint64, hedgeMode bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.generations[key] != generation {
		return
	}
	if c.hedgeMode == nil {
		c.hedgeMode = make(map[string]bool)
	}
	c.hedgeMode[key] = hedgeMode
}

// forget drops the account's cached mode and the mode of any lookup in flight
func (c *positionModeCache) forget(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.hedgeMode, key)
	if c.generations == nil {
		c.generations = make(map[string]uint64)
	}
	c.generations[key]++
}

// positionMode returns whether the account of ctx's credentials is in Hedge Mode, from the cache or else from Get
// Current Position Mode on the cancel pool when it can take the lookup at once, so a Long or Short order does not wait
// behind order book snapshots
func (e *Exchange) positionMode(ctx context.Context) (bool, error) {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return false, err
	}
	hedgeMode, ok, generation := e.positionModes.get(creds.Key)
	if ok {
		return hedgeMode, nil
	}
	var resp *USDTMarginedFuturesPositionModeResponse
	if err := e.sendOnShare(ctx, usdtMarginedFuturesPositionModeLookupRate, usdtMarginedFuturesPositionModeRate, func(ctx context.Context, f request.EndpointLimit) (err error) {
		resp, err = e.getCurrentPositionMode(ctx, f)
		return err
	}); err != nil {
		return false, err
	}
	e.positionModes.set(creds.Key, generation, resp.DualSidePosition)
	return resp.DualSidePosition, nil
}

// forgetPositionMode drops the cached position mode of ctx's account, so the next Long or Short order looks it up
func (e *Exchange) forgetPositionMode(ctx context.Context) {
	if creds, err := e.GetCredentials(ctx); err == nil {
		e.positionModes.forget(creds.Key)
	}
}

// USDTMarginedFuturesOrderHistoryDownloadID requests an order history file for a window of up to a year, from Get
// Download Id For Futures Order History
func (e *Exchange) USDTMarginedFuturesOrderHistoryDownloadID(ctx context.Context, startTime, endTime time.Time) (*USDTMarginedFuturesDownloadIDResponse, error) {
	return e.usdtMarginedFuturesDownloadID(ctx, "/fapi/v1/order/asyn", startTime, endTime)
}

// USDTMarginedFuturesTradeHistoryDownloadID requests a trade history file for a window of up to a year, from Get
// Download Id For Futures Trade History
func (e *Exchange) USDTMarginedFuturesTradeHistoryDownloadID(ctx context.Context, startTime, endTime time.Time) (*USDTMarginedFuturesDownloadIDResponse, error) {
	return e.usdtMarginedFuturesDownloadID(ctx, "/fapi/v1/trade/asyn", startTime, endTime)
}

// USDTMarginedFuturesTransactionHistoryDownloadID requests a transaction history file for a window of up to a year,
// from Get Download Id For Futures Transaction History
func (e *Exchange) USDTMarginedFuturesTransactionHistoryDownloadID(ctx context.Context, startTime, endTime time.Time) (*USDTMarginedFuturesDownloadIDResponse, error) {
	return e.usdtMarginedFuturesDownloadID(ctx, "/fapi/v1/income/asyn", startTime, endTime)
}

// usdtMarginedFuturesDownloadID requests one of the history files, whose endpoints share their parameters and reply
func (e *Exchange) usdtMarginedFuturesDownloadID(ctx context.Context, path string, startTime, endTime time.Time) (*USDTMarginedFuturesDownloadIDResponse, error) {
	if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	var resp *USDTMarginedFuturesDownloadIDResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, path, params, usdtMarginedFuturesDownloadIDRate, &resp)
}

// USDTMarginedFuturesOrderHistoryDownloadLink returns an order history file's download link, from Get Futures Order
// History Download Link by Id
func (e *Exchange) USDTMarginedFuturesOrderHistoryDownloadLink(ctx context.Context, downloadID string) (*USDTMarginedFuturesDownloadLinkResponse, error) {
	return e.usdtMarginedFuturesDownloadLink(ctx, "/fapi/v1/order/asyn/id", downloadID)
}

// USDTMarginedFuturesTradeHistoryDownloadLink returns a trade history file's download link, from Get Futures Trade
// Download Link by Id
func (e *Exchange) USDTMarginedFuturesTradeHistoryDownloadLink(ctx context.Context, downloadID string) (*USDTMarginedFuturesDownloadLinkResponse, error) {
	return e.usdtMarginedFuturesDownloadLink(ctx, "/fapi/v1/trade/asyn/id", downloadID)
}

// USDTMarginedFuturesTransactionHistoryDownloadLink returns a transaction history file's download link, from Get
// Futures Transaction History Download Link by Id
func (e *Exchange) USDTMarginedFuturesTransactionHistoryDownloadLink(ctx context.Context, downloadID string) (*USDTMarginedFuturesDownloadLinkResponse, error) {
	return e.usdtMarginedFuturesDownloadLink(ctx, "/fapi/v1/income/asyn/id", downloadID)
}

// usdtMarginedFuturesDownloadLink returns one of the history files' download links, whose endpoints share their
// parameters and reply
func (e *Exchange) usdtMarginedFuturesDownloadLink(ctx context.Context, path, downloadID string) (*USDTMarginedFuturesDownloadLinkResponse, error) {
	if downloadID == "" {
		return nil, errDownloadIDRequired
	}
	params := url.Values{}
	params.Set("downloadId", downloadID)
	var resp *USDTMarginedFuturesDownloadLinkResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, path, params, usdtMarginedFuturesDownloadLinkRate, &resp)
}

// USDTMarginedFuturesAccountIncomeHistory returns the account's income records from Get Income History; without a time
// window the last seven days are returned, and only the last three months are kept
func (e *Exchange) USDTMarginedFuturesAccountIncomeHistory(ctx context.Context, arg *USDTMarginedFuturesIncomeHistoryRequest) ([]USDTMarginedFuturesAccountIncomeHistoryResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if !arg.Symbol.IsEmpty() {
		symbol, err := e.FormatSymbol(arg.Symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbol)
	}
	if arg.IncomeType != "" {
		params.Set("incomeType", arg.IncomeType)
	}
	if arg.Page > 0 {
		params.Set("page", strconv.FormatUint(arg.Page, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []USDTMarginedFuturesAccountIncomeHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/income", params, usdtMarginedFuturesIncomeHistoryRate, &resp)
}

// USDTMarginedFuturesGetNotionalAndLeverageBrackets returns the notional and leverage brackets of one symbol or, with
// an empty symbol, every symbol, from Notional and Leverage Brackets
func (e *Exchange) USDTMarginedFuturesGetNotionalAndLeverageBrackets(ctx context.Context, symbol currency.Pair) ([]USDTMarginedFuturesNotionalLeverageAndBracketsResponse, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp objectOrArray[USDTMarginedFuturesNotionalLeverageAndBracketsResponse]
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/leverageBracket", params, usdtMarginedFuturesDefaultRate, &resp)
}

// GetUSDTMarginedFuturesUserRateLimits returns the account's order rate limits from Query User Rate Limit
func (e *Exchange) GetUSDTMarginedFuturesUserRateLimits(ctx context.Context) ([]RateLimitInfoResponse, error) {
	var resp []RateLimitInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/rateLimit/order", nil, usdtMarginedFuturesDefaultRate, &resp)
}

// USDTMarginedFuturesGetCommissionRates returns the account's commission rates for a symbol from User Commission Rate
func (e *Exchange) USDTMarginedFuturesGetCommissionRates(ctx context.Context, symbol currency.Pair) (*USDTMarginedFuturesCommissionRateResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	var resp *USDTMarginedFuturesCommissionRateResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/commissionRate", params, usdtMarginedFuturesUserCommissionRate, &resp)
}

// GetUSDTMarginedFuturesBasis returns the basis between a pair's index price and a contract's price from Basis; only
// the last 30 days are kept
func (e *Exchange) GetUSDTMarginedFuturesBasis(ctx context.Context, arg *USDTMarginedFuturesBasisRequest) ([]BasisInfoResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.ContractType == "" {
		return nil, errContractTypeIsRequired
	}
	if arg.Period == "" {
		return nil, errPeriodRequired
	}
	params := url.Values{}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	pair, err := e.FormatSymbol(arg.Pair, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params.Set("pair", pair)
	params.Set("contractType", arg.ContractType)
	params.Set("period", arg.Period)
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []BasisInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/futures/data/basis", params), futuresDataRate, &resp)
}

// USDTMarginedFuturesServerTime returns the server time from Check Server Time
func (e *Exchange) USDTMarginedFuturesServerTime(ctx context.Context) (time.Time, error) {
	var resp *USDTMarginedFuturesServerTimeResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, "/fapi/v1/time", usdtMarginedFuturesDefaultRate, &resp); err != nil {
		return time.Time{}, err
	}
	return resp.ServerTime.Time(), nil
}

// USDTMarginedFuturesCompositeIndexInfo returns the components of one composite index symbol or, with an empty symbol,
// every one, from Composite Index Symbol Information
func (e *Exchange) USDTMarginedFuturesCompositeIndexInfo(ctx context.Context, symbol currency.Pair) ([]USDTMarginedFuturesCompositeIndexResponse, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp objectOrArray[USDTMarginedFuturesCompositeIndexResponse]
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/indexInfo", params), usdtMarginedFuturesDefaultRate, &resp)
}

// USDTMarginedFuturesCompressedTrades returns aggregate market trades from Compressed/Aggregate Trades List; only the
// last 48 hours can be queried and a time window must be shorter than an hour
func (e *Exchange) USDTMarginedFuturesCompressedTrades(ctx context.Context, arg *USDTMarginedFuturesCompressedTradesRequest) ([]USDTMarginedFuturesCompressedTradeResponse, error) {
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
	symbol, err := e.FormatSymbol(arg.Symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	if arg.FromID > 0 {
		params.Set("fromId", strconv.FormatUint(arg.FromID, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []USDTMarginedFuturesCompressedTradeResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/aggTrades", params), usdtMarginedFuturesAggregateTradesRate, &resp)
}

// GetUSDTMarginedFuturesContinuousKlineData returns a contract type's klines from Continuous Contract Kline/Candlestick
// Data
func (e *Exchange) GetUSDTMarginedFuturesContinuousKlineData(ctx context.Context, arg *USDTMarginedFuturesContinuousKlineRequest) ([]USDTMarginedFuturesKlineResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.ContractType == "" {
		return nil, errContractTypeIsRequired
	}
	pair, err := e.FormatSymbol(arg.Pair, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("pair", pair)
	params.Set("contractType", arg.ContractType)
	var resp []USDTMarginedFuturesKlineResponse
	return resp, e.usdtMarginedFuturesKlineData(ctx, "/fapi/v1/continuousKlines", params, arg.Interval, arg.StartTime, arg.EndTime, arg.Limit, &resp)
}

// USDTMarginedFuturesExchangeInfo returns the trading rules and symbols from Exchange Information
func (e *Exchange) USDTMarginedFuturesExchangeInfo(ctx context.Context) (*USDTMarginedFuturesExchangeInfoResponse, error) {
	var resp *USDTMarginedFuturesExchangeInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, "/fapi/v1/exchangeInfo", usdtMarginedFuturesDefaultRate, &resp)
}

// USDTMarginedFuturesGetFundingHistory returns funding rate records from Get Funding Rate History, oldest first;
// without a time window the latest 200 are returned
func (e *Exchange) USDTMarginedFuturesGetFundingHistory(ctx context.Context, symbol currency.Pair, startTime, endTime time.Time, limit uint64) ([]FundingRateHistoryResponse, error) {
	params := url.Values{}
	if err := setTimeRangeParams(params, startTime, endTime); err != nil {
		return nil, err
	}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []FundingRateHistoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/fundingRate", params), usdtMarginedFuturesFundingRateHistoryRate, &resp)
}

// USDTMarginedFuturesGetFundingRateInfo returns the symbols whose funding rate cap, floor or interval was adjusted,
// from Get Funding Rate Info
func (e *Exchange) USDTMarginedFuturesGetFundingRateInfo(ctx context.Context) ([]FundingRateInfoResponse, error) {
	var resp []FundingRateInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, "/fapi/v1/fundingInfo", usdtMarginedFuturesFundingInfoRate, &resp)
}

// GetUSDTMarginedFuturesIndexPriceKlineCandlesticks returns a pair's index price klines from Index Price
// Kline/Candlestick Data
func (e *Exchange) GetUSDTMarginedFuturesIndexPriceKlineCandlesticks(ctx context.Context, arg *USDTMarginedFuturesIndexPriceKlineRequest) ([]USDTMarginedFuturesPriceKlineResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	pair, err := e.FormatSymbol(arg.Pair, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("pair", pair)
	var resp []USDTMarginedFuturesPriceKlineResponse
	return resp, e.usdtMarginedFuturesKlineData(ctx, "/fapi/v1/indexPriceKlines", params, arg.Interval, arg.StartTime, arg.EndTime, arg.Limit, &resp)
}

// USDTMarginedFuturesKlineData returns a symbol's klines from Kline/Candlestick Data
func (e *Exchange) USDTMarginedFuturesKlineData(ctx context.Context, arg *USDTMarginedFuturesKlineRequest) ([]USDTMarginedFuturesKlineResponse, error) {
	var resp []USDTMarginedFuturesKlineResponse
	return resp, e.usdtMarginedFuturesSymbolKlineData(ctx, "/fapi/v1/klines", arg, &resp)
}

// usdtMarginedFuturesSymbolKlineData sends one of the kline requests that identify the instrument by symbol
func (e *Exchange) usdtMarginedFuturesSymbolKlineData(ctx context.Context, path string, arg *USDTMarginedFuturesKlineRequest, result any) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.Symbol.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	symbol, err := e.FormatSymbol(arg.Symbol, asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	return e.usdtMarginedFuturesKlineData(ctx, path, params, arg.Interval, arg.StartTime, arg.EndTime, arg.Limit, result)
}

// usdtMarginedFuturesKlineData validates and sends a kline request whose params already identify the instrument
func (e *Exchange) usdtMarginedFuturesKlineData(ctx context.Context, path string, params url.Values, interval string, startTime, endTime time.Time, limit uint64, result any) error {
	if interval == "" {
		return kline.ErrInvalidInterval
	}
	if err := setTimeRangeParams(params, startTime, endTime); err != nil {
		return err
	}
	params.Set("interval", interval)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	return e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(path, params), usdtMarginedFuturesKlineLimit(limit), result)
}

// USDTMarginedFuturesGlobalLongShortRatio returns the long/short account ratio of all traders from Long/Short Ratio;
// only the last 30 days are kept
func (e *Exchange) USDTMarginedFuturesGlobalLongShortRatio(ctx context.Context, arg *USDTMarginedFuturesDataRequest) ([]USDTMarginedFuturesLongShortRatioResponse, error) {
	var resp []USDTMarginedFuturesLongShortRatioResponse
	return resp, e.usdtMarginedFuturesData(ctx, "/futures/data/globalLongShortAccountRatio", arg, &resp)
}

// USDTMarginedFuturesGetMarkPrice returns the mark price and funding rate of one symbol or, with an empty symbol, every
// symbol, from Mark Price
func (e *Exchange) USDTMarginedFuturesGetMarkPrice(ctx context.Context, symbol currency.Pair) ([]USDTMarginedFuturesMarkPriceResponse, error) {
	params := url.Values{}
	rateLimit := usdtMarginedFuturesMarkPriceAllRate
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		rateLimit = usdtMarginedFuturesDefaultRate
	}
	var resp objectOrArray[USDTMarginedFuturesMarkPriceResponse]
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/premiumIndex", params), rateLimit, &resp)
}

// GetUSDTMarginedFuturesMarkPriceKlineCandlesticks returns a symbol's mark price klines from Mark Price
// Kline/Candlestick Data
func (e *Exchange) GetUSDTMarginedFuturesMarkPriceKlineCandlesticks(ctx context.Context, arg *USDTMarginedFuturesKlineRequest) ([]USDTMarginedFuturesPriceKlineResponse, error) {
	var resp []USDTMarginedFuturesPriceKlineResponse
	return resp, e.usdtMarginedFuturesSymbolKlineData(ctx, "/fapi/v1/markPriceKlines", arg, &resp)
}

// GetUSDTMarginedFuturesMultiAssetModeAssetIndex returns the index of one asset pair or, with an empty symbol, every
// asset, from Asset Index (formerly Multi-Assets Mode Asset Index), which since the CM-UM integration also lists the
// COIN-M settlement assets
func (e *Exchange) GetUSDTMarginedFuturesMultiAssetModeAssetIndex(ctx context.Context, symbol currency.Pair) ([]AssetIndexResponse, error) {
	params := url.Values{}
	rateLimit := usdtMarginedFuturesAssetIndexAllRate
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		rateLimit = usdtMarginedFuturesDefaultRate
	}
	var resp objectOrArray[AssetIndexResponse]
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/assetIndex", params), rateLimit, &resp)
}

// USDTMarginedFuturesHistoricalTrades returns older market trades from Old Trades Lookup, starting at a trade ID or,
// when fromID is 0, with the most recent; only the last month is kept
func (e *Exchange) USDTMarginedFuturesHistoricalTrades(ctx context.Context, symbol currency.Pair, fromID, limit uint64) ([]USDTMarginedFuturesPublicTradesResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if fromID > 0 {
		params.Set("fromId", strconv.FormatUint(fromID, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []USDTMarginedFuturesPublicTradesResponse
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, common.EncodeURLValues("/fapi/v1/historicalTrades", params), usdtMarginedFuturesHistoricalTradesRate, &resp)
}

// USDTMarginedFuturesOpenInterest returns a symbol's present open interest from Open Interest
func (e *Exchange) USDTMarginedFuturesOpenInterest(ctx context.Context, symbol currency.Pair) (*USDTMarginedFuturesOpenInterestResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	var resp *USDTMarginedFuturesOpenInterestResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/openInterest", params), usdtMarginedFuturesDefaultRate, &resp)
}

// USDTMarginedFuturesOpenInterestStats returns a symbol's open interest history from Open Interest Statistics; only the
// last month is kept
func (e *Exchange) USDTMarginedFuturesOpenInterestStats(ctx context.Context, arg *USDTMarginedFuturesDataRequest) ([]USDTMarginedFuturesOpenInterestStatsResponse, error) {
	var resp []USDTMarginedFuturesOpenInterestStatsResponse
	return resp, e.usdtMarginedFuturesData(ctx, "/futures/data/openInterestHist", arg, &resp)
}

// USDTMarginedFuturesOrderbook returns a symbol's order book, which excludes retail price improvement orders, from
// Order Book. Valid limits are 5, 10, 20, 50, 100, 500 and 1000; 0 means the default of 500
func (e *Exchange) USDTMarginedFuturesOrderbook(ctx context.Context, symbol currency.Pair, limit uint64) (*USDTMarginedFuturesOrderBookResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp *USDTMarginedFuturesOrderBookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/depth", params), usdtMarginedFuturesOrderbookLimit(limit), &resp)
}

// GetUSDTMarginedFuturesPremiumIndexKlineCandlesticks returns a symbol's premium index klines from Premium Index Kline
// Data
func (e *Exchange) GetUSDTMarginedFuturesPremiumIndexKlineCandlesticks(ctx context.Context, arg *USDTMarginedFuturesKlineRequest) ([]USDTMarginedFuturesPriceKlineResponse, error) {
	var resp []USDTMarginedFuturesPriceKlineResponse
	return resp, e.usdtMarginedFuturesSymbolKlineData(ctx, "/fapi/v1/premiumIndexKlines", arg, &resp)
}

// GetUSDTMarginedFuturesQuarterlyContractSettlementPrice returns a pair's quarterly contract settlement prices from
// Quarterly Contract Settlement Price
func (e *Exchange) GetUSDTMarginedFuturesQuarterlyContractSettlementPrice(ctx context.Context, pair currency.Pair) ([]SettlementPriceResponse, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	pairValue, err := e.FormatSymbol(pair, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("pair", pairValue)
	var resp []SettlementPriceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/futures/data/delivery-price", params), futuresDataRate, &resp)
}

// GetUSDTMarginedFuturesIndexPriceConstituents returns the constituents of a symbol's index price from Query Index
// Price Constituents
func (e *Exchange) GetUSDTMarginedFuturesIndexPriceConstituents(ctx context.Context, symbol currency.Pair) (*USDTMarginedFuturesIndexPriceConstituentsResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	var resp *USDTMarginedFuturesIndexPriceConstituentsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/constituents", params), usdtMarginedFuturesIndexConstituentsRate, &resp)
}

// USDTMarginedFuturesRecentTrades returns a symbol's recent market trades from Recent Trades List
func (e *Exchange) USDTMarginedFuturesRecentTrades(ctx context.Context, symbol currency.Pair, limit uint64) ([]USDTMarginedFuturesPublicTradesResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp []USDTMarginedFuturesPublicTradesResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/trades", params), usdtMarginedFuturesRecentTradesRate, &resp)
}

// GetUSDTMarginedFuturesRetailPriceImprovementOrderbook returns a symbol's order book including retail price
// improvement orders, from RPI Order Book. The only valid limit is 1000, which 0 also means
func (e *Exchange) GetUSDTMarginedFuturesRetailPriceImprovementOrderbook(ctx context.Context, symbol currency.Pair, limit uint64) (*USDTMarginedFuturesOrderBookResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if limit > 0 {
		params.Set("limit", strconv.FormatUint(limit, 10))
	}
	var resp *USDTMarginedFuturesOrderBookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/rpiDepth", params), usdtMarginedFuturesRetailPriceImprovementOrderbookRate, &resp)
}

// USDTMarginedFuturesSymbolOrderbookTicker returns the best bid and ask of one symbol or, with an empty symbol, every
// symbol, from Symbol Order Book Ticker
func (e *Exchange) USDTMarginedFuturesSymbolOrderbookTicker(ctx context.Context, symbol currency.Pair) ([]USDTMarginedFuturesSymbolOrderbookTickerResponse, error) {
	params := url.Values{}
	rateLimit := usdtMarginedFuturesOrderbookTickerAllRate
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		rateLimit = usdtMarginedFuturesBookTickerRate
	}
	var resp objectOrArray[USDTMarginedFuturesSymbolOrderbookTickerResponse]
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/ticker/bookTicker", params), rateLimit, &resp)
}

// USDTMarginedFuturesSymbolPriceTickerV2 returns the latest price of one symbol or, with an empty symbol, every symbol,
// from Symbol Price Ticker V2
func (e *Exchange) USDTMarginedFuturesSymbolPriceTickerV2(ctx context.Context, symbol currency.Pair) ([]USDTMarginedFuturesSymbolPriceTickerResponse, error) {
	params := url.Values{}
	rateLimit := usdtMarginedFuturesSymbolPriceAllRate
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		rateLimit = usdtMarginedFuturesDefaultRate
	}
	var resp objectOrArray[USDTMarginedFuturesSymbolPriceTickerResponse]
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v2/ticker/price", params), rateLimit, &resp)
}

// USDTMarginedFuturesTakerBuySellVolume returns a symbol's taker buy and sell volumes from Taker Buy/Sell Volume; only
// the last 30 days are kept
func (e *Exchange) USDTMarginedFuturesTakerBuySellVolume(ctx context.Context, arg *USDTMarginedFuturesDataRequest) ([]USDTMarginedFuturesTakerVolumeResponse, error) {
	var resp []USDTMarginedFuturesTakerVolumeResponse
	return resp, e.usdtMarginedFuturesData(ctx, "/futures/data/takerlongshortRatio", arg, &resp)
}

// USDTMarginedFutures24HourTickerPriceChangeStats returns the 24 hour price change statistics of one symbol or, with an
// empty symbol, every symbol, from 24hr Ticker Price Change Statistics
func (e *Exchange) USDTMarginedFutures24HourTickerPriceChangeStats(ctx context.Context, symbol currency.Pair) ([]USDTMarginedFutures24HourPriceChangeStatsResponse, error) {
	params := url.Values{}
	rateLimit := usdtMarginedFuturesTicker24HourAllRate
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		rateLimit = usdtMarginedFuturesDefaultRate
	}
	var resp objectOrArray[USDTMarginedFutures24HourPriceChangeStatsResponse]
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/ticker/24hr", params), rateLimit, &resp)
}

// USDTMarginedFuturesTopAccountsLongShortRatio returns the long/short account ratio of the top 20% of traders by margin
// balance, from Top Trader Long/Short Account Ratio; only the last 30 days are kept
func (e *Exchange) USDTMarginedFuturesTopAccountsLongShortRatio(ctx context.Context, arg *USDTMarginedFuturesDataRequest) ([]USDTMarginedFuturesLongShortRatioResponse, error) {
	var resp []USDTMarginedFuturesLongShortRatioResponse
	return resp, e.usdtMarginedFuturesData(ctx, "/futures/data/topLongShortAccountRatio", arg, &resp)
}

// USDTMarginedFuturesTopPositionsLongShortRatio returns the long/short position ratio of the top 20% of traders by
// margin balance, from Top Trader Long/Short Position Ratio; only the last 30 days are kept
func (e *Exchange) USDTMarginedFuturesTopPositionsLongShortRatio(ctx context.Context, arg *USDTMarginedFuturesDataRequest) ([]USDTMarginedFuturesLongShortRatioResponse, error) {
	var resp []USDTMarginedFuturesLongShortRatioResponse
	return resp, e.usdtMarginedFuturesData(ctx, "/futures/data/topLongShortPositionRatio", arg, &resp)
}

// usdtMarginedFuturesData sends one of the /futures/data statistics requests, which share their parameters. Binance
// documents the two top trader ratios as MARKET_DATA, but they answer without an API key, so they stay public
func (e *Exchange) usdtMarginedFuturesData(ctx context.Context, path string, arg *USDTMarginedFuturesDataRequest, result any) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.Symbol.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if arg.Period == "" {
		return errPeriodRequired
	}
	params := url.Values{}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return err
	}
	symbol, err := e.FormatSymbol(arg.Symbol, asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	params.Set("symbol", symbol)
	params.Set("period", arg.Period)
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	return e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(path, params), futuresDataRate, result)
}

// USDTMarginedFuturesSetAssetsMode switches the account between Multi-Assets Mode (true) and Single-Asset Mode (false)
// on every symbol, with Change Multi-Assets Mode
func (e *Exchange) USDTMarginedFuturesSetAssetsMode(ctx context.Context, multiAssetsMargin bool) error {
	params := url.Values{}
	params.Set("multiAssetsMargin", strconv.FormatBool(multiAssetsMargin))
	return e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/multiAssetsMargin", params, usdtMarginedFuturesDefaultRate, nil)
}

// USDTMarginedFuturesChangePositionMode switches the account between Hedge Mode (true) and One-way Mode (false) on
// every symbol, with Change Position Mode. Since the CM-UM integration the switch applies to COIN-M too, and is
// rejected while either has open orders or positions
func (e *Exchange) USDTMarginedFuturesChangePositionMode(ctx context.Context, dualSidePosition bool) error {
	params := url.Values{}
	params.Set("dualSidePosition", strconv.FormatBool(dualSidePosition))
	// A lookup made while the change is in flight may read either mode, so the account's mode is looked up again after it
	e.forgetPositionMode(ctx)
	defer e.forgetPositionMode(ctx)
	return e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/positionSide/dual", params, usdtMarginedFuturesDefaultRate, nil)
}

// USDTMarginedFuturesAccountTradesHistory returns the account's trades in a symbol from Account Trade List, which may
// also list COIN-M trades. A time window may span at most seven days, and only the last three months are kept
func (e *Exchange) USDTMarginedFuturesAccountTradesHistory(ctx context.Context, arg *USDTMarginedFuturesAccountTradeListRequest) ([]USDTMarginedFuturesAccountTradeHistoryResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.FromID > 0 && (!arg.StartTime.IsZero() || !arg.EndTime.IsZero()) {
		return nil, errFromIDWithTimeWindow
	}
	params := url.Values{}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	symbol, err := e.FormatSymbol(arg.Symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	if arg.OrderID > 0 {
		params.Set("orderId", strconv.FormatUint(arg.OrderID, 10))
	}
	if arg.FromID > 0 {
		params.Set("fromId", strconv.FormatUint(arg.FromID, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []USDTMarginedFuturesAccountTradeHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/userTrades", params, usdtMarginedFuturesAccountTradeListRate, &resp)
}

// USDTMarginedFuturesAllAccountOrders returns the account's active, cancelled and filled orders from All Orders, which
// may also list COIN-M orders. A time window may span at most seven days, and cancelled or expired orders without fills
// drop out after three days
func (e *Exchange) USDTMarginedFuturesAllAccountOrders(ctx context.Context, arg *USDTMarginedFuturesAllOrdersRequest) ([]USDTMarginedFuturesAllOrdersResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if !arg.Symbol.IsEmpty() {
		symbol, err := e.FormatSymbol(arg.Symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbol)
	}
	if arg.OrderID > 0 {
		params.Set("orderId", strconv.FormatUint(arg.OrderID, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []USDTMarginedFuturesAllOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/allOrders", params, usdtMarginedFuturesGetAllOrdersRate, &resp)
}

// USDTMarginedFuturesAutoCancelAllOpenOrders cancels a symbol's open orders once the countdown elapses, with
// Auto-Cancel All Open Orders. Calling it again replaces the countdown, so it serves as a heartbeat, and a zero
// countdown stops it. Heartbeats draw on a rate limit share of their own, which COIN-M heartbeats share, while it can
// take them at once
func (e *Exchange) USDTMarginedFuturesAutoCancelAllOpenOrders(ctx context.Context, symbol currency.Pair, countdownTime time.Duration) (*USDTMarginedFuturesAutoCancelAllOpenOrdersResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	params.Set("countdownTime", strconv.FormatInt(countdownTime.Milliseconds(), 10))
	var resp *USDTMarginedFuturesAutoCancelAllOpenOrdersResponse
	return resp, e.sendOnShare(ctx, usdtMarginedFuturesCountdownCancelRate, usdtMarginedFuturesHeartbeatOverflowRate, func(ctx context.Context, f request.EndpointLimit) error {
		return e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/countdownCancelAll", params, f, &resp)
	})
}

// USDTMarginedFuturesQueryAlgoOrder returns an algo (conditional) order's state from Query Algo Order, by its algo ID
// or, when algoID is 0, its client algo ID
func (e *Exchange) USDTMarginedFuturesQueryAlgoOrder(ctx context.Context, algoID uint64, clientAlgoID string) (*USDTMarginedFuturesAlgoOrderResponse, error) {
	return e.usdtMarginedFuturesQueryAlgoOrder(ctx, algoID, clientAlgoID, usdtMarginedFuturesDefaultRate)
}

// usdtMarginedFuturesQueryAlgoOrder queries an algo order drawing on f, so that the lookup a cancellation makes can
// draw on the cancel pool as the cancellation does, rather than queue behind order book snapshots on the IP pool
func (e *Exchange) usdtMarginedFuturesQueryAlgoOrder(ctx context.Context, algoID uint64, clientAlgoID string, f request.EndpointLimit) (*USDTMarginedFuturesAlgoOrderResponse, error) {
	params, err := algoOrderIDParams(algoID, clientAlgoID)
	if err != nil {
		return nil, err
	}
	var resp *USDTMarginedFuturesAlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/algoOrder", params, f, &resp)
}

// USDTMarginedFuturesNewAlgoOrder places an algo (conditional) order, which is how stop, take profit and trailing stop
// orders are placed since Binance moved them off New Order, with New Algo Order
func (e *Exchange) USDTMarginedFuturesNewAlgoOrder(ctx context.Context, arg *USDTMarginedFuturesNewAlgoOrderRequest) (*USDTMarginedFuturesNewAlgoOrderResponse, error) {
	params, err := e.newAlgoOrderParams(arg, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	var resp *USDTMarginedFuturesNewAlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/algoOrder", params, usdtMarginedFuturesOrdersDefaultRate, &resp)
}

// newAlgoOrderParams builds the parameters of a New Algo Order for asset a, which Binance takes alike on the USDⓈ-M and
// COIN-M hosts
func (e *Exchange) newAlgoOrderParams(arg *USDTMarginedFuturesNewAlgoOrderRequest, a asset.Item) (url.Values, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.AlgoType == "" {
		return nil, errAlgoTypeRequired
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if arg.PriceMatch != "" && arg.Price != 0 {
		return nil, errPriceMatchWithPrice
	}
	if arg.ClosePosition && (arg.Quantity != 0 || arg.ReduceOnly) {
		return nil, errClosePositionWithQuantity
	}
	symbol, err := e.FormatSymbol(arg.Symbol, a)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("algoType", arg.AlgoType)
	params.Set("symbol", symbol)
	params.Set("side", arg.Side)
	params.Set("type", arg.OrderType)
	if arg.PositionSide != "" {
		params.Set("positionSide", arg.PositionSide)
	}
	if arg.TimeInForce != "" {
		params.Set("timeInForce", arg.TimeInForce)
	}
	if arg.Quantity != 0 {
		params.Set("quantity", strconv.FormatFloat(arg.Quantity, 'f', -1, 64))
	}
	if arg.Price != 0 {
		params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	}
	if arg.TriggerPrice != 0 {
		params.Set("triggerPrice", strconv.FormatFloat(arg.TriggerPrice, 'f', -1, 64))
	}
	if arg.WorkingType != "" {
		params.Set("workingType", arg.WorkingType)
	}
	if arg.PriceMatch != "" {
		params.Set("priceMatch", arg.PriceMatch)
	}
	if arg.ClosePosition {
		params.Set("closePosition", "true")
	}
	if arg.PriceProtect {
		params.Set("priceProtect", "true")
	}
	if arg.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	if arg.ActivatePrice != 0 {
		params.Set("activatePrice", strconv.FormatFloat(arg.ActivatePrice, 'f', -1, 64))
	}
	if arg.CallbackRate != 0 {
		params.Set("callbackRate", strconv.FormatFloat(arg.CallbackRate, 'f', -1, 64))
	}
	if arg.ClientAlgoID != "" {
		params.Set("clientAlgoId", arg.ClientAlgoID)
	}
	if arg.NewOrderResponseType != "" {
		params.Set("newOrderRespType", arg.NewOrderResponseType)
	}
	if arg.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", arg.SelfTradePreventionMode)
	}
	if !arg.GoodTillDate.IsZero() {
		params.Set("goodTillDate", strconv.FormatInt(arg.GoodTillDate.UnixMilli(), 10))
	}
	return params, nil
}

// USDTMarginedFuturesCancelAlgoOrder cancels an active algo (conditional) order with Cancel Algo Order, by its algo ID
// or, when algoID is 0, its client algo ID
func (e *Exchange) USDTMarginedFuturesCancelAlgoOrder(ctx context.Context, algoID uint64, clientAlgoID string) (*USDTMarginedFuturesCancelAlgoOrderResponse, error) {
	params, err := algoOrderIDParams(algoID, clientAlgoID)
	if err != nil {
		return nil, err
	}
	var resp *USDTMarginedFuturesCancelAlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodDelete, "/fapi/v1/algoOrder", params, usdtMarginedFuturesCancelRate, &resp)
}

// algoOrderIDParams returns the parameters identifying one algo order, which Binance requires by either ID
func algoOrderIDParams(algoID uint64, clientAlgoID string) (url.Values, error) {
	params := url.Values{}
	switch {
	case algoID > 0:
		params.Set("algoId", strconv.FormatUint(algoID, 10))
	case clientAlgoID != "":
		params.Set("clientAlgoId", clientAlgoID)
	default:
		return nil, order.ErrOrderIDNotSet
	}
	return params, nil
}

// USDTMarginedFuturesCancelAllAlgoOpenOrders cancels a symbol's open algo (conditional) orders with Cancel All Algo
// Open Orders
func (e *Exchange) USDTMarginedFuturesCancelAllAlgoOpenOrders(ctx context.Context, symbol currency.Pair) error {
	if symbol.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	return e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodDelete, "/fapi/v1/algoOpenOrders", params, usdtMarginedFuturesCancelRate, nil)
}

// USDTMarginedFuturesCancelAllOpenOrders cancels a symbol's open orders with Cancel All Open Orders
func (e *Exchange) USDTMarginedFuturesCancelAllOpenOrders(ctx context.Context, symbol currency.Pair) error {
	if symbol.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	return e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodDelete, "/fapi/v1/allOpenOrders", params, usdtMarginedFuturesCancelRate, nil)
}

// USDTMarginedFuturesModifyMultipleOrders modifies up to five limit orders with Modify Multiple Orders, which processes
// them concurrently and replies in request order, each element holding the modified order or the error that rejected it
func (e *Exchange) USDTMarginedFuturesModifyMultipleOrders(ctx context.Context, args []USDTMarginedFuturesModifyOrderRequest) ([]USDTMarginedFuturesModifyBatchOrderResponse, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	if err := checkListLength("orders", len(args), batchOrdersLimit); err != nil {
		return nil, err
	}
	orders := make([]map[string]string, len(args))
	for i := range args {
		if args[i].ReduceOnly {
			return nil, errBatchModifyReduceOnly
		}
		params, err := e.usdtMarginedFuturesModifyOrderParams(&args[i])
		if err != nil {
			return nil, err
		}
		orders[i] = usdtMarginedFuturesBatchItem(params)
	}
	batch, err := json.Marshal(orders)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("batchOrders", string(batch))
	var resp []USDTMarginedFuturesModifyBatchOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPut, "/fapi/v1/batchOrders", params, usdtMarginedFuturesBatchOrdersRate, &resp)
}

// USDTMarginedFuturesPlaceBatchOrders places up to five limit or market orders with Place Multiple Orders, which
// processes them concurrently and replies in request order, each element holding the new order or the error that
// rejected it
func (e *Exchange) USDTMarginedFuturesPlaceBatchOrders(ctx context.Context, args []USDTMarginedFuturesNewOrderRequest) ([]USDTMarginedFuturesPlaceBatchOrderResponse, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	if err := checkListLength("orders", len(args), batchOrdersLimit); err != nil {
		return nil, err
	}
	orders := make([]map[string]string, len(args))
	for i := range args {
		params, err := e.usdtMarginedFuturesNewOrderParams(&args[i])
		if err != nil {
			return nil, err
		}
		orders[i] = usdtMarginedFuturesBatchItem(params)
	}
	batch, err := json.Marshal(orders)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("batchOrders", string(batch))
	var resp []USDTMarginedFuturesPlaceBatchOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/batchOrders", params, usdtMarginedFuturesBatchOrdersRate, &resp)
}

// usdtMarginedFuturesBatchItem converts one order's parameters into a batchOrders element. Values stay strings, as in
// Binance's examples, and the JSON encoder sorts the keys, so a request always encodes the same way
func usdtMarginedFuturesBatchItem(params url.Values) map[string]string {
	item := make(map[string]string, len(params))
	for k := range params {
		item[k] = params.Get(k)
	}
	return item
}

// USDTMarginedFuturesCancelBatchOrders cancels up to ten of a symbol's orders by order ID or client order ID with
// Cancel Multiple Orders, each reply element holding the cancelled order or the error that rejected it
func (e *Exchange) USDTMarginedFuturesCancelBatchOrders(ctx context.Context, symbol currency.Pair, orderIDs []uint64, origClientOrderIDs []string) ([]USDTMarginedFuturesCancelBatchOrderResponse, error) {
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
	symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if len(orderIDs) != 0 {
		ids, err := json.Marshal(orderIDs)
		if err != nil {
			return nil, err
		}
		params.Set("orderIdList", string(ids))
	}
	if len(origClientOrderIDs) != 0 {
		ids, err := json.Marshal(origClientOrderIDs)
		if err != nil {
			return nil, err
		}
		params.Set("origClientOrderIdList", string(ids))
	}
	var resp []USDTMarginedFuturesCancelBatchOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodDelete, "/fapi/v1/batchOrders", params, usdtMarginedFuturesCancelRate, &resp)
}

// USDTMarginedFuturesGetOrderData returns an order's state from Query Order, by its order ID or, when orderID is 0, its
// client order ID. Cancelled or expired orders without fills drop out after three days, and every order after 90 days
func (e *Exchange) USDTMarginedFuturesGetOrderData(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*USDTMarginedFuturesOrderResponse, error) {
	params, err := e.usdtMarginedFuturesOrderIDParams(symbol, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *USDTMarginedFuturesOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/order", params, usdtMarginedFuturesDefaultRate, &resp)
}

// usdtMarginedFuturesOrderIDParams returns the parameters identifying one of a symbol's orders, which Binance requires
// by either ID
func (e *Exchange) usdtMarginedFuturesOrderIDParams(symbol currency.Pair, orderID uint64, origClientOrderID string) (url.Values, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if orderID == 0 && origClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if orderID > 0 {
		params.Set("orderId", strconv.FormatUint(orderID, 10))
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	return params, nil
}

// USDTMarginedFuturesModifyOrder modifies a limit order's price and quantity with Modify Order, which moves the order
// to the back of the matching queue
func (e *Exchange) USDTMarginedFuturesModifyOrder(ctx context.Context, arg *USDTMarginedFuturesModifyOrderRequest) (*USDTMarginedFuturesModifyOrderResponse, error) {
	params, err := e.usdtMarginedFuturesModifyOrderParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *USDTMarginedFuturesModifyOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPut, "/fapi/v1/order", params, usdtMarginedFuturesOrdersDefaultRate, &resp)
}

// usdtMarginedFuturesModifyOrderParams converts a modify request into its parameters, which Modify Order and Modify
// Multiple Orders share
func (e *Exchange) usdtMarginedFuturesModifyOrderParams(arg *USDTMarginedFuturesModifyOrderRequest) (url.Values, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.OrderID == 0 && arg.OrigClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Quantity <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.PriceMatch != "" && arg.Price != 0 {
		return nil, errPriceMatchWithPrice
	}
	if arg.PriceMatch == "" && arg.Price <= 0 {
		return nil, limits.ErrPriceBelowMin
	}
	symbol, err := e.FormatSymbol(arg.Symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	if arg.OrderID > 0 {
		params.Set("orderId", strconv.FormatUint(arg.OrderID, 10))
	}
	if arg.OrigClientOrderID != "" {
		params.Set("origClientOrderId", arg.OrigClientOrderID)
	}
	params.Set("symbol", symbol)
	params.Set("side", arg.Side)
	params.Set("quantity", strconv.FormatFloat(arg.Quantity, 'f', -1, 64))
	if arg.Price != 0 {
		params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	}
	if arg.PriceMatch != "" {
		params.Set("priceMatch", arg.PriceMatch)
	}
	if arg.ModifyID > 0 {
		params.Set("modifyId", strconv.FormatUint(arg.ModifyID, 10))
	}
	if arg.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	return params, nil
}

// USDTMarginedFuturesNewOrder places a limit or market order with New Order; stop, take profit and trailing stop orders
// go through USDTMarginedFuturesNewAlgoOrder
func (e *Exchange) USDTMarginedFuturesNewOrder(ctx context.Context, arg *USDTMarginedFuturesNewOrderRequest) (*USDTMarginedFuturesNewOrderResponse, error) {
	params, err := e.usdtMarginedFuturesNewOrderParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *USDTMarginedFuturesNewOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/order", params, usdtMarginedFuturesOrdersDefaultRate, &resp)
}

// usdtMarginedFuturesNewOrderParams converts a new order request into its parameters, which New Order and Place
// Multiple Orders share
func (e *Exchange) usdtMarginedFuturesNewOrderParams(arg *USDTMarginedFuturesNewOrderRequest) (url.Values, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if arg.PriceMatch != "" && arg.Price != 0 {
		return nil, errPriceMatchWithPrice
	}
	symbol, err := e.FormatSymbol(arg.Symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", arg.Side)
	params.Set("type", arg.OrderType)
	if arg.PositionSide != "" {
		params.Set("positionSide", arg.PositionSide)
	}
	if arg.TimeInForce != "" {
		params.Set("timeInForce", arg.TimeInForce)
	}
	if arg.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	if arg.Quantity != 0 {
		params.Set("quantity", strconv.FormatFloat(arg.Quantity, 'f', -1, 64))
	}
	if arg.Price != 0 {
		params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	}
	if arg.NewClientOrderID != "" {
		params.Set("newClientOrderId", arg.NewClientOrderID)
	}
	if arg.NewOrderResponseType != "" {
		params.Set("newOrderRespType", arg.NewOrderResponseType)
	}
	if arg.PriceMatch != "" {
		params.Set("priceMatch", arg.PriceMatch)
	}
	if arg.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", arg.SelfTradePreventionMode)
	}
	if !arg.GoodTillDate.IsZero() {
		params.Set("goodTillDate", strconv.FormatInt(arg.GoodTillDate.UnixMilli(), 10))
	}
	return params, nil
}

// USDTMarginedFuturesCancelOrder cancels an active order with Cancel Order, by its order ID or, when orderID is 0, its
// client order ID
func (e *Exchange) USDTMarginedFuturesCancelOrder(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*USDTMarginedFuturesCancelOrderResponse, error) {
	params, err := e.usdtMarginedFuturesOrderIDParams(symbol, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *USDTMarginedFuturesCancelOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodDelete, "/fapi/v1/order", params, usdtMarginedFuturesCancelRate, &resp)
}

// USDTMarginedFuturesChangeInitialLeverageRequest changes a symbol's initial leverage, 1 to 125, with Change Initial
// Leverage
func (e *Exchange) USDTMarginedFuturesChangeInitialLeverageRequest(ctx context.Context, symbol currency.Pair, leverage uint64) (*USDTMarginedFuturesChangeInitialLeverageResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if leverage == 0 {
		return nil, errInvalidLeverage
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	params.Set("leverage", strconv.FormatUint(leverage, 10))
	var resp *USDTMarginedFuturesChangeInitialLeverageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/leverage", params, usdtMarginedFuturesDefaultRate, &resp)
}

// USDTMarginedFuturesChangeInitialMarginType changes a symbol's margin type to ISOLATED or CROSSED with Change Margin
// Type
func (e *Exchange) USDTMarginedFuturesChangeInitialMarginType(ctx context.Context, symbol currency.Pair, marginType string) error {
	if symbol.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if marginType == "" {
		return margin.ErrInvalidMarginType
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	params.Set("marginType", marginType)
	return e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/marginType", params, usdtMarginedFuturesDefaultRate, nil)
}

// USDTMarginedFuturesCurrentAllAlgoOpenOrders returns open algo (conditional) orders from Current All Algo Open Orders;
// each argument is an optional filter, and an empty symbol returns every symbol's orders
func (e *Exchange) USDTMarginedFuturesCurrentAllAlgoOpenOrders(ctx context.Context, algoType string, symbol currency.Pair, algoID uint64) ([]USDTMarginedFuturesAlgoOrderResponse, error) {
	return e.usdtMarginedFuturesCurrentAllAlgoOpenOrders(ctx, algoType, symbol, algoID, usdtMarginedFuturesOpenAlgoOrdersAllRate)
}

// usdtMarginedFuturesCurrentAllAlgoOpenOrders lists open algo orders, drawing on allSymbols for a listing of every
// symbol's, so the listing a cancellation of every symbol's orders makes can draw on the cancel pool
func (e *Exchange) usdtMarginedFuturesCurrentAllAlgoOpenOrders(ctx context.Context, algoType string, symbol currency.Pair, algoID uint64, allSymbols request.EndpointLimit) ([]USDTMarginedFuturesAlgoOrderResponse, error) {
	params := url.Values{}
	rateLimit := allSymbols
	if algoType != "" {
		params.Set("algoType", algoType)
	}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		rateLimit = usdtMarginedFuturesDefaultRate
	}
	if algoID > 0 {
		params.Set("algoId", strconv.FormatUint(algoID, 10))
	}
	var resp []USDTMarginedFuturesAlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/openAlgoOrders", params, rateLimit, &resp)
}

// USDTMarginedFuturesAllAccountOpenOrders returns the open orders of one symbol or, with an empty symbol, every symbol,
// from Current All Open Orders
func (e *Exchange) USDTMarginedFuturesAllAccountOpenOrders(ctx context.Context, symbol currency.Pair) ([]USDTMarginedFuturesOrderResponse, error) {
	return e.usdtMarginedFuturesAllAccountOpenOrders(ctx, symbol, usdtMarginedFuturesGetAllOpenOrdersRate)
}

// usdtMarginedFuturesAllAccountOpenOrders lists open orders, drawing on allSymbols for a listing of every symbol's, so
// the listing a cancellation of every symbol's orders makes can draw on the cancel pool
func (e *Exchange) usdtMarginedFuturesAllAccountOpenOrders(ctx context.Context, symbol currency.Pair, allSymbols request.EndpointLimit) ([]USDTMarginedFuturesOrderResponse, error) {
	params := url.Values{}
	rateLimit := allSymbols
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		rateLimit = usdtMarginedFuturesDefaultRate
	}
	var resp []USDTMarginedFuturesOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/openOrders", params, rateLimit, &resp)
}

// GetUSDTMarginedFuturesOrderModifyHistory returns an order's modifications from Get Order Modify History; only the
// last three months are kept
func (e *Exchange) GetUSDTMarginedFuturesOrderModifyHistory(ctx context.Context, arg *USDTMarginedFuturesOrderModifyHistoryRequest) ([]FuturesOrderAmendmentResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.usdtMarginedFuturesOrderIDParams(arg.Symbol, arg.OrderID, arg.OrigClientOrderID)
	if err != nil {
		return nil, err
	}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []FuturesOrderAmendmentResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/orderAmendment", params, usdtMarginedFuturesDefaultRate, &resp)
}

// USDTMarginedFuturesPositionMarginChangeHistory returns a symbol's isolated position margin changes from Get Position
// Margin Change History; a time window may span at most 30 days, and only the last 30 days are kept
func (e *Exchange) USDTMarginedFuturesPositionMarginChangeHistory(ctx context.Context, arg *USDTMarginedFuturesPositionMarginChangeHistoryRequest) ([]USDTMarginedFuturesPositionMarginChangeResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if arg.ChangeType != "" {
		changeType, ok := validMarginChange[arg.ChangeType]
		if !ok {
			return nil, errMarginChangeTypeInvalid
		}
		params.Set("type", strconv.FormatInt(changeType, 10))
	}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	symbol, err := e.FormatSymbol(arg.Symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []USDTMarginedFuturesPositionMarginChangeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/positionMargin/history", params, usdtMarginedFuturesDefaultRate, &resp)
}

// USDTMarginedFuturesModifyIsolatedPositionMarginRequest adds margin to or reduces margin from an isolated position
// with Modify Isolated Position Margin; changeType is add or reduce, and positionSide is required in Hedge Mode
func (e *Exchange) USDTMarginedFuturesModifyIsolatedPositionMarginRequest(ctx context.Context, symbol currency.Pair, positionSide, changeType string, amount float64) (*USDTMarginedFuturesModifyIsolatedPositionMarginResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	marginChange, ok := validMarginChange[changeType]
	if !ok {
		return nil, errMarginChangeTypeInvalid
	}
	if amount <= 0 {
		return nil, errMarginAmountRequired
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if positionSide != "" {
		params.Set("positionSide", positionSide)
	}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	params.Set("type", strconv.FormatInt(marginChange, 10))
	var resp *USDTMarginedFuturesModifyIsolatedPositionMarginResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/positionMargin", params, usdtMarginedFuturesDefaultRate, &resp)
}

// USDTMarginedFuturesPositionsAutoDeleveragingEstimate returns the auto-deleveraging quantiles of one symbol's
// positions or, with an empty symbol, every symbol's, from Position ADL Quantile Estimation
func (e *Exchange) USDTMarginedFuturesPositionsAutoDeleveragingEstimate(ctx context.Context, symbol currency.Pair) ([]USDTMarginedFuturesPositionAutoDeleveragingQuantileResponse, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp []USDTMarginedFuturesPositionAutoDeleveragingQuantileResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/adlQuantile", params, usdtMarginedFuturesAutoDeleveragingQuantileRate, &resp)
}

// USDTMarginedFuturesPositionsInfoV2 returns the positions of one symbol or, with an empty symbol, every symbol, from
// Position Information V2
func (e *Exchange) USDTMarginedFuturesPositionsInfoV2(ctx context.Context, symbol currency.Pair) ([]USDTMarginedFuturesPositionInformationV2Response, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp []USDTMarginedFuturesPositionInformationV2Response
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v2/positionRisk", params, usdtMarginedFuturesPositionRiskRate, &resp)
}

// USDTMarginedFuturesPositionsInfoV3 returns the positions of one symbol or, with an empty symbol, every symbol with a
// position or open orders, from Position Information V3
func (e *Exchange) USDTMarginedFuturesPositionsInfoV3(ctx context.Context, symbol currency.Pair) ([]USDTMarginedFuturesPositionInformationV3Response, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp []USDTMarginedFuturesPositionInformationV3Response
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v3/positionRisk", params, usdtMarginedFuturesPositionRiskRate, &resp)
}

// USDTMarginedFuturesQueryAllAlgoOrders returns a symbol's active, cancelled, triggered and finished algo (conditional)
// orders from Query All Algo Orders. A time window may span at most seven days
func (e *Exchange) USDTMarginedFuturesQueryAllAlgoOrders(ctx context.Context, arg *USDTMarginedFuturesAllAlgoOrdersRequest) ([]USDTMarginedFuturesAlgoOrderResponse, error) {
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
	symbol, err := e.FormatSymbol(arg.Symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	if arg.AlgoID > 0 {
		params.Set("algoId", strconv.FormatUint(arg.AlgoID, 10))
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []USDTMarginedFuturesAlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/allAlgoOrders", params, usdtMarginedFuturesAllAlgoOrdersRate, &resp)
}

// USDTMarginedFuturesFetchOpenOrder returns an open order's state from Query Current Open Order, by its order ID or,
// when orderID is 0, its client order ID
func (e *Exchange) USDTMarginedFuturesFetchOpenOrder(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*USDTMarginedFuturesOrderResponse, error) {
	params, err := e.usdtMarginedFuturesOrderIDParams(symbol, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *USDTMarginedFuturesOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/openOrder", params, usdtMarginedFuturesDefaultRate, &resp)
}

// USDTMarginedFuturesAccountForcedOrders returns the account's liquidation and auto-deleveraging orders from User's
// Force Orders, which may also list COIN-M orders; only the last 90 days are kept
func (e *Exchange) USDTMarginedFuturesAccountForcedOrders(ctx context.Context, arg *USDTMarginedFuturesForceOrdersRequest) ([]USDTMarginedFuturesForceOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	rateLimit := usdtMarginedFuturesAllForceOrdersRate
	if !arg.Symbol.IsEmpty() {
		symbol, err := e.FormatSymbol(arg.Symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbol)
		rateLimit = usdtMarginedFuturesSymbolForceOrdersRate
	}
	if arg.AutoCloseType != "" {
		params.Set("autoCloseType", arg.AutoCloseType)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []USDTMarginedFuturesForceOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/forceOrders", params, rateLimit, &resp)
}

// KeepaliveUSDTMarginedFuturesUserDataStream extends the validity of the account's USDⓈ-M user data stream listen key
// by 60 minutes. A -1125 error means the key has expired and StartUSDTMarginedFuturesUserDataStream must create a new
// one
func (e *Exchange) KeepaliveUSDTMarginedFuturesUserDataStream(ctx context.Context) (*FuturesListenKeyResponse, error) {
	var resp *FuturesListenKeyResponse
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPut, "/fapi/v1/listenKey", usdtMarginedFuturesDefaultRate, &resp)
}

// StartUSDTMarginedFuturesUserDataStream starts a USDⓈ-M user data stream. An account with an active listen key gets
// that key back with its validity extended by 60 minutes
func (e *Exchange) StartUSDTMarginedFuturesUserDataStream(ctx context.Context) (*FuturesListenKeyResponse, error) {
	var resp *FuturesListenKeyResponse
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/listenKey", usdtMarginedFuturesDefaultRate, &resp)
}

// CloseUSDTMarginedFuturesUserDataStream closes the account's USDⓈ-M user data stream and invalidates its listen key
func (e *Exchange) CloseUSDTMarginedFuturesUserDataStream(ctx context.Context) error {
	return e.SendAPIKeyHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodDelete, "/fapi/v1/listenKey", usdtMarginedFuturesDefaultRate, nil)
}
