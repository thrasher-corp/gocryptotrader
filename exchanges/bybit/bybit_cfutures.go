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
// TODO: club redundant struct in futures type
// TODO: correct response struct in public endpoint
const (
	bybitFuturesAPIVersion = "/v2"

	// public endpoint
	cfuturesOrderbook         = "/public/orderBook/L2"
	cfuturesKline             = "/public/kline/list"
	cfuturesSymbolPriceTicker = "/public/tickers"
	cfuturesRecentTrades      = "/public/trading-records"
	cfuturesSymbolInfo        = "/public/symbols"
	cfuturesLiquidationOrders = "/public/liq-records"
	cfuturesMarkPriceKline    = "/public/mark-price-kline"
	cfuturesIndexKline        = "/public/index-price-kline"
	cfuturesIndexPremiumKline = "/public/premium-index-kline"
	cfuturesOpenInterest      = "/public/open-interest"
	cfuturesBigDeal           = "/public/big-deal"
	cfuturesAccountRatio      = "/public/account-ratio"

	// auth endpoint
	cfuturesCreateOrder             = "/private/order/create"
	cfuturesGetActiveOrders         = "/private/order/list"
	cfuturesCancelActiveOrder       = "/private/order/cancel"
	cfuturesCancelAllActiveOrders   = "/private/order/cancelAll"
	cfuturesReplaceActiveOrder      = "/private/order/replace"
	cfuturesGetActiveRealtimeOrders = "/private/order"

	cfuturesCreateConditionalOrder       = "/private/stop-order/create"
	cfuturesGetConditionalOrders         = "/private/stop-order/list"
	cfuturesCancelConditionalOrder       = "/private/stop-order/cancel"
	cfuturesCancelAllConditionalOrders   = "/private/stop-order/cancelAll"
	cfuturesReplaceConditionalOrder      = "/private/stop-order/replace"
	cfuturesGetConditionalRealtimeOrders = "/private/stop-order"
)

// GetFuturesOrderbook gets orderbook data for CoinMarginedFutures.
func (by *Bybit) GetFuturesOrderbook(symbol currency.Pair) (Orderbook, error) {
	var resp Orderbook
	var data []OrderbookData
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)

	path := common.EncodeURLValues(bybitFuturesAPIVersion+cfuturesOrderbook, params)
	err = by.SendHTTPRequest(exchange.RestCoinMargined, path, &data)
	if err != nil {
		return resp, err
	}

	for _, ob := range data {
		var price, quantity float64
		price, err = strconv.ParseFloat(ob.Price, 64)
		if err != nil {
			return resp, err
		}

		quantity = float64(ob.Size)
		if ob.Side == sideBuy {
			resp.Bids = append(resp.Bids, OrderbookItem{
				Price:  price,
				Amount: quantity,
			})
		} else {
			resp.Asks = append(resp.Asks, OrderbookItem{
				Price:  price,
				Amount: quantity,
			})
		}
	}
	return resp, nil
}

// GetFuturesKlineData gets futures kline data for CoinMarginedFutures.
func (by *Bybit) GetFuturesKlineData(symbol currency.Pair, interval string, limit int64, startTime time.Time) ([]FuturesCandleStick, error) {
	var resp []FuturesCandleStick
	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	} else {
		return resp, errors.New("symbol missing")
	}

	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesIntervals, interval) {
		return resp, errors.New("invalid interval parsed")
	}
	params.Set("interval", interval)

	if !startTime.IsZero() {
		params.Set("from", strconv.FormatInt(startTime.Unix(), 10))
	} else {
		return resp, errors.New("startTime can't be zero or missing")
	}

	path := common.EncodeURLValues(bybitFuturesAPIVersion+cfuturesKline, params)
	err := by.SendHTTPRequest(exchange.RestCoinMargined, path, &resp)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// GetFuturesSymbolPriceTicker gets price ticker for symbol.
func (by *Bybit) GetFuturesSymbolPriceTicker(symbol currency.Pair) ([]SymbolPriceTicker, error) {
	var resp []SymbolPriceTicker
	params := url.Values{}

	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)

	path := common.EncodeURLValues(bybitFuturesAPIVersion+cfuturesSymbolPriceTicker, params)
	return resp, by.SendHTTPRequest(exchange.RestCoinMargined, path, &resp)
}

// GetPublicTrades gets past public trades for CoinMarginedFutures.
func (by *Bybit) GetPublicTrades(symbol currency.Pair, limit, fromID int64) ([]FuturesPublicTradesData, error) {
	var resp []FuturesPublicTradesData
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if fromID != 0 {
		params.Set("fromID", strconv.FormatInt(fromID, 10))
	}

	path := common.EncodeURLValues(bybitFuturesAPIVersion+cfuturesRecentTrades, params)
	return resp, by.SendHTTPRequest(exchange.RestCoinMargined, path, &resp)
}

