package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges"
)

const (
	BITSTAMP_API_URL                 = "https://www.bitstamp.net/api"
	BITSTAMP_API_VERSION             = "2"
	BITSTAMP_API_TICKER              = "ticker"
	BITSTAMP_API_TICKER_HOURLY       = "ticker_hour"
	BITSTAMP_API_ORDERBOOK           = "order_book"
	BITSTAMP_API_TRANSACTIONS        = "transactions"
	BITSTAMP_API_EURUSD              = "eur_usd"
	BITSTAMP_API_BALANCE             = "balance"
	BITSTAMP_API_USER_TRANSACTIONS   = "user_transactions"
	BITSTAMP_API_OPEN_ORDERS         = "open_orders"
	BITSTAMP_API_ORDER_STATUS        = "order_status"
	BITSTAMP_API_CANCEL_ORDER        = "cancel_order"
	BITSTAMP_API_CANCEL_ALL_ORDERS   = "cancel_all_orders"
	BITSTAMP_API_BUY                 = "buy"
	BITSTAMP_API_SELL                = "sell"
	BITSTAMP_API_MARKET              = "market"
	BITSTAMP_API_WITHDRAWAL_REQUESTS = "withdrawal_requests"
	BITSTAMP_API_BITCOIN_WITHDRAWAL  = "bitcoin_withdrawal"
	BITSTAMP_API_BITCOIN_DEPOSIT     = "bitcoin_deposit_address"
	BITSTAMP_API_UNCONFIRMED_BITCOIN = "unconfirmed_btc"
	BITSTAMP_API_RIPPLE_WITHDRAWAL   = "ripple_withdrawal"
	BITSTAMP_API_RIPPLE_DESPOIT      = "ripple_address"
	BITSTAMP_API_TRANSFER_TO_MAIN    = "transfer-to-main"
	BITSTAMP_API_TRANSFER_FROM_MAIN  = "transfer-from-main"
	BITSTAMP_API_XRP_WITHDRAWAL      = "xrp_withdrawal"
	BITSTAMP_API_XRP_DESPOIT         = "xrp_address"
)

type Bitstamp struct {
	exchange.ExchangeBase
	Balance BitstampBalances
}

type BitstampTicker struct {
	Last      float64 `json:"last,string"`
	High      float64 `json:"high,string"`
	Low       float64 `json:"low,string"`
	Vwap      float64 `json:"vwap,string"`
	Volume    float64 `json:"volume,string"`
	Bid       float64 `json:"bid,string"`
	Ask       float64 `json:"ask,string"`
	Timestamp int64   `json:"timestamp,string"`
	Open      float64 `json:"open,string"`
}

type BitstampBalances struct {
	BTCReserved  float64 `json:"btc_reserved,string"`
	BTCEURFee    float64 `json:"btceur_fee,string"`
	BTCAvailable float64 `json:"btc_available,string"`
	XRPAvailable float64 `json:"xrp_available,string"`
	EURAvailable float64 `json:"eur_available,string"`
	USDReserved  float64 `json:"usd_reserved,string"`
	EURReserved  float64 `json:"eur_reserved,string"`
	XRPEURFee    float64 `json:"xrpeur_fee,string"`
	XRPReserved  float64 `json:"xrp_reserved,string"`
	XRPBalance   float64 `json:"xrp_balance,string"`
	XRPUSDFee    float64 `json:"xrpusd_fee,string"`
	EURBalance   float64 `json:"eur_balance,string"`
	BTCBalance   float64 `json:"btc_balance,string"`
	BTCUSDFee    float64 `json:"btcusd_fee,string"`
	USDBalance   float64 `json:"usd_balance,string"`
	USDAvailable float64 `json:"usd_available,string"`
	EURUSDFee    float64 `json:"eurusd_fee,string"`
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
	TradeID int64   `json:"tid,string"`
	Price   float64 `json:"price,string"`
	Type    int     `json:"type,string"`
	Amount  float64 `json:"amount,string"`
}

type BitstampEURUSDConversionRate struct {
	Buy  float64 `json:"buy,string"`
	Sell float64 `json:"sell,string"`
}

type BitstampUserTransactions struct {
	Date    string  `json:"datetime"`
	TransID int64   `json:"id"`
	Type    int     `json:"type,string"`
	USD     float64 `json:"usd"`
	EUR     float64 `json:"eur"`
	BTC     float64 `json:"btc"`
	XRP     float64 `json:"xrp"`
	BTCUSD  float64 `json:"btc_usd"`
	Fee     float64 `json:"fee,string"`
	OrderID int64   `json:"order_id"`
}

