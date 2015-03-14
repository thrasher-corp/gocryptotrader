package main

import (
	"log"
	"fmt"
	"strconv"
	"encoding/json"
	"crypto/sha512"
	"errors"
	"time"
	"strings"
	"net/url"
	"net/http"
	"io/ioutil"
)

const (
	KRAKEN_API_URL = "https://api.kraken.com"
	KRAKEN_API_VERSION = "0"
	KRAKEN_SERVER_TIME = "Time"
	KRAKEN_ASSETS = "Assets"
	KRAKEN_ASSET_PAIRS = "AssetPairs"
	KRAKEN_TICKER = "Ticker"
	KRAKEN_OHLC = "OHLC"
	KRAKEN_DEPTH = "Depth"
	KRAKEN_TRADES = "Trades"
	KRAKEN_SPREAD = "Spread"
	KRAKEN_BALANCE = "Balance"
	KRAKEN_TRADE_BALANCE = "TradeBalance"
	KRAKEN_OPEN_ORDERS = "OpenOrders"
	KRAKEN_CLOSED_ORDERS = "ClosedOrders"
	KRAKEN_QUERY_ORDERS = "QueryOrders"
	KRAKEN_TRADES_HISTORY = "TradesHistory"
	KRAKEN_QUERY_TRADES = "QueryTrades"
	KRAKEN_OPEN_POSITIONS = "OpenPositions"
	KRAKEN_LEDGERS = "Ledgers"
	KRAKEN_QUERY_LEDGERS = "QueryLedgers"
	KRAKEN_TRADE_VOLUME = "TradeVolume"
	KRAKEN_ORDER_CANCEL = "CancelOrder"
	KRAKEN_ORDER_PLACE = "AddOrder"
)

type Kraken struct {
	Name string
	Enabled bool
	Verbose bool
	ClientKey, APISecret string
	FiatFee, CryptoFee float64
}

type KrakenResponse struct {
	Error []string `json:error`
	Result map[string]interface{} `json:result`
}

func (k *Kraken) SetDefaults() {
	k.Name = "Kraken"
	k.Enabled = true
	k.FiatFee = 0.35
	k.CryptoFee = 0.10
	k.Verbose = false
}

func (k *Kraken) GetName() (string) {
	return k.Name
}

func (k *Kraken) SetEnabled(enabled bool) {
	k.Enabled = enabled
}

func (k *Kraken) IsEnabled() (bool) {
	return k.Enabled
}

func (k *Kraken) SetAPIKeys(apiKey, apiSecret string) {
	k.ClientKey = apiKey
	k.APISecret = apiSecret
}

func (k *Kraken) GetFee(cryptoTrade bool) (float64) {
	if cryptoTrade {
		return k.CryptoFee
	} else {
		return k.FiatFee
	}
}

func (k *Kraken) GetServerTime() {
	result, err := k.SendKrakenRequest(KRAKEN_SERVER_TIME)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) GetAssets() {
	result, err := k.SendKrakenRequest(KRAKEN_ASSETS)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) GetAssetPairs() {
	result, err := k.SendKrakenRequest(KRAKEN_ASSET_PAIRS)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) GetTicker(symbol string) interface{} {
	values := url.Values{}
	values.Set("pair", symbol)

	result, err := k.SendKrakenRequest(KRAKEN_TICKER + "?" + values.Encode())

	if err != nil {
		log.Println(err)
		return ""
	}
	if strings.Contains(symbol, "LTC") {
		return result["XLTCZUSD"]
	} else if strings.Contains(symbol, "XBT") {
		return result["XXBTZUSD"]
	}
	return nil
}

func (k *Kraken) GetOHLC(symbol string) {
	values := url.Values{}
	values.Set("pair", symbol)

	result, err := k.SendKrakenRequest(KRAKEN_OHLC + "?" + values.Encode())

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) GetDepth(symbol string) {
	values := url.Values{}
	values.Set("pair", symbol)

	result, err := k.SendKrakenRequest(KRAKEN_DEPTH + "?" + values.Encode())

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) GetTrades(symbol string) {
	values := url.Values{}
	values.Set("pair", symbol)

	result, err := k.SendKrakenRequest(KRAKEN_TRADES + "?" + values.Encode())

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) GetSpread(symbol string) {
	values := url.Values{}
	values.Set("pair", symbol)

	result, err := k.SendKrakenRequest(KRAKEN_SPREAD + "?" + values.Encode())

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) SendKrakenRequest(method string) (map[string]interface{}, error) {
	path := fmt.Sprintf("%s/%s/public/%s", KRAKEN_API_URL, KRAKEN_API_VERSION, method)
	resp := KrakenResponse{}
	err := SendHTTPRequest(path, true, &resp)

	log.Printf("Sending GET request to %s\n", path)

	if err != nil {
		return nil, err
	}

	if len(resp.Error) != 0 {
		return nil, errors.New(fmt.Sprintf("Kraken error: %s", resp.Error))
	}

	return resp.Result, nil
}

