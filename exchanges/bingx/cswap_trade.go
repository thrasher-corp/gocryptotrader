package bingx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

var (
	errCoinMarginedLeverageEmpty             = errors.New("leverage empty")
	errCoinMarginedMarginAdjustmentTypeEmpty = errors.New("margin adjustment type empty")
	errCoinMarginedLimitEmpty                = errors.New("limit empty")
	errCoinMarginedOrderIDEmpty              = errors.New("order ID empty")
)

// PlaceCoinMarginedOrder places a coin-M perpetual order through Trade order, with the take profit and stop loss
// orders the request holds
func (e *Exchange) PlaceCoinMarginedOrder(ctx context.Context, req *PlaceCoinMarginedOrderRequest) (*PlaceCoinMarginedOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if req.Type == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if req.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if req.PositionSide == "" {
		return nil, errPositionSideRequired
	}
	params, err := e.coinMarginedSymbolParams(req.Symbol)
	if err != nil {
		return nil, err
	}
	params.Set("type", string(req.Type))
	params.Set("side", string(req.Side))
	params.Set("positionSide", string(req.PositionSide))
	if req.Price != 0 {
		params.Set("price", strconv.FormatFloat(req.Price, 'f', -1, 64))
	}
	if req.Quantity != 0 {
		params.Set("quantity", strconv.FormatFloat(req.Quantity, 'f', -1, 64))
	}
	if req.StopPrice != 0 {
		params.Set("stopPrice", strconv.FormatFloat(req.StopPrice, 'f', -1, 64))
	}
	if req.WorkingType != "" {
		params.Set("workingType", string(req.WorkingType))
	}
	if req.StopLoss != nil {
		stopLoss, err := json.Marshal(req.StopLoss)
		if err != nil {
			return nil, fmt.Errorf("error encoding stop loss: %w", err)
		}
		params.Set("stopLoss", string(stopLoss))
	}
	if req.TakeProfit != nil {
		takeProfit, err := json.Marshal(req.TakeProfit)
		if err != nil {
			return nil, fmt.Errorf("error encoding take profit: %w", err)
		}
		params.Set("takeProfit", string(takeProfit))
	}
	if req.ClientOrderID != "" {
		params.Set("clientOrderId", req.ClientOrderID)
	}
	if req.TimeInForce != "" {
		params.Set("timeInForce", string(req.TimeInForce))
	}
	var resp *CoinMarginedOrderData[PlaceCoinMarginedOrderResponse]
	if err := e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestCoinMargined, http.MethodPost, "/openApi/cswap/v2/trade/order", params, coinMarginedPlaceOrderRate, &resp); err != nil {
		return nil, err
	}
	if resp == nil || resp.Order == nil {
		return nil, common.ErrNoResponse
	}
	if resp.Order.OrderID == 0 {
		return nil, errCoinMarginedOrderIDEmpty
	}
	return resp.Order, nil
}

// GetCoinMarginedTradeCommissionRate returns the account's coin-M perpetual taker and maker commission rates
func (e *Exchange) GetCoinMarginedTradeCommissionRate(ctx context.Context) (*CoinMarginedTradeCommissionRateResponse, error) {
	var resp *CoinMarginedTradeCommissionRateResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/openApi/cswap/v1/user/commissionRate", nil, coinMarginedCommissionRateRate, &resp)
}

// GetCoinMarginedLeverage returns a coin-M perpetual symbol's leverage for long and short positions
func (e *Exchange) GetCoinMarginedLeverage(ctx context.Context, symbol currency.Pair) (*CoinMarginedLeverageResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params, err := e.coinMarginedSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	var resp *CoinMarginedLeverageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/openApi/cswap/v1/trade/leverage", params, coinMarginedLeverageRate, &resp)
}

// SetCoinMarginedLeverage sets a coin-M perpetual symbol's leverage for its LONG or SHORT positions through Modify
// Leverage, returning the symbol's leverage
func (e *Exchange) SetCoinMarginedLeverage(ctx context.Context, symbol currency.Pair, side PositionSide, leverage uint64) (*CoinMarginedLeverageResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if side == "" {
		return nil, errPositionSideRequired
	}
	if leverage == 0 {
		return nil, errCoinMarginedLeverageEmpty
	}
	params, err := e.coinMarginedSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	params.Set("side", string(side))
	params.Set("leverage", strconv.FormatUint(leverage, 10))
	var resp *CoinMarginedLeverageResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, "/openApi/cswap/v1/trade/leverage", params, coinMarginedSetLeverageRate, &resp)
}

// CancelAllCoinMarginedOrders cancels the pending coin-M perpetual orders of a symbol, or of every symbol when symbol
// is empty, reporting the orders cancelled and the cancellations that failed
func (e *Exchange) CancelAllCoinMarginedOrders(ctx context.Context, symbol currency.Pair) (*CancelAllCoinMarginedOrdersResponse, error) {
	params, err := e.coinMarginedSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	var resp *CancelAllCoinMarginedOrdersResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestCoinMargined, http.MethodDelete, "/openApi/cswap/v1/trade/allOpenOrders", params, coinMarginedCancelAllOpenOrdersRate, &resp)
}

