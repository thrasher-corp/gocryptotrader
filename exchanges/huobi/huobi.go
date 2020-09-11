package huobi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	huobiAPIURL      = "https://api.huobi.pro"
	huobiURL         = "https://api.hbdm.com/"
	huobiAPIVersion  = "1"
	huobiAPIVersion2 = "2"

	// Futures endpoints
	fContractInfo              = "api/v1/contract_contract_info?"
	fContractIndexPrice        = "api/v1/contract_index?"
	fContractPriceLimitation   = "api/v1/contract_price_limit?"
	fContractOpenInterest      = "api/v1/contract_open_interest?"
	fEstimatedDeliveryPrice    = "api/v1/contract_delivery_price?"
	fContractMarketDepth       = "/market/depth?"
	fContractKline             = "/market/history/kline?"
	fMarketOverview            = "/market/detail/merged?"
	fLastTradeContract         = "/market/trade?"
	fContractBatchTradeRecords = "/market/history/trade?"
	fInsuranceAndClawback      = "api/v1/contract_risk_info?"
	fInsuranceBalanceHistory   = "api/v1/contract_insurance_fund?"
	fTieredAdjustmentFactor    = "api/v1/contract_adjustfactor?"
	fHisContractOpenInterest   = "api/v1/contract_his_open_interest?"
	fSystemStatus              = "api/v1/contract_api_state"
	fTopAccountsSentiment      = "api/v1/contract_elite_account_ratio?"
	fTopPositionsSentiment     = "api/v1/contract_elite_position_ratio?"
	fLiquidationOrders         = "api/v1/contract_liquidation_orders?"
	fIndexKline                = "/index/market/history/index?"
	fBasisData                 = "/index/market/history/basis?"

	fAccountData               = "api/v1/contract_account_info"
	fPositionInformation       = "api/v1/contract_position_info"
	fAllSubAccountAssets       = "api/v1/contract_sub_account_list"
	fSingleSubAccountAssets    = "api/v1/contract_sub_account_info"
	fSingleSubAccountPositions = "api/v1/contract_sub_position_info"
	fFinancialRecords          = "api/v1/contract_financial_record"
	fSettlementRecords         = "api/v1/contract_user_settlement_records"
	fOrderLimitInfo            = "api/v1/contract_order_limit"
	fContractTradingFee        = "api/v1/contract_fee"
	fTransferLimitInfo         = "api/v1/contract_transfer_limit"
	fPositionLimitInfo         = "api/v1/contract_position_limit"
	fQueryAssetsAndPositions   = "api/v1/contract_account_position_info"
	fTransfer                  = "api/v1/contract_master_sub_transfer"
	fTransferRecords           = "api/v1/contract_master_sub_transfer_record"
	fAvailableLeverage         = "api/v1/contract_available_level_rate"
	fOrder                     = "api/v1/contract_order"
	fBatchOrder                = "/v1/contract_batchorder"
	fCancelOrder               = "api/v1/contract_cancel"
	fCancelAllOrders           = "api/v1/contract_cancelall"
	fFlashCloseOrder           = "api/v1/lightning_close_position"
	fOrderInfo                 = "api/v1/contract_order_info"
	fOrderDetails              = "api/v1/contract_order_detail"
	fQueryOpenOrders           = "api/v1/contract_openorders"
	fOrderHistory              = "api/v1/contract_hisorders"
	fMatchResult               = "api/v1/contract_matchresults"
	fTriggerOrder              = "api/v1/contract_trigger_order"
	fCancelTriggerOrder        = "api/v1/contract_trigger_cancel"
	fCancelAllTriggerOrders    = "api/v1/contract_trigger_cancelall"
	fTriggerOpenOrders         = "api/v1/contract_trigger_openorders"
	fTriggerOrderHistory       = "api/v1/contract_trigger_hisorders"

	// Coin Margined Swap (perpetual futures) endpoints
	huobiSwapMarkets                     = "/swap-api/v1/swap_contract_info?"
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
	huobiSwapFundingRate                 = "swap-api/v1/swap_funding_rate?"
	huobiSwapHistoricalFundingRate       = "swap-api/v1/swap_historical_funding_rate?"
	huobiPremiumIndexKlineData           = "index/market/history/swap_premium_index_kline?"
	huobiPredictedFundingRateData        = "index/market/history/swap_estimated_rate_kline?"
	huobiBasisData                       = "index/market/history/swap_basis?"
	huobiSwapAccInfo                     = "swap-api/v1/swap_account_info"
	huobiSwapPosInfo                     = "swap-api/v1/swap_position_info"
	huobiSwapAssetsAndPosInfo            = "swap-api/v1/swap_account_position_info"
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

	// Spot endpoints
	huobiMarketHistoryKline    = "market/history/kline"
	huobiMarketDetail          = "market/detail"
	huobiMarketDetailMerged    = "market/detail/merged"
	huobiMarketDepth           = "market/depth"
	huobiMarketTrade           = "market/trade"
	huobiMarketTickers         = "market/tickers"
	huobiMarketTradeHistory    = "market/history/trade"
	huobiSymbols               = "common/symbols"
	huobiCurrencies            = "common/currencys"
	huobiTimestamp             = "common/timestamp"
	huobiAccounts              = "account/accounts"
	huobiAccountBalance        = "account/accounts/%s/balance"
	huobiAccountDepositAddress = "account/deposit/address"
	huobiAccountWithdrawQuota  = "account/withdraw/quota"
	huobiAggregatedBalance     = "subuser/aggregate-balance"
	huobiOrderPlace            = "order/orders/place"
	huobiOrderCancel           = "order/orders/%s/submitcancel"
	huobiOrderCancelBatch      = "order/orders/batchcancel"
	huobiBatchCancelOpenOrders = "order/orders/batchCancelOpenOrders"
	huobiGetOrder              = "order/orders/getClientOrder"
	huobiGetOrderMatch         = "order/orders/%s/matchresults"
	huobiGetOrders             = "order/orders"
	huobiGetOpenOrders         = "order/openOrders"
	huobiGetOrdersMatch        = "orders/matchresults"
	huobiMarginTransferIn      = "dw/transfer-in/margin"
	huobiMarginTransferOut     = "dw/transfer-out/margin"
	huobiMarginOrders          = "margin/orders"
	huobiMarginRepay           = "margin/orders/%s/repay"
	huobiMarginLoanOrders      = "margin/loan-orders"
	huobiMarginAccountBalance  = "margin/accounts/balance"
	huobiWithdrawCreate        = "dw/withdraw/api/create"
	huobiWithdrawCancel        = "dw/withdraw-virtual/%s/cancel"
	huobiStatusError           = "error"
	huobiMarginRates           = "margin/loan-info"
)

var validPeriods = []string{"5min", "15min", "30min", "60min", "4hour", "1day"}

var validBasisPriceTypes = []string{"open", "close", "high", "low", "average"}

var validAmountType = map[string]int64{
	"cont":           1,
	"cryptocurrency": 2,
}

var validTransferType = []string{
	"master_to_sub", "sub_to_master",
}

var validTradeTypes = map[string]int64{
	"filled": 0,
	"closed": 5,
	"open":   6,
}

var validOrderType = map[string]int64{
	"quotation":         1,
	"cancelledOrder":    2,
	"forcedLiquidation": 3,
	"deliveryOrder":     4,
}

var validOrderTypes = []string{
	"limit", "opponent", "lightning", "optimal_5", "optimal_10", "optimal_20",
	"fok", "ioc", "opponent_ioc", "lightning_ioc", "optimal_5_ioc",
	"optimal_10_ioc", "optimal_20_ioc", "opponent_fok", "optimal_20_fok",
}

var validTriggerType = map[string]string{
	"greaterOrEqual": "ge",
	"smallerOrEqual": "le",
}

var validOrderPriceType = []string{
	"limit", "optimal_5", "optimal_10", "optimal_20",
}

var validLightningOrderPriceType = []string{
	"lightning", "lightning_fok", "lightning_ioc",
}

var validTradeType = map[string]int64{
	"all":            0,
	"openLong":       1,
	"closeShort":     2,
	"openShort":      3,
	"closeLong":      4,
	"liquidateLong":  5,
	"liquidateShort": 6,
}

var validContractTypes = []string{
	"this_week", "next_week", "quarter", "next_quarter",
}

var validFuturesPeriods = []string{
	"1min", "5min", "15min", "30min", "60min", "1hour", "4hour", "1day",
}

var validFuturesOrderPriceTypes = []string{
	"limit", "opponent", "lightning", "optimal_5", "optimal_10",
	"optimal_20", "fok", "ioc", "opponent_ioc", "lightning_ioc",
	"optimal_5_ioc", "optimal_10_ioc", "optimal_20_ioc", "opponent_fok",
	"lightning_fok", "optimal_5_fok", "optimal_10_fok", "optimal_20_fok",
}

var validFuturesRecordTypes = map[string]string{
	"closeLong":                   "3",
	"closeShort":                  "4",
	"openOpenPositionsTakerFees":  "5",
	"openPositionsMakerFees":      "6",
	"closePositionsTakerFees":     "7",
	"closePositionsMakerFees":     "8",
	"closeLongDelivery":           "9",
	"closeShortDelivery":          "10",
	"deliveryFee":                 "11",
	"longLiquidationClose":        "12",
	"shortLiquidationClose":       "13",
	"transferFromSpotToContracts": "14",
	"transferFromContractsToSpot": "15",
	"settleUnrealizedLongPNL":     "16",
	"settleUnrealizedShortPNL":    "17",
	"clawback":                    "19",
	"system":                      "26",
	"activityPrizeRewards":        "28",
	"rebate":                      "29",
	"transferToSub":               "34",
	"transferFromSub":             "35",
	"transferToMaster":            "36",
	"transferFromMaster":          "37",
}

var validOffsetTypes = []string{
	"open", "close",
}

var validOPTypes = []string{
	"lightning", "lightning_fok", "lightning_ioc",
}

var validFuturesReqType = map[string]int64{
	"all":            1,
	"finishedStatus": 2,
}

var validFuturesOrderTypes = map[string]int64{
	"limit":        1,
	"opponent":     3,
	"lightning":    4,
	"triggerOrder": 5,
	"postOnly":     6,
	"optimal_5":    7,
	"optimal_10":   8,
	"optimal_20":   9,
	"fok":          10,
	"ioc":          11,
}

var validStatusTypes = map[string]int64{
	"all":       0,
	"success":   4,
	"failed":    5,
	"cancelled": 6,
}

// HUOBI is the overarching type across this package
type HUOBI struct {
	exchange.Base
	AccountID string
}

// Futures Contracts

