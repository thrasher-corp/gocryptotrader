package bybit

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const (

	// public endpoint
	ufuturesKline              = "/public/linear/kline"
	ufuturesRecentTrades       = "/public/linear/recent-trading-records"
	ufuturesMarkPriceKline     = "/public/linear/mark-price-kline"
	ufuturesIndexKline         = "/public/linear/index-price-kline"
	ufuturesIndexPremiumKline  = "/public/linear/premium-index-kline"
	ufuturesGetLastFundingRate = "/public/linear/funding/prev-funding-rate"
	ufuturesGetRiskLimit       = "/public/linear/risk-limit"

	// auth endpoint
	ufuturesCreateOrder             = "/private/linear/order/create"
	ufuturesGetActiveOrders         = "/private/linear/order/list"
	ufuturesCancelActiveOrder       = "/private/linear/order/cancel"
	ufuturesCancelAllActiveOrders   = "/private/linear/order/cancel-all"
	ufuturesReplaceActiveOrder      = "/private/linear/order/replace"
	ufuturesGetActiveRealtimeOrders = "/private/linear/order/search"

	ufuturesCreateConditionalOrder       = "/private/linear/stop-order/create"
	ufuturesGetConditionalOrders         = "/private/linear/stop-order/list"
	ufuturesCancelConditionalOrder       = "/private/linear/stop-order/cancel"
	ufuturesCancelAllConditionalOrders   = "/private/linear/stop-order/cancel-all"
	ufuturesReplaceConditionalOrder      = "/private/linear/stop-order/replace"
	ufuturesGetConditionalRealtimeOrders = "/private/linear/stop-order/search"

	ufuturesPosition           = "/private/linear/position/list"
	ufuturesSetAutoAddMargin   = "/private/linear/position/set-auto-add-margin"
	ufuturesSwitchMargin       = "/private/linear/position/switch-isolated"
	ufuturesSwitchPositionMode = "/private/linear/position/switch-mode"
	ufuturesSwitchPosition     = "/private/linear/tpsl/switch-mode"
	ufuturesUpdateMargin       = "/private/linear/position/add-margin"
	ufuturesSetLeverage        = "/private/linear/position/set-leverage"
	ufuturesSetTradingStop     = "/private/linear/position/trading-stop"
	ufuturesGetTrades          = "/private/linear/trade/execution/list"
	ufuturesGetClosedTrades    = "/private/linear/trade/closed-pnl/list"

	ufuturesSetRiskLimit        = "/private/linear/position/set-risk"
	ufuturesPredictFundingRate  = "/private/linear/funding/predicted-funding"
	ufuturesGetMyLastFundingFee = "/private/linear/funding/prev-funding"
)

// GetUSDTFuturesKlineData gets futures kline data for USDTMarginedFutures.
func (by *Bybit) GetUSDTFuturesKlineData(ctx context.Context, symbol currency.Pair, interval string, limit int64, startTime time.Time) ([]FuturesCandleStick, error) {
	resp := struct {
		Data []FuturesCandleStick `json:"result"`
		Error
	}{}

	params := url.Values{}
	if symbol.IsEmpty() {
		return resp.Data, errSymbolMissing
	}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)

	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesIntervals, interval) {
		return resp.Data, errInvalidInterval
	}
	if startTime.IsZero() {
		return nil, errInvalidStartTime
	}
	params.Set("interval", interval)
	params.Set("from", strconv.FormatInt(startTime.Unix(), 10))

	path := common.EncodeURLValues(ufuturesKline, params)
	return resp.Data, by.SendHTTPRequest(ctx, exchange.RestUSDTMargined, path, publicFuturesRate, &resp)
}

// GetUSDTPublicTrades gets past public trades for USDTMarginedFutures.
func (by *Bybit) GetUSDTPublicTrades(ctx context.Context, symbol currency.Pair, limit int64) ([]FuturesPublicTradesData, error) {
	resp := struct {
		Data []FuturesPublicTradesData `json:"result"`
		Error
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}

	path := common.EncodeURLValues(ufuturesRecentTrades, params)
	return resp.Data, by.SendHTTPRequest(ctx, exchange.RestUSDTMargined, path, publicFuturesRate, &resp)
}

