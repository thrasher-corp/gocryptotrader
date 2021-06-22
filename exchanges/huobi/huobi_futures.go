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
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// Unauth
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
	fSystemStatus              = "api/v1/contract_api_state?"
	fTopAccountsSentiment      = "api/v1/contract_elite_account_ratio?"
	fTopPositionsSentiment     = "api/v1/contract_elite_position_ratio?"
	fLiquidationOrders         = "api/v1/contract_liquidation_orders?"
	fIndexKline                = "/index/market/history/index?"
	fBasisData                 = "/index/market/history/basis?"

	// Auth
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
	fBatchOrder                = "api/v1/contract_batchorder"
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
)

// FGetContractInfo gets contract info for futures
func (h *HUOBI) FGetContractInfo(symbol, contractType string, code currency.Pair) (FContractInfoData, error) {
	var resp FContractInfoData
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if contractType != "" {
		if !common.StringDataCompare(validContractTypes, contractType) {
			return resp, fmt.Errorf("invalid contractType")
		}
		params.Set("contract_type", contractType)
	}
	if code != (currency.Pair{}) {
		codeValue, err := h.FormatSymbol(code, asset.Futures)
		if err != nil {
			return resp, err
		}
		params.Set("contract_code", codeValue)
	}
	path := fContractInfo + params.Encode()
	return resp, h.SendHTTPRequest(exchange.RestFutures, path, &resp)
}

// FIndexPriceInfo gets index price info for a futures contract
func (h *HUOBI) FIndexPriceInfo(symbol currency.Code) (FContractIndexPriceInfo, error) {
	var resp FContractIndexPriceInfo
	params := url.Values{}
	if symbol != (currency.Code{}) {
		codeValue, err := h.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", codeValue)
	}
	path := fContractIndexPrice + params.Encode()
	return resp, h.SendHTTPRequest(exchange.RestFutures, path, &resp)
}

// FContractPriceLimitations gets price limits for a futures contract
func (h *HUOBI) FContractPriceLimitations(symbol, contractType string, code currency.Pair) (FContractIndexPriceInfo, error) {
	var resp FContractIndexPriceInfo
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if contractType != "" {
		if !common.StringDataCompare(validContractTypes, contractType) {
			return resp, fmt.Errorf("invalid contractType: %s", contractType)
		}
		params.Set("contract_type", contractType)
	}
	if code != (currency.Pair{}) {
		codeValue, err := h.FormatSymbol(code, asset.Futures)
		if err != nil {
			return resp, err
		}
		params.Set("contract_code", codeValue)
	}
	path := fContractPriceLimitation + params.Encode()
	return resp, h.SendHTTPRequest(exchange.RestFutures, path, &resp)
}

// FContractOpenInterest gets open interest data for futures contracts
func (h *HUOBI) FContractOpenInterest(symbol, contractType string, code currency.Pair) (FContractOIData, error) {
	var resp FContractOIData
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if contractType != "" {
		if !common.StringDataCompare(validContractTypes, contractType) {
			return resp, fmt.Errorf("invalid contractType")
		}
		params.Set("contract_type", contractType)
	}
	if code != (currency.Pair{}) {
		codeValue, err := h.FormatSymbol(code, asset.Futures)
		if err != nil {
			return resp, err
		}
		params.Set("contract_code", codeValue)
	}
	path := fContractOpenInterest + params.Encode()
	return resp, h.SendHTTPRequest(exchange.RestFutures, path, &resp)
}

// FGetEstimatedDeliveryPrice gets estimated delivery price info for futures
func (h *HUOBI) FGetEstimatedDeliveryPrice(symbol currency.Code) (FEstimatedDeliveryPriceInfo, error) {
	var resp FEstimatedDeliveryPriceInfo
	params := url.Values{}
	codeValue, err := h.formatFuturesCode(symbol)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", codeValue)
	path := fEstimatedDeliveryPrice + params.Encode()
	return resp, h.SendHTTPRequest(exchange.RestFutures, path, &resp)
}