// FGetContractInfo gets contract info for futures
func (h *HUOBI) FGetContractInfo(symbol, contractType, code string) (FContractInfoData, error) {
	var resp FContractInfoData
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if common.StringDataCompare(validContractTypes, contractType) {
		params.Set("contract_type", contractType)
	}
	if code != "" {
		params.Set("contract_code", code)
	}
	path := huobiURL + fContractInfo + params.Encode()
	return resp, h.SendHTTPRequest(path, &resp)
}

// FIndexPriceInfo gets index price info for a futures contract
func (h *HUOBI) FIndexPriceInfo(symbol string) (FContractIndexPriceInfo, error) {
	var resp FContractIndexPriceInfo
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	path := huobiURL + fContractIndexPrice + params.Encode()
	return resp, h.SendHTTPRequest(path, &resp)
}

// FContractPriceLimitations gets price limits for a futures contract
func (h *HUOBI) FContractPriceLimitations(symbol, contractType, code string) (FContractIndexPriceInfo, error) {
	var resp FContractIndexPriceInfo
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if common.StringDataCompare(validContractTypes, contractType) {
		params.Set("contract_type", contractType)
	}
	if code != "" {
		params.Set("contract_code", code)
	}
	path := huobiURL + fContractPriceLimitation + params.Encode()
	return resp, h.SendHTTPRequest(path, &resp)
}

// FContractOpenInterest gets open interest data for futures contracts
func (h *HUOBI) FContractOpenInterest(symbol, contractType, code string) (FContractOIData, error) {
	var resp FContractOIData
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if common.StringDataCompare(validContractTypes, contractType) {
		params.Set("contract_type", contractType)
	}
	if code != "" {
		params.Set("contract_code", code)
	}
	path := huobiURL + fContractOpenInterest + params.Encode()
	return resp, h.SendHTTPRequest(path, &resp)
}

// FGetEstimatedDeliveryPrice gets estimated delivery price info for futures
func (h *HUOBI) FGetEstimatedDeliveryPrice(symbol string) (FEstimatedDeliveryPriceInfo, error) {
	var resp FEstimatedDeliveryPriceInfo
	params := url.Values{}
	params.Set("symbol", symbol)
	path := huobiURL + fEstimatedDeliveryPrice + params.Encode()
	return resp, h.SendHTTPRequest(path, &resp)
}

// FGetMarketDepth gets market depth data for futures contracts
func (h *HUOBI) FGetMarketDepth(symbol, dataType string) (OBData, error) {
	var resp OBData
	var tempData FMarketDepth
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("type", dataType)
	path := huobiURL + fContractMarketDepth + params.Encode()
	err := h.SendHTTPRequest(path, &tempData)
	if err != nil {
		return resp, err
	}
	resp.Symbol = symbol
	for x := range tempData.Tick.Asks {
		resp.Asks = append(resp.Asks, obItem{
			Price:    tempData.Tick.Asks[x][0],
			Quantity: tempData.Tick.Bids[x][1],
		})
	}
	for y := range tempData.Tick.Bids {
		resp.Bids = append(resp.Bids, obItem{
			Price:    tempData.Tick.Bids[y][0],
			Quantity: tempData.Tick.Bids[y][1],
		})
	}
	return resp, nil
}

// FGetKlineData gets kline data for futures
func (h *HUOBI) FGetKlineData(symbol, period string, size int64, startTime, endTime time.Time) (FKlineData, error) {
	var resp FKlineData
	params := url.Values{}
	params.Set("symbol", symbol)
	if !common.StringDataCompare(validFuturesPeriods, period) {
		return resp, fmt.Errorf("invalid period value received")
	}
	params.Set("period", period)
	if !(size > 1) && !(size < 2000) {
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
	path := huobiURL + fContractKline + params.Encode()
	return resp, h.SendHTTPRequest(path, &resp)
}

// FGetMarketOverviewData gets market overview data for futures
func (h *HUOBI) FGetMarketOverviewData(symbol string) (FMarketOverviewData, error) {
	var resp FMarketOverviewData
	params := url.Values{}
	params.Set("symbol", symbol)
	path := huobiURL + fMarketOverview + params.Encode()
	return resp, h.SendHTTPRequest(path, &resp)
}

// FLastTradeData gets last trade data for a futures contract
func (h *HUOBI) FLastTradeData(symbol string) (FLastTradeData, error) {
	var resp FLastTradeData
	params := url.Values{}
	params.Set("symbol", symbol)
	path := huobiURL + fLastTradeContract + params.Encode()
	return resp, h.SendHTTPRequest(path, &resp)
}

// FRequestPublicBatchTrades gets public batch trades for a futures contract
func (h *HUOBI) FRequestPublicBatchTrades(symbol string, size int64) (FBatchTradesForContractData, error) {
	var resp FBatchTradesForContractData
	params := url.Values{}
	params.Set("symbol", symbol)
	if size > 1 && size < 2000 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	path := huobiURL + fContractBatchTradeRecords + params.Encode()
	return resp, h.SendHTTPRequest(path, &resp)
}

// FQueryInsuranceAndClawbackData gets insurance and clawback data for a futures contract
func (h *HUOBI) FQueryInsuranceAndClawbackData(symbol string) (FClawbackRateAndInsuranceData, error) {
	var resp FClawbackRateAndInsuranceData
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	path := huobiURL + fInsuranceAndClawback + params.Encode()
	return resp, h.SendHTTPRequest(path, &resp)
}

// FQueryHistoricalInsuranceData gets insurance data
func (h *HUOBI) FQueryHistoricalInsuranceData(symbol string) (FHistoricalInsuranceRecordsData, error) {
	var resp FHistoricalInsuranceRecordsData
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	path := huobiURL + fInsuranceBalanceHistory + params.Encode()
	return resp, h.SendHTTPRequest(path, &resp)
}

// FQueryTieredAdjustmentFactor gets tiered adjustment factor for futures contracts
func (h *HUOBI) FQueryTieredAdjustmentFactor(symbol string) (FTieredAdjustmentFactorInfo, error) {
	var resp FTieredAdjustmentFactorInfo
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	path := huobiURL + fTieredAdjustmentFactor + params.Encode()
	return resp, h.SendHTTPRequest(path, &resp)
}

// FQueryHisOpenInterest gets open interest for futures contract
func (h *HUOBI) FQueryHisOpenInterest(symbol string) (FContractOIData, error) {
	var resp FContractOIData
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	path := huobiURL + fHisContractOpenInterest + params.Encode()
	return resp, h.SendHTTPRequest(path, &resp)
}

// FQuerySystemStatus gets system status data
func (h *HUOBI) FQuerySystemStatus(symbol string) (FContractOIData, error) {
	var resp FContractOIData
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	path := huobiURL + fHisContractOpenInterest + params.Encode()
	return resp, h.SendHTTPRequest(path, &resp)
}

// FQueryTopAccountsRatio gets top accounts' ratio
func (h *HUOBI) FQueryTopAccountsRatio(symbol, period string) (FTopAccountsLongShortRatio, error) {
	var resp FTopAccountsLongShortRatio
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if !common.StringDataCompare(validPeriods, period) {
		return resp, fmt.Errorf("invalid period")
	}
	params.Set("period", period)
	path := huobiURL + fTopAccountsSentiment + params.Encode()
	return resp, h.SendHTTPRequest(path, &resp)
}

// FQueryTopPositionsRatio gets top positions' long/short ratio for futures
func (h *HUOBI) FQueryTopPositionsRatio(symbol, period string) (FTopPositionsLongShortRatio, error) {
	var resp FTopPositionsLongShortRatio
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if !common.StringDataCompare(validPeriods, period) {
		return resp, fmt.Errorf("invalid period")
	}
	params.Set("period", period)
	path := huobiURL + fTopPositionsSentiment + params.Encode()
	return resp, h.SendHTTPRequest(path, &resp)
}

// FLiquidationOrders gets liquidation orders for futures contracts
func (h *HUOBI) FLiquidationOrders(code, tradeType string, pageIndex, pageSize, createDate int64) (FLiquidationOrdersInfo, error) {
	var resp FLiquidationOrdersInfo
	params := url.Values{}
	params.Set("contract_code", code)
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
	path := huobiURL + fLiquidationOrders + params.Encode()
	return resp, h.SendHTTPRequest(path, &resp)
}

// FIndexKline gets index kline data for futures contracts
func (h *HUOBI) FIndexKline(symbol, period string, size int64) (FIndexKlineData, error) {
	var resp FIndexKlineData
	params := url.Values{}
	params.Set("symbol", symbol)
	if !common.StringDataCompare(validFuturesPeriods, period) {
		return resp, fmt.Errorf("invalid period value received")
	}
	params.Set("period", period)
	if !(size > 1) && !(size < 2000) {
		return resp, fmt.Errorf("invalid size")
	}
	params.Set("size", strconv.FormatInt(size, 10))
	path := huobiURL + fIndexKline + params.Encode()
	return resp, h.SendHTTPRequest(path, &resp)
}

// FGetBasisData gets basis data futures contracts
func (h *HUOBI) FGetBasisData(symbol, period, basisPriceType string, size int64) (FBasisData, error) {
	var resp FBasisData
	params := url.Values{}
	params.Set("symbol", symbol)
	if !common.StringDataCompare(validFuturesPeriods, period) {
		return resp, fmt.Errorf("invalid period value received")
	}
	params.Set("period", period)
	params.Set("size", strconv.FormatInt(size, 10))
	if basisPriceType != "" {
		if common.StringDataCompare(validBasisPriceTypes, basisPriceType) {
			params.Set("basis_price_type", basisPriceType)
		}
	}
	path := huobiURL + fIndexKline + params.Encode()
	return resp, h.SendHTTPRequest(path, &resp)
}

// FGetAccountInfo gets user info for futures account
func (h *HUOBI) FGetAccountInfo(symbol string) (FUserAccountData, error) {
	var resp FUserAccountData
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fAccountData, nil, req, &resp, false)
}

// FGetPositionsInfo gets positions info for futures account
func (h *HUOBI) FGetPositionsInfo(symbol string) (FUserAccountData, error) {
	var resp FUserAccountData
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fPositionInformation, nil, req, &resp, false)
}

// FGetAllSubAccountAssets gets assets info for all futures subaccounts
func (h *HUOBI) FGetAllSubAccountAssets(symbol string) (FSubAccountAssetsInfo, error) {
	var resp FSubAccountAssetsInfo
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fAllSubAccountAssets, nil, req, &resp, false)
}

// FGetSingleSubAccountInfo gets assets info for a futures subaccount
func (h *HUOBI) FGetSingleSubAccountInfo(symbol, subUID string) (FSingleSubAccountAssetsInfo, error) {
	var resp FSingleSubAccountAssetsInfo
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	req["sub_uid"] = subUID
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fSingleSubAccountAssets, nil, req, &resp, false)
}

