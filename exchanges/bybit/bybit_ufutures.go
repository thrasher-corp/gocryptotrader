package bybit

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// TODO: handle rate limiting
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

	ufuturesPosition         = "/private/linear/position/list"
	ufuturesSetAutoAddMargin = "/private/linear/position/set-auto-add-margin"
	ufuturesSwitchMargin     = "/private/linear/position/switch-isolated"
	ufuturesSwitchPosition   = "/private/linear/tpsl/switch-mode"
	ufuturesUpdateMargin     = "/private/linear/position/add-margin"
	ufuturesSetLeverage      = "/private/linear/position/set-leverage"
	ufuturesSetTradingStop   = "/private/linear/position/trading-stop"
	ufuturesGetTrades        = "/private/linear/trade/execution/list"
	ufuturesGetClosedTrades  = "/private/linear/trade/closed-pnl/list"

	ufuturesSetRiskLimit        = "/private/linear/position/set-risk"
	ufuturesPredictFundingRate  = "/private/linear/funding/predicted-funding"
	ufuturesGetMyLastFundingFee = "/private/linear/funding/prev-funding"
)

// GetUSDTFuturesKlineData gets futures kline data for USDTMarginedFutures.
func (by *Bybit) GetUSDTFuturesKlineData(symbol currency.Pair, interval string, limit int64, startTime time.Time) ([]FuturesCandleStick, error) {
	resp := struct {
		Data []FuturesCandleStick `json:"result"`
	}{}

	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return resp.Data, err
		}
		params.Set("symbol", symbolValue)
	} else {
		return resp.Data, errors.New("symbol missing")
	}

	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesIntervals, interval) {
		return resp.Data, errors.New("invalid interval parsed")
	}
	params.Set("interval", interval)
	params.Set("from", strconv.FormatInt(startTime.Unix(), 10))

	path := common.EncodeURLValues(ufuturesKline, params)
	return resp.Data, by.SendHTTPRequest(exchange.RestUSDTMargined, path, publicFuturesRate, &resp)
}

