package main

import (
	"bytes"
	"errors"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
)

const (
	ITBIT_API_URL     = "https://api.itbit.com/v1"
	ITBIT_API_VERSION = "1"
)

type ItBit struct {
	Name                         string
	Enabled                      bool
	Verbose                      bool
	Websocket                    bool
	RESTPollingDelay             time.Duration
	AuthenticatedAPISupport      bool
	ClientKey, APISecret, UserID string
	MakerFee, TakerFee           float64
	BaseCurrencies               []string
	AvailablePairs               []string
	EnabledPairs                 []string
}

type ItBitTicker struct {
	Pair          string
	Bid           float64 `json:",string"`
	BidAmt        float64 `json:",string"`
	Ask           float64 `json:",string"`
	AskAmt        float64 `json:",string"`
	LastPrice     float64 `json:",string"`
	LastAmt       float64 `json:",string"`
	Volume24h     float64 `json:",string"`
	VolumeToday   float64 `json:",string"`
	High24h       float64 `json:",string"`
	Low24h        float64 `json:",string"`
	HighToday     float64 `json:",string"`
	LowToday      float64 `json:",string"`
	OpenToday     float64 `json:",string"`
	VwapToday     float64 `json:",string"`
	Vwap24h       float64 `json:",string"`
	ServertimeUTC string
}

func (i *ItBit) SetDefaults() {
	i.Name = "ITBIT"
	i.Enabled = false
	i.MakerFee = -0.10
	i.TakerFee = 0.50
	i.Verbose = false
	i.Websocket = false
	i.RESTPollingDelay = 10
}

func (i *ItBit) GetName() string {
	return i.Name
}

func (i *ItBit) SetEnabled(enabled bool) {
	i.Enabled = enabled
}

func (i *ItBit) IsEnabled() bool {
	return i.Enabled
}

func (i *ItBit) Setup(exch config.ExchangeConfig) {
	if !exch.Enabled {
		i.SetEnabled(false)
	} else {
		i.Enabled = true
		i.AuthenticatedAPISupport = exch.AuthenticatedAPISupport
		i.SetAPIKeys(exch.APIKey, exch.APISecret, exch.ClientID)
		i.RESTPollingDelay = exch.RESTPollingDelay
		i.Verbose = exch.Verbose
		i.Websocket = exch.Websocket
		i.BaseCurrencies = common.SplitStrings(exch.BaseCurrencies, ",")
		i.AvailablePairs = common.SplitStrings(exch.AvailablePairs, ",")
		i.EnabledPairs = common.SplitStrings(exch.EnabledPairs, ",")
	}
}

func (k *ItBit) GetEnabledCurrencies() []string {
	return k.EnabledPairs
}

func (i *ItBit) Start() {
	go i.Run()
}

func (i *ItBit) SetAPIKeys(apiKey, apiSecret, userID string) {
	i.ClientKey = apiKey
	i.APISecret = apiSecret
	i.UserID = userID
}

func (i *ItBit) GetFee(maker bool) float64 {
	if maker {
		return i.MakerFee
	} else {
		return i.TakerFee
	}
}

func (i *ItBit) Run() {
	if i.Verbose {
		log.Printf("%s polling delay: %ds.\n", i.GetName(), i.RESTPollingDelay)
		log.Printf("%s %d currencies enabled: %s.\n", i.GetName(), len(i.EnabledPairs), i.EnabledPairs)
	}

	for i.Enabled {
		for _, x := range i.EnabledPairs {
			currency := x
			go func() {
				ticker, err := i.GetTickerPrice(currency)
				if err != nil {
					log.Println(err)
					return
				}
				log.Printf("ItBit %s: Last %f High %f Low %f Volume %f\n", currency, ticker.Last, ticker.High, ticker.Low, ticker.Volume)
				AddExchangeInfo(i.GetName(), currency[0:3], currency[3:], ticker.Last, ticker.Volume)
			}()
		}
		time.Sleep(time.Second * i.RESTPollingDelay)
	}
}

