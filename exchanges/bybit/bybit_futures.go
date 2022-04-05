package bybit

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const (

	// auth endpoint
	futuresCreateOrder             = "/futures/private/order/create"
	futuresGetActiveOrders         = "/futures/private/order/list"
	futuresCancelActiveOrder       = "/futures/private/order/cancel"
	futuresCancelAllActiveOrders   = "/futures/private/order/cancelAll"
	futuresReplaceActiveOrder      = "/futures/private/order/replace"
	futuresGetActiveRealtimeOrders = "/futures/private/order"

	futuresCreateConditionalOrder       = "/futures/private/stop-order/create"
	futuresGetConditionalOrders         = "/futures/private/stop-order/list"
	futuresCancelConditionalOrder       = "/futures/private/stop-order/cancel"
	futuresCancelAllConditionalOrders   = "/futures/private/stop-order/cancelAll"
	futuresReplaceConditionalOrder      = "/futures/private/stop-order/replace"
	futuresGetConditionalRealtimeOrders = "/futures/private/stop-order"

	futuresPosition           = "/futures/private/position/list"
	futuresUpdateMargin       = "/futures/private/position/change-position-margin"
	futuresSetTradingStop     = "/futures/private/position/trading-stop"
	futuresSetLeverage        = "/futures/private/position/leverage/save"
	futuresSwitchPositionMode = "/futures/private/position/switch-mode"
	futuresSwitchPosition     = "/futures/private/tpsl/switch-mode"
	futuresSwitchMargin       = "/futures/private/position/switch-isolated"
	futuresGetTrades          = "/futures/private/execution/list"
	futuresGetClosedTrades    = "/futures/private/trade/closed-pnl/list"
	futuresSetRiskLimit       = "/futures/private/position/risk-limit"
)

// CreateFuturesOrder sends a new futures order to the exchange
func (by *Bybit) CreateFuturesOrder(ctx context.Context, positionMode int64, symbol currency.Pair, side, orderType, timeInForce,
	orderLinkID, takeProfitTriggerBy, stopLossTriggerBy string,
	quantity, price, takeProfit, stopLoss float64, closeOnTrigger, reduceOnly bool) (FuturesOrderDataResp, error) {
	resp := struct {
		Result FuturesOrderDataResp `json:"result"`
		Error
	}{}

	params := url.Values{}
	if positionMode < 0 || positionMode > 2 {
		return resp.Result, errInvalidPositionMode
	}
	params.Set("position_idx", strconv.FormatInt(positionMode, 10))

	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Result, err
	}
	params.Set("symbol", symbolValue)
	params.Set("side", side)
	params.Set("order_type", orderType)
	if quantity <= 0 {
		return resp.Result, errInvalidQuantity
	}
	params.Set("qty", strconv.FormatFloat(quantity, 'f', -1, 64))

	if price != 0 {
		params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	}
	if timeInForce == "" {
		return resp.Result, errInvalidTimeInForce
	}
	params.Set("time_in_force", timeInForce)

	if closeOnTrigger {
		params.Set("close_on_trigger", "true")
	}
	if orderLinkID != "" {
		params.Set("order_link_id", orderLinkID)
	}
	if takeProfit != 0 {
		params.Set("take_profit", strconv.FormatFloat(takeProfit, 'f', -1, 64))
	}
	if stopLoss != 0 {
		params.Set("stop_loss", strconv.FormatFloat(stopLoss, 'f', -1, 64))
	}
	if takeProfitTriggerBy != "" {
		params.Set("tp_trigger_by", takeProfitTriggerBy)
	}
	if stopLossTriggerBy != "" {
		params.Set("sl_trigger_by", stopLossTriggerBy)
	}
	if reduceOnly {
		params.Set("reduce_only", "true")
	}
	return resp.Result, by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresCreateOrder, params, &resp, FuturesCreateOrderRate)
}

// GetActiveFuturesOrders gets list of futures active orders
func (by *Bybit) GetActiveFuturesOrders(ctx context.Context, symbol currency.Pair, orderStatus, direction, cursor string, limit int64) ([]FuturesActiveOrder, error) {
	resp := struct {
		Result struct {
			Data   []FuturesActiveOrder `json:"data"`
			Cursor string               `json:"cursor"`
		} `json:"result"`
		Error
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Result.Data, err
	}
	params.Set("symbol", symbolValue)
	if orderStatus != "" {
		params.Set("order_status", orderStatus)
	}
	if direction != "" {
		params.Set("direction", direction)
	}
	if limit > 0 && limit <= 50 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	return resp.Result.Data, by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, futuresGetActiveOrders, params, &resp, FuturesGetActiveOrderRate)
}

