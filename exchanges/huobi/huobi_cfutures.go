package huobi

import (
	"errors"
	"fmt"
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
	huobiSwapMarkets                     = "swap-api/v1/swap_contract_info?"
	huobiSwapFunding                     = "swap-api/v1/swap_funding_rate?"
	huobiSwapIndexPriceInfo              = "swap-api/v1/swap_index?"
	huobiSwapPriceLimitation             = "swap-api/v1/swap_price_limit?"
	huobiSwapOpenInterestInfo            = "swap-api/v1/swap_open_interest?"
	huobiSwapMarketDepth                 = "swap-ex/market/depth?"
	huobiKLineData                       = "swap-ex/market/history/kline?"
	huobiMarketDataOverview              = "swap-ex/market/detail/merged?"
	huobiLastTradeContract               = "swap-ex/market/trade?"
	huobiRequestBatchOfTradingRecords    = "swap-ex/market/history/trade?"
	huobiInsuranceBalanceAndClawbackRate = "swap-api/v1/swap_risk_info?"
	huobiInsuranceBalanceHistory         = "swap-api/v1/swap_insurance_fund?"
	huobiTieredAdjustmentFactor          = "swap-api/v1/swap_adjustfactor?"
	huobiOpenInterestInfo                = "swap-api/v1/swap_his_open_interest?"
	huobiSwapSystemStatus                = "swap-api/v1/swap_api_state?"
	huobiSwapSentimentAccountData        = "swap-api/v1/swap_elite_account_ratio?"
	huobiSwapSentimentPosition           = "swap-api/v1/swap_elite_position_ratio?"
	huobiSwapLiquidationOrders           = "swap-api/v1/swap_liquidation_orders?"
	huobiSwapHistoricalFundingRate       = "swap-api/v1/swap_historical_funding_rate?"
	huobiPremiumIndexKlineData           = "index/market/history/swap_premium_index_kline?"
	huobiPredictedFundingRateData        = "index/market/history/swap_estimated_rate_kline?"
	huobiBasisData                       = "index/market/history/swap_basis?"
	huobiSwapAccInfo                     = "swap-api/v1/swap_account_info"
	huobiSwapPosInfo                     = "swap-api/v1/swap_position_info"
	huobiSwapAssetsAndPos                = "swap-api/v1/swap_account_position_info" // nolint // false positive gosec
	huobiSwapSubAccList                  = "swap-api/v1/swap_sub_account_list"
	huobiSwapSubAccInfo                  = "swap-api/v1/swap_sub_account_info"
	huobiSwapSubAccPosInfo               = "swap-api/v1/swap_sub_position_info"
	huobiSwapFinancialRecords            = "swap-api/v1/swap_financial_record"
	huobiSwapSettlementRecords           = "swap-api/v1/swap_user_settlement_records"
	huobiSwapAvailableLeverage           = "swap-api/v1/swap_available_level_rate"
	huobiSwapOrderLimitInfo              = "swap-api/v1/swap_order_limit"
	huobiSwapTradingFeeInfo              = "swap-api/v1/swap_fee"
	huobiSwapTransferLimitInfo           = "swap-api/v1/swap_transfer_limit"
	huobiSwapPositionLimitInfo           = "swap-api/v1/swap_position_limit"
	huobiSwapInternalTransferData        = "swap-api/v1/swap_master_sub_transfer"
	huobiSwapInternalTransferRecords     = "swap-api/v1/swap_master_sub_transfer_record"
	huobiSwapPlaceOrder                  = "/swap-api/v1/swap_order"
	huobiSwapPlaceBatchOrder             = "/swap-api/v1/swap_batchorder"
	huobiSwapCancelOrder                 = "/swap-api/v1/swap_cancel"
	huobiSwapCancelAllOrders             = "/swap-api/v1/swap_cancelall"
	huobiSwapLightningCloseOrder         = "/swap-api/v1/swap_lightning_close_position"
	huobiSwapOrderInfo                   = "/swap-api/v1/swap_order_info"
	huobiSwapOrderDetails                = "/swap-api/v1/swap_order_detail"
	huobiSwapOpenOrders                  = "/swap-api/v1/swap_openorders"
	huobiSwapOrderHistory                = "/swap-api/v1/swap_hisorders"
	huobiSwapTradeHistory                = "/swap-api/v1/swap_matchresults"
	huobiSwapTriggerOrder                = "swap-api/v1/swap_trigger_order"
	huobiSwapCancelTriggerOrder          = "/swap-api/v1/swap_trigger_cancel"
	huobiSwapCancelAllTriggerOrders      = "/swap-api/v1/swap_trigger_cancelall"
	huobiSwapTriggerOrderHistory         = "/swap-api/v1/swap_trigger_hisorders"
)

