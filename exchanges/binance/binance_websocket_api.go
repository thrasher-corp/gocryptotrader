package binance

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	binanceWebsocketAPIURL = "wss://ws-api.binance.com:443/ws-api/v3"
)

const spotWebsocketAPI = "Spot-Websocket-API"

// websocket request status codes
var websocketStatusCodes = map[int64]string{
	400: "request failed",
	403: "request blocked",
	409: "request partially failed but also partially succeeded",
	418: "auto-banned for repeated violation of rate limits",
	419: "exceeded API request rate limit",
}

// WsConnectAPI creates a new websocket connection to API server
func (e *Exchange) WsConnectAPI(ctx context.Context, conn websocket.Connection) (err error) {
	defer func() {
		if err != nil {
			e.SetIsAPIStreamConnected(false)
			return
		}
		e.SetIsAPIStreamConnected(true)
	}()

	if err := e.CurrencyPairs.IsAssetEnabled(asset.Spot); err != nil {
		return err
	}
	conn.SetURL(binanceWebsocketAPIURL)
	dialer := gws.Dialer{
		HandshakeTimeout: e.Config.HTTPTimeout,
		Proxy:            http.ProxyFromEnvironment,
	}
	if err = conn.Dial(ctx, &dialer, http.Header{}); err != nil {
		return fmt.Errorf("%v - Unable to connect to Websocket. Error: %s", e.Name, err)
	}

	conn.SetupPingHandler(request.UnAuth, websocket.PingHandler{
		UseGorillaHandler: true,
		MessageType:       gws.PongMessage,
		Delay:             pingDelay,
	})

	return nil
}

// IsAPIStreamConnected checks if the API stream connection is established
func (e *Exchange) IsAPIStreamConnected() bool {
	e.isAPIStreamConnectionLock.Lock()
	defer e.isAPIStreamConnectionLock.Unlock()
	return e.isAPIStreamConnected
}

// SetIsAPIStreamConnected sets a value of whether the API stream connection is established
func (e *Exchange) SetIsAPIStreamConnected(isAPIStreamConnected bool) {
	e.isAPIStreamConnectionLock.Lock()
	defer e.isAPIStreamConnectionLock.Unlock()
	e.isAPIStreamConnected = isAPIStreamConnected
}

// wsHandleSpotAPIData routes API response data.
func (e *Exchange) wsHandleSpotAPIData(_ context.Context, respRaw []byte) error {
	result := struct {
		Result json.RawMessage `json:"result"`
		ID     string          `json:"id"`
		Data   json.RawMessage `json:"data"`
	}{}
	err := json.Unmarshal(respRaw, &result)
	if err != nil {
		return err
	}
	if result.ID != "" {
		if !e.Websocket.Match.IncomingWithData(result.ID, respRaw) {
			return errors.New("Unhandled data: " + string(respRaw))
		}
		return nil
	}
	return fmt.Errorf("unhandled stream data %s", string(respRaw))
}

// SendWsRequest sends websocket endpoint request through the websocket connection
func (e *Exchange) SendWsRequest(method string, param, result any) error {
	conn, err := e.Websocket.GetConnection(spotWebsocketAPI)
	if err != nil {
		return err
	}
	input := &struct {
		ID     string `json:"id"`
		Method string `json:"method"`
		Params any    `json:"params"`
	}{
		ID:     e.MessageID(),
		Method: method,
		Params: param,
	}
	respRaw, err := conn.SendMessageReturnResponse(context.Background(), request.UnAuth, input.ID, input)
	if err != nil {
		return err
	}
	resp := &struct {
		ID         string             `json:"id"`
		Status     int64              `json:"status"`
		Result     any                `json:"result"`
		Error      *WebsocketAPIError `json:"error"`
		Ratelimits []RateLimitItem    `json:"ratelimits,omitempty"`
	}{
		Result: result,
	}
	err = json.Unmarshal(respRaw, &resp)
	if err != nil {
		return err
	}
	if resp.Status != 200 {
		if resp.Error != nil {
			return fmt.Errorf("status code: %d error code: %d msg: %s", resp.Status, resp.Error.Code, resp.Error.Message)
		}
		switch resp.Status {
		case 400, 403, 409, 418, 419:
			return fmt.Errorf("status code: %d, msg: %s", resp.Status, websocketStatusCodes[resp.Status])
		default:
			switch {
			case resp.Status >= 500 && resp.Error != nil:
				return fmt.Errorf("error code: %d msg: %s", resp.Error.Code, resp.Error.Message)
			case resp.Status >= 500:
				return fmt.Errorf("status code: %d, msg: internal server error", resp.Status)
			default:
				return fmt.Errorf("status code: %d, msg: request failed", resp.Status)
			}
		}
	}
	return nil
}