// FGetMarketDepth gets market depth data for futures contracts
func (h *HUOBI) FGetMarketDepth(symbol currency.Pair, dataType string) (OBData, error) {
	var resp OBData
	var tempData FMarketDepth
	params := url.Values{}
	symbolValue, err := h.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	params.Set("type", dataType)
	path := fContractMarketDepth + params.Encode()
	err = h.SendHTTPRequest(exchange.RestFutures, path, &tempData)
	if err != nil {
		return resp, err
	}
	resp.Symbol = symbolValue
	for x := range tempData.Tick.Asks {
		resp.Asks = append(resp.Asks, obItem{
			Price:    tempData.Tick.Asks[x][0],
			Quantity: tempData.Tick.Asks[x][1],
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
func (h *HUOBI) FGetKlineData(symbol currency.Pair, period string, size int64, startTime, endTime time.Time) (FKlineData, error) {
	var resp FKlineData
	params := url.Values{}
	symbolValue, err := h.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if !common.StringDataCompare(validFuturesPeriods, period) {
		return resp, fmt.Errorf("invalid period value received")
	}
	params.Set("period", period)
	if size <= 0 || size > 1200 {
		return resp, fmt.Errorf("invalid size provided values from 1-1200 supported")
	}
	params.Set("size", strconv.FormatInt(size, 10))
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	path := fContractKline + params.Encode()
	return resp, h.SendHTTPRequest(exchange.RestFutures, path, &resp)
}

// FGetMarketOverviewData gets market overview data for futures
func (h *HUOBI) FGetMarketOverviewData(symbol currency.Pair) (FMarketOverviewData, error) {
	var resp FMarketOverviewData
	params := url.Values{}
	symbolValue, err := h.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	path := fMarketOverview + params.Encode()
	return resp, h.SendHTTPRequest(exchange.RestFutures, path, &resp)
}

// FLastTradeData gets last trade data for a futures contract
func (h *HUOBI) FLastTradeData(symbol currency.Pair) (FLastTradeData, error) {
	var resp FLastTradeData
	params := url.Values{}
	symbolValue, err := h.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	path := fLastTradeContract + params.Encode()
	return resp, h.SendHTTPRequest(exchange.RestFutures, path, &resp)
}

// FRequestPublicBatchTrades gets public batch trades for a futures contract
func (h *HUOBI) FRequestPublicBatchTrades(symbol currency.Pair, size int64) (FBatchTradesForContractData, error) {
	var resp FBatchTradesForContractData
	params := url.Values{}
	symbolValue, err := h.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if size > 1 && size < 2000 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	path := fContractBatchTradeRecords + params.Encode()
	return resp, h.SendHTTPRequest(exchange.RestFutures, path, &resp)
}

// FQueryInsuranceAndClawbackData gets insurance and clawback data for a futures contract
func (h *HUOBI) FQueryInsuranceAndClawbackData(symbol currency.Code) (FClawbackRateAndInsuranceData, error) {
	var resp FClawbackRateAndInsuranceData
	params := url.Values{}
	if symbol != (currency.Code{}) {
		codeValue, err := h.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", codeValue)
	}
	path := fInsuranceAndClawback + params.Encode()
	return resp, h.SendHTTPRequest(exchange.RestFutures, path, &resp)
}

// FQueryHistoricalInsuranceData gets insurance data
func (h *HUOBI) FQueryHistoricalInsuranceData(symbol currency.Code) (FHistoricalInsuranceRecordsData, error) {
	var resp FHistoricalInsuranceRecordsData
	params := url.Values{}
	if symbol != (currency.Code{}) {
		codeValue, err := h.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", codeValue)
	}
	path := fInsuranceBalanceHistory + params.Encode()
	return resp, h.SendHTTPRequest(exchange.RestFutures, path, &resp)
}

// FQueryTieredAdjustmentFactor gets tiered adjustment factor for futures contracts
func (h *HUOBI) FQueryTieredAdjustmentFactor(symbol currency.Code) (FTieredAdjustmentFactorInfo, error) {
	var resp FTieredAdjustmentFactorInfo
	params := url.Values{}
	if symbol != (currency.Code{}) {
		codeValue, err := h.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", codeValue)
	}
	path := fTieredAdjustmentFactor + params.Encode()
	return resp, h.SendHTTPRequest(exchange.RestFutures, path, &resp)
}

// FQueryHisOpenInterest gets open interest for futures contract
func (h *HUOBI) FQueryHisOpenInterest(symbol, contractType, period, amountType string, size int64) (FOIData, error) {
	var resp FOIData
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if !common.StringDataCompare(validContractTypes, contractType) {
		return resp, fmt.Errorf("invalid contract type")
	}
	params.Set("contract_type", contractType)
	if !common.StringDataCompare(validPeriods, period) {
		return resp, fmt.Errorf("invalid period")
	}
	params.Set("period", period)
	if size > 0 || size <= 200 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	validAmount, ok := validAmountType[amountType]
	if !ok {
		return resp, fmt.Errorf("invalid amountType")
	}
	params.Set("amount_type", strconv.FormatInt(validAmount, 10))
	path := fHisContractOpenInterest + params.Encode()
	return resp, h.SendHTTPRequest(exchange.RestFutures, path, &resp)
}

// FQuerySystemStatus gets system status data
func (h *HUOBI) FQuerySystemStatus(symbol currency.Code) (FContractOIData, error) {
	var resp FContractOIData
	params := url.Values{}
	if symbol != (currency.Code{}) {
		codeValue, err := h.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", codeValue)
	}
	path := fSystemStatus + params.Encode()
	return resp, h.SendHTTPRequest(exchange.RestFutures, path, &resp)
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
	path := fTopAccountsSentiment + params.Encode()
	return resp, h.SendHTTPRequest(exchange.RestFutures, path, &resp)
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
	path := fTopPositionsSentiment + params.Encode()
	return resp, h.SendHTTPRequest(exchange.RestFutures, path, &resp)
}

// FLiquidationOrders gets liquidation orders for futures contracts
func (h *HUOBI) FLiquidationOrders(symbol, tradeType string, pageIndex, pageSize, createDate int64) (FLiquidationOrdersInfo, error) {
	var resp FLiquidationOrdersInfo
	params := url.Values{}
	params.Set("symbol", symbol)
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
	path := fLiquidationOrders + params.Encode()
	return resp, h.SendHTTPRequest(exchange.RestFutures, path, &resp)
}

// FIndexKline gets index kline data for futures contracts
func (h *HUOBI) FIndexKline(symbol currency.Pair, period string, size int64) (FIndexKlineData, error) {
	var resp FIndexKlineData
	params := url.Values{}
	symbolValue, err := h.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if !common.StringDataCompare(validFuturesPeriods, period) {
		return resp, fmt.Errorf("invalid period value received")
	}
	params.Set("period", period)
	if size <= 0 || size > 2000 {
		return resp, fmt.Errorf("invalid size")
	}
	params.Set("size", strconv.FormatInt(size, 10))
	path := fIndexKline + params.Encode()
	return resp, h.SendHTTPRequest(exchange.RestFutures, path, &resp)
}

// FGetBasisData gets basis data futures contracts
func (h *HUOBI) FGetBasisData(symbol currency.Pair, period, basisPriceType string, size int64) (FBasisData, error) {
	var resp FBasisData
	params := url.Values{}
	symbolValue, err := h.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if !common.StringDataCompare(validFuturesPeriods, period) {
		return resp, fmt.Errorf("invalid period value received")
	}
	params.Set("period", period)
	if basisPriceType != "" {
		if common.StringDataCompare(validBasisPriceTypes, basisPriceType) {
			params.Set("basis_price_type", basisPriceType)
		}
	}
	if size > 0 && size <= 2000 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	path := fBasisData + params.Encode()
	return resp, h.SendHTTPRequest(exchange.RestFutures, path, &resp)
}

// FGetAccountInfo gets user info for futures account
func (h *HUOBI) FGetAccountInfo(symbol currency.Code) (FUserAccountData, error) {
	var resp FUserAccountData
	req := make(map[string]interface{})
	if symbol != (currency.Code{}) {
		codeValue, err := h.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		req["symbol"] = codeValue
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fAccountData, nil, req, &resp)
}

// FGetPositionsInfo gets positions info for futures account
func (h *HUOBI) FGetPositionsInfo(symbol currency.Code) (FUserAccountData, error) {
	var resp FUserAccountData
	req := make(map[string]interface{})
	if symbol != (currency.Code{}) {
		codeValue, err := h.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		req["symbol"] = codeValue
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fPositionInformation, nil, req, &resp)
}

// FGetAllSubAccountAssets gets assets info for all futures subaccounts
func (h *HUOBI) FGetAllSubAccountAssets(symbol currency.Code) (FSubAccountAssetsInfo, error) {
	var resp FSubAccountAssetsInfo
	req := make(map[string]interface{})
	if symbol != (currency.Code{}) {
		codeValue, err := h.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		req["symbol"] = codeValue
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fAllSubAccountAssets, nil, req, &resp)
}

// FGetSingleSubAccountInfo gets assets info for a futures subaccount
func (h *HUOBI) FGetSingleSubAccountInfo(symbol, subUID string) (FSingleSubAccountAssetsInfo, error) {
	var resp FSingleSubAccountAssetsInfo
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	req["sub_uid"] = subUID
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fSingleSubAccountAssets, nil, req, &resp)
}

// FGetSingleSubPositions gets positions info for a single sub account
func (h *HUOBI) FGetSingleSubPositions(symbol, subUID string) (FSingleSubAccountPositionsInfo, error) {
	var resp FSingleSubAccountPositionsInfo
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	req["sub_uid"] = subUID
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fSingleSubAccountPositions, nil, req, &resp)
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
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fFinancialRecords, nil, req, &resp)
}

// FGetSettlementRecords gets settlement records for futures
func (h *HUOBI) FGetSettlementRecords(symbol currency.Code, pageIndex, pageSize int64, startTime, endTime time.Time) (FSettlementRecords, error) {
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
		req["start_time"] = strconv.FormatInt(startTime.Unix()*1000, 10)
		req["end_time"] = strconv.FormatInt(endTime.Unix()*1000, 10)
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fSettlementRecords, nil, req, &resp)
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
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fOrderLimitInfo, nil, req, &resp)
}

