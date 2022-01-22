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

// TODO: club redundant struct in futures type
const (
	bybitFuturesAPIVersion = "/v2"

	// public endpoint
	cfuturesOrderbook          = "/public/orderBook/L2"
	cfuturesKline              = "/public/kline/list"
	cfuturesSymbolPriceTicker  = "/public/tickers"
	cfuturesRecentTrades       = "/public/trading-records"
	cfuturesSymbolInfo         = "/public/symbols"
	cfuturesMarkPriceKline     = "/public/mark-price-kline"
	cfuturesIndexKline         = "/public/index-price-kline"
	cfuturesIndexPremiumKline  = "/public/premium-index-kline"
	cfuturesOpenInterest       = "/public/open-interest"
	cfuturesBigDeal            = "/public/big-deal"
	cfuturesAccountRatio       = "/public/account-ratio"
	cfuturesGetRiskLimit       = "/public/risk-limit/list"
	cfuturesGetLastFundingRate = "/public/funding/prev-funding-rate"
	cfuturesGetServerTime      = "/public/time"
	cfuturesGetAnnouncement    = "/public/announcement"

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

	cfuturesPosition        = "/private/position/list"
	cfuturesUpdateMargin    = "/private/position/change-position-margin"
	cfuturesSetTrading      = "/private/position/trading-stop"
	cfuturesSetLeverage     = "/private/position/leverage/save"
	cfuturesGetTrades       = "/private/execution/list"
	cfuturesGetClosedTrades = "/private/trade/closed-pnl/list"
	cfuturesSwitchPosition  = "/private/tpsl/switch-mode"
	cfuturesSwitchMargin    = "/private/position/switch-isolated"

	cfuturesSetRiskLimit                   = "/private/position/risk-limit"
	cfuturesGetMyLastFundingFee            = "/private/funding/prev-funding"
	cfuturesPredictFundingRate             = "/private/funding/predicted-funding"
	cfuturesGetAPIKeyInfo                  = "/private/account/api-key"
	cfuturesGetLiquidityContributionPoints = "/private/account/lcp"

	cfuturesGetWalletBalance           = "/private/wallet/balance"
	cfuturesGetWalletFundRecords       = "/private/wallet/fund/records"
	cfuturesGetWalletWithdrawalRecords = "/private/wallet/withdraw/list"
	cfuturesGetAssetExchangeRecords    = "/private/exchange-order/list"
)

