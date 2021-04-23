package coinbene

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Coinbene is the overarching type across this package
type Coinbene struct {
	exchange.Base
}

const (
	coinbeneAPIURL       = "https://openapi-exchange.coinbene.com/api/exchange/"
	coinbeneSwapAPIURL   = "https://openapi-contract.coinbene.com/api/usdt/"
	coinbeneAuthPath     = "/api/exchange/v2"
	coinbeneSwapAuthPath = "/api/usdt/v2"
	coinbeneAPIVersion   = "v2"

	// Public endpoints
	coinbeneGetTicker      = "/market/ticker/one"
	coinbeneGetTickersSpot = "/market/ticker/list"
	coinbeneGetTickers     = "/market/tickers"
	coinbeneGetOrderBook   = "/market/orderBook"
	coinbeneGetKlines      = "/market/klines"
	coinbeneGetInstruments = "/market/instruments"
	// TODO: Implement function ---
	coinbeneSpotKlines       = "/market/instruments/candles"
	coinbeneSpotExchangeRate = "/market/rate/list"
	// ---
	coinbeneGetTrades   = "/market/trades"
	coinbeneGetAllPairs = "/market/tradePair/list"
	coinbenePairInfo    = "/market/tradePair/one"

	// Authenticated endpoints
	coinbeneAccountInfo        = "/account/info"
	coinbeneGetUserBalance     = "/account/list"
	coinbeneAccountBalanceOne  = "/account/one"
	coinbenePlaceOrder         = "/order/place"
	coinbeneBatchPlaceOrder    = "/order/batchPlaceOrder"
	coinbeneTradeFills         = "/order/trade/fills"
	coinbeneOrderFills         = "/order/fills"
	coinbeneOrderInfo          = "/order/info"
	coinbeneCancelOrder        = "/order/cancel"
	coinbeneBatchCancel        = "/order/batchCancel"
	coinbeneOpenOrders         = "/order/openOrders"
	coinbeneOpenOrdersByPage   = "/order/openOrdersByPage"
	coinbeneClosedOrders       = "/order/closedOrders"
	coinbeneClosedOrdersByPage = "/order/closedOrdersByPage"
	coinbeneListSwapPositions  = "/position/list"
	coinbenePositionFeeRate    = "/position/feeRate"

	limitOrder      = "1"
	marketOrder     = "2"
	postOnlyOrder   = "8"
	fillOrKillOrder = "9"
	iosOrder        = "10"
	buyDirection    = "1"
	openLong        = "openLong"
	openShort       = "openShort"
	sellDirection   = "2"
)

// GetAllPairs gets all pairs on the exchange
func (c *Coinbene) GetAllPairs() ([]PairData, error) {
	resp := struct {
		Data []PairData `json:"data"`
	}{}
	path := coinbeneAPIVersion + coinbeneGetAllPairs
	return resp.Data, c.SendHTTPRequest(exchange.RestSpot, path, spotPairs, &resp)
}

// GetPairInfo gets info about a single pair
func (c *Coinbene) GetPairInfo(symbol string) (PairData, error) {
	resp := struct {
		Data PairData `json:"data"`
	}{}
	params := url.Values{}
	params.Set("symbol", symbol)
	path := common.EncodeURLValues(coinbeneAPIVersion+coinbenePairInfo, params)
	return resp.Data, c.SendHTTPRequest(exchange.RestSpot, path, spotPairInfo, &resp)
}

// GetOrderbook gets and stores orderbook data for given pair
func (c *Coinbene) GetOrderbook(symbol string, size int64) (Orderbook, error) {
	resp := struct {
		Data struct {
			Asks [][]string `json:"asks"`
			Bids [][]string `json:"bids"`
			Time time.Time  `json:"timestamp"`
		} `json:"data"`
	}{}

	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("depth", strconv.FormatInt(size, 10))
	path := common.EncodeURLValues(coinbeneAPIVersion+coinbeneGetOrderBook, params)
	err := c.SendHTTPRequest(exchange.RestSpot, path, spotOrderbook, &resp)
	if err != nil {
		return Orderbook{}, err
	}

	processOB := func(ob [][]string) ([]OrderbookItem, error) {
		var o []OrderbookItem
		for x := range ob {
			var price, amount float64
			amount, err = strconv.ParseFloat(ob[x][1], 64)
			if err != nil {
				return nil, err
			}
			price, err = strconv.ParseFloat(ob[x][0], 64)
			if err != nil {
				return nil, err
			}
			o = append(o, OrderbookItem{
				Price:  price,
				Amount: amount,
			})
		}
		return o, nil
	}

	var s Orderbook
	s.Bids, err = processOB(resp.Data.Bids)
	if err != nil {
		return s, err
	}
	s.Asks, err = processOB(resp.Data.Asks)
	if err != nil {
		return s, err
	}
	s.Time = resp.Data.Time
	return s, nil
}