func (i *ItBit) GetTicker(currency string) (ItBitTicker, error) {
	path := ITBIT_API_URL + "/markets/" + currency + "/ticker"
	var itbitTicker ItBitTicker
	err := common.SendHTTPGetRequest(path, true, &itbitTicker)
	if err != nil {
		return ItBitTicker{}, err
	}
	return itbitTicker, nil
}

func (i *ItBit) GetTickerPrice(currency string) (TickerPrice, error) {
	tickerNew, err := GetTicker(i.GetName(), currency[0:3], currency[3:])
	if err == nil {
		return tickerNew, nil
	}

	var tickerPrice TickerPrice
	ticker, err := i.GetTicker(currency)
	if err != nil {
		return tickerPrice, err
	}

	tickerPrice.Ask = ticker.Ask
	tickerPrice.Bid = ticker.Bid
	tickerPrice.FirstCurrency = currency[0:3]
	tickerPrice.SecondCurrency = currency[3:]
	tickerPrice.CurrencyPair = tickerPrice.FirstCurrency + "_" + tickerPrice.SecondCurrency
	tickerPrice.Last = ticker.LastPrice
	tickerPrice.High = ticker.High24h
	tickerPrice.Low = ticker.Low24h
	tickerPrice.Volume = ticker.Volume24h
	ProcessTicker(i.GetName(), tickerPrice.FirstCurrency, tickerPrice.SecondCurrency, tickerPrice)
	return tickerPrice, nil
}

type ItbitOrderbookEntry struct {
	Quantitiy float64 `json:"quantity,string"`
	Price     float64 `json:"price,string"`
}

type ItBitOrderbookResponse struct {
	ServerTimeUTC      string                `json:"serverTimeUTC"`
	LastUpdatedTimeUTC string                `json:"lastUpdatedTimeUTC"`
	Ticker             string                `json:"ticker"`
	Bids               []ItbitOrderbookEntry `json:"bids"`
	Asks               []ItbitOrderbookEntry `json:"asks"`
}

func (i *ItBit) GetOrderbook(currency string) (ItBitOrderbookResponse, error) {
	response := ItBitOrderbookResponse{}
	path := ITBIT_API_URL + "/markets/" + currency + "/order_book"
	err := common.SendHTTPGetRequest(path, true, &response)
	if err != nil {
		return ItBitOrderbookResponse{}, err
	}
	return response, nil
}

