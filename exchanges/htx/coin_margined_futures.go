package htx

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// QuerySwapIndexPriceInfo gets perpetual swap index's price info
func (e *Exchange) QuerySwapIndexPriceInfo(ctx context.Context, code currency.Pair) (SwapIndexPriceData, error) {
	var resp SwapIndexPriceData
	path := "/swap-api/v1/swap_index"
	if !code.IsEmpty() {
		codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		params := url.Values{}
		params.Set("contract_code", codeValue)
		path = common.EncodeURLValues(path, params)
	}
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSwapPriceLimits gets price caps for perpetual futures
func (e *Exchange) GetSwapPriceLimits(ctx context.Context, code currency.Pair) (SwapPriceLimitsData, error) {
	var resp SwapPriceLimitsData
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	path := common.EncodeURLValues("/swap-api/v1/swap_price_limit", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// SwapOpenInterestInformation gets open interest data for perpetual futures
func (e *Exchange) SwapOpenInterestInformation(ctx context.Context, code currency.Pair) (SwapOpenInterestData, error) {
	var resp SwapOpenInterestData
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	if !code.IsEmpty() {
		params.Set("contract_code", codeValue)
	}
	path := common.EncodeURLValues("/swap-api/v1/swap_open_interest", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSwapMarketDepth gets market depth for perpetual futures
func (e *Exchange) GetSwapMarketDepth(ctx context.Context, code currency.Pair, dataType string) (SwapMarketDepthData, error) {
	var resp SwapMarketDepthData
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("type", dataType)
	path := common.EncodeURLValues("/swap-ex/market/depth", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSwapKlineData gets kline data for perpetual futures
func (e *Exchange) GetSwapKlineData(ctx context.Context, code currency.Pair, period string, size int64, startTime, endTime time.Time) (SwapKlineData, error) {
	var resp SwapKlineData
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, errInvalidPeriod
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errStartTimeAfterEndTime
		}
		params.Set("from", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("to", strconv.FormatInt(endTime.Unix(), 10))
	}
	path := common.EncodeURLValues("/swap-ex/market/history/kline", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSwapMarketOverview gets market data overview for perpetual futures
func (e *Exchange) GetSwapMarketOverview(ctx context.Context, code currency.Pair) (MarketOverviewData, error) {
	var resp MarketOverviewData
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	path := common.EncodeURLValues("/swap-ex/market/detail/merged", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetLastTrade gets the last trade for a given perpetual contract
func (e *Exchange) GetLastTrade(ctx context.Context, code currency.Pair) (LastTradeData, error) {
	var resp LastTradeData
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	path := common.EncodeURLValues("/swap-ex/market/trade", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetBatchTrades gets batch trades for a specified contract (fetching size cannot be bigger than 2000)
func (e *Exchange) GetBatchTrades(ctx context.Context, code currency.Pair, size int64) (BatchTradesData, error) {
	var resp BatchTradesData
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("size", strconv.FormatInt(size, 10))
	path := common.EncodeURLValues("/swap-ex/market/history/trade", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetTieredAjustmentFactorInfo gets tiered adjustment factor data
func (e *Exchange) GetTieredAjustmentFactorInfo(ctx context.Context, code currency.Pair) (TieredAdjustmentFactorData, error) {
	var resp TieredAdjustmentFactorData
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	path := common.EncodeURLValues("/swap-api/v1/swap_adjustfactor", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetOpenInterestInfo gets open interest data
func (e *Exchange) GetOpenInterestInfo(ctx context.Context, code currency.Pair, period, amountType string, size int64) (OpenInterestData, error) {
	var resp OpenInterestData
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, errInvalidPeriod
	}
	if size <= 0 || size > 1200 {
		return resp, errInvalidSize
	}
	aType, ok := validAmountType[amountType]
	if !ok {
		return resp, errInvalidTradeType
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	params.Set("size", strconv.FormatInt(size, 10))
	params.Set("amount_type", strconv.FormatInt(aType, 10))
	path := common.EncodeURLValues("/swap-api/v1/swap_his_open_interest", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSystemStatusInfo gets system status data
func (e *Exchange) GetSystemStatusInfo(ctx context.Context, code currency.Pair) (SystemStatusData, error) {
	var resp SystemStatusData
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	path := common.EncodeURLValues("/swap-api/v1/swap_api_state", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetTraderSentimentIndexAccount gets top trader sentiment function-account
func (e *Exchange) GetTraderSentimentIndexAccount(ctx context.Context, code currency.Pair, period string) (TraderSentimentIndexAccountData, error) {
	var resp TraderSentimentIndexAccountData
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, errInvalidPeriod
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	path := common.EncodeURLValues("/swap-api/v1/swap_elite_account_ratio", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetTraderSentimentIndexPosition gets top trader sentiment function-position
func (e *Exchange) GetTraderSentimentIndexPosition(ctx context.Context, code currency.Pair, period string) (TraderSentimentIndexPositionData, error) {
	var resp TraderSentimentIndexPositionData
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}

	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, errInvalidPeriod
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	path := common.EncodeURLValues("/swap-api/v1/swap_elite_position_ratio", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetLiquidationOrders gets liquidation orders for a given perp
func (e *Exchange) GetLiquidationOrders(ctx context.Context, contract currency.Pair, tradeType string, startTime, endTime time.Time, direction string, fromID int64) (LiquidationOrdersData, error) {
	var resp LiquidationOrdersData
	formattedContract, err := e.FormatSymbol(contract, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	tType, ok := validTradeTypes[tradeType]
	if !ok {
		return resp, errInvalidTradeType
	}
	params := url.Values{}
	params.Set("contract", formattedContract)
	params.Set("trade_type", strconv.FormatInt(tType, 10))

	if !startTime.IsZero() {
		params.Set("start_time", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("end_time", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if direction != "" {
		params.Set("direct", direction)
	}
	if fromID != 0 {
		params.Set("from_id", strconv.FormatInt(fromID, 10))
	}
	path := common.EncodeURLValues("/swap-api/v3/swap_liquidation_orders", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetHistoricalFundingRatesForPair gets historical funding rates for perpetual futures
func (e *Exchange) GetHistoricalFundingRatesForPair(ctx context.Context, code currency.Pair, pageSize, pageIndex int64) (HistoricalFundingRateData, error) {
	var resp HistoricalFundingRateData
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	if pageIndex != 0 {
		params.Set("page_index", strconv.FormatInt(pageIndex, 10))
	}
	if pageSize != 0 {
		params.Set("page_size", strconv.FormatInt(pageSize, 10))
	}
	path := common.EncodeURLValues("/swap-api/v1/swap_historical_funding_rate", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetPremiumIndexKlineData gets kline data for premium index
func (e *Exchange) GetPremiumIndexKlineData(ctx context.Context, code currency.Pair, period string, size int64) (PremiumIndexKlineData, error) {
	var resp PremiumIndexKlineData
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, errInvalidPeriod
	}
	if size <= 0 || size > 1200 {
		return resp, errInvalidSize
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("size", strconv.FormatInt(size, 10))
	params.Set("period", period)
	path := common.EncodeURLValues("/index/market/history/swap_premium_index_kline", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetEstimatedFundingRates gets estimated funding rates for perpetual futures
func (e *Exchange) GetEstimatedFundingRates(ctx context.Context, code currency.Pair, period string, size int64) (EstimatedFundingRateData, error) {
	var resp EstimatedFundingRateData
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, errInvalidPeriod
	}
	if size <= 0 || size > 1200 {
		return resp, errInvalidSize
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	params.Set("size", strconv.FormatInt(size, 10))
	path := common.EncodeURLValues("/index/market/history/swap_estimated_rate_kline", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetBasisData gets basis data for perpetual futures
func (e *Exchange) GetBasisData(ctx context.Context, code currency.Pair, period, basisPriceType string, size int64) (BasisData, error) {
	var resp BasisData
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, errInvalidPeriod
	}
	if size <= 0 || size > 1200 {
		return resp, errInvalidSize
	}
	if !common.StringSliceCompareInsensitive(validBasisPriceTypes, basisPriceType) {
		return resp, errInvalidPeriod
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	params.Set("size", strconv.FormatInt(size, 10))
	path := common.EncodeURLValues("/index/market/history/swap_basis", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSwapAccountInfo gets swap account info
func (e *Exchange) GetSwapAccountInfo(ctx context.Context, code currency.Pair) (SwapAccountInformation, error) {
	var resp SwapAccountInformation
	req := make(map[string]any)
	if !code.IsEmpty() {
		codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_account_info", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSwapPositionsInfo gets swap positions' info
func (e *Exchange) GetSwapPositionsInfo(ctx context.Context, code currency.Pair) (SwapPositionInfo, error) {
	var resp SwapPositionInfo
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_position_info", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSwapAssetsAndPositions gets swap positions and asset info
func (e *Exchange) GetSwapAssetsAndPositions(ctx context.Context, code currency.Pair) (SwapAssetsAndPositionsData, error) {
	var resp SwapAssetsAndPositionsData
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_account_position_info", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSwapAllSubAccAssets gets asset info for all subaccounts
func (e *Exchange) GetSwapAllSubAccAssets(ctx context.Context, code currency.Pair) (SubAccountsAssetData, error) {
	var resp SubAccountsAssetData
	req := make(map[string]any)
	if !code.IsEmpty() {
		codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_sub_account_list", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// SwapSingleSubAccAssets gets a subaccount's assets info
func (e *Exchange) SwapSingleSubAccAssets(ctx context.Context, code currency.Pair, subUID int64) (SingleSubAccountAssetsInfo, error) {
	var resp SingleSubAccountAssetsInfo
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	req["sub_uid"] = subUID
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_sub_account_info", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSubAccPositionInfo gets a subaccount's positions info
func (e *Exchange) GetSubAccPositionInfo(ctx context.Context, code currency.Pair, subUID int64) (SingleSubAccountPositionsInfo, error) {
	var resp SingleSubAccountPositionsInfo
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	req["sub_uid"] = subUID
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_sub_position_info", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetAccountFinancialRecords gets the account's financial records
func (e *Exchange) GetAccountFinancialRecords(ctx context.Context, code currency.Pair, orderType string, createDate, pageIndex, pageSize int64) (FinancialRecordData, error) {
	var resp FinancialRecordData
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract"] = codeValue
	if orderType != "" {
		req["type"] = orderType
	}
	if createDate != 0 {
		if err := addV3HistoryTimeRange(req, createDate); err != nil {
			return resp, err
		}
	}
	if pageIndex != 0 {
		req["from_id"] = pageIndex
	}
	if pageSize != 0 {
		req["limit"] = pageSize
	}
	req["direct"] = v3HistoryDirectionNext
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v3/swap_financial_record", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSwapSettlementRecords gets the swap account's settlement records
func (e *Exchange) GetSwapSettlementRecords(ctx context.Context, code currency.Pair, startTime, endTime time.Time, pageIndex, pageSize int64) (FinancialRecordData, error) {
	var resp FinancialRecordData
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errStartTimeAfterEndTime
		}
		req["start_time"] = strconv.FormatInt(startTime.UnixMilli(), 10)
		req["end_time"] = strconv.FormatInt(endTime.UnixMilli(), 10)
	}
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_user_settlement_records", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetAvailableLeverage gets user's available leverage data
func (e *Exchange) GetAvailableLeverage(ctx context.Context, code currency.Pair) (AvailableLeverageData, error) {
	var resp AvailableLeverageData
	req := make(map[string]any)
	if !code.IsEmpty() {
		codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_available_level_rate", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// SwitchCoinMarginedLeverage changes the leverage used by a coin-margined perpetual contract.
func (e *Exchange) SwitchCoinMarginedLeverage(ctx context.Context, code currency.Pair, leverage uint64) error {
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return err
	}
	req := &SwitchCoinMarginedLeverageRequest{
		ContractCode: codeValue,
		LeverageRate: leverage,
	}
	var resp *SwitchCoinMarginedLeverageResponse
	return e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_switch_lever_rate", nil, req, &resp)
}

// GetSwapOrderLimitInfo gets order limit info for swaps
func (e *Exchange) GetSwapOrderLimitInfo(ctx context.Context, code currency.Pair, orderType string) (SwapOrderLimitInfo, error) {
	var resp SwapOrderLimitInfo
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	if !common.StringSliceCompareInsensitive(validOrderTypes, orderType) {
		return resp, errInvalidOrderType
	}
	req["order_price_type"] = orderType
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_order_limit", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSwapTradingFeeInfo gets trading fee info for swaps
func (e *Exchange) GetSwapTradingFeeInfo(ctx context.Context, code currency.Pair) (SwapTradingFeeData, error) {
	var resp SwapTradingFeeData
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_fee", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSwapTransferLimitInfo gets transfer limit info for swaps
func (e *Exchange) GetSwapTransferLimitInfo(ctx context.Context, code currency.Pair) (TransferLimitData, error) {
	var resp TransferLimitData
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_transfer_limit", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSwapPositionLimitInfo gets transfer limit info for swaps
func (e *Exchange) GetSwapPositionLimitInfo(ctx context.Context, code currency.Pair) (PositionLimitData, error) {
	var resp PositionLimitData
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_position_limit", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// AccountTransferData gets asset transfer data between master and subaccounts
func (e *Exchange) AccountTransferData(ctx context.Context, code currency.Pair, subUID, transferType string, amount float64) (InternalAccountTransferData, error) {
	var resp InternalAccountTransferData
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	req["subUid"] = subUID
	req["amount"] = amount
	if !common.StringSliceCompareInsensitive(validTransferType, transferType) {
		return resp, errInvalidTransferType
	}
	req["type"] = transferType
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_master_sub_transfer", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// AccountTransferRecords gets asset transfer records between master and subaccounts
func (e *Exchange) AccountTransferRecords(ctx context.Context, code currency.Pair, transferType string, createDate, pageIndex, pageSize int64) (InternalAccountTransferData, error) {
	var resp InternalAccountTransferData
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	if !common.StringSliceCompareInsensitive(validTransferType, transferType) {
		return resp, errInvalidTransferType
	}
	req["type"] = transferType
	if createDate > 90 {
		return resp, errInvalidCreateDate
	}
	req["create_date"] = strconv.FormatInt(createDate, 10)
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize > 0 && pageSize <= 50 {
		req["page_size"] = pageSize
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_master_sub_transfer_record", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// PlaceSwapOrders places orders for swaps
func (e *Exchange) PlaceSwapOrders(ctx context.Context, code currency.Pair, clientOrderID, direction, offset, orderPriceType string, price, volume, leverage float64) (SwapOrderData, error) {
	var resp SwapOrderData
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	if clientOrderID != "" {
		req["client_order_id"] = clientOrderID
	}
	req["direction"] = direction
	req["offset"] = offset
	if !common.StringSliceCompareInsensitive(validOrderTypes, orderPriceType) {
		return resp, errInvalidOrderType
	}
	req["order_price_type"] = orderPriceType
	req["price"] = price
	req["volume"] = volume
	req["lever_rate"] = leverage
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_order", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// PlaceSwapBatchOrders places a batch of orders for swaps
func (e *Exchange) PlaceSwapBatchOrders(ctx context.Context, data BatchOrderRequestType) (BatchOrderData, error) {
	var resp BatchOrderData
	req := make(map[string]any)
	if len(data.Data) > 10 || len(data.Data) == 0 {
		return resp, errBatchOrderLimitExceeded
	}
	for x := range data.Data {
		if data.Data[x].ContractCode == "" {
			continue
		}
		unformattedPair, err := currency.NewPairFromString(data.Data[x].ContractCode)
		if err != nil {
			return resp, err
		}
		codeValue, err := e.FormatSymbol(unformattedPair, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		data.Data[x].ContractCode = codeValue
	}
	req["orders_data"] = data.Data
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_batchorder", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// CancelSwapOrder sends a request to cancel an order
func (e *Exchange) CancelSwapOrder(ctx context.Context, orderID, clientOrderID string, contractCode currency.Pair) (CancelOrdersData, error) {
	var resp CancelOrdersData
	req := make(map[string]any)
	if orderID != "" {
		req["order_id"] = orderID
	}
	if clientOrderID != "" {
		req["client_order_id"] = clientOrderID
	}
	req["contract_code"] = contractCode
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_cancel", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// CancelAllSwapOrders sends a request to cancel an order
func (e *Exchange) CancelAllSwapOrders(ctx context.Context, contractCode currency.Pair) (CancelOrdersData, error) {
	var resp CancelOrdersData
	req := make(map[string]any)
	req["contract_code"] = contractCode
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_cancelall", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// PlaceLightningCloseOrder places a lightning close order
func (e *Exchange) PlaceLightningCloseOrder(ctx context.Context, contractCode currency.Pair, direction, orderPriceType string, volume float64, clientOrderID int64) (LightningCloseOrderData, error) {
	var resp LightningCloseOrderData
	req := make(map[string]any)
	req["contract_code"] = contractCode
	req["volume"] = volume
	req["direction"] = direction
	if clientOrderID != 0 {
		req["client_order_id"] = clientOrderID
	}
	if orderPriceType != "" {
		if !common.StringSliceCompareInsensitive(validLightningOrderPriceType, orderPriceType) {
			return resp, errInvalidOrderPriceType
		}
		req["order_price_type"] = orderPriceType
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_lightning_close_position", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSwapOrderDetails gets order info
func (e *Exchange) GetSwapOrderDetails(ctx context.Context, contractCode currency.Pair, orderID, createdAt, orderType string, pageIndex, pageSize int64) (SwapOrderData, error) {
	var resp SwapOrderData
	req := make(map[string]any)
	req["contract_code"] = contractCode
	req["order_id"] = orderID
	req["created_at"] = createdAt
	oType, ok := validOrderType[orderType]
	if !ok {
		return resp, errInvalidOrderType
	}
	req["order_type"] = oType
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize > 0 && pageSize <= 50 {
		req["page_size"] = pageSize
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_order_detail", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSwapOrderInfo gets info on a swap order
func (e *Exchange) GetSwapOrderInfo(ctx context.Context, contractCode currency.Pair, orderID, clientOrderID string) (SwapOrderInfo, error) {
	var resp SwapOrderInfo
	req := make(map[string]any)
	if !contractCode.IsEmpty() {
		codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	if orderID != "" {
		req["order_id"] = orderID
	}
	if clientOrderID != "" {
		req["client_order_id"] = clientOrderID
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_order_info", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSwapOpenOrders gets open orders for swap
func (e *Exchange) GetSwapOpenOrders(ctx context.Context, contractCode currency.Pair, pageIndex, pageSize int64) (SwapOpenOrdersData, error) {
	var resp SwapOpenOrdersData
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize > 0 && pageSize <= 50 {
		req["page_size"] = pageSize
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_openorders", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSwapOrderHistory gets swap order history using a lookback of at most two days.
func (e *Exchange) GetSwapOrderHistory(ctx context.Context, contractCode currency.Pair, tradeType, reqType string, status []order.Status, lookbackDays, pageIndex, pageSize int64) (SwapOrderHistory, error) {
	if lookbackDays < 0 || lookbackDays > 2 {
		return SwapOrderHistory{}, errInvalidCreateDate
	}
	var startTime, endTime time.Time
	if lookbackDays != 0 {
		endTime = time.Now().UTC()
		startTime = endTime.AddDate(0, 0, -int(lookbackDays))
	}
	return e.GetSwapOrderHistoryByTimeRange(ctx, contractCode, tradeType, reqType, status, startTime, endTime, pageIndex, pageSize)
}

// GetSwapOrderHistoryByTimeRange gets swap order history for an explicit interval.
func (e *Exchange) GetSwapOrderHistoryByTimeRange(ctx context.Context, contractCode currency.Pair, tradeType, reqType string, status []order.Status, startTime, endTime time.Time, pageIndex, pageSize int64) (SwapOrderHistory, error) {
	var resp SwapOrderHistory
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract"] = codeValue
	tType, ok := validFuturesTradeType[tradeType]
	if !ok {
		return resp, errInvalidTradeType
	}
	req["trade_type"] = tType
	rType, ok := validFuturesReqType[reqType]
	if !ok {
		return resp, errInvalidRequestType
	}
	req["type"] = rType
	reqStatus := "0"
	if len(status) > 0 {
		firstTime := true
		for x := range status {
			sType, ok := validOrderStatus[status[x]]
			if !ok {
				return resp, errInvalidOrderStatus
			}
			if firstTime {
				firstTime = false
				reqStatus = strconv.FormatInt(sType, 10)
				continue
			}
			reqStatus = reqStatus + "," + strconv.FormatInt(sType, 10)
		}
	}
	req["status"] = reqStatus
	if startTime.IsZero() != endTime.IsZero() {
		return resp, errInvalidCreateDate
	}
	if !startTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errStartTimeAfterEndTime
		}
		if endTime.Sub(startTime) > 48*time.Hour {
			return resp, errHistoryTimeRangeExceeded
		}
		req["start_time"] = startTime.UTC().UnixMilli()
		req["end_time"] = endTime.UTC().UnixMilli()
	}
	req["direct"] = v3HistoryDirectionNext
	if pageIndex != 0 {
		req["from_id"] = pageIndex
	}
	if pageSize != 0 {
		req["limit"] = pageSize
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v3/swap_hisorders", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSwapTradeHistory gets swap trade history
func (e *Exchange) GetSwapTradeHistory(ctx context.Context, contractCode currency.Pair, tradeType string, lookbackDays, pageIndex, pageSize int64) (AccountTradeHistoryData, error) {
	var resp AccountTradeHistoryData
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract"] = codeValue
	if lookbackDays < 0 {
		return resp, errInvalidCreateDate
	}
	tType, ok := validTradeType[tradeType]
	if !ok {
		return resp, errInvalidTradeType
	}
	req["trade_type"] = tType
	if err := addV3HistoryTimeRange(req, lookbackDays); err != nil {
		return resp, err
	}
	req["direct"] = v3HistoryDirectionNext
	if pageIndex != 0 {
		req["from_id"] = pageIndex
	}
	if pageSize != 0 {
		req["limit"] = pageSize
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v3/swap_matchresults", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// PlaceSwapTriggerOrder places a trigger order for a swap
func (e *Exchange) PlaceSwapTriggerOrder(ctx context.Context, contractCode currency.Pair, triggerType, direction, offset, orderPriceType string, triggerPrice, orderPrice, volume, leverageRate float64) (AccountTradeHistoryData, error) {
	var resp AccountTradeHistoryData
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	tType, ok := validTriggerType[triggerType]
	if !ok {
		return resp, errInvalidTriggerType
	}
	req["trigger_type"] = tType
	req["direction"] = direction
	req["offset"] = offset
	req["trigger_price"] = triggerPrice
	req["volume"] = volume
	req["lever_rate"] = leverageRate
	req["order_price"] = orderPrice
	if !common.StringSliceCompareInsensitive(validOrderPriceType, orderPriceType) {
		return resp, errInvalidOrderPriceType
	}
	req["order_price_type"] = orderPriceType
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_trigger_order", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// CancelSwapTriggerOrder cancels swap trigger order
func (e *Exchange) CancelSwapTriggerOrder(ctx context.Context, contractCode currency.Pair, orderID string) (CancelTriggerOrdersData, error) {
	var resp CancelTriggerOrdersData
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	req["order_id"] = orderID
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_trigger_cancel", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// CancelAllSwapTriggerOrders cancels all swap trigger orders
func (e *Exchange) CancelAllSwapTriggerOrders(ctx context.Context, contractCode currency.Pair) (CancelTriggerOrdersData, error) {
	var resp CancelTriggerOrdersData
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_trigger_cancelall", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSwapTriggerOrderHistory gets history for swap trigger orders
func (e *Exchange) GetSwapTriggerOrderHistory(ctx context.Context, contractCode currency.Pair, status, tradeType string, createDate, pageIndex, pageSize int64) (TriggerOrderHistory, error) {
	var resp TriggerOrderHistory
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	req["status"] = status
	tType, ok := validTradeType[tradeType]
	if !ok {
		return resp, errInvalidTradeType
	}
	req["trade_type"] = tType
	if createDate > 90 {
		return resp, errInvalidCreateDate
	}
	req["create_date"] = strconv.FormatInt(createDate, 10)
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize > 0 && pageSize <= 50 {
		req["page_size"] = pageSize
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_trigger_hisorders", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSwapMarkets gets data of swap markets
func (e *Exchange) GetSwapMarkets(ctx context.Context, contract currency.Pair) ([]SwapMarketsData, error) {
	vals := url.Values{}
	if !contract.IsEmpty() {
		codeValue, err := e.FormatSymbol(contract, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		vals.Set("contract_code", codeValue)
	}
	type response struct {
		Response
		Data []SwapMarketsData `json:"data"`
	}
	var result response
	err := e.SendHTTPRequest(ctx, exchange.RestFutures, "/swap-api/v1/swap_contract_info"+"?"+vals.Encode(), &result)
	if result.ErrorMessage != "" {
		return nil, htxError(result.ErrorMessage)
	}
	return result.Data, err
}

// GetSwapFundingRate gets funding rate data for one currency
func (e *Exchange) GetSwapFundingRate(ctx context.Context, contract currency.Pair) (FundingRatesData, error) {
	vals := url.Values{}
	codeValue, err := e.FormatSymbol(contract, asset.CoinMarginedFutures)
	if err != nil {
		return FundingRatesData{}, err
	}
	vals.Set("contract_code", codeValue)
	type response struct {
		Response
		Data FundingRatesData `json:"data"`
	}
	var result response
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, "/swap-api/v1/swap_funding_rate"+"?"+vals.Encode(), &result)
	if result.ErrorMessage != "" {
		return FundingRatesData{}, htxError(result.ErrorMessage)
	}
	return result.Data, err
}

// GetSwapFundingRates gets funding rates data
func (e *Exchange) GetSwapFundingRates(ctx context.Context) (SwapFundingRatesResponse, error) {
	var result SwapFundingRatesResponse
	err := e.SendHTTPRequest(ctx, exchange.RestFutures, "/swap-api/v1/swap_batch_funding_rate", &result)
	return result, err
}
