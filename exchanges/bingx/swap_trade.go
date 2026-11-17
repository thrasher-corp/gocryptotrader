package bingx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	swapTradeBatchOrdersLimit = 5
	swapTradeBatchCancelLimit = 10
)

var (
	errSwapTooManyItems                 = errors.New("list holds more items than BingX takes in one request")
	errSwapClosePositionWithReduceOnly  = errors.New("close position cannot be sent with reduce only")
	errSwapReduceOnlyInHedgeMode        = errors.New("reduce only cannot be sent with a LONG or SHORT position side")
	errSwapClientOrderIDUnsupportedType = errors.New("client order ID can only be sent with a LIMIT or MARKET order")
	errSwapCancelReplaceModeRequired    = errors.New("cancel replace mode required")
	errSwapMarginTypeRequired           = errors.New("margin type required")
	errSwapLeverageRequired             = errors.New("leverage required")
	errSwapMarginAdjustmentTypeRequired = errors.New("margin adjustment type required")
	errSwapTradingUnitRequired          = errors.New("trading unit required")
	errSwapPositionIDRequired           = errors.New("position ID required")
	errSwapCancelAllAfterTypeRequired   = errors.New("cancel all after type required")
	errSwapMainOrderIDRequired          = errors.New("main order ID required")
	errSwapTWAPPriceTypeRequired        = errors.New("price type required")
	errSwapTWAPPriceVarianceRequired    = errors.New("price variance required")
	errSwapTWAPIntervalRequired         = errors.New("interval required")
	errSwapAssetModeRequired            = errors.New("asset mode required")
	errSwapReverseTypeRequired          = errors.New("reverse type required")
)

// PlaceSwapTestOrder sends an order to Test Order, which validates it as Place Order does and replies with the order it
// would place, without placing it
func (e *Exchange) PlaceSwapTestOrder(ctx context.Context, arg *SwapOrderRequest) (*SwapNewOrderResponse, error) {
	return e.swapTradeNewOrder(ctx, "/openApi/swap/v2/trade/order/test", swapTestOrderRate, arg)
}

// PlaceSwapOrder places a USDT-M perpetual order with Place Order. BingX refuses orders for an offline symbol with
// code 109418, and an account holds at most 200 open orders, conditional orders included
func (e *Exchange) PlaceSwapOrder(ctx context.Context, arg *SwapOrderRequest) (*SwapNewOrderResponse, error) {
	return e.swapTradeNewOrder(request.WithRetryNotAllowed(ctx), "/openApi/swap/v2/trade/order", swapPlaceOrderRate, arg)
}

// swapTradeNewOrder sends an order to Place Order or Test Order, which take and return the same fields
func (e *Exchange) swapTradeNewOrder(ctx context.Context, path string, f request.EndpointLimit, arg *SwapOrderRequest) (*SwapNewOrderResponse, error) {
	item, err := e.swapTradeOrder(arg)
	if err != nil {
		return nil, err
	}
	var resp *SwapNewOrderDataResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, path, swapTradeOrderValues(item), f, &resp); err != nil {
		return nil, err
	}
	if resp.Order == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Order, nil
}

// swapTradeOrder validates an order and returns it in the form BingX takes, which single and batch requests share
func (e *Exchange) swapTradeOrder(arg *SwapOrderRequest) (*SwapBatchOrderRequest, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Type == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.ReduceOnly && arg.ClosePosition {
		return nil, errSwapClosePositionWithReduceOnly
	}
	if arg.ReduceOnly && (arg.PositionSide == PositionSideLong || arg.PositionSide == PositionSideShort) {
		return nil, errSwapReduceOnlyInHedgeMode
	}
	if err := swapTradeCheckClientOrderID(arg.Type, arg.ClientOrderID); err != nil {
		return nil, err
	}
	symbol, err := e.FormatSymbol(arg.Symbol, asset.USDTMarginedFutures)
	if err != nil {
		return nil, err
	}
	stopLoss, err := swapTradeTakeProfitStopLoss(arg.StopLoss)
	if err != nil {
		return nil, fmt.Errorf("error encoding stop loss: %w", err)
	}
	takeProfit, err := swapTradeTakeProfitStopLoss(arg.TakeProfit)
	if err != nil {
		return nil, fmt.Errorf("error encoding take profit: %w", err)
	}
	item := &SwapBatchOrderRequest{
		Symbol:             symbol,
		Type:               arg.Type,
		Side:               arg.Side,
		PositionSide:       arg.PositionSide,
		Price:              arg.Price,
		Quantity:           arg.Quantity,
		QuoteOrderQuantity: arg.QuoteOrderQuantity,
		StopPrice:          arg.StopPrice,
		PriceRate:          arg.PriceRate,
		WorkingType:        arg.WorkingType,
		StopLoss:           stopLoss,
		TakeProfit:         takeProfit,
		ClientOrderID:      arg.ClientOrderID,
		TimeInForce:        arg.TimeInForce,
		ActivationPrice:    arg.ActivationPrice,
		StopGuaranteed:     arg.StopGuaranteed,
		PositionID:         arg.PositionID,
	}
	if arg.ReduceOnly {
		item.ReduceOnly = "true"
	}
	if arg.ClosePosition {
		item.ClosePosition = "true"
	}
	return item, nil
}

// swapTradeCheckClientOrderID returns an error for a client order ID sent with an order of a type other than LIMIT or
// MARKET, the only types BingX documents taking one for
func swapTradeCheckClientOrderID(orderType OrderType, clientOrderID string) error {
	if clientOrderID != "" && orderType != OrderTypeLimit && orderType != OrderTypeMarket {
		return fmt.Errorf("%w: %s order", errSwapClientOrderIDUnsupportedType, orderType)
	}
	return nil
}