func (i *ItBit) GetTradeHistory(currency, timestamp string) bool {
	req := "/trades?since=" + timestamp
	err := common.SendHTTPGetRequest(ITBIT_API_URL+"markets/"+currency+req, true, nil)
	if err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (i *ItBit) GetWallets(params url.Values) {
	params.Set("userId", i.UserID)
	path := "/wallets?" + params.Encode()

	err := i.SendAuthenticatedHTTPRequest("GET", path, nil)

	if err != nil {
		log.Println(err)
	}
}

//TODO Get current holdings from ItBit
//GetExchangeAccountInfo : Retrieves balances for all enabled currencies for the ItBit exchange
func (e *ItBit) GetExchangeAccountInfo() (ExchangeAccountInfo, error) {
	var response ExchangeAccountInfo
	response.ExchangeName = e.GetName()
	return response, nil
}

func (i *ItBit) CreateWallet(walletName string) {
	path := "/wallets"
	params := make(map[string]interface{})
	params["userId"] = i.UserID
	params["name"] = walletName

	err := i.SendAuthenticatedHTTPRequest("POST", path, params)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) GetWallet(walletID string) {
	path := "/wallets/" + walletID
	err := i.SendAuthenticatedHTTPRequest("GET", path, nil)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) GetWalletBalance(walletID, currency string) {
	path := "/wallets/ " + walletID + "/balances/" + currency
	err := i.SendAuthenticatedHTTPRequest("GET", path, nil)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) GetWalletTrades(walletID string, params url.Values) {
	path := common.EncodeURLValues("/wallets/"+walletID+"/trades", params)
	err := i.SendAuthenticatedHTTPRequest("GET", path, nil)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) GetWalletOrders(walletID string, params url.Values) {
	path := common.EncodeURLValues("/wallets/"+walletID+"/orders", params)
	err := i.SendAuthenticatedHTTPRequest("GET", path, nil)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) PlaceWalletOrder(walletID, side, orderType, currency string, amount, price float64, instrument string, clientRef string) {
	path := "/wallets/" + walletID + "/orders"
	params := make(map[string]interface{})
	params["side"] = side
	params["type"] = orderType
	params["currency"] = currency
	params["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	params["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	params["instrument"] = instrument

	if clientRef != "" {
		params["clientOrderIdentifier"] = clientRef
	}

	err := i.SendAuthenticatedHTTPRequest("POST", path, params)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) GetWalletOrder(walletID, orderID string) {
	path := "/wallets/" + walletID + "/orders/" + orderID
	err := i.SendAuthenticatedHTTPRequest("GET", path, nil)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) CancelWalletOrder(walletID, orderID string) {
	path := "/wallets/" + walletID + "/orders/" + orderID
	err := i.SendAuthenticatedHTTPRequest("DELETE", path, nil)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) PlaceWithdrawalRequest(walletID, currency, address string, amount float64) {
	path := "/wallets/" + walletID + "/cryptocurrency_withdrawals"
	params := make(map[string]interface{})
	params["currency"] = currency
	params["amount"] = amount
	params["address"] = address

	err := i.SendAuthenticatedHTTPRequest("POST", path, params)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) GetDepositAddress(walletID, currency string) {
	path := "/wallets/" + walletID + "/cryptocurrency_deposits"
	params := make(map[string]interface{})
	params["currency"] = currency

	err := i.SendAuthenticatedHTTPRequest("POST", path, params)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) WalletTransfer(walletID, sourceWallet, destWallet string, amount float64, currency string) {
	path := "/wallets/" + walletID + "/wallet_transfers"
	params := make(map[string]interface{})
	params["sourceWalletId"] = sourceWallet
	params["destinationWalletId"] = destWallet
	params["amount"] = strconv.FormatFloat(amount, 'f', -1, 64)
	params["currencyCode"] = currency

	err := i.SendAuthenticatedHTTPRequest("POST", path, params)

	if err != nil {
		log.Println(err)
	}
}

func (i *ItBit) SendAuthenticatedHTTPRequest(method string, path string, params map[string]interface{}) (err error) {
	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)[0:13]
	nonce, err := strconv.Atoi(timestamp)

	if err != nil {
		return err
	}

	nonce -= 1
	request := make(map[string]interface{})
	url := ITBIT_API_URL + path

	if params != nil {
		for key, value := range params {
			request[key] = value
		}
	}

	PayloadJson := []byte("")

	if params != nil {
		PayloadJson, err = common.JSONEncode(request)

		if err != nil {
			return errors.New("SendAuthenticatedHTTPRequest: Unable to JSON Marshal request")
		}

		if i.Verbose {
			log.Printf("Request JSON: %s\n", PayloadJson)
		}
	}

	nonceStr := strconv.Itoa(nonce)
	message, err := common.JSONEncode([]string{method, url, string(PayloadJson), nonceStr, timestamp})
	if err != nil {
		log.Println(err)
		return
	}

	hash := common.GetSHA256([]byte(nonceStr + string(message)))
	hmac := common.GetHMAC(common.HASH_SHA512, []byte(url+string(hash)), []byte(i.APISecret))
	signature := common.Base64Encode(hmac)

	headers := make(map[string]string)
	headers["Authorization"] = i.ClientKey + ":" + signature
	headers["X-Auth-Timestamp"] = timestamp
	headers["X-Auth-Nonce"] = nonceStr
	headers["Content-Type"] = "application/json"

	resp, err := common.SendHTTPRequest(method, url, headers, bytes.NewBuffer([]byte(PayloadJson)))

	if i.Verbose {
		log.Printf("Recieved raw: \n%s\n", resp)
	}
	return nil
}
