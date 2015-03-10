package main

import (
	"net/http"
	"net/url"
	"strconv"
	"crypto/sha512"
	"errors"
	"strings"
	"time"
	"io/ioutil"
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

func (b *BTCE) GetInfo() {
	req := fmt.Sprintf("%s/%s/%s/", BTCE_API_PUBLIC_URL, BTCE_API_PUBLIC_VERSION, BTCE_INFO)
	err := SendHTTPRequest(req, true, nil)

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
	err := SendHTTPRequest(req, true, &response.Data)

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
	err := SendHTTPRequest(req, true, &response.Data)

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
	err := SendHTTPRequest(req, true, &response.Data)

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
	hmac := GetHMAC(sha512.New, []byte(encoded), []byte(b.APISecret))

	if b.Verbose {
		log.Printf("Sending POST request to %s calling method %s with params %s\n", BTCE_API_PRIVATE_URL, method, encoded)
	}

	req, err := http.NewRequest("POST", BTCE_API_PRIVATE_URL, strings.NewReader(encoded))

	if err != nil {
		return err
	}

	req.Header.Add("Key", b.APIKey)
	req.Header.Add("Sign", HexEncodeToString(hmac))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return errors.New("PostRequest: Unable to send request")
	}

	contents, _ := ioutil.ReadAll(resp.Body)

	if b.Verbose {
		log.Printf("Recieved raw: %s\n", string(contents))
	}
	
	resp.Body.Close()
	return nil

}