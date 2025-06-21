package huobi

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
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
	fContractInfo              = "/api/v1/contract_contract_info"
	fContractIndexPrice        = "/api/v1/contract_index"
	fContractPriceLimitation   = "/api/v1/contract_price_limit"
	fContractOpenInterest      = "/api/v1/contract_open_interest"
	fEstimatedDeliveryPrice    = "/api/v1/contract_delivery_price"
	fContractMarketDepth       = "/market/depth"
	fContractKline             = "/market/history/kline"
	fMarketOverview            = "/market/detail/merged"
	fLastTradeContract         = "/market/trade"
	fContractBatchTradeRecords = "/market/history/trade"
	fTieredAdjustmentFactor    = "/api/v1/contract_adjustfactor"
	fHisContractOpenInterest   = "/api/v1/contract_his_open_interest"
	fSystemStatus              = "/api/v1/contract_api_state"
	fTopAccountsSentiment      = "/api/v1/contract_elite_account_ratio"
	fTopPositionsSentiment     = "/api/v1/contract_elite_position_ratio"
	fLiquidationOrders         = "/api/v3/contract_liquidation_orders"
	fIndexKline                = "/index/market/history/index"
	fBasisData                 = "/index/market/history/basis"

	// Auth
	fAccountData               = "/api/v1/contract_account_info"
	fPositionInformation       = "/api/v1/contract_position_info"
	fAllSubAccountAssets       = "/api/v1/contract_sub_account_list"
	fSingleSubAccountAssets    = "/api/v1/contract_sub_account_info"
	fSingleSubAccountPositions = "/api/v1/contract_sub_position_info"
	fFinancialRecords          = "/api/v1/contract_financial_record"
	fSettlementRecords         = "/api/v1/contract_user_settlement_records"
	fOrderLimitInfo            = "/api/v1/contract_order_limit"
	fContractTradingFee        = "/api/v1/contract_fee"
	fTransferLimitInfo         = "/api/v1/contract_transfer_limit"
	fPositionLimitInfo         = "/api/v1/contract_position_limit"
	fQueryAssetsAndPositions   = "/api/v1/contract_account_position_info"
	fTransfer                  = "/api/v1/contract_master_sub_transfer"
	fTransferRecords           = "/api/v1/contract_master_sub_transfer_record"
	fAvailableLeverage         = "/api/v1/contract_available_level_rate"
	fOrder                     = "/api/v1/contract_order"
	fBatchOrder                = "/api/v1/contract_batchorder"
	fCancelOrder               = "/api/v1/contract_cancel"
	fCancelAllOrders           = "/api/v1/contract_cancelall"
	fFlashCloseOrder           = "/api/v1/lightning_close_position"
	fOrderInfo                 = "/api/v1/contract_order_info"
	fOrderDetails              = "/api/v1/contract_order_detail"
	fQueryOpenOrders           = "/api/v1/contract_openorders"
	fOrderHistory              = "/api/v1/contract_hisorders"
	fMatchResult               = "/api/v1/contract_matchresults"
	fTriggerOrder              = "/api/v1/contract_trigger_order"
	fCancelTriggerOrder        = "/api/v1/contract_trigger_cancel"
	fCancelAllTriggerOrders    = "/api/v1/contract_trigger_cancelall"
	fTriggerOpenOrders         = "/api/v1/contract_trigger_openorders"
	fTriggerOrderHistory       = "/api/v1/contract_trigger_hisorders"

	uContractOpenInterest = "/linear-swap-api/v1/swap_open_interest"
)

var (
	errInvalidContractType        = errors.New("invalid contract type")
	errInconsistentContractExpiry = errors.New("inconsistent contract expiry date codes")
)

