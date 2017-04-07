package btcmarkets

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
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
	exchange.ExchangeBase
	Ticker map[string]BTCMarketsTicker
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

func (b *BTCMarkets) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		b.SetEnabled(false)
	} else {
		b.Enabled = true
		b.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		b.SetAPIKeys(exch.APIKey, exch.APISecret, "", true)
		b.RESTPollingDelay = exch.RESTPollingDelay
		b.Verbose = exch.Verbose
		b.Websocket = exch.Websocket
		b.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		b.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		b.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")

	}
}

func (b *BTCMarkets) GetFee() float64 {
	return b.Fee
}

func (b *BTCMarkets) GetTicker(symbol string) (BTCMarketsTicker, error) {
	ticker := BTCMarketsTicker{}
	path := fmt.Sprintf("/market/%s/AUD/tick", symbol)
	err := common.SendHTTPGetRequest(BTCMARKETS_API_URL+path, true, &ticker)
	if err != nil {
		return BTCMarketsTicker{}, err
	}
	return ticker, nil
}

func (b *BTCMarkets) GetOrderbook(symbol string) (BTCMarketsOrderbook, error) {
	orderbook := BTCMarketsOrderbook{}
	path := fmt.Sprintf("/market/%s/AUD/orderbook", symbol)
	err := common.SendHTTPGetRequest(BTCMARKETS_API_URL+path, true, &orderbook)
	if err != nil {
		return BTCMarketsOrderbook{}, err
	}
	return orderbook, nil
}

func (b *BTCMarkets) GetTrades(symbol string, values url.Values) ([]BTCMarketsTrade, error) {
	trades := []BTCMarketsTrade{}
	path := common.EncodeURLValues(fmt.Sprintf("%s/market/%s/AUD/trades", BTCMARKETS_API_URL, symbol), values)
	err := common.SendHTTPGetRequest(path, true, &trades)
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
	order.Price = price * common.SATOSHIS_PER_BTC
	order.Volume = amount * common.SATOSHIS_PER_BTC
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
		resp.Orders[i].Price = resp.Orders[i].Price / common.SATOSHIS_PER_BTC
		resp.Orders[i].OpenVolume = resp.Orders[i].OpenVolume / common.SATOSHIS_PER_BTC
		resp.Orders[i].Volume = resp.Orders[i].Volume / common.SATOSHIS_PER_BTC

		for x := range resp.Orders[i].Trades {
			resp.Orders[i].Trades[x].Fee = resp.Orders[i].Trades[x].Fee / common.SATOSHIS_PER_BTC
			resp.Orders[i].Trades[x].Price = resp.Orders[i].Trades[x].Price / common.SATOSHIS_PER_BTC
			resp.Orders[i].Trades[x].Volume = resp.Orders[i].Trades[x].Volume / common.SATOSHIS_PER_BTC
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
		resp.Orders[i].Price = resp.Orders[i].Price / common.SATOSHIS_PER_BTC
		resp.Orders[i].OpenVolume = resp.Orders[i].OpenVolume / common.SATOSHIS_PER_BTC
		resp.Orders[i].Volume = resp.Orders[i].Volume / common.SATOSHIS_PER_BTC

		for x := range resp.Orders[i].Trades {
			resp.Orders[i].Trades[x].Fee = resp.Orders[i].Trades[x].Fee / common.SATOSHIS_PER_BTC
			resp.Orders[i].Trades[x].Price = resp.Orders[i].Trades[x].Price / common.SATOSHIS_PER_BTC
			resp.Orders[i].Trades[x].Volume = resp.Orders[i].Trades[x].Volume / common.SATOSHIS_PER_BTC
		}
	}
	return resp.Orders, nil
}

func (b *BTCMarkets) GetAccountBalance() ([]BTCMarketsAccountBalance, error) {
	balance := []BTCMarketsAccountBalance{}
	err := b.SendAuthenticatedRequest("GET", BTCMARKETS_ACCOUNT_BALANCE, nil, &balance)

	if err != nil {
		return nil, err
	}

	for i := range balance {
		if balance[i].Currency == "LTC" || balance[i].Currency == "BTC" {
			balance[i].Balance = balance[i].Balance / common.SATOSHIS_PER_BTC
			balance[i].PendingFunds = balance[i].PendingFunds / common.SATOSHIS_PER_BTC
		}
	}
	return balance, nil
}

func (b *BTCMarkets) SendAuthenticatedRequest(reqType, path string, data interface{}, result interface{}) (err error) {
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)[0:13]
	request := ""
	payload := []byte("")

	if data != nil {
		payload, err = common.JSONEncode(data)
		if err != nil {
			return err
		}
		request = path + "\n" + nonce + "\n" + string(payload)
	} else {
		request = path + "\n" + nonce + "\n"
	}

	hmac := common.GetHMAC(common.HASH_SHA512, []byte(request), []byte(b.APISecret))

	if b.Verbose {
		log.Printf("Sending %s request to URL %s with params %s\n", reqType, BTCMARKETS_API_URL+path, request)
	}

	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	headers["Accept-Charset"] = "UTF-8"
	headers["Content-Type"] = "application/json"
	headers["apikey"] = b.APIKey
	headers["timestamp"] = nonce
	headers["signature"] = common.Base64Encode(hmac)

	resp, err := common.SendHTTPRequest(reqType, BTCMARKETS_API_URL+path, headers, bytes.NewBuffer(payload))

	if err != nil {
		return err
	}

	if b.Verbose {
		log.Printf("Recieved raw: %s\n", resp)
	}

	err = common.JSONDecode([]byte(resp), &result)

	if err != nil {
		return err
	}

	return nil
}
