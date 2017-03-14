package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"
)

const (
	ANX_API_URL         = "https://anxpro.com/"
	ANX_API_VERSION     = "3"
	ANX_APIKEY          = "apiKey"
	ANX_DATA_TOKEN      = "dataToken"
	ANX_ORDER_NEW       = "order/new"
	ANX_ORDER_INFO      = "order/info"
	ANX_SEND            = "send"
	ANX_SUBACCOUNT_NEW  = "subaccount/new"
	ANX_RECEIVE_ADDRESS = "receive"
	ANX_CREATE_ADDRESS  = "receive/create"
	ANX_TICKER          = "money/ticker"
)

type ANX struct {
	Name                    string
	Enabled                 bool
	Verbose                 bool
	Websocket               bool
	RESTPollingDelay        time.Duration
	AuthenticatedAPISupport bool
	APIKey, APISecret       string
	TakerFee, MakerFee      float64
	BaseCurrencies          []string
	AvailablePairs          []string
	EnabledPairs            []string
}

type ANXOrder struct {
	OrderType                      string `json:"orderType"`
	BuyTradedCurrency              bool   `json:"buyTradedCurrency"`
	TradedCurrency                 string `json:"tradedCurrency"`
	SettlementCurrency             string `json:"settlementCurrency"`
	TradedCurrencyAmount           string `json:"tradedCurrencyAmount"`
	SettlementCurrencyAmount       string `json:"settlementCurrencyAmount"`
	LimitPriceInSettlementCurrency string `json:"limitPriceInSettlementCurrency"`
	ReplaceExistingOrderUUID       string `json:"replaceExistingOrderUuid"`
	ReplaceOnlyIfActive            bool   `json:"replaceOnlyIfActive"`
}

type ANXOrderResponse struct {
	BuyTradedCurrency              bool   `json:"buyTradedCurrency"`
	ExecutedAverageRate            string `json:"executedAverageRate"`
	LimitPriceInSettlementCurrency string `json:"limitPriceInSettlementCurrency"`
	OrderID                        string `json:"orderId"`
	OrderStatus                    string `json:"orderStatus"`
	OrderType                      string `json:"orderType"`
	ReplaceExistingOrderUUID       string `json:"replaceExistingOrderId"`
	SettlementCurrency             string `json:"settlementCurrency"`
	SettlementCurrencyAmount       string `json:"settlementCurrencyAmount"`
	SettlementCurrencyOutstanding  string `json:"settlementCurrencyOutstanding"`
	Timestamp                      int64  `json:"timestamp"`
	TradedCurrency                 string `json:"tradedCurrency"`
	TradedCurrencyAmount           string `json:"tradedCurrencyAmount"`
	TradedCurrencyOutstanding      string `json:"tradedCurrencyOutstanding"`
}

type ANXTickerComponent struct {
	Currency     string  `json:"currency"`
	Display      string  `json:"display"`
	DisplayShort string  `json:"display_short"`
	Value        float64 `json:"value,string"`
	ValueInt     int64   `json:"value_int,string"`
}

type ANXTicker struct {
	Result string `json:"result"`
	Data   struct {
		High       ANXTickerComponent `json:"high"`
		Low        ANXTickerComponent `json:"low"`
		Avg        ANXTickerComponent `json:"avg"`
		Vwap       ANXTickerComponent `json:"vwap"`
		Vol        ANXTickerComponent `json:"vol"`
		Last       ANXTickerComponent `json:"last"`
		Buy        ANXTickerComponent `json:"buy"`
		Sell       ANXTickerComponent `json:"sell"`
		Now        string             `json:"now"`
		UpdateTime string             `json:"dataUpdateTime"`
	} `json:"data"`
}

func (a *ANX) SetDefaults() {
	a.Name = "ANX"
	a.Enabled = false
	a.TakerFee = 0.6
	a.MakerFee = 0.3
	a.Verbose = false
	a.Websocket = false
	a.RESTPollingDelay = 10
}

