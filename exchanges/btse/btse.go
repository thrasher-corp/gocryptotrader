package btse

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

// BTSE is the overarching type across this package
type BTSE struct {
	exchange.Base
	WebsocketConn *wshandler.WebsocketConnection
}

const (
	btseAPIURL  = "https://api.btse.com"
	btseAPIPath = "/spot/v2/"

	// Public endpoints
	btseMarketOverview = "market_summary"
	btseMarkets        = "markets"
	btseOrderbook      = "orderbook"
	btseTrades         = "trades"
	btseTicker         = "ticker"
	btseStats          = "stats"
	btseTime           = "time"

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
	b.APIWithdrawPermissions = exchange.NoAPIWithdrawalMethods
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
	b.Websocket = wshandler.New()
	b.Websocket.Functionality = wshandler.WebsocketOrderbookSupported |
		wshandler.WebsocketTickerSupported |
		wshandler.WebsocketSubscribeSupported |
		wshandler.WebsocketUnsubscribeSupported
	b.WebsocketResponseMaxLimit = exchange.DefaultWebsocketResponseMaxLimit
	b.WebsocketResponseCheckTimeout = exchange.DefaultWebsocketResponseCheckTimeout
	b.WebsocketOrderbookBufferLimit = exchange.DefaultWebsocketOrderbookBufferLimit
}

// Setup takes in the supplied exchange configuration details and sets params
func (b *BTSE) Setup(exch *config.ExchangeConfig) {
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
		b.Websocket.SetWsStatusAndConnection(exch.Websocket)
		b.BaseCurrencies = exch.BaseCurrencies
		b.AvailablePairs = exch.AvailablePairs
		b.EnabledPairs = exch.EnabledPairs
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
		err = b.Websocket.Setup(b.WsConnect,
			b.Subscribe,
			b.Unsubscribe,
			exch.Name,
			exch.Websocket,
			exch.Verbose,
			btseWebsocket,
			exch.WebsocketURL,
			exch.AuthenticatedWebsocketAPISupport)
		if err != nil {
			log.Fatal(err)
		}
		b.WebsocketConn = &wshandler.WebsocketConnection{
			ExchangeName:         b.Name,
			URL:                  b.Websocket.GetWebsocketURL(),
			ProxyURL:             b.Websocket.GetProxyAddress(),
			Verbose:              b.Verbose,
			ResponseCheckTimeout: exch.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:     exch.WebsocketResponseMaxLimit,
		}
		b.Websocket.Orderbook.Setup(
			exch.WebsocketOrderbookBufferLimit,
			false,
			false,
			false,
			false,
			exch.Name)
	}
}

// GetMarketsSummary stores market summary data
func (b *BTSE) GetMarketsSummary() (*HighLevelMarketData, error) {
	var m HighLevelMarketData
	return &m, b.SendHTTPRequest(http.MethodGet, btseMarketOverview, &m)
}

// GetMarkets returns a list of markets available on BTSE
func (b *BTSE) GetMarkets() ([]Market, error) {
	var m []Market
	return m, b.SendHTTPRequest(http.MethodGet, btseMarkets, &m)
}

// FetchOrderBook gets orderbook data for a given pair
func (b *BTSE) FetchOrderBook(symbol string) (*Orderbook, error) {
	var o Orderbook
	endpoint := fmt.Sprintf("%s/%s", btseOrderbook, symbol)
	return &o, b.SendHTTPRequest(http.MethodGet, endpoint, &o)
}

// GetTrades returns a list of trades for the specified symbol
func (b *BTSE) GetTrades(symbol string) (*Trades, error) {
	var t Trades
	endpoint := fmt.Sprintf("%s/%s", btseTrades, symbol)
	return &t, b.SendHTTPRequest(http.MethodGet, endpoint, &t)

}