// CloseAllCoinMarginedPositions closes the coin-M perpetual positions of a symbol, or of every symbol when symbol is
// empty, through Close all positions in bulk
func (e *Exchange) CloseAllCoinMarginedPositions(ctx context.Context, symbol currency.Pair) (*CloseAllCoinMarginedPositionsResponse, error) {
	params, err := e.coinMarginedSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	var resp *CloseAllCoinMarginedPositionsResponse
	return resp, e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestCoinMargined, http.MethodPost, "/openApi/cswap/v1/trade/closeAllPositions", params, coinMarginedCloseAllPositionsRate, &resp)
}

// GetCoinMarginedPositions returns the coin-M perpetual positions of a symbol, or of every symbol when symbol is empty,
// through Query warehouse
func (e *Exchange) GetCoinMarginedPositions(ctx context.Context, symbol currency.Pair) ([]CoinMarginedPositionResponse, error) {
	params, err := e.coinMarginedSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	var resp []CoinMarginedPositionResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/openApi/cswap/v1/user/positions", params, coinMarginedPositionsRate, &resp)
}

// GetCoinMarginedAccountAssets returns the coin-M perpetual account's assets through Query Account Assets, filtered by
// symbol when it is set
func (e *Exchange) GetCoinMarginedAccountAssets(ctx context.Context, symbol currency.Pair) ([]CoinMarginedAccountAssetResponse, error) {
	params, err := e.coinMarginedSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	var resp []CoinMarginedAccountAssetResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/openApi/cswap/v1/user/balance", params, coinMarginedBalanceRate, &resp)
}

// GetCoinMarginedForceOrders returns the account's coin-M perpetual liquidation and auto-deleveraging orders through
// Query force orders. BingX documents the reply as a list of orders without naming their fields, so each is read with
// the fields the other coin-M order queries report
func (e *Exchange) GetCoinMarginedForceOrders(ctx context.Context, req *CoinMarginedForceOrdersRequest) ([]CoinMarginedOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	params, err := e.coinMarginedSymbolParams(req.Symbol)
	if err != nil {
		return nil, err
	}
	if req.AutoCloseType != "" {
		params.Set("autoCloseType", string(req.AutoCloseType))
	}
	if err := setTimeRange(params, "startTime", "endTime", req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	if req.Limit != 0 {
		params.Set("limit", strconv.FormatUint(req.Limit, 10))
	}
	var resp []CoinMarginedOrderResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/openApi/cswap/v1/trade/forceOrders", params, coinMarginedForceOrdersRate, &resp)
}

// GetCoinMarginedOrderTradeDetail returns the trades that filled a coin-M perpetual order through Query Order Trade
// Detail, a page at a time
func (e *Exchange) GetCoinMarginedOrderTradeDetail(ctx context.Context, req *CoinMarginedOrderTradeDetailRequest) ([]CoinMarginedOrderTradeDetailResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderID == 0 {
		return nil, order.ErrOrderIDNotSet
	}
	params := url.Values{}
	params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	if req.PageIndex != 0 {
		params.Set("pageIndex", strconv.FormatUint(req.PageIndex, 10))
	}
	if req.PageSize != 0 {
		params.Set("pageSize", strconv.FormatUint(req.PageSize, 10))
	}
	var resp []CoinMarginedOrderTradeDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/openApi/cswap/v1/trade/allFillOrders", params, coinMarginedFillOrdersRate, &resp)
}

// CancelCoinMarginedOrder cancels a pending coin-M perpetual order by its order ID, its client order ID or both
func (e *Exchange) CancelCoinMarginedOrder(ctx context.Context, symbol currency.Pair, orderID uint64, clientOrderID string) (*CoinMarginedOrderResponse, error) {
	params, err := e.coinMarginedOrderIdentityParams(symbol, orderID, clientOrderID)
	if err != nil {
		return nil, err
	}
	return e.coinMarginedOrder(ctx, http.MethodDelete, "/openApi/cswap/v1/trade/cancelOrder", params, coinMarginedCancelOrderRate)
}

// coinMarginedOrderIdentityParams returns the parameters identifying a coin-M perpetual order: its symbol and its order
// ID, client order ID or both
func (e *Exchange) coinMarginedOrderIdentityParams(symbol currency.Pair, orderID uint64, clientOrderID string) (url.Values, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if orderID == 0 && clientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	params, err := e.coinMarginedSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	if orderID != 0 {
		params.Set("orderId", strconv.FormatUint(orderID, 10))
	}
	if clientOrderID != "" {
		params.Set("clientOrderId", clientOrderID)
	}
	return params, nil
}

// coinMarginedOrder sends a signed coin-M request that replies with one order, and returns the order
func (e *Exchange) coinMarginedOrder(ctx context.Context, method, path string, params url.Values, f request.EndpointLimit) (*CoinMarginedOrderResponse, error) {
	var resp *CoinMarginedOrderData[CoinMarginedOrderResponse]
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, method, path, params, f, &resp); err != nil {
		return nil, err
	}
	if resp == nil || resp.Order == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Order, nil
}