//Setup is run on startup to setup exchange with config values
func (a *ANX) Setup(exch Exchanges) {
	if !exch.Enabled {
		a.SetEnabled(false)
	} else {
		a.Enabled = true
		a.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		a.SetAPIKeys(exch.APIKey, exch.APISecret)
		a.RESTPollingDelay = exch.RESTPollingDelay
		a.Verbose = exch.Verbose
		a.Websocket = exch.Websocket
		a.BaseCurrencies = SplitStrings(exch.BaseCurrencies, ",")
		a.AvailablePairs = SplitStrings(exch.AvailablePairs, ",")
		a.EnabledPairs = SplitStrings(exch.EnabledPairs, ",")
	}
}

//Start is run if exchange is enabled, after Setup
func (a *ANX) Start() {
	go a.Run()
}

func (a *ANX) GetName() string {
	return a.Name
}

func (a *ANX) SetEnabled(enabled bool) {
	a.Enabled = enabled
}

func (a *ANX) IsEnabled() bool {
	return a.Enabled
}

func (a *ANX) SetAPIKeys(apiKey, apiSecret string) {
	if !a.AuthenticatedAPISupport {
		return
	}

	a.APIKey = apiKey
	result, err := Base64Decode(apiSecret)

	if err != nil {
		log.Printf("%s unable to decode secret key. Authenticated API support disabled.", a.GetName())
		a.AuthenticatedAPISupport = false
		return
	}

	a.APISecret = string(result)
}

func (a *ANX) GetFee(maker bool) float64 {
	if maker {
		return a.MakerFee
	} else {
		return a.TakerFee
	}
}

