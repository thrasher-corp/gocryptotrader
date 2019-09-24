package coinbene

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// Coinbene is the overarching type across this package
type Coinbene struct {
	exchange.Base
}

const (
	coinbeneAPIURL     = "https://api.coinbene.com/"
	coinbeneAPIVersion = "v1"

	// Public endpoints
	coinbeneFetchTicker    = "/market/ticker"
	coinbeneFetchOrderBook = "/market/orderbook"
	coinbeneGetTrades      = "/market/trades"
	coinbeneGetAllPairs    = "/market/symbol"

	// Authenticated endpoints
	coinbeneGetUserBalance = "/trade/balance"
	coinbenePlaceOrder     = "/trade/order/place"
	coinbeneOrderInfo      = "/trade/order/info"
	coinbeneRemoveOrder    = "/trade/order/cancel"
	coinbeneOpenOrders     = "/trade/order/open-orders"
	coinbeneWithdrawApply  = "/withdraw/apply"
)

// SetDefaults sets the basic defaults for Coinbene
func (c *Coinbene) SetDefaults() {
	c.Name = "Coinbene"
	c.Enabled = false
	c.Verbose = false
	c.RESTPollingDelay = 10
	c.RequestCurrencyPairFormat.Delimiter = ""
	c.RequestCurrencyPairFormat.Uppercase = true
	c.ConfigCurrencyPairFormat.Delimiter = ""
	c.ConfigCurrencyPairFormat.Uppercase = true
	c.AssetTypes = []string{ticker.Spot}
	c.SupportsAutoPairUpdating = false
	c.SupportsRESTTickerBatching = false
	c.Requester = request.New(c.Name,
		request.NewRateLimit(time.Second, 0),
		request.NewRateLimit(time.Second, 0),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	c.APIUrlDefault = coinbeneAPIURL
	c.APIUrl = c.APIUrlDefault
	c.Websocket = wshandler.New()
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

		// If the exchange supports websocket, update the below block
		// err = c.WebsocketSetup(c.WsConnect,
		//	exch.Name,
		//	exch.Websocket,
		//	coinbeneWebsocket,
		//	exch.WebsocketURL)
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// c.WebsocketConn = &wshandler.WebsocketConnection{
		// 		ExchangeName:         c.Name,
		// 		URL:                  c.Websocket.GetWebsocketURL(),
		// 		ProxyURL:             c.Websocket.GetProxyAddress(),
		// 		Verbose:              c.Verbose,
		// 		ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
		// 		ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		// }
	}
}

// FetchTicker gets and stores ticker data for a currency pair
func (c *Coinbene) FetchTicker(symbol string) (TickerResponse, error) {
	var t TickerResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	path := fmt.Sprintf("%s%s%s?%s", c.APIUrl, coinbeneAPIVersion, coinbeneFetchTicker, params.Encode())
	return t, c.SendHTTPRequest(path, &t)
}

// FetchOrderbooks gets and stores orderbook data for given pair
func (c *Coinbene) FetchOrderbooks(symbol string) (OrderbookResponse, error) {
	var o OrderbookResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	path := fmt.Sprintf("%s%s%s?%s", c.APIUrl, coinbeneAPIVersion, coinbeneFetchOrderBook, params.Encode())
	return o, c.SendHTTPRequest(path, &o)
}

// GetTrades gets recent trades from the exchange
func (c *Coinbene) GetTrades(symbol, size string) (TradeResponse, error) {
	var t TradeResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("size", size)
	path := fmt.Sprintf("%s%s%s?%s", c.APIUrl, coinbeneAPIVersion, coinbeneGetTrades, params.Encode())
	return t, c.SendHTTPRequest(path, &t)
}

// GetAllPairs gets all pairs on the exchange
func (c *Coinbene) GetAllPairs() (AllPairResponse, error) {
	var a AllPairResponse
	path := fmt.Sprintf("%s%s%s", c.APIUrl, coinbeneAPIVersion, coinbeneGetAllPairs)
	return a, c.SendHTTPRequest(path, &a)
}

// GetUserBalance gets user balanace info
func (c *Coinbene) GetUserBalance() (UserBalanceResponse, error) {
	var resp UserBalanceResponse
	path := fmt.Sprintf("%s%s%s", c.APIUrl, coinbeneAPIVersion, coinbeneGetUserBalance)
	params := url.Values{}
	params.Set("account", "exchange")
	return resp, c.SendAuthHTTPRequest(http.MethodPost, path, params, &resp)
}