// FGetSingleSubPositions gets positions info for a single sub account
func (h *HUOBI) FGetSingleSubPositions(symbol, subUID string) (FSingleSubAccountPositionsInfo, error) {
	var resp FSingleSubAccountPositionsInfo
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	req["sub_uid"] = subUID
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fSingleSubAccountPositions, nil, req, &resp, false)
}

// FGetFinancialRecords gets financial records for futures
func (h *HUOBI) FGetFinancialRecords(symbol, recordType string, createDate, pageIndex, pageSize int64) (FFinancialRecords, error) {
	var resp FFinancialRecords
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	if recordType != "" {
		rType, ok := validFuturesRecordTypes[recordType]
		if !ok {
			return resp, fmt.Errorf("invalid recordType")
		}
		req["type"] = rType
	}
	if createDate > 0 && createDate < 90 {
		req["create_date"] = createDate
	}
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fFinancialRecords, nil, req, &resp, false)
}

// FGetSettlementRecords gets settlement records for futures
func (h *HUOBI) FGetSettlementRecords(symbol string, pageIndex, pageSize int64, startTime, endTime time.Time) (FSettlementRecords, error) {
	var resp FSettlementRecords
	req := make(map[string]interface{})
	req["symbol"] = symbol
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		req["start_time"] = strconv.FormatInt(startTime.Unix(), 10)
		req["end_time"] = strconv.FormatInt(endTime.Unix(), 10)
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fFinancialRecords, nil, req, &resp, false)
}

// FGetOrderLimits gets order limits for futures contracts
func (h *HUOBI) FGetOrderLimits(symbol, orderPriceType string) (FContractInfoOnOrderLimit, error) {
	var resp FContractInfoOnOrderLimit
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	if orderPriceType != "" {
		if !common.StringDataCompare(validFuturesOrderPriceTypes, orderPriceType) {
			return resp, fmt.Errorf("invalid orderPriceType")
		}
		req["order_price_type"] = orderPriceType
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fOrderLimitInfo, nil, req, &resp, false)
}

// FContractTradingFee gets futures contract trading fees
func (h *HUOBI) FContractTradingFee(symbol string) (FContractTradingFeeData, error) {
	var resp FContractTradingFeeData
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fContractTradingFee, nil, req, &resp, false)
}

// FGetTransferLimits gets transfer limits for futures
func (h *HUOBI) FGetTransferLimits(symbol string) (FTransferLimitData, error) {
	var resp FTransferLimitData
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fTransferLimitInfo, nil, req, &resp, false)
}

// FGetPositionLimits gets position limits for futures
func (h *HUOBI) FGetPositionLimits(symbol string) (FPositionLimitData, error) {
	var resp FPositionLimitData
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fPositionLimitInfo, nil, req, &resp, false)
}

// FGetAssetsAndPositions gets assets and positions for futures
func (h *HUOBI) FGetAssetsAndPositions(symbol string) (FAssetsAndPositionsData, error) {
	var resp FAssetsAndPositionsData
	req := make(map[string]interface{})
	req["symbol"] = symbol
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fQueryAssetsAndPositions, nil, req, &resp, false)
}

// FTransfer transfers assets between master and subaccounts
func (h *HUOBI) FTransfer(subUID, symbol, transferType string, amount float64) (FAccountTransferData, error) {
	var resp FAccountTransferData
	req := make(map[string]interface{})
	req["symbol"] = symbol
	req["subUid"] = subUID
	req["amount"] = amount
	if !common.StringDataCompare(validTransferType, transferType) {
		return resp, fmt.Errorf("inavlid transferType received")
	}
	req["type"] = transferType
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fTransfer, nil, req, &resp, false)
}

// FGetTransferRecords gets transfer records data for futures
func (h *HUOBI) FGetTransferRecords(symbol, transferType string, createDate int64, pageIndex, pageSize int64) (FTransferRecords, error) {
	var resp FTransferRecords
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	if !common.StringDataCompare(validTransferType, transferType) {
		return resp, fmt.Errorf("inavlid transferType received")
	}
	req["type"] = transferType
	if createDate < 0 && createDate > 90 {
		return resp, fmt.Errorf("invalid create date value: only supports up to 90 days")
	}
	req["create_date"] = strconv.FormatInt(createDate, 10)
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize > 0 && pageSize <= 50 {
		req["page_size"] = pageSize
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fTransferRecords, nil, req, &resp, false)
}

// FGetAvailableLeverage gets available leverage data for futures
func (h *HUOBI) FGetAvailableLeverage(symbol string) (FAvailableLeverageData, error) {
	var resp FAvailableLeverageData
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fAvailableLeverage, nil, req, &resp, false)
}

// FOrder places an order for futures
func (h *HUOBI) FOrder(symbol, contractType, contractCode, clientOrderID, direction, offset, orderPriceType string, price, volume, leverageRate float64) (FOrderData, error) {
	var resp FOrderData
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	if contractType != "" {
		if common.StringDataCompare(validContractTypes, contractType) {
			return resp, fmt.Errorf("invalid contractType")
		}
		req["contract_type"] = contractType
	}
	if contractCode != "" {
		req["contract_code"] = contractCode
	}
	if clientOrderID != "" {
		req["client_order_id"] = clientOrderID
	}
	req["direction"] = direction
	if common.StringDataCompare(validOffsetTypes, offset) {
		return resp, fmt.Errorf("invalid offset amounts")
	}
	if !common.StringDataCompare(validFuturesOrderPriceTypes, orderPriceType) {
		return resp, fmt.Errorf("invalid orderType")
	}
	req["order_price_type"] = orderPriceType
	req["lever_rate"] = leverageRate
	req["volume"] = volume
	req["price"] = price
	req["offset"] = offset
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fOrder, nil, req, &resp, false)
}

// FCancelOrder cancels a futures order
func (h *HUOBI) FCancelOrder(symbol, orderID, clientOrderID string) (FCancelOrderData, error) {
	var resp FCancelOrderData
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	if orderID != "" {
		req["order_id"] = orderID
	}
	if clientOrderID != "" {
		req["client_order_id"] = clientOrderID
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fCancelOrder, nil, req, &resp, false)
}

// FCancelAllOrders cancels all futures order for a given symbol
func (h *HUOBI) FCancelAllOrders(symbol, contractCode, contractType string) (FCancelOrderData, error) {
	var resp FCancelOrderData
	req := make(map[string]interface{})
	req["symbol"] = symbol
	if contractType != "" {
		if !common.StringDataCompare(validContractTypes, contractType) {
			return resp, fmt.Errorf("invalid contractType")
		}
		req["contract_type"] = contractType
	}
	if contractCode != "" {
		req["contract_code"] = contractCode
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fCancelOrder, nil, req, &resp, false)
}

// FFlashCloseOrder flash closes a futures order
func (h *HUOBI) FFlashCloseOrder(symbol, contractType, contractCode, direction, orderPriceType, clientOrderID string, volume float64) (FFlashCloseOrderData, error) {
	var resp FFlashCloseOrderData
	req := make(map[string]interface{})
	req["symbol"] = symbol
	if contractType != "" {
		if !common.StringDataCompare(validContractTypes, contractType) {
			return resp, fmt.Errorf("invalid contractType")
		}
		req["contract_type"] = contractType
	}
	if contractCode != "" {
		req["contract_code"] = contractCode
	}
	req["direction"] = direction
	req["volume"] = volume
	if clientOrderID != "" {
		req["client_order_id"] = clientOrderID
	}
	if orderPriceType != "" {
		if !common.StringDataCompare(validOPTypes, orderPriceType) {
			return resp, fmt.Errorf("invalid orderPriceType")
		}
		req["orderPriceType"] = orderPriceType
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fFlashCloseOrder, nil, req, &resp, false)
}

// FGetOrderInfo gets order info for futures
func (h *HUOBI) FGetOrderInfo(symbol, clientOrderID, orderID string) (FOrderInfo, error) {
	var resp FOrderInfo
	req := make(map[string]interface{})
	req["symbol"] = symbol
	if orderID != "" {
		req["order_id"] = orderID
	}
	if clientOrderID != "" {
		req["client_order_id"] = clientOrderID
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fOrderInfo, nil, req, &resp, false)
}

// FOrderDetails gets order details for futures orders
func (h *HUOBI) FOrderDetails(symbol, orderID, orderType string, createdAt time.Time, pageIndex, pageSize int64) (FOrderDetailsData, error) {
	var resp FOrderDetailsData
	req := make(map[string]interface{})
	req["symbol"] = symbol
	req["order_id"] = orderID
	req["created_at"] = strconv.FormatInt(createdAt.Unix(), 10)
	oType, ok := validOrderType[orderType]
	if !ok {
		return resp, fmt.Errorf("invalid orderType")
	}
	req["order_type"] = oType
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fOrderDetails, nil, req, &resp, false)
}

// FGetOpenOrders gets order details for futures orders
func (h *HUOBI) FGetOpenOrders(symbol string, pageIndex, pageSize int64) (FOpenOrdersData, error) {
	var resp FOpenOrdersData
	req := make(map[string]interface{})
	req["symbol"] = symbol
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fQueryOpenOrders, nil, req, &resp, false)
}

// FGetOrderHistory gets order order history for futures
func (h *HUOBI) FGetOrderHistory(symbol, tradeType, reqType, status, contractCode, orderType string, createDate, pageIndex, pageSize int64) (FOrderHistoryData, error) {
	var resp FOrderHistoryData
	req := make(map[string]interface{})
	req["symbol"] = symbol
	tType, ok := validTradeType[tradeType]
	if !ok {
		return resp, fmt.Errorf("invalid tradeType")
	}
	req["trade_type"] = tType
	rType, ok := validFuturesReqType[reqType]
	if !ok {
		return resp, fmt.Errorf("invalid reqType")
	}
	req["type"] = rType
	req["status"] = status
	if createDate < 0 || createDate > 90 {
		return resp, fmt.Errorf("invalid createDate")
	}
	req["create_date"] = createDate
	if contractCode != "" {
		req["contract_code"] = contractCode
	}
	if orderType != "" {
		oType, ok := validFuturesOrderTypes[orderType]
		if !ok {
			return resp, fmt.Errorf("invalid orderType")
		}
		req["order_type"] = oType
	}
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fOrderHistory, nil, req, &resp, false)
}

// FTradeHistory gets trade history data for futures
func (h *HUOBI) FTradeHistory(symbol, tradeType, contractCode string, createDate, pageIndex, pageSize int64) (FOrderHistoryData, error) {
	var resp FOrderHistoryData
	req := make(map[string]interface{})
	req["symbol"] = symbol
	tType, ok := validTradeType[tradeType]
	if !ok {
		return resp, fmt.Errorf("invalid tradeType")
	}
	req["trade_type"] = tType
	if contractCode != "" {
		req["contract_code"] = contractCode
	}
	if createDate <= 0 || createDate > 90 {
		return resp, fmt.Errorf("invalid createDate")
	}
	req["create_date"] = createDate
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fMatchResult, nil, req, &resp, false)
}

