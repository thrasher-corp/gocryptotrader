package binance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var errMaxLeverageRequired = errors.New("max leverage is required")

// AdjustCrossMarginMaxLeverage adjusts the cross margin max leverage. The margin level must exceed the initial risk
// ratio of the new leverage (1.5 for 3x, 1.25 for 5x); Cross Margin Classic accepts 3 or 5
func (e *Exchange) AdjustCrossMarginMaxLeverage(ctx context.Context, maxLeverage uint64) (*AdjustCrossMarginMaxLeverageResponse, error) {
	if maxLeverage == 0 {
		return nil, errMaxLeverageRequired
	}
	params := url.Values{}
	params.Set("maxLeverage", strconv.FormatUint(maxLeverage, 10))
	var resp *AdjustCrossMarginMaxLeverageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/margin/max-leverage", params, adjustCrossMarginMaxLeverageRate, &resp)
}

// GetIsolatedMarginAccountInfo returns the isolated margin account info of up to five symbols, or of every isolated
// margin account when symbols is empty
func (e *Exchange) GetIsolatedMarginAccountInfo(ctx context.Context, symbols currency.Pairs) (*IsolatedMarginAccountInfoResponse, error) {
	params := url.Values{}
	if len(symbols) > 0 {
		symbolValues := make([]string, len(symbols))
		for i := range symbols {
			symbolValue, err := e.FormatSymbol(symbols[i], asset.Margin)
			if err != nil {
				return nil, err
			}
			symbolValues[i] = symbolValue
		}
		params.Set("symbols", strings.Join(symbolValues, ","))
	}
	var resp *IsolatedMarginAccountInfoResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/isolated/account", params, getIsolatedMarginAccountInfoRate, &resp)
}

// EnableIsolatedMarginAccount enables the isolated margin account of a symbol; only accounts disabled earlier can be
// enabled again
func (e *Exchange) EnableIsolatedMarginAccount(ctx context.Context, symbol currency.Pair) (*IsolatedMarginAccountToggleResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.Margin)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	var resp *IsolatedMarginAccountToggleResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/margin/isolated/account", params, enableIsolatedMarginAccountRate, &resp)
}

// DisableIsolatedMarginAccount disables the isolated margin account of a symbol; each symbol can be disabled once
// every 24 hours
func (e *Exchange) DisableIsolatedMarginAccount(ctx context.Context, symbol currency.Pair) (*IsolatedMarginAccountToggleResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.Margin)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	var resp *IsolatedMarginAccountToggleResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodDelete, "/sapi/v1/margin/isolated/account", params, disableIsolatedMarginAccountRate, &resp)
}

// GetBNBBurnStatus returns whether BNB is burnt to pay spot trading fees and margin interest
func (e *Exchange) GetBNBBurnStatus(ctx context.Context) (*BNBBurnStatusResponse, error) {
	var resp *BNBBurnStatusResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/bnbBurn", nil, sapiDefaultRate, &resp)
}

// GetSummaryOfMarginAccount returns the summary of the margin account: the margin levels at which Binance treats it
// as normal, calls for margin and force liquidates it
func (e *Exchange) GetSummaryOfMarginAccount(ctx context.Context) (*MarginAccountSummaryResponse, error) {
	var resp *MarginAccountSummaryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/tradeCoeff", nil, marginAccountSummaryRate, &resp)
}