// GetUSDTMarkPriceKline gets mark price kline data for USDTMarginedFutures.
func (by *Bybit) GetUSDTMarkPriceKline(ctx context.Context, symbol currency.Pair, interval string, limit int64, startTime time.Time) ([]MarkPriceKlineData, error) {
	resp := struct {
		Data []MarkPriceKlineData `json:"result"`
		Error
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesIntervals, interval) {
		return resp.Data, errInvalidInterval
	}
	params.Set("interval", interval)
	if startTime.IsZero() {
		return resp.Data, errInvalidStartTime
	}
	params.Set("from", strconv.FormatInt(startTime.Unix(), 10))

	path := common.EncodeURLValues(ufuturesMarkPriceKline, params)
	return resp.Data, by.SendHTTPRequest(ctx, exchange.RestUSDTMargined, path, publicFuturesRate, &resp)
}

// GetUSDTIndexPriceKline gets index price kline data for USDTMarginedFutures.
func (by *Bybit) GetUSDTIndexPriceKline(ctx context.Context, symbol currency.Pair, interval string, limit int64, startTime time.Time) ([]IndexPriceKlineData, error) {
	resp := struct {
		Data []IndexPriceKlineData `json:"result"`
		Error
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesIntervals, interval) {
		return resp.Data, errInvalidInterval
	}
	params.Set("interval", interval)
	if startTime.IsZero() {
		return resp.Data, errInvalidStartTime
	}
	params.Set("from", strconv.FormatInt(startTime.Unix(), 10))

	path := common.EncodeURLValues(ufuturesIndexKline, params)
	return resp.Data, by.SendHTTPRequest(ctx, exchange.RestUSDTMargined, path, publicFuturesRate, &resp)
}

// GetUSDTPremiumIndexPriceKline gets premium index price kline data for USDTMarginedFutures.
func (by *Bybit) GetUSDTPremiumIndexPriceKline(ctx context.Context, symbol currency.Pair, interval string, limit int64, startTime time.Time) ([]IndexPriceKlineData, error) {
	resp := struct {
		Data []IndexPriceKlineData `json:"result"`
		Error
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesIntervals, interval) {
		return resp.Data, errInvalidInterval
	}
	params.Set("interval", interval)
	if startTime.IsZero() {
		return resp.Data, errInvalidStartTime
	}
	params.Set("from", strconv.FormatInt(startTime.Unix(), 10))

	path := common.EncodeURLValues(ufuturesIndexPremiumKline, params)
	return resp.Data, by.SendHTTPRequest(ctx, exchange.RestUSDTMargined, path, publicFuturesRate, &resp)
}

// GetUSDTLastFundingRate returns latest generated funding fee
func (by *Bybit) GetUSDTLastFundingRate(ctx context.Context, symbol currency.Pair) (USDTFundingInfo, error) {
	resp := struct {
		Data USDTFundingInfo `json:"result"`
		Error
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)

	path := common.EncodeURLValues(ufuturesGetLastFundingRate, params)
	return resp.Data, by.SendHTTPRequest(ctx, exchange.RestUSDTMargined, path, publicFuturesRate, &resp)
}

// GetUSDTRiskLimit returns risk limit
func (by *Bybit) GetUSDTRiskLimit(ctx context.Context, symbol currency.Pair) ([]RiskInfo, error) {
	resp := struct {
		Data []RiskInfo `json:"result"`
		Error
	}{}

	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return resp.Data, err
		}
		params.Set("symbol", symbolValue)
	}

	path := common.EncodeURLValues(ufuturesGetRiskLimit, params)
	return resp.Data, by.SendHTTPRequest(ctx, exchange.RestUSDTMargined, path, publicFuturesRate, &resp)
}