// FPlaceTriggerOrder places a trigger order for futures
func (h *HUOBI) FPlaceTriggerOrder(symbol, contractType, contractCode, triggerType, orderPriceType, direction, offset string, triggerPrice, orderPrice, volume, leverageRate float64) (FTriggerOrderData, error) {
	var resp FTriggerOrderData
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	if contractCode != "" {
		req["contract_code"] = contractCode
	}
	tType, ok := validTriggerType[triggerType]
	if !ok {
		return resp, fmt.Errorf("invalid trigger type")
	}
	req["trigger_type"] = tType
	req["direction"] = direction
	if !common.StringDataCompare(validOffsetTypes, offset) {
		return resp, fmt.Errorf("invalid offset")
	}
	req["offset"] = offset
	req["trigger_price"] = triggerPrice
	req["volume"] = volume
	req["lever_rate"] = leverageRate
	req["order_price"] = orderPrice
	if !common.StringDataCompare(validOrderPriceType, orderPriceType) {
		return resp, fmt.Errorf("invalid order price type")
	}
	req["order_price_type"] = orderPriceType
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fTriggerOrder, nil, req, &resp, false)
}

// FCancelTriggerOrder cancels trigger order for futures
func (h *HUOBI) FCancelTriggerOrder(symbol, orderID string) (FCancelTriggerOrdersData, error) {
	var resp FCancelTriggerOrdersData
	req := make(map[string]interface{})
	req["symbol"] = symbol
	req["order_id"] = orderID
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fCancelTriggerOrder, nil, req, &resp, false)
}

// FCancelAllTriggerOrders cancels all trigger order for futures
func (h *HUOBI) FCancelAllTriggerOrders(symbol, contractCode, contractType string) (FCancelTriggerOrdersData, error) {
	var resp FCancelTriggerOrdersData
	req := make(map[string]interface{})
	req["symbol"] = symbol
	if contractCode != "" {
		req["contract_code"] = contractCode
	}
	if contractType != "" {
		if !common.StringDataCompare(validContractTypes, contractType) {
			return resp, nil
		}
		req["contract_type"] = contractType
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fCancelAllTriggerOrders, nil, req, &resp, false)
}

// FQueryTriggerOpenOrders queries open trigger orders for futures
func (h *HUOBI) FQueryTriggerOpenOrders(symbol, contractCode string, pageIndex, pageSize int64) (FTriggerOpenOrders, error) {
	var resp FTriggerOpenOrders
	req := make(map[string]interface{})
	req["symbol"] = symbol
	if contractCode != "" {
		req["contract_code"] = contractCode
	}
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fCancelAllTriggerOrders, nil, req, &resp, false)
}

// FQueryTriggerOrderHistory queries trigger order history for futures
func (h *HUOBI) FQueryTriggerOrderHistory(symbol, contractCode, tradeType, status string, createDate, pageIndex, pageSize int64) (FTriggerOrderHistoryData, error) {
	var resp FTriggerOrderHistoryData
	req := make(map[string]interface{})
	req["symbol"] = symbol
	if contractCode != "" {
		req["contract_code"] = contractCode
	}
	if tradeType != "" {
		tType, ok := validTradeType[tradeType]
		if !ok {
			return resp, fmt.Errorf("invalid tradeType")
		}
		req["trade_type"] = tType
	}
	validStatus, ok := validStatusTypes[status]
	if !ok {
		return resp, fmt.Errorf("invalid status")
	}
	req["status"] = validStatus
	if createDate <= 0 || createDate > 90 {
		return resp, fmt.Errorf("invalid createDate")
	}
	req["create_date"] = createDate
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, fTriggerOrderHistory, nil, req, &resp, false)
}

//

//

//

//

//

//

// Coin Margined Swaps

// QuerySwapIndexPriceInfo gets perpetual swap index's price info
func (h *HUOBI) QuerySwapIndexPriceInfo(code string) (SwapIndexPriceData, error) {
	var resp SwapIndexPriceData
	path := huobiURL + huobiSwapIndexPriceInfo
	if code != "" {
		params := url.Values{}
		params.Set("contract_code", code)
		path = huobiURL + huobiSwapIndexPriceInfo + params.Encode()
	}
	return resp, h.SendHTTPRequest(path, &resp)
}

// GetSwapPriceLimits gets price caps for perpetual futures
func (h *HUOBI) GetSwapPriceLimits(code string) (SwapPriceLimitsData, error) {
	var resp SwapPriceLimitsData
	params := url.Values{}
	params.Set("contract_code", code)
	return resp, h.SendHTTPRequest(huobiURL+huobiSwapPriceLimitation+params.Encode(),
		&resp)
}

// SwapOpenInterestInformation gets open interest data for perpetual futures
func (h *HUOBI) SwapOpenInterestInformation(code string) (SwapOpenInterestData, error) {
	var resp SwapOpenInterestData
	params := url.Values{}
	params.Set("contract_code", code)
	return resp, h.SendHTTPRequest(huobiURL+huobiSwapOpenInterestInfo+params.Encode(), &resp)
}

// GetSwapMarketDepth gets market depth for perpetual futures
func (h *HUOBI) GetSwapMarketDepth(code, dataType string) (SwapMarketDepthData, error) {
	var resp SwapMarketDepthData
	params := url.Values{}
	params.Set("contract_code", code)
	params.Set("type", dataType)
	return resp, h.SendHTTPRequest(huobiURL+huobiSwapMarketDepth+params.Encode(), &resp)
}

// GetSwapKlineData gets kline data for perpetual futures
func (h *HUOBI) GetSwapKlineData(code, period string, size int64, startTime, endTime time.Time) (SwapKlineData, error) {
	var resp SwapKlineData
	params := url.Values{}
	params.Set("contract_code", code)
	if !common.StringDataCompare(validPeriods, period) {
		return resp, fmt.Errorf("invalid period value received")
	}
	params.Set("period", period)
	if !(size > 1) && !(size < 2000) {
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
	return resp, h.SendHTTPRequest(huobiURL+huobiKLineData+params.Encode(), &resp)
}

// GetSwapMarketOverview gets market data overview for perpetual futures
func (h *HUOBI) GetSwapMarketOverview(code string) (MarketOverviewData, error) {
	var resp MarketOverviewData
	params := url.Values{}
	params.Set("contract_code", code)
	return resp, h.SendHTTPRequest(huobiURL+huobiMarketDataOverview+params.Encode(), &resp)
}

// GetLastTrade gets the last trade for a given perpetual contract
func (h *HUOBI) GetLastTrade(code string) (LastTradeData, error) {
	var resp LastTradeData
	params := url.Values{}
	params.Set("contract_code", code)
	return resp, h.SendHTTPRequest(huobiURL+huobiLastTradeContract+params.Encode(), &resp)
}

// GetBatchTrades gets batch trades for a specified contract (fetching size cannot be bigger than 2000)
func (h *HUOBI) GetBatchTrades(code string, size int64) (BatchTradesData, error) {
	var resp BatchTradesData
	params := url.Values{}
	params.Set("contract_code", code)
	if !(size > 1) && !(size < 2000) {
		return resp, fmt.Errorf("invalid size")
	}
	params.Set("size", strconv.FormatInt(size, 10))
	return resp, h.SendHTTPRequest(huobiURL+huobiRequestBatchOfTradingRecords+params.Encode(), &resp)
}

// GetInsuranceData gets insurance fund data and clawback rates
func (h *HUOBI) GetInsuranceData(code string) (InsuranceAndClawbackData, error) {
	var resp InsuranceAndClawbackData
	params := url.Values{}
	params.Set("contract_code", code)
	return resp, h.SendHTTPRequest(huobiURL+huobiInsuranceBalanceAndClawbackRate+params.Encode(), &resp)
}

// GetHistoricalInsuranceData gets historical insurance fund data and clawback rates
func (h *HUOBI) GetHistoricalInsuranceData(code string, pageIndex, pageSize int64) (HistoricalInsuranceFundBalance, error) {
	var resp HistoricalInsuranceFundBalance
	params := url.Values{}
	params.Set("contract_code", code)
	if pageIndex != 0 {
		params.Set("page_index", strconv.FormatInt(pageIndex, 10))
	}
	if pageSize != 0 {
		params.Set("page_size", strconv.FormatInt(pageIndex, 10))
	}
	return resp, h.SendHTTPRequest(huobiURL+huobiInsuranceBalanceHistory+params.Encode(), &resp)
}

// GetTieredAjustmentFactorInfo gets tiered adjustment factor data
func (h *HUOBI) GetTieredAjustmentFactorInfo(code string) (TieredAdjustmentFactorData, error) {
	var resp TieredAdjustmentFactorData
	params := url.Values{}
	params.Set("contract_code", code)
	return resp, h.SendHTTPRequest(huobiURL+huobiTieredAdjustmentFactor+params.Encode(), &resp)
}

// GetOpenInterestInfo gets open interest data
func (h *HUOBI) GetOpenInterestInfo(code, period, amountType string, size int64) (OpenInterestData, error) {
	var resp OpenInterestData
	params := url.Values{}
	params.Set("contract_code", code)
	if !common.StringDataCompare(validPeriods, period) {
		return resp, fmt.Errorf("invalid period value received")
	}
	params.Set("period", period)
	if !(size > 0 && size <= 1200) {
		return resp, fmt.Errorf("invalid size provided values from 1-1200 supported")
	}
	params.Set("size", strconv.FormatInt(size, 10))
	aType, ok := validAmountType[amountType]
	if !ok {
		return resp, fmt.Errorf("invalid trade type")
	}
	params.Set("amount_type", strconv.FormatInt(aType, 10))
	return resp, h.SendHTTPRequest(huobiURL+huobiOpenInterestInfo+params.Encode(), &resp)
}

// GetSystemStatusInfo gets system status data
func (h *HUOBI) GetSystemStatusInfo(code, period, amountType string, size int64) (SystemStatusData, error) {
	var resp SystemStatusData
	params := url.Values{}
	params.Set("contract_code", code)
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
	return resp, h.SendHTTPRequest(huobiURL+huobiSwapSystemStatus+params.Encode(), &resp)
}

// GetTraderSentimentIndexAccount gets top trader sentiment function-account
func (h *HUOBI) GetTraderSentimentIndexAccount(code, period string) (TraderSentimentIndexAccountData, error) {
	var resp TraderSentimentIndexAccountData
	params := url.Values{}
	params.Set("contract_code", code)
	if !common.StringDataCompare(validPeriods, period) {
		return resp, fmt.Errorf("invalid period value received")
	}
	params.Set("period", period)
	return resp, h.SendHTTPRequest(huobiURL+huobiSwapSentimentAccountData+params.Encode(), &resp)
}

// GetTraderSentimentIndexPosition gets top trader sentiment function-position
func (h *HUOBI) GetTraderSentimentIndexPosition(code, period string) (TraderSentimentIndexPositionData, error) {
	var resp TraderSentimentIndexPositionData
	params := url.Values{}
	params.Set("contract_code", code)
	if !common.StringDataCompare(validPeriods, period) {
		return resp, fmt.Errorf("invalid period value received")
	}
	params.Set("period", period)
	return resp, h.SendHTTPRequest(huobiURL+huobiSwapSentimentPosition+params.Encode(), &resp)
}

// GetLiquidationOrders gets liquidation orders for a given perp
func (h *HUOBI) GetLiquidationOrders(code, tradeType string, pageIndex, pageSize, createDate int64) (LiquidationOrdersData, error) {
	var resp LiquidationOrdersData
	params := url.Values{}
	params.Set("contract_code", code)
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
	return resp, h.SendHTTPRequest(huobiURL+huobiSwapLiquidationOrders+params.Encode(), &resp)
}

// GetHistoricalFundingRates gets historical funding rates for perpetual futures
func (h *HUOBI) GetHistoricalFundingRates(code string, pageSize, pageIndex int64) (HistoricalFundingRateData, error) {
	var resp HistoricalFundingRateData
	params := url.Values{}
	params.Set("contract_code", code)
	if pageIndex != 0 {
		params.Set("page_index", strconv.FormatInt(pageIndex, 10))
	}
	if pageSize != 0 {
		params.Set("page_size", strconv.FormatInt(pageIndex, 10))
	}
	return resp, h.SendHTTPRequest(huobiURL+huobiSwapHistoricalFundingRate+params.Encode(), &resp)
}

// GetPremiumIndexKlineData gets kline data for premium index
func (h *HUOBI) GetPremiumIndexKlineData(code, period string, size int64) (PremiumIndexKlineData, error) {
	var resp PremiumIndexKlineData
	params := url.Values{}
	params.Set("contract_code", code)
	if !common.StringDataCompare(validPeriods, period) {
		return resp, fmt.Errorf("invalid period value received")
	}
	params.Set("period", period)
	if !(size > 1) && !(size < 2000) {
		return resp, fmt.Errorf("invalid size")
	}
	params.Set("size", strconv.FormatInt(size, 10))
	return resp, h.SendHTTPRequest(huobiURL+huobiPremiumIndexKlineData+params.Encode(), &resp)
}

// GetEstimatedFundingRates gets estimated funding rates for perpetual futures
func (h *HUOBI) GetEstimatedFundingRates(code, period string, size int64) (EstimatedFundingRateData, error) {
	var resp EstimatedFundingRateData
	params := url.Values{}
	params.Set("contract_code", code)
	if !common.StringDataCompare(validPeriods, period) {
		return resp, fmt.Errorf("invalid period value received")
	}
	params.Set("period", period)
	if !(size > 0 && size <= 1200) {
		return resp, fmt.Errorf("invalid size provided values from 1-1200 supported")
	}
	params.Set("size", strconv.FormatInt(size, 10))
	return resp, h.SendHTTPRequest(huobiURL+huobiPredictedFundingRateData+params.Encode(), &resp)
}

// GetBasisData gets basis data for perpetual futures
func (h *HUOBI) GetBasisData(code, period, basisPriceType string, size int64) (BasisData, error) {
	var resp BasisData
	params := url.Values{}
	params.Set("contract_code", code)
	if !common.StringDataCompare(validPeriods, period) {
		return resp, fmt.Errorf("invalid period value received")
	}
	params.Set("period", period)
	if !(size > 0 && size <= 1200) {
		return resp, fmt.Errorf("invalid size provided values from 1-1200 supported")
	}
	params.Set("size", strconv.FormatInt(size, 10))
	if !common.StringDataCompare(validBasisPriceTypes, basisPriceType) {
		return resp, fmt.Errorf("invalid period value received")
	}
	return resp, h.SendHTTPRequest(huobiURL+huobiBasisData+params.Encode(), &resp)
}

// GetSwapAccountInfo gets swap account info
func (h *HUOBI) GetSwapAccountInfo(code string) (SwapAccountInformation, error) {
	var resp SwapAccountInformation
	req := make(map[string]interface{})
	req["contract_code"] = code
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapAccInfo, nil, req, &resp, false)
}