func (k *Kraken) GetBalance() {
	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_BALANCE, url.Values{})

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) GetTradeBalance(symbol, asset string) {
	values := url.Values{}

	if len(symbol) > 0 {
		values.Set("aclass", symbol)
	}

	if len(asset) > 0 {
		values.Set("asset", asset)
	}

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_TRADE_BALANCE, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) GetOpenOrders(showTrades bool, userref int64) {
	values := url.Values{}

	if showTrades {
		values.Set("trades", "true")
	}

	if userref != 0 {
		values.Set("userref", strconv.FormatInt(userref, 10))
	}

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_OPEN_ORDERS, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) GetClosedOrders(showTrades bool, userref, start, end, offset int64, closetime string) {
	values := url.Values{}
	
	if showTrades {
		values.Set("trades", "true")
	}

	if userref != 0 {
		values.Set("userref", strconv.FormatInt(userref, 10))
	}

	if start != 0 {
		values.Set("start", strconv.FormatInt(start, 10))
	}

	if end != 0 {
		values.Set("end", strconv.FormatInt(end, 10))
	}

	if offset != 0 {
		values.Set("ofs", strconv.FormatInt(offset, 10))
	}

	if len(closetime) > 0 {
		values.Set("closetime", closetime)
	}

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_CLOSED_ORDERS, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) QueryOrdersInfo(showTrades bool, userref, txid int64) {
	values := url.Values{}
	
	if showTrades {
		values.Set("trades", "true")
	}

	if userref != 0 {
		values.Set("userref", strconv.FormatInt(userref, 10))
	}

	if txid != 0 {
		values.Set("txid",  strconv.FormatInt(userref, 10))
	}

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_QUERY_ORDERS, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) GetTradesHistory(tradeType string, showRelatedTrades bool, start, end, offset int64) {
	values := url.Values{}
	
	if len(tradeType) > 0 {
		values.Set("aclass", tradeType)
	}

	if showRelatedTrades {
		values.Set("trades", "true")
	}

	if start != 0 {
		values.Set("start", strconv.FormatInt(start, 10))
	}

	if end != 0 {
		values.Set("end", strconv.FormatInt(end, 10))
	}

	if offset != 0 {
		values.Set("offset", strconv.FormatInt(offset, 10))
	}

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_TRADES_HISTORY, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) QueryTrades(txid int64, showRelatedTrades bool) {
	values := url.Values{}
	values.Set("txid", strconv.FormatInt(txid, 10))

	if showRelatedTrades {
		values.Set("trades", "true")
	}

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_QUERY_TRADES, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) OpenPositions(txid int64, showPL bool) {
	values := url.Values{}
	values.Set("txid", strconv.FormatInt(txid, 10))

	if showPL {
		values.Set("docalcs", "true")
	}

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_OPEN_POSITIONS, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) GetLedgers(symbol, asset, ledgerType string, start, end, offset int64) {
	values := url.Values{}
	
	if len(symbol) > 0 {
		values.Set("aclass", symbol)
	}

	if len(asset) > 0 {
		values.Set("asset", asset)
	}

	if len(ledgerType) > 0 {
		values.Set("type", ledgerType)
	}

	if start != 0 {
		values.Set("start", strconv.FormatInt(start, 10))
	}

	if end != 0 {
		values.Set("end", strconv.FormatInt(end, 10))
	}

	if offset != 0 {
		values.Set("offset", strconv.FormatInt(offset, 10))
	}

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_LEDGERS, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) QueryLedgers(id string) {
	values := url.Values{}
	values.Set("id", id)

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_QUERY_LEDGERS, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) GetTradeVolume(symbol string) {
	values := url.Values{}
	values.Set("pair", symbol)

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_TRADE_VOLUME, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) AddOrder(symbol, side, orderType string, price, price2, volume, leverage, position float64) {
	values := url.Values{}
	values.Set("pairs", symbol)
	values.Set("type", side)
	values.Set("ordertype", orderType)
	values.Set("price", strconv.FormatFloat(price, 'f', 2, 64))
	values.Set("price2", strconv.FormatFloat(price, 'f', 2, 64))
	values.Set("volume", strconv.FormatFloat(volume, 'f', 2, 64))
	values.Set("leverage", strconv.FormatFloat(leverage, 'f', 2, 64))
	values.Set("position", strconv.FormatFloat(position, 'f', 2, 64))

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_ORDER_PLACE, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) CancelOrder(orderID int64) {
	values := url.Values{}
	values.Set("txid", strconv.FormatInt(orderID, 10))

	result, err := k.SendAuthenticatedHTTPRequest(KRAKEN_ORDER_CANCEL, values)

	if err != nil {
		log.Println(err)
		return
	}

	log.Println(result)
}

func (k *Kraken) SendAuthenticatedHTTPRequest(method string, values url.Values) (interface{}, error) {
	path := fmt.Sprintf("/%s/private/%s", KRAKEN_API_VERSION, method)
	values.Set("nonce", strconv.FormatInt(time.Now().UnixNano(), 10))
	secret, err := Base64Decode(k.APISecret)

	if err != nil {
		return nil, err
	}

	shasum := GetSHA256([]byte(values.Get("nonce") + values.Encode()))
	signature := Base64Encode(GetHMAC(sha512.New, append([]byte(path), shasum...), secret))

	if k.Verbose {
		log.Printf("Sending POST request to %s, path: %s.", KRAKEN_API_URL, path)
	}

	req, err := http.NewRequest("POST", KRAKEN_API_URL + path, strings.NewReader(values.Encode()))
	req.Header.Set("API-Key", k.ClientKey)
	req.Header.Set("API-Sign", signature)

	client := &http.Client{}
	resp, err := client.Do(req)
	defer resp.Body.Close()

	if err != nil {
		return nil, errors.New("SendAuthenticatedHTTPRequest: Unable to send request")
	}

	contents, _ := ioutil.ReadAll(resp.Body)

	if k.Verbose {
		log.Printf("Recieved raw: \n%s\n", string(contents))
	}
	
	kresp := KrakenResponse{}
	err = json.Unmarshal(contents, &kresp)

	if err != nil {
		return nil, errors.New("Unable to JSON response.")
	}

	if len(kresp.Error) != 0 {
		return nil, errors.New(fmt.Sprintf("Kraken error: %s", kresp.Error))
	}

	return kresp.Result, nil
}