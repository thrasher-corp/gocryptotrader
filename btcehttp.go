package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	BTCE_API_PUBLIC_URL      = "https://btc-e.com/api"
	BTCE_API_PRIVATE_URL     = "https://btc-e.com/tapi"
	BTCE_API_PUBLIC_VERSION  = "3"
	BTCE_API_PRIVATE_VERSION = "1"
	BTCE_INFO                = "info"
	BTCE_TICKER              = "ticker"
	BTCE_DEPTH               = "depth"
	BTCE_TRADES              = "trades"
	BTCE_ACCOUNT_INFO        = "getInfo"
	BTCE_TRADE               = "Trade"
	BTCE_ACTIVE_ORDERS       = "ActiveOrders"
	BTCE_ORDER_INFO          = "OrderInfo"
	BTCE_CANCEL_ORDER        = "CancelOrder"
	BTCE_TRADE_HISTORY       = "TradeHistory"
	BTCE_TRANSACTION_HISTORY = "TransHistory"
	BTCE_WITHDRAW_COIN       = "WithdrawCoin"
	BTCE_CREATE_COUPON       = "CreateCoupon"
	BTCE_REDEEM_COUPON       = "RedeemCoupon"
)

type BTCE struct {
	Name                    string
	Enabled                 bool
	Verbose                 bool
	Websocket               bool
	RESTPollingDelay        time.Duration
	AuthenticatedAPISupport bool
	APIKey, APISecret       string
	Fee                     float64
	BaseCurrencies          []string
	AvailablePairs          []string
	EnabledPairs            []string
	Ticker                  map[string]BTCeTicker
}

type BTCeTicker struct {
	High    float64
	Low     float64
	Avg     float64
	Vol     float64
	Vol_cur float64
	Last    float64
	Buy     float64
	Sell    float64
	Updated int64
}

type BTCEOrderbook struct {
	Asks [][]float64 `json:"asks"`
	Bids [][]float64 `json:"bids"`
}

type BTCETrades struct {
	Type      string  `json:"type"`
	Price     float64 `json:"bid"`
	Amount    float64 `json:"amount"`
	TID       int64   `json:"tid"`
	Timestamp int64   `json:"timestamp"`
}

type BTCEResponse struct {
	Return  interface{} `json:"return"`
	Success int         `json:"success"`
	Error   string      `json:"error"`
}

func (b *BTCE) SetDefaults() {
	b.Name = "BTCE"
	b.Enabled = false
	b.Fee = 0.2
	b.Verbose = false
	b.Websocket = false
	b.RESTPollingDelay = 10
	b.Ticker = make(map[string]BTCeTicker)
}

func (b *BTCE) GetName() string {
	return b.Name
}

func (b *BTCE) SetEnabled(enabled bool) {
	b.Enabled = enabled
}

func (b *BTCE) IsEnabled() bool {
	return b.Enabled
}

