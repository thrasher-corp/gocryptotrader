package main

import (
	"net/http"
	"io/ioutil"
	"fmt"
	"log"
	"encoding/json"
	"encoding/hex"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"strings"
	"strconv"
	"time"
)

const (
	BITFINEX_API_URL = "https://api.bitfinex.com/v1/"
	BITFINEX_TICKER = "pubticker/"
	BITFINEX_STATS = "stats/"
	BITFINEX_ORDERBOOK = "book/"
	BITFINEX_TRADES = "trades/"
	BITFINEX_SYMBOLS = "symbols/"
	BITFINEX_SYMBOLS_DETAILS = "symbols_details/"
	BITFINEX_DEPOSIT = "deposit/new"
	BITFINEX_ORDER_NEW = "order/new"
	BITFINEX_ORDER_CANCEL = "order/cancel"
	BITFINEX_ORDER_CANCEL_ALL = "order/cancel/all"
	BITFINEX_ORDER_STATUS = "order/status"
	BITFINEX_ORDERS = "orders"
	BITFINEX_POSITIONS = "positions"
	BITFINEX_CLAIM_POSITION = "position/claim"
	BITFINEX_HISTORY = "history"
	BITFINEX_TRADE_HISTORY = "mytrades"
	BITFINEX_ACCOUNT_INFO = "account_infos"
)

type BitfinexStats struct {
	Period int64
	Volume float64 `json:",string"`
}

type BitfinexTicker struct {
	Mid float64 `json:",string"`
	Bid float64 `json:",string"`
	Ask float64 `json:",string"`
	Last float64 `json:"Last_price,string"`
	Low float64 `json:",string"`
	High float64 `json:",string"`
	Volume float64 `json:",string"`
	Timestamp string
}

type BookStructure struct {
	Price, Amount, Timestamp string
}

type BitfinexFee struct {
	Currency string
	TakerFees float64
	MakerFees float64
}

type BitfinexOrderbook struct {
	Bids []BookStructure
	Asks []BookStructure
}

type TradeStructure struct {
	Timestamp, Tid int64
	Price, Amount, Exchange, Type string
}

type SymbolsDetails struct {
	Pair, Initial_margin, Minimum_margin, Maximum_order_size, Minimum_order_size, Expiration string
	Price_precision int
}

type Bitfinex struct {
	Name string
	Enabled bool
	Verbose bool
	APIKey, APISecret string
	Ticker BitfinexTicker
	Stats []BitfinexStats
	Orderbook BitfinexOrderbook
	Trades []TradeStructure
	SymbolsDetails []SymbolsDetails
	Fees []BitfinexFee
}

func (b *Bitfinex) SetDefaults() {
	b.Name = "Bitfinex"
	b.Enabled = true
	b.Verbose = false
}

func (b *Bitfinex) GetName() (string) {
	return b.Name
}

func (b *Bitfinex) SetEnabled(enabled bool) {
	b.Enabled = enabled
}

func (b *Bitfinex) IsEnabled() (bool) {
	return b.Enabled
}

func (b *Bitfinex) SetAPIKeys(apiKey, apiSecret string) {
	b.APIKey = apiKey
	b.APISecret = apiSecret
}

func (b *Bitfinex) GetFee(maker bool, symbol string) (float64, error) {
	for _, i := range b.Fees {
		if symbol == i.Currency {
			if maker {
				return i.MakerFees, nil
			} else {
				return i.TakerFees, nil
			}
		}
	}
	return 0, errors.New("Unable to find specified currency.")
}