// CancelActiveFuturesOrders cancels futures unfilled or partially filled orders
func (by *Bybit) CancelActiveFuturesOrders(ctx context.Context, symbol currency.Pair, orderID, orderLinkID string) (FuturesOrderCancelResp, error) {
	resp := struct {
		Result FuturesOrderCancelResp `json:"result"`
		Error
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Result, err
	}
	params.Set("symbol", symbolValue)
	if orderID == "" && orderLinkID == "" {
		return resp.Result, errOrderOrOrderLinkIDMissing
	}
	if orderID != "" {
		params.Set("order_id", orderID)
	}
	if orderLinkID != "" {
		params.Set("order_link_id", orderLinkID)
	}
	return resp.Result, by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresCancelActiveOrder, params, &resp, FuturesCancelOrderRate)
}

// CancelAllActiveFuturesOrders cancels all futures unfilled or partially filled orders
func (by *Bybit) CancelAllActiveFuturesOrders(ctx context.Context, symbol currency.Pair) ([]FuturesCancelOrderData, error) {
	resp := struct {
		Result []FuturesCancelOrderData `json:"result"`
		Error
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Result, err
	}
	params.Set("symbol", symbolValue)
	return resp.Result, by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresCancelAllActiveOrders, params, &resp, FuturesCancelAllOrderRate)
}

// ReplaceActiveFuturesOrders modify unfilled or partially filled orders
func (by *Bybit) ReplaceActiveFuturesOrders(ctx context.Context, symbol currency.Pair, orderID, orderLinkID, takeProfitTriggerBy, stopLossTriggerBy string,
	updatedQty, updatedPrice, takeProfitPrice, stopLossPrice float64) (string, error) {
	resp := struct {
		Result struct {
			OrderID string `json:"order_id"`
		} `json:"result"`
		Error
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return "", err
	}
	params.Set("symbol", symbolValue)
	if orderID == "" && orderLinkID == "" {
		return "", errOrderOrOrderLinkIDMissing
	}
	if orderID != "" {
		params.Set("order_id", orderID)
	}
	if orderLinkID != "" {
		params.Set("order_link_id", orderLinkID)
	}
	if updatedQty != 0 {
		params.Set("p_r_qty", strconv.FormatFloat(updatedQty, 'f', -1, 64))
	}
	if updatedPrice != 0 {
		params.Set("p_r_price", strconv.FormatFloat(updatedPrice, 'f', -1, 64))
	}
	if takeProfitPrice != 0 {
		params.Set("take_profit", strconv.FormatFloat(takeProfitPrice, 'f', -1, 64))
	}
	if stopLossPrice != 0 {
		params.Set("stop_loss", strconv.FormatFloat(stopLossPrice, 'f', -1, 64))
	}
	if takeProfitTriggerBy != "" {
		params.Set("tp_trigger_by", takeProfitTriggerBy)
	}
	if stopLossTriggerBy != "" {
		params.Set("sl_trigger_by", stopLossTriggerBy)
	}
	return resp.Result.OrderID, by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresReplaceActiveOrder, params, &resp, FuturesReplaceOrderRate)
}

// GetActiveRealtimeOrders query real time order data
func (by *Bybit) GetActiveRealtimeOrders(ctx context.Context, symbol currency.Pair, orderID, orderLinkID string) ([]FuturesActiveRealtimeOrder, error) {
	var data []FuturesActiveRealtimeOrder
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return data, err
	}
	params.Set("symbol", symbolValue)
	if orderID != "" {
		params.Set("order_id", orderID)
	}
	if orderLinkID != "" {
		params.Set("order_link_id", orderLinkID)
	}

	if orderID == "" && orderLinkID == "" {
		resp := struct {
			Result []FuturesActiveRealtimeOrder `json:"result"`
			Error
		}{}
		err = by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, futuresGetActiveRealtimeOrders, params, &resp, FuturesGetActiveRealtimeOrderRate)
		if err != nil {
			return data, err
		}
		data = append(data, resp.Result...)
	} else {
		resp := struct {
			Result FuturesActiveRealtimeOrder `json:"result"`
			Error
		}{}
		err = by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, futuresGetActiveRealtimeOrders, params, &resp, FuturesGetActiveRealtimeOrderRate)
		if err != nil {
			return data, err
		}
		data = append(data, resp.Result)
	}
	return data, nil
}

