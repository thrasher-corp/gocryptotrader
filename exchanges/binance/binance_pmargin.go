package binance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
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
func (b *Binance) NewUMOrder(ctx context.Context, arg *UMOrderParam) (*UM_CM_Order, error) {
	return b.newUMCMOrder(ctx, arg, "/papi/v1/um/order")
}

// NewCMOrder send in a new Coin margined order/orders.
func (b *Binance) NewCMOrder(ctx context.Context, arg *UMOrderParam) (*UM_CM_Order, error) {
	return b.newUMCMOrder(ctx, arg, "/papi/v1/cm/order")
}

func (b *Binance) newUMCMOrder(ctx context.Context, arg *UMOrderParam, path string) (*UM_CM_Order, error) {
	if arg == nil || (*arg) == (UMOrderParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	arg.OrderType = strings.ToUpper(arg.OrderType)
	if arg.OrderType == "limit" {
		if arg.TimeInForce == "" {
			return nil, errTimestampInfoRequired
		}
		if arg.Quantity <= 0 {
			return nil, order.ErrAmountBelowMin
		}
		if arg.Price <= 0 {
			return nil, order.ErrPriceBelowMin
		}
	} else if arg.OrderType == "MARKET" {
		if arg.Quantity <= 0 {
			return nil, order.ErrAmountBelowMin
		}
	} else {
		return nil, order.ErrUnsupportedOrderType
	}
	var resp *UM_CM_Order
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, path, nil, spotDefaultRate, arg, &resp)
}

// NewMarginOrder places a new cross margin order
func (b *Binance) NewMarginOrder(ctx context.Context, arg *MarginOrderParam) (*MarginOrderResp, error) {
	if arg == nil || *arg == (MarginOrderParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}

	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	var resp *MarginOrderResp
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/margin/order", nil, spotDefaultRate, arg, &resp)
}

// MarginAccountBorrow apply for margin loan
func (b *Binance) MarginAccountBorrow(ctx context.Context, ccy currency.Code, amount float64) (string, error) {
	return b.marginAccountBorrowRepay(ctx, ccy, amount, "/papi/v1/marginLoan")
}

// MarginAccountRepay repay for margin loan
func (b *Binance) MarginAccountRepay(ctx context.Context, ccy currency.Code, amount float64) (string, error) {
	return b.marginAccountBorrowRepay(ctx, ccy, amount, "/papi/v1/repayLoan")
}

func (b *Binance) marginAccountBorrowRepay(ctx context.Context, ccy currency.Code, amount float64, path string) (string, error) {
	if ccy.IsEmpty() {
		return "", currency.ErrCurrencyCodeEmpty
	}
	if amount <= 0 {
		return "", order.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("asset", ccy.String())
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	resp := &struct {
		TransactionID string `json:"tranId"`
	}{}
	return resp.TransactionID, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, path, params, spotDefaultRate, nil, &resp)
}

// MarginAccountNewOCO sends a new OCO order for a margin account.
func (b *Binance) MarginAccountNewOCO(ctx context.Context, arg *OCOOrderParam) (*OCOOrderResponse, error) {
	if arg == nil || *arg == (OCOOrderParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Amount <= 0 {
		return nil, order.ErrAmountBelowMin
	}
	if arg.Price <= 0 {
		return nil, order.ErrPriceBelowMin
	}
	if arg.StopPrice <= 0 {
		return nil, fmt.Errorf("%w, stopPrice is required", order.ErrPriceBelowMin)
	}
	var resp *OCOOrderResponse
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, "/papi/v1/margin/order/oco", nil, spotDefaultRate, arg, &resp)
}

// NewUMConditionalOrder places a new conditional USDT margined order
func (b *Binance) NewUMConditionalOrder(ctx context.Context, arg *ConditionalOrderParam) (*ConditionalOrder, error) {
	return b.placeConditionalOrder(ctx, arg, "/papi/v1/um/conditional/order")
}

// NewCMConditionalOrder posts a new coin margined futures conditional order.
func (b *Binance) NewCMConditionalOrder(ctx context.Context, arg *ConditionalOrderParam) (*ConditionalOrder, error) {
	return b.placeConditionalOrder(ctx, arg, "/papi/v1/cm/conditional/order")
}
func (b *Binance) placeConditionalOrder(ctx context.Context, arg *ConditionalOrderParam, path string) (*ConditionalOrder, error) {
	if arg == nil || *arg == (ConditionalOrderParam{}) {
		return nil, common.ErrNilPointer
	}
	if arg.Symbol == "" {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.StrategyType == "" {
		return nil, errors.New("strategy type is required")
	}
	var resp *ConditionalOrder
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodPost, path, nil, spotDefaultRate, arg, &resp)
}