// GetUSDTPublicTrades gets past public trades for USDTMarginedFutures.
func (by *Bybit) GetUSDTPublicTrades(symbol currency.Pair, limit int64) ([]FuturesPublicTradesData, error) {
	resp := struct {
		Data []FuturesPublicTradesData `json:"result"`
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
	return resp.Data, by.SendHTTPRequest(exchange.RestUSDTMargined, path, publicFuturesRate, &resp)
}

// GetMarkPriceKline gets mark price kline data for USDTMarginedFutures.
func (by *Bybit) GetUSDTMarkPriceKline(symbol currency.Pair, interval string, limit int64, startTime time.Time) ([]MarkPriceKlineData, error) {
	resp := struct {
		Data []MarkPriceKlineData `json:"result"`
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
		return resp.Data, errors.New("invalid interval parsed")
	}
	params.Set("interval", interval)
	if !startTime.IsZero() {
		params.Set("from", strconv.FormatInt(startTime.Unix(), 10))
	} else {
		return resp.Data, errors.New("startTime can't be zero or missing")
	}

	path := common.EncodeURLValues(ufuturesMarkPriceKline, params)
	return resp.Data, by.SendHTTPRequest(exchange.RestUSDTMargined, path, publicFuturesRate, &resp)
}

// GetUSDTIndexPriceKline gets index price kline data for USDTMarginedFutures.
func (by *Bybit) GetUSDTIndexPriceKline(symbol currency.Pair, interval string, limit int64, startTime time.Time) ([]IndexPriceKlineData, error) {
	resp := struct {
		Data []IndexPriceKlineData `json:"result"`
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
		return resp.Data, errors.New("invalid interval parsed")
	}
	params.Set("interval", interval)
	if !startTime.IsZero() {
		params.Set("from", strconv.FormatInt(startTime.Unix(), 10))
	} else {
		return resp.Data, errors.New("startTime can't be zero or missing")
	}

	path := common.EncodeURLValues(ufuturesIndexKline, params)
	return resp.Data, by.SendHTTPRequest(exchange.RestUSDTMargined, path, publicFuturesRate, &resp)
}

// GetUSDTPremiumIndexPriceKline gets premium index price kline data for USDTMarginedFutures.
func (by *Bybit) GetUSDTPremiumIndexPriceKline(symbol currency.Pair, interval string, limit int64, startTime time.Time) ([]IndexPriceKlineData, error) {
	resp := struct {
		Data []IndexPriceKlineData `json:"result"`
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
		return resp.Data, errors.New("invalid interval parsed")
	}
	params.Set("interval", interval)
	if !startTime.IsZero() {
		params.Set("from", strconv.FormatInt(startTime.Unix(), 10))
	} else {
		return resp.Data, errors.New("startTime can't be zero or missing")
	}

	path := common.EncodeURLValues(ufuturesIndexPremiumKline, params)
	return resp.Data, by.SendHTTPRequest(exchange.RestUSDTMargined, path, publicFuturesRate, &resp)
}

// GetUSDTLastFundingRate returns latest generated funding fee
func (by *Bybit) GetUSDTLastFundingRate(symbol currency.Pair) (USDTFundingInfo, error) {
	resp := struct {
		Data USDTFundingInfo `json:"result"`
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)

	path := common.EncodeURLValues(ufuturesGetLastFundingRate, params)
	return resp.Data, by.SendHTTPRequest(exchange.RestUSDTMargined, path, publicFuturesRate, &resp)
}

// GetUSDTRiskLimit returns risk limit
func (by *Bybit) GetUSDTRiskLimit(symbol currency.Pair) ([]RiskInfo, error) {
	resp := struct {
		Data []RiskInfo `json:"result"`
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
	return resp.Data, by.SendHTTPRequest(exchange.RestUSDTMargined, path, publicFuturesRate, &resp)
}

// CreateUSDTFuturesOrder sends a new USDT futures order to the exchange
func (by *Bybit) CreateUSDTFuturesOrder(symbol currency.Pair, side, orderType, timeInForce,
	orderLinkID, takeProfitTriggerBy, stopLossTriggerBy string,
	quantity, price, takeProfit, stopLoss float64, closeOnTrigger, reduceOnly bool) (FuturesOrderDataResp, error) {
	resp := struct {
		Data FuturesOrderDataResp `json:"result"`
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	if side != "" {
		params.Set("side", side)
	} else {
		return resp.Data, errors.New("side can't be empty or missing")
	}
	if orderType != "" {
		params.Set("order_type", orderType)
	} else {
		return resp.Data, errors.New("orderType can't be empty or missing")
	}
	if quantity != 0 {
		params.Set("qty", strconv.FormatFloat(quantity, 'f', -1, 64))
	} else {
		return resp.Data, errors.New("quantity can't be zero or missing")
	}
	if price != 0 {
		params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	}
	if timeInForce != "" {
		params.Set("time_in_force", timeInForce)
	} else {
		return resp.Data, errors.New("timeInForce can't be empty or missing")
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
	return resp.Data, by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodPost, ufuturesCreateOrder, params, &resp, uFuturesCreateOrderRate)
}

// GetActiveUSDTFuturesOrders gets list of USDT futures active orders
func (by *Bybit) GetActiveUSDTFuturesOrders(symbol currency.Pair, orderStatus, direction, orderID, orderLinkID string, page, limit int64) ([]FuturesActiveOrderResp, error) {
	resp := struct {
		Result struct {
			Data        []FuturesActiveOrderResp `json:"data"`
			CurrentPage int64                    `json:"current_page"`
			LastPage    int64                    `json:"last_page"`
		} `json:"result"`
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

	return resp.Result.Data, by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodGet, ufuturesGetActiveOrders, params, &resp, uFuturesGetActiveOrderRate)
}

// CancelActiveUSDTFuturesOrders cancels USDT futures unfilled or partially filled orders
func (by *Bybit) CancelActiveUSDTFuturesOrders(symbol currency.Pair, orderID, orderLinkID string) (string, error) {
	resp := struct {
		Data struct {
			OrderID string `json:"order_id"`
		} `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Data.OrderID, err
	}
	params.Set("symbol", symbolValue)
	if orderID == "" && orderLinkID == "" {
		return resp.Data.OrderID, errors.New("one among orderID or orderLinkID should be present")
	}
	if orderID != "" {
		params.Set("order_id", orderID)
	}
	if orderLinkID != "" {
		params.Set("order_link_id", orderLinkID)
	}
	return resp.Data.OrderID, by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodPost, ufuturesCancelActiveOrder, params, &resp, uFuturesCancelOrderRate)
}

// CancelAllActiveUSDTFuturesOrders cancels all USDT futures unfilled or partially filled orders
func (by *Bybit) CancelAllActiveUSDTFuturesOrders(symbol currency.Pair) ([]string, error) {
	resp := struct {
		Data []string `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	return resp.Data, by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodPost, ufuturesCancelAllActiveOrders, params, &resp, uFuturesCancelAllOrderRate)
}

// ReplaceActiveUSDTFuturesOrders modify unfilled or partially filled orders
func (by *Bybit) ReplaceActiveUSDTFuturesOrders(symbol currency.Pair, orderID, orderLinkID, takeProfitTriggerBy, stopLossTriggerBy string,
	updatedQty int64, updatedPrice, takeProfitPrice, stopLossPrice float64) (string, error) {
	resp := struct {
		Data struct {
			OrderID string `json:"order_id"`
		} `json:"result"`
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
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
	return resp.Data.OrderID, by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodPost, ufuturesReplaceActiveOrder, params, &resp, uFuturesDefaultRate)
}

// GetActiveUSDTRealtimeOrders query real time order data
func (by *Bybit) GetActiveUSDTRealtimeOrders(symbol currency.Pair, orderID, orderLinkID string) ([]FuturesActiveRealtimeOrder, error) {
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
		}{}
		err = by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodGet, ufuturesGetActiveRealtimeOrders, params, &resp, uFuturesGetActiveRealtimeOrderRate)
		if err != nil {
			return data, err
		}
		for _, d := range resp.Data {
			data = append(data, d)
		}
	} else {
		resp := struct {
			Data FuturesActiveRealtimeOrder `json:"result"`
		}{}
		err = by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodGet, ufuturesGetActiveRealtimeOrders, params, &resp, uFuturesGetActiveRealtimeOrderRate)
		if err != nil {
			return data, err
		}
		data = append(data, resp.Data)
	}
	return data, nil
}

