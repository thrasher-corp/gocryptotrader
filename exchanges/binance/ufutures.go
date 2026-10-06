package binance

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
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
	errPeriodRequired            = errors.New("period is required")
	errClosePositionWithQuantity = errors.New("close position cannot be sent with quantity or reduce only")
	errFromIDWithTimeWindow      = errors.New("from ID cannot be sent with a start or end time")
	errInvalidLeverage           = errors.New("leverage must be a whole number of at least 1")
	errMarginAmountRequired      = errors.New("margin amount must be greater than zero")
	errInvalidMarginAmount       = errors.New("margin amount must be a finite number")
	errBatchModifyReduceOnly     = errors.New("reduce only is not accepted by Modify Multiple Orders")
)

// UAccountInformationV2 returns the account's balances, positions and configuration from Account Information V2
func (e *Exchange) UAccountInformationV2(ctx context.Context) (*UAccountInformationV2Response, error) {
	var resp *UAccountInformationV2Response
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v2/account", nil, uFuturesAccountInformationRate, &resp)
}

// UAccountInformationV3 returns the account's balances and the positions of symbols with a position or open orders
// from Account Information V3
func (e *Exchange) UAccountInformationV3(ctx context.Context) (*UAccountInformationV3Response, error) {
	var resp *UAccountInformationV3Response
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v3/account", nil, uFuturesAccountInformationRate, &resp)
}

// UAccountBalanceV3 returns the account's asset balances from Futures Account Balance V3
func (e *Exchange) UAccountBalanceV3(ctx context.Context) ([]UAccountBalance, error) {
	var resp []UAccountBalance
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v3/balance", nil, uFuturesAccountInformationRate, &resp)
}

// UFuturesTradingQuantitativeRulesIndicators returns the account's Futures Trading Quantitative Rules indicators
// from Futures Trading Quantitative Rules Indicators, for one symbol or, with an empty symbol, every symbol
func (e *Exchange) UFuturesTradingQuantitativeRulesIndicators(ctx context.Context, symbol currency.Pair) (*TradingQuantitativeRulesIndicatorsResponse, error) {
	params := url.Values{}
	rateLimit := uFuturesTradingStatusAllRate
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		rateLimit = uFuturesDefaultRate
	}
	var resp *TradingQuantitativeRulesIndicatorsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/apiTradingStatus", params, rateLimit, &resp)
}

// GetAssetsMode returns whether the account is in Multi-Assets Mode, from Get Current Multi-Assets Mode
func (e *Exchange) GetAssetsMode(ctx context.Context) (bool, error) {
	var resp *UMultiAssetsModeResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/multiAssetsMargin", nil, uFuturesMultiAssetMarginRate, &resp); err != nil {
		return false, err
	}
	return resp.MultiAssetsMargin, nil
}

// GetCurrentPositionMode returns whether the account is in Hedge Mode, from Get Current Position Mode
func (e *Exchange) GetCurrentPositionMode(ctx context.Context) (*UPositionModeResponse, error) {
	var resp *UPositionModeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/positionSide/dual", nil, uFuturesPositionModeRate, &resp)
}

// UFuturesOrderHistoryDownloadID requests an order history file for a window of up to a year, from Get Download Id
// For Futures Order History
func (e *Exchange) UFuturesOrderHistoryDownloadID(ctx context.Context, startTime, endTime time.Time) (*UDownloadIDResponse, error) {
	return e.uFuturesDownloadID(ctx, "/fapi/v1/order/asyn", startTime, endTime)
}

// UFuturesTradeHistoryDownloadID requests a trade history file for a window of up to a year, from Get Download Id For
// Futures Trade History
func (e *Exchange) UFuturesTradeHistoryDownloadID(ctx context.Context, startTime, endTime time.Time) (*UDownloadIDResponse, error) {
	return e.uFuturesDownloadID(ctx, "/fapi/v1/trade/asyn", startTime, endTime)
}

// UFuturesTransactionHistoryDownloadID requests a transaction history file for a window of up to a year, from Get
// Download Id For Futures Transaction History
func (e *Exchange) UFuturesTransactionHistoryDownloadID(ctx context.Context, startTime, endTime time.Time) (*UDownloadIDResponse, error) {
	return e.uFuturesDownloadID(ctx, "/fapi/v1/income/asyn", startTime, endTime)
}

// uFuturesDownloadID requests one of the history files, whose endpoints share their parameters and reply
func (e *Exchange) uFuturesDownloadID(ctx context.Context, path string, startTime, endTime time.Time) (*UDownloadIDResponse, error) {
	if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	var resp *UDownloadIDResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, path, params, uFuturesDownloadIDRate, &resp)
}

// UFuturesOrderHistoryDownloadLink returns an order history file's download link, from Get Futures Order History
// Download Link by Id
func (e *Exchange) UFuturesOrderHistoryDownloadLink(ctx context.Context, downloadID string) (*UDownloadLinkResponse, error) {
	return e.uFuturesDownloadLink(ctx, "/fapi/v1/order/asyn/id", downloadID)
}

