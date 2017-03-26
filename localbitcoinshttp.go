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
	LOCALBITCOINS_API_URL             = "https://localbitcoins.com"
	LOCALBITCOINS_API_TICKER          = "/bitcoinaverage/ticker-all-currencies/"
	LOCALBITCOINS_API_BITCOINCHARTS   = "/bitcoincharts/"
	LOCALBITCOINS_API_PINCODE         = "pincode/"
	LOCALBITCOINS_API_WALLET          = "wallet/"
	LOCALBITCOINS_API_MYSELF          = "myself/"
	LOCALBITCOINS_API_WALLET_BALANCE  = "wallet-balance/"
	LOCALBITCOINS_API_WALLET_SEND     = "wallet-send/"
	LOCALBITCOINS_API_WALLET_SEND_PIN = "wallet-send-pin/"
	LOCALBITCOINS_API_WALLET_ADDRESS  = "wallet-addr/"
)

type LocalBitcoins struct {
	Name                        string
	Enabled                     bool
	Verbose                     bool
	Websocket                   bool
	RESTPollingDelay            time.Duration
	AuthenticatedAPISupport     bool
	Password, APIKey, APISecret string
	TakerFee, MakerFee          float64
	BaseCurrencies              []string
	AvailablePairs              []string
	EnabledPairs                []string
}

func (l *LocalBitcoins) SetDefaults() {
	l.Name = "LocalBitcoins"
	l.Enabled = false
	l.Verbose = false
	l.Verbose = false
	l.Websocket = false
	l.RESTPollingDelay = 10
}

func (l *LocalBitcoins) GetName() string {
	return l.Name
}

func (l *LocalBitcoins) SetEnabled(enabled bool) {
	l.Enabled = enabled
}

func (l *LocalBitcoins) IsEnabled() bool {
	return l.Enabled
}

func (l *LocalBitcoins) Setup(exch Exchanges) {
	if !exch.Enabled {
		l.SetEnabled(false)
	} else {
		l.Enabled = true
		l.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		l.SetAPIKeys(exch.APIKey, exch.APISecret)
		l.RESTPollingDelay = exch.RESTPollingDelay
		l.Verbose = exch.Verbose
		l.Websocket = exch.Websocket
		l.BaseCurrencies = SplitStrings(exch.BaseCurrencies, ",")
		l.AvailablePairs = SplitStrings(exch.AvailablePairs, ",")
		l.EnabledPairs = SplitStrings(exch.EnabledPairs, ",")
	}
}

func (k *LocalBitcoins) GetEnabledCurrencies() []string {
	return k.EnabledPairs
}

func (l *LocalBitcoins) Start() {
	go l.Run()
}

func (l *LocalBitcoins) GetFee(maker bool) float64 {
	if maker {
		return l.MakerFee
	} else {
		return l.TakerFee
	}
}