// GetTicker returns the ticker for a specified symbol
func (b *BTSE) GetTicker(symbol string) (*Ticker, error) {
	var t Ticker
	endpoint := fmt.Sprintf("%s/%s", btseTicker, symbol)
	err := b.SendHTTPRequest(http.MethodGet, endpoint, &t)
	if err != nil {
		return nil, err
	}
	return &t, nil
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

// GetAccountBalance returns the users account balance
func (b *BTSE) GetAccountBalance() (*AccountBalance, error) {
	var a AccountBalance
	return &a, b.SendAuthenticatedHTTPRequest(http.MethodGet, btseAccount, nil, &a)
}

// CreateOrder creates an order
func (b *BTSE) CreateOrder(amount, price float64, side, orderType, symbol, timeInForce, tag string) (*string, error) {
	req := make(map[string]interface{})
	req["amount"] = amount
	req["price"] = price
	if req["side"] != "" {
		req["side"] = side
	}
	req["side"] = side
	req["type"] = orderType
	req["symbol"] = symbol

	type orderResp struct {
		ID string `json:"id"`
	}

	var r orderResp
	return &r.ID, b.SendAuthenticatedHTTPRequest(http.MethodPost, btseOrder, req, &r)
}

// GetOrders returns all pending orders
func (b *BTSE) GetOrders(symbol string) (*OpenOrders, error) {
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	var o OpenOrders
	return &o, b.SendAuthenticatedHTTPRequest(http.MethodGet, btsePendingOrders, req, &o)
}

// CancelExistingOrder cancels an order
func (b *BTSE) CancelExistingOrder(orderID, symbol string) (*CancelOrder, error) {
	var c CancelOrder
	req := make(map[string]interface{})
	req["order_id"] = orderID
	req["symbol"] = symbol
	return &c, b.SendAuthenticatedHTTPRequest(http.MethodPost, btseDeleteOrder, req, &c)
}

// CancelOrders cancels all orders
// productID optional. If product ID is sent, all orders of that specified market
// will be cancelled. If not specified, all orders of all markets will be cancelled
func (b *BTSE) CancelOrders(symbol string) (*CancelOrder, error) {
	var c CancelOrder
	req := make(map[string]interface{})
	if symbol != "" {
		req["symbol"] = symbol
	}
	return &c, b.SendAuthenticatedHTTPRequest(http.MethodPost, btseDeleteOrders, req, &c)
}

// GetFills gets all filled orders
func (b *BTSE) GetFills(orderID, symbol, before, after, limit, username string) (*FilledOrders, error) {
	if orderID != "" && symbol != "" {
		return nil, errors.New("orderID and symbol cannot co-exist in the same query")
	} else if orderID == "" && symbol == "" {
		return nil, errors.New("orderID OR symbol must be set")
	}

	req := make(map[string]interface{})
	if orderID != "" {
		req["order_id"] = orderID
	}

	if symbol != "" {
		req["symbol"] = symbol
	}

	if before != "" {
		req["before"] = before
	}

	if after != "" {
		req["after"] = after
	}

	if limit != "" {
		req["limit"] = limit
	}
	if username != "" {
		req["username"] = username
	}

	var o FilledOrders
	return &o, b.SendAuthenticatedHTTPRequest(http.MethodPost, btseFills, req, &o)
}

// SendHTTPRequest sends an HTTP request to the desired endpoint
func (b *BTSE) SendHTTPRequest(method, endpoint string, result interface{}) error {
	return b.SendPayload(method,
		fmt.Sprintf("%s%s%s", btseAPIURL, btseAPIPath, endpoint),
		nil,
		nil,
		&result,
		false,
		false,
		b.Verbose,
		b.HTTPDebugging,
		b.HTTPRecording)
}

// SendAuthenticatedHTTPRequest sends an authenticated HTTP request to the desired endpoint
func (b *BTSE) SendAuthenticatedHTTPRequest(method, endpoint string, req map[string]interface{}, result interface{}) error {
	if !b.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, b.Name)
	}
	path := fmt.Sprintf("%s%s", btseAPIPath, endpoint)

	payload, err := common.JSONEncode(req)
	if err != nil {
		return errors.New("sendAuthenticatedAPIRequest: unable to JSON request")
	}

	var body io.Reader
	if len(req) != 0 {
		body = bytes.NewBufferString(string(payload))
	}
	headers := make(map[string]string)
	headers["btse-api"] = b.APIKey
	nonce := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	headers["btse-nonce"] = nonce

	var temp []byte

	if string(payload) == "{}" || string(payload) == "null" {
		temp = common.GetHMAC(common.HashSHA512_384, []byte((path + nonce)), []byte(b.APISecret))
	} else {
		temp = common.GetHMAC(common.HashSHA512_384, []byte((path + nonce + string(payload))), []byte(b.APISecret))
	}

	headers["btse-sign"] = common.HexEncodeToString(temp)

	if len(payload) > 0 {
		headers["Accept"] = "application/json"
		headers["Content-Type"] = "application/json"
	}

	if b.Verbose {
		log.Debugf("Sending %s request to URL %s with params %s\n", method, path, string(payload))
	}

	return b.SendPayload(method,
		btseAPIURL+path,
		headers,
		body,
		&result,
		true,
		false,
		b.Verbose,
		b.HTTPDebugging,
		b.HTTPRecording)
}

// GetFee returns an estimate of fee based on type of transaction
func (b *BTSE) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64

	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = calculateTradingFee(feeBuilder.IsMaker)
	case exchange.CryptocurrencyWithdrawalFee:
		if feeBuilder.Pair.Base.Match(currency.BTC) {
			fee = 0.0005
		} else if feeBuilder.Pair.Base.Match(currency.USDT) {
			fee = 5
		}
	case exchange.InternationalBankDepositFee:
		fee = getInternationalBankDepositFee(feeBuilder.Amount)
	case exchange.InternationalBankWithdrawalFee:
		fee = getInternationalBankWithdrawalFee(feeBuilder.Amount)
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.PurchasePrice, feeBuilder.Amount)
	}
	return fee, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(price, amount float64) float64 {
	return 0.0015 * price * amount
}

// getInternationalBankDepositFee returns international deposit fee
// Only when the initial deposit amount is less than $1000 or equivalent,
// BTSE will charge a small fee (0.25% or $3 USD equivalent, whichever is greater).
// The small deposit fee is charged in whatever currency it comes in.
func getInternationalBankDepositFee(amount float64) float64 {
	var fee float64
	if amount <= 1000 {
		fee = amount * 0.0025
		if fee < 3 {
			return 3
		}
	}
	return fee
}

// getInternationalBankWithdrawalFee returns international withdrawal fee
// 0.1% (min25 USD)
func getInternationalBankWithdrawalFee(amount float64) float64 {
	fee := amount * 0.001

	if fee < 25 {
		return 25
	}
	return fee
}

// calculateTradingFee BTSE has fee tiers, but does not disclose them via API,
// so the largest fee has to be assumed
func calculateTradingFee(isMaker bool) float64 {
	fee := 0.00050
	if !isMaker {
		fee = 0.0015
	}
	return fee
}

func parseOrderTime(timeStr string) time.Time {
	t, _ := time.Parse("2006-01-02 15:04:04", timeStr)
	return t
}