// UFuturesTradeHistoryDownloadLink returns a trade history file's download link, from Get Futures Trade Download Link
// by Id
func (e *Exchange) UFuturesTradeHistoryDownloadLink(ctx context.Context, downloadID string) (*UDownloadLinkResponse, error) {
	return e.uFuturesDownloadLink(ctx, "/fapi/v1/trade/asyn/id", downloadID)
}

// UFuturesTransactionHistoryDownloadLink returns a transaction history file's download link, from Get Futures
// Transaction History Download Link by Id
func (e *Exchange) UFuturesTransactionHistoryDownloadLink(ctx context.Context, downloadID string) (*UDownloadLinkResponse, error) {
	return e.uFuturesDownloadLink(ctx, "/fapi/v1/income/asyn/id", downloadID)
}

// uFuturesDownloadLink returns one of the history files' download links, whose endpoints share their parameters and
// reply
func (e *Exchange) uFuturesDownloadLink(ctx context.Context, path, downloadID string) (*UDownloadLinkResponse, error) {
	if downloadID == "" {
		return nil, errDownloadIDRequired
	}
	params := url.Values{}
	params.Set("downloadId", downloadID)
	var resp *UDownloadLinkResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, path, params, uFuturesDownloadLinkRate, &resp)
}

// UAccountIncomeHistory returns the account's income records from Get Income History; without a time window the
// last seven days are returned, and only the last three months are kept
func (e *Exchange) UAccountIncomeHistory(ctx context.Context, arg *UIncomeHistoryRequest) ([]UAccountIncomeHistory, error) {
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
	var resp []UAccountIncomeHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/income", params, uFuturesIncomeHistoryRate, &resp)
}

// UGetNotionalAndLeverageBrackets returns the notional and leverage brackets of one symbol or, with an empty symbol,
// every symbol, from Notional and Leverage Brackets
func (e *Exchange) UGetNotionalAndLeverageBrackets(ctx context.Context, symbol currency.Pair) ([]UNotionalLeverageAndBrackets, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp objectOrArray[UNotionalLeverageAndBrackets]
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/leverageBracket", params, uFuturesDefaultRate, &resp)
}

// GetUSDTUserRateLimits returns the account's order rate limits from Query User Rate Limit
func (e *Exchange) GetUSDTUserRateLimits(ctx context.Context) ([]RateLimitInfo, error) {
	var resp []RateLimitInfo
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/rateLimit/order", nil, uFuturesDefaultRate, &resp)
}

// UGetCommissionRates returns the account's commission rates for a symbol from User Commission Rate
func (e *Exchange) UGetCommissionRates(ctx context.Context, symbol currency.Pair) (*UCommissionRateResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	var resp *UCommissionRateResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/commissionRate", params, uFuturesUserCommissionRate, &resp)
}

// GetBasis returns the basis between a pair's index price and a contract's price from Basis; only the last 30 days
// are kept
func (e *Exchange) GetBasis(ctx context.Context, arg *UBasisRequest) ([]BasisInfo, error) {
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
	var resp []BasisInfo
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/futures/data/basis", params), futuresDataRate, &resp)
}

// UServerTime returns the server time from Check Server Time
func (e *Exchange) UServerTime(ctx context.Context) (time.Time, error) {
	var resp *UServerTimeResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, "/fapi/v1/time", uFuturesDefaultRate, &resp); err != nil {
		return time.Time{}, err
	}
	return resp.ServerTime.Time(), nil
}

// UCompositeIndexInfo returns the components of one composite index symbol or, with an empty symbol, every one,
// from Composite Index Symbol Information
func (e *Exchange) UCompositeIndexInfo(ctx context.Context, symbol currency.Pair) ([]UCompositeIndex, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp objectOrArray[UCompositeIndex]
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/indexInfo", params), uFuturesDefaultRate, &resp)
}

// UCompressedTrades returns aggregate market trades from Compressed/Aggregate Trades List; only the last 48 hours can
// be queried and a time window must be shorter than an hour
func (e *Exchange) UCompressedTrades(ctx context.Context, arg *UCompressedTradesRequest) ([]UCompressedTradeData, error) {
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
	var resp []UCompressedTradeData
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/aggTrades", params), uFuturesAggregateTradesRate, &resp)
}

// GetUFuturesContinuousKlineData returns a contract type's klines from Continuous Contract Kline/Candlestick Data
func (e *Exchange) GetUFuturesContinuousKlineData(ctx context.Context, arg *UContinuousKlineRequest) ([]UKline, error) {
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
	var resp []UKline
	return resp, e.uKlineData(ctx, "/fapi/v1/continuousKlines", params, arg.Interval, arg.StartTime, arg.EndTime, arg.Limit, &resp)
}