// GetSwapPositionsInfo gets swap positions' info
func (h *HUOBI) GetSwapPositionsInfo(code string) (SwapPositionInfo, error) {
	var resp SwapPositionInfo
	req := make(map[string]interface{})
	req["contract_code"] = code
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapPosInfo, nil, req, &resp, false)
}

// GetSwapAssetsAndPositions gets swap positions and asset info
func (h *HUOBI) GetSwapAssetsAndPositions(code string) (SwapAssetsAndPositionsData, error) {
	var resp SwapAssetsAndPositionsData
	req := make(map[string]interface{})
	req["contract_code"] = code
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapAssetsAndPosInfo, nil, req, &resp, false)
}

// GetSubAccAssetsInfo gets asset info for all subaccounts
func (h *HUOBI) GetSubAccAssetsInfo(code string, subUID int64) (SubAccountsAssetData, error) {
	var resp SubAccountsAssetData
	req := make(map[string]interface{})
	req["contract_code"] = code
	req["sub_uid"] = subUID
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapSubAccList, nil, req, &resp, false)
}

// GetSubAccPositionInfo gets a subaccount's positions info
func (h *HUOBI) GetSubAccPositionInfo(code string, subUID int64) (SingleSubAccountPositionsInfo, error) {
	var resp SingleSubAccountPositionsInfo
	req := make(map[string]interface{})
	req["contract_code"] = code
	req["sub_uid"] = subUID
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapSubAccList, nil, req, &resp, false)
}

// GetAccountFinancialRecords gets the account's financial records
func (h *HUOBI) GetAccountFinancialRecords(code, orderType string, createDate, pageIndex, pageSize int64) (FinancialRecordData, error) {
	var resp FinancialRecordData
	req := make(map[string]interface{})
	req["contract_code"] = code
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
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapFinancialRecords, nil, req, &resp, false)
}

// GetSwapSettlementRecords gets the swap account's settlement records
func (h *HUOBI) GetSwapSettlementRecords(code string, startTime, endTime time.Time, pageIndex, pageSize int64) (FinancialRecordData, error) {
	var resp FinancialRecordData
	req := make(map[string]interface{})
	req["contract_code"] = code
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
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapSettlementRecords, nil, req, &resp, false)
}

// GetAvailableLeverage gets user's available leverage data
func (h *HUOBI) GetAvailableLeverage(code string) (AvailableLeverageData, error) {
	var resp AvailableLeverageData
	req := make(map[string]interface{})
	if code != "" {
		req["contract_code"] = code
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapAvailableLeverage, nil, req, &resp, false)
}

// GetSwapOrderLimitInfo gets order limit info for swaps
func (h *HUOBI) GetSwapOrderLimitInfo(code, orderType string) (SwapOrderLimitInfo, error) {
	var resp SwapOrderLimitInfo
	req := make(map[string]interface{})
	req["contract_code"] = code
	if !common.StringDataCompare(validOrderTypes, orderType) {
		return resp, fmt.Errorf("inavlid ordertype provided")
	}
	req["order_price_type"] = orderType
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapOrderLimitInfo, nil, req, &resp, false)
}

// GetSwapTradingFeeInfo gets trading fee info for swaps
func (h *HUOBI) GetSwapTradingFeeInfo(code string) (SwapTradingFeeData, error) {
	var resp SwapTradingFeeData
	req := make(map[string]interface{})
	req["contract_code"] = code
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapTradingFeeInfo, nil, req, &resp, false)
}

// GetSwapTransferLimitInfo gets transfer limit info for swaps
func (h *HUOBI) GetSwapTransferLimitInfo(code string) (TransferLimitData, error) {
	var resp TransferLimitData
	req := make(map[string]interface{})
	req["contract_code"] = code
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapTransferLimitInfo, nil, req, &resp, false)
}

// GetSwapPositionLimitInfo gets transfer limit info for swaps
func (h *HUOBI) GetSwapPositionLimitInfo(code string) (PositionLimitData, error) {
	var resp PositionLimitData
	req := make(map[string]interface{})
	req["contract_code"] = code
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapPositionLimitInfo, nil, req, &resp, false)
}

// AccountTransferData gets asset transfer data between master and subaccounts
func (h *HUOBI) AccountTransferData(code, subUID, transferType string, amount float64) (InternalAccountTransferData, error) {
	var resp InternalAccountTransferData
	req := make(map[string]interface{})
	req["contract_code"] = code
	req["subUid"] = subUID
	req["amount"] = amount
	if !common.StringDataCompare(validTransferType, transferType) {
		return resp, fmt.Errorf("inavlid transferType received")
	}
	req["type"] = transferType
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapInternalTransferData, nil, req, &resp, false)
}

// AccountTransferRecords gets asset transfer records between master and subaccounts
func (h *HUOBI) AccountTransferRecords(code, transferType string, createDate, pageIndex, pageSize int64) (InternalAccountTransferData, error) {
	var resp InternalAccountTransferData
	req := make(map[string]interface{})
	req["contract_code"] = code
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
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapInternalTransferRecords, nil, req, &resp, false)
}

// PlaceSwapOrders places orders for swaps
func (h *HUOBI) PlaceSwapOrders(code, clientOrderID, direction, offset, orderPriceType string, price, volume, leverage float64) (SwapOrderData, error) {
	var resp SwapOrderData
	req := make(map[string]interface{})
	req["contract_code"] = code
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
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapPlaceOrder, nil, req, &resp, false)
}

// PlaceBatchOrders places a batch of orders for swaps
func (h *HUOBI) PlaceBatchOrders(data BatchOrderRequestType) (BatchOrderData, error) {
	var resp BatchOrderData
	req := make(map[string]interface{})
	if !((0 < len(data.Data)) && (len(data.Data) <= 10)) {
		return resp, fmt.Errorf("invalid data provided: maximum of 10 batch orders supported")
	}
	req["orders_data"] = data.Data
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapPlaceBatchOrder, nil, req, &resp, false)
}

// CancelSwapOrder sends a request to cancel an order
func (h *HUOBI) CancelSwapOrder(orderID, clientOrderID, contractCode string) (CancelOrdersData, error) {
	var resp CancelOrdersData
	req := make(map[string]interface{})
	if orderID != "" {
		req["order_id"] = orderID
	}
	if clientOrderID != "" {
		req["client_order_id"] = clientOrderID
	}
	req["contract_code"] = contractCode
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapCancelOrder, nil, req, &resp, false)
}