func (b *BTCE) Setup(exch Exchanges) {
	if !exch.Enabled {
		b.SetEnabled(false)
	} else {
		b.Enabled = true
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

func (k *BTCE) GetEnabledCurrencies() []string {
	return k.EnabledPairs
}

func (b *BTCE) Start() {
	go b.Run()
}

func (b *BTCE) SetAPIKeys(apiKey, apiSecret string) {
	b.APIKey = apiKey
	b.APISecret = apiSecret
}

func (b *BTCE) GetFee() float64 {
	return b.Fee
}

func (b *BTCE) Run() {
	if b.Verbose {
		log.Printf("%s Websocket: %s.", b.GetName(), IsEnabled(b.Websocket))
		log.Printf("%s polling delay: %ds.\n", b.GetName(), b.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", b.GetName(), len(b.EnabledPairs), b.EnabledPairs)
	}

	pairs := []string{}
	for _, x := range b.EnabledPairs {
		x = StringToLower(x[0:3] + "_" + x[3:6])
		pairs = append(pairs, x)
	}
	pairsString := JoinStrings(pairs, "-")

	for b.Enabled {
		go func() {
			ticker, err := b.GetTicker(pairsString)
			if err != nil {
				log.Println(err)
				return
			}
			for x, y := range ticker {
				x = StringToUpper(x[0:3] + x[4:])
				log.Printf("BTC-e %s: Last %f High %f Low %f Volume %f\n", x, y.Last, y.High, y.Low, y.Vol_cur)
				b.Ticker[x] = y
				AddExchangeInfo(b.GetName(), StringToUpper(x[0:3]), StringToUpper(x[4:]), y.Last, y.Vol_cur)
			}
		}()
		time.Sleep(time.Second * b.RESTPollingDelay)
	}
}

func (b *BTCE) GetInfo() {
	req := fmt.Sprintf("%s/%s/%s/", BTCE_API_PUBLIC_URL, BTCE_API_PUBLIC_VERSION, BTCE_INFO)
	err := SendHTTPGetRequest(req, true, nil)

	if err != nil {
		log.Println(err)
	}
}

func (b *BTCE) GetTicker(symbol string) (map[string]BTCeTicker, error) {
	type Response struct {
		Data map[string]BTCeTicker
	}

	response := Response{}
	req := fmt.Sprintf("%s/%s/%s/%s", BTCE_API_PUBLIC_URL, BTCE_API_PUBLIC_VERSION, BTCE_TICKER, symbol)
	err := SendHTTPGetRequest(req, true, &response.Data)

	if err != nil {
		return nil, err
	}
	return response.Data, nil
}

func (b *BTCE) GetTickerPrice(currency string) (TickerPrice, error) {
	var tickerPrice TickerPrice
	ticker, ok := b.Ticker[currency]
	if !ok {
		return tickerPrice, errors.New("Unable to get currency.")
	}
	tickerPrice.Ask = ticker.Buy
	tickerPrice.Bid = ticker.Sell
	tickerPrice.FirstCurrency = currency[0:3]
	tickerPrice.SecondCurrency = currency[3:]
	tickerPrice.Low = ticker.Low
	tickerPrice.Last = ticker.Last
	tickerPrice.Volume = ticker.Vol_cur
	tickerPrice.High = ticker.High
	ProcessTicker(b.GetName(), tickerPrice.FirstCurrency, tickerPrice.SecondCurrency, tickerPrice)
	return tickerPrice, nil
}

func (b *BTCE) GetDepth(symbol string) (BTCEOrderbook, error) {
	type Response struct {
		Data map[string]BTCEOrderbook
	}

	response := Response{}
	req := fmt.Sprintf("%s/%s/%s/%s", BTCE_API_PUBLIC_URL, BTCE_API_PUBLIC_VERSION, BTCE_DEPTH, symbol)

	err := SendHTTPGetRequest(req, true, &response.Data)
	if err != nil {
		return BTCEOrderbook{}, err
	}

	depth := response.Data[symbol]
	return depth, nil
}

func (b *BTCE) GetTrades(symbol string) ([]BTCETrades, error) {
	type Response struct {
		Data map[string][]BTCETrades
	}

	response := Response{}
	req := fmt.Sprintf("%s/%s/%s/%s", BTCE_API_PUBLIC_URL, BTCE_API_PUBLIC_VERSION, BTCE_TRADES, symbol)

	err := SendHTTPGetRequest(req, true, &response.Data)
	if err != nil {
		return []BTCETrades{}, err
	}

	trades := response.Data[symbol]
	return trades, nil
}

type BTCEAccountInfo struct {
	Funds      map[string]float64 `json:"funds"`
	OpenOrders int                `json:"open_orders"`
	Rights     struct {
		Info     int `json:"info"`
		Trade    int `json:"trade"`
		Withdraw int `json:"withdraw"`
	} `json:"rights"`
	ServerTime       float64 `json:"server_time"`
	TransactionCount int     `json:"transaction_count"`
}

func (b *BTCE) GetAccountInfo() (BTCEAccountInfo, error) {
	var result BTCEAccountInfo
	err := b.SendAuthenticatedHTTPRequest(BTCE_ACCOUNT_INFO, url.Values{}, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the BTCE exchange
func (e *BTCE) GetExchangeAccountInfo() (ExchangeAccountInfo, error) {
	var response ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	accountBalance, err := e.GetAccountInfo()
	if err != nil {
		return response, err
	}

	for x, y := range accountBalance.Funds {
		var exchangeCurrency ExchangeAccountCurrencyInfo
		exchangeCurrency.CurrencyName = StringToUpper(x)
		exchangeCurrency.TotalValue = y
		exchangeCurrency.Hold = 0
		response.Currencies = append(response.Currencies, exchangeCurrency)
	}

	return response, nil
}

type BTCEActiveOrders struct {
	Pair             string  `json:"pair"`
	Type             string  `json:"sell"`
	Amount           float64 `json:"amount"`
	Rate             float64 `json:"rate"`
	TimestampCreated float64 `json:"time_created"`
	Status           int     `json:"status"`
}

func (b *BTCE) GetActiveOrders(pair string) (map[string]BTCEActiveOrders, error) {
	req := url.Values{}
	req.Add("pair", pair)

	var result map[string]BTCEActiveOrders
	err := b.SendAuthenticatedHTTPRequest(BTCE_ACTIVE_ORDERS, req, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

type BTCEOrderInfo struct {
	Pair             string  `json:"pair"`
	Type             string  `json:"sell"`
	StartAmount      float64 `json:"start_amount"`
	Amount           float64 `json:"amount"`
	Rate             float64 `json:"rate"`
	TimestampCreated float64 `json:"time_created"`
	Status           int     `json:"status"`
}

func (b *BTCE) GetOrderInfo(OrderID int64) (map[string]BTCEOrderInfo, error) {
	req := url.Values{}
	req.Add("order_id", strconv.FormatInt(OrderID, 10))

	var result map[string]BTCEOrderInfo
	err := b.SendAuthenticatedHTTPRequest(BTCE_ORDER_INFO, req, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

type BTCECancelOrder struct {
	OrderID float64            `json:"order_id"`
	Funds   map[string]float64 `json:"funds"`
}

func (b *BTCE) CancelOrder(OrderID int64) (bool, error) {
	req := url.Values{}
	req.Add("order_id", strconv.FormatInt(OrderID, 10))

	var result BTCECancelOrder
	err := b.SendAuthenticatedHTTPRequest(BTCE_CANCEL_ORDER, req, &result)

	if err != nil {
		return false, err
	}

	return true, nil
}

type BTCETrade struct {
	Received float64            `json:"received"`
	Remains  float64            `json:"remains"`
	OrderID  float64            `json:"order_id"`
	Funds    map[string]float64 `json:"funds"`
}

//to-do: convert orderid to int64
func (b *BTCE) Trade(pair, orderType string, amount, price float64) (float64, error) {
	req := url.Values{}
	req.Add("pair", pair)
	req.Add("type", orderType)
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("rate", strconv.FormatFloat(price, 'f', -1, 64))

	var result BTCETrade
	err := b.SendAuthenticatedHTTPRequest(BTCE_TRADE, req, &result)

	if err != nil {
		return 0, err
	}

	return result.OrderID, nil
}

type BTCETransHistory struct {
	Type        int     `json:"type"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Description string  `json:"desc"`
	Status      int     `json:"status"`
	Timestamp   float64 `json:"timestamp"`
}

func (b *BTCE) GetTransactionHistory(TIDFrom, Count, TIDEnd int64, order, since, end string) (map[string]BTCETransHistory, error) {
	req := url.Values{}
	req.Add("from", strconv.FormatInt(TIDFrom, 10))
	req.Add("count", strconv.FormatInt(Count, 10))
	req.Add("from_id", strconv.FormatInt(TIDFrom, 10))
	req.Add("end_id", strconv.FormatInt(TIDEnd, 10))
	req.Add("order", order)
	req.Add("since", since)
	req.Add("end", end)

	var result map[string]BTCETransHistory
	err := b.SendAuthenticatedHTTPRequest(BTCE_TRANSACTION_HISTORY, req, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

type BTCETradeHistory struct {
	Pair      string  `json:"pair"`
	Type      string  `json:"type"`
	Amount    float64 `json:"amount"`
	Rate      float64 `json:"rate"`
	OrderID   float64 `json:"order_id"`
	MyOrder   int     `json:"is_your_order"`
	Timestamp float64 `json:"timestamp"`
}

func (b *BTCE) GetTradeHistory(TIDFrom, Count, TIDEnd int64, order, since, end, pair string) (map[string]BTCETradeHistory, error) {
	req := url.Values{}

	req.Add("from", strconv.FormatInt(TIDFrom, 10))
	req.Add("count", strconv.FormatInt(Count, 10))
	req.Add("from_id", strconv.FormatInt(TIDFrom, 10))
	req.Add("end_id", strconv.FormatInt(TIDEnd, 10))
	req.Add("order", order)
	req.Add("since", since)
	req.Add("end", end)
	req.Add("pair", pair)

	var result map[string]BTCETradeHistory
	err := b.SendAuthenticatedHTTPRequest(BTCE_TRADE_HISTORY, req, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

type BTCEWithdrawCoins struct {
	TID        int64              `json:"tId"`
	AmountSent float64            `json:"amountSent"`
	Funds      map[string]float64 `json:"funds"`
}

func (b *BTCE) WithdrawCoins(coin string, amount float64, address string) (BTCEWithdrawCoins, error) {
	req := url.Values{}

	req.Add("coinName", coin)
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	req.Add("address", address)

	var result BTCEWithdrawCoins
	err := b.SendAuthenticatedHTTPRequest(BTCE_WITHDRAW_COIN, req, &result)

	if err != nil {
		return result, err
	}
	return result, nil
}

type BTCECreateCoupon struct {
	Coupon  string             `json:"coupon"`
	TransID int64              `json:"transID"`
	Funds   map[string]float64 `json:"funds"`
}

func (b *BTCE) CreateCoupon(currency string, amount float64) (BTCECreateCoupon, error) {
	req := url.Values{}

	req.Add("currency", currency)
	req.Add("amount", strconv.FormatFloat(amount, 'f', -1, 64))

	var result BTCECreateCoupon
	err := b.SendAuthenticatedHTTPRequest(BTCE_CREATE_COUPON, req, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

type BTCERedeemCoupon struct {
	CouponAmount   float64 `json:"couponAmount,string"`
	CouponCurrency string  `json:"couponCurrency"`
	TransID        int64   `json:"transID"`
}

func (b *BTCE) RedeemCoupon(coupon string) (BTCERedeemCoupon, error) {
	req := url.Values{}

	req.Add("coupon", coupon)

	var result BTCERedeemCoupon
	err := b.SendAuthenticatedHTTPRequest(BTCE_REDEEM_COUPON, req, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (b *BTCE) SendAuthenticatedHTTPRequest(method string, values url.Values, result interface{}) (err error) {
	nonce := strconv.FormatInt(time.Now().Unix(), 10)
	values.Set("nonce", nonce)
	values.Set("method", method)

	encoded := values.Encode()
	hmac := GetHMAC(HASH_SHA512, []byte(encoded), []byte(b.APISecret))

	if b.Verbose {
		log.Printf("Sending POST request to %s calling method %s with params %s\n", BTCE_API_PRIVATE_URL, method, encoded)
	}

	headers := make(map[string]string)
	headers["Key"] = b.APIKey
	headers["Sign"] = HexEncodeToString(hmac)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := SendHTTPRequest("POST", BTCE_API_PRIVATE_URL, headers, strings.NewReader(encoded))

	if err != nil {
		return err
	}

	response := BTCEResponse{}
	err = JSONDecode([]byte(resp), &response)

	if err != nil {
		return err
	}

	if response.Success != 1 {
		return errors.New(response.Error)
	}

	jsonEncoded, err := JSONEncode(response.Return)

	if err != nil {
		return err
	}

	err = JSONDecode(jsonEncoded, &result)

	if err != nil {
		return err
	}
	return nil
}