// GetWsOrderbook returns full orderbook information
//
// OrderBookDataRequestParams contains the following members
// symbol: string of currency pair
// limit: returned limit amount
func (e *Exchange) GetWsOrderbook(obd *OrderBookDataRequestParams) (*OrderBook, error) {
	if obd == nil || *obd == (OrderBookDataRequestParams{}) {
		return nil, common.ErrEmptyParams
	}
	if err := e.CheckLimit(obd.Limit); err != nil {
		return nil, err
	}
	var resp *OrderBook
	return resp, e.SendWsRequest("depth", obd, &resp)
}

// GetWsMostRecentTrades returns recent trade activity through the websocket connection
// limit: Up to 500 results returned
func (e *Exchange) GetWsMostRecentTrades(rtr *RecentTradeRequestParams) ([]*RecentTrade, error) {
	if rtr == nil || *rtr == (RecentTradeRequestParams{}) {
		return nil, common.ErrEmptyParams
	}
	if rtr.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	var resp []*RecentTrade
	return resp, e.SendWsRequest("trades.recent", rtr, &resp)
}

// GetWsAggregatedTrades retrieves aggregated trade activity.
func (e *Exchange) GetWsAggregatedTrades(arg *WsAggregateTradeRequestParams) ([]*AggregatedTrade, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	var resp []*AggregatedTrade
	return resp, e.SendWsRequest("trades.aggregate", arg, &resp)
}

// GetWsCandlestick retrieves spot kline data through the websocket connection.
func (e *Exchange) GetWsCandlestick(arg *KlinesRequestParams) ([]*CandleStick, error) {
	return e.getWsKlines("klines", arg)
}

// GetWsOptimizedCandlestick retrieves spot candlestick bars through the websocket connection.
func (e *Exchange) GetWsOptimizedCandlestick(arg *KlinesRequestParams) ([]*CandleStick, error) {
	return e.getWsKlines("uiKlines", arg)
}

