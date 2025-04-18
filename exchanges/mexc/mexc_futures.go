package mexc

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// GetFuturesContracts retrieves list of detailed futures contract
func (me *MEXC) GetFuturesContracts(ctx context.Context, symbol string) (*FuturesContractsDetail, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *FuturesContractsDetail
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, contractsDetailEPL, http.MethodGet, "contract/detail", params, nil, &resp)
}

// GetTransferableCurrencies returns list of transferabe currencies
func (me *MEXC) GetTransferableCurrencies(ctx context.Context) (*TransferableCurrencies, error) {
	var resp *TransferableCurrencies
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, getTransferableCurrenciesEPL, http.MethodGet, "contract/support_currencies", nil, nil, &resp)
}

// GetContractDepthInformation returns orderbook depth data of a contract
func (me *MEXC) GetContractDepthInformation(ctx context.Context, symbol string, limit int64) (*ContractOrderbook, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *ContractOrderbook
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, getContractDepthInfoEPL, http.MethodGet, "contract/depth/"+symbol, params, nil, &resp)
}

// GetDepthSnapshotOfContract retrieves the order book details and depth information
// for a given contract, filtered by symbol and depth.
func (me *MEXC) GetDepthSnapshotOfContract(ctx context.Context, symbol string, limit int64) (*ContractOrderbookWithDepth, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if limit <= 0 {
		return nil, errLimitIsRequired
	}
	var resp *ContractOrderbookWithDepth
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, getDepthSnapshotOfContractEPL, http.MethodGet, "contract/depth_commits/"+symbol+"/"+strconv.FormatInt(limit, 10), nil, nil, &resp)
}

// GetContractIndexPrice retrieves contract's index price details
func (me *MEXC) GetContractIndexPrice(ctx context.Context, symbol string) (*ContractIndexPriceDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *ContractIndexPriceDetail
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, getContractIndexPriceEPL, http.MethodGet, "contract/index_price/"+symbol, nil, nil, &resp)
}

// GetContractFairPrice retrieves contracts fair price detail
func (me *MEXC) GetContractFairPrice(ctx context.Context, symbol string) (*ContractFairPrice, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *ContractFairPrice
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, getContractFairPriceEPL, http.MethodGet, "contract/fair_price/"+symbol, nil, nil, &resp)
}

// GetContractFundingPrice holds contract's funding price
func (me *MEXC) GetContractFundingPrice(ctx context.Context, symbol string) (*ContractFundingRate, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *ContractFundingRate
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, getContractFundingPriceEPL, http.MethodGet, "contract/funding_rate/"+symbol, nil, nil, &resp)
}

var contractIntervalToStringMap = map[kline.Interval]string{
	kline.OneMin: "Min1", kline.FiveMin: "Min5", kline.FifteenMin: "Min15", kline.ThirtyMin: "Min30",
	kline.OneHour: "Min60", kline.FourHour: "Hour4", kline.EightHour: "Hour8", kline.OneDay: "Day1",
	kline.OneWeek: "Week1", kline.OneMonth: "Month1",
}

// ContractIntervalString returns a string from kline.Interval instance
func ContractIntervalString(interval kline.Interval) (string, error) {
	intervalString, okay := contractIntervalToStringMap[interval]
	if !okay {
		return "", kline.ErrUnsupportedInterval
	}
	return intervalString, nil
}

// GetContractsCandlestickData retrieves futures contracts candlestick data
func (me *MEXC) GetContractsCandlestickData(ctx context.Context, symbol string, interval kline.Interval, startTime, endTime time.Time) (*ContractCandlestickData, error) {
	return me.getCandlestickData(ctx, symbol, "contract/kline/", interval, startTime, endTime)
}

// GetKlineDataOfIndexPrice retrieves kline data of an instrument by index price
func (me *MEXC) GetKlineDataOfIndexPrice(ctx context.Context, symbol string, interval kline.Interval, startTime, endTime time.Time) (*ContractCandlestickData, error) {
	return me.getCandlestickData(ctx, symbol, "contract/kline/index_price/", interval, startTime, endTime)
}

// GetKlineDataOfFairPrice retrieves fair kline price data
func (me *MEXC) GetKlineDataOfFairPrice(ctx context.Context, symbol string, interval kline.Interval, startTime, endTime time.Time) (*ContractCandlestickData, error) {
	return me.getCandlestickData(ctx, symbol, "contract/kline/fair_price/", interval, startTime, endTime)
}

