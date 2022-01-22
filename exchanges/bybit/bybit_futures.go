package bybit

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// TODO: handle rate limiting
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
func (by *Bybit) CreateFuturesOrder(positionMode int64, symbol currency.Pair, side, orderType, timeInForce,
	orderLinkID, takeProfitTriggerBy, stopLossTriggerBy string,
	quantity, price, takeProfit, stopLoss float64, closeOnTrigger, reduceOnly bool) (FuturesOrderDataResp, error) {
	resp := struct {
		Result FuturesOrderDataResp `json:"result"`
	}{}

	params := url.Values{}
	if positionMode >= 0 && positionMode <= 2 {
		params.Set("position_idx", strconv.FormatInt(positionMode, 10))
	} else {
		return resp.Result, errors.New("position mode is invalid")
	}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Result, err
	}
	params.Set("symbol", symbolValue)
	params.Set("side", side)
	params.Set("order_type", orderType)
	if quantity != 0 {
		params.Set("qty", strconv.FormatFloat(quantity, 'f', -1, 64))
	} else {
		return resp.Result, errors.New("quantity can't be zero or missing")
	}
	if price != 0 {
		params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	}
	if timeInForce != "" {
		params.Set("time_in_force", timeInForce)
	} else {
		return resp.Result, errors.New("timeInForce can't be empty or missing")
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
	if reduceOnly {
		params.Set("reduce_only", "true")
	}
	return resp.Result, by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodPost, futuresCreateOrder, params, &resp, FuturesCreateOrderRate)
}

// GetActiveFuturesOrders gets list of futures active orders
func (by *Bybit) GetActiveFuturesOrders(symbol currency.Pair, orderStatus, direction, cursor string, limit int64) ([]FuturesActiveOrder, error) {
	resp := struct {
		Result struct {
			Data   []FuturesActiveOrder `json:"data"`
			Cursor string               `json:"cursor"`
		} `json:"result"`
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
	return resp.Result.Data, by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodGet, futuresGetActiveOrders, params, &resp, FuturesGetActiveOrderRate)
}

// CancelActiveFuturesOrders cancels futures unfilled or partially filled orders
func (by *Bybit) CancelActiveFuturesOrders(symbol currency.Pair, orderID, orderLinkID string) (FuturesOrderCancelResp, error) {
	resp := struct {
		Result FuturesOrderCancelResp `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Result, err
	}
	params.Set("symbol", symbolValue)
	if orderID == "" && orderLinkID == "" {
		return resp.Result, errors.New("one among orderID or orderLinkID should be present")
	}
	if orderID != "" {
		params.Set("order_id", orderID)
	}
	if orderLinkID != "" {
		params.Set("order_link_id", orderLinkID)
	}
	return resp.Result, by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodPost, futuresCancelActiveOrder, params, &resp, FuturesCancelOrderRate)
}

// CancelAllActiveFuturesOrders cancels all futures unfilled or partially filled orders
func (by *Bybit) CancelAllActiveFuturesOrders(symbol currency.Pair) ([]FuturesCancelOrderData, error) {
	resp := struct {
		Result []FuturesCancelOrderData `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Result, err
	}
	params.Set("symbol", symbolValue)
	return resp.Result, by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodPost, futuresCancelAllActiveOrders, params, &resp, FuturesCancelAllOrderRate)
}

// ReplaceActiveFuturesOrders modify unfilled or partially filled orders
func (by *Bybit) ReplaceActiveFuturesOrders(symbol currency.Pair, orderID, orderLinkID, takeProfitTriggerBy, stopLossTriggerBy string,
	updatedQty, updatedPrice, takeProfitPrice, stopLossPrice float64) (string, error) {
	resp := struct {
		Result struct {
			OrderID string `json:"order_id"`
		} `json:"result"`
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return "", err
	}
	params.Set("symbol", symbolValue)
	if orderID == "" && orderLinkID == "" {
		return "", errors.New("one among orderID or orderLinkID should be present")
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
	return resp.Result.OrderID, by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodPost, futuresReplaceActiveOrder, params, &resp, FuturesReplaceOrderRate)
}

