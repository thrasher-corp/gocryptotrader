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
	BITSTAMP_API_TICKER_HOURLY       = "ticker_hour/"
	BITSTAMP_API_ORDERBOOK           = "order_book/"
	BITSTAMP_API_TRANSACTIONS        = "transactions/"
	BITSTAMP_API_EURUSD              = "eur_usd/"
	BITSTAMP_API_BALANCE             = "balance/"
	BITSTAMP_API_USER_TRANSACTIONS   = "user_transactions/"
	BITSTAMP_API_OPEN_ORDERS         = "open_orders/"
	BITSTAMP_API_ORDER_STATUS        = "order_status"
	BITSTAMP_API_CANCEL_ORDER        = "cancel_order/"
	BITSTAMP_API_CANCEL_ALL_ORDERS   = "cancel_all_orders/"
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
	AuthenticatedAPISupport     bool
	ClientID, APIKey, APISecret string
	Balance                     BitstampAccountBalance
	TakerFee, MakerFee          float64
	BaseCurrencies              []string
	AvailablePairs              []string
	EnabledPairs                []string
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

type BitstampOrderbookBase struct {
	Price  float64
	Amount float64
}

type BitstampOrderbook struct {
	Timestamp int64 `json:"timestamp,string"`
	Bids      []BitstampOrderbookBase
	Asks      []BitstampOrderbookBase
}

type BitstampTransactions struct {
	Date    int64   `json:"date,string"`
	TradeID int64   `json:"tid"`
	Price   float64 `json:"price,string"`
	Type    int     `json:"type"`
	Amount  float64 `json:"amount,string"`
}

type BitstampEURUSDConversionRate struct {
	Buy  float64 `json:"buy,string"`
	Sell float64 `json:"sell,string"`
}

type BitstampUserTransactions struct {
	Date    string      `json:"datetime"`
	TransID int64       `json:"id"`
	Type    int         `json:"type"`
	USD     float64     `json:"usd,string"`
	BTC     float64     `json:"btc,string"`
	BTCUSD  float64     `json:"btc_usd,string"`
	Fee     float64     `json:"fee,string"`
	OrderID interface{} `json:"order_id"`
}

type BitstampOrder struct {
	ID     int64   `json:"id"`
	Date   string  `json:"date"`
	Type   int     `json:"type"`
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"`
}

type BitstampOrderStatus struct {
	Status       string
	Transactions []struct {
		TradeID int64   `json:"tid"`
		USD     float64 `json:"usd,string"`
		Price   float64 `json:"price,string"`
		Fee     float64 `json:"fee,string"`
		BTC     float64 `json:"btc,string"`
	}
}

type BitstampWithdrawalRequests struct {
	OrderID int64   `json:"id"`
	Date    string  `json:"datetime"`
	Type    int     `json:"type"`
	Amount  float64 `json:"amount,string"`
	Status  int     `json:"status"`
	Data    interface{}
}

type BitstampUnconfirmedBTCTransactions struct {
	Amount        float64 `json:"amount,string"`
	Address       string  `json:"address"`
	Confirmations int     `json:"confirmations"`
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
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	if b.Websocket {
		go b.PusherClient()
	}

	for b.Enabled {
		for _, x := range b.EnabledPairs {
			currency := x
			go func() {
				ticker, err := b.GetTicker(true)
				if err != nil {
					log.Println(err)
					return
				}
				log.Printf("Bitstamp %s: Last %f High %f Low %f Volume %f\n", currency, ticker.Last, ticker.High, ticker.Low, ticker.Volume)
				AddExchangeInfo(b.GetName(), currency[0:3], currency[3:], ticker.Last, ticker.Volume)
			}()
		}
		time.Sleep(time.Second * b.RESTPollingDelay)
	}
}

func (b *Bitstamp) GetTicker(hourly bool) (BitstampTicker, error) {
	path := BITSTAMP_API_URL
	ticker := BitstampTicker{}

	if hourly {
		path += BITSTAMP_API_TICKER_HOURLY
	} else {
		path += BITSTAMP_API_TICKER
	}

	err := SendHTTPGetRequest(path, true, &ticker)

	if err != nil {
		return ticker, err
	}

	return ticker, nil
}

func (b *Bitstamp) GetOrderbook() (BitstampOrderbook, error) {
	type response struct {
		Timestamp int64 `json:"timestamp,string"`
		Bids      [][]string
		Asks      [][]string
	}

	resp := response{}
	err := SendHTTPGetRequest(BITSTAMP_API_URL+BITSTAMP_API_ORDERBOOK, true, &resp)
	if err != nil {
		return BitstampOrderbook{}, err
	}

	orderbook := BitstampOrderbook{}
	orderbook.Timestamp = resp.Timestamp

	for _, x := range resp.Bids {
		price, err := strconv.ParseFloat(x[0], 64)
		if err != nil {
			log.Println(err)
			continue
		}
		amount, err := strconv.ParseFloat(x[1], 64)
		if err != nil {
			log.Println(err)
			continue
		}
		orderbook.Bids = append(orderbook.Bids, BitstampOrderbookBase{price, amount})
	}

	for _, x := range resp.Asks {
		price, err := strconv.ParseFloat(x[0], 64)
		if err != nil {
			log.Println(err)
			continue
		}
		amount, err := strconv.ParseFloat(x[1], 64)
		if err != nil {
			log.Println(err)
			continue
		}
		orderbook.Asks = append(orderbook.Asks, BitstampOrderbookBase{price, amount})
	}

	return orderbook, nil
}