// CreateConditionalFuturesOrder sends a new conditional futures order to the exchange
func (by *Bybit) CreateConditionalFuturesOrder(ctx context.Context, positionMode int64, symbol currency.Pair, side, orderType, timeInForce,
	orderLinkID, takeProfitTriggerBy, stopLossTriggerBy, triggerBy string,
	quantity, price, takeProfit, stopLoss, basePrice, stopPrice float64, closeOnTrigger bool) (FuturesConditionalOrderResp, error) {
	resp := struct {
		Result FuturesConditionalOrderResp `json:"result"`
		Error
	}{}
	params := url.Values{}
	if positionMode < 0 || positionMode > 2 {
		return resp.Result, errInvalidPositionMode
	}
	params.Set("position_idx", strconv.FormatInt(positionMode, 10))

	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Result, err
	}
	params.Set("symbol", symbolValue)
	params.Set("side", side)
	params.Set("order_type", orderType)
	if quantity <= 0 {
		return resp.Result, errInvalidQuantity
	}
	params.Set("qty", strconv.FormatFloat(quantity, 'f', -1, 64))

	if price != 0 {
		params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	}
	if basePrice <= 0 {
		return resp.Result, errInvalidBasePrice
	}
	params.Set("base_price", strconv.FormatFloat(basePrice, 'f', -1, 64))

	if stopPrice <= 0 {
		return resp.Result, errInvalidStopPrice
	}
	params.Set("stop_px", strconv.FormatFloat(stopPrice, 'f', -1, 64))

	if timeInForce == "" {
		return resp.Result, errInvalidTimeInForce
	}
	params.Set("time_in_force", timeInForce)

	if triggerBy != "" {
		params.Set("trigger_by", triggerBy)
	}
	if closeOnTrigger {
		params.Set("close_on_trigger", "true")
	}
	if orderLinkID != "" {
		params.Set("order_link_id", orderLinkID)
	}
	if takeProfit != 0 {
		params.Set("take_profit", strconv.FormatFloat(takeProfit, 'f', -1, 64))
	}
	if stopLoss != 0 {
		params.Set("stop_loss", strconv.FormatFloat(stopLoss, 'f', -1, 64))
	}
	if takeProfitTriggerBy != "" {
		params.Set("tp_trigger_by", takeProfitTriggerBy)
	}
	if stopLossTriggerBy != "" {
		params.Set("sl_trigger_by", stopLossTriggerBy)
	}
	return resp.Result, by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresCreateConditionalOrder, params, &resp, FuturesCreateConditionalOrderRate)
}

// GetConditionalFuturesOrders gets list of futures conditional orders
func (by *Bybit) GetConditionalFuturesOrders(ctx context.Context, symbol currency.Pair, stopOrderStatus, direction, cursor string, limit int64) ([]FuturesConditionalOrders, error) {
	resp := struct {
		Result struct {
			Result []FuturesConditionalOrders `json:"data"`
			Cursor string                     `json:"cursor"`
		} `json:"result"`
		Error
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Result.Result, err
	}
	params.Set("symbol", symbolValue)
	if stopOrderStatus != "" {
		params.Set("stop_order_status", stopOrderStatus)
	}
	if direction != "" {
		params.Set("direction", direction)
	}
	if limit > 0 && limit <= 50 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	return resp.Result.Result, by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, futuresGetConditionalOrders, params, &resp, FuturesGetConditionalOrderRate)
}

// CancelConditionalFuturesOrders cancels untriggered conditional orders
func (by *Bybit) CancelConditionalFuturesOrders(ctx context.Context, symbol currency.Pair, stopOrderID, orderLinkID string) (string, error) {
	resp := struct {
		Result struct {
			StopOrderID string `json:"stop_order_id"`
		} `json:"result"`
		Error
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return "", err
	}
	params.Set("symbol", symbolValue)
	if stopOrderID == "" && orderLinkID == "" {
		return "", errStopOrderOrOrderLinkIDMissing
	}
	if stopOrderID != "" {
		params.Set("stop_order_id", stopOrderID)
	}
	if orderLinkID != "" {
		params.Set("order_link_id", orderLinkID)
	}
	return resp.Result.StopOrderID, by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresCancelConditionalOrder, params, &resp, FuturesCancelConditionalOrderRate)
}

