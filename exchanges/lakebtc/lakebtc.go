package lakebtc

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	lakeBTCAPIURL              = "https://api.lakebtc.com/api_v2"
	lakeBTCAPIVersion          = "2"
	lakeBTCTicker              = "ticker"
	lakeBTCOrderbook           = "bcorderbook"
	lakeBTCTrades              = "bctrades"
	lakeBTCGetAccountInfo      = "getAccountInfo"
	lakeBTCBuyOrder            = "buyOrder"
	lakeBTCSellOrder           = "sellOrder"
	lakeBTCOpenOrders          = "openOrders"
	lakeBTCGetOrders           = "getOrders"
	lakeBTCCancelOrder         = "cancelOrder"
	lakeBTCGetTrades           = "getTrades"
	lakeBTCGetExternalAccounts = "getExternalAccounts"
	lakeBTCCreateWithdraw      = "createWithdraw"

	lakeBTCAuthRate = 0
	lakeBTCUnauth   = 0
)

// LakeBTC is the overarching type across the LakeBTC package
type LakeBTC struct {
	exchange.Base
}

// SetDefaults sets LakeBTC defaults
func (l *LakeBTC) SetDefaults() {
	l.Name = "LakeBTC"
	l.Enabled = false
	l.TakerFee = 0.2
	l.MakerFee = 0.15
	l.Verbose = false
	l.Websocket = false
	l.RESTPollingDelay = 10
	l.RequestCurrencyPairFormat.Delimiter = ""
	l.RequestCurrencyPairFormat.Uppercase = true
	l.ConfigCurrencyPairFormat.Delimiter = ""
	l.ConfigCurrencyPairFormat.Uppercase = true
	l.AssetTypes = []string{ticker.Spot}
	l.SupportsAutoPairUpdating = true
	l.SupportsRESTTickerBatching = true
	l.Requester = request.New(l.Name, request.NewRateLimit(time.Second, lakeBTCAuthRate), request.NewRateLimit(time.Second, lakeBTCUnauth), common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout))
}

// Setup sets exchange configuration profile
func (l *LakeBTC) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		l.SetEnabled(false)
	} else {
		l.Enabled = true
		l.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		l.SetAPIKeys(exch.APIKey, exch.APISecret, "", false)
		l.SetHTTPClientTimeout(exch.HTTPTimeout)
		l.SetHTTPClientUserAgent(exch.HTTPUserAgent)
		l.RESTPollingDelay = exch.RESTPollingDelay
		l.Verbose = exch.Verbose
		l.Websocket = exch.Websocket
		l.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		l.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		l.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
		err := l.SetCurrencyPairFormat()
		if err != nil {
			log.Fatal(err)
		}
		err = l.SetAssetTypes()
		if err != nil {
			log.Fatal(err)
		}
		err = l.SetAutoPairDefaults()
		if err != nil {
			log.Fatal(err)
		}
	}
}

// GetTradablePairs returns a list of available pairs from the exchange
func (l *LakeBTC) GetTradablePairs() ([]string, error) {
	result, err := l.GetTicker()
	if err != nil {
		return nil, err
	}

	var currencies []string
	for x := range result {
		currencies = append(currencies, common.StringToUpper(x))
	}

	return currencies, nil
}

// GetFee returns maker or taker fee
func (l *LakeBTC) GetFee(maker bool) float64 {
	if maker {
		return l.MakerFee
	}
	return l.TakerFee
}