func (me *MEXC) getCandlestickData(ctx context.Context, symbol, path string, interval kline.Interval, startTime, endTime time.Time) (*ContractCandlestickData, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	if interval != 0 {
		intervalString, err := ContractIntervalString(interval)
		if err != nil {
			return nil, err
		}
		params.Set("interval", intervalString)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("start", strconv.FormatInt(startTime.Unix(), 10))
		params.Set("end", strconv.FormatInt(endTime.Unix(), 10))
	}
	var resp *ContractCandlestickData
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, getContractsCandlestickEPL, http.MethodGet, path+symbol, params, nil, &resp)
}

// GetContractTransactionData retrieves contract transaction data
func (me *MEXC) GetContractTransactionData(ctx context.Context, symbol string, limit int64) (*ContractTransactions, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	if limit > 0 {
		params.Set("symbol", symbol)
	}
	var resp *ContractTransactions
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, getContractTransactionEPL, http.MethodGet, "contract/deals/"+symbol, params, nil, &resp)
}

// GetContractTickers holds contract trend data
func (me *MEXC) GetContractTickers(ctx context.Context, symbol string) (*ContractTickers, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *ContractTickers
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, getContractTickersEPL, http.MethodGet, "contract/ticker", params, nil, &resp)
}

// GetAllContractRiskFundBalance holds a list of contracts risk fund balance
func (me *MEXC) GetAllContractRiskFundBalance(ctx context.Context) (*ContractRiskFundBalance, error) {
	var resp *ContractRiskFundBalance
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, getAllContrRiskFundBalanceEPL, http.MethodGet, "contract/risk_reverse", nil, nil, &resp)
}

// GetContractRiskFundBalanceHistory holds a list of contracts risk fund balance history
func (me *MEXC) GetContractRiskFundBalanceHistory(ctx context.Context, symbol string, pageNumber, pageSize int64) (*ContractRiskFundBalanceHistory, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if pageNumber <= 0 {
		return nil, errPageNumberRequired
	}
	if pageSize <= 0 {
		return nil, errPageSizeRequired
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("page_num", strconv.FormatInt(pageNumber, 10))
	params.Set("page_size", strconv.FormatInt(pageSize, 10))
	var resp *ContractRiskFundBalanceHistory
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, contractRiskFundBalanceEPL, http.MethodGet, "contract/risk_reverse/history", params, nil, &resp)
}

// GetContractFundingRateHistory holds contracts funding rate history
func (me *MEXC) GetContractFundingRateHistory(ctx context.Context, symbol string, pageNumber, pageSize int64) (*ContractFundingRateHistory, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if pageNumber > 0 {
		params.Set("page_num", strconv.FormatInt(pageNumber, 10))
	}
	if pageSize > 0 {
		params.Set("page_size", strconv.FormatInt(pageSize, 10))
	}
	var resp *ContractFundingRateHistory
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, contractFundingRateHistoryEPL, http.MethodGet, "contract/funding_rate/history", params, nil, &resp)
}

// GetAllUserAssetsInformation retrieves all user asset balances
func (me *MEXC) GetAllUserAssetsInformation(ctx context.Context) (*UserAssetsBalance, error) {
	var resp *UserAssetsBalance
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, allUserAssetsInfoEPL, http.MethodGet, "private/account/assets", nil, &resp, true)
}

// GetUserSingleCurrencyAssetInformation retrieves user's single asset balance
func (me *MEXC) GetUserSingleCurrencyAssetInformation(ctx context.Context, ccy currency.Code) (*UserAssetBalance, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	resp := &struct {
		Data *UserAssetBalance `json:"data"`
	}{}
	return resp.Data, me.SendHTTPRequest(ctx, exchange.RestFutures, userSingleCurrencyAssetInfoEPL, http.MethodGet, "private/account/asset/"+ccy.String(), nil, &resp, true)
}