// GetTicker gets and stores ticker data for a currency pair
func (c *Coinbene) GetTicker(symbol string) (TickerData, error) {
	resp := struct {
		TickerData TickerData `json:"data"`
	}{}
	params := url.Values{}
	params.Set("symbol", symbol)
	path := common.EncodeURLValues(coinbeneAPIVersion+coinbeneGetTicker, params)
	return resp.TickerData, c.SendHTTPRequest(exchange.RestSpot, path, spotSpecificTicker, &resp)
}

// GetTickers gets and all spot tickers supported by the exchange
func (c *Coinbene) GetTickers() ([]TickerData, error) {
	resp := struct {
		TickerData []TickerData `json:"data"`
	}{}

	path := coinbeneAPIVersion + coinbeneGetTickersSpot
	return resp.TickerData, c.SendHTTPRequest(exchange.RestSpot, path, spotTickerList, &resp)
}

// GetTrades gets recent trades from the exchange
func (c *Coinbene) GetTrades(symbol string, limit int64) (Trades, error) {
	resp := struct {
		Data [][]string `json:"data"`
	}{}

	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("limit", strconv.FormatInt(limit, 10))
	path := common.EncodeURLValues(coinbeneAPIVersion+coinbeneGetTrades, params)
	err := c.SendHTTPRequest(exchange.RestSpot, path, spotMarketTrades, &resp)
	if err != nil {
		return nil, err
	}

	var trades Trades
	for x := range resp.Data {
		tm, err := time.Parse(time.RFC3339, resp.Data[x][4])
		if err != nil {
			return nil, err
		}
		price, err := strconv.ParseFloat(resp.Data[x][1], 64)
		if err != nil {
			return nil, err
		}
		volume, err := strconv.ParseFloat(resp.Data[x][2], 64)
		if err != nil {
			return nil, err
		}
		trades = append(trades, TradeItem{
			CurrencyPair: resp.Data[x][0],
			Price:        price,
			Volume:       volume,
			Direction:    resp.Data[x][3],
			TradeTime:    tm,
		})
	}
	return trades, nil
}

// GetAccountBalances gets user balanace info
func (c *Coinbene) GetAccountBalances() ([]UserBalanceData, error) {
	resp := struct {
		Data []UserBalanceData `json:"data"`
	}{}
	path := coinbeneAPIVersion + coinbeneGetUserBalance
	err := c.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet,
		path,
		coinbeneGetUserBalance,
		false,
		nil,
		&resp,
		spotAccountInfo)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetAccountAssetBalance gets user balanace info
func (c *Coinbene) GetAccountAssetBalance(symbol string) (UserBalanceData, error) {
	v := url.Values{}
	v.Set("asset", symbol)
	resp := struct {
		Data UserBalanceData `json:"data"`
	}{}
	path := coinbeneAPIVersion + coinbeneAccountBalanceOne
	err := c.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet,
		path,
		coinbeneAccountBalanceOne,
		false,
		v,
		&resp,
		spotAccountAssetInfo)
	if err != nil {
		return UserBalanceData{}, err
	}
	return resp.Data, nil
}

// PlaceSpotOrder creates an order
func (c *Coinbene) PlaceSpotOrder(price, quantity float64, symbol, direction,
	orderType, clientID string, notional int) (OrderPlacementResponse, error) {
	var resp OrderPlacementResponse
	params := url.Values{}
	switch direction {
	case order.Buy.Lower():
		params.Set("direction", buyDirection)
	case order.Sell.Lower():
		params.Set("direction", sellDirection)
	default:
		return resp,
			fmt.Errorf("invalid direction '%v', must be either 'buy' or 'sell'",
				direction)
	}

	switch orderType {
	case order.Limit.Lower():
		params.Set("orderType", limitOrder)
	case order.Market.Lower():
		params.Set("orderType", marketOrder)
	case order.PostOnly.Lower():
		params.Set("orderType", postOnlyOrder)
	case order.FillOrKill.Lower():
		params.Set("orderType", fillOrKillOrder)
	case order.IOS.Lower():
		params.Set("orderType", iosOrder)
	default:
		return resp,
			errors.New("invalid order type, must be either 'limit', 'market', 'postOnly', 'fillOrKill', 'ios'")
	}

	params.Set("symbol", symbol)
	params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	params.Set("quantity", strconv.FormatFloat(quantity, 'f', -1, 64))
	if clientID != "" {
		params.Set("clientId", clientID)
	}
	if notional != 0 {
		params.Set("notional", strconv.Itoa(notional))
	}
	path := coinbeneAPIVersion + coinbenePlaceOrder
	err := c.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost,
		path,
		coinbenePlaceOrder,
		false,
		params,
		&resp,
		spotPlaceOrder)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// PlaceSpotOrders sets a batchful order request