// UExchangeInfo returns the trading rules and symbols from Exchange Information
func (e *Exchange) UExchangeInfo(ctx context.Context) (*UExchangeInfoResponse, error) {
	var resp *UExchangeInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, "/fapi/v1/exchangeInfo", uFuturesDefaultRate, &resp)
}

// UGetFundingHistory returns funding rate records from Get Funding Rate History, oldest first; without a time window
// the latest 200 are returned
func (e *Exchange) UGetFundingHistory(ctx context.Context, symbol currency.Pair, startTime, endTime time.Time, limit uint64) ([]FundingRateHistory, error) {
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
	var resp []FundingRateHistory
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/fundingRate", params), uFuturesFundingRateHistoryRate, &resp)
}

// UGetFundingRateInfo returns the symbols whose funding rate cap, floor or interval was adjusted, from Get Funding
// Rate Info
func (e *Exchange) UGetFundingRateInfo(ctx context.Context) ([]FundingRateInfoResponse, error) {
	var resp []FundingRateInfoResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, "/fapi/v1/fundingInfo", uFuturesFundingInfoRate, &resp)
}

// GetIndexPriceKlineCandlesticks returns a pair's index price klines from Index Price Kline/Candlestick Data
func (e *Exchange) GetIndexPriceKlineCandlesticks(ctx context.Context, arg *UIndexPriceKlineRequest) ([]UPriceKline, error) {
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
	var resp []UPriceKline
	return resp, e.uKlineData(ctx, "/fapi/v1/indexPriceKlines", params, arg.Interval, arg.StartTime, arg.EndTime, arg.Limit, &resp)
}

// UKlineData returns a symbol's klines from Kline/Candlestick Data
func (e *Exchange) UKlineData(ctx context.Context, arg *UKlineRequest) ([]UKline, error) {
	var resp []UKline
	return resp, e.uSymbolKlineData(ctx, "/fapi/v1/klines", arg, &resp)
}

// uSymbolKlineData sends one of the kline requests that identify the instrument by symbol
func (e *Exchange) uSymbolKlineData(ctx context.Context, path string, arg *UKlineRequest, result any) error {
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
	return e.uKlineData(ctx, path, params, arg.Interval, arg.StartTime, arg.EndTime, arg.Limit, result)
}

// uKlineData validates and sends a kline request whose params already identify the instrument
func (e *Exchange) uKlineData(ctx context.Context, path string, params url.Values, interval string, startTime, endTime time.Time, limit uint64, result any) error {
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
	return e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues(path, params), uFuturesKlineLimit(limit), result)
}

// UGlobalLongShortRatio returns the long/short account ratio of all traders from Long/Short Ratio; only the last 30
// days are kept
func (e *Exchange) UGlobalLongShortRatio(ctx context.Context, arg *UFuturesDataRequest) ([]ULongShortRatio, error) {
	var resp []ULongShortRatio
	return resp, e.uFuturesData(ctx, "/futures/data/globalLongShortAccountRatio", arg, &resp)
}

// UGetMarkPrice returns the mark price and funding rate of one symbol or, with an empty symbol, every symbol, from
// Mark Price
func (e *Exchange) UGetMarkPrice(ctx context.Context, symbol currency.Pair) ([]UMarkPrice, error) {
	params := url.Values{}
	rateLimit := uFuturesMarkPriceAllRate
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		rateLimit = uFuturesDefaultRate
	}
	var resp objectOrArray[UMarkPrice]
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/premiumIndex", params), rateLimit, &resp)
}

// GetMarkPriceKlineCandlesticks returns a symbol's mark price klines from Mark Price Kline/Candlestick Data
func (e *Exchange) GetMarkPriceKlineCandlesticks(ctx context.Context, arg *UKlineRequest) ([]UPriceKline, error) {
	var resp []UPriceKline
	return resp, e.uSymbolKlineData(ctx, "/fapi/v1/markPriceKlines", arg, &resp)
}

// GetMultiAssetModeAssetIndex returns the index of one asset pair or, with an empty symbol, every asset, from Asset
// Index (formerly Multi-Assets Mode Asset Index), which since the CM-UM integration also lists the COIN-M settlement
// assets
func (e *Exchange) GetMultiAssetModeAssetIndex(ctx context.Context, symbol currency.Pair) ([]AssetIndex, error) {
	params := url.Values{}
	rateLimit := uFuturesAssetIndexAllRate
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		rateLimit = uFuturesDefaultRate
	}
	var resp AssetIndexResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/assetIndex", params), rateLimit, &resp)
}

// UFuturesHistoricalTrades returns older market trades from Old Trades Lookup, starting at a trade ID or, when fromID
// is 0, with the most recent; only the last month is kept
func (e *Exchange) UFuturesHistoricalTrades(ctx context.Context, symbol currency.Pair, fromID, limit uint64) ([]UPublicTradesData, error) {
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
	var resp []UPublicTradesData
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, common.EncodeURLValues("/fapi/v1/historicalTrades", params), uFuturesHistoricalTradesRate, &resp)
}