// QuerySwapIndexPriceInfo gets perpetual swap index's price info
func (h *HUOBI) QuerySwapIndexPriceInfo(code currency.Pair) (SwapIndexPriceData, error) {
	var resp SwapIndexPriceData
	path := huobiSwapIndexPriceInfo
	if code != (currency.Pair{}) {
		codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		params := url.Values{}
		params.Set("contract_code", codeValue)
		path = huobiSwapIndexPriceInfo + params.Encode()
	}
	return resp, h.SendHTTPRequest(exchange.RestFutures, path, &resp)
}

// GetSwapPriceLimits gets price caps for perpetual futures
func (h *HUOBI) GetSwapPriceLimits(code currency.Pair) (SwapPriceLimitsData, error) {
	var resp SwapPriceLimitsData
	params := url.Values{}
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("contract_code", codeValue)
	return resp, h.SendHTTPRequest(exchange.RestFutures, huobiSwapPriceLimitation+params.Encode(),
		&resp)
}

// SwapOpenInterestInformation gets open interest data for perpetual futures
func (h *HUOBI) SwapOpenInterestInformation(code currency.Pair) (SwapOpenInterestData, error) {
	var resp SwapOpenInterestData
	params := url.Values{}
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("contract_code", codeValue)
	return resp, h.SendHTTPRequest(exchange.RestFutures, huobiSwapOpenInterestInfo+params.Encode(), &resp)
}

// GetSwapMarketDepth gets market depth for perpetual futures
func (h *HUOBI) GetSwapMarketDepth(code currency.Pair, dataType string) (SwapMarketDepthData, error) {
	var resp SwapMarketDepthData
	params := url.Values{}
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("contract_code", codeValue)
	params.Set("type", dataType)
	return resp, h.SendHTTPRequest(exchange.RestFutures, huobiSwapMarketDepth+params.Encode(), &resp)
}

// GetSwapKlineData gets kline data for perpetual futures
func (h *HUOBI) GetSwapKlineData(code currency.Pair, period string, size int64, startTime, endTime time.Time) (SwapKlineData, error) {
	var resp SwapKlineData
	params := url.Values{}
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("contract_code", codeValue)
	if !common.StringDataCompare(validPeriods, period) {
		return resp, fmt.Errorf("invalid period value received")
	}
	params.Set("period", period)
	if size == 1 || size > 2000 {
		return resp, fmt.Errorf("invalid size")
	}
	params.Set("size", strconv.FormatInt(size, 10))
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	return resp, h.SendHTTPRequest(exchange.RestFutures, huobiKLineData+params.Encode(), &resp)
}

// GetSwapMarketOverview gets market data overview for perpetual futures
func (h *HUOBI) GetSwapMarketOverview(code currency.Pair) (MarketOverviewData, error) {
	var resp MarketOverviewData
	params := url.Values{}
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("contract_code", codeValue)
	return resp, h.SendHTTPRequest(exchange.RestFutures, huobiMarketDataOverview+params.Encode(), &resp)
}

// GetLastTrade gets the last trade for a given perpetual contract
func (h *HUOBI) GetLastTrade(code currency.Pair) (LastTradeData, error) {
	var resp LastTradeData
	params := url.Values{}
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("contract_code", codeValue)
	return resp, h.SendHTTPRequest(exchange.RestFutures, huobiLastTradeContract+params.Encode(), &resp)
}

