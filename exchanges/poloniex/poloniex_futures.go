package poloniex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	poloniexFuturesAPIURL = "https://futures-api.poloniex.com"
)

// GetOpenContractList retrieves the info of all open contracts.
func (p *Poloniex) GetOpenContractList(ctx context.Context) (*Contracts, error) {
	var resp *Contracts
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, "/api/v1/contracts/active", &resp)
}

// GetOrderInfoOfTheContract info of the specified contract.
func (p *Poloniex) GetOrderInfoOfTheContract(ctx context.Context, symbol string) (*ContractItem, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *ContractItem
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, "/api/v1/contracts/"+symbol, &resp)
}

// GetRealTimeTicker real-time ticker 1.0 includes the last traded price, the last traded size, transaction ID,
// the side of the liquidity taker, the best bid price and size, the best ask price and size as well as the transaction time of the orders.
func (p *Poloniex) GetRealTimeTicker(ctx context.Context, symbol string) (*TickerDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *TickerDetail
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, "/api/v1/ticker?symbol="+symbol, &resp)
}

// GetFuturesRealTimeTickersOfSymbols retrieves real-time tickers includes tickers of all trading symbols.
func (p *Poloniex) GetFuturesRealTimeTickersOfSymbols(ctx context.Context) (*TickersDetail, error) {
	var resp *TickersDetail
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, "/api/v2/tickers", &resp)
}

// GetFullOrderbookLevel2 retrieves a snapshot of aggregated open orders for a symbol.
// level 2 order book includes all bids and asks (aggregated by price). This level returns only one aggregated size for each price (as if there was only one single order for that price).
func (p *Poloniex) GetFullOrderbookLevel2(ctx context.Context, symbol string) (*Orderbook, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *Orderbook
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, common.EncodeURLValues("/api/v1/level2/snapshot", params), &resp)
}

// GetPartialOrderbookLevel2 represents partial snapshot of aggregated open orders for a symbol.
// depth: depth5, depth10, depth20 , depth30 , depth50 or depth100
func (p *Poloniex) GetPartialOrderbookLevel2(ctx context.Context, symbol, depth string) (*Orderbook, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if depth == "" {
		return nil, errOrderbookDepthRequired
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("depth", depth)
	var resp *Orderbook
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, common.EncodeURLValues("/api/v1/level2/depth", params), &resp)
}

// Level2PullingMessages if the messages pushed by Websocket are not continuous, you can submit the following request and re-pull the data to ensure that the sequence is not missing.
func (p *Poloniex) Level2PullingMessages(ctx context.Context, symbol string, startSequence, endSequence int64) (*OrderbookChanges, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if startSequence <= 0 {
		return nil, fmt.Errorf("%w, start sequence %d", errInvalidSequenceNumber, startSequence)
	}
	if endSequence <= 0 {
		return nil, fmt.Errorf("%w, end sequence %d", errInvalidSequenceNumber, endSequence)
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("start", strconv.FormatInt(startSequence, 10))
	params.Set("end", strconv.FormatInt(endSequence, 10))
	var resp *OrderbookChanges
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, common.EncodeURLValues("/api/v1/level2/message/query", params), &resp)
}

// GetFullOrderBookLevel3 a snapshot of all the open orders for a symbol. The Level 3 order book includes all bids and asks (the data is non-aggregated, and each item means a single order).
// To ensure your local orderbook data is the latest one, please use Websocket incremental feed after retrieving the level 3 snapshot.
func (p *Poloniex) GetFullOrderBookLevel3(ctx context.Context, symbol string) (*Orderbook, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *Orderbook
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, common.EncodeURLValues("/api/v2/level3/snapshot", params), &resp)
}

// Level3PullingMessages If the messages pushed by the Websocket is not continuous, you can submit the following request and re-pull the data to ensure that the sequence is not missing.
func (p *Poloniex) Level3PullingMessages(ctx context.Context) (*Level3PullingMessageResponse, error) {
	var resp *Level3PullingMessageResponse
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, "/api/v1/level2/message/query", &resp)
}

// ----------------------------------------------------   Historical Data  ---------------------------------------------------------------

// GetTransactionHistory list the last 100 trades for a symbol.
func (p *Poloniex) GetTransactionHistory(ctx context.Context, symbol string) (*TransactionHistory, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *TransactionHistory
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, "/api/v1/trade/history?symbol="+symbol, &resp)
}

func (p *Poloniex) populateIndexParams(symbol string, startAt, endAt time.Time, reverse, forward bool, maxCount int64) url.Values {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if reverse {
		params.Set("reverse", "true")
	}
	if forward {
		params.Set("forward", "true")
	}
	if maxCount > 0 {
		params.Set("maxCount", strconv.FormatInt(maxCount, 10))
	}
	return params
}

// GetInterestRateList retrieves interest rate list.
func (p *Poloniex) GetInterestRateList(ctx context.Context, symbol string, startAt, endAt time.Time, reverse, forward bool, maxCount int64) (*IndexInfo, error) {
	params := p.populateIndexParams(symbol, startAt, endAt, reverse, forward, maxCount)
	var resp *IndexInfo
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, common.EncodeURLValues("/api/v1/interest/query", params), &resp)
}