func (b *Bitfinex) GetAccountFeeInfo() (bool, error) {
	type Fee struct {
		Pairs string `json:"pairs"`
		MakerFees string `json:"maker_fees"`
		TakerFees string `json:"taker_fees"`
	}

	type Response struct {
		Data []map[string][]Fee
	}

	var resp Response
	err := b.SendAuthenticatedHTTPRequest(BITFINEX_ACCOUNT_INFO, nil, &resp.Data)

	if err != nil {
		return false, err
	}

	Fees := []BitfinexFee{}

	for _, i := range resp.Data[0]["fees"] {
		var bfxFee BitfinexFee
		bfxFee.Currency = i.Pairs
		bfxFee.MakerFees, _ = strconv.ParseFloat(i.MakerFees, 64)
		bfxFee.TakerFees, _ = strconv.ParseFloat(i.TakerFees, 64)
		Fees = append(Fees, bfxFee)
	}

	b.Fees = Fees
	return true, nil
}

func (b *Bitfinex) SendAuthenticatedHTTPRequest(path string, params map[string]interface{}, result interface{}) (err error) {
	request := make(map[string]interface{})
	request["request"] = "/v1/" + path
	request["nonce"] = strconv.FormatInt(time.Now().UnixNano(), 10)

	if params != nil {
		for key, value:= range params {
			request[key] = value
		}
	}

	PayloadJson, err := json.Marshal(request)

	if b.Verbose {
		log.Printf("Request JSON: %s\n", PayloadJson)
	}

	if err != nil {
		return errors.New("SendAuthenticatedHTTPRequest: Unable to JSON request")
	}

	PayloadBase64 := base64.StdEncoding.EncodeToString(PayloadJson)
	hmac := hmac.New(sha512.New384, []byte(b.APISecret))
	hmac.Write([]byte(PayloadBase64))
	signature := hex.EncodeToString(hmac.Sum(nil))
	method := "GET"

	if strings.Contains(path, BITFINEX_ORDER_CANCEL_ALL) {
		method = "POST"
	} 

	req, err := http.NewRequest(method, BITFINEX_API_URL + path, strings.NewReader(""))
	req.Header.Set("X-BFX-APIKEY", string(b.APIKey))
	req.Header.Set("X-BFX-PAYLOAD", PayloadBase64)
	req.Header.Set("X-BFX-SIGNATURE", signature)

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return errors.New("SendAuthenticatedHTTPRequest: Unable to send request")
	}

	contents, _ := ioutil.ReadAll(resp.Body)

	if b.Verbose {
		log.Printf("Recieved raw: \n%s\n", string(contents))
	}
	
	err = json.Unmarshal(contents, &result)

	if err != nil {
		return errors.New("Unable to JSON response.")
	}

	resp.Body.Close()
	return nil
}

func (b *Bitfinex) GetTicker(symbol string) (BitfinexTicker) {
	err := SendHTTPRequest(BITFINEX_API_URL + BITFINEX_TICKER + symbol, true, &b.Ticker)
	if err != nil {
		fmt.Println(err)
		return BitfinexTicker{}
	}
	return b.Ticker
}

