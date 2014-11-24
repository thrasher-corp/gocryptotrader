package main

import (
	"net/http"
	"net/url"
	"io/ioutil"
	"fmt"
	"log"
	"encoding/hex"
	"crypto/hmac"
	"crypto/sha256"
	"strings"
	"strconv"
	"errors"
	"time"
)

const (
	BITSTAMP_API_URL = "https://www.bitstamp.net/api/"
	BITSTAMP_API_TICKER = "ticker/"
	BITSTAMP_API_ORDERBOOK = "order_book/"
	BITSTAMP_API_TRANSACTIONS = "transactions/"
	BITSTAMP_API_EURUSD = "eur_usd/"
	BITSTAMP_API_BALANCE = "balance/"
	BITSTAMP_API_USER_TRANSACTIONS = "user_transactions/"
	BITSTAMP_API_OPEN_ORDERS = "open_orders/"
	BITSTAMP_API_CANCEL_ORDER = "cancel_order/"
	BITSTAMP_API_BUY = "buy/"
	BITSTAMP_API_SELL = "sell/"
	BITSTAMP_API_WITHDRAWAL_REQUESTS = "withdrawal_requests/"
	BITSTAMP_API_BITCOIN_WITHDRAWAL = "bitcoin_withdrawal/"
	BITSTAMP_API_BITCOIN_DEPOSIT = "bitcoin_deposit_address/"
	BITSTAMP_API_UNCONFIRMED_BITCOIN = "unconfirmed_btc/"
	BITSTAMP_API_RIPPLE_WITHDRAWAL = "ripple_withdrawal/"
	BITSTAMP_API_RIPPLE_DESPOIT = "ripple_address/"
)

type Bitstamp struct { 
	ClientID, APIKey, APISecret string
	Ticker BitstampTicker
	Orderbook Orderbook
	ConversionRate ConversionRate
	Transactions []Transactions
}

type BitstampTicker struct {
	Last float64 `json:",string"`
	High float64 `json:",string"`
	Low float64 `json:",string"`
	Vwap float64 `json:",string"`
	Volume float64 `json:",string"`
	Bid float64 `json:",string"`
	Ask float64 `json:",string"`
}

type Orderbook struct {
	Timestamp string
	Bids [][]string
	Asks [][]string
}

type Transactions struct {
	Date, Price, Amount string
	Tid int64
}

type ConversionRate struct {
	Buy string
	Sell string
}

func (b *Bitstamp) GetTicker() (BitstampTicker) {
	err := SendHTTPRequest(BITSTAMP_API_URL + BITSTAMP_API_TICKER, true, &b.Ticker)

	if err != nil {
		fmt.Println(err) 
		return BitstampTicker{}
	}

	return b.Ticker
}

func (b *Bitstamp) GetOrderbook() {
	err := SendHTTPRequest(BITSTAMP_API_URL + BITSTAMP_API_ORDERBOOK, true, &b.Orderbook)

	if err != nil {
		fmt.Println(err) 
		return
	}
}

func (b *Bitstamp) GetTransactions() {
	err := SendHTTPRequest(BITSTAMP_API_URL + BITSTAMP_API_TRANSACTIONS, true, &b.Transactions)

	if err != nil {
		fmt.Println(err) 
		return
	}
}

func (b *Bitstamp) GetEURUSDConversionRate() {
	err := SendHTTPRequest(BITSTAMP_API_URL + BITSTAMP_API_EURUSD, true, &b.ConversionRate)

	if err != nil {
		fmt.Println(err) 
		return
	}
}

func (b *Bitstamp) GetBalance() {
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_BALANCE, url.Values{})

	if err != nil {
		fmt.Println(err)
	}
}

func (b *Bitstamp) GetUserTransactions(offset, limit, sort int64) {
	var req = url.Values{}

	req.Add("offset", strconv.FormatInt(offset, 10))
	req.Add("limit", strconv.FormatInt(limit, 10))
	req.Add("sort", strconv.FormatInt(sort, 10))

	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_USER_TRANSACTIONS, req)

	if err != nil {
		fmt.Println(err)
	}
}

func (b *Bitstamp) CancelOrder(OrderID int64) {
	var req = url.Values{}
	req.Add("id", strconv.FormatInt(OrderID, 10))

	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_CANCEL_ORDER, req)

	if err != nil {
		fmt.Println(err)
	}
}

func (b *Bitstamp) GetOpenOrders() {
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_OPEN_ORDERS, url.Values{})

	if err != nil {
		fmt.Println(err)
	}
}

func (b *Bitstamp) PlaceOrder(price float64, amount float64, Type int) {
	var req = url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', 8, 64))
	req.Add("price", strconv.FormatFloat(price, 'f', 2, 64))
	orderType := BITSTAMP_API_BUY

	if Type == 1 {
		orderType = BITSTAMP_API_SELL
	} 

	log.Printf("Placing %s order at price %f for %f amount.\n", orderType, price, amount)

	err := b.SendAuthenticatedHTTPRequest(orderType, req)

	if err != nil {
		fmt.Println(err)
	}
}

func (b *Bitstamp) GetWithdrawalRequests() {
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_WITHDRAWAL_REQUESTS, url.Values{})

	if err != nil {
		fmt.Println(err)
	}
}

func (b *Bitstamp) BitcoinWithdrawal(amount float64, address string) {
	var req = url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', 8, 64))
	req.Add("address", address)

	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_BITCOIN_WITHDRAWAL, req)

	if err != nil {
		fmt.Println(err)
	}
}

func (b *Bitstamp) BitcoinDepositAddress() {
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_BITCOIN_DEPOSIT, url.Values{})

	if err != nil {
		fmt.Println(err)
	}
}

func (b *Bitstamp) UnconfirmedBitcoin() {
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_UNCONFIRMED_BITCOIN, url.Values{})

	if err != nil {
		fmt.Println(err)
	}
}

func (b *Bitstamp) RippleWithdrawal(amount float64, address, currency string) {
	var req = url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', 8, 64))
	req.Add("address", address)
	req.Add("currency", currency)

	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_RIPPLE_WITHDRAWAL, req)

	if err != nil {
		fmt.Println(err)
	}
}

func (b *Bitstamp) RippleDepositAddress() {
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_RIPPLE_DESPOIT, url.Values{})

	if err != nil {
		fmt.Println(err)
	}
}

func (b *Bitstamp) SendAuthenticatedHTTPRequest(path string, values url.Values) (err error) {
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)
	values.Set("key", b.APIKey)
	values.Set("nonce", nonce)
	hmac := hmac.New(sha256.New, []byte(b.APISecret))
	hmac.Write([]byte(nonce + b.ClientID + b.APIKey))
	values.Set("signature", strings.ToUpper(hex.EncodeToString(hmac.Sum(nil))))

	reqBody := strings.NewReader(values.Encode())

	path = BITSTAMP_API_URL + path
	fmt.Println("Sending POST request to " + path)

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