// FContractTradingFee gets futures contract trading fees
func (h *HUOBI) FContractTradingFee(symbol currency.Code) (FContractTradingFeeData, error) {
	var resp FContractTradingFeeData
	req := make(map[string]interface{})
	if symbol != (currency.Code{}) {
		codeValue, err := h.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		req["symbol"] = codeValue
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fContractTradingFee, nil, req, &resp)
}

// FGetTransferLimits gets transfer limits for futures
func (h *HUOBI) FGetTransferLimits(symbol currency.Code) (FTransferLimitData, error) {
	var resp FTransferLimitData
	req := make(map[string]interface{})
	if symbol != (currency.Code{}) {
		codeValue, err := h.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		req["symbol"] = codeValue
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fTransferLimitInfo, nil, req, &resp)
}

// FGetPositionLimits gets position limits for futures
func (h *HUOBI) FGetPositionLimits(symbol currency.Code) (FPositionLimitData, error) {
	var resp FPositionLimitData
	req := make(map[string]interface{})
	if symbol != (currency.Code{}) {
		codeValue, err := h.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		req["symbol"] = codeValue
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fPositionLimitInfo, nil, req, &resp)
}

// FGetAssetsAndPositions gets assets and positions for futures
func (h *HUOBI) FGetAssetsAndPositions(symbol currency.Code) (FAssetsAndPositionsData, error) {
	var resp FAssetsAndPositionsData
	req := make(map[string]interface{})
	req["symbol"] = symbol
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fQueryAssetsAndPositions, nil, req, &resp)
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
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fTransfer, nil, req, &resp)
}

