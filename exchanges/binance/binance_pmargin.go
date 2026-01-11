package binance

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
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Definitions and Terminology
// Portfolio Margin is an advanced trading mode offered by Binance, designed for experienced traders who seek
// increased leverage and flexibility across various trading products. It incorporates a unique approach to margin
// calculations and risk management to offer a more comprehensive assessment of the trader's overall exposure.

// - Terminology
// Margin refers to Cross Margin
// UM refers to USD-M Futures
// CM refers to Coin-M Futures

// NewUMOrder send in a new USDT margined order/orders.
func (e *Exchange) NewUMOrder(ctx context.Context, arg *UMOrderParam) (*UMCMOrder, error) {
	return e.newUMCMOrder(ctx, arg, "/papi/v1/um/order")
}

// NewCMOrder send in a new Coin margined order/orders.
func (e *Exchange) NewCMOrder(ctx context.Context, arg *UMOrderParam) (*UMCMOrder, error) {
	return e.newUMCMOrder(ctx, arg, "/papi/v1/cm/order")
}

func (e *Exchange) newUMCMOrder(ctx context.Context, arg *UMOrderParam, path string) (*UMCMOrder, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	arg.OrderType = strings.ToUpper(arg.OrderType)
	switch arg.OrderType {
	case "limit", order.Limit.String():
		if arg.TimeInForce == "" {
			return nil, errTimeInForceRequired
		}
		if arg.Quantity <= 0 {
			return nil, limits.ErrAmountBelowMin
		}
		if arg.Price <= 0 {
			return nil, limits.ErrPriceBelowMin
		}
	case "MARKET":
		if arg.Quantity <= 0 {
			return nil, limits.ErrAmountBelowMin
		}
	default:
		return nil, order.ErrUnsupportedOrderType
	}
	var resp *UMCMOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, path, nil, pmDefaultRate, arg, &resp)
}

// NewMarginOrder places a new cross margin order
func (e *Exchange) NewMarginOrder(ctx context.Context, arg *MarginOrderParam) (*MarginOrderResp, error) {
	if *arg == (MarginOrderParam{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Symbol == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	var resp *MarginOrderResp
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/margin/order", nil, pmDefaultRate, arg, &resp)
}

// MarginAccountBorrow apply for margin loan
func (e *Exchange) MarginAccountBorrow(ctx context.Context, ccy currency.Code, amount float64) (string, error) {
	return e.marginAccountBorrowRepay(ctx, ccy, amount, "/papi/v1/marginLoan")
}

// MarginAccountRepay repay for margin loan
func (e *Exchange) MarginAccountRepay(ctx context.Context, ccy currency.Code, amount float64) (string, error) {
	return e.marginAccountBorrowRepay(ctx, ccy, amount, "/papi/v1/repayLoan")
}

func (e *Exchange) marginAccountBorrowRepay(ctx context.Context, ccy currency.Code, amount float64, path string) (string, error) {
	if ccy.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return "", limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("asset", ccy.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	resp := &struct {
		TransactionID string `json:"tranId"`
	}{}
	return resp.TransactionID, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, path, params, pmMarginAccountLoanAndRepayRate, nil, &resp)
}

// MarginAccountNewOCO sends a new OCO order for a margin account.
func (e *Exchange) MarginAccountNewOCO(ctx context.Context, arg *OCOOrderParam) (*OCOOrder, error) {
	if *arg == (OCOOrderParam{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.Price <= 0 {
		return nil, limits.ErrPriceBelowMin
	}
	if arg.StopPrice <= 0 {
		return nil, fmt.Errorf("%w, stopPrice is required", limits.ErrPriceBelowMin)
	}
	var resp *OCOOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/margin/order/oco", nil, pmDefaultRate, arg, &resp)
}

// NewUMConditionalOrder places a new conditional USDT margined order
func (e *Exchange) NewUMConditionalOrder(ctx context.Context, arg *ConditionalOrderParam) (*ConditionalOrder, error) {
	return e.placeConditionalOrder(ctx, arg, "/papi/v1/um/conditional/order")
}

// NewCMConditionalOrder posts a new coin margined futures conditional order.
func (e *Exchange) NewCMConditionalOrder(ctx context.Context, arg *ConditionalOrderParam) (*ConditionalOrder, error) {
	return e.placeConditionalOrder(ctx, arg, "/papi/v1/cm/conditional/order")
}

func (e *Exchange) placeConditionalOrder(ctx context.Context, arg *ConditionalOrderParam, path string) (*ConditionalOrder, error) {
	if arg == nil || *arg == (ConditionalOrderParam{}) {
		return nil, common.ErrEmptyParams
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.StrategyType == "" {
		return nil, errStrategyTypeRequired
	}
	var resp *ConditionalOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, path, nil, pmDefaultRate, arg, &resp)
}

// -------------------------------------------- Cancel Order Endpoints  ----------------------------------------------------

// CancelCMOrder cancels an active Coin Margined Futures limit order.
func (e *Exchange) CancelCMOrder(ctx context.Context, symbol currency.Pair, origClientOrderID string, orderID int64) (*UMCMOrder, error) {
	return e.cancelOrder(ctx, symbol, origClientOrderID, "/papi/v1/cm/order", orderID)
}

// CancelUMOrder cancels an active USDT Margined Futures limit order.
func (e *Exchange) CancelUMOrder(ctx context.Context, symbol currency.Pair, origClientOrderID string, orderID int64) (*UMCMOrder, error) {
	return e.cancelOrder(ctx, symbol, origClientOrderID, "/papi/v1/um/order", orderID)
}

func (e *Exchange) cancelOrder(ctx context.Context, symbol currency.Pair, origClientOrderID, path string, orderID int64) (*UMCMOrder, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if orderID == 0 && origClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	var resp *UMCMOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, path, params, pmDefaultRate, nil, &resp)
}

// CancelAllUMOrders cancels all active USDT Margined Futures limit orders on specific symbol
func (e *Exchange) CancelAllUMOrders(ctx context.Context, symbol currency.Pair) (*SuccessResponse, error) {
	return e.cancelAllUMCMOrders(ctx, symbol, "/papi/v1/um/allOpenOrders")
}

// CancelAllCMOrders cancels all active Coin Margined Futures limit orders on specific symbol
func (e *Exchange) CancelAllCMOrders(ctx context.Context, symbol currency.Pair) (*SuccessResponse, error) {
	return e.cancelAllUMCMOrders(ctx, symbol, "/papi/v1/cm/allOpenOrders")
}

func (e *Exchange) cancelAllUMCMOrders(ctx context.Context, symbol currency.Pair, path string) (*SuccessResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	var resp *SuccessResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, path, params, pmDefaultRate, nil, &resp)
}

// PMCancelMarginAccountOrder cancels margin account order
func (e *Exchange) PMCancelMarginAccountOrder(ctx context.Context, symbol currency.Pair, origClientOrderID string, orderID int64) (*MarginOrderResp, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if orderID == 0 && origClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	var resp *MarginOrderResp
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, "/papi/v1/margin/order", params, pmDefaultRate, nil, &resp)
}

// CancelAllMarginOpenOrdersBySymbol cancels all open margin account orders of a specific symbol.
func (e *Exchange) CancelAllMarginOpenOrdersBySymbol(ctx context.Context, symbol currency.Pair) (MarginAccOrdersList, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	var resp MarginAccOrdersList
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, "/papi/v1/margin/allOpenOrders", params, pmCancelMarginAccountOpenOrdersOnSymbolRate, nil, &resp)
}