func (c *Coinbene) PlaceSpotOrders(orders []PlaceOrderRequest) ([]OrderPlacementResponse, error) {
	if len(orders) == 0 {
		return nil, errors.New("orders is nil")
	}

	type ord struct {
		Symbol    string `json:"symbol"`
		Direction string `json:"direction"`
		Price     string `json:"price"`
		Quantity  string `json:"quantity"`
		OrderType string `json:"orderType"`
		Notional  string `json:"notional,omitempty"`
		ClientID  string `json:"clientId,omitempty"`
	}

	var reqOrders []ord
	for x := range orders {
		o := ord{
			Symbol:   orders[x].Symbol,
			Price:    strconv.FormatFloat(orders[x].Price, 'f', -1, 64),
			Quantity: strconv.FormatFloat(orders[x].Quantity, 'f', -1, 64),
		}
		switch orders[x].Direction {
		case order.Buy.Lower():
			o.Direction = buyDirection
		case order.Sell.Lower():
			o.Direction = sellDirection
		default:
			return nil,
				fmt.Errorf("invalid direction '%v', must be either 'buy' or 'sell'",
					orders[x].Direction)
		}

		switch orders[x].OrderType {
		case order.Limit.Lower():
			o.OrderType = limitOrder
		case order.Market.Lower():
			o.OrderType = marketOrder
		default:
			return nil,
				errors.New("invalid order type, must be either 'limit' or 'market'")
		}

		if orders[x].ClientID != "" {
			o.ClientID = orders[x].ClientID
		}
		if orders[x].Notional != 0 {
			o.Notional = strconv.Itoa(orders[x].Notional)
		}
		reqOrders = append(reqOrders, o)
	}

	resp := struct {
		Data []OrderPlacementResponse `json:"data"`
	}{}
	path := coinbeneAPIVersion + coinbeneBatchPlaceOrder
	err := c.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost,
		path,
		coinbeneBatchPlaceOrder,
		false,
		reqOrders,
		&resp,
		spotBatchOrder)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// FetchOpenSpotOrders finds open orders
func (c *Coinbene) FetchOpenSpotOrders(symbol string) (OrdersInfo, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	path := coinbeneAPIVersion + coinbeneOpenOrders
	var orders OrdersInfo
	for i := int64(1); ; i++ {
		temp := struct {
			Data OrdersInfo `json:"data"`
		}{}
		params.Set("pageNum", strconv.FormatInt(i, 10))
		err := c.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet,
			path,
			coinbeneOpenOrders,
			false,
			params,
			&temp,
			spotQueryOpenOrders)
		if err != nil {
			return nil, err
		}
		for j := range temp.Data {
			orders = append(orders, temp.Data[j])
		}

		if len(temp.Data) != 20 {
			break
		}
	}
	return orders, nil
}

// FetchClosedOrders finds open orders
func (c *Coinbene) FetchClosedOrders(symbol, latestID string) (OrdersInfo, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("latestOrderId", latestID)
	path := coinbeneAPIVersion + coinbeneClosedOrders
	var orders OrdersInfo
	for i := int64(1); ; i++ {
		temp := struct {
			Data OrdersInfo `json:"data"`
		}{}
		params.Set("pageNum", strconv.FormatInt(i, 10))
		err := c.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet,
			path,
			coinbeneClosedOrders,
			false,
			params,
			&temp,
			spotQueryClosedOrders)
		if err != nil {
			return nil, err
		}
		for j := range temp.Data {
			orders = append(orders, temp.Data[j])
		}
		if len(temp.Data) != 20 {
			break
		}
	}
	return orders, nil
}

// FetchSpotOrderInfo gets order info
func (c *Coinbene) FetchSpotOrderInfo(orderID string) (OrderInfo, error) {
	resp := struct {
		Data OrderInfo `json:"data"`
	}{}
	params := url.Values{}
	params.Set("orderId", orderID)
	path := coinbeneAPIVersion + coinbeneOrderInfo
	err := c.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet,
		path,
		coinbeneOrderInfo,
		false,
		params,
		&resp,
		spotQuerySpecficOrder)
	if err != nil {
		return resp.Data, err
	}
	if resp.Data.OrderID != orderID {
		return resp.Data, fmt.Errorf("%s orderID doesn't match the returned orderID %s",
			orderID, resp.Data.OrderID)
	}
	return resp.Data, nil
}