// FGetTransferRecords gets transfer records data for futures
func (h *HUOBI) FGetTransferRecords(symbol, transferType string, createDate, pageIndex, pageSize int64) (FTransferRecords, error) {
	var resp FTransferRecords
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	if !common.StringDataCompare(validTransferType, transferType) {
		return resp, fmt.Errorf("inavlid transferType received")
	}
	req["type"] = transferType
	if createDate < 0 || createDate > 90 {
		return resp, fmt.Errorf("invalid create date value: only supports up to 90 days")
	}
	req["create_date"] = strconv.FormatInt(createDate, 10)
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize > 0 && pageSize <= 50 {
		req["page_size"] = pageSize
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fTransferRecords, nil, req, &resp)
}

// FGetAvailableLeverage gets available leverage data for futures
func (h *HUOBI) FGetAvailableLeverage(symbol currency.Code) (FAvailableLeverageData, error) {
	var resp FAvailableLeverageData
	req := make(map[string]interface{})
	if symbol != (currency.Code{}) {
		codeValue, err := h.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		req["symbol"] = codeValue
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fAvailableLeverage, nil, req, &resp)
}

// FOrder places an order for futures
func (h *HUOBI) FOrder(contractCode currency.Pair, symbol, contractType, clientOrderID, direction, offset, orderPriceType string, price, volume, leverageRate float64) (FOrderData, error) {
	var resp FOrderData
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	if contractType != "" {
		if !common.StringDataCompare(validContractTypes, contractType) {
			return resp, fmt.Errorf("invalid contractType")
		}
		req["contract_type"] = contractType
	}
	if contractCode != (currency.Pair{}) {
		codeValue, err := h.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	if clientOrderID != "" {
		req["client_order_id"] = clientOrderID
	}
	req["direction"] = direction
	if !common.StringDataCompare(validOffsetTypes, offset) {
		return resp, fmt.Errorf("invalid offset amounts")
	}
	if !common.StringDataCompare(validFuturesOrderPriceTypes, orderPriceType) {
		return resp, fmt.Errorf("invalid orderPriceType")
	}
	req["order_price_type"] = orderPriceType
	req["lever_rate"] = leverageRate
	req["volume"] = volume
	req["price"] = price
	req["offset"] = offset
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fOrder, nil, req, &resp)
}

