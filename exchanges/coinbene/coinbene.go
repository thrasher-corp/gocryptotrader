package coinbene

import (
	"bytes"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
)

// Coinbene is the overarching type across this package
type Coinbene struct {
	exchange.Base
	WebsocketConn *wshandler.WebsocketConnection
}

const (
	coinbeneAPIURL       = "https://openapi-exchange.coinbene.com/api/exchange/"
	coinbeneSwapAPIURL   = "https://openapi-contract.coinbene.com/api/swap/"
	coinbeneAuthPath     = "/api/exchange/v2"
	coinbeneSwapAuthPath = "/api/swap/v2"
	coinbeneAPIVersion   = "v2"
	buy                  = "buy"
	sell                 = "sell"

	// Public endpoints
	coinbeneGetTicker    = "/market/ticker/one"
	coinbeneGetTickers   = "/market/tickers"
	coinbeneGetOrderBook = "/market/orderBook"
	coinbeneGetKlines    = "/market/klines"
	coinbeneGetTrades    = "/market/trades"
	coinbeneGetAllPairs  = "/market/tradePair/list"
	coinbenePairInfo     = "/market/tradePair/one"

	// Authenticated endpoints
	coinbeneAccountInfo       = "/account/info"
	coinbeneGetUserBalance    = "/account/list"
	coinbenePlaceOrder        = "/order/place"
	coinbeneOrderInfo         = "/order/info"
	coinbeneCancelOrder       = "/order/cancel"
	coinbeneOpenOrders        = "/order/openOrders"
	coinbeneClosedOrders      = "/order/closedOrders"
	coinbeneListSwapPositions = "/position/list"

	authRateLimit   = 150
	unauthRateLimit = 10
)

// GetTicker gets and stores ticker data for a currency pair
func (c *Coinbene) GetTicker(symbol string) (TickerResponse, error) {
	var t TickerResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	path := common.EncodeURLValues(c.API.Endpoints.URL+coinbeneAPIVersion+coinbeneGetTicker, params)
	return t, c.SendHTTPRequest(path, &t)
}

// GetOrderbook gets and stores orderbook data for given pair
func (c *Coinbene) GetOrderbook(symbol string, size int64) (OrderbookResponse, error) {
	var o OrderbookResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("depth", strconv.FormatInt(size, 10))
	path := common.EncodeURLValues(c.API.Endpoints.URL+coinbeneAPIVersion+coinbeneGetOrderBook, params)
	return o, c.SendHTTPRequest(path, &o)
}

// GetTrades gets recent trades from the exchange
func (c *Coinbene) GetTrades(symbol string) (TradeResponse, error) {
	var t TradeResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	path := common.EncodeURLValues(c.API.Endpoints.URL+coinbeneAPIVersion+coinbeneGetTrades, params)
	return t, c.SendHTTPRequest(path, &t)
}

// GetPairInfo gets info about a single pair
func (c *Coinbene) GetPairInfo(symbol string) (PairResponse, error) {
	var resp PairResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	path := common.EncodeURLValues(c.API.Endpoints.URL+coinbeneAPIVersion+coinbenePairInfo, params)
	return resp, c.SendHTTPRequest(path, &resp)
}

// GetAllPairs gets all pairs on the exchange
func (c *Coinbene) GetAllPairs() (AllPairResponse, error) {
	var a AllPairResponse
	path := c.API.Endpoints.URL + coinbeneAPIVersion + coinbeneGetAllPairs
	return a, c.SendHTTPRequest(path, &a)
}

