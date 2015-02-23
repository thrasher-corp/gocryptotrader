package main

import (
	"net/http"
	"net/url"
	"strconv"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"io/ioutil"
	"fmt"
)

const (
	LAKEBTC_API_URL = "https://www.LakeBTC.com/api_v1/"
	LAKEBTC_TICKER = "ticker"
	LAKEBTC_ORDERBOOK = "bcorderbook"
	LAKEBTC_ORDERBOOK_CNY = "bcorderbook_cny"
	LAKEBTC_TRADES = "bctrades"
	LAKEBTC_GET_ACCOUNT_INFO = "getAccountInfo"
	LAKEBTC_BUY_ORDER = "buyOrder"
	LAKEBTC_SELL_ORDER = "sellOrder"
	LAKEBTC_GET_ORDERS = "getOrders"
	LAKEBTC_CANCEL_ORDER = "cancelOrder"
	LAKEBTC_GET_TRADES = "getTrades"
)

type LakeBTC struct {
	Name string
	Enabled bool
	Email, APISecret string
}

type LakeBTCTicker struct {
	Last float64
	Bid float64
	Ask float64
	High float64
	Low float64
}

type LakeBTCTickerResponse struct {
	USD LakeBTCTicker
	CNY LakeBTCTicker
}

func (l *LakeBTC) SetDefaults() {
	l.Name = "LakeBTC"
	l.Enabled = true
}

func (l *LakeBTC) GetName() (string) {
	return l.Name
}

func (l *LakeBTC) SetEnabled(enabled bool) {
	l.Enabled = enabled
}

func (l *LakeBTC) IsEnabled() (bool) {
	return l.Enabled
}

func (l *LakeBTC) SetAPIKeys(apiKey, apiSecret string) {
	l.Email = apiKey
	l.APISecret = apiSecret
}

func (l *LakeBTC) GetTicker() (LakeBTCTickerResponse) {
	response := LakeBTCTickerResponse{}
	err := SendHTTPRequest(LAKEBTC_API_URL + LAKEBTC_TICKER, true, &response)
	if err != nil {
		fmt.Println(err)
		return response
	}
	return response
}

func (l *LakeBTC) GetOrderBook(currency string) (bool) {
	req := LAKEBTC_ORDERBOOK
	if currency == "CNY" {
		req = LAKEBTC_ORDERBOOK_CNY
	}

	err := SendHTTPRequest(LAKEBTC_API_URL + req, true, nil)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (l *LakeBTC) GetTradeHistory() (bool) {
	err := SendHTTPRequest(LAKEBTC_API_URL + LAKEBTC_TRADES, true, nil)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (l *LakeBTC) GetAccountInfo() {
	err := l.SendAuthenticatedHTTPRequest(LAKEBTC_GET_ACCOUNT_INFO, "")

	if err != nil {
		fmt.Println(err)
	}
}

func (l *LakeBTC) Trade(orderType int, amount, price float64, currency string) {
	params := strconv.FormatFloat(price, 'f', 8, 64) + "," + strconv.FormatFloat(amount, 'f', 8, 64) + "," + currency
	err := errors.New("")

	if orderType == 0 {
		err = l.SendAuthenticatedHTTPRequest(LAKEBTC_BUY_ORDER, params)
	} else {
		err = l.SendAuthenticatedHTTPRequest(LAKEBTC_SELL_ORDER, params)
	}

	if err != nil {
		fmt.Println(err)
	}
}

func (l *LakeBTC) GetOrders() {
	err := l.SendAuthenticatedHTTPRequest(LAKEBTC_GET_ORDERS, "")
	if err != nil {
		fmt.Println(err)
	}
}

func (l *LakeBTC) CancelOrder(orderID int64) {
	params := strconv.FormatInt(orderID, 10)
	err := l.SendAuthenticatedHTTPRequest(LAKEBTC_CANCEL_ORDER, params)
	if err != nil {
		fmt.Println(err)
	}
}

func (l *LakeBTC) GetTrades(timestamp time.Time) {
	params := ""

	if !timestamp.IsZero() {
		params = strconv.FormatInt(timestamp.Unix(), 10)
	}
	
	err := l.SendAuthenticatedHTTPRequest(LAKEBTC_GET_TRADES, params)
	if err != nil {
		fmt.Println(err)
	}
}

func (l *LakeBTC) SendAuthenticatedHTTPRequest(method, params string) (err error) {
	nonce := strconv.FormatInt(time.Now().Unix(), 10)
	v := url.Values{}
	v.Set("tnonce", nonce)
	v.Set("accesskey", l.Email)
	v.Set("requestmethod", "POST")
	v.Set("id", nonce)
	v.Set("method", method)
	v.Set("params", params)

	hmac := hmac.New(sha256.New, []byte(l.APISecret))
	encoded := v.Encode()
	hmac.Write([]byte(encoded))


	fmt.Printf("Sending POST request to %s calling method %s with params %s\n", LAKEBTC_API_URL, method, encoded)
	reqBody := strings.NewReader(encoded)
	hash := hex.EncodeToString(hmac.Sum(nil))
	b64 := base64.StdEncoding.EncodeToString([]byte(l.Email + ":" + hash))

	req, err := http.NewRequest("POST", LAKEBTC_API_URL, reqBody)

	if err != nil {
		return err
	}

	req.Header.Add("Json-Rpc-Tonce", nonce)
	req.Header.Add("Authorization: Basic", b64)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return errors.New("PostRequest: Unable to send request")
	}

	contents, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("Recieved raw: %s\n", string(contents))
	resp.Body.Close()
	return nil

}