// getWsKlines retrieves spot kline data through the websocket connection.
func (e *Exchange) getWsKlines(method string, arg *KlinesRequestParams) ([]*CandleStick, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Interval == "" {
		return nil, kline.ErrInvalidInterval
	}
	if !arg.StartTime.IsZero() && !arg.EndTime.IsZero() {
		if err := common.StartEndTimeCheck(arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
	}
	if !arg.StartTime.IsZero() {
		arg.StartTimestamp = arg.StartTime.UnixMilli()
	}
	if !arg.EndTime.IsZero() {
		arg.EndTimestamp = arg.EndTime.UnixMilli()
	}
	var resp []*CandleStick
	return resp, e.SendWsRequest(method, arg, &resp)
}

// GetWsCurrenctAveragePrice retrieves current average price for a symbol.
func (e *Exchange) GetWsCurrenctAveragePrice(symbol currency.Pair) (*SymbolAveragePrice, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	arg := &struct {
		Symbol currency.Pair `json:"symbol"`
	}{
		Symbol: symbol,
	}
	var resp *SymbolAveragePrice
	return resp, e.SendWsRequest("avgPrice", arg, &resp)
}

// GetWs24HourPriceChanges 24-hour rolling window price changes statistics through the websocket stream.
// 'type': 'FULL' (default) or 'MINI'
// 'timeZone' Default: 0 (UTC)
func (e *Exchange) GetWs24HourPriceChanges(arg *PriceChangeRequestParam) ([]*PriceChangeStats, error) {
	return e.tickerDataChange("ticker.24hr", arg)
}

// GetWsTradingDayTickers price change statistics for a trading day.
// 'type': 'FULL' (default) or 'MINI'
// 'timeZone' Default: 0 (UTC)
func (e *Exchange) GetWsTradingDayTickers(arg *PriceChangeRequestParam) ([]*PriceChangeStats, error) {
	return e.tickerDataChange("ticker.tradingDay", arg)
}

// tickerDataChange unifying method to make price change requests through the websocket stream.
func (e *Exchange) tickerDataChange(method string, arg *PriceChangeRequestParam) ([]*PriceChangeStats, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol == "" && len(arg.Symbols) == 0 {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	var resp PriceChanges
	return resp, e.SendWsRequest(method, arg, &resp)
}

// WindowSizeToString converts a duration instance and returns a string.
func (e *Exchange) WindowSizeToString(windowSize time.Duration) string {
	switch {
	case windowSize/(time.Hour*24) > 0:
		return strconv.FormatInt(int64(windowSize/(time.Hour*24)), 10) + "d"
	case (windowSize / time.Hour) > 0:
		return strconv.FormatInt(int64(windowSize/time.Hour), 10) + "h"
	case (windowSize / time.Minute) > 0:
		return strconv.FormatInt(int64((windowSize/time.Minute)), 10) + "m"
	}
	return ""
}

// GetSymbolPriceTicker represents a symbol ticker item information.
func (e *Exchange) GetSymbolPriceTicker(symbol currency.Pair) ([]*SymbolTickerItem, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	var resp SymbolTickers
	return resp, e.SendWsRequest("ticker.price", map[string]string{"symbol": symbol.String()}, &resp)
}

// GetWsRollingWindowPriceChanges retrieves rolling window price change statistics with a custom window.
// this request is similar to ticker.24hr, but statistics are computed on demand using the arbitrary window you specify
func (e *Exchange) GetWsRollingWindowPriceChanges(arg *WsRollingWindowPriceParams) ([]*PriceChangeStats, error) {
	if arg.Symbol == "" && len(arg.Symbols) == 0 {
		return nil, currency.ErrCurrencyPairEmpty
	}
	arg.WindowSize = e.WindowSizeToString(arg.WindowSizeDuration)
	var resp PriceChanges
	return resp, e.SendWsRequest("ticker", arg, &resp)
}

// GetWsSymbolOrderbookTicker retrieves the current best price and quantity on the order book.
func (e *Exchange) GetWsSymbolOrderbookTicker(symbols currency.Pairs) ([]*WsOrderbookTicker, error) {
	if len(symbols) == 0 || (len(symbols) == 1 && symbols[0].IsEmpty()) {
		return nil, currency.ErrCurrencyPairsEmpty
	}
	var (
		symbolString  string
		symbolsString []string
	)
	if len(symbols) > 1 {
		symbolsString = symbols.Strings()
	} else {
		symbolString = symbols[0].String()
	}
	arg := &struct {
		Symbol  string   `json:"symbol,omitempty"`
		Symbols []string `json:"symbols,omitempty"`
	}{
		Symbols: symbolsString,
		Symbol:  symbolString,
	}

	var resp WsOrderbookTickers
	return resp, e.SendWsRequest("ticker.book", arg, &resp)
}

func (e *Exchange) getSignature(arg any) (apiKey, signature string, err error) {
	mapValue, err := e.ToMap(arg)
	if err != nil {
		return apiKey, signature, err
	}
	return e.SignRequest(mapValue)
}

// SignRequest creates a signature given params map
func (e *Exchange) SignRequest(params map[string]any) (apiKey, signature string, err error) {
	creds, err := e.GetCredentials(context.Background())
	if err != nil {
		return "", "", err
	}
	timestampInfo, okay := params["timestamp"]
	if !okay {
		return "", "", errTimestampInfoRequired
	}
	timestampType := fmt.Sprintf("%T", timestampInfo)
	switch timestampType {
	case "float64", "int64", "float32", "int":
	default:
		return "", "", fmt.Errorf("invalid timestamp: %s %w", timestampType, errTimestampInfoRequired)
	}
	params["apiKey"] = creds.Key
	keys := SortMap(params)
	payloadString := fmt.Sprintf("%s=%v", keys[0], params[keys[0]])
	for i := 1; i < len(keys); i++ {
		payloadString += fmt.Sprintf("&%s=%v", keys[i], params[keys[i]])
	}
	var hmacSigned []byte
	hmacSigned, err = crypto.GetHMAC(crypto.HashSHA256,
		[]byte(payloadString),
		[]byte(creds.Secret))
	if err != nil {
		return "", "", err
	}
	return creds.Key, hex.EncodeToString(hmacSigned), nil
}

// SortMap gives a slice of sorted keys from the passed map
func SortMap(params map[string]any) []string {
	keys := make([]string, 0, len(params))
	for a := range params {
		count := 0
		added := false
		for count < len(keys) {
			if keys[count] >= a {
				keys = append(keys[:count], append([]string{a}, keys[count:]...)...)
				added = true
				break
			}
			count++
		}
		if !added {
			keys = append(keys, a)
		}
	}
	return keys
}

// GetQuerySessionStatus query the status of the WebSocket connection, inspecting which API key (if any) is used to authorize requests.
func (e *Exchange) GetQuerySessionStatus() (*FuturesAuthenticationResp, error) {
	var resp FuturesAuthenticationResp
	return &resp, e.SendWsRequest("session.status", nil, &resp)
}

// GetLogOutOfSession forget the API key previously authenticated. If the connection is not authenticated, this request does nothing.
func (e *Exchange) GetLogOutOfSession() (*FuturesAuthenticationResp, error) {
	var resp FuturesAuthenticationResp
	return &resp, e.SendWsRequest("session.logout", nil, &resp)
}

// ----------------------------------------------------------- Trading Requests ----------------------------------------------------

// WsPlaceNewOrder place new order
func (e *Exchange) WsPlaceNewOrder(arg *TradeOrderRequestParam) (*TradeOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	arg.Timestamp = time.Now().UnixMilli()
	apiKey, signature, err := e.getSignature(arg)
	if err != nil {
		return nil, err
	}
	arg.APIKey = apiKey
	arg.Signature = signature
	var resp TradeOrderResponse
	return &resp, e.SendWsRequest("order.place", arg, &resp)
}

// ValidatePlaceNewOrderRequest tests whether the request order is valid or not.
func (e *Exchange) ValidatePlaceNewOrderRequest(arg *TradeOrderRequestParam) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.Symbol.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return order.ErrTypeIsInvalid
	}
	arg.Timestamp = time.Now().UnixMilli()
	apiKey, signature, err := e.getSignature(arg)
	if err != nil {
		return err
	}
	arg.APIKey = apiKey
	arg.Signature = signature
	return e.SendWsRequest("order.test", arg, &struct{}{})
}

