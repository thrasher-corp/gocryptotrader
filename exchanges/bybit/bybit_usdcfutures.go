package bybit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (

	// public endpoint
	usdcfuturesGetOrderbook          = "/perpetual/usdc/openapi/public/v1/order-book"
	usdcfuturesGetContracts          = "/perpetual/usdc/openapi/public/v1/symbols"
	usdcfuturesGetSymbols            = "/perpetual/usdc/openapi/public/v1/tick"
	usdcfuturesGetKlines             = "/perpetual/usdc/openapi/public/v1/kline/list"
	usdcfuturesGetMarkPriceKlines    = "/perpetual/usdc/openapi/public/v1/mark-price-kline"
	usdcfuturesGetIndexPriceKlines   = "/perpetual/usdc/openapi/public/v1/index-price-kline"
	usdcfuturesGetPremiumIndexKlines = "/perpetual/usdc/openapi/public/v1/premium-index-kline"
	usdcfuturesGetOpenInterest       = "/perpetual/usdc/openapi/public/v1/open-interest"
	usdcfuturesGetLargeOrders        = "/perpetual/usdc/openapi/public/v1/big-deal"
	usdcfuturesGetAccountRatio       = "/perpetual/usdc/openapi/public/v1/account-ratio"
	usdcfuturesGetLatestTrades       = "/option/usdc/openapi/public/v1/query-trade-latest"
	usdcfuturesGetLastFundingRate    = "/perpetual/usdc/openapi/public/v1/prev-funding-rate"
	usdcfuturesGetRiskLimit          = "/perpetual/usdc/openapi/public/v1/risk-limit/list"

	// auth endpoint
	usdcfuturesPlaceOrder              = "/perpetual/usdc/openapi/private/v1/place-order"
	usdcfuturesModifyOrder             = "/perpetual/usdc/openapi/private/v1/replace-order"
	usdcfuturesCancelOrder             = "/perpetual/usdc/openapi/private/v1/cancel-order"
	usdcfuturesCancelAllActiveOrder    = "/perpetual/usdc/openapi/private/v1/cancel-all"
	usdcfuturesGetActiveOrder          = "/option/usdc/openapi/private/v1/query-active-orders"
	usdcfuturesGetOrderHistory         = "/option/usdc/openapi/private/v1/query-order-history"
	usdcfuturesGetTradeHistory         = "/option/usdc/openapi/private/v1/execution-list"
	usdcfuturesGetTransactionLog       = "/option/usdc/openapi/private/v1/query-transaction-log"
	usdcfuturesGetWalletBalance        = "/option/usdc/openapi/private/v1/query-wallet-balance"
	usdcfuturesGetAssetInfo            = "/option/usdc/openapi/private/v1/query-asset-info"
	usdcfuturesGetMarginInfo           = "/option/usdc/openapi/private/v1/query-margin-info"
	usdcfuturesGetPosition             = "/option/usdc/openapi/private/v1/query-position"
	usdcfuturesSetLeverage             = "/perpetual/usdc/openapi/private/v1/position/leverage/save"
	usdcfuturesGetSettlementHistory    = "/option/usdc/openapi/private/v1/session-settlement"
	usdcfuturesSetRiskLimit            = "/perpetual/usdc/openapi/private/v1/position/set-risk-limit"
	usdcfuturesGetPredictedFundingRate = "/perpetual/usdc/openapi/private/v1/predicted-funding"
)

// GetUSDCFuturesOrderbook gets orderbook data for USDCMarginedFutures.
func (by *Bybit) GetUSDCFuturesOrderbook(ctx context.Context, symbol currency.Pair) (*Orderbook, error) {
	var resp Orderbook
	data := struct {
		Result []USDCOrderbookData `json:"result"`
		USDCError
	}{}

	params := url.Values{}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
	if err != nil {
		return nil, err
	}
	params.Set("symbol", symbolValue)

	err = by.SendHTTPRequest(ctx, exchange.RestUSDCMargined, common.EncodeURLValues(usdcfuturesGetOrderbook, params), usdcPublicRate, &data)
	if err != nil {
		return nil, err
	}

	for x := range data.Result {
		switch data.Result[x].Side {
		case sideBuy:
			resp.Bids = append(resp.Bids, orderbook.Item{
				Price:  data.Result[x].Price.Float64(),
				Amount: data.Result[x].Size.Float64(),
			})
		case sideSell:
			resp.Asks = append(resp.Asks, orderbook.Item{
				Price:  data.Result[x].Price.Float64(),
				Amount: data.Result[x].Size.Float64(),
			})
		default:
			return nil, errInvalidSide
		}
	}
	return &resp, nil
}