// CancelMarginAccountOCOOrders cancels margin account OCO orders.
func (e *Exchange) CancelMarginAccountOCOOrders(ctx context.Context, symbol currency.Pair, listClientOrderID, newClientOrderID string, orderListID int64) (*OCOOrder, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if listClientOrderID != "" {
		params.Set("listClientOrderId", listClientOrderID)
	}
	if newClientOrderID != "" {
		params.Set("newClientOrderId", newClientOrderID)
	}
	if orderListID > 0 {
		params.Set("orderListId", strconv.FormatInt(orderListID, 10))
	}
	var resp *OCOOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, "/papi/v1/margin/orderList", params, pmCancelMarginAccountOCORate, nil, &resp)
}

// CancelUMConditionalOrder cancels a USDT margind futures conditional order
func (e *Exchange) CancelUMConditionalOrder(ctx context.Context, symbol currency.Pair, newClientStrategyID string, strategyID int64) (*ConditionalOrder, error) {
	return e.cancelUMCMConditionalOrder(ctx, symbol, newClientStrategyID, "/papi/v1/um/conditional/order", strategyID)
}

// CancelCMConditionalOrder cancels a Coin margined futures conditional order
func (e *Exchange) CancelCMConditionalOrder(ctx context.Context, symbol currency.Pair, newClientStrategyID string, strategyID int64) (*ConditionalOrder, error) {
	return e.cancelUMCMConditionalOrder(ctx, symbol, newClientStrategyID, "/papi/v1/cm/conditional/order", strategyID)
}

func (e *Exchange) cancelUMCMConditionalOrder(ctx context.Context, symbol currency.Pair, newClientStrategyID, path string, strategyID int64) (*ConditionalOrder, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if strategyID == 0 && newClientStrategyID == "" {
		return nil, fmt.Errorf("%w, either strategyId or newClientStrategyId is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if strategyID > 0 {
		params.Set("strategyId", strconv.FormatInt(strategyID, 10))
	}
	if newClientStrategyID != "" {
		params.Set("newClientStrategyId", newClientStrategyID)
	}
	var resp *ConditionalOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, path, params, pmDefaultRate, nil, &resp)
}

// CancelAllUMOpenConditionalOrders cancels all open conditional USDT margined orders
func (e *Exchange) CancelAllUMOpenConditionalOrders(ctx context.Context, symbol currency.Pair) (*SuccessResponse, error) {
	return e.cancelAllUMCMOpenConditionalOrders(ctx, symbol, "/papi/v1/um/conditional/allOpenOrders")
}

// CancelAllCMOpenConditionalOrders cancels all open conditional Coin margined orders
func (e *Exchange) CancelAllCMOpenConditionalOrders(ctx context.Context, symbol currency.Pair) (*SuccessResponse, error) {
	return e.cancelAllUMCMOpenConditionalOrders(ctx, symbol, "/papi/v1/cm/conditional/allOpenOrders")
}

func (e *Exchange) cancelAllUMCMOpenConditionalOrders(ctx context.Context, symbol currency.Pair, path string) (*SuccessResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	var resp *SuccessResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, path, params, pmDefaultRate, nil, &resp)
}

// --------------------------------------------------------   Query Order Endpoints  --------------------------------------------------------

// GetUMOrder check an USDT Margined order's status
// Orders can not be found if the order status is CANCELED or EXPIRED
func (e *Exchange) GetUMOrder(ctx context.Context, symbol currency.Pair, origClientOrderID string, orderID int64) (*UMCMOrder, error) {
	return e.getUMCMOrder(ctx, symbol, origClientOrderID, "/papi/v1/um/order", orderID)
}

// GetUMOpenOrder get current UM open order
func (e *Exchange) GetUMOpenOrder(ctx context.Context, symbol currency.Pair, origClientOrderID string, orderID int64) (*UMCMOrder, error) {
	return e.getUMCMOrder(ctx, symbol, origClientOrderID, "/papi/v1/um/openOrder", orderID)
}

func (e *Exchange) getUMCMOrder(ctx context.Context, symbol currency.Pair, origClientOrderID, path string, orderID int64) (*UMCMOrder, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if orderID == 0 && origClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	var resp *UMCMOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, path, params, pmDefaultRate, nil, &resp)
}