// UOpenInterest returns a symbol's present open interest from Open Interest
func (e *Exchange) UOpenInterest(ctx context.Context, symbol currency.Pair) (*UOpenInterestResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	var resp *UOpenInterestResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/openInterest", params), uFuturesDefaultRate, &resp)
}

// UOpenInterestStats returns a symbol's open interest history from Open Interest Statistics; only the last month is
// kept
func (e *Exchange) UOpenInterestStats(ctx context.Context, arg *UFuturesDataRequest) ([]UOpenInterestStats, error) {
	var resp []UOpenInterestStats
	return resp, e.uFuturesData(ctx, "/futures/data/openInterestHist", arg, &resp)
}

// UFuturesOrderbook returns a symbol's order book, which excludes retail price improvement orders, from Order Book.
// Valid limits are 5, 10, 20, 50, 100, 500 and 1000; 0 means the default of 500
func (e *Exchange) UFuturesOrderbook(ctx context.Context, symbol currency.Pair, limit uint64) (*UOrderBookResponse, error) {
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
	var resp *UOrderBookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/depth", params), uFuturesOrderbookLimit(limit), &resp)
}

// GetPremiumIndexKlineCandlesticks returns a symbol's premium index klines from Premium Index Kline Data
func (e *Exchange) GetPremiumIndexKlineCandlesticks(ctx context.Context, arg *UKlineRequest) ([]UPriceKline, error) {
	var resp []UPriceKline
	return resp, e.uSymbolKlineData(ctx, "/fapi/v1/premiumIndexKlines", arg, &resp)
}

// GetQuarterlyContractSettlementPrice returns a pair's quarterly contract settlement prices from Quarterly Contract
// Settlement Price
func (e *Exchange) GetQuarterlyContractSettlementPrice(ctx context.Context, pair currency.Pair) ([]SettlementPrice, error) {
	if pair.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	pairValue, err := e.FormatSymbol(pair, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("pair", pairValue)
	var resp []SettlementPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/futures/data/delivery-price", params), futuresDataRate, &resp)
}

// GetIndexPriceConstituents returns the constituents of a symbol's index price from Query Index Price Constituents
func (e *Exchange) GetIndexPriceConstituents(ctx context.Context, symbol currency.Pair) (*UIndexPriceConstituentsResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	var resp *UIndexPriceConstituentsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/constituents", params), uFuturesIndexConstituentsRate, &resp)
}

// URecentTrades returns a symbol's recent market trades from Recent Trades List
func (e *Exchange) URecentTrades(ctx context.Context, symbol currency.Pair, limit uint64) ([]UPublicTradesData, error) {
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
	var resp []UPublicTradesData
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/trades", params), uFuturesRecentTradesRate, &resp)
}

// GetURPIOrderbook returns a symbol's order book including retail price improvement orders, from RPI Order Book. The
// only valid limit is 1000, which 0 also means
func (e *Exchange) GetURPIOrderbook(ctx context.Context, symbol currency.Pair, limit uint64) (*UOrderBookResponse, error) {
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
	var resp *UOrderBookResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/rpiDepth", params), uFuturesRPIOrderbookRate, &resp)
}

// USymbolOrderbookTicker returns the best bid and ask of one symbol or, with an empty symbol, every symbol, from
// Symbol Order Book Ticker
func (e *Exchange) USymbolOrderbookTicker(ctx context.Context, symbol currency.Pair) ([]USymbolOrderbookTicker, error) {
	params := url.Values{}
	rateLimit := uFuturesOrderbookTickerAllRate
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		rateLimit = uFuturesBookTickerRate
	}
	var resp objectOrArray[USymbolOrderbookTicker]
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/ticker/bookTicker", params), rateLimit, &resp)
}

// USymbolPriceTickerV2 returns the latest price of one symbol or, with an empty symbol, every symbol, from Symbol
// Price Ticker V2
func (e *Exchange) USymbolPriceTickerV2(ctx context.Context, symbol currency.Pair) ([]USymbolPriceTicker, error) {
	params := url.Values{}
	rateLimit := uFuturesSymbolPriceAllRate
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		rateLimit = uFuturesDefaultRate
	}
	var resp objectOrArray[USymbolPriceTicker]
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v2/ticker/price", params), rateLimit, &resp)
}

// UTakerBuySellVol returns a symbol's taker buy and sell volumes from Taker Buy/Sell Volume; only the last 30 days
// are kept
func (e *Exchange) UTakerBuySellVol(ctx context.Context, arg *UFuturesDataRequest) ([]UTakerVolumeData, error) {
	var resp []UTakerVolumeData
	return resp, e.uFuturesData(ctx, "/futures/data/takerlongshortRatio", arg, &resp)
}