// GetTicker returns the current ticker from lakeBTC
func (l *LakeBTC) GetTicker() (map[string]Ticker, error) {
	response := make(map[string]TickerResponse)
	path := fmt.Sprintf("%s/%s", lakeBTCAPIURL, lakeBTCTicker)

	if err := l.SendHTTPRequest(path, &response); err != nil {
		return nil, err
	}

	result := make(map[string]Ticker)

	var addresses []string
	for k, v := range response {
		var ticker Ticker
		key := common.StringToUpper(k)
		if v.Ask != nil {
			ticker.Ask, _ = strconv.ParseFloat(v.Ask.(string), 64)
		}
		if v.Bid != nil {
			ticker.Bid, _ = strconv.ParseFloat(v.Bid.(string), 64)
		}
		if v.High != nil {
			ticker.High, _ = strconv.ParseFloat(v.High.(string), 64)
		}
		if v.Last != nil {
			ticker.Last, _ = strconv.ParseFloat(v.Last.(string), 64)
		}
		if v.Low != nil {
			ticker.Low, _ = strconv.ParseFloat(v.Low.(string), 64)
		}
		if v.Volume != nil {
			ticker.Volume, _ = strconv.ParseFloat(v.Volume.(string), 64)
		}
		result[key] = ticker
		addresses = append(addresses, key)
	}
	return result, nil
}

// GetOrderBook returns the order book from LakeBTC
func (l *LakeBTC) GetOrderBook(currency string) (Orderbook, error) {
	type Response struct {
		Bids [][]string `json:"bids"`
		Asks [][]string `json:"asks"`
	}
	path := fmt.Sprintf("%s/%s?symbol=%s", lakeBTCAPIURL, lakeBTCOrderbook, common.StringToLower(currency))
	resp := Response{}
	err := l.SendHTTPRequest(path, &resp)
	if err != nil {
		return Orderbook{}, err
	}
	orderbook := Orderbook{}

	for _, x := range resp.Bids {
		price, err := strconv.ParseFloat(x[0], 64)
		if err != nil {
			log.Println(err)
			continue
		}
		amount, err := strconv.ParseFloat(x[1], 64)
		if err != nil {
			log.Println(err)
			continue
		}
		orderbook.Bids = append(orderbook.Bids, OrderbookStructure{price, amount})
	}

	for _, x := range resp.Asks {
		price, err := strconv.ParseFloat(x[0], 64)
		if err != nil {
			log.Println(err)
			continue
		}
		amount, err := strconv.ParseFloat(x[1], 64)
		if err != nil {
			log.Println(err)
			continue
		}
		orderbook.Asks = append(orderbook.Asks, OrderbookStructure{price, amount})
	}
	return orderbook, nil
}

// GetTradeHistory returns the trade history for a given currency pair
func (l *LakeBTC) GetTradeHistory(currency string) ([]TradeHistory, error) {
	path := fmt.Sprintf("%s/%s?symbol=%s", lakeBTCAPIURL, lakeBTCTrades, common.StringToLower(currency))
	resp := []TradeHistory{}

	return resp, l.SendHTTPRequest(path, &resp)
}

// GetAccountInfo returns your current account information
func (l *LakeBTC) GetAccountInfo() (AccountInfo, error) {
	resp := AccountInfo{}

	return resp, l.SendAuthenticatedHTTPRequest(lakeBTCGetAccountInfo, "", &resp)
}

// Trade executes an order on the exchange and returns trade inforamtion or an
// error
func (l *LakeBTC) Trade(orderType int, amount, price float64, currency string) (Trade, error) {
	resp := Trade{}
	params := strconv.FormatFloat(price, 'f', -1, 64) + "," + strconv.FormatFloat(amount, 'f', -1, 64) + "," + currency

	if orderType == 1 {
		if err := l.SendAuthenticatedHTTPRequest(lakeBTCBuyOrder, params, &resp); err != nil {
			return resp, err
		}
	} else {
		if err := l.SendAuthenticatedHTTPRequest(lakeBTCSellOrder, params, &resp); err != nil {
			return resp, err
		}
	}

	if resp.Result != "order received" {
		return resp, fmt.Errorf("Unexpected result: %s", resp.Result)
	}

	return resp, nil
}

// GetOpenOrders returns all open orders associated with your account
func (l *LakeBTC) GetOpenOrders() ([]OpenOrders, error) {
	orders := []OpenOrders{}

	return orders, l.SendAuthenticatedHTTPRequest(lakeBTCOpenOrders, "", &orders)
}

