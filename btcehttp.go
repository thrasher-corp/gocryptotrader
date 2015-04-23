package main

import (
	"net/url"
	"strconv"
	"strings"
	"time"
	"fmt"
	"log"
)

const (
	BTCE_API_PUBLIC_URL = "https://btc-e.com/api"
	BTCE_API_PRIVATE_URL = "https://btc-e.com/tapi"
	BTCE_API_PUBLIC_VERSION = "3"
	BTCE_API_PRIVATE_VERSION = "1"
	BTCE_INFO = "info"
	BTCE_TICKER = "ticker"
	BTCE_DEPTH = "depth"
	BTCE_TRADES = "trades"
	BTCE_ACCOUNT_INFO = "getInfo"
	BTCE_TRANSACTION_HISTORY = "TransHistory"
	BTCE_TRADE_HISTORY = "TradeHistory"
	BTCE_ACTIVE_ORDERS = "ActiveOrders"
	BTCE_TRADE = "Trade"
	BTCE_CANCEL_ORDER = "CancelOrder"
)

type BTCE struct {
	Name string
	Enabled bool
	Verbose bool
	Websocket bool
	RESTPollingDelay time.Duration
	APIKey, APISecret string
	Fee float64
}

type BTCeTicker struct {
	High float64
	Low float64
	Avg float64
	Vol float64
	Vol_cur float64
	Last float64
	Buy float64
	Sell float64
	Updated int64
}

type BTCEOrderbook struct {
	Asks[][]float64 `json:"asks"`
	Bids[][]float64 `json:"bids"`
}

type BTCETrades struct {
	Type string `json:"type"`
	Price float64 `json:"bid"`
	Amount float64 `json:"amount"`
	TID int64 `json:"tid"`
	Timestamp int64 `json:"timestamp"`
}

func (b *BTCE) SetDefaults() {
	b.Name = "BTCE"
	b.Enabled = true
	b.Fee = 0.2
	b.Verbose = false
	b.Websocket = false
	b.RESTPollingDelay = 10
}

func (b *BTCE) GetName() (string) {
	return b.Name
}

func (b *BTCE) SetEnabled(enabled bool) {
	b.Enabled = enabled
}

func (b *BTCE) IsEnabled() (bool) {
	return b.Enabled
}

func (b *BTCE) SetAPIKeys(apiKey, apiSecret string) {
	b.APIKey = apiKey
	b.APISecret = apiSecret
}

func (b *BTCE) GetFee() (float64) {
	return b.Fee
}

func (b *BTCE) Run() {
	if b.Verbose {
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
	}

	for b.Enabled {
		go func() {
			BTCeBTC := b.GetTicker("btc_usd")
			log.Printf("BTC-e BTC: Last %f High %f Low %f Volume %f\n", BTCeBTC.Last, BTCeBTC.High, BTCeBTC.Low, BTCeBTC.Vol_cur)
			AddExchangeInfo(b.GetName(), "BTC", BTCeBTC.Last, BTCeBTC.Vol_cur)
		}()

		go func() {
			BTCeLTC := b.GetTicker("ltc_usd")
			log.Printf("BTC-e LTC: Last %f High %f Low %f Volume %f\n", BTCeLTC.Last, BTCeLTC.High, BTCeLTC.Low, BTCeLTC.Vol_cur)
			AddExchangeInfo(b.GetName(), "LTC", BTCeLTC.Last, BTCeLTC.Vol_cur)
		}()
		time.Sleep(time.Second * b.RESTPollingDelay)
	}
}