// GetCrossOrIsolatedMarginCapitalFlow returns the cross or isolated margin capital flow. Binance keeps the last 90
// days, rejects windows longer than 7 days and queries the last 7 days when no time is given
func (e *Exchange) GetCrossOrIsolatedMarginCapitalFlow(ctx context.Context, req *MarginCapitalFlowRequest) ([]MarginCapitalFlow, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !req.Asset.IsEmpty() {
		params.Set("asset", req.Asset.Upper().String())
	}
	if !req.Symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(req.Symbol, asset.Margin)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	if req.Type != "" {
		params.Set("type", req.Type)
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
	var resp []MarginCapitalFlow
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/capital-flow", params, marginCapitalFlowRate, &resp)
}

// GetCrossMarginAccountDetail returns the cross margin account details
func (e *Exchange) GetCrossMarginAccountDetail(ctx context.Context) (*CrossMarginAccountDetailResponse, error) {
	var resp *CrossMarginAccountDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/account", nil, getCrossMarginAccountDetailRate, &resp)
}

// GetCrossMarginFeeData returns the cross margin fee data of one coin, or of every coin when coin is empty, at a VIP
// level; a nil vipLevel returns the user's own level
func (e *Exchange) GetCrossMarginFeeData(ctx context.Context, vipLevel *uint64, coin currency.Code) ([]CrossMarginFeeData, error) {
	params := url.Values{}
	if vipLevel != nil {
		params.Set("vipLevel", strconv.FormatUint(*vipLevel, 10))
	}
	// Binance weighs a request for every coin at 5 and one for a single coin at 1
	rateLimit := allCrossMarginFeeDataRate
	if !coin.IsEmpty() {
		params.Set("coin", coin.Upper().String())
		rateLimit = sapiDefaultRate
	}
	var resp []CrossMarginFeeData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/crossMarginData", params, rateLimit, &resp)
}

// GetEnabledIsolatedMarginAccountLimit returns the number of enabled isolated margin accounts and the maximum allowed
func (e *Exchange) GetEnabledIsolatedMarginAccountLimit(ctx context.Context) (*IsolatedMarginAccountLimitResponse, error) {
	var resp *IsolatedMarginAccountLimitResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/isolated/accountLimit", nil, sapiDefaultRate, &resp)
}

// GetIsolatedMarginFeeData returns the isolated margin fee data of one symbol, or of every symbol when symbol is
// empty, at a VIP level; a nil vipLevel returns the user's own level
func (e *Exchange) GetIsolatedMarginFeeData(ctx context.Context, vipLevel *uint64, symbol currency.Pair) ([]IsolatedMarginFeeData, error) {
	params := url.Values{}
	if vipLevel != nil {
		params.Set("vipLevel", strconv.FormatUint(*vipLevel, 10))
	}
	// Binance weighs a request for every symbol at 10 and one for a single symbol at 1
	rateLimit := allIsolatedMarginFeeDataRate
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.Margin)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
		rateLimit = sapiDefaultRate
	}
	var resp []IsolatedMarginFeeData
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/isolatedMarginData", params, rateLimit, &resp)
}

// GetFutureHourlyInterestRate returns the next hourly interest rate estimate of each asset for cross or isolated
// margin
func (e *Exchange) GetFutureHourlyInterestRate(ctx context.Context, assets []currency.Code, isIsolated bool) ([]HourlyInterestRate, error) {
	if len(assets) == 0 {
		return nil, currency.ErrCurrencyCodesEmpty
	}
	assetNames := make([]string, len(assets))
	for i := range assets {
		assetNames[i] = assets[i].Upper().String()
	}
	params := url.Values{}
	params.Set("assets", strings.Join(assetNames, ","))
	params.Set("isIsolated", strings.ToUpper(strconv.FormatBool(isIsolated)))
	var resp []HourlyInterestRate
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/next-hourly-interest-rate", params, marginHourlyInterestRate, &resp)
}

// GetUserMarginInterestHistory returns the margin interest history, newest first, of cross margin or of one isolated
// symbol. Binance keeps the last 90 days, allows windows of up to 30 days and returns the last 7 days by default
func (e *Exchange) GetUserMarginInterestHistory(ctx context.Context, req *UserMarginInterestHistoryRequest) (*UserMarginInterestHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !req.Asset.IsEmpty() {
		params.Set("asset", req.Asset.Upper().String())
	}
	if !req.IsolatedSymbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(req.IsolatedSymbol, asset.Margin)
		if err != nil {
			return nil, err
		}
		params.Set("isolatedSymbol", symbolValue)
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Current != 0 {
		params.Set("current", strconv.FormatUint(req.Current, 10))
	}
	if req.Size != 0 {
		params.Set("size", strconv.FormatUint(req.Size, 10))
	}
	var resp *UserMarginInterestHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/interestHistory", params, sapiDefaultRate, &resp)
}