// GetFuturesOrderbook gets orderbook data for CoinMarginedFutures.
func (by *Bybit) GetFuturesOrderbook(symbol currency.Pair) (Orderbook, error) {
	var resp Orderbook
	data := struct {
		Result []OrderbookData `json:"result"`
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp, err
	}
	params.Set("symbol", symbolValue)

	path := common.EncodeURLValues(bybitFuturesAPIVersion+cfuturesOrderbook, params)
	err = by.SendHTTPRequest(exchange.RestCoinMargined, path, publicFuturesRate, &data)
	if err != nil {
		return resp, err
	}

	for _, ob := range data.Result {
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
func (by *Bybit) GetFuturesKlineData(symbol currency.Pair, interval string, limit int64, startTime time.Time) ([]FuturesCandleStickWithStringParam, error) {
	resp := struct {
		Data []FuturesCandleStickWithStringParam `json:"result"`
	}{}

	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
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

	path := common.EncodeURLValues(bybitFuturesAPIVersion+cfuturesKline, params)
	err := by.SendHTTPRequest(exchange.RestCoinMargined, path, publicFuturesRate, &resp)
	if err != nil {
		return resp.Data, err
	}

	return resp.Data, nil
}

// GetFuturesSymbolPriceTicker gets price ticker for symbol.
func (by *Bybit) GetFuturesSymbolPriceTicker(symbol currency.Pair) ([]SymbolPriceTicker, error) {
	resp := struct {
		Data []SymbolPriceTicker `json:"result"`
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)

	path := common.EncodeURLValues(bybitFuturesAPIVersion+cfuturesSymbolPriceTicker, params)
	return resp.Data, by.SendHTTPRequest(exchange.RestCoinMargined, path, publicFuturesRate, &resp)
}

// GetPublicTrades gets past public trades for CoinMarginedFutures.
func (by *Bybit) GetPublicTrades(symbol currency.Pair, limit, fromID int64) ([]FuturesPublicTradesData, error) {
	resp := struct {
		Data []FuturesPublicTradesData `json:"result"`
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if fromID != 0 {
		params.Set("fromID", strconv.FormatInt(fromID, 10))
	}

	path := common.EncodeURLValues(bybitFuturesAPIVersion+cfuturesRecentTrades, params)
	return resp.Data, by.SendHTTPRequest(exchange.RestCoinMargined, path, publicFuturesRate, &resp)
}

// GetSymbolsInfo gets all symbol pair information for CoinMarginedFutures.
func (by *Bybit) GetSymbolsInfo() ([]SymbolInfo, error) {
	resp := struct {
		Data []SymbolInfo `json:"result"`
	}{}
	return resp.Data, by.SendHTTPRequest(exchange.RestCoinMargined, bybitFuturesAPIVersion+cfuturesSymbolInfo, publicFuturesRate, &resp)
}

// GetMarkPriceKline gets mark price kline data
func (by *Bybit) GetMarkPriceKline(symbol currency.Pair, interval string, limit int64, startTime time.Time) ([]MarkPriceKlineData, error) {
	resp := struct {
		Data []MarkPriceKlineData `json:"result"`
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
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

	path := common.EncodeURLValues(bybitFuturesAPIVersion+cfuturesMarkPriceKline, params)
	return resp.Data, by.SendHTTPRequest(exchange.RestCoinMargined, path, publicFuturesRate, &resp)
}

// GetIndexPriceKline gets index price kline data
func (by *Bybit) GetIndexPriceKline(symbol currency.Pair, interval string, limit int64, startTime time.Time) ([]IndexPriceKlineData, error) {
	resp := struct {
		Data []IndexPriceKlineData `json:"result"`
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
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

	path := common.EncodeURLValues(bybitFuturesAPIVersion+cfuturesIndexKline, params)
	return resp.Data, by.SendHTTPRequest(exchange.RestCoinMargined, path, publicFuturesRate, &resp)
}

// GetPremiumIndexPriceKline gets premium index price kline data
func (by *Bybit) GetPremiumIndexPriceKline(symbol currency.Pair, interval string, limit int64, startTime time.Time) ([]IndexPriceKlineData, error) {
	resp := struct {
		Data []IndexPriceKlineData `json:"result"`
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
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

	path := common.EncodeURLValues(bybitFuturesAPIVersion+cfuturesIndexPremiumKline, params)
	return resp.Data, by.SendHTTPRequest(exchange.RestCoinMargined, path, publicFuturesRate, &resp)
}

// GetOpenInterest gets open interest data for a symbol.
func (by *Bybit) GetOpenInterest(symbol currency.Pair, period string, limit int64) ([]OpenInterestData, error) {
	resp := struct {
		Data []OpenInterestData `json:"result"`
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesPeriods, period) {
		return resp.Data, errors.New("invalid period parsed")
	}
	params.Set("period", period)

	path := common.EncodeURLValues(bybitFuturesAPIVersion+cfuturesOpenInterest, params)
	return resp.Data, by.SendHTTPRequest(exchange.RestCoinMargined, path, publicFuturesRate, &resp)
}

// GetLatestBigDeal gets filled orders worth more than 500,000 USD within the last 24h for symbol.
func (by *Bybit) GetLatestBigDeal(symbol currency.Pair, limit int64) ([]BigDealData, error) {
	resp := struct {
		Data []BigDealData `json:"result"`
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 1000 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}

	path := common.EncodeURLValues(bybitFuturesAPIVersion+cfuturesBigDeal, params)
	return resp.Data, by.SendHTTPRequest(exchange.RestCoinMargined, path, publicFuturesRate, &resp)
}

// GetAccountRatio gets user accounts long-short ratio.
func (by *Bybit) GetAccountRatio(symbol currency.Pair, period string, limit int64) ([]AccountRatioData, error) {
	resp := struct {
		Data []AccountRatioData `json:"result"`
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	if limit > 0 && limit <= 500 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	if !common.StringDataCompare(validFuturesPeriods, period) {
		return resp.Data, errors.New("invalid period parsed")
	}
	params.Set("period", period)

	path := common.EncodeURLValues(bybitFuturesAPIVersion+cfuturesAccountRatio, params)
	return resp.Data, by.SendHTTPRequest(exchange.RestCoinMargined, path, publicFuturesRate, &resp)
}

// GetRiskLimit returns risk limit
func (by *Bybit) GetRiskLimit(symbol currency.Pair) ([]RiskInfoWithStringParam, error) {
	resp := struct {
		Data []RiskInfoWithStringParam `json:"result"`
	}{}

	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return resp.Data, err
		}
		params.Set("symbol", symbolValue)
	}

	path := common.EncodeURLValues(bybitFuturesAPIVersion+cfuturesGetRiskLimit, params)
	return resp.Data, by.SendHTTPRequest(exchange.RestCoinMargined, path, publicFuturesRate, &resp)
}

// GetLastFundingRate returns latest generated funding fee
func (by *Bybit) GetLastFundingRate(symbol currency.Pair) (FundingInfo, error) {
	resp := struct {
		Data FundingInfo `json:"result"`
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)

	path := common.EncodeURLValues(bybitFuturesAPIVersion+cfuturesGetLastFundingRate, params)
	return resp.Data, by.SendHTTPRequest(exchange.RestCoinMargined, path, publicFuturesRate, &resp)
}

// GetServerTime returns Bybit server time in seconds
func (by *Bybit) GetServerTime() (float64, error) {
	resp := struct {
		TimeNow float64 `json:"time_now,string"`
	}{}
	return resp.TimeNow, by.SendHTTPRequest(exchange.RestCoinMargined, bybitFuturesAPIVersion+cfuturesGetServerTime, publicFuturesRate, &resp)
}

// GetAnnouncement returns announcements in the last 30 days in reverse order
func (by *Bybit) GetAnnouncement() ([]AnnouncementInfo, error) {
	resp := struct {
		Data []AnnouncementInfo `json:"result"`
	}{}
	return resp.Data, by.SendHTTPRequest(exchange.RestCoinMargined, bybitFuturesAPIVersion+cfuturesGetAnnouncement, publicFuturesRate, &resp)
}

// CreateCoinFuturesOrder sends a new futures order to the exchange
func (by *Bybit) CreateCoinFuturesOrder(symbol currency.Pair, side, orderType, timeInForce,
	orderLinkID, takeProfitTriggerBy, stopLossTriggerBy string,
	quantity, price, takeProfit, stopLoss float64, closeOnTrigger, reduceOnly bool) (FuturesOrderDataResp, error) {
	resp := struct {
		Data FuturesOrderDataResp `json:"result"`
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
	return resp.Data, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesCreateOrder, params, &resp, cFuturesCreateOrderRate)
}

// GetActiveCoinFuturesOrders gets list of futures active orders
func (by *Bybit) GetActiveCoinFuturesOrders(symbol currency.Pair, orderStatus, direction, cursor string, limit int64) ([]FuturesActiveOrderResp, error) {
	resp := struct {
		Result struct {
			Data   []FuturesActiveOrderResp `json:"data"`
			Cursor string                   `json:"cursor"`
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
	return resp.Result.Data, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, bybitFuturesAPIVersion+cfuturesGetActiveOrders, params, &resp, cFuturesGetActiveOrderRate)
}

// CancelActiveCoinFuturesOrders cancels futures unfilled or partially filled orders
func (by *Bybit) CancelActiveCoinFuturesOrders(symbol currency.Pair, orderID, orderLinkID string) (FuturesOrderDataResp, error) {
	resp := struct {
		Data FuturesOrderDataResp `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	if orderID == "" && orderLinkID == "" {
		return resp.Data, errors.New("one among orderID or orderLinkID should be present")
	}
	if orderID != "" {
		params.Set("order_id", orderID)
	}
	if orderLinkID != "" {
		params.Set("order_link_id", orderLinkID)
	}
	return resp.Data, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesCancelActiveOrder, params, &resp, cFuturesCancelActiveOrderRate)
}

// CancelAllActiveCoinFuturesOrders cancels all futures unfilled or partially filled orders
func (by *Bybit) CancelAllActiveCoinFuturesOrders(symbol currency.Pair) ([]FuturesOrderDataResp, error) {
	resp := struct {
		Data []FuturesOrderDataResp `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	return resp.Data, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesCancelAllActiveOrders, params, &resp, cFuturesCancelAllActiveOrderRate)
}

// ReplaceActiveCoinFuturesOrders modify unfilled or partially filled orders
func (by *Bybit) ReplaceActiveCoinFuturesOrders(symbol currency.Pair, orderID, orderLinkID, takeProfitTriggerBy, stopLossTriggerBy string,
	updatedQty int64, updatedPrice, takeProfitPrice, stopLossPrice float64) (string, error) {
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
	return resp.Data.OrderID, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesReplaceActiveOrder, params, &resp, cFuturesReplaceActiveOrderRate)
}

// GetActiveRealtimeCoinOrders query real time order data
func (by *Bybit) GetActiveRealtimeCoinOrders(symbol currency.Pair, orderID, orderLinkID string) ([]FuturesActiveRealtimeOrder, error) {
	var data []FuturesActiveRealtimeOrder
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
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
		err = by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, bybitFuturesAPIVersion+cfuturesGetActiveRealtimeOrders, params, &resp, cFuturesGetRealtimeOrderRate)
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
		err = by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, bybitFuturesAPIVersion+cfuturesGetActiveRealtimeOrders, params, &resp, cFuturesGetRealtimeOrderRate)
		if err != nil {
			return data, err
		}
		data = append(data, resp.Data)
	}
	return data, nil
}

// CreateConditionalCoinFuturesOrder sends a new conditional futures order to the exchange
func (by *Bybit) CreateConditionalCoinFuturesOrder(symbol currency.Pair, side, orderType, timeInForce,
	orderLinkID, takeProfitTriggerBy, stopLossTriggerBy, triggerBy string,
	quantity, price, takeProfit, stopLoss, basePrice, stopPrice float64, closeOnTrigger bool) (FuturesConditionalOrderResp, error) {
	resp := struct {
		Data FuturesConditionalOrderResp `json:"result"`
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
	return resp.Data, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesCreateConditionalOrder, params, &resp, cFuturesCreateConditionalOrderRate)
}

// GetConditionalCoinFuturesOrders gets list of futures conditional orders
func (by *Bybit) GetConditionalCoinFuturesOrders(symbol currency.Pair, stopOrderStatus, direction, cursor string, limit int64) ([]CoinFuturesConditionalOrders, error) {
	resp := struct {
		Result struct {
			Data   []CoinFuturesConditionalOrders `json:"data"`
			Cursor string                         `json:"cursor"`
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
	return resp.Result.Data, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, bybitFuturesAPIVersion+cfuturesGetConditionalOrders, params, &resp, cFuturesGetConditionalOrderRate)
}

// CancelConditionalCoinFuturesOrders cancels untriggered conditional orders
func (by *Bybit) CancelConditionalCoinFuturesOrders(symbol currency.Pair, stopOrderID, orderLinkID string) (string, error) {
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
	return resp.Data.StopOrderID, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesCancelConditionalOrder, params, &resp, cFuturesCancelConditionalOrderRate)
}

// CancelAllConditionalCoinFuturesOrders cancels all untriggered conditional orders
func (by *Bybit) CancelAllConditionalCoinFuturesOrders(symbol currency.Pair) ([]FuturesCancelOrderResp, error) {
	resp := struct {
		Data []FuturesCancelOrderResp `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	return resp.Data, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesCancelAllConditionalOrders, params, &resp, cFuturesCancelAllConditionalOrderRate)
}

// ReplaceConditionalCoinFuturesOrders modify unfilled or partially filled conditional orders
func (by *Bybit) ReplaceConditionalCoinFuturesOrders(symbol currency.Pair, stopOrderID, orderLinkID, takeProfitTriggerBy, stopLossTriggerBy string,
	updatedQty, updatedPrice, takeProfitPrice, stopLossPrice, orderTriggerPrice float64) (string, error) {
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
	if takeProfitTriggerBy != "" {
		params.Set("tp_trigger_by", takeProfitTriggerBy)
	}
	if stopLossTriggerBy != "" {
		params.Set("sl_trigger_by", stopLossTriggerBy)
	}
	return resp.Data.OrderID, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesReplaceConditionalOrder, params, &resp, cFuturesReplaceConditionalOrderRate)
}

// GetConditionalRealtimeCoinOrders query real time conditional order data
func (by *Bybit) GetConditionalRealtimeCoinOrders(symbol currency.Pair, stopOrderID, orderLinkID string) ([]CoinFuturesConditionalRealtimeOrder, error) {
	var data []CoinFuturesConditionalRealtimeOrder
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
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
			Data []CoinFuturesConditionalRealtimeOrder `json:"result"`
		}{}
		err = by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, bybitFuturesAPIVersion+cfuturesGetConditionalRealtimeOrders, params, &resp, cFuturesDefaultRate)
		if err != nil {
			return data, err
		}
		for _, d := range resp.Data {
			data = append(data, d)
		}
	} else {
		resp := struct {
			Data CoinFuturesConditionalRealtimeOrder `json:"result"`
		}{}
		err = by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, bybitFuturesAPIVersion+cfuturesGetConditionalRealtimeOrders, params, &resp, cFuturesDefaultRate)
		if err != nil {
			return data, err
		}
		data = append(data, resp.Data)
	}
	return data, nil
}

// GetCoinPositions returns list of user positions
func (by *Bybit) GetCoinPositions(symbol currency.Pair) ([]PositionResp, error) {
	var data []PositionResp
	params := url.Values{}

	if !symbol.IsEmpty() {
		resp := struct {
			Data PositionResp `json:"result"`
		}{}

		symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
		if err != nil {
			return data, err
		}
		params.Set("symbol", symbolValue)

		err = by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, bybitFuturesAPIVersion+cfuturesPosition, params, &resp, cFuturesPositionRate)
		if err != nil {
			return data, err
		}
		data = append(data, resp.Data)
	} else {
		resp := struct {
			Data []PositionResp `json:"result"`
		}{}
		err := by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, bybitFuturesAPIVersion+cfuturesPosition, params, &resp, cFuturesPositionRate)
		if err != nil {
			return data, err
		}
		for _, d := range resp.Data {
			data = append(data, d)
		}
	}
	return data, nil
}

// SetCoinMargin updates margin
func (by *Bybit) SetCoinMargin(symbol currency.Pair, margin string) (float64, error) {
	resp := struct {
		Data float64 `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	if margin != "" {
		params.Set("margin", margin)
	} else {
		return resp.Data, errors.New("margin can't be empty")
	}
	return resp.Data, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesUpdateMargin, params, &resp, cFuturesUpdateMarginRate)
}

// SetCoinTradingAndStop sets take profit, stop loss, and trailing stop for your open position
func (by *Bybit) SetCoinTradingAndStop(symbol currency.Pair, takeProfit, stopLoss, trailingStop, newTrailingActive, stopLossQty, takeProfitQty float64, takeProfitTriggerBy, stopLossTriggerBy string) (SetTradingAndStopResp, error) {
	resp := struct {
		Data SetTradingAndStopResp `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
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

	return resp.Data, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesSetTrading, params, &resp, cFuturesDefaultRate)
}

// SetCoinLeverage sets leverage
func (by *Bybit) SetCoinLeverage(symbol currency.Pair, leverage float64, leverageOnly bool) (float64, error) {
	resp := struct {
		Data float64 `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)
	if leverage > 0 {
		params.Set("leverage", strconv.FormatFloat(leverage, 'f', -1, 64))
	} else {
		return resp.Data, errors.New("leverage can't be zero or less then it")
	}
	if leverageOnly {
		params.Set("leverage_only", "true")
	}

	return resp.Data, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesSetLeverage, params, &resp, cFuturesSetLeverageRate)
}

// GetCoinTradeRecords returns list of user trades
func (by *Bybit) GetCoinTradeRecords(symbol currency.Pair, orderID, order string, startTime, page, limit int64) ([]TradeResp, error) {
	params := url.Values{}

	resp := struct {
		Data struct {
			OrderID string      `json:"order_id"`
			Trades  []TradeResp `json:"trade_list"`
		} `json:"result"`
	}{}

	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
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

	err = by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, bybitFuturesAPIVersion+cfuturesGetTrades, params, &resp, cFuturesTradeRate)
	if err != nil {
		return resp.Data.Trades, err
	}
	return resp.Data.Trades, nil
}

