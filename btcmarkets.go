package main

import (
	"strconv"
	"strings"
	"time"
	"log"
	"fmt"
)

const (
	BTCMARKETS_API_URL = "https://api.btcmarkets.net"
	BTCMARKETS_API_VERSION = "0"
)

type BTCMarkets struct {
	Name string
	Enabled bool
	Verbose bool
	PollingDelay time.Duration
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
	b.PollingDelay = 10
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

func (b *BTCMarkets) Run() {
	if b.Verbose {
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.PollingDelay)
	}
	
	for b.Enabled {
		go func() {
			BTCMarketsBTC := b.GetTicker("BTC")
			BTCMarketsBTCLastUSD, _ := ConvertCurrency(BTCMarketsBTC.LastPrice, "AUD", "USD")
			BTCMarketsBTCBestBidUSD, _ := ConvertCurrency(BTCMarketsBTC.BestBID, "AUD", "USD")
			BTCMarketsBTCBestAskUSD, _ := ConvertCurrency(BTCMarketsBTC.BestAsk, "AUD", "USD")
			log.Printf("BTC Markets BTC: Last %f (%f) Bid %f (%f) Ask %f (%f)\n", BTCMarketsBTCLastUSD, BTCMarketsBTC.LastPrice, BTCMarketsBTCBestBidUSD, BTCMarketsBTC.BestBID, BTCMarketsBTCBestAskUSD, BTCMarketsBTC.BestAsk)
		}()

		go func() {
			BTCMarketsLTC := b.GetTicker("LTC")
			BTCMarketsLTCLastUSD, _ := ConvertCurrency(BTCMarketsLTC.LastPrice, "AUD", "USD")
			BTCMarketsLTCBestBidUSD, _ := ConvertCurrency(BTCMarketsLTC.BestBID, "AUD", "USD")
			BTCMarketsLTCBestAskUSD, _ := ConvertCurrency(BTCMarketsLTC.BestAsk, "AUD", "USD")
			log.Printf("BTC Markets LTC: Last %f (%f) Bid %f (%f) Ask %f (%f)", BTCMarketsLTCLastUSD, BTCMarketsLTC.LastPrice, BTCMarketsLTCBestBidUSD, BTCMarketsLTC.BestBID, BTCMarketsLTCBestAskUSD, BTCMarketsLTC.BestAsk)
		}()
		time.Sleep(time.Second * b.PollingDelay)
	}
}

func (b *BTCMarkets) GetTicker(symbol string) (BTCMarketsTicker) {
	ticker := BTCMarketsTicker{}
	path := fmt.Sprintf("/market/%s/AUD/tick", symbol)
	err := SendHTTPGetRequest(BTCMARKETS_API_URL + path, true, &ticker)
	if err != nil {
		log.Println(err)
		return BTCMarketsTicker{}
	}
	return ticker
}

func (b *BTCMarkets) GetOrderbook(symbol string) {
	path := fmt.Sprintf("/market/%s/AUD/orderbook", symbol)
	err := SendHTTPGetRequest(BTCMARKETS_API_URL + path, true, nil)
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
	err := SendHTTPGetRequest(BTCMARKETS_API_URL + path, true, nil)
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

	hmac := GetHMAC(HASH_SHA512, []byte(request), []byte(b.APISecret))

	if b.Verbose {
		log.Printf("Sending %s request to %s path %s with params %s\n", reqType, BTCMARKETS_API_URL + path, path, request)
	}

	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	headers["Content-Type"] = "application/json"
	headers["Accept-Charset"] = "UTF-8"
	headers["apikey"] = b.APIKey
	headers["timestamp"] = nonce
	headers["signature"] = Base64Encode(hmac)

	resp, err := SendHTTPRequest(reqType, BTCMARKETS_API_URL + path, headers, strings.NewReader(""))

	if err != nil {
		return err
	}

	if b.Verbose {
		log.Printf("Recieved raw: %s\n", resp)
	}

	return nil
}