// GetBorrowOrRepayRecordsInMarginAccount returns the borrow or repay records in the margin account, newest first.
// TransactionID takes precedence over the time window, which defaults to the last 7 days
func (e *Exchange) GetBorrowOrRepayRecordsInMarginAccount(ctx context.Context, req *MarginBorrowRepayRecordsRequest) (*MarginBorrowRepayRecordsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Type == "" {
		return nil, errLendingTypeRequired
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("type", req.Type)
	if !req.Asset.IsEmpty() {
		params.Set("asset", req.Asset.Upper().String())
	}
	if !req.IsolatedSymbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(req.IsolatedSymbol, asset.Margin)
		if err != nil {
			return nil, err
		}
		params.Set("isolatedSymbol", symbolValue)
	}
	if req.TransactionID != 0 {
		params.Set("txId", strconv.FormatUint(req.TransactionID, 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Current != 0 {
		params.Set("current", strconv.FormatUint(req.Current, 10))
	}
	if req.Size != 0 {
		params.Set("size", strconv.FormatUint(req.Size, 10))
	}
	var resp *MarginBorrowRepayRecordsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/borrow-repay", params, borrowRepayRecordsInMarginAccountRate, &resp)
}

// MarginAccountBorrowRepay borrows or repays an asset in the cross margin account, or in the isolated margin account
// of Symbol when IsIsolated is set. Binance processes one borrow or repayment per account at a time and rejects
// requests sent while another is pending, so space consecutive requests by at least 100ms
func (e *Exchange) MarginAccountBorrowRepay(ctx context.Context, req *MarginAccountBorrowRepayRequest) (*MarginAccountBorrowRepayResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if req.IsIsolated && req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if req.Type == "" {
		return nil, errLendingTypeRequired
	}
	params := url.Values{}
	params.Set("asset", req.Asset.Upper().String())
	params.Set("isIsolated", strings.ToUpper(strconv.FormatBool(req.IsIsolated)))
	// Binance documents symbol for isolated margin only
	if req.IsIsolated {
		symbolValue, err := e.FormatSymbol(req.Symbol, asset.Margin)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	params.Set("amount", strconv.FormatFloat(req.Amount, 'f', -1, 64))
	params.Set("type", req.Type)
	var resp *MarginAccountBorrowRepayResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/margin/borrow-repay", params, marginAccountBorrowRepayRate, &resp)
}

// GetMarginInterestRateHistory returns the daily margin interest rate history of an asset; windows may span up to 30
// days
func (e *Exchange) GetMarginInterestRateHistory(ctx context.Context, req *MarginInterestRateHistoryRequest) ([]MarginInterestRate, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("asset", req.Asset.Upper().String())
	if req.VIPLevel != nil {
		params.Set("vipLevel", strconv.FormatUint(*req.VIPLevel, 10))
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	var resp []MarginInterestRate
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/interestRateHistory", params, sapiDefaultRate, &resp)
}

// GetMaxBorrow returns the maximum amount of an asset the cross margin account, or the isolated margin account of
// isolatedSymbol, can borrow
func (e *Exchange) GetMaxBorrow(ctx context.Context, assetName currency.Code, isolatedSymbol currency.Pair) (*MarginMaxBorrowableResponse, error) {
	if assetName.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("asset", assetName.Upper().String())
	if !isolatedSymbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(isolatedSymbol, asset.Margin)
		if err != nil {
			return nil, err
		}
		params.Set("isolatedSymbol", symbolValue)
	}
	var resp *MarginMaxBorrowableResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/maxBorrowable", params, marginMaxBorrowRate, &resp)
}

// GetCrossMarginCollateralRatio returns the cross margin collateral discount rate tiers of each group of assets
func (e *Exchange) GetCrossMarginCollateralRatio(ctx context.Context) ([]CrossMarginCollateralRatio, error) {
	var resp []CrossMarginCollateralRatio
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/crossMarginCollateralRatio", crossMarginCollateralRatioRate, &resp)
}

// GetAllCrossMarginPairs returns every cross margin pair, or one when symbol is set
func (e *Exchange) GetAllCrossMarginPairs(ctx context.Context, symbol currency.Pair) ([]CrossMarginPair, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.Margin)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp []CrossMarginPair
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, common.EncodeURLValues("/sapi/v1/margin/allPairs", params), sapiDefaultRate, &resp)
}

// GetAllIsolatedMarginSymbols returns every isolated margin symbol, or one when symbol is set
func (e *Exchange) GetAllIsolatedMarginSymbols(ctx context.Context, symbol currency.Pair) ([]IsolatedMarginSymbol, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.Margin)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp []IsolatedMarginSymbol
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, common.EncodeURLValues("/sapi/v1/margin/isolated/allPairs", params), allIsolatedMarginSymbolsRate, &resp)
}