// U24HourTickerPriceChangeStats returns the 24 hour price change statistics of one symbol or, with an empty symbol,
// every symbol, from 24hr Ticker Price Change Statistics
func (e *Exchange) U24HourTickerPriceChangeStats(ctx context.Context, symbol currency.Pair) ([]U24HourPriceChangeStats, error) {
	params := url.Values{}
	rateLimit := uFuturesTicker24HourAllRate
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		rateLimit = uFuturesDefaultRate
	}
	var resp objectOrArray[U24HourPriceChangeStats]
	return resp, e.SendHTTPRequest(ctx, exchange.RestUSDTMargined, common.EncodeURLValues("/fapi/v1/ticker/24hr", params), rateLimit, &resp)
}

// UTopAccountsLongShortRatio returns the long/short account ratio of the top 20% of traders by margin balance, from
// Top Trader Long/Short Account Ratio; only the last 30 days are kept
func (e *Exchange) UTopAccountsLongShortRatio(ctx context.Context, arg *UFuturesDataRequest) ([]ULongShortRatio, error) {
	var resp []ULongShortRatio
	return resp, e.uFuturesData(ctx, "/futures/data/topLongShortAccountRatio", arg, &resp)
}

// UTopPositionsLongShortRatio returns the long/short position ratio of the top 20% of traders by margin balance, from
// Top Trader Long/Short Position Ratio; only the last 30 days are kept
func (e *Exchange) UTopPositionsLongShortRatio(ctx context.Context, arg *UFuturesDataRequest) ([]ULongShortRatio, error) {
	var resp []ULongShortRatio
	return resp, e.uFuturesData(ctx, "/futures/data/topLongShortPositionRatio", arg, &resp)
}

// uFuturesData sends one of the /futures/data statistics requests, which share their parameters. Binance documents
// the two top trader ratios as MARKET_DATA, but they answer without an API key, so they stay public
func (e *Exchange) uFuturesData(ctx context.Context, path string, arg *UFuturesDataRequest, result any) error {
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

// SetAssetsMode switches the account between Multi-Assets Mode (true) and Single-Asset Mode (false) on every symbol,
// with Change Multi-Assets Mode
func (e *Exchange) SetAssetsMode(ctx context.Context, multiAssetsMargin bool) error {
	params := url.Values{}
	params.Set("multiAssetsMargin", strconv.FormatBool(multiAssetsMargin))
	return e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/multiAssetsMargin", params, uFuturesDefaultRate, nil)
}

// ChangePositionMode switches the account between Hedge Mode (true) and One-way Mode (false) on every symbol, with
// Change Position Mode. Since the CM-UM integration the switch applies to COIN-M too, and is rejected while either has
// open orders or positions
func (e *Exchange) ChangePositionMode(ctx context.Context, dualSidePosition bool) error {
	params := url.Values{}
	params.Set("dualSidePosition", strconv.FormatBool(dualSidePosition))
	return e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/positionSide/dual", params, uFuturesDefaultRate, nil)
}

// UAccountTradesHistory returns the account's trades in a symbol from Account Trade List, which may also list COIN-M
// trades. A time window may span at most seven days, and only the last three months are kept
func (e *Exchange) UAccountTradesHistory(ctx context.Context, arg *UAccountTradeListRequest) ([]UAccountTradeHistory, error) {
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
	var resp []UAccountTradeHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/userTrades", params, uFuturesAccountTradeListRate, &resp)
}

// UAllAccountOrders returns the account's active, cancelled and filled orders from All Orders, which may also list
// COIN-M orders. A time window may span at most seven days, and cancelled or expired orders without fills drop out
// after three days
func (e *Exchange) UAllAccountOrders(ctx context.Context, arg *UAllOrdersRequest) ([]UFuturesOrderData, error) {
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
	var resp []UFuturesOrderData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/allOrders", params, uFuturesGetAllOrdersRate, &resp)
}

// UAutoCancelAllOpenOrders cancels a symbol's open orders once the countdown elapses, with Auto-Cancel All Open
// Orders. Calling it again replaces the countdown, so it serves as a heartbeat, and a zero countdown stops it
func (e *Exchange) UAutoCancelAllOpenOrders(ctx context.Context, symbol currency.Pair, countdownTime time.Duration) (*UAutoCancelAllOpenOrdersResponse, error) {
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
	var resp *UAutoCancelAllOpenOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/countdownCancelAll", params, uFuturesCountdownCancelRate, &resp)
}

// UQueryAlgoOrder returns an algo (conditional) order's state from Query Algo Order, by its algo ID or, when algoID
// is 0, its client algo ID
func (e *Exchange) UQueryAlgoOrder(ctx context.Context, algoID uint64, clientAlgoID string) (*UAlgoOrderResponse, error) {
	params, err := uAlgoOrderIDParams(algoID, clientAlgoID)
	if err != nil {
		return nil, err
	}
	var resp *UAlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/algoOrder", params, uFuturesDefaultRate, &resp)
}

// UNewAlgoOrder places an algo (conditional) order, which is how stop, take profit and trailing stop orders are
// placed since Binance moved them off New Order, with New Algo Order
func (e *Exchange) UNewAlgoOrder(ctx context.Context, arg *UNewAlgoOrderRequest) (*UNewAlgoOrderResponse, error) {
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
	symbol, err := e.FormatSymbol(arg.Symbol, asset.USDTMarginedFutures)
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
	if arg.NewOrderRespType != "" {
		params.Set("newOrderRespType", arg.NewOrderRespType)
	}
	if arg.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", arg.SelfTradePreventionMode)
	}
	if !arg.GoodTillDate.IsZero() {
		params.Set("goodTillDate", strconv.FormatInt(arg.GoodTillDate.UnixMilli(), 10))
	}
	var resp *UNewAlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/algoOrder", params, uFuturesOrdersDefaultRate, &resp)
}

