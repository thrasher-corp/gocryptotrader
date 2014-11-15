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
	"fmt"
)

const (
	OKCOIN_API_URL = "https://www.okcoin.com/api/v1/"
	OKCOIN_API_URL_CHINA = "https://www.okcoin.cn/api/v1/"
)

type OKCoin struct {
	APIUrl, PartnerID, SecretKey string
}

type OKCoinTicker struct {
	Buy string
	High string
	Last string
	Low string
	Sell string
	Vol string
}

type OKCoinTickerResponse struct {
	Date string
	Ticker OKCoinTicker
}

func (o *OKCoin) SetURL(url string) {
	o.APIUrl = url
}

func (o *OKCoin) GetTicker(symbol string) (OKCoinTicker) {
	resp := OKCoinTickerResponse{}
	path := fmt.Sprintf("ticker.do?symbol=%s&ok=1", symbol)
	err := SendHTTPRequest(o.APIUrl + path, true, &resp)

	if err != nil {
		fmt.Println(err)
		return OKCoinTicker{}
	}
	return resp.Ticker
}

func (o *OKCoin) GetOrderBook(symbol string) (bool) {
	path := "depth.do?symbol=" + symbol
	err := SendHTTPRequest(o.APIUrl + path, true, nil)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (o *OKCoin) GetTradeHistory(symbol string) (bool) {
	path := "trades.do?symbol=" + symbol
	err := SendHTTPRequest(o.APIUrl + path, true, nil)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (o *OKCoin) GetUserInfo() {
	v := url.Values{}
	v.Set("partner", o.PartnerID)
	err := o.SendAuthenticatedHTTPRequest("userinfo.do", v)

	if err != nil {
		fmt.Println(err)
	}
}

func (o *OKCoin) Trade(amount, price float64, symbol, orderType string) {
	v := url.Values{}
	v.Set("partner", o.PartnerID)
	v.Set("amount", strconv.FormatFloat(amount, 'f', 8, 64))
	v.Set("price",  strconv.FormatFloat(price, 'f', 8, 64))
	v.Set("symbol", symbol)
	v.Set("type", orderType)

	err := o.SendAuthenticatedHTTPRequest("trade.do", v)

	if err != nil {
		fmt.Println(err)
	}
}

func (o *OKCoin) BatchTrade(orderData string, symbol, orderType string) {
	v := url.Values{} //to-do batch trade support for orders_data
	v.Set("partner", o.PartnerID)
	v.Set("orders_data", orderData)
	v.Set("symbol", symbol)
	v.Set("type", orderType)

	err := o.SendAuthenticatedHTTPRequest("batch_trade.do", v)

	if err != nil {
		fmt.Println(err)
	}
}

func (o *OKCoin) CancelOrder(orderID int64, symbol string) {
	v := url.Values{}
	v.Set("partner", o.PartnerID)
	v.Set("orders_id", strconv.FormatInt(orderID, 10))
	v.Set("symbol", symbol)

	err := o.SendAuthenticatedHTTPRequest("cancel_order.do", v)

	if err != nil {
		fmt.Println(err)
	}
}

func (o *OKCoin) GetOrderInfo(orderID int64, symbol string) {
	v := url.Values{}
	v.Set("partner", o.PartnerID)
	v.Set("orders_id", strconv.FormatInt(orderID, 10))
	v.Set("symbol", symbol)

	err := o.SendAuthenticatedHTTPRequest("order_info.do", v)

	if err != nil {
		fmt.Println(err)
	}
}

func (o *OKCoin) GetOrdersInfo(orderID int64, orderType string, symbol string) {
	v := url.Values{}
	v.Set("partner", o.PartnerID)
	v.Set("orders_id", strconv.FormatInt(orderID, 10))
	v.Set("type", orderType)
	v.Set("symbol", symbol)

	err := o.SendAuthenticatedHTTPRequest("orders_info.do", v)

	if err != nil {
		fmt.Println(err)
	}
}

func (o *OKCoin) GetOrderHistory(orderID, pageLength, currentPage int64, orderType string, status, symbol string) {
	v := url.Values{}
	v.Set("partner", o.PartnerID)
	v.Set("orders_id", strconv.FormatInt(orderID, 10))
	v.Set("type", orderType)
	v.Set("symbol", symbol)
	v.Set("status", status)
	v.Set("current_page", strconv.FormatInt(currentPage, 10))
	v.Set("page_length", strconv.FormatInt(pageLength, 10))

	err := o.SendAuthenticatedHTTPRequest("orders_info.do", v)

	if err != nil {
		fmt.Println(err)
	}
}

func (o *OKCoin) SendAuthenticatedHTTPRequest(method string, v url.Values) (err error) {
	hasher := md5.New()
	hasher.Write([]byte(v.Encode() + "&secret_key=" + o.SecretKey))
	signature := strings.ToUpper(hex.EncodeToString(hasher.Sum(nil)))

	v.Set("sign", signature)
	encoded := v.Encode() + "&partner=" + o.PartnerID

	fmt.Printf("Signature: %s\n", signature)
	path := o.APIUrl + method
	fmt.Printf("Sending POST request to %s with params %s\n", path, encoded)

	reqBody := strings.NewReader(encoded)
	req, err := http.NewRequest("POST", path, reqBody)

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