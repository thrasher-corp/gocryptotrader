package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"
)

const (
	BTCMARKETS_API_URL             = "https://api.btcmarkets.net"
	BTCMARKETS_API_VERSION         = "0"
	BTCMARKETS_ACCOUNT_BALANCE     = "/account/balance"
	BTCMARKETS_ORDER_CREATE        = "/order/create"
	BTCMARKETS_ORDER_CANCEL        = "/order/cancel"
	BTCMARKETS_ORDER_HISTORY       = "/order/history"
	BTCMARKETS_ORDER_OPEN          = "/order/open"
	BTCMARKETS_ORDER_TRADE_HISTORY = "/order/trade/history"
	BTCMARKETS_ORDER_DETAIL        = "/order/detail"
)

type BTCMarkets struct {
	Name                    string
	Enabled                 bool
	Verbose                 bool
	Websocket               bool
	RESTPollingDelay        time.Duration
	Fee                     float64
	Ticker                  map[string]BTCMarketsTicker
	AuthenticatedAPISupport bool
	APIKey, APISecret       string
	BaseCurrencies          []string
	AvailablePairs          []string
	EnabledPairs            []string
}

type BTCMarketsTicker struct {
	BestBID    float64
	BestAsk    float64
	LastPrice  float64
	Currency   string
	Instrument string
	Timestamp  int64
}

type BTCMarketsTrade struct {
	TradeID int64   `json:"tid"`
	Amount  float64 `json:"amount"`
	Price   float64 `json:"price"`
	Date    int64   `json:"date"`
}

type BTCMarketsOrderbook struct {
	Currency   string      `json:"currency"`
	Instrument string      `json:"instrument"`
	Timestamp  int64       `json:"timestamp"`
	Asks       [][]float64 `json:"asks"`
	Bids       [][]float64 `json:"bids"`
}

type BTCMarketsTradeResponse struct {
	ID           int64   `json:"id"`
	CreationTime float64 `json:"creationTime"`
	Description  string  `json:"description"`
	Price        float64 `json:"price"`
	Volume       float64 `json:"volume"`
	Fee          float64 `json:"fee"`
}

type BTCMarketsOrder struct {
	ID              int64                     `json:"id"`
	Currency        string                    `json:"currency"`
	Instrument      string                    `json:"instrument"`
	OrderSide       string                    `json:"orderSide"`
	OrderType       string                    `json:"ordertype"`
	CreationTime    float64                   `json:"creationTime"`
	Status          string                    `json:"status"`
	ErrorMessage    string                    `json:"errorMessage"`
	Price           float64                   `json:"price"`
	Volume          float64                   `json:"volume"`
	OpenVolume      float64                   `json:"openVolume"`
	ClientRequestId string                    `json:"clientRequestId"`
	Trades          []BTCMarketsTradeResponse `json:"trades"`
}

func (b *BTCMarkets) SetDefaults() {
	b.Name = "BTC Markets"
	b.Enabled = false
	b.Fee = 0.85
	b.Verbose = false
	b.Websocket = false
	b.RESTPollingDelay = 10
	b.Ticker = make(map[string]BTCMarketsTicker)
}

func (b *BTCMarkets) GetName() string {
	return b.Name
}

func (b *BTCMarkets) SetEnabled(enabled bool) {
	b.Enabled = enabled
}

func (b *BTCMarkets) IsEnabled() bool {
	return b.Enabled
}

func (b *BTCMarkets) Setup(exch Exchanges) {
	if !exch.Enabled {
		b.SetEnabled(false)
	} else {
		b.Enabled = false
		b.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		b.SetAPIKeys(exch.APIKey, exch.APISecret)
		b.RESTPollingDelay = exch.RESTPollingDelay
		b.Verbose = exch.Verbose
		b.Websocket = exch.Websocket
		b.BaseCurrencies = SplitStrings(exch.BaseCurrencies, ",")
		b.AvailablePairs = SplitStrings(exch.AvailablePairs, ",")
		b.EnabledPairs = SplitStrings(exch.EnabledPairs, ",")

	}
}

func (k *BTCMarkets) GetEnabledCurrencies() []string {
	return k.EnabledPairs
}

func (b *BTCMarkets) Start() {
	go b.Run()
}