// GetBatchTrades gets batch trades for a specified contract (fetching size cannot be bigger than 2000)
func (h *HUOBI) GetBatchTrades(code currency.Pair, size int64) (BatchTradesData, error) {
	var resp BatchTradesData
	params := url.Values{}
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("contract_code", codeValue)
	if size <= 0 || size > 1200 {
		return resp, fmt.Errorf("invalid size provided values from 1-1200 supported")
	}
	params.Set("size", strconv.FormatInt(size, 10))
	return resp, h.SendHTTPRequest(exchange.RestFutures, huobiRequestBatchOfTradingRecords+params.Encode(), &resp)
}

// GetInsuranceData gets insurance fund data and clawback rates
func (h *HUOBI) GetInsuranceData(code currency.Pair) (InsuranceAndClawbackData, error) {
	var resp InsuranceAndClawbackData
	params := url.Values{}
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("contract_code", codeValue)
	return resp, h.SendHTTPRequest(exchange.RestFutures, huobiInsuranceBalanceAndClawbackRate+params.Encode(), &resp)
}

// GetHistoricalInsuranceData gets historical insurance fund data and clawback rates
func (h *HUOBI) GetHistoricalInsuranceData(code currency.Pair, pageIndex, pageSize int64) (HistoricalInsuranceFundBalance, error) {
	var resp HistoricalInsuranceFundBalance
	params := url.Values{}
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("contract_code", codeValue)
	if pageIndex != 0 {
		params.Set("page_index", strconv.FormatInt(pageIndex, 10))
	}
	if pageSize != 0 {
		params.Set("page_size", strconv.FormatInt(pageIndex, 10))
	}
	return resp, h.SendHTTPRequest(exchange.RestFutures, huobiInsuranceBalanceHistory+params.Encode(), &resp)
}

// GetTieredAjustmentFactorInfo gets tiered adjustment factor data
func (h *HUOBI) GetTieredAjustmentFactorInfo(code currency.Pair) (TieredAdjustmentFactorData, error) {
	var resp TieredAdjustmentFactorData
	params := url.Values{}
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("contract_code", codeValue)
	return resp, h.SendHTTPRequest(exchange.RestFutures, huobiTieredAdjustmentFactor+params.Encode(), &resp)
}

// GetOpenInterestInfo gets open interest data
func (h *HUOBI) GetOpenInterestInfo(code currency.Pair, period, amountType string, size int64) (OpenInterestData, error) {
	var resp OpenInterestData
	params := url.Values{}
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("contract_code", codeValue)
	if !common.StringDataCompare(validPeriods, period) {
		return resp, fmt.Errorf("invalid period value received")
	}
	params.Set("period", period)
	if size <= 0 || size > 1200 {
		return resp, fmt.Errorf("invalid size provided values from 1-1200 supported")
	}
	params.Set("size", strconv.FormatInt(size, 10))
	aType, ok := validAmountType[amountType]
	if !ok {
		return resp, fmt.Errorf("invalid trade type")
	}
	params.Set("amount_type", strconv.FormatInt(aType, 10))
	return resp, h.SendHTTPRequest(exchange.RestFutures, huobiOpenInterestInfo+params.Encode(), &resp)
}

// GetSystemStatusInfo gets system status data
func (h *HUOBI) GetSystemStatusInfo(code currency.Pair, period, amountType string, size int64) (SystemStatusData, error) {
	var resp SystemStatusData
	params := url.Values{}
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("contract_code", codeValue)
	if !common.StringDataCompare(validPeriods, period) {
		return resp, fmt.Errorf("invalid period value received")
	}
	params.Set("period", period)
	if size > 0 && size <= 1200 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	aType, ok := validAmountType[amountType]
	if !ok {
		return resp, fmt.Errorf("invalid trade type")
	}
	params.Set("amount_type", strconv.FormatInt(aType, 10))
	return resp, h.SendHTTPRequest(exchange.RestFutures, huobiSwapSystemStatus+params.Encode(), &resp)
}

// GetTraderSentimentIndexAccount gets top trader sentiment function-account
func (h *HUOBI) GetTraderSentimentIndexAccount(code currency.Pair, period string) (TraderSentimentIndexAccountData, error) {
	var resp TraderSentimentIndexAccountData
	params := url.Values{}
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("contract_code", codeValue)
	if !common.StringDataCompare(validPeriods, period) {
		return resp, fmt.Errorf("invalid period value received")
	}
	params.Set("period", period)
	return resp, h.SendHTTPRequest(exchange.RestFutures, huobiSwapSentimentAccountData+params.Encode(), &resp)
}

