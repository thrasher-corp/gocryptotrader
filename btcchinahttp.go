package main

import (
	"net/http"
	"net/url"
	"strconv"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"io/ioutil"
	"fmt"
	"log"
)

const (
	BTCCHINA_API_URL = "https://api.btcchina.com/"
)

type BTCChina struct {
	APISecret, APIKey string
}

type BTCChinaTicker struct {
	High float64 `json:",string"`
	Low float64 `json:",string"`
	Buy float64 `json:",string"`
	Sell float64 `json:",string"`
	Last float64 `json:",string"`
	Vol float64 `json:",string"`
	Date int64
	Vwap float64 `json:",string"`
	Prev_close float64 `json:",string"`
	Open float64 `json:",string"`
}

func (b *BTCChina) GetTicker(symbol string) (BTCChinaTicker) {
	type Response struct {
		Ticker BTCChinaTicker
	}

	resp := Response{}
	req := fmt.Sprintf("%sdata/ticker?market=%s", BTCCHINA_API_URL, symbol)
	err := SendHTTPRequest(req, true, &resp)
	if err != nil {
		fmt.Println(err)
		return BTCChinaTicker{}
	}
	return resp.Ticker
}

func (b *BTCChina) GetTradesLast24h(symbol string) (bool) {
	req := fmt.Sprintf("%sdata/trades?market=%s", BTCCHINA_API_URL, symbol)
	err := SendHTTPRequest(req, true, nil)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (b *BTCChina) GetTradeHistory(symbol string, limit, sinceTid int64, time time.Time) (bool) {
	req := fmt.Sprintf("%sdata/historydata?market=%s", BTCCHINA_API_URL, symbol)
	v := url.Values{}

	if limit > 0 {
		v.Set("limit", strconv.FormatInt(limit, 10))
	}
	if sinceTid > 0 {
		v.Set("since", strconv.FormatInt(sinceTid, 10))
	}
	if !time.IsZero() {
		v.Set("sincetype", strconv.FormatInt(time.Unix(), 10))
	}

	values := v.Encode()
	if (len(values) > 0) {
		req += "?" + values
	}

	err := SendHTTPRequest(req, true, nil)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (b *BTCChina) GetOrderBook(symbol string, limit int) (bool) {
	req := fmt.Sprintf("%sdata/orderbook?market=%s&limit=%d", BTCCHINA_API_URL, symbol, limit)
	err := SendHTTPRequest(req, true, nil)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (b *BTCChina) GetAccountInfo() {
	err := b.SendAuthenticatedHTTPRequest("getAccountInfo", nil)

	if err != nil {
		fmt.Println(err)
	}
}

func (b *BTCChina) BuyOrder(price, amount float64) {
	params := []string{}

	if (price != 0) {
		params = append(params, strconv.FormatFloat(price, 'f', 8, 64))
	}

	err := b.SendAuthenticatedHTTPRequest("buyOrder2", nil)

	if err != nil {
		fmt.Println(err)
	}
}

func (b *BTCChina) SendAuthenticatedHTTPRequest(method string, params []string) (err error) {
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)[0:16]

	if (len(params) > 0) {
		params = nil
	}

	encoded := fmt.Sprintf("tonce=%s&accesskey=%s&requestmethod=post&id=%d&method=%s&params=%s", nonce, b.APIKey, 1, method, params)

	fmt.Println(encoded)

	hmac := hmac.New(sha1.New, []byte(b.APISecret))
	hmac.Write([]byte(encoded))
	hash := hex.EncodeToString(hmac.Sum(nil))

	postData := make(map[string]interface{})
	postData["method"] = method
	postData["params"] = []string{}
	postData["id"] = 1

	data, err := json.Marshal(postData)

	fmt.Println(string(data))

	if err != nil {
		return errors.New("Unable to JSON POST data")
	}

	log.Printf("Sending POST request to %s calling method %s with params %s\n", "https://api.btcchina.com/api_trade_v1.php", method, data)
	reqBody := strings.NewReader(string(data))

	b64 := base64.StdEncoding.EncodeToString([]byte(b.APIKey + ":" + hash))

	req, err := http.NewRequest("POST", "https://api.btcchina.com/api_trade_v1.php", reqBody)

	if err != nil {
		return err
	}

	req.Header.Add("Content-type", "application/json-rpc")
	req.Header.Add("Authorization", "Basic " + b64)
	req.Header.Add("Json-Rpc-Tonce", nonce)

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return errors.New("PostRequest: Unable to send request")
	}

	contents, _ := ioutil.ReadAll(resp.Body)
	log.Printf("Recv'd :%s", string(contents))
	resp.Body.Close()
	return nil

}