// GetIndexList check index list
func (p *Poloniex) GetIndexList(ctx context.Context, symbol string, startAt, endAt time.Time, reverse, forward bool, maxCount int64) (*IndexInfo, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := p.populateIndexParams(symbol, startAt, endAt, reverse, forward, maxCount)
	var resp *IndexInfo
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, common.EncodeURLValues("/api/v1/index/query", params), &resp)
}

// GetCurrentMarkPrice retrieves the current mark price.
func (p *Poloniex) GetCurrentMarkPrice(ctx context.Context, symbol string) (*MarkPriceDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *MarkPriceDetail
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, "/api/v1/mark-price/"+symbol+"/current", &resp)
}

// GetPremiumIndex request to get premium index.
func (p *Poloniex) GetPremiumIndex(ctx context.Context, symbol string, startAt, endAt time.Time, reverse, forward bool, maxCount int64) (*IndexInfo, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := p.populateIndexParams(symbol, startAt, endAt, reverse, forward, maxCount)
	var resp *IndexInfo
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, unauthEPL, common.EncodeURLValues("/api/v1/premium/query", params), &resp)
}

// GetCurrentFundingRate request to check the current mark price.
func (p *Poloniex) GetCurrentFundingRate(ctx context.Context, symbol string) (*FundingRate, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *FundingRate
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, "/api/v1/funding-rate/"+symbol+"/current", &resp)
}

// GetFuturesServerTime get the API server time. This is the Unix timestamp.
func (p *Poloniex) GetFuturesServerTime(ctx context.Context) (*ServerTimeResponse, error) {
	var resp *ServerTimeResponse
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, "/api/v1/timestamp", &resp)
}

// GetServiceStatus the service status.
func (p *Poloniex) GetServiceStatus(ctx context.Context) (*ServiceStatus, error) {
	var resp *ServiceStatus
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, "/api/v1/status", &resp)
}

// GetFuturesKlineDataOfContract retrieves candlestick information
func (p *Poloniex) GetFuturesKlineDataOfContract(ctx context.Context, symbol string, granularity int64, from, to time.Time) ([]KlineChartData, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if granularity == 0 {
		return nil, errGranularityRequired
	}
	params := url.Values{}
	if !from.IsZero() {
		params.Set("from", strconv.FormatInt(from.UnixMilli(), 10))
	}
	params.Set("symbol", symbol)
	params.Set("granularity", strconv.FormatInt(granularity, 10))
	if !to.IsZero() {
		params.Set("to", strconv.FormatInt(to.UnixMilli(), 10))
	}
	var resp *KlineChartResponse
	err := p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, common.EncodeURLValues("/api/v1/kline/query", params), &resp)
	if err != nil {
		return nil, err
	}
	return resp.ExtractKlineChart(), nil
}

// GetPublicFuturesWebsocketServerInstances retrieves the server list and temporary public token.
func (p *Poloniex) GetPublicFuturesWebsocketServerInstances(ctx context.Context) (*FuturesWebsocketServerInstances, error) {
	var resp *FuturesWebsocketServerInstances
	return resp, p.SendHTTPRequest(ctx, exchange.RestFutures, unauthEPL, "/api/v1/bullet-public", &resp)
}

// GetPrivateFuturesWebsocketServerInstances retrieves authenticated list of servers and temporary token.
func (p *Poloniex) GetPrivateFuturesWebsocketServerInstances(ctx context.Context) (*FuturesWebsocketServerInstances, error) {
	var resp *FuturesWebsocketServerInstances
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, unauthEPL, http.MethodPost, "/api/v1/bullet-private", nil, nil, &resp)
}

// GetFuturesAccountOverview retrieves futures account overview information.
func (p *Poloniex) GetFuturesAccountOverview(ctx context.Context, ccy currency.Code) (*FuturesAccountOverview, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var resp *FuturesAccountOverview
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, accountOverviewEPL, http.MethodGet, "/api/v1/account-overview", params, nil, &resp)
}

// GetFuturesAccountTransactionHistory retrieves the futures account transactions history.
// If there are open positions, the status of the first page returned will be Pending, indicating the realized profit and loss in the current 8-hour settlement period.
// Please specify the minimum offset number of the current page into the offset field to turn the page.
// Ccy: [Optional] Currency of transaction history XBT or USDT
// type possible values:	RealisedPNL, Deposit, TransferIn, TransferOut
// status possible values: Completed, Pending
func (p *Poloniex) GetFuturesAccountTransactionHistory(ctx context.Context, startAt, endAt time.Time, transactionType string, offset, maxCount int64, ccy currency.Code) (*FuturesTransactionHistory, error) {
	params := url.Values{}
	if !startAt.IsZero() {
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
	}
	if !endAt.IsZero() {
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if transactionType != "" {
		params.Set("type", transactionType)
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if maxCount > 0 {
		params.Set("maxCount", strconv.FormatInt(maxCount, 10))
	}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.String())
	}
	var resp *FuturesTransactionHistory
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, fTransactionHistoryRate, http.MethodGet, "/api/v1/transaction-history", params, nil, &resp)
}

// Trade Config endpoints.