// GetOrders returns your orders
func (l *LakeBTC) GetOrders(orders []int64) ([]Orders, error) {
	var ordersStr []string
	for _, x := range orders {
		ordersStr = append(ordersStr, strconv.FormatInt(x, 10))
	}

	resp := []Orders{}
	return resp,
		l.SendAuthenticatedHTTPRequest(lakeBTCGetOrders, common.JoinStrings(ordersStr, ","), &resp)
}

// CancelOrder cancels an order by ID number and returns an error
func (l *LakeBTC) CancelOrder(orderID int64) error {
	type Response struct {
		Result bool `json:"Result"`
	}

	resp := Response{}
	params := strconv.FormatInt(orderID, 10)
	err := l.SendAuthenticatedHTTPRequest(lakeBTCCancelOrder, params, &resp)
	if err != nil {
		return err
	}

	if resp.Result != true {
		return errors.New("unable to cancel order")
	}
	return nil
}

// GetTrades returns trades associated with your account by timestamp
func (l *LakeBTC) GetTrades(timestamp int64) ([]AuthenticatedTradeHistory, error) {
	params := ""
	if timestamp != 0 {
		params = strconv.FormatInt(timestamp, 10)
	}

	trades := []AuthenticatedTradeHistory{}
	return trades, l.SendAuthenticatedHTTPRequest(lakeBTCGetTrades, params, &trades)
}

// GetExternalAccounts returns your external accounts WARNING: Only for BTC!
func (l *LakeBTC) GetExternalAccounts() ([]ExternalAccounts, error) {
	resp := []ExternalAccounts{}

	return resp, l.SendAuthenticatedHTTPRequest(lakeBTCGetExternalAccounts, "", &resp)
}

// CreateWithdraw allows your to withdraw to external account WARNING: Only for
// BTC!
func (l *LakeBTC) CreateWithdraw(amount float64, accountID int64) (Withdraw, error) {
	resp := Withdraw{}
	params := strconv.FormatFloat(amount, 'f', -1, 64) + ",btc," + strconv.FormatInt(accountID, 10)

	return resp, l.SendAuthenticatedHTTPRequest(lakeBTCCreateWithdraw, params, &resp)
}

// SendHTTPRequest sends an unauthenticated http request
func (l *LakeBTC) SendHTTPRequest(path string, result interface{}) error {
	return l.SendPayload("GET", path, nil, nil, result, false, l.Verbose)
}

// SendAuthenticatedHTTPRequest sends an autheticated HTTP request to a LakeBTC
func (l *LakeBTC) SendAuthenticatedHTTPRequest(method, params string, result interface{}) (err error) {
	if !l.AuthenticatedAPISupport {
		return fmt.Errorf(exchange.WarningAuthenticatedRequestWithoutCredentialsSet, l.Name)
	}

	if l.Nonce.Get() == 0 {
		l.Nonce.Set(time.Now().UnixNano())
	} else {
		l.Nonce.Inc()
	}

	req := fmt.Sprintf("tonce=%s&accesskey=%s&requestmethod=post&id=1&method=%s&params=%s", l.Nonce.String(), l.APIKey, method, params)
	hmac := common.GetHMAC(common.HashSHA1, []byte(req), []byte(l.APISecret))

	if l.Verbose {
		log.Printf("Sending POST request to %s calling method %s with params %s\n", lakeBTCAPIURL, method, req)
	}

	postData := make(map[string]interface{})
	postData["method"] = method
	postData["id"] = 1
	postData["params"] = common.SplitStrings(params, ",")

	data, err := common.JSONEncode(postData)
	if err != nil {
		return err
	}

	headers := make(map[string]string)
	headers["Json-Rpc-Tonce"] = l.Nonce.String()
	headers["Authorization"] = "Basic " + common.Base64Encode([]byte(l.APIKey+":"+common.HexEncodeToString(hmac)))
	headers["Content-Type"] = "application/json-rpc"

	return l.SendPayload("POST", lakeBTCAPIURL, headers, strings.NewReader(string(data)), result, true, l.Verbose)
}