// GetActiveRealtimeOrders query real time order data
func (by *Bybit) GetActiveRealtimeOrders(symbol currency.Pair, orderID, orderLinkID string) ([]FuturesActiveRealtimeOrder, error) {
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
		}{}
		err = by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodGet, futuresGetActiveRealtimeOrders, params, &resp, FuturesGetActiveRealtimeOrderRate)
		if err != nil {
			return data, err
		}
		for _, d := range resp.Result {
			data = append(data, d)
		}
	} else {
		resp := struct {
			Result FuturesActiveRealtimeOrder `json:"result"`
		}{}
		err = by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodGet, futuresGetActiveRealtimeOrders, params, &resp, FuturesGetActiveRealtimeOrderRate)
		if err != nil {
			return data, err
		}
		data = append(data, resp.Result)
	}
	return data, nil
}

// CreateConditionalFuturesOrder sends a new conditional futures order to the exchange
func (by *Bybit) CreateConditionalFuturesOrder(positionMode int64, symbol currency.Pair, side, orderType, timeInForce,
	orderLinkID, takeProfitTriggerBy, stopLossTriggerBy, triggerBy string,
	quantity, price, takeProfit, stopLoss, basePrice, stopPrice float64, closeOnTrigger bool) (FuturesConditionalOrderResp, error) {
	resp := struct {
		Result FuturesConditionalOrderResp `json:"result"`
	}{}
	params := url.Values{}
	if positionMode >= 0 && positionMode <= 2 {
		params.Set("position_idx", strconv.FormatInt(positionMode, 10))
	} else {
		return resp.Result, errors.New("position mode is invalid")
	}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Result, err
	}
	params.Set("symbol", symbolValue)
	params.Set("side", side)
	params.Set("order_type", orderType)
	if quantity != 0 {
		params.Set("qty", strconv.FormatFloat(quantity, 'f', -1, 64))
	} else {
		return resp.Result, errors.New("quantity can't be zero or missing")
	}
	if price != 0 {
		params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	}
	if basePrice != 0 {
		params.Set("base_price", strconv.FormatFloat(basePrice, 'f', -1, 64))
	} else {
		return resp.Result, errors.New("basePrice can't be empty or missing")
	}
	if stopPrice != 0 {
		params.Set("stop_px", strconv.FormatFloat(stopPrice, 'f', -1, 64))
	} else {
		return resp.Result, errors.New("stopPrice can't be empty or missing")
	}
	if timeInForce != "" {
		params.Set("time_in_force", timeInForce)
	} else {
		return resp.Result, errors.New("timeInForce can't be empty or missing")
	}
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
	return resp.Result, by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodPost, futuresCreateConditionalOrder, params, &resp, FuturesCreateConditionalOrderRate)
}

// GetConditionalFuturesOrders gets list of futures conditional orders
func (by *Bybit) GetConditionalFuturesOrders(symbol currency.Pair, stopOrderStatus, direction, cursor string, limit int64) ([]FuturesConditionalOrders, error) {
	resp := struct {
		Result struct {
			Result []FuturesConditionalOrders `json:"data"`
			Cursor string                     `json:"cursor"`
		} `json:"result"`
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
	return resp.Result.Result, by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodGet, futuresGetConditionalOrders, params, &resp, FuturesGetConditionalOrderRate)
}

// CancelConditionalFuturesOrders cancels untriggered conditional orders
func (by *Bybit) CancelConditionalFuturesOrders(symbol currency.Pair, stopOrderID, orderLinkID string) (string, error) {
	resp := struct {
		Result struct {
			StopOrderID string `json:"stop_order_id"`
		} `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return "", err
	}
	params.Set("symbol", symbolValue)
	if stopOrderID == "" && orderLinkID == "" {
		return "", errors.New("one among stopOrderID or orderLinkID should be present")
	}
	if stopOrderID != "" {
		params.Set("stop_order_id", stopOrderID)
	}
	if orderLinkID != "" {
		params.Set("order_link_id", orderLinkID)
	}
	return resp.Result.StopOrderID, by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodPost, futuresCancelConditionalOrder, params, &resp, FuturesCancelCondtionalOrderRate)
}

// CancelAllConditionalFuturesOrders cancels all untriggered conditional orders
func (by *Bybit) CancelAllConditionalFuturesOrders(symbol currency.Pair) ([]FuturesCancelOrderResp, error) {
	resp := struct {
		Result []FuturesCancelOrderResp `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Result, err
	}
	params.Set("symbol", symbolValue)
	return resp.Result, by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodPost, futuresCancelAllConditionalOrders, params, &resp, FuturesCancelAllCondtionalOrderRate)
}