func (b *Bitstamp) GetTransactions(values url.Values) ([]BitstampTransactions, error) {
	path := EncodeURLValues(BITSTAMP_API_URL+BITSTAMP_API_TRANSACTIONS, values)
	transactions := []BitstampTransactions{}
	err := SendHTTPGetRequest(path, true, &transactions)
	if err != nil {
		return nil, err
	}
	return transactions, nil
}

func (b *Bitstamp) GetEURUSDConversionRate() (BitstampEURUSDConversionRate, error) {
	rate := BitstampEURUSDConversionRate{}
	err := SendHTTPGetRequest(BITSTAMP_API_URL+BITSTAMP_API_EURUSD, true, &rate)

	if err != nil {
		return rate, err
	}
	return rate, nil
}

func (b *Bitstamp) GetBalance() (BitstampAccountBalance, error) {
	balance := BitstampAccountBalance{}
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_BALANCE, url.Values{}, &balance)

	if err != nil {
		return balance, err
	}
	return balance, nil
}

func (b *Bitstamp) GetUserTransactions(values url.Values) ([]BitstampUserTransactions, error) {
	response := []BitstampUserTransactions{}
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_USER_TRANSACTIONS, values, &response)

	if err != nil {
		return nil, err
	}

	return response, nil
}

func (b *Bitstamp) GetOpenOrders() ([]BitstampOrder, error) {
	resp := []BitstampOrder{}
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_OPEN_ORDERS, nil, &resp)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (b *Bitstamp) GetOrderStatus(OrderID int64) (BitstampOrderStatus, error) {
	var req = url.Values{}
	req.Add("id", strconv.FormatInt(OrderID, 10))
	resp := BitstampOrderStatus{}

	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_CANCEL_ORDER, req, &resp)

	if err != nil {
		return resp, err
	}

	return resp, nil
}

func (b *Bitstamp) CancelOrder(OrderID int64) (bool, error) {
	var req = url.Values{}
	result := false
	req.Add("id", strconv.FormatInt(OrderID, 10))

	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_CANCEL_ORDER, req, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (b *Bitstamp) CancelAllOrders() (bool, error) {
	result := false
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_CANCEL_ALL_ORDERS, nil, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (b *Bitstamp) PlaceOrder(price float64, amount float64, buy bool) (BitstampOrder, error) {
	var req = url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("price", strconv.FormatFloat(price, 'f', -1, 64))
	response := BitstampOrder{}
	orderType := BITSTAMP_API_BUY

	if !buy {
		orderType = BITSTAMP_API_SELL
	}

	err := b.SendAuthenticatedHTTPRequest(orderType, req, &response)

	if err != nil {
		return response, err
	}

	return response, nil
}

func (b *Bitstamp) GetWithdrawalRequests() ([]BitstampWithdrawalRequests, error) {
	resp := []BitstampWithdrawalRequests{}
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_WITHDRAWAL_REQUESTS, url.Values{}, &resp)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (b *Bitstamp) BitcoinWithdrawal(amount float64, address string) (string, error) {
	var req = url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("address", address)

	type response struct {
		ID string `json:"id"`
	}

	resp := response{}
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_BITCOIN_WITHDRAWAL, req, &resp)

	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (b *Bitstamp) GetBitcoinDepositAddress() (string, error) {
	address := ""
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_BITCOIN_DEPOSIT, url.Values{}, &address)

	if err != nil {
		return address, err
	}
	return address, nil
}

func (b *Bitstamp) GetUnconfirmedBitcoinDeposits() ([]BitstampUnconfirmedBTCTransactions, error) {
	response := []BitstampUnconfirmedBTCTransactions{}
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_UNCONFIRMED_BITCOIN, nil, &response)

	if err != nil {
		return nil, err
	}

	return response, nil
}

func (b *Bitstamp) RippleWithdrawal(amount float64, address, currency string) (bool, error) {
	var req = url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("address", address)
	req.Add("currency", currency)

	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_RIPPLE_WITHDRAWAL, req, nil)

	if err != nil {
		return false, err
	}

	return true, nil
}

func (b *Bitstamp) GetRippleDepositAddress() (string, error) {
	type response struct {
		Address string
	}

	resp := response{}
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_RIPPLE_DESPOIT, nil, &resp)

	if err != nil {
		return "", err
	}

	return resp.Address, nil
}

func (b *Bitstamp) SendAuthenticatedHTTPRequest(path string, values url.Values, result interface{}) (err error) {
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)

	if values == nil {
		values = url.Values{}
	}

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