type BitstampOrder struct {
	ID     int64   `json:"id"`
	Date   string  `json:"datetime"`
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

type BitstampXRPDepositResponse struct {
	Address        string `json:"address"`
	DestinationTag int64  `json:"destination_tag"`
}

func (b *Bitstamp) SetDefaults() {
	b.Name = "Bitstamp"
	b.Enabled = false
	b.Verbose = false
	b.Websocket = false
	b.RESTPollingDelay = 10
}

func (b *Bitstamp) Start() {
	go b.Run()
}

func (b *Bitstamp) GetName() string {
	return b.Name
}

func (b *Bitstamp) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		b.SetEnabled(false)
	} else {
		b.Enabled = true
		b.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		b.SetAPIKeys(exch.ClientID, exch.APIKey, exch.APISecret)
		b.RESTPollingDelay = exch.RESTPollingDelay
		b.Verbose = exch.Verbose
		b.Websocket = exch.Websocket
		b.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		b.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		b.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
	}
}

func (k *Bitstamp) GetEnabledCurrencies() []string {
	return k.EnabledPairs
}

func (b *Bitstamp) SetEnabled(enabled bool) {
	b.Enabled = enabled
}

func (b *Bitstamp) IsEnabled() bool {
	return b.Enabled
}

func (b *Bitstamp) GetFee(currency string) float64 {
	switch currency {
	case "BTCUSD":
		return b.Balance.BTCUSDFee
	case "BTCEUR":
		return b.Balance.BTCEURFee
	case "XRPEUR":
		return b.Balance.XRPEURFee
	case "XRPUSD":
		return b.Balance.XRPUSDFee
	case "EURUSD":
		return b.Balance.EURUSDFee
	default:
		return 0
	}
}

func (b *Bitstamp) SetAPIKeys(clientID, apiKey, apiSecret string) {
	b.ClientID = clientID
	b.APIKey = apiKey
	b.APISecret = apiSecret
}