// CancelAllSwapOrders sends a request to cancel an order
func (h *HUOBI) CancelAllSwapOrders(contractCode string) (CancelOrdersData, error) {
	var resp CancelOrdersData
	req := make(map[string]interface{})
	req["contract_code"] = contractCode
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapCancelAllOrders, nil, req, &resp, false)
}

// PlaceLightningCloseOrder places a lightning close order
func (h *HUOBI) PlaceLightningCloseOrder(contractCode, direction, orderPriceType string, volume float64, clientOrderID int64) (LightningCloseOrderData, error) {
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
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapLightningCloseOrder, nil, req, &resp, false)
}

// GetSwapOrderDetails gets order info
func (h *HUOBI) GetSwapOrderDetails(contractCode, orderID, createdAt, orderType string, pageIndex, pageSize int64) (SwapOrderData, error) {
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
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapOrderDetails, nil, req, &resp, false)
}

// GetSwapOrderInfo gets info on a swap order
func (h *HUOBI) GetSwapOrderInfo(contractCode, orderID, clientOrderID string) (SwapOpenOrdersData, error) {
	var resp SwapOpenOrdersData
	req := make(map[string]interface{})
	if contractCode != "" {
		req["contract_code"] = contractCode
	}
	if orderID != "" {
		req["order_id"] = orderID
	}
	if clientOrderID != "" {
		req["client_order_id"] = clientOrderID
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapOrderInfo, nil, req, &resp, false)
}

// GetSwapOpenOrders gets open orders for swap
func (h *HUOBI) GetSwapOpenOrders(contractCode string, pageIndex, pageSize int64) (SwapOpenOrdersData, error) {
	var resp SwapOpenOrdersData
	req := make(map[string]interface{})
	req["contract_code"] = contractCode
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize > 0 && pageSize <= 50 {
		req["page_size"] = pageSize
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapOpenOrders, nil, req, &resp, false)
}

// GetSwapOrderHistory gets swap order history
func (h *HUOBI) GetSwapOrderHistory(contractCode string, pageIndex, pageSize int64) (SwapOrderHistory, error) {
	var resp SwapOrderHistory
	req := make(map[string]interface{})
	req["contract_code"] = contractCode
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize > 0 && pageSize <= 50 {
		req["page_size"] = pageSize
	}
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapOrderHistory, nil, req, &resp, false)
}

// GetSwapTradeHistory gets swap trade history
func (h *HUOBI) GetSwapTradeHistory(contractCode, tradeType string, createDate, pageIndex, pageSize int64) (AccountTradeHistoryData, error) {
	var resp AccountTradeHistoryData
	req := make(map[string]interface{})
	req["contract_code"] = contractCode
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
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapTradeHistory, nil, req, &resp, false)
}

// PlaceSwapTriggerOrder places a trigger order for a swap
func (h *HUOBI) PlaceSwapTriggerOrder(contractCode, triggerType, direction, offset, orderPriceType string, triggerPrice, orderPrice, volume, leverageRate float64) (AccountTradeHistoryData, error) {
	var resp AccountTradeHistoryData
	req := make(map[string]interface{})
	req["contract_code"] = contractCode
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
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapTriggerOrder, nil, req, &resp, false)
}

// CancelSwapTriggerOrder cancels swap trigger order
func (h *HUOBI) CancelSwapTriggerOrder(contractCode, orderID string) (CancelTriggerOrdersData, error) {
	var resp CancelTriggerOrdersData
	req := make(map[string]interface{})
	req["contract_code"] = contractCode
	req["order_id"] = orderID
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapCancelTriggerOrder, nil, req, &resp, false)
}

// CancelAllSwapTriggerOrders cancels all swap trigger orders
func (h *HUOBI) CancelAllSwapTriggerOrders(contractCode string) (CancelTriggerOrdersData, error) {
	var resp CancelTriggerOrdersData
	req := make(map[string]interface{})
	req["contract_code"] = contractCode
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapCancelAllTriggerOrders, nil, req, &resp, false)
}

// GetSwapTriggerOrderHistory gets history for swap trigger orders
func (h *HUOBI) GetSwapTriggerOrderHistory(contractCode, status, tradeType string, createDate, pageIndex, pageSize int64) (TriggerOrderHistory, error) {
	var resp TriggerOrderHistory
	req := make(map[string]interface{})
	req["contract_code"] = contractCode
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
	h.API.Endpoints.URL = huobiURL
	return resp, h.SendAuthenticatedHTTPRequest2(http.MethodPost, huobiSwapTriggerOrderHistory, nil, req, &resp, false)
}

// ************************************************************************

// GetSwapMarkets gets data of swap markets
func (h *HUOBI) GetSwapMarkets(contract string) ([]SwapMarketsData, error) {
	vals := url.Values{}
	vals.Set("contract_code", contract)
	type response struct {
		Response
		Data []SwapMarketsData `json:"data"`
	}
	var result response
	err := h.SendHTTPRequest(common.EncodeURLValues(huobiURL+huobiSwapMarkets, vals), &result)
	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Data, err
}

// GetSwapFundingRates gets funding rates data
func (h *HUOBI) GetSwapFundingRates(contract string) (FundingRatesData, error) {
	vals := url.Values{}
	vals.Set("contract_code", contract)
	type response struct {
		Response
		Data FundingRatesData `json:"data"`
	}
	var result response
	err := h.SendHTTPRequest(common.EncodeURLValues(huobiURL+huobiSwapFunding, vals), &result)
	if result.ErrorMessage != "" {
		return FundingRatesData{}, errors.New(result.ErrorMessage)
	}
	return result.Data, err
}

// SPOT section below ***************************************************************************************

// GetMarginRates gets margin rates
func (h *HUOBI) GetMarginRates(symbol string) (MarginRatesData, error) {
	vals := url.Values{}
	if symbol != "" {
		vals.Set("symbols", symbol)
	}
	var resp MarginRatesData
	return resp, h.SendAuthenticatedHTTPRequest(http.MethodGet, huobiMarginRates, vals, nil, &resp, false)
}

// GetSpotKline returns kline data
// KlinesRequestParams contains symbol, period and size
func (h *HUOBI) GetSpotKline(arg KlinesRequestParams) ([]KlineItem, error) {
	vals := url.Values{}
	vals.Set("symbol", arg.Symbol)
	vals.Set("period", arg.Period)

	if arg.Size != 0 {
		vals.Set("size", strconv.Itoa(arg.Size))
	}

	type response struct {
		Response
		Data []KlineItem `json:"data"`
	}

	var result response
	urlPath := fmt.Sprintf("%s/%s", h.API.Endpoints.URL, huobiMarketHistoryKline)

	err := h.SendHTTPRequest(common.EncodeURLValues(urlPath, vals), &result)
	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Data, err
}

// GetTickers returns the ticker for the specified symbol
func (h *HUOBI) GetTickers() (Tickers, error) {
	var result Tickers
	urlPath := fmt.Sprintf("%s/%s", h.API.Endpoints.URL, huobiMarketTickers)
	return result, h.SendHTTPRequest(urlPath, &result)
}

// GetMarketDetailMerged returns the ticker for the specified symbol
func (h *HUOBI) GetMarketDetailMerged(symbol string) (DetailMerged, error) {
	vals := url.Values{}
	vals.Set("symbol", symbol)

	type response struct {
		Response
		Tick DetailMerged `json:"tick"`
	}

	var result response
	urlPath := fmt.Sprintf("%s/%s", h.API.Endpoints.URL, huobiMarketDetailMerged)

	err := h.SendHTTPRequest(common.EncodeURLValues(urlPath, vals), &result)
	if result.ErrorMessage != "" {
		return result.Tick, errors.New(result.ErrorMessage)
	}
	return result.Tick, err
}

// GetDepth returns the depth for the specified symbol
func (h *HUOBI) GetDepth(obd OrderBookDataRequestParams) (Orderbook, error) {
	vals := url.Values{}
	vals.Set("symbol", obd.Symbol)

	if obd.Type != OrderBookDataRequestParamsTypeNone {
		vals.Set("type", string(obd.Type))
	}

	type response struct {
		Response
		Depth Orderbook `json:"tick"`
	}

	var result response
	urlPath := fmt.Sprintf("%s/%s", h.API.Endpoints.URL, huobiMarketDepth)

	err := h.SendHTTPRequest(common.EncodeURLValues(urlPath, vals), &result)
	if result.ErrorMessage != "" {
		return result.Depth, errors.New(result.ErrorMessage)
	}
	return result.Depth, err
}

// GetTrades returns the trades for the specified symbol
func (h *HUOBI) GetTrades(symbol string) ([]Trade, error) {
	vals := url.Values{}
	vals.Set("symbol", symbol)

	type response struct {
		Response
		Tick struct {
			Data []Trade `json:"data"`
		} `json:"tick"`
	}

	var result response
	urlPath := fmt.Sprintf("%s/%s", h.API.Endpoints.URL, huobiMarketTrade)

	err := h.SendHTTPRequest(common.EncodeURLValues(urlPath, vals), &result)
	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Tick.Data, err
}

// GetLatestSpotPrice returns latest spot price of symbol
//
// symbol: string of currency pair
func (h *HUOBI) GetLatestSpotPrice(symbol string) (float64, error) {
	list, err := h.GetTradeHistory(symbol, "1")

	if err != nil {
		return 0, err
	}
	if len(list) == 0 {
		return 0, errors.New("the length of the list is 0")
	}

	return list[0].Trades[0].Price, nil
}

// GetTradeHistory returns the trades for the specified symbol
func (h *HUOBI) GetTradeHistory(symbol, size string) ([]TradeHistory, error) {
	vals := url.Values{}
	vals.Set("symbol", symbol)

	if size != "" {
		vals.Set("size", size)
	}

	type response struct {
		Response
		TradeHistory []TradeHistory `json:"data"`
	}

	var result response
	urlPath := fmt.Sprintf("%s/%s", h.API.Endpoints.URL, huobiMarketTradeHistory)

	err := h.SendHTTPRequest(common.EncodeURLValues(urlPath, vals), &result)
	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.TradeHistory, err
}

// GetMarketDetail returns the ticker for the specified symbol
func (h *HUOBI) GetMarketDetail(symbol string) (Detail, error) {
	vals := url.Values{}
	vals.Set("symbol", symbol)

	type response struct {
		Response
		Tick Detail `json:"tick"`
	}

	var result response
	urlPath := fmt.Sprintf("%s/%s", h.API.Endpoints.URL, huobiMarketDetail)

	err := h.SendHTTPRequest(common.EncodeURLValues(urlPath, vals), &result)
	if result.ErrorMessage != "" {
		return result.Tick, errors.New(result.ErrorMessage)
	}
	return result.Tick, err
}