// GetUSDCContracts gets all contract information for USDCMarginedFutures.
func (by *Bybit) GetUSDCContracts(ctx context.Context, symbol currency.Pair, direction string, limit int64) ([]USDCContract, error) {
	resp := struct {
		Data []USDCContract `json:"result"`
		USDCError
	}{}

	params := url.Values{}
	if !symbol.IsEmpty() {
		symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
		if err != nil {
			return resp.Data, err
		}
		params.Set("symbol", symbolValue)
	}

	if direction != "" {
		params.Set("direction", direction)
	}
	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}

	return resp.Data, by.SendHTTPRequest(ctx, exchange.RestUSDCMargined, common.EncodeURLValues(usdcfuturesGetContracts, params), usdcPublicRate, &resp)
}

// GetUSDCSymbols gets all symbol information for USDCMarginedFutures.
func (by *Bybit) GetUSDCSymbols(ctx context.Context, symbol currency.Pair) (USDCSymbol, error) {
	resp := struct {
		Data USDCSymbol `json:"result"`
		USDCError
	}{}

	params := url.Values{}
	if symbol.IsEmpty() {
		return USDCSymbol{}, errSymbolMissing
	}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)

	return resp.Data, by.SendHTTPRequest(ctx, exchange.RestUSDCMargined, common.EncodeURLValues(usdcfuturesGetSymbols, params), usdcPublicRate, &resp)
}

// GetUSDCKlines gets kline of symbol for USDCMarginedFutures.
func (by *Bybit) GetUSDCKlines(ctx context.Context, symbol currency.Pair, period string, startTime time.Time, limit int64) ([]USDCKline, error) {
	resp := struct {
		Data []USDCKline `json:"result"`
		USDCError
	}{}

	params := url.Values{}
	if symbol.IsEmpty() {
		return nil, errSymbolMissing
	}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)

	if !common.StringDataCompare(validFuturesIntervals, period) {
		return resp.Data, errInvalidPeriod
	}
	params.Set("period", period)

	if startTime.IsZero() {
		return nil, errInvalidStartTime
	}
	params.Set("startTime", strconv.FormatInt(startTime.Unix(), 10))

	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp.Data, by.SendHTTPRequest(ctx, exchange.RestUSDCMargined, common.EncodeURLValues(usdcfuturesGetKlines, params), usdcPublicRate, &resp)
}

// GetUSDCMarkPriceKlines gets mark price kline of symbol for USDCMarginedFutures.
func (by *Bybit) GetUSDCMarkPriceKlines(ctx context.Context, symbol currency.Pair, period string, startTime time.Time, limit int64) ([]USDCKlineBase, error) {
	resp := struct {
		Data []USDCKlineBase `json:"result"`
		USDCError
	}{}

	params := url.Values{}
	if symbol.IsEmpty() {
		return nil, errSymbolMissing
	}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)

	if !common.StringDataCompare(validFuturesIntervals, period) {
		return resp.Data, errInvalidPeriod
	}
	params.Set("period", period)

	if startTime.IsZero() {
		return nil, errInvalidStartTime
	}
	params.Set("startTime", strconv.FormatInt(startTime.Unix(), 10))

	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp.Data, by.SendHTTPRequest(ctx, exchange.RestUSDCMargined, common.EncodeURLValues(usdcfuturesGetMarkPriceKlines, params), usdcPublicRate, &resp)
}

// GetUSDCIndexPriceKlines gets index price kline of symbol for USDCMarginedFutures.
func (by *Bybit) GetUSDCIndexPriceKlines(ctx context.Context, symbol currency.Pair, period string, startTime time.Time, limit int64) ([]USDCKlineBase, error) {
	resp := struct {
		Data []USDCKlineBase `json:"result"`
		USDCError
	}{}

	params := url.Values{}
	if symbol.IsEmpty() {
		return nil, errSymbolMissing
	}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)

	if !common.StringDataCompare(validFuturesIntervals, period) {
		return resp.Data, errInvalidPeriod
	}
	params.Set("period", period)

	if startTime.IsZero() {
		return nil, errInvalidStartTime
	}
	params.Set("startTime", strconv.FormatInt(startTime.Unix(), 10))

	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp.Data, by.SendHTTPRequest(ctx, exchange.RestUSDCMargined, common.EncodeURLValues(usdcfuturesGetIndexPriceKlines, params), usdcPublicRate, &resp)
}

// GetUSDCPremiumIndexKlines gets premium index kline of symbol for USDCMarginedFutures.
func (by *Bybit) GetUSDCPremiumIndexKlines(ctx context.Context, symbol currency.Pair, period string, startTime time.Time, limit int64) ([]USDCKlineBase, error) {
	resp := struct {
		Data []USDCKlineBase `json:"result"`
		USDCError
	}{}

	params := url.Values{}
	if symbol.IsEmpty() {
		return nil, errSymbolMissing
	}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)

	if !common.StringDataCompare(validFuturesIntervals, period) {
		return resp.Data, errInvalidPeriod
	}
	params.Set("period", period)

	if startTime.IsZero() {
		return nil, errInvalidStartTime
	}
	params.Set("startTime", strconv.FormatInt(startTime.Unix(), 10))

	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp.Data, by.SendHTTPRequest(ctx, exchange.RestUSDCMargined, common.EncodeURLValues(usdcfuturesGetPremiumIndexKlines, params), usdcPublicRate, &resp)
}