// GetAllUMOpenOrders retrieves all open USDT margined orders.
// If no symbol is provided, it will load all open USDT orders, taking more ratelimit weight than the ordinary endpoints.
func (e *Exchange) GetAllUMOpenOrders(ctx context.Context, symbol currency.Pair) ([]*UMCMOrder, error) {
	endpointLimit := pmDefaultRate
	if symbol.IsEmpty() {
		endpointLimit = pmRetrieveAllUMOpenOrdersForAllSymbolRate
	}
	return e.getUMOrders(ctx, symbol, "/papi/v1/um/openOrders", time.Time{}, time.Time{}, 0, 0, endpointLimit)
}

// GetAllUMOrders retrieves all USDT margined orders except for
// 1. CANCELED or EXPIRED orders.
// 2. Orders with not fill.
//  3. Order Created later earlier than three days from now.
//
// If orderId is set, it will get orders >= that orderId. Otherwise most recent orders are returned.
// The query time period must be less then 7 days.
func (e *Exchange) GetAllUMOrders(ctx context.Context, symbol currency.Pair, startTime, endTime time.Time, startingOrderID, limit int64) ([]*UMCMOrder, error) {
	return e.getUMOrders(ctx, symbol, "/papi/v1/um/allOrders", startTime, endTime, startingOrderID, limit, pmGetAllUMOrdersRate)
}