// CreateConditionalUSDTFuturesOrder sends a new conditional USDT futures order to the exchange
func (by *Bybit) CreateConditionalUSDTFuturesOrder(symbol currency.Pair, side, orderType, timeInForce,
	orderLinkID, takeProfitTriggerBy, stopLossTriggerBy, triggerBy string,
	quantity, price, takeProfit, stopLoss, basePrice, stopPrice float64, closeOnTrigger, reduceOnly bool) (USDTFuturesConditionalOrderResp, error) {
	resp := struct {
		Data USDTFuturesConditionalOrderResp `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	params.Set("side", side)
	params.Set("order_type", orderType)
	if quantity != 0 {
		params.Set("qty", strconv.FormatFloat(quantity, 'f', -1, 64))
	} else {
		return resp.Data, errors.New("quantity can't be zero or missing")
	}
	if price != 0 {
		params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	}
	if basePrice != 0 {
		params.Set("base_price", strconv.FormatFloat(basePrice, 'f', -1, 64))
	} else {
		return resp.Data, errors.New("basePrice can't be empty or missing")
	}
	if stopPrice != 0 {
		params.Set("stop_px", strconv.FormatFloat(stopPrice, 'f', -1, 64))
	} else {
		return resp.Data, errors.New("stopPrice can't be empty or missing")
	}
	if timeInForce != "" {
		params.Set("time_in_force", timeInForce)
	} else {
		return resp.Data, errors.New("timeInForce can't be empty or missing")
	}
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
	return resp.Data, by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodPost, ufuturesCreateConditionalOrder, params, &resp, uFuturesCreateConditionalOrderRate)
}

// GetConditionalUSDTFuturesOrders gets list of USDT futures conditional orders
func (by *Bybit) GetConditionalUSDTFuturesOrders(symbol currency.Pair, stopOrderStatus, direction, stopOrderID, orderLinkID string, limit, page int64) ([]USDTFuturesConditionalOrders, error) {
	resp := struct {
		Result struct {
			Data        []USDTFuturesConditionalOrders `json:"data"`
			CurrentPage int64                          `json:"current_page"`
			LastPage    int64                          `json:"last_page"`
		} `json:"result"`
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
	return resp.Result.Data, by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodGet, ufuturesGetConditionalOrders, params, &resp, uFuturesGetCondtionalOrderRate)
}

// CancelConditionalUSDTFuturesOrders cancels untriggered conditional orders
func (by *Bybit) CancelConditionalUSDTFuturesOrders(symbol currency.Pair, stopOrderID, orderLinkID string) (string, error) {
	resp := struct {
		Result struct {
			StopOrderID string `json:"stop_order_id"`
		} `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
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
	return resp.Result.StopOrderID, by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodPost, ufuturesCancelConditionalOrder, params, &resp, uFuturesCancelConditionalOrderRate)
}