// swapTradeTakeProfitStopLoss returns an attached take profit or stop loss as the JSON string BingX takes, or an empty
// string for none
func swapTradeTakeProfitStopLoss(arg *SwapTakeProfitStopLossRequest) (string, error) {
	if arg == nil {
		return "", nil
	}
	encoded, err := json.Marshal(arg)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// swapTradeOrderValues returns the query parameters of an order swapTradeOrder validated
func swapTradeOrderValues(item *SwapBatchOrderRequest) url.Values {
	params := url.Values{}
	params.Set("symbol", item.Symbol)
	params.Set("type", string(item.Type))
	params.Set("side", string(item.Side))
	if item.PositionSide != "" {
		params.Set("positionSide", string(item.PositionSide))
	}
	if item.ReduceOnly != "" {
		params.Set("reduceOnly", item.ReduceOnly)
	}
	if item.Price != 0 {
		params.Set("price", strconv.FormatFloat(item.Price, 'f', -1, 64))
	}
	if item.Quantity != 0 {
		params.Set("quantity", strconv.FormatFloat(item.Quantity, 'f', -1, 64))
	}
	if item.QuoteOrderQuantity != 0 {
		params.Set("quoteOrderQty", strconv.FormatFloat(item.QuoteOrderQuantity, 'f', -1, 64))
	}
	if item.StopPrice != 0 {
		params.Set("stopPrice", strconv.FormatFloat(item.StopPrice, 'f', -1, 64))
	}
	if item.PriceRate != 0 {
		params.Set("priceRate", strconv.FormatFloat(item.PriceRate, 'f', -1, 64))
	}
	if item.WorkingType != "" {
		params.Set("workingType", string(item.WorkingType))
	}
	if item.StopLoss != "" {
		params.Set("stopLoss", item.StopLoss)
	}
	if item.TakeProfit != "" {
		params.Set("takeProfit", item.TakeProfit)
	}
	if item.ClientOrderID != "" {
		params.Set("clientOrderId", item.ClientOrderID)
	}
	if item.TimeInForce != "" {
		params.Set("timeInForce", string(item.TimeInForce))
	}
	if item.ClosePosition != "" {
		params.Set("closePosition", item.ClosePosition)
	}
	if item.ActivationPrice != 0 {
		params.Set("activationPrice", strconv.FormatFloat(item.ActivationPrice, 'f', -1, 64))
	}
	if item.StopGuaranteed != "" {
		params.Set("stopGuaranteed", string(item.StopGuaranteed))
	}
	if item.PositionID != 0 {
		params.Set("positionId", strconv.FormatUint(item.PositionID, 10))
	}
	return params
}

// PlaceSwapBatchOrders places up to five orders with Place Multiple Orders, which processes them concurrently, so they
// may match in any order. BingX replies with the orders it accepted
func (e *Exchange) PlaceSwapBatchOrders(ctx context.Context, args []SwapOrderRequest) ([]SwapNewOrderResponse, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	if err := swapTradeCheckListLength("orders", len(args), swapTradeBatchOrdersLimit); err != nil {
		return nil, err
	}
	batch := make([]*SwapBatchOrderRequest, len(args))
	for i := range args {
		item, err := e.swapTradeOrder(&args[i])
		if err != nil {
			return nil, fmt.Errorf("order %d: %w", i, err)
		}
		batch[i] = item
	}
	batchOrders, err := json.Marshal(batch)
	if err != nil {
		return nil, fmt.Errorf("error encoding batch orders: %w", err)
	}
	params := url.Values{}
	params.Set("batchOrders", string(batchOrders))
	var resp *SwapNewOrdersDataResponse
	if err := e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestUSDTMargined, http.MethodPost, "/openApi/swap/v2/trade/batchOrders", params, swapPlaceBatchOrdersRate, &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

// swapTradeCheckListLength returns an error when a list holds more items than BingX takes in one request
func swapTradeCheckListLength(items string, length, maximum int) error {
	if length > maximum {
		return fmt.Errorf("%w: %d %s, at most %d", errSwapTooManyItems, length, items, maximum)
	}
	return nil
}

// ModifySwapOrder changes the quantity of a LIMIT order with Modify Order, the only change it supports
func (e *Exchange) ModifySwapOrder(ctx context.Context, arg *ModifySwapOrderRequest) (*ModifySwapOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.OrderID == 0 && arg.ClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.Quantity <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params, err := e.swapTradeSymbolParams(arg.Symbol)
	if err != nil {
		return nil, err
	}
	if arg.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(arg.OrderID, 10))
	}
	if arg.ClientOrderID != "" {
		params.Set("clientOrderId", arg.ClientOrderID)
	}
	params.Set("quantity", strconv.FormatFloat(arg.Quantity, 'f', -1, 64))
	var resp *ModifySwapOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/openApi/swap/v1/trade/amend", params, swapModifyOrderRate, &resp)
}

// swapTradeSymbolParams returns parameters holding a required symbol
func (e *Exchange) swapTradeSymbolParams(symbol currency.Pair) (url.Values, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := url.Values{}
	if err := e.swapTradeSetSymbol(params, symbol); err != nil {
		return nil, err
	}
	return params, nil
}

// swapTradeSetSymbol sets the symbol parameter when a pair is given, which endpoints covering every symbol take as a
// filter
func (e *Exchange) swapTradeSetSymbol(params url.Values, symbol currency.Pair) error {
	if symbol.IsEmpty() {
		return nil
	}
	symbolValue, err := e.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	params.Set("symbol", symbolValue)
	return nil
}