// CreateUSDTFuturesOrder sends a new USDT futures order to the exchange
func (by *Bybit) CreateUSDTFuturesOrder(ctx context.Context, symbol currency.Pair, side, orderType, timeInForce,
	orderLinkID, takeProfitTriggerBy, stopLossTriggerBy string,
	quantity, price, takeProfit, stopLoss float64, closeOnTrigger, reduceOnly bool) (FuturesOrderDataResp, error) {
	resp := struct {
		Data FuturesOrderDataResp `json:"result"`
		Error
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	if side == "" {
		return resp.Data, errInvalidSide
	}
	params.Set("side", side)

	if orderType == "" {
		return resp.Data, errInvalidOrderType
	}
	params.Set("order_type", orderType)

	if quantity <= 0 {
		return resp.Data, errInvalidQuantity
	}
	params.Set("qty", strconv.FormatFloat(quantity, 'f', -1, 64))

	if price != 0 {
		params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	}
	if timeInForce == "" {
		return resp.Data, errInvalidTimeInForce
	}
	params.Set("time_in_force", timeInForce)

	if closeOnTrigger {
		params.Set("close_on_trigger", "true")
	} else {
		params.Set("close_on_trigger", "false")
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
	} else {
		params.Set("reduce_only", "false")
	}
	return resp.Data, by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesCreateOrder, params, nil, &resp, uFuturesCreateOrderRate)
}

// GetActiveUSDTFuturesOrders gets list of USDT futures active orders
func (by *Bybit) GetActiveUSDTFuturesOrders(ctx context.Context, symbol currency.Pair, orderStatus, direction, orderID, orderLinkID string, page, limit int64) ([]FuturesActiveOrderResp, error) {
	resp := struct {
		Result struct {
			Data        []FuturesActiveOrderResp `json:"data"`
			CurrentPage int64                    `json:"current_page"`
			LastPage    int64                    `json:"last_page"`
		} `json:"result"`
		Error
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Result.Data, err
	}
	params.Set("symbol", symbolValue)
	if orderStatus != "" {
		params.Set("order_status", orderStatus)
	}
	if direction != "" {
		params.Set("order", direction)
	}
	if page > 0 && page <= 50 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 && limit <= 50 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if orderID != "" {
		params.Set("order_id", orderID)
	}
	if orderLinkID != "" {
		params.Set("order_link_id", orderLinkID)
	}

	return resp.Result.Data, by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesGetActiveOrders, params, nil, &resp, uFuturesGetActiveOrderRate)
}

// CancelActiveUSDTFuturesOrders cancels USDT futures unfilled or partially filled orders
func (by *Bybit) CancelActiveUSDTFuturesOrders(ctx context.Context, symbol currency.Pair, orderID, orderLinkID string) (string, error) {
	resp := struct {
		Data struct {
			OrderID string `json:"order_id"`
		} `json:"result"`
		Error
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Data.OrderID, err
	}
	params.Set("symbol", symbolValue)
	if orderID == "" && orderLinkID == "" {
		return resp.Data.OrderID, errOrderOrOrderLinkIDMissing
	}
	if orderID != "" {
		params.Set("order_id", orderID)
	}
	if orderLinkID != "" {
		params.Set("order_link_id", orderLinkID)
	}
	return resp.Data.OrderID, by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesCancelActiveOrder, params, nil, &resp, uFuturesCancelOrderRate)
}

// CancelAllActiveUSDTFuturesOrders cancels all USDT futures unfilled or partially filled orders
func (by *Bybit) CancelAllActiveUSDTFuturesOrders(ctx context.Context, symbol currency.Pair) ([]string, error) {
	resp := struct {
		Data []string `json:"result"`
		Error
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	return resp.Data, by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesCancelAllActiveOrders, params, nil, &resp, uFuturesCancelAllOrderRate)
}

// ReplaceActiveUSDTFuturesOrders modify unfilled or partially filled orders
func (by *Bybit) ReplaceActiveUSDTFuturesOrders(ctx context.Context, symbol currency.Pair, orderID, orderLinkID, takeProfitTriggerBy, stopLossTriggerBy string,
	updatedQty int64, updatedPrice, takeProfitPrice, stopLossPrice float64) (string, error) {
	resp := struct {
		Data struct {
			OrderID string `json:"order_id"`
		} `json:"result"`
		Error
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
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
		params.Set("p_r_qty", strconv.FormatInt(updatedQty, 10))
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
	return resp.Data.OrderID, by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesReplaceActiveOrder, params, nil, &resp, uFuturesDefaultRate)
}