// GetClosedCoinTrades returns closed profit and loss records
func (by *Bybit) GetClosedCoinTrades(symbol currency.Pair, executionType string, startTime, endTime, page, limit int64) ([]ClosedTrades, error) {
	params := url.Values{}

	resp := struct {
		Data struct {
			CurrentPage int64          `json:"current_page"`
			Trades      []ClosedTrades `json:"data"`
		} `json:"result"`
	}{}

	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
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

	err = by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, bybitFuturesAPIVersion+cfuturesGetClosedTrades, params, &resp, cFuturesDefaultRate)
	if err != nil {
		return resp.Data.Trades, err
	}
	return resp.Data.Trades, nil
}

// ChangeCoinMode switches mode between full or partial position
func (by *Bybit) ChangeCoinMode(symbol currency.Pair, takeProfitStopLoss string) (string, error) {
	resp := struct {
		Data struct {
			Mode string `json:"tp_sl_mode"`
		} `json:"result"`
	}{}
	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp.Data.Mode, err
	}
	params.Set("symbol", symbolValue)
	if takeProfitStopLoss != "" {
		params.Set("tp_sl_mode", takeProfitStopLoss)
	} else {
		return resp.Data.Mode, errors.New("takeProfitStopLoss can't be empty or missing")
	}

	return resp.Data.Mode, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesSwitchPosition, params, &resp, cFuturesSwitchPosition)
}