// CloseAllSwapPositions closes the positions of a symbol, or of every symbol when symbol is empty, at market price with
// Close All Positions
func (e *Exchange) CloseAllSwapPositions(ctx context.Context, symbol currency.Pair) (*CloseAllSwapPositionsResponse, error) {
	params := url.Values{}
	if err := e.swapTradeSetSymbol(params, symbol); err != nil {
		return nil, err
	}
	var resp *CloseAllSwapPositionsResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestUSDTMargined, http.MethodPost, "/openApi/swap/v2/trade/closeAllPositions", params, swapCloseAllPositionsRate, &resp)
}

// CancelSwapOrder cancels an open order with Cancel Order, by its order ID or, when orderID is 0, its client order ID.
// BingX cancels an order at most once a second and asks that cancellations not be resubmitted, so a cancellation whose
// reply times out is not sent again
func (e *Exchange) CancelSwapOrder(ctx context.Context, symbol currency.Pair, orderID uint64, clientOrderID string) (*SwapOrderResponse, error) {
	params, err := e.swapTradeOrderIDParams(symbol, orderID, clientOrderID)
	if err != nil {
		return nil, err
	}
	return e.swapTradeOrderReply(request.WithRetryNotAllowed(ctx), http.MethodDelete, "/openApi/swap/v2/trade/order", params, swapCancelOrderRate)
}

// swapTradeOrderIDParams returns the parameters identifying one of a symbol's orders, by order ID or client order ID
func (e *Exchange) swapTradeOrderIDParams(symbol currency.Pair, orderID uint64, clientOrderID string) (url.Values, error) {
	params, err := e.swapTradeSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	if orderID == 0 && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if orderID != 0 {
		params.Set("orderId", strconv.FormatUint(orderID, 10))
	}
	if clientOrderID != "" {
		params.Set("clientOrderId", clientOrderID)
	}
	return params, nil
}

// swapTradeOrderReply sends a request BingX answers with one order, and returns the order
func (e *Exchange) swapTradeOrderReply(ctx context.Context, method, path string, params url.Values, f request.EndpointLimit) (*SwapOrderResponse, error) {
	var resp *SwapOrderDataResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, method, path, params, f, &resp); err != nil {
		return nil, err
	}
	if resp.Order == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Order, nil
}

// CancelSwapBatchOrders cancels up to ten of a symbol's orders by order ID and up to ten by client order ID with Cancel
// Multiple Orders, which reports the orders it cancelled and those it could not. Like CancelSwapOrder, a cancellation
// whose reply times out is not sent again
func (e *Exchange) CancelSwapBatchOrders(ctx context.Context, symbol currency.Pair, orderIDs []uint64, clientOrderIDs []string) (*SwapCancelOrdersResponse, error) {
	params, err := e.swapTradeSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	if len(orderIDs) == 0 && len(clientOrderIDs) == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	if err := swapTradeCheckListLength("order IDs", len(orderIDs), swapTradeBatchCancelLimit); err != nil {
		return nil, err
	}
	if err := swapTradeCheckListLength("client order IDs", len(clientOrderIDs), swapTradeBatchCancelLimit); err != nil {
		return nil, err
	}
	if len(orderIDs) != 0 {
		ids, err := json.Marshal(orderIDs)
		if err != nil {
			return nil, fmt.Errorf("error encoding order IDs: %w", err)
		}
		params.Set("orderIdList", string(ids))
	}
	if len(clientOrderIDs) != 0 {
		ids, err := json.Marshal(clientOrderIDs)
		if err != nil {
			return nil, fmt.Errorf("error encoding client order IDs: %w", err)
		}
		params.Set("clientOrderIdList", string(ids))
	}
	var resp *SwapCancelOrdersResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestUSDTMargined, http.MethodDelete, "/openApi/swap/v2/trade/batchOrders", params, swapCancelBatchOrdersRate, &resp)
}

// CancelAllSwapOpenOrders cancels the open orders of a symbol, or of every symbol when symbol is empty, with Cancel All
// Open Orders. orderType limits the cancellation to one order type; BingX lists it as required, and it is left out
// when empty
func (e *Exchange) CancelAllSwapOpenOrders(ctx context.Context, symbol currency.Pair, orderType OrderType) (*SwapCancelOrdersResponse, error) {
	params := url.Values{}
	if err := e.swapTradeSetSymbol(params, symbol); err != nil {
		return nil, err
	}
	if orderType != "" {
		params.Set("type", string(orderType))
	}
	var resp *SwapCancelOrdersResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestUSDTMargined, http.MethodDelete, "/openApi/swap/v2/trade/allOpenOrders", params, swapCancelAllOpenOrdersRate, &resp)
}

// GetSwapOpenOrders returns the open orders of a symbol, or of every symbol when symbol is empty, of orderType or of
// every type when orderType is empty, with Current All Open Orders
func (e *Exchange) GetSwapOpenOrders(ctx context.Context, symbol currency.Pair, orderType OrderType) ([]SwapOrderResponse, error) {
	params := url.Values{}
	if err := e.swapTradeSetSymbol(params, symbol); err != nil {
		return nil, err
	}
	if orderType != "" {
		params.Set("type", string(orderType))
	}
	return e.swapTradeOrderList(ctx, "/openApi/swap/v2/trade/openOrders", params, swapOpenOrdersRate)
}