// GetUserBalance gets user balanace info
func (c *Coinbene) GetUserBalance() (UserBalanceResponse, error) {
	var resp UserBalanceResponse
	path := c.API.Endpoints.URL + coinbeneAPIVersion + coinbeneGetUserBalance
	err := c.SendAuthHTTPRequest(http.MethodGet,
		path,
		coinbeneGetUserBalance,
		false,
		nil,
		&resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// PlaceOrder creates an order
func (c *Coinbene) PlaceOrder(price, quantity float64, symbol, direction, clientID string) (PlaceOrderResponse, error) {
	var resp PlaceOrderResponse
	path := c.API.Endpoints.URL + coinbeneAPIVersion + coinbenePlaceOrder
	params := url.Values{}
	params.Set("symbol", symbol)
	switch direction {
	case order.Buy.Lower():
		params.Set("direction", "2")
	case order.Sell.Lower():
		params.Set("direction", "1")
	default:
		return resp,
			fmt.Errorf("passed in direction %s is invalid must be 'buy' or 'sell'",
				direction)
	}

	params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	params.Set("quantity", strconv.FormatFloat(quantity, 'f', -1, 64))
	params.Set("clientId", clientID)
	err := c.SendAuthHTTPRequest(http.MethodPost,
		path,
		coinbenePlaceOrder,
		false,
		params,
		&resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// FetchOrderInfo gets order info
func (c *Coinbene) FetchOrderInfo(orderID string) (OrderInfoResponse, error) {
	var resp OrderInfoResponse
	params := url.Values{}
	params.Set("orderId", orderID)
	path := c.API.Endpoints.URL + coinbeneAPIVersion + coinbeneOrderInfo
	err := c.SendAuthHTTPRequest(http.MethodGet, path, coinbeneOrderInfo, false, params, &resp)
	if err != nil {
		return resp, err
	}
	if resp.Order.OrderID != orderID {
		return resp, fmt.Errorf("%s orderID doesn't match the returned orderID %s",
			orderID, resp.Order.OrderID)
	}
	return resp, nil
}

// RemoveOrder removes a given order
func (c *Coinbene) RemoveOrder(orderID string) (RemoveOrderResponse, error) {
	var resp RemoveOrderResponse
	params := url.Values{}
	params.Set("orderId", orderID)
	path := c.API.Endpoints.URL + coinbeneAPIVersion + coinbeneCancelOrder
	err := c.SendAuthHTTPRequest(http.MethodPost, path, coinbeneCancelOrder, false, params, &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// FetchOpenOrders finds open orders
func (c *Coinbene) FetchOpenOrders(symbol string) (OpenOrderResponse, error) {
	var resp OpenOrderResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	path := c.API.Endpoints.URL + coinbeneAPIVersion + coinbeneOpenOrders
	for i := int64(1); ; i++ {
		var temp OpenOrderResponse
		params.Set("pageNum", strconv.FormatInt(i, 10))
		err := c.SendAuthHTTPRequest(http.MethodGet, path, coinbeneOpenOrders, false, params, &temp)
		if err != nil {
			return resp, err
		}
		for j := range temp.OpenOrders {
			resp.OpenOrders = append(resp.OpenOrders, temp.OpenOrders[j])
		}

		if len(temp.OpenOrders) != 20 {
			break
		}
	}
	return resp, nil
}

// FetchClosedOrders finds open orders
func (c *Coinbene) FetchClosedOrders(symbol, latestID string) (ClosedOrderResponse, error) {
	var resp ClosedOrderResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("latestOrderId", latestID)
	path := c.API.Endpoints.URL + coinbeneAPIVersion + coinbeneClosedOrders
	for i := int64(1); ; i++ {
		var temp ClosedOrderResponse
		params.Set("pageNum", strconv.FormatInt(i, 10))
		err := c.SendAuthHTTPRequest(http.MethodGet, path, coinbeneClosedOrders, false, params, &temp)
		if err != nil {
			return resp, err
		}
		for j := range temp.Data {
			resp.Data = append(resp.Data, temp.Data[j])
		}
		if len(temp.Data) != 20 {
			break
		}
	}
	return resp, nil
}

// GetSwapTickers returns a map of swap tickers
func (c *Coinbene) GetSwapTickers() (SwapTickers, error) {
	type resp struct {
		Data SwapTickers `json:"data"`
	}
	var r resp
	path := coinbeneSwapAPIURL + coinbeneAPIVersion + coinbeneGetTickers
	err := c.SendHTTPRequest(path, &r)
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

// GetSwapOrderbook returns an orderbook for the specified currency
func (c *Coinbene) GetSwapOrderbook(symbol, size string) (SwapOrderbook, error) {
	var s SwapOrderbook
	if symbol == "" {
		return s, fmt.Errorf("a symbol must be specified")
	}

	v := url.Values{}
	v.Set("symbol", symbol)
	if size != "" {
		v.Set("size", size)
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
	path := common.EncodeURLValues(coinbeneSwapAPIURL+coinbeneAPIVersion+coinbeneGetOrderBook, v)
	err := c.SendHTTPRequest(path, &r)
	if err != nil {
		return s, err
	}

	processOB := func(ob [][]string) ([]SwapOrderbookItem, error) {
		var o []SwapOrderbookItem
		for x := range ob {
			var price, amount float64
			var count int64
			price, err = strconv.ParseFloat(ob[x][0], 64)
			if err != nil {
				return nil, err
			}
			amount, err = strconv.ParseFloat(ob[x][1], 64)
			if err != nil {
				return nil, err
			}
			count, err = strconv.ParseInt(ob[x][2], 10, 64)
			if err != nil {
				return nil, err
			}
			o = append(o, SwapOrderbookItem{Price: price,
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

// GetSwapKlines data returns the kline data for a specific symbol
func (c *Coinbene) GetSwapKlines(symbol, startTime, endTime, resolution string) (SwapKlines, error) {
	var s SwapKlines
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("startTime", startTime)
	v.Set("endTime", endTime)
	v.Set("resolution", resolution)

	type resp struct {
		Data [][]string `json:"data"`
	}
	var r resp
	path := common.EncodeURLValues(coinbeneSwapAPIURL+coinbeneAPIVersion+coinbeneGetKlines, v)
	if err := c.SendHTTPRequest(path, &r); err != nil {
		return nil, err
	}

	for x := range r.Data {
		tm, err := strconv.ParseInt(r.Data[x][0], 10, 64)
		if err != nil {
			return nil, err
		}
		open, err := strconv.ParseFloat(r.Data[x][1], 64)
		if err != nil {
			return nil, err
		}
		closePrice, err := strconv.ParseFloat(r.Data[x][2], 64)
		if err != nil {
			return nil, err
		}
		high, err := strconv.ParseFloat(r.Data[x][3], 64)
		if err != nil {
			return nil, err
		}
		low, err := strconv.ParseFloat(r.Data[x][4], 64)
		if err != nil {
			return nil, err
		}
		volume, err := strconv.ParseFloat(r.Data[x][5], 64)
		if err != nil {
			return nil, err
		}
		turnover, err := strconv.ParseFloat(r.Data[x][6], 64)
		if err != nil {
			return nil, err
		}
		buyVolume, err := strconv.ParseFloat(r.Data[x][7], 64)
		if err != nil {
			return nil, err
		}
		buyTurnover, err := strconv.ParseFloat(r.Data[x][8], 64)
		if err != nil {
			return nil, err
		}
		s = append(s, SwapKlineItem{
			Time:        time.Unix(tm, 0),
			Open:        open,
			Close:       closePrice,
			High:        high,
			Low:         low,
			Volume:      volume,
			Turnover:    turnover,
			BuyVolume:   buyVolume,
			BuyTurnover: buyTurnover,
		})
	}
	return s, nil
}

// GetSwapTrades returns a list of trades for a swap symbol
func (c *Coinbene) GetSwapTrades(symbol, limit string) (SwapTrades, error) {
	v := url.Values{}
	v.Set("symbol", symbol)
	if limit != "" {
		v.Set("limit", limit)
	}
	type resp struct {
		Data [][]string `json:"data"`
	}
	var r resp
	path := common.EncodeURLValues(coinbeneSwapAPIURL+coinbeneAPIVersion+coinbeneGetTrades, v)
	if err := c.SendHTTPRequest(path, &r); err != nil {
		return nil, err
	}

	var s SwapTrades
	for x := range r.Data {
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
		tm, err := time.Parse(time.RFC3339, r.Data[x][3])
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
	var s SwapAccountInfo
	type resp struct {
		Data SwapAccountInfo `json:"data"`
	}
	var r resp
	path := coinbeneSwapAPIURL + coinbeneAPIVersion + coinbeneAccountInfo
	err := c.SendAuthHTTPRequest(http.MethodGet, path, coinbeneAccountInfo, true, nil, &r)
	if err != nil {
		return s, err
	}
	return r.Data, nil
}

// GetSwapPositions returns a list of open swap positions
func (c *Coinbene) GetSwapPositions(symbol string) (SwapPositions, error) {
	v := url.Values{}
	v.Set("symbol", symbol)
	var s SwapPositions
	type resp struct {
		Data SwapPositions `json:"data"`
	}
	var r resp
	path := coinbeneSwapAPIURL + coinbeneAPIVersion + coinbeneListSwapPositions
	err := c.SendAuthHTTPRequest(http.MethodGet, path, coinbeneListSwapPositions, true, v, &r)
	if err != nil {
		return s, err
	}
	return r.Data, nil
}

// PlaceSwapOrder places a swap order
func (c *Coinbene) PlaceSwapOrder(symbol, direction, orderType, marginMode,
	clientID string, price, quantity float64, leverage int) (SwapPlaceOrderResponse, error) {
	v := url.Values{}
	v.Set("symbol", symbol)
	v.Set("direction", direction)
	v.Set("leverage", strconv.Itoa(leverage))
	v.Set("orderType", orderType)
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
	path := coinbeneSwapAPIURL + coinbeneAPIVersion + coinbenePlaceOrder
	err := c.SendAuthHTTPRequest(http.MethodPost, path, coinbenePlaceOrder, true, v, &r)
	if err != nil {
		return SwapPlaceOrderResponse{}, err
	}
	return r.Data, nil
}

// CancelSwapOrder cancels a swap order
func (c *Coinbene) CancelSwapOrder(orderID string) (string, error) {
	v := url.Values{}
	v.Set("orderId", orderID)
	type resp struct {
		Data string `json:"data"`
	}
	var r resp
	path := coinbeneSwapAPIURL + coinbeneAPIVersion + coinbeneCancelOrder
	err := c.SendAuthHTTPRequest(http.MethodPost, path, coinbeneCancelOrder, true, v, &r)
	if err != nil {
		return "", err
	}
	return r.Data, nil
}

// GetSwapOpenOrders gets a list of open swap orders
func (c *Coinbene) GetSwapOpenOrders(symbol string) (SwapOrders, error) {
	return nil, nil
}

// SendHTTPRequest sends an unauthenticated HTTP request
func (c *Coinbene) SendHTTPRequest(path string, result interface{}) error {
	var resp json.RawMessage
	errCap := struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}{}

	if err := c.SendPayload(http.MethodGet,
		path,
		nil,
		nil,
		&resp,
		false,
		false,
		c.Verbose,
		c.HTTPDebugging,
		c.HTTPRecording); err != nil {
		return err
	}

	if err := common.JSONDecode(resp, &errCap); err == nil {
		if errCap.Code != 200 && errCap.Message != "" {
			return errors.New(errCap.Message)
		}
	}
	return common.JSONDecode(resp, result)
}

// SendAuthHTTPRequest sends an authenticated HTTP request
func (c *Coinbene) SendAuthHTTPRequest(method, path, epPath string, isSwap bool, params url.Values, result interface{}) error {
	if !c.AllowAuthenticatedRequest() {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet,
			c.Name)
	}

	if params == nil {
		params = url.Values{}
	}
	authPath := coinbeneAuthPath
	if isSwap {
		authPath = coinbeneSwapAuthPath
	}
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.999Z")
	var finalBody io.Reader
	var preSign string
	switch {
	case len(params) != 0 && method == http.MethodGet:
		preSign = fmt.Sprintf("%s%s%s%s?%s", timestamp, method, authPath, epPath, params.Encode())
		path = common.EncodeURLValues(path, params)
	case len(params) != 0:
		m := make(map[string]string)
		for k, v := range params {
			m[k] = strings.Join(v, "")
		}
		tempBody, err := json.Marshal(m)
		if err != nil {
			return err
		}
		finalBody = bytes.NewBufferString(string(tempBody))
		preSign = timestamp + method + authPath + epPath + string(tempBody)
	case len(params) == 0:
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

	if err := c.SendPayload(method,
		path,
		headers,
		finalBody,
		&resp,
		true,
		false,
		c.Verbose,
		c.HTTPDebugging,
		c.HTTPRecording); err != nil {
		return err
	}

	if err := common.JSONDecode(resp, &errCap); err == nil {
		if errCap.Code != 200 && errCap.Message != "" {
			return errors.New(errCap.Message)
		}
	}
	return common.JSONDecode(resp, result)
}