// ChangeCoinMargin switches margin between cross or isolated
func (by *Bybit) ChangeCoinMargin(symbol currency.Pair, buyLeverage, sellLeverage float64, isIsolated bool) error {
	var resp interface{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
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

	return by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesSwitchMargin, params, &resp, cFuturesDefaultRate)
}

// SetCoinRiskLimit sets risk limit
func (by *Bybit) SetCoinRiskLimit(symbol currency.Pair, riskID int64) (int64, error) {
	resp := struct {
		Data struct {
			RiskID int64 `json:"risk_id"`
		} `json:"result"`
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp.Data.RiskID, err
	}
	params.Set("symbol", symbolValue)

	if riskID > 0 {
		params.Set("risk_id", strconv.FormatInt(riskID, 10))
	} else {
		return resp.Data.RiskID, errors.New("riskID can't be zero or lesser")
	}

	return resp.Data.RiskID, by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodPost, bybitFuturesAPIVersion+cfuturesSetRiskLimit, params, &resp, cFuturesDefaultRate)
}

// GetCoinLastFundingFee returns last funding fees
func (by *Bybit) GetCoinLastFundingFee(symbol currency.Pair) (FundingFee, error) {
	params := url.Values{}

	resp := struct {
		Data FundingFee `json:"result"`
	}{}

	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)

	err = by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, bybitFuturesAPIVersion+cfuturesGetMyLastFundingFee, params, &resp, cFuturesLastFundingFeeRate)
	if err != nil {
		return resp.Data, err
	}
	return resp.Data, nil
}