// GetUSDCOpenInterest gets open interest of symbol for USDCMarginedFutures.
func (by *Bybit) GetUSDCOpenInterest(ctx context.Context, symbol currency.Pair, period string, limit int64) ([]USDCOpenInterest, error) {
	resp := struct {
		Data []USDCOpenInterest `json:"result"`
		USDCError
	}{}

	params := url.Values{}
	if symbol.IsEmpty() {
		return nil, errSymbolMissing
	}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)

	if !common.StringDataCompare(validFuturesPeriods, period) {
		return resp.Data, errInvalidPeriod
	}
	params.Set("period", period)

	if limit > 0 && limit <= 200 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp.Data, by.SendHTTPRequest(ctx, exchange.RestUSDCMargined, common.EncodeURLValues(usdcfuturesGetOpenInterest, params), usdcPublicRate, &resp)
}

// GetUSDCLargeOrders gets large order of symbol for USDCMarginedFutures.
func (by *Bybit) GetUSDCLargeOrders(ctx context.Context, symbol currency.Pair, limit int64) ([]USDCLargeOrder, error) {
	resp := struct {
		Data []USDCLargeOrder `json:"result"`
		USDCError
	}{}

	params := url.Values{}
	if symbol.IsEmpty() {
		return nil, errSymbolMissing
	}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)

	if limit > 0 && limit <= 100 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp.Data, by.SendHTTPRequest(ctx, exchange.RestUSDCMargined, common.EncodeURLValues(usdcfuturesGetLargeOrders, params), usdcPublicRate, &resp)
}

// GetUSDCAccountRatio gets account long short ratio of symbol for USDCMarginedFutures.
func (by *Bybit) GetUSDCAccountRatio(ctx context.Context, symbol currency.Pair, period string, limit int64) ([]USDCAccountRatio, error) {
	resp := struct {
		Data []USDCAccountRatio `json:"result"`
		USDCError
	}{}

	params := url.Values{}
	if symbol.IsEmpty() {
		return nil, errSymbolMissing
	}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
	if err != nil {
		return resp.Data, err
	}
	params.Set("symbol", symbolValue)

	if !common.StringDataCompare(validFuturesPeriods, period) {
		return resp.Data, errInvalidPeriod
	}
	params.Set("period", period)

	if limit > 0 && limit <= 500 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp.Data, by.SendHTTPRequest(ctx, exchange.RestUSDCMargined, common.EncodeURLValues(usdcfuturesGetAccountRatio, params), usdcPublicRate, &resp)
}

// GetUSDCLatestTrades gets latest 500 trades for USDCMarginedFutures.
func (by *Bybit) GetUSDCLatestTrades(ctx context.Context, symbol currency.Pair, category string, limit int64) ([]USDCTrade, error) {
	resp := struct {
		Result struct {
			ResultSize int64       `json:"resultTotalSize"`
			Cursor     string      `json:"cursor"`
			Data       []USDCTrade `json:"dataList"`
		} `json:"result"`
		USDCError
	}{}

	params := url.Values{}
	if category == "" {
		return nil, errInvalidCategory
	}
	params.Set("category", category)

	if !symbol.IsEmpty() {
		symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
		if err != nil {
			return resp.Result.Data, err
		}
		params.Set("symbol", symbolValue)
	}

	if limit > 0 && limit <= 500 {
		params.Set("limit", strconv.FormatInt(limit, 10))
	}
	return resp.Result.Data, by.SendHTTPRequest(ctx, exchange.RestUSDCMargined, common.EncodeURLValues(usdcfuturesGetLatestTrades, params), usdcPublicRate, &resp)
}

