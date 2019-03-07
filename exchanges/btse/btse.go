package btse

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// BTSE is the overarching type across this package
type BTSE struct {
	exchange.Base
	WebsocketConn *websocket.Conn
}

const (
	btseAPIURL     = "https://api.btse.com/v1/restapi"
	btseAPIVersion = "1"

	// Public endpoints
	btseMarkets = "markets"
	btseTrades  = "trades"
	btseTicker  = "ticker"
	btseStats   = "stats"
	btseTime    = "time"

	// Authenticated endpoints
	btseAccount       = "account"
	btseOrder         = "order"
	btsePendingOrders = "pending"
	btseDeleteOrder   = "deleteOrder"
	btseDeleteOrders  = "deleteOrders"
	btseFills         = "fills"
)

// SetDefaults sets the basic defaults for BTSE
func (b *BTSE) SetDefaults() {
	b.Name = "BTSE"
	b.Enabled = false
	b.Verbose = false
	b.RESTPollingDelay = 10
	b.RequestCurrencyPairFormat.Delimiter = "-"
	b.RequestCurrencyPairFormat.Uppercase = true
	b.ConfigCurrencyPairFormat.Delimiter = "-"
	b.ConfigCurrencyPairFormat.Uppercase = true
	b.AssetTypes = []string{ticker.Spot}
	b.Requester = request.New(b.Name,
		request.NewRateLimit(time.Second, 0),
		request.NewRateLimit(time.Second, 0),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
	b.APIUrlDefault = btseAPIURL
	b.APIUrl = b.APIUrlDefault
	b.SupportsAutoPairUpdating = true
	b.SupportsRESTTickerBatching = false
	b.WebsocketInit()
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *BTSE) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		b.SetEnabled(false)
	} else {
		b.Enabled = true
		b.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		b.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		b.SetHTTPClientTimeout(exch.HTTPTimeout)
		b.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		b.RESTPollingDelay = exch.RESTPollingDelay
		b.Verbose = exch.Verbose
		b.Websocket.SetEnabled(exch.Websocket)
		b.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		b.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		b.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		err := b.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = b.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
		err = b.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
		err = b.SetAPIURL(exch)
		if err != nil {
			log.Fatal(err)
		}
		err = b.SetClientProxyAddress(exch.ProxyAddress)
		if err != nil {
			log.Fatal(err)
		}
		err = b.WebsocketSetup(b.WsConnect,
			exch.Name,
			exch.Websocket,
			btseWebsocket,
			exch.WebsocketURL)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetMarkets returns a list of markets available on BTSE
func (b *BTSE) GetMarkets() (*Markets, error) {
	var m Markets
	return &m, b.SendHTTPRequest(http.MethodGet, btseMarkets, &m)
}

// GetTrades returns a list of trades for the specified symbol
func (b *BTSE) GetTrades(symbol string) (*Trades, error) {
	var t Trades
	endpoint := fmt.Sprintf("%s/%s", btseTrades, symbol)
	return &t, b.SendHTTPRequest(http.MethodGet, endpoint, &t)

}

// GetTicker returns the ticker for a specified symbol
func (b *BTSE) GetTicker(symbol string) (*Ticker, error) {
	type tickerResponse struct {
		Price  interface{} `json:"price"`
		Size   float64     `json:"size,string"`
		Bid    float64     `json:"bid,string"`
		Ask    float64     `json:"ask,string"`
		Volume float64     `json:"volume,string"`
		Time   string      `json:"time"`
	}

	var r tickerResponse
	endpoint := fmt.Sprintf("%s/%s", btseTicker, symbol)
	err := b.SendHTTPRequest(http.MethodGet, endpoint, &r)
	if err != nil {
		return nil, err
	}

	p := strings.Replace(r.Price.(string), ",", "", -1)
	price, err := strconv.ParseFloat(p, 64)
	if err != nil {
		return nil, err
	}

	return &Ticker{
		Price:  price,
		Size:   r.Size,
		Bid:    r.Bid,
		Ask:    r.Ask,
		Volume: r.Volume,
		Time:   r.Time,
	}, nil
}

// GetMarketStatistics gets market statistics for a specificed market
func (b *BTSE) GetMarketStatistics(symbol string) (*MarketStatistics, error) {
	var m MarketStatistics
	endpoint := fmt.Sprintf("%s/%s", btseStats, symbol)
	return &m, b.SendHTTPRequest(http.MethodGet, endpoint, &m)
}

// GetServerTime returns the exchanges server time
func (b *BTSE) GetServerTime() (*ServerTime, error) {
	var s ServerTime
	return &s, b.SendHTTPRequest(http.MethodGet, btseTime, &s)
}

// GetAccount returns the users account balance
func (b *BTSE) GetAccount() (*AccountInfo, error) {
	var a AccountInfo
	return &a, b.SendAuthenticatedHTTPRequest(http.MethodGet, btseAccount, nil, &a)
}

// CreateOrder creates an order
func (b *BTSE) CreateOrder(amount, price float64, side, orderType, symbol, timeInForce, tag string) (*string, error) {
	vals := url.Values{}
	vals.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	vals.Set("price", strconv.FormatFloat(price, 'f', -1, 64))
	vals.Set("side", side)
	vals.Set("type", orderType)
	vals.Set("product_id", symbol)

	if timeInForce != "" {
		vals.Set("time_in_force", timeInForce)
	}

	if tag != "" {
		vals.Set("tag", tag)
	}

	var orderID string
	return &orderID, b.SendAuthenticatedHTTPRequest(http.MethodPost, btseOrder, vals, &orderID)
}

// GetOrders returns all pending orders
func (b *BTSE) GetOrders(productID string) (*Orders, error) {
	vals := url.Values{}
	if productID != "" {
		vals.Set("product_id", productID)
	}
	var o Orders
	return &o, b.SendAuthenticatedHTTPRequest(http.MethodGet, btsePendingOrders, vals, &o)
}

// CancelExistingOrder cancels an order
func (b *BTSE) CancelExistingOrder(orderID, productID string) (*CancelOrder, error) {
	var c CancelOrder
	vals := url.Values{}
	vals.Set("order_id", orderID)
	vals.Set("product_id", productID)
	return &c, b.SendAuthenticatedHTTPRequest(http.MethodPost, btseDeleteOrder, vals, &c)
}

// CancelOrders cancels all orders
func (b *BTSE) CancelOrders(productID string) (*CancelOrder, error) {
	var c CancelOrder
	vals := url.Values{}
	if productID != "" {
		vals.Set("product_id", productID)
	}
	return &c, b.SendAuthenticatedHTTPRequest(http.MethodPost, btseDeleteOrders, vals, &c)
}

// GetFills gets all filled orders
func (b *BTSE) GetFills(orderID, productID, before, after, limit string) (*FilledOrders, error) {
	if orderID != "" && productID != "" {
		return nil, errors.New("orderID and productID cannot co-exist in the same query")
	} else if orderID == "" && productID == "" {
		return nil, errors.New("orderID OR productID must be set")
	}

	vals := url.Values{}

	if orderID != "" {
		vals.Set("order_id", orderID)
	}

	if productID != "" {
		vals.Set("product_id", productID)
	}

	if before != "" {
		vals.Set("before", before)
	}

	if after != "" {
		vals.Set("after", after)
	}

	if limit != "" {
		vals.Set("limit", limit)
	}

	var o FilledOrders
	return &o, b.SendAuthenticatedHTTPRequest(http.MethodPost, btseFills, vals, &o)
}

// SendHTTPRequest sends an HTTP request to the desired endpoint
func (b *BTSE) SendHTTPRequest(method, endpoint string, result interface{}) error {
	p := fmt.Sprintf("%s/%s", btseAPIURL, endpoint)
	return b.SendPayload(method, p, nil, nil, &result, false, b.Verbose)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request to the desired endpoint
func (b *BTSE) SendAuthenticatedHTTPRequest(method, endpoint string, vals url.Values, result interface{}) error {
	if !b.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, b.Name)
	}

	p := fmt.Sprintf("%s/%s", btseAPIURL, endpoint)
	headers := make(map[string]string)
	headers["API-KEY"] = b.APIKey
	headers["API-PASSPHRASE"] = b.APISecret
	return b.SendPayload(method, p, headers, strings.NewReader(vals.Encode()), &result, true, b.Verbose)
}
