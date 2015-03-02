package main

import (
	"net/http"
	"strconv"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"strings"
	"time"
	"log"
	"io/ioutil"
	"fmt"
)

const (
	BTCMARKETS_API_URL = "https://api.btcmarkets.net"
)

type BTCMarkets struct {
	Name string
	Enabled bool
	Verbose bool
	Fee float64
	APIKey, APISecret string
}

type BTCMarketsTicker struct {
	BestBID float64
	BestAsk float64
	LastPrice float64
	Currency string
	Instrument string
	Timestamp int64
}

func (b *BTCMarkets) SetDefaults() {
	b.Name = "BTC Markets"
	b.Enabled = true
	b.Fee = 0.85
	b.Verbose = false
}

func (b *BTCMarkets) GetName() (string) {
	return b.Name
}

func (b *BTCMarkets) SetEnabled(enabled bool) {
	b.Enabled = enabled
}

func (b *BTCMarkets) IsEnabled() (bool) {
	return b.Enabled
}

func (b *BTCMarkets) SetAPIKeys(apiKey, apiSecret string) {
	b.APIKey = apiKey
	b.APISecret = apiSecret
}

func (b *BTCMarkets) GetFee() (float64) {
	return b.Fee
}

func (b *BTCMarkets) GetTicker(symbol string) (BTCMarketsTicker) {
	ticker := BTCMarketsTicker{}
	path := fmt.Sprintf("/market/%s/AUD/tick", symbol)
	err := SendHTTPRequest(BTCMARKETS_API_URL + path, true, &ticker)
	if err != nil {
		log.Println(err)
		return BTCMarketsTicker{}
	}
	return ticker
}

func (b *BTCMarkets) GetOrderbook(symbol string) {
	path := fmt.Sprintf("/market/%s/AUD/orderbook", symbol)
	err := SendHTTPRequest(BTCMARKETS_API_URL + path, true, nil)
	if err != nil {
		log.Println(err)
	}
}

func (b *BTCMarkets) GetTrades(symbol, since string) {
	path := ""
	if len(since) > 0 {
		path = fmt.Sprintf("/market/%s/AUD/trades?since=%s", symbol, since)
	} else {
		path = fmt.Sprintf("/market/%s/AUD/trades", symbol)
	}
	err := SendHTTPRequest(BTCMARKETS_API_URL + path, true, nil)
	if err != nil {
		log.Println(err)
	}
}

func (b *BTCMarkets) SendAuthenticatedRequest(reqType, path, data string) (error) {
	nonce := strconv.FormatInt(time.Now().Unix(), 10)
	request := ""

	if len(data) > 0 {
		request = path + "\n" + nonce + "\n" + data 
	} else {
		request = path + "\n" + nonce + "\n"
	}
	
	hmac := hmac.New(sha512.New, []byte(b.APISecret))
	hmac.Write([]byte(request))

	if b.Verbose {
		log.Printf("Sending %s request to %s path %s with params %s\n", reqType, BTCMARKETS_API_URL + path, path, request)
	}
	
	req, err := http.NewRequest(reqType, BTCMARKETS_API_URL + path, strings.NewReader(""))

	if err != nil {
		return err
	}

	b64 := base64.StdEncoding.EncodeToString(hmac.Sum(nil))

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", "btc markets python client")
	req.Header.Add("Accept-Charset", "UTF-8")
	req.Header.Add("apikey", b.APIKey)
	req.Header.Add("timestamp", nonce)
	req.Header.Add("signature", b64)

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