// GetSymbolsInfo gets all symbol pair information for CoinMarginedFutures.
func (by *Bybit) GetSymbolsInfo(symbol currency.Pair, limit, fromID int64) ([]SymbolInfo, error) {
	var resp []SymbolInfo
	return resp, by.SendHTTPRequest(exchange.RestCoinMargined, cfuturesSymbolInfo, &resp)
}

// GetFuturesLiquidationOrders gets liquidation orders
func (by *Bybit) GetFuturesLiquidationOrders(symbol currency.Pair, fromID, limit int64, startTime, endTime time.Time) ([]AllLiquidationOrders, error) {
	var resp []AllLiquidationOrders
	params := url.Values{}

	if !symbol.IsEmpty() {
		symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return resp, err
		}
		params.Set("symbol", symbolValue)
	} else {
		return resp, errors.New("symbol missing")
	}
	if fromID != 0 {
		params.Set("from", strconv.FormatInt(fromID, 10))
	}
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if startTime.After(endTime) {
			return resp, errors.New("startTime cannot be after endTime")
		}
		params.Set("start_time", strconv.FormatInt(startTime.Unix()*1000, 10))
		params.Set("end_time", strconv.FormatInt(endTime.Unix()*1000, 10))
	}

	path := common.EncodeURLValues(bybitFuturesAPIVersion+cfuturesLiquidationOrders, params)
	return resp, by.SendHTTPRequest(exchange.RestCoinMargined, path, &resp)
}

// GetMarkPriceKline gets mark price kline data
func (by *Bybit) GetMarkPriceKline(symbol currency.Pair, interval string, limit int64, startTime time.Time) ([]MarkPriceKlineData, error) {
	var resp []MarkPriceKlineData
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesIntervals, interval) {
		return resp, errors.New("invalid interval parsed")
	}
	params.Set("interval", interval)
	if !startTime.IsZero() {
		params.Set("from", strconv.FormatInt(startTime.Unix(), 10))
	} else {
		return resp, errors.New("startTime can't be zero or missing")
	}

	path := common.EncodeURLValues(bybitFuturesAPIVersion+cfuturesMarkPriceKline, params)
	return resp, by.SendHTTPRequest(exchange.RestCoinMargined, path, &resp)
}

// GetIndexPriceKline gets index price kline data
func (by *Bybit) GetIndexPriceKline(symbol currency.Pair, interval string, limit int64, startTime time.Time) ([]IndexPriceKlineData, error) {
	var resp []IndexPriceKlineData
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesIntervals, interval) {
		return resp, errors.New("invalid interval parsed")
	}
	params.Set("interval", interval)
	if !startTime.IsZero() {
		params.Set("from", strconv.FormatInt(startTime.Unix(), 10))
	} else {
		return resp, errors.New("startTime can't be zero or missing")
	}

	path := common.EncodeURLValues(bybitFuturesAPIVersion+cfuturesIndexKline, params)
	return resp, by.SendHTTPRequest(exchange.RestCoinMargined, path, &resp)
}

// GetPremiumIndexPriceKline gets premium index price kline data
func (by *Bybit) GetPremiumIndexPriceKline(symbol currency.Pair, interval string, limit int64, startTime time.Time) ([]IndexPriceKlineData, error) {
	var resp []IndexPriceKlineData
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesIntervals, interval) {
		return resp, errors.New("invalid interval parsed")
	}
	params.Set("interval", interval)
	if !startTime.IsZero() {
		params.Set("from", strconv.FormatInt(startTime.Unix(), 10))
	} else {
		return resp, errors.New("startTime can't be zero or missing")
	}

	path := common.EncodeURLValues(bybitFuturesAPIVersion+cfuturesIndexPremiumKline, params)
	return resp, by.SendHTTPRequest(exchange.RestCoinMargined, path, &resp)
}

// GetOpenInterest gets open interest data for a symbol.
func (by *Bybit) GetOpenInterest(symbol currency.Pair, period string, limit int64) (OpenInterestData, error) {
	var resp OpenInterestData
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesPeriods, period) {
		return resp, errors.New("invalid period parsed")
	}
	params.Set("period", period)

	path := common.EncodeURLValues(bybitFuturesAPIVersion+cfuturesOpenInterest, params)
	return resp, by.SendHTTPRequest(exchange.RestCoinMargined, path, &resp)
}