// CancelAllConditionalUSDTFuturesOrders cancels all untriggered conditional orders
func (by *Bybit) CancelAllConditionalUSDTFuturesOrders(symbol currency.Pair) ([]string, error) {
	resp := struct {
		Data []string `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	return resp.Data, by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodPost, ufuturesCancelAllConditionalOrders, params, &resp, uFuturesCancelAllConditionalOrderRate)
}

// ReplaceConditionalUSDTFuturesOrders modify unfilled or partially filled conditional orders
func (by *Bybit) ReplaceConditionalUSDTFuturesOrders(symbol currency.Pair, stopOrderID, orderLinkID, takeProfitTriggerBy, stopLossTriggerBy string,
	updatedQty, updatedPrice, takeProfitPrice, stopLossPrice, orderTriggerPrice float64) (string, error) {
	resp := struct {
		Data struct {
			OrderID string `json:"stop_order_id"`
		} `json:"result"`
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
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
	return resp.Data.OrderID, by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodPost, ufuturesReplaceConditionalOrder, params, &resp, uFuturesDefaultRate)
}

// GetConditionalUSDTRealtimeOrders query real time conditional order data
func (by *Bybit) GetConditionalUSDTRealtimeOrders(symbol currency.Pair, stopOrderID, orderLinkID string) ([]USDTFuturesConditionalRealtimeOrder, error) {
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
		}{}
		err = by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodGet, ufuturesGetConditionalRealtimeOrders, params, &resp, uFuturesGetCondtionalRealtimeOrderRate)
		if err != nil {
			return data, err
		}
		for _, d := range resp.Result {
			data = append(data, d)
		}
	} else {
		resp := struct {
			Result USDTFuturesConditionalRealtimeOrder `json:"result"`
		}{}
		err = by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodGet, ufuturesGetConditionalRealtimeOrders, params, &resp, uFuturesGetCondtionalRealtimeOrderRate)
		if err != nil {
			return data, err
		}
		data = append(data, resp.Result)
	}
	return data, nil
}

// GetUSDTPositions returns list of user positions
func (by *Bybit) GetUSDTPositions(symbol currency.Pair) ([]USDTPositionResp, error) {
	var data []USDTPositionResp
	params := url.Values{}

	if !symbol.IsEmpty() {
		resp := struct {
			Result []USDTPositionResp `json:"result"`
		}{}

		symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
		if err != nil {
			return data, err
		}
		params.Set("symbol", symbolValue)

		err = by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodGet, ufuturesPosition, params, &resp, uFuturesPositionRate)
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
		}{}
		err := by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodGet, ufuturesPosition, params, &resp, uFuturesPositionRate)
		if err != nil {
			return data, err
		}
		for _, d := range resp.Result {
			data = append(data, d.Data)
		}
	}
	return data, nil
}

// SetAutoAddMargin sets auto add margin
func (by *Bybit) SetAutoAddMargin(symbol currency.Pair, autoAddMargin bool, side string) error {
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	params.Set("symbol", symbolValue)
	if side != "" {
		params.Set("side", side)
	} else {
		return errors.New("side can't be empty or missing")
	}
	if autoAddMargin {
		params.Set("take_profit", "true")
	} else {
		params.Set("take_profit", "false")
	}
	return by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodPost, ufuturesSetAutoAddMargin, params, &struct{}{}, uFuturesSetMarginRate)
}

// ChangeUSDTMargin switches margin between cross or isolated
func (by *Bybit) ChangeUSDTMargin(symbol currency.Pair, buyLeverage, sellLeverage float64, isIsolated bool) error {
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

	return by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodPost, ufuturesSwitchMargin, params, &struct{}{}, uFuturesSwitchMargin)
}

// ChangeUSDTMode switches mode between full or partial position
func (by *Bybit) ChangeUSDTMode(symbol currency.Pair, takeProfitStopLoss string) (string, error) {
	resp := struct {
		Result struct {
			Mode string `json:"tp_sl_mode"`
		} `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Result.Mode, err
	}
	params.Set("symbol", symbolValue)
	if takeProfitStopLoss != "" {
		params.Set("tp_sl_mode", takeProfitStopLoss)
	} else {
		return resp.Result.Mode, errors.New("takeProfitStopLoss can't be empty or missing")
	}

	return resp.Result.Mode, by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodPost, ufuturesSwitchPosition, params, &resp, uFuturesSwitchPosition)
}