func (e *Exchange) getUMOrders(ctx context.Context, symbol currency.Pair, path string, startTime, endTime time.Time, startingOrderID, limit int64, endpointLimit request.EndpointLimit) ([]*UMCMOrder, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if startingOrderID > 0 {
		params.Set("orderId", strconv.FormatInt(startingOrderID, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []*UMCMOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, path, params, endpointLimit, nil, &resp)
}

// GetCMOrder retrieves Coin Margined order instance.
func (e *Exchange) GetCMOrder(ctx context.Context, symbol currency.Pair, origClientOrderID string, orderID int64) (*UMCMOrder, error) {
	return e.getUMCMOrder(ctx, symbol, origClientOrderID, "/papi/v1/cm/order", orderID)
}

// GetCMOpenOrder retrieves Coin Margined open order instance.
func (e *Exchange) GetCMOpenOrder(ctx context.Context, symbol currency.Pair, origClientOrderID string, orderID int64) (*UMCMOrder, error) {
	return e.getUMCMOrder(ctx, symbol, origClientOrderID, "/papi/v1/cm/openOrder", orderID)
}

// GetAllCMOpenOrders retrieves all open Coin Margined futures orders on a symbol.
func (e *Exchange) GetAllCMOpenOrders(ctx context.Context, symbol currency.Pair, pair string) ([]*UMCMOrder, error) {
	endpointLimit := pmDefaultRate
	if symbol.IsEmpty() {
		endpointLimit = pmRetrieveAllCMOpenOrdersForAllSymbolRate
	}
	return e.getCMOrders(ctx, symbol, pair, "/papi/v1/cm/openOrders", time.Time{}, time.Time{}, 0, 0, endpointLimit)
}

// GetAllCMOrders get all account CM orders; active, canceled, or filled.
//
// Either symbol or pair must be sent.
// If orderId is set, it will get orders >= that orderId. Otherwise most recent orders are returned.
// These orders will not be found:
// - order status is CANCELED or EXPIRED, AND
// - order has NO filled trade, AND
// - created time + 3 days < current time
func (e *Exchange) GetAllCMOrders(ctx context.Context, symbol currency.Pair, pair string, startTime, endTime time.Time, startingOrderID, limit int64) ([]*UMCMOrder, error) {
	endpointLimit := pmAllCMOrderWithSymbolRate
	if symbol.IsEmpty() {
		endpointLimit = pmAllCMOrderWithoutSymbolRate
	}
	return e.getCMOrders(ctx, symbol, pair, "/papi/v1/cm/allOrders", startTime, endTime, startingOrderID, limit, endpointLimit)
}

func (e *Exchange) getCMOrders(ctx context.Context, symbol currency.Pair, pair, path string, startTime, endTime time.Time, startingOrderID, limit int64, endpointLimit request.EndpointLimit) ([]*UMCMOrder, error) {
	if symbol.IsEmpty() && pair == "" {
		return nil, fmt.Errorf("%w either symbol or pair is required", currency.ErrCurrencyPairEmpty)
	}
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	if pair == "" {
		params.Set("pair", pair)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if startingOrderID > 0 {
		params.Set("orderId", strconv.FormatInt(startingOrderID, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []*UMCMOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, path, params, endpointLimit, nil, &resp)
}

// GetOpenUMConditionalOrder retrieves a conditional USDT margined order
func (e *Exchange) GetOpenUMConditionalOrder(ctx context.Context, symbol currency.Pair, newClientStrategyID string, strategyID int64) (*ConditionalOrder, error) {
	return e.getOpenUMCMConditionalOrder(ctx, symbol, newClientStrategyID, "/papi/v1/um/conditional/openOrder", strategyID)
}

func (e *Exchange) getOpenUMCMConditionalOrder(ctx context.Context, symbol currency.Pair, newClientStrategyID, path string, strategyID int64) (*ConditionalOrder, error) {
	if strategyID == 0 && newClientStrategyID == "" {
		return nil, fmt.Errorf("%w, either strategyId or newClientStrategyId is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if strategyID > 0 {
		params.Set("strategyId", strconv.FormatInt(strategyID, 10))
	}
	if newClientStrategyID != "" {
		params.Set("newClientStrategyId", newClientStrategyID)
	}
	var resp *ConditionalOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, pmDefaultRate, nil, &resp)
}

// GetAllUMOpenConditionalOrders retrieves all open conditional orders on a symbol.
func (e *Exchange) GetAllUMOpenConditionalOrders(ctx context.Context, symbol currency.Pair) ([]*ConditionalOrder, error) {
	endpointLimit := pmDefaultRate
	if symbol.IsEmpty() {
		endpointLimit = pmUMOpenConditionalOrdersRate
	}
	return e.getAllUMCMOrders(ctx, symbol, "/papi/v1/um/conditional/openOrders", "", time.Time{}, time.Time{}, 0, 0, endpointLimit)
}

// GetAllUMConditionalOrderHistory retrieves all conditional order history a symbol.
func (e *Exchange) GetAllUMConditionalOrderHistory(ctx context.Context, symbol currency.Pair, newClientStrategyID string, strategyID int64) ([]*ConditionalOrder, error) {
	return e.getAllUMCMOrders(ctx, symbol, "/papi/v1/um/conditional/orderHistory", newClientStrategyID, time.Time{}, time.Time{}, strategyID, 0, pmDefaultRate)
}

// GetAllUMConditionalOrders retrieves conditional orders.
func (e *Exchange) GetAllUMConditionalOrders(ctx context.Context, symbol currency.Pair, startTime, endTime time.Time, strategyID, limit int64) ([]*ConditionalOrder, error) {
	endpointLimit := pmDefaultRate
	if symbol.IsEmpty() {
		endpointLimit = pmAllUMConditionalOrdersWithoutSymbolRate
	}
	return e.getAllUMCMOrders(ctx, symbol, "/papi/v1/um/conditional/allOrders", "", startTime, endTime, strategyID, limit, endpointLimit)
}

func (e *Exchange) getAllUMCMOrders(ctx context.Context, symbol currency.Pair, path, newClientStrategyID string, startTime, endTime time.Time, strategyID, limit int64, endpointLimit request.EndpointLimit) ([]*ConditionalOrder, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	if strategyID > 0 {
		params.Set("strategyId", strconv.FormatInt(strategyID, 10))
	}
	if newClientStrategyID != "" {
		params.Set("newClientStrategyId", newClientStrategyID)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []*ConditionalOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, path, params, endpointLimit, nil, &resp)
}

// GetOpenCMConditionalOrder get current Coin Margined open conditional order
func (e *Exchange) GetOpenCMConditionalOrder(ctx context.Context, symbol currency.Pair, newClientStrategyID string, strategyID int64) (*ConditionalOrder, error) {
	return e.getOpenUMCMConditionalOrder(ctx, symbol, newClientStrategyID, "", strategyID)
}

// GetAllCMOpenConditionalOrders retrieves all open conditional orders on a symbol.
func (e *Exchange) GetAllCMOpenConditionalOrders(ctx context.Context, symbol currency.Pair) ([]*ConditionalOrder, error) {
	endpointLimit := pmDefaultRate
	if symbol.IsEmpty() {
		endpointLimit = pmAllCMOpenConditionalOrdersWithoutSymbolRate
	}
	return e.getAllUMCMOrders(ctx, symbol, "/papi/v1/cm/conditional/openOrders", "", time.Time{}, time.Time{}, 0, 0, endpointLimit)
}

// GetAllCMConditionalOrderHistory retrieves all conditional order history a symbol.
func (e *Exchange) GetAllCMConditionalOrderHistory(ctx context.Context, symbol currency.Pair, newClientStrategyID string, strategyID int64) ([]*ConditionalOrder, error) {
	return e.getAllUMCMOrders(ctx, symbol, "/papi/v1/cm/conditional/orderHistory", newClientStrategyID, time.Time{}, time.Time{}, strategyID, 0, pmDefaultRate)
}

// GetAllCMConditionalOrders retrieves conditional orders.
func (e *Exchange) GetAllCMConditionalOrders(ctx context.Context, symbol currency.Pair, startTime, endTime time.Time, strategyID, limit int64) ([]*ConditionalOrder, error) {
	endpointLimit := pmDefaultRate
	if symbol.IsEmpty() {
		endpointLimit = pmAllCMConditionalOrderWithoutSymbolRate
	}
	return e.getAllUMCMOrders(ctx, symbol, "/papi/v1/cm/conditional/allOrders", "", startTime, endTime, strategyID, limit, endpointLimit)
}

// GetMarginAccountOrder retrieves margin account order.
func (e *Exchange) GetMarginAccountOrder(ctx context.Context, symbol currency.Pair, origClientOrderID string, orderID int64) (*MarginOrder, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if orderID == 0 && origClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	var resp *MarginOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/order", params, pmGetMarginAccountOrderRate, nil, &resp)
}

// GetCurrentMarginOpenOrder retrieves an open order.
// If the symbol is not sent, orders for all symbols will be returned in an array.
func (e *Exchange) GetCurrentMarginOpenOrder(ctx context.Context, symbol currency.Pair) ([]*MarginOrder, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	var resp []*MarginOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/openOrders", params, pmCurrentMarginOpenOrderRate, nil, &resp)
}

// GetAllMarginAccountOrders retrieves all margin account orders
func (e *Exchange) GetAllMarginAccountOrders(ctx context.Context, symbol currency.Pair, startTime, endTime time.Time, orderID, limit int64) ([]*MarginOrder, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []*MarginOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/allOrders", params, pmAllMarginAccountOrdersRate, nil, &resp)
}

// GetMarginAccountOCO retrieves a specific OCO based on provided optional parameters.
func (e *Exchange) GetMarginAccountOCO(ctx context.Context, orderListID int64, origClientOrderID string) (*OCOOrder, error) {
	params := url.Values{}
	if orderListID > 0 {
		params.Set("orderListId", strconv.FormatInt(orderListID, 10))
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	var resp *OCOOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/orderList", params, pmGetMarginAccountOCORate, nil, &resp)
}

// GetPMMarginAccountAllOCO a portfolio margin method to retrieve all OCO for a specific margin account based on provided optional parameters
func (e *Exchange) GetPMMarginAccountAllOCO(ctx context.Context, startTime, endTime time.Time, fromID, limit int64) ([]*OCOOrder, error) {
	params := url.Values{}
	if fromID > 0 {
		params.Set("fromId", strconv.FormatInt(fromID, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []*OCOOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/allOrderList", params, pmGetMarginAccountsAllOCOOrdersRate, nil, &resp)
}

// GetMarginAccountsOpenOCO retrieves a margin account open OCO order
func (e *Exchange) GetMarginAccountsOpenOCO(ctx context.Context) ([]*OCOOrder, error) {
	var resp []*OCOOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/openOrderList", nil, pmGetMarginAccountsOpenOCOOrdersRate, nil, &resp)
}

// GetPMMarginAccountTradeList retrieves margin account trade list
func (e *Exchange) GetPMMarginAccountTradeList(ctx context.Context, symbol currency.Pair, startTime, endTime time.Time, orderID, fromID, limit int64) ([]*TradeHistory, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params, err := ocoOrdersAndTradeParams(symbol, false, startTime, endTime, orderID, fromID, limit)
	if err != nil {
		return nil, err
	}
	var resp []*TradeHistory
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/myTrades", params, pmGetMarginAccountTradeListRate, nil, &resp)
}

//  ---------------------------------------------------  Account Endpoints  -------------------------------------------------------------------------------------

// GetAccountBalance retrieves all account balance information related to an asset/assets(if assetName is not provided).
func (e *Exchange) GetAccountBalance(ctx context.Context, assetName currency.Code) (AccountBalanceResponse, error) {
	params := url.Values{}
	if !assetName.IsEmpty() {
		params.Set("asset", assetName.String())
	}
	var resp AccountBalanceResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/balance", params, pmGetAccountBalancesRate, nil, &resp)
}

// GetPortfolioMarginAccountInformation retrieves an account information
func (e *Exchange) GetPortfolioMarginAccountInformation(ctx context.Context) (*AccountInformation, error) {
	var resp *AccountInformation
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/account", nil, pmGetAccountInformationRate, nil, &resp)
}

// GetPMMarginMaxBorrow holds the maximum borrowable amount limited by the account level.
func (e *Exchange) GetPMMarginMaxBorrow(ctx context.Context, assetName currency.Code) (*MaxBorrow, error) {
	if assetName.IsEmpty() {
		return nil, fmt.Errorf("%w, assetName is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("asset", assetName.String())
	var resp *MaxBorrow
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/maxBorrowable", params, pmMarginMaxBorrowRate, nil, &resp)
}

// GetMarginMaxWithdrawal retrieves the maximum withdrawal amount allowed for margin account.
func (e *Exchange) GetMarginMaxWithdrawal(ctx context.Context, assetName currency.Code) (float64, error) {
	if assetName.IsEmpty() {
		return 0, fmt.Errorf("%w, assetName is required", currency.ErrCurrencyCodeEmpty)
	}
	resp := &struct {
		Amount float64 `json:"amount"`
	}{}
	params := url.Values{}
	params.Set("amount", assetName.String())
	return resp.Amount, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/maxWithdraw", params, pmGetMarginMaxWithdrawalRate, nil, &resp)
}

// GetUMPositionInformation get current UM position information.
//
// for One-way Mode user, the response will only show the "BOTH" positions
// for Hedge Mode user, the response will show "LONG", and "SHORT" positions.
func (e *Exchange) GetUMPositionInformation(ctx context.Context, symbol currency.Pair) ([]*UMPositionInformation, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	var resp []*UMPositionInformation
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/positionRisk", params, pmGetUMPositionInformationRate, nil, &resp)
}

// GetCMPositionInformation retrieves current margin position information.
func (e *Exchange) GetCMPositionInformation(ctx context.Context, marginAsset currency.Code, pair string) ([]*CMPositionInformation, error) {
	params := url.Values{}
	if !marginAsset.IsEmpty() {
		params.Set("marginAsset", marginAsset.String())
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	var resp []*CMPositionInformation
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/positionRisk", params, pmDefaultRate, nil, &resp)
}

// ChangeUMInitialLeverage changes user's initial leverage of specific symbol in UM.
func (e *Exchange) ChangeUMInitialLeverage(ctx context.Context, symbol currency.Pair, leverage float64) (*InitialLeverage, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if leverage < 1 || leverage > 125 {
		return nil, fmt.Errorf("%w, leverage must be between 1 and 125", order.ErrSubmitLeverageNotSupported)
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	var resp *InitialLeverage
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/um/leverage", params, pmDefaultRate, nil, &resp)
}

// ChangeCMInitialLeverage change user's initial leverage of specific symbol in CM.
func (e *Exchange) ChangeCMInitialLeverage(ctx context.Context, symbol currency.Pair, leverage float64) (*CMInitialLeverage, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if leverage < 1 || leverage > 125 {
		return nil, fmt.Errorf("%w, leverage must be between 1 and 125", order.ErrSubmitLeverageNotSupported)
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	var resp *CMInitialLeverage
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/cm/leverage", params, pmDefaultRate, nil, &resp)
}

// ChangeUMPositionMode change user's position mode (Hedge Mode or One-way Mode ) on EVERY symbol in UM
func (e *Exchange) ChangeUMPositionMode(ctx context.Context, dualSidePosition bool) (*SuccessResponse, error) {
	return e.changeUMCMPositionMode(ctx, dualSidePosition, "/papi/v1/um/positionSide/dual")
}

// ChangeCMPositionMode change user's position mode (Hedge Mode or One-way Mode ) on EVERY symbol in CM
func (e *Exchange) ChangeCMPositionMode(ctx context.Context, dualSidePosition bool) (*SuccessResponse, error) {
	return e.changeUMCMPositionMode(ctx, dualSidePosition, "/papi/v1/cm/positionSide/dual")
}

func (e *Exchange) changeUMCMPositionMode(ctx context.Context, dualSidePosition bool, path string) (*SuccessResponse, error) {
	params := url.Values{}
	if dualSidePosition {
		params.Set("dualSidePosition", "true")
	} else {
		params.Set("dualSidePosition", "false")
	}
	var resp *SuccessResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, path, params, pmDefaultRate, nil, &resp)
}

// GetUMCurrentPositionMode get user's position mode (Hedge Mode or One-way Mode ) on EVERY symbol in UM
func (e *Exchange) GetUMCurrentPositionMode(ctx context.Context) (*DualPositionMode, error) {
	return e.getPositionMode(ctx, "/papi/v1/um/positionSide/dual", pmGetUMCurrentPositionModeRate)
}

// GetCMCurrentPositionMode get user's position mode (Hedge Mode or One-way Mode ) on EVERY symbol in CM
func (e *Exchange) GetCMCurrentPositionMode(ctx context.Context) (*DualPositionMode, error) {
	return e.getPositionMode(ctx, "/papi/v1/cm/positionSide/dual", pmGetCMCurrentPositionModeRate)
}

func (e *Exchange) getPositionMode(ctx context.Context, path string, endpointLimit request.EndpointLimit) (*DualPositionMode, error) {
	var resp *DualPositionMode
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, path, nil, endpointLimit, nil, &resp)
}

// GetUMAccountTradeList get trades for a specific account and UM symbol.
func (e *Exchange) GetUMAccountTradeList(ctx context.Context, symbol currency.Pair, startTime, endTime time.Time, fromID, limit int64) ([]*UMCMAccountTradeItem, error) {
	return e.getUMCMAccountTradeList(ctx, symbol, "/papi/v1/um/userTrades", startTime, endTime, fromID, limit, pmGetUMAccountTradeListRate)
}

// GetCMAccountTradeList get trades for a specific account and CM symbol.
func (e *Exchange) GetCMAccountTradeList(ctx context.Context, symbol currency.Pair, pair string, startTime, endTime time.Time, fromID, limit int64) ([]*UMCMAccountTradeItem, error) {
	if symbol.IsEmpty() && pair == "" {
		return nil, fmt.Errorf("%w, either symbol or pair is required", currency.ErrCurrencyPairEmpty)
	}
	endpointLimit := pmGetCMAccountTradeListWithPairRate
	if !symbol.IsEmpty() {
		endpointLimit = pmGetCMAccountTradeListWithSymbolRate
	}
	return e.getUMCMAccountTradeList(ctx, symbol, "/papi/v1/cm/userTrades", startTime, endTime, fromID, limit, endpointLimit)
}

func (e *Exchange) getUMCMAccountTradeList(ctx context.Context, symbol currency.Pair, path string, startTime, endTime time.Time, fromID, limit int64, endpointLimit request.EndpointLimit) ([]*UMCMAccountTradeItem, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if fromID > 0 {
		params.Set("fromId", strconv.FormatInt(fromID, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []*UMCMAccountTradeItem
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, path, params, endpointLimit, nil, &resp)
}

// GetUMNotionalAndLeverageBrackets query UM notional and leverage brackets
func (e *Exchange) GetUMNotionalAndLeverageBrackets(ctx context.Context, symbol currency.Pair) ([]*NotionalAndLeverage, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	var resp []*NotionalAndLeverage
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/leverageBracket", params, pmDefaultRate, nil, &resp)
}

// GetCMNotionalAndLeverageBrackets query UM notional and leverage brackets
func (e *Exchange) GetCMNotionalAndLeverageBrackets(ctx context.Context, symbol currency.Pair) ([]*CMNotionalAndLeverage, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	var resp []*CMNotionalAndLeverage
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/cm/leverageBracket", params, pmDefaultRate, nil, &resp)
}

// GetUsersMarginForceOrders query user's margin force orders
func (e *Exchange) GetUsersMarginForceOrders(ctx context.Context, startTime, endTime time.Time, current, size int64) (*MarginForceOrder, error) {
	params := url.Values{}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if current > 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	var resp *MarginForceOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/forceOrders", params, pmDefaultRate, nil, &resp)
}

// GetUsersUMForceOrders query User's UM Force Orders
func (e *Exchange) GetUsersUMForceOrders(ctx context.Context, symbol currency.Pair, autoCloseType string, startTime, endTime time.Time, limit int64) ([]*ForceOrder, error) {
	endpointLimit := pmGetUserUMForceOrdersWithSymbolRate
	if symbol.IsEmpty() {
		endpointLimit = pmGetUserUMForceOrdersWithoutSymbolRate
	}
	return e.getUsersUMCMForceOrders(ctx, symbol, autoCloseType, "/papi/v1/um/forceOrders", startTime, endTime, limit, endpointLimit)
}

// GetUsersCMForceOrders query User's CM Force Orders
func (e *Exchange) GetUsersCMForceOrders(ctx context.Context, symbol currency.Pair, autoCloseType string, startTime, endTime time.Time, limit int64) ([]*ForceOrder, error) {
	endpointLimit := pmGetUserCMForceOrdersWithSymbolRate
	if symbol.IsEmpty() {
		endpointLimit = pmGetUserCMForceOrdersWithoutSymbolRate
	}
	return e.getUsersUMCMForceOrders(ctx, symbol, autoCloseType, "/papi/v1/cm/forceOrders", startTime, endTime, limit, endpointLimit)
}

func (e *Exchange) getUsersUMCMForceOrders(ctx context.Context, symbol currency.Pair, autoCloseType, path string, startTime, endTime time.Time, limit int64, endpointLimit request.EndpointLimit) ([]*ForceOrder, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if autoCloseType != "" {
		params.Set("autoCloseType", autoCloseType)
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []*ForceOrder
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, path, params, endpointLimit, nil, &resp)
}

// GetPortfolioMarginUMTradingQuantitativeRulesIndicator retrieves rules that regulate general trading based on the quantitative indicators
func (e *Exchange) GetPortfolioMarginUMTradingQuantitativeRulesIndicator(ctx context.Context, symbol currency.Pair) (*TradingQuantitativeRulesIndicators, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	endpointLimit := pmDefaultRate
	if symbol.IsEmpty() {
		endpointLimit = pmUMTradingQuantitativeRulesIndicatorsRate
	}
	var resp *TradingQuantitativeRulesIndicators
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/um/apiTradingStatus", params, endpointLimit, nil, &resp)
}

// GetUMUserCommissionRate retrieves usdt margined account user's commission rate
func (e *Exchange) GetUMUserCommissionRate(ctx context.Context, symbol currency.Pair) (*CommissionRate, error) {
	return e.getUserCommissionRate(ctx, symbol, "/papi/v1/um/commissionRate", pmGetUMUserCommissionRate)
}

// GetCMUserCommissionRate retrieves coin margined account user's commission rate
func (e *Exchange) GetCMUserCommissionRate(ctx context.Context, symbol currency.Pair) (*CommissionRate, error) {
	return e.getUserCommissionRate(ctx, symbol, "/papi/v1/cm/commissionRate", pmGetCMUserCommissionRate)
}

func (e *Exchange) getUserCommissionRate(ctx context.Context, symbol currency.Pair, path string, endpointLimit request.EndpointLimit) (*CommissionRate, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol.String())
	var resp *CommissionRate
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, path, params, endpointLimit, nil, &resp)
}

func prepareMarginLoanOrRepayParams(assetName currency.Code, startTime, endTime time.Time, transactionID, current, size int64) (url.Values, error) {
	params := url.Values{}
	if !assetName.IsEmpty() {
		params.Set("assetName", assetName.String())
	}
	if transactionID > 0 {
		params.Set("txId", strconv.FormatInt(transactionID, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if current > 0 {
		params.Set("current", strconv.FormatInt(current, 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	return params, nil
}

// GetMarginLoanRecord query margin loan record
func (e *Exchange) GetMarginLoanRecord(ctx context.Context, assetName currency.Code, startTime, endTime time.Time, transactionID, current, size int64) (*MarginLoanRecord, error) {
	if assetName.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params, err := prepareMarginLoanOrRepayParams(assetName, startTime, endTime, transactionID, current, size)
	if err != nil {
		return nil, err
	}
	var resp *MarginLoanRecord
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/marginLoan", params, pmGetMarginLoanRecordRate, nil, &resp)
}

// GetMarginRepayRecord query margin repay record.
func (e *Exchange) GetMarginRepayRecord(ctx context.Context, assetName currency.Code, startTime, endTime time.Time, transactionID, current, size int64) (*MarginRepayRecord, error) {
	if assetName.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params, err := prepareMarginLoanOrRepayParams(assetName, startTime, endTime, transactionID, current, size)
	if err != nil {
		return nil, err
	}
	var resp *MarginRepayRecord
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/repayLoan", params, pmGetMarginRepayRecordRate, nil, &resp)
}

// GetMarginBorrowOrLoanInterestHistory retrieves margin borrow loan interest history
func (e *Exchange) GetMarginBorrowOrLoanInterestHistory(ctx context.Context, assetName currency.Code, startTime, endTime time.Time, transactionID, current, size int64) (*MarginBorrowOrLoanInterest, error) {
	params, err := prepareMarginLoanOrRepayParams(assetName, startTime, endTime, transactionID, current, size)
	if err != nil {
		return nil, err
	}
	var resp *MarginBorrowOrLoanInterest
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/marginInterestHistory", params, pmDefaultRate, nil, &resp)
}

// GetPortfolioMarginNegativeBalanceInterestHistory retrieves interest history of negative balance for portfolio margin.
func (e *Exchange) GetPortfolioMarginNegativeBalanceInterestHistory(ctx context.Context, assetName currency.Code, startTime, endTime time.Time, size int64) (*PortfolioMarginNegativeBalanceInterest, error) {
	params := url.Values{}
	if !assetName.IsEmpty() {
		params.Set("asset", assetName.String())
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if size > 0 {
		params.Set("size", strconv.FormatInt(size, 10))
	}
	var resp *PortfolioMarginNegativeBalanceInterest
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/portfolio/interest-history", params, pmGetPortfolioMarginNegativeBalanceInterestHistoryRate, nil, &resp)
}

// FundAutoCollection fund collection for Portfolio Margin
func (e *Exchange) FundAutoCollection(ctx context.Context) (string, error) {
	resp := &struct {
		Message string `json:"msg"`
	}{}
	return resp.Message, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/auto-collection", nil, pmFundAutoCollectionRate, nil, &resp)
}

// FundCollectionByAsset transfers specific asset from Futures Account to Margin account
// The BNB transfer is not be supported
func (e *Exchange) FundCollectionByAsset(ctx context.Context, assetName currency.Code) (string, error) {
	if assetName.IsEmpty() {
		return "", fmt.Errorf("%w, assetName is required", currency.ErrCurrencyCodeEmpty)
	}
	params := url.Values{}
	params.Set("asset", assetName.String())
	resp := &struct {
		Message string `json:"msg"`
	}{}
	return resp.Message, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/asset-collection", params, pmFundCollectionByAssetRate, nil, &resp)
}

// BNBTransfer Transfer BNB assets
// transferSize: "TO_UM","FROM_UM"
func (e *Exchange) BNBTransfer(ctx context.Context, amount float64, transferSide string) (int64, error) {
	return e.bnbTransfer(ctx, amount, transferSide, "/papi/v1/bnb-transfer", pmBNBTransferRate, exchange.RestFuturesSupplementary)
}

func (e *Exchange) bnbTransfer(ctx context.Context, amount float64, transferSide, path string, endpointLimit request.EndpointLimit, exchangeURL exchange.URL) (int64, error) {
	params := url.Values{}
	if amount > 0 {
		params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	}
	if transferSide != "" {
		params.Set("transferSide", transferSide)
	}
	resp := &struct {
		TransactionID int64 `json:"transId"`
	}{}
	return resp.TransactionID, e.SendAuthHTTPRequest(ctx, exchangeURL, http.MethodPost, path, params, endpointLimit, nil, &resp)
}

// GetUMIncomeHistory retrieves USDT margined futures income history
// possible incomeType values: TRANSFER, WELCOME_BONUS, REALIZED_PNL, FUNDING_FEE, COMMISSION, INSURANCE_CLEAR, REFERRAL_KICKBACK, COMMISSION_REBATE, API_REBATE, CONTEST_REWARD, CROSS_COLLATERAL_TRANSFER, OPTIONS_PREMIUM_FEE, OPTIONS_SETTLE_PROFIT, INTERNAL_TRANSFER, AUTO_EXCHANGE, DELIVERED_SETTLEMENT, COIN_SWAP_DEPOSIT, COIN_SWAP_WITHDRAW, POSITION_LIMIT_INCREASE_FEE
func (e *Exchange) GetUMIncomeHistory(ctx context.Context, symbol currency.Pair, incomeType string, startTime, endTime time.Time, limit int64) ([]*IncomeItem, error) {
	return e.getUMCMIncomeHistory(ctx, symbol, incomeType, "/papi/v1/um/income", startTime, endTime, limit, pmGetUMIncomeHistoryRate)
}

// GetCMIncomeHistory get current UM account asset and position information.
func (e *Exchange) GetCMIncomeHistory(ctx context.Context, symbol currency.Pair, incomeType string, startTime, endTime time.Time, limit int64) ([]*IncomeItem, error) {
	return e.getUMCMIncomeHistory(ctx, symbol, incomeType, "/papi/v1/cm/income", startTime, endTime, limit, pmGetCMIncomeHistoryRate)
}

func (e *Exchange) getUMCMIncomeHistory(ctx context.Context, symbol currency.Pair, incomeType, path string, startTime, endTime time.Time, limit int64, endpointLimit request.EndpointLimit) ([]*IncomeItem, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	if incomeType == "" {
		params.Set("incomeType", incomeType)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	if !startTime.IsZero() {
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
	}
	if !endTime.IsZero() {
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []*IncomeItem
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, path, params, endpointLimit, nil, &resp)
}

// GetUMAccountDetail get current UM account asset and position information.
func (e *Exchange) GetUMAccountDetail(ctx context.Context) (*AccountDetail, error) {
	return e.getUMCMAccountDetail(ctx, "/papi/v1/um/account", pmGetUMAccountDetailRate)
}

// GetCMAccountDetail gets current CM account asset and position information.
func (e *Exchange) GetCMAccountDetail(ctx context.Context) (*AccountDetail, error) {
	return e.getUMCMAccountDetail(ctx, "/papi/v1/cm/account", pmGetCMAccountDetailRate)
}

func (e *Exchange) getUMCMAccountDetail(ctx context.Context, path string, endpointLimit request.EndpointLimit) (*AccountDetail, error) {
	var resp *AccountDetail
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, path, nil, endpointLimit, nil, &resp)
}

// ChangeAutoRepayFuturesStatus change Auto-repay-futures Status
func (e *Exchange) ChangeAutoRepayFuturesStatus(ctx context.Context, autoRepay bool) (string, error) {
	return e.changeAutoRepayFuturesStatus(ctx, autoRepay, exchange.RestFuturesSupplementary, "/papi/v1/repay-futures-switch", pmChangeAutoRepayFuturesStatusRate)
}

func (e *Exchange) changeAutoRepayFuturesStatus(ctx context.Context, autoRepay bool, exchURL exchange.URL, path string, epl request.EndpointLimit) (string, error) {
	params := url.Values{}
	if autoRepay {
		params.Set("autoRepay", "true")
	} else {
		params.Set("autoRepay", "false")
	}
	resp := &struct {
		Message string `json:"msg"`
	}{}
	return resp.Message, e.SendAuthHTTPRequest(ctx, exchURL, http.MethodPost, path, params, epl, nil, &resp)
}

// GetAutoRepayFuturesStatus query Auto-repay-futures Status
func (e *Exchange) GetAutoRepayFuturesStatus(ctx context.Context) (*AutoRepayStatus, error) {
	var resp *AutoRepayStatus
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/repay-futures-switch", nil, pmGetAutoRepayFuturesStatusRate, nil, &resp)
}

// RepayFuturesNegativeBalance repay futures Negative Balance
func (e *Exchange) RepayFuturesNegativeBalance(ctx context.Context) (string, error) {
	resp := &struct {
		Message string `json:"msg"`
	}{}
	return resp.Message, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/repay-futures-negative-balance", nil, pmRepayFuturesNegativeBalanceRate, nil, &resp)
}

// GetUMPositionADLQuantileEstimation retrieves ADL Quantile Estimations for a symbol or symbols
//
// Values 0, 1, 2, 3, 4 shows the queue position and possibility of ADL from low to high.
// For positions of the symbol are in One-way Mode or isolated margined in Hedge Mode, "LONG", "SHORT", and "BOTH" will be returned to show the positions' adl quantiles of different position sides.
func (e *Exchange) GetUMPositionADLQuantileEstimation(ctx context.Context, symbol currency.Pair) ([]*ADLQuantileEstimation, error) {
	return e.getUMCMPositionADLQuantileEstimation(ctx, symbol, "/papi/v1/um/adlQuantile", pmGetUMPositionADLQuantileEstimationRate)
}

// GetCMPositionADLQuantileEstimation retrieves Coin Margined Futures position ADL Quantile estimation for symbol or symbols
func (e *Exchange) GetCMPositionADLQuantileEstimation(ctx context.Context, symbol currency.Pair) ([]*ADLQuantileEstimation, error) {
	return e.getUMCMPositionADLQuantileEstimation(ctx, symbol, "/papi/v1/cm/adlQuantile", pmGetCMPositionADLQuantileEstimationRate)
}

func (e *Exchange) getUMCMPositionADLQuantileEstimation(ctx context.Context, symbol currency.Pair, path string, endpointLimit request.EndpointLimit) ([]*ADLQuantileEstimation, error) {
	params := url.Values{}
	if !symbol.IsEmpty() {
		params.Set("symbol", symbol.String())
	}
	var resp []*ADLQuantileEstimation
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, path, params, endpointLimit, nil, &resp)
}

// GetUserRateLimits retrieves list of user's account rate-limit information
func (e *Exchange) GetUserRateLimits(ctx context.Context) ([]*RateLimitInfo, error) {
	var resp []*RateLimitInfo
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestSpotSupplementary, http.MethodGet, "/papi/v1/rateLimit/order", nil, request.UnAuth, nil, &resp)
}

// GetPortfolioMarginAssetIndexPrice query portfolio margin asset index price
func (e *Exchange) GetPortfolioMarginAssetIndexPrice(ctx context.Context, asset currency.Code) ([]*PortfolioMarginAssetIndexPrice, error) {
	if asset.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("asset", asset.String())
	var resp []*PortfolioMarginAssetIndexPrice
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, common.EncodeURLValues("/sapi/v1/portfolio/asset-index-price", params), request.UnAuth, &resp)
}