// GetSpotOrderFills returns a list of fills related to an order ID
func (c *Coinbene) GetSpotOrderFills(orderID string) ([]OrderFills, error) {
	resp := struct {
		Data []OrderFills `json:"data"`
	}{}
	params := url.Values{}
	params.Set("orderId", orderID)
	path := coinbeneAPIVersion + coinbeneTradeFills
	err := c.SendAuthHTTPRequest(exchange.RestSpot, http.MethodGet,
		path,
		coinbeneTradeFills,
		false,
		params,
		&resp,
		spotQueryTradeFills)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// CancelSpotOrder removes a given order
func (c *Coinbene) CancelSpotOrder(orderID string) (string, error) {
	resp := struct {
		Data string `json:"data"`
	}{}
	req := make(map[string]interface{})
	req["orderId"] = orderID
	path := coinbeneAPIVersion + coinbeneCancelOrder
	err := c.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost,
		path,
		coinbeneCancelOrder,
		false,
		req,
		&resp,
		spotCancelOrder)
	if err != nil {
		return "", err
	}
	return resp.Data, nil
}

// CancelSpotOrders cancels a batch of orders
func (c *Coinbene) CancelSpotOrders(orderIDs []string) ([]OrderCancellationResponse, error) {
	req := make(map[string]interface{})
	req["orderIds"] = orderIDs
	type resp struct {
		Data []OrderCancellationResponse `json:"data"`
	}

	var r resp
	path := coinbeneAPIVersion + coinbeneBatchCancel
	err := c.SendAuthHTTPRequest(exchange.RestSpot, http.MethodPost,
		path,
		coinbeneBatchCancel,
		false,
		req,
		&r,
		spotCancelOrdersBatch)
	if err != nil {
		return nil, err
	}
	return r.Data, nil
}

// GetSwapTickers returns a map of swap tickers
func (c *Coinbene) GetSwapTickers() (SwapTickers, error) {
	type resp struct {
		Data SwapTickers `json:"data"`
	}
	var r resp
	path := coinbeneAPIVersion + coinbeneGetTickers
	err := c.SendHTTPRequest(exchange.RestSwap, path, contractTickers, &r)
	if err != nil {
		return nil, err
	}
	return r.Data, nil
}

// GetSwapTicker returns a specific swap ticker
func (c *Coinbene) GetSwapTicker(symbol string) (SwapTicker, error) {
	tickers, err := c.GetSwapTickers()
	if err != nil {
		return SwapTicker{}, err
	}
	t, ok := tickers[strings.ToUpper(symbol)]
	if !ok {
		return SwapTicker{},
			fmt.Errorf("symbol %s not found in tickers map", symbol)
	}
	return t, nil
}

// GetSwapInstruments returns a list of tradable instruments
func (c *Coinbene) GetSwapInstruments() ([]Instrument, error) {
	resp := struct {
		Data []Instrument `json:"data"`
	}{}
	return resp.Data, c.SendHTTPRequest(exchange.RestSwap,
		coinbeneAPIVersion+coinbeneGetInstruments, contractInstruments, &resp)
}

// GetSwapOrderbook returns an orderbook for the specified currency
func (c *Coinbene) GetSwapOrderbook(symbol string, size int64) (Orderbook, error) {
	var s Orderbook
	if symbol == "" {
		return s, fmt.Errorf("a symbol must be specified")
	}

	v := url.Values{}
	v.Set("symbol", symbol)
	if size != 0 {
		v.Set("size", strconv.FormatInt(size, 10))
	}

	type resp struct {
		Data struct {
			Asks   [][]string `json:"asks"`
			Bids   [][]string `json:"bids"`
			Time   time.Time  `json:"timestamp"`
			Symbol string     `json:"symbol"`
		} `json:"data"`
	}

	var r resp
	path := common.EncodeURLValues(coinbeneAPIVersion+coinbeneGetOrderBook, v)
	err := c.SendHTTPRequest(exchange.RestSwap, path, contractOrderbook, &r)
	if err != nil {
		return s, err
	}

	processOB := func(ob [][]string) ([]OrderbookItem, error) {
		var o []OrderbookItem
		for x := range ob {
			var price, amount float64
			var count int64
			count, err = strconv.ParseInt(ob[x][2], 10, 64)
			if err != nil {
				return nil, err
			}
			price, err = strconv.ParseFloat(ob[x][0], 64)
			if err != nil {
				return nil, err
			}
			amount, err = strconv.ParseFloat(ob[x][1], 64)
			if err != nil {
				return nil, err
			}
			o = append(o, OrderbookItem{Price: price,
				Amount: amount,
				Count:  count,
			})
		}
		return o, nil
	}

	s.Bids, err = processOB(r.Data.Bids)
	if err != nil {
		return s, err
	}
	s.Asks, err = processOB(r.Data.Asks)
	if err != nil {
		return s, err
	}
	s.Time = r.Data.Time
	s.Symbol = r.Data.Symbol
	return s, nil
}

