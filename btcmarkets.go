package main

import (
	"strconv"
	"time"
	"log"
	"bytes"
	"fmt"
)

const (
	BTCMARKETS_API_URL = "https://api.btcmarkets.net"
	BTCMARKETS_API_VERSION = "0"
	BTCMARKETS_ACCOUNT_BALANCE = "/account/balance"
	BTCMARKETS_ORDER_CREATE = "/order/create"
	BTCMARKETS_ORDER_CANCEL = "/order/cancel"
	BTCMARKETS_ORDER_HISTORY = "/order/history"
	BTCMARKETS_ORDER_OPEN = "/order/open"
	BTCMARKETS_ORDER_TRADE_HISTORY = "/order/trade/history"
	BTCMARKETS_ORDER_DETAIL = "/order/detail"
)

type BTCMarkets struct {
	Name string
	Enabled bool
	Verbose bool
	Websocket bool
	RESTPollingDelay time.Duration
	Fee float64
	Ticker map[string]BTCMarketsTicker
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

type BTCMarketsTrade struct {
	TradeID int64 `json:"tid"`
	Amount float64 `json:"amount"`
	Price float64 `json:"price"`
	Date int64 `json:"date"`
}

type BTCMarketsOrderbook struct {
	Currency string `json:"currency"`
	Instrument string `json:"instrument"`
	Timestamp int64 `json:"timestamp"`
	Asks [][]float64 `json:"asks"`
	Bids [][]float64 `json:"bids"`
}

type BTCMarketsTradeResponse struct {
	ID int64 `json:"id"`
	CreationTime float64 `json:"creationTime"`
	Description string `json:"description"`
	Price float64 `json:"price"`
	Volume float64 `json:"volume"`
	Fee float64 `json:"fee"`
}

type BTCMarketsOrderResponse struct {
	ID float64 `json:"id"`
	Currency string `json:"currency"`
	Instrument string `json:"instrument"`
	OrderSide string `json:"orderSide"`
	OrderType string `json:"ordertype"`
	CreationTime float64 `json:"creationTime"`
	Status string `json:"status"`
	ErrorMessage string `json:"errorMessage"`
	Price float64 `json:"price"`
	Volume float64 `json:"volume"`
	OpenVolume float64 `json:"openVolume"`
	ClientRequestId string `json:"clientRequestId"`
}

func (b *BTCMarkets) SetDefaults() {
	b.Name = "BTC Markets"
	b.Enabled = true
	b.Fee = 0.85
	b.Verbose = false
	b.Websocket = false
	b.RESTPollingDelay = 10
	b.Ticker = make(map[string]BTCMarketsTicker)
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
	result, err := Base64Decode(apiSecret)

	if err != nil {
		log.Printf("%s unable to decode secret key.\n", b.GetName())
		b.Enabled = false
		return
	}

	b.APISecret = string(result)
}

func (b *BTCMarkets) GetFee() (float64) {
	return b.Fee
}

func (b *BTCMarkets) Run() {
	if b.Verbose {
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
	}

	for b.Enabled {
		go func() {
			BTCMarketsBTC, err := b.GetTicker("BTC")
			if err != nil {
				log.Println(err)
				return
			}
			b.Ticker["BTC"] = BTCMarketsBTC
			BTCMarketsBTCLastUSD, _ := ConvertCurrency(BTCMarketsBTC.LastPrice, "AUD", "USD")
			BTCMarketsBTCBestBidUSD, _ := ConvertCurrency(BTCMarketsBTC.BestBID, "AUD", "USD")
			BTCMarketsBTCBestAskUSD, _ := ConvertCurrency(BTCMarketsBTC.BestAsk, "AUD", "USD")
			log.Printf("BTC Markets BTC: Last %f (%f) Bid %f (%f) Ask %f (%f)\n", BTCMarketsBTCLastUSD, BTCMarketsBTC.LastPrice, BTCMarketsBTCBestBidUSD, BTCMarketsBTC.BestBID, BTCMarketsBTCBestAskUSD, BTCMarketsBTC.BestAsk)
			AddExchangeInfo(b.GetName(), "BTC", BTCMarketsBTCLastUSD, 0)
		}()

		go func() {
			BTCMarketsLTC, err := b.GetTicker("LTC")
			if err != nil {
				log.Println(err)
				return
			}
			b.Ticker["LTC"] = BTCMarketsLTC
			BTCMarketsLTCLastUSD, _ := ConvertCurrency(BTCMarketsLTC.LastPrice, "AUD", "USD")
			BTCMarketsLTCBestBidUSD, _ := ConvertCurrency(BTCMarketsLTC.BestBID, "AUD", "USD")
			BTCMarketsLTCBestAskUSD, _ := ConvertCurrency(BTCMarketsLTC.BestAsk, "AUD", "USD")
			log.Printf("BTC Markets LTC: Last %f (%f) Bid %f (%f) Ask %f (%f)", BTCMarketsLTCLastUSD, BTCMarketsLTC.LastPrice, BTCMarketsLTCBestBidUSD, BTCMarketsLTC.BestBID, BTCMarketsLTCBestAskUSD, BTCMarketsLTC.BestAsk)
			AddExchangeInfo(b.GetName(), "LTC", BTCMarketsLTCLastUSD, 0)
		}()
		time.Sleep(time.Second * b.RESTPollingDelay)
	}
}

func (b *BTCMarkets) GetTicker(symbol string) (BTCMarketsTicker, error) {
	ticker := BTCMarketsTicker{}
	path := fmt.Sprintf("/market/%s/AUD/tick", symbol)
	err := SendHTTPGetRequest(BTCMARKETS_API_URL + path, true, &ticker)
	if err != nil {
		return BTCMarketsTicker{}, err
	}
	return ticker, nil
}

func (b *BTCMarkets) GetOrderbook(symbol string) (BTCMarketsOrderbook, error) {
	orderbook := BTCMarketsOrderbook{}
	path := fmt.Sprintf("/market/%s/AUD/orderbook", symbol)
	err := SendHTTPGetRequest(BTCMARKETS_API_URL + path, true, &orderbook)
	if err != nil {
		return BTCMarketsOrderbook{}, err
	}
	return orderbook, nil
}

func (b *BTCMarkets) GetTrades(symbol, since string) ([]BTCMarketsTrade, error) {
	trades := []BTCMarketsTrade{}
	path := ""
	if len(since) > 0 {
		path = fmt.Sprintf("/market/%s/AUD/trades?since=%s", symbol, since)
	} else {
		path = fmt.Sprintf("/market/%s/AUD/trades", symbol)
	}
	err := SendHTTPGetRequest(BTCMARKETS_API_URL + path, true, &trades)
	if err != nil {
		return nil, err
	}
	return trades, nil
}

func (b *BTCMarkets) Order(currency, instrument string, price, amount int64, orderSide, orderType, clientReq string) (int, error) {
	type Order struct {
		Currency string `json:"currency"`
		Instrument string `json:"instrument"`
		Price int64 `json:"price"`
		Volume int64 `json:"volume"`
		OrderSide string `json:"orderSide"`
		OrderType string `json:"ordertype"`
		ClientRequestId string `json:"clientRequestId"`

	}
	order := Order{}
	order.Currency = currency
	order.Instrument = instrument
	order.Price = price
	order.Volume = amount
	order.OrderSide = orderSide
	order.OrderType = orderType
	order.ClientRequestId = clientReq

	JSONPayload, err := JSONEncode(order)
	if err != nil {
		return 0, err
	}

	type Response struct {
		Success bool `json:"success"`
		ErrorCode int `json:"errorCode"`
		ErrorMessage string `json:"errorMessage"`
		ID int `json:"id"`
		ClientRequestID string `json:"clientRequestId"`
	}
	var resp Response

	err = b.SendAuthenticatedRequest("POST", BTCMARKETS_ORDER_CREATE, JSONPayload, &resp)

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

	JSONPayload, err := JSONEncode(orders)
	if err != nil {
		return false, err
	}

	type Response struct {
		Success bool `json:"success"`
		ErrorCode int `json:"errorCode"`
		ErrorMessage string `json:"errorMessage"`
		Responses []struct {
			Success bool `json:"success"`
			ErrorCode int `json:"errorCode"`
			ErrorMessage string `json:"errorMessage"`
			ID int64 `json:"id"`
		}
		ClientRequestID string `json:"clientRequestId"`
	}
	var resp Response

	err = b.SendAuthenticatedRequest("POST", BTCMARKETS_ORDER_CANCEL, JSONPayload, &resp)

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

func (b *BTCMarkets) GetOrders(currency, instrument string, limit, since int64, historic bool) {
	request := make(map[string]interface{})
	request["currency"] = currency
	request["instrument"] = instrument
	request["limit"] = limit
	request["since"] = since

	JSONPayload, err := JSONEncode(request)
	if err != nil {
		log.Println(err)
		return
	}

	path := BTCMARKETS_ORDER_OPEN
	if historic {
		path = BTCMARKETS_ORDER_HISTORY
	}

	err = b.SendAuthenticatedRequest("POST", path, JSONPayload, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCMarkets) GetOrderDetail(orderID []int64) {
	type OrderDetail struct {
		OrderIDs []int64 `json:"orderIds"`
	}
	orders := OrderDetail{}
	orders.OrderIDs = append(orders.OrderIDs, orderID...)

	JSONPayload, err := JSONEncode(orders)
	if err != nil {
		log.Println(err)
		return
	}

	err = b.SendAuthenticatedRequest("POST", BTCMARKETS_ORDER_DETAIL, JSONPayload, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCMarkets) GetAccountBalance() {
	type Balance struct {
		Balance float64 `json:"balance"`
		PendingFunds float64 `json:"pendingFunds"`
		Currency string `json:"currency"`
	}

	balance := []Balance{}
	err := b.SendAuthenticatedRequest("GET", BTCMARKETS_ACCOUNT_BALANCE, nil, &balance)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCMarkets) SendAuthenticatedRequest(reqType, path string, data []byte, result interface{}) (error) {
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)[0:13]
	request := ""

	if data != nil {
		request = path + "\n" + nonce + "\n" + string(data)
	} else {
		request = path + "\n" + nonce + "\n"
	}

	hmac := GetHMAC(HASH_SHA512, []byte(request), []byte(b.APISecret))

	if b.Verbose {
		log.Printf("Sending %s request to URL %s with params %s\n", reqType, BTCMARKETS_API_URL + path, request)
	}

	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	headers["Accept-Charset"] = "UTF-8"
	headers["Content-Type"] = "application/json"
	headers["apikey"] = b.APIKey
	headers["timestamp"] = nonce
	headers["signature"] = Base64Encode(hmac)

	resp, err := SendHTTPRequest(reqType, BTCMARKETS_API_URL + path, headers, bytes.NewBuffer(data))

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