// GetAllMarginAssets returns every margin asset, or one when assetName is set
func (e *Exchange) GetAllMarginAssets(ctx context.Context, assetName currency.Code) ([]MarginAssetInfo, error) {
	params := url.Values{}
	if !assetName.IsEmpty() {
		params.Set("asset", assetName.Upper().String())
	}
	var resp []MarginAssetInfo
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, common.EncodeURLValues("/sapi/v1/margin/allAssets", params), sapiDefaultRate, &resp)
}

// GetTokensOrSymbolsDelistSchedule returns the delist schedule of cross margin assets and isolated margin symbols
func (e *Exchange) GetTokensOrSymbolsDelistSchedule(ctx context.Context) ([]MarginDelistSchedule, error) {
	var resp []MarginDelistSchedule
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/delist-schedule", marginTokensAndSymbolsDelistScheduleRate, &resp)
}

// GetIsolatedMarginTierData returns the isolated margin tier data of a symbol, for one tier or for every tier when
// tier is 0
func (e *Exchange) GetIsolatedMarginTierData(ctx context.Context, symbol currency.Pair, tier uint64) ([]IsolatedMarginTier, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.Margin)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if tier != 0 {
		params.Set("tier", strconv.FormatUint(tier, 10))
	}
	var resp []IsolatedMarginTier
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/isolatedMarginTier", params, sapiDefaultRate, &resp)
}

// GetMarginAvailableInventory returns the inventory available to borrow for marginType, MARGIN or ISOLATED
func (e *Exchange) GetMarginAvailableInventory(ctx context.Context, marginType string) (*MarginAvailableInventoryResponse, error) {
	if marginType == "" {
		return nil, margin.ErrInvalidMarginType
	}
	params := url.Values{}
	params.Set("type", marginType)
	var resp *MarginAvailableInventoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/available-inventory", params, marginAvailableInventoryRate, &resp)
}

// GetMarginPriceIndex returns the margin price index of a symbol
func (e *Exchange) GetMarginPriceIndex(ctx context.Context, symbol currency.Pair) (*MarginPriceIndexResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.Margin)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	var resp *MarginPriceIndexResponse
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, common.EncodeURLValues("/sapi/v1/margin/priceIndex", params), getPriceMarginIndexRate, &resp)
}

// GetForceLiquidationRecord returns the force liquidation records, newest first, of cross margin or of one isolated
// symbol
func (e *Exchange) GetForceLiquidationRecord(ctx context.Context, req *ForceLiquidationRecordRequest) (*ForceLiquidationRecordResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if !req.IsolatedSymbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(req.IsolatedSymbol, asset.Margin)
		if err != nil {
			return nil, err
		}
		params.Set("isolatedSymbol", symbolValue)
	}
	if req.Current != 0 {
		params.Set("current", strconv.FormatUint(req.Current, 10))
	}
	if req.Size != 0 {
		params.Set("size", strconv.FormatUint(req.Size, 10))
	}
	var resp *ForceLiquidationRecordResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/forceLiquidationRec", params, sapiDefaultRate, &resp)
}

// GetSmallLiabilityExchangeCoinList returns the liabilities MarginSmallLiabilityExchange can exchange
func (e *Exchange) GetSmallLiabilityExchangeCoinList(ctx context.Context) ([]SmallLiabilityExchangeCoin, error) {
	var resp []SmallLiabilityExchangeCoin
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/exchange-small-liability", nil, getSmallLiabilityExchangeCoinListRate, &resp)
}

// MarginSmallLiabilityExchange exchanges small cross margin liabilities. Binance accepts up to 10 assets, each with a
// liability valued under 10 USDT, once every 6 hours, and returns no data
func (e *Exchange) MarginSmallLiabilityExchange(ctx context.Context, assetNames []currency.Code) error {
	if len(assetNames) == 0 {
		return errEmptyCurrencyCodes
	}
	names := make([]string, len(assetNames))
	for i := range assetNames {
		names[i] = assetNames[i].Upper().String()
	}
	params := url.Values{}
	params.Set("assetNames", strings.Join(names, ","))
	return e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/margin/exchange-small-liability", params, marginSmallLiabilityExchangeRate, nil)
}

