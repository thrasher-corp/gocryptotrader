package main

import (
	"net/http"
	"net/url"
	"crypto/md5"
	"errors"
	"strings"
	"encoding/hex"
	"io/ioutil"
	"strconv"
	"time"
	"fmt"
)

const (
	HUOBI_API_URL = "https://api.huobi.com/apiv2.php"
)

type HUOBI struct {
	AccessKey, SecretKey string
}

type HuobiTicker struct {
	High float64 `json:",string"`
	Low float64 `json:",string"`
	Last float64 `json:",string"`
	Vol float64
	Buy float64 `json:",string"`
	Sell float64 `json:",string"`
}

type HuobiTickerResponse struct {
	Time int64
	Ticker HuobiTicker
}

func (h *HUOBI) GetTicker(symbol string) (HuobiTicker) {
	resp := HuobiTickerResponse{}
	path := fmt.Sprintf("http://market.huobi.com/staticmarket/ticker_%s_json.js", symbol)
	err := SendHTTPRequest(path, true, &resp)

	if err != nil {
		fmt.Println(err)
		return HuobiTicker{}
	}
	return resp.Ticker
}

func (h *HUOBI) GetOrderBook(symbol string) (bool) {
	path := fmt.Sprintf("http://market.huobi.com/staticmarket/depth_%s_json.js", symbol)
	err := SendHTTPRequest(path, true, nil)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (h *HUOBI) GetAccountInfo() {
	err := h.SendAuthenticatedRequest("get_account_info", url.Values{})

	if err != nil {
		fmt.Println(err)
	}
}


func (h *HUOBI) GetOrders(coinType int) {
	values := url.Values{}
	values.Set("coin_type", strconv.Itoa(coinType))
	err := h.SendAuthenticatedRequest("get_orders", values)

	if err != nil {
		fmt.Println(err)
	}
}

func (h *HUOBI) GetOrderInfo(orderID, coinType int) {
	values := url.Values{}
	values.Set("id", strconv.Itoa(orderID))
	values.Set("coin_type", strconv.Itoa(coinType))
	err := h.SendAuthenticatedRequest("order_info", values)

	if err != nil {
		fmt.Println(err)
	}
}

func (h *HUOBI) Trade(orderType string, coinType int, price, amount float64) {
	values := url.Values{}
	if orderType != "buy" {
		orderType = "sell"
	}
	values.Set("coin_type", strconv.Itoa(coinType))
	values.Set("amount", strconv.FormatFloat(amount, 'f', 8, 64))
	values.Set("price",  strconv.FormatFloat(price, 'f', 8, 64))
	err := h.SendAuthenticatedRequest(orderType, values)

	if err != nil {
		fmt.Println(err)
	}
}

func (h *HUOBI) MarketTrade(orderType string, coinType int, price, amount float64) {
	values := url.Values{}
	if orderType != "buy_market" {
		orderType = "sell_market"
	}
	values.Set("coin_type", strconv.Itoa(coinType))
	values.Set("amount", strconv.FormatFloat(amount, 'f', 8, 64))
	values.Set("price",  strconv.FormatFloat(price, 'f', 8, 64))
	err := h.SendAuthenticatedRequest(orderType, values)

	if err != nil {
		fmt.Println(err)
	}
}

func (h *HUOBI) CancelOrder(orderID, coinType int) {
	values := url.Values{}
	values.Set("coin_type", strconv.Itoa(coinType))
	values.Set("id", strconv.Itoa(orderID))
	err := h.SendAuthenticatedRequest("cancel_order", values)

	if err != nil {
		fmt.Println(err)
	}
}

func (h *HUOBI) ModifyOrder(orderType string, coinType, orderID int, price, amount float64) {
	values := url.Values{}
	values.Set("coin_type", strconv.Itoa(coinType))
	values.Set("id", strconv.Itoa(orderID))
	values.Set("amount", strconv.FormatFloat(amount, 'f', 8, 64))
	values.Set("price",  strconv.FormatFloat(price, 'f', 8, 64))
	err := h.SendAuthenticatedRequest("modify_order", values)

	if err != nil {
		fmt.Println(err)
	}
}

func (h *HUOBI) GetNewDealOrders(coinType int) {
	values := url.Values{}
	values.Set("coin_type", strconv.Itoa(coinType))
	err := h.SendAuthenticatedRequest("get_new_deal_orders", values)

	if err != nil {
		fmt.Println(err)
	}
}

func (h *HUOBI) GetOrderIDByTradeID(coinType, orderID int) {
	values := url.Values{}
	values.Set("coin_type", strconv.Itoa(coinType))
	values.Set("trade_id", strconv.Itoa(orderID))
	err := h.SendAuthenticatedRequest("get_order_id_by_trade_id", values)

	if err != nil {
		fmt.Println(err)
	}
}

func (h *HUOBI) SendAuthenticatedRequest(method string, v url.Values) (error) {
	v.Set("access_key", h.AccessKey)
	v.Set("created", strconv.FormatInt(time.Now().Unix(), 10))
	v.Set("method", method)

	hasher := md5.New()
	hasher.Write([]byte(v.Encode() + "&secret_key=" + h.SecretKey))
	signature := strings.ToUpper(hex.EncodeToString(hasher.Sum(nil)))
	v.Set("sign", signature)


	encoded := v.Encode()
	fmt.Printf("Signature: %s\n", signature)
	fmt.Printf("Sending POST request to %s with params %s\n", HUOBI_API_URL, encoded)

	reqBody := strings.NewReader(encoded)
	req, err := http.NewRequest("POST", HUOBI_API_URL, reqBody)

	if err != nil {
		return err
	}

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