// GetActiveUSDTRealtimeOrders query real time order data
func (by *Bybit) GetActiveUSDTRealtimeOrders(ctx context.Context, symbol currency.Pair, orderID, orderLinkID string) ([]FuturesActiveRealtimeOrder, error) {
	var data []FuturesActiveRealtimeOrder
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
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
			Data []FuturesActiveRealtimeOrder `json:"result"`
			Error
		}{}
		err = by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesGetActiveRealtimeOrders, params, nil, &resp, uFuturesGetActiveRealtimeOrderRate)
		if err != nil {
			return data, err
		}
		data = append(data, resp.Data...)
	} else {
		resp := struct {
			Data FuturesActiveRealtimeOrder `json:"result"`
			Error
		}{}
		err = by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesGetActiveRealtimeOrders, params, nil, &resp, uFuturesGetActiveRealtimeOrderRate)
		if err != nil {
			return data, err
		}
		data = append(data, resp.Data)
	}
	return data, nil
}

// CreateConditionalUSDTFuturesOrder sends a new conditional USDT futures order to the exchange
func (by *Bybit) CreateConditionalUSDTFuturesOrder(ctx context.Context, symbol currency.Pair, side, orderType, timeInForce,
	orderLinkID, takeProfitTriggerBy, stopLossTriggerBy, triggerBy string,
	quantity, price, takeProfit, stopLoss, basePrice, stopPrice float64, closeOnTrigger, reduceOnly bool) (USDTFuturesConditionalOrderResp, error) {
	resp := struct {
		Data USDTFuturesConditionalOrderResp `json:"result"`
		Error
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	params.Set("side", side)
	params.Set("order_type", orderType)
	if quantity <= 0 {
		return resp.Data, errInvalidQuantity
	}
	params.Set("qty", strconv.FormatFloat(quantity, 'f', -1, 64))

	if price != 0 {
		params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	}
	if basePrice <= 0 {
		return resp.Data, errInvalidBasePrice
	}
	params.Set("base_price", strconv.FormatFloat(basePrice, 'f', -1, 64))

	if stopPrice <= 0 {
		return resp.Data, errInvalidStopPrice
	}
	params.Set("stop_px", strconv.FormatFloat(stopPrice, 'f', -1, 64))

	if timeInForce == "" {
		return resp.Data, errInvalidTimeInForce
	}
	params.Set("time_in_force", timeInForce)

	if triggerBy != "" {
		params.Set("trigger_by", triggerBy)
	}
	if closeOnTrigger {
		params.Set("close_on_trigger", "true")
	} else {
		params.Set("close_on_trigger", "false")
	}
	if reduceOnly {
		params.Set("reduce_only", "true")
	} else {
		params.Set("reduce_only", "false")
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
	return resp.Data, by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesCreateConditionalOrder, params, nil, &resp, uFuturesCreateConditionalOrderRate)
}

// GetConditionalUSDTFuturesOrders gets list of USDT futures conditional orders
func (by *Bybit) GetConditionalUSDTFuturesOrders(ctx context.Context, symbol currency.Pair, stopOrderStatus, direction, stopOrderID, orderLinkID string, limit, page int64) ([]USDTFuturesConditionalOrders, error) {
	resp := struct {
		Result struct {
			Data        []USDTFuturesConditionalOrders `json:"data"`
			CurrentPage int64                          `json:"current_page"`
			LastPage    int64                          `json:"last_page"`
		} `json:"result"`
		Error
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Result.Data, err
	}
	params.Set("symbol", symbolValue)
	if stopOrderStatus != "" {
		params.Set("stop_order_status", stopOrderStatus)
	}
	if direction != "" {
		params.Set("order", direction)
	}
	if limit > 0 && limit <= 50 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if page != 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if stopOrderID != "" {
		params.Set("stop_order_id", stopOrderID)
	}
	if orderLinkID != "" {
		params.Set("order_link_id", orderLinkID)
	}
	return resp.Result.Data, by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesGetConditionalOrders, params, nil, &resp, uFuturesGetConditionalOrderRate)
}

// CancelConditionalUSDTFuturesOrders cancels untriggered conditional orders
func (by *Bybit) CancelConditionalUSDTFuturesOrders(ctx context.Context, symbol currency.Pair, stopOrderID, orderLinkID string) (string, error) {
	resp := struct {
		Result struct {
			StopOrderID string `json:"stop_order_id"`
		} `json:"result"`
		Error
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
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
	return resp.Result.StopOrderID, by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesCancelConditionalOrder, params, nil, &resp, uFuturesCancelConditionalOrderRate)
}