func (b *Bitfinex) GetStats(symbol string) (bool) {
	err := SendHTTPRequest(BITFINEX_API_URL + BITFINEX_STATS + symbol, true, &b.Stats)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (b *Bitfinex) GetOrderbook(symbol string) (bool) {
	err := SendHTTPRequest(BITFINEX_API_URL + BITFINEX_ORDERBOOK + symbol, true, &b.Orderbook)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (b *Bitfinex) GetTrades(symbol string) (bool) {
	err := SendHTTPRequest(BITFINEX_API_URL + BITFINEX_TRADES + symbol, true, &b.Trades)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (b *Bitfinex) GetSymbols() (bool) {
	err := SendHTTPRequest(BITFINEX_API_URL + BITFINEX_SYMBOLS, false, nil)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (b *Bitfinex) GetSymbolsDetails() (bool) {
	err := SendHTTPRequest(BITFINEX_API_URL + BITFINEX_SYMBOLS_DETAILS, false, &b.SymbolsDetails)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (b *Bitfinex) NewDeposit(Symbol, Method, Wallet string) {
	request := make(map[string]interface{})
	request["currency"] = Symbol
	request["method"] = Method
	request["wallet_name"] = Wallet

	err := b.SendAuthenticatedHTTPRequest(BITFINEX_DEPOSIT, request, nil)

	if err != nil {
		fmt.Println(err)
	}
}

func (b *Bitfinex) NewOrder(Symbol string, Amount float64, Price float64, Buy bool, Type string, Hidden bool) {
	request := make(map[string]interface{})
	request["symbol"] = Symbol
	request["amount"] = fmt.Sprintf("%.8f", Amount)
	request["price"] = fmt.Sprintf("%.5f", Price)
	request["exchange"] = "bitfinex"

	if Buy {
		request["side"] = "buy"
	} else {
		request["side"] = "sell"
	}

	//request["is_hidden"] - currently not implemented
	request["type"] = Type
	
	err := b.SendAuthenticatedHTTPRequest(BITFINEX_ORDER_NEW, request, nil)

	if err != nil {
		fmt.Println(err)
	}
}

func (b *Bitfinex) CancelOrder(OrderID int) {
	request := make(map[string]interface{})
	request["order_id"] = OrderID

	err := b.SendAuthenticatedHTTPRequest(BITFINEX_ORDER_CANCEL, request, nil)

	if err != nil {
		fmt.Println(err)
	}
}

func (b *Bitfinex) CancelMultiplateOrders(OrderIDs []int) {
	request := make(map[string]interface{})
	request["order_ids"] = OrderIDs

	err := b.SendAuthenticatedHTTPRequest(BITFINEX_ORDER_CANCEL, request, nil)

	if err != nil {
		fmt.Println(err)
	}
}

func (b *Bitfinex) CancelAllOrders() {
	err := b.SendAuthenticatedHTTPRequest(BITFINEX_ORDER_CANCEL_ALL, nil, nil)

	if err != nil {
		fmt.Println(err)
	}
}

func (b *Bitfinex) ReplaceOrder(OrderID int) {
	request := make(map[string]interface{})
	request["order_id"] = OrderID
}

func (b *Bitfinex) GetOrderStatus(OrderID int) {
	request := make(map[string]interface{})
	request["order_id"] = OrderID

	err := b.SendAuthenticatedHTTPRequest(BITFINEX_ORDER_STATUS, request, nil)

	if err != nil {
		fmt.Println(err)
	}
}

func (b *Bitfinex) GetActiveOrders() {
	err := b.SendAuthenticatedHTTPRequest(BITFINEX_ORDERS, nil, nil)

	if err != nil {
		fmt.Println(err)
	}
}

func (b *Bitfinex) GetActivePositions() {
	err := b.SendAuthenticatedHTTPRequest(BITFINEX_POSITIONS, nil, nil)

	if err != nil {
		fmt.Println(err)
	}
}

func (b *Bitfinex) ClaimPosition(PositionID int) {
	request := make(map[string]interface{})
	request["position_id"] = PositionID

	err := b.SendAuthenticatedHTTPRequest(BITFINEX_CLAIM_POSITION, nil, nil)

	if err != nil {
		fmt.Println(err)
	}
}

func (b *Bitfinex) GetBalanceHistory(symbol string, timeSince time.Time, timeUntil time.Time, limit int, wallet string) {
	request := make(map[string]interface{})
	request["currency"] = symbol
	request["since"] = timeSince
	request["until"] = timeUntil

	if limit > 0 {
		request["limit"] = limit
	}
	
	if len(wallet) > 0 {
		request["wallet"] = wallet
	}

	err := b.SendAuthenticatedHTTPRequest(BITFINEX_HISTORY, request, nil)

	if err != nil {
		fmt.Println(err)
	}
}

func (b *Bitfinex) GetTradeHistory(symbol string, timestamp time.Time, limit int) {
	request := make(map[string]interface{})
	request["currency"] = symbol
	request["timestamp"] = timestamp

	if (limit > 0) {
		request["limit_trades"] = limit
	}

	err := b.SendAuthenticatedHTTPRequest(BITFINEX_TRADE_HISTORY, nil, nil)

	if err != nil {
		fmt.Println(err)
	}
}