// PlaceOrder creates an order
func (c *Coinbene) PlaceOrder(price, quantity float64, symbol, orderType string) (PlaceOrderResponse, error) {
	var resp PlaceOrderResponse
	path := fmt.Sprintf("%s%s%s", c.APIUrl, coinbeneAPIVersion, coinbenePlaceOrder)
	params := url.Values{}
	params.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	params.Set("quantity", strconv.FormatFloat(quantity, 'f', -1, 64))
	params.Set("symbol", symbol)
	params.Set("type", orderType)
	// params.Set("account", "exchange")
	return resp, c.SendAuthHTTPRequest(http.MethodPost, path, params, &resp)
}

// FetchOrderInfo gets order info
func (c *Coinbene) FetchOrderInfo(orderID string) (OrderInfoResponse, error) {
	var resp OrderInfoResponse
	params := url.Values{}
	params.Set("orderid", orderID)
	path := fmt.Sprintf("%s%s%s", c.APIUrl, coinbeneAPIVersion, coinbeneOrderInfo)
	return resp, c.SendAuthHTTPRequest(http.MethodPost, path, params, &resp)
}

// RemoveOrder removes a given order
func (c *Coinbene) RemoveOrder(orderID string) (RemoveOrderResponse, error) {
	var resp RemoveOrderResponse
	params := url.Values{}
	params.Set("orderid", orderID)
	path := fmt.Sprintf("%s%s%s", c.APIUrl, coinbeneAPIVersion, coinbeneRemoveOrder)
	return resp, c.SendAuthHTTPRequest(http.MethodPost, path, params, &resp)
}

// FetchOpenOrders finds open orders
func (c *Coinbene) FetchOpenOrders(symbol string) (OpenOrderResponse, error) {
	var resp OpenOrderResponse
	params := url.Values{}
	params.Set("symbol", symbol)
	path := fmt.Sprintf("%s%s%s", c.APIUrl, coinbeneAPIVersion, coinbeneOpenOrders)
	return resp, c.SendAuthHTTPRequest(http.MethodPost, path, params, &resp)
}

// WithdrawApply sends a withdraw application
func (c *Coinbene) WithdrawApply(amount float64, asset, address, tag, chain string) (WithdrawResponse, error) {
	var resp WithdrawResponse
	params := url.Values{}
	params.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	params.Set("asset", asset)
	params.Set("address", address)
	params.Set("tag", tag)
	params.Set("chain", chain)
	path := fmt.Sprintf("%s%s%s", c.APIUrl, coinbeneAPIVersion, coinbeneWithdrawApply)
	return resp, c.SendAuthHTTPRequest(http.MethodPost, path, params, &resp)
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
func (c *Coinbene) SendAuthHTTPRequest(method, path string, params url.Values, result interface{}) error {
	if params == nil {
		params = url.Values{}
	}
	timestamp := strconv.FormatInt(time.Now().UnixNano()/1000000, 10)
	params.Set("apiid", c.APIKey)
	params.Set("timestamp", timestamp)
	var temp, tempSlice []string
	for x := range params {
		if params[x][0] == "" {
			continue
		}
		temp = append(temp, x)
	}
	temp = append(temp, "SECRET")
	sort.Strings(temp)

	for y := range temp {
		if temp[y] == "SECRET" {
			tempSlice = append(tempSlice, strings.ToUpper(fmt.Sprintf("%s=%s", temp[y], c.APISecret)))
		} else {
			tempSlice = append(tempSlice, strings.ToUpper(fmt.Sprintf("%s=%s", temp[y], params[temp[y]][0])))
		}
	}
	sort.Strings(tempSlice)
	fmt.Println("THIS IS MY SORTED TEMP STRING", tempSlice)
	signMsg := strings.Join(tempSlice, "&")
	log.Println(signMsg)

	md5 := common.GetMD5([]byte(signMsg))

	params.Set("sign", common.HexEncodeToString(md5))

	// path = common.EncodeURLValues(path, params)

	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"

	var postbody string
	for key, val := range params {
		postbody = postbody + `"` + key + `":"` + val[0] + `",`
	}

	postbody = `{` + postbody[0:len(postbody)-1] + `}`

	return c.SendPayload(method,
		path,
		headers,
		bytes.NewBufferString(postbody),
		&result,
		true,
		false,
		c.Verbose,
		c.HTTPDebugging,
		c.HTTPRecording)
}