// UCancelAlgoOrder cancels an active algo (conditional) order with Cancel Algo Order, by its algo ID or, when algoID
// is 0, its client algo ID
func (e *Exchange) UCancelAlgoOrder(ctx context.Context, algoID uint64, clientAlgoID string) (*UCancelAlgoOrderResponse, error) {
	params, err := uAlgoOrderIDParams(algoID, clientAlgoID)
	if err != nil {
		return nil, err
	}
	var resp *UCancelAlgoOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodDelete, "/fapi/v1/algoOrder", params, uFuturesDefaultRate, &resp)
}

// uAlgoOrderIDParams returns the parameters identifying one algo order, which Binance requires by either ID
func uAlgoOrderIDParams(algoID uint64, clientAlgoID string) (url.Values, error) {
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

// UCancelAllAlgoOpenOrders cancels a symbol's open algo (conditional) orders with Cancel All Algo Open Orders
func (e *Exchange) UCancelAllAlgoOpenOrders(ctx context.Context, symbol currency.Pair) error {
	if symbol.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	return e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodDelete, "/fapi/v1/algoOpenOrders", params, uFuturesDefaultRate, nil)
}

// UCancelAllOpenOrders cancels a symbol's open orders with Cancel All Open Orders
func (e *Exchange) UCancelAllOpenOrders(ctx context.Context, symbol currency.Pair) error {
	if symbol.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	return e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodDelete, "/fapi/v1/allOpenOrders", params, uFuturesDefaultRate, nil)
}

// UModifyMultipleOrders modifies up to five limit orders with Modify Multiple Orders, which processes them
// concurrently and replies in request order, each element holding the modified order or the error that rejected it
func (e *Exchange) UModifyMultipleOrders(ctx context.Context, args []UModifyOrderRequest) ([]UModifyBatchOrderResponse, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	orders := make([]map[string]string, len(args))
	for i := range args {
		if args[i].ReduceOnly {
			return nil, errBatchModifyReduceOnly
		}
		params, err := e.uModifyOrderParams(&args[i])
		if err != nil {
			return nil, err
		}
		orders[i] = uBatchItem(params)
	}
	batch, err := json.Marshal(orders)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("batchOrders", string(batch))
	var resp []UModifyBatchOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPut, "/fapi/v1/batchOrders", params, uFuturesBatchOrdersRate, &resp)
}

// UPlaceBatchOrders places up to five limit or market orders with Place Multiple Orders, which processes them
// concurrently and replies in request order, each element holding the new order or the error that rejected it
func (e *Exchange) UPlaceBatchOrders(ctx context.Context, args []UFuturesNewOrderRequest) ([]UPlaceBatchOrderResponse, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	orders := make([]map[string]string, len(args))
	for i := range args {
		params, err := e.uNewOrderParams(&args[i])
		if err != nil {
			return nil, err
		}
		orders[i] = uBatchItem(params)
	}
	batch, err := json.Marshal(orders)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("batchOrders", string(batch))
	var resp []UPlaceBatchOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/batchOrders", params, uFuturesBatchOrdersRate, &resp)
}

// uBatchItem converts one order's parameters into a batchOrders element. Values stay strings, as in Binance's
// examples, and the JSON encoder sorts the keys, so a request always encodes the same way
func uBatchItem(params url.Values) map[string]string {
	item := make(map[string]string, len(params))
	for k := range params {
		item[k] = params.Get(k)
	}
	return item
}

// UCancelBatchOrders cancels up to ten of a symbol's orders by order ID or client order ID with Cancel Multiple
// Orders, each reply element holding the cancelled order or the error that rejected it
func (e *Exchange) UCancelBatchOrders(ctx context.Context, symbol currency.Pair, orderIDs []uint64, origClientOrderIDs []string) ([]UCancelBatchOrderResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if len(orderIDs) == 0 && len(origClientOrderIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
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
	var resp []UCancelBatchOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodDelete, "/fapi/v1/batchOrders", params, uFuturesDefaultRate, &resp)
}