// GetCoinPredictedFundingRate returns predicted funding rates and fees
func (by *Bybit) GetCoinPredictedFundingRate(symbol currency.Pair) (float64, float64, error) {
	params := url.Values{}

	resp := struct {
		Data struct {
			PredictedFundingRate float64 `json:"predicted_funding_rate"`
			PredictedFundingFee  float64 `json:"predicted_funding_fee"`
		} `json:"result"`
	}{}

	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp.Data.PredictedFundingRate, resp.Data.PredictedFundingFee, err
	}
	params.Set("symbol", symbolValue)

	err = by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, bybitFuturesAPIVersion+cfuturesPredictFundingRate, params, &resp, cFuturesPredictFundingRate)
	if err != nil {
		return resp.Data.PredictedFundingRate, resp.Data.PredictedFundingFee, err
	}
	return resp.Data.PredictedFundingRate, resp.Data.PredictedFundingFee, nil
}

// GetAPIKeyInfo returns user API Key information
func (by *Bybit) GetAPIKeyInfo() ([]APIKeyData, error) {
	params := url.Values{}

	resp := struct {
		Data []APIKeyData `json:"result"`
	}{}

	err := by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, bybitFuturesAPIVersion+cfuturesGetAPIKeyInfo, params, &resp, cFuturesAPIKeyInfoRate)
	if err != nil {
		return resp.Data, err
	}
	return resp.Data, nil
}