// GetUserAssetTransferRecords retrieves user's asset transfer records
// possible values of status are: WAIT, SUCCESS, and FAILED
func (me *MEXC) GetUserAssetTransferRecords(ctx context.Context, ccy currency.Code, status, transferType string, pageNumber, pageSize int64) (*AssetTransfers, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	if status != "" {
		params.Set("state", status)
	}
	if transferType != "" {
		params.Set("type", transferType)
	}
	if pageNumber > 0 {
		params.Set("page_num", strconv.FormatInt(pageNumber, 10))
	}
	if pageSize > 0 {
		params.Set("page_size", strconv.FormatInt(pageSize, 10))
	}
	var resp *AssetTransfers
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, userAssetTransferRecordsEPL, http.MethodGet, "private/account/transfer_record", params, nil, &resp)
}

// GetUserPositionHistory retrieves the user's position history.
// Possible position type values are:
// - '1' for long positions
// - '2' for short positions.
func (me *MEXC) GetUserPositionHistory(ctx context.Context, symbol, positionType string, pageNumber, pageSize int64) (*Positions, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if positionType != "" {
		params.Set("type", positionType)
	}
	if pageNumber > 0 {
		params.Set("page_num", strconv.FormatInt(pageNumber, 10))
	}
	if pageSize > 0 {
		params.Set("page_size", strconv.FormatInt(pageSize, 10))
	}
	var resp *Positions
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, userPositionHistoryEPL, http.MethodGet, "private/position/list/history_positions", params, nil, &resp, true)
}

// GetUsersCurrentHoldingPositions retrieves user's current holding positions
func (me *MEXC) GetUsersCurrentHoldingPositions(ctx context.Context, symbol string) (*Positions, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *Positions
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, usersCurrentHoldingPositionsEPL, http.MethodGet, "private/position/open_positions", params, nil, &resp, true)
}

// GetUsersFundingRateDetails retrieves user's funding rate details
func (me *MEXC) GetUsersFundingRateDetails(ctx context.Context, symbol string, positionID, pageNumber, pageSize int64) (interface{}, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if positionID != 0 {
		params.Set("position_id", strconv.FormatInt(positionID, 10))
	}
	if pageNumber != 0 {
		params.Set("page_num", strconv.FormatInt(pageNumber, 10))
	}
	if pageSize != 0 {
		params.Set("page_size", strconv.FormatInt(pageSize, 10))
	}
	var resp *FundingRateHistory
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, usersFundingRateDetailsEPL, http.MethodGet, "private/position/funding_records", params, nil, &resp, true)
}

// GetUserCurrentPendingOrder holds users current pending orders
func (me *MEXC) GetUserCurrentPendingOrder(ctx context.Context, symbol string, pageNumber, pageSize int64) (*FuturesOrders, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	if pageNumber > 0 {
		params.Set("page_num", strconv.FormatInt(pageNumber, 10))
	}
	if pageSize > 0 {
		params.Set("page_size", strconv.FormatInt(pageSize, 10))
	}
	var resp *FuturesOrders
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, userCurrentPendingOrderEPL, http.MethodGet, "private/order/list/open_orders/"+symbol, params, nil, &resp, true)
}