// swapTradeOrderList sends a query BingX answers with a list of orders, and returns the orders
func (e *Exchange) swapTradeOrderList(ctx context.Context, path string, params url.Values, f request.EndpointLimit) ([]SwapOrderResponse, error) {
	var resp *SwapOrdersDataResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, path, params, f, &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

// GetSwapPendingOrderStatus returns an open order with Query Pending Order Status, by its order ID or, when orderID is
// 0, its client order ID
func (e *Exchange) GetSwapPendingOrderStatus(ctx context.Context, symbol currency.Pair, orderID uint64, clientOrderID string) (*SwapOrderResponse, error) {
	params, err := e.swapTradeOrderIDParams(symbol, orderID, clientOrderID)
	if err != nil {
		return nil, err
	}
	return e.swapTradeOrderReply(ctx, http.MethodGet, "/openApi/swap/v2/trade/openOrder", params, swapPendingOrderStatusRate)
}

// GetSwapOrderDetails returns an order with Query Order Details, by its order ID or, when orderID is 0, its client
// order ID
func (e *Exchange) GetSwapOrderDetails(ctx context.Context, symbol currency.Pair, orderID uint64, clientOrderID string) (*SwapOrderResponse, error) {
	params, err := e.swapTradeOrderIDParams(symbol, orderID, clientOrderID)
	if err != nil {
		return nil, err
	}
	return e.swapTradeOrderReply(ctx, http.MethodGet, "/openApi/swap/v2/trade/order", params, swapOrderDetailsRate)
}

// GetSwapMarginType returns a contract's margin type: isolated, crossed or separate isolated
func (e *Exchange) GetSwapMarginType(ctx context.Context, symbol currency.Pair) (*SwapMarginTypeResponse, error) {
	params, err := e.swapTradeSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	var resp *SwapMarginTypeResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/openApi/swap/v2/trade/marginType", params, swapMarginTypeRate, &resp); err != nil {
		return nil, err
	}
	if resp.MarginType == "" {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// ChangeSwapMarginType sets a contract's margin type with Change Margin Type
func (e *Exchange) ChangeSwapMarginType(ctx context.Context, symbol currency.Pair, marginType MarginType) error {
	params, err := e.swapTradeSymbolParams(symbol)
	if err != nil {
		return err
	}
	if marginType == "" {
		return errSwapMarginTypeRequired
	}
	params.Set("marginType", string(marginType))
	return e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/openApi/swap/v2/trade/marginType", params, swapChangeMarginTypeRate, nil)
}

// GetSwapLeverage returns a contract's long and short leverage and the positions they allow, with Query Leverage and
// Available Positions
func (e *Exchange) GetSwapLeverage(ctx context.Context, symbol currency.Pair) (*SwapLeverageResponse, error) {
	params, err := e.swapTradeSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	var resp *SwapLeverageResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/openApi/swap/v2/trade/leverage", params, swapLeverageRate, &resp); err != nil {
		return nil, err
	}
	if resp.LongLeverage == 0 && resp.ShortLeverage == 0 {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// SetSwapLeverage sets a contract's leverage with Set Leverage. side is the position side the leverage applies to:
// LONG or SHORT in hedge mode, BOTH in one-way mode
func (e *Exchange) SetSwapLeverage(ctx context.Context, symbol currency.Pair, side PositionSide, leverage uint64) (*SetSwapLeverageResponse, error) {
	params, err := e.swapTradeSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	if side == "" {
		return nil, errPositionSideRequired
	}
	if leverage == 0 {
		return nil, errSwapLeverageRequired
	}
	params.Set("side", string(side))
	params.Set("leverage", strconv.FormatUint(leverage, 10))
	var resp *SetSwapLeverageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/openApi/swap/v2/trade/leverage", params, swapSetLeverageRate, &resp)
}

// GetSwapForceOrders returns the account's liquidation and auto-deleveraging orders with User's Force Orders
func (e *Exchange) GetSwapForceOrders(ctx context.Context, arg *SwapForceOrdersRequest) ([]SwapOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setTimeRange(params, "startTime", "endTime", arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if err := e.swapTradeSetSymbol(params, arg.Symbol); err != nil {
		return nil, err
	}
	if !arg.Currency.IsEmpty() {
		params.Set("currency", arg.Currency.Upper().String())
	}
	if arg.AutoCloseType != "" {
		params.Set("autoCloseType", string(arg.AutoCloseType))
	}
	if arg.Limit != 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	return e.swapTradeOrderList(ctx, "/openApi/swap/v2/trade/forceOrders", params, swapForceOrdersRate)
}

// GetSwapOrderHistory returns filled and cancelled orders with Query Order History. A query spans at most seven days,
// and BingX refuses a longer one with code 109400. Without Currency it covers USDT-settled contracts; BingX asks for a
// USDC symbol, such as BTC-USDC, and Currency USDC to query USDC-settled ones
func (e *Exchange) GetSwapOrderHistory(ctx context.Context, arg *SwapOrderHistoryRequest) ([]SwapOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setTimeRange(params, "startTime", "endTime", arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if err := e.swapTradeSetSymbol(params, arg.Symbol); err != nil {
		return nil, err
	}
	if !arg.Currency.IsEmpty() {
		params.Set("currency", arg.Currency.Upper().String())
	}
	if arg.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(arg.OrderID, 10))
	}
	if arg.Limit != 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	return e.swapTradeOrderList(ctx, "/openApi/swap/v2/trade/allOrders", params, swapOrderHistoryRate)
}

// ModifySwapIsolatedPositionMargin adds margin to or removes margin from an isolated position with Modify Isolated
// Position Margin
func (e *Exchange) ModifySwapIsolatedPositionMargin(ctx context.Context, arg *ModifySwapIsolatedPositionMarginRequest) (*ModifySwapIsolatedPositionMarginResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.swapTradeSymbolParams(arg.Symbol)
	if err != nil {
		return nil, err
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.Type == 0 {
		return nil, errSwapMarginAdjustmentTypeRequired
	}
	params.Set("amount", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	params.Set("type", strconv.FormatUint(uint64(arg.Type), 10))
	if arg.PositionSide != "" {
		params.Set("positionSide", string(arg.PositionSide))
	}
	if arg.PositionID != 0 {
		params.Set("positionId", strconv.FormatUint(arg.PositionID, 10))
	}
	var resp *ModifySwapIsolatedPositionMarginResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestUSDTMargined, http.MethodPost, "/openApi/swap/v2/trade/positionMargin", params, swapModifyIsolatedPositionMarginRate, &resp)
}

// GetSwapHistoricalTransactionOrders returns the account's filled orders with Query Historical Transaction Orders.
// Without Currency it covers USDT-settled contracts; BingX asks for a USDC symbol, such as BTC-USDC, and Currency USDC
// to query USDC-settled ones
func (e *Exchange) GetSwapHistoricalTransactionOrders(ctx context.Context, arg *SwapHistoricalTransactionOrdersRequest) ([]SwapHistoricalTransactionOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.TradingUnit == "" {
		return nil, errSwapTradingUnitRequired
	}
	if arg.StartTime.IsZero() || arg.EndTime.IsZero() {
		return nil, common.ErrDateUnset
	}
	params := url.Values{}
	if err := setTimeRange(params, "startTs", "endTs", arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if err := e.swapTradeSetSymbol(params, arg.Symbol); err != nil {
		return nil, err
	}
	if !arg.Currency.IsEmpty() {
		params.Set("currency", arg.Currency.Upper().String())
	}
	if arg.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(arg.OrderID, 10))
	}
	params.Set("tradingUnit", string(arg.TradingUnit))
	var resp *SwapHistoricalTransactionOrdersDataResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/openApi/swap/v2/trade/allFillOrders", params, swapFillOrdersRate, &resp); err != nil {
		return nil, err
	}
	return resp.FillOrders, nil
}

// SetSwapPositionMode sets the account's position mode with Set Position Mode: hedge mode when dualSidePosition is
// true, one-way mode otherwise. The mode applies to every contract and changes only without open positions or orders.
// The account's cached mode, which orders and leverage changes follow, is forgotten along with any lookup of it in
// flight, so the next of them looks the mode up again
func (e *Exchange) SetSwapPositionMode(ctx context.Context, dualSidePosition bool) (*SwapPositionModeResponse, error) {
	params := url.Values{}
	params.Set("dualSidePosition", strconv.FormatBool(dualSidePosition))
	var resp *SwapPositionModeResponse
	err := e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/openApi/swap/v1/positionSide/dual", params, swapSetPositionModeRate, &resp)
	// Forgotten whatever the outcome, as a request whose reply was lost may still have changed the mode
	e.forgetSwapPositionMode(ctx)
	return resp, err
}

// GetSwapPositionMode returns the account's position mode with Query Position Mode
func (e *Exchange) GetSwapPositionMode(ctx context.Context) (*SwapPositionModeResponse, error) {
	var resp *SwapPositionModeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/openApi/swap/v1/positionSide/dual", nil, swapPositionModeRate, &resp)
}

// CancelReplaceSwapOrder cancels an order and places another on the same symbol with Cancel an Existing Order and Send
// a New Order. Both halves report their own result: with CancelReplaceModeAllowFailure the new order is placed even
// when the cancellation fails
func (e *Exchange) CancelReplaceSwapOrder(ctx context.Context, arg *CancelReplaceSwapOrderRequest) (*SwapCancelReplaceOrderResponse, error) {
	item, err := e.swapTradeCancelReplaceOrder(arg)
	if err != nil {
		return nil, err
	}
	params := swapTradeOrderValues(&item.SwapBatchOrderRequest)
	params.Set("cancelReplaceMode", string(item.CancelReplaceMode))
	if item.CancelOrderID != 0 {
		params.Set("cancelOrderId", strconv.FormatUint(item.CancelOrderID, 10))
	}
	if item.CancelClientOrderID != "" {
		params.Set("cancelClientOrderId", item.CancelClientOrderID)
	}
	if item.CancelRestrictions != "" {
		params.Set("cancelRestrictions", string(item.CancelRestrictions))
	}
	var resp *SwapCancelReplaceOrderResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestUSDTMargined, http.MethodPost, "/openApi/swap/v1/trade/cancelReplace", params, swapCancelReplaceOrderRate, &resp)
}

// swapTradeCancelReplaceOrder validates a cancel-replace and returns it in the form BingX takes, which single and batch
// requests share
func (e *Exchange) swapTradeCancelReplaceOrder(arg *CancelReplaceSwapOrderRequest) (*SwapBatchCancelReplaceOrderRequest, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.CancelReplaceMode == "" {
		return nil, errSwapCancelReplaceModeRequired
	}
	if arg.CancelOrderID == 0 && arg.CancelClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.PositionSide == "" {
		return nil, errPositionSideRequired
	}
	item, err := e.swapTradeOrder(&arg.SwapOrderRequest)
	if err != nil {
		return nil, err
	}
	return &SwapBatchCancelReplaceOrderRequest{
		CancelReplaceMode:     arg.CancelReplaceMode,
		CancelOrderID:         arg.CancelOrderID,
		CancelClientOrderID:   arg.CancelClientOrderID,
		CancelRestrictions:    arg.CancelRestrictions,
		SwapBatchOrderRequest: *item,
	}, nil
}

// CancelReplaceSwapBatchOrders cancels orders and places new ones in one request with Cancel Orders in Batches and
// Place Orders in Batches, replying with both halves of each cancel-replace
func (e *Exchange) CancelReplaceSwapBatchOrders(ctx context.Context, args []CancelReplaceSwapOrderRequest) ([]SwapCancelReplaceOrderResponse, error) {
	if len(args) == 0 {
		return nil, common.ErrEmptyParams
	}
	batch := make([]*SwapBatchCancelReplaceOrderRequest, len(args))
	for i := range args {
		item, err := e.swapTradeCancelReplaceOrder(&args[i])
		if err != nil {
			return nil, fmt.Errorf("order %d: %w", i, err)
		}
		batch[i] = item
	}
	batchOrders, err := json.Marshal(batch)
	if err != nil {
		return nil, fmt.Errorf("error encoding batch orders: %w", err)
	}
	params := url.Values{}
	params.Set("batchOrders", string(batchOrders))
	var resp []SwapCancelReplaceOrderResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestUSDTMargined, http.MethodPost, "/openApi/swap/v1/trade/batchCancelReplace", params, swapBatchCancelReplaceOrdersRate, &resp)
}