// PlaceUSDCOrder create new USDC derivatives order.
func (by *Bybit) PlaceUSDCOrder(ctx context.Context, symbol currency.Pair, orderType, orderFilter, side, timeInForce, orderLinkID string, orderPrice, orderQty, takeProfit, stopLoss, tptriggerby, slTriggerBy, triggerPrice, triggerBy float64, reduceOnly, closeOnTrigger, mmp bool) (USDCCreateOrderResp, error) {
	resp := struct {
		Result USDCCreateOrderResp `json:"result"`
		USDCError
	}{}

	req := make(map[string]interface{})
	if symbol.IsEmpty() {
		return USDCCreateOrderResp{}, errSymbolMissing
	}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
	if err != nil {
		return resp.Result, err
	}
	req["symbol"] = symbolValue

	if orderType == "" {
		return USDCCreateOrderResp{}, errInvalidOrderType
	}
	req["orderType"] = orderType

	if orderFilter == "" {
		return USDCCreateOrderResp{}, errInvalidOrderFilter
	}
	req["orderFilter"] = orderFilter

	if side == "" {
		return USDCCreateOrderResp{}, errInvalidSide
	}
	req["side"] = side

	if orderQty == 0 {
		return USDCCreateOrderResp{}, errInvalidQuantity
	}
	req["orderQty"] = strconv.FormatFloat(orderQty, 'f', -1, 64)

	if orderPrice != 0 {
		req["orderPrice"] = strconv.FormatFloat(orderPrice, 'f', -1, 64)
	}

	if timeInForce != "" {
		req["timeInForce"] = timeInForce
	}

	if orderLinkID != "" {
		req["orderLinkId"] = orderLinkID
	}

	if reduceOnly {
		req["reduceOnly"] = true
	} else {
		req["reduceOnly"] = false
	}

	if closeOnTrigger {
		req["closeOnTrigger"] = true
	} else {
		req["closeOnTrigger"] = false
	}

	if mmp {
		req["mmp"] = true
	} else {
		req["mmp"] = false
	}

	if takeProfit != 0 {
		req["takeProfit"] = strconv.FormatFloat(takeProfit, 'f', -1, 64)
	}

	if stopLoss != 0 {
		req["stopLoss"] = strconv.FormatFloat(stopLoss, 'f', -1, 64)
	}

	if tptriggerby != 0 {
		req["tptriggerby"] = tptriggerby
	}

	if slTriggerBy != 0 {
		req["slTriggerBy"] = strconv.FormatFloat(slTriggerBy, 'f', -1, 64)
	}

	if triggerPrice != 0 {
		req["triggerPrice"] = strconv.FormatFloat(triggerPrice, 'f', -1, 64)
	}

	if triggerBy != 0 {
		req["triggerBy"] = triggerBy
	}
	return resp.Result, by.SendUSDCAuthHTTPRequest(ctx, exchange.RestUSDCMargined, http.MethodPost, usdcfuturesPlaceOrder, req, &resp, usdcPlaceOrderRate)
}

// ModifyUSDCOrder modifies USDC derivatives order.
func (by *Bybit) ModifyUSDCOrder(ctx context.Context, symbol currency.Pair, orderFilter, orderID, orderLinkID string, orderPrice, orderQty, takeProfit, stopLoss, tptriggerby, slTriggerBy, triggerPrice float64) (string, error) {
	resp := struct {
		Result struct {
			OrderID       string `json:"orderId"`
			OrderLinkedID string `json:"orderLinkId"`
		} `json:"result"`
		USDCError
	}{}

	req := make(map[string]interface{})
	if symbol.IsEmpty() {
		return resp.Result.OrderID, errSymbolMissing
	}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
	if err != nil {
		return resp.Result.OrderID, err
	}
	req["symbol"] = symbolValue

	if orderFilter == "" {
		return resp.Result.OrderID, errInvalidOrderFilter
	}
	req["orderFilter"] = orderFilter

	if orderID == "" && orderLinkID == "" {
		return resp.Result.OrderID, errOrderOrOrderLinkIDMissing
	}

	if orderID != "" {
		req["orderId"] = orderID
	}

	if orderLinkID != "" {
		req["orderLinkId"] = orderLinkID
	}

	if orderPrice != 0 {
		req["orderPrice"] = strconv.FormatFloat(orderPrice, 'f', -1, 64)
	}

	if orderQty != 0 {
		req["orderQty"] = strconv.FormatFloat(orderQty, 'f', -1, 64)
	}

	if takeProfit != 0 {
		req["takeProfit"] = strconv.FormatFloat(takeProfit, 'f', -1, 64)
	}

	if stopLoss != 0 {
		req["stopLoss"] = strconv.FormatFloat(stopLoss, 'f', -1, 64)
	}

	if tptriggerby != 0 {
		req["tptriggerby"] = strconv.FormatFloat(tptriggerby, 'f', -1, 64)
	}

	if slTriggerBy != 0 {
		req["slTriggerBy"] = strconv.FormatFloat(slTriggerBy, 'f', -1, 64)
	}

	if triggerPrice != 0 {
		req["triggerPrice"] = strconv.FormatFloat(triggerPrice, 'f', -1, 64)
	}
	return resp.Result.OrderID, by.SendUSDCAuthHTTPRequest(ctx, exchange.RestUSDCMargined, http.MethodPost, usdcfuturesModifyOrder, req, &resp, usdcModifyOrderRate)
}