// GetTraderSentimentIndexPosition gets top trader sentiment function-position
func (h *HUOBI) GetTraderSentimentIndexPosition(code currency.Pair, period string) (TraderSentimentIndexPositionData, error) {
	var resp TraderSentimentIndexPositionData
	params := url.Values{}
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("contract_code", codeValue)
	if !common.StringDataCompare(validPeriods, period) {
		return resp, fmt.Errorf("invalid period value received")
	}
	params.Set("period", period)
	return resp, h.SendHTTPRequest(exchange.RestFutures, huobiSwapSentimentPosition+params.Encode(), &resp)
}

// GetLiquidationOrders gets liquidation orders for a given perp
func (h *HUOBI) GetLiquidationOrders(code currency.Pair, tradeType string, pageIndex, pageSize, createDate int64) (LiquidationOrdersData, error) {
	var resp LiquidationOrdersData
	params := url.Values{}
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("contract_code", codeValue)
	if createDate != 7 && createDate != 90 {
		return resp, fmt.Errorf("invalid createDate. 7 and 90 are the only supported values")
	}
	params.Set("create_date", strconv.FormatInt(createDate, 10))
	tType, ok := validTradeTypes[tradeType]
	if !ok {
		return resp, fmt.Errorf("invalid trade type")
	}
	params.Set("trade_type", strconv.FormatInt(tType, 10))
	if pageIndex != 0 {
		params.Set("page_index", strconv.FormatInt(pageIndex, 10))
	}
	if pageSize != 0 {
		params.Set("page_size", strconv.FormatInt(pageIndex, 10))
	}
	return resp, h.SendHTTPRequest(exchange.RestFutures, huobiSwapLiquidationOrders+params.Encode(), &resp)
}

// GetHistoricalFundingRates gets historical funding rates for perpetual futures
func (h *HUOBI) GetHistoricalFundingRates(code currency.Pair, pageSize, pageIndex int64) (HistoricalFundingRateData, error) {
	var resp HistoricalFundingRateData
	params := url.Values{}
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("contract_code", codeValue)
	if pageIndex != 0 {
		params.Set("page_index", strconv.FormatInt(pageIndex, 10))
	}
	if pageSize != 0 {
		params.Set("page_size", strconv.FormatInt(pageIndex, 10))
	}
	return resp, h.SendHTTPRequest(exchange.RestFutures, huobiSwapHistoricalFundingRate+params.Encode(), &resp)
}

// GetPremiumIndexKlineData gets kline data for premium index
func (h *HUOBI) GetPremiumIndexKlineData(code currency.Pair, period string, size int64) (PremiumIndexKlineData, error) {
	var resp PremiumIndexKlineData
	params := url.Values{}
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("contract_code", codeValue)
	if !common.StringDataCompare(validPeriods, period) {
		return resp, fmt.Errorf("invalid period value received")
	}
	params.Set("period", period)
	if size <= 0 || size > 1200 {
		return resp, fmt.Errorf("invalid size provided values from 1-1200 supported")
	}
	params.Set("size", strconv.FormatInt(size, 10))
	return resp, h.SendHTTPRequest(exchange.RestFutures, huobiPremiumIndexKlineData+params.Encode(), &resp)
}

// GetEstimatedFundingRates gets estimated funding rates for perpetual futures
func (h *HUOBI) GetEstimatedFundingRates(code currency.Pair, period string, size int64) (EstimatedFundingRateData, error) {
	var resp EstimatedFundingRateData
	params := url.Values{}
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("contract_code", codeValue)
	if !common.StringDataCompare(validPeriods, period) {
		return resp, fmt.Errorf("invalid period value received")
	}
	params.Set("period", period)
	if size <= 0 || size > 1200 {
		return resp, fmt.Errorf("invalid size provided values from 1-1200 supported")
	}
	params.Set("size", strconv.FormatInt(size, 10))
	return resp, h.SendHTTPRequest(exchange.RestFutures, huobiPredictedFundingRateData+params.Encode(), &resp)
}