// CancelAllConditionalUSDTFuturesOrders cancels all untriggered conditional orders
func (by *Bybit) CancelAllConditionalUSDTFuturesOrders(ctx context.Context, symbol currency.Pair) ([]string, error) {
	resp := struct {
		Data []string `json:"result"`
		Error
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	return resp.Data, by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesCancelAllConditionalOrders, params, nil, &resp, uFuturesCancelAllConditionalOrderRate)
}

// ReplaceConditionalUSDTFuturesOrders modify unfilled or partially filled conditional orders
func (by *Bybit) ReplaceConditionalUSDTFuturesOrders(ctx context.Context, symbol currency.Pair, stopOrderID, orderLinkID, takeProfitTriggerBy, stopLossTriggerBy string,
	updatedQty, updatedPrice, takeProfitPrice, stopLossPrice, orderTriggerPrice float64) (string, error) {
	resp := struct {
		Data struct {
			OrderID string `json:"stop_order_id"`
		} `json:"result"`
		Error
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
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
	return resp.Data.OrderID, by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesReplaceConditionalOrder, params, nil, &resp, uFuturesDefaultRate)
}

// GetConditionalUSDTRealtimeOrders query real time conditional order data
func (by *Bybit) GetConditionalUSDTRealtimeOrders(ctx context.Context, symbol currency.Pair, stopOrderID, orderLinkID string) ([]USDTFuturesConditionalRealtimeOrder, error) {
	var data []USDTFuturesConditionalRealtimeOrder
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
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
			Result []USDTFuturesConditionalRealtimeOrder `json:"result"`
			Error
		}{}
		err = by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesGetConditionalRealtimeOrders, params, nil, &resp, uFuturesGetConditionalRealtimeOrderRate)
		if err != nil {
			return data, err
		}
		data = append(data, resp.Result...)
	} else {
		resp := struct {
			Result USDTFuturesConditionalRealtimeOrder `json:"result"`
			Error
		}{}
		err = by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesGetConditionalRealtimeOrders, params, nil, &resp, uFuturesGetConditionalRealtimeOrderRate)
		if err != nil {
			return data, err
		}
		data = append(data, resp.Result)
	}
	return data, nil
}

// GetUSDTPositions returns list of user positions
func (by *Bybit) GetUSDTPositions(ctx context.Context, symbol currency.Pair) ([]USDTPositionResp, error) {
	var data []USDTPositionResp
	params := url.Values{}

	if !symbol.IsEmpty() {
		resp := struct {
			Result []USDTPositionResp `json:"result"`
			Error
		}{}

		symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return data, err
		}
		params.Set("symbol", symbolValue)

		err = by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesPosition, params, nil, &resp, uFuturesPositionRate)
		if err != nil {
			return data, err
		}
		data = resp.Result
	} else {
		resp := struct {
			Result []struct {
				IsValid bool             `json:"is_valid"`
				Data    USDTPositionResp `json:"data"`
			} `json:"result"`
			Error
		}{}
		err := by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesPosition, params, nil, &resp, uFuturesPositionRate)
		if err != nil {
			return data, err
		}
		for x := range resp.Result {
			data = append(data, resp.Result[x].Data)
		}
	}
	return data, nil
}

// SetAutoAddMargin sets auto add margin
func (by *Bybit) SetAutoAddMargin(ctx context.Context, symbol currency.Pair, autoAddMargin bool, side string) error {
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	params.Set("symbol", symbolValue)
	if side == "" {
		return errInvalidSide
	}
	params.Set("side", side)

	if autoAddMargin {
		params.Set("take_profit", "true")
	} else {
		params.Set("take_profit", "false")
	}
	return by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesSetAutoAddMargin, params, nil, nil, uFuturesSetMarginRate)
}

// ChangeUSDTMargin switches margin between cross or isolated
func (by *Bybit) ChangeUSDTMargin(ctx context.Context, symbol currency.Pair, buyLeverage, sellLeverage float64, isIsolated bool) error {
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
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

	return by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesSwitchMargin, params, nil, nil, uFuturesSwitchMargin)
}