// CancelUSDCOrder cancels USDC derivatives order.
func (by *Bybit) CancelUSDCOrder(ctx context.Context, symbol currency.Pair, orderFilter, orderID, orderLinkID string) (string, error) {
	resp := struct {
		Result struct {
			OrderID string `json:"orderId"`
		} `json:"result"`
		USDCError
	}{}

	req := make(map[string]interface{})
	if symbol.IsEmpty() {
		return resp.Result.OrderID, errSymbolMissing
	}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
	if err != nil {
		return resp.Result.OrderID, err
	}
	req["symbol"] = symbolValue

	if orderFilter == "" {
		return resp.Result.OrderID, errInvalidOrderFilter
	}
	req["orderFilter"] = orderFilter

	if orderID == "" && orderLinkID == "" {
		return resp.Result.OrderID, errOrderOrOrderLinkIDMissing
	}

	if orderID != "" {
		req["orderId"] = orderID
	}

	if orderLinkID != "" {
		req["orderLinkId"] = orderLinkID
	}
	return resp.Result.OrderID, by.SendUSDCAuthHTTPRequest(ctx, exchange.RestUSDCMargined, http.MethodPost, usdcfuturesCancelOrder, req, &resp, usdcCancelOrderRate)
}

// CancelAllActiveUSDCOrder cancels all active USDC derivatives order.
func (by *Bybit) CancelAllActiveUSDCOrder(ctx context.Context, symbol currency.Pair, orderFilter string) error {
	req := make(map[string]interface{})
	if symbol.IsEmpty() {
		return errSymbolMissing
	}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
	if err != nil {
		return err
	}
	req["symbol"] = symbolValue

	if orderFilter == "" {
		return errInvalidOrderFilter
	}
	req["orderFilter"] = orderFilter
	return by.SendUSDCAuthHTTPRequest(ctx, exchange.RestUSDCMargined, http.MethodPost, usdcfuturesCancelAllActiveOrder, req, nil, usdcCancelAllOrderRate)
}

// GetActiveUSDCOrder gets all active USDC derivatives order.
func (by *Bybit) GetActiveUSDCOrder(ctx context.Context, symbol currency.Pair, category, orderID, orderLinkID, orderFilter, direction, cursor string, limit int64) ([]USDCOrder, error) {
	resp := struct {
		Result struct {
			Cursor          string      `json:"cursor"`
			ResultTotalSize int64       `json:"resultTotalSize"`
			Data            []USDCOrder `json:"dataList"`
		} `json:"result"`
		USDCError
	}{}

	req := make(map[string]interface{})
	if !symbol.IsEmpty() {
		symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
		if err != nil {
			return resp.Result.Data, err
		}
		req["symbol"] = symbolValue
	}

	if category == "" {
		return nil, errInvalidCategory
	}
	req["category"] = category

	if orderID != "" {
		req["orderId"] = orderID
	}

	if orderLinkID != "" {
		req["orderLinkId"] = orderLinkID
	}

	if orderFilter != "" {
		req["orderFilter"] = orderFilter
	}

	if direction != "" {
		req["direction"] = direction
	}

	if limit != 0 {
		req["limit"] = strconv.FormatInt(limit, 10)
	}

	if cursor != "" {
		req["cursor"] = cursor
	}
	return resp.Result.Data, by.SendUSDCAuthHTTPRequest(ctx, exchange.RestUSDCMargined, http.MethodPost, usdcfuturesGetActiveOrder, req, &resp, usdcGetOrderRate)
}

// GetUSDCOrderHistory gets order history with support of last 30 days of USDC derivatives order.
func (by *Bybit) GetUSDCOrderHistory(ctx context.Context, symbol currency.Pair, category, orderID, orderLinkID, orderStatus, direction, cursor string, limit int64) ([]USDCOrderHistory, error) {
	resp := struct {
		Result struct {
			Cursor          string             `json:"cursor"`
			ResultTotalSize int64              `json:"resultTotalSize"`
			Data            []USDCOrderHistory `json:"dataList"`
		} `json:"result"`
		USDCError
	}{}

	req := make(map[string]interface{})
	if !symbol.IsEmpty() {
		symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
		if err != nil {
			return resp.Result.Data, err
		}
		req["symbol"] = symbolValue
	}

	if category == "" {
		return nil, errInvalidCategory
	}
	req["category"] = category

	if orderID != "" {
		req["orderId"] = orderID
	}

	if orderLinkID != "" {
		req["orderLinkId"] = orderLinkID
	}

	if orderStatus != "" {
		req["orderStatus"] = orderStatus
	}

	if direction != "" {
		req["direction"] = direction
	}

	if limit != 0 {
		req["limit"] = strconv.FormatInt(limit, 10)
	}

	if cursor != "" {
		req["cursor"] = cursor
	}
	return resp.Result.Data, by.SendUSDCAuthHTTPRequest(ctx, exchange.RestUSDCMargined, http.MethodPost, usdcfuturesGetOrderHistory, req, &resp, usdcGetOrderHistoryRate)
}