// CancelAllConditionalFuturesOrders cancels all untriggered conditional orders
func (by *Bybit) CancelAllConditionalFuturesOrders(ctx context.Context, symbol currency.Pair) ([]FuturesCancelOrderResp, error) {
	resp := struct {
		Result []FuturesCancelOrderResp `json:"result"`
		Error
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Result, err
	}
	params.Set("symbol", symbolValue)
	return resp.Result, by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresCancelAllConditionalOrders, params, &resp, FuturesCancelAllConditionalOrderRate)
}

// ReplaceConditionalFuturesOrders modify unfilled or partially filled conditional orders
func (by *Bybit) ReplaceConditionalFuturesOrders(ctx context.Context, symbol currency.Pair, stopOrderID, orderLinkID, takeProfitTriggerBy, stopLossTriggerBy string,
	updatedQty, updatedPrice, takeProfitPrice, stopLossPrice, orderTriggerPrice float64) (string, error) {
	resp := struct {
		Result struct {
			OrderID string `json:"stop_order_id"`
		} `json:"result"`
		Error
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return "", err
	}
	params.Set("symbol", symbolValue)
	if stopOrderID == "" && orderLinkID == "" {
		return "", errStopOrderOrOrderLinkIDMissing
	}
	if stopOrderID != "" {
		params.Set("stop_order_id", stopOrderID)
	}
	if orderLinkID != "" {
		params.Set("order_link_id", orderLinkID)
	}
	if updatedQty != 0 {
		params.Set("p_r_qty", strconv.FormatFloat(updatedQty, 'f', -1, 64))
	}
	if updatedPrice != 0 {
		params.Set("p_r_price", strconv.FormatFloat(updatedPrice, 'f', -1, 64))
	}
	if orderTriggerPrice != 0 {
		params.Set("p_r_trigger_price", strconv.FormatFloat(orderTriggerPrice, 'f', -1, 64))
	}
	if takeProfitPrice != 0 {
		params.Set("take_profit", strconv.FormatFloat(takeProfitPrice, 'f', -1, 64))
	}
	if stopLossPrice != 0 {
		params.Set("stop_loss", strconv.FormatFloat(stopLossPrice, 'f', -1, 64))
	}
	if takeProfitTriggerBy != "" {
		params.Set("tp_trigger_by", takeProfitTriggerBy)
	}
	if stopLossTriggerBy != "" {
		params.Set("sl_trigger_by", stopLossTriggerBy)
	}
	return resp.Result.OrderID, by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresReplaceConditionalOrder, params, &resp, FuturesReplaceConditionalOrderRate)
}

// GetConditionalRealtimeOrders query real time conditional order data
func (by *Bybit) GetConditionalRealtimeOrders(ctx context.Context, symbol currency.Pair, stopOrderID, orderLinkID string) ([]FuturesConditionalRealtimeOrder, error) {
	var data []FuturesConditionalRealtimeOrder
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return data, err
	}
	params.Set("symbol", symbolValue)
	if stopOrderID != "" {
		params.Set("stop_order_id", stopOrderID)
	}
	if orderLinkID != "" {
		params.Set("order_link_id", orderLinkID)
	}

	if stopOrderID == "" && orderLinkID == "" {
		resp := struct {
			Result []FuturesConditionalRealtimeOrder `json:"result"`
			Error
		}{}
		err = by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, futuresGetConditionalRealtimeOrders, params, &resp, FuturesGetConditionalRealtimeOrderRate)
		if err != nil {
			return data, err
		}
		data = append(data, resp.Result...)
	} else {
		resp := struct {
			Result FuturesConditionalRealtimeOrder `json:"result"`
			Error
		}{}
		err = by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, futuresGetConditionalRealtimeOrders, params, &resp, FuturesGetConditionalRealtimeOrderRate)
		if err != nil {
			return data, err
		}
		data = append(data, resp.Result)
	}
	return data, nil
}

