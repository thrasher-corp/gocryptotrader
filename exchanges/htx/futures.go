package htx

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// Unauth
	futuresContractInfo              = "/api/v1/contract_contract_info"
	futuresContractIndexPrice        = "/api/v1/contract_index"
	futuresContractPriceLimitation   = "/api/v1/contract_price_limit"
	futuresContractOpenInterest      = "/api/v1/contract_open_interest"
	futuresEstimatedDeliveryPrice    = "/api/v1/contract_delivery_price"
	futuresContractMarketDepth       = "/market/depth"
	futuresContractKline             = "/market/history/kline"
	futuresMarketOverview            = "/market/detail/merged"
	futuresLastTradeContract         = "/market/trade"
	futuresContractBatchTradeRecords = "/market/history/trade"
	futuresTieredAdjustmentFactor    = "/api/v1/contract_adjustfactor"
	futuresHisContractOpenInterest   = "/api/v1/contract_his_open_interest"
	futuresSystemStatus              = "/api/v1/contract_api_state"
	futuresTopAccountsSentiment      = "/api/v1/contract_elite_account_ratio"
	futuresTopPositionsSentiment     = "/api/v1/contract_elite_position_ratio"
	futuresLiquidationOrders         = "/api/v3/contract_liquidation_orders"
	futuresIndexKline                = "/index/market/history/index"
	futuresBasis                     = "/index/market/history/basis"
	v3HistoryDirectionNext           = "next"

	// Auth
	futuresAccount                   = "/api/v1/contract_account_info"
	futuresPositionInformation       = "/api/v1/contract_position_info"
	futuresAllSubAccountAssets       = "/api/v1/contract_sub_account_list"
	futuresSingleSubAccountAssets    = "/api/v1/contract_sub_account_info"
	futuresSingleSubAccountPositions = "/api/v1/contract_sub_position_info"
	futuresFinancialRecords          = "/api/v3/contract_financial_record"
	futuresSettlementRecords         = "/api/v1/contract_user_settlement_records"
	futuresOrderLimitInfo            = "/api/v1/contract_order_limit"
	futuresContractTradingFee        = "/api/v1/contract_fee"
	futuresTransferLimitInfo         = "/api/v1/contract_transfer_limit"
	futuresPositionLimitInfo         = "/api/v1/contract_position_limit"
	futuresQueryAssetsAndPositions   = "/api/v1/contract_account_position_info"
	futuresTransfer                  = "/api/v1/contract_master_sub_transfer"
	futuresTransferRecords           = "/api/v1/contract_master_sub_transfer_record"
	futuresAvailableLeverage         = "/api/v1/contract_available_level_rate"
	futuresSwitchLeverage            = "/api/v1/contract_switch_lever_rate"
	futuresOrder                     = "/api/v1/contract_order"
	futuresBatchOrder                = "/api/v1/contract_batchorder"
	futuresCancelOrder               = "/api/v1/contract_cancel"
	futuresCancelAllOrders           = "/api/v1/contract_cancelall"
	futuresFlashCloseOrder           = "/api/v1/lightning_close_position"
	futuresOrderInfo                 = "/api/v1/contract_order_info"
	futuresOrderDetails              = "/api/v1/contract_order_detail"
	futuresQueryOpenOrders           = "/api/v1/contract_openorders"
	futuresOrderHistory              = "/api/v3/contract_hisorders"
	futuresMatchResult               = "/api/v3/contract_matchresults"
	futuresTriggerOrder              = "/api/v1/contract_trigger_order"
	futuresCancelTriggerOrder        = "/api/v1/contract_trigger_cancel"
	futuresCancelAllTriggerOrders    = "/api/v1/contract_trigger_cancelall"
	futuresTriggerOpenOrders         = "/api/v1/contract_trigger_openorders"
	futuresTriggerOrderHistory       = "/api/v1/contract_trigger_hisorders"
)