// GetUSDCTradeHistory gets trade history with support of last 30 days of USDC derivatives trades.
func (by *Bybit) GetUSDCTradeHistory(ctx context.Context, symbol currency.Pair, category, orderID, orderLinkID, direction, cursor string, limit int64, startTime time.Time) ([]USDCTradeHistory, error) {
	resp := struct {
		Result struct {
			Cursor          string             `json:"cursor"`
			ResultTotalSize int64              `json:"resultTotalSize"`
			Data            []USDCTradeHistory `json:"dataList"`
		} `json:"result"`
		USDCError
	}{}

	req := make(map[string]interface{})
	if !symbol.IsEmpty() {
		symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
		if err != nil {
			return resp.Result.Data, err
		}
		req["symbol"] = symbolValue
	}

	if category == "" {
		return nil, errInvalidCategory
	}
	req["category"] = category

	if orderID == "" && orderLinkID == "" {
		return nil, errOrderOrOrderLinkIDMissing
	}

	if orderID != "" {
		req["orderId"] = orderID
	}

	if orderLinkID != "" {
		req["orderLinkId"] = orderLinkID
	}

	if startTime.IsZero() {
		return nil, errInvalidStartTime
	}
	req["startTime"] = strconv.FormatInt(startTime.Unix(), 10)

	if direction != "" {
		req["direction"] = direction
	}

	if limit > 0 && limit <= 50 {
		req["limit"] = strconv.FormatInt(limit, 10)
	}

	if cursor != "" {
		req["cursor"] = cursor
	}
	return resp.Result.Data, by.SendUSDCAuthHTTPRequest(ctx, exchange.RestUSDCMargined, http.MethodPost, usdcfuturesGetTradeHistory, req, &resp, usdcGetTradeHistoryRate)
}

// GetUSDCTransactionLog gets transaction logs with support of last 30 days of USDC derivatives trades.
func (by *Bybit) GetUSDCTransactionLog(ctx context.Context, startTime, endTime time.Time, txType, category, direction, cursor string, limit int64) ([]USDCTxLog, error) {
	resp := struct {
		Result struct {
			Cursor          string      `json:"cursor"`
			ResultTotalSize int64       `json:"resultTotalSize"`
			Data            []USDCTxLog `json:"dataList"`
		} `json:"result"`
		USDCError
	}{}

	req := make(map[string]interface{})
	if !startTime.IsZero() {
		req["startTime"] = strconv.FormatInt(startTime.Unix(), 10)
	}

	if !endTime.IsZero() {
		req["endTime"] = strconv.FormatInt(endTime.Unix(), 10)
	}

	if txType == "" {
		return nil, errors.New("type missing")
	}
	req["type"] = txType

	if category != "" {
		req["category"] = category
	}

	if direction != "" {
		req["direction"] = direction
	}

	if limit > 0 && limit <= 50 {
		req["limit"] = strconv.FormatInt(limit, 10)
	}

	if cursor != "" {
		req["cursor"] = cursor
	}
	return resp.Result.Data, by.SendUSDCAuthHTTPRequest(ctx, exchange.RestUSDCMargined, http.MethodPost, usdcfuturesGetTransactionLog, req, &resp, usdcGetTransactionRate)
}

// GetUSDCWalletBalance gets USDC wallet balance.
func (by *Bybit) GetUSDCWalletBalance(ctx context.Context) (USDCWalletBalance, error) {
	resp := struct {
		Result USDCWalletBalance `json:"result"`
		USDCError
	}{}

	return resp.Result, by.SendUSDCAuthHTTPRequest(ctx, exchange.RestUSDCMargined, http.MethodPost, usdcfuturesGetWalletBalance, nil, &resp, usdcGetWalletRate)
}

// GetUSDCAssetInfo gets USDC asset information.
func (by *Bybit) GetUSDCAssetInfo(ctx context.Context, baseCoin string) ([]USDCAssetInfo, error) {
	resp := struct {
		Result struct {
			ResultTotalSize int64           `json:"resultTotalSize"`
			Data            []USDCAssetInfo `json:"dataList"`
		} `json:"result"`
		USDCError
	}{}

	req := make(map[string]interface{})
	if baseCoin != "" {
		req["baseCoin"] = baseCoin
	}

	return resp.Result.Data, by.SendUSDCAuthHTTPRequest(ctx, exchange.RestUSDCMargined, http.MethodPost, usdcfuturesGetAssetInfo, req, &resp, usdcGetAssetRate)
}

// GetUSDCMarginInfo gets USDC account margin information.
func (by *Bybit) GetUSDCMarginInfo(ctx context.Context) (string, error) {
	resp := struct {
		Result struct {
			MarginMode string `json:"marginMode"`
		} `json:"result"`
		USDCError
	}{}

	return resp.Result.MarginMode, by.SendUSDCAuthHTTPRequest(ctx, exchange.RestUSDCMargined, http.MethodPost, usdcfuturesGetMarginInfo, nil, &resp, usdcGetMarginRate)
}