// GetPositions returns list of user positions
func (by *Bybit) GetPositions(ctx context.Context, symbol currency.Pair) ([]PositionResp, error) {
	var data []PositionResp
	params := url.Values{}
	resp := struct {
		Result []struct {
			Data    PositionResp `json:"data"`
			IsValid bool         `json:"is_valid"`
		} `json:"result"`
		Error
	}{}

	if !symbol.IsEmpty() {
		symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
		if err != nil {
			return data, err
		}
		params.Set("symbol", symbolValue)
	}
	err := by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, futuresPosition, params, &resp, FuturesPositionRate)
	if err != nil {
		return data, err
	}
	for x := range resp.Result {
		data = append(data, resp.Result[x].Data)
	}
	return data, nil
}

// SetMargin updates margin
func (by *Bybit) SetMargin(ctx context.Context, positionMode int64, symbol currency.Pair, margin string) (float64, error) {
	resp := struct {
		Result float64 `json:"result"`
		Error
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Result, err
	}
	params.Set("symbol", symbolValue)
	if positionMode < 0 || positionMode > 2 {
		return resp.Result, errInvalidPositionMode
	}
	params.Set("position_idx", strconv.FormatInt(positionMode, 10))

	if margin == "" {
		return resp.Result, errInvalidMargin
	}
	params.Set("margin", margin)

	return resp.Result, by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresUpdateMargin, params, &resp, FuturesUpdateMarginRate)
}

// SetTradingAndStop sets take profit, stop loss, and trailing stop for your open position
func (by *Bybit) SetTradingAndStop(ctx context.Context, positionMode int64, symbol currency.Pair, takeProfit, stopLoss, trailingStop, newTrailingActive, stopLossQty, takeProfitQty float64, takeProfitTriggerBy, stopLossTriggerBy string) (SetTradingAndStopResp, error) {
	resp := struct {
		Result SetTradingAndStopResp `json:"result"`
		Error
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Result, err
	}
	params.Set("symbol", symbolValue)
	if positionMode < 0 || positionMode > 2 {
		return resp.Result, errInvalidPositionMode
	}
	params.Set("position_idx", strconv.FormatInt(positionMode, 10))

	if takeProfit >= 0 {
		params.Set("take_profit", strconv.FormatFloat(takeProfit, 'f', -1, 64))
	}
	if stopLoss >= 0 {
		params.Set("stop_loss", strconv.FormatFloat(stopLoss, 'f', -1, 64))
	}
	if trailingStop >= 0 {
		params.Set("trailing_stop", strconv.FormatFloat(trailingStop, 'f', -1, 64))
	}
	if newTrailingActive != 0 {
		params.Set("new_trailing_active", strconv.FormatFloat(newTrailingActive, 'f', -1, 64))
	}
	if stopLossQty != 0 {
		params.Set("sl_size", strconv.FormatFloat(stopLossQty, 'f', -1, 64))
	}
	if takeProfitQty != 0 {
		params.Set("tp_size", strconv.FormatFloat(takeProfitQty, 'f', -1, 64))
	}
	if takeProfitTriggerBy != "" {
		params.Set("tp_trigger_by", takeProfitTriggerBy)
	}
	if stopLossTriggerBy != "" {
		params.Set("sl_trigger_by", stopLossTriggerBy)
	}
	return resp.Result, by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresSetTradingStop, params, &resp, FuturesSetTradingStopRate)
}

// SetLeverage sets leverage
func (by *Bybit) SetLeverage(ctx context.Context, symbol currency.Pair, buyLeverage, sellLeverage float64) (float64, error) {
	resp := struct {
		Result float64 `json:"result"`
		Error
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Result, err
	}
	params.Set("symbol", symbolValue)
	params.Set("buy_leverage", strconv.FormatFloat(buyLeverage, 'f', -1, 64))
	params.Set("sell_leverage", strconv.FormatFloat(sellLeverage, 'f', -1, 64))

	return resp.Result, by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresSetLeverage, params, &resp, FuturesSetLeverageRate)
}

// ChangeMode switches mode between One-Way or Hedge Mode
func (by *Bybit) ChangePositionMode(ctx context.Context, symbol currency.Pair, mode int64) error {
	var resp Error
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return err
	}
	params.Set("symbol", symbolValue)
	params.Set("mode", strconv.FormatInt(mode, 10))

	return by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresSwitchPositionMode, params, &resp, FuturesSwitchPositionModeRate)
}