// GetSmallLiabilityExchangeHistory returns a page of the small liability exchange history
func (e *Exchange) GetSmallLiabilityExchangeHistory(ctx context.Context, req *SmallLiabilityExchangeHistoryRequest) (*SmallLiabilityExchangeHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Current == 0 {
		return nil, errPageNumberRequired
	}
	if req.Size == 0 {
		return nil, errPageSizeRequired
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	params.Set("current", strconv.FormatUint(req.Current, 10))
	params.Set("size", strconv.FormatUint(req.Size, 10))
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	var resp *SmallLiabilityExchangeHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/exchange-small-liability-history", params, smallLiabilityExchangeHistoryRate, &resp)
}

// GetMarginAccountsOpenOrders returns the margin account's open orders on symbol, or on every symbol when symbol is
// empty, which Binance weighs as one request per trading symbol. Isolated margin requires symbol
func (e *Exchange) GetMarginAccountsOpenOrders(ctx context.Context, symbol currency.Pair, isIsolated bool) ([]MarginTradeOrderResponse, error) {
	if isIsolated && symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.Margin)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	if isIsolated {
		params.Set("isIsolated", "TRUE")
	}
	var resp []MarginTradeOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/openOrders", params, getMarginAccountsOpenOrdersRate, &resp)
}

// CancelAllOpenMarginAccountOrdersOnSymbol cancels every open order on a symbol in the margin account, order lists
// such as OCO included
func (e *Exchange) CancelAllOpenMarginAccountOrdersOnSymbol(ctx context.Context, symbol currency.Pair, isIsolated bool) ([]MarginCancelledOrder, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.Margin)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if isIsolated {
		params.Set("isIsolated", "TRUE")
	}
	var resp []MarginCancelledOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodDelete, "/sapi/v1/margin/openOrders", params, sapiDefaultRate, &resp)
}

// GetMarginAccountOCOOrder returns one of the margin account's order lists by orderListID or originalClientOrderID
func (e *Exchange) GetMarginAccountOCOOrder(ctx context.Context, symbol currency.Pair, isIsolated bool, orderListID uint64, originalClientOrderID string) (*MarginOrderListResponse, error) {
	params := url.Values{}
	if isIsolated {
		params.Set("isIsolated", "TRUE")
	}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.Margin)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	if orderListID != 0 {
		params.Set("orderListId", strconv.FormatUint(orderListID, 10))
	}
	if originalClientOrderID != "" {
		params.Set("origClientOrderId", originalClientOrderID)
	}
	var resp *MarginOrderListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/orderList", params, getMarginAccountOCOOrderRate, &resp)
}

// CancelMarginAccountOCOOrder cancels an entire order list in the margin account; cancelling one leg cancels the whole
// list
func (e *Exchange) CancelMarginAccountOCOOrder(ctx context.Context, req *MarginOCOCancelRequest) (*MarginOCOCancelResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	symbolValue, err := e.FormatSymbol(req.Symbol, asset.Margin)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if req.IsIsolated {
		params.Set("isIsolated", "TRUE")
	}
	if req.OrderListID != 0 {
		params.Set("orderListId", strconv.FormatUint(req.OrderListID, 10))
	}
	if req.ListClientOrderID != "" {
		params.Set("listClientOrderId", req.ListClientOrderID)
	}
	if req.NewClientOrderID != "" {
		params.Set("newClientOrderId", req.NewClientOrderID)
	}
	var resp *MarginOCOCancelResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodDelete, "/sapi/v1/margin/orderList", params, sapiUIDDefaultRate, &resp)
}

// GetMarginAccountsOrder returns one of the margin account's orders by orderID or originalClientOrderID
func (e *Exchange) GetMarginAccountsOrder(ctx context.Context, symbol currency.Pair, isIsolated bool, orderID uint64, originalClientOrderID string) (*MarginTradeOrderResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if orderID == 0 && originalClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.Margin)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if isIsolated {
		params.Set("isIsolated", "TRUE")
	}
	if orderID != 0 {
		params.Set("orderId", strconv.FormatUint(orderID, 10))
	}
	if originalClientOrderID != "" {
		params.Set("origClientOrderId", originalClientOrderID)
	}
	var resp *MarginTradeOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/order", params, getMarginAccountOrderRate, &resp)
}