// GetAllUserHistoricalOrders retrieves user all order history
func (me *MEXC) GetAllUserHistoricalOrders(ctx context.Context, symbol, states, category, side string, startTime, endTime time.Time, pageNumber, pageSize int64) (*FuturesOrders, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if states != "" {
		params.Set("states", states)
	}
	if category != "" {
		params.Set("category", category)
	}
	if side != "" {
		params.Set("side", side)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("start_time", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp *FuturesOrders
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, allUserHistoricalOrdersEPL, http.MethodGet, "private/order/list/history_orders", params, nil, &resp, true)
}

// GetOrderBasedOnExternalNumber retrieves a single order using the external order ID and symbol.
func (me *MEXC) GetOrderBasedOnExternalNumber(ctx context.Context, symbol, externalOrderID string) (interface{}, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if externalOrderID == "" {
		return nil, fmt.Errorf("%w: externalOrderID is missing", order.ErrOrderIDNotSet)
	}
	resp := &struct {
		Data *FuturesOrderDetail `json:"data"`
	}{}
	return resp.Data, me.SendHTTPRequest(ctx, exchange.RestFutures, getOrderBasedOnExternalNumberEPL, http.MethodGet, "private/order/external/"+symbol+"/"+externalOrderID, nil, &resp, true)
}

// GetOrderByOrderID retrieves a single order using order id
func (me *MEXC) GetOrderByOrderID(ctx context.Context, orderID string) (*FuturesOrderDetail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	resp := &struct {
		Data *FuturesOrderDetail `json:"data"`
	}{}
	return resp.Data, me.SendHTTPRequest(ctx, exchange.RestFutures, orderByOrderNumberEPL, http.MethodGet, "private/order/get/"+orderID, nil, nil, &resp)
}

// GetBatchOrdersByOrderID retrieves a batch of futures orders by order ids
func (me *MEXC) GetBatchOrdersByOrderID(ctx context.Context, orderIDs []string) (interface{}, error) {
	if len(orderIDs) == 0 {
		return nil, fmt.Errorf("%w: no order ID provided", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("order_ids", strings.Join(orderIDs, ","))
	var resp *FuturesOrders
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, batchOrdersByOrderIDEPL, http.MethodGet, "private/order/batch_query", params, nil, &resp, true)
}

// GetOrderTransactionDetailsByOrderID retrieves an order transactions by order ID
func (me *MEXC) GetOrderTransactionDetailsByOrderID(ctx context.Context, orderID string) (*OrderTransactions, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("order_id", orderID)
	var resp *OrderTransactions
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, orderTransactionDetailsByOrderIDEPL, http.MethodGet, "private/order/deal_details/"+orderID, params, nil, &resp, true)
}

// GetUserOrderAllTransactionDetails retrieves user order all transaction details.
func (me *MEXC) GetUserOrderAllTransactionDetails(ctx context.Context, symbol string, startTime, endTime time.Time, pageNumber, pageSize int64) (*OrderTransactions, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("start_time", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if pageNumber > 0 {
		params.Set("page_num", strconv.FormatInt(pageNumber, 10))
	}
	if pageSize > 0 {
		params.Set("page_size", strconv.FormatInt(pageSize, 10))
	}
	var resp *OrderTransactions
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, userOrderAllTransactionDetailsEPL, http.MethodGet, "private/order/list/order_deals", params, nil, &resp, true)
}

// GetTriggerOrderList retrieves a list of futures trigger orders
func (me *MEXC) GetTriggerOrderList(ctx context.Context, symbol, states string, startTime, endTime time.Time, pageNumber, pageSize int64) (*FuturesTriggerOrders, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if states != "" {
		params.Set("states", states)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("start_time", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if pageNumber > 0 {
		params.Set("page_num", strconv.FormatInt(pageNumber, 10))
	}
	if pageSize > 0 {
		params.Set("page_size", strconv.FormatInt(pageSize, 10))
	}
	var resp *FuturesTriggerOrders
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, triggerOrderListEPL, http.MethodGet, "private/planorder/list/orders", params, nil, &resp, true)
}

// GetFuturesStopLimitOrderList retrieves futures stop limit orders list
func (me *MEXC) GetFuturesStopLimitOrderList(ctx context.Context, symbol string, isFinished bool, startTime, endTime time.Time, pageNumber, pageSize int64) (interface{}, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if isFinished {
		params.Set("is_finished", "1")
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
		params.Set("start_time", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("end_time", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if pageNumber > 0 {
		params.Set("page_num", strconv.FormatInt(pageNumber, 10))
	}
	if pageSize > 0 {
		params.Set("page_size", strconv.FormatInt(pageSize, 10))
	}
	var resp *FuturesStopLimitOrders
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, futuresStopLimitOrderListEPL, http.MethodGet, "private/stoporder/list/orders", params, nil, &resp, true)
}

// GetFuturesRiskLimit retrieves futures symbols risk limits
func (me *MEXC) GetFuturesRiskLimit(ctx context.Context, symbol string) (*FutureRiskLimit, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *FutureRiskLimit
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, futuresRiskLimitEPL, http.MethodGet, "private/account/risk_limit", params, nil, &resp, true)
}

// GetFuturesCurrentTradingFeeRate holds futures current trading fee rates
func (me *MEXC) GetFuturesCurrentTradingFeeRate(ctx context.Context, symbol string) (*FuturesTradingFeeRates, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *FuturesTradingFeeRates
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, futuresCurrentTradingFeeRateEPL, http.MethodGet, "private/account/tiered_fee_rate", params, nil, &resp, true)
}

// IncreaseDecreaseMargin adjusts the margin amount in a futures trading account.
// Possible change type values:
// - 'ADD' to increase the margin
// - 'SUB' to decrease the margin.
func (me *MEXC) IncreaseDecreaseMargin(ctx context.Context, positionID int64, amount float64, changeType string) error {
	if positionID == 0 {
		return fmt.Errorf("%w: positionID is required", order.ErrOrderIDNotSet)
	}
	if amount <= 0 {
		return order.ErrAmountBelowMin
	}
	if changeType == "" {
		return fmt.Errorf("%w: changeType is required", order.ErrTypeIsInvalid)
	}
	params := url.Values{}
	params.Set("positionId", strconv.FormatInt(positionID, 10))
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	params.Set("type", changeType)
	return me.SendHTTPRequest(ctx, exchange.RestFutures, increaseDecreaseMarginEPL, http.MethodPost, "private/position/change_margin", params, nil, true)
}

// GetContractLeverage retrieves leverage information for a contract
func (me *MEXC) GetContractLeverage(ctx context.Context, symbol string) (*ContractLeverageInfo, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *ContractLeverageInfo
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, contractLeverageEPL, http.MethodGet, "private/position/leverage", params, nil, &resp, true)
}

// SwitchLeverage adjusts the leverage of an open position.
// Possible open type values:
// - 1: Isolated position
// - 2: Full position
// Possible position type values:
// - 1: Long position
// - 2: Short position
func (me *MEXC) SwitchLeverage(ctx context.Context, positionID, leverage, openType, positionType int64, symbol string) (*PositionLeverageResponse, error) {
	if positionID == 0 {
		return nil, fmt.Errorf("%w: positionID is required", order.ErrOrderIDNotSet)
	}
	if leverage <= 0 {
		return nil, errMissingLeverage
	}
	params := url.Values{}
	params.Set("positionId", strconv.FormatInt(positionID, 10))
	params.Set("leverage", strconv.FormatInt(leverage, 10))
	if openType != 0 {
		params.Set("openType", strconv.FormatInt(openType, 10))
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if positionType != 0 {
		params.Set("positionType", strconv.FormatInt(positionType, 10))
	}
	resp := &struct {
		Data PositionLeverageResponse `json:"data"`
	}{}
	return &resp.Data, me.SendHTTPRequest(ctx, exchange.RestFutures, switchLeverageEPL, http.MethodPost, "private/position/change_leverage", params, nil, &resp, true)
}

// GetPositionMode retrieves a list of position modes
func (me *MEXC) GetPositionMode(ctx context.Context) (*PositionMode, error) {
	var resp *PositionMode
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, getPositionModeEPL, http.MethodGet, "private/position/position_mode", nil, &resp, true)
}

// ChangePositionMode updates the position mode.
// Possible values:
// - 1: Hedge mode
// - 2: One-way mode
//
// The position mode can only be modified if there are no active orders,
// plan ned orders, or open positions; otherwise, the modification is not allowed.
//
// When switching between One-way and Hedge mode, the risk limit level
// will be reset to Level 1. If you need to change this setting via API, modify the call accordingly.
func (me *MEXC) ChangePositionMode(ctx context.Context, positionMode int64) (*StatusResponse, error) {
	if positionMode == 0 {
		return nil, errPositionModeRequired
	}
	params := url.Values{}
	params.Set("positionMode", strconv.FormatInt(positionMode, 10))
	var resp *StatusResponse
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, changePositionModeEPL, http.MethodPost, "private/position/change_position_mode", params, nil, &resp, true)
}