func (b *BTCMarkets) SetAPIKeys(apiKey, apiSecret string) {
	if !b.AuthenticatedAPISupport {
		return
	}

	b.APIKey = apiKey
	result, err := Base64Decode(apiSecret)

	if err != nil {
		log.Printf("%s unable to decode secret key.\n", b.GetName())
		b.Enabled = false
		return
	}

	b.APISecret = string(result)
}

func (b *BTCMarkets) GetFee() float64 {
	return b.Fee
}

func (b *BTCMarkets) Run() {
	if b.Verbose {
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	for b.Enabled {
		for _, x := range b.EnabledPairs {
			currency := x
			go func() {
				ticker, err := b.GetTickerPrice(currency)
				if err != nil {
					return
				}
				BTCMarketsLastUSD, _ := ConvertCurrency(ticker.Last, "AUD", "USD")
				BTCMarketsBestBidUSD, _ := ConvertCurrency(ticker.Bid, "AUD", "USD")
				BTCMarketsBestAskUSD, _ := ConvertCurrency(ticker.Ask, "AUD", "USD")
				log.Printf("BTC Markets %s: Last %f (%f) Bid %f (%f) Ask %f (%f)\n", currency, BTCMarketsLastUSD, ticker.Last, BTCMarketsBestBidUSD, ticker.Bid, BTCMarketsBestAskUSD, ticker.Ask)
				AddExchangeInfo(b.GetName(), currency[0:3], currency[3:], ticker.Last, 0)
				AddExchangeInfo(b.GetName(), currency[0:3], "USD", BTCMarketsLastUSD, 0)
			}()
		}
		time.Sleep(time.Second * b.RESTPollingDelay)
	}
}

func (b *BTCMarkets) GetTicker(symbol string) (BTCMarketsTicker, error) {
	ticker := BTCMarketsTicker{}
	path := fmt.Sprintf("/market/%s/AUD/tick", symbol)
	err := SendHTTPGetRequest(BTCMARKETS_API_URL+path, true, &ticker)
	if err != nil {
		return BTCMarketsTicker{}, err
	}
	return ticker, nil
}

func (b *BTCMarkets) GetTickerPrice(currency string) (TickerPrice, error) {
	tickerNew, err := GetTicker(b.GetName(), currency[0:3], currency[3:])
	if err == nil {
		return tickerNew, nil
	}

	var tickerPrice TickerPrice
	ticker, err := b.GetTicker(currency)
	if err != nil {
		return tickerPrice, err
	}
	tickerPrice.Ask = ticker.BestAsk
	tickerPrice.Bid = ticker.BestBID
	tickerPrice.FirstCurrency = currency[0:3]
	tickerPrice.SecondCurrency = currency[3:]
	tickerPrice.Last = ticker.LastPrice
	ProcessTicker(b.GetName(), tickerPrice.FirstCurrency, tickerPrice.SecondCurrency, tickerPrice)
	return tickerPrice, nil
}

func (b *BTCMarkets) GetOrderbook(symbol string) (BTCMarketsOrderbook, error) {
	orderbook := BTCMarketsOrderbook{}
	path := fmt.Sprintf("/market/%s/AUD/orderbook", symbol)
	err := SendHTTPGetRequest(BTCMARKETS_API_URL+path, true, &orderbook)
	if err != nil {
		return BTCMarketsOrderbook{}, err
	}
	return orderbook, nil
}

func (b *BTCMarkets) GetTrades(symbol string, values url.Values) ([]BTCMarketsTrade, error) {
	trades := []BTCMarketsTrade{}
	path := EncodeURLValues(fmt.Sprintf("%s/market/%s/AUD/trades", BTCMARKETS_API_URL, symbol), values)
	err := SendHTTPGetRequest(path, true, &trades)
	if err != nil {
		return nil, err
	}
	return trades, nil
}

func (b *BTCMarkets) Order(currency, instrument string, price, amount int64, orderSide, orderType, clientReq string) (int, error) {
	type Order struct {
		Currency        string `json:"currency"`
		Instrument      string `json:"instrument"`
		Price           int64  `json:"price"`
		Volume          int64  `json:"volume"`
		OrderSide       string `json:"orderSide"`
		OrderType       string `json:"ordertype"`
		ClientRequestId string `json:"clientRequestId"`
	}
	order := Order{}
	order.Currency = currency
	order.Instrument = instrument
	order.Price = price * SATOSHIS_PER_BTC
	order.Volume = amount * SATOSHIS_PER_BTC
	order.OrderSide = orderSide
	order.OrderType = orderType
	order.ClientRequestId = clientReq

	type Response struct {
		Success         bool   `json:"success"`
		ErrorCode       int    `json:"errorCode"`
		ErrorMessage    string `json:"errorMessage"`
		ID              int    `json:"id"`
		ClientRequestID string `json:"clientRequestId"`
	}
	var resp Response

	err := b.SendAuthenticatedRequest("POST", BTCMARKETS_ORDER_CREATE, order, &resp)

	if err != nil {
		return 0, err
	}

	if !resp.Success {
		return 0, fmt.Errorf("%s Unable to place order. Error message: %s\n", b.GetName(), resp.ErrorMessage)
	}
	return resp.ID, nil
}

func (b *BTCMarkets) CancelOrder(orderID []int64) (bool, error) {
	type CancelOrder struct {
		OrderIDs []int64 `json:"orderIds"`
	}
	orders := CancelOrder{}
	orders.OrderIDs = append(orders.OrderIDs, orderID...)

	type Response struct {
		Success      bool   `json:"success"`
		ErrorCode    int    `json:"errorCode"`
		ErrorMessage string `json:"errorMessage"`
		Responses    []struct {
			Success      bool   `json:"success"`
			ErrorCode    int    `json:"errorCode"`
			ErrorMessage string `json:"errorMessage"`
			ID           int64  `json:"id"`
		}
		ClientRequestID string `json:"clientRequestId"`
	}
	var resp Response

	err := b.SendAuthenticatedRequest("POST", BTCMARKETS_ORDER_CANCEL, orders, &resp)

	if err != nil {
		return false, err
	}

	if !resp.Success {
		return false, fmt.Errorf("%s Unable to cancel order. Error message: %s\n", b.GetName(), resp.ErrorMessage)
	}

	ordersToBeCancelled := len(orderID)
	ordersCancelled := 0
	for _, y := range resp.Responses {
		if y.Success {
			ordersCancelled++
			log.Printf("%s Cancelled order %d.\n", b.GetName(), y.ID)
		} else {
			log.Printf("%s Unable to cancel order %d. Error message: %s\n", b.GetName(), y.ID, y.ErrorMessage)
		}
	}

	if ordersCancelled == ordersToBeCancelled {
		return true, nil
	} else {
		return false, fmt.Errorf("%s Unable to cancel order(s).", b.GetName())
	}
}

func (b *BTCMarkets) GetOrders(currency, instrument string, limit, since int64, historic bool) ([]BTCMarketsOrder, error) {
	request := make(map[string]interface{})
	request["currency"] = currency
	request["instrument"] = instrument
	request["limit"] = limit
	request["since"] = since

	path := BTCMARKETS_ORDER_OPEN
	if historic {
		path = BTCMARKETS_ORDER_HISTORY
	}

	type response struct {
		Success      bool              `json:"success"`
		ErrorCode    int               `json:"errorCode"`
		ErrorMessage string            `json:"errorMessage"`
		Orders       []BTCMarketsOrder `json:"orders"`
	}

	resp := response{}
	err := b.SendAuthenticatedRequest("POST", path, request, &resp)

	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, errors.New(resp.ErrorMessage)
	}

	for i := range resp.Orders {
		resp.Orders[i].Price = resp.Orders[i].Price / SATOSHIS_PER_BTC
		resp.Orders[i].OpenVolume = resp.Orders[i].OpenVolume / SATOSHIS_PER_BTC
		resp.Orders[i].Volume = resp.Orders[i].Volume / SATOSHIS_PER_BTC

		for x := range resp.Orders[i].Trades {
			resp.Orders[i].Trades[x].Fee = resp.Orders[i].Trades[x].Fee / SATOSHIS_PER_BTC
			resp.Orders[i].Trades[x].Price = resp.Orders[i].Trades[x].Price / SATOSHIS_PER_BTC
			resp.Orders[i].Trades[x].Volume = resp.Orders[i].Trades[x].Volume / SATOSHIS_PER_BTC
		}
	}
	return resp.Orders, nil
}