// PostMarginAccountOrder places a new margin account order. Binance weighs orders with a MARGIN_BUY or
// AUTO_BORROW_REPAY side effect at 1500 instead of 6, advises setting AutoRepayAtCancel to false when orders are placed
// and cancelled frequently, and from 2026-10-14 rejects STOP_LOSS and TAKE_PROFIT orders with -3116 on accounts that do
// not support them
func (e *Exchange) PostMarginAccountOrder(ctx context.Context, req *MarginAccountOrderRequest) (*MarginAccountNewOrderResponse, error) {
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
	symbolValue, err := e.FormatSymbol(req.Symbol, asset.Margin)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if req.IsIsolated {
		params.Set("isIsolated", "TRUE")
	}
	params.Set("side", req.Side)
	params.Set("type", req.OrderType)
	if req.Quantity != 0 {
		params.Set("quantity", strconv.FormatFloat(req.Quantity, 'f', -1, 64))
	}
	if req.QuoteOrderQuantity != 0 {
		params.Set("quoteOrderQty", strconv.FormatFloat(req.QuoteOrderQuantity, 'f', -1, 64))
	}
	if req.Price != 0 {
		params.Set("price", strconv.FormatFloat(req.Price, 'f', -1, 64))
	}
	if req.StopPrice != 0 {
		params.Set("stopPrice", strconv.FormatFloat(req.StopPrice, 'f', -1, 64))
	}
	if req.NewClientOrderID != "" {
		params.Set("newClientOrderId", req.NewClientOrderID)
	}
	if req.IcebergQuantity != 0 {
		params.Set("icebergQty", strconv.FormatFloat(req.IcebergQuantity, 'f', -1, 64))
	}
	if req.NewOrderResponseType != "" {
		params.Set("newOrderRespType", req.NewOrderResponseType)
	}
	if req.SideEffectType != "" {
		params.Set("sideEffectType", req.SideEffectType)
	}
	if req.TimeInForce != "" {
		params.Set("timeInForce", req.TimeInForce)
	}
	if req.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", req.SelfTradePreventionMode)
	}
	if req.TrailingDelta != 0 {
		params.Set("trailingDelta", strconv.FormatUint(req.TrailingDelta, 10))
	}
	if req.AutoRepayAtCancel != nil {
		params.Set("autoRepayAtCancel", strconv.FormatBool(*req.AutoRepayAtCancel))
	}
	var resp *MarginAccountNewOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/margin/order", params, marginNewOrderLimit(req.SideEffectType), &resp)
}

// CancelMarginAccountOrder cancels an active margin account order by OrderID or OriginalClientOrderID
func (e *Exchange) CancelMarginAccountOrder(ctx context.Context, req *MarginOrderCancelRequest) (*MarginOrderCancelResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.OrderID == 0 && req.OriginalClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	symbolValue, err := e.FormatSymbol(req.Symbol, asset.Margin)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if req.IsIsolated {
		params.Set("isIsolated", "TRUE")
	}
	if req.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	}
	if req.OriginalClientOrderID != "" {
		params.Set("origClientOrderId", req.OriginalClientOrderID)
	}
	if req.NewClientOrderID != "" {
		params.Set("newClientOrderId", req.NewClientOrderID)
	}
	var resp *MarginOrderCancelResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodDelete, "/sapi/v1/margin/order", params, marginAccountCancelOrderRate, &resp)
}