// WsQueryOrder to query a trade order
func (e *Exchange) WsQueryOrder(arg *QueryOrderParam) (*TradeOrder, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.OrderID == 0 && arg.OrigClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	arg.Timestamp = time.Now().UnixMilli()
	apiKey, signature, err := e.getSignature(arg)
	if err != nil {
		return nil, err
	}
	arg.APIKey = apiKey
	arg.Signature = signature
	var resp *TradeOrder
	return resp, e.SendWsRequest("order.status", arg, &resp)
}

// WsCancelOrder cancel an active order.
func (e *Exchange) WsCancelOrder(arg *QueryOrderParam) (*TradeOrder, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.OrderID == 0 && arg.OrigClientOrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	arg.Timestamp = time.Now().UnixMilli()
	apiKey, signature, err := e.getSignature(arg)
	if err != nil {
		return nil, err
	}
	arg.APIKey = apiKey
	arg.Signature = signature
	var resp *TradeOrder
	return resp, e.SendWsRequest("order.cancel", &arg, &resp)
}

// WsCancelAndReplaceTradeOrder cancel an existing order and immediately place a new order instead of the canceled one.
func (e *Exchange) WsCancelAndReplaceTradeOrder(arg *WsCancelAndReplaceParam) (*WsCancelAndReplaceTradeOrderResponse, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.CancelReplaceMode == "" {
		return nil, errors.New("cancel replace mode is required")
	}
	if arg.CancelOrderID == "" {
		return nil, fmt.Errorf("cancelOrderId missing, %w", order.ErrOrderIDNotSet)
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	arg.Timestamp = time.Now().UnixMilli()
	apiKey, signature, err := e.getSignature(arg)
	if err != nil {
		return nil, err
	}
	arg.APIKey = apiKey
	arg.Signature = signature
	var resp *WsCancelAndReplaceTradeOrderResponse
	return resp, e.SendWsRequest("order.cancelReplace", &arg, &resp)
}

func (e *Exchange) openOrdersFilter(symbol currency.Pair, recvWindow int64) (map[string]any, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	arg := make(map[string]any)
	if recvWindow != 0 {
		arg["recvWindow"] = recvWindow
	}
	arg["symbol"] = symbol
	arg["timestamp"] = time.Now().UnixMilli()
	apiKey, signature, err := e.SignRequest(arg)
	if err != nil {
		return nil, err
	}
	arg["apiKey"] = apiKey
	arg["signature"] = signature
	return arg, nil
}

// WsCurrentOpenOrders retrieves list of open orders.
func (e *Exchange) WsCurrentOpenOrders(symbol currency.Pair, recvWindow int64) ([]*TradeOrder, error) {
	arg, err := e.openOrdersFilter(symbol, recvWindow)
	if err != nil {
		return nil, err
	}
	arg["timestamp"] = time.Now().UnixMilli()
	apiKey, signature, err := e.getSignature(arg)
	if err != nil {
		return nil, err
	}
	arg["apiKey"] = apiKey
	arg["signature"] = signature
	var resp []*TradeOrder
	return resp, e.SendWsRequest("openOrders.status", arg, &resp)
}

// WsCancelOpenOrders represents an open orders list
func (e *Exchange) WsCancelOpenOrders(symbol currency.Pair, recvWindow int64) ([]*WsCancelOrder, error) {
	arg, err := e.openOrdersFilter(symbol, recvWindow)
	if err != nil {
		return nil, err
	}
	arg["timestamp"] = time.Now().UnixMilli()
	apiKey, signature, err := e.getSignature(arg)
	if err != nil {
		return nil, err
	}
	arg["apiKey"] = apiKey
	arg["signature"] = signature
	var resp []*WsCancelOrder
	return resp, e.SendWsRequest("openOrders.cancelAll", arg, &resp)
}

// WsPlaceOCOOrder send in a new one-cancels-the-other (OCO) pair: LIMIT_MAKER + STOP_LOSS/STOP_LOSS_LIMIT orders (called legs), where activation of one order immediately cancels the other.
// Response format for orderReports is selected using the newOrderRespType parameter. The following example is for RESULT response type. See order.place for more examples.
func (e *Exchange) WsPlaceOCOOrder(arg *PlaceOCOOrderParam) (*OCOOrder, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.Quantity <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	if arg.StopPrice <= 0 {
		return nil, fmt.Errorf("stopPrice: %w", limits.ErrPriceBelowMin)
	}
	if arg.TrailingDelta <= 0 {
		return nil, errors.New("invalid trailingDelta value")
	}
	arg.Timestamp = time.Now().UnixMilli()
	apiKey, signature, err := e.getSignature(arg)
	if err != nil {
		return nil, err
	}
	arg.APIKey = apiKey
	arg.Signature = signature
	var resp *OCOOrder
	return resp, e.SendWsRequest("orderList.place", arg, &resp)
}

// WsQueryOCOOrder execution status of an OCO.
func (e *Exchange) WsQueryOCOOrder(origClientOrderID string, orderListID, recvWindow int64) (*OCOOrderInfo, error) {
	if origClientOrderID == "" {
		return nil, fmt.Errorf("origClientOrderID %w", order.ErrOrderIDNotSet)
	}
	params := map[string]any{
		"origClientOrderId": origClientOrderID,
	}
	if orderListID != 0 {
		params["orderListId"] = orderListID
	}
	if recvWindow != 0 {
		params["recvWindow"] = recvWindow
	}
	params["timestamp"] = time.Now().UnixMilli()
	apiKey, signature, err := e.SignRequest(params)
	if err != nil {
		return nil, err
	}
	params["apiKey"] = apiKey
	params["signature"] = signature
	var resp *OCOOrderInfo
	return resp, e.SendWsRequest("orderList.status", params, &resp)
}

// WsCancelOCOOrder cancel an active OCO order.
func (e *Exchange) WsCancelOCOOrder(symbol currency.Pair, orderListID, listClientOrderID, newClientOrderID string) (*OCOOrder, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if orderListID == "" {
		return nil, fmt.Errorf("orderListID %w", order.ErrOrderIDNotSet)
	}
	params := make(map[string]any)
	if listClientOrderID == "" {
		params["listClientOrderId"] = listClientOrderID
	}
	if newClientOrderID == "" {
		params["newClientOrderId"] = newClientOrderID
	}
	params["orderListId"] = orderListID
	params["timestamp"] = time.Now().UnixMilli()
	apiKey, signature, err := e.SignRequest(params)
	if err != nil {
		return nil, err
	}
	params["apiKey"] = apiKey
	params["signature"] = signature
	var resp *OCOOrder
	return resp, e.SendWsRequest("orderList.cancel", params, &resp)
}

// WsCurrentOpenOCOOrders query execution status of all open OCOs.
func (e *Exchange) WsCurrentOpenOCOOrders(recvWindow int64) ([]*OCOOrder, error) {
	params := make(map[string]any)
	if recvWindow != 0 {
		params["recvWindow"] = recvWindow
	}
	params["timestamp"] = time.Now().UnixMilli()
	apiKey, signature, err := e.SignRequest(params)
	if err != nil {
		return nil, err
	}
	params["apiKey"] = apiKey
	params["signature"] = signature
	var resp []*OCOOrder
	return resp, e.SendWsRequest("openOrderLists.status", params, &resp)
}

// WsPlaceNewSOROrder places an order using smart order routing (SOR).
func (e *Exchange) WsPlaceNewSOROrder(arg *WsOSRPlaceOrderParams) ([]*OSROrder, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return nil, order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return nil, order.ErrTypeIsInvalid
	}
	if arg.Quantity <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	arg.Timestamp = time.Now().UnixMilli()
	apiKey, signature, err := e.getSignature(arg)
	if err != nil {
		return nil, err
	}
	arg.APIKey = apiKey
	arg.Signature = signature
	var resp []*OSROrder
	return resp, e.SendWsRequest("sor.order.place", arg, &resp)
}