// GetKlines data returns the kline data for a specific symbol
func (c *Coinbene) GetKlines(pair string, start, end time.Time, period string) (resp CandleResponse, err error) {
	v := url.Values{}
	v.Add("symbol", pair)
	if !start.IsZero() {
		v.Add("start", strconv.FormatInt(start.Unix(), 10))
	}
	if !end.IsZero() {
		v.Add("end", strconv.FormatInt(end.Unix(), 10))
	}
	v.Add("period", period)

	path := common.EncodeURLValues(coinbeneAPIVersion+coinbeneSpotKlines, v)
	if err = c.SendHTTPRequest(exchange.RestSpot, path, contractKline, &resp); err != nil {
		return
	}

	if resp.Code != 200 {
		return resp, errors.New(resp.Message)
	}

	return
}

// GetSwapKlines data returns the kline data for a specific symbol
func (c *Coinbene) GetSwapKlines(symbol string, start, end time.Time, resolution string) (resp CandleResponse, err error) {
	v := url.Values{}
	v.Set("symbol", symbol)
	if !start.IsZero() {
		v.Add("startTime", strconv.FormatInt(start.Unix(), 10))
	}
	if !end.IsZero() {
		v.Add("endTime", strconv.FormatInt(end.Unix(), 10))
	}
	v.Set("resolution", resolution)

	path := common.EncodeURLValues(coinbeneAPIVersion+coinbeneGetKlines, v)
	if err = c.SendHTTPRequest(exchange.RestSwap, path, contractKline, &resp); err != nil {
		return
	}

	return
}

// GetSwapTrades returns a list of trades for a swap symbol
func (c *Coinbene) GetSwapTrades(symbol string, limit int) (SwapTrades, error) {
	v := url.Values{}
	v.Set("symbol", symbol)
	if limit != 0 {
		v.Set("limit", strconv.Itoa(limit))
	}
	type resp struct {
		Data [][]string `json:"data"`
	}
	var r resp
	path := common.EncodeURLValues(coinbeneAPIVersion+coinbeneGetTrades, v)
	if err := c.SendHTTPRequest(exchange.RestSwap, path, contractTrades, &r); err != nil {
		return nil, err
	}

	var s SwapTrades
	for x := range r.Data {
		tm, err := time.Parse(time.RFC3339, r.Data[x][3])
		if err != nil {
			return nil, err
		}
		price, err := strconv.ParseFloat(r.Data[x][0], 64)
		if err != nil {
			return nil, err
		}
		orderSide := order.Buy
		if r.Data[x][1] == "s" {
			orderSide = order.Sell
		}
		volume, err := strconv.ParseFloat(r.Data[x][2], 64)
		if err != nil {
			return nil, err
		}
		s = append(s, SwapTrade{
			Price:  price,
			Side:   orderSide,
			Volume: volume,
			Time:   tm,
		})
	}
	return s, nil
}

// GetSwapAccountInfo returns a users swap account balance info
func (c *Coinbene) GetSwapAccountInfo() (SwapAccountInfo, error) {
	type resp struct {
		Data SwapAccountInfo `json:"data"`
	}
	var r resp
	path := coinbeneAPIVersion + coinbeneAccountInfo
	err := c.SendAuthHTTPRequest(exchange.RestSwap, http.MethodGet,
		path,
		coinbeneAccountInfo,
		true,
		nil,
		&r,
		contractAccountInfo)
	if err != nil {
		return SwapAccountInfo{}, err
	}
	return r.Data, nil
}

