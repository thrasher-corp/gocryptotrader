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

const v3HistoryDirectionNext = "next"

// FuturesGetContractInfo gets contract info for futures
func (e *Exchange) FuturesGetContractInfo(ctx context.Context, symbol, contractType string, code currency.Pair) (*FuturesContractInfoResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if t := strings.ToLower(contractType); t != "" {
		if _, ok := contractExpiryNames[t]; !ok {
			return nil, fmt.Errorf("%w: %v", errInvalidContractType, t)
		}
		params.Set("contract_type", t)
	}
	if !code.IsEmpty() {
		codeValue, err := e.FormatSymbol(code, asset.Futures)
		if err != nil {
			return nil, err
		}
		params.Set("contract_code", codeValue)
	}
	path := common.EncodeURLValues("/api/v1/contract_contract_info", params)
	resp := new(FuturesContractInfoResponse)
	err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// FuturesIndexPriceInfo gets index price info for a futures contract
func (e *Exchange) FuturesIndexPriceInfo(ctx context.Context, symbol currency.Code) (*FuturesContractIndexPriceInfoResponse, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		codeValue, err := e.formatFuturesCode(symbol)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", codeValue)
	}
	path := common.EncodeURLValues("/api/v1/contract_index", params)
	resp := new(FuturesContractIndexPriceInfoResponse)
	err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// FuturesContractPriceLimitations gets price limits for a futures contract
func (e *Exchange) FuturesContractPriceLimitations(ctx context.Context, symbol, contractType string, code currency.Pair) (*FuturesContractIndexPriceInfoResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if t := strings.ToLower(contractType); t != "" {
		if _, ok := contractExpiryNames[t]; !ok {
			return nil, fmt.Errorf("%w: %v", errInvalidContractType, t)
		}
		params.Set("contract_type", t)
	}
	if !code.IsEmpty() {
		codeValue, err := e.FormatSymbol(code, asset.Futures)
		if err != nil {
			return nil, err
		}
		params.Set("contract_code", codeValue)
	}
	path := common.EncodeURLValues("/api/v1/contract_price_limit", params)
	resp := new(FuturesContractIndexPriceInfoResponse)
	err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// FuturesContractOpenInterest gets open interest data for futures contracts
func (e *Exchange) FuturesContractOpenInterest(ctx context.Context, symbol, contractType string, code currency.Pair) (*FuturesContractOpenInterestResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if t := strings.ToLower(contractType); t != "" {
		if _, ok := contractExpiryNames[t]; !ok {
			return nil, fmt.Errorf("%w: %v", errInvalidContractType, t)
		}
		params.Set("contract_type", t)
	}
	if !code.IsEmpty() {
		codeValue, err := e.formatFuturesPair(code, true)
		if err != nil {
			return nil, err
		}
		params.Set("contract_code", codeValue)
	}
	path := common.EncodeURLValues("/api/v1/contract_open_interest", params)
	resp := new(FuturesContractOpenInterestResponse)
	err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// FuturesGetEstimatedDeliveryPrice gets estimated delivery price info for futures
func (e *Exchange) FuturesGetEstimatedDeliveryPrice(ctx context.Context, symbol currency.Code) (*FuturesEstimatedDeliveryPriceResponse, error) {
	params := url.Values{}
	codeValue, err := e.formatFuturesCode(symbol)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", codeValue)
	path := common.EncodeURLValues("/api/v1/contract_delivery_price", params)
	resp := new(FuturesEstimatedDeliveryPriceResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
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
	path := common.EncodeURLValues("/market/depth", params)

	tempData := new(FuturesMarketDepthResponse)
	if err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, tempData); err != nil {
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
func (e *Exchange) FuturesGetKline(ctx context.Context, symbol currency.Pair, period string, size uint64, startTime, endTime time.Time) (*FuturesKlineResponse, error) {
	params := url.Values{}
	symbolValue, err := e.formatFuturesPair(symbol, false)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbolValue)
	if !common.StringSliceCompareInsensitive(validFuturesPeriods, period) {
		return nil, common.ErrInvalidPeriod
	}
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
	path := common.EncodeURLValues("/market/history/kline", params)
	resp := new(FuturesKlineResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// FuturesGetMarketOverview gets market overview data for futures
func (e *Exchange) FuturesGetMarketOverview(ctx context.Context, symbol currency.Pair) (*FuturesMarketOverviewResponse, error) {
	params := url.Values{}
	symbolValue, err := e.formatFuturesPair(symbol, false)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbolValue)
	path := common.EncodeURLValues("/market/detail/merged", params)
	resp := new(FuturesMarketOverviewResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// FuturesLastTrade gets last trade data for a futures contract
func (e *Exchange) FuturesLastTrade(ctx context.Context, symbol currency.Pair) (*FuturesLastTradeResponse, error) {
	params := url.Values{}
	symbolValue, err := e.formatFuturesPair(symbol, false)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbolValue)
	path := common.EncodeURLValues("/market/trade", params)
	resp := new(FuturesLastTradeResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// FuturesRequestPublicBatchTrades gets public batch trades for a futures contract
func (e *Exchange) FuturesRequestPublicBatchTrades(ctx context.Context, symbol currency.Pair, size uint64) (*FuturesBatchTradesForContractResponse, error) {
	params := url.Values{}
	symbolValue, err := e.formatFuturesPair(symbol, false)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbolValue)
	if size > 0 {
		params.Set("size", strconv.FormatUint(size, 10))
	}
	path := common.EncodeURLValues("/market/history/trade", params)
	resp := new(FuturesBatchTradesForContractResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// FuturesQueryTieredAdjustmentFactor gets tiered adjustment factor for futures contracts
func (e *Exchange) FuturesQueryTieredAdjustmentFactor(ctx context.Context, symbol currency.Code) (*FuturesTieredAdjustmentFactorInfoResponse, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		codeValue, err := e.formatFuturesCode(symbol)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", codeValue)
	}
	path := common.EncodeURLValues("/api/v1/contract_adjustfactor", params)
	resp := new(FuturesTieredAdjustmentFactorInfoResponse)
	err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// FuturesQueryHisOpenInterest gets open interest for futures contract
func (e *Exchange) FuturesQueryHisOpenInterest(ctx context.Context, symbol, contractType, period, amountType string, size uint64) (*FuturesOpenInterestResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	contractType = strings.ToLower(contractType)
	if _, ok := contractExpiryNames[contractType]; !ok {
		return nil, fmt.Errorf("%w: %v", errInvalidContractType, contractType)
	}
	params.Set("contract_type", contractType)
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return nil, common.ErrInvalidPeriod
	}
	params.Set("period", period)
	if size == 0 || size > 200 {
		return nil, errInvalidSize
	}
	params.Set("size", strconv.FormatUint(size, 10))
	validAmount, ok := validAmountType[amountType]
	if !ok {
		return nil, errInvalidAmountType
	}
	params.Set("amount_type", strconv.FormatUint(validAmount, 10))
	path := common.EncodeURLValues("/api/v1/contract_his_open_interest", params)
	resp := new(FuturesOpenInterestResponse)
	err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// FuturesQuerySystemStatus gets system status data
func (e *Exchange) FuturesQuerySystemStatus(ctx context.Context, symbol currency.Code) (*FuturesContractOpenInterestResponse, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		codeValue, err := e.formatFuturesCode(symbol)
		if err != nil {
			return nil, err
		}
		params.Set("symbol", codeValue)
	}
	path := common.EncodeURLValues("/api/v1/contract_api_state", params)
	resp := new(FuturesContractOpenInterestResponse)
	err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// FuturesQueryTopAccountsRatio gets top accounts' ratio
func (e *Exchange) FuturesQueryTopAccountsRatio(ctx context.Context, symbol, period string) (*FuturesTopAccountsLongShortRatioResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return nil, common.ErrInvalidPeriod
	}
	params.Set("period", period)
	path := common.EncodeURLValues("/api/v1/contract_elite_account_ratio", params)
	resp := new(FuturesTopAccountsLongShortRatioResponse)
	err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// FuturesQueryTopPositionsRatio gets top positions' long/short ratio for futures
func (e *Exchange) FuturesQueryTopPositionsRatio(ctx context.Context, symbol, period string) (*FuturesTopPositionsLongShortRatioResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return nil, common.ErrInvalidPeriod
	}
	params.Set("period", period)
	path := common.EncodeURLValues("/api/v1/contract_elite_position_ratio", params)
	resp := new(FuturesTopPositionsLongShortRatioResponse)
	err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// FuturesLiquidationOrders gets liquidation orders for futures contracts
func (e *Exchange) FuturesLiquidationOrders(ctx context.Context, symbol currency.Code, tradeType string, startTime, endTime uint64, direction string, fromID uint64) (*SwapLiquidationOrdersResponse, error) {
	tType, ok := validTradeTypes[tradeType]
	if !ok {
		return nil, errInvalidTradeType
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
	path := common.EncodeURLValues("/api/v3/contract_liquidation_orders", params)
	resp := new(SwapLiquidationOrdersResponse)
	err := e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// FuturesIndexKline gets index kline data for futures contracts
func (e *Exchange) FuturesIndexKline(ctx context.Context, symbol currency.Pair, period string, size uint64) (*FuturesIndexKlineResponse, error) {
	params := url.Values{}
	symbolValue, err := e.formatFuturesPair(symbol, false)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbolValue)
	if !common.StringSliceCompareInsensitive(validFuturesPeriods, period) {
		return nil, common.ErrInvalidPeriod
	}
	params.Set("period", period)
	if size == 0 || size > 2000 {
		return nil, errInvalidSize
	}
	params.Set("size", strconv.FormatUint(size, 10))
	path := common.EncodeURLValues("/index/market/history/index", params)
	resp := new(FuturesIndexKlineResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// FuturesGetBasis gets basis data futures contracts
func (e *Exchange) FuturesGetBasis(ctx context.Context, symbol currency.Pair, period, basisPriceType string, size uint64) (*FuturesBasisResponse, error) {
	params := url.Values{}
	symbolValue, err := e.formatFuturesPair(symbol, false)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbolValue)
	if !common.StringSliceCompareInsensitive(validFuturesPeriods, period) {
		return nil, common.ErrInvalidPeriod
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
	path := common.EncodeURLValues("/index/market/history/basis", params)
	resp := new(FuturesBasisResponse)
	err = e.SendHTTPRequest(ctx, exchange.RestFutures, path, resp)
	return resp, err
}

// FuturesGetAccountInfo gets user info for futures account
func (e *Exchange) FuturesGetAccountInfo(ctx context.Context, symbol currency.Code) (*FuturesUserAccountResponse, error) {
	req := make(map[string]any)
	if !symbol.IsEmpty() {
		codeValue, err := e.formatFuturesCode(symbol)
		if err != nil {
			return nil, err
		}
		req["symbol"] = codeValue
	}
	resp := new(FuturesUserAccountResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_account_info", nil, req, resp)
	return resp, err
}

// FuturesGetPositionsInfo gets positions info for futures account
func (e *Exchange) FuturesGetPositionsInfo(ctx context.Context, symbol currency.Code) (*FuturesUsersPositionsInfoResponse, error) {
	req := make(map[string]any)
	if !symbol.IsEmpty() {
		codeValue, err := e.formatFuturesCode(symbol)
		if err != nil {
			return nil, err
		}
		req["symbol"] = codeValue
	}
	resp := new(FuturesUsersPositionsInfoResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_position_info", nil, req, resp)
	return resp, err
}

// FuturesGetAllSubAccountAssets gets assets info for all futures subaccounts
func (e *Exchange) FuturesGetAllSubAccountAssets(ctx context.Context, symbol currency.Code) (*FuturesSubAccountAssetsInfoResponse, error) {
	req := make(map[string]any)
	if !symbol.IsEmpty() {
		codeValue, err := e.formatFuturesCode(symbol)
		if err != nil {
			return nil, err
		}
		req["symbol"] = codeValue
	}
	resp := new(FuturesSubAccountAssetsInfoResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_sub_account_list", nil, req, resp)
	return resp, err
}

// FuturesGetSingleSubAccountInfo gets assets info for a futures subaccount
func (e *Exchange) FuturesGetSingleSubAccountInfo(ctx context.Context, symbol, subUID string) (*FuturesSingleSubAccountAssetsInfoResponse, error) {
	req := make(map[string]any)
	if symbol != "" {
		req["symbol"] = symbol
	}
	req["sub_uid"] = subUID
	resp := new(FuturesSingleSubAccountAssetsInfoResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_sub_account_info", nil, req, resp)
	return resp, err
}

// FuturesGetSingleSubPositions gets positions info for a single sub account
func (e *Exchange) FuturesGetSingleSubPositions(ctx context.Context, symbol, subUID string) (*FuturesSingleSubAccountPositionsInfoResponse, error) {
	req := make(map[string]any)
	if symbol != "" {
		req["symbol"] = symbol
	}
	req["sub_uid"] = subUID
	resp := new(FuturesSingleSubAccountPositionsInfoResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_sub_position_info", nil, req, resp)
	return resp, err
}

// FuturesGetFinancialRecords gets financial records for futures
func (e *Exchange) FuturesGetFinancialRecords(ctx context.Context, symbol, recordType string, lookbackDays, fromID, limit uint64) (*FuturesFinancialRecordsResponse, error) {
	req := make(map[string]any)
	if symbol != "" {
		req["symbol"] = symbol
	}
	if recordType != "" {
		rType, ok := validFuturesRecordTypes[recordType]
		if !ok {
			return nil, errInvalidRecordType
		}
		req["type"] = rType
	}
	if err := addV3HistoryTimeRange(req, lookbackDays); err != nil {
		return nil, err
	}
	if fromID != 0 {
		req["from_id"] = fromID
	}
	if limit != 0 {
		req["limit"] = limit
	}
	req["direct"] = v3HistoryDirectionNext
	resp := new(FuturesFinancialRecordsResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v3/contract_financial_record", nil, req, resp)
	return resp, err
}

// FuturesGetSettlementRecords gets settlement records for futures
func (e *Exchange) FuturesGetSettlementRecords(ctx context.Context, symbol currency.Code, pageIndex, pageSize uint64, startTime, endTime time.Time) (*FuturesSettlementRecordsResponse, error) {
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
			return nil, err
		}
		req["start_time"] = strconv.FormatInt(startTime.Unix()*1000, 10)
		req["end_time"] = strconv.FormatInt(endTime.Unix()*1000, 10)
	}
	resp := new(FuturesSettlementRecordsResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_user_settlement_records", nil, req, resp)
	return resp, err
}

// FuturesGetOrderLimits gets order limits for futures contracts
func (e *Exchange) FuturesGetOrderLimits(ctx context.Context, symbol, orderPriceType string) (*FuturesContractOrderLimitResponse, error) {
	req := make(map[string]any)
	if symbol != "" {
		req["symbol"] = symbol
	}
	if orderPriceType != "" {
		if !common.StringSliceCompareInsensitive(validFuturesOrderPriceTypes, orderPriceType) {
			return nil, errInvalidOrderPriceType
		}
		req["order_price_type"] = orderPriceType
	}
	resp := new(FuturesContractOrderLimitResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_order_limit", nil, req, resp)
	return resp, err
}

// FuturesContractTradingFee gets futures contract trading fees
func (e *Exchange) FuturesContractTradingFee(ctx context.Context, symbol currency.Code) (*FuturesContractTradingFeeResponse, error) {
	req := make(map[string]any)
	if !symbol.IsEmpty() {
		codeValue, err := e.formatFuturesCode(symbol)
		if err != nil {
			return nil, err
		}
		req["symbol"] = codeValue
	}
	resp := new(FuturesContractTradingFeeResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_fee", nil, req, resp)
	return resp, err
}

// FuturesGetTransferLimits gets transfer limits for futures
func (e *Exchange) FuturesGetTransferLimits(ctx context.Context, symbol currency.Code) (*FuturesTransferLimitResponse, error) {
	req := make(map[string]any)
	if !symbol.IsEmpty() {
		codeValue, err := e.formatFuturesCode(symbol)
		if err != nil {
			return nil, err
		}
		req["symbol"] = codeValue
	}
	resp := new(FuturesTransferLimitResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_transfer_limit", nil, req, resp)
	return resp, err
}

// FuturesGetPositionLimits gets position limits for futures
func (e *Exchange) FuturesGetPositionLimits(ctx context.Context, symbol currency.Code) (*FuturesPositionLimitResponse, error) {
	req := make(map[string]any)
	if !symbol.IsEmpty() {
		codeValue, err := e.formatFuturesCode(symbol)
		if err != nil {
			return nil, err
		}
		req["symbol"] = codeValue
	}
	resp := new(FuturesPositionLimitResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_position_limit", nil, req, resp)
	return resp, err
}

// FuturesGetAssetsAndPositions gets assets and positions for futures
func (e *Exchange) FuturesGetAssetsAndPositions(ctx context.Context, symbol currency.Code) (*FuturesAssetsAndPositionsResponse, error) {
	req := make(map[string]any)
	req["symbol"] = symbol
	resp := new(FuturesAssetsAndPositionsResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_account_position_info", nil, req, resp)
	return resp, err
}

// FuturesTransfer transfers assets between master and subaccounts
func (e *Exchange) FuturesTransfer(ctx context.Context, subUID, symbol, transferType string, amount float64) (*FuturesAccountTransferResponse, error) {
	req := make(map[string]any)
	req["symbol"] = symbol
	req["subUid"] = subUID
	req["amount"] = amount
	if !common.StringSliceCompareInsensitive(validTransferType, transferType) {
		return nil, errInvalidTransferType
	}
	req["type"] = transferType
	resp := new(FuturesAccountTransferResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_master_sub_transfer", nil, req, resp)
	return resp, err
}

// FuturesGetTransferRecords gets transfer records data for futures
func (e *Exchange) FuturesGetTransferRecords(ctx context.Context, symbol, transferType string, createDate, pageIndex, pageSize uint64) (*FuturesTransferRecordsResponse, error) {
	req := make(map[string]any)
	if symbol != "" {
		req["symbol"] = symbol
	}
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
	resp := new(FuturesTransferRecordsResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_master_sub_transfer_record", nil, req, resp)
	return resp, err
}

// FuturesGetAvailableLeverage gets available leverage data for futures
func (e *Exchange) FuturesGetAvailableLeverage(ctx context.Context, symbol currency.Code) (*FuturesAvailableLeverageResponse, error) {
	req := make(map[string]any)
	if !symbol.IsEmpty() {
		codeValue, err := e.formatFuturesCode(symbol)
		if err != nil {
			return nil, err
		}
		req["symbol"] = codeValue
	}
	resp := new(FuturesAvailableLeverageResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_available_level_rate", nil, req, resp)
	return resp, err
}

// FuturesSwitchLeverage changes the leverage used by coin-margined delivery contracts.
func (e *Exchange) FuturesSwitchLeverage(ctx context.Context, symbol currency.Code, leverage uint64) error {
	req := &FuturesSwitchLeverageRequest{
		Symbol:       symbol.String(),
		LeverageRate: leverage,
	}
	var resp *FuturesSwitchLeverageResponse
	return e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_switch_lever_rate", nil, req, &resp)
}

// FuturesOrder places an order for futures
func (e *Exchange) FuturesOrder(ctx context.Context, contractCode currency.Pair, symbol, contractType, clientOrderID, direction, offset, orderPriceType string, price, volume float64, leverageRate uint64) (*FuturesOrderResponse, error) {
	req := make(map[string]any)
	if symbol != "" {
		req["symbol"] = symbol
	}
	if t := strings.ToLower(contractType); t != "" {
		if _, ok := contractExpiryNames[t]; !ok {
			return nil, fmt.Errorf("%w: %v", errInvalidContractType, t)
		}
		req["contract_type"] = t
	}
	if !contractCode.IsEmpty() {
		codeValue, err := e.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return nil, err
		}
		req["contract_code"] = codeValue
	}
	if clientOrderID != "" {
		// HTX accepts client order IDs from 1 to the largest signed 64-bit integer.
		id, err := strconv.ParseUint(clientOrderID, 10, 63)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", errInvalidClientOrderID, err)
		}
		if id == 0 {
			return nil, errInvalidClientOrderID
		}
		req["client_order_id"] = id
	}
	req["direction"] = direction
	if !common.StringSliceCompareInsensitive(validOffsetTypes, offset) {
		return nil, errInvalidOffsetAmounts
	}
	if !common.StringSliceCompareInsensitive(validFuturesOrderPriceTypes, orderPriceType) {
		return nil, errInvalidOrderPriceType
	}
	req["order_price_type"] = orderPriceType
	req["lever_rate"] = leverageRate
	req["volume"] = volume
	req["price"] = price
	req["offset"] = offset
	resp := new(FuturesOrderResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_order", nil, req, resp)
	return resp, err
}

// FuturesPlaceBatchOrder places a batch of orders for futures
func (e *Exchange) FuturesPlaceBatchOrder(ctx context.Context, data []*FuturesBatchOrderRequest) (*FuturesBatchOrderResponse, error) {
	req := make(map[string]any)
	if len(data) == 0 {
		return nil, errEmptyOrderBatch
	}
	if len(data) > 10 {
		return nil, errBatchOrderLimitExceeded
	}
	for x := range data {
		if data[x] == nil {
			return nil, common.ErrNilPointer
		}
		if data[x].ContractCode != "" {
			unformattedPair, err := currency.NewPairFromString(data[x].ContractCode)
			if err != nil {
				return nil, err
			}
			formattedPair, err := e.FormatExchangeCurrency(unformattedPair, asset.Futures)
			if err != nil {
				return nil, err
			}
			data[x].ContractCode = formattedPair.String()
		}
		if data[x].ContractType != "" {
			if _, ok := contractExpiryNames[strings.ToLower(data[x].ContractType)]; !ok {
				return nil, fmt.Errorf("%w %v", errInvalidContractType, data[x].ContractType)
			}
		}
		if !common.StringSliceCompareInsensitive(validOffsetTypes, data[x].Offset) {
			return nil, errInvalidOffsetAmounts
		}
		if !common.StringSliceCompareInsensitive(validFuturesOrderPriceTypes, data[x].OrderPriceType) {
			return nil, errInvalidOrderPriceType
		}
	}
	req["orders_data"] = data
	resp := new(FuturesBatchOrderResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_batchorder", nil, req, resp)
	return resp, err
}

// FuturesCancelOrder cancels a futures order
func (e *Exchange) FuturesCancelOrder(ctx context.Context, baseCurrency currency.Code, orderID, clientOrderID string) (*FuturesCancelOrderResponse, error) {
	if orderID == "" && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if orderID != "" && clientOrderID != "" {
		return nil, errMixedOrderIdentifiers
	}
	if strings.Count(orderID+clientOrderID, ",") >= 10 {
		return nil, errCancelOrderLimitExceeded
	}
	req := make(map[string]any)
	if baseCurrency.IsEmpty() {
		return nil, fmt.Errorf("cannot cancel futures order %w", currency.ErrCurrencyCodeEmpty)
	}
	req["symbol"] = baseCurrency.String() // Upper and lower case are supported
	if orderID != "" {
		req["order_id"] = orderID
	}
	if clientOrderID != "" {
		req["client_order_id"] = clientOrderID
	}
	resp := new(FuturesCancelOrderResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_cancel", nil, req, resp)
	return resp, err
}

// FuturesCancelAllOrders cancels all futures order for a given symbol
func (e *Exchange) FuturesCancelAllOrders(ctx context.Context, contractCode currency.Pair, symbol, contractType string) (*FuturesCancelOrderResponse, error) {
	req := make(map[string]any)
	if symbol != "" {
		req["symbol"] = symbol
	}
	if t := strings.ToLower(contractType); t != "" {
		if _, ok := contractExpiryNames[t]; !ok {
			return nil, fmt.Errorf("%w: %v", errInvalidContractType, t)
		}
		req["contract_type"] = t
	}
	if !contractCode.IsEmpty() {
		codeValue, err := e.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return nil, err
		}
		req["contract_code"] = codeValue
	}
	resp := new(FuturesCancelOrderResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_cancelall", nil, req, resp)
	return resp, err
}

// FuturesFlashCloseOrder flash closes a futures order
func (e *Exchange) FuturesFlashCloseOrder(ctx context.Context, contractCode currency.Pair, symbol, contractType, direction, orderPriceType, clientOrderID string, volume float64) (*FuturesOrderResponse, error) {
	req := make(map[string]any)
	req["symbol"] = symbol
	if t := strings.ToLower(contractType); t != "" {
		if _, ok := contractExpiryNames[t]; !ok {
			return nil, fmt.Errorf("%w: %v", errInvalidContractType, t)
		}
		req["contract_type"] = t
	}
	if !contractCode.IsEmpty() {
		codeValue, err := e.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return nil, err
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
			return nil, errInvalidOrderPriceType
		}
		req["orderPriceType"] = orderPriceType
	}
	resp := new(FuturesOrderResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/lightning_close_position", nil, req, resp)
	return resp, err
}

// FuturesGetOrderInfo gets order info for futures
func (e *Exchange) FuturesGetOrderInfo(ctx context.Context, symbol, clientOrderID, orderID string) (*FuturesOrderInfoResponse, error) {
	req := make(map[string]any)
	req["symbol"] = symbol
	if orderID != "" {
		req["order_id"] = orderID
	}
	if clientOrderID != "" {
		req["client_order_id"] = clientOrderID
	}
	resp := new(FuturesOrderInfoResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_order_info", nil, req, resp)
	return resp, err
}

// FuturesOrderDetails gets order details for futures orders
func (e *Exchange) FuturesOrderDetails(ctx context.Context, symbol, orderID, orderType string, createdAt time.Time, pageIndex, pageSize uint64) (*FuturesOrderDetailsResponse, error) {
	req := make(map[string]any)
	req["symbol"] = symbol
	req["order_id"] = orderID
	req["created_at"] = strconv.FormatInt(createdAt.Unix(), 10)
	oType, ok := validOrderType[orderType]
	if !ok {
		return nil, errInvalidOrderType
	}
	req["order_type"] = oType
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	resp := new(FuturesOrderDetailsResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_order_detail", nil, req, resp)
	return resp, err
}

// FuturesGetOpenOrders gets order details for futures orders
func (e *Exchange) FuturesGetOpenOrders(ctx context.Context, symbol currency.Code, pageIndex, pageSize uint64) (*FuturesOpenOrdersResponse, error) {
	req := make(map[string]any)
	req["symbol"] = symbol
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	resp := new(FuturesOpenOrdersResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_openorders", nil, req, resp)
	return resp, err
}

// FuturesGetOrderHistory gets order history for futures using a lookback of at most two days.
func (e *Exchange) FuturesGetOrderHistory(ctx context.Context, contractCode currency.Pair, symbol, tradeType, reqType, orderType string, status []order.Status, lookbackDays, fromID, limit uint64) (*FuturesOrderHistoryResponse, error) {
	if lookbackDays > 2 {
		return nil, errInvalidLookbackDays
	}
	var startTime, endTime time.Time
	if lookbackDays != 0 {
		endTime = time.Now().UTC()
		startTime = endTime.AddDate(0, 0, -int(lookbackDays))
	}
	return e.FuturesGetOrderHistoryByTimeRange(ctx, contractCode, symbol, tradeType, reqType, orderType, status, startTime, endTime, fromID, limit)
}

// FuturesGetOrderHistoryByTimeRange gets futures order history for an explicit interval.
func (e *Exchange) FuturesGetOrderHistoryByTimeRange(ctx context.Context, contractCode currency.Pair, symbol, tradeType, reqType, orderType string, status []order.Status, startTime, endTime time.Time, fromID, limit uint64) (*FuturesOrderHistoryResponse, error) {
	req := make(map[string]any)
	req["symbol"] = symbol
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
	if !contractCode.IsEmpty() {
		codeValue, err := e.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return nil, err
		}
		req["contract"] = codeValue
	}
	if orderType != "" {
		oType, ok := validFuturesOrderTypes[orderType]
		if !ok {
			return nil, errInvalidOrderType
		}
		req["order_type"] = oType
	}
	if fromID != 0 {
		req["from_id"] = fromID
	}
	if limit != 0 {
		req["limit"] = limit
	}
	resp := new(FuturesOrderHistoryResponse)
	err = e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v3/contract_hisorders", nil, req, resp)
	return resp, err
}

// FuturesTradeHistory gets trade history data for futures
func (e *Exchange) FuturesTradeHistory(ctx context.Context, contractCode currency.Pair, symbol, tradeType string, lookbackDays, fromID, limit uint64) (*FuturesTradeHistoryResponse, error) {
	req := make(map[string]any)
	req["symbol"] = symbol
	tType, ok := validTradeType[tradeType]
	if !ok {
		return nil, errInvalidTradeType
	}
	req["trade_type"] = tType
	if !contractCode.IsEmpty() {
		codeValue, err := e.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return nil, err
		}
		req["contract"] = codeValue
	}
	if lookbackDays == 0 {
		return nil, errInvalidLookbackDays
	}
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
	resp := new(FuturesTradeHistoryResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v3/contract_matchresults", nil, req, resp)
	return resp, err
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
func (e *Exchange) FuturesPlaceTriggerOrder(ctx context.Context, contractCode currency.Pair, symbol, contractType, triggerType, orderPriceType, direction, offset string, triggerPrice, orderPrice, volume float64, leverageRate uint64) (*FuturesTriggerOrderResponse, error) {
	req := make(map[string]any)
	if symbol != "" {
		req["symbol"] = symbol
	}
	if t := strings.ToLower(contractType); t != "" {
		if _, ok := contractExpiryNames[t]; !ok {
			return nil, fmt.Errorf("%w: %v", errInvalidContractType, t)
		}
		req["contract_type"] = t
	}
	if !contractCode.IsEmpty() {
		codeValue, err := e.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return nil, err
		}
		req["contract_code"] = codeValue
	}
	tType, ok := validTriggerType[triggerType]
	if !ok {
		return nil, errInvalidTriggerType
	}
	req["trigger_type"] = tType
	req["direction"] = direction
	if !common.StringSliceCompareInsensitive(validOffsetTypes, offset) {
		return nil, errInvalidOffset
	}
	req["offset"] = offset
	req["trigger_price"] = triggerPrice
	req["volume"] = volume
	req["lever_rate"] = leverageRate
	req["order_price"] = orderPrice
	if !common.StringSliceCompareInsensitive(validOrderPriceType, orderPriceType) {
		return nil, errInvalidOrderPriceType
	}
	req["order_price_type"] = orderPriceType
	resp := new(FuturesTriggerOrderResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_trigger_order", nil, req, resp)
	return resp, err
}

// FuturesCancelTriggerOrder cancels trigger order for futures
func (e *Exchange) FuturesCancelTriggerOrder(ctx context.Context, symbol, orderID string) (*FuturesCancelOrderResponse, error) {
	req := make(map[string]any)
	req["symbol"] = symbol
	req["order_id"] = orderID
	resp := new(FuturesCancelOrderResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_trigger_cancel", nil, req, resp)
	return resp, err
}

// FuturesCancelAllTriggerOrders cancels all trigger order for futures
func (e *Exchange) FuturesCancelAllTriggerOrders(ctx context.Context, contractCode currency.Pair, symbol, contractType string) (*FuturesCancelOrderResponse, error) {
	req := make(map[string]any)
	req["symbol"] = symbol
	if !contractCode.IsEmpty() {
		codeValue, err := e.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return nil, err
		}
		req["contract_code"] = codeValue
	}
	if t := strings.ToLower(contractType); t != "" {
		if _, ok := contractExpiryNames[t]; !ok {
			return nil, fmt.Errorf("%w: %v", errInvalidContractType, t)
		}
		req["contract_type"] = t
	}
	resp := new(FuturesCancelOrderResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_trigger_cancelall", nil, req, resp)
	return resp, err
}

// FuturesQueryTriggerOpenOrders queries open trigger orders for futures
func (e *Exchange) FuturesQueryTriggerOpenOrders(ctx context.Context, contractCode currency.Pair, symbol string, pageIndex, pageSize uint64) (*FuturesTriggerOpenOrdersResponse, error) {
	req := make(map[string]any)
	req["symbol"] = symbol
	if !contractCode.IsEmpty() {
		codeValue, err := e.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return nil, err
		}
		req["contract_code"] = codeValue
	}
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	resp := new(FuturesTriggerOpenOrdersResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_trigger_openorders", nil, req, resp)
	return resp, err
}

// FuturesQueryTriggerOrderHistory queries trigger order history for futures
func (e *Exchange) FuturesQueryTriggerOrderHistory(ctx context.Context, contractCode currency.Pair, symbol, tradeType, status string, createDate, pageIndex, pageSize uint64) (*FuturesTriggerOrderHistoryResponse, error) {
	req := make(map[string]any)
	req["symbol"] = symbol
	if !contractCode.IsEmpty() {
		codeValue, err := e.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return nil, err
		}
		req["contract_code"] = codeValue
	}
	if tradeType != "" {
		tType, ok := validTradeType[tradeType]
		if !ok {
			return nil, errInvalidTradeType
		}
		req["trade_type"] = tType
	}
	validStatus, ok := validStatusTypes[status]
	if !ok {
		return nil, errInvalidOrderStatus
	}
	req["status"] = validStatus
	if createDate == 0 || createDate > 90 {
		return nil, errInvalidCreateDate
	}
	req["create_date"] = createDate
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	resp := new(FuturesTriggerOrderHistoryResponse)
	err := e.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, "/api/v1/contract_trigger_hisorders", nil, req, resp)
	return resp, err
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

	response := &httpResponse{
		Result:        result,
		Authenticated: true,
		Futures:       true,
		Version5:      strings.HasPrefix(endpoint, "/v5/") || strings.Contains(endpoint, "/v3/"),
	}
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
			Result:                 response,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}

	if err := e.SendPayload(ctx, getRateLimitID(ep, endpoint, method, true), newRequest, request.AuthenticatedRequest); err != nil {
		return err
	}
	if !response.Received && result != nil {
		return common.AppendError(errExpectedResponseBody, request.ErrAuthRequestFailed)
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