// SwitchPositionMode switches mode between MergedSingle: One-Way Mode or BothSide: Hedge Mode
func (by *Bybit) SwitchPositionMode(ctx context.Context, symbol currency.Pair, mode string) error {
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	params.Set("symbol", symbolValue)
	if mode == "" {
		return errInvalidMode
	}
	params.Set("mode", mode)

	return by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesSwitchPositionMode, params, nil, nil, uFuturesSwitchPosition)
}

// ChangeUSDTMode switches mode between full or partial position
func (by *Bybit) ChangeUSDTMode(ctx context.Context, symbol currency.Pair, takeProfitStopLoss string) (string, error) {
	resp := struct {
		Result struct {
			Mode string `json:"tp_sl_mode"`
		} `json:"result"`
		Error
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Result.Mode, err
	}
	params.Set("symbol", symbolValue)
	if takeProfitStopLoss == "" {
		return resp.Result.Mode, errInvalidTakeProfitStopLoss
	}
	params.Set("tp_sl_mode", takeProfitStopLoss)

	return resp.Result.Mode, by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesSwitchPosition, params, nil, &resp, uFuturesSwitchPosition)
}

// SetUSDTMargin updates margin
func (by *Bybit) SetUSDTMargin(ctx context.Context, symbol currency.Pair, side, margin string) (UpdateMarginResp, error) {
	resp := struct {
		Result struct {
			Data             UpdateMarginResp
			WalletBalance    float64
			AvailableBalance float64
		} `json:"result"`
		Error
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Result.Data, err
	}
	params.Set("symbol", symbolValue)
	if side == "" {
		return resp.Result.Data, errInvalidSide
	}
	params.Set("side", side)

	if margin == "" {
		return resp.Result.Data, errInvalidMargin
	}
	params.Set("margin", margin)

	return resp.Result.Data, by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesUpdateMargin, params, nil, &resp, uFuturesUpdateMarginRate)
}

// SetUSDTLeverage sets leverage
func (by *Bybit) SetUSDTLeverage(ctx context.Context, symbol currency.Pair, buyLeverage, sellLeverage float64) error {
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	params.Set("symbol", symbolValue)
	if buyLeverage <= 0 {
		return errInvalidBuyLeverage
	}
	params.Set("buy_leverage", strconv.FormatFloat(buyLeverage, 'f', -1, 64))

	if sellLeverage <= 0 {
		return errInvalidSellLeverage
	}
	params.Set("sell_leverage", strconv.FormatFloat(sellLeverage, 'f', -1, 64))

	return by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesSetLeverage, params, nil, nil, uFuturesSetLeverageRate)
}

// SetUSDTTradingAndStop sets take profit, stop loss, and trailing stop for your open position
func (by *Bybit) SetUSDTTradingAndStop(ctx context.Context, symbol currency.Pair, takeProfit, stopLoss, trailingStop, stopLossQty, takeProfitQty float64, side, takeProfitTriggerBy, stopLossTriggerBy string) error {
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	params.Set("symbol", symbolValue)
	if side == "" {
		return errInvalidSide
	}
	params.Set("side", side)

	if takeProfit >= 0 {
		params.Set("take_profit", strconv.FormatFloat(takeProfit, 'f', -1, 64))
	}
	if stopLoss >= 0 {
		params.Set("stop_loss", strconv.FormatFloat(stopLoss, 'f', -1, 64))
	}
	if trailingStop >= 0 {
		params.Set("trailing_stop", strconv.FormatFloat(trailingStop, 'f', -1, 64))
	}
	if takeProfitQty != 0 {
		params.Set("tp_size", strconv.FormatFloat(takeProfitQty, 'f', -1, 64))
	}
	if stopLossQty != 0 {
		params.Set("sl_size", strconv.FormatFloat(stopLossQty, 'f', -1, 64))
	}
	if takeProfitTriggerBy != "" {
		params.Set("tp_trigger_by", takeProfitTriggerBy)
	}
	if stopLossTriggerBy != "" {
		params.Set("sl_trigger_by", stopLossTriggerBy)
	}

	return by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesSetTradingStop, params, nil, nil, uFuturesSetTradingStopRate)
}