// WsTestNewOrderUsingSOR test new order creation and signature/recvWindow using smart order routing (SOR).
// Creates and validates a new order but does not send it into the matching engine.
func (e *Exchange) WsTestNewOrderUsingSOR(arg *WsOSRPlaceOrderParams) error {
	if err := common.NilGuard(arg); err != nil {
		return err
	}
	if arg.Symbol.IsEmpty() {
		return currency.ErrCurrencyPairEmpty
	}
	if arg.Side == "" {
		return order.ErrSideIsInvalid
	}
	if arg.OrderType == "" {
		return order.ErrTypeIsInvalid
	}
	if arg.Quantity <= 0 {
		return limits.ErrAmountBelowMin
	}
	arg.Timestamp = time.Now().UnixMilli()
	apiKey, signature, err := e.getSignature(arg)
	if err != nil {
		return err
	}
	arg.APIKey = apiKey
	arg.Signature = signature
	return e.SendWsRequest("sor.order.place", arg, &struct{}{})
}

// ToMap creates a map out of struct instances
func (e *Exchange) ToMap(input any) (map[string]any, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	var resp map[string]any
	return resp, json.Unmarshal(data, &resp)
}

// ------------------------------------------- Account Requests --------------------------------

// GetWsAccountInfo query information about your account.
func (e *Exchange) GetWsAccountInfo(recvWindow int64) (*Account, error) {
	params := map[string]any{}
	if recvWindow != 0 {
		params["recvWindow"] = recvWindow
	}
	params["timestamp"] = time.Now().UnixMilli()
	apiKey, signatures, err := e.SignRequest(params)
	if err != nil {
		return nil, err
	}
	params["apiKey"] = apiKey
	params["signature"] = signatures
	var resp *Account
	return resp, e.SendWsRequest("account.status", params, &resp)
}

