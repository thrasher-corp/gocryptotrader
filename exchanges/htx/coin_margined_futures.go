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
func (e *Exchange) QuerySwapIndexPriceInfo(ctx context.Context, code currency.Pair) (*SwapIndexPriceResponse, error) {
	path := "/swap-api/v1/swap_index"
	if !code.IsEmpty() {
		codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		params := url.Values{}
		params.Set("contract_code", codeValue)
		path = common.EncodeURLValues(path, params)
	}
	resp := new(SwapIndexPriceResponse)
	err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// GetSwapPriceLimits gets price caps for perpetual futures
func (e *Exchange) GetSwapPriceLimits(ctx context.Context, code currency.Pair) (*SwapPriceLimitsResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	path := common.EncodeURLValues("/swap-api/v1/swap_price_limit", params)
	resp := new(SwapPriceLimitsResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// SwapOpenInterestInformation gets open interest data for perpetual futures
func (e *Exchange) SwapOpenInterestInformation(ctx context.Context, code currency.Pair) (*SwapOpenInterestResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	if !code.IsEmpty() {
		params.Set("contract_code", codeValue)
	}
	path := common.EncodeURLValues("/swap-api/v1/swap_open_interest", params)
	resp := new(SwapOpenInterestResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// GetSwapMarketDepth gets market depth for perpetual futures
func (e *Exchange) GetSwapMarketDepth(ctx context.Context, code currency.Pair, dataType string) (*SwapMarketDepthResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("type", dataType)
	path := common.EncodeURLValues("/swap-ex/market/depth", params)
	resp := new(SwapMarketDepthResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// GetSwapKlineData gets kline data for perpetual futures
func (e *Exchange) GetSwapKlineData(ctx context.Context, code currency.Pair, period string, size uint64, startTime, endTime time.Time) (*SwapKlineResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return nil, common.ErrInvalidPeriod
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	if size > 0 {
		params.Set("size", strconv.FormatUint(size, 10))
	}
	if !startTime.IsZero() || !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("from", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("to", strconv.FormatInt(endTime.Unix(), 10))
	}
	path := common.EncodeURLValues("/swap-ex/market/history/kline", params)
	resp := new(SwapKlineResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// GetSwapMarketOverview gets market data overview for perpetual futures
func (e *Exchange) GetSwapMarketOverview(ctx context.Context, code currency.Pair) (*SwapMarketOverviewResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	path := common.EncodeURLValues("/swap-ex/market/detail/merged", params)
	resp := new(SwapMarketOverviewResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// GetLastTrade gets the last trade for a given perpetual contract
func (e *Exchange) GetLastTrade(ctx context.Context, code currency.Pair) (*SwapLastTradeResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	path := common.EncodeURLValues("/swap-ex/market/trade", params)
	resp := new(SwapLastTradeResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// GetBatchTrades gets batch trades for a specified contract (fetching size cannot be bigger than 2000)
func (e *Exchange) GetBatchTrades(ctx context.Context, code currency.Pair, size uint64) (*SwapTradesResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("size", strconv.FormatUint(size, 10))
	path := common.EncodeURLValues("/swap-ex/market/history/trade", params)
	resp := new(SwapTradesResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// GetTieredAjustmentFactorInfo gets tiered adjustment factor data
func (e *Exchange) GetTieredAjustmentFactorInfo(ctx context.Context, code currency.Pair) (*SwapTieredAdjustmentFactorsResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	path := common.EncodeURLValues("/swap-api/v1/swap_adjustfactor", params)
	resp := new(SwapTieredAdjustmentFactorsResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// GetOpenInterestInfo gets open interest data
func (e *Exchange) GetOpenInterestInfo(ctx context.Context, code currency.Pair, period, amountType string, size uint64) (*SwapHistoricalOpenInterestResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return nil, common.ErrInvalidPeriod
	}
	if size == 0 || size > 1200 {
		return nil, errInvalidSize
	}
	aType, ok := validAmountType[amountType]
	if !ok {
		return nil, errInvalidAmountType
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	params.Set("size", strconv.FormatUint(size, 10))
	params.Set("amount_type", strconv.FormatUint(aType, 10))
	path := common.EncodeURLValues("/swap-api/v1/swap_his_open_interest", params)
	resp := new(SwapHistoricalOpenInterestResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// GetSystemStatusInfo gets system status data
func (e *Exchange) GetSystemStatusInfo(ctx context.Context, code currency.Pair) (*SwapSystemStatusResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	path := common.EncodeURLValues("/swap-api/v1/swap_api_state", params)
	resp := new(SwapSystemStatusResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// GetTraderSentimentIndexAccount gets top trader sentiment function-account
func (e *Exchange) GetTraderSentimentIndexAccount(ctx context.Context, code currency.Pair, period string) (*SwapAccountSentimentResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return nil, common.ErrInvalidPeriod
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	path := common.EncodeURLValues("/swap-api/v1/swap_elite_account_ratio", params)
	resp := new(SwapAccountSentimentResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// GetTraderSentimentIndexPosition gets top trader sentiment function-position
func (e *Exchange) GetTraderSentimentIndexPosition(ctx context.Context, code currency.Pair, period string) (*SwapPositionSentimentResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}

	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return nil, common.ErrInvalidPeriod
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	path := common.EncodeURLValues("/swap-api/v1/swap_elite_position_ratio", params)
	resp := new(SwapPositionSentimentResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// GetLiquidationOrders gets liquidation orders for a given perp
func (e *Exchange) GetLiquidationOrders(ctx context.Context, contract currency.Pair, tradeType string, startTime, endTime time.Time, direction string, fromID uint64) (*SwapLiquidationOrdersResponse, error) {
	formattedContract, err := e.FormatSymbol(contract, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	tType, ok := validTradeTypes[tradeType]
	if !ok {
		return nil, errInvalidTradeType
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
	resp := new(SwapLiquidationOrdersResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// GetHistoricalFundingRatesForPair gets historical funding rates for perpetual futures
func (e *Exchange) GetHistoricalFundingRatesForPair(ctx context.Context, code currency.Pair, pageSize, pageIndex uint64) (*SwapHistoricalFundingRatesResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
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
	resp := new(SwapHistoricalFundingRatesResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// GetPremiumIndexKlineData gets kline data for premium index
func (e *Exchange) GetPremiumIndexKlineData(ctx context.Context, code currency.Pair, period string, size uint64) (*SwapPremiumIndexKlineResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return nil, common.ErrInvalidPeriod
	}
	if size == 0 || size > 1200 {
		return nil, errInvalidSize
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("size", strconv.FormatUint(size, 10))
	params.Set("period", period)
	path := common.EncodeURLValues("/index/market/history/swap_premium_index_kline", params)
	resp := new(SwapPremiumIndexKlineResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// GetEstimatedFundingRates gets estimated funding rates for perpetual futures
func (e *Exchange) GetEstimatedFundingRates(ctx context.Context, code currency.Pair, period string, size uint64) (*SwapEstimatedFundingRateResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return nil, common.ErrInvalidPeriod
	}
	if size == 0 || size > 1200 {
		return nil, errInvalidSize
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	params.Set("size", strconv.FormatUint(size, 10))
	path := common.EncodeURLValues("/index/market/history/swap_estimated_rate_kline", params)
	resp := new(SwapEstimatedFundingRateResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// GetBasisData gets basis data for perpetual futures
func (e *Exchange) GetBasisData(ctx context.Context, code currency.Pair, period, basisPriceType string, size uint64) (*SwapBasisResponse, error) {
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return nil, common.ErrInvalidPeriod
	}
	if size == 0 || size > 1200 {
		return nil, errInvalidSize
	}
	if !common.StringSliceCompareInsensitive(validBasisPriceTypes, basisPriceType) {
		return nil, errInvalidBasisPriceType
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	params.Set("basis_price_type", basisPriceType)
	params.Set("size", strconv.FormatUint(size, 10))
	path := common.EncodeURLValues("/index/market/history/swap_basis", params)
	resp := new(SwapBasisResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// GetSwapAccountInfo gets swap account info
func (e *Exchange) GetSwapAccountInfo(ctx context.Context, code currency.Pair) (*SwapAccountResponse, error) {
	req := make(map[string]any)
	if !code.IsEmpty() {
		codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		req["contract_code"] = codeValue
	}
	resp := new(SwapAccountResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_account_info", nil, req, resp)
	return resp, err
}

// GetSwapPositionsInfo gets swap positions' info
func (e *Exchange) GetSwapPositionsInfo(ctx context.Context, code currency.Pair) (*SwapPositionsResponse, error) {
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	req["contract_code"] = codeValue
	resp := new(SwapPositionsResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_position_info", nil, req, resp)
	return resp, err
}

// GetSwapAssetsAndPositions gets swap positions and asset info
func (e *Exchange) GetSwapAssetsAndPositions(ctx context.Context, code currency.Pair) (*SwapAssetsAndPositionsResponse, error) {
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	req["contract_code"] = codeValue
	resp := new(SwapAssetsAndPositionsResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_account_position_info", nil, req, resp)
	return resp, err
}

// GetSwapAllSubAccAssets gets asset info for all subaccounts
func (e *Exchange) GetSwapAllSubAccAssets(ctx context.Context, code currency.Pair) (*SwapSubaccountAssetsResponse, error) {
	req := make(map[string]any)
	if !code.IsEmpty() {
		codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		req["contract_code"] = codeValue
	}
	resp := new(SwapSubaccountAssetsResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_sub_account_list", nil, req, resp)
	return resp, err
}

// SwapSingleSubAccAssets gets a subaccount's assets info
func (e *Exchange) SwapSingleSubAccAssets(ctx context.Context, code currency.Pair, subUID uint64) (*SwapSingleSubaccountAssetsResponse, error) {
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	req["contract_code"] = codeValue
	req["sub_uid"] = subUID
	resp := new(SwapSingleSubaccountAssetsResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_sub_account_info", nil, req, resp)
	return resp, err
}

// GetSubAccPositionInfo gets a subaccount's positions info
func (e *Exchange) GetSubAccPositionInfo(ctx context.Context, code currency.Pair, subUID uint64) (*SwapSubaccountPositionsResponse, error) {
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	req["contract_code"] = codeValue
	req["sub_uid"] = subUID
	resp := new(SwapSubaccountPositionsResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_sub_position_info", nil, req, resp)
	return resp, err
}

// GetAccountFinancialRecords gets the account's financial records
func (e *Exchange) GetAccountFinancialRecords(ctx context.Context, code currency.Pair, orderType string, lookbackDays, fromID, limit uint64) (*SwapFinancialRecordsResponse, error) {
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	req["contract"] = codeValue
	if orderType != "" {
		req["type"] = orderType
	}
	if lookbackDays != 0 {
		if err := addV3HistoryTimeRange(req, lookbackDays); err != nil {
			return nil, err
		}
	}
	if fromID != 0 {
		req["from_id"] = fromID
	}
	if limit != 0 {
		req["limit"] = limit
	}
	req["direct"] = v3HistoryDirectionNext
	resp := new(SwapFinancialRecordsResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v3/swap_financial_record", nil, req, resp)
	return resp, err
}

// GetSwapSettlementRecords gets the swap account's settlement records
func (e *Exchange) GetSwapSettlementRecords(ctx context.Context, code currency.Pair, startTime, endTime time.Time, pageIndex, pageSize uint64) (*SwapFinancialRecordsResponse, error) {
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	req["contract_code"] = codeValue
	if !startTime.IsZero() || !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
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
	resp := new(SwapFinancialRecordsResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_user_settlement_records", nil, req, resp)
	return resp, err
}

// GetAvailableLeverage gets user's available leverage data
func (e *Exchange) GetAvailableLeverage(ctx context.Context, code currency.Pair) (*SwapAvailableLeverageResponse, error) {
	req := make(map[string]any)
	if !code.IsEmpty() {
		codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		req["contract_code"] = codeValue
	}
	resp := new(SwapAvailableLeverageResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_available_level_rate", nil, req, resp)
	return resp, err
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
func (e *Exchange) GetSwapOrderLimitInfo(ctx context.Context, code currency.Pair, orderType string) (*SwapOrderLimitsResponse, error) {
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	req["contract_code"] = codeValue
	if !common.StringSliceCompareInsensitive(validOrderTypes, orderType) {
		return nil, errInvalidOrderType
	}
	req["order_price_type"] = orderType
	resp := new(SwapOrderLimitsResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_order_limit", nil, req, resp)
	return resp, err
}

// GetSwapTradingFeeInfo gets trading fee info for swaps
func (e *Exchange) GetSwapTradingFeeInfo(ctx context.Context, code currency.Pair) (*SwapTradingFeesResponse, error) {
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	req["contract_code"] = codeValue
	resp := new(SwapTradingFeesResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_fee", nil, req, resp)
	return resp, err
}

// GetSwapTransferLimitInfo gets transfer limit info for swaps
func (e *Exchange) GetSwapTransferLimitInfo(ctx context.Context, code currency.Pair) (*SwapTransferLimitsResponse, error) {
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	req["contract_code"] = codeValue
	resp := new(SwapTransferLimitsResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_transfer_limit", nil, req, resp)
	return resp, err
}

// GetSwapPositionLimitInfo gets transfer limit info for swaps
func (e *Exchange) GetSwapPositionLimitInfo(ctx context.Context, code currency.Pair) (*SwapPositionLimitsResponse, error) {
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	req["contract_code"] = codeValue
	resp := new(SwapPositionLimitsResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_position_limit", nil, req, resp)
	return resp, err
}

// AccountTransferData gets asset transfer data between master and subaccounts
func (e *Exchange) AccountTransferData(ctx context.Context, code currency.Pair, subUID, transferType string, amount float64) (*SwapInternalTransferResponse, error) {
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	req["contract_code"] = codeValue
	req["subUid"] = subUID
	req["amount"] = amount
	if !common.StringSliceCompareInsensitive(validTransferType, transferType) {
		return nil, errInvalidTransferType
	}
	req["type"] = transferType
	resp := new(SwapInternalTransferResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_master_sub_transfer", nil, req, resp)
	return resp, err
}

// AccountTransferRecords gets asset transfer records between master and subaccounts
func (e *Exchange) AccountTransferRecords(ctx context.Context, code currency.Pair, transferType string, createDate, pageIndex, pageSize uint64) (*SwapInternalTransferResponse, error) {
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	req["contract_code"] = codeValue
	if !common.StringSliceCompareInsensitive(validTransferType, transferType) {
		return nil, errInvalidTransferType
	}
	req["type"] = transferType
	if createDate > 90 {
		return nil, errInvalidCreateDate
	}
	req["create_date"] = strconv.FormatUint(createDate, 10)
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize > 0 && pageSize <= 50 {
		req["page_size"] = pageSize
	}
	resp := new(SwapInternalTransferResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_master_sub_transfer_record", nil, req, resp)
	return resp, err
}

// PlaceSwapOrders places orders for swaps
func (e *Exchange) PlaceSwapOrders(ctx context.Context, code currency.Pair, clientOrderID, direction, offset, orderPriceType string, price, volume, leverage float64) (*SwapOrderResponse, error) {
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	req["contract_code"] = codeValue
	if clientOrderID != "" {
		req["client_order_id"] = clientOrderID
	}
	req["direction"] = direction
	req["offset"] = offset
	if !common.StringSliceCompareInsensitive(validOrderTypes, orderPriceType) {
		return nil, errInvalidOrderType
	}
	req["order_price_type"] = orderPriceType
	req["price"] = price
	req["volume"] = volume
	req["lever_rate"] = leverage
	resp := new(SwapOrderResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_order", nil, req, resp)
	return resp, err
}

// PlaceSwapBatchOrders places a batch of orders for swaps
func (e *Exchange) PlaceSwapBatchOrders(ctx context.Context, data *SwapBatchOrderRequest) (*SwapBatchOrderResponse, error) {
	if data == nil {
		return nil, common.ErrNilPointer
	}
	req := make(map[string]any)
	if len(data.Orders) == 0 {
		return nil, errEmptyOrderBatch
	}
	if len(data.Orders) > 10 {
		return nil, errBatchOrderLimitExceeded
	}
	for x := range data.Orders {
		if data.Orders[x].ContractCode == "" {
			continue
		}
		unformattedPair, err := currency.NewPairFromString(data.Orders[x].ContractCode)
		if err != nil {
			return nil, err
		}
		codeValue, err := e.FormatSymbol(unformattedPair, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		data.Orders[x].ContractCode = codeValue
	}
	req["orders_data"] = data.Orders
	resp := new(SwapBatchOrderResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_batchorder", nil, req, resp)
	return resp, err
}

// CancelSwapOrder sends a request to cancel an order
func (e *Exchange) CancelSwapOrder(ctx context.Context, orderID, clientOrderID string, contractCode currency.Pair) (*SwapCancelOrdersResponse, error) {
	if orderID == "" && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if orderID != "" && clientOrderID != "" {
		return nil, errMixedOrderIdentifiers
	}
	identifiers := orderID
	if identifiers == "" {
		identifiers = clientOrderID
	}
	if count := strings.Count(identifiers, ",") + 1; count > 10 {
		return nil, fmt.Errorf("%w: maximum 10, received %d", errCancelOrderLimitExceeded, count)
	}
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	if orderID != "" {
		req["order_id"] = orderID
	}
	if clientOrderID != "" {
		req["client_order_id"] = clientOrderID
	}
	req["contract_code"] = codeValue
	resp := new(SwapCancelOrdersResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_cancel", nil, req, resp)
	return resp, err
}

// CancelAllSwapOrders sends a request to cancel an order
func (e *Exchange) CancelAllSwapOrders(ctx context.Context, contractCode currency.Pair) (*SwapCancelOrdersResponse, error) {
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	req["contract_code"] = codeValue
	resp := new(SwapCancelOrdersResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_cancelall", nil, req, resp)
	return resp, err
}

// PlaceLightningCloseOrder places a lightning close order
func (e *Exchange) PlaceLightningCloseOrder(ctx context.Context, contractCode currency.Pair, direction, orderPriceType string, volume float64, clientOrderID uint64) (*SwapLightningCloseOrderResponse, error) {
	if clientOrderID > math.MaxInt64 {
		return nil, errInvalidClientOrderID
	}
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	req["contract_code"] = codeValue
	req["volume"] = volume
	req["direction"] = direction
	if clientOrderID != 0 {
		req["client_order_id"] = clientOrderID
	}
	if orderPriceType != "" {
		if !common.StringSliceCompareInsensitive(validLightningOrderPriceType, orderPriceType) {
			return nil, errInvalidOrderPriceType
		}
		req["order_price_type"] = orderPriceType
	}
	resp := new(SwapLightningCloseOrderResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_lightning_close_position", nil, req, resp)
	return resp, err
}

// GetSwapOrderDetails gets order info
func (e *Exchange) GetSwapOrderDetails(ctx context.Context, contractCode currency.Pair, orderID, createdAt, orderType string, pageIndex, pageSize uint64) (*SwapOrderResponse, error) {
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	req["contract_code"] = codeValue
	req["order_id"] = orderID
	req["created_at"] = createdAt
	oType, ok := validOrderType[orderType]
	if !ok {
		return nil, errInvalidOrderType
	}
	req["order_type"] = oType
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize > 0 && pageSize <= 50 {
		req["page_size"] = pageSize
	}
	resp := new(SwapOrderResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_order_detail", nil, req, resp)
	return resp, err
}

// GetSwapOrderInfo gets info on a swap order
func (e *Exchange) GetSwapOrderInfo(ctx context.Context, contractCode currency.Pair, orderID, clientOrderID string) (*SwapOrderInfoResponse, error) {
	req := make(map[string]any)
	if !contractCode.IsEmpty() {
		codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
		if err != nil {
			return nil, err
		}
		req["contract_code"] = codeValue
	}
	if orderID != "" {
		req["order_id"] = orderID
	}
	if clientOrderID != "" {
		req["client_order_id"] = clientOrderID
	}
	resp := new(SwapOrderInfoResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_order_info", nil, req, resp)
	return resp, err
}

// GetSwapOpenOrders gets open orders for swap
func (e *Exchange) GetSwapOpenOrders(ctx context.Context, contractCode currency.Pair, pageIndex, pageSize uint64) (*SwapOpenOrdersResponse, error) {
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	req["contract_code"] = codeValue
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize > 0 && pageSize <= 50 {
		req["page_size"] = pageSize
	}
	resp := new(SwapOpenOrdersResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_openorders", nil, req, resp)
	return resp, err
}

// GetSwapOrderHistory gets swap order history using a lookback of at most two days.
func (e *Exchange) GetSwapOrderHistory(ctx context.Context, contractCode currency.Pair, tradeType, reqType string, status []order.Status, lookbackDays, fromID, limit uint64) (*SwapOrderHistoryResponse, error) {
	if lookbackDays > 2 {
		return nil, errInvalidLookbackDays
	}
	var startTime, endTime time.Time
	if lookbackDays != 0 {
		endTime = time.Now().UTC()
		startTime = endTime.AddDate(0, 0, -int(lookbackDays))
	}
	return e.GetSwapOrderHistoryByTimeRange(ctx, contractCode, tradeType, reqType, status, startTime, endTime, fromID, limit)
}

// GetSwapOrderHistoryByTimeRange gets swap order history for an explicit interval.
func (e *Exchange) GetSwapOrderHistoryByTimeRange(ctx context.Context, contractCode currency.Pair, tradeType, reqType string, status []order.Status, startTime, endTime time.Time, fromID, limit uint64) (*SwapOrderHistoryResponse, error) {
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	req["contract"] = codeValue
	tType, ok := validFuturesTradeType[tradeType]
	if !ok {
		return nil, errInvalidTradeType
	}
	req["trade_type"] = tType
	rType, ok := validFuturesReqType[reqType]
	if !ok {
		return nil, errInvalidRequestType
	}
	req["type"] = rType
	reqStatus, err := formatFuturesOrderStatuses(status)
	if err != nil {
		return nil, err
	}
	req["status"] = reqStatus
	if startTime.IsZero() != endTime.IsZero() {
		return nil, errInvalidCreateDate
	}
	if !startTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		if endTime.Sub(startTime) > 48*time.Hour {
			return nil, errHistoryTimeRangeExceeded
		}
		req["start_time"] = startTime.UTC().UnixMilli()
		req["end_time"] = endTime.UTC().UnixMilli()
	}
	req["direct"] = v3HistoryDirectionNext
	if fromID != 0 {
		req["from_id"] = fromID
	}
	if limit != 0 {
		req["limit"] = limit
	}
	resp := new(SwapOrderHistoryResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v3/swap_hisorders", nil, req, resp)
	return resp, err
}

// GetSwapTradeHistory gets swap trade history
func (e *Exchange) GetSwapTradeHistory(ctx context.Context, contractCode currency.Pair, tradeType string, lookbackDays, fromID, limit uint64) (*SwapTradeHistoryResponse, error) {
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	req["contract"] = codeValue
	tType, ok := validTradeType[tradeType]
	if !ok {
		return nil, errInvalidTradeType
	}
	req["trade_type"] = tType
	if err := addV3HistoryTimeRange(req, lookbackDays); err != nil {
		return nil, err
	}
	req["direct"] = v3HistoryDirectionNext
	if fromID != 0 {
		req["from_id"] = fromID
	}
	if limit != 0 {
		req["limit"] = limit
	}
	resp := new(SwapTradeHistoryResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v3/swap_matchresults", nil, req, resp)
	return resp, err
}

// PlaceSwapTriggerOrder places a trigger order for a swap
func (e *Exchange) PlaceSwapTriggerOrder(ctx context.Context, contractCode currency.Pair, triggerType, direction, offset, orderPriceType string, triggerPrice, orderPrice, volume, leverageRate float64) (*SwapTradeHistoryResponse, error) {
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	req["contract_code"] = codeValue
	tType, ok := validTriggerType[triggerType]
	if !ok {
		return nil, errInvalidTriggerType
	}
	req["trigger_type"] = tType
	req["direction"] = direction
	req["offset"] = offset
	req["trigger_price"] = triggerPrice
	req["volume"] = volume
	req["lever_rate"] = leverageRate
	req["order_price"] = orderPrice
	if !common.StringSliceCompareInsensitive(validOrderPriceType, orderPriceType) {
		return nil, errInvalidOrderPriceType
	}
	req["order_price_type"] = orderPriceType
	resp := new(SwapTradeHistoryResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_trigger_order", nil, req, resp)
	return resp, err
}

// CancelSwapTriggerOrder cancels swap trigger order
func (e *Exchange) CancelSwapTriggerOrder(ctx context.Context, contractCode currency.Pair, orderID string) (*SwapCancelTriggerOrdersResponse, error) {
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	req["contract_code"] = codeValue
	req["order_id"] = orderID
	resp := new(SwapCancelTriggerOrdersResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_trigger_cancel", nil, req, resp)
	return resp, err
}

// CancelAllSwapTriggerOrders cancels all swap trigger orders
func (e *Exchange) CancelAllSwapTriggerOrders(ctx context.Context, contractCode currency.Pair) (*SwapCancelTriggerOrdersResponse, error) {
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	req["contract_code"] = codeValue
	resp := new(SwapCancelTriggerOrdersResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_trigger_cancelall", nil, req, resp)
	return resp, err
}

// GetSwapTriggerOrderHistory gets history for swap trigger orders
func (e *Exchange) GetSwapTriggerOrderHistory(ctx context.Context, contractCode currency.Pair, status, tradeType string, createDate, pageIndex, pageSize uint64) (*SwapTriggerOrderHistoryResponse, error) {
	req := make(map[string]any)
	codeValue, err := e.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	req["contract_code"] = codeValue
	req["status"] = status
	tType, ok := validTradeType[tradeType]
	if !ok {
		return nil, errInvalidTradeType
	}
	req["trade_type"] = tType
	if createDate > 90 {
		return nil, errInvalidCreateDate
	}
	req["create_date"] = strconv.FormatUint(createDate, 10)
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize > 0 && pageSize <= 50 {
		req["page_size"] = pageSize
	}
	resp := new(SwapTriggerOrderHistoryResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/swap-api/v1/swap_trigger_hisorders", nil, req, resp)
	return resp, err
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
	result := new(SwapMarketsResponse)
	err := e.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues("/swap-api/v1/swap_contract_info", vals), result)
	return result.Markets, err
}

// GetSwapFundingRate gets funding rate data for one currency
func (e *Exchange) GetSwapFundingRate(ctx context.Context, contract currency.Pair) (*SwapFundingRate, error) {
	vals := url.Values{}
	codeValue, err := e.FormatSymbol(contract, asset.CoinMarginedFutures)
	if err != nil {
		return nil, err
	}
	vals.Set("contract_code", codeValue)
	result := new(SwapFundingRateResponse)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, common.EncodeURLValues("/swap-api/v1/swap_funding_rate", vals), result); err != nil {
		return nil, err
	}
	return &result.FundingRate, nil
}

// GetSwapFundingRates gets funding rates data
func (e *Exchange) GetSwapFundingRates(ctx context.Context) (*SwapFundingRatesResponse, error) {
	result := new(SwapFundingRatesResponse)
	err := e.SendHTTPRequest(ctx, exchange.RestFutures, "/swap-api/v1/swap_batch_funding_rate", result)
	return result, err
}