// CancelAllSwapOrdersAfter starts, extends or stops the countdown of Cancel All After, which cancels every open order
// once it ends. timeout is sent in whole seconds, from 10 to 120. While a cancellation runs BingX refuses further
// requests
func (e *Exchange) CancelAllSwapOrdersAfter(ctx context.Context, requestType CancelAllAfterType, timeout time.Duration) (*SwapCancelAllAfterResponse, error) {
	if requestType == "" {
		return nil, errSwapCancelAllAfterTypeRequired
	}
	params := url.Values{}
	params.Set("type", string(requestType))
	params.Set("timeOut", strconv.FormatInt(int64(timeout/time.Second), 10))
	var resp *SwapCancelAllAfterResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestUSDTMargined, http.MethodPost, "/openApi/swap/v2/trade/cancelAllAfter", params, swapCancelAllAfterRate, &resp)
}

// CloseSwapPosition closes a position at market price with Close Position by Position ID
func (e *Exchange) CloseSwapPosition(ctx context.Context, positionID uint64) (*CloseSwapPositionResponse, error) {
	if positionID == 0 {
		return nil, errSwapPositionIDRequired
	}
	params := url.Values{}
	params.Set("positionId", strconv.FormatUint(positionID, 10))
	var resp *CloseSwapPositionResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestUSDTMargined, http.MethodPost, "/openApi/swap/v1/trade/closePosition", params, swapClosePositionRate, &resp)
}