// FPlaceBatchOrder places a batch of orders for futures
func (h *HUOBI) FPlaceBatchOrder(data []fBatchOrderData) (FBatchOrderResponse, error) {
	var resp FBatchOrderResponse
	req := make(map[string]interface{})
	if len(data) > 10 || len(data) == 0 {
		return resp, fmt.Errorf("invalid data provided: maximum of 10 batch orders supported")
	}
	for x := range data {
		if data[x].ContractCode != "" {
			unformattedPair, err := currency.NewPairFromString(data[x].ContractCode)
			if err != nil {
				return resp, err
			}
			formattedPair, err := h.FormatExchangeCurrency(unformattedPair, asset.Futures)
			if err != nil {
				return resp, err
			}
			data[x].ContractCode = formattedPair.String()
		}
		if data[x].ContractType != "" {
			if !common.StringDataCompare(validContractTypes, data[x].ContractType) {
				return resp, fmt.Errorf("invalid contractType")
			}
		}
		if !common.StringDataCompare(validOffsetTypes, data[x].Offset) {
			return resp, fmt.Errorf("invalid offset amounts")
		}
		if !common.StringDataCompare(validFuturesOrderPriceTypes, data[x].OrderPriceType) {
			return resp, fmt.Errorf("invalid orderPriceType")
		}
	}
	req["orders_data"] = data
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fBatchOrder, nil, req, &resp)
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
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fCancelOrder, nil, req, &resp)
}

// FCancelAllOrders cancels all futures order for a given symbol
func (h *HUOBI) FCancelAllOrders(contractCode currency.Pair, symbol, contractType string) (FCancelOrderData, error) {
	var resp FCancelOrderData
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	if contractType != "" {
		if !common.StringDataCompare(validContractTypes, contractType) {
			return resp, fmt.Errorf("invalid contractType")
		}
		req["contract_type"] = contractType
	}
	if contractCode != (currency.Pair{}) {
		codeValue, err := h.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fCancelAllOrders, nil, req, &resp)
}

// FFlashCloseOrder flash closes a futures order
func (h *HUOBI) FFlashCloseOrder(contractCode currency.Pair, symbol, contractType, direction, orderPriceType, clientOrderID string, volume float64) (FOrderData, error) {
	var resp FOrderData
	req := make(map[string]interface{})
	req["symbol"] = symbol
	if contractType != "" {
		if !common.StringDataCompare(validContractTypes, contractType) {
			return resp, fmt.Errorf("invalid contractType")
		}
		req["contract_type"] = contractType
	}
	if contractCode != (currency.Pair{}) {
		codeValue, err := h.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
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
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fFlashCloseOrder, nil, req, &resp)
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
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fOrderInfo, nil, req, &resp)
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
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fOrderDetails, nil, req, &resp)
}

// FGetOpenOrders gets order details for futures orders
func (h *HUOBI) FGetOpenOrders(symbol currency.Code, pageIndex, pageSize int64) (FOpenOrdersData, error) {
	var resp FOpenOrdersData
	req := make(map[string]interface{})
	req["symbol"] = symbol
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fQueryOpenOrders, nil, req, &resp)
}