// GetFuturesMaxActiveOrderLimit this endpoint to get the maximum active order and stop order quantity limit.
func (p *Poloniex) GetFuturesMaxActiveOrderLimit(ctx context.Context) (*MaxActiveOrderLimit, error) {
	var resp *MaxActiveOrderLimit
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, authNonResourceIntensiveEPL, http.MethodGet, "/api/v1/maxRiskLimit", nil, nil, &resp)
}

// GetFuturesMaxRiskLimit query this endpoint to get the maximum of risk limit.
func (p *Poloniex) GetFuturesMaxRiskLimit(ctx context.Context, symbol string) (*FuturesMaxRiskLimit, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *FuturesMaxRiskLimit
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, authNonResourceIntensiveEPL, http.MethodGet, "/api/v1/maxRiskLimit", params, nil, &resp)
}

// GetFuturesUserFeeRate retrieves user fee rate.
func (p *Poloniex) GetFuturesUserFeeRate(ctx context.Context) (*FuturesUserFeeRate, error) {
	var resp *FuturesUserFeeRate
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, authNonResourceIntensiveEPL, http.MethodGet, "/api/v1/userFeeRate", nil, nil, &resp)
}

// Margin Mode endpoints

// GetFuturesMarginMode retrieves a margin mode.
func (p *Poloniex) GetFuturesMarginMode(ctx context.Context, symbol string) (int64, error) {
	if symbol == "" {
		return 0, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp int64
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, authNonResourceIntensiveEPL, http.MethodGet, "/api/v1/marginType/query", params, nil, &resp)
}

// ChangeMarginMode changes the margin mode of the account.
func (p *Poloniex) ChangeMarginMode(ctx context.Context, symbol string, marginType margin.Type) error {
	if symbol == "" {
		return currency.ErrSymbolStringEmpty
	}
	if marginType != margin.Isolated && marginType != margin.Multi {
		return margin.ErrInvalidMarginType
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if marginType == margin.Isolated {
		params.Set("marginType", "0")
	}
	if marginType == margin.Multi {
		params.Set("marginType", "1")
	}
	return p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, authNonResourceIntensiveEPL, http.MethodPost, "/api/v1/marginType/change", params, nil, nil)
}

func futuresOrderParamsFilter(arg *FuturesOrderParams) error {
	if *arg == (FuturesOrderParams{}) {
		return common.ErrNilPointer
	}
	if arg.Symbol == "" {
		return currency.ErrSymbolStringEmpty
	}
	if arg.Side == "" {
		return order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return order.ErrTypeIsInvalid
	}
	if arg.OrderType == "limit" {
		if arg.Price <= 0 {
			return order.ErrPriceBelowMin
		}
		if arg.Size <= 0 {
			return order.ErrAmountBelowMin
		}
	}
	return nil
}

// PlaceFuturesOrder places a futures order.
func (p *Poloniex) PlaceFuturesOrder(ctx context.Context, arg *FuturesOrderParams) (*OrderIDResponse, error) {
	err := futuresOrderParamsFilter(arg)
	if err != nil {
		return nil, err
	}
	var resp *OrderIDResponse
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, fOrderEPL, http.MethodPost, "/api/v1/orders", nil, arg, &resp)
}

// PlaceMultipleFuturesOrder places a batch of orders.
func (p *Poloniex) PlaceMultipleFuturesOrder(ctx context.Context, args []FuturesOrderParams) ([]OrderIDResponse, error) {
	if len(args) == 0 {
		return nil, common.ErrNilPointer
	}
	for i := range args {
		err := futuresOrderParamsFilter(&(args[i]))
		if err != nil {
			return nil, err
		}
	}
	input := &struct {
		BatchOrders []FuturesOrderParams `json:"batchOrders"`
	}{
		BatchOrders: args,
	}
	var resp []OrderIDResponse
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, authResourceIntensiveEPL, http.MethodPost, "/api/v1/batchOrders", nil, input, &resp)
}

// CancelFuturesOrderByID cancels a single futures order by ID.
func (p *Poloniex) CancelFuturesOrderByID(ctx context.Context, orderID string) (*FuturesCancelOrderResponse, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *FuturesCancelOrderResponse
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, fCancelOrderEPL, http.MethodDelete, "/api/v1/orders/"+orderID, nil, nil, &resp)
}

// CancelAllFuturesLimitOrders cancels all open orders(excluding stop orders). The response is a list of orderIDs of the canceled orders.
func (p *Poloniex) CancelAllFuturesLimitOrders(ctx context.Context, symbol, side string) (*FuturesCancelOrderResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if side != "" {
		params.Set("side", side)
	}
	var resp *FuturesCancelOrderResponse
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, fCancelAllLimitOrdersEPL, http.MethodDelete, "/api/v1/orders", params, nil, &resp)
}

// CancelMultipleFuturesLimitOrders cancel multiple open orders (excluding stop orders).
// The response is a list of orderIDs (or clientOids) of the canceled orders.
func (p *Poloniex) CancelMultipleFuturesLimitOrders(ctx context.Context, orderIDs, clientOrderIDs []string) (*FuturesCancelOrderResponse, error) {
	if len(orderIDs) == 0 && len(clientOrderIDs) == 0 {
		return nil, errClientOrderIDOROrderIDsRequired
	}
	params := url.Values{}
	if len(orderIDs) > 0 {
		valString, err := json.Marshal(orderIDs)
		if err != nil {
			return nil, err
		}
		params.Set("orderIds", string(valString))
	}
	if len(clientOrderIDs) > 0 {
		valString, err := json.Marshal(clientOrderIDs)
		if err != nil {
			return nil, err
		}
		params.Set("clientOids", string(valString))
	}
	var resp *FuturesCancelOrderResponse
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, fCancelMultipleLimitOrdersEPL, http.MethodDelete, "/api/v1/batchOrders", params, nil, &resp)
}