// PlaceFuturesOrder placed a futures order
func (me *MEXC) PlaceFuturesOrder(ctx context.Context, arg *PlaceFuturesOrderParams) (int64, error) {
	params, err := validateOrderParams(arg)
	if err != nil {
		return 0, err
	}
	var value int64
	resp := &StatusResponse{
		Data: &value,
	}
	return value, me.SendHTTPRequest(ctx, exchange.RestFutures, placeFuturesOrderEPL, http.MethodPost, "private/order/submit", params, nil, &resp, true)
}

func validateOrderParams(arg *PlaceFuturesOrderParams) (url.Values, error) {
	if arg == nil || *arg == (PlaceFuturesOrderParams{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Price <= 0 {
		return nil, order.ErrPriceBelowMin
	}
	if arg.Volume <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("symbol", arg.Symbol)
	params.Set("price", strconv.FormatFloat(arg.Price, 'f', -1, 64))
	params.Set("vol", strconv.FormatFloat(arg.Volume, 'f', -1, 64))
	switch {
	case arg.Side.IsLong():
		params.Set("side", "1")
	case arg.Side.IsShort():
		params.Set("side", "2")
	default:
		return nil, fmt.Errorf("%w: order side is missing", order.ErrSideIsInvalid)
	}
	switch arg.OrderType {
	case order.Market:
		params.Set("type", "5")
	case order.Limit:
		params.Set("type", "1")
	case order.FillOrKill:
		params.Set("type", "4")
	case order.PostOnly:
		params.Set("type", "2")
	case order.ImmediateOrCancel:
		params.Set("type", "3")
	case order.Chase:
		params.Set("type", "6")
	default:
		return nil, fmt.Errorf("%w: type: %v", order.ErrUnsupportedOrderType, arg.OrderType)
	}
	switch arg.MarginType {
	case margin.Isolated:
		params.Set("openType", "1")
	case margin.Multi:
		params.Set("openType", "2")
	default:
		return nil, fmt.Errorf("%w: %v", margin.ErrInvalidMarginType, arg.MarginType)
	}
	if arg.PositionID != 0 {
		params.Set("positionId", strconv.FormatInt(arg.PositionID, 10))
	}
	if arg.ExternalOrderID != "" {
		params.Set("externalOid", arg.ExternalOrderID)
	}
	if arg.StopLossPrice != 0 {
		params.Set("stopLossPrice", strconv.FormatFloat(arg.StopLossPrice, 'f', -1, 64))
	}
	if arg.TakeProfitPrice != 0 {
		params.Set("takeProfitPrice", strconv.FormatFloat(arg.TakeProfitPrice, 'f', -1, 64))
	}
	if arg.PositionMode != 0 {
		params.Set("positionMode", strconv.FormatInt(arg.PositionMode, 10))
	}
	if arg.ReduceOnly {
		params.Set("reduceOnly", "true")
	}
	return params, nil
}

// PostFuturesBatchOrders
func (me *MEXC) PostFuturesBatchOrders(ctx context.Context, args []PlaceFuturesOrderParams) ([]FuturesOrderInfo, error) {
	return nil, nil
}

// func (me *MEXC) CancelOrder(ctx context.Context, )

// TODO: I skipped the place batch orders and cancel order endpoints because of the umbiguity of how the data payload should be sent.

// CancelOrderByClientOrderID cancels a single order by client supplied(external) order ID
func (me *MEXC) CancelOrderByClientOrderID(ctx context.Context, symbol, externalOrderID string) (*OrderCancellationResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if externalOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("externalOid", externalOrderID)
	var resp *OrderCancellationResponse
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, cancelOrderByClientOrderIDEPL, "private/order/cancel_with_external", http.MethodPost, params, nil, &resp, true)
}

// CancelAllOpenOrders cancels all open contracts under this account
func (me *MEXC) CancelAllOpenOrders(ctx context.Context, symbol string) ([]OrderCancellationResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []OrderCancellationResponse
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, cancelAllOpenOrdersEPL, http.MethodPost, "private/order/cancel_all", params, nil, &resp, true)
}