// FGetOrderHistory gets order order history for futures
func (h *HUOBI) FGetOrderHistory(contractCode currency.Pair, symbol, tradeType, reqType, orderType string, status []order.Status, createDate, pageIndex, pageSize int64) (FOrderHistoryData, error) {
	var resp FOrderHistoryData
	req := make(map[string]interface{})
	req["symbol"] = symbol
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
	var reqStatus string = "0"
	if len(status) > 0 {
		var firstTime bool = true
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
	if contractCode != (currency.Pair{}) {
		codeValue, err := h.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
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
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fOrderHistory, nil, req, &resp)
}

// FTradeHistory gets trade history data for futures
func (h *HUOBI) FTradeHistory(contractCode currency.Pair, symbol, tradeType string, createDate, pageIndex, pageSize int64) (FOrderHistoryData, error) {
	var resp FOrderHistoryData
	req := make(map[string]interface{})
	req["symbol"] = symbol
	tType, ok := validTradeType[tradeType]
	if !ok {
		return resp, fmt.Errorf("invalid tradeType")
	}
	req["trade_type"] = tType
	if contractCode != (currency.Pair{}) {
		codeValue, err := h.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
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
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fMatchResult, nil, req, &resp)
}

// FPlaceTriggerOrder places a trigger order for futures
func (h *HUOBI) FPlaceTriggerOrder(contractCode currency.Pair, symbol, contractType, triggerType, orderPriceType, direction, offset string, triggerPrice, orderPrice, volume, leverageRate float64) (FTriggerOrderData, error) {
	var resp FTriggerOrderData
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	if contractType != "" {
		if !common.StringDataCompare(validContractTypes, contractType) {
			return resp, fmt.Errorf("invalid contractType: %s", contractType)
		}
		req["contract_type"] = contractType
	}
	if contractCode != (currency.Pair{}) {
		codeValue, err := h.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
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
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fTriggerOrder, nil, req, &resp)
}

// FCancelTriggerOrder cancels trigger order for futures
func (h *HUOBI) FCancelTriggerOrder(symbol, orderID string) (FCancelOrderData, error) {
	var resp FCancelOrderData
	req := make(map[string]interface{})
	req["symbol"] = symbol
	req["order_id"] = orderID
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fCancelTriggerOrder, nil, req, &resp)
}

// FCancelAllTriggerOrders cancels all trigger order for futures
func (h *HUOBI) FCancelAllTriggerOrders(contractCode currency.Pair, symbol, contractType string) (FCancelOrderData, error) {
	var resp FCancelOrderData
	req := make(map[string]interface{})
	req["symbol"] = symbol
	if contractCode != (currency.Pair{}) {
		codeValue, err := h.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	if contractType != "" {
		if !common.StringDataCompare(validContractTypes, contractType) {
			return resp, nil
		}
		req["contract_type"] = contractType
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fCancelAllTriggerOrders, nil, req, &resp)
}

// FQueryTriggerOpenOrders queries open trigger orders for futures
func (h *HUOBI) FQueryTriggerOpenOrders(contractCode currency.Pair, symbol string, pageIndex, pageSize int64) (FTriggerOpenOrders, error) {
	var resp FTriggerOpenOrders
	req := make(map[string]interface{})
	req["symbol"] = symbol
	if contractCode != (currency.Pair{}) {
		codeValue, err := h.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fTriggerOpenOrders, nil, req, &resp)
}

// FQueryTriggerOrderHistory queries trigger order history for futures
func (h *HUOBI) FQueryTriggerOrderHistory(contractCode currency.Pair, symbol, tradeType, status string, createDate, pageIndex, pageSize int64) (FTriggerOrderHistoryData, error) {
	var resp FTriggerOrderHistoryData
	req := make(map[string]interface{})
	req["symbol"] = symbol
	if contractCode != (currency.Pair{}) {
		codeValue, err := h.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
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
	return resp, h.FuturesAuthenticatedHTTPRequest(exchange.RestFutures, http.MethodPost, fTriggerOrderHistory, nil, req, &resp)
}

// FuturesAuthenticatedHTTPRequest sends authenticated requests to the HUOBI API
func (h *HUOBI) FuturesAuthenticatedHTTPRequest(ep exchange.URL, method, endpoint string, values url.Values, data, result interface{}) error {
	if !h.AllowAuthenticatedRequest() {
		return fmt.Errorf("%s %w", h.Name, exchange.ErrAuthenticatedRequestWithoutCredentialsSet)
	}
	ePoint, err := h.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
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
	urlPath :=
		common.EncodeURLValues(ePoint+endpoint, values) + "&" + sigValues.Encode()
	var body io.Reader
	var payload []byte
	if data != nil {
		payload, err = json.Marshal(data)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(payload)
	}
	var tempResp json.RawMessage
	var errCap errorCapture
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

func (h *HUOBI) formatFuturesCode(p currency.Code) (string, error) {
	pairFmt, err := h.GetPairFormat(asset.Futures, true)
	if err != nil {
		return "", err
	}
	if pairFmt.Uppercase {
		return p.Upper().String(), nil
	}
	return p.Lower().String(), nil
}