func (a *ANX) Run() {
	if a.Verbose {
		log.Printf("%s polling delay: %ds.\n", a.GetName(), a.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", a.GetName(), len(a.EnabledPairs), a.EnabledPairs)
	}

	for a.Enabled {
		for _, x := range a.EnabledPairs {
			currency := x
			go func() {
				ticker, err := a.GetTickerPrice(currency)
				if err != nil {
					log.Println(err)
					return
				}
				log.Printf("ANX %s: Last %f High %f Low %f Volume %f\n", currency, ticker.Last, ticker.High, ticker.Low, ticker.Volume)
				AddExchangeInfo(a.GetName(), currency[0:3], currency[3:], ticker.Last, ticker.Volume)
			}()
		}
		time.Sleep(time.Second * a.RESTPollingDelay)
	}
}

func (a *ANX) GetTicker(currency string) (ANXTicker, error) {
	var ticker ANXTicker
	err := SendHTTPGetRequest(fmt.Sprintf("%sapi/2/%s/%s", ANX_API_URL, currency, ANX_TICKER), true, &ticker)
	if err != nil {
		return ANXTicker{}, err
	}
	return ticker, nil
}

func (a *ANX) GetTickerPrice(currency string) (TickerPrice, error) {
	tickerNew, err := GetTicker(a.GetName(), currency[0:3], currency[3:])
	if err == nil {
		return tickerNew, nil
	}

	var tickerPrice TickerPrice
	ticker, err := a.GetTicker(currency)
	if err != nil {
		return tickerPrice, err
	}

	tickerPrice.Ask = ticker.Data.Buy.Value
	tickerPrice.Bid = ticker.Data.Sell.Value
	tickerPrice.FirstCurrency = currency[0:3]
	tickerPrice.SecondCurrency = currency[3:]
	tickerPrice.Low = ticker.Data.Low.Value
	tickerPrice.Last = ticker.Data.Last.Value
	tickerPrice.Volume = ticker.Data.Vol.Value
	tickerPrice.High = ticker.Data.High.Value
	ProcessTicker(a.GetName(), tickerPrice.FirstCurrency, tickerPrice.SecondCurrency, tickerPrice)
	return tickerPrice, nil
}

func (a *ANX) GetEnabledCurrencies() []string {
	return a.EnabledPairs
}

func (a *ANX) GetAPIKey(username, password, otp, deviceID string) (string, string) {
	request := make(map[string]interface{})
	request["nonce"] = strconv.FormatInt(time.Now().UnixNano(), 10)[0:13]
	request["username"] = username
	request["password"] = password

	if otp != "" {
		request["otp"] = otp
	}

	request["deviceId"] = deviceID

	type APIKeyResponse struct {
		APIKey     string `json:"apiKey"`
		APISecret  string `json:"apiSecret"`
		ResultCode string `json:"resultCode"`
		Timestamp  int64  `json:"timestamp"`
	}
	var response APIKeyResponse

	err := a.SendAuthenticatedHTTPRequest(ANX_APIKEY, request, &response)

	if err != nil {
		log.Println(err)
		return "", ""
	}

	if response.ResultCode != "OK" {
		log.Printf("Response code is not OK: %s\n", response.ResultCode)
		return "", ""
	}

	return response.APIKey, response.APISecret
}

func (a *ANX) GetDataToken() string {
	request := make(map[string]interface{})

	type DataTokenResponse struct {
		ResultCode string `json:"resultCode"`
		Timestamp  int64  `json:"timestamp"`
		Token      string `json:"token"`
		UUID       string `json:"uuid"`
	}
	var response DataTokenResponse

	err := a.SendAuthenticatedHTTPRequest(ANX_DATA_TOKEN, request, &response)

	if err != nil {
		log.Println(err)
		return ""
	}

	if response.ResultCode != "OK" {
		log.Printf("Response code is not OK: %s\n", response.ResultCode)
		return ""
	}

	return response.Token
}

func (a *ANX) NewOrder(orderType string, buy bool, tradedCurrency, tradedCurrencyAmount, settlementCurrency, settlementCurrencyAmount, limitPriceSettlement string,
	replace bool, replaceUUID string, replaceIfActive bool) {
	request := make(map[string]interface{})

	var order ANXOrder
	order.OrderType = orderType
	order.BuyTradedCurrency = buy

	if buy {
		order.TradedCurrencyAmount = tradedCurrencyAmount
	} else {
		order.SettlementCurrencyAmount = settlementCurrencyAmount
	}

	order.TradedCurrency = tradedCurrency
	order.SettlementCurrency = settlementCurrency
	order.LimitPriceInSettlementCurrency = limitPriceSettlement

	if replace {
		order.ReplaceExistingOrderUUID = replaceUUID
		order.ReplaceOnlyIfActive = replaceIfActive
	}

	request["order"] = order

	type OrderResponse struct {
		OrderID    string `json:"orderId"`
		Timestamp  int64  `json:"timestamp"`
		ResultCode string `json:"resultCode"`
	}
	var response OrderResponse

	err := a.SendAuthenticatedHTTPRequest(ANX_ORDER_NEW, request, &response)

	if err != nil {
		log.Println(err)
		return
	}

	if response.ResultCode != "OK" {
		log.Printf("Response code is not OK: %s\n", response.ResultCode)
		return
	}
}

func (a *ANX) OrderInfo(orderID string) (ANXOrderResponse, error) {
	request := make(map[string]interface{})
	request["orderId"] = orderID

	type OrderInfoResponse struct {
		Order      ANXOrderResponse `json:"order"`
		ResultCode string           `json:"resultCode"`
		Timestamp  int64            `json:"timestamp"`
	}
	var response OrderInfoResponse

	err := a.SendAuthenticatedHTTPRequest(ANX_ORDER_INFO, request, &response)

	if err != nil {
		return ANXOrderResponse{}, err
	}

	if response.ResultCode != "OK" {
		log.Printf("Response code is not OK: %s\n", response.ResultCode)
		return ANXOrderResponse{}, errors.New(response.ResultCode)
	}
	return response.Order, nil
}

func (a *ANX) Send(currency, address, otp, amount string) (string, error) {
	request := make(map[string]interface{})
	request["ccy"] = currency
	request["amount"] = amount
	request["address"] = address

	if otp != "" {
		request["otp"] = otp
	}

	type SendResponse struct {
		TransactionID string `json:"transactionId"`
		ResultCode    string `json:"resultCode"`
		Timestamp     int64  `json:"timestamp"`
	}
	var response SendResponse

	err := a.SendAuthenticatedHTTPRequest(ANX_SEND, request, &response)

	if err != nil {
		return "", err
	}

	if response.ResultCode != "OK" {
		log.Printf("Response code is not OK: %s\n", response.ResultCode)
		return "", errors.New(response.ResultCode)
	}
	return response.TransactionID, nil
}

func (a *ANX) CreateNewSubAccount(currency, name string) (string, error) {
	request := make(map[string]interface{})
	request["ccy"] = currency
	request["customRef"] = name

	type SubaccountResponse struct {
		SubAccount string `json:"subAccount"`
		ResultCode string `json:"resultCode"`
		Timestamp  int64  `json:"timestamp"`
	}
	var response SubaccountResponse

	err := a.SendAuthenticatedHTTPRequest(ANX_SUBACCOUNT_NEW, request, &response)

	if err != nil {
		return "", err
	}

	if response.ResultCode != "OK" {
		log.Printf("Response code is not OK: %s\n", response.ResultCode)
		return "", errors.New(response.ResultCode)
	}
	return response.SubAccount, nil
}

func (a *ANX) GetDepositAddress(currency, name string, new bool) (string, error) {
	request := make(map[string]interface{})
	request["ccy"] = currency

	if name != "" {
		request["subAccount"] = name
	}

	type AddressResponse struct {
		Address    string `json:"address"`
		SubAccount string `json:"subAccount"`
		ResultCode string `json:"resultCode"`
		Timestamp  int64  `json:"timestamp"`
	}
	var response AddressResponse

	path := ANX_RECEIVE_ADDRESS
	if new {
		path = ANX_CREATE_ADDRESS
	}

	err := a.SendAuthenticatedHTTPRequest(path, request, &response)

	if err != nil {
		return "", err
	}

	if response.ResultCode != "OK" {
		log.Printf("Response code is not OK: %s\n", response.ResultCode)
		return "", errors.New(response.ResultCode)
	}

	return response.Address, nil
}

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the ANX exchange
func (e *ANX) GetExchangeAccountInfo() (ExchangeAccountInfo, error) {
	var response ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	return response, nil
}

func (a *ANX) SendAuthenticatedHTTPRequest(path string, params map[string]interface{}, result interface{}) (err error) {
	request := make(map[string]interface{})
	request["nonce"] = strconv.FormatInt(time.Now().UnixNano(), 10)[0:13]
	path = fmt.Sprintf("api/%s/%s", ANX_API_VERSION, path)

	if params != nil {
		for key, value := range params {
			request[key] = value
		}
	}

	PayloadJson, err := JSONEncode(request)

	if err != nil {
		return errors.New("SendAuthenticatedHTTPRequest: Unable to JSON request")
	}

	if a.Verbose {
		log.Printf("Request JSON: %s\n", PayloadJson)
	}

	hmac := GetHMAC(HASH_SHA512, []byte(path+string("\x00")+string(PayloadJson)), []byte(a.APISecret))
	headers := make(map[string]string)
	headers["Rest-Key"] = a.APIKey
	headers["Rest-Sign"] = Base64Encode([]byte(hmac))
	headers["Content-Type"] = "application/json"

	resp, err := SendHTTPRequest("POST", ANX_API_URL+path, headers, bytes.NewBuffer(PayloadJson))

	if a.Verbose {
		log.Printf("Recieved raw: \n%s\n", resp)
	}

	err = JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("Unable to JSON Unmarshal response.")
	}

	return nil
}