// GetSwapPositions returns a list of open swap positions
func (c *Coinbene) GetSwapPositions(symbol string) (SwapPositions, error) {
	v := url.Values{}
	v.Set("symbol", symbol)
	type resp struct {
		Data SwapPositions `json:"data"`
	}
	var r resp
	path := coinbeneAPIVersion + coinbeneListSwapPositions
	err := c.SendAuthHTTPRequest(exchange.RestSwap, http.MethodGet,
		path,
		coinbeneListSwapPositions,
		true,
		v,
		&r,
		contractPositionInfo)
	if err != nil {
		return nil, err
	}
	return r.Data, nil
}

// PlaceSwapOrder places a swap order
func (c *Coinbene) PlaceSwapOrder(symbol, direction, orderType, marginMode,
	clientID string, price, quantity float64, leverage int) (SwapPlaceOrderResponse, error) {
	v := url.Values{}
	v.Set("symbol", symbol)

	switch direction {
	case order.Buy.Lower():
		v.Set("direction", openLong)
	case order.Sell.Lower():
		v.Set("direction", openShort)
	default:
		return SwapPlaceOrderResponse{},
			fmt.Errorf("invalid direction '%v', must be either 'buy' or 'sell'",
				direction)
	}

	switch orderType {
	case order.Limit.Lower():
		v.Set("orderType", limitOrder)
	case order.Market.Lower():
		v.Set("orderType", marketOrder)
	default:
		return SwapPlaceOrderResponse{},
			errors.New("invalid order type, must be either 'limit' or 'market'")
	}

	v.Set("leverage", strconv.Itoa(leverage))
	v.Set("orderPrice", strconv.FormatFloat(price, 'f', -1, 64))
	v.Set("quantity", strconv.FormatFloat(quantity, 'f', -1, 64))
	if marginMode != "" {
		v.Set("marginMode", marginMode)
	}
	if clientID != "" {
		v.Set("clientId", clientID)
	}

	type resp struct {
		Data SwapPlaceOrderResponse `json:"data"`
	}
	var r resp
	path := coinbeneAPIVersion + coinbenePlaceOrder
	err := c.SendAuthHTTPRequest(exchange.RestSwap, http.MethodPost,
		path,
		coinbenePlaceOrder,
		true,
		v,
		&r,
		contractPlaceOrder)
	if err != nil {
		return SwapPlaceOrderResponse{}, err
	}
	return r.Data, nil
}

// CancelSwapOrder cancels a swap order
func (c *Coinbene) CancelSwapOrder(orderID string) (string, error) {
	params := make(map[string]interface{})
	params["orderId"] = orderID
	type resp struct {
		Data string `json:"data"`
	}
	var r resp
	path := coinbeneAPIVersion + coinbeneCancelOrder
	err := c.SendAuthHTTPRequest(exchange.RestSwap, http.MethodPost,
		path,
		coinbeneCancelOrder,
		true,
		params,
		&r,
		contractCancelOrder)
	if err != nil {
		return "", err
	}
	return r.Data, nil
}

// GetSwapOpenOrders gets a list of open swap orders
func (c *Coinbene) GetSwapOpenOrders(symbol string, pageNum, pageSize int) (SwapOrders, error) {
	v := url.Values{}
	v.Set("symbol", symbol)
	if pageNum != 0 {
		v.Set("pageNum", strconv.Itoa(pageNum))
	}
	if pageSize != 0 {
		v.Set("pageSize", strconv.Itoa(pageSize))
	}
	type resp struct {
		Data SwapOrders `json:"data"`
	}
	var r resp
	path := coinbeneAPIVersion + coinbeneOpenOrders
	err := c.SendAuthHTTPRequest(exchange.RestSwap, http.MethodGet,
		path,
		coinbeneOpenOrders,
		true,
		v,
		&r,
		contractGetOpenOrders)
	if err != nil {
		return nil, err
	}
	return r.Data, nil
}

// GetSwapOpenOrdersByPage gets a list of open orders by page
func (c *Coinbene) GetSwapOpenOrdersByPage(symbol string, latestOrderID int64) (SwapOrders, error) {
	v := url.Values{}
	if symbol != "" {
		v.Set("symbol", symbol)
	}
	if latestOrderID != 0 {
		v.Set("latestOrderId", strconv.FormatInt(latestOrderID, 10))
	}
	type resp struct {
		Data SwapOrders `json:"data"`
	}
	var r resp
	path := coinbeneAPIVersion + coinbeneOpenOrdersByPage
	err := c.SendAuthHTTPRequest(exchange.RestSwap, http.MethodGet,
		path,
		coinbeneOpenOrdersByPage,
		true,
		v,
		&r,
		contractOpenOrdersByPage)
	if err != nil {
		return nil, err
	}
	return r.Data, nil
}

