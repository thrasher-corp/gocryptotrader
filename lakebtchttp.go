package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

const (
	LAKEBTC_API_URL               = "https://api.lakebtc.com/api_v2"
	LAKEBTC_API_VERSION           = "2"
	LAKEBTC_TICKER                = "ticker"
	LAKEBTC_ORDERBOOK             = "bcorderbook"
	LAKEBTC_TRADES                = "bctrades"
	LAKEBTC_GET_ACCOUNT_INFO      = "getAccountInfo"
	LAKEBTC_BUY_ORDER             = "buyOrder"
	LAKEBTC_SELL_ORDER            = "sellOrder"
	LAKEBTC_OPEN_ORDERS           = "openOrders"
	LAKEBTC_GET_ORDERS            = "getOrders"
	LAKEBTC_CANCEL_ORDER          = "cancelOrder"
	LAKEBTC_GET_TRADES            = "getTrades"
	LAKEBTC_GET_EXTERNAL_ACCOUNTS = "getExternalAccounts"
	LAKEBTC_CREATE_WITHDRAW       = "createWithdraw"
)

type LakeBTC struct {
	Name                    string
	Enabled                 bool
	Verbose                 bool
	Websocket               bool
	RESTPollingDelay        time.Duration
	AuthenticatedAPISupport bool
	APIKey, APISecret       string
	TakerFee, MakerFee      float64
	BaseCurrencies          []string
	AvailablePairs          []string
	EnabledPairs            []string
}

type LakeBTCTicker struct {
	Last   float64
	Bid    float64
	Ask    float64
	High   float64
	Low    float64
	Volume float64
}

type LakeBTCOrderbookStructure struct {
	Price  float64
	Amount float64
}

type LakeBTCOrderbook struct {
	Bids []LakeBTCOrderbookStructure `json:"bids"`
	Asks []LakeBTCOrderbookStructure `json:"asks"`
}

func (l *LakeBTC) SetDefaults() {
	l.Name = "LakeBTC"
	l.Enabled = false
	l.TakerFee = 0.2
	l.MakerFee = 0.15
	l.Verbose = false
	l.Websocket = false
	l.RESTPollingDelay = 10
}

func (l *LakeBTC) GetName() string {
	return l.Name
}

func (l *LakeBTC) SetEnabled(enabled bool) {
	l.Enabled = enabled
}

func (l *LakeBTC) IsEnabled() bool {
	return l.Enabled
}

func (l *LakeBTC) Setup(exch Exchanges) {
	if !exch.Enabled {
		l.SetEnabled(false)
	} else {
		l.Enabled = true
		l.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		l.SetAPIKeys(exch.APIKey, exch.APISecret)
		l.RESTPollingDelay = exch.RESTPollingDelay
		l.Verbose = exch.Verbose
		l.Websocket = exch.Websocket
		l.BaseCurrencies = SplitStrings(exch.BaseCurrencies, ",")
		l.AvailablePairs = SplitStrings(exch.AvailablePairs, ",")
		l.EnabledPairs = SplitStrings(exch.EnabledPairs, ",")
	}
}

func (k *LakeBTC) GetEnabledCurrencies() []string {
	return k.EnabledPairs
}

func (l *LakeBTC) Start() {
	go l.Run()
}

func (l *LakeBTC) SetAPIKeys(apiKey, apiSecret string) {
	l.APIKey = apiKey
	l.APISecret = apiSecret
}

func (l *LakeBTC) GetFee(maker bool) float64 {
	if maker {
		return l.MakerFee
	} else {
		return l.TakerFee
	}
}