// ChangeMode switches mode between full or partial position
func (by *Bybit) ChangeMode(ctx context.Context, symbol currency.Pair, takeProfitStopLoss string) (string, error) {
	resp := struct {
		Result struct {
			Mode string `json:"tp_sl_mode"`
		} `json:"result"`
		Error
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Result.Mode, err
	}
	params.Set("symbol", symbolValue)
	if takeProfitStopLoss == "" {
		return resp.Result.Mode, errInvalidTakeProfitStopLoss
	}
	params.Set("tp_sl_mode", takeProfitStopLoss)

	return resp.Result.Mode, by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresSwitchPosition, params, &resp, FuturesSwitchPositionRate)
}

// ChangeMargin switches margin between cross or isolated
func (by *Bybit) ChangeMargin(ctx context.Context, symbol currency.Pair, buyLeverage, sellLeverage float64, isIsolated bool) error {
	var resp Error
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return err
	}
	params.Set("symbol", symbolValue)
	params.Set("buy_leverage", strconv.FormatFloat(buyLeverage, 'f', -1, 64))
	params.Set("sell_leverage", strconv.FormatFloat(sellLeverage, 'f', -1, 64))

	if isIsolated {
		params.Set("is_isolated", "true")
	} else {
		params.Set("is_isolated", "false")
	}

	return by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresSwitchMargin, params, &resp, FuturesSwitchMarginRate)
}

// GetTradeRecords returns list of user trades
func (by *Bybit) GetTradeRecords(ctx context.Context, symbol currency.Pair, orderID, order string, startTime, page, limit int64) ([]TradeResp, error) {
	params := url.Values{}
	resp := struct {
		Data struct {
			OrderID string      `json:"order_id"`
			Trades  []TradeResp `json:"trade_list"`
		} `json:"result"`
		Error
	}{}

	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Data.Trades, err
	}
	params.Set("symbol", symbolValue)

	if orderID != "" {
		params.Set("order_id", orderID)
	}
	if order != "" {
		params.Set("order", order)
	}
	if startTime != 0 {
		params.Set("start_time", strconv.FormatInt(startTime, 10))
	}
	if page != 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}

	return resp.Data.Trades, by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, futuresGetTrades, params, &resp, FuturesGetTradeRate)
}

// GetClosedTrades returns closed profit and loss records
func (by *Bybit) GetClosedTrades(ctx context.Context, symbol currency.Pair, executionType string, startTime, endTime, page, limit int64) ([]ClosedTrades, error) {
	params := url.Values{}
	resp := struct {
		Data struct {
			CurrentPage int64          `json:"current_page"`
			Trades      []ClosedTrades `json:"data"`
		} `json:"result"`
		Error
	}{}

	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Data.Trades, err
	}
	params.Set("symbol", symbolValue)

	if executionType != "" {
		params.Set("execution_type", executionType)
	}
	if startTime != 0 {
		params.Set("start_time", strconv.FormatInt(startTime, 10))
	}
	if endTime != 0 {
		params.Set("end_time", strconv.FormatInt(endTime, 10))
	}
	if page > 0 && page <= 50 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 && limit <= 50 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}

	return resp.Data.Trades, by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodGet, futuresGetClosedTrades, params, &resp, FuturesDefaultRate)
}

// SetRiskLimit sets risk limit
func (by *Bybit) SetRiskLimit(ctx context.Context, symbol currency.Pair, riskID, positionMode int64) (int64, error) {
	resp := struct {
		Result struct {
			RiskID int64 `json:"risk_id"`
		} `json:"result"`
		Error
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Result.RiskID, err
	}
	params.Set("symbol", symbolValue)

	if riskID <= 0 {
		return resp.Result.RiskID, errInvalidRiskID
	}
	params.Set("risk_id", strconv.FormatInt(riskID, 10))

	if positionMode < 0 || positionMode > 2 {
		return resp.Result.RiskID, errInvalidPositionMode
	}
	params.Set("position_idx", strconv.FormatInt(positionMode, 10))
	return resp.Result.RiskID, by.SendAuthHTTPRequest(ctx, exchange.RestFutures, http.MethodPost, futuresSetRiskLimit, params, &resp, FuturesDefaultRate)
}