// GetSwapAllOrders returns the account's orders in any status with All Orders
func (e *Exchange) GetSwapAllOrders(ctx context.Context, arg *SwapAllOrdersRequest) ([]SwapOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setTimeRange(params, "startTime", "endTime", arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if err := e.swapTradeSetSymbol(params, arg.Symbol); err != nil {
		return nil, err
	}
	if arg.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(arg.OrderID, 10))
	}
	if arg.Limit != 0 {
		params.Set("limit", strconv.FormatUint(arg.Limit, 10))
	}
	return e.swapTradeOrderList(ctx, "/openApi/swap/v1/trade/fullOrder", params, swapAllOrdersRate)
}

// GetSwapMaintenanceMarginRatio returns a contract's position value tiers and their maintenance margin ratios with
// Position and Maintenance Margin Ratio
func (e *Exchange) GetSwapMaintenanceMarginRatio(ctx context.Context, symbol currency.Pair) ([]SwapMaintenanceMarginRatioResponse, error) {
	params, err := e.swapTradeSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	var resp []SwapMaintenanceMarginRatioResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/openApi/swap/v1/maintMarginRatio", params, swapMaintenanceMarginRatioRate, &resp)
}

// GetSwapHistoricalTransactionDetails returns a page of the account's fills with Query Historical Transaction Details,
// in ascending fill time. It covers the last seven days and at most 1000 fills. Without Currency it covers USDT-settled
// contracts; BingX asks for a USDC symbol, such as BTC-USDC, and Currency USDC to query USDC-settled ones
func (e *Exchange) GetSwapHistoricalTransactionDetails(ctx context.Context, arg *SwapHistoricalTransactionDetailsRequest) (*SwapHistoricalTransactionDetailsResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params := url.Values{}
	if err := setTimeRange(params, "startTs", "endTs", arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if err := e.swapTradeSetSymbol(params, arg.Symbol); err != nil {
		return nil, err
	}
	if !arg.Currency.IsEmpty() {
		params.Set("currency", arg.Currency.Upper().String())
	}
	if arg.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(arg.OrderID, 10))
	}
	if arg.PageIndex != 0 {
		params.Set("pageIndex", strconv.FormatUint(arg.PageIndex, 10))
	}
	if arg.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(arg.PageSize, 10))
	}
	var resp *SwapHistoricalTransactionDetailsResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/openApi/swap/v2/trade/fillHistory", params, swapFillHistoryRate, &resp)
}

// GetSwapPositionHistory returns a contract's closed positions with Query Position History
func (e *Exchange) GetSwapPositionHistory(ctx context.Context, arg *SwapPositionHistoryRequest) ([]SwapPositionHistoryResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.swapTradeSymbolParams(arg.Symbol)
	if err != nil {
		return nil, err
	}
	if arg.StartTime.IsZero() || arg.EndTime.IsZero() {
		return nil, common.ErrDateUnset
	}
	if err := setTimeRange(params, "startTime", "endTime", arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if !arg.Currency.IsEmpty() {
		params.Set("currency", arg.Currency.Upper().String())
	}
	if arg.PositionID != 0 {
		params.Set("positionId", strconv.FormatUint(arg.PositionID, 10))
	}
	if arg.PageIndex != 0 {
		params.Set("pageIndex", strconv.FormatUint(arg.PageIndex, 10))
	}
	if arg.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(arg.PageSize, 10))
	}
	var resp []SwapPositionHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/openApi/swap/v2/trade/positionHistory", params, swapPositionHistoryRate, &resp)
}