// GetUSDTTradeRecords returns list of user trades
func (by *Bybit) GetUSDTTradeRecords(ctx context.Context, symbol currency.Pair, executionType string, startTime, endTime, page, limit int64) ([]TradeData, error) {
	params := url.Values{}
	resp := struct {
		Data struct {
			CurrentPage int64       `json:"current_page"`
			Trades      []TradeData `json:"data"`
		} `json:"result"`
		Error
	}{}

	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Data.Trades, err
	}
	params.Set("symbol", symbolValue)
	if executionType != "" {
		params.Set("exec_type", executionType)
	}
	if startTime != 0 {
		params.Set("start_time", strconv.FormatInt(startTime, 10))
	}
	if endTime != 0 {
		params.Set("end_time", strconv.FormatInt(endTime, 10))
	}
	if page != 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp.Data.Trades, by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesGetTrades, params, nil, &resp, uFuturesGetTradesRate)
}

// GetClosedUSDTTrades returns closed profit and loss records
func (by *Bybit) GetClosedUSDTTrades(ctx context.Context, symbol currency.Pair, executionType string, startTime, endTime time.Time, page, limit int64) ([]ClosedTrades, error) {
	params := url.Values{}

	resp := struct {
		Data struct {
			CurrentPage int64          `json:"current_page"`
			Trades      []ClosedTrades `json:"data"`
		} `json:"result"`
		Error
	}{}

	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Data.Trades, err
	}
	params.Set("symbol", symbolValue)
	if executionType != "" {
		params.Set("execution_type", executionType)
	}
	if !startTime.IsZero() {
		params.Set("start_time", strconv.FormatInt(startTime.Unix(), 10))
	}
	if !endTime.IsZero() {
		params.Set("end_time", strconv.FormatInt(endTime.Unix(), 10))
	}
	if page > 0 && page <= 50 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 && limit <= 50 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp.Data.Trades, by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesGetClosedTrades, params, nil, &resp, uFuturesGetClosedTradesRate)
}

// SetUSDTRiskLimit sets risk limit
func (by *Bybit) SetUSDTRiskLimit(ctx context.Context, symbol currency.Pair, side string, riskID int64) (int64, error) {
	resp := struct {
		Result struct {
			RiskID int64 `json:"risk_id"`
		} `json:"result"`
		Error
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Result.RiskID, err
	}
	params.Set("symbol", symbolValue)
	if side == "" {
		return 0, errInvalidSide
	}
	params.Set("side", side)

	if riskID <= 0 {
		return resp.Result.RiskID, errInvalidRiskID
	}
	params.Set("risk_id", strconv.FormatInt(riskID, 10))

	return resp.Result.RiskID, by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodPost, ufuturesSetRiskLimit, params, nil, &resp, uFuturesDefaultRate)
}

// GetPredictedUSDTFundingRate returns predicted funding rates and fees
func (by *Bybit) GetPredictedUSDTFundingRate(ctx context.Context, symbol currency.Pair) (fundingRate, fundingFee float64, err error) {
	params := url.Values{}
	resp := struct {
		Result struct {
			PredictedFundingRate float64 `json:"predicted_funding_rate"`
			PredictedFundingFee  float64 `json:"predicted_funding_fee"`
		} `json:"result"`
		Error
	}{}

	var symbolValue string
	symbolValue, err = by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Result.PredictedFundingRate, resp.Result.PredictedFundingFee, err
	}
	params.Set("symbol", symbolValue)

	err = by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesPredictFundingRate, params, nil, &resp, uFuturesPredictFundingRate)
	fundingRate = resp.Result.PredictedFundingRate
	fundingFee = resp.Result.PredictedFundingFee
	return
}

// GetLastUSDTFundingFee returns last funding fees
func (by *Bybit) GetLastUSDTFundingFee(ctx context.Context, symbol currency.Pair) (FundingFee, error) {
	params := url.Values{}
	resp := struct {
		Result FundingFee `json:"result"`
		Error
	}{}

	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Result, err
	}
	params.Set("symbol", symbolValue)

	return resp.Result, by.SendAuthHTTPRequest(ctx, exchange.RestUSDTMargined, http.MethodGet, ufuturesGetMyLastFundingFee, params, nil, &resp, uFuturesGetMyLastFundingFeeRate)
}