func (b *BTCMarkets) GetOrderDetail(orderID []int64) ([]BTCMarketsOrder, error) {
	type OrderDetail struct {
		OrderIDs []int64 `json:"orderIds"`
	}
	orders := OrderDetail{}
	orders.OrderIDs = append(orders.OrderIDs, orderID...)

	type response struct {
		Success      bool              `json:"success"`
		ErrorCode    int               `json:"errorCode"`
		ErrorMessage string            `json:"errorMessage"`
		Orders       []BTCMarketsOrder `json:"orders"`
	}

	resp := response{}
	err := b.SendAuthenticatedRequest("POST", BTCMARKETS_ORDER_DETAIL, orders, &resp)

	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, errors.New(resp.ErrorMessage)
	}

	for i := range resp.Orders {
		resp.Orders[i].Price = resp.Orders[i].Price / SATOSHIS_PER_BTC
		resp.Orders[i].OpenVolume = resp.Orders[i].OpenVolume / SATOSHIS_PER_BTC
		resp.Orders[i].Volume = resp.Orders[i].Volume / SATOSHIS_PER_BTC

		for x := range resp.Orders[i].Trades {
			resp.Orders[i].Trades[x].Fee = resp.Orders[i].Trades[x].Fee / SATOSHIS_PER_BTC
			resp.Orders[i].Trades[x].Price = resp.Orders[i].Trades[x].Price / SATOSHIS_PER_BTC
			resp.Orders[i].Trades[x].Volume = resp.Orders[i].Trades[x].Volume / SATOSHIS_PER_BTC
		}
	}
	return resp.Orders, nil
}