// -------------------------------------------- Cancel Order Endpoints  ----------------------------------------------------

// CancelCMOrder cancels an active Coin Margined Futures limit order.
func (b *Binance) CancelCMOrder(ctx context.Context, symbol, origClientOrderID string, orderID int64) (*UM_CM_Order, error) {
	return b.cancelOrder(ctx, symbol, origClientOrderID, "/papi/v1/cm/order", orderID)
}

// CancelUMOrder cancels an active USDT Margined Futures limit order.
func (b *Binance) CancelUMOrder(ctx context.Context, symbol, origClientOrderID string, orderID int64) (*UM_CM_Order, error) {
	return b.cancelOrder(ctx, symbol, origClientOrderID, "/papi/v1/um/order", orderID)
}

func (b *Binance) cancelOrder(ctx context.Context, symbol, origClientOrderID, path string, orderID int64) (*UM_CM_Order, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if orderID == 0 && origClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	var resp *UM_CM_Order
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, path, params, spotDefaultRate, nil, &resp)
}

// CancelAllUMOrders cancels all active USDT Margined Futures limit orders on specific symbol
func (b *Binance) CancelAllUMOrders(ctx context.Context, symbol string) (*SuccessResponse, error) {
	return b.cancelAllUMCMOrders(ctx, symbol, "/papi/v1/um/allOpenOrders")
}

// CancelAllCMOrders cancels all active Coin Margined Futures limit orders on specific symbol
func (b *Binance) CancelAllCMOrders(ctx context.Context, symbol string) (*SuccessResponse, error) {
	return b.cancelAllUMCMOrders(context.Background(), symbol, "/papi/v1/cm/allOpenOrders")
}

func (b *Binance) cancelAllUMCMOrders(ctx context.Context, symbol, path string) (*SuccessResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *SuccessResponse
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, path, params, spotDefaultRate, nil, &resp)
}

// CancelMarginAccountOrder cancels margin account order
func (b *Binance) CancelMarginAccountOrder(ctx context.Context, symbol, origClientOrderID string, orderID int64) (*MarginOrderResp, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if orderID == 0 && origClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	var resp *MarginOrderResp
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, "/papi/v1/margin/order", params, spotDefaultRate, nil, &resp)
}

// CancelAllMarginOpenOrdersBySymbol cancels all open margin account orders of a specific symbol.
func (b *Binance) CancelAllMarginOpenOrdersBySymbol(ctx context.Context, symbol string) (MarginAccOrdersList, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp MarginAccOrdersList
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, "/papi/v1/margin/allOpenOrders", params, spotDefaultRate, nil, &resp)
}

// CancelMarginAccountOCOOrders cancels margin account OCO orders.
func (b *Binance) CancelMarginAccountOCOOrders(ctx context.Context, symbol, listClientOrderID, newClientOrderID string, orderListID int64) (*OCOOrderResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if listClientOrderID != "" {
		params.Set("listClientOrderId", listClientOrderID)
	}
	if newClientOrderID != "" {
		params.Set("newClientOrderId", newClientOrderID)
	}
	if orderListID > 0 {
		params.Set("orderListId", strconv.FormatInt(orderListID, 10))
	}
	var resp *OCOOrderResponse
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, "/papi/v1/margin/orderList", params, spotDefaultRate, nil, &resp)
}

// CancelUMConditionalOrder cancels a USDT margind futures conditional order
func (b *Binance) CancelUMConditionalOrder(ctx context.Context, symbol, newClientStrategyID string, strategyID int64) (*ConditionalOrder, error) {
	return b.cancelUMCMConditionalOrder(ctx, symbol, newClientStrategyID, "/papi/v1/um/conditional/order", strategyID)
}

// CancelCMConditionalOrder cancels a Coin margined futures conditional order
func (b *Binance) CancelCMConditionalOrder(ctx context.Context, symbol, newClientStrategyID string, strategyID int64) (*ConditionalOrder, error) {
	return b.cancelUMCMConditionalOrder(ctx, symbol, newClientStrategyID, "/papi/v1/cm/conditional/order", strategyID)
}

func (b *Binance) cancelUMCMConditionalOrder(ctx context.Context, symbol, newClientStrategyID, path string, strategyID int64) (*ConditionalOrder, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if strategyID == 0 && newClientStrategyID == "" {
		return nil, fmt.Errorf("%w, either strategyId or newClientStrategyId is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if strategyID > 0 {
		params.Set("strategyId", strconv.FormatInt(strategyID, 10))
	}
	if newClientStrategyID != "" {
		params.Set("newClientStrategyId", newClientStrategyID)
	}
	var resp *ConditionalOrder
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, path, params, spotDefaultRate, nil, &resp)
}