func (l *LakeBTC) Run() {
	if l.Verbose {
		log.Printf("%s polling delay: %ds.\n", l.GetName(), l.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", l.GetName(), len(l.EnabledPairs), l.EnabledPairs)
	}

	for l.Enabled {
		for _, x := range l.EnabledPairs {
			ticker, err := l.GetTickerPrice(x)
			if err != nil {
				log.Println(err)
				continue
			}
			log.Printf("LakeBTC BTC %s: Last %f High %f Low %f Volume %f\n", x[3:], ticker.Last, ticker.High, ticker.Low, ticker.Volume)
			AddExchangeInfo(l.GetName(), x[0:3], x[3:], ticker.Last, ticker.Volume)
		}
		time.Sleep(time.Second * l.RESTPollingDelay)
	}
}

/* Silly hack due to API returning null instead of strings */
type LakeBTCTickerResponse struct {
	Last   interface{}
	Bid    interface{}
	Ask    interface{}
	High   interface{}
	Low    interface{}
	Volume interface{}
}

func (l *LakeBTC) GetTicker() (map[string]LakeBTCTicker, error) {
	response := make(map[string]LakeBTCTickerResponse)
	path := fmt.Sprintf("%s/%s", LAKEBTC_API_URL, LAKEBTC_TICKER)
	err := SendHTTPGetRequest(path, true, &response)
	if err != nil {
		return nil, err
	}
	result := make(map[string]LakeBTCTicker)

	var addresses []string
	for k, v := range response {
		var ticker LakeBTCTicker
		key := StringToUpper(k)
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

func (l *LakeBTC) GetTickerPrice(currency string) (TickerPrice, error) {
	tickerNew, err := GetTicker(l.GetName(), currency[0:3], currency[3:])
	if err == nil {
		return tickerNew, nil
	}

	ticker, err := l.GetTicker()
	if err != nil {
		return TickerPrice{}, err
	}

	result, ok := ticker[currency]
	if !ok {
		return TickerPrice{}, err
	}

	var tickerPrice TickerPrice
	tickerPrice.Ask = result.Ask
	tickerPrice.Bid = result.Bid
	tickerPrice.Volume = result.Volume
	tickerPrice.High = result.High
	tickerPrice.Low = result.Low
	tickerPrice.Last = result.Last
	tickerPrice.FirstCurrency = currency[0:3]
	tickerPrice.SecondCurrency = currency[3:]
	tickerPrice.CurrencyPair = tickerPrice.FirstCurrency + "_" + tickerPrice.SecondCurrency
	ProcessTicker(l.GetName(), tickerPrice.FirstCurrency, tickerPrice.SecondCurrency, tickerPrice)
	return tickerPrice, nil
}

func (l *LakeBTC) GetOrderBook(currency string) (LakeBTCOrderbook, error) {
	type Response struct {
		Bids [][]string `json:"bids"`
		Asks [][]string `json:"asks"`
	}
	path := fmt.Sprintf("%s/%s?symbol=%s", LAKEBTC_API_URL, LAKEBTC_ORDERBOOK, StringToLower(currency))
	resp := Response{}
	err := SendHTTPGetRequest(path, true, &resp)
	if err != nil {
		return LakeBTCOrderbook{}, err
	}
	orderbook := LakeBTCOrderbook{}

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
		orderbook.Bids = append(orderbook.Bids, LakeBTCOrderbookStructure{price, amount})
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
		orderbook.Asks = append(orderbook.Asks, LakeBTCOrderbookStructure{price, amount})
	}
	return orderbook, nil
}

type LakeBTCTradeHistory struct {
	Date   int64   `json:"data"`
	Price  float64 `json:"price,string"`
	Amount float64 `json:"amount,string"`
	TID    int64   `json:"tid"`
}

func (l *LakeBTC) GetTradeHistory(currency string) ([]LakeBTCTradeHistory, error) {
	path := fmt.Sprintf("%s/%s?symbol=%s", LAKEBTC_API_URL, LAKEBTC_TRADES, StringToLower(currency))
	resp := []LakeBTCTradeHistory{}
	err := SendHTTPGetRequest(path, true, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type LakeBTCAccountInfo struct {
	Balance map[string]string `json:"balance"`
	Locked  map[string]string `json:"locked"`
	Profile struct {
		Email             string `json:"email"`
		UID               string `json:"uid"`
		BTCDepositAddress string `json:"btc_deposit_addres"`
	} `json:"profile"`
}

func (l *LakeBTC) GetAccountInfo() (LakeBTCAccountInfo, error) {
	resp := LakeBTCAccountInfo{}
	err := l.SendAuthenticatedHTTPRequest(LAKEBTC_GET_ACCOUNT_INFO, "", &resp)

	if err != nil {
		return resp, err
	}

	return resp, nil
}

func (l *LakeBTC) GetExchangeAccountInfo() (ExchangeAccountInfo, error) {
	var response ExchangeAccountInfo
	response.ExchangeName = l.GetName()
	accountInfo, err := l.GetAccountInfo()
	if err != nil {
		return response, err
	}

	for x, y := range accountInfo.Balance {
		for z, w := range accountInfo.Locked {
			if z == x {
				var exchangeCurrency ExchangeAccountCurrencyInfo
				exchangeCurrency.CurrencyName = StringToUpper(x)
				exchangeCurrency.TotalValue, _ = strconv.ParseFloat(y, 64)
				exchangeCurrency.Hold, _ = strconv.ParseFloat(w, 64)
				response.Currencies = append(response.Currencies, exchangeCurrency)
			}
		}
	}
	return response, nil
}

type LakeBTCTrade struct {
	ID     int64  `json:"id"`
	Result string `json:"result"`
}

func (l *LakeBTC) Trade(orderType int, amount, price float64, currency string) (LakeBTCTrade, error) {
	resp := LakeBTCTrade{}
	params := strconv.FormatFloat(price, 'f', -1, 64) + "," + strconv.FormatFloat(amount, 'f', -1, 64) + "," + currency
	err := errors.New("")

	if orderType == 1 {
		err = l.SendAuthenticatedHTTPRequest(LAKEBTC_BUY_ORDER, params, &resp)
	} else {
		err = l.SendAuthenticatedHTTPRequest(LAKEBTC_SELL_ORDER, params, &resp)
	}

	if err != nil {
		return resp, err
	}

	if resp.Result != "order received" {
		return resp, fmt.Errorf("Unexpected result: %s", resp.Result)
	}

	return resp, nil
}

type LakeBTCOpenOrders struct {
	ID     int64   `json:"id"`
	Amount float64 `json:"amount,string"`
	Price  float64 `json:"price,string"`
	Symbol string  `json:"symbol"`
	Type   string  `json:"type"`
	At     int64   `json:"at"`
}

func (l *LakeBTC) GetOpenOrders() ([]LakeBTCOpenOrders, error) {
	orders := []LakeBTCOpenOrders{}
	err := l.SendAuthenticatedHTTPRequest(LAKEBTC_OPEN_ORDERS, "", &orders)

	if err != nil {
		return nil, err
	}
	return orders, nil
}

type LakeBTCOrders struct {
	ID             int64   `json:"id"`
	OriginalAmount float64 `json:"original_amount,string"`
	Amount         float64 `json:"amount,string"`
	Price          float64 `json:"price,string"`
	Symbol         string  `json:"symbol"`
	Type           string  `json:"type"`
	State          string  `json:"state"`
	At             int64   `json:"at"`
}

func (l *LakeBTC) GetOrders(orders []int64) ([]LakeBTCOrders, error) {
	var ordersStr []string
	for _, x := range orders {
		ordersStr = append(ordersStr, strconv.FormatInt(x, 10))
	}

	resp := []LakeBTCOrders{}
	err := l.SendAuthenticatedHTTPRequest(LAKEBTC_GET_ORDERS, JoinStrings(ordersStr, ","), &resp)

	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (l *LakeBTC) CancelOrder(orderID int64) error {
	type Response struct {
		Result bool `json:"Result"`
	}

	resp := Response{}
	params := strconv.FormatInt(orderID, 10)
	err := l.SendAuthenticatedHTTPRequest(LAKEBTC_CANCEL_ORDER, params, &resp)
	if err != nil {
		return err
	}

	if resp.Result != true {
		return errors.New("Unable to cancel order.")
	}

	return nil
}

type LakeBTCAuthenticaltedTradeHistory struct {
	Type   string  `json:"type"`
	Symbol string  `json:"symbol"`
	Amount float64 `json:"amount,string"`
	Total  float64 `json:"total,string"`
	At     int64   `json:"at"`
}

func (l *LakeBTC) GetTrades(timestamp int64) ([]LakeBTCAuthenticaltedTradeHistory, error) {
	params := ""
	if timestamp != 0 {
		params = strconv.FormatInt(timestamp, 10)
	}

	trades := []LakeBTCAuthenticaltedTradeHistory{}
	err := l.SendAuthenticatedHTTPRequest(LAKEBTC_GET_TRADES, params, &trades)
	if err != nil {
		return nil, err
	}

	return trades, nil
}

type LakeBTCExternalAccounts struct {
	ID         int64       `json:"id,string"`
	Type       string      `json:"type"`
	Address    string      `json:"address"`
	Alias      interface{} `json:"alias"`
	Currencies string      `json:"currencies"`
	State      string      `json:"state"`
	UpdatedAt  int64       `json:"updated_at,string"`
}

/* Only for BTC */
func (l *LakeBTC) GetExternalAccounts() ([]LakeBTCExternalAccounts, error) {
	resp := []LakeBTCExternalAccounts{}
	err := l.SendAuthenticatedHTTPRequest(LAKEBTC_GET_EXTERNAL_ACCOUNTS, "", &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

type LakeBTCWithdraw struct {
	ID                int64   `json:"id,string"`
	Amount            float64 `json:"amount,string"`
	Currency          string  `json:"currency"`
	Fee               float64 `json:"fee,string"`
	State             string  `json:"state"`
	Source            string  `json:"source"`
	ExternalAccountID int64   `json:"external_account_id,string"`
	At                int64   `json:"at"`
}

/* Only for BTC */
func (l *LakeBTC) CreateWithdraw(amount float64, accountID int64) (LakeBTCWithdraw, error) {
	resp := LakeBTCWithdraw{}
	params := strconv.FormatFloat(amount, 'f', -1, 64) + ",btc," + strconv.FormatInt(accountID, 10)
	err := l.SendAuthenticatedHTTPRequest(LAKEBTC_CREATE_WITHDRAW, params, &resp)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (l *LakeBTC) SendAuthenticatedHTTPRequest(method, params string, result interface{}) (err error) {
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)
	req := fmt.Sprintf("tonce=%s&accesskey=%s&requestmethod=post&id=1&method=%s&params=%s", nonce, l.APIKey, method, params)
	hmac := GetHMAC(HASH_SHA1, []byte(req), []byte(l.APISecret))

	if l.Verbose {
		log.Printf("Sending POST request to %s calling method %s with params %s\n", LAKEBTC_API_URL, method, req)
	}

	postData := make(map[string]interface{})
	postData["method"] = method
	postData["id"] = 1
	postData["params"] = SplitStrings(params, ",")

	data, err := JSONEncode(postData)
	if err != nil {
		return err
	}

	headers := make(map[string]string)
	headers["Json-Rpc-Tonce"] = nonce
	headers["Authorization"] = "Basic " + Base64Encode([]byte(l.APIKey+":"+HexEncodeToString(hmac)))
	headers["Content-Type"] = "application/json-rpc"

	resp, err := SendHTTPRequest("POST", LAKEBTC_API_URL, headers, strings.NewReader(string(data)))
	if err != nil {
		return err
	}

	if l.Verbose {
		log.Printf("Recieved raw: %s\n", resp)
	}

	type ErrorResponse struct {
		Error string `json:"error"`
	}

	errResponse := ErrorResponse{}
	err = JSONDecode([]byte(resp), &errResponse)
	if err != nil {
		return errors.New("Unable to check response for error.")
	}

	if errResponse.Error != "" {
		return errors.New(errResponse.Error)
	}

	err = JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("Unable to JSON Unmarshal response.")
	}

	return nil
}
