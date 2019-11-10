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

	"github.com/idoall/gocryptotrader/common"
	"github.com/idoall/gocryptotrader/config"
	exchange "github.com/idoall/gocryptotrader/exchanges"
	"github.com/idoall/gocryptotrader/exchanges/request"
	"github.com/idoall/gocryptotrader/exchanges/ticker"
	"github.com/idoall/gocryptotrader/exchanges/websocket/wshandler"
	log "github.com/idoall/gocryptotrader/logger"
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
	buy                = "buy"
	sell               = "sell"

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

// SetDefaults sets the basic defaults for Coinbene
func (c *Coinbene) SetDefaults() {
	c.Name = "Coinbene"
	c.Enabled = false
	c.Verbose = false
	c.RESTPollingDelay = 10
	c.RequestCurrencyPairFormat.Delimiter = "/"
	c.RequestCurrencyPairFormat.Uppercase = true
	c.ConfigCurrencyPairFormat.Delimiter = "/"
	c.ConfigCurrencyPairFormat.Uppercase = true
	c.AssetTypes = []string{ticker.Spot}
	c.SupportsAutoPairUpdating = true
	c.SupportsRESTTickerBatching = false
	c.Requester = request.New(c.Name,
		request.NewRateLimit(time.Minute, authRateLimit),
		request.NewRateLimit(time.Second, unauthRateLimit),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	c.APIUrlDefault = coinbeneAPIURL
	c.APIUrl = c.APIUrlDefault
	c.Websocket = wshandler.New()
	c.WebsocketURL = coinbeneWsURL
	c.Websocket.Functionality = wshandler.WebsocketTickerSupported |
		wshandler.WebsocketTradeDataSupported |
		wshandler.WebsocketKlineSupported |
		wshandler.WebsocketAccountDataSupported |
		wshandler.WebsocketOrderbookSupported |
		wshandler.WebsocketSubscribeSupported |
		wshandler.WebsocketUnsubscribeSupported |
		wshandler.WebsocketAuthenticatedEndpointsSupported
	c.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	c.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	c.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (c *Coinbene) Setup(exch *config.ExchangeConfig) {
	if !exch.Enabled {
		c.SetEnabled(false)
	} else {
		c.Enabled = true
		c.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		c.AuthenticatedWebsocketAPISupport = exch.AuthenticatedWebsocketAPISupport
		c.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		c.SetHTTPClientTimeout(exch.HTTPTimeout)
		c.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		c.RESTPollingDelay = exch.RESTPollingDelay
		c.Verbose = exch.Verbose
		c.Websocket.SetWsStatusAndConnection(exch.Websocket)
		c.BaseCurrencies = exch.BaseCurrencies
		c.AvailablePairs = exch.AvailablePairs
		c.EnabledPairs = exch.EnabledPairs
		err := c.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = c.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
		err = c.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
		err = c.SetAPIURL(exch)
		if err != nil {
			log.Fatal(err)
		}
		err = c.SetClientProxyAddress(exch.ProxyAddress)
		if err != nil {
			log.Fatal(err)
		}

		err = c.Websocket.Setup(c.WsConnect,
			c.Subscribe,
			c.Unsubscribe,
			exch.Name,
			exch.Websocket,
			exch.Verbose,
			coinbeneWsURL,
			exch.WebsocketURL,
			exch.AuthenticatedWebsocketAPISupport)
		if err != nil {
			log.Fatal(err)
		}
		c.WebsocketConn = &wshandler.WebsocketConnection{
			ExchangeName:         c.Name,
			URL:                  c.Websocket.GetWebsocketURL(),
			ProxyURL:             c.Websocket.GetProxyAddress(),
			Verbose:              c.Verbose,
			ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		}
		c.Websocket.Orderbook.Setup(
			exch.WebsocketOrderbookBufferLimit,
			true,
			true,
			false,
			false,
			exch.Name)
	}
}

// FetchTicker gets and stores ticker data for a currency pair
func (c *Coinbene) FetchTicker(symbol string) (TickerResponse, error) {
	var t TickerResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	path := common.EncodeURLValues(c.APIUrl+coinbeneAPIVersion+coinbeneFetchTicker, params)
	return t, c.SendHTTPRequest(path, &t)
}

// FetchOrderbooks gets and stores orderbook data for given pair
func (c *Coinbene) FetchOrderbooks(symbol string, size int64) (OrderbookResponse, error) {
	var o OrderbookResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("depth", strconv.FormatInt(size, 10))
	path := common.EncodeURLValues(c.APIUrl+coinbeneAPIVersion+coinbeneFetchOrderBook, params)
	return o, c.SendHTTPRequest(path, &o)
}

// GetTrades gets recent trades from the exchange
func (c *Coinbene) GetTrades(symbol string) (TradeResponse, error) {
	var t TradeResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	path := common.EncodeURLValues(c.APIUrl+coinbeneAPIVersion+coinbeneGetTrades, params)
	return t, c.SendHTTPRequest(path, &t)
}

// GetPairInfo gets info about a single pair
func (c *Coinbene) GetPairInfo(symbol string) (PairResponse, error) {
	var resp PairResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	path := common.EncodeURLValues(c.APIUrl+coinbeneAPIVersion+coinbenePairInfo, params)
	return resp, c.SendHTTPRequest(path, &resp)
}

// GetAllPairs gets all pairs on the exchange
func (c *Coinbene) GetAllPairs() (AllPairResponse, error) {
	var a AllPairResponse
	path := c.APIUrl + coinbeneAPIVersion + coinbeneGetAllPairs
	return a, c.SendHTTPRequest(path, &a)
}

// GetUserBalance gets user balanace info
func (c *Coinbene) GetUserBalance() (UserBalanceResponse, error) {
	var resp UserBalanceResponse
	path := c.APIUrl + coinbeneAPIVersion + coinbeneGetUserBalance
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
	path := c.APIUrl + coinbeneAPIVersion + coinbenePlaceOrder
	params := url.Values{}
	params.Set("symbol", symbol)
	switch direction {
	case sell:
		params.Set("direction", "2")
	case buy:
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
	path := c.APIUrl + coinbeneAPIVersion + coinbeneOrderInfo
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
	path := c.APIUrl + coinbeneAPIVersion + coinbeneRemoveOrder
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
	path := c.APIUrl + coinbeneAPIVersion + coinbeneOpenOrders
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
	path := c.APIUrl + coinbeneAPIVersion + coinbeneClosedOrders
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
	tempSign := common.GetHMAC(common.HashSHA256, []byte(preSign), []byte(c.APISecret))
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	headers["ACCESS-KEY"] = c.APIKey
	headers["ACCESS-SIGN"] = common.HexEncodeToString(tempSign)
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