// GetLCPInfo returns latest LCP information
func (by *Bybit) GetLCPInfo(symbol currency.Pair) ([]LCPData, error) {
	params := url.Values{}

	resp := struct {
		Data struct {
			LCPList []LCPData `json:"lcp_list"`
		} `json:"result"`
	}{}

	symbolValue, err := by.FormatSymbol(symbol, asset.CoinMarginedFutures)
	if err != nil {
		return resp.Data.LCPList, err
	}
	params.Set("symbol", symbolValue)

	err = by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, bybitFuturesAPIVersion+cfuturesGetLiquidityContributionPoints, params, &resp, cFuturesDefaultRate)
	if err != nil {
		return resp.Data.LCPList, err
	}
	return resp.Data.LCPList, nil
}

// GetFutureWalletBalance returns wallet balance
func (by *Bybit) GetFutureWalletBalance(coin string) (map[string]WalletData, error) {
	params := url.Values{}

	resp := struct {
		Wallets map[string]WalletData `json:"result"`
	}{}

	if coin != "" {
		params.Set("coin", coin)
	}

	err := by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, bybitFuturesAPIVersion+cfuturesGetWalletBalance, params, &resp, cFuturesWalletBalanceRate)
	if err != nil {
		return resp.Wallets, err
	}
	return resp.Wallets, nil
}