// PlaceFuturesTriggerOrder places a futures trigger order
func (me *MEXC) PlaceFuturesTriggerOrder(ctx context.Context, arg *PlaceFuturesTriggerOrderParams) (interface{}, error) {
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Volume <= 0 {
		return nil, fmt.Errorf("%w: volume is required", order.ErrAmountBelowMin)
	}
	params := url.Values{}
	switch arg.Side {
	case order.Sell, order.Short:
	case order.Buy, order.Long:
		params.Set("side", "1")
	default:
		return nil, fmt.Errorf("%w: orderSide is required", order.ErrSideIsInvalid)
	}
	switch arg.MarginType {
	case margin.Isolated:
	case margin.Multi:
	default:
		return nil, fmt.Errorf("%w: MarginType is required", margin.ErrInvalidMarginType)
	}
	if arg.TriggerPrice <= 0 {
		return nil, fmt.Errorf("%w: TriggerPrice is required", order.ErrPriceBelowMin)
	}
	if arg.TriggerPriceType == 0 {
		return nil, fmt.Errorf("%w: TriggerPriceType is required", order.ErrUnknownPriceType)
	}
	if arg.ExecutionCycle <= 0 {
		// return nil,
	}
	switch arg.OrderType {
	case order.Limit:
		params.Set("orderType", "1")
	case order.PostOnly:
		params.Set("orderType", "2")
	case order.ImmediateOrCancel:
		params.Set("orderType", "3")
	case order.FillOrKill:
		params.Set("orderType", "4")
	case order.Market:
		params.Set("orderType", "5")
	default:
		return nil, fmt.Errorf("%w: OrderType is not supported", order.ErrUnsupportedOrderType)
	}
	switch arg.PriceType {
	case order.LastPrice:
		params.Set("trend", "2")
	case order.MarkPrice:
		params.Set("trend", "2")
	case order.IndexPrice:
		params.Set("trend", "2")
	default:
		return nil, fmt.Errorf("%w: Price Type is required", order.ErrUnknownPriceType)
	}
	var resp interface{}
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, placeFuturesTriggerOrderEPL, http.MethodPost, "private/planorder/place", params, nil, &resp, true)
}