// GetSwapIsolatedMarginChangeHistory returns a page of an isolated position's margin changes with Isolated Margin
// Change History
func (e *Exchange) GetSwapIsolatedMarginChangeHistory(ctx context.Context, arg *SwapIsolatedMarginChangeHistoryRequest) (*SwapIsolatedMarginChangeHistoryResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.swapTradeSymbolParams(arg.Symbol)
	if err != nil {
		return nil, err
	}
	if arg.PositionID == 0 {
		return nil, errSwapPositionIDRequired
	}
	if arg.StartTime.IsZero() || arg.EndTime.IsZero() {
		return nil, common.ErrDateUnset
	}
	if err := setTimeRange(params, "startTime", "endTime", arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	params.Set("positionId", strconv.FormatUint(arg.PositionID, 10))
	if arg.PageIndex != 0 {
		params.Set("pageIndex", strconv.FormatUint(arg.PageIndex, 10))
	}
	if arg.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(arg.PageSize, 10))
	}
	var resp *SwapIsolatedMarginChangeHistoryResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/openApi/swap/v1/positionMargin/history", params, swapIsolatedMarginHistoryRate, &resp)
}

// ApplySwapVST adds VST to or removes VST from a demo account with Apply VST, which only the demo trading host
// https://open-api-vst.bingx.com serves, so the USDT-M REST endpoint must point at it. amount is a whole number of VST,
// at most 1,000,000 a request and 10,000,000 in total
func (e *Exchange) ApplySwapVST(ctx context.Context, adjustType VSTAdjustType, amount uint64) (*ApplySwapVSTResponse, error) {
	params := url.Values{}
	if adjustType != "" {
		params.Set("adjustType", string(adjustType))
	}
	if amount != 0 {
		params.Set("amount", strconv.FormatUint(amount, 10))
	}
	var resp *ApplySwapVSTResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestUSDTMargined, http.MethodPost, "/openApi/swap/v2/trade/getVst", params, swapApplyVSTRate, &resp)
}

// PlaceSwapTWAPOrder places a TWAP order with Place TWAP Order, which splits it into child orders placed over up to 24
// hours to lessen its market impact
func (e *Exchange) PlaceSwapTWAPOrder(ctx context.Context, arg *SwapTWAPOrderRequest) (*PlaceSwapTWAPOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.swapTradeSymbolParams(arg.Symbol)
	if err != nil {
		return nil, err
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.PositionSide == "" {
		return nil, errPositionSideRequired
	}
	if arg.PriceType == "" {
		return nil, errSwapTWAPPriceTypeRequired
	}
	if arg.PriceVariance <= 0 {
		return nil, errSwapTWAPPriceVarianceRequired
	}
	if arg.TriggerPrice <= 0 {
		return nil, limits.ErrPriceBelowMin
	}
	if arg.Interval < time.Second {
		return nil, errSwapTWAPIntervalRequired
	}
	if arg.AmountPerOrder <= 0 {
		return nil, fmt.Errorf("%w: amount per order", limits.ErrAmountBelowMin)
	}
	if arg.TotalAmount <= 0 {
		return nil, fmt.Errorf("%w: total amount", limits.ErrAmountBelowMin)
	}
	params.Set("side", string(arg.Side))
	params.Set("positionSide", string(arg.PositionSide))
	params.Set("priceType", string(arg.PriceType))
	params.Set("priceVariance", strconv.FormatFloat(arg.PriceVariance, 'f', -1, 64))
	params.Set("triggerPrice", strconv.FormatFloat(arg.TriggerPrice, 'f', -1, 64))
	params.Set("interval", strconv.FormatInt(int64(arg.Interval/time.Second), 10))
	params.Set("amountPerOrder", strconv.FormatFloat(arg.AmountPerOrder, 'f', -1, 64))
	params.Set("totalAmount", strconv.FormatFloat(arg.TotalAmount, 'f', -1, 64))
	var resp *PlaceSwapTWAPOrderResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestUSDTMargined, http.MethodPost, "/openApi/swap/v1/twap/order", params, swapPlaceTWAPOrderRate, &resp)
}

// GetSwapTWAPEntrustedOrders returns the TWAP orders of a symbol, or of every symbol when symbol is empty, that are
// waiting or running, with Query TWAP Entrusted Order
func (e *Exchange) GetSwapTWAPEntrustedOrders(ctx context.Context, symbol currency.Pair) (*SwapTWAPOrdersResponse, error) {
	params := url.Values{}
	if err := e.swapTradeSetSymbol(params, symbol); err != nil {
		return nil, err
	}
	var resp *SwapTWAPOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/openApi/swap/v1/twap/openOrders", params, swapTWAPOpenOrdersRate, &resp)
}

// GetSwapTWAPHistoricalOrders returns a page of TWAP orders that have finished, partly or fully filled, cancelled or
// failed, with Query TWAP Historical Orders
func (e *Exchange) GetSwapTWAPHistoricalOrders(ctx context.Context, arg *SwapTWAPHistoricalOrdersRequest) (*SwapTWAPOrdersResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.StartTime.IsZero() || arg.EndTime.IsZero() {
		return nil, common.ErrDateUnset
	}
	params := url.Values{}
	if err := setTimeRange(params, "startTime", "endTime", arg.StartTime, arg.EndTime); err != nil {
		return nil, err
	}
	if err := e.swapTradeSetSymbol(params, arg.Symbol); err != nil {
		return nil, err
	}
	if arg.PageIndex != 0 {
		params.Set("pageIndex", strconv.FormatUint(arg.PageIndex, 10))
	}
	if arg.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(arg.PageSize, 10))
	}
	var resp *SwapTWAPOrdersResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/openApi/swap/v1/twap/historyOrders", params, swapTWAPHistoryOrdersRate, &resp)
}

