package huobi

import (
	"context"
	"errors"
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

const (
	// Coin Margined Swap (perpetual futures) endpoints
	huobiSwapMarkets                  = "/swap-api/v1/swap_contract_info"
	huobiSwapFunding                  = "/swap-api/v1/swap_funding_rate"
	huobiSwapBatchFunding             = "/swap-api/v1/swap_batch_funding_rate"
	huobiSwapIndexPriceInfo           = "/swap-api/v1/swap_index"
	huobiSwapPriceLimitation          = "/swap-api/v1/swap_price_limit"
	huobiSwapOpenInterestInfo         = "/swap-api/v1/swap_open_interest"
	huobiSwapMarketDepth              = "/swap-ex/market/depth"
	huobiKLineData                    = "/swap-ex/market/history/kline"
	huobiMarketDataOverview           = "/swap-ex/market/detail/merged"
	huobiLastTradeContract            = "/swap-ex/market/trade"
	huobiRequestBatchOfTradingRecords = "/swap-ex/market/history/trade"
	huobiTieredAdjustmentFactor       = "/swap-api/v1/swap_adjustfactor"
	huobiOpenInterestInfo             = "/swap-api/v1/swap_his_open_interest"
	huobiSwapSystemStatus             = "/swap-api/v1/swap_api_state"
	huobiSwapSentimentAccountData     = "/swap-api/v1/swap_elite_account_ratio"
	huobiSwapSentimentPosition        = "/swap-api/v1/swap_elite_position_ratio"
	huobiSwapLiquidationOrders        = "/swap-api/v3/swap_liquidation_orders"
	huobiSwapHistoricalFundingRate    = "/swap-api/v1/swap_historical_funding_rate"
	huobiPremiumIndexKlineData        = "/index/market/history/swap_premium_index_kline"
	huobiPredictedFundingRateData     = "/index/market/history/swap_estimated_rate_kline"
	huobiBasisData                    = "/index/market/history/swap_basis"
	huobiSwapAccInfo                  = "/swap-api/v1/swap_account_info"
	huobiSwapPosInfo                  = "/swap-api/v1/swap_position_info"
	huobiSwapAssetsAndPos             = "/swap-api/v1/swap_account_position_info" //nolint // false positive gosec
	huobiSwapSubAccList               = "/swap-api/v1/swap_sub_account_list"
	huobiSwapSubAccInfo               = "/swap-api/v1/swap_sub_account_info"
	huobiSwapSubAccPosInfo            = "/swap-api/v1/swap_sub_position_info"
	huobiSwapFinancialRecords         = "/swap-api/v1/swap_financial_record"
	huobiSwapSettlementRecords        = "/swap-api/v1/swap_user_settlement_records"
	huobiSwapAvailableLeverage        = "/swap-api/v1/swap_available_level_rate"
	huobiSwapOrderLimitInfo           = "/swap-api/v1/swap_order_limit"
	huobiSwapTradingFeeInfo           = "/swap-api/v1/swap_fee"
	huobiSwapTransferLimitInfo        = "/swap-api/v1/swap_transfer_limit"
	huobiSwapPositionLimitInfo        = "/swap-api/v1/swap_position_limit"
	huobiSwapInternalTransferData     = "/swap-api/v1/swap_master_sub_transfer"
	huobiSwapInternalTransferRecords  = "/swap-api/v1/swap_master_sub_transfer_record"
	huobiSwapPlaceOrder               = "/swap-api/v1/swap_order"
	huobiSwapPlaceBatchOrder          = "/swap-api/v1/swap_batchorder"
	huobiSwapCancelOrder              = "/swap-api/v1/swap_cancel"
	huobiSwapCancelAllOrders          = "/swap-api/v1/swap_cancelall"
	huobiSwapLightningCloseOrder      = "/swap-api/v1/swap_lightning_close_position"
	huobiSwapOrderInfo                = "/swap-api/v1/swap_order_info"
	huobiSwapOrderDetails             = "/swap-api/v1/swap_order_detail"
	huobiSwapOpenOrders               = "/swap-api/v1/swap_openorders"
	huobiSwapOrderHistory             = "/swap-api/v1/swap_hisorders"
	huobiSwapTradeHistory             = "/swap-api/v1/swap_matchresults"
	huobiSwapTriggerOrder             = "/swap-api/v1/swap_trigger_order"
	huobiSwapCancelTriggerOrder       = "/swap-api/v1/swap_trigger_cancel"
	huobiSwapCancelAllTriggerOrders   = "/swap-api/v1/swap_trigger_cancelall"
	huobiSwapTriggerOrderHistory      = "/swap-api/v1/swap_trigger_hisorders"
)

// QuerySwapIndexPriceInfo gets perpetual swap index's price info
func (h *HUOBI) QuerySwapIndexPriceInfo(ctx context.Context, code currency.Pair) (SwapIndexPriceData, error) {
	var resp SwapIndexPriceData
	path := huobiSwapIndexPriceInfo
	if !code.IsEmpty() {
		codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		params := url.Values{}
		params.Set("contract_code", codeValue)
		path = common.EncodeURLValues(path, params)
	}
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// GetSwapPriceLimits gets price caps for perpetual futures
func (h *HUOBI) GetSwapPriceLimits(ctx context.Context, code currency.Pair) (SwapPriceLimitsData, error) {
	var resp SwapPriceLimitsData
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	path := common.EncodeURLValues(huobiSwapPriceLimitation, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// SwapOpenInterestInformation gets open interest data for perpetual futures
func (h *HUOBI) SwapOpenInterestInformation(ctx context.Context, code currency.Pair) (SwapOpenInterestData, error) {
	var resp SwapOpenInterestData
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	if !code.IsEmpty() {
		params.Set("contract_code", codeValue)
	}
	path := common.EncodeURLValues(huobiSwapOpenInterestInfo, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// GetSwapMarketDepth gets market depth for perpetual futures
func (h *HUOBI) GetSwapMarketDepth(ctx context.Context, code currency.Pair, dataType string) (SwapMarketDepthData, error) {
	var resp SwapMarketDepthData
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("type", dataType)
	path := common.EncodeURLValues(huobiSwapMarketDepth, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// GetSwapKlineData gets kline data for perpetual futures
func (h *HUOBI) GetSwapKlineData(ctx context.Context, code currency.Pair, period string, size int64, startTime, endTime time.Time) (SwapKlineData, error) {
	var resp SwapKlineData
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, errors.New("invalid period value received")
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("from", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("to", strconv.FormatInt(endTime.Unix(), 10))
	}
	path := common.EncodeURLValues(huobiKLineData, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// GetSwapMarketOverview gets market data overview for perpetual futures
func (h *HUOBI) GetSwapMarketOverview(ctx context.Context, code currency.Pair) (MarketOverviewData, error) {
	var resp MarketOverviewData
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	path := common.EncodeURLValues(huobiMarketDataOverview, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// GetLastTrade gets the last trade for a given perpetual contract
func (h *HUOBI) GetLastTrade(ctx context.Context, code currency.Pair) (LastTradeData, error) {
	var resp LastTradeData
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	path := common.EncodeURLValues(huobiLastTradeContract, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// GetBatchTrades gets batch trades for a specified contract (fetching size cannot be bigger than 2000)
func (h *HUOBI) GetBatchTrades(ctx context.Context, code currency.Pair, size int64) (BatchTradesData, error) {
	var resp BatchTradesData
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("size", strconv.FormatInt(size, 10))
	path := common.EncodeURLValues(huobiRequestBatchOfTradingRecords, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// GetTieredAjustmentFactorInfo gets tiered adjustment factor data
func (h *HUOBI) GetTieredAjustmentFactorInfo(ctx context.Context, code currency.Pair) (TieredAdjustmentFactorData, error) {
	var resp TieredAdjustmentFactorData
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	path := common.EncodeURLValues(huobiTieredAdjustmentFactor, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// GetOpenInterestInfo gets open interest data
func (h *HUOBI) GetOpenInterestInfo(ctx context.Context, code currency.Pair, period, amountType string, size int64) (OpenInterestData, error) {
	var resp OpenInterestData
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, errors.New("invalid period value received")
	}
	if size <= 0 || size > 1200 {
		return resp, errors.New("invalid size provided, only values between 1-1200 are supported")
	}
	aType, ok := validAmountType[amountType]
	if !ok {
		return resp, errors.New("invalid trade type")
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	params.Set("size", strconv.FormatInt(size, 10))
	params.Set("amount_type", strconv.FormatInt(aType, 10))
	path := common.EncodeURLValues(huobiOpenInterestInfo, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// GetSystemStatusInfo gets system status data
func (h *HUOBI) GetSystemStatusInfo(ctx context.Context, code currency.Pair) (SystemStatusData, error) {
	var resp SystemStatusData
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	path := common.EncodeURLValues(huobiSwapSystemStatus, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// GetTraderSentimentIndexAccount gets top trader sentiment function-account
func (h *HUOBI) GetTraderSentimentIndexAccount(ctx context.Context, code currency.Pair, period string) (TraderSentimentIndexAccountData, error) {
	var resp TraderSentimentIndexAccountData
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, errors.New("invalid period value received")
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	path := common.EncodeURLValues(huobiSwapSentimentAccountData, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// GetTraderSentimentIndexPosition gets top trader sentiment function-position
func (h *HUOBI) GetTraderSentimentIndexPosition(ctx context.Context, code currency.Pair, period string) (TraderSentimentIndexPositionData, error) {
	var resp TraderSentimentIndexPositionData
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}

	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, errors.New("invalid period value received")
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	path := common.EncodeURLValues(huobiSwapSentimentPosition, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// GetLiquidationOrders gets liquidation orders for a given perp
func (h *HUOBI) GetLiquidationOrders(ctx context.Context, contract currency.Pair, tradeType string, startTime, endTime time.Time, direction string, fromID int64) (LiquidationOrdersData, error) {
	var resp LiquidationOrdersData
	formattedContract, err := h.FormatSymbol(contract, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	tType, ok := validTradeTypes[tradeType]
	if !ok {
		return resp, errors.New("invalid trade type")
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
	path := common.EncodeURLValues(huobiSwapLiquidationOrders, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// GetHistoricalFundingRatesForPair gets historical funding rates for perpetual futures
func (h *HUOBI) GetHistoricalFundingRatesForPair(ctx context.Context, code currency.Pair, pageSize, pageIndex int64) (HistoricalFundingRateData, error) {
	var resp HistoricalFundingRateData
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	if pageIndex != 0 {
		params.Set("page_index", strconv.FormatInt(pageIndex, 10))
	}
	if pageSize != 0 {
		params.Set("page_size", strconv.FormatInt(pageIndex, 10))
	}
	path := common.EncodeURLValues(huobiSwapHistoricalFundingRate, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// GetPremiumIndexKlineData gets kline data for premium index
func (h *HUOBI) GetPremiumIndexKlineData(ctx context.Context, code currency.Pair, period string, size int64) (PremiumIndexKlineData, error) {
	var resp PremiumIndexKlineData
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, errors.New("invalid period value received")
	}
	if size <= 0 || size > 1200 {
		return resp, errors.New("invalid size provided, only values between 1-1200 are supported")
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("size", strconv.FormatInt(size, 10))
	params.Set("period", period)
	path := common.EncodeURLValues(huobiPremiumIndexKlineData, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// GetEstimatedFundingRates gets estimated funding rates for perpetual futures
func (h *HUOBI) GetEstimatedFundingRates(ctx context.Context, code currency.Pair, period string, size int64) (EstimatedFundingRateData, error) {
	var resp EstimatedFundingRateData
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, errors.New("invalid period value received")
	}
	if size <= 0 || size > 1200 {
		return resp, errors.New("invalid size provided, only values between 1-1200 are supported")
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	params.Set("size", strconv.FormatInt(size, 10))
	path := common.EncodeURLValues(huobiPredictedFundingRateData, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// GetBasisData gets basis data for perpetual futures
func (h *HUOBI) GetBasisData(ctx context.Context, code currency.Pair, period, basisPriceType string, size int64) (BasisData, error) {
	var resp BasisData
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, errors.New("invalid period value received")
	}
	if size <= 0 || size > 1200 {
		return resp, errors.New("invalid size provided, only values between 1-1200 are supported")
	}
	if !common.StringSliceCompareInsensitive(validBasisPriceTypes, basisPriceType) {
		return resp, errors.New("invalid period value received")
	}
	params := url.Values{}
	params.Set("contract_code", codeValue)
	params.Set("period", period)
	params.Set("size", strconv.FormatInt(size, 10))
	path := common.EncodeURLValues(huobiBasisData, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// GetSwapAccountInfo gets swap account info
func (h *HUOBI) GetSwapAccountInfo(ctx context.Context, code currency.Pair) (SwapAccountInformation, error) {
	var resp SwapAccountInformation
	req := make(map[string]any)
	if !code.IsEmpty() {
		codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapAccInfo, nil, req, &resp)
}

// GetSwapPositionsInfo gets swap positions' info
func (h *HUOBI) GetSwapPositionsInfo(ctx context.Context, code currency.Pair) (SwapPositionInfo, error) {
	var resp SwapPositionInfo
	req := make(map[string]any)
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapPosInfo, nil, req, &resp)
}

// GetSwapAssetsAndPositions gets swap positions and asset info
func (h *HUOBI) GetSwapAssetsAndPositions(ctx context.Context, code currency.Pair) (SwapAssetsAndPositionsData, error) {
	var resp SwapAssetsAndPositionsData
	req := make(map[string]any)
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapAssetsAndPos, nil, req, &resp)
}

// GetSwapAllSubAccAssets gets asset info for all subaccounts
func (h *HUOBI) GetSwapAllSubAccAssets(ctx context.Context, code currency.Pair) (SubAccountsAssetData, error) {
	var resp SubAccountsAssetData
	req := make(map[string]any)
	if !code.IsEmpty() {
		codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapSubAccList, nil, req, &resp)
}

// SwapSingleSubAccAssets gets a subaccount's assets info
func (h *HUOBI) SwapSingleSubAccAssets(ctx context.Context, code currency.Pair, subUID int64) (SingleSubAccountAssetsInfo, error) {
	var resp SingleSubAccountAssetsInfo
	req := make(map[string]any)
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	req["sub_uid"] = subUID
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapSubAccInfo, nil, req, &resp)
}

// GetSubAccPositionInfo gets a subaccount's positions info
func (h *HUOBI) GetSubAccPositionInfo(ctx context.Context, code currency.Pair, subUID int64) (SingleSubAccountPositionsInfo, error) {
	var resp SingleSubAccountPositionsInfo
	req := make(map[string]any)
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	req["sub_uid"] = subUID
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapSubAccPosInfo, nil, req, &resp)
}

// GetAccountFinancialRecords gets the account's financial records
func (h *HUOBI) GetAccountFinancialRecords(ctx context.Context, code currency.Pair, orderType string, createDate, pageIndex, pageSize int64) (FinancialRecordData, error) {
	var resp FinancialRecordData
	req := make(map[string]any)
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	if orderType != "" {
		req["type"] = orderType
	}
	if createDate != 0 {
		req["create_date"] = createDate
	}
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapFinancialRecords, nil, req, &resp)
}

// GetSwapSettlementRecords gets the swap account's settlement records
func (h *HUOBI) GetSwapSettlementRecords(ctx context.Context, code currency.Pair, startTime, endTime time.Time, pageIndex, pageSize int64) (FinancialRecordData, error) {
	var resp FinancialRecordData
	req := make(map[string]any)
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
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
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapSettlementRecords, nil, req, &resp)
}

// GetAvailableLeverage gets user's available leverage data
func (h *HUOBI) GetAvailableLeverage(ctx context.Context, code currency.Pair) (AvailableLeverageData, error) {
	var resp AvailableLeverageData
	req := make(map[string]any)
	if !code.IsEmpty() {
		codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapAvailableLeverage, nil, req, &resp)
}

// GetSwapOrderLimitInfo gets order limit info for swaps
func (h *HUOBI) GetSwapOrderLimitInfo(ctx context.Context, code currency.Pair, orderType string) (SwapOrderLimitInfo, error) {
	var resp SwapOrderLimitInfo
	req := make(map[string]any)
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	if !common.StringSliceCompareInsensitive(validOrderTypes, orderType) {
		return resp, errors.New("invalid ordertype provided")
	}
	req["order_price_type"] = orderType
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapOrderLimitInfo, nil, req, &resp)
}

// GetSwapTradingFeeInfo gets trading fee info for swaps
func (h *HUOBI) GetSwapTradingFeeInfo(ctx context.Context, code currency.Pair) (SwapTradingFeeData, error) {
	var resp SwapTradingFeeData
	req := make(map[string]any)
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapTradingFeeInfo, nil, req, &resp)
}

// GetSwapTransferLimitInfo gets transfer limit info for swaps
func (h *HUOBI) GetSwapTransferLimitInfo(ctx context.Context, code currency.Pair) (TransferLimitData, error) {
	var resp TransferLimitData
	req := make(map[string]any)
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapTransferLimitInfo, nil, req, &resp)
}

// GetSwapPositionLimitInfo gets transfer limit info for swaps
func (h *HUOBI) GetSwapPositionLimitInfo(ctx context.Context, code currency.Pair) (PositionLimitData, error) {
	var resp PositionLimitData
	req := make(map[string]any)
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapPositionLimitInfo, nil, req, &resp)
}

// AccountTransferData gets asset transfer data between master and subaccounts
func (h *HUOBI) AccountTransferData(ctx context.Context, code currency.Pair, subUID, transferType string, amount float64) (InternalAccountTransferData, error) {
	var resp InternalAccountTransferData
	req := make(map[string]any)
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	req["subUid"] = subUID
	req["amount"] = amount
	if !common.StringSliceCompareInsensitive(validTransferType, transferType) {
		return resp, errors.New("invalid transferType received")
	}
	req["type"] = transferType
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapInternalTransferData, nil, req, &resp)
}

// AccountTransferRecords gets asset transfer records between master and subaccounts
func (h *HUOBI) AccountTransferRecords(ctx context.Context, code currency.Pair, transferType string, createDate, pageIndex, pageSize int64) (InternalAccountTransferData, error) {
	var resp InternalAccountTransferData
	req := make(map[string]any)
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	if !common.StringSliceCompareInsensitive(validTransferType, transferType) {
		return resp, errors.New("invalid transferType received")
	}
	req["type"] = transferType
	if createDate > 90 {
		return resp, errors.New("invalid create date value: only supports up to 90 days")
	}
	req["create_date"] = strconv.FormatInt(createDate, 10)
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize > 0 && pageSize <= 50 {
		req["page_size"] = pageSize
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapInternalTransferRecords, nil, req, &resp)
}

// PlaceSwapOrders places orders for swaps
func (h *HUOBI) PlaceSwapOrders(ctx context.Context, code currency.Pair, clientOrderID, direction, offset, orderPriceType string, price, volume, leverage float64) (SwapOrderData, error) {
	var resp SwapOrderData
	req := make(map[string]any)
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
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
		return resp, errors.New("invalid ordertype provided")
	}
	req["order_price_type"] = orderPriceType
	req["price"] = price
	req["volume"] = volume
	req["lever_rate"] = leverage
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapPlaceOrder, nil, req, &resp)
}

// PlaceSwapBatchOrders places a batch of orders for swaps
func (h *HUOBI) PlaceSwapBatchOrders(ctx context.Context, data BatchOrderRequestType) (BatchOrderData, error) {
	var resp BatchOrderData
	req := make(map[string]any)
	if len(data.Data) > 10 || len(data.Data) == 0 {
		return resp, errors.New("invalid data provided: maximum of 10 batch orders supported")
	}
	for x := range data.Data {
		if data.Data[x].ContractCode == "" {
			continue
		}
		unformattedPair, err := currency.NewPairFromString(data.Data[x].ContractCode)
		if err != nil {
			return resp, err
		}
		codeValue, err := h.FormatSymbol(unformattedPair, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		data.Data[x].ContractCode = codeValue
	}
	req["orders_data"] = data.Data
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapPlaceBatchOrder, nil, req, &resp)
}

// CancelSwapOrder sends a request to cancel an order
func (h *HUOBI) CancelSwapOrder(ctx context.Context, orderID, clientOrderID string, contractCode currency.Pair) (CancelOrdersData, error) {
	var resp CancelOrdersData
	req := make(map[string]any)
	if orderID != "" {
		req["order_id"] = orderID
	}
	if clientOrderID != "" {
		req["client_order_id"] = clientOrderID
	}
	req["contract_code"] = contractCode
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapCancelOrder, nil, req, &resp)
}

// CancelAllSwapOrders sends a request to cancel an order
func (h *HUOBI) CancelAllSwapOrders(ctx context.Context, contractCode currency.Pair) (CancelOrdersData, error) {
	var resp CancelOrdersData
	req := make(map[string]any)
	req["contract_code"] = contractCode
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapCancelAllOrders, nil, req, &resp)
}

// PlaceLightningCloseOrder places a lightning close order
func (h *HUOBI) PlaceLightningCloseOrder(ctx context.Context, contractCode currency.Pair, direction, orderPriceType string, volume float64, clientOrderID int64) (LightningCloseOrderData, error) {
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
			return resp, errors.New("invalid orderPriceType")
		}
		req["order_price_type"] = orderPriceType
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapLightningCloseOrder, nil, req, &resp)
}

// GetSwapOrderDetails gets order info
func (h *HUOBI) GetSwapOrderDetails(ctx context.Context, contractCode currency.Pair, orderID, createdAt, orderType string, pageIndex, pageSize int64) (SwapOrderData, error) {
	var resp SwapOrderData
	req := make(map[string]any)
	req["contract_code"] = contractCode
	req["order_id"] = orderID
	req["created_at"] = createdAt
	oType, ok := validOrderType[orderType]
	if !ok {
		return resp, errors.New("invalid ordertype")
	}
	req["order_type"] = oType
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize > 0 && pageSize <= 50 {
		req["page_size"] = pageSize
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapOrderDetails, nil, req, &resp)
}

// GetSwapOrderInfo gets info on a swap order
func (h *HUOBI) GetSwapOrderInfo(ctx context.Context, contractCode currency.Pair, orderID, clientOrderID string) (SwapOrderInfo, error) {
	var resp SwapOrderInfo
	req := make(map[string]any)
	if !contractCode.IsEmpty() {
		codeValue, err := h.FormatSymbol(contractCode, asset.CoinMarginedFutures)
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
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapOrderInfo, nil, req, &resp)
}

// GetSwapOpenOrders gets open orders for swap
func (h *HUOBI) GetSwapOpenOrders(ctx context.Context, contractCode currency.Pair, pageIndex, pageSize int64) (SwapOpenOrdersData, error) {
	var resp SwapOpenOrdersData
	req := make(map[string]any)
	codeValue, err := h.FormatSymbol(contractCode, asset.CoinMarginedFutures)
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
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapOpenOrders, nil, req, &resp)
}

// GetSwapOrderHistory gets swap order history
func (h *HUOBI) GetSwapOrderHistory(ctx context.Context, contractCode currency.Pair, tradeType, reqType string, status []order.Status, createDate, pageIndex, pageSize int64) (SwapOrderHistory, error) {
	var resp SwapOrderHistory
	req := make(map[string]any)
	codeValue, err := h.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	tType, ok := validFuturesTradeType[tradeType]
	if !ok {
		return resp, errors.New("invalid tradeType")
	}
	req["trade_type"] = tType
	rType, ok := validFuturesReqType[reqType]
	if !ok {
		return resp, errors.New("invalid reqType")
	}
	req["type"] = rType
	reqStatus := "0"
	if len(status) > 0 {
		firstTime := true
		for x := range status {
			sType, ok := validOrderStatus[status[x]]
			if !ok {
				return resp, errors.New("invalid status")
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
	if createDate < 0 || createDate > 90 {
		return resp, errors.New("invalid createDate")
	}
	req["create_date"] = createDate
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapOrderHistory, nil, req, &resp)
}

// GetSwapTradeHistory gets swap trade history
func (h *HUOBI) GetSwapTradeHistory(ctx context.Context, contractCode currency.Pair, tradeType string, createDate, pageIndex, pageSize int64) (AccountTradeHistoryData, error) {
	var resp AccountTradeHistoryData
	req := make(map[string]any)
	codeValue, err := h.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	if createDate > 90 {
		return resp, errors.New("invalid create date value: only supports up to 90 days")
	}
	tType, ok := validTradeType[tradeType]
	if !ok {
		return resp, errors.New("invalid trade type")
	}
	req["trade_type"] = tType
	req["create_date"] = strconv.FormatInt(createDate, 10)
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize > 0 && pageSize <= 50 {
		req["page_size"] = pageSize
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapTradeHistory, nil, req, &resp)
}

// PlaceSwapTriggerOrder places a trigger order for a swap
func (h *HUOBI) PlaceSwapTriggerOrder(ctx context.Context, contractCode currency.Pair, triggerType, direction, offset, orderPriceType string, triggerPrice, orderPrice, volume, leverageRate float64) (AccountTradeHistoryData, error) {
	var resp AccountTradeHistoryData
	req := make(map[string]any)
	codeValue, err := h.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	tType, ok := validTriggerType[triggerType]
	if !ok {
		return resp, errors.New("invalid trigger type")
	}
	req["trigger_type"] = tType
	req["direction"] = direction
	req["offset"] = offset
	req["trigger_price"] = triggerPrice
	req["volume"] = volume
	req["lever_rate"] = leverageRate
	req["order_price"] = orderPrice
	if !common.StringSliceCompareInsensitive(validOrderPriceType, orderPriceType) {
		return resp, errors.New("invalid order price type")
	}
	req["order_price_type"] = orderPriceType
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapTriggerOrder, nil, req, &resp)
}

// CancelSwapTriggerOrder cancels swap trigger order
func (h *HUOBI) CancelSwapTriggerOrder(ctx context.Context, contractCode currency.Pair, orderID string) (CancelTriggerOrdersData, error) {
	var resp CancelTriggerOrdersData
	req := make(map[string]any)
	req["contract_code"] = contractCode
	req["order_id"] = orderID
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapCancelTriggerOrder, nil, req, &resp)
}

// CancelAllSwapTriggerOrders cancels all swap trigger orders
func (h *HUOBI) CancelAllSwapTriggerOrders(ctx context.Context, contractCode currency.Pair) (CancelTriggerOrdersData, error) {
	var resp CancelTriggerOrdersData
	req := make(map[string]any)
	codeValue, err := h.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapCancelAllTriggerOrders, nil, req, &resp)
}

// GetSwapTriggerOrderHistory gets history for swap trigger orders
func (h *HUOBI) GetSwapTriggerOrderHistory(ctx context.Context, contractCode currency.Pair, status, tradeType string, createDate, pageIndex, pageSize int64) (TriggerOrderHistory, error) {
	var resp TriggerOrderHistory
	req := make(map[string]any)
	codeValue, err := h.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	req["status"] = status
	tType, ok := validTradeType[tradeType]
	if !ok {
		return resp, errors.New("invalid trade type")
	}
	req["trade_type"] = tType
	if createDate > 90 {
		return resp, errors.New("invalid create date value: only supports up to 90 days")
	}
	req["create_date"] = strconv.FormatInt(createDate, 10)
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize > 0 && pageSize <= 50 {
		req["page_size"] = pageSize
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, huobiSwapTriggerOrderHistory, nil, req, &resp)
}

// GetSwapMarkets gets data of swap markets
func (h *HUOBI) GetSwapMarkets(ctx context.Context, contract currency.Pair) ([]SwapMarketsData, error) {
	vals := url.Values{}
	if !contract.IsEmpty() {
		codeValue, err := h.FormatSymbol(contract, asset.CoinMarginedFutures)
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
	err := h.SendHTTPRequest(ctx, exchange.RestFutures, huobiSwapMarkets+"?"+vals.Encode(), &result)
	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Data, err
}

// GetSwapFundingRate gets funding rate data for one currency
func (h *HUOBI) GetSwapFundingRate(ctx context.Context, contract currency.Pair) (FundingRatesData, error) {
	vals := url.Values{}
	codeValue, err := h.FormatSymbol(contract, asset.CoinMarginedFutures)
	if err != nil {
		return FundingRatesData{}, err
	}
	vals.Set("contract_code", codeValue)
	type response struct {
		Response
		Data FundingRatesData `json:"data"`
	}
	var result response
	err = h.SendHTTPRequest(ctx, exchange.RestFutures, huobiSwapFunding+"?"+vals.Encode(), &result)
	if result.ErrorMessage != "" {
		return FundingRatesData{}, errors.New(result.ErrorMessage)
	}
	return result.Data, err
}

// GetSwapFundingRates gets funding rates data
func (h *HUOBI) GetSwapFundingRates(ctx context.Context) (SwapFundingRatesResponse, error) {
	var result SwapFundingRatesResponse
	err := h.SendHTTPRequest(ctx, exchange.RestFutures, huobiSwapBatchFunding, &result)
	return result, err
}