// GetWalletFundRecords returns wallet fund records
func (by *Bybit) GetWalletFundRecords(startDate, endDate, currency, coin, walletFundType string, page, limit int64) ([]FundRecord, error) {
	params := url.Values{}

	resp := struct {
		Data struct {
			Records []FundRecord `json:"data"`
		} `json:"result"`
	}{}

	if startDate != "" {
		params.Set("start_date", startDate)
	}
	if endDate != "" {
		params.Set("end_date", endDate)
	}
	if currency != "" {
		params.Set("currency", currency)
	}
	if coin != "" {
		params.Set("coin", coin)
	}
	if walletFundType != "" {
		params.Set("wallet_fund_type", walletFundType)
	}
	if page != 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 && limit <= 50 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}

	err := by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, bybitFuturesAPIVersion+cfuturesGetWalletFundRecords, params, &resp, cFuturesWalletFundRecordRate)
	if err != nil {
		return resp.Data.Records, err
	}
	return resp.Data.Records, nil
}

// GetWalletWithdrawalRecords returns wallet withdrawal records
func (by *Bybit) GetWalletWithdrawalRecords(startDate, endDate, coin, status string, page, limit int64) ([]FundWithdrawalRecord, error) {
	params := url.Values{}

	resp := struct {
		Data struct {
			Records []FundWithdrawalRecord `json:"data"`
		} `json:"result"`
	}{}

	if startDate != "" {
		params.Set("start_date", startDate)
	}
	if endDate != "" {
		params.Set("end_date", endDate)
	}
	if coin != "" {
		params.Set("coin", coin)
	}
	if status != "" {
		params.Set("status", status)
	}
	if page != 0 {
		params.Set("page", strconv.FormatInt(page, 10))
	}
	if limit > 0 && limit <= 50 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}

	err := by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, bybitFuturesAPIVersion+cfuturesGetWalletWithdrawalRecords, params, &resp, cFuturesWalletWithdrawalRate)
	if err != nil {
		return resp.Data.Records, err
	}
	return resp.Data.Records, nil
}

// GetAssetExchangeRecords returns wallet asset exchange records
func (by *Bybit) GetAssetExchangeRecords(direction string, from, limit int64) ([]AssetExchangeRecord, error) {
	params := url.Values{}

	resp := struct {
		Data []AssetExchangeRecord `json:"result"`
	}{}

	if direction != "" {
		params.Set("direction", direction)
	}

	if from != 0 {
		params.Set("from", strconv.FormatInt(from, 10))
	}
	if limit > 0 && limit <= 50 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}

	err := by.SendAuthHTTPRequest(exchange.RestCoinMargined, http.MethodGet, bybitFuturesAPIVersion+cfuturesGetAssetExchangeRecords, params, &resp, cFuturesDefaultRate)
	if err != nil {
		return resp.Data, err
	}
	return resp.Data, nil
}