// NewMarginAccountOCOOrder places a new OCO order list in the margin account. Both legs share the quantity; a sell
// needs limit price > last price > stop price and a buy the reverse, and the list counts as two orders against the
// order rate limit. Binance weighs lists with a MARGIN_BUY or AUTO_BORROW_REPAY side effect at 1500 instead of 6, and
// from 2026-10-14 rejects lists without StopLimitPrice and StopLimitTimeInForce with -3116 on accounts that do not
// support STOP_LOSS orders
func (e *Exchange) NewMarginAccountOCOOrder(ctx context.Context, req *MarginOCOOrderRequest) (*MarginOCOOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if req.Quantity <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if req.Price <= 0 {
		return nil, limits.ErrPriceBelowMin
	}
	if req.StopPrice <= 0 {
		return nil, fmt.Errorf("%w: stop price", limits.ErrPriceBelowMin)
	}
	if req.StopLimitPrice != 0 && req.StopLimitTimeInForce == "" {
		return nil, fmt.Errorf("%w: stop limit time in force is required with a stop limit price", order.ErrInvalidTimeInForce)
	}
	symbolValue, err := e.FormatSymbol(req.Symbol, asset.Margin)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if req.IsIsolated {
		params.Set("isIsolated", "TRUE")
	}
	if req.ListClientOrderID != "" {
		params.Set("listClientOrderId", req.ListClientOrderID)
	}
	params.Set("side", req.Side)
	params.Set("quantity", strconv.FormatFloat(req.Quantity, 'f', -1, 64))
	if req.LimitClientOrderID != "" {
		params.Set("limitClientOrderId", req.LimitClientOrderID)
	}
	params.Set("price", strconv.FormatFloat(req.Price, 'f', -1, 64))
	if req.LimitIcebergQuantity != 0 {
		params.Set("limitIcebergQty", strconv.FormatFloat(req.LimitIcebergQuantity, 'f', -1, 64))
	}
	if req.StopClientOrderID != "" {
		params.Set("stopClientOrderId", req.StopClientOrderID)
	}
	params.Set("stopPrice", strconv.FormatFloat(req.StopPrice, 'f', -1, 64))
	if req.StopLimitPrice != 0 {
		params.Set("stopLimitPrice", strconv.FormatFloat(req.StopLimitPrice, 'f', -1, 64))
	}
	if req.StopIcebergQuantity != 0 {
		params.Set("stopIcebergQty", strconv.FormatFloat(req.StopIcebergQuantity, 'f', -1, 64))
	}
	if req.StopLimitTimeInForce != "" {
		params.Set("stopLimitTimeInForce", req.StopLimitTimeInForce)
	}
	if req.NewOrderResponseType != "" {
		params.Set("newOrderRespType", req.NewOrderResponseType)
	}
	if req.SideEffectType != "" {
		params.Set("sideEffectType", req.SideEffectType)
	}
	if req.SelfTradePreventionMode != "" {
		params.Set("selfTradePreventionMode", req.SelfTradePreventionMode)
	}
	if req.AutoRepayAtCancel != nil {
		params.Set("autoRepayAtCancel", strconv.FormatBool(*req.AutoRepayAtCancel))
	}
	var resp *MarginOCOOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/margin/order/oco", params, marginNewOrderLimit(req.SideEffectType), &resp)
}

// MarginManualLiquidation liquidates the margin account of marginType, MARGIN or ISOLATED; ISOLATED requires symbol
// and Binance supports it only in restricted regions
func (e *Exchange) MarginManualLiquidation(ctx context.Context, marginType string, symbol currency.Pair) (*MarginManualLiquidationResponse, error) {
	if marginType == "" {
		return nil, margin.ErrInvalidMarginType
	}
	if marginType == "ISOLATED" && symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("type", marginType)
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.Margin)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp *MarginManualLiquidationResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, "/sapi/v1/margin/manual-liquidation", params, marginManualLiquidationRate, &resp)
}

// GetCurrentMarginOrderCountUsage returns the margin account's current order count usage for every rate limit
// interval
func (e *Exchange) GetCurrentMarginOrderCountUsage(ctx context.Context, symbol currency.Pair, isIsolated bool) ([]MarginOrderCountUsage, error) {
	params := url.Values{}
	if isIsolated {
		params.Set("isIsolated", "TRUE")
	}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.Margin)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp []MarginOrderCountUsage
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/rateLimit/order", params, marginCurrentOrderCountUsageRate, &resp)
}

// GetMarginAccountAllOCO returns the margin account's order lists
func (e *Exchange) GetMarginAccountAllOCO(ctx context.Context, req *MarginAccountAllOCORequest) ([]MarginOrderListResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if req.IsIsolated {
		params.Set("isIsolated", "TRUE")
	}
	if !req.Symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(req.Symbol, asset.Margin)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
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
	var resp []MarginOrderListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/allOrderList", params, getMarginAccountAllOCORate, &resp)
}