// GetBasisData gets basis data for perpetual futures
func (h *HUOBI) GetBasisData(code currency.Pair, period, basisPriceType string, size int64) (BasisData, error) {
	var resp BasisData
	params := url.Values{}
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("contract_code", codeValue)
	if !common.StringDataCompare(validPeriods, period) {
		return resp, fmt.Errorf("invalid period value received")
	}
	params.Set("period", period)
	if size <= 0 || size > 1200 {
		return resp, fmt.Errorf("invalid size provided values from 1-1200 supported")
	}
	params.Set("size", strconv.FormatInt(size, 10))
	if !common.StringDataCompare(validBasisPriceTypes, basisPriceType) {
		return resp, fmt.Errorf("invalid period value received")
	}
	return resp, h.SendHTTPRequest(exchange.RestFutures, huobiBasisData+params.Encode(), &resp)
}

// GetSwapAccountInfo gets swap account info
func (h *HUOBI) GetSwapAccountInfo(code currency.Pair) (SwapAccountInformation, error) {
	var resp SwapAccountInformation
	req := make(map[string]interface{})
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapAccInfo, nil, req, &resp)
}

// GetSwapPositionsInfo gets swap positions' info
func (h *HUOBI) GetSwapPositionsInfo(code currency.Pair) (SwapPositionInfo, error) {
	var resp SwapPositionInfo
	req := make(map[string]interface{})
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapPosInfo, nil, req, &resp)
}

// GetSwapAssetsAndPositions gets swap positions and asset info
func (h *HUOBI) GetSwapAssetsAndPositions(code currency.Pair) (SwapAssetsAndPositionsData, error) {
	var resp SwapAssetsAndPositionsData
	req := make(map[string]interface{})
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapAssetsAndPos, nil, req, &resp)
}

// GetSwapAllSubAccAssets gets asset info for all subaccounts
func (h *HUOBI) GetSwapAllSubAccAssets(code currency.Pair) (SubAccountsAssetData, error) {
	var resp SubAccountsAssetData
	req := make(map[string]interface{})
	if code != (currency.Pair{}) {
		codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapSubAccList, nil, req, &resp)
}

// SwapSingleSubAccAssets gets a subaccount's assets info
func (h *HUOBI) SwapSingleSubAccAssets(code currency.Pair, subUID int64) (SingleSubAccountAssetsInfo, error) {
	var resp SingleSubAccountAssetsInfo
	req := make(map[string]interface{})
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	req["sub_uid"] = subUID
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapSubAccInfo, nil, req, &resp)
}

// GetSubAccPositionInfo gets a subaccount's positions info
func (h *HUOBI) GetSubAccPositionInfo(code currency.Pair, subUID int64) (SingleSubAccountPositionsInfo, error) {
	var resp SingleSubAccountPositionsInfo
	req := make(map[string]interface{})
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	req["sub_uid"] = subUID
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapSubAccPosInfo, nil, req, &resp)
}

// GetAccountFinancialRecords gets the account's financial records
func (h *HUOBI) GetAccountFinancialRecords(code currency.Pair, orderType string, createDate, pageIndex, pageSize int64) (FinancialRecordData, error) {
	var resp FinancialRecordData
	req := make(map[string]interface{})
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
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapFinancialRecords, nil, req, &resp)
}

// GetSwapSettlementRecords gets the swap account's settlement records
func (h *HUOBI) GetSwapSettlementRecords(code currency.Pair, startTime, endTime time.Time, pageIndex, pageSize int64) (FinancialRecordData, error) {
	var resp FinancialRecordData
	req := make(map[string]interface{})
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		req["start_time"] = strconv.FormatInt(startTime.Unix(), 10)
		req["end_time"] = strconv.FormatInt(endTime.Unix(), 10)
	}
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapSettlementRecords, nil, req, &resp)
}

// GetAvailableLeverage gets user's available leverage data
func (h *HUOBI) GetAvailableLeverage(code currency.Pair) (AvailableLeverageData, error) {
	var resp AvailableLeverageData
	req := make(map[string]interface{})
	if code != (currency.Pair{}) {
		codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapAvailableLeverage, nil, req, &resp)
}

