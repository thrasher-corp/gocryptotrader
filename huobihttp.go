package main

import (
	"net/http"
	"net/url"
	"errors"
	"strings"
	"io/ioutil"
	"strconv"
	"time"
	"fmt"
	"log"
)

const (
	HUOBI_API_URL = "https://api.huobi.com/apiv2.php"
	HUOBI_API_VERSION = "2"
)

type HUOBI struct {
	Name string
	Enabled bool
	Verbose bool
	AccessKey, SecretKey string
	Fee float64
}

type HuobiTicker struct {
	High float64
	Low float64
	Last float64 
	Vol float64 
	Buy float64 
	Sell float64 
}

type HuobiTickerResponse struct {
	Time string
	Ticker HuobiTicker
}

func (h *HUOBI) SetDefaults() {
	h.Name = "Huobi"
	h.Enabled = true
	h.Fee = 0
	h.Verbose = false
}

func (h *HUOBI) GetName() (string) {
	return h.Name
}

func (h *HUOBI) SetEnabled(enabled bool) {
	h.Enabled = enabled
}

func (h *HUOBI) IsEnabled() (bool) {
	return h.Enabled
}

func (h *HUOBI) SetAPIKeys(apiKey, apiSecret string) {
	h.AccessKey = apiKey
	h.SecretKey = apiSecret
}

func (h *HUOBI) GetFee() (float64) {
	return h.Fee
}

func (h *HUOBI) GetTicker(symbol string) (HuobiTicker) {
	resp := HuobiTickerResponse{}
	path := fmt.Sprintf("http://market.huobi.com/staticmarket/ticker_%s_json.js", symbol)
	err := SendHTTPRequest(path, true, &resp)

	if err != nil {
		log.Println(err)
		return HuobiTicker{}
	}
	return resp.Ticker
}

func (h *HUOBI) GetOrderBook(symbol string) (bool) {
	path := fmt.Sprintf("http://market.huobi.com/staticmarket/depth_%s_json.js", symbol)
	err := SendHTTPRequest(path, true, nil)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (h *HUOBI) GetAccountInfo() {
	err := h.SendAuthenticatedRequest("get_account_info", url.Values{})

	if err != nil {
		log.Println(err)
	}
}


func (h *HUOBI) GetOrders(coinType int) {
	values := url.Values{}
	values.Set("coin_type", strconv.Itoa(coinType))
	err := h.SendAuthenticatedRequest("get_orders", values)

	if err != nil {
		log.Println(err)
	}
}

func (h *HUOBI) GetOrderInfo(orderID, coinType int) {
	values := url.Values{}
	values.Set("id", strconv.Itoa(orderID))
	values.Set("coin_type", strconv.Itoa(coinType))
	err := h.SendAuthenticatedRequest("order_info", values)

	if err != nil {
		log.Println(err)
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
		log.Println(err)
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
		log.Println(err)
	}
}

func (h *HUOBI) CancelOrder(orderID, coinType int) {
	values := url.Values{}
	values.Set("coin_type", strconv.Itoa(coinType))
	values.Set("id", strconv.Itoa(orderID))
	err := h.SendAuthenticatedRequest("cancel_order", values)

	if err != nil {
		log.Println(err)
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
		log.Println(err)
	}
}

func (h *HUOBI) GetNewDealOrders(coinType int) {
	values := url.Values{}
	values.Set("coin_type", strconv.Itoa(coinType))
	err := h.SendAuthenticatedRequest("get_new_deal_orders", values)

	if err != nil {
		log.Println(err)
	}
}

func (h *HUOBI) GetOrderIDByTradeID(coinType, orderID int) {
	values := url.Values{}
	values.Set("coin_type", strconv.Itoa(coinType))
	values.Set("trade_id", strconv.Itoa(orderID))
	err := h.SendAuthenticatedRequest("get_order_id_by_trade_id", values)

	if err != nil {
		log.Println(err)
	}
}

func (h *HUOBI) SendAuthenticatedRequest(method string, v url.Values) (error) {
	v.Set("access_key", h.AccessKey)
	v.Set("created", strconv.FormatInt(time.Now().Unix(), 10))
	v.Set("method", method)
	hash := GetMD5([]byte(v.Encode() + "&secret_key=" + h.SecretKey))
	v.Set("sign", strings.ToLower(HexEncodeToString(hash)))
	encoded := v.Encode()

	if h.Verbose {
		log.Printf("Sending POST request to %s with params %s\n", HUOBI_API_URL, encoded)
	}

	req, err := http.NewRequest("POST", HUOBI_API_URL, strings.NewReader(encoded))

	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	defer resp.Body.Close()

	if err != nil {
		return errors.New("PostRequest: Unable to send request")
	}

	contents, _ := ioutil.ReadAll(resp.Body)

	if h.Verbose {
		log.Printf("Recieved raw: %s\n", string(contents))
	}
	
	return nil
}