// GetSwapOrderInfo gets order info for a specific order
func (c *Coinbene) GetSwapOrderInfo(orderID string) (SwapOrder, error) {
	v := url.Values{}
	v.Set("orderId", orderID)
	type resp struct {
		Data SwapOrder `json:"data"`
	}
	var r resp
	path := coinbeneAPIVersion + coinbeneOrderInfo
	err := c.SendAuthHTTPRequest(exchange.RestSwap, http.MethodGet,
		path,
		coinbeneOrderInfo,
		true,
		v,
		&r,
		contractGetOrderInfo)
	if err != nil {
		return SwapOrder{}, err
	}
	return r.Data, nil
}

// GetSwapOrderHistory returns the swap order history for a given symbol
func (c *Coinbene) GetSwapOrderHistory(beginTime, endTime, symbol string, pageNum,
	pageSize int, direction, orderType string) (SwapOrders, error) {
	v := url.Values{}
	if beginTime != "" {
		v.Set("beginTime", beginTime)
	}
	if endTime != "" {
		v.Set("endTime", endTime)
	}
	v.Set("symbol", symbol)
	if pageNum != 0 {
		v.Set("pageNum", strconv.Itoa(pageNum))
	}
	if pageSize != 0 {
		v.Set("pageSize", strconv.Itoa(pageSize))
	}
	if direction != "" {
		v.Set("direction", direction)
	}
	if orderType != "" {
		v.Set("orderType", orderType)
	}

	type resp struct {
		Data SwapOrders `json:"data"`
	}

	var r resp
	path := coinbeneAPIVersion + coinbeneClosedOrders
	err := c.SendAuthHTTPRequest(exchange.RestSwap, http.MethodGet,
		path,
		coinbeneClosedOrders,
		true,
		v,
		&r,
		contractGetClosedOrders)
	if err != nil {
		return nil, err
	}
	return r.Data, nil
}

// GetSwapOrderHistoryByOrderID returns a list of historic orders based on user params
func (c *Coinbene) GetSwapOrderHistoryByOrderID(beginTime, endTime, symbol, status string,
	latestOrderID int64) (SwapOrders, error) {
	v := url.Values{}
	if beginTime != "" {
		v.Set("beginTime", beginTime)
	}
	if endTime != "" {
		v.Set("endTime", endTime)
	}
	if symbol != "" {
		v.Set("symbol", symbol)
	}
	if status != "" {
		v.Set("status", status)
	}
	if latestOrderID != 0 {
		v.Set("latestOrderId", strconv.FormatInt(latestOrderID, 10))
	}
	type resp struct {
		Data SwapOrders `json:"data"`
	}

	var r resp
	path := coinbeneAPIVersion + coinbeneClosedOrdersByPage
	err := c.SendAuthHTTPRequest(exchange.RestSwap, http.MethodGet,
		path,
		coinbeneClosedOrdersByPage,
		true,
		v,
		&r,
		contractGetClosedOrdersbyPage)
	if err != nil {
		return nil, err
	}
	return r.Data, nil
}

// CancelSwapOrders cancels multiple swap order IDs
func (c *Coinbene) CancelSwapOrders(orderIDs []string) ([]OrderCancellationResponse, error) {
	if len(orderIDs) > 10 {
		return nil, errors.New("only 10 orderIDs are allowed at a time")
	}
	req := make(map[string]interface{})
	req["orderIds"] = orderIDs
	type resp struct {
		Data []OrderCancellationResponse `json:"data"`
	}

	var r resp
	path := coinbeneAPIVersion + coinbeneBatchCancel
	err := c.SendAuthHTTPRequest(exchange.RestSwap, http.MethodPost,
		path,
		coinbeneBatchCancel,
		true,
		req,
		&r,
		contractCancelMultipleOrders)
	if err != nil {
		return nil, err
	}
	return r.Data, nil
}

// GetSwapOrderFills returns a list of swap order fills
func (c *Coinbene) GetSwapOrderFills(symbol, orderID string, lastTradeID int64) (SwapOrderFills, error) {
	v := url.Values{}
	if symbol != "" {
		v.Set("symbol", symbol)
	}
	if orderID != "" {
		v.Set("orderId", orderID)
	}
	if lastTradeID != 0 {
		v.Set("lastTradedId", strconv.FormatInt(lastTradeID, 10))
	}
	type resp struct {
		Data SwapOrderFills `json:"data"`
	}

	var r resp
	path := coinbeneAPIVersion + coinbeneOrderFills
	err := c.SendAuthHTTPRequest(exchange.RestSwap, http.MethodGet,
		path,
		coinbeneOrderFills,
		true,
		v,
		&r,
		contractGetOrderFills)
	if err != nil {
		return nil, err
	}
	return r.Data, nil
}

