package main

import (
	"net/http"
	"net/url"
	"strconv"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"io/ioutil"
	"fmt"
)

const (
	BTCE_API_URL = "https://btc-e.com/tapi"
	BTCE_GET_INFO = "getInfo"
	BTCE_TRANSACTION_HISTORY = "TransHistory"
	BTCE_TRADE_HISTORY = "TradeHistory"
	BTCE_ACTIVE_ORDERS = "ActiveOrders"
	BTCE_TRADE = "Trade"
	BTCE_CANCEL_ORDER = "CancelOrder"
)

type BTCE struct {
	APIKey, APISecret string
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
	Server_time int64
}

func (b *BTCE) GetTicker(symbol string) (BTCeTicker) {
	type Response struct {
		Ticker BTCeTicker
	}

	response := Response{}
	req := fmt.Sprintf("https://btc-e.com/api/2/%s/ticker", symbol)
	err := SendHTTPRequest(req, true, &response)
	if err != nil {
		fmt.Println(err)
		return BTCeTicker{}
	}
	return response.Ticker
}

func (b *BTCE) GetInfo() {
	err := b.SendAuthenticatedHTTPRequest(BTCE_GET_INFO, url.Values{})

	if err != nil {
		fmt.Println(err)
	}
}

func (b *BTCE) GetActiveOrders(pair string) {
	req := url.Values{}
	req.Add("pair", pair)

	err := b.SendAuthenticatedHTTPRequest(BTCE_ACTIVE_ORDERS, req)

	if err != nil {
		fmt.Println(err)
	}
}

func (b *BTCE) CancelOrder(OrderID int64) {
	req := url.Values{}
	req.Add("order_id", strconv.FormatInt(OrderID, 10))

	err := b.SendAuthenticatedHTTPRequest(BTCE_CANCEL_ORDER, req)

	if err != nil {
		fmt.Println(err)
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
		fmt.Println(err)
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
		fmt.Println(err)
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
		fmt.Println(err)
	}
}

func (b *BTCE) SendAuthenticatedHTTPRequest(method string, values url.Values) (err error) {
	nonce := strconv.FormatInt(time.Now().Unix(), 10)
	values.Set("nonce", nonce)
	values.Set("method", method)

	hmac := hmac.New(sha512.New, []byte(b.APISecret))
	encoded := values.Encode()
	hmac.Write([]byte(encoded))


	fmt.Printf("Sending POST request to %s calling method %s with params %s\n", BTCE_API_URL, method, encoded)
	reqBody := strings.NewReader(encoded)

	req, err := http.NewRequest("POST", BTCE_API_URL, reqBody)

	if err != nil {
		return err
	}

	req.Header.Add("Key", b.APIKey)
	req.Header.Add("Sign", hex.EncodeToString(hmac.Sum(nil)))
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