// GetSwapOrderLimitInfo gets order limit info for swaps
func (h *HUOBI) GetSwapOrderLimitInfo(code currency.Pair, orderType string) (SwapOrderLimitInfo, error) {
	var resp SwapOrderLimitInfo
	req := make(map[string]interface{})
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	if !common.StringDataCompare(validOrderTypes, orderType) {
		return resp, fmt.Errorf("inavlid ordertype provided")
	}
	req["order_price_type"] = orderType
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapOrderLimitInfo, nil, req, &resp)
}

// GetSwapTradingFeeInfo gets trading fee info for swaps
func (h *HUOBI) GetSwapTradingFeeInfo(code currency.Pair) (SwapTradingFeeData, error) {
	var resp SwapTradingFeeData
	req := make(map[string]interface{})
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapTradingFeeInfo, nil, req, &resp)
}

// GetSwapTransferLimitInfo gets transfer limit info for swaps
func (h *HUOBI) GetSwapTransferLimitInfo(code currency.Pair) (TransferLimitData, error) {
	var resp TransferLimitData
	req := make(map[string]interface{})
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapTransferLimitInfo, nil, req, &resp)
}

// GetSwapPositionLimitInfo gets transfer limit info for swaps
func (h *HUOBI) GetSwapPositionLimitInfo(code currency.Pair) (PositionLimitData, error) {
	var resp PositionLimitData
	req := make(map[string]interface{})
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapPositionLimitInfo, nil, req, &resp)
}

// AccountTransferData gets asset transfer data between master and subaccounts
func (h *HUOBI) AccountTransferData(code currency.Pair, subUID, transferType string, amount float64) (InternalAccountTransferData, error) {
	var resp InternalAccountTransferData
	req := make(map[string]interface{})
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	req["subUid"] = subUID
	req["amount"] = amount
	if !common.StringDataCompare(validTransferType, transferType) {
		return resp, fmt.Errorf("inavlid transferType received")
	}
	req["type"] = transferType
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapInternalTransferData, nil, req, &resp)
}

// AccountTransferRecords gets asset transfer records between master and subaccounts
func (h *HUOBI) AccountTransferRecords(code currency.Pair, transferType string, createDate, pageIndex, pageSize int64) (InternalAccountTransferData, error) {
	var resp InternalAccountTransferData
	req := make(map[string]interface{})
	codeValue, err := h.FormatSymbol(code, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	if !common.StringDataCompare(validTransferType, transferType) {
		return resp, fmt.Errorf("inavlid transferType received")
	}
	req["type"] = transferType
	if createDate > 90 {
		return resp, fmt.Errorf("invalid create date value: only supports up to 90 days")
	}
	req["create_date"] = strconv.FormatInt(createDate, 10)
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize > 0 && pageSize <= 50 {
		req["page_size"] = pageSize
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapInternalTransferRecords, nil, req, &resp)
}

// PlaceSwapOrders places orders for swaps
func (h *HUOBI) PlaceSwapOrders(code currency.Pair, clientOrderID, direction, offset, orderPriceType string, price, volume, leverage float64) (SwapOrderData, error) {
	var resp SwapOrderData
	req := make(map[string]interface{})
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
	if !common.StringDataCompare(validOrderTypes, orderPriceType) {
		return resp, fmt.Errorf("inavlid ordertype provided")
	}
	req["order_price_type"] = orderPriceType
	req["price"] = price
	req["volume"] = volume
	req["lever_rate"] = leverage
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapPlaceOrder, nil, req, &resp)
}

// PlaceSwapBatchOrders places a batch of orders for swaps
func (h *HUOBI) PlaceSwapBatchOrders(data BatchOrderRequestType) (BatchOrderData, error) {
	var resp BatchOrderData
	req := make(map[string]interface{})
	if len(data.Data) > 10 || len(data.Data) == 0 {
		return resp, fmt.Errorf("invalid data provided: maximum of 10 batch orders supported")
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
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapPlaceBatchOrder, nil, req, &resp)
}

