package main

import (
	"errors"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	BITSTAMP_API_URL                 = "https://www.bitstamp.net/api/"
	BITSTAMP_API_VERSION             = "0"
	BITSTAMP_API_TICKER              = "ticker/"
	BITSTAMP_API_ORDERBOOK           = "order_book/"
	BITSTAMP_API_TRANSACTIONS        = "transactions/"
	BITSTAMP_API_EURUSD              = "eur_usd/"
	BITSTAMP_API_BALANCE             = "balance/"
	BITSTAMP_API_USER_TRANSACTIONS   = "user_transactions/"
	BITSTAMP_API_OPEN_ORDERS         = "open_orders/"
	BITSTAMP_API_CANCEL_ORDER        = "cancel_order/"
	BITSTAMP_API_BUY                 = "buy/"
	BITSTAMP_API_SELL                = "sell/"
	BITSTAMP_API_WITHDRAWAL_REQUESTS = "withdrawal_requests/"
	BITSTAMP_API_BITCOIN_WITHDRAWAL  = "bitcoin_withdrawal/"
	BITSTAMP_API_BITCOIN_DEPOSIT     = "bitcoin_deposit_address/"
	BITSTAMP_API_UNCONFIRMED_BITCOIN = "unconfirmed_btc/"
	BITSTAMP_API_RIPPLE_WITHDRAWAL   = "ripple_withdrawal/"
	BITSTAMP_API_RIPPLE_DESPOIT      = "ripple_address/"
)

type Bitstamp struct {
	Name                        string
	Enabled                     bool
	Verbose                     bool
	Websocket                   bool
	RESTPollingDelay            time.Duration
	ClientID, APIKey, APISecret string
	Ticker                      BitstampTicker
	Orderbook                   Orderbook
	ConversionRate              ConversionRate
	Transactions                []Transactions
	Balance                     BitstampAccountBalance
	TakerFee, MakerFee          float64
	BaseCurrencies              []string
	Pairs                       []string
}

type BitstampTicker struct {
	Last   float64 `json:",string"`
	High   float64 `json:",string"`
	Low    float64 `json:",string"`
	Vwap   float64 `json:",string"`
	Volume float64 `json:",string"`
	Bid    float64 `json:",string"`
	Ask    float64 `json:",string"`
}

type BitstampAccountBalance struct {
	BTCReserved  float64 `json:"usd_balance,string"`
	Fee          float64 `json:",string"`
	BTCAvailable float64 `json:"btc_balance,string"`
	USDReserved  float64 `json:"usd_reserved,string"`
	BTCBalance   float64 `json:"btc_balance,string"`
	USDBalance   float64 `json:"usd_balance,string"`
	USDAvailable float64 `json:"usd_available,string"`
}

type Orderbook struct {
	Timestamp string
	Bids      [][]string
	Asks      [][]string
}

type Transactions struct {
	Date, Price, Amount string
	Tid                 int64
}

type ConversionRate struct {
	Buy  string
	Sell string
}

func (b *Bitstamp) SetDefaults() {
	b.Name = "Bitstamp"
	b.Enabled = true
	b.Verbose = false
	b.Websocket = false
	b.RESTPollingDelay = 10
}

func (b *Bitstamp) GetName() string {
	return b.Name
}

func (b *Bitstamp) SetEnabled(enabled bool) {
	b.Enabled = enabled
}

func (b *Bitstamp) IsEnabled() bool {
	return b.Enabled
}

func (b *Bitstamp) GetFee() float64 {
	return b.Balance.Fee
}

func (b *Bitstamp) SetAPIKeys(clientID, apiKey, apiSecret string) {
	b.ClientID = clientID
	b.APIKey = apiKey
	b.APISecret = apiSecret
}

