package coinbene

import (
	"bytes"
	"encoding/json"
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
	coinbeneAPIURL     = "https://openapi-exchange.coinbene.com/api/exchange/"
	coinbeneAuthPath   = "/api/exchange/v2"
	coinbeneAPIVersion = "v2"

	// Public endpoints
	coinbeneFetchTicker    = "/market/ticker/one"
	coinbeneFetchOrderBook = "/market/orderBook"
	coinbeneGetTrades      = "/market/trades"
	coinbeneGetAllPairs    = "/market/tradePair/list"
	coinbenePairInfo       = "/market/tradePair/one"

	// Authenticated endpoints
	coinbeneGetUserBalance = "/account/list"
	coinbenePlaceOrder     = "/order/place"
	coinbeneOrderInfo      = "/order/info"
	coinbeneRemoveOrder    = "/order/cancel"
	coinbeneOpenOrders     = "/order/openOrders"
	coinbeneClosedOrders   = "/order/closedOrders"

	authRateLimit   = 150
	unauthRateLimit = 10
)

// GetTicker gets and stores ticker data for a currency pair
func (c *Coinbene) GetTicker(symbol string) (TickerResponse, error) {
	var t TickerResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	path := common.EncodeURLValues(c.API.Endpoints.URL+coinbeneAPIVersion+coinbeneFetchTicker, params)
	return t, c.SendHTTPRequest(path, &t)
}

// GetOrderbook gets and stores orderbook data for given pair
func (c *Coinbene) GetOrderbook(symbol string, size int64) (OrderbookResponse, error) {
	var o OrderbookResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("depth", strconv.FormatInt(size, 10))
	path := common.EncodeURLValues(c.API.Endpoints.URL+coinbeneAPIVersion+coinbeneFetchOrderBook, params)
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
	err := c.SendAuthHTTPRequest(http.MethodGet, path, coinbeneGetUserBalance, nil, &resp)
	if err != nil {
		return resp, err
	}
	if resp.Code != 200 {
		return resp, fmt.Errorf(resp.Message)
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
		params,
		&resp)
	if err != nil {
		return resp, err
	}
	if resp.Code != 200 {
		return resp, fmt.Errorf(resp.Message)
	}
	return resp, nil
}

// FetchOrderInfo gets order info
func (c *Coinbene) FetchOrderInfo(orderID string) (OrderInfoResponse, error) {
	var resp OrderInfoResponse
	params := url.Values{}
	params.Set("orderId", orderID)
	path := c.API.Endpoints.URL + coinbeneAPIVersion + coinbeneOrderInfo
	err := c.SendAuthHTTPRequest(http.MethodGet, path, coinbeneOrderInfo, params, &resp)
	if err != nil {
		return resp, err
	}
	if resp.Code != 200 {
		return resp, fmt.Errorf(resp.Message)
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
	path := c.API.Endpoints.URL + coinbeneAPIVersion + coinbeneRemoveOrder
	err := c.SendAuthHTTPRequest(http.MethodPost, path, coinbeneRemoveOrder, params, &resp)
	if err != nil {
		return resp, err
	}
	if resp.Code != 200 {
		return resp, fmt.Errorf(resp.Message)
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
		err := c.SendAuthHTTPRequest(http.MethodGet, path, coinbeneOpenOrders, params, &temp)
		if err != nil {
			return resp, err
		}
		if temp.Code != 200 {
			return resp, fmt.Errorf(temp.Message)
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
		err := c.SendAuthHTTPRequest(http.MethodGet, path, coinbeneClosedOrders, params, &temp)
		if err != nil {
			return resp, err
		}
		if temp.Code != 200 {
			return resp, fmt.Errorf(temp.Message)
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

// SendHTTPRequest sends an unauthenticated HTTP request
func (c *Coinbene) SendHTTPRequest(path string, result interface{}) error {
	return c.SendPayload(http.MethodGet,
		path,
		nil,
		nil,
		&result,
		false,
		false,
		c.Verbose,
		c.HTTPDebugging,
		c.HTTPRecording)
}

// SendAuthHTTPRequest sends an authenticated HTTP request
func (c *Coinbene) SendAuthHTTPRequest(method, path, epPath string, params url.Values, result interface{}) error {
	if !c.AllowAuthenticatedRequest() {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, c.Name)
	}

	if params == nil {
		params = url.Values{}
	}

	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.999Z")
	var finalBody io.Reader
	var preSign string
	switch {
	case len(params) != 0 && method == http.MethodGet:
		preSign = fmt.Sprintf("%s%s%s%s?%s", timestamp, method, coinbeneAuthPath, epPath, params.Encode())
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
		preSign = timestamp + method + coinbeneAuthPath + epPath + string(tempBody)
	case len(params) == 0:
		preSign = timestamp + method + coinbeneAuthPath + epPath
	}
	tempSign := crypto.GetHMAC(crypto.HashSHA256,
		[]byte(preSign),
		[]byte(c.API.Credentials.Secret))
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	headers["ACCESS-KEY"] = c.API.Credentials.Key
	headers["ACCESS-SIGN"] = crypto.HexEncodeToString(tempSign)
	headers["ACCESS-TIMESTAMP"] = timestamp
	return c.SendPayload(method,
		path,
		headers,
		finalBody,
		&result,
		true,
		false,
		c.Verbose,
		c.HTTPDebugging,
		c.HTTPRecording)
}
