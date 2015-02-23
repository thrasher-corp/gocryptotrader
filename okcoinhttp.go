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
	Name string
	Enabled bool
	APIUrl, PartnerID, SecretKey string
}

type OKCoinTicker struct {
	Buy float64 `json:",string"`
	High float64 `json:",string"`
	Last float64 `json:",string"`
	Low float64 `json:",string"`
	Sell float64 `json:",string"`
	Vol float64 `json:",string"`
}

type OKCoinTickerResponse struct {
	Date string
	Ticker OKCoinTicker
}
type OKCoinFuturesTicker struct {
	Last float64
	Buy float64
	Sell float64
	High float64
	Low float64
	Vol float64
	Contract_ID float64
	Unit_Amount float64
}

type OKCoinFuturesTickerResponse struct {
	Date string
	Ticker OKCoinFuturesTicker
}

func (o *OKCoin) SetDefaults() {
	if (o.APIUrl == OKCOIN_API_URL) {
		o.Name = "OKCOIN International"
	} else if (o.APIUrl == OKCOIN_API_URL_CHINA) {
		o.Name = "OKCOIN China"
	}
	o.Enabled = true
}

func (o *OKCoin) GetName() (string) {
	return o.Name
}

func (o *OKCoin) SetEnabled(enabled bool) {
	o.Enabled = enabled
}

func (o *OKCoin) IsEnabled() (bool) {
	return o.Enabled
}

func (o *OKCoin) SetURL(url string) {
	o.APIUrl = url
}

func (o *OKCoin) SetAPIKeys(apiKey, apiSecret string) {
	o.PartnerID = apiKey
	o.SecretKey = apiSecret
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

func (o *OKCoin) GetFuturesTicker(symbol, contractType string) (OKCoinFuturesTicker) {
	resp := OKCoinFuturesTickerResponse{}
	path := fmt.Sprintf("future_ticker.do?symbol=%s&contract_type=%s", symbol, contractType)
	err := SendHTTPRequest(o.APIUrl + path, true, &resp)
	if err != nil {
		fmt.Println(err)
		return OKCoinFuturesTicker{}
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

func (o *OKCoin) GetFuturesDepth(symbol, contractType string) (bool) {
	path := fmt.Sprintf("future_depth.do?symbol=%s&contract_type=%s", symbol, contractType)
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

func (o *OKCoin) GetFuturesTrades(symbol, contractType string) (bool) {
	path := fmt.Sprintf("future_trades.do?symbol=%s&contract_type=%s", symbol, contractType)
	err := SendHTTPRequest(o.APIUrl + path, true, nil)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (o *OKCoin) GetFuturesIndex(symbol string) (bool) {
	path := "future_index.do?symbol=" + symbol
	err := SendHTTPRequest(o.APIUrl + path, true, nil)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (o *OKCoin) GetFuturesExchangeRate() (bool) {
	err := SendHTTPRequest(o.APIUrl + "exchange_rate.do", true, nil)
	if err != nil {
		fmt.Println(err)
	}
	return true
}

func (o *OKCoin) GetFuturesEstimatedPrice(symbol string) (bool) {
	path := "future_estimated_price.do?symbol=" + symbol
	err := SendHTTPRequest(o.APIUrl + path, true, nil)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (o *OKCoin) GetFuturesTradeHistory(symbol, date string, since int64) (bool) {
	path := fmt.Sprintf("future_trades.do?symbol=%s&date%s&since=%d", symbol, date, since)
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

func (o *OKCoin) GetFuturesUserInfo() {
	v := url.Values{}
	v.Set("partner", o.PartnerID)
	err := o.SendAuthenticatedHTTPRequest("future_userinfo.do", v)

	if err != nil {
		fmt.Println(err)
	}
}

func (o *OKCoin) GetFuturesPosition(symbol, contractType string) {
	v := url.Values{}
	v.Set("partner", o.PartnerID)
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	err := o.SendAuthenticatedHTTPRequest("future_userinfo.do", v)

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

func (o *OKCoin) FuturesTrade(amount, price float64, matchPrice, leverage int64, symbol, contractType, orderType string) {
	v := url.Values{}
	v.Set("partner", o.PartnerID)
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	v.Set("price",  strconv.FormatFloat(price, 'f', 8, 64))
	v.Set("amount", strconv.FormatFloat(amount, 'f', 8, 64))
	v.Set("type", orderType)
	v.Set("match_price", strconv.FormatInt(matchPrice, 10))
	v.Set("lever_rate", strconv.FormatInt(leverage, 10))

	err := o.SendAuthenticatedHTTPRequest("future_trade.do", v)

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

func (o *OKCoin) FuturesBatchTrade(orderData, symbol, contractType string, leverage int64, orderType string) {
	v := url.Values{} //to-do batch trade support for orders_data
	v.Set("partner", o.PartnerID)
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	v.Set("orders_data", orderData)
	v.Set("lever_rate", strconv.FormatInt(leverage, 10))

	err := o.SendAuthenticatedHTTPRequest("future_batch_trade.do", v)

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

func (o *OKCoin) CancelFuturesOrder(orderID int64, symbol, contractType string) {
	v := url.Values{}
	v.Set("partner", o.PartnerID)
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	v.Set("order_id", strconv.FormatInt(orderID, 10))

	err := o.SendAuthenticatedHTTPRequest("future_cancel.do", v)

	if err != nil {
		fmt.Println(err)
	}
}

func (o *OKCoin) GetOrderInfo(orderID int64, symbol string) {
	v := url.Values{}
	v.Set("partner", o.PartnerID)
	v.Set("symbol", symbol)
	v.Set("order_id", strconv.FormatInt(orderID, 10))

	err := o.SendAuthenticatedHTTPRequest("orders_info.do", v)

	if err != nil {
		fmt.Println(err)
	}
}

func (o *OKCoin) GetFuturesOrderInfo(orderID, status, currentPage, pageLength int64, symbol, contractType string) {
	v := url.Values{}
	v.Set("partner", o.PartnerID)
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	v.Set("status", strconv.FormatInt(status, 10))
	v.Set("order_id", strconv.FormatInt(orderID, 10))
	v.Set("current_page", strconv.FormatInt(currentPage, 10))
	v.Set("page_length", strconv.FormatInt(pageLength, 10))

	err := o.SendAuthenticatedHTTPRequest("future_order_info.do", v)

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

	err := o.SendAuthenticatedHTTPRequest("order_history.do", v)

	if err != nil {
		fmt.Println(err)
	}
}

func (o *OKCoin) GetFuturesUserInfo4Fix() {
	v := url.Values{}
	v.Set("partner", o.PartnerID)

	err := o.SendAuthenticatedHTTPRequest("future_userinfo_4fix.do", v)

	if err != nil {
		fmt.Println(err)
	}
}

func (o *OKCoin) GetFuturesUserPosition4Fix(symbol, contractType string) {
	v := url.Values{}
	v.Set("partner", o.PartnerID)
	v.Set("symbol", symbol)
	v.Set("contract_type", contractType)
	v.Set("type", strconv.FormatInt(1, 10))

	err := o.SendAuthenticatedHTTPRequest("future_position_4fix.do", v)

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