// GetSymbols returns an array of symbols supported by Huobi
func (h *HUOBI) GetSymbols() ([]Symbol, error) {
	type response struct {
		Response
		Symbols []Symbol `json:"data"`
	}

	var result response
	urlPath := fmt.Sprintf("%s/v%s/%s", h.API.Endpoints.URL, huobiAPIVersion, huobiSymbols)

	err := h.SendHTTPRequest(urlPath, &result)
	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Symbols, err
}

// GetCurrencies returns a list of currencies supported by Huobi
func (h *HUOBI) GetCurrencies() ([]string, error) {
	type response struct {
		Response
		Currencies []string `json:"data"`
	}

	var result response
	urlPath := fmt.Sprintf("%s/v%s/%s", h.API.Endpoints.URL, huobiAPIVersion, huobiCurrencies)

	err := h.SendHTTPRequest(urlPath, &result)
	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Currencies, err
}

// GetTimestamp returns the Huobi server time
func (h *HUOBI) GetTimestamp() (int64, error) {
	type response struct {
		Response
		Timestamp int64 `json:"data"`
	}

	var result response
	urlPath := fmt.Sprintf("%s/v%s/%s", h.API.Endpoints.URL, huobiAPIVersion, huobiTimestamp)

	err := h.SendHTTPRequest(urlPath, &result)
	if result.ErrorMessage != "" {
		return 0, errors.New(result.ErrorMessage)
	}
	return result.Timestamp, err
}

// GetAccounts returns the Huobi user accounts
func (h *HUOBI) GetAccounts() ([]Account, error) {
	result := struct {
		Accounts []Account `json:"data"`
	}{}
	err := h.SendAuthenticatedHTTPRequest(http.MethodGet, huobiAccounts, url.Values{}, nil, &result, false)
	return result.Accounts, err
}

// GetAccountBalance returns the users Huobi account balance
func (h *HUOBI) GetAccountBalance(accountID string) ([]AccountBalanceDetail, error) {
	result := struct {
		AccountBalanceData AccountBalance `json:"data"`
	}{}
	endpoint := fmt.Sprintf(huobiAccountBalance, accountID)
	v := url.Values{}
	v.Set("account-id", accountID)
	err := h.SendAuthenticatedHTTPRequest(http.MethodGet, endpoint, v, nil, &result, false)
	return result.AccountBalanceData.AccountBalanceDetails, err
}

// GetAggregatedBalance returns the balances of all the sub-account aggregated.
func (h *HUOBI) GetAggregatedBalance() ([]AggregatedBalance, error) {
	result := struct {
		AggregatedBalances []AggregatedBalance `json:"data"`
	}{}
	err := h.SendAuthenticatedHTTPRequest(
		http.MethodGet,
		huobiAggregatedBalance,
		nil,
		nil,
		&result,
		false,
	)
	return result.AggregatedBalances, err
}

// SpotNewOrder submits an order to Huobi
func (h *HUOBI) SpotNewOrder(arg SpotNewOrderRequestParams) (int64, error) {
	data := struct {
		AccountID int    `json:"account-id,string"`
		Amount    string `json:"amount"`
		Price     string `json:"price"`
		Source    string `json:"source"`
		Symbol    string `json:"symbol"`
		Type      string `json:"type"`
	}{
		AccountID: arg.AccountID,
		Amount:    strconv.FormatFloat(arg.Amount, 'f', -1, 64),
		Symbol:    arg.Symbol,
		Type:      string(arg.Type),
	}

	// Only set price if order type is not equal to buy-market or sell-market
	if arg.Type != SpotNewOrderRequestTypeBuyMarket && arg.Type != SpotNewOrderRequestTypeSellMarket {
		data.Price = strconv.FormatFloat(arg.Price, 'f', -1, 64)
	}

	if arg.Source != "" {
		data.Source = arg.Source
	}

	result := struct {
		OrderID int64 `json:"data,string"`
	}{}
	err := h.SendAuthenticatedHTTPRequest(
		http.MethodPost,
		huobiOrderPlace,
		nil,
		data,
		&result,
		false,
	)
	return result.OrderID, err
}

// CancelExistingOrder cancels an order on Huobi
func (h *HUOBI) CancelExistingOrder(orderID int64) (int64, error) {
	resp := struct {
		OrderID int64 `json:"data,string"`
	}{}
	endpoint := fmt.Sprintf(huobiOrderCancel, strconv.FormatInt(orderID, 10))
	err := h.SendAuthenticatedHTTPRequest(http.MethodPost, endpoint, url.Values{}, nil, &resp, false)
	return resp.OrderID, err
}

// CancelOrderBatch cancels a batch of orders -- to-do
func (h *HUOBI) CancelOrderBatch(_ []int64) ([]CancelOrderBatch, error) {
	type response struct {
		Response
		Data []CancelOrderBatch `json:"data"`
	}

	var result response
	err := h.SendAuthenticatedHTTPRequest(http.MethodPost, huobiOrderCancelBatch, url.Values{}, nil, &result, false)

	if result.ErrorMessage != "" {
		return nil, errors.New(result.ErrorMessage)
	}
	return result.Data, err
}

// CancelOpenOrdersBatch cancels a batch of orders -- to-do
func (h *HUOBI) CancelOpenOrdersBatch(accountID, symbol string) (CancelOpenOrdersBatch, error) {
	params := url.Values{}

	params.Set("account-id", accountID)
	var result CancelOpenOrdersBatch

	data := struct {
		AccountID string `json:"account-id"`
		Symbol    string `json:"symbol"`
	}{
		AccountID: accountID,
		Symbol:    symbol,
	}

	err := h.SendAuthenticatedHTTPRequest(http.MethodPost, huobiBatchCancelOpenOrders, url.Values{}, data, &result, false)
	if result.Data.FailedCount > 0 {
		return result, fmt.Errorf("there were %v failed order cancellations", result.Data.FailedCount)
	}

	return result, err
}

// GetOrder returns order information for the specified order
func (h *HUOBI) GetOrder(orderID int64) (OrderInfo, error) {
	resp := struct {
		Order OrderInfo `json:"data"`
	}{}
	urlVal := url.Values{}
	urlVal.Set("clientOrderId", strconv.FormatInt(orderID, 10))
	err := h.SendAuthenticatedHTTPRequest(http.MethodGet,
		huobiGetOrder,
		urlVal,
		nil,
		&resp,
		false)
	return resp.Order, err
}

// GetOrderMatchResults returns matched order info for the specified order
func (h *HUOBI) GetOrderMatchResults(orderID int64) ([]OrderMatchInfo, error) {
	resp := struct {
		Orders []OrderMatchInfo `json:"data"`
	}{}
	endpoint := fmt.Sprintf(huobiGetOrderMatch, strconv.FormatInt(orderID, 10))
	err := h.SendAuthenticatedHTTPRequest(http.MethodGet, endpoint, url.Values{}, nil, &resp, false)
	return resp.Orders, err
}

// GetOrders returns a list of orders
func (h *HUOBI) GetOrders(symbol, types, start, end, states, from, direct, size string) ([]OrderInfo, error) {
	resp := struct {
		Orders []OrderInfo `json:"data"`
	}{}

	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("states", states)

	if types != "" {
		vals.Set("types", types)
	}

	if start != "" {
		vals.Set("start-date", start)
	}

	if end != "" {
		vals.Set("end-date", end)
	}

	if from != "" {
		vals.Set("from", from)
	}

	if direct != "" {
		vals.Set("direct", direct)
	}

	if size != "" {
		vals.Set("size", size)
	}

	err := h.SendAuthenticatedHTTPRequest(http.MethodGet, huobiGetOrders, vals, nil, &resp, false)
	return resp.Orders, err
}

// GetOpenOrders returns a list of orders
func (h *HUOBI) GetOpenOrders(accountID, symbol, side string, size int64) ([]OrderInfo, error) {
	resp := struct {
		Orders []OrderInfo `json:"data"`
	}{}

	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("accountID", accountID)
	if len(side) > 0 {
		vals.Set("side", side)
	}
	vals.Set("size", strconv.FormatInt(size, 10))

	err := h.SendAuthenticatedHTTPRequest(http.MethodGet, huobiGetOpenOrders, vals, nil, &resp, false)
	return resp.Orders, err
}

// GetOrdersMatch returns a list of matched orders
func (h *HUOBI) GetOrdersMatch(symbol, types, start, end, from, direct, size string) ([]OrderMatchInfo, error) {
	resp := struct {
		Orders []OrderMatchInfo `json:"data"`
	}{}

	vals := url.Values{}
	vals.Set("symbol", symbol)

	if types != "" {
		vals.Set("types", types)
	}

	if start != "" {
		vals.Set("start-date", start)
	}

	if end != "" {
		vals.Set("end-date", end)
	}

	if from != "" {
		vals.Set("from", from)
	}

	if direct != "" {
		vals.Set("direct", direct)
	}

	if size != "" {
		vals.Set("size", size)
	}

	err := h.SendAuthenticatedHTTPRequest(http.MethodGet, huobiGetOrdersMatch, vals, nil, &resp, false)
	return resp.Orders, err
}

// MarginTransfer transfers assets into or out of the margin account
func (h *HUOBI) MarginTransfer(symbol, currency string, amount float64, in bool) (int64, error) {
	data := struct {
		Symbol   string `json:"symbol"`
		Currency string `json:"currency"`
		Amount   string `json:"amount"`
	}{
		Symbol:   symbol,
		Currency: currency,
		Amount:   strconv.FormatFloat(amount, 'f', -1, 64),
	}

	path := huobiMarginTransferIn
	if !in {
		path = huobiMarginTransferOut
	}

	resp := struct {
		TransferID int64 `json:"data"`
	}{}
	err := h.SendAuthenticatedHTTPRequest(http.MethodPost, path, nil, data, &resp, false)
	return resp.TransferID, err
}

// MarginOrder submits a margin order application
func (h *HUOBI) MarginOrder(symbol, currency string, amount float64) (int64, error) {
	data := struct {
		Symbol   string `json:"symbol"`
		Currency string `json:"currency"`
		Amount   string `json:"amount"`
	}{
		Symbol:   symbol,
		Currency: currency,
		Amount:   strconv.FormatFloat(amount, 'f', -1, 64),
	}

	resp := struct {
		MarginOrderID int64 `json:"data"`
	}{}
	err := h.SendAuthenticatedHTTPRequest(http.MethodPost, huobiMarginOrders, nil, data, &resp, false)
	return resp.MarginOrderID, err
}