// CancelAllUMOpenConditionalOrders cancels all open conditional USDT margined orders
func (b *Binance) CancelAllUMOpenConditionalOrders(ctx context.Context, symbol string) (*SuccessResponse, error) {
	return b.cancelAllUMCMOpenConditionalOrders(ctx, symbol, "/papi/v1/um/conditional/allOpenOrders")
}

// CancelAllCMOpenConditionalOrders cancels all open conditional Coin margined orders
func (b *Binance) CancelAllCMOpenConditionalOrders(ctx context.Context, symbol string) (*SuccessResponse, error) {
	return b.cancelAllUMCMOpenConditionalOrders(ctx, symbol, "/papi/v1/cm/conditional/allOpenOrders")
}

func (b *Binance) cancelAllUMCMOpenConditionalOrders(ctx context.Context, symbol, path string) (*SuccessResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	var resp *SuccessResponse
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodDelete, path, params, spotDefaultRate, nil, &resp)
}

// --------------------------------------------------------   Query Order Endpoints  --------------------------------------------------------

// GetUMOrder check an USDT Margined order's status
// Orders can not be found if the order status is CANCELED or EXPIRED
func (b *Binance) GetUMOrder(ctx context.Context, symbol, origClientOrderID string, orderID int64) (*UM_CM_Order, error) {
	return b.getUM_CMOrder(ctx, symbol, origClientOrderID, "/papi/v1/um/order", orderID)
}

// GetUMOpenOrder get current UM open order
func (b *Binance) GetUMOpenOrder(ctx context.Context, symbol, origClientOrderID string, orderID int64) (*UM_CM_Order, error) {
	return b.getUM_CMOrder(ctx, symbol, origClientOrderID, "/papi/v1/um/openOrder", orderID)
}

func (b *Binance) getUM_CMOrder(ctx context.Context, symbol, origClientOrderID, path string, orderID int64) (*UM_CM_Order, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if orderID == 0 && origClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	var resp *UM_CM_Order
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, path, params, spotDefaultRate, nil, &resp)
}

// GetAllUMOpenOrders retrieves all open USDT margined orders.
// If no symbol is provided, it will load all open USDT orders, taking more ratelimit weight than the ordinary endpoints.
func (b *Binance) GetAllUMOpenOrders(ctx context.Context, symbol string) ([]UM_CM_Order, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	return b.getUMOrders(ctx, symbol, "/papi/v1/um/openOrders", time.Time{}, time.Time{}, 0, 0)
}

// GetAllUMOrders retrieves all USDT margined orders except for
// 1. CANCELED or EXPIRED orders.
// 2. Orders with not fill.
//  3. Order Created later earlier than three days from now.
//
// If orderId is set, it will get orders >= that orderId. Otherwise most recent orders are returned.
// The query time period must be less then 7 days.
func (b *Binance) GetAllUMOrders(ctx context.Context, symbol string, startTime, endTime time.Time, startingOrderID, limit int64) ([]UM_CM_Order, error) {
	return b.getUMOrders(ctx, symbol, "/papi/v1/um/allOrders", startTime, endTime, startingOrderID, limit)
}