func (l *LocalBitcoins) Run() {
	if l.Verbose {
		log.Printf("%s polling delay: %ds.\n", l.GetName(), l.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", l.GetName(), len(l.EnabledPairs), l.EnabledPairs)
	}

	for l.Enabled {
		for _, x := range l.EnabledPairs {
			currency := x[3:]
			ticker, err := l.GetTickerPrice("BTC" + currency)

			if err != nil {
				log.Println(err)
				return
			}

			log.Printf("LocalBitcoins BTC %s: Last %f Volume %f\n", currency, ticker.Last, ticker.Volume)
			AddExchangeInfo(l.GetName(), x[0:3], x[3:], ticker.Last, ticker.Volume)
		}
		time.Sleep(time.Second * l.RESTPollingDelay)
	}
}

func (l *LocalBitcoins) SetAPIKeys(apiKey, apiSecret string) {
	l.APIKey = apiKey
	l.APISecret = apiSecret
}

type LocalBitcoinsTicker struct {
	Avg12h float64 `json:"avg_12h,string"`
	Avg1h  float64 `json:"avg_1h,string"`
	Avg24h float64 `json:"avg_24h,string"`
	Rates  struct {
		Last float64 `json:"last,string"`
	} `json:"rates"`
	VolumeBTC float64 `json:"volume_btc,string"`
}

func (l *LocalBitcoins) GetTicker() (map[string]LocalBitcoinsTicker, error) {
	result := make(map[string]LocalBitcoinsTicker)
	err := SendHTTPGetRequest(LOCALBITCOINS_API_URL+LOCALBITCOINS_API_TICKER, true, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

func (l *LocalBitcoins) GetTickerPrice(currency string) (TickerPrice, error) {
	tickerNew, err := GetTicker(l.GetName(), currency[0:3], currency[3:])
	if err == nil {
		return tickerNew, nil
	}

	ticker, err := l.GetTicker()
	if err != nil {
		return TickerPrice{}, err
	}

	var tickerPrice TickerPrice
	for key, value := range ticker {
		tickerPrice.Last = value.Rates.Last
		tickerPrice.FirstCurrency = currency[0:3]
		tickerPrice.SecondCurrency = key
		tickerPrice.CurrencyPair = tickerPrice.FirstCurrency + "_" + tickerPrice.SecondCurrency
		tickerPrice.Volume = value.VolumeBTC
		ProcessTicker(l.GetName(), tickerPrice.FirstCurrency, tickerPrice.SecondCurrency, tickerPrice)
	}
	return tickerPrice, nil
}

type LocalBitcoinsTrade struct {
	TID    int64   `json:"tid"`
	Date   int64   `json:"date"`
	Amount float64 `json:"amount,string"`
	Price  float64 `json:"price,string"`
}

func (l *LocalBitcoins) GetTrades(currency string, values url.Values) ([]LocalBitcoinsTrade, error) {
	path := EncodeURLValues(fmt.Sprintf("%s/%s/trades.json", LOCALBITCOINS_API_URL+LOCALBITCOINS_API_BITCOINCHARTS, currency), values)
	result := []LocalBitcoinsTrade{}
	err := SendHTTPGetRequest(path, true, &result)

	if err != nil {
		return result, err
	}

	return result, nil
}

type LocalBitcoinsOrderbookStructure struct {
	Price  float64
	Amount float64
}

type LocalBitcoinsOrderbook struct {
	Bids []LocalBitcoinsOrderbookStructure `json:"bids"`
	Asks []LocalBitcoinsOrderbookStructure `json:"asks"`
}

func (l *LocalBitcoins) GetOrderbook(currency string) (LocalBitcoinsOrderbook, error) {
	type response struct {
		Bids [][]string `json:"bids"`
		Asks [][]string `json:"asks"`
	}

	path := fmt.Sprintf("%s/%s/orderbook.json", LOCALBITCOINS_API_URL+LOCALBITCOINS_API_BITCOINCHARTS, currency)
	resp := response{}
	err := SendHTTPGetRequest(path, true, &resp)

	if err != nil {
		return LocalBitcoinsOrderbook{}, err
	}

	orderbook := LocalBitcoinsOrderbook{}

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
		orderbook.Bids = append(orderbook.Bids, LocalBitcoinsOrderbookStructure{price, amount})
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
		orderbook.Asks = append(orderbook.Asks, LocalBitcoinsOrderbookStructure{price, amount})
	}

	return orderbook, nil
}

type LocalBitcoinsAccountInfo struct {
	Username             string    `json:"username"`
	CreatedAt            time.Time `json:"created_at"`
	AgeText              string    `json:"age_text"`
	TradingPartners      int       `json:"trading_partners_count"`
	FeedbacksUnconfirmed int       `json:"feedbacks_unconfirmed_count"`
	TradeVolumeText      string    `json:"trade_volume_text"`
	HasCommonTrades      bool      `json:"has_common_trades"`
	HasFeedback          bool      `json:"has_feedback"`
	ConfirmedTradesText  string    `json:"confirmed_trade_count_text"`
	BlockedCount         int       `json:"blocked_count"`
	FeedbackScore        int       `json:"feedback_score"`
	FeedbackCount        int       `json:"feedback_count"`
	URL                  string    `json:"url"`
	TrustedCount         int       `json:"trusted_count"`
	IdentityVerifiedAt   time.Time `json:"identify_verified_at"`
}

func (l *LocalBitcoins) GetAccountInfo(username string, self bool) (LocalBitcoinsAccountInfo, error) {
	type response struct {
		Data LocalBitcoinsAccountInfo `json:"data"`
	}
	resp := response{}

	if self {
		err := l.SendAuthenticatedHTTPRequest("GET", LOCALBITCOINS_API_MYSELF, nil, &resp)

		if err != nil {
			return resp.Data, err
		}
	} else {
		path := fmt.Sprintf("%s/api/account_info/%s/", LOCALBITCOINS_API_URL, username)
		err := SendHTTPGetRequest(path, true, &resp)

		if err != nil {
			return resp.Data, err
		}
	}

	return resp.Data, nil
}

func (l *LocalBitcoins) CheckPincode(pin int) (bool, error) {
	type response struct {
		Data struct {
			PinOK bool `json:"pincode_ok"`
		} `json:"data"`
	}
	resp := response{}
	values := url.Values{}
	values.Set("pincode", strconv.Itoa(pin))
	err := l.SendAuthenticatedHTTPRequest("POST", LOCALBITCOINS_API_PINCODE, values, &resp)

	if err != nil {
		return false, err
	}

	if !resp.Data.PinOK {
		return false, errors.New("Pin invalid.")
	}

	return true, nil
}

type LocalBitcoinsBalance struct {
	Balance  float64 `json:"balance,string"`
	Sendable float64 `json:"Sendable,string"`
}

type LocalBitcoinsWalletTransaction struct {
	TXID        string    `json:"txid"`
	Amount      float64   `json:"amount,string"`
	Description string    `json:"description"`
	TXType      int       `json:"tx_type"`
	CreatedAt   time.Time `json:"created_at"`
}

type LocalBitcoinsWalletAddressList struct {
	Address  string  `json:"address"`
	Received float64 `json:"received,string"`
}

type LocalBitcoinsWalletInfo struct {
	Message                 string                           `json:"message"`
	Total                   LocalBitcoinsBalance             `json:"total"`
	SentTransactions30d     []LocalBitcoinsWalletTransaction `json:"sent_transactions_30d"`
	ReceivedTransactions30d []LocalBitcoinsWalletTransaction `json:"received_transactions_30d"`
	ReceivingAddressCount   int                              `json:"receiving_address_count"`
	ReceivingAddressList    []LocalBitcoinsWalletAddressList `json:"receiving_address_list"`
}

func (l *LocalBitcoins) GetWalletInfo() (LocalBitcoinsWalletInfo, error) {
	type response struct {
		Data LocalBitcoinsWalletInfo `json:"data"`
	}
	resp := response{}
	err := l.SendAuthenticatedHTTPRequest("GET", LOCALBITCOINS_API_WALLET, nil, &resp)

	if err != nil {
		return LocalBitcoinsWalletInfo{}, err
	}

	if resp.Data.Message != "OK" {
		return LocalBitcoinsWalletInfo{}, errors.New("Unable to fetch wallet info.")
	}

	return resp.Data, nil
}

type LocalBitcoinsWalletBalanceInfo struct {
	Message               string                           `json:"message"`
	Total                 LocalBitcoinsBalance             `json:"total"`
	ReceivingAddressCount int                              `json:"receiving_address_count"` // always 1
	ReceivingAddressList  []LocalBitcoinsWalletAddressList `json:"receiving_address_list"`
}

func (l *LocalBitcoins) GetWalletBalance() (LocalBitcoinsWalletBalanceInfo, error) {
	type response struct {
		Data LocalBitcoinsWalletBalanceInfo `json:"data"`
	}
	resp := response{}
	err := l.SendAuthenticatedHTTPRequest("GET", LOCALBITCOINS_API_WALLET_BALANCE, nil, &resp)

	if err != nil {
		return LocalBitcoinsWalletBalanceInfo{}, err
	}

	if resp.Data.Message != "OK" {
		return LocalBitcoinsWalletBalanceInfo{}, errors.New("Unable to fetch wallet balance.")
	}

	return resp.Data, nil
}

func (l *LocalBitcoins) WalletSend(address string, amount float64, pin int) (bool, error) {
	values := url.Values{}
	values.Set("address", address)
	values.Set("amount", strconv.FormatFloat(amount, 'f', -1, 64))
	path := LOCALBITCOINS_API_WALLET_SEND

	if pin > 0 {
		values.Set("pincode", strconv.Itoa(pin))
		path = LOCALBITCOINS_API_WALLET_SEND_PIN
	}

	type response struct {
		Data struct {
			Message string `json:"message"`
		} `json:"data"`
	}

	resp := response{}
	err := l.SendAuthenticatedHTTPRequest("POST", path, values, &resp)
	if err != nil {
		return false, err
	}

	if resp.Data.Message != "Money is being sent" {
		return false, errors.New("Unable to send Bitcoins.")
	}

	return true, nil
}

func (l *LocalBitcoins) GetWalletAddress() (string, error) {
	type response struct {
		Data struct {
			Message string `json:"message"`
			Address string `json:"address"`
		}
	}
	resp := response{}
	err := l.SendAuthenticatedHTTPRequest("POST", LOCALBITCOINS_API_WALLET_ADDRESS, nil, &resp)
	if err != nil {
		return "", err
	}

	if resp.Data.Message != "OK!" {
		return "", errors.New("Unable to fetch wallet address.")
	}

	return resp.Data.Address, nil
}

func (l *LocalBitcoins) SendAuthenticatedHTTPRequest(method, path string, values url.Values, result interface{}) (err error) {
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)
	payload := ""
	path = "/api/" + path

	if len(values) > 0 {
		payload = values.Encode()
	}

	message := string(nonce) + l.APIKey + path + payload
	hmac := GetHMAC(HASH_SHA256, []byte(message), []byte(l.APISecret))
	headers := make(map[string]string)
	headers["Apiauth-Key"] = l.APIKey
	headers["Apiauth-Nonce"] = string(nonce)
	headers["Apiauth-Signature"] = StringToUpper(HexEncodeToString(hmac))
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := SendHTTPRequest(method, LOCALBITCOINS_API_URL+path, headers, bytes.NewBuffer([]byte(payload)))

	if l.Verbose {
		log.Printf("Recieved raw: \n%s\n", resp)
	}

	err = JSONDecode([]byte(resp), &result)

	if err != nil {
		return errors.New("Unable to JSON Unmarshal response.")
	}

	return nil
}

//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the LocalBitcoins exchange
func (e *LocalBitcoins) GetExchangeAccountInfo() (ExchangeAccountInfo, error) {
	var response ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	accountBalance, err := e.GetWalletBalance()
	if err != nil {
		return response, err
	}
	var exchangeCurrency ExchangeAccountCurrencyInfo
	exchangeCurrency.CurrencyName = "BTC"
	exchangeCurrency.TotalValue = accountBalance.Total.Balance

	response.Currencies = append(response.Currencies, exchangeCurrency)
	return response, nil
}