// WsQueryAccountOrderRateLimits query your current order rate limit.
func (e *Exchange) WsQueryAccountOrderRateLimits(recvWindow int64) ([]*RateLimitItem, error) {
	params := map[string]any{}
	if recvWindow > 0 {
		params["recvWindow"] = recvWindow
	}
	params["timestamp"] = time.Now().UnixMilli()
	apiKey, signature, err := e.SignRequest(params)
	if err != nil {
		return nil, err
	}
	params["apiKey"] = apiKey
	params["signature"] = signature
	var resp []*RateLimitItem
	return resp, e.SendWsRequest("account.rateLimits.orders", params, &resp)
}

// WsQueryAccountOrderHistory query information about all your orders – active, canceled, filled – filtered by time range.
// Status reports for orders are identical to order.status.
func (e *Exchange) WsQueryAccountOrderHistory(arg *AccountOrderRequestParam) ([]*TradeOrder, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	arg.Timestamp = time.Now().UnixMilli()
	apiKey, signature, err := e.getSignature(arg)
	if err != nil {
		return nil, err
	}
	arg.APIKey = apiKey
	arg.Signature = signature
	var resp []*TradeOrder
	return resp, e.SendWsRequest("allOrders", arg, &resp)
}

// WsQueryAccountOCOOrderHistory query information about all your OCOs, filtered by time range.
// Status reports for OCOs are identical to orderList.status.
func (e *Exchange) WsQueryAccountOCOOrderHistory(fromID, limit, recvWindow int64, startTime, endTime time.Time) ([]*OCOOrder, error) {
	params := make(map[string]any)
	if fromID != 0 {
		params["fromId"] = fromID
	}
	if limit != 0 {
		params["limit"] = limit
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		if err := common.StartEndTimeCheck(startTime, endTime); err != nil {
			return nil, err
		}
	}
	if !startTime.IsZero() {
		params["startTime"] = startTime.UnixMilli()
	}
	if !endTime.IsZero() {
		params["endTime"] = endTime.UnixMilli()
	}
	if recvWindow != 0 {
		params["recvWindow"] = recvWindow
	}
	params["timestamp"] = time.Now().UnixMilli()
	apiKey, signature, err := e.SignRequest(params)
	if err != nil {
		return nil, err
	}
	params["apiKey"] = apiKey
	params["signature"] = signature
	var resp []*OCOOrder
	return resp, e.SendWsRequest("allOrderLists", params, &resp)
}