// CancelSwapOrder sends a request to cancel an order
func (h *HUOBI) CancelSwapOrder(orderID, clientOrderID string, contractCode currency.Pair) (CancelOrdersData, error) {
	var resp CancelOrdersData
	req := make(map[string]interface{})
	if orderID != "" {
		req["order_id"] = orderID
	}
	if clientOrderID != "" {
		req["client_order_id"] = clientOrderID
	}
	req["contract_code"] = contractCode
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapCancelOrder, nil, req, &resp)
}

// CancelAllSwapOrders sends a request to cancel an order
func (h *HUOBI) CancelAllSwapOrders(contractCode currency.Pair) (CancelOrdersData, error) {
	var resp CancelOrdersData
	req := make(map[string]interface{})
	req["contract_code"] = contractCode
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapCancelAllOrders, nil, req, &resp)
}

// PlaceLightningCloseOrder places a lightning close order
func (h *HUOBI) PlaceLightningCloseOrder(contractCode currency.Pair, direction, orderPriceType string, volume float64, clientOrderID int64) (LightningCloseOrderData, error) {
	var resp LightningCloseOrderData
	req := make(map[string]interface{})
	req["contract_code"] = contractCode
	req["volume"] = volume
	req["direction"] = direction
	if clientOrderID != 0 {
		req["client_order_id"] = clientOrderID
	}
	if orderPriceType != "" {
		if !common.StringDataCompare(validLightningOrderPriceType, orderPriceType) {
			return resp, fmt.Errorf("invalid orderPriceType")
		}
		req["order_price_type"] = orderPriceType
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapLightningCloseOrder, nil, req, &resp)
}

// GetSwapOrderDetails gets order info
func (h *HUOBI) GetSwapOrderDetails(contractCode currency.Pair, orderID, createdAt, orderType string, pageIndex, pageSize int64) (SwapOrderData, error) {
	var resp SwapOrderData
	req := make(map[string]interface{})
	req["contract_code"] = contractCode
	req["order_id"] = orderID
	req["created_at"] = createdAt
	oType, ok := validOrderType[orderType]
	if !ok {
		return resp, fmt.Errorf("invalid ordertype")
	}
	req["order_type"] = oType
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize > 0 && pageSize <= 50 {
		req["page_size"] = pageSize
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapOrderDetails, nil, req, &resp)
}

// GetSwapOrderInfo gets info on a swap order
func (h *HUOBI) GetSwapOrderInfo(contractCode currency.Pair, orderID, clientOrderID string) (SwapOrderInfo, error) {
	var resp SwapOrderInfo
	req := make(map[string]interface{})
	if contractCode != (currency.Pair{}) {
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
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapOrderInfo, nil, req, &resp)
}

// GetSwapOpenOrders gets open orders for swap
func (h *HUOBI) GetSwapOpenOrders(contractCode currency.Pair, pageIndex, pageSize int64) (SwapOpenOrdersData, error) {
	var resp SwapOpenOrdersData
	req := make(map[string]interface{})
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
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapOpenOrders, nil, req, &resp)
}

