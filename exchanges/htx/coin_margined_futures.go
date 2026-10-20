package htx

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// QuerySwapIndexPriceInfo gets perpetual swap index's price info
func (e *Exchange) QuerySwapIndexPriceInfo(ctx context.Context, code currency.Pair) (SwapIndexPriceResponse, error) {
	var resp SwapIndexPriceResponse
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
func (e *Exchange) GetSwapPriceLimits(ctx context.Context, code currency.Pair) (SwapPriceLimitsResponse, error) {
	var resp SwapPriceLimitsResponse
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
func (e *Exchange) SwapOpenInterestInformation(ctx context.Context, code currency.Pair) (SwapOpenInterestResponse, error) {
	var resp SwapOpenInterestResponse
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
func (e *Exchange) GetSwapMarketDepth(ctx context.Context, code currency.Pair, dataType string) (SwapMarketDepthResponse, error) {
	var resp SwapMarketDepthResponse
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
func (e *Exchange) GetSwapKlineData(ctx context.Context, code currency.Pair, period string, size uint64, startTime, endTime time.Time) (SwapKlineResponse, error) {
	var resp SwapKlineResponse
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, common.ErrInvalidPeriod
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	if size > 0 {
		params.Set("size", strconv.FormatUint(size, 10))
	}
	if !startTime.IsZero() || !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return resp, err
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
func (e *Exchange) GetSwapMarketOverview(ctx context.Context, code currency.Pair) (SwapMarketOverviewResponse, error) {
	var resp SwapMarketOverviewResponse
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
func (e *Exchange) GetLastTrade(ctx context.Context, code currency.Pair) (SwapLastTradeResponse, error) {
	var resp SwapLastTradeResponse
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
func (e *Exchange) GetBatchTrades(ctx context.Context, code currency.Pair, size uint64) (SwapTradesResponse, error) {
	var resp SwapTradesResponse
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("size", strconv.FormatUint(size, 10))
	path := common.EncodeURLValues("/swap-ex/market/history/trade", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetTieredAjustmentFactorInfo gets tiered adjustment factor data
func (e *Exchange) GetTieredAjustmentFactorInfo(ctx context.Context, code currency.Pair) (SwapTieredAdjustmentFactorsResponse, error) {
	var resp SwapTieredAdjustmentFactorsResponse
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
func (e *Exchange) GetOpenInterestInfo(ctx context.Context, code currency.Pair, period, amountType string, size uint64) (SwapHistoricalOpenInterestResponse, error) {
	var resp SwapHistoricalOpenInterestResponse
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, common.ErrInvalidPeriod
	}
	if size == 0 || size > 1200 {
		return resp, errInvalidSize
	}
	aType, ok := validAmountType[amountType]
	if !ok {
		return resp, errInvalidAmountType
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	params.Set("size", strconv.FormatUint(size, 10))
	params.Set("amount_type", strconv.FormatUint(aType, 10))
	path := common.EncodeURLValues("/swap-api/v1/swap_his_open_interest", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSystemStatusInfo gets system status data
func (e *Exchange) GetSystemStatusInfo(ctx context.Context, code currency.Pair) (SwapSystemStatusResponse, error) {
	var resp SwapSystemStatusResponse
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
func (e *Exchange) GetTraderSentimentIndexAccount(ctx context.Context, code currency.Pair, period string) (SwapAccountSentimentResponse, error) {
	var resp SwapAccountSentimentResponse
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, common.ErrInvalidPeriod
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
func (e *Exchange) GetTraderSentimentIndexPosition(ctx context.Context, code currency.Pair, period string) (SwapPositionSentimentResponse, error) {
	var resp SwapPositionSentimentResponse
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}

	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, common.ErrInvalidPeriod
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
func (e *Exchange) GetLiquidationOrders(ctx context.Context, contract currency.Pair, tradeType string, startTime, endTime time.Time, direction string, fromID uint64) (SwapLiquidationOrdersResponse, error) {
	var resp SwapLiquidationOrdersResponse
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
	params.Set("trade_type", strconv.FormatUint(tType, 10))

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
		params.Set("from_id", strconv.FormatUint(fromID, 10))
	}
	path := common.EncodeURLValues("/swap-api/v3/swap_liquidation_orders", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetHistoricalFundingRatesForPair gets historical funding rates for perpetual futures
func (e *Exchange) GetHistoricalFundingRatesForPair(ctx context.Context, code currency.Pair, pageSize, pageIndex uint64) (SwapHistoricalFundingRatesResponse, error) {
	var resp SwapHistoricalFundingRatesResponse
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	if pageIndex != 0 {
		params.Set("page_index", strconv.FormatUint(pageIndex, 10))
	}
	if pageSize != 0 {
		params.Set("page_size", strconv.FormatUint(pageSize, 10))
	}
	path := common.EncodeURLValues("/swap-api/v1/swap_historical_funding_rate", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetPremiumIndexKlineData gets kline data for premium index
func (e *Exchange) GetPremiumIndexKlineData(ctx context.Context, code currency.Pair, period string, size uint64) (SwapPremiumIndexKlineResponse, error) {
	var resp SwapPremiumIndexKlineResponse
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, common.ErrInvalidPeriod
	}
	if size == 0 || size > 1200 {
		return resp, errInvalidSize
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("size", strconv.FormatUint(size, 10))
	params.Set("period", period)
	path := common.EncodeURLValues("/index/market/history/swap_premium_index_kline", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetEstimatedFundingRates gets estimated funding rates for perpetual futures
func (e *Exchange) GetEstimatedFundingRates(ctx context.Context, code currency.Pair, period string, size uint64) (SwapEstimatedFundingRateResponse, error) {
	var resp SwapEstimatedFundingRateResponse
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, common.ErrInvalidPeriod
	}
	if size == 0 || size > 1200 {
		return resp, errInvalidSize
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	params.Set("size", strconv.FormatUint(size, 10))
	path := common.EncodeURLValues("/index/market/history/swap_estimated_rate_kline", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetBasisData gets basis data for perpetual futures
func (e *Exchange) GetBasisData(ctx context.Context, code currency.Pair, period, basisPriceType string, size uint64) (SwapBasisResponse, error) {
	var resp SwapBasisResponse
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, common.ErrInvalidPeriod
	}
	if size == 0 || size > 1200 {
		return resp, errInvalidSize
	}
	if !common.StringSliceCompareInsensitive(validBasisPriceTypes, basisPriceType) {
		return resp, errInvalidBasisPriceType
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	params.Set("basis_price_type", basisPriceType)
	params.Set("size", strconv.FormatUint(size, 10))
	path := common.EncodeURLValues("/index/market/history/swap_basis", params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSwapAccountInfo gets swap account info
func (e *Exchange) GetSwapAccountInfo(ctx context.Context, code currency.Pair) (SwapAccountResponse, error) {
	var resp SwapAccountResponse
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
func (e *Exchange) GetSwapPositionsInfo(ctx context.Context, code currency.Pair) (SwapPositionsResponse, error) {
	var resp SwapPositionsResponse
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
func (e *Exchange) GetSwapAssetsAndPositions(ctx context.Context, code currency.Pair) (SwapAssetsAndPositionsResponse, error) {
	var resp SwapAssetsAndPositionsResponse
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
func (e *Exchange) GetSwapAllSubAccAssets(ctx context.Context, code currency.Pair) (SwapSubaccountAssetsResponse, error) {
	var resp SwapSubaccountAssetsResponse
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
func (e *Exchange) SwapSingleSubAccAssets(ctx context.Context, code currency.Pair, subUID uint64) (SwapSingleSubaccountAssetsResponse, error) {
	var resp SwapSingleSubaccountAssetsResponse
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
func (e *Exchange) GetSubAccPositionInfo(ctx context.Context, code currency.Pair, subUID uint64) (SwapSubaccountPositionsResponse, error) {
	var resp SwapSubaccountPositionsResponse
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
func (e *Exchange) GetAccountFinancialRecords(ctx context.Context, code currency.Pair, orderType string, lookbackDays, fromID, limit uint64) (SwapFinancialRecordsResponse, error) {
	var resp SwapFinancialRecordsResponse
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract"] = codeValue
	if orderType != "" {
		req["type"] = orderType
	}
	if lookbackDays != 0 {
		if err := addV3HistoryTimeRange(req, lookbackDays); err != nil {
			return resp, err
		}
	}
	if fromID != 0 {
		req["from_id"] = fromID
	}
	if limit != 0 {
		req[orderPriceTypeLimit] = limit
	}
	req["direct"] = v3HistoryDirectionNext
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v3/swap_financial_record", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSwapSettlementRecords gets the swap account's settlement records
func (e *Exchange) GetSwapSettlementRecords(ctx context.Context, code currency.Pair, startTime, endTime time.Time, pageIndex, pageSize uint64) (SwapFinancialRecordsResponse, error) {
	var resp SwapFinancialRecordsResponse
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	if !startTime.IsZero() || !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return resp, err
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
func (e *Exchange) GetAvailableLeverage(ctx context.Context, code currency.Pair) (SwapAvailableLeverageResponse, error) {
	var resp SwapAvailableLeverageResponse
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
func (e *Exchange) GetSwapOrderLimitInfo(ctx context.Context, code currency.Pair, orderType string) (SwapOrderLimitsResponse, error) {
	var resp SwapOrderLimitsResponse
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
func (e *Exchange) GetSwapTradingFeeInfo(ctx context.Context, code currency.Pair) (SwapTradingFeesResponse, error) {
	var resp SwapTradingFeesResponse
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
func (e *Exchange) GetSwapTransferLimitInfo(ctx context.Context, code currency.Pair) (SwapTransferLimitsResponse, error) {
	var resp SwapTransferLimitsResponse
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
func (e *Exchange) GetSwapPositionLimitInfo(ctx context.Context, code currency.Pair) (SwapPositionLimitsResponse, error) {
	var resp SwapPositionLimitsResponse
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
func (e *Exchange) AccountTransferData(ctx context.Context, code currency.Pair, subUID, transferType string, amount float64) (SwapInternalTransferResponse, error) {
	var resp SwapInternalTransferResponse
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
func (e *Exchange) AccountTransferRecords(ctx context.Context, code currency.Pair, transferType string, createDate, pageIndex, pageSize uint64) (SwapInternalTransferResponse, error) {
	var resp SwapInternalTransferResponse
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
	req["create_date"] = strconv.FormatUint(createDate, 10)
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
func (e *Exchange) PlaceSwapOrders(ctx context.Context, code currency.Pair, clientOrderID, direction, offset, orderPriceType string, price, volume, leverage float64) (SwapOrderResponse, error) {
	var resp SwapOrderResponse
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
func (e *Exchange) PlaceSwapBatchOrders(ctx context.Context, data SwapBatchOrderRequest) (SwapBatchOrderResponse, error) {
	var resp SwapBatchOrderResponse
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
func (e *Exchange) CancelSwapOrder(ctx context.Context, orderID, clientOrderID string, contractCode currency.Pair) (SwapCancelOrdersResponse, error) {
	var resp SwapCancelOrdersResponse
	if orderID == "" && clientOrderID == "" {
		return resp, order.ErrOrderIDNotSet
	}
	if orderID != "" && clientOrderID != "" {
		return resp, errMixedOrderIdentifiers
	}
	identifiers := orderID
	if identifiers == "" {
		identifiers = clientOrderID
	}
	if count := strings.Count(identifiers, ",") + 1; count > 10 {
		return resp, fmt.Errorf("%w: maximum 10, received %d", errCancelOrderLimitExceeded, count)
	}
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	if orderID != "" {
		req["order_id"] = orderID
	}
	if clientOrderID != "" {
		req["client_order_id"] = clientOrderID
	}
	req["contract_code"] = codeValue
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_cancel", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// CancelAllSwapOrders sends a request to cancel an order
func (e *Exchange) CancelAllSwapOrders(ctx context.Context, contractCode currency.Pair) (SwapCancelOrdersResponse, error) {
	var resp SwapCancelOrdersResponse
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_cancelall", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// PlaceLightningCloseOrder places a lightning close order
func (e *Exchange) PlaceLightningCloseOrder(ctx context.Context, contractCode currency.Pair, direction, orderPriceType string, volume float64, clientOrderID uint64) (SwapLightningCloseOrderResponse, error) {
	var resp SwapLightningCloseOrderResponse
	if clientOrderID > math.MaxInt64 {
		return resp, errInvalidClientOrderID
	}
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
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
func (e *Exchange) GetSwapOrderDetails(ctx context.Context, contractCode currency.Pair, orderID, createdAt, orderType string, pageIndex, pageSize uint64) (SwapOrderResponse, error) {
	var resp SwapOrderResponse
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
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
func (e *Exchange) GetSwapOrderInfo(ctx context.Context, contractCode currency.Pair, orderID, clientOrderID string) (SwapOrderInfoResponse, error) {
	var resp SwapOrderInfoResponse
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
func (e *Exchange) GetSwapOpenOrders(ctx context.Context, contractCode currency.Pair, pageIndex, pageSize uint64) (SwapOpenOrdersResponse, error) {
	var resp SwapOpenOrdersResponse
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
func (e *Exchange) GetSwapOrderHistory(ctx context.Context, contractCode currency.Pair, tradeType, reqType string, status []order.Status, lookbackDays, fromID, limit uint64) (SwapOrderHistoryResponse, error) {
	if lookbackDays > 2 {
		return SwapOrderHistoryResponse{}, errInvalidLookbackDays
	}
	var startTime, endTime time.Time
	if lookbackDays != 0 {
		endTime = time.Now().UTC()
		startTime = endTime.AddDate(0, 0, -int(lookbackDays))
	}
	return e.GetSwapOrderHistoryByTimeRange(ctx, contractCode, tradeType, reqType, status, startTime, endTime, fromID, limit)
}

// GetSwapOrderHistoryByTimeRange gets swap order history for an explicit interval.
func (e *Exchange) GetSwapOrderHistoryByTimeRange(ctx context.Context, contractCode currency.Pair, tradeType, reqType string, status []order.Status, startTime, endTime time.Time, fromID, limit uint64) (SwapOrderHistoryResponse, error) {
	var resp SwapOrderHistoryResponse
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
	reqStatus, err := formatFuturesOrderStatuses(status)
	if err != nil {
		return resp, err
	}
	req["status"] = reqStatus
	if startTime.IsZero() != endTime.IsZero() {
		return resp, errInvalidCreateDate
	}
	if !startTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return resp, err
		}
		if endTime.Sub(startTime) > 48*time.Hour {
			return resp, errHistoryTimeRangeExceeded
		}
		req["start_time"] = startTime.UTC().UnixMilli()
		req["end_time"] = endTime.UTC().UnixMilli()
	}
	req["direct"] = v3HistoryDirectionNext
	if fromID != 0 {
		req["from_id"] = fromID
	}
	if limit != 0 {
		req[orderPriceTypeLimit] = limit
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v3/swap_hisorders", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// GetSwapTradeHistory gets swap trade history
func (e *Exchange) GetSwapTradeHistory(ctx context.Context, contractCode currency.Pair, tradeType string, lookbackDays, fromID, limit uint64) (SwapTradeHistoryResponse, error) {
	var resp SwapTradeHistoryResponse
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract"] = codeValue
	tType, ok := validTradeType[tradeType]
	if !ok {
		return resp, errInvalidTradeType
	}
	req["trade_type"] = tType
	if err := addV3HistoryTimeRange(req, lookbackDays); err != nil {
		return resp, err
	}
	req["direct"] = v3HistoryDirectionNext
	if fromID != 0 {
		req["from_id"] = fromID
	}
	if limit != 0 {
		req[orderPriceTypeLimit] = limit
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v3/swap_matchresults", nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// PlaceSwapTriggerOrder places a trigger order for a swap
func (e *Exchange) PlaceSwapTriggerOrder(ctx context.Context, contractCode currency.Pair, triggerType, direction, offset, orderPriceType string, triggerPrice, orderPrice, volume, leverageRate float64) (SwapTradeHistoryResponse, error) {
	var resp SwapTradeHistoryResponse
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
func (e *Exchange) CancelSwapTriggerOrder(ctx context.Context, contractCode currency.Pair, orderID string) (SwapCancelTriggerOrdersResponse, error) {
	var resp SwapCancelTriggerOrdersResponse
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
func (e *Exchange) CancelAllSwapTriggerOrders(ctx context.Context, contractCode currency.Pair) (SwapCancelTriggerOrdersResponse, error) {
	var resp SwapCancelTriggerOrdersResponse
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
func (e *Exchange) GetSwapTriggerOrderHistory(ctx context.Context, contractCode currency.Pair, status, tradeType string, createDate, pageIndex, pageSize uint64) (SwapTriggerOrderHistoryResponse, error) {
	var resp SwapTriggerOrderHistoryResponse
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
	req["create_date"] = strconv.FormatUint(createDate, 10)
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
func (e *Exchange) GetSwapMarkets(ctx context.Context, contract currency.Pair) ([]SwapMarket, error) {
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
		Data []SwapMarket `json:"data"`
	}
	var result response
	return result.Data, e.SendHTTPRequest(ctx, exchange.RestFutures, "/swap-api/v1/swap_contract_info"+"?"+vals.Encode(), &result)
}

// GetSwapFundingRate gets funding rate data for one currency
func (e *Exchange) GetSwapFundingRate(ctx context.Context, contract currency.Pair) (SwapFundingRate, error) {
	vals := url.Values{}
	codeValue, err := e.FormatSymbol(contract, asset.CoinMarginedFutures)
	if err != nil {
		return SwapFundingRate{}, err
	}
	vals.Set("contract_code", codeValue)
	type response struct {
		Response
		Data SwapFundingRate `json:"data"`
	}
	var result response
	return result.Data, e.SendHTTPRequest(ctx, exchange.RestFutures, "/swap-api/v1/swap_funding_rate"+"?"+vals.Encode(), &result)
}

// GetSwapFundingRates gets funding rates data
func (e *Exchange) GetSwapFundingRates(ctx context.Context) (SwapFundingRatesResponse, error) {
	var result SwapFundingRatesResponse
	return result, e.SendHTTPRequest(ctx, exchange.RestFutures, "/swap-api/v1/swap_batch_funding_rate", &result)
}