// CancelAllFuturesStopOrders cancel all untriggered stop orders. The response is a list of orderIDs of the canceled stop orders. To cancel triggered stop orders, please use 'Limit Order Mass Cancelation'.
func (p *Poloniex) CancelAllFuturesStopOrders(ctx context.Context, symbol string) (*FuturesCancelOrderResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *FuturesCancelOrderResponse
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, fCancelAllStopOrdersEPL, http.MethodDelete, "/api/v1/stopOrders", params, nil, &resp)
}

// GetFuturesOrderList retrieves list of current orders.
func (p *Poloniex) GetFuturesOrderList(ctx context.Context, status, symbol, side, orderType string, startAt, endAt time.Time, marginType margin.Type) (*FuturesOrders, error) {
	params := url.Values{}
	if status != "" {
		params.Set("status", status)
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if side != "" {
		params.Set("side", side)
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if !startAt.IsZero() && !endAt.IsZero() {
		err := common.StartEndTimeCheck(startAt, endAt)
		if err != nil {
			return nil, err
		}
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	switch marginType {
	case margin.Multi:
		params.Set("marginType", "0")
	case margin.Isolated:
		params.Set("marginType", "1")
	}
	var resp *FuturesOrders
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, fGetOrdersEPL, http.MethodGet, "/api/v1/orders", params, nil, &resp)
}

// GetFuturesUntriggeredStopOrderList retrieves list of untriggered futures orders.
func (p *Poloniex) GetFuturesUntriggeredStopOrderList(ctx context.Context, symbol, side, orderType string, startAt, endAt time.Time, marginType margin.Type) (*FuturesOrders, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if side != "" {
		params.Set("side", side)
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if !startAt.IsZero() && !endAt.IsZero() {
		err := common.StartEndTimeCheck(startAt, endAt)
		if err != nil {
			return nil, err
		}
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	switch marginType {
	case margin.Multi:
		params.Set("marginType", "0")
	case margin.Isolated:
		params.Set("marginType", "1")
	}
	var resp *FuturesOrders
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, fGetUntriggeredStopOrderEPL, http.MethodGet, "/api/v1/stopOrders", params, nil, &resp)
}

// GetFuturesCompletedOrdersIn24Hour gets list of 1000 completed orders in the last 24 hours.
func (p *Poloniex) GetFuturesCompletedOrdersIn24Hour(ctx context.Context) ([]FuturesOrder, error) {
	var resp []FuturesOrder
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, fGetCompleted24HrEPL, http.MethodGet, "/api/v1/recentDoneOrders", nil, nil, &resp)
}

// GetFuturesSingleOrderDetailByOrderID retrieves a single order detail.
func (p *Poloniex) GetFuturesSingleOrderDetailByOrderID(ctx context.Context, orderID string) (*FuturesOrder, error) {
	return p.getFuturesByID(ctx, "/api/v1/orders/", orderID)
}

// GetFuturesSingleOrderDetailByClientOrderID retrieves a single order detail by client supplied order id.
func (p *Poloniex) GetFuturesSingleOrderDetailByClientOrderID(ctx context.Context, clientOrderID string) (*FuturesOrder, error) {
	return p.getFuturesByID(ctx, "/api/v1/clientOrderId/", clientOrderID)
}

func (p *Poloniex) getFuturesByID(ctx context.Context, path, orderID string) (*FuturesOrder, error) {
	if orderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	var resp *FuturesOrder
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, fGetSingleOrderDetailEPL, http.MethodGet, path+orderID, nil, nil, &resp)
}

// GetFuturesOrderListV2 retrieves futures orders.
func (p *Poloniex) GetFuturesOrderListV2(ctx context.Context, status, symbol, side, orderType,
	direct string, startAt, endAt time.Time, limit int64) (*FuturesOrdersV2, error) {
	params := url.Values{}
	if status != "" {
		params.Set("status", status)
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if side != "" {
		params.Set("side", side)
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if !startAt.IsZero() && !endAt.IsZero() {
		err := common.StartEndTimeCheck(startAt, endAt)
		if err != nil {
			return nil, err
		}
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if direct != "" {
		params.Set("direct", direct)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *FuturesOrdersV2
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, fGetFuturesOrdersV2EPL, http.MethodGet, "/api/v2/orders", params, nil, &resp)
}

// ----------------------------------------------------------------- Fills Endpoints ----------------------------------------------------------------

// GetFuturesOrderFills retrieves futures order fills.
func (p *Poloniex) GetFuturesOrderFills(ctx context.Context, orderID, symbol, side, orderType string, startAt, endAt time.Time) (*FuturesOrderFills, error) {
	params := url.Values{}
	if orderID != "" {
		params.Set("orderId", orderID)
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if side != "" {
		params.Set("side", side)
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if !startAt.IsZero() && !endAt.IsZero() {
		err := common.StartEndTimeCheck(startAt, endAt)
		if err != nil {
			return nil, err
		}
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	var resp *FuturesOrderFills
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, fGetFuturesFillsEPL, http.MethodGet, "/api/v1/fills", params, nil, &resp)
}

// GetFuturesActiveOrderValueCalculation query this endpoint to get the total number and value of the all your active orders.
func (p *Poloniex) GetFuturesActiveOrderValueCalculation(ctx context.Context, symbol string) (*FuturesActiveOrdersValue, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *FuturesActiveOrdersValue
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, fGetActiveOrderValueCalculationEPL, http.MethodGet, "/api/v1/openOrderStatistics", params, nil, &resp)
}

// GetFuturesFillsV2 retrieves futures orders fills v2.
func (p *Poloniex) GetFuturesFillsV2(ctx context.Context, status, symbol, side, orderType, from, direct string, startAt, endAt time.Time, marginType margin.Type, limit int64) (*FuturesOrderFillsV2, error) {
	params := url.Values{}
	if status != "" {
		params.Set("status", status)
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if side != "" {
		params.Set("side", side)
	}
	if orderType != "" {
		params.Set("type", orderType)
	}
	if from != "" {
		params.Set("from", from)
	}
	if direct != "" {
		params.Set("direct", direct)
	}
	if !startAt.IsZero() && !endAt.IsZero() {
		err := common.StartEndTimeCheck(startAt, endAt)
		if err != nil {
			return nil, err
		}
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	switch marginType {
	case margin.Isolated:
		params.Set("marginType", "0")
	case margin.Multi:
		params.Set("marginType", "1")
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *FuturesOrderFillsV2
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, fGetFillsV2EPL, http.MethodGet, "/api/v2/fills", params, nil, &resp)
}

// -----------------------------------------------------------------------------------   Positions  -------------------------------------------------------------------------------------------

// GetFuturesPositionDetails retrieves futures positions details.
func (p *Poloniex) GetFuturesPositionDetails(ctx context.Context, symbol string) (*FuturesPositionDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *FuturesPositionDetail
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, fGetFuturesPositionDetailsEPL, http.MethodGet, "/api/v1/position", params, nil, &resp)
}

// GetFuturesPositionList get the position details of a specified position.
func (p *Poloniex) GetFuturesPositionList(ctx context.Context) ([]FuturesPositionDetail, error) {
	var resp []FuturesPositionDetail
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, fGetPositionListEPL, http.MethodGet, "/api/v1/positions", nil, nil, &resp)
}

func filterManualMarginParams(arg *AlterMarginManuallyParams) error {
	if *arg == (AlterMarginManuallyParams{}) {
		return common.ErrNilPointer
	}
	if arg.Symbol.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if arg.MarginAmount <= 0 {
		return fmt.Errorf("%w, margin amount is required", order.ErrAmountBelowMin)
	}
	if arg.BizNo == "" {
		return errBizNoRequired
	}
	return nil
}

// FuturesAddMarginManually adds a margin manually.
func (p *Poloniex) FuturesAddMarginManually(ctx context.Context, arg *AlterMarginManuallyParams) error {
	if err := filterManualMarginParams(arg); err != nil {
		return err
	}
	return p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, authNonResourceIntensiveEPL, http.MethodPost, "/api/v1/position/margin/deposit-margin", nil, arg, nil)
}

// FuturesRemoveMarginManually removed a margin manually.
func (p *Poloniex) FuturesRemoveMarginManually(ctx context.Context, arg *AlterMarginManuallyParams) error {
	if err := filterManualMarginParams(arg); err != nil {
		return err
	}
	return p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, authNonResourceIntensiveEPL, http.MethodPost, "/api/v1/position/margin/withdraw-margin", nil, arg, nil)
}

// GetFuturesLeverage allows users to query the leverage level
func (p *Poloniex) GetFuturesLeverage(ctx context.Context, symbol string) (*FuturesLeverageResp, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *FuturesLeverageResp
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, authNonResourceIntensiveEPL, http.MethodGet, "/api/v2/position/leverage", params, nil, &resp)
}

// SetFuturesLeverage allows users to set the leverage level
func (p *Poloniex) SetFuturesLeverage(ctx context.Context, symbol string, leverage float64) (*FuturesLeverageResp, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if leverage <= 0 {
		return nil, fmt.Errorf("%w leverage %f", order.ErrSubmitLeverageNotSupported, leverage)
	}
	var resp *FuturesLeverageResp
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, authNonResourceIntensiveEPL, http.MethodPost, "/api/v2/position/leverage", nil, map[string]interface{}{
		"symbol": symbol,
		"lever":  leverage,
	}, &resp)
}

// GetFuturesFundingHistory retrieves the funding history of a symbol.
func (p *Poloniex) GetFuturesFundingHistory(ctx context.Context, symbol string, startAt, endAt time.Time, reverse, forward bool, offset, maxCount int64) (*FuturesFundingHistory, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if !startAt.IsZero() && !endAt.IsZero() {
		err := common.StartEndTimeCheck(startAt, endAt)
		if err != nil {
			return nil, err
		}
		params.Set("startAt", strconv.FormatInt(startAt.UnixMilli(), 10))
		params.Set("endAt", strconv.FormatInt(endAt.UnixMilli(), 10))
	}
	if !reverse {
		params.Set("reverse", "false")
	}
	if !forward {
		params.Set("forward", "false")
	}
	if offset > 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}
	if maxCount > 0 {
		params.Set("maxCount", strconv.FormatInt(maxCount, 10))
	}
	var resp *FuturesFundingHistory
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestFutures, fGetFundingRateEPL, http.MethodGet, "/api/v1/funding-history", params, nil, &resp)
}