// SwitchStopLimitLimitedOrderPrice toggles stop limig limited order price
func (me *MEXC) SwitchStopLimitLimitedOrderPrice(ctx context.Context, orderID string, stopLossPrice, takeProfitPrice float64) (interface{}, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("orderId", orderID)
	if stopLossPrice > 0 {
		params.Set("stopLossPrice", strconv.FormatFloat(stopLossPrice, 'f', -1, 64))
	}
	if takeProfitPrice > 0 {
		params.Set("takeProfitPrice", strconv.FormatFloat(takeProfitPrice, 'f', -1, 64))
	}
	var resp interface{}
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, switchSLLimitedOrderPriceEPL, http.MethodPost, "private/stoporder/change_price", params, nil, &resp, true)
}

// SwitchStopLimitPriceOfTriggerPrice toggles the stop limit price of trigger price order
func (me *MEXC) SwitchStopLimitPriceOfTriggerPrice(ctx context.Context, stopPlanOrderID string, stopLossPrice, takeProfitPrice float64) (interface{}, error) {
	if stopPlanOrderID == "" {
		return nil, fmt.Errorf("%w: StopPlanOrderID is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("stopPlanOrderId", stopPlanOrderID)
	if stopLossPrice > 0 {
		params.Set("stopLossPrice", strconv.FormatFloat(stopLossPrice, 'f', -1, 64))
	}
	if takeProfitPrice > 0 {
		params.Set("takeProfitPrice", strconv.FormatFloat(takeProfitPrice, 'f', -1, 64))
	}
	var resp interface{}
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, switchSLPriceOfTriggerPriceEPL, http.MethodPost, "private/stoporder/change_plan_price", params, nil, &resp, true)
}

// CancelAllStopLimitPriceTriggerOrders cancels all price triggered stop limit orders given a position ID and contract symbol
func (me *MEXC) CancelAllStopLimitPriceTriggerOrders(ctx context.Context, positionID, symbol string) (interface{}, error) {
	params := url.Values{}
	if positionID != "" {
		params.Set("positionId", positionID)
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp interface{}
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, cancelALlSLPriceTriggerOrdersEPL, http.MethodGet, "private/stoporder/cancel_all", params, nil, &resp, true)
}

// CancelAllTriggerOrders cancels all futures trigger orders
func (me *MEXC) CancelAllTriggerOrders(ctx context.Context, symbol string) ([]OrderIDDetail, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []OrderIDDetail
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, cancelAllTriggerOrdersEPL, http.MethodPost, "private/planorder/cancel_all", params, nil, &resp, true)
}

// CancelStopLimitTriggerOrder cancels a stop limit trigger order by stop plan order ID
// TODO: Under mentainance | incomplete
func (me *MEXC) CancelStopLimitTriggerOrder(ctx context.Context, stopPlanOrderIDs []string) (interface{}, error) {
	params := url.Values{}
	var resp interface{}
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, cancelSLTriggerOrderEPL, http.MethodPost, "private/stoporder/cancel", params, nil, &resp, true)
}