// GetSwapFundingRates returns a list of funding rates
func (c *Coinbene) GetSwapFundingRates(pageNum, pageSize int) ([]SwapFundingRate, error) {
	v := url.Values{}
	if pageNum != 0 {
		v.Set("pageNum", strconv.Itoa(pageNum))
	}
	if pageSize != 0 {
		v.Set("pageSize", strconv.Itoa(pageSize))
	}
	type resp struct {
		Data []SwapFundingRate `json:"data"`
	}

	var r resp
	path := coinbeneAPIVersion + coinbenePositionFeeRate
	err := c.SendAuthHTTPRequest(exchange.RestSwap, http.MethodGet,
		path,
		coinbenePositionFeeRate,
		true,
		v,
		&r,
		contractGetFundingRates)
	if err != nil {
		return nil, err
	}
	return r.Data, nil
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (c *Coinbene) SendHTTPRequest(ep exchange.URL, path string, f request.EndpointLimit, result interface{}) error {
	endpoint, err := c.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	var resp json.RawMessage
	errCap := struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}{}

	if err := c.SendPayload(context.Background(), &request.Item{
		Method:        http.MethodGet,
		Path:          endpoint + path,
		Result:        &resp,
		Verbose:       c.Verbose,
		HTTPDebugging: c.HTTPDebugging,
		HTTPRecording: c.HTTPRecording,
		Endpoint:      f,
	}); err != nil {
		return err
	}

	if err := json.Unmarshal(resp, &errCap); err == nil {
		if errCap.Code != 200 && errCap.Message != "" {
			return errors.New(errCap.Message)
		}
	}
	return json.Unmarshal(resp, result)
}

// SendAuthHTTPRequest sends an authenticated HTTP request
func (c *Coinbene) SendAuthHTTPRequest(ep exchange.URL, method, path, epPath string, isSwap bool,
	params, result interface{}, f request.EndpointLimit) error {
	if !c.AllowAuthenticatedRequest() {
		return fmt.Errorf("%s %w", c.Name, exchange.ErrAuthenticatedRequestWithoutCredentialsSet)
	}
	endpoint, err := c.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}
	authPath := coinbeneAuthPath
	if isSwap {
		authPath = coinbeneSwapAuthPath
	}
	now := time.Now()
	timestamp := now.UTC().Format("2006-01-02T15:04:05.999Z")
	var finalBody io.Reader
	var preSign string
	switch {
	case params != nil && method == http.MethodGet:
		p, ok := params.(url.Values)
		if !ok {
			return fmt.Errorf("params is not of type url.Values")
		}
		preSign = timestamp + method + authPath + epPath + "?" + p.Encode()
		path = common.EncodeURLValues(path, p)
	case params != nil:
		var i interface{}
		switch p := params.(type) {
		case url.Values:
			m := make(map[string]string)
			for k, v := range p {
				m[k] = strings.Join(v, "")
			}
			i = m
		default:
			i = p
		}
		tempBody, err := json.Marshal(i)
		if err != nil {
			return err
		}
		finalBody = bytes.NewBufferString(string(tempBody))
		preSign = timestamp + method + authPath + epPath + string(tempBody)
	default:
		preSign = timestamp + method + authPath + epPath
	}
	tempSign := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(preSign),
		[]byte(c.API.Credentials.Secret))
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	headers["ACCESS-KEY"] = c.API.Credentials.Key
	headers["ACCESS-SIGN"] = crypto.HexEncodeToString(tempSign)
	headers["ACCESS-TIMESTAMP"] = timestamp

	var resp json.RawMessage
	errCap := struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}{}

	// Expiry of timestamp doesn't appear to be documented, so making a reasonable assumption
	ctx, cancel := context.WithDeadline(context.Background(), now.Add(15*time.Second))
	defer cancel()
	if err := c.SendPayload(ctx, &request.Item{
		Method:        method,
		Path:          endpoint + path,
		Headers:       headers,
		Body:          finalBody,
		Result:        &resp,
		AuthRequest:   true,
		Verbose:       c.Verbose,
		HTTPDebugging: c.HTTPDebugging,
		HTTPRecording: c.HTTPRecording,
		Endpoint:      f,
	}); err != nil {
		return err
	}

	if err := json.Unmarshal(resp, &errCap); err == nil {
		if errCap.Code != 200 && errCap.Message != "" {
			return errors.New(errCap.Message)
		}
	}
	return json.Unmarshal(resp, result)
}