// ReplaceConditionalFuturesOrders modify unfilled or partially filled conditional orders
func (by *Bybit) ReplaceConditionalFuturesOrders(symbol currency.Pair, stopOrderID, orderLinkID, takeProfitTriggerBy, stopLossTriggerBy string,
	updatedQty, updatedPrice, takeProfitPrice, stopLossPrice, orderTriggerPrice float64) (string, error) {
	resp := struct {
		Result struct {
			OrderID string `json:"stop_order_id"`
		} `json:"result"`
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return "", err
	}
	params.Set("symbol", symbolValue)
	if stopOrderID == "" && orderLinkID == "" {
		return "", errors.New("one among stopOrderID or orderLinkID should be present")
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
	return resp.Result.OrderID, by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodPost, futuresReplaceConditionalOrder, params, &resp, FuturesReplaceConditionalOrderRate)
}

// GetConditionalRealtimeOrders query real time conditional order data
func (by *Bybit) GetConditionalRealtimeOrders(symbol currency.Pair, stopOrderID, orderLinkID string) ([]FuturesConditionalRealtimeOrder, error) {
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
		}{}
		err = by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodGet, futuresGetConditionalRealtimeOrders, params, &resp, FuturesGetConditionalRealtimeOrderRate)
		if err != nil {
			return data, err
		}
		for _, d := range resp.Result {
			data = append(data, d)
		}
	} else {
		resp := struct {
			Result FuturesConditionalRealtimeOrder `json:"result"`
		}{}
		err = by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodGet, futuresGetConditionalRealtimeOrders, params, &resp, FuturesGetConditionalRealtimeOrderRate)
		if err != nil {
			return data, err
		}
		data = append(data, resp.Result)
	}
	return data, nil
}

// GetPositions returns list of user positions
func (by *Bybit) GetPositions(symbol currency.Pair) ([]PositionResp, error) {
	var data []PositionResp
	params := url.Values{}
	resp := struct {
		Result []struct {
			Data    PositionResp `json:"data"`
			IsValid bool         `json:"is_valid"`
		} `json:"result"`
	}{}

	if !symbol.IsEmpty() {
		symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
		if err != nil {
			return data, err
		}
		params.Set("symbol", symbolValue)
	}
	err := by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodGet, futuresPosition, params, &resp, FuturesPositionRate)
	if err != nil {
		return data, err
	}
	for _, d := range resp.Result {
		data = append(data, d.Data)
	}
	return data, nil
}

// SetMargin updates margin
func (by *Bybit) SetMargin(positionMode int64, symbol currency.Pair, margin string) (float64, error) {
	resp := struct {
		Result float64 `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Result, err
	}
	params.Set("symbol", symbolValue)
	if positionMode >= 0 && positionMode <= 2 {
		params.Set("position_idx", strconv.FormatInt(positionMode, 10))
	} else {
		return resp.Result, errors.New("position mode is invalid")
	}
	if margin != "" {
		params.Set("margin", margin)
	} else {
		return resp.Result, errors.New("margin can't be empty")
	}
	return resp.Result, by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodPost, futuresUpdateMargin, params, &resp, FuturesUpdateMarginRate)
}

// SetTradingAndStop sets take profit, stop loss, and trailing stop for your open position
func (by *Bybit) SetTradingAndStop(positionMode int64, symbol currency.Pair, takeProfit, stopLoss, trailingStop, newTrailingActive, stopLossQty, takeProfitQty float64, takeProfitTriggerBy, stopLossTriggerBy string) (SetTradingAndStopResp, error) {
	resp := struct {
		Result SetTradingAndStopResp `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Result, err
	}
	params.Set("symbol", symbolValue)
	if positionMode >= 0 && positionMode <= 2 {
		params.Set("position_idx", strconv.FormatInt(positionMode, 10))
	} else {
		return resp.Result, errors.New("position mode is invalid")
	}
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
	return resp.Result, by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodPost, futuresSetTradingStop, params, &resp, FuturesSetTradingStopRate)
}