// GetLatestBigDeal gets filled orders worth more than 500,000 USD within the last 24h for symbol.
func (by *Bybit) GetLatestBigDeal(symbol currency.Pair, limit int64) (BigDealData, error) {
	var resp BigDealData
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}

	path := common.EncodeURLValues(bybitFuturesAPIVersion+cfuturesBigDeal, params)
	return resp, by.SendHTTPRequest(exchange.RestCoinMargined, path, &resp)
}

// GetAccountRatio gets user accounts long-short ratio.
func (by *Bybit) GetAccountRatio(symbol currency.Pair, period string, limit int64) (AccountRatioData, error) {
	var resp AccountRatioData
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 500 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesPeriods, period) {
		return resp, errors.New("invalid period parsed")
	}
	params.Set("period", period)

	path := common.EncodeURLValues(bybitFuturesAPIVersion+cfuturesAccountRatio, params)
	return resp, by.SendHTTPRequest(exchange.RestCoinMargined, path, &resp)
}

// CreateFuturesOrder sends a new futures order to the exchange
func (by *Bybit) CreateFuturesOrder(symbol currency.Pair, side, orderType, timeInForce,
	orderLinkID, takeProfitTriggerBy, stopLossTriggerBy string,
	quantity, price, takeProfit, stopLoss float64, closeOnTrigger, reduceOnly bool) (FuturesOrderData, error) {
	var resp FuturesOrderData
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	params.Set("side", side)
	params.Set("order_type", orderType)
	if quantity != 0 {
		params.Set("qty", strconv.FormatFloat(quantity, 'f', -1, 64))
	} else {
		return resp, errors.New("quantity can't be zero or missing")
	}
	if price != 0 {
		params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	}
	if timeInForce != "" {
		params.Set("time_in_force", timeInForce)
	} else {
		return resp, errors.New("timeInForce can't be empty or missing")
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
	return resp, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesCreateOrder, params, &resp, bybitAuthRate)
}

// GetActiveFuturesOrders gets list of futures active orders
func (by *Bybit) GetActiveFuturesOrders(symbol currency.Pair, orderStatus, direction, cursor string, limit int64) ([]FuturesActiveOrders, error) {
	resp := struct {
		Result struct {
			Data   []FuturesActiveOrders `json:"data"`
			Cursor string                `json:"cursor"`
		} `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
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
	return resp.Result.Data, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, bybitFuturesAPIVersion+cfuturesGetActiveOrders, params, &resp, bybitAuthRate)
}

// CancelActiveFuturesOrders cancels futures unfilled or partially filled orders
func (by *Bybit) CancelActiveFuturesOrders(symbol currency.Pair, orderID, orderLinkID string) (FuturesOrderData, error) {
	var resp FuturesOrderData
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	if orderID == "" && orderLinkID == "" {
		return resp, errors.New("one among orderID or orderLinkID should be present")
	}
	if orderID != "" {
		params.Set("order_id", orderID)
	}
	if orderLinkID != "" {
		params.Set("order_link_id", orderLinkID)
	}
	return resp, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesCancelActiveOrder, params, &resp, bybitAuthRate)
}

// CancelAllActiveFuturesOrders cancels all futures unfilled or partially filled orders
func (by *Bybit) CancelAllActiveFuturesOrders(symbol currency.Pair) (FuturesOrderData, error) {
	var resp FuturesOrderData
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)
	return resp, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesCancelAllActiveOrders, params, &resp, bybitAuthRate)
}

// ReplaceActiveFuturesOrders modify unfilled or partially filled orders
func (by *Bybit) ReplaceActiveFuturesOrders(symbol currency.Pair, orderID, orderLinkID string,
	updatedQty, updatedPrice, takeProfitPrice, stopLossPrice, takeProfitTriggerBy, stopLossTriggerBy float64) (string, error) {
	resp := struct {
		Data struct {
			OrderID string `json:"order_id"`
		} `json:"result"`
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
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
	if takeProfitTriggerBy != 0 {
		params.Set("tp_trigger_by", strconv.FormatFloat(takeProfitTriggerBy, 'f', -1, 64))
	}
	if stopLossTriggerBy != 0 {
		params.Set("sl_trigger_by", strconv.FormatFloat(stopLossTriggerBy, 'f', -1, 64))
	}
	return resp.Data.OrderID, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesReplaceActiveOrder, params, &resp, bybitAuthRate)
}

// GetActiveRealtimeOrders query real time order data
func (by *Bybit) GetActiveRealtimeOrders(symbol currency.Pair, orderID, orderLinkID string) ([]FuturesActiveRealtimeOrder, error) {
	var data []FuturesActiveRealtimeOrder
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return data, err
	}
	params.Set("symbol", symbolValue)
	if orderID == "" && orderLinkID == "" {
		return data, errors.New("one among orderID or orderLinkID should be present")
	}
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
		err = by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesGetActiveRealtimeOrders, params, &resp, bybitAuthRate)
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
		err = by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesGetActiveRealtimeOrders, params, &resp, bybitAuthRate)
		if err != nil {
			return data, err
		}
		data = append(data, resp.Data)
	}
	return data, nil
}

// CreateConditionalFuturesOrder sends a new conditional futures order to the exchange
func (by *Bybit) CreateConditionalFuturesOrder(symbol currency.Pair, side, orderType, timeInForce,
	orderLinkID, takeProfitTriggerBy, stopLossTriggerBy, triggerBy string,
	quantity, price, takeProfit, stopLoss, basePrice, stopPrice float64, closeOnTrigger bool) (FuturesConditionalOrderData, error) {
	resp := struct {
		Data FuturesConditionalOrderData `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
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
	return resp.Data, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesCreateConditionalOrder, params, &resp, bybitAuthRate)
}