// GetSwapTWAPOrderDetails returns a TWAP order with TWAP Order Details
func (e *Exchange) GetSwapTWAPOrderDetails(ctx context.Context, mainOrderID string) (*SwapTWAPOrderResponse, error) {
	if mainOrderID == "" {
		return nil, errSwapMainOrderIDRequired
	}
	params := url.Values{}
	params.Set("mainOrderId", mainOrderID)
	var resp *SwapTWAPOrderResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/openApi/swap/v1/twap/orderDetail", params, swapTWAPOrderDetailRate, &resp); err != nil {
		return nil, err
	}
	if resp.Symbol == "" {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// CancelSwapTWAPOrder cancels a TWAP order with Cancel TWAP Order. Like CancelSwapOrder, a cancellation whose reply
// times out is not sent again
func (e *Exchange) CancelSwapTWAPOrder(ctx context.Context, mainOrderID string) (*SwapTWAPOrderResponse, error) {
	if mainOrderID == "" {
		return nil, errSwapMainOrderIDRequired
	}
	params := url.Values{}
	params.Set("mainOrderId", mainOrderID)
	var resp *SwapTWAPOrderResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestUSDTMargined, http.MethodPost, "/openApi/swap/v1/twap/cancelOrder", params, swapCancelTWAPOrderRate, &resp)
}

// SwitchSwapMultiAssetsMode switches the account between single asset and multi-assets mode with Switch Multi-Assets
// Mode
func (e *Exchange) SwitchSwapMultiAssetsMode(ctx context.Context, assetMode AssetMode) (*SwapMultiAssetsModeResponse, error) {
	if assetMode == "" {
		return nil, errSwapAssetModeRequired
	}
	params := url.Values{}
	params.Set("assetMode", string(assetMode))
	var resp *SwapMultiAssetsModeResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/openApi/swap/v1/trade/assetMode", params, swapSwitchMultiAssetsModeRate, &resp)
}

// GetSwapMultiAssetsMode returns the account's multi-assets mode with Query Multi-Assets Mode
func (e *Exchange) GetSwapMultiAssetsMode(ctx context.Context) (*SwapMultiAssetsModeResponse, error) {
	var resp *SwapMultiAssetsModeResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/openApi/swap/v1/trade/assetMode", nil, swapMultiAssetsModeRate, &resp); err != nil {
		return nil, err
	}
	if resp.AssetMode == "" {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetSwapMultiAssetsRules returns the assets that can margin USDT-M perpetuals in multi-assets mode and how they are
// valued, with Query Multi-Assets Rules
func (e *Exchange) GetSwapMultiAssetsRules(ctx context.Context) ([]SwapMultiAssetsRuleResponse, error) {
	var resp []SwapMultiAssetsRuleResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/openApi/swap/v1/trade/multiAssetsRules", nil, swapMultiAssetsRulesRate, &resp)
}

// GetSwapMultiAssetsMargin returns the assets margining the account in multi-assets mode with Query Multi-Assets
// Margin
func (e *Exchange) GetSwapMultiAssetsMargin(ctx context.Context) ([]SwapMultiAssetsMarginResponse, error) {
	var resp []SwapMultiAssetsMarginResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, "/openApi/swap/v1/user/marginAssets", nil, swapMultiAssetsMarginRate, &resp)
}

// ReverseSwapPosition closes a position and opens one of the same size on the other side with One-Click Reverse
// Position, at once or once a trigger price is reached
func (e *Exchange) ReverseSwapPosition(ctx context.Context, arg *ReverseSwapPositionRequest) (*ReverseSwapPositionResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Type == "" {
		return nil, errSwapReverseTypeRequired
	}
	params, err := e.swapTradeSymbolParams(arg.Symbol)
	if err != nil {
		return nil, err
	}
	params.Set("type", string(arg.Type))
	if arg.TriggerPrice != 0 {
		params.Set("triggerPrice", strconv.FormatFloat(arg.TriggerPrice, 'f', -1, 64))
	}
	if arg.WorkingType != "" {
		params.Set("workingType", string(arg.WorkingType))
	}
	if arg.PositionID != 0 {
		params.Set("positionId", strconv.FormatUint(arg.PositionID, 10))
	}
	var resp *ReverseSwapPositionResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestUSDTMargined, http.MethodPost, "/openApi/swap/v1/trade/reverse", params, swapReversePositionRate, &resp)
}

// SetSwapAutoAddMargin turns automatic margin addition on or off for a hedge mode position with Hedge Mode Position -
// Automatic Margin Addition
func (e *Exchange) SetSwapAutoAddMargin(ctx context.Context, arg *SwapAutoAddMarginRequest) (*SwapAutoAddMarginResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	params, err := e.swapTradeSymbolParams(arg.Symbol)
	if err != nil {
		return nil, err
	}
	if arg.PositionID == 0 {
		return nil, errSwapPositionIDRequired
	}
	params.Set("positionId", strconv.FormatUint(arg.PositionID, 10))
	params.Set("functionSwitch", strconv.FormatBool(arg.FunctionSwitch))
	if arg.Amount != 0 {
		params.Set("amount", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	}
	var resp *SwapAutoAddMarginResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, "/openApi/swap/v1/trade/autoAddMargin", params, swapAutoAddMarginRate, &resp)
}