// -----------------------------------------------------------------------------------------------------------------------------------------

// GetAccountBalance get information about your Futures account.
func (p *Poloniex) GetAccountBalance(ctx context.Context) (*FuturesAccountBalance, error) {
	var resp *FuturesAccountBalance
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Unset, http.MethodGet, "/v3/account/balance", nil, nil, &resp, true)
}

// GetAccountBills retrieve the account’s bills.
func (p *Poloniex) GetAccountBills(ctx context.Context, startTime, endTime time.Time, offset, limit int64, direction, billType string) ([]BillDetail, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("eTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if offset > 0 {
		params.Set("from", strconv.FormatInt(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if direction != "" {
		params.Set("direct", direction)
	}
	if billType != "" {
		params.Set("type", billType)
	}
	var resp []BillDetail
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.Unset, http.MethodGet, "/v3/account/bills", params, nil, &resp, true)
}

// PlaceV3FuturesOrder place an order in futures trading.
func (p *Poloniex) PlaceV3FuturesOrder(ctx context.Context, arg *FuturesV2Params) (*FuturesV3OrderIDResponse, error) {
	if arg == nil || *arg == (FuturesV2Params{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.PositionSide == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if arg.Size <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	var resp *FuturesV3OrderIDResponse
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "/v3/trade/order", nil, arg, &resp, true)
}

// PlaceV3FuturesMultipleOrders place orders in a batch. A maximum of 10 orders can be placed per request.
func (p *Poloniex) PlaceV3FuturesMultipleOrders(ctx context.Context, args []FuturesV2Params) ([]FuturesV3OrderIDResponse, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	for x := range args {
		err := validationOrderCreationParam(&args[x])
		if err != nil {
			return nil, err
		}
	}
	var resp []FuturesV3OrderIDResponse
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "/v3/trade/orders", nil, args, &resp, true)
}

func validationOrderCreationParam(arg *FuturesV2Params) error {
	if *arg == (FuturesV2Params{}) {
		return common.ErrEmptyParams
	}
	if arg.Symbol == "" {
		return currency.ErrSymbolStringEmpty
	}
	if arg.Side == "" {
		return order.ErrSideIsInvalid
	}
	if arg.PositionSide == "" {
		return order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return order.ErrTypeIsInvalid
	}
	if arg.Size <= 0 {
		return order.ErrAmountBelowMin
	}
	return nil
}

// CancelV3FuturesOrder cancels an order in futures trading.
func (p *Poloniex) CancelV3FuturesOrder(ctx context.Context, arg *CancelOrderParams) (*FuturesV3OrderIDResponse, error) {
	if arg == nil || *arg == (CancelOrderParams{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *FuturesV3OrderIDResponse
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodDelete, "/v3/trade/order", nil, arg, &resp, true)
}

// CancelMultipleV3FuturesOrders cancel orders in a batch. A maximum of 10 orders can be cancelled per request.
func (p *Poloniex) CancelMultipleV3FuturesOrders(ctx context.Context, args *CancelOrdersParams) ([]FuturesV3OrderIDResponse, error) {
	if args == nil {
		return nil, common.ErrEmptyParams
	}
	if args.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp []FuturesV3OrderIDResponse
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodDelete, "/v3/trade/batchOrders", nil, args, &resp, true)
}

// CancelAllV3FuturesOrders cancel all current pending orders.
func (p *Poloniex) CancelAllV3FuturesOrders(ctx context.Context, symbol, side string) ([]FuturesV3OrderIDResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	arg := &struct {
		Symbol string `json:"symbol"`
		Side   string `json:"side,omitempty"`
	}{
		Symbol: symbol,
		Side:   side,
	}
	var resp []FuturesV3OrderIDResponse
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodDelete, "/v3/trade/allOrders", nil, arg, &resp, true)
}

// CloseAtMarketPrice close orders at market price.
func (p *Poloniex) CloseAtMarketPrice(ctx context.Context, symbol, marginMode, positionSide, clientOrderID string) (*FuturesV3OrderIDResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if marginMode == "" {
		return nil, margin.ErrInvalidMarginType
	}
	if clientOrderID == "" {
		return nil, order.ErrClientOrderIDMustBeSet
	}
	arg := &struct {
		Symbol       string `json:"symbol"`
		MgnMode      string `json:"mgnMode"`
		ClOrdID      string `json:"clOrdId"`
		PositionSide string `json:"posSide,omitempty"`
	}{
		Symbol:       symbol,
		MgnMode:      marginMode,
		ClOrdID:      clientOrderID,
		PositionSide: positionSide,
	}
	var resp *FuturesV3OrderIDResponse
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "/v3/trade/position", nil, arg, &resp, true)
}