func (b *Binance) getUMOrders(ctx context.Context, symbol, path string, startTime, endTime time.Time, startingOrderID, limit int64) ([]UM_CM_Order, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if startingOrderID > 0 {
		params.Set("orderId", strconv.FormatInt(startingOrderID, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []UM_CM_Order
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, path, params, spotDefaultRate, nil, &resp)
}

// GetCMOrder retrieves Coin Margined order instance.
func (b *Binance) GetCMOrder(ctx context.Context, symbol, origClientOrderID string, orderID int64) (*UM_CM_Order, error) {
	return b.getUM_CMOrder(ctx, symbol, origClientOrderID, "/papi/v1/cm/order", orderID)
}

// GetCMOpenOrder retrieves Coin Margined open order instance.
func (b *Binance) GetCMOpenOrder(ctx context.Context, symbol, origClientOrderID string, orderID int64) (*UM_CM_Order, error) {
	return b.getUM_CMOrder(ctx, symbol, origClientOrderID, "/papi/v1/cm/openOrder", orderID)
}

// GetAllCMOpenOrders retrieves all open Coin Margined futures orders on a symbol.
func (b *Binance) GetAllCMOpenOrders(ctx context.Context, symbol, pair string) ([]UM_CM_Order, error) {
	return b.getCMOrders(ctx, symbol, pair, "/papi/v1/cm/openOrders", time.Time{}, time.Time{}, 0, 0)
}

// GetAllCMOrders get all account CM orders; active, canceled, or filled.
//
// Either symbol or pair must be sent.
// If orderId is set, it will get orders >= that orderId. Otherwise most recent orders are returned.
// These orders will not be found:
// - order status is CANCELED or EXPIRED, AND
// - order has NO filled trade, AND
// - created time + 3 days < current time
func (b *Binance) GetAllCMOrders(ctx context.Context, symbol, pair string, startTime, endTime time.Time, startingOrderID, limit int64) ([]UM_CM_Order, error) {
	return b.getCMOrders(ctx, symbol, pair, "/papi/v1/cm/allOrders", startTime, endTime, startingOrderID, limit)
}

func (b *Binance) getCMOrders(ctx context.Context, symbol, pair, path string, startTime, endTime time.Time, startingOrderID, limit int64) ([]UM_CM_Order, error) {
	if symbol == "" && pair == "" {
		return nil, fmt.Errorf("%w either symbol or pair is required", currency.ErrSymbolStringEmpty)
	}
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if pair != "" {
		params.Set("pair", pair)
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if startingOrderID > 0 {
		params.Set("orderId", strconv.FormatInt(startingOrderID, 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []UM_CM_Order
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, path, params, spotDefaultRate, nil, &resp)
}

// GetOpenUMConditionalOrder retrieves a conditional USDT margined order
func (b *Binance) GetOpenUMConditionalOrder(ctx context.Context, symbol, newClientStrategyID string, strategyID int64) (*ConditionalOrder, error) {
	return b.getOpenUMCMConditionalOrder(context.Background(), symbol, newClientStrategyID, "/papi/v1/um/conditional/openOrder", strategyID)
}

func (b *Binance) getOpenUMCMConditionalOrder(ctx context.Context, symbol, newClientStrategyID, path string, strategyID int64) (*ConditionalOrder, error) {
	if strategyID == 0 && newClientStrategyID == "" {
		return nil, fmt.Errorf("%w, either strategyId or newClientStrategyId is required", order.ErrOrderIDNotSet)
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if strategyID > 0 {
		params.Set("strategyId", strconv.FormatInt(strategyID, 10))
	}
	if newClientStrategyID != "" {
		params.Set("newClientStrategyId", newClientStrategyID)
	}
	var resp *ConditionalOrder
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestSpot, http.MethodGet, path, params, spotDefaultRate, nil, &resp)
}

// GetAllUMOpenConditionalOrders retrieves all open conditional orders on a symbol.
func (b *Binance) GetAllUMOpenConditionalOrders(ctx context.Context, symbol string) ([]ConditionalOrder, error) {
	return b.getAllUMCMOrders(ctx, symbol, "/papi/v1/um/conditional/openOrders", "", time.Time{}, time.Time{}, 0, 0)
}

// GetAllUMConditionalOrderHistory retrieves all conditional order history a symbol.
func (b *Binance) GetAllUMConditionalOrderHistory(ctx context.Context, symbol, newClientStrategyID string, strategyID int64) ([]ConditionalOrder, error) {
	return b.getAllUMCMOrders(ctx, symbol, "/papi/v1/um/conditional/orderHistory", newClientStrategyID, time.Time{}, time.Time{}, strategyID, 0)
}

// GetAllUMConditionalOrders retrieves conditional orders.
func (b *Binance) GetAllUMConditionalOrders(ctx context.Context, symbol string, startTime, endTime time.Time, strategyID, limit int64) ([]ConditionalOrder, error) {
	return b.getAllUMCMOrders(ctx, symbol, "/papi/v1/um/conditional/allOrders", "", startTime, endTime, strategyID, limit)
}

func (b *Binance) getAllUMCMOrders(ctx context.Context, symbol, path, newClientStrategyID string, startTime, endTime time.Time, strategyID, limit int64) ([]ConditionalOrder, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	if strategyID > 0 {
		params.Set("strategyId", strconv.FormatInt(strategyID, 10))
	}
	if newClientStrategyID != "" {
		params.Set("newClientStrategyId", newClientStrategyID)
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
	var resp []ConditionalOrder
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, path, params, spotDefaultRate, nil, &resp)
}

// GetOpenCMConditionalOrder get current Coin Margined open conditional order
func (b *Binance) GetOpenCMConditionalOrder(ctx context.Context, symbol, newClientStrategyID string, strategyID int64) (*ConditionalOrder, error) {
	return b.getOpenUMCMConditionalOrder(context.Background(), symbol, newClientStrategyID, "", strategyID)
}

// GetAllCMOpenConditionalOrders retrieves all open conditional orders on a symbol.
func (b *Binance) GetAllCMOpenConditionalOrders(ctx context.Context, symbol string) ([]ConditionalOrder, error) {
	return b.getAllUMCMOrders(ctx, symbol, "/papi/v1/cm/conditional/openOrders", "", time.Time{}, time.Time{}, 0, 0)
}

// GetAllCMConditionalOrderHistory retrieves all conditional order history a symbol.
func (b *Binance) GetAllCMConditionalOrderHistory(ctx context.Context, symbol, newClientStrategyID string, strategyID int64) ([]ConditionalOrder, error) {
	return b.getAllUMCMOrders(ctx, symbol, "/papi/v1/cm/conditional/orderHistory", newClientStrategyID, time.Time{}, time.Time{}, strategyID, 0)
}

// GetAllCMConditionalOrders retrieves conditional orders.
func (b *Binance) GetAllCMConditionalOrders(ctx context.Context, symbol string, startTime, endTime time.Time, strategyID, limit int64) ([]ConditionalOrder, error) {
	return b.getAllUMCMOrders(ctx, symbol, "/papi/v1/cm/conditional/allOrders", "", startTime, endTime, strategyID, limit)
}

// GetMarginAccountOrder retrieves margin account order.
func (b *Binance) GetMarginAccountOrder(ctx context.Context, symbol, origClientOrderID string, orderID int64) (*MarginOrder, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	if orderID == 0 && origClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	var resp *MarginOrder
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/order", params, spotDefaultRate, nil, &resp)
}

// GetCurrentMarginOpenOrder retrieves an open order.
// If the symbol is not sent, orders for all symbols will be returned in an array.
func (b *Binance) GetCurrentMarginOpenOrder(ctx context.Context, symbol string) ([]MarginOrder, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp []MarginOrder
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/openOrders", params, spotDefaultRate, nil, &resp)
}

// GetAllMarginAccountOrders retrieves all margin account orders
func (b *Binance) GetAllMarginAccountOrders(ctx context.Context, symbol string, startTime, endTime time.Time, orderID, limit int64) ([]MarginOrder, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []MarginOrder
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/allOrders", params, spotDefaultRate, nil, &resp)
}

// GetMarginAccountOCO retrieves a specific OCO based on provided optional parameters.
func (b *Binance) GetMarginAccountOCO(ctx context.Context, orderListID int64, origClientOrderID string) (*OCOOrderResponse, error) {
	params := url.Values{}
	if orderListID > 0 {
		params.Set("orderListId", strconv.FormatInt(orderListID, 10))
	}
	if origClientOrderID != "" {
		params.Set("origClientOrderId", origClientOrderID)
	}
	var resp *OCOOrderResponse
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/orderList", params, spotDefaultRate, nil, &resp)
}

// GetMarginAccountAllOCO retrieves all OCO for a specific margin account based on provided optional parameters
func (b *Binance) GetMarginAccountAllOCO(ctx context.Context, startTime, endTime time.Time, fromID, limit int64) ([]OCOOrder, error) {
	params := url.Values{}
	if fromID > 0 {
		params.Set("fromId", strconv.FormatInt(fromID, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []OCOOrder
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/allOrderList", params, spotDefaultRate, nil, &resp)
}

// GetMarginAccountsOpenOCO retrieves a margin account open OCO order
func (b *Binance) GetMarginAccountsOpenOCO(ctx context.Context) ([]OCOOrder, error) {
	var resp []OCOOrder
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/openOrderList", nil, spotDefaultRate, nil, &resp)
}

// GetMarginAccountTradeList retrieves margin account trade list
func (b *Binance) GetMarginAccountTradeList(ctx context.Context, symbol string, startTime, endTime time.Time, orderID, fromID, limit int64) ([]MarginAccountTradeItem, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	if orderID != 0 {
		params.Set("orderId", strconv.FormatInt(orderID, 10))
	}
	if fromID != 0 {
		params.Set("fromId", strconv.FormatInt(fromID, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params.Set("startTime", strconv.FormatInt(startTime.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(endTime.UnixMilli(), 10))
	}
	if limit > 0 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	var resp []MarginAccountTradeItem
	return resp, b.SendAuthHTTPRequest(ctx, exchange.RestFuturesSupplementary, http.MethodGet, "/papi/v1/margin/myTrades", params, spotDefaultRate, nil, &resp)
}

//  ---------------------------------------------------  Account Endpoints  -------------------------------------------------------------------------------------