// UGetOrderData returns an order's state from Query Order, by its order ID or, when orderID is 0, its client order
// ID. Cancelled or expired orders without fills drop out after three days, and every order after 90 days
func (e *Exchange) UGetOrderData(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*UOrderResponse, error) {
	params, err := e.uOrderIDParams(symbol, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *UOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/order", params, uFuturesDefaultRate, &resp)
}

// uOrderIDParams returns the parameters identifying one of a symbol's orders, which Binance requires by either ID
func (e *Exchange) uOrderIDParams(symbol currency.Pair, orderID uint64, origClientOrderID string) (url.Values, error) {
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

// UModifyOrder modifies a limit order's price and quantity with Modify Order, which moves the order to the back of the
// matching queue
func (e *Exchange) UModifyOrder(ctx context.Context, arg *UModifyOrderRequest) (*UModifyOrderResponse, error) {
	params, err := e.uModifyOrderParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *UModifyOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPut, "/fapi/v1/order", params, uFuturesOrdersDefaultRate, &resp)
}

// uModifyOrderParams converts a modify request into its parameters, which Modify Order and Modify Multiple Orders
// share
func (e *Exchange) uModifyOrderParams(arg *UModifyOrderRequest) (url.Values, error) {
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

// UFuturesNewOrder places a limit or market order with New Order; stop, take profit and trailing stop orders go
// through UNewAlgoOrder
func (e *Exchange) UFuturesNewOrder(ctx context.Context, arg *UFuturesNewOrderRequest) (*UNewOrderResponse, error) {
	params, err := e.uNewOrderParams(arg)
	if err != nil {
		return nil, err
	}
	var resp *UNewOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/order", params, uFuturesOrdersDefaultRate, &resp)
}

// uNewOrderParams converts a new order request into its parameters, which New Order and Place Multiple Orders share
func (e *Exchange) uNewOrderParams(arg *UFuturesNewOrderRequest) (url.Values, error) {
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
	if arg.NewOrderRespType != "" {
		params.Set("newOrderRespType", arg.NewOrderRespType)
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

// UCancelOrder cancels an active order with Cancel Order, by its order ID or, when orderID is 0, its client order ID
func (e *Exchange) UCancelOrder(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*UCancelOrderResponse, error) {
	params, err := e.uOrderIDParams(symbol, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *UCancelOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodDelete, "/fapi/v1/order", params, uFuturesDefaultRate, &resp)
}

// UChangeInitialLeverageRequest changes a symbol's initial leverage, 1 to 125, with Change Initial Leverage
func (e *Exchange) UChangeInitialLeverageRequest(ctx context.Context, symbol currency.Pair, leverage uint64) (*UChangeInitialLeverageResponse, error) {
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
	var resp *UChangeInitialLeverageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/leverage", params, uFuturesDefaultRate, &resp)
}

// UChangeInitialMarginType changes a symbol's margin type to ISOLATED or CROSSED with Change Margin Type
func (e *Exchange) UChangeInitialMarginType(ctx context.Context, symbol currency.Pair, marginType string) error {
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
	return e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/marginType", params, uFuturesDefaultRate, nil)
}

// UCurrentAllAlgoOpenOrders returns open algo (conditional) orders from Current All Algo Open Orders; each argument is
// an optional filter, and an empty symbol returns every symbol's orders
func (e *Exchange) UCurrentAllAlgoOpenOrders(ctx context.Context, algoType string, symbol currency.Pair, algoID uint64) ([]UAlgoOrder, error) {
	params := url.Values{}
	rateLimit := uFuturesOpenAlgoOrdersAllRate
	if algoType != "" {
		params.Set("algoType", algoType)
	}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		rateLimit = uFuturesDefaultRate
	}
	if algoID > 0 {
		params.Set("algoId", strconv.FormatUint(algoID, 10))
	}
	var resp []UAlgoOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/openAlgoOrders", params, rateLimit, &resp)
}

// UAllAccountOpenOrders returns the open orders of one symbol or, with an empty symbol, every symbol, from Current All
// Open Orders
func (e *Exchange) UAllAccountOpenOrders(ctx context.Context, symbol currency.Pair) ([]UOrderResponse, error) {
	params := url.Values{}
	rateLimit := uFuturesGetAllOpenOrdersRate
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		rateLimit = uFuturesDefaultRate
	}
	var resp []UOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/openOrders", params, rateLimit, &resp)
}

// GetUSDTOrderModifyHistory returns an order's modifications from Get Order Modify History; only the last three
// months are kept
func (e *Exchange) GetUSDTOrderModifyHistory(ctx context.Context, arg *UOrderModifyHistoryRequest) ([]FuturesOrderAmendment, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.uOrderIDParams(arg.Symbol, arg.OrderID, arg.OrigClientOrderID)
	if err != nil {
		return nil, err
	}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []FuturesOrderAmendment
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/orderAmendment", params, uFuturesDefaultRate, &resp)
}

// UPositionMarginChangeHistory returns a symbol's isolated position margin changes from Get Position Margin Change
// History; a time window may span at most 30 days, and only the last 30 days are kept
func (e *Exchange) UPositionMarginChangeHistory(ctx context.Context, arg *UPositionMarginChangeHistoryRequest) ([]UPositionMarginChange, error) {
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
	var resp []UPositionMarginChange
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/positionMargin/history", params, uFuturesDefaultRate, &resp)
}

// UModifyIsolatedPositionMarginReq adds margin to or reduces margin from an isolated position with Modify Isolated
// Position Margin; changeType is add or reduce, and positionSide is required in Hedge Mode
func (e *Exchange) UModifyIsolatedPositionMarginReq(ctx context.Context, symbol currency.Pair, positionSide, changeType string, amount float64) (*UModifyIsolatedPositionMarginResponse, error) {
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
	var resp *UModifyIsolatedPositionMarginResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/positionMargin", params, uFuturesDefaultRate, &resp)
}

// UPositionsADLEstimate returns the auto-deleveraging quantiles of one symbol's positions or, with an empty symbol,
// every symbol's, from Position ADL Quantile Estimation
func (e *Exchange) UPositionsADLEstimate(ctx context.Context, symbol currency.Pair) ([]UPositionADLQuantile, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp []UPositionADLQuantile
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/adlQuantile", params, uFuturesADLQuantileRate, &resp)
}

// UPositionsInfoV2 returns the positions of one symbol or, with an empty symbol, every symbol, from Position
// Information V2
func (e *Exchange) UPositionsInfoV2(ctx context.Context, symbol currency.Pair) ([]UPositionInformationV2, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp []UPositionInformationV2
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v2/positionRisk", params, uFuturesPositionRiskRate, &resp)
}

// UPositionsInfoV3 returns the positions of one symbol or, with an empty symbol, every symbol with a position or open
// orders, from Position Information V3
func (e *Exchange) UPositionsInfoV3(ctx context.Context, symbol currency.Pair) ([]UPositionInformationV3, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp []UPositionInformationV3
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v3/positionRisk", params, uFuturesPositionRiskRate, &resp)
}

// UQueryAllAlgoOrders returns a symbol's active, cancelled, triggered and finished algo (conditional) orders from
// Query All Algo Orders. A time window may span at most seven days
func (e *Exchange) UQueryAllAlgoOrders(ctx context.Context, arg *UAllAlgoOrdersRequest) ([]UAlgoOrder, error) {
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
	var resp []UAlgoOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/allAlgoOrders", params, uFuturesAllAlgoOrdersRate, &resp)
}

// UFetchOpenOrder returns an open order's state from Query Current Open Order, by its order ID or, when orderID is 0,
// its client order ID
func (e *Exchange) UFetchOpenOrder(ctx context.Context, symbol currency.Pair, orderID uint64, origClientOrderID string) (*UOrderResponse, error) {
	params, err := e.uOrderIDParams(symbol, orderID, origClientOrderID)
	if err != nil {
		return nil, err
	}
	var resp *UOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/openOrder", params, uFuturesDefaultRate, &resp)
}

// UAccountForcedOrders returns the account's liquidation and auto-deleveraging orders from User's Force Orders, which
// may also list COIN-M orders; only the last 90 days are kept
func (e *Exchange) UAccountForcedOrders(ctx context.Context, arg *UForceOrdersRequest) ([]UForceOrder, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setTimeRangeParams(params, arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	rateLimit := uFuturesAllForceOrdersRate
	if !arg.Symbol.IsEmpty() {
		symbol, err := e.FormatSymbol(arg.Symbol, asset.USDTMarginedFutures)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbol)
		rateLimit = uFuturesSymbolForceOrdersRate
	}
	if arg.AutoCloseType != "" {
		params.Set("autoCloseType", arg.AutoCloseType)
	}
	if arg.Limit > 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	var resp []UForceOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/fapi/v1/forceOrders", params, rateLimit, &resp)
}

// KeepaliveUFuturesUserDataStream extends the validity of the account's USDⓈ-M user data stream listen key by 60
// minutes. A -1125 error means the key has expired and StartUFuturesUserDataStream must create a new one
func (e *Exchange) KeepaliveUFuturesUserDataStream(ctx context.Context) (*FuturesListenKeyResponse, error) {
	var resp *FuturesListenKeyResponse
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPut, "/fapi/v1/listenKey", uFuturesDefaultRate, &resp)
}

// StartUFuturesUserDataStream starts a USDⓈ-M user data stream. An account with an active listen key gets that key
// back with its validity extended by 60 minutes
func (e *Exchange) StartUFuturesUserDataStream(ctx context.Context) (*FuturesListenKeyResponse, error) {
	var resp *FuturesListenKeyResponse
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/fapi/v1/listenKey", uFuturesDefaultRate, &resp)
}

// CloseUFuturesUserDataStream closes the account's USDⓈ-M user data stream and invalidates its listen key
func (e *Exchange) CloseUFuturesUserDataStream(ctx context.Context) error {
	return e.SendAPIKeyHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodDelete, "/fapi/v1/listenKey", uFuturesDefaultRate, nil)
}