// WsAccountTradeHistory query information about all your trades, filtered by time range.
func (e *Exchange) WsAccountTradeHistory(arg *AccountOrderRequestParam) ([]*TradeHistory, error) {
	if err := common.NilGuard(arg); err != nil {
		return nil, err
	}
	if arg.Symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	arg.Timestamp = time.Now().UnixMilli()
	apiKey, signatures, err := e.getSignature(arg)
	if err != nil {
		return nil, err
	}
	arg.APIKey = apiKey
	arg.Signature = signatures
	var resp []*TradeHistory
	return resp, e.SendWsRequest("myTrades", arg, &resp)
}

// WsAccountPreventedMatches displays the list of orders that were expired because of STP trigger.
//
// These are the combinations supported:
// symbol + preventedMatchId
// symbol + orderId
// symbol + orderId + fromPreventedMatchId (limit will default to 500)
// symbol + orderId + fromPreventedMatchId + limit
func (e *Exchange) WsAccountPreventedMatches(symbol currency.Pair, preventedMatchID, orderID, fromPreventedMatchID, limit, recvWindow int64) ([]*SelfTradePrevention, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	if orderID == 0 && preventedMatchID == 0 {
		return nil, fmt.Errorf("%w, either orderID or preventedMatchID is required", order.ErrOrderIDNotSet)
	}
	params := make(map[string]any)
	params["symbol"] = symbol
	if preventedMatchID != 0 {
		params["preventedMatchId"] = preventedMatchID
	}
	if orderID != 0 {
		params["orderId"] = orderID
	}
	if fromPreventedMatchID != 0 {
		params["fromPreventedMatchId"] = fromPreventedMatchID
	}
	if limit > 0 {
		params["limit"] = limit
	}
	if recvWindow > 0 {
		params["recvWindow"] = recvWindow
	}
	params["timestamp"] = time.Now().UnixMilli()
	apiKey, signature, err := e.SignRequest(params)
	if err != nil {
		return nil, err
	}
	params["apiKey"] = apiKey
	params["signature"] = signature
	var resp []*SelfTradePrevention
	return resp, e.SendWsRequest("myPreventedMatches", params, &resp)
}