// SetLeverage sets leverage
func (by *Bybit) SetLeverage(symbol currency.Pair, buyLeverage, sellLeverage float64) (float64, error) {
	resp := struct {
		Result float64 `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Result, err
	}
	params.Set("symbol", symbolValue)
	params.Set("buy_leverage", strconv.FormatFloat(buyLeverage, 'f', -1, 64))
	params.Set("sell_leverage", strconv.FormatFloat(sellLeverage, 'f', -1, 64))

	return resp.Result, by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodPost, futuresSetLeverage, params, &resp, FuturesSetLeverateRate)
}

// ChangeMode switches mode between One-Way or Hedge Mode
func (by *Bybit) ChangePositionMode(symbol currency.Pair, mode int64) error {
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return err
	}
	params.Set("symbol", symbolValue)
	params.Set("mode", strconv.FormatInt(mode, 10))

	return by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodPost, futuresSwitchPositionMode, params, &struct{}{}, FuturesSwitchPositionModeRate)
}

// ChangeCoinMode switches mode between full or partial position
func (by *Bybit) ChangeMode(symbol currency.Pair, takeProfitStopLoss string) (string, error) {
	resp := struct {
		Result struct {
			Mode string `json:"tp_sl_mode"`
		} `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Result.Mode, err
	}
	params.Set("symbol", symbolValue)
	if takeProfitStopLoss != "" {
		params.Set("tp_sl_mode", takeProfitStopLoss)
	} else {
		return resp.Result.Mode, errors.New("takeProfitStopLoss can't be empty or missing")
	}

	return resp.Result.Mode, by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodPost, futuresSwitchPosition, params, &resp, FuturesSwitchPositionRate)
}

// ChangeMargin switches margin between cross or isolated
func (by *Bybit) ChangeMargin(symbol currency.Pair, buyLeverage, sellLeverage float64, isIsolated bool) error {
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

	return by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodPost, futuresSwitchMargin, params, &struct{}{}, FuturesSwitchMarginRate)
}

// GetTradeRecords returns list of user trades
func (by *Bybit) GetTradeRecords(symbol currency.Pair, orderID, order string, startTime, page, limit int64) ([]TradeResp, error) {
	params := url.Values{}
	resp := struct {
		Data struct {
			OrderID string      `json:"order_id"`
			Trades  []TradeResp `json:"trade_list"`
		} `json:"result"`
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

	return resp.Data.Trades, by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodGet, futuresGetTrades, params, &resp, FuturesGetTradeRate)
}

// GetClosedTrades returns closed profit and loss records
func (by *Bybit) GetClosedTrades(symbol currency.Pair, executionType string, startTime, endTime, page, limit int64) ([]ClosedTrades, error) {
	params := url.Values{}
	resp := struct {
		Data struct {
			CurrentPage int64          `json:"current_page"`
			Trades      []ClosedTrades `json:"data"`
		} `json:"result"`
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

	return resp.Data.Trades, by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodGet, futuresGetClosedTrades, params, &resp, FuturesDefaultRate)
}

// SetRiskLimit sets risk limit
func (by *Bybit) SetRiskLimit(symbol currency.Pair, riskID, positionMode int64) (int64, error) {
	resp := struct {
		Result struct {
			RiskID int64 `json:"risk_id"`
		} `json:"result"`
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.Futures)
	if err != nil {
		return resp.Result.RiskID, err
	}
	params.Set("symbol", symbolValue)

	if riskID > 0 {
		params.Set("risk_id", strconv.FormatInt(riskID, 10))
	} else {
		return resp.Result.RiskID, errors.New("riskID can't be zero or lesser")
	}

	if positionMode >= 0 && positionMode <= 2 {
		params.Set("position_idx", strconv.FormatInt(positionMode, 10))
	} else {
		return resp.Result.RiskID, errors.New("position mode is invalid")
	}
	return resp.Result.RiskID, by.SendAuthHTTPRequest(exchange.RestFutures, http.MethodPost, futuresSetRiskLimit, params, &resp, FuturesDefaultRate)
}