type BTCMarketsAccountBalance struct {
	Balance      float64 `json:"balance"`
	PendingFunds float64 `json:"pendingFunds"`
	Currency     string  `json:"currency"`
}

func (b *BTCMarkets) GetAccountBalance() ([]BTCMarketsAccountBalance, error) {
	balance := []BTCMarketsAccountBalance{}
	err := b.SendAuthenticatedRequest("GET", BTCMARKETS_ACCOUNT_BALANCE, nil, &balance)

	if err != nil {
		return nil, err
	}

	for i := range balance {
		if balance[i].Currency == "LTC" || balance[i].Currency == "BTC" {
			balance[i].Balance = balance[i].Balance / SATOSHIS_PER_BTC
			balance[i].PendingFunds = balance[i].PendingFunds / SATOSHIS_PER_BTC
		}
	}
	return balance, nil
}

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the BTCMarkets exchange
func (e *BTCMarkets) GetExchangeAccountInfo() (ExchangeAccountInfo, error) {
	var response ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	accountBalance, err := e.GetAccountBalance()
	if err != nil {
		return response, err
	}
	for i := 0; i < len(accountBalance); i++ {
		var exchangeCurrency ExchangeAccountCurrencyInfo
		exchangeCurrency.CurrencyName = accountBalance[i].Currency
		exchangeCurrency.TotalValue = accountBalance[i].Balance
		exchangeCurrency.Hold = accountBalance[i].PendingFunds

		response.Currencies = append(response.Currencies, exchangeCurrency)
	}
	return response, nil
}

func (b *BTCMarkets) SendAuthenticatedRequest(reqType, path string, data interface{}, result interface{}) (err error) {
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)[0:13]
	request := ""
	payload := []byte("")

	if data != nil {
		payload, err = JSONEncode(data)
		if err != nil {
			return err
		}
		request = path + "\n" + nonce + "\n" + string(payload)
	} else {
		request = path + "\n" + nonce + "\n"
	}

	hmac := GetHMAC(HASH_SHA512, []byte(request), []byte(b.APISecret))

	if b.Verbose {
		log.Printf("Sending %s request to URL %s with params %s\n", reqType, BTCMARKETS_API_URL+path, request)
	}

	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	headers["Accept-Charset"] = "UTF-8"
	headers["Content-Type"] = "application/json"
	headers["apikey"] = b.APIKey
	headers["timestamp"] = nonce
	headers["signature"] = Base64Encode(hmac)

	resp, err := SendHTTPRequest(reqType, BTCMARKETS_API_URL+path, headers, bytes.NewBuffer(payload))

	if err != nil {
		return err
	}

	if b.Verbose {
		log.Printf("Recieved raw: %s\n", resp)
	}

	err = JSONDecode([]byte(resp), &result)

	if err != nil {
		return err
	}

	return nil
}