// WsAccountAllocation retrieves allocations resulting from SOR order placement.
func (e *Exchange) WsAccountAllocation(symbol currency.Pair, startTime, endTime time.Time, orderID, fromAllocationID, recvWindow, limit int64) ([]*SORReplacements, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := map[string]any{
		"symbol": symbol.String(),
	}
	switch {
	case !startTime.IsZero() && !endTime.IsZero():
		if common.StartEndTimeCheck(startTime, endTime) == nil {
			params["startTime"] = startTime.UnixMilli()
			params["endTime"] = endTime.UnixMilli()
		}
	case !startTime.IsZero():
		params["startTime"] = startTime.UnixMilli()
	case !endTime.IsZero():
		params["endTime"] = endTime.UnixMilli()
	}
	if fromAllocationID != 0 {
		params["fromAllocationId"] = fromAllocationID
	}
	if limit > 0 {
		params["limit"] = limit
	}
	if orderID > 0 {
		params["orderId"] = orderID
	}
	if recvWindow > 0 {
		params["recvWindow"] = recvWindow
	}
	params["timestamp"] = time.Now().UnixMilli()
	apiKey, signature, err := e.SignRequest(params)
	if err != nil {
		return nil, err
	}
	params["apiKey"] = apiKey
	params["signature"] = signature
	var resp []*SORReplacements
	return resp, e.SendWsRequest("myAllocations", params, &resp)
}

// WsAccountCommissionRates get current account commission rates.
func (e *Exchange) WsAccountCommissionRates(symbol currency.Pair) (*CommissionRateInto, error) {
	if symbol.IsEmpty() {
		return nil, currency.ErrCurrencyPairEmpty
	}
	params := map[string]any{
		"symbol": symbol.String(),
	}
	params["timestamp"] = time.Now().UnixMilli()
	apiKey, signature, err := e.SignRequest(params)
	if err != nil {
		return nil, err
	}
	params["apiKey"] = apiKey
	params["signature"] = signature
	var resp *CommissionRateInto
	return resp, e.SendWsRequest("account.commission", params, &resp)
}

// --------------------------------- User Data Stream requests ---------------------------------

// WsStartUserDataStream start a new user data stream.
// The response will output a listen key that can be subscribed through on the Websocket stream afterwards.
func (e *Exchange) WsStartUserDataStream() (string, error) {
	creds, err := e.GetCredentials(context.Background())
	if err != nil {
		return "", err
	}
	params := map[string]any{
		"apiKey": creds.Key,
	}
	resp := &struct {
		ListenKey string `json:"listenKey,omitempty"`
	}{}
	return resp.ListenKey, e.SendWsRequest("userDataStream.start", params, &resp)
}

// WsPingUserDataStream ping a user data stream to keep it alive.
// User data streams close automatically after 60 minutes, even if you're listening to them on WebSocket Streams.
// In order to keep the stream open, you have to regularly send pings using the userDataStream.ping request.
// It is recommended to send a ping once every 30 minutes.
func (e *Exchange) WsPingUserDataStream(listenKey string) error {
	if listenKey == "" {
		return errListenKeyIsRequired
	}
	creds, err := e.GetCredentials(context.Background())
	if err != nil {
		return err
	}
	params := map[string]any{
		"apiKey":    creds.Key,
		"listenKey": listenKey,
	}
	return e.SendWsRequest("userDataStream.ping", params, struct{}{})
}

// WsStopUserDataStream explicitly stop and close the user data stream.
func (e *Exchange) WsStopUserDataStream(listenKey string) error {
	if listenKey == "" {
		return errListenKeyIsRequired
	}
	creds, err := e.GetCredentials(context.Background())
	if err != nil {
		return err
	}
	params := map[string]any{
		"apiKey":    creds.Key,
		"listenKey": listenKey,
	}
	return e.SendWsRequest("userDataStream.stop", params, struct{}{})
}