func (b *Bitstamp) Run() {
	if b.Verbose {
		log.Printf("%s Websocket: %s.", b.GetName(), common.IsEnabled(b.Websocket))
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
				ticker, err := b.GetTickerPrice(currency)
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

func (b *Bitstamp) GetTicker(currency string, hourly bool) (BitstampTicker, error) {
	tickerEndpoint := BITSTAMP_API_TICKER
	if hourly {
		tickerEndpoint = BITSTAMP_API_TICKER_HOURLY
	}

	path := fmt.Sprintf("%s/v%s/%s/%s/", BITSTAMP_API_URL, BITSTAMP_API_VERSION, tickerEndpoint, common.StringToLower(currency))
	ticker := BitstampTicker{}

	err := common.SendHTTPGetRequest(path, true, &ticker)

	if err != nil {
		return ticker, err
	}

	return ticker, nil
}

func (b *Bitstamp) GetTickerPrice(currency string) (TickerPrice, error) {
	tickerNew, err := GetTicker(b.GetName(), currency[0:3], currency[3:])
	if err == nil {
		return tickerNew, nil
	}

	var tickerPrice TickerPrice
	ticker, err := b.GetTicker(currency, false)
	if err != nil {
		return tickerPrice, err

	}
	tickerPrice.Ask = ticker.Ask
	tickerPrice.Bid = ticker.Bid
	tickerPrice.FirstCurrency = currency[0:3]
	tickerPrice.SecondCurrency = currency[3:]
	tickerPrice.CurrencyPair = tickerPrice.FirstCurrency + "_" + tickerPrice.SecondCurrency
	tickerPrice.Low = ticker.Low
	tickerPrice.Last = ticker.Last
	tickerPrice.Volume = ticker.Volume
	tickerPrice.High = ticker.High
	ProcessTicker(b.GetName(), tickerPrice.FirstCurrency, tickerPrice.SecondCurrency, tickerPrice)
	return tickerPrice, nil
}

func (b *Bitstamp) GetOrderbook(currency string) (BitstampOrderbook, error) {
	type response struct {
		Timestamp int64 `json:"timestamp,string"`
		Bids      [][]string
		Asks      [][]string
	}

	resp := response{}
	path := fmt.Sprintf("%s/v%s/%s/%s/", BITSTAMP_API_URL, BITSTAMP_API_VERSION, BITSTAMP_API_ORDERBOOK, common.StringToLower(currency))
	err := common.SendHTTPGetRequest(path, true, &resp)
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

func (b *Bitstamp) GetTransactions(currency string, values url.Values) ([]BitstampTransactions, error) {
	path := common.EncodeURLValues(fmt.Sprintf("%s/v%s/%s/%s/", BITSTAMP_API_URL, BITSTAMP_API_VERSION, BITSTAMP_API_TRANSACTIONS, common.StringToLower(currency)), values)
	transactions := []BitstampTransactions{}
	err := common.SendHTTPGetRequest(path, true, &transactions)
	if err != nil {
		return nil, err
	}
	return transactions, nil
}

func (b *Bitstamp) GetEURUSDConversionRate() (BitstampEURUSDConversionRate, error) {
	rate := BitstampEURUSDConversionRate{}
	path := fmt.Sprintf("%s/%s", BITSTAMP_API_URL, BITSTAMP_API_EURUSD)
	err := common.SendHTTPGetRequest(path, true, &rate)

	if err != nil {
		return rate, err
	}
	return rate, nil
}

func (b *Bitstamp) GetBalance() (BitstampBalances, error) {
	balance := BitstampBalances{}
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_BALANCE, true, url.Values{}, &balance)

	if err != nil {
		return balance, err
	}
	return balance, nil
}

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the Bitstamp exchange
func (e *Bitstamp) GetExchangeAccountInfo() (ExchangeAccountInfo, error) {
	var response ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	accountBalance, err := e.GetBalance()
	if err != nil {
		return response, err
	}

	var btcExchangeInfo ExchangeAccountCurrencyInfo
	btcExchangeInfo.CurrencyName = "BTC"
	btcExchangeInfo.TotalValue = accountBalance.BTCBalance
	btcExchangeInfo.Hold = accountBalance.BTCReserved
	response.Currencies = append(response.Currencies, btcExchangeInfo)

	var usdExchangeInfo ExchangeAccountCurrencyInfo
	usdExchangeInfo.CurrencyName = "USD"
	usdExchangeInfo.TotalValue = accountBalance.USDBalance
	usdExchangeInfo.Hold = accountBalance.USDReserved
	response.Currencies = append(response.Currencies, usdExchangeInfo)

	return response, nil
}

func (b *Bitstamp) GetUserTransactions(values url.Values) ([]BitstampUserTransactions, error) {
	type Response struct {
		Date    string      `json:"datetime"`
		TransID int64       `json:"id"`
		Type    int         `json:"type,string"`
		USD     interface{} `json:"usd"`
		EUR     float64     `json:"eur"`
		XRP     float64     `json:"xrp"`
		BTC     interface{} `json:"btc"`
		BTCUSD  interface{} `json:"btc_usd"`
		Fee     float64     `json:"fee,string"`
		OrderID int64       `json:"order_id"`
	}

	response := []Response{}
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_USER_TRANSACTIONS, true, values, &response)

	if err != nil {
		return nil, err
	}

	transactions := []BitstampUserTransactions{}

	for _, y := range response {
		tx := BitstampUserTransactions{}
		tx.Date = y.Date
		tx.TransID = y.TransID
		tx.Type = y.Type

		/* Hack due to inconsistent JSON values... */
		varType := reflect.TypeOf(y.USD).String()
		if varType == "string" {
			tx.USD, _ = strconv.ParseFloat(y.USD.(string), 64)
		} else {
			tx.USD = y.USD.(float64)
		}

		tx.EUR = y.EUR
		tx.XRP = y.XRP

		varType = reflect.TypeOf(y.BTC).String()
		if varType == "string" {
			tx.BTC, _ = strconv.ParseFloat(y.BTC.(string), 64)
		} else {
			tx.BTC = y.BTC.(float64)
		}

		varType = reflect.TypeOf(y.BTCUSD).String()
		if varType == "string" {
			tx.BTCUSD, _ = strconv.ParseFloat(y.BTCUSD.(string), 64)
		} else {
			tx.BTCUSD = y.BTCUSD.(float64)
		}

		tx.Fee = y.Fee
		tx.OrderID = y.OrderID
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

func (b *Bitstamp) GetOpenOrders(currency string) ([]BitstampOrder, error) {
	resp := []BitstampOrder{}
	path := fmt.Sprintf("%s/%s", BITSTAMP_API_OPEN_ORDERS, common.StringToLower(currency))
	err := b.SendAuthenticatedHTTPRequest(path, true, nil, &resp)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (b *Bitstamp) GetOrderStatus(OrderID int64) (BitstampOrderStatus, error) {
	var req = url.Values{}
	req.Add("id", strconv.FormatInt(OrderID, 10))
	resp := BitstampOrderStatus{}

	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_CANCEL_ORDER, false, req, &resp)

	if err != nil {
		return resp, err
	}

	return resp, nil
}

func (b *Bitstamp) CancelOrder(OrderID int64) (bool, error) {
	var req = url.Values{}
	result := false
	req.Add("id", strconv.FormatInt(OrderID, 10))

	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_CANCEL_ORDER, true, req, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (b *Bitstamp) CancelAllOrders() (bool, error) {
	result := false
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_CANCEL_ALL_ORDERS, false, nil, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (b *Bitstamp) PlaceOrder(currency string, price float64, amount float64, buy, market bool) (BitstampOrder, error) {
	var req = url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("price", strconv.FormatFloat(price, 'f', -1, 64))
	response := BitstampOrder{}
	orderType := BITSTAMP_API_BUY
	path := ""

	if !buy {
		orderType = BITSTAMP_API_SELL
	}

	path = fmt.Sprintf("%s/%s", orderType, common.StringToLower(currency))

	if market {
		path = fmt.Sprintf("%s/%s/%s", orderType, BITSTAMP_API_MARKET, common.StringToLower(currency))
	}

	err := b.SendAuthenticatedHTTPRequest(path, true, req, &response)

	if err != nil {
		return response, err
	}

	return response, nil
}

func (b *Bitstamp) GetWithdrawalRequests(values url.Values) ([]BitstampWithdrawalRequests, error) {
	resp := []BitstampWithdrawalRequests{}
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_WITHDRAWAL_REQUESTS, false, values, &resp)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (b *Bitstamp) BitcoinWithdrawal(amount float64, address string, instant bool) (string, error) {
	var req = url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("address", address)

	if instant {
		req.Add("instant", "1")
	} else {
		req.Add("instant", "0")
	}

	type response struct {
		ID string `json:"id"`
	}

	resp := response{}
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_BITCOIN_WITHDRAWAL, false, req, &resp)

	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (b *Bitstamp) GetBitcoinDepositAddress() (string, error) {
	address := ""
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_BITCOIN_DEPOSIT, false, url.Values{}, &address)

	if err != nil {
		return address, err
	}
	return address, nil
}

func (b *Bitstamp) GetUnconfirmedBitcoinDeposits() ([]BitstampUnconfirmedBTCTransactions, error) {
	response := []BitstampUnconfirmedBTCTransactions{}
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_UNCONFIRMED_BITCOIN, false, nil, &response)

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

	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_RIPPLE_WITHDRAWAL, false, req, nil)

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
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_RIPPLE_DESPOIT, false, nil, &resp)

	if err != nil {
		return "", err
	}

	return resp.Address, nil
}

func (b *Bitstamp) TransferAccountBalance(amount float64, currency, subAccount string, toMain bool) (bool, error) {
	var req = url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("currency", currency)
	req.Add("subAccount", subAccount)

	path := BITSTAMP_API_TRANSFER_TO_MAIN
	if !toMain {
		path = BITSTAMP_API_TRANSFER_FROM_MAIN
	}

	err := b.SendAuthenticatedHTTPRequest(path, true, req, nil)

	if err != nil {
		return false, err
	}

	return true, nil
}

func (b *Bitstamp) XRPWithdrawal(amount float64, address, destTag string) (string, error) {
	var req = url.Values{}
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("address", address)
	if destTag != "" {
		req.Add("destination_tag", destTag)
	}

	type response struct {
		ID string `json:"id"`
	}

	resp := response{}
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_XRP_WITHDRAWAL, true, req, &resp)

	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (b *Bitstamp) GetXRPDepositAddress() (BitstampXRPDepositResponse, error) {
	resp := BitstampXRPDepositResponse{}
	err := b.SendAuthenticatedHTTPRequest(BITSTAMP_API_XRP_DESPOIT, true, nil, &resp)

	if err != nil {
		return BitstampXRPDepositResponse{}, err
	}

	return resp, nil
}

func (b *Bitstamp) SendAuthenticatedHTTPRequest(path string, v2 bool, values url.Values, result interface{}) (err error) {
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)

	if values == nil {
		values = url.Values{}
	}

	values.Set("key", b.APIKey)
	values.Set("nonce", nonce)
	hmac := common.GetHMAC(common.HASH_SHA256, []byte(nonce+b.ClientID+b.APIKey), []byte(b.APISecret))
	values.Set("signature", common.StringToUpper(common.HexEncodeToString(hmac)))

	if v2 {
		path = fmt.Sprintf("%s/v%s/%s/", BITSTAMP_API_URL, BITSTAMP_API_VERSION, path)
	} else {
		path = fmt.Sprintf("%s/%s/", BITSTAMP_API_URL, path)
	}

	if b.Verbose {
		log.Println("Sending POST request to " + path)
	}

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := common.SendHTTPRequest("POST", path, headers, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}

	if b.Verbose {
		log.Printf("Recieved raw: %s\n", resp)
	}

	err = common.JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("Unable to JSON Unmarshal response.")
	}

	return nil
}