// GetConditionalFuturesOrders gets list of futures conditional orders
func (by *Bybit) GetConditionalFuturesOrders(symbol currency.Pair, stopOrderStatus, direction, cursor string, limit int64) ([]FuturesConditionalOrders, error) {
	resp := struct {
		Result struct {
			Data   []FuturesConditionalOrders `json:"data"`
			Cursor string                     `json:"cursor"`
		} `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp.Result.Data, err
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
	return resp.Result.Data, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, bybitFuturesAPIVersion+cfuturesGetConditionalOrders, params, &resp, bybitAuthRate)
}

// CancelConditionalFuturesOrders cancels untriggered conditional orders
func (by *Bybit) CancelConditionalFuturesOrders(symbol currency.Pair, stopOrderID, orderLinkID string) (string, error) {
	resp := struct {
		Data struct {
			StopOrderID string `json:"stop_order_id"`
		} `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
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
	return resp.Data.StopOrderID, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesCancelConditionalOrder, params, &resp, bybitAuthRate)
}

// CancelAllConditionalFuturesOrders cancels all untriggered conditional orders
func (by *Bybit) CancelAllConditionalFuturesOrders(symbol currency.Pair) ([]FuturesCancelOrderData, error) {
	resp := struct {
		Data []FuturesCancelOrderData `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	return resp.Data, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesCancelAllConditionalOrders, params, &resp, bybitAuthRate)
}

// ReplaceConditionalFuturesOrders modify unfilled or partially filled conditional orders
func (by *Bybit) ReplaceConditionalFuturesOrders(symbol currency.Pair, stopOrderID, orderLinkID string,
	updatedQty, updatedPrice, takeProfitPrice, stopLossPrice, takeProfitTriggerBy, stopLossTriggerBy, orderTriggerPrice float64) (string, error) {
	resp := struct {
		Data struct {
			OrderID string `json:"stop_order_id"`
		} `json:"result"`
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
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
	if takeProfitTriggerBy != 0 {
		params.Set("tp_trigger_by", strconv.FormatFloat(takeProfitTriggerBy, 'f', -1, 64))
	}
	if stopLossTriggerBy != 0 {
		params.Set("sl_trigger_by", strconv.FormatFloat(stopLossTriggerBy, 'f', -1, 64))
	}
	return resp.Data.OrderID, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesReplaceConditionalOrder, params, &resp, bybitAuthRate)
}

// GetConditionalRealtimeOrders query real time considitional order data
func (by *Bybit) GetConditionalRealtimeOrders(symbol currency.Pair, stopOrderID, orderLinkID string) ([]FuturesConditionalRealtimeOrder, error) {
	var data []FuturesConditionalRealtimeOrder
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return data, err
	}
	params.Set("symbol", symbolValue)
	if stopOrderID == "" && orderLinkID == "" {
		return data, errors.New("one among stopOrderID or orderLinkID should be present")
	}
	if stopOrderID != "" {
		params.Set("stop_order_id", stopOrderID)
	}
	if orderLinkID != "" {
		params.Set("order_link_id", orderLinkID)
	}

	if stopOrderID == "" && orderLinkID == "" {
		resp := struct {
			Data []FuturesConditionalRealtimeOrder `json:"result"`
		}{}
		err = by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesGetConditionalRealtimeOrders, params, &resp, bybitAuthRate)
		if err != nil {
			return data, err
		}
		for _, d := range resp.Data {
			data = append(data, d)
		}
	} else {
		resp := struct {
			Data FuturesConditionalRealtimeOrder `json:"result"`
		}{}
		err = by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesGetConditionalRealtimeOrders, params, &resp, bybitAuthRate)
		if err != nil {
			return data, err
		}
		data = append(data, resp.Data)
	}
	return data, nil
}