// MarginRepayment repays a margin amount for a margin ID
func (h *HUOBI) MarginRepayment(orderID int64, amount float64) (int64, error) {
	data := struct {
		Amount string `json:"amount"`
	}{
		Amount: strconv.FormatFloat(amount, 'f', -1, 64),
	}

	resp := struct {
		MarginOrderID int64 `json:"data"`
	}{}

	endpoint := fmt.Sprintf(huobiMarginRepay, strconv.FormatInt(orderID, 10))
	err := h.SendAuthenticatedHTTPRequest(http.MethodPost, endpoint, nil, data, &resp, false)
	return resp.MarginOrderID, err
}

// GetMarginLoanOrders returns the margin loan orders
func (h *HUOBI) GetMarginLoanOrders(symbol, currency, start, end, states, from, direct, size string) ([]MarginOrder, error) {
	vals := url.Values{}
	vals.Set("symbol", symbol)
	vals.Set("currency", currency)

	if start != "" {
		vals.Set("start-date", start)
	}

	if end != "" {
		vals.Set("end-date", end)
	}

	if states != "" {
		vals.Set("states", states)
	}

	if from != "" {
		vals.Set("from", from)
	}

	if direct != "" {
		vals.Set("direct", direct)
	}

	if size != "" {
		vals.Set("size", size)
	}

	resp := struct {
		MarginLoanOrders []MarginOrder `json:"data"`
	}{}
	err := h.SendAuthenticatedHTTPRequest(http.MethodGet, huobiMarginLoanOrders, vals, nil, &resp, false)
	return resp.MarginLoanOrders, err
}

// GetMarginAccountBalance returns the margin account balances
func (h *HUOBI) GetMarginAccountBalance(symbol string) ([]MarginAccountBalance, error) {
	resp := struct {
		Balances []MarginAccountBalance `json:"data"`
	}{}
	vals := url.Values{}
	if symbol != "" {
		vals.Set("symbol", symbol)
	}
	err := h.SendAuthenticatedHTTPRequest(http.MethodGet, huobiMarginAccountBalance, vals, nil, &resp, false)
	return resp.Balances, err
}

// Withdraw withdraws the desired amount and currency
func (h *HUOBI) Withdraw(c currency.Code, address, addrTag string, amount, fee float64) (int64, error) {
	resp := struct {
		WithdrawID int64 `json:"data"`
	}{}

	data := struct {
		Address  string `json:"address"`
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
		Fee      string `json:"fee,omitempty"`
		AddrTag  string `json:"addr-tag,omitempty"`
	}{
		Address:  address,
		Currency: c.Lower().String(),
		Amount:   strconv.FormatFloat(amount, 'f', -1, 64),
	}

	if fee > 0 {
		data.Fee = strconv.FormatFloat(fee, 'f', -1, 64)
	}

	if c == currency.XRP {
		data.AddrTag = addrTag
	}

	err := h.SendAuthenticatedHTTPRequest(http.MethodPost, huobiWithdrawCreate, nil, data, &resp.WithdrawID, false)
	return resp.WithdrawID, err
}

// CancelWithdraw cancels a withdraw request
func (h *HUOBI) CancelWithdraw(withdrawID int64) (int64, error) {
	resp := struct {
		WithdrawID int64 `json:"data"`
	}{}
	vals := url.Values{}
	vals.Set("withdraw-id", strconv.FormatInt(withdrawID, 10))

	endpoint := fmt.Sprintf(huobiWithdrawCancel, strconv.FormatInt(withdrawID, 10))
	err := h.SendAuthenticatedHTTPRequest(http.MethodPost, endpoint, vals, nil, &resp, false)
	return resp.WithdrawID, err
}

// QueryDepositAddress returns the deposit address for a specified currency
func (h *HUOBI) QueryDepositAddress(cryptocurrency string) (DepositAddress, error) {
	resp := struct {
		DepositAddress []DepositAddress `json:"data"`
	}{}

	vals := url.Values{}
	vals.Set("currency", cryptocurrency)

	err := h.SendAuthenticatedHTTPRequest(http.MethodGet, huobiAccountDepositAddress, vals, nil, &resp, true)
	if err != nil {
		return DepositAddress{}, err
	}
	if len(resp.DepositAddress) == 0 {
		return DepositAddress{}, errors.New("deposit address data isn't populated")
	}
	return resp.DepositAddress[0], nil
}

// QueryWithdrawQuotas returns the users cryptocurrency withdraw quotas
func (h *HUOBI) QueryWithdrawQuotas(cryptocurrency string) (WithdrawQuota, error) {
	resp := struct {
		WithdrawQuota WithdrawQuota `json:"data"`
	}{}

	vals := url.Values{}
	vals.Set("currency", cryptocurrency)

	err := h.SendAuthenticatedHTTPRequest(http.MethodGet, huobiAccountWithdrawQuota, vals, nil, &resp, true)
	if err != nil {
		return WithdrawQuota{}, err
	}
	return resp.WithdrawQuota, nil
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (h *HUOBI) SendHTTPRequest(path string, result interface{}) error {
	return h.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodGet,
		Path:          path,
		Result:        result,
		Verbose:       h.Verbose,
		HTTPDebugging: h.HTTPDebugging,
		HTTPRecording: h.HTTPRecording,
	})
}

// SendAuthenticatedHTTPRequest2 sends authenticated requests to the HUOBI API
func (h *HUOBI) SendAuthenticatedHTTPRequest2(method, endpoint string, values url.Values, data, result interface{}, isVersion2API bool) error {
	if !h.AllowAuthenticatedRequest() {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, h.Name)
	}
	if values == nil {
		values = url.Values{}
	}
	now := time.Now()
	values.Set("AccessKeyId", h.API.Credentials.Key)
	values.Set("SignatureMethod", "HmacSHA256")
	values.Set("SignatureVersion", "2")
	values.Set("Timestamp", now.UTC().Format("2006-01-02T15:04:05"))
	sigPath := fmt.Sprintf("%s\napi.hbdm.com\n/%s\n%s",
		method, endpoint, values.Encode())
	headers := make(map[string]string)
	if method == http.MethodGet {
		headers["Content-Type"] = "application/x-www-form-urlencoded"
	} else {
		headers["Content-Type"] = "application/json"
	}
	hmac := crypto.GetHMAC(crypto.HashSHA256, []byte(sigPath), []byte(h.API.Credentials.Secret))
	sigValues := url.Values{}
	sigValues.Add("Signature", crypto.Base64Encode(hmac))
	urlPath := h.API.Endpoints.URL +
		common.EncodeURLValues(endpoint, values) + "&" + sigValues.Encode()

	var body io.Reader
	var payload []byte
	var err error
	if data != nil {
		payload, err = json.Marshal(data)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(payload)
	}

	var tempResp json.RawMessage
	errCap := struct {
		Status string `json:"status"`
		Code   int64  `json:"err_code"`
		ErrMsg string `json:"err_msg"`
	}{}

	ctx, cancel := context.WithDeadline(context.Background(), now.Add(15*time.Second))
	defer cancel()
	if err := h.SendPayload(ctx, &request.Item{
		Method:        method,
		Path:          urlPath,
		Headers:       headers,
		Body:          body,
		Result:        &tempResp,
		AuthRequest:   true,
		Verbose:       h.Verbose,
		HTTPDebugging: h.HTTPDebugging,
		HTTPRecording: h.HTTPRecording,
	}); err != nil {
		return err
	}

	if err := json.Unmarshal(tempResp, &errCap); err == nil {
		if errCap.Code != 200 && errCap.ErrMsg != "" {
			return errors.New(errCap.ErrMsg)
		}
	}
	return json.Unmarshal(tempResp, result)
}

// SendAuthenticatedHTTPRequest sends authenticated requests to the HUOBI API
func (h *HUOBI) SendAuthenticatedHTTPRequest(method, endpoint string, values url.Values, data, result interface{}, isVersion2API bool) error {
	if !h.AllowAuthenticatedRequest() {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, h.Name)
	}

	if values == nil {
		values = url.Values{}
	}

	now := time.Now()
	values.Set("AccessKeyId", h.API.Credentials.Key)
	values.Set("SignatureMethod", "HmacSHA256")
	values.Set("SignatureVersion", "2")
	values.Set("Timestamp", now.UTC().Format("2006-01-02T15:04:05"))

	if isVersion2API {
		endpoint = fmt.Sprintf("/v%s/%s", huobiAPIVersion2, endpoint)
	} else {
		endpoint = fmt.Sprintf("/v%s/%s", huobiAPIVersion, endpoint)
	}

	payload := fmt.Sprintf("%s\napi.huobi.pro\n%s\n%s",
		method, endpoint, values.Encode())

	headers := make(map[string]string)

	if method == http.MethodGet {
		headers["Content-Type"] = "application/x-www-form-urlencoded"
	} else {
		headers["Content-Type"] = "application/json"
	}

	hmac := crypto.GetHMAC(crypto.HashSHA256, []byte(payload), []byte(h.API.Credentials.Secret))
	values.Set("Signature", crypto.Base64Encode(hmac))
	urlPath := h.API.Endpoints.URL + common.EncodeURLValues(endpoint, values)

	var body []byte
	if data != nil {
		encoded, err := json.Marshal(data)
		if err != nil {
			return err
		}
		body = encoded
	}

	// Time difference between your timestamp and standard should be less than 1 minute.
	ctx, cancel := context.WithDeadline(context.Background(), now.Add(time.Minute))
	defer cancel()
	interim := json.RawMessage{}
	err := h.SendPayload(ctx, &request.Item{
		Method:        method,
		Path:          urlPath,
		Headers:       headers,
		Body:          bytes.NewReader(body),
		Result:        &interim,
		AuthRequest:   true,
		Verbose:       h.Verbose,
		HTTPDebugging: h.HTTPDebugging,
		HTTPRecording: h.HTTPRecording,
	})
	if err != nil {
		return err
	}

	if isVersion2API {
		var errCap ResponseV2
		if err = json.Unmarshal(interim, &errCap); err == nil {
			if errCap.Code != 200 && errCap.Message != "" {
				return errors.New(errCap.Message)
			}
		}
	} else {
		var errCap Response
		if err = json.Unmarshal(interim, &errCap); err == nil {
			if errCap.Status == huobiStatusError && errCap.ErrorMessage != "" {
				return errors.New(errCap.ErrorMessage)
			}
		}
	}
	return json.Unmarshal(interim, result)
}

// GetFee returns an estimate of fee based on type of transaction
func (h *HUOBI) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	if feeBuilder.FeeType == exchange.OfflineTradeFee || feeBuilder.FeeType == exchange.CryptocurrencyTradeFee {
		fee = calculateTradingFee(feeBuilder.Pair, feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	if fee < 0 {
		fee = 0
	}

	return fee, nil
}

func calculateTradingFee(c currency.Pair, price, amount float64) float64 {
	if c.IsCryptoFiatPair() {
		return 0.001 * price * amount
	}
	return 0.002 * price * amount
}
