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
	ufuturesMarginSwitch     = "/private/linear/position/switch-isolated"
	ufuturesPositionSwitch   = "/private/linear/tpsl/switch-mode"
	ufuturesAddMargin        = "/private/linear/position/add-margin"
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
	err := by.SendHTTPRequest(exchange.RestUSDTMargined, path, &resp)
	if err != nil {
		return resp.Data, err
	}

	return resp.Data, nil
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
	return resp.Data, by.SendHTTPRequest(exchange.RestUSDTMargined, path, &resp)
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
	return resp.Data, by.SendHTTPRequest(exchange.RestUSDTMargined, path, &resp)
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
	return resp.Data, by.SendHTTPRequest(exchange.RestUSDTMargined, path, &resp)
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
	return resp.Data, by.SendHTTPRequest(exchange.RestUSDTMargined, path, &resp)
}

// GetUSDTLastFundingRate returns latest generated funding fee
func (by *Bybit) GetUSDTLastFundingRate(symbol currency.Pair) (FundingInfo, error) {
	resp := struct {
		Data FundingInfo `json:"result"`
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)

	path := common.EncodeURLValues(ufuturesGetLastFundingRate, params)
	return resp.Data, by.SendHTTPRequest(exchange.RestUSDTMargined, path, &resp)
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
	return resp.Data, by.SendHTTPRequest(exchange.RestUSDTMargined, path, &resp)
}

// CreateUSDTFuturesOrder sends a new futures order to the exchange
func (by *Bybit) CreateUSDTFuturesOrder(symbol currency.Pair, side, orderType, timeInForce,
	orderLinkID, takeProfitTriggerBy, stopLossTriggerBy string,
	quantity, price, takeProfit, stopLoss float64, closeOnTrigger, reduceOnly bool) (FuturesOrderData, error) {
	resp := struct {
		Data FuturesOrderData `json:"result"`
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
	return resp.Data, by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodPost, ufuturesCreateOrder, params, &resp, bybitAuthRate)
}

// GetUSDTActiveFuturesOrders gets list of futures active orders
func (by *Bybit) GetUSDTActiveFuturesOrders(symbol currency.Pair, orderStatus, direction, orderID, orderLinkID string, page, limit int64) ([]FuturesActiveOrders, error) {
	resp := struct {
		Result struct {
			Data        []FuturesActiveOrders `json:"data"`
			CurrentPage int64                 `json:"current_page"`
			LastPage    int64                 `json:"last_page"`
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

	return resp.Result.Data, by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodGet, ufuturesGetActiveOrders, params, &resp, bybitAuthRate)
}

// CancelUSDTActiveFuturesOrders cancels futures unfilled or partially filled orders
func (by *Bybit) CancelUSDTActiveFuturesOrders(symbol currency.Pair, orderID, orderLinkID string) (string, error) {
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
	return resp.Data.OrderID, by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodPost, ufuturesCancelActiveOrder, params, &resp, bybitAuthRate)
}

// CancelAllUSDTActiveFuturesOrders cancels all futures unfilled or partially filled orders
func (by *Bybit) CancelAllUSDTActiveFuturesOrders(symbol currency.Pair) ([]string, error) {
	resp := struct {
		Data []string `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDTMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	return resp.Data, by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodPost, ufuturesCancelAllActiveOrders, params, &resp, bybitAuthRate)
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
	return resp.Data.OrderID, by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodPost, ufuturesReplaceActiveOrder, params, &resp, bybitAuthRate)
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
		err = by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodGet, ufuturesGetActiveRealtimeOrders, params, &resp, bybitAuthRate)
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
		err = by.SendAuthHTTPRequest(exchange.RestUSDTMargined, http.MethodGet, ufuturesGetActiveRealtimeOrders, params, &resp, bybitAuthRate)
		if err != nil {
			return data, err
		}
		data = append(data, resp.Data)
	}
	return data, nil
}