// FGetContractInfo gets contract info for futures
func (h *HUOBI) FGetContractInfo(ctx context.Context, symbol, contractType string, code currency.Pair) (FContractInfoData, error) {
	var resp FContractInfoData
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
		codeValue, err := h.FormatSymbol(code, asset.Futures)
		if err != nil {
			return resp, err
		}
		params.Set("contract_code", codeValue)
	}
	path := common.EncodeURLValues(fContractInfo, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// FIndexPriceInfo gets index price info for a futures contract
func (h *HUOBI) FIndexPriceInfo(ctx context.Context, symbol currency.Code) (FContractIndexPriceInfo, error) {
	var resp FContractIndexPriceInfo
	params := url.Values{}
	if !symbol.IsEmpty() {
		codeValue, err := h.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", codeValue)
	}
	path := common.EncodeURLValues(fContractIndexPrice, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// FContractPriceLimitations gets price limits for a futures contract
func (h *HUOBI) FContractPriceLimitations(ctx context.Context, symbol, contractType string, code currency.Pair) (FContractIndexPriceInfo, error) {
	var resp FContractIndexPriceInfo
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
		codeValue, err := h.FormatSymbol(code, asset.Futures)
		if err != nil {
			return resp, err
		}
		params.Set("contract_code", codeValue)
	}
	path := common.EncodeURLValues(fContractPriceLimitation, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// ContractOpenInterestUSDT gets open interest data for futures contracts
func (h *HUOBI) ContractOpenInterestUSDT(ctx context.Context, contractCode, pair currency.Pair, contractType, businessType string) ([]UContractOpenInterest, error) {
	params := url.Values{}
	if !contractCode.IsEmpty() {
		cc, err := h.formatFuturesPair(contractCode, true)
		if err != nil {
			return nil, err
		}
		params.Set("contract_code", cc)
	}
	if !pair.IsEmpty() {
		p, err := h.formatFuturesPair(pair, true)
		if err != nil {
			return nil, err
		}
		params.Set("pair", p)
	}
	if t := strings.ToLower(contractType); t != "" {
		if _, ok := contractExpiryNames[t]; !ok {
			return nil, fmt.Errorf("%w: %v", errInvalidContractType, t)
		}
		params.Set("contract_type", t)
	}
	if businessType != "" {
		params.Set("business_type", businessType)
	}
	path := common.EncodeURLValues(uContractOpenInterest, params)
	var resp struct {
		Data []UContractOpenInterest `json:"data"`
	}
	return resp.Data, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// FContractOpenInterest gets open interest data for futures contracts
func (h *HUOBI) FContractOpenInterest(ctx context.Context, symbol, contractType string, code currency.Pair) (FContractOIData, error) {
	var resp FContractOIData
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
		codeValue, err := h.formatFuturesPair(code, true)
		if err != nil {
			return resp, err
		}
		params.Set("contract_code", codeValue)
	}
	path := common.EncodeURLValues(fContractOpenInterest, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// FGetEstimatedDeliveryPrice gets estimated delivery price info for futures
func (h *HUOBI) FGetEstimatedDeliveryPrice(ctx context.Context, symbol currency.Code) (FEstimatedDeliveryPriceInfo, error) {
	var resp FEstimatedDeliveryPriceInfo
	params := url.Values{}
	codeValue, err := h.formatFuturesCode(symbol)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", codeValue)
	path := common.EncodeURLValues(fEstimatedDeliveryPrice, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// FGetMarketDepth gets market depth data for futures contracts
func (h *HUOBI) FGetMarketDepth(ctx context.Context, symbol currency.Pair, dataType string) (*OBData, error) {
	symbolValue, err := h.formatFuturesPair(symbol, false)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbolValue)
	params.Set("type", dataType)
	path := common.EncodeURLValues(fContractMarketDepth, params)

	var tempData FMarketDepth
	err = h.SendHTTPRequest(ctx, exchange.RestFutures, path, &tempData)
	if err != nil {
		return nil, err
	}

	resp := OBData{
		Symbol: symbolValue,
		Bids:   make([]obItem, len(tempData.Tick.Bids)),
		Asks:   make([]obItem, len(tempData.Tick.Asks)),
	}
	resp.Symbol = symbolValue
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

// FGetKlineData gets kline data for futures
func (h *HUOBI) FGetKlineData(ctx context.Context, symbol currency.Pair, period string, size int64, startTime, endTime time.Time) (FKlineData, error) {
	var resp FKlineData
	params := url.Values{}
	symbolValue, err := h.formatFuturesPair(symbol, false)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if !common.StringSliceCompareInsensitive(validFuturesPeriods, period) {
		return resp, errors.New("invalid period value received")
	}
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
	path := common.EncodeURLValues(fContractKline, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// FGetMarketOverviewData gets market overview data for futures
func (h *HUOBI) FGetMarketOverviewData(ctx context.Context, symbol currency.Pair) (FMarketOverviewData, error) {
	var resp FMarketOverviewData
	params := url.Values{}
	symbolValue, err := h.formatFuturesPair(symbol, false)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	path := common.EncodeURLValues(fMarketOverview, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// FLastTradeData gets last trade data for a futures contract
func (h *HUOBI) FLastTradeData(ctx context.Context, symbol currency.Pair) (FLastTradeData, error) {
	var resp FLastTradeData
	params := url.Values{}
	symbolValue, err := h.formatFuturesPair(symbol, false)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	path := common.EncodeURLValues(fLastTradeContract, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// FRequestPublicBatchTrades gets public batch trades for a futures contract
func (h *HUOBI) FRequestPublicBatchTrades(ctx context.Context, symbol currency.Pair, size int64) (FBatchTradesForContractData, error) {
	params := url.Values{}
	symbolValue, err := h.formatFuturesPair(symbol, false)
	if err != nil {
		return FBatchTradesForContractData{}, err
	}
	params.Set("symbol", symbolValue)
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	var resp FBatchTradesForContractData
	path := common.EncodeURLValues(fContractBatchTradeRecords, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// FQueryTieredAdjustmentFactor gets tiered adjustment factor for futures contracts
func (h *HUOBI) FQueryTieredAdjustmentFactor(ctx context.Context, symbol currency.Code) (FTieredAdjustmentFactorInfo, error) {
	var resp FTieredAdjustmentFactorInfo
	params := url.Values{}
	if !symbol.IsEmpty() {
		codeValue, err := h.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", codeValue)
	}
	path := common.EncodeURLValues(fTieredAdjustmentFactor, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// FQueryHisOpenInterest gets open interest for futures contract
func (h *HUOBI) FQueryHisOpenInterest(ctx context.Context, symbol, contractType, period, amountType string, size int64) (FOIData, error) {
	var resp FOIData
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
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	if size > 0 || size <= 200 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	validAmount, ok := validAmountType[amountType]
	if !ok {
		return resp, errors.New("invalid amountType")
	}
	params.Set("amount_type", strconv.FormatInt(validAmount, 10))
	path := common.EncodeURLValues(fHisContractOpenInterest, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// FQuerySystemStatus gets system status data
func (h *HUOBI) FQuerySystemStatus(ctx context.Context, symbol currency.Code) (FContractOIData, error) {
	var resp FContractOIData
	params := url.Values{}
	if !symbol.IsEmpty() {
		codeValue, err := h.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", codeValue)
	}
	path := common.EncodeURLValues(fSystemStatus, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// FQueryTopAccountsRatio gets top accounts' ratio
func (h *HUOBI) FQueryTopAccountsRatio(ctx context.Context, symbol, period string) (FTopAccountsLongShortRatio, error) {
	var resp FTopAccountsLongShortRatio
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	path := common.EncodeURLValues(fTopAccountsSentiment, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// FQueryTopPositionsRatio gets top positions' long/short ratio for futures
func (h *HUOBI) FQueryTopPositionsRatio(ctx context.Context, symbol, period string) (FTopPositionsLongShortRatio, error) {
	var resp FTopPositionsLongShortRatio
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if !common.StringSliceCompareInsensitive(validPeriods, period) {
		return resp, errors.New("invalid period")
	}
	params.Set("period", period)
	path := common.EncodeURLValues(fTopPositionsSentiment, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// FLiquidationOrders gets liquidation orders for futures contracts
func (h *HUOBI) FLiquidationOrders(ctx context.Context, symbol currency.Code, tradeType string, startTime, endTime int64, direction string, fromID int64) (LiquidationOrdersData, error) {
	var resp LiquidationOrdersData
	tType, ok := validTradeTypes[tradeType]
	if !ok {
		return resp, errors.New("invalid trade type")
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	params.Set("trade_type", strconv.FormatInt(tType, 10))

	if startTime != 0 {
		params.Set("start_time", strconv.FormatInt(startTime, 10))
	}
	if endTime != 0 {
		params.Set("end_time", strconv.FormatInt(startTime, 10))
	}
	if direction != "" {
		params.Set("direct", direction)
	}
	if fromID != 0 {
		params.Set("from_id", strconv.FormatInt(fromID, 10))
	}
	path := common.EncodeURLValues(fLiquidationOrders, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// FIndexKline gets index kline data for futures contracts
func (h *HUOBI) FIndexKline(ctx context.Context, symbol currency.Pair, period string, size int64) (FIndexKlineData, error) {
	var resp FIndexKlineData
	params := url.Values{}
	symbolValue, err := h.formatFuturesPair(symbol, false)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if !common.StringSliceCompareInsensitive(validFuturesPeriods, period) {
		return resp, errors.New("invalid period value received")
	}
	params.Set("period", period)
	if size <= 0 || size > 2000 {
		return resp, errors.New("invalid size")
	}
	params.Set("size", strconv.FormatInt(size, 10))
	path := common.EncodeURLValues(fIndexKline, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// FGetBasisData gets basis data futures contracts
func (h *HUOBI) FGetBasisData(ctx context.Context, symbol currency.Pair, period, basisPriceType string, size int64) (FBasisData, error) {
	var resp FBasisData
	params := url.Values{}
	symbolValue, err := h.formatFuturesPair(symbol, false)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if !common.StringSliceCompareInsensitive(validFuturesPeriods, period) {
		return resp, errors.New("invalid period value received")
	}
	params.Set("period", period)
	if basisPriceType != "" {
		if common.StringSliceCompareInsensitive(validBasisPriceTypes, basisPriceType) {
			params.Set("basis_price_type", basisPriceType)
		}
	}
	if size > 0 && size <= 2000 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	path := common.EncodeURLValues(fBasisData, params)
	return resp, h.SendHTTPRequest(ctx, exchange.RestFutures, path, &resp)
}

// FGetAccountInfo gets user info for futures account
func (h *HUOBI) FGetAccountInfo(ctx context.Context, symbol currency.Code) (FUserAccountData, error) {
	var resp FUserAccountData
	req := make(map[string]any)
	if !symbol.IsEmpty() {
		codeValue, err := h.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		req["symbol"] = codeValue
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fAccountData, nil, req, &resp)
}

// FGetPositionsInfo gets positions info for futures account
func (h *HUOBI) FGetPositionsInfo(ctx context.Context, symbol currency.Code) (FUsersPositionsInfo, error) {
	var resp FUsersPositionsInfo
	req := make(map[string]any)
	if !symbol.IsEmpty() {
		codeValue, err := h.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		req["symbol"] = codeValue
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fPositionInformation, nil, req, &resp)
}

// FGetAllSubAccountAssets gets assets info for all futures subaccounts
func (h *HUOBI) FGetAllSubAccountAssets(ctx context.Context, symbol currency.Code) (FSubAccountAssetsInfo, error) {
	var resp FSubAccountAssetsInfo
	req := make(map[string]any)
	if !symbol.IsEmpty() {
		codeValue, err := h.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		req["symbol"] = codeValue
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fAllSubAccountAssets, nil, req, &resp)
}

// FGetSingleSubAccountInfo gets assets info for a futures subaccount
func (h *HUOBI) FGetSingleSubAccountInfo(ctx context.Context, symbol, subUID string) (FSingleSubAccountAssetsInfo, error) {
	var resp FSingleSubAccountAssetsInfo
	req := make(map[string]any)
	if symbol != "" {
		req["symbol"] = symbol
	}
	req["sub_uid"] = subUID
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fSingleSubAccountAssets, nil, req, &resp)
}

// FGetSingleSubPositions gets positions info for a single sub account
func (h *HUOBI) FGetSingleSubPositions(ctx context.Context, symbol, subUID string) (FSingleSubAccountPositionsInfo, error) {
	var resp FSingleSubAccountPositionsInfo
	req := make(map[string]any)
	if symbol != "" {
		req["symbol"] = symbol
	}
	req["sub_uid"] = subUID
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fSingleSubAccountPositions, nil, req, &resp)
}

// FGetFinancialRecords gets financial records for futures
func (h *HUOBI) FGetFinancialRecords(ctx context.Context, symbol, recordType string, createDate, pageIndex, pageSize int64) (FFinancialRecords, error) {
	var resp FFinancialRecords
	req := make(map[string]any)
	if symbol != "" {
		req["symbol"] = symbol
	}
	if recordType != "" {
		rType, ok := validFuturesRecordTypes[recordType]
		if !ok {
			return resp, errors.New("invalid recordType")
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
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fFinancialRecords, nil, req, &resp)
}

// FGetSettlementRecords gets settlement records for futures
func (h *HUOBI) FGetSettlementRecords(ctx context.Context, symbol currency.Code, pageIndex, pageSize int64, startTime, endTime time.Time) (FSettlementRecords, error) {
	var resp FSettlementRecords
	req := make(map[string]any)
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
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fSettlementRecords, nil, req, &resp)
}

// FGetOrderLimits gets order limits for futures contracts
func (h *HUOBI) FGetOrderLimits(ctx context.Context, symbol, orderPriceType string) (FContractInfoOnOrderLimit, error) {
	var resp FContractInfoOnOrderLimit
	req := make(map[string]any)
	if symbol != "" {
		req["symbol"] = symbol
	}
	if orderPriceType != "" {
		if !common.StringSliceCompareInsensitive(validFuturesOrderPriceTypes, orderPriceType) {
			return resp, errors.New("invalid orderPriceType")
		}
		req["order_price_type"] = orderPriceType
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fOrderLimitInfo, nil, req, &resp)
}

// FContractTradingFee gets futures contract trading fees
func (h *HUOBI) FContractTradingFee(ctx context.Context, symbol currency.Code) (FContractTradingFeeData, error) {
	var resp FContractTradingFeeData
	req := make(map[string]any)
	if !symbol.IsEmpty() {
		codeValue, err := h.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		req["symbol"] = codeValue
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fContractTradingFee, nil, req, &resp)
}

// FGetTransferLimits gets transfer limits for futures
func (h *HUOBI) FGetTransferLimits(ctx context.Context, symbol currency.Code) (FTransferLimitData, error) {
	var resp FTransferLimitData
	req := make(map[string]any)
	if !symbol.IsEmpty() {
		codeValue, err := h.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		req["symbol"] = codeValue
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fTransferLimitInfo, nil, req, &resp)
}

// FGetPositionLimits gets position limits for futures
func (h *HUOBI) FGetPositionLimits(ctx context.Context, symbol currency.Code) (FPositionLimitData, error) {
	var resp FPositionLimitData
	req := make(map[string]any)
	if !symbol.IsEmpty() {
		codeValue, err := h.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		req["symbol"] = codeValue
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fPositionLimitInfo, nil, req, &resp)
}

// FGetAssetsAndPositions gets assets and positions for futures
func (h *HUOBI) FGetAssetsAndPositions(ctx context.Context, symbol currency.Code) (FAssetsAndPositionsData, error) {
	var resp FAssetsAndPositionsData
	req := make(map[string]any)
	req["symbol"] = symbol
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fQueryAssetsAndPositions, nil, req, &resp)
}

// FTransfer transfers assets between master and subaccounts
func (h *HUOBI) FTransfer(ctx context.Context, subUID, symbol, transferType string, amount float64) (FAccountTransferData, error) {
	var resp FAccountTransferData
	req := make(map[string]any)
	req["symbol"] = symbol
	req["subUid"] = subUID
	req["amount"] = amount
	if !common.StringSliceCompareInsensitive(validTransferType, transferType) {
		return resp, errors.New("invalid transferType received")
	}
	req["type"] = transferType
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fTransfer, nil, req, &resp)
}

// FGetTransferRecords gets transfer records data for futures
func (h *HUOBI) FGetTransferRecords(ctx context.Context, symbol, transferType string, createDate, pageIndex, pageSize int64) (FTransferRecords, error) {
	var resp FTransferRecords
	req := make(map[string]any)
	if symbol != "" {
		req["symbol"] = symbol
	}
	if !common.StringSliceCompareInsensitive(validTransferType, transferType) {
		return resp, errors.New("invalid transferType received")
	}
	req["type"] = transferType
	if createDate < 0 || createDate > 90 {
		return resp, errors.New("invalid create date value: only supports up to 90 days")
	}
	req["create_date"] = strconv.FormatInt(createDate, 10)
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize > 0 && pageSize <= 50 {
		req["page_size"] = pageSize
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fTransferRecords, nil, req, &resp)
}

// FGetAvailableLeverage gets available leverage data for futures
func (h *HUOBI) FGetAvailableLeverage(ctx context.Context, symbol currency.Code) (FAvailableLeverageData, error) {
	var resp FAvailableLeverageData
	req := make(map[string]any)
	if !symbol.IsEmpty() {
		codeValue, err := h.formatFuturesCode(symbol)
		if err != nil {
			return resp, err
		}
		req["symbol"] = codeValue
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fAvailableLeverage, nil, req, &resp)
}

// FOrder places an order for futures
func (h *HUOBI) FOrder(ctx context.Context, contractCode currency.Pair, symbol, contractType, clientOrderID, direction, offset, orderPriceType string, price, volume, leverageRate float64) (FOrderData, error) {
	var resp FOrderData
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
		codeValue, err := h.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	if clientOrderID != "" {
		// Client order id is an integer, convert it here
		// https://huobiapi.github.io/docs/dm/v1/en/#place-an-order
		id, err := strconv.Atoi(clientOrderID)
		if err != nil {
			return resp,
				fmt.Errorf("unable to convert client order id to integer, %s: %w", clientOrderID, err)
		}
		req["client_order_id"] = id
	}
	req["direction"] = direction
	if !common.StringSliceCompareInsensitive(validOffsetTypes, offset) {
		return resp, errors.New("invalid offset amounts")
	}
	if !common.StringSliceCompareInsensitive(validFuturesOrderPriceTypes, orderPriceType) {
		return resp, errors.New("invalid orderPriceType")
	}
	req["order_price_type"] = orderPriceType
	req["lever_rate"] = leverageRate
	req["volume"] = volume
	req["price"] = price
	req["offset"] = offset
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fOrder, nil, req, &resp)
}

// FPlaceBatchOrder places a batch of orders for futures
func (h *HUOBI) FPlaceBatchOrder(ctx context.Context, data []fBatchOrderData) (FBatchOrderResponse, error) {
	var resp FBatchOrderResponse
	req := make(map[string]any)
	if len(data) > 10 || len(data) == 0 {
		return resp, errors.New("invalid data provided: maximum of 10 batch orders supported")
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
			if _, ok := contractExpiryNames[strings.ToLower(data[x].ContractType)]; !ok {
				return resp, fmt.Errorf("%w %v", errInvalidContractType, data[x].ContractType)
			}
		}
		if !common.StringSliceCompareInsensitive(validOffsetTypes, data[x].Offset) {
			return resp, errors.New("invalid offset amounts")
		}
		if !common.StringSliceCompareInsensitive(validFuturesOrderPriceTypes, data[x].OrderPriceType) {
			return resp, errors.New("invalid orderPriceType")
		}
	}
	req["orders_data"] = data
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fBatchOrder, nil, req, &resp)
}

// FCancelOrder cancels a futures order
func (h *HUOBI) FCancelOrder(ctx context.Context, baseCurrency currency.Code, orderID, clientOrderID string) (FCancelOrderData, error) {
	var resp FCancelOrderData
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
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fCancelOrder, nil, req, &resp)
}

// FCancelAllOrders cancels all futures order for a given symbol
func (h *HUOBI) FCancelAllOrders(ctx context.Context, contractCode currency.Pair, symbol, contractType string) (FCancelOrderData, error) {
	var resp FCancelOrderData
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
		codeValue, err := h.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fCancelAllOrders, nil, req, &resp)
}

// FFlashCloseOrder flash closes a futures order
func (h *HUOBI) FFlashCloseOrder(ctx context.Context, contractCode currency.Pair, symbol, contractType, direction, orderPriceType, clientOrderID string, volume float64) (FOrderData, error) {
	var resp FOrderData
	req := make(map[string]any)
	req["symbol"] = symbol
	if t := strings.ToLower(contractType); t != "" {
		if _, ok := contractExpiryNames[t]; !ok {
			return resp, fmt.Errorf("%w: %v", errInvalidContractType, t)
		}
		req["contract_type"] = t
	}
	if !contractCode.IsEmpty() {
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
		if !common.StringSliceCompareInsensitive(validOPTypes, orderPriceType) {
			return resp, errors.New("invalid orderPriceType")
		}
		req["orderPriceType"] = orderPriceType
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fFlashCloseOrder, nil, req, &resp)
}

// FGetOrderInfo gets order info for futures
func (h *HUOBI) FGetOrderInfo(ctx context.Context, symbol, clientOrderID, orderID string) (FOrderInfo, error) {
	var resp FOrderInfo
	req := make(map[string]any)
	req["symbol"] = symbol
	if orderID != "" {
		req["order_id"] = orderID
	}
	if clientOrderID != "" {
		req["client_order_id"] = clientOrderID
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fOrderInfo, nil, req, &resp)
}

// FOrderDetails gets order details for futures orders
func (h *HUOBI) FOrderDetails(ctx context.Context, symbol, orderID, orderType string, createdAt time.Time, pageIndex, pageSize int64) (FOrderDetailsData, error) {
	var resp FOrderDetailsData
	req := make(map[string]any)
	req["symbol"] = symbol
	req["order_id"] = orderID
	req["created_at"] = strconv.FormatInt(createdAt.Unix(), 10)
	oType, ok := validOrderType[orderType]
	if !ok {
		return resp, errors.New("invalid orderType")
	}
	req["order_type"] = oType
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fOrderDetails, nil, req, &resp)
}

// FGetOpenOrders gets order details for futures orders
func (h *HUOBI) FGetOpenOrders(ctx context.Context, symbol currency.Code, pageIndex, pageSize int64) (FOpenOrdersData, error) {
	var resp FOpenOrdersData
	req := make(map[string]any)
	req["symbol"] = symbol
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fQueryOpenOrders, nil, req, &resp)
}

// FGetOrderHistory gets order history for futures
func (h *HUOBI) FGetOrderHistory(ctx context.Context, contractCode currency.Pair, symbol, tradeType, reqType, orderType string, status []order.Status, createDate, pageIndex, pageSize int64) (FOrderHistoryData, error) {
	var resp FOrderHistoryData
	req := make(map[string]any)
	req["symbol"] = symbol
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
	if !contractCode.IsEmpty() {
		codeValue, err := h.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	if orderType != "" {
		oType, ok := validFuturesOrderTypes[orderType]
		if !ok {
			return resp, errors.New("invalid orderType")
		}
		req["order_type"] = oType
	}
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fOrderHistory, nil, req, &resp)
}

// FTradeHistory gets trade history data for futures
func (h *HUOBI) FTradeHistory(ctx context.Context, contractCode currency.Pair, symbol, tradeType string, createDate, pageIndex, pageSize int64) (FOrderHistoryData, error) {
	var resp FOrderHistoryData
	req := make(map[string]any)
	req["symbol"] = symbol
	tType, ok := validTradeType[tradeType]
	if !ok {
		return resp, errors.New("invalid tradeType")
	}
	req["trade_type"] = tType
	if !contractCode.IsEmpty() {
		codeValue, err := h.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	if createDate <= 0 || createDate > 90 {
		return resp, errors.New("invalid createDate")
	}
	req["create_date"] = createDate
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fMatchResult, nil, req, &resp)
}

// FPlaceTriggerOrder places a trigger order for futures
func (h *HUOBI) FPlaceTriggerOrder(ctx context.Context, contractCode currency.Pair, symbol, contractType, triggerType, orderPriceType, direction, offset string, triggerPrice, orderPrice, volume, leverageRate float64) (FTriggerOrderData, error) {
	var resp FTriggerOrderData
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
		codeValue, err := h.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	tType, ok := validTriggerType[triggerType]
	if !ok {
		return resp, errors.New("invalid trigger type")
	}
	req["trigger_type"] = tType
	req["direction"] = direction
	if !common.StringSliceCompareInsensitive(validOffsetTypes, offset) {
		return resp, errors.New("invalid offset")
	}
	req["offset"] = offset
	req["trigger_price"] = triggerPrice
	req["volume"] = volume
	req["lever_rate"] = leverageRate
	req["order_price"] = orderPrice
	if !common.StringSliceCompareInsensitive(validOrderPriceType, orderPriceType) {
		return resp, errors.New("invalid order price type")
	}
	req["order_price_type"] = orderPriceType
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fTriggerOrder, nil, req, &resp)
}

// FCancelTriggerOrder cancels trigger order for futures
func (h *HUOBI) FCancelTriggerOrder(ctx context.Context, symbol, orderID string) (FCancelOrderData, error) {
	var resp FCancelOrderData
	req := make(map[string]any)
	req["symbol"] = symbol
	req["order_id"] = orderID
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fCancelTriggerOrder, nil, req, &resp)
}

// FCancelAllTriggerOrders cancels all trigger order for futures
func (h *HUOBI) FCancelAllTriggerOrders(ctx context.Context, contractCode currency.Pair, symbol, contractType string) (FCancelOrderData, error) {
	var resp FCancelOrderData
	req := make(map[string]any)
	req["symbol"] = symbol
	if !contractCode.IsEmpty() {
		codeValue, err := h.FormatSymbol(contractCode, asset.Futures)
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
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fCancelAllTriggerOrders, nil, req, &resp)
}

// FQueryTriggerOpenOrders queries open trigger orders for futures
func (h *HUOBI) FQueryTriggerOpenOrders(ctx context.Context, contractCode currency.Pair, symbol string, pageIndex, pageSize int64) (FTriggerOpenOrders, error) {
	var resp FTriggerOpenOrders
	req := make(map[string]any)
	req["symbol"] = symbol
	if !contractCode.IsEmpty() {
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
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fTriggerOpenOrders, nil, req, &resp)
}

// FQueryTriggerOrderHistory queries trigger order history for futures
func (h *HUOBI) FQueryTriggerOrderHistory(ctx context.Context, contractCode currency.Pair, symbol, tradeType, status string, createDate, pageIndex, pageSize int64) (FTriggerOrderHistoryData, error) {
	var resp FTriggerOrderHistoryData
	req := make(map[string]any)
	req["symbol"] = symbol
	if !contractCode.IsEmpty() {
		codeValue, err := h.FormatSymbol(contractCode, asset.Futures)
		if err != nil {
			return resp, err
		}
		req["contract_code"] = codeValue
	}
	if tradeType != "" {
		tType, ok := validTradeType[tradeType]
		if !ok {
			return resp, errors.New("invalid tradeType")
		}
		req["trade_type"] = tType
	}
	validStatus, ok := validStatusTypes[status]
	if !ok {
		return resp, errors.New("invalid status")
	}
	req["status"] = validStatus
	if createDate <= 0 || createDate > 90 {
		return resp, errors.New("invalid createDate")
	}
	req["create_date"] = createDate
	if pageIndex != 0 {
		req["page_index"] = pageIndex
	}
	if pageSize != 0 {
		req["page_size"] = pageSize
	}
	return resp, h.FuturesAuthenticatedHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, fTriggerOrderHistory, nil, req, &resp)
}

// FuturesAuthenticatedHTTPRequest sends authenticated requests to the HUOBI API
func (h *HUOBI) FuturesAuthenticatedHTTPRequest(ctx context.Context, ep exchange.URL, method, endpoint string, values url.Values, data, result any) error {
	creds, err := h.GetCredentials(ctx)
	if err != nil {
		return err
	}
	ePoint, err := h.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	if ep == exchange.RestFutures && ePoint[len(ePoint)-1] == '/' {
		// prevent signature errors for non-standard paths until we can
		// have a method to force update endpoints
		ePoint = ePoint[:len(ePoint)-1]
	}
	if values == nil {
		values = url.Values{}
	}

	var tempResp json.RawMessage
	newRequest := func() (*request.Item, error) {
		values.Set("AccessKeyId", creds.Key)
		values.Set("SignatureMethod", "HmacSHA256")
		values.Set("SignatureVersion", "2")
		values.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05"))
		sigPath := fmt.Sprintf("%s\napi.hbdm.com\n%s\n%s",
			method, endpoint, values.Encode())
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

		values.Add("Signature", base64.StdEncoding.EncodeToString(hmac))
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
			Method:        method,
			Path:          common.EncodeURLValues(ePoint+endpoint, values),
			Headers:       headers,
			Body:          body,
			Result:        &tempResp,
			Verbose:       h.Verbose,
			HTTPDebugging: h.HTTPDebugging,
			HTTPRecording: h.HTTPRecording,
		}, nil
	}

	err = h.SendPayload(ctx, request.Unset, newRequest, request.AuthenticatedRequest)
	if err != nil {
		return err
	}

	var errCap errorCapture
	if err = json.Unmarshal(tempResp, &errCap); err == nil {
		if errCap.ErrMsgType1 != "" {
			return fmt.Errorf("%w error code: %v error message: %s", request.ErrAuthRequestFailed, errCap.CodeType1, errCap.ErrMsgType1)
		}
		if errCap.ErrMsgType2 != "" {
			return fmt.Errorf("%w error code: %v error message: %s", request.ErrAuthRequestFailed, errCap.CodeType2, errCap.ErrMsgType2)
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

// formatFuturesPair handles pairs in the format as "BTC-NW" and "BTC210827"
func (h *HUOBI) formatFuturesPair(p currency.Pair, convertQuoteToExpiry bool) (string, error) {
	if slices.Contains(validContractExpiryCodes, strings.ToUpper(p.Quote.String())) {
		if convertQuoteToExpiry {
			cp, err := h.pairFromContractExpiryCode(p)
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

	return h.FormatSymbol(p, asset.Futures)
}

// pairFromContractExpiryCode converts a pair with contract expiry shorthand in the Quote to a concrete tradable pair
// We need this because some apis, such as ticker, use BTC_CW, NW, CQ, NQ
// Other apis, such as contract_info, use contract type of this_week, next_week, quarter (sic), and next_quater
func (h *HUOBI) pairFromContractExpiryCode(p currency.Pair) (currency.Pair, error) {
	h.futureContractCodesMutex.RLock()
	defer h.futureContractCodesMutex.RUnlock()
	exp, ok := h.futureContractCodes[p.Quote.String()]
	if !ok {
		return p, fmt.Errorf("%w: %s", errInvalidContractType, p.Quote.String())
	}
	p.Quote = exp
	return p, nil
}