// GetCoinMarginedOpenOrders returns the pending coin-M perpetual orders of a symbol, or of every symbol when symbol is
// empty, through Query all current pending orders
func (e *Exchange) GetCoinMarginedOpenOrders(ctx context.Context, symbol currency.Pair) ([]CoinMarginedOrderResponse, error) {
	params, err := e.coinMarginedSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	var resp *CoinMarginedOrderListResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/openApi/cswap/v1/trade/openOrders", params, coinMarginedOpenOrdersRate, &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

// GetCoinMarginedOrderDetail returns a coin-M perpetual order by its order ID, its client order ID or both, through
// Query Order
func (e *Exchange) GetCoinMarginedOrderDetail(ctx context.Context, symbol currency.Pair, orderID uint64, clientOrderID string) (*CoinMarginedOrderResponse, error) {
	params, err := e.coinMarginedOrderIdentityParams(symbol, orderID, clientOrderID)
	if err != nil {
		return nil, err
	}
	return e.coinMarginedOrder(ctx, http.MethodGet, "/openApi/cswap/v1/trade/orderDetail", params, coinMarginedOrderDetailRate)
}

// GetCoinMarginedOrderHistory returns the filled and cancelled coin-M perpetual orders of a symbol, or of every symbol
// when the request's symbol is empty, through User's History Orders
func (e *Exchange) GetCoinMarginedOrderHistory(ctx context.Context, req *CoinMarginedOrderHistoryRequest) ([]CoinMarginedOrderResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Limit == 0 {
		return nil, errCoinMarginedLimitEmpty
	}
	params, err := e.coinMarginedSymbolParams(req.Symbol)
	if err != nil {
		return nil, err
	}
	if req.OrderID != 0 {
		params.Set("orderId", strconv.FormatUint(req.OrderID, 10))
	}
	if err := setTimeRange(params, "startTime", "endTime", req.StartTime, req.EndTime); err != nil {
		return nil, err
	}
	params.Set("limit", strconv.FormatUint(req.Limit, 10))
	var resp *CoinMarginedOrderListResponse
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/openApi/cswap/v1/trade/orderHistory", params, coinMarginedOrderHistoryRate, &resp); err != nil {
		return nil, err
	}
	return resp.Orders, nil
}

// GetCoinMarginedMarginType returns a coin-M perpetual symbol's margin type in a list, as documented; a reply holding a
// single object is read as a list of one
func (e *Exchange) GetCoinMarginedMarginType(ctx context.Context, symbol currency.Pair) ([]CoinMarginedMarginTypeResponse, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params, err := e.coinMarginedSymbolParams(symbol)
	if err != nil {
		return nil, err
	}
	var resp objectOrArray[CoinMarginedMarginTypeResponse]
	if err := e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodGet, "/openApi/cswap/v1/trade/marginType", params, coinMarginedMarginTypeRate, &resp); err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// SetCoinMarginedMarginType sets a coin-M perpetual symbol's margin type, ISOLATED or CROSSED
func (e *Exchange) SetCoinMarginedMarginType(ctx context.Context, symbol currency.Pair, marginType MarginType) error {
	if symbol.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if marginType == "" {
		return margin.ErrInvalidMarginType
	}
	params, err := e.coinMarginedSymbolParams(symbol)
	if err != nil {
		return err
	}
	params.Set("marginType", string(marginType))
	return e.SendAuthHTTPRequest(ctx, exchange.RestCoinMargined, http.MethodPost, "/openApi/cswap/v1/trade/marginType", params, coinMarginedSetMarginTypeRate, nil)
}

// AdjustCoinMarginedIsolatedMargin adds margin to or removes margin from an isolated coin-M perpetual position
func (e *Exchange) AdjustCoinMarginedIsolatedMargin(ctx context.Context, req *AdjustCoinMarginedIsolatedMarginRequest) error {
	if err := common.NilGuard(req); err != nil {
		return err
	}
	if req.Symbol.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if req.Amount <= 0 {
		return order.ErrAmountIsInvalid
	}
	if req.Type == "" {
		return errCoinMarginedMarginAdjustmentTypeEmpty
	}
	if req.PositionSide == "" {
		return errPositionSideRequired
	}
	params, err := e.coinMarginedSymbolParams(req.Symbol)
	if err != nil {
		return err
	}
	params.Set("amount", strconv.FormatFloat(req.Amount, 'f', -1, 64))
	params.Set("type", string(req.Type))
	params.Set("positionSide", string(req.PositionSide))
	return e.SendAuthHTTPRequest(request.WithRetryNotAllowed(ctx), exchange.RestCoinMargined, http.MethodPost, "/openApi/cswap/v1/trade/positionMargin", params, coinMarginedAdjustIsolatedMarginRate, nil)
}