// FuturesGetContractInfo gets contract info for futures
func (e *Exchange) FuturesGetContractInfo(ctx context.Context, symbol, contractType string, code currency.Pair) (FuturesContractInfoResponse, error) {
	var resp FuturesContractInfoResponse
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if t := strings.ToLower(contractType); t != "" {
		if _, ok := contractExpiryNames[t]; !ok {
			return resp, fmt.Errorf("%w: %v", errInvalidContractType, t)
		}
		params.Set("contract_type", t)
	}
	if !code.IsEmpty() {
		codeValue, err := e.FormatSymbol(code, asset.Futures)
		if err != nil {
			return resp, err
		}
		params.Set("contract_code", codeValue)
	}
	path := common.EncodeURLValues(futuresContractInfo, params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesIndexPriceInfo gets index price info for a futures contract
func (e *Exchange) FuturesIndexPriceInfo(ctx context.Context, symbol currency.Code) (FuturesContractIndexPriceInfoResponse, error) {
	var resp FuturesContractIndexPriceInfoResponse
	params := url.Values{}
	if !symbol.IsEmpty() {
		codeValue, err := e.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", codeValue)
	}
	path := common.EncodeURLValues(futuresContractIndexPrice, params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesContractPriceLimitations gets price limits for a futures contract
func (e *Exchange) FuturesContractPriceLimitations(ctx context.Context, symbol, contractType string, code currency.Pair) (FuturesContractIndexPriceInfoResponse, error) {
	var resp FuturesContractIndexPriceInfoResponse
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if t := strings.ToLower(contractType); t != "" {
		if _, ok := contractExpiryNames[t]; !ok {
			return resp, fmt.Errorf("%w: %v", errInvalidContractType, t)
		}
		params.Set("contract_type", t)
	}
	if !code.IsEmpty() {
		codeValue, err := e.FormatSymbol(code, asset.Futures)
		if err != nil {
			return resp, err
		}
		params.Set("contract_code", codeValue)
	}
	path := common.EncodeURLValues(futuresContractPriceLimitation, params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesContractOpenInterest gets open interest data for futures contracts
func (e *Exchange) FuturesContractOpenInterest(ctx context.Context, symbol, contractType string, code currency.Pair) (FuturesContractOpenInterestResponse, error) {
	var resp FuturesContractOpenInterestResponse
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if t := strings.ToLower(contractType); t != "" {
		if _, ok := contractExpiryNames[t]; !ok {
			return resp, fmt.Errorf("%w: %v", errInvalidContractType, t)
		}
		params.Set("contract_type", t)
	}
	if !code.IsEmpty() {
		codeValue, err := e.formatFuturesPair(code, true)
		if err != nil {
			return resp, err
		}
		params.Set("contract_code", codeValue)
	}
	path := common.EncodeURLValues(futuresContractOpenInterest, params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesGetEstimatedDeliveryPrice gets estimated delivery price info for futures
func (e *Exchange) FuturesGetEstimatedDeliveryPrice(ctx context.Context, symbol currency.Code) (FuturesEstimatedDeliveryPriceResponse, error) {
	var resp FuturesEstimatedDeliveryPriceResponse
	params := url.Values{}
	codeValue, err := e.formatFuturesCode(symbol)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", codeValue)
	path := common.EncodeURLValues(futuresEstimatedDeliveryPrice, params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesGetMarketDepth gets market depth data for futures contracts
func (e *Exchange) FuturesGetMarketDepth(ctx context.Context, symbol currency.Pair, dataType string) (*FuturesOrderbook, error) {
	symbolValue, err := e.formatFuturesPair(symbol, false)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	params.Set("type", dataType)
	path := common.EncodeURLValues(futuresContractMarketDepth, params)

	var tempData FuturesMarketDepthResponse
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, &tempData)
	if err != nil {
		return nil, err
	}

	resp := FuturesOrderbook{
		Symbol: symbolValue,
		Bids:   make([]obItem, len(tempData.Tick.Bids)),
		Asks:   make([]obItem, len(tempData.Tick.Asks)),
	}
	for x := range tempData.Tick.Asks {
		resp.Asks[x] = obItem{
			Price:    tempData.Tick.Asks[x][0],
			Quantity: tempData.Tick.Asks[x][1],
		}
	}
	for y := range tempData.Tick.Bids {
		resp.Bids[y] = obItem{
			Price:    tempData.Tick.Bids[y][0],
			Quantity: tempData.Tick.Bids[y][1],
		}
	}
	return &resp, nil
}

// FuturesGetKline gets kline data for futures
func (e *Exchange) FuturesGetKline(ctx context.Context, symbol currency.Pair, period string, size uint64, startTime, endTime time.Time) (FuturesKlineResponse, error) {
	var resp FuturesKlineResponse
	params := url.Values{}
	symbolValue, err := e.formatFuturesPair(symbol, false)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if !common.StringSliceCompareInsensitive(validFuturesPeriods, period) {
		return resp, common.ErrInvalidPeriod
	}
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
	path := common.EncodeURLValues(futuresContractKline, params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesGetMarketOverview gets market overview data for futures
func (e *Exchange) FuturesGetMarketOverview(ctx context.Context, symbol currency.Pair) (FuturesMarketOverviewResponse, error) {
	var resp FuturesMarketOverviewResponse
	params := url.Values{}
	symbolValue, err := e.formatFuturesPair(symbol, false)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	path := common.EncodeURLValues(futuresMarketOverview, params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesLastTrade gets last trade data for a futures contract
func (e *Exchange) FuturesLastTrade(ctx context.Context, symbol currency.Pair) (FuturesLastTradeResponse, error) {
	var resp FuturesLastTradeResponse
	params := url.Values{}
	symbolValue, err := e.formatFuturesPair(symbol, false)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	path := common.EncodeURLValues(futuresLastTradeContract, params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesRequestPublicBatchTrades gets public batch trades for a futures contract
func (e *Exchange) FuturesRequestPublicBatchTrades(ctx context.Context, symbol currency.Pair, size uint64) (FuturesBatchTradesForContractResponse, error) {
	params := url.Values{}
	symbolValue, err := e.formatFuturesPair(symbol, false)
	if err != nil {
		return FuturesBatchTradesForContractResponse{}, err
	}
	params.Set("symbol", symbolValue)
	if size > 0 {
		params.Set("size", strconv.FormatUint(size, 10))
	}
	var resp FuturesBatchTradesForContractResponse
	path := common.EncodeURLValues(futuresContractBatchTradeRecords, params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesQueryTieredAdjustmentFactor gets tiered adjustment factor for futures contracts
func (e *Exchange) FuturesQueryTieredAdjustmentFactor(ctx context.Context, symbol currency.Code) (FuturesTieredAdjustmentFactorInfoResponse, error) {
	var resp FuturesTieredAdjustmentFactorInfoResponse
	params := url.Values{}
	if !symbol.IsEmpty() {
		codeValue, err := e.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", codeValue)
	}
	path := common.EncodeURLValues(futuresTieredAdjustmentFactor, params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesQueryHisOpenInterest gets open interest for futures contract
func (e *Exchange) FuturesQueryHisOpenInterest(ctx context.Context, symbol, contractType, period, amountType string, size uint64) (FuturesOpenInterestResponse, error) {
	var resp FuturesOpenInterestResponse
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	contractType = strings.ToLower(contractType)
	if _, ok := contractExpiryNames[contractType]; !ok {
		return resp, fmt.Errorf("%w: %v", errInvalidContractType, contractType)
	}
	params.Set("contract_type", contractType)
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, common.ErrInvalidPeriod
	}
	params.Set("period", period)
	if size == 0 || size > 200 {
		return resp, errInvalidSize
	}
	params.Set("size", strconv.FormatUint(size, 10))
	validAmount, ok := validAmountType[amountType]
	if !ok {
		return resp, errInvalidAmountType
	}
	params.Set("amount_type", strconv.FormatUint(validAmount, 10))
	path := common.EncodeURLValues(futuresHisContractOpenInterest, params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesQuerySystemStatus gets system status data
func (e *Exchange) FuturesQuerySystemStatus(ctx context.Context, symbol currency.Code) (FuturesContractOpenInterestResponse, error) {
	var resp FuturesContractOpenInterestResponse
	params := url.Values{}
	if !symbol.IsEmpty() {
		codeValue, err := e.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", codeValue)
	}
	path := common.EncodeURLValues(futuresSystemStatus, params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesQueryTopAccountsRatio gets top accounts' ratio
func (e *Exchange) FuturesQueryTopAccountsRatio(ctx context.Context, symbol, period string) (FuturesTopAccountsLongShortRatioResponse, error) {
	var resp FuturesTopAccountsLongShortRatioResponse
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, common.ErrInvalidPeriod
	}
	params.Set("period", period)
	path := common.EncodeURLValues(futuresTopAccountsSentiment, params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesQueryTopPositionsRatio gets top positions' long/short ratio for futures
func (e *Exchange) FuturesQueryTopPositionsRatio(ctx context.Context, symbol, period string) (FuturesTopPositionsLongShortRatioResponse, error) {
	var resp FuturesTopPositionsLongShortRatioResponse
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, common.ErrInvalidPeriod
	}
	params.Set("period", period)
	path := common.EncodeURLValues(futuresTopPositionsSentiment, params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesLiquidationOrders gets liquidation orders for futures contracts
func (e *Exchange) FuturesLiquidationOrders(ctx context.Context, symbol currency.Code, tradeType string, startTime, endTime uint64, direction string, fromID uint64) (SwapLiquidationOrdersResponse, error) {
	var resp SwapLiquidationOrdersResponse
	tType, ok := validTradeTypes[tradeType]
	if !ok {
		return resp, errInvalidTradeType
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	params.Set("trade_type", strconv.FormatUint(tType, 10))

	if startTime != 0 {
		params.Set("start_time", strconv.FormatUint(startTime, 10))
	}
	if endTime != 0 {
		params.Set("end_time", strconv.FormatUint(endTime, 10))
	}
	if direction != "" {
		params.Set("direct", direction)
	}
	if fromID != 0 {
		params.Set("from_id", strconv.FormatUint(fromID, 10))
	}
	path := common.EncodeURLValues(futuresLiquidationOrders, params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesIndexKline gets index kline data for futures contracts
func (e *Exchange) FuturesIndexKline(ctx context.Context, symbol currency.Pair, period string, size uint64) (FuturesIndexKlineResponse, error) {
	var resp FuturesIndexKlineResponse
	params := url.Values{}
	symbolValue, err := e.formatFuturesPair(symbol, false)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if !common.StringSliceCompareInsensitive(validFuturesPeriods, period) {
		return resp, common.ErrInvalidPeriod
	}
	params.Set("period", period)
	if size == 0 || size > 2000 {
		return resp, errInvalidSize
	}
	params.Set("size", strconv.FormatUint(size, 10))
	path := common.EncodeURLValues(futuresIndexKline, params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesGetBasis gets basis data futures contracts
func (e *Exchange) FuturesGetBasis(ctx context.Context, symbol currency.Pair, period, basisPriceType string, size uint64) (FuturesBasisResponse, error) {
	var resp FuturesBasisResponse
	params := url.Values{}
	symbolValue, err := e.formatFuturesPair(symbol, false)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if !common.StringSliceCompareInsensitive(validFuturesPeriods, period) {
		return resp, common.ErrInvalidPeriod
	}
	params.Set("period", period)
	if basisPriceType != "" {
		if common.StringSliceCompareInsensitive(validBasisPriceTypes, basisPriceType) {
			params.Set("basis_price_type", basisPriceType)
		}
	}
	if size > 0 && size <= 2000 {
		params.Set("size", strconv.FormatUint(size, 10))
	}
	path := common.EncodeURLValues(futuresBasis, params)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesGetAccountInfo gets user info for futures account
func (e *Exchange) FuturesGetAccountInfo(ctx context.Context, symbol currency.Code) (FuturesUserAccountResponse, error) {
	var resp FuturesUserAccountResponse
	req := make(map[string]any)
	if !symbol.IsEmpty() {
		codeValue, err := e.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		req["symbol"] = codeValue
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresAccount, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesGetPositionsInfo gets positions info for futures account
func (e *Exchange) FuturesGetPositionsInfo(ctx context.Context, symbol currency.Code) (FuturesUsersPositionsInfoResponse, error) {
	var resp FuturesUsersPositionsInfoResponse
	req := make(map[string]any)
	if !symbol.IsEmpty() {
		codeValue, err := e.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		req["symbol"] = codeValue
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresPositionInformation, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesGetAllSubAccountAssets gets assets info for all futures subaccounts
func (e *Exchange) FuturesGetAllSubAccountAssets(ctx context.Context, symbol currency.Code) (FuturesSubAccountAssetsInfoResponse, error) {
	var resp FuturesSubAccountAssetsInfoResponse
	req := make(map[string]any)
	if !symbol.IsEmpty() {
		codeValue, err := e.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		req["symbol"] = codeValue
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresAllSubAccountAssets, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesGetSingleSubAccountInfo gets assets info for a futures subaccount
func (e *Exchange) FuturesGetSingleSubAccountInfo(ctx context.Context, symbol, subUID string) (FuturesSingleSubAccountAssetsInfoResponse, error) {
	var resp FuturesSingleSubAccountAssetsInfoResponse
	req := make(map[string]any)
	if symbol != "" {
		req["symbol"] = symbol
	}
	req["sub_uid"] = subUID
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresSingleSubAccountAssets, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesGetSingleSubPositions gets positions info for a single sub account
func (e *Exchange) FuturesGetSingleSubPositions(ctx context.Context, symbol, subUID string) (FuturesSingleSubAccountPositionsInfoResponse, error) {
	var resp FuturesSingleSubAccountPositionsInfoResponse
	req := make(map[string]any)
	if symbol != "" {
		req["symbol"] = symbol
	}
	req["sub_uid"] = subUID
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresSingleSubAccountPositions, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesGetFinancialRecords gets financial records for futures
func (e *Exchange) FuturesGetFinancialRecords(ctx context.Context, symbol, recordType string, lookbackDays, fromID, limit uint64) (FuturesFinancialRecordsResponse, error) {
	var resp FuturesFinancialRecordsResponse
	req := make(map[string]any)
	if symbol != "" {
		req["symbol"] = symbol
	}
	if recordType != "" {
		rType, ok := validFuturesRecordTypes[recordType]
		if !ok {
			return resp, errInvalidRecordType
		}
		req["type"] = rType
	}
	if err := addV3HistoryTimeRange(req, lookbackDays); err != nil {
		return resp, err
	}
	if fromID != 0 {
		req["from_id"] = fromID
	}
	if limit != 0 {
		req[orderPriceTypeLimit] = limit
	}
	req["direct"] = v3HistoryDirectionNext
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresFinancialRecords, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesGetSettlementRecords gets settlement records for futures
func (e *Exchange) FuturesGetSettlementRecords(ctx context.Context, symbol currency.Code, pageIndex, pageSize uint64, startTime, endTime time.Time) (FuturesSettlementRecordsResponse, error) {
	var resp FuturesSettlementRecordsResponse
	req := make(map[string]any)
	req["symbol"] = symbol
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	if !startTime.IsZero() || !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return resp, err
		}
		req["start_time"] = strconv.FormatInt(startTime.Unix()*1000, 10)
		req["end_time"] = strconv.FormatInt(endTime.Unix()*1000, 10)
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresSettlementRecords, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesGetOrderLimits gets order limits for futures contracts
func (e *Exchange) FuturesGetOrderLimits(ctx context.Context, symbol, orderPriceType string) (FuturesContractOrderLimitResponse, error) {
	var resp FuturesContractOrderLimitResponse
	req := make(map[string]any)
	if symbol != "" {
		req["symbol"] = symbol
	}
	if orderPriceType != "" {
		if !common.StringSliceCompareInsensitive(validFuturesOrderPriceTypes, orderPriceType) {
			return resp, errInvalidOrderPriceType
		}
		req["order_price_type"] = orderPriceType
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresOrderLimitInfo, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesContractTradingFee gets futures contract trading fees
func (e *Exchange) FuturesContractTradingFee(ctx context.Context, symbol currency.Code) (FuturesContractTradingFeeResponse, error) {
	var resp FuturesContractTradingFeeResponse
	req := make(map[string]any)
	if !symbol.IsEmpty() {
		codeValue, err := e.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		req["symbol"] = codeValue
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresContractTradingFee, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesGetTransferLimits gets transfer limits for futures
func (e *Exchange) FuturesGetTransferLimits(ctx context.Context, symbol currency.Code) (FuturesTransferLimitResponse, error) {
	var resp FuturesTransferLimitResponse
	req := make(map[string]any)
	if !symbol.IsEmpty() {
		codeValue, err := e.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		req["symbol"] = codeValue
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresTransferLimitInfo, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesGetPositionLimits gets position limits for futures
func (e *Exchange) FuturesGetPositionLimits(ctx context.Context, symbol currency.Code) (FuturesPositionLimitResponse, error) {
	var resp FuturesPositionLimitResponse
	req := make(map[string]any)
	if !symbol.IsEmpty() {
		codeValue, err := e.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		req["symbol"] = codeValue
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresPositionLimitInfo, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesGetAssetsAndPositions gets assets and positions for futures
func (e *Exchange) FuturesGetAssetsAndPositions(ctx context.Context, symbol currency.Code) (FuturesAssetsAndPositionsResponse, error) {
	var resp FuturesAssetsAndPositionsResponse
	req := make(map[string]any)
	req["symbol"] = symbol
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresQueryAssetsAndPositions, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesTransfer transfers assets between master and subaccounts
func (e *Exchange) FuturesTransfer(ctx context.Context, subUID, symbol, transferType string, amount float64) (FuturesAccountTransferResponse, error) {
	var resp FuturesAccountTransferResponse
	req := make(map[string]any)
	req["symbol"] = symbol
	req["subUid"] = subUID
	req["amount"] = amount
	if !common.StringSliceCompareInsensitive(validTransferType, transferType) {
		return resp, errInvalidTransferType
	}
	req["type"] = transferType
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresTransfer, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesGetTransferRecords gets transfer records data for futures
func (e *Exchange) FuturesGetTransferRecords(ctx context.Context, symbol, transferType string, createDate, pageIndex, pageSize uint64) (FuturesTransferRecordsResponse, error) {
	var resp FuturesTransferRecordsResponse
	req := make(map[string]any)
	if symbol != "" {
		req["symbol"] = symbol
	}
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
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresTransferRecords, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesGetAvailableLeverage gets available leverage data for futures
func (e *Exchange) FuturesGetAvailableLeverage(ctx context.Context, symbol currency.Code) (FuturesAvailableLeverageResponse, error) {
	var resp FuturesAvailableLeverageResponse
	req := make(map[string]any)
	if !symbol.IsEmpty() {
		codeValue, err := e.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		req["symbol"] = codeValue
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresAvailableLeverage, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesSwitchLeverage changes the leverage used by coin-margined delivery contracts.
func (e *Exchange) FuturesSwitchLeverage(ctx context.Context, symbol currency.Code, leverage uint64) error {
	req := &FuturesSwitchLeverageRequest{
		Symbol:       symbol.String(),
		LeverageRate: leverage,
	}
	var resp *FuturesSwitchLeverageResponse
	return e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresSwitchLeverage, nil, req, &resp)
}

// FuturesOrder places an order for futures
func (e *Exchange) FuturesOrder(ctx context.Context, contractCode currency.Pair, symbol, contractType, clientOrderID, direction, offset, orderPriceType string, price, volume float64, leverageRate uint64) (FuturesOrderResponse, error) {
	var resp FuturesOrderResponse
	req := make(map[string]any)
	if symbol != "" {
		req["symbol"] = symbol
	}
	if t := strings.ToLower(contractType); t != "" {
		if _, ok := contractExpiryNames[t]; !ok {
			return resp, fmt.Errorf("%w: %v", errInvalidContractType, t)
		}
		req["contract_type"] = t
	}
	if !contractCode.IsEmpty() {
		codeValue, err := e.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	if clientOrderID != "" {
		// HTX accepts client order IDs from 1 to the largest signed 64-bit integer.
		id, err := strconv.ParseUint(clientOrderID, 10, 63)
		if err != nil {
			return resp, fmt.Errorf("%w: %w", errInvalidClientOrderID, err)
		}
		if id == 0 {
			return resp, errInvalidClientOrderID
		}
		req["client_order_id"] = id
	}
	req["direction"] = direction
	if !common.StringSliceCompareInsensitive(validOffsetTypes, offset) {
		return resp, errInvalidOffsetAmounts
	}
	if !common.StringSliceCompareInsensitive(validFuturesOrderPriceTypes, orderPriceType) {
		return resp, errInvalidOrderPriceType
	}
	req["order_price_type"] = orderPriceType
	req["lever_rate"] = leverageRate
	req["volume"] = volume
	req["price"] = price
	req["offset"] = offset
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresOrder, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesPlaceBatchOrder places a batch of orders for futures
func (e *Exchange) FuturesPlaceBatchOrder(ctx context.Context, data []FuturesBatchOrderRequest) (FuturesBatchOrderResponse, error) {
	var resp FuturesBatchOrderResponse
	req := make(map[string]any)
	if len(data) > 10 || len(data) == 0 {
		return resp, errBatchOrderLimitExceeded
	}
	for x := range data {
		if data[x].ContractCode != "" {
			unformattedPair, err := currency.NewPairFromString(data[x].ContractCode)
			if err != nil {
				return resp, err
			}
			formattedPair, err := e.FormatExchangeCurrency(unformattedPair, asset.Futures)
			if err != nil {
				return resp, err
			}
			data[x].ContractCode = formattedPair.String()
		}
		if data[x].ContractType != "" {
			if _, ok := contractExpiryNames[strings.ToLower(data[x].ContractType)]; !ok {
				return resp, fmt.Errorf("%w %v", errInvalidContractType, data[x].ContractType)
			}
		}
		if !common.StringSliceCompareInsensitive(validOffsetTypes, data[x].Offset) {
			return resp, errInvalidOffsetAmounts
		}
		if !common.StringSliceCompareInsensitive(validFuturesOrderPriceTypes, data[x].OrderPriceType) {
			return resp, errInvalidOrderPriceType
		}
	}
	req["orders_data"] = data
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresBatchOrder, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesCancelOrder cancels a futures order
func (e *Exchange) FuturesCancelOrder(ctx context.Context, baseCurrency currency.Code, orderID, clientOrderID string) (FuturesCancelOrderResponse, error) {
	var resp FuturesCancelOrderResponse
	if orderID == "" && clientOrderID == "" {
		return resp, order.ErrOrderIDNotSet
	}
	if orderID != "" && clientOrderID != "" {
		return resp, errMixedOrderIdentifiers
	}
	if strings.Count(orderID+clientOrderID, ",") >= 10 {
		return resp, errCancelOrderLimitExceeded
	}
	req := make(map[string]any)
	if baseCurrency.IsEmpty() {
		return resp, fmt.Errorf("cannot cancel futures order %w", currency.ErrCurrencyCodeEmpty)
	}
	req["symbol"] = baseCurrency.String() // Upper and lower case are supported
	if orderID != "" {
		req["order_id"] = orderID
	}
	if clientOrderID != "" {
		req["client_order_id"] = clientOrderID
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresCancelOrder, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesCancelAllOrders cancels all futures order for a given symbol
func (e *Exchange) FuturesCancelAllOrders(ctx context.Context, contractCode currency.Pair, symbol, contractType string) (FuturesCancelOrderResponse, error) {
	var resp FuturesCancelOrderResponse
	req := make(map[string]any)
	if symbol != "" {
		req["symbol"] = symbol
	}
	if t := strings.ToLower(contractType); t != "" {
		if _, ok := contractExpiryNames[t]; !ok {
			return resp, fmt.Errorf("%w: %v", errInvalidContractType, t)
		}
		req["contract_type"] = t
	}
	if !contractCode.IsEmpty() {
		codeValue, err := e.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresCancelAllOrders, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesFlashCloseOrder flash closes a futures order
func (e *Exchange) FuturesFlashCloseOrder(ctx context.Context, contractCode currency.Pair, symbol, contractType, direction, orderPriceType, clientOrderID string, volume float64) (FuturesOrderResponse, error) {
	var resp FuturesOrderResponse
	req := make(map[string]any)
	req["symbol"] = symbol
	if t := strings.ToLower(contractType); t != "" {
		if _, ok := contractExpiryNames[t]; !ok {
			return resp, fmt.Errorf("%w: %v", errInvalidContractType, t)
		}
		req["contract_type"] = t
	}
	if !contractCode.IsEmpty() {
		codeValue, err := e.FormatSymbol(contractCode, asset.Futures)
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
		if !common.StringSliceCompareInsensitive(validOPTypes, orderPriceType) {
			return resp, errInvalidOrderPriceType
		}
		req["orderPriceType"] = orderPriceType
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresFlashCloseOrder, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesGetOrderInfo gets order info for futures
func (e *Exchange) FuturesGetOrderInfo(ctx context.Context, symbol, clientOrderID, orderID string) (FuturesOrderInfoResponse, error) {
	var resp FuturesOrderInfoResponse
	req := make(map[string]any)
	req["symbol"] = symbol
	if orderID != "" {
		req["order_id"] = orderID
	}
	if clientOrderID != "" {
		req["client_order_id"] = clientOrderID
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresOrderInfo, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesOrderDetails gets order details for futures orders
func (e *Exchange) FuturesOrderDetails(ctx context.Context, symbol, orderID, orderType string, createdAt time.Time, pageIndex, pageSize uint64) (FuturesOrderDetailsResponse, error) {
	var resp FuturesOrderDetailsResponse
	req := make(map[string]any)
	req["symbol"] = symbol
	req["order_id"] = orderID
	req["created_at"] = strconv.FormatInt(createdAt.Unix(), 10)
	oType, ok := validOrderType[orderType]
	if !ok {
		return resp, errInvalidOrderType
	}
	req["order_type"] = oType
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresOrderDetails, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesGetOpenOrders gets order details for futures orders
func (e *Exchange) FuturesGetOpenOrders(ctx context.Context, symbol currency.Code, pageIndex, pageSize uint64) (FuturesOpenOrdersResponse, error) {
	var resp FuturesOpenOrdersResponse
	req := make(map[string]any)
	req["symbol"] = symbol
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresQueryOpenOrders, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesGetOrderHistory gets order history for futures using a lookback of at most two days.
func (e *Exchange) FuturesGetOrderHistory(ctx context.Context, contractCode currency.Pair, symbol, tradeType, reqType, orderType string, status []order.Status, lookbackDays, fromID, limit uint64) (FuturesOrderHistoryResponse, error) {
	if lookbackDays > 2 {
		return FuturesOrderHistoryResponse{}, errInvalidLookbackDays
	}
	var startTime, endTime time.Time
	if lookbackDays != 0 {
		endTime = time.Now().UTC()
		startTime = endTime.AddDate(0, 0, -int(lookbackDays))
	}
	return e.FuturesGetOrderHistoryByTimeRange(ctx, contractCode, symbol, tradeType, reqType, orderType, status, startTime, endTime, fromID, limit)
}

// FuturesGetOrderHistoryByTimeRange gets futures order history for an explicit interval.
func (e *Exchange) FuturesGetOrderHistoryByTimeRange(ctx context.Context, contractCode currency.Pair, symbol, tradeType, reqType, orderType string, status []order.Status, startTime, endTime time.Time, fromID, limit uint64) (FuturesOrderHistoryResponse, error) {
	var resp FuturesOrderHistoryResponse
	req := make(map[string]any)
	req["symbol"] = symbol
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
	if !contractCode.IsEmpty() {
		codeValue, err := e.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return resp, err
		}
		req["contract"] = codeValue
	}
	if orderType != "" {
		oType, ok := validFuturesOrderTypes[orderType]
		if !ok {
			return resp, errInvalidOrderType
		}
		req["order_type"] = oType
	}
	if fromID != 0 {
		req["from_id"] = fromID
	}
	if limit != 0 {
		req[orderPriceTypeLimit] = limit
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresOrderHistory, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesTradeHistory gets trade history data for futures
func (e *Exchange) FuturesTradeHistory(ctx context.Context, contractCode currency.Pair, symbol, tradeType string, lookbackDays, fromID, limit uint64) (FuturesTradeHistoryResponse, error) {
	var resp FuturesTradeHistoryResponse
	req := make(map[string]any)
	req["symbol"] = symbol
	tType, ok := validTradeType[tradeType]
	if !ok {
		return resp, errInvalidTradeType
	}
	req["trade_type"] = tType
	if !contractCode.IsEmpty() {
		codeValue, err := e.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return resp, err
		}
		req["contract"] = codeValue
	}
	if lookbackDays == 0 {
		return resp, errInvalidLookbackDays
	}
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
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresMatchResult, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

func addV3HistoryTimeRange(req map[string]any, lookbackDays uint64) error {
	if lookbackDays > 2 {
		return errInvalidLookbackDays
	}
	if lookbackDays == 0 {
		return nil
	}
	endTime := time.Now().UTC()
	req["start_time"] = endTime.AddDate(0, 0, -int(lookbackDays)).UnixMilli()
	req["end_time"] = endTime.UnixMilli()
	return nil
}

func formatFuturesOrderStatuses(statuses []order.Status) (string, error) {
	if len(statuses) == 0 {
		return "0", nil
	}
	formatted := make([]string, len(statuses))
	for i := range statuses {
		status, ok := validOrderStatus[statuses[i]]
		if !ok {
			return "", fmt.Errorf("%w: %s", errInvalidOrderStatus, statuses[i])
		}
		formatted[i] = strconv.FormatUint(status, 10)
	}
	return strings.Join(formatted, ","), nil
}

// FuturesPlaceTriggerOrder places a trigger order for futures
func (e *Exchange) FuturesPlaceTriggerOrder(ctx context.Context, contractCode currency.Pair, symbol, contractType, triggerType, orderPriceType, direction, offset string, triggerPrice, orderPrice, volume float64, leverageRate uint64) (FuturesTriggerOrderResponse, error) {
	var resp FuturesTriggerOrderResponse
	req := make(map[string]any)
	if symbol != "" {
		req["symbol"] = symbol
	}
	if t := strings.ToLower(contractType); t != "" {
		if _, ok := contractExpiryNames[t]; !ok {
			return resp, fmt.Errorf("%w: %v", errInvalidContractType, t)
		}
		req["contract_type"] = t
	}
	if !contractCode.IsEmpty() {
		codeValue, err := e.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	tType, ok := validTriggerType[triggerType]
	if !ok {
		return resp, errInvalidTriggerType
	}
	req["trigger_type"] = tType
	req["direction"] = direction
	if !common.StringSliceCompareInsensitive(validOffsetTypes, offset) {
		return resp, errInvalidOffset
	}
	req["offset"] = offset
	req["trigger_price"] = triggerPrice
	req["volume"] = volume
	req["lever_rate"] = leverageRate
	req["order_price"] = orderPrice
	if !common.StringSliceCompareInsensitive(validOrderPriceType, orderPriceType) {
		return resp, errInvalidOrderPriceType
	}
	req["order_price_type"] = orderPriceType
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresTriggerOrder, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesCancelTriggerOrder cancels trigger order for futures
func (e *Exchange) FuturesCancelTriggerOrder(ctx context.Context, symbol, orderID string) (FuturesCancelOrderResponse, error) {
	var resp FuturesCancelOrderResponse
	req := make(map[string]any)
	req["symbol"] = symbol
	req["order_id"] = orderID
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresCancelTriggerOrder, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesCancelAllTriggerOrders cancels all trigger order for futures
func (e *Exchange) FuturesCancelAllTriggerOrders(ctx context.Context, contractCode currency.Pair, symbol, contractType string) (FuturesCancelOrderResponse, error) {
	var resp FuturesCancelOrderResponse
	req := make(map[string]any)
	req["symbol"] = symbol
	if !contractCode.IsEmpty() {
		codeValue, err := e.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	if t := strings.ToLower(contractType); t != "" {
		if _, ok := contractExpiryNames[t]; !ok {
			return resp, fmt.Errorf("%w: %v", errInvalidContractType, t)
		}
		req["contract_type"] = t
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresCancelAllTriggerOrders, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesQueryTriggerOpenOrders queries open trigger orders for futures
func (e *Exchange) FuturesQueryTriggerOpenOrders(ctx context.Context, contractCode currency.Pair, symbol string, pageIndex, pageSize uint64) (FuturesTriggerOpenOrdersResponse, error) {
	var resp FuturesTriggerOpenOrdersResponse
	req := make(map[string]any)
	req["symbol"] = symbol
	if !contractCode.IsEmpty() {
		codeValue, err := e.FormatSymbol(contractCode, asset.Futures)
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
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresTriggerOpenOrders, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesQueryTriggerOrderHistory queries trigger order history for futures
func (e *Exchange) FuturesQueryTriggerOrderHistory(ctx context.Context, contractCode currency.Pair, symbol, tradeType, status string, createDate, pageIndex, pageSize uint64) (FuturesTriggerOrderHistoryResponse, error) {
	var resp FuturesTriggerOrderHistoryResponse
	req := make(map[string]any)
	req["symbol"] = symbol
	if !contractCode.IsEmpty() {
		codeValue, err := e.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	if tradeType != "" {
		tType, ok := validTradeType[tradeType]
		if !ok {
			return resp, errInvalidTradeType
		}
		req["trade_type"] = tType
	}
	validStatus, ok := validStatusTypes[status]
	if !ok {
		return resp, errInvalidOrderStatus
	}
	req["status"] = validStatus
	if createDate == 0 || createDate > 90 {
		return resp, errInvalidCreateDate
	}
	req["create_date"] = createDate
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	if err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresTriggerOrderHistory, nil, req, &resp); err != nil {
		return resp, err
	}
	return resp, nil
}

// FuturesAuthenticatedHTTPRequest sends authenticated requests to the HTX API
func (e *Exchange) FuturesAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method, endpoint string, values url.Values, data, result any) error {
	creds, err := e.GetCredentials(ctx)
	if err != nil {
		return err
	}
	ePoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	if ep == exchange.RestFutures {
		// prevent signature errors for non-standard paths until we can
		// have a method to force update endpoints
		ePoint = strings.TrimSuffix(ePoint, "/")
	}
	signatureHost, err := getSignatureHost(ePoint)
	if err != nil {
		return err
	}
	if values == nil {
		values = url.Values{}
	}

	var tempResp json.RawMessage
	responseHeaders := make(http.Header)
	newRequest := func() (*request.Item, error) {
		values.Del("Signature")
		values.Set("AccessKeyId", creds.Key)
		values.Set("SignatureMethod", "HmacSHA256")
		values.Set("SignatureVersion", "2")
		values.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05"))
		sigPath := method + "\n" + signatureHost + "\n" + endpoint + "\n" + values.Encode()
		headers := make(map[string]string)
		if method == http.MethodGet {
			headers["Content-Type"] = "application/x-www-form-urlencoded"
		} else {
			headers["Content-Type"] = "application/json"
		}

		hmac, err := crypto.GetHMAC(crypto.HashSHA256, []byte(sigPath), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}

		values.Set("Signature", base64.StdEncoding.EncodeToString(hmac))
		var body io.Reader
		var payload []byte
		if data != nil {
			payload, err = json.Marshal(data)
			if err != nil {
				return nil, err
			}
			body = bytes.NewBuffer(payload)
		}

		return &request.Item{
			Method:                 method,
			Path:                   common.EncodeURLValues(ePoint+endpoint, values),
			Headers:                headers,
			Body:                   body,
			Result:                 &tempResp,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
			HeaderResponse:         &responseHeaders,
		}, nil
	}

	err = e.SendPayload(ctx, getRateLimitID(ep, endpoint, method, true), newRequest, request.AuthenticatedRequest)
	if err != nil {
		return err
	}
	if responseHeaders.Get("Status") == strconv.Itoa(http.StatusNoContent) {
		if len(bytes.TrimSpace(tempResp)) != 0 {
			return fmt.Errorf("%w: status %d returned a response body", request.ErrAuthRequestFailed, http.StatusNoContent)
		}
		if result == nil {
			return nil
		}
		return common.AppendError(errExpectedResponseBody, request.ErrAuthRequestFailed)
	}

	var errCap errorCapture
	if err = json.Unmarshal(tempResp, &errCap); err == nil {
		if errCap.ErrMsgType1 != "" {
			return fmt.Errorf("%w error code: %v error message: %s", request.ErrAuthRequestFailed, errCap.CodeType1, errCap.ErrMsgType1)
		}
		if errCap.ErrMsgType2 != "" {
			return fmt.Errorf("%w error code: %v error message: %s", request.ErrAuthRequestFailed, errCap.CodeType2, errCap.ErrMsgType2)
		}
		if errCap.Status == htxStatusError {
			return fmt.Errorf("%w: status %s", request.ErrAuthRequestFailed, errCap.Status)
		}
	}
	if strings.HasPrefix(endpoint, "/v5/") || strings.Contains(endpoint, "/v3/") {
		var resp V5Response
		if err = json.Unmarshal(tempResp, &resp); err == nil && resp.Code != 0 && resp.Code != http.StatusOK {
			message := resp.Message
			if message == "" {
				message = resp.Msg
			}
			return fmt.Errorf("%w error code: %v error message: %s", request.ErrAuthRequestFailed, resp.Code, message)
		}
	}
	if err = unmarshalResponse(tempResp, result); err != nil {
		return common.AppendError(err, request.ErrAuthRequestFailed)
	}
	return nil
}

func (e *Exchange) formatFuturesCode(p currency.Code) (string, error) {
	pairFmt, err := e.GetPairFormat(asset.Futures, true)
	if err != nil {
		return "", err
	}
	if pairFmt.Uppercase {
		return p.Upper().String(), nil
	}
	return p.Lower().String(), nil
}

// formatFuturesPair handles pairs in the format as "BTC-NW" and "BTC210827"
func (e *Exchange) formatFuturesPair(p currency.Pair, convertQuoteToExpiry bool) (string, error) {
	if slices.Contains(validContractExpiryCodes, strings.ToUpper(p.Quote.String())) {
		if convertQuoteToExpiry {
			cp, err := e.pairFromContractExpiryCode(p)
			if err != nil {
				return "", err
			}
			return cp.String(), nil
		}
		return p.Format(currency.PairFormat{Delimiter: "_", Uppercase: true}).String(), nil
	}
	if p.Quote.IsStableCurrency() {
		return p.Format(currency.PairFormat{Delimiter: "-", Uppercase: true}).String(), nil
	}

	return e.FormatSymbol(p, asset.Futures)
}

// pairFromContractExpiryCode converts a pair with contract expiry shorthand in the Quote to a concrete tradable pair
// We need this because some apis, such as ticker, use BTC_CW, NW, CQ, NQ
// Other apis, such as contract_info, use contract type of this_week, next_week, quarter (sic), and next_quater
func (e *Exchange) pairFromContractExpiryCode(p currency.Pair) (currency.Pair, error) {
	e.futureContractCodesMutex.RLock()
	defer e.futureContractCodesMutex.RUnlock()
	exp, ok := e.futureContractCodes[p.Quote.String()]
	if !ok {
		return p, fmt.Errorf("%w: %s", errInvalidContractType, p.Quote.String())
	}
	p.Quote = exp
	return p, nil
}