// GetSwapOrderHistory gets swap order history
func (h *HUOBI) GetSwapOrderHistory(contractCode currency.Pair, tradeType, reqType string, status []order.Status, createDate, pageIndex, pageSize int64) (SwapOrderHistory, error) {
	var resp SwapOrderHistory
	req := make(map[string]interface{})
	codeValue, err := h.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	tType, ok := validFuturesTradeType[tradeType]
	if !ok {
		return resp, fmt.Errorf("invalid tradeType")
	}
	req["trade_type"] = tType
	rType, ok := validFuturesReqType[reqType]
	if !ok {
		return resp, fmt.Errorf("invalid reqType")
	}
	req["type"] = rType
	reqStatus := "0"
	if len(status) > 0 {
		firstTime := true
		for x := range status {
			sType, ok := validOrderStatus[status[x]]
			if !ok {
				return resp, fmt.Errorf("invalid status")
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
		return resp, fmt.Errorf("invalid createDate")
	}
	req["create_date"] = createDate
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapOrderHistory, nil, req, &resp)
}

// GetSwapTradeHistory gets swap trade history
func (h *HUOBI) GetSwapTradeHistory(contractCode currency.Pair, tradeType string, createDate, pageIndex, pageSize int64) (AccountTradeHistoryData, error) {
	var resp AccountTradeHistoryData
	req := make(map[string]interface{})
	codeValue, err := h.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	if createDate > 90 {
		return resp, fmt.Errorf("invalid create date value: only supports up to 90 days")
	}
	tType, ok := validTradeType[tradeType]
	if !ok {
		return resp, fmt.Errorf("invalid trade type")
	}
	req["trade_type"] = tType
	req["create_date"] = strconv.FormatInt(createDate, 10)
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize > 0 && pageSize <= 50 {
		req["page_size"] = pageSize
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapTradeHistory, nil, req, &resp)
}

// PlaceSwapTriggerOrder places a trigger order for a swap
func (h *HUOBI) PlaceSwapTriggerOrder(contractCode currency.Pair, triggerType, direction, offset, orderPriceType string, triggerPrice, orderPrice, volume, leverageRate float64) (AccountTradeHistoryData, error) {
	var resp AccountTradeHistoryData
	req := make(map[string]interface{})
	codeValue, err := h.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	tType, ok := validTriggerType[triggerType]
	if !ok {
		return resp, fmt.Errorf("invalid trigger type")
	}
	req["trigger_type"] = tType
	req["direction"] = direction
	req["offset"] = offset
	req["trigger_price"] = triggerPrice
	req["volume"] = volume
	req["lever_rate"] = leverageRate
	req["order_price"] = orderPrice
	if !common.StringDataCompare(validOrderPriceType, orderPriceType) {
		return resp, fmt.Errorf("invalid order price type")
	}
	req["order_price_type"] = orderPriceType
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapTriggerOrder, nil, req, &resp)
}

// CancelSwapTriggerOrder cancels swap trigger order
func (h *HUOBI) CancelSwapTriggerOrder(contractCode currency.Pair, orderID string) (CancelTriggerOrdersData, error) {
	var resp CancelTriggerOrdersData
	req := make(map[string]interface{})
	req["contract_code"] = contractCode
	req["order_id"] = orderID
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapCancelTriggerOrder, nil, req, &resp)
}

// CancelAllSwapTriggerOrders cancels all swap trigger orders
func (h *HUOBI) CancelAllSwapTriggerOrders(contractCode currency.Pair) (CancelTriggerOrdersData, error) {
	var resp CancelTriggerOrdersData
	req := make(map[string]interface{})
	codeValue, err := h.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapCancelAllTriggerOrders, nil, req, &resp)
}

// GetSwapTriggerOrderHistory gets history for swap trigger orders
func (h *HUOBI) GetSwapTriggerOrderHistory(contractCode currency.Pair, status, tradeType string, createDate, pageIndex, pageSize int64) (TriggerOrderHistory, error) {
	var resp TriggerOrderHistory
	req := make(map[string]interface{})
	codeValue, err := h.FormatSymbol(contractCode, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	req["contract_code"] = codeValue
	req["status"] = status
	tType, ok := validTradeType[tradeType]
	if !ok {
		return resp, fmt.Errorf("invalid trade type")
	}
	req["trade_type"] = tType
	if createDate > 90 {
		return resp, fmt.Errorf("invalid create date value: only supports up to 90 days")
	}
	req["create_date"] = strconv.FormatInt(createDate, 10)
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize > 0 && pageSize <= 50 {
		req["page_size"] = pageSize
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, huobiSwapTriggerOrderHistory, nil, req, &resp)
}

// GetSwapMarkets gets data of swap markets
func (h *HUOBI) GetSwapMarkets(contract currency.Pair) ([]SwapMarketsData, error) {
	vals := url.Values{}
	if contract != (currency.Pair{}) {
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
	err := h.SendHTTPRequest(exchange.RestFutures, huobiSwapMarkets+vals.Encode(), &result)
	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Data, err
}

// GetSwapFundingRates gets funding rates data
func (h *HUOBI) GetSwapFundingRates(contract currency.Pair) (FundingRatesData, error) {
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
	err = h.SendHTTPRequest(exchange.RestFutures, huobiSwapFunding+vals.Encode(), &result)
	if result.ErrorMessage != "" {
		return FundingRatesData{}, errors.New(result.ErrorMessage)
	}
	return result.Data, err
}
