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
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// GetContractsDetail retrieves list of detailed futures contract
func (me *MEXC) GetContractsDetail(ctx context.Context, symbol string) (*FuturesContractsDetail, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *FuturesContractsDetail
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "contract/detail", params, &resp)
}

// GetTransferableCurrencies returns list of transferabe currencies
func (me *MEXC) GetTransferableCurrencies(ctx context.Context) (*TransferableCurrencies, error) {
	var resp *TransferableCurrencies
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "contract/support_currencies", nil, &resp)
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
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "contract/depth/"+symbol, params, &resp)
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
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "contract/depth_commits/"+symbol+"/"+strconv.FormatInt(limit, 10), nil, &resp)
}

// GetContractIndexPrice retrieves contract's index price details
func (me *MEXC) GetContractIndexPrice(ctx context.Context, symbol string) (*ContractIndexPriceDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *ContractIndexPriceDetail
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "contract/index_price/"+symbol, nil, &resp)
}

// GetContractFairPrice retrieves contracts fair price detail
func (me *MEXC) GetContractFairPrice(ctx context.Context, symbol string) (*ContractFairPrice, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *ContractFairPrice
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "contract/fair_price/"+symbol, nil, &resp)
}

// GetContractFundingPrice holds contract's funding price
func (me *MEXC) GetContractFundingPrice(ctx context.Context, symbol string) (*ContractFundingRate, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *ContractFundingRate
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "contract/funding_rate/"+symbol, nil, &resp)
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
		params.Set("start", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("end", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	var resp *ContractCandlestickData
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.UnAuth, http.MethodGet, path+symbol, params, &resp)
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
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.UnAuth, http.MethodGet, "contract/deals/"+symbol, params, &resp)
}

// GetContractTickers holds contract trend data
func (me *MEXC) GetContractTickers(ctx context.Context, symbol string) (*ContractTickers, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *ContractTickers
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.UnAuth, http.MethodGet, "contract/ticker", params, &resp)
}

// GetAllContractRiskFundBalance holds a list of contracts risk fund balance
func (me *MEXC) GetAllContractRiskFundBalance(ctx context.Context) (*ContractRiskFundBalance, error) {
	var resp *ContractRiskFundBalance
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.UnAuth, http.MethodGet, "contract/risk_reverse", nil, &resp)
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
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.UnAuth, http.MethodGet, "contract/risk_reverse/history", params, &resp)
}

// GetContractFundingRateHistory holds contracts funding rate history
func (me *MEXC) GetContractFundingRateHistory(ctx context.Context, symbol string, pageNumber, pageSize int64) (*ContractFundingRateHistory, error) {
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
	var resp *ContractFundingRateHistory
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.UnAuth, http.MethodGet, "contract/funding_rate/history", params, &resp)
}

// GetAllUserAssetsInformation retrieves all user asset balances
func (me *MEXC) GetAllUserAssetsInformation(ctx context.Context) (*UserAssetsBalance, error) {
	var resp *UserAssetsBalance
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "private/account/assets", nil, &resp, true)
}

// GetUserSingleCurrencyAssetInformation retrieves user's single asset balance
func (me *MEXC) GetUserSingleCurrencyAssetInformation(ctx context.Context, ccy currency.Code) (*UserAssetBalance, error) {
	if ccy.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	resp := &struct {
		Data *UserAssetBalance `json:"data"`
	}{}
	return resp.Data, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "private/account/asset/"+ccy.String(), nil, &resp, true)
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
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "private/account/transfer_record", params, &resp)
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
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "private/position/list/history_positions", params, &resp, true)
}

// GetUsersCurrentHoldingPositions retrieves user's current holding positions
func (me *MEXC) GetUsersCurrentHoldingPositions(ctx context.Context, symbol string) (*Positions, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *Positions
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "private/position/open_positions", params, &resp, true)
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
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "private/position/funding_records", params, &resp, true)
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
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "private/order/list/open_orders/"+symbol, params, &resp, true)
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
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "private/order/list/history_orders", params, &resp, true)
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
	return resp.Data, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "private/order/external/"+symbol+"/"+externalOrderID, nil, &resp, true)
}

// GetOrderByOrderNumber retrieves a single order using order id
func (me *MEXC) GetOrderByOrderNumber(ctx context.Context, orderID string) (*FuturesOrderDetail, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	resp := &struct {
		Data *FuturesOrderDetail `json:"data"`
	}{}
	return resp.Data, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "private/order/get/"+orderID, nil, &resp)
}

// GetBatchOrdersByOrderID retrieves a batch of futures orders by order ids
func (me *MEXC) GetBatchOrdersByOrderID(ctx context.Context, orderIDs []string) (interface{}, error) {
	if len(orderIDs) == 0 {
		return nil, fmt.Errorf("%w: no order ID provided", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("order_ids", strings.Join(orderIDs, ","))
	var resp *FuturesOrders
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "private/order/batch_query", params, &resp, true)
}

// GetOrderTransactionDetailsByOrderID retrieves an order transactions by order ID
func (me *MEXC) GetOrderTransactionDetailsByOrderID(ctx context.Context, orderID string) (*OrderTransactions, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("order_id", orderID)
	var resp *OrderTransactions
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "private/order/deal_details/"+orderID, params, &resp, true)
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
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "private/order/list/order_deals", params, &resp, true)
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
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "private/planorder/list/orders", params, &resp, true)
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
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "private/stoporder/list/orders", params, &resp, true)
}

// GetFuturesRiskLimit retrieves futures symbols risk limits
func (me *MEXC) GetFuturesRiskLimit(ctx context.Context, symbol string) (*FutureRiskLimit, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *FutureRiskLimit
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "private/account/risk_limit", params, &resp, true)
}

// GetFuturesCurrentTradingFeeRate holds futures current trading fee rates
func (me *MEXC) GetFuturesCurrentTradingFeeRate(ctx context.Context, symbol string) (*FuturesTradingFeeRates, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *FuturesTradingFeeRates
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "private/account/tiered_fee_rate", params, &resp, true)
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
	return me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodPost, "private/position/change_margin", params, nil, true)
}

// GetContractLeverage retrieves leverage information for a contract
func (me *MEXC) GetContractLeverage(ctx context.Context, symbol string) (*ContractLeverageInfo, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *ContractLeverageInfo
	return resp, me.SendHTTPRequest(ctx, exchange.RestFutures, request.Auth, http.MethodGet, "private/position/leverage", params, &resp, true)
}