// SetUSDTMargin updates margin
func (by *Bybit) SetUSDTMargin(symbol currency.Pair, side, margin string) (UpdateMarginResp, error) {
	resp := struct {
		Result struct {
			Data             UpdateMarginResp
			WalletBalance    float64
			AvailableBalance float64
		} `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Result.Data, err
	}
	params.Set("symbol", symbolValue)
	if side != "" {
		params.Set("side", side)
	} else {
		return resp.Result.Data, errors.New("side can't be empty")
	}
	if margin != "" {
		params.Set("margin", margin)
	} else {
		return resp.Result.Data, errors.New("margin can't be empty")
	}
	return resp.Result.Data, by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodPost, ufuturesUpdateMargin, params, &resp, uFuturesUpdateMarginRate)
}

// SetUSDTLeverage sets leverage
func (by *Bybit) SetUSDTLeverage(symbol currency.Pair, buyLeverage, sellLeverage float64) error {
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	params.Set("symbol", symbolValue)
	if buyLeverage > 0 {
		params.Set("buy_leverage", strconv.FormatFloat(buyLeverage, 'f', -1, 64))
	} else {
		return errors.New("buyLeverage can't be zero or less then it")
	}
	if sellLeverage > 0 {
		params.Set("sell_leverage", strconv.FormatFloat(sellLeverage, 'f', -1, 64))
	} else {
		return errors.New("sellLeverage can't be zero or less then it")
	}

	return by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodPost, ufuturesSetLeverage, params, &struct{}{}, uFuturesSetLeverageRate)
}

// SetUSDTTradingAndStop sets take profit, stop loss, and trailing stop for your open position
func (by *Bybit) SetUSDTTradingAndStop(symbol currency.Pair, takeProfit, stopLoss, trailingStop, stopLossQty, takeProfitQty float64, side, takeProfitTriggerBy, stopLossTriggerBy string) error {
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	params.Set("symbol", symbolValue)
	if side != "" {
		params.Set("side", side)
	} else {
		return errors.New("side can't be empty")
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

	return by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodPost, ufuturesSetTradingStop, params, &struct{}{}, uFuturesSetTradingStopRate)
}

// GetUSDTTradeRecords returns list of user trades
func (by *Bybit) GetUSDTTradeRecords(symbol currency.Pair, executionType string, startTime, endTime, page, limit int64) ([]TradeData, error) {
	params := url.Values{}
	resp := struct {
		Data struct {
			CurrentPage int64       `json:"current_page"`
			Trades      []TradeData `json:"data"`
		} `json:"result"`
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
	return resp.Data.Trades, by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodGet, ufuturesGetTrades, params, &resp, uFuturesGetTradesRate)
}

// GetClosedUSDTTrades returns closed profit and loss records
func (by *Bybit) GetClosedUSDTTrades(symbol currency.Pair, executionType string, startTime, endTime, page, limit int64) ([]ClosedTrades, error) {
	params := url.Values{}

	resp := struct {
		Data struct {
			CurrentPage int64          `json:"current_page"`
			Trades      []ClosedTrades `json:"data"`
		} `json:"result"`
	}{}

	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
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
	return resp.Data.Trades, by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodGet, ufuturesGetClosedTrades, params, &resp, uFuturesGetClosedTradesRate)
}

// SetUSDTRiskLimit sets risk limit
func (by *Bybit) SetUSDTRiskLimit(symbol currency.Pair, side string, riskID int64) (int64, error) {
	resp := struct {
		Result struct {
			RiskID int64 `json:"risk_id"`
		} `json:"result"`
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Result.RiskID, err
	}
	params.Set("symbol", symbolValue)
	if side != "" {
		params.Set("side", side)
	} else {
		return 0, errors.New("side can't be empty")
	}
	if riskID > 0 {
		params.Set("risk_id", strconv.FormatInt(riskID, 10))
	} else {
		return resp.Result.RiskID, errors.New("riskID can't be zero or lesser")
	}

	return resp.Result.RiskID, by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodPost, ufuturesSetRiskLimit, params, &resp, uFuturesDefaultRate)
}

// GetPredictedUSDTFundingRate returns predicted funding rates and fees
func (by *Bybit) GetPredictedUSDTFundingRate(symbol currency.Pair) (float64, float64, error) {
	params := url.Values{}
	resp := struct {
		Result struct {
			PredictedFundingRate float64 `json:"predicted_funding_rate"`
			PredictedFundingFee  float64 `json:"predicted_funding_fee"`
		} `json:"result"`
	}{}

	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Result.PredictedFundingRate, resp.Result.PredictedFundingFee, err
	}
	params.Set("symbol", symbolValue)

	return resp.Result.PredictedFundingRate, resp.Result.PredictedFundingFee, by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodGet, ufuturesPredictFundingRate, params, &resp, uFuturesPredictFundingRate)
}

// GetLastUSDTFundingFee returns last funding fees
func (by *Bybit) GetLastUSDTFundingFee(symbol currency.Pair) (FundingFee, error) {
	params := url.Values{}
	resp := struct {
		Result FundingFee `json:"result"`
	}{}

	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Result, err
	}
	params.Set("symbol", symbolValue)

	return resp.Result, by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodGet, ufuturesGetMyLastFundingFee, params, &resp, uFuturesGetMyLastFundingFeeRate)
}