func (b *Bitstamp) Run() {
	if b.Verbose {
		log.Printf("%s Websocket: %s.", b.GetName(), IsEnabled(b.Websocket))
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
	}

	b.GetBalance()

	if b.Websocket {
		go b.PusherClient()
	}

	for b.Enabled {
		go func() {
			BitstampBTC := b.GetTicker()
			log.Printf("Bitstamp BTC: Last %f High %f Low %f Volume %f\n", BitstampBTC.Last, BitstampBTC.High, BitstampBTC.Low, BitstampBTC.Volume)
			AddExchangeInfo(b.GetName(), "BTC", BitstampBTC.Last, BitstampBTC.Volume)
		}()
		time.Sleep(time.Second * b.RESTPollingDelay)
	}
}

func (b *Bitstamp) GetTicker() BitstampTicker {
	err := SendHTTPGetRequest(BITSTAMP_API_URL+BITSTAMP_API_TICKER, true, &b.Ticker)

	if err != nil {
		log.Println(err)
		return BitstampTicker{}
	}

	return b.Ticker
}

func (b *Bitstamp) GetOrderbook() {
	err := SendHTTPGetRequest(BITSTAMP_API_URL+BITSTAMP_API_ORDERBOOK, true, &b.Orderbook)

	if err != nil {
		log.Println(err)
		return
	}
}

func (b *Bitstamp) GetTransactions() {
	err := SendHTTPGetRequest(BITSTAMP_API_URL+BITSTAMP_API_TRANSACTIONS, true, &b.Transactions)

	if err != nil {
		log.Println(err)
		return
	}
}

func (b *Bitstamp) GetEURUSDConversionRate() {
	err := SendHTTPGetRequest(BITSTAMP_API_URL+BITSTAMP_API_EURUSD, true, &b.ConversionRate)

	if err != nil {
		log.Println(err)
		return
	}
}

func (b *Bitstamp) GetBalance() {
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_BALANCE, url.Values{}, &b.Balance)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitstamp) GetUserTransactions(offset, limit, sort int64) {
	var req = url.Values{}

	req.Add("offset", strconv.FormatInt(offset, 10))
	req.Add("limit", strconv.FormatInt(limit, 10))
	req.Add("sort", strconv.FormatInt(sort, 10))

	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_USER_TRANSACTIONS, req, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitstamp) CancelOrder(OrderID int64) {
	var req = url.Values{}
	req.Add("id", strconv.FormatInt(OrderID, 10))

	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_CANCEL_ORDER, req, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitstamp) GetOpenOrders() {
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_OPEN_ORDERS, url.Values{}, nil)

	if err != nil {
		log.Println(err)
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

	err := b.SendAuthenticatedHTTPRequest(orderType, req, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitstamp) GetWithdrawalRequests() {
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_WITHDRAWAL_REQUESTS, url.Values{}, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitstamp) BitcoinWithdrawal(amount float64, address string) {
	var req = url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', 8, 64))
	req.Add("address", address)

	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_BITCOIN_WITHDRAWAL, req, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitstamp) BitcoinDepositAddress() {
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_BITCOIN_DEPOSIT, url.Values{}, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitstamp) UnconfirmedBitcoin() {
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_UNCONFIRMED_BITCOIN, url.Values{}, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitstamp) RippleWithdrawal(amount float64, address, currency string) {
	var req = url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', 8, 64))
	req.Add("address", address)
	req.Add("currency", currency)

	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_RIPPLE_WITHDRAWAL, req, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitstamp) RippleDepositAddress() {
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_RIPPLE_DESPOIT, url.Values{}, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *Bitstamp) SendAuthenticatedHTTPRequest(path string, values url.Values, result interface{}) (err error) {
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)
	values.Set("key", b.APIKey)
	values.Set("nonce", nonce)
	hmac := GetHMAC(HASH_SHA256, []byte(nonce+b.ClientID+b.APIKey), []byte(b.APISecret))
	values.Set("signature", strings.ToUpper(HexEncodeToString(hmac)))
	path = BITSTAMP_API_URL + path

	if b.Verbose {
		log.Println("Sending POST request to " + path)
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := SendHTTPRequest("POST", path, headers, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}

	if b.Verbose {
		log.Printf("Recieved raw: %s\n", resp)
	}

	err = JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("Unable to JSON Unmarshal response.")
	}

	return nil
}