// GetUSDCPosition gets USDC position information.
func (by *Bybit) GetUSDCPosition(ctx context.Context, symbol currency.Pair, category, direction, cursor string, limit int64) ([]USDCPosition, error) {
	resp := struct {
		Result struct {
			Cursor          string         `json:"cursor"`
			ResultTotalSize int64          `json:"resultTotalSize"`
			Data            []USDCPosition `json:"dataList"`
		} `json:"result"`
		USDCError
	}{}

	req := make(map[string]interface{})
	if !symbol.IsEmpty() {
		symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
		if err != nil {
			return resp.Result.Data, err
		}
		req["symbol"] = symbolValue
	}

	if category == "" {
		return nil, errInvalidCategory
	}
	req["category"] = category

	if cursor != "" {
		req["cursor"] = cursor
	}

	if direction != "" {
		req["direction"] = direction
	}

	if limit > 0 && limit <= 50 {
		req["limit"] = strconv.FormatInt(limit, 10)
	}

	return resp.Result.Data, by.SendUSDCAuthHTTPRequest(ctx, exchange.RestUSDCMargined, http.MethodPost, usdcfuturesGetPosition, req, &resp, usdcGetPositionRate)
}

// SetUSDCLeverage sets USDC leverage.
func (by *Bybit) SetUSDCLeverage(ctx context.Context, symbol currency.Pair, leverage float64) (float64, error) {
	resp := struct {
		Result struct {
			Leverage convert.StringToFloat64 `json:"leverage"`
		} `json:"result"`
		USDCError
	}{}

	req := make(map[string]interface{})
	if symbol.IsEmpty() {
		return 0, errSymbolMissing
	}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
	if err != nil {
		return 0, err
	}
	req["symbol"] = symbolValue

	if leverage <= 0 {
		return 0, errInvalidLeverage
	}
	req["leverage"] = strconv.FormatFloat(leverage, 'f', -1, 64)

	return resp.Result.Leverage.Float64(),
		by.SendUSDCAuthHTTPRequest(ctx, exchange.RestUSDCMargined, http.MethodPost, usdcfuturesSetLeverage, req, &resp, usdcSetLeverageRate)
}

// GetUSDCSettlementHistory gets USDC settlement history with support of last 30 days.
func (by *Bybit) GetUSDCSettlementHistory(ctx context.Context, symbol currency.Pair, direction, cursor string, limit int64) ([]USDCSettlementHistory, error) {
	resp := struct {
		Result struct {
			Cursor          string                  `json:"cursor"`
			ResultTotalSize int64                   `json:"resultTotalSize"`
			Data            []USDCSettlementHistory `json:"dataList"`
		} `json:"result"`
		USDCError
	}{}

	req := make(map[string]interface{})
	if symbol.IsEmpty() {
		return resp.Result.Data, errSymbolMissing
	}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
	if err != nil {
		return resp.Result.Data, err
	}
	req["symbol"] = symbolValue

	if cursor != "" {
		req["cursor"] = cursor
	}

	if direction != "" {
		req["direction"] = direction
	}

	if limit > 0 && limit <= 50 {
		req["limit"] = strconv.FormatInt(limit, 10)
	}

	return resp.Result.Data, by.SendUSDCAuthHTTPRequest(ctx, exchange.RestUSDCMargined, http.MethodPost, usdcfuturesGetSettlementHistory, req, &resp, usdcGetSettlementRate)
}

// GetUSDCRiskLimit gets USDC risk limits data.
func (by *Bybit) GetUSDCRiskLimit(ctx context.Context, symbol currency.Pair) ([]USDCRiskLimit, error) {
	resp := struct {
		Result []USDCRiskLimit `json:"result"`
		USDCError
	}{}

	params := url.Values{}
	if symbol.IsEmpty() {
		return nil, errSymbolMissing
	}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
	if err != nil {
		return resp.Result, err
	}
	params.Set("symbol", symbolValue)

	return resp.Result, by.SendHTTPRequest(ctx, exchange.RestUSDCMargined, common.EncodeURLValues(usdcfuturesGetRiskLimit, params), usdcPublicRate, &resp)
}

// SetUSDCRiskLimit sets USDC risk limit.
func (by *Bybit) SetUSDCRiskLimit(ctx context.Context, symbol currency.Pair, riskID int64) (string, error) {
	resp := struct {
		Result struct {
			RiskID string `json:"riskId"`
		} `json:"result"`
		USDCError
	}{}

	req := make(map[string]interface{})
	if symbol.IsEmpty() {
		return resp.Result.RiskID, errSymbolMissing
	}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
	if err != nil {
		return resp.Result.RiskID, err
	}
	req["symbol"] = symbolValue

	if riskID <= 0 {
		return resp.Result.RiskID, errInvalidRiskID
	}
	req["riskId"] = riskID

	return resp.Result.RiskID, by.SendUSDCAuthHTTPRequest(ctx, exchange.RestUSDCMargined, http.MethodPost, usdcfuturesSetRiskLimit, req, &resp, usdcSetRiskRate)
}