// CloseAllAtMarketPrice close all orders at market price.
func (p *Poloniex) CloseAllAtMarketPrice(ctx context.Context) ([]FuturesV3OrderIDResponse, error) {
	var resp []FuturesV3OrderIDResponse
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "/v3/trade/positionAll", nil, nil, &resp, true)
}

// GetCurrentOrders get unfilled futures orders. If no request parameters are specified, you will get all open orders sorted on the creation time in chronological order.
func (p *Poloniex) GetCurrentOrders(ctx context.Context, symbol, side, orderID, clientOrderID, direction string, offset, limit int64) ([]FuturesV3Order, error) {
	params := url.Values{}
	if side != "" {
		params.Set("side", side)
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	if clientOrderID != "" {
		params.Set("clOrdId", clientOrderID)
	}
	if direction != "" {
		params.Set("direct", direction)
	}
	if offset > 0 {
		params.Set("from", strconv.FormatInt(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []FuturesV3Order
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "/v3/trade/order/opens", params, nil, &resp, true)
}

// GetOrderExecutionDetails retrieves detailed information about your executed futures order
func (p *Poloniex) GetOrderExecutionDetails(ctx context.Context, symbol, orderID, clientOrderID, direction string, startTime, endTime time.Time, offset, limit int64) ([]FuturesV3Order, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("eTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	if clientOrderID != "" {
		params.Set("clOrdId", clientOrderID)
	}
	if direction != "" {
		params.Set("direct", direction)
	}
	if offset > 0 {
		params.Set("from", strconv.FormatInt(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []FuturesV3Order
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "/v3/trade/order/trades", params, nil, &resp, true)
}

// GetV3FuturesOrderHistory retrieves previous futures orders. Orders that are completely canceled (no transaction has occurred) initiated through the API can only be queried for 4 hours.
func (p *Poloniex) GetV3FuturesOrderHistory(ctx context.Context, symbol, side, orderState, orderID, clientOrderID, direction string, startTime, endTime time.Time, offset, limit int64) ([]FuturesV3Order, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("eTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if side != "" {
		params.Set("side", side)
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if orderState != "" {
		params.Set("state", orderState)
	}
	if orderID != "" {
		params.Set("ordId", orderID)
	}
	if clientOrderID != "" {
		params.Set("clOrdId", clientOrderID)
	}
	if direction != "" {
		params.Set("direct", direction)
	}
	if offset > 0 {
		params.Set("from", strconv.FormatInt(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []FuturesV3Order
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "/v3/trade/order/history", params, nil, &resp, true)
}

// ------------------------------------------------- Position Endpoints ----------------------------------------------------

// GetV3FuturesCurrentPosition retrieves  information about your current position.
func (p *Poloniex) GetV3FuturesCurrentPosition(ctx context.Context, symbol string) ([]V3FuturesPosition, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []V3FuturesPosition
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "/v3/trade/position/opens", params, nil, &resp, true)
}

// GetV3FuturesPositionHistory get information about previous positions.
func (p *Poloniex) GetV3FuturesPositionHistory(ctx context.Context, symbol, marginMode, positionSide, direction string, startTime, endTime time.Time, offset, limit int64) ([]V3FuturesPosition, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if marginMode != "" {
		params.Set("mgnMode", marginMode)
	}
	if positionSide != "" {
		params.Set("posSide", positionSide)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("eTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if direction != "" {
		params.Set("direct", direction)
	}
	if offset > 0 {
		params.Set("from", strconv.FormatInt(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []V3FuturesPosition
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "/v3/trade/position/history", params, nil, &resp, true)
}

// AdjustMarginForIsolatedMarginTradingPositions add or reduce margin for positions in isolated margin mode.
func (p *Poloniex) AdjustMarginForIsolatedMarginTradingPositions(ctx context.Context, symbol, positionSide, adjustType string, amount float64) (*AdjustV3FuturesMarginResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if adjustType == "" {
		return nil, errMarginAdjustTypeMissing
	}
	arg := &struct {
		Symbol       string  `json:"symbol"`
		PositionSide string  `json:"posSide,omitempty"`
		Amount       float64 `json:"amt,string"`
		Type         string  `json:"type"`
	}{
		Symbol:       symbol,
		PositionSide: positionSide,
		Amount:       amount,
		Type:         adjustType,
	}
	var resp *AdjustV3FuturesMarginResponse
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "/v3/trade/position/margin", nil, arg, &resp, true)
}

// GetV3FuturesLeverage retrieves the list of leverage.
func (p *Poloniex) GetV3FuturesLeverage(ctx context.Context, symbol, marginMode string) ([]V3FuturesLeverage, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if marginMode != "" {
		params.Set("mgnMode", marginMode)
	}
	var resp []V3FuturesLeverage
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "/v3/position/leverages", params, nil, &resp, true)
}

// SetV3FuturesLeverage change leverage
func (p *Poloniex) SetV3FuturesLeverage(ctx context.Context, symbol, marginMode, positionSide string, leverage int64) (*V3FuturesLeverage, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if marginMode == "" {
		return nil, margin.ErrInvalidMarginType
	}
	if positionSide == "" {
		return nil, order.ErrSideIsInvalid
	}
	if leverage <= 0 {
		return nil, order.ErrSubmitLeverageNotSupported
	}
	var resp *V3FuturesLeverage
	return resp, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "/v3/position/leverage", nil, map[string]string{
		"symbol":  symbol,
		"mgnMode": marginMode,
		"posSide": positionSide,
		"lever":   strconv.FormatInt(leverage, 10),
	}, &resp, true)
}

// SwitchPositionMode switch the current position mode. Please ensure you do not have open positions and open orders under this mode before the switch.
// Position mode, HEDGE: LONG/SHORT, ONE_WAY: BOTH
func (p *Poloniex) SwitchPositionMode(ctx context.Context, positionMode string) error {
	if positionMode == "" {
		return errPositionModeInvalid
	}
	return p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodPost, "/v3/position/mode", nil, map[string]string{"posMode": positionMode}, &struct{}{}, true)
}

// GetPositionMode get the current position mode.
func (p *Poloniex) GetPositionMode(ctx context.Context) (string, error) {
	resp := &struct {
		PositionMode string `json:"posMode"`
	}{}
	return resp.PositionMode, p.SendAuthenticatedHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, http.MethodGet, "/v3/position/mode", nil, nil, &resp, true)
}

// GetV3FuturesOrderBook get market depth data of the designated trading pair
func (p *Poloniex) GetV3FuturesOrderBook(ctx context.Context, symbol string, depth, limit int64) (*FuturesV3Orderbook, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if depth > 0 {
		params.Set("scale", strconv.FormatInt(depth, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp *FuturesV3Orderbook
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/orderBook", params), &resp, true)
}

var intervalToStringMap = map[kline.Interval]string{kline.OneMin: "MINUTE_1", kline.FiveMin: "MINUTE_5", kline.FifteenMin: "MINUTE_15", kline.ThirtyMin: "MINUTE_30", kline.OneHour: "HOUR_1", kline.TwoHour: "HOUR_2", kline.FourHour: "HOUR_4", kline.TwelveHour: "HOUR_12", kline.OneDay: "DAY_1", kline.ThreeDay: "DAY_3", kline.OneWeek: "WEEK_1"}

func IntervalString(interval kline.Interval) (string, error) {
	intervalString, ok := intervalToStringMap[interval]
	if !ok {
		return "", kline.ErrUnsupportedInterval
	}
	return intervalString, nil
}

// GetV3FuturesKlineData retrieves K-line data of the designated trading pair
func (p *Poloniex) GetV3FuturesKlineData(ctx context.Context, symbol string, interval kline.Interval, startTime, endTime time.Time, limit int64) ([]V3FuturesCandle, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	intervalString, err := IntervalString(interval)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbol)
	params.Set("interval", intervalString)
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("eTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []V3FuturesCandle
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/candles", params), &resp, true)
}

// GetV3FuturesExecutionInfo get the latest execution information. The default limit is 500, with a maximum of 1,000.
func (p *Poloniex) GetV3FuturesExecutionInfo(ctx context.Context, symbol string, limit int64) ([]V3FuturesExecutionInfo, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []V3FuturesExecutionInfo
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/trades", params), &resp, true)
}

// GetV3LiquidiationOrder get Liquidation Order Interface
func (p *Poloniex) GetV3LiquidiationOrder(ctx context.Context, symbol, direction string, startTime, endTime time.Time, offset, limit int64) ([]LiquidiationPriceInfo, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("eTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if direction != "" {
		params.Set("direct", direction)
	}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if offset > 0 {
		params.Set("from", strconv.FormatInt(offset, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []LiquidiationPriceInfo
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/liquidationOrder", params), &resp, true)
}

// GetV3FuturesMarketInfo get the market information of trading pairs in the past 24 hours.
func (p *Poloniex) GetV3FuturesMarketInfo(ctx context.Context, symbol string) ([]V3FuturesTickerDetail, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []V3FuturesTickerDetail
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/tickers", params), &resp, true)
}

// GetV3FuturesIndexPrice get the current index price.
func (p *Poloniex) GetV3FuturesIndexPrice(ctx context.Context, symbol string) (*InstrumentIndexPrice, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *InstrumentIndexPrice
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/indexPrice", params), &resp, true)
}

// GetV3IndexPriceComponents get the index price components for a trading pair.
func (p *Poloniex) GetV3IndexPriceComponents(ctx context.Context, symbol string) (*IndexPriceComponent, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *IndexPriceComponent
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/indexPriceComponents", params), &resp, true)
}

// GetIndexPriceKlineData obtain the K-line data for the index price.
func (p *Poloniex) GetIndexPriceKlineData(ctx context.Context, symbol string, interval kline.Interval, startTime, endTime time.Time, limit int64) (interface{}, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	intervalString, err := IntervalString(interval)
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", intervalString)
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("sTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("eTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []V3FuturesIndexPriceData
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/indexPriceCandlesticks", params), &resp, true)
}

// GetV3FuturesMarkPrice get the current mark price.
func (p *Poloniex) GetV3FuturesMarkPrice(ctx context.Context, symbol string) (*V3FuturesMarkPrice, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *V3FuturesMarkPrice
	return resp, p.SendHTTPRequest(ctx, exchange.RestSpot, request.UnAuth, common.EncodeURLValues("/v3/market/markPrice", params), &resp, true)
}