func (b *BTCE) GetInfo() {
	req := fmt.Sprintf("%s/%s/%s/", BTCE_API_PUBLIC_URL, BTCE_API_PUBLIC_VERSION, BTCE_INFO)
	err := SendHTTPGetRequest(req, true, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCE) GetTicker(symbol string) (BTCeTicker) {
	type Response struct {
		Data map[string]BTCeTicker
	}

	response := Response{}
	req := fmt.Sprintf("%s/%s/%s/%s", BTCE_API_PUBLIC_URL, BTCE_API_PUBLIC_VERSION, BTCE_TICKER, symbol)
	err := SendHTTPGetRequest(req, true, &response.Data)

	if err != nil {
		log.Println(err)
		return BTCeTicker{}
	}
	return response.Data[symbol]
}

func (b *BTCE) GetDepth(symbol string) () {
	type Response struct {
		Data map[string]BTCEOrderbook
	}

	response := Response{}
	req := fmt.Sprintf("%s/%s/%s/%s", BTCE_API_PUBLIC_URL, BTCE_API_PUBLIC_VERSION, BTCE_DEPTH, symbol)
	err := SendHTTPGetRequest(req, true, &response.Data)

	if err != nil {
		log.Println(err)
		return
	}

	depth := response.Data[symbol]
	log.Println(depth)
}

func (b *BTCE) GetTrades(symbol string) () {
	type Response struct {
		Data map[string][]BTCETrades
	}

	response := Response{}
	req := fmt.Sprintf("%s/%s/%s/%s", BTCE_API_PUBLIC_URL, BTCE_API_PUBLIC_VERSION, BTCE_TRADES, symbol)
	err := SendHTTPGetRequest(req, true, &response.Data)

	if err != nil {
		log.Println(err)
	}

	trades := response.Data[symbol]
	log.Println(trades)
}

func (b *BTCE) GetAccountInfo() {
	err := b.SendAuthenticatedHTTPRequest(BTCE_ACCOUNT_INFO, url.Values{})

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCE) GetActiveOrders(pair string) {
	req := url.Values{}
	req.Add("pair", pair)

	err := b.SendAuthenticatedHTTPRequest(BTCE_ACTIVE_ORDERS, req)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCE) CancelOrder(OrderID int64) {
	req := url.Values{}
	req.Add("order_id", strconv.FormatInt(OrderID, 10))

	err := b.SendAuthenticatedHTTPRequest(BTCE_CANCEL_ORDER, req)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCE) Trade(pair, orderType string, amount, price float64) {
	req := url.Values{}
	req.Add("pair", pair)
	req.Add("type", orderType)
	req.Add("amount", strconv.FormatFloat(amount, 'f', 8, 64))
	req.Add("rate", strconv.FormatFloat(price, 'f', 2, 64))

	err := b.SendAuthenticatedHTTPRequest(BTCE_TRADE, req)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCE) GetTransactionHistory(TIDFrom, Count, TIDEnd int64, order, since, end string) {
	req := url.Values{}
	req.Add("from", strconv.FormatInt(TIDFrom, 10))
	req.Add("count", strconv.FormatInt(Count, 10))
	req.Add("from_id", strconv.FormatInt(TIDFrom, 10))
	req.Add("end_id", strconv.FormatInt(TIDFrom, 10))
	req.Add("order", order)
	req.Add("since", order)
	req.Add("end", order)

	err := b.SendAuthenticatedHTTPRequest(BTCE_TRANSACTION_HISTORY, req)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCE) GetTradeHistory(TIDFrom, Count, TIDEnd int64, order, since, end, pair string) {
	req := url.Values{}
	
	req.Add("from", strconv.FormatInt(TIDFrom, 10))
	req.Add("count", strconv.FormatInt(Count, 10))
	req.Add("from_id", strconv.FormatInt(TIDFrom, 10))
	req.Add("end_id", strconv.FormatInt(TIDFrom, 10))
	req.Add("order", order)
	req.Add("since", order)
	req.Add("end", order)
	req.Add("pair", pair)

	err := b.SendAuthenticatedHTTPRequest(BTCE_TRANSACTION_HISTORY, req)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCE) SendAuthenticatedHTTPRequest(method string, values url.Values) (err error) {
	nonce := strconv.FormatInt(time.Now().Unix(), 10)
	values.Set("nonce", nonce)
	values.Set("method", method)

	encoded := values.Encode()
	hmac := GetHMAC(HASH_SHA512, []byte(encoded), []byte(b.APISecret))

	if b.Verbose {
		log.Printf("Sending POST request to %s calling method %s with params %s\n", BTCE_API_PRIVATE_URL, method, encoded)
	}

	headers := make(map[string]string)
	headers["Key"] = b.APIKey
	headers["Sign"] = HexEncodeToString(hmac)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := SendHTTPRequest("POST", BTCE_API_PRIVATE_URL, headers, strings.NewReader(encoded))

	if err != nil {
		return err
	}

	if b.Verbose {
		log.Printf("Recieved raw: %s\n",resp)
	}
	
	return nil
}