// GetUSDCLastFundingRate gets USDC last funding rates.
func (by *Bybit) GetUSDCLastFundingRate(ctx context.Context, symbol currency.Pair) (USDCFundingInfo, error) {
	resp := struct {
		Result USDCFundingInfo `json:"result"`
		USDCError
	}{}

	params := url.Values{}
	if symbol.IsEmpty() {
		return resp.Result, errSymbolMissing
	}
	symbolValue, err := by.FormatSymbol(symbol, asset.USDCMarginedFutures)
	if err != nil {
		return resp.Result, err
	}
	params.Set("symbol", symbolValue)

	return resp.Result, by.SendHTTPRequest(ctx, exchange.RestUSDCMargined, common.EncodeURLValues(usdcfuturesGetLastFundingRate, params), usdcPublicRate, &resp)
}

// GetUSDCPredictedFundingRate gets predicted funding rates and my predicted funding fee.
func (by *Bybit) GetUSDCPredictedFundingRate(ctx context.Context, symbol currency.Pair) (predictedFundingRate, predictedFundingFee float64, err error) {
	resp := struct {
		Result struct {
			PredictedFundingRate convert.StringToFloat64 `json:"predictedFundingRate"`
			PredictedFundingFee  convert.StringToFloat64 `json:"predictedFundingFee"`
		} `json:"result"`
		USDCError
	}{}

	req := make(map[string]interface{})
	var symbolValue string
	if symbol.IsEmpty() {
		return 0, 0, errSymbolMissing
	}
	symbolValue, err = by.FormatSymbol(symbol, asset.USDCMarginedFutures)
	if err != nil {
		return 0, 0, err
	}
	req["symbol"] = symbolValue

	err = by.SendUSDCAuthHTTPRequest(ctx, exchange.RestUSDCMargined, http.MethodPost, usdcfuturesGetPredictedFundingRate, req, &resp, usdcGetPredictedFundingRate)
	predictedFundingRate = resp.Result.PredictedFundingRate.Float64()
	predictedFundingFee = resp.Result.PredictedFundingFee.Float64()
	return
}

// SendUSDCAuthHTTPRequest sends an authenticated HTTP request
func (by *Bybit) SendUSDCAuthHTTPRequest(ctx context.Context, ePath exchange.URL, method, path string, data interface{}, result UnmarshalTo, f request.EndpointLimit) error {
	creds, err := by.GetCredentials(ctx)
	if err != nil {
		return err
	}

	if result == nil {
		result = &USDCError{}
	}

	endpointPath, err := by.API.Endpoints.GetURL(ePath)
	if err != nil {
		return err
	}

	err = by.SendPayload(ctx, f, func() (*request.Item, error) {
		nowTimeInMilli := strconv.FormatInt(time.Now().UnixMilli(), 10)
		headers := make(map[string]string)
		var payload, hmacSigned []byte

		if data != nil {
			d, ok := data.(map[string]interface{})
			if !ok {
				return nil, common.GetTypeAssertError("map[string]interface{}", data)
			}
			payload, err = json.Marshal(d)
			if err != nil {
				return nil, err
			}
		}

		signInput := nowTimeInMilli + creds.Key + defaultRecvWindow + string(payload)
		hmacSigned, err = crypto.GetHMAC(crypto.HashSHA256, []byte(signInput), []byte(creds.Secret))
		if err != nil {
			return nil, err
		}

		headers["Content-Type"] = "application/json"
		headers["X-BAPI-API-KEY"] = creds.Key
		headers["X-BAPI-SIGN"] = crypto.HexEncodeToString(hmacSigned)
		headers["X-BAPI-SIGN-TYPE"] = "2"
		headers["X-BAPI-TIMESTAMP"] = nowTimeInMilli
		headers["X-BAPI-RECV-WINDOW"] = defaultRecvWindow

		return &request.Item{
			Method:        method,
			Path:          endpointPath + path,
			Headers:       headers,
			Body:          bytes.NewBuffer(payload),
			Result:        &result,
			Verbose:       by.Verbose,
			HTTPDebugging: by.HTTPDebugging,
			HTTPRecording: by.HTTPRecording}, nil
	}, request.AuthenticatedRequest)
	if err != nil {
		return err
	}
	return result.GetError(true)
}

// USDCError defines all error information for each USDC request
type USDCError struct {
	ReturnCode int64  `json:"retCode"`
	ReturnMsg  string `json:"retMsg"`
}

// GetError checks and returns an error if it is supplied.
func (e USDCError) GetError(isAuthRequest bool) error {
	if e.ReturnCode != 0 && e.ReturnMsg != "" {
		if isAuthRequest {
			return fmt.Errorf("%w %v", request.ErrAuthRequestFailed, e.ReturnMsg)
		}
		return errors.New(e.ReturnMsg)
	}
	return nil
}