// GetMarginAccountAllOrders returns the margin account's orders on a symbol from OrderID onwards, or from the last 24
// hours when OrderID is 0; windows must be shorter than 24 hours
func (e *Exchange) GetMarginAccountAllOrders(ctx context.Context, req *MarginAccountAllOrdersRequest) ([]MarginTradeOrderResponse, error) {
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
	symbolValue, err := e.FormatSymbol(req.Symbol, asset.Margin)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if req.IsIsolated {
		params.Set("isIsolated", "TRUE")
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
	var resp []MarginTradeOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/allOrders", params, marginAccountsAllOrdersRate, &resp)
}

// GetMarginAccountsOpenOCOOrder returns the margin account's open OCO order lists
func (e *Exchange) GetMarginAccountsOpenOCOOrder(ctx context.Context, symbol currency.Pair, isIsolated bool) ([]MarginOrderListResponse, error) {
	params := url.Values{}
	if isIsolated {
		params.Set("isIsolated", "TRUE")
	}
	if !symbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(symbol, asset.Margin)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbolValue)
	}
	var resp []MarginOrderListResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/openOrderList", params, marginAccountOpenOCOOrdersRate, &resp)
}

// GetMarginAccountTradeList returns the margin account's trades on a symbol from FromID onwards, or from the last 24
// hours when FromID is 0; windows must be shorter than 24 hours
func (e *Exchange) GetMarginAccountTradeList(ctx context.Context, req *MarginAccountTradeListRequest) ([]MarginAccountTrade, error) {
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
	symbolValue, err := e.FormatSymbol(req.Symbol, asset.Margin)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	if req.IsIsolated {
		params.Set("isIsolated", "TRUE")
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
	var resp []MarginAccountTrade
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/myTrades", params, marginAccountTradeListRate, &resp)
}

// GetCrossMarginTransferHistory returns the cross margin transfer history, newest first. Windows may span up to 30
// days and the last 7 days are returned by default
func (e *Exchange) GetCrossMarginTransferHistory(ctx context.Context, req *CrossMarginTransferHistoryRequest) (*CrossMarginTransferHistoryResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if !req.StartTime.IsZero() && !req.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(req.StartTime, req.EndTime); err != nil {
			return nil, err
		}
	}
	params := url.Values{}
	if !req.Asset.IsEmpty() {
		params.Set("asset", req.Asset.Upper().String())
	}
	if req.Type != "" {
		params.Set("type", req.Type)
	}
	if !req.StartTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(req.StartTime.UnixMilli(), 10))
	}
	if !req.EndTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(req.EndTime.UnixMilli(), 10))
	}
	if req.Current != 0 {
		params.Set("current", strconv.FormatUint(req.Current, 10))
	}
	if req.Size != 0 {
		params.Set("size", strconv.FormatUint(req.Size, 10))
	}
	if !req.IsolatedSymbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(req.IsolatedSymbol, asset.Margin)
		if err != nil {
			return nil, err
		}
		params.Set("isolatedSymbol", symbolValue)
	}
	var resp *CrossMarginTransferHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/transfer", params, sapiDefaultRate, &resp)
}

// GetMaxTransferOutAmount returns the maximum amount of an asset that can be transferred out of the cross margin
// account, or out of the isolated margin account of isolatedSymbol
func (e *Exchange) GetMaxTransferOutAmount(ctx context.Context, assetName currency.Code, isolatedSymbol currency.Pair) (*MaxTransferOutAmountResponse, error) {
	if assetName.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("asset", assetName.Upper().String())
	if !isolatedSymbol.IsEmpty() {
		symbolValue, err := e.FormatSymbol(isolatedSymbol, asset.Margin)
		if err != nil {
			return nil, err
		}
		params.Set("isolatedSymbol", symbolValue)
	}
	var resp *MaxTransferOutAmountResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/sapi/v1/margin/maxTransferable", params, maxTransferOutRate, &resp)
}

// CreateMarginListenToken creates a listen token that subscribes a WebSocket API connection to the margin account's
// User Data Stream with WsSubscribeUserDataStreamWithListenToken (Create Margin Account listenToken)
func (e *Exchange) CreateMarginListenToken(ctx context.Context, arg *MarginListenTokenRequest) (*MarginListenTokenResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.IsIsolated && arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if !arg.Symbol.IsEmpty() {
		symbol, err := e.FormatSymbol(arg.Symbol, asset.Margin)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", symbol)
	}
	if arg.IsIsolated {
		params.Set("isIsolated", "true")
	}
	if arg.Validity > 0 {
		params.Set("validity", strconv.FormatInt(arg.Validity.Milliseconds(), 10))
	}
	var resp *MarginListenTokenResponse
	return resp, e.SendAPIKeyHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodPost, common.EncodeURLValues("/sapi/v1/userListenToken", params), sapiUIDDefaultRate, &resp)
}
